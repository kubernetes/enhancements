---
title: Add NonPreempting Option For PriorityClasses
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

# Allow PriorityClasses To Be Non-Preempting

## Table of Contents

<!-- toc -->
- [Summary](#summary)
- [Motivation](#motivation)
  - [Goals](#goals)
  - [Non-Goals](#non-goals)
- [Proposal](#proposal)
  - [Risks and Mitigations](#risks-and-mitigations)
- [Graduation Criteria](#graduation-criteria)
- [Testing Plan](#testing-plan)
- [Implementation History](#implementation-history)
<!-- /toc -->

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

Add a boolean to PriorityClasses,
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

### Risks and Mitigations

The new feature may malfuction,
or existing preemption functionality may be impaired.
New tests (covering both nonpreepting workloads and mixed workloads),
and the existing preempting PriorityClass tests should be used to prove stability.

## Graduation Criteria

**Typical user story:**
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

## Testing Plan
Add detailed unit and integration tests for nonpreempting workloads.

Add basic e2e tests, to ensure all components are working together.

Ensure existing tests (for preempting PriorityClasses) do not break.

## Implementation History

[Original Github issue](https://github.com/kubernetes/kubernetes/issues/67671)

Pod Priority and Preemption are tracked as part of [enhancement#564](https://github.com/kubernetes/enhancements/issues/564).
The proposal for Pod Priority can be [found here](https://github.com/kubernetes/community/blob/master/contributors/design-proposals/scheduling/pod-priority-api.md)
and Preemption proposal is [here](https://github.com/kubernetes/community/blob/master/contributors/design-proposals/scheduling/pod-preemption.md).
