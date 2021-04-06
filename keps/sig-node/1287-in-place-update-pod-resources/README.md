# In-place Update of Pod Resources

## Table of Contents

<!-- toc -->
- [Summary](#summary)
- [Motivation](#motivation)
  - [Goals](#goals)
  - [Non-Goals](#non-goals)
- [Proposal](#proposal)
  - [API Changes](#api-changes)
    - [Subresource](#subresource)
    - [Container Resize Policy](#container-resize-policy)
    - [Resize Status](#resize-status)
    - [CRI Changes](#cri-changes)
  - [Kubelet and API Server Interaction](#kubelet-and-api-server-interaction)
    - [Kubelet Restart Tolerance](#kubelet-restart-tolerance)
  - [Scheduler and API Server Interaction](#scheduler-and-api-server-interaction)
  - [Flow Control](#flow-control)
    - [Container resource limit update ordering](#container-resource-limit-update-ordering)
    - [Container resource limit update failure handling](#container-resource-limit-update-failure-handling)
    - [Notes](#notes)
  - [Affected Components](#affected-components)
  - [Future Enhancements](#future-enhancements)
  - [Risks and Mitigations](#risks-and-mitigations)
- [Test Plan](#test-plan)
  - [Unit Tests](#unit-tests)
  - [Pod Resize E2E Tests](#pod-resize-e2e-tests)
  - [Resource Quota and Limit Ranges](#resource-quota-and-limit-ranges)
  - [Resize Policy Tests](#resize-policy-tests)
  - [Backward Compatibility and Negative Tests](#backward-compatibility-and-negative-tests)
- [Graduation Criteria](#graduation-criteria)
  - [Alpha](#alpha)
  - [Beta](#beta)
  - [Stable](#stable)
- [Implementation History](#implementation-history)
<!-- /toc -->

## Summary

This proposal aims at allowing Pod resource requests & limits to be updated
in-place, without a need to restart the Pod or its Containers.

The **core idea** behind the proposal is to make PodSpec mutable with regards to
Resources, denoting **desired** resources. Additionally, PodStatus is extended to
reflect resources **allocated** to a Pod and to provide information about
**actual** resources applied to the Pod and its Containers.

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

* Primary: allow to change container resource requests & limits without
  necessarily restarting the container.
* Secondary: allow actors (users, VPA, StatefulSet, JobController) to decide
  how to proceed if in-place resource resize is not possible.
* Secondary: allow users to specify which Containers can be resized without a
  restart.

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
limits. PodStatus is extended to show the resources allocated for and applied
to the Pod and its Containers.

Thanks to the above:
* Pod.Spec.Containers[i].Resources becomes purely a declaration, denoting the
  **desired** state of Pod resources,
* Pod.Status.ContainerStatuses[i].ResourcesAllocated (new field, type
  v1.ResourceList) denotes the Node resources **allocated** to the Pod and its
  Containers,
* Pod.Status.ContainerStatuses[i].Resources (new field, type
  v1.ResourceRequirements) shows the **actual** resources held by the Pod and
  its Containers.
* Pod.Status.Resize (new field, type map[string]string) explains what is
  happening for a given resource on a given container.

The new `ResourcesAllocated` field represents in-flight resize operations and
is driven by state kept in the node checkpoint.  Schedulers should use the
larger of `Spec.Containers[i].Resources` and
`Status.ContainerStatuses[i].ResourcesAllocated` when considering available
space on a node.

#### Subresource

For alpha, resource changes will be made by updating the pod spec.  For beta
(or maybe a followup in alpha), a new subresource, /resize, will be defined.
This subresource could eventually apply to other resources that carry
PodTemplates, such as Deployments, ReplicaSets, Jobs, and StatefulSets.  This
will allow users to grant RBAC access to controllers like VPA without allowing
full write access to pod specs.

The exact API here is TBD.

#### Container Resize Policy

To provide fine-grained user control, PodSpec.Containers is extended with
ResizePolicy - a list of named subobjects (new object) that supports 'cpu' and
'memory' as names. It supports the following policy values:
* RestartNotRequired - default value; try to resize the Container without
  restarting it, if possible.
* Restart - the container requires a restart to apply new resource values.
  (e.g.  Java process needs to change its Xmx flag) By using ResizePolicy, user
  can mark Containers as safe (or unsafe) for in-place resource update. Kubelet
  uses it to determine the required action.

Note: `RestartNotRequired` does not *guarantee* that a container won't be
restarted. The runtime may choose to stop the container if it is unable to
apply the new resources without doing so.

Setting the flag to separately control CPU & memory is due to an observation
that usually CPU can be added/removed without much problem whereas changes to
available memory are more probable to require restarts.

If more than one resource type with different policies are updated at the same
time, then any `Restart` policy takes precedence over `RestartNotRequired` policies.

If a pod's RestartPolicy is `Never`, the ResizePolicy fields must be set to
`RestartNotRequired` to pass validation.  That said, any in-place resize may
result in the container being stopped *and not restarted*, if the system can
not perform the resize in place.

#### Resize Status

In addition to the above, a new field `Pod.Status.Resize[]`
will be added.  This field indicates whether kubelet has accepted or rejected a
proposed resize operation for a given resource.  Any time the
`Pod.Spec.Containers[i].Resources.Requests` field differs from the
`Pod.Status.ContainerStatuses[i].Resources` field, this new field explains why.

This field can be set to one of the following values:
* Proposed - the proposed resize (in Spec...Resources) has not been accepted or
  rejected yet.
* InProgress - the proposed resize has been accepted and is being actuated.
* Deferred - the proposed resize is feasible in theory (it fits on this node)
  but is not possible right now; it will be re-evaluated.
* Infeasible - the proposed resize is not feasible and is rejected; it will not
  be re-evaluated.
* (no value) - there is no proposed resize

Any time the apiserver observes a proposed resize (a modification of a
`Spec...Resources` field), it will automatically set this field to "Proposed".

To make this field future-safe, consumers should assume that any unknown value
means the same as "Deferred".

#### CRI Changes

Kubelet calls UpdateContainerResources CRI API which currently takes
*runtimeapi.LinuxContainerResources* parameter that works for Docker and Kata,
but not for Windows. This parameter changes to *runtimeapi.ContainerResources*,
that is runtime agnostic, and will contain platform-specific information.

Additionally, ContainerStatus CRI API is extended to hold
*runtimeapi.ContainerResources* so that it allows Kubelet to query Container's
CPU and memory limit configurations from runtime.

These CRI changes are a separate effort that does not affect the design
proposed in this KEP.

### Kubelet and API Server Interaction

When a new Pod is created, Scheduler is responsible for selecting a suitable
Node that accommodates the Pod.

For a newly created Pod, the apiserver will set the `ResourcesAllocated` field
to match `Resources.Requests` for each container. When Kubelet admits a
Pod, values in `ResourcesAllocated` are used to determine if there is enough
room to admit the Pod. Kubelet does not set `ResourcesAllocated` when admitting
a Pod.

When a Pod resize is requested, Kubelet attempts to update the resources
allocated to the Pod and its Containers. Kubelet first checks if the new
desired resources can fit the Node allocable resources by computing the sum of
resources allocated (Pod.Spec.Containers[i].ResourcesAllocated) for all Pods in
the Node, except the Pod being resized. For the Pod being resized, it adds the
new desired resources (i.e Spec.Containers[i].Resources.Requests) to the sum.
* If new desired resources fit, Kubelet accepts the resize by updating
  Status...ResourcesAllocated field and setting Status.Resize to
  "InProgress".  It then invokes the UpdateContainerResources CRI API to update
  Container resource limits.  Once all Containers are successfully updated, it
  updates Status...Resources to reflect new resource values and unsets
  Status.Resize.
* If new desired resources don't fit, Kubelet will update the Status.Resize
  field to "Infeasible" and does not act on the resize.
* If new desired resources fit but are in-use at the moment, Kubelet will
  update the Status.Resize field to "Deferred".

In addition to the above, kubelet will generate Events on the Pod whenever a
resize is accepted or rejected, and if possible at key steps during the resize
process.  This will allow humans to know that progress is being made.

If multiple Pods need resizing, they are handled sequentially in an order
defined by the Kubelet (e.g. in order of arrivial).

Scheduler may, in parallel, assign a new Pod to the Node because it uses cached
Pods to compute Node allocable values. If this race condition occurs, Kubelet
resolves it by rejecting that new Pod if the Node has no room after Pod resize.

Note: After a Pod is rejected, the scheduler could try to reschedule the
replacement pod on the same node that just rejected it.  This is a general
statement about Kubernetes and is outside the scope of this KEP.

#### Kubelet Restart Tolerance

If Kubelet were to restart amidst handling a Pod resize, then upon restart, all
Pods are admitted at their current Status...ResourcesAllocated
values, and resizes are handled after all existing Pods have been added. This
ensures that resizes don't affect previously admitted existing Pods.

### Scheduler and API Server Interaction

Scheduler continues to use Pod's Spec.Containers[i].Resources.Requests for
scheduling new Pods, and continues to watch Pod updates, and updates its cache.
To compute the Node resources allocated to Pods, it must consider pending
resizes, as described by Status.Resize.

For containers which have Status.Resize = "InProgress" or "Infeasible", it can
simply use Status.ContainerStatus[i].ResourcesAllocated.

For containers which have Status.Resize = "Proposed", it must be pessimistic
and assume that the resize will be imminently accepted.  Therefore it must use
the larger of the Pod's Spec...Resources.Requests and
Status...ResourcesAllocated values

### Flow Control

The following steps denote the flow of a series of in-place resize operations
for a Pod with ResizePolicy set to RestartNotRequired for all its Containers.
This is intentionally hitting various edge-cases to demonstrate.

```
T=0: A new pod is created
    - `spec.containers[0].resources.requests[cpu]` = 1
    - all status is unset

T=1: apiserver defaults are applied
    - `spec.containers[0].resources.requests[cpu]` = 1
    - `status.containerStatuses[0].resourcesAllocated[cpu]` = 1
    - `status.resize[cpu]` = unset

T=2: kubelet runs the pod and updates the API
    - `spec.containers[0].resources.requests[cpu]` = 1
    - `status.containerStatuses[0].resourcesAllocated[cpu]` = 1
    - `status.resize[cpu]` = unset
    - `status.containerStatuses[0].resources.requests[cpu]` = 1

T=3: Resize #1: cpu = 1.5 (via PUT or PATCH or /resize)
    - apiserver validates the request (e.g. `limits` are not below
      `requests`, ResourceQuota not exceeded, etc) and accepts the operation
    - apiserver sets `resize[cpu]` to "Proposed"
    - `spec.containers[0].resources.requests[cpu]` = 1.5
    - `status.containerStatuses[0].resourcesAllocated[cpu]` = 1
    - `status.resize[cpu]` = "Proposed"
    - `status.containerStatuses[0].resources.requests[cpu]` = 1

T=4: Kubelet watching the pod sees resize #1 and accepts it
    - kubelet sends patch {
        `resourceVersion` = `<previous value>` # enable conflict detection
        `status.containerStatuses[0].resourcesAllocated[cpu]` = 1.5
        `status.resize[cpu]` = "InProgress"'
      }
    - `spec.containers[0].resources.requests[cpu]` = 1.5
    - `status.containerStatuses[0].resourcesAllocated[cpu]` = 1.5
    - `status.resize[cpu]` = "InProgress"
    - `status.containerStatuses[0].resources.requests[cpu]` = 1

T=5: Resize #2: cpu = 2
    - apiserver validates the request and accepts the operation
    - apiserver sets `resize[cpu]` to "Proposed"
    - `spec.containers[0].resources.requests[cpu]` = 2
    - `status.containerStatuses[0].resourcesAllocated[cpu]` = 1.5
    - `status.resize[cpu]` = "Proposed"
    - `status.containerStatuses[0].resources.requests[cpu]` = 1

T=6: Container runtime applied cpu=1.5
    - kubelet sends patch {
        `resourceVersion` = `<previous value>` # enable conflict detection
        `status.containerStatuses[0].resources.requests[cpu]` = 1.5
        `status.resize[cpu]` = unset
      }
    - apiserver fails the operation with a "conflict" error

T=7: kubelet refreshes and sees resize #2 (cpu = 2)
    - kubelet decides this is possible, but not right now
    - kubelet sends patch {
        `resourceVersion` = `<updated value>` # enable conflict detection
        `status.containerStatuses[0].resources.requests[cpu]` = 1.5
        `status.resize[cpu]` = "Deferred"
      }
    - `spec.containers[0].resources.requests[cpu]` = 2
    - `status.containerStatuses[0].resourcesAllocated[cpu]` = 1.5
    - `status.resize[cpu]` = "Deferred"
    - `status.containerStatuses[0].resources.requests[cpu]` = 1.5

T=8: Resize #3: cpu = 1.6
    - apiserver validates the request and accepts the operation
    - apiserver sets `resize[cpu]` to "Proposed"
    - `spec.containers[0].resources.requests[cpu]` = 1.6
    - `status.containerStatuses[0].resourcesAllocated[cpu]` = 1.5
    - `status.resize[cpu]` = "Proposed"
    - `status.containerStatuses[0].resources.requests[cpu]` = 1.5

T=9: Kubelet watching the pod sees resize #3 and accepts it
    - kubelet sends patch {
        `resourceVersion` = `<previous value>` # enable conflict detection
        `status.containerStatuses[0].resourcesAllocated[cpu]` = 1.6
        `status.resize[cpu]` = "InProgress"'
      }
    - `spec.containers[0].resources.requests[cpu]` = 1.6
    - `status.containerStatuses[0].resourcesAllocated[cpu]` = 1.6
    - `status.resize[cpu]` = "InProgress"
    - `status.containerStatuses[0].resources.requests[cpu]` = 1.5

T=10: Container runtime applied cpu=1.6
    - kubelet sends patch {
        `resourceVersion` = `<previous value>` # enable conflict detection
        `status.containerStatuses[0].resources.requests[cpu]` = 1.6
        `status.resize[cpu]` = unset
      }
    - `spec.containers[0].resources.requests[cpu]` = 1.6
    - `status.containerStatuses[0].resourcesAllocated[cpu]` = 1.6
    - `status.resize[cpu]` = unset
    - `status.containerStatuses[0].resources.requests[cpu]` = 1.6

T=11: Resize #4: cpu = 100
    - apiserver validates the request and accepts the operation
    - apiserver sets `resize[cpu]` to "Proposed"
    - `spec.containers[0].resources.requests[cpu]` = 100
    - `status.containerStatuses[0].resourcesAllocated[cpu]` = 1.6
    - `status.resize[cpu]` = "Proposed"
    - `status.containerStatuses[0].resources.requests[cpu]` = 1.6

T=12: Kubelet watching the pod sees resize #4
    - this node does not have 100 CPUs, so kubelet cannot accept
    - kubelet sends patch {
        `resourceVersion` = `<previous value>` # enable conflict detection
        `status.resize[cpu]` = "Infeasible"'
      }
    - `spec.containers[0].resources.requests[cpu]` = 100
    - `status.containerStatuses[0].resourcesAllocated[cpu]` = 1.6
    - `status.resize[cpu]` = "Infeasible"
    - `status.containerStatuses[0].resources.requests[cpu]` = 1.6
```

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

#### Container resource limit update failure handling

If multiple Containers in a Pod are being updated, and UpdateContainerResources
CRI API fails for any of the containers, Kubelet will backoff and retry at a
later time. Kubelet does not attempt to update limits for containers that are
lined up for update after the failing container. This ensures that sum of the
container limits does not exceed Pod-level cgroup limit at any point. Once all
the container limits have been successfully updated, Kubelet updates the Pod's
Status.ContainerStatuses[i].Resources to match the desired limit values.

#### Notes

* If CPU Manager policy for a Node is set to 'static', then only integral
  values of CPU resize are allowed. If non-integral CPU resize is requested
  for a Node with 'static' CPU Manager policy, that resize is rejected, and
  an error message is logged to the event stream.
* To avoid races and possible gamification, all components will use Pod's
  Status.ContainerStatuses[i].ResourcesAllocated when computing resources used
  by Pods.
* If additional resize requests arrive when a Pod is being resized, those
  requests are handled after completion of the resize that is in progress. And
  resize is driven towards the latest desired state.
* Lowering memory limits may not always take effect quickly if the application
  is holding on to pages. Kubelet will use a control loop to set the memory
  limits near usage in order to force a reclaim, and update the Pod's
  Status.ContainerStatuses[i].Resources only when limit is at desired value.
* Impact of Pod Overhead: Kubelet adds Pod Overhead to the resize request to
  determine if in-place resize is possible.
* At this time, Vertical Pod Autoscaler should not be used with Horizontal Pod
  Autoscaler on CPU, memory. This enhancement does not change that limitation.

### Affected Components

Pod v1 core API:
* extend API
* auto-reset Status.Resize on changes to Resources
* added validation allowing only CPU and memory resource changes,
* init ResourcesAllocated on Create (but not update)
* set default for ResizePolicy

Admission Controllers: LimitRanger, ResourceQuota need to support Pod Updates:
* for ResourceQuota, podEvaluator.Handler implementation is modified to allow
  Pod updates, and verify that sum of Pod.Spec.Containers[i].Resources for all
  Pods in the Namespace don't exceed quota,
* PodResourceAllocation admission plugin is ordered before ResourceQuota.
* for LimitRanger we check that a resize request does not violate the min and
  max limits specified in LimitRange for the Pod's namespace.

Kubelet:
* set Pod's Status.ContainerStatuses[i].Resources for Containers upon placing
  a new Pod on the Node,
* update Pod's Status.Resize and Status...ResourcesAllocated upon resize,
* change UpdateContainerResources CRI API to work for both Linux & Windows.

Scheduler:
* compute resource allocations using ResourcesAllocated.

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
1. Extend controllers (Job, Deployment, etc) to propagate Template resources
   update to running Pods.
1. Allow resizing local ephemeral storage.
1. Allow resource limits to be updated (VPA feature).
1. Handle pod-scoped resources (https://github.com/kubernetes/enhancements/pull/1592)

### Risks and Mitigations

1. Backward compatibility: When Pod.Spec.Containers[i].Resources becomes
   representative of desired state, and Pod's true resource allocations are
   tracked in Pod.Status.ContainerStatuses[i].ResourcesAllocated, applications
   that query PodSpec and rely on Resources in PodSpec to determine resource
   allocations will see values that may not represent actual allocations. As a
   mitigation, this change needs to be documented and highlighted in the
   release notes, and in top-level Kubernetes documents.
1. Resizing memory lower: Lowering cgroup memory limits may not work as pages
   could be in use, and approaches such as setting limit near current usage may
   be required. This issue needs further investigation.
1. Older client versions: Previous versions of clients that are unaware of the
   new ResourcesAllocated and ResizePolicy fields would set them to nil. To
   keep compatibility, PodResourceAllocation admission controller mutates such
   an update by copying non-nil values from the old Pod to current Pod.

## Test Plan

### Unit Tests

Unit tests will cover the sanity of code changes that implements the feature,
and the policy controls that are introduced as part of this feature.

### Pod Resize E2E Tests

End-to-End tests resize a Pod via PATCH to Pod's Spec.Containers[i].Resources.
The e2e tests use docker as container runtime.
  - Resizing of Requests are verified by querying the values in Pod's
    Status.ContainerStatuses[i].ResourcesAllocated field.
  - Resizing of Limits are verified by querying the cgroup limits of the Pod's
    containers.

E2E test cases for Guaranteed class Pod with one container:
1. Increase, decrease Requests & Limits for CPU only.
1. Increase, decrease Requests & Limits for memory only.
1. Increase, decrease Requests & Limits for CPU and memory.
1. Increase CPU and decrease memory.
1. Decrease CPU and increase memory.

E2E test cases for Burstable class single container Pod that specifies
both CPU & memory:
1. Increase, decrease Requests - CPU only.
1. Increase, decrease Requests - memory only.
1. Increase, decrease Requests - both CPU & memory.
1. Increase, decrease Limits - CPU only.
1. Increase, decrease Limits - memory only.
1. Increase, decrease Limits - both CPU & memory.
1. Increase, decrease Requests & Limits - CPU only.
1. Increase, decrease Requests & Limits - memory only.
1. Increase, decrease Requests & Limits - both CPU and memory.
1. Increase CPU (Requests+Limits) & decrease memory(Requests+Limits).
1. Decrease CPU (Requests+Limits) & increase memory(Requests+Limits).
1. Increase CPU Requests while decreasing CPU Limits.
1. Decrease CPU Requests while increasing CPU Limits.
1. Increase memory Requests while decreasing memory Limits.
1. Decrease memory Requests while increasing memory Limits.
1. CPU: increase Requests, decrease Limits, Memory: increase Requests, decrease Limits.
1. CPU: decrease Requests, increase Limits, Memory: decrease Requests, increase Limits.

E2E tests for Burstable class single container Pod that specifies CPU only:
1. Increase, decrease CPU - Requests only.
1. Increase, decrease CPU - Limits only.
1. Increase, decrease CPU - both Requests & Limits.

E2E tests for Burstable class single container Pod that specifies memory only:
1. Increase, decrease memory - Requests only.
1. Increase, decrease memory - Limits only.
1. Increase, decrease memory - both Requests & Limits.

E2E tests for Guaranteed class Pod with three containers (c1, c2, c3):
1. Increase CPU & memory for all three containers.
1. Decrease CPU & memory for all three containers.
1. Increase CPU, decrease memory for all three containers.
1. Decrease CPU, increase memory for all three containers.
1. Increase CPU for c1, decrease c2, c3 unchanged - no net CPU change.
1. Increase memory for c1, decrease c2, c3 unchanged - no net memory change.
1. Increase CPU for c1, decrease c2 & c3 - net CPU decrease for Pod.
1. Increase memory for c1, decrease c2 & c3 - net memory decrease for Pod.
1. Increase CPU for c1 & c3, decrease c2 - net CPU increase for Pod.
1. Increase memory for c1 & c3, decrease c2 - net memory increase for Pod.

### Resource Quota and Limit Ranges

Setup a namespace with ResourceQuota and a single, valid Pod.
1. Resize the Pod within resource quota - CPU only.
1. Resize the Pod within resource quota - memory only.
1. Resize the Pod within resource quota - both CPU and memory.
1. Resize the Pod to exceed resource quota - CPU only.
1. Resize the Pod to exceed resource quota - memory only.
1. Resize the Pod to exceed resource quota - both CPU and memory.

Setup a namespace with min and max LimitRange and create a single, valid Pod.
1. Increase, decrease CPU within min/max bounds.
1. Increase CPU to exceed max value.
1. Decrease CPU to go below min value.
1. Increase memory to exceed max value.
1. Decrease memory to go below min value.

### Resize Policy Tests

Setup a guaranteed class Pod with two containers (c1 & c2).
1. No resize policy specified, defaults to RestartNotRequired. Verify that CPU and
   memory are resized without restarting containers.
1. RestartNotRequired (cpu, memory) policy for c1, Restart (cpu, memory) for c2.
   Verify that c1 is resized without restart, c2 is restarted on resize.
1. RestartNotRequired cpu, Restart memory policy for c1. Resize c1 CPU only,
   verify container is resized without restart.
1. RestartNotRequired cpu, Restart memory policy for c1. Resize c1 memory only,
   verify container is resized with restart.
1. RestartNotRequired cpu, Restart memory policy for c1. Resize c1 CPU & memory,
   verify container is resized with restart.

### Backward Compatibility and Negative Tests

1. Verify that Node is allowed to update only a Pod's ResourcesAllocated field.
1. Verify that only Node account is allowed to udate ResourcesAllocated field.
1. Verify that updating Pod Resources in workload template spec retains current
   behavior:
   - Updating Pod Resources in Job template is not allowed.
   - Updating Pod Resources in Deployment template continues to result in Pod
     being restarted with updated resources.
1. Verify Pod updates by older version of client-go doesn't result in current
   values of ResourcesAllocated and ResizePolicy fields being dropped.
1. Verify that only CPU and memory resources are mutable by user.

TODO: Identify more cases

## Graduation Criteria

### Alpha
- In-Place Pod Resouces Update functionality is implemented for running Pods,
- LimitRanger and ResourceQuota handling are added,
- Resize Policies functionality is implemented,
- Unit tests and E2E tests covering basic functionality are added,
- E2E tests covering multiple containers are added.

### Beta
- VPA alpha integration of feature completed and any bugs addressed,
- E2E tests covering Resize Policy, LimitRanger, and ResourceQuota are added,
- Negative tests are identified and added.
- A "/resize" subresource is defined and implemented.
- Pod-scoped resources are handled if that KEP is past alpha

### Stable
- VPA integration of feature moved to beta,
- User feedback (ideally from atleast two distinct users) is green,
- No major bugs reported for three months.
- Pod-scoped resources are handled if that KEP is past alpha

## Implementation History

- 2018-11-06 - initial KEP draft created
- 2019-01-18 - implementation proposal extended
- 2019-03-07 - changes to flow control, updates per review feedback
- 2019-08-29 - updated design proposal
- 2019-10-25 - update key open items and move KEP to implementable
- 2020-01-06 - API review suggested changes incorporated
- 2020-01-13 - Test plan and graduation criteria added
- 2020-01-21 - Graduation criteria updated per review feedback
- 2020-11-06 - Updated with feedback from reviews
- 2020-12-09 - Add "Deferred"
- 2021-02-05 - Final consensus on resourcesAllocated[] and resize[]
