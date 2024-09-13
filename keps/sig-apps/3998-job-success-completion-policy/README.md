# KEP-3998: Job success/completion policy

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
  - [Notes/Constraints/Caveats (Optional)](#notesconstraintscaveats-optional)
    - [No support JobSuccessPolicy for the NonIndexed Job](#no-support-jobsuccesspolicy-for-the-nonindexed-job)
    - [Difference between &quot;Complete&quot; and &quot;SuccessCriteriaMet&quot;](#difference-between-complete-and-successcriteriamet)
    - [The CronJob concurrentPolicy is not affected by JobSuccessPolicy](#the-cronjob-concurrentpolicy-is-not-affected-by-jobsuccesspolicy)
    - [Status never switches from &quot;SuccessCriteriaMet&quot; to &quot;Failed&quot;](#status-never-switches-from-successcriteriamet-to-failed)
    - [The scope of the SuccessCriteriaMet condition](#the-scope-of-the-successcriteriamet-condition)
  - [Risks and Mitigations](#risks-and-mitigations)
- [Design Details](#design-details)
  - [Job API](#job-api)
  - [Evaluation](#evaluation)
  - [Transition of &quot;status.conditions&quot;](#transition-of-statusconditions)
    - [The situations where successPolicy conflicts other terminating policies](#the-situations-where-successpolicy-conflicts-other-terminating-policies)
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
  - [Relax a validation for the &quot;completions&quot; of the indexed job](#relax-a-validation-for-the-completions-of-the-indexed-job)
  - [Alternative API Name, &quot;Criteria&quot;](#alternative-api-name-criteria)
  - [Hold succeededIndexes as []int typed in successPolicy](#hold-succeededindexes-as-int-typed-in-successpolicy)
  - [Acceptable percentage of total succeeded indexes in the succeededCount field](#acceptable-percentage-of-total-succeeded-indexes-in-the-succeededcount-field)
  - [Match succeededIndexes using CEL](#match-succeededindexes-using-cel)
  - [Use JobSet instead of Indexed Job](#use-jobset-instead-of-indexed-job)
  - [Possibility for the lingering pods to continue running after the job meets the successPolicy](#possibility-for-the-lingering-pods-to-continue-running-after-the-job-meets-the-successpolicy)
    - [Additional Story](#additional-story)
    - [Job API](#job-api-1)
    - [Evaluation](#evaluation-1)
    - [Transition of &quot;status.conditions&quot;](#transition-of-statusconditions-1)
  - [Possibility for introducing a new CronJob concurrentPolicy, &quot;ForbidUntilJobSuccessful&quot;](#possibility-for-introducing-a-new-cronjob-concurrentpolicy-forbiduntiljobsuccessful)
  - [Possibility for the configurable reason for the &quot;SuccessCriteriaMet&quot; condition](#possibility-for-the-configurable-reason-for-the-successcriteriamet-condition)
    - [Additional Story](#additional-story-1)
    - [Job API](#job-api-2)
      - [Set the entire reason](#set-the-entire-reason)
      - [Set the suffix of the reason](#set-the-suffix-of-the-reason)
<!-- /toc -->

## Release Signoff Checklist

Items marked with (R) are required *prior to targeting to a milestone / release*.

- [x] (R) Enhancement issue in release milestone, which links to KEP dir in [kubernetes/enhancements] (not the initial KEP PR)
- [x] (R) KEP approvers have approved the KEP status as `implementable`
- [x] (R) Design details are appropriately documented
- [x] (R) Test plan is in place, giving consideration to SIG Architecture and SIG Testing input (including test refactors)
  - [x] e2e Tests for all Beta API Operations (endpoints)
  - [ ] (R) Ensure GA e2e tests meet requirements for [Conformance Tests](https://github.com/kubernetes/community/blob/master/contributors/devel/sig-architecture/conformance-tests.md) 
  - [ ] (R) Minimum Two Week Window for GA e2e tests to prove flake free
- [x] (R) Graduation criteria is in place
  - [ ] (R) [all GA Endpoints](https://github.com/kubernetes/community/pull/1806) must be hit by [Conformance Tests](https://github.com/kubernetes/community/blob/master/contributors/devel/sig-architecture/conformance-tests.md) 
- [ ] (R) Production readiness review completed
- [ ] (R) Production readiness review approved
- [x] "Implementation History" section is up-to-date for milestone
- [x] User-facing documentation has been created in [kubernetes/website], for publication to [kubernetes.io]
- [x] Supporting documentation—e.g., additional design documents, links to mailing list discussions/SIG meetings, relevant PRs/issues, release notes

[kubernetes.io]: https://kubernetes.io/
[kubernetes/enhancements]: https://git.k8s.io/enhancements
[kubernetes/kubernetes]: https://git.k8s.io/kubernetes
[kubernetes/website]: https://git.k8s.io/website

## Summary

This KEP extends the Job API to allow setting conditions under which an Indexed Job can be declared as succeeded.

## Motivation

There are cases where a batch workload requires an indexed job that want to care 
only about leader indexes in determining the success or failure of a Job,
for example MPI and PyTorch etc. This is currently not possible because the indexed job 
is marked as Completed only if all indexes succeeded.

Some third-party frameworks have implemented success policy.

- [Kubeflow Training Operator](https://www.kubeflow.org/docs/components/training)
- [Flux Operator](https://flux-framework.org/flux-operator/)
- [JobSet](https://github.com/kubernetes-sigs/jobset/)

### Goals

- Allow to mark a job as a succeeded according to a declared policy.
- Once the job meets the successPolicy, the lingering pods are terminated.

### Non-Goals

- Change the existing behavior of Jobs without a SuccessPolicy.
- Support SuccessPolicy for the job with `NonIndexed` mode: The SuccessPolicy can be theoretically supported for the job with `NonIndexed` mode. 
However, we don't work on the job with `NonIndexed` mode in the first iteration since there aren't any effective use cases for the NonIndexed job.

## Proposal

We propose new policies under which a job can be declared as succeeded.
Those policies can be modeled in the following:

1. An indexed job completes if a set of [x, y, z...] indexes are successful.
2. An indexed job completes if x of indexes are successful.

Then, when the job meets one of the success policies, a new condition, `SuccessCriteriaMet,` is added.

### User Stories (Optional)

#### Story 1

As a machine-learning researcher, I run an indexed job which a leader is launched
as index=0 and workers are launched as index=1+.
I want to care about only the leaders when the result of job is evaluated.

In addition, we want to terminate the lingering pods if the leader index (index=0) is succeeded
because the workers often don't have any ways to terminate themselves due to launching daemon processes like ssh-server.

```yaml
apiVersion: batch/v1
kind: Job
spec:
  parallelism: 10
  completions: 10
  completionMode: Indexed
  successPolicy:
    rules:
    - succeededIndexes: "0"
  template:
    spec:
      restartPolicy: Never
      containers:
      - name: job-container
        image: job-image
        command: ["./sample"]
```

#### Story 2

As a simulation researcher/engineer for fluid dynamics/biochemistry.
I want to mark the job as successful and terminate the lingering pods
when the job meets the one of following conditions:

1. The case of the leader index (index=0) is succeeded
2. The case of some worker indexes (index=1+) are succeeded

Because succeeded leader index means that the whole simulation is succeeded,
and succeeded some worker indexes means that the minimum required value is satisfied.

```yaml
apiVersion: batch/v1
kind: Job
spec:
  parallelism: 10
  completions: 10
  completionMode: Indexed
  successPolicy:
    rules:
    - succeededIndexes: 0
    - succeededCount: 5
      succeededIndexes: "1-9"
  template:
    spec:
      restartPolicy: Never
      containers:
        - name: job-container
          image: job-image
          command: ["./sample"]
```

### Notes/Constraints/Caveats (Optional)

#### No support JobSuccessPolicy for the NonIndexed Job

As I described in [Non-Goals](#non-goals), we don't support the SuccessPolicy for the job with `NonIndexed` mode.

#### Difference between "Complete" and "SuccessCriteriaMet"
The similar job conditions, `Complete` and `SuccessCriteriaMet`, are different in the following ways:

- `Complete` means that all pods completed and either all of them were successful or the Job already had `SuccessCriteriaMet=true`.
- `SuccessCriteriaMet` means that the job meets at least one of successPolicies.

So, the job could have both conditions, `Complete` and `SuccessCriteriaMet`.

#### The CronJob concurrentPolicy is not affected by JobSuccessPolicy
Even after introducing the JobSuccessPolicy, all CronJob concurrentPolicies work as before
since the JobSuccessPolicy doesn't change the semantics of the existing Job `Complete` condition 
and the Job declares succeeded by a new condition, `SuccessCriteriaMet`.

Specifically, the CronJob with `Forbid` concurrentPolicy created Jobs based on Job's `Complete` condition as before.

#### Status never switches from "SuccessCriteriaMet" to "Failed"
Switching the status from `SuccessCriteriaMet` to `Failed` would bring the confusions to the systems,
which depends on the Job API.

So, the status can never switch from `SucessCriteriaMet` to `Failed`.
Additionally, once the job has `SuccessCriteriaMet=true` condition, the job definitely ends with `Complete=true` condition
even if the lingering pods could potentially meet the failure policies.

#### The scope of the SuccessCriteriaMet condition

As part of this KEP we introduced the `SuccessCriteriaMet` condition scoped to
the success policy.

However, we are going to extend the scope of the condition to the scenario when
the Job completes by reaching the `.spec.completions`, as part of fixing
(issue #123775)[https://github.com/kubernetes/kubernetes/issues/123775].

See more details in the
[Job API managed-by mechanism](https://github.com/kubernetes/enhancements/blob/master/keps/sig-apps/4368-support-managed-by-for-batch-jobs/README.md).

### Risks and Mitigations

- If the job object's size reaches to limit of the etcd and
the job controller can't store a correct value in `.status.completedIndexes`,
we probably can not evaluate the SuccessPolicy correctly.

- If we allow to set unlimited size of the value in `.spec.successPolicy.rules.succeededIndexes`,
we have a risk similar to [KEP-3850: Backoff Limits Per Index For Indexed Jobs](https://github.com/kubernetes/enhancements/tree/76dcd4f342cc0388feb085e685d4cc018ebe1dc9/keps/sig-apps/3850-backoff-limits-per-index-for-indexed-jobs#risks-and-mitigations).
So, we limit the size of `succeededIndexes` to 64KiB.

## Design Details

### Job API

We extend the Job API to set different policies by which a Job can be declared as succeeded.

```golang
type JobSpec struct {
	...
	// successPolicy specifies the policy when the Job can be declared as succeeded.
	// If empty, the default behavior applies - the Job is declared as succeeded
	// only when the number of succeeded pods equals to the completions.
	// When the field is specified, it must be immutable and works only for the Indexed Jobs.
	// Once the Job meets the SuccessPolicy, the lingering pods are terminated.
	//
	// This field  is alpha-level. To use this field, you must enable the
	// `JobSuccessPolicy` feature gate (disabled by default).
	// +optional
	SuccessPolicy *SuccessPolicy
}

// SuccessPolicy describes when a Job can be declared as succeeded based on the success of some indexes.
type SuccessPolicy struct {
	// rules represents the list of alternative rules for the declaring the Jobs
	// as successful before `.status.succeeded >= .spec.completions`. Once any of the rules are met,
	// the "SucceededCriteriaMet" condition is added, and the lingering pods are removed.
	// The terminal state for such a Job has the "Complete" condition.
	// Additionally, these rules are evaluated in order; Once the Job meets one of the rules,
	// other rules are ignored. At most 20 elements are allowed.
	// +listType=atomic
	Rules []SuccessPolicyRule
}

// SuccessPolicyRule describes a rule for declaring a Job as succeeded.
// Each rules must have at least one of the "succeededIndexes" or "succeededCount" specified.
type SuccessPolicyRule struct {
	// succeededIndexes specifies the set of indexes
	// which need to be contained in the actual set of the succeeded indexes for the Job.
	// The list of indexes must be within 0 to ".spec.completions-1" and
	// must not contain duplicates. At least one element is required.
	// The indexes are represented as intervals separated by commas.
	// The intervals can be a decimal integer or a pair of decimal integers separated by a hyphen.
	// The number are listed in represented by the first and last element of the series,
	// separated by a hyphen.
	// For example, if the completed indexes are 1, 3, 4, 5 and 7, they are
	// represented as "1,3-5,7".
	// When this field is null, this field doesn't default to any value
	// and is never evaluated at any time.
	//
	// +optional
	SucceededIndexes *string

	// succeededCount specifies the minimal required size of the actual set of the succeeded indexes
	// for the Job. When succeededCount is used along with succeededIndexes, the check is
	// constrained only to the set of indexes specified by succeededIndexes.
	// For example, given that succeededIndexes is "1-4", succeededCount is "3",
	// and completed indexes are "1", "3", and "5", the Job isn't declared as succeeded
	// because only "1" and "3" indexes are considered in that rules.
	// When this field is null, this doesn't default to any value and
	// is never evaluated at any time.
	// When specified it needs to be a positive integer.
	//
	// +optional
	SucceededCount *int32
}
...

// These are valid conditions of a job.
const (
	// JobSuccessCriteriaMet means the job has been succeeded.
	JobSucceessCriteriaMet JobConditionType = "SuccessCriteriaMet"
	...
)
```

Moreover, we validate the following constraints for the `rules` and `status.conditions`:
- `rules`
  - whether each criterion have at least one of the `succeededIndexes` or `succeededCount` specified.
  - whether the specified indexes in the `succeededIndexes` and
    the number of indexes in the `succeededCount` don't exceed the value of `completions`.
  - whether `Indexed` is specified in the `completionMode` field.
  - whether the size of `succeededIndexes` is under 64Ki.
  - whether the `succeededIndexes` field has a valid format.
  - whether the `succeededCount` field has an absolute number.
  - whether the rules haven't changed.
  - whether the successPolicies meet the `succeededCount <= |succeededIndexes|`, 
  where `|succeededIndexes|` means the number of indexes in the `succeededIndexes`.
- `status.conditions`
  - whether the `SuccessCriteriaMet` condition isn't removed when the Job is updated.
  - whether the `SuccessCriteriaMet` condition isn't added after the Job already has only `Complete` condition.
  - whether the `SuccessCriteriaMet` condition isn't added to NonIndexed Job.
  - whether the Job doesn't have both `Failed` and `SuccessCriteriaMet` conditions.
  - whether the Job doesn't have both `FailureTarget` and `SuccessCriteriaMet` conditions.
  - whether the Job without SuccessPolicy doesn't have `SuccessCriteriaMet` condition.
  - whether the Job with SuccessPolicy doesn't have only `Complete` condition. The Job with SuccessPolicy need to have both `SuccessCriteriaMet` and `Complete` conditions.

### Evaluation

Every time the pod condition are updated, the job-controller evaluates the successPolicies following the rules in order:

- `succeededIndexes`: the job-controller evaluates `.status.completedIndexes` to see if a set of indexes is there.
- `succeededCount`: the job-controller evaluates `.status.succeeded` to see if the value is `succeededCount` or more.

After that, the job-controller adds a `SuccessCriteriaMet` condition instead of a `Failed` condition to `.status.conditions`
and the job-controller terminates the lingering pods. At that time, `JobSuccessPolicy` is set to the `status.reason` field.

Note that when the job meets one of successPolicies, other successPolicies are ignored.

Finally, once all pods have terminated, the job-controller adds a `Complete` condition to `.status.conditions`.
If any successPolicy isn't set, the job-controller adds an only `Complete` condition to the Job after the Job finished.

Furthermore, the behavior of `FailureTarget` and `SuccessCriteriaMet` is similar in that the Job with this condition triggers the termination of lingering pods;
after all pods are terminated, the terminal condition (`Failed` or `Complete`) is added:

- `FailureTarget` is added to the Job matched with FailurePolicy with `action=FailJob` and triggers the termination of the lingering pods.
Then, after the lingering pods are terminated, the `Failed` condition is added to the Job.
- `SuccessCriteriaMet` is added to the Job matched with SuccessPolicy and triggers the termination of lingering pods. 
Then, after the lingering pods are terminated, the `Complete` condition is added to the Job.

### Transition of "status.conditions"

After extending the scope of the `SuccessCriteriaMet` and `FailureTarget` conditions
as proposed in [The scope of the SuccessCriteriaMet condition](#the-scope-of-the-successcriteriamet-condition)
the diagram of transitions looks like below:

```mermaid
stateDiagram-v2
    [*] --> Running
    Running --> FailureTarget: Exceeded backoffLimit
    Running --> FailureTarget: Exceeded activeDeadlineSeconds
    Running --> FailureTarget: Matched FailurePolicy with action=FailJob
    FailureTarget --> Failed: All pods are terminated
    Failed --> [*]
    Running --> SuccessCriteriaMet: Matched SuccessPolicy
    Running --> SuccessCriteriaMet: Achieved the expected completions
    SuccessCriteriaMet --> Complete: All pods are terminated
    Complete --> [*]
```

It means that the job's `.status.conditions` follows the following rules:  

- The job could have both `SuccessCriteriaMet=true` and `Complete=true` conditions.
- The job can't have both `Failed=true` and `SuccessCriteriaMet=true` conditions.
- The job can't have both `FailureTarget=true` and `SuccessCriteriaMet=true` conditions.
- The job can't have both `Failed=true` and `Complete=true` conditions.

#### The situations where successPolicy conflicts other terminating policies

The successPolicy has potential conflicts with other terminating policies such as the [pod failure policy](https://kubernetes.io/docs/tasks/job/pod-failure-policy/)
, backoffLimit, and [backoffLimitPerIndex](https://kubernetes.io/docs/concepts/workloads/controllers/job/#backoff-limit-per-index)
in the following situations:

- when the job meets the successPolicy and some pod failure policies with the `FailJob` action.
- when the job meets the successPolicy and the number of failed pods exceeds `backoffLimit`.
- when the job meets the successPolicy and the number of failed pods per indexes exceeds `backoffLimitPerIndex` in all indexes.

To avoid the above conflicts, terminating policies are evaluated the first before successPolicies.
This means that the terminating policies are respected rather than the successPolicies,
if the Job doesn't have the `FailureTarget` or `SuccessCriteriaMet` conditions yet.

### Test Plan

[x] I/we understand the owners of the involved components may require updates to
existing tests to make this code solid enough prior to committing the changes necessary
to implement this enhancement.

##### Prerequisite testing updates

##### Unit tests

- Test cases:
  - tests for Defaulting and Validating
  - verify whether a job has a SuccessCriteriaMet condition if the job meets to successPolicy and some indexes fail.
  - verify whether a job has both complete and SuccessCriteriaMet conditions 
  if the job meets to successPolicy and all pods are terminated
  - verify whether a job has a failed condition if the job meets to both successPolicy and terminating policies
  in the same reconcile cycle

- `k8s.io/kubernetes/pkg/controller/job`: `2 February 2024` - `91.5%`
- `k8s.io/kubernetes/pkg/apis/batch/validation`: `2 February 2024` - `98.0%`

##### Integration tests

- Test scenarios:
  - [enabling, disabling and re-enabling of the `JobSuccessPolicy` feature gate](https://github.com/kubernetes/kubernetes/blob/6346b9d1327c4b8be2398d9715bdae5475e27569/test/integration/job/job_test.go#L794)
  - [handling of successPolicy when all indexes succeeded](https://github.com/kubernetes/kubernetes/blob/6346b9d1327c4b8be2398d9715bdae5475e27569/test/integration/job/job_test.go#L539)
  - [handling of the `.spec.successPolicy.rules.succeededIndexes` when some indexes remain pending](https://github.com/kubernetes/kubernetes/blob/6346b9d1327c4b8be2398d9715bdae5475e27569/test/integration/job/job_test.go#L608)
  - [handling of the `.spec.successPolicy.rules.succeededCount` when some indexes remain pending](https://github.com/kubernetes/kubernetes/blob/6346b9d1327c4b8be2398d9715bdae5475e27569/test/integration/job/job_test.go#L653) 
  - [handling of successPolicy when some indexes of job with `backOffLimitPerIndex` fail](https://github.com/kubernetes/kubernetes/blob/6346b9d1327c4b8be2398d9715bdae5475e27569/test/integration/job/job_test.go#L698)

##### e2e tests

- Test scenarios:
  - handling of successPolicy when all indexes succeeded
  - handling of the `.spec.successPolicy.rules.succeededIndexes` when some indexes remain pending
  - handling of the `.spec.successPolicy.rules.succeededCount` when some indexes remain pending

### Graduation Criteria

#### Alpha

- Feature implemented behind the `JobSuccessPolicy` feature gate.
- Unit and integration tests passed as designed in [TestPlan](#test-plan).

#### Beta

- E2E tests passed as designed in [TestPlan](#test-plan).
- Introduced a new `job_succeeded_total` metric in [Monitoring Requirements](#monitoring-requirements).
- Feature is enabled by default.
- Address all issues reported by users.

#### GA

- No negative feedback.
- Verify reason for the Job's `Complete` condition in all e2e conformance tests
  (see [example](https://github.com/kubernetes/kubernetes/blob/e5dd48efd07e8a052604b3073e0fafe7361ca689/test/e2e/apps/job.go#L804))

### Upgrade / Downgrade Strategy

- Upgrade
  - If the feature gate is enabled, `JobSuccessPolicy` are allowed to use only.
  - If the feature gate is enabled without `JobSuccessPolicy`, 
  the default values will be applied to a job object.
  - Even if the feature gate is enabled, the Job controller doesn't update
  `.status.conditions` in already finished jobs.
- Downgrade
  - Previously configured values will be ignored, and the job will be marked
  as completed only when all indexes succeed.
  - the Job controller doesn't update `.status.conditions` in already finished jobs.

### Version Skew Strategy

The apiserver's version should be consistent with the kube-controller-manager version,
or this feature will not work.

This feature is limited to control plane.

Note that, the kube-apiserver can be in the N+1 skew version relative to the
kube-controller-manager as described [here](https://kubernetes.io/releases/version-skew-policy/#kube-controller-manager-kube-scheduler-and-cloud-controller-manager).
If it's enabled, jobs with SuccessPolicy set will have it respected. Otherwise, it will be ignored by the job controller.

## Production Readiness Review Questionnaire

### Feature Enablement and Rollback

###### How can this feature be enabled / disabled in a live cluster?

- [x] Feature gate (also fill in values in `kep.yaml`)
  - Feature gate name: JobSuccessPolicy
  - Components depending on the feature gate:
    - kube-apiserver
    - kube-controller-manager

###### Does enabling the feature change any default behavior?

No, the default behavior of a job and cronJob stays the same.
The newly added field is optional and has to be explicitly set by the user to use this new feature.

Regarding the CronJob, please see more details in [#The CronJob concurrentPolicy is not affected by JobSuccessPolicy](#the-cronjob-concurrentpolicy-is-not-affected-by-jobsuccesspolicy).

###### Can the feature be disabled once it has been enabled (i.e. can we roll back the enablement)?

Yes, we can disable the `JobSuccessPolicy` feature gate.
When the feature is disabled, the job controller stop evaluating the successPolicy even if
the `.spec.successPolicy` is set.

###### What happens if we reenable the feature if it was previously rolled back?

The Job controller considers the `.spec.successPolicy` when it updates `.status.conditions`
only for running Jobs and don't update `.status.conditions` for already finished jobs.

###### Are there any tests for feature enablement/disablement?

Yes, we added the "enablement -> disablement -> re-enablement" flow integration tests for the new APIs from the alpha stage 
[here](https://github.com/kubernetes/kubernetes/blob/6346b9d1327c4b8be2398d9715bdae5475e27569/test/integration/job/job_test.go#L794):

### Rollout, Upgrade and Rollback Planning

###### How can a rollout or rollback fail? Can it impact already running workloads?

Even if the kube-controller-manager is rolled out or rollback fail, already running workloads aren't any impact.
The default behavior will be applied to running workloads.

###### What specific metrics should inform a rollback?

An increase in the `job_sync_duration_seconds` metrics can mean that finished jobs
are taking longer to evaluate.

The users should check whether the completed jobs have the appropriate condition,
specifically the reason.

###### Were upgrade and rollback tested? Was the upgrade->downgrade->upgrade path tested?

In the alpha stage, the upgrade->downgrade->upgrade testing was added in the integration tests
[here](https://github.com/kubernetes/kubernetes/blob/6346b9d1327c4b8be2398d9715bdae5475e27569/test/integration/job/job_test.go#L794).

In terms of a manual test for the upgrade and rollback, we can use the v1.30.

The upgrade->downgrade->upgrade testing was done manually using the `alpha` version in v1.30 with the following steps:

1. Start the cluster with the `JobSuccessPolicy` feature gate enabled:

Create a KIND cluster with v1.30 and use the `Cluster` configuration below to turn this feature on. 

```shell
kind create cluster --config config.yaml
```

using `config.yaml`:

```yaml
kind: Cluster
apiVersion: kind.x-k8s.io/v1alpha4
featureGates:
  "JobSuccessPolicy": true
nodes:
  - role: control-plane
    image: kindest/node:v1.30.0
  - role: worker
    image: kindest/node:v1.30.0
```

Then, create the job using the `.spec.successPolicy.rules[0].succeededCount=1,succeedeIndexes=0`:

```shell
kubectl create -f job.yaml
```

using `job.yaml`:

```yaml
apiVersion: batch/v1
kind: Job
metadata:
  name: job-success-policy
spec:
  parallelism: 3
  completions: 3
  completionMode: Indexed
  successPolicy:
    rules:
    - succeededCount: 2
  template:
    spec:
      restartPolicy: Never
      containers:
      - name: main
        image: python:3.12
        command:
        - python3
        - -c
        - |
          import os, sys, time
          index = os.environ.get("JOB_COMPLETION_INDEX")
          sys.exit(0) if index == "0" else time.sleep(300)
          sys.exit(0) if index == "1" else time.sleep(3600)
        imagePullPolicy: IfNotPresent
```

Await for the pods to be running and the pod with index=0 to be succeeded.

2. Simulate downgrade by disabling the feature for api server and control-plane.

Then, await for the pod with index=1 to be succeeded and
verify that the pod with index=2 still running and the Job doesn't have `SuccessCriteriaMet`.

3. Simulate upgrade by enabling the feature for api server and control-plane.

Then, verify that the pod with index=2 is terminated and the Job has `SuccessCriteriaMet` and `Complete` conditions.

###### Is the rollout accompanied by any deprecations and/or removals of features, APIs, fields of API types, flags, etc.?

No.

### Monitoring Requirements

###### How can an operator determine if the feature is in use by workloads?

We will introduce the new `job_succeeded_total` metric with `JobSuccessPolicy` and `Completions` reasons,
which indicates the following situations: 

- `JobSuccessPolicy` indicates a job is declared as `SuccessCriteriaMet` because the job meets `spec.succesPolicy`.
- `Completions` indicates a job is declared as `SuccessCriteriaMet` because the job meets `spec.completions`.

###### How can someone using this feature know that it is working for their instance?

- [x] Job API .status
  - The Job controller will add a condition with `JobSuccessPolicy` reason to `conditions`.

###### What are the reasonable SLOs (Service Level Objectives) for the enhancement?

99% percentile over day for Job syncs is <= 15s for a client-side 50 QPS limit.

###### What are the SLIs (Service Level Indicators) an operator can use to determine the health of the service?

- [x] Metrics
  - Metric name: `job_sync_duration_seconds` (existing): can be used to see how much the
    feature enablement increases the time spent in the sync job
  - Components exposing the metric:

###### Are there any missing metrics that would be useful to have to improve observability of this feature?

No.

### Dependencies

###### Does this feature depend on any specific services running in the cluster?

No.

### Scalability

No.

###### Will enabling / using this feature result in any new API calls?

Yes, if the Job meets the SuccessPolicy,
the job-controller must make an additional API call to update the condition with `SuccessCriteriaMet`.

###### Will enabling / using this feature result in introducing new API types?

No.

###### Will enabling / using this feature result in any new calls to the cloud provider?

No.

###### Will enabling / using this feature result in increasing size or count of the existing API objects?

Yes, it will increase the size of existing API objects only when the `.spec.successPolicy` is set.

- API type(s): Job
- Estimated increase in size: `.spec.successPolicy.rules.succeededIndexes` field are impacted.
In the worst case, the size of `succeededIndexes` can be estimated about 64KiB (see [Risks and Mitigations](#risks-and-mitigations)).

###### Will enabling / using this feature result in increasing time taken by any operations covered by existing SLIs/SLOs?

No.

###### Will enabling / using this feature result in non-negligible increase of resource usage (CPU, RAM, disk, IO, ...) in any components?

Yes, the job-controller will consume more CPU and memory to compute the set of indexes from the `succeededIndexes`. 
Especially, if there are many of them (approaching the maximum size of them, 64KiB), 
the consumed resources might be non-negligible.

###### Can enabling / using this feature result in resource exhaustion of some node resources (PIDs, sockets, inodes, etc.)?

No.

### Troubleshooting

###### How does this feature react if the API server and/or etcd is unavailable?

The job controller will declare that the job is "Succeeded" based on the `.status.completedIndexes`.
So, in this case, the job controller can not correctly evaluate the successPolicy 
because the `.status.completedIndexes` won't be updated.

###### What are other known failure modes?

None.

###### What steps should be taken if SLOs are not being met to determine the problem?

If a successPolicy isn't respected even though the job doesn't match other policies 
such as a [pod failure policy](https://kubernetes.io/docs/tasks/job/pod-failure-policy/) and backoffLimit:

- Check reachability between Kubernetes components.
- Consider increasing the logging level to trace when the issues occur.
- Check the job controller's `job_sync_duration_seconds` metric to check if the job controller's processing duration increases. 

If many requests are rejected, re-queued many times or increased the job controller's processing duration,
consider tuning the parameters for [APF](https://kubernetes.io/docs/concepts/cluster-administration/flow-control/).

## Implementation History
 
- 2023.06.06: This KEP is ready for review.
- 2023.06.09: API design is updated.
- 2023.10.03: API design is updated.
- 2024.02.07: API is finalized for the alpha stage.
- 2024.03.09: "Criteria" is replaced with "Rules".
- 2024.06.11: Beta Graduation.

## Drawbacks

Adds more complexity to the criteria to be terminated Job.

## Alternatives

### Relax a validation for the "completions" of the indexed job
Currently, the indexed job is restricted `.spec.completion!=nil`.
By relaxing the validation, the indexed job can be declared as succeeded when some indexes succeeded,
similar to NonIndexed jobs.

### Alternative API Name, "Criteria"

The `criteria` would be matched to express the behavior of the successPolicy compared with `rules`.
But, we decided to adopt the API name, `rules` to keep consistency with the existing `podFailurePolicy.rules`.

```golang
// SuccessPolicy describes when a Job can be declared as succeeded based on the success of some indexes.
type SuccessPolicy struct {
	// Criteria represents the list of alternative criteria for declaring the jobs 
	// as successful before its completion. Once any of the criteria are met,
	// the "SuccessCriteriaMet" condition is added, and the lingering pods are removed.
	// The terminal state for such a job has the "Complete" condition.
	// Additionally, these criteria are evaluated in order; Once the job meets one of the criteria,
	// other criteria are ignored.
	//
	// +optional
	Criteria []SuccessPolicyCriteria
}

// SuccessPolicyCriteria describes a criteria for declaring a Job as succeeded.
// Each criteria must have at least one of the "succeededIndexes" or "succeededCount" specified.
type SuccessPolicyCriteria struct{
	...
}

...

// These are valid conditions of a job.
const (
	// JobSuccessCriteriaMet means the job has been succeeded.
	JobSucceessCriteriaMet JobConditionType = "SuccessCriteriaMet"
	...
)
```

### Hold succeededIndexes as []int typed in successPolicy

We can hold the `succeededIndexes` as []int typed that a job can be declared as succeeded.

```golang
// SuccessPolicyRule describes rule for declaring a Job as succeeded.
// Each rule must have at least one of the "succeededIndexes" or "succeededCount" specified.
type SuccessPolicyRule struct {
	// Specifies a set of required indexes.
	// The job is declared as succeeded if a set of indexes are succeeded. 
	// The list of indexes must come within 0 to `.spec.completions` and
	// must not contain duplicates. At least one element is required. 
	// 
	// +optional
	SucceededIndexes []int
...
}
```

However, if we allow users to set all `succeededIndexes` (`0-10^5`) to `.spec.successPolicy.creteria.succeededIndexes`
and don't limit the number of list sizes, there are cases that the size of a job object is too big.

In the worst case, allowed all indexes (`0-10^5`) are added to `succeededIndexes`, and a `succeededIndexes` size
is `SUM[9*10^n*(n+1)]+2+6≈5.6656MiB`, where:
- `n` starts from `0` and goes up to `5`.
- `1` of `n+1` means a separator that indicates `,`.
- `2` is the sum of the index `0` and `,`.
- `6` is the size of indexes `10^5`.

So, if we select this alternative API design, we need to limit the size of `succeededIndexes`. 

### Acceptable percentage of total succeeded indexes in the succeededCount field

We can accept a percentage of total succeeded indexes in the `succeededCount` field for the job with autoscaling semantics.
However, there is no effective use case for typical stories using elastic horovod or PyTorch elastic training,
as all pods must be completed.

```golang
// SuccessPolicyRule describes rule for declaring a Job as succeeded.
// Each rule must have at least one of the "succeededIndexes" or "succeededCount" specified.
type SuccessPolicyRule struct {
	...
	// Specifies the required number of indexes when .spec.completionMode =
	// "Indexed".
	// Value can be an absolute number (ex: 5) or an absolute percentage of total indexes
	// when the job can be declared as succeeded (ex: 50%).
	// The absolute number is calculated from the percentage by rounding up.
	// 
	// +optional
	SucceededCount *intstr.IntOrString
	...
}
```

### Match succeededIndexes using CEL

We can accept a set of required indexes represented by CEL in the `succeededIndexes` field. 
However, it is difficult to represent the set of indexes without regularity. 

```golang
// SuccessPolicyRule describes rule for declaring a Job as succeeded.
// Each rule must have at least one of the "succeededIndexes" or "succeededCount" specified.
type SuccessPolicyRule struct {
	...
	// Specifies a set of required indexes using CEL.
	// For example, if the completed indexes are only the last index, they are
	// represented as (job.completions -1).
	//
	// +optional
	SucceededIndexesMatchExpression *string
	...
}
```

### Use JobSet instead of Indexed Job

The [JobSet](https://github.com/kubernetes-sigs/jobset) is a custom resource for managing a group of Job as a unit.

Some of the stories are better served using JobSet.
Specifically, cases that make assumptions about what an index represents could be mapped as jobs in JobSet,
with names representing the semantics of those different groups of pods.

However, both Job level and JobSet level successPolicies would be valuable
since there are some cases in which we want to launch a Job by a single podTemplate.

### Possibility for the lingering pods to continue running after the job meets the successPolicy

There are cases where a batch workload can be declared as succeeded, and continue the lingering pods if a number of pods succeed.
So, it is possible to introduce a new field, `whenCriteriaAchieved` to make configurable the action for the lingering pods.

#### Additional Story

As a simulation researcher/engineer for fluid dynamics/biochemistry.
I want to mark the job as successful and continue the lingering pods if some indexes succeed
because I set the minimum required value for sampling in the `.successPolicy.rules.succeededCount` and
perform the same simulation in all indexes.

```yaml
apiVersion: batch/v1
kind: Job
spec:
  parallelism: 10
  completions: 10
  completionMode: Indexed
  successPolicy:
    rules:
    - succeededCount: 5
      succeededIndexes: "1-9"
      whenCriteriaAchieved: continue
  template:
    spec:
      restartPolicy: Never
      containers:
        - name: job-container
          image: job-image
          command: ["./sample"]
``` 

#### Job API

```golang
// SuccessPolicyRules describes a Job can be succeeded based on succeeded indexes.
type SuccessPolicyRule struct {
	...
	// Specifies the action to be taken on the not finished (successCriteriaMet or failed) pods 
	// when the job achieved the requirements.
	// Possible values are:
	// - Continue indicates that all pods wouldn't be terminated. 
	//   When the lingering pods failed, the pods would ignore the terminating policies (backoffLimit, 
	//   backoffLimitPerIndex, and podFailurePolicy, etc.) and the pods aren't re-created.
	// - ContinueWithRecreations indicates that all pods wouldn't be terminated.
	//   When the lingering pods failed, the pods would follow the terminating policies (backoffLimit, 
	//   backoffLimitPerIndex, and podFailurePolicy, etc.) and the pods are re-created.
	// - Terminate indicates that not finished pods would be terminated.
	//
	// Default value is Terminate.
	WhenCriteriaAchieved WhenCriteriaAchievedSuccessPolicy	
}

// WhenCriteriaAchievedSuccessPolicy specifies the action to be taken on the pods
// when the job achieved the requirements.
// +enum
type WhenCriteriaAchievedSuccessPolicy string

const (
	// All pods wouldn't be terminated when the job reached successPolicy.
	// When the lingering pods failed, the pods would ignore the terminating policies (backoffLimit,
	// backoffLimitPerIndex, and podFailurePolicy, etc.) and the pods aren't re-created.
	ContinueWhenCriteriaAchievedSuccessPolicy WhenCriteriaAchievedSuccessPolicy = "Continue"

	// All pods wouldn't be terminated when the job reached successPolicy.
	// When the lingering pods failed, the pods would follow the terminating policies (backoffLimit,
	// backoffLimitPerIndex, and podFailurePolicy, etc.) and the pods are re-created.
	ContinueWithRecreationsWhenCriteriaAchievedSuccessPolicy WhenCriteriaAchievedSuccessPolicy = "ContinueWithRecreations"

	// Not finished pods would be terminated when the job reached successPolicy.
	TerminateWhenCriteriaAchievedSuccessPolicy WhenCriteriaAchievedSuccessPolicy = "Terminate"
)
```

#### Evaluation

We need to have more discussions if we support the `continue` and `continueWithRecreatios` in the `whenCriteriaAchieved`.
We have main discussion points here:

1. After the job meets any successPolicy with `whenCriteriaAchieved=continue` and the job gets `SuccessCriteriaMet` condition,
   what we would expect to happen, when the lingering pods are failed.
   We may be able to select one of the actions in `a: Failed pods follow terminating policies like backoffLimit and podFailurePolicy`
   or `b: Failed pods are terminated immediately, and the terminating policies are ignored`.
   Moreover, as an alternative, we may be able to select the action `b` for the `whenCriteriaAchieved=continue`,
   and then we may be possible to introduce a new `whenCriteriaAchieved`, `continueWithRecreations`, for the action `a`.

- `terminate`: The current supported behavior. All pods would be terminated by the job controller.
- `continue`: This behavior isn't supported in the alpha stage.
  The lingering pods wouldn't be terminated when the job reached successPolicy.
  Additionally, when the lingering pods failed, the pods are re-created followed terminating policies.
- `continueWithRecreations`: This behavior isn't supported in the alpha stage.
  The lingering pods wouldn't be terminated when the job reached successPolicy.
  Additionally, when the lingering pods failed, the pods are re-created followed terminating policies.

#### Transition of "status.conditions"

When the job with `whenCriteriaAchieved=continue` is submitted, the job `status.conditions` transits in the following:
Note that the Job doesn't have an actual `Running` condition in the `status.conditions`.

```mermaid
stateDiagram-v2
    [*] --> Running
    Running --> Failed: Exceeded backoffLimit
    Running --> FailureTarget: Matched FailurePolicy with action=FailJob
    FailureTarget --> Failed: All pods are terminated
    Failed --> [*]
    Running --> SuccessCriteriaMet: Matched SuccessPolicy
    Running --> SuccessCriteriaMet: Matched SuccessPolicy
    SuccessCriteriaMet --> SuccessCriteriaMet: Wait for all pods are finalized
    SuccessCriteriaMet --> Complete: All pods are finished
    Complete --> [*]
```

### Possibility for introducing a new CronJob concurrentPolicy, "ForbidUntilJobSuccessful"

It is potentially possible to add a new CronJob concurrentPolicy, `ForbidUntilJobSuccessful`,
which the CronJob with `ForbidUntilJobSuccessful` creates Jobs based on Job's `SuccessCriteriaMet` condition.

```go
type ConcurrentPolicy string

const (
	...
	// ForbidUntilJobSuccessful means that the CronJob creates Jobs based on Job's SuccessCriteriaMet condition.
	ForbidUntilJobSuccessful ConcurrentPolicy = "ForbidUntilJobSuccessful"
)
```

### Possibility for the configurable reason for the "SuccessCriteriaMet" condition

It is possible to add a configurable reason for the "SuccessCriteriaMet" condition.
The machine-readable reason would be useful when the external programs like custom controllers implements the mechanism
so that the CustomJob API can change the actions based on the reason similar to 
the [PodFailrePolicyReason (KEP-4443)](https://github.com/kubernetes/enhancements/pull/4479).

#### Additional Story

As a developer of CustomJob API built with Job API like JobSet, I want to implement the reconcile logic so that
the controller can change the actions based on the reason why the job has been succeeded.

So, it should be included in the `reason` field instead of `message` field since the reason field should be machine-readable.

```yaml
apiVersion: batch/v1
kind: Job
spec:
  parallelism: 10
  completions: 10
  completionMode: Indexed
  successPolicy:
    rules:
    - succeededIndexes: "0"
      setSuccessCriteriaMetReason: "LeaderSucceeded"
    - succeededIndexes: "1-9"
      setSuccessCriteriaMetReason: "WorkersSucceeded"
  template:
    spec:
      restartPolicy: Never
      containers:
        - name: job-container
          image: job-image
          command: ["./sample"]
status:
  conditions:
  - type: "SuccessCriteriaMet"
    status: "True"
    reason: "LeaderSucceeded"
```

#### Job API

Selecting one of the following API designs is possible, but the first, `setSuccessCriteriaMetReason` was preferred
during the JobSuccessPolicy alpha stage discussions. Because the second, `SuccessCriteriaMetReasonSuffix` would decrease the machine-readability
since we need to parse the reason by the separation, `As`.

Furthermore, allowing the `reason` to have merging field responsibility wouldn't better and
decreasing the machine-readability would decrease the valuable that we have this reason in the `status.reason` field instead of `status.message` field.

##### Set the entire reason

```golang
// SuccessPolicyRule describes rule for declaring a Job as succeeded.
// Each rule must have at least one of the "succeededIndexes" or "succeededCount" specified.
type SuccessPolicyRule struct {
	// SetSuccessCriteriaMetReason specifies the CamelCase reason for the "SuccessCriteriaMet" condition.
	// Once the job meets this rule, the specified reason is set to the "status.reason".
	// The default value is "JobSuccessPolicy".
	//
	// +optional
	SetSuccessCriteriaMetReason *string
}
```

##### Set the suffix of the reason

```golang
// SuccessPolicyRule describes a rule for declaring a Job as succeeded.
// Each rule must have at least one of the "succeededIndexes" or "succeededCount" specified.
type SuccessPolicyRule struct {
	// SuccessCriteriaMetReasonSuffix specifies the CamelCase suffix of the reason for the "SuccessCriteriaMet" condition.
	// Once the job meets these rule, "JobSuccessPolicy" and the specified suffix is combined with "As".
	// For example, if specified suffix is "LeaderSucceeded", it is represented as "JobSuccessPolicyAsLeaderSucceeded".
	//
	// +optional
	SuccessCriteriaMetReasonSuffix *string
}
```
