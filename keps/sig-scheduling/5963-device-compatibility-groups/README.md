# KEP-5963: DRA Device Compatibility Groups

- [Release Signoff Checklist](#release-signoff-checklist)
- [Summary](#summary)
- [Motivation](#motivation)
  - [Goals](#goals)
  - [Non-Goals](#non-goals)
- [Proposal](#proposal)
  - [User Stories](#user-stories)
    - [Story 1](#story-1)
  - [Notes/Constraints/Caveats](#notesconstraintscaveats)
  - [Risks and Mitigations](#risks-and-mitigations)
- [Design Details](#design-details)
  - [API](#api)
    - [CompatibilityGroups Assignment](#compatibilitygroups-assignment)
  - [Examples](#examples)
    - [Example 1: What the existing API enables](#example-1-what-the-existing-api-enables)
    - [Example 2: How the existing API does not solve the problem](#example-2-how-the-existing-api-does-not-solve-the-problem)
    - [Example 3: How the proposed API solves the problem](#example-3-how-the-proposed-api-solves-the-problem)
    - [Example 4: Multiple compatible groups with an incompatible group](#example-4-multiple-compatible-groups-with-an-incompatible-group)
  - [Scheduler Changes](#scheduler-changes)
  - [Interaction with Multi-Request Claims and Device Constraints](#interaction-with-multi-request-claims-and-device-constraints)
  - [Driver Responsibilities](#driver-responsibilities)
  - [Test Plan](#test-plan)
    - [Prerequisite testing updates](#prerequisite-testing-updates)
    - [Unit tests](#unit-tests)
    - [Integration tests](#integration-tests)
    - [e2e tests](#e2e-tests)
    - [Performance tests](#performance-tests)
  - [Graduation Criteria](#graduation-criteria)
    - [Alpha](#alpha)
    - [Beta](#beta)
    - [GA](#ga)
  - [Upgrade / Downgrade Strategy](#upgrade--downgrade-strategy)
    - [Upgrade](#upgrade)
    - [Downgrade](#downgrade)
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
  - [Current Workaround: Driver-level Preparation Failure](#current-workaround-driver-level-preparation-failure)
  - [Inverted naming: `mutualExclusionGroups`](#inverted-naming-mutualexclusiongroups)
- [Infrastructure Needed (Optional)](#infrastructure-needed-optional)

## Release Signoff Checklist

Items marked with (R) are required *prior to targeting to a milestone / release*.

- (R) Enhancement issue in release milestone, which links to KEP dir in [kubernetes/enhancements](https://git.k8s.io/enhancements) (not the initial KEP PR)
- (R) KEP approvers have approved the KEP status as `implementable`
- (R) Design details are appropriately documented
- (R) Test plan is in place, giving consideration to SIG Architecture and SIG Testing input (including test refactors)
  - e2e Tests for all Beta API Operations (endpoints)
  - (R) Ensure GA e2e tests meet requirements for [Conformance Tests](https://github.com/kubernetes/community/blob/master/contributors/devel/sig-architecture/conformance-tests.md)
  - (R) Minimum Two Week Window for GA e2e tests to prove flake free
- (R) Graduation criteria is in place
  - (R) [all GA Endpoints](https://github.com/kubernetes/community/pull/1806) must be hit by [Conformance Tests](https://github.com/kubernetes/community/blob/master/contributors/devel/sig-architecture/conformance-tests.md) within one minor version of promotion to GA
- (R) Production readiness review completed
- (R) Production readiness review approved
- "Implementation History" section is up-to-date for milestone
- User-facing documentation has been created in [kubernetes/website](https://git.k8s.io/website), for publication to [kubernetes.io](https://kubernetes.io/)
- Supporting documentation—e.g., additional design documents, links to mailing list discussions/SIG meetings, relevant PRs/issues, release notes

## Summary

This KEP proposes an extension to the Dynamic Resource Allocation (DRA) ResourceSlice API to
support mutually exclusive device allocation constraints between sets of devices. Hardware devices often
support multiple partitioning or virtualization schemes (for example, GPU MIG
slicing vs. vGPU profiles) that provide different trade-offs in terms of isolation,
performance, and resource sharing. These schemes are frequently mutually exclusive
at the hardware level: once a physical device is partitioned or configured using
one scheme, it cannot be reconfigured to use a different scheme until all existing
allocations are released.

The current DRA Partitionable Devices API has no mechanism for drivers to express
these mutual exclusivity constraints. A shared counter with a capacity of one
can ensure mutual exclusion, but this cannot be used here: such a counter
would have to be decremented once when allocating the *first* device from a set
of compatible devices, not once for *each* device. This cannot be expressed
at the moment.

## Motivation

Without a mechanism to express these constraints in DRA, the following problems
arise:

1. **Late Failure Detection**: Incompatible allocations are only detected during
  resource preparation, after scheduling decisions have already been made.
2. **Scheduler Unawareness**: The scheduler may allocate incompatible devices,
  leading to pod startup failures.
3. **Poor User Experience**: Users receive cryptic preparation failures instead
  of clear scheduling feedback.

The current workaround—having DRA drivers fail resource preparation when
incompatible allocations are attempted—is insufficient because does not prevent
repeated failed attempts when a replacement pod gets created for a failed one.

### Goals

- Allow DRA drivers to specify compatibility between virtual devices within a
single physical device.
- Allow the scheduler to make informed allocation decisions that respect
compatibility rules declared in ResourceSlice objects.
- Provide a generic mechanism applicable to any hardware with partitioning
constraints, not just GPUs.

### Non-Goals

- Providing a centralized or cluster-wide registry of compatibility group
  names. Group names are opaque strings scoped to a single resource pool
  and are meaningful only to the driver that publishes them.
- Enabling the scheduler to *reconfigure* a physical device between
  partitioning schemes (e.g., MIG ↔ vGPU) as part of scheduling. This KEP only
  addresses rejecting incompatible allocations; transitions between schemes
  remain a driver concern and typically require draining existing allocations.
- Expressing compatibility constraints on `ResourceClaim` objects. The field
  is driver-authored and lives only on `ResourceSlice`.
- Replacing existing counter-capacity checks. `compatibilityGroups` is an
  additional predicate; capacity math on `sharedCounters` continues to apply
  unchanged.

## Proposal

This KEP extends the DRA `ResourceSlice` API with a driver-authored
`compatibilityGroups` field on each `device.consumesCounters[]` entry,
and `DeviceRequestAllocationResult` (in `ResourceClaim.status.allocation`) with a
matching snapshot of the same groups. Drivers tag each virtual device
with the named groups it agrees to be allocated alongside; the
scheduler enforces, at candidate-filtering time, that all devices
co-allocated on the same counter set share at least one group. This
moves detection of incompatible co-allocation from preparation-time
failure to scheduling-time rejection.

### User Stories

#### Story 1

As a DRA driver for NVIDIA GPUs, I want to express in my ResourceSlice
that MIG-partitioned devices and vGPU-partitioned devices are incompatible on the
same physical GPU at the same time. When a pod requesting a MIG partition is already
running on a GPU, I want the scheduler to exclude all vGPU-partitioned devices on 
that same GPU from consideration for new allocations, rather than allowing an 
allocation that will fail at device preparation time.

### Notes/Constraints/Caveats

This feature depends on the `DRAPartitionableDevices` feature gate being
enabled. `compatibilityGroups` is declared on
`device.consumesCounters[]` entries, which only exist when partitionable
devices are in use; a cluster without `DRAPartitionableDevices` has
nothing to attach the field to.

### Risks and Mitigations

**Scheduler performance impact**

Evaluating compatibility constraints during device selection adds work to each 
scheduling cycle that involves DRA devices.

**Gate-disabled schedulers ignoring devices with compatibilityGroups**

At alpha, if the `DRADeviceCompatibilityGroups` feature-gate is disabled, 
devices which present the `compatibilityGroups` field will be ignored by `kube-scheduler`.
This is in order to allow enablement of the feature without user intervention (pod deletion) 
when graduating to beta. 

**Drivers must be aware of the enablement status of the `DRADeviceCompatibilityGroups` flag**

The feature flag can be enabled/disabled at runtime. This requires drivers to identify
the flag status and update the `ResourceSlice`s they manage accordingly.

In general, admins should avoid deploying DRA drivers with features enabled 
that aren't also enabled in the cluster.

## Design Details

### API

#### CompatibilityGroups Assignment

The slice-side `compatibilityGroups` field has type `[]string` and lives on
each `device.consumesCounters[]` entry for drivers to populate. For two 
devices consuming counters from the same counter set to be allocated together,
either both must leave the field unset, or both must declare it and share at 
least one group name. A nil `compatibilityGroups` and an empty `compatibilityGroups: []` 
are treated identically. This means a device that declares the field is never
allocated on a shared counter at the same time with a sibling that omits it.

The field is placed on each `consumesCounters[]` entry rather than on the
device itself because compatibility is a physical-hardware property scoped to
the shared resource represented by the counter set. A single virtual device
that consumes from multiple counter sets may therefore declare different
groups per counter set, reflecting different exclusivity constraints on
different pieces of underlying hardware. Two devices that do not share any
counter set are never compared via this field, even if they live on the same
node or in the same `ResourceSlice`.

```go
// ResourceSlice API
type DeviceCounterConsumption struct {
    CounterSet string             `json:"counterSet" protobuf:"bytes,1,opt,name=counterSet"`
    Counters   map[string]Counter `json:"counters,omitempty" protobuf:"bytes,2,opt,name=counters"`

    // CompatibilityGroups is declared by drivers on the ResourceSlice.
    // +optional
    // +listType=atomic
    // +featureGate=DRADeviceCompatibilityGroups
    CompatibilityGroups []string `json:"compatibilityGroups,omitempty" protobuf:"bytes,3,rep,name=compatibilityGroups"`
}
```

To enforce compatibility at scheduling time, the scheduler needs the
`compatibilityGroups` of every already-allocated device on a counter set.
Reading them back from the source `ResourceSlice` is unsafe: the slice may
have been updated since allocation, or the device may have been
re-published with different groups. The scheduler therefore records a
snapshot of each allocated device's `compatibilityGroups` on its claim
status entry, mirroring how `ConsumedCapacity` is recorded for the
consumable-capacity feature.

The snapshot on `DeviceRequestAllocationResult` has type
`map[string][]string`, keyed by counter-set name (matching
`consumesCounters[*].counterSet`). Each value is the list of groups
declared on the allocated device's `consumesCounters[]` entry for that
counter set at the time of allocation. Counter sets the device does not
consume from are omitted from the map.

```go
// ResourceClaim API
type DeviceRequestAllocationResult struct {
    // ... existing fields ...

    // CompatibilityGroups is written by the scheduler at allocation time
    // and is a per-counter-set snapshot of the allocated device's
    // declared groups. It is consulted on subsequent allocations against
    // the same counter set.
    //
    // +optional
    // +featureGate=DRADeviceCompatibilityGroups
    CompatibilityGroups map[string][]string `json:"compatibilityGroups,omitempty" protobuf:"bytes,11,rep,name=compatibilityGroups"`
}
```

`DeviceRequestAllocationResult` population and lifecycle:

- The scheduler populates this field as part of writing
  `ResourceClaim.status.allocation`. Drivers do not write it.
- It is present only when the allocated device's slice entry declares at
  least one `compatibilityGroup` on any counter set; otherwise
  the field is omitted from the status.

### Examples

**Naming convention used in examples.** A device's `compatibilityGroups`
lists the groups it agrees to be allocated alongside other devices in.
Group names in the examples are chosen for readability (e.g., `mig`,
`vgpu`, `foobar`) and hint at which devices agree to be in the group;
the scheduler does not parse them, so any opaque strings will do as
long as compatible devices declare a common group.

The following examples demonstrate the problem and the proposed solution using
a GPU that supports two mutually exclusive partitioning schemes: MIG (hardware
partitioning into isolated instances) and vGPU (virtualized GPU profiles).
All examples share the following `DeviceClass`:

```yaml
apiVersion: resource.k8s.io/v1
kind: DeviceClass
metadata:
  name: gpu.example.com
spec:
  selectors:
    - cel:
        expression: device.driver == 'gpu.example.com'
```

#### Example 1: What the existing API enables

The DRA Partitionable Devices API uses shared counter sets to track the
capacity of a physical device across multiple dimensions. When all virtual
devices on a GPU use the same partitioning scheme, the counter capacity check
is sufficient to ensure correct allocation.

ResourceSlices — a single GPU advertising three MIG 1g partitions:

```yaml
apiVersion: resource.k8s.io/v1
kind: ResourceSlice
metadata:
  name: node-1-gpu-0-counters
spec:
  driver: gpu.example.com
  pool:
    name: node-1-pool
    generation: 1
    resourceSliceCount: 2
  sharedCounters:
    - name: gpu-0-counters
      counters:
        multiprocessors:
          value: "100"
---
apiVersion: resource.k8s.io/v1
kind: ResourceSlice
metadata:
  name: node-1-gpu-0-devices
spec:
  driver: gpu.example.com
  pool:
    name: node-1-pool
    generation: 1
    resourceSliceCount: 2
  nodeName: node-1
  devices:
    - name: gpu-0-mig-1g-0
      attributes:
        type:
          string: "mig-1g"
      consumesCounters:
        - counterSet: gpu-0-counters
          counters:
            multiprocessors:
              value: "20"
    - name: gpu-0-mig-1g-1
      attributes:
        type:
          string: "mig-1g"
      consumesCounters:
        - counterSet: gpu-0-counters
          counters:
            multiprocessors:
              value: "20"
    - name: gpu-0-mig-1g-2
      attributes:
        type:
          string: "mig-1g"
      consumesCounters:
        - counterSet: gpu-0-counters
          counters:
            multiprocessors:
              value: "20"
```

ResourceClaims — two pods each requesting a MIG 1g partition:

```yaml
apiVersion: resource.k8s.io/v1
kind: ResourceClaim
metadata:
  name: pod-a-gpu
  namespace: default
spec:
  devices:
    requests:
      - name: gpu
        deviceClassName: gpu.example.com
        selectors:
          - cel:
              expression: device.attributes['type'].string == 'mig-1g'
---
apiVersion: resource.k8s.io/v1
kind: ResourceClaim
metadata:
  name: pod-b-gpu
  namespace: default
spec:
  devices:
    requests:
      - name: gpu
        deviceClassName: gpu.example.com
        selectors:
          - cel:
              expression: device.attributes['type'].string == 'mig-1g'
```

The scheduler allocates `gpu-0-mig-1g-0` to pod-a and `gpu-0-mig-1g-1` to
pod-b, and binds both to `node-1` because their consumptions from the `gpu-0-counters` 
counter set are valid (20 + 20 = 40 <= 100). Both pods start successfully because the 
devices allocated to them are compatible (Both MIG slices).

#### Example 2: How the existing API does not solve the problem

When a driver advertises devices from multiple mutually exclusive partitioning
schemes on the same GPU, all sharing the same counter set, the current API has
no way to express that these schemes cannot coexist.

ResourceSlices — the same GPU now advertising both MIG and vGPU devices:

```yaml
apiVersion: resource.k8s.io/v1
kind: ResourceSlice
metadata:
  name: node-1-gpu-0-counters
spec:
  driver: gpu.example.com
  pool:
    name: node-1-pool
    generation: 1
    resourceSliceCount: 2
  sharedCounters:
    - name: gpu-0-counters
      counters:
        multiprocessors:
          value: "100"
---
apiVersion: resource.k8s.io/v1
kind: ResourceSlice
metadata:
  name: node-1-gpu-0-devices
spec:
  driver: gpu.example.com
  pool:
    name: node-1-pool
    generation: 1
    resourceSliceCount: 2
  nodeName: node-1
  devices:
    # MIG partitions
    - name: gpu-0-mig-1g-0
      attributes:
        type:
          string: "mig-1g"
      consumesCounters:
        - counterSet: gpu-0-counters
          counters:
            multiprocessors:
              value: "20"
    - name: gpu-0-mig-1g-1
      attributes:
        type:
          string: "mig-1g"
      consumesCounters:
        - counterSet: gpu-0-counters
          counters:
            multiprocessors:
              value: "20"
    # vGPU profiles
    - name: gpu-0-vgpu-0
      attributes:
        type:
          string: "vgpu"
      consumesCounters:
        - counterSet: gpu-0-counters
          counters:
            multiprocessors:
              value: "50"
    - name: gpu-0-vgpu-1
      attributes:
        type:
          string: "vgpu"
      consumesCounters:
        - counterSet: gpu-0-counters
          counters:
            multiprocessors:
              value: "50"
```

ResourceClaims — pod-a requests a MIG partition, pod-b requests a vGPU profile:

```yaml
apiVersion: resource.k8s.io/v1
kind: ResourceClaim
metadata:
  name: pod-a-gpu
  namespace: default
spec:
  devices:
    requests:
      - name: gpu
        deviceClassName: gpu.example.com
        selectors:
          - cel:
              expression: device.attributes['type'].string == 'mig-1g'
---
apiVersion: resource.k8s.io/v1
kind: ResourceClaim
metadata:
  name: pod-b-gpu
  namespace: default
spec:
  devices:
    requests:
      - name: gpu
        deviceClassName: gpu.example.com
        selectors:
          - cel:
              expression: device.attributes['type'].string == 'vgpu'
```

The scheduler sees `gpu-0-mig-1g-0` (20 SMs) and `gpu-0-vgpu-0` (50 SMs).
Total: 70 <= 100 — the counter capacity check passes. The scheduler allocates
both. But at preparation time, the driver fails because MIG and vGPU cannot be
active simultaneously on the same physical GPU. Pod-b gets a cryptic
preparation error.

#### Example 3: How the proposed API solves the problem

With `compatibilityGroups`, the driver declares that MIG devices belong to the
`mig` group and vGPU devices belong to the `vgpu` group. The scheduler
enforces that devices sharing a counter set must share at least one
compatibility group.

ResourceSlices — same devices, now with compatibility groups:

```yaml
apiVersion: resource.k8s.io/v1
kind: ResourceSlice
metadata:
  name: node-1-gpu-0-counters
spec:
  driver: gpu.example.com
  pool:
    name: node-1-pool
    generation: 1
    resourceSliceCount: 2
  sharedCounters:
    - name: gpu-0-counters
      counters:
        multiprocessors:
          value: "100"
---
apiVersion: resource.k8s.io/v1
kind: ResourceSlice
metadata:
  name: node-1-gpu-0-devices
spec:
  driver: gpu.example.com
  pool:
    name: node-1-pool
    generation: 1
    resourceSliceCount: 2
  nodeName: node-1
  devices:
    # MIG partitions
    - name: gpu-0-mig-1g-0
      attributes:
        type:
          string: "mig-1g"
      consumesCounters:
        - counterSet: gpu-0-counters
          compatibilityGroups:
            - mig
          counters:
            multiprocessors:
              value: "20"
    - name: gpu-0-mig-1g-1
      attributes:
        type:
          string: "mig-1g"
      consumesCounters:
        - counterSet: gpu-0-counters
          compatibilityGroups:
            - mig
          counters:
            multiprocessors:
              value: "20"
    # vGPU profiles
    - name: gpu-0-vgpu-0
      attributes:
        type:
          string: "vgpu"
      consumesCounters:
        - counterSet: gpu-0-counters
          compatibilityGroups:
            - vgpu
          counters:
            multiprocessors:
              value: "50"
    - name: gpu-0-vgpu-1
      attributes:
        type:
          string: "vgpu"
      consumesCounters:
        - counterSet: gpu-0-counters
          compatibilityGroups:
            - vgpu
          counters:
            multiprocessors:
              value: "50"
```

ResourceClaims — identical to Example 2:

```yaml
apiVersion: resource.k8s.io/v1
kind: ResourceClaim
metadata:
  name: pod-a-gpu
  namespace: default
spec:
  devices:
    requests:
      - name: gpu
        deviceClassName: gpu.example.com
        selectors:
          - cel:
              expression: device.attributes['type'].string == 'mig-1g'
---
apiVersion: resource.k8s.io/v1
kind: ResourceClaim
metadata:
  name: pod-b-gpu
  namespace: default
spec:
  devices:
    requests:
      - name: gpu
        deviceClassName: gpu.example.com
        selectors:
          - cel:
              expression: device.attributes['type'].string == 'vgpu'
```

The scheduler allocates `gpu-0-mig-1g-0` (group: `mig`) to pod-a. When
evaluating `gpu-0-vgpu-0` (group: `vgpu`) for pod-b, it checks
compatibility: both devices consume from `gpu-0-counters`, but they share no
compatibility group (`mig` vs `vgpu`). The scheduler rejects the allocation and
pod-b becomes Unschedulable with generic event: "could not allocate all claims".
No cryptic preparation failure post-scheduling, no resource thrashing.

Two MIG devices (both group: `mig`) or two vGPU devices (both group: `vgpu`) can
still be allocated at the same time, consuming from the same counter set, since they share a group.

#### Example 4: Multiple compatible groups with an incompatible group

A device may be compatible with multiple groups.
In this example, devices advertise compatibility with multiple groups

ResourceSlices — devices advertising compatibility with the `foo`, `bar`, and `baz` groups:

```yaml
apiVersion: resource.k8s.io/v1
kind: ResourceSlice
metadata:
  name: node-1-device-0-counters
spec:
  driver: device.example.com
  pool:
    name: node-1-pool
    generation: 1
    resourceSliceCount: 2
  sharedCounters:
    - name: device-0-counters
      counters:
        units:
          value: "100"
---
apiVersion: resource.k8s.io/v1
kind: ResourceSlice
metadata:
  name: node-1-device-0-devices
spec:
  driver: device.example.com
  pool:
    name: node-1-pool
    generation: 1
    resourceSliceCount: 2
  nodeName: node-1
  devices:
    # foo partitions
    - name: device-0-foo-0
      attributes:
        type:
          string: "foo"
      consumesCounters:
        - counterSet: device-0-counters
          compatibilityGroups:
            - foo
            - foobar
          counters:
            units:
              value: "25"
    # bar partitions
    - name: device-0-bar-0
      attributes:
        type:
          string: "bar"
      consumesCounters:
        - counterSet: device-0-counters
          compatibilityGroups:
            - bar
            - foobar
          counters:
            units:
              value: "25"
    # baz partitions
    - name: device-0-baz-0
      attributes:
        type:
          string: "baz"
      consumesCounters:
        - counterSet: device-0-counters
          compatibilityGroups:
            - baz
          counters:
            units:
              value: "50"
```

`device-0-foo-0` (groups: `foo`, `foobar`) and `device-0-bar-0` (groups:
`bar`, `foobar`) share the `foobar` group, so they can be allocated together.
`device-0-baz-0` (groups: `baz`) as no common group with both, so it cannot be
allocated with them.

For instance, if pod-a is allocated `device-0-foo-0`, a subsequent pod
requesting `device-0-bar-0` succeeds (both share `foobar`), but a pod
requesting `device-0-baz-0` is rejected (`foo`/`foobar` vs `baz` — no shared
group).

### Scheduler Changes

The DRA scheduler plugin enforces `compatibilityGroups` during candidate
filtering. For each counter set `c` involved in an allocation decision,
the scheduler calculates a **rolling intersection** `I_c` of
`compatibilityGroups` across the devices currently allocated on `c`,
sourced from the `DeviceRequestAllocationResult.CompatibilityGroups`
snapshot on each existing `ResourceClaim.status`.

A candidate device `k` consuming from counter set `c` is admitted iff
`I_c ∩ cg(k, c) ≠ ∅`, where `cg(k, c)` is the candidate's declared
`compatibilityGroups` for counter set `c`. When `k` is admitted, the
rolling intersection advances: `I_c ← I_c ∩ cg(k, c)`. A candidate
consuming from multiple counter sets must be admitted on each one
independently.

### Interaction with Multi-Request Claims and Device Constraints

**Multiple requests within one claim.** The compatibility predicate is evaluated 
between each candidate and the intersection of `compatibilityGroups` across 
all devices already allocated on the same counter set, regardless of whether 
an allocated device belongs to the same claim, a different claim on the same pod,
or a different pod entirely. Two candidate devices for a single `ResourceClaim` 
that land on the same counter set are therefore subject to the same check: 
the second request sees the first folded into the rolling intersection.

**Allocation order.** The scheduler does not reorder requests within a claim
to improve feasibility. If requests are ordered such that an early compatible
pick later blocks a mandatory pick, the claim becomes Unschedulable and
standard retry behavior applies in the next scheduling cycle.
This matches how existing DRA constraints behave.

**Composition with `DeviceConstraints`.** `compatibilityGroups` is a
driver-authored, ResourceSlice-side constraint. `DeviceConstraints` (e.g.,
`matchAttribute`) is a user-authored, ResourceClaim-side constraint. The two
are evaluated independently and both must pass for a candidate to be
allocated. A claim can never *relax* a driver-declared compatibility group,
and a driver can never *force* a claim-side `matchAttribute`. They compose by
conjunction.

### Driver Responsibilities

Resource drivers are responsible for:

1. Populating `compatibilityGroups` for all devices with compatibility
   requirements.
2. Detecting the `DRADeviceCompatibilityGroups` feature gate status and,
   when the gate is disabled, omitting `compatibilityGroups` from
   `ResourceSlice` entries. Devices declaring the field on
   a gate-off cluster will be skipped by the scheduler and remain
   unschedulable until either the gate is enabled or the field is removed.
3. Continuing to validate allocations at resource preparation time for
   version-skew safety and to detect incorrect allocations made by a
   scheduler that does not (or no longer) enforces compatibility.

### Test Plan

[X] I/we understand the owners of the involved components may require updates to  
existing tests to make this code solid enough prior to committing the changes necessary  
to implement this enhancement.

##### Prerequisite testing updates

None. The DRA scheduler plugin, `ResourceSlice` and `ResourceClaim` validation already have
unit and integration coverage; new tests are additive.

##### Unit tests

- `k8s.io/dynamic-resource-allocation/structured`: group-intersection
  predicate against the rolling intersection of the allocated set (empty,
  nil, single, multiple groups; nil-vs-nil, nil-vs-set, set-vs-set; `[]`
  treated as nil).
- `k8s.io/kubernetes/pkg/scheduler/framework/plugins/dynamicresources`: filter
  behavior with mixed compatible and incompatible candidates on the same
  counter set; when the feature gate is disabled, ignore devices which declare groups, but 
  respect snapshotted counter set groups from claims.
- `k8s.io/kubernetes/pkg/apis/resource/validation`: field validation —
  accepted shapes, max group-name length, max groups per counter consumption.

##### Integration tests

- Feature gate enablement/disablement round-trip: field is persisted when
  enabled, dropped on write when disabled (see https://github.com/kubernetes/kubernetes/blob/1f77090cd12d05c462e2e180b4f8becc12735728/test/integration/dra/core.go#L161).
- Scheduler correctly records `compatibilityGroups` in claim status after binding it.
- Scheduler rejects a claim when the only remaining candidate on a node
  belongs to an incompatible group; admits it when a compatible candidate
  exists on another node.
- Upgrade → downgrade → upgrade: allocations made during the "upgrade" phase
  remain valid after downgrade; re-enabling enforcement does not re-evaluate
  existing allocations (see https://github.com/kubernetes/kubernetes/blob/1f77090cd12d05c462e2e180b4f8becc12735728/test/e2e_dra/upgradedowngrade_test.go#L234-L287).
- Snapshot is authoritative over a mutated slice: after a device is allocated,
  update the originating `ResourceSlice` to change (or remove) that device's
  `compatibilityGroups`, then schedule a second pod against the same counter
  set. The scheduler must evaluate the second candidate against the
  `compatibilityGroups` snapshot recorded on the existing claim status, not
  the mutated slice, producing the same decision as if the slice were
  unchanged.

##### e2e tests

- Fake DRA driver advertising two mutually exclusive groups (`mig`, `vgpu`) on
  a single counter set. Scheduling a `mig` pod followed by a `vgpu` pod on
  the same node leaves the second pod Unschedulable; reversing the order reproduces the behavior symmetrically.
- Same driver with devices who are compatible with each other (declare a shared group) — both pods schedule.

##### Performance tests

A scheduler_perf workload under
[`test/integration/scheduler_perf/dra`](https://github.com/kubernetes/kubernetes/tree/master/test/integration/scheduler_perf/dra)
will measure the scheduling-throughput impact of the compatibility check.

### Graduation Criteria

#### Alpha

- API defined and implemented
- All relevant code is merged and placed behind a feature flag
- Unit, integration and E2E tests implemented and passing reliably
- Driver-author documentation published under `kubernetes/website` (DRA
  drivers section), including the strict nil-matching rule and a worked
  MIG/vGPU example.

#### Beta

- Validated with at least one production DRA driver (out-of-tree testing)

#### GA

- At least 1 release as beta

### Upgrade / Downgrade Strategy

#### Upgrade

Upon upgrading, no `ResourceSlice` leverages the new optional fields 
yet because DRA drivers should be updated after the cluster upgrade 
is complete, so the current behavior remains as-is.
In the unlikely case of a DRA driver trying to use the feature while 
it's still being rolled out (enabled in apiserver, disabled in scheduler), 
the scheduler >= 1.37 will ignore the devices instead of doing incorrect
allocations. They will get used as soon as the feature gets enabled also in the scheduler.

#### Downgrade

If downgrading to a version that does not have this enhancement implemented, 
older schedulers and api-servers do not know of the added optional fields, 
and revert to their defined behavior prior to this enhancement when the current 
version is the initial alpha. When downgrading to the alpha release in 1.37, 
the scheduler will refuse both to allocating devices which depend on the feature, 
and allocating devices to counter sets which previously had allocations with 
`compatibilityGroups` declared. Eventually, the downgraded DRA driver will remove 
those devices from `ResourceSlices`, however, existing `ResourceClaim`s 
with `compatibilityGroups` declared in their status must be deleted manually.

Allocated devices that leveraged this new field will remain allocated, and future
allocations will not take `compatibilityGroups` into consideration.

### Version Skew Strategy

The feature introduces two new optional fields in `ResourceSlice`, `ResourceClaim.Status`, and new
enforcement logic in the scheduler. Both `kube-apiserver` and
`kube-scheduler` can be running an old version (which doesn't know the
fields), a new version with the feature gate disabled, or a new version
with the feature gate enabled. The table below summarises the behaviour
for every combination on a single cluster.

| kube-apiserver ↓ \ kube-scheduler → | old | new, gate off | new, gate on |
|---|---|---|---|
| **old**           | Pre-KEP      | Pre-KEP          | Pre-KEP          |
| **new, gate off** | Pre-KEP      | Pre-KEP          | Pre-KEP          |
| **new, gate on**  | Driver-only  | Devices skipped  | **Full feature** |

- **Pre-KEP.** The apiserver does not serve the `compatibilityGroups` fields
  on either `ResourceSlice`s or `ResourceClaim`s (it
  either doesn't know them, or has the gate off, in which case it
  strips on writes). The scheduler sees no
  constraints and allocates as before this KEP. Drivers reject
  incompatible allocations at preparation time.
- **Driver-only.** The apiserver persists and serves
  both `compatibilityGroups` fields, but an old scheduler doesn't recognise them
  and allocates without considering it. Pods may be scheduled
  with incompatible devices; the DRA driver rejects them at preparation
  time.
- **Devices skipped.** The apiserver serves both `compatibilityGroups` fields, and
  a new scheduler with the gate off recognises them but is not
  permitted to enforce them. To avoid allocations it cannot validate, the
  scheduler excludes any device that declares `compatibilityGroups`
  from consideration, and also prevents consumption of `counterSet`s that have 
  `compatibilityGroups` assigned to them (known from claim statuses).
  Devices that do not declare the field are scheduled normally.
- **Full feature.** The scheduler filters incompatible candidates
  during allocation. Drivers continue to validate at preparation time
  for defense in depth.

**Downgrade with in-flight allocations.** Devices already allocated
under the new rules remain allocated across a downgrade; the
post-downgrade scheduler will not consider `compatibilityGroups` for
future allocations, reverting to pre-KEP behavior. No existing
allocations are invalidated.

## Production Readiness Review Questionnaire

### Feature Enablement and Rollback

###### How can this feature be enabled / disabled in a live cluster?

- Feature gate
  - Feature gate name: DRADeviceCompatibilityGroups
  - Components depending on the feature gate: kube-scheduler, kube-apiserver
- Gate behavior per component:
  - **kube-apiserver**: When enabled, persists `compatibilityGroups` to devices in `ResourceSlice`s and `ResourceClaim.Status`es.
    When disabled, strips `compatibilityGroups` from both on writes.
  - **kube-scheduler**: When enabled, respects `compatibilityGroups` declared by devices in `ResourceSlice`s
    and maintains and respects `compatibilityGroups` in `ResourceClaim.Status`es.
    When disabled, skips devices that declare `compatibilityGroups`, but respects `compatibilityGroups` in `ResourceClaim.Status`es.
- Partial control-plane downtime is required to toggle the gate - `kube-apiserver` and `kube-scheduler` need to restart.
- No node downtime or reprovisioning is required.

###### Does enabling the feature change any default behavior?

No, this KEP proposes additional optional fields to the `ResourceSlice` and `ResourceClaim` APIs

###### Can the feature be disabled once it has been enabled (i.e. can we roll back the enablement)?

Yes, rolling back the enablement will revert the cluster to its pre-enablement behavior

###### What happens if we reenable the feature if it was previously rolled back?

Existing `compatibilityGroups` configurations in `ResourceSlice`s will become effective again

###### Are there any tests for feature enablement/disablement?

Yes, there will be integration tests to verify feature enablement/disablement

### Rollout, Upgrade and Rollback Planning

###### How can a rollout or rollback fail? Can it impact already running workloads?

Rollout risk is limited to the two components touched by the feature gate
(kube-apiserver field handling and kube-scheduler filter logic).
Already-running workloads are not affected: compatibility filtering only runs
during scheduling of *new* allocations, so disabling the gate or rolling back
binaries does not disturb existing pod/device bindings.

###### What specific metrics should inform a rollback?

This KEP does not include new metrics.
An increase in scheduling failures for workloads requesting DRA devices is the
metric cluster-operators should watch for, along with the scheduler latency
histograms listed in [*What are the SLIs an operator can use to determine the
health of the service?*](#what-are-the-slis-service-level-indicators-an-operator-can-use-to-determine-the-health-of-the-service)

###### Were upgrade and rollback tested? Was the upgrade->downgrade->upgrade path tested?

Upgrade → downgrade → upgrade will be covered by the integration test
described in Test Plan → Integration tests.

###### Is the rollout accompanied by any deprecations and/or removals of features, APIs, fields of API types, flags, etc.?

No

### Monitoring Requirements

###### How can an operator determine if the feature is in use by workloads?

This feature is not intended for use by workloads, it is intended for DRA Drivers.
Workloads use it indirectly when they allocate devices which use the feature, which is visible in the allocation result.

###### How can someone using this feature know that it is working for their instance?

After enabling the feature, and upgrading DRA drivers to versions that utilize it, cluster-operators
should no longer see `FailedPrepareDynamicResources` on container startups.

###### What are the reasonable SLOs (Service Level Objectives) for the enhancement?

This enhancement introduces no new SLOs and is not expected to change any
existing SLOs. The added compatibility check runs within the scheduler's
existing Filter phase and is expected to stay within the current pod
scheduling latency objectives.

###### What are the SLIs (Service Level Indicators) an operator can use to determine the health of the service?

The scheduler's existing latency histograms, scoped to the DRA plugin:

- `scheduler_plugin_execution_duration_seconds{plugin="DynamicResources", extension_point="Filter"}`
- `scheduler_framework_extension_point_duration_seconds{extension_point="Filter"}`

A rise in these after enabling the feature indicates the compatibility
check is adding measurable scheduling latency.

###### Are there any missing metrics that would be useful to have to improve observability of this feature?

N/A

### Dependencies

DRA Partitionable Devices enabled

###### Does this feature depend on any specific services running in the cluster?

No

### Scalability

###### Will enabling / using this feature result in any new API calls?

No

###### Will enabling / using this feature result in introducing new API types?

No

###### Will enabling / using this feature result in any new calls to the cloud provider?

No

###### Will enabling / using this feature result in increasing size or count of the existing API objects?

Yes, 2 additional fields to the `ResourceSlice` and `ResourceClaim` APIs

###### Will enabling / using this feature result in increasing time taken by any operations covered by existing SLIs/SLOs?

Yes — scheduling cycles involving DRA devices incur an additional per-counter-set intersection check,
which IMO is negligible compared to existing DRA implementation.

###### Will enabling / using this feature result in non-negligible increase of resource usage (CPU, RAM, disk, IO, ...) in any components?

No

###### Can enabling / using this feature result in resource exhaustion of some node resources (PIDs, sockets, inodes, etc.)?

No

### Troubleshooting

###### How does this feature react if the API server and/or etcd is unavailable?

No new side effects

###### What are other known failure modes?

**Driver publishes inconsistent `compatibilityGroups` across `ResourceSlice`
updates.** A driver may, intentionally or by bug, change the
`compatibilityGroups` declared on a device between slice generations —
either flipping a device into or out of a group, or rewriting the group
list entirely. Already-allocated devices are unaffected because the
scheduler reads their groups from the `DeviceRequestAllocationResult`
snapshot recorded at allocation time, not from the (possibly mutated)
slice. New allocations on the same counter set, however, evaluate against
the *current* slice for candidates and the *snapshotted* groups for
already-allocated peers. Drivers that need consistent enforcement across
slice rewrites must avoid changing the group declarations of devices in
use.

If an inconsistency does land — the driver still rejects at preparation time
(the existing version-skew safety net), so the failure mode degrades to
the pre-KEP behavior for that one pod rather than corrupting cluster
state.

###### What steps should be taken if SLOs are not being met to determine the problem?

See [*What specific metrics should inform a rollback?*](#what-specific-metrics-should-inform-a-rollback). 

## Implementation History

- 1.37 - initial alpha implementation


## Drawbacks

Adding compatibility constraint support to the scheduler increases the  
complexity of the DRA scheduling logic.

## Alternatives

### Current Workaround: Driver-level Preparation Failure

The existing workaround is for DRA drivers to fail resource preparation when
incompatible allocations are attempted. This approach is insufficient because:

- It detects incompatibilities only after scheduling has committed to the
allocation, leading to pod startup failures.
- It provides no mechanism to inform the scheduler so it can try other nodes
or device combinations.
- It results in resource thrashing as the scheduler retries the same failing
combination.

### Inverted naming: `mutualExclusionGroups`

An alternative API would invert the semantics: instead of declaring which
groups a device *belongs to* (co-allocation predicate), declare which groups
a device is *incompatible with* (exclusion predicate). Two devices would then
be co-allocatable if and only if the intersection of their exclusion sets and
their own group memberships is empty.

The inverted model is arguably more intuitive for the motivating case — a MIG
device "excludes vGPU," full stop — and does not require drivers to list each
peer group in their own entry (as Example 4 does, where `foo` devices must
include `bar` in their group list). It was rejected because:

- The co-allocation framing composes naturally with the existing DRA model,
  where counter-set membership already expresses "can share resources." A
  group is a finer-grained membership within the same model.
- Exclusion semantics require two fields to express the same information (the
  groups you *are* in, and the groups you *exclude*), or a global registry of
  group names. Membership-only is simpler.

## Infrastructure Needed (Optional)

N/A

