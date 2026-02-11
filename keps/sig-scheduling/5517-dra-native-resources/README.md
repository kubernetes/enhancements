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
      - [Resource Representation Examples](#resource-representation-examples)
    - [Pod API Changes](#pod-api-changes)
    - [Kube-Scheduler Workflow](#kube-scheduler-workflow)
      - [Resource Calculation](#resource-calculation)
    - [Integration with Pod Level Resources](#integration-with-pod-level-resources)
    - [Handling Shared Claims](#handling-shared-claims)
    - [Multiple Claims per Container](#multiple-claims-per-container)
    - [Unreferenced Claims](#unreferenced-claims)
  - [Kubelet Admission Control](#kubelet-admission-control)
  - [Node Resource Enforcement and Isolation](#node-resource-enforcement-and-isolation)
  - [Use Case Walkthroughs](#use-case-walkthroughs)
    - [Use Case 1: Pod with Standard and DRA CPU and Memory Request](#use-case-1-pod-with-standard-and-dra-cpu-and-memory-request)
    - [Use Case 2: Pod with Fungible Resource Claim (GPU or CPU)](#use-case-2-pod-with-fungible-resource-claim-gpu-or-cpu)
    - [Use Case 3: Combined Native (DRA CPU) and Auxiliary Request (GPU)](#use-case-3-combined-native-dra-cpu-and-auxiliary-request-gpu)
    - [Use Case 4: Pod Level Resources with shared CPU DRA Claim and sidecars](#use-case-4-pod-level-resources-with-shared-cpu-dra-claim-and-sidecars)
  - [Future Enhancements](#future-enhancements)
    - [Kubelet QoS and Cgroup Management](#kubelet-qos-and-cgroup-management)
    - [Kube-Scheduler Scoring and Resource Quota](#kube-scheduler-scoring-and-resource-quota)
    - [Integration with In-Place Pod Vertical Scaling](#integration-with-in-place-pod-vertical-scaling)
    - [Accounting Policies](#accounting-policies)
      - [API with Accounting Policy](#api-with-accounting-policy)
      - [Accounting Policy Precedence](#accounting-policy-precedence)
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
    - [Alpha2 / Beta](#alpha2--beta)
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
- [] "Implementation History" section is up-to-date for milestone
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

This KEP proposes a solution for managing native resources like CPU, Memory and Hugepages with Dynamic Resource Allocation (DRA). Currently, when native resources are managed via DRA, there is a fundamental disconnect across the control plane and the Node. In the scheduler, having two independent accounting systems (one for standard resources, one for DRA) which are managing the same underlying resource, which leads to resource overcommitment. On the node, the kubelet is completely unaware of DRA allocations which may result in incorrect QoS class assignment which has many downstream implications. This forces users into fragile workarounds that are incompatible with all the use cases.

The proposed solution in this KEP addresses the native resource accounting in the kube-scheduler. The standard resource (`NodeResourcesFit` plugin) and DRA (`DynamicResources` plugin) will be enhanced to synchronize their accounting, creating a single, authoritative ledger. The kubelet will also be enhanced to consider the native resource request made through both the pod spec, and the DRA `ResourceClaim` to correctly calculate QoS, configure cgroups, and protect high-priority pods. This provides a robust, backward-compatible solution for advanced resource management in Kubernetes.

## Motivation

Dynamic Resource Allocation (DRA) provides a powerful framework for managing specialized hardware
resources such as GPUs, FPGAs, and high-performance network interfaces. It also enables fine-grained
management of native resources like CPU and Memory, for example, through the
[dra-driver-cpu](https://github.com/kubernetes-sigs/dra-driver-cpu). However, when a native resource
is managed via DRA, while it provides added advantages of being able to specify more detailed
requirements, a fundamental disconnect emerges between the scheduler, the kubelet, and the DRA
framework, which breaks the resource guarantees.

Additionally, specialized resources like accelerators have implicit dependency on native resources
like CPU or Hugepages for the application to interact with it. Currently, users must manually
research and declare these auxiliary native resource requirements, typically as additional requests
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

**Kubelet-Level Guarantee Failure:** The kubelet is the component that enforces resource guarantees on
the node. It determines a pod's Quality of Service (QoS) class, configures its cgroups, and makes
critical lifecycle decisions like eviction based *only* on the `pod.spec`. Because it is unaware of
resources allocated via DRA, it will:

* **Misclassify QoS:** A pod with a guaranteed CPU `ResourceClaim` may be misclassified as
  `BestEffort`. This would have downstream effects like  
  * Apply Incorrect Cgroups: It will set the wrong `cpu.shares` and `cpu.quota`, potentially
    throttling high-performance workloads.  
  * Make Incorrect Eviction Decisions: The misclassified pod will be the first to be evicted under
    node pressure.  
  * Incorrect OOM Score calculation.

Current workarounds for DRA-managed native resources (like
[CPU DRA driver](https://github.com/kubernetes-sigs/dra-driver-cpu)) force users to duplicate
resource requests in both the `ResourceClaim` and the standard `pod.spec.containers.resources`.
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
* To ensure the solution is compatible with different ways native resources can be represented and
  allocated within DRA, including as individual devices, consumable capacities
  ([KEP-5075](https://github.com/kubernetes/enhancements/issues/5075)), and partitionable devices
  ([KEP-4815](https://github.com/kubernetes/enhancements/issues/4815))
* To enable specialized devices, such as accelerators, to declare any auxiliary native resource
  requirements (e.g., CPU, Memory) they depend on for their operation.
* To maintain backward compatibility with existing workloads and ecosystem tools that rely on
  `node.status.allocatable` and the scheduler's view of node resource utilization.

### Non-Goals

* To move all resource management logic into the DRA driver. The Kubelet will remain the primary agent
  for cgroup management and QoS enforcement, ensuring that the benefits of its existing stability and
  lifecycle management features are preserved.  
* To replace the standard `pod.spec.containers.resources` API for requesting native resources. This KEP
  aims to enhance the system by adding a clear path for native resource requests via DRA while ensuring
  it works coherently with the existing PodSpec-based requests.
* Changes to the Kubelet for QoS classification, cgroup management, and eviction logic based on DRA
  native resource allocations are not in scope for the Alpha release of this KEP.
* Interaction with In-Place Pod Resizing and Pod Level Resources will be a non goal for alpha. More
  details in [Future Enhancements](#future-enhancements) section.

## Proposal

This KEP introduces a unified accounting model within the kube-scheduler to integrate native resources managed 
by Dynamic Resource Allocation (DRA) with the scheduler's standard resource tracking. By bridging the gap 
between `pod.spec.resources` and DRA `ResourceClaim` allocations, we can achieve consistent resource accounting 
and prevent node overcommitment.

### Background

To understand the proposed solution, it is essential to first understand how kube-scheduler currently
manage standard resource requests and DRA ResourceClaims.

The Kubernetes scheduler is built on a plugin-based framework that executes a series of stages to place
a pod. This KEP is primarily concerned with the interaction between `NodeResourcesFit` and
`DynamicResource` plugins at the `PreFilter`, `Filter`, and `Bind` stages of the
[scheduling framework](https://kubernetes.io/docs/concepts/scheduling-eviction/scheduling-framework/).

##### Standard Resource Accounting

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
* The `NodeInfo.Requested` value is updated by the  scheduler framework when a pod is "assumed" on the node.
  This happens after a node is selected in the `Scoring` phase, and before the actual binding to the 
  API server, ensuring the cache is accurate for subsequent scheduling decisions.

##### Dynamic Resource Allocation (DRA) Accounting

The `DynamicResources` plugin manages resources requested via `pod.spec.resourceClaims`. Its accounting
system is entirely separate from the standard resources.

* The DRA driver/s on the node reports resource availability through the `ResourceSlice` objects.  
* During the `Filter` stage, the `DynamicResources` plugin determines if the inventory in the
  `ResourceSlice` objects is sufficient to satisfy the pod's `ResourceClaim`, after accounting for
  devices already allocated to other claims.  
* When a pod is scheduled, the `DynamicResources` plugin, in its `PreBind` stage, makes an API call to
  update the `ResourceClaim` object's status. This update makes the allocation permanent and visible
  to the rest of the cluster.

These standard resources and the dynamic resources accounting systems are completely independent. The
`NodeInfo` cache is not aware of allocations recorded in `ResourceClaim` objects, which is the root
cause of the accounting gap for native resources when they are managed through DRA.

### User Stories

**Story 1 (Resource Alignment):** A HPC workload needs a certain number of exclusive CPUs and memory
that are aligned on the same NUMA node as a specific NIC for maximum performance. The user creates a
`ResourceClaim` with co-location constraints to enforce this. The scheduler correctly accounts for the
CPU and memory requests made through the claim, adding them to the node's total requested resources, so
the node is not oversubscribed.

**Story 2 (Dedicated and Shared resources):** A Telco application has some high-priority application
containers and some lower-priority sidecar containers. The user wants to dedicate some CPU cores
exclusively to the application containers for low latency, while allowing sidecar containers to run on
the node's general shared CPU pool. They use DRA to request exclusive cores and standard `pod.spec`
requests for the shared CPU portion. The scheduler should correctly account for both dedicated and shared
requests made through these different mechanisms. 

**Story 3 (Accelerator with Native Resource Dependency):** An AI inference job requests a GPU through
a `ResourceClaim`. The specific GPU model also requires certain number of CPUs and Hugepages that are
required for the application to interact with the accelerator. Instead of requiring the user to know
about these auxiliary CPU and HugePages requests and add it to their PodSpec, the GPU Device can be
configured to declare these dependencies. The Kubernetes scheduler accounts for both the CPU/HugePages
needs for the GPU device and the standard pod spec requests, ensuring the pod lands on a node with
sufficient capacity for all requirements. The user experience is simplified, as they only need to ask
for the primary device they care about.

**Story 4 (Fungibility):** An ML inference job can use either a full GPU or, if none is available, a
slice of 8 exclusive CPUs. The user creates a `ResourceClaim` with a `firstAvailable` list to
represent this fungible need. The scheduler evaluates both paths against a node's available
resources. It finds a node with 8 available CPUs, correctly reserves them in its central `NodeInfo`
cache, and schedules the pod. The user did not need to guess which resource to put in the `pod.spec`.  

### Risks and Mitigations

* Increased API and user complexity by having two ways to request native resources (PodSpec and
  ResourceClaim). To mitigate, the documentation would be enhanced with clear guidelines and use cases
  for DRA for Native Resources.
* Bugs in the kube-scheduler's new accounting logic would lead to incorrect node resource calculations
  and node oversubscription. Extensive unit and integration tests covering various resource claim and
  standard request combinations should help mitigate this. The feature will also be rolled out
  gradually, beginning with an alpha release to gather feedback and address potential concerns.
* Until Kubelet is made DRA-aware for native resources (a non-goal for Alpha), QoS and node-level
  enforcement will not fully reflect DRA allocations. This is an accepted limitation for the initial
  Alpha scope.

## Design Details

The proposal here is to enhance the kube-scheduler to implement a **"Unified Accounting"** model for native resources requested through the standard pod Spec or through Dynamic Resource Allocation (DRA) claims. This involves modifications in `NodeResourcesFit` and `DynamicResources` plugins in how they track resource usage on the node. This also includes updates to the DRA API for drivers to declare native resource implications, and Pod Status to record DRA-based native resource allocations. The core principle is that, when a Pod has native resource requested through a DRA claim, the responsibility for checking the node resource fit is delegated to `DynamicResources` plugin, and standard checks in `NodeResourcesFit` are bypassed. The delegation should ensure correct resource accounting irrespective of the execution order of these plugins.

### API Changes

To support unified accounting for native resources, this KEP proposes API extensions to `DeviceClass` and `Device`. 

#### DeviceClass API Extensions

A new field `NativeResourceAccountingPolicies` is added to `DeviceClassSpec`.

```go
// In k8s.io/api/resource/v1/types.go
type DeviceClassSpec struct {
  // ManagesNativeResources indicates if devices of this class manages native resources like cpu, memory and/or hugepages.
  // +optional
  // +featureGate=DRANativeResources
  ManagesNativeResources bool
}
```
#### Device API Extensions

The new field `NativeResourceMappings` within the `ResourceSlice.Device` spec is used to define the native resource quantities and any device-specific policy overrides.

```go
// In k8s.io/api/resource/v1/types.go
type Device struct {
    // ... existing fields
    // NativeResourceMappings defines the native resource (CPU, Memory, Hugepages) 
    // footprint of this device. This includes resources provided by the device 
    // acting as a source (e.g., a CPU DRA driver exposing CPUs on a NUMA node 
    // as a device) or native resources required as a dependency (e.g., a GPU 
    // requiring host memory or CPU to function). The map's key is the native 
    // resource name (e.g., "cpu", "memory", "hugepages-1Gi").
    // +optional
    // +featureGate=DRANativeResources
    NativeResourceMappings map[ResourceName]NativeResourceMapping
}

type NativeResourceMapping struct {
    // QuantityFrom defines how the quantity of the native resource is 
    // determined.
    QuantityFrom NativeResourceQuantity
    // Other fields, such as AccountingPolicy, may be added in the future.
}

// NativeResourceQuantity defines the method to identify how we obtain native resource quantity from the Claim.
// Only one of PerInstanceQuantity or Capacity must be specified.
type NativeResourceQuantity struct {
    // PerInstanceQuantity specifies a fixed amount of the native resource 
    // for each allocated instance of this device. This is used when the 
    // quantity is constant per device, such as a CPU core providing 1 CPU 
    // or a GPU requiring 2Gi of host memory.
    // +optional
    PerInstanceQuantity resource.Quantity


    // Capacity indicates that the native resource quantity is tied to a 
    // capacity defined in the device's capacity map. The native resource quantity is 
    // derived from the ResourceClaim based on the key defined here. 
    // For example: if "dra.example.com/memory"  is represented as a capacity in the ResourceSlice, 
    // this field is set to "dra.example.com/memory" and the scheduler will look up the specific 
    // quantity allocated to the ResourceClaim for that key to determine the claim's memory footprint.
    // +optional
    Capacity QualifiedName
}
```
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
        nativeResourceMappings:
          cpu: 
            quantityFrom: { perInstanceQuantity: "1" }
      - name: cpu1
        attributes:
          numaNode: 0
        nativeResourceMappings:
          cpu: 
            quantityFrom: { perInstanceQuantity: "1" }
    # ... other cpu devices
  ```
  
  *   Each device instance (like `cpu0`) in the `ResourceSlice` represents a single unit of CPU.
  *   Each Device uses `nativeResourceMappings` to specify its impact on native resources. The `quantityFrom.PerInstanceQuantity` field indicates
      the amount of a native resource per device instance. For example, if `cpu0` represents a single CPU thread, this would be "1".
      If a device represents a physical CPU core (e.g., with 2 threads), `PerInstanceQuantity` would be "2".

2.  **Native resource represented as Consumable Pool**

  *   In this model, a `Device` in the `ResourceSlice` acts as a host for a pool of native resources (e.g., a CPU socket providing 128 cores).
  *   By setting allowMultipleAllocations: true on the device, the DRA framework allows multiple ResourceClaims to be allocated against that same device instance simultaneously
  *   This example uses the `Capacity` field within `QuantityFrom` to link to `device.capacity` for the native resource represented as consumable capacity.
  *   When a `ResourceClaim` is allocated against this device, it might only request a small slice e.g., 8 CPUs from the 128 CPUs available in `dra.example.com/cpu`.
      The `nativeResourceMappings["cpu"]` entry tells the scheduler to look for the `'dra.example.com/cpu'` key within that specific claim's allocation to determine the claim's CPU footprint.
      This ensures only the allocated slice, rather than the entire device capacity, is accounted for on the node.

  ```yaml
    # DeviceClass
    apiVersion: resource.k8s.io/v1
    kind: DeviceClass
    metadata:
      name: additional-cpu-memory
    spec:
      selectors:
      - cel: 'device.driver == "dra.example.com"'
      managesNativeResources: true
    ---
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
          "dra.example.com/cpu":  "128"
          "dra.example.com/memory":  "256Gi"
        nativeResourceMappings: 
          cpu:
            quantityFrom:
              capacity: "dra.example.com/cpu"
          memory:
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
      managesNativeResources: true
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
            quantityFrom:
              capacity: "dra.example.com/cpu"
  ```

4.  **Auxiliary native resource requests for Accelerators**

  *   The accelerator device uses `NativeResourceMapping` to indicate it needs additional CPU and Memory. These amounts will be *added* to the pod's total requests.
  *   **Importantly, the native resources specified in `NativeResourceMapping` are not necessarily managed by the DRA driver in the same way as the accelerator itself.** 
      Instead, this mechanism primarily serves as an accounting system for the kube-scheduler to not overcommit the node.

  ```yaml
    # DeviceClass
    apiVersion: resource.k8s.io/v1
    kind: DeviceClass
    metadata:
      name: ai-accelerators
    spec:
      selectors:
      - cel: 'device.driver == "xpu.example.com"'
      managesNativeResources: true
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
          memory:
            quantityFrom: { perInstanceQuantity: "8Gi" }
  ```

#### Pod API Changes

We add a new field `NativeResourceClaimStatus` to `PodStatus` as a way to pass the allocation details from `DynamicResources` plugin to the kube-scheduler accounting logic.

```go
// In k8s.io/api/core/v1/types.go

// PodStatus represents information about the status of a pod.
type PodStatus struct {
    // ... existing fields

  // NativeResourceClaimStatus contains the status of native resources (like cpu, memory)
  // that were allocated for this pod through DRA claims.
  // +featureGate=DRANativeResources
  // +optional
  NativeResourceClaimStatus []PodNativeResourceClaimStatus
}

// PodNativeResourceClaimStatus describes the status of native resources allocated via DRA.
type PodNativeResourceClaimStatus struct {
  // ClaimInfo holds a reference to the ResourceClaim that resulted in this allocation.
  ClaimInfo ObjectReference
  // Containers lists the names of all containers in this pod that reference the claim.
  Containers []string
  // Resources lists the native resources and quantities allocated by this claim.
  Resources []NativeResourceAllocation
}

// NativeResourceAllocation describes the allocation of a native resource.
type NativeResourceAllocation struct {
    // ResourceName is the native resource name (e.g., "cpu", "memory").
    ResourceName ResourceName
    // Quantity is the amount of native resource allocated through this claim for this resource.
    Quantity resource.Quantity
}
```

#### Kube-Scheduler Workflow

The scheduling process for a Pod involves several stages. The following describes how the `NodeResourcesFit` and
`DynamicResources` plugins interact within the kube-scheduler framework to achieve unified accounting for native resources
managed by DRA. The key goal is to ensure that the delegation mechanism works regardless of the execution order of these
plugins.

1.  **PreFilter Stage:**
    *   **DynamicResources Plugin:**  Validates the `ResourceClaim` and its associated `DeviceClass`. If any `DeviceClass`
        involved in the pod's claims is configured to manage native resources (`deviceClass.Spec.ManagesNativeResources: true`),
        this dependency is recorded in the scheduling cycle state (`framework.CycleState`).
    *   **NodeResourcesFit Plugin:**  We need to check if a `ResourceClaim` in the pod spec is associated with a `DeviceClass`
        that has `Spec.ManagesNativeResources: true`. This is necessary to delegate the resource fit check to the
        `DynamicResources` plugin. For Alpha, as native resource claims can only add to standard requests, the delegation mechanism 
        between the plugins is optional. Without delegation there is a dual resource fit check in both the `NodeResourcesFit`and the
        `DynamicResources` plugins, but the `DynamicResources` plugin's check is the authoritative check. The delegation may become 
        a strict requirement if we introduce non-additive [accounting policies][#accounting-policies].

2.  **Filter Stage:** This stage performs the node-level checks to determine if a pod fits on a specific node.
    *   **NodeResourcesFit Plugin:** In the Alpha stage, this plugin would continue to do the resource fit based on standard requests.
    *   **DynamicResources Plugin:** This plugin takes on the authoritative role for checking native resource fit if any of
        the pods `ResourceClaim`s request for native resources.
        *   The plugin tries to allocate devices to all the resource claims of the pod.
        *   **Claim Resource Calculation:** For each allocated device, we check the `Device.NativeResourceMappings` and
            determines the amount of each native resource (CPU, Memory, etc.) associated with the allocated device instance
            using the `QuantityFrom` field.
            *   If only `NativeResourceMappings[].PerInstanceQuantity` is set, that fixed amount is obtained from this field
            *   If `NativeResourceMappings[].Capacity` is set, the native resource quantity is derived by looking at the
                `ResourceClaim` for which the device is allocated.
        *   The plugin calculates the total effective demand for each native resource by
            *  Summing up container requests from the pod spec requests and the amounts determined from DRA claims.
            *  If a claim is referenced by multiple containers, its accounted for only once.
            *  If pod level resources are also specific, that takes precedence and determines the resource footprint of the pod.
        *   **Validation**: the plugin validation for the below scenarios
            *   If Pod Level Resources are defined, the plugin will validate that the sum of effective 
                requests (standard + DRA claims) does not exceed the budget set at the pod level in `pod.spec.resources`([details](#integration-with-pod-level-resources)).
            *   For Alpha, the plugin would reject a pod with a native resource claim if the claim is referenced by an existing pod ([details](#handling-shared-claims)).
        *   This total effective demand is checked against the node's allocatable resources and node is filtered out if it does not have enough capacity.
        *   The calculated native resource allocations for the pod on this specific node (`PodNativeResourceClaimStatus`) are
            stored in the `CycleState`. This is needed for passing the node-specific allocation details to the later
            `Assume` and `PreBind` stages.

3.  **Scheduler Internal Cache Update:** After a node is selected, the scheduler updates its internal cache to reflect the
    resources consumed by the new pod. This stage is critical for maintaining the internal cache consistent. The scheduler
    framework "assumes" the pod will run on the selected node and updates its cache without waiting for bind (updating the
    API server) to succeed. Without an "assume" step, the scheduler might try to place other pods on the same node using
    stale resource information, potentially leading to oversubscription. The Assume phase reserves the resources in the
    scheduler's in-memory cache immediately.
    *   The scheduler framework retrieves the node-specific `PodNativeResourceClaimStatus` from `CycleState` which was populated 
        during the `DynamicResources` Filter stage.
    *   This is then applied to the in-memory copy of the Pod object's status (`pod.status.nativeResourceClaimStatus`) that the 
        scheduler is about to "assume". This is passed to `NodeInfo` cache update ([update()](https://github.com/kubernetes/kubernetes/blob/8c9c67c000104450cfc5a5f48053a9a84b73cf93/pkg/scheduler/framework/types.go#L425)).
    *   The pod's effective native resource demand is calculated based on standard pod request and native resource claims
        as detailed in the [Resource Calculation](#resource-calculation) section. This is added to `nodeInfo.Requested`.

4.  **PreBind Stage:** This stage performs actions right before the pod is immutably bound to the node.
    *   **DynamicResources Plugin:** The plugin updates the `ResourceClaim.Status` to reflect the allocated devices. It also
        patches the `Pod.Status` to add the `NativeResourceClaimStatus` field , persisting the information calculated during
        the Filter stage (`PodNativeResourceClaimStatus`) and making this information available for components (like
        kubelet).

5.  **Bind Stage:** This stage executes asynchronously after the main scheduling cycle has decided on a node. The scheduler
    listens for pod `Update` events, and transitions the pod from the "assumed" state to "bound" if the bind process
    succeeded. The resource accounting on the `NodeInfo` does not change at this point (as they were previously accounted for
    during the "Assume" step). If the bind fails, or if the Kubelet later rejects the Pod, the scheduler detects this and
    reverts the resource allocation in its cache, decrementing `nodeInfo.Requested`.

#####  Resource Calculation

To ensure consistent resource accounting across multiple consumers, the core logic for calculating a pod's total
resource footprint, including DRA-managed native resources, will be centralized in the `PodRequests` function within the
`k8s.io/component-helpers/resource` package. This helper function is currently used by various components, including scheduler
plugins like `NodeResourcesFit`, `NodeInfo` cache update, and Kubelet's admission handler.

The total native resource requirements for a pod are determined by aggregating the following:
*   If pod level resources are specified for a resource, that determines the overall footprint for the pod. 
    The individual container level requests are not considered and including requests made through claims.
*   It iterates through all containers (init and regular) in the pod and determines the aggregate resource request based on existing logic.
*   If `DRANativeResources` is enabled and the pod's `status.nativeResourceClaimStatus` is populated:
    *   Iterate though each claim and obtain the native resource quantities allocated from `nativeResourceClaimStatus[].resources`
    *   For each resource, the `resources.quantity` is added to the pod's total request.
*   If pod overheads are specified in `pod.spec.overhead`, they are added to the final sum.


#### Integration with Pod Level Resources

When Pod Level Resources are specified (`pod.spec.resources`), it continues to set the overall budget for the pod.
Native resources added to individual containers via DRA claims must be accounted for within this pod-level budget.
The effective resource request for a container is the sum of its base request specified in `spec.containers[].resources.requests` 
and any additional resources allocated through DRA claims.

Currently, with pod level resources, an admission time validation ensures that the sum of container requests does not 
exceed pod level requests. However, this is insufficient for pods with native resource claims, as their exact quantities 
are only determined after the `DynamicResources` scheduler plugin allocates devices. This allocation can be dynamic, 
especially with claims with [prioritized lists](https://github.com/kubernetes/enhancements/blob/master/keps/sig-scheduling/4816-dra-prioritized-list/README.md) (fungobility usecases).
Therefore, the `DynamicResources` plugin must perform an additional validation step during its `Filter` stage. After allocating 
devices to claims and calculating the native resources added, the plugin will verify that the total effective pod demand 
(standard container requests + DRA native resources) does not surpass the limits set in `pod.spec.Resources`.

If a pod requests a specific set of devices via DRA claims, and the resulting native resource footprint 
(base container + DRA additions) exceeds the `pod.spec`.Resources budget, this failure is global to the pod. 
The `DynamicResources` plugin would return `UnschedulableAndUnresolvable`. 


#### Handling Shared Claims

**Intra-Pod Sharing:**
Containers within the same pod can reference the same `ResourceClaim`. The native resources associated with the claim are accounted for 
only once for the entire pod, as described in the Resource Calculation section. The resource calculation shared library function 
`PodRequests()` can effectively handle de-duplication for claims shared within a single pod, as all necessary information is self-contained 
within the Pod scope (standard requests in Spec and DRA requests in `status.NativeResourceClaimStatus`)

**Inter-Pod Sharing:**

In the current Alpha scope, sharing `ResourceClaim`s that manage native resources between different pods and the `DynamicResources` plugin 
would reject (`UnschedulableAndUnresolvable`) the pod referencing a claim referenced by an existing pod. This is becase of the following reasons:

1.  When multiple pods, each potentially having its own Pod Level Resources budget (`pod.spec.resources`), reference the same native DRA claim, 
    it's ambiguous how to attribute the cost of these shared native resources against each pod's individual resource footprint.
2.  Node-level cgroups enforcement would be challenging if native resources can be shared between pods. Dynamically adjusting cgroup settings 
    for all consumer pods as pods referencing the same shared claim start/stop would be extremely complex and hard to support.

A new field `NativeDRAClaimStates` is added in `NodeInfo` to track the state of native resource DRA claims on this node. The `DynamicResources` 
plugin will use `NodeInfo.NativeDRAClaimStates` during the `Filter` stage (validation step) to check if the `ResourceClaim` is assigned to an existing pod.

```go
    // In pkg/scheduler/framework/types.go
    type NodeInfo struct {
        // ... existing fields

        // NativeDRAClaimStates tracks the state of native resource DRA claims on this node.
        // The key is the UID of the ResourceClaim.
        NativeDRAClaimStates map[types.UID]*NativeDRAClaimAllocationState
    }

    // NativeDRAClaimAllocationState holds information about a native resource DRA claim's allocation on a node.
    type NativeDRAClaimAllocationState struct {
      // Pods using this claim on this node.
      ConsumerPods sets.Set[types.UID]
    }
```

#### Multiple Claims per Container

A single container can reference multiple DRA claims. The native resources from each distinct claim is summed up to contribute to the pod's total resource requirements.

**Example:**
* Combining additive policies.
    ClaimA - requests 4 CPUs
    ClaimB - requests 2 CPU
    * Pod 1
      1. Container "c1"
        * Spec: requests 1 CPU
        * claims: ClaimA, ClaimB
      2. Container "c2"
        * Spec: requests 2 CPU
        * claims: ClaimA`
    * **Result:** 
      * Pod Effective CPU = 1 (c1 PodSpec) +  4 (ClaimA) + 2 (ClaimB) + 2 (c2 PodSpec) = 9 CPUs.
      * Claim A is accounted for only once

#### Unreferenced Claims

If a `ResourceClaim` is listed in `pod.spec.resourceClaims` but not referenced by any container in `pod.spec.containers[*].resources.claims`.
The resources associated with this claim ARE still accounted for against the node's capacity once. This is because the DRA allocator allocates
the devices to the claim making them unavailable to others (Eg: exclusive CPUs requested through a claim). This will be enforced in the 
`PodRequests()` helper function when computing the pod resource footprint.

### Kubelet Admission Control

The Kubelet has its own admission check
([AdmissionCheck](https://github.com/kubernetes/kubernetes/blob/4925c6bea44efd05082cbe03d02409e0e7201252/pkg/kubelet/lifecycle/predicate.go#L436))
to ensure a pod can run on the node, even after the scheduler has placed it. It utilizes the `PodRequests()` function from
the `k8s.io/component-helpers/resource`. This shared helper has been enhanced to support unified accounting. When
calculating a pod's requirements, it aggregates the standard requests from pod Spec with the DRA allocations recorded in
`pod.status.nativeResourceClaimStatus`. Because the scheduler populates this status field during the PreBind stage, the
Kubelet validates the pod's comprehensive resource footprint.

### Node Resource Enforcement and Isolation

In the Alpha phase, the Kubelet does not account for native resources requested through DRA for QOS class determination, cgroup management,
and eviction decisions. These mechanisms solely rely on the `requests` and `limits` specified in the `pod.spec.containers[*].resources`
or `pod.spec.initcontainers[*].resources`. This creates a discrepancy where a user may specify native resource requests through `ResourceClaim`s, 
but the Kubelet enforces runtime limits based solely on the `pod.spec`.
1.  A pod requesting CPU/Memory via DRA claims may be classified as `BestEffort`  (no CPU/Memory requests or limits in its pod spec), 
    or as `Burstable` (limits greater than request), as the DRA-provided resources are not considered in the QoS calculation. 
    The QoS class directly determines the pod's parent directory within the cgroup filesystem hierarchy. This hierarchical 
    directory structure is critical for enforcing resource controls in the Linux kernel. 
2.  Kubelet currently sets CPU and memory cgroup settings only based on pod spec. This would result in incorrect runtime enforcements.
    For CPU, the container could get low CPU shares or could be incorrectly throttled. For memory, if the memory allocation exceeds 
    the limit in the spec, it could be OOM killed.
3.  To prevent a critical system daemon from failing to start, the Kubelet will preempt pods on its node to free up the required requests. 
    This decision is based primarily on QoS Class. Pods with DRA native resource request but a low QoS class (BestEffort or Burstable) 
    would have a higher risk of being evicted under node resource pressure.

**Mitigation:**
*   Define an overall pod budget using `pod.spec.resources`, the Kubelet uses this to compute QOS class and set the overall cgroup limits 
    for the pod. The pod's actual runtime usage on the node is bounded by the pod level limits. 
*   If using container level requests and limits, the user must increase the container limits to be equal to or greater than the 
    sum of the base container request in the spec and the DRA claim request. This would result in the pod being classified as 
    **Burstable** (limit > request). This ensures the Kubelet sets the Cgroup limit high enough to allow full usage of the DRA resource, 
    preventing throttling or OOMs. The request in the spec need not include claim request as they are already accounted for by the scheduler.
*   For critical infrastructure (e.g., the DRA driver DaemonSet itself), set `priorityClassName` in the `pod.spec` to 
    `system-node-critical` or `system-cluster-critical` to reduce the risk of eviction. The high priority class ensures
    the pod is evaluated last for eviction among all workloads exceeding their requests.

In a future Alpha or Beta stage, the Kubelet will natively calculate effective requests and limits by combining the standard request from
the pod spec and the DRA Claim and configure node level settings like QOS class, Cgroup settings etc. correctly.

### Use Case Walkthroughs

#### Use Case 1: Pod with Standard and DRA CPU and Memory Request

```yaml
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
        quantityFrom:
          capacity: "dra.example.com/cpu"
      memory:
        quantityFrom:
          capacity: "dra.example.com/memory"
---
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
    - name: my-app1
      image: my-image
      resources:
        requests:
          cpu: 100m
          memory: 100Mi
        claims:
        - name: "my-cpu-mem-claim"
     - name: my-app2
      image: my-image-2
      claims:
        - name: "my-cpu-mem-claim"
    resourceClaims:
    - name: "my-cpu-mem-claim"
      resourceClaimName: cpu-mem-claim
```

**Expected behavior:**

*   `NodeResourcesFit`: Checks node capacity against standard container requests {cpu: 100m, memory: 100Mi}.
*   `DynamicResources`: Allocates from the `socket0` device in `node1-slice`.
    *   **DRA Native Resources:** {cpu: 4, memory: 8Gi} from claim `my-cpu-mem-claim`.
    *   **Standard Container Requests:** {cpu: 100m, memory: 100Mi} from `my-app1`.
    *   **Effective Pod Demand:** {cpu: 4100m, memory: 8.1Gi}
    *   Checks node capacity against Effective Pod Demand. 
*   Scheduler Cache Update: Node's requested resources increase by 4.1 CPU and 8.1Gi Memory.

#### Use Case 2: Pod with Fungible Resource Claim (GPU or CPU)

```yaml
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
  status:
    nativeResourceClaimStatus: # Populated only if CPU device is selected
    - claimInfo:
        name: gpu-or-cpu
      containers:
      - my-app
      resources:
      - resourceName: cpu
        quantity: 30
```

**Expected behavior:**

*   `NodeResourcesFit`: Checks node capacity against standard container requests {cpu: 1, memory: 1Gi}.
*   `DynamicResources`:
    *   **Scenario A: GPU Selected**
        *   DRA Native Resources: None
        *   Standard Container Requests: {cpu: 1, memory: 1Gi}
        *   Effective Pod Demand: {cpu: 1, memory: 1Gi}
        *   Checks node capacity against Effective Pod Demand.
        *   Scheduler Cache Update: Node requested increases by 1 CPU, 1Gi Memory.
    *   **Scenario B: CPU Selected**
        *   DRA Native Resources: {cpu: 30} from claim `gpu-or-cpu`.
        *   Standard Container Requests: {cpu: 1, memory: 1Gi}
        *   Effective Pod Demand: {cpu: 31, memory: 1Gi}
        *   Checks node capacity against Effective Pod Demand.
        *   Scheduler Cache Update: Node requested increases by 31 CPU, 1Gi Memory.

#### Use Case 3: Combined Native (DRA CPU) and Auxiliary Request (GPU)

```yaml
# --- gpu-claim, cpu-claim, and ResourceSlices defined as before ---
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
        cpu: "100m"
        memory: "1Gi"
      claims:
       - name: "my-cpu-claim"
       - name: "my-gpu-claim"
  - name: my-app2
    image: my-image2
    resources:
      requests:
        cpu: "200m"
        memory: "2Gi"
      claims:
       - name: "my-cpu-claim"
       - name: "my-gpu-claim"
  resourceClaims:
  - name: "my-cpu-claim"
    resourceClaimName: cpu-claim
  - name: "my-gpu-claim"
    resourceClaimName: gpu-claim
```
**Expected Behavior:**

*   `NodeResourcesFit`: Checks node capacity against standard container requests {cpu: 300m, memory: 3Gi}.
*   `DynamicResources`:
    *   **DRA Native Resources:**
        *   `my-cpu-claim`: {cpu: 10}
        *   `my-gpu-claim`: {cpu: 2, memory: 4Gi} (Auxiliary)
    *   **Standard Container Requests:**
        *   `my-app1`: {cpu: 100m, memory: 1Gi}
        *   `my-app2`: {cpu: 200m, memory: 2Gi}
    *   **Effective Pod Demand:**
        *   CPU: 100m + 200m + 10 (my-cpu-claim) + 2 (my-gpu-claim) = 12.3 CPU
        *   Memory: 1Gi + 2Gi + 4Gi (my-gpu-claim) = 7Gi
    *   Checks node capacity against Effective Pod Demand.
*   Scheduler Cache Update: Node's requested increases by 12.3 CPU and 7Gi Memory.

#### Use Case 4: Pod Level Resources with shared CPU DRA Claim and sidecars

This use case demonstrates using PLR to set the overall CPU budget for a pod, where two containers share a DRA claim for 10 dedicated CPUs, and two sidecar containers run with best-effort in-pod placement.

```yaml
# --- cpu-req-10-cpus ResourceClaim defined as before ---
# Pod
apiVersion: v1
kind: Pod
metadata:
  name: dra-pod-with-plr-besteffort-sidecars
spec:
  resources:
    requests:
      cpu: "11" # 10 from shared claims + additional 1 for all the sidecars
      memory: "10Gi"
    limits:
      cpu: "11"
      memory: "10Gi"
  containers:
  - name: my-app1
    image: my-image1
    resources:
      claims:
      - name: "cpu-req-10-cpus"
  - name: my-app2
    image: my-image-2
    resources:
      claims:
      - name: "cpu-req-10-cpus"
  - name: sidecar-container-1
    image: my-image-3
  - name: sidecar-container-2
    image: my-image-4
  resourceClaims:
  - name: "cpu-req-10-cpus"
    resourceClaimName: cpu-req-10-cpus
```

**Expected Behavior:**

*   `NodeResourcesFit`: Checks node capacity against PLR {cpu: 11, memory: 10Gi}.
*   `DynamicResources`:
    *   DRA Native Resources: {cpu: 10} from `cpu-req-10-cpus`.
    *   Standard Container Requests: {cpu: 11, memory: 10Gi} (pod level requests take precedence).
    *   Total effective demand for PLR check: {cpu: 11, memory: 10Gi} (pod level requests take precedence).
    *   Checks node capacity against PLR {cpu: 11, memory: 10Gi} (This check is redundant as NodeResourcesFit already does it).
*   Scheduler Cache Update: Node's requested resources increase by 11 CPU and 10Gi Memory.

### Future Enhancements

#### Kubelet QoS and Cgroup Management

As noted in the Non-Goals, full Kubelet awareness of DRA native resources for QoS classification and cgroup management is not in scope for first Alpha.
This work will involve:

*   Updating Kubelet's QoS class calculation to include native resources from `pod.status.nativeResourceClaimStatus`.
*   Ensuring Kubelet's cgroup manager correctly configures CPU and Memory limits/shares based on the sum of PodSpec 
    requests and DRA-provided native resources.
*   Aligning eviction thresholds with the true resource footprint, including DRA.

#### Kube-Scheduler Scoring and Resource Quota

**Scoring:** In the current Alpha, the `NodeResourcesFit` plugin's scoring only considers native resources requested directly in the pod Spec.
It does not yet account for native resources allocated through DRA claims. The `DynamicResources` plugin's scoring is based on the
DRA allocation decisions themselves and is independent of the native resource quantities involved. It may be desirable to unify the 
scoring for native resources instead of independently scoring in two different plugins. 

**Quota:** Currently, `ResourceQuota` only accounts for resources defined in the `pod.spec` and including native resources allocated via DRA
in `ResourceQuota` enforcement is not included in the Alpha scope. 

Enhancing `Scoring` and `ResourceQuota` to be aware of DRA native resources should be considered for a future milestone. 
These components rely on the same `PodRequests()` helper function (from `k8s.io/component-helpers/resource`) used by the scheduler 
framework and plugins to calculate resource footprints. Integrating DRA native resources would involve ensuring this helper is called 
with the appropriate options to include `pod.status.nativeResourceClaimStatus`. The implications of this change need to be discussed.

#### Integration with In-Place Pod Vertical Scaling

In-Place Pod Vertical Scaling allows updating a container's resource requests and limits without
restarting the pod restart. The Kubelet actuates these changes by updating the container's cgroup settings 
to match the new values in the PodSpec.

Kubelet, in this Alpha, does not account for native resources allocated via DRA (i.e., from `pod.status.nativeResourceClaimStatus`) 
when setting container-level cgroups. For alpha, any attempt to use the In-Place Pod Resizing `/resize` subresource on a Pod that 
has entries in `pod.status.nativeResourceClaimStatus` will be rejected by the API server. Validation will be added to the `/resize` 
subresource handler to enforce this.  Integration of In-Place Pod Resizing with DRA Native Resources will addressed during future KEP 
iterations along with the Kubelet enhancements to consider `pod.status.nativeResourceClaimStatus` when calculating and enforcing 
container and pod level cgroup settings
    

####  Accounting Policies

The Alpha release of this KEP implements an implicit native resource accounting policy: any native resource 
quantities specified in the `NativeResourceMappings` of allocated devices are added to the pod's total resource 
requirements, accounted for once per `ResourceClaim`.

Future enhancements could introduce explicit **Native Resource Accounting Policies** to provide more control over
how DRA-based native resources are aggregated with standard PodSpec requests. This would likely involve adding new 
fields, such as `AccountingPolicy`, to the `NativeResourceMapping` struct to specify the desired policy. The impact on 
these accounting policies on existing features like Pod Level Resources and In-Place Pod Vertical Scaling also
needs more consideration.

##### API with Accounting Policy

**Device Class**

```go

// NativeResourceAccountingPolicy defines how native resource quantities like CPU, Memory
//  allocated via DRA are aggregated with standard resource requests in the PodSpec.
type NativeResourceAccountingPolicy string

const (
  // PolicyAddPerClaim indicates that the native resource quantity in the DRA claim 
  // is treated as additional to the pod spec requests. This quantity is accounted 
  // for exactly once per claim instance, regardless of the number of containers referencing it. 
  // This applies whether those referencing containers belong to a single pod or are across different pods.
	PolicyAddPerClaim NativeResourceAccountingPolicy = "AddPerClaim"

  // PolicyAddPerReference indicates that the native resource quantity in the DRA 
  // claim is treated as additional to the pod spec requests. This quantity is 
  // accounted for cumulatively for every reference to the claim. 
  // Each container that references the claim adds the claim's quantity to its 
  // native resource request in the pod spec.
	PolicyAddPerReference NativeResourceAccountingPolicy = "AddPerReference"

  // PolicyMax indicates that effective request is the greater value between the standard container 
  // request and the DRA claim for the same resource.
  PolicyMax NativeResourceAccountingPolicy = "Max"

  // PolicyConsumeFrom indicates that a DRA claim is defined to represent the native 
  // resource pool capacity. A DRA claim is defined to represent the native resource pool capacity. All the
  // containers or pods referencing the claim are satisfied from the capacity pool defined by the DRA
  // claim. Pods access this pool by referencing the corresponding `ResourceClaim` in their
  // `spec.containers[].resources.claims`. The scheduler ensures that the sum of requests from all
  // containers sharing this claim on a node does not exceed the pool's capacity. The entire pool
  // capacity reserved on the node, making it unavailable for other pods outside this pool.
  PolicyConsumeFrom NativeResourceAccountingPolicy = "ConsumeFrom"
)

// In k8s.io/api/resource/v1/types.go
type DeviceClassSpec struct {
  // ManagesNativeResources indicates if devices of this class manages native resources like cpu, memory and/or hugepages.
  // +optional
  // +featureGate=DRANativeResources
  ManagesNativeResources bool
  // NativeResourceAccountingPolicies defines how the native resource represented by the devices 
  // in this class should be accounted for and aggregated with any standard request for the same resource
  // in the pod spec (pod.spec.containers[].resources.requests or `pod.spec`.initContainers[].resources.requests)
  // If an accounting policy is also defined in a Device mapping, that device-specific policy takes 
  // precedence. The map's key is the native resource name (e.g., "cpu", "memory", "hugepages-1Gi").
  // +optional
  // +featureGate=DRANativeResources
  NativeResourceAccountingPolicies map[ResourceName]NativeResourceAccountingPolicy
}
```
**Device**

```go
// In k8s.io/api/resource/v1/types.go
type Device struct {
    // ... existing fields
    // NativeResourceMappings defines the native resource (CPU, Memory, Hugepages) 
    // footprint of this device. This includes resources provided by the device 
    // acting as a source (e.g., a CPU DRA driver exposing CPUs on a NUMA node 
    // as a device) or native resources required as a dependency (e.g., a GPU 
    // requiring host memory or CPU to function). The map's key is the native 
    // resource name (e.g., "cpu", "memory", "hugepages-1Gi").
    // +optional
    // +featureGate=DRANativeResources
    NativeResourceMappings map[ResourceName]NativeResourceMapping
}

type NativeResourceMapping struct {
    // AccountingPolicy defines how the native resource quantity from this mapping
    // should be accounted for and aggregated with any standard request for the same resource
    // in the pod spec (pod.spec.containers[].resources.requests or  `pod.spec`.initContainers[].resources.requests).
    // If not set, the policy defined in the DeviceClass is used.
    // +optional
    AccountingPolicy NativeResourceAccountingPolicy
    . . .
}
```

**Pod Status**

```go
// In k8s.io/api/core/v1/types.go

// PodStatus represents information about the status of a pod.
type PodStatus struct {
  // ... existing fields
  // NativeResourceClaimStatus contains the status of native resources (like cpu, memory)
  // that were allocated for this pod through DRA claims.
  // +featureGate=DRANativeResources
  // +optional
  NativeResourceClaimStatus []PodNativeResourceClaimStatus
}

// PodNativeResourceClaimStatus describes the status of native resources allocated via DRA.
type PodNativeResourceClaimStatus struct {
  ...
  // Resources lists the native resources and quantities allocated by this claim.
  Resources []NativeResourceAllocation
}

// NativeResourceAllocation describes the allocation of a native resource.
type NativeResourceAllocation struct {
  ...
  AccountingPolicy NativeResourceAccountingPolicy
}
```

##### Accounting Policy Precedence

When determining the `AccountingPolicy` for a native resource from a DRA claim:

1.  The `AccountingPolicy` specified within the `Device.NativeResourceMappings` for the specific `ResourceName` takes highest precedence.
2.  If the `AccountingPolicy` is not set in the `Device` mapping, the policy is taken from the `DeviceClass.Spec.NativeResourceAccountingPolicies` map for the matching `ResourceName`.
3.  If no policy is found in either location for a `ResourceName` that has a quantity defined in the `Device` mapping, it is considered an error, and the device will not be allocatable for the claim.

This model supports both **Admin-Defined Policy** and **Driver-Defined Policy**:

*   **Admin-Defined Policy (e.g., CPU/Memory):** A CPU/Memory DRA driver can publish `Device` objects with `NativeResourceMappings` containing only the `QuantityFrom`, leaving the `AccountingPolicy` field unset. The cluster administrator then defines the desired accounting behavior (e.g., `AddPerReference`, `AddPerClaim`) by creating `DeviceClass` objects with appropriate entries in `NativeResourceAccountingPolicies`.  This allows different consumption models for the same underlying CPU resources, controlled by the admin.

*   **Driver-Defined Policy (e.g., Accelerators):** An accelerator driver (e.g., for GPUs) often knows the exact auxiliary resources (like CPU or Memory) required and the most appropriate accounting method. The driver can specify both the `QuantityFrom` and the `AccountingPolicy` (e.g., `AddPerReference`) directly in the `Device.NativeResourceMappings`.

This combined approach provides flexibility, allowing the policy to be defined at the most appropriate level.

If a `NativeResourceMapping` entry exists for a resource but `AccountingPolicy` missing from both the `Device` mapping and the `DeviceClass`, this is an invalid configuration. The scheduler will fail to schedule the pod referencing the claim.

##### Resource Representation

1.  **Native Resource as a Consumable Pool in ResourceClaim**

*   The device in `ResourceSlice` represents a consumable pool with `AccountingPolicy` set to `ConsumeFrom`.
*   When the device is assigned to a `ResourceClaim`, the request from the pod's
    `pod.spec.containers[].resources.requests` is consumed out of the claim's pool.

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
        nativeResourceAccountingPolicies: 
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

Since `Max` and `ConsumeFrom` policies are not additive, we could have complex interactions between
different claims of a container and the pod spec. Validation rules become necessary to ensure
predictable behavior and prevent conflicting resource requests.

The following rules would need to be enforced by the scheduler, within the `DynamicResources` plugin's
`Filter stage` to handle these interactions.

1.  If multiple claims affect the same native resource in the same container using `Max`, they must all
    be from the same DRA driver. The sum of all the claim requests would be considered while comparing
    with the container spec.
2.  If multiple claims affect the same native resource in the same container using `ConsumeFrom`, they
    must all be from the same DRA driver.
3.  A container cannot have claims requesting devices with `PolicyConsumeFrom` for a native resource if
    it also has claims using `PolicyMax`
4.  A container can use a claim with `PolicyMax` for a native resource (e.g., from a CPU DRA driver) to
    set its base request, while simultaneously using other claims for the same native resource with
    `PolicyAddPerClaim` or `PolicyAddPerReference` (e.g., from a GPU driver for auxiliary CPU). The
    scheduler will sum the overridden value with rest of the additive policies while accounting for
    node resources.
5.  A container can use a claim with `PolicyConsumeFrom` for a native resource to set its base request,
    while  using other claims for the same native resource with `PolicyAddPerClaim` or
    `PolicyAddPerReference` (e.g., from a GPU driver for auxiliary CPU). The container's
    `resources.requests` are still drawn from the `ConsumeFrom` pool and the
    `PolicyAddPerClaim`/`PolicyAddPerReference` are accounted for against the node's general
    allocatable resources.

**Invalid Scenarios:**

1. A container cannot have multiple `Max` or `ConsumeFrom` policies for the same resource backed by
   different drivers
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
    *   `NodeResourcesFit`: Skips native resource node fit check as the DeviceClass has
        `managesNativeResources: true`.
    *   `DynamicResources`: Sees `ConsumeFrom` policy. The claim requested 100 CPUs from the pool.
        Checks if `container-a`'s request of 10 CPU fits within the 100 CPUs. It does.
    *   `NodeInfo` Update: `NativeDRAClaimStates` for `shared-cpu-claim` UID is created. `Allocated`
        is set to {cpu: 100}. `Consumed` is set to {cpu: 10}. `NodeInfo.Requested` increases by 100 CPUs.

2.  **Scheduling Pod2:**
    *   `NodeResourcesFit`:  Skips native resource node fit check as the DeviceClass has
        `managesNativeResources: true`.
    *   `DynamicResources`: Sees `ConsumeFrom`. Retrieves `NativeDRAClaimStates`. `Allocated` (Pool
        Capacity) is 100, `Consumed` is 10. Remaining pool capacity: 100 - 10 = 90. Checks if
        `container-b`'s request of 20 CPU fits: 20 <= 90. It fits.
    *   `NodeInfo` Update: `NativeDRAClaimStates` for `shared-cpu-claim` has `Consumed` updated to {cpu: 30}.
        `Allocated` and `NodeInfo.Requested.MilliCPU` remain unchanged.

3.  **Pod Deletion:**
    *   If Pod1 is deleted: `NodeInfo.update` subtracts 10 from `NativeDRAClaimStates[].Consumed`.
        `NodeInfo.Requested` is unchanged.
    *   If Pod2 is then deleted: `NodeInfo.update` subtracts 20 from `NativeDRAClaimStates[].Consumed`.
        `Consumers` becomes empty. The *entire* 100 CPU pool capacity is subtracted from
        `NodeInfo.Requested`. The `NativeDRAClaimStates` entry for `shared-cpu-claim` is removed.

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
    -   Verify the accurate calculation of a pod's total native resource demand, ensuring it correctly considers standard `pod.spec` requests
        with DRA-based allocations. These tests must cover all supported ways to model native resource including scenarios involving 
        consumable capacity, partitionable devices, and auxiliary resource requests for other devices.
    -   Validating that `pod.status.nativeResourceClaimStatus` is updated correctly.
-   Scheduler Framework:
    -   Verify `NodeInfo` cache updates correctly in the `Assume` stage and reflects resources allocated to native resource claims.
    -   Verify that when a pod using DRA native resources is deleted, the resources are correctly released 
        and become available for other pods in the scheduler's cache.
-   Component helper (`k8s.io/component-helpers/resource`)
    -   Testing the `PodRequests` helper function's updated logic to include DRA native resources.
        -   Ensure existing calculations for pods without DRA claims or PLR remain correct, properly aggregating init 
            and regular container requests.
        -   Verify pod level resources when specified for a resource, continues to take precedence over per-container requests,
            include native claim requests.
        -   Verify that the native resources from `pod.status.nativeResourceClaimStatus` are correctly added to the pod's effective standard resource requests.
        -   Test that existing logic for different `PodResourcesOptions` (e.g., `ExcludeOverhead`, `SkipPodLevelResources`) continues to 
            work as expected when DRA native resources are present, including correct handling of `pod.spec.overhead`.
-   Kubelet Admission Check
    -   Verifying that the admission check correctly uses the DRA native resource from the pod's `status.nativeResourceClaimStatus` field.

** Current Test Coverage:**
- `pkg/scheduler/framework/plugins/dynamicresources`: `20260203` - `79.2`
- `pkg/scheduler/framework/plugins/noderesources`: `20260203` - `89.6`
- `pkg/scheduler/schedule_one.go`: `20260203` - `86.6`
- `pkg/scheduler/framework/types.go`: `20260203` - `66.4`
- `pkg/scheduler/eventhandlers.go`: `20260203` - `71.4`
- `staging/src/k8s.io/component-helpers/resource/helpers.go`: `20260203` - `82.4`


##### Integration tests

Integration tests will be added in `test/integration/dynamicresource` to cover the end-to-end scheduling flow:

**Kube-Scheduler:**
-   Tests to ensure correct interaction between `NodeResourcesFit` and `DynamicResources` plugins.
-   Test that the scheduler's internal cache (`NodeInfo.Requested`) is accurately updated to reflect 
    the resources consumed by pods with DRA native resource claims.
-   Ensure that resources are correctly released in the scheduler cache when a pod with DRA native resources is deleted.
-   Validate that fungible claims resulting in different native resource footprints are accounted for correctly on a per-node basis.
-   Tests to validate the `pod.status.nativeResourceClaimStatus` is populated correctly and the kubelet
    admission check correctly computes the effective pod resource request.

**Kubelet:**
-   Test that the Kubelet's admission handler correctly factors in the native resources specified in `pod.status.nativeResourceClaimStatus` 
    when deciding whether to admit a pod.

##### e2e tests

E2E tests will be added to `test/e2e/dra`:

-   Verify these pods are scheduled onto nodes with sufficient capacity, considering both the pod's standard requests and the DRA-added native resources.
    These tests should cover various DRA modeling scenarios:
    -   Native resources as individual devices. 
    -   Native resources as consumable capacity from a pool.
    -   Native resources from partitionable devices.
    -   Auxiliary native resources required by other devices (e.g., additional memory for an accelerator).
    -   Fungible claims involving native resources

### Graduation Criteria

#### Alpha

-   Feature implemented behind the `DRANativeResources` feature gate and disabled by default.
-   Core API changes for `DeviceClass`, `Device`, and `PodStatus` introduced.
-   Kube-Scheduler:
    *   The `DynamicResources` plugin is updated to calculate and enforce node resource fit based on standard requests and native resource claims.
    *   The scheduler's internal cache update logic is enhanced to incorporate DRA native resource allocations.
-    `k8s.io/component-helpers/resource` shared library is enhanced to compute effective pod resource footprint.
-   The Kubelet's admission handler is updated to consider native resource claims in `Pod.Status`.
-   All unit and integration tests outlined in the Test Plan are implemented and verified.

#### Alpha2 / Beta

-   Gather feedback from alpha.
-   Enhance Kubelet to utilize `pod.status.nativeResourceClaimStatus` for accurate QoS classification and cgroup management.
-   Design and implement support for different accounting policies with native resouce claims and standard requests.
-   Define the interactions between DRA native resources and In-Place Pod Vertical Scaling.
-   Add E2E tests for kube-scheduler and Kubelet changes, including correct QOS and cgroup enforcement.

### Upgrade / Downgrade Strategy

-   **Upgrade:** Enabling the feature gate on an existing cluster is safe. The new accounting logic will apply
    to any newly scheduled pods or pods that are re-scheduled. Existing pods with native resource claims would 
    continue to run, but their claim request will not be reflected in the scheduler's `NodeInfo` cache as these 
    pods lack `pod.status.nativeResourceClaimStatus` field. To fully resynchronize the accounting, the pods with 
    native resources claims must be restarted.

-   **Downgrade:** Disabling the feature gate requires a kube-scheduler restart. Upon startup, the scheduler rebuilds
    the NodeInfo cache without considering DRA native resources. The scheduler's view of resource usage for existing pods 
    will be incomplete (underestimated) as it does not consider claim based requests. This could potentially lead to 
    oversubscription of the node if new pods are scheduled.


### Version Skew Strategy

An older scheduler will not understand the new API fields or perform unified accounting. If `DeviceClass` or
`ResourceSlice` objects contain the new fields, they will be ignored.

## Production Readiness Review Questionnaire

### Feature Enablement and Rollback

###### How can this feature be enabled / disabled in a live cluster?

- [x] Feature gate (also fill in values in `kep.yaml`)
  - Feature gate name: `DRANativeResources`
  - Components depending on the feature gate: `kube-scheduler`, `kubelet`, `kube-apiserver`.

###### Does enabling the feature change any default behavior?

No. This feature only takes effect if users create Pods that request native resources via
`pod.spec.resourceClaims` and DRA drivers are installed and configured to expose native resources via
`nativeResourceMappings` in `ResourceSlice` objects. Existing pods are unaffected.

###### Can the feature be disabled once it has been enabled (i.e. can we roll back the enablement)?

Yes. Disabling the feature gate `DRANativeResources` will prevent the scheduler from performing the unified accounting. 
Pods already scheduled using DRA native resource accounting will continue to run. However, when new pods are scheduled 
while the gate is disabled, any native resources specified in their DRA claims will not be considered by the scheduler.
This can lead to node oversubscription as the scheduler's view of available resources on the node will be incomplete.  

###### What happens if we reenable the feature if it was previously rolled back?

The scheduler will resume its unified accounting logic for pods with DRA native resource claims. API
validation for the new fields will be re-enabled. The `NodeInfo` cache may be incorrect as it's not
retroactively updated to consider native resource claims for previously scheduled pods. This inconsistent 
state would persist until kube-scheduler restarts or all pods with native resources claims are restarted.

###### Are there any tests for feature enablement/disablement?

Unit tests in `kube-scheduler` and `kube-apiserver` will verify the behavior of the scheduler plugins
(`NodeResourcesFit`, `DynamicResources`) and API validation with the feature gate enabled and disabled.

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

- `DeviceClass` objects with `spec.managesNativeResources: true`.
- `Device` objects within `ResourceSlice` having non-empty `nativeResourceMappings`.
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
- [x] API .status
    - Other field: pod.status.nativeResourceClaimStatus
    - Details: Pod's referencing native resource claims should have the pod status updated with `nativeResourceClaimStatus`.
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

No
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

No
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

No. The this KEP proposes extensions to an existing type, but not a new type itself.
<!--
Describe them, providing:
  - API type
  - Supported number of objects per cluster
  - Supported number of objects per namespace (for namespace-scoped objects)
-->

###### Will enabling / using this feature result in any new calls to the cloud provider?

No.
<!--
Describe them, providing:
  - Which API(s):
  - Estimated increase:
-->

###### Will enabling / using this feature result in increasing size or count of the existing API objects?

Yes. With the API changes proposed in this KEP, individual `DeviceClass`, `ResourceSlice` and `Pod` objects would have additional fields, thus increasing their overall signature.

<!--
Describe them, providing:
  - API type(s):
  - Estimated increase in size: (e.g., new annotation of size 32B)
  - Estimated amount of new objects: (e.g., new Object X for every existing Pod)
-->

###### Will enabling / using this feature result in increasing time taken by any operations covered by existing SLIs/SLOs?

Yes. The time to schedule a pod would increase if it reference claims with native resources. The `DynamicResources` 
scheduler plugin would need to allocate the device to the pod and would also need to perform additional 
validations and node resource fit check.

<!--
Look at the [existing SLIs/SLOs].

Think about adding additional work or introducing new steps in between
(e.g. need to do X to start a container), etc. Please describe the details.

[existing SLIs/SLOs]: https://git.k8s.io/community/sig-scalability/slos/slos.md#kubernetes-slisslos
-->

###### Will enabling / using this feature result in non-negligible increase of resource usage (CPU, RAM, disk, IO, ...) in any components?

No.
<!--
Things to keep in mind include: additional in-memory state, additional
non-trivial computations, excessive access to disks (including increased log
volume), significant amount of data sent and/or received over network, etc.
This through this both in small and large cases, again with respect to the
[supported limits].

[supported limits]: https://git.k8s.io/community//sig-scalability/configs-and-limits/thresholds.md
-->

###### Can enabling / using this feature result in resource exhaustion of some node resources (PIDs, sockets, inodes, etc.)?

No
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

// NativeResourceAccountingPolicy, NativeResourceAccountingPolicy, NativeResourceQuantity
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