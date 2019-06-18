---
title: In-place Update of Pod Resources
authors:
  - "@kgolab"
  - "@bskiba"
  - "@schylek"
  - "@vinaykul"
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
            * [Container Restart Policy](#container-restart-policy)
            * [CRI Changes](#cri-changes)
         * [Flow Control](#flow-control)
            * [Transitions of the ResizingPod PodCondition](#transitions-of-the-resizingpod-podcondition)
            * [Container resource limit update ordering](#container-resource-limit-update-ordering)
            * [Container resource limit update failure handling](#container-resource-limit-update-failure-handling)
            * [Notes](#notes)
         * [Affected Components](#affected-components)
         * [Possible Extensions](#possible-extensions)
         * [Risks and Mitigations](#risks-and-mitigations)
      * [Graduation Criteria](#graduation-criteria)
      * [Implementation History](#implementation-history)
      * [Alternatives](#alternatives)

## Summary

This proposal aims at allowing Pod resource requests & limits to be updated
in-place, without a need to restart the Pod or its Containers.

The **core idea** behind the proposal is to make PodSpec mutable with regards to
Resources, denoting **desired** resources.
Additionally, PodStatus is extended to provide information about **actual**
resource allocation.

This document builds upon [proposal for live and in-place vertical scaling][] and
[Vertical Resources Scaling in Kubernetes][].

[proposal for live and in-place vertical scaling]: https://github.com/kubernetes/community/pull/1719
[Vertical Resources Scaling in Kubernetes]: https://docs.google.com/document/d/18K-bl1EVsmJ04xeRq9o_vfY2GDgek6B6wmLjXw-kos4/edit?ts=5b96bf40

## Motivation

Resources allocated to a Pod's Container can require a change for various reasons:
* load handled by the Pod has increased significantly and current resources are
  not sufficient,
* load has decreased significantly and allocated resources are unused and wasted,
* resources have simply been set improperly.

Currently, changing resource allocation requires the Pod to be recreated since
the PodSpec's Container Resources is immutable.

While many stateless workloads are designed to withstand such a disruption, some
are more sensitive, especially when using low number of Pod replicas.

Moreover, for stateful or batch workloads, a Pod restart is a serious
disruption, resulting in lower availability or higher cost of running.

Allowing Resources to be changed without recreating the Pod or restarting the
Containers addresses this issue directly.

### Goals

* Primary: allow to change Pod resource requests & limits without restarting its
  Containers.
* Secondary: allow actors (users, VPA, StatefulSet, JobController) to decide
  how to proceed if in-place resource resize is not possible.
* Secondary: allow users to specify which Pods and Containers can be resized
  without a restart.

### Non-Goals

The explicit non-goal of this KEP is to avoid controlling full life-cycle of a
Pod which failed in-place resource resizing. This should be handled by actors
which initiated the resizing.

Other identified non-goals are:
* allow to change Pod QoS class without a restart,
* to change resources of Init Containers without a restart,
* updating extended resources or any other resource types besides CPU, memory.

## Proposal

### API Changes

PodSpec becomes mutable with regards to Container resources requests and limits.
Additionally, PodSpec becomes a Pod subresource to allow fine-grained access control.

PodSpec is extended with information about actually allocated Container resources.

Thanks to the above:
* PodSpec.Container.ResourceRequirements becomes purely a declaration, denoting
  **desired** state of the Pod resources,
* PodStatus.ContainerStatus.ResourcesAllocated (new object) shows the resources
  held by the Pod and its Containers.

In order to determine the state of a Pod resource resize, we add a new PodCondition
named ResizingPod. This PodCondition describes status of the last resize request.

The PodCondition ResizingPod can have the following values:
* Status: false - Pod resize operation completed,
  - Reason: (empty) - Initial state, no resize requested since Pod creation,
  - Reason: Success - Pod and its Containers were successfully resized,
  - Reason: FailedNodeCapacity - Node does not have room to resize the Pod,
* Status: true  - Pod is in the process of being resized,
  - Reason: InProgress - Kubelet is performing Pod resize,
  - Reason: PreEmpting - Lower priority Pods are being evicted.

#### Container Restart Policy

To provide some fine-grained user control, PodSpec.Container.ResourceRequirements
is extended with ResizePolicy flag for each resource type (CPU, memory):
* NoRestart - the default value for CPU; resize Container without restarting it,
* RestartContainer - the default value for memory; restart the Container in-place
  to apply new resource values (e.g. Java process needs to change its Xmx flag),
* RestartPod - restart the whole Pod in-place to apply new resource values
  (e.g. Pod requires its Init Containers to re-run).

By using the ResizePolicy flag, user can mark Containers or Pods as safe
(or unsafe) for in-place resource update.

This flag is used by Kubelet to determine the actions needed. This flag **may** be
used by the actors starting the update to decide if the process should be started
at all (for example VPA might decide to evict Pod with RestartPod policy).

Setting the flag to separately control CPU & memory is due to an observation
that usually CPU can be added/removed without much problem whereas changes to
available memory are more probable to require restarts.

If more than one resource type with different policies are updated, then
RestartPod policy takes precedence over RestartContainer, which in turn takes
precedence over NoRestart policy.

#### CRI Changes

Kubelet calls UpdateContainerResources CRI API which currently takes
*runtimeapi.LinuxContainerResources* parameter that works for Docker and Kata,
but not for Windows. This parameter is changed to *runtimeapi.ContainerResources*,
that is runtime agnostic.

### Flow Control

When a new Pod is created, Scheduler is responsible for selecting a suitable
Node that accommodates the Pod. When a Pod resize is requested, Kubelet attempts
to update the Pod and its Containers. If the Kubelet is unable to do so on its own,
Scheduler helps, if possible, by evicting lower priority Pods from the Node.

The following steps denote a typical flow of an in-place resize operation for a Pod
with ResizePolicy set to Update for all its Containers.

1. The initiating actor updates the Pod's Container.ResourceRequirements using
   PATCH verb.
1. API Server validates the new ResourceRequirements (e.g. limits are not below
   requested resources, QoS class does not change, ResourceQuota not exceeded..).
1. API Server calls all Admission Controllers to verify the Pod Update.
   * If any of the Controllers reject the update, API Server responds with an
     appropriate error message.
1. API Server updates PodSpec object with the new desired ResourceRequirements.
1. Kubelet observes that Pod's Container.ResourceRequirements and
   ContainerStatus.ResourcesAllocated differ.
   It checks its Node allocatable resources to determine if new resources fit.
   * _Case 1_: Kubelet finds new ResourceRequirements fit. It sets ResizingPod.Status
     to true, and ResizingPod.Reason to InProgress, then applies resized cgroup
     limits to the Pod and its Containers, and once successfully done, updates
     Pod's ContainerStatus.ResourcesAllocated to reflect the new
     ResourceRequirements. It then sets ResizingPod.Status to false, and
     ResizingPod.Reason to Success.
     - If at the same time, a new Pod was scheduled against the capacity used by
       this resource resize, that Pod is rejected during Kubelet admission if
       Node has no more room.
   * _Case 2_: Kubelet finds new ResourceRequirements don't fit Node’s allocatable
     resources, and sets ResizingPod.Reason to FailedNodeCapacity, and
     ResizingPod.Status to false.
     - Kubelet uses max(ResourceRequirements, ResourcesAllocated) for Node
       available resources accounting.
1. Scheduler, in parallel, observes that Container.ResourceRequirements and
   ContainerStatus.ResourcesAllocated differ, and uses
   max(ResourceRequirements, ResourcesAllocated) when computing resources
   available on the Node.
   * This can temporarily result in sum of Pod resources for the Node exceeding
     Node's allocatable resources if a Pod was scheduled to that Node in parallel.
1. Scheduler observes ResizingPod.Reason and/or ContainerStatus.ResourcesAllocated
   fields have changed.
   * _Case 1_: ResizingPod.Status is false, and ResizingPod.Reason is Success,
     and Pod's Container.ResourceRequirements matches
     ContainerStatus.ResourcesAllocated. Scheduler updates its cache with resized
     ContainerStatus.ResourcesAllocated values.
   * _Case 2_: ResizingPod.Status is false, ResizingPod.Reason is FailedNodeCapacity.
     - If evicting lower priority Pods on the Node can successfully resize Pod,
       it sets ResizingPod.Status to true, and ResizingPod.Reason to PreEmpting,
       and initiates eviction of lower priority Pods. Once lower priority Pods
       have been evicted, it sets ResizingPod.Reason to InProgress, and the flow
       continues.
     - If Scheduler cannot help, Kubelet is expected to retry the resize at a later
       time, and optionally when other Pods depart the Node (thus creating room).
1. The initiating actor observes that ResizingPod.Reason and/or
   ContainerStatus.ResourcesAllocated fields have changed.
   * _Case 1_: ResizingPod.Status is false, ResizingPod.Reason is Success,
     and Pod's Container.ResourceRequirements matches
     ContainerStatus.ResourcesAllocated, signifying a successful completion of
     in-place Pod resources resizing.
   * _Case 2_: ResizingPod.Status is false, ResizingPod.Reason is FailedNodeCapacity,
     and the initiating actor may take alternative action. For example, based on
     retry policy, initiating actor such as VPA may choose to:
     - Evict the Pod to trigger a replacement Pod with updated resources,
     - Wait for Scheduler to evict lower priority Pods,
     - Do nothing, and let Kubelet backoff and retry in-place resize.

#### Transitions of the ResizingPod PodCondition

The following diagram shows possible transitions of ResizingPod PodCondition's
Status and Reason fields respectively.

```text

    +------------------------------------+
    |                                   2|
    |                               +----v----+
    |                               |         |
    |         +---------------------+  false  |
    |         |                     | Success |
    |         |                     |         |
    |         |                     +----+----+
    |         |                          |
    |        1|                         3|
    |  +------v-----+         +----------v---------+
    |  |            |         |                    |
    +--+    true    <---------+        false       |
       | InProgress |6        | FailedNodeCapacity |
       |            |         |                    |
       +------^-----+         +----------+---------+
             5|                          |
              |                         4|
              |                   +------v-----+
              |                   |            |
              |                   |    true    |
              +-------------------+ PreEmpting |
                                  |            |
                                  +------------+

```

1. Kubelet, on initiating in-place resize.
1. Kubelet, on successful completion of in-place resize.
1. Kubelet, on Node not having capacity to resize.
1. Scheduler, on initiating pre-emption of lower priority Pods.
1. Scheduler, on completing pre-emption of lower priority Pods.
1. Kubelet, on initiating in-place resize via retry.

#### Container resource limit update ordering

When in-place resize is desired for multiple Containers in a Pod, Kubelet updates
resource limit for the Containers as detailed below:
  1. If resource resizing results in net-increase of a resource type (CPU or Memory),
     Kubelet first updates Pod-level cgroup limit for the resource type, and then
     updates the Container resource limit.
  1. If resource resizing results in net-decrease of a resource type, Kubelet first
     updates the Container resource limit, and then updates Pod-level cgroup limit.
  1. If resource update results in no net change of a resource type, only the Container
     resource limits are updated.
In all the above cases, Kubelet applies Container resource limit decreases before
applying limit increases.

#### Container resource limit update failure handling

For simplicity, if Container resource limits update fails, Kubelet restarts the
Container in-place to allow new limits to take effect, and the action is logged.

#### Notes

* As an alternative to Scheduler pre-empting Pods, Kubelet can identify Pods to
  pre-empt and safely evict them via eviction API. Investigation TBD as to
  whether this can be done by Kubelet same as Scheduler would.
* To avoid races and possible gamification, all components should use
  max(ResourceRequirements, ResourcesAllocated) when computing resources used by
  a Pod.
* If another resource update arrives when a previous update is being handled,
  that and all subsequent updates should be buffered at the Controller, and
  applied upon success/failed completion of the update that is in progress.
* Lowering memory limits may not always work if the application is holding on
  to pages. Kubelet will use a control loop to set the memory limits near usage
  in order to force a reclaim, and update ContainerStatus.ResourcesAllocated
  only when limit is at desired value.
* Impact of Pod Overhead: Scheduler adds Pod Overhead to the resize request
  to determine if in-place resize is possible. Kubelet implements Pod Overhead
  if the values have changed.
* Impact of memory-backed emptyDir volumes: If memory-backed emptyDir is in use,
  Kubelet will clear out any files in emptyDir upon Container restart.

### Affected Components

Pod v1 core API:
* extended model,
* added validation.

Admission Controllers: LimitRanger, ResourceQuota need to support Pod Updates:
* for ResourceQuota it should be enough to change podEvaluator.Handler
  implementation to allow Pod updates; max(ResourceRequirements, ResourcesAllocated)
  should be used to be in line with current ResourceQuota behavior
  which blocks resources before they are used (e.g. for Pending Pods),
* for LimitRanger TBD.

Kubelet:
* support in-place resource resize,
* set Pod's ContainerStatus.ResourcesAllocated for Containers on placing the Pod on Node,
* change UpdateContainerResources CRI API so that it works for both Linux and Windows.

Scheduler:
* update its cache to use max(ResourceRequirements, ResourcesAllocated),
* aid resize by evicting lower priority Pods on the Node.

Controllers:
* propagate Template resources update to running Pod instances.

Other components:
* check how the change of meaning of resource requests influence other kubernetes components.

### Possible Extensions

1. Allow resource limits to be updated too (VPA feature).
1. Allow ResizePolicy to be set on Pod level, acting as default if
   (some of) the Containers do not have it set on their own.
1. Extend ResizePolicy flag to separately control resource increase and decrease
   (e.g. a container can be given more memory in-place but
   decreasing memory requires container restart).
1. Allow resizing local ephemeral storage.

### Risks and Mitigations

1. Backward compatibility: When Resources in PodSpec becomes representative of
   desired state, and Pod's true resource allocations tracked in PodStatus,
   applications that query PodSpec and rely on Resources in PodSpec to determine
   resource usage will see values that may not represent actual allocations at
   the time of query. To mitigate, this change needs to be documented and
   highlighted in the release notes, and in top-level kubernetes documents.
1. Resizing memory lower: Lowering cgroup memory limits may not work as pages
   could be in use, and approaches such as setting limit near current usage may
   be required. This issue needs further investigation.

## Graduation Criteria

TODO

## Implementation History

- 2018-11-06 - initial KEP draft created
- 2019-01-18 - implementation proposal extended
- 2019-03-07 - changes to flow control, updates per review feedback

## Alternatives

TODO

