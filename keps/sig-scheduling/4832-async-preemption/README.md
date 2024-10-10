# KEP-4832: Asynchronous Preemption

<!-- toc -->
- [Release Signoff Checklist](#release-signoff-checklist)
- [Summary](#summary)
- [Motivation](#motivation)
  - [Goals](#goals)
  - [Non-Goals](#non-goals)
- [Proposal](#proposal)
  - [Risks and Mitigations](#risks-and-mitigations)
    - [When kube-apiserver is unstable](#when-kube-apiserver-is-unstable)
- [Design Details](#design-details)
  - [Consideration to race condition](#consideration-to-race-condition)
    - [The pod2's scheduling is successful (pod2 is equal or lower priority than pod1)](#the-pod2s-scheduling-is-successful-pod2-is-equal-or-lower-priority-than-pod1)
    - [The pod2's scheduling is successful (pod2 is higher priority than pod1)](#the-pod2s-scheduling-is-successful-pod2-is-higher-priority-than-pod1)
    - [The pod2's scheduling is failed and starts the preemption (pod2 is equal or lower priority than pod1)](#the-pod2s-scheduling-is-failed-and-starts-the-preemption-pod2-is-equal-or-lower-priority-than-pod1)
    - [The pod2's scheduling is failed and starts the preemption (pod2 is higher priority than pod1)](#the-pod2s-scheduling-is-failed-and-starts-the-preemption-pod2-is-higher-priority-than-pod1)
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
  - [Introduce a new extension point](#introduce-a-new-extension-point)
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

This KEP proposes decoupling the API calls for the preemption from the scheduling cycle, to enhance the scheduling throughput of the scheduling failure scenarios.

## Motivation

The scheduler is basically only one in a cluster,
and hence scheduling throughput is the crucial metric for the scheduler.

The scheduler schedules Pods one by one within the scheduling cycle, 
and we basically try to reduce the API calls as much as possible to enhance the scheduling cycle throughput.

The binding cycle is the example for this motivation; 
1. The scheduling cycle decides where Pod should go to,
2. At the end of the scheduling cycle, the scheduler reserves the Node within the scheduler's cache so that next scheduling cycle will take the current pod into consideration.
3. The scheduling cycle ends and the binding cycle starts; the binding cycle is run asynchronously, and the scheduler starts the next scheduling cycle.

This flow allows us to decouple the API call to assign Pod to the Node from the scheduling cycle so that the API call doesn't block the scheduling throughput.

But, we have the similar problem with the preemption; the preemption is run at PostFilter extension point which is the part of the scheduling cycle.
The preemption has to make some API calls to update Pods' condition and delete Pods after all, which could block the scheduling throughput.

scheduler-perf [actually shows](https://github.com/kubernetes/kubernetes/blob/342da505bdefbd849b808cca3cb76c24a993025f/test/integration/scheduler_perf/config/performance-config.yaml#L641) currently the preemption scenario takes too long time, compared to others.

### Goals

- Improve scheduling throughput when pods require issuing preemptions by making API calls asynchronous

### Non-Goals

- Making the same enhancement for DRA is not a goal of this KEP because it's an under-construction feature yet.
  - If DRA maintainers want, technically they can along with this KEP. But, at least in this KEP, we don't discuss how.

## Proposal

The preemption plugin makes API calls for the preemption asynchronously after `PostFilter` extension point 
so that the scheduler can continue to other Pods' scheduling while making API calls for preemption.
After the preemption goroutine is done, the scheduling for the Pod that triggered the preemption will be retried.

### Risks and Mitigations

#### When kube-apiserver is unstable

When kube-apiserver is unstable and API calls at the preemption goroutine fails frequently,
the scheduler could make a non-optimal scheduling decision 
because the scheduler nominates pods at `PostFilter` though, those Pods won't be scheduled on nodes because the preemption API calls fail.

Let's say many mid-priority Pods are making the preemption API calls.
With the scheduler after this proposal, during the preemption goroutine for them are runnning, 
the scheduler assumes they'll be scheduled at the Nodes eventually
that the preemptions are targeting via `.Status.NominatedNodeName`.
So, other mid-priority or lower priority Pods' scheduling take those preemptor Pods into consideration, 
which is correct if the preemption goroutine finishes successful actually, while which results in non-best scheduling results otherwise.
(Higher priority Pods won't be affected; Pods can take place of reserved for lower priority Pods via `.Status.NominatedNodeName`)

But, in the first place though, when kube-apiserver is unstable, the scheduler doesn't behave well 
because it works with a lot of communication with kube-apiserver.
Even if the scheduler makes the best scheduling result, the binding API might fail after all.

So, we don't have to pay a special attention to this issue.

## Design Details

To achieve an asynchronous preemption, we will change the preemption plugin's implementation like the following:
1. The preemption PostFilter plugin calculates the preemption target and nominate the Pod for the Node. (We'll use `AddNominatedPod` API exposed from the scheduling framework to plugins.)
2. The preemption PostFilter plugin starts the goroutine to make API calls inside, and return success status (= not wait for the goroutine to finish).
3. The preemption plugin blocks the Pod while the preemption routine is in-progress, using PreEnqueue extension point, so that the target Pod won't be retried during this time.

Then, afterwards the preemption goroutine makes actual API calls to delete victime Pods and set `Pod.Status.NominatedNodeName`. 
If the preemption goroutine fails at some point, it reverts the nomination via `AddNominatedPod` with [`clearNominatedNode`](https://github.com/kubernetes/kubernetes/blob/f5c538418189e119a8dbb60e2a2b22394548e326/pkg/scheduler/schedule_one.go#L135).

If the preemption goroutine is complete, the preemption plugin ungates the Pod; 
the Pod is queued back to the queue with the Pod/delete event, and (hopefully) scheduled on the nominated node in the next scheduling cycle.

### Consideration to race condition

Thanks to the nomination at `PostFilter`, this new asynchronous preemption shouldn't make any race condition between several scheduling cycles.

Here, I'll discuss what happens in which scenario, and make sure there's no worry.

Let's say pod1 is during the preemption process (node1) at the preemption goroutine, the next scheduling cycle is scheduling pod2. 

#### The pod2's scheduling is successful (pod2 is equal or lower priority than pod1)

As I described above, pod1's `PostFilter` nominates pod1 for node1.

At the scheduling cycle, the scheduler takes such nominated pods that are equal or higher priority than pod1 into consideration;
meaning, pod2 won't rob pod1 of the place on node1.

#### The pod2's scheduling is successful (pod2 is higher priority than pod1)

Even though pod1 is nominated for the node, the scheduler allows pod2 to take node1, where the pod1's preemption made the space. 

Then, when pod1 comes back to the scheduling cycle, it may not be able to land on node1 because pod2 is scheduled there now.
It happens with both the current and this KEP's scheduler, so no issue here.

#### The pod2's scheduling is failed and starts the preemption (pod2 is equal or lower priority than pod1)

The preemption also takes nominated Pods into consideration when calculating the preemption target.

Therefore, if, coincidently, two preemptions for pod1 and pod2 select the same Node after all, 
then the preemption for pod2 should decide to make the space for pod1 and pod2.

So, we don't have to worry about two preemption targeting the same Node make any issue.

#### The pod2's scheduling is failed and starts the preemption (pod2 is higher priority than pod1)

The pod2's preemption ignores pod1's nomination for node1.

If, coincidently, two preemptions for pod1 and pod2 select the same Node after all, 
then the preemption for pod2 may just select the same preemption targets as pod1,
and when pod1 comes back to the scheduling cycle, it (probably) cannot be scheduled on node1 because of pod2.

But, this isn't an issue because the final result is completely the same as the current scheduler;
with the current scheduler, pod1 preempts some Pods on node1, then pod2's scheduling starts, pod2 takes node1, 
and when pod1 comes back to the scheduling cycle, it (probably) cannot be scheduled on node1 because of pod2.

### Test Plan

[x] I/we understand the owners of the involved components may require updates to
existing tests to make this code solid enough prior to committing the changes necessary
to implement this enhancement.

##### Prerequisite testing updates

##### Unit tests

- `/pkg/scheduler/framework/plugins/defaultpreemption/default_preemption.go`: `2024-09-07` - `85.4`
- `/pkg/scheduler/framework/preemption/preemption.go`: `2024-09-07` - `27.2`

Because the coverage for preemption.go is pretty low, we have to improve the testing there before the change for this KEP.

##### Integration tests

We have to add integration tests to make sure the asynchronous preemption is performed appropriately, 
especially in the scenarios listed in [Consideration to race condition](#consideration-to-race-condition).

##### e2e tests

We'll add test cases that multiple pods are trigger preemption.

### Graduation Criteria

#### Alpha

- Feature implemented behind a feature flag
- All tests mentioned in [Test Plan](#test-plan) are implemented.

#### Beta

- Gather feedback from users and fix reported bugs.
- Change the feature flag to be enabled by default.

#### GA

- Gather feedback from users and fix reported bugs.

### Upgrade / Downgrade Strategy

**Upgrade**

During the alpha period, users have to enable the feature gate `SchedulerAsyncPreemption` to opt in this feature.
This is purely in-memory feature for kube-scheduler, so no other special actions are required outside the scheduler.

**Downgrade**

Users need to disable the feature gate.

### Version Skew Strategy

This is purely in-memory feature for kube-scheduler, and hence no version skew strategy.

## Production Readiness Review Questionnaire

### Feature Enablement and Rollback

###### How can this feature be enabled / disabled in a live cluster?

- [x] Feature gate (also fill in values in `kep.yaml`)
  - Feature gate name: `SchedulerAsyncPreemption`
  - Components depending on the feature gate: kube-scheduler
- [ ] Other
  - Describe the mechanism:
  - Will enabling / disabling the feature require downtime of the control
    plane?
  - Will enabling / disabling the feature require downtime or reprovisioning
    of a node?

###### Does enabling the feature change any default behavior?

No. The feature is a performance optimization that affects every Pod that needs preemption, but there are no functional changes: the result of the preemption is the same.
But, like mentioned in [When kube-apiserver is unstable](#when-kube-apiserver-is-unstable), scheduling results could be different.

###### Can the feature be disabled once it has been enabled (i.e. can we roll back the enablement)?

Yes.
The feature can be disabled in Alpha and Beta versions
by restarting kube-scheduler with the feature-gate off.

###### What happens if we reenable the feature if it was previously rolled back?

The scheduler again starts to run PostFilter asynchronously.

###### Are there any tests for feature enablement/disablement?

Given it's purely in-memory feature and enablement/disablement requires restarting the component (to change the value of feature flag), 
having feature tests is enough.

### Rollout, Upgrade and Rollback Planning

<!--
This section must be completed when targeting beta to a release.
-->

###### How can a rollout or rollback fail? Can it impact already running workloads?

The partly failure in the rollout isn't there because the scheduler is only the component to rollout this feature. 
But, if upgrading the scheduler itself fails somehow, new Pods won't be scheduled anymore. 
If there's a bug in the preemption because of this enhancement, and also downgrading the scheduler fails somehow,
running Pods could be affected, for example, by being deleted by mistake (depending on bugs).

###### What specific metrics should inform a rollback?

Maybe something goes wrong with the preemption if `goroutines_duration_seconds{operation=preemption}` takes too long time.
Also, if `preemption_attempts_total` increases too much, then that might also imply some bugs around the preemption.

###### Were upgrade and rollback tested? Was the upgrade->downgrade->upgrade path tested?

No. This feature is an in-memory feature of the scheduler, 
and just upgrading it and upgrade->downgrade->upgrade are both the same.

###### Is the rollout accompanied by any deprecations and/or removals of features, APIs, fields of API types, flags, etc.?

No.

### Monitoring Requirements

###### How can an operator determine if the feature is in use by workloads?

This feature is used during all Pods' preemption if the feature gate is enabled.
You can see if the scheduler triggers any preemptions via `preemption_attempts_total` metric.

You can find Pods that have triggered the preemption by referring to `.Status.NominatedNodeName`,
and Pods that have been preempted by referring to their condition with `type: DisruptionTarget` and `reason: PreemptionByScheduler`.

###### How can someone using this feature know that it is working for their instance?

- [x] API .status
  - Other field: If `.Status.NominatedNodeName` of Pods is non-empty, they have experienced the preemption running asynchronously.

###### What are the reasonable SLOs (Service Level Objectives) for the enhancement?

- The failure rate of the preemption goroutine (`goroutines_execution_total{result=error, operation=preemption}`/`goroutines_execution_total{operation=preemption}`) should be < 0.01.

###### What are the SLIs (Service Level Indicators) an operator can use to determine the health of the service?

- [x] Metrics
  - Metric name: `goroutines_execution_total{result=error, operation=preemption}` 
  - Components exposing the metric: kube-scheduler

###### Are there any missing metrics that would be useful to have to improve observability of this feature?

- `goroutines_duration_seconds` (w/ label: `operation`): to observe how long each preemption goroutine takes to complete.
- `goroutines_execution_total` (w/ labels: `operation`, `result`): to observe how many preemption goroutines have failed.

### Dependencies

###### Does this feature depend on any specific services running in the cluster?

No.

### Scalability

###### Will enabling / using this feature result in any new API calls?

No. Just move the existing API calls from `PostFilter` into goroutines.

###### Will enabling / using this feature result in introducing new API types?

No.

###### Will enabling / using this feature result in any new calls to the cloud provider?

No.

###### Will enabling / using this feature result in increasing size or count of the existing API objects?

No.

###### Will enabling / using this feature result in increasing time taken by any operations covered by existing SLIs/SLOs?


No.

###### Will enabling / using this feature result in non-negligible increase of resource usage (CPU, RAM, disk, IO, ...) in any components?

The scheduler starts to run more goroutines in the preemption plugin, so maybe the CPU usage go up.

###### Can enabling / using this feature result in resource exhaustion of some node resources (PIDs, sockets, inodes, etc.)?

No.

### Troubleshooting

###### How does this feature react if the API server and/or etcd is unavailable?

In such cases, API calls for the preemption fails in the preemption goroutines.
But, the scheduler cannot perform not only the preemption, but anything essentially because it cannot get objects, bind Pods to Nodes, etc.

###### What are other known failure modes?

Nothing.

###### What steps should be taken if SLOs are not being met to determine the problem?

## Implementation History

- Sep 07, 2024: The initial KEP is submitted.

## Drawbacks

<!--
Why should this KEP _not_ be implemented?
-->

## Alternatives

### Introduce a new extension point

To make this kind of scenario easier to implement for other plugins, we can implement a new extension point `AsyncPostFilter`.
We calculate the preemption target and nominate the Pod for the Node at `PostFilter`, and then `AsyncPostFilter` starts asynchronously, in which the preemption plugin makes API calls for the preemption.

The Pod won't be queued back to the queue until `AsyncPostFilter` is done.

We don't go with this idea because we can implement the async preemption without introducing a new extension point.
Adding a new extension point unnecessarily may result in the regret in the future, and also we can implement it if it's really necessary.
