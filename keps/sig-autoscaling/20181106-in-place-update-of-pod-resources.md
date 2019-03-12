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
            * [Retry Policy](#retry-policy)
            * [CRI Changes](#cri-changes)
         * [Flow Control](#flow-control)
            * [Transitions of ResourceResizeRequired PodCondition](#transitions-of-resourceresizerequired-podcondition)
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

PodStatus is extended with information about actually allocated Container resources.

Thanks to the above:
* PodSpec.Container.ResourceRequirements becomes purely a declaration,
  denoting **desired** state of the Pod,
* PodStatus.ContainerStatus.ResourceAllocated (new object) denotes **actual**
  state of the Pod resources.

To distinguish between possible states of a Pod resource update,
a new PodCondition named ResourceResizeRequired is added, with the following states:
* (empty) - the default value; resource update awaits reconciliation
  if ResourceRequirements differs from ResourceAllocated,
* Requested - Scheduler determined in-place resource resizing is possible, and
  requested Kubelet to update Pod's resource allocations and limits,
* Awaiting - awaiting resources to be freed (e.g. via pre-emption),
* Failed - resource resizing could not have been performed in-place
  but might be possible if some conditions change,
* Rejected - resource update was rejected by any of the components involved.

To provide some fine-grained control to the user,
PodSpec.Container.ResourceRequirements is extended with ResizePolicy flag
for each resource type (CPU, memory) :
* NoRestart - the default value; resize the Container without restarting it,
* RestartContainer - restart the Container in-place to apply new resource
  values (e.g. Java process needs to change its Xmx flag),
* RestartPod - restart the whole Pod in-place to apply new resource values
  (e.g. Pod requires its Init Containers to re-run).

By using the ResizePolicy flag, user can mark Containers or Pods as safe
(or unsafe) for in-place resource update.

This flag is used by Kubelet to determine the actions needed. This flag **may** be
used by the actors starting the update to decide if the process should be started
at all (for example VPA might decide to evict Pod with RestartPod policy).

Setting the flag to separately control CPU & memory is due to an observation
that usually CPU can be added/removed without much problems whereas
changes to available memory are more probable to require restarts.

If more than one resource type with different policies are updated, then
RestartPod policy takes precedence over RestartContainer, which in turn takes
precedence over NoRestart policy.

#### Retry Policy

If resource update fails, say due to lack of space on Node, default behavior
is to let the initiating actor such as VPA handle the failure. Alternately, a
Controller can either retry the update, or reschedule the pod based on policy.

PodSpec is extended with a new flag, PodSpec.RetryPolicy, with possible values:
* (empty) - the default value; do nothing, initiating actor handles failure,
* RetryUpdate - Controller retries resource update in-place when other pods depart,
* Reschedule - Controller deletes pod, and creates updated pod for scheduling.

#### CRI Changes

Kubelet calls UpdateContainerResources CRI API which currently takes
*runtimeapi.LinuxContainerResources* parameter that works for Docker and Kata,
but not for Windows. This parameter is changed to *runtimeapi.ContainerResources*,
that is runtime agnostic.

### Flow Control

The following steps denote a typical flow of an in-place resize process for a Pod
with ResizePolicy set to Update for all its Containers.

1. The initiating actor updates ResourceRequirements using PATCH verb.
1. API Server validates the new ResourceRequirements
   (e.g. limits are not below requested resources, QoS class does not change).
1. API Server calls all Admission Controllers to verify the Pod Update.
   * If any of the Controllers rejects the update, ResourceResizeRequired
     PodCondition.Reason is set to Rejected, and PodCondition.Status is false.
1. API Server updates the PodSpec object, clears ResourceResizeRequired
   PodCondition.Reason, and sets the PodCondition.Status to true.
1. Scheduler observes that ResourceRequirements and ResourceAllocated differ.
   It checks its cache to determine if in-place resource resizing is possible.
   * If Node has capacity to accommodate new resource values, it updates
     its resource cache to use max(ResourceRequirements, ResourceAllocated),
     and sets ResourceResizeRequired PodCondition.Reason to Requested.
   * If required it pre-empts lower-priority Pods, setting the
     ResourceResizeRequired PodCondition.Reason to Awaiting. Once the
     lower-priority Pods are evicted, Scheduler clears the
     ResourceResizeRequired PodCondition.Reason and the flow continues.
   * If Node does not have capacity to accommodate new resource values, it
     sets ResourceResizeRequired PodCondition.Reason to Failed.
1. Kubelet observes that ResourceResizeRequired PodCondition.Reason has been
   set to Requested, and checks its Node allocatable resources against the new
   ResourceRequirements for fit.
   * Kubelet sees that new ResourceRequirements fits, updates the PodStatus
     ResourceAllocated to match ResourceRequirements, clears the
     ResourceResizeRequired PodCondition.Reason, sets PodCondition.Status to
     false, and then applies the new cgroup limits to the Pod and its Containers.
   * Kubelet sees that new ResourceRequirements does not fit Node’s allocatable
     resources and sets the ResourceResizeRequired PodCondition.Reason to Failed.
     This can happen due to race-condition with multiple schedulers.
1. Scheduler observes that PodCondition.Reason has changed.
   * Case 1: ResourceResizeRequired PodCondition.Reason is clear, and
     ResourceRequirements matches ResourceAllocated. Scheduler updates cache to
     use the updated ResourceAllocated values.
   * Case 2: ResourceResizeRequired PodCondition.Reason is Failed. Scheduler
     updates cache to use the unchanged ResourceAllocated values for accounting.
1. The initiating actor observes that ResourceAllocated has changed.
   * Case 1: ResourceRequirements and ResourceAllocated match again, signifying
     a successful completion of Pod resources in-place resizing.
   * Case 2: ResourceResizeRequired PodCondition.Reason shows Failed, and
     initiating actor may take action.
     A few possible examples (perhaps controlled by a Retry policy):
     * Initiating actor (user/VPA) handles it maybe by deleting the Pod to
       trigger a replacement Pod with new resources for scheduling.
     * Initiating actor is a Controller (Job,Deployment,..), and it clears the
       ResourceResizeRequired PodCondition.Reason and sets PodCondition.Status
       to true (based on other Pods departing, thus freeing resources), and
       causes Scheduler to retry in-place resource resizing.

#### Transitions of ResourceResizeRequired PodCondition

The following diagram shows possible transitions of ResourceResizeRequired
PodCondition.Reason field.

```text

                   +----------+
                   |          |
                   | Rejected |
                   |          |
                   +----^-----+
                        |
                        |
                       5|
                   +----+----+
                   |         <-----------+
       +-----------+ (empty) |           |
       |           |         +---------+ |
       |           +--+---^--+         | |
      1|             2|   |           4| |6
 +-----v----+         |   |        +---v-+--+
 |          |         |   |        |        |
 | Awaiting |         |   |        | Failed |
 |          |         |   |        |        |
 +-------+--+         |   |        +---^----+
        2|            |   |3           |4
         |        +---v---+---+        |
         |        |           |        |
         +--------> Requested +--------+
                  |           |
                  +-----------+

```

1. Scheduler, on starting pre-emption.
1. Scheduler, after pre-emption or no pre-emption needed.
1. Kubelet, on successful resizing.
1. Scheduler or Kubelet, if not enough space on Node.
1. Any Controller, on permanent issue.
1. Initiating actor, on retry.

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

* To avoid races and possible gamification, all components should use
  max(ResourceRequirements, ResourceAllocated) when computing resources
  used by a Pod. TBD if this can be weakened when ResourceResizeRequired
  PodCondition.Reason is set to Rejected, or should the initiating actor
  update ResourceRequirements back to reclaim resources.
* If another resource update arrives when a previous update is being handled,
  that and all subsequent updates should be buffered at the Controller, and
  applied upon success/failed completion of the update that is in progress.
* Impact of memory backed emptyDir volumes: TBD - investigation needed.

### Affected Components

Pod v1 core API:
* extended model,
* added validation.

Admission Controllers: LimitRanger, ResourceQuota need to support Pod Updates:
* for ResourceQuota it should be enough to change podEvaluator.Handler
  implementation to allow Pod updates; max(ResourceRequirements, ResourceAllocated)
  should be used to be in line with current ResourceQuota behavior
  which blocks resources before they are used (e.g. for Pending Pods),
* for LimitRanger TBD.

Kubelet:
* support in-place resource management,
* set PodStatus ResourceAllocated for Containers on placing the Pod on Node.
* change UpdateContainerResources CRI API so that it works for both Linux and Windows.

Scheduler:
* determine if in-place resize is possible, updates its cache depending on resizing outcome.
* updates its cache based on resizing action by Kubelet.

Controllers:
* propagate Template resources update to running Pod instances.
* initiate resource update retries or reschedule pod (controlled by policy) that failed resize.

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

