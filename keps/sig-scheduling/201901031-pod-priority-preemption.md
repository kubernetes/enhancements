---
title: Promote Pod Priority and Preemption to GA
authors:
  - "@bsalamat"
owning-sig: sig-scheduling
participating-sigs:
  - sig-scheduling
reviewers:
  - "@k82cn"
approvers:
  - "@liggitt"
editor: Babak Salamat
creation-date: 2019-01-31
last-updated: 2017-01-31
status: impementable
see-also:
replaces:
superseded-by:
---

# Promote Pod Priority and Preemption to GA

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

Pod Priority and Preemption are features introduced in Kubernetes 1.8 as alpha features and 
promoted to beta in 1.11. Pod Priority enables users to specify importance of
a Pod. Pods with higher priority are scheduled ahead of other pods with
lower priority. When a cluster does not have enough capacity for running a high
priority pod, the scheduler preempts and removes lower priority pods in order to
make room for the high priority pod.

## Motivation

Pod Priority and Preemption have existed in the past several releases and some
of our most critical components of Kubernetes, i.e. critical DaemonSet Pods,
rely on this feature for guaranteed scheduling since Kubernetes 1.12.

### Goals

Promote Pod Priority and Preemption to GA.

### Non-Goals

Make any change of functionality to the features.

## Proposal

Create `scheduling.k8s.io/v1` API group and add `PriorityClass` to it.

Make necessary changes to our code base to use `scheduling.k8s.io/v1` instead of
`scheduling.k8s.io/v1beta1`.

Update our documentation to reflect the new version and status of the features.

### Risks and Mitigations

Given that there is no functionality changes, we don't expect any logical errors
caused by this change.

## Graduation Criteria

* The features have been stable and reliable in the past several releases.
* Adequate documentation exists for the features.
* Test coverage of the features is acceptable.

## Implementation History

Pod Priority and Preemption are tracked as part of [enhancement#564](https://github.com/kubernetes/enhancements/issues/564).
The proposal for Pod Priority can be [found here](https://github.com/kubernetes/community/blob/master/contributors/design-proposals/scheduling/pod-priority-api.md)
and Preemption proposal is [here](https://github.com/kubernetes/community/blob/master/contributors/design-proposals/scheduling/pod-preemption.md).
