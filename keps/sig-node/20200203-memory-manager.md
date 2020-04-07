---
title: Memory Manager
authors:
  - "@bg-chun"
  - "@c.zukowski"
  - "@alukiano"
owning-sig: sig-node
participating-sigs:
  - sig-node
reviewers:
  - TBD
  - TBD
approvers:
  - TBD
  - TBD
editor: TBD
creation-date: 2020-02-03
last-updated: TBD
status: implementable
see-also:
replaces:
superseded-by:
---

# Memory Manager

_Authors:_

- @bg-chun - Byonggon Chun &lt;bg.chun@samsung.com&gt;
- @c.zukowski - Cezary Zukowski &lt;c.zukowski@samsung.com&gt;
- @alukiano - Artyom Lukianov &lt;alukiano@redhat.com&gt;

## Table of Contents

<!-- toc -->
- [Overview](#overview)
- [Motivation](#motivation)
  - [Related Features](#related-features)
  - [Related issues](#related-issues)
  - [Goals](#goals)
  - [Non-Goals](#non-goals)
  - [User Stories](#user-stories)
    - [Story 1 : Networking Acceleration using DPDK](#story-1--networking-acceleration-using-dpdk)
    - [Story 2 : Database](#story-2--database)
- [Proposal](#proposal)
  - [Proposed Changes](#proposed-changes)
    - [New Component: Memory Manager](#new-component-memory-manager)
      - [The Concept of Memory Map](#the-concept-of-memory-map)
      - [Computing NUMA affinity](#computing-numa-affinity)
      - [The configuration of Memory Manager](#the-configuration-of-memory-manager)
      - [OOM Score Enforcement](#oom-score-enforcement)
      - [CPUSET Enforcement](#cpuset-enforcement)
      - [New Interfaces (Sketch)](#new-interfaces-sketch)
    - [Changes of existing components](#changes-of-existing-components)
      - [Container Manager changes](#container-manager-changes)
      - [Topology Manager changes](#topology-manager-changes)
      - [Internal Container Lifecycle changes](#internal-container-lifecycle-changes)
    - [Feature Gate and Kubelet Flags](#feature-gate-and-kubelet-flags)
- [Test Plan](#test-plan)
- [Graduation Criteria](#graduation-criteria)
  - [Phase 1: Alpha (target v1.19)](#phase-1-alpha-target-v119)
  - [Phase 2: Beta](#phase-2-beta)
  - [GA (stable)](#ga-stable)
- [Implementation History](#implementation-history)
- [Appendix](#appendix)
  - [Limitation of Linux kernel to manage memory](#limitation-of-linux-kernel-to-manage-memory)
  - [How Linux OOM system works in Kubernetes](#how-linux-oom-system-works-in-kubernetes)
  - [Requirements to guarantee performance of DPDK](#requirements-to-guarantee-performance-of-dpdk)
<!-- /toc -->

# Overview

NUMA node topology should be taken into account for containers running performance-sensitive application like DPDK-based packet processing. When the resources are allocated from multiple NUMA nodes in a host machine, an unexpected performance degradation of application may happen due to inter-node communication overhead. To resolve this problem, in Kubernetes 1.16 release, Topology Manager component is introduced to be able to coordinate allocatable resources according to NUMA affinity that satisfies a requirement of the requested resources of containers.

```
+-------------+                +-------------+               +-------------+
|   Kubelet   |                |   Topology  |               |    Hint     |
|             |                |   Manager   |               |  Providers  |
+------+------+                +------+------+               +------+------+
       |                              |                             |
       |           Admit()            |                             |
       +----------------------------->+                             |
       |                              |      GetTopologyHint()      |
       |                              +---------------------------->+
       |                              |                             |
       |                              |            Hints            |
       |                              +<----------------------------+
       |                              |                             |
       |                              +-+----------+                |
       |                              | |          |Hint Merging &  |
       |                              +-+<---------+Decide Admission|
       |      admit or reject Pod     |                             |
       +<-----------------------------+                             |
       |                              |                             |
       +                              +                             +

```
_Figure: The sequence of Pod admission in Kubelet._

Detailedly, to get an information that a resource that container requests is available or not, Topology Manager talks to other managers like CPU Manager and Device Manager, called _hint providers_, which can give the needed information of CPU cores and PCIe peripherals like NIC and GPU, respectively, in each NUMA node. Once collecting all the hints in each NUMA node from the managers, then Topology Manager finally determines which NUMA node's resources in a host can be allocatable for the requested container.

However, the problem raising by this KEP is that there is no hint provider which can give a hint for memory and hugepages resources. That is, Topology Manager could not determine NUMA-aware resource allocation in best way due to absence of memory-related information, which causes inter-node communication overhead among CPU, memory, and PCIe devices and leads overall performance degradation of containers.

To resolve the problem, Memory Manager is proposed in this KEP to provide a way to guarantee NUMA affinity of memory as well as other resources (i.e., CPU and PCIe peripherals) in a procedure of container deployment.

# Motivation

## Related Features

- [Topology Manager][topology-manager] is a feature that collects topology hints from various hint providers (e.g., CPU Manager and Device Manager) to calculate which NUMA node can give a requested amount of resources for a container. With this calculated hints, Topology Manager also refers to node topology policy (i.e. _best-effort, restricted, single-numa-policy_) and NUMA affinity of containers, and then it finally determines if container in a pod can be deployed onto the host machine.

- [CPU Manager][cpu-manager] is a feature that provides a CPU pinning functionality to a container by cgroups cpuset subsystem and also provides topology hint of CPU core availability in NUMA nodes to Topology Manager.

- [Device Manager][device-manager] is a feature that device vendors can advertise their device resources like NIC and GPU devices through their own device-plugins so that kubelet can discover the devices to be used by containers. And also, Device Manager can give a hint of corresponding device's availability in NUMA nodes to Topology Manager.

- [Hugepages][hugepages] is a feature to assign pre-allocated hugepage resources to a container if requested.

- [Node Allocatable Feature][node-allocatable-feature] is a feature that helps to reserve compute resources to isolate whole resources on a host by certain purpose. For example, by this feature, kube-reserved and system-reserved can be defined to reserve resources for kubelet and system (i.e. OS kernel), respectively. In release 1.17, the feature supports following reservable resources: CPU, memory, and ephemeral storage.

## Related issues

- [Hardware topology awareness at node level (including NUMA)][numa-issue]
- [Support Container Isolation of Hugepages][hugepage-issue]
- [Support isolating memory to single NUMA node][memory-issue]

## Goals

- To guarantee isolation of memory and hugepages to single NUMA node for containers belonging to the Guaranteed Pod.

- To provide topology hint of memory and hugepage resources to the Topology Manager, which will allow Topology Manager to allocate memory, hugepages, CPUs, and PCI devices on the same NUMA node.

## Non-Goals

- Supporting cross-NUMA affinity is out of scope.

- Isolating memory & hugepages to a NUMA node for multiple containers is out of scope.

- Updating K8s scheduler and Pod spec is out of scope at this point.

- This proposal only focuses on Linux based systems.

## User Stories

### Story 1 : Networking Acceleration using DPDK

- The system such as real-time trading system and 5G system, which requires networking acceleration, uses DPDK to guarantee low latency of packet processing. DPDK (Data Plane Development Kit) is set of libraries to accelerate packet processing on userspace. DPDK requires dedicated resources (such as exclusive CPU, hugepages, and DPDK-compatible Network Interface Card) and alignment of resources to sigle NUMA node to prevent unexpected performance degradation due to inter-node communication overhead. For this reason, there should be a way to guarantee a resource reservation of memory and hugepage as well as other computing resources (e.g., CPU core and NIC) from a single NUMA node for DPDK-based containers.

### Story 2 : Database

- Databases (e.g., Oracle, PostgreSQL, and MySQL) require lots of memory and hugepages resource intensively to access a massive volume of data. To reduce memory access latency and improve its performance dramatically, all resources (CPU core, memory, hugepages, and I/O devices) should be aligned to a single NUMA node.

# Proposal

## Proposed Changes

### New Component: Memory Manager

Memory Manager is a proposed component of Kubelet which is a new hint provider of Topology Manager especially for memory and hugepage. It provides NUMA affinity to Topology Manager to guarantee NUMA-aware memory consumption for a container which especially belongs to Guaranteed QoS Pod.

When Guaranteed QoS Pod admission is requested from kubelet, Topology Manager asks NUMA affinity of memory and hugepage from Memory Manager. For each container in the Pod, Memory Manager calculates NUMA affinity based on its internal database (i.e. `Memory Map`) which stores a usage of memory and hugepage resource for already-deployed containers. And then, it sends the affinity back to the Topology Manager to help Topology Manager to figure out appropriate NUMA node where adequate resources are available. This calculation is performed for all containers in the Pod and if none of containers is rejected, the Pod is finally admitted to be deployed.

Note:
- Memory Manager only supports single NUMA affinity, because Linux kernel does not have such a mechanism to reserve memory or set memory limit per NUMA node.

In deployment phase, Memory Manager updates its `Memory Map` on an amount of memory and hugepages as much as a container requests. After that, Memory Manager enforces a consumption of memory and hugepage for the container to a specific NUMA node which Topology Manager selected, by seting [CPUSET enforcement](#cpuset-enforcement). Note that containers of Guaranteed Pod of which Memory Manager takes care are guaranteed to consume allocated memory and hugepages on the designated NUMA node while other Pods are deployed later, by utilizing [OOM score enforcement](#oom-score-enforcement) system of Kubelet.

By the help of Memory Manager, Topology Manager with _single-numa-node_ policy can coordinate memory and hugepages along with other compute resources like CPU cores and NIC device to same NUMA node. More details of this component are listed below.

#### The Concept of Memory Map

Memory Manager has internal database (i.e. `Memory Map`) which is used to store a information of reserved memory of deployed containers in a host and calculate NUMA affinity when a container is being deployed. Memory Manager deals with a conventional memory and hugepages of all sizes like 2 MiB and 1 GiB as different types of memory and manages them individually to calculate a topology hint in precise way.

- For example, on a host which has hugepages of 1 GiB and 2 MiB, there are three types of memory: a conventional memory, hugepages-1Gi, and hugepages-2Mi.

At the beginning, Memory Manager initializes a collection of `Memory Table` for each NUMA node and memory type, which composes `Memory Map`. Below shows how `Memory Table` and `Memory Map` are structured in golang style.

```go
type MemoryTable struct {
  TotalMemSize uint64
  PreReserved  uint64
  Allocatable  uint64
  Reserved     uint64
  Free         uint64
}

type MemoryMap map[NUMA_NODE_INDEX]map[MEMORY_TYPES]MemoryTable
```
_Figure: The construction of Memory Table and Memory Map._

- For a brief example, on a two-socket machine with one size of pre-allocated hugepages, Memory Manager will generate four memory tables (two of NUMA nodes * two of memory types including a conventional memory type). (If SNC is enabled, Memory Manager will detect two NUMA nodes from one socket as well.)

`Memory Table` is proposed to represent amount of pre-reserved and allocatable memory for a certain memory type on a certain NUMA node. The table consists of _Pre-reserved Zone_ and _Allocatable Zone_. The following figure shows the structure of `Memory Table`, briefly.

```
+-------------------+------------+--------------+
|                   |     Allocatable Zone      |
+ Pre-reserved Zone +------------+--------------+
|                   |  Reserved  |     Free     |
+-------------------+------------+--------------+
```
_Figure: The construction of Memory Table._

The size of the two zones (_Pre-reserved Zone_ and _Allocatable Zone_) are pre-configured in kubelet configuration by administrator and are immutable values in runtime.
- _Pre-reserved Zone_ indicates the certain amount of memory which is pre-reserved for system such as kernel, OS daemon, and core components of kubernetes like kubelet. For more details how to configure this size, see [The configuration of Memory Manager](#the-configuration-of-memory-manager) section.

- _Allocatable Zone_ indicates the total size of allocatable memory for containers in Guaranteed QoS Pod. The size of this zone is simply the rest of  _Pre-reserved Zone_ from the total size of the Memory Table.

- _Reserved_ section of _Allocatable Zone_ indicates the total size of reserved memory for already-deployed containers in Guaranteed QoS Pods. The Guaranteed memory of certain container is taken from Free memory section.

- _Free_ section of _Allocatable Zone_ indicates a free memory for new containers in a Guaranteed QoS Pod. When a new container is deployed, _Free_ memory is subtracted by the requested memory size of the container. And when a running container is terminated, the requested memory size of the container is returned to _Free_ memory.

#### Computing NUMA affinity

Memory Manager calculates a topology hint for a container in Guaranteed QoS Pod for conventional memory and hugepages of all sizes. The topology hint is well known as _NUMA affinity_ which represents a possible NUMA node that has enough capacity of required memory for all memory types.

To generate a NUMA affinity for a container, Memory Manager refers to `Memory Map` and decides if _Free_ memory of _Allocatable Zone_ for all memory types from each NUMA node is enough to assign for the container. After collecting _NUMA affinity_ for each NUMA node from the perspective of memory, Memory Manager merges them to a topology hint and send it back to Topology Manager.

- Here is a brief example for two socket machine.
  - A Pod requests 2Gi of memory and 1 hugepages-1Gi.
  - Memory Manager found that NUMA node #0 has enough memory and hugepage but NUMA node #1 does not so that the calculated NUMA affinity is NUMA node #0.
  - Memory Manager returns the result as a topology hint with NUMA node #0.

#### The configuration of Memory Manager

The Node Allocatable memory of Node Allocatable Feature is calculated by following formula:
- [Node-Allocatable] = [Node-Capacity] - [Kube-Reserved] - [System-Reserved] - [Eviction-Threshold]

The Node Allocatable is exposed to API server as part of `v1.Node` object and referred by scheduler to select appropriate worker node to bind a Pod.

Configuring Memory Manager is important, Memory Manager assumes that components in kube/system-reserved consume memory from Pre Reserved Zone so that total amount of Pre Reserved Zone for every NUMA node and memory type should be identical to sum of `Kube-Reserved`, `System-Reserved` and `Eviction-Threshold`.

This constraint is proposed to avoid misconfiguration of Memory Manager that may lead node to Pod binding issue.

TBD: example of binding issue

To minimize Pod binding issue, total amount of Allocatable Zone memory should be identical to Node Allocatable memory for each memory type. Memory Manager validates it's configuration flags to ensure equality between total amount of Allocatable Zone memory and Node Allocatable memory for each memory type.

Nevertheless, this indenticality doesn't always guarantee success of Pod binding with single NUMA memory affinity. In some cases, a node may satisfies memory request of a Pod across NUMA nodes, but particular NUMA nodes may not have adequate memory for given container in a Pod. In these cases, Memory Manager will not provide any topology hint to the Topology Manager.

Following feature gate and flag is used to config Memory Manager:
- Feature Gate: --feature-gate=MemoryManager=true
- Kubelet Flag: --pre-reserved-zone=[{numa-node=int, memory-type=string, limit=string}][,][...]

As described in Memory Map section, the size of Pre Reserved Zone and Allocatable Zone are determined by the configuration of Memory Manager. Administrators are expected to have high confidence in the configuration of `--pre-reserved-zone` flag.
As mentioned above, This feature will provide basic validation of configuration for administrators.

NOTE:
- For hugepages, at the moment(v1.17), Node Allocateble feature does not support reserving hugepages, so that Memory Manager will not have Pre Reserved Zone for hugepages. Once Node Allocatable feature will support hugepages reservation(#83541), Pre Reserved Zone for hugepages will work in the same manner of conventional memory.

#### OOM Score Enforcement

The fundamental idea of this feature depends on OOM score enforcement of Kubelet. Kubelet enforces different `oom_score_adj` values for each QoS Class of Pod. Kubelet sets the value of high priority(`-998`) for Guaranteed QoS Class Pod. It protects the containers from global OOM event while the containers consume on designated NUMA node.

Based on `OOM score enforcement`, Memory Manager makes strong assumption that certain amount of reserved memory shall be available on the designated NUMA node.

However, some containers ,which in the non Guaranteed Pod which has the `oom_score_adj` value of low priority(`2~1000`), may consume some reserved memory and makes no more memory is available on the designated NUMA node at some moment. In this case, global OOM event will be occurred while reserved memory is being requested by the proper container then preempted reserved memory will be retrieved by termination of container(s) in non Guaranteed QoS Pod. Consequently, the reserved memory will be allocated to the proper container. Terminated containers are expected to restart depends on its `restartPolicy`.

#### CPUSET Enforcement

Memory Manager restricts memory access of managed containers to a specific NUMA node by using CRI-API. The CRI-API support to use cgroup cpuset subsystem which has `cpuset.cpus` and `cpuset.mems` file interface that constrains cpu and memory usage.

```protobuf
// LinuxContainerResources specifies Linux specific configuration for
// resources.
// TODO: Consider using Resources from opencontainers/runtime-spec/specs-go
// directly.
message LinuxContainerResources {
    // CPU CFS (Completely Fair Scheduler) period. Default: 0 (not specified).
    int64 cpu_period = 1;
    // CPU CFS (Completely Fair Scheduler) quota. Default: 0 (not specified).
    int64 cpu_quota = 2;
    // CPU shares (relative weight vs. other containers). Default: 0 (not specified).
    int64 cpu_shares = 3;
    // Memory limit in bytes. Default: 0 (not specified).
    int64 memory_limit_in_bytes = 4;
    // OOMScoreAdj adjusts the oom-killer score. Default: 0 (not specified).
    int64 oom_score_adj = 5;
    // CpusetCpus constrains the allowed set of logical CPUs. Default: "" (not specified).
    string cpuset_cpus = 6;
    // CpusetMems constrains the allowed set of memory nodes. Default: "" (not specified).
    string cpuset_mems = 7;
    // List of HugepageLimits to limit the HugeTLB usage of container per page size. Default: nil (not specified).
    repeated HugepageLimit hugepage_limits = 8;
}
```
_Figure: LinuxContainerResources of CRI-API

The cpuset subsystem provides the mechanism(`cpuset.mems`) for assigning a set of memory nodes(it equals to NUMA nodes) for linux processes.  In kubelet, CPU Manager already uses `cpuset.cpus` to allocate exclusive logical cpu core to a container. Similarly, Memory Manager uses `cpuset.mems` to restrict containers memory access to specific memory node. 

#### New Interfaces (Sketch)

```go
// k8s.io/kubernetes/pkg/kubelet/cm/memorymanager
package memorymanager

// Manager interface provides methods for Kubelet to manage pod memory and hugepages
type Manager interface {
  // Start is called by Container Manager during Kubelet initialization.
  Start(activePods ActivePodsFunc, sourcesReady config.SourcesReady, podStatusProvider status.PodStatusProvider, containerRuntime runtimeService)
  // Allocate is called to pre-allocate memory resources during Pod admission.
  Allocate(pod *v1.Pod, container *v1.Container) error
  // AddContainer is called between container create and container start
  AddContainer(p *v1.Pod, c *v1.Container, containerID string) error
  // RemoveContainer is called after Kubelet decides to kill or delete a container.
  RemoveContainer(containerID string) error
  // State returns a read-only interface to the internal memory manager state.
  State() state.Reader
  // GetTopologyHints implements the Topology Manager Interface and is
  // consulted to make Topology aware resource alignments and returns
  // map which has key/value as resource name, topology hints.
  GetTopologyHints(pod v1.Pod, container v1.Container) map[string][]topologymanager.TopologyHint
}

type manager struct {
  sync.Mutex

  // reconcilePeriod is the duration between calls to reconcileState.
  reconcilePeriod time.Duration

  // state allows pluggable memory assignment policies while sharing a common
  // representation of state for the system to inspect and reconcile.
  state state.State

  // containerRuntime is the container runtime service interface needed
  // to make UpdateContainerResources() calls against the containers.

  containerRuntime runtimeService
  // activePods is a method for listing active pods on the node
  // so all the containers can be updated in the reconciliation loop.
  activePods ActivePodsFunc

  // podStatusProvider provides a method for obtaining pod statuses
  // and the containerID of their containers
  podStatusProvider status.PodStatusProvider

  // numaNodeInfo is used to get the number of NUMA nodes
  numaNodeInfo cputopology.NUMANodeInfo

  // machineInfo is used to get total memory information of the host
  machineInfo *cadvisorapi.MachineInfo

  // nodeAllocatableReservation is used to calculate allocatable resource
  nodeAllocatableReservation v1.ResourceList

  // sourcesReady provides the readiness of kubelet configuration sources such as apiserver update readiness.
  // sourcesReady is used to remove inactive container from State
  sourcesReady config.SourcesReady

  // stateFileDirectory is the file name where memory manager stores memory map
  stateFileDirectory string
}

// UpdateContainerResources is used to update containers CPUSET configuration for cpuset.mems.
type runtimeService interface {
  UpdateContainerResources(id string, resources *runtimeapi.LinuxContainerResources) error
}

// k8s.io/kubernetes/pkg/kubelet/cm/memorymanager/block
package block

// Block is data structure to represent certain amount of memory
type Block struct
{
  affinity uint64
  memType  string
  size     uint64
}

// k8s.io/kubernetes/pkg/kubelet/cm/memorymanager/algorithmprovider
package algorithmprovider

type SingleNumaAffinity interface {
  // CalculateAffinity calculates topologyHint for container
  CalculateAffinity(pod *v1.Pod, container *v1.Container) map[string][]topologymanager.TopologyHint
  // GetAffinity returns stored topologyHint by Topology Manager
  GetAffinity(pod *v1.Pod, container *v1.Container) map[string][]topologymanager.TopologyHint
  // Allocate reserves memory as requested for container
  Allocate(pod *v1.Pod, container *v1.Container) error
  // Reclaim reclaims freed blocks from removed container
  Reclaim(podUID string, containerName string) error
}

type MemoryTable struct {
  TotalMemSize uint64
  PreReserved  uint64
  Allocatable  uint64
  Reserved     uint64
  Free         uint64
}

//type MemoryMap map[NUMA_NODE_INDEX]map[MEMORY_TYPES]MemoryTable
type MemoryMap map[uint64]map[string]MemoryTable


// k8s.io/kubernetes/pkg/kubelet/cm/memorymanager/state
package state

// ContainerMemoryAssignments stores memory assignments of containers
type ContainerMemoryAssignments map[string]map[string][]Block

// Reader interface used to read current memory assignment state
type Reader interface {
  // GetMachineState returns Memory Map that stored in State
  GetMachineState() MemoryMap
  // GetMemoryBlocks returns memory assignments of container
  GetMemoryBlocks(podUID string, containerName string) []Block
  // GetMemoryAssignments returns ContainerMemoryAssignments
  GetMemoryAssignments() ContainerMemoryAssignments
}

type writer interface {
  // SetMachineState stores MemoryMap in State
  SetMachineState(memoryMap MemoryMap)
  // SetMemoryBlocks stores memory assignments of container
  SetMemoryBlocks(podUID string, containerName string, blocks []Block)
  // SetMemoryAssignments sets ContainerMemoryAssignments by passed parameter
  SetMemoryAssignments(ContainerMemoryAssignments)
  // Delete deletes corresponding Block from ContainerMemoryAssignments
  Delete(podUID string, containerName string)
  // ClearState clears machineState and ContainerMemoryAssignments
  ClearState()
}


// State interface provides methods for tracking and setting memory assignment
type State interface {
  Reader
  writer
}

```

### Changes of existing components

#### Container Manager changes

Container Manager will create Memory Manager and register to Topology Manager as hint provider.

#### Topology Manager changes

Topology Manager will call out Memory Manager to gather topology hint and allocate memory resources during its admit sequence.

#### Internal Container Lifecycle changes

InternalContainerLifecycle will call out Memory Manager on container lifecycle events(Add/Stop) to allocate and reclaim memory resources.

### Feature Gate and Kubelet Flags

A new feature gate will be added to enable the Memory Manager feature. This feature gate will be disabled by default in the Alpha release.

- Proposed Feature Gate:  
  `--feature-gate=MemoryManager=true`

This will be also followed by a Kubelet Flag for the Memory Manager configuration, which is described above in [The configuration of Memory Manager](#the-configuration-of-memory-manager) section.
- Proposed Configuration Flag:  
  `--pre-reserved-zone=[{numa-node=int, memory-type=string, limit=string}][,][...]`
- The example of configuration flag:
   `--pre-reserved-zone= {numa-node=0, memory-type=memory, limit=1Gi}, {numa-node=1, memory-type=memory, limit=1Gi}`
# Test Plan

- Multi-NUMA System Tests
  - E2E test will enable the Memory Manager feature and Topology Manager feature with the single-numa-node policy then the test will deploy a Pod, which has Guaranteed QoS class. A container in the Pod will request exclusive cpu, memory, hugepage. Once Pod deployed, the NUMA node index of cpu assignment, memory, hugepage will be validated together.

# Graduation Criteria

## Phase 1: Alpha (target v1.19)
- Feature gate is disabled by default.
- Alpha implementation of Memory Manager will generate topology hint for single NUMA selection.

## Phase 2: Beta
- E2E test for Memory Manager.
- Feature gate is enabled by default.

## GA (stable)

- User feedback
- TBD

# Implementation History
- TBD

# Appendix

## Limitation of Linux kernel to manage memory

TBD

## How Linux OOM system works in Kubernetes

TBD

## Requirements to guarantee performance of DPDK

TBD

[topology-manager]: https://github.com/kubernetes/enhancements/blob/dcc8c7241513373b606198ab0405634af643c500/keps/sig-node/0035-20190130-topology-manager.md
[cpu-manager]: https://github.com/kubernetes/community/blob/master/contributors/design-proposals/node/cpu-manager.md
[device-manager]: https://github.com/kubernetes/community/blob/master/contributors/design-proposals/resource-management/device-plugin.md
[hugepages]: https://github.com/kubernetes/enhancements/blob/master/keps/sig-node/20190129-hugepages.md
[node-allocatable-feature]: https://github.com/kubernetes/community/blob/master/contributors/design-proposals/node/node-allocatable.md
[numa-issue]: https://github.com/kubernetes/kubernetes/issues/49964
[hugepage-issue]: https://github.com/kubernetes/kubernetes/issues/80716
[memory-issue]: https://github.com/kubernetes/kubernetes/issues/81009

