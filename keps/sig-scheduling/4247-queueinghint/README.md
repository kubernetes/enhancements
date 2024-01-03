<!--
**Note:** When your KEP is complete, all of these comment blocks should be removed.

To get started with this template:

- [ ] **Pick a hosting SIG.**
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
# KEP-4247: Per-plugin callback functions for efficient requeueing in the scheduling queue

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
  - [User Stories (Optional)](#user-stories-optional)
    - [Story 1](#story-1)
    - [Story 2](#story-2)
  - [Notes/Constraints/Caveats (Optional)](#notesconstraintscaveats-optional)
  - [Risks and Mitigations](#risks-and-mitigations)
    - [mistake in the implementation could result in Pods being stuck in the unschedulable Pod pool in a long time unnecessarily.](#mistake-in-the-implementation-could-result-in-pods-being-stuck-in-the-unschedulable-pod-pool-in-a-long-time-unnecessarily)
    - [the increase in the memory usage](#the-increase-in-the-memory-usage)
    - [Breaking change in <code>EventsToRegister</code> in <code>EnqueueExtension</code>](#breaking-change-in--in-)
- [Design Details](#design-details)
  - [Overview](#overview)
  - [When to skip/not skip backoff](#when-to-skipnot-skip-backoff)
  - [How QueueingHint is executed in the scheduling queue](#how-queueinghint-is-executed-in-the-scheduling-queue)
    - [Pod rejected by one or more plugins](#pod-rejected-by-one-or-more-plugins)
    - [Pod rejected by <code>Pending</code> status](#pod-rejected-by--status)
  - [Track Pods being processed in the scheduling queue](#track-pods-being-processed-in-the-scheduling-queue)
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
  - [Return <code>QueueImmediately</code>, <code>QueueAfterBackoff</code>, and <code>QueueSkip</code> from <code>QueueingHintFn</code> instead of introducing new status <code>Pending</code>](#return---and--from--instead-of-introducing-new-status-)
  - [Implement <code>Blocked</code> status to block a next scheduling retry until the plugin returns <code>Queue</code>](#implement--status-to-block-a-next-scheduling-retry-until-the-plugin-returns-)
- [Infrastructure Needed (Optional)](#infrastructure-needed-optional)
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
  - [x] (R) Ensure GA e2e tests meet requirements for [Conformance Tests](https://github.com/kubernetes/community/blob/master/contributors/devel/sig-architecture/conformance-tests.md) 
  - [ ] (R) Minimum Two Week Window for GA e2e tests to prove flake free
- [x] (R) Graduation criteria is in place
  - [ ] (R) [all GA Endpoints](https://github.com/kubernetes/community/pull/1806) must be hit by [Conformance Tests](https://github.com/kubernetes/community/blob/master/contributors/devel/sig-architecture/conformance-tests.md) 
- [x] (R) Production readiness review completed
- [x] (R) Production readiness review approved
- [x] "Implementation History" section is up-to-date for milestone
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

<!--
This section is incredibly important for producing high-quality, user-focused
documentation such as release notes or a development roadmap. It should be
possible to collect this information before implementation begins, in order to
avoid requiring implementors to split their attention between writing release
notes and implementing the feature itself. KEP editors and SIG Docs
should help to ensure that the tone and content of the `Summary` section is
useful for a wide audience.

A good summary is probably at least a paragraph in length.

Both in this section and below, follow the guidelines of the [documentation
style guide]. In particular, wrap lines to a reasonable length, to make it
easier for reviewers to cite specific portions, and to minimize diff churn on
updates.

[documentation style guide]: https://github.com/kubernetes/community/blob/master/contributors/guide/style-guide.md
-->

The scheduler gets a new functionality called `QueueingHint` to get suggestion for how to requeue Pods from each plugin.
It helps reducing useless scheduling retries and thus improving the scheduling throughput.  

Also, by giving an ability to skip backoff in appropriate cases, the time to take to schedule Pods with dynamic resource allocation is improved.

## Motivation

<!--
This section is for explicitly listing the motivation, goals, and non-goals of
this KEP.  Describe why the change is important and the benefits to users. The
motivation section can optionally provide links to [experience reports] to
demonstrate the interest in a KEP within the wider Kubernetes community.

[experience reports]: https://github.com/golang/go/wiki/ExperienceReports
-->

**Retry Pods only when the probability of getting scheduled is high**

Currently, each plugin can define when to retry Pods, rejected by the plugin, to schedule roughly via `EventsToRegister`.

For example, NodeAffinity retries the Pods scheduling when Node is added or updated ([ref](https://github.com/kubernetes/kubernetes/blob/v1.27.6/pkg/scheduler/framework/plugins/nodeaffinity/node_affinity.go#L86)) because added/updated Node may have the label which matches with the NodeAffinity on the Pod.
But, actually, a lot of Node update events happens in the cluster, which cannot make the Pod previously rejected by NodeAffinity schedulable.
By introducing the callback function to filter out events more finely, the scheduler can retry scheduling of Pods which is only likely to be scheduled in the next scheduling cycle.

**Skip the backoff**

DRA plugin sometimes needs to reject Pods to wait for the update from the device driver. 
So, it's natural by its design to take several scheduling cycles to finish the scheduling of a Pod.

But, it takes time to go through backoff rather than waiting for the update from the device driver actually.
https://github.com/kubernetes/kubernetes/pull/117561

We want to improve the performance there by giving ability to plugins to skip backoff in selected cases.

### Goals

<!--
List the specific goals of the KEP. What is it trying to achieve? How will we
know that this has succeeded?
-->

Improve scheduling throughput with the following changes:
- Introduce `QueueingHint` to `EventsToRegister` and the scheduling queue requeues Pods based on the result from `QueueingHint`
- Improve how the Pods being processed are tracked by the scheduling queue and requeued to an appropriate queue if they are rejected and back to the queue.

### Non-Goals

<!--
What is out of scope for this KEP? Listing non-goals helps to focus discussion
and make progress.
-->

- Add a user-facing API.
- Remove the backoff mechanism completely in the scheduling queue.
- Overload the new functionality in the `PreEnqueue` extension point.
  - `QueueingHint` and `PreEnqueue` are both for the scheduling queue, but the responsibilities are completely different from each other.

## Proposal

<!--
This is where we get down to the specifics of what the proposal actually is.
This should have enough detail that reviewers can understand exactly what
you're proposing, but should not include things like API designs or
implementation. What is the desired outcome and how do we measure success?.
The "Design Details" section below is for the real
nitty-gritty.
-->

### User Stories (Optional)

<!--
Detail the things that people will be able to do if this KEP is implemented.
Include as much detail as possible so that people can understand the "how" of
the system. The goal here is to make this feel real for users without getting
bogged down.
-->

#### Story 1

Supposing developping the `NodeAffinity` plugin.

When `NodeAffinity` rejects Pods, those Pods might be schedulable in the following case:
- when a new Node is created, which matches the Pod's NodeAffinity.
- when an existing Node's label is updated and becomes matching the Pod's NodeAffinity.

In such events, QueueingHint of the NodeAffinity plugin returns `Queue`, otherwise returns `QueueSkip`.

#### Story 2

Supposing developping the `DynamicResourceAllocation` plugin.

After the scheduling cycle calculates the best Node, 
`DynamicResourceAllocation` needs to reject Pods once in the reserve extension point 
to wait for the update from the device driver.

So, Pods with dynamic resources need to go through several scheduling cycle by its design.

In this case, we can skip backoff by returning the status of `Pending` in a reserve extension point
so that the scheduling queue can understand that this Pod should skip the backoff when it's moved to activeQ.

### Notes/Constraints/Caveats (Optional)

<!--
What are the caveats to the proposal?
What are some important details that didn't come across above?
Go in to as much detail as necessary here.
This might be a good place to talk about core concepts and how they relate.
-->

### Risks and Mitigations

<!--
What are the risks of this proposal, and how do we mitigate? Think broadly.
For example, consider both security and how this will impact the larger
Kubernetes ecosystem.

How will security be reviewed, and by whom?

How will UX be reviewed, and by whom?

Consider including folks who also work outside the SIG or subproject.
-->

#### mistake in the implementation could result in Pods being stuck in the unschedulable Pod pool in a long time unnecessarily.

If a plugin has QueueingHint and it misses some events which can make Pods schedulable, 
Pods rejected by it may be stuck in the unschedulable Pod pool. 

The scheduling queue flushes the Pods in the unschedulable Pod pool priodically, and the interval of flushing is configurable. (5m by default)

It's on the way of being removed as the following issue described though, 
we will postpone its removal until all QueueingHint are implemented and we see no bug report for a while.
https://github.com/kubernetes/kubernetes/issues/87850

#### the increase in the memory usage

The memory usage in kube-scheduler is supposed to increase because the scheduling queue needs to keep the events happened during scheduling. 
Thus, the busier cluster it is, the more memory it's likely to require.

By freeing cached events as soon as possible, the impact on memory will be smaller. 
(although we cannot eliminate the memory usage increase completely.)

#### Breaking change in `EventsToRegister` in `EnqueueExtension`

It requires the action for the custom scheduler plugin developers. 
The `EventsToRegister` in `EnqueueExtension` changed the return value from `ClusterEvent` to `ClusterEventWithHint`. `ClusterEventWithHint` allows each plugin to filter out more useless events via the callback function named `QueueingHintFn`.

For the ease of migration, nil `QueueingHintFn` is treated as always returning `Queue`. 
So, if they want to just keep the existing behavior, they only have to change `ClusterEvent` to `ClusterEventWithHint` and register no `QueueingHintFn`. 

## Design Details

<!--
This section should contain enough information that the specifics of your
change are understandable. This may include API specs (though not always
required) or even code snippets. If there's any ambiguity about HOW your
proposal will be implemented, this is the place to discuss them.
-->

### Overview

The returning type of `EventsToRegister` is changed to `[]ClusterEventWithHint`

```go
// EnqueueExtensions is an optional interface that plugins can implement to efficiently
// move unschedulable Pods in internal scheduling queues. Plugins
// that fail pod scheduling (e.g., Filter plugins) are expected to implement this interface.
type EnqueueExtensions interface {
	Plugin
	// EventsToRegister returns a series of possible events that may cause a Pod
	// failed by this plugin schedulable. Each event has a callback function that
	// filters out events to reduce useless retry of Pod's scheduling.
	// The events will be registered when instantiating the internal scheduling queue,
	// and leveraged to build event handlers dynamically.
	// Note: the returned list needs to be static (not depend on configuration parameters);
	// otherwise it would lead to undefined behavior.
	EventsToRegister() []ClusterEventWithHint
}
```

Each `ClusterEventWithHint` has `ClusterEvent` and `QueueingHintFn`, which is executed when the event happens
and determine whether the event could make the Pod schedulable or not.
See [How QueueingHint is executed in the scheduling queue](#how-queueinghint-is-executed-in-the-scheduling-queue) to see the detail.

```go
type ClusterEventWithHint struct {
	Event ClusterEvent
	// QueueingHintFn is executed for the plugin rejected by this plugin when the above Event happens,
	// and filters out events to reduce useless retry of Pod's scheduling.
	// It's an optional field. If not set,
	// the scheduling of Pods will be always retried when this Event happens.
	// (the same as Queue)
	QueueingHintFn QueueingHintFn
}

// QueueingHintFn returns a hint that signals whether the event can make a Pod,
// which was rejected by this plugin in the past scheduling cycle, schedulable or not.
// It's called before a Pod gets moved from unschedulableQ to backoffQ or activeQ.
// If it returns an error, we'll take the returned QueueingHint as `QueueAfterBackoff` at the caller whatever we returned here so that
// we can prevent the Pod from being stuck in the unschedulable pod pool.
//
// - `pod`: the Pod to be enqueued, which is rejected by this plugin in the past.
// - `oldObj` `newObj`: the object involved in that event.
//   - For example, the given event is "Node deleted", the `oldObj` will be that deleted Node.
//   - `oldObj` is nil if the event is add event.
//   - `newObj` is nil if the event is delete event.
type QueueingHintFn func(logger klog.Logger, pod *v1.Pod, oldObj, newObj interface{}) (QueueingHint, error)

type QueueingHint int

const (
	// QueueSkip implies that the cluster event has no impact on
	// scheduling of the pod.
	QueueSkip QueueingHint = iota

	// Queue implies that the Pod may be schedulable by the event.
	Queue
)
```

### When to skip/not skip backoff

BackoffQ is a light way of keeping throughput high 
by preventing pods that are "permanently unschedulable" from blocking the queue.

And, the more the Pod has been rejected in the scheduling cycle, the longer the Pod needs to wait as backoff.
**We can regard the backoff as a penalty of wasting the scheduling cycle.**

So, when, for example, NodeAffinity rejected the Pod and later returns `Queue` in its `QueueingHintFn`,
the Pod's scheduling is retried after going through the backoff. 
It's because the past scheduling cycle was wasted by that Pod.

But, some plugins need to go through some failures in the scheduling cycle by design.
[DRA](https://github.com/kubernetes/kubernetes/tree/v1.27.6/pkg/scheduler/framework/plugins/dynamicresources) plugin is one example in in-tree plugins - at the Reserve extension point, it tells the resource driver the scheduling result, and rejects the Pod once to wait for the response from the resource driver.
In this kind of rejections, we cannot say the scheduling cycle is wasted because the scheduling result from it is used to proceed the Pod's scheduling forward, that particular scheduling cycle is failed though.
So, Pods rejected by such reasons don't need to suffer a penalty (backoff).

In order to support such cases, we introduces a new status `Pending`.
When the `DRA` plugin rejected the Pod with `Pending` and later returns `Queue` in its `QueueingHintFn`,
the pod skips the backoff and the Pod's scheduling is retried. 

### How QueueingHint is executed in the scheduling queue

When the cluster event happens, the scheduling queue executes `QueueingHintFn`
of plugins which rejected the Pod in a previous scheduling cycle.

Here are some scenarios to describe how they're executed and how the Pod is moved.

#### Pod rejected by one or more plugins

Let's say there are three Nodes. 
When the Pod goes to the scheduling cycle, one Node is rejected due to no enough capacity, 
other two Nodes are rejected because they don't match Pod's NodeAffinity.

In this case, the Pod gets `NodeResourceFit` and `NodeAffinity` as unschedulable plugins,
and it's put back to the unschedulable pod pool.

After then, every time the cluster events registered in those plugins happen,
the scheduling queue notifies them through QueueingHint.
If either of QueueingHintFn from `NodeResourceFit` or `NodeAffinity` returns `Queue`,
the Pod is moved to activeQ/backoffQ.
(For example, when `NodeAdded` event happens, the QueueingHint of `NodeResourceFit` return `Queue`
because the Pod may be schedulable to that new Node.)

Whether it's moved to activeQ or backoffQ, that depends how long this Pod has stayed in the unschedulable pod pool.
If the time staying in the unschedulable pod pool is longer than an expected backoff delay for the pod,
it directly goes to activeQ. Otherwise, it goes to backoffQ.

#### Pod rejected by `Pending` status

When DRA plugin returns `Pending` to the Pod in a Reserve extension point,
the Pod goes back to the scheduling queue and the scheduling queue records DRA as pending plugins of the Pod. 

When DRA plugin's QueueingHint returns `Queue` for a event after that, 
the scheduling queue put this Pod directly into activeQ.

### Track Pods being processed in the scheduling queue

By introducing QueueingHint, we can retry the scheduling only when particular event happens.
But, what if such events happen during Pod's scheduling?

The scheduler takes snapshot of the cluster and schedules Pods based on the snapshot. And the snapshot is updated everytime the scheduling cycle is started, in other words, the same snapshot is used in the same scheduling cycle.

Thinking about a problematic scenario, for example, Pod is being scheduled and it's going to be rejected by NodeAffinity because no Node matches the Pod's NodeAffinity. But, actually, during the scheduling, one new Node is created, which matches the Pod's NodeAffinity.

As mentioned, that new Node doesn't get in the candidates during this scheduling cycle, so this Pod is rejected by NodeAffinity anyways. 
The problem here is that, if the scheduling queue put this Pod into the unschedulable Pod pool, this Pod would need to wait for another event, although there is already a Node matching the Pod's NodeAffinity.

In order to prevent such Pods from missing the events during its scheduling, 
the scheduling queue remembers events happened during Pods's scheduling 
and decide where the Pod is enqueued to based on those events and `QueueingHint`.

So, the scheduling queue caches all events since the Pod leaves the scheduling queue 
until the Pod come back to the scheduling queue or got scheduled. 
And, cached events are discarded when cached events are no longer needed. 

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

- `k8s.io/kubernetes/pkg/scheduler/internal/queue`: `10-01 20:28 JST` - `88.4`

##### Integration tests

<!--
Integration tests are contained in k8s.io/kubernetes/test/integration.
Integration tests allow control of the configuration parameters used to start the binaries under test.
This is different from e2e tests which do not allow configuration of parameters.
Doing this allows testing non-default options and multiple different and potentially conflicting command line options.
-->

<!--
This question should be filled when targeting a release.
For Alpha, describe what tests will be added to ensure proper quality of the enhancement.

For Beta and GA, add links to added tests together with links to k8s-triage for those tests:
https://storage.googleapis.com/k8s-triage/index.html
-->

- [`k8s.io/kubernetes/test/integration/scheduler/rescheduling_test.go`](https://github.com/kubernetes/kubernetes/blob/v1.28.0/test/integration/scheduler/rescheduling_test.go#L117): 
  - https://storage.googleapis.com/k8s-triage/index.html?test=TestReScheduling
- [scheduler_perf](https://github.com/kubernetes/kubernetes/tree/master/test/integration/scheduler_perf)
  - We'll add scenarios where the cluster size gets changed several times so that we can make sure there is no regression in such cases of the cluster situation being changed a lot. 

##### e2e tests

<!--
This question should be filled when targeting a release.
For Alpha, describe what tests will be added to ensure proper quality of the enhancement.

For Beta and GA, add links to added tests together with links to k8s-triage for those tests:
https://storage.googleapis.com/k8s-triage/index.html

We expect no non-infra related flakes in the last month as a GA graduation criteria.
-->


n/a

This feature doesn't introduce any new API endpoints and doesn't interact with other components. 
So, E2E tests doesn't add extra value to integration tests.

But, regarding the performance test, 
we'll keep monitoring the regression in scheduler_perf results, specially at high number of nodes.
https://perf-dash.k8s.io/#/?jobname=scheduler-perf-benchmark&metriccategoryname=Scheduler&metricname=BenchmarkPerfResults&Metric=SchedulingThroughput&Name=SchedulingBasic%2F5000Nodes%2Fnamespace-2&extension_point=not%20applicable&result=not%20applicable

### Graduation Criteria

It was suggested we have a KEP for QueueingHint after we implemented it. 
It's kind of a special case though, we can assume DRA is the parent KEP and this KEP stems from it. 
And I set the alpha version v1.26 which is the same as DRA KEP, 
and the beta version v1.28 which we actually implemented it and enable it via the beta feature flag (enabled by default).

Slack discussion: https://kubernetes.slack.com/archives/C5P3FE08M/p1695639140018139?thread_ts=1694167948.846139&cid=C5P3FE08M

#### Alpha

n/a

#### Beta

- The scheduling queue is changed to work with QueueingHint.
- No performance degradation is confirmed via scheduler_perf.
- The feature gate is implemented. (disabled by default) 
- QueueingHint implementation in all plugins.
- The integration tests are implemented for requeueing scenarios in all plugins.
- `PreCheck` feature in the scheduling queue is completely removed.
- No significant degradation in memory comsumption.
- No performance degradation is confirmed via scheduler_perf.
- The feature gate is enabled by default.
- No bug report for a while after enabling it by default.

#### GA

- No bug report for a while after reaching Beta.

### Upgrade / Downgrade Strategy

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

**Upgrade**

Nothing needs to be done to opt-in this feature. (The feature gate is enabled by default)
This is purely in-memory feature for kube-scheduler, so no special actions are required outside the scheduler.

**Downgrade**

Users need to disable the feature gate.

### Version Skew Strategy

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

This is purely in-memory feature for kube-scheduler, so version skew issues don't exist.

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
  - Feature gate name: `SchedulerQueueingHints`
  - Components depending on the feature gate: kube-scheduler
- [ ] Other
  - Describe the mechanism:
  - Will enabling / disabling the feature require downtime of the control
    plane?
  - Will enabling / disabling the feature require downtime or reprovisioning
    of a node?

###### Does enabling the feature change any default behavior?

<!--
Any change of default behavior may be surprising to users or break existing
automations, so be extremely careful here.
-->

No, basically. 
But, if there is a bug in the implementation, Pods' rescheduling may be delayed up to `--pod-max-in-unschedulable-pods-duration` (5min by default). 

###### Can the feature be disabled once it has been enabled (i.e. can we roll back the enablement)?

<!--
Describe the consequences on existing workloads (e.g., if this is a runtime
feature, can it break the existing applications?).

Feature gates are typically disabled by setting the flag to `false` and
restarting the component. No other changes should be necessary to disable the
feature.

NOTE: Also set `disable-supported` to `true` or `false` in `kep.yaml`.
-->

Yes.
The feature can be disabled in Alpha and Beta versions
by restarting kube-scheduler with the feature-gate off.

###### What happens if we reenable the feature if it was previously rolled back?

The scheduling queue again starts to work with `QueueingHint`.

###### Are there any tests for feature enablement/disablement?

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

Given it's purely in-memory feature and enablement/disablement requires restarting the component (to change the value of feature flag), 
having feature tests is enough.

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

The partly failure in the rollout isn't there because the scheduler is only the component to rollout this feature. 
But, if upgrading the scheduler itself fails somehow, new Pods won't be scheduled anymore. 
(while Pods, which are already scheduled, won't be affected in any cases.)

###### What specific metrics should inform a rollback?

<!--
What signals should users be paying attention to when the feature is young
that might indicate a serious problem?
-->

Maybe something goes wrong with QueueingHint and Pods are stuck in the queue if 
- `scheduler_pending_pods` metric with `queue: unschedulable` label grows and keeps high number abnormally 
- `pod_scheduling_sli_duration_seconds` metric grows abnormally

###### Were upgrade and rollback tested? Was the upgrade->downgrade->upgrade path tested?

<!--
Describe manual testing that was done and the outcomes.
Longer term, we may want to require automated upgrade/rollback tests, but we
are missing a bunch of machinery and tooling and can't do that now.
-->

No. This feature is a in-memory feature of the scheduler 
and thus calculations start from the beginning every time the scheduler is restarted.
So, just upgrading it and upgrade->downgrade->upgrade are both the same.

###### Is the rollout accompanied by any deprecations and/or removals of features, APIs, fields of API types, flags, etc.?

<!--
Even if applying deprecation policies, they may still surprise some users.
-->

No.

### Monitoring Requirements

<!--
This section must be completed when targeting beta to a release.

For GA, this section is required: approvers should be able to confirm the
previous answers based on experience in the field.
-->

###### How can an operator determine if the feature is in use by workloads?

<!--
Ideally, this should be a metric. Operations against the Kubernetes API (e.g.,
checking if there are objects with field X set) may be a last resort. Avoid
logs or events for this purpose.
-->

This feature is used during all Pods' scheduling if the feature gate is enabled.

###### How can someone using this feature know that it is working for their instance?

<!--
For instance, if this is a pod-related feature, it should be possible to determine if the feature is functioning properly
for each individual pod.
Pick one more of these and delete the rest.
Please describe all items visible to end users below with sufficient detail so that they can verify correct enablement
and operation of this feature.
Recall that end users cannot usually observe component logs or access metrics.
-->

n/a

###### What are the reasonable SLOs (Service Level Objectives) for the enhancement?

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

In the default scheduler, we should see the throughput around 100-150 pods/s ([ref](https://perf-dash.k8s.io/#/?jobname=gce-5000Nodes&metriccategoryname=Scheduler&metricname=LoadSchedulingThroughput&TestName=load)), and this feature shouldn't bring any regression there.

Based on that: 
- `schedule_attempts_total` shouldn't be less than 100 in a second.
- the average of `scheduling_algorithm_duration_seconds` shouldn't be above 10 ms.

###### What are the SLIs (Service Level Indicators) an operator can use to determine the health of the service?

<!--
Pick one more of these and delete the rest.
-->

- [x] Metrics
  - Metric name: 
    - `schedule_attempts_total` 
    - `scheduling_algorithm_duration_seconds` 
    - `scheduler_pending_pods` with `queue: unschedulable`
  - Components exposing the metric: kube-scheduler

###### Are there any missing metrics that would be useful to have to improve observability of this feature?

<!--
Describe the metrics themselves and the reasons why they weren't added (e.g., cost,
implementation difficulties, etc.).
-->

No.

### Dependencies

<!--
This section must be completed when targeting beta to a release.
-->

###### Does this feature depend on any specific services running in the cluster?

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

No.

### Scalability

<!--
For alpha, this section is encouraged: reviewers should consider these questions
and attempt to answer them.

For beta, this section is required: reviewers must answer these questions.

For GA, this section is required: approvers should be able to confirm the
previous answers based on experience in the field.
-->

###### Will enabling / using this feature result in any new API calls?

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

No.

###### Will enabling / using this feature result in introducing new API types?

<!--
Describe them, providing:
  - API type
  - Supported number of objects per cluster
  - Supported number of objects per namespace (for namespace-scoped objects)
-->

No.

###### Will enabling / using this feature result in any new calls to the cloud provider?

<!--
Describe them, providing:
  - Which API(s):
  - Estimated increase:
-->

No.

###### Will enabling / using this feature result in increasing size or count of the existing API objects?

<!--
Describe them, providing:
  - API type(s):
  - Estimated increase in size: (e.g., new annotation of size 32B)
  - Estimated amount of new objects: (e.g., new Object X for every existing Pod)
-->

No.

###### Will enabling / using this feature result in increasing time taken by any operations covered by existing SLIs/SLOs?

<!--
Look at the [existing SLIs/SLOs].

Think about adding additional work or introducing new steps in between
(e.g. need to do X to start a container), etc. Please describe the details.

[existing SLIs/SLOs]: https://git.k8s.io/community/sig-scalability/slos/slos.md#kubernetes-slisslos
-->

No.

###### Will enabling / using this feature result in non-negligible increase of resource usage (CPU, RAM, disk, IO, ...) in any components?

<!--
Things to keep in mind include: additional in-memory state, additional
non-trivial computations, excessive access to disks (including increased log
volume), significant amount of data sent and/or received over network, etc.
This through this both in small and large cases, again with respect to the
[supported limits].

[supported limits]: https://git.k8s.io/community//sig-scalability/configs-and-limits/thresholds.md
-->

Yes. 
The memory usage in kube-scheduler is supposed to increase because the scheduling queue needs to keep the events happened during scheduling. Thus, the busier cluster it is, the more memory it's likely to require.

###### Can enabling / using this feature result in resource exhaustion of some node resources (PIDs, sockets, inodes, etc.)?

<!--
Focus not just on happy cases, but primarily on more pathological cases
(e.g. probes taking a minute instead of milliseconds, failed pods consuming resources, etc.).
If any of the resources can be exhausted, how this is mitigated with the existing limits
(e.g. pods per node) or new limits added by this KEP?

Are there any tests that were run/should be run to understand performance characteristics better
and validate the declared limits?
-->

No.

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

n/a

###### What are other known failure modes?

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

- If a plugin' QueueingHint implementation has bugs and, for example, misses some events that can make Pods schedulable,
Pods rejected by those plugins may be stuck in the unschedulable Pod pool for a long time.
  - Detection: Pods get `FailedScheduling` event, but not retried during 5 min even if the cluster should have a state that can accommodate those Pods. 
  - Mitigations: The scheduling queue priodically flushing Pods in the unschedulable Pod pool. So, even if such bug exists, Pods' scheduling are retried after a certain period, which is 5 min by default. You can shorten the max duration that Pods can stay in the unschedulable Pod pool by using `--pod-max-in-unschedulable-pods-duration`.
  - Diagnostics: If you increases the log level to more than 5, you can see the logs related to `QueueingHint` in the scheduling queue. Also, the in-tree plugins emits all logs in QueueingHint with log level 5. (If you have a custom plugin, you may want to check the log level in its QueueingHint.)
  - Testing: There are multiple unit tests to confirm `flushUnschedulablePodsLeftover` is working expectedly.

###### What steps should be taken if SLOs are not being met to determine the problem?

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

- Jun 26, 2023: The QueueingHint is implemented and the `EnqueueExtension` interface is changed.
- Jul 15, 2023: The feature gate is implemented. (enabled by default)
- Jul 18, 2023: The scheduling queue tracks the Pod being processed to put it back to an appropriate queue.
- Oct 01, 2023: The initial KEP is submitted.
- Dec 13, 2023: The feature gate is changed to be disabled by default.
- Dec 31, 2023: The KEP is updated based on the situation as of v1.30 release cycle. The beta/GA criteria is sorted.

## Drawbacks

<!--
Why should this KEP _not_ be implemented?
-->

## Alternatives

<!--
What other approaches did you consider, and why did you rule them out? These do
not need to be as detailed as the proposal, but should include enough
information to express the idea and why it was not acceptable.
-->

### Return `QueueImmediately`, `QueueAfterBackoff`, and `QueueSkip` from `QueueingHintFn` instead of introducing new status `Pending`

Instead of requeueing Pods based on why it was rejected, we can do the same by introducing separate `QueueingHint` for queueing - `QueueImmediately` and `QueueAfterBackoff`.

But, as explained in [When to skip/not skip backoff](#when-to-skipnot-skip-backoff), the backoff is a penalty of wasting the scheduling cycle. Also, some few scenario (DRA) don't waste the scheduling cycle, they reject Pods in that scheduling cycle though.

So, whether skipping backoff or not, it's something very close to why the Pod was rejected,
and thus it's easier to be decided when the Pod is rejected than when the Pod is actually requeued.

### Implement `Blocked` status to block a next scheduling retry until the plugin returns `Queue`

For example, when a PVC for the Pod isn't found, the Pod cannot be scheduled and `VolumeBinding` plugin returns `UnschedulableAndUnresolvable` in this case.
The point here is that this Pod will never be schedulable until the appropriate PVC is created for the Pod. 

For such cases, we introduced a new supplemental status `Blocked`, which can be used like this:

```go
func (pl *VolumeBinding) PreFilter(ctx context.Context, state *framework.CycleState, pod *v1.Pod) (*framework.PreFilterResult, *framework.Status) {
	if hasPVC, err := pl.podHasPVCs(pod); err != nil {
    if apierrors.IsNotFound(err) {
      // PVC isn't found for this Pod.
      // This rejection must be resolved before retrying this Pod's scheduling.
      // Otherwise, the retry would just result in the same rejection from this plugin here.
      return UnschedulableAndUnresolvable | Blocked
    }
    //...
}
```

Thinking about the current usecase of it, my first thought is that many PreFilter and Reserve plugins would want to return `Blocked`. 

But, Looking at how PreFilter and Reserve plugins are executed from the scheduling framework runtime,
when one of them return unschedulable, the runtime stops the iteration at that point 
and the rest of plugins aren't executed.

So, in other words, when one of PreFilter and Reserve plugins return unschedulable,
the plugin would be the only one registered in the unschedulable plugins of the Pod, 
and the Pod will stay in the unschedulable Pod pool until the plugin return `Queue` in QueueingHint.

Meaning, PreFilter and Reserve plugins don't need to return `Blocked`.

The next question is that any Filter plugins would want to use `Blocked` or not.
But, I don't think any of in-tree Filter plugins want.

## Infrastructure Needed (Optional)

<!--
Use this section if you need things from the project/SIG. Examples include a
new subproject, repos requested, or GitHub details. Listing these here allows a
SIG to get the process for these resources started right away.
-->
