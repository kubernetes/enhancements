# KEP-5832: Decouple PodGroup from Workload API

<!-- toc -->
- [Release Signoff Checklist](#release-signoff-checklist)
- [Summary](#summary)
- [Motivation](#motivation)
  - [Goals](#goals)
  - [Non-Goals](#non-goals)
- [Proposal](#proposal)
  - [User Stories (Optional)](#user-stories-optional)
    - [Story 1: Scaling a Training Job](#story-1-scaling-a-training-job)
    - [Story 2: DRA Claims per Replica](#story-2-dra-claims-per-replica)
  - [Notes/Constraints/Caveats (Optional)](#notesconstraintscaveats-optional)
  - [Risks and Mitigations](#risks-and-mitigations)
- [Design Details](#design-details)
  - [API](#api)
  - [Naming](#naming)
  - [Validation Policy](#validation-policy)
  - [Future Plans](#future-plans)
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
- [Infrastructure Needed (Optional)](#infrastructure-needed-optional)
<!-- /toc -->

## Release Signoff Checklist

Items marked with (R) are required *prior to targeting to a milestone / release*.

- [x] (R) Enhancement issue in release milestone, which links to KEP dir in [kubernetes/enhancements] (not the initial KEP PR)
- [ ] (R) KEP approvers have approved the KEP status as `implementable`
- [x] (R) Design details are appropriately documented
- [x] (R) Test plan is in place, giving consideration to SIG Architecture and SIG Testing input (including test refactors)
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

## Summary

This KEP proposes decoupling the PodGroup API from the Workload API by introducing PodGroup as a standalone runtime object. In the current design, PodGroups are embedded within the Workload spec, which creates challenges around immutability, scaling, and lifecycle management. Under this proposal, the following changes are proposed:

- `Workload` becomes a static policy definition that specifies what workload behavior should be applied
- `PodGroup` becomes a runtime object that is automatically created by true controllers (Job, JobSet, LeaderWorkerSet) when they create workloads, rather than being embedded within the Workload spec. It will be created based on the `podGroupTemplate` defined in the referenced Workload.

## Motivation

The current design embeds PodGroups within the Workload spec, creating several integration challenges:

- The immutability of `workload.spec.podGroups` forces controllers to delete/recreate `Workload` when scaling.
- Tight coupling: The lifecycle of individual PodGroups can be significantly shorter than that of the Workload as a whole. However, changes to one `PodGroup` (i.e., leader policy in LWS) require recreating the entire `Workload`.
- Extending the Workload object to store the runtime status for all podgroups would lead to significant scalability issues.
  - *Size Limit*: Large Workloads (i.e. large number of PodGroups) may easily hit the 1.5MB etcd object limit.
  - *Contention*: Updating the status of a single podGroup would require read-modify-write on the central massive Workload object. In addition, any status change triggers watches for all controllers observing the Workload.
- Resource claims (DRA) may belong to a specific subset of a workload (i.e., a specific replica in a LeaderWorkerSet). Currently, there is no distinct API object to attach these claims to, making garbage collection difficult.

By decoupling `PodGroup` as a standalone runtime object:

- `Workload` defines static scheduling policy
- `PodGroupTemplate` provides the blueprint for runtime PodGroup creation
- `PodGroup` becomes a mutable, controller-owned object with its own lifecycle

### Goals

- Decouple `PodGroup` lifecycle from `Workload` lifecycle
- Enhance status ownership by making `PodGroup` status tracks podGroup-level runtime state
- Simplify integration with `Workload` API and true controllers
- Ensure proper ownership of `PodGroup`

### Non-Goals

- Change pod creation responsibility
- Replace or modify true workload controllers
- Modify existing Workload API beyond decoupling
- Change current gang-scheduler plugin algorithm

## Proposal

> This KEP is heavily depends on [KEP-4671: Gang Scheduling using Workload Object](https://github.com/kubernetes/enhancements/tree/master/keps/sig-scheduling/4671-gang-scheduling). It is building on foundations and assumes the knowledge of the concepts introduced there.

Introduce a new `PodGroup` API that is a standalone runtime object. It will be created based on the `podGroupTemplate` defined in the referenced `Workload` API.

In v1.36 we introduce phase 1 which will focus of Static batch workloads as flat PodGroups. In phase 2 that will be in v1.37 we will introduce dynamic/advanced batch workloads that will support hierarchical PodGroups.

```yaml
apiVersion: scheduling.k8s.io/v1alpha1
kind: PodGroup
metadata:
  name: pd-1
  namespace: ns-1
spec:
  workloadRef:
    name: training-workload
  podGroupTemplateRef:
    name: pd-1-template
  replicaIndex: 1
  # TBD: Add resourceClaimTemplates
  # TBD: Add PodGroupSets
  # TBD: Add TopologyPolicy
status:
 phase: Scheduled
 scheduledPods: 2
  ...
 conditions:
  ...
```

### User Stories (Optional)

#### Story 1: Scaling a Training Job

As a user running distributed training jobs, I want to scale my job parallelism from 4 to 8 workers without recreating the Workload object.

#### Story 2: DRA Claims per Replica

As a GPU cluster admin, I want `ResourceClaims` for GPUs to be owned by specific PodGroups so that when a replica is deleted, its GPU claims are properly garbage collected.

### Notes/Constraints/Caveats (Optional)

### Risks and Mitigations

TBD

## Design Details

### API

The `PodGroup` type will be defined with the following structure:

```go
// API Group: scheduling.k8s.io/v1alpha1

// PodGroup represents a runtime instance of pods grouped for gang scheduling.
// PodGroups are created by workload controllers (Job, LWS, JobSet, etc...) from
// Workload.podGroupTemplates. Each PodGroup corresponds to one replica of the workload.
type PodGroup struct {
   metav1.TypeMeta
   
   // Standard object's metadata.
   // Name must be a DNS subdomain.
   //
   // +optional
   metav1.ObjectMeta

   // Spec defines the desired state of the PodGroup.
   // +required
   Spec PodGroupSpec

   // Status represents the current observed state of the PodGroup.
   // +optional
   Status PodGroupStatus
}

// PodGroupSpec defines the desired state of a PodGroup.
type PodGroupSpec struct {
   // WorkloadRef references the Workload that defines the policy.
   // This allows the scheduler to locate the static policy.
   // +required
   WorkloadRef *corev1.ObjectReference

   // PodGroupTemplateName references the PodGroupTemplate name that defines 
   // the template for this PodGroup.
   // +required
   PodGroupTemplateName string

   // ReplicaIndex identifies which replica of the workload this PodGroup represents.
   // Used for applying per-replica overrides and for identification.
   // +required
   ReplicaIndex int32

  // TBD: Add PriorityClass Name 
  // TBD: Add PodGroupSets
  // TBD: Add resourceClaimTemplates
  // TBD: Add TopologyPolicy
}

// PodGroupStatus represents the detailed observed state of a PodGroup.
type PodGroupStatus struct {
   // Phase represents the overall scheduling phase of this PodGroup.
   // +optional
   Phase PodGroupPhase

   // Conditions represent the latest observations of the PodGroup's state.
   // Known condition types: Scheduled, Running, GangSatisfied, SchedulingTimeout.
   // +optional
   Conditions []metav1.Condition

   // ScheduledPods is the count of pods that have been assigned to nodes.
   // +optional
   ScheduledPods int32
}

// PodGroupPhase represents the scheduling state of a PodGroup.
// +enum
type PodGroupPhase string

const (
   // PodGroupPending means the PodGroup is waiting to be scheduled.
   // The scheduler has not yet found sufficient resources for the gang.
   PodGroupPending PodGroupPhase = "Pending"

   // PodGroupScheduling means the scheduler is actively trying to place pods.
   // For gang scheduling, this means waiting for minCount pods to be schedulable.
   PodGroupScheduling PodGroupPhase = "Scheduling"

   // PodGroupScheduled means all required pods have been assigned to nodes.
   // The gang requirement (minCount) has been satisfied.
   PodGroupScheduled PodGroupPhase = "Scheduled"

   // PodGroupRunning means all pods in the PodGroup are in Running state.
   PodGroupRunning PodGroupPhase = "Running"

   // PodGroupFailed means the PodGroup failed to schedule within the timeout,
   // or too many pods have failed.
   PodGroupFailed PodGroupPhase = "Failed"
)
```

### Naming

TBD

### Validation Policy

To maintain cluster stability and prevent "deadlock" scenarios where a scaled-down Job cannot satisfy an old minCount, we could use `ValidatingAdmissionPolicies` (VAP).

This mechanism works effectively for both native resources (Jobs) and CRDs (MPIJobs). We'd use VAP to strictly forbid updates to parallelism/replicas for any resource labeled where Workload API is enabled (TBD how).

```yaml
apiVersion: admissionregistration.k8s.io/v1
kind: ValidatingAdmissionPolicy
metadata:
  name: block-workload-resizing
spec:
  failurePolicy: Fail
  matchConstraints:
    resourceRules:
    # Native Resources
    - apiGroups: ["batch", "apps"]
      apiVersions: ["v1"]
      operations: ["UPDATE"]
      resources: ["jobs", "statefulsets"]
    # Custom Resources (e.g., Kubeflow)
    - apiGroups: ["kubeflow.org"]
      apiVersions: ["v1"]
      operations: ["UPDATE"]
      resources: ["mpijobs", "pytorchjobs"]
  validations:
    - expression: >
        < TBD: Workload API not used > || 
        object.spec == oldObject.spec
      message: "Updates to spec (resizing) are not supported for Gang-Scheduled workloads in v1.36. Recreate the object or wait for v1.37 features."
```

### Future Plans

TBD

### Test Plan

[x] I/we understand the owners of the involved components may require updates to
existing tests to make this code solid enough prior to committing the changes necessary
to implement this enhancement.

##### Prerequisite testing updates

##### Unit tests

- k8s.io/kubernetes/pkg/apis/scheduling/v1alpha1: `2026-01-23` - `62.7%`
- k8s.io/kubernetes/pkg/apis/scheduling/validation: `2026-01-23` - `97.8%`
- k8s.io/kubernetes/pkg/scheduler: `2026-01-23` - `81.7%`

##### Integration tests

We will add integration tests for `PodGroup`  to ensure the basic functionalities of `PodGroup` including:

- Pods belonging to a `PodGroup` are scheduled together
- `PodGroup` status is updated correctly
- `PodGroup` is garbage collected when the replica is deleted
- Pods linked to the non-existing workload or podGroup are not scheduled
- Pods get unblocked when workload or podGroup is created and observed by scheduler
- Pods are not scheduled if there is no space for the whole PodGroup

##### e2e tests

We will add basic API tests for the the new `PodGroup` API.

### Graduation Criteria

#### Alpha

- `PodGroup` is introduced behind `GangScheduling-PodGroup` feature flag
- API tests for `PodGroup` API are added and passing
- kube-scheduler implements the `PodGroup` API

#### Beta

- Workload and PodGroup APIs are able to get integrated with true workload controllers
- e2e tests for `PodGroup` are added and passing

#### GA

- TBD in for Beta release

### Upgrade / Downgrade Strategy

> This KEP is completely additive and can safely fallback to the original behavior on downgrade.

### Version Skew Strategy

- For kubelets: The feature is limited to the control plane, so the version skew with nodes (kubelets) doesn't matter.
- For true workload controllers: Controllers running older versions continue to work with embedded PodGroups
- For kube-apiserver: For the new API, the old version of components in particular kube-apiserver may not handle those. Thus, users should not set those fields before confirming all control-plane instances were upgraded to the version supporting those.
- For kube-scheduler: This is purely kube-scheduler in-memory feature, so the skew doesn't really matter, since there is always only single kube-scheduler instance being a leader.

## Production Readiness Review Questionnaire

### Feature Enablement and Rollback

###### How can this feature be enabled / disabled in a live cluster?

- [x] Feature gate
  - Feature gate name: GangScheduling-PodGroup
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

No. PodGroup objects will only be triggered by the existence of Workload objects and those are not yet created automatically behind the scenes.

###### Can the feature be disabled once it has been enabled (i.e. can we roll back the enablement)?

Yes. The `GangScheduling-PodGroup` feature gate needs to be switched off to disable the feature.

###### What happens if we reenable the feature if it was previously rolled back?

The feature will start working again. However, there might be some Workload objects already stored in etcd and may affect the behavior of some of the existing workloads.

###### Are there any tests for feature enablement/disablement?

No.

### Rollout, Upgrade and Rollback Planning

###### How can a rollout or rollback fail? Can it impact already running workloads?

###### What specific metrics should inform a rollback?

###### Were upgrade and rollback tested? Was the upgrade->downgrade->upgrade path tested?

###### Is the rollout accompanied by any deprecations and/or removals of features, APIs, fields of API types, flags, etc.?

No.

### Monitoring Requirements

###### How can an operator determine if the feature is in use by workloads?

###### How can someone using this feature know that it is working for their instance?

- [ ] Events
  - Event Reason:
- [x] API .status
  - Condition name: `PodGroupScheduled`
  - Other field:
- [ ] Other (treat as last resort)
  - Details:

###### What are the reasonable SLOs (Service Level Objectives) for the enhancement?

###### What are the SLIs (Service Level Indicators) an operator can use to determine the health of the service?

- [ ] Metrics
  - Metric name:
  - [Optional] Aggregation method:
  - Components exposing the metric:
- [ ] Other (treat as last resort)
  - Details:

###### Are there any missing metrics that would be useful to have to improve observability of this feature?

### Dependencies

###### Does this feature depend on any specific services running in the cluster?

### Scalability

###### Will enabling / using this feature result in any new API calls?

Yes.

1. Watching for PodGroups:

- API call type: LIST+WATCH PodGroups
- estimated throughput: < XX/s
- originating component: kube-scheduler

2. Status updates:

- API call type: PUT/PATCH PodGroups
- estimated throughput: < XX/s
- originating component: kube-scheduler

###### Will enabling / using this feature result in introducing new API types?

Yes. New API type `PodGroup`

###### Will enabling / using this feature result in any new calls to the cloud provider?

No.

###### Will enabling / using this feature result in increasing size or count of the existing API objects?

No.

###### Will enabling / using this feature result in increasing time taken by any operations covered by existing SLIs/SLOs?

No. (Need to check if we should add Workload or Pod into the SLIs/SLOs time)

###### Will enabling / using this feature result in non-negligible increase of resource usage (CPU, RAM, disk, IO, ...) in any components?

The increase of CPU/MEM consumption of kube-apiserver and kube-scheduler should be negligible percentage of the current resource usage.

###### Can enabling / using this feature result in resource exhaustion of some node resources (PIDs, sockets, inodes, etc.)?

No.

### Troubleshooting

###### How does this feature react if the API server and/or etcd is unavailable?

###### What are other known failure modes?

###### What steps should be taken if SLOs are not being met to determine the problem?

## Implementation History

## Drawbacks

## Alternatives

## Infrastructure Needed (Optional)
