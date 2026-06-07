# KEP-5732: Topology-aware workload scheduling

<!-- toc -->
- [Release Signoff Checklist](#release-signoff-checklist)
- [Summary](#summary)
- [Motivation](#motivation)
  - [Goals](#goals)
  - [Non-Goals](#non-goals)
- [Proposal](#proposal)
  - [User Stories (Optional)](#user-stories-optional)
    - [Story 1: AI Training in a Single Rack](#story-1-ai-training-in-a-single-rack)
  - [Notes/Constraints/Caveats (Optional)](#notesconstraintscaveats-optional)
  - [Risks and Mitigations](#risks-and-mitigations)
- [Design Details](#design-details)
  - [Workload API Changes](#workload-api-changes)
  - [Scheduling Framework Extensions](#scheduling-framework-extensions)
    - [1. Data Structures](#1-data-structures)
    - [2. New Plugin Interfaces](#2-new-plugin-interfaces)
  - [Scheduling Algorithm Phases](#scheduling-algorithm-phases)
    - [Phase 1: Candidate Placement Generation](#phase-1-candidate-placement-generation)
    - [Phase 2: Pod-Level Filtering and Feasibility Check](#phase-2-pod-level-filtering-and-feasibility-check)
    - [Phase 3: Placement Scoring and Selection](#phase-3-placement-scoring-and-selection)
  - [Scheduler Plugins](#scheduler-plugins)
  - [Beta Extensions](#beta-extensions)
  - [Future Extensions](#future-extensions)
  - [Test Plan](#test-plan)
    - [Prerequisite testing updates](#prerequisite-testing-updates)
    - [Unit tests](#unit-tests)
    - [Integration tests](#integration-tests)
    - [e2e tests](#e2e-tests)
  - [Graduation Criteria](#graduation-criteria)
    - [Alpha](#alpha)
    - [Beta](#beta)
    - [GA](#ga)
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
  - [Pod Inter-Affinities](#pod-inter-affinities)
  - [Standalone Schedulers (e.g., Volcano)](#standalone-schedulers-eg-volcano)
- [Infrastructure Needed (Optional)](#infrastructure-needed-optional)
<!-- /toc -->

## Release Signoff Checklist

Items marked with (R) are required *prior to targeting to a milestone / release*.

- [X] (R) Enhancement issue in release milestone, which links to KEP dir in [kubernetes/enhancements] (not the initial KEP PR)
- [X] (R) KEP approvers have approved the KEP status as `implementable`
- [X] (R) Design details are appropriately documented
- [X] (R) Test plan is in place, giving consideration to SIG Architecture and SIG Testing input (including test refactors)
  - [ ] e2e Tests for all Beta API Operations (endpoints)
  - [ ] (R) Ensure GA e2e tests meet requirements for [Conformance Tests](https://github.com/kubernetes/community/blob/master/contributors/devel/sig-architecture/conformance-tests.md)
  - [ ] (R) Minimum Two Week Window for GA e2e tests to prove flake free
- [X] (R) Graduation criteria is in place
  - [ ] (R) [all GA Endpoints](https://github.com/kubernetes/community/pull/1806) must be hit by [Conformance Tests](https://github.com/kubernetes/community/blob/master/contributors/devel/sig-architecture/conformance-tests.md) within one minor version of promotion to GA
- [X] (R) Production readiness review completed
- [X] (R) Production readiness review approved
- [X] "Implementation History" section is up-to-date for milestone
- [X] User-facing documentation has been created in [kubernetes/website], for publication to [kubernetes.io]
- [X] Supporting documentation—e.g., additional design documents, links to mailing list discussions/SIG meetings, relevant PRs/issues, release notes

<!--
**Note:** This checklist is iterative and should be reviewed and updated every time this enhancement is being considered for a milestone.
-->

[kubernetes.io]: https://kubernetes.io/
[kubernetes/enhancements]: https://git.k8s.io/enhancements
[kubernetes/kubernetes]: https://git.k8s.io/kubernetes
[kubernetes/website]: https://git.k8s.io/website

## Summary

This KEP describes the architectural design and implementation details for
integrating a Topology-Aware workload scheduling algorithm into
the Kubernetes kube-scheduler to address the complex placement requirements of
modern, high-performance distributed applications.

The proposed topology algorithm leverages the workload-oriented scheduling
lifecycle introduced in KEP-4671, rather than fundamentally altering the scheduling
loop itself. It extends this foundation by enabling the evaluation of scheduling
options within specific "Placements" (subsets of the cluster). These Placements
represent candidate domains (sets of
nodes) where the entire workload is theoretically feasible. The
scheduler then simulates the placement of the full group of pods within these
domains, utilizing existing filtering and scoring logic to ensure high-fidelity
decisions before committing resources.

This design introduces specific extensions to the Kubernetes Workload API to
support `TopologyConstraints`, defines new interfaces
within the Scheduling Framework (`PlacementGeneratePlugin`,
`PlacementScorePlugin`), and details the algorithmic flow required to schedule Pod
Groups while maintaining compatibility with the scheduler's existing ecosystem.

## Motivation

Distributed workloads, particularly those driving the current AI/ML era, often
require high-bandwidth and low-latency communication between multiple pods to
function efficiently. While the [KEP-4671: Workload API](https://kep.k8s.io/4671)
makes the first step towards managing these applications as cohesive units, it
primarily establishes the API structure. For workloads sensitive to inter-pod
communication, simply grouping pods is insufficient; their physical placement
within the cluster's network topology is a decisive factor in their performance.

In this KEP, we propose an algorithm for topology-aware scheduling
that operates directly within the Kubernetes kube-scheduler. The core objective
is to ensure that pods belonging to a Workload are co-located within optimal
topological domains - such as specific racks or blocks that require cohesive management.
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
introduce operational complexity. We believe that embedding topology
awareness deeply into the kube-scheduler is critical enough to warrant
standardization. This integration allows the algorithm to leverage the full
fidelity of the scheduler's existing pod-level filtering and scoring plugins,
ensuring highly accurate feasibility checks and placement outcomes without the
need for external dependencies.

### Goals

- To enhance kube-scheduler to be able to perform topology-aware
  scheduling for multi-pod workloads, as defined by the Workload API
  ([KEP-4671](https://kep.k8s.io/4671)).
- To optimize the placement of distributed workloads by co-locating pods based
  on network topology.
- To introduce new extension points and phases within the Kubernetes scheduler
  framework to support the concept of "Placements" (candidate sets of nodes).
- To define the required changes to the Workload API (KEP-4671) to support
  Topology scheduling constraints.
- To leverage the scheduler's existing pod-level filtering and scoring logic
  within the evaluation of each Placement.
- To provide a flexible framework extensible by plugins for various topology
  sources (e.g., node labels) and resource types (e.g., DRA).

### Non-Goals

- To define the required changes to the Workload API (KEP-4671) to support
  ResourceClaims for DRA-aware workload scheduling and their scheduling.
  These changes have been proposed in a separate KEP:
  [KEP-5729: DRA: ResourceClaim Support for Workloads](https://kep.k8s.io/5729)
- To replace the functionality of external workload queueing and admission
  control systems like Kueue. This proposal focuses on the in-scheduler
  placement decision for a single Workload at a time.
- To implement Workload-level queueing, fairness, or resource quotas within
  kube-scheduler.
- To handle all aspects of the workload lifecycle management beyond
  scheduling.
- To implement Workload-level preemption logic.
- To support complex multi pod dependency resolution with backtracking or
  parallel processing in the initial version.
- To automatically discover network topology; the mechanisms rely on topology
  information being present (e.g., via node labels).

## Proposal

This proposal introduces an API to define constraints on a PodGroup (a
collection of pods within a Workload) requiring it to be scheduled onto a
specific subset of nodes or resources.

We support one type of constraints:

1. **Topology Constraint (Node Label Co-location)**: Ensures all pods in a
   PodGroup are placed onto nodes sharing a common topological characteristic
   (e.g., same rack), defined by a specific node label.

The scheduler is extended to interpret these new PodGroup level scheduling constraints
and similarly to scheduling pods on nodes (available scheduling options), find
a "Placement" for this PodGroup among the feasible options (subsets of nodes)
that satisfies them.

### User Stories (Optional)

#### Story 1: AI Training in a Single Rack

As a data scientist, I want to run a distributed training job where all pods
need to be located in the same server rack to minimize latency. I define a
`TopologyConstraint` on the Workload's PodGroup specifying the rack topology
label. The scheduler identifies a rack with sufficient capacity and schedules
all pods there at once.

### Notes/Constraints/Caveats (Optional)

### Risks and Mitigations

- **Scheduling Latency:** Evaluating multiple placements involves running
  filter/score plugins multiple times (multiple attempts to schedule a PodGroup considering all topology options).

  - **Mitigation:** Implement early placement rejection optimization to stop
    feasibility checks for placements which cannot fit gang scheduled PodGroups.

- **Complexity of Pod Group Scheduling:** Scheduling heterogeneous Pod Groups
  can be complex.

  - **Mitigation:** We support only sequential processing of pods
    within a PodGroup, avoiding complex backtracking or parallel processing.

## Design Details

### Workload API Changes

The Workload API (KEP-4671) and PodGroup API (KEP-5832) will be extended to allow specifying group-level
scheduling constraints. An optional `ScheduleConstraints` field is added to the
`PodGroupTemplate` spec and `PodGroupSpec`.

```go
// PodGroupTemplate (definition from KEP-4671, with additions)
type PodGroupTemplate struct {
    // Existing fields go here

    // SchedulingConstraints defines optional scheduling constraints (e.g. topology) for this PodGroupTemplate.
    // This field is only available when the TopologyAwareWorkloadScheduling feature gate is enabled.
    SchedulingConstraints *PodGroupSchedulingConstraints
}

// PodGroup (definition from KEP-5832, with additions)
type PodGroupSpec struct {
    // Existing fields go here

    // SchedulingConstraints defines optional scheduling constraints (e.g. topology) for this PodGroup.
    // Controllers are expected to fill this field by copying it from a PodGroupTemplate.
    // This field is immutable.
    // This field is only available when the TopologyAwareWorkloadScheduling feature gate is enabled.
    SchedulingConstraints *PodGroupSchedulingConstraints
}

// PodGroupSchedulingConstraints defines scheduling constraints (e.g. topology) for a PodGroup.
type PodGroupSchedulingConstraints struct {
    // Topology defines the topology constraints for the pod group.
    // Currently only a single topology constraint can be specified. This may change in the future.
    Topology []TopologyConstraint
}

// TopologyConstraint defines a topology constraint for a PodGroup.
type TopologyConstraint struct {
    // Key specifies the key of the node label representing the topology domain.
    // All pods within the PodGroup must be colocated within the same domain instance.
    // Different PodGroups can land on different domain instances even if they derive from the same PodGroupTemplate.
    // Examples: "topology.kubernetes.io/rack"
    Level string
}
```

The Workload API changes for DRA-aware scheduling, including the definition of
DRA constraints and their scheduling, are out of scope of this KEP. These changes
will be defined in a separate KEP: 
[KEP-5729: DRA: ResourceClaim Support for Workloads](https://kep.k8s.io/5729).

Note: For this KEP scope, only a single TopologyConstraint will be
supported.

### Scheduling Framework Extensions

The scheduler framework requires new plugin interfaces to handle "Placements". A
Placement represents a candidate domain (nodes and resources) for a PodGroup.

#### 1. Data Structures

```go
// Placement determines the resources to be considered when scheduling a pod group.
// Pod group scheduling cycle can check multiple placements and select the one that results
// in the best pod assignments.
type Placement struct {
    // Name uniquely identifies the placement.
    // This is used for diagnostics and debugability.
    // The choice of the name is up to the PlacementGeneratePlugin.
    Name string

    // Nodes specifies the nodes that are valid for this placement.
    // Scheduler will try to schedule the pod group using only those nodes.
    Nodes []NodeInfo
}
```

#### 2. New Plugin Interfaces

**PlacementGeneratePlugin:** Generates candidate placements based on constraints.

```go
// GeneratePlacementsResult represents the result of the PlacementGeneratePlugin.
type GeneratePlacementsResult struct {
    // Placements is the set of placements that the plugin wants to partition the resources into.
    // The partitions can overlap.
    //
    // To represent no valid partitions, set the array to nil or empty.
    Placements []*Placement
}

// PlacementGeneratePlugin is an interface for plugins that generate candidate Placements.
type PlacementGeneratePlugin interface {
    Plugin

    // GeneratePlacements generates a list of potential Placements for the given PodGroup within the parent placement.
    // Each Placement represents a candidate set of resources, e.g., nodes matching a selector.
    GeneratePlacements(ctx context.Context, state PodGroupCycleState, podGroup PodGroupInfo, parentPlacement *Placement) (*GeneratePlacementsResult, *Status)
}
```

**PlacementScorePlugin:** Scores feasible placements to select the best one.

```go
// PlacementScore stores result of a placement score plugin to be later used for normalization.
type PlacementScore struct {
    // Placement is the placement for which the score was computed
    Placement *Placement

    // Score is the score for a given placement, which is used to rank the placements and pick the best one.
    Score int64
}

// PlacementScoreExtensions is an interface for PlacementScore extended functionality.
type PlacementScoreExtensions interface {
  	// NormalizePlacementScore is called for all placement scores produced by the same plugin's "ScorePlacement"
  	// method. A successful run of NormalizePlacementScore will update the scores list and return
  	// a success status.
  	NormalizePlacementScore(ctx context.Context, state PodGroupCycleState, podGroup PodGroupInfo, placementScores []PlacementScore) *Status
}

// PlacementScorePlugin is an interface for plugins that score feasible Placements.
type PlacementScorePlugin interface {
    Plugin

    // ScorePlacement calculates a score for a given Placement.
    // This function is called only for Placements that have been deemed feasible for the sufficient number of pods in the PodGroup scheduling cycle.
    // The PodGroupAssignments indicates the node assigned to each pod within this Placement.
    // The returned score is a int64 with higher scores generally indicating more preferable Placements.
    // Plugins can implement various scoring strategies, such as bin packing to minimize resource fragmentation.
    ScorePlacement(ctx context.Context, state PlacementCycleState, podGroup PodGroupInfo, placement *PodGroupAssignments) (int64, *Status)

    // PlacementScoreExtensions returns a PlacementScoreExtensions interface if it implements one, or nil if does not.
    PlacementScoreExtensions() PlacementScoreExtensions
}
```

### Scheduling Algorithm Phases

The algorithm proceeds in three main phases for a given PodGroup.

#### Phase 1: Candidate Placement Generation

- **Input:** PodGroupInfo.

- **Action:** Iterate over distinct values of the topology label (TAS).

- **Output:** A list of Placement objects.

- Placement generation is provided with a parent placement which
  includes all nodes in the cluster giving a plugin 
  a chance to get the list of nodes which should be considered when
  generating placements.

- Example: If the label is rack, placements are generated for rack-1, rack-2,
  etc.

#### Phase 2: Pod-Level Filtering and Feasibility Check

- **Action:** For each generated Placement:

  1. Run default workload scheduling algorithm with the given set of nodes.

  2. If all required pods (at least `minCount` pods for Gang scheduling policy
     and at least one pod for Basic scheduling policy) fit, the Placement
     is marked Feasible.

- **Basic Scheduling Policy Handling:** The current algorithm may exhibit
  inconsistent behavior when used with the PodGroup Basic Scheduling Policy.
  Because the scheduler may only observe a subset of pods when scheduling
  a PodGroup, placement feasibility is only validated for those specific
  pods rather than the entire group. This limitation may be addressed in
  future releases; currently, scheduling gates may be used as a partial
  mitigation.

- **Heterogeneous PodGroup Handling**: Sequential processing will be used
  initially. Pods are processed sequentially; if any fail, the placement is
  rejected.

#### Phase 3: Placement Scoring and Selection

- **Action:** Call `ScorePlacement` for all feasible placements.

- **Selection:** Select the Placement with the highest score.

- **Binding:** Proceed to bind pods to the assigned nodes and resources using
  pod-by-pod scheduling logic with each pod prebound to the selected node
  by setting `nominatedNodeName` value.

### Scheduler Plugins

**TopologyPlacementGenerator (New)** Implements `PlacementGeneratePlugin`. Generates
Placements based on distinct values of the designated node label (TAS).

**NodeResourcesFit (Existsing)** Implements `PlacementScorePlugin`. Scores
Placements to maximize utilization (tightest fit) and minimize fragmentation.

**PodGroupPodsCount (New)** Implements `PlacementScorePlugin`. Scores
Placements based on the number of pods fiting into each Placement.

### Beta Extensions

In the beta release, the following features and performance optimizations have been
introduced to improve the throughput, efficiency, and flexibility of Topology-Aware
Scheduling:

1. Multiple PlacementGeneratePlugins
   
   The beta version supports defining multiple PlacementGeneratePlugins. When multiple
   such plugins are configured, the scheduler framework runs them independently. It then
   merges their results by calculating non-empty intersections of the placements returned
   by the different plugins, allowing the system to easily handle complex placement
   requirements.

2. Early Rejection of Placements

  To optimize the scheduling cycle, the scheduler will abort placement evaluation early
  if there are not enough remaining pods to satisfy a PodGroup's minCount. Previously,
  all pods were evaluated, which missed an opportunity for optimization. This capability
  is implemented via a new "pseudo extension point" called PlacementFeasiblePlugin
  (similar to PodGroupPostFilter) to break out of the pod group scheduling cycle.
  
  More details on this optimization can be found in [KEP-4671: Workload API](https://kep.k8s.io/4671).

3. Limit on the Number of Checked Placements

  To ensure high pod throughput, especially in large clusters, the beta version
  introduces a limit on the number of scored placements, similar to how the scheduler
  limits the number of scored nodes in non-TAS scenarios. This limit is controlled via
  a new scheduler parameter PercentageOfPlacementsToScore.

  By default, an adaptive limit will be applied based on the total number of nodes to
  evaluate across all generated placements. The limit interpolates from 100% for very
  small clusters to 10% for clusters with 5000 nodes, with a hard lower bound of 5%.
  
  This strategy ensures predictable, robust throughput of topology-aware scheduling.

### Future Extensions

[KEP-5729: DRA: ResourceClaim Support for Workloads](https://kep.k8s.io/5729)
introduced support for PodGroup-level ResourceClaims. In its alpha version,
KEP-5729 does not include specialized scheduling logic for these PodGroup-level
claims. As a future enhancement, the scheduler should evaluate user-defined
DRA claims during placement decisions to ensure workloads are assigned to nodes
capable of satisfying their collective resource requirements. This logic will
build upon the extension points introduced in this KEP. Specifically,
the DRAPlugin will be enhanced to generate placements based on the ResourceClaim
objects associated with a PodGroup. The plugin will interact directly with
the DRA framework to verify that the selected placement can fulfill the workload's
resource demands, as defined by its ResourceClaim.

The following features are out of scope for this KEP but will be implemented in
future separate KEPs improving and extending the proposed functionality:

1. **Prioritized Placement Scheduling:** Allowing a set of preferred placements
   with fallbacks (e.g., prefer Rack, fallback to Block). This would introduce
   a Rank field to the Placement struct.

2. **Optional/Preferred Scheduling Constraints:** Constraints that serve purely
   as scoring mechanisms without hard requirements.

3. **Multi-level Scheduling Constraints:** Handling nested constraints (e.g.,
   Block -> Rack). This would involve iterative placement generation and a
   Parent field in the Placement struct.

4. **Pod Group Replicas Optimization:** Optimizing scheduling for identical
   PodGroups (replicas) by scheduling the maximum feasible number of replicas
   within a single placement pass.

5. **Explicit Topology Definition:** Using a dedicated resource (NodeTopology) to
   define and alias topology levels, removing the need for users to know exact
   node label keys and opening additional optimization and validation options.

### Test Plan

[X] I/we understand the owners of the involved components may require updates to
existing tests to make this code solid enough prior to committing the changes
necessary to implement this enhancement.

#### Prerequisite testing updates

N/A

#### Unit tests

- PlacementGeneratePlugin: Test generation of placements for various topology
  labels.
  
- PlacementScorePlugin: Test scoring of placements.

- Algorithm Logic: Test the sequential processing of Placements and the
  selection logic based on scores.

#### Integration tests

For Alpha we implemented integration tests to ensure basic functionalities of
to

Topology Aware Scheduling With Gang Policy:

- Resource Availability: Schedules successfully if space permits, or remains
  pending if resources are insufficient or consumed by existing pods.

- Scoring & Placement: Places pods based on the highest score or allocation
  percentage using default scoring algorithms.

- Multiple Gangs: Handles consecutive scheduling across the same/different
  racks, varying topology keys, and capacity bottlenecks.

- Edge Cases: Verifies behavior when minCount is less than the total pod
  count, and when preemption is required to free up resources.

Topology Aware Scheduling With Basic Policy:

- Standard Placement: Schedules entire podgroups on appropriate racks, leveraging
  placement scores and allocation percentages (even with pre-existing pods).

- Capacity Constraints: Skips or partially schedules pods when a single rack or
  the entire cluster lacks the required resources.

- Multiple PodGroups: Manages consecutive scheduling across various racks,
  zones, and restrictive cluster capacities.

- Dynamic Updates: Successfully resumes scheduling pending pods once blocked
  resources become available again.

Those tests are located at [tas_test.go‎](https://github.com/kubernetes/kubernetes/blob/eb01d62d2676cfe009382cadd5d65c2ad654998d/test/integration/scheduler/podgroup/topology_aware_scheduling/tas_test.go‎).

#### e2e tests

- End-to-End Workload Scheduling: Submit a Workload with TopologyConstraint
  (e.g., Rack) and verify all pods land on the same rack.

### Graduation Criteria

#### Alpha

- Feature implemented behind a feature flag.
- PodGroupSchedulingConstraints API defined.
- Basic topology (Node Label) working.
- Initial unit and integration tests.

#### Beta

- Scalability tests on large clusters with high placement counts.
- Comprehensive e2e testing.
- Cluster autoscaling compomnents are aware of workload topology constraints.

#### GA

- All issues and gaps identified as feedback during beta are resolved.
- Promote the e2e API tests to conformance.

### Upgrade / Downgrade Strategy

This KEP is additive and can safely fallback to the original behavior on
downgrade.

When a user upgrades the cluster to the version which supports topology-aware
workload scheduling:

- they can enable scheduling plugins implementing new Scheduling Framework
  interfaces in kube-scheduler config
- they can start using the new API to create PodGroup objects with
  `schedulingConstraints` field
- scheduler will use enabled plugins to generate placements for PodGroup and
  check their feasibility

When user downgrades the cluster to the version that no longer supports
topology-aware workload scheduling:

- the `schedulingConstraints` field can no longer be set on the PodGroups
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

### Feature Enablement and Rollback

###### How can this feature be enabled / disabled in a live cluster?

- [X] Feature gate (also fill in values in `kep.yaml`)
  - Feature gate name: TopologyAwareWorkloadScheduling
  - Components depending on the feature gate:
    - kube-apiserver
    - kube-scheduler
- [ ] Other
  - Describe the mechanism:
  - Will enabling / disabling the feature require downtime of the control
    plane?
  - Will enabling / disabling the feature require downtime or reprovisioning
    of a node?

###### Does enabling the feature change any default behavior?

No - even with a feature enabled scheduler by default will use existing scheduling
algorithm to schedule workloads. Only when workload will have an explicit topology
constraint set an alternative algorithm will be used.

###### Can the feature be disabled once it has been enabled (i.e. can we roll back the enablement)?

Yes, the workload scheduling algorithm changes can be disabled by simply disabling
the feature gate in kube-scheduler.

The new API changes can also be disabled by disabling the feature gate in kube-apiserver.
However that doesn't result in clearing the new fields for workloads that already have
them set in the storage.

###### What happens if we reenable the feature if it was previously rolled back?

The feature starts working again.

###### Are there any tests for feature enablement/disablement?

The scheduler algorithm changes are purely in-memory and doesn't require any dedicated
enablement/disablement tests - the logic will be covered by regular feature tests.

For the newly introduced API fields, dedicated enablement/disablement tests at the
kube-apiserver registry layer will be added.

### Rollout, Upgrade and Rollback Planning

###### How can a rollout or rollback fail? Can it impact already running workloads?

Workloads that do not use the PodGroup APIs should not be impacted, since the
functionality remains unchanged for them. During a rolling upgrade, if the
active scheduler instance has the feature disabled, it will schedule pods using the
standard non-TAS method, falling back to a default workload scheduling algorithm.

This results in a fallback to the status quo behavior, meaning that pods will be
still scheduled, but PodGroup-level toplogy scheduling constraints won't be applied.

###### What specific metrics should inform a rollback?

- `scheduler_schedule_attempts_total{result="error"}`: A sudden spike indicates internal
  errors or panics within the scheduling loop, possibly caused by the TAS logic.
- `process_start_time_seconds`: Frequent resets of this metric indicate that the scheduler
  process is crashing and restarting (crash loop).
- `scheduler_podgroup_scheduling_duration_seconds`: A significant regression in P99 latency for
  pod groups would indicate that the overhead of the new logic is unacceptable.

###### Were upgrade and rollback tested? Was the upgrade->downgrade->upgrade path tested?

We'll perform manual testing of the upgrade -> downgrade -> upgrade path using the following sequence:

1. Start a local Kubernetes v1.36 cluster with `TopologyAwareWorkloadScheduling` feature
   gate disabled and `GenericWorkload` feature gate enabled (default behavior).
2. Attempt to create a PodGroup object with `spec.schedulingConstraints.topologyConstraints[0].level`
   set to `kubernetes.io/hostname`.
3. The `spec.schedulingConstraints` field is dropped by the API server. The PodGroup is created
   successfully but without the `spec.schedulingConstraints` reference.
4. Restart/Upgrade API Server and Scheduler to v1.37 with feature gates enabled.
5. Create five PodGroup objects: `tas-test-A` to `tas-test-E` all with `minCount`
   set to 2 and `spec.schedulingConstraints.topologyConstraints[0].level` set to `kubernetes.io/hostname`.
6. Create `test-pod-1` and `test-pod-2` Pods with `spec.schedulingGroup` pointing to `tas-test-A`
   and node affinities pointing to two different nodes.
7. The Pod stays in `Pending` state (waiting for the TAS scheduling). Verify that
   `scheduler_pending_entities{type="podgroup", queue="gated"}` metric is incremented.
8. Create `test-pod-3` and `test-pod-4` Pods with `spec.schedulingGroup` pointing to `tas-test-B`
   and node affinities pointing to the same node.
9. Both pods are scheduled successfully (TAS works). 
10. Downgrade API Server and Scheduler to v1.36 with feature gates disabled.
11. Create `test-pod-5` and `test-pod-6` Pods with `spec.schedulingGroup` pointing to `tas-test-C`
    and node affinities pointing to two different nodes. Note: We use a pod group created in step 5
    because creating new PodGroup objects with schedulingConstraints is disabled.
13. The pods are scheduled successfully (schedulingConstraints logic is ignored because the schedulingGroup
    field is dropped by the v1.36 API server).
15. Upgrade API Server and Scheduler back to v1.37 with feature gates enabled.
16. Repeat steps 6 to 9 with `tas-test-D` and `tas-test-E`.

###### Is the rollout accompanied by any deprecations and/or removals of features, APIs, fields of API types, flags, etc.?

No.

### Monitoring Requirements

###### How can an operator determine if the feature is in use by workloads?

Operators can check the new `plugin_execution_duration_seconds{plugin="TopologyPlacementGenerator", extension_point="GeneratePlacements"}`
metric. A value greater than zero indicates that the scheduler is using TAS .

Alternatively, checking for the existence of `PodGroup` via `kubectl get podgroups`,
and checking the `PodGroup.spec.spec.schedulingConstraints` field confirms that users
are actively using the feature.

###### How can someone using this feature know that it is working for their instance?

- [X] API .status
  - Object: PodGroup
  - Condition Name: `PodGroupScheduled`

###### What are the reasonable SLOs (Service Level Objectives) for the enhancement?

Since there are no formal SLOs for the kube-scheduler apart from scalability SLOs, we define the objectives for this
feature primarily in terms of non-regression to ensure the toplogy aware scheduling does not degrade the performance
of the standard scheduling loop.

###### What are the SLIs (Service Level Indicators) an operator can use to determine the health of the service?

- Scheduling Throughput: There should be no significant regression in the system-wide scheduling throughput (pods/s) 
  when scheduling PodGroup using TAS compared to scheduling an equivalent number of individual pods.
  This can be measured by the number of Pod binding API calls arriving to the API server
  (`apiserver_request_total{resource="pods", subresource="binding"}`).

- Scheduling Latency: There should be no significant regression in pod scheduling latency 
  (`scheduler_pod_scheduling_duration_seconds`) for both TAS and non-TAS pods compared to the baseline
  (behavior with the feature disabled).

###### Are there any missing metrics that would be useful to have to improve observability of this feature?

No.

### Dependencies

<!--
This section must be completed when targeting beta to a release.
-->

###### Does this feature depend on any specific services running in the cluster?

No dependencies other than the components where the feature is implemented
(kube-apiserver and kube-scheduler).

### Scalability

###### Will enabling / using this feature result in any new API calls?

No.

###### Will enabling / using this feature result in introducing new API types?

No.

###### Will enabling / using this feature result in any new calls to the cloud provider?

No.

###### Will enabling / using this feature result in increasing size or count of the existing API objects?

Using this feature will require setting topology constraint on Workload object.
The related increase in size of the Workload object should however be negligible.

###### Will enabling / using this feature result in increasing time taken by any operations covered by existing SLIs/SLOs?

Although the proposed algorithm was designed with performance in mind, the scheduling
latency / Pod Startup SLO may potentially increase especially for large clusters and
fine grained topology constraints.

We will measure the exact impact using performance benchmarks and scalability tests and
update the section based on the results. The complexity of scheduling of a single worklaod
is O(#pods * #nodes), which is comparable to the algorithm not using topology constraints,
so the benchmarks are primarily to validate the potential inefficiencies of the implementation.

###### Will enabling / using this feature result in non-negligible increase of resource usage (CPU, RAM, disk, IO, ...) in any components?

For large clusters and fine grained toplogy constraints we may observe some increase in CPU
and RAM usage for kube-scheduler. The exact scale of this increase will be confirmed by
scalability tests.

###### Can enabling / using this feature result in resource exhaustion of some node resources (PIDs, sockets, inodes, etc.)?

No.

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

The behavior is consistent with the status quo. Since the scheduler cannot bind pods or update statuses without the
API server, any in-flight TAS scheduling will eventually fail at the binding/update stage. These attempts will be
retried with standard exponential backoff once connectivity is restored.

###### What are other known failure modes?

- Pods Pending Indefinitely - PodGroup cannot fit in any Placement (Resource Constraints)
  - Detection: Check Pod Events/Status. Expected reason: a message indicating that minCount pods could not be
    scheduled.
  - Metrics: `scheduler_podgroup_schedule_attempts_total` with result unschedulable.
  - Mitigations:
    - Scale up the cluster (add nodes) or delete other real-workloads to free up space.
    - If intended, recreate the PodGroup object without `schedulingConstraints`
      to disable TAS scheduling (fallback to default workload scheduling) if acceptable.
  - Diagnostics:
    - Scheduler logs at v=4 searching for "podgroup" to see detailed reasons why the placement failed.
  - Testing:
    - Covered by integration tests submitting unschedulable PodGroups.

###### What steps should be taken if SLOs are not being met to determine the problem?

1. Analyze Latency Metrics: Check `scheduler_podgroup_scheduling_attempt_duration_seconds` and 
   `scheduler_podgroup_scheduling_algorithm_duration_seconds`. High values here indicate that the Toplogy Aware
   Schedling logic itself is computationally expensive and causing the regression.
3. Inspect Logs: Enable scheduler logging at `-v=6` (or `-v=10` for deep tracing) to trace the execution time of
   individual Workload Scheduling Cycles and identify if specific PodGroups which are blocking the queue. 
4. Disable Feature: If the regression is critical and impacting cluster health, disable the
   TopologyAwareWorkloadScheduling feature gate. This will revert the scheduler to the standard Workload Scheduling
   logic, restoring baseline performance (at the cost of losing topology semantics).

## Implementation History

- 2025-12: Initial KEP-5732 proposal.
- 2026-02: KEP-5732 created for TAS alpha release.
- 2026-02: KEP-5732 updated to sync with decoupling of PodGroup/Workload API.
- 2026-05: KEP updated to promote to beta in v1.37.

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

N/A
