# [KEP-4815](https://github.com/kubernetes/enhancements/issues/4815): DRA: Add support for partitionable devices

<!-- toc -->
- [Release Signoff Checklist](#release-signoff-checklist)
- [Summary](#summary)
- [Motivation](#motivation)
  - [Dynamic allocation of Multi-Instance GPUs (MIG) on NVIDIA hardware](#dynamic-allocation-of-multi-instance-gpus-mig-on-nvidia-hardware)
  - [Multi-host Tensor Processing Unit (TPU) scheduling](#multi-host-tensor-processing-unit-tpu-scheduling)
  - [Goals](#goals)
  - [Non-Goals](#non-goals)
- [Proposal](#proposal)
  - [Risks and Mitigations](#risks-and-mitigations)
    - [Partial scheduling of pods for multi-host devices](#partial-scheduling-of-pods-for-multi-host-devices)
- [Design Details](#design-details)
  - [Extending device and capacity pool with a set of mixins](#extending-device-and-capacity-pool-with-a-set-of-mixins)
  - [Defining device partitions in terms of consumed capacity in a composite device](#defining-device-partitions-in-terms-of-consumed-capacity-in-a-composite-device)
  - [Defining multi-host devices](#defining-multi-host-devices)
    - [Multi-host scheduling limitations](#multi-host-scheduling-limitations)
  - [Putting it all together for the MIG use-case](#putting-it-all-together-for-the-mig-use-case)
  - [Using DRA for the multi-host use-case](#using-dra-for-the-multi-host-use-case)
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
<!-- /toc -->

## Release Signoff Checklist

Items marked with (R) are required *prior to targeting to a milestone / release*.

- [x] (R) Enhancement issue in release milestone, which links to KEP dir in [kubernetes/enhancements] (not the initial KEP PR)
- [x] (R) KEP approvers have approved the KEP status as `implementable`
- [x] (R) Design details are appropriately documented
- [ ] (R) Test plan is in place, giving consideration to SIG Architecture and SIG Testing input (including test refactors)
  - [ ] e2e Tests for all Beta API Operations (endpoints)
  - [ ] (R) Ensure GA e2e tests meet requirements for [Conformance Tests](https://github.com/kubernetes/community/blob/master/contributors/devel/sig-architecture/conformance-tests.md) 
  - [ ] (R) Minimum Two Week Window for GA e2e tests to prove flake free
- [ ] (R) Graduation criteria is in place
  - [ ] (R) [all GA Endpoints](https://github.com/kubernetes/community/pull/1806) must be hit by [Conformance Tests](https://github.com/kubernetes/community/blob/master/contributors/devel/sig-architecture/conformance-tests.md) 
- [x] (R) Production readiness review completed
- [x] (R) Production readiness review approved
- [x] "Implementation History" section is up-to-date for milestone
- [ ] User-facing documentation has been created in [kubernetes/website], for publication to [kubernetes.io]
- [ ] Supporting documentation—e.g., additional design documents, links to mailing list discussions/SIG meetings, relevant PRs/issues, release notes

<!--
**Note:** This checklist is iterative and should be reviewed and updated every time this enhancement is being considered for a milestone.
-->

[kubernetes.io]: https://kubernetes.io/
[kubernetes/enhancements]: https://git.k8s.io/enhancements
[kubernetes/kubernetes]: https://git.k8s.io/kubernetes
[kubernetes/website]: https://git.k8s.io/website

## Summary

One of the original use-cases for Dynamic Resource Allocation (DRA) was the
ability to dynamically allocate partitions of a full device (in addition to
the full device itself). Whereas the traditional device plugin API forces users
to statically partition devices ahead of time, DRA was designed to allow those
partitions to be created on demand. This leads to increased resource
utilization as the size of each partitioned device can be matched in real-time
to the workload requesting it.

Devices represented in DRA don't necessarily have to be a single unit connected
to a single machine, but can also be a logical device comprised of multiple
devices connected to multiple machines. Similar to the single device partitioning,
users might require either the full multi-host device or a subset.

As DRA has evolved from what we now call "classic" DRA to "structured
parameters" this ability to dynamically partition devices has been lost.
This KEP proposes a method to bring this capability back within the framework
that "structured parameters" provides. Additionally, it provides primitives to
represent both full devices and their partitions in a more compact way than is
possible today.

Note that the extensions proposed in this KEP are completely transparent to the
end-user. In 1.31, it is already possible to select a device partition via a
`ResourceClaim`, so long as the device vendor has pre-partitioned its devices
and advertised them as such inside  a `ResourceSlice`. The user-facing
mechanism to select these devices doesn't change with the extensions proposed
in this KEP. Instead, the proposed extensions provide the ability for a vendor
to advertise "overlapping" partitions, such that the scheduler will never
allocate conflicting ones at the same time. This, in turn, gives the vendor the
opportunity to dynamically create these partitions *after* they have been
allocated, rather than requiring them to be created before.

## Motivation

We have several motivating examples for supporting partitionable devices with
DRA, with the first two described in detail in this document. Additional details
of the other use-cases can be found
[here](https://docs.google.com/document/d/1lXGfnrBixRIMW9ESa-mv09Kisb2myVFV_A3nqPJ4FCQ/edit?usp=sharing).

* Partitioning a single GPU into smaller partitions.
* Multi-host scheduling of interconnected TPUs.
* SR-IOV
* Multi-host logical groupings to avoid deadlocks in LeaderWorkSet scheduling
* Single-host allocations of multiple TPUs in valid topologies
* Single-host allocations of multiple Inferentia and Trainium device in valid topologies

### Dynamic allocation of Multi-Instance GPUs (MIG) on NVIDIA hardware

MIG devices are represented as fixed-size partitions
of a full GPU that consume a portion of its capacity across multiple
dimensions. These dimensions include things like number of JPEG engines, number
of multiprocessors, and the allocation of a specific set of fixed-size memory
slices.

In general, the exact capacity needed to instantiate a given MIG device is
defined by its MIG "profile". Multiple instances of a MIG profile can be
instantiated into concrete MIG devices if and only if enough resources are
available to accommodate it.

For example, the following table shows all of the MIG profiles available on an
NVIDIA A100 40GB GPU:

| Profile Name | Memory Size | Memory Slices | Multiprocessors | Copy Engines | Decoders | Encoders | JPEG Engines | OFA Engines |
|--------------|-------------|---------------|-----------------|--------------|----------|----------|--------------|-------------|
|   7g.40gb    |   40192Mi   |       8       |       98        |      7       |    5     |    0     |      1       |      1      |
|   4g.20gb    |   19968Mi   |       4       |       56        |      4       |    2     |    0     |      0       |      0      |
|   3g.20gb    |   19968Mi   |       4       |       42        |      3       |    2     |    0     |      0       |      0      |
|   2g.10gb    |    9856Mi   |       2       |       28        |      2       |    1     |    0     |      0       |      0      |
|   1g.10gb    |    9856Mi   |       2       |       14        |      1       |    1     |    0     |      0       |      0      |
|  1g.5gb+me   |    4864Mi   |       1       |       14        |      1       |    1     |    0     |      1       |      1      |
|   1g.5gb     |    4864Mi   |       1       |       14        |      1       |    0     |    0     |      0       |      0      |

Zooming in on a single dimension (memory slices), the following diagram shows
the physical layout where multiple instances of each profile "could" be
instantiated (so long as enough resources are available to accommodate them).
Only those instances which do not overlap vertically can be allocated
simultaneously (because otherwise they would consume the same physical memory
slice). The X in some of the columns denotes slots where a particular profile
*could* fit in terms of consuming memory slices, but there are not enough of
the other required resources for that profile available to accommodate it.

```
+------------+------------+------------+------------+------------+------------+------------+------------+
| MemSlice 0 | MemSlice 1 | MemSlice 2 | MemSlice 3 | MemSlice 4 | Memslice 5 | MemSlice 6 | MemSlice 7 |
+------------+------------+------------+------------+------------+------------+------------+------------+
|                                                 7g.40gb                                               |
+------------+------------+------------+------------+------------+------------+------------+------------+
|                       4g.20gb                     |                         X                         |
+------------+------------+------------+------------+------------+------------+------------+------------+
|                       3g.20gb                     |                       3g.20gb                     |
+------------+------------+------------+------------+------------+------------+------------+------------+
|          2g.10gb        |          2g.10gb        |          2g.10gb        |            X            |
+------------+------------+------------+------------+------------+------------+------------+------------+
|          1g.10gb        |          1g.10gb        |          1g.10gb        |          1g.10gb        |
+------------+------------+------------+------------+------------+------------+------------+------------+
|   1g.5gb   |   1g.5gb   |   1g.5gb   |   1g.5gb   |   1g.5gb   |   1g.5gb   |   1g.5gb   |     X      |
+------------+------------+------------+------------+------------+------------+------------+------------+
| 1g.5gb+me  | 1g.5gb+me  | 1g.5gb+me  | 1g.5gb+me  | 1g.5gb+me  | 1g.5gb+me  | 1g.5gb+me  |     X      |
+------------+------------+------------+------------+------------+------------+------------+------------+
```

The most important thing to note from all of this is that partitioning a GPU
into a set of MIG devices is not as simple as slicing it up into a set of
fixed-size, non-overlapping partitions. Instead, one can think of a GPU as a
"bag" of resources that can be pulled together in a discrete set of ways to
construct a MIG device with a particular profile. When instatiating a MIG
device with a particular profile, the set of resources available to other MIG
devices (or the full GPU itself) gets depleted.

In other words, the allocation of one MIG device may nullify the
ability to allocate some subset of other possible MIG devices (but not all of
them). A MIG device is considered allocatable so long as the full GPU has
enough *unallocated* capacity to satisfy its capacity constraints across all
dimensions. If any single dimension is unavailable, that MIG device cannot be
allocated (but others might still be able to).

For example, the following `ResourceClaim` can be used to select a set of
non-overlapping MIG devices from a specific GPU.
```yaml
apiVersion: resource.k8s.io/v1alpha3
kind: ResourceClaim
metadata:
  name: mig-devices
spec:
  spec:
    devices:
      requests:
      - name: mig-1g-5gb-0
        deviceClassName: mig.nvidia.com
        selectors:
        - cel:
            expression: "device.attributes['gpu.nvidia.com'].profile == '1g.5gb'"
      - name: mig-1g-5gb-1
        deviceClassName: mig.nvidia.com
        selectors:
        - cel:
            expression: "device.attributes['gpu.nvidia.com'].profile == '1g.5gb'"
      - name: mig-2g-10gb
        deviceClassName: mig.nvidia.com
        selectors:
        - cel:
            expression: "device.attributes['gpu.nvidia.com'].profile == '2g.10gb'"
      - name: mig-3g-20gb
        deviceClassName: mig.nvidia.com
        selectors:
        - cel:
            expression: "device.attributes['gpu.nvidia.com'].profile == '3g.20gb'"
      constraints:
      - requests: []
        matchAttribute: "gpu.nvidia.com/parentUUID"
```

This would result in the following set of non-overlapping partitions:
```
+------------+------------+------------+------------+------------+------------+------------+------------+
| MemSlice 0 | MemSlice 1 | MemSlice 2 | MemSlice 3 | MemSlice 4 | Memslice 5 | MemSlice 6 | MemSlice 7 |
+------------+------------+------------+------------+------------+------------+------------+------------+
|                         X                         |                       3g.20gb                     |
+------------+------------+------------+------------+------------+------------+------------+------------+
|            X            |          2g.10gb        |            X            |            X            |
+------------+------------+------------+------------+------------+------------+------------+------------+
|   1g.5gb   |   1g.5gb   |     X      |     X      |     X      |     X      |     X      |     X      |
+------------+------------+------------+------------+------------+------------+------------+------------+
```

Note that the YAML provided above is actually a *working* example for
allocating MIG devices with the NVIDIA DRA driver for GPUs in Kubernetes 1.31.
However, it requires that the MIG profiles being requested are pre-partitioned
on a GPU ahead of time.

In contrast, the extensions proposed in this KEP allow this partitioning to be
done *after* the scheduler has allocated these devices, keeping the GPU free to
be partitioned in different ways until the actual user-workload requesting them
has been submitted.

### Multi-host Tensor Processing Unit (TPU) scheduling

TPUs are connected to VMs, usually four TPUs per VM. In order to run large
workloads that require multiple TPUs, groups of TPUs can be connected over
a high-speed inter-chip interconnect, which is important to achieve the best
performance. However, not all TPUs in the group are connected to each other,
so we need to consider the topology when we make decisions about the allocation
of TPUs to workloads.

Due to the topology, only certain specific slices of TPUs can be used.
For example, in a 64 TPU node pool there will be 16 VMs, each with 4
TPUs. This allows for a number of possible multi-VM slices of different
sizes:
* 8x8 slice, which provides 64 TPUs across 16 nodes (shown in black)
* 4x8 slices, which provides 32 TPUs across 8 nodes (shown in purple)
* 4x4 slices, which provides 16 TPUs across 4 nodes (shown in green)
* 2x4 slices, which provides 8 TPUs across 2 nodes (shown in red)

![image](tpu-topology.png)

For example, a user can request a 4x4 slice of TPUs with a `ResourceClaim`
like the following:

```yaml
apiVersion: resource.k8s.io/v1beta1
kind: ResourceClaim
metadata:
  name: tpu-device
spec:
  spec:
    devices:
      requests:
      - name: 4x4-tpu
        deviceClassName: tpu.google.com
        selectors:
        - cel:
            expression: 'device.capacity['google-tpu'].tpus == quantity("16")
```
There are four "good" allocations for this request:
* All TPUs on nodes 1, 2, 5, and 6.
* All TPUs on nodes 3, 4, 7, and 8.
* All TPUs on nodes 9, 10, 13, and 14.
* All TPUs on nodes 11, 12, 15, and 16.
A request like the one above must be allocated one of the four 4x4 slices
or it should not succeed. A request asking for just 16 TPUs will likely
result in allocation of TPUs across many VMs and without the interconnect,
leading to poor performance. So we need to allow users to request a 
partition of a device (in this case a 8x8 slice of TPUs) and account for
the fact that this uses some of the capacity required for other slices.

With these motivating examples in mind. We define the following goals and
non-goals of this KEP.

### Goals

* Introduce the ability for "structured parameters" DRA to allocate both full
  devices and fixed-size partitions of full devices (across multiple
  dimensions). Both full devices and their valid set of partitions must be
  explicitly listed in a ResourceSlice for the scheduler to consider them for
  allocation.

* Keep the user-facing mechanism to request a full device or one of its
  partitions the same. In 1.31, it is already possible to select a device
  partition via a `Resourceclaim`, so long as the device vendor has
  pre-partitioned its devices and advertised them in a `ResourceSlice`. The
  user-facing mechanism to select these devices shouldn't change -- only the
  ability for the vendor to advertise "overlapping" partitions, such that the
  scheduler will never allocate conflicting ones at the same time.

* Provide abstractions to keep the overall footprint of a `ResourceSlice`
  small, such that all full devices (and their partitions) can be represented
  in a concise way.

### Non-Goals

* Allow a user to allocate arbitrarily-sized partitions of a device. In other
  words, only those partitions explicitly defined by a `ResourceSlice` will be
  allocatable by a user. Arbitrary slicing is out of scope for this KEP.

## Proposal

At a high level, this proposal is based around three new concepts:

1. Capacity pool, which is a construct for defining the capacities available for
   sharing among partitions of a single physical device. Drivers will
   typically create a capacity pool that specifies all the available capacities of
   a physical device and then define the partitions available for DRA. Each
   partition will specify how much capacity it will consume from the capacity
   pool, which allows the scheduler to keep track of available capacity in
   a pool and decide whether a partition can be allocated. A device will
   separately define the capacities that are accessible to the user for the partition.

1. Mixins, which allows devices and resource pools to reference a mixin
   object that defines some properties and that when used, will extend the properties
   they define explicitly on the device or resource pool. The objective is to reduce
   the duplication and overall footprint of `ResourceSlices`.

1. Node selection delegated to devices. Node selection can be delegated to
   each individual device, which allows different devices in the same
   `ResourceSlice` to be available from different nodes. This provides
   the flexibility to describe multi-host scenarios where a single logical
   device might span multiple nodes.

The implementation of these concepts requires several changes to the
ResourceSlice API. It introduces two new fields on the `ResourceSliceSpec`.
`SharedCounters` defines a list of `CounterSet` while `Mixins` define the
mixins. It also introduces a new field `PerDeviceNodeSelection` on the
`ResourceSliceSpec` and new fields on the device that mirrors the node selector
fields on the `ResourceSlice`.

1. The `SharedCounters` field is a list of named `CounterSet`s. Each
   defines a set of capacities that is available for devices. This makes it possible
   to define overlapping partitions of devices, while still making sure that no
   device can be allocated if the necessary capacity is not available. A `CounterSet`
   can optionally reference one or more `CounterSetMixin`s using the `Includes` field
   to extend the set of capacities it defines explicitly.

1. The `Mixins` field is of type `ResourceSliceMixin`, which has fields for the three
   types of mixins available in a ResourceSlice. These are `CounterSet`,
   `Device`, and `DeviceCounterConsumption`.

    1. The `CounterSet` field defines a list of named `CounterSetMixin`s. These
      define a set of capacities that can be used to extend the capacities explicitly
      defined in a `CounterSet`. This allows for reduced duplication if there are many
      identical physical devices that must be represented as capacity pools. `CounterSetMixin`s
      can not be referenced directly by devices.

    1. The `Device` field is a list of named `DeviceMixin`s. These define
      attributes and capacities that can be used to extend what is defined
      explicitly in `BasicDevice` (introduced
      in more detail below). Mixins cannot be allocated directly,
      but can only be referenced by composite devices.
    
    1. The `DeviceCounterConsumption` field defines a list of named 
       `DeviceCounterConsumptionMixin`s. These define capacities that can be
       used to extend the capacity consumption defined explicitly in fields under
       `BasicDevice`. The capacity pool is not specified in the
       `DeviceCounterConsumptionMixin`, but rather provided when the mixin
       is referenced from the device.

1. Introduce a couple of new fields added under `BasicDevice`. The first is a field called
   `Includes` and lists the device mixins for the device, while the second is
   called `ConsumesCounter` and defines the capacity the device will draw
   from the capacity pools.

    1. The `Includes` field serves to reference a set of `DeviceMixin`s that a
      `BasicDevice` can reference to extend the set of attributes,
      capacities it defines explicitly.

    1. The `ConsumesCounter` field defines the capacities the device
      will draw from the capacity pool when allocated. Therefore, if this capacity isn't
      available in the pool, the device can not be allocated. Only references
      to capacity pools in the same `ResourceSlice` is supported.

1. Add a new field `PerDeviceNodeSelection` to the `ResourceSliceSpec` and the
   fields `NodeName`, `NodeSelector` and `AllNodes` on the `Device`.

      1. The `PerDeviceNodeSelection` field is of type boolean and is mutually
      exclusive with the existing node selection fields in the `ResourceSliceSpec`
      (`NodeName`, `NodeSelector`, and `AllNodes`). If the value of this field is
      `true`, then the node association must be specified on each device.

      1. The fields `NodeName`, `NodeSelector`, and `AllNodes` fields mirror the
      fields on the `ResourceSliceSpec` and are mutually exlusive. Setting
      `NodeName` means the device is available on a specific node, setting the
      `NodeSelector` field means the device is available on all nodes matching
      the selector, while setting the `AllNodes` field to `true` means the
      device is available on all nodes in the cluster.

With these additions in place, the scheduler has everything it needs to support
the dynamic allocation of full devices, their (possibly overlapping)
fixed-size partitions, and multi-host devices. That is to say, the scheduler now has
the ability to "flatten" all devices and capacity pools by applying any referenced mixins as
well as track the capacities consumed by allocated devices. More details
on the actual algorithm the scheduler follows to make allocation decisions
based on the capacity pools can be found in the Design Details section below.

### Risks and Mitigations

#### Partial scheduling of pods for multi-host devices

With multi-host devices, there will typically be multiple pods sharing a single
`ResourceClaim`. DRA guarantees that the pods will not end up on nodes that are not
part of the multi-host device. But it can not guarantee that all pods will be
scheduled, since pods will be subject to any other constraints (like sufficient
CPU and memory) during scheduling.

A better story should be in place for beta, including a plan for alignment and
possible integration with Kueue.

## Design Details

The exact set of proposed API changes can be seen below:
```go
// ResourceSliceSpec contains the information published by the driver in one ResourceSlice.
type ResourceSliceSpec struct {
  ...

  // SharedCounters defines a list of counter sets, each of which
  // has a name and a list of counters available.
  //
  // The names of the SharedCounters must be unique in the ResourceSlice.
  //
  // The maximum number of SharedCounters is 32.
  //
  // +optional
  // +listType=atomic
  SharedCounters []CounterSet

  // Mixins defines the mixins available for devices and capacity pools
  // in the ResourceSlice.
  //
  // +optional
  Mixins ResourceSliceMixins

  // PerDeviceNodeSelection defines whether the access from nodes to
  // resources in the pool is set on the ResourceSlice level or on each
  // device. If it is set to true, every device defined the ResourceSlice
  // must specify this individually.
  //
  // Exactly one of NodeName, NodeSelector, AllNodes, and PerDeviceNodeSelection
  // must be set.
  //
  // +optional
  // +oneOf=NodeSelection
  PerDeviceNodeSelection bool
}

// ResourceSliceMixins defines mixins for the ResourceSlice.
type ResourceSliceMixins struct {
  // Device represents a list of device mixins, i.e. a collection of
  // shared attributes and capacities that an actual device can "include"
  // to extend the set of attributes and capacities it already defines.
  //
  // The main purposes of these mixins is to reduce the memory footprint
  // of devices since they can reference the mixins provided here rather
  // than duplicate them.
  //
  // The total number of device mixins, device capacity consumption mixins, 
  // capacity pool mixins, basic devices must be less than 128.
  //
  // +optional
  // +listType=atomic
  Device []DeviceMixin

  // DeviceCounterConsumption represents a list of counters
  // consumption mixins, each of which contains a set of counters
  // that a device will consume from a counter set.
  //
  // This makes it possible to define a set of shared counters that
  // are not tied to a specific counter set. The counter set is inferred by context
  // in which the DeviceCounterConsumptionMixin is referenced from
  // the device.
  //
  // The total number of device mixins, device counter consumption mixins, 
  // counter set mixins, basic devices must be
  // less than 128.
  //
  // +optional
  // +listType=atomic
  DeviceCounterConsumption []DeviceCcounterConsumptionMixin

  // CounterSet represents a list of counter set mixins, i.e.
  // a collection of counters that a CounterSet can "include"
  // to extend the set of counters it already defines.
  //
  // The main purposes of these mixins is to reduce the memory footprint
  // of SharedCounters since they can reference the mixins provided here rather
  // than duplicate them.
  //
  // The total number of device mixins, device counter consumption mixins, 
  // counter set mixins, and basic devices must be
  // less than 128.
  //
  // +optional
  // +listType=atomic
  CounterSet []CouterSetMixin
}

// DeviceMixin defines a specific device mixin for each device type.
// Besides the name, exactly one field must be set.
type DeviceMixin struct {
  // Name is a unique identifier among all device mixins in the ResourceSlice.
  // It must be a DNS label.
  //
  // +required
  Name string

  // Composite defines a mixin usable by a composite device.
  //
  // +optional
  // +oneOf=deviceMixinType
  Composite *CompositeDeviceMixin
}

// CompositeDeviceMixin defines a mixin that a composite device can include.
type CompositeDeviceMixin struct {
  // Attributes defines the set of attributes for this mixin.
  // The name of each attribute must be unique in that set.
  //
  // To ensure this uniqueness, attributes defined by the vendor
  // must be listed without the driver name as domain prefix in
  // their name. All others must be listed with their domain prefix.
  //
  // Conflicting attributes from those provided via other mixins are
  // overwritten by the ones provided here.
  //
  // The maximum number of attributes and capacities combined is 32.
  //
  // +optional
  Attributes map[QualifiedName]DeviceAttribute

  // Capacity defines the set of capacities for this mixin.
  // The name of each capacity must be unique in that set.
  //
  // To ensure this uniqueness, capacities defined by the vendor
  // must be listed without the driver name as domain prefix in
  // their name. All others must be listed with their domain prefix.
  //
  // Conflicting capacities from those provided via other mixins are
  // overwritten by the ones provided here.
  //
  // The maximum number of attributes and capacities combined is 32.
  //
  // +optional
  Capacity map[QualifiedName]DeviceCapacity
}

// DeviceCounterConsumptionMixin defines a mixin that partitionable
// devices can include to adopt the consuption counter defined in
// the mixin.
type DeviceCounterConsumptionMixin struct {
  // Name is a unique identifier among all device capacity consumption 
  // mixins in the ResourceSlice. It must be a DNS label.
  //
  // +required
  Name string

  // Counters defines a set of counters
  // that a device will consume from a counter set.
  //
  // The counter set is not specified here but is determined
  // from the context in which the DeviceCounterConsumptionMixin
  // is referenced from the device.
  // 
  // The maximum number of counters is 32
  //
  // +required
  Counters map[string]Counter
}

// Device represents one individual hardware instance that can be selected based
// on its attributes. Besides the name, exactly one field must be set.
// +k8s:deepcopy-gen=true
type Device struct {
  // Name is unique identifier among all devices managed by
  // the driver in the pool. It must be a DNS label.
  //
  // +required
  Name string

  // Basic defines one device instance.
  //
  // +optional
  // +oneOf=deviceType
  Basic *BasicDevice
  
  // Includes defines the set of device mixins that this device includes.
  //
  // The propertes of each included mixin are applied to this device in
  // order. Conflicting properties from multiple mixins are taken from the
  // last mixin listed that contains them. Properties set on the device will
  // always override properties from mixins.
  //
  // The mixins referenced here must be defined in the same
  // ResourceSlice.
  //
  // The maximum number of mixins that can be included is 8.
  //
  // +optional
  // +listType=atomic
  Includes []DeviceMixinRef

  // Attributes defines the set of attributes for this device.
  // The name of each attribute must be unique in that set.
  //
  // To ensure this uniqueness, attributes defined by the vendor
  // must be listed without the driver name as domain prefix in
  // their name. All others must be listed with their domain prefix.
  //
  // Conflicting attributes from those provided via mixins are
  // overwritten by the ones provided here.
  //
  // The maximum number of attributes and capacities combined is 32.
  //
  // +optional
  Attributes map[QualifiedName]DeviceAttribute

  // Capacity defines the set of capacities for this device.
  // The name of each capacity must be unique in that set.
  //
  // To ensure this uniqueness, capacities defined by the vendor
  // must be listed without the driver name as domain prefix in
  // their name. All others must be listed with their domain prefix.
  //
  // Conflicting capacities from those provided via other mixins are
  // overwritten by the ones provided here.
  //
  // The maximum number of attributes and capacities combined is 32.
  //
  // +optional
  Capacity map[QualifiedName]DeviceCapacity

  // ConsumesCounter defines a list of references to shared counters
  // and the set of counters that the device will
  // consume from those counter sets.
  //
  // The counters can be defined either by referencing one
  // or more DeviceCounterConsumptionMixins by listing
  // the counters directly. The latter will always override
  // any counters coming in from the mixins.
  //
  // The maximum number of device counter consumption entries
  // is 32. This is the same as the maximum number of shared counter
  // allowed in a ResourceSlice.
  //
  // +required
  // +listType=atomic
  ConsumesCounters []DeviceCounterConsumption

  // NodeName identifies the node where the device is available.
  //
  // Must only be set if Spec.PerDeviceNodeSelection is set.
  // At most one of NodeName, NodeSelector and AllNodes can be set.
  //
  // +optional
  // +oneOf=DeviceNodeSelection
  NodeName string

  // NodeSelector defines the nodes where the device is available.
  //
  // Must use exactly one term.
  //
  // Must only be set if Spec.PerDeviceNodeSelection is set.
  // At most one of NodeName, NodeSelector and AllNodes can be set.
  //
  // +optional
  // +oneOf=DeviceNodeSelection
  NodeSelector *core.NodeSelector

  // AllNodes indicates that all nodes have access to the device.
  //
  // Must only be set if Spec.PerDeviceNodeSelection is set.
  // At most one of NodeName, NodeSelector and AllNodes can be set.
  //
  // +optional
  // +oneOf=DeviceNodeSelection
  AllNodes bool
}

// CounterSetMixin defines a mixin that a counter set can include.
type CounterSetMixin struct {
  // Name is a unique identifier among all counter set mixins in the ResourceSlice.
  // It must be a DNS label.
  //
  // +required
  Name string

  // Counters defines the set of counters for this mixin.
  // The name of each counter must be unique in that set and a DNS label.
  //
  // Conflicting counters from those provided via other mixins are
  // overwritten by the ones provided here.
  //
  // The maximum number of counters is 32.
  //
  // +required
  Counters map[string]Counter
}

// CounterSet defines a named set of counters
// that are available to be used by devices defined in the
// ResourceSlice.
//
// The counters are not allocatable by themselves, but
// can be referenced by devices. When a device is allocated,
// the portion of counters it uses will no longer be available for use
// by other devices.
type CounterSet struct {
    // Name defines the name of the counter set.
    // It must be a DNS label.
    //
    // +required
    Name string
    
    // Counters defines the set of counters for this CounterSet
    // The name of each counter must be unique in that set and must be a DNS label.
    //
    // The maximum number of counters is 32.
    //
    // +required
    Counters map[string]Counter

  // Includes defines the set of counter set mixins that this counter set includes.
  //
  // The propertes of each included mixin are applied to this counter set in
  // order. Conflicting properties from multiple mixins are taken from the
  // last mixin listed that contains them. Properties set on the counter set will
  // always override properties from mixins.
  //
  // The mixins referenced here must be defined in the same
  // ResourceSlice.
  //
  // The maximum number of mixins that can be included is 8.
  //
  // +optional
  // +listType=atomic
  Includes []CounterSetMixinRef
}

// DeviceMixinRef defines a reference to a device mixin.
type DeviceMixinRef struct {
  // Name refers to the name of a device mixin in the pool.
  //
  // +required
  Name string
}

// CounterSetMixinRef defines a reference from a counter set
// to a counter set mixin.
type CounterSetMixinRef struct {
  // Name is the name of a CounterSetMixin.
  //
  // +required
  Name string
}

type DeviceCounterConsumptionMixinRef struct {
  // Name is the name of a DeviceCounterConsumptionMixin.
  //
  // +required
  Name string
}

// DeviceCounterConsumption defines a set of counters that
// a device will consume from a CounterSet.
type DeviceCounterConsumption struct {
    // SharedCounter defines the shared counter from which the
    // counters defined will be consumed.
    //
    // +required
    SharedCounter string
    
    // Counters defines the Counter that will be consumed by
    // the device.
    //
    //
    // The maximum number of Counters is 32.
    //
    // +required
    Counters map[string]Counter

  // Includes defines a list of references to DeviceCounterConsumptionMixins.
  // The counters listed in these will be included in among the
  // counters that will be consumed by the device.
  //
  // Counters listed directly will override any counters coming
  // from mixins.
  //
  // The maximum number of mixins that can be included is 8.
  //
  // +optional
  Includes []DeviceCapacityConsumptionMixinRef
}

// Counter describes a quantity associated with a device.
type Counter struct {
    // Value defines how much of a certain device counter is available.
    //
    // +required
    Value resource.Quantity
}
```

As mentioned previously, the main features being added here are (1) the ability
to include a set of mixins in a device or capacity pool definition, (2) the
ability to express that multiple devices draw from the same pool of capacity, so
allocation of one device might make other devices unallocatable, and (3) the
ability to define multi-host devices.

To simplify the conversation, we discuss each new feature separately, starting
with "mixins" for both capacity pools and devices, which allows a set of mixins to
extend the properties defined explicitly.

### Extending device and capacity pool with a set of mixins

A simple example the defines a set of mixins and includes them in the
definition of 4 NVIDIA A100 GPUs can be seen below:

```yaml
sharedCounters:
- name: gpu-0-pool
  includes:
  - name: gpu-pool-mixin
- name: gpu-1-pool
  includes:
  - name: gpu-pool-mixin
- name: gpu-2-pool
  includes:
  - name: gpu-pool-mixin
- name: gpu-3-pool
  includes:
  - name: gpu-pool-mixin
mixins:
  counterSet:
  - name: gpu-pool-mixin
    counters:
      memory: 
        value: 40Gi
  device:
  - name: system-attributes
    basic:
      attributes:
        cudaDriverVersion:
          version: 12.6.0
        driverVersion:
          version: 560.35.3
  - name: common-gpu-attributes
    basic:
      attributes:
        type:
          string: gpu
        architecture:
          string: Ampere
        brand:
          string: Nvidia
        productName:
          string: NVIDIA A100-SXM4-40GB
        cudaComputeCapability:
          version: 8.0.0
  - name: common-gpu-capacities
    capacity:
      memory: 
        value: 40Gi
  deviceCounterConsumption:
  - name: common-gpu-consumption
    counters:
      memory: 
        value: 40Gi
devices:
- name: gpu-0
  basic:
    includes:
    - name: system-attributes
    - name: common-gpu-attributes
    - name: common-gpu-capacities
    attributes:
      index:
        int: 0
      minor:
        int: 0
      uuid:
        string: GPU-4cf8db2d-06c0-7d70-1a51-e59b25b2c16c
    consumesCounter:
    - counterSet: gpu-0-pool
      includes:
      - name: common-gpu-consumption
- name: gpu-1
  basic:
    includes:
    - name: system-attributes
    - name: common-gpu-attributes
    - name: common-gpu-capacities
    attributes:
      index:
        int: 1
      minor:
        int: 1
      uuid:
        string: GPU-4404041a-04cf-1ccf-9e70-f139a9b1e23c
    consumesCounter:
    - counterSet: gpu-1-pool
      includes:
      - name: common-gpu-consumption
- name: gpu-2
  basic:
    includes:
    - name: system-attributes
    - name: common-gpu-attributes
    - name: common-gpu-capacities
    attributes:
      index:
        int: 2
      minor:
        int: 2
      uuid:
        string: GPU-79a2ba02-a537-ccbf-2965-8e9d90c0bd54
    consumesCounter:
    - counterSet: gpu-2-pool
      includes:
      - name: common-gpu-consumption
- name: gpu-3
  basic:
    includes:
    - name: system-attributes
    - name: common-gpu-attributes
    - name: common-gpu-capacities
    attributes:
      index:
        int: 3
      minor:
        int: 3
      uuid:
        string: GPU-662077db-fa3f-0d8f-9502-21ab0ef058a2
    consumesCounter:
    - counterSet: gpu-3-pool
      includes:
      - name: common-gpu-consumption
```

As you can see, a capacity pool mixin is created and it defines the
capacities available for each gpu. This is then referenced in four
separate capacity pools, each of which maps to a single physical
GPU.
Then, three device mixins are created called "system-attributes",
"common-gpu-attributes", and "common-gpu-capacities", which all get included in
the definitons of the actual GPU devices themselves, along with their own
device-specific attributes (and device-specific capacities if there were any). 
A device capacity consumption mixin is also created called "common-gpu-consumption"
which is referenced from all the devices together with a reference to the
capacity pool from which the capacities will be drawn.

With this in place, the scheduler can parse these device definitions and
"flatten" them into a set devices that pull capacity from a set of pools. The
scheduler can use this information to allocate device partitions without
overcommitting the capacity of the physical devices.

### Defining device partitions in terms of consumed capacity in a composite device

A simple example demonstrating how capacity pools can be used
to define multiple, allocatable partitions of a single overarching device can be
seen below.

```yaml
sharedCounters:
- name: gpu-0-pool
  counters:
    memory: 
      value: 40Gi
mixins:
  device:
  - name: gpu-partition-capacity
    capacity:
      memory: 
        value: 10Gi
  deviceCounterConsumption:
  - name: gpu-partition-consumption
    counters:
      memory: 
        value: 10Gi
devices:
- name: gpu-0
  basic:
    capacity:
      memory: 
        value: 40Gi
    consumesCounter:
    - counterSet: gpu-0-pool
      counters:
        memory: 
          value: 40Gi
- name: gpu-0-partition-0
  basic:
    includes:
    - name: gpu-partition-capacity
    consumesCounter:
    - counterSet: gpu-pool-mixin
    - includes:
      - name: gpu-partition-consumption
- name: gpu-0-partition-1
  basic:
    includes:
    - name: gpu-partition-capacity
    consumesCounter:
    - counterSet: gpu-pool-mixin
    - includes:
      - name: gpu-partition-consumption
- name: gpu-0-partition-2
  basic:
    includes:
    - name: gpu-partition-capacity
    consumesCounter:
    - counterSet: gpu-pool-mixin
    - includes:
      - name: gpu-partition-consumption
- name: gpu-0-partition-3
  basic:
    includes:
    - name: gpu-partition-capacity
    consumesCounter:
    - counterSet: gpu-pool-mixin
    - includes:
      - name: gpu-partition-consumption
```

In this example, five devices are defined: a full GPU called "gpu-0" and four
partitions of "gpu-0" called "gpu-0-partion-0", "gpu-0-partion-1",
"gpu-0-partion-2", and "gpu-0-partion-3" respectively. The physical GPU is
represented as a shared capacity pool, from which all five devices draw their
capacity. It also sets up a device mixin for the partitions that defines the
advertised resource availability and a device capacity consumption mixin that
defines the capacity the device draws from the capaity pool. This allows the
definition of each partition to be quite compact.

After all mixins have been "flattened", the scheduler ends up with a list
of devices that reference a set of pools. It must track the available capacity
in all capacity pools as devices gets allocated and deallocated, making sure that
the committed capacity never exceeds what is available.

Allocated resources are tracked in the ResourceClaims and that is where the
scheduler currently gets information about which devices are available. Using
this information, the scheduler can compute the available capacity for each
resource pool. Since each device specifically defines the capacity it needs,
there is no ambiguity about what capacity is being used by an allocated device.

When the scheduler tries to allocate devices for a new ResourceClaim, it will
go through the list of devices and check which of them are available. Since it
knows the free capacity in all capacity pools, it can easily check whether a
specific device can be allocated.

Note that since the scheduler will go through these in order and select the first
(not best) fit, it is recommended that ResourceSlices include devices in
smallest-to-largest order. Otherwise, the largest one will always be allocated.
This may be improved through scoring mechanisms, but that is outside of the scope
of this KEP. It is being tracked in https://github.com/kubernetes/enhancements/issues/4970.

### Defining multi-host devices

An example of a small 4x4 TPU slice with its partitions will look like the
example below. Since the devices in the slice is connected to multiple nodes,
it will typically be the responsibility of a central controller to publish the
ResourceSlice.

```yaml
kind: ResourceSlice
apiVersion: resource.k8s.io/v1beta1
...
spec:
  perDeviceNodeSelection: true
  pool:
    ...
  driver: tpu.dra.example.com
  sharedCounters:
  - name: tpu-pool
    counters:
      tpus-node-1:
        value: "4"
      tpus-node-2:
        value: "4"
      tpus-node-5:
        value: "4"
      tpus-node-6:
        value: "4"
  devices:
  # 4x4 slice
  - name: tpu-4x4-1
    basic:
      nodeSelector:
        nodeSelectorTerms:
        - matchExpressions:
          - key: kubernetes.io/hostname
            operator: IN
            values:
            - node-1
            - node-2
            - node-5
            - node-6
      capacity:
        tpus: "16"
      consumesCounter:
      - counterSet: tpu-pool
        counters:
          tpus-node-1:
            value: "4"
          tpus-node-2:
            value: "4"
          tpus-node-5:
            value: "4"
          tpus-node-6:
            value: "4"
  # 2x4 slices
  - name: tpu-2x4-1
    basic:
      nodeSelector:
        nodeSelectorTerms:
        - matchExpressions:
          - key: kubernetes.io/hostname
            operator: IN
            values:
            - node-1
            - node-2
      capacity:
        tpus: "8"
      consumesCounter:
      - counterSet: tpu-pool
        counters:
          tpus-node-1:
            value: "4"
          tpus-node-2:
            value: "4"
  - name: tpu-2x4-2
    basic:
      nodeSelector:
        nodeSelectorTerms:
        - matchExpressions:
          - key: kubernetes.io/hostname
            operator: IN
            values:
            - node-5
            - node-6
      capacity:
        tpus: "8"
      consumesCounter:
      - counterSet: tpu-pool
        counters:
          tpus-node-5:
            value: "4"
          tpus-node-6:
            value: "4"
  # 2x2 slices
  - name: tpu-2x2-1
    basic:
      nodeName: node-1
      capacity:
        tpus: "4"
      consumesCounter:
      - counterSet: tpu-pool
        counters:
          tpus-node-1:
            value: "4"
  - name: tpu-2x2-2
    basic:
      nodeName: node-2
      capacity:
        tpus: "4"
      consumesCounter:
      - counterSet: tpu-pool
        counters:
          tpus-node-2:
            value: "4"
  - name: tpu-2x2-3
    basic:
      nodeName: node-5
      capacity:
        tpus: "4"
      consumesCounter:
      - counterSet: tpu-pool
        counters:
          tpus-node-5:
            value: "4"
  - name: tpu-2x2-4
    basic:
      nodeName: node-6
      capacity:
        tpus: "4"
      consumesCounter:
      - counterSet: tpu-pool
        counters:
          tpus-node-6:
            value: "4"
```

In the example we defined a single 4x4 slice. That means 16 TPUs and with
4 TPUs per node, the device is available across four nodes. The node selector
on the devices selects the 4 nodes used by this device. In the example it
does with by the `IN` operator on the `kubernetes.io/hostname` key, but this
could also be just a regular selector on a single label set on all nodes.

The `CounterSet` declares the available capacity. It needs to handle the
TPU capacity on each node separately, since the TPUs are not interchangibly
between nodes. The granularity required here depends on which devices are being
made available, but for this example doing capacity per-node is sufficient.
The devices declares the capacity they consume from the pool allowing the
scheduler to determine which devices are still allocatable when some of the
capacity has been used by other allocated devices.

In the typical case, when a multi-host device is requested, the workload would
have a number of pods that equals the number of nodes that make up the device.
These pods will share the device, so they must be set up with a shared
ResourceClaim. When the scheduler attempts to schedule the first pod for the
workload, it will find a device that matches the request and allocate it for the
ResourceClaim. Once the a device has been allocated for the claim, this also
restricts the nodes where future pods using the device can be scheduled. To make
sure that future pods will only be attempted for scheduling on eligible nodes, the
scheduler will use `nodeName` or `nodeSelector` value from the device to determine the
`nodeSelector` field on the `AllocationResult` in the `ResourceClaim`, rather
than the `nodeName` or `nodeSelector` from the `ResourceSlice`. This makes sure
that pods sharing the `ResourceClaim` can not get scheduled on nodes that aren't
part of the device.

#### Multi-host scheduling limitations
The shared `ResourceClaim` and the device node selectors only guarantee that
the pods for the workload will not be scheduled on nodes that are not part of
the multi-host device. However, there is no guarantee that they can get scheduled on
the nodes that make up the device, since that will be subject to any other
constraints (like sufficient CPU and memory) during the scheduling process.

Similarly, it is possible for users to create workloads that references multiple
`ResourceClaim`s. These might reference different multi-host devices which might
have node selectors that are only partially overlapping or not overlapping at
all. In this situation, none or only a subset of the pods might end up being
scheduled.

DRA does not guarantee that all or none of the pods can be scheduled (i.e.
group scheduling), so handling those situations will be up to the user or
higher-level frameworks. For beta we aim to improve the story here,
possibly through integration with Kueue.

### Putting it all together for the MIG use-case

The example in the previous section only lists a **single** "memory" capacity
on "gpu-0" that gets divided evenly across 4 possible sub-partitions. While
this simple use case is certainly supported, it is not representative of the
more complex use-cases we envision the extensions in this KEP to be used for.
Specifically, for partitioning devices across multiple dimensions, such as is
necessary to support the MIG use-case for NVIDIA GPUs described previously.

With MIG, one can think of a GPU as a "bag" of resources that can be pulled
together in a discrete set of ways to construct a MIG device with a particular
"profile". These resources include things like "memory", "number of JPEG
engines", "discrete memory slices on the physical GPU die", etc.

Using the capacity pool to represent the "bag" of capacities, we can define
a set of devices that pull specific capacity from that pool.

For example, a capacity pool for an NVIDIA A100 GPU looks as follows:
```yaml
sharedCounters:
- name: gpu-0-pool
  counters:
    copy-engines:
      value: "7"
    decoders:
      value: "5"
    encoders:
      value: "0"
    jpeg-engines:
      value: "1"
    memory:
      value: 40Gi
    memorySlice0:
      value: "1"
    memorySlice1:
      value: "1"
    memorySlice2:
      value: "1"
    memorySlice3:
      value: "1"
    memorySlice4:
      value: "1"
    memorySlice5:
      value: "1"
    memorySlice6:
      value: "1"
    memorySlice7:
      value: "1"
    multiprocessors:
      value: "98"
    ofa-engines:
      value: "1"
```

Three example devices representing MIG partitions can be defined as follows:
```yaml
sharedCounters:
- name: gpu-0-pool
  ...
deviceMixins:
- name: mig-1g.5gb
  capacity:
    copy-engines:
      value: "1"
    decoders:
      value: "0"
    encoders:
      value: "0"
    jpeg-engines:
      value: "0"
    memory:
      value: 4864Mi
    memorySlice0:
      value: "1"
    multiprocessors:
      value: "14"
    ofa-engines:
      value: "0"
  sharedCounterConsumed:
  - name: copy-engines
    capacity: "1"
  - name: decoders
    capacity: "0"
  - name: encoders
    capacity: "0"
  - name: jpeg-engines
    capacity: "0"
  - name: memory
    capacity: 4864Mi
  - name: memorySlice0
    capacity: "1"
  - name: multiprocessors
    capacity: "14"
  - name: ofa-engines
    capacity: "0"
- name: mig-2g.10gb
  capacity:
    copy-engines:
      value: "2"
    decoders:
      value: "1"
    encoders:
      value: "0"
    jpeg-engines:
      value: "0"
    memory:
      value: 9856Mi
    memorySlice0:
      value: "1"
    memorySlice1:
      value: "1"
    multiprocessors:
      value: "28"
    ofa-engines:
      value: "0"
  sharedCounterConsumed:
  - name: copy-engines
    capacity: "2"
  - name: decoders
    capacity: "1"
  - name: encoders
    capacity: "0"
  - name: jpeg-engines
    capacity: "0"
  - name: memory
    capacity: 9856Mi
  - name: memorySlice0
    capacity: "1"
  - name: memorySlice1
    capacity: "1"
  - name: multiprocessors
    capacity: "28"
  - name: ofa-engines
    capacity: "0"
devices:
- name: gpu-0-mig-1g.5gb-0
  basic:
    attributes:
      ...
    includes:
    - name: mig-1g.5gb
      counterSet: gpu-0-pool
- name: gpu-0-mig-1g.5gb-1
  basic:
    attributes:
      ...
    includes:
    - name: mig-1g.5gb
      counterSet: gpu-0-pool
- name: gpu-0-mig-2g.10gb-0-1
  basic:
    attributes:
      ...
    includes:
    - name: mig-2g.10gb
      counterSet: gpu-0-pool
```

The first two MIG devices can be allocated together because there is enough
capacity across all dimensions in the full GPU's capacity to satisfy both of
them simultaneously. However, neither of the first two MIG devices can be
allocated together with the third one because they compete on the capacity
available in the full GPU across its "memorySlice0" and "memorySlice1"
dimensions respectively.

Note that the use of the two device mixins that represents the two MIG profiles
allows us to define the "shape" of the devices once and use them to define
the actual devices in just a few lines of yaml.

A comprehensive example of the actual MIG partitions that would be created for
a 2 GPU DGXA100 server that ties together the concepts of both mixins and
capacity pools can be seen below.

```yaml
sharedCounters:
- includes:
  - name: mock-nvidia-a100
  name: gpu-0-capacity-pool
- includes:
  - name: mock-nvidia-a100
  name: gpu-1-capacity-pool
devices:
- basic:
    consumesCounter:
    - counterSet: gpu-0-capacity-pool
      includes:
      - name: common-gpu-mock-nvidia-a100-sxm4-40gb-capacity-consumption
      - name: memory-slices-0-7
    includes:
    - name: system-attributes
    - name: common-gpu-mock-nvidia-a100-sxm4-40gb-attributes
    - name: common-gpu-mock-nvidia-a100-sxm4-40gb-capacities
    - name: specific-gpu-0-attributes
    - name: memory-slices-0-7
  name: gpu-0
- basic:
    consumesCounter:
    - counterSet: gpu-0-capacity-pool
      includes:
      - name: common-mig-1g.10gb-mock-nvidia-a100-sxm4-40gb
      - name: memory-slices-0-1
    includes:
    - name: system-attributes
    - name: common-mig-mock-nvidia-a100-sxm4-40gb-attributes
    - name: common-mig-1g.10gb-mock-nvidia-a100-sxm4-40gb
    - name: specific-gpu-0-mig-attributes
    - name: memory-slices-0-1
  name: gpu-0-mig-1g.10gb-0-1
- basic:
    consumesCounter:
    - counterSet: gpu-0-capacity-pool
      includes:
      - name: common-mig-1g.10gb-mock-nvidia-a100-sxm4-40gb
      - name: memory-slices-2-3
    includes:
    - name: system-attributes
    - name: common-mig-mock-nvidia-a100-sxm4-40gb-attributes
    - name: common-mig-1g.10gb-mock-nvidia-a100-sxm4-40gb
    - name: specific-gpu-0-mig-attributes
    - name: memory-slices-2-3
  name: gpu-0-mig-1g.10gb-2-3
- basic:
    consumesCounter:
    - counterSet: gpu-0-capacity-pool
      includes:
      - name: common-mig-1g.10gb-mock-nvidia-a100-sxm4-40gb
      - name: memory-slices-4-5
    includes:
    - name: system-attributes
    - name: common-mig-mock-nvidia-a100-sxm4-40gb-attributes
    - name: common-mig-1g.10gb-mock-nvidia-a100-sxm4-40gb
    - name: specific-gpu-0-mig-attributes
    - name: memory-slices-4-5
  name: gpu-0-mig-1g.10gb-4-5
- basic:
    consumesCounter:
    - counterSet: gpu-0-capacity-pool
      includes:
      - name: common-mig-1g.10gb-mock-nvidia-a100-sxm4-40gb
      - name: memory-slices-6-7
    includes:
    - name: system-attributes
    - name: common-mig-mock-nvidia-a100-sxm4-40gb-attributes
    - name: common-mig-1g.10gb-mock-nvidia-a100-sxm4-40gb
    - name: specific-gpu-0-mig-attributes
    - name: memory-slices-6-7
  name: gpu-0-mig-1g.10gb-6-7
- basic:
    consumesCounter:
    - counterSet: gpu-0-capacity-pool
      includes:
      - name: common-mig-1g.5gb-mock-nvidia-a100-sxm4-40gb
      - name: memory-slices-0
    includes:
    - name: system-attributes
    - name: common-mig-mock-nvidia-a100-sxm4-40gb-attributes
    - name: common-mig-1g.5gb-mock-nvidia-a100-sxm4-40gb
    - name: specific-gpu-0-mig-attributes
    - name: memory-slices-0
  name: gpu-0-mig-1g.5gb-0
- basic:
    consumesCounter:
    - counterSet: gpu-0-capacity-pool
      includes:
      - name: common-mig-1g.5gb-mock-nvidia-a100-sxm4-40gb
      - name: memory-slices-1
    includes:
    - name: system-attributes
    - name: common-mig-mock-nvidia-a100-sxm4-40gb-attributes
    - name: common-mig-1g.5gb-mock-nvidia-a100-sxm4-40gb
    - name: specific-gpu-0-mig-attributes
    - name: memory-slices-1
  name: gpu-0-mig-1g.5gb-1
- basic:
    consumesCounter:
    - counterSet: gpu-0-capacity-pool
      includes:
      - name: common-mig-1g.5gb-mock-nvidia-a100-sxm4-40gb
      - name: memory-slices-2
    includes:
    - name: system-attributes
    - name: common-mig-mock-nvidia-a100-sxm4-40gb-attributes
    - name: common-mig-1g.5gb-mock-nvidia-a100-sxm4-40gb
    - name: specific-gpu-0-mig-attributes
    - name: memory-slices-2
  name: gpu-0-mig-1g.5gb-2
- basic:
    consumesCounter:
    - counterSet: gpu-0-capacity-pool
      includes:
      - name: common-mig-1g.5gb-mock-nvidia-a100-sxm4-40gb
      - name: memory-slices-3
    includes:
    - name: system-attributes
    - name: common-mig-mock-nvidia-a100-sxm4-40gb-attributes
    - name: common-mig-1g.5gb-mock-nvidia-a100-sxm4-40gb
    - name: specific-gpu-0-mig-attributes
    - name: memory-slices-3
  name: gpu-0-mig-1g.5gb-3
- basic:
    consumesCounter:
    - counterSet: gpu-0-capacity-pool
      includes:
      - name: common-mig-1g.5gb-mock-nvidia-a100-sxm4-40gb
      - name: memory-slices-4
    includes:
    - name: system-attributes
    - name: common-mig-mock-nvidia-a100-sxm4-40gb-attributes
    - name: common-mig-1g.5gb-mock-nvidia-a100-sxm4-40gb
    - name: specific-gpu-0-mig-attributes
    - name: memory-slices-4
  name: gpu-0-mig-1g.5gb-4
- basic:
    consumesCounter:
    - counterSet: gpu-0-capacity-pool
      includes:
      - name: common-mig-1g.5gb-mock-nvidia-a100-sxm4-40gb
      - name: memory-slices-5
    includes:
    - name: system-attributes
    - name: common-mig-mock-nvidia-a100-sxm4-40gb-attributes
    - name: common-mig-1g.5gb-mock-nvidia-a100-sxm4-40gb
    - name: specific-gpu-0-mig-attributes
    - name: memory-slices-5
  name: gpu-0-mig-1g.5gb-5
- basic:
    consumesCounter:
    - counterSet: gpu-0-capacity-pool
      includes:
      - name: common-mig-1g.5gb-mock-nvidia-a100-sxm4-40gb
      - name: memory-slices-6
    includes:
    - name: system-attributes
    - name: common-mig-mock-nvidia-a100-sxm4-40gb-attributes
    - name: common-mig-1g.5gb-mock-nvidia-a100-sxm4-40gb
    - name: specific-gpu-0-mig-attributes
    - name: memory-slices-6
  name: gpu-0-mig-1g.5gb-6
- basic:
    consumesCounter:
    - counterSet: gpu-0-capacity-pool
      includes:
      - name: common-mig-1g.5gb-me-mock-nvidia-a100-sxm4-40gb
      - name: memory-slices-0
    includes:
    - name: system-attributes
    - name: common-mig-mock-nvidia-a100-sxm4-40gb-attributes
    - name: common-mig-1g.5gb-me-mock-nvidia-a100-sxm4-40gb
    - name: specific-gpu-0-mig-attributes
    - name: memory-slices-0
  name: gpu-0-mig-1g.5gb-me-0
- basic:
    consumesCounter:
    - counterSet: gpu-0-capacity-pool
      includes:
      - name: common-mig-1g.5gb-me-mock-nvidia-a100-sxm4-40gb
      - name: memory-slices-1
    includes:
    - name: system-attributes
    - name: common-mig-mock-nvidia-a100-sxm4-40gb-attributes
    - name: common-mig-1g.5gb-me-mock-nvidia-a100-sxm4-40gb
    - name: specific-gpu-0-mig-attributes
    - name: memory-slices-1
  name: gpu-0-mig-1g.5gb-me-1
- basic:
    consumesCounter:
    - counterSet: gpu-0-capacity-pool
      includes:
      - name: common-mig-1g.5gb-me-mock-nvidia-a100-sxm4-40gb
      - name: memory-slices-2
    includes:
    - name: system-attributes
    - name: common-mig-mock-nvidia-a100-sxm4-40gb-attributes
    - name: common-mig-1g.5gb-me-mock-nvidia-a100-sxm4-40gb
    - name: specific-gpu-0-mig-attributes
    - name: memory-slices-2
  name: gpu-0-mig-1g.5gb-me-2
- basic:
    consumesCounter:
    - counterSet: gpu-0-capacity-pool
      includes:
      - name: common-mig-1g.5gb-me-mock-nvidia-a100-sxm4-40gb
      - name: memory-slices-3
    includes:
    - name: system-attributes
    - name: common-mig-mock-nvidia-a100-sxm4-40gb-attributes
    - name: common-mig-1g.5gb-me-mock-nvidia-a100-sxm4-40gb
    - name: specific-gpu-0-mig-attributes
    - name: memory-slices-3
  name: gpu-0-mig-1g.5gb-me-3
- basic:
    consumesCounter:
    - counterSet: gpu-0-capacity-pool
      includes:
      - name: common-mig-1g.5gb-me-mock-nvidia-a100-sxm4-40gb
      - name: memory-slices-4
    includes:
    - name: system-attributes
    - name: common-mig-mock-nvidia-a100-sxm4-40gb-attributes
    - name: common-mig-1g.5gb-me-mock-nvidia-a100-sxm4-40gb
    - name: specific-gpu-0-mig-attributes
    - name: memory-slices-4
  name: gpu-0-mig-1g.5gb-me-4
- basic:
    consumesCounter:
    - counterSet: gpu-0-capacity-pool
      includes:
      - name: common-mig-1g.5gb-me-mock-nvidia-a100-sxm4-40gb
      - name: memory-slices-5
    includes:
    - name: system-attributes
    - name: common-mig-mock-nvidia-a100-sxm4-40gb-attributes
    - name: common-mig-1g.5gb-me-mock-nvidia-a100-sxm4-40gb
    - name: specific-gpu-0-mig-attributes
    - name: memory-slices-5
  name: gpu-0-mig-1g.5gb-me-5
- basic:
    consumesCounter:
    - counterSet: gpu-0-capacity-pool
      includes:
      - name: common-mig-1g.5gb-me-mock-nvidia-a100-sxm4-40gb
      - name: memory-slices-6
    includes:
    - name: system-attributes
    - name: common-mig-mock-nvidia-a100-sxm4-40gb-attributes
    - name: common-mig-1g.5gb-me-mock-nvidia-a100-sxm4-40gb
    - name: specific-gpu-0-mig-attributes
    - name: memory-slices-6
  name: gpu-0-mig-1g.5gb-me-6
- basic:
    consumesCounter:
    - counterSet: gpu-0-capacity-pool
      includes:
      - name: common-mig-2g.10gb-mock-nvidia-a100-sxm4-40gb
      - name: memory-slices-0-1
    includes:
    - name: system-attributes
    - name: common-mig-mock-nvidia-a100-sxm4-40gb-attributes
    - name: common-mig-2g.10gb-mock-nvidia-a100-sxm4-40gb
    - name: specific-gpu-0-mig-attributes
    - name: memory-slices-0-1
  name: gpu-0-mig-2g.10gb-0-1
- basic:
    consumesCounter:
    - counterSet: gpu-0-capacity-pool
      includes:
      - name: common-mig-2g.10gb-mock-nvidia-a100-sxm4-40gb
      - name: memory-slices-2-3
    includes:
    - name: system-attributes
    - name: common-mig-mock-nvidia-a100-sxm4-40gb-attributes
    - name: common-mig-2g.10gb-mock-nvidia-a100-sxm4-40gb
    - name: specific-gpu-0-mig-attributes
    - name: memory-slices-2-3
  name: gpu-0-mig-2g.10gb-2-3
- basic:
    consumesCounter:
    - counterSet: gpu-0-capacity-pool
      includes:
      - name: common-mig-2g.10gb-mock-nvidia-a100-sxm4-40gb
      - name: memory-slices-4-5
    includes:
    - name: system-attributes
    - name: common-mig-mock-nvidia-a100-sxm4-40gb-attributes
    - name: common-mig-2g.10gb-mock-nvidia-a100-sxm4-40gb
    - name: specific-gpu-0-mig-attributes
    - name: memory-slices-4-5
  name: gpu-0-mig-2g.10gb-4-5
- basic:
    consumesCounter:
    - counterSet: gpu-0-capacity-pool
      includes:
      - name: common-mig-3g.20gb-mock-nvidia-a100-sxm4-40gb
      - name: memory-slices-0-3
    includes:
    - name: system-attributes
    - name: common-mig-mock-nvidia-a100-sxm4-40gb-attributes
    - name: common-mig-3g.20gb-mock-nvidia-a100-sxm4-40gb
    - name: specific-gpu-0-mig-attributes
    - name: memory-slices-0-3
  name: gpu-0-mig-3g.20gb-0-3
- basic:
    consumesCounter:
    - counterSet: gpu-0-capacity-pool
      includes:
      - name: common-mig-3g.20gb-mock-nvidia-a100-sxm4-40gb
      - name: memory-slices-4-7
    includes:
    - name: system-attributes
    - name: common-mig-mock-nvidia-a100-sxm4-40gb-attributes
    - name: common-mig-3g.20gb-mock-nvidia-a100-sxm4-40gb
    - name: specific-gpu-0-mig-attributes
    - name: memory-slices-4-7
  name: gpu-0-mig-3g.20gb-4-7
- basic:
    consumesCounter:
    - counterSet: gpu-0-capacity-pool
      includes:
      - name: common-mig-4g.20gb-mock-nvidia-a100-sxm4-40gb
      - name: memory-slices-0-3
    includes:
    - name: system-attributes
    - name: common-mig-mock-nvidia-a100-sxm4-40gb-attributes
    - name: common-mig-4g.20gb-mock-nvidia-a100-sxm4-40gb
    - name: specific-gpu-0-mig-attributes
    - name: memory-slices-0-3
  name: gpu-0-mig-4g.20gb-0-3
- basic:
    consumesCounter:
    - counterSet: gpu-0-capacity-pool
      includes:
      - name: common-mig-7g.40gb-mock-nvidia-a100-sxm4-40gb
      - name: memory-slices-0-7
    includes:
    - name: system-attributes
    - name: common-mig-mock-nvidia-a100-sxm4-40gb-attributes
    - name: common-mig-7g.40gb-mock-nvidia-a100-sxm4-40gb
    - name: specific-gpu-0-mig-attributes
    - name: memory-slices-0-7
  name: gpu-0-mig-7g.40gb-0-7
- basic:
    consumesCounter:
    - counterSet: gpu-1-capacity-pool
      includes:
      - name: common-gpu-mock-nvidia-a100-sxm4-40gb-capacity-consumption
      - name: memory-slices-0-7
    includes:
    - name: system-attributes
    - name: common-gpu-mock-nvidia-a100-sxm4-40gb-attributes
    - name: common-gpu-mock-nvidia-a100-sxm4-40gb-capacities
    - name: specific-gpu-1-attributes
    - name: memory-slices-0-7
  name: gpu-1
- basic:
    consumesCounter:
    - counterSet: gpu-1-capacity-pool
      includes:
      - name: common-mig-1g.10gb-mock-nvidia-a100-sxm4-40gb
      - name: memory-slices-0-1
    includes:
    - name: system-attributes
    - name: common-mig-mock-nvidia-a100-sxm4-40gb-attributes
    - name: common-mig-1g.10gb-mock-nvidia-a100-sxm4-40gb
    - name: specific-gpu-1-mig-attributes
    - name: memory-slices-0-1
  name: gpu-1-mig-1g.10gb-0-1
- basic:
    consumesCounter:
    - counterSet: gpu-1-capacity-pool
      includes:
      - name: common-mig-1g.10gb-mock-nvidia-a100-sxm4-40gb
      - name: memory-slices-2-3
    includes:
    - name: system-attributes
    - name: common-mig-mock-nvidia-a100-sxm4-40gb-attributes
    - name: common-mig-1g.10gb-mock-nvidia-a100-sxm4-40gb
    - name: specific-gpu-1-mig-attributes
    - name: memory-slices-2-3
  name: gpu-1-mig-1g.10gb-2-3
- basic:
    consumesCounter:
    - counterSet: gpu-1-capacity-pool
      includes:
      - name: common-mig-1g.10gb-mock-nvidia-a100-sxm4-40gb
      - name: memory-slices-4-5
    includes:
    - name: system-attributes
    - name: common-mig-mock-nvidia-a100-sxm4-40gb-attributes
    - name: common-mig-1g.10gb-mock-nvidia-a100-sxm4-40gb
    - name: specific-gpu-1-mig-attributes
    - name: memory-slices-4-5
  name: gpu-1-mig-1g.10gb-4-5
- basic:
    consumesCounter:
    - counterSet: gpu-1-capacity-pool
      includes:
      - name: common-mig-1g.10gb-mock-nvidia-a100-sxm4-40gb
      - name: memory-slices-6-7
    includes:
    - name: system-attributes
    - name: common-mig-mock-nvidia-a100-sxm4-40gb-attributes
    - name: common-mig-1g.10gb-mock-nvidia-a100-sxm4-40gb
    - name: specific-gpu-1-mig-attributes
    - name: memory-slices-6-7
  name: gpu-1-mig-1g.10gb-6-7
- basic:
    consumesCounter:
    - counterSet: gpu-1-capacity-pool
      includes:
      - name: common-mig-1g.5gb-mock-nvidia-a100-sxm4-40gb
      - name: memory-slices-0
    includes:
    - name: system-attributes
    - name: common-mig-mock-nvidia-a100-sxm4-40gb-attributes
    - name: common-mig-1g.5gb-mock-nvidia-a100-sxm4-40gb
    - name: specific-gpu-1-mig-attributes
    - name: memory-slices-0
  name: gpu-1-mig-1g.5gb-0
- composite:
    consumesCapacity:
    - capacityPool: gpu-1-capacity-pool
      includes:
      - name: common-mig-1g.5gb-mock-nvidia-a100-sxm4-40gb
      - name: memory-slices-1
    includes:
    - name: system-attributes
    - name: common-mig-mock-nvidia-a100-sxm4-40gb-attributes
    - name: common-mig-1g.5gb-mock-nvidia-a100-sxm4-40gb
    - name: specific-gpu-1-mig-attributes
    - name: memory-slices-1
  name: gpu-1-mig-1g.5gb-1
- basic:
    consumesCounter:
    - counterSet: gpu-1-capacity-pool
      includes:
      - name: common-mig-1g.5gb-mock-nvidia-a100-sxm4-40gb
      - name: memory-slices-2
    includes:
    - name: system-attributes
    - name: common-mig-mock-nvidia-a100-sxm4-40gb-attributes
    - name: common-mig-1g.5gb-mock-nvidia-a100-sxm4-40gb
    - name: specific-gpu-1-mig-attributes
    - name: memory-slices-2
  name: gpu-1-mig-1g.5gb-2
- basic:
    consumesCounter:
    - counterSet: gpu-1-capacity-pool
      includes:
      - name: common-mig-1g.5gb-mock-nvidia-a100-sxm4-40gb
      - name: memory-slices-3
    includes:
    - name: system-attributes
    - name: common-mig-mock-nvidia-a100-sxm4-40gb-attributes
    - name: common-mig-1g.5gb-mock-nvidia-a100-sxm4-40gb
    - name: specific-gpu-1-mig-attributes
    - name: memory-slices-3
  name: gpu-1-mig-1g.5gb-3
- basic:
    consumesCounter:
    - counterSet: gpu-1-capacity-pool
      includes:
      - name: common-mig-1g.5gb-mock-nvidia-a100-sxm4-40gb
      - name: memory-slices-4
    includes:
    - name: system-attributes
    - name: common-mig-mock-nvidia-a100-sxm4-40gb-attributes
    - name: common-mig-1g.5gb-mock-nvidia-a100-sxm4-40gb
    - name: specific-gpu-1-mig-attributes
    - name: memory-slices-4
  name: gpu-1-mig-1g.5gb-4
- basic:
    consumesCounter:
    - counterSet: gpu-1-capacity-pool
      includes:
      - name: common-mig-1g.5gb-mock-nvidia-a100-sxm4-40gb
      - name: memory-slices-5
    includes:
    - name: system-attributes
    - name: common-mig-mock-nvidia-a100-sxm4-40gb-attributes
    - name: common-mig-1g.5gb-mock-nvidia-a100-sxm4-40gb
    - name: specific-gpu-1-mig-attributes
    - name: memory-slices-5
  name: gpu-1-mig-1g.5gb-5
- basic:
    consumesCounter:
    - counterSet: gpu-1-capacity-pool
      includes:
      - name: common-mig-1g.5gb-mock-nvidia-a100-sxm4-40gb
      - name: memory-slices-6
    includes:
    - name: system-attributes
    - name: common-mig-mock-nvidia-a100-sxm4-40gb-attributes
    - name: common-mig-1g.5gb-mock-nvidia-a100-sxm4-40gb
    - name: specific-gpu-1-mig-attributes
    - name: memory-slices-6
  name: gpu-1-mig-1g.5gb-6
- basic:
    consumesCounter:
    - counterSet: gpu-1-capacity-pool
      includes:
      - name: common-mig-1g.5gb-me-mock-nvidia-a100-sxm4-40gb
      - name: memory-slices-0
    includes:
    - name: system-attributes
    - name: common-mig-mock-nvidia-a100-sxm4-40gb-attributes
    - name: common-mig-1g.5gb-me-mock-nvidia-a100-sxm4-40gb
    - name: specific-gpu-1-mig-attributes
    - name: memory-slices-0
  name: gpu-1-mig-1g.5gb-me-0
- basic:
    consumesCounter:
    - counterSet: gpu-1-capacity-pool
      includes:
      - name: common-mig-1g.5gb-me-mock-nvidia-a100-sxm4-40gb
      - name: memory-slices-1
    includes:
    - name: system-attributes
    - name: common-mig-mock-nvidia-a100-sxm4-40gb-attributes
    - name: common-mig-1g.5gb-me-mock-nvidia-a100-sxm4-40gb
    - name: specific-gpu-1-mig-attributes
    - name: memory-slices-1
  name: gpu-1-mig-1g.5gb-me-1
- basic:
    consumesCounter:
    - counterSet: gpu-1-capacity-pool
      includes:
      - name: common-mig-1g.5gb-me-mock-nvidia-a100-sxm4-40gb
      - name: memory-slices-2
    includes:
    - name: system-attributes
    - name: common-mig-mock-nvidia-a100-sxm4-40gb-attributes
    - name: common-mig-1g.5gb-me-mock-nvidia-a100-sxm4-40gb
    - name: specific-gpu-1-mig-attributes
    - name: memory-slices-2
  name: gpu-1-mig-1g.5gb-me-2
- basic:
    consumesCounter:
    - counterSet: gpu-1-capacity-pool
      includes:
      - name: common-mig-1g.5gb-me-mock-nvidia-a100-sxm4-40gb
      - name: memory-slices-3
    includes:
    - name: system-attributes
    - name: common-mig-mock-nvidia-a100-sxm4-40gb-attributes
    - name: common-mig-1g.5gb-me-mock-nvidia-a100-sxm4-40gb
    - name: specific-gpu-1-mig-attributes
    - name: memory-slices-3
  name: gpu-1-mig-1g.5gb-me-3
- basic:
    consumesCounter:
    - counterSet: gpu-1-capacity-pool
      includes:
      - name: common-mig-1g.5gb-me-mock-nvidia-a100-sxm4-40gb
      - name: memory-slices-4
    includes:
    - name: system-attributes
    - name: common-mig-mock-nvidia-a100-sxm4-40gb-attributes
    - name: common-mig-1g.5gb-me-mock-nvidia-a100-sxm4-40gb
    - name: specific-gpu-1-mig-attributes
    - name: memory-slices-4
  name: gpu-1-mig-1g.5gb-me-4
- basic:
    consumesCounter:
    - counterSet: gpu-1-capacity-pool
      includes:
      - name: common-mig-1g.5gb-me-mock-nvidia-a100-sxm4-40gb
      - name: memory-slices-5
    includes:
    - name: system-attributes
    - name: common-mig-mock-nvidia-a100-sxm4-40gb-attributes
    - name: common-mig-1g.5gb-me-mock-nvidia-a100-sxm4-40gb
    - name: specific-gpu-1-mig-attributes
    - name: memory-slices-5
  name: gpu-1-mig-1g.5gb-me-5
- basic:
    consumesCounter:
    - counterSet: gpu-1-capacity-pool
      includes:
      - name: common-mig-1g.5gb-me-mock-nvidia-a100-sxm4-40gb
      - name: memory-slices-6
    includes:
    - name: system-attributes
    - name: common-mig-mock-nvidia-a100-sxm4-40gb-attributes
    - name: common-mig-1g.5gb-me-mock-nvidia-a100-sxm4-40gb
    - name: specific-gpu-1-mig-attributes
    - name: memory-slices-6
  name: gpu-1-mig-1g.5gb-me-6
- basic:
    consumesCounter:
    - counterSet: gpu-1-capacity-pool
      includes:
      - name: common-mig-2g.10gb-mock-nvidia-a100-sxm4-40gb
      - name: memory-slices-0-1
    includes:
    - name: system-attributes
    - name: common-mig-mock-nvidia-a100-sxm4-40gb-attributes
    - name: common-mig-2g.10gb-mock-nvidia-a100-sxm4-40gb
    - name: specific-gpu-1-mig-attributes
    - name: memory-slices-0-1
  name: gpu-1-mig-2g.10gb-0-1
- basic:
    consumesCounter:
    - counterSet: gpu-1-capacity-pool
      includes:
      - name: common-mig-2g.10gb-mock-nvidia-a100-sxm4-40gb
      - name: memory-slices-2-3
    includes:
    - name: system-attributes
    - name: common-mig-mock-nvidia-a100-sxm4-40gb-attributes
    - name: common-mig-2g.10gb-mock-nvidia-a100-sxm4-40gb
    - name: specific-gpu-1-mig-attributes
    - name: memory-slices-2-3
  name: gpu-1-mig-2g.10gb-2-3
- basic:
    consumesCounter:
    - counterSet: gpu-1-capacity-pool
      includes:
      - name: common-mig-2g.10gb-mock-nvidia-a100-sxm4-40gb
      - name: memory-slices-4-5
    includes:
    - name: system-attributes
    - name: common-mig-mock-nvidia-a100-sxm4-40gb-attributes
    - name: common-mig-2g.10gb-mock-nvidia-a100-sxm4-40gb
    - name: specific-gpu-1-mig-attributes
    - name: memory-slices-4-5
  name: gpu-1-mig-2g.10gb-4-5
- basic:
    consumesCounter:
    - counterSet: gpu-1-capacity-pool
      includes:
      - name: common-mig-3g.20gb-mock-nvidia-a100-sxm4-40gb
      - name: memory-slices-0-3
    includes:
    - name: system-attributes
    - name: common-mig-mock-nvidia-a100-sxm4-40gb-attributes
    - name: common-mig-3g.20gb-mock-nvidia-a100-sxm4-40gb
    - name: specific-gpu-1-mig-attributes
    - name: memory-slices-0-3
  name: gpu-1-mig-3g.20gb-0-3
- basic:
    consumesCounter:
    - counterSet: gpu-1-capacity-pool
      includes:
      - name: common-mig-3g.20gb-mock-nvidia-a100-sxm4-40gb
      - name: memory-slices-4-7
    includes:
    - name: system-attributes
    - name: common-mig-mock-nvidia-a100-sxm4-40gb-attributes
    - name: common-mig-3g.20gb-mock-nvidia-a100-sxm4-40gb
    - name: specific-gpu-1-mig-attributes
    - name: memory-slices-4-7
  name: gpu-1-mig-3g.20gb-4-7
- basic:
    consumesCounter:
    - counterSet: gpu-1-capacity-pool
      includes:
      - name: common-mig-4g.20gb-mock-nvidia-a100-sxm4-40gb
      - name: memory-slices-0-3
    includes:
    - name: system-attributes
    - name: common-mig-mock-nvidia-a100-sxm4-40gb-attributes
    - name: common-mig-4g.20gb-mock-nvidia-a100-sxm4-40gb
    - name: specific-gpu-1-mig-attributes
    - name: memory-slices-0-3
  name: gpu-1-mig-4g.20gb-0-3
- basic:
    consumesCounter:
    - counterSet: gpu-1-capacity-pool
      includes:
      - name: common-mig-7g.40gb-mock-nvidia-a100-sxm4-40gb
      - name: memory-slices-0-7
    includes:
    - name: system-attributes
    - name: common-mig-mock-nvidia-a100-sxm4-40gb-attributes
    - name: common-mig-7g.40gb-mock-nvidia-a100-sxm4-40gb
    - name: specific-gpu-1-mig-attributes
    - name: memory-slices-0-7
  name: gpu-1-mig-7g.40gb-0-7
mixins:
  counterSet:
  - counters:
      copy-engines:
        quantity: "7"
      decoders:
        quantity: "5"
      encoders:
        quantity: "0"
      jpeg-engines:
        quantity: "1"
      memory:
        quantity: 40192Mi
      memorySlice0:
        quantity: "1"
      memorySlice1:
        quantity: "1"
      memorySlice2:
        quantity: "1"
      memorySlice3:
        quantity: "1"
      memorySlice4:
        quantity: "1"
      memorySlice5:
        quantity: "1"
      memorySlice6:
        quantity: "1"
      memorySlice7:
        quantity: "1"
      multiprocessors:
        quantity: "98"
      ofa-engines:
        quantity: "1"
    name: mock-nvidia-a100
  device:
  - basic:
      attributes:
        architecture:
          string: Ampere
        brand:
          string: Nvidia
        cudaComputeCapability:
          string: "8.0"
        productName:
          string: Mock NVIDIA A100-SXM4-40GB
        type:
          string: gpu
    name: common-gpu-mock-nvidia-a100-sxm4-40gb-attributes
  - basic:
      capacity:
        copy-engines:
          quantity: "7"
        decoders:
          quantity: "5"
        encoders:
          quantity: "0"
        jpeg-engines:
          quantity: "1"
        memory:
          quantity: 40Gi
        multiprocessors:
          quantity: "98"
        ofa-engines:
          quantity: "1"
    name: common-gpu-mock-nvidia-a100-sxm4-40gb-capacities
  - basic:
      attributes:
        profile:
          string: 1g.10gb
      capacity:
        copy-engines:
          quantity: "1"
        decoders:
          quantity: "1"
        encoders:
          quantity: "0"
        jpeg-engines:
          quantity: "0"
        memory:
          quantity: 9856Mi
        multiprocessors:
          quantity: "14"
        ofa-engines:
          quantity: "0"
    name: common-mig-1g.10gb-mock-nvidia-a100-sxm4-40gb
  - basic:
      attributes:
        profile:
          string: 1g.5gb+me
      capacity:
        copy-engines:
          quantity: "1"
        decoders:
          quantity: "1"
        encoders:
          quantity: "0"
        jpeg-engines:
          quantity: "1"
        memory:
          quantity: 4864Mi
        multiprocessors:
          quantity: "14"
        ofa-engines:
          quantity: "1"
    name: common-mig-1g.5gb-me-mock-nvidia-a100-sxm4-40gb
  - basic:
      attributes:
        profile:
          string: 1g.5gb
      capacity:
        copy-engines:
          quantity: "1"
        decoders:
          quantity: "0"
        encoders:
          quantity: "0"
        jpeg-engines:
          quantity: "0"
        memory:
          quantity: 4864Mi
        multiprocessors:
          quantity: "14"
        ofa-engines:
          quantity: "0"
    name: common-mig-1g.5gb-mock-nvidia-a100-sxm4-40gb
  - basic:
      attributes:
        profile:
          string: 2g.10gb
      capacity:
        copy-engines:
          quantity: "2"
        decoders:
          quantity: "1"
        encoders:
          quantity: "0"
        jpeg-engines:
          quantity: "0"
        memory:
          quantity: 9856Mi
        multiprocessors:
          quantity: "28"
        ofa-engines:
          quantity: "0"
    name: common-mig-2g.10gb-mock-nvidia-a100-sxm4-40gb
  - basic:
      attributes:
        profile:
          string: 3g.20gb
      capacity:
        copy-engines:
          quantity: "3"
        decoders:
          quantity: "2"
        encoders:
          quantity: "0"
        jpeg-engines:
          quantity: "0"
        memory:
          quantity: 19968Mi
        multiprocessors:
          quantity: "42"
        ofa-engines:
          quantity: "0"
    name: common-mig-3g.20gb-mock-nvidia-a100-sxm4-40gb
  - basic:
      attributes:
        profile:
          string: 4g.20gb
      capacity:
        copy-engines:
          quantity: "4"
        decoders:
          quantity: "2"
        encoders:
          quantity: "0"
        jpeg-engines:
          quantity: "0"
        memory:
          quantity: 19968Mi
        multiprocessors:
          quantity: "56"
        ofa-engines:
          quantity: "0"
    name: common-mig-4g.20gb-mock-nvidia-a100-sxm4-40gb
  - basic:
      attributes:
        profile:
          string: 7g.40gb
      capacity:
        copy-engines:
          quantity: "7"
        decoders:
          quantity: "5"
        encoders:
          quantity: "0"
        jpeg-engines:
          quantity: "1"
        memory:
          quantity: 40192Mi
        multiprocessors:
          quantity: "98"
        ofa-engines:
          quantity: "1"
    name: common-mig-7g.40gb-mock-nvidia-a100-sxm4-40gb
  - basic:
      attributes:
        architecture:
          string: Ampere
        brand:
          string: Nvidia
        cudaComputeCapability:
          string: "8.0"
        productName:
          string: Mock NVIDIA A100-SXM4-40GB
        type:
          string: mig
    name: common-mig-mock-nvidia-a100-sxm4-40gb-attributes
  - basic:
      capacity:
        memorySlice0:
          quantity: "1"
    name: memory-slices-0
  - basic:
      capacity:
        memorySlice0:
          quantity: "1"
        memorySlice1:
          quantity: "1"
    name: memory-slices-0-1
  - basic:
      capacity:
        memorySlice0:
          quantity: "1"
        memorySlice1:
          quantity: "1"
        memorySlice2:
          quantity: "1"
        memorySlice3:
          quantity: "1"
    name: memory-slices-0-3
  - basic:
      capacity:
        memorySlice0:
          quantity: "1"
        memorySlice1:
          quantity: "1"
        memorySlice2:
          quantity: "1"
        memorySlice3:
          quantity: "1"
        memorySlice4:
          quantity: "1"
        memorySlice5:
          quantity: "1"
        memorySlice6:
          quantity: "1"
        memorySlice7:
          quantity: "1"
    name: memory-slices-0-7
  - basic:
      capacity:
        memorySlice1:
          quantity: "1"
    name: memory-slices-1
  - basic:
      capacity:
        memorySlice2:
          quantity: "1"
    name: memory-slices-2
  - basic:
      capacity:
        memorySlice2:
          quantity: "1"
        memorySlice3:
          quantity: "1"
    name: memory-slices-2-3
  - basic:
      capacity:
        memorySlice3:
          quantity: "1"
    name: memory-slices-3
  - basic:
      capacity:
        memorySlice4:
          quantity: "1"
    name: memory-slices-4
  - basic:
      capacity:
        memorySlice4:
          quantity: "1"
        memorySlice5:
          quantity: "1"
    name: memory-slices-4-5
  - basic:
      capacity:
        memorySlice4:
          quantity: "1"
        memorySlice5:
          quantity: "1"
        memorySlice6:
          quantity: "1"
        memorySlice7:
          quantity: "1"
    name: memory-slices-4-7
  - basic:
      capacity:
        memorySlice5:
          quantity: "1"
    name: memory-slices-5
  - basic:
      capacity:
        memorySlice6:
          quantity: "1"
    name: memory-slices-6
  - basic:
      capacity:
        memorySlice6:
          quantity: "1"
        memorySlice7:
          quantity: "1"
    name: memory-slices-6-7
  - basic:
      attributes:
        index:
          int: 0
        minor:
          int: 0
        uuid:
          string: GPU-f27658d7-1427-4974-9dfe-45cc9ca77b34
    name: specific-gpu-0-attributes
  - basic:
      attributes:
        parentIndex:
          int: 0
        parentMinor:
          int: 0
        parentUUID:
          string: GPU-f27658d7-1427-4974-9dfe-45cc9ca77b34
    name: specific-gpu-0-mig-attributes
  - basic:
      attributes:
        index:
          int: 1
        minor:
          int: 1
        uuid:
          string: GPU-e43c1138-8af2-4cfe-af1c-988dbd476754
    name: specific-gpu-1-attributes
  - basic:
      attributes:
        parentIndex:
          int: 1
        parentMinor:
          int: 1
        parentUUID:
          string: GPU-e43c1138-8af2-4cfe-af1c-988dbd476754
    name: specific-gpu-1-mig-attributes
  - basic:
      attributes:
        cudaDriverVersion:
          version: "12.4"
        driverVersion:
          version: 550.54.15
    name: system-attributes
  deviceCounterConsumption:
  - counter:
      memorySlice1:
        quantity: "1"
    name: memory-slices-1
  - counters:
      copy-engines:
        quantity: "4"
      decoders:
        quantity: "2"
      encoders:
        quantity: "0"
      jpeg-engines:
        quantity: "0"
      memory:
        quantity: 19968Mi
      multiprocessors:
        quantity: "56"
      ofa-engines:
        quantity: "0"
    name: common-mig-4g.20gb-mock-nvidia-a100-sxm4-40gb
  - counters:
      copy-engines:
        quantity: "7"
      decoders:
        quantity: "5"
      encoders:
        quantity: "0"
      jpeg-engines:
        quantity: "1"
      memory:
        quantity: 40Gi
      multiprocessors:
        quantity: "98"
      ofa-engines:
        quantity: "1"
    name: common-gpu-mock-nvidia-a100-sxm4-40gb-capacity-consumption
  - counters:
      copy-engines:
        quantity: "2"
      decoders:
        quantity: "1"
      encoders:
        quantity: "0"
      jpeg-engines:
        quantity: "0"
      memory:
        quantity: 9856Mi
      multiprocessors:
        quantity: "28"
      ofa-engines:
        quantity: "0"
    name: common-mig-2g.10gb-mock-nvidia-a100-sxm4-40gb
  - counters:
      memorySlice0:
        quantity: "1"
      memorySlice1:
        quantity: "1"
    name: memory-slices-0-1
  - counters:
      copy-engines:
        quantity: "3"
      decoders:
        quantity: "2"
      encoders:
        quantity: "0"
      jpeg-engines:
        quantity: "0"
      memory:
        quantity: 19968Mi
      multiprocessors:
        quantity: "42"
      ofa-engines:
        quantity: "0"
    name: common-mig-3g.20gb-mock-nvidia-a100-sxm4-40gb
  - counters:
      copy-engines:
        quantity: "1"
      decoders:
        quantity: "1"
      encoders:
        quantity: "0"
      jpeg-engines:
        quantity: "0"
      memory:
        quantity: 9856Mi
      multiprocessors:
        quantity: "14"
      ofa-engines:
        quantity: "0"
    name: common-mig-1g.10gb-mock-nvidia-a100-sxm4-40gb
  - counters:
      memorySlice0:
        quantity: "1"
    name: memory-slices-0
  - counters:
      memorySlice3:
        quantity: "1"
    name: memory-slices-3
  - counters:
      memorySlice4:
        quantity: "1"
    name: memory-slices-4
  - counters:
      memorySlice5:
        quantity: "1"
    name: memory-slices-5
  - counters:
      copy-engines:
        quantity: "1"
      decoders:
        quantity: "0"
      encoders:
        quantity: "0"
      jpeg-engines:
        quantity: "0"
      memory:
        quantity: 4864Mi
      multiprocessors:
        quantity: "14"
      ofa-engines:
        quantity: "0"
    name: common-mig-1g.5gb-mock-nvidia-a100-sxm4-40gb
  - counters:
      memorySlice0:
        quantity: "1"
      memorySlice1:
        quantity: "1"
      memorySlice2:
        quantity: "1"
      memorySlice3:
        quantity: "1"
    name: memory-slices-0-3
  - counters:
      memorySlice4:
        quantity: "1"
      memorySlice5:
        quantity: "1"
      memorySlice6:
        quantity: "1"
      memorySlice7:
        quantity: "1"
    name: memory-slices-4-7
  - counters:
      copy-engines:
        quantity: "7"
      decoders:
        quantity: "5"
      encoders:
        quantity: "0"
      jpeg-engines:
        quantity: "1"
      memory:
        quantity: 40192Mi
      multiprocessors:
        quantity: "98"
      ofa-engines:
        quantity: "1"
    name: common-mig-7g.40gb-mock-nvidia-a100-sxm4-40gb
  - counters:
      memorySlice0:
        quantity: "1"
      memorySlice1:
        quantity: "1"
      memorySlice2:
        quantity: "1"
      memorySlice3:
        quantity: "1"
      memorySlice4:
        quantity: "1"
      memorySlice5:
        quantity: "1"
      memorySlice6:
        quantity: "1"
      memorySlice7:
        quantity: "1"
    name: memory-slices-0-7
  - counters:
      copy-engines:
        quantity: "1"
      decoders:
        quantity: "1"
      encoders:
        quantity: "0"
      jpeg-engines:
        quantity: "1"
      memory:
        quantity: 4864Mi
      multiprocessors:
        quantity: "14"
      ofa-engines:
        quantity: "1"
    name: common-mig-1g.5gb-me-mock-nvidia-a100-sxm4-40gb
  - counters:
      memorySlice6:
        quantity: "1"
      memorySlice7:
        quantity: "1"
    name: memory-slices-6-7
  - counters:
      memorySlice2:
        quantity: "1"
    name: memory-slices-2
  - counters:
      memorySlice6:
        quantity: "1"
    name: memory-slices-6
  - counters:
      memorySlice2:
        quantity: "1"
      memorySlice3:
        quantity: "1"
    name: memory-slices-2-3
  - counters:
      memorySlice4:
        quantity: "1"
      memorySlice5:
        quantity: "1"
    name: memory-slices-4-5
```

The flattened version of this example can be found
[here](https://gist.github.com/mortent/ddda505e2499c872549fa831dd2459c4).

### Using DRA for the multi-host use-case

In order to allocate a 2x4 TPU slice using the ResourceSlice
[shown above](#defining-multi-host-devices), a ResourceClaim like the
following can be used:

```yaml
apiVersion: resource.k8s.io/v1beta1
kind: ResourceClaim
metadata:
  name: tpu-consumer-resource-claim
spec:
  devices:
    requests:
    - name: tpu-request
      deviceClassName: tpu.google.com
      selectors:
      - cel:
          expression: 'device.capacity["tpu.google.com"].tpus == quantity("8")'
```

This simply requests a device with 8 TPUs. Since there are 4 TPUs per node, this requires
two pods, one for each node. A Deployment can be used to create the necessary number of
pods:

```yaml
apiVersion: apps/v1
kind: Deployment
metadata:
  name: tpu-consumer
spec:
  replicas: 2
  selector:
    matchLabels:
      app: tpu-consumer
  template:
    metadata:
      labels:
        app: tpu-consumer
    spec:
      affinity:
        podAntiAffinity:
          requiredDuringSchedulingIgnoredDuringExecution:
          - weight: 100
            podAffinityTerm:
              labelSelector:
                matchLabels:
                  app: tpu-consumer
              topologyKey: kubernetes.io/hostname
      resourceClaims:
      - name: "tpu"
        resourceClaimName: tpu-consumer-resource-claim
      containers:
      - name: workload
        image: my-app
        command: ["/bin/program"]
        resources:
          claims:
          - name: "tpu"
```

Since the PodSpec references a ResourceClaim rather than a ResourceClaimTemplate, they will
share the ResourceClaim. This will then also restrict the pods to run on the nodes that are
targeted by the node selector on the allocated device. Now, in order to be able to take
advantage of the TPUs that are connected to the two nodes, the pods need to be scheduled
on separate nodes. The antiaffinity stanza in the PodSpec makes sure this happens.

An alternative way to make sure the pods are scheduled on different nodes is to use both
a shared claim for the logical multi-host device and separate single-node per-pod claims. This
will force the pods to end up on separate nodes.


### Test Plan

<!--
[testing-guidelines]: https://git.k8s.io/community/contributors/devel/sig-testing/testing.md
-->

[x] I/we understand the owners of the involved components may require updates to
existing tests to make this code solid enough prior to committing the changes necessary
to implement this enhancement.

##### Prerequisite testing updates

##### Unit tests

<!--
Additionally, for Alpha try to enumerate the core package you will be touching
to implement this enhancement and provide the current unit coverage for those
in the form of:
- <package>: <date> - <current test coverage>
The data can be easily read from:
https://testgrid.k8s.io/sig-testing-canaries#ci-kubernetes-coverage-unit

This can inform certain test coverage improvements that we want to do before
extending the production code to implement this enhancement.
- `<package>`: `<date>` - `<test coverage>`
-->

Start of v1.32 development cycle (v1.32.0-alpha.1-178-gd9c46d8ecb1):

- `k8s.io/dynamic-resource-allocation/cel`: 88.8%
- `k8s.io/dynamic-resource-allocation/structured`: 82.7%
- `k8s.io/kubernetes/pkg/controller/resourceclaim`: 70.0%
- `k8s.io/kubernetes/pkg/scheduler/framework/plugins/dynamicresources`: 72.9%

We also plan to add unit tests to verify that the theoretical maximum size
of the ResourceSlice resource remains within the size limitations of etcd. As
the resource has become more complex with additional fields, it has become
harder to do simple back-of-the-envelope calculations.

##### Integration tests

<!--
This question should be filled when targeting a release.
For Alpha, describe what tests will be added to ensure proper quality of the enhancement.

For Beta and GA, add links to added tests together with links to k8s-triage for those tests:
https://storage.googleapis.com/k8s-triage/index.html
-->

The existing [integration tests for kube-scheduler which measure
performance](https://github.com/kubernetes/kubernetes/tree/master/test/integration/scheduler_perf#readme)
will be extended to cover the overheaad of running the additional logic to
support the features in this KEP. These also serve as [correctness
tests](https://github.com/kubernetes/kubernetes/commit/cecebe8ea2feee856bc7a62f4c16711ee8a5f5d9)
as part of the normal Kubernetes "integration" jobs which cover [the dynamic
resource
controller](https://github.com/kubernetes/kubernetes/blob/294bde0079a0d56099cf8b8cf558e3ae7230de12/test/integration/scheduler_perf/util.go#L135-L139).

##### e2e tests

<!--
This question should be filled when targeting a release.
For Alpha, describe what tests will be added to ensure proper quality of the enhancement.

For Beta and GA, add links to added tests together with links to k8s-triage for those tests:
https://storage.googleapis.com/k8s-triage/index.html

We expect no non-infra related flakes in the last month as a GA graduation criteria.
-->

End-to-end testing depends on a working resource driver and a container runtime
with CDI support. A [test
driver](https://github.com/kubernetes/kubernetes/tree/master/test/e2e/dra/test-driver)
was developed as part of the overall DRA development effort. We will extend
this test driver to enable support for `PartitonableDevice`s and add tests to
ensure they are handled by the scheduler as described in this KEP.

### Graduation Criteria

#### Alpha

- Feature implemented behind a feature flag
- Initial e2e tests completed and enabled

#### Beta

- Gather feedback
- Additional tests are in Testgrid and linked in KEP
- Define the alignment and possible integration with Kueue
- Improve the story for group scheduling

#### GA

- 3 examples of vendors making use of the extensions proposed in this KEP
- Scalability tests that mirror real-world usage as determined by user feedback
- Allowing time for feedback

### Upgrade / Downgrade Strategy

The usual Kubernetes upgrade and downgrade strategy applies for in-tree
components. Vendors must take care that upgrades and downgrades work with the
drivers that they provide to customers.

### Version Skew Strategy

All of the API extensions proposed in this KEP are embedded under the
`BasicDevice` type which is a one-of inside of the existing `Device`
type. The new function will be offered through the newly added fields under `BasicDevice`.
The kube-scheduler is expected to match the kube-apiserver minor version, 
but may be up to one minor version older (to allow live upgrades).
In the release it's been added, the feature will be disabled by default and not recognized by other components. 
Whoever enabled the feature manually would take the risk of component like kube-scheduler being old and not recognize 
the fields. After one releases, it should work perfectly.


Since all API extensions are embedded in this way, there is no risk for
version skew downgrades because these devices will never have existed in older
clusters. For upgrades, we will need to be sure not to directly extend this new
device type with new fields, but rather introduce a new one-of if more /
alternate functionality is needed in the future.

## Production Readiness Review Questionnaire

### Feature Enablement and Rollback

###### How can this feature be enabled / disabled in a live cluster?

- [x] Feature gate (also fill in values in `kep.yaml`)
  - Feature gate name: DRAPartitionableDevices
  - Components depending on the feature gate:
    - kube-apiserver
    - kube-scheduler

###### Does enabling the feature change any default behavior?

No

###### Can the feature be disabled once it has been enabled (i.e. can we roll back the enablement)?

Yes. Applications that were already deployed and are running will continue to
work. They will also continue to work when restarted because the CDI devices that
have been prepared for them won't change across the restart.

The DRA driver itself should also be able to survive a rollback, so long as it
has been written to advertise any partitions as "pre-partitioned" devices. It
will just lose the ability to set up new partitions dynamically.

###### What happens if we reenable the feature if it was previously rolled back?

The scheduler may lose track of what devices it has allocated to what pods. Any
pods that had previously allocated devices with the feature enabled will need
to be deleted to ensure they are freed back to their corresponding driver  and
the accounting for them is updated in the scheduler.

###### Are there any tests for feature enablement/disablement?

Objects with the API additions introduced in the KEP are never written by
Kubernetes components themselves. They are written by 3rd-party drivers.
However, the scheduler does consume these objects and track information from
them in order to make scheduling decisions.

Unit tests in will be written in the scheduler to verify that enabling /
disabling of the DRAPartitionableDevices feature gate is non-disruptive to the
scheduler.

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
Will be considered for beta.

###### What specific metrics should inform a rollback?

<!--
What signals should users be paying attention to when the feature is young
that might indicate a serious problem?
-->
Will be considered for beta.

###### Were upgrade and rollback tested? Was the upgrade->downgrade->upgrade path tested?

<!--
Describe manual testing that was done and the outcomes.
Longer term, we may want to require automated upgrade/rollback tests, but we
are missing a bunch of machinery and tooling and can't do that now.
-->
Will be considered for beta.

###### Is the rollout accompanied by any deprecations and/or removals of features, APIs, fields of API types, flags, etc.?

<!--
Even if applying deprecation policies, they may still surprise some users.
-->
Will be considered for beta.

### Monitoring Requirements

<!--
This section must be completed when targeting beta to a release.

For GA, this section is required: approvers should be able to confirm the
previous answers based on experience in the field.
-->
Will be considered for beta.

###### How can an operator determine if the feature is in use by workloads?

<!--
Ideally, this should be a metric. Operations against the Kubernetes API (e.g.,
checking if there are objects with field X set) may be a last resort. Avoid
logs or events for this purpose.
-->
Will be considered for beta.

###### How can someone using this feature know that it is working for their instance?

<!--
For instance, if this is a pod-related feature, it should be possible to determine if the feature is functioning properly
for each individual pod.
Pick one more of these and delete the rest.
Please describe all items visible to end users below with sufficient detail so that they can verify correct enablement
and operation of this feature.
Recall that end users cannot usually observe component logs or access metrics.

- [ ] Events
  - Event Reason: 
- [ ] API .status
  - Condition name: 
  - Other field: 
- [ ] Other (treat as last resort)
  - Details:
-->
Will be considered for beta.

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
Will be considered for beta.

###### What are the SLIs (Service Level Indicators) an operator can use to determine the health of the service?

<!--
Pick one more of these and delete the rest.

- [ ] Metrics
  - Metric name:
  - [Optional] Aggregation method:
  - Components exposing the metric:
- [ ] Other (treat as last resort)
  - Details:
-->
Will be considered for beta.

###### Are there any missing metrics that would be useful to have to improve observability of this feature?

<!--
Describe the metrics themselves and the reasons why they weren't added (e.g., cost,
implementation difficulties, etc.).
-->
Will be considered for beta.

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
Will be considered for beta.

### Scalability

No. The API extensions in this KEP are limited to the existing `ResourceSlice`
object, with no additional requirements to consume this object by additional
components.

###### Will enabling / using this feature result in introducing new API types?

No. The this KEP proposes extensions to an existing type, but not a new type itself.

###### Will enabling / using this feature result in any new calls to the cloud provider?

No.

###### Will enabling / using this feature result in increasing size or count of the existing API objects?

Yes and No. With the extensions proposed in this KEP, individual
`ResourceSlices` have additional fields available to them, thus increasing
their overall signature. However, the purpose of the new `Mixins` abstraction
is to actually *reduce* the footprint of these objects, not increase them. As
such, we expect the size of thes eobjects to actually decrease, not increase,
but it is ultimately up to how 3rd party vendors decide to use them.

###### Will enabling / using this feature result in increasing time taken by any operations covered by existing SLIs/SLOs?

Yes. The time to allocate a device to a claim (and thus schedule the first pod
that references that claim) will be affected.

###### Will enabling / using this feature result in non-negligible increase of resource usage (CPU, RAM, disk, IO, ...) in any components?

No.

###### Can enabling / using this feature result in resource exhaustion of some node resources (PIDs, sockets, inodes, etc.)?

No.

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

Will be considered for beta.

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
Will be considered for beta.

###### What steps should be taken if SLOs are not being met to determine the problem?

Will be considered for beta.

## Implementation History

- Kubernetes 1.32: KEP accepted as "implementable".

## Drawbacks

It adds complexity to the scheduler.

## Alternatives

Many different approaches were considered. Some of them have similarities with
the chosen approach while others are pretty different.

Specifically, the following eight alternative proposals were considered:

**Option 1:** Top-level, single "bag" of resources with no mixins

This approach is similar to what is described in this KEP, except that all
shared resources are provided via a single abstract "bag" of resources rather
than being attached to any particular device. Additionally, there was no notion
of a mixin to help reduce the footprint of a ResourceSlice, but this is
somewhat orthogonal to the proposals ability to support partitionable devices.

This approach was ultimately discarded for four reasons. First, having an
abstract "bag" of resources not tied to any particular device was perceived
as non-intuitive since the resources actually ARE tied to some physical device
that can be allocated. Second, having just a single "bag" of resources meant
that every resource in that bag needed to be prefaced with some notion of the
device that would ultimately be pulling form it. This was overly verbose and
error prone. Third, there was no support for nested partitioning,
which is necessary for supporting the two-level partitioning scheme that MIG
provides, as well as for supporting SRIOV devices. Fourth, there is no ability
to share attributes across devices like we have in this KEP using mixins.

**Option 2:** Top-level, multiple "bags" of resources with no mixins

This approach is similar to option 1, except that multiple, named resource
groups could be created, from which different devices could pull their
resources. This change made it so that each "bag" of resources could be tied to
a particular device (and its partitions) without needing to embed the name of
the device on the resource name itself. This made for a more compact
representation as well as reference to resources from a device.

This approach was rejected for similar reasons as option 1: i.e. having an
abstract "bag" of resources not tied to any particular device was perceived as
non-intuitive and there is no support for nested partitioning or sharing of
attributes.

**Option 3:** Top-level, multiple "bags" of resources with shared attributes

This option is identical to option 2 except that we introduced an abstraction
similar to the mixins described in this KEP. This change allowed us to overcome
1 of the 4 reasons to reject this option, but the other 3 were still valid.

**Option 4:** Top-level, "DeviceShape" defining common capacities and resources for a device

This option is similar to option 3 except that it combines the shared resources
of a device and its common attributes into a top-level abstraction called a
"DeviceShape". As part of this, a "DeviceShape" would explicitly define what
partitions could exist for such a device (without needing to explicitly list
them out in a ResoruceSlice itself). Actual device instances in the
ResourceSlice would then reference a given DeviceShape to define both its own
device as well as any of its partitions.

This approach was ultimately rejected because it was not flexible enough to
express the fact that every device (and its partitions) need some number of
device specific attributes. It also didn't allow for nested partitions or the
ability to create devices that are the "combination" of resources from more
than one source.

All of that said, these "DeviceShapes" paved the way for the mixin idea that is
included in this KEP.

**Option 5:** Top-level, "DeviceShape" with DSL to fill in common capacities and resources for a devic

This option is similar to option 4 except that it generalized the construction
of a device in terms of its "DeviceShape" by introducing a DSL templating
language to "fill in" any device specific attributes when referenced.

This approach was ultimately rejected because of the complexity of maintaining
/ understanding such a templating language embedded in the YAML. It also still
didn't allow nested partitions because of the way the "DeviceShapes" were
constructed.

**Option 6:** Top-level, "PartitionTemplates" defining common capacities and resources for a device or its partitions

This approach is similar to option 4 except that there is not a single device
shape with embedded partitions. Instead top-level devices and their partitions
could each be defined in terms of a "PartitionTemplate" that would be
referenced by a specific device when instantiated. This is essentially what has
now become mixins, with the exception that this option allowed one
"PartitionTemplate" to extend another "PartitionTemplate" whereas mixins must
be standalone. An earlier iteration of mixins also had this "inherit"
capability, but it made the ResourceSlice less readable since you had to
back-track recursively through all such mixins to figure out exactly what set
of attributes/capacities were being added to a device.

This approach was not outright "rejected", but rather has morphed into what is
now being proposed in this KEP, just with different names for the various
abstractions.

**Option 7:**

This approach is essentially option 6, but with all "common" mixins pushed out
to their own object rather than included in each ResoureSlice. This seems
logical to do on the surface, however it becomes complicated when you start to
think about which component owns the life-cycle of these shared objects (and
what happens if they are missing). Its not something we have completely
ruled-out as a future enhancement since without them there is ALOT of redudant
information repeated in each ResourceSlice. However, it is out of scope of the
current KEP and may be considered in a later one.

**Option 8:**

This approach doesn't treat the capacities of the physical device as separate
from the devices published through the API (i.e. as a "bag" of resources).
Instead a hierarchy of devices are defined, with references between then defining
which devices consumes capacities shared with other devices. By walking the directed
graph formed by the references and tracking available capacity for all devices, the
scheduler can determine which devices are allocatable. This design didn't separate
the capacities a device consumes from the capacities it advertises, which makes
tracking available devices difficult and also makes it difficult to model some
types of devices. This approach also used mixin system that is similar to the chosen
design.


See the following for more details on the first seven of the alternate approaches
discussed above:
https://github.com/kubernetes-sigs/wg-device-management/issues/20. The eight option
is described in more detail in https://github.com/kubernetes/enhancements/pull/4874.
