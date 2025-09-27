# KEP-5547: Expose workloadRef in the Job API for scheduler coordination

<!-- toc -->
- [Release Signoff Checklist](#release-signoff-checklist)
- [Summary](#summary)
- [Motivation](#motivation)
  - [Goals](#goals)
  - [Non-Goals](#non-goals)
- [Proposal](#proposal)
  - [User Stories (Optional)](#user-stories-optional)
    - [Story 1: Coordinated Gang Scheduling for ML Training Jobs](#story-1-coordinated-gang-scheduling-for-ml-training-jobs)
    - [Story 2: Prevent Race Conditions Between Job Controller and Scheduler](#story-2-prevent-race-conditions-between-job-controller-and-scheduler)
  - [Notes/Constraints/Caveats (Optional)](#notesconstraintscaveats-optional)
  - [Risks and Mitigations](#risks-and-mitigations)
    - [Misconfiguration or Invalid References](#misconfiguration-or-invalid-references)
    - [API Coupling and Evolution Risk](#api-coupling-and-evolution-risk)
- [Design Details](#design-details)
  - [Test Plan](#test-plan)
      - [Prerequisite testing updates](#prerequisite-testing-updates)
      - [Unit tests](#unit-tests)
      - [Integration tests](#integration-tests)
      - [e2e tests](#e2e-tests)
  - [Graduation Criteria](#graduation-criteria)
    - [Alpha](#alpha)
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

## Summary

Introduce a new optional field in the Job API spec to explicitly associate a Job with a Workload object, enabling safe coordination between workload-aware (Gang) scheduling and job controllers without introducing race conditions or forcing the scheduler to perform controller duties.

## Motivation

Workload-aware and gang scheduling logic rely on treating a group of pods as a single schedulable unit, which require the scheduler to operate with full knowledge of how Pods relate to higher-level workloads. While Job currently creates Pods directly, the linkage to any Workload concept is implicit and subject to race conditions during controller and scheduler interactions.

Without an explicit `workloadRef`, schedulers must guess which Job created a given Pod, causing unsafe scheduling or requiring speculative heuristics. This KEP makes the workload-pod relation first-class by allowing Jobs to opt-in to associating with a Workload object directly.

### Goals

- Introduce a new optional `workloadRef` field in the `JobSpec`, allowing a Job to declare an explicit association with a higher-level workload object.
- Keep the Job API backward-compatible and aligned with SIG Apps ownership, without altering existing Job behavior or introducing mandatory new semantics.

### Non-Goals

- Not replacing `PodSet` or `minAvailable` directly, rather enabling cleaner linkage.
- Not enforcing mutual exclusivity (i.e. Job may be used with or without a `workloadRef`).

## Proposal

Add a new optional field to JobSpec:

```go
type JobSpec struct {
    ...
    // WorkloadRef allows this job to declare an association to a Workload object.
    // The scheduler may use this to coordinate gang placement or workload-level decisions.
    // This field is optional and has no effect on job execution semantics.
    WorkloadRef *corev1.ObjectReference `json:"workloadRef,omitempty"`
}
```

### User Stories (Optional)

#### Story 1: Coordinated Gang Scheduling for ML Training Jobs

**Context**: As a platform operator running ML training pipelines composed of multiple Jobs, I want to associate each Job with a shared Workload object that specifies gang scheduling constraints (e.g., minAvailable), So that the scheduler can treat the set of pods across Jobs as a single schedulable unit and either co-schedule them or delay all together. Without having to track down the workload topology based on labels or timing.

**Solution**: Ensures all pods for a distributed training job land together or wait together, avoiding partial runs and wasted GPU allocation.

#### Story 2: Prevent Race Conditions Between Job Controller and Scheduler

**Context**: As a scheduler maintainer, I want the Job object to explicitly declare which workload it belongs to via a structured `workloadRef`, So that I can fetch the workload metadata during scheduling without relying on label selectors or waiting for controller propagation, And avoid risky correlation logic or inconsistent state across Job creation and pod scheduling.

**Solution**: Enables deterministic, controller-agnostic workload coordination and makes it safe to enforce workload-level policies during scheduling.

### Notes/Constraints/Caveats (Optional)

### Risks and Mitigations

#### Misconfiguration or Invalid References

**Risk Description**: Users or controllers may set an invalid or non-existent `workloadRef`, pointing to a workload that doesn’t exist, is in the wrong namespace, or isn’t intended to be compatible with the scheduler logic.

**Mitigation Strategies**:

- Controllers and admission webhooks validate the presence and correctness of the referenced object.
- The scheduler should fail gracefully if the `workloadRef` cannot be resolved or is incompatible.
- The field is optional, which means the default behavior is preserved when unset.

#### API Coupling and Evolution Risk

**Risk Description**: If the workload API evolves (i.e. API group changes), older Jobs with workloadRef might break or behave unexpectedly.

**Mitigation Strategies**:

- The use of a structured `ObjectReference` (vs. the workload name string) allows future evolution of the workload object’s type/version.
- The scheduler should resolve and type-check the object at runtime, enforcing known versions/kinds before attempting coordination.
- API evolution policies apply to the Workload resource itself.

## Design Details

### Test Plan

[x] I/we understand the owners of the involved components may require updates to
existing tests to make this code solid enough prior to committing the changes necessary
to implement this enhancement.

##### Prerequisite testing updates

##### Unit tests

- `<package>`: `<date>` - `<test coverage>`

##### Integration tests

- [test name](https://github.com/kubernetes/kubernetes/blob/2334b8469e1983c525c0c6382125710093a25883/test/integration/...): [integration master](https://testgrid.k8s.io/sig-release-master-blocking#integration-master?include-filter-by-regex=MyCoolFeature), [triage search](https://storage.googleapis.com/k8s-triage/index.html?test=MyCoolFeature)

##### e2e tests

- [test name](https://github.com/kubernetes/kubernetes/blob/2334b8469e1983c525c0c6382125710093a25883/test/e2e/...): [SIG ...](https://testgrid.k8s.io/sig-...?include-filter-by-regex=MyCoolFeature), [triage search](https://storage.googleapis.com/k8s-triage/index.html?test=MyCoolFeature)

### Graduation Criteria

#### Alpha

- Field added to `JobSpec`.
- Job controller populates it via Same Gang Scheduler FeatureGate.
- Scheduler validates and uses it safely.

### Upgrade / Downgrade Strategy

### Version Skew Strategy

## Production Readiness Review Questionnaire

### Feature Enablement and Rollback

###### How can this feature be enabled / disabled in a live cluster?

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

###### Can the feature be disabled once it has been enabled (i.e. can we roll back the enablement)?

###### What happens if we reenable the feature if it was previously rolled back?

###### Are there any tests for feature enablement/disablement?

### Rollout, Upgrade and Rollback Planning

###### How can a rollout or rollback fail? Can it impact already running workloads?

###### What specific metrics should inform a rollback?

###### Were upgrade and rollback tested? Was the upgrade->downgrade->upgrade path tested?

###### Is the rollout accompanied by any deprecations and/or removals of features, APIs, fields of API types, flags, etc.?

### Monitoring Requirements

###### How can an operator determine if the feature is in use by workloads?

###### How can someone using this feature know that it is working for their instance?

- [ ] Events
  - Event Reason:
- [ ] API .status
  - Condition name:
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

###### Will enabling / using this feature result in introducing new API types?

###### Will enabling / using this feature result in any new calls to the cloud provider?

###### Will enabling / using this feature result in increasing size or count of the existing API objects?

###### Will enabling / using this feature result in increasing time taken by any operations covered by existing SLIs/SLOs?

###### Will enabling / using this feature result in non-negligible increase of resource usage (CPU, RAM, disk, IO, ...) in any components?

###### Can enabling / using this feature result in resource exhaustion of some node resources (PIDs, sockets, inodes, etc.)?

### Troubleshooting

###### How does this feature react if the API server and/or etcd is unavailable?

###### What are other known failure modes?

###### What steps should be taken if SLOs are not being met to determine the problem?

## Implementation History

## Drawbacks

## Alternatives

## Infrastructure Needed (Optional)

NA
