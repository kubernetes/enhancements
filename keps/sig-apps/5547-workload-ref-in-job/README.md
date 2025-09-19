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
- [x] (R) KEP approvers have approved the KEP status as `implementable`
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

Workload-aware and gang scheduling[^1] logic rely on treating a group of pods as a single schedulable unit, which require the scheduler to operate with full knowledge of how Pods relate to higher-level workloads. While Job currently creates Pods directly, the linkage to any Workload concept is implicit and subject to race conditions during controller and scheduler interactions.

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

**Context**: As a platform operator running ML training pipelines composed of multiple Jobs, I want to associate each Job with a Workload object that specifies gang scheduling constraints (e.g., minAvailable), So that the scheduler can treat the set of pods across Jobs as a single schedulable unit and either co-schedule them or delay all together. Without having to track down the workload topology based on labels or timing.

**Example Configuration:**
```yaml
apiVersion: batch/v1
kind: Job
metadata:
  name: job-1
spec:
 ...
  template:
    spec:
      workloadRef:
        apiVersion: scheduling/v1alpha1
        name: w-job-1
        namespace: demo-workload
      containers:
        - name: job-container
          image: job-image
          command: ["./sample"]
          ...
```

#### Story 2: Prevent Race Conditions Between Job Controller and Scheduler

**Context**: As a scheduler maintainer, I want the Job object to explicitly declare which workload it belongs to via a structured `workloadRef`, So that I can fetch the workload metadata during scheduling without relying on label selectors or waiting for controller propagation, And avoid risky correlation logic or inconsistent state across Job creation and pod scheduling.

**Example Configuration:**

```yaml
apiVersion: scheduling/v1alpha1  
kind: Workload
metadata:
  name: w-job-2
  namespace: demo-workload
spec:
  controllerRef:
    name: job-2
    kind: Job
    apiGroup: batch
    ...
---
apiVersion: batch/v1
kind: Job
metadata:
  name: job-2
spec:
 ...
  template:
    spec:
      workloadRef:
        apiVersion: scheduling/v1alpha1
        name: w-job-2
        namespace: demo-workload
      containers:
        - name: job-container
          image: job-image
          command: ["./sample"]
          ...
```
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

- `pkg/controller/job/job_controller_test.go`: `09-30-2025` - `89.1%`
- `pkg/registry/batch/job/strategy_test.go`: `09-30-2025` - `93%`
- `pkg/apis/batch/validation/validation_test.go`: `09-30-2025` - `86.3%`

##### Integration tests

Update the following test cases to include the option of workloadRef field:

