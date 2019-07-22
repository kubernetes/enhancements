---
title: Node Topology Manager
authors:
  - "@ConnorDoyle"
  - "@balajismaniam"
  - "@lmdaly"
owning-sig: sig-node
participating-sigs:
  - sig-node
reviewers:
  - "@vikasc"
  - "@derekwaynecarr"
  - "@jeremyeder"
  - "@RenaudWasTaken"
approvers:
  - "@dawnchen"
  - "@derekwaynecarr"
editor: Louise Daly
creation-date: 2019-01-30
last-updated: 2019-01-30
status: implementable
see-also:
replaces:
superseded-by:
---

# Node Topology Manager

_Authors:_

* @ConnorDoyle - Connor Doyle &lt;connor.p.doyle@intel.com&gt;
* @balajismaniam - Balaji Subramaniam &lt;balaji.subramaniam@intel.com&gt;
* @lmdaly - Louise M. Daly &lt;louise.m.daly@intel.com&gt;

## Table of Contents

<!-- toc -->
- [Overview](#overview)
- [Motivation](#motivation)
  - [Goals](#goals)
  - [Non-Goals](#non-goals)
  - [User Stories](#user-stories)
- [Proposal](#proposal)
  - [Proposed Changes](#proposed-changes)
    - [New Component: Topology Manager](#new-component-topology-manager)
      - [Topology Policy](#topology-policy)
        - [Best-Effort Policy](#best-effort-policy)
        - [Strict Policy](#strict-policy)
      - [New Interfaces](#new-interfaces)
    - [Feature Gate and Kubelet Flags](#feature-gate-and-kubelet-flags)
    - [Changes to Existing Components](#changes-to-existing-components)
- [Graduation Criteria](#graduation-criteria)
  - [Phase 1: Alpha (target v1.15)](#phase-1-alpha-target-v115)
  - [Phase 2: Beta (later versions)](#phase-2-beta-later-versions)
  - [GA (stable)](#ga-stable)
- [Challenges](#challenges)
- [Limitations](#limitations)
- [Alternatives](#alternatives)
- [References](#references)
<!-- /toc -->

# Overview

An increasing number of systems leverage a combination of CPUs and
hardware accelerators to support latency-critical execution and
high-throughput parallel computation. These include workloads in fields
such as telecommunications, scientific computing, machine learning,
financial services and data analytics. Such hybrid systems comprise a
high performance environment.

In order to extract the best performance, optimizations related to CPU
isolation and memory and device locality are required. However, in
Kubernetes, these optimizations are handled by a disjoint set of
components.

This proposal provides a mechanism to coordinate fine-grained hardware
resource assignments for different components in Kubernetes.

# Motivation

Multiple components in the Kubelet make decisions about system
topology-related assignments:

- CPU manager
  - The CPU manager makes decisions about the set of CPUs a container is
allowed to run on. The only implemented policy as of v1.8 is the static
one, which does not change assignments for the lifetime of a container.
- Device manager
  - The device manager makes concrete device assignments to satisfy
container resource requirements. Generally devices are attached to one
peripheral interconnect. If the device manager and the CPU manager are
misaligned, all communication between the CPU and the device can incur
an additional hop over the processor interconnect fabric.
- Container Network Interface (CNI)
  - NICs including SR-IOV Virtual Functions have affinity to one socket,
with measurable performance ramifications.

*Related Issues:*

- [Hardware topology awareness at node level (including NUMA)][k8s-issue-49964]
- [Discover nodes with NUMA architecture][nfd-issue-84]
- [Support VF interrupt binding to specified CPU][sriov-issue-10]
- [Proposal: CPU Affinity and NUMA Topology Awareness][proposal-affinity]

Note that all of these concerns pertain only to multi-socket systems. Correct
behavior requires that the kernel receive accurate topology information from
the underlying hardware (typically via the SLIT table). See section 5.2.16
and 5.2.17 of the
[ACPI Specification](http://www.acpi.info/DOWNLOADS/ACPIspec50.pdf) for more
information.

## Goals

- Arbitrate preferred socket affinity for containers based on input from
  CPU manager and Device Manager.
- Provide an internal interface and pattern to integrate additional
  topology-aware Kubelet components.

## Non-Goals

- _Inter-device connectivity:_ Decide device assignments based on direct
  device interconnects. This issue can be separated from socket
  locality. Inter-device topology can be considered entirely within the
  scope of the Device Manager, after which it can emit possible
  socket affinities. The policy to reach that decision can start simple
  and iterate to include support for arbitrary inter-device graphs.
- _HugePages:_ This proposal assumes that pre-allocated HugePages are
  spread among the available memory nodes in the system. We further assume
  the operating system provides best-effort local page allocation for
  containers (as long as sufficient HugePages are free on the local memory
  node.
- _CNI:_ Changing the Container Networking Interface is out of scope for
  this proposal. However, this design should be extensible enough to
  accommodate network interface locality if the CNI adds support in the
  future. This limitation is potentially mitigated by the possibility to
  use the device plugin API as a stopgap solution for specialized
  networking requirements.

## User Stories

*Story 1: Fast virtualized network functions*

A user asks for a "fast network" and automatically gets all the various
pieces coordinated (hugepages, cpusets, network device) co-located on a
socket.

*Story 2: Accelerated neural network training*

A user asks for an accelerator device and some number of exclusive CPUs
in order to get the best training performance, due to socket-alignment of
the assigned CPUs and devices.

# Proposal

*Main idea: Two phase topology coherence protocol*

Topology affinity is tracked at the container level, similar to devices and
CPU affinity. At pod admission time, a new component called the Topology
Manager collects possible configurations from the Device Manager and the
CPU Manager. The Topology Manager acts as an oracle for local alignment by
those same components when they make concrete resource allocations. We
expect the consulted components to use the inferred QoS class of each
pod in order to prioritize the importance of fulfilling optimal locality.

## Proposed Changes

### New Component: Topology Manager

This proposal is focused on a new component in the Kubelet called the
Topology Manager. The Topology Manager implements the pod admit handler
interface and participates in Kubelet pod admission. When the `Admit()`
function is called, the Topology Manager collects topology hints from other
Kubelet components.

The Topology Manager calculates the best hint with the following algorithm:

1. Iterate over the permutations of hints.
1. Execute the policy.Merge and returns the merged hint.
1. Select the better hint with better preference and narrower SocketAffinity.

If the hints are not compatible, the Topology Manager may choose to
reject the pod. Behavior in this case depends on a new Kubelet configuration
value to choose the topology policy.

The Topology Manager component will be disabled behind a feature gate until
graduation from alpha to beta.

#### Topology Policy

A topology hint indicates a preference for some well-known local resources.
Initially, the only supported reference resource is a mask of CPU socket IDs.
After collecting hints from all resources, the Topology Manager using the
Topology Policy selects the best hint.

As described above, the policy controls on how to merge the permutation of hints,
and if to reject or not the pod when the best hint is not preferred. In the future
we may add additional policies per use-cases and users requirements.

The following sections describe policies algorithms:

##### Best-Effort Policy

The best-effort policy tries to align all resources to one NUMA Node. Theses hints
are marked as preferred.

Here is a sketch:

1. Iterate over the hints in the permutation and set the merged hint to preferred
   if all hints are preferred.
1. Bitwise-and over the hints in each permutation using best-effort policy.Merge.
1. Select the better hint with better preference and narrower SocketAffinity.

The behavior when the best hint is not preferred it to admit the pod.


##### Strict Policy

The strict policy aligns all resources to one NUMA Node. Theses hints
are marked as preferred.

Here is a sketch:

1. Bitwise-or over the hints in each permutation using strict policy.Merge.
1. Set preferred if merged hint SocketAffinity count is one.
1. Select the better hint with better preference and narrower SocketAffinity.

The behavior when the best hint is not preferred it to reject the pod.

#### New Interfaces

```go
package topologymanager

// TopologyManager helps to coordinate local resource alignment
// within the Kubelet.
type Manager interface {
    // Implements pod admit handler interface
    lifecycle.PodAdmitHandler
    // Adds a hint provider to manager to indicate the hint provider
    //wants to be consoluted when making topology hints
    AddHintProvider(HintProvider)
    // Adds pod to Manager for tracking
    AddContainer(pod *v1.Pod, containerID string) error
    // Removes pod from Manager tracking
    RemoveContainer(containerID string) error
    // Interface for storing pod topology hints
    Store
}

// SocketMask is a bitmask-like type denoting a subset of available sockets.
type SocketMask interface {
    Add(sockets ...int) error
    Remove(sockets ...int) error
    And(masks ...SocketMask)
    Or(masks ...SocketMask)
    Clear()
    Fill()
    IsEqual(mask SocketMask) bool
    IsEmpty() bool
    IsSet(socket int) bool
    IsNarrowerThan(mask SocketMask) bool
    String() string
    Count() int
    GetSockets() []int
}
func NewSocketMask(sockets ...int) (SocketMask, error) { ... }

// TopologyHint encodes locality to local resources. Each HintProvider provides
// a list of these hints to the TopoologyManager for each container at pod
// admission time.
type TopologyHint struct {
    SocketAffinity socketmask.SocketMask
    // Preferred is set to true when the SocketMask encodes a preferred
    // allocation for the Container. It is set to false otherwise.
    Preferred bool
}

// HintProvider is implemented by Kubelet components that make
// topology-related resource assignments. The Topology Manager consults each
// hint provider at pod admission time.
type HintProvider interface {
  // Returns hints if this hint provider has a preference; otherwise
  // returns `nil` to indicate "don't care".
  GetTopologyHints(pod v1.Pod, containerName string) []TopologyHint
}

// Store manages state related to the Topology Manager.
type Store interface {
  // GetAffinity returns the preferred affinity as calculated by the
  // TopologyManager across all hint providers for the supplied pod and
  // container.
  GetAffinity(podName string, containerName string) TopologyHint
}

//Policy interface deinfes the merge hint algorithm and if we can admit pod.
type Policy interface {
        // Returns Policy Name
        Name() string
        // Returns Pod Admit Handler Response based on hints and policy type
        CanAdmitPodResult(admit bool) lifecycle.PodAdmitResult
        // Returns merged TopologyHint based on permutation of hints
        Merge(permutation map[string]TopologyHint) TopologyHint
}

```

_Listing: Topology Manager and related interfaces (sketch)._

![topology-manager-components](https://user-images.githubusercontent.com/379372/47447523-8efd2b00-d772-11e8-924d-eea5a5e00037.png)

_Figure: Topology Manager components._

![topology-manager-instantiation](https://user-images.githubusercontent.com/379372/47447526-945a7580-d772-11e8-9761-5213d745e852.png)

_Figure: Topology Manager instantiation and inclusion in pod admit lifecycle._
 
### Feature Gate and Kubelet Flags
 
A new feature gate will be added to enable the Topology Manager feature. This feature gate will be enabled in Kubelet, and will be disabled by default in the Alpha release.  

 * Proposed Feature Gate:  
  `--feature-gate=TopologyManager=true`  
 
 This will be also followed by a Kubelet Flag for the Topology Manager Policy, which is described above. The `preferred` policy will be the default policy.
 
 * Proposed Policy Flag:  
 `--topology-manager-policy=preferred|strict`  
 
### Changes to Existing Components

1. Kubelet consults Topology Manager for pod admission (discussed above.)
1. Add two implementations of Topology Manager interface and a feature gate.
    1. As much Topology Manager functionality as possible is stubbed when the
       feature gate is disabled.
    1. Add a functional Topology Manager that queries hint providers in order
       to compute a preferred socket mask for each container.
1. Add `GetTopologyHints()` method to CPU Manager.
    1. CPU Manager static policy calls `GetAffinity()` method of
       Topology Manager when deciding CPU affinity.
1. Add `GetTopologyHints()` method to Device Manager.
    1. Add Socket ID to Device structure in the device plugin
       interface. Plugins should be able to determine the socket
       when enumerating supported devices. See the protocol diff below.
    1. Device Manager calls `GetAffinity()` method of Topology Manager when
       deciding device allocation.
 
```diff
diff --git a/pkg/kubelet/apis/deviceplugin/v1beta1/api.proto b/pkg/kubelet/apis/deviceplugin/v1beta1/api.proto
index efbd72c133..f86a1a5512 100644
--- a/pkg/kubelet/apis/deviceplugin/v1beta1/api.proto
+++ b/pkg/kubelet/apis/deviceplugin/v1beta1/api.proto
@@ -73,6 +73,10 @@ message ListAndWatchResponse {
 	repeated Device devices = 1;
 }

+message TopologyInfo {
+  optional int32 socketID = 1 [default = -1];
+}
+
 /* E.g:
 * struct Device {
 *    ID: "GPU-fef8089b-4820-abfc-e83e-94318197576e",
@@ -85,6 +89,8 @@ message Device {
 	string ID = 1;
 	// Health of the device, can be healthy or unhealthy, see constants.go
 	string health = 2;
+	// Topology details of the device (optional.)
+	optional TopologyInfo topology = 3;
 }
```

_Listing: Amended device plugin gRPC protocol._

![topology-manager-wiring](https://user-images.githubusercontent.com/379372/47447533-9a505680-d772-11e8-95ca-ef9a8290a46a.png)

_Figure: Topology Manager hint provider registration._

![topology-manager-hints](https://user-images.githubusercontent.com/379372/47447543-a0463780-d772-11e8-8412-8bf4a0571513.png)

_Figure: Topology Manager fetches affinity from hint providers._

# Graduation Criteria

## Phase 1: Alpha (target v1.15)

* Feature gate is disabled by default.
* Alpha-level documentation.
* Unit test coverage.
* Node e2e tests.
* CPU Manager allocation policy takes topology hints into account.
* Device plugin interface includes socket ID.
* Device Manager allocation policy takes topology hints into account.

## Phase 2: Beta (later versions)

* Feature gate is enabled by default.
* Alpha-level documentation.
* Support hugepages alignment.
* User feedback.

## GA (stable)

* *TBD*

# Challenges

* Testing the Topology Manager in a continuous integration environment
  depends on cloud infrastructure to expose multi-node topologies
  to guest virtual machines.
* Implementing the `GetHints()` interface may prove challenging.

# Limitations

* *TBD*

# Alternatives

* [AutoNUMA][numa-challenges]: This kernel feature affects memory
  allocation and thread scheduling, but does not address device locality.

# References

* *TBD*

[k8s-issue-49964]: https://github.com/kubernetes/kubernetes/issues/49964
[nfd-issue-84]: https://github.com/kubernetes-incubator/node-feature-discovery/issues/84
[sriov-issue-10]: https://github.com/hustcat/sriov-cni/issues/10
[proposal-affinity]: https://github.com/kubernetes/community/pull/171
[numa-challenges]: https://queue.acm.org/detail.cfm?id=2852078
