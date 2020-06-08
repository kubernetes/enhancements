# Add NonPreempting Option For PriorityClasses

## Table of Contents

<!-- toc -->
- [Release Signoff Checklist](#release-signoff-checklist)
- [Summary](#summary)
- [Motivation](#motivation)
  - [Goals](#goals)
  - [Non-Goals](#non-goals)
- [Proposal](#proposal)
  - [User Stories](#user-stories)
  - [Risks and Mitigations](#risks-and-mitigations)
- [Design Details](#design-details)
  - [Testing Plan](#testing-plan)
  - [Graduation Criteria](#graduation-criteria)
    - [Alpha (v1.15):](#alpha-v115)
    - [Beta (v1.19):](#beta-v119)
- [Production Readiness Review Questionnaire](#production-readiness-review-questionnaire)
  - [Feature enablement and rollback](#feature-enablement-and-rollback)
  - [Rollout, Upgrade and Rollback Planning](#rollout-upgrade-and-rollback-planning)
  - [Scalability](#scalability)
  - [Troubleshooting](#troubleshooting)
- [Implementation History](#implementation-history)
<!-- /toc -->

## Release Signoff Checklist

Items marked with (R) are required *prior to targeting to a milestone / release*.

- [x] (R) Enhancement issue in release milestone, which links to KEP dir in [kubernetes/enhancements] (not the initial KEP PR)
- [x] (R) KEP approvers have approved the KEP status as `implementable`
- [x] (R) Design details are appropriately documented
- [x] (R) Test plan is in place, giving consideration to SIG Architecture and SIG Testing input
- [x] (R) Graduation criteria is in place
- [x] (R) Production readiness review completed
- [ ] Production readiness review approved
- [x] "Implementation History" section is up-to-date for milestone
- [x] User-facing documentation has been created in [kubernetes/website], for publication to [kubernetes.io]
- [x] Supporting documentation e.g., additional design documents, links to mailing list discussions/SIG meetings, relevant PRs/issues, release notes

[kubernetes.io]: https://kubernetes.io/
[kubernetes/enhancements]: https://git.k8s.io/enhancements
[kubernetes/kubernetes]: https://git.k8s.io/kubernetes
[kubernetes/website]: https://git.k8s.io/website

## Summary

[PriorityClasses](https://kubernetes.io/docs/concepts/configuration/pod-priority-preemption/) are a GA feature as on 1.14,
which impact the scheduling and eviction of pods.
Pods are be scheduled according to descending priority.
If a pod cannot be scheduled due to insufficient resources,
lower-priority pods will be preempted to make room.

This proposal makes the preempting behavior optional for a PriorityClass,
by adding a new field to PriorityClasses,
which in turn populates PodSpec.
If a pod is waiting to be scheduled,
and it does not have preemption enabled,
it will not trigger preemption of other pods.

## Motivation

Allowing PriorityClasses to be non-preempting is important for running batch workloads.

Batch workloads typically have a backlog of work,
with unscheduled pods.
Higher-priority workloads can be assigned a higher priority via a PriorityClass,
but this may result in pods with partially-completed work being preempted.
Adding the non-preempting option allows users to prioritize the scheduling queue,
without discarding incomplete work.

### Goals

- Add a boolean flag to PriorityClasses,
to enable or disable preemption for pods of that PriorityClass.

### Non-Goals

* Protecting pods from preemption. PodDisruptionBudget should be used.

## Proposal

Add a Preempting field to both PodSpec and PriorityClass.
This field will default to true,
for backwards compatibility.

If Preempting is true for a pod,
the scheduler will preempt lower priority pods to schedule this pod,
as is current behavior.

If Preempting is false,
a pod of that priority will not preempt other pods.

Setting the Preempting field in PriorityClass provides a straightforward interface,
and allows ResourceQuotas to restrict preemption.

PriorityClass type example:
```
type PriorityClass struct {
	metav1.TypeMeta
	metav1.ObjectMeta
	Value int32
	GlobalDefault bool
	Description string
	Preempting *bool // New option
}
```

The Preempting field in PodSpec will be populated during pod admission,
similarly to how the PriorityClass Value is populated.
Storing the Preempting field in the pod spec has several benefits:
* The scheduler does not need to be aware of PiorityClasses,
as all relevant information is in the pod.
* Mutating PriorityClass objects does not impact existing pods.
* Kubelets can set Preempting on static pods.

PodSpec type example:
```
type PodSpec struct {
    ...
    Preempting *bool
    ...
}
```

This feature should be gated in alpha, provisionally under the gate `NonPreemptingPriority`.

Documentation must be updated to reflect the new feature,
and changes to PriorityClass/PodSpec fields.

### User Stories
A user is running batch workloads on a cluster.
The user has a high-priority job,
that they wish to schedule before other workloads in the queue.
As the user does not want to preempt running batch workloads and discard work,
the user creates the new workload with a high-priority,
non-preempting PriorityClass.
The new workload's pods are scheduled ahead of the queue,
without disrupting running workloads.

* Users are able to run preempting and non-preempting workloads in a stable manner,
and are not requesting additional changes.
* The feature has been stable and reliable in at least 2 releases.
* Adequate documentation exists for preemption and the optional field.
* Test coverage includes non-preempting use cases.
* Conformance requirements for non-preempting PriorityClasses are agreed upon.

### Risks and Mitigations

The new feature may malfuction,
or existing preemption functionality may be impaired.
New tests (covering both nonpreepting workloads and mixed workloads),
and the existing preempting PriorityClass tests should be used to prove stability.

## Design Details
### Testing Plan
Add detailed unit and integration tests for nonpreempting workloads.

Add basic e2e tests, to ensure all components are working together.

Ensure existing tests (for preempting PriorityClasses) do not break.

### Graduation Criteria
#### Alpha (v1.15):

- [x] Support NonPreemptingPriority in PriorityClasses

#### Beta (v1.19):

- [ ] Add integration test for NonPreemptingPriority.
- [ ] Graduate NonPreemptingPriority to Beta.
- [ ] Update documents to reflect the changes.



## Production Readiness Review Questionnaire

### Feature enablement and rollback
* **How can this feature be enabled / disabled in a live cluster?**
  - [x] Feature gate
    - Feature gate name: NonPreemptingPriority
    - Components depending on the feature gate:
      - kube-apiserver
      - kube-scheduler

* **Does enabling the feature change any default behavior?**
  No

* **Can the feature be disabled once it has been enabled (i.e. can we rollback
  the enablement)?**
  Yes, the feature can be disabled if the PreemptionPolicy isn't set.

* **What happens if we reenable the feature if it was previously rolled back?**
  If we reenable the feature, the Pod with high priority and NonPreemptionPolicy will be eligible to preempt other pods with low priority when cluster resources are tight.

* **Are there any tests for feature enablement/disablement?**
  No

### Rollout, Upgrade and Rollback Planning
* **How can a rollout fail? Can it impact already running workloads?**
  The scheduler errors and exits during start up. Existing workloads are not
  affected.

* **What specific metrics should inform a rollback?**
  N/A.

* **Were upgrade and rollback tested? Was upgrade->downgrade->upgrade path tested?**
  N/A.

* **Is the rollout accompanied by any deprecations and/or removals of features?**
  N/A.

### Scalability
* **Will enabling / using this feature result in any new API calls?**
  No

* **Will enabling / using this feature result in introducing new API types?**
  No

* **Will enabling / using this feature result in any new calls to cloud
  provider?**
  No

* **Will enabling / using this feature result in increasing size or count
  of the existing API objects?**
  No

* **Will enabling / using this feature result in increasing time taken by any
  operations covered by [existing SLIs/SLOs][]?**
  No

* **Will enabling / using this feature result in non-negligible increase of
  resource usage (CPU, RAM, disk, IO, ...) in any components?**
  No

### Troubleshooting
* **How does this feature react if the API server and/or etcd is unavailable?**
  N/A.

* **What are other known failure modes?**
  N/A.

* **What steps should be taken if SLOs are not being met to determine the problem?**
1. Errors for the preempt process are visible in logs.
2. check the metrics below to determine if there is an exception
  - pod_preemption_victims
  - total_preemption_attempts
  - scheduling_algorithm_preemption_evaluation_seconds

## Implementation History

[Original Github issue](https://github.com/kubernetes/kubernetes/issues/67671)

Pod Priority and Preemption are tracked as part of [enhancement#564](https://github.com/kubernetes/enhancements/issues/564).
The proposal for Pod Priority can be [found here](https://github.com/kubernetes/community/blob/master/contributors/design-proposals/scheduling/pod-priority-api.md)
and Preemption proposal is [here](https://github.com/kubernetes/community/blob/master/contributors/design-proposals/scheduling/pod-preemption.md).
