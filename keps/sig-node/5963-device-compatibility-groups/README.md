# KEP-5963: DRA Device Compatibility Groups

- [Release Signoff Checklist](#release-signoff-checklist)
- [Summary](#summary)
- [Motivation](#motivation)
  - [Goals](#goals)
  - [Non-Goals](#non-goals)
- [Proposal](#proposal)
  - [User Stories](#user-stories)
    - [Story 1](#story-1)
    - [Story 2](#story-2)
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
  - [Driver Responsibilities](#driver-responsibilities)
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

This KEP proposes an extension to the Dynamic Resource Allocation (DRA) API to
support mutually exclusive device allocation constraints. Hardware devices often
support multiple partitioning or virtualization schemes (for example, GPU MIG
slicing vs. MPS sharing) that provide different trade-offs in terms of isolation,
performance, and resource sharing. These schemes are frequently mutually exclusive
at the hardware level: once a physical device is partitioned or configured using
one scheme, it cannot be reconfigured to use a different scheme until all existing
allocations are released.

The current DRA Partitionable Devices API has no mechanism for drivers to express
these mutual exclusivity constraints. Without it, incompatible allocations are only
detected during resource preparation, after the scheduler has already made its
decisions, leading to pod startup failures and resource thrashing. This KEP
introduces API and scheduler changes so that compatibility constraints can be
declared in ResourceSlice objects and enforced at scheduling time.

## Motivation

Hardware devices often support multiple partitioning or virtualization schemes
that are mutually exclusive at the hardware level. For example, an NVIDIA GPU
can be configured for MIG (Multi-Instance GPU) slicing or MPS (Multi-Process
Service) sharing, but not both simultaneously on the same physical device.

Without a mechanism to express these constraints in DRA, the following problems
arise:

1. **Late Failure Detection**: Incompatible allocations are only detected during
  resource preparation, after scheduling decisions have already been made.
2. **Scheduler Unawareness**: The scheduler may allocate incompatible devices,
  leading to pod startup failures.
3. **Poor User Experience**: Users receive cryptic preparation failures instead
  of clear scheduling feedback.
4. **Resource Thrashing**: The scheduler may repeatedly attempt incompatible
  allocations before giving up.

The current workaround—having DRA drivers fail resource preparation when
incompatible allocations are attempted—is insufficient because it provides no
mechanism to inform the scheduler, and does not prevent repeated failed attempts.

### Goals

- Allow DRA drivers to specify compatibility between virtual devices within a
single physical device.
- Allow the scheduler to make informed allocation decisions that respect
compatibility rules declared in ResourceSlice objects.
- Provide a generic mechanism applicable to any hardware with partitioning
constraints, not just GPUs.
- Maintain backward compatibility with existing ResourceSlice specifications.

### Non-Goals

- Allow DRA drivers to specify compatibility between physical or virtual devices
across different physical devices. The scope of compatibility constraints is limited to virtual devices sharing the same
underlying physical device.

## Proposal

**CompatibilityGroups Assignment**

Add a `device.consumesCounters[].compatibilityGroups` field. Devices declare which  
named groups they belong to. For two devices consuming counters from the same  
counter set to be co-allocated, they must share at least one compatibility group.

Devices without this field are considered compatible with other devices that dont
specify this field, for backwards compatibility. 

### User Stories

#### Story 1

As a GPU operator using NVIDIA GPUs, I want to express in my ResourceSlice
that MIG-partitioned virtual devices and MPS-sharing virtual devices on the
same physical GPU are mutually exclusive. When a pod requesting a MIG partition
is already running on a GPU, I want the scheduler to automatically exclude all
MPS devices on that same GPU from consideration for new allocations, rather than
allowing an allocation that will fail at device preparation time.

#### Story 2

As a hardware vendor publishing DRA drivers for an accelerator that supports
multiple exclusive operating modes (for example, exclusive mode, software
partitioning, and hardware partitioning), I want to declare the compatibility
constraints directly in my ResourceSlice, so that the Kubernetes scheduler
can enforce those constraints without requiring my driver to fail pod startup
with cryptic error messages.

### Notes/Constraints/Caveats

The compatibility constraint is bidirectional and transitive: if device A
specifies a constraint that excludes device B, then allocating A must prevent
B from being allocated, and vice versa. This proposal implements this
bidirectional check in the scheduler.

### Risks and Mitigations

**Scheduler performance impact**: Evaluating compatibility constraints during  
device selection adds work to each scheduling cycle that involves DRA devices.

**Older schedulers ignoring new field**: A kube-scheduler that does not  
understand `compatibilityGroups` will ignore this  
field and may allocate incompatible devices. This degrades to the current  
behavior (driver fails at preparation time). Mitigation: document the version  
skew behavior clearly; drivers must still validate at preparation time even  
when the scheduler enforces constraints.

**Incorrect driver declarations**: If a driver declares incorrect compatibility
constraints, the scheduler may either reject valid allocations or permit invalid
ones. Mitigation: the API is driver-authored and opt-in; drivers are responsible
for correctness and documentation of their compatibility matrix.

## Design Details

### API

#### CompatibilityGroups Assignment

A new field `compatibilityGroups` is added inside each entry of
`device.consumesCounters[]`. It contains a list of string group names.
For two devices consuming counters from the same counter set to be allocated
together, they must share at least one group name. Devices that omit this
field are considered compatible with all groups.

Example showing MIG and FOO partitions on the same physical GPU:

```yaml
apiVersion: resource.k8s.io/v1
kind: ResourceSlice
spec:
  driver: gpu.example.com
  pool:
    name: node-1-pool
    generation: 1
    resourceSliceCount: 2
  sharedCounters:
    - name: gpu-1-cs
      counters:
        multiprocessors:
          value: "152"
---
apiVersion: resource.k8s.io/v1
kind: ResourceSlice
spec:
  driver: gpu.example.com
  pool:
    name: node-1-pool
    generation: 1
    resourceSliceCount: 2
  nodeName: node-1
  devices:
    - name: gpu-1-mig1
      consumesCounters:
        - counterSet: gpu-1-cs
          compatibilityGroups:
            - mig
          counters:
            multiprocessors:
              value: "2"
    - name: gpu-1-foo-part
      consumesCounters:
        - counterSet: gpu-1-cs
          compatibilityGroups:
            - foo
            - bar
          counters:
            multiprocessors:
              value: "17"
    - name: gpu-1-bar-part
      consumesCounters:
        - counterSet: gpu-1-cs
          compatibilityGroups:
            - foo
            - bar
          counters:
            multiprocessors:
              value: "17"
```

- `gpu-1-mig1` and `gpu-1-foo-part` share no compatibility group (`mig` vs
`foo`/`bar`), so they cannot be co-allocated on the same counter set.
- `gpu-1-foo-part` and `gpu-1-bar-part` share compatibility groups (`foo`, `bar`),  
so they can be co-allocated on the same counter set.

### Examples

The following examples demonstrate the problem and the proposed solution using
a GPU that supports two mutually exclusive partitioning schemes: MIG (hardware
partitioning into isolated instances) and MPS (software-level time-sharing).

#### Example 1: What the existing API enables

The DRA Partitionable Devices API uses shared counter sets to track the
capacity of a physical device across its virtual partitions. When all virtual
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
        selectors:
          - cel:
              expression: >-
                device.driver == 'gpu.example.com' &&
                device.attributes['type'].string == 'mig-1g'
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
        selectors:
          - cel:
              expression: >-
                device.driver == 'gpu.example.com' &&
                device.attributes['type'].string == 'mig-1g'
```

The scheduler allocates `gpu-0-mig-1g-0` to pod-a and `gpu-0-mig-1g-1` to
pod-b. Both consume from `gpu-0-counters` (20 + 20 = 40 <= 100). Both pods
start successfully because both devices use the same MIG partitioning mode.

#### Example 2: How the existing API does not solve the problem

When a driver advertises devices from multiple mutually exclusive partitioning
schemes on the same GPU, all sharing the same counter set, the current API has
no way to express that these schemes cannot coexist.

ResourceSlices — the same GPU now advertising both MIG and MPS devices:

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
    # MPS shares
    - name: gpu-0-mps-0
      attributes:
        type:
          string: "mps"
      consumesCounters:
        - counterSet: gpu-0-counters
          counters:
            multiprocessors:
              value: "50"
    - name: gpu-0-mps-1
      attributes:
        type:
          string: "mps"
      consumesCounters:
        - counterSet: gpu-0-counters
          counters:
            multiprocessors:
              value: "50"
```

ResourceClaims — pod-a requests a MIG partition, pod-b requests an MPS share:

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
        selectors:
          - cel:
              expression: >-
                device.driver == 'gpu.example.com' &&
                device.attributes['type'].string == 'mig-1g'
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
        selectors:
          - cel:
              expression: >-
                device.driver == 'gpu.example.com' &&
                device.attributes['type'].string == 'mps'
```

The scheduler sees `gpu-0-mig-1g-0` (20 SMs) and `gpu-0-mps-0` (50 SMs).
Total: 70 <= 100 — the counter capacity check passes. The scheduler allocates
both. But at preparation time, the driver fails because MIG and MPS cannot be
active simultaneously on the same physical GPU. Pod-b gets a cryptic
preparation error. The scheduler may retry the same incompatible combination
repeatedly, causing resource thrashing.

#### Example 3: How the proposed API solves the problem

With `compatibilityGroups`, the driver declares that MIG devices belong to the
`"mig"` group and MPS devices belong to the `"mps"` group. The scheduler
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
    # MPS shares
    - name: gpu-0-mps-0
      attributes:
        type:
          string: "mps"
      consumesCounters:
        - counterSet: gpu-0-counters
          compatibilityGroups:
            - mps
          counters:
            multiprocessors:
              value: "50"
    - name: gpu-0-mps-1
      attributes:
        type:
          string: "mps"
      consumesCounters:
        - counterSet: gpu-0-counters
          compatibilityGroups:
            - mps
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
        selectors:
          - cel:
              expression: >-
                device.driver == 'gpu.example.com' &&
                device.attributes['type'].string == 'mig-1g'
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
        selectors:
          - cel:
              expression: >-
                device.driver == 'gpu.example.com' &&
                device.attributes['type'].string == 'mps'
```

The scheduler allocates `gpu-0-mig-1g-0` (group: `mig`) to pod-a. When
evaluating `gpu-0-mps-0` (group: `mps`) for pod-b, it checks
compatibility: both devices consume from `gpu-0-counters`, but they share no
compatibility group (`mig` vs `mps`). The scheduler rejects the allocation and
pod-b becomes Unschedulable with event: "claim violates device compatibility
constraints". No cryptic preparation failure, no resource thrashing.

Two MIG devices (both group: `mig`) or two MPS devices (both group: `mps`) can
still be co-allocated, since they share a group.

#### Example 4: Multiple compatible groups with an incompatible group

A device may support more than two partitioning schemes, some of which can
coexist. In this example, a device advertises three partition types: `foo`,
`bar`, and `baz`. `foo` and `bar` can coexist on the same device, but `baz`
is incompatible with both. To express this, `foo` devices include `bar` in
their compatibility groups and vice versa, while `baz` devices only list
their own group.

ResourceSlices — a device advertising foo, bar, and baz partitions:

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
            - bar
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
            - foo
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

`device-0-foo-0` (groups: `foo`, `bar`) and `device-0-bar-0` (groups: `bar`,
`foo`) share compatibility groups, so they can be co-allocated. `device-0-baz-0`
belongs only to `baz`, which shares no group with either foo or bar devices, so
it cannot be co-allocated with them.

For instance, if pod-a is allocated `device-0-foo-0`, a subsequent pod
requesting `device-0-bar-0` succeeds (both share `foo` and `bar`), but a pod
requesting `device-0-baz-0` is rejected (`foo`/`bar` vs `baz` — no shared
group).

### Scheduler Changes

The DRA scheduler plugin is enhanced to:

1. Maintain a cache of allocated devices per node, including their compatibility
  fields (`compatibilityGroups` values).
2. For each candidate device during allocation, evaluate whether it is compatible
  with all currently allocated devices on the node, and whether all allocated
   devices are compatible with it (bidirectional check).
3. Remove candidate devices from consideration if they violate compatibility
  constraints.
4. Emit clear scheduling events when a device is rejected due to compatibility.

### Driver Responsibilities

Resource drivers are responsible for:

1. Populating `compatibilityGroups` for all devices with compatibility requirements.
2. Ensuring compatibility rules are symmetric and consistent across all devices
  in a ResourceSlice.
3. Documenting their compatibility matrix.
4. Continuing to validate at resource preparation time for version-skew safety.

### Test Plan

[X] I/we understand the owners of the involved components may require updates to  
existing tests to make this code solid enough prior to committing the changes necessary  
to implement this enhancement.

##### Prerequisite testing updates

##### Unit tests

- TBD

##### Integration tests

- TBD

##### e2e tests

- TBD

### Graduation Criteria
#### Alpha
- API defined and implemented
- All relevant code is merged and placed behind a feature flag
- Unit and integration tests
- Documentation

#### Beta
- E2E tests passing in CI 
- Validated with at least one production DRA driver (out-of-tree testing)

#### GA
- At least 2 releases as beta

### Upgrade / Downgrade Strategy
#### Upgrade
Upon upgrading, no `ResourceSlice` leverages the new optional fields yet, so the current behavior remains as-is

#### Downgrade
If downgrading to a version that does not have this enhancement implemented, older schedulers and api-servers do not know of the added optional field, and revert to their defined behavior prior to this enhancement

Allocated devices that leveraged this new field will remain allocated, and future allocations will not take `compatibilityGroups` into consideration.


### Version Skew Strategy
No version skew concerns

## Production Readiness Review Questionnaire

### Feature Enablement and Rollback

###### How can this feature be enabled / disabled in a live cluster?

- Feature gate
  - Feature gate name: DRADeviceCompatibilityGroups
  - Components depending on the feature gate: kube-scheduler, kube-apiserver
- Other
  - Describe the mechanism:
  - Will enabling / disabling the feature require downtime of the control
  plane?
  - Will enabling / disabling the feature require downtime or reprovisioning
  of a node?

###### Does enabling the feature change any default behavior?
No, this KEP proposes an additional optional field to the `ResourceSlice` API

###### Can the feature be disabled once it has been enabled (i.e. can we roll back the enablement)?
Yes, rolling back the enablement will revert the cluster to its pre-enablement behavior

###### What happens if we reenable the feature if it was previously rolled back?
Existing `compatibilityGroup` configurations in `ResourceSlice`s will become effective again

###### Are there any tests for feature enablement/disablement?
Yes, there will be integration tests to verify feature enablement/disablement

### Rollout, Upgrade and Rollback Planning

###### How can a rollout or rollback fail? Can it impact already running workloads?
I expect code changes in `kube-apiserver` and `kube-scheduler`, so something can go wrong with those.
No impact on already running workloads.

###### What specific metrics should inform a rollback?
TBD

###### Were upgrade and rollback tested? Was the upgrade->downgrade->upgrade path tested?
TBD

###### Is the rollout accompanied by any deprecations and/or removals of features, APIs, fields of API types, flags, etc.?
No

### Monitoring Requirements

###### How can an operator determine if the feature is in use by workloads?
This feature is not intended for use by workload usage, it is intended for DRA Drivers

###### How can someone using this feature know that it is working for their instance?

- Events
  - Scheduling events:
    - When all allocated devices in all Nodes are not compatible with any device that is considered for allocation the following event will be emitted by the scheduler for each Node: "No available nodes found: claim violates device compatibility constraints"
- Pod.status
  - Condition name: Unschedulable

###### What are the reasonable SLOs (Service Level Objectives) for the enhancement?
N/A

###### What are the SLIs (Service Level Indicators) an operator can use to determine the health of the service?
N/A

###### Are there any missing metrics that would be useful to have to improve observability of this feature?
No

### Dependencies
DRA Partitionable Devices enabled

###### Does this feature depend on any specific services running in the cluster?
No

### Scalability

###### Will enabling / using this feature result in any new API calls?
No

###### Will enabling / using this feature result in introducing new API types?
No, only a new API field

###### Will enabling / using this feature result in any new calls to the cloud provider?
No

###### Will enabling / using this feature result in increasing size or count of the existing API objects?
Yes, additional field to the `ResourceSlice` API

###### Will enabling / using this feature result in increasing time taken by any operations covered by existing SLIs/SLOs?
Scheduling cycles will take longer to complete due to the additional responsibility the scheduler will recieve, I expect it to be negligible

###### Will enabling / using this feature result in non-negligible increase of resource usage (CPU, RAM, disk, IO, ...) in any components?
No

###### Can enabling / using this feature result in resource exhaustion of some node resources (PIDs, sockets, inodes, etc.)?
No

### Troubleshooting

###### How does this feature react if the API server and/or etcd is unavailable?
No new side effects

###### What are other known failure modes?
N/A

###### What steps should be taken if SLOs are not being met to determine the problem?
TBD

## Implementation History

## Drawbacks

Adding compatibility constraint support to the scheduler increases the  
complexity of the DRA scheduling logic. The new field must be evaluated for  
every device candidate during every scheduling cycle that involves DRA  
resources, which adds latency and memory overhead.

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

## Infrastructure Needed (Optional)

