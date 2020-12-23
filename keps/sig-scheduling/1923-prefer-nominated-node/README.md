# KEP-1923: Prefer Nominated Node

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
- [x] (R) Graduation criteria is in place
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

This KEP proposes to change the scheduling cycle such that nominated node of a pod is evaluated first
and schedule the pod on that node if it fits. If the nominated node doesn't fit the pod, only then the
scheduling cycle continues with the standard logic of evaluating the rest of the nodes in the cluster.

## Motivation

If the scheduler fails to fit an incoming pod on any node, it will try to preempt lower priority pods
running on a selected node to make room for the pod. The name of this node will be set in the
pod's `.status.nominatedNodeName`.

The Node is called *Nominated* to indicate the intent for the Pod to be scheduled on it once preemption
of other Pods finishes. However, the Pod's `.status.nominatedNodeName` information is not fully utilized
in the Pod's following scheduling attempts.

Pod scheduling is split into two phases, the scheduling cycle and the binding cycle, the scheduling cycle
primarily includes filtering and scoring.

When preemption happens in a previous scheduling cycle, there is a high chance that the nominated node is
the *only* node that satisfies the filters for the unscheduled Pod that triggered preemption.

In real production environment, pods can have different priorites due to business needs, the preemption
could happen to make sure higher priority pods could get scheduled.

In cluster with large number of computing nodes, evaluating all nodes when scheduling a pod is time consuming.

### Goals

Prefer scheduling a pod to its `.status.nominatedNodeName` if set, if the nominated node doesn't fit the pod,
the scheduling cycle will continue to evaluate the rest of the nodes in the cluster just like we do today.


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
   first if the incoming pod has the `.status.nominatedNodeName` defined and the feature gate is enabled.

2. In case the nominated node doesn't suit for the incoming pod anymore, get `err` from `findNodesThatPassFilters` where
   `NominatedNode` is firstly evaluated, the `err` will be padded with more information to tell that scheduler is evaluating
   the feasibility of `NominatedNode` and failed on that node.

   If no error is returned but `NominatedNode` cannot pass all the filtering, this is possibly caused by the resource that
   claims to be removed but has not been fully released yet.

   For both of above cases, scheduler will continue to evaluate the rest of nodes to check if there is any node already
   available for the coming pod.

   Scheduler will retry until matching either of the following cases,
   - `NominatedNode` eventually released all the resource and the preemptor pod can be scheduled on that node.
   - Another node in the cluster released enough resources and pod get scheduled on that node instead.
     [Discuss] Should scheduler clear the `NominatedNode` in this case?
   - Resource cannot be released on the `NominatedNode` and no other candidate node could be found in the cluster, this will
     be covered by [issue 95752](https://github.com/kubernetes/kubernetes/issues/95752).
     

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
    - Feature gate name: PreferNominatedNode
    - Components depending on the feature gate: kube-scheduler

* **Does enabling the feature change any default behavior?**
  Yes. The pod with the nominated node set will be evaluated first in any scheduling cycle,
  this is only the default process logic that is handled by scheduler, end-user will not
  and need not aware of any difference.

* **Can the feature be disabled once it has been enabled (i.e. can we roll back
  the enablement)?**
  Yes. This could be opt-out by the feature gate, once switch off, it will fall back
  to original behavior.

* **What happens if we reenable the feature if it was previously rolled back?**
  Nothing needs to be aware of for the end-user.

* **Are there any tests for feature enablement/disablement?**
  unittest will cover this.


## Implementation History

- 2020-09-29: Initial KEP sent out for review https://github.com/kubernetes/enhancements/pull/2026
