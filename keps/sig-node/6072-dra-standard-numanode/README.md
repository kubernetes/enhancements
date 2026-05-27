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
  - [Driver Changes](#driver-changes)
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
  - [Scalability](#scalability)
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

Standardize `resource.kubernetes.io/numaNode` as a well-known DRA device attribute alongside the existing `pcieRoot` and `pciBusID`. The value is an **integer list** (requires KEP-5491 `DRAListTypeAttributes` feature gate) where:

- The first element is the device's physical NUMA node (from the kernel's `numa_node` sysfs entry)
- Additional elements are same-socket NUMA nodes at the minimum ACPI SLIT distance

This enables cross-driver NUMA co-placement via `matchAttribute` with KEP-5491 non-empty intersection matching:

```yaml
constraints:
- matchAttribute: resource.kubernetes.io/numaNode
  requests: [gpu, nic, cpu, mem]
```

CPU device `[4]` ∩ NIC `[6, 4, 5, 7]` = `{4}` ≠ ∅ → match.

Today, six DRA drivers publish NUMA node information under five different vendor-specific attribute names. `matchAttribute` requires a common name. Users cannot write a cross-driver NUMA co-placement constraint without middleware. This proposal standardizes the name and semantics so that one constraint works across all drivers.

## Motivation

### Goals

- Standardize `resource.kubernetes.io/numaNode` (integer list) as a well-known device attribute in the `deviceattribute` library
- Use ACPI SLIT distances with a socket boundary filter to determine which NUMA nodes are local to each device
- Provide `GetNUMANodeListByPCIBusID()` and `GetNUMANodeList()` helper functions for driver authors
- Enable cross-driver NUMA co-placement via `matchAttribute` with KEP-5491 intersection matching
- Restore the NUMA coordination capability that was lost when devices moved from device plugins to DRA

### Non-Goals

- Standardize `cpuSocketID` — this is correlated with `numaNode` (coarser grouping), not orthogonal. Can be proposed separately if needed.
- Add `enforcement: preferred` to `matchAttribute` — independently useful but separable. Can be proposed as a separate KEP.
- Require drivers to publish `numaNode` — drivers MAY publish it. The standardization defines the name and semantics for those that do.

## Proposal

### Standard Attribute Definition

Add `resource.kubernetes.io/numaNode` to the set of well-known device attributes:

| Attribute | Type | Requires | Semantics |
|-----------|------|----------|-----------|
| `resource.kubernetes.io/numaNode` | int list | `DRAListTypeAttributes` | Physical NUMA node first, followed by same-socket NUMA nodes at minimum SLIT distance. For CPU/memory devices, single-element `[N]`. |

This attribute measures **memory topology** — which memory controllers are closest to the device. It is orthogonal to `pcieRoot`, which measures **bus topology** — which PCIe switch tree the device sits behind.

### List Construction Algorithm

The `numaNode` list is built from two filters applied to the ACPI SLIT distance matrix:

1. **Minimum distance** — only NUMA nodes at the minimum non-self SLIT distance from the device's physical node are included
2. **Same socket** — nodes on a different socket (different `physical_package_id`) are excluded, even if their SLIT distance matches

The physical NUMA node (from sysfs `numa_node`) is always the first element.

**CPU and memory devices** publish `[N]` — a single-element list, since they ARE a NUMA node. They don't need SLIT expansion because CPU/memory is definitionally local to exactly one memory controller.

**I/O devices** (GPUs, NICs, NVMe) publish `[physical, nodes at min SLIT distance on same socket...]`. On hardware where multiple NUMA nodes are equidistant to an I/O device (e.g., AMD EPYC chiplets under NPS4 where all same-socket NUMA nodes are at distance 12), the list includes all same-socket nodes. On future multi-IOD hardware with asymmetric intra-socket SLIT distances (e.g., Intel Granite Rapids with 2 IODs per socket), the minimum distance filter would naturally narrow the list to same-IOD nodes only.

**Examples (AMD EPYC 9825, NPS4, 8 NUMA nodes):**

| Device | Physical NUMA | SLIT distances | `numaNode` list |
|--------|--------------|----------------|-----------------|
| CPU on NUMA 4 | 4 | (is NUMA node) | `[4]` |
| Memory on NUMA 4 | 4 | (is NUMA node) | `[4]` |
| NVMe on NUMA 0 | 0 | 10, 12, 12, 12, 32, 32, 32, 32 | `[0, 1, 2, 3]` |
| NIC VF on NUMA 6 | 6 | 32, 32, 32, 32, 12, 12, 10, 12 | `[6, 4, 5, 7]` |

**matchAttribute intersection matching (KEP-5491):**

| Constraint | CPU `[4]` | NIC `[6, 4, 5, 7]` | Intersection | Result |
|------------|-----------|---------------------|-------------|--------|
| `matchAttribute: numaNode` | `{4}` | `{6, 4, 5, 7}` | `{4}` ≠ ∅ | match ✓ |

| Constraint | CPU `[0]` | NIC `[6, 4, 5, 7]` | Intersection | Result |
|------------|-----------|---------------------|-------------|--------|
| `matchAttribute: numaNode` | `{0}` | `{6, 4, 5, 7}` | ∅ | no match ✓ |

### Helper Functions

Add to `k8s.io/dynamic-resource-allocation/deviceattribute`:

```go
// GetNUMANodeListByPCIBusID returns the numaNode list attribute for a PCI
// device. First element is the physical NUMA node, additional elements are
// same-socket nodes at minimum SLIT distance.
func GetNUMANodeListByPCIBusID(pciBusID string, mods ...MachineModifier) (DeviceAttribute, error)

// GetNUMANodeList returns the numaNode list attribute for a device that
// already knows its NUMA node. Suitable for CPU and memory devices.
func GetNUMANodeList(numaNode int, mods ...MachineModifier) DeviceAttribute

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

// GetNUMANodeListByPCIBusID reads numa_node from sysfs, then applies
// SLIT distance + socket boundary filters.
func GetNUMANodeListByPCIBusID(pciBusID string, mods ...MachineModifier) (DeviceAttribute, error) {
    physicalNode := readNUMANode(pciBusID)     // /sys/bus/pci/devices/<BDF>/numa_node
    equidistant := getEquidistantNUMANodes(physicalNode)  // SLIT min distance + socket filter
    return DeviceAttribute{
        Name:  StandardDeviceAttributeNUMANode,
        Value: resourceapi.DeviceAttribute{IntValues: [physicalNode, equidistant...]},
    }
}

// GetNUMANodeList for devices that already know their NUMA node.
// Returns [N] for CPU/memory, or [N, equidistant...] for I/O devices.
func GetNUMANodeList(numaNode int, mods ...MachineModifier) DeviceAttribute
```

All functions use the existing `machine` abstraction with `MachineModifier` for testability via mock sysfs.

### Driver Changes

Each DRA driver adds one call to publish the list attribute:

```go
// I/O device (GPU, NIC, NVMe) — uses PCI BDF to read SLIT distances
numaAttr, err := deviceattribute.GetNUMANodeListByPCIBusID(pciBusID)
if err == nil {
    device.Attributes[numaAttr.Name] = numaAttr.Value
}

// CPU/memory device — publishes single-element [N]
device.Attributes[deviceattribute.StandardDeviceAttributeNUMANode] = resourceapi.DeviceAttribute{
    IntValues: []int64{int64(numaID)},
}
```

### Test Plan

[x] I/we understand the owners of the involved components may require updates to existing tests to make this code solid enough prior to committing the changes necessary to implement this enhancement.

#### Prerequisite testing updates

None — this adds a new attribute and helper functions, does not modify existing behavior.

#### Unit tests

- `deviceattribute` package: tests for `GetNUMANodeListByPCIBusID()`, `GetNUMANodeList()`, and `GetNUMANodeForCPU()` with mock sysfs
- Tests with varying SLIT distance matrices (symmetric, asymmetric, single-node)
- Socket boundary filter tests (2-socket, NPS1 vs NPS4)

#### Integration tests

- DRA allocator integration test: CPU `[4]` and NIC `[4, 5, 6, 7]` are co-placed by `matchAttribute: resource.kubernetes.io/numaNode` (intersection `{4}` ≠ ∅)
- DRA allocator integration test: CPU `[0]` and NIC `[4, 5, 6, 7]` fail `matchAttribute` (intersection ∅)
- Consumable capacity test: multiple pods share the same CPU device with `matchAttribute` constraint

#### e2e tests

- End-to-end test with at least two DRA drivers publishing `resource.kubernetes.io/numaNode` as a list and a ResourceClaim with `matchAttribute` across them

### Graduation Criteria

#### Alpha

- Helper functions in `deviceattribute` library
- Standard attribute name and list semantics documented
- Unit and integration tests passing
- At least one upstream DRA driver publishes the attribute
- Depends on `DRAListTypeAttributes` feature gate (KEP-5491)

#### Beta

- At least three DRA drivers publish the attribute
- Real-world feedback from topology-aware workloads
- No major outstanding bugs

#### GA

- At least two real-world deployments demonstrating cross-driver NUMA co-placement
- Allowing time for feedback from driver authors and workload operators

### Upgrade / Downgrade Strategy

**Upgrade:** Existing claims are unaffected. Drivers can start publishing `resource.kubernetes.io/numaNode` at any time — it's just a new attribute value.

**Downgrade:** If `DRAListTypeAttributes` is disabled, the API server will reject ResourceSlices with `IntValues` fields. Drivers would need to fall back to publishing `numaNode` as a scalar int (losing the SLIT-based list). Claims using `matchAttribute: numaNode` would revert to equality matching.

### Version Skew Strategy

The attribute depends on KEP-5491 `DRAListTypeAttributes` for the list type. All components (API server, scheduler) must have this feature gate enabled. The scheduler must also have the allocator wiring fix (kubernetes/kubernetes#139332) to select the experimental allocator that implements intersection matching.

## Production Readiness Review Questionnaire

### Feature Enablement and Rollback

###### How can this feature be enabled / disabled in a live cluster?

- Depends on feature gate: `DRAListTypeAttributes` (KEP-5491)
- Components: kube-apiserver (accepts `IntValues` in ResourceSlice), kube-scheduler (intersection matching in experimental allocator)
- The helper functions are library code used by drivers at build time.

###### Does enabling the feature change any default behavior?

No. The attribute is additive. Existing claims and drivers are unaffected.

###### Can the feature be disabled once it has been enabled?

Yes. Disabling `DRAListTypeAttributes` prevents new ResourceSlices with list attributes. Existing ResourceSlices retain their values but new ones must use scalar attributes. Claims with `matchAttribute: numaNode` fall back to equality matching.

###### Are there any tests for feature enablement/disablement?

Tests will verify list attribute acceptance when `DRAListTypeAttributes` is enabled and rejection when disabled.

### Scalability

###### Will enabling / using this feature result in any new API calls?

No. The attribute is published as part of existing ResourceSlice updates.

###### Will enabling / using this feature result in introducing new API types?

No. Uses existing `DeviceAttribute` with `IntValues` field from KEP-5491.

###### Will enabling / using this feature result in increasing size or count of the existing API objects?

Minimal. Each device gains one additional attribute entry. The list is typically 1-4 elements (matching the number of same-socket NUMA nodes).

###### Will enabling / using this feature result in increasing time taken by any operations?

Negligible. The SLIT distance read is a single file read plus socket lookup (~microseconds), performed during driver startup device discovery.

## Implementation History

- 2026-05-12: Initial KEP draft (numaNode as scalar int)
- 2026-05-27: Updated to integer list with SLIT-based construction (based on @kad feedback and testing)
- 2026-05-27: Implementation PRs:
  - kubernetes/kubernetes#139332 — Fix: pass ListTypeAttributes to AllocatorFeatures
  - johnahull/kubernetes `feature/standard-numanode-list-v2` — numaNode constant + SLIT helpers
- 2026-05-27: Tested on Dell R7725 (AMD EPYC 9825, NPS4) with 4 DRA drivers, 16 pods

## Drawbacks

- Depends on KEP-5491 `DRAListTypeAttributes` which is alpha in 1.36. If KEP-5491 doesn't graduate, this KEP is blocked.
- The SLIT-based list may include more NUMA nodes than strictly necessary on hardware where all intra-socket distances are equal. This is a safe over-approximation but may reduce scheduling precision.

## Alternatives

### Do nothing — use vendor-specific names

Drivers continue publishing NUMA under vendor-specific names. Cross-driver matching requires middleware (topology coordinator, CEL selectors with hardcoded names, manual alias tables in each driver).

**Rejected because:** The alias tables are growing organically and inconsistently. No single name covers all drivers.

### Use dra.net/numaNode as the informal convention

Four of six drivers already publish or alias `dra.net/numaNode`. Make it the convention without formal standardization.

**Rejected because:** GPU drivers (NVIDIA, AMD) don't participate in the `dra.net` convention. The `dra.net/` prefix implies ownership by the dranet project, which is misleading for GPU and CPU devices.

### numaNode as a scalar int

Publish `numaNode` as a plain integer (the kernel's `numa_node` value). Use equality matching via `matchAttribute`.

**Rejected because:** On modern hardware with shared I/O dies (AMD EPYC chiplets, Intel Granite Rapids multi-IOD), a device is equidistant to multiple memory controllers. A scalar value captures only the kernel's reported node, not the full topology. Under NPS4, devices on different IOD quadrants within the same socket have different scalar `numaNode` values but are equidistant — equality matching fails for cross-quadrant co-placement that should succeed. This was the original KEP proposal and was updated based on @kad's feedback and testing.

### Separate numaNode (int) and localNUMANodes (list)

Publish two attributes: `numaNode` (scalar int, physical node) and `localNUMANodes` (list, SLIT equidistant set). Use `numaNode` for guest NUMA placement (KubeVirt VEP-115) and `localNUMANodes` for scheduling.

**Rejected because:** The first element of the list already IS the physical node, so a separate scalar is redundant. Having two attributes for the same concept creates confusion about which to use for `matchAttribute`. The single list with physical-first ordering serves both purposes — the scheduler uses intersection matching on the full list, and consumers that need the physical node read the first element.

## Infrastructure Needed

None.
