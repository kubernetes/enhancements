# KEP-5981: DRA Sharing Affinity

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
    - [Composition with DRA Preemption (KEP-5690)](#composition-with-dra-preemption-kep-5690)
    - [Handling Legacy Claims with Unreconstructable Affinity](#handling-legacy-claims-with-unreconstructable-affinity)
    - [Compatibility Matrix](#compatibility-matrix)
  - [Risks and Mitigations](#risks-and-mitigations)
- [Design Details](#design-details)
  - [API Enhancement](#api-enhancement)
    - [ResourceSlice Spec](#resourceslice-spec)
    - [Scheduler Enhancement](#scheduler-enhancement)
  - [Feature Gates](#feature-gates)
  - [Examples](#examples)
    - [ResourceSlice with Sharing Affinity](#resourceslice-with-sharing-affinity)
    - [ResourceClaim (status quo opaque config)](#resourceclaim-status-quo-opaque-config)
    - [Multi-key Sharing Affinity Example](#multi-key-sharing-affinity-example)
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
    - [Recommended Rollout](#recommended-rollout)
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
  - [Well-known JSON schema inside OpaqueDeviceConfiguration](#well-known-json-schema-inside-opaquedeviceconfiguration)
  - [Typed `Structured` sibling on `DeviceConfiguration`](#typed-structured-sibling-on-deviceconfiguration)
  - [Claim-side-only SharingAffinity (on DeviceRequest)](#claim-side-only-sharingaffinity-on-devicerequest)
  - [Object Reference-based Affinity Matching](#object-reference-based-affinity-matching)
  - [CEL on runtime scheduler lock state (rejected variant)](#cel-on-runtime-scheduler-lock-state-rejected-variant)
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

This KEP introduces a `sharingAffinity` field on `ResourceSlice.spec`
that allows drivers to declare, via CEL expressions, how to extract
the affinity keys that constrain sharing from the driver's existing
opaque device configuration. `sharingAffinity` is published as
**pool-level metadata** in a dedicated slice within the pool — a
slice that carries `sharingAffinity` does not carry `devices`, and
vice versa — mirroring the established `sharedCounters` (KEP-4815)
pattern. This gives one source of truth per pool for how to
interpret the driver's opaque-config schema, even when the pool is
chunked across many ResourceSlices.

On the claim side, there is no API change — workloads continue to
author their driver's opaque config exactly as they do today. When
the scheduler is evaluating a candidate device, it looks up the
pool's metadata slice, runs each published CEL extractor whose
selector matches the candidate device (or that has no selector)
against the opaque-config objects in scope for the request
currently being filtered (per `DeviceClaimConfiguration.Requests`),
and — if every applicable group is fully satisfied —
records the resulting key/value pairs in `AllocatedState`
alongside consumed capacity. This enables the scheduler to gate remaining capacity
on locked devices and safely reuse them for compatible claims when
selected by the existing allocator. Alpha provides correctness only —
affinity-aware preference (packing) is delivered in beta; see
[Goals](#goals).

In addition, if a device already has active allocations whose
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
3. Update ResourceSlice with actual capacity, writing the affinity value into the device's `attributes` map
4. Use CEL selector to match against that `attributes` entry

**Problems**:
- Race condition: Second pod may go to different device before expansion
- ResourceSlice churn: Constant updates as pods come and go
- Driver complexity: State machine for expand/contract lifecycle

### Goals

- Enable the scheduler to gate remaining capacity on a device based on a
  required affinity key
- Provide a mechanism for drivers to signal compatibility requirements for
  shared hardware via `sharingAffinity` on the ResourceSlice — without
  requiring drivers to change their existing opaque config schemas, and
  without requiring workload authors to learn a new claim-side API
- Reduce fragmentation of cluster resources by enabling the scheduler to
  pack workloads with compatible sharing requirements onto already-locked
  devices (delivered in beta as a sharing-affinity term added to
  `DynamicResources.computeScore` — see [Affinity-aware scoring (planned
  for Beta)](#affinity-aware-scoring-planned-for-beta); alpha provides
  correctness only)
- Track affinity values in `AllocatedState` so subsequent scheduling decisions
  respect the first claim's lock-in
- Maintain backward compatibility with devices that have no sharing affinity
  constraints, and with existing opaque-config workloads

### Non-Goals

- Defining hardware-specific affinity key names (these remain driver-defined)
- Managing the physical lifecycle of the device configuration (this remains
  the driver's responsibility)
- Changing how capacity is tracked (that's KEP-5075)
- Supporting affinity across multiple devices. The lock is scoped to a
  single `Device` object in `ResourceSlice.spec.devices[]` — affinity is
  never shared across separate `Device` objects, even ones of the same
  type within the same pool, and certainly not across device types or
  pools.
- Retrofitting affinity-aware sharing onto already-in-use devices when active
  claims do not expose reconstructable affinity values. In alpha, such devices
  are treated conservatively until they drain clean.
- Guaranteeing **lock-breaking preemption**. This KEP does not
  introduce any preemption logic. Baseline DRA preemption is being
  introduced separately by
  [KEP-5690](https://github.com/kubernetes/enhancements/issues/5690);
  this KEP is designed to compose transparently with it (see
  [Composition with DRA Preemption (KEP-5690)](#composition-with-dra-preemption-kep-5690))
  but does not depend on or deliver it.

## Proposal

Add a `sharingAffinity` field to `ResourceSlice.spec` that publishes
a list of driver-defined CEL extractors. Each list entry is an
**applicability group**: an all-or-nothing bundle of CEL expressions
that together describe one schema variant the driver supports. The
scheduler evaluates an extractor's CEL expressions against every
opaque-config object the claim carries; each expression returns
either a string value (the affinity key for that object) or the
empty string `""` (meaning "this extractor does not apply to this
object"). A claim satisfies an applicability group when every key
in that group's `cel` map returns a non-empty value across the
claim's opaque configs. The non-empty returns from a
fully-satisfied group form the claim's effective affinity key map.

`sharingAffinity` is published in a dedicated **metadata slice**
that is mutually exclusive with `devices`. Device-bearing slices in the
same pool reference back to the metadata slice via the standard
pool tuple `(driver, pool.name, pool.generation, nodeName)`. This
keeps the driver-schema declaration in one place per pool even when
the pool is chunked across many slices.

```yaml
# Metadata slice: carries sharingAffinity, no devices.
apiVersion: resource.k8s.io/v1
kind: ResourceSlice
metadata:
  name: networking-node-a-meta
spec:
  driver: networking.example.com
  nodeName: node-a
  pool:
    name: node-a
    generation: 7
    resourceSliceCount: 2
  sharingAffinity:
    - cel:
        subnet: |
          object.apiVersion == "networking.example.com/v1"
            && object.kind == "NICConfig"
            ? object.subnetID : ""
        pkey: |
          object.apiVersion == "networking.example.com/v1"
            && object.kind == "NICConfig"
            ? object.ibPKey : ""
---
# Device slice: carries devices, no sharingAffinity. Joined to the
# metadata slice by the (driver, pool.name, generation, nodeName)
# tuple.
apiVersion: resource.k8s.io/v1
kind: ResourceSlice
metadata:
  name: networking-node-a-0
spec:
  driver: networking.example.com
  nodeName: node-a
  pool:
    name: node-a
    generation: 7
    resourceSliceCount: 2
  devices:
    - name: eth1
      allowMultipleAllocations: true
      capacity:
        networking.example.com/slots:
          value: "16"
```

Each CEL expression is responsible for guarding its own applicability
on `object.apiVersion` and `object.kind` and returning the empty
string when the opaque-config object does not match. This contract
gives drivers full control over which opaque-config schemas a given
extractor applies to, and avoids forcing the scheduler to know
anything about driver schema taxonomies. CEL runtime errors (for
example, dereferencing a field that does not exist on the parsed
object) are *not* equivalent to returning `""` — they are extraction
failures and cause the device to be filtered out and an Event to be
emitted. The `apiVersion`/`kind` guard prevents this for objects
that belong to other schemas.

If a driver supports multiple opaque-config schemas where the
extracted fields live at the same path on each, one extractor can
handle them all:

```yaml
sharingAffinity:
  - cel:
      # Same field path on both v1beta1 and v1 → one extractor.
      subnet: |
        (object.apiVersion == "networking.example.com/v1" ||
         object.apiVersion == "networking.example.com/v1beta1")
          && object.kind == "NICConfig"
          ? object.subnetID : ""
```

A heterogeneous pool can publish per-kind extractors so each
device kind binds to its own keys, and can additionally publish a
no-selector "default" extractor for keys every device in the pool
shares. Multiple applicable extractors for a candidate device are
all evaluated and Strict Gating requires every one to be fully
satisfied (see [Targeting extractors at specific device
kinds](#targeting-extractors-at-specific-device-kinds)):

```yaml
sharingAffinity:
  # Applies to every device in the pool (no selector). Common
  # affinity dimension shared by NICs and VFs alike.
  - cel:
      vendor: |
        object.apiVersion == "networking.example.com/v1"
          && has(object.vendor)
          ? object.vendor : ""
  # Applies only to NIC devices. Subnet and partition key (pkey)
  # are the NIC-specific affinity dimensions.
  - selector:
      cel:
        expression: 'device.attributes["type"].string == "nic"'
    cel:
      subnet: |
        object.apiVersion == "networking.example.com/v1"
          && object.kind == "NICConfig"
          ? object.subnetID : ""
      pkey: |
        object.apiVersion == "networking.example.com/v1"
          && object.kind == "NICConfig"
          ? object.ibPKey : ""
  # Applies only to VF devices. Parent-NIC UUID is the VF-specific
  # affinity dimension.
  - selector:
      cel:
        expression: 'device.attributes["type"].string == "vf"'
    cel:
      parentNIC: |
        object.apiVersion == "networking.example.com/v1"
          && object.kind == "VFConfig"
          ? object.parentNICUUID : ""
```

For a candidate NIC device, the *common* and *NIC* extractors are
both applicable, so the claim's opaque configs must produce
non-empty `vendor`, `subnet`, *and* `pkey`. The merged effective
affinity map is `{vendor, subnet, pkey}`. For a VF device, the
applicable set is *common + VF* and the merged map is
`{vendor, parentNIC}`.

A claim continues to author the driver's opaque config exactly as it
does today — no new claim-side API field is introduced:

```yaml
config:
  - requests: ["nic"]
    opaque:
      driver: networking.example.com
      parameters:
        apiVersion: networking.example.com/v1
        kind: NICConfig
        vendor: acme
        subnetID: subnet-A
        ibPKey: "0x8001"
        qos: gold       # driver-private; not extracted, not seen by scheduler
        mtu: 9000       # driver-private
        vlanTag: 100    # driver-private
```

When the scheduler evaluates a multi-allocatable device backed by a
pool that declares `sharingAffinity`:

1. **First claim**: When evaluating a candidate device in this pool,
   the scheduler looks up the pool's metadata slice and, for each
   `sharingAffinity` entry whose `selector` matches the candidate
   device (or has no selector), runs the entry's CEL expressions
   against every opaque config object that is in scope for the
   request currently being filtered (per
   `DeviceClaimConfiguration.Requests`). For each such applicable
   group, if every CEL key in that group's `cel` map
   produces a non-empty return across the claim's opaque configs,
   the group is *satisfied*. The claim is eligible for the device
   only if **every** applicable group is satisfied; the union of
   their key/value contributions is the claim's effective affinity
   map and is recorded in `AllocatedState` alongside consumed
   capacity.
2. **Subsequent claims**: The scheduler runs the same extraction for
   each new claim and compares the merged key set to
   `AllocatedState`'s `LockedAffinity`.
3. **Mismatch**: If any extracted key has a different value than the
   recorded lock, the device is filtered out for that request and the
   scheduler tries another candidate device.
4. **Match**: If all extracted keys match and capacity is available,
   allocation proceeds.

Keys absent from the slice's CEL expression set are not extracted, so
driver-private config (e.g., `qos`, `mtu`, `vlanTag` above) flows
through to `NodePrepareResources` untouched and never participates in
lock evaluation. The driver remains the sole authority for those
fields.

**Alpha Design Decisions**

**1. Placement of extraction: ResourceSlice (driver-side, pool-level metadata slice)**

`sharingAffinity` lives on `ResourceSlice.spec` and is **mutually
exclusive with `devices`**. A pool publishes one metadata slice that carries
`sharingAffinity` and zero devices; the pool's device-bearing slices
carry devices and no `sharingAffinity`.

The driver is the natural owner: the extraction logic describes the
driver's own opaque config schema, which is uniform across all
devices the driver publishes in this pool and evolves with driver
versions, not with hardware instances. Pool-level placement avoids
per-slice duplication, removes the drift risk of having to keep the
extractor block in sync across many slices when a pool is chunked,
and aligns the declaration site with what is being declared (a
driver-schema property, not a device property). Per-device overrides
can be added later if a driver ever needs them.

**2. How affinity values reach the scheduler: CEL on opaque config**

The scheduler does not interpret driver-private fields — it only
evaluates the CEL expressions the driver has explicitly published as
extraction logic. Each extractor is responsible for recognizing the
schema(s) it applies to (typically by inspecting `object.apiVersion`
/ `object.kind` and returning `""` when it does not).

```go
// SharingAffinityExtractor declares CEL expressions that produce
// sharing-affinity key/value pairs from a driver's opaque device
// config. Each entry in a pool's `sharingAffinity` list is an
// applicability group. See Design Details → API Enhancement for
// the canonical godoc.
//
// +featureGate=DRASharingAffinity
type SharingAffinityExtractor struct {
    // Optional; restricts this extractor to devices whose attributes
    // satisfy the CEL expression. Omitted = applies to every device
    // in the pool.
    Selector *DeviceSelector

    // Map from affinity key name to a CEL expression over the
    // claim's opaque-config object. Empty-string return = no
    // contribution. Max 8 entries.
    CEL map[string]string
}
```

Properties of this design:

- **No claim-side API change**: workloads keep authoring the
  driver's existing opaque config. No migration cost on the user
  side.
- **Minimal scheduler assumption**: the scheduler's only assumption
  is "run some CEL against the claim's opaque configs and see what
  key/value pairs come back." It never owns or interprets the
  driver's schema.
- **Driver-private keys are invisible to the scheduler**: anything
  the driver does not publish a CEL expression for is never
  extracted. The scheduler cannot inadvertently lock on
  hardware-configuration fields (qos, mtu, vlan) that drivers want
  to keep within their domain.

**Extractor applicability contract**

An extractor's CEL is evaluated against every opaque-config object in
the claim. For each (extractor, opaque-config, key) triple:

- A **non-empty string** return contributes that `key → value` to the
  claim's effective affinity map.
- An **empty string** return contributes nothing for that key from
  that object. This is the idiom drivers use to express "this
  extractor does not handle this opaque-config schema." Empty returns
  are not errors.

After evaluating each applicable extractor (its selector matches
the candidate device, or no selector is set) against every
opaque-config object, the scheduler checks whether the group of
each such extractor is *satisfied* — every key in its `cel` map
produces a non-empty return across the claim's opaque configs.
The device is eligible only if **every** applicable group is fully
satisfied (Strict Gating — per-group, all-or-nothing, applied
across all applicable groups).

**Extraction failure modes**

The scheduler's contract for a sharing-affinity device on a candidate
claim is:

1. **Some applicable group not satisfied — Strict Gating**: at
    least one applicable entry in the pool's `sharingAffinity` list
    (its `selector` matches the candidate device, or it has no
    selector) does not have all of its declared keys produced as
    non-empty by the claim's opaque configs (the extreme case is
    "no keys produced at all"; a partial case is a NIC schema group
    that produced `subnet` but not `pkey`). The claim is filtered
    out for any device in such a pool. This is the safe default —
    the driver declared that sharing requires scheduler-readable
    extraction across every applicable schema variant, and a claim
    that fully matches none of them cannot participate. The claim
    remains eligible for devices in pools that do not declare
    `sharingAffinity`.
2. **Inconsistent extraction within one claim**: two (extractor,
    opaque-config) pairs produce the same key with different non-empty
    values. The claim is rejected for that device (Event emitted;
    treated the same as a self-inconsistent claim).
3. **Key collision across applicable extractors** (ResourceSlice
    author error): two or more applicable extractors declare the
    same key name in their `cel` maps for the candidate device.
    The lock state would be ambiguous about which group's value to
    store. The device is filtered out for that request, an Event
    is emitted on the pod, and the slice's author is expected to
    deduplicate keys or tighten selectors so that each candidate
    device sees at most one extractor per key name. See
    [Targeting extractors at specific device kinds](#targeting-extractors-at-specific-device-kinds)
    for admission-time and runtime enforcement.
4. **CEL evaluation error or non-string return**: the device is
   filtered out for that claim, an Event is emitted on the pod, and
   the scheduler does **not** silently lock the device to an empty
   key set.
5. **Cost-budget exhaustion**: same as evaluation error.
6. **Missing field on the parsed object**: a CEL expression
   dereferences a field absent from `object` (e.g., `object.subnet`
   on an object with no `subnet`), raising a CEL runtime error.
   Sub-case of case 4 (filter + Event). This may indicate:
   - the claim carries an opaque-config object the extractor was
     not meant to inspect (unrelated driver-tuning config, or a
     config for a different driver kind in the same claim);
   - an authoring mistake (typo, wrong field name);
   - schema evolution where the driver renamed or removed a field
     across versions.

   The canonical extractor idiom — guard with `apiVersion` / `kind`
   (or `has(...)`) and return `""` from the non-applicable branch —
   distinguishes "this object intentionally does not contribute"
   (`""`) from "this object should have contributed but the schema
   does not match" (runtime error). Only the latter filters the
   device.

**Alpha scope**

Alpha fully resolves the design around driver-side slice-level
extraction described above. Claims do not control lock-setting
behavior: any compatible claim may establish the initial lock on a
clean device. Claim-side lock-setting policy (for example,
`CanSetLock`/`NeverSetLock`) is deferred to [Future
Enhancements](#future-enhancements).

Alpha standardizes driver-declared CEL extraction, a single feature
gate that governs both the slice-level field and the scheduler logic,
and correct lock enforcement on already-locked devices — but
intentionally stops short of affinity-aware scoring (planned as a
beta-scope contribution to `DynamicResources.computeScore`, see
[Affinity-aware scoring (planned for
Beta)](#affinity-aware-scoring-planned-for-beta)).

**Alpha limitations**

Alpha enforces lock compatibility but does not preempt incompatible
lock-holders. A higher-priority Pod requiring a different affinity
value than the current lock will remain unschedulable on that device
until the lock-holder exits or a compatible alternative appears. This
is consistent with baseline DRA, which has no preemption support in
the absence of
[KEP-5690](https://github.com/kubernetes/enhancements/issues/5690).
See [Composition with DRA Preemption (KEP-5690)](#composition-with-dra-preemption-kep-5690)
for how this KEP is structured to benefit transparently when KEP-5690
is present.

Alpha also restricts CEL return values to `string`. List-valued
return types (a claim accepting any of several values for one key)
are a recognized future extension but are out of scope for alpha —
all motivating use cases (subnet IDs, PKeys, bitstream identifiers,
NUMA tags) are single-valued.

A related extensibility note (raised by @pohly in PR review): a
scalar `string` return type forecloses carrying any metadata
alongside the extracted value. A future revision could return a
structured object (for example a CEL message/struct with fields
such as `value`, qualifiers, or per-key options) so that the
contract can grow without another API break. Alpha deliberately
stays on `string` because every motivating use case is a single
opaque identifier and a struct return adds CEL-side type plumbing
and scheduler-side parsing that is not justified by current
requirements; the option is preserved by treating the return type
as a versioned part of the extractor contract rather than a free
parameter. See [Future Enhancements](#future-enhancements) for the
extension path.

### User Stories

#### Story 1: RDMA Partition Key Alignment

A user runs a distributed training job where every Pod must share the same
RDMA Partition Key (PKey) to communicate. The NIC supports 16 VFs. The driver
publishes `sharingAffinity` on the slice with a CEL expression pulling the
PKey out of its existing opaque config (e.g., `pkey: "object.ibPKey"`). The
scheduler only co-allocates Pods whose claimed PKey matches the NIC's current
lock (or selects an unlocked NIC and establishes the lock from the first claim).

- Pod A (pkey-0x8001) is allocated to mlx5_0 → mlx5_0 is now locked to pkey-0x8001
- Pod B (pkey-0x8001) arrives → matches affinity, is eligible to share mlx5_0
- Pod C (pkey-0x8002) arrives → affinity mismatch on mlx5_0; mlx5_0 is
  filtered out; Pod C is allocated to mlx5_1 instead

#### Story 2: FPGA Bitstream Sharing

An inference service uses FPGAs to accelerate a specific model. Loading a
bitstream takes several seconds. The driver publishes a CEL expression on
the slice that extracts the bitstream identifier from its opaque config
(e.g., `bitstream: "object.bitstreamID"`). The scheduler only co-allocates
Pods that request a compatible bitstream onto an already-locked FPGA;
affinity-aware *preference* for FPGAs that already have the bitstream loaded
(over fresh ones) is delivered in beta.

- Pod A (bitstream-ml-v2) is allocated an FPGA → FPGA locks to bitstream-ml-v2
- Pod B (bitstream-ml-v2) arrives → eligible to share the same FPGA
- Pod C (bitstream-crypto-v1) arrives → filtered out from the locked FPGA;
  uses a different FPGA or waits

#### Story 3: Single-subnet NIC Sharing

A network DRA driver advertises NICs that can be shared across up to 16 pods,
but only if pods belong to the same subnet. The driver publishes a CEL
expression on the slice that extracts the subnet from its opaque config
(e.g., `subnet: "object.subnetID"`).

- Pod A (subnet-X) is allocated to eth1 → eth1 is now locked to subnet-X
- Pod B (subnet-X) arrives → matches affinity, is eligible to share eth1
- Pod C (subnet-Y) arrives → affinity mismatch on eth1; eth1 is filtered
  out; Pod C is allocated to eth2 instead

### Notes/Constraints/Caveats

- **Affinity is set by the first compatible claim on a clean device**: Once a
  device is allocated with an affinity value, that value is locked until all
  claims release the device.
- **Extractors and applicability groups**: The pool's
  `sharingAffinity` lists one or more extractors; each entry is an
  applicability group. Evaluation semantics (Strict Gating, per-group
  all-or-nothing, selector-based per-device dispatch) are detailed in
  [Filter Phase and Device Selection](#filter-phase-and-device-selection)
  and [Targeting extractors at specific device kinds](#targeting-extractors-at-specific-device-kinds).
- **Driver-private keys are invisible to the scheduler**: Keys not present
  in the slice's `cel` map (e.g., qos, mtu, vlan) are not extracted and do
  not participate in lock evaluation. They flow through opaquely to the
  driver at `NodePrepareResources` as today. This is the mechanism by which
  the scheduler stays out of driver-private config.
- **String-only matching in alpha**: CEL expressions must return `string`
  values. Non-string returns (numbers, booleans, lists, objects) are treated
  as extraction failures. Workloads use the natural string form of their
  identifiers (subnet IDs, PKey hex strings, bitstream names, FQDNs).
- **CEL evaluation errors**: A CEL expression that returns a non-string,
  errors, or exceeds the per-evaluation cost budget causes the device to
  be filtered out for that claim and an Event is emitted on the pod. The
  scheduler never silently establishes an empty lock when extraction fails.
- **Devices opting out of SharingAffinity extraction**: Devices in slices without `sharingAffinity` behave
  as before — any claim can share them regardless of opaque config content.
- **Legacy allocations with unknown affinity are conservative in alpha**:
  If a device has active allocations for which the scheduler cannot
  reconstruct the required affinity values (for example, claims created
  before the feature was enabled, or whose opaque configs produce no
  non-empty extractor returns under the currently-published CEL), that
  device is treated as having unknown affinity state and is filtered out
  for new sharing-affinity scheduling until it becomes fully clean.

#### Composition with DRA Preemption (KEP-5690)

This KEP does not deliver preemption. Lock-breaking preemption is
**not a KEP-5981 deliverable** in any milestone.

The design is structured so lock-breaking falls out transparently
when [KEP-5690 (DRA Preemption)](https://github.com/kubernetes/enhancements/issues/5690)
is present in the cluster: affinity locks live in the same
`dynamicresources.stateData` structure that KEP-5690 mutates via
`AddPod`/`RemovePod`. The only implementation requirement on the
KEP-5981 side is that `AddPod`/`RemovePod` correctly mutate
affinity-lock state alongside capacity — which is already required
for KEP-5690 compatibility.

**Known limitation: affinity-blind reprieve ordering.**
`SelectVictimsOnNode`'s reprieve order is `(priority, PDB, runtime)`
and does not consider which `Device` a Pod's claim is allocated to.
When a node has multiple candidate devices each holding a lock to a
different value, with asymmetric lock-holder counts, the algorithm
may converge on a victim set on the larger-lock-count device when
sacrificing the smaller one would have sufficed. Always correct,
sometimes over-evicts. A DRA-aware reprieve-ordering hook in
`DefaultPreemption` would resolve this cleanly but is its own
scheduler-framework enhancement, out of scope for this KEP.

If KEP-5690 is not present or is disabled, this KEP provides no
preemption capability — see
[Preemption Cannot Break Affinity Locks](#preemption-cannot-break-affinity-locks).

#### Handling Legacy Claims with Unreconstructable Affinity

| Device State | New Claim | Result |
|---|---|---|
| 5 legacy claims, affinity unknown | Claim whose CEL extraction yields `subnet: A` | **Filtered out**. Existing allocations have unknown affinity, so no new sharing-affinity lock may be established yet. |
| 5 legacy claims, affinity unknown | Claim whose extraction yields no non-empty keys | **Filtered out**. Missing required scheduler-readable affinity information. |
| Legacy claims drained; device now clean | Claim whose CEL extraction yields `subnet: A` | Lock set to `subnet: A`; device now locked. |
| Device locked to `subnet: A` | Claim whose CEL extraction yields `subnet: A` | Allowed (values match). |
| Device locked to `subnet: A` | Claim whose CEL extraction yields `subnet: B` | **Filtered out** (mismatch with lock). |
| All claims released | — | Device fully clean and eligible to establish a new lock. |

Legacy claims continue to run and are not evicted. However, until all unknown
allocations on a sharing-affinity device are released, the scheduler does not
assume it knows the device's effective modal state.

##### Changing `sharingAffinity` on a Slice with Active Allocations

In alpha, mutating a slice's `sharingAffinity` (adding, removing, or
changing extractor entries or CEL expressions) while bound claims
reference devices in that slice is not supported. On the next
reconciliation or scheduler restart, the scheduler re-runs the new CEL
against each bound claim's opaque config. If any bound claim no longer
yields the same key set as the current `LockedAffinity` (because a key
was added, removed, renamed, or the CEL now returns a different
value), the device is marked `AffinityStates[deviceID].Status = AffinityStatusUnreconstructable`
and filtered out for new sharing-affinity scheduling until all such
claims drain.

This is the same conservative-fallback behavior used for legacy claims and
is the deliberate alpha trade-off: the scheduler refuses to silently
downgrade its safety guarantee in the face of an asymmetric extraction
change. Drivers that need to evolve `sharingAffinity` for in-use slices
should drain affected devices before publishing the change, or stage
schema evolution by adding a new extractor entry (e.g., one whose CEL
guards on a new `apiVersion`) and migrating workloads to author against
the new schema before retiring the old extractor.

**Driver responsibility**: drivers should avoid hot-swapping
`sharingAffinity` entries on slices with active allocations. When
extraction changes are unavoidable (e.g., a hardware capability evolves),
drivers should expect the affected devices to be ineligible for new
affinity-aware scheduling until they drain clean, and should plan rollouts
accordingly (for example, by cordoning the device or rolling out the
extraction change as part of a node reimage).

#### Compatibility Matrix

To clarify the interaction between claims and devices, the following matrix
outlines how the scheduler and driver evaluate candidates based on whether
the device's pool declares a `sharingAffinity` Affinity Extractor list
(**AE**) and whether the claim
fully satisfies an applicability group in that pool's
`sharingAffinity` list (**GS** — group satisfied; under Strict Gating this
is per-group, all-or-nothing):

| Scenario | Pool AE | Claim GS | Scheduler Outcome | Driver Outcome |
|---|---|---|---|---|
| **Standard Feature Use** | Yes | Yes | **Match enforced.** Extracted values match lock + capacity available → scheduled. | **Validates** hardware mode matches claim config at `NodePrepareResources`. Rejects if stale or inconsistent. |
| **Strict Gating** | Yes | No | **Filtered out.** Device excluded — at least one applicable group for this candidate device was not fully satisfied by the claim's opaque configs (anything from "no keys at all" to "an applicable group had at least one key missing"). | **N/A** — claim never reaches the driver for this device. |
| **Legacy Device Transition** | Yes (newly added) | Yes | **Filtered out** while legacy claims are active (`Status: Unreconstructable`). Allowed once device drains clean. | **Validates** as normal once claim reaches the driver. During transition, driver continues serving legacy claims. |
| **Permissive Sharing** | No | Yes | **Allowed.** Pool has no `sharingAffinity`; opaque config is not evaluated for affinity. Standard capacity matching applies. | **Must enforce** hardware compatibility independently. Scheduler provides no affinity gating for this device. |
| **Legacy/Basic** | No | No | **Allowed.** Standard DRA capacity and attribute matching. | **Must enforce** hardware compatibility independently. This is the pre-KEP-5981 behavior. |

The top rows show the scheduler as the primary enforcer with the driver as
a backstop. The bottom rows show the driver as the sole enforcer with
the scheduler being permissive. The transition row shows the scheduler being
conservative (filtering) while the driver continues serving existing
workloads.

### Risks and Mitigations

#### Fragmentation (Poisoning)

**Risk**: A claim with a rare or unique affinity value can lock a
high-capacity device to that value, stranding the device's remaining
capacity against **peer claims** (any priority) that don't share the
value. Pure capacity-stranding, not a priority problem — even claims
of equal or lower priority cannot use the locked device. This section
covers the peer-claim case only; the priority-aware case is
[Preemption Cannot Break Affinity Locks](#preemption-cannot-break-affinity-locks).

**Mitigation (Alpha)**: None beyond Filter correctness. The DRA allocator
currently uses a first-fit algorithm with no affinity-aware preference, so
the scheduler does not actively pack compatible claims onto already-locked
devices. Where domain-specific validation is feasible, cluster
administrators can use `DeviceClass` CEL selectors to restrict which
affinity values are accepted (e.g., constraining subnet IDs to a known
set) — this is an admin-side guardrail against rare or arbitrary values
poisoning devices. Affinity-aware preference (within-node) and a Score
contribution (cross-node) are planned as a beta-scope addition to
`DynamicResources.computeScore`, following the same per-feature additive
pattern that Prioritized List (KEP-4816, shipped in 1.35) and Extended
Resources (KEP-5004) already use; see [Affinity-aware scoring (planned for
Beta)](#affinity-aware-scoring-planned-for-beta). Until that lands,
fragmentation mitigation is best-effort and depends on the existing
first-fit ordering of devices in ResourceSlices. General-purpose DRA
scoring discussion continues in
[kubernetes/enhancements#4970](https://github.com/kubernetes/enhancements/issues/4970),
but KEP-5981's contribution does not block on a unified framework.

#### Preemption Cannot Break Affinity Locks

**Risk**: Standard Kubernetes preemption triggers on resource shortage,
not on affinity-lock mismatch. A higher-priority Pod requiring a
different affinity value than the current lock cannot evict the
lock-holder under standard preemption — capacity is technically free,
so preemption is never triggered. This is the same gap that affects
all of DRA today, not specific to this KEP.

**Mitigation**: This KEP does not deliver preemption. The mitigation
within this KEP's scope is scoring/packing — see
[Fragmentation (Poisoning)](#fragmentation-poisoning) and the Beta
scoring contribution in
[Affinity-aware scoring (planned for Beta)](#affinity-aware-scoring-planned-for-beta).
Lock-breaking is a property that emerges when DRA preemption is
present in the cluster; see
[Composition with DRA Preemption (KEP-5690)](#composition-with-dra-preemption-kep-5690).

#### Packing Depends on DRA-Aware Scoring (Alpha Limitation)

**Risk**: The Filter phase guarantees correctness (incompatible locked
devices are excluded), but not packing — neither across nodes nor within
a node:

- **Cross-node**: standard Kubernetes scorers do not see DRA shared-device
  consumption, so they cannot prefer a node that already has a
  compatibly-locked device. Two compatible claims may land on two
  different nodes — each locking a separate device — even when
  consolidating onto one would have sufficed.
- **Within-node**: the DRA allocator currently uses first-fit. When a
  node has both a compatibly-locked device with capacity and a clean
  device, the allocator may pick whichever appears first in the
  ResourceSlice rather than preferring the locked one.

**Mitigation**: See [Fragmentation (Poisoning)](#fragmentation-poisoning)
for the beta scoring / packing plan; the `AllocatedState.AffinityStates`
structure introduced in alpha is the substrate the beta score function
reads, so alpha is the infrastructure step, not a dead end.

## Design Details

### API Enhancement

#### ResourceSlice Spec

```go
type ResourceSliceSpec struct {
    // ... existing fields (Driver, NodeName, Pool, Devices, etc.) ...

    // SharingAffinity declares per-pool affinity extractors. Mutually
    // exclusive with Devices in a single ResourceSlice. See
    // SharingAffinityExtractor for semantics.
    //
    // +optional
    // +listType=atomic
    // +k8s:maxItems=8
    // +featureGate=DRASharingAffinity
    SharingAffinity []SharingAffinityExtractor
}

// SharingAffinityExtractor declares CEL expressions that produce
// sharing-affinity key/value pairs from a driver's opaque device
// config. Each extractor entry in a pool's `sharingAffinity` list is
// an applicability group — an all-or-nothing bundle describing one
// schema variant the driver supports.
//
// +featureGate=DRASharingAffinity
type SharingAffinityExtractor struct {
    // Selector, if set, restricts this extractor to devices in the
    // pool whose attributes satisfy the CEL expression. The same
    // DeviceSelector type used by DeviceRequest.selectors is reused
    // here, so the `device` binding inside the CEL expression refers
    // to the candidate device's attributes. If omitted, the extractor
    // applies to every device in the pool (implicit dispatch).
    //
    // +optional
    Selector *DeviceSelector

    // CEL is a map from affinity key name to a CEL expression.
    // Inside the expression, the variable `object` refers to the
    // parsed opaque-config object. The expression must return a
    // string. An empty-string return means "no contribution for this
    // key from this object" — the idiom an extractor uses to ignore
    // opaque-config objects whose apiVersion / kind it does not
    // handle.
    //
    // +required
    // +k8s:maxProperties=8
    CEL map[string]string
}

const SharingAffinityMaxEntries = 8
const SharingAffinityCELMaxKeys = 8
```

#### Scheduler Enhancement

##### Source of Truth for Affinity Locks

The scheduler derives affinity locks **solely from CEL extraction over
active claims' opaque configs** — not from device attributes on the
ResourceSlice. The driver is NOT required to write locked affinity
values back to the ResourceSlice.

- The ResourceSlice declares *how* to extract sharing-affinity
  key/value pairs (`sharingAffinity`).
- The claims declare *what* values they need (via their normal opaque
  `OpaqueDeviceConfiguration.parameters` blob — driver's existing
  schema).
- The scheduler combines these by running CEL at Filter time and
  maintains the lock in `AllocatedState`.

This avoids two sources of truth that could diverge, eliminates
ResourceSlice churn (no update every time a lock is set/cleared), and
keeps driver implementation simple. Drivers MAY optionally publish
current locked values as regular device attributes for observability
(e.g., visible via `kubectl`), but the scheduler does not depend on
them.

When the last claim on a device is released, the scheduler clears the
lock. The scheduler's notion of "clean" is **allocation-clean, not
hardware-ready** — a device is considered clean once no allocated
claim references it, regardless of whether the driver has finished
in-flight hardware reconfiguration on the node. The driver remains
responsible for device lifecycle: tearing down the old configuration
(via `NodeUnprepareResources`) and reconfiguring for new claims (via
`NodePrepareResources`). Driver-level prepare/unprepare sequencing is
the authoritative guard against reuse before reconfiguration
completes; drivers that need stronger guarantees should hold their
own per-device readiness state and reject prepare calls until
reconfiguration is complete.

##### Safety Model and Responsibility Split

This feature intentionally keeps placement knowledge and hardware
enforcement separate:

- **Scheduler guarantee**: when it has successfully extracted affinity
  keys (via the slice's `sharingAffinity` CEL) for all active
  allocations on a device, it will not intentionally co-place claims
  with incompatible affinity values on that device.
- **Conservative fallback**: if the scheduler cannot reconstruct the
  effective affinity state of a device (for example, due to legacy
  claims, opaque configs that no longer produce any non-empty
  extraction under the current `sharingAffinity` entries, or CEL
  evaluation failures), it treats that device as unknown and filters
  it out for new sharing-affinity placements until the device becomes
  clean.
- **Driver guarantee**: the driver remains the final authority for
  programming and validating the actual hardware mode during
  `NodePrepareResources`.
- **Failure handling**: stale scheduler state or races may still cause
  prepare-time rejection, and that rejection remains the final safety
  backstop.

##### Cache Extension: Effective Device State

To prevent race conditions during high-volume scheduling, the scheduler
maintains affinity locks in its internal cache rather than relying on API
server round-trips. This is consistent with how DRA already handles
capacity tracking via `inFlightAllocations`.

The scheduler's `AllocatedState` is extended to track affinity values
alongside consumed capacity:

```go
type AffinityStatus string

const (
    // AffinityStatusClean: no active claims on the device.
    // LockedAffinity is nil.
    AffinityStatusClean AffinityStatus = "Clean"

    // AffinityStatusLocked: at least one claim is active and the
    // scheduler has reconstructed the device's affinity values.
    // LockedAffinity holds the current lock.
    AffinityStatusLocked AffinityStatus = "Locked"

    // AffinityStatusUnreconstructable: at least one active claim's
    // affinity values cannot be reconstructed (CEL extraction
    // failed, did not produce a non-empty value for some declared
    // key, or claim predates the feature). The device is filtered
    // for new sharing-affinity placements until it becomes Clean.
    // LockedAffinity is nil.
    AffinityStatusUnreconstructable AffinityStatus = "Unreconstructable"
)

type AffinityState struct {
    // Status is the device's current affinity state. See the
    // AffinityStatus constants for the meaning of each value and
    // when LockedAffinity is populated.
    Status AffinityStatus

    // LockedAffinity holds the device's affinity lock when
    // Status == AffinityStatusLocked. Nil for Clean and
    // Unreconstructable.
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

**Filter phase**: For a given node, the scheduler evaluates each device.
A device whose pool declares `sharingAffinity` (looked up via the
pool's metadata slice) is a candidate ONLY if:

1. It has sufficient consumable capacity (KEP-5075).
2. The device's `AffinityStates[deviceID].Status` is not
   `AffinityStatusUnreconstructable`.
3. For each extractor in the pool's `sharingAffinity` list whose
   selector (if set) matches the candidate device's attributes
   (extractors without a selector are considered applicable to every
   device in the pool), the scheduler evaluates the extractor's CEL
   expressions against every opaque-config object in scope for this
   request (per-request scoping via
   `DeviceClaimConfiguration.Requests`). Every CEL expression in the
   extractor's `cel` map must evaluate successfully and return a
   string value. CEL evaluation errors, non-string returns, missing
   fields, or cost-budget exhaustion cause the device to be filtered
   out for that claim and an Event to be emitted. Extractors whose
   selector does not match the candidate device are skipped — their
   `cel` map is not evaluated against this device.
4. Every applicable extractor's group is fully
   satisfied — every key in each such group's `cel` map produces a
   non-empty return across the claim's opaque configs (Strict
   Gating, per-group, all-or-nothing, applied across all applicable
   groups). The contributions from satisfied groups are merged into
   the claim's effective affinity map. Two contributions producing
   the same key with different non-empty values are treated as a
   self-inconsistent claim and the device is filtered out. Two
   applicable extractors declaring the same key name are treated
   as a ResourceSlice author error (key collision); the device is
   filtered out for that request and an Event is emitted so the
   driver can republish the slice with deduplicated keys or
   tightened selectors.
5. The device's `AffinityStates[deviceID].LockedAffinity` is either
   empty (unlocked) OR matches the claim's effective affinity map
   exactly for ALL extracted keys.

If a device has `AffinityStates[deviceID].Status == AffinityStatusUnreconstructable`, or if no
applicability group is fully satisfied by the claim, or extraction
fails (any reason in #3), the device is filtered out for
sharing-affinity scheduling. This is the safe default: the
driver declared that sharing requires scheduler-readable extraction,
and a scheduler that cannot reconstruct the current or requested
affinity state cannot evaluate placement safely. Claims that do not
need sharing-constrained devices should target devices in slices
without `sharingAffinity`.

**Device selection within a node**: Among feasible devices on a chosen node,
alpha does not introduce affinity-aware preference. Device selection
continues to use the existing structured-parameters allocator (first-fit).
This means that on a node with both a compatibly-locked device with
capacity and a clean device, the allocator may pick whichever appears
first in the ResourceSlice rather than preferring the locked one.

Affinity-aware preference (within a node) and node-level scoring (across
nodes) are planned for beta — see [Affinity-aware scoring (planned for
Beta)](#affinity-aware-scoring-planned-for-beta).

##### Targeting extractors at specific device kinds

When a pool publishes structurally different device kinds (for
example, a networking driver that publishes both NICs and
sub-functions/VFs), the extractor's optional `Selector` field
(`DeviceSelector`, the same type used by `DeviceRequest.selectors`)
restricts an extractor to candidate devices whose attributes
satisfy the CEL expression. Drivers reuse existing device
attributes (`type`, `family`, `vendor`, etc.) for dispatch — no
extra group-name pointer is required on the device side.

If an extractor's `selector` is omitted, the extractor applies to
every device in the pool. Homogeneous pools (one device kind) leave
`selector` empty and the behavior is identical to the
implicit-dispatch model. Heterogeneous pools set a selector per
extractor; for a given candidate device, only the extractors whose
selector matches participate in Filter step 3/4 evaluation,
bounding work to the relevant groups. Multiple applicable
extractors are allowed (for example, a no-selector "common"
extractor plus one or more device-kind-specific extractors); see
the key-collision rule below.

Key collisions across applicable extractors are treated as a
ResourceSlice author error. Within the set of extractors
applicable to any single candidate device, key names must be
unique — otherwise the lock state on that device would be
ambiguous about which extractor's value to store. Two enforcement
layers catch this:

- **Admission-time (best-effort)**: API server validation rejects
  the slice when two extractors share textually-identical (or
  normalized-equivalent) selectors and overlap on at least one key
  name, including the case of two entries that both omit
  `selector`. This catches the common authoring mistake of
  duplicating a block and forgetting to rename keys.
- **Runtime (authoritative)**: during Filter, the scheduler counts
  key contributions from applicable extractors for the candidate
  device. If the same key name is produced by more than one
  applicable extractor, the device is treated as non-viable for
  the request, an Event is emitted on the pod, and the claim stays
  Pending — the driver may republish the slice with deduplicated
  keys or tightened selectors to fix.

This permits a common factoring pattern: a no-selector "default"
extractor for keys every device in the pool shares, plus per-kind
extractors keyed off `device.attributes["type"]` for kind-specific
keys. It also permits two selectored extractors to reuse a key
name (e.g., both NIC and VF extractors declaring `subnet`) as long
as no candidate device matches both selectors. See the worked
example in [Proposal](#proposal) for the factored pattern in YAML.

##### Reserve Phase: Tentative Locking

Once a node/device is selected, the Reserve plugin establishes a "tentative
lock" in the scheduler cache before the Binding phase. Reserve reuses the
key map Filter already extracted — no second CEL pass — and atomically:

- If the device is still unlocked: record the claim's keys as the device's
  `LockedAffinity` and proceed.
- If the device became locked since Filter (another pod won the race) or
  was already locked: re-check the cached keys against the current
  `LockedAffinity`. On match, co-allocate. On mismatch, fail Reserve and
  let the scheduler retry with the next candidate.

The tentative lock is immediately visible to subsequent scheduling cycles:
if Pod-B is evaluated milliseconds after Pod-A's Reserve (before Pod-A's
bind reaches the API server), Pod-B's Filter phase sees Pod-A's tentative
lock and either joins it or skips the device.

If scheduling later fails for the pod (Unreserve), the tentative lock is
removed unless another already-bound claim is still co-located on the
device.

##### Scheduler Restart: State Reconstruction

On scheduler restart, the in-memory `AffinityStates` map is empty and must
be rebuilt from already-cached state (bound `ResourceClaim`s and their
`ResourceSlice`s) before the first scheduling cycle. The behavior
contract is:

- **Recovery**: for each bound claim on a device whose pool declares
  `sharingAffinity` (looked up via the pool's metadata slice), the
  scheduler re-derives the claim's key map and records it as the
  device's `LockedAffinity`. No new API calls are required.
- **Conservative fallback on ambiguity**: if extraction fails, yields no
  keys, or yields inconsistent values across claims sharing the device
  (which by construction should not happen, but may arise from historical
  bugs, manual etcd edits, or version skew), the device is marked
  `AffinityStatusUnreconstructable` and excluded from new sharing-affinity
  placements until all claims on it drain. The scheduler never infers a
  lock from ambiguous data.
- **No new persistence**: reconstruction uses the same informer-cached
  data the scheduler already consumes; no migration, no new API surface.


### Feature Gates

This KEP introduces one feature gate:

- **`DRASharingAffinity`** (alpha): adds the `sharingAffinity` field
  on `ResourceSlice.spec`, the `AllocatedState.AffinityStates` cache
  dimension, and the scheduler Filter / Reserve logic that runs CEL
  extraction over claim opaque configs and matches the result against
  the lock. The gate must be enabled on both `kube-apiserver` and
  `kube-scheduler` for the feature to function. Asymmetric enablement
  is non-destructive but produces predictable degenerate behavior:

  - **apiserver on, scheduler off**: drivers can write `sharingAffinity`
    and the apiserver persists it, but the scheduler ignores the field.
    Affinity declarations sit in storage without influencing placement;
    the cluster behaves as if no affinity is configured.

  - **apiserver off, scheduler on**: per standard alpha-field handling,
    the apiserver strips `sharingAffinity` from new writes. Slices
    persisted during a prior enabled period are still served on read,
    so the scheduler continues to honor existing locks. New driver
    attempts to declare affinity silently have no effect —
    operationally safe but a confusing failure mode for operators.

  The expected rollout order mirrors other DRA gates: enable on the
  apiserver first, then the scheduler; disable in reverse.

Because the claim side carries no new API surface (workloads keep
their existing opaque configs), no separate API-only gate is needed.


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
  sharingAffinity:
    - cel:
        subnet: |
          object.apiVersion == "networking.example.com/v1"
            && object.kind == "NICConfig"
            ? object.subnetID : ""
  devices:
    - name: eth1
      allowMultipleAllocations: true
      attributes:
        networking.example.com/type:
          string: "sriov-vf"
      capacity:
        networking.example.com/slots:
          value: "16"
    - name: eth2
      allowMultipleAllocations: true
      attributes:
        networking.example.com/type:
          string: "sriov-vf"
      capacity:
        networking.example.com/slots:
          value: "16"
```

The `sharingAffinity` block declares a single key (`subnet`) to be
pulled from any opaque-config object that identifies itself as
`networking.example.com/v1 / NICConfig`. The CEL guard on
`object.apiVersion` and `object.kind` ensures the expression returns
`""` (no contribution) when applied to an opaque-config object from
a different schema, rather than failing with a missing-field runtime
error. See the Proposal section for the canonical guard idiom and
the multi-version variant.

#### ResourceClaim (status quo opaque config)

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
      - requests: ["nic"]
        opaque:
          driver: networking.example.com
          parameters:
            apiVersion: networking.example.com/v1
            kind: NICConfig
            subnetID: subnet-X        # extracted as `subnet` by the slice's CEL
            vlanId: 100               # driver-private; not extracted
            mtu: 9000                 # driver-private; not extracted
```

> **Note**: The claim shape is unchanged from pre-KEP-5981 DRA — the
> workload author writes the driver's existing opaque config. The
> scheduler runs the slice-declared CEL expression against the parsed
> `parameters` — the `apiVersion`/`kind` guard matches, so the
> expression returns `subnet-X` and the scheduler records
> `subnet: subnet-X` for affinity lock evaluation. The driver-private
> fields (`vlanId`, `mtu`) flow through to `NodePrepareResources`
> untouched.

#### Multi-key Sharing Affinity Example

This example illustrates the alpha semantics when a slice extracts
multiple keys.

A driver advertises shared RDMA-capable NICs where both subnet and PKey
must match for pods to share the same device:

```yaml
apiVersion: resource.k8s.io/v1
kind: ResourceSlice
spec:
  driver: networking.example.com
  nodeName: node1
  sharingAffinity:
    - cel:
        subnet: |
          object.apiVersion == "networking.example.com/v1"
            && object.kind == "NICConfig"
            ? object.subnetID : ""
        pkey: |
          object.apiVersion == "networking.example.com/v1"
            && object.kind == "NICConfig"
            ? object.ibPKey : ""
  devices:
    - name: mlx5_0
      allowMultipleAllocations: true
      capacity:
        networking.example.com/slots:
          value: "16"
```

A matching claim provides both values inside its driver-defined opaque
config:

```yaml
config:
  - requests: ["rdma-nic"]
    opaque:
      driver: networking.example.com
      parameters:
        apiVersion: networking.example.com/v1
        kind: NICConfig
        subnetID: subnet-a
        ibPKey: "0x8001"
        vlan: "100"           # not extracted (no CEL declared for it)
```

Alpha matching behavior:

- If the device is clean, the first compatible claim sets the lock to:
  - `subnet = subnet-a`
  - `pkey = 0x8001`
- A later claim whose extraction yields the same `subnet` and `pkey`
  may share the device.
- A claim whose extraction yields `subnet = subnet-a` but `pkey =
  0x8002` is filtered out for that device because all declared keys
  must match.
- A claim whose opaque config has no `ibPKey` field will cause
  `object.ibPKey` to error during CEL evaluation; the device is
  filtered out and an Event is emitted on the pod.
- The `vlan` field is ignored because the driver did not declare a CEL
  expression for it. It flows through opaquely to the driver.

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
  - Filter: extraction yields no non-empty keys for the claim → device filtered out
  - Filter: claim with extra fields in opaque config beyond what the
    slice's CEL extracts → extra fields ignored, device passes if extracted
    keys match
  - Filter: claim has multiple opaque configs producing conflicting
    same-key values across one or more extractors → device filtered out
  - Filter: CEL evaluation error (missing field, non-string return, cost
    budget exhausted) → device filtered out, Event emitted, no silent
    empty lock
  - Filter: device with `AffinityStates[deviceID].Status == AffinityStatusUnreconstructable` is excluded
    for new sharing-affinity scheduling
  - First-fit verification: with two feasible devices on a node (one
    locked-compatible, one clean), allocator picks the first in
    ResourceSlice order (no affinity-aware preference; preference lands in
    beta)
  - Reserve: first claim sets lock; second claim with same extracted
    values succeeds
  - Reserve: second claim with conflicting extracted values fails
  - Unreserve: tentative lock is rolled back
  - Legacy claims with non-reconstructable affinity (no keys produced
    or CEL fails) cause the device to be marked unknown rather than
    establishing or joining a lock
  - Legacy-claim handling: all scenarios from the `Handling Legacy Claims
    with Unreconstructable Affinity` table
  - Compatibility matrix: device in a slice without `sharingAffinity`
    is unaffected — claims with or without matching opaque configs both
    pass (Legacy/Basic and Permissive Sharing rows)
  - Strict Gating: device's pool has `sharingAffinity` but at
    least one applicable group is not fully satisfied by the
    claim's opaque configs (at least one of its keys returns
    empty) → device filtered out
  - Multi-request scoping: claim with two requests (`mgmt-nic` and
    `data-nic`) each with distinct opaque configs → each request resolves
    independently; one request's extracted values do not influence the
    other's filter decision or lock state on a different device
- `staging/src/k8s.io/api/resource/v1`: Coverage for the new
  `ResourceSliceSpec.SharingAffinity` field, including:
  - Validation: `sharingAffinity` exceeding max 8 entries is rejected
  - Validation: per-entry `cel` map exceeding max 8 keys is rejected
  - Validation: CEL expressions are syntactically valid at admission
    (parse-check only; runtime cost is bounded per-eval at the scheduler)
  - Validation: a single `ResourceSlice` with both `devices` and
    `sharingAffinity` set is rejected (mutual exclusion)
  - Validation: two `sharingAffinity` entries with textually-identical
    (or normalized-equivalent) selectors that overlap on at least one
    CEL key name are rejected at admission (best-effort key-collision
    check); the case where both entries omit `selector` is included
  - Validation: per-entry `selector.cel.expression` is syntactically
    valid at admission (parse-check only); the same `DeviceSelector`
    validation path used by `DeviceRequest.selectors` is reused
  - Round-trip serialization of `SharingAffinityExtractor`
    (including the optional `Selector` field)

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
- Scheduler restart: `AffinityStates` correctly reconstructed by
  re-running CEL extraction against the opaque configs of existing
  bound ResourceClaims; devices with non-reconstructable active claims
  (no keys produced, CEL eval failure) have `AffinityStates[deviceID].Status == AffinityStatusUnreconstructable`
- Parallel scheduling: two Pods whose extracted affinity values conflict
  target the same device — one wins Reserve, the other is requeued
- `DRASharingAffinity` disabled: `sharingAffinity` field is ignored
  (or absent on writes); devices are treated as unconditionally shareable
- `DRASharingAffinity` toggled: enabling after claims exist does not
  disrupt already-bound workloads, and legacy in-use devices are
  conservatively filtered until clean
- Invalid opaque config at scheduling time: regression test that
  malformed configs (no apiVersion/kind, unparseable parameters) cause
  CEL extraction to fail deterministically and the device is filtered
  out rather than crashing the scheduler
- Permissive Sharing (slice has no AE): Slice without `sharingAffinity`,
  claim with arbitrary opaque config — verify scheduler allows
  the allocation and opaque config is not evaluated for affinity
- Ghost Lock: Pod is Assumed (tentative lock set) but Bind fails — verify
  the lock is cleared immediately and the next Pod in the queue can claim
  the device with a different affinity value
- Legacy Device Migration: 5 Pods are already running on NICs in a slice
  without `sharingAffinity`; the driver republishes the slice with
  `sharingAffinity` declared; a 6th Pod arrives with a matching opaque
  config — verify each affected device has
  `AffinityStates[deviceID].Status == AffinityStatusUnreconstructable` and the 6th Pod is filtered from
  those devices until all legacy claims drain
- Partial Key (heterogeneous group): Pool declares two applicability
  groups — a NIC group with `subnet` + `pkey`, and a VF group with
  `parentNIC`. Claim's opaque config has `subnetID` (so `subnet`
  resolves) but no `ibPKey` (so the NIC group is not fully
  satisfied), and is not a VFConfig (so the VF group is not
  satisfied either) — verify the device is filtered out (some
  applicable group not satisfied)
- Heterogeneous Pool Match: Same pool as above. Claim is a `NICConfig`
  with both `subnetID` and `ibPKey` set — verify the NIC group is
  satisfied and a NIC device in the pool is eligible. A second claim
  with `VFConfig` and `parentNICUUID` set — verify the VF group is
  satisfied and a VF device in the pool is eligible. Neither claim
  interferes with the other's lock state.
- Selector-Scoped Dispatch (heterogeneous pool): Pool declares a NIC
  extractor with `selector: device.attributes["type"].string == "nic"`
  and a VF extractor with `selector: device.attributes["type"].string == "vf"`.
  A `NICConfig` claim is evaluated against a VF device — the NIC
  extractor's selector does not match the VF device, so its CEL is
  not evaluated; the VF extractor's CEL returns empty for a
  `NICConfig`; no applicable group is satisfied; verify the
  device is filtered out. Conversely, the same `NICConfig` claim
  against a NIC device satisfies the NIC group and the device is
  eligible. Confirms per-extractor selector gating prevents
  wrong-group lock pollution and bounds CEL work to applicable
  extractors only.
- Factored Extractors (no-selector + per-kind): Pool declares a
  no-selector "common" extractor with key `vendor`, a NIC
  extractor (`type == "nic"`) with key `subnet`, and a VF
  extractor (`type == "vf"`) with key `parentNIC`. A `NICConfig`
  claim providing `vendor` and `subnetID` makes both applicable
  groups (common + NIC) satisfied on a NIC device — verify the
  device is eligible and the merged effective affinity map is
  `{vendor, subnet}`. The same claim missing `vendor` fails the
  common group — verify the device is filtered out (some
  applicable group not satisfied).
- Key Collision Across Applicable Extractors (slice-author error):
  Pool declares a no-selector extractor with key `subnet` and a
  NIC extractor (`type == "nic"`) also with key `subnet`. A claim
  evaluated against a NIC device makes both applicable. Verify the
  scheduler treats the device as non-viable for the request,
  emits an Event on the pod, and the claim stays Pending. After
  the driver republishes the slice with the colliding key renamed
  (e.g., the no-selector extractor's `subnet` becomes
  `commonSubnet`), the next scheduling cycle succeeds.
- Shared Key Across Disjoint Selectors (allowed): Pool declares a
  NIC extractor (`type == "nic"`) and a VF extractor
  (`type == "vf"`), both with key `subnet`. A NIC claim sees only
  the NIC extractor as applicable on a NIC device — verify the
  device is eligible and no collision is reported. The VF
  extractor's `subnet` does not collide because it never applies
  to the same device.
- First-Fit Behavior: Two devices available on a node, one already locked
  to subnet-X, one clean; new claim whose extraction yields subnet-X —
  verify the claim is allocated successfully (Filter excludes nothing;
  either device is feasible) and that the chosen device matches the
  allocator's existing first-fit ordering. Alpha does not require the
  locked device to be preferred; affinity-aware preference is delivered
  in beta
- Driver Backstop: Slice has no `sharingAffinity`, two claims with
  incompatible opaque config land on the same device — verify scheduler
  allows both (permissive), and `NodePrepareResources` rejects the
  incompatible claim
- NodePrepareResources failure does not clear lock: Claim is bound and
  lock is set in the scheduler cache, but `NodePrepareResources` fails
  on the node — verify the affinity lock remains in the scheduler cache
- Multi-request, multi-device, no cross-talk: Single claim with two
  requests (`mgmt-nic`, `data-nic`), each with a distinct opaque config
  yielding `subnet=A` and `subnet=B` targeting two different devices in
  slices that declare extraction — verify both requests succeed in the
  same scheduling cycle and each device locks to its own request's
  extracted values without influence from the sibling request
- Restart with inconsistent reconstructable locks: two active
  reconstructable claims on the same device produce conflicting extracted
  key values during restart reconstruction — verify the device is marked
  `AffinityStates[deviceID].Status = AffinityStatusUnreconstructable`, a warning is logged, and
  new sharing-affinity scheduling is blocked on that device until all
  claims drain
- `sharingAffinity` mutation with active claims: slice republishes
  with renamed CEL keys or a new extractor; pre-existing claims now extract a
  different key set than the current `LockedAffinity` — verify the device
  has `AffinityStates[deviceID].Status == AffinityStatusUnreconstructable` after the next
  reconciliation/restart and is filtered out for new sharing-affinity
  scheduling until existing claims drain
- CEL cost budget: pathological CEL expression intentionally exhausts the
  per-eval cost limit — verify the scheduler treats this as an extraction
  failure (device filtered out + Event), does not stall, and remains
  responsive to other scheduling work

##### e2e tests

- End-to-end test with mock DRA driver publishing `sharingAffinity`
- Multi-pod scheduling: Pods with matching extracted affinity values share the same device
- Multi-pod scheduling: Pods with conflicting extracted affinity values are placed on
  different devices
- Lock lifecycle: last Pod deleted → lock cleared → new Pod with different
  affinity value can claim the device
- Rollout scenario: existing Pods running on devices in a slice with no
  `sharingAffinity`; driver republishes the slice with
  `sharingAffinity`; verify existing Pods continue running and new Pods
  respect the new constraint after legacy claims drain

### Graduation Criteria

#### Alpha

- Feature implemented behind a single feature gate `DRASharingAffinity`
  covering both the `ResourceSlice.spec.sharingAffinity` API field and
  the scheduler logic that runs CEL extraction and matches results against
  the lock
- API field added to ResourceSlice (`SharingAffinityExtractor` on
  `ResourceSliceSpec`)
- Scheduler runs CEL extraction over claim opaque configs to derive
  affinity keys; CEL evaluates in the standard ValidatingAdmissionPolicy
  environment with a per-evaluation cost budget
- Scheduler Filter plugin enforces affinity matching
- Scheduler tracks affinity in `AllocatedState.AffinityStates`
- Unit and integration tests
- Documentation for driver authors covering CEL expression authoring,
  multi-version guard idioms (e.g., `object.apiVersion == "x/v1" && object.kind == "Foo" ? ... : ""`),
  and rollout guidance
- Alpha documentation explicitly calls out the lack of lock-breaking
  preemption semantics for incompatible locks
- Alpha documentation explicitly calls out string-only affinity matching
  (alpha CEL expressions must return `string`)
- Distinct alpha diagnostics emitted via scheduler logs and (best-effort)
  events for: (a) compatibility mismatch with the current lock, (b) no
  keys produced by any extractor for the claim, (c) CEL evaluation
  failure (including missing fields, non-string return, or cost-budget
  exhaustion), and (d) unknown lock state due to legacy /
  non-reconstructable / inconsistent active claims — sufficient
  to attribute a filtered scheduling decision without relying on metric
  labels alone

#### Beta

- Gather feedback from DRA driver developers
- Address any issues found in alpha
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

**Upgrade**: Existing ResourceSlices without `sharingAffinity`
continue to work. New field is additive. See the
[Compatibility Matrix](#compatibility-matrix) for how the scheduler and
driver behave across all combinations of slice `sharingAffinity`
and claim opaque config presence.

#### Recommended Rollout

This is a single-stage rollout: no claim-side API change is required,
and workloads keep their existing opaque configs throughout.

The recommended sequence:

1. **Enable the gate (`DRASharingAffinity`)** on apiserver and scheduler.
   Until any slice declares `sharingAffinity`, this is a no-op for
   scheduling.

2. **Publish `sharingAffinity` on ResourceSlices**: add the CEL
   extractor block to the relevant slices. Strongly recommended to do
   this on idle slices first (see "Adding `sharingAffinity` to an
   in-use slice" below). From this point on, the scheduler locks a
   device when a claim targeting it produces a non-empty CEL
   extraction; claims whose opaque configs produce no non-empty
   extractions are filtered out of those devices.

**Why this order**: when the `DRASharingAffinity` gate is OFF on the
apiserver, `sharingAffinity` is stripped from incoming ResourceSlice
writes (standard alpha-field handling). A driver that publishes the
field before the gate is enabled will see it silently dropped at write
time; enabling the gate later does not retroactively restore it, and
the driver must republish.

**Additional rollout consideration — workload-schema readiness**:
independent of step ordering, enabling extraction on slices before
workloads have rolled to a compatible opaque-config schema may cause
those workloads to be filtered out of the affected devices on their
next scheduling attempt, either landing them on non-extraction slices
(capacity strand) or driving them to `Pending` if no non-extraction
slices exist. The slice update should be coordinated with any required
workload-side opaque-config evolution.

For the common case where the driver's opaque config schema is
unchanged and the slice's CEL expressions simply pull existing fields
out, no workload-side change is needed at all.

**Minimizing capacity stranding**: when adding `sharingAffinity` to
slices for the first time, the ideal sequence is:

1. Wait for affected devices to be idle (clean).
2. Update the ResourceSlice to include the `sharingAffinity` block.
3. Allow the scheduler to establish the first known lock with a new
   claim.

During mixed rollouts (some slices with `sharingAffinity`, some
without), Strict Gating automatically routes claims whose opaque
configs lack the declared keys onto non-extraction slices — they are
filtered out of extraction slices by design. The remaining gap is for
claims whose opaque configs *do* produce the keys: in alpha they pass
Filter on both extraction and non-extraction slices and the allocator
has no preference between them, so capacity can strand on either side
of the rollout. Affinity-aware preference is delivered in beta. For
predictable rollout behavior, drain devices before adding
`sharingAffinity`, as recommended above.

**Adding `sharingAffinity` to an in-use slice**: A driver may add or
update `sharingAffinity` on a slice whose devices already have bound
ResourceClaims. The scheduler handles this conservatively:

- Pre-existing claims continue to run and are not evicted.
- For each in-use device on the updated slice, the scheduler re-runs
  CEL extraction against every bound claim's opaque config and
  populates `LockedAffinity` if all claims yield a consistent key
  map. If extraction fails or claims yield conflicting maps, the
  device is marked `AffinityStatusUnreconstructable` and excluded
  from new sharing-affinity placements until all its claims drain.
- Devices on the slice with no bound claims are unaffected: they
  behave normally on next allocation.

**Driver Upgrades and Schema Evolution**: The opaque-config schema
and the slice's CEL extractors both evolve with driver releases. The
recommended playbook depends on the kind of change:

| Upgrade class | Recommended action |
|---|---|
| Driver patch with **no schema change, same CEL** | None. Driver republishes an identical slice; transparent to the scheduler. |
| Driver release with **additive schema fields**, same extracted keys | None. Existing CEL guards on `apiVersion`/`kind` continue to match; new fields are unread by the extractor and flow through to the driver. |
| Driver release adds **v2 schema alongside v1** | Extend each extractor's CEL with a `v1 OR v2` apiVersion guard reading the same field path (the canonical multi-version idiom). Both old and new workloads extract identically. After workloads have migrated off v1, a later driver release drops the v1 guard. |
| Driver release with **breaking schema rename** (e.g., `subnetID` → `subnet`) — *affects opaque-config field path only; declared key set unchanged* | Republish the slice with an extractor that reads both paths into the same key, e.g. `has(object.subnet) ? object.subnet : object.subnetID`. Both old and new workloads extract identically; the legacy branch can be dropped in a later driver release once workloads have migrated, or kept indefinitely (the cost is one ternary in the CEL string). Existing locks remain valid — no `Unreconstructable` transitions. |
| Driver release with **renamed extracted key** (e.g., `subnet` → `vpc-subnet`) — *affects the declared key set itself* | This is a `sharingAffinity` mutation: re-extraction against bound claims yields a different key map than the stored `LockedAffinity`. Devices with active legacy claims transition to `AffinityStates[deviceID].Status == AffinityStatusUnreconstructable` until they drain. Drivers should prefer to drain affected slices first (per the rollout sequence above) and stage the change to off-hours. |
| **DaemonSet rolling upgrade** across nodes | Each node's slice flips when its driver pod restarts. Transient heterogeneity across slices is bounded by the rollout window. Strict Gating plus `Unreconstructable` quarantine make the window safe but capacity-stranding; rolling-update `maxSurge` / `maxUnavailable` should be tuned with this in mind. |
| **Workload-side schema lag** (workloads behind driver) | Drivers should keep multi-version-guard CEL through at least one workload rollout cycle. Otherwise workloads hit "no keys produced → Strict Gating → Pending" until they upgrade. |
| **Driver pod removed or unhealthy** | Existing DRA behavior applies — the slice eventually becomes stale and is garbage-collected by the kubelet plugin manager. Devices in a stale slice are unschedulable regardless of `sharingAffinity`. No additional handling is introduced by this KEP. |

**Handling claims that produce no keys**: The scheduler treats a slice with
`sharingAffinity` as a protected resource. If a claim's opaque
configs yield no non-empty value from any declared extractor's CEL,
every device in that slice is filtered out for the claim. CEL evaluation
errors (missing field, non-string return, cost-budget exhaustion) are
treated the same way. API validation prevents most malformed CEL
expressions from reaching the scheduler in the first place (parse-check
at admission).

**Downgrade**: If the feature gate is disabled:

- Disabling `DRASharingAffinity` causes the API server to strip the
  `sharingAffinity` field from new or updated ResourceSlices, and
  the scheduler ignores the field on any persisted ResourceSlices that
  still carry it (e.g., objects created while the gate was on) —
  slices return to unconditional sharing (pre-KEP-5981 behavior).
- The DRA driver becomes the sole authority for enforcing hardware
  compatibility at `NodePrepareResources`.

### Version Skew Strategy

- **kube-apiserver**: Must be upgraded first to accept the new
  `sharingAffinity` field on `ResourceSlice`.
- **kube-scheduler**:
  - A scheduler that understands this feature enforces extraction,
    tracks `AffinityStates`, and may conservatively set
    `AffinityStates[deviceID].Status = AffinityStatusUnreconstructable` when effective affinity cannot
    be reconstructed.
  - An older scheduler ignores `sharingAffinity`. In that skew
    case, placement may be overly permissive and the DRA driver
    remains the final safety backstop during `NodePrepareResources`.
- **kubelet**: No changes required; kubelet does not interpret
  `sharingAffinity`.
- **DRA driver**:
  - Drivers publish ResourceSlices with the `sharingAffinity` field.
  - Drivers must continue validating actual hardware compatibility at
    prepare time, especially during skew where an older scheduler may
    not enforce affinity constraints.

During version skew, the main outcomes are permissive scheduling by an
older scheduler or conservative filtering by a newer scheduler when
affinity state cannot be reconstructed. Both are operationally safe as
long as the driver continues rejecting incompatible prepare-time
configurations.

## Production Readiness Review Questionnaire

### Feature Enablement and Rollback

###### How can this feature be enabled / disabled in a live cluster?

- [x] Feature gate
  - Feature gate name: `DRASharingAffinity` — gates
    `ResourceSlice.spec.sharingAffinity` and the scheduler logic
    that runs CEL extraction and enforces matching
  - Components depending on the feature gate: kube-apiserver, kube-scheduler

###### Does enabling the feature change any default behavior?

No. Slices without `sharingAffinity` behave exactly as before. The
feature only affects slices that explicitly opt-in via the new field.

###### Can the feature be disabled once it has been enabled (i.e. can we roll back the enablement)?

Yes. Disabling `DRASharingAffinity` causes:
- API server to strip the `sharingAffinity` field from new or
  updated ResourceSlices before persisting (writes succeed, field is
  not stored)
- Scheduler to ignore existing `sharingAffinity` fields for future
  placement decisions

Existing allocations continue to work. New allocations may become more
permissive, so the driver must continue validating compatibility at
prepare time.

###### What happens if we reenable the feature if it was previously rolled back?

The scheduler resumes enforcing `sharingAffinity` for future placement
decisions. Existing allocations are not evicted. However, ResourceSlices
that were created or updated while the gate was disabled will not have
the `sharingAffinity` field (it was stripped by the API server); drivers
must republish their slices with `sharingAffinity` for the feature to
take effect.

Because Filter was not enforcing affinity while the gate was disabled,
the scheduler may have allowed multiple bound claims onto the same
device whose opaque configs decode to *different* affinity values
(permissive admission). On reenable, the scheduler reconstructs lock
state per device by re-running CEL extraction over each device's active
claims. A device is marked `AffinityStatusUnreconstructable` if the
extracted key maps across its claims are inconsistent, or if extraction
produces no non-empty keys at all. Such devices are excluded from new
sharing-affinity placements until all active claims drain, after which
they become clean and can establish a lock normally.

###### Are there any tests for feature enablement/disablement?

Yes, unit tests will cover the feature gate behavior for API validation and scheduler logic.

### Rollout, Upgrade and Rollback Planning

###### How can a rollout or rollback fail? Can it impact already running workloads?

Rollout failure modes include:

- **Older scheduler after API enablement**: the scheduler ignores
  `sharingAffinity` and placement may be overly permissive. The
  driver remains the safety backstop at prepare time.
- **Newer scheduler enabling conservative handling on legacy in-use devices**:
  devices with non-reconstructable active claims may be filtered until they are
  clean, which can temporarily reduce effective schedulable capacity.

Rollback failure mode: if the scheduler is rolled back while the API server
still serves the field, placement returns to permissive behavior for new
scheduling decisions.

Running workloads are not evicted by this feature; the impact is on future
placement decisions, not on already-running pods.

###### What specific metrics should inform a rollback?

Each signal below names a metric trajectory and the condition under which rollback beats a forward fix.

- `sharing_affinity_unknown_device_total` does not decay over an
  extended window (remains near its post-enablement peak after the
  expected workload-churn window). Indicates legacy in-use devices are not
  draining naturally and conservative handling will continue to
  suppress schedulable capacity indefinitely. Rollback restores
  permissive sharing for these devices; forward-fix has no lever.
- `sharing_affinity_filter_mismatch_total` rises for claims that
  operators expect to be compatible (cross-check against
  `sharing_affinity_compatible_reuse_total` staying flat). Indicates
  the driver's CEL extractor is producing incorrect keys and
  over-filtering legitimate placements. Rollback re-enables fungible
  sharing while the extractor is corrected and re-published.
- `sharing_affinity_filter_missing_parameters_total` rises for
  workloads that previously placed successfully. Indicates
  extraction is silently dropping required parameters — typically
  claim opaque-config schema drift or extractor mis-targeting.
  Rollback restores placement while the driver re-publishes correct
  extractors.
- Sustained increase in unschedulable DRA-backed pods beyond
  pre-enablement baseline (post-enablement P95 stays elevated over
  the cluster's typical workload-churn window, with no corresponding
  workload increase). Indicates the combined effect of conservative handling and
  extractor-driven filtering is starving production placements
  faster than the feature's benefits accrue.
- Scheduler Filter/PreFilter latency regression measurable in
  `scheduler_framework_extension_point_duration_seconds` traceable
  to the affinity code path on DRA-heavy clusters. Rollback removes
  the per-claim CEL evaluation cost while the regression is profiled
  and fixed.
- Scheduler panics or crashes in the affinity code path that
  cannot be patched in a short window. Rollback (disable the feature
  gate) is the lower-risk remediation than rolling an in-flight fix
  to production schedulers.

Note: rising rates of *driver prepare-time rejections* are
**not** a rollback signal for this feature — those are exactly the
failure mode KEP-5981 reduces. If they rise after enablement, the
cause is upstream of the scheduler (driver bug, extractor bug) and
rolling back would make them more frequent, not fewer.

###### Were upgrade and rollback tested? Was the upgrade->downgrade->upgrade path tested?

Will be tested before beta.

###### Is the rollout accompanied by any deprecations and/or removals of features, APIs, fields of API types, flags, etc.?

No.

### Monitoring Requirements

###### How can an operator determine if the feature is in use by workloads?

Operators can determine usage by inspecting `ResourceSlice` objects that
declare `sharingAffinity` and by observing `ResourceClaim`s whose
opaque configs yield non-empty keys under one of the declared
extractors.

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

- rate of scheduling attempts filtered due to sharing-affinity mismatch
  (no keys produced, CEL eval error, or lock mismatch),
- rate of devices with `AffinityStates[deviceID].Status == AffinityStatusUnreconstructable`,
- share of successful placements that reuse already-locked compatible
  devices,
- prepare-time rejections by the DRA driver caused by incompatible hardware
  configuration.

###### Are there any missing metrics that would be useful to have to improve observability of this feature?

This feature would benefit from scheduler-observable counters and/or events for:

- `sharing_affinity_filter_mismatch_total` — claims rejected because the device's locked affinity values differ from the claim's extracted keys.
- `sharing_affinity_filter_missing_parameters_total` — claims rejected because extraction yielded no non-empty keys.
- `sharing_affinity_unknown_device_total` — devices in conservative quarantine because lock state cannot be reconstructed (legacy claims).
- `sharing_affinity_compatible_reuse_total` — successful placements onto an already-locked device whose affinity values matched (the intended-benefit counter).

Counters above give cluster-wide visibility; individual pods that fail to schedule also need a clear per-pod reason. The scheduler should surface a specific diagnostic on the pod's `PodScheduled=False` condition (and corresponding scheduling event) whenever an affinity check rejects a device, so the workload owner can tell *why* their pod is stuck. For example:

- claim's opaque configs produced no non-empty key under any
  `sharingAffinity` extractor on slice `<slice>` (device `<id>` filtered),
- CEL expression `<key>` failed to evaluate against claim's opaque
  config: `<error>` (device `<id>` filtered),
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

- ResourceSlice: One additional metadata slice per pool that uses
  this feature (carrying `sharingAffinity` and zero devices); the
  metadata slice holds up to 8 extractors, each with up to 8 CEL
  expressions. Device-bearing slices are unchanged.
- ResourceClaim: **No change.** The claim side carries no new fields.

###### Will enabling / using this feature result in increasing time taken by any operations covered by existing SLIs/SLOs?

Negligible. The Filter phase evaluates every extractor's CEL against
every opaque-config object on the claim. Worst-case CEL evaluation count per candidate device is
O(extractors × opaque-configs × keys), bounded by ≤8 extractors × N
opaque configs (typically ≤2) × ≤8 keys per extractor, and gated by
the per-evaluation cost budget (reused from the existing CEL
infrastructure used by ValidatingAdmissionPolicy). CEL programs are
compiled at slice-write admission time and cached; per-evaluation cost
is small and bounded. The lock-comparison itself is O(k) where k ≤ 8 — a
map lookup per key.

###### Will enabling / using this feature result in non-negligible increase of resource usage (CPU, RAM, disk, IO, ...) in any components?

No. The per-component impact is bounded:

- **Scheduler RAM**: `AffinityStates` adds one `map[string]string` (up to
  8 entries) per device with active affinity locks — proportional to
  active shared allocations, not total devices. A cluster with 1,000
  actively-shared devices carries well under 1 MB of state; unshared
  devices contribute zero. Compiled CEL programs are cached cluster-wide
  keyed by expression string; the cache is bounded by
  (≤8 extractors × ≤8 keys) = ≤64 unique programs per slice in the
  worst case, with cross-slice deduplication in practice.
- **Scheduler CPU**: Running CEL extraction during Filter is small
  per-candidate and bounded by the cost budget. Programs are compiled
  once and reused.
- **etcd disk**: Slightly larger ResourceSlice objects (bounded by the
  caps above). ResourceClaim size is unchanged.

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

- **No keys produced**: the claim's opaque configs do not yield any
  non-empty value from any declared `sharingAffinity` extractor on the
  slice; the device is filtered out for that claim.
- **CEL evaluation failure**: a CEL expression returns a non-string,
  references a missing field on the parsed opaque parameters, or
  exhausts the per-evaluation cost budget. The device is filtered out
  and an Event is emitted on the pod. The scheduler never silently
  establishes an empty lock.
- **Unreconstructable affinity state**: the device has active allocations whose
  affinity cannot be reconstructed (legacy claims, extractors changed
  such that they no longer produce keys, or CEL failure during
  reconstruction), so it is conservatively filtered until clean.
- **Prepare-time driver rejection**: despite scheduler filtering, the
  driver may still reject an incompatible or stale placement and that
  rejection is the final safety backstop.
- **Partial feature gate enablement**: if the feature gate is enabled
  on the API server but not the scheduler (or vice versa), the
  `sharingAffinity` field may be persisted but not enforced, or
  enforced from cached objects but unable to be persisted on new
  writes. Ensure the gate is enabled on both `kube-apiserver` and
  `kube-scheduler`.

###### What steps should be taken if SLOs are not being met to determine the problem?

Recommended debugging flow:

1. Inspect the relevant `ResourceSlice` and confirm it declares the
   expected `sharingAffinity` extractors (well-formed CEL
   expressions, expected key names).
2. Inspect the `ResourceClaim` and confirm at least one of its
   opaque-config objects has the shape the slice's CEL expressions
   expect (matching `apiVersion`/`kind` guards, populated fields).
3. Mentally (or via `kubectl` + a CEL test harness) run each CEL
   expression against the claim's parsed opaque parameters and verify
   it returns a non-empty string value.
4. Check whether the target device is already locked to incompatible
   values (lock state is in the scheduler's in-memory cache — check
   scheduler logs for filter reasons mentioning affinity mismatch).
5. Check whether the device is being treated as having **unknown
   affinity state** because of legacy or non-reconstructable active
   claims.
6. Review scheduler logs/events for explicit filter reasons (no keys
   produced, CEL eval failure, cost-budget exhaustion, lock mismatch,
   unknown state).
7. If the scheduler allowed placement but the driver rejected prepare,
   inspect driver logs to determine whether the issue was stale
   scheduler state, unsupported config, or an actual device-level
   incompatibility.

## Implementation History

- 2026-03-27: Initial KEP issue created
- 2026-03-30: KEP document drafted
- 2026-04-28: Pivoted from a well-known JSON schema inside
  `OpaqueDeviceConfiguration` to a typed `Structured` sibling field on
  `DeviceConfiguration`, based on wg-device-management feedback that the
  scheduler should not interpret opaque payloads.
- 2026-05-14: Pivoted again from the typed `Structured` sibling on
  `DeviceConfiguration` to driver-published CEL extraction on
  `ResourceSlice.spec.sharingAffinity`, per @johnbelamaric and
  @pohly review feedback on PR #5987. The claim side reverts to the
  status-quo opaque config (no new typed claim-side surface), and the
  scheduler's only assumption becomes "find some GVK objects to which
  it can apply CEL." Removed the `DRAStructuredDeviceConfiguration`
  feature gate (no longer needed). Consolidated into a single
  `DRASharingAffinity` gate.

- 2026-05-16: Per @pohly review feedback (#discussion_r3246775761),
  dropped the `GVK` field from `SharingAffinityExtractor`. GVK
  matching is now the CEL author's responsibility: a CEL expression
  guards on `object.apiVersion` and `object.kind` and returns the
  empty string when it does not apply to a given opaque-config
  object. The scheduler unions non-empty returns across all
  (extractor, opaque-config) pairs. Simplifies the API and lets one
  extractor handle multiple compatible config versions (e.g.,
  v1beta1 + v1 with same field paths) without duplication.
- 2026-05-17: Added a Driver Upgrades and Schema Evolution playbook
  under the Recommended Rollout section, covering schema-additive,
  multi-version-add, breaking-rename, key-rename, DaemonSet rolling
  upgrade, workload-side lag, and driver-pod removal cases. Codifies
  the multi-version-guard CEL idiom as the recommended upgrade
  primitive for breaking schema changes.

## Drawbacks

- Adds a new cache dimension (`AffinityStates`) to the scheduler's
  allocation tracking, increasing the surface area for reconstruction
  bugs on restart
- Once a device is locked, its effective affinity cannot change until
  all claims on that device are released
- Fragmentation risk remains if affinity values are too fine-grained
- Conservative handling of legacy in-use devices can temporarily strand
  schedulable capacity during rollout or migration
- If a pool declares `sharingAffinity` but a claim's opaque configs
  do not fully satisfy every applicable group for a
  given candidate device (every key in those groups' `cel` maps
  non-empty), that device is filtered out for that claim by the
  "Strict Gating" rule; if no device in the pool has all applicable
  groups satisfied by the claim, the entire pool is filtered out.
  Drivers should coordinate with workload teams to ensure claims
  carry compatible opaque configs matching a published group before
  enabling `sharingAffinity` on the pool.
- Per-evaluation CEL cost is bounded but non-zero; under pathological
  CEL expressions a slice could elevate per-candidate Filter cost. The
  per-eval cost budget and admission-time parse check are the primary
  mitigations.
- Affinity locks are purely in-memory with no API or status field to
  inspect which devices are locked to which values. Debugging lock
  state in alpha requires scheduler logs; a future enhancement (tracked
  under Beta graduation) is to surface effective lock state via
  scheduler-side metrics, scheduler Events on Pending pods, aggregation
  over existing `ResourceClaim.status.allocation`, or a dedicated
  scheduler-owned API resource — exact mechanism TBD.

## Alternatives

### Well-known JSON schema inside OpaqueDeviceConfiguration

An earlier iteration of this KEP supplied claim-side affinity values via a
scheduler-recognized JSON schema embedded in `OpaqueDeviceConfiguration` —
i.e., a magic `driver: resource.k8s.io` opaque payload with
`apiVersion: resource.k8s.io/v1alpha1, kind: StructuredParameters` that the
scheduler would decode at Filter time.

**Rejected because**:

- **Violates the contract of `Opaque`**: `OpaqueDeviceConfiguration` is, by
  name and design, opaque to the *core* API; only the driver is supposed to
  own its schema. A reserved global driver namespace decoded by the scheduler
  effectively turns part of the opaque payload into a core API surface
  without giving it API-server validation.
- **Weakly typed**: A JSON-schema-inside-string approach is not strongly
  typed. API validation cannot enforce the schema, so most invariants
  shift to scheduler-side runtime decode/validation.
- **Constrained evolution path**: Schema versioning lives inside an opaque
  blob rather than in the Kubernetes API surface.

### Typed `Structured` sibling on `DeviceConfiguration`

A subsequent iteration of this KEP added a typed `Structured` sibling on
`DeviceConfiguration` so the claim could carry affinity values in a
strongly-typed, API-validated map:

```go
type DeviceConfiguration struct {
    Opaque     *OpaqueDeviceConfiguration
    Structured *StructuredDeviceConfiguration   // proposed
}

type StructuredDeviceConfiguration struct {
    Requests   []string
    Parameters map[string]StructuredParameterValue
}
```

The scheduler would read `Structured.Parameters` directly to extract
affinity values, with no need to interpret opaque payloads.

**Rejected because**:

- **New typed surface on a core API**: Adds a new field to the
  `resource.k8s.io` API group that exists solely to serve scheduler-readable
  parameter extraction — a non-trivial API surface cost for a single
  consumer, when the underlying claim already carries the same data in
  driver-specific form via `Opaque`.
- **Forces claim duplication**: For most realistic claims the affinity keys
  (subnet, pkey, vlan, …) are already present in the driver's opaque config.
  The `Structured` field would require the claim author to duplicate those
  values in a second, scheduler-visible location, with no automatic
  enforcement that the two stay consistent.
- **Reviewer guidance (per @johnbelamaric, @pohly on PR #5987)**: The
  scheduler does not need a new typed claim-side field; it needs a way to
  *extract* affinity keys from the existing opaque config. The right
  mechanism for that extraction is driver-published CEL — which the driver
  already owns the schema for — placed on the ResourceSlice next to the
  device declarations.

### Claim-side-only SharingAffinity (on DeviceRequest)

An alternative design adds a dedicated `SharingAffinity` field directly on
`DeviceRequest` within ResourceClaim, with no corresponding declaration on
the device or ResourceSlice. Sharing intent is expressed entirely from the
consumer side — the device is mute on whether or how it can be shared:

```go
type DeviceRequest struct {
    // ... existing fields ...
    SharingAffinity *SharingAffinity
}

type SharingAffinity struct {
    AffinityKey string
    Value       string
    Strategy    SharingStrategy
}
```

**Rejected because**:

- **Wrong layer**: Sharing affinity is a property of how a *device* can be
  shared (a hardware-modal constraint declared by the driver), not a
  property of an individual request. Putting it on `DeviceRequest` would
  imply the consumer chooses the strategy, when in reality the device (and
  its driver) dictates which keys must agree across consumers.
- **Single key only**: The shape above implies one `AffinityKey`/`Value`
  pair per request. Multi-key affinity (e.g., `subnet` + `pkey` + `vlan`)
  would require either repeating the field or introducing a list, both of
  which converge structurally to "a map of typed parameters" — which is
  exactly what the adopted CEL extraction produces, without requiring a
  new claim-side field at all.

### Object Reference-based Affinity Matching

An alternative approach replaces inline affinity values with external
object references. Instead of extracting values from the claim's opaque
config, the claim would reference a CRD (e.g., `NetworkConfiguration`) by
name, and the device would declare which object kinds constrain sharing.

**Rejected because**:
- Requires new fields on both ResourceClaim and Device (or ResourceSlice),
  whereas the adopted approach adds only a single slice-level
  `sharingAffinity` field.
- Requires external CRD definitions, adding operational burden for cluster
  administrators.
- Multi-dimensional affinity: A device may need affinity on multiple
  independent axes (e.g., subnet + VLAN). With object references, each
  axis would need its own CRD.
- Indirect object references raise authorization and lifecycle concerns
  (who owns the CRD instance? what happens when it is deleted while claims
  reference it?).

### CEL on runtime scheduler lock state (rejected variant)

A related CEL-based design — considered before the design pivot — would
have had the driver publish a CEL expression on the ResourceSlice that
evaluates whether a claim is *compatible with the device's current lock
state*:

```yaml
sharingAffinity:
  lockExpression: >
    device.affinityLock['subnet'] == '' ||
    device.affinityLock['subnet'] == claim.AffinityValues['subnet']
```

**Rejected because**:
- `device.affinityLock` is runtime scheduler state, not a static device
  attribute. Exposing it in CEL requires extending the evaluation context
  to include the scheduler's in-memory `AllocatedState`, which breaks the
  current model where CEL evaluates against the ResourceSlice snapshot.
- CEL expressions are powerful but opaque to the scheduler — it cannot
  extract *which* keys constrain sharing or *what* values to record in
  `AllocatedState`. The scheduler would need to both evaluate the
  expression AND separately track lock state, duplicating logic.
- A circular variant — where one claim's eligibility depends on another
  claim's allocation — produces non-deterministic results depending on
  evaluation order.

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

*Removed in PR review (2026-05): lock-breaking preemption is not a
KEP-5981 deliverable. See [Composition with DRA Preemption
(KEP-5690)](#composition-with-dra-preemption-kep-5690) under
Notes/Constraints/Caveats.*

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
tracking via CEL-extracted parameters.

### Soft / Preferred Affinity Keys

The Alpha design enforces hard all-or-nothing matching at the
applicability-group level: every key in the satisfied group must
agree with the device's current lock, or the device is filtered out.
Real-world hardware may have hierarchical constraints where some keys
are strict sharing requirements (e.g., Subnet) and others are
scheduling preferences (e.g., Traffic-Class or bandwidth profile).

A future enhancement could let the driver mark individual CEL
expressions as `required` vs `preferred`:

- **`required`** (default): Mismatch → device filtered out; key contributes to the lock (current behavior).
- **`preferred`**: Mismatch → device passes Filter but is deprioritized in device selection; key does not contribute to the lock.

## Infrastructure Needed

None
