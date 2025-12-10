# KEP-5732: Topology-aware workload scheduling

<!-- toc -->

-   [Release Signoff Checklist](#release-signoff-checklist)
-   [Summary](#summary)
-   [Motivation](#motivation)
    -   [Goals](#goals)
    -   [Non-Goals](#non-goals)
-   [Proposal](#proposal)
    -   [User Stories (Optional)](#user-stories-optional)
    -   [Story 1: AI Training in a Single Rack](#story-1-ai-training-in-a-single-rack)
    -   [Story 2: Workload using Interconnected DRA Devices](#story-2-workload-using-interconnected-dra-devices)
    -   [Notes/Constraints/Caveats (Optional)](#notesconstraintscaveats-optional)
    -   [Risks and Mitigations](#risks-and-mitigations)
-   [Design Details](#design-details)
    -   [Workload API Changes](#workload-api-changes)
    -   [Scheduling Framework Extensions](#scheduling-framework-extensions)
    -   [1. Data Structures](#1-data-structures)
    -   [2. New Plugin Interfaces](#2-new-plugin-interfaces)
    -   [Scheduling Algorithm Phases](#scheduling-algorithm-phases)
    -   [Phase 1: Candidate Placement Generation](#phase-1-candidate-placement-generation)
    -   [Phase 2: Pod-Level Filtering and Feasibility Check](#phase-2-pod-level-filtering-and-feasibility-check)
    -   [Phase 3: Placement Scoring and Selection](#phase-3-placement-scoring-and-selection)
    -   [Scheduler Plugins](#scheduler-plugins)
    -   [Potential Future Extensions (Beta Candidates)](#potential-future-extensions-beta-candidates)
    -   [Test Plan](#test-plan)
    -   [Prerequisite testing updates](#prerequisite-testing-updates)
    -   [Unit tests](#unit-tests)
    -   [Integration tests](#integration-tests)
    -   [e2e tests](#e2e-tests)
    -   [Graduation Criteria](#graduation-criteria)
    -   [Alpha](#alpha)
    -   [Beta](#beta)
    -   [Upgrade / Downgrade Strategy](#upgrade--downgrade-strategy)
    -   [Version Skew Strategy](#version-skew-strategy)
-   [Production Readiness Review Questionnaire](#production-readiness-review-questionnaire)
    -   [Feature Enablement and Rollback](#feature-enablement-and-rollback)
    -   [Rollout, Upgrade and Rollback Planning](#rollout-upgrade-and-rollback-planning)
    -   [Monitoring Requirements](#monitoring-requirements)
    -   [Dependencies](#dependencies)
    -   [Scalability](#scalability)
    -   [Troubleshooting](#troubleshooting)
-   [Implementation History](#implementation-history)
-   [Drawbacks](#drawbacks)
-   [Alternatives](#alternatives)
    -   [Pod Inter-Affinities](#pod-inter-affinities)
    -   [Standalone Schedulers (e.g., Volcano)](#standalone-schedulers-eg-volcano)
-   [Infrastructure Needed (Optional)](#infrastructure-needed-optional)
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
- [ ] Supporting documentation—e.g., additional design documents, links to mailing list discussions/SIG meetings, relevant PRs/issues, release notes

<!--
**Note:** This checklist is iterative and should be reviewed and updated every time this enhancement is being considered for a milestone.
-->

[kubernetes.io]: https://kubernetes.io/
[kubernetes/enhancements]: https://git.k8s.io/enhancements
[kubernetes/kubernetes]: https://git.k8s.io/kubernetes
[kubernetes/website]: https://git.k8s.io/website

## Summary

This KEP describes the architectural design and implementation details for
integrating a Topology-Aware and DRA-Aware workload scheduling algorithm into
the Kubernetes kube-scheduler to address the complex placement requirements of
modern, high-performance distributed applications.

The proposed algorithm fundamentally alters the scheduling lifecycle for gang
scheduled workloads. Instead of evaluating pods individually against the cluster
state - a process prone to fragmentation and deadlocks - the new mechanism
generates "Placements". These Placements represent candidate domains (sets of
nodes or DRA resources) where the entire workload is theoretically feasible. The
scheduler then simulates the placement of the full group of pods within these
domains, utilizing existing filtering and scoring logic to ensure high-fidelity
decisions before committing resources.

This design introduces specific extensions to the Kubernetes Workload API to
support `TopologyConstraints` and `DRAConstraints`, defines new interfaces
within the Scheduling Framework (`PlacementGenerator`, `PlacementState`,
`PlacementScorer`), and details the algorithmic flow required to schedule Pod
Groups while maintaining compatibility with the scheduler's existing ecosystem.

## Motivation

Distributed workloads, particularly those driving the current AI/ML era, often
require high-bandwidth and low-latency communication between multiple pods to
function efficiently. While the [KEP-4671: Workload API] makes the first step
towards managing these applications as cohesive units, it primarily establishes
the API structure. For workloads sensitive to inter-pod communication, simply
grouping pods is insufficient; their physical placement within the cluster's
network topology is a decisive factor in their performance.

In this KEP, we propose an algorithm for topology-aware and DRA-aware scheduling
that operates directly within the Kubernetes kube-scheduler. The core objective
is to ensure that pods belonging to a Workload are co-located within optimal
topological domains - such as specific racks or blocks - or are bound to shared
Dynamic Resource Allocation (DRA) devices that require cohesive management.
Without this level of precision, workloads may be fragmented across disparate
network domains, drastically degrading performance and wasting the potential of
expensive hardware.

Given the economics of high-performance accelerators and network infrastructure,
maximizing application performance and resource utilization is a primary goal
for users. Achieving this requires intelligent placement decisions that
understand the physical constraints of the cluster. However, the default
scheduler's current pod-centric logic lacks the native mechanisms to efficiently
resolve these complex group-level constraints during the scheduling cycle.

Topology-aware scheduling is not a new concept and is currently addressed by
external admission control systems like Kueue or alternative schedulers like
Volcano. However, relying on external admission controllers decouples the
topology decision from the scheduler's core logic, while alternative schedulers
introduce operational complexity. We believe that embedding topology and DRA
awareness deeply into the kube-scheduler is critical enough to warrant
standardization. This integration allows the algorithm to leverage the full
fidelity of the scheduler's existing pod-level filtering and scoring plugins,
ensuring highly accurate feasibility checks and placement outcomes without the
need for external dependencies.

### Goals

- To enhance kube-scheduler to perform topology-aware and DRA-aware scheduling
  for multi-pod workloads, as defined by the Workload API (KEP-4671).
- To optimize the placement of distributed workloads by co-locating pods based
  on network topology and DRA resource availability.
- To introduce new extension points and phases within the Kubernetes scheduler
  framework to support the concept of "Placements" (candidate sets of nodes
  and DRA resources).
- To define the required changes to the Workload API (KEP-4671) to support
  scheduling constraints.
- To leverage the scheduler's existing pod-level filtering and scoring logic
  within the evaluation of each Placement.
- To provide a flexible framework extensible by plugins for various topology
  sources (e.g., node labels) and resource types (e.g., DRA).

### Non-Goals

- To replace the functionality of external workload queueing and admission
  control systems like Kueue. This proposal focuses on the in-scheduler
  placement decision for a single Workload at a time.
- To implement Workload-level queueing, fairness, or resource quotas within
  kube-scheduler.
- To handle all aspects of the workload lifecycle management beyond
  scheduling.
- To implement Workload-level preemption logic.
- To integrate with cluster autoscaling mechanisms in this phase.
- To support complex multi-PodSet dependency resolution with backtracking or
  parallel processing in the initial version.
- To automatically discover network topology; the mechanisms rely on topology
  information being present (e.g., via node labels or DRA ResourceSlices).

## Proposal

This proposal introduces an API to define constraints on a PodGroup (a
collection of pods within a Workload) requiring it to be scheduled onto a
specific subset of nodes or resources.

We support two fundamental types of constraints:

1. **Topology Constraint (Node Label Co-location)**: Ensures all pods in a
   PodGroup are placed onto nodes sharing a common topological characteristic
   (e.g., same rack), defined by a specific node label.

2. **DRA Constraint (Shared Dynamic Resource Allocation)**: Ensures all pods in a
   PodGroup bind to a single DRA claim fulfilled from a single, shared,
   co-located resource (e.g., interconnected network interfaces or
   accelerators).

The scheduler is extended to interpret these constraints and find a "Placement"
(a subset of nodes and DRA resources) that satisfies them.

### User Stories (Optional)

#### Story 1: AI Training in a Single Rack

As a data scientist, I want to run a distributed training job where all pods
need to be located in the same server rack to minimize latency. I define a
`TopologyConstraint` on the Workload's PodGroup specifying the rack topology
label. The scheduler identifies a rack with sufficient capacity and schedules
all pods there at once.

#### Story 2: Workload using Interconnected DRA Devices

As a cluster administrator, I want to schedule a workload that requires a set of
specialized accelerators that are physically interconnected. I use a
`DRAConstraint` targeting a specific `ResourceClaimTemplate`. The scheduler
finds a set of DRA resources (ResourceSlice) that are co-located and binds the
workload's pods to them.

### Notes/Constraints/Caveats (Optional)

### Risks and Mitigations

- **Scheduling Latency:** Evaluating multiple placements involves running
  filter/score plugins multiple times.

  - **Mitigation:** Implement pre-filtering optimizations to reject infeasible
    placements early based on aggregate resource availability.

- **Complexity of Pod Group Scheduling:** Scheduling heterogeneous Pod Groups
  can be complex.

  - **Mitigation:** The initial version supports sequential processing of pods
    within a PodGroup, avoiding complex backtracking or parallel processing
    in the alpha release.

## Design Details

### Workload API Changes

The Workload API (KEP-4671) will be extended to allow specifying group-level
scheduling constraints. An optional `ScheduleConstraints` field is added to the
`PodGroup` spec.

```go
// PodGroup (definition from KEP-4671, with additions)
type PodGroup struct {
    Name *string

    // SchedulingConstraints defines group-level scheduling requirements,
    // including topology and DRA colocation.
    SchedulingConstraints *PodGroupSchedulingConstraints
}

// PodGroupSchedulingConstraints holds the scheduling constraints for the PodGroup.
type PodGroupSchedulingConstraints struct {
    // TopologyConstraints specifies desired topological placements for all pods
    // within this PodGroup.
    TopologyConstraints []TopologyConstraint

    // DRAConstraints specifies constraints on how Dynamic Resources are allocated
    // across the PodGroup.
    DRAConstraints []DRAConstraint
}

// TopologyConstraint describes a desired topological colocation for all pods in the PodGroup.
type TopologyConstraint struct {
    // Level specifies the key of the node label representing the topology domain.
    // All pods within the PodGroup must be colocated within the same domain instance.
    // Examples: "topology.kubernetes.io/rack"
    Level string
}

// DRAConstraint provides constraints on how specific DRA claims across the group should
// be fulfilled.
type DRAConstraint struct {
    // ResourceClaimName specifies the name of a specific ResourceClaim
    // within the PodGroup's pods that this constraint applies to.
    ResourceClaimName *string

    // ResourceClaimTemplateName specifies the name of a ResourceClaimTemplate.
    // This applies to all ResourceClaim instances generated from this template.
    ResourceClaimTemplateName *string
}
```

Note: For the initial alpha scope, only a single TopologyConstraint or
DRAConstraint will be supported.

### Scheduling Framework Extensions

The scheduler framework requires new plugin interfaces to handle "Placements". A
Placement represents a candidate domain (nodes and resources) for a PodGroup.

#### 1. Data Structures

```go
// PodGroupInfo holds information about a specific PodGroup within a Workload,
// including a reference to the Workload, the PodGroup's name, and its replica index.
// This struct is designed to be extensible with more fields in the future.
type PodGroupInfo struct {
    // WorkloadRef is a reference to the parent Workload object.
    WorkloadRef *workloadv1alpha1.Workload

    // PodGroupName is the name of the PodGroup.
    PodGroupName string

    // PodGroupReplicaIndex is the index of the PodGroup replica, as defined in KEP-4671.
    // This is relevant for PodGroups that have more than one replica.
    PodGroupReplicaIndex int

    // -- Add other fields below for future extensions --
}

// PodSetInfo holds information about a specific PodSet within a PodGroup,
// primarily the list of Pods.
// This struct is designed to be extensible with more fields in the future.
type PodSetInfo struct {
    // Pods is a list of Pod objects belonging to this PodSet.
    Pods []*corev1.Pod

    // -- Add other fields below for future extensions --
}

// Placement represents a candidate domain for scheduling a PodSet.
// It defines a set of nodes and/or proposed Dynamic Resource Allocation (DRA)
// resource bindings necessary to satisfy the PodSet's requirements within that domain.
type Placement struct {
    // NodeAffinity specifies the node constraints for this Placement.
    // For Topology this is derived from topology labels (e.g., all nodes with label
    // 'topology-rack: rack-1').
    // For DRA, this Affinity would be constructed based on nodeSelector from
    // DRA's AllocationResult from DRAAllocations.
    // All pods within the PodSet, when being evaluated against this Placement,
    // are restricted to the nodes matching this NodeAffinity.
    NodeAffinity *corev1.NodeAffinity

    // DRAAllocations details the proposed DRA resource assignments for
    // the ResourceClaims made by the PodSet. This field is primarily used
    // by DRA-aware plugins.
    DRAAllocations []DraClaimAllocation
}

// DraClaimAllocation maps a specific ResourceClaim name to a set of proposed
// device allocations. These allocations are tentative and used by the scheduler's
// AssumePlacement phase to simulate resource commitment.
type DraClaimAllocation struct {
    // ResourceClaimName is the name of the ResourceClaim within the PodSet's context
    // that these allocations are intended to satisfy.
    ResourceClaimName string

    // Allocation contains DRA AllocationResult structures, specifying devices
    // from ResourceSlices that are proposed to fulfill the ResourceClaim.
    // The scheduler will use this information in AssumePlacement to temporarily
    // consider these devices as allocated.
    Allocation dra.AllocationResult
}
```

#### 2. New Plugin Interfaces

**PlacementGenerator:** Generates candidate placements based on constraints.

```go
// PlacementGenerator is an interface for plugins that generate candidate Placements.
type PlacementGenerator interface {
    Name() string

    // GeneratePlacements generates a list of potential Placements for the given PodGroup and PodSet.
    // Each Placement represents a candidate set of resources (e.g., nodes matching a selector)
    // and potential DRA allocations where the PodSet might be scheduled.
    GeneratePlacements(ctx context.Context, state *framework.CycleState, podGroup *PodGroupInfo, podSet *PodSetInfo, parentPlacements []*Placement) ([]*Placement, *framework.Status)
}
```

**PlacementState:** Manages state changes (simulating binding) during
feasibility checks.

```go
// PlacementState is an interface for plugins that manage state changes
// when a Placement is being considered.
type PlacementState interface {
    Name() string

    // AssumePlacement temporarily configures the scheduling context to evaluate the feasibility
    // of the given Placement for the PodGroup and PodSet.
    AssumePlacement(ctx context.Context, state *framework.CycleState, podGroup *PodGroupInfo, podSet *PodSetInfo, placement *Placement) *framework.Status

    // RevertPlacement reverts the temporary scheduling context changes made by AssumePlacement.
    // This should be called after the evaluation of a Placement is complete to restore
    // the scheduler's state and allow other Placements to be considered.
    RevertPlacement(ctx context.Context, state *framework.CycleState, podGroup *PodGroupInfo, podSet *PodSetInfo, placement *Placement) *framework.Status
}
```

**PlacementScorer:** Scores feasible placements to select the best one.

```go
// PodSetAssignment represents the assignment of pods to nodes within a PodSet for a specific Placement.
type PodSetAssignment struct {
    // PodToNodeMap maps a Pod name (string) to a Node name (string).
    PodToNodeMap map[string]string
}

// PlacementScorer is an interface for plugins that score feasible Placements.
type PlacementScorer interface {
    Name() string

    // ScorePlacement calculates a score for a given Placement. This function is called in Phase 3
    // (Placement Scoring and Selection) only for Placements that have been deemed feasible
    // for all pods in the PodSet during Phase 2. The PodSetAssignment indicates the
    // node assigned to each pod within this Placement. The returned score is a float64,
    // with higher scores generally indicating more preferable Placements.
    // Plugins can implement various scoring strategies, such as bin packing to minimize
    // resource fragmentation.
    ScorePlacement(ctx context.Context, state *framework.CycleState, podGroup *PodGroupInfo, podSet *PodSetInfo, placement *Placement, podsAssignment *PodSetAssignment) (float64, *framework.Status)
}
```

### Scheduling Algorithm Phases

The algorithm proceeds in three main phases for a given Workload/PodGroup.

#### Phase 1: Candidate Placement Generation

- **Input:** PodGroupInfo and PodSetInfo.

- **Action:** Iterate over distinct values of the topology label (TAS) or
  available ResourceSlices (DRA).

- **Output:** A list of Placement objects.

- Example: If the label is rack, placements are generated for rack-1, rack-2,
  etc.

#### Phase 2: Pod-Level Filtering and Feasibility Check

- **Action:** For each generated Placement:

  1. Call `AssumePlacement` (binds context to the specific node selector/DRA
     resources).

  2. Iterate through every pod in the PodSet.

  3. Run standard Pod-level Filter and Score.

  4. Use internal logic to simulate placing the pod on a node.

  5. If all pods fit, the Placement is marked Feasible.

  6. Call `RevertPlacement`.

- **Potential Optimization:** Pre-filtering can check aggregate resources
  before running the full simulation.

- **Heterogeneous PodGroup Handling**: Sequential Processing will be used
  initially. Pods are processed sequentially; if any fail, the placement is
  rejected.

#### Phase 3: Placement Scoring and Selection

- **Action:** Call `ScorePlacement` for all feasible placements.

- **Selection:** Select the Placement with the highest score.

- **Binding:** Proceed to bind pods to the assigned nodes and resources.

### Scheduler Plugins

**TopologyPlacementPlugin (New)** Implements `PlacementGenerator`. Generates
Placements based on distinct values of the designated node label (TAS) .

**DRAPlugin (Extension)** Extended to implement `PlacementGenerator` and
`PlacementState`.

- **Generator:** Returns Placements derived from available ResourceSlices
  satisfying shared claims.

- **State:** Temporarily assigns AllocationResults to ResourceClaims during
  the Assume phase.

**PlacementBinPackingPlugin (New)** Implements `PlacementScorer`. Scores
Placements to maximize utilization (tightest fit) and minimize fragmentation.

### Potential Future Extensions (Beta Candidates)

The following features are out of scope for the initial Alpha implementation but
are considered for future releases (post-1.36):

1. **Prioritized Placement Scheduling:** Allowing a set of preferred placements
   with fallbacks (e.g., prefer Rack, fallback to Block). This would introduce
   a Rank field to the Placement struct.

2. **Optional/Preferred Scheduling Constraints:** Constraints that serve purely
   as scoring mechanisms without hard requirements.

3. **Multi-level Scheduling Constraints:** Handling nested constraints (e.g.,
   Block -> Rack). This would involve iterative placement generation and a
   Parent field in the Placement struct.

4. **Pod Group Replicas Support:** Optimizing scheduling for identical
   PodGroups (replicas) by scheduling the maximum feasible number of replicas
   within a single placement pass.

5. **Explicit Topology Definition:** Using a Custom Resource (NodeTopology) to
   define and alias topology levels, removing the need for users to know exact
   node label keys.

### Test Plan

[ ] I/we understand the owners of the involved components may require updates to
existing tests to make this code solid enough prior to committing the changes
necessary to implement this enhancement.

#### Prerequisite testing updates

#### Unit tests

- PlacementGenerator: Test generation of placements for various topology
  labels and DRA ResourceSlices.

- PlacementState: Verify AssumePlacement and RevertPlacement correctly modify
  and restore the CycleState.

- Algorithm Logic: Test the sequential processing of Placements and the
  selection logic based on scores.

- DRA Integration: specific tests for DRAConstraint resolution.

#### Integration tests

- Topology Awareness: Verify that pods with TopologyConstraint are correctly
  co-located on nodes sharing the label.

- DRA Awareness: Verify that pods with DRAConstraint are bound to shared
  ResourceSlices.

- Infeasibility: Verify that Workloads remain pending if no Placement
  satisfies the constraints.

#### e2e tests

- End-to-End Workload Scheduling: Submit a Workload with TopologyConstraint
  (e.g., Rack) and verify all pods land on the same rack.

- DRA Co-location: Submit a Workload requiring shared DRA devices and verify
  correct allocation and placement.

### Graduation Criteria

#### Alpha

- Feature implemented behind a feature flag.
- PodGroupSchedulingConstraints API defined.
- Basic topology (Node Label) and DRA constraints working.
- Initial unit and integration tests.

#### Beta

- Support for "Potential Future Extensions" (Prioritized placement, etc.)
  evaluated.
- Scalability tests on large clusters with high placement counts.
- Comprehensive e2e testing.

### Upgrade / Downgrade Strategy

This KEP is additive and can safely fallback to the original behavior on
downgrade.

When a user upgrades the cluster to the version which supports topology-aware
workload scheduling:

- they can enable scheduling plugins implementing new Scheduling Framework
  interfaces in kube-scheduler config
- they can start using the new API to create Workload objects with
  `schedulingConstraints` field
- scheduler will use enabled plugins to generate placements for Workload and
  check their feasibility

When user downgrades the cluster to the version that no longer supports
topology-aware workload scheduling:

- the `schedulingConstraints` field can no longer be set on the Workloads
  (the already set fields continue to be set though)
- scheduler will revert to the original behavior of scheduling pods belonging
  to a gang, without considering different potential placements.

### Version Skew Strategy

The feature is limited to the control plane, so the version skew with nodes
(kubelets) doesn't matter.

For the API changes, the old version of components (in particular
kube-apiserver) may not handle those. Thus, users should not set those fields
before confirming all control-plane instances were upgraded to the version
supporting those.

For the topology-aware workload scheduling itself, this is purely kube-scheduler
in-memory feature, so the skew doesn't matter (as there is always only a single
kube-scheduler instance being a leader).

## Production Readiness Review Questionnaire

<!--

Production readiness reviews are intended to ensure that features merging into
Kubernetes are observable, scalable and supportable; can be safely operated in
production environments, and can be disabled or rolled back in the event they
cause increased failures in production. See more in the PRR KEP at
https://git.k8s.io/enhancements/keps/sig-architecture/1194-prod-readiness.

The production readiness review questionnaire must be completed and approved
for the KEP to move to `implementable` status and be included in the release.

In some cases, the questions below should also have answers in `kep.yaml`. This
is to enable automation to verify the presence of the review, and to reduce review
burden and latency.

The KEP must have a approver from the
[`prod-readiness-approvers`](http://git.k8s.io/enhancements/OWNERS_ALIASES)
team. Please reach out on the
[#prod-readiness](https://kubernetes.slack.com/archives/CPNHUMN74) channel if
you need any help or guidance.
-->

### Feature Enablement and Rollback

<!--
This section must be completed when targeting alpha to a release.
-->

###### How can this feature be enabled / disabled in a live cluster?

<!--
Pick one of these and delete the rest.

Documentation is available on [feature gate lifecycle] and expectations, as
well as the [existing list] of feature gates.

[feature gate lifecycle]: https://git.k8s.io/community/contributors/devel/sig-architecture/feature-gates.md
[existing list]: https://kubernetes.io/docs/reference/command-line-tools-reference/feature-gates/
-->

- [ ] Feature gate (also fill in values in `kep.yaml`)
  - Feature gate name:
  - Components depending on the feature gate:
- [ ] Other
  - Describe the mechanism:
  - Will enabling / disabling the feature require downtime of the control
    plane?
  - Will enabling / disabling the feature require downtime or reprovisioning
    of a node?

###### Does enabling the feature change any default behavior?

<!--
Any change of default behavior may be surprising to users or break existing
automations, so be extremely careful here.
-->

###### Can the feature be disabled once it has been enabled (i.e. can we roll back the enablement)?

<!--
Describe the consequences on existing workloads (e.g., if this is a runtime
feature, can it break the existing applications?).

Feature gates are typically disabled by setting the flag to `false` and
restarting the component. No other changes should be necessary to disable the
feature.

NOTE: Also set `disable-supported` to `true` or `false` in `kep.yaml`.
-->

###### What happens if we reenable the feature if it was previously rolled back?

###### Are there any tests for feature enablement/disablement?

<!--
The e2e framework does not currently support enabling or disabling feature
gates. However, unit tests in each component dealing with managing data, created
with and without the feature, are necessary. At the very least, think about
conversion tests if API types are being modified.

Additionally, for features that are introducing a new API field, unit tests that
are exercising the `switch` of feature gate itself (what happens if I disable a
feature gate after having objects written with the new field) are also critical.
You can take a look at one potential example of such test in:
https://github.com/kubernetes/kubernetes/pull/97058/files#diff-7826f7adbc1996a05ab52e3f5f02429e94b68ce6bce0dc534d1be636154fded3R246-R282
-->

### Rollout, Upgrade and Rollback Planning

<!--
This section must be completed when targeting beta to a release.
-->

###### How can a rollout or rollback fail? Can it impact already running workloads?

<!--
Try to be as paranoid as possible - e.g., what if some components will restart
mid-rollout?

Be sure to consider highly-available clusters, where, for example,
feature flags will be enabled on some API servers and not others during the
rollout. Similarly, consider large clusters and how enablement/disablement
will rollout across nodes.
-->

###### What specific metrics should inform a rollback?

<!--
What signals should users be paying attention to when the feature is young
that might indicate a serious problem?
-->

###### Were upgrade and rollback tested? Was the upgrade->downgrade->upgrade path tested?

<!--
Describe manual testing that was done and the outcomes.
Longer term, we may want to require automated upgrade/rollback tests, but we
are missing a bunch of machinery and tooling and can't do that now.
-->

###### Is the rollout accompanied by any deprecations and/or removals of features, APIs, fields of API types, flags, etc.?

<!--
Even if applying deprecation policies, they may still surprise some users.
-->

### Monitoring Requirements

<!--
This section must be completed when targeting beta to a release.

For GA, this section is required: approvers should be able to confirm the
previous answers based on experience in the field.
-->

###### How can an operator determine if the feature is in use by workloads?

<!--
Ideally, this should be a metric. Operations against the Kubernetes API (e.g.,
checking if there are objects with field X set) may be a last resort. Avoid
logs or events for this purpose.
-->

###### How can someone using this feature know that it is working for their instance?

<!--
For instance, if this is a pod-related feature, it should be possible to determine if the feature is functioning properly
for each individual pod.
Pick one more of these and delete the rest.
Please describe all items visible to end users below with sufficient detail so that they can verify correct enablement
and operation of this feature.
Recall that end users cannot usually observe component logs or access metrics.
-->

- [ ] Events
  - Event Reason:
- [ ] API .status
  - Condition name:
  - Other field:
- [ ] Other (treat as last resort)
  - Details:

###### What are the reasonable SLOs (Service Level Objectives) for the enhancement?

<!--
This is your opportunity to define what "normal" quality of service looks like
for a feature.

It's impossible to provide comprehensive guidance, but at the very
high level (needs more precise definitions) those may be things like:
  - per-day percentage of API calls finishing with 5XX errors <= 1%
  - 99% percentile over day of absolute value from (job creation time minus expected
    job creation time) for cron job <= 10%
  - 99.9% of /health requests per day finish with 200 code

These goals will help you determine what you need to measure (SLIs) in the next
question.
-->

###### What are the SLIs (Service Level Indicators) an operator can use to determine the health of the service?

<!--
Pick one more of these and delete the rest.
-->

- [ ] Metrics
  - Metric name:
  - [Optional] Aggregation method:
  - Components exposing the metric:
- [ ] Other (treat as last resort)
  - Details:

###### Are there any missing metrics that would be useful to have to improve observability of this feature?

<!--
Describe the metrics themselves and the reasons why they weren't added (e.g., cost,
implementation difficulties, etc.).
-->

### Dependencies

<!--
This section must be completed when targeting beta to a release.
-->

###### Does this feature depend on any specific services running in the cluster?

<!--
Think about both cluster-level services (e.g. metrics-server) as well
as node-level agents (e.g. specific version of CRI). Focus on external or
optional services that are needed. For example, if this feature depends on
a cloud provider API, or upon an external software-defined storage or network
control plane.

For each of these, fill in the following—thinking about running existing user workloads
and creating new ones, as well as about cluster-level services (e.g. DNS):
  - [Dependency name]
    - Usage description:
      - Impact of its outage on the feature:
      - Impact of its degraded performance or high-error rates on the feature:
-->

### Scalability

<!--
For alpha, this section is encouraged: reviewers should consider these questions
and attempt to answer them.

For beta, this section is required: reviewers must answer these questions.

For GA, this section is required: approvers should be able to confirm the
previous answers based on experience in the field.
-->

###### Will enabling / using this feature result in any new API calls?

<!--
Describe them, providing:
  - API call type (e.g. PATCH pods)
  - estimated throughput
  - originating component(s) (e.g. Kubelet, Feature-X-controller)
Focusing mostly on:
  - components listing and/or watching resources they didn't before
  - API calls that may be triggered by changes of some Kubernetes resources
    (e.g. update of object X triggers new updates of object Y)
  - periodic API calls to reconcile state (e.g. periodic fetching state,
    heartbeats, leader election, etc.)
-->

###### Will enabling / using this feature result in introducing new API types?

<!--
Describe them, providing:
  - API type
  - Supported number of objects per cluster
  - Supported number of objects per namespace (for namespace-scoped objects)
-->

###### Will enabling / using this feature result in any new calls to the cloud provider?

<!--
Describe them, providing:
  - Which API(s):
  - Estimated increase:
-->

###### Will enabling / using this feature result in increasing size or count of the existing API objects?

<!--
Describe them, providing:
  - API type(s):
  - Estimated increase in size: (e.g., new annotation of size 32B)
  - Estimated amount of new objects: (e.g., new Object X for every existing Pod)
-->

###### Will enabling / using this feature result in increasing time taken by any operations covered by existing SLIs/SLOs?

<!--
Look at the [existing SLIs/SLOs].

Think about adding additional work or introducing new steps in between
(e.g. need to do X to start a container), etc. Please describe the details.

[existing SLIs/SLOs]: https://git.k8s.io/community/sig-scalability/slos/slos.md#kubernetes-slisslos
-->

###### Will enabling / using this feature result in non-negligible increase of resource usage (CPU, RAM, disk, IO, ...) in any components?

<!--
Things to keep in mind include: additional in-memory state, additional
non-trivial computations, excessive access to disks (including increased log
volume), significant amount of data sent and/or received over network, etc.
This through this both in small and large cases, again with respect to the
[supported limits].

[supported limits]: https://git.k8s.io/community//sig-scalability/configs-and-limits/thresholds.md
-->

###### Can enabling / using this feature result in resource exhaustion of some node resources (PIDs, sockets, inodes, etc.)?

<!--
Focus not just on happy cases, but primarily on more pathological cases
(e.g. probes taking a minute instead of milliseconds, failed pods consuming resources, etc.).
If any of the resources can be exhausted, how this is mitigated with the existing limits
(e.g. pods per node) or new limits added by this KEP?

Are there any tests that were run/should be run to understand performance characteristics better
and validate the declared limits?
-->

### Troubleshooting

<!--
This section must be completed when targeting beta to a release.

For GA, this section is required: approvers should be able to confirm the
previous answers based on experience in the field.

The Troubleshooting section currently serves the `Playbook` role. We may consider
splitting it into a dedicated `Playbook` document (potentially with some monitoring
details). For now, we leave it here.
-->

###### How does this feature react if the API server and/or etcd is unavailable?

###### What are other known failure modes?

<!--
For each of them, fill in the following information by copying the below template:
  - [Failure mode brief description]
    - Detection: How can it be detected via metrics? Stated another way:
      how can an operator troubleshoot without logging into a master or worker node?
    - Mitigations: What can be done to stop the bleeding, especially for already
      running user workloads?
    - Diagnostics: What are the useful log messages and their required logging
      levels that could help debug the issue?
      Not required until feature graduated to beta.
    - Testing: Are there any tests for failure mode? If not, describe why.
-->

###### What steps should be taken if SLOs are not being met to determine the problem?

## Implementation History

<!--
Major milestones in the lifecycle of a KEP should be tracked in this section.
Major milestones might include:
- the `Summary` and `Motivation` sections being merged, signaling SIG acceptance
- the `Proposal` section being merged, signaling agreement on a proposed design
- the date implementation started
- the first Kubernetes release where an initial version of the KEP was available
- the version of Kubernetes where the KEP graduated to general availability
- when the KEP was retired or superseded
-->

## Drawbacks

- **Complexity:** This proposal adds significant logic to the kube-scheduler
  framework, specifically the "Placement" abstraction and the simulation loop
  (Phase 2).

- **Performance:** Generating and simulating a large number of Placements
  (e.g., every rack in a massive cluster) could be computationally expensive.

  - **Mitigation:** Pre-filtering of Placements will be implemented to discard
    clearly infeasible Placements (insufficient total resources) before the
    expensive pod-level simulation.

## Alternatives

### Pod Inter-Affinities

Currently, users may attempt to simulate gang scheduling using podAffinity (to
co-locate pods) or podAntiAffinity.

- **Pros:** Native to Kubernetes, no new CRDs.
- **Cons:** Affinity is evaluated per-Pod at the time of that Pod's
  scheduling. It does not look ahead. This means that the scheduler might
  place the first Pod on a node that satisfies its immediate affinity needs
  but prevents the rest of the group from scheduling (e.g., locking a topology
  domain that is too small for the rest of the group).

### Standalone Schedulers (e.g., Volcano)

Users can run a secondary scheduler like Volcano or Yunikorn.

- **Pros:** Feature-rich, mature for batch workloads.
- **Cons:** Operationally complex (two schedulers), race conditions when
  sharing cluster resources, and lack of integration with standard Kubernetes
  features like common admission controllers or newer features like DRA
  (initially).

## Infrastructure Needed (Optional)

<!--
Use this section if you need things from the project/SIG. Examples include a
new subproject, repos requested, or GitHub details. Listing these here allows a
SIG to get the process for these resources started right away.
-->