- [TestIndexedJob](https://github.com/kubernetes/kubernetes/blob/master/test/integration/job/job_test.go#L2905): [result]()
- [TestImmediateJobRecreation](https://github.com/kubernetes/kubernetes/blob/master/test/integration/job/job_test.go#L2291) : [result]()
- [TestParallelJob](https://github.com/kubernetes/kubernetes/blob/master/test/integration/job/job_test.go#L2671) : [result]()
- [BenchmarkLargeIndexedJob](https://github.com/kubernetes/kubernetes/blob/master/test/integration/job/job_test.go#L3508) : [result]()
- [TestJobFailedWithInterrupts](https://github.com/kubernetes/kubernetes/blob/master/test/integration/job/job_test.go#L3908): [result]()


##### e2e tests

Add the following new e2e test case:

- [should run a job to completion when workload reference is added](https://github.com/kubernetes/kubernetes/blob/master/test/e2e/apps/job.go): [result]()

### Graduation Criteria

#### Alpha

- Job controller populates new field via Same Gang Scheduler FeatureGate.
- Unit and integration tests passed as designed in [TestPlan](#test-plan).

#### Beta

- E2E tests passed as designed in [TestPlan](#test-plan).
- Feature is enabled by default.
- Address all issues reported by users.

#### GA

- No negative feedback.

### Upgrade / Downgrade Strategy

- Upgrade
  - If the feature gate is enabled, `Job` workloads are allowed to use only.
  - Even if the feature gate is enabled, but the `Job` workloads don't have the `workloadRef` field, the Job controller should function normally.
- Downgrade
  - Previously configured values will be ignored, and the job will be running normally but pods won't scheduled all at once, instead pod by pod.

### Version Skew Strategy

## Production Readiness Review Questionnaire

### Feature Enablement and Rollback

###### How can this feature be enabled / disabled in a live cluster?

- [x] Feature gate (also fill in values in `kep.yaml`)
  - Feature gate name: Workload/GenericWorkload/NativeWorkload
  - Components depending on the feature gate:
    - kube-apiserver
    - kube-scheduler
  - Feature gate name: GangScheduling
  - Components depending on the feature gate:
    - kube-scheduler
- [ ] Other
  - Describe the mechanism:
  - Will enabling / disabling the feature require downtime of the control
    plane?
  - Will enabling / disabling the feature require downtime or reprovisioning
    of a node?

###### Does enabling the feature change any default behavior?

No. Gang scheduling is triggerred purely via existence of Workload objects and
those are not yet created automatically behind the scenes.

###### Can the feature be disabled once it has been enabled (i.e. can we roll back the enablement)?
Yes. However, the content of `spec.workloadRef` field in Job objects will not be cleared, as well as the existing Workload objects will not be deleted.

###### What happens if we reenable the feature if it was previously rolled back?
The feature should start working again.
However, the user need to remember that some Workload references could already be stored in etcd and may affect the behavior of some of the existing jobs.

###### Are there any tests for feature enablement/disablement?
Yes. [Integration tests](#integration-tests) will cover the feature enablement.

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
No.

### Monitoring Requirements

<!--
This section must be completed when targeting beta to a release.
For GA, this section is required: approvers should be able to confirm the
previous answers based on experience in the field.
-->

###### How can an operator determine if the feature is in use by workloads?

###### How can someone using this feature know that it is working for their instance?

- [x] Events
  - Event Reason:
- [ ] API .status
  - Condition name:
  - Other field:
- [ ] Other (treat as last resort)
  - Details:

###### What are the reasonable SLOs (Service Level Objectives) for the enhancement?

###### What are the SLIs (Service Level Indicators) an operator can use to determine the health of the service?

- [x] Metrics
  - Metric name: `job_sync_duration_seconds` (existing): can be used to see how much the feature enablement increases the time spent in the sync job.
  - [Optional] Aggregation method:
  - Components exposing the metric:
- [ ] Other (treat as last resort)
  - Details:

###### Are there any missing metrics that would be useful to have to improve observability of this feature?

### Dependencies

###### Does this feature depend on any specific services running in the cluster?
Yes, the [Gang scheduler](https://github.com/kubernetes/enhancements/pull/5558) feature.
### Scalability

###### Will enabling / using this feature result in any new API calls?

Yes. 
- List/Get Workload objects.
- Put/Patch status updates (potentially not in Alpha)

###### Will enabling / using this feature result in introducing new API types?

Yes. New field in Job API.

###### Will enabling / using this feature result in any new calls to the cloud provider?

No.

###### Will enabling / using this feature result in increasing size or count of the existing API objects?

Yes, it will increase the size of existing API objects only when the `.spec.workloadRef` is set.

- API type(s): Job
- Estimated increase in size: ~100-400 bytes per Job when workloadRef is set, depending on the length of names and which optional fields are populated.

###### Will enabling / using this feature result in increasing time taken by any operations covered by existing SLIs/SLOs?

Job startup SLI/SLO may be affected and should be adjusted appropriately. The reason is that job controller needs to get the workload object reference now.

###### Will enabling / using this feature result in non-negligible increase of resource usage (CPU, RAM, disk, IO, ...) in any components?
The increase of CPU/MEM consumption of kube-apiserver and kube-scheduler should be negligible percentage of the current resource usage.

###### Can enabling / using this feature result in resource exhaustion of some node resources (PIDs, sockets, inodes, etc.)?
No.

### Troubleshooting

###### How does this feature react if the API server and/or etcd is unavailable?

###### What are other known failure modes?

###### What steps should be taken if SLOs are not being met to determine the problem?

## Implementation History

- 2025.09.30: KEP is ready for review

## Drawbacks

## Alternatives
The longer version of this design describing the whole thought process of choosing the proposed approach can be found in the [API design/proposal] (https://docs.google.com/document/d/1ulO5eUnAsBWzqJdk_o5L-qdq5DIVwGcE7gWzCQ80SCM/edit?usp=sharing) document.


## Infrastructure Needed (Optional)

NA

[^1]: The Kubernetes community uses the term "gang scheduling" to mean "all-or-nothing scheduling of a set of pods" [1,2,3,4,5,6,7,8,9,10,11,12,13]. In the Kubernetes context, it does not imply time-multiplexing (in contrast to prior academic work such as [Feitelson and Rudolph](https://doi.org/10.1016/0743-7315(92)90014-E), and in contrast to [Slurm Gang Scheduling](https://slurm.schedmd.com/gang_scheduling.html)).  
