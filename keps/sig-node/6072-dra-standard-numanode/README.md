# KEP-6072: DRA: Standard `numaNode` Device Attribute

<!-- toc -->
- [Release Signoff Checklist](#release-signoff-checklist)
- [Summary](#summary)
- [Motivation](#motivation)
  - [Goals](#goals)
  - [Non-Goals](#non-goals)
- [Proposal](#proposal)
  - [Standard Attribute Definition](#standard-attribute-definition)
  - [List Construction Algorithm](#list-construction-algorithm)
  - [Helper Functions](#helper-functions)
  - [Why Not pcieRoot Alone](#why-not-pcieroot-alone)
  - [Current Driver Attribute Names](#current-driver-attribute-names)
  - [User Stories](#user-stories)
    - [Story 1: ML Engineer deploys inference pod](#story-1-ml-engineer-deploys-inference-pod)
    - [Story 2: Platform admin partitions GPU node](#story-2-platform-admin-partitions-gpu-node)
    - [Story 3: Cross-quadrant I/O co-placement under NPS4](#story-3-cross-quadrant-io-co-placement-under-nps4)
  - [Risks and Mitigations](#risks-and-mitigations)
- [Design Details](#design-details)
  - [API Changes](#api-changes)
  - [Helper Implementation](#helper-implementation)
    - [Platform scope](#platform-scope)
  - [Driver Changes](#driver-changes)
    - [Choosing the list or scalar form](#choosing-the-list-or-scalar-form)
  - [Consumer Expectations](#consumer-expectations)
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
    - [Component skew during a rolling upgrade](#component-skew-during-a-rolling-upgrade)
    - [Decoupling from KEP-5491 graduation](#decoupling-from-kep-5491-graduation)
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
  - [Do nothing — use vendor-specific names](#do-nothing--use-vendor-specific-names)
  - [Use dra.net/numaNode as the informal convention](#use-dranetnumanode-as-the-informal-convention)
  - [numaNode as a scalar int](#numanode-as-a-scalar-int)
  - [Separate numaNode (int) and localNUMANodes (list)](#separate-numanode-int-and-localnumanodes-list)
- [Infrastructure Needed](#infrastructure-needed)
<!-- /toc -->

## Release Signoff Checklist

Items marked with (R) are required *prior to targeting to a milestone / release*.

- [ ] (R) Enhancement issue in release milestone, which links to KEP dir in [kubernetes/enhancements] (not the initial KEP PR)
- [ ] (R) KEP approvers have approved the KEP status as `implementable`
- [ ] (R) Design details are appropriately documented
- [ ] (R) Test plan is in place, giving consideration to SIG Architecture and SIG Testing input (including test refactors)
  - [ ] e2e Tests for all Beta API Operations (Endpoints)
  - [ ] (R) Ensure GA e2e tests meet requirements per https://github.com/kubernetes/community/blob/master/sig-testing/e2e-tests.md
- [ ] (R) Graduation criteria is in place
- [ ] (R) Production readiness review completed
- [ ] (R) Production readiness review approved
- [ ] "Implementation History" section is up-to-date for milestone
- [ ] User-facing documentation has been created in [kubernetes/website], for publication to [kubernetes.io]
- [ ] Supporting documentation — e.g., additional design documents, links to mailing list discussions/SIG meetings, relevant PRs/issues, release notes

## Summary

Standardize `resource.kubernetes.io/numaNode` as a well-known DRA device attribute alongside the existing `pcieRoot` and `pciBusID`. The value can be:

- A **scalar int** — the device's physical NUMA node (from the kernel's `numa_node` sysfs entry). Works without any feature gate.
- An **integer list** (requires KEP-5491 `DRAListTypeAttributes` feature gate) — physical NUMA node first, followed by same-socket NUMA nodes at the minimum ACPI SLIT distance

This enables cross-driver NUMA co-placement via `matchAttribute` with KEP-5491 non-empty intersection matching:

```yaml
constraints:
- matchAttribute: resource.kubernetes.io/numaNode
  requests: [gpu, nic, cpu, mem]
```

CPU device `4` (scalar) ∩ NIC `[6, 4, 5, 7]` = `{4}` ≠ ∅ → match. (`matchAttribute` compares a scalar from one driver against a list from another, so the CPU driver can publish a plain `4` rather than a single-element list.)

Today, six DRA drivers publish NUMA node information under five different vendor-specific attribute names. `matchAttribute` requires a common name. Users cannot write a cross-driver NUMA co-placement constraint without middleware. This proposal standardizes the name and semantics so that one constraint works across all drivers.

## Motivation

### Goals

- Standardize `resource.kubernetes.io/numaNode` (integer list) as a well-known device attribute in the `deviceattribute` library
- Use ACPI SLIT distances with a socket boundary filter to determine which NUMA nodes are local to each device
- Provide `GetNUMANodeAttributeByPCIBusID()` and `GetNUMANodeAttribute()` helper functions for driver authors
- Enable cross-driver NUMA co-placement via `matchAttribute` with KEP-5491 intersection matching
- Restore the NUMA coordination capability that was lost when devices moved from device plugins to DRA

### Non-Goals

- Standardize `cpuSocketID` — this is correlated with `numaNode` (coarser grouping), not orthogonal. Can be proposed separately if needed.
- Add `enforcement: preferred` to `matchAttribute` — independently useful but separable. Can be proposed as a separate KEP.
- Require drivers to publish `numaNode` — drivers MAY publish it. The standardization defines the name and semantics for those that do.
- Discover NUMA topology on Windows. The SLIT-based helper functions are Linux-only (they read sysfs and the ACPI SLIT). The attribute name and matching semantics are platform-neutral, so a Windows DRA driver could publish `numaNode` through a Windows-native mechanism in the future, but that is out of scope for this KEP. See [Platform scope](#platform-scope).

## Proposal

### Standard Attribute Definition

Add `resource.kubernetes.io/numaNode` to the set of well-known device attributes:

| Attribute | Type | Requires | Semantics |
|-----------|------|----------|-----------|
| `resource.kubernetes.io/numaNode` | int or int list | int: none; list: `DRAListTypeAttributes` | Physical NUMA node (scalar) or physical NUMA node first, followed by same-socket NUMA nodes at minimum SLIT distance (list). |

Drivers MAY publish `numaNode` as a **scalar int** (the physical NUMA node only). Drivers SHOULD publish it as an **integer list** when `DRAListTypeAttributes` (KEP-5491) is enabled. `matchAttribute` handles both — it can match a scalar from one driver against a list from another.

This makes the attribute immediately useful for same-NUMA matching without requiring `DRAListTypeAttributes`, while enabling richer cross-quadrant co-placement when list attributes are available.

This attribute measures **memory topology** — which memory controllers are closest to the device. It is orthogonal to `pcieRoot`, which measures **bus topology** — which PCIe switch tree the device sits behind.

### List Construction Algorithm

The `numaNode` list is built from two filters applied to the ACPI SLIT distance matrix:

1. **Minimum distance** — only NUMA nodes at the minimum non-self SLIT distance from the device's physical node are included
2. **Same socket** — nodes on a different socket (different `physical_package_id`) are excluded, even if their SLIT distance matches

The physical NUMA node (from sysfs `numa_node`) is always the first element.

Entries in the list are **unordered and equal-weight**: every node in the list is at the same minimum SLIT distance on the same socket, so there is no basis to rank them. Nodes that are *not* equally local are excluded by the minimum-distance filter rather than down-ranked within the list.

**CPU and memory devices** typically publish their physical NUMA node as a scalar `N` or single-element list `[N]`. Under NPS/SNC configurations where multiple NUMA nodes share symmetric memory controller access (e.g., AMD EPYC with memory controllers on the I/O die), drivers MAY publish a multi-element list. What each driver publishes is up to the driver authors.

**I/O devices** (GPUs, NICs, NVMe) publish `[physical, nodes at min SLIT distance on same socket...]`. On hardware where multiple NUMA nodes are equidistant to an I/O device (e.g., AMD EPYC chiplets under NPS4 where all same-socket NUMA nodes are at distance 12), the list includes all same-socket nodes. On future multi-IOD hardware with asymmetric intra-socket SLIT distances (e.g., Intel Xeon 6 with 2 IODs per socket), the minimum distance filter would naturally narrow the list to same-IOD nodes only.

**Examples (AMD EPYC 9825, NPS4, 8 NUMA nodes):**

| Device | Physical NUMA | SLIT distances | `numaNode` value |
|--------|--------------|----------------|------------------|
| CPU on NUMA 4 | 4 | (is NUMA node) | `4` (scalar) or `[4]` (list) |
| Memory on NUMA 4 | 4 | (is NUMA node) | `4` (scalar) or `[4]` (list) |
| NVMe on NUMA 0 | 0 | 10, 12, 12, 12, 32, 32, 32, 32 | `0` (scalar) or `[0, 1, 2, 3]` (list) |
| NIC VF on NUMA 6 | 6 | 32, 32, 32, 32, 12, 12, 10, 12 | `6` (scalar) or `[6, 4, 5, 7]` (list) |

**matchAttribute matching:**

Scalar-to-scalar (without `DRAListTypeAttributes`):

| Constraint | CPU `4` | NIC `6` | Match | Result |
|------------|---------|---------|-------|--------|
| `matchAttribute: numaNode` | 4 | 6 | 4 ≠ 6 | no match ✓ |

| Constraint | CPU `4` | NIC `4` | Match | Result |
|------------|---------|---------|-------|--------|
| `matchAttribute: numaNode` | 4 | 4 | 4 = 4 | match ✓ |

Scalar-to-list and list-to-list (with `DRAListTypeAttributes` — intersection matching):

| Constraint | CPU `4` (scalar) | NIC `[6, 4, 5, 7]` (list) | Intersection | Result |
|------------|------------------|---------------------------|-------------|--------|
| `matchAttribute: numaNode` | `{4}` | `{6, 4, 5, 7}` | `{4}` ≠ ∅ | match ✓ |

| Constraint | CPU `0` (scalar) | NIC `[6, 4, 5, 7]` (list) | Intersection | Result |
|------------|------------------|---------------------------|-------------|--------|
| `matchAttribute: numaNode` | `{0}` | `{6, 4, 5, 7}` | ∅ | no match ✓ |

### Helper Functions

Add to `k8s.io/dynamic-resource-allocation/deviceattribute`:

```go
// GetNUMANodeAttributeByPCIBusID returns the numaNode attribute for a PCI
// device. With listEnabled it returns the SLIT-based list (physical node
// first, then same-socket nodes at the minimum SLIT distance); otherwise it
// returns the scalar physical node. Returns an error for a device with no
// NUMA affinity.
func GetNUMANodeAttributeByPCIBusID(pciBusID string, listEnabled bool, mods ...MachineModifier) (DeviceAttribute, error)

// GetNUMANodeAttribute returns the numaNode attribute for a device that
// already knows its NUMA node (CPU/memory). listEnabled selects list or scalar.
func GetNUMANodeAttribute(numaNode int, listEnabled bool, mods ...MachineModifier) (DeviceAttribute, error)

// GetNUMANodeForCPU returns the NUMA node ID for a given CPU core.
func GetNUMANodeForCPU(cpuID int, mods ...MachineModifier) (int, error)
```

### Why Not pcieRoot Alone

`pcieRoot` is the only standardized topology attribute today. It is insufficient for cross-driver NUMA co-placement because:

1. **Many GPUs don't share a PCIe root with any NIC.** On the Dell XE8640 (4x H100), 1 of 4 GPUs shares a switch with a NIC. On the Dell R760xa (2x A40), 0 of 2. `matchAttribute: pcieRoot` excludes most GPUs.

2. **CPUs and memory have no pcieRoot.** They are not PCI devices. Any `pcieRoot` constraint including CPU or memory requests is unsatisfiable.

3. **Board layouts vary.** On the Supermicro AS-8126GS-TNMR (8x MI325X), every GPU has a co-located NIC on the same PCIe switch — `pcieRoot` gives 100% coverage. On the Dell R7725, GPU expansion slots and NICs are on different IOD quadrants — `pcieRoot` gives 0%. `numaNode` as a list handles both via SLIT-based intersection.

### Current Driver Attribute Names

Each driver publishes NUMA under a different name:

| Driver | Primary attribute | Compatibility aliases |
|--------|------------------|----------------------|
| NVIDIA GPU | `numa` (VFIO only) | — |
| AMD GPU | `numaNode` | — |
| dranet | `dra.net/numaNode` | — |
| SR-IOV NIC | `dra.net/numaNode` | — |
| CPU | `dra.cpu/numaNodeID` | `dra.net/numaNode` |
| Memory | `dra.memory/numaNode` | `dra.cpu/numaNodeID`, `dra.net/numaNode` |

With standardization, each driver publishes `resource.kubernetes.io/numaNode` alongside its vendor-specific name. The alias tables become unnecessary.

### User Stories

#### Story 1: ML Engineer deploys inference pod

An ML engineer deploys a vLLM inference pod with 1 GPU, 1 NIC, and CPU cores. They need RDMA between the GPU and NIC to stay within one memory controller domain.

```yaml
constraints:
- matchAttribute: resource.kubernetes.io/numaNode
  requests: [gpu, nic, cpu]
```

One constraint, three drivers. The GPU publishes `[0, 1, 2, 3]` (equidistant to 4 NUMA nodes on its socket), the NIC publishes `[0, 1, 2, 3]` (same socket), and the CPU publishes `[0]`. Intersection `{0}` → all co-placed on Socket 0.

#### Story 2: Platform admin partitions GPU node

A platform admin partitions a multi-GPU node for 4 independent inference pods. Each pod gets 1 GPU + 1 SR-IOV NIC VF + CPUs, with `matchAttribute: resource.kubernetes.io/numaNode`. The consumable CPU capacity naturally spreads pods across NUMA nodes as each fills up — no per-node CEL selectors needed.

#### Story 3: Cross-quadrant I/O co-placement under NPS4

On a Dell R7725 (AMD EPYC 9825, NPS4), GPU expansion slots are on NUMA 5 and NICs are on NUMA 6 — different IOD quadrants. A scalar `numaNode` would fail to co-place them. With the list attribute:

- GPU: `numaNode: [5, 4, 6, 7]`
- NIC: `numaNode: [6, 4, 5, 7]`
- Intersection: `{4, 5, 6, 7}` ≠ ∅ → match ✓

The SLIT distances show both devices are equidistant to the same set of NUMA nodes (all within the same socket), so they can be co-placed despite being on different quadrants.

### Risks and Mitigations

**Risk:** SNC (Intel) and NPS (AMD) change what NUMA IDs mean, potentially over-constraining.

**Mitigation:** The SLIT-based list naturally adapts. Under NPS4, I/O devices list all same-socket nodes (minimum intra-socket distance). Under NPS1, the list is `[N]` (single node per socket). The list is always correct for the current NPS/SNC configuration because it reads the live SLIT distances, not a static assumption.

**Risk:** On future multi-IOD-per-socket hardware, all intra-socket SLIT distances might be equal even though IOD affinity differs.

**Mitigation:** If SLIT distances don't differentiate IODs, the list includes all same-socket nodes — which is a safe over-approximation. If SLIT does differentiate, the minimum distance filter naturally narrows to same-IOD nodes. No code changes needed.

## Design Details

### API Changes

No new API types. The standard attribute name `resource.kubernetes.io/numaNode` is added to the documented set of well-known device attributes in the `deviceattribute` library, alongside `pcieRoot` and `pciBusID`. The attribute uses the `IntValues` (list) field from `DeviceAttribute`, which requires the `DRAListTypeAttributes` feature gate (KEP-5491).

### Helper Implementation

```go
// In k8s.io/dynamic-resource-allocation/deviceattribute

const StandardDeviceAttributeNUMANode = StandardDeviceAttributePrefix + "numaNode"

// GetNUMANodeAttributeByPCIBusID reads numa_node from sysfs and returns the
// numaNode attribute. With listEnabled it applies the SLIT distance + socket
// boundary filters and returns the list form ([physicalNode, equidistant...]);
// otherwise it returns the scalar physical node. It returns an error for a
// device with no NUMA affinity (sysfs numa_node = -1).
func GetNUMANodeAttributeByPCIBusID(pciBusID string, listEnabled bool, mods ...MachineModifier) (DeviceAttribute, error)

// GetNUMANodeAttribute returns the numaNode attribute for a device that already
// knows its NUMA node (CPU/memory). listEnabled selects the list or scalar form.
func GetNUMANodeAttribute(numaNode int, listEnabled bool, mods ...MachineModifier) (DeviceAttribute, error)
```

All functions use the existing `machine` abstraction with `MachineModifier` for testability via mock sysfs.

#### Platform scope

The derivation helpers (`GetNUMANodeAttributeByPCIBusID`, `GetNUMANodeAttribute`) read Linux sysfs (`/sys/bus/pci/devices/<BDF>/numa_node`) and ACPI SLIT distances, so the automatic SLIT-based list construction is **Linux-only**. This matches the current state of DRA drivers, which are Linux-only in practice.

The standard attribute name and its semantics, however, are platform-neutral. The attribute is a plain integer (or integer list) value with no Linux-specific encoding, so a Windows DRA driver MAY publish `resource.kubernetes.io/numaNode` directly if it obtains NUMA topology through a Windows-native mechanism — the helper functions are a convenience for Linux drivers, not a requirement of the attribute. Windows NUMA topology discovery for DRA is out of scope for this KEP; if it is pursued, it would supply the value through a separate Windows-specific code path (see related Windows affinity work in KEP-4885) and reuse this same attribute name and matching semantics. No part of the scheduler-side `matchAttribute` intersection logic is platform-dependent.

### Driver Changes

Each DRA driver adds one call to publish the attribute. `listEnabled` comes from the driver's configuration (see [Choosing the list or scalar form](#choosing-the-list-or-scalar-form)):

```go
// I/O device (GPU, NIC, NVMe): list (when listEnabled) built from sysfs numa_node + SLIT distances
numaAttr, err := deviceattribute.GetNUMANodeAttributeByPCIBusID(pciBusID, listEnabled)
if err == nil {
    device.Attributes[numaAttr.Name] = numaAttr.Value
}

// CPU/memory device that already knows its NUMA node
numaAttr, err = deviceattribute.GetNUMANodeAttribute(numaID, listEnabled)
if err == nil {
    device.Attributes[numaAttr.Name] = numaAttr.Value
}
```

A device with no NUMA affinity returns an error from both helpers; the driver omits the attribute for it rather than publishing a meaningless value.

#### Choosing the list or scalar form

A DRA driver decides which form to publish; it does not detect the cluster's feature state at runtime. A driver cannot read the apiserver's feature gates directly (feature gates are per-process), and feature support is not atomic across components during an upgrade, so runtime discovery would be unreliable. Following the standard "components only use features the operator enabled" model, the form is an operator-supplied input:

- The `deviceattribute` helpers take a `listEnabled` argument. When true they return the list form; when false they return the scalar physical node. When unset it defaults to false (scalar), which is always valid regardless of the gate.
- The operator sets `listEnabled` on the driver, through whatever configuration the driver author exposes (a flag or config field), to match what the cluster has enabled. The operator enables `DRAListTypeAttributes` across the apiserver and scheduler first, then configures the driver to publish lists; on downgrade the driver is reconfigured first, then the cluster.
- If a driver is configured to publish the list while `DRAListTypeAttributes` is off in the cluster, the apiserver drops the list value, the resulting value-less attribute fails validation, and the publish is rejected. The driver should treat this as a fatal configuration error rather than adapting.

This selection is **transitional**. Once `DRAListTypeAttributes` is GA and unconditionally available, the list form is always valid, drivers can publish it unconditionally, and the `listEnabled` selection (and the scalar form) can be retired. The scalar value remains semantically a single-element set, so that retirement is a code simplification, not an API break for consumers.

### Consumer Expectations

Because a driver may publish either the scalar or the list form, consumers of `resource.kubernetes.io/numaNode` must account for both. How much this matters depends on how a consumer reads the attribute.

**Matching via `matchAttribute` (the recommended case).** A ResourceClaim that constrains `matchAttribute: resource.kubernetes.io/numaNode` is unaffected by the form. The constraint is evaluated as non-empty set intersection (KEP-5491), and a scalar is treated as a single-element set, so a scalar `4` matches a list `[4, 5, 6, 7]` because `{4}` intersects `{4, 5, 6, 7}`. Mixed forms across drivers work with no special handling. This is the recommended way to consume the attribute and requires nothing of the consumer.

**Reading the value directly.** A consumer that inspects the value itself (for example a controller aligning a workload to a device's NUMA node) must handle both the scalar field and the list field:

- The **physical NUMA node is always recoverable the same way**: it is the scalar value when the scalar field is set, or the first element of the list when the list field is set.
- By construction the first list element is the physical node. The full list is treated as an unordered, equal-weight set for matching; the first-element convention exists only so a direct reader can recover the physical node. The order of the remaining elements is not a priority.

```go
// physicalNUMANode recovers the device's physical NUMA node from either form.
func physicalNUMANode(attr resourceapi.DeviceAttribute) (int64, bool) {
    if attr.IntValue != nil {
        return *attr.IntValue, true
    }
    if len(attr.IntValues) > 0 {
        return attr.IntValues[0], true
    }
    return 0, false
}
```

**CEL selectors.** A CEL expression referencing the attribute must accommodate both forms: use `includes()` against the list and equality against the scalar, or write the expression for whichever form the target cluster produces.

**Absent attribute.** A device with no NUMA affinity does not publish `numaNode` at all. A consumer must treat the absence of the attribute as "NUMA node unknown", never as node 0, and must not require the attribute to be present.

As with the publishing side, this dual-form handling is **transitional**: once `DRAListTypeAttributes` is GA and unconditionally available, only the list form is published and consumers may assume the list representation.

### Test Plan

[x] I/we understand the owners of the involved components may require updates to existing tests to make this code solid enough prior to committing the changes necessary to implement this enhancement.

#### Prerequisite testing updates

None — this adds a new attribute and helper functions, does not modify existing behavior.

#### Unit tests

- `deviceattribute` package: tests for `GetNUMANodeAttributeByPCIBusID()`, `GetNUMANodeAttribute()`, and `GetNUMANodeForCPU()` with mock sysfs
- Tests with varying SLIT distance matrices (symmetric, asymmetric, single-node)
- Socket boundary filter tests (2-socket, NPS1 vs NPS4)

#### Integration tests

- DRA allocator integration test: CPU `[4]` and NIC `[4, 5, 6, 7]` are co-placed by `matchAttribute: resource.kubernetes.io/numaNode` (intersection `{4}` ≠ ∅)
- DRA allocator integration test: CPU `[0]` and NIC `[4, 5, 6, 7]` fail `matchAttribute` (intersection ∅)
- Consumable capacity test: multiple pods share the same CPU device with `matchAttribute` constraint

#### e2e tests

- End-to-end test with at least two DRA drivers publishing `resource.kubernetes.io/numaNode` as a list and a ResourceClaim with `matchAttribute` across them

### Graduation Criteria

This KEP standardizes a device attribute and ships helper library code rather than a gated API or controller, so the stages track **adoption and validation maturity** rather than the enablement lifecycle of a feature gate:

- **Alpha** — the attribute name, semantics, and helper functions exist and are tested, with at least one driver publishing the value. The name is documented as "can also be a list," so drivers adopting it early is forward-compatible and not an API break.
- **Beta** — enough independent drivers have adopted it (target: three) to confirm the name and SLIT-based semantics work across real hardware, with feedback from topology-aware workloads.
- **GA** — real-world cross-driver co-placement deployments demonstrate the standard is stable and sufficient.

Because the scalar form carries no feature gate (see [Decoupling from KEP-5491 graduation](#decoupling-from-kep-5491-graduation)), these stages gate confidence in the *convention*, not the availability of a gated code path.

#### Alpha

- Helper functions in `deviceattribute` library (scalar and list variants)
- Standard attribute name and semantics documented
- Unit and integration tests passing
- At least one upstream DRA driver publishes the attribute (scalar or list)
- List form depends on `DRAListTypeAttributes` feature gate (KEP-5491); scalar form has no dependency

#### Beta

- At least three DRA drivers publish the attribute
- Real-world feedback from topology-aware workloads
- No major outstanding bugs

#### GA

- At least two real-world deployments demonstrating cross-driver NUMA co-placement
- Allowing time for feedback from driver authors and workload operators

### Upgrade / Downgrade Strategy

**Upgrade:** Existing claims are unaffected. Drivers can start publishing `resource.kubernetes.io/numaNode` at any time — it's just a new attribute value.

**Downgrade:** If `DRAListTypeAttributes` is disabled, the API server **silently drops** the `IntValues` field from device attributes on write (the standard `dropDisabledFields` behavior), rather than rejecting the ResourceSlice. Existing ResourceSlices that already carry list values are preserved by ratcheting. Because `DeviceAttribute` is a strict union, a driver must not publish a `numaNode` attribute with only `IntValues` set in this state — the dropped field would leave a value-less attribute that fails validation. A driver configured for such a cluster publishes `numaNode` as a scalar int instead (`listEnabled=false`, losing the SLIT-based list), as described under [Choosing the list or scalar form](#choosing-the-list-or-scalar-form). Claims using `matchAttribute: numaNode` then revert to equality matching on the scalar.

### Version Skew Strategy

The attribute depends on KEP-5491 `DRAListTypeAttributes` for the list type. All components (API server, scheduler) must have this feature gate enabled. The scheduler must also have the allocator wiring fix (kubernetes/kubernetes#139332) to select the experimental allocator that implements intersection matching.

#### Component skew during a rolling upgrade

Intersection matching over list attributes is implemented only in the scheduler's **experimental** allocator, and that allocator is selected automatically when `DRAListTypeAttributes` is enabled in the scheduler (it is the only allocator variant whose supported-feature set includes list-type attributes). The "stable" and "incubating" allocators have no handling for list-valued attributes at all — a `matchAttribute` constraint that encounters one treats it as an unknown value type and fails the match.

The relevant gate is therefore the **scheduler's**, not the API server's. During a rolling upgrade where the API server has `DRAListTypeAttributes` enabled (so ResourceSlices store `IntValues`) but the scheduler does not yet have it enabled (old binary, or gate off):

- The scheduler runs the stable/incubating allocator, which does not understand list attributes, so a ResourceClaim using `matchAttribute: resource.kubernetes.io/numaNode` against list-valued attributes cannot satisfy the constraint.
- The claim is **not allocated** and the pod stays `Pending`. The match **fails closed** — there is no fallback to a looser comparison, and the device is never placed on a wrong/non-matching NUMA domain.
- Once the scheduler is upgraded and has the gate enabled, it switches to the experimental allocator, the pending claim is re-evaluated, and it allocates normally via intersection matching.

This is fail-safe and self-healing: the only effect during the skew window is delayed scheduling of claims that use list-valued `numaNode` matching, with no incorrect placement and no state to reconcile afterward. (Drivers that only publish the scalar form are unaffected, since scalar `matchAttribute` works on all allocator variants.)

#### Decoupling from KEP-5491 graduation

This KEP does not graduate in lockstep with `DRAListTypeAttributes`. There is no requirement that the two reach the same stage at the same time:

- The **scalar form** carries no dependency on KEP-5491 and is usable at any stage, including before `DRAListTypeAttributes` exists in a cluster. This is the baseline that makes the standard attribute name useful on its own.
- The **list form** simply consumes whatever `DRAListTypeAttributes` provides at its current stage. If KEP-5491 is at beta while this KEP is at alpha (or vice versa), drivers and the scheduler use the list type at the stage KEP-5491 happens to be in; this KEP adds no additional gating on top of it.

Consequently, KEP-5491 graduating, stalling, or being rolled back does not block or regress the scalar behavior of this attribute — it only changes whether the richer list form is available. The two KEPs are layered, not bound to a shared graduation schedule.

## Production Readiness Review Questionnaire

### Feature Enablement and Rollback

###### How can this feature be enabled / disabled in a live cluster?

- The standard attribute name and scalar int form require no feature gate.
- The integer list form depends on feature gate: `DRAListTypeAttributes` (KEP-5491)
- Components for list form: kube-apiserver (accepts `IntValues` in ResourceSlice), kube-scheduler (intersection matching in experimental allocator)
- The helper functions are library code used by drivers at build time.

###### Does enabling the feature change any default behavior?

No. The attribute is additive. Existing claims and drivers are unaffected.

###### Can the feature be disabled once it has been enabled?

Yes. Disabling `DRAListTypeAttributes` prevents new ResourceSlices with list attributes. Existing ResourceSlices retain their values but new ones must use scalar attributes. Claims with `matchAttribute: numaNode` fall back to equality matching.

###### What happens if we reenable the feature if it was previously rolled back?

Drivers will resume publishing `resource.kubernetes.io/numaNode` as a list attribute in their ResourceSlices. The scheduler will resume using intersection matching for `matchAttribute` constraints referencing it. No state migration is needed — ResourceSlices are recreated on driver startup, and claims are re-evaluated by the scheduler. Previously allocated claims are unaffected since allocation results are stored in the claim status, not derived from ResourceSlice attributes at runtime.

###### Are there any tests for feature enablement/disablement?

Tests will verify list attribute acceptance when `DRAListTypeAttributes` is enabled and that the `IntValues` field is silently dropped (not the whole object rejected) when disabled.

### Rollout, Upgrade and Rollback Planning

<!--
This section must be completed when targeting beta to a release.
-->

###### How can a rollout or rollback fail? Can it impact already running workloads?

TBD for beta.

###### What specific metrics should inform a rollback?

None specific to this feature. A rollback would be prompted by qualitative signals rather than a dedicated metric: ResourceClaims using `matchAttribute: resource.kubernetes.io/numaNode` failing to allocate, or DRA drivers reporting errors when publishing the attribute. Operators can watch existing DRA scheduling-failure signals and driver logs.

###### Were upgrade and rollback tested? Was the upgrade->downgrade->upgrade path tested?

TBD for beta.

###### Is the rollout accompanied by any deprecations and/or removals of features, APIs, fields of API types, flags, etc.?

No.

### Monitoring Requirements

<!--
This section must be completed when targeting beta to a release.
-->

###### How can an operator determine if the feature is in use by workloads?

TBD for beta. At alpha: check if any ResourceSlice contains a device attribute named `resource.kubernetes.io/numaNode` with an `IntValues` field, and if any ResourceClaim uses `matchAttribute: resource.kubernetes.io/numaNode`.

###### How can someone using this feature know that it is working for their instance?

TBD for beta. At alpha: a ResourceClaim with `matchAttribute: resource.kubernetes.io/numaNode` across multiple driver requests is successfully allocated (claim status shows devices from the same NUMA domain).

###### What are the reasonable SLOs (Service Level Objectives) for the enhancement?

None specific to this feature. It introduces no dedicated SLI, so there is no separate SLO; the relevant objective is simply that DRA allocation continues to meet its existing scheduling-latency and success expectations.

###### What are the SLIs (Service Level Indicators) an operator can use to determine the health of the service?

None specific to this feature. Health is observed indirectly through existing DRA behavior: ResourceClaims using `matchAttribute: resource.kubernetes.io/numaNode` either allocate successfully or remain pending, and the scheduler/driver logs surface allocation failures. No dedicated SLI is introduced.

###### Are there any missing metrics that would be useful to have in this category?

None. Nothing in core Kubernetes tracks which device-attribute names are used in match constraints, so there is no natural metric for this feature, and adding one would require new, potentially performance-sensitive instrumentation in the scheduler allocator for little operator value. This feature only standardizes a device-attribute name and ships helper library code; it adds no control loop or API surface whose health a metric would meaningfully report. (Per discussion with @pohly and @Champbreed.)

### Dependencies

###### Does this feature depend on any specific services running in the cluster?

No cluster services. The helper functions are library code linked into DRA driver binaries.

The list attribute type depends on KEP-5491 `DRAListTypeAttributes` feature gate being enabled on:
- **kube-apiserver** — to accept `IntValues` fields in ResourceSlice device attributes
- **kube-scheduler** — to use the experimental allocator that implements intersection matching for list attributes (requires kubernetes/kubernetes#139332)

If `DRAListTypeAttributes` is not enabled, drivers cannot publish list attributes and must fall back to scalar `numaNode` with equality matching.

### Scalability

###### Will enabling / using this feature result in any new API calls?

No. The attribute is published as part of existing ResourceSlice updates.

###### Will enabling / using this feature result in introducing new API types?

No. Uses existing `DeviceAttribute` with `IntValues` field from KEP-5491.

###### Will enabling / using this feature result in increasing size or count of the existing API objects?

Minimal. Each device gains one additional attribute entry. The list is typically 1-4 elements (matching the number of same-socket NUMA nodes).

###### Will enabling / using this feature result in increasing time taken by any operations?

Negligible. The SLIT distance read is a single file read plus socket lookup (~microseconds), performed during driver startup device discovery.

### Troubleshooting

<!--
This section must be completed when targeting beta to a release.
-->

###### How does this feature react if the API server and/or etcd is unavailable?

TBD for beta. At alpha: drivers cannot publish ResourceSlices if the API server is unavailable, but this is existing DRA behavior, not specific to this feature.

###### What are other known failure modes?

TBD for beta.

###### What steps should be taken if SLOs are not being met to determine the problem?

TBD for beta.

## Implementation History

- 2026-05-12: Initial KEP draft (numaNode as scalar int)
- 2026-05-27: Updated to integer list with SLIT-based construction (based on @kad feedback and testing)
- 2026-05-27: Implementation PRs:
  - kubernetes/kubernetes#139332 — Fix: pass ListTypeAttributes to AllocatorFeatures
  - johnahull/kubernetes `feature/standard-numanode-list-v2` — numaNode constant + SLIT helpers
- 2026-05-27: Tested on Dell R7725 (AMD EPYC 9825, NPS4) with 4 DRA drivers, 16 pods

## Drawbacks

- The list form depends on KEP-5491 `DRAListTypeAttributes` which is alpha in 1.36. If KEP-5491 doesn't graduate, the attribute is still useful as a scalar int for same-NUMA matching, but cross-quadrant co-placement (NPS4, shared I/O die) requires the list form.
- The SLIT-based list may include more NUMA nodes than strictly necessary on hardware where all intra-socket distances are equal. This is a safe over-approximation but may reduce scheduling precision.

## Alternatives

### Do nothing — use vendor-specific names

Drivers continue publishing NUMA under vendor-specific names. Cross-driver matching requires CEL selectors with hardcoded vendor attribute names or manual alias tables in each driver.

**Rejected because:** The alias tables are growing organically and inconsistently. No single name covers all drivers.

### Use dra.net/numaNode as the informal convention

Four of six drivers already publish or alias `dra.net/numaNode`. Make it the convention without formal standardization.

**Rejected because:** GPU drivers (NVIDIA, AMD) don't participate in the `dra.net` convention. The `dra.net/` prefix implies ownership by the dranet project, which is misleading for GPU and CPU devices.

### numaNode as a scalar int

Publish `numaNode` as a plain integer (the kernel's `numa_node` value). Use equality matching via `matchAttribute`.

**Rejected because:** On modern hardware with shared I/O dies (AMD EPYC chiplets, Intel Xeon 6 multi-IOD), a device is equidistant to multiple memory controllers. A scalar value captures only the kernel's reported node, not the full topology. Under NPS4, devices on different IOD quadrants within the same socket have different scalar `numaNode` values but are equidistant — equality matching fails for cross-quadrant co-placement that should succeed. This was the original KEP proposal and was updated based on @kad's feedback and testing.

### Separate numaNode (int) and localNUMANodes (list)

Publish two attributes: `numaNode` (scalar int, physical node) and `localNUMANodes` (list, SLIT equidistant set). Use `numaNode` for guest NUMA placement (KubeVirt VEP-115) and `localNUMANodes` for scheduling.

**Rejected because:** The first element of the list already IS the physical node, so a separate scalar is redundant. Having two attributes for the same concept creates confusion about which to use for `matchAttribute`. The single list with physical-first ordering serves both purposes — the scheduler uses intersection matching on the full list, and consumers that need the physical node read the first element.

## Infrastructure Needed

None.
