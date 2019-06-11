---
title: Add Preemption Option For PriorityClasses
authors:
  - "@vllry"
owning-sig: sig-scheduling
participating-sigs:
  - sig-scheduling
reviewers:
  - "k82cn"
  - "wgliang"
approvers:
  - "bsalamat"
editor: Vallery Lancey
creation-date: 2019-03-17
last-updated: 2019-03-28
status: implementable
see-also:
replaces:
superseded-by:
---

# Allow PriorityClasses To Be Non-Preempting or Non-Preemptible

## Table of Contents

* [Table of Contents](#table-of-contents)
* [Summary](#summary)
* [Motivation](#motivation)
    * [Goals](#goals)
    * [Non-Goals](#non-goals)
* [Proposal](#proposal)
    * [Risks and Mitigations](#risks-and-mitigations)
* [Graduation Criteria](#graduation-criteria)
* [Implementation History](#implementation-history)


## Summary

[PriorityClasses](https://kubernetes.io/docs/concepts/configuration/pod-priority-preemption/) are a GA feature as on 1.14,
which impact the scheduling and eviction of pods.
Pods are be scheduled according to descending priority.
If a pod cannot be scheduled due to insufficient resources,
lower-priority pods will be preempted to make room.

This proposal makes the preemption behavior optional for a PriorityClass,
by adding a new policy field to PriorityClasses,
which in turn populates PodSpec.
If a pod is waiting to be scheduled,
and it does not have preempting enabled,
it will not trigger preemption of other pods.
While a pod is a candidate to be preempted,
but it has the non-preemptible policy,
it will not be taken as victim.

## Motivation

Allowing PriorityClasses to be non-preempting/non-preemptible is important for running batch workloads.

Batch workloads typically have a backlog of work,
with unscheduled pods.
Higher-priority workloads can be assigned a higher priority via a PriorityClass,
but this may result in pods with partially-completed work being preempted.
Adding the non-preempting/non-preemptible option allows users to prioritize the scheduling queue,
without discarding incomplete work.

### Goals

Add a preemption policy filed to PriorityClasses,
to enable or disable preemption for pods of that PriorityClass.

The policy includes non-preemting and non-preemptible.

### Non-Goals

* Pod high available and autoscaling. PodDisruptionBudget should be used.

## Proposal

Add a PreemptionPolicy field to both PodSpec and PriorityClass.
This field will default to `PreemptLowerPriority`,
for backwards compatibility.

* PreemptLowerPriority means that pod can preempt other pods with lower priority and
can be preempted by other pods with higher priority.
* PreemptNever means that pod never preempts other pods with lower priority and
can be preempted by other pods with higher priority.
* NonPreemptible means that pod can preempt other pods with lower priority and
can not be preempted by other pods with higher priority.
* NonPreemptiblePreemptNever means that pod can not preempt other pods with lower priority and
can not be preempted by other pods with higher priority.

PriorityClass type example:
```
type PreemptionPolicy string

type PriorityClass struct {
	metav1.TypeMeta
	metav1.ObjectMeta
	Value int32
	GlobalDefault bool
	Description string
	PreemptionPolicy *PreemptionPolicy // New option
}
```

The PreemptionPolicy field in PodSpec will be populated during pod admission,
similarly to how the PriorityClass Value is populated.
Storing the PreemptionPolicy field in the pod spec has several benefits:
* The scheduler does not need to be aware of PiorityClasses,
as all relevant information is in the pod.
* Mutating PriorityClass objects does not impact existing pods.
* Kubelets can set PreemptionPolicy on static pods.

PodSpec type example:
```
type PodSpec struct {
    ...
	PreemptionPolicy *PreemptionPolicy
    ...
}
```

This feature should be gated in alpha, provisionally under the gate `NonPreemptingPriority`.

Documentation must be updated to reflect the new feature,
and changes to PriorityClass/PodSpec fields.

### Risks and Mitigations

The new feature may malfuction,
or existing preemption functionality may be impaired.
New tests (covering both nonpreepting workloads and mixed workloads),
and the existing preempting PriorityClass tests should be used to prove stability.

## Graduation Criteria

**Typical user story A:**
A user is running batch workloads on a cluster.
The user has a high-priority job,
that they wish to schedule before other workloads in the queue.
As the user does not want to preempt running batch workloads and discard work,
the user creates the new workload with a high-priority,
non-preempting PriorityClass.
The new workload's pods are scheduled ahead of the queue,
without disrupting running workloads.

**Typical user story B:**
A user is running batch workloads on a cluster.
The user has a low-priority jobs, and it's non-interruptible,
that they wish to schedule after other higer priority workloads in the queue.
As the user does not want it to be preempted once it's running and until finishing,
the user creates the new workload with a low-priority,
non-preemptible PriorityClass.
The new workload's pods will be scheduled base on the priority queue,
and when it's running it would not be preempted by higher priority tasks.


* Users are able to run preempting and non-preempting workloads in a stable manner,
and are not requesting additional changes.
* The feature has been stable and reliable in at least 2 releases.
* Adequate documentation exists for preemption and the optional field.
* Test coverage includes non-preempting use cases.
* Conformance requirements for non-preempting PriorityClasses are agreed upon.

## Testing Plan
Add detailed unit and integration tests for non-preempting/non-preemptible workloads.

Add basic e2e tests, to ensure all components are working together.

Ensure existing tests (for preempting PriorityClasses) do not break.

## Implementation History

[Original Github issue](https://github.com/kubernetes/kubernetes/issues/67671)

Pod Priority and Preemption are tracked as part of [enhancement#564](https://github.com/kubernetes/enhancements/issues/564).
The proposal for Pod Priority can be [found here](https://github.com/kubernetes/community/blob/master/contributors/design-proposals/scheduling/pod-priority-api.md)
and Preemption proposal is [here](https://github.com/kubernetes/community/blob/master/contributors/design-proposals/scheduling/pod-preemption.md).
