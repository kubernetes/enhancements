# KEP-6072: DRA: Standard `numaNode` Device Attribute

<!-- toc -->
- [Release Signoff Checklist](#release-signoff-checklist)
- [Summary](#summary)
- [Motivation](#motivation)
  - [Goals](#goals)
  - [Non-Goals](#non-goals)
- [Proposal](#proposal)
  - [Standard Attribute Definition](#standard-attribute-definition)
  - [Helper Functions](#helper-functions)
  - [Why Not pcieRoot Alone](#why-not-pcieroot-alone)
  - [Current Driver Attribute Names](#current-driver-attribute-names)
  - [User Stories](#user-stories)
    - [Story 1: ML Engineer deploys inference pod](#story-1-ml-engineer-deploys-inference-pod)
    - [Story 2: Platform admin partitions GPU node](#story-2-platform-admin-partitions-gpu-node)
    - [Story 3: KubeVirt VM with correct guest NUMA topology](#story-3-kubevirt-vm-with-correct-guest-numa-topology)
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
  - [Use <code>dra.net/numaNode</code> as the informal convention](#use-dranetnumanode-as-the-informal-convention)
  - [Use pcieRoot-as-list (KEP-5491) instead](#use-pcieroot-as-list-kep-5491-instead)
  - [Standardize <code>cpuSocketID</code> alongside <code>numaNode</code>](#standardize-cpusocketid-alongside-numanode)
- [Infrastructure Needed](#infrastructure-needed)
<!-- /toc -->

## Release Signoff Checklist

Items marked with (R) are required *prior to targeting to a milestone / release*.

- [ ] (R) Enhancement issue in release milestone, which links to KEP dir in [kubernetes/enhancements] (not the initial KEP PR)
- [ ] (R) KEP approvers have approved the KEP status as `implementable`
- [ ] (R) Design details are appropriately documented
- [ ] (R) Test plan is in place, giving consideration to SIG Architecture and SIG Testing input (including test refactors)
  - [ ] e2e Tests for all Beta API Operations (Endpoints)
  - [ ] (R) Ensure GA://github.com/kubernetes/community/blob/master/sig-testing/e2e-tests.md
- [ ] (R) Graduation criteria is in place
- [ ] (R) Production readiness review completed
- [ ] (R) Production readiness review approved
- [ ] "Implementation History" section is up-to-date for milestone
- [ ] User-facing documentation has been created in [kubernetes/website], for publication to [kubernetes.io]
- [ ] Supporting documentation — e.g., additional design documents, links to mailing list discussions/SIG meetings, relevant PRs/issues, release notes

## Summary

Standardize `resource.kubernetes.io/numaNode` as a well-known DRA device attribute alongside the existing `pcieRoot` and `pciBusID`. This attribute identifies which memory controller services a device — an orthogonal signal to `pcieRoot` (which identifies the PCIe switch tree). Adding a shared helper function in the `deviceattribute` library and having DRA drivers publish this attribute under a common name enables cross-driver NUMA co-placement via a single `matchAttribute` constraint.

Today, six DRA drivers publish NUMA node information under five different vendor-specific attribute names. `matchAttribute` requires a common name. Users cannot write a cross-driver NUMA co-placement constraint without middleware. This proposal standardizes the name so that one constraint works across all drivers:

```yaml
constraints:
- matchAttribute: resource.kubernetes.io/numaNode
  requests: [gpu, nic, cpu, mem]
```

## Motivation

### Goals

- Standardize `resource.kubernetes.io/numaNode` (int) as a well-known device attribute in the `deviceattribute` library
- Provide `GetNUMANodeByPCIBusID()` and `GetNUMANodeForCPU()` helper functions for driver authors
- Enable cross-driver NUMA co-placement via `matchAttribute` without requiring middleware or alias tables
- Restore the NUMA coordination capability that was lost when devices moved from device plugins to DRA

### Non-Goals

- Standardize `cpuSocketID` — this is correlated with `numaNode` (coarser grouping), not orthogonal. Can be proposed separately if needed.
- Add `enforcement: preferred` to `matchAttribute` — independently useful but separable. Can be proposed as a separate KEP.
- Change the DRA scheduler's constraint evaluation logic — this KEP only defines a standard attribute name and helper functions.
- Require drivers to publish `numaNode` — drivers MAY publish it. The standardization defines the name and semantics for those that do.

## Proposal

### Standard Attribute Definition

Add `resource.kubernetes.io/numaNode` to the set of well-known device attributes:

| Attribute | Type | Source | Semantics |
|-----------|------|--------|-----------|
| `resource.kubernetes.io/numaNode` | int | `/sys/bus/pci/devices/<BDF>/numa_node` (PCI devices), `/sys/devices/system/node/` (CPU devices), memory controller zone (memory devices) | Which memory controller services this device. Devices with the same `numaNode` value share a memory controller — local DMA, no inter-controller hop. |

This attribute measures **memory topology** — which memory controller is closest to the device. It is orthogonal to `pcieRoot`, which measures **bus topology** — which PCIe switch tree the device sits behind. A GPU and NIC can be connected to different PCIe switches but the same memory controller. Neither attribute is a finer or coarser version of the other; they measure different physical properties of different hardware subsystems.

### Helper Functions

Add to `k8s.io/dynamic-resource-allocation/deviceattribute`:

```go
// GetNUMANodeByPCIBusID returns the NUMA node for a PCI device.
// It reads /sys/bus/pci/devices/<BDF>/numa_node.
func GetNUMANodeByPCIBusID(pciBusID string) (int, error)

// GetNUMANodeForCPU returns the NUMA node for a CPU core.
// It reads /sys/devices/system/node/node<N>/cpulist to find which
// NUMA node contains the given CPU ID.
func GetNUMANodeForCPU(cpuID int) (int, error)
```

### Why Not pcieRoot Alone

`pcieRoot` is the only standardized topology attribute today. It is insufficient for cross-driver NUMA co-placement because:

1. **Many GPUs don't share a PCIe root with any NIC.** On the Dell XE8640 (4x H100), 1 of 4 GPUs shares a switch with a NIC. On the Dell R760xa (2x A40), 0 of 2. `matchAttribute: pcieRoot` excludes most GPUs.

2. **CPUs and memory have no pcieRoot.** They are not PCI devices. Any `pcieRoot` constraint including CPU or memory requests is unsatisfiable.

3. **KEP-5491 list types don't close the gap.** CPU-as-pivot matching works for GPU↔CPU and NIC↔CPU, but GPU and NIC on different PCIe roots have zero intersection — you cannot derive memory proximity from bus addresses.

### Current Driver Attribute Names

Each driver publishes NUMA under a different name. Some maintain manual alias tables for cross-driver compatibility:

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

An ML engineer deploys a vLLM inference pod with 1 GPU, 1 NIC, and CPU cores. They need RDMA between the GPU and NIC to stay within one memory controller — crossing the inter-socket link costs 58% throughput ([Ojea 2025](https://arxiv.org/abs/2506.23628)).

```yaml
constraints:
- matchAttribute: resource.kubernetes.io/numaNode
  requests: [gpu, nic, cpu]
```

One constraint, three drivers, all devices on the same memory controller.

#### Story 2: Platform admin partitions GPU node

A platform admin partitions a multi-GPU node for 4 independent inference pods. Each pod gets 1 GPU + 1 SR-IOV NIC VF + CPUs, all on the same NUMA node. Today this requires per-node CEL selectors with hardcoded vendor attribute names. With standardization, it's a single portable `matchAttribute` constraint in a DeviceClass.

#### Story 3: KubeVirt VM with correct guest NUMA topology

A KubeVirt operator passes GPUs through to a VM via VFIO. The VM's AI framework reads `numa_node` inside the guest to make topology-aware decisions. KubeVirt's virt-launcher reads KEP-5304 device metadata to build guest NUMA topology (VEP 115). Today it must try multiple attribute names. With standardization, one lookup: `dev.Attributes["resource.kubernetes.io/numaNode"]`.

### Risks and Mitigations

**Risk:** SNC (Intel) and NPS (AMD) change what NUMA IDs mean, potentially over-constraining.

**Mitigation:** The sysfs value is always correct — it reports which memory controller services the device. SNC makes `numaNode` finer-grained, not incorrect. GPU servers run SNC/NPS off by default. For the rare SNC-on case, `cpuSocketID` can be proposed separately as a coarser grouping.

**Risk:** Drivers may not adopt the standard attribute quickly.

**Mitigation:** The attribute is additive — drivers publish it alongside existing vendor-specific names. No breaking changes. The helper function makes adoption trivial (~10 lines per driver).

## Design Details

### API Changes

No new API types. The standard attribute name `resource.kubernetes.io/numaNode` is added to the documented set of well-known device attributes in the `deviceattribute` library, alongside `pcieRoot` and `pciBusID`.

### Helper Implementation

```go
// In k8s.io/dynamic-resource-allocation/deviceattribute

const StandardDeviceAttributeNUMANode = StandardDeviceAttributePrefix + "numaNode"

func GetNUMANodeByPCIBusID(pciBusID string) (int, error) {
    path := filepath.Join("/sys/bus/pci/devices", pciBusID, "numa_node")
    data, err := os.ReadFile(path)
    if err != nil {
        return -1, fmt.Errorf("reading NUMA node for %s: %w", pciBusID, err)
    }
    node, err := strconv.Atoi(strings.TrimSpace(string(data)))
    if err != nil {
        return -1, fmt.Errorf("parsing NUMA node for %s: %w", pciBusID, err)
    }
    return node, nil
}

func GetNUMANodeForCPU(cpuID int) (int, error) {
    // Walk /sys/devices/system/node/node*/cpulist to find which
    // NUMA node contains cpuID
    matches, err := filepath.Glob("/sys/devices/system/node/node*/cpulist")
    if err != nil {
        return -1, fmt.Errorf("globbing NUMA nodes: %w", err)
    }
    for _, match := range matches {
        data, err := os.ReadFile(match)
        if err != nil {
            continue
        }
        if cpuInList(cpuID, strings.TrimSpace(string(data))) {
            // Extract node number from path
            dir := filepath.Dir(match)
            nodeName := filepath.Base(dir)
            nodeNum, err := strconv.Atoi(strings.TrimPrefix(nodeName, "node"))
            if err != nil {
                continue
            }
            return nodeNum, nil
        }
    }
    return -1, fmt.Errorf("CPU %d not found in any NUMA node", cpuID)
}
```

### Driver Changes

Each DRA driver that calls `GetPCIeRootAttributeByPCIBusID()` today would add one more call:

```go
// Example: GPU driver adding numaNode
numaNode, err := deviceattribute.GetNUMANodeByPCIBusID(pciBusID)
if err == nil {
    device.Attributes[deviceattribute.StandardDeviceAttributeNUMANode] = resourceapi.DeviceAttribute{
        IntValue: ptr.To(int64(numaNode)),
    }
}
```

### Test Plan

[x] I/we understand the owners of the involved components may require updates to existing tests to make this code solid enough prior to committing the changes necessary to implement this enhancement.

#### Prerequisite testing updates

None — this adds a new attribute and helper functions, does not modify existing behavior.

#### Unit tests

- `deviceattribute` package: tests for `GetNUMANodeByPCIBusID()` and `GetNUMANodeForCPU()` with mock sysfs
- Validation that `resource.kubernetes.io/numaNode` is accepted as a well-known attribute

#### Integration tests

- DRA allocator integration test: two devices with matching `numaNode` values are co-placed by `matchAttribute: resource.kubernetes.io/numaNode`
- DRA allocator integration test: two devices with different `numaNode` values fail `matchAttribute` as expected

#### e2e tests

- End-to-end test with at least two DRA drivers publishing `resource.kubernetes.io/numaNode` and a ResourceClaim with `matchAttribute: resource.kubernetes.io/numaNode` across them

### Graduation Criteria

#### Alpha

- Feature implemented behind `DRAStandardNUMANode` feature gate
- Helper functions in `deviceattribute` library
- Standard attribute name documented
- Unit and integration tests passing
- At least one upstream DRA driver publishes the attribute

#### Beta

- Feature gate enabled by default
- At least three DRA drivers publish the attribute
- Real-world feedback from topology-aware workloads
- No major outstanding bugs

#### GA

- At least two real-world deployments demonstrating cross-driver NUMA co-placement
- Allowing time for feedback from driver authors and workload operators

### Upgrade / Downgrade Strategy

**Upgrade:** Existing claims are unaffected. Drivers can start publishing `resource.kubernetes.io/numaNode` at any time — it's just a new attribute value.

**Downgrade:** If the feature gate is disabled, the attribute name is still valid (it's a standard string in the `resource.kubernetes.io/` namespace). Claims referencing it will still work — the constraint is evaluated as a normal `matchAttribute`. The helper functions are library code and have no runtime feature gate dependency.

### Version Skew Strategy

No version skew concerns. The attribute is a convention (a well-known name), not a new API field. Older schedulers evaluate it as a normal `matchAttribute` constraint. Older drivers don't publish it, so cross-driver matching won't find them — but this is the same behavior as today (no standard name exists).

## Production Readiness Review Questionnaire

### Feature Enablement and Rollback

###### How can this feature be enabled / disabled in a live cluster?

- Feature gate name: `DRAStandardNUMANode`
- Components depending on the feature gate: kube-apiserver (for validation of the well-known attribute name)
- Note: The helper functions are library code used by drivers at build time. The feature gate controls whether the attribute name is validated as a well-known standard attribute.

###### Does enabling the feature change any default behavior?

No. Enabling the feature gate adds `resource.kubernetes.io/numaNode` to the set of well-known device attributes. Existing claims and drivers are unaffected.

###### Can the feature be disabled once it has been enabled (i.e. can we roll back the enablement)?

Yes. Disabling the feature gate removes the attribute from the well-known set. Existing ResourceSlices with the attribute retain it (it's a valid attribute string regardless of feature gate). Claims with `matchAttribute: resource.kubernetes.io/numaNode` continue to work — the constraint is evaluated normally.

###### Are there any tests for feature enablement/disablement?

Tests will verify that the attribute is accepted when the feature gate is enabled and that existing behavior is preserved when disabled.

### Scalability

###### Will enabling / using this feature result in any new API calls?

No. The attribute is published as part of existing ResourceSlice updates. No additional API calls.

###### Will enabling / using this feature result in introducing new API types?

No. The attribute uses existing `DeviceAttribute` types.

###### Will enabling / using this feature result in increasing size or count of the existing API objects?

Minimal. Each device gains one additional attribute entry (~50 bytes). Within the existing 32-attribute-per-device budget.

###### Will enabling / using this feature result in increasing time taken by any operations?

No. The sysfs read is a single file read (~microseconds), performed during driver startup device discovery.

## Implementation History

- 2026-05-12: Initial KEP draft

## Drawbacks

- Adds one more well-known attribute name to the set that driver authors should publish. However, the helper function makes this trivial.
- The SNC/NPS concern means `numaNode` may be too fine-grained on some hardware configurations. However, the sysfs value is always correct for its defined semantics (memory controller zone), and GPU servers run SNC off.

## Alternatives

### Do nothing — use vendor-specific names

Drivers continue publishing NUMA under vendor-specific names. Cross-driver matching requires middleware (topology coordinator, CEL selectors with hardcoded names, manual alias tables in each driver). This is the current state and the reason three drivers already maintain compatibility aliases.

**Rejected because:** The alias tables are growing organically and inconsistently. No single name covers all drivers. The problem gets worse as more drivers are added.

### Use `dra.net/numaNode` as the informal convention

Four of six drivers already publish or alias `dra.net/numaNode`. Make it the convention without formal standardization.

**Rejected because:** GPU drivers (NVIDIA, AMD) don't participate in the `dra.net` convention. The `dra.net/` prefix implies ownership by the dranet project, which is misleading for GPU and CPU devices. A `resource.kubernetes.io/` prefix correctly signals that this is a cross-driver standard.

### Use pcieRoot-as-list (KEP-5491) instead

Have CPUs publish a list of local PCIe roots via KEP-5491 list types, enabling CPU-as-pivot matching.

**Rejected as a replacement (accepted as complementary):** GPU and NIC on different PCIe roots have zero intersection — list types can't bridge bus topology to memory topology. Memory has no pcieRoot. The CPU pcieRoot list encodes the NUMA boundary as a more complex data structure. `numaNode` is the simpler, direct representation of memory controller proximity. Both approaches are complementary — pcieRoot for bus topology, numaNode for memory topology.

### Standardize `cpuSocketID` alongside `numaNode`

Add both attributes in one KEP.

**Rejected for this KEP:** `cpuSocketID` is correlated with `numaNode` (coarser grouping), not orthogonal. Including it increases scope and re-engages the SNC/NPS debate. `numaNode` alone covers the vast majority of GPU deployments (SNC off). `cpuSocketID` can be proposed separately if SNC-on use cases emerge.

## Infrastructure Needed

None.
