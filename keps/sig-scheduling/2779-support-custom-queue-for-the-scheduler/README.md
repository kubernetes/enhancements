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

Provide an interface so that users can implement a custom queue of scheduler.
With such, users can build their custom queue implementation at will, and plumb into
the scheduler seamlessly. This can satisfy the business needs such as sophisticated
pods sorting and internal queuing mechanics.

## Motivation
The current internal queue implementation ([scheduling_queue.go#PriorityQueue]
(https://github.com/kubernetes/kubernetes/blob/ee459b8969ed2abfed79a07d4ac9d41f13f18ce6/pkg/scheduler/internal/queue/scheduling_queue.go#L126))
works for most cases, but the limited interfaces exposure makes it hard to
implement a scheduler plugin for sophisticated requirements.

One requirement in terms of multi-tenancy support:  
There are many pods from 2 users (userA and userB), and the target is to ensure resource
usage (E.g. userA gets x CPUs) ratio between userA and userB is 1:1. Sorting the pods by
the resource usage in `Less` function doesn't work properly, because the resource usage for 
userA/userB will be updated dynamically once a pod is selected and bound, while the 
algorithm (heapsort) that used by the current queue to get the next pods depends on static
data.

Another requirement in terms of custom function support:  
For example, the interval of function flushUnschedulableQLeftover cannot be updated dynamically, while
user want it to be shorter or longer at different times as shorter interval can get pods to be scheduled
faster and longer one can get less logs flushed.

Given that the business needs may vary greatly, it'd be good to provide a replaceable
`SchedulerQueue` interface and plumbing mechanics. So that the developers can
implement their custom queue implementation, while the upstream maintainers focus
on keeping the core small and extensible, and thus only maintain the internal queue piece

### Goals

Support custom queue of the scheduler.

### Non-Goals

Support custom rules for pod selection in the current internal queue of scheduler.

## Proposal

Provide an interface like current scheduler plugins, users can provide a
custom queue of scheduler and register it, then the kubernetes
scheduler will use this custom queue for pod management.

This is an extension of the current scheduler plugins, users can control
more details of the scheuler with this enhancement.

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

The codes will be updated as following:

1. User implements a custom queue with the interface SchedulingQueue,
the name is myQueue, and registers it

```
	command := app.NewSchedulerCommand(
		app.WithPlugin(coscheduling.Name, coscheduling.New),
                        ......
	+	app.WithCustomQueue(myQueue.New),
	)
```

2. The registered custom queue will be recorded by updated Registry struct.

```
  type Registry struct {
    Pf          map[string]PluginFactory
    CustomQueue schedulingQueue.SchedulingQueue
  }
```

3. In function (c *Configurator) create(), pass the custom queue to the scheduler
if it exists.
```
  if c.registry.CustomQueue != nil {
      podQueue = c.registry.CustomQueue
  } else {
      podQueue = schedulingQueue.NewSchedulingQueue()
  }
```

4. Basic inputs for queue's initialization will not be changed, a new interface `Init` is added in
SchedulingQueue, users must implement it.

```
-func NewPriorityQueue(
+func (p *PriorityQueue) Init(
 	lessFn framework.LessFunc,
 	informerFactory informers.SharedInformerFactory,
 	opts ...Option,
)
```

5. For the scheduler internal packages under ([internal] (https://github.com/kubernetes/kubernetes/tree/b6c75bee15e150628fcc240ab32dba6190d254e4/pkg/scheduler/internal)), queue and heap will be moved out of this folder so that users can access and use the structures inside them, cache and parallelize will not be touched.

### Test Plan

- **Unit Tests**: All core changes must be covered by unit tests.
- **Integration Tests**: One integration test for the custom queue.
- **Benchmark Tests**: The performance benchmark test result is same as before if custom queue is not used.

### Graduation Criteria

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

An alternative is to enhance the current queue to meet some requests. The drawback
is that we cannot handle all the potentail requests as the effort is big, we should 
give the ball to the users, and they can do what they want.
