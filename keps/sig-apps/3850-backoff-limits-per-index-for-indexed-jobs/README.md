# KEP-3850: Backoff Limits Per Index For Indexed Jobs

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
  - [Notes/Constraints/Caveats (Optional)](#notesconstraintscaveats-optional)
  - [Risks and Mitigations](#risks-and-mitigations)
    - [The Job object too big](#the-job-object-too-big)
  - [Expotential backoff delay issue](#expotential-backoff-delay-issue)
  - [Too fast Job status updates](#too-fast-job-status-updates)
- [Design Details](#design-details)
  - [Job API](#job-api)
  - [Tracking the number of failures per index](#tracking-the-number-of-failures-per-index)
  - [Failed indexes format](#failed-indexes-format)
  - [Job completion](#job-completion)
  - [FailIndex action](#failindex-action)
  - [Expotential backoff delay per index](#expotential-backoff-delay-per-index)
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
    - [Upgrade](#upgrade)
    - [Downgrade](#downgrade)
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
  - [backoffLimitPerIndex inside new runPolicy](#backofflimitperindex-inside-new-runpolicy)
  - [Mark Job Complete if some indexes failed](#mark-job-complete-if-some-indexes-failed)
  - [Support backoffLimitPerIndex when restartPolicy=OnFailure](#support-backofflimitperindex-when-restartpolicyonfailure)
  - [Mutually exclusive backoffLimit and backoffLimitPerIndex](#mutually-exclusive-backofflimit-and-backofflimitperindex)
  - [Use bool field](#use-bool-field)
  - [Use enum field](#use-enum-field)
  - [Global expotential backoff delay](#global-expotential-backoff-delay)
  - [Expotential backoff delay with in-memory tracking](#expotential-backoff-delay-with-in-memory-tracking)
  - [Alternative ways to support high number of completions](#alternative-ways-to-support-high-number-of-completions)
    - [Keep failedIndexes field as a bitmap](#keep-failedindexes-field-as-a-bitmap)
    - [Keep the list of failed indexes in a dedicated API object](#keep-the-list-of-failed-indexes-in-a-dedicated-api-object)
    - [Implicit limit on the number of failed indexes](#implicit-limit-on-the-number-of-failed-indexes)
  - [Skip uncountedTerminatedPods when backoffLimitPerIndex is used](#skip-uncountedterminatedpods-when-backofflimitperindex-is-used)
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
- [x] (R) Graduation criteria is in place
  - [ ] (R) [all GA Endpoints](https://github.com/kubernetes/community/pull/1806) must be hit by [Conformance Tests](https://github.com/kubernetes/community/blob/master/contributors/devel/sig-architecture/conformance-tests.md)
- [x] (R) Production readiness review completed
- [x] (R) Production readiness review approved
- [x] "Implementation History" section is up-to-date for milestone
- [x] User-facing documentation has been created in [kubernetes/website], for publication to [kubernetes.io]
- [ ] Supporting documentation—e.g., additional design documents, links to mailing list discussions/SIG meetings, relevant PRs/issues, release notes

[kubernetes.io]: https://kubernetes.io/
[kubernetes/enhancements]: https://git.k8s.io/enhancements
[kubernetes/kubernetes]: https://git.k8s.io/kubernetes
[kubernetes/website]: https://git.k8s.io/website

## Summary

This KEP extends the Job API to support indexed jobs where the backoff limit is
per index, and the Job can continue execution despite some of its indexes failing.

## Motivation

Currently, the indexes of an indexed job share a single backoff limit.
When the job reaches this shared backoff limit, the job controller marks the entire
job as failed, and the resources are cleaned up, including indexes that have yet
to run to completion.

As a result, the current implementation does not cover the situation where the workload
is truly embarrassingly parallel and each index is independent of other indexes.

For instance, if indexed jobs were used as the basis for a suite of long-running integration tests,
then each test run would only be able to find a single test failure.

Other popular batch services like AWS Batch use a separate backoff limit for each index,
showing that this is a common use case that should be supported by Kubernetes.

### Goals

- allow to count failures towards the backoffLimit independently for all indexes,
- allow to continue Job execution despite some of its indexes failing,
- allow to fail an index (stop recreating pods for the index) using pod failure policy.

### Non-Goals

- allow to control the number of retries per index when pod's `restartPolicy=OnFailure`
(see [Support backoffLimitPerIndex when restartPolicy=OnFailure](#support-backofflimitperindex-when-restartpolicyonfailure)).

<!--
What is out of scope for this KEP? Listing non-goals helps to focus discussion
and make progress.
-->

## Proposal

We propose a new policy for running Indexed Jobs in which the backoff limit
controls the number of retries per index. When the new policy is used all
indexes execute until their success or failure. We also propose a new API field
to control the number of failed indexes.

Additionally, we propose a new action in [PodFailurePolicy](https://github.com/kubernetes/enhancements/tree/master/keps/sig-apps/3329-retriable-and-non-retriable-failures), called FailIndex,
to short-circuit failing of the index before the backoff limit per index is
reached.

### User Stories (Optional)

#### Story 1

As a CI/CD platform administrator, I want to use Indexed Jobs to run
suites of integration tests, one suite per index. A failure of one suite should
not interrupt running of other suites. Additionally, I would like to be able
to control the maximal number of retries per index.

The following Job configuration could satisfy my use case:

```yaml
apiVersion: v1
kind: Job
spec:
  parallelism: 10
  completions: 10
  completionMode: Indexed
  backoffLimitPerIndex: 1
  template:
    spec:
      restartPolicy: Never
      containers:
      - name: job-container
        image: job-image
        command: ["./tests-runner"]
```

In this case, we run 10 indexes representing the test suites. We allow for one
failure per index.

#### Story 2

As a CI/CD platform administrator from the [Story 1](#story-1) I want to be able
to control the failures with the pod failure policy. In particular, I want
to be able to use pod failure policy to avoid restarts of some indexes, based
on exit codes.

The following Job configuration could satisfy my use case:

```yaml
apiVersion: v1
kind: Job
spec:
  parallelism: 10
  completions: 10
  completionMode: Indexed
  backoffLimitPerIndex: 1
  template:
    spec:
      restartPolicy: Never
      containers:
      - name: job-container
        image: job-image
        command: ["./tests-runner"]
  podFailurePolicy:
    rules:
    - action: FailIndex
      onExitCodes:
        operator: In
        values: [42]
```

#### Story 3

As a CI/CD platform administrator from the [Story 1](#story-1) I want to be able
to fail the entire Job if the number of failed indexes exceeds 50%. I want to
do this in order to cut down costs of running the tests in case of compilation
issues that would result in all tests failing.

The following Job configuration could satisfy my use case:

```yaml
apiVersion: v1
kind: Job
spec:
  parallelism: 10
  completions: 10
  completionMode: Indexed
  backoffLimitPerIndex: 1
  maxFailedIndexes: 5
  template:
    spec:
      restartPolicy: Never
      containers:
      - name: job-container
        image: job-image
        command: ["./tests-runner"]
```

### Notes/Constraints/Caveats (Optional)

<!--
What are the caveats to the proposal?
What are some important details that didn't come across above?
Go in to as much detail as necessary here.
This might be a good place to talk about core concepts and how they relate.
-->

### Risks and Mitigations

#### The Job object too big

With the new field `.status.failedIndexes` the Job object can be significantly
larger as every failed index is recorded in the field.

Note that, the similar risk is also present for Indexed Jobs, regarding the
already existing `.status.completedIndexes` field (see
[Indexed Jobs can break with high number of parallelism or completions](https://github.com/kubernetes/kubernetes/issues/118085)).

In order to mitigate this risk we first constrain the `.spec.maxFailedIndexes`
to `10^5`, which is the same limit as for `.spec.parallelism` currently.

Second, we validate if the fields are inside of the scalability limits:
1. `.spec.completions<=10^5`, `.spec.parallelism<=10^5`, `spec.maxFailedIndexes<=10^5`
2. `spec.completions` unlimited (<= max int32 ~2*10^9), `.spec.parallelism<=10^4`, `spec.maxFailedIndexes<=10^4`

In (1.), in the worst case scenario, every index is either present
in `completedIndexes` or `failedIndexes`, but not in both. Thus the total
sum of both fields is limited by `(5+1)*10^5=0.572Mi`, where:
- 5 is the maximal number of digits in the indexes,
- 1 is for separation character,
- 10^5 is the total number of listed indexes.

In (2.) the worst case scenario for the `completedIndexes` field is when every
third index is not in the field, because it corresponds to either a failed or
a hanging indexes, so it is a "gap". Then, between every gap we have two indexes
listed. Thus, the size of the `completedIndexes` field is limited
by: `(10+1)*2*(10^4+10^4)=0.42Mi`, where:
- 10 is the maximal number of digits in the indexes
- 1 is for the separation character
- 2*(10^4+10^4) is the number of indexes explicitly listed in the field - two indexes per gap.

The size of the `failedIndexes` field is limited by: `(10+1)*10^4=0.105Mi`, where:
- 10 is the maximal number of digits in the indexes,
- 1 is for the separation character
- 10^4 is the maximal number of indexes present in the field.

Thus, the size of both fields is capped at `0.572Mi` for the limits in (1.) and
`0.525Mi` for the limits in (2.).

For comparison, before the introduction of `.status.failedIndexes`, the max
size of the `.status.completedIndexes` was limited by `(5+1)*10^5*2/3=0.382Mi` in
the (1.) case, and `(10+1)*2*10^4=0.21Mi` in the (2.) case. This means an increase
of `0.19Mi`.

The values of the limits are aligned with the values for the soft limits proposed
as a fix for the for regular indexed jobs
(see [here](https://github.com/kubernetes/kubernetes/issues/118085#issuecomment-1564520559)).
However, in case when `backoffLimitPerIndex` is used we propose these limits
to be hard.

We believe that the scalability limits should be enough for most of Job use-cases.
For workloads requiring larger jobs users should be able to create multiple Jobs,
orchestrated by the [JobSet](https://github.com/kubernetes-sigs/jobset).

### Expotential backoff delay issue

Currently, a pod is recreated by the Job controller with expotential backoff
delay (10s, 20s, 40s ...), counted from the last failure time.

One complication is that the last failure time for failed pods may increase with
time, as it fallbacks to `now` in some cases
(see in [code](https://github.com/kubernetes/kubernetes/blob/master/pkg/controller/job/backoff_utils.go#L160-L182)).
Thus, there is a risk that due to the presence
of pods hitting the fallback the last failure time is continuously bumped,
thus shifting the time to recreate the pod.

This risk is present both when computing the expotential backoff delay globally
(as for regular indexed Jobs), or per-index as proposed in in this KEP
(see [Expotential backoff delay per index](#expotential-backoff-delay-per-index)).

In order to mitigate this risk currently the time of last failure is recorded
in-memory (globally for all pods within a Job). And a new failed pod may bump
it only until it is added to the `uncountedTerminatedPods` structure.

However, tracking the last failure time per index might be costly for memory
consumption (see [Expotential backoff delay with in-memory tracking](#expotential-backoff-delay-with-in-memory-tracking)).

Thus, in order to mitigate this risk we propose to compute the finish time for
a pod as the first available value of the following (avoiding the ever-increasing
fallback to `now`):
1. max `finishAt` of all containers, if specified for all containers
2. `LastTransitionTime` for the `Ready=False` condition
3. `deletionTimestamp` - `deletionGracePeriodSeconds` if `deletionTimestamp` is set

Here (3.) is used to mark the moment of deletion which is used to approximate
the current behavior. (2.) is used when Kubelet loses track of one of its containers,
the `Ready=False` condition is set by Kubelet when transitioning a pod to `Failed`
phase: https://github.com/kubernetes/kubernetes/blob/release-1.27/pkg/kubelet/status/status_manager.go#L1060-L1068.
When none of the above conditions is satisfied to compute the finish time we
fallback to the pod's creation time.

This fix can be considered a preparatory PR before the KEP, as to some extent
is solves the preexisting issue.

### Too fast Job status updates

In this KEP the Job controller needs to keep updating the new status field
`.status.failedIndexes` to reflect the current status of the Job. This can raise
concerns of overwhelming the API server with status updates.

First, observe that the new field does not entail additional Job status updates.
When a pod terminates (either failure or success), it triggers Job status update
to increment the `status.failed` or `.status.succeeded` counter fields. These
updates are also used to update the pre-existing `status.completedIndexes`
field, and the new `status.failedIndexes` field.

Second, in order to mitigate this risk there is already a mechanism present in
the Job controller, to bulk Job status updates per Job.

The way the mechanism works is that Job controller maintains a queue of `syncJob`
invocations per job
(see [in code](https://github.com/kubernetes/kubernetes/blob/72a3990728b2a8979effb37b9800beb3117349f6/pkg/controller/job/job_controller.go#L118)).
New items are added to the queue with a delay (1s for pod events, such as:
delete, add, update). The delay allows for deduplication of the sync per Job.

One place to queue a new item in the queue, specific to this KEP, is when
the expotential backoff delay hasn't elapsed for any index (allowing pod
recreation), then we requeue the next Job status update. The delay is computed
as minimum of all delays computed for all indexes requiring pod recreation,
but not less that 1s.

<!--
What are the risks of this proposal, and how do we mitigate? Think broadly.
For example, consider both security and how this will impact the larger
Kubernetes ecosystem.

How will security be reviewed, and by whom?

How will UX be reviewed, and by whom?

Consider including folks who also work outside the SIG or subproject.
-->

## Design Details

We introduce a new Job API field, called `.spec.backoffLimitPerIndex`.
When set it limits the number of retries, counted independently for all indexes.

Additionally, we propose the `.spec.maxFailedIndexes` to control
the maximal number of failed indexes. Once the number is exceeded the entire
Job is marked Failed and its execution is terminated.

We also propose to extend the PodFailurePolicy with a new action, called
`FailIndex` to allow an index to fail fast before reaching the backoff limit
per index.

### Job API

```golang

// PodFailurePolicyAction specifies how a Pod failure is handled.
// +enum
type PodFailurePolicyAction string

const (
  // This is an action which might be taken on a pod failure - mark the
  // Job's index as failed to avoid restarts within this index. This action
  // can only be used when backoffLimitPerIndex is set.
  PodFailurePolicyActionFailIndex PodFailurePolicyAction = "FailIndex"
  ...
)
...

// JobSpec describes how the job execution will look like.
type JobSpec struct {
  ...
  // Specifies the limit for the number of retries within an
  // index before marking this index as failed. When enabled the number of
  // failures per index is kept in the pod's
  // batch.kubernetes.io/job-index-failure-count annotation. It can only
  // be set when Job's completionMode=Indexed, and the Pod's restart
  // policy is Never. The field is immutable.
  // +optional
  BackoffLimitPerIndex *int32

  // Specifies the maximal number of failed indexes before marking the Job as
  // failed, when backoffLimitPerIndex is set. Once the number of failed
  // indexes exceeds this number the entire Job is marked as Failed and its
  // execution is terminated. When left as null the job continues execution of
  // all of its indexes and is marked with the `Complete` Job condition.
  // It can only be specified when backoffLimitPerIndex is set.
  // It can be null or up to completions. It is required and must be
  // less than or equal to 10^4 when is completions greater than 10^5.
  // +optional
  MaxFailedIndexes *int32
  ...
}

type JobStatus struct {
  ...

  // FailedIndexes holds the failed indexes when backoffLimitPerIndex is set.
  // The indexes are represented in the text format analogous as for the
  // `completedIndexes` field, ie. they are kept as decimal integers
  // separated by commas. The numbers are listed in increasing order. Three or
  // more consecutive numbers are compressed and represented by the first and
  // last element of the series, separated by a hyphen.
  // For example, if the failed indexes are 1, 3, 4, 5 and 7, they are
  // represented as "1,3-5,7".
  // +optional
  FailedIndexes *string
}
```

Note that, the `PodFailurePolicyAction` type is already defined in master with
three possible enum values: `Ignore`, `FailJob` and `Count` (see [here](https://github.com/kubernetes/kubernetes/blob/72a3990728b2a8979effb37b9800beb3117349f6/pkg/apis/batch/types.go#L113-L131)).

We allow to specify custom `.spec.backoffLimit` and `.spec.backoffLimitPerIndex`.
This allows for a controlled downgrade. Also, when `.spec.backoffLimitPerIndex`
is specified, then we default `.spec.backoffLimit` to max int32 value. This way
we ensure old clients of the API wouldn't break when reading or trying to modify
the `.spec.backoffLimit` that has nil value.

### Tracking the number of failures per index

In order to determine if the backoff limit per index is exceeded we keep
track of the number of failures per index. For this purpose we use the Pod
annotation, `batch.kubernetes.io/job-index-failure-count`, which holds the value
of the number of pod failures for a given index. It is set to `0` for the first
pod created for a given index.

When Job controller sees a failed pod corresponding to a given index, and the
value of the annotation `batch.kubernetes.io/job-index-failure-count` is greater
or equal to the configured backoff limit per index then the index is marked
as failed and added to `.status.failedIndexes`.

When Job controller creates replacement pods for failed pods for a given
index it checks if the index isn't finished yet (it is not in
`.status.failedIndexes` nor `.status.completedIndexes`).
Then, if `x` is the highest `batch.kubernetes.io/job-index-failure-count`
for the index, the newly created pod will have the annotation set to `x+1`.
An exception is when the newly failed pod matches the `Ignore` action in pod
failure policy. In this case the replacement pod does not increment the
value in the annotation.

In order to keep track of the number of failures per index, the Job controller
removes finalizers of a failed pod for a given index, only once the replacement
pod (with incremented value of `batch.kubernetes.io/job-index-failure-count`) is
created, or the index is marked as failed in `.status.failedIndexes`. This means
that these are the main steps when handling a failed pod to prepare it for
deletion:
1. Pod is recognized as failed
2. pod UID is recorded in Job status (`.status.uncountedTerminatedPods`)
3. the replacement Pod is created
4. Pod's finalizer is removed

Here, the new feature adds a dependency between steps (3.) and (4.) as previously
these steps could be performed in any order. Note that, typically when a pod is
deleted or fails the replacement pod is created with a backoff delay, starting
from 10s. This means, that after the proposed change the pod finalizer removal
will be paused for at least 10s, until the backoff elapses and the replacement
pod is created. While this may result in pods hanging around before garbage
collection, it does not affect directly the rate of pod recreation.

Note that, the first step (1.) will also be impacted by
[KEP-3939: Consider Terminating pods as active pods in Jobs.](https://github.com/kubernetes/enhancements/issues/3939)

### Failed indexes format

The format of the `.status.failedIndexes` field is analogous to the one used for
successful indexes represented by the [`completedIndexes` field](https://github.com/kubernetes/enhancements/tree/master/keps/sig-apps/2214-indexed-job#track-completed-indexes-in-job-status)), which is a
text format grouping consecutive integers into ranges. In a special case, when
the indexes are non-consecutive they are represented by comma-separated numbers.
In the worst-case scenario this is a string of comma-separated even values. In
order to constrain the size of the field we cap the number of completions
(see [The Job object too big](#the-job-object-too-big) for more details).

### Job completion

When backoff limit per index is used, then we execute indexes until all of them
are completed (either failed or succeeded), or the number of failed indexes
exceeds the specified `.spec.maxFailedIndexes`.

Then, the Job is marked as completed (the `Complete` Job condition type) when
all indexes are succeeded. The Job is marked as failed (the `Failed` Job condition)
when at least one index is failed. The `Failed` condition is added once
all indexes completed their execution (either failed or succeeded), or when
the number of failed indexes exceeds the specified `.spec.maxFailedIndexes`.

### FailIndex action

In order to allow early termination of indexes with the `FailIndex` action
we add the corresponding index to the set of failed indexes represented by
`.status.failedIndexes`. This action can only be used if backoff limit per index
is used.

### Expotential backoff delay per index

First, we solve the issue of increasing failure time for deleted pods when the
finalizer removal is delayed, by modifying the definition of the pod finish time,
to avoid fallback to `now`
(see also [Expotential backoff delay issue](#expotential-backoff-delay-issue)).

Second, we compute the backoff delay within each index independently. The number
of consecutive failures per-index can be derived from the
`batch.kubernetes.io/job-index-failure-count` annotation of the last failed pod,
plus one. This is because any successful pod marks the index as successful and
stops retries. Note that, using the annotation value means that failed pods
matching the Ignore rule are skipped in the calculation, but this behavior is
consistent with handling ignored pod failures for regular backoff limit.

### Test Plan

<!--
**Note:** *Not required until targeted at a release.*
The goal is to ensure that we don't accept enhancements with inadequate testing.

All code is expected to have adequate tests (eventually with coverage
expectations). Please adhere to the [Kubernetes testing guidelines][testing-guidelines]
when drafting this test plan.

[testing-guidelines]: https://git.k8s.io/community/contributors/devel/sig-testing/testing.md
-->

[x] I/we understand the owners of the involved components may require updates to
existing tests to make this code solid enough prior to committing the changes necessary
to implement this enhancement.

##### Prerequisite testing updates

<!--
Based on reviewers feedback describe what additional tests need to be added prior
implementing this enhancement to ensure the enhancements have also solid foundations.
-->

##### Unit tests

Unit tests will be added along with any new code introduced. In particular,
the following scenarios will be covered with unit tests:
- handling or ignoring of `.spec.backoffLimitPerIndex` by the Job
  controller when the feature gate is enabled or disabled, respectively,
- handling of ignoring of the pod failure policy rule with `FailIndex` action
- the `JobBackoffLimitPerIndex` feature gate is enabled or disabled, respectively,
- validation of a job configuration with respect to `.spec.backoffLimitPerIndex` by
  kube-apiserver (including limits for `.spec.maxFailedIndexes`,
  `.spec.parallelism` and `.spec.completions`), when the feature gate is enabled
  or disabled,
- marking of the Job as `Complete` only once all indexes are completed,
- termination of Job execution and marking it as failed when
  `.spec.maxFailedIndexes` is exceeded.
- calculation of the expotential backoff delay per index when `backoffLimitPerIndex`
  is used.
- a fuzzer roundtrip test for API when `backoffLimit` is set to max int32.

<!--
In principle every added code should have complete unit test coverage, so providing
the exact set of tests will not bring additional value.
However, if complete unit test coverage is not possible, explain the reason of it
together with explanation why this is acceptable.
-->

<!--
Additionally, for Alpha try to enumerate the core package you will be touching
to implement this enhancement and provide the current unit coverage for those
in the form of:
- <package>: <date> - <current test coverage>
The data can be easily read from:
https://testgrid.k8s.io/sig-testing-canaries#ci-kubernetes-coverage-unit

This can inform certain test coverage improvements that we want to do before
extending the production code to implement this enhancement.
-->

The core packages (with their unit test coverage) which are going to be modified during the implementation:
- `k8s.io/kubernetes/pkg/controller/job`: `27 Apr 2023` - `90.4%`  <!--(main logic to handle backoffLimitPerIndex, FailIndex and maxFailedIndexes)-->
- `k8s.io/kubernetes/pkg/apis/batch/validation`: `27 Apr 2023` - `98.5%` <!--(validation of the job configuration for backoffLimitPerIndex and FailIndex)-->

##### Integration tests

<!--
This question should be filled when targeting a release.
For Alpha, describe what tests will be added to ensure proper quality of the enhancement.

For Beta and GA, add links to added tests together with links to k8s-triage for those tests:
https://storage.googleapis.com/k8s-triage/index.html
-->

The following scenarios will be covered with integration tests:
- enabling, disabling and re-enabling of the `JobBackoffLimitPerIndex` feature gate
- handling of the `.spec.backoffLimitPerIndex` when the `FailIndex` action is used,
- handling of the `.spec.backoffLimitPerIndex` when `.spec.maxFailedIndexes` isn't set,
- handling of the `.spec.backoffLimitPerIndex` when `.spec.maxFailedIndexes` is set,
- handling of the `.spec.backoffLimit` when `.spec.backoffLimitPerIndex` is set,
- handling of the expotential backoff delay per index when `.spec.backoffLimitPerIndex` is set.

More integration tests might be added to ensure good code coverage based on the
actual implementation.

##### e2e tests

<!--
This question should be filled when targeting a release.
For Alpha, describe what tests will be added to ensure proper quality of the enhancement.

For Beta and GA, add links to added tests together with links to k8s-triage for those tests:
https://storage.googleapis.com/k8s-triage/index.html

We expect no non-infra related flakes in the last month as a GA graduation criteria.
-->

The following scenarios will be covered with e2e tests:
- handling of the `.spec.backoffLimitPerIndex` when the `FailIndex` is used -
  the Job's index is marked as failed,
- handling of the `.spec.backoffLimitPerIndex` when the number of failures for
  an index exceeds the backoff - the index is marked as failed, but the Job
  continues its execution until all indexes are finished.
- handling of the `.spec.backoffLimitPerIndex` when `.spec.maxFailedIndexes`
  is set and exceeded - the Job is marked as Failed and its execution is
  terminated.

More integration tests might be added to ensure good code coverage based on the
actual implementation.

### Graduation Criteria

<!--
**Note:** *Not required until targeted at a release.*

Define graduation milestones.

These may be defined in terms of API maturity, [feature gate] graduations, or as
something else. The KEP should keep this high-level with a focus on what
signals will be looked at to determine graduation.

Consider the following in developing the graduation criteria for this enhancement:
- [Maturity levels (`alpha`, `beta`, `stable`)][maturity-levels]
- [Feature gate][feature gate] lifecycle
- [Deprecation policy][deprecation-policy]

Clearly define what graduation means by either linking to the [API doc
definition](https://kubernetes.io/docs/concepts/overview/kubernetes-api/#api-versioning)
or by redefining what graduation means.

In general we try to use the same stages (alpha, beta, GA), regardless of how the
functionality is accessed.

[feature gate]: https://git.k8s.io/community/contributors/devel/sig-architecture/feature-gates.md
[maturity-levels]: https://git.k8s.io/community/contributors/devel/sig-architecture/api_changes.md#alpha-beta-and-stable-versions
[deprecation-policy]: https://kubernetes.io/docs/reference/using-api/deprecation-policy/

Below are some examples to consider, in addition to the aforementioned [maturity levels][maturity-levels].

#### Alpha

- Feature implemented behind a feature flag
- Initial e2e tests completed and enabled

#### Beta

- Gather feedback from developers and surveys
- Complete features A, B, C
- Additional tests are in Testgrid and linked in KEP

#### GA

- N examples of real-world usage
- N installs
- More rigorous forms of testing—e.g., downgrade tests and scalability tests
- Allowing time for feedback

**Note:** Generally we also wait at least two releases between beta and
GA/stable, because there's no opportunity for user feedback, or even bug reports,
in back-to-back releases.

**For non-optional features moving to GA, the graduation criteria must include
[conformance tests].**

[conformance tests]: https://git.k8s.io/community/contributors/devel/sig-architecture/conformance-tests.md

#### Deprecation

- Announce deprecation and support policy of the existing flag
- Two versions passed since introducing the functionality that deprecates the flag (to address version skew)
- Address feedback on usage/changed behavior, provided on GitHub issues
- Deprecate the flag
-->

#### Alpha

- the feature implemented behind the `JobBackoffLimitPerIndex` feature flag
- change the logic of computing the expotential backoff delay (see [here](#expotential-backoff-delay-issue))
- user-facing documentation, including the warning for setting completions > 10^5
- The `JobBackoffLimitPerIndex` feature flag disabled by default
- Tests: unit and integration

#### Beta

- Address reviews and bug reports from Alpha users
- Implement the `job_finished_indexes_total` metric
- E2e tests are in Testgrid and linked in KEP
- Move the [new reason declarations](https://github.com/kubernetes/kubernetes/blob/dc28eeaa3a6e18ef683f4b2379234c2284d5577e/pkg/controller/job/job_controller.go#L82-L89) from Job controller to the API package
- Evaluate performance of Job controller for jobs using backoff limit per index
  with benchmarks at the integration or e2e level (discussion pointers from Alpha
  review: [thread1](https://github.com/kubernetes/kubernetes/pull/118009#discussion_r1261694406) and [thread2](https://github.com/kubernetes/kubernetes/pull/118009#discussion_r1263862076))
- The feature flag enabled by default

#### GA

- Address reviews and bug reports from Beta users
- Write a blog post about the feature
- Graduate e2e tests as conformance tests
- Lock the `JobBackoffLimitPerIndex` feature gate
- Declare deprecation of the `JobBackoffLimitPerIndex` feature gate in documentation

### Upgrade / Downgrade Strategy

#### Upgrade

An upgrade to a version which supports this feature should not require any
additional configuration changes. In order to use this feature after an upgrade
users will need to configure their Jobs by specifying
`.spec.backoffLimitPerIndex`.
There is no difference in behavior of Jobs if `.spec.backoffLimitPerIndex` is
not set.

#### Downgrade

A downgrade to a version which does not support this feature should not require
any additional configuration changes. Jobs which specified
`.spec.backoffLimitPerIndex` (to make use of this feature) will be
handled in a default way, ie. using the `.spec.backoffLimit`.
However, since the `.spec.backoffLimit` defaults to max int32 value
(see [here](#job-api)) is might require a manual setting of the `.spec.backoffLimit`
to ensure failed pods are not retried indefinitely.

<!--
If applicable, how will the component be upgraded and downgraded? Make sure
this is in the test plan.

Consider the following in developing an upgrade/downgrade strategy for this
enhancement:
- What changes (in invocations, configurations, API use, etc.) is an existing
  cluster required to make on upgrade, in order to maintain previous behavior?
- What changes (in invocations, configurations, API use, etc.) is an existing
  cluster required to make on upgrade, in order to make use of the enhancement?
-->

### Version Skew Strategy

This feature is limited to control plane.

Note that, kube-apiserver can be in the N+1 skew version relative to the
kube-controller-manager (see [here](https://kubernetes.io/releases/version-skew-policy/#kube-controller-manager-kube-scheduler-and-cloud-controller-manager)).
In that case, the Job controller operates on the version of the Job object that
already supports the new Job API.

<!--
If applicable, how will the component handle version skew with other
components? What are the guarantees? Make sure this is in the test plan.

Consider the following in developing a version skew strategy for this
enhancement:
- Does this enhancement involve coordinating behavior in the control plane and
  in the kubelet? How does an n-2 kubelet without this feature available behave
  when this feature is used?
- Will any other components on the node change? For example, changes to CSI,
  CRI or CNI may require updating that component before the kubelet.
-->

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

- [x] Feature gate (also fill in values in `kep.yaml`)
  - Feature gate name: JobBackoffLimitPerIndex
  - Components depending on the feature gate: kube-apiserver, kube-controller-manager
- [ ] Other
  - Describe the mechanism:
  - Will enabling / disabling the feature require downtime of the control
    plane?
  - Will enabling / disabling the feature require downtime or reprovisioning
    of a node? (Do not assume `Dynamic Kubelet Config` feature is enabled).

###### Does enabling the feature change any default behavior?

No.

<!--
Any change of default behavior may be surprising to users or break existing
automations, so be extremely careful here.
-->

###### Can the feature be disabled once it has been enabled (i.e. can we roll back the enablement)?

Yes. Using the feature gate is the recommended way. When the feature is disabled
the Job controller manager handles pod failures in the default way, even if
`.spec.backoffLimitPerIndex` is set.

<!--
Describe the consequences on existing workloads (e.g., if this is a runtime
feature, can it break the existing applications?).

Feature gates are typically disabled by setting the flag to `false` and
restarting the component. No other changes should be necessary to disable the
feature.

NOTE: Also set `disable-supported` to `true` or `false` in `kep.yaml`.
-->

###### What happens if we reenable the feature if it was previously rolled back?

The Job controller starts to handle pod failures according to the specified
`.spec.backoffLimitPerIndex` or `.spec.maxFailedIndexes` fields.

###### Are there any tests for feature enablement/disablement?

Yes, there is an [integration test](https://github.com/kubernetes/kubernetes/blob/dc28eeaa3a6e18ef683f4b2379234c2284d5577e/test/integration/job/job_test.go#L763)
which tests the following path: enablement -> disablement -> re-enablement.

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

This change does not impact how the rollout or rollback fail.

The change is opt-in, thus a rollout doesn't impact already running pods.

The rollback might affect how pod failures are handled, since they will
be counted only against `.spec.backoffLimit`, which is defaulted to max int32
value, when using `.spec.backoffLimitPerIndex` (see [here](#job-api)).
Thus, similarly as in case of a downgrade (see [here](#downgrade))
it might be required to manually set `spec.backoffLimit` to ensure failed pods
are not retried indefinitely.

<!--
Try to be as paranoid as possible - e.g., what if some components will restart
mid-rollout?

Be sure to consider highly-available clusters, where, for example,
feature flags will be enabled on some API servers and not others during the
rollout. Similarly, consider large clusters and how enablement/disablement
will rollout across nodes.
-->

###### What specific metrics should inform a rollback?

A substantial increase in the `job_sync_duration_seconds`.

Also, a substantial increase in the total number of pods, as it may take
additional time to get the finalizers removed.

Additionally, a substantial increase in the difference of
`terminated_pods_tracking_finalizer_total` for the `add` and `delete` labels may
indicate that it takes too long to delete the finalizers.

The feature is opt-in so in case of issues it is enough not to use the
backoffLimitPerIndex API field.

<!--
What signals should users be paying attention to when the feature is young
that might indicate a serious problem?
-->

###### Were upgrade and rollback tested? Was the upgrade->downgrade->upgrade path tested?

The Upgrade->downgrade->upgrade testing was done manually using the `alpha`
version in 1.28 with the following steps:

1. Start the cluster with the `JobBackoffLimitPerIndex` enabled:
```sh
kind create cluster --name per-index --image kindest/node:v1.28.0 --config config.yaml
```
using `config.yaml`:
```yaml
kind: Cluster
apiVersion: kind.x-k8s.io/v1alpha4
featureGates:
  "JobBackoffLimitPerIndex": true
nodes:
- role: control-plane
- role: worker
```

Then, create the job using `.spec.backoffLimitPerIndex=1`:

```sh
kubectl create -f job.yaml
```
using `job.yaml`:
```yaml
apiVersion: batch/v1
kind: Job
metadata:
  name: job-longrun
spec:
  parallelism: 3
  completions: 3
  completionMode: Indexed
  backoffLimitPerIndex: 1
  template:
    spec:
      restartPolicy: Never
      containers:
      - name: sleep
        image: busybox:1.36.1
        command: ["sleep"]
        args: ["1800"]  # 30min
        imagePullPolicy: IfNotPresent
```

Await for the pods to be running and delete 0-indexed pod:
```sh
kubectl delete pods -l job-name=job-longrun -l batch.kubernetes.io/job-completion-index=0 --grace-period=1
```
Await for the replacement pod to be created and repeat the deletion.

Check job status and confirm `.status.failedIndexes="0"`

```sh
kubectl get jobs -ljob-name=job-longrun -oyaml
```
Also, notice that `.status.active=2`, because the pod for a failed index is not
re-created.

2. Simulate downgrade by disabling the feature for api server and control-plane.

Then, verify that 3 pods are running again, and the `.status.failedIndexes` is
gone by:
```sh
kubectl get jobs -ljob-name=job-longrun -oyaml
```
this will produce output similar to:
```yaml
  ...
  status:
    active: 3
    failed: 2
    ready: 2
```

3. Simulate upgrade by re-enabling the feature for api server and control-plane.

Then, delete 1-indexed pod:
```sh
kubectl delete pods -l job-name=job-longrun -l batch.kubernetes.io/job-completion-index=1 --grace-period=1
```
Await for the replacement pod to be created and repeat the deletion.
Check job status and confirm `.status.failedIndexes="1"`

```sh
kubectl get jobs -ljob-name=job-longrun -oyaml
```
Also, notice that `.status.active=2`, because the pod for a failed index is not
re-created.

This demonstrates that the feature is working again for the job.

<!--
Describe manual testing that was done and the outcomes.
Longer term, we may want to require automated upgrade/rollback tests, but we
are missing a bunch of machinery and tooling and can't do that now.
-->

###### Is the rollout accompanied by any deprecations and/or removals of features, APIs, fields of API types, flags, etc.?

No.

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

By the presence of the `.spec.backoffLimitPerIndex` field in the jobs.

For Beta we are also considering to introduce `job_finished_indexes_total`
metric
(see also [here](#are-there-any-missing-metrics-that-would-be-useful-to-have-to-improve-observability-of-this-feature)).

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

- [x] Job API .status
  - field: `failedIndexes` will not be empty as indexes fail
- [x] Pod API
  - annotation: `batch.kubernetes.io/job-index-failure-count` is present for
    pods created by Jobs with this feature enabled

###### What are the reasonable SLOs (Service Level Objectives) for the enhancement?

This feature does not propose SLOs.

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

- [x] Metrics
  - Metric name:
    - `job_sync_duration_seconds` (existing): can be used to see how much the
feature enablement increases the time spent in the sync job
    - `job_finished_indexes_total` (new): can be used to determine if the indexes
are marked failed,
  - Components exposing the metric: kube-controller-manager

###### Are there any missing metrics that would be useful to have to improve observability of this feature?

For Beta we will introduce a new metric `job_finished_indexes_total`
with labels `status=(failed|succeeded)`, and `backoffLimit=(perIndex|global)`.
It will count the number of failed and succeeded indexes across jobs using
`backoffLimitPerIndex`, or regular Indexed Jobs (using only `.spec.backoffLimit`).
It might be useful to determine the global ratio of failed vs. succeeded indexes
when `backoffLimitPerIndex` is used.

<!--
Describe the metrics themselves and the reasons why they weren't added (e.g., cost,
implementation difficulties, etc.).
-->

### Dependencies

<!--
This section must be completed when targeting beta to a release.
-->

###### Does this feature depend on any specific services running in the cluster?

No.

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

No.

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

No.

<!--
Describe them, providing:
  - API type
  - Supported number of objects per cluster
  - Supported number of objects per namespace (for namespace-scoped objects)
-->

###### Will enabling / using this feature result in any new calls to the cloud provider?

No.

<!--
Describe them, providing:
  - Which API(s):
  - Estimated increase:
-->

###### Will enabling / using this feature result in increasing size or count of the existing API objects?

  Yes, but only when the `.spec.backoffLimitPerIndex` field is set.

  - API type(s): Job
  - Estimated increase in size:
    - New `.status.failedIndexes` field in Status and `.status.completedIndexes`
pre-existing field are impacted. When the scalability limits are respected,
then the maximal increase of the total size of both fields can be estimated
as `190Ki` (see [The Job object too big](#the-job-object-too-big) for more details),
    - New `.spec.backoffLimitPerIndex` field of `*int32` is 12 bytes.

  - API type(s): Pod
  - Estimated increase in size:
    the new annotation `batch.kubernetes.io/job-index-failure-count` to keep the
    current number of retries per index. Is around 50 bytes.


<!--
Describe them, providing:
  - API type(s):
  - Estimated increase in size: (e.g., new annotation of size 32B)
  - Estimated amount of new objects: (e.g., new Object X for every existing Pod)
-->

###### Will enabling / using this feature result in increasing time taken by any operations covered by existing SLIs/SLOs?

We don't expect this increase to be captured by existing
[SLO/SLIs](https://github.com/kubernetes/community/blob/master/sig-scalability/slos/slos.md).

<!--
Look at the [existing SLIs/SLOs].

Think about adding additional work or introducing new steps in between
(e.g. need to do X to start a container), etc. Please describe the details.

[existing SLIs/SLOs]: https://git.k8s.io/community/sig-scalability/slos/slos.md#kubernetes-slisslos
-->

###### Will enabling / using this feature result in non-negligible increase of resource usage (CPU, RAM, disk, IO, ...) in any components?

The added dependency of removing finalizers only after pod
recreation [Tracking the number of failures per index](#tracking-the-number-of-failures-per-index)
may keep pods around longer (around 10s which is the backoff for pod recreation)
before actual deletion (requested or by PodGC).

This can increase the RAM consumption, but only for a short period of time. Also,
it is only affecting the failing pods.

<!--
Things to keep in mind include: additional in-memory state, additional
non-trivial computations, excessive access to disks (including increased log
volume), significant amount of data sent and/or received over network, etc.
This through this both in small and large cases, again with respect to the
[supported limits].

[supported limits]: https://git.k8s.io/community//sig-scalability/configs-and-limits/thresholds.md
-->

###### Can enabling / using this feature result in resource exhaustion of some node resources (PIDs, sockets, inodes, etc.)?

No. This feature does not introduce any resource exhaustive operations.

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

No change from existing behavior of the Job controller.

###### What are other known failure modes?

None.

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

N/A.

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

- 2023-01-23: Initial version of the KEP PR [Backoff Limit Per Job #3774](https://github.com/kubernetes/enhancements/pull/3774)
- 2023-04-26: The KEP PR [Backoff limit per Job Index #3967](https://github.com/kubernetes/enhancements/pull/3967) takes over from [#3774](https://github.com/kubernetes/enhancements/pull/3774)
- 2023-05-08: The KEP PR ready for review
- 2023-06-07: The KEP PR merged
- 2023-07-13: The implementation PR [Support BackoffLimitPerIndex in Jobs #118009](https://github.com/kubernetes/kubernetes/pull/118009) under review
- 2023-07-18: Merge the API PR [Extend the Job API for BackoffLimitPerIndex](https://github.com/kubernetes/kubernetes/pull/119294)
- 2023-07-18: Merge the Job Controller PR [Support BackoffLimitPerIndex in Jobs](https://github.com/kubernetes/kubernetes/pull/118009)
- 2023-08-04: Merge user-facing docs PR [Docs update for Job's backoff limit per index (alpha in 1.28)](https://github.com/kubernetes/website/pull/41921)
- 2023-08-06: Merge KEP update reflecting decisions during the implementation phase [Update for KEP3850 "Backoff Limit Per Index"](https://github.com/kubernetes/enhancements/pull/4123)

## Drawbacks

<!--
Why should this KEP _not_ be implemented?
-->

## Alternatives

### backoffLimitPerIndex inside new runPolicy

We could nest the new fields (`maxFailedIndexes` and `backoffLimitPerIndex`) inside
another field. Proposed alternative names for the field:
1. `runPolicy`
2. `completionPolicy`
3. `failurePolicy`

For example:
```yaml
apiVersion: v1
kind: Job
spec:
  parallelism: 10
  completions: 10
  completionMode: Indexed
  backoffLimit: 4
  runPolicy:
    backoffLimitPerIndex: true
    maxFailedIndexes: 1
  ...
```

The option (3.) suggests that the fields are about declaring the Job as failed.
However, the `backoffLimitPerIndex` field not only allows to count failures
towards the backoff limit per index, but also allows all indexes to execute
despite failures, thus more generic names, like (1.) and (2.) are preferred.

Also the options (1.) and (2.) may be reused in the context of success policy
which is subject of
[Job success/completion policy](https://github.com/kubernetes/kubernetes/issues/117600).
It might be beneficial for the API to consider the conditions for the Job
success or failure under the same field.

**Reasons for deferring / rejecting**

It is not clear what is the best name going forward. Also, it seems that the
`backoffLimitPerIndex` should be next to `backoffLimit`. It was discussed
and the consensus is that "top-level" is fine
(see [here](https://github.com/kubernetes/enhancements/pull/3967#discussion_r1196170192)).

### Mark Job Complete if some indexes failed

The alternative to the proposed [Job completion](#job-completion) strategy.

Allow execution of all indexes, up to `.spec.maxFailedIndexes` of
failed indexes. Then, mark the Job `Complete` even if some indexes failed.
The Job is marked `Failed` only if the number of failed indexes exceeds the
specified `.spec.maxFailedIndexes` limit, in that case, the `reason`
field could be `FailedIndexes`, and the `message` field would list the failed
indexes up to a couple of them.

**Reasons for deferring / rejecting**

This approach is less intuitive to the end-users of the API, compared
to the proposal. In particular, in some cases it would require custom logic in
the user's controller to determine if the Job is failed.

### Support backoffLimitPerIndex when restartPolicy=OnFailure

We've considered supporting the backoffLimitPerIndex when pod's `restartPolicy=OnFailure`.

**Reasons for deferring / rejecting**

When restartPolicy=OnFailure it is Kubelet's responsibility to restart the pod.
On the other hand if the maximal number of restarts would be enforced by the
Job controller, then race conditions are possible. For example, in-between the
checks by the Job controller, Kubelet execute more restarts than the specified
`.spec.backoffLimit`. The problematic counting of failures in the
restartPolicy=OnFailure has been ticketed
[When restartPolicy=OnFailure the calculation for number of retries is not accurate](https://github.com/kubernetes/kubernetes/issues/109870).

We believe that this feature can be supported well by using the pod-level API,
started in this KEP:
[Add a new field maxRestartTimes to podSpec when running into RestartPolicyOnFailure](https://github.com/kubernetes/enhancements/issues/3322).

Once the pod-level API is done, it could be considered to support `.spec.backoffLimitPerIndex`
when`restartPolicy=OnFailure` in pod's spec. In this case we could set the pod-level
`maxRestartTimes` field based on the Job-level `.spec.backoffLimit`, leaving the
responsibility of enforcing the limit to the Kubelet.

We will re-assess the decision of the Pod-level API graduates to GA in the
KEP: [Add a new field maxRestartTimes to podSpec when running into RestartPolicyOnFailure](https://github.com/kubernetes/enhancements/issues/3322).
For example, when maxRestartTimes is specified for `restartPolicy=OnFailure`, then
we could support `maxFailedIndexes` which would allow to control the number of
failed indexes (that exceeded the `maxRestartTimes` and are marked failed).

### Mutually exclusive backoffLimit and backoffLimitPerIndex

We've also considered to make the `backoffLimit` and `backoffLimitPerIndex`
fields mutually exclusive.

**Reasons for deferring / rejecting**

There is no way to control downgrade, as the value of `backoffLimit` would
always default to 6. Also, old API clients may error trying to read or modify
Job objects with backoffLimit=nil.

### Use bool field

We've considered to use a bool `backoffLimitPerIndex` field. Here is an example:


```yaml
apiVersion: v1
kind: Job
spec:
  parallelism: 10
  completions: 10
  completionMode: Indexed
  backoffLimit: 1
  backoffLimitPerIndex: true
  ...
```

**Reasons for deferring / rejecting**

It does not allow to specify both `.spec.backoffLimit` and `.spec.backoffLimitPerIndex`
in the same config. While setting both fields can be confusing in regular use
it can be helpful to support the use case of controlled downgrade.

### Use enum field

We've considered to use an enum `backoffLimitTarget: Job|Index` field (another
name for this concept could be `backoffLimitGranularity`), to specify that the
failures should be tracked per-index. Here, the default would be `Job`. Here is
an example:


```yaml
apiVersion: v1
kind: Job
spec:
  parallelism: 10
  completions: 10
  completionMode: Indexed
  backoffLimit: 1
  backoffLimitTarget: Index
  ...
```

**Reasons for deferring / rejecting**

No other targets, than `Job` and `Index`, will be added in a foreseeable
future. Thus, it seems like an unnecessary complication. The dedicated name
`backoffLimitPerIndex` seems to also better reflect the user's intention.

Similarly as in the bool case field [Use bool field](#use-bool-field) it does
not allow to set both `.spec.backoffLimit` and `.spec.backoffLimitPerIndex`
to control the downgrade.

<!--
What other approaches did you consider, and why did you rule them out? These do
not need to be as detailed as the proposal, but should include enough
information to express the idea and why it was not acceptable.
-->

### Global expotential backoff delay

We could also consider leaving the expotential backoff delay as global and
be enabled by a dedicated API field in the future KEP, say `backoffDelayPerIndex`.

**Reasons for deferring / rejecting**

The idea of using `backoffLimitPerIndex` is to make the indexes independent.
Thus, failures or successes in one index should not influence backoff delays
for another index. We are leaving the decision to the community feeback and
discussions though.

### Expotential backoff delay with in-memory tracking

Instead of modifying the definition of pod's finish time (see [Expotential backoff delay issue](#expotential-backoff-delay-issue))
we could keep track of the "failure time" for failed pods in-memory.

**Reasons for deferring / rejecting**

As the number of failed indexes is capped at 10^5 keeping track of failure
times for all pods will be at least 8B per failed pod, which is around 1Mi per
Job in the worst-case scenario. This is a non-negligible memory increase.

The extra tracking information is not needed counting pods as terminated is done
in [KEP-3939: Consider terminating pods in job controller](https://github.com/kubernetes/enhancements/pull/3940).
In this case we can assume that the failure time of each pod does not change
after its phase is terminal.

### Alternative ways to support high number of completions

In the current proposal the high number of completions (like 10^6) is supported
by specifying the `.spec.maxFailedIndexes` field. This way the size
of the `failedIndexes` field is controlled.

See below for alternative approaches proposed.

#### Keep failedIndexes field as a bitmap

In order to squeeze more failed indexes we could use bitmap.

**Reasons for deferring / rejecting**

- it is not human readable which might be useful for manual inspection
- it is harder to parse by user-provided controllers
- it introduces another format to keeping the succeeded indexes in `.status.completedIndexes`

#### Keep the list of failed indexes in a dedicated API object

The idea is to keep the heavy fields outside of the Job API object itself.
It could be a new API object, for example JobFailedIndexes.

**Reasons for deferring / rejecting**

This approach significantly increases the complexity of the Job controller that
needs to register and manage another API object. This may also have performance
impact as the Job controller needs to query the object. Finally, it is also
a complication to the end users who want to fetch the list of failed indexes.

#### Implicit limit on the number of failed indexes

An alternative is to have an implicit limit on the number of failed indexes, for
example, by controlling the size of the `.status.failedIndexes` field down to
300KB. This can allow to run a job with completions at the level of 10^6, without
explicit limit for maximal number of failed indexes.

**Reasons for deferring / rejecting**

It may behave unpredictably, impacting the user experience. For example,
when a user sets `maxFailedIndexes` as 10^6 the Job may complete if the indexes
and consecutive, but the Job may also fail if the size of the object exceeds the
limits due to non-consecutive indexes failing.

### Skip uncountedTerminatedPods when backoffLimitPerIndex is used

It's been proposed (see [link](https://github.com/kubernetes/kubernetes/pull/118009#discussion_r1263879848))
that when backoffLimitPerIndex is used, then we could skip the interim step of
recording terminated pods in `.status.uncountedTerminatedPods`.

**Reasons for deferring / rejecting**

First, if we stop using `.status.uncountedTerminatedPods` it means that
`.status.failed` can no longer track the number of failed pods. Thus, it would
require a change of semantic to denote just the number of failed indexes. This
has downsides:
- two different semantics of the field, depending on the used feature
- lost information about some failed pods within an index (some users may care
to investigate succeeded indexes with at least one failed pod)

Second, it would only optimize the unhappy path, where there are failures. Also,
the saving is only 1 request per 500 failed pods, which does not seem essential.


## Infrastructure Needed (Optional)

<!--
Use this section if you need things from the project/SIG. Examples include a
new subproject, repos requested, or GitHub details. Listing these here allows a
SIG to get the process for these resources started right away.
-->
