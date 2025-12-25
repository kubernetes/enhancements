# KEP-5517: DRA: Native Resource Requests

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
    - [DeviceClass API Extensions](#deviceclass-api-extensions)
    - [Device API Extensions](#device-api-extensions)
    - [Accounting Policy Precedence](#accounting-policy-precedence)
      - [Resource Representation Examples](#resource-representation-examples)
    - [Pod API Changes](#pod-api-changes)
    - [Kube-Scheduler Workflow](#kube-scheduler-workflow)
    - [Kubelet Admission Control](#kubelet-admission-control)
    - [Multiple Claims per Container](#multiple-claims-per-container)
    - [Shared Claims](#shared-claims)
      - [Multiple Containers Sharing a Claim](#multiple-containers-sharing-a-claim)
      - [Multiple Pods Sharing a Claim](#multiple-pods-sharing-a-claim)
    - [Unreferenced Claims](#unreferenced-claims)
    - [Handling Pod Overheads](#handling-pod-overheads)
  - [Node Resource Enforcement and Isolation](#node-resource-enforcement-and-isolation)
  - [Use Case Walkthroughs](#use-case-walkthroughs)
    - [Use Case 1: Pod with Standard and DRA CPU and Memory Request](#use-case-1-pod-with-standard-and-dra-cpu-and-memory-request)
    - [Use Case 2: Pod with Fungible Resource Claim (GPU or CPU)](#use-case-2-pod-with-fungible-resource-claim-gpu-or-cpu)
    - [Use Case 3: Combined Native (DRA CPU) and Auxiliary Request (GPU)](#use-case-3-combined-native-dra-cpu-and-auxiliary-request-gpu)
  - [Future Enhancements](#future-enhancements)
    - [Kubelet QoS and Cgroup Management](#kubelet-qos-and-cgroup-management)
    - [Integration with InPlacePodResizing](#integration-with-inplacepodresizing)
    - [Integration with Pod Level Resources](#integration-with-pod-level-resources)
    - [Additional Accounting Policies](#additional-accounting-policies)
      - [Resource Representation](#resource-representation)
      - [Accounting Policy Compatibility and Validation](#accounting-policy-compatibility-and-validation)
      - [Use Case: Pod Consuming from a Shared CPU Pool](#use-case-pod-consuming-from-a-shared-cpu-pool)
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

The proposed solution in this KEP addresses the native resource accounting in the kube-scheduler. The standard resource (`NodeResourcesFit` plugin) and DRA (`DynamicResources` plugin) will be enhanced to synchronize their accounting, creating a single, authoritative ledger. The kubelet will also be enhanced to consider the native resource request made through both the pod spec, and the DRA `ResourceClaim` to correctly calculate QoS, configure cgroups, and protect high-priority pods. This provides a robust, backward-compatible solution for advanced resource management in Kubernetes.

## Motivation

Dynamic Resource Allocation (DRA) provides a powerful framework for managing specialized hardware resources such as GPUs, FPGAs, and high-performance network interfaces. It also enables fine-grained management of native resources like CPU and Memory, for example, through the [dra-driver-cpu](https://github.com/kubernetes-sigs/dra-driver-cpu). However, when a native resource is managed via DRA, while it provides added advantages of being able to specify more detailed requirements, a fundamental disconnect emerges between the scheduler, the kubelet, and the DRA framework, which breaks the resource guarantees. 

Additionally, specialized resources like accelerators have implicit dependency on native resources like CPU or Hugepages for the application to interact with it. Currently, users must manually research and declare these auxiliary native resource requirements, typically as additional requests in the PodSpec. This process is error-prone and adds complexity to workload configuration. Furthermore, there is no existing mechanism to express critical co-location requirements. For example, there is no way to ensure an accelerator allocated via DRA is NUMA-aligned with the specific hugepages or CPUs it needs, as the standard and DRA resource models are entirely independent.

### Core Problem

The core problem is that the same underlying physical resource is advertised and consumed through two parallel, uncoordinated mechanisms. 

* **Dual Publication:** A node's total CPU/Memory capacity is advertised in two different places:  
  * Via the Kubelet in the `Node.Status.Allocatable` field.  
  * Via the DRA driver in `ResourceSlice` objects.

* **Dual Consumption:** Pods can consume this CPU capacity in two different ways:  
  * Via pod spec requests (`pod.spec.containers[].resources.requests`, `pod.spec.initcontainers[].resources.requests`), which is considered in the `NodeResourcesFit` scheduler plugin to find a Node that fits.  
  * Via `ResourceClaim`, which is considered in the `DynamicResources` scheduler plugin to allocate devices.

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
* To replace the standard `pod.spec.containers.resources` API for requesting native resources. This KEP aims to enhance the system by adding a clear path for native resource requests via DRA while ensuring it works coherently with the
  existing PodSpec-based requests.
* Changes to the Kubelet for QoS classification, cgroup management, and eviction logic based on DRA native resource allocations are not in scope for the Alpha release of this KEP.
* Interaction with In-Place Pod Resizing and Pod Level Resources will be a non goal for alpha. More details in [Future Enhancements](#future-enhancements) section.

## Proposal

This KEP introduces a unified accounting model within the kube-scheduler to integrate native resources managed by Dynamic Resource Allocation (DRA) with the scheduler's standard resource tracking. By bridging the gap between `pod.spec.resources` and DRA `ResourceClaim` allocations, we can achieve consistent resource accounting and prevent node overcommitment.

### Background

To understand the proposed solution, it is essential to first understand how kube-scheduler currently manage standard resource requests and DRA ResourceClaims.

The Kubernetes scheduler is built on a plugin-based framework that executes a series of stages to place a pod. This KEP is primarily concerned with the interaction between `NodeResourcesFit` and `DynamicResource` plugins at the `PreFilter`, `Filter`, and `Bind` stages of the [scheduling framework](https://kubernetes.io/docs/concepts/scheduling-eviction/scheduling-framework/).

##### Standard Resource Accounting

The Kubelet is the source of truth for a node's available resources. It inspects the machine's total capacity, subtracts resources reserved for the operating system (`--system-reserved`) and Kubernetes system daemons (`--kube-reserved`), and reports the result in the `Node.Status.Allocatable` field. The scheduler continuously watches for updates to this field and uses it to maintain its internal, in-memory cache (`NodeInfo`) of each node's capacity. This cache is the baseline for all its scheduling decisions.

**Kube-Scheduler Resource Accounting**  

* The scheduler maintains an in-memory `NodeInfo` object for each node, which stores the `Allocatable`, which is the capacity of the node and `Requested`, which is an aggregated sum of the resources requested by all pods assumed to be on that node (`Requested`).
* During the `Filter` stage of scheduling, the `NodeResourcesFit` plugin checks if a pod's requested resources can fit on the node (`NodeInfo.Allocatable - NodeInfo.Requested >= Pod request`). 
* The `NodeInfo.Requested` value is updated by the  scheduler framework only after a pod is successfully bound to a node. This ensures that the `NodeInfo` cache remains a source of truth for all standard resource allocations.

##### Dynamic Resource Allocation (DRA) Accounting

The `DynamicResources` plugin manages resources requested via `pod.spec.resourceClaims`. Its accounting system is entirely separate from the standard resources.

* The DRA driver/s on the node reports resource availability through the `ResourceSlice` objects.  
* During the `Filter` stage, the `DynamicResources` plugin determines if the inventory in the `ResourceSlice` objects is sufficient to satisfy the pod's `ResourceClaim`, after accounting for devices already allocated to other claims.  
* When a pod is scheduled, the `DynamicResources` plugin, in its `PreBind` stage, makes an API call to update the `ResourceClaim` object's status. This update makes the allocation permanent and visible to the rest of the cluster.

These standard resources and the dynamic resources accounting systems are completely independent. The `NodeInfo` cache is not aware of allocations recorded in `ResourceClaim` objects, which is the root cause of the accounting gap for native resources when they are managed through DRA.

### User Stories

**Story 1 (Resource Alignment):** A HPC workload needs a certain number of exclusive CPUs and memory that are aligned on the same NUMA node as a specific NIC for maximum performance. The user creates a `ResourceClaim` with co-location constraints to enforce this. The scheduler correctly accounts for the CPU and memory requests made through the claim, adding them to the node's total requested resources, so the node is not oversubscribed.

**Story 2 (Dedicated and Shared resources):** A Telco application has some high-priority application containers and some lower-priority sidecar containers. The user wants to dedicate some CPU cores exclusively to the application containers for low latency, while allowing sidecar containers to run on the node's general shared CPU pool. They use DRA to request exclusive cores and standard Pod Spec requests for the shared CPU portion. The scheduler should correctly account for both dedicated and shared requests made through these different mechanisms. 

* **Story 3 (Accelerator with Native Resource Dependency):** An AI inference job requests a GPU through a `ResourceClaim`. The specific GPU model also requires certain number of CPUs and Hugepages that are required for the application to interact with the accelerator. Instead of requiring the user to know about these auxiliary CPU and HugePages requests and add it to their PodSpec, the GPU Device can be configured to declare these dependencies. The Kubernetes scheduler accounts for both the CPU/HugePages needs for the GPU device and the standard pod spec requests, ensuring the pod lands on a node with sufficient capacity for all requirements. The user experience is simplified, as they only need to ask for the primary device they care about.

* **Story 4 (Fungibility):** An ML inference job can use either a full GPU or, if none is available, a slice of 8 exclusive CPUs. The user creates a `ResourceClaim` with a `firstAvailable` list to represent this fungible need. The scheduler evaluates both paths against a node's available resources. It finds a node with 8 available CPUs, correctly reserves them in its central `NodeInfo` cache, and schedules the pod. The user did not need to guess which resource to put in the `pod.spec`.  

* **Story 5 (Shared Resource Pool):** We want to reserve a pool of 100 CPUs for a set of pods. We define a `ResourceClaim` for this pool. Individual pods reference this claim and specify their CPU requirements via standard `pod.spec.containers[].resources.requests`. The scheduler ensures that the sum of requests from pods consuming from this pool does not exceed the pool's 100 CPU capacity, and these 100 CPUs are marked as used on the node.

### Risks and Mitigations

* Increased API and user complexity by having two ways to request native resources (PodSpec and ResourceClaim). To mitigate, the documentation would be enhanced with clear guidelines and use cases for DRA for Native Resources.
* Bugs in the kube-scheduler's new accounting logic would lead to incorrect node resource calculations and node oversubscription. Extensive unit and integration tests covering various resource claim and standard request combinations should help mitigate this. The feature will also be rolled out gradually, beginning with an alpha release gather feedback and address potential concerns.
* Until Kubelet is made DRA-aware for native resources (a non-goal for Alpha), QoS and node-level enforcement will not fully reflect DRA allocations. This is an accepted limitation for the initial Alpha scope.

## Design Details

The proposal here is to enhance the kube-scheduler to implement a **"Unified Accounting"** model for native resources requested through the standard pod Spec or through Dynamic Resource Allocation (DRA) claims. This involves modifications in `NodeResourcesFit` and `DynamicResources` plugins in how they track resource usage on the node. This also includes updates to the DRA API for drivers to declare native resource implications, and Pod Status to record DRA-based native resource allocations. The core principle is that, when a Pod has native resource requested through a DRA claim, the responsibility for checking the node resource fit is delegated to `DynamicResources` plugin, and standard checks in `NodeResourcesFit` are bypassed. The delegation should ensure correct resource accounting irrespective of the execution order of these plugins.

### API Changes

To support unified accounting for native resources, this KEP proposes API extensions to `DeviceClass` and `Device`. This model allows defining the `AccountingPolicy` in either the `DeviceClass` or on the individual `Device` object, providing flexibility for different use cases.

#### DeviceClass API Extensions

A new field `NativeResourcePolicies` is added to `DeviceClassSpec`.

```go
// In k8s.io/api/resource/v1/types.go
type DeviceClassSpec struct {
  // ManagesNativeResources indicates if devices of this class manages native resources like cpu, memory and/or hugepages.
  // +optional
  // +featureGate=DRANativeResources
  ManagesNativeResources bool `json:"managesNativeResources,omitempty"`
  // NativeResourcePolicies defines the accounting policies for native resources
  // (like cpu, memory) to be applied to devices selected by this class.
  // The key is the native resource name (e.g., "cpu", "memory").
  // The value is the AccountingPolicy to apply.
  // +optional
  // +featureGate=DRANativeResources
  NativeResourcePolicies map[v1.ResourceName]v1.NativeResourceAccountingPolicy `json:"nativeResourcePolicies,omitempty"`
}

// NativeResourceAccountingPolicy defines how the DRA quantity interacts with PodSpec.
type NativeResourceAccountingPolicy string

const (
  // PolicyAddPerClaim means the DRA claim is added to the standard request. If multiple containers reference the claim, the claim request is added only once.
  PolicyAddPerClaim      NativeResourceAccountingPolicy = "AddPerClaim"
  // PolicyAddPerReference means the DRA claim is added to the standard request.  If multiple containers reference the claim, the claim request is added once per each reference.
  PolicyAddPerReference NativeResourceAccountingPolicy = "AddPerReference"
)
```
*   `DeviceClassSpec.ManagesNativeResources`: If true, it signals to the scheduler that devices of this class affect native resources.
*   `DeviceClassSpec.NativeResourcePolicies`: This field allows administrators to specify default accounting policies for native resources for devices matching this class. This can be left empty if the `AccountingPolicy` is provided in the `Device` objects. The key of this map is the native resource name (e.g., `cpu`, `memory`, `hugepages-1Gi`)
    *   **AccountingPolicy:** Defines how the native resource quantity from the DRA claim is accounted for along with any requests for the same resource made in the container's `resources.requests` in the Pod Spec.
        1.   **AddPerClaim:** The quantity from the DRA claim is added to the standard request for the resource in the pod spec. If the claim is shared by multiple containers in the same pod, the request in the claim is added once to the Pod's total requests for this resource.
        2.   **AddPerReference:** The quantity from the DRA claim is added to the standard request for the resource in the pod spec. If the claim is shared by multiple containers in the same pod, the request in the claim is added once every time the claim is referenced.

#### Device API Extensions

The new field `NativeResourceMappings` within the `ResourceSlice.Device` spec is used to define the native resource quantities and any device-specific policy overrides.

```go
// In k8s.io/api/resource/v1/types.go
type Device struct {
    // ... existing fields
    // NativeResourceMappings contains information about the native resources that this Device
    // is a Source of or has a Dependency on.
    // +optional
    // +featureGate=DRANativeResources
    NativeResourceMappings map[v1.ResourceName]NativeResourceMapping `json:"nativeResourceMappings,omitempty"`
}

type NativeResourceMapping struct {
    // AccountingPolicy defines how the native resource quantity from this mapping
    // should be accounted for and aggregated with any standard request for the same resource
    // in the pod.spec.containers[].resources.requests.
    // +optional
    AccountingPolicy NativeResourceAccountingPolicy `json:"accountingPolicy,omitempty"`
    // QuantityFrom defines how the quantity of the native resource is determined.
    QuantityFrom NativeResourceQuantity `json:"quantityFrom"`
}

// NativeResourceQuantity defines the method to identify how we obtain native resource quantity from the Claim.
// Only one of PerInstanceQuantity or Capacity must be specified.
type NativeResourceQuantity struct {
    // PerInstanceQuantity: Each allocated device instance contributes this Quantity to the native resource.
    // Used when devices in the ResourceSlice represent discrete units of the native resource.
    // +optional
    PerInstanceQuantity resource.Quantity `json:"perInstanceQuantity,omitempty"`

    // Capacity: The native resource quantity is derived from a DRA capacity
    // with the specified QualifiedName. This should match a key in Device.Capacity.
    // +optional
    Capacity QualifiedName `json:"capacity,omitempty"`
}
```

*   **`Device.NativeResourceMappings`**: This new struct within `Device` object in a `ResourceSlice` provides the specific details of how this particular device instance relates to native resources. The key of this map is the native resource name (e.g., `cpu`, `memory`, `hugepages-1Gi`)
    *   **AccountingPolicy:**  Optionally defines a device-specific accounting policy override for this native resource.
    *   **QuantityFrom:** Specifies how the quantity of the native resource allocated to the `ResourceClaim` is derived. This is a struct with mutually exclusive fields:
        *   **PerInstanceQuantity:** Used when each device instance allocated contributes a fixed amount of the native resource. Suitable for models where devices are discrete units (e.g., a "l3Cache" device is always 8 CPU, a "core" device is 2 CPU).
        *   **Capacity:** Used when the native resource quantity is tied to a capacity within the DRA device's definition in the `ResourceSlice` (e.g., drawing from a "cpu-capacity" counter within a NUMA group device). This is used when the resource is represented as a consumable capacity in the resource slice.

#### Accounting Policy Precedence

When determining the `AccountingPolicy` for a native resource from a DRA claim:

1.  The `AccountingPolicy` specified within the `Device.NativeResourceMappings` for the specific `ResourceName` takes highest precedence.
2.  If the `AccountingPolicy` is not set in the `Device` mapping, the policy is taken from the `DeviceClass.Spec.NativeResourcePolicies` map for the matching `ResourceName`.
3.  If no policy is found in either location for a `ResourceName` that has a quantity defined in the `Device` mapping, it is considered an error, and the device will not be allocatable for the claim.

This model supports both **Admin-Defined Policy** and **Driver-Defined Policy**:

*   **Admin-Defined Policy (e.g., CPU/Memory):** A CPU/Memory DRA driver can publish `Device` objects with `NativeResourceMappings` containing only the `QuantityFrom`, leaving the `AccountingPolicy` field unset. The cluster administrator then defines the desired accounting behavior (e.g., `AddPerReference`, `AddPerClaim`) by creating `DeviceClass` objects with appropriate entries in `NativeResourcePolicies`.  This allows different consumption models for the same underlying CPU resources, controlled by the admin.

*   **Driver-Defined Policy (e.g., Accelerators):** An accelerator driver (e.g., for GPUs) often knows the exact auxiliary resources (like CPU or Memory) required and the most appropriate accounting method. The driver can specify both the `QuantityFrom` and the `AccountingPolicy` (e.g., `AddPerReference`) directly in the `Device.NativeResourceMappings`.

This combined approach provides flexibility, allowing the policy to be defined at the most appropriate level.

If a `NativeResourceMapping` entry exists for a resource but `AccountingPolicy` can be resolved from either the `Device` or the `DeviceClass`, this is an invalid configuration. The scheduler will fail to schedule the pod referencing the claim.

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
      - cel: 'device.driver == "dra.example.com"'
      managesNativeResources: true
      nativeResourcePolicies:
        cpu: AddPerClaim
    ---
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
        attributes:
          numaNode: 0
        nativeResourceMappings: # Accounting policy specified in the device class
          cpu: 
            quantityFrom: { perInstanceQuantity: "1" }
      - name: cpu1
        attributes:
          numaNode: 0
        nativeResourceMappings: # Accounting policy specified in the device class
          cpu: 
            quantityFrom: { perInstanceQuantity: "1" }
    # ... other cpu devices
  ```
  
  *   Each device instance (like `cpu0`) in the `ResourceSlice` represents a single unit of CPU.
  *   Each Device uses `nativeResourceMappings` to specify its impact on native resources. The `quantityFrom.PerInstanceQuantity` field indicates the amount of a native resource per device instance. For example, if `cpu0` represents a single CPU thread, this would be "1". If a device represents a physical CPU core (e.g., with 2 threads), `PerInstanceQuantity` would be "2".

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
      - cel: 'device.driver == "dra.example.com"'
      managesNativeResources: true
      nativeResourcePolicies:
        cpu: AddPerClaim
        memory: AddPerClaim
    ---
    # ResourceSlice
    apiVersion: resource.k8s.io/v1
    kind: ResourceSlice
    metadata:
      name: cpu-pool-slice
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
          "dra.example.com/cpu": "128"
          "dra.example.com/memory": "256Gi"
        nativeResourceMappings: 
          cpu: # Accounting policy specified in the device class
            quantityFrom:
              capacity: "dra.example.com/cpu"
          memory: # Accounting policy specified in the device class
            quantityFrom:
              capacity: "dra.example.com/memory"
  ```

3.  **Partitionable Devices**

  *   In the below example CPU is represented as a partitionable device with NUMA Node and L3 cache partitions.
  *   The `node-cpu-counters` CounterSet holds the total 128 CPUs.
  *   Allocating `socket-0-numa-0` would notionally reserve 32 CPUs from `node-cpu-counters` counter set.
  *   Allocating `socket-0-numa-0-l3-0` consumes 8 CPUs from the same `node-cpu-counters`.
  *   `nativeResourceMappings.QuantityFrom.Capacity` links the native resource accounting to this device-specific capacity.

  ```yaml
    # DeviceClass
    apiVersion: resource.k8s.io/v1
    kind: DeviceClass
    metadata:
      name: dra-l3-caches
    spec:
      selectors:
      - cel: 'device.driver == "dra.example.com"'
      managesNativeResources: true # Accounting policy specified in the device
    ---
    apiVersion: resource.k8s.io/v1
    kind: ResourceSlice
    metadata:
      name: cpu-counters-slice
    spec:
      driver: dra.example.com
      sharedCounters:
      - name: node-cpu-counters
        counters:
          "dra.example.com/cpu": { value: "128" }
    ---
    apiVersion: resource.k8s.io/v1
    kind: ResourceSlice
    # ...
    spec:
      # ...
      devices:
      - name: socket-0-l3-0
        attributes:
          dra.example.com/type: l3cache
          dra.example.com/numaID: "0"
        capacity:
          "dra.example.com/cpu": "8" # This L3 cache contains 8 CPUs
        consumesCounters:
        - counterSet: node-cpu-counters
          counters:
            "dra.example.com/cpu": "8"
        nativeResourceMappings:
          cpu:
            accountingPolicy: "AddPerClaim"
            quantityFrom:
              capacity: "dra.example.com/cpu"
      . . .
      - name: socket-0-numa-0
        attributes:
          dra.example.com/type: numa
          dra.example.com/numaID: "0"
        capacity:
          "dra.example.com/cpu": "32" # This numa node contains 32 CPUs
        consumesCounters:
        - counterSet: node-cpu-counters
          counters:
            "dra.example.com/cpu": "32"
        nativeResourceMappings:
          cpu:
            accountingPolicy: "AddPerClaim"
            quantityFrom:
              capacity: "dra.example.com/cpu"
  ```

4.  **Auxiliary native resource requests for Accelerators**

  *   The accelerator device uses `NativeResourceMapping` to indicate it needs additional CPU and Memory. These amounts will be *added* to the pod's total requests.
  *   **Importantly, the native resources specified in `NativeResourceMapping` (e.g., CPU, Memory) are not necessarily managed by the DRA driver in the same way as the accelerator itself.** Instead, this mechanism primarily serves as an accounting system for the kube-scheduler to not overcommit the node.

  ```yaml
    # DeviceClass
    apiVersion: resource.k8s.io/v1
    kind: DeviceClass
    metadata:
      name: ai-accelerators
    spec:
      selectors:
      - cel: 'device.driver == "xpu.example.com"'
      managesNativeResources: true # Accounting policy specified in the device
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
          cpu:
            quantityFrom: { perInstanceQuantity: "2" }
            accountingPolicy: "AddPerReference"
          memory:
            quantityFrom: { perInstanceQuantity: "8Gi" }
            accountingPolicy: "AddPerReference"
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
  // +featureGate=DRANativeResources
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
    // AccountingPolicy specifies how this native resource from the DRA claim
    // was combined with any standard request for the same resource
    // in the pod.spec.containers[].resources.requests.
    // This is the effective policy derived from `DeviceClass.Spec.NativeResourcePolicies` and `Device.NativeResourceMapping` based on precedence rules.
    AccountingPolicy NativeResourceAccountingPolicy
    // Quantity is the amount of native resource allocated.
    Quantity resource.Quantity
    // DriverName is the DRA driver name that allocated this resource.
    DriverName string `json:"driverName,omitempty"`
}

```

#### Kube-Scheduler Workflow

The scheduling process for a Pod involves several stages. The following describes how the `NodeResourcesFit` and `DynamicResources` plugins interact within the kube-scheduler framework to achieve unified accounting for native resources managed by DRA. The key goal is to ensure that the delegation mechanism works regardless of the execution order of these plugins.

1.  **PreFilter Stage:** This stage is for initial checks and pre-computations to quickly filter out non-viable nodes or prepare data for the `Filter` stage, minimizing work on each node.
    *   **NodeResourcesFit Plugin**: It determines if any `ResourceClaim` in the pod spec is associated with a `DeviceClass` that has `Spec.ManagesNativeResources: true`. This result is stored in an internal state object for the current scheduling cycle. It still checks for non-native resources like ephemeral-storage and scalar resources not managed by DRA. The responsibility for native resource capacity checking delegated to the `DynamicResources` plugin.
    *   **DynamicResources Plugin**: Performs standard validation of `ResourceClaim` and `DeviceClass` existence. It also notes if any referenced `DeviceClass` manages native resources.

2.  **Filter Stage:** This stage performs the node-level checks to determine if a pod fits on a specific node.
    *   **NodeResourcesFit Plugin:** If the PreFilter stage indicated that DRA manages native resources for this pod, this plugin *skips* its standard capacity checks for native resources.
    *   **DynamicResources Plugin:**  The `DynamicResources` plugin handles the native resources requested by pods using DRA.
        *   The plugin tries to allocate all the resource claims referenced in the pod.
        *   The plugin validates the compatibility of `AccountingPolicy` settings of all the referenced claims. For the Alpha scope, which only includes additive policies (AddPerClaim, AddPerReference), all combinations are compatible (non-additive policies discussed in [Future Enhancements](#future-enhancements)). The plugin also checks for conflicts with other features, such as the use of Pod Level Resources (discussed in [Future Enhancements](#future-enhancements)). If any validation fails, the pod is rejected.
        *   For each native resource specified in the `Device.NativeResourceMappings`, the plugin determines the effective `AccountingPolicy` by first checking the `AccountingPolicy` field within the `Device`'s mapping. If it's not set, it falls back to the `DeviceClass.spec.NativeResourcePolicies` for the given resource name. An error occurs if no policy can be resolved.
        *   It then calculates the pod's total demand for each native resource by combining any standard PodSpec requests with the native resource mappings from the allocated devices. The `AccountingPolicy` from the `Device`'s `NativeResourceMapping` dictates how the quantities are combined.
        *   It then checks if the node has enough allocatable resources to meet this total effective demand. If all checks pass, the node is considered feasible.
        *   The results of the native resource calculations (quantities and policies per claim) are stored in the internal cycle state for use in the PreBind stage to update the PodStatus.

3.  **PreBind Stage:** This stage performs actions right before the pod is immutably bound to the node.
    *   **DynamicResources Plugin:** The plugin updates the `ResourceClaim.Status` to reflect the allocated devices. It also patches the `Pod.Status` to add the `NativeResourceClaimStatus` field. This new field contains information about the native resources being provided via DRA, including the quantities, the `AccountingPolicy` applied, and which containers reference the claim. This makes the DRA contribution to the pod's native resources explicit in the Pod's status.

4.  **Bind Stage (Framework Cache Update):** This is the final step where the scheduler records the pod's resource consumption on the node in the `NodeInfo` cache. The core logic in the scheduler framework is enhanced to use `pod.Status.NativeResourceClaimStatus` and new fields in `NodeInfo` to track DRA claim states.
    *  The scheduler framework's core logic for updating the `NodeInfo.Requested` is enhanced. When a pod is bound ([`NodeInfo.update`](https://github.com/kubernetes/kubernetes/blob/4925c6bea44efd05082cbe03d02409e0e7201252/pkg/scheduler/framework/types.go#L425) method), the framework reads `pod.Status.NativeResourceClaimStatus` and combines that with the standard request to determine the effective native resources allocated.
    *  Additionally, to manage claims shared by different pods, a field `NativeDRAClaimStates` is added in `NodeInfo` that tracks all the native resource claim allocations. This is used to de-duplicate shared requests during accounting.
  
    ```go
    // In pkg/scheduler/framework/types.go
    type NodeInfo struct {
        // ... existing fields

        // NativeDRAClaimStates tracks the state of native resource DRA claims on this node.
        // The key is the UID of the ResourceClaim.
        NativeDRAClaimStates map[types.UID]*NativeDRAClaimAllocationState
    }

    // NativeDRAClaimAllocationState holds information about a DRA claim's allocation on a node.
    type NativeDRAClaimAllocationState struct {
        // Consumers is a set of Pod UIDs currently consuming this claim on this node.
        Consumers sets.String
    }
    ```
    *  The `NodeInfo.update` method adjusts the total `Requested` resources on the node. The logic for adding or subtracting the DRA-based native resources depends on the `AccountingPolicy`.
       *  With `AddPerClaim`, the native resource quantity for the claim is added to `NodeInfo.Requested` only when the first pod using this claim is added to the node. It's subtracted when the last pod using this claim is removed, as tracked by the `Consumers` set in `NativeDRAClaimStates`.
       *  With `AddPerReference`, the native resource quantity is added to `NodeInfo.Requested` for each time the claim is referenced by containers or pods. When the pods are removed, the quantity is subtracted from `NodeInfo.Requested` per reference.

#### Kubelet Admission Control

The Kubelet has its own admission check ([AdmissionCheck](https://github.com/kubernetes/kubernetes/blob/4925c6bea44efd05082cbe03d02409e0e7201252/pkg/kubelet/lifecycle/predicate.go#L436)) to ensure a pod can run on the node, even after the scheduler has placed it. Currently, the Kubelet's admission process reuses parts of the scheduler's node filtering logic, specifically including the checks from the `NodeResourcesFit` plugin to check if a pod fits on the node.

The core function `[resource.PodRequests](https://github.com/kubernetes/kubernetes/blob/4925c6bea44efd05082cbe03d02409e0e7201252/staging/src/k8s.io/component-helpers/resource/helpers.go#L149)` (from `k8s.io/component-helpers/resource`) is the standard API used in several components, including the Kubelet's admission control and the scheduler's `NodeResourcesFit` plugin to determine the aggregate resource needs of a pod.

To support DRA native resources, the `resource.PodRequests` function would be enhanced. When the `DRANativeResources` feature is enabled, this function now inspects the `pod.Status.NativeResourceClaimStatus`. This field is populated by the `DynamicResources` scheduler plugin and contains the quantities of native resources allocated to the pod via its DRA claims. This enhancement ensures that both the scheduler and the Kubelet's admission control use a consistent method for calculating the pod's total native resource requirements, taking into account both the `pod.Spec` and the DRA allocations recorded in `pod.Status`. The Kubelet, therefore, checks the pod's comprehensive resource footprint against the node's allocatable resources.

#### Multiple Claims per Container

A single container can reference multiple DRA claims that affect the same native resource. Since all currently supported policies are additive, any combination of these policies for the same resource within a container is compatible. The resource quantities from each claim are summed up according to their respective policy rules.

**Example:**
* Combining additive policies.
    ClaimA: `{cpu, AddPerClaim, 4 CPU}`
    ClaimB: `{cpu, AddPerReference, 2 CPU}`
    * Pod 1
      1. Container "c1"
        * Spec: requests 1 CPU
        * claims: [ClaimA, ClaimB]
      2. Container "c2"
        * Spec: requests 2 CPU
        * claims: [ClaimA, ClaimB]
    * **Result:** 
      * Pod Effective CPU = 1 (c1 PodSpec) +  4 (c1 ClaimA) + 2 (c1 ClaimB) + 2 (c2 PodSpec) + 2 (c2 ClaimB) = 11 CPU.
      * Claim A is accounted for only once

#### Shared Claims

##### Multiple Containers Sharing a Claim

When multiple containers within the same pod reference the same `ResourceClaim`, the resource accounting is based on policy. 

*  `AddPerClaim`: The native resource quantity from the claim's `NativeResourceMapping` is accounted for only once for the entire pod.
*  `AddPerReference`: The native resource quantity is counted each time a claim is referenced by a container.

##### Multiple Pods Sharing a Claim

To account for a `ResourceClaim`s shared by multiple pods on the same node, the scheduler uses the `NodeInfo.NativeDRAClaimStates` map. This map tracks the UIDs of pods consuming each claim.

*   For **`PolicyAddPerClaim`**: The native resource quantity is added to the node's total `Requested` resources only when the *first* pod consuming the claim lands on the node. It's subtracted when the *last* consuming pod is removed.
*   For **`PolicyAddPerReference`**: Since this policy is per-reference, each pod contributes to the node's `Requested` resources based on its own container's references to the claim.

#### Unreferenced Claims

If a `ResourceClaim` is listed in `pod.Spec.ResourceClaims` but not referenced by any container in `pod.Spec.Containers[*].Resources.Claims`. The resources associated with this claim ARE still accounted for against the node's capacity. The DRA allocator reserves the devices for the pod, making them unavailable to others. The `PodNativeResourceClaimStatus` entry for this claim will have an empty `Containers` list.

#### Handling Pod Overheads

The unified accounting must include the overheads specified in `pod.Spec.Overhead` in the total resource calculation. The `dynamicResources` plugin sums the effective Requests of all containers (calculated based on standard resource requests and allocation policy) and then adds the values from `pod.Spec.Overhead` to this sum.

### Node Resource Enforcement and Isolation

In the Alpha phase, the Kubelet does not account for native resources requested through DRA for QOS class determination, cgroup management, and eviction decisions. These mechanisms solely rely on the `requests` and `limits` specified in the `pod.spec.containers[*].resources` or `pod.spec.initcontainers[*].resources`. This creates a discrepancy where a user may specify native resource requests through `ResourceClaim`s, but the Kubelet enforces runtime limits based solely on the Pod Spec.
1.  A pod requesting CPU/Memory via DRA claims may be classified as `BestEffort`  (no CPU/Memory requests or limits in its pod spec), or as `Burstable` (limits greater than request), as the DRA-provided resources are not considered in the QoS calculation. The QoS class directly determines the pod's parent directory within the cgroup filesystem hierarchy. This hierarchical directory structure is critical for enforcing resource controls in the linux kernel. 
2.  Kubelet currently sets CPU and memory cgroup settings only based on pod spec. This would result in incorrect runtime enforcements. For CPU, the container could get low CPU shares or could be incorrectly throttled. For memory, if the memory allocation exceeds the limit in the spec, it could be OOM killed.
3.  To prevent a critical system daemon from failing to start, the Kubelet will preempt pods on its node to free up the required requests. This decision is based primarily on QoS Class. Pods with DRA native resource request but a low QoS class (BestEffort or Burstable) would have a higher risk of being evicted under node resource pressure.

**Mitigation:**
*  The user must increase the limits in the Pod Spec to be equal to or greater than the sum of the base container request in the spec and the DRA claim request. This would result in the pod being classified as **Burstable** (limit > request). This ensures the Kubelet sets the Cgroup limit high enough to allow full usage of the DRA resource, preventing throttling or OOMs. The request in the spec need not include claim request as they are already accounted for by the scheduler.
*  For critical infrastructure (e.g., the DRA driver DaemonSet itself), set `priorityClassName` in the Pod Spec to `system-node-critical` or `system-cluster-critical` to reduce the risk of eviction. The high priority class ensures the pod is evaluated last for eviction among all workloads exceeding their requests.

In a future Alpha or Beta stage, the Kubelet will natively calculate effective requests and limits by combining the standard request from the pod spec and the DRA Claim (based on the allocation policy) and configure node level settings like QOS class, Cgroup settings etc. correctly.

### Use Case Walkthroughs

#### Use Case 1: Pod with Standard and DRA CPU and Memory Request

```yaml
# ResourceSlice with Consumable Capacity - Max Policy
apiVersion: resource.k8s.io/v1
kind: ResourceSlice
metadata:
  name: node1-slice
spec:
  driver: dra.example.com
  nodeName: node1
  devices:
  - name: socket0
    attributes: {"dra.example.com/type": "socket"}
    allowMultipleAllocations: true
    capacity:
      "dra.example.com/cpu": "128"
      "dra.example.com/memory": "256Gi"
    nativeResourceMappings:
      cpu: 
        accountingPolicy: AddPerClaim
        quantityFrom:
          capacity: "dra.example.com/cpu"
      memory:
        accountingPolicy: AddPerClaim
        quantityFrom:
          capacity: "dra.example.com/memory"
---
# ResourceClaim for Max
apiVersion: resource.k8s.io/v1
kind: ResourceClaim
metadata:
  name: cpu-mem-claim
spec:
  devices:
    requests:
    - name: cpu-mem-req
      exactly:
        deviceClassName: cpu-mem-socket
        capacity:
          requests:
            "dra.example.com/cpu": "4"
            "dra.example.com/memory": "8Gi"
  ---
  # Pod
  apiVersion: v1
  kind: Pod
  metadata:
    name: dra-pod
  spec:
    containers:
    - name: my-app
      image: my-image
      resources:
        requests:
          cpu: 100m
          memory: 100Mi
        claims:
        - name: "my-cpu-mem-claim"
    resourceClaims:
    - name: "my-cpu-mem-claim"
      resourceClaimName: cpu-mem-claim
```

**Expected behavior:**

*   `NodeResourcesFit`: Skips CPU and Memory checks.
*   `DynamicResources`: Allocates from the `socket0` device in `node1-slice`.
    *   Effective CPU: 4 from claim + 100m from spec
    *   Effective Memory: 8Gi from claim + 100Mi from spec
*   Scheduler Cache Update: Node's requested CPU increases by 4.1, Memory by 8.1 Gi.

#### Use Case 2: Pod with Fungible Resource Claim (GPU or CPU)

```yaml
  # ResourceSlice for CPU (Max Policy)
  apiVersion: resource.k8s.io/v1
  kind: ResourceSlice
  metadata:
    name: node1-slice
  spec:
    driver: dra.example.com
    nodeName: node1
    pool: {name: node1-pool, generation: 1, resourceSliceCount: 1}
    devices:
    - name: socket0
      attributes: {"dra.example.com/type": "socket"}
      allowMultipleAllocations: true
      capacity:
        "dra.example.com/cpu": "128"
      nativeResourceMappings:
       cpu:
        accountingPolicy: AddPerClaim
        quantityFrom:
          capacity: "dra.example.com/cpu"
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
            deviceClassName: gpu-class
            count: 1
          - name: cpu
            deviceClassName: cpu-class
            capacity:
              requests:
                "dra.example.com/cpu": "30"
  ---
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
    *   Selects CPU: Effective CPU 31 (30 from claim + 1 from Spec), 1Gi Mem (PodSpec).
*   Scheduler Cache Update: Node's requested Memory increases by 1Gi, and CPU increases by 1 or 31 (based on device allocated to claim).

#### Use Case 3: Combined Native (DRA CPU) and Auxiliary Request (GPU)

```yaml
# ResourceSlice for CPU 
apiVersion: resource.k8s.io/v1
kind: ResourceSlice
metadata:
  name: node1-slice
spec:
  driver: dra.example.com
  nodeName: node1
  pool: {name: node1-pool, generation: 1, resourceSliceCount: 1}
  devices:
  - name: socket0
    attributes: {"dra.example.com/type": "socket"}
    allowMultipleAllocations: true
    capacity:
      "dra.example.com/cpu": "128"
    nativeResourceMappings:
      cpu:
        accountingPolicy: AddPerClaim
        quantityFrom:
          capacity: "dra.example.com/cpu"
---
# ResourceSlice for GPU
apiVersion: resource.k8s.io/v1
kind: ResourceSlice
metadata:
  name: node1-gpu
spec:
  driver: gpu.example.com
  nodeName: node1
  pool: {name: node1-pool, generation: 1, resourceSliceCount: 1}
  devices:
  - name: gpu0
    nativeResourceMappings:
      cpu:
        quantityFrom: { perInstanceQuantity: "2" }
        accountingPolicy: AddPerReference
      memory:
        quantityFrom: { perInstanceQuantity: "4Gi" }
        accountingPolicy: AddPerReference
---
# ResourceClaim for CPU
apiVersion: resource.k8s.io/v1
kind: ResourceClaim
metadata:
  name: cpu-claim
spec:
  devices:
    requests:
    - name: cpu-req
      exactly:
        deviceClassName: cpu-class
        capacity:
          requests:
            "dra.example.com/cpu": "10"
---
# ResourceClaim for GPU
apiVersion: resource.k8s.io/v1
kind: ResourceClaim
metadata:
  name: gpu-claim
spec:
  devices:
    requests:
    - name: gpu-request
      exactly:
        deviceClassName: gpu-class
        count: 1
---
# Pod
apiVersion: v1
kind: Pod
metadata:
  name: combined-dra-pod
spec:
  containers:
  - name: my-app1
    image: my-image1
    resources:
      requests:
        cpu: "100m" # Will be added to GPU claim's CPU dependency and the CPU claim.
        memory: "1Gi" # Will be ADDED to GPU claim's memory dependency
      claims:
       - name: "my-cpu-claim"
       - name: "my-gpu-claim"
  - name: my-app2
    image: my-image2
    resources:
      requests:
        cpu: "200m" # Will be added to GPU claim's CPU dependency and the CPU claim.
        memory: "2Gi"  # Will be ADDED to GPU claim's memory dependency
      claims:
       - name: "my-cpu-claim"
       - name: "my-gpu-claim"
  resourceClaims:
  - name: "my-cpu-claim"
    resourceClaimName: cpu-claim
  - name: "my-gpu-claim"
    resourceClaimName: my-gpu-claim
```
**Expected Behavior:**

*   `NodeResourcesFit`: Skips CPU checks.
*   `DynamicResources`:
    *   **CPU Claim:** Allocates 10 CPUs
    *   **GPU Claim:** Allocates 1 `gpu0` device.
    *   **Effective CPU:** 
        * 100m from spec for `my-app1` + 10 from `cpu-claim` for `my-app1` and `my-app2` (added only once because of `AddPerClaim`) + 2 from `gpu-claim` for `my-app1` (`AddPerReference`) + 200m from spec for `my-app2` + 2 from `gpu-claim` for `my-app2` (`AddPerReference`) =  14.3 CPUs
    *   **Effective Memory:** 
        * 1Gi from `my-app1` spec + 4GB from `gpu-claim` for `my-app1` + 2Gi from `my-app2` spec + 4GB from `gpu-claim` for `my-app2`= 11 GB.
*   Scheduler Cache Update: Node's requested CPU increases by 14.3, Memory by 11 GB.

### Future Enhancements

#### Kubelet QoS and Cgroup Management

As noted in the Non-Goals, full Kubelet awareness of DRA native resources for QoS classification and cgroup management is not in scope for first Alpha. This work will involve:

*   Updating Kubelet's QoS class calculation to include native resources from `pod.Status.NativeResourceClaimStatus`.
*   Ensuring Kubelet's cgroup manager correctly configures CPU and Memory limits/shares based on the sum of PodSpec requests and DRA-provided native resources.
*   Aligning eviction thresholds with the true resource footprint, including DRA.

#### Integration with InPlacePodResizing

In-Place Pod Resizing (IPPR) allows updating a container's resource requests and limits without restarting the pod. The interaction of IPPR with DRA native resources needs more consideration. If a user triggers an in-place resize for a container's CPU or Memory, this updates the base values in the `pod.Spec`. Since the DRA native resource contributions are currently additive (`AddPerClaim`, `AddPerReference`), the effective resource request for the container will be the *new* resized value from the `pod.Spec` plus the amounts added by the DRA claims. The scheduler's accounting, which uses `resource.PodRequests` and considers the `pod.Status.NativeResourceClaimStatus`, should correctly reflect the new total. The interaction will become more complex with the introduction of non-additive policies like `Max` and `ConsumeFrom` (discussed in [Additional Accounting Policies](#additional-accounting-policies)). A disconnect between user's expectation and the actual state would arise when the standard request is not being used for accounting (with `Max`) of coming out of a shared pool (with `ConsumeFrom`)

**Possible Solution:**

The **Kubelet** should reject IPPR PATCH requests (`/resize` subresource) targeting a resource within a container if that same resource is under the control of a DRA claim with an `Max` or `ConsumeFrom` policy for that container. This check should be added to the Kubelet's admission logic for the `/resize` subresource. The Kubelet can determine this by inspecting the `pod.Status.NativeResourceClaimStatus` field to see which resources are overridden by DRA for each container.

#### Integration with Pod Level Resources

A challenge arises when using DRA to manage native resources (like CPU or memory) on pods that also utilize Pod Level Resources using `pod.Spec.Resources`. The core challenge is to define how these two features should interact. Since pod level resources is designed to set a total for the pod, and DRA modifies container-level needs, their interaction is not straightforward.

The core questions are:

1.  How should a DRA `AccountingPolicy` applied to a container's native resource interact with a `pod.Spec.Resources` request for the same resource type? For example, if a container's CPU is requested through DRA claims, does this affect the CPU amount requested at the pod level?
2.  What is the correct way to aggregate the pod-level requests with the container-level requests that have been modified by DRA policies?
3.  Can shared DRA claims, particularly with policies like `ConsumeFrom`, serve as an alternative mechanism to achieve the goals of pod level resources?

Plan for Alpha:
To prevent ambiguity and potential misconfigurations in the Alpha release, the `DynamicResources` plugin should validate and reject the pod in the filter stage if it uses both pod level resources and native resouce claims. Admission-time validation cannot be done as `ResourceClaim` objects can be created asynchronously.

Further design and community feedback are needed to define the precise semantics and aggregation logic for combining Pod Level Resources with DRA Native Resources. This will involve establishing clear rules for precedence and interaction between `pod.Spec.Resources` and native resource DRA claims for the same resource.

#### Additional Accounting Policies

The following accounting policies are not in scope for the Alpha release of this KEP.

**Definitions:**

1.   **Max:** The effective request is the greater value between the standard container request and the DRA claim for the same resource. 
2.   **ConsumeFrom:** A DRA claim is defined to represent the native resource pool capacity. All the containers or pods referencing the claim are satisfied from the capacity pool defined by the DRA claim. Pods access this pool by referencing the corresponding `ResourceClaim` in their spec.`containers[].resources.claims`. The scheduler ensures that the sum of requests from all containers sharing this claim on a node does not exceed the pool's capacity. The entire pool capacity reserved on the node, making it unavailable for other pods outside this pool.

**NodeInfo Changes for Future Policies:**

To support policies like `ConsumeFrom`, the `NativeDRAClaimAllocationState` struct within `NodeInfo` would need additional fields:

```go
  // NativeDRAClaimAllocationState holds information about a DRA claim's allocation on a node.
  type NativeDRAClaimAllocationState struct {
      // Consumers is a set of Pod UIDs currently consuming this claim on this node.
      Consumers sets.String

      // Allocated represents the total native resources this claim instance
      // reserves on the node, making them unavailable for other general pods.
      // - For ConsumeFrom: This is the total pool capacity.
      Allocated v1.ResourceList

      // Consumed is the aggregated quantity of native resources drawn from
      // this claim's pool by container requests.
      // For ConsumeFrom, this represents the consumed capacity out of the allocated pool.
      Consumed v1.ResourceList
  }
```

*  The `NodeInfo.Requested` field, which sums up the resources used by all pods on the node, is adjusted based on the policy:
    *  `Max`: The native resource amount defined in the claim is added to `NodeInfo.Requested` only when the *first* pod using this claim lands on the node. This amount is subtracted when the last pod using the claim is removed.
    *  `ConsumeFrom`: When the first pod using a `ConsumeFrom` claim is bound to the node, the entire pool capacity specified in the claim is added to `NodeInfo.Requested`, effectively reserving it.  This amount is subtracted when the last pod using the claim is removed.  The actual resources requested by individual containers within their pod spec is tracked in `NodeInfo.NativeDRAClaimStates[claimUID].Consumed`.

##### Resource Representation

1.  **Native Resource as a Consumable Pool in ResourceClaim**

*   The device in `ResourceSlice` represents a consumable pool with `AccountingPolicy` set to `ConsumeFrom`.
*   When the device is assigned to a `ResourceClaim`, the request from the pod's `pod.spec.containers[].resources.requests` is consumed out of the claim's pool.

    ```yaml
      # DeviceClass
      apiVersion: resource.k8s.io/v1
      kind: DeviceClass
      metadata:
        name: shared-cpu-pool
      spec:
        selectors:
        - cel: 'device.driver == "dra.example.com"'
        managesNativeResources: true
        nativeResourcePolicies: 
          cpu: "ConsumeFrom"
      ---
      # ResourceSlice
      apiVersion: resource.k8s.io/v1
      kind: ResourceSlice
      metadata:
        name: shared-cpu-pool-slice
      spec:
        devices:
        - name: shared-pool-instance-1
          allowMultipleAllocations: true
          capacity:
            "dra.example.com/cpu": "128"
          nativeResourceMappings:
            cpu: # Accounting policy specified in the device class
              quantityFrom:
                capacity: "dra.example.com/cpu"
    ```

##### Accounting Policy Compatibility and Validation

Since `Max` and `ConsumeFrom` policies are not additive, we could have complex interations between different claims of a container and the pod spec. Validation rules become necessary to ensure predictable behavior and prevent conflicting resource requests.

The following rules would need to be enforced by the scheduler, within the `DynamicResources` plugin's `Filter stage` to handle these interactions.

1.  If multiple claims affect the same native resource in the same container using `Max`, they must all be from the same DRA driver. The sum of all the claim requests would be considered while comparing with the container spec.
2.  If multiple claims affect the same native resource in the same container using `ConsumeFrom`, they must all be from the same DRA driver.
3.  A container cannot have claims requesting devices with `PolicyConsumeFrom` for a native resource if it also has claims using `PolicyMax`
4.  A container can use a claim with `PolicyMax` for a native resource (e.g., from a CPU DRA driver) to set its base request, while simultaneously using other claims for the same native resource with `PolicyAddPerClaim` or `PolicyAddPerReference` (e.g., from a GPU driver for auxiliary CPU). The scheduler will sum the overridden value with rest of the additive policies while accounting for node resources.
5.  A container can use a claim with `PolicyConsumeFrom` for a native resource to set its base request, while  using other claims for the same native resource with `PolicyAddPerClaim` or `PolicyAddPerReference` (e.g., from a GPU driver for auxiliary CPU). The container's `resources.requests` are still drawn from the `ConsumeFrom` pool and the `PolicyAddPerClaim`/`PolicyAddPerReference` are accounted for against the node's general allocatable resources.

**Invalid Scenarios:**

1. A container cannot have multiple `Max` or `ConsumeFrom` policies for the same resource backed by different drivers
  * Container "c1":
    * ClaimA: {cpu (DriverX), Max, 4 CPU}
    * ClaimB: {cpu (DriverY), ConsumeFrom, 8 CPU}

2.  A container cannot have multiple `ConsumeFrom` policies for the same resource from different drivers
  * Container "c1":
    * ClaimA: {cpu (DriverX), ConsumeFrom, 100 CPU Pool}
    * ClaimB: {cpu (DriverY), ConsumeFrom, 50 CPU Pool}

3.  A container cannot have multiple `Max` policies for the same resource from different drivers
  * Container "c1":
    * ClaimA: {cpu (DriverX), Max, 100 CPU Pool}
    * ClaimB: {cpu (DriverY), Max, 50 CPU Pool}

##### Use Case: Pod Consuming from a Shared CPU Pool

```yaml
  # ResourceSlice with 128 CPU consumable capacity
  apiVersion: resource.k8s.io/v1
  kind: ResourceSlice
  metadata:
    name: shared-cpu-pool-slice
  spec:
    devices:
    - name: shared-pool-instance-1
      capacity:
        "dra.example.com/cpu": "128"
      nativeResourceMappings:
        cpu:
          accountingPolicy: "ConsumeFrom"
          quantityFrom:
            capacity: "dra.example.com/cpu"
  ---
  # ResourceClaim for the shared pool of 100 CPUs
  apiVersion: resource.k8s.io/v1
  kind: ResourceClaim
  metadata:
    name: shared-cpu-claim
  spec:
    devices:
      requests:
      - name: pool
        exactly:
          deviceClassName: shared-cpu-pool
          capacity:
            requests:
              "dra.example.com/cpu": "100"
  ---
  # Pod 1 consumes 10 CPUs from the shared pool
  apiVersion: v1
  kind: Pod
  metadata:
    name: pod1
  spec:
    containers:
    - name: container-a
      resources:
        requests:
          cpu: "10"
        claims:
        - name: my-pool
    resourceClaims:
    - name: my-pool
      resourceClaimName: shared-cpu-claim
  ---
  # Pod 2 consumes 20 CPUs from the shared pool
  apiVersion: v1
  kind: Pod
  metadata:
    name: pod2
  spec:
    containers:
    - name: container-b
      resources:
        requests:
          cpu: "20"
        claims:
        - name: my-pool
    resourceClaims:
    - name: my-pool
      resourceClaimName: shared-cpu-claim
```

**Expected Behavior & Accounting:**

1.  **Scheduling Pod1:**
    *   `NodeResourcesFit`: Skips native resource node fit check as the DeviceClass has `managesNativeResources: true`.
    *   `DynamicResources`: Sees `ConsumeFrom` policy. The claim requested 100 CPUs from the pool. Checks if `container-a`'s request of 10 CPU fits within the 100 CPUs. It does.
    *   `NodeInfo` Update: `NativeDRAClaimStates` for `shared-cpu-claim` UID is created. `Allocated` is set to {cpu: 100}. `Consumed` is set to {cpu: 10}. `NodeInfo.Requested` increases by 100 CPUs.

2.  **Scheduling Pod2:**
    *   `NodeResourcesFit`:  Skips native resource node fit check as the DeviceClass has `managesNativeResources: true`.
    *   `DynamicResources`: Sees `ConsumeFrom`. Retrieves `NativeDRAClaimStates`. `Allocated` (Pool Capacity) is 100, `Consumed` is 10. Remaining pool capacity: 100 - 10 = 90. Checks if `container-b`'s request of 20 CPU fits: 20 <= 90. It fits.
    *   `NodeInfo` Update: `NativeDRAClaimStates` for `shared-cpu-claim` has `Consumed` updated to {cpu: 30}. `Allocated` and `NodeInfo.Requested.MilliCPU` remain unchanged.

3.  **Pod Deletion:**
    *   If Pod1 is deleted: `NodeInfo.update` subtracts 10 from `NativeDRAClaimStates[].Consumed`. `NodeInfo.Requested` is unchanged.
    *   If Pod2 is then deleted: `NodeInfo.update` subtracts 20 from `NativeDRAClaimStates[].Consumed`. `Consumers` becomes empty. The *entire* 100 CPU pool capacity is subtracted from `NodeInfo.Requested`. The `NativeDRAClaimStates` entry for `shared-cpu-claim` is removed.

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

-   Ensuring the new fields in `DeviceClass` and `Device` are validated correctly.
-   Scheduler Plugin Logic (`NodeResourcesFit`, `DynamicResources`):
    -   Verifying the correct deferral of native resource checks in `NodeResourcesFit`.
    -   Verify calculation of total pod native resource demand, respecting the `AccountingPolicy` and `QuantityFrom` settings.
    -   Validate the correct application of AccountingPolicy based on the precedence rules (Device override, DeviceClass default).
    -   Validating the population of `Pod.Status.NativeResourceClaimStatus`.
-   Scheduler Framework:
    -   Testing modifications to `NodeInfo` updates to accurately reflect resource usage based on DRA native resource claims, including shared claim handling.
-   Component helper (`k8s.io/component-helpers/resource`)
    -   Testing the `PodRequests` helper function's updated logic to include DRA native resources from the pod status.
-   Kubelet Admission Check
    -   Verifying that the Admission Check correctly uses the DRA native resource from `Pod.Status`

##### Integration tests

Integration tests will be added in `test/integration/dynamicresource` to cover the end-to-end scheduling flow:

-   Tests to ensure correct interaction between `NodeResourcesFit` and `DynamicResources` plugins. 
-   Tests to ensure accounting policies (`AddPerClaim`, `AddPerReference`) are correctly used for resource aggregation and scheduler's cache (`NodeInfo.Requested`) is correctly updated.
-   Tests to validate the `Pod.Status.NativeResourceClaimStatus` is populated correctly and the kubelet admission check correctly computes the effective pod resource request.

##### e2e tests

E2E tests will be added to `test/e2e/dra`:

-   Verify that standard pods requesting CPU/Memory without DRA are not affected.
-   Tests deploying pods with various native resource claim configurations and verifying they are scheduled correctly.
-   Verify Pods are scheduled on nodes with adequate resources, considering both PodSpec and DRA requests.
-   Test scenarios with multiple containers and pods sharing claims.
-   Test scenarios with multiple claims per pod with different `AccountingPolicy` and `QuantityFrom` settings.

### Graduation Criteria

#### Alpha

-   Feature implemented behind the `DRANativeResources` feature gate and disabled by default.
-   Core API changes for `DeviceClass`, `Device`, and `PodStatus` introduced.
-   Kube-scheduler changes in `DynamicResources` and`NodeResourcesFit` plugins added and the accounting logic updated.
-   All unit and integration tests outlined in the Test Plan are implemented and verified.

#### Beta

-   Gather feedback from alpha.
-   Add E2E tests for kube-scheduler changes.
-   Enhance Kubelet to utilize `Pod.Status.NativeResourceClaimStatus` for accurate QoS classification and cgroup management.
-   E2E tests demonstrating correct Kubelet enforcement and QoS handling.
-   Design and implement scheduler support for non-additive accounting policies like `PolicyMax` and `PolicyConsumeFrom`, including their validation and interaction rules.
-   Define the interactions between DRA native resources and other features like In-Place Pod Resizing (IPPR) and Pod Level Resources.

### Upgrade / Downgrade Strategy

-   **Upgrade:** Enabling the feature gate on an existing cluster is safe. The new accounting logic will apply to any newly scheduled pods or pods that are re-scheduled. Existing pods that are already running on nodes will not have their resource accounting in the scheduler's `NodeInfo` cache immediately updated to reflect DRA native resources. Their DRA-based resources will only be correctly accounted for by the scheduler if they are evicted and rescheduled.

-   **Downgrade:** The scheduler will stop processing the native resource fields in DRA Device and DeviceClass objects. The scheduler's view of their resource usage might be incomplete, potentially leading to oversubscription of the node. Pods already scheduled with DRA native resources will continue to run. Re-enabling the gate will enable correct accounting for new pods, however, `NodeInfo` cache might still be incorrect as pods that were scheduled while the gate was off will not have their DRA native resources reflected in the cache. 

If a pod that was scheduled with the feature enabled and is deleted after the feature gate is disabled, the resources added to the NodeInfo cache from `Pod.Status.NativeResourceClaimStatus` would not be subtracted when the pod is removed. This would result in the scheduler cache overestimating resource usage on the node, potentially preventing new pods from scheduling. This inconsistent state would persist until the node is drained.

### Version Skew Strategy

An older scheduler will not understand the new API fields or perform unified accounting. If `DeviceClass` or `ResourceSlice` objects contain the new fields, they will be ignored.

## Production Readiness Review Questionnaire

### Feature Enablement and Rollback

###### How can this feature be enabled / disabled in a live cluster?

- [x] Feature gate (also fill in values in `kep.yaml`)
  - Feature gate name: `DRANativeResources`
  - Components depending on the feature gate: `kube-scheduler`, `kubelet`, `kube-apiserver`.

###### Does enabling the feature change any default behavior?

No. This feature only takes effect if users create Pods that request native resources via `pod.spec.resourceClaims` and DRA drivers are installed and configured to expose native resources via `nativeResourceMappings` in `ResourceSlice` objects. Existing pods are unaffected.

###### Can the feature be disabled once it has been enabled (i.e. can we roll back the enablement)?

Yes. Disabling the feature gate `DRANativeResources` will prevent new API objects from using the new fields and prevent the scheduler from performing the unified accounting. Pods already scheduled using DRA native resource accounting will continue to run. However, when new pods are scheduled while the gate is disabled, any native resources specified in their DRA claims will not be considered by the scheduler. This can lead to node oversubscription as the scheduler's view of available resources on the node will be incomplete.  

###### What happens if we reenable the feature if it was previously rolled back?

The scheduler will resume its unified accounting logic for pods with DRA native resource claims. API validation for the new fields will be re-enabled. The `NodeInfo` cache may be incorrect as it's not automatically updated to consider native resource claims for pods that were scheduled when the gate was disabled.  This inconsistent state would persist until the node is drained.

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

- `DeviceClass` objects with `spec.ManagesNativeResources: true`.
- `Device` objects within ResourceSlices having non-empty `spec.NativeResourceMappings`.
- Pods with `status.nativeResourceClaimStatus` populated.

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

// NativeResourceMapping, NativeResourceAccountingPolicy, NativeResourceQuantity
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