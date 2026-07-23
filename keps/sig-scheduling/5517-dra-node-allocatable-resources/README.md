# KEP-5517: DRA: Node Allocatable Resources

<!-- toc -->
- [Release Signoff Checklist](#release-signoff-checklist)
- [Summary](#summary)
- [Motivation](#motivation)
  - [Core Problem](#core-problem)
  - [Goals](#goals)
  - [Non-Goals](#non-goals)
- [Proposal](#proposal)
  - [Background](#background)
    - [Kube-Scheduler Background](#kube-scheduler-background)
    - [Node Resource Enforcement Background](#node-resource-enforcement-background)
      - [Cgroup Enforcement](#cgroup-enforcement)
      - [OOM Score Adjustments](#oom-score-adjustments)
  - [User Stories](#user-stories)
  - [Risks and Mitigations](#risks-and-mitigations)
- [Design Details](#design-details)
  - [Conceptual Mapping: Pod Spec Requests and Limits with DRA](#conceptual-mapping-pod-spec-requests-and-limits-with-dra)
  - [API Changes](#api-changes)
    - [Device API Extensions](#device-api-extensions)
    - [Pod API Changes](#pod-api-changes)
      - [Resource Representation Examples](#resource-representation-examples)
    - [API Validation](#api-validation)
  - [Kube-Scheduler Changes](#kube-scheduler-changes)
    - [Resource Calculation](#resource-calculation)
    - [Integration with Pod Level Resources](#integration-with-pod-level-resources)
    - [Handling Shared Claims](#handling-shared-claims)
    - [Multiple Claims per Container](#multiple-claims-per-container)
    - [Unreferenced Claims](#unreferenced-claims)
    - [Preemption](#preemption)
  - [Node Resource Enforcement and Isolation](#node-resource-enforcement-and-isolation)
    - [Scope](#scope)
    - [Key Principles](#key-principles)
    - [Cgroup Enforcement](#cgroup-enforcement-1)
      - [Pod-Level Cgroup Settings](#pod-level-cgroup-settings)
      - [Container-Level Cgroup Settings](#container-level-cgroup-settings)
      - [QoS Class Mismatch Risks](#qos-class-mismatch-risks)
      - [Handling Pod Level Resources](#handling-pod-level-resources)
      - [Handling Missing Limits](#handling-missing-limits)
      - [Handling Kubelet Disabling Quota with Exclusive CPUs](#handling-kubelet-disabling-quota-with-exclusive-cpus)
    - [Enforcement Use Case Walkthroughs](#enforcement-use-case-walkthroughs)
    - [OOM Score Adjustment with DRA](#oom-score-adjustment-with-dra)
    - [Integration with Memory QoS](#integration-with-memory-qos)
      - [Current Memory QOS settings](#current-memory-qos-settings)
      - [Integration with DRA](#integration-with-dra)
    - [Pod Status Updates](#pod-status-updates)
    - [Kubelet Internal Resource States](#kubelet-internal-resource-states)
    - [Integration with In-Place Pod Vertical Scaling](#integration-with-in-place-pod-vertical-scaling)
  - [Kubelet Admission Control](#kubelet-admission-control)
  - [Future Enhancements](#future-enhancements)
    - [Kube-Scheduler Scoring and Resource Quota](#kube-scheduler-scoring-and-resource-quota)
      - [Scoring](#scoring)
      - [Quota](#quota)
    - [Pass Allocation Details from Driver to Kubelet](#pass-allocation-details-from-driver-to-kubelet)
      - [API Changes](#api-changes-1)
      - [Node Cgroup Enforcement](#node-cgroup-enforcement)
  - [Test Plan](#test-plan)
      - [Prerequisite testing updates](#prerequisite-testing-updates)
      - [Unit tests](#unit-tests)
      - [Integration tests](#integration-tests)
      - [e2e tests](#e2e-tests)
  - [Graduation Criteria](#graduation-criteria)
    - [Alpha](#alpha)
    - [Alpha2](#alpha2)
    - [Beta](#beta)
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
  - [DeviceClass API Extension for NodeAllocatableResourceMappings](#deviceclass-api-extension-for-nodeallocatableresourcemappings)
  - [Explicit AccountingPolicy in DeviceClass and PodStatus](#explicit-accountingpolicy-in-deviceclass-and-podstatus)
  - [Alternative Model for pod level resources + DRA](#alternative-model-for-pod-level-resources--dra)
    - [1. Kubelet Cgroup Enforcement](#1-kubelet-cgroup-enforcement)
    - [2. Kube-Scheduler Changes](#2-kube-scheduler-changes)
    - [Enforcement Use Case Walkthroughs with this model](#enforcement-use-case-walkthroughs-with-this-model)
      - [1. pod level Request and Limit + DRA Claim (Single container references claim)](#1-pod-level-request-and-limit--dra-claim-single-container-references-claim)
      - [2. pod level Request and Limit + Fungible DRA Claim (Prioritized List)](#2-pod-level-request-and-limit--fungible-dra-claim-prioritized-list)
- [Infrastructure Needed (Optional)](#infrastructure-needed-optional)
<!-- /toc -->

## Release Signoff Checklist

Items marked with (R) are required *prior to targeting to a milestone / release*.

- [x] (R) Enhancement issue in release milestone, which links to KEP dir in [kubernetes/enhancements] (not the initial KEP PR)
- [x] (R) KEP approvers have approved the KEP status as `implementable`
- [x] (R) Design details are appropriately documented
- [x] (R) Test plan is in place, giving consideration to SIG Architecture and SIG Testing input (including test refactors)
  - [ ] e2e Tests for all Beta API Operations (endpoints)
  - [ ] (R) Ensure GA e2e tests meet requirements for [Conformance Tests](https://github.com/kubernetes/community/blob/master/contributors/devel/sig-architecture/conformance-tests.md)
  - [ ] (R) Minimum Two Week Window for GA e2e tests to prove flake free
- [ ] (R) Graduation criteria is in place
  - [ ] (R) [all GA Endpoints](https://github.com/kubernetes/community/pull/1806) must be hit by [Conformance Tests](https://github.com/kubernetes/community/blob/master/contributors/devel/sig-architecture/conformance-tests.md) within one minor version of promotion to GA
- [x] (R) Production readiness review completed
- [ ] (R) Production readiness review approved
- [ ] "Implementation History" section is up-to-date for milestone
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

This KEP proposes a solution for managing node allocatable resources via Dynamic Resource Allocation (DRA). Node allocatable resources are resources currently reported in `v1.Node` `status.allocatable` that are not extended resources (examples include CPU, Memory, Ephemeral-storage, and Hugepages). Currently, when these node allocatable resources are managed via DRA, there is a fundamental disconnect across the control plane and the Node. In the scheduler, having two independent accounting systems (one for standard resources, one for DRA) managing the same underlying resource leads to resource overcommitment. On the node, the kubelet is completely unaware of DRA allocations, which may result in incorrect QoS class assignment and has many downstream implications. This forces users into fragile workarounds that are incompatible with all use cases.

The proposed solution in this KEP addresses node allocatable resource accounting and enforcement in kube-scheduler and kubelet:
1.  **Kube-Scheduler Accounting**: The standard resource (`NodeResourcesFit` plugin) and DRA (`DynamicResources` plugin) synchronize their accounting, creating a single, authoritative ledger to prevent node overcommitment.
2.  **Kubelet Enforcement**: kubelet natively incorporates node allocatable resource allocations made through DRA `ResourceClaim`s to configure Linux container and pod cgroups and calculate OOM score.

## Motivation

Dynamic Resource Allocation (DRA) provides a powerful framework for managing specialized hardware
resources such as GPUs, FPGAs, and high-performance network interfaces. It also enables fine-grained
management of node allocatable resources like CPU and Memory, for example, through the
[dra-driver-cpu](https://github.com/kubernetes-sigs/dra-driver-cpu). However, when a node allocatable resource
is managed via DRA, while it provides added advantages of being able to specify more detailed
requirements, a fundamental disconnect emerges between the scheduler, the kubelet, and the DRA
framework, which breaks the resource guarantees.

Additionally, specialized resources like accelerators often have implicit dependencies on node allocatable resources
like CPU or Hugepages for the application to interact with it. Currently, users must manually
research and declare these auxiliary node allocatable resource requirements, typically as additional requests
in the PodSpec. This process is error-prone and adds complexity to workload configuration.
Furthermore, there is no existing mechanism to express critical co-location requirements. For
example, there is no way to ensure an accelerator allocated via DRA is NUMA-aligned with the specific
hugepages or CPUs it needs, as the standard and DRA resource models are entirely independent.

### Core Problem

The core problem is that the same underlying physical resource is advertised and consumed through
two parallel, uncoordinated mechanisms.

* **Dual Publication:** A node's total CPU/Memory capacity is advertised in two different places:  
  * Via the Kubelet in the `Node.Status.Allocatable` field.  
  * Via the DRA driver in `ResourceSlice` objects.

* **Dual Consumption:** Pods can consume this CPU capacity in two different ways:  
  * Via pod spec requests (`pod.spec.containers[].resources.requests`,  
    `pod.spec.initcontainers[].resources.requests`), which is considered in the `NodeResourcesFit`
    scheduler plugin to find a Node that fits.  
  * Via `ResourceClaim`, which is considered in the `DynamicResources` scheduler plugin to allocate
    devices.

**Scheduler-Level Resource Oversubscription**: The kubelet is the source of truth for a node's
available resources. The scheduler continuously watches the `Node` object and uses
`Node.Status.Allocatable` to maintain an internal, in-memory cache (`NodeInfo`) of each node's
capacity. This cache is the baseline for all its scheduling decisions, ensuring it does not place
more pods on a node than the node reports it can handle.

It is completely blind to the fact that the DRA (like CPU `ResourceClaim`) draws from the same
physical resource as a standard request. This gap leads to the scheduler overcommitting a node's CPU
resources by scheduling more pods than the node resource capacity.

**Kubelet-Level Guarantee Failure:** The kubelet is the component that enforces resource guarantees on the node. It configures Linux cgroups, calculates Out-Of-Memory (OOM) score adjustments, and makes critical lifecycle decisions like eviction based *only* on standard `pod.Spec` requests and limits. Because Kubelet is unaware of resources allocated via DRA, workloads suffer from an **Enforcement Gap**:
-   Even if the scheduler correctly reserves capacity for both standard and DRA requests on a node, the container remains hard-restricted by the Kubelet's Linux cgroups to its standard Spec bounds. For example, if a container requests 2 CPU in its Spec and references a claim for 5 CPU, the container runtime applies a cgroup CPU quota of only 2 CPU. If the application attempts to consume the 5 CPU burst allocated via DRA, it will be hard-throttled by the kernel.
-   If a workload relies on memory provided via a DRA claim but its standard Spec memory limit is lower:
    -   The kernel will terminate the container when its usage exceeds the standard memory limit.
    -   Kubelet sets a higher OOM score based strictly on the smaller standard memory request, making the workload a prime target for the kernel OOM kill during host memory exhaustion.

Current workarounds for DRA-managed node allocatable resources (like
[CPU DRA driver](https://github.com/kubernetes-sigs/dra-driver-cpu)) force users to duplicate
resource requests in both the `ResourceClaim` and the standard `pod.spec.containers[].resources`.
However, this approach is fragile, error-prone, and difficult to manage, especially for complex pods
with shared resource claims. It is also incompatible with advanced DRA features like
[Prioritized Lists](https://github.com/kubernetes/enhancements/blob/master/keps/sig-scheduling/4816-dra-prioritized-list/README.md)

This KEP proposes to solve this problem by creating a single, unified resource model that spans the
entire control plane, from the scheduler to the kubelet. The goal is not just to fix an accounting
issue in the scheduler, but to provide a complete, native way for Kubernetes to handle core
resources that are backed by DRA.

### Goals

* To create a unified accounting model within the kube-scheduler that prevents overcommitment of core
  resources (like CPU) when they are allocated via both standard `pod.spec` requests and DRA
  `ResourceClaims`.
* To ensure the solution is compatible with different ways node allocatable resources can be represented and
  allocated within DRA, including as individual devices, consumable capacities
  ([KEP-5075](https://github.com/kubernetes/enhancements/issues/5075)), and partitionable devices
  ([KEP-4815](https://github.com/kubernetes/enhancements/issues/4815))
* To enable specialized devices, such as accelerators, to declare any auxiliary node allocatable resource
  requirements (e.g., CPU, Memory) they depend on for their operation.
* To natively integrate DRA node allocatable resource allocations into Kubelet cgroup enforcement.
* To maintain backward compatibility with existing workloads and ecosystem tools that rely on
  `node.status.allocatable` and the scheduler's view of node resource utilization.

### Non-Goals

* To move all resource management logic into the DRA driver. The Kubelet will remain the primary agent
  for cgroup management and QoS enforcement, ensuring that the benefits of its existing stability and
  lifecycle management features are preserved.  
* To replace the standard `pod.spec.containers.resources` API for requesting node allocatable resources. This KEP
  aims to enhance the system by adding a clear path for node allocatable resource requests via DRA while ensuring
  it works coherently with the existing PodSpec-based requests.
* Modifying Kubelet's core QoS class classification logic is a non-goal
  for this KEP. QoS will still be based strictly on standard Spec
  requests and limits.


## Proposal

This KEP introduces a unified accounting and enforcement model within kube-scheduler and the Kubelet to integrate 
node allocatable resources managed by Dynamic Resource Allocation (DRA) with standard resource tracking. By bridging 
the gap between `pod.spec.resources` and DRA `ResourceClaim` allocations, we can achieve consistent resource
accounting and prevent node overcommitment.

### Background

To understand the proposed solution, it is essential to first understand how the control plane and the node currently manage standard resource requests and DRA ResourceClaims.

#### Kube-Scheduler Background

The Kubernetes scheduler is built on a plugin-based framework that executes a series of stages to place
a pod. This KEP is primarily concerned with the interaction between `NodeResourcesFit` and
`DynamicResource` plugins at the `PreFilter`, `Filter`, and `Bind` stages of the
[scheduling framework](https://kubernetes.io/docs/concepts/scheduling-eviction/scheduling-framework/).

###### Standard Resource Accounting

The Kubelet is the source of truth for a node's available resources. It inspects the machine's total
capacity, subtracts resources reserved for the operating system (`--system-reserved`) and Kubernetes
system daemons (`--kube-reserved`), and reports the result in the `Node.Status.Allocatable` field. The
scheduler continuously watches for updates to this field and uses it to maintain its internal, in-memory
cache (`NodeInfo`) of each node's capacity. This cache is the baseline for all its scheduling decisions.

**Kube-Scheduler Resource Accounting**  

* The scheduler maintains an in-memory `NodeInfo` object for each node, which stores the `Allocatable`,
  which is the capacity of the node and `Requested`, which is an aggregated sum of the resources
  requested by all pods assumed to be on that node (`Requested`).
* During the `Filter` stage of scheduling, the `NodeResourcesFit` plugin checks if a pod's requested
  resources can fit on the node (`NodeInfo.Allocatable - NodeInfo.Requested >= Pod request`). 
* The `NodeInfo.Requested` value is updated by the scheduler framework when a pod is "assumed" on the node.
  This happens after a node is selected in the `Scoring` phase, and before the actual binding to the 
  API server, ensuring the cache is accurate for subsequent scheduling decisions.

###### Dynamic Resource Allocation (DRA) Accounting

The `DynamicResources` plugin manages resources requested via `pod.spec.resourceClaims`. Its accounting
system is entirely separate from the standard resources.

* The DRA driver/s on the node reports resource availability through the `ResourceSlice` objects.  
* During the `Filter` stage, the `DynamicResources` plugin determines if the inventory in the
  `ResourceSlice` objects is sufficient to satisfy the pod's `ResourceClaim`, after accounting for
  devices already allocated to other claims.  
* When a pod is scheduled, the `DynamicResources` plugin, in its `PreBind` stage, makes an API call to
  update the `ResourceClaim` object's status. This update makes the allocation permanent and visible
  to the rest of the cluster.

The standard resource and dynamic resource accounting systems are completely independent. The
`NodeInfo` cache is not aware of allocations recorded in `ResourceClaim` objects, which is the root
cause of the accounting gap for node allocatable resources when they are managed through DRA.

#### Node Resource Enforcement Background

To enforce physical resource guarantees and isolation on the host, the Kubelet configures the kernel cgroup settings and Out-Of-Memory (OOM) score adjustments based on the pod specification.

##### Cgroup Enforcement

The Kubelet establishes resource boundaries at both the top-level pod cgroup and individual container cgroups via the Container Runtime Interface (CRI):
*   **Container-Level cgroups**: By default, the Kubelet translates the requests and limits specified in `pod.Spec.Containers[].Resources` directly into container-level cgroup parameters:
    *   **CPU Requests** establish the relative weight (`cpu.weight` or `cpu.shares`) for fair scheduling during machine contention.
    *   **CPU Limits** configure the hard threshold (`cpu.max` or `cpu.cfs_quota_us`). Workloads attempting to burst above this threshold are throttled by the kernel.
    *   **Memory Limits** set the memory usage threshold (`memory.max` or `memory.limit_in_bytes`). Exceeding this limit triggers an immediate Out-Of-Memory kill.
*   **Pod-Level cgroups**: When Pod Level Resources (`pod.spec.resources`) are explicitly specified, the Kubelet applies the overall resource request and limit directly to the parent pod-level cgroup. 
    *   The aggregate resource consumption of all containers combined (including init, sidecar, and regular containers) is hard-capped by this pod-level limit.
    *   If an individual container omits its own limit while a pod-level limit is set, the Kubelet applies the pod-level limit to that container's cgroup maximum value. This explicit fallback is critical because container-level limits are implied under a pod budget, and runtimes (such as the Java Virtual Machine) inspect container-level cgroup maximums to fine-tune internal memory pools and thread allocations.
    *   If pod level resources are not explicitly specified, the Kubelet sums up the container-level resource requests and limits and sets pod-level cgroups

##### OOM Score Adjustments

To ensure node stability during memory exhaustion, the Kubelet configures the `oom_score_adj` parameter for each container. This value informs the Linux kernel OOM killer which processes to terminate first:
*   For Guaranteed and BestEffort pods, the Kubelet applies static constant scores (`-997` and `1000`).
*   For Burstable pods, the score is dynamically calculated based on the container's standard memory requests relative to the node's memory capacity. Higher memory requests yield more protective (lower) scores, reducing the likelihood of premature termination.

### User Stories

**Story 1 (Resource Alignment):** An HPC workload needs a certain number of exclusive CPUs and memory
that are aligned on the same NUMA node as a specific NIC for maximum performance. The user creates a
`ResourceClaim` with co-location constraints to enforce this. The scheduler correctly accounts for the
CPU and memory requests made through the claim, adding them to the node's total requested resources, so
the node is not oversubscribed.

**Story 2 (Dedicated and Shared resources):** A telco application has some high-priority application
containers and some lower-priority sidecar containers. The user wants to dedicate some CPU cores
exclusively to the application containers for low latency, while allowing sidecar containers to run on
the node's general shared CPU pool. They use DRA to request exclusive cores and standard `pod.spec`
requests for the shared CPU portion. The scheduler should correctly account for both dedicated and shared
requests made through these different mechanisms. 

**Story 3 (Accelerator with Node Allocatable Resource Dependency):** An AI inference job requests a GPU through
a `ResourceClaim`. The specific GPU model also requires a certain number of CPUs and Hugepages that are
required for the application to interact with the accelerator. Instead of requiring the user to know
about these auxiliary CPU and HugePages requests and add it to their PodSpec, the GPU device can be configured to declare these dependencies. The Kubernetes scheduler accounts for both the CPU/HugePages
needs for the GPU device and the standard pod spec requests, ensuring the pod lands on a node with
sufficient capacity for all requirements. The user experience is simplified, as they only need to ask
for the primary device they care about.

**Story 4 (Fungibility):** An ML inference job can use either a full GPU or, if none is available, a
slice of 8 exclusive CPUs. The user creates a `ResourceClaim` with a `firstAvailable` list to
represent this fungible need. The scheduler evaluates both paths against a node's available
resources. It finds a node with 8 available CPUs, correctly reserves them in its central `NodeInfo`
cache, and schedules the pod. The user did not need to guess which resource to put in the `pod.spec`.  

### Risks and Mitigations

* Increased API and user complexity by having two ways to request node allocatable resources (PodSpec and
  ResourceClaim). To mitigate, the documentation would be enhanced with clear guidelines and use cases
  for DRA for Node Allocatable Resources.
* Bugs in the kube-scheduler's new accounting logic could lead to incorrect node resource calculations
  and node oversubscription. Extensive unit and integration tests covering various resource claim and
  standard request combinations should help mitigate this. The feature will also be rolled out
  gradually, beginning with an alpha release to gather feedback and address potential concerns.
* While the Kubelet considers DRA for cgroup enforcement, QoS class classification remains purely based on the standard Spec. 
  Pods that only use DRA claims to request node allocatable resources are classified as `BestEffort` pods and are more
  susceptible to node eviction and stricter cgroup enforcement compared to pods requesting the same amount of resources through standard requests.
  This is discussed in the [QoS Class Mismatch Risks](#qos-class-mismatch-risks) section.

## Design Details

The proposal here is to implement a **"Unified Accounting and Enforcement"** model across the control plane and the host for node allocatable resources requested through the standard pod Spec or through Dynamic Resource Allocation (DRA) claims. This involves:
1.  **API Changes**: Updates to the DRA API for drivers to declare node allocatable resource implications in `Device` objects, and PodStatus to record DRA-based node allocatable resource allocations.
2.  **Kube-Scheduler Changes**: Modifications in `NodeResourcesFit` and `DynamicResources` plugins to synchronize node resource usage tracking, delegating authoritative node-fit checks to the `DynamicResources` plugin when a pod utilizes DRA claims.
3.  **Kubelet Changes**: Updates in Kubelet to take into account resources allocated through DRA in the cgroup enforcement.

### Conceptual Mapping: Pod Spec Requests and Limits with DRA

Traditional resources like CPU and Memory in the Pod Spec have allocations split into requests (for capacity reservation and cgroup weight) and limits
(for hard cgroup ceilings). Since DRA is primarily used for hardware devices like accelerators and NICs, DRA API lacks the concept of separate requests and 
limits. To bridge standard resource enforcements with DRA claims, we use DRA allocations along with traditional requests and limits as follows:

*   **In the Scheduler**: In addition to standard requests, the DRA allocation acts as a **request** to deduct capacity from the node and prevent overcommitment.
*   **On the Node**: The DRA allocation acts as both a **request** (cgroup shares/weight) to enforce pod level cgroup bounds based on the scheduler-reserved 
    resource footprint and a **limit** to allow the containers to utilize the capacity.

Importantly, these DRA allocations are strictly **additive** to the standard resources declared in the Pod Spec; they enhance cgroup boundaries without replacing the existing Pod Spec-based requests and limits.

### API Changes

To support unified accounting for node allocatable resources, this KEP proposes API extensions to the `Device` object and `PodStatus`.

#### Device API Extensions

The new field `NodeAllocatableResources` within the
`ResourceSlice.Device` spec is used to define the node allocatable
resource quantities.

```go
// In k8s.io/api/resource/v1/types.go
type Device struct {
    // ... existing fields
    // NodeAllocatableResources defines the mapping of node resources
    // that are managed by the DRA driver exposing this device. These are resources currently
    // reported in v1.Node `status.allocatable` that are not extended resources
    // (see https://kubernetes.io/docs/concepts/configuration/manage-resources-containers/#extended-resources).
    // The only allowed keys are "cpu", "memory", and "hugepages-<size>".
    // In addition to standard requests made through the Pod `spec`, these resources
    // can also be requested through claims and allocated by the DRA driver.
    // For example, a CPU DRA driver might allocate exclusive CPUs or auxiliary node memory
    // dependencies of an accelerator device.
    // The keys of this map are the node-allocatable resource names (e.g., "cpu", "memory").
    // Extended resource names are not permitted as keys.
    // +optional
    // +featureGate=DRANodeAllocatableResources
    NodeAllocatableResources map[v1.ResourceName]NodeAllocatableResource `json:"nodeAllocatableResources,omitempty" protobuf:"bytes,14,opt,name=nodeAllocatableResources"`
}

// NodeAllocatableResource defines the translation between the DRA device/capacity
// units requested to the corresponding quantity of the node allocatable resource.
// At least one of Mapping or Overhead must be specified. Not specifying either is an invalid configuration.
type NodeAllocatableResource struct {
	// Mapping is used when the device directly models a node allocatable resource like standard CPU or memory
	// (e.g., with a CPU DRA driver). The calculated quantity is accounted for exactly once per claim instance
	// on the node. To prevent node cgroup isolation friction, the scheduler explicitly
	// blocks sharing mapped device claims across multiple pods.
	// +optional
	// +k8s:optional
	Mapping *NodeAllocatableMapping `json:"mapping,omitempty" protobuf:"bytes,3,opt,name=mapping"`

	// Overhead contains fields for modeling auxiliary overhead incurred on node allocatable resources
	// when allocating devices that are not themselves modeling a node allocatable resource (e.g., host memory overhead for GPUs).
	// Sharing overhead-mapped claims across multiple pods is allowed. The node allocatable overhead is accounted
	// for individually for each pod referencing the claim.
	// Overhead is always subtracted from the node's allocatable capacity for the resource, even when mapping
	// is specified for the same resource.
	// Eg: If a device models memory capacity per socket as a consumable capacity pool via Mapping (with CapacityKey),
	// any overhead specified for the same resource will be subtracted from the node's general allocatable capacity
	// and not from the per-socket capacity pool in Mapping.
	// +optional
	// +k8s:optional
	Overhead *NodeAllocatableOverhead `json:"overhead,omitempty" protobuf:"bytes,4,opt,name=overhead"`
}

// NodeAllocatableMapping defines how a DRA allocation directly translates into a node allocatable resource quantity.
// The mapping can be derived from either the count of allocated devices (via deviceMultiplier) or the specific capacity consumed (via capacityKey and capacityMultiplier). These options are mutually exclusive.
// Kubelet adds this mapped resource quantity from claim to both requests and limits at the pod-level cgroup, and to limits at the container-level cgroup for each container referencing the claim.
type NodeAllocatableMapping struct {
	// CapacityKey references a capacity name defined as a key in the
	// `spec.devices[*].capacity` map. When this field is set, the value associated with
	// this key in the `status.allocation.devices.results[*].consumedCapacity` map
	// (for a specific claim allocation) determines the base quantity for
	// the node allocatable resource. `capacityMultiplier` must also be set and is
	// multiplied with the base quantity.
	// For example, if `spec.devices[*].capacity` has an entry "dra.example.com/memory": "128Gi",
	// and this field is set to "dra.example.com/memory", then for a claim allocation
	// that consumes { "dra.example.com/memory": "4Gi" } the base quantity for the
	// node allocatable resource mapping will be "4Gi".
	// The final node allocatable resource amount is `consumedCapacity[capacityKey]` * `capacityMultiplier`.
	// +optional
	// +k8s:optional
	// +k8s:unionMember
	// +k8s:alpha(since: "1.37")=+k8s:dependentRequired("capacityMultiplier")
	CapacityKey *QualifiedName `json:"capacityKey,omitempty" protobuf:"bytes,1,opt,name=capacityKey"`

	// CapacityMultiplier is used as a multiplier for the allocated capacity consumed.
	// It is only valid if `capacityKey` is set.
	// The final node allocatable resource amount is `consumedCapacity[capacityKey]` * `capacityMultiplier`.
	// For example, if a Device's capacity "dra.example.com/cores" is consumed,
	// and each "core" provides 2 "cpu"s, the mapping would be:
	// {ResourceName: "cpu", capacityKey: "dra.example.com/cores", capacityMultiplier: "2"}.
	// If a claim consumes 8 "dra.example.com/cores", the CPU footprint is 8 * 2 = 16.
	// +optional
	// +k8s:optional
	// +k8s:alpha(since: "1.37")=+k8s:dependentRequired("capacityKey")
	CapacityMultiplier *resource.Quantity `json:"capacityMultiplier,omitempty" protobuf:"bytes,2,opt,name=capacityMultiplier"`

	// DeviceMultiplier is used as a multiplier for the allocated device count in the claim.
	// The final node allocatable resource amount is `deviceCount` * `deviceMultiplier`.
	// For example, a DRA driver representing each cache complex (CCX) as a device would have
	// {ResourceName: "cpu", deviceMultiplier: "8"} in its `nodeAllocatableResources`.
	// If 2 devices (CCX) are allocated to the claim, 2 * 8 = 16 CPUs would be considered as allocated.
	// It is only valid when `capacityKey` and `capacityMultiplier` are not set.
	// +optional
	// +k8s:optional
	// +k8s:unionMember
	DeviceMultiplier *resource.Quantity `json:"deviceMultiplier,omitempty" protobuf:"bytes,3,opt,name=deviceMultiplier"`
}

// NodeAllocatableOverhead defines auxiliary resource overheads incurred when allocating a device.
// Overheads can be specified as a fixed cost per pod referencing the claim, a variable cost per container reference, or both.
// Kubelet accounts for this overhead by adding it to both the pod-level and container-level cgroups of referencing containers.
type NodeAllocatableOverhead struct {
	// PerPod is overhead applied once per pod referencing the claim on this node.
	// This is a flat overhead incurred for every pod referencing the claim.
	// +optional
	// +k8s:optional
	PerPod *resource.Quantity `json:"perPod,omitempty" protobuf:"bytes,1,opt,name=perPod"`

	// PerContainer is applied per container reference to the claim.
	// This models overhead scaling linearly with the number of containers actively using the device.
	// When both PerPod and PerContainer are specified, the total overhead allocated for each pod referencing
	// the claim is computed as:
	// Quantity = PerPod + (PerContainer * NumReferences)
	// Kubelet accounts for this overhead in cgroups:
	// - Pod-level cgroup (requests and limits): Kubelet adds PerPod + (PerContainer * NumReferences).
	// - Container-level cgroup (limits only): Kubelet adds PerPod + PerContainer for each referencing container.
	// This allows any single container to access the pod-level overhead, while the parent cgroup caps the total usage to account for PerPod exactly once.
	// +optional
	// +k8s:optional
	PerContainer *resource.Quantity `json:"perContainer,omitempty" protobuf:"bytes,2,opt,name=perContainer"`
}
```

#### Pod API Changes

We add a new field `NodeAllocatableResourceClaimStatuses` to `PodStatus` as a way to pass the allocation details from the `DynamicResources` plugin to the kube-scheduler accounting logic.

```go
// In k8s.io/api/core/v1/types.go

// PodStatus represents information about the status of a pod.
type PodStatus struct {
    // ... existing fields

  // NodeAllocatableResourceClaimStatuses contains the status of node-allocatable resources
  // that were allocated for this pod through DRA claims. This includes resources currently
  // reported in v1.Node `status.allocatable` that are not extended resources
  // (see https://kubernetes.io/docs/concepts/configuration/manage-resources-containers/#extended-resources).
  // Examples include "cpu", "memory", "ephemeral-storage", and hugepages.
  // +featureGate=DRANodeAllocatableResources
  // +optional
  // +listType=atomic
  NodeAllocatableResourceClaimStatuses []NodeAllocatableResourceClaimStatus `json:"nodeAllocatableResourceClaimStatuses,omitempty" protobuf:"bytes,25,rep,name=nodeAllocatableResourceClaimStatuses"`
}

// NodeAllocatableResourceClaimStatus describes the status of node allocatable resources allocated via DRA.
type NodeAllocatableResourceClaimStatus struct {
	// ResourceClaimName is the resource claim referenced by the pod that resulted in this node allocatable resource allocation.
	// +required
	// +k8s:required
	ResourceClaimName string `json:"resourceClaimName" protobuf:"bytes,1,opt,name=resourceClaimName"`
	// Containers lists the names of all containers in this pod that reference the claim.
	// +optional
	// +listType=set
	// +k8s:optional
	// +k8s:listType=set
	Containers []string `json:"containers,omitempty" protobuf:"bytes,2,rep,name=containers"`

	// Resources is tombstoned since it got replaced with more granular Mapping and Overhead fields.
	// Resources map[ResourceName]resource.Quantity `json:"resources,omitempty" protobuf:"bytes,3,rep,name=resources"`

	// Mapping contains allocations through devices mapped in the device spec's `nodeAllocatableResources[...].mapping` field.
	// This is used by kubelet for pod level and container-level cgroup enforcement.
	// +optional
	// +patchStrategy=merge
	// +patchMergeKey=name
	// +listType=map
	// +listMapKey=name
	// +k8s:optional
	// +k8s:listType=map
	// +k8s:listMapKey=name
	Mapping []NodeAllocatableMappedResources `json:"mapping,omitempty" patchStrategy:"merge" patchMergeKey:"name" protobuf:"bytes,4,rep,name=mapping"`
	// Overhead contains allocations through devices mapped in the device spec's `nodeAllocatableResources[...].overhead` field.
	// This is used by kubelet for pod level and container-level cgroup enforcement.
	// +optional
	// +patchStrategy=merge
	// +patchMergeKey=name
	// +listType=map
	// +listMapKey=name
	// +k8s:optional
	// +k8s:listType=map
	// +k8s:listMapKey=name
	Overhead []NodeAllocatableOverheadResources `json:"overhead,omitempty" patchStrategy:"merge" patchMergeKey:"name" protobuf:"bytes,5,rep,name=overhead"`
}

// NodeAllocatableMappedResources describes mapped node allocatable resource allocations.
type NodeAllocatableMappedResources struct {
	// Name is the name of the resource (e.g., cpu, memory).
	// +required
	// +k8s:required
	Name ResourceName `json:"name" protobuf:"bytes,1,opt,name=name,casttype=ResourceName"`
	// Quantity is the total node allocatable resource capacity allocated for the claim.
	// This claim's allocated devices is shared by all the containers referencing the claim.
	// Kubelet adds this value to both requests and limits at the pod-level cgroup, and to limits at the container-level cgroup for each container referencing the claim.
	// +required
	// +k8s:required
	Quantity *resource.Quantity `json:"quantity" protobuf:"bytes,2,opt,name=quantity"`
}

// NodeAllocatableOverheadResources describes auxiliary overhead resource allocations.
type NodeAllocatableOverheadResources struct {
	// Name is the name of the resource (e.g., cpu, memory).
	// +required
	// +k8s:required
	Name ResourceName `json:"name" protobuf:"bytes,1,opt,name=name,casttype=ResourceName"`
	// PerPod is the flat overhead quantity allocated per pod.
	// Adding to each container limit allows individual containers to utilize the overhead, while the parent pod-level cgroup limit caps the total usage at the pod boundary where the overhead is accounted for exactly once.
	// At least one of PerPod or PerContainer must be specified. Specifying neither is an invalid configuration.
	// +optional
	// +k8s:optional
	PerPod *resource.Quantity `json:"perPod,omitempty" protobuf:"bytes,2,opt,name=perPod"`
	// PerContainer is the variable overhead quantity applied for each container referencing the claim.
	// The container references are recorded in `nodeAllocatableResourceClaimStatuses.containers`.
	// The total overhead quantity allocated for the claim is computed as:
	// Quantity = PerPod + (PerContainer * NumReferences)
	// Kubelet accounts for this overhead in cgroups:
	// - Pod-level cgroup (requests and limits): Kubelet adds PerPod + (PerContainer * NumReferences).
	// - Container-level cgroup (limits only): Kubelet adds PerPod + PerContainer for each referencing container.
	// This allows any single container to access the pod-level overhead, while the parent cgroup caps the total usage to account for PerPod exactly once.
	// At least one of PerPod or PerContainer must be specified. Specifying neither is an invalid configuration.
	// +optional
	// +k8s:optional
	PerContainer *resource.Quantity `json:"perContainer,omitempty" protobuf:"bytes,3,opt,name=perContainer"`
}
```

##### Resource Representation Examples


1.  Direct Device Mapping with Individual Devices
  
  *   Each device instance in the slice corresponds directly to a fixed unit of the node allocatable resource.
  *   The `deviceMultiplier` determines the resource footprint per device instance.
  *   The number of devices allocated to the claim multiplied by `deviceMultiplier` determines the overall node allocatable resource footprint and is recorded in the pod status.

  ```yaml
    # ResourceSlice
    apiVersion: resource.k8s.io/v1
    kind: ResourceSlice
    metadata:
      name: cpu-slice
    spec:
      driver: dra.example.com
      nodeName: my-node
      pool: { name: "node-pool", generation: 1, resourceSliceCount: 1 }
      devices:
      - name: cpu0
        attributes: { numaNode: 0 }
        nodeAllocatableResources:
          cpu: 
            mapping:
              deviceMultiplier: "1"
      - name: cpu1
        attributes: { numaNode: 0 }
        nodeAllocatableResources:
          cpu: 
            mapping:
              deviceMultiplier: "1"
    ---
    # ResourceClaim
    apiVersion: resource.k8s.io/v1
    kind: ResourceClaim
    metadata:
      name: cpu-claim
    spec:
      devices:
        requests:
        - name: cpu-req
          exactly:
            deviceClassName: cpu-core
            count: 2
    ---
    # Pod
    apiVersion: v1
    kind: Pod
    metadata:
      name: pod1
    spec:
      containers:
      - name: worker
        resources:
          claims:
          - name: my-cpu-claim
      resourceClaims:
      - name: my-cpu-claim
        resourceClaimName: cpu-claim
    status:
      nodeAllocatableResourceClaimStatuses:
      - resourceClaimName: cpu-claim
        containers:
        - worker
        mapping:
        - name: cpu
          quantity: "2" # Derived from 2 allocated devices * multiplier 1
  ```

2.  Direct Device Mapping with Consumable Capacity
   
  *   The device is represented as a consumable capacity.
  *   The `capacityKey` links the mapping directly to a specific capacity attribute inside the device.
  *   The scheduler reads the exact consumed capacity from the claim allocation results to determine the base quantity.
  *   Applying a `capacityMultiplier` allows translating between pool capacity units and standard resource units, converting one pool core into two standard CPUs.
  *   The final calculated amount is recorded in the pod status.

  ```yaml
    # ResourceSlice
    apiVersion: resource.k8s.io/v1
    kind: ResourceSlice
    metadata:
      name: native-resource-slice
    spec:
      driver: dra.example.com
      nodeName: my-node
      pool: { name: "node-pool", generation: 1, resourceSliceCount: 1 }
      devices:
      - name: socket0
        attributes:
          "dra.example.com/type": "socket"
        allowMultipleAllocations: true
        capacity:
          "dra.example.com/cores": "64"
          "dra.example.com/memory": "256Gi"
        nodeAllocatableResources: 
          cpu:
            mapping:
              capacityKey: "dra.example.com/cores"
              capacityMultiplier: "2"
          memory:
            mapping:
              capacityKey: "dra.example.com/memory"
              capacityMultiplier: "1"
    ---
    # ResourceClaim
    apiVersion: resource.k8s.io/v1
    kind: ResourceClaim
    metadata:
      name: shared-cpu-pool-claim
    spec:
      devices:
        requests:
        - name: cpu-pool-request
          exactly:
            deviceClassName: additional-cpu-memory
            capacity:
              requests:
                "dra.example.com/cores": "2"
    ---
    # Pod
    apiVersion: v1
    kind: Pod
    metadata:
      name: hpc-workload-pod
    spec:
      containers:
      - name: app
        resources:
          requests:
            cpu: "1"
          claims:
          - name: cpu-claim
      resourceClaims:
      - name: cpu-claim
        resourceClaimName: shared-cpu-pool-claim
    status:
      nodeAllocatableResourceClaimStatuses:
      - resourceClaimName: shared-cpu-pool-claim
        containers:
        - app
        mapping:
        - name: cpu
          quantity: "4" # Derived from consumed pool cores (2 cores * multiplier 2)
  ```

3.  Accelerator with Node Allocatable Resource Overhead Shared Across Multiple Containers

  *   The device publishes auxiliary resource overheads incurred per pod or container reference.
  *   Specifying both a fixed cost per pod and a variable cost per container allows modeling complex host memory dependencies.
  *   The scheduler compiles the active referencing containers array to compute the total overhead.
  *   These overheads accumulate without requiring it be specified inside the pod specification.

  ```yaml
    # ResourceSlice
    apiVersion: resource.k8s.io/v1
    kind: ResourceSlice
    metadata:
      name: my-node-xpus
    spec:
      driver: xpu.example.com
      nodeName: my-node
      devices:
      - name: xpu-model-x-001
        attributes:
          example.com/model: "model-x"
        nodeAllocatableResources:
          memory:
            overhead:
              perPod: "1Gi"
              perContainer: "500Mi"
    ---
    # ResourceClaim
    apiVersion: resource.k8s.io/v1
    kind: ResourceClaim
    metadata:
      name: tensor-accelerator-claim
    spec:
      devices:
        requests:
        - name: xpu-request
          exactly:
            deviceClassName: ai-accelerators
            count: 1
    ---
    # Pod
    apiVersion: v1
    kind: Pod
    metadata:
      name: ml-inference-pod
    spec:
      containers:
      - name: app-c1
        resources:
          claims:
          - name: gpu-ref
      - name: app-c2
        resources:
          claims:
          - name: gpu-ref
      resourceClaims:
      - name: gpu-ref
        resourceClaimName: tensor-accelerator-claim
    status:
      nodeAllocatableResourceClaimStatuses:
      - resourceClaimName: tensor-accelerator-claim
        containers:
        - app-c1
        - app-c2
        overhead:
        - name: memory
          perPod: "1Gi"
          perContainer: "500Mi"
  ```

4.  Partitionable Devices
  *   The resource is modeled hierarchically across NUMA or cache boundaries using shared counter sets.
  *   The specific capacity consumed from the shared counter set determines the direct resource footprint.

  ```yaml
    # ResourceSlice
    apiVersion: resource.k8s.io/v1
    kind: ResourceSlice
    metadata:
      name: cpu-topology-slice
    spec:
      driver: dra.example.com
      nodeName: my-node
      sharedCounters:
      - name: node-cpu-counters
        counters:
          "dra.example.com/cpu": { value: "32" }
      devices:
      # NUMA Level Devices
      - name: numa-0
        attributes:
          dra.example.com/type: numa
          dra.example.com/numaID: "0"
        capacity:
          "dra.example.com/cpu": "16"
        consumesCounters:
        - counterSet: node-cpu-counters
          counters:
            "dra.example.com/cpu": "16"
        nodeAllocatableResources:
          cpu:
            mapping:
              capacityKey: "dra.example.com/cpu"
              capacityMultiplier: "1"
      # L3 Cache Level Devices
      - name: numa-0-l3-0
        attributes:
          dra.example.com/type: l3cache
          dra.example.com/numaID: "0"
          dra.example.com/l3ID: "0"
        capacity:
          "dra.example.com/cpu": "8" # L3 cache drawing 8 CPUs
        consumesCounters:
        - counterSet: node-cpu-counters
          counters:
            "dra.example.com/cpu": "8"
        nodeAllocatableResources:
          cpu:
            mapping:
              capacityKey: "dra.example.com/cpu"
              capacityMultiplier: "1"
      - name: numa-0-l3-1
        attributes:
          dra.example.com/type: l3cache
          dra.example.com/numaID: "0"
          dra.example.com/l3ID: "1"
        capacity:
          "dra.example.com/cpu": "8"
        consumesCounters:
        - counterSet: node-cpu-counters
          counters:
            "dra.example.com/cpu": "8"
        nodeAllocatableResources:
          cpu:
            mapping:
              capacityKey: "dra.example.com/cpu"
              capacityMultiplier: "1"
      # ... additional devices for numa-1
    ---
    # ResourceClaim
    apiVersion: resource.k8s.io/v1
    kind: ResourceClaim
    metadata:
      name: l3-cache-claim
    spec:
      devices:
        requests:
        - name: l3-req
          exactly:
            deviceClassName: dra-l3-caches
            count: 1
    ---
    # Pod
    apiVersion: v1
    kind: Pod
    metadata:
      name: pod1
    spec:
      containers:
      - name: fast-app
        resources:
          claims:
          - name: cache-claim
      resourceClaims:
      - name: cache-claim
        resourceClaimName: l3-cache-claim
    status:
      nodeAllocatableResourceClaimStatuses:
      - resourceClaimName: l3-cache-claim
        containers:
        - fast-app
        mapping:
        - name: cpu
          quantity: "8" # Derived from specific consumed capacity key of the L3 cache device
  ```

5.  Fungible Resource Claim (GPU or CPU)
  *   The claim template uses `firstAvailable` to request either a GPU or a slice of 30 exclusive CPUs.
  *   If the scheduler selects the GPU, `nodeAllocatableResourceClaimStatuses` remains empty because the GPU does not manage node allocatable resources.
  *   If the scheduler selects the CPU slice, `nodeAllocatableResourceClaimStatuses` is populated with the 30 CPUs.

  ```yaml
    # ResourceClaimTemplate for Fungibility
    apiVersion: resource.k8s.io/v1
    kind: ResourceClaimTemplate
    metadata:
      name: gpu-or-cpu-template
    spec:
      spec:
        devices:
          requests:
          - name: gpu-or-cpu-req
            firstAvailable:
            - name: gpu
              deviceClassName: gpu-class
              count: 1
            - name: cpu
              deviceClassName: cpu-class
              capacity:
                requests:
                  "dra.example.com/cpu": "30"
    ---
    # Pod
    apiVersion: v1
    kind: Pod
    metadata:
      name: fungible-pod
    spec:
      containers:
      - name: my-app
        resources:
          requests: { cpu: "1", memory: "1Gi" }
          claims: [{ name: "gpu-or-cpu" }]
      resourceClaims:
      - name: gpu-or-cpu
        resourceClaimTemplateName: gpu-or-cpu-template
    ---
    # Pod Status (Scenario A: GPU Selected)
    status:
      nodeAllocatableResourceClaimStatuses: []
    ---
    # Pod Status (Scenario B: CPU Selected)
    status:
      nodeAllocatableResourceClaimStatuses:
      - resourceClaimName: gpu-or-cpu
        containers: ["my-app"]
        mapping:
        - name: cpu
          quantity: "30"
  ```

#### API Validation

* The keys in the `nodeAllocatableResources` map must be exactly `cpu`, `memory`, or `hugepages-<size>`. All other names, including extended resources and `ephemeral-storage`, are rejected (`ephemeral-storage` is deferred to beta together with DRA-aware eviction in kubelet).
* Within a single resource mapping, at least one of the `mapping` or `overhead` fields must be specified.
* If `mapping` is specified, it must use either `deviceMultiplier` or a combination of `capacityKey` and `capacityMultiplier`. These options are mutually exclusive.
* If `capacityKey` is specified, it must be a valid qualified name and `capacityMultiplier` is required.
* If the `overhead` field is specified, it must contain at least one non-negative value for either the `perPod` or `perContainer` overhead quantities.
* For `PodStatus` updates, each entry in the `nodeAllocatableResourceClaimStatuses` array must reference a valid claim name and contain correctly formatted resource quantities.

### Kube-Scheduler Changes

The scheduling process for a Pod involves several stages. The following describes how the `NodeResourcesFit` and
`DynamicResources` plugins interact within the kube-scheduler framework to achieve unified accounting for node allocatable resources
managed by DRA. The key goal is to ensure that the delegation mechanism works regardless of the execution order of these
plugins.

1. **PreFilter Stage:**
   *  **DynamicResources Plugin:** Validates the `ResourceClaim` and its associated `DeviceClass`. It ensures that the referenced classes exist.
   *  **NodeResourcesFit Plugin:** Calculates and caches the pod's total standard resource requests (summing up containers). It does **not**
      perform resource fit checks or filter nodes at this stage. As node allocatable resource claims can only add to standard requests, the
      delegation mechanism between the plugins is optional. Without delegation there is a dual resource fit check in both the 
      `NodeResourcesFit` and the `DynamicResources` plugins, but the `DynamicResources` plugin's check is the authoritative check.

2. **Filter Stage:** This stage performs the node-level checks to determine if a pod fits on a specific node.
   *  **NodeResourcesFit Plugin:** In the Alpha stage, this plugin would continue to do the resource fit based on standard requests.
   *  **DynamicResources Plugin:** This plugin takes on the authoritative role for checking node allocatable resource fit if any of the
      pod's `ResourceClaim`s request node allocatable resources.
      *   The plugin tries to allocate devices to all the resource claims of the pod.
      *   **Claim Resource Calculation:** For each allocated device, the plugin checks `nodeAllocatableResources` and computes the quantity for each node allocatable resource based on whether a mapping and/or overhead mapping is specified:
          *   If `mapping` is specified, the quantity is derived using the `capacityKey`, `capacityMultiplier`, or `deviceMultiplier` fields. If `capacityKey` is set, the base quantity is the consumed capacity from the claim allocation results multiplied by `capacityMultiplier`. If `capacityKey` is omitted, the `deviceMultiplier` is applied directly to the count of allocated devices.
          *   If `overhead` is specified, the auxiliary overhead is calculated by summing any `perPod` cost and the variable `perContainer` cost scaled by the number of active container references.
      *   The plugin calculates the total effective demand for each node allocatable resource by:
          *   Summing up container requests from the pod spec requests and the amounts determined from DRA claims.
          *   If a claim is referenced by multiple containers, it is accounted for only once.
          *   If pod level resources are also specified, that takes precedence and determines the resource footprint of the pod. 
              This interacts with the prioritized list DRA feature such that when explicit pod level resources are used, the Pod footprint remains the same 
              regardless of the chosen device request; but without explicit pod level resources, the Pod footprint will vary based on which device request is 
              chosen.
      *   **Validation**: The plugin validates the following scenarios:
          *   If Pod Level Resources are defined, the plugin will validate that the sum of effective 
              requests (standard + DRA claims) does not exceed the budget set at the pod level in `pod.spec.resources`([details](#integration-with-pod-level-resources)).
          *   The plugin enforces sharing rules based on mapping. If a claim is already assigned to
              an existing pod and the allocated device uses direct device mappings
              (`nodeAllocatableResources[...].mapping`), shared access is blocked across pods to prevent
              cgroup conflicts. Auxiliary overhead mappings (`nodeAllocatableResources[...].overhead`) are
              allowed to share across pods ([details](#handling-shared-claims)).
      *   This total effective demand is checked against the node's allocatable resources and node is filtered out if it does not have enough capacity.
      *   The calculated node allocatable resource allocations for the pod on this specific node (`NodeAllocatableResourceClaimStatus`) are
          stored in the `CycleState`. This is needed for passing the node-specific allocation details to the later
          `Assume` and `PreBind` stages.

3.  **Scheduler Internal Cache Update:** After a node is selected, the scheduler updates its internal cache to reflect the
    resources consumed by the new pod. This stage is critical for maintaining the internal cache consistent. The scheduler
    framework "assumes" the pod will run on the selected node and updates its cache without waiting for bind (updating the
    API server) to succeed. Without an "assume" step, the scheduler might try to place other pods on the same node using
    stale resource information, potentially leading to oversubscription. The Assume phase reserves the resources in the
    scheduler's in-memory cache immediately.
    *   The scheduler framework retrieves the node-specific allocation status from the cycle state which was populated 
        during the `DynamicResources` Filter stage.
    *   This is then applied to the in-memory copy of the Pod object's status (`pod.status.nodeAllocatableResourceClaimStatuses`) that the 
        scheduler is about to "assume".
    *   The pod's overall resource footprint is natively computed via `PodInfo.CalculateResource()` (`pkg/scheduler/framework/types.go`), which 
        checks the `UseDRANodeAllocatableResourceClaimStatus` option to sum standard requests and DRA status allocations. This is added to `nodeInfo.Requested`.

4.  **PreBind Stage:** This stage performs actions right before the pod is immutably bound to the node.
    *   **DynamicResources Plugin:** The plugin updates the `ResourceClaim.Status` to reflect the allocated devices. It also
        patches the `Pod.Status` to add the `NodeAllocatableResourceClaimStatuses` field, persisting the information calculated during
        the Filter stage and making this information available for components like the Kubelet. Kubelet consumes the status field directly 
        during [pod admission](#kubelet-admission-control) and [cgroup enforcement](#cgroup-enforcement).

5.  **Bind Stage:** This stage executes asynchronously after the main scheduling cycle has decided on a node. The scheduler
    listens for pod `Update` events, and transitions the pod from the "assumed" state to "bound" if the bind process
    succeeded. The resource accounting on the `NodeInfo` does not change at this point (as they were previously accounted for
    during the "Assume" step). If the bind fails, or if the Kubelet later rejects the Pod, the scheduler detects this and
    reverts the resource allocation in its cache, decrementing `nodeInfo.Requested`.

#### Resource Calculation

To ensure consistent resource accounting across multiple consumers, the core logic for calculating a pod's total
resource footprint, including DRA-managed node allocatable resources, will be centralized in the `PodRequests` function within the
`k8s.io/component-helpers/resource` package. This helper function is currently used by various components, including scheduler plugins like `NodeResourcesFit`, the `NodeInfo` cache update, and the Kubelet's admission handler.

The total node allocatable resource requirements for a pod are determined as follows:
*   **With Pod-Level Resources**: If pod-level resources (`pod.spec.resources.requests`) are specified for a resource, they define the overall footprint for that resource. Individual container-level requests and any DRA status allocations/overheads are ignored.
*   **Without Pod-Level Resources**: The footprint is calculated by combining standard container requests and DRA status allocations:
    - For each container, its effective request is the sum of its standard resource requests and any DRA allocations it references. We get these 
      DRA allocations from the fields in `pod.status.nodeAllocatableResourceClaimStatuses` (both `mapping` and `overhead` mappings).
    - If init containers reference a claim with an overhead.perContainer mapping, we rely on the existing logic used with standard requests where the
      peak of regular and init containers' resources is considered.
    - Any pod-scoped DRA overheads (`overhead.perPod`) are added directly to this total.
*   **Pod Overhead**: In both cases, if standard pod overhead (`pod.spec.overhead`) is specified, it is added to the final calculated sum.
*   **Interaction with In-Place Resizing**:
    - With Pod-Level Resources:
      - When a running pod is resized, the pod-level Spec (`pod.spec.resources.requests`) is updated. Before the Kubelet accepts and actuates this resize, the scheduler computes the footprint (in `PodRequests()`) 
        using the maximum of `desired` (`pod.spec.resources.requests`), `allocated` (`pod.status.allocatedResources`), and `actuated` (`pod.status.resources.requests`) resources.
      - Because the pod-level `allocated` and `actuated` status APIs are updated to include DRA, this `max` calculation automatically accounts for the DRA resources. We do not need to include `pod.status.nodeAllocatableResourceClaimStatuses` again.
    - Without Pod-Level Resources:
      - When a running pod is resized, standard container requests are updated in the Spec. Before Kubelet actuates the resize, `PodRequests()` computes the standard container requests using the maximum of `desired` (`container.resources.requests`),
       `allocated` (`containerStatuses[*].allocatedResources`), and `actuated` (`containerStatuses[*].resources.requests`) resources, and adds the static DRA resources.  Since `actuated` already contains DRA enforced values, we need to deduplicate 
       this before adding `pod.status.nodeAllocatableResourceClaimStatuses` so that DRA resources are not double-counted.

#### Integration with Pod Level Resources

When Pod Level Resources are specified (`pod.spec.resources`), it continues to set the overall budget for the pod.
Node allocatable resources added to individual containers via DRA claims must be accounted for within this pod-level budget.
The effective resource request for a container is the sum of its base request specified in `spec.containers[].resources.requests` 
and any additional resources allocated through DRA claims.

Currently, with pod level resources, an admission time validation ensures that the sum of container requests does not 
exceed pod level requests. However, this is insufficient for pods with node allocatable resource claims, as their exact quantities 
are only determined after the `DynamicResources` scheduler plugin allocates devices. This allocation can be dynamic, 
especially for claims with [prioritized lists](https://github.com/kubernetes/enhancements/blob/master/keps/sig-scheduling/4816-dra-prioritized-list/README.md) (fungibility use cases).
Therefore, the `DynamicResources` plugin must perform an additional validation step during its `Filter` stage. After allocating 
devices to claims and calculating the node allocatable resources added, the plugin will verify that the total effective pod demand 
(standard container requests + DRA node allocatable resources) does not surpass the limits set in `pod.spec.resources`.

If a pod requests a specific set of devices via DRA claims, and the resulting node allocatable resource footprint 
(base container + DRA additions) exceeds the `pod.spec.resources` budget, this failure is global to the pod. 
The `DynamicResources` plugin would return `UnschedulableAndUnresolvable`. 

**Note:**
DRA Prioritized Lists (Fungibility) Limitation: Because pod level resources acts as a strict ceiling, using prioritized lists with pod level resources 
is a known limitation. The pod level budget must be sized to fit the maximum resource option in the prioritized list. If the scheduler chooses a
lower-overhead option, the capacity remains unused. It is not recommended to use prioritized lists with pod level resources.

#### Handling Shared Claims

**Intra-Pod Sharing:**
Containers within the same pod can reference the same `ResourceClaim`. The node allocatable resources associated with the claim are accounted for 
only once for the entire pod, as described in the Resource Calculation section. The resource calculation shared library function 
`PodRequests()` can effectively handle de-duplication for claims shared within a single pod, as all necessary information is self-contained 
within the Pod scope (standard requests in Spec and DRA requests in `status.nodeAllocatableResourceClaimStatuses`).

**Inter-Pod Sharing:**

Sharing `ResourceClaim`s that manage node allocatable resources across different pods is evaluated differentially depending on the mapping type established in the `Device` mapping:
1.  **CPU/Memory Direct Mappings (`Mapping` field is set)**: The `DynamicResources` plugin **continues to block sharing across pods** (returning `UnschedulableAndUnresolvable`). 
    Sharing pools of direct native resources creates severe accounting ambiguities (attributing fractional pool costs against distinct pod-level budgets) and intense Kubelet cgroup reconciliation friction.
2.  **Accelerator Overheads (Only `Overhead` field is set)**: The `DynamicResources` plugin **allows sharing across pods**. Auxiliary overheads represent host memory or 
    auxiliary tracking structures required per consumer pod/reference. Because these represent standard additive overheads without dynamic draw-down interactions, 
    the scheduler and Kubelet safely accumulate and sum all resources directly from `pod.Status.NodeAllocatableResourceClaimStatuses` for each individual pod independently.

The `DynamicResources` plugin enforces the sharing restriction during the `Filter` stage by inspecting the claim's
existing consumers (`claim.status.reservedFor`): if an allocated claim with a `mapping` entry is already reserved for
another pod, the node is rejected with `UnschedulableAndUnresolvable`. Pods reserved together in the same scheduling
cycle (e.g., gang scheduling) are permitted, since their footprints are accounted together.

No new scheduler framework API is required for this. The `NodeAllocatableDRAClaimState` type and the corresponding
`NodeInfo` tracking introduced in the initial alpha (v1.36) were removed in the alpha2 rework of the
`k8s.io/kube-scheduler` staging module.

#### Multiple Claims per Container

A single container can reference multiple DRA claims. The node allocatable resources from each distinct claim are summed up to contribute to the pod's total resource requirements.

**Example:**
* Combining additive policies.
    ClaimA - requests 4 CPUs
    ClaimB - requests 2 CPUs
    * Pod 1
      1. Container "c1"
        * Spec: requests 1 CPU
        * claims: ClaimA, ClaimB
      2. Container "c2"
        * Spec: requests 2 CPU
        * claims: ClaimA
    * **Result:** 
      * Pod Effective CPU = 1 (c1 PodSpec) +  4 (ClaimA) + 2 (ClaimB) + 2 (c2 PodSpec) = 9 CPUs.
      * Claim A is accounted for only once

#### Unreferenced Claims

If a `ResourceClaim` is listed in `pod.spec.resourceClaims` but not referenced by any container in `pod.spec.containers[*].resources.claims`, 
the resources associated with this claim are still accounted for against the node's capacity once. This is because 
the DRA allocator allocates the devices to the claim making them unavailable to others (e.g., exclusive CPUs requested through a claim). 
This will be enforced in the `PodRequests()` helper function when computing the pod resource footprint.

#### Preemption

If a high-priority Pod is unschedulable due to insufficient resources, the scheduler tries to find a suitable node by preempting lower-priority pods:
*   The default preemption plugin simulates evicting ([`SelectVictimsOnNode()`](https://github.com/kubernetes/kubernetes/blob/451b50df783cc381f15c9c2a35d2948a699c249a/pkg/scheduler/framework/plugins/defaultpreemption/default_preemption.go#L252)) lower-priority pods. Because the victim pods are already running on the node, 
    and the pod status is populated with DRA allocations, the resource calculation helper function (`PodRequests()`)
    accurately subtracts both the victim's Spec requests and its dynamic status claim allocations.
*   When the default plugin simulates adding back candidate victims one by one to see if the incoming pod still fits, this check automatically aggregates both standard Spec 
    requests and dynamic status claim allocations for the reprieved pods.
*   During these eviction and reprieve simulations, the preemption plugin always checks ([RunFilterPluginsWithNominatedPods()](https://github.com/kubernetes/kubernetes/blob/451b50df783cc381f15c9c2a35d2948a699c249a/pkg/scheduler/framework/plugins/defaultpreemption/default_preemption.go#L302)) if the pod fits. The dynamic resources plugin node-fit check includes DRA allocations, the preemption plugin correctly identifies candidate nodes.
*   There is an independent proposal for [DRA preemption](https://github.com/kubernetes/enhancements/pull/6113). However, because node allocatable claims are mapped to standard resources and are already included in the scheduler resource footprint
    calculation and internal cache updates, DRA-based node allocatable requests are automatically considered during preemption even without the DRA preemption feature enabled.

### Node Resource Enforcement and Isolation

#### Scope

The Kubelet's primary responsibility is to set up the cgroup hierarchy, set pod-level ceilings, and container-level headroom (limits). It guarantees that the 
pod-level parent cgroup bounds have the correct resource ceilings, and container-level cgroups have safe defaults (e.g., CFS quota, memory limits) so that workloads
can utilize their claim resources without throttling or OOM kills. DRA drivers can then modify these container-specific settings configured by the Kubelet or apply 
new enforcements (e.g., CPU pinning or binding memory to specific NUMA nodes) by interfacing directly with the Container Runtime (e.g., a CPU DRA driver using NRI to set `cpuset.cpus`). 
Considering DRA resources in Kubelet cgroup enforcement guarantees that any container-level modifications or overrides applied by a DRA driver are contained and 
cannot affect other co-located pods on the node. This helps to keep the KEP generic and independent of specific DRA driver implementations.

#### Key Principles
*   Kubelet's DRA-specific adjustments to cgroup enforcement are derived solely from 
    `pod.status.nodeAllocatableResourceClaimStatuses` as updated by the scheduler.
*   If Pod Level Resources are explicitly specified, that takes precedence at both the scheduler level for
    accounting and the node level for cgroup enforcement.
*   The QoS classification of a pod remains determined strictly by the standard requests and limits in the
    PodSpec. DRA claims do not alter the pod's QoS tier.
*   If a standard request or limit is not specified in the spec, the defaulting mechanism that we currently
    have (for example, setting CPU shares to 2, or quota to unlimited) remains true. The defaulting logic at
    the pod level and container level cgroups is still determined based on standard Spec, and DRA does not change that.

#### Cgroup Enforcement

To enforce container and pod-level cgroup settings, Kubelet reads `NodeAllocatableResourceClaimStatuses` from `pod.Status` and uses this 
information along with standard resource requests and limits specified in the Pod Spec (`pod.spec.containers[].resources` and `pod.spec.resources` 
when using Pod-Level Resources) to determine the overall cgroup allocations. Kubelet evaluates cgroup settings at both the pod level and container level.

Workload resource boundaries are actuated at two distinct levels in the host cgroup v2 hierarchy:
*   **Pod-Level parent cgroups** 
    -   Establish the overall aggregate resource boundary for the entire pod. 
    -   This parent cgroup acts as a shared pool of resources, enabling containers to dynamically share CPU and memory while safely bounding the pod's overall resource footprint.
    -   Enforced directly by kubelet.
*   **Container-Level cgroups** 
    -   Applies granular resource isolation boundaries directly to the container based on container Spec (or default values when not specified).
    -   Enforced through CRI.

Kubelet translates Pod Spec resource requests and limits into corresponding cgroup settings using these core cgroup properties:
*   **CPU Requests** are mapped to **CPU Shares/Weight** (`cpu.weight`): Controls the relative CPU scheduling weight/priority of the pod or container when the node experiences CPU contention.
*   **CPU Limits** are mapped to **CPU Quota** (`cpu.max`): Caps the absolute maximum CPU time the pod/container can consume in a time window (configurable).
*   **Memory Limits** are mapped to **Memory Limit** (`memory.max`): Caps the absolute maximum memory (RAM) the pod/container can consume.
*   **HugePages Limits** are mapped to **HugePages Limit** (`hugepages.limit_in_bytes`): Caps the maximum hugepage allocation size.

Kubelet also sets up the cgroup directories for the pod based on the QoS class (`Guaranteed`, `BestEffort` or `Burstable`). DRA based allocation **does not**
have an influence on the QOS class of the pod and how Kubelet sets up cgroup hierarchies.

Kubelet evaluates cgroup settings at both the pod level and container level as follows:

##### Pod-Level Cgroup Settings

**Without DRA:**
If `PodLevelResources` are enabled and explicitly specified (`pod.spec.resources.requests` and `pod.spec.resources.limits`), Kubelet sets the pod-level cgroup settings 
exactly to those explicit values. If `PodLevelResources` are not specified, Kubelet sums up all container-level requests and limits and sets the pod level cgroup settings.

**With DRA:**
If `PodLevelResources` are enabled and explicitly specified (`pod.spec.resources.requests` and `pod.spec.resources.limits`), Kubelet sets the pod-level cgroup settings exactly 
to those explicit values **without adding DRA allocations**. If `PodLevelResources` are not specified, Kubelet sums up all container-level requests and limits and **adds DRA allocations**.

At the pod level, Kubelet sets the cgroup parameters as follows:

```
CPU Shares      = MilliCPUToShares( Sum(Spec.Requests[cpu]) + DRADirectMapped(cpu) + DRAOverheadMappedPodTotal(cpu) )
CPU Quota       = Sum(Spec.Limits[cpu]) + DRADirectMapped(cpu) + DRAOverheadMappedPodTotal(cpu)
Memory Limit    = Sum(Spec.Limits[memory]) + DRADirectMapped(memory) + DRAOverheadMappedPodTotal(memory)
HugePages Limit = Sum(Spec.Limits[hugepages-<size>]) + DRADirectMapped(hugepages-<size>) + DRAOverheadMappedPodTotal(hugepages-<size>)
```

*   **`Sum(Spec.Requests[resource])`**: Sum of requests across all containers in the pod.
*   **`Sum(Spec.Limits[resource])`**: Sum of limits across all containers in the pod.
*   **`DRADirectMapped(resource)`**: Sum of direct mapped DRA allocations for all the claims referenced in the pod (obtained from `pod.status.nodeAllocatableResourceClaimStatuses[].mapping[].quantity`).
*   **`DRAOverheadMappedPodTotal(resource)`**: Sum of overhead mapped DRA allocations across all distinct claims allocated to the pod, obtained as `PerPod + (PerContainer * len(containers))`.

**Why Pod Level Cgroup Limits includes DRA allocations?**

*   The pod's cgroup slice establishes the absolute upper ceiling (`cpu.max`, `memory.max`, `hugepages.limit_in_bytes`) for the entire pod workloads footprint.
*   If DRA allocations (direct or overhead) are not added to the pod workloads cgroup limits, the pod-level ceiling remains locked at standard Spec-pure limits
    The moment any container attempts to utilize its DRA capacity, the overall pod usage will hit the uninflated parent boundary, resulting in immediate CPU throttling, memory OOM kills, or hugepage allocation failures.
*   If `PodLevelResources` are explicitly declared in `pod.spec.resources.limits`, the Kubelet respects the user's aggregate pod limits budget and **does not
    add** DRA allocations, expecting the user to have configured the pod level settings to include DRA allocations.


**Why Pod Level Requests / CPU Shares includes DRA allocation ?**

*   Since DRA CPU resources are accounted during node capacity calculations during scheduling, the scheduler has already reserved and deducted these CPUs from the node's capacity. 
    Including the DRA values at the pod-level cgroup ensures that the host kernel actually honors this scheduler-level resource reservation under node contention.
*   In Linux, CPU shares (`cpu.weight`) act as relative priority weights that are only enforced when the entire node experiences heavy CPU contention. Including DRA requests at the
    pod level ensures the entire pod successfully secures its aggregate resource footprint against other pods on the node.
    Including the DRA values at the pod-level cgroup ensures that the host kernel actually honors this scheduler-level resource reservation under node contention.
    -   *Example*: If a container requests `100m` CPU through a standard request, and gets `1 CPU` through a DRA claim for a GPU device (overhead), setting the 
        CPU shares only based on the standard `100m` CPU request would starve the container during node CPU contention.
*   This is in line with the [scope](#scope) of the KEP that Kubelet sets the pod-level cgroup boundaries based on DRA and sets safe defaults at the container level allowing for the DRA driver to modify. 
    This allows for the DRA drivers to model both shared and exclusive resources.

##### Container-Level Cgroup Settings

At the container level, Kubelet sets the cgroup parameters as follows:

```
CPU Shares      =  MilliCPUToShares(Spec.Requests[cpu]) # No changes
CPU Quota       = Spec.Limits[cpu] + DRADirectMapped(cpu) + DRAOverheadMappedPerContainer(cpu) + DRAOverheadMappedPerPod(cpu)
Memory Limit    = Spec.Limits[memory] + DRADirectMapped(memory) + DRAOverheadMappedPerContainer(memory) + DRAOverheadMappedPerPod(memory)
HugePages Limit = Spec.Limits[hugepages-<size>] + DRADirectMapped(hugepages-<size>) + DRAOverheadMappedPerContainer(hugepages-<size>) + DRAOverheadMappedPerPod(hugepages-<size>)
```

*   **`Spec.Requests[resource]`**: Standard request specified in `pod.spec.containers[].resources.requests` (or default value if unset)
*   **`Spec.Limits[resource]`**: Standard limit specified in `pod.spec.containers[].resources.limits`. If container-level limits are omitted
    but `PodLevelResources` (`pod.spec.resources.limits`) are explicitly specified, this value falls back to the pod level resource limit.
*   **`DRADirectMapped(resource)`**: Sum of direct compute resources allocated via DRA (e.g., resources allocated via cpu/memory dra driver), obtained from `pod.status.nodeAllocatableResourceClaimStatuses[].mapping[].quantity`.
*   **`DRAOverheadMappedPerContainer(resource)`**: Sum of overhead resources allocated via DRA (e.g., additional cpu/memory resources for a GPU device), obtained from `pod.status.nodeAllocatableResourceClaimStatuses[].overhead[].perContainer`.
*   **`DRAOverheadMappedPerPod(resource)`**: Sum of overhead DRA allocations for the pod, obtained from `pod.status.nodeAllocatableResourceClaimStatuses[].overhead[].perPod`.
    *   Since the claim resources are shared by all containers referencing the claim, the per-pod overhead is included in the limit of all the containers, but is counted exactly once at the parent pod-level cgroup ceiling.

**Why Container Level Limits includes DRA allocations?**

*   To allow containers to successfully consume and utilize their allocated DRA claims, their nested container level cgroup limits must be inflated to
    accommodate the additional capacity. Without this, the container would be immediately throttled or OOM-killed by its spec-only cgroup boundary, completely rendering the DRA allocations unusable.

**Why Container Level Requests / CPU Shares DOES NOT INCLUDE DRA allocations?**

*   The Kubelet lacks the context to know whether a DRA allocation represents exclusive resources or shared capacity. If DRA allocates exclusive CPUs, 
    considering those to determine the shared CPU weight would allow the container to unfairly dominate the shared CPU pool during contention with other 
    containers in the pod that do not use exclusive CPUs. 
*   If a claim is shared by multiple containers within a pod, attempting to split the claim's request among those referencing containers CPU shares would introduce 
    enforcement complexity and ambiguity. To perfectly set the container level Cgroup settings, we would need to know the exact type of resource allocation made through 
    DRA and can be explored as a future enhancement ([Pass Allocation Details from Driver to Kubelet](#pass-allocation-details-from-driver-to-kubelet)).

*   The risk here is that the DRA allocations are not added to CPU shares, a container using only a claim and no standard request receives minimal CPU weight 
    (`2`), risking starvation during contention within the containers of the pod. 
    However, keeping container-level CPU shares only based on spec is a safe and sufficient default for the alpha implementation due to the following reasons:
    *   Including DRA allocation at the pod level CPU shares provides guarantees and due to the cgroup hierarchy, the pod as a whole gets the shares proportional to scheduler allocated resources.
    *   This is only relevant if the DRA driver does not allocate exclusive CPUs. If the driver allocates exclusive CPUs, there is no contention with other containers in the pod.
    *   This risk is fully manageable. The [scope](#scope) is strictly to configure the baseline cgroup settings, which the DRA driver can then modify or optimize.

##### QoS Class Mismatch Risks

Because a pod's Quality of Service (QoS) class is determined strictly by the standard container resource definitions in `pod.Spec` and ignores DRA Status allocations,
workloads can experience degradation because of how cgroups are configured by kubelet. The risks vary based on the pod's resulting QoS category:

**1. Pod Categorized as BestEffort**

If the Pod Spec completely omits both requests and limits for both CPU and Memory (either at the pod level in `pod.Spec.Resources` when using Pod-Level Resources, 
or across all containers in `pod.Spec.Containers[*]`), the pod is classified as a **BestEffort** QoS class. The risks of a pod with DRA claims being categorized as BestEffort are:

  - Kubelet places the pod under `kubepods.slice/kubepods-besteffort.slice/`. This parent slice has CPU shares (`cpu.weight`) set to `MinShares (2)`. Under node-wide CPU contention, 
    the container can be starved because of this parent boundary, regardless of its internal cgroup weight (which can be set by the DRA driver). This CPU starvation 
    risk is only relevant if the workload runs in a shared CPU pool; if the DRA driver allocates exclusive CPU cores and pins the container via cgroup cpuset configurations,
    CPU shares are completely ignored and the core allocation is fully guaranteed without starvation.
  - BestEffort pods receive the maximum OOM score adjustment (`1000`) and are ranked first for preemption and eviction by the Eviction Manager during memory or disk pressure.

**2. Pod Categorized as Burstable**

If the Pod Spec specifies any standard CPU or Memory request or limit (either at the pod level in `pod.Spec.Resources`, or for at least one container in `pod.Spec.Containers[*]`), 
but the pod does not meet the strict requirements for the `Guaranteed` QoS (i.e., where requests must match limits exactly for both CPU and Memory), the pod is classified as **Burstable** QoS class.
  - Since CPU shares (`cpu.weight`) remain based strictly on standard requests, a container requesting a small standard amount but receiving a large allocation via DRA would still 
    have lower CPU shares. Similar to BestEffort, this is not relevant if the DRA driver allocates exclusive CPUs and manages core pinning directly.

###### Potential Mitigations

Any container-level risks due to Kubelet setting defaults/baseline values not considering exact intent of the claim can be solved at the DRA driver level by updating these base values set by kubelet. 
However, because the driver is strictly confined to operate at the container level, it cannot modify the parent-level Pod cgroup boundaries.

*   Ensure that pods using DRA for CPUs are not classified as BestEffort by specifying a non-zero standard CPU or memory request on one of the containers in `pod.Spec`. 
    This promotes the pod to the Burstable QoS tier, moving it out of the BestEffort slice where cgroup values are locked at the parent level.
*   Use Pod-Level Resources to declare the total aggregate requests (including DRA allocations) at the pod level in `pod.Spec.Resources`. This works well only when the claim 
    resources are completely deterministic, and it is not suitable for advanced use cases where the mapping between CPU/Memory and the DRA allocation is not 1:1 (such as modeling L3 caches instead of CPUs directly) 
    or when using a DRA prioritized list where the actual allocation quantity is not known until scheduling time.

###### Long-Term Mitigation - Explicit QoS Class

A robust long-term solution would be to allow workloads to declare an explicit QoS class directly in the Pod Spec, rather than relying on implicit derivations inside Kubelet. 
This was also explored as part of [KEP-1287](https://github.com/kubernetes/enhancements/tree/master/keps/sig-node/1287-in-place-update-pod-resources#design-sketch-explicit-qos-class) to 
loosen QoS restrictions during in-place pod resizing. With multiple independent variables now affecting a pod's resource footprint (standard container specs, Pod-Level Resources, in-place resizing, and now DRA), 
attempting to implicitly derive the QoS class by coordinating all these inputs is highly complicated and remains a maintenance challenge and exploring explicit QoS class configuration is a more desirable path.

##### Handling Pod Level Resources

When `PodLevelResources` is used, the Kubelet's cgroup enforcement must reconcile explicit pod-level limits with DRA allocations. This requires two specific adjustments:
*   **Pod-Level Cgroup Ceilings**:
    If explicit pod-level limits are specified, they determine the overall pod budget. The Kubelet sets the pod's cgroup ceiling exactly to the specified `pod.spec.resources.limits`. 
    It **does not add the DRA allocations** to the pod-level limit, because the DRA resources are already encompassed within this overall budget.
*   **Container-Level Fallbacks**: 
    If a container lacks its own limit, the pod-level limit is applied to the container's cgroup maximum value.

##### Handling Missing Limits

When a container omits limits for CPU, Memory, or HugePages, the Kubelet sets cgroup default values or sets it based on pod-level settings:

*   **CPU and Memory**:
    *   Kubelet defaults the container limit to unlimited.
    *   Kubelet ignores the DRA allocation values for setting limits.
*   **HugePages**:
    *   Kubelet defaults the container limit to 0.
    *   Following the same model as CPU and Memory (default to "unlimited") for HugePages breaks because by setting a hard limit of zero, 
        we block the container from consuming any hugepages allocated by the DRA driver. If DRA requests HugePages, Kubelet sets the limit to DRA.

**Container-Level Cgroup Defaults:**

```
CPU Quota = -1 (unlimited)
Memory Limit = unset (unlimited)
HugePages Limit = DRAMapped(hugepages-<size>) + DRAOverhead(hugepages-<size>)
```

**Pod-Level Cgroup Defaults:**

If `PodLevelResources` are explicitly specified (`pod.spec.resources.limits`), the pod-level cgroup enforces those absolute limits. If `PodLevelResources` are not specified, the pod-level cgroup limits inherit the unbounded container defaults, summing up HugePages while deduplicating shared claims:

```
CPU Quota = -1 (unlimited)
Memory Limit = unset (unlimited)
HugePages Limit = DRAMappedUnique(hugepages-<size>) + DRAOverheadUnique(hugepages-<size>)
```

##### Handling Kubelet Disabling Quota with Exclusive CPUs

When a container is allocated exclusive CPUs by Kubelet (using static CPU policy for a Guaranteed QoS pod with integer CPU requests), Kubelet disables 
CPU quota enforcement (`cpu.max = -1`) at both the container and pod levels. This is to prevent unexpected throttling (details in [Issue 70585](https://github.com/kubernetes/kubernetes/issues/70585)). 
With this KEP, this behavior **remains the same**, with the key distinction that Kubelet natively only checks for exclusive CPUs allocated through its standard static CPU policy.

In the case where exclusive CPU allocation is **not** managed by Kubelet (i.e., static CPU policy is disabled) but is instead handled independently by a DRA driver, 
Kubelet lacks visibility into this allocation. Consequently, Kubelet will enforce CFS CPU quotas at both the container and pod levels (if all other conditions for setting quota 
are met — i.e., all containers have limits set or limits are defined at the pod level).

**Risk**: With Kubelet enforcing quotas while the DRA driver allocates exclusive physical CPUs, the workload could experience the same throttling issues as in [issue 70585](https://github.com/kubernetes/kubernetes/issues/70585).
**Current Mitigation**: While the DRA driver can use container-level hooks to override Kubelet's defaults and set the container cgroup to unlimited, it cannot modify Kubelet-managed 
pod-level parent cgroups. To mitigate this, the container requesting exclusive CPUs through the DRA claim can skip setting limits in the container spec. Under this configuration, Kubelet's cgroup manager natively skips quota configuration at both container and pod levels and they remain unlimited (`cpu.max = -1`).
**Potential Long-term Mitigation**: A proper long-term solution would involve a better coordination mechanism between Kubelet and the DRA driver to delegate cgroup enforcement 
responsibilities and avoid having multiple components configuring the same cgroup settings. It needs more design work to establish this handshake mechanism and is currently out of
scope for the alpha stage of this KEP.

#### Enforcement Use Case Walkthroughs

1. Claim + Standard Request

A pod references a shared CPU claim alongside a standard container request and limit.

```yaml
# Pod Spec
spec:
  containers:
  - name: c1
    resources:
      requests: { cpu: "2", memory: "2Gi" }
      limits: { cpu: "4", memory: "4Gi" }
      claims: [{ name: "cpu-claim" }]
  resourceClaims:
  - name: cpu-claim
    resourceClaimName: shared-cpu-claim

# Pod Status
status:
  nodeAllocatableResourceClaimStatuses:
  - resourceClaimName: shared-cpu-claim
    containers: ["c1"]
    mapping:
    - name: cpu
      quantity: "5"
    - name: memory
      quantity: "5Gi"
```

* **Pod Level Cgroup**:
  * `cpu.weight` (CPU Shares): Set based on standard request + DRA mapping: 2 + 5 = **7 CPUs**.
  * `cpu.max` (CPU Quota): Set based on standard limit + DRA (4 + 5) - **9 CPUs**.
  * `memory.max` (Memory Limit): Set based on standard limit + DRA (4 + 5) - **9 GiB**.
* **Container Level Cgroup**:
  * C1
    *   `cpu.weight` (CPU Shares): Set based on standard request - **2 CPUs**.
    *   `cpu.max` (CPU Quota): Set based on standard limit + DRA (4 + 5) - **9 CPUs**.
    *   `memory.max` (Memory Limit): Set based on standard limit + DRA (4 + 5) - **9 GiB**.
*   **Outcome**: 
    The container can burst up to 9 CPUs and 9 GiB memory. If the DRA driver allocates exclusive CPUs, the container has sole access to them. 
    The standard request from the container spec comes from the shared pool, by setting shares based on Spec request of 2 ensures inter-pod fairness during contention.

2. Only Claim, No Standard Request and Limit Specified

A pod references a CPU claim but specifies no standard requests or limits in its Spec.

```yaml
# Pod Spec
spec:
  containers:
  - name: c1
    resources:
      claims: [{ name: "cpu-claim" }]
  resourceClaims:
  - name: cpu-claim
    resourceClaimName: shared-cpu-claim

# Pod Status
status:
  nodeAllocatableResourceClaimStatuses:
  - resourceClaimName: shared-cpu-claim
    containers: ["c1"]
    mapping:
    - name: cpu
      quantity: "5"
```

*   **Pod Level Cgroup**:
  * `cpu.weight` (CPU Shares): Set based on standard request + DRA mapping: 0 + 5 = **5 CPUs**.
  * `cpu.max` (CPU Quota): -1 (Unlimited).
* **Container Level Cgroup**:
  * C1
    * `cpu.weight` (CPU Shares): Defaults to default minimum value (2 shares).
    * `cpu.max` (CPU Quota): -1 (Unlimited).
*   **Outcome**: 
    The container CPU limit remains unlimited as the values are not set in the spec.


3. Multiple Containers Sharing a Claim + Standard Request

Two containers in the same pod share a CPU claim and declare individual standard requests and limits.

```yaml
# Pod Spec
spec:
  containers:
  - name: c1
    resources:
      requests: { cpu: "2", memory: "2Gi" }
      limits: { cpu: "4", memory: "4Gi" }
      claims: [{ name: "shared-claim" }]
  - name: c2
    resources:
      requests: { cpu: "4", memory: "4Gi" }
      limits: { cpu: "8", memory: "8Gi" }
      claims: [{ name: "shared-claim" }]
  resourceClaims:
  - name: shared-claim
    resourceClaimName: shared-cpu-claim

# Pod Status
status:
  nodeAllocatableResourceClaimStatuses:
  - resourceClaimName: shared-cpu-claim
    containers: ["c1", "c2"]
    mapping:
    - name: cpu
      quantity: "5"
    - name: memory
      quantity: "5Gi"
```

* **Pod Level Cgroup**:
  * `cpu.weight` (CPU Shares): Set based on standard requests sum + DRA mapping: (2 + 4) + 5 = **11 CPUs**.
  * `cpu.max` (CPU Quota): Set based on standard limit sum + DRA counted once (4 + 8 + 5) - **17 CPUs**.
  * `memory.max` (Memory Limit): Set based on standard limit sum + DRA counted once (4 + 8 + 5) - **17 GiB**.
* **Container Level C1 Cgroup**:
  * C1
    * `cpu.weight` (CPU Shares): Set based on standard request - **2 CPUs**.
    * `cpu.max` (CPU Quota): Set based on standard limit + DRA (4 + 5) - **9 CPUs**.
    * `memory.max` (Memory Limit): Set based on standard limit + DRA (4 + 5) - **9 GiB**.
  * C2
    * `cpu.weight` (CPU Shares): Set based on standard request - **4**.
    * `cpu.max` (CPU Quota): Set based on standard limit + DRA (8 + 5) - **13 CPUs**.
    * `memory.max` (Memory Limit): Set based on standard limit + DRA (8 + 5) - **13 GiB**.
*   **Outcome**: 
    Both containers can burst up to their limit + claim amount individually. Over-subscription of limits is allowed. 
    However, by counting the shared claim only once at the pod-level cgroup ceiling, Kubelet guarantees that if both 
    C1 and C2 burst simultaneously, they cannot collectively exceed the reserved pod-level budget of 17. 
    If the DRA driver allocates exclusive CPUs, both containers have access to all the claim CPUs, but if there is 
    contention, C2 gets higher priority based on shares.


4. Pod Level Request and Limit + Shared DRA Claim

A pod defines explicit Pod Level Resources, and two containers share a DRA claim without specifying container-level limits.

```yaml
# Pod Spec
spec:
  resources:
    requests: { cpu: "5", memory: "5Gi" }
    limits: { cpu: "5", memory: "5Gi" }
  containers:
  - name: c1
    resources:
      claims: [{ name: "shared-claim" }]
  - name: c2
    resources:
      claims: [{ name: "shared-claim" }]
  resourceClaims:
  - name: shared-claim
    resourceClaimName: shared-cpu-claim

# Pod Status
status:
  nodeAllocatableResourceClaimStatuses:
  - resourceClaimName: shared-cpu-claim
    containers: ["c1", "c2"]
    mapping:
    - name: cpu
      quantity: "5"
    - name: memory
      quantity: "5Gi"
```

* **Pod Level Cgroup**:
  * `cpu.weight` (CPU Shares): Set based on explicit pod request - **5**.
  * `cpu.max` (CPU Quota): Set based on explicit pod limit - **5 CPUs**.
  * `memory.max` (Memory Limit): Set based on explicit pod limit - **5 GiB**.
* **Container Level Cgroup**:
  * C1 & C2
    * `cpu.weight` (CPU Shares): Defaults to minimal value - **2 CPUs**.
    * `cpu.max` (CPU Quota): Inherited from pod-level limit - **5 CPUs**.
    * `memory.max` (Memory Limit): Inherited from pod-level limit - **5 GiB**.
*   **Outcome**: 
    Because the containers do not specify their own limits, they inherit the pod-level limit as their container cgroup maximum value. 
    Pod Level Resources act as the absolute maximum overall budget for the pod, DRA allocations must fit within this budget.

5. Pod Level Request and Limit + Container Requests and Limits + Shared DRA Claims + Sidecar

A pod defines explicit Pod Level Resources, two regular containers share a DRA claim and define individual limits, and a sidecar runs without container limits.

```yaml
# Pod Spec
spec:
  resources:
    requests: { cpu: "8", memory: "8Gi" }
    limits: { cpu: "15", memory: "15Gi" }
  containers:
  - name: c1
    resources:
      requests: { cpu: "2", memory: "2Gi" }
      limits: { cpu: "4", memory: "4Gi" }
      claims: [{ name: "shared-claim" }]
  - name: c2
    resources:
      requests: { cpu: "4", memory: "4Gi" }
      limits: { cpu: "8", memory: "8Gi" }
      claims: [{ name: "shared-claim" }]
  initContainers:
  - name: sidecar
    restartPolicy: Always
    # No resources specified for sidecar
  resourceClaims:
  - name: shared-claim
    resourceClaimName: shared-cpu-claim

# Pod Status
status:
  nodeAllocatableResourceClaimStatuses:
  - resourceClaimName: shared-cpu-claim
    containers: ["c1", "c2"]
    mapping:
    - name: cpu
      quantity: "5"
    - name: memory
      quantity: "5Gi"
```

* **Pod Level Cgroup**:
  * `cpu.weight` (CPU Shares): Set based on explicit pod request - **8**.
  * `cpu.max` (CPU Quota): Set based on explicit pod limit - **15 CPUs**.
  * `memory.max` (Memory Limit): Set based on explicit pod limit - **15 GiB**.
* **Container Level Cgroup**:
  * C1
    * `cpu.weight` (CPU Shares): Set based on standard request - **2 CPUs**.
    * `cpu.max` (CPU Quota): Set based on standard limit + DRA (4 + 5) - **9 CPUs**.
    * `memory.max` (Memory Limit): Set based on standard limit + DRA (4 + 5) - **9 GiB**.
  * C2
    * `cpu.weight` (CPU Shares): Set based on standard request - **4**.
    * `cpu.max` (CPU Quota): Set based on standard limit + DRA (8 + 5) - **13 CPUs**.
    * `memory.max` (Memory Limit): Set based on standard limit + DRA (8 + 5) - **13 GiB**.
  * Sidecar
    * `cpu.weight` (CPU Shares): Defaults to minimal value - **2**.
    * `cpu.max` (CPU Quota): Inherited from pod-level limit - **15 CPUs**.
    * `memory.max` (Memory Limit): Inherited from pod-level limit - **15 GiB**.
*   **Outcome**:
    C1 and C2 calculate their limits by adding the DRA burst to their explicit standard limits (9 and 13 respectively). 
    Because the sidecar omits container limits, it inherits the pod-level limit as its container cgroup maximum value (15). 
    The total aggregate bursting for all containers combined is hard-capped at 15 CPUs.

6. Multiple Containers Sharing a Claim with Host Resource Overhead

Two containers in the same pod share a GPU claim that incurs both flat pod-level and variable container-level CPU/Memory overheads.

```yaml
# Pod Spec
spec:
  containers:
  - name: c1
    resources:
      requests: { cpu: "2", memory: "2Gi" }
      limits: { cpu: "2", memory: "4Gi" }
      claims: [{ name: "shared-gpu" }]
  - name: c2
    resources:
      requests: { cpu: "2", memory: "2Gi" }
      limits: { cpu: "4", memory: "8Gi" }
      claims: [{ name: "shared-gpu" }]
  resourceClaims:
  - name: shared-gpu
    resourceClaimName: shared-gpu-claim

# Pod Status
status:
  nodeAllocatableResourceClaimStatuses:
  - resourceClaimName: shared-gpu-claim
    containers: ["c1", "c2"]
    overhead:
    - name: cpu
      perPod: "1"
      perContainer: "500m"
    - name: memory
      perPod: "1Gi"
      perContainer: "500Mi"
```

* **Pod Level Cgroup**:
  * `cpu.weight` (CPU Shares): Set based on standard requests sum + DRA overhead: 2(C1 Spec request) + 2(C2 Spec request)+ 1(perPod) + 500m * 2 (perContainer for C1 and C2)- **6 CPUs**.
  * `cpu.max` (CPU Quota): Set based on standard limits sum + DRA overhead: 2(C1 Spec limit) + 4(C2 Spec limit) + 1(perPod) + 500m * 2 (perContainer for C1 and C2): **8 CPUs**.
  * `memory.max` (Memory Limit): Set based on standard limits sum + DRA overhead: 4(C1 Spec limit) + 8(C2 Spec limit) + 1Gi(perPod) * 500Mi * 2 (perContainer for C1 and C2): - **14 GiB**.
* **Container Level Cgroup**:
  * C1 
    * `cpu.weight` (CPU Shares): Set based on standard request - **2 CPUs**.
    * `cpu.max` (CPU Quota): Set based on standard limit + container overhead + pod overhead (2 + 0.5 + 1) - **3.5 CPUs**.
    * `memory.max` (Memory Limit): Set based on standard limit + container overhead + pod overhead (4 + 0.5 + 1) - **5.5 GiB**.
  * C2
    * `cpu.weight` (CPU Shares): Set based on standard request - **2 CPUs**.
    * `cpu.max` (CPU Quota): Set based on standard limit + container overhead + pod overhead (4 + 0.5 + 1) - **5.5 CPUs**.
    * `memory.max` (Memory Limit): Set based on standard limit + container overhead + pod overhead (8 + 0.5 + 1) - **9.5 GiB**.
*   **Outcome**:
    Both containers can burst up to their individual cgroup quotas (3.5 and 5.5 CPUs respectively) to accommodate container-specific driver overheads and the flat pod overhead when operating alone. 
    However, if both containers execute overhead tasks simultaneously, their combined CPU and memory footprint is hard-capped at the pod-level parent ceilings (8 CPUs and 14 GiB memory).

#### OOM Score Adjustment with DRA
To manage node stability during Out-Of-Memory (OOM) events, Kubelet applies DRA adjustments while calculating OOM score:
1.  DRA claims are **not** considered when computing the pod's QoS class.
2.  Pods classified as `Guaranteed` or `BestEffort` based on standard Spec continue to receive their static scores (`-997` and `1000`), and does not change based on DRA.
3.  For pods classified as `Burstable`, Kubelet incorporates DRA memory requests to calculate a more protective score.

    ```
      # claimMemory: Total memory quantity allocated to the DRA claim
      # numContainerReferences: Number of containers in the pod referencing this claim
      draMemoryShare = claimMemory / numContainerReferences

      # containerMemReq: Base memory request specified in the container's standard Spec
      # remainingReqPerContainer: Per-container share of unallocated pod-level resources memory request (0 if PodLevelResources is disabled)
      effectiveMemReq = containerMemReq + remainingReqPerContainer + draMemoryShare

      # memoryCapacity: Total physical memory capacity of the host node
      oomScoreAdjust = 1000 - (1000 * effectiveMemReq / memoryCapacity)
    ```
4.  If multiple containers share a single DRA memory claim, Kubelet divides the claim's memory quantity equally among the sharing containers. 
    This equal split is an intentional design simplification as Kubelet cannot dynamically track actual memory distribution between the 
    containers sharing the claim and update the OOM score. This follows the same established pattern with Pod Level Resources (PLR), 
    where pod-level memory requests are distributed equally among containers that omit container-level memory requests.

#### Integration with Memory QoS

Memory QoS [KEP-2570](https://github.com/kubernetes/enhancements/pull/6143) is proposed for beta graduation in v1.37. This configures cgroup v2 memory knobs at both container-level
and pod-level cgroups to manage memory isolation and throttling as follows:

*   **`memory.min`**: Hard memory reclaim protection (configured for Guaranteed QoS pods), mapped from container or pod memory requests.
*   **`memory.low`**: Soft memory reclaim protection (configured for Burstable QoS pods), mapped from container or pod memory requests.
*   **`memory.high`**: Memory throttling threshold (configured for Burstable and BestEffort QoS pods at the container level). If a container's 
      memory usage crosses this threshold, the kernel reclaims memory aggressively and throttles all processes in that cgroup.
*   **`memory.max`**: Hard memory limit (configured at both container and pod levels). If a cgroup's memory usage reaches this limit and cannot be reduced,
    the kernel OOM killer is invoked. Memory QoS does not modify this knob; it remains mapped to standard container or pod memory limits.

##### Current Memory QOS settings
With KEP-2570, cgroup v2 knobs are calculated dynamically based on QoS classes and applied at both container-level and pod-level cgroups:

*   **Guaranteed QoS Pods**:
    *   **Container Level**:
        *   `memory.min` = container request
        *   `memory.low`, `memory.high` = disabled
        *   `memory.max` = container limit
    *   **Pod Level**:
        *   `memory.min` = sum of container requests (or pod-level request if specified)
        *   `memory.low`, `memory.high` = disabled
        *   `memory.max` = sum of container limits (or pod-level limit if specified)
*   **Burstable QoS Pods**:
    *   **Container Level**:
        *   `memory.min` = 0
        *   `memory.low` = container request
        *   `memory.high` = `requests.memory + memory_throttling_factor * (limits.memory - requests.memory)`
            *   `limits.memory` defaults to node allocatable capacity if container limit is unset.
        *   `memory.max` = container limit
    *   **Pod Level**:
        *   `memory.min` = 0
        *   `memory.low` = sum of container requests (or pod-level request if specified)
        *   `memory.high` = disabled
        *   `memory.max` = sum of container limits (or pod-level limit if specified)
*   **BestEffort QoS Pods**:
    *   **Container Level**:
        *   `memory.min`, `memory.low`, `memory.max` = disabled
        *   `memory.high` = `memory_throttling_factor * node_allocatable_capacity`
    *   **Pod Level**:
        *   `memory.min`, `memory.low`, `memory.high`, `memory.max` = disabled

#####  Integration with DRA

Not including DRA allocations in memory cgroup settings triggers the following issues:

1. If `memory.high` is calculated based only on standard Spec limits, the container will suffer kernel reclaim at a threshold far below its actual allocated capacity.
2. If `memory.min` or `memory.low` is computed based strictly on standard Spec requests, the DRA memory allocation will be treated as unprotected, allowing the host kernel to reclaim it aggressively under system pressure.

**Example Scenario (Without Integration)**

Consider a Burstable container with a default memory throttling factor of `0.9`:
*   **Container Spec**: `requests.memory = 1GiB`, `limits.memory = 2GiB`.
*   **DRA allocation**: `5GiB` of direct memory.
*   **Cgroup Configuration with Memory QoS**:
    *   `memory.max`  = `2GiB (Spec Limit) + 5GiB (DRA)` = **`7GiB`**. ([cgroup enforment section](#cgroup-enforcement))
    *   `memory.high`  = `1GiB + 0.9 * (2GiB - 1GiB)` = **`1.9GiB`**.
    *   *Outcome*: Although the workload is allocated 7GiB of memory, its processes are **actively throttled and compressed as soon as memory usage crosses 1.9GiB**.

###### Memory QoS Settings with DRA

**We maintain consistency with the CPU resource model. Kubelet applies a similar strategy for memory cgroups when a pod is allocated memory via a DRA `ResourceClaim`.**
*   Requests are inflated at the pod level and kept uninflated at the container level.
*   Limits are inflated at both pod and container level.

1.  **Container-Level Cgroups**:
    *   Set container `memory.max` using the inflated limit (`limits.memory (Container) + DRA`).
    *   Set container `memory.min` / `memory.low` using the uninflated Spec request.
    *   Set container `memory.high` using the standard Memory QoS formula, but with the **inflated** limit (`limits.memory (Container) + DRA`) used for `memory.max` calculation.
2.  **Pod-Level Cgroups**:
    *   Set pod `memory.max` using sum of container limits + DRA, or pod-level limit if specified.
    *   Set pod `memory.min` / `memory.low` using sum of container requests + DRA, or pod-level request if specified.
    *   *Note: `memory.high` is not set at the pod level with KEP-2570, so nothing changes here.*

**Example Scenario (With Integration)**

Consider the same Burstable container under the integrated CPU-consistent configuration:
*   **Container Specification**: `requests.memory = 1GiB`, `limits.memory = 2GiB`.
*   **DRA Memory claim allocation**: `5GiB` of direct memory.
*   **Cgroup Configuration with Integration (CPU-Consistent)**:
    *   `memory.max` = `2GiB + 5GiB` = **`7GiB`**.
    *   `memory.high` = `1GiB + 0.9 * ((2GiB + 5GiB) - 1GiB)` = **`6.4GiB`**.
    *   *Outcome*: Throttling occurs correctly at 6.4GiB, allowing the container to utilize its full allocated 7GiB memory budget safely.

#### Pod Status Updates

**Current Behavior:**
1.  Allocated Resources (`pod.status.allocatedResources` and `pod.status.containerStatuses[*].allocatedResources`):
    *   Represents the desired intent or reservation. It publishes **only requests**.
    *   Kubelet sets this to match `pod.spec.containers[*].resources.requests` (and `pod.spec.resources.requests`
        at the pod level) upon successful pod admission or after successfully admitting a desired in-place resize.
2.  Resources (`pod.status.resources` and `pod.status.containerStatuses[*].resources`):
    *   Represents the actuated state or reality. It publishes **both requests and limits**.
    *   `pod.status.resources`: Kubelet reads the actual requests and limits enforced on the pod-level cgroup
        directory directly from the host's cgroup filesystem.
    *   `pod.status.containerStatuses[*].resources`: For running containers, Kubelet reads the cgroup state via
        CRI (e.g., CPU shares, quota, and memory limit).

**Behavior with DRA:**
When DRA node allocatable resources are utilized, Kubelet enforces a split model to preserve intent tracking
while accurately reporting actuated cgroup reality:

1.  Allocated Resources:
    *   `pod.status.allocatedResources`: Set to pod-level resources if specified. If not, set to the sum of container-level standard requests and DRA requests.
    *   `pod.status.containerStatuses[*].allocatedResources`: No Change. It continues to be populated strictly based on standard requests in the PodSpec. For the Alpha scope, 
        we do not plan to include container-level allocated resources to include DRA allocations as this field is not currently utilized for scheduler accounting. Shared claims across 
        multiple containers make it difficult to attribute DRA resource allocation at the container status level. It continues to be populated strictly based on standard requests in the PodSpec.
2.  Resources (`pod.status.resources` and `pod.status.containerStatuses[*].resources`):
    *   Requests: Populated by reading the actual cgroup enforcement on the node. If the DRA driver/NRI plugin has adjusted these cgroup settings to actuate DRA resource allocations, 
        the reported requests will reflect those changes. Since memory requests are currently not used to configure cgroup settings, we fallback to report what is requested in the spec and 
        this would now include DRA requests.
    *   Limits:  Populated by reading the actual cgroup enforcement on the node including DRA driver/NRI plugin modifications.

#### Kubelet Internal Resource States

In-place pod resizing and cgroup management introduce four distinct sets of resources that Kubelet tracks for
each pod and container. The following defines these internal resource states and how they interact with DRA
node allocatable resources:

1.  **Desired Resources**:
    *   What the user (or controller) asked for.
    *   Recorded in the API as the spec resources (`.spec.containers[i].resources`).
    *   **Behavior with DRA**: No change. Desired standard resources remain in `.spec.containers[i].resources`,
        while DRA node allocatable resource claims are requested separately in `.spec.resourceClaims`.
2.  **Allocated Resources**:
    *   The resources that the Kubelet admitted, and intends to actuate.
    *   Persisted locally on the node in a checkpoint file.
    *   Used to update the pod status (`.status.allocatedResources` and `.status.containerStatuses[i].allocatedResources`).
    *   **Behavior with DRA**: No change. The node's internal `allocated` checkpoint remains strictly limited to standard Spec requests and limits at both the pod and container levels. DRA allocations are completely excluded.
        *   Note: At the pod level, the API representation (`pod.status.allocatedResources`) diverges from this internal state as the checkpoint does not include DRA requests. 
            The pod status field accurately represents the total resource reservation including DRA, while the checkpoint remains spec-only to prevent Kubelet from triggering 
            infinite resizing loops when comparing spec with the checkpointed state.
3.  **Actuated Resources**:
    *   The resource configuration that the Kubelet passed to the runtime to actuate.
    *   Not reported in the API.
    *   Persisted locally on the node in a checkpoint file.
    *   **Behavior with DRA**: No change. To ensure steady-state reconciliation loops (`computePodResizeAction`)
        do not trigger unnecessary CRI updates or cgroup resets, Kubelet maintains the internal `actuated`
        checkpoint strictly limited to standard Spec requests and limits. DRA allocations are excluded from
        the checkpoint.
    *   **Divergence**: This design introduces an intentional divergence where kubelet's `actuated` checkpoint excludes DRA allocations, diverging from the 
        actual cgroup settings enforced on the node. This prevents the steady-state reconciliation loops from seeing a difference between `.spec` and cgroups 
        ensuring that we do not revert the DRA included cgroup settings.
4.  **Actual Resources**:
    *   The actual resource configuration the containers are running with, reported by the runtime, typically
        read directly from the cgroup configuration.
    *   Reported in the API via the `.status.containerStatuses[i].resources` field.
    *   **Behavior with DRA**: During cgroup generation (`generateLinuxContainerResources` and
        `ResourceConfigForPod`), Kubelet dynamically inflates limits by summing standard Spec limits and DRA
        allocations read from `pod.status.nodeAllocatableResourceClaimStatuses`. Therefore, the actual limits
        reported in `.status.resources.limits` and `containerStatuses[*].resources.limits` natively reflect
        the combined standard and DRA resources based on the defined [cgroup enforcement](#cgroup-enforcement)
        rules. Both initial pod creation and resize actuation share the exact same cgroup configuration code. 
        Because these paths are identical, in-place vertical scaling preserves and applies the same DRA inflated
        cgroup values during actuation.

#### Integration with In-Place Pod Vertical Scaling

In Alpha 1, prior to introducing Kubelet cgroup enforcement, API validation was added in
`pkg/apis/core/validation/validation.go` to block In-Place Pod Resizing (IPPR) for pods utilizing DRA node
allocatable resources. Now that Kubelet cgroup enforcement is introduced, this validation restriction can
be safely removed. At the API layer, resizing operations target standard Spec requests and limits in `pod.spec`,
while DRA `ResourceClaim` allocations remain immutable.

In the control plane, when the scheduler computes a resizing pod's footprint, because `PodRequests()` aggregates
the DRA allocations from `pod.status.nodeAllocatableResourceClaimStatuses`, the scheduler accurately tracks
total resource footprint during resize.

On the node, when Kubelet evaluates whether a resize fits on the node (`canAdmitPod`), the Allocation Manager
computes the resource footprint including DRA. When actuating the admitted resize at the container level we sum
the newly resized standard Spec limits with the constant DRA resources, passing the combined limits to CRI.

### Kubelet Admission Control

The Kubelet has its own admission check
([AdmissionCheck](https://github.com/kubernetes/kubernetes/blob/4925c6bea44efd05082cbe03d02409e0e7201252/pkg/kubelet/lifecycle/predicate.go#L436))
to ensure a pod can run on the node, even after the scheduler has placed it. It utilizes the `PodRequests()` function from
the `k8s.io/component-helpers/resource`. This shared helper has been enhanced to support unified accounting. When
calculating a pod's requirements, it aggregates the standard requests from pod Spec with the DRA allocations recorded in
`pod.status.nodeAllocatableResourceClaimStatuses`. Because the scheduler populates this status field during the PreBind stage, the
Kubelet validates the pod's comprehensive resource footprint. 

This admission-time lookup reads directly from the **`pod.status.nodeAllocatableResourceClaimStatuses`** API field. This allows Kubelet's 
Vertical Scaling Admission Controller (`canAdmitPod` inside `AllocationManager`) to accurately evaluate resource-fit during vertical resizing
without needing to persist DRA allocations in Kubelet's local disk checkpoints (`allocatedState` or `actuatedState`). Because DRA allocations 
are immutable after scheduling, Kubelet can bypass the local checkpoints for DRA evaluation, relying instead on this API status field as
the source of truth.

### Future Enhancements

#### Kube-Scheduler Scoring and Resource Quota

##### Scoring

In the current Alpha implementation, unified scoring for node allocatable resources is only partially achieved:
*   For existing (assumed) pods on the node, The `NodeResourcesFit` plugin's scoring accurately accounts for their combined footprint. 
    This is because the scheduler's `Assume` stage updates `NodeInfo.Requested` with both standard Spec requests and dynamic DRA status claim allocations
    for all previously assumed pods on the node.
*   For the incoming pod being scored, scoring in `NodeResourcesFit` only considers CPU and Memory requests defined directly in the pod's Spec. It does 
    not account for the incoming pod's DRA based allocations.

The root cause of this limitation lies in the sequential execution and encapsulation between plugins and the scheduler's lifecycle stages:
1.  **Filter Stage (`DynamicResources` Plugin)**: DRA device allocations are resolved, and the dynamic CPU/Memory resource overheads are calculated for each candidate node.
    These node-specific allocations are stored transiently in the in-memory `CycleState`.
2.  **Score Stage (`NodeResourcesFit` Plugin)**: Nodes are scored using CPU/Memory spreading or packing algorithms. Although the allocations exist in `CycleState` at this point
    `NodeResourcesFit` does not read them because:
    *   `PreScore` calculates the pod's resource footprint once for the entire cycle, to be able to include DRA based allocations, the `NodeResourcesFit` plugin should read 
    `DynamicResources`' internal state which is challenging and introduces coupling between plugins.
3.  **PreBind Stage (`DynamicResources` Plugin)**: Only after a node is selected and reserved does the scheduler patch the **`Pod.Status`** in the API server to persist the 
    `NodeAllocatableResourceClaimStatuses` field.

**Potential Options to Explore:**
To achieve fully unified scoring in future milestones, we need to explore `CycleState` sharing between scheduling plugins. Alternatively, we can continue scoring strictly based
on the pod's Spec requests (our default fallback).
*   **Pros:** Keeps core scheduler plugins (`NodeResourcesFit` and `DynamicResources`) completely decoupled and avoids cross-plugin sharing.
*   **Cons:** Degrades ranking quality for pods with large DRA allocations. We might pack a pod onto a node that appears to have low occupancy but is actually heavily committed due to
    DRA claims, though the `Filter` stage still strictly guarantees the node has sufficient physical capacity.

##### Quota

Currently, `ResourceQuota` only accounts for resources defined in the standard `pod.spec` requests/limits. Including node allocatable resources allocated via DRA `ResourceClaims` in `ResourceQuota` enforcement is not included in the initial Alpha scope.

Two primary implementation options are proposed for future milestones:

**Option A: Separate Standard Requests and DRA-Based Quotas**

In this option, standard compute quotas (`requests.cpu`, `requests.memory`) and DRA-based device quotas are kept entirely separate. 
A separate namespace quota is created to track device counts for each `DeviceClass` (e.g., using keys like `<deviceclass>.deviceclass.resource.k8s.io/devices`). Standard CPU and Memory requests defined in the pod Spec are charged against the traditional namespace compute quotas, while DRA-allocated CPU or Memory are evaluated and charged independently as custom resources. This is how things work currently with standard DRA-based quota.
* **Pros:** Simple, highly decoupled, and matches the current standard DRA quota design. Avoids complex integration or synchronization between standard ResourceQuota admission and scheduler-driven DRA allocation states.
* **Cons:** Fragmented quota tracking for compute. Users cannot define a single, unified `requests.cpu` ceiling that restricts both direct pod spec cpu requests and dynamic DRA-managed exclusive CPU claims.

**Option B: Quota Enforcement in the Scheduler**

In this option, standard compute resource quotas (e.g., `requests.cpu`, `requests.memory`) are unified to account for both pod spec requests and DRA-allocated node allocatable resources, with the quota validation and enforcement executed by the scheduler. This can only happen during the scheduling cycle because
  - The `ResourceClaim` can be created asynchronously after the Pod passes admission.
  - If a claim uses prioritized list (e.g., GPU or CPU), the selected resource type is only resolved by the scheduler during node selection.
  - The exact resource footprint depends on the target node's topology and driver configurations, which are resolved after scheduling.

Once the scheduler selects a node and resolves DRA claim allocations, it sums the pod spec standard requests with the newly calculated DRA cgroup-burst resource requests. It evaluates this unified footprint against the remaining namespace `ResourceQuota`. If the computed usage exceeds the remaining quota, the node is filtered out during the scheduling cycle.

* **Pros:** Provides a single quota ceiling for CPU and Memory, regardless of whether they are requested in the PodSpec or allocated dynamically via DRA claims.
* **Cons:** Pods exceeding quota are accepted by the API server and remain in a `Pending` state indefinitely (emitting `FailedScheduling` events) instead of being synchronously rejected at creation time. Requires state synchronization and a custom namespace quota cache inside the scheduler, introducing risk of split-brain quota enforcement.

Integrating DRA node allocatable resources would involve ensuring this helper is called with the appropriate options to include `pod.status.nodeAllocatableResourceClaimStatuses`. The implications of this change need to be discussed.

#### Pass Allocation Details from Driver to Kubelet

Currently, Kubelet is blind to the type of resource allocation performed by the DRA driver. Passing this information from the DRA driver to Kubelet enables better 
coordination and node-level cgroup enforcement. This can be solved by adding an `AllocationType` field inside `NodeAllocatableResources` and propagating 
it all the way to the `pod.status` in the API.

The two types of allocation that can be configured are:
1.  **Exclusive**: Dedicates and physically isolates the resource capacity (e.g., pinning CPUs by setting `cpuset.cpus` in the DRA driver).
2.  **Shared**: Binds resources to a specific domain that is also shared with other containers not referencing the same claim 
    (e.g., binding memory to a specific NUMA node, or binding CPUs to a socket that is also shared by other workloads).

##### API Changes

**Device Spec (`k8s.io/api/resource/v1/types.go`):**
```go
// AllocationType specifies the isolation and scheduling strategy.
type AllocationType string

const (
    // AllocationTypeShared indicates the resource is allocated from a shared general pool.
    AllocationTypeShared AllocationType = "Shared"

    // AllocationTypeExclusive indicates the resource represents dedicated, physically isolated capacity (e.g., dedicated cores).
    AllocationTypeExclusive AllocationType = "Exclusive"
)

type Device struct {
    // existing fields
    // +optional
    NodeAllocatableResources map[v1.ResourceName]NodeAllocatableResource
}

type NodeAllocatableResource struct {
    Mapping *NodeAllocatableMapping
    Overhead *NodeAllocatableOverhead
}

type NodeAllocatableMapping struct {
    CapacityKey          *QualifiedName
    CapacityMultiplier *resource.Quantity 
    
    // AllocationType describes whether the resources represent exclusive or shared capacity.
    // If omitted, it defaults to AllocationTypeShared.
    // +optional
    AllocationType *AllocationType `json:"allocationType,omitempty" protobuf:"bytes,3,opt,name=allocationType"`
}
```

**Pod Status API (`k8s.io/api/core/v1/types.go`):**
```go
type PodStatus struct {
  // ... existing fields ...
  // +featureGate=DRANodeAllocatableResources
  // +optional
  NodeAllocatableResourceClaimStatuses []NodeAllocatableResourceClaimStatus `json:"nodeAllocatableResourceClaimStatuses,omitempty" protobuf:"bytes,25,rep,name=nodeAllocatableResourceClaimStatuses"`
}

type NodeAllocatableResourceClaimStatus struct {
  ResourceClaimName string
  Containers []string
  Mapping []NodeAllocatableMappedResources
  Overhead []NodeAllocatableOverheadResources
}

type NodeAllocatableMappedResources struct {
  Name           ResourceName      `json:"name" protobuf:"bytes,1,opt,name=name"`
  Quantity       resource.Quantity `json:"quantity" protobuf:"bytes,2,opt,name=quantity"`
  
  // AllocationType is resolved from `device.nodeAllocatableResources.mapping.allocationType` and added here.
  // +required
  AllocationType AllocationType    `json:"allocationType" protobuf:"bytes,3,opt,name=allocationType"`
}
```

**Example:** 

* ResourceSlice

  ```yaml
  apiVersion: resource.k8s.io/v1
  kind: ResourceSlice
  metadata:
    name: native-resource-slice
  spec:
    driver: dra.cpu.com
    nodeName: my-node
    pool: { name: "node-pool", generation: 1, resourceSliceCount: 1 }
    devices:
    - name: socket0
      attributes:
        "dra.example.com/type": "socket"
      allowMultipleAllocations: true
      capacity:
        "dra.example.com/cores": "64"
      nodeAllocatableResources: 
        cpu:
          mapping:
            capacityKey: "dra.example.com/cores"
            capacityMultiplier: "2"
            allocationType: "Exclusive"
  ```

* Pod Status

  ```json
  "nodeAllocatableResourceClaimStatuses": [
    {
      "resourceClaimName": "cpu-claim",
      "containers": ["worker"],
      "direct": [
        {
          "name": "cpu",
          "quantity": "4",
          "allocationType": "Exclusive"
        }
      ]
    }
  ]
  ```

##### Node Cgroup Enforcement

###### Pod Level Cgroup

*   **CPU Limits**: Set based on standard limits sum + unique direct mapped resources (refer to the [Pod-Level Cgroup Limits](#pod-level-cgroup-limits) calculation section above).
*   **CPU Requests**:
    *   In the current alpha implementation, CPU shares are configured strictly based on the standard pod Spec requests sum.
    *   Under the proposed `AllocationType`-aware future design:
        *   **Exclusive Mode (`AllocationType: Exclusive`)**: Shares remain configured strictly based on the standard pod Spec requests sum. Since the DRA driver dedicates and physically 
            isolates CPU capacity to the container (e.g., cpuset pinning), the workloads do not experience scheduling contention with other co-located pods on the node, making CFS shares inflation unnecessary. 
            Setting shared based on exclusive resouces reserved by the DRA driver also gives the container/pod unfair advantage in the shared resource pool during resource contention.
        *   **Shared Mode (`AllocationType: Shared`)**: Shares are inflated by adding the standard pod Spec requests sum and the resolved direct CPU quantity mapped by the claim 
            (obtained from `pod.status.nodeAllocatableResourceClaimStatuses[].mapping[].quantity`). Since the workload competes inside the node's general shared resource pool, this inflation guarantees 
            that the pod as a whole obtains its scheduler-reserved resources under contention.
*   **Memory Limits**: Set based on standard limits sum + unique direct mapped memory resources (refer to the [Pod-Level Cgroup Limits](#pod-level-cgroup-limits) calculation section above).
*   **Memory Requests**: Currently in kubelet, we do not set memory cgroups based on requests.

###### Container Level Cgroup

*   **CPU Limits**: Set based on standard limits + direct resources + container overhead + pod overhead (refer to the [Container-Level Cgroup Limits](#container-level-cgroup-limits) calculation section above).
*   **CPU Requests**: Configured strictly based on the container's standard Spec request (`pod.spec.containers[].resources.requests.cpu`).
*   **Why we do not set container-level shares based on DRA CPU**:
    * By setting the inflated CPU weight strictly at the pod-level parent cgroup, Kubelet guarantees correct resource priority relative to other pods in the cgroup hierarchy during resource contention. 
      Inside the pod's cgroup tree, sibling containers time-share the pod's aggregate budget proportionally based on their relative standard Spec requests.
    * When multiple containers in the same pod reference the same claim, dividing the claim's CPU shares across container-level cgroups introduces complexity.
*   **Memory Limits**: Set based on standard limits + direct resources + container overhead + pod overhead (refer to the [Container-Level Cgroup Limits](#container-level-cgroup-limits) calculation section above).
*   **Memory Requests**: Currently in kubelet, we do not set memory cgroups based on requests.

###### Enforcement Example:

```yaml
# Pod Spec
spec:
  containers:
  - name: c1
    resources:
      requests: { cpu: "2", memory: "2Gi" }
      limits: { cpu: "4", memory: "4Gi" }
      claims: [{ name: "shared-claim" }]
  - name: c2
    resources:
      requests: { cpu: "4", memory: "4Gi" }
      limits: { cpu: "8", memory: "8Gi" }
      claims: [{ name: "shared-claim" }]
  resourceClaims:
  - name: shared-claim
    resourceClaimName: shared-cpu-claim
```

Depending on the allocation mapping type, cgroup parameters are actuated as follows:

**1. Shared Mode (AllocationType = Shared)**

In this case, the DRA driver allocates from a general node pool, so the status contains:
```yaml
# Pod Status
status:
  nodeAllocatableResourceClaimStatuses:
  - resourceClaimName: shared-cpu-claim
    containers: ["c1", "c2"]
    mapping:
    - name: cpu
      quantity: "5"
      allocationType: Shared
```

Cgroup bounds are set as:

*   **Pod Level Cgroup**:
    *   `cpu.weight` (CPU Shares): Inflated based on standard requests sum + DRA direct CPU quantity (2 + 4 + 5): **11 CPUs**.
    *   `cpu.max` (CPU Quota): Set based on [Pod-Level Cgroup Limits](#pod-level-cgroup-limits): 17 CPUs**.
    *   `memory.max` (Memory Limit): Set based on [Pod-Level Cgroup Limits](#pod-level-cgroup-limits): 12 GiB.
*   **Container Level C1 Cgroup**:
    *   `cpu.weight` (CPU Shares): Configured strictly based on standard container request: 2 CPUs.
    *   `cpu.max` (CPU Quota): Set based on [Container-Level Cgroup Limits](#container-level-cgroup-limits): 9 CPUs.
    *   `memory.max` (Memory Limit): Set based on [Container-Level Cgroup Limits](#container-level-cgroup-limits): 4 GiB.
*   **Container Level C2 Cgroup**:
    *   `cpu.weight` (CPU Shares): Configured strictly based on standard container request: 4 CPUs.
    *   `cpu.max` (CPU Quota): Set based on [Container-Level Cgroup Limits](#container-level-cgroup-limits): 13 CPUs.
    *   `memory.max` (Memory Limit): Set based on [Container-Level Cgroup Limits](#container-level-cgroup-limits): 8 GiB.

**2. Exclusive Mode (AllocationType = Exclusive)**

```yaml
# Pod Status
status:
  nodeAllocatableResourceClaimStatuses:
  - resourceClaimName: shared-cpu-claim
    containers: ["c1", "c2"]
    mapping:
    - name: cpu
      quantity: "5"
      allocationType: Exclusive
```

Cgroup bounds are set as:
*   **Pod Level Cgroup**:
    *   `cpu.weight` (CPU Shares): Kept uninflated, configured strictly based on standard requests sum (2 + 4): **6 CPUs**.
    *   `cpu.max` (CPU Quota): Set based on [Pod-Level Cgroup Limits](#pod-level-cgroup-limits): 17 CPUs.
    *   `memory.max` (Memory Limit): Set based on [Pod-Level Cgroup Limits](#pod-level-cgroup-limits): 12 GiB.
*   **Container Level C1 Cgroup**:
    *   `cpu.weight` (CPU Shares): Configured strictly based on standard container request: 2 CPUs.
    *   `cpu.max` (CPU Quota): Set based on [Container-Level Cgroup Limits](#container-level-cgroup-limits): 9 CPUs.
    *   `memory.max` (Memory Limit): Set based on [Container-Level Cgroup Limits](#container-level-cgroup-limits): 4 GiB.
*   **Container Level C2 Cgroup**:
    *   `cpu.weight` (CPU Shares): Configured strictly based on standard container request: 4 CPUs.
    *   `cpu.max` (CPU Quota): Set based on [Container-Level Cgroup Limits](#container-level-cgroup-limits): 13 CPUs.
    *   `memory.max` (Memory Limit): Set based on [Container-Level Cgroup Limits](#container-level-cgroup-limits): 8 GiB.


### Test Plan

[x] I/we understand the owners of the involved components may require updates to
existing tests to make this code solid enough prior to committing the changes necessary
to implement this enhancement.

##### Prerequisite testing updates

<!--
Based on reviewers feedback describe what additional tests need to be added prior
implementing this enhancement to ensure the enhancements have also solid foundations.
-->

##### Unit tests

<!--
In principle every added code should have complete unit test coverage, so providing
the exact set of tests will not bring additional value.
However, if complete unit test coverage is not possible, explain the reason of it
together with explanation why this is acceptable.
-->

<!--
Additionally, for Alpha try to enumerate the core package you will be touching
to implement this enhancement and provide the current unit coverage for those
in the form of:
- <package>: <date> - <current test coverage>
The data can be easily read from:
https://testgrid.k8s.io/sig-testing-canaries#ci-kubernetes-coverage-unit

This can inform certain test coverage improvements that we want to do before
extending the production code to implement this enhancement.
-->

Unit tests will be added for all new and modified logic within the `kube-scheduler` and `kubelet` components.

-   Ensuring the new fields in `Device` and `PodStatus` are validated correctly.
-   Scheduler Plugin Logic (`NodeResourcesFit`, `DynamicResources`):
    -   Verifying the correct deferral of node allocatable resource checks in `NodeResourcesFit`.
    -   Verify the accurate calculation of a pod's total node allocatable resource demand across both `Direct`
        mappings (device counts or capacity key drawdowns) and `Overhead` mappings (per-pod or per-reference
        auxiliary overheads).
    -   Verify that inter-pod sharing of `Direct` mapped device claims is correctly blocked during the Filter stage,
        while inter-pod sharing of `Overhead`-mapped claims is permitted.
    -   Validating that `pod.status.nodeAllocatableResourceClaimStatuses` is updated correctly.
-   Scheduler Framework:
    -   Verify `NodeInfo` cache updates correctly in the `Assume` stage and reflects resources allocated to node allocatable resource claims.
    -   Verify that when a pod using DRA node allocatable resources is deleted, the resources are correctly released 
        and become available for other pods in the scheduler's cache.
-   Component helper (`k8s.io/component-helpers/resource`)
    -   Testing the `PodRequests` helper function's updated logic to include DRA node allocatable resources.
        -   Ensure existing calculations for pods without DRA claims or PLR remain correct, properly aggregating init 
            and regular container requests.
        -   Verify pod level resources when specified for a resource, continues to take precedence over per-container requests,
            include node allocatable claim requests.
        -   Verify that the node allocatable resources from `pod.status.nodeAllocatableResourceClaimStatuses` are correctly added to the pod's effective standard resource requests.
        -   Test that existing logic for different `PodResourcesOptions` (e.g., `ExcludeOverhead`, `SkipPodLevelResources`) continues to 
            work as expected when DRA node allocatable resources are present, including correct handling of `pod.spec.overhead`.
-   Kubelet Admission Check
    -   Verifying that the admission check correctly uses the DRA node allocatable resource from the pod's `status.nodeAllocatableResourceClaimStatuses` field.
-   Kubelet Cgroup Enforcement (`pkg/kubelet/kuberuntime/kuberuntime_container_linux.go`, `pkg/kubelet/cm/helpers_linux.go`):
    -   Verify that container and pod-level CPU quota and memory limit correctly sum standard Spec limits and DRA allocations from `pod.status.nodeAllocatableResourceClaimStatuses`.
    -   Verify that CPU shares remain purely based on standard Spec requests.
    -   Verify that container OOM score adjustments (`oom_score_adj`) correctly incorporate DRA memory status allocations 
        for Burstable pods using equal-splitting across referencing containers.
    -   Verify cgroup generation across multiple test cases involving Pod Level Resources, including containers
        specifying their own limits and containers inheriting pod-level ceilings without limits.
    -   Verify that if container-level HugePages limits are omitted, Kubelet sets the limit to match DRA
        allocations.
-   Kubelet Allocation Manager (`pkg/kubelet/allocation/allocation_manager.go`):
    -   Verify that during steady-state reconciliation loops, Kubelet maintains the `allocated`
        checkpoint strictly limited to standard Spec requests and limits, while correctly incorporating DRA
        allocations when evaluating node capacity during pod admission and resize checks.

<!--
Generated with:
go test -cover ./pkg/scheduler/framework/plugins/dynamicresources ./pkg/scheduler/framework/plugins/noderesources ./pkg/scheduler ./pkg/scheduler/framework ./staging/src/k8s.io/component-helpers/resource ./pkg/kubelet/kuberuntime ./pkg/kubelet/cm ./pkg/kubelet/allocation
-->

-  pkg/scheduler/framework/plugins/dynamicresources: 20260517 - 82.5%
-  pkg/scheduler/framework/plugins/noderesources: 20260517 - 89.1%
-  pkg/scheduler/schedule_one.go: 20260517 - 76.8%
-  pkg/scheduler/framework/types.go: 20260517 - 73.0%
-  pkg/scheduler/eventhandlers.go: 20260517 - 76.8%
-  staging/src/k8s.io/component-helpers/resource/helpers.go: 20260517 - 82.7%
-  pkg/kubelet/kuberuntime: 20260517 - 70.9%
-  pkg/kubelet/cm: 20260517 - 24.3%
-  pkg/kubelet/allocation: 20260517 - 82.8%


##### Integration tests

Integration tests will be added in `test/integration/dynamicresource` to cover the end-to-end scheduling flow:

**Kube-Scheduler:**
-   Tests to ensure correct interaction between `NodeResourcesFit` and `DynamicResources` plugins.
-   Test that the scheduler's internal cache (`NodeInfo.Requested`) is accurately updated to reflect 
    the resources consumed by pods with DRA node allocatable resource claims.
-   Ensure that resources are correctly released in the scheduler cache when a pod with DRA node allocatable resource claims is deleted.
-   Validate that fungible claims resulting in different node allocatable resource footprints are accounted for correctly on a per-node basis.
-   Verify that the scheduler correctly enforces inter-pod sharing restrictions, blocking pods that attempt to
    share `Direct`-mapped devices.
-   Tests to validate the `pod.status.nodeAllocatableResourceClaimStatuses` is populated correctly and the kubelet
    admission check correctly computes the effective pod resource request.

**Kubelet:**
-   Test that the Kubelet's admission handler correctly factors in the node allocatable resources specified in `pod.status.nodeAllocatableResourceClaimStatuses` 
    when deciding whether to admit a pod.
-   Test that Kubelet correctly generates Linux cgroup configurations summing standard Spec limits and DRA allocations.

##### e2e tests

E2E tests will be added to `test/e2e/dra`:

-   Verify these pods are scheduled onto nodes with sufficient capacity, considering both the pod's standard requests and the DRA-added node allocatable resources.
    These tests should cover various DRA modeling scenarios:
    -   Node allocatable resources as individual devices. 
    -   Node allocatable resources as consumable capacity from a pool.
    -   Node allocatable resources from partitionable devices.
    -   Auxiliary node allocatable resources required by other devices (e.g., additional memory for an accelerator).
    -   Fungible claims involving node allocatable resources.
-   Verify that Kubelet enforces correct cgroup limits on running containers without kernel throttling or OOM kills, and applies correct OOM score adjustments.

### Graduation Criteria

#### Alpha

-   Feature implemented behind the `DRANodeAllocatableResources` feature gate and disabled by default.
-   Core API changes for `Device` and `PodStatus` introduced.
-   Kube-Scheduler:
    *   The `DynamicResources` plugin is updated to calculate and enforce node resource fit based on standard requests and node allocatable resource claims.
    *   The scheduler's internal cache update logic is enhanced to incorporate DRA node allocatable resource allocations.
-   `k8s.io/component-helpers/resource` shared library is enhanced to compute effective pod resource footprint.
-   The Kubelet's admission handler is updated to consider node allocatable resource claims in `Pod.Status`.
-   API validation restriction implemented in `pkg/apis/core/validation/validation.go` blocking In-Place Pod
    Resizing for pods utilizing DRA node allocatable resources.
-   All unit and integration tests outlined in the Test Plan are implemented and verified.

#### Alpha2

-   Enhance Kubelet to utilize `pod.status.nodeAllocatableResourceClaimStatuses` for cgroup management
    and OOM score adjustments.
-   Support use cases where DRA directly models node allocatable resources (such as exclusive CPU allocation or
    consumable capacity pools) as well as use cases where specialized devices declare auxiliary node allocatable
    resource dependencies (such as accelerator host memory overhead).
-   Remove API validation restrictions in `pkg/apis/core/validation/validation.go` to allow resizing standard Spec
    resources for pods utilizing DRA node allocatable resources.
-   Add E2E tests for kube-scheduler and Kubelet changes, including correct cgroup enforcement and OOM score
    adjustments across various device mapping models.

#### Beta

-   At least one DRA driver has integrated the API extensions and successfully validated the node allocatable resource mapping in ResourceSlice.
-   Integrate DRA-allocated node allocatable resources into the Kubelet Eviction Manager to ensure accurate eviction decisions during node pressure.
-   Support unified ResourceQuota and LimitRange enforcement for DRA-allocated node allocatable resources.
-   Support node allocatable resource mappings to ephemeral storage.

### Upgrade / Downgrade Strategy

-   **Upgrade:** Enabling the feature gate on an existing cluster is safe. The new accounting logic will apply
    to any newly scheduled pods or pods that are re-scheduled. Existing pods with node allocatable resource claims would 
    continue to run, but their claim request will not be reflected in the scheduler's `NodeInfo` cache as these 
    pods lack `pod.status.nodeAllocatableResourceClaimStatuses` field. On the node, Kubelet will continue to 
    enforce cgroups based solely on standard Spec limits for existing pods. To fully resynchronize control-plane 
    accounting and node cgroup limit inflation, the pods with node allocatable resource claims must be restarted.

-   **Downgrade:** Disabling the feature gate requires a kube-scheduler and kubelet restart. Upon startup, the
    scheduler rebuilds the NodeInfo cache without considering DRA node allocatable resources. The scheduler's view
    of resource usage for existing pods will be incomplete (underestimated) as it does not consider claim-based
    requests, potentially leading to oversubscription of the node if new pods are scheduled. On the node, Kubelet 
    will not dynamically trigger cgroup updates during regular sync loops. Running containers will continue to operate
    with their existing DRA-included cgroup limits. Kubelet will ignore `pod.status.nodeAllocatableResourceClaimStatuses` 
    and revert cgroup limits to standard Spec only limits upon container restarts.


### Version Skew Strategy

-   **API Skew**: An older scheduler will not understand the new API fields. If `ResourceSlice` or
    `Pod` objects contain the new fields, they will be ignored.
-   **New Scheduler, Older Kubelet**: 
    - To proactively prevent pods utilizing DRA node allocatable resources from landing on older Kubelets that do not enforce 
      cgroup restriction based on DRA, the scheduler must use the [Node Declared Features framework](https://github.com/kubernetes/enhancements/tree/master/keps/sig-node/5328-node-declared-features).
    - A new declared feature `DRANodeAllocatableResources` is registered in `node.status.declaredFeatures`.
      The API server admission controller uses this framework to validate and reject `ResourceSlice` objects that contain `nodeAllocatableResources` mappings if they target a node that lacks support for this feature.

## Production Readiness Review Questionnaire

### Feature Enablement and Rollback

###### How can this feature be enabled / disabled in a live cluster?

- [x] Feature gate (also fill in values in `kep.yaml`)
  - Feature gate name: `DRANodeAllocatableResources`
  - Components depending on the feature gate: `kube-scheduler`, `kubelet`, `kube-apiserver`.

###### Does enabling the feature change any default behavior?

No. This feature only takes effect if users create Pods that request node allocatable resources via
`pod.spec.resourceClaims` and DRA drivers are installed and configured to expose node allocatable resources via
`nodeAllocatableResources` in `ResourceSlice` objects. Existing pods are unaffected.

###### Can the feature be disabled once it has been enabled (i.e. can we roll back the enablement)?

Yes. Disabling the feature gate `DRANodeAllocatableResources` will prevent the scheduler from performing the unified accounting. 
Pods already scheduled using DRA node allocatable resource accounting will continue to run. However, when new pods are scheduled 
while the gate is disabled, any node allocatable resources specified in their DRA claims will not be considered by the scheduler.
This can lead to node oversubscription as the scheduler's view of available resources on the node will be incomplete. 

On nodes, running containers will continue to operate with their existing DRA included cgroup limits. Kubelet will only
ignore `pod.status.nodeAllocatableResourceClaimStatuses` and revert cgroup limits back to standard Spec
limits upon subsequent container restarts or recreations. 

###### What happens if we reenable the feature if it was previously rolled back?

The scheduler will resume its unified accounting logic for pods with DRA node allocatable resource claims. API
validation for the new fields will be re-enabled. The `NodeInfo` cache may be incorrect as it's not
retroactively updated to consider node allocatable resource claims for previously scheduled pods. This inconsistent 
state would persist until kube-scheduler restarts or all pods with node allocatable resource claims are restarted.
On nodes, running containers that were started while the gate was disabled will remain at standard Spec limits. To fully 
resynchronize control-plane accounting and node cgroup limit inflation, pods utilizing DRA node allocatable claims must be restarted.

###### Are there any tests for feature enablement/disablement?

Unit tests in `kube-scheduler`, `kubelet`, and `kube-apiserver` will verify the behavior of the scheduler plugins
(`NodeResourcesFit`, `DynamicResources`), Kubelet cgroup enforcement, and
API validation with the feature gate enabled and disabled.

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

###### What specific metrics should inform a rollback?

<!--
What signals should users be paying attention to when the feature is young
that might indicate a serious problem?
-->

###### Were upgrade and rollback tested? Was the upgrade->downgrade->upgrade path tested?

<!--
Describe manual testing that was done and the outcomes.
Longer term, we may want to require automated upgrade/rollback tests, but we
are missing a bunch of machinery and tooling and can't do that now.
-->

###### Is the rollout accompanied by any deprecations and/or removals of features, APIs, fields of API types, flags, etc.?

<!--
Even if applying deprecation policies, they may still surprise some users.
-->

### Monitoring Requirements

<!--
This section must be completed when targeting beta to a release.

For GA, this section is required: approvers should be able to confirm the
previous answers based on experience in the field.
-->

###### How can an operator determine if the feature is in use by workloads?

- `ResourceSlice` objects containing `Device` entries with `nodeAllocatableResources`.
- Pods with `status.nodeAllocatableResourceClaimStatuses` populated.

###### How can someone using this feature know that it is working for their instance?

<!--
For instance, if this is a pod-related feature, it should be possible to determine if the feature is functioning properly
for each individual pod.
Pick one more of these and delete the rest.
Please describe all items visible to end users below with sufficient detail so that they can verify correct enablement
and operation of this feature.
Recall that end users cannot usually observe component logs or access metrics.
-->

- [ ] Events
  - Event Reason: 
- [x] API .status
    - Other field: pod.status.nodeAllocatableResourceClaimStatuses
    - Details: Pods referencing node allocatable resource claims should have the pod status updated with `nodeAllocatableResourceClaimStatuses`.
- [ ] Other (treat as last resort)
  - Details:

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

###### What are the SLIs (Service Level Indicators) an operator can use to determine the health of the service?

<!--
Pick one more of these and delete the rest.
-->

- [ ] Metrics
  - Metric name:
  - [Optional] Aggregation method:
  - Components exposing the metric:
- [ ] Other (treat as last resort)
  - Details:

###### Are there any missing metrics that would be useful to have to improve observability of this feature?

<!--
Describe the metrics themselves and the reasons why they weren't added (e.g., cost,
implementation difficulties, etc.).
-->

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

No

### Scalability

<!--
For alpha, this section is encouraged: reviewers should consider these questions
and attempt to answer them.

For beta, this section is required: reviewers must answer these questions.

For GA, this section is required: approvers should be able to confirm the
previous answers based on experience in the field.
-->

###### Will enabling / using this feature result in any new API calls?

<!--
Describe them, providing:
  - API call type (e.g. PATCH pods)
  - estimated throughput
  - originating component(s) (e.g. Kubelet, Feature-X-controller)
Focusing mostly on:
  - components listing and/or watching resources they didn't before
  - API calls that may be triggered by changes of some Kubernetes resources
    (e.g. update of object X triggers new updates of object Y)
  - periodic API calls to reconcile state (e.g. periodic fetching state,
    heartbeats, leader election, etc.)
-->

No

###### Will enabling / using this feature result in introducing new API types?

<!--
Describe them, providing:
  - API type
  - Supported number of objects per cluster
  - Supported number of objects per namespace (for namespace-scoped objects)
-->

No. This KEP proposes extensions to an existing type, but not a new type itself.

###### Will enabling / using this feature result in any new calls to the cloud provider?

<!--
Describe them, providing:
  - Which API(s):
  - Estimated increase:
-->

No.

###### Will enabling / using this feature result in increasing size or count of the existing API objects?

<!--
Describe them, providing:
  - API type(s):
  - Estimated increase in size: (e.g., new annotation of size 32B)
  - Estimated amount of new objects: (e.g., new Object X for every existing Pod)
-->

Yes. Individual `ResourceSlice` and `Pod` objects will have additional structured fields (`nodeAllocatableResources`
and `nodeAllocatableResourceClaimStatuses`). However, because these fields are populated only for specialized workloads
utilizing DRA node allocatable claims, the overall cluster-wide memory and etcd storage footprint increase is minimal.

###### Will enabling / using this feature result in increasing time taken by any operations covered by existing SLIs/SLOs?

<!--
Look at the [existing SLIs/SLOs].

Think about adding additional work or introducing new steps in between
(e.g. need to do X to start a container), etc. Please describe the details.

[existing SLIs/SLOs]: https://git.k8s.io/community/sig-scalability/slos/slos.md#kubernetes-slisslos
-->

Yes. For pods utilizing DRA node allocatable claims, scheduling latency will slightly increase. The `DynamicResources` plugin 
evaluates effective node capacity by summing standard Spec requests with DRA allocations. This increase is expected to be minimal.

###### Will enabling / using this feature result in non-negligible increase of resource usage (CPU, RAM, disk, IO, ...) in any components?

<!--
Things to keep in mind include: additional in-memory state, additional
non-trivial computations, excessive access to disks (including increased log
volume), significant amount of data sent and/or received over network, etc.
This through this both in small and large cases, again with respect to the
[supported limits].

[supported limits]: https://git.k8s.io/community//sig-scalability/configs-and-limits/thresholds.md
-->

No.

###### Can enabling / using this feature result in resource exhaustion of some node resources (PIDs, sockets, inodes, etc.)?

<!--
Focus not just on happy cases, but primarily on more pathological cases
(e.g. probes taking a minute instead of milliseconds, failed pods consuming resources, etc.).
If any of the resources can be exhausted, how this is mitigated with the existing limits
(e.g. pods per node) or new limits added by this KEP?

Are there any tests that were run/should be run to understand performance characteristics better
and validate the declared limits?
-->

No

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

###### What steps should be taken if SLOs are not being met to determine the problem?

## Implementation History

<!--
Major milestones in the lifecycle of a KEP should be tracked in this section.
Major milestones might include:
- the `Summary` and `Motivation` sections being merged, signaling SIG acceptance
- the `Proposal` section being merged, signaling agreement on a proposed design
- the date implementation started
- the first Kubernetes release where an initial version of the KEP was available
- the version of Kubernetes where the KEP graduated to general availability
- when the KEP was retired or superseded
-->

## Drawbacks

<!--
Why should this KEP _not_ be implemented?
-->

## Alternatives

### DeviceClass API Extension for NodeAllocatableResourceMappings

In this option, the primary information about how a DeviceClass relates to node allocatable resources is contained within the `DeviceClassSpec`.

```go
// In k8s.io/api/resource/v1/types.go
type DeviceClassSpec struct {
    // ... existing fields
    // NodeAllocatableResourceMappings lists the node allocatable resources that this DeviceClass can provide or depend on.
    // +optional
    // +featureGate=DRANodeAllocatableResources
    NodeAllocatableResourceMappings []NodeAllocatableResourceMapping `json:"nodeAllocatableResourceMappings,omitempty"`
}

// NodeAllocatableResourceAccountingPolicy, NodeAllocatableResourceQuantity
// are defined the same as in the main proposal.
```

**Reason for Not Choosing:**

While defining `NodeAllocatableResourceMappings` in the `DeviceClass` is simpler, it lacks the granularity needed for many real-world scenarios. The Device API Extension approach allows these mappings to be specified per-Device instance within the `ResourceSlice`. This is advantageous because:

1.  **Heterogeneous Devices:** Even within the same `DeviceClass`, individual device instances can have different node allocatable resource implications. For example, different GPU models or even the same model on different parts of the system topology might have varying CPU/memory overheads. Option 1 cannot express this.
2.  **Complex Resources:** Resources where we use Partitionable Devices to model hierarchies (e.g., sockets, NUMA nodes, caches, cores). The node allocatable resource capacity (e.g., number of CPUs) is associated with specific instances in the hierarchy changes and this is best represented in individual `Device` entries.

### Explicit AccountingPolicy in DeviceClass and PodStatus

In the initial Alpha 1 proposal (`KEP_orig.md`), future enhancements for accounting policies explored defining an explicit string enum `NodeAllocatableResourceAccountingPolicy` configured inside `DeviceClass` and tracked in `PodStatus`.

```go
// NodeAllocatableResourceAccountingPolicy defines how node allocatable resource quantities like CPU, Memory
// allocated via DRA are aggregated with standard resource requests in the PodSpec.
type NodeAllocatableResourceAccountingPolicy string

const (
  // PolicyAddPerClaim indicates that the node allocatable resource quantity in the DRA claim 
  // is treated as additional to the pod spec requests. This quantity is accounted 
  // for exactly once per claim instance, regardless of the number of containers referencing it. 
	PolicyAddPerClaim NodeAllocatableResourceAccountingPolicy = "AddPerClaim"

  // PolicyAddPerReference indicates that the node allocatable resource quantity in the DRA 
  // claim is treated as additional to the pod spec requests. This quantity is 
  // accounted for cumulatively for every reference to the claim. 
	PolicyAddPerReference NodeAllocatableResourceAccountingPolicy = "AddPerReference"

  // PolicyMax indicates that effective request is the greater value between the standard container 
  // request and the DRA claim for the same resource.
  PolicyMax NodeAllocatableResourceAccountingPolicy = "Max"

  // PolicyConsumeFrom indicates that a DRA claim is defined to represent the node 
  // resource pool capacity. All containers or pods referencing the claim are satisfied from the capacity pool defined by the DRA claim.
  PolicyConsumeFrom NodeAllocatableResourceAccountingPolicy = "ConsumeFrom"
)

// In k8s.io/api/resource/v1/types.go
type DeviceClassSpec struct {
  // ... existing fields ...
  // NodeAllocatableResourceAccountingPolicies defines how the node allocatable resource represented by the devices 
  // in this class should be accounted for and aggregated with any standard request for the same resource.
  // +optional
  // +featureGate=DRANodeAllocatableResources
  NodeAllocatableResourceAccountingPolicies map[ResourceName]NodeAllocatableResourceAccountingPolicy
}

// In k8s.io/api/core/v1/types.go
type NodeAllocatableResourceClaimStatus struct {
  // ... existing fields ...
  // AccountingPolicy tells Kubelet which policy was used by the scheduler.
  AccountingPolicy map[ResourceName]NodeAllocatableResourceAccountingPolicy
}
```

**Reason for Not Choosing:**

1.  **Granularity for Additive Policies**: While configuring an explicit `AccountingPolicy` enum in `DeviceClass` is simpler, it lacks the granularity needed for complex device configurations. Instead, Alpha 2 transitions to a structured union directly inside `Device.NodeAllocatableResourceMappings` (`Direct` vs `Overhead`). This allows drivers to declare exact capacity consumption or auxiliary per-pod/per-reference overheads natively per device instance rather than forcing a single flat policy across an entire device class.
2.  **Generic Reservation Solution for ConsumeFrom**: The `ConsumeFrom` policy (reserving a pool of resources and drawing down from it) represents a much broader concept that applies to all cluster resources, not just node allocatable resources. Attempting to solve `ConsumeFrom` exclusively within this KEP would create duplicate, domain-specific reservation mechanisms. Therefore, `ConsumeFrom` is excluded from this KEP in favor of a unified, generic reservation solution being explored in [Kubernetes Enhancement Issue #6048](https://github.com/kubernetes/enhancements/issues/6048).

### Alternative Model for pod level resources + DRA

This section explores an alternative design where the pod footprint is always
calculated as the sum of pod level resources and allocated DRA claims (additive
model: pod level resources + DRA).

If pod level resources are not specified in the PodSpec, the behavior is
identical to the current proposal (sum-of-containers plus DRA allocations).

Under this alternative model, cgroup enforcement and scheduling components would
be configured as follows:

#### 1. Kubelet Cgroup Enforcement

*   **pod level Parent Cgroup**:
    If pod level resources are specified, Kubelet sets the parent pod cgroup
    limits by adding the DRA resource allocation to the pod level limits:

    ```
    Request   = pod level requests + DRA claims
    Limit     = pod level limits + DRA claims
    ```

*   **Container-Level Fallback Capping**:
    Similar to the current proposal, container-level cgroup settings are
    configured according to container-level requests/limits if explicitly
    specified. The cgroup enforcement changes only for the fallback behavior
    when container-level limits are omitted:
    *   For containers without claims:
        `Limits = pod level limits` (caps them at the pod level resources
        baseline to prevent leaking into sibling claims).
    *   For containers with claims:
        `Limits = pod level limits + DRA claim`

#### 2. Kube-Scheduler Changes

*   **Footprint Calculation**:
    The dynamic validation check in the `DynamicResources` scheduler plugin
    (which verifies that resolved DRA claims fit within the pod level resource
    ceiling) is removed. The scheduler plugin instead calculates the effective
    pod requests as:
    ```
    Effective Pod Request = pod level requests + DRA claim requests
    ```
    This sum is checked against node allocatable capacity during the resource fit check.


#### Enforcement Use Case Walkthroughs with this model

To demonstrate how cgroup enforcement and limit configurations would work under
this alternative model, consider the following walkthroughs:

##### 1. pod level Request and Limit + DRA Claim (Single container references claim)

*   **Setup**: The pod defines explicit pod level resources. Container `c1` references the DRA claim, which resolves to a **Direct mapped** allocation of 5 CPU and 5 GiB memory. Container `c2` has no claims and specifies no container-level limits.
*   **Pod Spec**:
    ```yaml
    spec:
      resources:
        requests: { cpu: "5", memory: "5Gi" }
        limits: { cpu: "5", memory: "5Gi" }
      containers:
      - name: c1
        resources:
          claims: [{ name: "dra-claim" }]
      - name: c2
      resourceClaims:
      - name: dra-claim
        resourceClaimName: dra-claim
    ```
*   **Pod Status (Allocated)**:
    ```yaml
    status:
      nodeAllocatableResourceClaimStatuses:
      - resourceClaimName: dra-claim
        containers: ["c1"]
        mapping:
        - name: cpu
          quantity: "5"
        - name: memory
          quantity: "5Gi"
    ```
*   **Cgroup Bounds Configuration**:
    *   **pod level Cgroup**:
        *   `cpu.weight` (CPU Shares): Inflated by adding DRA requests to pod level resources requests (5 + 5): **10** (10240 shares).
        *   `cpu.max` (CPU Quota): Inflated by adding DRA limits to pod level resources limits (5 + 5): **10 CPUs**.
        *   `memory.max` (Memory Limit): Inflated by adding DRA limits to pod level resources limits (5 GiB + 5 GiB): **10 GiB**.
    *   **Container Level Cgroups**:
        *   Container `c1` (references claim):
            *   `cpu.max` (CPU Quota): Capped at `claim + pod level limit` (5 + 5): **10 CPUs**.
            *   `memory.max` (Memory Limit): Capped at `claim + pod level limit` (5 + 5): **10 GiB**.
        *   Container `c2` (does NOT reference claim):
            *   `cpu.max` (CPU Quota): Capped at pure pod level limit: **5 CPUs**.
            *   `memory.max` (Memory Limit): Capped at pure pod level limit: **5 GiB**.
*   **Outcome**: Only container `c1` (which references the claim) includes the DRA allocation in its limit. Sibling container `c2` is restricted to the baseline pod level Resources limits.
 
##### 2. pod level Request and Limit + Fungible DRA Claim (Prioritized List)

*   **Setup**: The pod defines explicit pod level Resources (5 CPU, 5 GiB memory). Container `c1` references a fungible DRA claim representing a prioritized list:
    *   **Option A (Preferred)**: GPU device mapping with flat 2 CPU and 2 GiB memory host overhead.
    *   **Option B (Fallback)**: CPU device mapping with direct allocation of 4 CPU (no memory/CPU overhead).
*   **Pod Spec**:
    ```yaml
    spec:
      resources:
        requests: { cpu: "5", memory: "5Gi" }
        limits: { cpu: "5", memory: "5Gi" }
      containers:
      - name: c1
        resources:
          claims: [{ name: "fungible-claim" }]
      - name: c2
      resourceClaims:
      - name: fungible-claim
        resourceClaimName: preferred-gpu-fallback-cpu
    ```

**Sub-Case 4a: Resolved to Option A (GPU Allocated)**

*   **Pod Status (Allocated)**:
    ```yaml
    status:
      nodeAllocatableResourceClaimStatuses:
      - resourceClaimName: preferred-gpu-fallback-cpu
        containers: ["c1"]
        overhead:
        - name: cpu
          quantity: "2"
        - name: memory
          quantity: "2Gi"
    ```
*   **Cgroup Bounds Configuration**:
    *   **pod level Cgroup**:
        *   `cpu.weight` (CPU Shares): Inflated by adding DRA requests to pod level resources requests (5 + 2): **7** (7168 shares).
        *   `cpu.max` (CPU Quota): Inflated by adding DRA limits to pod level resources limits (5 + 2): **7 CPUs**.
        *   `memory.max` (Memory Limit): Inflated by adding DRA limits to pod level resources limits (5 GiB + 2 GiB): **7 GiB**.
    *   **Container Level Cgroups**:
        *   Container `c1` (references claim):
            *   `cpu.max` (CPU Quota): Capped at `claim + pod level limit` (2 + 5): **7 CPUs**.
            *   `memory.max` (Memory Limit): Capped at `claim + pod level limit` (2 + 5): **7 GiB**.
        *   Container `c2` (does NOT reference claim):
            *   `cpu.max` (CPU Quota): Capped at pure pod level limit: **5 CPUs**.
            *   `memory.max` (Memory Limit): Capped at pure pod level limit: **5 GiB**.

**Sub-Case 4b: Resolved to Option B (Fallback CPU Allocated)**

*   **Pod Status (Allocated)**:
    ```yaml
    status:
      nodeAllocatableResourceClaimStatuses:
      - resourceClaimName: preferred-gpu-fallback-cpu
        containers: ["c1"]
        mapping:
        - name: cpu
          quantity: "4"
    ```
*   **Cgroup Bounds Configuration**:
    *   **pod level Cgroup**:
        *   `cpu.weight` (CPU Shares): Inflated by adding DRA requests to pod level resources requests (5 + 4): **9** (9216 shares).
        *   `cpu.max` (CPU Quota): Inflated by adding DRA limits to pod level resources limits (5 + 4): **9 CPUs**.
        *   `memory.max` (Memory Limit): Since no DRA memory is allocated, matches pure pod level resources Memory limit: **5 GiB**.
    *   **Container Level Cgroups**:
        *   Container `c1` (references claim):
            *   `cpu.max` (CPU Quota): Capped at `claim + pod level limit` (4 + 5): **9 CPUs**.
            *   `memory.max` (Memory Limit): Capped at pure pod level limit: **5 GiB**.
        *   Container `c2` (does NOT reference claim):
            *   `cpu.max` (CPU Quota): Capped at pure pod level limit: **5 CPUs**.
            *   `memory.max` (Memory Limit): Capped at pure pod level limit: **5 GiB**.

*   **Outcome**: Under Option A, the pod footprint automatically sets itself to
    7 CPUs and 7 GiB. Under Option B, the footprint is 9 CPUs and 5 GiB. In both
    options, the sibling container `c2` is safely restricted to the baseline pod
    level resources spec limit (5 CPUs, 5 GiB), and scheduling succeeds without
    requiring the user to over-provision the pod's limit in the Spec.

**Pros:**
1. Footprint dynamically scales based on the allocated claim, which avoids
   sizing the pod level requests to the maximum choice in prioritized lists.
2. DRA claims remain consistently additive on top of standard requests at
   both pod and container levels.

**Cons:**
1. We lose the current semantic that pod level resources serve as the absolute
   upper bound of the pod's resource footprint.
2. Existing DRA drivers face onboarding friction, requiring custom
   coordination to prevent double accounting in scheduler node capacity.
3. The current proposal of using pod level resources as an upper ceiling
   provides immediate solutions for quota enforcement, LimitRange, VPA, and
   Cluster Autoscaler that were solved for pod level resources, provided we
   have a restriction that pod level resources are specified for pods with
   node allocatable claims.

## Infrastructure Needed (Optional)

<!--
Use this section if you need things from the project/SIG. Examples include a
new subproject, repos requested, or GitHub details. Listing these here allows a
SIG to get the process for these resources started right away.
-->