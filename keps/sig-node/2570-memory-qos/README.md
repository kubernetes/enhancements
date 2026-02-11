# KEP-2570: Support Memory QoS with cgroups v2
<!-- toc -->
- [Release Signoff Checklist](#release-signoff-checklist)
- [Latest Update](#latest-update)
  - [Why Alpha v3 Instead of Beta](#why-alpha-v3-instead-of-beta)
  - [Future Considerations (Beta candidates)](#future-considerations-beta-candidates)
  - [Previous Status (v1.28)](#previous-status-v128)
- [Summary](#summary)
- [Motivation](#motivation)
  - [Goals](#goals)
  - [Non-Goals](#non-goals)
- [Proposal](#proposal)
    - [Alpha v1.22](#alpha-v122)
    - [Alpha v1.27](#alpha-v127)
    - [Beta v1.28 - Cancelled](#beta-v128---cancelled)
    - [Alpha v1.36](#alpha-v136)
  - [User Stories (Optional)](#user-stories-optional)
    - [Memory Sensitive Workload](#memory-sensitive-workload)
    - [Node Availability](#node-availability)
  - [Comparison with Memory Manager](#comparison-with-memory-manager)
  - [memory.low vs memory.min](#memorylow-vs-memorymin)
  - [Notes/Constraints/Caveats (Optional)](#notesconstraintscaveats-optional)
  - [Risks and Mitigations](#risks-and-mitigations)
- [Design Details](#design-details)
  - [Prerequisite](#prerequisite)
  - [Feature Gate](#feature-gate)
  - [Mapping Rules](#mapping-rules)
    - [Container/Pod](#containerpod)
    - [Node](#node)
  - [Interactive](#interactive)
  - [Workflow](#workflow)
    - [Container](#container)
    - [Pod](#pod)
    - [QoS](#qos)
    - [Node](#node-1)
  - [Cgroup Hierarchy](#cgroup-hierarchy)
  - [Cgroup v2 Support](#cgroup-v2-support)
  - [Container Runtime Interface (CRI) Changes](#container-runtime-interface-cri-changes)
  - [Test Plan](#test-plan)
      - [Prerequisite testing updates](#prerequisite-testing-updates)
      - [Unit tests](#unit-tests)
      - [Integration tests](#integration-tests)
      - [e2e tests](#e2e-tests)
  - [Graduation Criteria](#graduation-criteria)
    - [Alpha Graduation (v1.36 - Alpha v3)](#alpha-graduation-v136---alpha-v3)
    - [Beta Graduation](#beta-graduation)
    - [GA Graduation](#ga-graduation)
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
<!-- /toc -->

## Release Signoff Checklist
- [x] (R) Enhancement issue in release milestone, which links to KEP dir in [kubernetes/enhancements] (not the initial KEP PR)
- [x] (R) KEP approvers have approved the KEP status as `implementable`
- [x] (R) Design details are appropriately documented
- [x] (R) Test plan is in place, giving consideration to SIG Architecture and SIG Testing input (including test refactors)
- [x] (R) Graduation criteria is in place
- [x] (R) Production readiness review completed
- [x] (R) Production readiness review approved
- [x] "Implementation History" section is up-to-date for milestone
- [ ] User-facing documentation has been created in [kubernetes/website], for publication to [kubernetes.io]
- [ ] Supporting documentation—e.g., additional design documents, links to mailing list discussions/SIG meetings, relevant PRs/issues, release notes


## Latest Update

Targeting Alpha-3 in v1.36. The livelock issue that blocked Beta in v1.28 has been resolved—kernel 5.9+ contains a fix ([commit b3ff929](https://github.com/torvalds/linux/commit/b3ff92916af3b458712110bb83976a23471c12fa)) that prevents indefinite throttling at `memory.high`. See [Alpha v1.36](#alpha-v136) for details.

### Why Alpha v3 Instead of Beta

This release is Alpha v3 rather than Beta based on PRR feedback:

1. **New KubeletConfiguration field requires Alpha iteration**: Adding `memoryReservationPolicy` to KubeletConfiguration is an API-level change. Per Kubernetes policy, features adding API fields must start in Alpha. This field provides independent control over `memory.min` protection, addressing concerns about validating memory.min behavior separately.

2. **memory.min stability concerns need benchmark validation**: Mapping `requests.memory` to `memory.min` could impact node stability under memory pressure if too much memory is protected for pods. The design mitigates this by also setting `memory.min` for kube-reserved and system-reserved cgroups when `--enforce-node-allocatable` is configured. The scheduler ensures `sum(requests) ≤ allocatable`, and the kernel invokes OOM killer (not hang) when `memory.min` cannot be satisfied. Benchmark testing under sustained memory pressure is needed to validate this before Beta.

3. **Scope consolidation**: The Alpha v3 iteration adds `memoryReservationPolicy` for independent memory.min control, new observability metrics, a kernel version check to mitigate livelock issues, and documented failure modes—addressing the gaps identified since the v1.28 Beta attempt was cancelled.

### Future Considerations (Beta candidates)

1. **Tiered memory protection (memory.low for Burstable/BestEffort)**: Currently, `memory.min` (hard protection) is used for all QoS classes. If Alpha v3 feedback or benchmarks show that padded requests cause excessive OOM thrash, implement a tiered approach for Beta: `memory.min` for Guaranteed pods, `memory.low` (soft protection) for Burstable/BestEffort. This allows the kernel to reclaim unused memory under pressure while still providing best-effort protection. Tracked in [kubernetes/kubernetes#131077](https://github.com/kubernetes/kubernetes/issues/131077).

2. **Benchmark testing**: Before Beta graduation, benchmark testing is planned to validate node behavior under sustained memory pressure with `sum(memory.min)` near capacity.

### Previous Status (v1.28)

Work on Memory QoS was paused after issues were uncovered during the Beta promotion process
in v1.28. This section documents the lessons learned from that experience.
Note: Kubernetes 1.28 did not receive the Beta promotion.

Initial Plan: Use cgroup v2 memory.high knob to set memory throttling limit. As per the initial understanding, 
setting memory.high would have caused memory allocation to be slowed down once the memory usage level in the containers
reached `memory.high` level. When memory usage goes beyond memory.max, kernel will trigger OOM Kill.

Actual Finding: According to the [test results](https://docs.google.com/document/d/1mY0MTT34P-Eyv5G1t_Pqs4OWyIH-cg9caRKWmqYlSbI/edit?usp=sharing), it was observed that for a container process trying to allocate large chunks of memory, once the memory.high level is reached,
it doesn't progress further and stays stuck indefinitely. Upon investigating further, it was observed that when memory usage 
within a cgroup reaches the memory.high level, the kernel initiates memory reclaim as expected. However, the process gets stuck
because its memory consumption rate is faster than what the memory reclaim can recover. This creates a livelock situation where
the process rapidly consumes the memory reclaimed by the kernel causing the memory usage to reach memory.high level again, 
leading to another round of memory reclamation by the kernel. By increasingly slowing growth in memory usage, it becomes
harder and harder for workloads to reach the memory.max intervention point. (Ref: https://lkml.org/lkml/2023/6/1/1300)

Note: The original plan suggested using PSI with memory.high to implement userspace OOM policies. With the kernel 5.9+ fix preventing livelock, this is no longer required. PSI metrics are used only for memory throttling debugging and observability. See [Monitoring Requirements](#monitoring-requirements) for the updated approach. 

See [Future Considerations](#future-considerations) for planned follow-up work including `memory.low` for Burstable/BestEffort pods and benchmark testing.

## Summary
This KEP introduces Memory Quality of Service (QoS) support for Kubernetes using cgroups v2 memory controller features. It maps pod memory requests to `memory.min` (guaranteed memory protection from reclaim) and calculates `memory.high` (throttling threshold) based on a configurable throttling factor. This enables better memory isolation and protection for containerized workloads.

## Motivation
In traditional cgroups v1 implementation in Kubernetes, we can only limit CPU resources, such as `cpu_shares / cpu_set / cpu_quota / cpu_period`, memory QoS has not been implemented yet. cgroups v2 brings new capabilities for memory controller and it would help Kubernetes enhance memory isolation quality.

### Goals
- Provide guarantees around memory availability for pod and container memory requests and limits
- Provide guarantees around memory availability for node resource
- Make use of new cgroup v2 memory knobs(`memory.min/memory.high`) for pod and container level cgroup
- Make use of new cgroup v2 memory knobs(`memory.min`) for node level cgroup

### Non-Goals
- Additional QoS design
- Support other resources QoS
- Consider QOSReserved feature

## Proposal
This proposal uses memory controller of cgroups v2 to support memory qos for guaranteeing pod/container memory requests/limits and node resource.

Currently we only use `memory.limit_in_bytes=sum(pod.spec.containers.resources.limits[memory])` with cgroups v1 and `memory.max=sum(pod.spec.containers.resources.limits[memory])` with cgroups v2 to limit memory usage. `resources.requests[memory]` is not yet used by either cgroups v1 or cgroups v2 to protect memory requests. About memory protection, we use `oom_scores` to determine the order of killing container processes when OOM occurs. Besides, kubelet can only reserve memory from node allocatable at node level, there is no other memory protection for node resources.

So some memory protection is missing, which may cause:
- Pod/Container memory requests can't be fully reserved, page cache is at risk of being recycled
- Pod/Container memory allocation is not well protected, and allocation latency may occur frequently when node memory nearly runs out
- Memory overcommit of container is not throttled, which may increase the risk of node memory pressure
- Memory resource of node can't be fully retained and protected

Cgroups v2 introduces a better way to protect and guarantee memory quality.

| File | Description |
| -------- | -------- |
| memory.min | memory.min specifies a minimum amount of memory the cgroup must always retain, i.e., memory that can never be reclaimed by the system. This protects the cgroup's memory from reclaim pressure. If the system cannot free enough memory because too much is protected by memory.min across cgroups, the OOM killer will be invoked. **We map it to `requests.memory`.** |
| memory.max | memory.max is the memory usage hard limit, acting as the final protection mechanism: If a cgroup's memory usage reaches this limit and can't be reduced, the system OOM killer is invoked on the cgroup. Under certain circumstances, usage may go over the memory.high limit temporarily. When the high limit is used and monitored properly, memory.max serves mainly to provide the final safety net. The default is max. **We map it to `limits.memory` as consistent with existing `memory.limit_in_bytes` for cgroups v1.** |
| memory.low | memory.low is the best-effort memory protection, a "soft guarantee" that if the cgroup and all its descendants are below this threshold, the cgroup's memory won't be reclaimed unless memory can’t be reclaimed from any unprotected cgroups. Not yet considered for now. |
| memory.high | memory.high is the memory usage throttle limit. This is the main mechanism to control a cgroup's memory use. If a cgroup's memory use goes over the high boundary specified here, the cgroup’s processes are throttled and put under heavy reclaim pressure. The default is max, meaning there is no limit. **We use a formula to calculate `memory.high` depending on `limits.memory/node allocatable memory` and a memory throttling factor.** |

This proposal sets `requests.memory` to `memory.min` for protecting container memory requests. `limits.memory` is set to `memory.max` (this is consistent with existing `memory.limit_in_bytes` for cgroups v1, we do nothing because [cgroup_v2](https://github.com/kubernetes/enhancements/tree/master/keps/sig-node/2254-cgroup-v2) has implemented for that).

We also introduce `memory.high` for container cgroup to throttle container memory overcommit allocation. 
***Note***: memory.high is set for container-level cgroup, and not for pod-level cgroup. If a container in a pod sees a spike in memory usage, it could result in total pod-level memory usage to reach memory.high level set at pod-level cgroup. This will induce throttling in other containers as the pod-level memory.high was hit. Hence to avoid containers from affecting each other, we set memory.high for only container-level cgroup.

#### Alpha v1.22
It is based on a formula:
```
memory.high=(limits.memory or node allocatable memory) * memory throttling factor, 
where default value of memory throttling factor is set to 0.8
```

e.g. If a container has `requests.memory=50, limits.memory=100`, and we have a throttling factor of .8, `memory.high` would be 80. If a container has no memory limit specified, we substitute `limits.memory` for `node allocatable memory` and apply the throttling factor of .8 to that value.
It must be ensured that `memory.high` is always greater than `memory.min`.

Node reserved resources(kube-reserved/system-reserved) are also considered. It is tied to `--enforce-node-allocatable` and `memory.min` will be set properly.

Brief map as follows:
| type | memory.min | memory.high |
| -------- | -------- | -------- | 
| container | requests.memory | limits.memory/node allocatable memory * memory throttling factor |
| pod | sum(requests.memory) | N/A |
| node | pods, kube-reserved, system-reserved | N/A |

#### Alpha v1.27
The formula for memory.high for container cgroup is modified in Alpha stage of the feature in K8s v1.27. It will be set based on formula:
```
memory.high=floor[(requests.memory + memory throttling factor * (limits.memory or node allocatable memory - requests.memory))/pageSize] * pageSize, where default value of memory throttling factor is set to 0.9
```
Note: If a container has no memory limit specified, we substitute `limits.memory` for `node allocatable memory` and apply the throttling factor of .9 to that value.


The table below runs over the examples with different values requests.memory and 1Mi pageSize:

| limits.memory (1000) | memory throttling factor (0.9)|
| ---------------------- | ----------------------------- |
| request 0    | 900  |
| request 100  | 910  |
| request 200  | 920  |
| request 300  | 930  |
| request 400  | 940  |
| request 500  | 950  |
| request 600  | 960  |
| request 700  | 970  |
| request 800  | 980  |
| request 900  | 990  |
| request 1000 | 1000 |


Node reserved resources(kube-reserved/system-reserved) are also considered. It is tied to `--enforce-node-allocatable` and `memory.min` will be set properly.

Brief map as follows:
| type | memory.min | memory.high |
| -------- | -------- | -------- | 
| container | requests.memory | floor[(requests.memory + memory throttling factor * (limits.memory or node allocatable memory - requests.memory))/pageSize] * pageSize |
| pod | sum(requests.memory) | N/A |
| node | pods, kube-reserved, system-reserved | N/A |

###### Reasons for changing the formula of memory.high calculation in Alpha v1.27

The formula for memory.high has changed in K8s v1.27 as the Alpha v1.22 implementation has following problems:
1. It fails to throttle when requested memory is closer to memory limits (or node allocatable) as it results in memory.high being less than requests.memory.

   For example, if `requests.memory = 85, limits.memory=100`, and we have a throttling factor of 0.8, then as per the Alpha implementation memory.high =  memory throttling factor * limits.memory i.e. memory.high = 80. In this case the level at which throttling is supposed to occur i.e. memory.high is less than requests.memory. Hence there won't be any throttling as the Alpha v1.22 implementation doesn't allow memory.high to be less than requested memory. 

2. It could result in early throttling putting the processes under early heavy reclaim pressure. 
  
    For example, 
    * `requests.memory` = 800Mi

      `memory throttling factor` = 0.8

      `limits.memory` = 1000Mi

      As per Alpha v1.22 implementation,

      `memory.high` = memory throttling factor * limits.memory  = 0.8 * 1000Mi = 800Mi

      This results in early throttling and puts the processes under heavy reclaim pressure at 800Mi memory usage levels. There's a significant difference of 200Mi between the memory throttling limit (800Mi) and memory usage hard limit (1000Mi). 

    * `requests.memory` = 500Mi

      `memory throttling factor` = 0.6

      `limits.memory` = 1000Mi
      
      As per Alpha v1.22 implementation,

      `memory.high` = memory throttling factor * limits.memory = 0.6 * 1000Mi = 600Mi
      
      Throttling occurs at 600Mi which is just 100Mi over the requested memory. There's a significant difference of 400Mi between the memory throttle limit (600Mi) and memory usage hard limit (1000Mi).
  

3. Default throttling factor of 0.8 may be too aggressive for some applications that are latency sensitive and always use memory close to memory limits.

   For example, there are some known Java workloads that use 85% of the memory will start to get throttled once this feature is enabled by default. Hence the default 0.8 memoryThrottlingFactor value may not be a good value for many applications due to inducing throttling too early.

<br>
Some more examples to compare memory.high using Alpha v1.22 and Alpha v1.27 are listed below:

| Limit 1000Mi <br /> Request, factor | Alpha v1.22: memory.high = memory throttling factor \* memory.limit (or node allocatable if memory.limit is not set) | Alpha v1.27: memory.high = floor[(requests.memory + memory throttling factor * (limits.memory or node allocatable memory - requests.memory))/pageSize] * pageSize assuming 1Mi pageSize
| -------------------------------- | ------------------------------------------------------- | ------------------------------------------------
| request 500Mi, factor 0.6        | 600Mi (very early throttling when memory usage is just 100Mi above requested memory; 400Mi unused) | 800Mi
| request 800Mi, factor 0.6        | no throttling (600 < 800 i.e. memory.high < memory.request => no throttling) | 920Mi
| request 1000Mi, factor 0.6       | max | max
| request 500Mi, factor 0.8        | 800Mi (early throttling at 800Mi, when 200Mi is unused) | 900Mi
| request 850Mi, factor 0.8        | no throttling (800 < 850 i.e. memory.high < memory.request => no throttling) | 970Mi
| request 500Mi, factor 0.4        | no throttling (400 < 500  i.e. memory.high < memory.request => no throttling) | 700Mi

***Note***: As seen from the examples in the table, the formula used in Alpha v1.27 implementation eliminates the cases of memory.high being less than memory.request. However, it still can result in early throttling if memory throttling factor is set low. Hence, it is recommended to set a high memory throttling factor to avoid early throttling.

###### Quality of Service for Pods

In addition to the change in formula for memory.high, we are also adding the support for memory.high to be set as per `Quality of Service(QoS) for Pod` classes. Based on user feedback in Alpha v1.22, some users would like to opt-out of MemoryQoS on a per pod basis to ensure there is no early memory throttling. By making user's pods guaranteed, they will be able to do so. Guaranteed pod, by definition, are not overcommitted, so memory.high does not provide significant value.

Following are the different cases for setting memory.high as per QOS classes:
1. Guaranteed
Guaranteed pods by their QoS definition require memory requests=memory limits and are not overcommitted. Hence MemoryQoS feature is disabled on those pods by not setting memory.high. This ensures that Guaranteed pods can fully use their memory requests up to their set limit, and not hit any throttling.

2. Burstable
Burstable pods by their QoS definition require at least one container in the Pod with CPU or memory request or limit set. 

    Case I: When requests.memory and limits.memory are set, the formula is used as-is: 
    ```
    memory.high = floor[ (requests.memory + memory throttling factor * (limits.memory - requests.memory)) / pageSize ] * pageSize
    ```

    Case II. When requests.memory is set, limits.memory is not set, we substitute limits.memory for node allocatable memory in the formula:
    ```
    memory.high = floor[ (requests.memory + memory throttling factor * (node allocatable memory - requests.memory))/ pageSize ] * pageSize
    ```

    Case III. When requests.memory is not set and  limits.memory is set, we set `requests.memory = 0` in the formula:
    ```
    memory.high = floor[ (memory throttling factor * limits.memory) / pageSize ] * pageSize
    ```

3. BestEffort 
The pod gets a BestEffort class if limits.memory and requests.memory are not set. We set `requests.memory = 0` and substitute limits.memory for node allocatable memory in the formula:
    ```
    memory.high = floor[ (memoryThrottlingFactor * node allocatable memory) / pageSize ] * pageSize
    ```

###### Alternative solutions for implementing memory.high
Alternative solutions that were discussed (but not preferred) before finalizing the implementation for memory.high are:
1. Allow customers to set memoryThrottlingFactor for each pod in annotations.

   Proposal: Add a new annotation for customers to set memoryThrottlingFactor to override kubelet level memoryThrottlingFactor.
	  * Pros
        * Allows more flexibility.
        * Can be quickly implemented.
	  * Cons
        * Customers might not need per pod memoryThrottlingFactor configuration.
        * It is too low-level detail to expose to customers.

2. Allow customers to set MemoryThrottlingFactor in pod yaml.

   Proposal: Add a new field in API for customers to set memoryThrottlingFactor to override kubelet level memoryThrottlingFactor.
    * Pros
        * Allows more flexibility.
    * Cons
        * Customers might not need per pod memoryThrottlingFactor configuration.
        * API changes take a lot of time, and we might eventually realize that the customers don’t need per pod level setting.
        * It is too low-level detail to expose to customers, and it is highly unlikely to get an API approval.

***[Preferred Alternative]***: Considering the cons of the alternatives mentioned above, adding support for memory QoS looks more preferable over other solutions for following reasons:
  * Memory QoS complies with QoS which is a wider known concept.
  * It is simple to understand as it uses kubelet-level configuration (`memoryThrottlingFactor`, `memoryReservationPolicy`) rather than per-pod settings.
  * It doesn't require Pod API changes, keeping the low-level cgroup details abstracted from workload authors.

#### Beta v1.28 - Cancelled
The feature was planned to graduate to Beta in v1.28 but was backed out due to a livelock issue:

workloads hitting `memory.high` on kernels < 5.9 would hang indefinitely instead of progressing
toward OOM. The kernel's memory reclaim rate couldn't keep up with allocation rate, causing
processes to stall at near-zero CPU with no forward progress.

**Root cause**: When memory usage hits memory.high, the kernel triggers synchronous reclaim. On kernels < 5.9, if the workload's allocation rate exceeded the reclaim rate, the process would enter a livelock where it repeatedly allocating, hitting the limit, reclaiming, allocating again and never reaching memory.max where OOM would terminate it cleanly.

**Resolution**: Kernel 5.9+ (October 2020) resolved this with [commit b3ff929](https://github.com/torvalds/linux/commit/b3ff92916af3b458712110bb83976a23471c12fa), which ensures forward progress even when allocation rate exceeds reclaim rate.

See the [Previous Status (v1.28)](#previous-status-v128) for the original investigation and
test results documenting this behavior.

#### Alpha v1.36
**Status:** The livelock issue that blocked v1.28 has been resolved—kernel 5.9+ (October 2020) contains a fix that prevents indefinite throttling at `memory.high`.

**Changes from Alpha v1.27:**
- No changes to memory.high formula
- Kubelet adds a kernel version check that requires Kernel 5.9+ to mitigate the livelock issue. If the kernel version < 5.9, the Kubelet will log a warning that may exhibit livelocks at memory.high.
- Add metrics for memory usage observability:
    - `kubelet_memory_qos_memory_min_bytes`: Gauge showing memory.min value configured per container
    - `kubelet_memory_qos_memory_high_bytes`: Gauge showing memory.high value configured per container
    - `kubelet_memory_qos_throttle_events_total`: Counter for memory.high throttle events (from memory.events)
- Add `memoryReservationPolicy` enum to KubeletConfiguration (default: `None`). When set to `HardReservation`, kubelet sets `memory.min` for containers and pods. This provides independent control over memory protection:
    - `memoryReservationPolicy: None` → disables memory.min protection
    - `memoryThrottlingFactor: 1.0` → effectively disables early throttling (memory.high = limit)
    - Operators can combine both for full opt-out while keeping the feature gate enabled 
- Comprehensive documentation of failure modes and troubleshooting
- Verified feature interactions with In-Place Pod Resize ([KEP-1287](https://github.com/kubernetes/enhancements/blob/master/keps/sig-node/1287-in-place-update-pod-resources/README.md)), DRA ([KEP-4381](https://github.com/kubernetes/enhancements/tree/master/keps/sig-node/4381-dra-structured-parameters/README.md)), and Swap ([KEP-2400](https://github.com/kubernetes/enhancements/blob/master/keps/sig-node/2400-node-swap/README.md))
Note: While PSI is valuable for monitoring, implementing userspace OOM policies (such as systemd-oomd integration for graceful eviction) is outside the scope of this KEP.

**Kernel Requirement:**

Linux kernel 5.9+ is required for correct `memory.high` behavior. See [Prerequisite](#prerequisite) for details.

| Kernel Version | Status | Notes |
|---------------|--------|-------|
| < 5.9 | Not Recommended | May exhibit livelock at memory.high |
| 5.9+ | Supported | Contains livelock fix (commit b3ff92916af3) |

**Container Runtime Requirement:**

| Runtime | Minimum Version |
|---------|-----------------|
| containerd | 1.6.0 |
| CRI-O | 1.22 |


### User Stories (Optional)
#### Memory Sensitive Workload
Some workloads are sensitive to memory allocation and availability, slight delays may cause service outage. In this case, a mechanism is needed to ensure the quality of memory.
We must provide guarantee in both of the following aspects:
- Retain memory requests to reduce allocation latency
- Protect memory requests from being reclaimed

#### Node Availability
The stability of kubelet node is very important to users. As the key resource of the node, the availability of memory is the key factor for the stability of the node. We should do something to protect node reserved memory.

### Comparison with Memory Manager
The Memory Manager is a new component of kubelet ecosystem proposed to enable single-NUMA and multi-NUMA guaranteed memory allocation at topology level. Memory QoS proposal mainly uses cgroups v2 to improve the quality of memory requests, thereby improving the memory qos of `Guaranteed` and `Burstable` pods and even entire node. 
See also https://github.com/kubernetes/enhancements/tree/master/keps/sig-node/1769-memory-manager

### memory.low vs memory.min
In cgroups v2, `memory.low` is designed for best-effort memory protection which is more like "soft guarantee" and won't be reclaimed unless memory can't be reclaimed from any unprotected cgroups. `memory.min` is a bit aggressive. It will always retain specified amount of memory and it can never be reclaimed. When requirement is not satisfied, system OOM killer will be invoked. 


### Notes/Constraints/Caveats (Optional)

<!--
What are the caveats to the proposal?
What are some important details that didn't come across above?
Go in to as much detail as necessary here.
This might be a good place to talk about core concepts and how they relate.
-->

**Container Type Handling:**

- **Init containers**: Run sequentially before app containers. Their memory requests are NOT summed with app containers for pod-level memory.min since they don't run concurrently. Each init container gets its own memory.min while running.
- **Sidecar containers (KEP-753)**: Restartable init containers that run concurrently with app containers. Their memory requests ARE included in pod-level memory.min calculation.
- **Ephemeral containers**: Debug containers with no resource requests. They do not affect memory.min. For Guaranteed pods, ephemeral containers don't change QoS class and the pod remains exempt from memory.high.

**Other Considerations:**

- **Pod overhead (RuntimeClass)**: Pod overhead is included in pod-level memory.min calculation. The `ResourceConfigForPod()` function calls `PodRequests()` which includes overhead by default. This means overhead memory receives memory.min protection at the pod cgroup level.
- **In-Place Pod Resize (KEP-1287)**: When container memory requests/limits are resized in-place, memory.min and memory.high are recalculated during the next cgroup reconciliation cycle.
- **Swap (KEP-2400)**: When swap is enabled, memory.high triggers reclaim which may push pages to swap rather than throttle allocations. This is expected behavior.
- **memory.min overcommit**: The scheduler ensures sum(pod_requests) ≤ node_allocatable before placing pods. Since memory.min = requests.memory, memory.min overcommit is prevented at scheduling time. In edge cases (e.g., node allocatable decreases after pods are scheduled), if sum of memory.min exceeds physical memory, the kernel may OOM kill to honor guarantees.
- **memoryThrottlingFactor validation**: Valid range is (0, 1.0]. Values outside this range are rejected by kubelet configuration validation. Setting to 1.0 effectively disables early throttling (memory.high = limit).
- **memoryReservationPolicy**: When set to `HardReservation`, kubelet sets memory.min for containers and pods. Default is `None`.
- **pageSize**: The formula uses the system's base page size (typically 4KiB on x86_64, configurable on ARM64). Hugepages are not used for the pageSize calculation

### Risks and Mitigations

<!--
What are the risks of this proposal, and how do we mitigate? Think broadly.
For example, consider both security and how this will impact the larger
Kubernetes ecosystem.

How will security be reviewed, and by whom?

How will UX be reviewed, and by whom?

Consider including folks who also work outside the SIG or subproject.
-->

The main risk of this proposal is throttling applications too early.
We intend to mitigate this by (1) setting a `memory.high` that is closer to the
limit and (2) only throttling when usage > request.

**Blast Radius Alpha v1.36:** in Alpha v1.36, the `MemoryQoS ` feature gate is disabled by default. There is no impact on existing clusters unless the feature is explicitly opted in. Mitigations for cluster with `MemoryQoS ` turned on:
- Operators can set `memoryReservationPolicy: Disabled` to disable memory.min protection or `memoryThrottlingFactor: 1.0` to disable early throttling
- The default throttling factor (0.9) is conservative, leaving 10% headroom before throttling
- Guaranteed pods are exempt from memory.high throttling
- Operators should test with the feature explicitly enabled before production upgrades

## Design Details
### Prerequisite
1. Kernel 5.9+ with cgroups v2 unified hierarchy (kernel 5.9 includes livelock fix) 
2. CRI runtime supports [cgroups v2 Unified Spec](https://github.com/opencontainers/runtime-spec/blob/7c549cb0939af03d5a2a8b271e2ad6871309e228/specs-go/config.go#L376) for container level
3. Kubelet enables `--enforce-node-allocatable=<pods, kube-reserved, system-reserved>` 

### Feature Gate
Set `--feature-gates=MemoryQoS=true` to enable the feature.

When enabled, the following KubeletConfiguration fields control behavior:
- `memoryThrottlingFactor` (float, range (0, 1.0], default 0.9): Controls memory.high calculation. Set to 1.0 to effectively disable early throttling.
- `memoryReservationPolicy` (enum, default `None`): Controls whether memory.min is set for containers/pods. Set to `HardReservation` to enable memory.min protection (memory.min = requests.memory).

### Mapping Rules
#### Container/Pod
![](./memory-high.png)
1. If container sets `requests.memory`, we set `memory.min=pod.spec.containers[i].resources.requests[memory]` for container level cgroup
2. If any containers in pod sets `requests.memory`, we set `memory.min=sum(pod.spec.containers[i].resources.requests[memory])` for pod level cgroup
3. If container sets `limits.memory`, we set `memory.high=floor[(requests.memory + memoryThrottlingFactor * (limits.memory - requests.memory))/pageSize] * pageSize` for container level cgroup if `memory.high>memory.min`
4. If container doesn't set `limits.memory`, we substitute `node allocatable memory` for `limits.memory` in the formula above
5. If kubelet sets `--cgroups-per-qos=true`, we set `memory.min=sum(pod[i].spec.containers[j].resources.requests[memory])` to make ancestor cgroups propagation effective
6. There are no changes regarding memory limit, that is `memory.max=memory_limits` (same as existing cgroup v2 implementation)
#### Node
1. If kubelet sets `--enforce-node-allocatable=kube-reserved`, `--kube-reserved=[a]` and `--kube-reserved-cgroup=[b]`, we set `memory.min=[a]` for node level cgroup `[b]`
2. If kubelet sets `--enforce-node-allocatable=system-reserved`, `--system-reserved=[a]` and `--system-reserved-cgroup=[b]`, we set `memory.min=[a]` for node level cgroup `[b]`
3. If kubelet sets `--enforce-node-allocatable=pods`, we set `memory.min=sum(pod[i].spec.containers[j].resources.requests[memory])` for kubepods cgroup

### Interactive
New `Unified` field will be added in both CRI and QoS Manager for cgroups v2 extra parameters. It is recommended to have the same semantics as opencontainers/runtime-spec#1040

- container level: `Unified` added in `LinuxContainerResources`
- pod/node level: `Unified` added in `cm.ResourceConfig` 

### Workflow
#### Container 
![](./container-level.png)

#### Pod 
![](./pod-level.png)

#### QoS 
![](./qos-level.png)

#### Node 
![](./node-level.png)

### Cgroup Hierarchy

**Note:** Paths shown use cgroupfs driver format. For systemd cgroup driver (common in production), paths use slice notation:
- cgroupfs: `/cgroup2/kubepods/burstable/pod<UID>/`
- systemd: `/sys/fs/cgroup/kubepods.slice/kubepods-burstable.slice/kubepods-burstable-pod<UID>.slice/`

Container/Pod (cgroupfs format):
```
// Container
/cgroup2/kubepods/pod<UID>/<container-id>/memory.min=pod.spec.containers[i].resources.requests[memory]
/cgroup2/kubepods/pod<UID>/<container-id>/memory.high=floor[(requests.memory + memoryThrottlingFactor * (limits.memory - requests.memory))/pageSize] * pageSize // Burstable
// Pod
/cgroup2/kubepods/pod<UID>/memory.min=sum(pod.spec.containers[i].resources.requests[memory])
// QoS ancestor cgroup
/cgroup2/kubepods/burstable/memory.min=sum(pod[i].spec.containers[j].resources.requests[memory])
```

Node:
```
/cgroup2/kubepods/memory.min=sum(pod[i].spec.containers[j].resources.requests[memory])
/cgroup2/<kube-reserved-cgroup,system-reserved-cgroup>/memory.min=<kube-reserved,system-reserved>
```

### Cgroup v2 Support
After Kubernetes v1.19, kubelet can identify cgroups v2 and do the conversion. Since [v1.0.0-rc93](https://github.com/opencontainers/runc/releases/tag/v1.0.0-rc93), runc supports `Unified` to pass through cgroups v2 parameters. So we use this variable to pass `memory.min` when cgroups v2 mode is detected.

### Container Runtime Interface (CRI) Changes
We need add new field `Unified` in CRI api which is basically passthrough for OCI spec Unified field and has same semantics: opencontainers/runtime-spec#1040

```
type LinuxContainerResources struct {
    ...
    Unified map[string]string `json:"unified,omitempty"`
}
```

### Test Plan

<!--
**Note:** *Not required until targeted at a release.*
The goal is to ensure that we don't accept enhancements with inadequate testing.

All code is expected to have adequate tests (eventually with coverage
expectations). Please adhere to the [Kubernetes testing guidelines][testing-guidelines]
when drafting this test plan.

[testing-guidelines]: https://git.k8s.io/community/contributors/devel/sig-testing/testing.md
-->

[X] I/we understand the owners of the involved components may require updates to
existing tests to make this code solid enough prior to committing the changes necessary
to implement this enhancement.

Overall Test plan:

For `Alpha`, unit tests were added to test functionality for container/pod/node level cgroup with containerd and CRI-O.

For second alpha iteration, (1.27), we plan to add new node e2e tests to
validate the MemoryQoS settings are applied correctly.

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

- `pkg/kubelet/cm`: `02/09/2026` - `67.2`

##### Integration tests

<!--
This question should be filled when targeting a release.
For Alpha, describe what tests will be added to ensure proper quality of the enhancement.

For Beta and GA, add links to added tests together with links to k8s-triage for those tests:
https://storage.googleapis.com/k8s-triage/index.html
-->

n/a: plan to use node e2e tests (see below)

##### e2e tests

<!--
This question should be filled when targeting a release.
For Alpha, describe what tests will be added to ensure proper quality of the enhancement.

For Beta and GA, add links to added tests together with links to k8s-triage for those tests:
https://storage.googleapis.com/k8s-triage/index.html

We expect no non-infra related flakes in the last month as a GA graduation criteria.
-->

As part of alpha, we plan to add a new node e2e test to validate that the MemoryQoS settings will be correctly set on both pods as well as node allocatable.
The test will reside in `test/e2e_node`.

### Graduation Criteria

#### Alpha Graduation (v1.36 - Alpha v3)
- [cgroup_v2](https://github.com/kubernetes/enhancements/tree/master/keps/sig-node/2254-cgroup-v2) is in `GA` (graduated in v1.25)
- Kernel 5.9+ required for correct memory.high behavior (livelock fix)
- `memoryReservationPolicy` KubeletConfiguration field for independent control over memory.min protection
- New metrics: `kubelet_memory_qos_memory_min_bytes`, `kubelet_memory_qos_memory_high_bytes`, `kubelet_memory_qos_throttle_events_total`
- Memory QoS is covered by unit and e2e-node tests
- Memory QoS supports containerd 1.6+ and CRI-O 1.22+

#### Beta Graduation
- All Alpha v3 criteria met
- Benchmark testing validates node stability under sustained memory pressure with `sum(memory.min)` near capacity
- Observability via cgroup files (memory.min, memory.high, memory.events) and existing cadvisor metrics (container_oom_events_total)
- Production feedback from Alpha v3 users confirms no regressions

#### GA Graduation
- Memory QoS has been in Beta for at least 2 releases
- Memory QoS sees adoption in production environments
- Memory QoS is covered by conformance tests

### Upgrade / Downgrade Strategy

If `MemoryQoS` enabled, needs Kubelet to verify the kernel version 5.9+ before upgrade

### Version Skew Strategy

Kubelet and Kernel Skew: This feature requires kernel 5.9+. The Kubelet will check the kernel version, if the kernel is < 5.9, log a warning that may exhibit livelocks at memory.high. Operators can set `memoryThrottlingFactor: 1.0` to disable early throttling or `memoryReservationPolicy: Disabled` to disable memory.min protection.

Kubelet and CRI skew: If the CRI does not support the Unified cgroup v2, upgrade containerd to 1.6+ or CRI-O to 1.22+.

## Production Readiness Review Questionnaire

<!--

Production readiness reviews are intended to ensure that features merging into
Kubernetes are observable, scalable and supportable; can be safely operated in
production environments, and can be disabled or rolled back in the event they
cause increased failures in production. See more in the PRR KEP at
https://git.k8s.io/enhancements/keps/sig-architecture/1194-prod-readiness.

The production readiness review questionnaire must be completed and approved
for the KEP to move to `implementable` status and be included in the release.

In some cases, the questions below should also have answers in `kep.yaml`. This
is to enable automation to verify the presence of the review, and to reduce review
burden and latency.

The KEP must have a approver from the
[`prod-readiness-approvers`](http://git.k8s.io/enhancements/OWNERS_ALIASES)
team. Please reach out on the
[#prod-readiness](https://kubernetes.slack.com/archives/CPNHUMN74) channel if
you need any help or guidance.
-->

### Feature Enablement and Rollback

<!--
This section must be completed when targeting alpha to a release.
-->

###### How can this feature be enabled / disabled in a live cluster?

<!--
Pick one of these and delete the rest.
-->

- [x] Feature gate (also fill in values in `kep.yaml`)
  - Feature gate name: MemoryQoS
  - Components depending on the feature gate: kubelet

###### Does enabling the feature change any default behavior?

<!--
Any change of default behavior may be surprising to users or break existing
automations, so be extremely careful here.
-->
Yes, the kubelet will set `memory.min` for Guaranteed and Burstable pod/container level cgroup. It also will set `memory.high` for burstable and best effort containers, which may cause memory allocation to be slowed down if the memory usage level in the containers reaches `memory.high` level. `memory.min` for qos or node level cgroup will be set when `--cgroups-per-qos` or `--enforce-node-allocatable` is satisfied.

###### Can the feature be disabled once it has been enabled (i.e. can we roll back the enablement)?

<!--
Describe the consequences on existing workloads (e.g., if this is a runtime
feature, can it break the existing applications?).

NOTE: Also set `disable-supported` to `true` or `false` in `kep.yaml`.
-->
Yes, related cgroups can be rolled back, `memory.min/memory.high` will reset to default value.

###### What happens if we reenable the feature if it was previously rolled back?
The kubelet will reconcile `memory.min/memory.high` with related cgroups.

###### Are there any tests for feature enablement/disablement?

<!--
The e2e framework does not currently support enabling or disabling feature
gates. However, unit tests in each component dealing with managing data, created
with and without the feature, are necessary. At the very least, think about
conversion tests if API types are being modified.
-->
Yes, unit tests cover both feature enabled and disabled states. When enabled, we test `memory.min/memory.high` for workloads and node cgroups are set correctly. When transitioning from enabled to disabled, we verify `memory.min/memory.high` reset to default values. Tests also cover the `memoryReservationPolicy` configuration field to verify memory.min is skipped when set to `Disabled`.

### Rollout, Upgrade and Rollback Planning

<!--
This section must be completed when targeting beta to a release.
-->
Rollout this feature requires enabling the `MemoryQoS` feature gate in kubelet. The feature also adds two KubeletConfiguration fields:
- `memoryThrottlingFactor` (float, default 0.9): Controls memory.high calculation
- `memoryReservationPolicy` (enum, default `None`): Controls whether memory.min is set

It doesn't require any special opt-in by the user in their PodSpec. The kubelet will reconcile `memory.min/memory.high` with related cgroups depending on whether the feature gate is enabled and the configuration values.

###### How can a rollout or rollback fail? Can it impact already running workloads?

<!--
Try to be as paranoid as possible - e.g., what if some components will restart
mid-rollout?

Be sure to consider highly-available clusters, where, for example,
feature flags will be enabled on some API servers and not others during the
rollout. Similarly, consider large clusters and how enablement/disablement
will rollout across nodes.
-->
When the feature gate is enabled and kubelet restarts, the kubelet reconciles cgroup settings for all pods. This means:
- Existing pods will have `memory.min/memory.high` set during the next cgroup reconciliation cycle
- Node-level `memory.min` will be set immediately on kubelet startup
- Impact is gradual as pods are reconciled, not instantaneous

If the feature gate is disabled after being enabled, kubelet will reset `memory.min` to 0 and `memory.high` to max during reconciliation.

###### What specific metrics should inform a rollback?

<!--
What signals should users be paying attention to when the feature is young
that might indicate a serious problem?
-->

- Increased OOM kills on nodes where feature is enabled (`container_oom_events_total` via cadvisor)
- `container_memory_working_set_bytes` from cadvisor shows memory approaching limits
-  memory.events high counter (from cgroup files) shows throttle events
- PSI memory.pressure (if kernel supports it) shows stall time
- Application-level P99 latency (if instrumented) correlated with throttling 
- Containers stuck with near-zero CPU usage despite "Running" status (symptom of livelock on kernel < 5.9)

###### Were upgrade and rollback tested? Was the upgrade->downgrade->upgrade path tested?

<!--
Describe manual testing that was done and the outcomes.
Longer term, we may want to require automated upgrade/rollback tests, but we
are missing a bunch of machinery and tooling and can't do that now.
-->
Yes. Manual testing was performed:
- Upgrade: Enabling MemoryQoS on a running kubelet correctly sets memory.min/memory.high on new pods and updates node-level cgroups
- Rollback: Disabling MemoryQoS resets memory.min to 0 and memory.high to max for all managed cgroups
- Upgrade->downgrade->upgrade: Cgroup values are correctly reconciled in each state; no orphaned settings observed

###### Is the rollout accompanied by any deprecations and/or removals of features, APIs, fields of API types, flags, etc.?

<!--
Even if applying deprecation policies, they may still surprise some users.
-->
No

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

An operator could run ls `/sys/fs/cgroup/kubepods.slice/kubepods-burstable.slice/kubepods-burstable-pod<SOME_ID>.slice` on a node with cgroupv2 enabled to confirm the presence of `memory.min` file which tells us that the feature is in use by the workloads.

###### How can someone using this feature know that it is working for their instance?

<!--
For instance, if this is a pod-related feature, it should be possible to determine if the feature is functioning properly
for each individual pod.
Pick one more of these and delete the rest.
Please describe all items visible to end users below with sufficient detail so that they can verify correct enablement
and operation of this feature.
Recall that end users cannot usually observe component logs or access metrics.
-->

- [] Events
  - Event Reason: 
- [ ] API .status
  - Condition name: 
  - Other field: 
- [X] Other (treat as last resort)
  - Details: Operators can verify Memory QoS is working by inspecting cgroup v2 files
  in the container's cgroup hierarchy. Check `memory.min` and `memory.high` values
  are set according to the pod's requests and limits. The `memory.events` file shows
  breach counters for `high` (throttling events) and `low`/`min` protection events.

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
- Pod startup latency SLO should not be affected (cgroup setup adds negligible overhead)
- Application throughput may decrease for memory-intensive Burstable/BestEffort workloads due to memory.high throttling—this is by design to prevent OOM
- Node stability should improve as memory.min protects guaranteed memory from reclaim

###### What are the SLIs (Service Level Indicators) an operator can use to determine the health of the service?

<!--
Pick one more of these and delete the rest.
-->

- [X] Metrics
  - Metric name: `container_oom_events_total` (existing cadvisor metric)
    - Aggregation method: rate() to detect OOM kill spikes
    - Components exposing the metric: kubelet (via cadvisor)
  - Metric name: `container_memory_working_set_bytes` (existing cadvisor metric)
    - Aggregation method: compare against memory.high threshold to detect throttling
    - Components exposing the metric: kubelet (via cadvisor)
- [X] Other
  - Details: Monitor cgroup files directly for throttling events:
    - `memory.events` file shows `high` counter (throttle events)
    - Compare `memory.current` vs `memory.high` to detect near-throttle conditions
    - PSI provides observability to determine if the workload is undergoing significant memory throttling:

    ```bash
    # Check container memory pressure
    cat /sys/fs/cgroup/.../memory.pressure
    # full avg10 > 5 indicates significant throttling
    ```

###### Are there any missing metrics that would be useful to have to improve observability of this feature?

<!--
Describe the metrics themselves and the reasons why they weren't added (e.g., cost,
implementation difficulties, etc.).
-->
No.

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
- Linux kernel 5.9+
  - Usage description: Required for correct memory.high behavior (contains livelock fix)
  - Impact of its outage on the feature: N/A - kernel is always available
  - Impact of its degraded performance or high-error rates on the feature: Kernels 4.5-5.8 have cgroups v2 but lack the livelock fix, which can cause workloads to hang indefinitely when hitting memory.high

- Container runtime with cgroup v2 support
  - Usage description: Runtime must support setting memory.min and memory.high cgroup v2 parameters
  - Impact of its outage on the feature: Feature cannot be used without cgroup v2 runtime support
  - Impact of its degraded performance or high-error rates on the feature: N/A - runtime either supports cgroup v2 or it doesn't


### Scalability

<!--
For alpha, this section is encouraged: reviewers should consider these questions
and attempt to answer them.

For beta, this section is required: reviewers must answer these questions.

For GA, this section is required: approvers should be able to confirm the
previous answers based on experience in the field.
-->

###### Will enabling / using this feature result in any new API calls?

No new API calls will be generated.

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

No.

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

No.

###### Will enabling / using this feature result in increasing time taken by any operations covered by existing SLIs/SLOs?

<!--
Look at the [existing SLIs/SLOs].

Think about adding additional work or introducing new steps in between
(e.g. need to do X to start a container), etc. Please describe the details.

[existing SLIs/SLOs]: https://git.k8s.io/community/sig-scalability/slos/slos.md#kubernetes-slisslos
-->

No.

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

No, resources like PIDs, sockets, inodes will not be affected. However,
additional memory throttling can be experienced which is intended by this
feature.

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

This feature operates entirely at the kubelet level:
- Existing cgroup settings persist
- Running pods continue with their memory.min/memory.high values
- Memory protection is maintained for existing pods
- New pods cannot be scheduled (standard Kubernetes behavior when API server unavailable)


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

- **Livelock at memory.high (kernel < 5.9)**
  - Detection: Container CPU usage near zero despite "Running" status; `cat /sys/fs/cgroup/.../memory.pressure` shows `full avg10 > 10` sustained
  - Mitigations: Upgrade to kernel 5.9+; disable MemoryQoS feature gate; increase container memory limits
  - Diagnostics: Check kernel version with `uname -r`; verify < 5.9; check `memory.events` high counter incrementing rapidly
  - Testing: E2E tests require kernel 5.9+ environments

- **memory.min protection ineffective**
  - Detection: Pod memory drops below requests.memory under pressure
  - Mitigations: Verify parent cgroup has memory.min set: `cat /sys/fs/cgroup/kubepods.slice/memory.min`
  - Diagnostics: Walk cgroup hierarchy checking memory.min at each level; verify kubelet logs for QoS manager errors
  - Testing: Unit tests verify parent cgroup configuration

- **Cgroups v2 not available**
  - Detection: Feature silently disabled; `memory.min`/`memory.high` files don't exist
  - Mitigations: Boot with `systemd.unified_cgroup_hierarchy=1`
  - Diagnostics: `stat /sys/fs/cgroup/cgroup.controllers` fails; `mount | grep cgroup` shows cgroup v1
  - Testing: Feature detection skips MemoryQoS on cgroups v1 systems

- **Runtime doesn't support unified map**
  - Detection: memory.min/memory.high not set despite feature enabled
  - Mitigations: Upgrade containerd to 1.6+ or CRI-O to 1.22+
  - Diagnostics: Check runtime version; compare CRI request (kubelet logs -v=6) with actual cgroup values

- **Cgroups v2 available but kernel < 5.9**
  - Detection: Workloads hitting memory.high may exhibit livelock (high CPU, no progress, near-zero memory allocation rate)
  - Mitigations: Upgrade kernel to 5.9+ or disable feature gate
  - Diagnostics: Check kernel version with `uname -r`; kernels 4.5-5.8 have cgroups v2 but lack the livelock fix
  - Note: Kubelet does not currently validate kernel version at startup. This is documented behavior—operators should verify kernel 5.9+ before enabling the feature. A startup warning for older kernels is planned for GA
 
 
###### What steps should be taken if SLOs are not being met to determine the problem?

1. Verify feature enablement: `ps aux | grep kubelet | grep MemoryQoS`
2. Check cgroups v2: `cat /sys/fs/cgroup/cgroup.controllers`
3. Check kernel version: `uname -r` (should be 5.9+)
4. Verify cgroup values are set:
   ```bash
   POD_UID=$(kubectl get pod <name> -o jsonpath='{.metadata.uid}' | tr '-' '_')
   cat /sys/fs/cgroup/kubepods.slice/kubepods-burstable.slice/kubepods-burstable-pod${POD_UID}.slice/*/memory.min
   ```
5. Check for throttling: `cat /sys/fs/cgroup/.../memory.events | grep high`
6. If issues persist, set `memoryReservationPolicy: Disabled` (disable memory.min) or `memoryThrottlingFactor: 1.0` (disable early throttling) in KubeletConfiguration and restart kubelet

## Implementation History
- 2020/03/14: initial proposal
- 2020/05/05: target Alpha to v1.22
- 2023/03/03: target Alpha v2 to v1.27
- 2023/06/14: target Beta to v1.28
- 2026/02/09: Alpha v3 targeted for v1.36

## Drawbacks

<!--
Why should this KEP _not_ be implemented?
-->

The main drawbacks are concerns about unintended memory throttling and
additional complexity due to utilization of several new cgroup v2
based memory controls (i.e., memory.min, memory.high).

However, we believe that impact of unintended throttling will be minimized due
to a high throttling factor (see above) and the additional complexity is
justified due to the additional resource management benefits

## Alternatives

<!--
What other approaches did you consider, and why did you rule them out? These do
not need to be as detailed as the proposal, but should include enough
information to express the idea and why it was not acceptable.
-->

Please refer to alternatives mentioned above in the [proposal](#proposal) section,
which discusses the alternatives and changes from the original alpha design to
the newly updated alpha design.

## Infrastructure Needed (Optional)

<!--
Use this section if you need things from the project/SIG. Examples include a
new subproject, repos requested, or GitHub details. Listing these here allows a
SIG to get the process for these resources started right away.
-->

N/A, no new infrastructure is needed, this KEP aims to reuse the existing node e2e jobs and framework.