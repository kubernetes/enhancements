# KEP-1923: Try Nominated Node First

<!-- toc -->
- [Release Signoff Checklist](#release-signoff-checklist)
- [Summary](#summary)
- [Motivation](#motivation)
  - [Goals](#goals)
- [Proposal](#proposal)
  - [User Stories (Optional)](#user-stories-optional)
  - [Notes/Constraints/Caveats (Optional)](#notesconstraintscaveats-optional)
- [Design Details](#design-details)
  - [Implementation Details](#implementation-details)
  - [Alternatives](#alternatives)
  - [Test Plan](#test-plan)
  - [Graduation Criteria](#graduation-criteria)
    - [Alpha (v1.21):](#alpha-v121)
- [Production Readiness Review Questionnaire](#production-readiness-review-questionnaire)
  - [Feature Enablement and Rollback](#feature-enablement-and-rollback)
- [Implementation History](#implementation-history)
<!-- /toc -->

## Release Signoff Checklist

Items marked with (R) are required *prior to targeting to a milestone / release*.

- [x] (R) Enhancement issue in release milestone, which links to KEP dir in [kubernetes/enhancements] (not the initial KEP PR)
- [ ] (R) KEP approvers have approved the KEP status as `implementable`
- [x] (R) Design details are appropriately documented
- [ ] (R) Test plan is in place, giving consideration to SIG Architecture and SIG Testing input
- [ ] (R) Graduation criteria is in place
- [ ] (R) Production readiness review completed
- [ ] Production readiness review approved
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

If the scheduler fails to fit an incoming pod on any node, the scheduler will try to preempt lower
priority pods running on a selected node to make room for the pod. The name of this node will be set
in the pods' `pod.Status.NominatedNodeName`.

The Node is called *Nominated* to indicate the intent for the Pod to be scheduled on it once preemption
of other Pods finish. However, the `Pod.status.nominatedNodeName` information is not directly used in
the Pod's following scheduling attempts.

Pod scheduling is split into two phases, the scheduling cycle and the binding cycle, the scheduling cycle
primarily includes filtering and scoring.

When preemption happens in a previous scheduling cycle, there is a high chance that the nominated node is
the *only* node that satisfies the filters for the unscheduled Pod that triggered preemption.

This KEP proposes to change the scheduling cycle such that nominated node of a pod is evaluated first
and schedule the pod on that node if it fits. If the nominated node doesn't fit the pod, only then the
scheduling cycle continues with the standard logic of evaluating the rest of the nodes in the cluster.

## Motivation

In real production environment, pods can have different priorites due to business needs, the preemption
could happen to make sure higher priority pods could get scheduled.

In cluster with large number of computing nodes, evaluating all nodes when scheduling a pod is time consuming.

### Goals

In the case where `pod.Status.NominatedNodeName` is set for an incoming pod, the scheduler will evaluate the
nominated node first; if the nominated node doesn't fit the pod, the scheduling cycle will continue to evaluate
the rest of the nodes in the cluster just like we do today.


## Proposal

### User Stories (Optional)

Users want faster scheduling. Since it is highly likely the pod will only fit on the nominated node, the improvement
in scheduling latency will come at negligible cost (the cost being placing the pod on a less optimal node).

### Notes/Constraints/Caveats (Optional)

When this feature is enabled the preemptor Pod might not be dispatched to the best candidated node in some corner case,
e.g. another node releases the resources and becomes the best candidate while the victim pods got removed from the
nominated node.

## Design Details

### Implementation Details

1. In filtering phase, which is currently implemented in the method of `findNodesThatFitPod`, check the nominated node
   first if the incoming pod has the `pod.Status.NominatedNodeName` defined and the feature gate is enabled.

2. In case the nominated node doesn't suit for the incoming pod anymore, return `ErrNominateNode`
   instead of `core.FitError`, because this will give scheduler a chance to clean up the nominated
   node from the pod and find a new node to schedule instead of preemption again (might already have
   another node available for scheduling during the period). A fresh new scheduling cycle will be
   started later.

   A new error `ErrNominateNode` should be defined to describe what's going wrong on the nominated node.

   If the nominated node doesn't suit for the pod anymore, the scheduling failure will be recorded
   and the `updatePod` will be called, here we change the logic to update the pod as long as the
   parameter `nominatedNode` is different with what pod holds in `pod.Status.NominatedNodeName`.
   In this case, parameter `nominatedNode` is an empty string so that the nominated node will be
   cleaned from the pod and the pod will be moved to the active queue. It lets scheduler find another
   place for the pod in the next scheduling cycle.

### Alternatives

- Should keep trying on the nominated node in case the failure of scheduling?

  This is the case when the pod deletion is still on the fly, the deletion of preemptor pods has
  been triggered and sent to apiserver but has not actually been deleted by `kubelet` or container runtime.

  Here are several things we need to consider, and this is why this approach is not adopted,

  1. Keep trying in this scheduling cycle until the deletion is done

     This will block the scheduler and we never know when the deletion will be done, something might
     block this for a long time, for example, docker service is down and cannot get recovered.

  2. Reserve the `pod.Status.NominatedNodeName` for the preemptor pod, so that the nominated node will be
     tried in the following scheduling cycle (not clean up the `nominatedNode` on failure)

     This will not resolve the issue mentioned above either, this will generate an infinite looping on the
     nominated node.

  3. There are other cases should be considered beside the pod deletion, which cause the nominated node
     not able to fit for the preemptor anymore, for example, nominated node becomes unschedulable, another
     node in the cluster releases enough room for the coming pod, topology update due to pod deletion on another
     node which makes the nominated node not fits for `PodTopologySpread` filter anymore.

     All those cases require us to start a fresh new scheduling cycle and find a better one instead of the
     selected nominated node in previous cycle.


- Should go on the preemption evaluation and try the nominated node there? update the nominated node if necessary.

  In order to continue the preemption on the failure on the nominated node, scheduler should return `core.FitError`
  so that preemption will continue.

  1. [Debatable]: For the issue #3 mentioned above, assume it will continue to go on the preemption evaluation,
     if there is another candidate node which doesn't preempt any victim pods, this node should be chosen as the
     new nominated node, it is also true if this is done in the new scheduling cycle, the same node will be chosen
     for both approaches, there is nearly no major difference, but the case like this looks more like should be handled
     by the normal scheduling process instead of pod preemption phase, this is sound like anti-pattern of what is the
     preemption designed.

  2. If the nominated node doesn't fit due to the victim pods deletion is still on the fly, and nothing else is
     changed, the nominated node will be chosen again either it goes to preemption evaluation or after the normal
     scheduling cycle following by a preemption evaluation.
     We got more chance to finish the deletion for the latter case, and the nominated node will be chosen
     in the normal scheduling cycle or as the selected node in the following preemption evaluation phase.
     pod deletion might be triggered again on that victim pod/pods if finally go to the preemption, there is no harm
     to do that.
     But we need to note the shorter time might be needed for the former case.

### Test Plan

Following tests will be covered or considered:

- **Unit Tests**: All core changes must be covered by unit tests.
- **Integration Tests**: Integration test will be provided if necessary, for example,
  - enable the feature
  - preempt the victim pods on the nominated node
  - check pod will be scheduled on the nominated node
- **Benchmark Tests**: A benchmark test which compares the performance before and after the change.
  The performance improvement is visible by benchmark of `scheduling_algorithm_predicate_evaluation_seconds`.
  Other benchmark will be created on-demand along with the code review process.


### Graduation Criteria

#### Alpha (v1.21):

- [ ] New feature gate proposed to enable the feature.
- [ ] Implementation of the new feature in scheduling framework.
- [ ] Test cases mentioned in the [Test Plan](#test-plan).

## Production Readiness Review Questionnaire

### Feature Enablement and Rollback

_This section must be completed when targeting alpha to a release._

* **How can this feature be enabled / disabled in a live cluster?**
  - [x] Feature gate (also fill in values in `kep.yaml`)
    - Feature gate name: TryNominatedNodeFirst
    - Components depending on the feature gate: kube-scheduler

* **Are there any tests for feature enablement/disablement?**
  unittest will cover this.


## Implementation History

- 2020-09-29: Initial KEP sent out for review https://github.com/kubernetes/enhancements/pull/2026
