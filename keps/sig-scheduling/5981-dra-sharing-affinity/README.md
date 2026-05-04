# KEP-5981: DRA Sharing Affinity for Conditional Fungibility

<!-- toc -->
- [Release Signoff Checklist](#release-signoff-checklist)
- [Summary](#summary)
- [Motivation](#motivation)
  - [Status Quo: Driver-Side Placeholder Pattern](#status-quo-driver-side-placeholder-pattern)
  - [Goals](#goals)
  - [Non-Goals](#non-goals)
- [Proposal](#proposal)
  - [User Stories](#user-stories)
    - [Story 1: RDMA Partition Key Alignment](#story-1-rdma-partition-key-alignment)
    - [Story 2: FPGA Bitstream Sharing](#story-2-fpga-bitstream-sharing)
    - [Story 3: Single-subnet NIC Sharing](#story-3-single-subnet-nic-sharing)
  - [Notes/Constraints/Caveats](#notesconstraintscaveats)
  - [Risks and Mitigations](#risks-and-mitigations)
- [Design Details](#design-details)
  - [API Enhancement](#api-enhancement)
    - [ResourceSlice Device Spec](#resourceslice-device-spec)
    - [Scheduler Enhancement](#scheduler-enhancement)
  - [Examples](#examples)
    - [ResourceSlice with Sharing Affinity](#resourceslice-with-sharing-affinity)
    - [ResourceClaim with Affinity Value](#resourceclaim-with-affinity-value)
    - [Multi-key SharingAffinity Example](#multi-key-sharingaffinity-example)
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
  - [Claim-side SharingAffinity (on DeviceRequest)](#claim-side-sharingaffinity-on-devicerequest)
  - [Object Reference-based Affinity Matching](#object-reference-based-affinity-matching)
  - [CEL-based Affinity Matching](#cel-based-affinity-matching)
- [Future Enhancements](#future-enhancements)
- [Infrastructure Needed](#infrastructure-needed)
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

This KEP proposes an extension to Dynamic Resource Allocation (DRA) that allows
the `kube-scheduler` to handle resources that are **conditionally fungible**.

[KEP-5075 (Consumable Capacity)](https://github.com/kubernetes/enhancements/issues/5075)
introduced the ability to track numerical capacity (e.g., 16 slots of a NIC)
and share devices across multiple claims via `allowMultipleAllocations`.
However, it assumes all claims are fungible—any claim can share the device with
any other claim.

Real-world hardware is often **modal** (i.e., once partially allocated, it
must operate in a single configuration mode for all of its current
consumers): the device requires all subsequent consumers to share a specific
configuration. For example:

- **Multi-pod NIC sharing**: A network DRA driver shares a NIC across 16 pods,
  but all pods must belong to the same subnet. Once the first pod configures the
  NIC for Subnet A, the remaining 15 slots are restricted to Subnet A.
- **FPGA bitstream sharing**: An FPGA can serve multiple inference pods, but all
  must use the same bitstream. Once bitstream-ml-v2 is loaded, other pods
  needing bitstream-crypto-v1 must use a different FPGA.

This KEP introduces a `SharingAffinity` field in the ResourceSlice `Device`
spec that allows drivers to declare which parameter keys constrain
sharing compatibility. On the claim side, it adds
`StructuredDeviceConfiguration` so workloads can express affinity values
(e.g. `subnet: subnet-A`) in a strongly-typed, scheduler-readable form.
The scheduler's `AllocatedState` is enhanced to track both consumed
capacity and the affinity values that lock a device to a particular
sharing group, enabling it to gate remaining capacity on locked devices
and safely reuse them for compatible claims when selected by the existing
allocator. Alpha provides correctness only — affinity-aware preference
(packing) is delivered in beta; see [Goals](#goals).

Alpha intentionally does not provide lock-breaking
preemption. In addition, if a device already has active allocations whose
affinity cannot be reconstructed (for example, legacy claims created before the
feature was enabled), the scheduler treats that device conservatively and does
not place new `sharingAffinity` allocations on it until the device becomes
clean.

`sharingAffinity` in this KEP refers specifically to compatibility for
co-allocation on a shared device; it is distinct from pod affinity,
anti-affinity, or topology-aware placement.

## Motivation

As AI and HPC workloads move toward higher density, hardware partitioning
(SR-IOV, GPU slicing, FPGA multi-tenancy) is becoming standard. These
physical devices often have a "modal" constraint (see [Summary](#summary)
for the definition and concrete examples).

Currently, the scheduler is unaware of this "lock." It may schedule a Pod
requiring a different configuration to the same device because it sees
"available capacity."
In short: **In these scenarios, Quantitative Sharing (how many slots?) fails
without Qualitative Gating (what mode are those slots in?).** This leads to:

1. **Allocation failures at the node level**: The driver rejects incompatible
   binds at prepare time, after the scheduler has already committed
2. **High scheduling latency**: The scheduler retries the same failing
   combination, thrashing between candidates
3. **Resource starvation**: Without affinity awareness, same-subnet pods
   spread across multiple devices instead of consolidating—wasting capacity
4. **Complex driver workarounds**: Drivers resort to placeholder patterns
   with race conditions and ResourceSlice churn (see [Status Quo](#status-quo-driver-side-placeholder-pattern) below)

The scheduler's `AllocatedState` currently tracks consumed capacity but not the
affinity values that determine sharing compatibility. This KEP closes that gap.

### Status Quo: Driver-Side Placeholder Pattern

Without this KEP, drivers must use a "placeholder pattern" today:

1. Publish devices with `capacity: 1` initially
2. Wait for first claim to determine affinity value
3. Update ResourceSlice with actual capacity and affinity as attribute
4. Use CEL selector to match affinity attribute

**Problems**:
- Race condition: Second pod may go to different device before expansion
- ResourceSlice churn: Constant updates as pods come and go
- Driver complexity: State machine for expand/contract lifecycle

### Goals

- Enable the scheduler to gate remaining capacity on a device based on a
  required sharing attribute
- Provide a mechanism for drivers to signal compatibility requirements for
  shared hardware via `SharingAffinity` in ResourceSlice
- Reduce fragmentation of cluster resources by enabling the scheduler to
  pack workloads with compatible sharing requirements onto already-locked
  devices (delivered in beta as a sharing-affinity term added to
  `DynamicResources.computeScore` — see [Affinity-aware scoring (planned
  for Beta)](#affinity-aware-scoring-planned-for-beta); alpha provides
  correctness only)
- Track affinity values in `AllocatedState` so subsequent scheduling decisions
  respect the first claim's lock-in
- Maintain backward compatibility with devices that have no sharing affinity
  constraints

### Non-Goals

- Defining hardware-specific attribute names (these remain driver-defined)
- Managing the physical lifecycle of the device configuration (this remains
  the driver's responsibility)
- Changing how capacity is tracked (that's KEP-5075)
- Supporting affinity across different device types or pools
- Retrofitting affinity-aware sharing onto already-in-use devices when active
  claims do not expose reconstructable affinity values. In alpha, such devices
  are treated conservatively until they drain clean.
- Guaranteeing **lock-breaking preemption** in alpha.
  Alpha enforces compatibility, but does not guarantee packing or
  lock-breaking preemption — both are planned for beta.

## Proposal

Add a `sharingAffinity` field to `Device` in ResourceSlice that specifies which device attribute keys constrain sharing:

```yaml
apiVersion: resource.k8s.io/v1
kind: ResourceSlice
spec:
  devices:
    - name: eth1
      allowMultipleAllocations: true
      sharingAffinity:
        parameterKeys: ["networking.example.com/subnet"]
      capacity:
        networking.example.com/slots:
          value: "16"
```

When the scheduler allocates a multi-allocatable device with `sharingAffinity`:

1. **First claim**: The scheduler reads the values for the keys declared
   by the device in `sharingAffinity.parameterKeys` from the claim's typed
   `Structured.Parameters` map and records them in `AllocatedState`
   alongside consumed capacity. Any additional keys present in the claim's
   `Structured.Parameters` but not declared by the device are ignored for
   this device.
2. **Subsequent claims**: The scheduler checks if the new claim's affinity values match those recorded in `AllocatedState`
3. **Mismatch**: If values don't match, the device is filtered out for
   that request (the scheduler tries another candidate device)
4. **Match**: If values match and capacity is available, allocation proceeds

**Alpha Design Decisions**

**1. Placement of SharingAffinity: ResourceSlice (driver-side)**

This KEP places `SharingAffinity` on the ResourceSlice `Device` (driver-
defined). We chose driver-side placement because the hardware modal constraint
is a property of the device, not the workload. The driver knows that "once a
NIC is configured for subnet A, it can only serve subnet A"—this is a
device-level constraint that should be declared once on the device.

An alternative design places `SharingAffinity` on the `DeviceRequest` in the
`ResourceClaim` (user-defined). See [Alternatives: Claim-side
SharingAffinity](#claim-side-sharingaffinity-on-devicerequest) for the
trade-off analysis.

**2. How claims communicate affinity values to the scheduler**

The driver declares `sharingAffinity.parameterKeys` on the device, telling the
scheduler which attribute keys constrain sharing. The scheduler learns the
requested values for those keys from a new typed sibling of
`OpaqueDeviceConfiguration` on `DeviceConfiguration`.

Today its only member is `Opaque`, which by design the scheduler does
not interpret. This KEP adds a sibling member `Structured` that carries
scheduler-readable, typed values. This also relaxes the existing
`DeviceConfiguration` invariant from "exactly one field set" to "at
least one," so that `Opaque` and `Structured` can coexist on the same
entry when a claim needs both driver-private and scheduler-readable
configuration for the same request:

```go
// DeviceConfiguration must have at least one field set. Both Opaque
// (driver-private) and Structured (scheduler-readable) may be set on
// the same entry; they are orthogonal and can describe the same
// request without semantic conflict.
type DeviceConfiguration struct {
    // Opaque is driver-private; the scheduler does not interpret it.
    Opaque *OpaqueDeviceConfiguration `json:"opaque,omitempty"`

    // Structured provides scheduler-readable, typed parameters for this
    // request. Used by features such as sharingAffinity (this KEP) that
    // need scheduler-input semantics rather than driver-private config.
    //
    // +featureGate=DRAStructuredDeviceConfiguration
    Structured *StructuredDeviceConfiguration `json:"structured,omitempty"`
}

// StructuredDeviceConfiguration carries typed, scheduler-readable
// parameters for a device request. Unlike OpaqueDeviceConfiguration, there
// is no Driver field: the consumer is the scheduler, and the vendor
// namespace is carried in each parameter key (FullyQualifiedName).
type StructuredDeviceConfiguration struct {
    // Parameters is a typed map from fully-qualified attribute name to
    // string value. In alpha, only string values are supported, matching
    // the device-side parameterKeys constraint on string-only matching.
    //
    // The cap of 8 mirrors the device-side
    // SharingAffinityParameterKeysMaxSize and is sized for known
    // typed-config use cases (1-3 keys per request is typical). It can be
    // relaxed in a backwards-compatible way if future typed-config
    // consumers (e.g. KEP-5993) need more.
    //
    // +required
    // +k8s:maxProperties=8
    Parameters map[FullyQualifiedName]string `json:"parameters"`
}
```

A plain `map[FullyQualifiedName]string` is chosen for alpha to keep API
validation, scheduler consumption, and restart reconstruction
straightforward; all motivating affinity values (subnets, partition keys,
bitstream identifiers) are naturally string-valued and matched by equality.
Richer typed alternatives (a discriminated union, or a shape mirroring
attribute value schemas) were considered but rejected for alpha because
they add validation, parsing, and restart-reconstruction complexity
without solving any known alpha use case. Richer typed siblings (e.g.
`IntParameters`, `BoolParameters`) can be added in a
backwards-compatible way without changing this map's shape if a future
consumer needs them.

The claim selects which requests these parameters apply to via the existing
`DeviceClaimConfiguration.Requests []string` selector — the same per-request
scoping mechanism already used for `Opaque`. No new plumbing is needed: the
DRA structured-parameters allocator already carries `DeviceConfiguration`
values through to `AllocationResult.Devices.Config[]` without inspecting
their content.

**Alpha Structured-Parameters Contract**

For alpha, the scheduler-readable parameter format is the typed
`StructuredDeviceConfiguration` field with the following rules:

1. **Per-request uniqueness**: For a given request, there must be at most
   one `DeviceClaimConfiguration` whose `Structured` is set and whose
   `Requests` selector includes that request. Multiple matching entries
   for the same request are rejected by API validation.
2. **Coexistence with driver config**: A claim may include both `Structured`
   and `Opaque` config blocks for the same request — either set on the
   same `DeviceClaimConfiguration` entry (most ergonomic) or split across
   separate entries. The two fields are orthogonal: `Opaque` is
   driver-private and `Structured` is scheduler-readable, so there is no
   semantic conflict in carrying both. API validation does require *at
   least one* of the two to be set on a `DeviceConfiguration` entry: an
   entry with neither field has no purpose and would only add ambiguity
   to per-request config selection.
3. **Source of truth for placement**: The scheduler is authoritative for
   placement based only on `Structured.Parameters` (filtering,
   lock-matching). The driver is authoritative for hardware programming
   based on `Opaque` (and may, optionally, cross-validate `Structured`
   against `Opaque` if it understands both — but is not required to).
   Drivers are not required to parse `Structured` at all. If a workload
   author encodes the same logical setting in both places with divergent
   values, that is a workload-side authoring bug — the scheduler will
   lock the device based on `Structured`, the driver will configure based
   on `Opaque`, and the resulting mismatch surfaces at runtime the same
   way any other misconfigured Opaque payload would.
4. **String-only affinity values in alpha**: For any key referenced by
   `sharingAffinity.parameterKeys`, the matching `Structured.Parameters`
   entry provides a string value. Other value types are not supported in
   alpha; the schema may be extended in beta.
5. **Missing entry**: If a claim targets a device with `sharingAffinity`
   but does not provide a `Structured` config entry covering the relevant
   request, the device is filtered out. This does not make the claim
   universally unschedulable; it only makes the claim ineligible for
   devices that declare `sharingAffinity`.
6. **Validation**: API validation rejects structurally invalid entries
   (e.g., duplicate `Structured` coverage for one request, more than 8
   keys per entry); semantic validation of specific value domains
   remains driver- or policy-specific. The scheduler additionally treats
   persisted claims that violate these structural invariants as if
   `Structured` were absent for that claim — covering version-skew or
   feature-gate-flap scenarios where an object reached etcd before the
   current validation applied.

The typed `Structured` field is a generic API extension intended to be
reusable by multiple consumers, so it is gated by its own feature gate
`DRAStructuredDeviceConfiguration`, separate from the behavioral
`DRASharingAffinity` gate that adds the `Device.SharingAffinity` field and
the scheduler logic that reads `Structured.Parameters`. See
[Feature Gates](#feature-gates) below for the dependency relationship.

**Alpha API invariants (cheat sheet)**

For quick reference during review, the contracts above reduce to:

- *At most one* `DeviceClaimConfiguration` with `Structured` set may target
  any given request name (per-request uniqueness).
- *All* keys declared in a device's `sharingAffinity.parameterKeys` must be
  present in the matching `Structured.Parameters`; otherwise the device is
  filtered out for that request.
- *Extra* keys in `Structured.Parameters` beyond what the device declares
  are ignored for that device (forward-compatibility for cross-device
  claims).
- *Affinity values are strings* in alpha; non-string siblings are reserved
  for beta extension.
- *Both `Opaque` and `Structured`* are permitted on the same
  `DeviceConfiguration` entry; they are orthogonal (driver-private vs
  scheduler-readable).
- *Inconsistent, missing, or non-reconstructable* lock state always yields
  conservative filtering (`AffinityStates[deviceID].Unknown = true`), never
  best-effort matching.

**Alpha scope**

Alpha fully resolves the design around driver-side placement and the
structured-parameters approach described above. Claims do not control
lock-setting behavior in alpha: any compatible claim may establish the initial
lock on a clean device. Claim-side lock-setting policy (for example,
`CanSetLock`/`NeverSetLock`) is deferred to [Future
Enhancements](#future-enhancements).

In other words, alpha standardizes driver-declared compatibility keys, a
typed `Structured` configuration field on `DeviceConfiguration`, and
correct lock enforcement on already-locked devices — but intentionally
stops short of affinity-aware scoring (planned as a beta-scope contribution
to `DynamicResources.computeScore`, see [Affinity-aware scoring (planned for
Beta)](#affinity-aware-scoring-planned-for-beta)) and of lock-breaking
preemption semantics.

**Alpha limitations**

Alpha provides correct lock enforcement, but it does
not provide lock-breaking preemption. A lower-priority
Pod may continue holding a device lock even when the device still has nominal
capacity and a higher-priority Pod needs the same device with a different
affinity value. In that case the higher-priority Pod may remain unschedulable
until a compatible alternative appears or the lock-holder exits. This is an
expected alpha limitation, not a correctness bug, and is addressed later under
[Future Enhancements: Priority-based Lock Preemption](#priority-based-lock-preemption).

### User Stories

#### Story 1: RDMA Partition Key Alignment

A user runs a distributed training job where every Pod must share the same
RDMA Partition Key (PKey) to communicate. The NIC supports 16 VFs. The driver
sets `sharingAffinity.parameterKeys: ["networking.example.com/pkey"]`. The scheduler
only co-allocates Pods whose claimed PKey matches the NIC's current lock
(or selects an unlocked NIC and establishes the lock from the first claim).

- Pod A (pkey-0x8001) is allocated to mlx5_0 → mlx5_0 is now locked to pkey-0x8001
- Pod B (pkey-0x8001) arrives → matches affinity, is eligible to share mlx5_0
- Pod C (pkey-0x8002) arrives → affinity mismatch on mlx5_0; mlx5_0 is
  filtered out; Pod C is allocated to mlx5_1 instead

#### Story 2: FPGA Bitstream Sharing

An inference service uses FPGAs to accelerate a specific model. Loading a
bitstream takes several seconds. The driver sets
`sharingAffinity.parameterKeys: ["fpga.example.com/bitstream"]`. The scheduler
only co-allocates Pods that request a compatible bitstream onto an
already-locked FPGA; affinity-aware *preference* for FPGAs that already
have the bitstream loaded (over fresh ones) is delivered in beta.

- Pod A (bitstream-ml-v2) is allocated an FPGA → FPGA locks to bitstream-ml-v2
- Pod B (bitstream-ml-v2) arrives → eligible to share the same FPGA
- Pod C (bitstream-crypto-v1) arrives → filtered out from the locked FPGA;
  uses a different FPGA or waits

#### Story 3: Single-subnet NIC Sharing

A network DRA driver advertises NICs that can be shared across up to 16 pods,
but only if pods belong to the same subnet. The driver sets
`sharingAffinity.parameterKeys: ["networking.example.com/subnet"]`.

- Pod A (subnet-X) is allocated to eth1 → eth1 is now locked to subnet-X
- Pod B (subnet-X) arrives → matches affinity, is eligible to share eth1
- Pod C (subnet-Y) arrives → affinity mismatch on eth1; eth1 is filtered
  out; Pod C is allocated to eth2 instead

### Notes/Constraints/Caveats

- **Affinity is set by the first compatible claim on a clean device**: Once a
  device is allocated with an affinity value, that value is locked until all
  claims release the device.
- **Attribute keys must be declared**: The device's
  `sharingAffinity.parameterKeys` lists which attribute keys constrain sharing;
  claims must provide values for all of these keys in their typed
  `Structured.Parameters` map or the device is filtered out.
- **Multiple keys**: If multiple attribute keys are specified, ALL must match
  (both presence and value).
- **Extra keys in claim**: If a claim's `Structured.Parameters` map contains
  keys beyond what the device declares in `parameterKeys`, the extra keys
  are ignored for that device. Only the device's declared keys are
  evaluated. This allows "generic" claims to work across devices with
  different sharing requirements (e.g., a claim with both `subnet` and
  `vlan` can match a device that only constrains on `subnet`).
- **String-only matching in alpha**: For keys referenced by
  `sharingAffinity.parameterKeys`, `Structured.Parameters` values are
  string-typed (the alpha map type is `map[FullyQualifiedName]string`).
  This is sufficient for all motivating use cases — subnet IDs, PKeys/GUIDs,
  partition names, model identifiers, NUMA tags, and FQDN-shaped
  identifiers are naturally string-valued, and sharing affinity matches by
  equality only (no numeric ranges or ordering), so richer types add
  complexity without unlocking use cases. Typed values can be added later
  in a backwards-compatible way by introducing sibling fields on
  `StructuredDeviceConfiguration` (e.g., `IntParameters`, `BoolParameters`)
  alongside the existing string map.
- **Missing keys in claim**: If the claim does not provide a value for a key
  the device declares in `parameterKeys`, the device is filtered out (see
  Filter phase).
- **Duplicate Structured entries for one request**: API validation rejects
  more than one `DeviceClaimConfiguration` whose `Structured` field is set
  and whose `Requests` selector covers the same request. Persisted claims
  predating the validation rule are treated as invalid for `sharingAffinity`
  scheduling until corrected.
- **Multi-request claims (per-request scoping)**: If a claim requests multiple
  devices (e.g., `mgmt-nic` and `data-nic`), each `DeviceClaimConfiguration`
  block targets specific requests via its `requests` slice. Different config
  blocks can specify different `Structured.Parameters` for different requests.
  This means `mgmt-nic` can be locked to Subnet-A while `data-nic` is locked
  to Subnet-B within the same claim — there is no cross-talk between requests
  because the scoping is structural. Lock state is enforced per physical
  device instance, not per request name: per-request scoping determines
  which `Structured` parameters apply to which allocation decision, but
  compatibility is always checked against the chosen device's
  `AffinityStates[deviceID]` entry.
- **Empty affinity**: Devices without `sharingAffinity` behave as before — any
  claim can share them regardless of whether it provides `Structured` parameters.
- **Legacy allocations with unknown affinity are conservative in alpha**:
  If a device has active allocations for which the scheduler cannot reconstruct
  the required affinity values (for example, claims created before the feature
  was enabled or invalid persisted claims), that device is treated as having
  unknown affinity state and is filtered out for new `sharingAffinity`
  scheduling until it becomes fully clean.

#### Handling Legacy Claims with Unknown Affinity

| Device State | New Claim | Result |
|---|---|---|
| 5 legacy claims, affinity unknown | Claim with `subnet: A` | **Filtered out**. Existing allocations have unknown affinity, so no new `sharingAffinity` lock may be established yet. |
| 5 legacy claims, affinity unknown | Claim without `Structured` parameters | **Filtered out**. Missing required scheduler-readable affinity information. |
| Legacy claims drained; device now clean | Claim with `subnet: A` | Lock set to `subnet: A`; device now locked. |
| Device locked to `subnet: A` | Claim with `subnet: A` | Allowed (values match). |
| Device locked to `subnet: A` | Claim with `subnet: B` | **Filtered out** (mismatch with lock). |
| All claims released | — | Device fully clean and eligible to establish a new lock. |

Legacy claims continue to run and are not evicted. However, until all unknown
allocations on a `sharingAffinity` device are released, the scheduler does not
assume it knows the device's effective modal state.

##### Changing `parameterKeys` on a Device with Active Allocations

In alpha, mutating `sharingAffinity.parameterKeys` on a device that already has
active allocations is not supported. If a driver adds, removes, or otherwise
changes the set of parameter keys while claims are bound to the device, the
scheduler's restart reconstruction will compare the current ResourceSlice
`parameterKeys` against the keys present in the bound claims' `Structured`
parameters. If they no longer line up — for example, the driver added a new
required key that pre-existing claims do not provide — the device is treated
as having unknown affinity state and filtered out for new `sharingAffinity`
scheduling until all such claims drain.

This is the same conservative-fallback behavior used for legacy claims and is
the deliberate alpha trade-off: the scheduler refuses to silently downgrade
its safety guarantee in the face of an asymmetric API change. Drivers that
need to evolve `parameterKeys` for an in-use device should drain the device
before publishing the change.

**Driver responsibility**: drivers should avoid hot-swapping
`parameterKeys` on devices with active allocations. When key changes are
unavoidable (e.g. a hardware capability evolves), drivers should expect
the affected device to be ineligible for new affinity-aware scheduling
until it drains clean, and should plan rollouts accordingly (for example,
by cordoning the device or rolling out the key change as part of a node
reimage).

#### Compatibility Matrix

To clarify the interaction between claims and devices, the following matrix
outlines how the scheduler and driver evaluate candidates based on whether
`SharingAffinity` (**SA**) is declared on the device (in ResourceSlice) and
whether the claim includes structured parameters (**SP** — i.e., a
`Structured` config entry covering the relevant request, in ResourceClaim):

| Scenario | Device SA | Claim SP | Scheduler Outcome | Driver Outcome |
|---|---|---|---|---|
| **Standard Feature Use** | Yes | Yes | **Match enforced.** Values match lock + capacity available → scheduled. | **Validates** hardware mode matches claim config at `NodePrepareResources`. Rejects if stale or inconsistent. |
| **Strict Gating** | Yes | No | **Filtered out.** Device excluded — requires affinity signal the claim does not provide. | **N/A** — claim never reaches the driver for this device. |
| **Legacy Device Transition** | Yes (newly added) | Yes | **Filtered out** while legacy claims are active (`Unknown: true`). Allowed once device drains clean. | **Validates** as normal once claim reaches the driver. During transition, driver continues serving legacy claims. |
| **Permissive Sharing** | No | Yes | **Allowed.** Device has no `sharingAffinity`; SP values are not evaluated for affinity. Standard capacity matching applies. | **Must enforce** hardware compatibility independently. Scheduler provides no affinity gating for this device. |
| **Legacy/Basic** | No | No | **Allowed.** Standard DRA capacity and attribute matching. | **Must enforce** hardware compatibility independently. This is the pre-KEP-5981 behavior. |

The top rows show the scheduler as the primary enforcer with the driver as
a backstop. The bottom rows show the driver as the sole enforcer with
the scheduler being permissive. The transition row shows the scheduler being
conservative (filtering) while the driver continues serving existing
workloads.

### Risks and Mitigations

#### Fragmentation (Poisoning)

**Risk**: One Pod with a unique affinity value could "lock" a high-capacity
device, preventing other more common workloads from using the rest of its
capacity.

**Mitigation (Alpha)**: None beyond Filter correctness. The DRA allocator
currently uses a first-fit algorithm with no affinity-aware preference, so
the scheduler does not actively pack compatible claims onto already-locked
devices. Affinity-aware preference (within-node) and a Score contribution
(cross-node) are planned as a beta-scope addition to
`DynamicResources.computeScore`, following the same per-feature additive
pattern that Prioritized List (KEP-4816, shipped in 1.35) and Extended
Resources (KEP-5004) already use; see [Affinity-aware scoring (planned for
Beta)](#affinity-aware-scoring-planned-for-beta). Until that lands,
fragmentation mitigation is best-effort and depends on the existing
first-fit ordering of devices in ResourceSlices. General-purpose DRA
scoring discussion continues in
[kubernetes/enhancements#4970](https://github.com/kubernetes/enhancements/issues/4970),
but KEP-5981's contribution does not block on a unified framework.

#### Priority Inversion (Preemption Blindness)

**Risk**: Standard Kubernetes preemption is blind to affinity locks. It triggers
on *resource shortage*, not affinity mismatch. If a NIC has 15/16 slots
available but is locked to the wrong subnet, the scheduler sees plenty of
capacity and never enters the preemption path. A single low-priority Pod can
permanently hold a high-capacity device hostage by setting a lock that no
high-priority Pod can break.

Even if preemption were triggered by an unrelated shortage, victim selection
asks "which Pods free up slots?" — not "which Pods clear the lock?" The
scheduler might preempt an unrelated Pod, freeing a slot on a device still
locked to the wrong value.

**Mitigation (Alpha)**: None. Alpha does not provide lock-breaking
preemption, and lock-aware preemption is the actual fix — not packing.
Even a perfectly-packed cluster can reach a state where every device is
locked to incompatible values, at which point a higher-priority claim has
no path forward without the ability to evict lock-holders.

**Alpha limitation**: In alpha, a lower-priority Pod may continue to hold a
lock that blocks a higher-priority incompatible Pod even when nominal capacity
remains on the device. This is an expected limitation of the alpha scope rather
than a correctness bug.

**Mitigation (Beta)**: Lock-aware preemption (see [Beta graduation criteria](#beta))
will teach the scheduler's PostFilter phase to detect affinity mismatch as a
preemption-solvable problem and identify lock-holder Pods as preemption victims.

#### Cache Staleness and Delayed Release Visibility

**Risk**: Like other informer-based scheduler state, sharing-affinity lock state may
briefly lag external claim release, pod deletion, eviction, or ResourceSlice updates.
Because the scheduler maintains derived lock state in its internal cache, there can be
a short propagation window in which a device is still observed as locked or in unknown-affinity
state after the underlying API state has changed. During that window, the scheduler may
conservatively skip the device for a scheduling cycle. This is not unique to sharing affinity;
it is the feature-specific manifestation of normal cache propagation delay in scheduler-managed
state. The result is a temporary loss of placement optimality rather than a correctness violation

**Mitigation**: For scheduler-driven transitions such as Reserve / Unreserve, the cache is updated
immediately. For externally driven transitions, informer reconciliation eventually converges the
state. This matches the existing consistency model used elsewhere in scheduler and DRA cache-based decisions.

**Scope of "release"**: The scheduler derives lock state purely from
ResourceClaim allocation/deallocation events; it does not observe kubelet's
`NodePrepareResources` / `NodeUnprepareResources` progress, which remains the
driver's responsibility. The scheduler's notion of "clean" is purely
allocation-clean, not hardware-ready — a device is considered clean once no
allocated claim references it, regardless of whether the driver has finished
in-flight hardware reconfiguration on the node. Driver-level prepare/unprepare
sequencing remains the authoritative guard against reuse before
reconfiguration completes; drivers that need stronger guarantees should
hold their own per-device readiness state and reject prepare calls until
reconfiguration is complete.

#### Unexpected Affinity Values

**Risk**: A claim specifies an unexpected or unique affinity value (e.g., an
arbitrary subnet GUID or name), further fragmenting devices by locking them to
rare values.

**Mitigation**: In many cases, affinity values are externally defined (subnet
names, partition keys) and cannot be validated by the driver. Cluster
administrators can use `DeviceClass` CEL selectors to restrict which
attribute values are accepted where domain-specific validation is feasible.
Allocator-level packing of compatible workloads onto already-locked devices
(which would naturally limit fragmentation) is delivered in beta as a
sharing-affinity term added to `DynamicResources.computeScore`; see
[Affinity-aware scoring (planned for
Beta)](#affinity-aware-scoring-planned-for-beta).

#### Packing Depends on DRA-Aware Scoring (Alpha Limitation)

**Risk**: This KEP's Filter phase guarantees correctness (incompatible
locked devices are excluded), but it does not guarantee packing — neither
across nodes nor within a node:

- **Cross-node**: standard Kubernetes scorers do not see DRA shared-device
  consumption, so they cannot prefer a node that already has a
  compatibly-locked device. Two compatible claims may land on two different
  nodes — each locking a separate device — even when consolidating onto one
  node would have sufficed.
- **Within-node**: the DRA allocator currently uses a first-fit algorithm.
  When a node has both a compatibly-locked device with capacity and a
  clean device, the allocator may pick whichever appears first in the
  ResourceSlice rather than preferring the locked one.

**Mitigation (Alpha)**: None within this KEP's scope. Alpha provides
correctness only; packing of any kind is best-effort first-fit.

**Long-term fix**: A sharing-affinity-aware Score contribution is planned
for beta as an additive term in `DynamicResources.computeScore`, following
the same per-feature pattern that Prioritized List (KEP-4816, shipped in
1.35) and Extended Resources (KEP-5004) already use. Within-node device
preference among feasible devices is similarly addressed by extending the
allocator's selection logic. The `AllocatedState.AffinityStates` structure
introduced in alpha is the substrate the beta score function reads, so
the alpha design is not a dead end — it is the necessary infrastructure
for the eventual packing optimization. See [Affinity-aware scoring (planned for
Beta)](#affinity-aware-scoring-planned-for-beta). General-purpose DRA
scoring discussion is tracked in
[kubernetes/enhancements#4970](https://github.com/kubernetes/enhancements/issues/4970),
but this KEP's contribution does not block on a unified scoring framework.

#### Memory Overhead

**Risk**: Affinity values accumulate in `AllocatedState`, increasing memory usage.

**Mitigation**: In alpha, affinity values are stored as small strings (for
example subnet or PKey identifiers), capped at 8 attribute keys per device.
Per-device overhead is bounded at 8 key-value pairs in
`AllocatedState.AffinityStates`, and entries are cleared when all claims release
the device. The total overhead is proportional to active shared allocations,
not total devices.

## Design Details

### API Enhancement

#### ResourceSlice Device Spec

```go
type Device struct {
    // ... existing fields (Name, Attributes, Capacity,
    // AllowMultipleAllocations, Taints, etc.) ...

    // SharingAffinity specifies constraints for sharing this device across
    // multiple allocations. If set, only claims with matching affinity values
    // for the specified attribute keys can share this device.
    //
    // This field is only meaningful when AllowMultipleAllocations is true.
    //
    // +optional
    // +featureGate=DRASharingAffinity
    SharingAffinity *DeviceSharingAffinity
}

// DeviceSharingAffinity defines which device attribute keys constrain
// sharing across multiple claims.
type DeviceSharingAffinity struct {
    // parameterKeys lists the fully-qualified device attribute names that
    // must have matching values across all claims sharing this device.
    //
    // In alpha, the corresponding values must be provided as strings in the
    // claim's StructuredDeviceConfiguration.Parameters map (see
    // DeviceClaimConfiguration.Structured). Support for additional value
    // types is deferred.
    //
    // When the first claim is allocated to this device, the affinity values
    // for these keys are recorded in AllocatedState. Subsequent claims can
    // only share the device if their affinity values match exactly.
    //
    // The maximum number of attribute keys is 8.
    //
    // +required
    // +listType=atomic
    // +k8s:maxItems=8
    parameterKeys []FullyQualifiedName
}

const SharingAffinityParameterKeysMaxSize = 8
```

#### Scheduler Enhancement

##### Source of Truth for Affinity Locks

The scheduler derives affinity locks **solely from active claims' typed
`Structured` config entries** — not from device attributes on the
ResourceSlice. The driver is NOT required to write locked affinity values
back to the ResourceSlice.

- The ResourceSlice declares *which* keys constrain sharing (`parameterKeys`)
- The claims declare *what* values they need (via `Structured.Parameters`)
- The scheduler combines these to maintain the lock in `AllocatedState`

This avoids two sources of truth that could diverge, eliminates ResourceSlice
churn (no update every time a lock is set/cleared), and keeps driver
implementation simple. Drivers MAY optionally publish current locked values as
regular device attributes for observability (e.g., visible via `kubectl`), but
the scheduler does not depend on them.

When the last claim on a device is released, the scheduler clears the lock. The
driver is responsible for device lifecycle — tearing down the old configuration
(via `NodeUnprepareResources`) and reconfiguring for new claims (via
`NodePrepareResources`). The scheduler does not track hardware reconfiguration
state.

##### Safety Model and Responsibility Split

This feature intentionally keeps placement knowledge and hardware
enforcement separate:

- **Scheduler guarantee**: when it has typed `Structured` parameters for all
  active allocations on a `sharingAffinity` device, it will not intentionally
  co-place claims with incompatible affinity values on that device.
- **Conservative fallback**: if the scheduler cannot reconstruct the effective
  affinity state of a device (for example, due to legacy or invalid persisted
  claims), it treats that device as unknown and filters it out for new
  `sharingAffinity` placements until the device becomes clean.
- **Driver guarantee**: the driver remains the final authority for programming
  and validating the actual hardware mode during `NodePrepareResources`.
- **Failure handling**: stale scheduler state or races may still cause prepare-
  time rejection, and that rejection remains the final safety backstop.

##### Cache Extension: Effective Device State

To prevent race conditions during high-volume scheduling, the scheduler
maintains affinity locks in its internal cache rather than relying on API server
round-trips. This is consistent with how DRA already handles capacity tracking
via `inFlightAllocations`.

The scheduler's `AllocatedState` is extended to track affinity values alongside
consumed capacity:

```go
type AffinityState struct {
    // Unknown indicates that one or more active claims on the device do not
    // expose reconstructable affinity values. When true, the device is filtered
    // for new sharing-affinity placements until fully clean.
    Unknown bool

    // LockedAffinity stores the known lock for a device when Unknown is false.
    // Empty means the device is clean/unlocked.
    LockedAffinity map[string]string
}

type AllocatedState struct {
    AllocatedDevices         sets.Set[DeviceID]
    AllocatedSharedDeviceIDs sets.Set[SharedDeviceID]
    AggregatedCapacity       ConsumedCapacityCollection
    
    // +featureGate=DRASharingAffinity
    AffinityStates map[DeviceID]AffinityState
}
```


##### Filter Phase and Device Selection

**Filter phase**: For a given node, the scheduler evaluates each device. A
device with `sharingAffinity` is a candidate ONLY if:

1. It has sufficient consumable capacity (KEP-5075)
2. The device's `AffinityStates[deviceID].Unknown` is not true
3. The claim has exactly one `DeviceClaimConfiguration` whose `Structured`
   field is set and whose `Requests` selector covers the relevant request
4. The claim provides values for ALL keys in `sharingAffinity.parameterKeys`
   (missing key → device is not a candidate)
5. For each required affinity key, the `Structured.Parameters` map provides
   a string value (validated by API admission; non-string values cannot be
   persisted in alpha)
6. The device's `AffinityStates[deviceID].LockedAffinity` is either empty
   (unlocked) OR matches the claim's affinity values for ALL keys

If a device has `AffinityStates[deviceID].Unknown` set, or if a required
request has no `Structured` config entry, more than one such entry, or a
required affinity key missing, the device is filtered out for
`sharingAffinity` scheduling. This is the safe default: the driver declared
that sharing requires specific scheduler-readable parameters, and a
scheduler that cannot reconstruct the current or requested affinity state
cannot evaluate placement safely. Claims that do not need
sharing-constrained devices should target devices without `sharingAffinity`.

**Device selection within a node**: For correctness, the Filter phase alone
is sufficient — incompatible locked devices are excluded, and any remaining
candidate (locked-compatible or unlocked) produces a valid allocation.

Among the remaining feasible devices on a chosen node, alpha does not
introduce affinity-aware preference. Device selection continues to use the
existing structured-parameters allocator (first-fit). This means that on a
node with both a compatibly-locked device with capacity and a clean device,
the allocator may pick whichever appears first in the ResourceSlice rather
than preferring the locked one.

Affinity-aware preference (within a node) and node-level scoring (across
nodes) are planned as a beta-scope contribution to
`DynamicResources.computeScore`, following the same per-feature additive
pattern that Prioritized List (KEP-4816) and Extended Resources (KEP-5004)
already use. See [Affinity-aware scoring (planned for
Beta)](#affinity-aware-scoring-planned-for-beta) for the design sketch.
General-purpose DRA scoring discussion is tracked in
[kubernetes/enhancements#4970](https://github.com/kubernetes/enhancements/issues/4970),
but this KEP's contribution does not block on it. KEP-5981's data-plane
contribution to that future work is to expose the lock state in
`AllocatedState.AffinityStates` so the score function (and any
generalized successor) can consume it.


##### Reserve Phase: Tentative Locking

Once a node/device is selected, the Reserve plugin establishes a "tentative
lock" in the scheduler cache before the Binding phase:

1. Scheduler evaluates a multi-allocatable device with `sharingAffinity`
2. If device has no existing allocations (unlocked):
   - Read affinity values for `sharingAffinity.parameterKeys` from the
     claim's `Structured.Parameters` map
   - Record values in `AllocatedState.AffinityStates[deviceID].LockedAffinity`
   - Proceed with allocation (device is now tentatively locked)
3. If device has existing allocations (locked):
   - Compare claim's affinity values against `AllocatedState.AffinityStates[deviceID].LockedAffinity`
   - If all keys match: proceed with allocation (co-allocate onto locked device)
   - If any key mismatches: skip this device, try next candidate

This tentative lock is immediately visible to subsequent scheduling cycles. If
Pod-B is evaluated milliseconds after Pod-A's Reserve (before Pod-A's bind
reaches the API server), Pod-B's Filter phase will see Pod-A's tentative lock
and either join it or skip the device. This follows the same pattern used by
`SignalClaimPendingAllocation()` for capacity tracking.

##### State Transitions

| Event | Cache Action | Result |
|-------|-------------|--------|
| Pod scheduled (Reserve) | Set `AffinityStates[deviceID].LockedAffinity` | Device locked; subsequent claims must match |
| Scheduling failure (Unreserve) | Remove tentative lock if no other claims share it | Device may become unlocked |
| All claims released | Clear `AffinityStates[deviceID]` | Device becomes unlocked |
| Driver adds `sharingAffinity` to in-use device | Mark `AffinityStates[deviceID].Unknown` if active claims are non-reconstructable | Device blocked for new sharing workloads until legacy claims drain |

##### Implementation Note: Snapshot Consistency

Since the scheduler works on a snapshot of the cache for each Pod, the Reserve
phase must update the primary cache so that subsequent snapshots in the same
scheduling cycle reflect the new lock. This aligns with how VolumeBinding and
PodAffinity currently handle "assumed" states.

**Parallel scheduling**: In clusters with parallel scheduling enabled, multiple
pods may reach the Filter phase concurrently. Without protection, two pods with
*different* affinities could both pass Filter for the same clean device in the
same millisecond. To prevent this, all reads and writes to
`AllocatedState.AffinityStates` must be protected by the `AllocatedState` mutex.
The Filter phase acquires a read lock to check the current affinity state; the
Reserve phase acquires a write lock to set the tentative lock atomically. This
ensures that once one pod's Reserve completes, the next pod's Filter sees the
updated lock.

##### Scheduler Restart: State Reconstruction

On scheduler restart, the in-memory `AffinityStates` map is empty. The scheduler
must reconstruct affinity locks from persisted state before the first scheduling
cycle begins.

**Reconstruction algorithm**:

1. On startup, the scheduler iterates all `Bound` ResourceClaims (same path
   as existing `GatherAllocatedState()` for capacity reconstruction).
2. For each bound claim, check if the allocated device has `SharingAffinity`
   defined in the corresponding ResourceSlice.
3. If yes, look for a `DeviceClaimConfiguration` whose `Structured` field
   is set and whose `Requests` selector covers the request that produced
   the allocation. Read the values for the device's required
   `parameterKeys` directly from `Structured.Parameters`.
4. If all required keys are present (and string-valued, as enforced by API
   validation), populate
   `AffinityStates[deviceID].LockedAffinity` with those values.
5. If the claim has no matching `Structured` entry, has multiple matching
   entries, or is missing a required affinity key, set
   `AffinityStates[deviceID].Unknown = true` and log a warning. The
   scheduler must not infer lock state from ambiguous data.
6. If multiple claims share the same device and any one of them causes the
   device to become unknown, the device remains with `AffinityStates[deviceID].Unknown` set
   until all claims on that device are released.
7. If multiple reconstructable claims share the same device, verify their values
   are consistent. By construction they should be — the Reserve plugin only
   admits compatible claims onto a locked device — but if reconstruction
   nevertheless yields inconsistent values (e.g. due to historical bugs,
   manual etcd edits, or unsupported version skew), set
   `AffinityStates[deviceID].Unknown = true` and log a warning. The device
   is then filtered out for new sharing-affinity placements until it becomes
   clean, consistent with the conservative-fallback model used elsewhere
   for ambiguous lock state.

This follows the same pattern used to reconstruct `AggregatedCapacity` from
bound claims on startup. No new API calls are needed; the data is already
available from the ResourceClaim spec and ResourceSlice spec cached by the
scheduler's informers.


### Feature Gates

This KEP introduces two feature gates with a dependency relationship:

- **`DRAStructuredDeviceConfiguration`** (alpha): adds the typed
  `Structured *StructuredDeviceConfiguration` sibling field on
  `DeviceConfiguration` and its API validation. This gate carries no
  scheduler behavior on its own — it is a generic, reusable API extension
  intended to be consumed by multiple features (this KEP, KEP-5993
  Context-Locked Effective Capacity, and potentially others). Gating the
  field separately allows the API surface to merge and graduate
  independently of any single consumer.

- **`DRASharingAffinity`** (alpha): adds the behavioral surface specific to
  this KEP — `Device.SharingAffinity`, `AllocatedState.AffinityStates`, and
  the Filter / Reserve logic that *reads* `Structured.Parameters` for
  affinity matching. `DRASharingAffinity` must not be enabled unless
  `DRAStructuredDeviceConfiguration` is also enabled on `kube-apiserver`
  and `kube-scheduler`: if the API gate is off, claims cannot reliably
  carry scheduler-readable parameters, so behavioral enforcement would be
  invalid. The scheduler treats this combination as a misconfiguration
  and no-ops affinity enforcement. This degraded behavior is intentional:
  the driver remains the safety backstop at `NodePrepareResources`, so
  permissive scheduling is operationally safer than hard-failing scheduler
  startup, which would destabilize control plane availability for the
  whole cluster.

Both gates target the same alpha release and must be enabled on
`kube-apiserver` and `kube-scheduler`. KEP-5993 will reuse
`DRAStructuredDeviceConfiguration` and add its own behavioral gate; the API field
itself does not need to graduate per consumer.


### Examples

#### ResourceSlice with Sharing Affinity

```yaml
apiVersion: resource.k8s.io/v1
kind: ResourceSlice
metadata:
  name: node1-nics
spec:
  driver: networking.example.com
  nodeName: node1
  devices:
    - name: eth1
      allowMultipleAllocations: true
      sharingAffinity:
        parameterKeys: ["networking.example.com/subnet"]
      attributes:
        networking.example.com/type:
          string: "sriov-vf"
      capacity:
        networking.example.com/slots:
          value: "16"
    - name: eth2
      allowMultipleAllocations: true
      sharingAffinity:
        parameterKeys: ["networking.example.com/subnet"]
      attributes:
        networking.example.com/type:
          string: "sriov-vf"
      capacity:
        networking.example.com/slots:
          value: "16"
```

#### ResourceClaim with Affinity Value

```yaml
apiVersion: resource.k8s.io/v1
kind: ResourceClaim
metadata:
  name: pod-a-nic
spec:
  devices:
    requests:
      - name: nic
        exactly:
          deviceClassName: shared-nic
    config:
      # Both scheduler-readable (structured) and driver-private (opaque)
      # config can live on the same entry — they target different audiences
      # and are orthogonal.
      - requests: ["nic"]
        structured:
          parameters:
            networking.example.com/subnet: "subnet-X"
        opaque:
          driver: networking.example.com
          parameters:
            apiVersion: networking.example.com/v1
            kind: NICConfig
            vlanId: 100
```

> **Note**: The `structured` field is read by the scheduler for affinity
> matching (no decoding). The `opaque` field is standard driver-private
> config that only the driver reads. Both fields can be set on the same
> `DeviceClaimConfiguration` entry (recommended for ergonomics) or split
> across separate entries targeting the same request — both forms are
> valid. For simple drivers that only need scheduler-readable parameters,
> a single `structured` entry is sufficient.

#### Multi-key SharingAffinity Example

This example illustrates the alpha semantics when a device constrains sharing on
multiple keys.

A driver advertises a shared RDMA-capable NIC where both subnet and PKey
must match for pods to share the same device:

```yaml
apiVersion: resource.k8s.io/v1
kind: ResourceSlice
spec:
  devices:
    - name: mlx5_0
      allowMultipleAllocations: true
      sharingAffinity:
        parameterKeys:
          - networking.example.com/subnet
          - networking.example.com/pkey
      capacity:
        networking.example.com/slots:
          value: "16"
```

A matching claim provides both values as typed `Structured` parameters:

```yaml
config:
  - requests: ["rdma-nic"]
    structured:
      parameters:
        networking.example.com/subnet: "subnet-a"
        networking.example.com/pkey: "0x8001"
        networking.example.com/vlan: "100"
```

Alpha matching behavior:

- If the device is clean, the first compatible claim sets the lock to:
  - `subnet = subnet-a`
  - `pkey = 0x8001`
- A later claim with the same `subnet` and same `pkey` may share the
  device.
- A claim with `subnet = subnet-a` but `pkey = 0x8002` is filtered out for
  that device because all declared keys must match.
- A claim that provides only `subnet` but omits `pkey` is filtered out for
  that device because missing declared keys are invalid.
- The extra `vlan` key is ignored for this device because the driver did not
  declare `networking.example.com/vlan` in `parameterKeys`.

### Test Plan

[x] I/we understand the owners of the involved components may require updates to
existing tests to make this code solid enough prior to committing the changes necessary
to implement this enhancement.

##### Prerequisite testing updates

Existing DRA scheduling tests should pass before adding sharing affinity tests.

##### Unit tests

- `pkg/scheduler/framework/plugins/dynamicresources`: Coverage for affinity matching
  logic, including:
  - Filter: device with matching lock passes
  - Filter: device with conflicting lock is excluded
  - Filter: unlocked device with sufficient capacity passes
  - Filter: claim missing a required `parameterKey` → device filtered out
  - Filter: claim with extra keys beyond device's declared `parameterKeys` → extra
    keys ignored, device passes if declared keys match
  - Filter: no `Structured` config entry covering a sharing-constrained
    request → device filtered out
  - Filter: device with `AffinityStates[deviceID].Unknown` set is excluded for new
    `sharingAffinity` scheduling
  - First-fit verification: with two feasible devices on a node (one locked-compatible,
    one clean), allocator picks the first in ResourceSlice order (no affinity-aware
    preference; preference lands in beta — see [Affinity-aware scoring
    (planned for Beta)](#affinity-aware-scoring-planned-for-beta))
  - Reserve: first claim sets lock; second claim with same values succeeds
  - Reserve: second claim with conflicting values fails
  - Unreserve: tentative lock is rolled back
  - Legacy claims with non-reconstructable affinity cause the device to be marked
    unknown rather than establishing or joining a lock
  - Legacy-claim handling: all scenarios from the `Handling Legacy Claims with
    Unknown Affinity` table
  - Compatibility matrix: device without `sharingAffinity` is unaffected —
    claims with or without `Structured` parameters both pass (Legacy/Basic and
    Permissive Sharing rows)
  - Strict Gating: device has `sharingAffinity` but claim provides no
    `Structured` config entry covering the request → device filtered out
  - Multi-request scoping: claim with two requests (`mgmt-nic` and `data-nic`)
    each with distinct `Structured` config blocks → each request resolves
    independently; one request's affinity values do not influence the other's
    filter decision or lock state on a different device
- `staging/src/k8s.io/api/resource/v1`: Coverage for the new typed
  `Structured` field on `DeviceConfiguration` and the `SharingAffinity` API,
  including:
  - Validation: `parameterKeys` exceeding max 8 limit is rejected
  - Validation: `Structured.Parameters` exceeding max 8 entries is rejected
  - Validation: more than one `DeviceClaimConfiguration` with `Structured`
    set covering the same request is rejected
  - Validation: at least one of `Opaque` or `Structured` is set per
    `DeviceConfiguration` (an entry with neither has no purpose); both
    being set on the same entry is permitted because they are orthogonal
    (driver-private vs. scheduler-readable)
  - Round-trip serialization of `SharingAffinity` and
    `StructuredDeviceConfiguration`

##### Integration tests

- Affinity matching with multiple claims to same device: in a single-device
  topology (or with selectors that constrain feasibility to one device),
  verify that a second compatible claim shares the locked device and
  extends the affinity lock. Tests Filter correctness and Reserve-phase
  lock extension; preference for the locked device when alternatives
  exist is a beta concern (see [Affinity-aware scoring (planned for
  Beta)](#affinity-aware-scoring-planned-for-beta)).
- Affinity mismatch causing allocation to different device
- Affinity lock clearing when all claims release a device
- Interaction with consumable capacity constraints (KEP-5075)
- Scheduler restart: `AffinityStates` correctly reconstructed from existing
  bound ResourceClaims, and devices with non-reconstructable active claims have
  `AffinityStates[deviceID].Unknown` set
- Parallel scheduling: two Pods with conflicting affinity values targeting the
  same device — one wins Reserve, the other is requeued
- `DRASharingAffinity` disabled: `sharingAffinity` fields are ignored;
  devices are treated as unconditionally shareable
- `DRASharingAffinity` toggled: enabling after claims exist does not
  disrupt already-bound workloads, and legacy in-use devices are
  conservatively filtered until clean
- Invalid `Structured` parameters at scheduling time: regression test that
  malformed claims rejected at API admission cannot reach the scheduler, and
  that any historical claim slipping through deterministically excludes
  sharing-constrained devices rather than crashing
- Permissive Sharing (no SA): Device without `sharingAffinity`, claim with
  `Structured` parameters — verify scheduler allows the allocation and
  `Structured.Parameters` are not evaluated for affinity
- Ghost Lock: Pod is Assumed (tentative lock set) but Bind fails — verify
  the lock is cleared immediately and the next Pod in the queue can claim the
  device with a different affinity value
- Legacy Device Migration: 5 Pods are already running on a NIC; the driver
  updates `ResourceSlice` to add `sharingAffinity`; a 6th Pod arrives with
  `Structured` parameters — verify the device has
  `AffinityStates[deviceID].Unknown` set and the 6th Pod is filtered from
  that device until all legacy claims drain
- Partial Key: Device requires `subnet` and `pkey` in `parameterKeys`;
  claim provides only `subnet` — verify the device is filtered out
- First-Fit Behavior: Two devices available on a node, one already locked to
  subnet-X, one clean; new claim for subnet-X — verify the claim is allocated
  successfully (Filter excludes nothing; either device is feasible) and that
  the chosen device matches the allocator's existing first-fit ordering. Alpha
  does not require the locked device to be preferred; affinity-aware
  preference is delivered in beta — see [Affinity-aware scoring (planned for
  Beta)](#affinity-aware-scoring-planned-for-beta).
- Driver Backstop: Device without `sharingAffinity`, two claims with
  incompatible config land on the same device — verify scheduler allows both
  (permissive), and `NodePrepareResources` rejects the incompatible claim
- Gate-dependency misconfiguration: `DRASharingAffinity` enabled while
  `DRAStructuredDeviceConfiguration` is disabled on apiserver/scheduler —
  verify the scheduler no-ops affinity enforcement (no panic, no false
  filtering) and that a clear log/event is emitted indicating the
  unsupported configuration
- NodePrepareResources failure does not clear lock: Claim is bound and
  lock is set in the scheduler cache, but `NodePrepareResources` fails on
  the node — verify the affinity lock remains in the scheduler cache
- Multi-request, multi-device, no cross-talk: Single claim with two requests
  (`mgmt-nic`, `data-nic`), each with distinct `Structured` parameters
  (`subnet=A` and `subnet=B`) targeting two different `sharingAffinity`
  devices on the same node — verify both requests succeed in the same
  scheduling cycle and each device locks to its own request's values without
  influence from the sibling request
- Restart with inconsistent reconstructable locks: two active reconstructable
  claims on the same device disagree on a required key value during restart
  reconstruction — verify the device is marked
  `AffinityStates[deviceID].Unknown = true`, a warning is logged, and new
  sharing-affinity scheduling is blocked on that device until all claims drain
- parameterKeys mutation with active claims: device locked with claims
  carrying `parameterKeys=[subnet]`; driver updates ResourceSlice to
  `parameterKeys=[subnet, vlan]` — verify the device is marked
  `AffinityStates[deviceID].Unknown` after the next reconciliation/restart
  (because pre-existing claims do not provide the new key in `Structured`)
  and is filtered out for new `sharingAffinity` scheduling until existing claims
  drain

##### e2e tests

- End-to-end test with mock DRA driver using sharing affinity
- Multi-pod scheduling: Pods with matching affinity values share the same device
- Multi-pod scheduling: Pods with conflicting affinity values are placed on
  different devices
- Lock lifecycle: last Pod deleted → lock cleared → new Pod with different
  affinity value can claim the device
- Rollout scenario: existing Pods running without `sharingAffinity`; driver
  adds `sharingAffinity` to ResourceSlice; verify existing Pods continue
  running and new Pods respect the new constraint after legacy claims drain

### Graduation Criteria

#### Alpha

- Feature implemented behind two feature gates:
  - `DRAStructuredDeviceConfiguration` — adds the typed `Structured` sibling on
    `DeviceConfiguration` (generic, reusable API surface)
  - `DRASharingAffinity` — adds `Device.SharingAffinity`,
    `AllocatedState.AffinityStates`, and the scheduler logic that reads
    `Structured.Parameters` for affinity matching; depends on
    `DRAStructuredDeviceConfiguration`
- API fields added to ResourceSlice (`SharingAffinity` on `Device`)
- Typed `Structured *StructuredDeviceConfiguration` sibling field added on `DeviceConfiguration`
- Scheduler reads `Structured.Parameters` for affinity matching; no opaque
  decoding involved
- Scheduler Filter plugin enforces affinity matching
- Affinity-aware packing (both within-node and cross-node) is out of scope for alpha
- Scheduler tracks affinity in AllocatedState
- Unit and integration tests
- Documentation for driver authors
- Alpha documentation explicitly calls out the lack of lock-breaking preemption
  semantics for incompatible locks
- Alpha documentation explicitly calls out string-only affinity matching
  (alpha `Structured.Parameters` value type is `string`)
- Distinct alpha diagnostics emitted via scheduler logs and (best-effort)
  events for: (a) compatibility mismatch with the current lock, (b) missing
  required `Structured` parameters for a `sharingAffinity` device, and
  (c) unknown lock state due to legacy / non-reconstructable / inconsistent
  active claims — sufficient to attribute a filtered scheduling decision
  without relying on metric labels alone

#### Beta

- Gather feedback from DRA driver developers
- Address any issues found in alpha
- **Lock-aware preemption**: PostFilter detects affinity mismatch as a
  preemption-solvable problem; identifies lock-holder Pods as victims when a
  higher-priority Pod needs a device locked to an incompatible value
- **Affinity-aware scoring contribution**: Extend
  `DynamicResources.computeScore` with a sharing-affinity term — prefer
  nodes where the target device is already locked to a compatible value
  (consolidation), then clean devices, with unknown-affinity devices
  deprioritized. Extend the structured-parameters allocator's per-node
  device selection with the same preference. Follows the per-feature
  additive scoring pattern that Prioritized List (KEP-4816, shipped in
  1.35) and Extended Resources (KEP-5004) already use; not blocked on the
  general-purpose scoring discussion in
  [#4970](https://github.com/kubernetes/enhancements/issues/4970). See
  [Affinity-aware scoring (planned for
  Beta)](#affinity-aware-scoring-planned-for-beta).
- **Observability of lock state**: Surface effective per-device lock state
  for operators — exact mechanism TBD in beta. Candidates include
  scheduler-side metrics/gauges keyed by device and parameter hash,
  scheduler Events on Pending pods naming the locking claim, aggregation
  tooling over existing `ResourceClaim.status.allocation`, or a dedicated
  scheduler-owned API resource.
- E2e tests stable
- Performance validation with high pod churn

#### GA

- At least 2 production drivers using sharing affinity
- No significant issues reported
- Conformance tests if applicable

### Upgrade / Downgrade Strategy

**Upgrade**: Existing ResourceSlices without `sharingAffinity` continue to work.
New field is additive. See the [Compatibility Matrix](#compatibility-matrix) for
how the scheduler and driver behave across all combinations of device
`sharingAffinity` and claim `Structured` parameters presence.

#### Recommended Rollout (responsibility split)

The two-feature-gate split is intentional: `DRAStructuredDeviceConfiguration`
gates the API surface (claim-side `Structured.Parameters`) and is meant to
land first so workload teams can adopt the new field while it remains a
no-op for placement; `DRASharingAffinity` gates enforcement (device-side
`sharingAffinity` + scheduler matching) and should be turned on only after
workloads have adopted `Structured.Parameters` for the relevant claims.

The recommended sequence, mapped to who typically owns each step:

1. **Platform team — enable the API gate (`DRAStructuredDeviceConfiguration`)**
   on apiserver and scheduler. This is behaviorally a no-op: it only allows
   the typed `Structured` field to be persisted in `ResourceClaim`s. No
   scheduling decisions change. Safe to enable cluster-wide.

2. **Workload teams — adopt `Structured.Parameters`**: update
   `ResourceClaim` (or `ResourceClaimTemplate`) authoring to include
   `Structured.Parameters` for the keys the workload cares about
   (e.g., `subnet`, `pkey`, `partition`). Until any device declares
   `sharingAffinity`, these values are persisted but ignored by the
   scheduler — see the "device without SA + claim with SP" row of the
   Compatibility Matrix. This phase can run for as long as workload
   teams need to roll out at their own pace; there is no deadline
   imposed by the platform side.

3. **Platform team — enable the behavioral gate (`DRASharingAffinity`)**
   on the scheduler. Still a no-op until any device declares
   `sharingAffinity.parameterKeys`.

4. **Platform / driver team — publish `sharingAffinity` on devices**:
   upgrade the driver to populate `sharingAffinity.parameterKeys` on
   `ResourceSlice.Device`. Strongly recommended on clean (idle) devices
   first (see "Adding `sharingAffinity` to an in-use device" below).
   From this point on, on `sharingAffinity`-declaring devices, claims
   that provide matching `Structured.Parameters` lock the device, and
   claims that do not provide them are filtered out of those devices.

**Why this order matters**: enabling SA on devices before workloads
have adopted SP causes legacy claims (no SP) to be filtered out of
those devices on their next scheduling attempt, either landing them on
non-SA devices (capacity strand) or driving them to `Pending` if no
non-SA devices exist. Doing SP adoption first is silent-and-safe; doing
SA last is the visible, enforced change. Concretely: clusters should not
flip all relevant devices to `sharingAffinity` until claim authorship
has broadly adopted `Structured.Parameters`, or workloads targeting only
those devices may become unschedulable.

**Recommended rollout sequence (driver-only view)**: To minimize capacity stranding, drivers should
ideally:

1. Wait for a device to be idle (clean).
2. Update the ResourceSlice to include the `sharingAffinity` field.
3. Allow the scheduler to establish the first known lock with a new claim.

During mixed rollouts (some devices with `sharingAffinity`, some without),
alpha provides no scheduler-side steering toward upgraded devices —
affinity-aware preference is delivered in beta (see [Affinity-aware
scoring (planned for Beta)](#affinity-aware-scoring-planned-for-beta)).
Drivers that want predictable rollout behavior should drain devices
before adding `sharingAffinity`, as recommended above.

**Adding `sharingAffinity` to an in-use device**: A driver may add or update
`sharingAffinity` on a device that already has active (bound) ResourceClaims.
This can happen during driver upgrades or when enabling the feature on existing
hardware. The scheduler handles this as follows:

- Pre-existing claims continue to run and are not evicted.
- If any active claim on that device does not provide reconstructable
  affinity values for the required keys, the scheduler marks the device as
  `AffinityStates[deviceID].Unknown = true`.
- A device with `AffinityStates[deviceID].Unknown` set is not eligible for new
  `sharingAffinity` placements, even if it still has nominal shared capacity.
- Once all active claims on that device are released, the device becomes
  clean and subsequent allocations can establish and enforce affinity normally.
- Drivers enabling this feature on existing hardware should prefer doing so on
  clean devices, because alpha intentionally chooses conservative correctness
  over mid-flight reuse of devices whose effective modal state is unknown.

> **Note**: The API server does not cross-validate ResourceSlice updates against
> active ResourceClaims. Enforcing "no `sharingAffinity` changes while claims
> are active" would require a new admission controller with cross-object
> validation, which is fragile and out of scope for this KEP. Drivers should
> avoid adding `sharingAffinity` mid-flight when possible, but the scheduler
> must handle it safely when it occurs.

**Handling missing parameters**: The scheduler treats a device with
`sharingAffinity` as a protected resource. If a device requires both
`subnet` and `pkey` but a claim only provides `subnet`, the device is filtered
out — all declared keys are mandatory. API validation prevents most malformed
`Structured` parameters from reaching the scheduler in the first place; if
any historical claim slips through with missing required keys, the device
is excluded.

**Downgrade**: If feature gates are disabled:

- Disabling `DRASharingAffinity` causes the API server to strip the
  `sharingAffinity` field from new or updated ResourceSlices, and the
  scheduler ignores the field on any persisted ResourceSlices that still
  carry it (e.g., objects created while the gate was on) — devices
  return to unconditional sharing (pre-KEP-5981 behavior). The
  `Structured` API field remains available if
  `DRAStructuredDeviceConfiguration` is still enabled (other consumers
  unaffected).
- Disabling `DRAStructuredDeviceConfiguration` additionally strips the typed
  `Structured` field from new writes on `DeviceConfiguration`. This
  implicitly disables `sharingAffinity` enforcement (no scheduler-readable
  parameters reach the scheduler) and any other feature consuming the
  field.
- The DRA driver becomes the sole authority for enforcing hardware
  compatibility at `NodePrepareResources`.

### Version Skew Strategy

- **kube-apiserver**: Must be upgraded first to accept the new
  `sharingAffinity` API field on `ResourceSlice`.
- **kube-scheduler**:
  - A scheduler that understands this feature enforces `sharingAffinity`, tracks
    `AffinityStates`, and may conservatively mark devices with
    `AffinityStates[deviceID].Unknown` set when effective affinity cannot be reconstructed.
  - An older scheduler ignores `sharingAffinity`. In that skew case, placement
    may be overly permissive and the DRA driver remains the final safety
    backstop during `NodePrepareResources`.
- **kubelet**: No changes required; kubelet does not interpret
  `sharingAffinity`.
- **DRA driver**:
  - Drivers publish ResourceSlices with the `sharingAffinity` field on devices.
  - Drivers must continue validating actual hardware compatibility at prepare
    time, especially during skew where an older scheduler may not enforce the
    affinity constraints.

During version skew, the main outcomes are permissive scheduling by an older
scheduler or conservative filtering by a newer scheduler when affinity state
cannot be reconstructed. Both are operationally safe as long as the driver
continues rejecting incompatible prepare-time configurations.

## Production Readiness Review Questionnaire

### Feature Enablement and Rollback

###### How can this feature be enabled / disabled in a live cluster?

- [x] Feature gate
  - Feature gate names:
    - `DRAStructuredDeviceConfiguration` — gates the typed `Structured` sibling
      on `DeviceConfiguration` (API surface)
    - `DRASharingAffinity` — gates `Device.SharingAffinity` and the
      scheduler logic that reads `Structured.Parameters`; depends on
      `DRAStructuredDeviceConfiguration`
  - Components depending on the feature gates: kube-apiserver, kube-scheduler

###### Does enabling the feature change any default behavior?

No. Devices without `sharingAffinity` behave exactly as before. The feature only affects devices that explicitly opt-in via the new field.

###### Can the feature be disabled once it has been enabled (i.e. can we roll back the enablement)?

Yes. Disabling `DRASharingAffinity` causes:
- API server to strip the `sharingAffinity` field from new or updated
  ResourceSlices before persisting (writes succeed, field is not stored)
- Scheduler to ignore existing `sharingAffinity` fields for future placement
  decisions

Disabling `DRAStructuredDeviceConfiguration` additionally strips the `Structured`
sibling field from `DeviceConfiguration` on new writes, which implicitly
disables `sharingAffinity` enforcement (no scheduler-readable parameters
reach the scheduler) and disables any other feature consuming the field.

Existing allocations continue to work in both cases. New allocations may
become more permissive, so the driver must continue validating compatibility
at prepare time.

###### What happens if we reenable the feature if it was previously rolled back?

The scheduler resumes enforcing `sharingAffinity` for future placement
decisions. Existing allocations are not evicted. However, ResourceSlices that
were created or updated while the gate was disabled will not have the
`sharingAffinity` field (it was stripped by the API server). Drivers must
republish their ResourceSlices with `sharingAffinity` for the feature to take
effect. If there are active allocations whose affinity cannot be reconstructed
at that point, the corresponding devices are treated conservatively until they
become clean.

###### Are there any tests for feature enablement/disablement?

Yes, unit tests will cover the feature gate behavior for API validation and scheduler logic.

### Rollout, Upgrade and Rollback Planning

###### How can a rollout or rollback fail? Can it impact already running workloads?

Rollout failure modes include:

- **Older scheduler after API enablement**: the scheduler ignores `sharingAffinity`
  and placement may be overly permissive. The driver remains the safety backstop
  at prepare time.
- **Newer scheduler enabling conservative handling on legacy in-use devices**:
  devices with non-reconstructable active claims may be filtered until they are
  clean, which can temporarily reduce effective schedulable capacity.

Rollback failure mode: if the scheduler is rolled back while the API server
still serves the field, placement returns to permissive behavior for new
scheduling decisions.

Running workloads are not evicted by this feature; the impact is on future
placement decisions, not on already-running pods.

###### What specific metrics should inform a rollback?

Illustrative rollback signals include:

- `sharing_affinity_filter_mismatch_total` increasing unexpectedly,
- `sharing_affinity_unknown_device_total` remaining elevated after rollout,
- spikes in affinity-related scheduler events or unschedulable pods,
- driver prepare failures due to incompatible or stale configurations.

###### Were upgrade and rollback tested? Was the upgrade->downgrade->upgrade path tested?

Will be tested before beta.

###### Is the rollout accompanied by any deprecations and/or removals of features, APIs, fields of API types, flags, etc.?

No.

### Monitoring Requirements

###### How can an operator determine if the feature is in use by workloads?

Operators can determine usage by inspecting `ResourceSlice` objects that set
`sharingAffinity` and by observing `ResourceClaim`s that include a typed
`Structured` config entry for requests targeting those devices.

###### How can someone using this feature know that it is working for their instance?

A user should be able to observe that:

- compatible claims are eligible to reuse already-locked devices (alpha
  does not actively prefer them; affinity-aware preference lands in beta),
- incompatible claims are filtered before bind/prepare when the scheduler has
  reconstructable affinity state,
- devices with unknown legacy affinity state are conservatively excluded until
  they become clean.

In practice, this should be visible through scheduler logs, scheduler events,
and (where implemented) scheduler metrics.

###### What are the reasonable SLOs (Service Level Objectives) for the enhancement?

This enhancement should not materially regress baseline DRA scheduling latency
for clusters that do not use `sharingAffinity`.

For clusters that do use the feature, the primary objective is **correctness of
compatibility-aware placement** with bounded incremental scheduling overhead.

###### What are the SLIs (Service Level Indicators) an operator can use to determine the health of the service?

Useful SLIs include:

- rate of scheduling attempts filtered due to `sharingAffinity` mismatch,
- rate of devices with `AffinityStates[deviceID].Unknown` set,
- share of successful placements that reuse already-locked compatible
  devices,
- prepare-time rejections by the DRA driver caused by incompatible hardware
  configuration.

###### Are there any missing metrics that would be useful to have to improve observability of this feature?

This feature would benefit from scheduler-observable counters and/or events for:

- `sharing_affinity_filter_mismatch_total`
- `sharing_affinity_filter_missing_parameters_total`
- `sharing_affinity_unknown_device_total`
- `sharing_affinity_compatible_reuse_total`

Exact metric names are illustrative and implementation-specific, but
equivalent observability is strongly recommended.

In addition, user-facing diagnostics should make the reason for filtering clear,
for example:

- missing required `Structured` parameters for request `<name>`,
- device `<id>` is locked to incompatible affinity values,
- device `<id>` has unknown affinity state due to legacy or invalid active
  claims.

### Dependencies

###### Does this feature depend on any specific services running in the cluster?

- DRA must be enabled (GA in 1.34)
- KEP-5075 (Consumable Capacity) for multi-allocatable devices

### Scalability

###### Will enabling / using this feature result in any new API calls?

No new API calls. Affinity data is extracted from ResourceSlice and ResourceClaim
objects already fetched by existing informers.

###### Will enabling / using this feature result in introducing new API types?

No. Only new fields on existing types.

###### Will enabling / using this feature result in any new calls to the cloud provider?

No.

###### Will enabling / using this feature result in increasing size or count of the existing API objects?

- ResourceSlice: Small increase per device with `sharingAffinity` — the
  `parameterKeys` field adds up to 8 fully-qualified key names (capped by
  `SharingAffinityParameterKeysMaxSize = 8`)
- ResourceClaim: Small increase when claims include a `Structured` config
  entry with affinity values (up to 8 string-valued parameters, matching the
  device-side cap)

###### Will enabling / using this feature result in increasing time taken by any operations covered by existing SLIs/SLOs?

Negligible. The Filter phase reads the typed `Structured.Parameters` map
once per candidate device (bounded by the 8-key cap). The affinity comparison
itself is O(k) where k ≤ 8 — a map lookup per key.

###### Will enabling / using this feature result in non-negligible increase of resource usage (CPU, RAM, disk, IO, ...) in any components?

No. The per-component impact is bounded:

- **Scheduler RAM**: `AffinityStates` adds one `map[string]string` (up to 8
  entries) per device with active affinity locks — proportional to active shared
  allocations, not total devices.
- **Scheduler CPU**: Reading the typed `Structured.Parameters` map during
  Filter is a small per-candidate cost (no decode), bounded by the 8-key cap.
- **etcd disk**: Slightly larger ResourceSlice and ResourceClaim objects (see
  API size answer above), bounded by the same caps.

###### Can enabling / using this feature result in resource exhaustion of some node resources (PIDs, sockets, inodes, etc.)?

No.

### Troubleshooting

###### How does this feature react if the API server and/or etcd is unavailable?

Like existing scheduler-driven DRA logic, this feature depends on informer state
and cached API data. Temporary API server or etcd unavailability does not by
itself invalidate already-computed in-memory lock state, but new pods will not
be scheduled during unavailability. Sustained control-plane unavailability may
delay reconciliation of claim release, slice updates, or restart reconstruction.

The driver remains the final enforcement authority at prepare time.

###### What are other known failure modes?

Known failure modes include:

- **Missing required keys**: the claim cannot be matched against a device that
  declares those keys (device is filtered out for that claim).
- **Unknown affinity state**: the device has active allocations whose affinity
  cannot be reconstructed, so it is conservatively filtered until clean.
- **Prepare-time driver rejection**: despite scheduler filtering, the driver may
  still reject an incompatible or stale placement and that rejection is the
  final safety backstop.
- **Partial feature gate enablement**: if the feature gate is enabled on the API
  server but not the scheduler (or vice versa), the `sharingAffinity` field may
  be persisted but not enforced, or enforced but not accepted on writes. Ensure
  the gate is enabled on both `kube-apiserver` and `kube-scheduler`.

###### What steps should be taken if SLOs are not being met to determine the problem?

Recommended debugging flow:

1. Inspect the relevant `ResourceSlice` and confirm the device declares the
   expected `sharingAffinity.parameterKeys`.
2. Inspect the `ResourceClaim` and confirm there is a `Structured` config
   entry covering the relevant request.
3. Verify that every required affinity key is present in
   `Structured.Parameters`.
4. Check whether the target device is already locked to incompatible values
   (lock state is in the scheduler's in-memory cache — check scheduler logs
   for filter reasons mentioning affinity mismatch).
5. Check whether the device is being treated as having **unknown affinity
   state** because of legacy or invalid active claims.
6. Review scheduler logs/events for explicit filter reasons.
7. If the scheduler allowed placement but the driver rejected prepare, inspect
   driver logs to determine whether the issue was stale scheduler state,
   unsupported config, or an actual device-level incompatibility.

User-facing diagnostics should prefer concrete messages over generic
unschedulable errors whenever possible.

## Implementation History

- 2026-03-27: Initial KEP issue created
- 2026-03-30: KEP document drafted
- 2026-04-28: Pivoted from a well-known JSON schema inside
  `OpaqueDeviceConfiguration` to a typed `Structured` sibling field on
  `DeviceConfiguration`, based on wg-device-management feedback that the
  scheduler should not interpret opaque payloads.

## Drawbacks

- Adds a new cache dimension (`AffinityStates`) to the scheduler's allocation
  tracking, increasing the surface area for reconstruction bugs on restart
- Once a device is locked, its effective affinity cannot change until all
  claims on that device are released
- Fragmentation risk remains if affinity values are too fine-grained
- Conservative handling of legacy in-use devices can temporarily strand
  schedulable capacity during rollout or migration
- If a driver declares `sharingAffinity` on a device but no claims ever provide
  a `Structured` config entry, that device becomes effectively unschedulable
  for sharing workloads — all claims are filtered out by the "Strict Gating"
  rule. Drivers should coordinate with workload teams to ensure claims include
  `Structured` parameters before enabling `sharingAffinity` on devices.
- Adds a new typed API field (`Structured` sibling on `DeviceConfiguration`)
  to the `resource.k8s.io` API group — a one-time API surface cost (justified
  by the per-request scoping and reuse by KEP-5993, but still a new field).
- Affinity locks are purely in-memory with no API or status field to inspect
  which devices are locked to which values. Debugging lock state in alpha
  requires scheduler logs; a future enhancement (tracked under Beta
  graduation) is to surface effective lock state via scheduler-side
  metrics, scheduler Events on Pending pods, aggregation over existing
  `ResourceClaim.status.allocation`, or a dedicated scheduler-owned API
  resource — exact mechanism TBD.

## Alternatives

### Well-known JSON schema inside OpaqueDeviceConfiguration

An earlier iteration of this KEP supplied claim-side affinity values via a
scheduler-recognized JSON schema embedded in `OpaqueDeviceConfiguration` —
i.e., a magic `driver: resource.k8s.io` opaque payload with
`apiVersion: resource.k8s.io/v1alpha1, kind: StructuredParameters` that the
scheduler would decode at Filter time:

```yaml
config:
  - requests: ["nic"]
    opaque:
      driver: resource.k8s.io
      parameters:
        apiVersion: resource.k8s.io/v1alpha1
        kind: StructuredParameters
        attributes:
          networking.example.com/subnet: { string: "subnet-A" }
```

**Rejected because**:

- **Violates the contract of `Opaque`**: `OpaqueDeviceConfiguration` is, by
  name and design, opaque to the scheduler — only the driver is supposed to
  read it. Having the scheduler peek into a specific magic driver namespace
  conflicts with that contract and with @johnbelamaric's stated concern that
  the scheduler should not interpret opaque payloads.
- **Weakly typed**: A JSON-schema-inside-string approach is not strongly
  typed. API validation cannot enforce the schema, so most invariants
  shift to scheduler-side runtime decode/validation, multiplying error
  paths (malformed payload, duplicate entries, non-string values, schema
  evolution).
- **Constrained evolution path**: Schema versioning lives inside an opaque blob
  rather than in the Kubernetes API surface, making feature-gating,
  conversion, and review harder.
- **No structural per-request scoping**: Per-request scoping would still
  rely on `DeviceClaimConfiguration.Requests`, but because the apiserver
  cannot introspect opaque payloads, invariants like "at most one
  recognized entry per request" cannot be enforced at admission and must
  be re-checked by the scheduler at runtime — turning what should be a
  structural guarantee into a runtime invariant with its own error paths
  and version-skew concerns.

The proposed approach adds a typed `Structured *StructuredDeviceConfiguration`
sibling field on `DeviceConfiguration`, which is strongly typed, validated by
API server, naturally scoped per request via the existing `Requests` selector,
and does not require the scheduler to interpret opaque payloads.

### Claim-side SharingAffinity (on DeviceRequest)

An alternative design adds a dedicated `SharingAffinity` field directly on
`DeviceRequest` within ResourceClaim:

```go
type DeviceRequest struct {
    // ... existing fields ...
    SharingAffinity *SharingAffinity
}

type SharingAffinity struct {
    AttributeName string          // e.g., "networking.k8s.io/pkey"
    Value         string          // e.g., "0x8001"
    Strategy      SharingStrategy // e.g., "LockOnFirstUse"
}
```

**Rejected because**:

- **Wrong layer**: Sharing affinity is a property of how a *device* can be
  shared (a hardware-modal constraint declared by the driver), not a property
  of an individual request. Putting it on `DeviceRequest` would imply the
  consumer chooses the strategy, when in reality the device dictates which
  keys must agree across consumers. The proposed approach keeps the
  declaration on `Device` (driver-side) and exposes only the per-key *values*
  on the claim side via the typed `Structured` field.
- **Single key only**: The shape above implies one `AttributeName`/`Value`
  pair per request. Multi-key affinity (e.g., `subnet` + `pkey` + `vlan`)
  would require either repeating the field or introducing a list, both of
  which converge structurally to "a map of typed parameters" — which is
  exactly what the chosen `Structured.Parameters` provides without the
  conceptual confusion of a "Strategy" knob on the claim.

### Object Reference-based Affinity Matching

An alternative approach replaces inline affinity values with external object
references. Instead of embedding values in the typed `Structured.Parameters`
map, the claim would reference a CRD (e.g., `NetworkConfiguration`) by name,
and the device would declare which object kinds constrain sharing:

```yaml
# External CRD
kind: NetworkConfiguration
metadata:
  name: subnet-a
spec:
  subnet: 10.0.1.0/24

# ResourceClaim
config:
  objectRefs:           # new field
  - kind: NetworkConfiguration
    name: subnet-a

# Device
commonConfigKind:       # new field
- NetworkConfiguration
```

**Rejected because**:
- Requires new fields on both ResourceClaim (`objectRefs`) and Device
  (`commonConfigKind`), whereas the chosen approach adds a typed sibling on
  `DeviceConfiguration` and a single `SharingAffinity` field on `Device`
- Requires external CRD definitions, adding operational burden for cluster
  administrators
- Multi-dimensional affinity: A device may need affinity on multiple independent axes
  (e.g., subnet + VLAN). With object references, each axis would need its own CRD.
- Inline values are easier to reason about and validate than indirect object
  references, which would also raise authorization and lifecycle concerns
  (who owns the CRD instance? what happens when it is deleted while claims
  reference it?).

### CEL-based Affinity Matching

An alternative approach uses CEL expressions to evaluate affinity compatibility,
rather than introducing new structured API fields. Two variants were considered:

**Variant A: Claim-to-claim CEL matching**

Allow CEL expressions in a ResourceClaim to reference other claims' allocations
on the same device. For example:

```yaml
constraints:
  - cel:
      expression: >
        device.allocations.all(a,
          !has(a.config.subnet) || a.config.subnet == "subnet-X")
```

**Rejected because**:
- Creates a circular dependency: Claim A's eligibility depends on Claim B's
  allocation, and vice versa. The scheduler cannot evaluate both simultaneously.
- CEL evaluation order becomes undefined—the result depends on which claim is
  evaluated first, making scheduling non-deterministic.
- The CEL environment would need to expose `device.allocations`, a runtime
  collection of other claims' configs. This is a fundamentally different
  evaluation model from today's single-device CEL selectors.

**Variant B: Driver-published CEL lock expressions on ResourceSlice**

The driver publishes a CEL expression on the ResourceSlice that evaluates
whether a claim is compatible with the device's current lock state:

```yaml
devices:
  - name: eth1
    sharingAffinity:
      lockExpression: >
        device.affinityLock['subnet'] == '' ||
        device.affinityLock['subnet'] == claim.AffinityValues['subnet']
```

**Rejected because**:
- `device.affinityLock` is runtime scheduler state, not a static device
  attribute. Exposing it in CEL requires extending the evaluation context to
  include the scheduler's in-memory `AllocatedState`, which breaks the current
  model where CEL only evaluates against the ResourceSlice snapshot.
- `claim.AffinityValues` is not currently part of the CEL evaluation context
  either. Adding it requires changes to the CEL environment definition, the
  scheduler's expression compiler, and the cost estimator.
- CEL expressions are powerful but opaque to the scheduler—it cannot extract
  *which* keys constrain sharing or *what* values to record in `AllocatedState`.
  The scheduler would need to both evaluate the expression AND separately track
  lock state, duplicating logic.
- While Kubernetes is adopting CEL broadly (ValidatingAdmissionPolicy, DRA
  selectors), those use cases evaluate static data. Affinity matching requires
  reasoning about mutable runtime state, which is a qualitatively different
  problem better served by a purpose-built mechanism.

## Future Enhancements

The following ideas are out of scope for alpha but are worth exploring in
beta/GA based on real-world feedback:

### Affinity-aware scoring (planned for Beta)

This KEP's Filter phase is sufficient for **correctness** — an incompatible
locked device is filtered out, and any remaining candidate produces a valid
allocation. It is not, however, sufficient for **packing**, at either
scope:

- **Cross-node**: stock Kubernetes scorers do not factor in
  `sharingAffinity` when scoring nodes yet. A `subnet=X` claim
  is just as likely to land on a node with a clean device as on a node
  that already has a compatibly-locked device.
- **Within-node**: once a node is selected, the DRA structured-parameters
  allocator picks the first feasible device (first-fit). Among multiple
  feasible devices on the chosen node — say, one already locked to a
  compatible affinity value and one clean — there is no preference logic;
  the allocator may pick whichever appears first in the ResourceSlice.

The `DynamicResources` plugin already implements scoring for DRA
(shipped in K8s 1.35). Prioritized List (KEP-4816) and Extended
Resources (KEP-5004) already contribute their own additive terms to
`computeScore`. This is the canonical extension point for new DRA
features that need scoring.

KEP-5981 plans to add a sharing-affinity term to `computeScore` as a
beta deliverable, following the same per-feature additive pattern. The
broader "general-purpose DRA scoring" discussion continues in
[kubernetes/enhancements#4970](https://github.com/kubernetes/enhancements/issues/4970),
but this KEP's contribution does **not** block on a unified framework
or on #4970 producing a generic API. Doing per-feature scoring here is
consistent with how Prioritized List and Extended Resources already
shipped scoring, and avoids stranding sharing affinity behind a
multi-feature design effort.

The detailed score terms, weights, and tie-breakers will be designed
and proposed as part of the beta graduation. Alpha intentionally does
not commit to specific scoring shape so that beta has freedom to
incorporate operational feedback from alpha.

**Within-node device selection** is addressed similarly by extending the
allocator's per-node selection logic so that, among feasible devices for
a chosen sub-request, the allocator prefers a locked-compatible device
over a clean one. This is a localized change to the structured-parameters
allocator and is also a beta deliverable.

**Alpha contract: correctness only**. Packing of any kind (within-node or
cross-node) is best-effort first-fit and is documented as a known
limitation in [Risks and Mitigations](#packing-depends-on-dra-aware-scoring-alpha-limitation).

### Priority-based Lock Preemption

This section addresses a deliberate alpha limitation: alpha enforces lock
compatibility, but does not provide any mechanism for a
higher-priority Pod to break an incompatible lock.

Standard Kubernetes preemption is blind to affinity locks. It triggers on
*resource shortage* (insufficient CPU, memory, or device slots), not on
qualitative state mismatch. This creates a critical gap:

1. **Invisible shortage**: A NIC has 15/16 slots available but is locked to
   Subnet-X. A high-priority Pod needs Subnet-Y. The scheduler sees plenty of
   capacity → preemption is never triggered. The Pod is simply unschedulable.

2. **Wrong victim selection**: Even if preemption were triggered by an unrelated
   shortage, victim selection asks "which Pods free up slots?" not "which Pods
   clear the lock?" The scheduler might preempt an unrelated Pod, freeing a
   slot on a device still locked to the wrong subnet.

3. **Permanent poisoning**: Without lock-aware preemption, a single low-priority
   Pod can hold a high-capacity device hostage indefinitely.

**Lock-aware preemption** (targeted for Beta) extends the scheduler's PostFilter
phase:

1. **Detection**: When a Pod fails Filter specifically due to
   `SharingAffinityMismatch`, the PostFilter identifies the device and its
   current lock-holder claims.
2. **Evaluation**: It calculates the collective priority of all Pods holding
   claims that share the lock. If the incoming Pod's priority exceeds the
   group's maximum priority, preemption is viable.
3. **Action**: The scheduler preempts all lock-holder Pods on the device,
   releasing their claims and clearing the affinity lock. The device returns
   to a clean state for the high-priority Pod.

This is scoped for Beta because the core Filter/Reserve mechanism must
be proven in Alpha first, and lock-aware preemption requires careful
integration with the existing DRA preemption path. Key design considerations
include:

- **Victim minimization**: When multiple devices could satisfy the incoming Pod,
  the preemption logic should prefer the device with the fewest lock-holding
  Pods to minimize disruption.
- **Atomicity**: Preemption in Kubernetes is asynchronous—victim Pods are
  deleted but do not disappear instantly. During the eviction window the old
  lock is still active, so a newly-arriving compatible Pod could land on the
  device and re-establish the lock, creating a preemption cascade. Standard
  preemption solves the analogous problem with NominatedNode; lock-breaking
  would need a similar mechanism (e.g., marking the device's lock as
  "transitioning to the new value for the preempting Pod") so that future
  scheduling cycles treat the device as locked to the new value, filtering
  out Pods compatible only with the old lock.

### SharingStrategy (`CanSetLock` / `NeverSetLock`)

Alpha intentionally does not let claims control whether they may establish
a new lock on a clean device. Any compatible claim can set the initial lock,
and subsequent compatible claims can then reuse that locked device.

A future enhancement could add an explicit **SharingStrategy** on the claim
side to control lock-setting behavior. Two candidate strategies are:

- **`CanSetLock`** (default): The claim may land on a clean device and
  establish the lock. This matches the alpha behavior.
- **`NeverSetLock`**: The claim may only be allocated to a device that already
  has a matching lock established by another claim. This is useful for
  background or batch jobs that should never consume a clean device and
  potentially fragment capacity. **Caveat**: `NeverSetLock` is a follower-only
  strategy — it requires at least one `CanSetLock` claim to establish the lock
  first. If no device is locked to the requested value, a `NeverSetLock` pod
  will remain unschedulable indefinitely. Implementations should document this
  dependency clearly and consider surfacing a scheduling event when a pod is
  blocked waiting for a lock that no leader has established.

If introduced in beta or later, the scheduler would evaluate this policy before
capacity and key matching for unlocked devices. A claim with `NeverSetLock`
would reject an unlocked device immediately, then continue searching for an
already-locked compatible device.

This is deferred from alpha to keep the initial scope focused on the core
problem: driver-declared sharing constraints plus scheduler-enforced lock
tracking via structured parameters.

### Soft / Preferred Affinity Keys

The Alpha design enforces hard all-or-nothing matching: all declared
`parameterKeys` must match or the device is filtered out. Real-world hardware
may have hierarchical constraints where some keys are strict sharing
requirements (e.g., Subnet) and others are scheduling preferences (e.g.,
Traffic-Class or bandwidth profile).

A future enhancement could add a `required` vs `preferred` flag on individual
entries in `parameterKeys`:

- **`required`** (default): Mismatch → device filtered out (current behavior)
- **`preferred`**: Mismatch → device passes Filter but is deprioritized in device selection

This would allow device selection to optimize for Traffic-Class alignment while
only enforcing hard locks on Subnet. The lock itself would only be set for
`required` keys — `preferred` keys would remain advisory and never block
scheduling. This avoids complicating the atomic lock model while still
enabling soft optimization.

### Typed Affinity Values Beyond Strings

Alpha limits affinity matching to string equality (`map[string]string`), which
covers all known use cases (subnets, bitstreams, partition keys). Non-string
types could be added in the future if concrete use cases arise.

## Infrastructure Needed

None
