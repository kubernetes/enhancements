# KEP-6007: NUMA Utilization-Aware Topology Manager Policy Options

<!-- toc -->
- [Release Signoff Checklist](#release-signoff-checklist)
- [Summary](#summary)
- [Motivation](#motivation)
  - [Goals](#goals)
  - [Non-Goals](#non-goals)
- [Proposal](#proposal)
  - [User Stories](#user-stories)
    - [Story 1: Balanced NUMA Utilization on Multi-Die Processors](#story-1-balanced-numa-utilization-on-multi-die-processors)
    - [Story 2: Workload Consolidation for Power Efficiency](#story-2-workload-consolidation-for-power-efficiency)
    - [Story 3: Database Spreading](#story-3-database-spreading)
    - [Story 4: GPU Workload Prioritization](#story-4-gpu-workload-prioritization)
  - [Notes/Constraints/Caveats](#notesconstraintscaveats)
  - [Risks and Mitigations](#risks-and-mitigations)
- [Design Details](#design-details)
  - [Topology Hint Score Plumbing](#topology-hint-score-plumbing)
  - [Allocation-Aware Policy Options](#allocation-aware-policy-options)
  - [Interaction with Topology Manager Scope](#interaction-with-topology-manager-scope)
    - [Preserving Current Placement Behavior](#preserving-current-placement-behavior)
  - [Score-Aware Preferred-First Merge Optimization (Beta)](#score-aware-preferred-first-merge-optimization-beta)
  - [Per-Resource Weights](#per-resource-weights)
  - [Kubelet Configuration](#kubelet-configuration)
    - [Example Configurations](#example-configurations)
  - [Feature Gate](#feature-gate)
  - [Test Plan](#test-plan)
      - [Prerequisite testing updates](#prerequisite-testing-updates)
      - [Unit tests](#unit-tests)
      - [Integration tests](#integration-tests)
      - [e2e tests](#e2e-tests)
  - [Graduation Criteria](#graduation-criteria)
    - [Alpha](#alpha)
    - [Alpha to Beta Graduation](#alpha-to-beta-graduation)
    - [Beta to GA Graduation](#beta-to-ga-graduation)
  - [Graduation Criteria of Options](#graduation-criteria-of-options)
  - [Upgrade / Downgrade Strategy](#upgrade--downgrade-strategy)
  - [Version Skew Strategy](#version-skew-strategy)
- [Production Readiness Review Questionnaire](#production-readiness-review-questionnaire)
  - [Feature Enablement and Rollback](#feature-enablement-and-rollback)
  - [Rollout, Upgrade and Rollback Planning](#rollout-upgrade-and-rollback-planning)
  - [Monitoring Requirements](#monitoring-requirements)
  - [Dependencies](#dependencies)
  - [Scalability](#scalability)
  - [Troubleshooting](#troubleshooting)
- [Implementation History](#implementation-history)
- [Drawbacks](#drawbacks)
- [Alternatives](#alternatives)
  - [Alternative 1: Scheduler-Based NUMA Balancing](#alternative-1-scheduler-based-numa-balancing)
  - [Alternative 2: Pod-Level Annotations](#alternative-2-pod-level-annotations)
  - [Alternative 3: Static NUMA Assignment](#alternative-3-static-numa-assignment)
- [Infrastructure Needed (Optional)](#infrastructure-needed-optional)
<!-- /toc -->

## Release Signoff Checklist

Items marked with (R) are required *prior to targeting to a milestone / release*.

- [ ] (R) Enhancement issue in release milestone, which links to KEP dir in [kubernetes/enhancements] (not the initial KEP PR)
- [ ] (R) KEP approvers have approved the KEP status as `implementable`
- [ ] (R) Design details are appropriately documented
- [ ] (R) Test plan is in place, giving consideration to SIG Architecture and SIG Testing input (including test refactors)
  - [ ] e2e Tests for all Beta API Operations (endpoints)
  - [ ] (R) Ensure GA e2e tests meet requirements for [Conformance Tests](https://github.com/kubernetes/community/blob/master/contributors/devel/sig-architecture/conformance-tests.md)
  - [ ] (R) Minimum Two Week Window for GA e2e tests to prove flake free
- [ ] (R) Graduation criteria is in place
  - [ ] (R) [all GA Endpoints](https://github.com/kubernetes/community/pull/1806) must be hit by [Conformance Tests](https://github.com/kubernetes/community/blob/master/contributors/devel/sig-architecture/conformance-tests.md) within one minor version of promotion to GA
- [ ] (R) Production readiness review completed
- [ ] (R) Production readiness review approved
- [ ] "Implementation History" section is up-to-date for milestone
- [ ] User-facing documentation has been created in [kubernetes/website], for publication to [kubernetes.io]
- [ ] Supporting documentation, e.g., additional design documents, links to mailing list discussions/SIG meetings, relevant PRs/issues, release notes

[kubernetes.io]: https://kubernetes.io/
[kubernetes/enhancements]: https://git.k8s.io/enhancements
[kubernetes/kubernetes]: https://git.k8s.io/kubernetes
[kubernetes/website]: https://git.k8s.io/website

## Summary

Extend the Topology Manager with a `numa-allocation-strategy` policy option
that controls NUMA node selection based on current allocation state. Valid
values are `none` (default - preserves existing behavior), `most-allocated`
(packing), and `least-allocated` (spreading).

These new policy options build on the Topology Manager policy options framework
introduced in [KEP-3545: Improved multi-numa alignment in Topology Manager](/keps/sig-node/3545-improved-multi-numa-alignment).
Like `prefer-closest-numa-nodes` from KEP-3545, the new options modify the
behavior of existing Topology Manager policies (`best-effort`, `restricted`,
`single-numa-node`) without introducing new policies.

## Motivation

Modern multi-die CPU architectures expose multiple NUMA domains per socket
(e.g., three compute dies per processor with Sub-NUMA Clustering enabled). For
high-performance workloads such as packet-core network functions, each pod
requires all of its allocated CPUs on a single NUMA node to maintain locality,
while pod allocations must be balanced across the available NUMA nodes.

The Topology Manager currently selects NUMA nodes based purely on structural
properties: affinity mask width (narrowest) or NUMA distance (closest, via
KEP-3545). It has no awareness of how heavily each NUMA node is already
allocated. The current allocation algorithm packs workloads onto the first
NUMA node with sufficient available resources (lowest-ID tiebreak), resulting
in unbalanced NUMA utilization (for example, four pods on NUMA 0 and two on
NUMA 1) even when equivalent capacity exists elsewhere. This has been
identified in [kubernetes/kubernetes#125453](https://github.com/kubernetes/kubernetes/issues/125453).

This creates two classes of problems:

- **NUMA imbalance on multi-die processors:** On CPUs with 3+ NUMA domains per
  socket, the lowest-ID tiebreak concentrates workloads on one compute die
  while others sit idle, causing resource contention, turbo frequency
  imbalance, and degraded performance for latency-sensitive applications.
- **Fragmentation:** Workloads spread across partially-used NUMA nodes, leaving
  no fully-free nodes available for large allocations that require full NUMA
  locality.

The closest existing option (`distribute-cpus-across-numa`) addresses a
different use case: it spreads a single pod's CPUs across multiple NUMA nodes,
which is effectively the opposite of the desired behavior.

Operators today have no mechanism within the Topology Manager to influence NUMA
node selection based on allocation state. This KEP addresses that gap.

### Goals

- Introduce a `numa-allocation-strategy` policy option for the Topology Manager
  that selects NUMA nodes based on current allocation state, with values
  `none` (default, no preference, preserves existing behavior),
  `most-allocated` (packing), and `least-allocated` (spreading).
- Support per-resource weighting via `numa-score-weights` so operators can
  prioritize CPU, memory, or specific device types when computing utilization
  scores. Weights are specified as a comma-separated string of `resource=weight`
  pairs (e.g., `"cpu=3,memory=1,nvidia.com/gpu=6"`), where each weight is an
  integer in the range [0, 100]. Resources the string does not name default to a
  weight of 1, so naming a resource raises its influence rather than silencing
  the others.
- Maintain existing topology guarantee semantics. The new options only
  influence NUMA node selection among equally valid candidates and do not
  change which hints are considered preferred.
- Ensure no user-visible behavior change when the feature is disabled.

### Non-Goals

- Runtime resource utilization monitoring (e.g., actual CPU usage percent).
  This feature uses allocation state only.
- Cross-node balancing or cluster-wide scheduling decisions. These remain the
  scheduler's responsibility.
- Device-specific utilization metrics (e.g., GPU memory usage). This feature
  uses simple device count for now.
- Automatic weight tuning or workload profiling.

## Proposal

Add a `Score` field to `TopologyHint` that captures per-NUMA-node allocation
utilization. Hint providers (CPU manager, memory manager, device manager) each
compute a score representing how heavily a NUMA node set is allocated. The
Topology Manager then uses these scores, combined with new policy options, to
prefer either the most-allocated or least-allocated NUMA nodes.

The design has three parts: score plumbing in `TopologyHint` and hint
providers, allocation-aware policy options, and per-resource weight support
via `numa-score-weights`. A score-aware merge optimization is planned for
beta to improve performance on systems with many NUMA nodes.

### User Stories

#### Story 1: Balanced NUMA Utilization on Multi-Die Processors

As a cluster operator running latency-sensitive network functions on processors
with multiple compute dies (e.g., 3 or more NUMA domains per socket using
sub-NUMA clustering), I want pod allocations to spread equally across all NUMA
nodes so that no single compute die is overloaded while others sit idle.

Today, each pod requires all of its CPUs on a single NUMA node for performance
locality, but the Topology Manager's lowest-ID tiebreak concentrates pods onto
NUMA node 0 until it is exhausted. On multi-die architectures this creates
severe imbalance: one compute die runs near capacity while adjacent dies remain
empty, leading to uneven turbo frequency behavior, thermal throttling, and
wasted compute resources. The existing `distribute-cpus-across-numa` option
addresses the opposite problem: it spreads a single pod's CPUs across multiple
NUMA nodes, which breaks the single-NUMA-node locality these workloads require.

With `numa-allocation-strategy: "least-allocated"`, the Topology Manager would
select the least-utilized NUMA node for each new pod, naturally balancing
allocation across all compute dies while preserving per-pod NUMA locality.

See also: kubernetes/kubernetes#125453.

#### Story 2: Workload Consolidation for Power Efficiency

As a cluster operator running mixed batch and latency-sensitive workloads, I
want batch jobs to pack onto already-utilized NUMA nodes so that empty nodes
remain available for large latency-sensitive allocations requiring full NUMA
locality. On multi-die processors, consolidating workloads onto fewer compute
dies also allows idle dies to enter deeper power-saving states, reducing overall
power consumption.

#### Story 3: Database Spreading

As a database administrator, I want database replicas to spread across
under-utilized NUMA nodes to avoid thermal hotspots and improve sustained
throughput across the machine.

#### Story 4: GPU Workload Prioritization

As an ML cluster operator, I want to weight GPU allocation higher than CPU when
selecting NUMA nodes, so that GPU-heavy workloads prefer nodes where GPUs are
already allocated, consolidating GPU usage even if CPU utilization is
asymmetric.

### Notes/Constraints/Caveats

- The policy options only affect NUMA node selection among candidates that are
  otherwise equivalent (same preferred status, same affinity width). They do
  not override the existing narrowest/closest tiebreakers. Score is only
  consulted when affinities are equal.
- Score is based on allocation state (assigned CPUs, reserved memory bytes,
  allocated device count), not runtime utilization. This is a deliberate
  choice: allocation state is immediately available in the kubelet without
  additional monitoring infrastructure.
- The `most-allocated` and `least-allocated` values for `numa-allocation-strategy`
  are mutually exclusive (the field accepts a single value).
- `numa-score-weights` is coupled to `numa-allocation-strategy`. Weights only
  have an effect when `numa-allocation-strategy` is set to `most-allocated` or
  `least-allocated`. When it is `none` (or unset), scores are ignored entirely,
  so weights do nothing.
- The effect of `numa-allocation-strategy` differs between
  `topologyManagerScope: "pod"` and `topologyManagerScope: "container"` (the
  default), because under container scope the strategy is applied once per
  container against allocation state that its predecessors have already
  updated. See
  [Interaction with Topology Manager Scope](#interaction-with-topology-manager-scope).
- `numa-score-weights` uses `,` and `=` as delimiters. Resource names
  containing `=` or `,` would break parsing. In practice, Kubernetes resource
  names follow DNS subdomain / slash / name conventions (e.g.,
  `nvidia.com/gpu`, `intel.com/sriov-nic`) and do not use these characters,
  so this is not expected to be an issue.
- These options compose with `prefer-closest-numa-nodes` from
  [KEP-3545](/keps/sig-node/3545-improved-multi-numa-alignment): the comparison
  order is preferred flag, then narrowest/closest structural tiebreak, then
  score-based preference. This preserves structural guarantees while using
  score to break ties.

### Risks and Mitigations

- **Score aggregation adds CPU overhead to hint comparison.** Aggregation is
  O(n) where n is the number of providers (typically 2-4). Expected overhead
  is less than 1%.

- **Users misconfigure weights, causing unexpected behavior.** The weight
  string is validated at kubelet startup. Malformed strings (e.g., missing `=`
  delimiter, non-numeric values) and weights outside the integer range [0, 100]
  are rejected with clear error messages. The kubelet will not start with an
  invalid configuration.

- **Bugs in the implementation lead to kubelet crash.** Comprehensive unit
  and e2e testing will be used to mitigate this risk. If a crash does occur,
  disable the policy option and restart the kubelet. Pods already placed are
  not affected.

## Design Details

### Topology Hint Score Plumbing

Add a `Score` field to `TopologyHint` that captures allocation utilization:

```go
type TopologyHint struct {
    NUMANodeAffinity bitmask.BitMask
    Preferred        bool
    Score            int64  // utilization score, 0=unscored, scored range [1,100]
}
```

Each hint provider computes a score per NUMA node set:

- **CPU manager**: `Score = max(1, assignedExclusiveCPUs * 100 / allocatableCPUs)`
- **Memory manager**: `Score = max(1, assignedBytes * 100 / allocatableBytes)`
- **Device manager**: `Score = max(1, allocatedDeviceCount * 100 / allocatableDeviceCount)`

The Topology Manager aggregates these into a single score during hint merge.
By default this is an equal-weight average, `aggregatedScore = sum(scores) /
count`, which is the unweighted case of the general formula described in
[Per-Resource Weights](#per-resource-weights).

Score is populated by all hint providers but does not influence hint selection
on its own. It is only used when `numa-allocation-strategy` is set to
`most-allocated` or `least-allocated`.

### Allocation-Aware Policy Options

Add the `numa-allocation-strategy` policy option (with `none`,
`most-allocated`, and `least-allocated` values) that makes Score a first-class
selection criterion. When set to `none` (the default), Score is ignored and
existing Narrowest/Closest behavior is preserved.

Updated comparison priority order:

1. Preferred flag (topology constraint satisfaction, unchanged)
2. Affinity mask comparison: Narrowest or Closest (structural tiebreak, unchanged)
3. Score-based preference (if policy option is set, new)

This ensures that structural guarantees from `prefer-closest-numa-nodes` are
preserved. Score only breaks ties among structurally equivalent candidates
(same width, same distance).

Changes to hint comparison:

```go
// In compareNumaAffinityMasks(current, candidate *TopologyHint) *TopologyHint:
// Step 1: Preferred always wins
if current.Preferred != candidate.Preferred {
    if candidate.Preferred { return candidate }
    return current
}

// Step 2: Structural comparison (Narrowest or Closest)
if best := compareStructural(current, candidate, opts); best != nil {
    return best  // Different width or distance, use structural
}

// Step 3: Score-based preference (only when structure is identical)
if opts.NUMAAllocationStrategy == "most-allocated" {
    if candidate.Score > current.Score { return candidate }  // higher = more packed
    if candidate.Score < current.Score { return current }
}
if opts.NUMAAllocationStrategy == "least-allocated" {
    if candidate.Score < current.Score { return candidate }  // lower = more empty
    if candidate.Score > current.Score { return current }
}

// Step 4: unscored (Score == 0) or equal scores: keep current, the same
// lowest-ID outcome the merge produces today
return current
```

When `numa-allocation-strategy` is `"none"` (or unset), Score is ignored and
existing behavior is unchanged.

### Interaction with Topology Manager Scope

The Topology Manager aligns resources either per container or per pod depending
on `topologyManagerScope`, as defined in
[KEP-693](/keps/sig-node/693-topology-manager). Because
`numa-allocation-strategy` consults allocation state rather than only
structural properties, the two scopes lead to different intra-pod placement:

- **`pod` scope:** hints are collected and merged once for the whole pod, so
  the strategy is applied a single time against one snapshot of allocation
  state. All containers in the pod share the resulting affinity, and the
  strategy has no intra-pod effect.

- **`container` scope (the default):** hints are collected, merged, and
  allocated per container in sequence. Each container's allocation is committed
  to the CPU, memory, and device managers before the next container's hints are
  generated, so the scores seen by a later container already reflect the
  containers admitted before it. With `least-allocated` this means containers
  of the same pod tend to be placed on *different* NUMA nodes; with
  `most-allocated` they tend to be co-located.

The combination that warrants attention is therefore `container` scope with
`least-allocated`, which is the only case where this feature makes containers
of a pod less likely to share a NUMA node than they are today.

Intra-pod spreading in that combination is a consequence of the strategy rather
than a violation of the topology guarantee. `container` scope promises
per-container alignment only and has never promised a common NUMA node across
containers. The co-location seen today is incidental rather than contractual:
it falls out of the lowest-ID tiebreak and already breaks once the lowest-ID
node is exhausted, at which point the next container spills onto another NUMA
node. `least-allocated` does not remove a guarantee; it surfaces an existing
non-guarantee earlier and more often.

Score quantization further limits how often this arises. Scores are integers in
[1,100] computed as `assigned * 100 / allocatable`, so on a node with 128 CPUs
per NUMA node a single-CPU container moves the score by 0.78, which truncates
to 0. When the score does not change, the comparison falls through to the
existing structural tiebreak and containers are placed as they are today.
Spreading under `container` scope therefore only takes effect once a container
is large enough relative to the NUMA node to move the score.

#### Preserving Current Placement Behavior

Operators who require containers of a pod to share a NUMA node have three
options, none of which require changes to this design:

- **Use `most-allocated` instead.** Under `container` scope it strengthens
  co-location rather than weakening it: each container raises the score of the
  node it lands on, which attracts the next container to the same node. This is
  a stronger form of co-location than today's tiebreak provides.
- **Set `topologyManagerScope: "pod"`.** This guarantees a common affinity
  across all containers, which is stricter than anything `container` scope
  offers today. It is not a drop-in substitution: pod scope admits a pod only
  if all of its containers can achieve a *common* alignment, so some pods that
  are admitted under container scope today will be rejected. That tradeoff is
  pre-existing KEP-693 behavior and is not introduced by this KEP. All example
  configurations in this KEP use pod scope.
- **Leave `numa-allocation-strategy` as `none`.** Placement is then unchanged.

Recovering pod-level co-location while remaining on `container` scope with
`least-allocated` is explicitly not a goal. The policy interface is
per-container by construction, and a rule that steered a container toward the
NUMA node its siblings already occupy would reimplement pod scope inside
container scope while contradicting what container scope means.

E2e tests will cover both scopes so that the difference is verified rather than
incidental.

### Score-Aware Preferred-First Merge Optimization (Beta)

This optimization is deferred to beta. In alpha, the existing `Merge()` logic
is used with Score evaluated during comparison.

With Score driving selection, the merge can be made more efficient. Instead of
exploring all O(N^K) hint permutations, a `mergePreferred()` method would
explore only preferred hints sorted by Score so the best candidate is found
first. On systems with many NUMA nodes (8+) and multiple providers, this
reduces the search space significantly (e.g., from 49 permutations to 3 on a
3-NUMA, 2-provider system).

### Per-Resource Weights

The `numa-score-weights` policy option is specified as a comma-separated
string of `resource=weight` pairs in `TopologyManagerPolicyOptions`:

```yaml
topologyManagerPolicyOptions:
  numa-score-weights: "cpu=3,memory=1,nvidia.com/gpu=6"
```

This keeps `TopologyManagerPolicyOptions` as `map[string]string`, consistent
with all existing policy options. The string is parsed at kubelet startup into
a `map[string]int` for internal use.

Weights are integers in the range [0, 100], the same form kube-scheduler's
`NodeResourcesFit` plugin uses for its per-resource weights.

The Topology Manager preserves resource names through the merge pipeline to
enable weighted score aggregation. The `providersHints` structure is
`map[string][]TopologyHint` where keys are resource names ("cpu", "memory",
"nvidia.com/gpu", etc.). During merge, resource names are threaded alongside
hints so the aggregation function can look up `weights[resourceName]` for each
provider's score. This avoids duplicating resource names in every TopologyHint
struct.

Scores are aggregated as a weighted average over the providers that reported a
score for the candidate NUMA node set:

```
aggregatedScore = sum(weight[r] * score[r]) / sum(weight[r])
```

Because the denominator is the sum of the applicable weights, the aggregation
is self-normalizing and the result stays in the same [1,100] range as the
individual provider scores.

A provider the weight string does not name is given a weight of 1. Because 1 is
also the smallest weight an operator can explicitly assign, an unnamed provider
can never outweigh a named one: naming a resource raises its influence relative
to everything else and never lowers it. When `numa-score-weights` is unset
entirely every provider sits at 1, and the formula reduces to the equal-weight
average `sum(scores) / count` described in
[Topology Hint Score Plumbing](#topology-hint-score-plumbing).

Since the baseline is 1, the only way to drop a resource from
scoring altogether is to give it an explicit weight of 0.

The table below gives the effective weight of each provider for a pod requesting
CPU, memory, GPU, and an SR-IOV NIC, where `intel.com/sriov-nic` is never named
in the weight string:

| Configuration | CPU | Memory | GPU | SR-IOV | Effect |
|---------------|-----|--------|-----|--------|--------|
| (unset) | 1 | 1 | 1 | 1 | Equal weighting, the same aggregation used when the option is absent |
| `"nvidia.com/gpu=10"` | 1 | 1 | 10 | 1 | GPU counts ten times as much as each other provider |
| `"cpu=3,memory=1,nvidia.com/gpu=6"` | 3 | 1 | 6 | 1 | GPU leads, CPU is intermediate, memory and the NIC sit at the baseline |
| `"cpu=5,nvidia.com/gpu=5"` | 5 | 1 | 5 | 1 | CPU and GPU rank equally, memory and the NIC still contribute |
| `"nvidia.com/gpu=10,cpu=0"` | 0 | 1 | 10 | 1 | GPU dominates and CPU is excluded outright |
| `"cpu=100,memory=100,nvidia.com/gpu=100"` | 100 | 100 | 100 | 1 | The three named providers swamp the NIC without silencing it |

Only the ratios between weights matter, so
`"cpu=30,memory=10,nvidia.com/gpu=60"` and `"cpu=3,memory=1,nvidia.com/gpu=6"`
behave identically; the smaller numbers are easier to read. A device type that
appears on the node later contributes at weight 1 as soon as a pod requests it,
which is the behavior we want for NUMA placement: a resource the operator has
not ranked still affects locality, just less than the ones they have.

If the total applicable weight comes out as 0 the weighted average is undefined.
In practice this means no provider reported a score, which is the case for a pod
that requests none of the resources the score-aware providers cover. The
Topology Manager then compares hints using the existing Narrowest/Closest logic
alone, the behavior it already has when `numa-allocation-strategy` is `"none"`.

Provider score weights are validated when the kubelet configuration is parsed:

- All weights must be integers in the range [0, 100]. Negative, fractional, and
  out-of-range values are rejected with an error. A weight of 0 is accepted and
  excludes the resource from scoring.
- Resource names are not validated against the providers present on the node.
  The applicable provider set is a property of the pod being admitted rather
  than of the node, so a name that matches nothing on the node is not
  distinguishable from a name that simply is not requested by a given pod. A
  weight string naming a resource no workload requests is accepted; that
  resource never contributes a score.

### Kubelet Configuration

Both new policy options are string values in the existing
`TopologyManagerPolicyOptions` (`map[string]string`). No type change is
needed.

| Key | Value | Description |
|-----|-------|-------------|
| `numa-allocation-strategy` | `"none"`, `"most-allocated"`, `"least-allocated"` | Controls NUMA node selection based on allocation state |
| `numa-score-weights` | `"resource=weight,..."` (e.g., `"cpu=3,nvidia.com/gpu=6"`) | Comma-separated resource=weight pairs (integers in [0, 100]) for weighted score aggregation |

#### Example Configurations

**No preference (default, preserves existing behavior):**
```yaml
apiVersion: kubelet.config.k8s.io/v1beta1
kind: KubeletConfiguration
topologyManagerPolicy: "single-numa-node"
topologyManagerScope: "pod"
topologyManagerPolicyOptions:
  numa-allocation-strategy: "none"
```

**Packing (consolidate workloads):**
```yaml
apiVersion: kubelet.config.k8s.io/v1beta1
kind: KubeletConfiguration
topologyManagerPolicy: "single-numa-node"
topologyManagerScope: "pod"
topologyManagerPolicyOptions:
  numa-allocation-strategy: "most-allocated"
```

**Spreading (balance load):**
```yaml
apiVersion: kubelet.config.k8s.io/v1beta1
kind: KubeletConfiguration
topologyManagerPolicy: "restricted"
topologyManagerScope: "pod"
topologyManagerPolicyOptions:
  numa-allocation-strategy: "least-allocated"
```

**GPU-weighted packing:**
```yaml
apiVersion: kubelet.config.k8s.io/v1beta1
kind: KubeletConfiguration
topologyManagerPolicy: "single-numa-node"
topologyManagerScope: "pod"
topologyManagerPolicyOptions:
  numa-allocation-strategy: "most-allocated"
  numa-score-weights: "nvidia.com/gpu=6,cpu=3,memory=1"
```

### Feature Gate

**Name:** `TopologyManagerPolicyAlphaOptions` (existing gate from
[KEP-3545](/keps/sig-node/3545-improved-multi-numa-alignment), reused for new
alpha-stage options)

**Behavior when disabled:**
- `numa-allocation-strategy` is ignored if specified.
- `numa-score-weights` is ignored if specified.
- Score field exists in `TopologyHint` but is not used in hint comparison.
- Existing Narrowest/Closest behavior is preserved.

### Test Plan

[x] I/we understand the owners of the involved components may require updates
to existing tests to make this code solid enough prior to committing the
changes necessary to implement this enhancement.

##### Prerequisite testing updates

None. Existing Topology Manager test infrastructure is sufficient.

##### Unit tests

- `k8s.io/kubernetes/pkg/kubelet/cm/topologymanager`: `<date>` - `<coverage>`

Unit tests will cover:
- Score calculation for each provider (CPU/Memory/Device)
- Score aggregation with various weight combinations (equal-weight, CPU-weighted,
  GPU-weighted)
- Weight validation: range [0, 100], rejection of negative, fractional, and
  out-of-range values
- Weight normalization: verify `"cpu=30,memory=10,nvidia.com/gpu=60"` yields the
  same result as `"cpu=3,memory=1,nvidia.com/gpu=6"`
- Default weight resolution: verify all providers are weighted equally when
  `numa-score-weights` is unset, and that unlisted providers default to weight 1
  when it is set
- Explicit exclusion: verify weight=0 excludes a resource from scoring
- Policy comparison with most/least-allocated preferences
- Edge cases: no scores, equal scores, and a container that requests none of the
  score-reporting providers (no applicable weight, expect fallback to
  Narrowest/Closest logic)
- Mixed named and unnamed providers: a container requesting a resource the
  weight string does not name is still scored over that provider at weight 1,
  and the self-normalizing denominator keeps the aggregate in the [1,100] range
- Validation of `numa-allocation-strategy` values (`none`, `most-allocated`,
  `least-allocated`)
- Feature gate on/off behavior

##### Integration tests

Integration tests will cover:
- Multi-provider hint merge with scores
- Policy behavior with feature gate enabled and disabled
- Weight normalization end-to-end
- Interaction with `prefer-closest-numa-nodes` from KEP-3545

##### e2e tests

E2e tests will cover:
- Pod admission with packing policy option (verify NUMA node selection prefers
  most-allocated nodes)
- Pod admission with spreading policy option (verify NUMA node selection
  prefers least-allocated nodes)
- Default behavior preserved when `numa-allocation-strategy` is `"none"` or
  unset
- Both `topologyManagerScope: "pod"` and `topologyManagerScope: "container"`,
  verifying that a multi-container pod shares a NUMA node under pod scope and
  that container scope applies the strategy per container as described in
  [Interaction with Topology Manager Scope](#interaction-with-topology-manager-scope)

### Graduation Criteria

#### Alpha

- [ ] Feature implemented behind `TopologyManagerPolicyAlphaOptions` feature
  gate
- [ ] Score plumbing in TopologyHint and hint providers
- [ ] `numa-allocation-strategy` policy option implemented (with `none`,
  `most-allocated`, and `least-allocated` values)
- [ ] Per-resource weights (`numa-score-weights`) implemented
- [ ] Add proper e2e node tests

#### Alpha to Beta Graduation

- [ ] Gather feedback from consumers of the new policy options
- [ ] No major bugs reported in the previous cycle
- [ ] Score-aware preferred-first merge optimization implemented

#### Beta to GA Graduation

- [ ] Allowing time for feedback (at least 2 releases as beta)
- [ ] Risks have been addressed

### Graduation Criteria of Options

This KEP follows the same option graduation model introduced in
[KEP-3545](/keps/sig-node/3545-improved-multi-numa-alignment) and also used by
[CPUManagerPolicyOptions](https://github.com/kubernetes/enhancements/tree/master/keps/sig-node/2625-cpumanager-policies-thread-placement#graduation-criteria-of-options):

- **Alpha:** Options are hidden by default and require both
  `TopologyManagerPolicyOptions` and `TopologyManagerPolicyAlphaOptions`
  feature gates to be enabled.
- **Beta:** Options move to `TopologyManagerPolicyBetaOptions` and are
  available when `TopologyManagerPolicyOptions` is enabled (which is on by
  default since KEP-3545 graduated).
- **GA:** Options become stable and are always available when
  `TopologyManagerPolicyOptions` is enabled.

### Upgrade / Downgrade Strategy

**Upgrade:**
- Feature gate default-off; users opt-in via kubelet config.
- Score field added to `TopologyHint` with zero-value default. No behavior
  change for existing workloads.

**Downgrade:**
- Rolling back kubelet removes policy options from config.
- Pods already placed remain where they are.
- New pods revert to Narrowest/Closest selection.

No changes to existing cluster configurations are required on upgrade.

### Version Skew Strategy

Not applicable. This is a node-local kubelet feature with no cross-component
dependencies. The feature does not affect the API server, scheduler, or any
other control plane component.

## Production Readiness Review Questionnaire

<!--
The following PRR answers are required at beta release.
-->

### Feature Enablement and Rollback

###### How can this feature be enabled / disabled in a live cluster?

- [x] Feature gate (also fill in values in `kep.yaml`)
  - Feature gate names:
    - `TopologyManagerPolicyOptions`
    - `TopologyManagerPolicyAlphaOptions`
  - Components depending on the feature gate: kubelet
- [x] Change the kubelet configuration to set a `TopologyManager` policy
  (`best-effort`, `restricted`, or `single-numa-node`) and add
  `numa-allocation-strategy` (with value `most-allocated` or
  `least-allocated`) in `TopologyManagerPolicyOptions`.
  - Will enabling / disabling the feature require downtime of the control
    plane? No.
  - Will enabling / disabling the feature require downtime or reprovisioning
    of a node? Yes, a kubelet restart is required.

###### Does enabling the feature change any default behavior?

No. The policy options are opt-in. When `numa-allocation-strategy` is unset or
`"none"`, behavior is identical to the current Topology Manager behavior.

###### Can the feature be disabled once it has been enabled (i.e. can we roll back the enablement)?

Yes. Either:
- Disable the `TopologyManagerPolicyAlphaOptions` feature gate, or
- Set `numa-allocation-strategy` to `"none"`, or
- Remove the policy option from kubelet configuration.

In all cases, a kubelet restart is required. Existing pod placements are not
affected; only new pod admissions will revert to the default Narrowest/Closest
selection.

###### What happens if we reenable the feature if it was previously rolled back?

No impact on existing containers. The allocation-aware policy option will apply
to new container admissions only.

###### Are there any tests for feature enablement/disablement?

There will be specific unit and e2e tests demonstrating that:
- Default behavior is preserved when the feature gate is disabled.
- Policy options are rejected when the feature gate is disabled.
- Score field is ignored in hint comparison when the feature gate is disabled.

### Rollout, Upgrade and Rollback Planning

###### How can a rollout or rollback fail? Can it impact already running workloads?

Kubelet may fail to start if invalid policy option values are provided (e.g.,
an unrecognized value for `numa-allocation-strategy`).
Already running workloads are not affected. The feature only influences new
pod admissions.

###### What specific metrics should inform a rollback?

An increase in topology admission errors or unexpected pod placement patterns
(e.g., pods not landing on expected NUMA nodes) should prompt investigation.

###### Were upgrade and rollback tested? Was the upgrade->downgrade->upgrade path tested?

Will be tested as part of beta graduation criteria.

###### Is the rollout accompanied by any deprecations and/or removals of features, APIs, fields of API types, flags, etc.?

No.

### Monitoring Requirements

###### How can an operator determine if the feature is in use by workloads?

Inspect the kubelet configuration of the nodes: check the feature gate status
and the `numa-allocation-strategy` value in `topologyManagerPolicyOptions`.

###### How can someone using this feature know that it is working for their instance?

- [ ] Other (treat as last resort)
  - Details: Launch a pod requiring resources from a specific NUMA node on a
    node with known asymmetric allocation. Verify via `taskset -cp 1` and
    `numactl -H` inside the container that resources are assigned from the
    expected NUMA node (most-allocated or least-allocated depending on the
    configured option).

###### What are the reasonable SLOs (Service Level Objectives) for the enhancement?

N/A. This is a node-local admission-time optimization. It does not introduce
new API latency or availability concerns.

###### What are the SLIs (Service Level Indicators) an operator can use to determine the health of the service?

- [x] Metrics
  - Metric name: `topology_manager_admission_duration_ms`
  - Components exposing the metric: kubelet

This existing metric captures the time spent in Topology Manager admission.
Any regression introduced by score computation would be visible as an increase
in this metric.

###### Are there any missing metrics that would be useful to have to improve observability of this feature?

A metric tracking the score-based selection outcome (e.g., which NUMA node was
selected and its score) would be useful for debugging and can be considered
for beta.

### Dependencies

###### Does this feature depend on any specific services running in the cluster?

No. The feature is entirely within the kubelet and uses allocation state
already tracked by CPU manager, memory manager, and device manager.

### Scalability

###### Will enabling / using this feature result in any new API calls?

No.

###### Will enabling / using this feature result in introducing new API types?

No.

###### Will enabling / using this feature result in any new calls to the cloud provider?

No.

###### Will enabling / using this feature result in increasing size or count of the existing API objects?

No.

###### Will enabling / using this feature result in increasing time taken by any operations covered by existing SLIs/SLOs?

No. The merge optimization (`mergePreferred()`) is expected to reduce hint
merge time for common cases. Score computation adds O(n) overhead where n is
the number of providers (typically 2-4), which is negligible.

###### Will enabling / using this feature result in non-negligible increase of resource usage (CPU, RAM, disk, IO, ...) in any components?

No. The additional Score field in `TopologyHint` adds 8 bytes per hint. Score
computation reuses allocation state already maintained by hint providers.

###### Can enabling / using this feature result in resource exhaustion of some node resources (PIDs, sockets, inodes, etc.)?

No.

### Troubleshooting

###### How does this feature react if the API server and/or etcd is unavailable?

N/A. This is a kubelet-local feature that does not depend on the API server
or etcd for its operation. Kubelet will continue to make admission decisions
using local allocation state.

###### What are other known failure modes?

- Invalid `numa-allocation-strategy` value (e.g., unrecognized string):
  - Detection: Kubelet fails to start with a clear error message listing
    valid values (`none`, `most-allocated`, `least-allocated`).
  - Diagnostics: Kubelet startup log will contain the validation error.
  - Testing: Unit tests cover value validation.

- Invalid `numa-score-weights` values (out of range [0, 100], fractional, NaN):
  - Detection: Kubelet fails to start with a clear error message.
  - Mitigations: Fix the weight values in kubelet configuration. Weights must
    be integers in the range [0, 100], matching kube-scheduler's
    `NodeResourcesFit` plugin.
  - Diagnostics: Kubelet startup log will contain the validation error.
  - Testing: Unit tests cover weight validation.

- Containers of a pod land on different NUMA nodes under `least-allocated`:
  - Detection: Expected behavior under `topologyManagerScope: "container"`
    rather than a defect. See
    [Interaction with Topology Manager Scope](#interaction-with-topology-manager-scope).
  - Mitigations: See
    [Preserving Current Placement Behavior](#preserving-current-placement-behavior).
  - Diagnostics: Compare `topologyManagerScope` in the kubelet configuration
    against the pod's observed CPU and memory affinity.
  - Testing: E2e tests cover both scopes.

###### What steps should be taken if SLOs are not being met to determine the problem?

N/A.

## Implementation History

- 2026-09-08: KEP created

## Drawbacks

Adds complexity to the Topology Manager hint comparison and merge logic.
However, the additional complexity is gated behind opt-in policy options and
has no effect on the default code path.

## Alternatives

### Alternative 1: Scheduler-Based NUMA Balancing

Move NUMA allocation decisions to the scheduler instead of the kubelet.

**Pros:** Cluster-wide visibility, better global optimization.
**Cons:** Requires scheduler-kubelet protocol changes; breaks existing Topology
Manager design; significantly larger scope.

### Alternative 2: Pod-Level Annotations

Let users specify packing/spreading preference per pod via annotations.

**Pros:** More flexible, users opt-in per workload.
**Cons:** Increases configuration burden; hard to enforce cluster-wide
policies; inconsistent with the existing node-level Topology Manager model.

### Alternative 3: Static NUMA Assignment

Pre-assign NUMA nodes to pod QoS classes.

**Pros:** Simple, predictable.
**Cons:** Inflexible, wastes resources when QoS classes don't match NUMA
topology.

## Infrastructure Needed (Optional)

- E2e test infrastructure with multi-NUMA nodes (at least 2 NUMA nodes per
  test worker).
- Performance test cluster for benchmarking overhead.
