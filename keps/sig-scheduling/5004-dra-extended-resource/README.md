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
  - [Resource Claim API](#resource-claim-api)
  - [Pod API](#pod-api)
  - [Scheduling for Extended Resource backed by DRA](#scheduling-for-extended-resource-backed-by-dra)
    - [EventsToRegister](#eventstoregister)
    - [PreFilter](#prefilter)
    - [Filter](#filter)
    - [Score](#score)
    - [Reserve](#reserve)
    - [Unreserve](#unreserve)
    - [Prebind](#prebind)
  - [Actuation for Extended Resource backed by DRA](#actuation-for-extended-resource-backed-by-dra)
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
  possibly one node a time.

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

* Introduce the ability to advertise DRA resources as extended resources, and
  for the scheduler to consider them for allocation.

* Enable application operators to use the existing extended resource request in
  pod spec to request for DRA resources.

* Extended resource support is not added just for easing the transition to DRA
  for the short term. Its ease of use is one big advantage to keep it remaining
  useful for the long term.

* Device plugin API must not change. The existing device plugin drivers must
  continue working without change.

* DRA driver API must not change. Core Kubernetes (kube-scheduler, kubelet) is
  preferred over DRA driver for any change needed to support the feature.

### Non-Goals

* Minimize kubelet or kube-scheduler changes. The feature requires necessary
  changes in both scheduling and actuation.

* Keep advertising `node.status.Capacity` for extended resources backed by DRA.
  It is used for extended resources backed by device plugin only.

## Proposal

The basic idea is the following:

1. Introduce `extended resource backed by DRA`. It is like the current extended
   resource backed by device plugin, in that, it has a string name, and a
   discrete countable quantity. Its capacity is provided through DRA
   `ResourceSlice`, its consumption is specified through pod's extended
   resource request.
1. Introduce a field `ExtendedResourceName` to `DeviceClass` to allow cluster
   administrators to advertise certain class of devices as extended resource.
1. Introduce a special `ResourceClaim` object to keep track of device allocations
   for all extended resource requests backed by DRA for a pod. kube-scheduler
   uses DRA scheduling algorithm to fit pod's extended resource request to a
   node that advertises the extended resource in DRA `ResorceSlice` or extended
   resources backed by device plugin. When using DRA devices, it creates a
   special `ResourceClaim` for the pod with the allocation result recording
   which devices were picked. More details on this special `ResourceClaim`
   follow below.  When using extended resources advertised for a node by device
   plugin, the existing resource tracking reserves them.
1. kubelet asks DRA driver to prepare devices in the special `ResourceClaim`,
   and pass the devices to containers in a pod with the extended resource requests.

Some quick clarifications around the basic concepts: extended resource backed by
device plugin, extended resource backed by DRA, and dynamic resource.

* extended resource backed by device plugin uses pod's
  spec.containers[].resources.requests to request for resources, it consumes the capacity
  from node's status.capacity. It is of type (string, int64)
* dynamic resource uses `ResourceClaim` to request for resources, and
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

* One node in cluster has a extended resource backed by DRA, and another node in the
cluster has the same named extended resource backend by device plugin.

* One node in a cluster cannot have both extended resource backed by DRA, and same
  named extended resource backed by device plugin at the same time. This implies
  that either the resource is advertised through `ResourceSlice`,
  or `Node`'s status.capacity.

## Design Details

### Device Class API
The extended resource name to DRA device mapping can be specified at
`DeviceClassSpec`. The same extended resource name can be given to different
device classes, and one device class can have at most one extended resource name.

Cluster administrator is soly responsible for creating device classes, and the
mapping between the class of devices and the extended resource name.
`DeviceClass` is cluster scoped, application developers and operators cannot change it.

The mapping of DRA devices and extended resources is stored in k8s data store
(e.g. etcd). It is created after cluster creation, before deployment of the
application that uses the devices.

```go
// DeviceClassSpec is used in a [DeviceClass] to define what can be allocated
// and how to configure it.
type DeviceClassSpec struct {
	// Each selector must be satisfied by a device which is claimed via this class.
	//
	// +optional
	// +listType=atomic
	Selectors []DeviceSelector `json:"selectors,omitempty" protobuf:"bytes,1,opt,name=selectors"`

	// Config defines configuration parameters that apply to each device that is claimed via this class.
	// Some classses may potentially be satisfied by multiple drivers, so each instance of a vendor
	// configuration applies to exactly one driver.
	//
	// They are passed to the driver, but are not considered while allocating the claim.
	//
	// +optional
	// +listType=atomic
	Config []DeviceClassConfiguration `json:"config,omitempty" protobuf:"bytes,2,opt,name=config"`

	// ExtendedResourceName is the extended resource name the device class is advertised as.
	// All devices matched by the device class can be used to satisfy the extended resource requests in pod's spec.
	//
	// +optional
	ExtendedResourceName *string `json:"extendedResourceName,omitempty" protobuf:"bytes,4,opt,name=extendedResourceName"`
}
```

### Resource Claim API

A special resource claim object is created to keep track of device allocations for
extended resource. The resource claim object has following properties:

  * It is namespace scoped, like other resource claim objects.
  * It is owned by a pod, like other resource claim objects.
  * It has `spec` of devices.requests, with each request name being a hash of the
    container name and the extended resource backed by DRA name inside the container.
  * Its `status.allocation.devices` and `status.allocation.reservedFor` are
    used.
  * It does not have annotation `resource.kubernetes.io/pod-claim-name:` as
    it is created for the extended resource request in a pod spec, not for a
    claim in the pod spec.
  * At most one such claim object is created per pod. For example, if the pod
    requests for foo1.domain/bar and foo2.domain/bar, the allocation of devices
    for each are recorded in DeviceResourceRequestAllocationResult, and
    one claim object with two `Results` is created for the pod.

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
	Name string `json:"name" protobuf:"bytes,1,name=name"`
}
```

As shown above, device request must have a name, and it must be a DNS label.
kube-scheduler creates one `DeviceRequest` per extended resource backed by DRA
per container in the pod. The name of the `DeviceRequest` is determined by the
container name, and the extended resource backed by DRA name. It is in the
form of "c%d-e%d", where the first %d is the index of the container in the pod.
The second %d is the index of the extended resource inside the container
resource requests. For example, if the first container in the pod has an
extended resource backed by DRA which is the 3rd such request in the container,
then the name of the `DeviceRequest` is "c0-e2".

### Pod API

A new field `extendedResourceClaimStatus` is added to Pod's status to track
the special resouceclaim object created for the extended resource requests
in the pod. This is needed for kublet to pass the devices allocated by driver
to the containers in the pod.

```go
// PodExtendedResourceClaimStatus is stored in the PodStatus for each extended
// resource requests backed by DRA. It stores the generated name for 
// the corresponding special ResourceClaim created by scheduler.
type PodExtendedResourceClaimStatus struct {
        // Names identifies the mapping of <container, extended resource backed by DRA> to  device request.
        // +patchMergeKey=requestName
        // +patchStrategy=merge,retainKeys
        // +listType=map
        // +listMapKey=requestName
        // +featureGate=DynamicResourceAllocation
        Names []ContainerExtendedResourceRequest `json:"names" patchStrategy:"merge,retainKeys" patchMergeKey:"requestName" protobuf:"bytes,1,rep,name=names"`
           
        // ResourceClaimName is the name of the ResourceClaim that was
        // generated for the Pod in the namespace of the Pod.
        ResourceClaimName string `json:"resourceClaimName" protobuf:"bytes,2,name=resourceClaimName"`
}

type ContainerExtendedResourceRequest struct {
        // ContainerName is the unique container name within the pod.
        ContainerName string `json:"containerName" protobuf:"bytes,1,name=containerName"`
        // ExtendedResourceName is the extended resource name backed by DRA inside
        // the container's requests.
        ExtendedResourceName string `json:"extendedResourceName" protobuf:"bytes,2,name=extendedResourceName"`
        // RequestName is the device request name in the special resource claim
        // created for extended resource requests backed by DRA.
        RequestName string `json:"requestName" protobuf:"bytes,3,name=requestName"`
}

// PodStatus represents information about the status of a pod. Status may trail the actual
// state of a system, especially if the node that hosts the pod cannot contact the control
// plane.
type PodStatus struct {
        // Status of resource claims.
        // +patchMergeKey=name
        // +patchStrategy=merge,retainKeys
        // +listType=map 
        // +listMapKey=name
        // +featureGate=DynamicResourceAllocation
        // +optional
        ResourceClaimStatuses []PodResourceClaimStatus `json:"resourceClaimStatuses,omitempty" patchStrategy:"merge,retainKeys" patchMergeKey:"name" protobuf:"bytes,15,rep,name=resourceClaimStatuses"`
        // Status of extended resource claim backed by DRA.
        // +featureGate=DynamicResourceAllocation
        // +optional
        ExtendedResourceClaimStatus *PodExtendedResourceClaimStatus `json:"extendedResourceClaimStatus,omitempty" protobuf:"bytes,17,opt,name=extendedResourceClaimStatus"`
}
```

For example, if a pod has requested for foo.domain/bar, and it is
scheduled to run on a node that has advertised foo.domain/bar in `ResourceSlice`,
then the pod's status is like below:

```yaml
status:
   extendedResourceClaimStatus:
   - names:
     - container-name
     - foo.domain/bar
     - c0-e2
   resourceClaimName: ccc-gpu-57999b9c4c-vpq68-gpu-8s27z
```
where `deviceRequest` name is "c0-e2", and container-name is the first container
in the pod, foo.domain/bar is the 3rd extended resource in the container's requests.

Note the validations for extendedResourceClaimStatus are different from the
validations for resourceClaimStatuses.

1. resourceClaimStatuses requires `name` must be DNS label,
   extendedResourceClaimStatus's names' `containerName` and `RequestName` must
   be a DNS label, while the `extendedResourceName` is not a DNS label.
1. resourceClaimStatuses requires `name` must be one of the claim's name in the
   pod spec. extendedResourceClaimStatus requires `containerName` must be one
   of the container name in the pod spec, and `extendedReourceName` must be one
   of the extended resource name in that container.

### Scheduling for Extended Resource backed by DRA

A new field `DynamicResources` is added to
[`Resource`](https://github.com/kubernetes/kubernetes/blob/c81431de59a3bf516489317433a165b050322339/pkg/scheduler/framework/types.go#L798),
it works similar to ScalarResources. It is used to keep track of the extended
resources backed by DRA on a node, i.e. those that are advertised by `ResourceSlice`.

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
	// DynamicResources: keep track of extended resources backed by DRA to device classes
  // The map's key is the extended resource name that has at least one device
  // class advertises it.
	DynamicResources map[v1.ResourceName][]string
}
```

type `NodeInfo` is used by scheduler to keep track of the information for each
node in memory. Its `Allocatable` field is used to keep track of the allocatable
resources in memory. At the beginning of each scheduling cycle, scheduler takes
a snapshot of all the nodes in the cluster, and updates their corresponding
`NodeInfo`.

For the scheduler with DRA enabled, right after taking the node snapshot, the
scheduler also takes a snapshot of `DeviceClass`, and updates
`NodeInfo.DynamicResources` if there is extended resource backed by DRA.

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

The existing dynamicresurce plugin has registered all the events needed or
extended reource backed by DRA, hence no change is needed.

#### PreFilter
It checks if the pod has any container requests for extended resources backed by DRA.
If not, and no claims in the pod, then the plugin can return early, as there
is nothing to do.

If the pod still needs to be considered by the plugin, then it creates a
special resource claim that has empty spec. The exact spec needs to be decided
during Filter phase, as some node may have device plugin provide the capacity
for the extended resource, some other node may have DRA provide the capacity.
The requsts in the special resource claim need to vary for each node.

#### Filter
If a pod has extended resource backed by DRA, and the node does *not* have
device plugin to provide the capacity for the resource, then the
dynamicresource plugin needs to try allocate the resource by filling in the
special claim's `Spec.Devices.Requests` field.

One `request` is created per container, and per extended resource backed by DRA
in the container. The `DeviceClass` in the request is randomly picked if there
are multiple device classes advertising the extended resource. The `Name` of the
request is determined by the container name and the extended resource name, in
case the name conflicts with an existing device request, the next choice is
deterministically picked until there is no conflict.

The allocator needs to be modified to allow for the special resource claim for
extended resource backed by DRA, which could vary by node. The `Allocate`
method takes the claim as a parameter, in adddition to node parameter. The
algorithm uses the passed in special claim whenever it processes the last claim
in the claims slice, which is an instantiation of the special claim template
created during the preFilter phase.

If there is an allocation for a node, the allocation, and the claim are
recorded in cyclestate.

#### Score
If two nodes can fit the pod, one node has installed device plugin, the other
has installed DRA, score the DRA node higher. This is not going to be
considered in ALPHA.

#### Reserve
Reserve the in-memory `ResourceClaim` and its allocation results in the assume
cache, a map of in-flight claims.

#### Unreserve
The plugin  removes the allocation for the special `ResourceClaim` for extended
resource backed by DRA. Because it cannot be scheduled after all. It does not need
to remove the Pod from the claim.status.reservedFor field, as the special claim
has not been created in API server yet.

#### Prebind
This is called in a separate goroutine. The plugin makes API call to create the
`ResourceClaim` and updates the pod's status `ExtendedResourceClaimStatus`. If
some API request fails now, PreBind fails and the pod must be retried.

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

Start of v1.33 development cycle (v1.33.0-alpha.x-xxx-xxxxxxxxxxxx):

- `k8s.io/dynamic-resource-allocation/cel`: ??.?%
- `k8s.io/dynamic-resource-allocation/structured`: ??.?%
- `k8s.io/kubernetes/pkg/controller/resourceclaim`: ??.?%
- `k8s.io/kubernetes/pkg/scheduler/framework/plugins/dynamicresources`: ??.?%

##### Integration tests

<!--
This question should be filled when targeting a release.
For Alpha, describe what tests will be added to ensure proper quality of the enhancement.

For Beta and GA, add links to added tests together with links to k8s-triage for those tests:
https://storage.googleapis.com/k8s-triage/index.html
-->

The existing [integration tests for kube-scheduler which measure
performance](https://github.com/kubernetes/kubernetes/tree/master/test/integration/scheduler_perf#readme)
will be extended to cover the overheaad of running the additional logic to
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
was developed as part of the overall DRA development effort. We will extend
this test driver to enable support for `ExtendedResourceName`s and add tests to
ensure they are handled by the scheduler as described in this KEP.

### Graduation Criteria

#### Alpha

- Feature implemented behind a feature flag
- Initial e2e tests completed and enabled

#### Beta

- Gather feedback
- Additional tests are in Testgrid and linked in KEP

#### GA

- 3 examples of vendors making use of the extensions proposed in this KEP
- Scalability tests that mirror real-world usage as determined by user feedback
- Allowing time for feedback

### Upgrade / Downgrade Strategy

The usual Kubernetes upgrade and downgrade strategy applies for in-tree
components. Vendors must take care that upgrades and downgrades work with the
drivers that they provide to customers.

### Version Skew Strategy

All of the API extensions proposed in this KEP is the optional
`ExtendedResourceName`. There is no risk for version skew downgrades
because these `ResourceSlice` and `Devices` will never have existed in
older clusters.

## Production Readiness Review Questionnaire

### Feature Enablement and Rollback

###### How can this feature be enabled / disabled in a live cluster?

- [x] Feature gate (also fill in values in `kep.yaml`)
  - Feature gate name: DRAExtendedResource
  - Components depending on the feature gate:
    - kube-apiserver
    - kube-scheduler

###### Does enabling the feature change any default behavior?

No

###### Can the feature be disabled once it has been enabled (i.e. can we roll back the enablement)?

Yes. Applications that were already deployed and are running will continue to
work. They will also continue to work when restarted because the CDI devices that
have been prepared for them won't change across the restart.

The DRA driver itself should also be able to survive a rollback, It
will just lose the ability to advertise devices as extended resources.

###### What happens if we reenable the feature if it was previously rolled back?

The scheduler may lose track of what devices it has allocated to what pods. Any
pods that had previously allocated devices with the feature enabled will need
to be deleted to ensure they are freed back to their corresponding driver  and
the accounting for them is updated in the scheduler.

###### Are there any tests for feature enablement/disablement?

Objects with the API additions introduced in the KEP are never written by
Kubernetes components themselves. They are written by 3rd-party drivers.
However, the scheduler does consume these objects and track information from
them in order to make scheduling decisions.

Unit tests will be written in the scheduler to verify that enabling /
disabling of the DRAExtendedResource feature gate is non-disruptive to the
scheduler.

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
Will be considered for beta.

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
Will be considered for beta.

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
Will be considered for beta.

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
Will be considered for beta.

###### Are there any missing metrics that would be useful to have to improve observability of this feature?

<!--
Describe the metrics themselves and the reasons why they weren't added (e.g., cost,
implementation difficulties, etc.).
-->
Will be considered for beta.

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
Will be considered for beta.

### Scalability

No. The API extensions in this KEP are limited to the existing `ResourceSlice`
object, with no additional requirements to consume this object by additional
components.

###### Will enabling / using this feature result in introducing new API types?

No. The this KEP proposes extensions to an existing type, but not a new type itself.

###### Will enabling / using this feature result in any new calls to the cloud provider?

No.

###### Will enabling / using this feature result in increasing size or count of the existing API objects?

Yes. With the extensions proposed in this KEP, individual
`ResourceSlices` have additional fields available to them, thus increasing
their overall signature.but it is ultimately up to how 3rd party vendors decide to use them.

###### Will enabling / using this feature result in increasing time taken by any operations covered by existing SLIs/SLOs?

Yes. The time to allocate a device to a claim (and thus schedule the first pod
that references that claim) will be affected.

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
Will be considered for beta.

###### What steps should be taken if SLOs are not being met to determine the problem?

Will be considered for beta.

## Implementation History

- Kubernetes 1.33: KEP accepted as "???".

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
