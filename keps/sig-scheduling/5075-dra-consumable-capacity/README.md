<!--
**Note:** When your KEP is complete, all of these comment blocks should be removed.

To get started with this template:

- [ ] **Fill out as much of the kep.yaml file as you can.**
  At minimum, you should fill in the "Title", "Authors", "Owning-sig",
  "Status", and date-related fields.
- [ ] **Merge early and iterate.**
  Avoid getting hung up on specific details and instead aim to get the goals of
  the KEP clarified and merged quickly. The best way to do this is to just
  start with the high-level sections and fill out details incrementally in
  subsequent PRs.

Just because a KEP is merged does not mean it is complete or approved. Any KEP
marked as `provisional` is a working document and subject to change. You can
denote sections that are under active debate as follows:

```
<<[UNRESOLVED optional short context or usernames ]>>
Stuff that is being argued.
<<[/UNRESOLVED]>>
```

When editing KEPS, aim for tightly-scoped, single-topic PRs to keep discussions
focused. If you disagree with what is already in a document, open a new PR
with suggested changes.

One KEP corresponds to one "feature" or "enhancement" for its whole lifecycle.
You do not need a new KEP to move from beta to GA, for example. If
new details emerge that belong in the KEP, edit the KEP. Once a feature has become
"implemented", major changes should get new KEPs.

**Note:** Any PRs to move a KEP to `implementable`, or significant changes once
it is marked `implementable`, must be approved by each of the KEP approvers.
If none of those approvers are still appropriate, then changes to that list
should be approved by the remaining approvers and/or the owning SIG (or
SIG Architecture for cross-cutting KEPs).
-->
# KEP-5075: DRA: Consumable Capacity

