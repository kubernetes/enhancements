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

The enhancement enables allocating sharable devices to more than one resource claims under consumable capacity of each per-device resource. 
This KEP introduces (1) device fields to distinguish between “shareable” and “unshareable” devices and “consumable” and “nonconsumable” capacity, 
(2) per-device resource requests in the resource claim, 
(3) consumability check in allocation mechanism with corresponding per-device resource field in the allocation result of the claim. 

## Motivation

A motivating use case is to allocate a shared network device in [CNI DRA driver](https://github.com/kubernetes-sigs/cni-dra-driver)
which can be selected by more than one pods on demand (on claim). 
The original discussion is in [this PR's comment thread](https://github.com/kubernetes-sigs/cni-dra-driver/pull/1#discussion_r1889265214). 
The limitation of current implementation has been addressed [here](https://github.com/kubernetes-sigs/cni-dra-driver/pull/1#discussion_r1890166449).
The virtual network device is created and configured once the CNI is called based on the information of the master network device. 
The configured information specific to the generated device cannot be listed in the ResourceSlice in advance. 

This feature is also beneficial for the other sharable devices those are not with a scope of [KEP-4815](https://github.com/kubernetes/enhancements/issues/4815).
For instance, this feature will be allow reserving memory fraction of virtual GPU in [the AWS virtual GPU device plugin](https://github.com/awslabs/aws-virtual-gpu-device-plugin).
In other words, the device capacity allocation is determined by the user's claim. 

Relations to related KEPs:
- KEP 4815: The partitioned devices can further be a sharable device. 
- KEP 5007: The allocated share can be provisioned at the pre-bind step.

### Goals

- Introduce an ability to allocating shared devices via DRA to more than one pods.
  This should cover the use cases of macvlan or ipvlan in a DRA driver for CNI 
  and virtual accelerator devices with on-demand memory fraction.
- Enhance a capability of secondary networks to dynamically allocate secondary networks 
  based on present capacities such as bandwidth.
- Enable capacity field to be consumable.

### Non-Goals

- Define driver-specific attributes and configs (such as CNI parameter config).
- Support network security policy.
- Support an aggregated resource consumption request. 
  By default, the shared device can be allocated once for each pod's allocation.
  However, a user may want an aggrated ammount of resources which can come from a single or multiple shared device. 
  This is related to [the comment about `distinctAttributes`](https://github.com/kubernetes/enhancements/pull/5104#discussion_r1943835445).

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

See [all concrete use cases](https://docs.google.com/document/d/1U0u2uErpYcf-RooPEws5oDMiJ9kT2uDoWNjcWrLH224/edit?tab=t.f3ylp1uxsq1c).

This KEP focuses on the request with selection of shared device (`device.shared == "true"`).

```yaml
selectors:
  - cel:
      expression: |-
        device.shared == "true"
```

Story|Consumable Capacity|Selector|Resource Request|AllocationMode|Context
---|---|---|---|---|---
1|no|no|no|Exact(1)|network
2|no|no|no|Exact(2)|network
3|no|yes|no|Exact(1)|network
4|yes|no|yes|Exact(1)|network
5|no|no|yes|Exact(1)|network
6|both|no|yes|Exact(1)|network

#### Story 1

A DRA driver advertises a shared device 
and a user simply requests a shared device.

```yaml
kind: ResourceSlice
...
spec:
  driver: simple-cni.dra.networking.x-k8s.io
  devices:
  - name: eth1
    basic:
      shared: "true"
      attributes:
        name:
          string: "eth1"
  - name: eth2
    basic:
      shared: "true"
      attributes:
        name:
          string: "eth2"
```

A common shared device class is defined below.

```yaml
kind: DeviceClass
metadata:
  name: simple-shared.networking.x-k8s.io
  selectors:
  - cel:
      expression: |-
        device.driver == "simple-cni.dra.networking.x-k8s.io" &&
        device.shared == "true"
```

Then, a user defines the following resource claim.

```yaml
kind: ResourceClaim
...
spec:
  devices:
    requests:
    - name: macvlan
      deviceClassName: simple-shared.networking.x-k8s.io
    config:
    - requests:
      - macvlan
      opaque:
        driver: simple-cni.dra.networking.x-k8s.io
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
> The device config is out of the KEP scope. From this point, the request config will be omitted for simplicity.

#### Story 2

With the same resource as story 1, a user requests two devices.

```yaml
kind: ResourceClaim
...
spec:
  devices:
    requests:
    - name: macvlan-1
      deviceClassName: simple-shared.networking.x-k8s.io
      allocationMode: ExactCount
      count: 1
    - name: macvlan-2
      deviceClassName: simple-shared.networking.x-k8s.io
      allocationMode: ExactCount
      count: 1
    constraints:
    - requests:
      - macvlan-1
      - macvlan-2
      distinctAttribute: name
```

#### Story 3
With the same resource as story 1, a user specifies a sharted device by some attributes such as name.

```yaml
kind: ResourceClaim
...
spec:
  devices:
    requests:
    - name: net1
      deviceClassName: simple-shared.networking.x-k8s.io
      selectors:
      - cel:
          expression: |-
            device.attributes["simple-cni.dra.networking.x-k8s.io"].name == "eth1"
```

#### Story 4

A DRA driver specifies a consumable capacity with a minimum consume condition,
and a user requests the consumable resource.
A scheduler selects devices according to the availability.

```yaml
kind: ResourceSlice
...
spec:
  driver: guaranteed-cni.dra.networking.x-k8s.io
  devices:
  - name: eth1
    basic:
      shared: "true"
      attributes:
        name:
          string: "eth1"
      capacity:
        bandwidth:
          consumable:
            range:
              minimum: "1Mi"
              step: "8"
          value: "10Gi"
```

A shared device class with guaranteed bandwidth is defined below.

```yaml
kind: DeviceClass
metadata:
  name: bandwidth-guaranteed-cni.networking.x-k8s.io
  selectors:
  - cel:
      expression: |-
        device.driver == "guaranteed-cni.dra.networking.x-k8s.io" &&
        device.shared == "true" &&
        device.capacities["guaranteed-cni.dra.networking.x-k8s.io"].bandwidth.consumable == "true"
```

Then, a user defines the following resource claim.

```yaml
kind: ResourceClaim
...
spec:
  devices:
    requests:
    - name: net1
      deviceClassName: bandwidth-guaranteed-cni.networking.x-k8s.io
      capacity:
        requests:
          bandwidth: "1Gi"
```

#### Story 5

A DRA driver specifies a non-consumable capacity, 
and a user specifies a minimum bandwidth requested on a shared device.

```yaml
kind: ResourceSlice
...
spec:
  driver: extended-cni.dra.networking.x-k8s.io
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
```

A shared device class is defined below.

```yaml
kind: DeviceClass
metadata:
  name: extended-cni.networking.x-k8s.io
  selectors:
  - cel:
      expression: |-
        device.driver == "extended-cni.dra.networking.x-k8s.io" &&
        device.shared == "true" &&
        device.capacities["extended-cni.dra.networking.x-k8s.io"].bandwidth.consumable != "true"
```

Then, a user defines the following resource claim.

```yaml
kind: ResourceClaim
...
spec:
  devices:
    requests:
    - name: net1
      deviceClassName: extended-cni.networking.x-k8s.io
      capacity:
        requests:
          memory: "1Gi"
```

#### Story 6

A DRA driver specifies both consumable count to limit number of claim to share and non-consumable bandwidth capacity, 
and a user specifies a minimum bandwidth requested on a shared device.

```yaml
kind: ResourceSlice
...
spec:
  driver: complex-cni.dra.networking.x-k8s.io
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
        count:
          consumable:
            set:
              default: "1"
              set: ["1"]
          value: 1000
```

A shared device class is defined below.

```yaml
kind: DeviceClass
metadata:
  name: complex-cni.networking.x-k8s.io
  selectors:
  - cel:
      expression: |-
        device.driver == "complex-cni.dra.networking.x-k8s.io" &&
        device.shared == "true" &&
        device.capacities["complex-cni.dra.networking.x-k8s.io"].bandwidth.consumable != "true"
```

Then, a user defines the following resource claim.

```yaml
kind: ResourceClaim
...
spec:
  devices:
    requests:
    - name: net1
      deviceClassName: extended-cni.networking.x-k8s.io
      capacity:
        requests:
          memory: "1Gi"
          count: 1
```

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

This enhancement introduces a `shared` field within the `BasicDevice` of the ResourceSlice to mark whether the device is a shared device. 
The shared device can be assigned to more than one claim if it satisfies the request. 
User needs to explicitly add the select condition device.shared to identify whether to select the shared device or not. 

The enhancement also adds a `consumable`  field to `DeviceCapacity`. 
This field is used to identify the capacity which has a limited quantity and to specify consumable conditions. 
If the consumable field is not empty. Either one of the consumable conditions must be defined with default consumed value. 

Users can define specific per-device resource requests through the newly added `CapacityRequest` field in the `DeviceRequest`. 
`Requests` of `CapacityRequest` is used for verifying minimum device capacity. 
If the capacity is consumable, the amount available for allocation is determined by subtracting the aggregated allocation results of current claims 
from the device's capacity as defined in the resource slice. 
The subtracted amount will be used solely by the allocator and will not be reflected in the resource slice. 
The subtracted amount is calculated based on the consumable conditions 
and will round the requested capacity up to the nearest valid amount under those conditions. 
If users do not specify a capacity request for consumable capacity, the default consumed value will be applied.

A shared device can only be allocated once its consumability has been verified
and its attributes match the request's selectors and constraints.
The newly added `capacity` field in the `DeviceRequestAllocationResult` will be set when the allocation is successful.

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
   //
   // +optional
   // +default=false
   // +featureGate=DRAConsumableCapacity
   Shared *bool `json:"shared" protobuf:"bytes,8,opt,name=shared"`
}

// DeviceCapacity describes a quantity associated with a device.
type DeviceCapacity struct {
   // Value defines how much of a certain device capacity is available.
   //
   // +required
   Value resource.Quantity `json:"value" protobuf:"bytes,1,rep,name=value"`


   // Consumable specifies a consumable property of capacity.
   // and refines constraints for consumable capacity.
   // If this field is not defined, the capacity is not consumable.
   //
   // +optional
   // +featureGate=DRAConsumableCapacity
   Consumable *ConsumableSpec `json:"consumable" protobuf:"bytes,2,rep,name=consumable"`
}

// ConsumableSpec defines constraints for consumable capacity.
// Either one of the consumable conditions must be defined.
type ConsumableSpec struct {
   // ConsumableSet defines a set of acceptable quantities of consuming requests.
   // +optional
   // +oneOf=ConsumeCondition
   Set *ConsumableSet `json:"set" protobuf:"bytes,1,opt,name=set"`


   // ConsumableRange defines an acceptable quantity range of consuming requests.
   // +optional
   // +oneOf=ConsumeCondition
   Range *ConsumableRange `json:"range" protobuf:"bytes,2,opt,name=range"`
}

// ConsumableSet defines a discrete set of consuming capacity.
// default field is required as a default value of consumed capacity.
type ConsumableSet struct {
   // +required
   Default resource.Quantity `json:"default" protobuf:"bytes,1,name=default"`
   // +required
   ValidValues []resource.Quantity `json:"validValues" protobuf:"bytes,2,rep,name=validValues"`
}

// ConsumableRange defines a valid range of consuming capacity.
// minimum field is required as a default value of consumed capacity.
// step field is used to define the block consuming.
type ConsumableRange struct {
   // +required
   Minimum resource.Quantity `json:"minimum" protobuf:"bytes,1,name=minimum"`
   // +optional
   Maximum *resource.Quantity `json:"maximum" protobuf:"bytes,2,opt,name=maximum"`
   // +optional
   Step *resource.Quantity `json:"step" protobuf:"bytes,3,opt,name=step"`
}

```

#### ResourceClaimSpec's DeviceRequest

```go

type DeviceRequest struct {
...
   // Capacity defines resource requirements against capacity.
   //
   // +optional
   // +featureGate=DRAConsumableCapacity
   Capacity *CapacityRequirements `json:"resources,omitempty" protobuf:"bytes,7,opt,name=resources"`
}


// CapacityRequirements define minimum capacity requests for a specific device request.
type CapacityRequirements struct {
   // Requests describe the amount of resources to be reserved from the device.
   // If Requests is omitted, it defaults to Limits if that is explicitly specified,
   // otherwise to an implementation-defined value. Requests cannot exceed Limits.
   // +optional
   Requests map[QualifiedName]resource.Quantity `json:"requests,omitempty" protobuf:"bytes,2,rep,name=requests"`
}

// DeviceConstraint must have exactly one field set besides Requests.
type DeviceConstraint struct {
	...
   // DistinctAttribute requires that all devices in question have this
   // attribute and that its type and value are unique across those
   // devices.
   //
   // For example, specify attribute name to get virtual devices from distinct shared physical devices.
   // Must include the domain qualifier.
   //
   // +optional
   // +oneOf=ConstraintType
   DistinctAttribute *FullyQualifiedName `json:"distinctAttribute,omitempty" protobuf:"bytes,3,opt,name=distinctAttribute"`
}
```

#### ResourceClaimStatus's DeviceRequestAllocationResult

```go

type DeviceRequestAllocationResult struct {
 ...
   // Shared indicates whether the allocated device is shared.
   //
   // +required
   // +featureGate=DRAConsumableCapacity
   Shared bool `json:"shared" protobuf:"bytes,6,name=device"`


   // ConsumedCapacity indicates a per-device capacity amount consumed by the claim request.
   // A summation of consumed request capacity must be less than or equal each corresponding capacity.
   //
   // +optional
   // +featureGate=DRAConsumableCapacity
   ConsumedCapacity *map[QualifiedName]resource.Quantity `json:"consumedCapacity,omitempty" protobuf:"bytes,7,rep,name=consumedCapacity"`
}

```

### Scheduling enhancement
- define [share-related types and functions](https://github.com/sunya-ch/kubernetes/blob/kep-5075/staging/src/k8s.io/dynamic-resource-allocation/structured/share.go).

  ```go
  // AllocatedCapacity define a quantity set which is updatable.
  // This field is used for aggregating allocated capacity,
  // and for calculating consumability.
  type AllocatedCapacity map[resourceapi.QualifiedName]*resource.Quantity

  // AllocatedCapacityCollection collects a set of AllocatedCapacity
  // for each shared device.
  type AllocatedCapacityCollection map[DeviceID]AllocatedCapacity
  ```

- shared device are handled separately from the other device.

  - define `ListAllAllocatedCapacity` function separately from `ListAllAllocatedDevices`
    ```go
    type ResourceClaimTracker interface {
      ...
  	  ListAllAllocatedDevices() (sets.Set[structured.DeviceID], error)
	    // ListAllAllocatedCapacity lists all allocated capacity of shared devices from allocated ResourceClaims. 
      // The result is guaranteed to immediately include
	    // any changes made via AssumeClaimAfterAPICall(), and SignalClaimPendingAllocation().
      ListAllAllocatedCapacity() (structured.AllocatedCapacityCollection, error)
      ...
    }
    ```

  - define `foreachAllocatedCapacity` and add condition to not add shared device in `foreachAllocatedDevice`.

    ```go
    func foreachAllocatedDevice(claim *resourceapi.ResourceClaim, cb func(deviceID s)){
        ...
        if result.Shared {
          continue
        }
        deviceID := structured.MakeDeviceID(result.Driver, result.Pool, result.Device)
        cb(deviceID)
    }

    func foreachAllocatedCapacity(claim *resourceapi.ResourceClaim, cb func(allocatedSharedDevice structured.SharedDeviceAllocation)) {
      ...
        if !result.Shared || result.ConsumedCapacity == nil {
          continue
        }
        deviceID := structured.MakeDeviceID(result.Driver, result.Pool, result.Device)
        sharedAllocation := structured.NewSharedDeviceAllocation(deviceID, result.ConsumedCapacity)
        cb(sharedAllocation)
      ...
    }
    ```

  - use `foreachAllocatedCapacity` for `ListAllAllocatedCapacity`, `addDevices` and `removeDevices`.

  - In addition to `allocatedDevices`, pass `aggregatedCapacity` from all claims' allocation result to `allocator`.

  - In addition to `allocatingDevices`, initialize `allocatingCapacity`.

  - After `isSelectable` check, check whether the (rounded-up) requested capacity is larger than capacity deducted by `aggregatedCapacity` and `allocatingCapacity`.

    ```go

    func (alloc *allocator) allocateOne(r deviceIndices) (bool, error) {
      ...
        for _, slice := range pool.Slices {
          for deviceIndex := range slice.Spec.Devices {
            shared := alloc.isSharedDevice(slice, deviceIndex)
            ...
            success, err := alloc.CmpRequestOverCapacity(requestIndices{claimIndex: r.claimIndex, requestIndex: r.requestIndex}, slice, deviceIndex)
            if err != nil {
              return false, err
            }
            if !success {
              alloc.logger.V(7).Info("Device has no enough capacity", "device", deviceID)
              continue
            }
            ...
            allocated, deallocate, err := alloc.allocateDevice(r, device, false, shared)
            ...
          }
        }
      ...
    }
    ```

    [CmpRequestOverCapacity Implementation](https://github.com/sunya-ch/kubernetes/blob/kep-5075/staging/src/k8s.io/dynamic-resource-allocation/structured/consumable_capacity.go)

  - Add `allocatingCapacity` instead of `allocatingDevice` to allow selecting same shared device.

    ```go
    if !shared && !request.adminAccess() && (alloc.allocatedDevices.Has(device.id) || alloc.allocatingDevices[device.id]) {
        alloc.logger.V(7).Info("Device in use", "device", device.id)
        return false, nil, nil
    
    }
    ...

    if !request.adminAccess() && !shared {
        alloc.allocatingDevices[device.id] = true
    }

    var consumedCapacity map[resourceapi.QualifiedName]resource.Quantity
    if alloc.features.ConsumableCapacity {
        var err error
        convertedCapacity := *(*map[resourceapi.QualifiedName]resourceapi.DeviceCapacity)(unsafe.Pointer(&device.slice.Spec.Devices[r.deviceIndex].Basic.Capacity))
        consumedCapacity = GetConsumedCapacityFromRequest(request.capacity(), convertedCapacity)
        if err != nil {
          return false, nil, fmt.Errorf("failed to get requested capacity: %w", err)
        }
        if shared {
          alloc.allocatingCapacity.Insert(NewDeviceAllocatedCapacity(device.id, consumedCapacity))
        }
    }
    ```

- add request's `capacity` and `shared` fields in `internalDeviceResult` for updating ResourceClaim status correspondingly

  ```go
  type internalDeviceResult struct {
    ...
    shared      bool
    capacity    *map[resourceapi.QualifiedName]resource.Quantity
  }
  ```

  ```go
  func (a *Allocator) Allocate(ctx context.Context, node *v1.Node) (finalResult []resourceapi.AllocationResult, finalErr error) {
        ...
        for i, internal := range internalResult.devices {
          allocationResult.Devices.Results[i] = resourceapi.DeviceRequestAllocationResult{
            ...
            Shared:      internal.shared,
          }
          if internal.capacity != nil {
              allocationResult.Devices.Results[i].ConsumedCapacity = *internal.capacity
          }
        }
  }
  ```

- update CEL selector to recognize shared attribute

  ```go
  const (
    ...
    sharedType = apiservercel.BoolType
  )
  ...
  deviceType := apiservercel.NewObjectType("kubernetes.DRADevice", fields(
      ...
      field(sharedVar, sharedType, true),
  ))
  ...
  shared := false
  if input.Shared != nil {
		shared = *input.Shared
  }

  variables := map[string]any{ 
    deviceVar: map[string]any{
      ...
      sharedVar:     shared,
    },
  }
  ```

- add `distinctAttribute` constraint

  ```go
  case constraint.DistinctAttribute != nil:
    distinctAttribute := draapi.FullyQualifiedName(*constraint.DistinctAttribute)
    logger := alloc.logger
    if loggerV := alloc.logger.V(6); loggerV.Enabled() {
      logger = klog.LoggerWithName(logger, "distinctAttributeConstraint")
      logger = klog.LoggerWithValues(logger, "distinctAttribute", distinctAttribute)
    }
    m := &distinctAttributeConstraint{
      logger:        logger,
      requestNames:  sets.New(constraint.Requests...),
      attributeName: distinctAttribute,
      attributes:    make(map[string]draapi.DeviceAttribute),
    }
    constraints[i] = m
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

The unit tests should include
- Unit test for consumable capacity related computations
  
  | Function | Test case |
  |---|---|
  |TestAllocatedCapacity|New/Add/Sub|
  |TestAllocatedCapacityCollection|New/Insert/Remove|
  |TestViolateConstraints|no constraint|
  ||less than maximum|
  ||more than maximum|
  ||in set|
  ||not in set|
  |TestCalculateConsumedCapacity|empty|
  ||min in range|
  ||default in set|
  ||more than min in range|
  ||less than min in range|
  ||with step (round up)|
  ||with step (no remaining)|
  |TestGetConsumedCapacityFromRequest|no request|

  [Test Implementation](https://github.com/sunya-ch/kubernetes/blob/kep-5075/staging/src/k8s.io/dynamic-resource-allocation/structured/consumable_capacity_test.go)

- Extension of TestAllocator in `structured` module. 

  The allocator should be able to handle the above user stories.

  `AllocatedCapacityCollection` will be added to the test case structure.

  ```go
    testcases := map[string]struct {
      ...
      allocatedCapacityDevices AllocatedCapacityCollection
      ...
    }
  ```

  |Test case|Shared|Device(s) Capacity|Allocated Capacity|Shared Class|Claim request(s)|Expected success
  |---|---|---|---|---|---|---|
  |shared-device-with-consumable-capacity|yes|2 consumable|0|yes|1+1|yes
  |shared-device-with-exceeded-consumable-capacity-request|yes|2 consumable|0|yes|1+2|no
  |shared-device-with-some-remaining-consumable-capacity|yes|2 consumable|1|yes|1|yes
  |shared-device-with-no-available-consumable-capacity|yes|1 consumable|1|yes|1|no
  |shared-device-with-unconsumable-capacity|yes|1 unconsumable|0|yes|1+1|yes
  |non-shared-device-with-single-consumable-capacity-request|no|1 consumable|0|yes|1|yes
  |non-shared-device-with-multiple-consumable-capacity-request|no|1 consumable|0|yes|1+1|no
  |exclude-non-shared-device-from-class-selector|yes|0|0|no|0|no
  |distinct-shared-device-success|yes|2x1 consumable|0|yes|1+1, distinct|yes
  |distinct-shared-device-failed|yes|2 consumable|0|yes|1+1, distinct|no

  [Test Implementation](https://github.com/sunya-ch/kubernetes/blob/kep-5075/staging/src/k8s.io/dynamic-resource-allocation/structured/allocator_test.go)

- `ListAllAllocatedCapacity` unit test to get AllocatedCapacityCollection without non-sharable devices.

- <package>: <date> - <current test coverage>

##### Integration tests

- Add test user story 1 to 6 in in `scheduler_perf/dra.go` when defining a shared device with and without consumable capacity.
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