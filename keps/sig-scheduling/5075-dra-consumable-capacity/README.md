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
    - [Story 3](#story-3)
  - [Risks and Mitigations](#risks-and-mitigations)
- [Design Details](#design-details)
  - [API enhancement](#api-enhancement)
    - [ResourceSliceSpec's Device](#resourceslicespecs-device)
    - [ResourceClaimSpec's DeviceRequest](#resourceclaimspecs-devicerequest)
    - [ResourceClaimStatus's DeviceRequestAllocationResult](#resourceclaimstatuss-devicerequestallocationresult)
  - [Scheduling enhancement](#scheduling-enhancement)
  - [Examples](#examples)
    - [Shareable DeviceClass's selector](#shareable-deviceclasss-selector)
    - [ResourceClaim with capacity requirement](#resourceclaim-with-capacity-requirement)
    - [ResourceClaim's status](#resourceclaims-status)
    - [ResourceClaim with distinctAttribute](#resourceclaim-with-distinctattribute)
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
  - [Selecting/Deselecting Shareable Devices](#selectingdeselecting-shareable-devices)
  - [Preventing Same Shareable Device from Being Allocated Multiple Times in the Same Claim](#preventing-same-shareable-device-from-being-allocated-multiple-times-in-the-same-claim)
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
With this KEP, independent resource claims (and/or requests within a claim) can allocate shares of the same underlying device.
This enables resource sharing across pods that are completely unrelated, potentially even across different namespaces.

Additionally, if a device supports sharing, its resource (capacity) can be managed through a defined sharing policy.
When such a policy is specified, the device may be allocated to multiple independent requests, up to its total capacity,
with the platform enforcing the policy and managing allocations on each request accordingly.
In contrast, if no sharing policy is defined, the device is treated as freely shareable and not dedicated to any specific request.
As a result, the resource without a sharing policy imposes no constraints on how new requests are processed.
Notably, each of these independent resource claims can still be referenced by one or more pods.
However, the device resources allocated to each request are shared without any isolation guarantees among the pods that reference the same request.

To achieve this, this KEP introduces
- a new device field to distinguish between “shareable” and “unshareable” devices,
- a capacity-aware scheduling mechanism that allows limiting or guaranteeing the shareable device capacity,
- a new capacity requirement field in the device request of the resource claim,
- a new consumed capacity field in the allocation result of the resource claim,
- a distinct attribute constraint to prevent allocating the same shareable device in the same claim multiple times.

Relations to other KEPs:
- [KEP 4815](https://github.com/kubernetes/enhancements/issues/4815): The partitioned devices can be a shareable device or have mutually exclusive partitions where one partition is shareable and the other is not.
- [KEP 5007](https://github.com/kubernetes/enhancements/issues/5007): The allocated share can be provisioned at the pre-bind step.
- [KEP 4817](https://github.com/kubernetes/enhancements/issues/4817): A single network device can be shared across multiple pods, with each allocated share's `NetworkData` identified by a unique Share UID.

A motivating use case is to allocate a shareable network device in the [CNI DRA driver](https://github.com/kubernetes-sigs/cni-dra-driver)
which can be selected by more than one pod on demand during scheduling.
The original discussion is in [this PR's comment thread](https://github.com/kubernetes-sigs/cni-dra-driver/pull/1#discussion_r1889265214). 
The limitation of current implementation has been addressed [here](https://github.com/kubernetes-sigs/cni-dra-driver/pull/1#discussion_r1890166449).
The virtual network device is created and configured once the CNI is called based on the information of the master network device. 
The configured information specific to the generated device cannot be listed in the ResourceSlice in advance. 

This feature is also beneficial for the other shareable devices which are not within scope of [KEP-4815](https://github.com/kubernetes/enhancements/issues/4815).
For instance, this feature will be allow reserving memory fraction of virtual GPU in [the AWS virtual GPU device plugin](https://github.com/awslabs/aws-virtual-gpu-device-plugin).
In other words, the device capacity allocation is determined by the user's claim. 

### Goals

- Introduce an ability to allocate a shareable device via DRA multiple times
  in scenarios where pre-defined partitions are not viable, for example because there would be too many of them.
- Let users specify in device requests how much of certain device resources they require.

### Non-Goals

- Define driver-specific attributes and configs (such as CNI parameter config).
- Support network security policy.
- Support an aggregated resource consumption request. 
  By default, the shareable device can be allocated once for each pod's allocation.
  However, a user may want an aggregated amount of resources which can come from a single or multiple shareable device.
  This is related to [the comment about `distinctAttributes`](https://github.com/kubernetes/enhancements/pull/5104#discussion_r1943835445).

## Proposal

### User Stories (Optional)

#### Story 1

A DRA driver for networks advertises shareable devices for two interfaces eth1 and eth2
which each connect to the same virtual LAN, 
the admin makes those device available through a DeviceClass selecting only shareable devices,
and users request access through a request which references that DeviceClass.

#### Story 2

When requesting two interfaces, the user requests two devices. To ensure that they don't end up with the same shareable device for each request, they specify that the driver-specific "interfaceName" attribute must be different.

#### Story 3

A DRA driver for networks supports QoS guaranteed bandwidth which can ensures a specific bandwidth amount of the shareable network can be reserved exclusively to resource requests. A DRA driver also specifies minimum, maximum amount of reserved capacity for each resource request.
When requesting the guaranteed network device, 
users specifies their required guaranteed bandwidth. Otherwise, the default value defined by the DRA driver is applied.

### Risks and Mitigations

- The requested amount in the resource claim may not satisfy the capacity sharing policy,
  especially if the requested amount exceeds the maximum allowed consumption.

  This scenario should be handled similarly to other scheduling issues, such as when the request exceeds the allocatable capacity. 
  In such cases, the allocation fails and the pod remains pending.

- The driver includes both shareable and unshareable devices. There is a risk that a user may be allocated a shareable device
  (e.g., a shareable network device) and accidentally configure it with the HostDevice CNI plugin.
  This would move the device from the host into the user's pod, preventing other users from accessing the shareable device.

  **Mitigation:**
  - Administrators should define clear device classes for shareable and unshareable devices to prevent such misallocations.

## Design Details

This enhancement introduces a `shareable` field within the `Device` of the ResourceSlice
to mark whether the device is a shareable device.
The shareable device can be assigned to more than one request if it satisfies the selection criteria and constraints.
The select condition `device.shareable == true/false` is used to identify to select the device with a `shareable` property or not.

The enhancement also adds a `SharingPolicy` field to `DeviceCapacity`.
This field specifies how the capacity can be shareable between different requests.
The sharing policy can either specify a range of valid values or a discrete set of them.
Each policy has a default value, either explicitly or implicitly.

Users can define specific per-device resource requests through the newly added `CapacityRequests` field in the `DeviceRequest`. Each contains a minimum required device capacity.
If the capacity is consumable, the amount available for allocation is determined
by subtracting the aggregated allocation results of current claims from the device's capacity as defined in the resource slice.
The remaining amount will be used solely by the allocator and will not be reflected in the resource slice.
The calculation of capacity requirements will round the requested capacity up to the nearest valid amount,
based on the capacity's sharing policy.

If users do not specify a capacity request for consumable capacity, the default consumed value will be applied.
There is always such a default because capacities without a policy are not consumable.

A shareable device can only be allocated once its consumability has been verified
and its attributes match the request's selectors and constraints.
The newly added `consumedCapacities` field in the `DeviceRequestAllocationResult` will be set to the calculated capacity upon a successful allocation.
This value may differ from the originally requested amount, as it is rounded up to the nearest valid value based on the device’s sharing policy.
If no specific amount is requested, a default consumption value is applied.

### API enhancement
To enable this enhancement, the following API updates are proposed.

#### ResourceSliceSpec's Device

```go

// Device represents one individual hardware instance that can be selected based
// on its attributes. Besides the name, exactly one field must be set.
type Device struct {
...
   // Shareable marks whether the device is shareable.
   //
   // A device with shareable="true" can be allocated more than once,
   // and its capacity is shared, regardless of whether the CapacitySharingPolicy is defined or not.
   //
   // +optional
   // +featureGate=DRAConsumableCapacity
   Shareable *bool
}

// DeviceCapacity describes a quantity associated with a device.
type DeviceCapacity struct {
    // Value defines how much of a certain device capacity is available.
    //
    // If the capacity is consumable (i.e., a ClaimPolicy is specified),
    // the consumed amount is deducted and cached in memory by the scheduler.
    // Note that the remaining capacity is not reflected in the resource slice.
    //
    // +required
    Value resource.Quantity

   // SharingPolicy specifies that this device capacity must be consumed
   // by each resource claim according to the defined sharing policy.
   // The Device must be shareable.
   //
   // +optional
   // +featureGate=DRAConsumableCapacity
   SharingPolicy *CapacitySharingPolicy
}

// CapacitySharingPolicy defines how requests consume the available capacity.
// The sharing policy can either specify a range of valid values or a discrete set of them.
// Each policy has a default value, either explicitly or implicitly.
// Exactly one of the consumption policies must be defined.
type CapacitySharingPolicy struct {
   // DiscreteValues defines a set of acceptable quantity values in consuming requests.
   //
   // +optional
   // +oneOf=SharingPolicy
   DiscreteValues *CapacitySharingPolicyDiscrete


   // ValueRange defines an acceptable quantity value range in consuming requests.
   //
   // +optional
   // +oneOf=SharingPolicy
   ValueRange *CapacitySharingPolicyRange
}

// CapacitySharingPolicyDiscrete defines a set of discrete allowed capacity values.
// If Options is not provided, only default value is valid.
//
// - If the requested amount is not listed in the options, it is rounded up to the next higher valid value.
// - If the requested amount exceeds the maximum value in the available options, the request does not satisfy the policy,
//   and the device cannot be allocated.
type CapacitySharingPolicyDiscrete struct {
    // Default specifies the default capacity to be used for a consumption request
    // if no value is explicitly provided.
    //
    // +required
    Default resource.Quantity

    // Options defines a list of additional valid capacity values that can be requested.
    // The Default must not be one of the Options.
    //
    // +optional
    // +listType=atomic
    Options []resource.Quantity
}


// CapacitySharingPolicyRange defines a valid range for consumable capacity values.
//
// - If the requested amount is less than Minimum, it is rounded up to the Minimum value.
// - If the requested amount is between Minimum and Maximum, ChunkSize is set,
//   and the amount does not align with the ChunkSize,
//   it will be rounded up to the next value matching Minimum + (n * ChunkSize).
// - If the requested or rounded amount exceeds Maximum (if set), the request does not satisfy the policy,
//   and the device cannot be allocated.
//
// - If ChunkSize is not set, the requested amount is used as-is, provided it falls within the range.
type CapacitySharingPolicyRange struct {
    // Minimum specifies the minimum capacity allowed for a consumption request.
    // This also acts as the default if no value is specified.
    //
    // +required
    Minimum resource.Quantity

    // Maximum defines the upper limit for capacity that can be requested.
    //
    // +optional
    Maximum *resource.Quantity

    // ChunkSize defines the step size between valid capacity amounts within the range.
    // If set, requested amounts are rounded up to the nearest multiple of ChunkSize from the Minimum.
    // Maximum and Minimum must be a multiple of ChunkSize.
    //
    // +optional
    ChunkSize *resource.Quantity
}
```

#### ResourceClaimSpec's DeviceRequest

```go

type DeviceRequest struct {
...
   // CapacityRequests define resource requirements against each capacity.
   //
   // +optional
   // +featureGate=DRAConsumableCapacity
   CapacityRequests *CapacityRequirements
}


// CapacityRequirements defines the capacity requirements for a specific device request.
type CapacityRequirements struct {
    // Minimum defines the minimum amount of each device capacity required for the request.
    // Each minimum amount must be a non-negative integer.
    //
    // If the capacity has a sharing policy, this value is rounded up to the nearest valid amount
    // according to that policy. The rounded value is used during scheduling to determine how much capacity to consume.
    //
    // If the quantity does not have a sharing policy, this value is used as an additional filtering
    // condition against the available capacity on the device.
    // This is semantically equivalent to a CEL selector with
    // `device.capacity[<domain>].<name>.compareTo(quantity(<minimum quantity>)) >= 0`
    // For example, device.capacity['test-driver.cdi.k8s.io'].counters.compareTo(quantity('2')) >= 0
    //
    // +optional
   Minimum map[QualifiedName]resource.Quantity
}

// DeviceConstraint must have exactly one field set besides Requests.
type DeviceConstraint struct {
	...
   // DistinctAttribute requires that all devices in question have this
   // attribute and that its type and value are unique across those devices.
   //
   // This acts as the inverse of MatchAttribute.
   //
   // This constraint is used to avoid allocating multiple requests to the same shareable device
   // by ensuring attribute-level differentiation.
   //
   // This is useful for scenarios where resource requests must be fulfilled by separate physical devices.
   // For example, a container requests two network interfaces that must be allocated from two different physical NICs.
   //
   // +optional
   // +oneOf=ConstraintType
   DistinctAttribute *FullyQualifiedName
}

```

#### ResourceClaimStatus's DeviceRequestAllocationResult

```go
type ResourceClaimStatus struct {
  ...
	// +optional
	// +listType=map
	// +listMapKey=driver
	// +listMapKey=device
	// +listMapKey=pool
	// +listMapKey=shareUID
	// +featureGate=DRAResourceClaimDeviceStatus
	Devices []AllocatedDeviceStatus `json:"devices,omitempty" protobuf:"bytes,4,opt,name=devices"`
}

type DeviceRequestAllocationResult struct {
  ...

   // ShareUID uniquely identifies the specific allocation result of the shareable device.
   // Set only when allocation is on a shareable device.
   //
   // +optional
   // +featureGate=DRAConsumableCapacity
   ShareUID *types.UID

  // Alternatively, SharedAllocationIndex could have been used as a reference to the allocation result.
  // However, the index may become outdated if the allocation is reallocated (should that ever be supported),
  // making it less reliable than ShareUID.

  // ConsumedCapacities tracks the amount of capacity consumed per device as part of the claim request.
  // The consumed amount may differ from the requested amount: it is rounded up to the nearest valid
  // value based on the device’s sharing policy.
  //
  // The total consumed capacity for each device must not exceed its available capacity.
  //
  // This field references only consumable capacities of a shareable device and is empty when there are none.
  //
  // +optional
  // +featureGate=DRAConsumableCapacity
   ConsumedCapacities map[QualifiedName]resource.Quantity
}
```

### Scheduling enhancement
- When the scheduler invokes the `Allocate` function in the allocator, 
  the total allocated capacity is calculated by aggregating the ConsumedCapacities from all resource claims's `DeviceRequestAllocationResult` that have already been allocated.
- Before allocation proceeds, existing selection criteria (defined by `alloc.isSelectable`) are evaluated. 
  These include the class selector and request selector.
- A new `device.shareable` key is introduced in the CEL selector,
  enabling policies and constraints to recognize whether a device supports shareable allocation.
- If a device is considered selectable, the `CmpRequestOverCapacity` function is invoked to verify 
  whether the consumed capacity would exceed the device's remaining capacity. 
  The remaining capacity is calculated based on the sum of already allocated and currently allocating capacities.
  - consumed capacity is derived from the requested amount specified in the resource claim, adjusted by the device’s capacity sharing policy, if defined.
  - This value may differ from the originally requested amount—it is rounded up to the nearest valid capacity according to the policy (e.g., using Minimum + ⌈(Requested - Minimum)/ChunkSize⌉ × ChunkSize logic).
- If the device has enough remaining capacity to satisfy the consumed amount, constraint checks are applied. 
  In addition to the existing MatchAttribute, this proposal introduces a new constraint: `DistinctAttribute`, which ensures attribute uniqueness across allocated devices.
- Once all selection and constraint checks pass, the allocation is valid. The allocation result is updated with:
  - The shareable allocation identifier, which uniquely identifies the allocation on a shareable device.
  - The calculated consumed capacity, if a capacity sharing policy was applied.
    This consumed capacity is tracked as part of the device’s `allocatingCapacity`, 
    allowing it to be included in remaining capacity calculations for future allocations within the same call.
- Finally, the shareable allocation identifiers and consumed capacities from all internal results
  are propagated to the DeviceRequestAllocationResult.

### Examples

#### Shareable DeviceClass's selector

```yaml
selectors:
  - cel:
      expression: |-
        device.shareable == true
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
      shareable: true
      attributes:
        name:
          string: "eth1"
      capacity:
        bandwidth:
          sharingPolicy:
            range:
              minimum: "1Mi"
              chunkSize: "8"
          value: "10Gi"
```

#### ResourceClaim's request

```yaml
kind: ResourceClaim
...
spec:
  devices:
    requests:
    - name: nic
      deviceClassName: qos-aware-shared.device.x-k8s.io
      capacityRequests:
        minimum:
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
      - consumedCapacities:
          bandwidth: 1Mi
        device: eth1
        shareUID: c9b1a7d2-45e4-4a2e-b8e9-9a3c6b8c1f23
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
      deviceClassName: simple-shareable.networking.x-k8s.io
      allocationMode: ExactCount
      count: 1
    - name: macvlan-2
      deviceClassName: simple-shareable.networking.x-k8s.io
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

  |Test case|Shareable|Device(s) Capacity|Allocated Capacity|Shareable Device Class|Claim request(s)|Expected success
  |---|---|---|---|---|---|---|
  |shareable-device-with-consumable-capacity|yes|2 consumable|0|yes|1+1|yes
  |shareable-device-with-exceeded-consumable-capacity-request|yes|2 consumable|0|yes|1+2|no
  |shareable-device-with-some-remaining-consumable-capacity|yes|2 consumable|1|yes|1|yes
  |shareable-device-with-no-available-consumable-capacity|yes|1 consumable|1|yes|1|no
  |shareable-device-with-unconsumable-capacity|yes|1 unconsumable|0|yes|1+1|yes
  |unshareable-device-with-single-consumable-capacity-request|no|1 consumable|0|yes|1|yes
  |unshareable-device-with-multiple-consumable-capacity-request|no|1 consumable|0|yes|1+1|no
  |exclude-unshareable-device-from-class-selector|yes|0|0|no|0|no
  |one-shareable-device-with-distinct-constraint|yes|2 consumable|0|yes|1+1, distinct|no
  |two-shareable-devices-with-distinct-constraint|yes|2x1 consumable|0|yes|1+1, distinct|yes

  [Test Implementation](https://github.com/sunya-ch/kubernetes/blob/kep-5075/staging/src/k8s.io/dynamic-resource-allocation/structured/allocator_test.go)

- `ListAllAllocatedCapacity` unit test to get AllocatedCapacityCollection without unshareable devices.

- Combintation with partitionable devices

- <package>: <date> - <current test coverage>

##### Integration tests

- Add test user story 1 to 6 in in `scheduler_perf/dra.go` when defining a shareable device with and without consumable capacity.

- Ensure integration with [KEP 4817](https://github.com/kubernetes/enhancements/issues/4817) that server-side-apply works with the additional map key (test/integration/dra)

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

-->

### Upgrade / Downgrade Strategy

In the context of this enhancement, the following strategy is proposed:

* **All introduced fields are optional and can be omitted if empty.** This means that during the upgrade or downgrade process, if certain fields or configurations are not required, they can be left out without causing issues or disrupting the upgrade process.

* **The introduced mechanisms will only be applied if the "shareable" field of the device is not set.**
  This ensures that the feature only activates when specific conditions are met, providing flexibility in how the feature is applied.

  If the "shareable" field is not set, the scheduling mechanisms related to the shared device (e.g., allocating network resources, managing devices) will be triggered according to the introduced enhancement.

* **The upgrade and downgrade processes will follow the DRA strategy.**

### Version Skew Strategy

During version skew, where the API server supports the feature but the scheduler does not, the introduced field can be set, but it will be ignored. In this case, all devices will be treated as non-shareable, and all capacities will be considered non-consumable. No errors or warnings will be triggered, but the field will have no effect.

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

The enablement and disablement of this feature are tested as part of the integration tests. 
Additionally, the feature enablement/disablement tests cover the scenario where the feature gate is switched 
from enabled to disabled after a shared allocation has already been made. 
In this case, the existing resource claim should remain valid, 
but the remaining device capacity must no longer be shareable.

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

### Identifying Shareable Property of Device

**Current Approach:**

Use a **boolean** field to indicate whether a device can be shared.

**Alternative:**

Use an **enum**, such as `DeviceClaimMode`, with defined values like:

- `DeviceClaimModeOnce` — device can only be claimed once  
- `DeviceClaimModeMany` — device can be claimed multiple times

Pros:
- Provides flexibility for future extension according to [Kubernetes API conventions](https://github.com/kubernetes/community/blob/master/contributors/devel/sig-architecture/api-conventions.md#primitive-types).

Cons:
- Increases the program’s memory footprint 
compared to a boolean when there is only a single binary option to serve the purpose.

### Selecting/Deselecting Shareable Devices

**Current Approach:**

Extend the CEL selector to recognize device.shareable for filtering shareable devices.

**Alternative:**

Introduce explicit flags in the resource request:

- AllowShared: Opt-in to allow shareable devices.
- RequireShared: Only allow shareable devices.

(Default: shareable devices are excluded unless explicitly allowed.)

Pros:
- Does not affect unshareable device selection.
- Easier for users to understand and configure, reducing the risk of mistakes.
- More user-friendly than writing CEL expressions manually.

Cons:
- Adds complexity to the allocation logic for shareable devices.
- Introduces an additional field in resource requests.
- May require an abstraction layer if more device features are added in the future.
- Less explicit and expressive than CEL for advanced use cases.

### Preventing Same Shareable Device from Being Allocated Multiple Times in the Same Claim

**Current Approach:**

Introduce a new API-level constraint: DistinctAttribute, ensuring devices in a single claim have unique attribute values.

**Alternative:**

The scheduler enforces this behavior implicitly—never allocate the same shareable device multiple times to the same resource claim.

Pros:
- Avoids any API changes—logic handled internally.

Cons:
- Doesn’t support cases where a pod legitimately permits multiple fractions of capacity from the same shareable device.
  For example, when a pod uses two vGPUs for parallel processing,
  it may not require them to come from different devices.
  It can accept allocations from either the same or different shareable devices.
- Not configurable—users can't override this behavior when needed.

## Infrastructure Needed (Optional)

<!--
Use this section if you need things from the project/SIG. Examples include a
new subproject, repos requested, or GitHub details. Listing these here allows a
SIG to get the process for these resources started right away.
-->