<!-- toc -->
- [Release Signoff Checklist](#release-signoff-checklist)
- [Summary](#summary)
- [Motivation](#motivation)
  - [Goals](#goals)
  - [Non-Goals](#non-goals)
- [Proposal](#proposal)
  - [User Stories (Optional)](#user-stories-optional)
    - [Story 1](#story-1)
    - [Story 2](#story-2)
  - [Notes/Constraints/Caveats (Optional)](#notesconstraintscaveats-optional)
  - [Risks and Mitigations](#risks-and-mitigations)
- [Design Details](#design-details)
  - [Test Plan](#test-plan)
      - [Prerequisite testing updates](#prerequisite-testing-updates)
      - [Unit tests](#unit-tests)
      - [Integration tests](#integration-tests)
      - [e2e tests](#e2e-tests)
  - [Graduation Criteria](#graduation-criteria)
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
- [Infrastructure Needed (Optional)](#infrastructure-needed-optional)
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

[kubernetes.io]: https://kubernetes.io/
[kubernetes/enhancements]: https://git.k8s.io/enhancements
[kubernetes/kubernetes]: https://git.k8s.io/kubernetes
[kubernetes/website]: https://git.k8s.io/website

## Summary

This KEP is an extended use case from the partitionable device [KEP-4815: DRA: Add support for partitionable devices](https://github.com/kubernetes/enhancements/tree/master/keps/sig-node/4815-dra-partitionable-devices).
The enhancement enables a shared device allocation with consumable capacity values.
A shared device can be allocated to more than one resource claim with per-device resource requests.

## Motivation

A motivating use case for supporting a shared network device which can be selected by more than one pods on demand (on claim). 
The original discussion is in [this PR's comment thread](https://github.com/kubernetes-sigs/cni-dra-driver/pull/1#discussion_r1889265214).
The limitation of current implementation has been addressed [here](https://github.com/kubernetes-sigs/cni-dra-driver/pull/1#discussion_r1890166449).
The virtual network device is created and configured once the CNI is called based on the information of the master network device.
The configured information specific to the generated device should be at the ResourceClaim, not the ResourceSlice. 
However, the device in the ResourceClaim is now present, even though the device is listed in the ResourceSlice, and there is no attribute to differentiate between the master device and the actual share of the device.

There are also similar use cases for requesting the device based on a specific root or device type in other contexts,
including [Peer Pods](https://github.com/confidential-containers/cloud-api-adaptor/blob/main/docs/architecture.md)
and [Composable Disaggregated Infrastructure (CDI)](https://www.wwt.com/article/an-introduction-to-composable-disaggregated-infrastructure).

This feature is also beneficial for the other sharable devices those are not with a scope of [KEP-4815](https://github.com/kubernetes/enhancements/issues/4815).
For instance, this feature will be allow reserving memory fraction of virtual GPU in [the AWS virtual GPU device plugin](https://github.com/awslabs/aws-virtual-gpu-device-plugin) via DRA.
The number of shares can be limited if needed.

Relations to related KEPs:
- KEP 4815: The partitioned devices can further be a sharable device. 
- KEP 5007: The allocated share can be provisioned at the pre-bind step.

### Goals

- Introduce an ability to allocating on-demand shared devices via DRA. 
  This should cover the use cases of macvlan or ipvlan in a DRA driver for CNI 
  and virtual accelerator devices with on-demand memory fraction.
- Enhance a capability of secondary networks to dynamically allocate secondary networks 
  based on present availabilities such as bandwidth.
- Enable capacity field to be consumable.

### Non-Goals

- Define driver-specific attributes and configs (such as CNI parameter config).
- Support network security policy.
- Support an aggregated resource consumption request. By default, the shared device can be allocated once for each pod's allocation. However, a user may want an aggrated ammount of resources which can come from a single or multiple shared device. This is related to [the comment about `distinctAttributes`](https://github.com/kubernetes/enhancements/pull/5104#discussion_r1943835445).
- Support refined capacity consumption such as predefined blocks and minimum and maximum amount.

## Proposal

<!--
This is where we get down to the specifics of what the proposal actually is.
This should have enough detail that reviewers can understand exactly what
you're proposing, but should not include things like API designs or
implementation. What is the desired outcome and how do we measure success?.
The "Design Details" section below is for the real
nitty-gritty.
-->

### User Stories (Optional)


Story|Driver|Claim|Context
---|---|----|---
1|shared device with capacity|request with resources|network
2|-|request with resources|gpu
3|-|request with selector|network
4|-|request with all resources|network
5|-|request with requests and limits|network
6|shared device without capacity|request with no resource|network
7|-|request with nil resource and AllowShared mode|any

#### Story 1
A user requests a secondary network based on their bandwidth demands without wanting to specify a device name 
nor present available bandwidth of each specific device. 
The DRA scheduler can dynamically select available devices based on their availability.

```yaml
kind: ResourceSlice
...
spec:
  driver: cni.dra.networking.x-k8s.io
  devices:
  - name: eth1
    basic:
      shared: "true"
      attributes:
        name:
          string: "eth1"
      capacity:
        bandwidth:
          value: 10Gi
- name: eth2
    basic:
      attributes:
        name:
          string: "eth1"
      capacity:
        bandwidth:
          value: 10Gi
---
kind: DeviceClass
metadata:
  name: vlan-cni.networking.x-k8s.io
```

With the below ResourceClaim, either `eth1` or `eth2` can be selected. 
The master field in the CNI config can be filled according to the selected device by the CNI DRA driver.

```yaml
kind: ResourceClaim
...
spec:
  devices:
    requests:
    - name: macvlan
      deviceClassName: vlan-cni.networking.x-k8s.io
      resources:
        requests:
          bandwidth: "1Gi"
    config:
    - requests:
      - macvlan
      opaque:
        driver: cni.dra.networking.x-k8s.io
        parameters: # CNIParameters with the GVK, interface name and CNI Config (in YAML format).
          apiVersion: cni.networking.x-k8s.io/v1alpha1
          kind: CNI
          ifName: "net1"
          config:
            cniVersion: 1.0.0
            name: net1
            plugins:
            - type: macvlan
              mode: bridge
              ipam:
                type: host-local
                ranges:
                - - subnet: 10.10.1.0/24
```

#### Story 2
A user reserves that fractions of memory for the virtual GPU.

```yaml
apiVersion: resource.k8s.io/v1beta1
kind: ResourceClaim
...
spec:
  devices:
    requests:
    - name: vgpu0
      deviceClassName: vgpu.nvidia.com
      resources:
        requests:
          memory: "10Gi"
```

#### Story 3
A user specifies a sharted device by some attributes such as name.

```yaml
apiVersion: resource.k8s.io/v1beta1
kind: ResourceClaim
...
spec:
  devices:
    requests:
    - name: net1
      deviceClassName: vlan-cni.networking.x-k8s.io
      selectors:
      - cel:
          expression: "device.attributes['vlan-cni.networking.x-k8s.io'].name == 'eth1'"
      resources:
        requests:
          bandwidth: "1Gi"
```

#### Story 4
A user reserves the shared device and blocks the others to get the same shared device.

```yaml
apiVersion: resource.k8s.io/v1beta1
kind: ResourceClaim
...
spec:
  devices:
    requests:
    - name: net1
      deviceClassName: vlan-cni.networking.x-k8s.io
      resources:
        all: "true"
```

#### Story 5
A user defines resource limits when the device driver supports a burstable use.
The CNI DRA driver can add a chained bandwidth CNI based on the bandwidth request.

```yaml
apiVersion: resource.k8s.io/v1beta1
kind: ResourceClaim
...
spec:
  devices:
    requests:
    - name: net1
      deviceClassName: vlan-cni.networking.x-k8s.io
      resources:
        requests:
          bandwidth: "1Gi"
        limits:
          bandwidth: "2Gi"
```

#### Story 6
A user requests one paired virtual device from a shared device with inifinite capacity. 

```yaml
kind: ResourceSlice
...
spec:
  driver: cni.dra.networking.x-k8s.io
  devices:
  - name: eth1
    shared: "true"
    basic:
      attributes:
        name:
          string: "eth1"
---
kind: DeviceClass
metadata:
  name: vlan-cni.networking.x-k8s.io
```

```yaml
apiVersion: resource.k8s.io/v1beta1
kind: ResourceClaim
...
spec:
  devices:
    requests:
    - name: net1
      deviceClassName: vlan-cni.networking.x-k8s.io
      resources: {}
  ...
  status:
    reserveFor:
    - name: example-pod
      ...
    allocation:
      devices:
        results:
        - request: net1
          device: eth1
          resources: {}
```

#### Story 7
A user requests a device regardless of whether the device is sharable or not.

```yaml
apiVersion: resource.k8s.io/v1beta1
kind: ResourceClaim
...
spec:
  devices:
    requests:
    - name: any
      deviceClassName: any.device
      allocationMode: AllowShared
```

Regardless of shared or non-sharable devices, 
`resources` field in ResourceClaim's status is set to nil,
and this device cannot be allocated to the other claim.

### Notes/Constraints/Caveats (Optional)

<!--
What are the caveats to the proposal?
What are some important details that didn't come across above?
Go in to as much detail as necessary here.
This might be a good place to talk about core concepts and how they relate.
-->

### Risks and Mitigations

<!--
What are the risks of this proposal, and how do we mitigate? Think broadly.
For example, consider both security and how this will impact the larger
Kubernetes ecosystem.

How will security be reviewed, and by whom?

How will UX be reviewed, and by whom?

Consider including folks who also work outside the SIG or subproject.
-->

## Design Details

This enhancement introduces a `shared` field within the `BasicDevice` of the ResourceSlice
to mark whether the device is a shared device.
If the device is marked as a shared device, all values in the capacity field are considered as consumable.
If there is no capacity defined, the device is considered as having an **infinite** sharable capacity.

Users can define specific per-device resource requests through
the newly added `ResourceRequest` field in the `DeviceRequest` of the `ResourceClaim`.
In a resource request, users may define `Requests` and `Limits` or either of them
to specify the claim's requirements. Similarly to the node resources, a request must be less than its limit.
If either value is omitted, the other value will be implicitly applied. 
`Requests` is used for verifying consumability of the shared device
while `Limits` is optionally used for supporting burstable consumption controlled by the device driver.

To request all available resources in the device capacity, users must set a flag `all` to "true".
Regardless of whether the shared device has infinite sharable capacity or not,
this flag will block the allocation of the shared device from the other claim.

If the `resources` field has been defined, only shared device can be allocated.
On the other hand, if the `resources` field is nil, only non-sharable devices are expected
except the allocation mode `AllowShared` has been set.
For allocationMode=AllowShared, both non-sharable devices and shared devices can be allocated.

A shared device can only be allocated once its consumability has been verified
and its attributes match the request's selectors and constraints.
The newly added `resources` field in the `DeviceRequestAllocationResult` will be set when the allocation is successful.
A device can be shared (allocated) to different pods. 
However, claims from the same pod cannot allocate shares from the same shared device.

### API enhancement
To enable this enhancement, the following API updates are proposed.
#### ResourceSliceSpec's BasicDevice

```go

// BasicDevice defines one device instance.
type BasicDevice struct {
...
   // Shared marks whether the device is shared.
   // The device with shared="true" can be allocated to more than one claim,
   // and all value in capacity is considered as consumable.
   // If there is no capacity defined,
   // the device is considered as having an infinity sharable capacity.
   // +optional
   // +default=false
   // +featureGate=ConsumableCapacity
   Shared bool `json:"shared" protobuf:"bytes,3,opt,name=shared"`
}
```

#### ResourceClaimSpec's DeviceRequest

```go
// ResourceRequest is a per-device resource request specification.
type ResourceRequest struct {
   // All marks requesting all resources from the shared device.
   // Regardless of the number of capacity defined in the shared device, 
   // this flag will block the allocation of the shared device from the other claims.
   // +optional
   // +default=false
   All bool `json:"all" protobuf:"bytes,1,name=all"`


   // Requirements define specific values of resource requirements of the device request.
   // If all="true", this field is ignored.
   // +optional
   Requirements ResourceRequirements `json:",inline"`
}


type ResourceRequirements struct {
   // Requests describe the amount of resources to be reserved from the device.
   // If Requests is omitted, it defaults to Limits if that is explicitly specified,
   // otherwise to an implementation-defined value. Requests cannot exceed Limits.
   // +optional
   Requests map[QualifiedName]resource.Quantity `json:"requests,omitempty" protobuf:"bytes,2,rep,name=requests"`


   // Limits define the maximum amount of per-device resources allowed.
   // If Limits is omitted, it defaults to Requests if that is explicitly specified.
   // This enables burstable usage when applicable.
   // +optional
   Limits map[QualifiedName]resource.Quantity `json:"limits" protobuf:"bytes,3,rep,name=limits"`
}

```

#### ResourceClaimStatus's DeviceRequestAllocationResult

```go
type DeviceRequestAllocationResult struct {
 ...
   // Resources indicates a per-device resource amount allocated by the claim request.
   // Only the consumable capacity can be partially allocated.
   // A summation of allocated request resources must be less than or equal each corresponding capacity.
   // nil if the device is unshareable.
   // +optional
   // +featureGate=ConsumableCapacity
   Resources *ResourceRequirements `json:"resources" protobuf:"bytes,6,opt,name=resources"`
}
```

### Scheduling enhancement
- define [share-related types and functions](https://github.com/sunya-ch/kubernetes/blob/kep-5075/staging/src/k8s.io/dynamic-resource-allocation/structured/share.go).

  ```go
  // AllocatedResources define a quantity set which is updatable.
  // This field is used for aggregating allocated resources,
  // and for calculating consumability.
  type AllocatedResources map[resourceapi.QualifiedName]*resource.Quantity

  // AllocatedResourceCollection collects a set of AllocatedResources
  // for each shared device.
  type AllocatedResourceCollection map[DeviceID]AllocatedResources
  ```

- shared device are handled separately from the other device.

  - define `ListAllAllocatedShares` function separately from `ListAllAllocatedDevices`
    ```go
    type ResourceClaimTracker interface {
      ...
  	  ListAllAllocatedDevices() (sets.Set[structured.DeviceID], error)
      // ListAllAllocatedShares lists all shared allocation from allocated ResourceClaims. The result is guaranteed to immediately include
      // any changes made via AssumeClaimAfterAPICall(), and SignalClaimPendingAllocation().
      ListAllAllocatedShares() (structured.AllocatedResourceCollection, error)
      ...
    }
    ```

  - define `foreachAllocatedResources` and add condition to not add shared device in `foreachAllocatedDevice`.

    ```go
    func foreachAllocatedDevice(claim *resourceapi.ResourceClaim, cb func(deviceID s)){
        ...
        if result.Resources != nil {
          // Is considered as shared allocation.
          continue
        }
    }

    // foreachAllocatedResources invokes the provided callback for each
    // device in the claim's resource allocation result which was allocated
    // exclusively for the claim.
    //
    // Devices allocated with admin access can be shared with other
    // claims and are skipped without invoking the callback.
    //
    // foreachAllocatedResources does nothing if the claim is not allocated.
    func foreachAllocatedResources(claim *resourceapi.ResourceClaim, cb func(allocatedSharedDevice structured.SharedDeviceAllocation)) {
      if claim.Status.Allocation == nil {
        return
      }
      for _, result := range claim.Status.Allocation.Devices.Results {
        // Kubernetes 1.31 did not set this, 1.32 always does.
        // Supporting 1.31 is not worth the additional code that
        // would have to be written (= looking up in request) because
        // it is extremely unlikely that there really is a result
        // that still exists in a cluster from 1.31 where this matters.
        if ptr.Deref(result.AdminAccess, false) {
          // Is not considered as allocated.
          continue
        }
        claimedResources := result.Resources
        if claimedResources == nil {
          // Is not considered as shared allocation.
          continue
        }
        deviceID := structured.MakeDeviceID(result.Driver, result.Pool, result.Device)
        sharedAllocation := structured.NewSharedDeviceAllocation(deviceID, *claimedResources)

        // None of the users of this helper need to abort iterating,
        // therefore it's not supported as it only would add overhead.
        cb(sharedAllocation)
      }
    }
    ```

  - use `foreachAllocatedShare` for `ListAllAllocatedShares`, `addDevices` and `removeDevices`.

  - when allocate, 
    - skip if the allocation is not `Shared`.
    - check shared device condition and define `isConsumable` function to check whether the device is allocatable.

    ```go

    const (
      DeviceAllocationModeExactCount  = DeviceAllocationMode("ExactCount")
      DeviceAllocationModeAll         = DeviceAllocationMode("All")
      DeviceAllocationModeAllowShared = DeviceAllocationMode("AllowShared")
    )

    func (alloc *allocator) allocateOne(r deviceIndices) (bool, error) {
      ...
        for _, slice := range pool.Slices {
          for deviceIndex := range slice.Spec.Devices {
            shared := alloc.isSharedDevice(slice, deviceIndex)
            requestSharedDevice := request.Resources != nil
            allowShared := request.AllocationMode == resourceapi.DeviceAllocationModeAllowShared
            // Skip a non-shared deivce if a request includes resources to consume.
            // Skip a shared device if an allocation mode is not AllowShared.
            if !allowShared && shared || requestSharedDevice && !shared {
              continue
            }
            ...
            if shared {
              // Next check consumable.
              consumable, err := alloc.isConsumable(requestIndices{claimIndex: r.claimIndex, requestIndex: r.requestIndex}, slice, deviceIndex)
              if err != nil {
                return false, err
              }
              if !consumable {
                alloc.logger.V(7).Info("Device not consumable", "device", deviceID)
                continue
              }
            }
            ...
          }
        }
      ...
    }

    // isSharedDevice checks whether the device is shared.
    // A device is considered as a shared device if the flag is set.
    func (alloc *allocator) isSharedDevice(slice *draapi.ResourceSlice, deviceIndex int) bool {
      basicDevice := slice.Spec.Devices[deviceIndex].Basic
      if basicDevice == nil {
        return false
      }
      return slice.Spec.Devices[deviceIndex].Basic.Shared
    }
    ```

- add request's `resources` in `internalDeviceResult` for updating ResourceClaim status correspondingly

  ```go
  type internalDeviceResult struct {
    request     string
    id          DeviceID
    slice       *draapi.ResourceSlice
    resources   *resourceapi.ResourceRequirements
    adminAccess *bool
  }
  ```

  ```go
  func (a *Allocator) Allocate(ctx context.Context, node *v1.Node) (finalResult []resourceapi.AllocationResult, finalErr error) {
        ...
      for i, internal := range internalResult.devices {
        allocationResult.Devices.Results[i] = resourceapi.DeviceRequestAllocationResult{
          ...
          Resources: internal.resources,
        }
      }
  }
  ```

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
-->

- `k8s.io/kubernetes/staging/src/k8s.io/dynamic-resource-allocation/structured`: `<date>` - 85.6%

##### Integration tests

<!--
Integration tests are contained in k8s.io/kubernetes/test/integration.
Integration tests allow control of the configuration parameters used to start the binaries under test.
This is different from e2e tests which do not allow configuration of parameters.
Doing this allows testing non-default options and multiple different and potentially conflicting command line options.
-->

<!--
This question should be filled when targeting a release.
For Alpha, describe what tests will be added to ensure proper quality of the enhancement.

For Beta and GA, add links to added tests together with links to k8s-triage for those tests:
https://storage.googleapis.com/k8s-triage/index.html
-->

- <test>: <link to test coverage>

##### e2e tests

<!--
This question should be filled when targeting a release.
For Alpha, describe what tests will be added to ensure proper quality of the enhancement.

For Beta and GA, add links to added tests together with links to k8s-triage for those tests:
https://storage.googleapis.com/k8s-triage/index.html

We expect no non-infra related flakes in the last month as a GA graduation criteria.
-->

- <test>: <link to test coverage>

### Graduation Criteria

<!--
**Note:** *Not required until targeted at a release.*

Define graduation milestones.

These may be defined in terms of API maturity, [feature gate] graduations, or as
something else. The KEP should keep this high-level with a focus on what
signals will be looked at to determine graduation.

Consider the following in developing the graduation criteria for this enhancement:
- [Maturity levels (`alpha`, `beta`, `stable`)][maturity-levels]
- [Feature gate][feature gate] lifecycle
- [Deprecation policy][deprecation-policy]

Clearly define what graduation means by either linking to the [API doc
definition](https://kubernetes.io/docs/concepts/overview/kubernetes-api/#api-versioning)
or by redefining what graduation means.

In general we try to use the same stages (alpha, beta, GA), regardless of how the
functionality is accessed.

[feature gate]: https://git.k8s.io/community/contributors/devel/sig-architecture/feature-gates.md
[maturity-levels]: https://git.k8s.io/community/contributors/devel/sig-architecture/api_changes.md#alpha-beta-and-stable-versions
[deprecation-policy]: https://kubernetes.io/docs/reference/using-api/deprecation-policy/

Below are some examples to consider, in addition to the aforementioned [maturity levels][maturity-levels].

#### Alpha

- Feature implemented behind a feature flag
- Initial e2e tests completed and enabled

#### Beta

- Gather feedback from developers and surveys
- Complete features A, B, C
- Additional tests are in Testgrid and linked in KEP

#### GA

- N examples of real-world usage
- N installs
- More rigorous forms of testing—e.g., downgrade tests and scalability tests
- Allowing time for feedback

**Note:** Generally we also wait at least two releases between beta and
GA/stable, because there's no opportunity for user feedback, or even bug reports,
in back-to-back releases.

**For non-optional features moving to GA, the graduation criteria must include
[conformance tests].**

[conformance tests]: https://git.k8s.io/community/contributors/devel/sig-architecture/conformance-tests.md

#### Deprecation

- Announce deprecation and support policy of the existing flag
- Two versions passed since introducing the functionality that deprecates the flag (to address version skew)
- Address feedback on usage/changed behavior, provided on GitHub issues
- Deprecate the flag
-->

### Upgrade / Downgrade Strategy

<!--
If applicable, how will the component be upgraded and downgraded? Make sure
this is in the test plan.

Consider the following in developing an upgrade/downgrade strategy for this
enhancement:
- What changes (in invocations, configurations, API use, etc.) is an existing
  cluster required to make on upgrade, in order to maintain previous behavior?
- What changes (in invocations, configurations, API use, etc.) is an existing
  cluster required to make on upgrade, in order to make use of the enhancement?
-->

### Version Skew Strategy

<!--
If applicable, how will the component handle version skew with other
components? What are the guarantees? Make sure this is in the test plan.

Consider the following in developing a version skew strategy for this
enhancement:
- Does this enhancement involve coordinating behavior in the control plane and nodes?
- How does an n-3 kubelet or kube-proxy without this feature available behave when this feature is used?
- How does an n-1 kube-controller-manager or kube-scheduler without this feature available behave when this feature is used?
- Will any other components on the node change? For example, changes to CSI,
  CRI or CNI may require updating that component before the kubelet.
-->

## Production Readiness Review Questionnaire

<!--

Production readiness reviews are intended to ensure that features merging into
Kubernetes are observable, scalable and supportable; can be safely operated in
production environments, and can be disabled or rolled back in the event they
cause increased failures in production. See more in the PRR KEP at
https://git.k8s.io/enhancements/keps/sig-architecture/1194-prod-readiness.

The production readiness review questionnaire must be completed and approved
for the KEP to move to `implementable` status and be included in the release.

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

<!--
This section must be completed when targeting alpha to a release.
-->

###### How can this feature be enabled / disabled in a live cluster?

<!--
Pick one of these and delete the rest.

Documentation is available on [feature gate lifecycle] and expectations, as
well as the [existing list] of feature gates.

[feature gate lifecycle]: https://git.k8s.io/community/contributors/devel/sig-architecture/feature-gates.md
[existing list]: https://kubernetes.io/docs/reference/command-line-tools-reference/feature-gates/
-->

- [ ] Feature gate (also fill in values in `kep.yaml`)
  - Feature gate name:
  - Components depending on the feature gate:
- [ ] Other
  - Describe the mechanism:
  - Will enabling / disabling the feature require downtime of the control
    plane?
  - Will enabling / disabling the feature require downtime or reprovisioning
    of a node?

###### Does enabling the feature change any default behavior?

<!--
Any change of default behavior may be surprising to users or break existing
automations, so be extremely careful here.
-->

###### Can the feature be disabled once it has been enabled (i.e. can we roll back the enablement)?

<!--
Describe the consequences on existing workloads (e.g., if this is a runtime
feature, can it break the existing applications?).

Feature gates are typically disabled by setting the flag to `false` and
restarting the component. No other changes should be necessary to disable the
feature.

NOTE: Also set `disable-supported` to `true` or `false` in `kep.yaml`.
-->

###### What happens if we reenable the feature if it was previously rolled back?

###### Are there any tests for feature enablement/disablement?

<!--
The e2e framework does not currently support enabling or disabling feature
gates. However, unit tests in each component dealing with managing data, created
with and without the feature, are necessary. At the very least, think about
conversion tests if API types are being modified.

Additionally, for features that are introducing a new API field, unit tests that
are exercising the `switch` of feature gate itself (what happens if I disable a
feature gate after having objects written with the new field) are also critical.
You can take a look at one potential example of such test in:
https://github.com/kubernetes/kubernetes/pull/97058/files#diff-7826f7adbc1996a05ab52e3f5f02429e94b68ce6bce0dc534d1be636154fded3R246-R282
-->

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

###### What specific metrics should inform a rollback?

<!--
What signals should users be paying attention to when the feature is young
that might indicate a serious problem?
-->

###### Were upgrade and rollback tested? Was the upgrade->downgrade->upgrade path tested?

<!--
Describe manual testing that was done and the outcomes.
Longer term, we may want to require automated upgrade/rollback tests, but we
are missing a bunch of machinery and tooling and can't do that now.
-->

###### Is the rollout accompanied by any deprecations and/or removals of features, APIs, fields of API types, flags, etc.?

<!--
Even if applying deprecation policies, they may still surprise some users.
-->

### Monitoring Requirements

<!--
This section must be completed when targeting beta to a release.

For GA, this section is required: approvers should be able to confirm the
previous answers based on experience in the field.
-->

###### How can an operator determine if the feature is in use by workloads?

<!--
Ideally, this should be a metric. Operations against the Kubernetes API (e.g.,
checking if there are objects with field X set) may be a last resort. Avoid
logs or events for this purpose.
-->

###### How can someone using this feature know that it is working for their instance?

<!--
For instance, if this is a pod-related feature, it should be possible to determine if the feature is functioning properly
for each individual pod.
Pick one more of these and delete the rest.
Please describe all items visible to end users below with sufficient detail so that they can verify correct enablement
and operation of this feature.
Recall that end users cannot usually observe component logs or access metrics.
-->

- [ ] Events
  - Event Reason: 
- [ ] API .status
  - Condition name: 
  - Other field: 
- [ ] Other (treat as last resort)
  - Details:

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

###### What are the SLIs (Service Level Indicators) an operator can use to determine the health of the service?

<!--
Pick one more of these and delete the rest.
-->

- [ ] Metrics
  - Metric name:
  - [Optional] Aggregation method:
  - Components exposing the metric:
- [ ] Other (treat as last resort)
  - Details:

###### Are there any missing metrics that would be useful to have to improve observability of this feature?

<!--
Describe the metrics themselves and the reasons why they weren't added (e.g., cost,
implementation difficulties, etc.).
-->

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

### Scalability

<!--
For alpha, this section is encouraged: reviewers should consider these questions
and attempt to answer them.

For beta, this section is required: reviewers must answer these questions.

For GA, this section is required: approvers should be able to confirm the
previous answers based on experience in the field.
-->

###### Will enabling / using this feature result in any new API calls?

<!--
Describe them, providing:
  - API call type (e.g. PATCH pods)
  - estimated throughput
  - originating component(s) (e.g. Kubelet, Feature-X-controller)
Focusing mostly on:
  - components listing and/or watching resources they didn't before
  - API calls that may be triggered by changes of some Kubernetes resources
    (e.g. update of object X triggers new updates of object Y)
  - periodic API calls to reconcile state (e.g. periodic fetching state,
    heartbeats, leader election, etc.)
-->

###### Will enabling / using this feature result in introducing new API types?

<!--
Describe them, providing:
  - API type
  - Supported number of objects per cluster
  - Supported number of objects per namespace (for namespace-scoped objects)
-->

###### Will enabling / using this feature result in any new calls to the cloud provider?

<!--
Describe them, providing:
  - Which API(s):
  - Estimated increase:
-->

###### Will enabling / using this feature result in increasing size or count of the existing API objects?

<!--
Describe them, providing:
  - API type(s):
  - Estimated increase in size: (e.g., new annotation of size 32B)
  - Estimated amount of new objects: (e.g., new Object X for every existing Pod)
-->

###### Will enabling / using this feature result in increasing time taken by any operations covered by existing SLIs/SLOs?

<!--
Look at the [existing SLIs/SLOs].

Think about adding additional work or introducing new steps in between
(e.g. need to do X to start a container), etc. Please describe the details.

[existing SLIs/SLOs]: https://git.k8s.io/community/sig-scalability/slos/slos.md#kubernetes-slisslos
-->

###### Will enabling / using this feature result in non-negligible increase of resource usage (CPU, RAM, disk, IO, ...) in any components?

<!--
Things to keep in mind include: additional in-memory state, additional
non-trivial computations, excessive access to disks (including increased log
volume), significant amount of data sent and/or received over network, etc.
This through this both in small and large cases, again with respect to the
[supported limits].

[supported limits]: https://git.k8s.io/community//sig-scalability/configs-and-limits/thresholds.md
-->

###### Can enabling / using this feature result in resource exhaustion of some node resources (PIDs, sockets, inodes, etc.)?

<!--
Focus not just on happy cases, but primarily on more pathological cases
(e.g. probes taking a minute instead of milliseconds, failed pods consuming resources, etc.).
If any of the resources can be exhausted, how this is mitigated with the existing limits
(e.g. pods per node) or new limits added by this KEP?

Are there any tests that were run/should be run to understand performance characteristics better
and validate the declared limits?
-->

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

###### What steps should be taken if SLOs are not being met to determine the problem?

## Implementation History

<!--
Major milestones in the lifecycle of a KEP should be tracked in this section.
Major milestones might include:
- the `Summary` and `Motivation` sections being merged, signaling SIG acceptance
- the `Proposal` section being merged, signaling agreement on a proposed design
- the date implementation started
- the first Kubernetes release where an initial version of the KEP was available
- the version of Kubernetes where the KEP graduated to general availability
- when the KEP was retired or superseded
-->

## Drawbacks

<!--
Why should this KEP _not_ be implemented?
-->

## Alternatives

<!--
What other approaches did you consider, and why did you rule them out? These do
not need to be as detailed as the proposal, but should include enough
information to express the idea and why it was not acceptable.
-->

## Infrastructure Needed (Optional)

<!--
Use this section if you need things from the project/SIG. Examples include a
new subproject, repos requested, or GitHub details. Listing these here allows a
SIG to get the process for these resources started right away.
-->