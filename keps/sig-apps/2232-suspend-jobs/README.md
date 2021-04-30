<!--
**Note:** When your KEP is complete, all of these comment blocks should be removed.

To get started with this template:

- [x] **Pick a hosting SIG.**
  Make sure that the problem space is something the SIG is interested in taking
  up. KEPs should not be checked in without a sponsoring SIG.
- [ ] **Create an issue in kubernetes/enhancements**
  When filing an enhancement tracking issue, please make sure to complete all
  fields in that template. One of the fields asks for a link to the KEP. You
  can leave that blank until this KEP is filed, and then go back to the
  enhancement and add the link.
- [ ] **Make a copy of this template directory.**
  Copy this template into the owning SIG's directory and name it
  `NNNN-short-descriptive-title`, where `NNNN` is the issue number (with no
  leading-zero padding) assigned to your enhancement above.
- [ ] **Fill out as much of the kep.yaml file as you can.**
  At minimum, you should fill in the "Title", "Authors", "Owning-sig",
  "Status", and date-related fields.
- [ ] **Fill out this file as best you can.**
  At minimum, you should fill in the "Summary" and "Motivation" sections.
  These should be easy if you've preflighted the idea of the KEP with the
  appropriate SIG(s).
- [ ] **Create a PR for this KEP.**
  Assign it to people in the SIG who are sponsoring this process.
- [ ] **Merge early and iterate.**
  Avoid getting hung up on specific details and instead aim to get the goals of
  the KEP clarified and merged quickly. The best way to do this is to just
  start with the high-level sections and fill out details incrementally in
  subsequent PRs.

Just because a KEP is merged does not mean it is complete or approved. Any KEP
marked as `provisional` is a working document and subject to change. You can
denote sections that are under active debate as follows:

```
<<[UNRESOLVED optional short context or usernames ]>>
Stuff that is being argued.
<<[/UNRESOLVED]>>
```

When editing KEPS, aim for tightly-scoped, single-topic PRs to keep discussions
focused. If you disagree with what is already in a document, open a new PR
with suggested changes.

One KEP corresponds to one "feature" or "enhancement" for its whole lifecycle.
You do not need a new KEP to move from beta to GA, for example. If
new details emerge that belong in the KEP, edit the KEP. Once a feature has become
"implemented", major changes should get new KEPs.

The canonical place for the latest set of instructions (and the likely source
of this file) is [here](/keps/NNNN-kep-template/README.md).

**Note:** Any PRs to move a KEP to `implementable`, or significant changes once
it is marked `implementable`, must be approved by each of the KEP approvers.
If none of those approvers are still appropriate, then changes to that list
should be approved by the remaining approvers and/or the owning SIG (or
SIG Architecture for cross-cutting KEPs).
-->
# KEP-2322: Suspend Jobs

