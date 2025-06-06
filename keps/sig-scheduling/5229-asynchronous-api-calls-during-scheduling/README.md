# KEP-5229: Asynchronous API calls during scheduling

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
  - [API calls categorization](#api-calls-categorization)
  - [1: Where and how to handle API calls in the kube-scheduler](#1-where-and-how-to-handle-api-calls-in-the-kube-scheduler)
    - [1.1: Handle API calls in the scheduling queue](#11-handle-api-calls-in-the-scheduling-queue)
    - [1.2: Handle API calls in the handleSchedulingFailure](#12-handle-api-calls-in-the-handleschedulingfailure)
    - [1.3: Use advanced queue and don't block the pod from being scheduled in the meantime](#13-use-advanced-queue-and-dont-block-the-pod-from-being-scheduled-in-the-meantime)
  - [2: How to make the API calls asynchronous](#2-how-to-make-the-api-calls-asynchronous)
    - [2.1: Just dispatch goroutines](#21-just-dispatch-goroutines)
    - [2.2: Make the API calls queued](#22-make-the-api-calls-queued)
    - [2.3: Send API calls through a kube-scheduler's cache](#23-send-api-calls-through-a-kube-schedulers-cache)
  - [Another things worth considering](#another-things-worth-considering)
  - [Notes/Constraints/Caveats (Optional)](#notesconstraintscaveats-optional)
  - [Risks and Mitigations](#risks-and-mitigations)
- [Design Details](#design-details)
  - [Test Plan](#test-plan)
      - [Prerequisite testing updates](#prerequisite-testing-updates)
      - [Unit tests](#unit-tests)
      - [Integration tests](#integration-tests)
      - [e2e tests](#e2e-tests)
  - [Graduation Criteria](#graduation-criteria)
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

This KEP proposes making all API calls during scheduling asynchronous, by introducing a new kube-scheduler-wide way of handling such calls.

## Motivation

<!--
This section is for explicitly listing the motivation, goals, and non-goals of
this KEP.  Describe why the change is important and the benefits to users. The
motivation section can optionally provide links to [experience reports] to
demonstrate the interest in a KEP within the wider Kubernetes community.

[experience reports]: https://github.com/golang/go/wiki/ExperienceReports
-->

Scheduling performance is crucial. One of the bottlenecks is the API calls done during the scheduling cycle. 
The binding cycle is already asynchronous, but it would still be beneficial to re-evaluate whether the current model of busy-waiting goroutines is good long-term.

Making one universal approach for handling API calls in the kube-scheduler could allow these calls to be consistent and better control the number of dispatched goroutines.
Already asynchronous calls could also be migrated to this approach.

### Goals

<!--
List the specific goals of the KEP. What is it trying to achieve? How will we
know that this has succeeded?
-->

- P0: Make the scheduling cycle free of blocking API calls, i.e., make all API calls asynchronous.
- P0: Make the solution extendable for future use cases.
- P1: Skip some types of updates if they soon become irrelevant by consecutive updates.
- Nice to have: Prioritize high-importance updates (like binding) over low-importance ones if updates to the kube-apiserver get throttled.

### Non-Goals

<!--
What is out of scope for this KEP? Listing non-goals helps to focus discussion
and make progress.
-->

## Proposal

<!--
This is where we get down to the specifics of what the proposal actually is.
This should have enough detail that reviewers can understand exactly what
you're proposing, but should not include things like API designs or
implementation. What is the desired outcome and how do we measure success?.
The "Design Details" section below is for the real
nitty-gritty.
-->

There are a few ways to make API calls asynchronous.
They are introduced below to facilitate discussion and identify the most suitable solution.

These questions have to be answered:
1) Handling Pod rescheduling while waiting for the API call to complete
2) What component should handle the API calls

Also, races (collisions) between multiple API calls for a single object should be mitigated by the design.

### API calls categorization

Before selecting the best approach, the kube-scheduler's API calls have to be analyzed against the goals.
The following operations involve API calls during the main scheduling cycle and have to be made asynchronous (1st goal):

1) Updating a Pod status in `handleSchedulingFailure` when a Pod is unschedulable.
2) [Feature proposal: [#130668](https://github.com/kubernetes/kubernetes/issues/130668)] Updating the status of a Pod that is rejected by the `PreEnqueue` plugins in the scheduling queue.

These API calls are already asynchronous in their own ways:

3) [Feature proposal: [KEP-5278](https://github.com/kubernetes/enhancements/issues/5278)] Set `nominatedNodeName` in delayed binding scenarios.
4) Preemption - `ClearNominatedNodeName` and Pod eviction (made asynchronous by [KEP-4832](https://github.com/kubernetes/enhancements/issues/4832)).
5) Pod binding - is in the asynchronous binding phase.

All three of the above API calls could be migrated to the new mechanism.

In-tree plugins' operations that involve non-Pod API calls during scheduling and could be made asynchronous
(but don't have to be supported from the very beginning):

6) Volume binding - is in the `PreBind` phase, hence asynchronous.
7) DRA ResourceClaim deallocating in `PostFilter`.
8) DRA removing `ReservedFor` in `Unreserve`.
9) DRA ResourceClaims binding - is in the `PreBind` phase, hence asynchronous.
10) [Feature proposal: [KEP-5004](https://github.com/kubernetes/enhancements/issues/5278)] Extended resource feature will add `ResourceClaim` creation API call to the `PreBind` phase.
11) Other potential DRA features.

API calls relevance order in which they could cancel less relevant calls for the same Pod (3rd goal):

- Pod deletion caused by preemption (4) should cancel all Pod-based API calls for such a Pod.
- Pod binding (5) should cancel Pod status update API calls (1 - 3), because they are no longer relevant.
- Updating Pod status (1, 2) and setting `nominatedNodeName` (3) should cancel previous such updates.
  Both are calls to the `status` subresource of a Pod, so they should overwrite (merge) the previous calls properly
  when the newest status is stored in-memory.
- API calls for non-Pod resources (6 - 11) should be further analyzed as they are not likely to consider the Pod-based API calls,
  hence implementing those shouldn't block making (1 - 3) calls asynchronous.

There is no need to send two API calls for one Pod, because more relevant calls should override less relevant ones,
and status updates can be combined into one call.
There is no scenario in which two API calls, but for different Pods, or even **any** two API calls that do not involve the same object,
should be canceled or merged, so the relevance order between them should not be analyzed.

In terms of API call priority, the order might be different (4th goal):

- Pod binding (5) should have the highest priority as this is the main purpose of the kube-scheduler.
- Pod deletion caused by preemption (4) should also be important to free up space for high-priority Pods.
- Updating Pod status (1, 2) could be less important and called if there is space for it.
  It's worth considering if setting `nominatedNodeName` (3) should have the same priority or higher,
  because the higher delay might affect other components like Cluster Autoscaler or Karpenter.
- API calls for non-Pod resources (6 - 11) could be analyzed case by case, but are likely equally important to (5) or (4).


### 1: Handling Pod rescheduling while waiting for the API call to complete

There are multiple possible ways to handle such API calls, especially for Pod status updates.
Other (potential) use cases should also be considered when choosing the solution.
Three ways were analyzed, but the non-blocking approach, presented below, was selected.


#### Use advanced queue and don't block the Pod from being scheduled in the meantime

This approach allows the Pod to enter the scheduling queue and be scheduled again even before the status update API call completes, without blocking it.
This requires implementing advanced logic for queueing API calls in the kube-scheduler and migrating **all** Pod-based API calls done during scheduling to this method,
including the binding API call. The new component should be able to resolve any conflicts in the incoming API calls as well as parallelize them properly,
e.g., don't parallelize two updates of the same Pod. This requires [making the API calls queued in a separate component](#21-make-the-api-calls-queued-in-a-separate-component) or
[sending API calls through a kube-scheduler's cache](#22-send-api-calls-through-a-kube-schedulers-cache), presented below, to be implemented.

All Pod-based scenarios (1 - 5) could and should be implemented when choosing this approach.
Still, a single error reporting path for Pod condition updates could be considered but wouldn't be required.

Pros:
- Allows the Pod to be scheduled again even before the API call completes, which could reduce end-to-end Pod startup latency.
- Simplifies introducing new API calls to the kube-scheduler if the collision handling logic is configured correctly.

Cons:
- Requires implementing complex, advanced queueing logic.
- Necessitates migrating **all** Pod-based API calls to this method, but introduces unification, which could be desirable.
- Implementing collision resolution (e.g., for same-Pod updates) is complex, but could allow optimizing the number of API calls overall.


### 2: What component should handle the API calls

Another thing worth considering is how to indeed make the API calls asynchronous and what component should be responsible for this.


#### 2.1: Make the API calls queued in a separate component

To make asynchronous dispatching more advanced, a queueing in a separate component approach could be explored.
A new component might understand what the API calls are intended to do and eventually delay, skip, or merge them,
e.g., don't set `nominatedNodeName` when Pod binding is enqueued.
Initially, it could be a framework, which might be extended in the future, e.g., by introducing the possibility of setting delays.

However, it is questionable what should happen if two update API calls for the same Pod are enqueued.
See [API calls categorization](#api-calls-categorization) for more details.

Pros:
- Allows for advanced goroutine dispatching logic.
- Can potentially delay, skip, or merge API calls based on type (e.g., skip `nominatedNodeName` if binding is pending).
- All collisions could be resolved at the new component level, not relying on higher-level mechanisms.
- Allows supporting all scenarios without additional structures.
- Provides a framework that can be extended in the future.

Cons:
- Requires complex logic to handle potential conflicts between different update types for the same Pod.
- Needs a clear strategy for how to update the in-memory Pod object during scheduling.
- Requires extra steps to cache the updated objects.


#### 2.2: Send API calls through a kube-scheduler's cache

A second approach could be to have a consistent Pod state in the kube-scheduler itself first and then change it through the API.
This means that all API calls would have to go through the kube-scheduler's cache, change the Pod there, and after that, execute.
However, Pod updates might come from outside the kube-scheduler, e.g., a user changes the spec or something changes the status (if it is even possible).
This extended cache would have to merge the internal state of the Pod with the external state,
including the Pod update made by the kube-scheduler that will come as an event as well.
Now, the Pod object stored in the cache is based only on events that come to the kube-scheduler.

Another thing to think of is that the cache stores only the bound Pods. The rest of the Pods are stored in the scheduling queue,
so once again, API calls might need to go through the scheduling queue itself.

The cache proposal would still need to reuse some ideas of the first approach to achieve merging or skipping API calls.

Pros:
- Aims for a consistent internal state of the Pod within the kube-scheduler before calling the API, possibly simplifying conflict resolution.
- Allows for advanced goroutine dispatching logic.
- All collisions could be resolved at the cache, not relying on higher-level mechanisms.
- Can potentially delay, skip, or merge API calls based on type (e.g., skip `nominatedNodeName` if binding is pending),
  but merging would be possible if it stores additional data (what fields should be updated, etc.).

Cons:
- Requires the cache to handle and merge updates coming from both the kube-scheduler's internal actions and external API events.
- The cache currently only stores bound Pods, requiring integration with the scheduling queue for pending Pods.
- Complex logic is needed to handle external updates arriving while an internal update is pending or in progress.


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

## Design Details

<!--
This section should contain enough information that the specifics of your
change are understandable. This may include API specs (though not always
required) or even code snippets. If there's any ambiguity about HOW your
proposal will be implemented, this is the place to discuss them.
-->

This section describes the most important design details and three proposals based on the above ideas, that combine queueing, caching,
and having a separate component managing the API calls.

### Proposal A: Create a separate component managing API calls

![proposal A](proposal-A-separate-component.png)

Implementing an API queue could be made by adding a new component to the scheduler that will have to understand the API calls' details
as well as be (potentially) able to modify the cache (see dotted lines in the diagram). This approach would provide an extensible interface and understand the precedence of API calls.
Having a new component on its own would cause the cache to be less informed, i.e., not updated with API calls' details, providing the scheduler with outdated data.
It could be prevented by making an API queue a middleware between the event handler and a cache (dotted lines). This won't have to be fully implemented in the first place (only support a subset of use cases),
but will allow handling multiple cached storages that are currently in the scheduler, i.e., scheduler cache, nominator, DRA manager (`claimTracker`), and volume binding `AssumeCache`.

The interface for the new component could look like the following:

```go
type APICallType string

const (
	StatusUpdateCall APICallType = "status_update"
	BindingCall      APICallType = "binding"
	PreemptionCall   APICallType = "preemption"
	// PVCBinding etc.
)

// APICall describes the API call to be made and store all required data to make the call,
// e.g. fields that should be updated or object to be added/removed.
type APICall interface {
	// CallType returns an API call type
	CallType() APICallType
	// UID returns UID of an object that this call is related to
	UID() types.UID
	// Execute makes the actual API call
	Execute(client clientset.Interface) error
	// Merge merges two API calls with the same APICallType into one
	Merge(oldObj APICall) (bool, error)
  
	// Not required from the very beginning:

	// Update updates the obj using APICall details and returns the new version
	Update(obj any) (any, error)
}

type QueuedAPICall struct {
	APICall
	// OnFinish is a channel where the API call result is sent.
	// It allows to synchronize on the call completeness, e.g., in binding
	// and handle its result well.
	OnFinish chan<- error
}

type APIQueue struct {
	...
}

func (aq *APIQueue) Add(apiCall QueuedAPICall) error {
	// If API call for specific UID is already enqueued,
	// check the callType and skip, replace or merge the call depending on precedence.
	...
}

func (aq *APIQueue) Update(obj any) (any, error) {
	// Update the object using API call details if any is enqueued for its UID.
	...
}

func (aq *APIQueue) Run() {
	// Dispatch limited number of goroutines if queue is non empty.
	...
}
```

APIQueue would provide an `Add()` method would would be used to enqueue an API call that has to be executed.
`APICall` would provide all required methods to handle it, especially `Execute()` for running, `Merge()` for merging it with the same call type (e.g. `StatusUpdateCall`) that is already enqueued.
Supporting a cache would need adding `Update()` method that would take the object and update it with API call details (e.g., set NominatedNodeName in a Pod that will be soon updated by the call).
This updated object could be then stored in the cache, and having the call details would allow to know what fields would need to be changed if any future update occurs before the API call is executed.


### Proposal B: Make a scheduler's cache managing API calls

![proposal B](proposal-B-cache.png)

This approach differs from the previous one. Instead of creating a separate component, this would reuse the scheduler's cache to handle API calls.
Its advantage would be keeping a consistent state of the updated object in the scheduler and invisibly dispatching API calls if needed.
The largest caveat could be refactoring the scheduler's cache if non-Pod API call would have to be supported - the cache is currently split into multiple, more specialized caches,
i.e., scheduler cache, nominator, DRA manager (`claimTracker`), and volume binding `AssumeCache`. This means that the scheduler's cache might need to be extended by these use cases or
be able to support those custom storage options using some interfaces. Having a cache would still require storing additional metadata (details), similar to proposal A,
required to make the API calls and to be able to handle incoming updates from the event handler properly (store information about what the API call will change and be able to apply them on an updated object).

It would also require adding specialized methods to the cache to consume details needed to merge the calls and objects properly; for instance, the default `UpdatePod` method might not be useful,
because it would be too generic for our use cases. Supporting out-of-tree plugins might also be harder, as it would require making the cache extensible to store some custom objects
and somehow add new methods.


### Proposal C: Create a separate component managing API calls, but treat the cache as a middleware

![proposal C](proposal-C-cache-and-separate-component.png)

This proposal combines the strengths of proposals A and B by making a cache a middleware between scheduling/binding cycles, plugins, and event handlers.
This way, we could achieve the cache advantages of proposal B, while also allowing multiple caches to coexist.
Direct API queue operations would still be possible (e.g., for some out-of-tree plugins that don't need to cache any object).

The `APIQueue` design from proposal A could be largely reused in this approach. If an object needs to be modified, it would first go through the cache,
then be added to the API queue, and, based on the result, properly stored in the cache. This decoupled approach would allow adding a `StatusUpdateCall` through the scheduler's cache,
but for example, a `ResourceClaimUpdate` could go through the DRA manager, simplifying the adaptation of this KEP.

This proposal could be implemented as a second step extension of proposal A.


### Summary of API call management

Below is a summary of the steps in API call management that would be introduced by the proposals above.


#### Enqueueing a new API call

Having a separate component (`APIQueue` in proposal A and partially C) would make the API calls explicit to the caller by directly calling `Add()` on the `APIQueue`.
This means it will be visible from the scheduler or plugins that an API call will be sent, and various options could be easily passed.

Using a cache (proposal B and C), the API call will be hidden and executed implicitly when needed, based on the cache's internal logic.
It's questionable how to pass some options to the API call, e.g., an `OnFinish` channel or additional metadata. Error handling might also be less verbose for the caller.

Updating a cache with API call details would be similar across all proposals. Given the details, it would be possible to know precisely which fields will be updated by the API call.
Some `Update()` method could then apply these changes to an object, and the result could be stored in the cache. If any future update appears, it will be routed similarly.

In all proposals, if there isn't any API call already enqueued for a given object, its UID will be added to the queue that will later be consumed by the API calls runner.
In other scenarios, more advanced logic will be required. See the section below for more details.


#### Enqueueing another API call for the same object

Another API call for the same object could be enqueued, while the previous one is still waiting to be executed.
Based on API calls categorization, some updates might need to be merged. This logic has to be implemented and could be achieved similarly for all three proposals.
In general, given the API calls categorization, the calls could be simply merged by overwriting the details with the new ones, if applicable.
For `StatusUpdateCall`, merging will check if the `NominatedNodeName` or Pod condition changed and then overwrite these fields accordingly.

Skipping or overwriting less or more important API calls could be done by configuring an importance value for each `CallType`
and then making a decision based on comparison while adding a new API call. Not all API calls would need to implement their merging strategy.
Merging should also allow deciding if the API call should be removed from the queue when the update reverts a previous one that wasn't executed yet.

In proposal A and C, the merging strategy (`Merge()` method in `APICall`) would implement this merging logic.
In proposal B, some other configurable method would need to be designed to implement this.

Merging, overwriting, or skipping a call could get more complicated if the previous API call is already in flight.
See the [enqueueing an API call while a previous one is in-flight](#enqueueing-an-api-call-while-a-previous-one-is-in-flight) section for more details.
In proposal B, setting the merging strategy might be more complicated and could require providing custom logic through some interfaces.


#### Receiving object update through event handlers

An object might get updated or deleted externally in the meantime, while some API call is enqueued for the same object.
One such scenario might be setting `NominatedNodeName` by an external component (see [KEP-5278](https://github.com/kubernetes/enhancements/issues/5278)).
For Pod status updates themselves, making an update based on the old object wouldn't cause trouble,
because of the strategic merge patch used – it will just overwrite the Pod condition or `NominatedNodeName` if needed.
It is assumed that the scheduler should overwrite all such updates according to the actual needs,
and if it's not expected, custom logic could always be added using an `APICall` interface.

However, to support other potential use cases and have the newest object possible in the cache (proposals B and C, and optionally A),
merging the object received by event handlers with API call details should also be added.
It would work similarly to updating a cache in the section above.

It also should be defined how to handle such external updates if the API call is completed and the scheduler is waiting for the update to come in event handlers.
The `ResourceVersion` of the object could be used to distinguish it, i.e., apply the API call details
as long as the `ResourceVersion` of the received object is older than the version received by the update API call.
Doing so would require storing the `ResourceVersion` of the updated object received from the API call somewhere in the cache or the queue.


#### Executing the API call

In all three proposals, executing the API call could be done by having a goroutine (API calls runner) that will check if there is any goroutine available in the pool
(could be a configurable number) and it will try to fetch the first resource ID from a queue. Then, in the new goroutine, the API call for this resource will be executed, and after it completes,
it will be freed for the next call.


#### Enqueueing an API call while a previous one is in-flight

One other possible scenario is when an API call is executing (is in-flight) and a new API call for the same object wants to be added.
If both have the same call type, standard merging logic could be used, i.e., merge the new API call with the API call in flight.
If the new call is less relevant, it should be skipped, but if it's more relevant, it should be stored, and after the previous call ends,
the object UID should be re-added to the queue.


#### Waiting for the API call to finish

In some use cases, the caller would like to wait for the asynchronous API call to finish.
This could be achieved by passing an `OnFinish` channel along with the call that will receive the API call result (nil or error).
This way, already asynchronous calls like binding can be easily migrated to the new mechanism just by blocking on the call completion,
as binding is already asynchronous. This channel could be easily used with proposal A,
but proposals B and C would require passing it through cache methods, which could be less readable.


#### Retrying API calls

As API calls are getting overwritten or skipped, failure of one call might end up in losing multiple operations.
That's why, for retryable errors, it should be possible to re-enqueue the API call and try it again soon
Such logic could be explored, but having an `OnFinish` channel and handling errors by the caller should be enough for the actual use cases.


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

- `<package>`: `<date>` - `<test coverage>`

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

- <test>: <link to test coverage>

##### e2e tests

<!--
This question should be filled when targeting a release.
For Alpha, describe what tests will be added to ensure proper quality of the enhancement.

For Beta and GA, add links to added tests together with links to k8s-triage for those tests:
https://storage.googleapis.com/k8s-triage/index.html

We expect no non-infra related flakes in the last month as a GA graduation criteria.
-->

- <test>: <link to test coverage>

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

### Version Skew Strategy

<!--
If applicable, how will the component handle version skew with other
components? What are the guarantees? Make sure this is in the test plan.

Consider the following in developing a version skew strategy for this
enhancement:
- Does this enhancement involve coordinating behavior in the control plane and nodes?
- How does an n-3 kubelet or kube-proxy without this feature available behave when this feature is used?
- How does an n-1 kube-controller-manager or kube-scheduler without this feature available behave when this feature is used?
- Will any other components on the node change? For example, changes to CSI,
  CRI or CNI may require updating that component before the kubelet.
-->

## Production Readiness Review Questionnaire

### Feature Enablement and Rollback

###### How can this feature be enabled / disabled in a live cluster?

- [x] Feature gate (also fill in values in `kep.yaml`)
  - Feature gate name: `SchedulerAsyncAPICalls`
  - Components depending on the feature gate: kube-scheduler

###### Does enabling the feature change any default behavior?

<!--
Any change of default behavior may be surprising to users or break existing
automations, so be extremely careful here.
-->

###### Can the feature be disabled once it has been enabled (i.e. can we roll back the enablement)?

Yes.
The feature can be disabled in Beta version by restarting the kube-scheduler with the feature-gate off.

###### What happens if we reenable the feature if it was previously rolled back?

The kube-scheduler again starts to run API calls asynchronously.

###### Are there any tests for feature enablement/disablement?

Given it's a purely in-memory feature and enablement/disablement requires restarting the component 
(to change the value of the feature flag), having feature tests is enough.

### Rollout, Upgrade and Rollback Planning

###### How can a rollout or rollback fail? Can it impact already running workloads?

The partial failure in the rollout isn't there because the kube-scheduler is the only component to roll out this feature.
But, if upgrading the kube-scheduler itself fails somehow, new Pods won't be scheduled anymore,
while Pods, which are already scheduled, won't be affected in any case.

###### What specific metrics should inform a rollback?

<!--
What signals should users be paying attention to when the feature is young
that might indicate a serious problem?
-->

###### Were upgrade and rollback tested? Was the upgrade->downgrade->upgrade path tested?

No. This feature is an in-memory feature of the scheduler
and thus calculations start from the beginning every time the scheduler is restarted.
So, just upgrading it and upgrade->downgrade->upgrade are both the same.

###### Is the rollout accompanied by any deprecations and/or removals of features, APIs, fields of API types, flags, etc.?

No

### Monitoring Requirements

###### How can an operator determine if the feature is in use by workloads?

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

- [ ] Events
  - Event Reason: 
- [ ] API .status
  - Condition name: 
  - Other field: 
- [ ] Other (treat as last resort)
  - Details:

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

###### What are the SLIs (Service Level Indicators) an operator can use to determine the health of the service?

<!--
Pick one more of these and delete the rest.
-->

- [ ] Metrics
  - Metric name:
  - [Optional] Aggregation method:
  - Components exposing the metric:
- [ ] Other (treat as last resort)
  - Details:

###### Are there any missing metrics that would be useful to have to improve observability of this feature?

<!--
Describe the metrics themselves and the reasons why they weren't added (e.g., cost,
implementation difficulties, etc.).
-->

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

<!--
Look at the [existing SLIs/SLOs].

Think about adding additional work or introducing new steps in between
(e.g. need to do X to start a container), etc. Please describe the details.

[existing SLIs/SLOs]: https://git.k8s.io/community/sig-scalability/slos/slos.md#kubernetes-slisslos
-->

###### Will enabling / using this feature result in non-negligible increase of resource usage (CPU, RAM, disk, IO, ...) in any components?

<!--
Things to keep in mind include: additional in-memory state, additional
non-trivial computations, excessive access to disks (including increased log
volume), significant amount of data sent and/or received over network, etc.
This through this both in small and large cases, again with respect to the
[supported limits].

[supported limits]: https://git.k8s.io/community//sig-scalability/configs-and-limits/thresholds.md
-->

###### Can enabling / using this feature result in resource exhaustion of some node resources (PIDs, sockets, inodes, etc.)?

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

There were other alternatives considered in two topics:
1) Where and how to handle API calls during queueing and scheduling.
2) How to make the API calls asynchronous.

#### 1.1: Handle API calls in the scheduling queue

One possible approach is to send the API calls through a scheduling queue.
This allows delaying putting the pod into `unschedulablePods` after updating the pod.
This prevents race conditions from parallel updates of a single pod because, during the API call,
the pod is in-flight and thus not eligible for rescheduling.

A new method could be added to the `PriorityQueue`, which will take the function to be called asynchronously.
It should also make sure the pod is stored in `inFlightPods` to register the cluster events that will happen during the asynchronous part.
Calling `AddUnschedulableIfNotPresent` at the end ensures there won't be any race with the asynchronous pod update.
Because the pod would need to be in `inFlightPods` during the API call, the size of `inFlightEvents` might increase,
but as long as the API call executes quickly, there won't be a significant memory pressure.

Example solution could look like:

```go
// Author: @sanposhiho
func (p *PriorityQueue) AddUnschedulableAsync(pInfo *framework.QueuedPodInfo, fn func() error) {
	// Make sure the Pod is in inFlightPods before starting the goroutine

	go func() { // Or another way of dispatching
		// Run fn first 
		if err := fn(); err != nil { ... }

		// Push the pod back to the unschedQ after completing fn().
		p.AddUnschedulableIfNotPresent(...)
	}()
}
```

This way, we could cover pod status updates during the failure handler (1) and pod status updates for `PreEnqueue` plugins (2).
Asynchronous preemption (4) could be migrated to this approach by adding a possibility to return a function from `PostFilter` plugins in `PostFilterResult`
and calling this function probably in the failure handler together with the status update.

However, this method cannot be used for setting the `nominatedNodeName` scenario (3) because this operation occurs in the successful scheduling as well.
Therefore, additional effort would have to be made to specifically ensure that the `nominatedNodeName` doesn't collide with a potential status update.
Probably, before this status update in the failure handler, the code should try to cancel the set `nominatedNodeName` API call or wait until it finishes.
After that, it should proceed with setting the unschedulable status via the API. The binding call might similarly need to wait.

Another aspect to consider is how to dispatch the goroutines, as discussed in [how to make the API calls asynchronous](#2-how-to-make-the-api-calls-asynchronous) section.

Pros:
- Allows delaying putting unschedulable pods back to the queue until the API update completes.
- Prevents race conditions for parallel updates of a single pod by delaying the `AddUnschedulableIfNotPresent` call.
- Can easily cover status updates for both scheduling failures and `PreEnqueue` failures.
- Asynchronous preemption could be migrated to this approach, increasing consistency.

Cons:
- Handling of failures might not be consistent, requiring `AddUnschedulableAsync` to be called in two places.
- Delaying the `AddUnschedulableAsync` call increases pod queuing latency because the initial backoff timestamp is set there.
- Cannot be used for the `nominatedNodeName` scenario, requiring additional effort and separate handling.
- Might visibly increase the size of `inFlightEvents` if API calls are slow or if there are many calls.


#### 1.2: Handle API calls in the handleSchedulingFailure

Another approach could be to make all unschedulable status update API calls within `handleSchedulingFailure`.
This would make this handler the only error reporting path. Synchronous API calls within this handler could be made asynchronous,
but additional effort would be needed to prevent race conditions. This could be achieved by blocking the retries of the pod using `PreEnqueue`
(similar to asynchronous preemption) or by implementing advanced queueing logic.

This way, again, we could cover pod status updates during the failure handler (1),
but pod status updates for `PreEnqueue` plugins (2) will require more refactoring by either:
- Running a simplified scheduling cycle for pods that were rejected by the `PreEnqueue` to update the pod condition.
  This might negatively impact scheduling performance because a portion of the scheduling cycles will be spent for pods that are ultimately unschedulable
  Moreover, `PreEnqueue` plugins might also need to be called within this simplified scheduling cycle, 
  or alternatively, `PreFilter` plugins could implement the necessary PreEnqueue logic, duplicating it.
- Calling `handleSchedulingFailure` directly from the scheduling queue when a pod is rejected by the `PreEnqueue`. 
  This might be feasible, although it would create a circular dependency between the scheduling queue and the handler;
  however, it wouldn't have the same performance implications as the solution above.

Asynchronous preemption could also be migrated to this approach by exposing a function,
provided that the blocking behavior in `PreEnqueue` is consistent with the actual preemption blocking mechanism.

Again, for setting the `nominatedNodeName` scenario (3), this method cannot be used because this operation occurs in the successful scheduling as well. 
Therefore, additional effort would have to be made to specifically ensure that the `nominatedNodeName` doesn't collide with a potential status update.

Pros:
- Makes the failure handler the single path of reporting unschedulable status errors.
- Asynchronous preemption could potentially be migrated to this approach, increasing consistency.
- Pod would be immediately put into the scheduling queue, starting the backoff timer right away.

Cons:
- Requires additional effort to prevent race conditions for updates.
- Handling PreEnqueue rejections requires significant refactoring (implementing a `simplified scheduling cycle or direct `handleSchedulingFailure` call).
  - Simplified scheduling cycle for `PreEnqueue` rejections could impact performance and duplicate `PreEnqueue` logic.
  - Direct `handleSchedulingFailure` call would introduce circular dependency.
- Cannot be used for the `nominatedNodeName` scenario, requiring additional effort and separate handling.


#### 2.1: Just dispatch goroutines

With appropriate handling of races during updates, we could just dispatch goroutines with API calls.
A potential drawback is that we won't limit the number of these goroutines and won't be able to, e.g., delay the calls.
Limiting goroutines could still be easily achieved by having some group with a limited number of goroutines and a simple queue that will store pending calls.
Some delay might potentially appear due to side effects, especially when there will be problems with the kube-apiserver,
so some higher-level mechanism such as (1.1) or (1.2) would need to prevent pod update races.

Pros:
- Simple to implement if the appropriate race handling is chosen.
- Can easily be extended with a simple queue and worker pool to limit number of goroutines.

Cons:
- Does not inherently support delaying calls.
- Higher-level mechanisms (like 1.1 or 1.2) would be needed to prevent pod update races.
- `nominatedNodeName` scenario support would require more effort in (1.1) or (1.2).
- Prevents from further optimizations, e.g. can't merge two API calls.


## Infrastructure Needed (Optional)

<!--
Use this section if you need things from the project/SIG. Examples include a
new subproject, repos requested, or GitHub details. Listing these here allows a
SIG to get the process for these resources started right away.
-->
