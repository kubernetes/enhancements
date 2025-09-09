# [KEP-5004](https://github.com/kubernetes/enhancements/issues/5004): DRA: Handle extended resource requests via DRA Driver

<!-- toc -->
- [Release Signoff Checklist](#release-signoff-checklist)
- [Summary](#summary)
- [Motivation](#motivation)
  - [Goals](#goals)
  - [Non-Goals](#non-goals)
- [Proposal](#proposal)
- [Design Details](#design-details)
  - [Device Class API](#device-class-api)
    - [Implicit Extended Resource Name](#implicit-extended-resource-name)
  - [Resource Claim API](#resource-claim-api)
  - [Pod API](#pod-api)
  - [Resource Quota](#resource-quota)
  - [Scheduling for Extended Resource backed by DRA](#scheduling-for-extended-resource-backed-by-dra)
    - [EventsToRegister](#eventstoregister)
    - [PreFilter](#prefilter)
    - [Filter](#filter)
    - [PostFilter](#postfilter)
    - [Reserve](#reserve)
    - [Unreserve](#unreserve)
    - [Prebind](#prebind)
    - [Failure handling](#failure-handling)
  - [Actuation for Extended Resource backed by DRA](#actuation-for-extended-resource-backed-by-dra)
  - [Cluster Autoscaler integration](#cluster-autoscaler-integration)
  - [Test Plan](#test-plan)
      - [Prerequisite testing updates](#prerequisite-testing-updates)
      - [Unit tests](#unit-tests)
      - [Integration tests](#integration-tests)
      - [e2e tests](#e2e-tests)
  - [Graduation Criteria](#graduation-criteria)
    - [Alpha](#alpha)
    - [Beta](#beta)
    - [GA](#ga)
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

Items marked with (R) are required *prior to targeting to a milestone / release*.

- [ ] (R) Enhancement issue in release milestone, which links to KEP dir in [kubernetes/enhancements] (not the initial KEP PR)
- [ ] (R) KEP approvers have approved the KEP status as `implementable`
- [ ] (R) Design details are appropriately documented
- [ ] (R) Test plan is in place, giving consideration to SIG Architecture and SIG Testing input (including test refactors)
  - [ ] e2e Tests for all Beta API Operations (endpoints)
  - [ ] (R) Ensure GA e2e tests meet requirements for [Conformance Tests](https://github.com/kubernetes/community/blob/master/contributors/devel/sig-architecture/conformance-tests.md)
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

Extended resource provides a simple, concise approach to describe resource
capacity, and resource consumption. In constrast, Dynamic Resource
Allocation (DRA) provides a more expressive, flexible approach, yet
more complicated, and harder to use.

This KEP provides a solution to enable cluster administrators to advertise the
dynamic resources (in `ResourceSlice`) as extended resource via `DeviceClass`.
and enables the application developers, and operators to continue using
extended resource to request for such resources.

This KEP provides dynamic allocation of resources to requests made through
either extended resource, or DRA resource claim.

## Motivation

There are three major motivations for the solution in this KEP.

* Enable existing applications to run without modification.

* Enable application developers and operators to transition to DRA gradually at
  their own pace.

* Enable cluster administrators to transition to DRA gradually at their own pace,
  possibly one node a time, which means supporting clusters where some nodes use
  device plugins and some nodes use DRA drivers for the same hardware at the same
  time.

For example, the following `Deployment` can be installed without modification on a
cluster with DRA `ResourceSlice`,`DeviceClass` and `Node` below. The 1 GPU out
of the 8 GPUs on the node is dynamically allocated to the pod, with the
remaining 7 GPUs left for allocation for future requests from either extended
resource, or DRA resource claim.

Note that another node in the same cluster has installed device plugin, which
may have advertised e.g. 'example.com/gpu: 2' in its `Node`'s Capacity. The same
`Deployment` can possibly be scheduled and run on that node too.

```yaml
apiVersion: apps/v1
kind: Deployment
metadata:
  name: demo
spec:
  replicas: 1
  selector:
    matchLabels:
      app: demo
  template:
    metadata:
      labels:
        app: demo
    spec:
      containers:
      - name: demo
        image: nvidia/cuda:8.0-runtime
        command: ["/bin/sh", "-c"]
        args: ["nvidia-smi && tail -f /dev/null"]
        resources:
          limits:
            example.com/gpu: 1
```

```yaml
apiVersion: resource.k8s.io/v1beta1
kind: DeviceClass
metadata:
  name: gpu.example.com
spec:
  selectors:
  - cel:
      expression: device.driver == 'gpu.example.com' && device.attributes['gpu.example.com'].type
        == 'gpu'
  extendedResourceName: example.com/gpu
```

```yaml
apiVersion: resource.k8s.io/v1beta1
kind: ResourceSlice
metadata:
  name: gke-drabeta-n1-standard-4-2xt4-346fe653-zrw2-gpu.coqj92d
spec:
  devices:
  - basic:
    name: gpu-0
  - basic:
    name: gpu-1
  - basic:
    name: gpu-2
  - basic:
    name: gpu-3
  - basic:
    name: gpu-4
  - basic:
    name: gpu-5
  - basic:
    name: gpu-6
  - basic:
    name: gpu-7
  driver: gpu.example.com
  nodeName: gke-drabeta-n1-standard-4-2xt4-346fe653-zrw2
```

```yaml
apiVersion: v1
kind: Node
metadata:
  name: gke-drabeta-n1-standard-4-2xt4-346fe653-zrw2
status:
  capacity:
    cpu: "4"
    ephemeral-storage: 101430960Ki
    hugepages-1Gi: "0"
    hugepages-2Mi: "0"
    memory: 15335536Ki
    pods: "110"
```

```yaml
apiVersion: v1
kind: Node
metadata:
  name: gke-drabeta-n1-standard-4-2xt4-346fe653-xyz8
status:
  capacity:
    cpu: "4"
    ephemeral-storage: 101430960Ki
    hugepages-1Gi: "0"
    hugepages-2Mi: "0"
    memory: 15335536Ki
    pods: "110"
    example.com/gpu: 2
```


With this motivating example in mind. We define the following goals and
non-goals of this KEP.

### Goals

* Enable cluster administrators to specify devices advertised by DRA drivers to satisfy
  extended resource requests.

* Enable application operators to use the existing extended resource request in
  pod spec to request DRA resources.

* Extended resource support is not added just for easing the transition to DRA
  for the short term. Its ease of use is one big advantage to keep it remaining
  useful for the long term.

* Device plugin API must not change. The existing device plugin drivers must
  continue working without change.

* DRA driver API must not change. Core Kubernetes (kube-scheduler, kubelet) is
  preferred over DRA driver for any change needed to support the feature.

* Keep advertising only extended resources backed by device plugin in `node.status.Capacity`
  for Alpha. It will be revisited for Beta, based on Alpha feedback.

### Non-Goals

* Minimize kubelet or kube-scheduler changes. The feature requires necessary
  changes in both scheduling and actuation.

* One node has both extended resource backed by DRA, and the same named
  extended resource backed by device plugin at the same time.

## Proposal

The basic idea is the following:

1. Introduce an `extended resource backed by DRA` concept. It is like the current extended
   resource backed by device plugin, in that, it has a string name, and a
   discrete countable quantity. Its capacity can be derived from DRA
   `ResourceSlice`, its consumption is specified through pod's extended
   resource request.
1. Introduce a field `ExtendedResourceName` to `DeviceClass` to allow cluster
   administrators to treat certain class of devices as an extended resource.
1. Introduce a special `ResourceClaim` object to keep track of device allocations. It
   is special only in the sense that it is created by the scheduler. No
   semantic changes are needed in the ResourceClaim API for it. kube-scheduler
   uses DRA scheduling algorithm to fit pod's extended resource request to a
   node that advertises the extended resource in DRA `ResorceSlice` or extended
   resources backed by device plugin. When using DRA devices, it creates a
   special `ResourceClaim` for the pod with the allocation result recording
   which devices were picked. More details on this special `ResourceClaim`
   follow below.  When using extended resources advertised for a node by device
   plugin, the existing resource tracking reserves them.
1. Introduce a field `ExtendedResourceClaimStatus` to pod's `Status`, such that:
    - the kubelet can find the special `ResourceClaim` while looking for claims to prepare
    - the kubelet can pass the devices to containers in the pod with the extended
   resource requests, based on the container/extended resource to device request mapping
   in the `ExtendedResourceClaimStatus`. Containers can be initContainers, regular
   containers, but cannot be ephemeral containers.
   
Some quick clarifications around the basic concepts: extended resource backed by
device plugin, extended resource backed by DRA, and dynamic resource.

* extended resource backed by device plugin uses pod's
  spec.containers[].resources.requests to request for resources, it consumes the capacity
  from node's status.capacity. It is of type (string, int64)
* dynamic resource uses `ResourceClaim` to request resources, and
  `ResourceSlice` to provide resource capacity. A pod asks for resources through
  resource claim requests in pod's spec.resources.claims. Dynamic resource type
  is described in resource slice, simply speaking, it is a list of devices, with
  each device being described as structured parameters.
* extended resource backend by DRA is a combination of the two above. It uses pods'
  spec.containers[].resources.requests to request for resources, and uses
  `ResourceSlice` to provide resource capacity. Hence, it is of type (string, int64)
   on the consumption side, and list of devices with a common
  `ExtendedResourceName` on the capacity side.

With these additions in place, the DRA devices can be consumed by extended resource
requests, or by DRA resouce claims. The scheduler has everything it needs to support
the dynamic allocation of devices to requests made through extended resource and
resource claims. No static partition of resources between extended resources and
resource claims is needed. The kubelet and DRA driver has everything they need
to admit a pod and pass the allocated devices to the containers in the pod to run.

Note the following cluster setup configuration and constraint:

* One node in a cluster can have an extended resource backed by DRA, and another node in the
same cluster can have the same named extended resource backed by Device Plugin.

* One node in a cluster cannot have both extended resource backed by DRA, and same
  named extended resource backed by device plugin at the same time. This implies
  that either the resource is advertised through `ResourceSlice`,
  or `Node`'s status.capacity.

## Design Details

### Device Class API
The extended resource name to DRA device mapping can be specified at
`DeviceClassSpec`. The same extended resource name should be given to at most one 
device class. If there are more than one device classes, the one created later is picked 
at scheduling time, if two are created at the same time, the name
lexicographically sorted first is picked.

Cluster administrator is soly responsible for creating device classes, and the
mapping between the class of devices and the extended resource name.
`DeviceClass` is cluster scoped, application developers and operators cannot change it.

The mapping of DRA devices and extended resources is stored in k8s data store
(e.g. etcd). An application using the extended resources can only request the
devices from DRA after the device class with the mapping is created. Before
that, the application can request the devices from device plugin only.

```go
// DeviceClassSpec is used in a DeviceClass to define what can be allocated
// and how to configure it.
type DeviceClassSpec struct {
	// ExtendedResourceName defines a mapping to the extended resource API.
	// All devices matched by the device class can be used to satisfy extended resource requests in pod's spec using this name.
	//
	// +optional
	ExtendedResourceName *string
}
```
#### Implicit Extended Resource Name

In addition to this optional extended resource name that is explicitly defined, every device class can be accessed
as an extended resource using the name `deviceclass.resource.kubernetes.io/<device-class-name>`. This implicit extended
resource name allows the simpler API to be used for DRA resource when no special DRA features beyond those
available via `DeviceClass` are needed.

There is a mismatch between what the API server allows to be a valid device class
name and extended resource name:

  * DeviceClass metadata.name must match IsDNS1123Subdomain, can be 253 characters long with dots
  * extended resource name must match IsQualifiedName, name part can be 63 characters, with dots

As a result, cluster admin must pick a DeviceClass name that conforms to the extended resource
name requirement, to be able to use it as implicit extended resource name. Failing that, cluster
admin can still set the extened resource name field explicitly in the DeviceClass.

### Resource Claim API

A special resource claim object is created to keep track of device allocations for
extended resource. The resource claim object has the following properties:

  * It is namespace scoped, like other resource claim objects.
  * It is owned by a pod, like other resource claim objects.
  * It has `Spec` of device.requests, with each request name being an encoding
    of the container name and the extended resource backed by DRA name inside
    the container.
  * Its `status.allocation.devices` and `status.allocation.reservedFor` are
    used.
  * It does not have annotation `resource.kubernetes.io/pod-claim-name:` as
    it is created for the extended resource request(s) in a pod spec, not for a
    claim in the pod spec.
  * It does have annotation `resource.kubernetes.io/extended-resource-claim: pod-name` as
    it is created, deleted, updated by the scheduler. It is used by scheduler
    to find the resource claim it has created, and ensure at most one such
    claim per pod.
  * At most one such claim object is created per pod. For example, if a pod
    requests for foo1.domain/bar and foo2.domain/bar, the allocation of devices
    for each are recorded in DeviceResourceRequestAllocationResult, and just 
    one claim object with allocation `Results` that lists all allocated devices
    is created for the pod.

The special resource claim object lifecycle is managed by the scheduler and
garbage collector.

  * It is *created* in a namespace when there is a pod with extended resource
  request, and the extended resource is advertised by `ResourceSlice` and
  scheduler has fit the pod to a node with the `ResourceSlice`.
  * It is *created* by the scheduler dynamic resource plugin during
    preBind phase. The in-memory one in the assumed cache is created earlier
    during Reserve phase.
  * It is *deleted*
    * either together with the owning pod's deletion.
    * or by the scheduler dynamic resource plugin during unReserve phase.
    * or by the scheduler dynamic resource plugin during postFilter phase.
  * It is *discovered* by the kubelet via `pod.Status.ExtendedResourceClaimStatus`
  * It is *read* by the kubelet DRA device driver to prepare the devices listed
    therein when preparing to run the pod.

```go
type DeviceRequest struct {
	// Name can be used to reference this request in a pod.spec.containers[].resources.claims
	// entry and in a constraint of the claim.
	//
	// Must be a DNS label.
	//
	// +required
	Name string
}
```

To enable the kubelet to map devices back to the containers which requested them,
the kube-scheduler creates one `DeviceRequest` per extended resource backed by DRA
per container in the pod. containers can be initContainers, regular containers,
but cannot be ephemeral containers. The name of the `DeviceRequest` has the form
"container-%d-request-%d", where the first %d is the index of the container in the pod.
The second %d is the index of the extended resource inside the container
resource requests. For example, if the first container in the pod has an
extended resource backed by DRA which is the 3rd such request in the container,
then the name of the `DeviceRequest` is "container-0-request-2".

Documenting this naming is merely informational, it is not part of the API.
The kubelet must not rely on it. Instead, the
`ContainerExtendedResourceRequest` field below specifies the mapping.

### Pod API

A new field `extendedResourceClaimStatus` is added to Pod's status to track
the special `ResourceClaim` object created for the extended resource requests
in the pod. This is needed for kubelet to pass the devices allocated by driver
to the containers in the pod. containers can be initContainers, regular containers,
but cannot be ephemeral containers. 

```go
// PodExtendedResourceClaimStatus is stored in the PodStatus for each extended
// resource requests backed by DRA. It stores the generated name for 
// the corresponding special ResourceClaim created by scheduler.
type PodExtendedResourceClaimStatus struct {
        // ResourceClaimName is the name of the ResourceClaim that was
        // generated for the Pod in the namespace of the Pod.
        ResourceClaimName string

        // RequestMapping identifies the mapping of <container, extended resource backed by DRA> to  device request.
        // +patchMergeKey=requestName
        // +patchStrategy=merge,retainKeys
        // +listType=atomic
        // +listMapKey=requestName
        // +featureGate=DynamicResourceAllocation
        RequestMapping []ContainerExtendedResourceRequest
}

type ContainerExtendedResourceRequest struct {
        // ContainerName is the unique container name within the pod.
        ContainerName string
        // ExtendedResourceName is the extended resource name backed by DRA inside
        // the container's requests.
        ExtendedResourceName string
        // RequestName is the device request name in the special resource claim
        // created for extended resource requests backed by DRA.
        RequestName string
}

type PodStatus struct {
    ...

    // Status of extended resource claim backed by DRA.
    // +featureGate=DynamicResourceAllocation
    // +optional
    ExtendedResourceClaimStatus *PodExtendedResourceClaimStatus
}
```

For example, if a pod has requested for foo.domain/bar, and it is
scheduled to run on a node where foo.domain/bar was mapped to devices in a DeviceClass,
then the pod's status is like below:

```yaml
status:
  extendedResourceClaimStatus:
    resourceClaimName: ccc-gpu-57999b9c4c-vpq68-gpu-8s27z
    requestMapping:
    - containerName: container-name
      extendedResourceName: foo.domain/bar
      requestName: container-0-request-2
```
where `deviceRequest` name is "container-0-request-2", and container-name is the first container
in the pod, foo.domain/bar is the 3rd extended resource in the container's requests.

Note the validations for extendedResourceClaimStatus are different from the
validations for resourceClaimStatuses.

1. resourceClaimStatuses requires `name` must be DNS label,
   extendedResourceClaimStatus's requestMapping's `containerName` and `RequestName` must
   be a DNS label, while the `extendedResourceName` is not a DNS label.
1. resourceClaimStatuses requires `name` must be one of the claim's name in the
   pod spec. extendedResourceClaimStatus requires `containerName` must be one
   of the container name in the pod spec, and `extendedResourceName` must be one
   of the extended resource name in that container.

### Resource Quota

Currently, there are two different applicable quotas, one is
device-class-name.deviceclass.resource.k8s.io/devices that limits the
resource claims in a namespace as described in [KEP](https://github.com/kubernetes/enhancements/tree/master/keps/sig-node/4381-dra-structured-parameters#resourcequota).
The other is the [extended resource quota](https://kubernetes.io/docs/concepts/policy/resource-quotas/#resource-quota-for-extended-resources).

As there is a one to one mapping between device class, and extended resource,
the two quota mechanisms above should keep track of the usages of the same
class of devices the same way.

But currently, the extended resource quota keeps track of the devices provided
from device plugin, and DRA resource slice. The resource claim quota currently
only keeps track of the devices provided from DRA resource slice. This must be
enhanced to have it keep track of the devices from device plugin too.

As a device can be requested by resource claim, or by extended resource, the
cluster admin MUST create two quotas with the same limit on one class of devices
to effectively quota the usage of that device class.

For example, a cluster admin plans to allow 10 example.com/gpu devices in a
given namespace, they MUST create the following:

```yaml
apiVersion: v1
kind: ResourceQuota
metadata:
  name: gpu
spec:
  hard:
    requests.example.com/gpu: 10
    gpu.example.com.deviceclass.resource.k8s.io/devices: 10
```

Provided that the device class gpu.example.com is mapped to the extended
resource example.com/gpu.
```yaml
apiVersion: resource.k8s.io/v1beta1
kind: DeviceClass
metadata:
  name: gpu.example.com
spec:
  extendedResourceName: example.com/gpu
```

Resource Quota controller reconciles away the differences if any between the
usage of the two quota, and ensures their usage are always kept the same. For
that, the controller needs to have the permission to list the device classes
in the cluster to establish the mapping between device class and extended
resource.

### Scheduling for Extended Resource backed by DRA

A new field `DynamicResources` is added to
[`Resource`](https://github.com/kubernetes/kubernetes/blob/c81431de59a3bf516489317433a165b050322339/pkg/scheduler/framework/types.go#L798),
it works similar to ScalarResources. It is used to keep track of the extended
resources backed by DRA, i.e. those that are advertised by `ResourceSlice`,
and mapped via `DeviceClass` extendedResourceName field.

```go
type Resource struct {
	MilliCPU         int64
	Memory           int64
	EphemeralStorage int64
	// We store allowedPodNumber (which is Node.Status.Allocatable.Pods().Value())
	// explicitly as int, to avoid conversions and improve performance.
	AllowedPodNumber int
	// ScalarResources
	ScalarResources map[v1.ResourceName]int64

	// NEW!
	// DynamicResources: keep track of extended resources backed by DRA to device class
	// The map's key is the extended resource name that has exactly one device
	// class advertises it.
	DynamicResources map[v1.ResourceName]string
}
```

type `NodeInfo` is used by scheduler to keep track of the information for each
node in memory. Its `Allocatable` field is used to keep track of the allocatable
resources in memory. At the beginning of each scheduling cycle, scheduler takes
a snapshot of all the nodes in the cluster, and updates their corresponding
`NodeInfo`.

For the scheduler with DRA enabled, right after taking the node snapshot, the
scheduler also takes a snapshot of `DeviceClass`, and updates
`NodeInfo.DynamicResources` if there is an extended resource backed by DRA.

For a node with extended resources from device plugin, its NodeInfo's
Allocatable.ScalarResources is updated with the k8s `Node`'s object.
For a node with extended resources backed by DRA, its NodeInfo's
Allocatable.DynamicResources is updated based on DRA `DeviceClass` objects.

The existing 'noderesources' plugin needs to be modified, such that a pod's
extended resource request is checked against a NodeInfo's ScalarResources if the
node uses device plugin, and checked against a NodeInfo's DynamicResources, if
the request is for extended resources backed by DRA, then 'noderesources' plugin
would pass and leave it to 'dynamicresource' plugin to check if it can be satisfied.

The existing 'dynamicresources' plugin needs to be modified to account for the
extended resource backed by DRA requests.

#### EventsToRegister
This registers all cluster events that might make an unschedulable pod schedulable,
like finishing the allocation of a claim, or resource slice updates.

The existing dynamicresource plugin has registered almost all the events needed for
extended resource backed by DRA, with one addition `framework.UpdateNodeAllocatable`
for node action.

#### PreFilter
It checks if the pod has any container requests for extended resources backed by DRA.
If not, and no claims in the pod, then the plugin can return early, as there
is nothing to do.

If the pod still needs to be considered by the plugin, then it checks if the
special resource claim for extended resources backed by DRA has been created
before by scheduler, by checking resource claim name having pod name in the
annotation `resource.kubernetes.io/extended-resource-claim: pod-name`.

If found, scheduler would reuse it. If not found, scheduler would create a
special resource claim that has empty spec. The exact spec needs to be decided
during Filter phase, as some node may have device plugin provide the capacity
for the extended resource, some other node may have DRA provide the capacity.
The requests in the special resource claim need to vary for each node.

#### Filter
If a pod has an extended resource backed by DRA, and the node does *not* have
device plugin to provide the capacity for the resource, then the
dynamicresource plugin needs to try to allocate the resource by filling in the
special claim's `Spec.Devices.Requests` field.

One `request` is created per container, and per extended resource backed by DRA
in the container. The `DeviceClass` in the request is the device class that has
the matching `ExtendedResourceName` field (one extended resource name can be in
at most one device class). The `Name` of the request is determined by the
container name and the extended resource name.

The allocator needs to be modified to allow for the special resource claim for
extended resource backed by DRA, which could vary by node. The `Allocate`
method takes the claim as a parameter, in adddition to node parameter. The
algorithm uses the passed in special claim whenever it processes the last claim
in the claims slice, which is an instantiation of the special claim template
created during the preFilter phase.

If there is an allocation for a node, the allocation, and the claim are
recorded in cyclestate.

#### PostFilter
If the special resource claim is not available, i.e., the claim cannot be bound
to the node, then scheduler would deallocate it, and delete it during
PostFilter phase.

#### Reserve
Reserve the in-memory `ResourceClaim` and its allocation results in the assume
cache, a map of in-flight claims.

#### Unreserve
The plugin deletes the special `ResourceClaim` for extended resource backed by DRA,
because it cannot be scheduled after all.

#### Prebind
This is called in a separate goroutine. The plugin makes API call to create the
`ResourceClaim` and updates the pod's status `ExtendedResourceClaimStatus`. If
some API request fails now, PreBind fails and the pod must be retried.

#### Failure handling
The special resourceclaim for extended resources backed by DRA may fail to be
created, updated in kuberentes API server. Below discusses the possible
failures in the write API calls added in this KEP.

During Prebind phase, if the special resourceclaim is new, i.e. it is not written
to API server before, then it is first created in API server. Then followed by
claim finalizer field update, claim status update if needed. These updates have local
retries in case there is a conflict. (Note these updates logic is not new, they
apply to other regular claims too). After that,
Pod.Status.ExtendedResourceClaimStatus is updated if needed. Both the claim create, and
claim finalizer update, and claim status, or pod status update could fail,
which will fail the Prebind phase, [framework.Error](https://github.com/kubernetes/kubernetes/blob/master/pkg/scheduler/framework/interface.go#L198)
code is returned. The scheduler framework will first call Unreserve phase to
clean up, then requeue the pod to activeQ/backoffQ soon.

During Unreserve phase, the special resourcelclaim's finalizer is first removed with
an update API call, then the claim is deleted, then
Pod.Status.ExtendedResourceClaimStatus is updated. If the update or delete fails, the
failure is logged, and continued. Unreserve needs to be idempotent, scheduler
framework will retry later if there is failure.

During Postfilter phase, if the special resourceclaim is picked to be deleted, then
the special resourcelclaim's finalizer is first removed with
an update API call, then the claim is deleted, then
Pod.Status.ExtendedResourceClaimStatus is updated. If the update or delete fails,
then [framework.Error](https://github.com/kubernetes/kubernetes/blob/master/pkg/scheduler/framework/interface.go#L198)
code is returned. The scheduler framework will requeue the pod to activeQ/backoffQ soon.

During Prefilter phase, if the special resource claim has already been created
before, it is validated, and reused if still valid. If not valid, then return
[framework.Unschedulable](https://github.com/kubernetes/kubernetes/blob/master/pkg/scheduler/framework/interface.go#L208),
then the invalid resourceclaim may be picked during PostFilter phase for deletion.
If not found, then a new in-memory resource claim template is created, which
will be instantiated at Filter phase, persisted at PreBind phase.

During Filter phase, if the special resource claim allocation's
node selector does not match, then return
[framework.Unschedulable](https://github.com/kubernetes/kubernetes/blob/master/pkg/scheduler/framework/interface.go#L208),
then the resourceclaim may be picked for deletion during PostFilter phase.

### Actuation for Extended Resource backed by DRA
When a pod with extended resources requests is picked up by the kubelet on the
node it is scheduled to run, the following are particularly important:

1. Kubelet tries to admit the pod, the pod's extended resources requests
   should not be checked against the `Node`'s allocatable, as the resources are
   in `ResourceSlice`, not in `Node`. In reality, the current predicate.go has
   already removed the missing extended resources from node info for
   cluster-level resources, hence there is no extra logic needed to admit the
   extended resources backed by DRA.

1. Kubelet (DRA manager) passes the special `ResoureClaim` to DRA driver to
   prepare the devices, in the same way as that for normal `ResourceClaim`.

1. Kubelet passes the device IDs through CDI to the containers with the
   extended resource requests. This is different from actuation of a pod with
   resource claim, as the pod does *not* have claim requests in containers or
   pods. Instead, the pod.status.extendedResourceClaimStatus has the mapping of
   container name and extended resource name to request in
   `claim.spec.devices.requests`, DRA manager uses this status information to
   pass the proper allocated device IDs to the proper container.

### Cluster Autoscaler integration

The new NodeInfo.Allocatable.DynamicResources field inside NodeInfo may need to
be correctly set in cluster autoscaler, based on its own internal cluster state,
which means there may be a need to expose a public method to set it.

The special resource claim created in PreFilter has to go through the
ResourceClaimTracker from SharedDRAManager so that cluster autoscaler can
reflect the claim in-memory. The special claim is currently reserved by calling
 SignalClaimPendingAllocation() in Reserve phase and persisted to API server in
 PreBind phase. There might be a need to expand ResourceClaimTracker to
 integrate with cluster autoscaler.

### Test Plan

<!--
[testing-guidelines]: https://git.k8s.io/community/contributors/devel/sig-testing/testing.md
-->

[x] I/we understand the owners of the involved components may require updates to
existing tests to make this code solid enough prior to committing the changes necessary
to implement this enhancement.

##### Prerequisite testing updates

##### Unit tests

<!--
Additionally, for Alpha try to enumerate the core package you will be touching
to implement this enhancement and provide the current unit coverage for those
in the form of:
- <package>: <date> - <current test coverage>
The data can be easily read from:
https://testgrid.k8s.io/sig-testing-canaries#ci-kubernetes-coverage-unit

This can inform certain test coverage improvements that we want to do before
extending the production code to implement this enhancement.
- `<package>`: `<date>` - `<test coverage>`
-->

Start of v1.34 development cycle (v1.33.0):

- `k8s.io/dynamic-resource-allocation/cel`: 88.2%
- `k8s.io/dynamic-resource-allocation/structured`: 90.5%
- `k8s.io/kubernetes/pkg/controller/resourceclaim`: 74.6%
- `k8s.io/kubernetes/pkg/scheduler/framework/plugins/dynamicresources`: 65.4%

##### Integration tests

<!--
This question should be filled when targeting a release.
For Alpha, describe what tests will be added to ensure proper quality of the enhancement.

For Beta and GA, add links to added tests together with links to k8s-triage for those tests:
https://storage.googleapis.com/k8s-triage/index.html
-->

The existing [integration tests for kube-scheduler which measure
performance](https://github.com/kubernetes/kubernetes/tree/master/test/integration/scheduler_perf#readme)
will be extended to cover the overhead of running the additional logic to
support the features in this KEP. These also serve as [correctness
tests](https://github.com/kubernetes/kubernetes/commit/cecebe8ea2feee856bc7a62f4c16711ee8a5f5d9)
as part of the normal Kubernetes "integration" jobs which cover [the dynamic
resource
controller](https://github.com/kubernetes/kubernetes/blob/294bde0079a0d56099cf8b8cf558e3ae7230de12/test/integration/scheduler_perf/util.go#L135-L139).

##### e2e tests

<!--
This question should be filled when targeting a release.
For Alpha, describe what tests will be added to ensure proper quality of the enhancement.

For Beta and GA, add links to added tests together with links to k8s-triage for those tests:
https://storage.googleapis.com/k8s-triage/index.html

We expect no non-infra related flakes in the last month as a GA graduation criteria.
-->

End-to-end testing depends on a working resource driver and a container runtime
with CDI support. A [test
driver](https://github.com/kubernetes/kubernetes/tree/master/test/e2e/dra/test-driver)
was developed as part of the overall DRA development effort. We will add tests to
ensure `ExtendedResourceName`s are handled by the scheduler as described in this KEP.

### Graduation Criteria

#### Alpha

- Feature implemented behind a feature flag
- Initial e2e tests completed and enabled

#### Beta

- Reevaluate where to create the special resource claim, in scheduler or some
  other controller, based on feedback from Alpha and the nomination concept.
- Gather feedback from developers and surveys
- 3 examples of vendors making use of the extensions proposed in this KEP
- Scalability tests that mirror real-world usage as determined by user feedback
- Additional tests are in Testgrid and linked in KEP
- All functionality completed
- All security enforcement completed
- All testing requirements completed
- All known pre-release issues and gaps resolved 


#### GA
- Allowing time for feedback
- All issues and gaps identified as feedback during beta are resolved

### Upgrade / Downgrade Strategy

The usual Kubernetes upgrade and downgrade strategy applies for in-tree
components. Vendors must take care that upgrades and downgrades work with the
drivers that they provide to customers.

### Version Skew Strategy

All of the API extensions proposed in this KEP is the optional
`ExtendedResourceName` in `DeviceClass`, and `ExtendedResourceClaimStatus` in
`Pod`. There is no risk for version skew downgrades
because these `DeviceClass` and `Pod` will never have existed in
older clusters.

## Production Readiness Review Questionnaire

### Feature Enablement and Rollback

###### How can this feature be enabled / disabled in a live cluster?

- [x] Feature gate (also fill in values in `kep.yaml`)
  - Feature gate name: DRAExtendedResource
  - Components depending on the feature gate:
    - kube-apiserver
    - kube-scheduler
    - kubelet

###### Does enabling the feature change any default behavior?

No

###### Can the feature be disabled once it has been enabled (i.e. can we roll back the enablement)?

Yes. Applications that were already deployed and are running will continue to
work. They will continue to work when restarted because the CDI devices that
have been prepared for them won't change across the restart.

The DRA driver itself should also be able to survive a rollback, as there is no
DRA driver change in this KEP.

###### What happens if we reenable the feature if it was previously rolled back?

The scheduler may lose track of what devices it has allocated to what pods. Any
pods that had previously allocated devices with the feature enabled will need
to be deleted to ensure they are freed back to their corresponding driver and
the accounting for them is updated in the scheduler.

###### Are there any tests for feature enablement/disablement?

Unit tests will be written in the scheduler and kubelet to verify that enabling /
disabling of the DRAExtendedResource feature gate is non-disruptive to the
scheduler and kubelet.

### Rollout, Upgrade and Rollback Planning

<!--
This section must be completed when targeting beta to a release.
-->

###### How can a rollout or rollback fail? Can it impact already running workloads?

<!--
Try to be as paranoid as possible - e.g., what if some components will restart
mid-rollout?

Be sure to consider highly-available clusters, where, for example,
feature flags will be enabled on some API servers and not others during the
rollout. Similarly, consider large clusters and how enablement/disablement
will rollout across nodes.
-->
Will be considered for beta.

###### What specific metrics should inform a rollback?

<!--
What signals should users be paying attention to when the feature is young
that might indicate a serious problem?
-->
Will be considered for beta.

###### Were upgrade and rollback tested? Was the upgrade->downgrade->upgrade path tested?

<!--
Describe manual testing that was done and the outcomes.
Longer term, we may want to require automated upgrade/rollback tests, but we
are missing a bunch of machinery and tooling and can't do that now.
-->
Will be considered for beta.

###### Is the rollout accompanied by any deprecations and/or removals of features, APIs, fields of API types, flags, etc.?

<!--
Even if applying deprecation policies, they may still surprise some users.
-->
Will be considered for beta.

### Monitoring Requirements

<!--
This section must be completed when targeting beta to a release.

For GA, this section is required: approvers should be able to confirm the
previous answers based on experience in the field.
-->
Will be considered for beta.

###### How can an operator determine if the feature is in use by workloads?

<!--
Ideally, this should be a metric. Operations against the Kubernetes API (e.g.,
checking if there are objects with field X set) may be a last resort. Avoid
logs or events for this purpose.
-->
`kube_pod_resource_limit` and `kube_pod_resource_request`
(label: `namespace`, `pod`, `node`, `scheduler`, `priority`, **`resource`**, `unit`)
can be used to determine if the feature is in use by workloads though it doesn't differentiate 
between extended resources backed by DRA or device plugin.

`resourceclaim_controller_resource_claims` (label: `admin_access`, `allocated`, `source`)
should be a good metric to determine if the resource claim is created by extended resource backed by DRA.

###### How can someone using this feature know that it is working for their instance?

<!--
For instance, if this is a pod-related feature, it should be possible to determine if the feature is functioning properly
for each individual pod.
Pick one more of these and delete the rest.
Please describe all items visible to end users below with sufficient detail so that they can verify correct enablement
and operation of this feature.
Recall that end users cannot usually observe component logs or access metrics.

- [ ] Events
  - Event Reason: 
- [ ] API .status
  - Condition name: 
  - Other field: 
- [ ] Other (treat as last resort)
  - Details:
-->
- [ x ] API .status
    - Other field: `.status.extendedResourceClaimStatus` will have a list of resource claims that are created for
      DRA extended resources.

###### What are the reasonable SLOs (Service Level Objectives) for the enhancement?

<!--
This is your opportunity to define what "normal" quality of service looks like
for a feature.

It's impossible to provide comprehensive guidance, but at the very
high level (needs more precise definitions) those may be things like:
  - per-day percentage of API calls finishing with 5XX errors <= 1%
  - 99% percentile over day of absolute value from (job creation time minus expected
    job creation time) for cron job <= 10%
  - 99.9% of /health requests per day finish with 200 code

These goals will help you determine what you need to measure (SLIs) in the next
question.
-->
Existing DRA and related SLOs continue to apply.
Pod scheduling duration with this feature should be as fast as existing DRA.

###### What are the SLIs (Service Level Indicators) an operator can use to determine the health of the service?

<!--
Pick one more of these and delete the rest.

- [ ] Metrics
  - Metric name:
  - [Optional] Aggregation method:
  - Components exposing the metric:
- [ ] Other (treat as last resort)
  - Details:
-->
These are the same as for the main DRA feature:

- [x] Metrics
    - Metric name: resourceclaim_controller_creates_total
    - Metric name: resourceclaim_controller_resource_claims
    - Metric name: workqueue with name="resource_claim"
    - Metric name: scheduler_pending_pods
    - Metric name: scheduler_plugin_execution_duration_seconds

###### Are there any missing metrics that would be useful to have to improve observability of this feature?

<!--
Describe the metrics themselves and the reasons why they weren't added (e.g., cost,
implementation difficulties, etc.).
-->
No

### Dependencies

<!--
This section must be completed when targeting beta to a release.
-->

###### Does this feature depend on any specific services running in the cluster?

<!--
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
-->
No.

### Scalability

###### Will enabling / using this feature result in any new API calls?

Yes. scheduler make new API calls to create, update, and delete the special resource claim for extended resource backed by DRA.

###### Will enabling / using this feature result in introducing new API types?

No. The this KEP proposes extensions to an existing type, but not a new type itself.

###### Will enabling / using this feature result in any new calls to the cloud provider?

No.

###### Will enabling / using this feature result in increasing size or count of the existing API objects?

Yes. With the extensions proposed in this KEP, individual
`DeviceClass`  and `Pod` have additional fields, thus increasing
their overall signature. In addition,  there is the special resource claim for
extended resource by DRA, there is at most one such claim per pod.

###### Will enabling / using this feature result in increasing time taken by any operations covered by existing SLIs/SLOs?

Yes. The time to allocate a device to a pod with extended resource request will be affected.

###### Will enabling / using this feature result in non-negligible increase of resource usage (CPU, RAM, disk, IO, ...) in any components?

No.

###### Can enabling / using this feature result in resource exhaustion of some node resources (PIDs, sockets, inodes, etc.)?

No.

### Troubleshooting

<!--
This section must be completed when targeting beta to a release.

For GA, this section is required: approvers should be able to confirm the
previous answers based on experience in the field.

The Troubleshooting section currently serves the `Playbook` role. We may consider
splitting it into a dedicated `Playbook` document (potentially with some monitoring
details). For now, we leave it here.
-->

###### How does this feature react if the API server and/or etcd is unavailable?

Will be considered for beta.

###### What are other known failure modes?

<!--
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
-->
  - [Pod pending due to extended resource backed by DRA requests no less than 128 devices]
    - Detection: inspect pod status 'Pending'
    - Mitigations: reduce the number of devices requested in one extended resource backed by DRA requests
    - Diagnostics: scheduler logs at level 5 show the reason for the scheduling failure.
    - Testing: Will be considered for beta.

###### What steps should be taken if SLOs are not being met to determine the problem?

Will be considered for beta.

## Implementation History

- Kubernetes 1.34: KEP accepted.

## Drawbacks

It adds complexity to the scheduler.

## Alternatives

Many different approaches were considered.

Specifically, the following two alternative proposals were considered:

**Option 1:** webhook rewrite extended resource requests in pod spec

This approach requires cluster administrator deploy a mutation webhook to the
cluster, and configure the webhook with rewrite rules that can rewrite the
extended resource requests and node selectors. This approach is not taken due to
the webhook's extra configuration, and maintenance overhead.

**Option 2:** client CLI tool to rewrite extended resource requests in pod spec

This approach requires application developers, operators to run the client CLI
tool to rewrite the application YAML with extended resources to DRA resource
claims. This adds extra overhead to the application deployment flow.
