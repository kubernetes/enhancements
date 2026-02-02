# KEP-5547: Integrate Workload APIs with Job Controller

<!-- toc -->
- [Release Signoff Checklist](#release-signoff-checklist)
- [Summary](#summary)
- [Motivation](#motivation)
  - [Goals](#goals)
  - [Non-Goals](#non-goals)
- [Proposal](#proposal)
  - [User Stories (Optional)](#user-stories-optional)
    - [ML Training Job with Gang Scheduling](#ml-training-job-with-gang-scheduling)
    - [Standard Batch Job with Workload Tracking](#standard-batch-job-with-workload-tracking)
  - [Notes/Constraints/Caveats (Optional)](#notesconstraintscaveats-optional)
    - [Alpha Constraints](#alpha-constraints)
  - [Risks and Mitigations](#risks-and-mitigations)
- [Design Details](#design-details)
  - [Job Controller Changes](#job-controller-changes)
    - [Object Creation Order](#object-creation-order)
  - [Admission Validation for Parallelism Changes](#admission-validation-for-parallelism-changes)
  - [Test Plan](#test-plan)
      - [Prerequisite testing updates](#prerequisite-testing-updates)
      - [Unit tests](#unit-tests)
      - [Integration tests](#integration-tests)
      - [e2e tests](#e2e-tests)
  - [Graduation Criteria](#graduation-criteria)
    - [Alpha (v1.36)](#alpha-v136)
    - [Beta](#beta)
    - [GA](#ga)
    - [Deprecation](#deprecation)
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
- [x] (R) Design details are appropriately documented
- [x] (R) Test plan is in place, giving consideration to SIG Architecture and SIG Testing input (including test refactors)
  - [ ] e2e Tests for all Beta API Operations (endpoints)
  - [ ] (R) Ensure GA e2e tests meet requirements for [Conformance Tests](https://github.com/kubernetes/community/blob/master/contributors/devel/sig-architecture/conformance-tests.md)
  - [ ] (R) Minimum Two Week Window for GA e2e tests to prove flake free
- [ ] (R) Graduation criteria is in place
  - [ ] (R) [all GA Endpoints](https://github.com/kubernetes/community/pull/1806) must be hit by [Conformance Tests](https://github.com/kubernetes/community/blob/master/contributors/devel/sig-architecture/conformance-tests.md) within one minor version of promotion to GA
- [ ] (R) Production readiness review completed
- [ ] (R) Production readiness review approved
- [x] "Implementation History" section is up-to-date for milestone
- [ ] User-facing documentation has been created in [kubernetes/website], for publication to [kubernetes.io]
- [ ] Supporting documentation—e.g., additional design documents, links to mailing list discussions/SIG meetings, relevant PRs/issues, release notes

## Summary

This KEP introduces native integration between the Job controller and the gang scheduling[^1] APIs ([Workload](https://github.com/kubernetes/kubernetes/blob/release-1.35/pkg/apis/scheduling/types.go#L97) and [PodGroup](https://github.com/kubernetes/enhancements/pull/5833)).

The Job controller will automatically create `Workload` and `PodGroup` objects before creating pods for parallel Jobs, enabling native gang scheduling support in Kubernetes. 

## Motivation

The Kubernetes Job Controller currently creates pods independently without workload-aware scheduling constraints. This creates challenges for parallel applications (i.e., AI/ML training workloads, MPI jobs) that require all pods to be scheduled and run together or none(gang scheduling[^1]). Since there is now a native mechanism to express gang scheduling requirements now via `Workload` and `PodGroup` APIs, this KEP brings gang scheduling feature to Job Controller by integrating these APIs directly into the Job controller lifecycle.

### Goals

- Job controller automatically creates `Workload` and `PodGroup` objects for Jobs that require gang scheduling.
- Job with `parallelism > 1` will use `GangSchedulingPolicy` with `minCount = parallelism`
- Jobs that don't qualify for gang scheduling will use `BasicSchedulingPolicy`
- Ensure proper ordering of Workload/PodGroup creation before pods creation
- Existing Jobs without gang scheduling continue to work normally

### Non-Goals

- Supporting dynamic changes to `minCount` or gang membership at runtime
- Complex workload structures with multiple nested PodGroups are not supported in alpha.

## Proposal
> This KEP depends on:
> - [KEP-4671: Gang Scheduling using Workload Object](https://github.com/kubernetes/enhancements/tree/master/keps/sig-scheduling/4671-gang-scheduling)
> - [KEP-5832: Decouple PodGroup API from Workload API](https://github.com/kubernetes/enhancements/pull/5833)

The Job controller will be extended to create `Workload` and `PodGroup` objects as part of its pod management lifecycle. 
This integration ensures that pods belonging to a Job are scheduled according to the appropriate scheduling policy (gang or basic) before they are created. 

For the alpha release, this feature is optimized for static batch workloads with a flat API structure where 
`minCount` is immutable. The key design principles are:
- One `Job` creates one `Workload` with one `PodGroup` representing a single homogeneous group of pods.
- The automatic policy selection is based on `Job` Type
  - Jobs with `parallelism > 1` use gang scheduling policy where `minCount` equals the Job's parallelism.
  - Jobs without indexed completion mode or `completions = 1`, use basic scheduling policy (pod-by-pod scheduling - `minCount`).
- Elastic Jobs (changing parallelism at runtime) are not supported when gang scheduling is active.
- Jobs created by higher-level controllers (i.e., JobSet) are skipped, the parent controller manages Workload/PodGroup lifecycle

An example of the Job and the corresponding Workload/PodGroup creation:

```yaml
apiVersion: batch/v1
kind: Job
metadata:
  name: distributed-training
  namespace: training
spec:
  parallelism: 8
  completions: 8
  completionMode: Indexed
  template:
    spec:
      containers:
      - name: trainer
        image: training-image:latest
        resources:
          limits:
            nvidia.com/gpu: 1
---
apiVersion: scheduling.k8s.io/v1alpha1
kind: PodGroup
metadata:
  name: pg-w-distributed-training
  namespace: training
  ownerReferences:
  - apiVersion: batch/v1
    kind: Job
    name: distributed-training
    uid: <job-uid>
    controller: true
spec:
  workloadRef:
    name: w-distributed-training
  podGroupTemplate:
    name: main
    schedulingPolicy:
      gang:
        minCount: 8  # Equal to Job.spec.parallelism
---
apiVersion: scheduling.k8s.io/v1alpha1
kind: Workload
metadata:
  name: w-distributed-training
  namespace: training
  ownerReferences:
  - apiVersion: batch/v1
    kind: Job
    name: distributed-training
    uid: <job-uid>
    controller: true
spec:
  podGroupTemplate:
  - name: pg-w-distributed-training
    policy:
      gang:
        minCount: 8  # Equal to Job.spec.parallelism
```

Then, the Job Controller will create the corresponding pods and set the `workloadRef` field:

```yaml
apiVersion: v1
kind: Pod
metadata:
  name: <job-name>-<random-suffix>
  namespace: training
  ownerReferences:
  - apiVersion: batch/v1
    kind: Job
    name: distributed-training
spec:
  workloadRef:
    name: w-distributed-training
    podGroupName: pg-w-distributed-training
  containers:
  - name: ...
```

### User Stories (Optional)

#### ML Training Job with Gang Scheduling

As a machine learning engineer, I want to run a distributed training job with 8 workers that must all be scheduled together. 
If only 7 workers can be scheduled, I don't want any pods to start because partial training would waste resources.

#### Standard Batch Job with Workload Tracking

As a data engineer, I want to run a batch processing job that processes files sequentially without gang scheduling requirements.

### Notes/Constraints/Caveats (Optional)

#### Alpha Constraints

- The alpha release targets simple, static batch workloads where the workload requirements are known at creation time.
- Each Job maps to one `PodGroup`. All pods in the Job are identical from a scheduling policy perspective.
- The `minCount` field in the Workload's `GangSchedulingPolicy` mirrors the Job's parallelism.
- There is no mechanism to opt-out of `Workload`/`PodGroup` creation for indexed (parallel) jobs if feature gate is enabled.
- When gang scheduling is active (parallel jobs), changes to `spec.parallelism` are blocked via admission validation because this would require changing `minCount`
- If a Job has `ownerReferences` indicating it is managed by another controller (i.e., JobSet), the Job controller 
will not create `Workload`/`PodGroup` objects.

### Risks and Mitigations

## Design Details

### Job Controller Changes

The Job controller reconciliation loop for processes each Job will be extended to ensure `Workload` and `PodGroup` objects exist before creating pods. The modified workflow proceeds as follows:

- Check if a `Workload` object already exists for this `Job`.
  - If not, determine the appropriate scheduling policy and create the `Workload` object with the determined policy.
  - If it already exists, verify the existing `Workload` matches the `Job` spec. If not, update the `Workload` object.
- Check if a `PodGroup` object already exists for this `Job`.
  - If not, create the `PodGroup` object referencing the `Workload`
  - If it already exists, verify the existing `PodGroup` correctly references the `Workload`.
- Execute existing pod management logic to create pods, include `workloadRef.podGroupName` in the pod spec to associate pods with the `PodGroup`.

The Job Controller will create `Workload` and `PodGroup` based on Job configuration:
 - `parallelism > 1`: `GangSchedulingPolicy` with `minCount=parallelism`
 - `parallelism = 1` (default): `BasicSchedulingPolicy`
 - `completions = 1, parallelism = 1`: `BasicSchedulingPolicy`
 - `completions > 1, parallelism = 1`: `BasicSchedulingPolicy`

The controller will require additional informers and listers for `Workload` and `PodGroup` objects. Both `Workload` and `PodGroup` are automatically garbage collected when the `Job` is deleted.

#### Object Creation Order

The Job controller must create objects in a strict order to ensures that the scheduler can properly validate pods 
against their scheduling policy before attempting to schedule them. The order is as follows:
1. `Workload` object
2. `PodGroup` object that will references the `Workload`
3. `Pod` objects which will reference `PodGroup` and `Workload`

### Admission Validation for Parallelism Changes

When gang scheduling is active, the Job controller relies on admission validation to block changes to `Job.spec.parallelism`. Since changing this field would require changing `minCount` in the `Workload` object, which is immutable.

### Test Plan

[x] I/we understand the owners of the involved components may require updates to
existing tests to make this code solid enough prior to committing the changes necessary
to implement this enhancement.

##### Prerequisite testing updates


##### Unit tests

- `k8s.io/kubernetes/pkg/controller/job_controller`: `2026-01-29` - `89.1%`
- `k8s.io/kubernetes/pkg/apis/batch/validation`:
- `k8s.io/kubernetes/pkg/registry/batch/job`: 
- Add test that verifies 
  - SchedulingPolicy for various Job configurations
  - Workload and PodGroup creation
  - pod creation includes correct `workloadRef`
  - Parallelism change is blocked for gang-scheduled Jobs and allowed for basic-scheduled Jobs
  - Job deletion cascades to Workload and PodGroup deletion
  - Feature gate disabled: Jobs work without Workload/PodGroup creation
  - Jobs with ownerReferences (managed by higher-level controllers) do not create Workload/PodGroup

##### Integration tests

We will add the following integration tests to the Job controller `https://github.com/kubernetes/kubernetes/blob/v1.35.0/test/integration/job/job_test.go`:
- Gang and Basic Scheduling Lifecycle Test (create, update, delete Job, verify Workload and PodGroup creation, verify pods have workloadRef, verify Job deletion cascades to Workload and PodGroup deletion)
- Failure Recovery Test (create Job with Workload API unavailable, verify Job controller retries, verify Workload is eventually created)
- Feature gate disable/enable (Jobs work without Workload/PodGroup creation (Jobs with ownerReferences managed by higher-level controllers do not create Workload/PodGroup))

##### e2e tests

- End-to-end gang scheduling, all pods scheduled together or none
- Mixed workloads, gang and basic Jobs coexist
- Failure scenarios, i.e., insufficient resources for gang, partial failures

### Graduation Criteria

#### Alpha (v1.36)

- Feature is implemented behind feature gate `EnableWorkloadWithJob` (default: disabled)
- Job controller creates `Workload` and `PodGroup` objects for Jobs when feature gate is enabled
- Job controller sets `podGroupKind: PodGroup` in pod specs, opting into the explicit runtime PodGroup model
- Gang scheduling policy applied to indexed parallel Jobs (`parallelism > 1` with `Indexed` completion mode)
- Basic scheduling policy applied to all other Job types
- Jobs managed by higher-level controllers skip Workload/PodGroup creation
- Admission validation blocks `parallelism` changes for gang scheduling Jobs
- Unit tests for all new Job controller logic
- Integration tests for Workload/PodGroup creation flow
- Documentation for enabling and using the feature

#### Beta
- Feature gate `EnableWorkloadWithJob` is enabled by default
- Address feedback from alpha
- E2e tests covering gang scheduling scenarios
- Metrics for monitoring Workload/PodGroup creation and scheduling outcomes
- Performance testing to validate no significant impact on Job creation latency

#### GA

TBD after beta release

#### Deprecation

N/A for alpha release

### Upgrade / Downgrade Strategy

- **Upgrade:**
  1. Upgrade kube-apiserver
  2. Upgrade kube-scheduler
  3. Enable feature gate on kube-controller-manager
  4. New Jobs automatically get Workload/PodGroup objects
  5. Existing Jobs continue to work (no Workload created for them)

- **Downgrade:**
  1. Disable feature gate on kube-controller-manager
  2. New Jobs no longer get Workload/PodGroup objects
  3. Existing `Workload` and `PodGroup` objects remain
  4. Jobs with `workloadRef` on pods continue to run (field ignored)

- **Migration for Existing Jobs:**
  - Existing Jobs before upgrade do not automatically get Workload objects
  - To add gang scheduling to existing Jobs, delete and recreate them

### Version Skew Strategy

- kube-apiserver must be upgraded first to serve Workload API
- kube-scheduler should be upgraded next to handle gang scheduling
- kube-controller-manager can be upgraded last

If kube-controller-manager creates `Workload` but scheduler doesn't understand it:
  - Pods will have `workloadRef` but scheduler ignores it
  - Standard pod-by-pod scheduling occurs with no gang scheduling benefit

If scheduler supports gang scheduling but controller doesn't create `Workload` objects, pods without `workloadRef` are scheduled normally with no gang scheduling benefit

## Production Readiness Review Questionnaire


### Feature Enablement and Rollback


###### How can this feature be enabled / disabled in a live cluster?

- [x] Feature gate (also fill in values in `kep.yaml`)
  - Feature gate name: `EnableWorkloadWithJob`
  - Components depending on the feature gate:
    - kube-apiserver
    - kube-controller-manager
    - kube-scheduler
- [ ] Other
  - Describe the mechanism:
  - Will enabling / disabling the feature require downtime of the control
    plane?
  - Will enabling / disabling the feature require downtime or reprovisioning
    of a node?

###### Does enabling the feature change any default behavior?
Yes. When the feature gate is enabled:

1. The Job controller creates `Workload` and `PodGroup` objects for all Indexed parallel Jobs before creating pods.
2. Jobs (`parallelism > 1` with `Indexed` completion mode) use gang scheduling, meaning all pods must be scheduled together or none are scheduled.
3. Updates to `Job.spec.parallelism` are rejected for Jobs using gang scheduling.
4. Pod creation is delayed until the `PodGroup` object is acknowledged by the scheduler.

###### Can the feature be disabled once it has been enabled (i.e. can we roll back the enablement)?

Yes. The feature can be disabled (`EnableWorkloadWithJob: false`).

###### What happens if we reenable the feature if it was previously rolled back?

When the feature is re-enabled:
- New Jobs will have `Workload` and `PodGroup` objects created
- Existing Jobs will have `Workload` and `PodGroup` objects created on their next reconciliation cycle
- Jobs that were running without gang scheduling will be evaluated again; if they match gang scheduling criteria, a `Workload` with gang policy will be created
- Pods already running are not affected; gang scheduling only applies to newly created pods

###### Are there any tests for feature enablement/disablement?

Yes. We will add unit tests and integration tests for feature enablement/disablement.

### Rollout, Upgrade and Rollback Planning


###### How can a rollout or rollback fail? Can it impact already running workloads?

- If the API server doesn't support `Workload` and `PodGroup` APIs, the Job controller will fail to create Jobs (error creating `Workload`). Jobs will be requeued until the API server is upgraded.
- If the scheduler doesn't have gang scheduling feature enabled, pods will remain pending indefinitely.
- Already running Jobs are not affected by enabling the feature. Pods that are already scheduled and running continue to run. New Jobs or Jobs being reconciled will be affected.
- For the rollback, disabling the feature gate allows Jobs to work without `Workload` and `PodGroup` creation. `Workload` and `PodGroup` objects don't cause issues; they're ignored when the feature is disabled

###### What specific metrics should inform a rollback?

The following metrics should be monitored:

- `job_sync_duration_seconds`: If Job sync duration increases significantly, it may indicate issues with Workload/PodGroup creation
- `job_pods_creation_total`: If pod creation rate drops, gang scheduling may be blocking pod creation

###### Were upgrade and rollback tested? Was the upgrade->downgrade->upgrade path tested?

Yes. This will be tested as part of alpha release validation.

###### Is the rollout accompanied by any deprecations and/or removals of features, APIs, fields of API types, flags, etc.?

No.

### Monitoring Requirements

###### How can an operator determine if the feature is in use by workloads?

- `kubectl get workloads -A` will show `Workload` objects created by the Job controller
- `kubectl get podgroups -A` will show `PodGroup` objects created by the Job controller

###### How can someone using this feature know that it is working for their instance?

- [x] Events
  - Event Reason: `WorkloadCreated` - Emitted when `Workload` object is created for a Job
  - Event Reason: `PodGroupCreated` - Emitted when `PodGroup` object is created for a Job
- [ ] API .status
  - Condition name: 
  - Other field: 
- [ ] Other (treat as last resort)
  - Details:

###### What are the reasonable SLOs (Service Level Objectives) for the enhancement?

 - 99.9% of Workload/PodGroup creations succeed within 10 seconds

###### What are the SLIs (Service Level Indicators) an operator can use to determine the health of the service?

- [x] Metrics
  - Metric name:
  - [Optional] Aggregation method:
  - Components exposing the metric:
- [ ] Other (treat as last resort)
  - Details:

###### Are there any missing metrics that would be useful to have to improve observability of this feature?

### Dependencies


###### Does this feature depend on any specific services running in the cluster?

- `scheduling.k8s.io/v1alpha1` for `Workload` and `PodGroup` APIs
- `kube-scheduler` with gang scheduling feature enabled

### Scalability


###### Will enabling / using this feature result in any new API calls?

Yes. For each Job, the following additional API calls are made:
- `GET Workload` - 1 per Job sync
- `CREATE Workload` - 1 per Job creation
- `GET PodGroup` - 1 per Job sync
- `CREATE PodGroup` - 1 per Job creation
- `PATCH PodGroup.status` - 1 per scheduling decision
- `WATCH Workload` - Continuous
- `WATCH PodGroup` - Continuous

###### Will enabling / using this feature result in introducing new API types?

No.

###### Will enabling / using this feature result in any new calls to the cloud provider?

No.

###### Will enabling / using this feature result in increasing size or count of the existing API objects?

Yes. Each Job creates 1 `Workload`(~500 bytes) and 1 `PodGroup`(~500 bytes) object. In addition to Each Pod gains a `workloadRef` field (~100 bytes).

For a cluster with 10,000 Jobs, this adds approximately:
- 10,000 Workload objects
- 10,000 PodGroup objects
- ~10MB additional etcd storage

###### Will enabling / using this feature result in increasing time taken by any operations covered by existing SLIs/SLOs?

Yes. Due to creating `Workload` and `PodGroup` objects for each Job. And For scheduler waiting time.

###### Will enabling / using this feature result in non-negligible increase of resource usage (CPU, RAM, disk, IO, ...) in any components?

Yes.
- Kube-controller-manager: Additional memory for `Workload` and `PodGroup` informers. Estimated ~50MB for 10,000 objects.
- Kube-scheduler: Additional memory for `Workload` and `PodGroup` caches. Estimated ~50MB for 10,000 objects.
- etcd: Additional storage for `Workload` and `PodGroup` objects. Estimated ~10MB for 10,000 Jobs.
- kube-apiserver: Additional watches for `Workload` and `PodGroup` resources. Minimal CPU impact.

###### Can enabling / using this feature result in resource exhaustion of some node resources (PIDs, sockets, inodes, etc.)?

No. This feature is purely control-plane and does not affect node resources.

### Troubleshooting


###### How does this feature react if the API server and/or etcd is unavailable?

- Job Controller cannot create Workloads/PodGroups
- Retry with exponential backoff when kube-apiserver recovers
- Existing Jobs with Workloads continue to run

###### What are other known failure modes?


###### What steps should be taken if SLOs are not being met to determine the problem?

- Verify `EnableWorkloadWithJob` is enabled on all control plane components
- Check controller-manager logs for errors related to Workload/PodGroup creation
- Review `job_sync_duration_seconds`, `workload_creation_duration_seconds`
- Check resource constraints since gang scheduling may fail if cluster doesn't have sufficient resources

## Implementation History
- 2026-01-29: KEP created

## Drawbacks

## Alternatives


## Infrastructure Needed (Optional)

[^1]: The Kubernetes community uses the term "gang scheduling" to mean "all-or-nothing scheduling of a set of pods" [1,2,3,4,5,6,7,8,9,10,11,12,13]. In the Kubernetes context, it does not imply time-multiplexing (in contrast to prior academic work such as [Feitelson and Rudolph](https://doi.org/10.1016/0743-7315(92)90014-E), and in contrast to [Slurm Gang Scheduling](https://slurm.schedmd.com/gang_scheduling.html). ↩
