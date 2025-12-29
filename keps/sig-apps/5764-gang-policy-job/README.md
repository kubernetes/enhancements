# KEP-5764: Gang Policy for Job

<!-- toc -->
- [Release Signoff Checklist](#release-signoff-checklist)
- [Summary](#summary)
- [Motivation](#motivation)
  - [Goals](#goals)
  - [Non-Goals](#non-goals)
- [Proposal](#proposal)
  - [User Stories (Optional)](#user-stories-optional)
    - [Story 1](#story-1)
    - [Story 2](#story-2)
    - [Story 3](#story-3)
    - [Story 4](#story-4)
  - [Notes/Constraints/Caveats (Optional)](#notesconstraintscaveats-optional)
  - [Risks and Mitigations](#risks-and-mitigations)
- [Design Details](#design-details)
  - [API Changes](#api-changes)
  - [Feature Gate](#feature-gate)
  - [Behavior](#behavior)
  - [Workflow example](#workflow-example)
  - [Limitations](#limitations)
  - [Relation to future Workflow Aware Scheduling features](#relation-to-future-workflow-aware-scheduling-features)
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
  - [Annotations](#annotations)
  - [External Controller Only](#external-controller-only)
  - [Workload Template in Job Spec](#workload-template-in-job-spec)
- [Infrastructure Needed (Optional)](#infrastructure-needed-optional)
<!-- /toc -->

## Release Signoff Checklist

Items marked with (R) are required *prior to targeting to a milestone / release*.

- [x] (R) Enhancement issue in release milestone, which links to KEP dir in [kubernetes/enhancements] (not the initial KEP PR)
- [x] (R) KEP approvers have approved the KEP status as `implementable`
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

[kubernetes.io]: https://kubernetes.io/
[kubernetes/enhancements]: https://git.k8s.io/enhancements
[kubernetes/kubernetes]: https://git.k8s.io/kubernetes
[kubernetes/website]: https://git.k8s.io/website

## Summary

This proposal introduces gang scheduling support for Kubernetes Jobs. Gang scheduling
allows a group of pods (a "gang") to be scheduled together. This ensures that either
all pods in the job are scheduled, or none are, preventing partial scheduling which
can be problematic for certain workloads (e.g., distributed training, MPI jobs) that
require all workers to be available simultaneously.

## Motivation

Many batch workloads, such as machine learning training jobs (TensorFlow, PyTorch)
and MPI applications, require "all-or-nothing" scheduling. If only a subset of the
pods in a job are scheduled, the job may hang or fail, wasting resources. Currently,
users often rely on third-party schedulers (like Volcano or Kueue) to achieve this.
Native support for gang scheduling in the Job API simplifies the user experience and
integrates this capability directly into Kubernetes.

### Goals

- Allow users to specify a gang scheduling policy in the Job spec.
- Support "all-or-nothing" scheduling for Jobs.
- Integrate with the `scheduling.k8s.io` Workload API (via a feature gate).
- Provide a clear API for configuring gang scheduling behavior.

### Non-Goals

- Replacing advanced batch scheduling features provided by external projects
  (Volcano, Kueue) beyond basic gang scheduling.
- Supporting Elastic Indexed Jobs (due to Workload object immutability constraints).

## Proposal

We propose adding a `GangPolicy` field to the `JobSpec` that allows users to declare
their intent for gang scheduling. When this field is set, the Job Controller will
create a corresponding `Workload` object from the `scheduling.k8s.io` API group,
which can be consumed by gang-scheduling-aware schedulers.

### User Stories

#### Story 1

As a data scientist running distributed TensorFlow training jobs, I want all my
worker pods to be scheduled at the same time so that my training job doesn't hang
waiting for missing workers and waste GPU resources on partially scheduled pods.

#### Story 2

As an HPC administrator running MPI jobs on Kubernetes, I need all processes in my
MPI job to start together because MPI requires all ranks to be present for the
collective communication to work properly.

#### Story 3

As a platform engineer, I want to use native Kubernetes features for gang scheduling
without deploying additional third-party schedulers, reducing operational complexity
while still enabling my users to run batch workloads that require all-or-nothing
scheduling.

#### Story 4

As a platform engineer, I have a controller that composes Jobs but I want to
handle gang scheduling in my application differently.
I want to be able to disable gang scheduling in Job controller to avoid conflicts.

### Notes/Constraints/Caveats (Optional)

One option was to allow users to set WorkloadTemplates as a policy option
but the workload API is v1alpha1 and the Job API is v1.

It may be best to table this option until the workload API matures to avoid
API breaking changes in the V1 API of Job.

### Risks and Mitigations

**Risk**: Users may expect gang scheduling to work out of the box without a
gang-scheduling-aware scheduler.

**Mitigation**: Clear documentation stating that this feature requires a compatible
scheduler. The API field will have clear documentation and validation messages.

**Risk**: The `Workload` API from `scheduling.k8s.io` may evolve, causing
compatibility issues.

**Mitigation**: This feature is gated behind `JobGangPolicy` feature gate.
Changes to the Workload API will be tracked and the integration updated accordingly.

**Risk**: Additional API objects (Workload) increase etcd storage requirements.

**Mitigation**: Workload objects are lightweight and only created for Jobs that
explicitly opt into gang scheduling.

## Design Details

### API Changes

We propose adding a `GangPolicy` field to the `JobSpec`.

```go
// GangSchedulingPolicy specifies the gang scheduling mode for a Job.
// +enum
// +k8s:enum
type GangSchedulingPolicy string

const (
	// NoGang means that the Job does not use gang scheduling.
	NoGang GangSchedulingPolicy = "NoGang"
	// JobAsGang means that all pods in the Job are scheduled as a gang.
	JobAsGang GangSchedulingPolicy = "JobAsGang"
)

// GangPolicy defines the gang scheduling configuration for a Job.
type GangPolicy struct {
	// Policy specifies the gang scheduling mode.
	// +optional
	Policy GangSchedulingPolicy
}

type JobSpec struct {
    // ... existing fields ...

	// GangPolicy specifies the gang scheduling configuration for this Job.
	// When set, all pods in the Job are scheduled as a group according to the
	// specified policy.
	// This is only valid if JobGangPolicy feature gate is enabled.
	// +optional
	GangPolicy *GangPolicy
}
```

### Feature Gate

A new feature gate `JobGangPolicy` will be introduced to control the availability
of this feature. It will initially be Alpha.

This feature gate has dependencies on:
- `GangScheduling`: Enables gang scheduling support in the scheduler
- `GenericWorkload`: Enables the Workload API in `scheduling.k8s.io`

### Behavior

When `JobGangPolicy` is enabled and `GangPolicy` is set to `JobAsGang`:

1. The Job Controller will create a `Workload` object (from `scheduling.k8s.io/v1alpha1`)
   associated with the Job. The Workload will have an owner reference to the Job,
   ensuring automatic garbage collection when the Job is deleted.
2. The `Workload` will define a `PodGroup` using the name of job with a `MinCount` equal
   to the Job's `Parallelism`.
3. Pods created by the Job Controller will have a `WorkloadRef` in their PodSpec
   pointing to the created `Workload` (with `Name` set to the Job name and `PodGroup`
   set to the Job name).

### Workflow example

For the following Job:

```yaml
apiVersion: batch/v1
kind: Job
metadata:
  name: sleep-gang-job
spec:
  parallelism: 10
  completions: 10
  completionMode: Indexed
  gangPolicy: 
    policy: JobAsGang
  template:
    spec:
      containers:
      - name: sleep-container
        image: busybox:latest
        command: ["sleep", "10000"]
        resources:
          requests:
            cpu: 100m
            memory: 100Mi
          limits:
            cpu: 100m
            memory: 100Mi
      restartPolicy: Never
  backoffLimit: 4
```

When the gangPolicy.policy is equal to JobAsGang the following workload will be created:

```yaml
apiVersion: v1
items:
- apiVersion: scheduling.k8s.io/v1alpha1
  kind: Workload
  metadata:
    creationTimestamp: "2026-01-16T20:53:31Z"
    name: sleep-gang-job
    namespace: default
    ownerReferences:
    - apiVersion: batch/v1
      blockOwnerDeletion: true
      controller: true
      kind: Job
      name: sleep-gang-job
      uid: 85b9436d-14df-4a86-872f-266a6fe6109f
    resourceVersion: "343"
    uid: 7df77956-d5d8-4adb-ba54-f5c0619049c2
  spec:
    controllerRef:
      apiGroup: batch
      kind: Job
      name: sleep-gang-job
    podGroups:
    - name: sleep-gang-job
      policy:
        gang:
          minCount: 10
```

The pods that are created by this job would have the following values in their workload:

```yaml
  workloadRef:
    name: sleep-gang-job
    podGroup: sleep-gang-job
```

**Defaulting behavior**:
When the feature gate is enabled and `GangPolicy` is not specified, it defaults to
`NoGang` (no gang scheduling).

**Elastic Indexed Jobs**:
When a Job's `Parallelism` changes (for elastic indexed jobs), the Job Controller
will delete the existing Workload and recreate it with the updated `MinCount` to
match the new parallelism value.

<<Unresolved kannon92
Deleting workloads and recreating is not ideal. 
One option could be to forbid gang scheduling and elastic jobs for now.
One could reject the updates if gangPolicy is set to JobAsGang.
And elastic jobs is only supported for non gang scheduled jobs.
>>

**Feature gate disabled behavior**:
When the feature gate is disabled, the `GangPolicy` field is dropped from new Jobs.
For existing Jobs that already have `GangPolicy` set (created when the gate was
enabled), the field is preserved but no Workload objects are created and no
`WorkloadRef` is set on new pods.

### Limitations

The `Workload` object in `scheduling.k8s.io` is immutable. To handle elastic indexed
jobs where `Parallelism` changes, the Job Controller deletes and recreates the
`Workload` object with the updated `MinCount`. This approach has caveats:

- During workload recreation, there is a brief window where the Workload does not
  exist. Pods that were already created still have a `WorkloadRef` pointing to
  the workload name, but the workload object is temporarily absent. This means
  gang scheduling semantics will not work during this window.
- Future work may explore relaxing immutability requirements of the `Workload`
  object to allow in-place updates.

### Relation to future Workflow Aware Scheduling features

The workload aware scheduling umbrella work is full of interesting turns. To provide useful functionality early,
we can streamline gang scheduling so jobs can be scheduled as all or nothing.

Future work like Topology Aware Scheduling or DRA can be supported by potentially giving an option to create the WorkloadTemplate.
So the potential support could be a new enum policy where users can put in the WorkloadTemplate they want and the job controller would create that API.
Right now we assume that `MinCount` = parallelism and are not providing much flexibility.
The author believes this is useful as is as many HPC schedulers do not provide opt-out support for gang scheduling so it at least provides 
a critical feature for AI/ML workloads.

### Test Plan

[x] I/we understand the owners of the involved components may require updates to
existing tests to make this code solid enough prior to committing the changes necessary
to implement this enhancement.

##### Prerequisite testing updates

No prerequisite testing updates are required. The Job controller already has
comprehensive test coverage.

##### Unit tests

- `pkg/apis/batch/validation`: validation logic for `GangPolicy` field
- `pkg/controller/job`: Job controller logic for Workload creation and management
- `pkg/registry/batch/job`: API defaulting and validation

Coverage targets for alpha:
- `pkg/apis/batch/validation`: 2025-01-02 - 85%
- `pkg/controller/job`: 2025-01-02 - 80%

Specific test cases implemented:

**Defaulting tests** (`pkg/apis/batch/v1/defaults_test.go`):
- `TestSetDefaultJob_GangScheduling`:
  - GangPolicy unspecified with feature gate enabled -> defaults to NoGang
  - GangPolicy unspecified with feature gate disabled -> GangPolicy remains nil
  - GangPolicy explicitly set to JobAsGang with feature gate enabled -> no change
  - GangPolicy explicitly set to NoGang with feature gate enabled -> no change

**Validation tests** (`pkg/apis/batch/validation/validation_test.go`):
- Valid gang policy with NoGang
- Valid gang policy with JobAsGang
- Invalid: GangPolicy with empty policy string (Required error)
- Invalid: GangPolicy with unsupported policy value (NotSupported error)

**Immutability tests** (`pkg/apis/batch/validation/validation_test.go`):
- Add gang policy to existing job without gang policy (Invalid error)
- Update gang policy from NoGang to JobAsGang (Invalid error)
- Remove gang policy from job with gang policy (Invalid error)

**Job Controller tests** (`pkg/controller/job/job_controller_test.go`):
- `TestEnsureWorkloadForJob`:
  - Creates workload for JobAsGang policy when feature gate enabled
  - Does not create workload for JobAsGang policy when feature gate disabled
  - Does not create workload for job without gang policy
  - Does not create workload for NoGang policy
  - Verifies workload spec (name, PodGroups, minCount, owner references)
  - Verifies idempotency (calling again should not error)
- `TestEnsureWorkloadForJobRecreatesOnParallelismChange`:
  - Recreates workload when parallelism increases (for elastic indexed jobs)
  - Recreates workload when parallelism decreases
  - Does not recreate workload when parallelism unchanged
- `TestWorkloadRefSetOnPodTemplates`:
  - WorkloadRef set on PodTemplates for JobAsGang policy with feature gate enabled
  - WorkloadRef not set for JobAsGang policy when feature gate disabled
  - WorkloadRef not set for NoGang policy
  - WorkloadRef not set when no gang policy
  - Verifies WorkloadRef.Name and WorkloadRef.PodGroup values

##### Integration tests

Integration tests will cover:

- Workload creation when `JobAsGang` is set
- WorkloadRef is correctly set on Pods created by the Job
- Behavior when feature gate is disabled (field is preserved but not acted upon)
- Workload deletion when Job is deleted (via owner reference garbage collection)
- Job status updates based on Workload status

##### e2e tests

For Beta and GA:
- e2e tests with a gang-scheduling-aware scheduler to verify end-to-end behavior
- Tests for failure scenarios (scheduler unavailable, etc.)

### Graduation Criteria

#### Alpha

- Feature implemented behind the `JobGangPolicy` feature flag
- Initial unit and integration tests completed and enabled
- API fields added with appropriate validation
- Documentation for the feature

#### Beta

- Gather feedback from developers and early adopters
- Integration tests are stable in Testgrid
- Address any issues discovered during alpha
- Metrics for observability are implemented
- e2e tests with gang-scheduling scheduler

#### GA

- Multiple examples of real-world usage
- Allowing time for feedback (at least 2 releases in beta)
- All known issues resolved
- Conformance tests if applicable

### Upgrade / Downgrade Strategy

**Upgrade**: When upgrading to a version with `JobGangPolicy` enabled, existing Jobs
are unaffected. New Jobs can start using the `GangPolicy` field.

**Downgrade**: When downgrading to a version without `JobGangPolicy`:
- The `GangPolicy` field will be preserved in existing Job objects (standard API
  field preservation behavior)
- The Job Controller will not create new Workload objects
- Existing Workload objects will remain but won't be managed
- Pods will be scheduled normally without gang scheduling semantics

No special migration steps are required.

### Version Skew Strategy

This feature involves the kube-apiserver and kube-controller-manager:

- **kube-apiserver**: Validates and stores the `GangPolicy` field
- **kube-controller-manager**: Creates Workload objects and sets WorkloadRef on Pods

During version skew (n-1 controller-manager with n apiserver):
- The API server will accept Jobs with `GangPolicy` set
- The older controller-manager will ignore the field and not create Workload objects
- Jobs will function normally but without gang scheduling

This is acceptable behavior during upgrade windows. The feature is fully functional
only when both components are upgraded.

The kubelet is not involved in this feature, so node version skew is not a concern.

## Production Readiness Review Questionnaire

### Feature Enablement and Rollback

###### How can this feature be enabled / disabled in a live cluster?

- [x] Feature gate (also fill in values in `kep.yaml`)
  - Feature gate name: `JobGangPolicy`
  - Components depending on the feature gate:
    - kube-apiserver
    - kube-controller-manager

###### Does enabling the feature change any default behavior?

No. The feature is opt-in. Jobs without `GangPolicy` set behave exactly as before.

###### Can the feature be disabled once it has been enabled (i.e. can we roll back the enablement)?

Yes. Disabling the feature gate will:
- Prevent new Jobs from having `GangPolicy` validated (field is preserved but ignored)
- Stop the Job Controller from creating new Workload objects
- Existing Jobs with `GangPolicy` will continue to run, but new pods will not have
  WorkloadRef set

Existing workloads are not disrupted; they simply lose gang scheduling semantics.

###### What happens if we reenable the feature if it was previously rolled back?

The Job Controller will resume creating Workload objects for Jobs with `GangPolicy`
set. New pods for those Jobs will have WorkloadRef set. There is no data loss or
corruption.

###### Are there any tests for feature enablement/disablement?

Yes. Integration tests will cover:
- Creating a Job with `GangPolicy` when feature is enabled
- Behavior when feature is disabled (field preserved, no Workload created)
- Re-enabling the feature and verifying Workload creation resumes

### Rollout, Upgrade and Rollback Planning

###### How can a rollout or rollback fail? Can it impact already running workloads?

A rollout could fail if:
- The Workload CRD from `scheduling.k8s.io` is not installed in the cluster
- There are issues with the Job Controller creating Workload objects

Impact on running workloads:
- Existing Jobs without `GangPolicy` are completely unaffected
- Existing Jobs with `GangPolicy` may not have Workloads created if the controller
  fails, but pods will still be scheduled (just without gang semantics)

###### What specific metrics should inform a rollback?

- Increase in `job_sync_duration_seconds` indicating controller slowdowns
- Errors in Job Controller logs related to Workload creation
- Jobs with `GangPolicy` stuck in pending state

###### Were upgrade and rollback tested? Was the upgrade->downgrade->upgrade path tested?

This will be tested manually before beta graduation. The test will verify:
- Jobs with `GangPolicy` continue functioning after downgrade (without gang semantics)
- Re-upgrade restores gang scheduling functionality

###### Is the rollout accompanied by any deprecations and/or removals of features, APIs, fields of API types, flags, etc.?

No.

### Monitoring Requirements

###### How can an operator determine if the feature is in use by workloads?

- Check for Jobs with `.spec.gangPolicy` set:
  `kubectl get jobs -A -o jsonpath='{.items[?(@.spec.gangPolicy)].metadata.name}'`
- Check for Workload objects created by the Job controller:
  `kubectl get workloads -A`

###### How can someone using this feature know that it is working for their instance?

- [x] API .status
  - Other field: Check that the Workload object exists for the Job and has the
    expected PodGroup configuration
- [x] Other
  - Details: Verify pods have the WorkloadRef annotation pointing to the correct
    Workload object

###### What are the reasonable SLOs (Service Level Objectives) for the enhancement?

- 99% of Jobs with `GangPolicy` should have their Workload object created within
  1 second of Job creation
- Job sync latency should not increase by more than 10% for Jobs with `GangPolicy`

###### What are the SLIs (Service Level Indicators) an operator can use to determine the health of the service?

- [x] Metrics
  - Metric name: `job_sync_duration_seconds`
  - Aggregation method: histogram percentiles (p50, p99)
  - Components exposing the metric: kube-controller-manager
- [x] Metrics
  - Metric name: `job_syncs_total`
  - Aggregation method: counter by result (success/error)
  - Components exposing the metric: kube-controller-manager

###### Are there any missing metrics that would be useful to have to improve observability of this feature?

A new metric `job_gang_workload_creation_total` could be added to track Workload
creation specifically for gang-scheduled jobs. This will be considered for beta.

### Dependencies

###### Does this feature depend on any specific services running in the cluster?

- Workload CRD from `scheduling.k8s.io`
  - Usage description: The Job Controller creates Workload objects for gang-scheduled
    Jobs
  - Impact of its outage on the feature: Jobs with `GangPolicy` will not have
    Workload objects created, and gang scheduling will not work
  - Impact of its degraded performance or high-error rates on the feature: Workload
    creation may be delayed, affecting gang scheduling timeliness

- Gang-scheduling-aware scheduler (optional but required for actual gang scheduling)
  - Usage description: Consumes Workload objects to enforce gang scheduling semantics
  - Impact of its outage on the feature: Gang scheduling semantics won't be enforced;
    pods may be scheduled individually
  - Impact of its degraded performance or high-error rates on the feature: Delayed
    or partial gang scheduling

### Scalability

###### Will enabling / using this feature result in any new API calls?

Yes:
- CREATE Workload: One per Job with `GangPolicy` set, originating from
  kube-controller-manager
- GET/UPDATE Workload: During Job reconciliation, originating from
  kube-controller-manager
- Estimated throughput: Same as Job creation rate for Jobs using this feature

###### Will enabling / using this feature result in introducing new API types?

No. This feature uses the existing Workload type from `scheduling.k8s.io`.

###### Will enabling / using this feature result in any new calls to the cloud provider?

No.

###### Will enabling / using this feature result in increasing size or count of the existing API objects?

- API type(s): Job
- Estimated increase in size: ~50 bytes for the `GangPolicy` field
- Estimated amount of new objects: One Workload object per Job with `GangPolicy` set

###### Will enabling / using this feature result in increasing time taken by any operations covered by existing SLIs/SLOs?

Job creation/sync may take slightly longer due to Workload creation, but this should
be negligible (< 10ms additional latency).

###### Will enabling / using this feature result in non-negligible increase of resource usage (CPU, RAM, disk, IO, ...) in any components?

Minimal increase:
- kube-controller-manager: Slight increase in memory for watching Workload objects
- etcd: Storage for Workload objects (small, ~1KB per object)

###### Can enabling / using this feature result in resource exhaustion of some node resources (PIDs, sockets, inodes, etc.)?

No. This feature operates entirely in the control plane and does not affect node
resources.

### Troubleshooting

###### How does this feature react if the API server and/or etcd is unavailable?

The Job Controller cannot create or update Workload objects. Jobs with `GangPolicy`
will not have their Workloads created until the API server is available again.
This is consistent with how other controllers behave during API server unavailability.

###### What are other known failure modes?

- Workload CRD not installed
  - Detection: Errors in kube-controller-manager logs about missing Workload resource
  - Mitigations: Install the `scheduling.k8s.io` CRDs
  - Diagnostics: Check controller-manager logs for "no matches for kind Workload"
  - Testing: Integration test verifying graceful handling of missing CRD

- Workload creation fails due to validation errors
  - Detection: Job events showing Workload creation failure
  - Mitigations: Check Workload spec requirements and Job configuration
  - Diagnostics: Events on the Job object, controller-manager logs
  - Testing: Unit tests for various validation scenarios

###### What steps should be taken if SLOs are not being met to determine the problem?

1. Check kube-controller-manager logs for errors related to Job or Workload handling
2. Verify the Workload CRD is installed and accessible
3. Check `job_sync_duration_seconds` metric for increased latency
4. Verify no resource constraints on the controller-manager
5. Check etcd health and latency

## Implementation History

- 2025-12-29: Initial KEP draft created

## Drawbacks

- Adds complexity to the Job API and controller
- Requires users to understand the relationship between Jobs and Workloads
- Gang scheduling semantics depend on an external scheduler; native Kubernetes
  scheduler does not implement gang scheduling
- Additional API objects (Workloads) increase storage requirements

## Alternatives

### Annotations

Use annotations to trigger gang scheduling instead of a proper API field.

Rejected because:
- Less explicit and harder to validate than a proper API field
- Annotations are not versioned and lack schema validation
- Difficult to document and discover

### External Controller Only

Rely entirely on external controllers (like Kueue or JobSet) to manage Workloads for Jobs.

Rejected because:
- Doesn't provide a native "Job as Gang" UX where the user simply declares the intent
  on the Job resource
- Requires users to create additional resources (Workloads) manually or understand
  external controller semantics
- The solution recommended for by this KEP provides a cleaner integration point that external controllers can still
  leverage

###  Workload Template in Job Spec

Add gang scheduling configuration directly to Job specs

Rejected because:
- Workload API is v1alpha 1 so this would break the V1 guarantee of the Job API

This can be revisted as the workload API matures.

## Infrastructure Needed (Optional)

None.
