---
title: In-place Update of Pod Resources
authors:
  - "@kgolab"
  - "@bskiba"
  - "@schylek"
owning-sig: sig-autoscaling
participating-sigs:
  - sig-node
  - sig-scheduling
reviewers:
  - "@bsalamat"
  - "@derekwaynecarr"
  - "@dchen1107"
approvers:
  - TBD
editor: TBD
creation-date: 2018-11-06
last-updated: 2018-11-06
status: provisional
see-also:
replaces:
superseded-by:
---

# In-place Update of Pod Resources

## Table of Contents

   * [In-place Update of Pod Resources](#in-place-update-of-pod-resources)
      * [Table of Contents](#table-of-contents)
      * [Summary](#summary)
      * [Motivation](#motivation)
         * [Goals](#goals)
         * [Non-Goals](#non-goals)
      * [Proposal](#proposal)
         * [API Changes](#api-changes)
         * [Flow Control](#flow-control)
            * [Transitions of InPlaceResize condition](#transitions-of-inplaceresize-condition)
            * [Notes](#notes)
         * [Affected Components](#affected-components)
         * [Risks and Mitigations](#risks-and-mitigations)
      * [Graduation Criteria](#graduation-criteria)
      * [Implementation History](#implementation-history)
      * [Alternatives](#alternatives)

## Summary

This proposal aims at allowing Pod resource requests & limits to be updated
in-place, without a need to restart the Pod or its Containers.

The **core idea** behind the proposal is to make PodSpec mutable with regards to
Resources, denoting **desired** resources.
Additionally PodStatus is extended to provide information about **actual**
resource allocation.

This document builds upon [proposal for live and in-place vertical scaling][] and
[Vertical Resources Scaling in Kubernetes][].

[proposal for live and in-place vertical scaling]: https://github.com/kubernetes/community/pull/1719
[Vertical Resources Scaling in Kubernetes]: https://docs.google.com/document/d/18K-bl1EVsmJ04xeRq9o_vfY2GDgek6B6wmLjXw-kos4/edit?ts=5b96bf40

## Motivation

Resources allocated to a Pod's Container can require a change for various reasons:
* load handled by the Pod has increased significantly and current resources are
  not enough to handle it,
* load has decreased significantly and currently allocated resources are unused
  and thus wasted,
* Resources have simply been set improperly.

Currently changing Resource allocation requires the Pod to be recreated since
the PodSpec is immutable.

While many stateless workloads are designed to withstand such a disruption, some
are more sensitive, especially when using low number of Pod replicas.

Moreover, for stateful or batch workloads, a Pod restart is a serious
disruption, resulting in lower availability or higher cost of running.

Allowing Resources to be changed without recreating a Pod nor restarting a
Container addresses this issue directly.

### Goals

* Primary: allow to change Pod resource requests & limits without restarting its
  Containers.
* Secondary: allow actors (users, VPA, StatefulSet, JobController) to decide
  how to proceed if in-place resource update is not available.
* Secondary: allow users to specify which Pods and Containers can be updated
  without a restart.

### Non-Goals

The explicit non-goal of this KEP is to avoid controlling full life-cycle of a
Pod which failed an in-place resource update. These cases should be handled by
actors which initiated the update.

Other identified non-goals are:
* allow to change Pod QoS class without a restart,
* to change resources of Init Containers without a restart,
* updating extended resources or any other resource types besides CPU, memory.

## Proposal

### API Changes

PodSpec becomes mutable with regards to resources and limits.
Additionally, PodSpec becomes a Pod subresource to allow fine-grained access control.

PodStatus is extended with information about actually allocated resources.

Thanks to the above:
* PodSpec.Container.ResourceRequirements becomes purely a declaration,
  denoting **desired** state of the Pod,
* PodStatus.ContainerStatus.ResourceAllocated (new object) denotes **actual**
  state of the Pod resources.

To distinguish between possible states of the Pod resources,
a new PodCondition InPlaceResize is added, with the following states:
* (empty) - the default value; resource update awaits reconciliation
  (if ResourceRequirements differs from ResourceAllocated),
* Awaiting - awaiting resources to be freed (e.g. via pre-emption),
* Failed - resource update could not have been performed in-place
  but might be possible if some conditions change,
* Rejected - resource update was rejected by any of the components involved.

To provide some fine-grained control to the user,
PodSpec.Container.ResourceRequirements is extended with ResizingPolicy flag,
available per each resource request (CPU, memory) :
* InPlace - the default value; allow in-place resize of the Container,
* RestartContainer - restart the Container to apply new resource values
  (e.g. Java process needs to change its Xmx flag),
* RestartPod - restart whole Pod to apply new resource values
  (e.g. Pod requires its Init Containers to re-run).

By using the ResizingPolicy flag the user can mark Containers or Pods as safe
(or unsafe) for in-place resources update.

This flag **may** be used by the actors starting the process to decide if
the process should be started at all (for example VPA might decide to
evict Pod with RestartPod policy).
This flag **must** be used by Kubelet to verify the actions needed.

Setting the flag to separately control CPU & memory is due to an observation
that usually CPU can be added/removed without much problems whereas
changes to available memory are more probable to require restarts.

### Flow Control

The following steps denote a positive flow of an in-place update,
for a Pod having ResizingPolicy set to InPlace for all its Containers.
Some alternative flows are given in indented steps,
unless noted otherwise they abort the flow.

1. The initiating actor updates ResourceRequirements using PATCH verb.
1. API Server validates the new ResourceRequirements
   (e.g. limits are not below requested resources, QoS class does not change).
1. API Server calls all Admission Controllers to verify the Pod Update.
   1. If any of the controllers rejects the update,
      the InPlaceResize PodCondition is set to Rejected.
1. API Server updates the PodSpec object and clears InPlaceResize condition.
1. Scheduler observes that ResourceRequirements and ResourceAllocated differ.
   It updates its resource cache to use max(ResourceRequirements, ResourceAllocated).
   1. If required it pre-empts lower-priority Pods, setting
      the InPlaceResize PodCondition to Awaiting.
      Once the lower-priority Pods are evicted, Scheduler clears
      the InPlaceResize PodCondition and the flow continues.
1. Kubelet observes that ResourceRequirements and ResourceAllocated differ
   and the InPlaceResize condition is clear.
   This is done potentially prior to Scheduler pre-empting lower-priority Pods.
   1. Kubelet checks that new ResourceRequirements do not fit Node’s
      allocatable resources and sets the InPlaceResize condition to Failed.
1. Kubelet applies new resource values to cgroups, updates values
   in ResourceAllocated to match ResourceRequirements
   and clears InPlaceResize condition.
1. Scheduler observes that ResourceAllocated has changed.
   It updates its resource cache to use new value of ResourceAllocated
   for the given Pod.
1. The initating actor observes that ResourceRequirements and
   ResourceAllocated match again which signifies the completion of an update.

#### Transitions of InPlaceResize condition

The following diagram shows possible transitions of InPlaceResize condition.

```text
                   +---------+
       +-----------+         +-----------+
       |           | (empty) |           |
       | +--------->         <---------+ |
       | |         +----+----+         | |
      1| |2            3|             4| |5
 +-----v-+--+           |          +---+-v--+
 |          |           |          |        |
 | Awaiting |           |          | Failed |
 |          |           |          |        |
 +-------+--+           |          +---+----+
        3|              |              |3
         |         +----v-----+        |
         |         |          |        |
         +---------> Rejected <--------+
                   |          |
                   +----------+
```

1. Scheduler, on pre-emption.
1. Scheduler, after pre-emption finishes.
1. Any Controller, on permanent issue.
1. Kubelet, on successful retry.
1. Kubelet, if not enough space on Node.

#### Notes

* In case when there is no pre-emption required, Kubelet and Scheduler
  will pick up the ResourceRequirements change in parallel.
* In case when there is pre-emption required Kubelet and Scheduler might
  pick up the ResourceRequirements change in parallel,
  Kubelet will then set the InPlaceResize condition to Failed
  and Scheduler will clear it once pre-emption is done.
* Kubelet might try to apply new resources also if InPlaceResize
  condition is set to Failed, as a normal retry mechanism.
* To avoid races and possible gamification, all components should use
  max(ResourceRequirements, ResourceAllocated) when computing resources
  used by a Pod. TBD if this can be weakened when InPlaceResize condition
  is set to Rejected, or should the initiating actor update
  ResourceRequirements back to reclaim resources.

### Affected Components

Pod v1 core API:
* extended model,
* added validation.

Admission Controllers: LimitRanger, ResourceQuota need to support Pod Updates:
* for ResourceQuota it should be enough to change podEvaluator.Handler
  implementation to allow Pod updates; max(ResourceRequirements, ResourceAllocated)
  should be used to be in line with current ResourceQuota behaviour
  which blocks resources before they are used (e.g. for Pending Pods),
* for LimitRanger TBD.

Kubelet
* support in-place resource management,
* set ResourceRequirements on placing the Pod on Node.

Scheduler:
* update its caches with proper resources, depending on InPlaceResize condition.

Other components:
* check how the change of meaning of resource requests influence other kubernetes components.

### Possible Extensions

1. Allow resource limits to be updated too.
1. Allow ResizingPolicy to be set on Pod level, acting as default if
   (some of) the Containers do not have it set on their own.
1. Extend ResizingPolicy flag to separately control resource increase and decrease
   (e.g. a container can be given more memory in-place but
   decreasing memory requires container restart).

### Risks and Mitigations

TODO

## Graduation Criteria

TODO

## Implementation History

- 2018-11-06 - initial KEP draft created
- 2019-01-18 - implementation proposal extended

## Alternatives

TODO

