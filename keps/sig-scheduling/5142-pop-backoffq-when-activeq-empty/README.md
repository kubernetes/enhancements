# KEP-5142: Pop pod from backoffQ when activeQ is empty

<!--
This is the title of your KEP. Keep it short, simple, and descriptive. A good
title can help communicate what the KEP is and should be considered as part of
any review.
-->

<!--
A table of contents is helpful for quickly jumping to sections of a KEP and for
highlighting any additional information provided beyond the standard KEP
template.

Ensure the TOC is wrapped with
  <code>&lt;!-- toc --&rt;&lt;!-- /toc --&rt;</code>
tags, and then generate with `hack/update-toc.sh`.
-->

<!-- toc -->
- [Release Signoff Checklist](#release-signoff-checklist)
- [Summary](#summary)
- [Motivation](#motivation)
  - [Goals](#goals)
  - [Non-Goals](#non-goals)
- [Proposal](#proposal)
  - [Risks and Mitigations](#risks-and-mitigations)
    - [A tiny delay on the first scheduling attempts for newly created pods](#a-tiny-delay-on-the-first-scheduling-attempts-for-newly-created-pods)
    - [Backoff won't be working as a natural rate limiter in case of errors](#backoff-wont-be-working-as-a-natural-rate-limiter-in-case-of-errors)
    - [One pod in the backoffQ could starve the others](#one-pod-in-the-backoffq-could-starve-the-others)
  - [Low priority pod could be chosen to pop, even if high priority pod has a slightly later backoff expiration](#low-priority-pod-could-be-chosen-to-pop-even-if-high-priority-pod-has-a-slightly-later-backoff-expiration)
- [Design Details](#design-details)
  - [Popping from the backoffQ in activeQ's pop()](#popping-from-the-backoffq-in-activeqs-pop)
  - [Notifying activeQ condition when a new pod appears in the backoffQ](#notifying-activeq-condition-when-a-new-pod-appears-in-the-backoffq)
  - [Calling PreEnqueue for the backoffQ](#calling-preenqueue-for-the-backoffq)
  - [Change backoffQ less function](#change-backoffq-less-function)
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
  - [Move pods in flushBackoffQCompleted when activeQ is empty](#move-pods-in-flushbackoffqcompleted-when-activeq-is-empty)
<!-- /toc -->

## Release Signoff Checklist

<!--
**ACTION REQUIRED:** In order to merge code into a release, there must be an
issue in [kubernetes/enhancements] referencing this KEP and targeting a release
milestone **before the [Enhancement Freeze](https://git.k8s.io/sig-release/releases)
of the targeted release**.

For enhancements that make changes to code or processes/procedures in core
Kubernetes—i.e., [kubernetes/kubernetes], we require the following Release
Signoff checklist to be completed.

Check these off as they are completed for the Release Team to track. These
checklist items _must_ be updated for the enhancement to be released.
-->

Items marked with (R) are required *prior to targeting to a milestone / release*.

- [x] (R) Enhancement issue in release milestone, which links to KEP dir in [kubernetes/enhancements] (not the initial KEP PR)
- [x] (R) KEP approvers have approved the KEP status as `implementable`
- [x] (R) Design details are appropriately documented
- [x] (R) Test plan is in place, giving consideration to SIG Architecture and SIG Testing input (including test refactors)
  - [ ] e2e Tests for all Beta API Operations (endpoints)
  - [ ] (R) Ensure GA e2e tests meet requirements for [Conformance Tests](https://github.com/kubernetes/community/blob/master/contributors/devel/sig-architecture/conformance-tests.md) 
  - [ ] (R) Minimum Two Week Window for GA e2e tests to prove flake free
- [ ] (R) Graduation criteria is in place
  - [ ] (R) [all GA Endpoints](https://github.com/kubernetes/community/pull/1806) must be hit by [Conformance Tests](https://github.com/kubernetes/community/blob/master/contributors/devel/sig-architecture/conformance-tests.md) 
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

This KEP proposes improving scheduling queue behavior by popping pods from the backoffQ when the activeQ is empty.
This would allow to process potentially schedulable pods ASAP, eliminating a penalty effect of the backoff queue.

## Motivation

There are three queues in the scheduler:
- activeQ contains pods ready for scheduling,
- unschedulableQ holds pods that were unschedulable in their scheduling cycle and are waiting for cluster state to change.
  These pods are then moved to backoffQ to apply a penalty.
- backoffQ stores pods that failed scheduling attempts (either due to being unschedulable or errors) and could be schedulable again,
  but applying a backoff penalty, scaled with the number of attempts.

When the activeQ is not empty, the scheduler pops the highest priority pod from the activeQ.
However, when the activeQ is empty, the kube-scheduler idles,
even if pods are in the backoffQ waiting for their backoff period to expire.
To avoid delaying assessment of potentially schedulable pods,
kube-scheduler should consider those pods for scheduling, even if the backoff time hasn't expired yet.
However, pods that are in the backoffQ due to errors, should not bypass the backoff time,
since it plays also rate limiting role, avoiding system overload due to too frequent retries.

### Goals

- Improve scheduling throughput and kube-scheduler utilization when the activeQ is empty, but pods are waiting in the backoffQ.
- Run `PreEnqueue` plugins when putting a pod into the backoffQ.

### Non-Goals

- Refactor the scheduling queue by changing backoff logic or merging the activeQ with the backoffQ.

## Proposal

At the beginning of the scheduling cycle, a pod is popped from the activeQ.
Currently, when activeQ is empty, it waits until some pod is placed into the queue.
This KEP proposes to pop the pod from the backoffQ when the activeQ is empty,
however the current mechanism of moving pods from the backoffQ to activeQ (aka flushing)
will still work as before to avoid the problem of pods starvation, 
which was the original reason of introducing the backoffQ.

To ensure the `PreEnqueue` is called for each pod taken into the scheduling cycle,
`PreEnqueue` plugins would be called before putting pods into the backoffQ.
It won't be done again when moving pods from the backoffQ to the activeQ.

### Risks and Mitigations

#### A tiny delay on the first scheduling attempts for newly created pods

While the scheduler handles a pod directly popping from the backoffQ, another pod that should be scheduled before the pod being scheduled now, may appear in the activeQ.
However, in the real world, if the scheduling latency is short enough, there won't be a visible downgrade in throughput.
This will only happen if there are no pods in the activeQ, so this can be mitigated by an appropriate rate of pod creation.

#### Backoff won't be working as a natural rate limiter in case of errors

In case of API calls errors (e.g., network issues), the backoffQ allows to limit the number of retries in a short term.
This proposal will take those pods earlier, leading to losing this mechanism.

After merging [kubernetes#128748](github.com/kubernetes/kubernetes/pull/128748),
it will be possible to distinguish pods backing off because of errors from those backing off because of an unschedulable attempt.
To preserve the efficiency of the pop() function, it will be necessary to divide the backoffQ into two queues: 
one for pods that were unschedulable, and another for those rejected due to an error.
Then popping will be performed only from the former, keeping the error backoff intact.

This has to be resolved before the beta is released, which means before the release of the feature.

#### One pod in the backoffQ could starve the others

The head of the BackoffQ is the pod with the closest backoff expiration,
and the backoff time is calculated based on the number of scheduling failures that the pod has experienced.
If one pod has a smaller attempt counter than others,
could the scheduler keep popping this pod ahead of other pods because the pod's backoff expires faster than others?
Actually, that wouldn't happen because the scheduler would increment the attempt counter of pods from the backoffQ as well,
which would make the backoff time larger after each scheduling attempt,
and the pod that had a smaller attempt number eventually won't be popped out.

### Low priority pod could be chosen to pop, even if high priority pod has a slightly later backoff expiration

The current mechanism of flushing from backoffQ to activeQ is done each second, taking all pods with backoff expired.
It means that, when they come to activeQ, they are sorted by priority there and taken in this order from activeQ.
It is important, because preemption of a lower priority pod could happen if a higher priority pod is scheduled later.

To mitigate this, `lessFn` function of backoffQ's heap will be changed, splitting the time to make one second windows (by ignoring milliseconds)
in which pods will be sorted by priority.
Those whole windows will be eventually flushed to activeQ, making no change in current behavior.

## Design Details

### Popping from the backoffQ in activeQ's pop()

To achieve the goal, activeQ's `pop()` method needs to be changed:
1. If the activeQ is empty, then instead of waiting for a pod to arrive at the activeQ, popping from the backoffQ is tried.
2. If the backoffQ is empty, then `pop()` is waiting for a pod as previously.
3. If the backoffQ is not empty, then the pod is processed like the pod would be taken from the activeQ, including increasing attempts number.
   It is popping from a heap data structure, so it should be fast enough not to cause any performance troubles.

To support monitoring, when popping from the backoffQ,
the `scheduler_queue_incoming_pods_total` metric with an `activeQ` queue and a new `PopFromBackoffQ` event label will be incremented.

### Notifying activeQ condition when a new pod appears in the backoffQ

Pods might appear in the backoffQ while `pop()` is hanging on point 2.
That's why it will be required to call `broadcast()` on the condition after adding a pod to the backoffQ.
It shouldn't cause any performance issues.

We could eventually want to move the backoffQ under activeQ's lock, but it's out of scope of this KEP.

### Calling PreEnqueue for the backoffQ

Currently, we call `PreEnqueue` at a single place, every time pods are being moved to the activeQ.
But, with this proposal, `PreEnqueue` will be called before moving a pod to the backoffQ, not when popping pods directly from the backoffQ.
Otherwise, a direct popping would be inefficient: it has to take the top backoffQ pod, check if it goes through `PreEnqueue` plugins, 
if not check the next backoffQ pod, until it finds the pod that goes through all `PreEnqueue` plugins.
Also, it means we'd have two paths that `PreEnqueue` plugins are invoked: when new pods are created and entering the scheduling queue, 
and when pods are pushed into the backoffQ.
At the moveToActiveQ level, these two paths could be distinguished by checking if the event is equal to `BackoffComplete`.

### Change backoffQ less function

As [mentioned](#low-priority-pod-could-be-chosen-to-pop-even-if-high-priority-pod-has-a-slightly-later-backoff-expiration) in risks,
backoffQ's heap `lessFn` function has to be changed to apply priority within 1 second windows.
The actual implementation takes backoff expiration times of two pods and compares which is lower.
The new version will ignore the milliseconds and use priorities to compare pods within those windows.
To make ordering predictable, in case of equal priorities within the same window,
the whole backoff time expiration will be eventually compared. See the pseudocode:

```go
func podsCompareBackoffCompleted(pInfo1, pInfo2 *framework.QueuedPodInfo) bool {
	if pInfo1.BackoffTime.InSeconds() != pInfo2.BackoffTime.InSeconds() {
		return pInfo1.BackoffTime.Before(pInfo2.BackoffTime)
	}
	if pInfo1.Priority != pInfo2.Priority {
		return pInfo1.Priority > pInfo2.Priority
	}
	return pInfo1.BackoffTime.Before(pInfo2.BackoffTime)
}
```

### Test Plan

[x] I/we understand the owners of the involved components may require updates to
existing tests to make this code solid enough prior to committing the changes necessary
to implement this enhancement.

##### Prerequisite testing updates

##### Unit tests

- `k8s.io/kubernetes/pkg/scheduler/backend/queue`: `2025-09-12` - `91.4`

##### Integration tests

- [`k8s.io/kubernetes/test/integration/scheduler/queueing`](https://github.com/kubernetes/kubernetes/tree/master/test/integration/scheduler/queueing) - added `TestPopFromBackoffQWhenActiveQEmpty` that covers the scenario.
- [scheduler_perf](https://github.com/kubernetes/kubernetes/tree/master/test/integration/scheduler_perf) - no perf test has been added to measure this particular scenario, as it's very difficult to simulate the conditions (empty active queue, pods waiting in backoff). At the same in time the available metrics for existing perf tests show that scheduling throughput has not decreased since 1.33 (when this featured was switched on as Beta).

##### e2e tests

The feature is scoped within the kube-scheduler internally, so there is no interaction between other components.
The whole feature should be already covered by integration tests.

### Graduation Criteria

The feature started as beta in 1.33 and has been enabled by default, because it is an internal kube-scheduler feature and guarded by a flag.

#### Alpha

N/A

#### Beta

- Feature implemented behind a feature flag and enabled by default.
- All tests from [Test Plan](#test-plan) implemented.
- Make sure [backoff in case of error](#backoff-wont-be-working-as-a-natural-rate-limiter-in-case-of-errors) is not skipped.

#### GA

- No issues have been reported in relation to this feature.

### Upgrade / Downgrade Strategy

**Upgrade**

After promoting to GA the feature gate `SchedulerPopFromBackoffQ` is enabled by default, so users don't need to opt in.
This is a purely in-memory feature for the kube-scheduler, so no special actions are required outside the scheduler.

**Downgrade**

Users need to disable the feature gate.

### Version Skew Strategy

This is a purely in-memory feature for the kube-scheduler, and hence no version skew strategy.

## Production Readiness Review Questionnaire

### Feature Enablement and Rollback

###### How can this feature be enabled / disabled in a live cluster?

- [x] Feature gate (also fill in values in `kep.yaml`)
  - Feature gate name: `SchedulerPopFromBackoffQ`
  - Components depending on the feature gate: kube-scheduler

###### Does enabling the feature change any default behavior?

Pods that are backing off might be scheduled earlier when the activeQ is empty.

###### Can the feature be disabled once it has been enabled (i.e., can we roll back the enablement)?

Yes.
The feature can be disabled in Beta version by restarting the kube-scheduler with the feature-gate off.

###### What happens if we re-enable the feature if it was previously rolled back?

The scheduler again starts to pop pods from the backoffQ when the activeQ is empty.

###### Are there any tests for feature enablement/disablement?

Given it's a purely in-memory feature and enablement/disablement requires restarting the component (to change the value of the feature flag),
having feature tests is enough.

### Rollout, Upgrade and Rollback Planning

###### How can a rollout or rollback fail? Can it impact already running workloads?

The partial failure in the rollout isn't there because the scheduler is the only component to roll out this feature.
But, if upgrading the scheduler itself fails somehow, new Pods won't be scheduled anymore,
while Pods, which are already scheduled, won't be affected in any case.

###### What specific metrics should inform a rollback?

Abnormal values of metrics related to the scheduling queue, meaning pods are stuck in the activeQ:
- The `scheduler_schedule_attempts_total` metric with the `scheduled` label is almost constant, while there are pending pods that should be schedulable.
  This could mean that pods from the backoffQ are taken instead of those from the activeQ.
- The `scheduler_pending_pods` metric with the `active` label is not decreasing, while with the `backoff` is almost constant.
- The `scheduler_queue_incoming_pods_total` metric with the `PopFromBackoffQ` label is increasing when there are pods in the activeQ.
  If this metric with this specific label is always higher than for other labels, it could also mean that this feature should be rolled back.
- The `scheduler_pod_scheduling_sli_duration_seconds` metric is visibly higher for schedulable pods.

###### Were upgrade and rollback tested? Was the upgrade->downgrade->upgrade path tested?

No. This feature is an in-memory feature of the scheduler
and thus calculations start from the beginning every time the scheduler is restarted.
So, just upgrading it and upgrade->downgrade->upgrade are both the same.

###### Is the rollout accompanied by any deprecations and/or removals of features, APIs, fields of API types, flags, etc.?

No

### Monitoring Requirements

###### How can an operator determine if the feature is in use by workloads?

They can check `scheduler_queue_incoming_pods_total` with the `PopFromBackoffQ` event label.

###### How can someone using this feature know that it is working for their instance?

N/A

###### What are the reasonable SLOs (Service Level Objectives) for the enhancement?

In the default scheduler, we should see the throughput around 100-150 pods/s ([ref](https://perf-dash.k8s.io/#/?jobname=gce-5000Nodes&metriccategoryname=Scheduler&metricname=LoadSchedulingThroughput&TestName=load)),
and this feature shouldn't bring any regression there.

Based on that `schedule_attempts_total` shouldn't be less than 100 in a second,
if there are enough unscheduled pods within the cluster.

###### What are the SLIs (Service Level Indicators) an operator can use to determine the health of the service?

- [x] Metrics
  - Metric name:
    - `schedule_attempts_total`
    - `scheduler_schedule_attempts_total` with `scheduled` label
    - `scheduler_pending_pods` with `active` and `backoff` labels
    - `scheduler_pod_scheduling_sli_duration_seconds`
  - Components exposing the metric: kube-scheduler

###### Are there any missing metrics that would be useful to have to improve observability of this feature?

No

### Dependencies

###### Does this feature depend on any specific services running in the cluster?

No

### Scalability

###### Will enabling / using this feature result in any new API calls?

No

###### Will enabling / using this feature result in introducing new API types?

No

###### Will enabling / using this feature result in any new calls to the cloud provider?

No

###### Will enabling / using this feature result in increasing size or count of the existing API objects?

No

###### Will enabling / using this feature result in increasing time taken by any operations covered by existing SLIs/SLOs?

No

###### Will enabling / using this feature result in a non-negligible increase of resource usage (CPU, RAM, disk, IO,...) in any components?

No

###### Can enabling / using this feature result in resource exhaustion of some node resources (PIDs, sockets, inodes, etc.)?

No

### Troubleshooting

###### How does this feature react if the API server and/or etcd is unavailable?

N/A

###### What are other known failure modes?

Unknown

###### What steps should be taken if SLOs are not being met to determine the problem?

## Implementation History

- 6th Feb 2025: The initial KEP is submitted.
- Feb-Mar 2025: Feature is implemented in the kubernetes codebase. PRs:

#130214 Split backoffQ into backoffQ and errorBackoffQ in scheduler kubernetes

#130492 Call PreEnqueue plugins before adding pod to backoffQ kubernetes

#130680 Update backoffQ's less function to order pods by priority in windows kubernetes

#130772 Pop from backoffQ when activeQ is empty kubernetes
- 15th Sep 2025: Feature gate updated in tests, KEP updated to upgrade to GA

## Drawbacks

<!--
Why should this KEP _not_ be implemented?
-->

## Alternatives

### Move pods in flushBackoffQCompleted when activeQ is empty

Moving the pod popping from the backoffQ to the existing `flushBackoffQCompleted` function (which already periodically moves pods to the activeQ) avoids changing `PreEnqueue` behavior, but it has some downsides.
Because flushing runs every second, it would be needed to pop more pods when the activeQ is empty.
This requires figuring out how many pods to pop, either by making it configurable or calculating it.
Also, if schedulable pods show up in the activeQ between flushes, a bunch of pods from the backoffQ might break activeQ priorities and slow down scheduling for the pods that are ready to go.
