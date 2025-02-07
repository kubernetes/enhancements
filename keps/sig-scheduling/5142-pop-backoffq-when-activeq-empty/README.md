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
    - [Scheduling throughput might be affected](#scheduling-throughput-might-be-affected)
    - [Backoff won't be working as natural rate limiter in case of errors](#backoff-wont-be-working-as-natural-rate-limiter-in-case-of-errors)
    - [One pod in backoffQ could starve the others](#one-pod-in-backoffq-could-starve-the-others)
- [Design Details](#design-details)
  - [Popping from backoffQ in activeQ's pop()](#popping-from-backoffq-in-activeqs-pop)
  - [Notifying activeQ condition when new pod appears in backoffQ](#notifying-activeq-condition-when-new-pod-appears-in-backoffq)
  - [Calling PreEnqueue for backoffQ](#calling-preenqueue-for-backoffq)
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

- [ ] (R) Enhancement issue in release milestone, which links to KEP dir in [kubernetes/enhancements] (not the initial KEP PR)
- [ ] (R) KEP approvers have approved the KEP status as `implementable`
- [ ] (R) Design details are appropriately documented
- [ ] (R) Test plan is in place, giving consideration to SIG Architecture and SIG Testing input (including test refactors)
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

This KEP proposes improving scheduling queue behavior by popping pods from backoffQ when activeQ is empty. 
This would allow to increase utilization of kube-scheduler cycles as well as reduce waiting time for pending pods 
that were previously unschedulable.

## Motivation

There are three queues in scheduling queue: 
- activeQ contains pods ready for scheduling, 
- unschedulableQ holds pods that were unschedulable in their scheduling cycle and are waiting for cluster state to change,
- backoffQ stores pods that failed scheduling attempts (either due to being unschedulable or errors) and could be schedulable again,
  but applying a backoff penalty, scaled with the number of attempts.

When activeQ is not empty, scheduler pops the highest priority pod from activeQ.
However, when activeQ is empty, kube-scheduler idles, waiting for any pod being present in activeQ, 
even if pods are in the backoffQ but their backoff period hasn't expired. 
In scenarios when pods are waiting, but in backoffQ, 
kube-scheduler should be able to consider those pods for scheduling, even if the backoff is not completed, to avoid the idle time.

### Goals

- Improve scheduling throughput and kube-scheduler utilization when activeQ is empty, but pods are waiting in backoffQ.
- Run `PreEnqueue` plugins when putting pod into backoffQ.

### Non-Goals

- Refactor scheduling queue by changing backoff logic or merging activeQ with backoffQ.

## Proposal

At the beginning of scheduling cycle, pod is popped from activeQ. 
If activeQ is empty, it waits until a pod is placed into the queue.
This KEP proposes to pop the pod from backoffQ when activeQ is empty.

To ensure the `PreEnqueue` is called for each pod taken into scheduling cycle,
`PreEnqueue` plugins would be called before putting pods into backoffQ.
It won't be done again when moving pods from backoffQ to activeQ.

### Risks and Mitigations

#### Scheduling throughput might be affected

While popping from backoffQ, another pod might appear in activeQ ready to be scheduled.
If the pop operation is short enough, there won't be a visible downgrade in throughput.
The only concern might be that less pods from activeQ might be taken in some period of time in favor of backoffQ, 
but that's a user responsibility to create enough amount of pods to be scheduled from activeQ, not to cause this KEP behavior to happen.

#### Backoff won't be working as natural rate limiter in case of errors

In case of API calls errors (e.g. network issues), backoffQ allows to limit number of retries in a short term.
This proposal will take those pods earlier, leading to losing this mechanism.

After merging [kubernetes#128748](github.com/kubernetes/kubernetes/pull/128748), 
it will be possible to distinguish pods backing off because of errors from those backing off because of unschedulable attempt. 
This information could be used when popping, by filtering only the pods that are from unschedulable attempt or even splitting backoffQ. 

#### One pod in backoffQ could starve the others

If a pod popped from the backoffQ fails its scheduling attempt and come back to the queue, it might be selected again, ahead of other pods.

To prevent this, while popping pod from backoffQ, its attempt counter will be incremented as if it had been taken from the activeQ.
This will give other pods a chance to be scheduled.

## Design Details

### Popping from backoffQ in activeQ's pop()

To achieve the goal, activeQ's `pop()` method needs to be changed:
1. If activeQ is empty, then instead of waiting on condition, popping from backoffQ is tried.
2. If backoffQ is empty, then `pop()` is waiting on condition as previously.
3. If backoffQ is not empty, then the pod is processed like the pod would be taken from activeQ, including increasing attempts number.
   It is poping from a heap data structure, so it should be fast enough not to cause any performance troubles.

### Notifying activeQ condition when new pod appears in backoffQ

Pods might appear in backoffQ while `pop()` is hanging on point 2. 
That's why it will be required to call `broadcast()` on condition after adding a pod to backoffQ.
It shouldn't cause any performance issues.

We could eventually want to move backoffQ under activeQ's lock, but it's out of scope of this KEP.

### Calling PreEnqueue for backoffQ

`PreEnqueue` plugins have to be called for every pod before they are taken to scheduling cycle.
Initially, those plugins were called before moving pod to activeQ.
With this proposal, `PreEnqueue` will need to be called before moving pod to backoffQ 
and those calls need to be skipped for the pods that are moved later from backoffQ to activeQ.
At moveToActiveQ level, these two paths could be distinguished by checking if event is equal to `BackoffComplete`.

### Test Plan

[x] I/we understand the owners of the involved components may require updates to
existing tests to make this code solid enough prior to committing the changes necessary
to implement this enhancement.

##### Prerequisite testing updates

##### Unit tests

- `k8s.io/kubernetes/pkg/scheduler/backend/queue`: `2025-02-06` - `91.4`

##### Integration tests

- [`k8s.io/kubernetes/test/integration/scheduler/queueing`](https://github.com/kubernetes/kubernetes/tree/master/test/integration/scheduler/queueing) - add test cases covering the scenario.
- [scheduler_perf](https://github.com/kubernetes/kubernetes/tree/master/test/integration/scheduler_perf) - add test cases measuring performance in this scenario.

##### e2e tests

Feature is scoped within kube-scheduler internally, so there is no interaction between other components.
Whole feature should be already covered by integration tests.

### Graduation Criteria

#### Alpha

- Feature implemented behind a feature flag.
- All tests from [Test Plan](#test-plan) implemented.

#### Beta

- Gather feedback from users and fix reported bugs.
- Change the feature flag to be enabled by default.

#### GA

- Gather feedback from users and fix reported bugs.

### Upgrade / Downgrade Strategy

**Upgrade**

During the alpha period, users have to enable the feature gate `PopBackoffQWhenEmptyActiveQ` to opt in this feature.
This is purely in-memory feature for kube-scheduler, so no special actions are required outside the scheduler.

**Downgrade**

Users need to disable the feature gate.

### Version Skew Strategy

This is purely in-memory feature for kube-scheduler, and hence no version skew strategy.

## Production Readiness Review Questionnaire

### Feature Enablement and Rollback

###### How can this feature be enabled / disabled in a live cluster?

- [x] Feature gate (also fill in values in `kep.yaml`)
  - Feature gate name: `PopBackoffQWhenEmptyActiveQ`
  - Components depending on the feature gate: kube-scheduler

###### Does enabling the feature change any default behavior?

Pods that are backoffQ might be scheduled earlier when activeQ is empty.

###### Can the feature be disabled once it has been enabled (i.e. can we roll back the enablement)?

Yes.
The feature can be disabled in Alpha and Beta versions
by restarting kube-scheduler with the feature-gate off.

###### What happens if we reenable the feature if it was previously rolled back?

The scheduler again starts to pop pods from backoffQ when activeQ is empty.

###### Are there any tests for feature enablement/disablement?

Given it's purely in-memory feature and enablement/disablement requires restarting the component (to change the value of feature flag), 
having feature tests is enough.

### Rollout, Upgrade and Rollback Planning

<!--
This section must be completed when targeting beta to a release.
-->

###### How can a rollout or rollback fail? Can it impact already running workloads?

The partly failure in the rollout isn't there because the scheduler is the only component to rollout this feature. 
But, if upgrading the scheduler itself fails somehow, new Pods won't be scheduled anymore,
while Pods, which are already scheduled, won't be affected in any case.

###### What specific metrics should inform a rollback?

Abnormal values of metrics related to scheduling queue, meaning pods are stuck in activeQ:
- `scheduler_schedule_attempts_total` metric with `scheduled` label is almost constant, while there are pending pods that should be schedulable. 
  This could mean that pods from backoffQ are taken instead of those from activeQ.
- `scheduler_pending_pods` metric with `active` label is not decreasing, while with `backoff` is almost constant.
- `scheduler_pod_scheduling_sli_duration_seconds` metric is visibly higher for schedulable pods.

###### Were upgrade and rollback tested? Was the upgrade->downgrade->upgrade path tested?

No. This feature is a in-memory feature of the scheduler 
and thus calculations start from the beginning every time the scheduler is restarted.
So, just upgrading it and upgrade->downgrade->upgrade are both the same.

###### Is the rollout accompanied by any deprecations and/or removals of features, APIs, fields of API types, flags, etc.?

No

### Monitoring Requirements

###### How can an operator determine if the feature is in use by workloads?

This feature is used during scheduling when activeQ is empty and if the feature gate is enabled.

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

###### Will enabling / using this feature result in non-negligible increase of resource usage (CPU, RAM, disk, IO, ...) in any components?

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

## Drawbacks

<!--
Why should this KEP _not_ be implemented?
-->

## Alternatives

### Move pods in flushBackoffQCompleted when activeQ is empty

Moving the pod popping from backoffQ to the existing `flushBackoffQCompleted` function (which already periodically moves pods to activeQ) avoids changing `PreEnqueue` behavior, but it has some downsides. 
Because flushing runs every second, it would be needed to pop more pods when activeQ is empty. 
This require to figure out how many pods to pop, either by making it configurable it or calculating it. 
Also, if schedulable pods show up in activeQ between flushes, a bunch of pods from backoffQ might break activeQ priorities and slow down scheduling for the pods that are ready to go.
