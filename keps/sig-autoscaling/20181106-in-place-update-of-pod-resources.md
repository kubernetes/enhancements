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
            * [Container Resize Policy](#container-resize-policy)
            * [CRI Changes](#cri-changes)
         * [Kubelet and API Server interaction](#kubelet-and-api-server-interaction)
            * [Kubelet Restart Tolerance](#kubelet-restart-tolerance)
         * [Scheduler and API Server interaction](#scheduler-and-api-server-interaction)
         * [Flow Control](#flow-control)
            * [Container resource limit update ordering](#container-resource-limit-update-ordering)
            * [Notes](#notes)
         * [Affected Components](#affected-components)
         * [Future Enhancements](#future-enhancements)
         * [Risks and Mitigations](#risks-and-mitigations)
      * [Graduation Criteria](#graduation-criteria)
      * [Implementation History](#implementation-history)

## Summary

This proposal aims at allowing Pod resource requests & limits to be updated
in-place, without a need to restart the Pod or its Containers.

The **core idea** behind the proposal is to make PodSpec mutable with regards to
Resources, denoting **desired** resources. Additionally, PodSpec is extended to
reflect resources **allocated** to a Pod, and PodStatus is extended to provide
information about **actual** resources applied to the Pod and its Containers.

This document builds upon [proposal for live and in-place vertical scaling][]
and [Vertical Resources Scaling in Kubernetes][].

[proposal for live and in-place vertical scaling]:
https://github.com/kubernetes/community/pull/1719
[Vertical Resources Scaling in Kubernetes]:
https://docs.google.com/document/d/18K-bl1EVsmJ04xeRq9o_vfY2GDgek6B6wmLjXw-kos4

## Motivation

Resources allocated to a Pod's Container(s) can require a change for various
reasons:
* load handled by the Pod has increased significantly, and current resources
  are not sufficient,
* load has decreased significantly, and allocated resources are unused,
* resources have simply been set improperly.

Currently, changing resource allocation requires the Pod to be recreated since
the PodSpec's Container Resources is immutable.

While many stateless workloads are designed to withstand such a disruption,
some are more sensitive, especially when using low number of Pod replicas.

Moreover, for stateful or batch workloads, Pod restart is a serious disruption,
resulting in lower availability or higher cost of running.

Allowing Resources to be changed without recreating the Pod or restarting the
Containers addresses this issue directly.

### Goals

* Primary: allow to change Pod resource requests & limits without restarting
  its Containers.
* Secondary: allow actors (users, VPA, StatefulSet, JobController) to decide
  how to proceed if in-place resource resize is not possible.
* Secondary: allow users to specify which Pods and Containers can be resized
  without a restart.

### Non-Goals

The explicit non-goal of this KEP is to avoid controlling full lifecycle of a
Pod which failed in-place resource resizing. This should be handled by actors
which initiated the resizing.

Other identified non-goals are:
* allow to change Pod QoS class without a restart,
* to change resources of Init Containers without a restart,
* eviction of lower priority Pods to facilitate Pod resize,
* updating extended resources or any other resource types besides CPU, memory.

## Proposal

### API Changes

PodSpec becomes mutable with regards to Container resources requests and
limits. PodSpec is extended with information of resources allocated on the
Node for the Pod. PodStatus is extended to show the actual resources applied
to the Pod and its Containers.

Thanks to the above:
* Pod.Spec.Containers[i].Resources becomes purely a declaration, denoting the
  **desired** state of Pod resources,
* Pod.Spec.Containers[i].ResourcesAllocated (new object, type v1.ResourceList)
  denotes the Node resources **allocated** to the Pod and its Containers,
* Pod.Status.ContainerStatuses[i].Resources (new object, type
  v1.ResourceRequirements) shows the **actual** resources held by the Pod and
  its Containers.

A new Pod subresource named 'resourceallocation' is introduced to allow
fine-grained access control that enables Kubelet to set or update resources
allocated to a Pod.

#### Container Resize Policy

To provide fine-grained user control, PodSpec.Containers is extended with
ResizePolicy map (new object) for each resource type (CPU, memory):
* NoRestart - the default value; resize Container without restarting it,
* RestartContainer - restart the Container in-place to apply new resource
  values. (e.g. Java process needs to change its Xmx flag)

By using ResizePolicy, user can mark Containers as safe (or unsafe) for
in-place resource update. Kubelet uses it to determine the required action.

Setting the flag to separately control CPU & memory is due to an observation
that usually CPU can be added/removed without much problem whereas changes to
available memory are more probable to require restarts.

If more than one resource type with different policies are updated, then
RestartContainer policy takes precedence over NoRestart policy.

Additionally, if RestartPolicy is 'Never', ResizePolicy should be set to
NoRestart in order to pass validation.

#### CRI Changes

Kubelet calls UpdateContainerResources CRI API which currently takes
*runtimeapi.LinuxContainerResources* parameter that works for Docker and Kata,
but not for Windows. This parameter changes to *runtimeapi.ContainerResources*,
that is runtime agnostic, and will contain platform-specific information.

### Kubelet and API Server Interaction

When a new Pod is created, Scheduler is responsible for selecting a suitable
Node that accommodates the Pod.

For a newly created Pod, Spec.Containers[i].ResourcesAllocated must match
Spec.Containers[i].Resources.Requests. When Kubelet admits a new Pod, values in
Spec.Containers[i].ResourcesAllocated are used to determine if there is enough
room to admit the Pod. Kubelet does not set Pod's ResourcesAllocated after
admitting a new Pod.

When a Pod resize is requested, Kubelet attempts to update the resources
allocated to the Pod and its Containers. Kubelet first checks if the new
desired resources can fit the Node allocable resources by computing the sum of
resources allocated (Pod.Spec.Containers[i].ResourcesAllocated) for all Pods in
the Node, except the Pod being resized. For the Pod being resized, it adds the
new desired resources (i.e Spec.Containers[i].Resources.Requests) to the sum.
* If new desired resources fit, Kubelet accepts the resize by updating
  Pod.Spec.Containers[i].ResourcesAllocated via pods/resourceallocation
  subresource, and then proceeds to invoke UpdateContainerResources CRI API
  to update the Container resource limits. Once all Containers are successfully
  updated, it updates Pod.Status.ContainerStatuses[i].Resources to reflect the
  new resource values.
* If new desired resources don't fit, Kubelet rejects the resize, and no
  further action is taken.
  - Kubelet retries the Pod resize at a later time.

Scheduler may, in parallel, assign a new Pod to the Node because it uses cached
Pods to compute Node allocable values. If this race condition occurs, Kubelet
resolves it by rejecting that new Pod if the Node has no room after Pod resize.

#### Kubelet Restart Tolerance

If Kubelet were to restart amidst handling a Pod resize, then upon restart, all
Pods are admitted at their current Pod.Spec.Containers[i].ResourcesAllocated
values, and resizes are handled after all existing Pods have been added. This
ensures that resizes don't affect previously admitted existing Pods.

### Scheduler and API Server Interaction

Scheduler continues to use Pod's Spec.Containers[i].Resources.Requests for
scheduling new Pods, and continues to watch Pod updates, and updates its cache.
It uses the cached Pod's Spec.Containers[i].ResourcesAllocated values to
compute the Node resources allocated to Pods. This ensures that it always uses
the most recently available resource allocations in making new Pod scheduling
decisions.

### Flow Control

The following steps denote a typical flow of an in-place resize operation for a
Pod with ResizePolicy set to NoRestart for all its Containers.

1. Initiating actor updates Pod's Spec.Containers[i].Resources via PATCH verb.
1. API Server validates the new Resources. (e.g. Limits are not below
   Requests, QoS class doesn't change, ResourceQuota not exceeded...)
1. API Server calls all Admission Controllers to verify the Pod Update.
   * If any of the Controllers reject the update, API Server responds with an
     appropriate error message.
1. API Server updates PodSpec object with the new desired Resources.
1. Kubelet observes that Pod's Spec.Containers[i].Resources.Requests and
   Spec.Containers[i].ResourcesAllocated differ. It checks its Node allocable
   resources to determine if the new desired Resources fit the Node.
   * _Case 1_: Kubelet finds new desired Resources fit. It accepts the resize
     and sets Spec.Containers[i].ResourcesAllocated equal to the values of
     Spec.Containers[i].Resources.Requests by invoking resourceallocation
     subresource. It then applies the new cgroup limits to the Pod and its
     Containers, and once successfully done, sets Pod's
     Status.ContainerStatuses[i].Resources to reflect the desired resources.
     - If at the same time, a new Pod was assigned to this Node against the
       capacity taken up by this resource resize, that new Pod is rejected by
       Kubelet during admission if Node has no more room.
   * _Case 2_: Kubelet finds that the new desired Resources does not fit.
     - If Kubelet determines there isn't enough room, it simply retries the Pod
       resize at a later time.
1. Scheduler uses cached Pod's Spec.Containers[i].ResourcesAllocated to compute
   resources available on the Node while a Pod resize may be in progress.
   * If a new Pod is assigned to that Node in parallel, it can temporarily
     result in actual sum of Pod resources for the Node exceeding Node's
     allocable resources. This is resolved when Kubelet rejects that new Pod
     during admission due to lack of room.
   * Once Kubelet that accepted a parallel Pod resize updates that Pod's
     Spec.Containers[i].ResourcesAllocated, and subsequently the Scheduler
     updates its cache, accounting will reflect updated Pod resources for
     future computations and scheduling decisions.
1. The initiating actor (e.g. VPA) observes the following:
   * _Case 1_: Pod's Spec.Containers[i].ResourcesAllocated values have changed
     and matches Spec.Containers[i].Resources.Requests, signifying that desired
     resize has been accepted, and Pod is being resized. The resize operation
     is complete when Pod's Status.ContainerStatuses[i].Resources and
     Spec.Containers[i].Resources match.
   * _Case 2_: Pod's Spec.Containers[i].ResourcesAllocated remains unchanged,
     and continues to differ from desired Spec.Containers[i].Resources.Requests.
     After a certain (user defined) timeout, initiating actor may take alternate
     action. For example, based on Retry policy, initiating actor may:
     - Evict the Pod to trigger a replacement Pod with new desired resources,
     - Do nothing and let Kubelet back off and later retry the in-place resize.

#### Container resource limit update ordering

When in-place resize is requested for multiple Containers in a Pod, Kubelet
updates resource limit for the Pod and its Containers in the following manner:
  1. If resource resizing results in net-increase of a resource type (CPU or
     Memory), Kubelet first updates Pod-level cgroup limit for the resource
     type, and then updates the Container resource limit.
  1. If resource resizing results in net-decrease of a resource type, Kubelet
     first updates the Container resource limit, and then updates Pod-level
     cgroup limit.
  1. If resource update results in no net change of a resource type, only the
     Container resource limits are updated.

In all the above cases, Kubelet applies Container resource limit decreases
before applying limit increases.

#### Notes

* If CPU Manager policy for a Node is set to 'static', then only integral
  values of CPU resize are allowed. If non-integral CPU resize is requested
  for a Node with 'static' CPU Manager policy, that resize is rejected, and
  an error message is logged to the event stream.
* To avoid races and possible gamification, all components will use Pod's
  Spec.Containers[i].ResourcesAllocated when computing resources used by Pods.
* If additional resize requests arrive when a Pod is being resized, those
  requests are handled after completion of the resize that is in progress. And
  resize is driven towards the latest desired state.
* Lowering memory limits may not always take effect quickly if the application
  is holding on to pages. Kubelet will use a control loop to set the memory
  limits near usage in order to force a reclaim, and update the Pod's
  Status.ContainerStatuses[i].Resources only when limit is at desired value.
* Impact of Pod Overhead: Kubelet adds Pod Overhead to the resize request to
  determine if in-place resize is possible.
* Impact of memory-backed emptyDir volumes: If memory-backed emptyDir is in
  use, Kubelet will clear out any files in emptyDir upon Container restart.
* At this time, Vertical Pod Autoscaler should not be used with Horizontal Pod
  Autoscaler on CPU, memory. This enhancement does not change that limitation.

### Affected Components

Pod v1 core API:
* extended model,
* new subresource,
* added validation.

Admission Controllers: LimitRanger, ResourceQuota need to support Pod Updates:
* for ResourceQuota, podEvaluator.Handler implementation is modified to allow
  Pod updates, and verify that sum of Pod.Spec.Containers[i].Resources for all
  Pods in the Namespace don't exceed quota,
* for LimitRanger we check that a resize request does not violate the min and
  max limits specified in LimitRange for the Pod's namespace.

Kubelet:
* set Pod's Status.ContainerStatuses[i].Resources for Containers upon placing
  a new Pod on the Node,
* update Pod's Spec.Containers[i].ResourcesAllocated upon resize,
* change UpdateContainerResources CRI API to work for both Linux & Windows.

Scheduler:
* compute resource allocations using Pod.Spec.Containers[i].ResourcesAllocated.

Controllers:
* propagate Template resources update to running Pod instances.

Other components:
* check how the change of meaning of resource requests influence other
  Kubernetes components.

### Future Enhancements

1. Kubelet (or Scheduler) evicts lower priority Pods from Node to make room for
   resize. Pre-emption by Kubelet may be simpler and offer lower latencies.
1. Allow ResizePolicy to be set on Pod level, acting as default if (some of)
   the Containers do not have it set on their own.
1. Extend ResizePolicy to separately control resource increase and decrease
   (e.g. a Container can be given more memory in-place but decreasing memory
   requires Container restart).
1. Extend Node Information API to report the CPU Manager policy for the Node,
   and enable validation of integral CPU resize for nodes with 'static' CPU
   Manager policy.
1. Allow resizing local ephemeral storage.
1. Allow resource limits to be updated (VPA feature).

### Risks and Mitigations

1. Backward compatibility: When Pod.Spec.Containers[i].Resources becomes
   representative of desired state, and Pod's true resource allocations are
   tracked in Pod.Spec.Containers[i].ResourcesAllocated, applications that
   query PodSpec and rely on Resources in PodSpec to determine resource
   allocations will see values that may not represent actual allocations. As
   a mitigation, this change needs to be documented and highlighted in the
   release notes, and in top-level Kubernetes documents.
1. Resizing memory lower: Lowering cgroup memory limits may not work as pages
   could be in use, and approaches such as setting limit near current usage may
   be required. This issue needs further investigation.

## Graduation Criteria

TODO

## Implementation History

- 2018-11-06 - initial KEP draft created
- 2019-01-18 - implementation proposal extended
- 2019-03-07 - changes to flow control, updates per review feedback
- 2019-08-29 - updated design proposal
