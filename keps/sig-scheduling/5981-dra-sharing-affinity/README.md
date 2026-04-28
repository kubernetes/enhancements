# KEP-5981: DRA Sharing Affinity for Conditional Fungibility

<!-- toc -->
- [Release Signoff Checklist](#release-signoff-checklist)
- [Summary](#summary)
- [Motivation](#motivation)
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
  - [Placeholder Pattern Workaround](#placeholder-pattern-workaround)
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

Real-world hardware is often **modal**: once partially allocated, the device
requires all subsequent consumers to share a specific configuration. For
example:

- **Multi-pod NIC sharing**: A network DRA driver shares a NIC across 16 pods,
  but all pods must belong to the same subnet. Once the first pod configures the
  NIC for Subnet A, the remaining 15 slots are restricted to Subnet A.
- **FPGA bitstream sharing**: An FPGA can serve multiple inference pods, but all
  must use the same bitstream. Once bitstream-ml-v2 is loaded, other pods
  needing bitstream-crypto-v1 must use a different FPGA.

This KEP introduces a `SharingAffinity` field in the ResourceSlice `Device`
spec that allows drivers to declare which parameter keys constrain
sharing compatibility. The scheduler's `AllocatedState` is enhanced to track
both consumed capacity and the affinity values that lock a device to a
particular sharing group, enabling it to gate remaining capacity and
pack compatible workloads onto already-locked devices.

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
physical devices often have a "modal" constraint: once partially allocated,
the device requires all subsequent consumers to share a specific configuration
(see [Summary](#summary) for concrete examples).

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
   with race conditions and ResourceSlice churn

The scheduler's `AllocatedState` currently tracks consumed capacity but not the
affinity values that determine sharing compatibility. This KEP closes that gap.

### Goals

- Enable the scheduler to gate remaining capacity on a device based on a
  required sharing attribute
- Provide a mechanism for drivers to signal compatibility requirements for
  shared hardware via `SharingAffinity` in ResourceSlice
- Minimize fragmentation of cluster resources by enabling the scheduler to
  pack workloads with identical sharing requirements onto already-locked devices
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
  Alpha enforces compatibility and improves packing, but does not yet guarantee
  that a higher-priority Pod can displace an incompatible lock-holder.

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

1. **First claim**: The scheduler decodes the claim's well-known structured
   parameters from opaque config, reads the affinity values for the specified
   attribute key(s), and records them in `AllocatedState` alongside consumed
   capacity
2. **Subsequent claims**: The scheduler checks if the new claim's affinity values match those recorded in `AllocatedState`
3. **Mismatch**: If values don't match, the device is skipped (try another device)
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
requested values for those keys by decoding a well-known JSON schema stored
inside `OpaqueDeviceConfiguration`.

The claim's opaque config (`DeviceConfiguration.Opaque.Parameters`) is a
`runtime.RawExtension`—raw bytes the scheduler cannot parse generically. For
this feature, drivers that want sharing affinity encode scheduler-readable
parameters using a community-governed JSON schema, similar to the pattern in
[k8s.io/dynamic-resource-allocation/api/metadata](https://github.com/kubernetes/kubernetes/tree/master/staging/src/k8s.io/dynamic-resource-allocation/api/metadata).
This avoids any API changes to `DeviceConfiguration` while giving the scheduler
a decodable format for affinity-relevant parameters.

**Approach: Well-known JSON schema inside OpaqueDeviceConfiguration**

The schema reuses the same qualified key naming convention as `ResourceSlice`
attributes and follows a `DeviceAttribute`-like envelope. In alpha,
`sharingAffinity` matching is limited to string-valued attributes for the
keys referenced by `sharingAffinity.parameterKeys`, which keeps equality
semantics simple and aligns with the scheduler's in-memory lock representation.

```json
{
  "apiVersion": "resource.k8s.io/v1alpha1",
  "kind": "StructuredParameters",
  "attributes": {
    "networking.example.com/subnet": {"string": "subnet-X"},
    "networking.example.com/pkey": {"string": "0x8001"}
  }
}
```

The scheduler decodes this JSON from the opaque blob and extracts string
values for the keys listed in the device's `sharingAffinity.parameterKeys`. The
decoding overhead is small compared to the overall scheduling effort.

If drivers also need additional, differently structured configuration
parameters (e.g., MTU, QoS settings), users provide two config entries in
the claim: one using the standard schema (scheduler reads) and one using the
vendor format (driver reads). The scheduler only considers configurations
matching the well-known schema.

**Key advantages:**
- **No API changes** to `DeviceConfiguration` — the feature uses existing
  opaque config with a well-known schema
- **No duplication** for simple cases — the driver can read the same structured
  parameters it programs (e.g., subnet, PKey)
- **Extensible** — the well-known schema can support future scheduler-readable
  hints beyond sharing affinity

**Alpha StructuredParameters Contract**

For alpha, the scheduler-readable structured-parameters format is a
scheduler-recognized contract with the following rules:

1. **Recognition**: The scheduler recognizes a config entry as structured
   parameters only when the `opaque.driver` is `resource.k8s.io` and the
   embedded payload has `apiVersion: resource.k8s.io/v1alpha1` and
   `kind: StructuredParameters`.
2. **Per-request uniqueness**: For a given request, there must be at most
   one structured-parameters config entry targeted at that request. Multiple
   matching entries for the same request are invalid.
3. **Coexistence with driver config**: The structured-parameters entry may
   coexist with one or more driver-specific opaque config entries for the same
   request. The scheduler reads only the recognized structured-parameters
   entry. The driver-specific entries are ignored by the scheduler.
4. **Conflict handling**: If the same logical setting is encoded both in
   `StructuredParameters` and in driver-specific config, the scheduler uses only
   the `StructuredParameters` value for placement decisions and does not attempt to
   compare or reconcile the driver-specific opaque payload. If a conflict exists,
   the driver should reject the request during `NodePrepareResources` with a clear error,
   rather than silently accepting divergent values.
5. **String-only affinity values in alpha**: For any key referenced by
   `sharingAffinity.parameterKeys`, the recognized structured-parameters entry
   must provide a `string` value in alpha. Other value types are not matched in
   alpha and are treated as invalid for `sharingAffinity` scheduling.
6. **Malformed payloads**: If a recognized structured-parameters entry is
   malformed, has the wrong schema, or cannot be decoded, it is treated as
   invalid for scheduling purposes.
7. **Missing recognized entry**: If a claim targets a device with
   `sharingAffinity` but does not provide a recognized structured-parameters
   entry for that request, the device is filtered out. This does not make the
   claim universally unschedulable. It only makes the claim ineligible for devices
   that declare `sharingAffinity`. If all feasible devices for the request declare
   `sharingAffinity`, then the request may remain unschedulable until a recognized
   `StructuredParameters` entry is provided or non-sharing-affinity capacity is available.
8. **Validation intent**: API validation should reject malformed or duplicate
   structured-parameters entries when feasible. The scheduler must still
   handle invalid persisted objects defensively and deterministically.

In alpha, this keeps the contract explicit without introducing a new API field:
the scheduler depends only on a single, community-governed, recognized payload
shape and ignores all other opaque config.

For alpha, `StructuredParameters` is a scheduler-recognized sub-protocol
defined by this KEP. The scheduler interprets only payloads explicitly
recognized as `opaque.driver: resource.k8s.io` together with the embedded
`apiVersion`/`kind` for `StructuredParameters`; all vendor-defined opaque
payloads remain opaque. The sub-protocol is versioned via the embedded
`apiVersion` and future revisions must define compatibility and upgrade
behavior explicitly.

**Alpha scope**

Alpha fully resolves the design around driver-side placement and the
structured-parameters approach described above. Claims do not control
lock-setting behavior in alpha: any compatible claim may establish the initial
lock on a clean device. Claim-side lock-setting policy (for example,
`CanSetLock`/`NeverSetLock`) is deferred to [Future
Enhancements](#future-enhancements).

In other words, alpha standardizes driver-declared compatibility keys, a
scheduler-recognized structured-parameters contract, and correct lock
enforcement / packing behavior — but intentionally stops short of
lock-breaking preemption semantics.

**Alpha limitations**

Alpha provides correct lock enforcement and better packing, but it does
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
sets `sharingAffinity.parameterKeys: ["networking.example.com/pkey"]`. The scheduler finds a node where
a NIC has enough capacity and is either "unlocked" or already locked to that
specific PKey.

- Pod A (pkey-0x8001) gets allocated to mlx5_0 → mlx5_0 is now locked to pkey-0x8001
- Pod B (pkey-0x8001) arrives → matches affinity, shares mlx5_0
- Pod C (pkey-0x8002) arrives → affinity mismatch, gets mlx5_1 instead

#### Story 2: FPGA Bitstream Sharing

An inference service uses FPGAs to accelerate a specific model. Loading a
bitstream takes several seconds. The driver sets
`sharingAffinity.parameterKeys: ["fpga.example.com/bitstream"]`. The scheduler ensures new Pods
for this model are scheduled onto FPGAs that already have the bitstream loaded,
even if other "fresh" FPGAs are available.

- Pod A (bitstream-ml-v2) gets the FPGA → locks to bitstream-ml-v2
- Pod B (bitstream-ml-v2) shares the same FPGA
- Pod C (bitstream-crypto-v1) must wait or use a different FPGA

#### Story 3: Single-subnet NIC Sharing

A network DRA driver advertises NICs that can be shared across up to 16 pods,
but only if pods belong to the same subnet. The driver sets
`sharingAffinity.parameterKeys: ["networking.example.com/subnet"]`.

- Pod A (subnet-X) gets allocated to eth1 → eth1 is now locked to subnet-X
- Pod B (subnet-X) arrives → matches affinity, shares eth1
- Pod C (subnet-Y) arrives → affinity mismatch, gets eth2 instead

### Notes/Constraints/Caveats

- **Affinity is set by the first compatible claim on a clean device**: Once a
  device is allocated with an affinity value, that value is locked until all
  claims release the device.
- **Attribute keys must be declared**: The device's
  `sharingAffinity.parameterKeys` lists which attribute keys constrain sharing;
  claims must provide values for all of these keys in the well-known structured
  parameters or the device is filtered out.
- **Multiple keys**: If multiple attribute keys are specified, ALL must match
  (both presence and value).
- **Extra keys in claim**: If a claim's structured parameters contain keys beyond
  what the device declares in `parameterKeys`, the extra keys are ignored
  for that device. Only the device's declared keys are evaluated. This allows
  "generic" claims to work across devices with different sharing requirements
  (e.g., a claim with both `subnet` and `vlan` can match a device that only
  constrains on `subnet`).
- **String-only matching in alpha**: For keys referenced by
  `sharingAffinity.parameterKeys`, the scheduler only matches `string` values
  in alpha. If a required key is present with a non-string value, the device is
  filtered out for that claim.
- **Missing keys in claim**: If the claim does not provide a value for a key
  the device declares in `parameterKeys`, the device is filtered out (see
  Filter phase).
- **Malformed structured parameters**: If the scheduler-recognized
  `StructuredParameters` entry is malformed, undecodable, or uses the wrong
  schema, it is treated as invalid and the claim cannot use devices that rely
  on `sharingAffinity`.
- **Duplicate structured parameters for one request**: If more than one
  recognized `StructuredParameters` config entry targets the same request, the
  claim is treated as invalid for `sharingAffinity` scheduling until corrected.
- **Multi-request claims (per-request scoping)**: If a claim requests multiple
  devices (e.g., `mgmt-nic` and `data-nic`), each `DeviceClaimConfiguration`
  block targets specific requests via its `requests` slice. Different config
  blocks can specify different structured parameters for different requests. This
  means `mgmt-nic` can be locked to Subnet-A while `data-nic` is locked to
  Subnet-B within the same claim — there is no cross-talk between requests.
- **Empty affinity**: Devices without `sharingAffinity` behave as before — any
  claim can share them regardless of whether it provides structured parameters.
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
| 5 legacy claims, affinity unknown | Claim without structured parameters | **Filtered out**. Missing required scheduler-readable affinity information. |
| Legacy claims drained; device now clean | Claim with `subnet: A` | Lock set to `subnet: A`; device now locked. |
| Device locked to `subnet: A` | Claim with `subnet: A` | Allowed (values match). |
| Device locked to `subnet: A` | Claim with `subnet: B` | **Rejected** (mismatch with lock). |
| All claims released | — | Device fully clean and eligible to establish a new lock. |

Legacy claims continue to run and are not evicted. However, until all unknown
allocations on a `sharingAffinity` device are released, the scheduler does not
assume it knows the device's effective modal state.

#### Compatibility Matrix

To clarify the interaction between claims and devices, the following matrix
outlines how the scheduler and driver evaluate candidates based on whether
`SharingAffinity` (SA) is declared on the device and whether
`StructuredParameters` (SP) are provided in the claim:

| Scenario | Device SA | Claim SP | Scheduler Outcome | Driver Outcome |
|---|---|---|---|---|
| **Standard Feature Use** | Yes | Yes | **Match enforced.** Values match lock + capacity available → scheduled. | **Validates** hardware mode matches claim config at `NodePrepareResources`. Rejects if stale or inconsistent. |
| **Strict Gating** | Yes | No | **Filtered out.** Device excluded — requires affinity signal the claim does not provide. | **N/A** — claim never reaches the driver for this device. |
| **Legacy Device Transition** | Yes (newly added) | Yes | **Filtered out** while legacy claims are active (`Unknown: true`). Allowed once device drains clean. | **Validates** as normal once claim reaches the driver. During transition, driver continues serving legacy claims. |
| **Permissive Sharing** | No | Yes | **Allowed.** Device has no `sharingAffinity`; structured parameters are not evaluated for affinity. Standard capacity matching applies. | **Must enforce** hardware compatibility independently. Scheduler provides no affinity gating for this device. |
| **Legacy/Basic** | No | No | **Allowed.** Standard DRA capacity and attribute matching. | **Must enforce** hardware compatibility independently. This is the pre-KEP-5981 behavior. |

The top rows show the scheduler as the primary enforcer with the driver as
a backstop. The bottom rows show the driver as the sole enforcer with
the scheduler being permissive. The transition row shows the scheduler being
conservative (filtering) while the driver continues serving existing
workloads.

### Risks and Mitigations

#### Fragmentation (Poisoning)

**Risk**: One Pod with a unique affinity value could "lock" a high-capacity
device, preventing other more common workloads from using the remaining 90%
capacity.

**Mitigation**: The allocator's device-selection preference will prioritize packing compatible workloads
onto already-locked devices before consuming "clean" (unlocked) devices. This
minimizes the number of devices locked to a single affinity group.

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

**Mitigation (Alpha)**: The allocator's device-selection preference reduces the probability by packing
compatible workloads and preserving clean devices. However, this is a soft
mitigation — it does not guarantee that a clean device will always be available,
and alpha does not provide lock-breaking preemption.

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

#### Unexpected Affinity Values

**Risk**: A claim specifies an unexpected or unique affinity value (e.g., an
arbitrary subnet GUID or name), further fragmenting devices by locking them to
rare values.

**Mitigation**: In many cases, affinity values are externally defined (subnet
names, partition keys) and cannot be validated by the driver. The primary
mitigation is the allocator's device-selection preference: by packing compatible workloads onto
already-locked devices before consuming clean ones, the scheduler naturally
limits fragmentation even when affinity values are unpredictable. Additionally,
cluster administrators can use `DeviceClass` CEL selectors to restrict which
attribute values are accepted where domain-specific validation is feasible.

#### Memory Overhead

**Risk**: Affinity values accumulate in `AllocatedState`, increasing memory usage.

**Mitigation**: In alpha, affinity values are stored as small strings (for
example subnet or PKey identifiers), capped at 8 attribute keys per device.
Per-device overhead is bounded at 8 key-value pairs in
`AllocatedState.AffinityStates`, and entries are cleared when all claims release
the device. The total overhead is proportional to active shared allocations,
not total devices.

## Open Questions

The following design choices are open for WG Device Management and sig-scheduling
discussion. Reviewer input is welcomed.

### Structured Parameters: inside opaque config vs. dedicated typed field

The current design embeds a well-known JSON schema inside `OpaqueDeviceConfiguration`,
which the scheduler recognizes via `opaque.driver: resource.k8s.io` plus an
`apiVersion` / `kind` discriminator in the payload. This approach was suggested
to avoid adding a new API field on `ResourceClaim`. An alternative is to
introduce a dedicated structured field outside opaque config — either on
`DeviceRequest` or as a peer to `OpaqueDeviceConfiguration` — that the scheduler
reads natively.

Both approaches have been raised in WG feedback. The tradeoffs:

| Dimension | Well-known schema inside opaque (current) | Dedicated structured field outside opaque |
|:---|:---|:---|
| API change to `ResourceClaim` | None | New typed field required |
| Validation | Runtime scheduler-side recognition; malformed payload handling specified in this KEP | Native API admission validation; schema enforced by API server |
| Versioning | Embedded `apiVersion`/`kind` in payload | Native Kubernetes API versioning |
| Tooling visibility | `kubectl` sees opaque bytes; users must know the convention | `kubectl` and other tooling see typed fields directly |
| Contract surface | Recognition rules, duplicate handling, and conflict semantics described in this KEP | Typed Go struct in `staging/src/k8s.io/api/resource` |
| Strictness | Convention-based; driver-private and scheduler-recognized payloads coexist in the same slice | Explicit separation of driver-private from scheduler-readable configuration |
| Migration cost to typed field later | Requires migration path to a typed field if we change approach | Already typed |
| Barrier to adoption | Low — reuses existing opaque config path | Requires API review and approval |

The current design lands on the opaque-with-schema approach to minimize API
surface and enable alpha without a new typed field. WG feedback has noted that a
dedicated structured path may be more appropriate long-term given that the data
is first-class scheduler input, not driver-private configuration. We welcome
guidance on whether to ship alpha with the opaque-schema approach and migrate
later, or invest in the dedicated field up front.

### Per-device scoring vs. node-level scoring

DRA's Score extension point scores nodes, not individual devices. This KEP's
Filter phase is sufficient for correctness — an incompatible locked device is
filtered out, and any remaining candidate (locked-compatible or unlocked)
produces a valid allocation. The open question is how far to take scoring for
packing efficiency.

The current design describes an allocator-internal device-selection preference
(prefer locked-compatible over clean within a node) as an optimization, not a
correctness requirement. An explicit node-level Score contribution has also been
proposed: rank a node higher when it has at least one device compatibly locked
to the claim's affinity values, lower when only clean devices are available.

What we lose without any scoring (Filter-only):

- **Cross-node fragmentation**: Without node-level scoring, the scheduler may
  place compatible claims on different nodes even when one node has a
  compatibly-locked device and another has only clean devices. This spreads
  locks across more physical devices than necessary, defeating the packing
  goal of the feature.
- **Mixed-rollout steering**: During driver upgrades where some devices have
  `sharingAffinity` and some do not, Filter alone cannot steer claims toward
  upgraded devices.

What we lose without per-device (within-node) scoring:

- **Within-node fragmentation**: When a node has multiple `sharingAffinity`
  devices, some locked-compatible and some clean, the allocator may consume a
  clean device when a compatible locked one exists on the same node. This
  locks an additional device unnecessarily.
- **Bin-packing granularity**: Among multiple locked-compatible devices on the
  same node, there is no way to prefer the most-full one (minimizing the
  number of partially-filled devices).
- **Score dilution**: A node with 1 compatibly-locked device ranks the same as
  a node with many, making tie-breaking across nodes coarse.

Patrick Ohly's feedback in the WG discussion was that per-device scoring risks
a combinatorial explosion in large clusters and recommended node-level scoring
as the primary mechanism. Node-level scoring provides most of the packing
benefit with bounded computational cost. Per-device preference can be retained
as an allocator-internal optimization where feasible.

The open questions:

1. Is node-level Score sufficient for alpha, with allocator-internal device
   preference as a best-effort optimization?
2. Is the within-node fragmentation listed above an acceptable alpha
   limitation, or does it warrant a more formal per-device scoring mechanism?
3. If per-device scoring is pursued later, where should it live — extension of
   the DRA allocator, a new Score sub-interface, or elsewhere?

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
    // recognized StructuredParameters entry. Support for additional value types
    // is deferred.
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

The scheduler derives affinity locks **solely from active claims' structured
parameters** (decoded from the well-known JSON schema in opaque config) — not
from device attributes on the ResourceSlice. The driver is NOT required to write
locked affinity values back to the ResourceSlice.

- The ResourceSlice declares *which* keys constrain sharing (`parameterKeys`)
- The claims declare *what* values they need (via well-known structured parameters)
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

- **Scheduler guarantee**: when it has recognized structured parameters for all
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
3. The claim has exactly one scheduler-recognized `StructuredParameters`
   config entry targeting the relevant request
4. That entry can be decoded successfully using the well-known schema
5. The claim provides values for ALL keys in `sharingAffinity.parameterKeys`
   (missing key → device is not a candidate)
6. For each required affinity key, the recognized entry provides a string
   value (non-string values are invalid in alpha)
7. The device's `AffinityStates[deviceID].LockedAffinity` is either empty (unlocked) OR matches the
   claim's affinity values for ALL keys

The scheduler identifies the structured-parameters entry by `opaque.driver:
resource.k8s.io` plus `apiVersion: resource.k8s.io/v1alpha1` and
`kind: StructuredParameters` in the embedded payload. Driver-specific config
entries are ignored by the scheduler.

If a device has `AffinityStates[deviceID].Unknown` set, or if a required request has no
recognized structured-parameters entry, more than one recognized entry, an
entry that fails schema/decoding checks, or a required affinity key with a
non-string value, the device is filtered out for `sharingAffinity`
scheduling. This is the safe default: the driver declared that sharing
requires specific scheduler-readable parameters, and a scheduler that cannot
reconstruct the current or requested affinity state cannot evaluate placement
safely. Claims that do not need sharing-constrained devices should target
devices without `sharingAffinity`.

**Device selection preference**: DRA's Score extension point scores *nodes*,
not individual devices. Within a node, device selection is handled by the
allocator's internal logic. For correctness, the Filter phase alone is
sufficient — incompatible locked devices are excluded, and any remaining
candidate (locked-compatible or unlocked) produces a valid allocation.

For packing efficiency, the allocator MAY internally prefer devices in this
order when multiple candidates are feasible:

1. A device already locked to a compatible affinity value — reuses an existing
   lock and avoids consuming a clean device.
2. A clean (unlocked) device with `sharingAffinity` — can establish a new lock
   and enable packing for future claims.
3. A device without `sharingAffinity` — the scheduler has no affinity
   enforcement for this device, so packing benefits are lost.
4. An incompatible locked device, or a device with
   `AffinityStates[deviceID].Unknown` set, is not considered because it was
   already filtered out.

This preference reduces fragmentation by preserving unlocked devices for future
workloads with different affinity values. During mixed rollouts (some devices
with `sharingAffinity`, some without), it naturally steers affinity-aware claims
toward upgraded devices. This preference is an optimization; it is not required
for correctness in alpha. The exact implementation strategy (e.g., ordering
candidates before constraint solving) is implementation-defined.


##### Reserve Phase: Tentative Locking

Once a node/device is selected, the Reserve plugin establishes a "tentative
lock" in the scheduler cache before the Binding phase:

1. Scheduler evaluates a multi-allocatable device with `sharingAffinity`
2. If device has no existing allocations (unlocked):
   - Extract affinity values for `sharingAffinity.parameterKeys` from the claim's
     structured parameters (decoded from opaque config)
   - Record values in `AllocatedState.AffinityStates[deviceID].LockedAffinity`
   - Proceed with allocation (device is now tentatively locked)
3. If device has existing allocations (locked):
   - Compare claim's affinity values against `AllocatedState.AffinityStates[deviceID].LockedAffinity`
   - If all keys match: proceed with allocation (pack onto locked device)
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

1. On startup, the scheduler iterates all `Bound` ResourceClaims (same path as
   existing `GatherAllocatedState()` for capacity reconstruction).
2. For each bound claim, check if the allocated device has `SharingAffinity`
   defined in the corresponding ResourceSlice.
3. If yes, attempt to decode the claim's opaque config using the well-known JSON
   schema and extract the required structured parameters.
4. If decoding succeeds and all required affinity keys are present as strings,
   populate `AffinityStates[deviceID].LockedAffinity` with those values.
5. If the claim has no recognized `StructuredParameters` entry, malformed
   structured parameters, non-string values for a required affinity key, or
   multiple recognized structured-parameters entries for the same request,
   set `AffinityStates[deviceID].Unknown = true` and log a warning. The scheduler
   must not infer lock state from ambiguous or invalid data.
6. If multiple claims share the same device and any one of them causes the
   device to become unknown, the device remains with `AffinityStates[deviceID].Unknown` set
   until all claims on that device are released.
7. If multiple reconstructable claims share the same device, verify their values
   are consistent (they must be, by construction—but log a warning if not).

This follows the same pattern used to reconstruct `AggregatedCapacity` from
bound claims on startup. No new API calls are needed; the data is already
available from the ResourceClaim spec and ResourceSlice spec cached by the
scheduler's informers.


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
      # Well-known structured parameters (scheduler decodes for affinity matching)
      - requests: ["nic"]
        opaque:
          driver: resource.k8s.io
          parameters:
            apiVersion: resource.k8s.io/v1alpha1
            kind: StructuredParameters
            attributes:
              networking.example.com/subnet:
                string: "subnet-X"
      # Driver-specific opaque config (scheduler ignores this)
      - requests: ["nic"]
        opaque:
          driver: networking.example.com
          parameters:
            apiVersion: networking.example.com/v1
            kind: NICConfig
            vlanId: 100
```

> **Note**: The first config block uses the well-known `StructuredParameters`
> schema with `driver: resource.k8s.io`, which the scheduler recognizes and
> decodes for affinity matching. The second config block is standard opaque
> driver config that only the driver reads. For simple cases where the driver
> can read both, a single well-known config block may be sufficient.

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

A matching claim provides both values in the scheduler-recognized structured
parameters:

```json
{
  "apiVersion": "resource.k8s.io/v1alpha1",
  "kind": "StructuredParameters",
  "attributes": {
    "networking.example.com/subnet": {"string": "subnet-a"},
    "networking.example.com/pkey": {"string": "0x8001"},
    "networking.example.com/vlan": {"string": "100"}
  }
}
```

Alpha matching behavior:

- If the device is clean, the first compatible claim sets the lock to:
  - `subnet = subnet-a`
  - `pkey = 0x8001`
- A later claim with the same `subnet` and same `pkey` may share the
  device.
- A claim with `subnet = subnet-a` but `pkey = 0x8002` is rejected for that
  device because all declared keys must match.
- A claim that provides only `subnet` but omits `pkey` is rejected for that
  device because missing declared keys are invalid.
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
  - Filter: no recognized `StructuredParameters` entry for a sharing-constrained
    request → device filtered out
  - Filter: malformed recognized `StructuredParameters` payload → device filtered out
  - Filter: duplicate recognized `StructuredParameters` entries for one request →
    device filtered out
  - Filter: non-string value for a required affinity key → device filtered out
  - Filter: device with `AffinityStates[deviceID].Unknown` set is excluded for new
    `sharingAffinity` scheduling
  - Device preference: allocator prefers locked-compatible device over clean device
  - Reserve: first claim sets lock; second claim with same values succeeds
  - Reserve: second claim with conflicting values fails
  - Unreserve: tentative lock is rolled back
  - Legacy claims with non-reconstructable affinity cause the device to be marked
    unknown rather than establishing or joining a lock
  - Legacy-claim handling: all scenarios from the `Handling Legacy Claims with
    Unknown Affinity` table
  - Compatibility matrix: device without `sharingAffinity` is unaffected —
    claims with or without `StructuredParameters` both pass (Legacy/Basic and
    Permissive Sharing rows)
  - Strict Gating: device has `sharingAffinity` but claim provides zero
    `StructuredParameters` config entries → device filtered out
- `staging/src/k8s.io/api/resource/v1`: Coverage for new API types and the
  recognized structured-parameters contract, including:
  - Validation: `parameterKeys` exceeding max 8 limit is rejected
  - Validation: structured parameters exceeding max 8 attributes is rejected
  - Validation: duplicate recognized structured-parameters entries for the same
    request are rejected when validation can detect them
  - Validation: non-string values for keys referenced by `sharingAffinity`
    are rejected when validation can detect them
  - Round-trip serialization of `SharingAffinity` and well-known schema

##### Integration tests

- Affinity matching with multiple claims to same device
- Affinity mismatch causing allocation to different device
- Affinity lock clearing when all claims release a device
- Interaction with consumable capacity constraints (KEP-5075)
- Scheduler restart: `AffinityStates` correctly reconstructed from existing
  bound ResourceClaims, and devices with non-reconstructable active claims have
  `AffinityStates[deviceID].Unknown` set
- Parallel scheduling: two Pods with conflicting affinity values targeting the
  same device — one wins Reserve, the other is requeued
- Feature gate disabled: `sharingAffinity` fields are ignored; devices are
  treated as unconditionally shareable
- Feature gate toggled: enabling after claims exist does not disrupt already-bound
  workloads, and legacy in-use devices are conservatively filtered until clean
- Invalid structured parameters: malformed payload or duplicate recognized
  entries for one request do not crash scheduling and deterministically exclude
  sharing-constrained devices
- Invalid value type: a required affinity key encoded as a non-string value is
  rejected for `sharingAffinity` scheduling and does not populate lock state
- Ghost Lock: Pod is Assumed (tentative lock set) but Bind fails — verify
  the lock is cleared immediately and the next Pod in the queue can claim the
  device with a different affinity value
- Legacy Device Migration: 5 Pods are already running on a NIC; the driver
  updates `ResourceSlice` to add `sharingAffinity`; a 6th Pod arrives with
  structured parameters — verify the device has `AffinityStates[deviceID].Unknown` set
  and the 6th Pod is filtered from that device until all legacy claims drain
- Partial Key: Device requires `subnet` and `pkey` in `parameterKeys`;
  claim provides only `subnet` — verify the device is filtered out
- Device Preference Packing: Two devices available, one already locked to subnet-X;
  new claim for subnet-X → verify the claim is placed on the locked device,
  not the clean one (full Filter→Reserve pipeline with allocator preference)
- Permissive Sharing (no SA): Device without `sharingAffinity`, claim with
  `StructuredParameters` — verify scheduler allows the allocation and SP are
  not evaluated for affinity
- Driver Backstop: Device without `sharingAffinity`, two claims with
  incompatible config land on the same device — verify scheduler allows both
  (permissive), and `NodePrepareResources` rejects the incompatible claim
- NodePrepareResources failure does not clear lock: Claim is bound and
  lock is persisted, but `NodePrepareResources` fails on the node — verify
  the affinity lock remains in the scheduler cache

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

- Feature implemented behind `DRASharingAffinity` feature gate
- API fields added to ResourceSlice (`SharingAffinity` on `Device`)
- Well-known `StructuredParameters` JSON schema defined for opaque config
- Scheduler decodes well-known schema from opaque config for affinity matching
- Scheduler Filter plugin enforces affinity matching
- Allocator prefers locked-compatible devices over clean devices when selecting within a node
- Scheduler tracks affinity in AllocatedState
- Unit and integration tests
- Documentation for driver authors
- Alpha documentation explicitly calls out the lack of lock-breaking preemption
  semantics for incompatible locks
- Alpha documentation explicitly calls out string-only affinity matching and
  the rejection of non-string values for `sharingAffinity` keys

#### Beta

- Gather feedback from DRA driver developers
- Address any issues found in alpha
- **Lock-aware preemption**: PostFilter detects affinity mismatch as a
  preemption-solvable problem; identifies lock-holder Pods as victims when a
  higher-priority Pod needs a device locked to an incompatible value
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
`sharingAffinity` and claim `StructuredParameters` presence.

**Recommended rollout sequence**: To minimize capacity stranding, drivers should
ideally:

1. Wait for a device to be idle (clean).
2. Update the ResourceSlice to include the `sharingAffinity` field.
3. Allow the scheduler to establish the first known lock with a new claim.

During mixed rollouts (some devices with `sharingAffinity`, some without), the
device-selection preference for `sharingAffinity` devices (see
[Device Selection Preference](#filter-phase-and-device-selection)) naturally steers affinity-aware claims toward
upgraded devices.

**Adding `sharingAffinity` to an in-use device**: A driver may add or update
`sharingAffinity` on a device that already has active (bound) ResourceClaims.
This can happen during driver upgrades or when enabling the feature on existing
hardware. The scheduler handles this as follows:

- **Pre-existing claims continue to run** and are not evicted.
- If any active claim on that device does not provide reconstructable
  affinity values for the required keys, the scheduler marks the device as
  `AffinityStates[deviceID].Unknown = true`.
- A device with `AffinityStates[deviceID].Unknown` set is not eligible for new
  `sharingAffinity` placements, even if it still has nominal shared capacity.
- **Once all active claims on that device are released**, the device becomes
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

**Handling missing or malformed parameters**: The scheduler treats a device
with `sharingAffinity` as a protected resource. If a device requires both
`subnet` and `pkey` but a claim only provides `subnet`, the device is filtered
out — all declared keys are mandatory. If a claim's `StructuredParameters`
entry is malformed or contains non-string values for required keys in alpha,
the device is excluded.

**Downgrade**: If the feature gate is disabled:

- The `sharingAffinity` field is not persisted on new writes.
- The scheduler ignores the field — all devices return to unconditional sharing
  (pre-KEP-5981 behavior).
- The DRA driver becomes the sole authority for enforcing hardware compatibility
  at `NodePrepareResources`.

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
  - Feature gate name: `DRASharingAffinity`
  - Components depending on the feature gate: kube-apiserver, kube-scheduler

###### Does enabling the feature change any default behavior?

No. Devices without `sharingAffinity` behave exactly as before. The feature only affects devices that explicitly opt-in via the new field.

###### Can the feature be disabled once it has been enabled (i.e. can we roll back the enablement)?

Yes. Disabling the feature gate causes:
- API server to strip the `sharingAffinity` field from new or updated
  ResourceSlices before persisting (writes succeed, field is not stored)
- Scheduler to ignore existing `sharingAffinity` fields for future placement
  decisions

Existing allocations continue to work. New allocations may become more
permissive, so the driver must continue validating compatibility at prepare
time.

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
`sharingAffinity` and by observing `ResourceClaim`s that include recognized
`StructuredParameters` for requests targeting those devices.

###### How can someone using this feature know that it is working for their instance?

A user should be able to observe that:

- compatible claims preferentially pack onto already-locked devices,
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
- rate of malformed or duplicate recognized `StructuredParameters` payloads,
- share of successful placements that pack onto already-locked compatible
  devices,
- prepare-time rejections by the DRA driver caused by incompatible hardware
  configuration.

###### Are there any missing metrics that would be useful to have to improve observability of this feature?

This feature would benefit from scheduler-observable counters and/or events for:

- `sharing_affinity_filter_mismatch_total`
- `sharing_affinity_filter_missing_parameters_total`
- `sharing_affinity_filter_invalid_parameters_total`
- `sharing_affinity_unknown_device_total`
- `sharing_affinity_packed_allocation_total`

Exact metric names are illustrative and implementation-specific, but
equivalent observability is strongly recommended.

In addition, user-facing diagnostics should make the reason for filtering clear,
for example:

- missing required structured parameters for request `<name>`,
- duplicate recognized `StructuredParameters` entries for request `<name>`,
- required key `<key>` has a non-string value in alpha,
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
- ResourceClaim: Small increase when claims include a `StructuredParameters`
  opaque config entry with affinity values (up to 8 string-valued attributes,
  matching the device-side cap)

###### Will enabling / using this feature result in increasing time taken by any operations covered by existing SLIs/SLOs?

Negligible. The Filter phase decodes the `StructuredParameters` opaque config
payload once per candidate device (bounded by payload size and 8-key cap).
The affinity comparison itself is O(k) where k ≤ 8 — a map lookup per key.

###### Will enabling / using this feature result in non-negligible increase of resource usage (CPU, RAM, disk, IO, ...) in any components?

No. The per-component impact is bounded:

- **Scheduler RAM**: `AffinityStates` adds one `map[string]string` (up to 8
  entries) per device with active affinity locks — proportional to active shared
  allocations, not total devices.
- **Scheduler CPU**: JSON decoding of the `StructuredParameters` opaque config
  entry during Filter adds a small per-candidate cost, bounded by the 8-key cap.
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

- **Malformed structured parameters**: the scheduler cannot decode the
  recognized payload and filters the device.
- **Duplicate recognized entries for one request**: the scheduler treats the
  request as invalid for `sharingAffinity` scheduling.
- **Missing required keys**: the claim cannot be matched against a device that
  declares those keys.
- **Non-string values for required keys in alpha**: the device is filtered for
  that claim.
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
2. Inspect the `ResourceClaim` and confirm there is exactly one recognized
   `StructuredParameters` entry for the relevant request.
3. Verify that every required affinity key is present and string-valued.
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

## Drawbacks

- Adds a new cache dimension (`AffinityStates`) to the scheduler's allocation
  tracking, increasing the surface area for reconstruction bugs on restart
- Once a device is locked, its effective affinity cannot change until all
  claims on that device are released
- Fragmentation risk remains if affinity values are too fine-grained
- Conservative handling of legacy in-use devices can temporarily strand
  schedulable capacity during rollout or migration
- If a driver declares `sharingAffinity` on a device but no claims ever provide
  `StructuredParameters`, that device becomes effectively unschedulable for
  sharing workloads — all claims are filtered out by the "Strict Gating" rule.
  Drivers should coordinate with workload teams to ensure claims include
  `StructuredParameters` before enabling `sharingAffinity` on devices.
- The scheduler now depends on decoding a well-known JSON schema from opaque
  config — a new coupling that didn't exist before. If the schema evolves,
  backward compatibility must be maintained across scheduler versions.
- Affinity locks are purely in-memory with no API or status field to inspect
  which devices are locked to which values. Debugging lock state requires
  scheduler logs.

## Alternatives

### Claim-side SharingAffinity (on DeviceRequest)

Instead of using StructuredParameters in opaque config to supply constraint
values, an alternative design adds a dedicated `SharingAffinity` field on
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
- **Requires an API change to ResourceClaim**: Adding a typed `SharingAffinity`
  field to `DeviceRequest` introduces a new API field, whereas supplying the
  same constraint values via StructuredParameters in existing opaque config
  requires no API change at all. The opaque config path is the preferred
  approach per @pohly's guidance—use well-known schemas inside existing opaque
  config rather than adding new structured fields to ResourceClaim.

### Object Reference-based Affinity Matching

An alternative approach replaces inline affinity values with external object
references. Instead of embedding values in opaque config, the claim would
reference a CRD (e.g., `NetworkConfiguration`) by name, and the device would
declare which object kinds constrain sharing:

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
  (`commonConfigKind`), whereas the chosen approach adds a field only to
  Device and uses existing opaque config for claim-side values
- Requires external CRD definitions, adding operational burden for cluster
  administrators
- Multi-dimensional affinity: A device may need affinity on multiple independent axes
  (e.g., subnet + VLAN). With object references, each axis would need its own CRD.
- Conflicts with the direction from @pohly to avoid new API fields on claims
  and use well-known schemas inside existing opaque config

### Placeholder Pattern Workaround

Without this KEP, drivers must use a "placeholder pattern" today:

1. Publish devices with `capacity: 1` initially
2. Wait for first claim to determine affinity value
3. Update ResourceSlice with actual capacity and affinity as attribute
4. Use CEL selector to match affinity attribute


**Problems**:
- Race condition: Second pod may go to different device before expansion
- ResourceSlice churn: Constant updates as pods come and go
- Driver complexity: State machine for expand/contract lifecycle

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
and the scheduler then packs subsequent compatible claims onto that device.

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
