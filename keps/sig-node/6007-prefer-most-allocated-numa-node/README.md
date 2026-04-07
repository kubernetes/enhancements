# KEP-6007: Add Topology Manager option to improve workload density for single-numa-node

[enhancement tracking issue]: https://github.com/kubernetes/enhancements/issues/6007

<!-- toc -->
- [Release Signoff Checklist](#release-signoff-checklist)
- [Summary](#summary)
- [Motivation](#motivation)
  - [Problem statement](#problem-statement)
  - [Illustrative NUMA packing](#illustrative-numa-packing)
  - [Goals](#goals)
  - [Non-Goals](#non-goals)
- [Proposal](#proposal)
  - [User Stories](#user-stories)
  - [Proposed API](#proposed-api)
  - [Algorithm](#algorithm)
  - [Notes / Constraints / Caveats](#notes--constraints--caveats)
  - [Risks and Mitigations](#risks-and-mitigations)
- [Design Details](#design-details)
  - [Topology manager ↔ resource manager interaction](#topology-manager--resource-manager-interaction)
  - [Test Plan](#test-plan)
      - [Prerequisite testing updates](#prerequisite-testing-updates)
      - [Unit tests](#unit-tests)
      - [Integration tests](#integration-tests)
      - [e2e tests](#e2e-tests)
  - [Rollout and Documentation](#rollout-and-documentation)
  - [Graduation Criteria](#graduation-criteria)
- [Production Readiness Review Questionnaire](#production-readiness-review-questionnaire)
  - [Feature Enablement and Rollback](#feature-enablement-and-rollback)
  - [Rollout, Upgrade and Rollback Planning](#rollout-upgrade-and-rollback-planning)
  - [Monitoring Requirements](#monitoring-requirements)
  - [Dependencies](#dependencies)
  - [Scalability](#scalability)
  - [Troubleshooting](#troubleshooting)
- [Drawbacks](#drawbacks)
- [Alternatives](#alternatives)
- [Implementation History](#implementation-history)
<!-- /toc -->

## Release Signoff Checklist

Items marked with (R) are required *prior to targeting to a milestone / release*.

- [ ] (R) Enhancement issue in release milestone, which links to KEP dir in [kubernetes/enhancements]
- [ ] (R) KEP approvers have approved the KEP status as `implementable`
- [x] (R) Design details are appropriately documented
- [x] (R) Test plan is in place
- [x] (R) Graduation criteria is in place
- [ ] (R) Production readiness review completed
- [ ] (R) Production readiness review approved

[kubernetes/enhancements]: https://git.k8s.io/enhancements

## Summary

Clusters that enforce strict NUMA locality with `topologyManagerPolicy: single-numa-node`
often give up **workload density**: pods fail to schedule even when node has **enough CPUs
in total**, because exclusive CPUs are fragmented across NUMAs.
Among **tied** equally valid single-NUMA merged outcomes, Topology Manager today breaks ties
with **`Narrowest`** (fewest NUMA nodes in the affinity mask; under `single-numa-node` all masks have width 1, so this degenerates to **lowest NUMA node ID**), **not**
**free or contiguous** exclusive-CPU headroom—so it cannot prefer the choice that leaves
**more room for the next** large single-NUMA pod.

This KEP proposes an **optional** Topology Manager policy option **`prefer-most-allocated-numa-node`**
so that, under **`single-numa-node`**, kubelet can break ties using **kubelet-local** signals
(static CPU and Memory managers)—in the spirit of **“most allocated”** packing (favor placing the pod
on the NUMA that **preserves a larger contiguous exclusive-CPU hole** on other NUMA for the **next**
workload). Default behavior stays **unchanged** when the option is off.

## Motivation

### Problem statement

- **Why `best-effort` provides “more capacity”:** The same Guaranteed workload with
  integer CPUs can be admitted under `best-effort` with **cross-NUMA** CPU placement.
  `single-numa-node` rejects unless the request fits on **one** NUMA; the difference is
  topology locality, not raw millis.
- **Observed failure mode:** Workloads pinned to one NUMA (devices, hugepages etc) make
  the node **asymmetric**. For **new** pods whose merged hints allow **either** NUMA,
  **always preferring the lower NUMA id** reduces the overall workload density on the node.

### Illustrative NUMA packing

The two figures below show why **where** work lands on each NUMA affects **single-NUMA**
workloads that need a **contiguous** block. Pod B is NUMA pinned due to device affinity.

![Illustration: capacity-not-aware NUMA selection (today’s default).](capacity-not-aware.png)

*Capacity-not-aware (today):* among valid NUMAs, the merger **prefers the lower NUMA id**
(via bitmask ordering) when there are **no** stronger hints from devices, hugepages, or
static memory—**without** considering remaining contiguous exclusive-CPU headroom.

![Illustration: prefer-most-allocated style selection leaves a larger contiguous free region on one NUMA.](capacity-aware.png)

*With `prefer-most-allocated-numa-node` (proposed):* NUMA scoring favors an outcome in the
**most-allocated / consolidate** spirit—**concentrating** use so the **other** NUMA keeps a
**larger contiguous** exclusive-CPU region—often better for the **next** Guaranteed pod under
static CPU.

### Goals

- **Improve workload density** (better use of nodes)
  for operators who use **`single-numa-node`** for strict locality (like in Telco).
- Integrate via **`TopologyManagerPolicyOptions`**, using the same **feature gate /
  graduation pattern** as for other topology policy options.

### Non-Goals

- Reusing the scheduler's **`NodeResourcesFit`** scoring. Kubelet admission already
  imports scheduler code—`predicateAdmitHandler` calls `scheduler.AdmissionCheck`,
  which runs `noderesources.Fits`, `nodeaffinity.Match`, etc.—but those checks
  operate at **node** granularity. This feature needs **per-NUMA** scoring using
  CPU and Memory manager state the scheduler does not have.

**Side benefit:** Because the default kube-scheduler is **not NUMA-aware**, it may place a
pod on a node whose aggregate resources look sufficient, only for kubelet to **reject** the
pod when no single NUMA node has enough contiguous free space. By acting as a **local
defragmenter**—consolidating smaller workloads on one NUMA to preserve larger contiguous
free regions on the other—this option reduces such "last-mile" admission failures and the
wasted scheduling cycles they cause.

## Proposal

### User Stories

1. **As a cluster operator** running `single-numa-node` with static CPU and Memory managers,
   I want **better density** under strict single-NUMA locality—without switching to
   **`best-effort`** (which results in cross-NUMA CPUs).

### Proposed API

Introduce a new Topology Manager policy option **`prefer-most-allocated-numa-node`**, configurable
via kubelet configuration alongside existing options:
- Only in effect when **`topologyManagerPolicy` is `single-numa-node`**.
  With any other policy (`best-effort`, `restricted`, `none`) the option is
  accepted but has no effect — only `single-numa-node` has a NUMA scorer.
- `TopologyManagerPolicyOptions` / alpha-beta gates as for other new options.

When the option is **disabled** (default), behavior remains **unchanged** from today.

```yaml
kind: KubeletConfiguration
apiVersion: kubelet.config.k8s.io/v1beta1
featureGates:
  ...
  TopologyManagerPolicyAlphaOptions: true
topologyManagerPolicyOptions:
  prefer-most-allocated-numa-node: "true"
topologyManagerPolicy: single-numa-node
memoryManagerPolicy: Static
cpuManagerPolicy: static
...
```

### Algorithm

**Trigger:** `topologyManagerPolicy` is **`single-numa-node`**, **`prefer-most-allocated-numa-node`**
is set in **`topologyManagerPolicyOptions`** (with required feature gates), and the hint merger is
comparing **preferred** merged hints whose NUMA affinity is a **single** NUMA node (bit count 1).

**Scoring (kubelet aggregator):**

Each signal computes a **utilization score** per NUMA using the same formula as
kube-scheduler's `MostAllocated` plugin: `score = (assigned × 100) / allocatable`,
where **allocatable** accounts for reserved resources so the ratio reflects true
utilization.

1. **CPU score (static CPU manager):** Score each NUMA by exclusive-CPU utilization
   (assigned / allocatable, where allocatable excludes reserved CPUs).
   Non-static policy → score 0 for all NUMAs.
2. **Memory score (static memory manager):** Score each NUMA by regular-memory utilization
   (assigned / allocatable, where allocatable excludes per-NUMA reserved memory).
   Non-static policy → score 0 for all NUMAs.
3. **Aggregate:** For each candidate NUMA, compute
   `aggregateScore = (cpuScore + memoryScore) / 2`, mirroring the scheduler's
   `MostAllocated` weighted-average approach. The NUMA with the higher aggregate score
   wins. Equal aggregate scores → **`Narrowest`** fallback (lowest NUMA node ID).

**Interactions:**

- **Scheduler / descheduler:** Not a substitute for this option. The **scheduler** chooses a
  **node**; it does not run Topology Manager or finalize **static** CPU / memory **NUMA** placement
  on the node. The **descheduler** evicts pods so they can be scheduled again. This KEP answers
  the question "which single-NUMA outcome wins when several are equivalent?".
- **Merge:** Still driven by hint providers; this step only affects **which** valid
  single-NUMA preferred outcome wins when multiple exist.


### Notes / Constraints / Caveats

This KEP scopes the NUMA scorer to **exclusive CPUs** (static CPU manager) and
**pinned regular memory** (static memory manager) only. Devices (including hugepages) and
DRA resources are not in the scope.
A non-static manager contributes zero scores for all NUMAs, so the feature
falls back to a single-signal score.

Kube-scheduler supports weighted scoring for `MostAllocated` and defaults to equal weight.
This KEP uses equal-weight aggregation between CPU and memory because CPU and memory usage
usually track together in production and rarely diverge.
If real-world feedback shows one resource dominates placement decisions, weighted scoring
can be added in Beta.

### Risks and Mitigations

| Risk | Mitigation |
|------|------------|
| Admission latency | Scoring runs only on the tie path and is O(NUMA nodes), typically 2–8 iterations of simple integer arithmetic |
| Behavior surprise when enabled | Opt-in; metrics `topology_manager_admission_*` already exist |
| Placement affects subsequent admissions | The feature cannot fail the current pod's admission (it only selects among already-valid NUMAs), but placement choices are designed to improve admission for subsequent pods |

## Design Details

### Topology manager ↔ resource manager interaction

Under `prefer-most-allocated-numa-node` option, the topology manager needs to break
NUMA selection ties using utilization data it does not own. The CPU and Memory
managers hold this data.

**Design pattern: Strategy (via callback registration).** The topology manager
defines a NUMA scoring interface. Any component with the knowledge to score
NUMA nodes registers a strategy. The topology manager only invokes it when a
tie needs breaking — it never reaches into manager internals.

**Interface sketch:**

```go
type NUMAScorer interface {
    Score(current, candidate bitmask.BitMask) NUMAScoringResult
}
```

During kubelet startup, the container manager calls
`TopologyManager.SetNUMAScorer(numaScorer)` to register the strategy, which is a
new NUMA scorer that sources utilization data from CPU and Memory manager.
During pod admission, the topology manager calls `numaScorer.Score()`
when a NUMA tie needs breaking.

**Why a new interface instead of extending `HintProvider`?** `HintProvider` answers
"which NUMAs are valid for this resource?" — each resource manager implements it
independently with no cross-resource awareness. Extending `HintProvider` is not appropriate
because what is needed is to score already-merged hint results using utilization data
instead of adding a new hint for Topology Manager to merge.


### Test Plan

[X] Owners of involved components may require updates to existing tests before merge.

##### Prerequisite testing updates

No prerequisite test changes are required. The modified packages already have solid
unit test coverage.

##### Unit tests

Coverage baselines for modified packages
([source](https://testgrid.k8s.io/sig-testing-canaries#ci-kubernetes-coverage-unit)):

- `k8s.io/kubernetes/pkg/kubelet/cm/topologymanager`: `2026-06-05`
- `k8s.io/kubernetes/pkg/kubelet/cm/cpumanager`: `2026-06-05`
- `k8s.io/kubernetes/pkg/kubelet/cm/memorymanager`: `2026-06-05`
- `k8s.io/kubernetes/pkg/kubelet/cm`: `2026-06-05`

Five test suites cover each layer of the feature:

1. **CPU signal scoring** (`pkg/kubelet/cm/cpumanager/cpu_scoring_test.go`) —
   Verifies the CPU manager's per-NUMA utilization score: symmetric allocatable → equal
   scores, asymmetric reserved CPUs → different scores, non-static policy → zero scores.

2. **Memory signal scoring** (`pkg/kubelet/cm/memorymanager/memory_scoring_test.go`) —
   Verifies the memory manager's per-NUMA utilization score: symmetric allocatable → equal
   scores, asymmetric per-NUMA reserved memory → different scores, non-static policy → zero
   scores.

3. **Aggregator combine rules** (`pkg/kubelet/cm/numa_scorer_test.go`) —
   Stub-driven tests for the aggregate-score logic: equal aggregates → fallback,
   CPU-dominated, memory-dominated, and mixed scores producing a clear winner.

4. **Topology Manager merge** (`pkg/kubelet/cm/topologymanager/policy_prefer_most_allocated_test.go`) —
   End-to-end merge through `singleNumaNodePolicy`: scorer overrides default, and
   absent scorer falls back to Narrowest.

5. **Policy option gating** (`pkg/kubelet/cm/topologymanager/policy_options_test.go`) —
   `prefer-most-allocated-numa-node` accepted only with `TopologyManagerPolicyAlphaOptions`
   enabled; rejected otherwise.

##### Integration tests

No new integration tests for kubelet are planned.

##### e2e tests

E2e tests are planned for Beta. They will use the existing multi-NUMA CI
infrastructure that runs the topology manager e2e suite today (2 and 4 NUMA
node configurations). Planned scenarios:

- Multi-NUMA **static** CPU + `single-numa-node`: ordered admission of Guaranteed pods
  (no devices) to validate **baseline** vs **option** NUMA choice when ties exist.
- Scenarios where **one** NUMA already holds an exclusive / device-bound pod and pods could
  admit to **either** NUMA—assert the option changes **which** NUMA wins vs low-index default.

### Rollout and Documentation

- Alpha: new option behind `TopologyManagerPolicyAlphaOptions`
- User-facing docs: kubelet configuration reference, relationship to `single-numa-node` and
  static managers
- Release notes per phase.

### Graduation Criteria

- **Alpha:**
  - Implementation behind `TopologyManagerPolicyAlphaOptions` feature gate.
  - Unit tests covering scoring, aggregation, merge override, and policy option gating.
  - Documented semantics and known limitations in the KEP.
- **Beta:**
  - Gate promotion to `TopologyManagerPolicyBetaOptions`.
  - E2e tests running in CI with existing multi-NUMA topology infrastructure (2 and 4 NUMA nodes).
  - New observability metrics added (`memory_manager_allocation_per_numa`,
    `topology_manager_utilization_per_numa`).
  - Evaluate weighted scoring and other changes based on Alpha feedback.
- **GA:**
  - E2e tests stable for at least 2 releases.
  - No open P0/P1 bugs related to the feature.
  - Option promotion per SIG Node policy for topology manager policy options.

## Production Readiness Review Questionnaire

### Feature Enablement and Rollback

###### How can this feature be enabled / disabled in a live cluster?

- [x] Feature gate (also fill in values in `kep.yaml`)
  - Feature gate name: `TopologyManagerPolicyAlphaOptions`
  - Components depending on the feature gate: `kubelet`
- [x] Change the kubelet configuration to set `topologyManagerPolicy: single-numa-node` and
  `topologyManagerPolicyOptions: {"prefer-most-allocated-numa-node": "true"}`
  - Will enabling / disabling the feature require downtime of the control plane? No
  - Will enabling / disabling the feature require downtime or reprovisioning of a node?
    No. A kubelet restart is sufficient to pick up the configuration change. New
    workloads will use the updated scoring logic immediately. To adjust existing
    workloads, drain the node so pods are rescheduled under the new behavior.

###### Does enabling the feature change any default behavior?

Yes. When enabled, the NUMA scoring changes from always picking the lowest NUMA id
to preferring the most-allocated NUMA node. This only takes effect when the user
explicitly enables `TopologyManagerPolicyAlphaOptions`, sets `topologyManagerPolicy`
to `single-numa-node`, and adds `prefer-most-allocated-numa-node: "true"` to
`topologyManagerPolicyOptions`.

###### Can the feature be disabled once it has been enabled (i.e. can we roll back the enablement)?

Yes. Set `prefer-most-allocated-numa-node: "false"` in `topologyManagerPolicyOptions`
(or disable `TopologyManagerPolicyAlphaOptions` if this is the only alpha option in use)
and restart kubelet. Existing workloads retain their current CPU/memory placement;
only new admissions revert to the default Narrowest selection.

###### What happens if we reenable the feature if it was previously rolled back?

No impact on running workloads. Newly admitted pods will again use the
most-allocated scoring logic.

###### Are there any tests for feature enablement/disablement?

Unit tests in `pkg/kubelet/cm/topologymanager/policy_options_test.go` validate that
the option is accepted only when `TopologyManagerPolicyAlphaOptions` is enabled and
rejected otherwise.

### Rollout, Upgrade and Rollback Planning

###### How can a rollout or rollback fail? Can it impact already running workloads?

No. Enabling or disabling is a kubelet configuration change validated at startup.
It cannot impact already running workloads — the scoring logic is only invoked
during new pod admission and does not touch existing placements.

###### What specific metrics should inform a rollback?

This feature only affects the scoring choice among already-valid NUMA outcomes and
cannot cause admission failures. An operator would rollback if no workload density
increase is observed.

###### Were upgrade and rollback tested? Was the upgrade->downgrade->upgrade path tested?

Manual testing in multi-NUMA environments confirms that toggling the option and
restarting kubelet preserves running workloads and changes only future admissions.

###### Is the rollout accompanied by any deprecations and/or removals of features, APIs, fields of API types, flags, etc.?

No.

### Monitoring Requirements

###### How can an operator determine if the feature is in use by workloads?

Inspect the kubelet configuration on the node to verify `TopologyManagerPolicyAlphaOptions`
is enabled and `prefer-most-allocated-numa-node: "true"` is set in
`topologyManagerPolicyOptions`.

###### How can someone using this feature know that it is working for their instance?

- [x] Other (treat as last resort)
  - Details: When the scorer overrides the default NUMA selection, kubelet
    emits an info-level log with the selected and default NUMA IDs plus CPU,
    memory, and aggregate utilization scores for both. Operators can grep
    kubelet logs for `"Prefer-most-allocated scorer overrode default NUMA selection"`
    to confirm the feature is active and understand why each decision was made.

###### What are the reasonable SLOs (Service Level Objectives) for the enhancement?

The scoring runs only during topology admission when a NUMA tie exists. No measurable
impact on pod startup latency is expected.

###### What are the SLIs (Service Level Indicators) an operator can use to determine the health of the service?

- [x] Metrics
  - Metric name: `topology_manager_admission_duration_ms`
    - Components exposing the metric: kubelet
    - Expected behavior: unaffected (the scoring adds negligible overhead).
  - Metric name: `topology_manager_admission_requests_total`
    - Components exposing the metric: kubelet
    - Expected behavior: unaffected (same pods go through admission).
  - Metric name: `topology_manager_admission_errors_total`
    - Components exposing the metric: kubelet
    - Expected behavior: expected to trend lower over time for workloads that
      frequently hit single-NUMA capacity limits, since better packing preserves
      larger contiguous holes for subsequent admissions.

###### Are there any missing metrics that would be useful to have to improve observability of this feature?

No new metrics are introduced in Alpha. The three existing topology manager metrics
(`admission_duration_ms`, `admission_requests_total`, `admission_errors_total`)
continue to apply.

There is an existing `cpu_manager_allocation_per_numa` gauge that reports the
number of CPUs allocated per NUMA node, but no equivalent exists for memory.
In Beta, the following new metrics can be considered:

- `memory_manager_allocation_per_numa`: a per-NUMA gauge reporting memory bytes
  allocated per NUMA node, mirroring the existing CPU gauge.
- `topology_manager_utilization_per_numa`: a per-NUMA gauge reporting the
  aggregated utilization score that the scorer uses to make decisions,
  giving operators direct visibility into NUMA packing density.

Real user feedback may dictate other needs, we will see.

### Dependencies

###### Does this feature depend on any specific services running in the cluster?

No. It operates entirely within the kubelet using existing CPU and Memory manager state.

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

No. Scoring is a lightweight computation invoked only when a NUMA tie exists.

###### Will enabling / using this feature result in non-negligible increase of resource usage (CPU, RAM, disk, IO, ...) in any components?

No.

###### Can enabling / using this feature result in resource exhaustion of some node resources (PIDs, sockets, inodes, etc.)?

No.

### Troubleshooting

###### How does this feature react if the API server and/or etcd is unavailable?

N/A — the feature operates entirely within kubelet and does not depend on API server
or etcd.

###### What are other known failure modes?

None. The feature is a scorer that only selects among already-valid NUMA outcomes.
It cannot cause admission failures or affect running workloads.

###### What steps should be taken if SLOs are not being met to determine the problem?

Inspect `topology_manager_admission_duration_ms` for latency regression and
`topology_manager_admission_errors_total` for unexpected admission failures.
If either trends abnormally, disable the option and restart kubelet to rule it out.

## Drawbacks

- More logic and coupling between Topology Manager and CPU/Memory manager state.

## Alternatives

1. **Status quo:** Keep bitmask / low-NUMA-id selection; accept lower density on **asymmetric**
   nodes when symmetric pods **always** stack on the lower-id NUMA first.

## Implementation History

- 2026-04-07: Draft created.
