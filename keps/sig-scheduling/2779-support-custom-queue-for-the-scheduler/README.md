# KEP-2779: support custom queue for the scheduler

<!-- toc -->
- [Release Signoff Checklist](#release-signoff-checklist)
- [Summary](#summary)
- [Motivation](#motivation)
  - [Goals](#goals)
  - [Non-Goals](#non-goals)
- [Proposal](#proposal)
  - [Risks and Mitigations](#risks-and-mitigations)
- [Design Details](#design-details)
  - [Test Plan](#test-plan)
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
<!-- /toc -->

## Release Signoff Checklist

Items marked with (R) are required *prior to targeting to a milestone / release*.

- [ ] (R) Enhancement issue in release milestone, which links to KEP dir in [kubernetes/enhancements] (not the initial KEP PR)
- [ ] (R) KEP approvers have approved the KEP status as `implementable`
- [x] (R) Design details are appropriately documented
- [ ] (R) Test plan is in place, giving consideration to SIG Architecture and SIG Testing input (including test refactors)
  - [ ] e2e Tests for all Beta API Operations (endpoints)
  - [ ] (R) Ensure GA e2e tests for meet requirements for [Conformance Tests](https://github.com/kubernetes/community/blob/master/contributors/devel/sig-architecture/conformance-tests.md) 
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

Provide an interface so that users (custom scheduler developers) can implement a custom queue
of scheduler. With such, users can build their custom queue implementation at will, and plumb
into the scheduler seamlessly. This can satisfy the business needs such as sophisticated
pods sorting and internal queuing mechanics.

## Motivation

The current internal queue implementation ([scheduling_queue.go#PriorityQueue](https://github.com/kubernetes/kubernetes/blob/ee459b8969ed2abfed79a07d4ac9d41f13f18ce6/pkg/scheduler/internal/queue/scheduling_queue.go#L126))
works for most cases, but the limited interfaces exposure makes it hard to
implement a scheduler plugin for sophisticated requirements.

One requirement in terms of multi-tenancy support:  
There are many pods from 2 users (userA and userB), and the target is to ensure resource
usage (E.g. userA gets x CPUs) ratio between userA and userB is 1:1. Sorting the pods by
the resource usage in `Less` function doesn't work properly, because the resource usage for
userA/userB will be updated dynamically once a pod is selected and bound, while the
algorithm (heapsort) that used by the current queue to get the next pods depends on static
data.

Other requirements come from current semi-exposed functions/parameters:
For example, the interval of function flushUnschedulableQLeftover, although configurable,
is immutable after it gets initialized. A requirement is to adjust the interval dynamically
based on the pressure of pending workloads (e.g., length of queue or other metrics).

Given that the business needs may vary greatly, it'd be desirable to provide a replaceable
`SchedulerQueue` interface and plumbing mechanics. So that the developers can
implement their custom queue implementation, while the upstream maintainers focus
on keeping the core small and extensible, and thus only maintain the internal queue piece.

This KEP is not used to change the internal queue, or queue sorting plugin, quite the contrary,
that logic will be kept as is. The main idea is to give scheduler developers an option to manage the whole
queuing logic, including how to prioritize/pop/backoff/flush pods. It's all up to the scheduler developers.
To ensure the developers can have full control of the queuing logic, the whole interface of the internal
queue will be exposed instead of part of it that can only meet above examples. By exposing the whole interface,
we needn't update the interface that need to be exposed again and again for coming demands, this can save
developers' effort as they needn't raise a KEP for such requests, this can also save community's effort.

### Goals

1. Support custom queue implementation of the scheduler.
2. Roll out the design gradually. In phase 1, keep current internal queue implementation
intact.

### Non-Goals

1. Support custom rules for pod selection in the current internal queue of scheduler.
2. Enable scheduler developers to change the internal queue, or queue sorting plugin.

## Proposal

Provide an interface like current scheduler plugins, users can provide a
custom queue of scheduler at build time, then the kubernetes
scheduler will use this custom queue for pod management.

This is an extension of the current scheduler plugins, users can control
more details of the scheduler with this enhancement.

Pros:  
Users can get full control of the pod queuing logic, they can pop/re-queue/backoff pods with
custom logic.

The scheduler plugins design is enhanced, the custom queue and the extension
points can work together to meet more requests.

Cons:  
Users need to understand the details of the current queue which has 13
functions now, it is not easy to implement a custom queue.

### Risks and Mitigations

A poor-implemented queue may not function well, in both functionality and performance.
But the default scheduler works as before and thus won't be impacted.

## Design Details

The scheduler internal queue, as its name implies, was originally implemented for internal
usage. Due to that, some structs (like cache, Option) are not designed for external extension,
and the internal queue is initialized with specified parameters like LessFunc,
SharedInformerFactory, and Option by [factory.go#create](https://github.com/kubernetes/kubernetes/blob/f1f0183d2bbcde33024b2a05d6f39df32f11e037/pkg/scheduler/factory.go#L172). Although refactoring the internal queuing mechanics to be reusable/extensible is the eventual
goal, we'd like to approach that gradually, and thus is not the goal of phase 1 of this design.
In phase 1, we'd like to exercise the idea of "a replaceable scheduler queue" in a "minimal viable product"
manner.

A practical design for phase 1 is described as below:

1. Make `SchedulingQueue` a public interface which is private now in ([scheduling_queue.go](https://github.com/kubernetes/kubernetes/blob/ee459b8969ed2abfed79a07d4ac9d41f13f18ce6/pkg/scheduler/internal/queue/scheduling_queue.go#L126)). Users can choose to implement their own queue.

2. Provide a new function `WithCustomQueue`, so users can register the custom queue with this
function like other plugins.

    ```go
         command := app.NewSchedulerCommand(
             app.WithPlugin(coscheduling.Name, coscheduling.New),
             ......
       +     app.WithCustomQueue(customQueue.New),
         )
    ```

3. The registered custom queue will be passed to the scheduler
if it exists, or the current internal queue will be used.

4. Basic inputs for queue's initialization will not be changed, a new method `Init` is added in
the interface `SchedulingQueue`, users must implement it.

    ```go
      + func (p *PriorityQueue) Init(
      +        lessFn framework.LessFunc,
      +        informerFactory informers.SharedInformerFactory,
      +        opts ...framework.Option,
      + ) {
    ```

### Test Plan

- **Unit Tests**: All core changes must be covered by unit tests.
- **Integration Tests**: At least one integration test to craft a custom queue to exercise an end-to-end flow.
- **Benchmark Tests**: The performance benchmark test result is same as before if custom queue is not used.

### Graduation Criteria

#### Alpha -> Beta Graduation

- Users can implement their own custom queues.
- No user complaints regarding correctness.

#### Beta -> GA Graduation

- Allowing time for feedback to ensure that the new interface sufficiently expresses users requirements.

### Upgrade / Downgrade Strategy

N/A

### Version Skew Strategy

N/A

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

N/A

###### How can this feature be enabled / disabled in a live cluster?

- [x] Other
  - Describe the mechanism: Restart the scheduler with/without custom queue.
  - Will enabling / disabling the feature require downtime of the control
    plane? Yes
  - Will enabling / disabling the feature require downtime or reprovisioning
    of a node? Yes

###### Does enabling the feature change any default behavior?

Yes, the logic in queue of scheduler will be updated by user's logics.

###### Can the feature be disabled once it has been enabled (i.e. can we roll back the enablement)?

Yes.

###### What happens if we reenable the feature if it was previously rolled back?

This feature will work as we described before. The history operations have no
impact on the behavior.

###### Are there any tests for feature enablement/disablement?

N/A

### Rollout, Upgrade and Rollback Planning

N/A

###### How can a rollout or rollback fail? Can it impact already running workloads?

N/A

###### What specific metrics should inform a rollback?

N/A

###### Were upgrade and rollback tested? Was the upgrade->downgrade->upgrade path tested?

N/A

###### Is the rollout accompanied by any deprecations and/or removals of features, APIs, fields of API types, flags, etc.?

No

### Monitoring Requirements

###### How can an operator determine if the feature is in use by workloads?

This is a feature of scheduler, and operator can decide which scheduler will
be used.

###### How can someone using this feature know that it is working for their instance?

- [x] Events
  - Event Reason: the custom queue can raise events for some actions
- [ ] API .status
  - Condition name: 
  - Other field: 
- [ ] Other (treat as last resort)
  - Details:

###### What are the reasonable SLOs (Service Level Objectives) for the enhancement?

The performance is only impacted by the logics in custom queue, not impacted by the
design structure.

###### What are the SLIs (Service Level Indicators) an operator can use to determine the health of the service?

- [ ] Metrics
  - Metric name:
  - [Optional] Aggregation method:
  - Components exposing the metric:
- [x] Other (treat as last resort)
  - Details: the performance is only impacted by the custom queue's logic.

###### Are there any missing metrics that would be useful to have to improve observability of this feature?

No

### Dependencies

N/A

###### Does this feature depend on any specific services running in the cluster?

No

### Scalability

N/A

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

### Troubleshooting

The user who provides custom queue is responsible for providing troubleshooting messages/events.

###### How does this feature react if the API server and/or etcd is unavailable?

It depends on the related logics in custom queue.

###### What are other known failure modes?

No

###### What steps should be taken if SLOs are not being met to determine the problem?

Using the current queue as a workaround and check the logics in custom queue.

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

For the multi-tenancy support case the mentioned in `motivation` section, with this KEP, scheduler developers
can add some sub-queues, one sub-queue is for one user and put the pods for this user into it, when the
scheduler wants to pop a pod, the new logic will go through the sub-queues, get the most under-used sub-queue
(by comparing the `target resources by ratio`/`allocated resources` of every sub-queue) and pop one pod from it.
Instead of comparing all the pending pods, we only need to compare the sub-queues here, which is much faster.
The most important benefit for the developers is that they have full control of this logic, they can update the
logic at anytime at will, needn't talk with k8s community at all.

An alternative is to enhance the current queue to meet this request, resorting all the pending pods when trying
to get a pod to pop. The drawback is that this special change is only for the multi-tenancy case, and we need a way
to control the impact (E.g., add a flag), the users who needn't multi-tenancy support cannot benefit from it.
There will be many similar requests in the future as different customers have different requests, to meet all the
requests, the changes will make the logic of internal queue complex and hard to maintain.

Another alternative is to have a high level controller to suspend a job and don't let it go unless the resource
sharing policy will not be broken, this is a good solution to support multi-tenancy case at job level, and needn't
a KEP to update the current design. The drawback is that all the pods in a job will be impacted by this way, it
cannot handle the case that only some pods in a job need to be touched. The controller needs to predict whether
a job can run or not before resuming a job, or the resource sharing may be broken, and preemption need to be
triggered to re-balance the resource sharing, some workload will be interrupted, this is another drawback.

Another alternative is to make the pod that will break the resource sharing policy with `Unschedulable` before
binding it with a node. This is the ideal solution as it is very clean and no negative impact. The drawback is
that making such a decision needs to know whether other pods can run or not, we need a whole picture of all the
pods and nodes, the relations for pods-to-pods, pods-to-nodes and nodes-to-nodes, and how to adjust the
decision once there are changes (E.g., a node cannot be accessed anymore), the logic is complex, and the
performance should be low when there are thousands of pods and nodes. Another drawback is that this solution may
conflict with developer's logic in scheduler plugin. This solution's decision should not be changed by other
logic, or the decision is not correct, while the developer's logic in the plugin can have impact on pod's status
(E.g., mark a `Schedulable` pod in the decision as `Unschedulable`), the decision can be changed.
