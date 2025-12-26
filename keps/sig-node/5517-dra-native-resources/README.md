# KEP-NNNN: DRA: Native Resource Requests

<!-- toc -->
  - [Release Signoff Checklist](#release-signoff-checklist)
  - [Summary](#summary)
  - [Motivation](#motivation)
    - [Core Problem](#core-problem)
    - [Goals](#goals)
    - [Non-Goals](#non-goals)
  - [Proposal](#proposal)
    - [Background](#background)
        - [Standard Resource Accounting](#standard-resource-accounting)
        - [Dynamic Resource Allocation (DRA) Accounting](#dynamic-resource-allocation-dra-accounting)
    - [User Stories](#user-stories)
    - [Risks and Mitigations](#risks-and-mitigations)
  - [Design Details](#design-details)
    - [API Changes](#api-changes)
      - [DeviceClass and Device API Extensions](#deviceclass-and-device-api-extensions)
        - [Resource Representation Examples](#resource-representation-examples)
- [DeviceClass](#deviceclass)
  - [nativeResources: [&quot;cpu&quot;]](#nativeresources-cpu)
- [ResourceSlice](#resourceslice)
      - [Pod API Changes](#pod-api-changes)
      - [Kube-Scheduler Workflow](#kube-scheduler-workflow)
      - [Special Cases](#special-cases)
        - [Handling Multiple Claims for the Same Native Resource with Different Aggregation Policy](#handling-multiple-claims-for-the-same-native-resource-with-different-aggregation-policy)
        - [Multiple Containers Sharing a Claim](#multiple-containers-sharing-a-claim)
        - [Multiple Pods Sharing a Claim](#multiple-pods-sharing-a-claim)
        - [Claim Specified but Not Used in Containers](#claim-specified-but-not-used-in-containers)
    - [Use Case Walkthroughs](#use-case-walkthroughs)
      - [Use Case 1: Standard Pod (No Native Resource Claim)](#use-case-1-standard-pod-no-native-resource-claim)
      - [Use Case 2: Pod with Standard and DRA CPU and Memory Request (Override)](#use-case-2-pod-with-standard-and-dra-cpu-and-memory-request-override)
      - [Use Case 3: Pod with Fungible Resource Claim (GPU or CPU)](#use-case-3-pod-with-fungible-resource-claim-gpu-or-cpu)
      - [Use Case 4: Combined DRA CPU (Override) and GPU (Dependency)](#use-case-4-combined-dra-cpu-override-and-gpu-dependency)
    - [Test Plan](#test-plan)
        - [Prerequisite testing updates](#prerequisite-testing-updates)
        - [Unit tests](#unit-tests)
        - [Integration tests](#integration-tests)
        - [e2e tests](#e2e-tests)
    - [Graduation Criteria](#graduation-criteria)
      - [Alpha](#alpha)
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
    - [DeviceClass API Extension for NativeResourceMappings](#deviceclass-api-extension-for-nativeresourcemappings)
  - [Infrastructure Needed (Optional)](#infrastructure-needed-optional)
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
  - [ ] (R) [all GA Endpoints](https://github.com/kubernetes/community/pull/1806) must be hit by [Conformance Tests](https://github.com/kubernetes/community/blob/master/contributors/devel/sig-architecture/conformance-tests.md) within one minor version of promotion to GA
- [ ] (R) Production readiness review completed
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

This KEP proposes a solution for managing native resources like CPU, Memory and Hugepages with Dynamic Resource Allocation (DRA). Currently, when native resources are managed via DRA, there is a fundamental disconnect across the control plane and the Node. In the scheduler, having two independent accounting systems (one for standard resources, one for DRA) which are managing the same underlying resource and that leads to resource overcommitment. On the node, the kubelet is completely unaware of DRA allocations which may result in incorrect QoS class assignment which has many downstream implications. This forces users into fragile workarounds that are incompatible with all the use cases.

The proposed solution in this KEP addresses the native resource accounting in the kube-scheduler. The standard resource (`NodeResourcesFit` plugin) and DRA (`DynamicResources` plugin) will be enhanced to synchronize their accounting, creating a single, authoritative ledger. The kubelet should also be enhanced to consider the native resouce request made through both the pod spec, and the DRA `RescoudeClaim` to correctly calculate QoS, configure cgroups, and protect high-priority pods. This provides a robust, backward-compatible solution for advanced resource management in Kubernetes.

## Motivation

Dynamic Resource Allocation (DRA) provides a powerful framework for managing specialized hardware resources such as GPUs, FPGAs, and high-performance network interfaces. It also enables fine-grained management of native resources like CPU and Memory, for example, through the [dra-driver-cpu](https://github.com/kubernetes-sigs/dra-driver-cpu). However, when a native resource is managed via DRA, a fundamental disconnect emerges between the scheduler, the kubelet, and the DRA framework which breaks the resource guarantees. 

Additionally, specialized resources like accelerators have implicit dependency on native resources like CPU or Hugepages for the application to interact with it. Currently, users must manually research and declare these auxiliary native resource requirements, typically as additional requests in the PodSpec. This process is error-prone and adds complexity to workload configuration.

### Core Problem

The core problem is that the same underlying physical resource is advertised and consumed through two parallel, uncoordinated mechanisms. 

* **Dual Publication:** A node's total CPU/Memory capacity is advertised in two different places:  
  * Via the Kubelet in the `Node.Status.Allocatable` field.  
  * Via the DRA driver in `ResourceSlice` objects.

* **Dual Consumption:** Pods can consume this CPU capacity in two different ways:  
  * Via `pod.spec.resources`, which is accounted for by the `NodeResourcesFit` scheduler plugin.  
  * Via `ResourceClaim`, which is accounted for by the `DynamicResources` scheduler plugin.

**Scheduler-Level Resource Oversubscription**: The kubelet is the source of truth for a node's available resources. The scheduler continuously watches the `Node` object and uses `Node.Status.Allocatable` to maintain an internal, in-memory cache (`NodeInfo`) of each node's capacity. This cache is the baseline for all its scheduling decisions, ensuring it does not place more pods on a node than the node reports it can handle.

It is completely blind to the fact that the DRA (like CPU `ResourceClaim`) draws from the same physical resource as a standard request. This gap leads to the scheduler overcommitting a node's CPU resources by scheduling more pods than the node resource capacity.

**Kubelet-Level Guarantee Failure:** The kubelet is the component that enforces resource guarantees on the node. It determines a pod's Quality of Service (QoS) class, configures its cgroups, and makes critical life cycle decisions like eviction based *only* on the `pod.spec`. Because it is unaware of resources allocated via DRA, it will:

* **Misclassify QoS:** A pod with a guaranteed CPU `ResourceClaim` may be misclassified as `BestEffort`. This would have downstream effects like  
  * Apply Incorrect Cgroups: It will set the wrong `cpu.shares` and `cpu.quota`, potentially throttling high-performance workloads.  
  * Make Incorrect Eviction Decisions: The misclassified pod will be the first to be evicted under node pressure.  
  * Incorrect OOM Score calculation.

Current workarounds for DRA-managed native resources (like [CPU DRA driver](https://github.com/kubernetes-sigs/dra-driver-cpu)) force users to duplicate resource requests in both the `ResourceClaim` and the standard `pod.spec.containers.resources`. However, this approach is fragile, error-prone, and difficult to manage, especially for complex pods with shared resource claims. It is also incompatible with advanced DRA features like [Prioritized Lists](https://github.com/kubernetes/enhancements/blob/master/keps/sig-scheduling/4816-dra-prioritized-list/README.md)

This KEP proposes to solve this problem by creating a single, unified resource model that spans the entire control plane, from the scheduler to the kubelet. The goal is not just to fix an accounting issue in the scheduler, but to provide a complete, native way for Kubernetes to handle core resources that are backed by DRA.

### Goals

* To create a unified accounting model within the kube-scheduler that prevents overcommitment of core resources (like CPU) when they are allocated via both standard `pod.spec` requests and DRA `ResourceClaims`.  
* To ensure the solution is compatible with different ways native resources can be represented and allocated within DRA, including as individual devices, consumable capacities ([KEP-5075](https://github.com/kubernetes/enhancements/issues/5075)), and partitionable devices ([KEP-4815](https://github.com/kubernetes/enhancements/issues/4815))
* To enable specialized devices, such as accelerators, to declare any auxiliary native resource requirements (e.g., CPU, Memory) they depend on for their operation.
* To maintain backward compatibility with existing workloads and ecosystem tools that rely on `node.status.allocatable` and the scheduler's view of node resource utilization.

### Non-Goals

* To move all resource management logic into the DRA driver. The Kubelet will remain the primary agent for cgroup management and QoS enforcement, ensuring that the benefits of its existing stability and lifecycle management features are preserved.  
* To replace the standard `pod.spec.containers.resources` API for requesting shared resources. This KEP enhances the system by adding a clear path for guaranteed resources via DRA, not by deprecating the existing, well-understood API for shared resources.  
* Changes to the Kubelet for QoS classification, cgroup management, and eviction logic based on DRA native resource allocations are not in scope for the Alpha release of this KEP.

## Proposal

This KEP introduces a unified accounting model within the kube-scheduler to integrate native resources managed by Dynamic Resource Allocation (DRA) with the scheduler's standard resource tracking. By bridging the gap between `pod.spec.resources` and DRA `ResourceClaim` allocations, we can achieve consistent resource accounting and prevent node overcommitment.

### Background

To understand the proposed solution, it is essential to first understand how kube-scheduler currently manage standard resource requests and DRA ResourceClaims.

The Kubernetes scheduler is built on a plugin-based framework that executes a series of stages to place a pod. This KEP is primarily concerned with the interaction between `NodeResourcesFit` and `DynamicResource` plugins at the `PreFilter`, `Filter`, and `Bind` stages of the [scheduling framework](https://kubernetes.io/docs/concepts/scheduling-eviction/scheduling-framework/).

##### Standard Resource Accounting

The Kubelet is the source of truth for a node's available resources. It inspects the machine's total capacity, subtracts resources reserved for the operating system (`--system-reserved`) and Kubernetes system daemons (`--kube-reserved`), and reports the result in the `Node.Status.Allocatable` field. The scheduler continuously watches for updates to this field and uses it to maintain its internal, in-memory cache (`NodeInfo`) of each node's capacity. This cache is the baseline for all its scheduling decisions.

**Kube-Scheduler Resource Accounting**  

* The scheduler maintains an in-memory `NodeInfo` object for each node, which stores the `Allocatable`, which is the the capacity of the node and `Requested`, which is an aggregated sum of the resources requested by all pods assumed to be on that node (`Requested`).
* During the `Filter` stage of scheduling, the `NodeResourcesFit` plugin checks is pod's requested resources can fit on the node (`NodeInfo.Allocatable - NodeInfo.Requested >= Pod request`). 
* The `NodeInfo.Requested` value is updated by the  scheduler framework only after a pod is successfully bound to a node. This ensures that the `NodeInfo` cache remains a source of truth for all standard resource allocations.

##### Dynamic Resource Allocation (DRA) Accounting

The `DynamicResources` plugin manages resources requested via `pod.spec.resourceClaims`. Its accounting system is entirely separate from the standard resources.

* The DRA driver/s on the node reports resource availability through the `ResourceSlice` objects.  
* During the `Filter` stage, the `DynamicResources` plugin determines if the inventory in the `ResourceSlice` objects is sufficient to satisfy the pod's `ResourceClaim`, after accounting for devices already allocated to other claims.  
* When a pod is scheduled, the `DynamicResources` plugin, in its `PreBind` stage, makes an API call to update the `ResourceClaim` object's status. This update makes the allocation permanent and visible to the rest of the cluster.

These standard resources and the dynamic resources accounting systems are completely independent. The `NodeInfo` cache is not aware of allocations recorded in `ResourceClaim` objects, which is the root cause of the accounting gap for native resources when they are managed through DRA.

### User Stories

* **Story 1 (Resource Alignment):** A HPC workload needs a certain number of exclusive CPUs and memory that are aligned on the same NUMA node as a specific NIC for maximum performance. The user creates a `ResourceClaim` with co-location constraints to enforce this. The scheduler correctly accounts for the CPU and memory requests made through the claim, adding them to the node's total requested resources, so the node is not oversubscribed.

**Story 2 (Dedicated and Shared resources):** A Telco application has some high-priority application containers and some lower-priority sidecar containers. The user wants to dedicate some CPU cores exclusively to the application containers for low latency, while allowing sidecar containers to run on the node's general shared CPU pool. They use DRA to request exclusive cores and standard Pod Spec requests for the shared CPU portion. The scheduler should correctly account for both dedicated and shared requests made through these different mechanisms. 

* **Story 3 (Accelerator with Native Resource Dependency):** An AI inference job requests a GPU through a `ResourceClaim`. The specific GPU model also requires certain number of CPUs and Hugepages that are required for the application to interact with the accelerator. Instead of requiring the user to know about these auxiliary CPU and HugePages requests and add it to their PodSpec, the GPU Device can be configured to declare these dependencies. The Kubernetes scheduler accounts for both the CPU/HugePages needs for the GPU device and the standard pod spec requests, ensuring the pod lands on a node with sufficient capacity for all requirements. The user experience is simplified, as they only need to ask for the primary device they care about.

* **Story 4 (Fungibility):** An ML inference job can use either a full GPU or, if none is available, a slice of 8 exclusive CPUs. The user creates a `ResourceClaim` with a `firstAvailable` list to represent this fungible need. The scheduler evaluates both paths against a node's available resources. It finds a node with 8 available CPUs, correctly reserves them in its central `NodeInfo` cache, and schedules the pod. The user did not need to guess which resource to put in the `pod.spec`.  

### Risks and Mitigations

* Increased API and user complexity by having two ways to request native resources (PodSpec and ResourceClaim). To mitigate, the documentation would be enhanced with clear guidelines and use cases for DRA for Native Resources.
* Bugs in the kube-scheduler's new accounting logic would lead to incorrect node resource calculations and node oversubscription. Extensive unit and integration tests covering various resource claim and standard request combinations should help mitigate this. The feature will also be rolled out gradually, beginning with an alpha release gather feedback and address potential concerns.
* Until Kubelet is made DRA-aware for native resources (a non-goal for Alpha), QoS and node-level enforcement will not fully reflect DRA allocations. This is an accepted limitation for the initial Alpha scope.

## Design Details

The proposal here is to enhance the kube-scheduler to implement a **"Unified Accounting"** model for native resources requested through the standard pod Spec or through Dynamic Resource Allocation (DRA) claims. This involves modifications in `NodeResourcesFit` and `DynamicResources` plugins in how they track resource usage on the node. This also includes updates to the DRA API for drivers to declare native resource implications, and Pod Status to record DRA-based native resource allocations. This model is designed to work with the fixed, default scheduler plugin execution order (`NodeResourcesFit` runs before `DynamicResources`). 

### API Changes


#### DeviceClass and Device API Extensions

The primary mechanism for a DRA driver to inform the scheduler about native resource associations is by extending the `DeviceClass` and `Device` resources. This approach allows per-device granularity in specifying native resource mappings.

```go
// In k8s.io/api/resource/v1/types.go
type DeviceClassSpec struct {
    // ... existing fields
    // NativeResources lists the name of native resources that this DeviceClass can include.
    // +optional
    // +featureGate=DRANativeResources
    NativeResources []core.ResourceName `json:"nativeResources,omitempty"`
}
```


```go
// In k8s.io/api/resource/v1/types.go
type Device struct {
    // ... existing fields
    // NativeResourceMappings contains information about the native resources that this Device
    // is a Source of or has a Dependency on.
    // +optional
    // +featureGate=DRANativeResources
    NativeResourceMappings []NativeResourceMapping `json:"nativeResourceMappings,omitempty"`
}

// NativeResourceAggregationPolicy defines how the DRA quantity interacts with PodSpec.
type NativeResourceAggregationPolicy string

const (
  // PolicyOverride means the DRA claim value replaces the pod spec request.
  PolicyOverride NativeResourceAggregationPolicy = "Override"
  // PolicyAdd means the DRA claim value is added to the pod spec request.
  PolicyAdd      NativeResourceAggregationPolicy = "Add"
)

// NativeResourceQuantity defines the method to identify how we obtain native resource quantity from the Claim.
// Only one of PerInstanceQuantity or Capacity must be specified.
type NativeResourceQuantity struct {
    // PerInstanceQuantity: Each allocated device instance contributes this Quantity to the native resource.
    // Used when devices in the ResourceSlice represent discrete units of the native resource.
    // +optional
    PerInstanceQuantity *resource.Quantity `json:"perInstanceQuantity,omitempty"`

    // Capacity: The native resource quantity is derived from a DRA capacity
    // with the specified QualifiedName. This should match a key in Device.spec.capacity.
    // +optional
    Capacity QualifiedName `json:"capacity,omitempty"`
}

type NativeResourceMapping struct {
    // ResourceName is the name of the core v1 resource (e.g., "cpu", "memory").
    ResourceName core.ResourceName `json:"resourceName"`

    // QuantityFrom defines how the quantity of the native resource is determined.
    QuantityFrom NativeResourceQuantity `json:"quantityFrom"`

    // AggregationPolicy defines how the native resource quantity from this mapping
    // should be combined with any standard request for the same resource
    // in the pod.spec.containers[].resources.requests.
    // +optional
    AggregationPolicy core.NativeResourceAggregationPolicy `json:"aggregationPolicy,omitempty"`

    // CountPerReference determines how the native resource quantity derived
    // from this mapping is accounted when multiple containers
    // reference the ResourceClaim.
    // +optional
    CountPerReference bool `json:"countPerReference,omitempty"`
}
```

*   **`DeviceClass.spec.NativeResources`**: A list of `core.ResourceName` (e.g., `cpu`, `memory`, `hugepages-2Mi`). This field signals to the scheduler's `NodeResourcesFit` plugin which native resources is managed by devices of this class so that node fit check can be delegated to the `DynamicResources` plugin.

*   **`Device.spec.nativeResourceMappings`**: This new struct within `Device` object in a `ResourceSlice` provides the specific details of how this particular device instance relates to native resources.
    *   **ResourceName:** The core v1 resource name (e.g., `cpu`, `memory`, `hugepages-1Gi`).
    *   **QuantityFrom:** Specifies how the quantity of the native resource allocated to the `ResourceClaim` is derived. This is a struct with mutually exclusive fields:
        *   **PerInstanceQuantity:** Used when each device instance allocated contributes a fixed amount of the native resource. Suitable for models where devices are discrete units (e.g., a "l3Cache" device is always 8 CPU, a "core" device is 2 CPU).
        *   **Capacity:** Used when the native resource quantity is tied to a named capacity within the DRA device's definition in the `ResourceSlice` (e.g., drawing from a "cpu-capacity" counter within a NUMA group device).
    *   **AggregationPolicy:** Defines how the native resource quantity derived from the DRA claim interacts with any requests for the same resource made in the container's `resources.requests` in the Pod Spec.
        *   **PolicyOverride:** The quantity from the DRA claim fully replaces any request in the PodSpec for the same resource for scheduler accounting. 
        *   **PolicyAdd:** The quantity from the DRA claim is added to the quantity in the PodSpec requests.
    *   **CountPerReference:** This is relevant when the same claim is reference by multiple containers or pods. If true, the native resource cost derived from `QuantityFrom` should be accounted towards Node reservation by the scheduler every time the claim is referenced. If false, the accounting is done only once per claim and all the containers sharing the claim will consume the same resource pool. Defaults to `false`.  

##### Resource Representation Examples

The Device API Extension model is flexible enough to support various ways of representing native resources.

1.  **Native resource represented as individual devices**

    ```yaml
    # DeviceClass
    apiVersion: resource.k8s.io/v1
    kind: DeviceClass
    metadata:
      name: cpu-core
    spec:
      selectors:
      - cel: 'device.driver == "dra.native.com"'
      nativeResources: ["cpu"]
    ---
    # ResourceSlice
    apiVersion: resource.k8s.io/v1
    kind: ResourceSlice
    metadata:
      name: cpu-slice
    spec:
      driver: dra.native.com
      nodeName: my-node
      pool: { name: "node-pool", generation: 1, resourceSliceCount: 1 }
      devices:
      - name: cpu0
        attributes:
          numaNode: 0
        nativeResourceMappings:
        - resourceName: "cpu"
          aggregationPolicy: "Override"
          quantityFrom: { perInstanceQuantity: "1" }
      # ... other cpu devices
    ```
    *   Each device instance (like `cpu0`) in the `ResourceSlice` represents a single unit of CPU.
    *   The `nativeResourceMappings` in each `Device` clearly states its contribution.

2.  **Native resource represented as Consumable Pool**

   *   This example uses the `Capacity` field within `QuantityFrom` to link to `device.capacity` for the native resource represented as consumable capacity.

    ```yaml
    # DeviceClass
    apiVersion: resource.k8s.io/v1
    kind: DeviceClass
    metadata:
      name: cpu-pool
    spec:
      selectors:
      - cel: 'device.driver == "dra.native.com"'
      nativeResources: ["cpu", "memory"]
    ---
    # ResourceSlice
    apiVersion: resource.k8s.io/v1
    kind: ResourceSlice
    metadata:
      name: cpu-pool-slice
    spec:
      driver: dra.native.com
      nodeName: my-node
      pool: { name: "node-pool", generation: 1, resourceSliceCount: 1 }
      devices:
      - name: socket0
        attributes:
          "dra.native.com/type": "socket"
        allowMultipleAllocations: true
        capacity:
          "dra.native.com/cpu": "128"
          "dra.native.com/memory": "256Gi"
        nativeResourceMappings:
        - resourceName: "cpu"
          aggregationPolicy: "Add"
          quantityFrom:
            capacity: "dra.native.com/cpu"
        - resourceName: "memory"
          aggregationPolicy: "Add"
          quantityFrom:
            capacity: "dra.native.com/memory"
    ```

3.  **Partitionable Devices (KEP-4815)**

    *   In the below example CPU is represented as a partitionable device with NUMA Node and L3 cache partitions. 
    *   The `Device` objects representing the L3 cache and NUMA node partitions declare their CPU capacity.
    *   `QuantityFrom.Capacity` links the native resource accounting to this device-specific capacity.

    ```yaml
    # DeviceClass
    apiVersion: resource.k8s.io/v1
    kind: DeviceClass
    metadata:
      name: dra-l3-caches
    spec:
      selectors:
      - cel: 'device.driver == "dra.native.com"'
      nativeResources: ["cpu"]
    ---
    # ResourceSlice for L3 cache devices
    apiVersion: resource.k8s.io/v1
    kind: ResourceSlice
    # ...
    spec:
      # ...
      devices:
      - name: socket-0-l3-0
        attributes:
          dra.native.com/type: l3cache
          dra.native.com/numaID: "0"
        capacity:
          "dra.native.com/cpu": "8" # This L3 cache contains 8 CPUs
        nativeResourceMappings:
        - resourceName: "cpu"
          aggregationPolicy: "Override"
          quantityFrom:
            capacity: "dra.native.com/cpu"
      - name: socket-0-numa-0
        attributes:
          dra.native.com/type: numa
          dra.native.com/numaID: "0"
        capacity:
          "dra.native.com/cpu": "32" # This numa node contains 32 CPUs
        nativeResourceMappings:
        - resourceName: "cpu"
          aggregationPolicy: "Override"
          quantityFrom:
            capacity: "dra.native.com/cpu"
    ```

4.  **Auxiliary native resource requests for Accelerators**

  *   The accelerator device uses `NativeResourceMapping` to indicate it needs additional CPU and Memory. These amounts will be *added* to the pod's total requests.
  *   **Importantly, the native resources specified in `NativeResourceMapping` (e.g., CPU, Memory) are not necessarily managed by the DRA driver in the same way as the accelerator itself.** Instead, this mechanism primarily serves as an accounting system for the kube-scheduler to not over commit the node.

    ```yaml
    # DeviceClass
    apiVersion: resource.k8s.io/v1
    kind: DeviceClass
    metadata:
      name: ai-accelerators
    spec:
      selectors:
      - cel: 'device.driver == "xpu.example.com"'
      nativeResources: ["cpu", "memory"]
    ---
    # ResourceSlice
    apiVersion: resource.k8s.io/v1
    kind: ResourceSlice
    metadata:
      name: my-node-xpus
    spec:
      driver: xpu.example.com
      nodeName: my-node
      # ...
      devices:
      - name: xpu-model-x-001
        attributes:
          example.com/model: "model-x"
        nativeResourceMappings:
        - resourceName: "cpu"
          quantityFrom: { perInstanceQuantity: "2" }
          aggregationPolicy: "Add"
        - resourceName: "memory"
          quantityFrom: { perInstanceQuantity: "8Gi" }
          aggregationPolicy: "Add"
    ```

#### Pod API Changes

We add a new field `NativeResourceClaimStatus` to `PodStatus` as a way to pass the allocation details from `DynamicResources` plugin to the kube-scheduler accounting logic.

```go
// In k8s.io/api/core/v1/types.go

// PodStatus represents information about the status of a pod.
type PodStatus struct {
    // ... existing fields

  // NativeResourceClaimStatus contains the status of native resources (like cpu, memory)
  // that were allocated for this pod via the Dynamic Resource Allocation framework
  // It may be empty if no native resources were allocated with this claim.
  // +featureGate=DRANativeResource
  // +optional
  NativeResourceClaimStatus []PodNativeResourceClaimStatus
}

// PodNativeResourceClaimStatus describes the status of native resources allocated via DRA.
type PodNativeResourceClaimStatus struct {
  // ClaimInfo holds the reference to the ResourceClaim that provided the allocation.
  ClaimInfo ObjectReference
  // Containers lists the names of all containers in this pod that are
  // sharing the allocation from this claim.
  Containers []string
  // Resources lists the native resources and quantities allocated by this claim
  // for the containers listed in ContainerNames.
  Resources []NativeResourceAllocation
}

// NativeResourceAllocation describes the allocation of a native resource.
type NativeResourceAllocation struct {
     // ResourceName is the native resource name (e.g., "cpu", "memory").
     ResourceName ResourceName
     // Quantity is the amount of native resource allocated.
     Quantity resource.Quantity
     // AggregationPolicy specifies how this native resource from the DRA claim
     // was combined with any standard request for the same resource
     // in the pod.spec.containers[].resources.requests.
     // This is copied from NativeResourceMapping in Device API.
     AggregationPolicy NativeResourceAggregationPolicy
     // CountPerReference determines how the native resource quantity derived
     // from this mapping is accounted when multiple containers
     // reference the ResourceClaim.
     // This is copied from NativeResourceMapping in Device API
     CountPerReference bool
}

```

#### Kube-Scheduler Workflow

The scheduling process for a Pod involves several stages. The following describes how the `NodeResourcesFit` and `DynamicResources` plugins interact within the kube-scheduler framework to achieve unified accounting for native resources managed by DRA.

1.  **PreFilter Stage:** This stage is for initial checks and pre-computations to quickly filter out non-viable nodes or prepare data for the `Filter` stage, minimizing work on each node.
    *   **NodeResourcesFit Plugin**: It examines the `DeviceClass` from `pod.spec.resourceClaims` to determine which `DeviceClass`and determines if any resource claims are associated with native resources. This is cached within the plugin's context for the Filter stage.
    *   **DynamicResources Plugin**:Executes its standard PreFilter checks on all ResourceClaim objects, including validation of claim and class existence. For claims involving native resources, it additionally validates by checking if aggregation policies of different claims are compatible. Any failure here makes the pod unschedulable on any node.

2.  **Filter Stage:** This stage performs the node-level checks to determine if a pod fits on a specific node.
    *   **NodeResourcesFit Plugin:** This plugin runs first. When evaluating resource fit for a container, it checks if the resource is identified in the PreFilter stage as being managed by DRA. If so, it *skips* its standard accounting for that resource, effectively deferring the responsibility to the `DynamicResources` plugin.
    *   **DynamicResources Plugin:** Runs after `NodeResourcesFit`. This plugin handles the device allocation from `ResourceSlice` objects. It then calculates the pod's total demand for each native resource by combining any standard PodSpec requests with the native resource mappings from the allocated devices. This calculation is done based on the Device configurations like `AggregationPolicy`, `QuantityFrom`, and `CountPerReference`. Finally, it checks if the node has enough allocatable resources to meet this total effective demand, considering resources already consumed by other pods on the node. The details are stored in the scheduler's cycle state, to be used by the PreBind stage to update the PodStatus.

3.  **PreBind Stage:** This stage performs actions right before the pod is immutably bound to the node.
    *   **DynamicResources Plugin:** The plugin updates the `ResourceClaim.Status` to reflect the allocated devices. It also patches the `Pod.Status` to add the `NativeResourceClaimStatus` field. This new field contains a structured summary of the native resources being provided via DRA, including the quantities, the `AggregationPolicy` applied, and which containers reference the claim. This makes the DRA contribution to the pod's native resources explicit in the Pod's status.

4.  **Bind Stage (Framework Cache Update):** This is the final step where the scheduler officially records the pod's resource consumption on the node.
    *   The scheduler framework's core logic for updating the `NodeInfo.Requested` cache is enhanced. It now reads the `pod.Status.NativeResourceClaimStatus` to understand the native resources allocated by DRA. It aggregates these resources based on the `AggregationPolicy` and `CountPerReference` information in the status, and combines them with any non-DRA managed resources from the PodSpec. This total amount is then added to the node's `Requested` resources, ensuring the scheduler's view of node utilization is complete and accurate.

This workflow ensures that the scheduler's view of node resources is always accurate, reflecting the combined impact of standard requests and DRA-managed native resources, according to the configured policies.


#### Special Cases


##### Handling Multiple Claims for the Same Native Resource with Different Aggregation Policy

A Pod may reference multiple `ResourceClaim`s with `NativeResourceMappings` and these claims might be for different device classes. The scheduler must have a clear set of rules for how to aggregate the native resource requests from these different claims, especially when they pertain to the same `core.ResourceName` (e.g., "cpu") and have different `AggregationPolicy`.

For any given `core.ResourceName` (e.g., "cpu") within a single Pod, all `NativeResourceMapping` entries with `AggregationPolicy: PolicyOverride` across all devices allocated for the Pod must stem from the same `DeviceClass`. The `DynamicResouces` plugin will throw an error in the `PreFilter` stage if we have devices from different DeviceClasses attempting to Override the same native resource. `NativeResourceMapping` entries with `AggregationPolicy: PolicyAdd` are additive. Their quantities are summed together.

**Example:**

Valid Case:
  * PodSpec: Container "c1" requests 1 CPU (standard request).
  * Claim 1: Uses DeviceClass "A". Device mapping: {Resource: cpu, AggregationPolicy: Override, Quantity: 4 CPU}
  * Claim 2: Uses DeviceClass "A". Device mapping: {Resource: cpu, AggregationPolicy: Override, Quantity: 8 CPU}
  * Claim 3: Uses DeviceClass "B". Device mapping: {Resource: cpu, AggregationPolicy: Add, Quantity: 2 CPU}
  
  Resource Footprint:
    * Since claims 1 and 2 use `Override` policy and both have the same DeviceClass "A", their quantities are summed up to form the base 12 CPU (4+8). The PodSpec request of 1 CPU is ignored.
    * Claim 3 has `Add` which makes it 14 CPU (12 + 2)

Invalid case:
  * Claim 1: DeviceClass "A", {memory, 1 GB}
  * Claim 2: DeviceClass "B", {memory, 2 GB}

  This is invalid because different DeviceClasses are trying to override the same resource.

##### Multiple Containers Sharing a Claim

When multiple containers within the same pod reference the same `ResourceClaim`:

*   If `CountPerReference` is `false` (default): The native resource quantity from the claim's `NativeResourceMapping` is accounted for only *once* for the entire pod.
*   If `CountPerReference` is `true`: The native resource quantity is counted each time a claim is referenced by a container.

**Example:**

Pod with Container A and Container B both using `claimX`.
`claimX` maps to a device with `NativeResourceMapping`: {ResourceName: "cpu", QuantityFrom: {PerInstanceQuantity: "2"}, CountPerReference: true}

*   Effective CPU overhead from `claimX` = 2 + 2 = 4 CPU. This is added to other CPU requests.
If `CountPerReference` was `false`, the overhead would be just 2 CPU.

##### Multiple Pods Sharing a Claim

To account for multiple pods sharing the claim, `NodeInfo` would be enhanced to track not just the total resources but also which pods are linked to each claim. A new variable `DRAClaimConsumers` would be added to `NodeInfo` to map claim to the pods consuming it. 
* For each DRA claim the pod uses, the pod's UID is added to the `DRAClaimConsumers` set for that claim UID. If this is the first pod to consume this claim on this node (i.e., the consumer set was empty), the native resources from `DRAClaimResources` for this claim are added to the node's total `Requested` amount.
* When a pod is removed, its reference is removed from `DRAClaimConsumers`. If the consumer set becomes empty, the native resources for this claim are subtracted from the node's total `Requested` amount.

##### Claim Specified but Not Used in Containers

If a `ResourceClaim` is listed in `pod.Spec.ResourceClaims` but not referenced by any container in `pod.Spec.Containers[*].Resources.Claims`. The resources associated with this claim ARE still accounted for against the node's capacity. The DRA allocator reserves the devices for the pod, making them unavailable to others. The `PodNativeResourceClaimStatus` entry for this claim will have an empty `Containers` list.

### Use Case Walkthroughs

#### Use Case 1: Standard Pod (No Native Resource Claim)

- **Pod:**

```yaml
apiVersion: v1
kind: Pod
metadata:
  name: standard-pod
spec:
  containers:
  - name: my-app
    image: my-image
    resources:
      requests:
        cpu: "1"
        memory: "1Gi"
```

**Expected behavior:** 

* Since there is no resource claim for native resources, the `NodeResourcesFit` scheduler plugin should continue to check the resource fit for CPU and Memory.    

#### Use Case 2: Pod with Standard and DRA CPU and Memory Request (Override)

```yaml
# ResourceSlice with Consumable Capacity - Override Policy
apiVersion: resource.k8s.io/v1
kind: ResourceSlice
metadata:
  name: node1-socket0-slice-override
spec:
  driver: dra.native.com
  nodeName: node1
  pool: {name: node1-pool, generation: 1, resourceSliceCount: 1}
  devices:
  - name: socket0
    attributes: {"dra.native.com/type": "socket"}
    allowMultipleAllocations: true
    capacity:
      "dra.native.com/cpu": "128"
      "dra.native.com/memory": "256Gi"
    nativeResourceMappings:
    - resourceName: "cpu"
      aggregationPolicy: Override
      quantityFrom:
        capacity: "dra.native.com/cpu"
    - resourceName: "memory"
      aggregationPolicy: Override
      quantityFrom:
        capacity: "dra.native.com/memory"
---
# ResourceClaim for Override
apiVersion: resource.k8s.io/v1
kind: ResourceClaim
metadata:
  name: cpu-mem-override-claim
spec:
  deviceClassName: cpu-mem-socket
  devices:
    requests:
    - name: cpu-mem-req
      capacity:
        requests:
          "dra.native.com/cpu": "4"
          "dra.native.com/memory": "8Gi"
```

**Pod:**

```yaml
# Pod
apiVersion: v1
kind: Pod
metadata:
  name: dra-pod-override
spec:
  containers:
  - name: my-app
    image: my-image
    resources:
      requests:
        cpu: "1" # This will be IGNORED for accounting
        memory: "1Gi" # This will be IGNORED for accounting
      claims:
       - name: "my-cpu-mem-claim"
  resourceClaims:
  - name: "my-cpu-mem-claim"
    resourceClaimName: cpu-mem-override-claim
```

**Expected behavior:**

*   `NodeResourcesFit`: Skips CPU and Memory checks.
*   `DynamicResources`: Allocates from the `socket0` device in `node1-socket0-slice-override`.
    *   Effective CPU: 4 (from claim, Override policy).
    *   Effective Memory: 8Gi (from claim, Override policy).
*   Scheduler Cache Update: Node's requested CPU increases by 4, Memory by 8Gi.

#### Use Case 3: Pod with Fungible Resource Claim (GPU or CPU)

```yaml
# ResourceSlice for CPU (Override Policy)
apiVersion: resource.k8s.io/v1
kind: ResourceSlice
metadata:
  name: node1-socket0-slice-override
spec:
  driver: dra.native.com
  nodeName: node1
  pool: {name: node1-pool, generation: 1, resourceSliceCount: 1}
  devices:
  - name: socket0
    attributes: {"dra.native.com/type": "socket"}
    allowMultipleAllocations: true
    capacity:
      "dra.native.com/cpu": "128"
    nativeResourceMappings:
    - resourceName: "cpu"
      aggregationPolicy: Override
      quantityFrom:
        capacity: "dra.native.com/cpu"
---
# ResourceSlice for GPUs
apiVersion: resource.k8s.io/v1
kind: ResourceSlice
metadata:
  name: node1-gpus
spec:
  driver: gpu.example.com
  nodeName: node1
  pool: {name: node1-pool, generation: 1, resourceSliceCount: 1}
  devices:
  - name: gpu0
---
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
          deviceClassName: gpu-vendor-a
          count: 1
        - name: cpu
          deviceClassName: cpu-mem-socket
          capacity:
            requests:
              "dra.native.com/cpu": "8"
```

**Pod:**

```yaml
# Pod
apiVersion: v1
kind: Pod
metadata:
  name: fungible-pod
spec:
  containers:
  - name: my-app
    image: my-image
    resources:
      requests:
        cpu: "1"
        memory: "1Gi"
      claims:
      - name: "gpu-or-cpu"
  resourceClaims:
  - name: "gpu-or-cpu"
    resourceClaimTemplateName: gpu-or-cpu-template
```

**Expected behavior:**

*   `NodeResourcesFit`: Skips CPU check.
*   `DynamicResources`:
    *   Selects GPU: Effective CPU 1 (PodSpec), 1Gi Mem.
    *   Selects CPU: Effective CPU 8 (DRA Override), 1Gi Mem (PodSpec).
*   Scheduler Cache Update: Node's requested Memory increases by 8Gi, and CPU increases by 1 or 8 (based on device allocated to claim).

#### Use Case 4: Combined DRA CPU (Override) and GPU (Dependency)

```yaml
# ResourceSlice for CPU (Override Policy)
apiVersion: resource.k8s.io/v1
kind: ResourceSlice
metadata:
  name: node1-socket0-slice-override
spec:
  driver: dra.native.com
  nodeName: node1
  pool: {name: node1-pool, generation: 1, resourceSliceCount: 1}
  devices:
  - name: socket0
    attributes: {"dra.native.com/type": "socket"}
    allowMultipleAllocations: true
    capacity:
      "dra.native.com/cpu": "128"
    nativeResourceMappings:
    - resourceName: "cpu"
      aggregationPolicy: Override
      quantityFrom:
        capacity: "dra.native.com/cpu"
---
# ResourceSlice for GPU
apiVersion: resource.k8s.io/v1
kind: ResourceSlice
metadata:
  name: node1-accelerator
spec:
  driver: acc.example.com
  nodeName: node1
  pool: {name: node1-pool, generation: 1, resourceSliceCount: 1}
  devices:
  - name: accel0
    nativeResourceMappings:
    - resourceName: "cpu"
      quantityFrom: { perInstanceQuantity: "2" }
      countPerReference: true
    - resourceName: "memory"
      quantityFrom: { perInstanceQuantity: "4Gi" }
      countPerReference: true
---
# ResourceClaim for CPU Override
apiVersion: resource.k8s.io/v1
kind: ResourceClaim
metadata:
  name: cpu-override-claim-2cpu
spec:
  deviceClassName: cpu-mem-socket
  devices:
    requests:
    - name: cpu-req
      capacity:
        requests:
          "dra.native.com/cpu": "2"
---
# ResourceClaim for GPU
apiVersion: resource.k8s.io/v1
kind: ResourceClaim
metadata:
  name: some-device-claim
spec:
  devices:
    requests:
    - name: accelerator-request
      deviceClassName: accelerator-resources
      count: 1
---
# Pod
apiVersion: v1
kind: Pod
metadata:
  name: combined-dra-pod
spec:
  containers:
  - name: my-app
    image: my-image
    resources:
      requests:
        cpu: "500m" # Will be IGNORED for accounting due to CPU claim
        memory: "1Gi" # Will be ADDED to GPU claim's memory dependency
      claims:
       - name: "cpu-claim"
       - name: "accel-claim"
  resourceClaims:
  - name: "cpu-claim"
    resourceClaimName: cpu-override-claim-2cpu
  - name: "accel-claim"
    resourceClaimName: my-accel-claim
```   - name: "my-claim"
```

**Expected Behavior:**

*   `NodeResourcesFit`: Skips CPU checks.
*   `DynamicResources`:
    *   Allocates 2 CPU from CPU device from claim (Override).
    *   Allocates 1 `accelerator-x` device.
    *   Effective CPU: 2 (CPU claim Override) + 2 (GPU Dependency) = 4 CPU. Request in PodSpec ignored.
    *   Effective Memory: 1Gi (PodSpec) + 4Gi (GPU Dependency) = 5Gi.
*   Scheduler Cache Update: Node's requested CPU increases by 4, Memory by 5Gi.

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

Unit tests will be added for all new and modified logic within the `kube-scheduler` components.

-   `pkg/scheduler/framework/plugins/noderesources/fit.go`: Tests to verify that native resource checks are correctly deferred to the `DynamicResources` plugin based on `DeviceClass`.
-   `pkg/scheduler/framework/plugins/dynamicresources/dynamicresources.go`: Tests for PreFilter, Filter, Reserve, and PreBind stages to ensure correct handling of native resource claims, including different `CombinationPolicy` scenarios, and proper calculation of resource demands.
-   `pkg/scheduler/framework/types.go`: Tests for any modifications to `NodeInfo` to support DRA native resource accounting, including the new map fields and update logic.
-   API validation tests for `DeviceClass`, `Device` and `PodStatus` to ensure the new fields are validated correctly.

##### Integration tests

Integration tests will be added in `test/integration/dynamicresource` to cover the end-to-end scheduling flow:

-   Test cases for each usecases outlined in the Use Case Walkthroughs section (Override, Add, Dependency, Fungibility, Multiple Claims, Shared Claims).
-   Tests to ensure correct interaction between `NodeResourcesFit` and `DynamicResources` plugins.
-   Test the interaction between standard `pod.spec.resources` and DRA native resource requests.
-   Tests to validate the `Pod.Status.NativeResourceClaimStatus` is populated correctly.
-   Tests to confirm that node resource accounting in the scheduler's cache is accurate.

##### e2e tests

E2E tests will be added to `test/e2e/dra`:

-   Verify that standard pods requesting CPU/Memory without DRA are not affected.
-   Tests deploying pods with various native resource claim configurations (Override, Add, Dependency) and verifying they are scheduled correctly.
-   Verify Pods are scheduled on nodes with adequate resources, considering both PodSpec and DRA requests.
-   Test scenarios with multiple containers and pods sharing claims.
-   Test scecarios with multiple claims per pod with different `AggregationPolicy` and `CountPerReference` settings.

### Graduation Criteria

#### Alpha

-   Feature implemented behind the `DRANativeResources` feature gate and disabled by default.
-   Core API changes for `DeviceClass`, `Device`, and `PodStatus` introduced.
-   Kube-scheduler changes in `DynamicResources` and`NodeResourcesFit` plugins added and the accounting logic updated.
-   All unit and integration tests outlined in the Test Plan are implemented and verified.

#### Beta

-   Gather feedback from alpha.
-   Add E2E tests for kube-scheduler changes.
-   Kubelet Integration 
    *  Implement Kubelet changes to consume `Pod.Status.NativeResourceClaimStatus`.
    *  Update Kubelet's QoS class calculation to correctly account for native resources from DRA claims.
    *  Ensure Kubelet's cgroup management uses the native resources from DRA claims.

### Upgrade / Downgrade Strategy

-   **Upgrade:** Enabling the feature gate on an existing cluster is safe. The new accounting logic will apply to any newly scheduled pods or pods that are re-scheduled. Existing pods that are already running on nodes will not have their resource accounting in the scheduler's `NodeInfo` cache immediately updated to reflect DRA native resources. Their DRA-based resources will only be correctly accounted for by the scheduler if they are evicted and rescheduled.

-   **Downgrade:** The scheduler will stop processing the native resource fields in DRA Device and DeviceClass objects. The scheduler's view of their resource usage might be incomplete, potentially leading to oversubscription if not handled carefully. Pods already scheduled with DRA native resources will continue to run. Re-enabling the gate will enable correct accounting for new pods, however, `NodeInfo` cache might still be incorrect as pods that were scheduled while the gate was off will not have their DRA native resources reflected in the cache. 

If a pod that was scheduled with the feature enabled and is deleted after the feature gate is disabled, the resources added to the NodeInfo cache from `Pod.Status.NativeResourceClaimStatus` would not be subtracted when the pod is removed. This would result in the scheduler cache overestimating resource usage on the node, potentially preventing new pods from scheduling. This inconsistent state would persist until the node is drained.

### Version Skew Strategy

An older scheduler will not understand the new API fields or perform unified accounting. If `DeviceClass` or `ResourceSlice` objects contain the new fields, they will be ignored.

## Production Readiness Review Questionnaire

### Feature Enablement and Rollback

###### How can this feature be enabled / disabled in a live cluster?

- [x] Feature gate (also fill in values in `kep.yaml`)
  - Feature gate name: `DRANativeResources`
  - Components depending on the feature gate: `kube-scheduler`, `kube-apiserver` (for API validation)

###### Does enabling the feature change any default behavior?

No. This feature only takes effect if users create Pods that request native resources via `pod.spec.resourceClaims` and DRA drivers are installed and configured to expose native resources via `nativeResourceMappings` in `ResourceSlice` objects. Existing pods are unaffected.

###### Can the feature be disabled once it has been enabled (i.e. can we roll back the enablement)?

Yes. Disabling the feature gate `DRANativeResources` will prevent new API objects from using the new fields and prevent the scheduler from performing the unified accounting. Pods already scheduled using DRA native resource accounting will continue to run. However, when *new* pods are scheduled while the gate is disabled, any native resources specified in their DRA claims will *not* be considered by the scheduler. This can lead to node oversubscription, as the scheduler's view of available resources on the node will be incomplete.  

###### What happens if we reenable the feature if it was previously rolled back?

The scheduler will resume its unified accounting logic for pods with DRA native resource claims. API validation for the new fields will be re-enabled. The `NodeInfo` cache maybe be incorrect as its not automatically updated to consider native resource claims for pods that were scheduled when the gate was disable.  This inconsistent state would persist until the node is drained.

###### Are there any tests for feature enablement/disablement?

Unit tests in `kube-scheduler` and `kube-apiserver` will verify the behavior of the scheduler plugins (`NodeResourcesFit`, `DynamicResources`) and API validation with the feature gate enabled and disabled.

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

<!--
Ideally, this should be a metric. Operations against the Kubernetes API (e.g.,
checking if there are objects with field X set) may be a last resort. Avoid
logs or events for this purpose.
-->

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
- [ ] API .status
  - Condition name: 
  - Other field: 
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

###### Will enabling / using this feature result in introducing new API types?

<!--
Describe them, providing:
  - API type
  - Supported number of objects per cluster
  - Supported number of objects per namespace (for namespace-scoped objects)
-->

###### Will enabling / using this feature result in any new calls to the cloud provider?

<!--
Describe them, providing:
  - Which API(s):
  - Estimated increase:
-->

###### Will enabling / using this feature result in increasing size or count of the existing API objects?

<!--
Describe them, providing:
  - API type(s):
  - Estimated increase in size: (e.g., new annotation of size 32B)
  - Estimated amount of new objects: (e.g., new Object X for every existing Pod)
-->

###### Will enabling / using this feature result in increasing time taken by any operations covered by existing SLIs/SLOs?

<!--
Look at the [existing SLIs/SLOs].

Think about adding additional work or introducing new steps in between
(e.g. need to do X to start a container), etc. Please describe the details.

[existing SLIs/SLOs]: https://git.k8s.io/community/sig-scalability/slos/slos.md#kubernetes-slisslos
-->

###### Will enabling / using this feature result in non-negligible increase of resource usage (CPU, RAM, disk, IO, ...) in any components?

<!--
Things to keep in mind include: additional in-memory state, additional
non-trivial computations, excessive access to disks (including increased log
volume), significant amount of data sent and/or received over network, etc.
This through this both in small and large cases, again with respect to the
[supported limits].

[supported limits]: https://git.k8s.io/community//sig-scalability/configs-and-limits/thresholds.md
-->

###### Can enabling / using this feature result in resource exhaustion of some node resources (PIDs, sockets, inodes, etc.)?

<!--
Focus not just on happy cases, but primarily on more pathological cases
(e.g. probes taking a minute instead of milliseconds, failed pods consuming resources, etc.).
If any of the resources can be exhausted, how this is mitigated with the existing limits
(e.g. pods per node) or new limits added by this KEP?

Are there any tests that were run/should be run to understand performance characteristics better
and validate the declared limits?
-->

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

### DeviceClass API Extension for NativeResourceMappings

In this option, the primary information about how a DeviceClass relates to native resources is contained within the `DeviceClassSpec`.

```go
// In k8s.io/api/resource/v1/types.go
type DeviceClassSpec struct {
    // ... existing fields
    // NativeResourceMappings lists the native resources that this DeviceClass can provide or depend on.
    // +optional
    // +featureGate=DRANativeResources
    NativeResourceMappings []NativeResourceMapping `json:"nativeResourceMappings,omitempty"`
}

// NativeResourceMapping, NativeResourceMappingRole, NativeResourceAggregationPolicy, NativeResourceQuantity
// are defined the same as in the main proposal.
```

**Reason for Not Choosing:**

While defining `NativeResourceMappings` in the `DeviceClass` is simpler, it lacks the granularity needed for many real-world scenarios. The Device API Extension approach allows these mappings to be specified per-Device instance within the `ResourceSlice`. This is advantageous because:

1.  **Heterogeneous Devices:** Even within the same `DeviceClass`, individual device instances can have different native resource implications. For example, different GPU models or even the same model on different parts of the system topology might have varying CPU/memory overheads. Option 1 cannot express this.
2.  **Complex Resources:** Resources where we use Partitionable Devices to model hierarchies (e.g., sockets, NUMA nodes, caches, cores). The native resource capacity (e.g., number of CPUs) is associated with specific instances in the hierarchy changes and this is best represented in individual `Device` entries.


## Infrastructure Needed (Optional)

<!--
Use this section if you need things from the project/SIG. Examples include a
new subproject, repos requested, or GitHub details. Listing these here allows a
SIG to get the process for these resources started right away.
-->