<!-- toc -->
- [Release Signoff Checklist](#release-signoff-checklist)
- [Summary](#summary)
- [Motivation](#motivation)
  - [Goals](#goals)
  - [Non-Goals](#non-goals)
- [Proposal](#proposal)
  - [User Stories](#user-stories)
    - [Story 1](#story-1)
    - [Story 2](#story-2)
  - [Notes/Constraints/Caveats](#notesconstraintscaveats)
  - [Risks and Mitigations](#risks-and-mitigations)
- [Design Details](#design-details)
  - [Test Plan](#test-plan)
  - [Graduation Criteria](#graduation-criteria)
    - [Alpha -&gt; Beta Graduation](#alpha---beta-graduation)
    - [Beta -&gt; GA Graduation](#beta---ga-graduation)
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
- [Infrastructure Needed](#infrastructure-needed)
<!-- /toc -->

## Release Signoff Checklist

Items marked with (R) are required *prior to targeting to a milestone / release*.

- [x] (R) Enhancement issue in release milestone, which links to KEP dir in [kubernetes/enhancements] (not the initial KEP PR)
- [x] (R) KEP approvers have approved the KEP status as `implementable`
- [x] (R) Design details are appropriately documented
- [x] (R) Test plan is in place, giving consideration to SIG Architecture and SIG Testing input
- [x] (R) Graduation criteria is in place
- [x] (R) Production readiness review completed
- [x] Production readiness review approved
- [x] "Implementation History" section is up-to-date for milestone
- [ ] User-facing documentation has been created in [kubernetes/website], for publication to [kubernetes.io]
- [ ] Supporting documentation—e.g., additional design documents, links to mailing list discussions/SIG meetings, relevant PRs/issues, release notes

[kubernetes.io]: https://kubernetes.io/
[kubernetes/enhancements]: https://git.k8s.io/enhancements
[kubernetes/kubernetes]: https://git.k8s.io/kubernetes
[kubernetes/website]: https://git.k8s.io/website

## Summary

The Jobs API allows users to create Pods using the Job controller with specific
requirements such as Pod-level parallelism, completing a certain number of
executions, and restart policies.

This KEP proposes an enhancement that will allow suspending and resuming Jobs.
Suspending a Job will delete all active Pods owned by that Job and also instruct
the controller manager to not create new Pods until the Job is resumed. Users
will also be able to create Jobs in the suspended state, thereby delaying Pod
creation indefinitely.

## Motivation

The Job controller tracks and manages all Jobs within Kubernetes. When a new
Job is created by the user, the Job controller will immediately begin creating
pods to satisfy the requirements of the Job until the Job is complete.

However, there are use-cases that require a Job to be suspended in the middle
of its execution. Currently, this is impossible; as a workaround, users can
delete and re-create the Job, but this will also delete Job metadata such as
the list of successful/failed Pods and logs. Therefore, it is desirable to
suspend Jobs and resume them later when convenient.

### Goals

- Allow suspending and resuming Jobs.
- Allow indefinitely delaying the creation of Pods owned by a Job.

### Non-Goals

- Being able to suspending and resume jobs makes Job preemption, all-or-nothing
  scheduling, and Job queueing possible. However, we don't propose to create
  such higher-level controllers as a part of this KEP.
- It might be useful to restrict who can set/modify the `suspend` flag.
  However, we don't propose to create the validating and/or mutating webhooks
  necessary to achieve that as a part of this KEP.

## Proposal

### User Stories

#### Story 1

Consider a cloud provider where servers are cheaper at night. My Job takes
several days to complete, so I'd like to suspend my Job early in the morning
every day and resume it after dark to save money. However, I don't want to
delete and re-create my Job every day as I don't want to lose track of
completed Pods, logs, and other metadata.

#### Story 2

Let's say I'm a system administrator and there are many users submitting Jobs
to my cluster. All user-submitted Jobs are created with `suspend: true`.
There's only a finite amount of resources, so I must resume these suspended
Jobs in the right order at the right time.

I can write a higher-level Job queueing controller to do this based on external
factors. For example, the controller could choose to simply unpause Jobs in the
FIFO order. Alternatively, Jobs could be assigned priority and, just like
kube-scheduler, the controller can make a decision based on the suspended Job
queue (it can even do Job preemption). Each Job could request a different
amount of resources, so the higher-level controller may also want to resize the
cluster to just fit the Job it's going to run. Regardless of what logic the
controller uses to queue jobs, being able to suspend Jobs indefinitely and then
resume them later is important.

### Notes/Constraints/Caveats

System administrators may want to restrict who can set/modify the `suspend`
field when creating or updating Jobs. This can be achieved with validating
and/or mutating webhooks.

When a Job is suspended, the time is recorded as part of the Job status. Users
can infer how long a Job has been suspended for through this field. This can be
useful when making decisions around which Job should be resumed.

A Job that is complete cannot be suspended.

### Risks and Mitigations

Suspending an active Job deletes *all* active pods. Users must design their
application to gracefully handle this.

## Design Details

We propose adding a `suspend` field to the `JobSpec` API:

```go
type JobSpec struct {
	// Suspend specifies whether the Job controller should create Pods or not. If
	// a Job is created with suspend set to true, no Pods are created by the Job
	// controller. If a Job is suspended after creation (i.e. the flag goes from
	// false to true), the Job controller will delete all active Pods associated
	// with this Job. Users must design their workload to gracefully handle this.
	// This is an alpha field and requires enabling the SuspendJob feature gate.
	// Defaults to false.
	// +optional
	Suspend *bool `json:"suspend,omitempty"`

	...
}
```

As described in the comment, when the boolean is set to true, the
controller-manager abstains from creating Pods even if there's work left to be
done. If the Job is already active and is updated with `suspend: true`, the Job
controller calls Delete on all its active Pods. This causes the kubelet to send
a SIGTERM signal and completely remove the Pod after its graceful termination
period is honoured. Pods terminated this way are considered a failure and the
controller does not count terminated Pods towards completions. This behaviour
is similar to decreasing the Job's parallelism to zero.

Similar to existing [JobConditionType](https://github.com/kubernetes/kubernetes/blob/c98f6bf30890f2c5826067ae50cfc36958106e68/staging/src/k8s.io/api/batch/v1/types.go#L167)s
"Complete" and "Failed", we propose adding a new condition type called
"Suspended" as a part of the Job's status as follows:

```go
// These are valid conditions of a job.
const (
	// JobSuspended means the job has been suspended.
	JobSuspended JobConditionType = "Suspended"
	// JobComplete means the job has completed its execution.
	JobComplete JobConditionType = "Complete"
	// JobFailed means the job has failed its execution.
	JobFailed JobConditionType = "Failed"
)
```

To determine if a Job has been suspended, users must look for the
`JobCondition` with `Type` as "Suspended". If such a `JobCondition` does not
exist or if the `Status` field is false, the Job is not suspended. Otherwise,
if the `Status` field is true, the Job is suspended. Note that when the
`suspend` flag in the Job spec is flipped from true to false, the Job
controller simply updates the existing suspend `JobCondition` status to false;
it does not remove the condition or add a new one.

Inferring suspension status from the Job spec's `suspend` field is not
recommended as the Job controller may not have seen the update yet. When a Job
is suspended, the Job controller sets the `JobCondition` only after all active
Pods owned by the Job are terminating or have been deleted. 

The suspend `JobCondition` also has a `LastTransitionTime` field. This can be
used to infer how long a Job has been suspended for (if `Status` is true).

The `StartTime` field of the Job status is reset to the current time every time
the Job is resumed from suspension. If a Job is created with `suspend: true`,
the `StartTime` field of the Job status is set only when it is resumed for the
first time.

If a Job is suspended (at creation or through an update), the `ActiveDeadlineSeconds`
timer will effectively be stopped and reset when the Job is resumed again. That
is, Jobs will never be terminated for exceeding `ActiveDeadlineSeconds` when a
Job is suspended. Users must interpret `ActiveDeadlineSeconds` as the duration
for which a Job can be *continuously* active before which it is terminated.

When a Job is suspended or created in the suspended state, a "Suspended" event
is recorded. Similarly, when a Job is resumed from its suspended state, a
"Resumed" event is recorded.

### Test Plan

Unit, integration, and end-to-end tests will be added to test that:

- Creating a Job with `suspend: true` should not create pods
- Suspending a Job should delete active pods
- Resuming a Job should re-create pods
- Jobs should remember completions count after a suspend-resume cycle

### Graduation Criteria

#### Alpha -> Beta Graduation
* Metrics with observability in to the Job controller available
* Implemented feedback from alpha testers

#### Beta -> GA Graduation
* We're confident that no further semantical changes will be needed to achieve the goals of the KEP
* All known functional bugs have been fixed

<!--
**Note:** *Not required until targeted at a release.*

Define graduation milestones.

These may be defined in terms of API maturity, or as something else. The KEP
should keep this high-level with a focus on what signals will be looked at to
determine graduation.

Consider the following in developing the graduation criteria for this enhancement:
- [Maturity levels (`alpha`, `beta`, `stable`)][maturity-levels]
- [Deprecation policy][deprecation-policy]

Clearly define what graduation means by either linking to the [API doc
definition](https://kubernetes.io/docs/concepts/overview/kubernetes-api/#api-versioning)
or by redefining what graduation means.

In general we try to use the same stages (alpha, beta, GA), regardless of how the
functionality is accessed.

[maturity-levels]: https://git.k8s.io/community/contributors/devel/sig-architecture/api_changes.md#alpha-beta-and-stable-versions
[deprecation-policy]: https://kubernetes.io/docs/reference/using-api/deprecation-policy/

Below are some examples to consider, in addition to the aforementioned [maturity levels][maturity-levels].

#### Alpha -> Beta Graduation

- Gather feedback from developers and surveys
- Complete features A, B, C
- Tests are in Testgrid and linked in KEP

#### Beta -> GA Graduation

- N examples of real-world usage
- N installs
- More rigorous forms of testing—e.g., downgrade tests and scalability tests
- Allowing time for feedback

**Note:** Generally we also wait at least two releases between beta and
GA/stable, because there's no opportunity for user feedback, or even bug reports,
in back-to-back releases.

#### Removing a Deprecated Flag

- Announce deprecation and support policy of the existing flag
- Two versions passed since introducing the functionality that deprecates the flag (to address version skew)
- Address feedback on usage/changed behavior, provided on GitHub issues
- Deprecate the flag

**For non-optional features moving to GA, the graduation criteria must include 
[conformance tests].**

[conformance tests]: https://git.k8s.io/community/contributors/devel/sig-architecture/conformance-tests.md
-->

### Upgrade / Downgrade Strategy

Upgrading from 1.20 and below will not change the behaviour of how Jobs work.

To make use of this feature, the `SuspendJob` feature gate must be explicitly
enabled on the API server and the controller manager and the `suspend` field
must be explicitly set in the Job spec.

### Version Skew Strategy

The change is entirely limited to the control plane. Version skew across
control plane / kubelet does not change anything.

In HA clusters, version skew across different replicas in the control plane
should also work seamlessly because only one controller manager will be active
at any given time.

## Production Readiness Review Questionnaire

### Feature Enablement and Rollback

* **How can this feature be enabled / disabled in a live cluster?**
  - [x] Feature gate (also fill in values in `kep.yaml`)
    - Feature gate name: `SuspendJob`
    - Components depending on the feature gate:
      - kube-apiserver
      - kube-controller-manager

* **Does enabling the feature change any default behavior?** 
No, using the feature requires explicitly opting in by setting the `suspend`
field.

* **Can the feature be disabled once it has been enabled (i.e. can we roll back
  the enablement)?** Yes. Turning off the feature gate will disable the
  feature. Once the feature gate is turned off, the default Job controller will
  ignore the `suspend` field in all jobs, so this will cause existing suspended
  jobs to be resumed indirectly when the controller manager is restarted.

* **What happens if we reenable the feature if it was previously rolled back?**
  Jobs that have the flag set will be suspended, and new jobs or updates to existing
  ones to the field will be persisted.

* **Are there any tests for feature enablement/disablement?** Yes. Integration
  tests have exhaustive testing switching between different feature enablement
  states whilst using the feature at the same time. Unit tests and end-to-end
  tests test feature enablement too.

### Rollout, Upgrade and Rollback Planning

_This section must be completed when targeting beta graduation to a release._

* **How can a rollout fail? Can it impact already running workloads?** Impact
  to existing Jobs that previously didn't use this feature in alpha is
  impossible. In workloads using the feature in an older version, suspended
  Jobs may inadvertently be resumed (or Jobs may be inadvertently suspended) if
  there are storage-related issues arising from components crashing
  mid-rollout.

* **What specific metrics should inform a rollback?** `job_sync_duration_seconds`
  and `job_sync_total` should be observed. Unexpected spikes in the metric with
  labels `result=error` and `action=pods_deleted` is potentially an indicator
  that:
    1. Job suspension is producing errors in the Job controller,
    1. Jobs are getting suspended when they shouldn't be, or
    1. Job sync latency is high when Job are suspended.
  While the above list isn't exhaustive, they're signals in favour of rollbacks.

* **Were upgrade and rollback tested? Was the upgrade->downgrade->upgrade path
  tested?** <!-- I'll answer this after implementation.
  Describe manual testing that was done and the outcomes.
  Longer term, we may want to require automated upgrade/rollback tests, but we
  are missing a bunch of machinery and tooling and can't do that now. -->

* **Is the rollout accompanied by any deprecations and/or removals of features,
  APIs, fields of API types, flags, etc.?** No.

### Monitoring Requirements

_This section must be completed when targeting beta graduation to a release._

* **How can an operator determine if the feature is in use by workloads?**
  The `.spec.suspend` field is set to true by Jobs. The status conditions of a
  Job can also be used to determine whether a Job is using the feature (look
  for a condition of type "Suspended").

* **What are the SLIs (Service Level Indicators) an operator can use to
  determine the health of the service?**
  - [x] Metrics
    - Metric name: The metrics `job_sync_duration_seconds` and `job_sync_total`
      get a new label named `action` to allow operators to filter Job sync
      latency and error rate, respectively, by the action performed. There are
      four mutually-exclusive values possible for this label:
        - `reconciling` when the Job's pod creation/deletion expectations are
          unsatisfied and the controller is waiting for issued Pod
          creation/deletions to complete.
        - `tracking` when the Job's pod creation/deletion expectations are
          satisfied and the number of active Pods matches expectations (i.e. no
          pod creation/deletions issued in this sync). This is expected to be
          the action in most of the syncs.
        - `pods_created` when the controller creates Pods. This can happen
          when the number of active Pods is less than the wanted Job
          parallelism.
        - `pods_deleted` when the controller deletes Pods. This can happen if a
          Job is suspended or if the number of active Pods is more than
          parallelism.
      Each sample of the two metrics will have exactly one of the above values
      for the `action` label.
    - Components exposing the metric:
      - kube-controller-manager

* **What are the reasonable SLOs (Service Level Objectives) for the above
  SLIs?**
  - per-day percentage of `job_sync_total` with labels `result=error` and
    `action=pods_deleted` <= 1%
  - 99% percentile over day for `job_sync_duration_seconds` with label
    `action=pods_deleted` is <= 15s, assuming a client-side QPS limit of 50
    calls per second

* **Are there any missing metrics that would be useful to have to improve
  observability of this feature?** No.

### Dependencies

_This section must be completed when targeting beta graduation to a release._

* **Does this feature depend on any specific services running in the cluster?**
  Feature is restricted to kube-apiserver and kube-controller-manager.

### Scalability

* **Will enabling / using this feature result in any new API calls?** Yes. when
  a job is suspended, all pods will be issued a DELETE request; similarly when
  a job is resumed, pods will be created again.

* **Will enabling / using this feature result in introducing new API types?**
  No.

* **Will enabling / using this feature result in any new calls to the cloud 
  provider?** No.

* **Will enabling / using this feature result in increasing size or count of 
  the existing API objects?** Each JobSpec object will increase by the size of
  a boolean. Each JobStatus may have an additional `JobCondition` entry.

* **Will enabling / using this feature result in increasing time taken by any 
  operations covered by [existing SLIs/SLOs]?** No.

* **Will enabling / using this feature result in non-negligible increase of 
  resource usage (CPU, RAM, disk, IO, ...) in any components?** No.

### Troubleshooting

The Troubleshooting section currently serves the `Playbook` role. We may consider
splitting it into a dedicated `Playbook` document (potentially with some monitoring
details). For now, we leave it here.

_This section must be completed when targeting beta graduation to a release._

* **How does this feature react if the API server and/or etcd is unavailable?**
  Updates to suspend or resume a Job will not work. The controller will not be
  able to create or delete Pods. Events, logs, and status conditions for Jobs
  will not be updated to reflect their suspended status.

* **What are other known failure modes?** None. The API server, etcd, and the
  controller manager are the only possible points of failure.

* **What steps should be taken if SLOs are not being met to determine the
  problem?**
  - Verify that kube-apiserver and etcd are healthy. If not, the Job controller
    cannot operate, so you must fix those problems first.
  - Verify that `job_sync_total` is unexpectedly high for `result=error` and
    `action=pods_deleted` in comparison to other actions.
  - Verify that `job_sync_duration_seconds` is noticeably larger for
    `action=pods_deleted` in comparison to the other actions.
  - If control plane components are starved for CPU, which could be a potential
    reason behind Job sync latency spikes, consider increasing the control
    plane's resources.

[supported limits]: https://git.k8s.io/community//sig-scalability/configs-and-limits/thresholds.md
[existing SLIs/SLOs]: https://git.k8s.io/community/sig-scalability/slos/slos.md#kubernetes-slisslos

## Implementation History

2021-02-01: Initial KEP merged, alpha targeted for 1.21
2021-03-08: Implementation merged in 1.21 with feature gate disabled by default
2021-04-22: KEP updated for beta graduation in 1.22

## Drawbacks

Alternative strategies to achieve something similar were explored (see KEP
issue for design details), so if one of the other less-preferred options were
chosen instead, this KEP should not be implemented.

## Alternatives

Instead of making this a native Kubernetes feature, one could use an external
controller to handle Jobs that need delayed Pod creation. This can be achieved
with an `orchestratorName` field that can tell the default Job controller to
ignore a Job entirely. While this approach is similar to the `schedulerName`
field used in kube-scheduler, it adds unnecessary complexity and the need for
additional control plane components to handle Jobs. In addition, this approach
makes ownership of the Job hard to track.

## Infrastructure Needed

None.
