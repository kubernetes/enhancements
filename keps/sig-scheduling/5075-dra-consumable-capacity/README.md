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
# KEP-5075: DRA Consumable Capacity

<!-- toc -->
- [Release Signoff Checklist](#release-signoff-checklist)
- [Summary](#summary)
  - [Goals](#goals)
  - [Non-Goals](#non-goals)
- [Proposal](#proposal)
  - [User Stories (Optional)](#user-stories-optional)
    - [Story 1](#story-1)
    - [Story 2](#story-2)
    - [Story 3](#story-3)
  - [Risks and Mitigations](#risks-and-mitigations)
- [Design Details](#design-details)
  - [API enhancement](#api-enhancement)
    - [ResourceSliceSpec's Device](#resourceslicespecs-device)
    - [CELDeviceSelector's description](#celdeviceselectors-description)
    - [ResourceClaimSpec's DeviceRequest](#resourceclaimspecs-devicerequest)
    - [ResourceClaimStatus's DeviceRequestAllocationResult](#resourceclaimstatuss-devicerequestallocationresult)
  - [Scheduling enhancement](#scheduling-enhancement)
  - [Handles Device Updates for <code>AllowMultipleAllocations</code> and <code>RequestPolicy</code>](#handles-device-updates-for-allowmultipleallocations-and-requestpolicy)
  - [Examples](#examples)
    - [DeviceClass's selector](#deviceclasss-selector)
    - [ResourceClaim with capacity requirement](#resourceclaim-with-capacity-requirement)
    - [ResourceClaim's request](#resourceclaims-request)
    - [ResourceClaim's status](#resourceclaims-status)
    - [ResourceClaim with distinctAttribute](#resourceclaim-with-distinctattribute)
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
  - [Identifying multi-allocatable Property of Device](#identifying-multi-allocatable-property-of-device)
  - [Selecting/Deselecting multi-allocatable Devices](#selectingdeselecting-multi-allocatable-devices)
  - [Preventing Same multi-allocatable Device from Being Allocated Multiple Times in the Same Claim](#preventing-same-multi-allocatable-device-from-being-allocated-multiple-times-in-the-same-claim)
  - [Identifying shared device in the device status](#identifying-shared-device-in-the-device-status)
  - [Defining a valid range for RequestPolicy](#defining-a-valid-range-for-requestpolicy)
  - [Alternative words](#alternative-words)
  - [Future Possibilities](#future-possibilities)
    - [RequestPolicy](#requestpolicy)
    - [CapacityRequirements](#capacityrequirements)
- [Infrastructure Needed (Optional)](#infrastructure-needed-optional)
<!-- /toc -->

## Release Signoff Checklist

Items marked with (R) are required *prior to targeting to a milestone / release*.

- [x] (R) Enhancement issue in release milestone, which links to KEP dir in [kubernetes/enhancements] (not the initial KEP PR)
- [x] (R) KEP approvers have approved the KEP status as `implementable`
- [x] (R) Design details are appropriately documented
- [x] (R) Test plan is in place, giving consideration to SIG Architecture and SIG Testing input (including test refactors)
  - [x] e2e Tests for all Beta API Operations (endpoints)
  - [ ] (R) Ensure GA e2e tests meet requirements for [Conformance Tests](https://github.com/kubernetes/community/blob/master/contributors/devel/sig-architecture/conformance-tests.md) 
  - [ ] (R) Minimum Two Week Window for GA e2e tests to prove flake free
- [ ] (R) Graduation criteria is in place
  - [ ] (R) [all GA Endpoints](https://github.com/kubernetes/community/pull/1806) must be hit by [Conformance Tests](https://github.com/kubernetes/community/blob/master/contributors/devel/sig-architecture/conformance-tests.md) 
- [x] (R) Production readiness review completed
- [x] (R) Production readiness review approved
- [ ] "Implementation History" section is up-to-date for milestone
- [ ] User-facing documentation has been created in [kubernetes/website], for publication to [kubernetes.io]
- [ ] Supporting documentation—e.g., additional design documents, links to mailing list discussions/SIG meetings, relevant PRs/issues, release notes

[kubernetes.io]: https://kubernetes.io/
[kubernetes/enhancements]: https://git.k8s.io/enhancements
[kubernetes/kubernetes]: https://git.k8s.io/kubernetes
[kubernetes/website]: https://git.k8s.io/website

## Summary

Without this KEP, device sharing is done by having multiple pods (and/or containers) reference the same resource claim,
and that resource claim has allocated the device.
This can be considered exclusive or dedicated allocation.
With this KEP, independent resource claims (and/or requests within a claim) can allocate shares of resources provided by the same underlying device.
This enables resource sharing across pods that are completely unrelated, potentially even across different namespaces.

When a device is shared across multiple resource claims, this enhancement enables device resource allocation to be drawn from the device’s overall capacity.
It ensures that the total resources consumed by all claim requests remain within the device’s capacity and comply with any defined `requestPolicy`, such as minimum per-claim resource requirements, if specified.
This concept is referred to as consumable capacity.
If a request does not specify particular device resource requirements, it implies an expectation of full device capacity.

Notably, each of these independent resource claims can still be referenced by one or more pods.
However, the device resources allocated to each request are shared without any isolation guarantees among the pods that reference the same request.

To achieve this, this KEP introduces
- a new device property field to distinguish between devices those can be allocated only once and those can be allocated multiple times,
- a capacity-aware scheduling mechanism that allows limiting or guaranteeing the capacity of devices among the resource claims (or requests) those are sharing,
- a new capacity requirement field in the device request of the resource claim,
- a new consumed capacity field in the allocation result of the resource claim,
- a method to associate the allocated device status to the allocation result in the resource claim.

With those in place, a resource claim with multiple requests might allocate the same device multiple times. This may or may not be desired, so this KEP also introduces:
- a distinct attribute constraint to prevent allocating the same multi-allocatable device in the same claim multiple times.

Relations to other KEPs:
- [KEP 4815](https://github.com/kubernetes/enhancements/issues/4815): The partitioned devices can be a multi-allocatable device or have mutually exclusive partitions where one partition is multi-allocatable and the other is not.
  The partitioning constraints (the remaining `SharedCounters`) are only checked once during the allocation of a multi-allocatable partitioned device.
  Meanwhile, the constraints introduced in this KEP (the remaining `Capacity` of the multi-allocatable partitioned device) are checked during every allocation of the same device.
- [KEP 5007](https://github.com/kubernetes/enhancements/issues/5007): The allocated share can be provisioned at the pre-bind step.
- [KEP 4817](https://github.com/kubernetes/enhancements/issues/4817): A single network device can be shared across multiple pods, with each allocated share's `NetworkData` identified by a unique Share ID.
- [KEP 4816](https://github.com/kubernetes/enhancements/issues/4816): The enhancement must be able to handle subrequests when the DRAPrioritizedList feature is enabled.

A motivating use case is to allocate a multi-allocatable network device in the [CNI DRA driver](https://github.com/kubernetes-sigs/cni-dra-driver)
which can be selected by more than one pod on demand during scheduling.
The original discussion is in [this PR's comment thread](https://github.com/kubernetes-sigs/cni-dra-driver/pull/1#discussion_r1889265214). 
The limitation of current implementation has been addressed [here](https://github.com/kubernetes-sigs/cni-dra-driver/pull/1#discussion_r1890166449).
The virtual network device is created and configured once the CNI is called based on the information of the master network device. 
The configured information specific to the generated device cannot be listed in the ResourceSlice in advance. 

This feature is also beneficial for the other multi-allocatable devices which are not within scope of [KEP-4815](https://github.com/kubernetes/enhancements/issues/4815).
For instance, this feature will be allow reserving memory fraction of virtual GPU in [the AWS virtual GPU device plugin](https://github.com/awslabs/aws-virtual-gpu-device-plugin).
In other words, the device capacity allocation is determined by the user's claim. 

### Goals

- Introduce an ability to allocate a multi-allocatable device via DRA multiple times
  in scenarios where pre-defined partitions are not viable, for example because there would be too many of them.
- Let DRA driver declare which device-level resource it can guarantee or reserve to a specific request and what are valid values that can be reserved,
- Let users specify in device requests how much of certain device resources they require.

### Non-Goals

- Define driver-specific attributes and configs (such as CNI parameter config).
- Support network security policy.
- Support aggregated resource consumption where multiple devices are allocated to satisfy a single capacity request.
  This is related to [the comment about `distinctAttributes`](https://github.com/kubernetes/enhancements/pull/5104#discussion_r1943835445).
- Support an extended use case where the resource guaranteeing behavior is determined by the first user request.
  For example, if the first request does not require a guarantee, the resource remains unguaranteed.
  However, if the first request requires a guarantee, the resource is marked as guaranteed, and all subsequent requests must adhere to that guarantee.

## Proposal

### User Stories (Optional)

#### Story 1

A DRA driver for networks advertises multi-allocatable devices for two interfaces eth1 and eth2
which each connect to the same virtual LAN, 
the admin makes those device available through a DeviceClass selecting only multi-allocatable devices,
and users request access through a request which references that DeviceClass.

#### Story 2

When requesting two interfaces, the user requests two devices. To ensure that they don't end up with the same multi-allocatable device for each request, they specify that the driver-specific "interfaceName" attribute must be different.

#### Story 3

A DRA driver for networks supports QoS guaranteed bandwidth which can ensures a specific bandwidth amount of the multi-allocatable network can be reserved exclusively to resource requests. A DRA driver also specifies minimum, maximum amount of reserved capacity for each resource request.
When requesting the guaranteed network device, 
users specifies their required guaranteed bandwidth. Otherwise, the default value defined by the DRA driver is applied.

### Risks and Mitigations

- The requested amount in the resource claim may not satisfy the capacity request policy,
  especially if the requested amount exceeds the maximum allowed consumption.

  This scenario should be handled similarly to other scheduling issues, such as when the request exceeds the allocatable capacity. 
  In such cases, the allocation fails and the pod remains pending.

- The driver includes both multi-allocatable and dedicated (non-multi-allocatable) devices. There is a risk that a user may be allocated a multi-allocatable device
  (e.g., a multi-allocatable network device) and accidentally configure it with the HostDevice CNI plugin.
  This would move the device from the host into the user's pod, preventing other users from accessing the multi-allocatable device.

  **Mitigation:**
  - Device drivers should define a concrete request policy. If the device is intended to be shared without capacity limits among requests, the request policy should set the consumable value to zero.
  - Additionally, administrators should define clear device classes for multi-allocatable and dedicated devices to prevent such misallocations.

- When a driver changes a device property from dedicated to multi-allocatable, existing resource claims that have no specified consumed capacity will adopt a default quantity based on the defined request policy. This default may represent a fraction of the device, potentially altering the behavior of existing claims.

  **Mitigation:**
  - The existing allocation result, which has no share ID (as it was previously a dedicated device), will be included in the allocated list. Scheduler must ensure that the device cannot be allocated for another resource claim during the scheduling process.

- When a driver updates the request policy, the behavior of resource claims changes.

  **Mitigation:**
  - Device driver should avoid updating the request policy, or do so with caution, if any devices have already been allocated to the ResourceClaim or are under preparation.

## Design Details

This enhancement introduces a `AllowMultipleAllocations` field within the `Device` of the ResourceSlice
to mark whether the device is multi-allocatable among multiple resource claims (or requests).
The multi-allocatable device can be assigned to more than one request if it satisfies the selection criteria and constraints.
The select condition `device.allowMultipleAllocations == true/false` can be used to select the device with a `AllowMultipleAllocations` property or not in a CEL selector.

The enhancement also adds a `RequestPolicy` field to `DeviceCapacity`.
This field specifies how the device resources can be drawn from the device’s capacity for each claim request.
The request policy can either specify a range of valid values or a discrete set of them.
Each policy must have a default value.

If a device with the `AllowMultipleAllocations` property does not contain any `Capacity`, it can be allocated multiple times without device capacity constraints—that is, infinitely, as long as other scheduling conditions are met.

Users can define specific per-device resource requests using the newly added `Capacity` field,
which is available in each supported device request type under `DeviceRequest`.
`Capacity` contains a `Requests` map, where each entry specifies the required amount of a device resource.
The amount available for allocation is determined
by subtracting the aggregated allocation results of current claims from the device's capacity as defined in the resource slice.
The remaining amount will be used solely by the allocator and will not be reflected in the resource slice.
The calculation of capacity requirements will round the requested capacity up to the nearest valid amount,
based on the capacity's request policy.

If users do not specify a capacity request, the consumed value will be:

- the device’s full capacity or
- the default value if it was specified by the request policy or
- none if the device usage is unlimited

A device with `AllowMultipleAllocations` property can only be allocated
when its consumability has been verified and its attributes match the request's selectors and constraints.
The newly added `ConsumedCapacity` field in the `DeviceRequestAllocationResult` will be set to the calculated capacity upon a successful allocation.
This value may differ from the originally requested amount, as it is rounded up to the nearest valid value based on the device’s request policy.

### API enhancement
To enable this enhancement, the following API updates are proposed.

#### ResourceSliceSpec's Device

```go
type Device struct {
...
    // AllowMultipleAllocations marks whether the device is allowed to be allocated to multiple DeviceRequests.
    //
    // If AllowMultipleAllocations is set to true, the device can be allocated more than once,
    // and all of its capacity is consumable, regardless of whether the requestPolicy is defined or not.
    //
    // +optional
    // +featureGate=DRAConsumableCapacity
    AllowMultipleAllocations *bool
}

type DeviceCapacity struct {
    // Value defines how much of a certain capacity that device has.
    //
    // This field reflects the fixed total capacity and does not change.
    // The consumed amount is tracked separately by scheduler
    // and does not affect this value.
    //
    // +required
    Value resource.Quantity

    // RequestPolicy defines how this DeviceCapacity must be consumed
    // when the device is allowed to be shared by multiple allocations.
    //
    // The Device must have allowMultipleAllocations set to true in order to set a requestPolicy.
    //
    // If unset, capacity requests are unconstrained:
    // requests can consume any amount of capacity, as long as the total consumed
    // across all allocations does not exceed the device's defined capacity.
    // If request is also unset, default is the full capacity value.
    //
    // +optional
    // +featureGate=DRAConsumableCapacity
    RequestPolicy *CapacityRequestPolicy
}

// CapacityRequestPolicy defines how requests consume device capacity.
//
// Must not set more than one ValidRequestValues.
type CapacityRequestPolicy struct {
    // Default specifies how much of this capacity is consumed by a request
    // that does not contain an entry for it in DeviceRequest's Capacity.
    //
    // +optional
    Default *resource.Quantity

    // ValidValues defines a set of acceptable quantity values in consuming requests.
    //
    // Must not contain more than 10 entries.
    // Must be sorted in ascending order.
    //
    // If this field is set,
    // Default must be defined and it must be included in ValidValues list.
    //
    // If the requested amount does not match any valid value but smaller than some valid values,
    // the scheduler calculates the smallest valid value that is greater than or equal to the request.
    // That is: min(ceil(requestedValue) ∈ validValues), where requestedValue ≤ max(validValues).
    //
    // If the requested amount exceeds all valid values, the request violates the policy,
    // and this device cannot be allocated.
    //
    // +optional
    // +listType=atomic
    // +oneOf=ValidRequestValues
    ValidValues []resource.Quantity

    // ValidRange defines an acceptable quantity value range in consuming requests.
    //
    // If this field is set,
    // Default must be defined and it must fall within the defined ValidRange.
    //
    // If the requested amount does not fall within the defined range, the request violates the policy,
    // and this device cannot be allocated.
    //
    // If the request doesn't contain this capacity entry, Default value is used.
    //
    // +optional
    // +oneOf=ValidRequestValues
    ValidRange *CapacityRequestPolicyRange
}

// CapacityRequestPolicyRange defines a valid range for consumable capacity values.
//
//   - If the requested amount is less than Min, it is rounded up to the Min value.
//   - If Step is set and the requested amount is between Min and Max but not aligned with Step,
//     it will be rounded up to the next value equal to Min + (n * Step).
//   - If Step is not set, the requested amount is used as-is if it falls within the range Min to Max (if set).
//   - If the requested or rounded amount exceeds Max (if set), the request does not satisfy the policy,
//     and the device cannot be allocated.
type CapacityRequestPolicyRange struct {
    // Min specifies the minimum capacity allowed for a consumption request.
    //
    // Min must be greater than or equal to zero,
    // and less than or equal to the capacity value.
    // requestPolicy.default must be more than or equal to the minimum.
    //
    // +required
    Min *resource.Quantity

    // Max defines the upper limit for capacity that can be requested.
    //
    // Max must be less than or equal to the capacity value.
    // Min and requestPolicy.default must be less than or equal to the maximum.
    //
    // +optional
    Max *resource.Quantity

    // Step defines the step size between valid capacity amounts within the range.
    //
    // Max (if set) and requestPolicy.default must be a multiple of Step.
    // Min + Step must be less than or equal to the capacity value.
    //
    // +optional
    Step *resource.Quantity
}
```

#### CELDeviceSelector's description

```go
type CELDeviceSelector struct {
    // ...
    // The expression's input is an object named "device", which carries
    // the following properties:
    //  - driver (string): the name of the driver which defines this device.
    //  - attributes (map[string]object): the device's attributes, grouped by prefix
    //    (e.g. device.attributes["dra.example.com"] evaluates to an object with all
    //    of the attributes which were prefixed by "dra.example.com".
    //  - capacity (map[string]object): the device's capacities, grouped by prefix.
    //  - allowMultipleAllocations (bool): the allowMultipleAllocations property of the device
    //    (v1.34+ with the DRAConsumableCapacity feature enabled).
    // ...
    // +required
    Expression string
}
```

#### ResourceClaimSpec's DeviceRequest

The `Capacity` field is defined within each supported device request type, such as `DeviceSubRequest` and `ExactDeviceRequest`.

```go
type DeviceSubRequest struct {

    // Capacity define resource requirements against each capacity.
    //
    // If this field is unset and the device supports multiple allocations,
    // the default value will be applied to each capacity according to requestPolicy.
    // For the capacity that has no requestPolicy, default is the full capacity value.
    //
    // Applies to each device allocation.
    // If Count > 1,
    // the request fails if there aren't enough devices that meet the requirements.
    // If AllocationMode is set to All,
    // the request fails if there are devices that otherwise match the request,
    // and have this capacity, with a value >= the requested amount, but which cannot be allocated to this request.
    //
    // +optional
    // +featureGate=DRAConsumableCapacity
    Capacity *CapacityRequirements

}

type ExactDeviceRequest struct {

    // Capacity define resource requirements against each capacity.
    //
    // If this field is unset and the device supports multiple allocations,
    // the default value will be applied to each capacity according to requestPolicy.
    // For the capacity that has no requestPolicy, default is the full capacity value.
    //
    // Applies to each device allocation.
    // If Count > 1,
    // the request fails if there aren't enough devices that meet the requirements.
    // If AllocationMode is set to All,
    // the request fails if there are devices that otherwise match the request,
    // and have this capacity, with a value >= the requested amount, but which cannot be allocated to this request.
    //
    // +optional
    // +featureGate=DRAConsumableCapacity
    Capacity *CapacityRequirements

}

// CapacityRequirements defines the capacity requirements for a specific device request.
type CapacityRequirements struct {
    // Requests represent individual device resource requests for distinct resources,
    // all of which must be provided by the device.
    //
    // This value is used as an additional filtering condition against the available capacity on the device.
    // This is semantically equivalent to a CEL selector with
    // `device.capacity[<domain>].<name>.compareTo(quantity(<request quantity>)) >= 0`.
    // For example, device.capacity['test-driver.cdi.k8s.io'].counters.compareTo(quantity('2')) >= 0.
    //
    // When a requestPolicy is defined, the requested amount is adjusted upward
    // to the nearest valid value based on the policy.
    // If the requested amount cannot be adjusted to a valid value—because it exceeds what the requestPolicy allows—
    // the device is considered ineligible for allocation.
    //
    // For any capacity that is not explicitly requested:
    // - If no requestPolicy is set, the default consumed capacity is equal to the full device capacity
    //   (i.e., the whole device is claimed).
    // - If a requestPolicy is set, the default consumed capacity is determined according to that policy.
    //
    // If the device allows multiple allocation,
    // the aggregated amount across all requests must not exceed the capacity value.
    // The consumed capacity, which may be adjusted based on the requestPolicy if defined,
    // is recorded in the resource claim’s status.devices[*].consumedCapacity field.
    //
    // +optional
    Requests map[QualifiedName]resource.Quantity
}


type DeviceConstraint struct {

    // DistinctAttribute requires that all devices in question have this
    // attribute and that its type and value are unique across those devices.
    //
    // This acts as the inverse of MatchAttribute.
    //
    // This constraint is used to avoid allocating multiple requests to the same device
    // by ensuring attribute-level differentiation.
    //
    // This is useful for scenarios where resource requests must be fulfilled by separate physical devices.
    // For example, a container requests two network interfaces that must be allocated from two different physical NICs.
    //
    // +optional
    // +oneOf=ConstraintType
    // +featureGate=DRAConsumableCapacity
    DistinctAttribute *FullyQualifiedName
}
```

#### ResourceClaimStatus's DeviceRequestAllocationResult

```go
type DeviceRequestAllocationResult struct {

    // ShareID uniquely identifies an individual allocation share of the device,
    // used when the device supports multiple simultaneous allocations.
    // It serves as an additional map key to differentiate concurrent shares
    // of the same device.
    //
    // +optional
    // +featureGate=DRAConsumableCapacity
    ShareID *types.UID

    // ConsumedCapacity tracks the amount of capacity consumed per device as part of the claim request.
    // The consumed amount may differ from the requested amount: it is rounded up to the nearest valid
    // value based on the device’s requestPolicy if applicable (i.e., may not be less than the requested amount).
    //
    // The total consumed capacity for each device must not exceed the DeviceCapacity's Value.
    //
    // This field is populated only for devices that allow multiple allocations.
    // All capacity entries are included, even if the consumed amount is zero.
    //
    // +optional
    // +featureGate=DRAConsumableCapacity
    ConsumedCapacity map[QualifiedName]resource.Quantity
}

type ResourceClaimStatus struct {

    // Devices contains the status of each device allocated for this
    // claim, as reported by the driver. This can include driver-specific
    // information. Entries are owned by their respective drivers.
    //
    // +optional
    // +listType=map
    // +listMapKey=driver
    // +listMapKey=device
    // +listMapKey=pool
    // +listMapKey=shareID
    // +featureGate=DRAResourceClaimDeviceStatus
    Devices []AllocatedDeviceStatus
}

type AllocatedDeviceStatus struct {

    // ShareID uniquely identifies an individual allocation share of the device.
    //
    // +optional
    // +featureGate=DRAConsumableCapacity
    ShareID *string
}
```

### Scheduling enhancement
- When the scheduler invokes the `Allocate` function in the allocator, 
  the total allocated capacity is calculated by aggregating the consumedCapacity from all resource claims's `DeviceRequestAllocationResult` that have already been allocated.
- Before allocation proceeds, existing selection criteria (defined by `alloc.isSelectable`) are evaluated. 
  These include the class selector and request selector.
- A new `device.allowMultipleAllocations` key is introduced in the CEL selector,
  enabling policies and constraints to recognize whether a device supports allocation by multiple requests.
- If a device is considered selectable, the `CmpRequestOverCapacity` function is invoked to verify 
  whether the consumed capacity would exceed the device's remaining capacity. 
  The remaining capacity is calculated based on the sum of already allocated and currently allocating capacities.
  - consumed capacity is derived from the requested amount specified in the resource claim, adjusted by the device’s capacity request policy, if defined.
  - This value may differ from the originally requested amount—it is rounded up to the nearest valid capacity according to the policy (e.g., using Min + ⌈(Requested - Min)/Step⌉ × Step logic).
- If the device has enough remaining capacity to satisfy the consumed amount, constraint checks are applied. 
  In addition to the existing MatchAttribute, this proposal introduces a new constraint: `DistinctAttribute`, which ensures attribute uniqueness across allocated devices.
- Once all selection and constraint checks pass, the allocation is valid. The allocation result is updated with:
  - The share identifier (ShareID), which uniquely identifies the allocation on a device.
  - The consumed capacity. This consumed capacity is tracked as part of the device’s `allocatingCapacity`, 
    allowing it to be included in remaining capacity calculations for future allocations within the same call.
- Finally, the share identifiers and consumed capacities from all internal results
  are propagated to the DeviceRequestAllocationResult.

### Handles Device Updates for `AllowMultipleAllocations` and `RequestPolicy`

- If a device is updated from **dedicated** (`allowMultipleAllocations: false`) to **multi-allocatable** (`allowMultipleAllocations: true`), it must continue to behave as a dedicated device and not allow sharing **until all existing resource claims for that device are released**.
- If a device is updated from multi-allocatable to dedicated, it should no longer be available for new allocations. However, already allocated devices should not be deallocated.
- If the **request policy** is later set, update, or unset, the change will apply only to **future allocations**. No rollback or changes will be applied to shared devices that have already been allocated.

### Examples

#### DeviceClass's selector

```yaml
selectors:
  - cel:
      expression: |-
        device.allowMultipleAllocations == true
```

#### ResourceClaim with capacity requirement

```yaml
kind: ResourceSlice
...
spec:
  driver: guaranteed-cni.dra.networking.x-k8s.io
  devices:
  - name: eth1
    basic:
      allowMultipleAllocations: true
      attributes:
        name:
          string: "eth1"
      capacity:
        bandwidth:
          requestPolicy:
            default: "1Mi"
            validRange:
              min: "1Mi"
              step: "8"
          value: "10Gi"
```

#### ResourceClaim's request

```yaml
kind: ResourceClaim
...
spec:
  devices:
    requests: # for devices
    - name: nic
      exactly:
        deviceClassName: qos-aware-shared.device.x-k8s.io
        capacity:
          requests: # for resources which must be provided by those devices
            bandwidth: 5Gi
```

#### ResourceClaim's status

```yaml
kind: ResourceClaim
...
status:
  allocation:
    devices:
      results:
      - consumedCapacity:
          bandwidth: 1Mi
        device: eth1
        shareID: "a671734a-e8e5-11e4-8fde-42010af09327"
        ...
 devices:
    - data:
        cniVersion: 1.1.0
        ips:
        - address: 10.0.103.49/16
      device: eth1
      shareID: "a671734a-e8e5-11e4-8fde-42010af09327"
      ...
```

#### ResourceClaim with distinctAttribute

```yaml
kind: ResourceClaim
...
spec:
  devices:
    requests:
    - name: macvlan-1
      exactly:
        deviceClassName: simple-multialloc.networking.x-k8s.io
        allocationMode: ExactCount
        count: 1
    - name: macvlan-2
      exactly:
        deviceClassName: simple-multialloc.networking.x-k8s.io
        allocationMode: ExactCount
        count: 1
    constraints:
    - requests:
      - macvlan-1
      - macvlan-2
      distinctAttribute: interfaceName
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

###### API Validations

**Sharing Policy (Device Capacity Test)**

- Default value is required.
- The default must be included in the options for `validValues`, or fall within the specified `validRange`.
- `validValues` and `validRange` must not be defined at the same time within a single `sharingpolicy`.
- `validValues` must be a list of unique values.
- `validValues` must be in ascending order.
- The `validValues` size should be kept within 10 to avoid excessive growth.
- The minimum must be less than or equal to the maximum in the `validRange`.
- If a chunk size is defined, both the default and the maximum must be multiples of the chunk size.
- The minimum, maximum, and (minimum + chunk size) must each be less than the capacity value.
- If `AllowMultipleAllocations` of the device is not set or set to false, `RequestPolicy`for any of its capacity must not be defined.

**Distinct Attribute**

- Similar to the `matchAttribute`, check for a missing domain and required name (invalid request).
- If the feature gate is enabled, exactly one of `matchAttribute` or `distinctAttribute` must be provided.
- If the feature gate is disabled, `matchAttribute` is required.

**Share ID**

- When this feature gate and `DRAResourceClaimDeviceStatus` ([KEP 4817](https://github.com/kubernetes/enhancements/issues/4817)) are enabled, the combination keys of `driver`, `pool`, `device`, and `shareID`in `Status.Devices` must be a one-to-one mapping with those keys in `Status.Allocation.Devices`.
- Must be a valid UID

**Create Strategy**

- Keep fields if the feature is enabled for `ResourceSlice`, `ResourceClaim`, and `ResourceClaimTemplate`
- Drop fields if the feature is disabled for `ResourceSlice`, `ResourceClaim`, and `ResourceClaimTemplate`

**Update Strategy**

- Keep existing fields if the feature is enabled `ResourceSlice`, `ResourceClaim`, and `ResourceClaimTemplate`
- Keep existing fields of `ResourceSlice` if any feature field is set in the old `ResourceSlice`.
- Keep existing fields of `ResourceClaim` and `ResourceClaimTemplate` if any feature field is set in the old `ResourceClaim` or `ResourceClaimTemplate`.
- Keep existing fields of `ResourceClaim` if any feature field is set in the old `ResourceClaim.Status`.
- Drop fields if the feature is disabled and the fields has not been used by the old resource as described above.
  - If the `DRAPrioritizedList` is enabled, the `Capacity` of `DeviceSubRequest` in `FirstAvailable` must be dropped as well.
- The same strategy for `ResourceClaim` must be followed regardless of `DRADeviceStatus` feature enablement.

###### Allocator

**Allow Multiple Allocations**

- can allocate a device which allow multiple allocations for multiple times
- must not allocate a device which do not allow multiple allocations more than once
- can exclude dedicated device from allocation with CEL
- can limit allocation to multi-allocatable device with CEL
- can work with `DRAPartitionableDevices` feature.

**Consumable Capacity**

- can gather consumed capacity from allocated resource claims
- can add/remove consumed capacity of allocating devices
- can round up and compute user-requesting minimum capacity according to request policy range and chunk size
- requested capacity for non-consumable capacity acts like a `>=` filter
- can work with `DRAPrioritizedList` feature's subrequests.

**Distinct Attribute**

- can prevent allocating the same device in the same request with a distinct constraint
- can allocate different device in the same request with a distinct contraint

###### Coverage

- `k8s.io/dynamic-resource-allocation/structured/internal/experimental`: `4/2/2026` - `93.1`
- `k8s.io/dynamic-resource-allocation/structured/internal/incubating`: `4/2/2026` - `93.1`
- `k8s.io/kubernetes/pkg/apis/resource/validation`: `4/2/2026` - `96.8`
- `k8s.io/kubernetes/pkg/registry/resource/resourceclaimtemplate`: `4/2/2026` - `72.0`
- `k8s.io/kubernetes/pkg/registry/resource/resourceclaim`: `4/2/2026` - `87.1`
- `k8s.io/kubernetes/pkg/registry/resource/resourceslice`: `4/2/2026` - `77.7`
- `k8s.io/kubernetes/pkg/kubelet/cm/dra`: `4/2/2026` - `83.5`
- `k8s.io/kubernetes/pkg/kubelet/cm/dra/plugin`: `4/2/2026` - `83.5`
- `k8s.io/kubernetes/pkg/kubelet/cm/dra/state`: `4/2/2026` - `44.2`

##### Integration tests

The existing [integration tests for kube-scheduler which measure performance](https://github.com/kubernetes/kubernetes/tree/master/test/integration/scheduler_perf#readme) will be extended to cover the overheaad of running the additional logic to support the features in this KEP. 

We extend the test for creating large ResourceSlices to ensure that a ResourceSlice using the new fields satisfies the etcd limits.

##### e2e tests

<!--
This question should be filled when targeting a release.
For Alpha, describe what tests will be added to ensure proper quality of the enhancement.

For Beta and GA, add links to added tests together with links to k8s-triage for those tests:
https://storage.googleapis.com/k8s-triage/index.html

We expect no non-infra related flakes in the last month as a GA graduation criteria.
-->

We extend [the DRA test driver](https://github.com/kubernetes/kubernetes/tree/master/test/e2e/dra/test-driver) to enable support for this feature and add tests to ensure they are handled by the scheduler as described in this KEP.

The following functionalities should be covered in E2E tests:

- **ResourceSlice creation**: The ResourceSlice must be created successfully with `AllowMultipleAllocations` and a `RequestPolicy`.
- **Pod Scheduling with Available Capacity**: A Pod with a resource claim must run successfully when the requested capacity is available.
- **Capacity Enforcement**: A Pod must stay in Pending state if it requests more than the remaining capacity, even if the request is less than the total capacity.
- **Capacity Release and Re-Scheduling**: When a Pod is deleted, its reserved capacity must be released, and any pending Pod with a satisfied request must start running.

### Graduation Criteria

#### Alpha

- Feature implemented behind feature gates (`DRAConsumableCapacity`). Feature Gates are disabled by default.
- Documentation provided
- Initial unit, integration and e2e tests completed and enabled.

### Beta

- Feature Gates are enabled by default.
- No major outstanding bugs.
- 2 examples of real-world use cases.
  - CNI DRA driver (kubernetes-sigs/cni-dra-driver) can use this feature to manage and limit bandwidth quota.
  - DRA Driver for CPU (kubernetes-sigs/dra-driver-cpu) can use this feature to manage and limit CPU resources.
- Feedback collected from the community (developers and users) with adjustments provided, implemented and tested.

### GA

- 2 examples of real-world use cases.
  - CNI DRA driver (kubernetes-sigs/cni-dra-driver) can use this feature to manage and limit bandwidth quota.
  - Acelerator DRA driver can use this feature for on-demand virtual memory allocation.
- Allowing time for feedback from developers and users.

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

-->

### Upgrade / Downgrade Strategy

In the context of this enhancement, the following strategy is proposed:

* **All introduced fields are optional and can be omitted if empty.** This means that during the upgrade or downgrade process, if certain fields or configurations are not required, they can be left out without causing issues or disrupting the upgrade process.

* The general upgrade and downgrade processes will follow the DRA strategy.

* The upgrade and downgrade processes of shareID will follow the optional map keys strategy.

* The upgrade and downgrade processes of allowMultipleAllocations CEL will follow the VersionOptions method.

### Version Skew Strategy

During version skew, where the API server supports the feature but the scheduler does not, the scheduler will throw an error if the `ResourceClaim` contains `Capacity` to prevent allocating the devices that doesn't meet the user requests. If there is no `Capacity`, the scheduler continues scheduling and ignores the `allowMultipleAllocation` and `requestPolicy` fields in `ResourceSlice`.

If the feature is enabled on scheduler but is disabled on API server. The scheduler can continue scheduling as-is without feature fields.

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

- [x] Feature gate (also fill in values in `kep.yaml`)
  - Feature gate name: DRAConsumableCapacity
  - Components depending on the feature gate:
    - kube-scheduler
    - kubelet
    - kube-apiserver
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

No

###### Can the feature be disabled once it has been enabled (i.e. can we roll back the enablement)?

<!--
Describe the consequences on existing workloads (e.g., if this is a runtime
feature, can it break the existing applications?).

Feature gates are typically disabled by setting the flag to `false` and
restarting the component. No other changes should be necessary to disable the
feature.

NOTE: Also set `disable-supported` to `true` or `false` in `kep.yaml`.
-->

Yes, this feature can be disabled once it has been enabled.
The `AllowMultipleAllocations` flag, `RequestPolicy` and `Capacity` fields will be dropped.
However, the `ShareID`, `ConsumedCapacity`, and renamed device (`<device id>/<share id>`) in device status needs to remain to keep the existing allocation result reference valid.

###### What happens if we reenable the feature if it was previously rolled back?

The fields will be available again for read and write. 
However, the previously dropped `RequestPolicy`, `Capacity`, and `ConsumedCapacity` will be missing. 

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

The enablement and disablement of this feature are tested as part of the integration tests. 
Additionally, the feature enablement/disablement tests cover the scenario where the feature gate is switched 
from enabled to disabled after an allocation has already been made. 
In this case, the existing resource claim should remain valid, 
but the remaining device capacity must no longer be multi-allocatable.

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

- Enabling the feature gate will enable the field to be written and therefore invoke validation of the field.
- Disabling the feature gate will drop the ability to consume the capacity in scheduling so that the `ConsumedCapacity` in the allocation result should be also dropped. If the external party uses the reference to this field to manage the QoS-aware devices, it may fail if there is no handler.
- Disabling the feature gate is equivalent to unset `AllowMultipleAllocations` and `RequestPolicy`, the scheduler will handle as described in [this previous section](#handles-device-updates-for-allowmultipleallocations-and-sharingpolicy).

###### What specific metrics should inform a rollback?

When we notice unexpected `scheduler_unschedulable_pods{plugin="DynamicResources"}` or metric `scheduler_plugin_execution_duration_seconds{plugin="DynamicResources"}` in the kube-scheduler suddenly increases.

###### Were upgrade and rollback tested? Was the upgrade->downgrade->upgrade path tested?

The manual test was performed on a local Kind cluster by manually disabling and enabling the feature gate for all control plane components on the Kind node.

When this feature is enabled, a `ResourceClaim` with the added fields can be deployed and the driver can advertise 10G of bandwidth. Workloads which requests 5G can request capacity from devices that allow multiple allocations, and the consumed capacity is updated in `ResourceClaim.Status`.

When the feature is disabled, existing workloads continue running, and there is no change to `ResourceClaim`, including the status of consumed capacity. However, new workloads, requesting 2G, that include a capacity request are rejected and remain in a pending state. Additionally, the fields added by this feature are removed when applying a new `ResourceClaimTemplate`.

When the feature is re-enabled, a new `ResourceClaimTemplates` can be created with the added fields. The scheduler can properly prevent over-provisioning of capacity when trying to deploy another workload which requests 8G while allow the workload which requests only 2G to run with their consumed capacity tracked in `ResourceClaim.Status`, as intended by this feature, without impacting on the first workload that was already running.

###### Is the rollout accompanied by any deprecations and/or removals of features, APIs, fields of API types, flags, etc.?

<!--
Even if applying deprecation policies, they may still surprise some users.
-->

No

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

Check the `allowMultiAllocation` flag in the resource slice.

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
- [x] API .status
  - Condition name: 
  - Other field: `ResourceClaim.Status.Allocation.Devices.Results[].ShareID`
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

Existing DRA and related SLOs continue to apply.

###### What are the SLIs (Service Level Indicators) an operator can use to determine the health of the service?

- [x] Metrics
  - Metric names:
    - `apiserver_request` with `resource="resourceclaims"`
    - `scheduler_unschedulable_pods` with `plugin="DynamicResources"`
    - `scheduler_plugin_execution_duration_seconds` with `plugin="DynamicResources"`
        - For state gathering, `extension_point="PreFilter"`
        - For allocation, `extension_point="Filter"`
        - For status update, `extension_point="PostFilter"`
  - [Optional] Aggregation method:
  - Components exposing the metric: kube-apiserver, kube-scheduler
- [ ] Other (treat as last resort)
  - Details:

###### Are there any missing metrics that would be useful to have to improve observability of this feature?

<!--
Describe the metrics themselves and the reasons why they weren't added (e.g., cost,
implementation difficulties, etc.).
-->

No.

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

This feature depends on the DRA structured parameters feature being enabled, and on DRA drivers that support the feature being deployed.
This feature also works with DRA device status feature if it is enabled.

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

No.

###### Will enabling / using this feature result in introducing new API types?

<!--
Describe them, providing:
  - API type
  - Supported number of objects per cluster
  - Supported number of objects per namespace (for namespace-scoped objects)
-->

There will be `CapacityRequestPolicy` and `CapacityRequirements` struct added to `DeviceCapacity` in `ResourceSlice` and `DeviceSubRequest/ExactDeviceRequest` in `ResourceClaim`.

###### Will enabling / using this feature result in any new calls to the cloud provider?

<!--
Describe them, providing:
  - Which API(s):
  - Estimated increase:
-->

No.

###### Will enabling / using this feature result in increasing size or count of the existing API objects?

<!--
Describe them, providing:
  - API type(s):
  - Estimated increase in size: (e.g., new annotation of size 32B)
  - Estimated amount of new objects: (e.g., new Object X for every existing Pod)
-->

Yes, when using this field, the user will add additional data in their `ResourceSlice`, `ResourceClaim` and `ResourceClaimTemplate` objects.
This is an incremental increase on top of the existing structures.

Estimated increase in size:
- ~ 10 bytes of boolean pointer per device
- ~ 200-1100 bytes per request policy (max 10 options)
- ~ 100 bytes per capacitiy per request and allocation result
  (`ResourceSliceMaxAttributesAndCapacitiesPerDevice`=32)
- ~ 40 bytes of share ID per resource allocation
- 7 bytes extended name in device name if the device status feature is enabled

###### Will enabling / using this feature result in increasing time taken by any operations covered by existing SLIs/SLOs?

<!--
Look at the [existing SLIs/SLOs].

Think about adding additional work or introducing new steps in between
(e.g. need to do X to start a container), etc. Please describe the details.

[existing SLIs/SLOs]: https://git.k8s.io/community/sig-scalability/slos/slos.md#kubernetes-slisslos
-->

Scheduling a claim that uses this feature may take a bit longer, if it is necessary to calculate aggregation of consumed capacity before finding a suitable device.
We will measure in beta timeframe.

###### Will enabling / using this feature result in non-negligible increase of resource usage (CPU, RAM, disk, IO, ...) in any components?

<!--
Things to keep in mind include: additional in-memory state, additional
non-trivial computations, excessive access to disks (including increased log
volume), significant amount of data sent and/or received over network, etc.
This through this both in small and large cases, again with respect to the
[supported limits].

[supported limits]: https://git.k8s.io/community//sig-scalability/configs-and-limits/thresholds.md
-->

No.

###### Can enabling / using this feature result in resource exhaustion of some node resources (PIDs, sockets, inodes, etc.)?

<!--
Focus not just on happy cases, but primarily on more pathological cases
(e.g. probes taking a minute instead of milliseconds, failed pods consuming resources, etc.).
If any of the resources can be exhausted, how this is mitigated with the existing limits
(e.g. pods per node) or new limits added by this KEP?

Are there any tests that were run/should be run to understand performance characteristics better
and validate the declared limits?
-->

No.

### Troubleshooting

The troubleshooting section in https://github.com/kubernetes/enhancements/tree/master/keps/sig-node/4381-dra-structured-parameters#troubleshooting
still applies. The only additional failure modes comes from version skew
in the cluster and the troubleshooting steps provided through the link above
should be sufficient to determine the cause.

###### How does this feature react if the API server and/or etcd is unavailable?

See https://github.com/kubernetes/enhancements/tree/master/keps/sig-node/4381-dra-structured-parameters#how-does-this-feature-react-if-the-api-server-andor-etcd-is-unavailable.

###### What are other known failure modes?

See https://github.com/kubernetes/enhancements/tree/master/keps/sig-node/4381-dra-structured-parameters#what-are-other-known-failure-modes.

- kube-scheduler cannot allocate ResourceClaims.

  The shared device may not have sufficient capacity to satisfy the request. The log message `Device capacity not enough` and the `capacities` field in the log `Allocating one device` can provide further clues for investigation (require -v=7 on kube-scheduler).

If the feature is disabled but a ResourceClaim still requests capacity, the scheduler log will report:
has capacity requests, but the DRAConsumableCapacity feature is disabled. Nevertheless, when using the allocator in stable mode, no logs related to the DRAConsumableCapacity feature will be emitted.


###### What steps should be taken if SLOs are not being met to determine the problem?

N/A

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

Alpha 1.34:

- [Initial implementation](https://github.com/kubernetes/kubernetes/pull/132522) merged on 2025-07-30

Alpha 1.35:

- [Fix scheduler perf test (simplified)](https://github.com/kubernetes/kubernetes/pull/133983) merged on 2025-09-10
- [Fix 133705 - failed to schedule next device PR 133706](https://github.com/kubernetes/kubernetes/pull/133706) has been pushed on 2025-08-26
- [Fix 134100 - integration with partitionable device PR 134103](https://github.com/kubernetes/kubernetes/pull/134103) has been pushed on 2025-09-17
- [Fix 134519 - add ShareID to kubelet plugin API PR 134520](https://github.com/kubernetes/kubernetes/pull/134520) has been pushed on 2025-10-10
- [Increase test coverage PR 134615](https://github.com/kubernetes/kubernetes/pull/134615) has been pushed on 2025-10-15

Beta 1.36:

- [Fix 136734 - missing GetSharedDeviceIDs bug in GatherAllocatedState](https://github.com/kubernetes/kubernetes/pull/136734) has been pushed on 2025-02-04
- [Promote DRAConsumableCapacity to Beta PR 136611](https://github.com/kubernetes/kubernetes/pull/136611) has been pushed on 2026-01-29

## Drawbacks

<!--
Why should this KEP _not_ be implemented?
-->

This adds complexity to the scheduler.

## Alternatives

### Identifying multi-allocatable Property of Device

**Current Approach:**

Use a **boolean** to indicate whether a device can be shared among multiple resource claims (or requests).

Pros:
- Simple

Cons:
- Implicit infinite sharing if no consuming capacity defined

**Alternatives:**

1. Use an **enum**, such as `Allocatable`, with defined values like:

  - `AllocatableOnce` — device can only be allocated once
  - `AllocatableMultipleTimes` — device can be allocated multiple times

    Pros:
    - Provides flexibility for future extension according to [Kubernetes API conventions](https://github.com/kubernetes/community/blob/master/contributors/devel/sig-architecture/api-conventions.md#primitive-types).

    Cons:
    - Increases the program’s memory footprint
    compared to a boolean when there is only a single binary option to serve the purpose.

2. Use a **count** field to specify how many times a device can be reallocated to different resource requests.

    Pros:
    - Simple.
    - No implicit infinite sharing.

    Cons:
    - Not equivalent to the legacy CNI, which places no limit on the number of master devices, as long as the Pod can be successfully created.

### Selecting/Deselecting multi-allocatable Devices

**Current Approach:**

Extend the CEL selector to recognize device.shareable for filtering multi-allocatable devices.

**Alternative:**

Introduce explicit flags in the resource request:

- AllowShared: Opt-in to allow multi-allocatable devices.
- RequireShared: Only allow multi-allocatable devices.

(Default: multi-allocatable devices are excluded unless explicitly allowed.)

Pros:
- Does not affect dedicated device selection.
- Easier for users to understand and configure, reducing the risk of mistakes.
- More user-friendly than writing CEL expressions manually.

Cons:
- Adds complexity to the allocation logic for multi-allocatable devices.
- Introduces an additional field in resource requests.
- May require an abstraction layer if more device features are added in the future.
- Less explicit and expressive than CEL for advanced use cases.

### Preventing Same multi-allocatable Device from Being Allocated Multiple Times in the Same Claim

**Current Approach:**

Introduce a new API-level constraint: DistinctAttribute, ensuring devices in a single claim have unique attribute values.

**Alternative:**

The scheduler enforces this behavior implicitly—never allocate the same multi-allocatable device multiple times to the same resource claim.

Pros:
- Avoids any API changes—logic handled internally.

Cons:
- Doesn’t support cases where a pod legitimately permits multiple fractions of capacity from the same multi-allocatable device.
  For example, when a pod uses two vGPUs for parallel processing,
  it may not require them to come from different devices.
  It can accept allocations from either the same or different multi-allocatable devices.
- Not configurable—users can't override this behavior when needed.

### Identifying shared device in the device status

When the same device are allocated multiple times to different requests, `ShareID` is required to differentiate between different allocation especially useful when the different allocation has a different status set in the `AllocatedDeviceStatus`.

`ShareID`, which is intended to serve as part of a composite key for identifying devices, cannot be added to the map keys of the listType because fields used as map keys must be required. Making `ShareID` required is not an option, as the API feature gate should not introduce required fields.

**Current Approach:**

Update `api-machinery` to support adding new keys to listType=map - [GitHub Issue](https://github.com/kubernetes/kubernetes/issues/132524)

**Alternative:**

Append the `Device` with `/<share id>` and workarounds on validation function.

### Defining a valid range for RequestPolicy

**Current Approach:**

Newly define minimum, maximum, and chunk size and implement a function to validate the value in range.

**Alternative:**

The range can be defined and validated using a `LimitRange` match.
Enforcing min/max/default values via `LimitRange` is a generally useful mechanism.

### Alternative words

- `Device.AllowMultipleAllocations`: The alternative names proposed were: `Shareable`, `Shared`, and `AllowShared`. 

- `Step`: The alternative names proposed were: `ChunkSize`, `StepSize`, `UnitSize`.

- `DeviceRequest.Capacity`: The alternative names proposed were: `Capacities`, `Capacity`.

- `DeviceRequest.Capacity.Requests`:
  The alternative names proposed were: `Required`, `Reservation`, `Consumption`, `Minimum` and `Min`.

  > `Requests` was dropped once since it's already used in the DRA API for device requests.
  `Minimum` was selected as an alternative because the actual consumed capacity can be rounded up
  based on the request policy — for example, to match a defined chunk size or meet a minimum requirement.
  However, `Requests` was reselected during API review because it is more align with the container spec and
  matches present semantic definition used elsewhere in the API (minimum guaranteed, must be satisfied).
  with the need of clear description to distinguish between requests for devices and requests
  for resources which must be provided by those devices.

### Future Possibilities

#### RequestPolicy

- The allocation strategy can be introduced for each capacity attribute defined in the `RequestPolicy`.
For example, a `strategy` field could be added to explicitly define the scheduling behavior for a specific capacity:

  ```yaml
  requestPolicy:
    strategy: ...
  ```

  For example,
  - `AlwaysConsumed`: The default behavior. A predefined default value is always applied if no capacity is explicitly requested.
  - `ConsumedOrNever`: If the first consumer specifies a capacity request, that capacity becomes consumable. If not, it remains non-consumable until the first consumer releases it.
  - `BlockOrShare`: The inverse of `ConsumedOrNever`. If the first consumer requests no capacity, it consumes the entire device (i.e., full capacity). If it does specify a capacity request, the device remains multi-allocatable up to the guaranteed amount.

  The current default behavior is `AlwaysConsumed`.

- A common RequestPolicy can be defined in the Device struct (similar to mixins)
  and reference it using new fields named `RequestPolicyRef` or `RequestPolicyName`,
  which are mutually exclusive with the Default field as discussed in [this comment's thread](https://github.com/kubernetes/kubernetes/pull/132522#discussion_r2208705536).

- Defining an inifinite requestPolicy `zeroConsumption` as a mutual exclusive definition to other valid value policies. This flag is equivalent to `{default: 0, validValues{{0}}}`. If the request doesn't contain this capacity entry, zero value is used and Default must not be defined. See [this comment]( https://github.com/kubernetes/kubernetes/pull/132522#discussion_r2229422565) for future discussion.

#### CapacityRequirements

- `Limits` field to describe burstable consumption. 
  Handling burstability would be the responsibility of the individual device driver,
  similar to how the CPU manager handles CPU burst behavior.

## Infrastructure Needed (Optional)

<!--
Use this section if you need things from the project/SIG. Examples include a
new subproject, repos requested, or GitHub details. Listing these here allows a
SIG to get the process for these resources started right away.
-->