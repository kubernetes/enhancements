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
    - [Validation](#validation)
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
  - [Lifecycle Nuances](#lifecycle-nuances)
  - [Atomic Resizes](#atomic-resizes)
  - [Sidecars](#sidecars)
  - [QOS Class](#qos-class)
  - [Resource Quota](#resource-quota)
  - [Affected Components](#affected-components)
  - [Instrumentation](#instrumentation)
  - [Future Enhancements](#future-enhancements)
  - [Test Plan](#test-plan)
    - [Prerequisite testing updates](#prerequisite-testing-updates)
    - [Unit Tests](#unit-tests)
    - [Integration tests](#integration-tests)
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
* allow to change Pod QoS class
* to change resources of non-restartable InitContainers
* eviction of lower priority Pods to facilitate Pod resize
* updating extended resources or any other resource types besides CPU, memory
* support for CPU/memory manager policies besides the default 'None' policy

Definition of expected behavior of a Container runtime when it handles CRI APIs
related to a Container's resources is intended to be a high level guide.  It is
a non-goal of this proposal to define a detailed or specific way to implement
these functions. Implementation specifics are left to the runtime, within the
bounds of expected behavior.

## Proposal

### API Changes

Container resource requests & limits can now be mutated by via the `/resize` pod subresource.
PodStatus is extended to show the resources allocated for and applied to the Pod and its Containers.

Thanks to the above:
* Pod.Spec.Containers[i].Resources becomes purely a declaration, denoting the
  **desired** state of Pod resources,
* Pod.Status.ContainerStatuses[i].AllocatedResources (new field, type
  v1.ResourceList) denotes the Node resources **allocated** to the Pod and its
  Containers,
* Pod.Status.ContainerStatuses[i].Resources (new field, type
  v1.ResourceRequirements) shows the **actual** resources held by the Pod and
  its Containers.
* Pod.Status.Resize (new field, type map[string]string) explains what is
  happening for a given resource on a given container.

The new `AllocatedResources` field represents in-flight resize operations and
is driven by state kept in the node checkpoint.  Schedulers should use the
larger of `Spec.Containers[i].Resources` and
`Status.ContainerStatuses[i].AllocatedResources` when considering available
space on a node.

Additionally, a new `Pod.Spec.Containers[i].ResizePolicy[]` field (type
`[]v1.ContainerResizePolicy`) governs whether containers need to be restarted on resize. See
[Container Resize Policy](#container-resize-policy) for more details.

#### Subresource

Resource changes can only be made via the new `/resize` subresource. The request & response types
for this subresource are the full pod object, but only the following fields are allowed to be
modified:

* `.spec.containers[*].resources`
* `.spec.initContainers[*].resources` (only for sidecars)
* `.spec.resizePolicy`

The `.status.resize` field will be reset to `Proposed` in the response, but cannot be modified in the
request.

#### Validation

Resource fields remain immutable via pod update.

The following API validation rules will be applied for updates via the `/resize` subresource:

1. Resources & ResizePolicy must be valid under pod create validation.
1. Computed QOS class cannot be lowered. See [QOS Class](#qos-class) for more details.
2. Running pods without the `Pod.Status.ContainerStatuses[i].Resources` field set cannot be resized.
   See [Version Skew Strategy](#version-skew-strategy) for more details.

#### Container Resize Policy

To provide fine-grained user control, PodSpec.Containers is extended with
ResizeRestartPolicy - a list of named subobjects (new object) that supports
'cpu' and 'memory' as names. It supports the following restart policy values:
* NotRequired - default value; resize the Container without restart, if possible.
* RestartContainer - the container requires a restart to apply new resource values.
  (e.g.  Java process needs to change its Xmx flag) By using ResizePolicy, user
  can mark Containers as safe (or unsafe) for in-place resource update. Kubelet
  uses it to determine the required action.

Note: `NotRequired` restart policy for resize does not *guarantee* that a container
won't be restarted. The runtime may choose to stop the container if it is unable to
apply the new resources without restarts.

Setting the flag to separately control CPU & memory is due to an observation
that usually CPU can be added/removed without much problem whereas changes to
available memory are more probable to require restarts.

If more than one resource type with different policies are updated at the same
time, then `RestartContainer` policy takes precedence over `NotRequired` policy.

If a pod's RestartPolicy is `Never`, the ResizePolicy fields must be set to
`NotRequired` to pass validation.  That said, any in-place resize may result
in the container being stopped *and not restarted*, if the system can not
perform the resize in place.

The `ResizePolicy` field is **mutable**, but must have an entry for every resizable resource type
with a request or limit on the container.

#### Resize Status

In addition to the above, a new field `Pod.Status.Resize[]`
will be added.  This field indicates whether kubelet has accepted or rejected a
proposed resize operation for a given resource.  Any time the
`Pod.Spec.Containers[i].Resources.Requests` field differs from the
`Pod.Status.ContainerStatuses[i].Resources` field, this new field explains why.

This field can be set to one of the following values:
* `Proposed` - the proposed resize (in Spec...Resources) has not been accepted or
  rejected yet. `resources != allocatedResources`
* `InProgress` - the proposed resize has been accepted and is being actuated. A new resize request
  will reset the status to `Proposed`.
  `resources == allocatedResources && allocatedResources != status.resources`
* `Deferred` - the proposed resize is feasible in theory (it fits on this node)
  but is not possible right now; it will be re-evaluated.
  `resources != allocatedResources`
* `Infeasible` - the proposed resize is not feasible and is rejected; it will not
  be re-evaluated.
  `resources != allocatedResources`
* (no value) - there is no proposed resize

Any time the apiserver observes a proposed resize (a modification of a
`Spec...Resources` field), it will automatically set this field to `Proposed`.

To make this field future-safe, consumers should assume that any unknown value
means the same as `Deferred`.

#### CRI Changes

Kubelet calls UpdateContainerResources CRI API which currently takes
*runtimeapi.LinuxContainerResources* parameter that works for Docker and Kata,
but not for Windows. This parameter changes to *runtimeapi.ContainerResources*,
that is platform agnostic, and will contain platform-specific information. This
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
+       // UpdateContainerResources updates ContainerConfig of the container synchronously.
+       // If runtime fails to transactionally update the requested resources, an error is returned.
+       UpdateContainerResources(containerID string, resources *runtimeapi.ContainerResources) error
        // ExecSync executes a command in the container, and returns the stdout output.
        // If command exits with a non-zero exit code, an error is returned.
        ExecSync(containerID string, cmd []string, timeout time.Duration) (stdout []byte, stderr []byte, err error)
```

* Kubelet code is modified to leverage these changes.

### Risks and Mitigations

1. Backward compatibility: When Pod.Spec.Containers[i].Resources becomes
   representative of desired state, and Pod's true resource allocations are
   tracked in Pod.Status.ContainerStatuses[i].AllocatedResources, applications
   that query PodSpec and rely on Resources in PodSpec to determine resource
   allocations will see values that may not represent actual allocations. As a
   mitigation, this change needs to be documented and highlighted in the
   release notes, and in top-level Kubernetes documents.
1. Resizing memory lower: Lowering cgroup memory limits may not work as pages
   could be in use, and approaches such as setting limit near current usage may
   be required. This issue needs further investigation.
1. Older client versions: Previous versions of clients that are unaware of the
   new AllocatedResources and ResizePolicy fields would set them to nil. To
   keep compatibility, PodResourceAllocation admission controller mutates such
   an update by copying non-nil values from the old Pod to current Pod.

## Design Details

### Kubelet and API Server Interaction

When a new Pod is created, Scheduler is responsible for selecting a suitable
Node that accommodates the Pod.

For a newly created Pod, `(Init)ContainerStatuses` will be nil until the Pod is
scheduled to a node. When Kubelet admits a Pod, it will record the admitted
requests & limits to its internal allocated resources checkpoint, and write the
admitted requests to the `AllocatedResources` field in the container status.

When a Pod resize is requested, Kubelet attempts to update the resources
allocated to the Pod and its Containers. Kubelet first checks if the new
desired resources can fit the Node allocable resources by computing the sum of
resources allocated (Pod.Spec.Containers[i].AllocatedResources) for all Pods in
the Node, except the Pod being resized. For the Pod being resized, it adds the
new desired resources (i.e Spec.Containers[i].Resources.Requests) to the sum.
* If new desired resources fit, Kubelet accepts the resize by updating
  Status...AllocatedResources field and setting Status.Resize to
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
Pods are admitted at their current Status...AllocatedResources
values, and resizes are handled after all existing Pods have been added. This
ensures that resizes don't affect previously admitted existing Pods.

### Scheduler and API Server Interaction

Scheduler continues to use Pod's Spec.Containers[i].Resources.Requests for
scheduling new Pods, and continues to watch Pod updates, and updates its cache.
To compute the Node resources allocated to Pods, it must consider pending
resizes, as described by Status.Resize.

For containers which have Status.Resize = "InProgress" or "Infeasible", it can
simply use Status.ContainerStatus[i].AllocatedResources.

For containers which have Status.Resize = "Proposed", it must be pessimistic
and assume that the resize will be imminently accepted.  Therefore it must use
the larger of the Pod's Spec...Resources.Requests and
Status...AllocatedResources values

### Flow Control

The following steps denote the flow of a series of in-place resize operations
for a Pod with ResizePolicy set to NotRequired for all its Containers.
This is intentionally hitting various edge-cases to demonstrate.

```
T=0: A new pod is created
    - `spec.containers[0].resources.requests[cpu]` = 1
    - all status is unset

T=1: apiserver defaults are applied
    - `spec.containers[0].resources.requests[cpu]` = 1
    - `status.containerStatuses` = unset
    - `status.resize[cpu]` = unset

T=2: kubelet runs the pod and updates the API
    - `spec.containers[0].resources.requests[cpu]` = 1
    - `status.containerStatuses[0].allocatedResources[cpu]` = 1
    - `status.resize[cpu]` = unset
    - `status.containerStatuses[0].resources.requests[cpu]` = 1

T=3: Resize #1: cpu = 1.5 (via PUT or PATCH or /resize)
    - apiserver validates the request (e.g. `limits` are not below
      `requests`, ResourceQuota not exceeded, etc) and accepts the operation
    - apiserver sets `resize[cpu]` to "Proposed"
    - `spec.containers[0].resources.requests[cpu]` = 1.5
    - `status.containerStatuses[0].allocatedResources[cpu]` = 1
    - `status.resize[cpu]` = "Proposed"
    - `status.containerStatuses[0].resources.requests[cpu]` = 1

T=4: Kubelet watching the pod sees resize #1 and accepts it
    - kubelet sends patch {
        `resourceVersion` = `<previous value>` # enable conflict detection
        `status.containerStatuses[0].allocatedResources[cpu]` = 1.5
        `status.resize[cpu]` = "InProgress"'
      }
    - `spec.containers[0].resources.requests[cpu]` = 1.5
    - `status.containerStatuses[0].allocatedResources[cpu]` = 1.5
    - `status.resize[cpu]` = "InProgress"
    - `status.containerStatuses[0].resources.requests[cpu]` = 1

T=5: Resize #2: cpu = 2
    - apiserver validates the request and accepts the operation
    - apiserver sets `resize[cpu]` to "Proposed"
    - `spec.containers[0].resources.requests[cpu]` = 2
    - `status.containerStatuses[0].allocatedResources[cpu]` = 1.5
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
    - `status.containerStatuses[0].allocatedResources[cpu]` = 1.5
    - `status.resize[cpu]` = "Deferred"
    - `status.containerStatuses[0].resources.requests[cpu]` = 1.5

T=8: Resize #3: cpu = 1.6
    - apiserver validates the request and accepts the operation
    - apiserver sets `resize[cpu]` to "Proposed"
    - `spec.containers[0].resources.requests[cpu]` = 1.6
    - `status.containerStatuses[0].allocatedResources[cpu]` = 1.5
    - `status.resize[cpu]` = "Proposed"
    - `status.containerStatuses[0].resources.requests[cpu]` = 1.5

T=9: Kubelet watching the pod sees resize #3 and accepts it
    - kubelet sends patch {
        `resourceVersion` = `<previous value>` # enable conflict detection
        `status.containerStatuses[0].allocatedResources[cpu]` = 1.6
        `status.resize[cpu]` = "InProgress"'
      }
    - `spec.containers[0].resources.requests[cpu]` = 1.6
    - `status.containerStatuses[0].allocatedResources[cpu]` = 1.6
    - `status.resize[cpu]` = "InProgress"
    - `status.containerStatuses[0].resources.requests[cpu]` = 1.5

T=10: Container runtime applied cpu=1.6
    - kubelet sends patch {
        `resourceVersion` = `<previous value>` # enable conflict detection
        `status.containerStatuses[0].resources.requests[cpu]` = 1.6
        `status.resize[cpu]` = unset
      }
    - `spec.containers[0].resources.requests[cpu]` = 1.6
    - `status.containerStatuses[0].allocatedResources[cpu]` = 1.6
    - `status.resize[cpu]` = unset
    - `status.containerStatuses[0].resources.requests[cpu]` = 1.6

T=11: Resize #4: cpu = 100
    - apiserver validates the request and accepts the operation
    - apiserver sets `resize[cpu]` to "Proposed"
    - `spec.containers[0].resources.requests[cpu]` = 100
    - `status.containerStatuses[0].allocatedResources[cpu]` = 1.6
    - `status.resize[cpu]` = "Proposed"
    - `status.containerStatuses[0].resources.requests[cpu]` = 1.6

T=12: Kubelet watching the pod sees resize #4
    - this node does not have 100 CPUs, so kubelet cannot accept
    - kubelet sends patch {
        `resourceVersion` = `<previous value>` # enable conflict detection
        `status.resize[cpu]` = "Infeasible"'
      }
    - `spec.containers[0].resources.requests[cpu]` = 100
    - `status.containerStatuses[0].allocatedResources[cpu]` = 1.6
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
  Status.ContainerStatuses[i].AllocatedResources when computing resources used
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

### Lifecycle Nuances

* Terminated containers can be "resized" in that the resize is permitted by the API, and the Kubelet
  will accept the changes. This makes race conditions where the container terminates around the
  resize "fail open", and prevents a resize of a terminated container from blocking the resize of a
  running container (see [Atomic Resizes](#atomic-resizes)).
* Resizing pods in a graceful shutdown state is permitted.

### Atomic Resizes

A single resize request can change multiple values, including any or all of:
* Multiple resource types
* Requests & Limits
* Multiple containers

These resource requests & limits can have interdependencies that Kubernetes may not be aware of. For
example, two containers coordinating work may need to be scaled in tandem. It probably doesn't makes
sense to scale limits independently of requests, and scaling CPU without memory could just waste
resources. To mitigate these issues and simplify the design, the Kubelet will treat the requests &
limits for all containers in the spec as a single atomic request, and won't accept any of the
changes unless all changes can be accepted. If multiple requests mutate the resources spec before
the Kubelet has accepted any of the changes, it will treat them as a single atomic request.

`AllocatedResources` only accounts for accepted requests, so the Kubelet will need to record
allocated limits in its internal checkpoint.

Note: If a second infeasible resize is made before the Kubelet allocates the first resize, there can
be a race condition where the Kubelet may or may not accept the first resize, depending on whether
it admits the first change before seeing the second. This race condition is accepted as working as
intended.

### Sidecars

Sidecars, a.k.a. resizeable InitContainers can be resized the same as regular containers. There are
no special considerations here. Non-restartable InitContainers cannot be resized.

### QOS Class

A pod's QOS class cannot be changed once the pod is started, independent of any resizes.

To clarify the discussion of QOS Class changes, the following terms are defined:

* "Original QOS Class" - The QOS class that was computed based on the original resource requests &
  limits when the pod was first created.
* "Suggested QOS Class" - The QOS class that would be computed based on the current resource
  requests & limits.

With in-place vertical scaling, the _suggested QOS Class_ must be greater than or equal to the
_original QOS Class_:

* Guaranteed pods: must maintain `requests == limits`
* Burstable pods: _can_ be resized such that `requests == limits`, but their original QOS
class will stay burstable. Must retain at least one CPU or memory request or limit.
* BestEffort pods: can be freely resized, but stay BestEffort.

Even though the suggested QOS Class is allowed to change, the original QOS class is used for all
decisions based on QOS class:

* `.status.qosClass` always reports the original QOS class
* Pod cgroup hierarchy is static, using the original QOS class
* Non-guaranteed pods remain ineligible for guaranteed CPUs or NUMA pinning
* Preemption uses the original QOS Class
* OOMScoreAdjust is calculated with the original QOS Class
* Memory pressure eviction is unaffected (doesn't consider QOS Class)

In order to maintain the original QOS class, the Kubelet will checkpoint the original QOS class.

### Resource Quota

With InPlacePodVerticalScaling enabled, resource quota needs to consider pending resizes. Similarly
to how this is handled by scheduling, resource quota will use the maximum of
`.spec.container[*].resources.requests` and `.status.containerStatuses[*].allocatedResources` to
determine the effective request values. Allocated limits are not reported by the API, so resource
quota instead uses the larger of `.spec.container[*].resources.limits` and
`.status.containerStatuses[*].resources.limits`.

To properly handle scale-down, this means that the resource quota controller now needs to evaluate
pod updates where either `.status...allocatedResources` or `.status...resources` changed.

### Affected Components

Pod v1 core API:
* extend API
* auto-reset Status.Resize on changes to Resources
* added validation allowing only CPU and memory resource changes,
* init AllocatedResources on Create (but not update)
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
* update Pod's Status.Resize and Status...AllocatedResources upon resize,
* change UpdateContainerResources CRI API to work for both Linux & Windows.

Scheduler:
* compute resource allocations using AllocatedResources.

Other components:
* check how the change of meaning of resource requests influence other
  Kubernetes components.

### Instrumentation

The following new metric will be added to track total resize requests, counted at the pod level. In
otherwords, a single pod update changing multiple containers and/or resources will count as a single
resize request.

`kubelet_container_resize_requests_total` - Total number of resize requests observed by the Kubelet.

Label: `state` - Count resize request state transitions. This closely tracks the [Resize status](#resize-status) state transitions, omitting `InProgress`. Possible values:
  - `proposed` - Initial request state
  - `infeasible` - Resize request cannot be completed.
  - `deferred` - Resize request cannot initially be completed, but will retry
  - `completed` - Resize operation completed successfully (`spec.Resources == status.Allocated == status.Resources`)
  - `canceled` - Pod was terminated before resize was completed, or a new resize request was started.

In steady state, `proposed` should equal `infeasible + completed + canceled`.

The metric is recorded as a counter instead of a gauge to ensure that usage can be tracked over
time, irrespective of scrape interval.

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

#### Prerequisite testing updates

<!--
Based on reviewers feedback describe what additional tests need to be added prior
implementing this enhancement to ensure the enhancements have also solid foundations.
-->

#### Unit Tests

Unit tests will cover the sanity of code changes that implements the feature,
and the policy controls that are introduced as part of this feature.

CRI unit tests are updated to reflect use of ContainerResources object in
UpdateContainerResources and ContainerStatus APIs.

#### Integration tests

Comprehensive E2E tests provide good coverage for alpha. We may replicate and/or move
some of the E2E tests functionality into integration tests before Beta using data from
any issues we uncover that are not covered by planned and implemented tests.

#### Pod Resize E2E Tests

End-to-End tests resize a Pod via PATCH to Pod's Spec.Containers[i].Resources.
The e2e tests use docker as container runtime.
  - Resizing of Requests are verified by querying the values in Pod's
    Status.ContainerStatuses[i].AllocatedResources field.
  - Resizing of Limits are verified by querying the cgroup limits of the Pod's
    containers.

E2E test cases for Guaranteed class Pod with one container:
1. Increase, decrease Requests & Limits for CPU only.
1. Increase, decrease Requests & Limits for memory only.
1. Increase, decrease Requests & Limits for CPU and memory.
1. Increase CPU and decrease memory.
1. Decrease CPU and increase memory.
1. Add memory request & limit for CPU only container.
1. Remove memory request & limit for CPU & memory container.

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
1. Set requests == limits, ensure QOS class remains Burstable

E2E tests for Burstable class single container Pod that specifies CPU only:
1. Increase, decrease CPU - Requests only.
1. Increase, decrease CPU - Limits only.
1. Increase, decrease CPU - both Requests & Limits.

E2E tests for Burstable class single container Pod that specifies memory only:
1. Increase, decrease memory - Requests only.
1. Increase, decrease memory - Limits only.
1. Increase, decrease memory - both Requests & Limits.

E2E tests for BestEffort class single container Pod:
1. Add CPU requests & limits, QOS class remains BestEffort
2. Add Memory requests & limits, QOS class remains BestEffort

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

E2E tests for sidecar containers
1. InitContainer, then sidecar - can increase & decrease CPU & memory of sidecar
2. Sidecar then InitContainer - can increase & decrease CPU & memory of sidecar
3. Resize sidecar along with container

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
1. No resize policy specified, defaults to NotRequired. Verify that CPU and
   memory are resized without restarting containers.
1. NotRequired (cpu, memory) policy for c1, RestartContainer (cpu, memory) for c2.
   Verify that c1 is resized without restart, c2 is restarted on resize.
1. NotRequired cpu, RestartContainer memory policy for c1. Resize c1 CPU only,
   verify container is resized without restart.
1. NotRequired cpu, RestartContainer memory policy for c1. Resize c1 memory only,
   verify container is resized with restart.
1. NotRequired cpu, RestartContainer memory policy for c1. Resize c1 CPU & memory,
   verify container is resized with restart.

#### Backward Compatibility and Negative Tests

1. Verify that Node is allowed to update only a Pod's AllocatedResources field.
1. Verify that only Node account is allowed to udate AllocatedResources field.
1. Verify that updating Pod Resources in workload template spec retains current
   behavior:
   - Updating Pod Resources in Job template is not allowed.
   - Updating Pod Resources in Deployment template continues to result in Pod
     being restarted with updated resources.
1. Verify Pod updates by older version of client-go doesn't result in current
   values of AllocatedResources and ResizePolicy fields being dropped.
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
- VPA alpha integration of feature completed and any bugs addressed.
- E2E tests covering Resize Policy, LimitRanger, and ResourceQuota are added.
- Negative tests are identified and added.
- A "/resize" subresource is defined and implemented.
- Pod-scoped resources are handled if that KEP is past alpha
- ContainerStatus API change tests are enforced and containerd runtime must comply.
- ContainerStatus API change tests are enforced and Windows runtime should comply.

#### Stable
- VPA integration of feature moved to beta,
- User feedback (ideally from at least two distinct users) is green,
- No major bugs reported for three months.
- Pod-scoped resources are handled if that KEP is past alpha

### Upgrade / Downgrade Strategy
Scheduler and API server should be updated before Kubelets in that order.
Kubelet and the runtime versions should use the same CRI version in lock-step.
Upgrade involves draining all pods from a node, installing a CRI runtime with this
version of the API and update to a matching kubelet and making node schedulable again.
Downgrade involves doing the above in reverse.

### Version Skew Strategy
CRI changes were merged in v1.25 in order to enable runtimes to implement support.
  - containerd added support for this feature in 1.6.9

Previous versions of clients that are unaware of the new ResizePolicy fields would set them
to nil. API server mutates such updates by copying non-nil values from old Pod to the current
Pod.

Prior to v1.31, with InPlacePodVerticalScaling disabled, the kubelet interprets mutation to Pod
Resources as a Container definition change and will restart the container with the new Resources.
This could lead to Node resource over-subscription. In v1.31, the kubelet no longer considers
resource changes a change in the pod definition and doesn't restart the container. In this case, the
change to the new resource value happens if the container is restart for any other reason, making
the change non-deterministic and not reflected in the API. Both of these cases are undesirable, so
the API server should reject a resize request if the Kubelet does not support it
(InPlacePodVerticalScaling enabled).

To achieve this, the apiserver will check if the `.status.containerStatuses[*].resources` field is
non-nil on any running containers. This field is set by the kubelet on running containers if and
only if IPPVS is enabled, and can therefore be used as a proxy to determine if the Kubelet running
the pod has the feature enabled. The apiserver logic to determine if a resource mutation is allowed
then becomes:

```go
if !InPlacePodVerticalScaling {
  return false
}
for _, c := range pod.Status.ContainerStatuses {
  if c.State.Running != nil {
    return c.Resources != nil
  }
}
// No running containers
return true
```

Note that even if the container does not specify any resources requests, the status
Resources is still set to the non-nill empty value `{}`.

If a pod has not yet been scheduled, the resize is allowed, and the new values are used when
scheduling & starting the pod.

If a pod has been scheduled but does not have any running containers, there is no signal indicating
whether the assigned node supports resize, so we default to allowing resize. If the node does not
have resize enabled in this case, then a resized container will be started with the new resource
value. It is possible that the node could end up over-provisioned in this case.

It is also possible for a race condition to occur: resize on a non-running container is allowed, but
the Kubelet simultaneously starts the container. The resulting behavior would depend on the version:
prior to v1.31, the container is restarted with the new values. After v1.31, the container continues
running with the old resource values. Since this race condition only exists during enablement skew,
we choose to accept it as a known-issue.

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
    - Feature gate name: `InPlacePodVerticalScaling`
    - Components depending on the feature gate: kubelet, kube-apiserver, kube-scheduler

* **Does enabling the feature change any default behavior?**

  - Kubelet sets several pod status fields: `AllocatedResources`, `Resources`

* **Can the feature be disabled once it has been enabled (i.e. can we roll back
  the enablement)?** Yes

  - The feature should not be disabled on a running node (create a new node instead).

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

  - Failure scenarios are already covered by the version skew strategy.

* **What specific metrics should inform a rollback?**

  - Scheduler indicators:
    - `scheduler_pending_pods`
    - `scheduler_pod_scheduling_attempts`
    - `scheduler_pod_scheduling_duration_seconds`
    - `scheduler_unschedulable_pods`
  - Kubelet indicators:
    - `kubelet_pod_worker_duration_seconds`
    - `kubelet_runtime_operations_errors_total{operation_type=update_container}` 


* **Were upgrade and rollback tested? Was the upgrade->downgrade->upgrade path tested?**

  Testing plan:

  1. Create test pod
  2. Upgrade API server
  3. Attempt resize of test pod
     - Expected outcome: resize is rejected (see version skew section for details)
  4. Create upgraded node
  5. Create second test pod, scheduled to upgraded node
  6. Attempt resize of second test pod
    - Expected outcome: resize successful
  7. Delete upgraded node
  8. Restart API server with feature disabled
    - Ensure original test pod is still running
  9. Attempt resize of original test pod
    - Expected outcome: request rejected by apiserver
  10. Restart API server with feature enabled
    - Verify original test pod is still running

* **Is the rollout accompanied by any deprecations and/or removals of features, APIs,
fields of API types, flags, etc.?**

  No.

### Monitoring Requirements

_This section must be completed when targeting beta graduation to a release._

* **How can an operator determine if the feature is in use by workloads?**

  Metric: `kubelet_container_resize_requests_total` (see [Instrumentation](#instrumentation))

* **What are the SLIs (Service Level Indicators) an operator can use to determine
the health of the service?**
  - [x] Metrics
    - Metric name: `kubelet_container_resize_requests_total`
      - Components exposing the metric: kubelet
    - Metric name: `runtime_operations_duration_seconds{operation_type=container_update}`
      - Components exposing the metric: kubelet
    - Metric name: `runtime_operations_errors_total{operation_type=container_update}`
      - Components exposing the metric: kubelet

* **What are the reasonable SLOs (Service Level Objectives) for the above SLIs?**

  - Using `kubelet_container_resize_requests_total`, `completed + infeasible + canceled` request count
  should approach `proposed` request count in steady state.
  - Resource update operations should complete quickly (`runtime_operations_duration_seconds{operation_type=container_update} < X` for 99% of requests)
  - Resource update error rate should be low (`runtime_operations_errors_total{operation_type=container_update}/runtime_operations_total{operation_type=container_update}`)

* **Are there any missing metrics that would be useful to have to improve observability
of this feature?**

  - Kubelet admission rejections: https://github.com/kubernetes/kubernetes/issues/125375
  - Resize operate duration (time from the Kubelet seeing the request to actuating the changes): this would require persisting more state about when the resize was first observed.

### Dependencies

_This section must be completed when targeting beta graduation to a release._

* **Does this feature depend on any specific services running in the cluster?**

  Compatible container runtime (see [CRI changes](#cri-changes)).

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

* **Can enabling / using this feature result in resource exhaustion of some node resources (PIDs, sockets, inodes, etc.)?** No

### Troubleshooting

The Troubleshooting section currently serves the `Playbook` role. We may consider
splitting it into a dedicated `Playbook` document (potentially with some monitoring
details). For now, we leave it here.

_This section must be completed when targeting beta graduation to a release._

* **How does this feature react if the API server and/or etcd is unavailable?**

  - If the API is unavailable prior to the resize request being made, the request wil not go through.
  - If the API is unavailable before the Kubelet observes the resize, the request will remain pending until the Kubelet sees it.
  - If the API is unavailable after the Kubelet observes the resize, then the pod status may not
    accurately reflect the running pod state. The Kubelet tracks the resource state internally.

* **What are other known failure modes?**

  - TBD

* **What steps should be taken if SLOs are not being met to determine the problem?**

  - Investigate Kubelet and/or container runtime logs.

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
- 2021-02-05 - Final consensus on allocatedResources[] and resize[]
- 2022-05-01 - KEP 2273-kubelet-container-resources-cri-api-changes merged with this KEP
- 2023-04-08 - Catch up KEP details to what is actually implemented

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
