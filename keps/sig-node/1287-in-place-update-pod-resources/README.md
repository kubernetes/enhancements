# In-place Update of Pod Resources

## Table of Contents

<!-- toc -->
- [Release Signoff Checklist](#release-signoff-checklist)
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
  - [Risks and Mitigations](#risks-and-mitigations)
- [Design Details](#design-details)
  - [Kubelet and API Server Interaction](#kubelet-and-api-server-interaction)
    - [Kubelet Restart Tolerance](#kubelet-restart-tolerance)
  - [Scheduler and API Server Interaction](#scheduler-and-api-server-interaction)
  - [Flow Control](#flow-control)
    - [Container resource limit update ordering](#container-resource-limit-update-ordering)
    - [Container resource limit update failure handling](#container-resource-limit-update-failure-handling)
    - [CRI Changes Flow](#cri-changes-flow)
    - [Notes](#notes)
  - [Affected Components](#affected-components)
  - [Future Enhancements](#future-enhancements)
  - [Test Plan](#test-plan)
    - [Prerequisite testing updates](#prerequisite-testing-updates)
    - [Unit Tests](#unit-tests)
    - [Pod Resize E2E Tests](#pod-resize-e2e-tests)
    - [CRI E2E Tests](#cri-e2e-tests)
    - [Resource Quota and Limit Ranges](#resource-quota-and-limit-ranges)
    - [Resize Policy Tests](#resize-policy-tests)
    - [Backward Compatibility and Negative Tests](#backward-compatibility-and-negative-tests)
  - [Graduation Criteria](#graduation-criteria)
    - [Alpha](#alpha)
    - [Beta](#beta)
    - [Stable](#stable)
  - [Upgrade / Downgrade Strategy](#upgrade--downgrade-strategy)
  - [Version Skew Strategy](#version-skew-strategy)
- [Production Readiness Review Questionnaire](#production-readiness-review-questionnaire)
  - [Feature Enablement and Rollback](#feature-enablement-and-rollback)
  - [Rollout, Upgrade and Rollback Planning](#rollout-upgrade-and-rollback-planning)
  - [Monitoring Requirements](#monitoring-requirements)
  - [Dependencies](#dependencies)
  - [Scalability](#scalability)
  - [Troubleshooting](#troubleshooting)
- [Implementation History](#implementation-history)
- [Drawbacks](#drawbacks)
- [Alternatives](#alternatives)
<!-- /toc -->

## Release Signoff Checklist

<!--
**ACTION REQUIRED:** In order to merge code into a release, there must be an
issue in [kubernetes/enhancements] referencing this KEP and targeting a release
milestone **before the [Enhancement Freeze](https://git.k8s.io/sig-release/releases)
of the targeted release**.

For enhancements that make changes to code or processes/procedures in core
Kubernetes—i.e., [kubernetes/kubernetes], we require the following Release
Signoff checklist to be completed.

Check these off as they are completed for the Release Team to track. These
checklist items _must_ be updated for the enhancement to be released.
-->

Items marked with (R) are required *prior to targeting to a milestone / release*.

- [ ] (R) Enhancement issue in release milestone, which links to KEP dir in [kubernetes/enhancements] (not the initial KEP PR)
- [ ] (R) KEP approvers have approved the KEP status as `implementable`
- [ ] (R) Design details are appropriately documented
- [ ] (R) Test plan is in place, giving consideration to SIG Architecture and SIG Testing input (including test refactors)
  - [ ] e2e Tests for all Beta API Operations (endpoints)
  - [ ] (R) Ensure GA e2e tests for meet requirements for [Conformance Tests](https://github.com/kubernetes/community/blob/master/contributors/devel/sig-architecture/conformance-tests.md)
  - [ ] (R) Minimum Two Week Window for GA e2e tests to prove flake free
- [ ] (R) Graduation criteria is in place
  - [ ] (R) [all GA Endpoints](https://github.com/kubernetes/community/pull/1806) must be hit by [Conformance Tests](https://github.com/kubernetes/community/blob/master/contributors/devel/sig-architecture/conformance-tests.md)
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

This proposal also aims to improve the Container Runtime Interface (CRI) APIs for
managing a Container's CPU and memory resource configurations on the runtime.
It seeks to extend UpdateContainerResources CRI API such that it works for
Windows, and other future runtimes besides Linux. It also seeks to extend
ContainerStatus CRI API to allow Kubelet to discover the current resources
configured on a Container.

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

Additioally, In-Place Pod Vertical Scaling feature relies on Container Runtime
Interface (CRI) to update CPU and/or memory requests/limits for a Pod's Container(s).

The current CRI API set has a few drawbacks that need to be addressed:
1. UpdateContainerResources CRI API takes a parameter that describes Container
   resources to update for Linux Containers, and this may not work for Windows
   Containers or other potential non-Linux runtimes in the future.
1. There is no CRI mechanism that lets Kubelet query and discover the CPU and
   memory limits configured on a Container from the Container runtime.
1. The expected behavior from a runtime that handles UpdateContainerResources
   CRI API is not very well defined or documented.

### Goals

* Primary: allow to change container resource requests & limits without
  necessarily restarting the container.
* Secondary: allow actors (users, VPA, StatefulSet, JobController) to decide
  how to proceed if in-place resource resize is not possible.
* Secondary: allow users to specify which Containers can be resized without a
  restart.

Additionally, this proposal has two goals for CRI:
  - Modify UpdateContainerResources to allow it to work for Windows Containers,
    as well as Containers managed by other runtimes besides Linux,
  - Provide CRI API mechanism to query the Container runtime for CPU and memory
    resource configurations that are currently applied to a Container.

An additional goal of this proposal is to better define and document the
expected behavior of a Container runtime when handling resource updates.

### Non-Goals

The explicit non-goal of this KEP is to avoid controlling full lifecycle of a
Pod which failed in-place resource resizing. This should be handled by actors
which initiated the resizing.

Other identified non-goals are:
* allow to change Pod QoS class without a restart,
* to change resources of Init Containers without a restart,
* eviction of lower priority Pods to facilitate Pod resize,
* updating extended resources or any other resource types besides CPU, memory,
* support for CPU/memory manager policies besides the default 'None' policy.

Definition of expected behavior of a Container runtime when it handles CRI APIs
related to a Container's resources is intended to be a high level guide.  It is
a non-goal of this proposal to define a detailed or specific way to implement
these functions. Implementation specifics are left to the runtime, within the
bounds of expected behavior.

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
that is runtime agnostic, and will contain platform-specific information. This
would make UpdateContainerResources API work for Windows, and any other future
runtimes, besides Linux by making the resources parameter passed in the API
specific to the target runtime.

Additionally, ContainerStatus CRI API is extended to hold
*runtimeapi.ContainerResources* so that it allows Kubelet to query Container's
CPU and memory limit configurations from runtime. This expects runtime to respond
with CPU and memory resource values currently applied to the Container.

These CRI changes are a separate effort that does not affect the design
proposed in this KEP.

To accomplish aforementioned CRI changes:

* A new protobuf message object named *ContainerResources* that encapsulates
LinuxContainerResources and WindowsContainerResources is introduced as below.
  - This message can easily be extended for future runtimes by simply adding a
    new runtime-specific resources struct to the ContainerResources message.
```
// ContainerResources holds resource configuration for a container.
message ContainerResources {
    // Resource configuration specific to Linux container.
    LinuxContainerResources linux = 1;
    // Resource configuration specific to Windows container.
    WindowsContainerResources windows = 2;
}
```

* UpdateContainerResourcesRequest message is extended to carry
  ContainerResources field as below.
  - For Linux runtimes, Kubelet fills UpdateContainerResourcesRequest.Linux in
    additon to UpdateContainerResourcesRequest.Resources.Linux fields.
    - This keeps backward compatibility by letting runtimes that rely on the
      current LinuxContainerResources continue to work, while enabling newer
      runtime versions to use UpdateContainerResourcesRequest.Resources.Linux,
    - It enables deprecation of UpdateContainerResourcesRequest.Linux field.
```
message UpdateContainerResourcesRequest {
    // ID of the container to update.
    string container_id = 1;
    // Resource configuration specific to Linux container.
    LinuxContainerResources linux = 2;
    // Resource configuration for the container.
    ContainerResources resources = 3;
}
```

* ContainerStatus message is extended to return ContainerResources as below.
  - This enables Kubelet to query the runtime and discover resources currently
    applied to a Container using ContainerStatus CRI API.
```
@@ -914,6 +912,8 @@ message ContainerStatus {
     repeated Mount mounts = 14;
     // Log path of container.
     string log_path = 15;
+    // Resource configuration of the container.
+    ContainerResources resources = 16;
 }
```

* ContainerManager CRI API service interface is modified as below.
  - UpdateContainerResources takes ContainerResources parameter instead of
    LinuxContainerResources.
```
--- a/staging/src/k8s.io/cri-api/pkg/apis/services.go
+++ b/staging/src/k8s.io/cri-api/pkg/apis/services.go
@@ -43,8 +43,10 @@ type ContainerManager interface {
        ListContainers(filter *runtimeapi.ContainerFilter) ([]*runtimeapi.Container, error)
        // ContainerStatus returns the status of the container.
        ContainerStatus(containerID string) (*runtimeapi.ContainerStatus, error)
-       // UpdateContainerResources updates the cgroup resources for the container.
-       UpdateContainerResources(containerID string, resources *runtimeapi.LinuxContainerResources) error
+       // UpdateContainerResources updates resource configuration for the container.
+       UpdateContainerResources(containerID string, resources *runtimeapi.ContainerResources) error
        // ExecSync executes a command in the container, and returns the stdout output.
        // If command exits with a non-zero exit code, an error is returned.
        ExecSync(containerID string, cmd []string, timeout time.Duration) (stdout []byte, stderr []byte, err error)
```

* Kubelet code is modified to leverage these changes.

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

## Design Details

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

#### CRI Changes Flow

Below diagram is an overview of Kubelet using UpdateContainerResources and
ContainerStatus CRI APIs to set new container resource limits, and update the
Pod Status in response to user changing the desired resources in Pod Spec.

```
   +-----------+                   +-----------+                  +-----------+
   |           |                   |           |                  |           |
   | apiserver |                   |  kubelet  |                  |  runtime  |
   |           |                   |           |                  |           |
   +-----+-----+                   +-----+-----+                  +-----+-----+
         |                               |                              |
         |       watch (pod update)      |                              |
         |------------------------------>|                              |
         |     [Containers.Resources]    |                              |
         |                               |                              |
         |                            (admit)                           |
         |                               |                              |
         |                               |  UpdateContainerResources()  |
         |                               |----------------------------->|
         |                               |                         (set limits)
         |                               |<- - - - - - - - - - - - - - -|
         |                               |                              |
         |                               |      ContainerStatus()       |
         |                               |----------------------------->|
         |                               |                              |
         |                               |     [ContainerResources]     |
         |                               |<- - - - - - - - - - - - - - -|
         |                               |                              |
         |      update (pod status)      |                              |
         |<------------------------------|                              |
         | [ContainerStatuses.Resources] |                              |
         |                               |                              |

```

* Kubelet invokes UpdateContainerResources() CRI API in ContainerManager
  interface to configure new CPU and memory limits for a Container by
  specifying those values in ContainerResources parameter to the API. Kubelet
  sets ContainerResources parameter specific to the target runtime platform
  when calling this CRI API.

* Kubelet calls ContainerStatus() CRI API in ContainerManager interface to get
  the CPU and memory limits applied to a Container. It uses the values returned
  in ContainerStatus.Resources to update ContainerStatuses[i].Resources.Limits
  for that Container in the Pod's Status.

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

### Test Plan

<!--
**Note:** *Not required until targeted at a release.*
The goal is to ensure that we don't accept enhancements with inadequate testing.

All code is expected to have adequate tests (eventually with coverage
expectations). Please adhere to the [Kubernetes testing guidelines][testing-guidelines]
when drafting this test plan.

[testing-guidelines]: https://git.k8s.io/community/contributors/devel/sig-testing/testing.md
-->

[x] I/we understand the owners of the involved components may require updates to
existing tests to make this code solid enough prior to committing the changes necessary
to implement this enhancement.

##### Prerequisite testing updates

<!--
Based on reviewers feedback describe what additional tests need to be added prior
implementing this enhancement to ensure the enhancements have also solid foundations.
-->

#### Unit Tests

Unit tests will cover the sanity of code changes that implements the feature,
and the policy controls that are introduced as part of this feature.

CRI unit tests are updated to reflect use of ContainerResources object in
UpdateContainerResources and ContainerStatus APIs.

#### Pod Resize E2E Tests

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

#### CRI E2E Tests

1. E2E test is added to verify UpdateContainerResources API with containerd runtime.
1. E2E test is added to verify ContainerStatus API using containerd runtime.
1. E2E test is added to verify backward compatibility using containerd runtime.

#### Resource Quota and Limit Ranges

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

#### Resize Policy Tests

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

#### Backward Compatibility and Negative Tests

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

### Graduation Criteria

#### Alpha
- In-Place Pod Resouces Update functionality is implemented for running Pods,
- LimitRanger and ResourceQuota handling are added,
- Resize Policies functionality is implemented,
- Unit tests and E2E tests covering basic functionality are added,
- E2E tests covering multiple containers are added.
- UpdateContainerResources API changes are done and tested with containerd
  runtime, backward compatibility is maintained.
- ContainerStatus API changes are done. Tests are ready but not enforced.

#### Beta
- VPA alpha integration of feature completed and any bugs addressed,
- E2E tests covering Resize Policy, LimitRanger, and ResourceQuota are added,
- Negative tests are identified and added.
- A "/resize" subresource is defined and implemented.
- Pod-scoped resources are handled if that KEP is past alpha
- ContainerStatus API change tests are enforced and containerd runtime must comply.
- ContainerStatus API change tests are enforced and Windows runtime should comply.

#### Stable
- VPA integration of feature moved to beta,
- User feedback (ideally from atleast two distinct users) is green,
- No major bugs reported for three months.
- Pod-scoped resources are handled if that KEP is past alpha

### Upgrade / Downgrade Strategy
Scheduler and API server should be updated before Kubelets in that order.
Kubelet and the runtime versions should use the same CRI version in lock-step.
Upgrade involves draining all pods from a node, installing a CRI runtime with this
version of the API and update to a matching kubelet and making node schedulable again.
Downgrade involves doing the above in reverse.

### Version Skew Strategy
Kubelet and the CRI runtime versions are expected to match so we don't have to worry about.

Previous versions of clients that are unaware of the new ResizePolicy fields would set them
to nil. API server mutates such updates by copying non-nil values from old Pod to the current
Pod.

A previous version of kubelet interprets mutation to Pod Resources as a Container definition
change and will restart the container with the new Resources. This could lead to Node resource
over-subscription. In order to address this, the feature-gate will remain default false for
atleast two versions after the initial release that carries it.

## Production Readiness Review Questionnaire

<!--

Production readiness reviews are intended to ensure that features merging into
Kubernetes are observable, scalable and supportable; can be safely operated in
production environments, and can be disabled or rolled back in the event they
cause increased failures in production. See more in the PRR KEP at
https://git.k8s.io/enhancements/keps/sig-architecture/20190731-production-readiness-review-process.md.

The production readiness review questionnaire must be completed for features in
v1.19 or later, but is non-blocking at this time. That is, approval is not
required in order to be in the release.

In some cases, the questions below should also have answers in `kep.yaml`. This
is to enable automation to verify the presence of the review, and to reduce review
burden and latency.

The KEP must have a approver from the
[`prod-readiness-approvers`](http://git.k8s.io/enhancements/OWNERS_ALIASES)
team. Please reach out on the
[#prod-readiness](https://kubernetes.slack.com/archives/CPNHUMN74) channel if
you need any help or guidance.

-->

### Feature Enablement and Rollback

_This section must be completed when targeting alpha to a release._

* **How can this feature be enabled / disabled in a live cluster?**
  - [x] Feature gate (also fill in values in `kep.yaml`)
    - Feature gate name: InPlacePodVerticalScaling
    - Components depending on the feature gate: kubelet, kube-apiserver, kube-scheduler
  - [ ] Other
    - Describe the mechanism:
    - Will enabling / disabling the feature require downtime of the control
      plane? No.
    - Will enabling / disabling the feature require downtime or reprovisioning
      of a node? (Do not assume `Dynamic Kubelet Config` feature is enabled). No.

* **Does enabling the feature change any default behavior?** No

* **Can the feature be disabled once it has been enabled (i.e. can we roll back
  the enablement)?** Yes

* **What happens if we reenable the feature if it was previously rolled back?**
  - API will once again permit modification of Resources for 'cpu' and 'memory'.
  - Actual resources applied will be reflected in in Pod's ContainerStatuses.

* **Are there any tests for feature enablement/disablement?**
  Unit tests and E2E tests.
   - Unit tests verify that feature does not introduce any regression.
   - E2E tests run against a local cluster verify that feature works as expected.

### Rollout, Upgrade and Rollback Planning

_This section must be completed when targeting beta graduation to a release._

* **How can a rollout fail? Can it impact already running workloads?**
  Try to be as paranoid as possible - e.g., what if some components will restart
   mid-rollout?

* **What specific metrics should inform a rollback?**

* **Were upgrade and rollback tested? Was the upgrade->downgrade->upgrade path tested?**
  Describe manual testing that was done and the outcomes.
  Longer term, we may want to require automated upgrade/rollback tests, but we
  are missing a bunch of machinery and tooling and can't do that now.

* **Is the rollout accompanied by any deprecations and/or removals of features, APIs,
fields of API types, flags, etc.?**
  Even if applying deprecation policies, they may still surprise some users.

### Monitoring Requirements

_This section must be completed when targeting beta graduation to a release._

* **How can an operator determine if the feature is in use by workloads?**
  Ideally, this should be a metric. Operations against the Kubernetes API (e.g.,
  checking if there are objects with field X set) may be a last resort. Avoid
  logs or events for this purpose.

* **What are the SLIs (Service Level Indicators) an operator can use to determine
the health of the service?**
  - [ ] Metrics
    - Metric name:
    - [Optional] Aggregation method:
    - Components exposing the metric:
  - [ ] Other (treat as last resort)
    - Details:

* **What are the reasonable SLOs (Service Level Objectives) for the above SLIs?**
  At a high level, this usually will be in the form of "high percentile of SLI
  per day <= X". It's impossible to provide comprehensive guidance, but at the very
  high level (needs more precise definitions) those may be things like:
  - per-day percentage of API calls finishing with 5XX errors <= 1%
  - 99% percentile over day of absolute value from (job creation time minus expected
    job creation time) for cron job <= 10%
  - 99,9% of /health requests per day finish with 200 code

* **Are there any missing metrics that would be useful to have to improve observability
of this feature?**
  Describe the metrics themselves and the reasons why they weren't added (e.g., cost,
  implementation difficulties, etc.).

### Dependencies

_This section must be completed when targeting beta graduation to a release._

* **Does this feature depend on any specific services running in the cluster?**
  Think about both cluster-level services (e.g. metrics-server) as well
  as node-level agents (e.g. specific version of CRI). Focus on external or
  optional services that are needed. For example, if this feature depends on
  a cloud provider API, or upon an external software-defined storage or network
  control plane.

  For each of these, fill in the following—thinking about running existing user workloads
  and creating new ones, as well as about cluster-level services (e.g. DNS):
  - [Dependency name]
    - Usage description:
      - Impact of its outage on the feature:
      - Impact of its degraded performance or high-error rates on the feature:

### Scalability

_For alpha, this section is encouraged: reviewers should consider these questions
and attempt to answer them._

_For beta, this section is required: reviewers must answer these questions._

_For GA, this section is required: approvers should be able to confirm the
previous answers based on experience in the field._

* **Will enabling / using this feature result in any new API calls?** Yes
  Describe them, providing:
  - API call type (e.g. PATCH pods)
    - One new PATCH PodStatus API call in response to Pod resize request.
    - No additional overhead unless Pod resize is invoked.
  - estimated throughput
  - originating component(s) (e.g. Kubelet, Feature-X-controller)
    - Kubelet
  focusing mostly on:
  - components listing and/or watching resources they didn't before
  - API calls that may be triggered by changes of some Kubernetes resources
    (e.g. update of object X triggers new updates of object Y)
  - periodic API calls to reconcile state (e.g. periodic fetching state,
    heartbeats, leader election, etc.)

* **Will enabling / using this feature result in introducing new API types?** No
  Describe them, providing:
  - API type
  - Supported number of objects per cluster
  - Supported number of objects per namespace (for namespace-scoped objects)

* **Will enabling / using this feature result in any new calls to the cloud
provider?** No

* **Will enabling / using this feature result in increasing size or count of
the existing API objects?** Yes
  Describe them, providing:
  - API type(s):
  - Estimated increase in size: (e.g., new annotation of size 32B)
  - Estimated amount of new objects: (e.g., new Object X for every existing Pod)
    - type Container has new field ResizePolicy, a list that adds upto 50 bytes.
    - type PodStatus has a new field, a list that adds upto 32 bytes.
    - type ContainerStatus has new field of type v1.ResourceList that mirrors
      Container.Resources.Requests in size.
    - type ContainerStatus has new field of type v1.ResourceRequirements that
      mirrors Container.Resources in size.

* **Will enabling / using this feature result in increasing time taken by any
operations covered by [existing SLIs/SLOs]?** No
  Think about adding additional work or introducing new steps in between
  (e.g. need to do X to start a container), etc. Please describe the details.

* **Will enabling / using this feature result in non-negligible increase of
resource usage (CPU, RAM, disk, IO, ...) in any components?** No
  Things to keep in mind include: additional in-memory state, additional
  non-trivial computations, excessive access to disks (including increased log
  volume), significant amount of data sent and/or received over network, etc.
  This through this both in small and large cases, again with respect to the
  [supported limits].

### Troubleshooting

The Troubleshooting section currently serves the `Playbook` role. We may consider
splitting it into a dedicated `Playbook` document (potentially with some monitoring
details). For now, we leave it here.

_This section must be completed when targeting beta graduation to a release._

* **How does this feature react if the API server and/or etcd is unavailable?**

* **What are other known failure modes?**
  For each of them, fill in the following information by copying the below template:
  - [Failure mode brief description]
    - Detection: How can it be detected via metrics? Stated another way:
      how can an operator troubleshoot without logging into a master or worker node?
    - Mitigations: What can be done to stop the bleeding, especially for already
      running user workloads?
    - Diagnostics: What are the useful log messages and their required logging
      levels that could help debug the issue?
      Not required until feature graduated to beta.
    - Testing: Are there any tests for failure mode? If not, describe why.

* **What steps should be taken if SLOs are not being met to determine the problem?**

[supported limits]: https://git.k8s.io/community//sig-scalability/configs-and-limits/thresholds.md
[existing SLIs/SLOs]: https://git.k8s.io/community/sig-scalability/slos/slos.md#kubernetes-slisslos

## Implementation History

- 2018-11-06 - initial KEP draft created
- 2019-01-18 - implementation proposal extended
- 2019-03-07 - changes to flow control, updates per review feedback
- 2019-08-29 - updated design proposal
- 2019-10-25 - Initial CRI changes KEP draft created
- 2019-10-25 - update key open items and move KEP to implementable
- 2020-01-06 - API review suggested changes incorporated
- 2020-01-13 - Test plan and graduation criteria added
- 2020-01-14 - CRI changes test plan and graduation criteria added
- 2020-01-21 - Graduation criteria updated per review feedback
- 2020-11-06 - Updated with feedback from reviews
- 2020-12-09 - Add "Deferred"
- 2021-02-05 - Final consensus on resourcesAllocated[] and resize[]
- 2022-05-01 - KEP 2273-kubelet-container-resources-cri-api-changes merged with this KEP

## Drawbacks

<!--
Why should this KEP _not_ be implemented?
-->

There are no drawbacks that we are aware of.

## Alternatives

<!--
What other approaches did you consider, and why did you rule them out? These do
not need to be as detailed as the proposal, but should include enough
information to express the idea and why it was not acceptable.
-->

We considered having scheduler approve the resize. We also considered PodSpec as
the location to checkpoint allocated resources.
