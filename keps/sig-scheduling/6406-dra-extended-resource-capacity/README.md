# KEP-6406: DRA: Capacity Requests for Extended Resource Mapping

<!-- toc -->
- [Release Signoff Checklist](#release-signoff-checklist)
- [Summary](#summary)
- [Motivation](#motivation)
  - [Goals](#goals)
  - [Non-Goals](#non-goals)
- [Proposal](#proposal)
  - [User Stories (Optional)](#user-stories-optional)
    - [Story 1: Migrating NVIDIA time-sliced GPUs from device plugin to DRA](#story-1-migrating-nvidia-time-sliced-gpus-from-device-plugin-to-dra)
    - [Story 2: Fixed-size shared NIC bandwidth slices](#story-2-fixed-size-shared-nic-bandwidth-slices)
  - [Notes/Constraints/Caveats](#notesconstraintscaveats)
  - [Risks and Mitigations](#risks-and-mitigations)
- [Design Details](#design-details)
  - [API](#api)
    - [DeviceClassSpec](#deviceclassspec)
    - [Validation](#validation)
    - [Example](#example)
  - [Scheduler changes](#scheduler-changes)
  - [Interaction with existing features](#interaction-with-existing-features)
  - [Test Plan](#test-plan)
      - [Prerequisite testing updates](#prerequisite-testing-updates)
      - [Unit tests](#unit-tests)
      - [Integration tests](#integration-tests)
      - [e2e tests](#e2e-tests)
  - [Graduation Criteria](#graduation-criteria)
    - [Alpha](#alpha)
    - [Beta](#beta)
    - [GA](#ga)
    - [Deprecation](#deprecation)
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

[KEP-5004 (DRA Extended Resource)](/keps/sig-scheduling/5004-dra-extended-resource) lets a `DeviceClass` declare an `extendedResourceName`, so that a container's classic `resources.requests`/`resources.limits` entry (e.g. `nvidia.com/gpu: 1`) is satisfied by devices published through a DRA driver.
The scheduler synthesizes a `ResourceClaim` on the pod's behalf, with one `DeviceRequest` per container per extended resource, requesting `Count` whole devices from the matching `DeviceClass`.

[KEP-5075 (DRA Consumable Capacity)](/keps/sig-scheduling/5075-dra-consumable-capacity) lets any `DeviceRequest` carry a `Capacity` field (a `CapacityRequirements` map of capacity name to requested quantity), so that a device advertising `allowMultipleAllocations: true` is only partially consumed by the allocation instead of being handed out whole.

These two features do not currently compose.
The `DeviceRequest` the scheduler synthesizes for an extended resource never sets `Capacity`, so extended resources can only ever request whole devices.
This KEP adds an optional `extendedResourceCapacityRequests` field to `DeviceClassSpec`.
When set, the scheduler copies its value onto the `Capacity` field of every `ExactDeviceRequest` it synthesizes for that class's `ExtendedResourceName`, reusing the existing `CapacityRequirements` type and the existing per-allocation semantics of `DeviceRequest.Capacity` as-is.
No new allocation algorithm, no new API type, and no change to how a pod expresses its request.
The extended resource name is what changes behavior, exactly as it already does today for which `DeviceClass` gets selected.

This lets a cluster administrator expose a fixed-size "slice" of a shareable device (for example, a fixed number of GPU time-slicing shares) through the classic extended resource API, enabling drop-in migration of extended-resource-based workloads (no pod spec changes) onto DRA-managed, capacity-aware device sharing.

## Motivation

Extended resources remain the simplest way for a workload to request a resource: a name and an integer quantity, with no awareness of `ResourceClaim`s, `DeviceClass`es, or DRA at all.
KEP-5004 preserved that simplicity while letting the devices behind the name be managed by a DRA driver instead of (or alongside) a device plugin.

Separately, DRA drivers are increasingly modeling devices that can be split into shares rather than handed out whole.
The DRA driver for NVIDIA GPUs, for example, already supports advertising a `"shares"` device capacity (a fixed number of time-slicing shares per GPU, gated behind `DRAConsumableCapacity`) as a DRA-native alternative to the older device-plugin-based GPU time-slicing mechanism (`nvidia.com/gpu.shared`, configured through NVIDIA's device plugin `ConfigMap`).
Other DRA drivers model comparable shareable resources: fractions of NIC bandwidth, slices of device memory, and so on.

Today, migrating a workload off device-plugin-based fractional GPU access and onto DRA-based capacity sharing requires rewriting its pod spec to use a `ResourceClaim` or `ResourceClaimTemplate` with an explicit `capacity.requests` block.
That is a reasonable ask for a workload owner who is choosing to adopt DRA directly, but it is friction that KEP-5004 specifically set out to avoid for extended-resource consumers who are not.
KEP-5004's own Goals state that "cluster administrators [can] specify devices advertised by DRA drivers to satisfy extended resource requests" and "application operators [can] use existing extended resource requests in pod's spec to request DRA resources."
Without a capacity mapping on `DeviceClass`, that promise stops at "whole devices only."

This KEP closes that specific, narrow gap.
It lets the fixed capacity amount live where the rest of the extended-resource-to-DRA mapping already lives, on the `DeviceClass`, so an administrator who wants to offer, say, "2-share GPU slices" as `nvidia.com/gpu.shared-2` can do so the same way they already offer whole GPUs as `nvidia.com/gpu`, and existing workloads using the classic resource name keep working unmodified.

### Goals

- Let a `DeviceClass` author declare a fixed capacity request (one or more capacity name/quantity pairs) that the scheduler applies to every `ResourceClaim` it synthesizes for that class's `ExtendedResourceName`.
- Enable migration of existing device-plugin-based fractional/time-sliced device consumption (for example NVIDIA's `nvidia.com/gpu.shared`) onto DRA-managed consumable capacity, with zero changes to consuming pod specs, by having the administrator publish one `DeviceClass` (and one extended resource name) per desired slice size.
- Keep the mechanism generic and driver-agnostic: it works with any capacity name any DRA driver advertises through `Device.Capacity` (GPU shares, NIC bandwidth, device memory, etc.), and places no NVIDIA-specific (or any vendor-specific) concept in the API.
- Reuse the existing `CapacityRequirements` type and existing `DeviceRequest.Capacity` allocation semantics from KEP-5075 unchanged, rather than defining new capacity-request semantics for the extended-resource path.

### Non-Goals

- Letting a pod, at request time, choose or override the capacity amount for an extended resource request.
  The mapping from extended resource name to capacity amount is a fixed, administrator-declared property of the `DeviceClass`, exactly as the mapping from extended resource name to `DeviceClass` itself already is in KEP-5004.
  A workload that needs a different slice size requests a different extended resource name.
- Changing anything about how extended resources map to whole-device requests when `extendedResourceCapacityRequests` is left unset.
  That existing KEP-5004 behavior is unchanged.
- Changing `DeviceRequest.Capacity` semantics, `CapacityRequestPolicy` evaluation, or any other part of the KEP-5075 allocation algorithm.
  This KEP only changes what values reach an existing field.
- Statically validating, at `DeviceClass` admission time, that devices matched by the class's selectors actually advertise the referenced capacity name(s).
  Selectors are evaluated dynamically against whatever `ResourceSlice`s exist at scheduling time, the same reason KEP-5004 does not statically validate that a class's selectors match any devices at all.
  This is handled the same way KEP-5075 already handles an unsatisfiable capacity request: the pod stays unschedulable with an explanatory event.
- Introducing a way to request "any" or "at least N" shares instead of a fixed amount.
  `CapacityRequirements` is a fixed request today, and this KEP does not change that.

## Proposal

### User Stories (Optional)

#### Story 1: Migrating NVIDIA time-sliced GPUs from device plugin to DRA

A cluster today runs the NVIDIA device plugin configured for GPU time-slicing with 4 replicas, so nodes advertise `nvidia.com/gpu.shared: 4` and workloads request e.g. `nvidia.com/gpu.shared: 1` to get one of four time-slices of a GPU.
The cluster administrator wants to move to the DRA driver for NVIDIA GPUs, which supports the same underlying time-slicing through `DRAConsumableCapacity` (a `"shares"` device capacity, configured via `--consumable-shares=4` on the kubelet plugin), without requiring every team's Deployment/Job spec to change.

With this KEP, the administrator publishes:

```yaml
apiVersion: resource.k8s.io/v1
kind: DeviceClass
metadata:
  name: gpu.nvidia.com-shared-1
spec:
  selectors:
  - cel:
      expression: device.driver == "gpu.nvidia.com" && device.capacity["shares"].isGreaterThan(quantity("0"))
  extendedResourceName: nvidia.com/gpu.shared
  extendedResourceCapacityRequests:
    requests:
      shares: "1"
```

Existing workloads that already request `nvidia.com/gpu.shared: 1` continue to work unmodified.
The scheduler now routes them to the `gpu.nvidia.com-shared-1` class, synthesizes a `ResourceClaim` with `Capacity: {requests: {shares: "1"}}`, and the DRA allocation algorithm places them on a GPU with an available share, exactly as the device plugin did, but under DRA's management.

#### Story 2: Fixed-size shared NIC bandwidth slices

A DRA network driver advertises a shareable NIC with a `"bandwidth"` capacity (an `AllowMultipleAllocations` device whose total capacity is, say, `100Gi`).
An administrator wants to offer a classic extended resource, `example.com/nic.5g`, that hands out fixed 5Gi slices to any pod that requests `example.com/nic.5g: 1`, without those pods needing to know DRA exists:

```yaml
apiVersion: resource.k8s.io/v1
kind: DeviceClass
metadata:
  name: nic.example.com-5g
spec:
  selectors:
  - cel:
      expression: device.driver == "nic.example.com"
  extendedResourceName: example.com/nic.5g
  extendedResourceCapacityRequests:
    requests:
      bandwidth: "5Gi"
```

This story exists to demonstrate that the mechanism is not GPU- or NVIDIA-specific: it composes with any driver's capacity model.

### Notes/Constraints/Caveats

- `extendedResourceCapacityRequests` only has an observable effect when `ExtendedResourceName` is also set on the same `DeviceClass`.
  A class with only `extendedResourceCapacityRequests` and no `ExtendedResourceName` is never used to synthesize a `ResourceClaim` in the first place, so the field is simply inert (see [Validation](#validation) for whether this is rejected or merely a no-op).
- The requested capacity amount is fixed per `DeviceClass`/extended-resource-name, and it is applied identically to every unit requested.
  If a container requests `nvidia.com/gpu.shared: 3`, the synthesized `DeviceRequest` has `Count: 3` and `Capacity: {requests: {shares: "1"}}`.
  Per the existing, already-shipped semantics of `DeviceRequest.Capacity` ("Applies to each device allocation. If Count > 1, the request fails if there aren't enough devices that meet the requirements."), this means 3 independent 1-share allocations, not one allocation of 3 shares.
  This is the same semantics KEP-5075 already defines for `Count > 1` combined with `Capacity`.
  This KEP does not change it, but Design Details calls it out since it is easy to misread as "3 shares total."
- As with any extended resource, a pod that requests fractional quantities (e.g. `500m`) will have them rejected the same way KEP-5004 already rejects non-integer extended resource quantities.
  This KEP does not change extended resource quantity handling.

### Risks and Mitigations

- **Administrator misconfiguration is a silent, hard-to-diagnose scheduling failure.** If `extendedResourceCapacityRequests` references a capacity name (e.g. a typo, or a name no device matched by the class's selectors actually advertises), every pod using that extended resource becomes permanently unschedulable.
  This is not a new failure mode introduced by this KEP. KEP-5075 already has the identical risk for hand-written `ResourceClaim`s with a bad `capacity.requests` entry, and it is mitigated the same way.
  The `DynamicResources` scheduler plugin surfaces an explanatory `PodSchedulingUnschedulable` event and status naming the unsatisfied requirement, and existing `kubectl describe pod` and scheduler event tooling covers it.
  We will ensure the event text distinguishes "no matching capacity" from other DRA scheduling failures so administrators debugging an extended-resource migration are not left guessing whether the problem is the selector, the capacity name, or something else.
- **Interaction between `Count` and `Capacity` is unintuitive for extended resources specifically**, since extended resources have historically implied "N separate, identical whole units," and it is easy to misread "3 requested, Capacity: shares=1" as "3 shares total" rather than "3 independent 1-share allocations."
  This is called out in Notes/Constraints/Caveats and Design Details, and will be documented in the user-facing docs for this feature with a worked example, rather than relying on the reader to infer it from KEP-5075.
- **A `DeviceClass` referencing `extendedResourceCapacityRequests` without `DRAConsumableCapacity` enabled would be silently inert** (the field would be persisted but the scheduler could never produce an allocatable request from it, since no device could ever have been marked `AllowMultipleAllocations` without that gate).
  Mitigated by validation requiring `DRAConsumableCapacity` to be enabled whenever `extendedResourceCapacityRequests` is set, matching how KEP-5075 itself gates `Capacity`, `AllowMultipleAllocations`, and `RequestPolicy` (see [Validation](#validation)).
- **Scope creep and narrow-API risk:** `extendedResourceCapacityRequests` is narrow by design. It only ever influences one field (`Capacity`) of the synthesized `ExactDeviceRequest`.
  A fair question is what happens the next time something else about the synthesized request needs to be administrator-controlled, for example `Tolerations`, `AdminAccess`, or additional `Selectors`.
  This KEP does not attempt to answer that generically now.
  See [Alternatives](#alternatives) for two more general designs that were considered, a broader request-shaping mechanism on `DeviceClass` and referencing an existing `ResourceClaimTemplate`, and why both were rejected in favor of staying narrow.
  The expectation is that each future need drives its own small, typed, purpose-built field and KEP, the same way this one does, rather than a shared generic mechanism designed ahead of a second concrete use case.

## Design Details

### API

#### DeviceClassSpec

A new optional field is added to `DeviceClassSpec` (`k8s.io/api/resource/v1`), immediately after the existing `ExtendedResourceName` field (next available protobuf field number, 5, following `Selectors` (1), `Config` (2), the tombstoned `SuitableNodes` (3), and `ExtendedResourceName` (4)):

```go
type DeviceClassSpec struct {
	// ... Selectors, Config, ExtendedResourceName unchanged ...

	// ExtendedResourceCapacityRequests, if set, defines the fixed capacity
	// amounts that the scheduler requests on every ResourceClaim it
	// synthesizes to satisfy ExtendedResourceName.
	//
	// Each key must be the name of a capacity advertised by the devices
	// selected by this class (see Device.Capacity in ResourceSlice), and
	// each value is the fixed amount requested for that capacity. The
	// scheduler copies this value, unmodified, onto the Capacity field of
	// every ExactDeviceRequest it synthesizes for this class, so it has
	// the exact same semantics as DeviceRequest.Capacity: it applies to
	// each device allocation, so a Count greater than one requests that
	// many independent allocations of this capacity amount, not a single
	// allocation of Count times this amount.
	//
	// This field only has an effect together with ExtendedResourceName.
	// It is ignored, and before Beta rejected by validation, if
	// ExtendedResourceName is unset. Devices selected by a class using
	// this field must support multiple allocations
	// (AllowMultipleAllocations) and advertise the referenced capacity
	// names. A class that does not select any such devices will simply
	// never satisfy its extended resource requests, the same way an
	// unsatisfiable selector does today.
	//
	// +optional
	// +featureGate=DRAExtendedResourceCapacity
	ExtendedResourceCapacityRequests *CapacityRequirements `json:"extendedResourceCapacityRequests,omitempty" protobuf:"bytes,5,opt,name=extendedResourceCapacityRequests"`
}
```

`CapacityRequirements` is the existing type introduced by KEP-5075 (`Requests map[QualifiedName]resource.Quantity`), no new type is introduced.

#### Validation

- `extendedResourceCapacityRequests` is only accepted (non-nil) when `DRAExtendedResourceCapacity` is enabled.
  Otherwise it is dropped on write, following the standard feature-gate-drop-on-disable pattern also used by `ExtendedResourceName` and by the KEP-5075 fields.
- At Alpha, setting `extendedResourceCapacityRequests` without also setting `ExtendedResourceName` is permitted but has no effect (documented as inert in the godoc).
  We will revisit whether to make this a hard validation error before Beta based on alpha feedback, since rejecting it outright is easy to add later but hard to remove once relied upon.
- Setting `extendedResourceCapacityRequests` while `DRAConsumableCapacity` is disabled is rejected the same way setting `Capacity` on a hand-written `DeviceRequest` is already rejected today, since the field cannot have any effect without it.
- Key and value validation for the `Requests` map reuses the existing validation already implemented for `CapacityRequirements` and `ExactDeviceRequest.Capacity` (qualified-name format for keys, positive quantities), unchanged.
  No new validation code is introduced for the map's shape.

#### Example

See [Story 1](#story-1-migrating-nvidia-time-sliced-gpus-from-device-plugin-to-dra) and [Story 2](#story-2-fixed-size-shared-nic-bandwidth-slices) above for full worked examples.

### Scheduler changes

Extended-resource-to-`ResourceClaim` synthesis happens in the `DynamicResources` scheduler plugin (`pkg/scheduler/framework/plugins/dynamicresources/extendeddynamicresources.go`).
Today, `createRequestsAndMappings` resolves the matching `DeviceClass` for each extended resource via `DeviceClassResolver.GetDeviceClass(resourceName)`, which already returns the full `*resourceapi.DeviceClass` object and not just its name, and then calls `createResourceRequestAndMappings`, passing along only `class.Name`.
That function builds:

```go
deviceReq := resourceapi.DeviceRequest{
	Name: reqName,
	Exactly: &resourceapi.ExactDeviceRequest{
		DeviceClassName: className,
		AllocationMode:  resourceapi.DeviceAllocationModeExactCount,
		Count:           crq - sum,
	},
}
```

`Capacity` is left unset.
This KEP's implementation threads the resolved class's `Spec.ExtendedResourceCapacityRequests` through to this call site, alongside `className`, and sets it on the `ExactDeviceRequest`:

```go
Exactly: &resourceapi.ExactDeviceRequest{
	DeviceClassName: className,
	AllocationMode:  resourceapi.DeviceAllocationModeExactCount,
	Count:           crq - sum,
	Capacity:        classCapacityRequests, // new: class.Spec.ExtendedResourceCapacityRequests, deep-copied
},
```

This is the only functional change to the scheduler.
No new plugin, no new scheduling phase, no change to `PreFilter`, `Filter`, or `PreBind` staging of the synthesized claim, and no change to the DRA allocation algorithm itself.
The synthesized `DeviceRequest` simply now carries a `Capacity` value the existing KEP-5075 allocation code already knows how to honor.
(This section describes the intended integration point for reviewers' benefit. The actual code changes are out of scope for this KEP document, per [Non-Goals](#non-goals).)

### Interaction with existing features

- **KEP-5004 (DRA Extended Resource):** this KEP is purely additive to it. Any `DeviceClass` that does not set `extendedResourceCapacityRequests` behaves exactly as it does today.
- **KEP-5075 (DRA Consumable Capacity):** this KEP does not change `CapacityRequirements`, `CapacityRequestPolicy`, `Device.Capacity`, or the allocation algorithm. It only changes how one more `DeviceRequest.Capacity` value can originate (from a `DeviceClass`, via the extended-resource synthesis path, instead of only from a hand-written `ResourceClaim`).
- **Partitionable devices (KEP-4815) and other device-modeling KEPs:** orthogonal. This KEP does not care how a device came to advertise a capacity, only that it does.

### Test Plan

[x] I/we understand the owners of the involved components may require updates to
existing tests to make this code solid enough prior to committing the changes necessary
to implement this enhancement.

##### Prerequisite testing updates

None. This KEP builds entirely on the existing `DynamicResources` scheduler plugin test scaffolding introduced for KEP-5004 and KEP-5075.

##### Unit tests

- `k8s.io/api/resource/v1`: round-trip (de)serialization and defaulting tests for the new `ExtendedResourceCapacityRequests` field, mirroring the existing tests for `ExtendedResourceName`.
- `pkg/apis/resource/validation`: new field accepted or rejected correctly depending on `DRAExtendedResourceCapacity` and `DRAConsumableCapacity` gate state, and dropped-on-disable behavior.
- `pkg/scheduler/framework/plugins/dynamicresources`: `createResourceRequestAndMappings`/`createRequestsAndMappings` produce a `DeviceRequest` with the expected `Capacity` value when the resolved class sets `extendedResourceCapacityRequests`, and an unset `Capacity` when it does not (regression coverage for the KEP-5004 behavior staying unchanged).

##### Integration tests

- A pod requesting an extended resource backed by a `DeviceClass` with `extendedResourceCapacityRequests` set is scheduled onto a shareable device, and the resulting synthesized `ResourceClaim`'s allocation reflects the requested capacity amount (`ConsumedCapacity`, `ShareID`).
- Two pods requesting the same extended resource, sized so they fit on the same physical device's remaining capacity but not on two separate devices, are both scheduled onto that one device (proving the capacity-aware path, not just whole-device allocation, is exercised).
- A pod requesting an extended resource whose class's `extendedResourceCapacityRequests` names a capacity no matching device advertises remains unschedulable with an explanatory event, and does not panic or hot-loop the scheduler.
- `Count > 1` behavior: a container requesting quantity 3 of an extended resource backed by a 1-share-per-allocation class results in 3 independent single-share allocations (see [Notes/Constraints/Caveats](#notesconstraintscaveats)).

##### e2e tests

- End-to-end scheduling and pod startup for a workload using a classic extended resource name, backed by a test DRA driver advertising a shareable capacity, verifying the pod runs with the expected fraction of the device's capacity reserved and that a second, appropriately-sized pod can co-locate on the same device.
- Upgrade/downgrade-safe rollout test reusing the pattern already established for `DRAExtendedResource` and `DRAConsumableCapacity`'s own e2e suites.

### Graduation Criteria

#### Alpha

- `ExtendedResourceCapacityRequests` field added to `DeviceClassSpec`, gated by `DRAExtendedResourceCapacity`.
- Scheduler change to populate `Capacity` on synthesized `DeviceRequest`s implemented and covered by unit/integration tests above.
- `DRAExtendedResourceCapacity` requires `DRAConsumableCapacity` to be enabled (validated at the API and documented as a prerequisite).

#### Beta

- Feature gate enabled by default.
- Positive operator/adopter feedback from at least one real DRA driver (in addition to the NVIDIA GPU driver prototype that motivated this KEP) exercising this path in a non-trivial cluster.
- Decide, based on alpha feedback, whether `extendedResourceCapacityRequests` without `ExtendedResourceName` should become a hard validation error (see [Validation](#validation)).
- e2e tests running in CI signal, consistently green.

#### GA

- No changes needed to the API since Beta for at least one release.
- Conformance-relevant behavior (if any is identified during Beta) covered by conformance tests.
- Documentation published on kubernetes.io with a worked migration example (mirroring [Story 1](#story-1-migrating-nvidia-time-sliced-gpus-from-device-plugin-to-dra)).

#### Deprecation

Not applicable. This is a new, additive field, not a replacement for existing behavior.
Standard API deprecation policy would apply if the field were ever deprecated in the future.

### Upgrade / Downgrade Strategy

- **Upgrade:** existing `DeviceClass` objects are unaffected. The new field defaults to unset (nil) and existing extended-resource-backed scheduling behaves exactly as before, so no action is required to upgrade.
- **Downgrade:** on downgrade to a version without this feature, or with the gate disabled, `ExtendedResourceCapacityRequests` is dropped from `DeviceClass` objects on next write, following the same drop-on-disable pattern KEP-5075 already documents for its own fields.
  `DeviceClass` objects that relied on it revert to KEP-5004's whole-device behavior for their extended resource.
  This is a behavior change for affected workloads, since they go back to requesting and consuming whole devices, but not a correctness or safety issue, because it is strictly more conservative: a workload that previously got a device slice now gets an entire device, never less than it asked for.
- Already-allocated `ResourceClaim`s (and their `ConsumedCapacity`/`ShareID` allocation results) are unaffected by this KEP's downgrade path in exactly the way KEP-5075 already documents, since this KEP does not touch `ResourceClaimStatus`.

### Version Skew Strategy

This feature only involves `kube-apiserver` (new field, validation) and `kube-scheduler` (synthesis logic).
It does not require any kubelet or node-level component changes beyond what `DRAExtendedResource` and `DRAConsumableCapacity` already require. Standard version-skew handling applies:

- An older `kube-scheduler` talking to a newer `kube-apiserver` simply never reads `ExtendedResourceCapacityRequests` and continues synthesizing whole-device requests, identical to pre-KEP behavior.
- A newer `kube-scheduler` talking to an older `kube-apiserver` that does not yet support the field will never observe it set (the apiserver will not have accepted or stored it), so it also falls back to existing whole-device synthesis.
- No control-plane component may run more than one skewed version apart per standard Kubernetes support policy, so no additional handling beyond the above is required.

## Production Readiness Review Questionnaire

### Feature Enablement and Rollback

###### How can this feature be enabled / disabled in a live cluster?

- [x] Feature gate (also fill in values in `kep.yaml`)
  - Feature gate name: `DRAExtendedResourceCapacity`
  - Components depending on the feature gate: `kube-apiserver`, `kube-scheduler`

###### Does enabling the feature change any default behavior?

No. `ExtendedResourceCapacityRequests` defaults to unset on every `DeviceClass`, and unset means "synthesize a whole-device request," identical to today's `DRAExtendedResource` behavior. Only `DeviceClass` objects an administrator explicitly opts in, by setting the new field, are affected.

###### Can the feature be disabled once it has been enabled?

Yes. Disabling `DRAExtendedResourceCapacity` causes `ExtendedResourceCapacityRequests` to be dropped from `DeviceClass` objects on next write (standard drop-on-disable), and the scheduler stops populating `Capacity` on synthesized requests, reverting to whole-device requests for the affected extended resource names. See [Upgrade / Downgrade Strategy](#upgrade--downgrade-strategy).

###### What happens if we reenable the feature if it was previously rolled back?

`DeviceClass` objects that had `ExtendedResourceCapacityRequests` dropped need to have it set again explicitly.
The field is not retroactively restored, consistent with standard feature-gate drop-on-disable semantics elsewhere in the API.

###### Are there any tests for feature enablement/disablement?

Yes, planned as part of the unit test coverage described in [Test Plan](#test-plan): field accepted/dropped correctly based on gate state, and scheduler behavior reverting to whole-device synthesis when the gate (or the field) is off.

### Rollout, Upgrade and Rollback Planning

###### How can a rollout or rollback fail?

A rollout could fail the same way any `DeviceClass` misconfiguration fails today: an administrator sets `extendedResourceCapacityRequests` referencing a capacity name no matched device advertises, and pods using that extended resource become unschedulable.
This is a scheduling-time failure, visible via standard pod events, not a control-plane crash or data-loss risk.
Rollback (disabling the gate) can only make behavior more conservative (whole devices instead of slices), never less schedulable in a way that strands running workloads, since already-allocated claims are unaffected.

###### What specific metrics should inform a rollback?

`scheduler_unschedulable_pods{plugin="DynamicResources"}` increasing specifically for pods using an extended resource backed by a class with `extendedResourceCapacityRequests` set would indicate a misconfiguration or unexpected interaction, and is the primary signal to investigate before considering rollback.

###### Were upgrade and rollback tested?

Will be verified as part of Beta graduation criteria, reusing the upgrade/downgrade test pattern already established for `DRAExtendedResource` and `DRAConsumableCapacity`.

###### Is the rollout accompanied by any deprecations and/or removals?

No.

### Monitoring Requirements

###### How can an operator determine if the feature is in use?

By checking whether any `DeviceClass` in the cluster has `spec.extendedResourceCapacityRequests` set (`kubectl get deviceclasses -o json | jq` or similar), and/or by `apiserver_request_total{group="resource.k8s.io", resource="deviceclasses"}` write activity correlated with the field's presence.

###### How can someone using this feature know that it is working?

Pods using the mapped extended resource are scheduled and, once running, the underlying device driver reports the expected partial capacity consumption (for the NVIDIA GPU driver, via its existing status/telemetry for consumable shares). At the API level, the synthesized `ResourceClaim`'s `status.allocation` will show a `ConsumedCapacity` value matching what was declared on the `DeviceClass`.

###### What are the reasonable SLOs?

No new SLOs beyond those already defined for `DRAExtendedResource` and `DRAConsumableCapacity` scheduling latency (`scheduler_plugin_execution_duration_seconds{plugin="DynamicResources"}`).
This KEP does not add a new scheduling phase or algorithm, only a value carried through an existing one.

###### What are the SLIs?

Reuses the existing `DynamicResources` plugin SLIs. No new SLIs are introduced by this KEP.

###### Are there any missing metrics?

None identified beyond what KEP-5004 and KEP-5075 already expose.
This KEP does not introduce new failure modes distinct enough to warrant dedicated metrics at Alpha.
This will be revisited at Beta if operator feedback shows the existing metrics are insufficient to distinguish this feature's failures from other DRA scheduling failures.

### Dependencies

###### Does this feature depend on any specific services running in the cluster?

It depends on `DRAExtendedResource` (GA since v1.37) and `DRAConsumableCapacity` (Beta since v1.36) both being enabled, and on at least one DRA driver in the cluster advertising `AllowMultipleAllocations` devices with the referenced capacity name(s). Without a driver implementing consumable capacity, the field can be set but will never be satisfiable.

### Scalability

###### Will enabling / using this feature result in any new API calls?

No new API call patterns. `DeviceClass` writes are unchanged in frequency, administrator-driven and infrequent.
Synthesized `ResourceClaim` creation during scheduling is unchanged in frequency from KEP-5004, one per container per extended resource same as today, only its content differs by one additional populated field.

###### Will enabling / using this feature result in introducing new API types?

No. `ExtendedResourceCapacityRequests` reuses the existing `CapacityRequirements` type from KEP-5075.

###### Will enabling / using this feature result in any new calls to the cloud provider?

No.

###### Will enabling / using this feature result in increasing size or count of existing API objects?

`DeviceClass` objects that opt in grow by the size of one `CapacityRequirements` map entry per referenced capacity name (comparable to the per-device growth KEP-5075 already documented as small, on the order of tens to low hundreds of bytes). Synthesized `ResourceClaim`s gain one additional populated field (`Capacity`) they did not have before, which is already accounted for in KEP-5075's own object-size analysis for `DeviceRequest.Capacity`.

###### Will enabling / using this feature result in increasing time taken by operations?

Negligible. The scheduler change is copying an already-resolved, in-memory value onto a struct field it already constructs.
It does not add a lookup, an API call, or additional CEL evaluation to the synthesis path.

###### Will enabling / using this feature result in non-negligible increase of resource usage?

No.

###### Can enabling / using this feature result in resource exhaustion?

No new exhaustion vector beyond what KEP-5075 already analyzed for `Capacity`-bearing `DeviceRequest`s in general.
This KEP does not change how many claims or requests can be created, only what one existing field on already-created requests may contain.

### Troubleshooting

###### How does this feature react if the API server and/or etcd is unavailable?

No different from any other DRA scheduling activity: the scheduler cannot read `DeviceClass`/`ResourceSlice` objects or create `ResourceClaim`s, and affected pods remain pending, exactly as they would for any other apiserver-unavailability scenario.

###### What are other known failure modes?

- **Referenced capacity name not advertised by any matched device:** pods using the affected extended resource are permanently unschedulable until the `DeviceClass` or the driver's capacity advertisement is corrected. Detection: `scheduler_unschedulable_pods{plugin="DynamicResources"}` plus the pod's scheduling event. Mitigation: correct the `DeviceClass`'s `extendedResourceCapacityRequests` or selectors.
- **`DRAConsumableCapacity` disabled while `DRAExtendedResourceCapacity` is enabled:** rejected at validation time (see [Validation](#validation)), so this surfaces immediately as an API write error on the `DeviceClass`, not as a silent scheduling failure later.

###### What steps should be taken if SLOs are not being met?

Same triage path as any other `DynamicResources` scheduling latency regression: check `scheduler_plugin_execution_duration_seconds{plugin="DynamicResources"}`, and since this KEP adds no new computation to the hot path (a single field copy), a regression localized to this feature would point at an issue in the underlying KEP-5075 capacity-aware allocation algorithm rather than in this KEP's own, minimal addition.

## Implementation History

- 2026-09-21: Initial KEP drafted.

## Drawbacks

- Adds one more optional field to an already fairly dense `DeviceClassSpec`, and one more thing for a `DeviceClass` author to understand (the `Count`-versus-total-capacity distinction in particular is a plausible source of confusion, see [Notes/Constraints/Caveats](#notesconstraintscaveats)).
- Ties the usefulness of this feature to adoption of `DRAConsumableCapacity` by individual DRA drivers. Until a driver implements consumable capacity, this field has nothing to do.
  This is an inherent property of building on top of KEP-5075 rather than a drawback specific to this KEP's design.
- A fixed, class-level capacity mapping cannot express "give me whatever capacity is left" or per-pod-tunable slice sizes.
  Workloads that need that flexibility still need to use `ResourceClaim` or `ResourceClaimTemplate` directly, as they do today.
  This KEP does not attempt to close that gap (see [Non-Goals](#non-goals)).

## Alternatives

- **Let the extended resource *quantity* itself encode the capacity amount** (e.g. `nvidia.com/gpu.shared: 2` meaning "2 shares of one device," rather than "2 independent 1-share devices"). Rejected: this would special-case extended-resource quantity semantics in a way that diverges from every other extended resource in Kubernetes (where quantity has always meant "how many," not "how much of one"), and it would not compose with `Count`-based allocation the way the rest of DRA already does. Keeping `Count` meaning "how many independent allocations" and `extendedResourceCapacityRequests` meaning "how big is each one" keeps this KEP's addition consistent with both KEP-5004 and KEP-5075's existing semantics, at the cost of the naming subtlety called out in Risks and Mitigations.
- **Put the capacity mapping on the pod/container instead of the `DeviceClass`** (e.g. a pod annotation or a new Pod API field requesting a specific share count alongside the extended resource). Rejected: this reintroduces exactly the DRA-awareness-in-the-pod-spec problem that KEP-5004 exists to avoid, and duplicates functionality `ResourceClaim`/`ResourceClaimTemplate` already provide for workloads willing to be DRA-aware.
- **Encode the slice size in the extended resource name itself via a naming convention** (e.g. always requiring names like `nvidia.com/gpu.shared.2` and having the scheduler parse the suffix), instead of an explicit field. Rejected: this invents an implicit, driver-specific naming grammar that the API would have to parse and validate, is not discoverable from the `DeviceClass` object itself, and does not generalize to non-numeric capacity units (like `5Gi` of bandwidth in [Story 2](#story-2-fixed-size-shared-nic-bandwidth-slices)). An explicit typed field is more consistent with how the rest of the `resource.k8s.io` API is designed.
- **Do nothing. Require DRA-aware workloads for capacity sharing.** Rejected as the status quo this KEP addresses. It works for new workloads, but leaves no low-friction path for the substantial number of existing extended-resource-based GPU-sharing workloads referenced in the Motivation.
- **Generalize `DeviceClassSpec` into a broader request-shaping mechanism** that could influence any field of the synthesized `ExactDeviceRequest`, not just `Capacity`, anticipating future needs beyond this KEP.
  Rejected for now. With only one concrete need identified (capacity), a generic mechanism would be designed against a single example and would likely not generalize correctly to whatever the next need turns out to be.
  A mechanism wide enough to plausibly cover "whatever might be needed later" converges on mirroring most of `ExactDeviceRequest` inside `DeviceClassSpec`, which reintroduces the DeviceClass-as-ResourceClaim-template outcome this KEP specifically avoids.
  `DeviceClassSpec.Config` already exists as the generic, driver-opaque extensibility point for data the scheduler does not need to interpret.
  Fields the scheduler must understand and act on at allocation time, like `Capacity`, are a different category, and are better served by their own explicit, reviewed field each time, as KEP-5075 did for `Capacity` itself and as this KEP proposes to do again.
- **Let `DeviceClass` reference an existing `ResourceClaimTemplate`** and reuse its full expressiveness (arbitrary `Devices.Requests`, `Capacity`, `Tolerations`, `Constraints`, etc.) for extended-resource synthesis, instead of adding new fields to `DeviceClassSpec` piecemeal.
  Rejected. `DeviceClass` is cluster-scoped by design, an administrator-managed, cluster-wide policy object, while `ResourceClaim` and `ResourceClaimTemplate` are namespace-scoped.
  A cluster-scoped `DeviceClass` cannot reference a single namespaced `ResourceClaimTemplate` as *the* template for every namespace's pods without either picking one namespace's template arbitrarily for the whole cluster (a cross-tenant leak), or requiring the administrator to replicate an identical template into every namespace.
  The latter reintroduces the multi-object synchronization problem a single cluster-scoped field is meant to avoid, and reopens the tenancy and RBAC boundary that keeps the extended-resource mapping fully administrator-controlled and workload-transparent, which is core to KEP-5004.
  Resolving that scope mismatch would need its own mechanism, for example a new cluster-scoped template type, which is out of scope for the narrow problem this KEP addresses.

## Infrastructure Needed (Optional)

None beyond standard `kubernetes/kubernetes` CI lanes already exercising `DRAExtendedResource` and `DRAConsumableCapacity`.
This KEP's e2e tests extend those existing suites rather than requiring new test infrastructure.
