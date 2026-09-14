# KEP-3953: In-place Node Resource Resize

<!--
A table of contents is helpful for quickly jumping to sections of a KEP and for
highlighting any additional information provided beyond the standard KEP
template.

Ensure the TOC is wrapped with
  <code>&lt;!-- toc --&rt;&lt;!-- /toc --&rt;</code>
tags, and then generate with `hack/update-toc.sh`.
-->

<!-- toc -->
- [Release Signoff Checklist](#release-signoff-checklist)
- [Glossary](#glossary)
- [Summary](#summary)
- [Motivation](#motivation)
  - [Goals](#goals)
  - [Non-Goals](#non-goals)
- [Proposal](#proposal)
  - [User Stories](#user-stories)
    - [Story 1: Maximizing Specialized Hardware](#story-1-maximizing-specialized-hardware)
    - [Story 2: Vertical Scaling for Performance](#story-2-vertical-scaling-for-performance)
    - [Story 3: Reducing Operational Complexity (Scale-Up vs. Scale-Out)](#story-3-reducing-operational-complexity-scale-up-vs-scale-out)
    - [Story 4: Instant Capacity Utilization](#story-4-instant-capacity-utilization)
    - [Story 5: Zero-Disruption Operations](#story-5-zero-disruption-operations)
    - [Story 6: Emergency Hardware Removal (Self-Protection)](#story-6-emergency-hardware-removal-self-protection)
    - [Story 7: Dynamic Storage Expansion](#story-7-dynamic-storage-expansion)
    - [Story 8: Orchestrated Capacity Downscale](#story-8-orchestrated-capacity-downscale)
  - [Notes/Constraints/Caveats (Optional)](#notesconstraintscaveats-optional)
  - [Risks and Mitigations](#risks-and-mitigations)
    - [OOMScoreAdjust Drift for Existing Pods](#oomscoreadjust-drift-for-existing-pods)
    - [Container Swap Limit Re-calculation Overhead](#container-swap-limit-re-calculation-overhead)
    - [API Status Clamping](#api-status-clamping)
    - [Application-Level Hardware Assumptions](#application-level-hardware-assumptions)
    - [Coordination with External NRI/Runtime Plugins](#coordination-with-external-nriruntime-plugins)
- [Design Details](#design-details)
  - [API Changes](#api-changes)
  - [Node Conditions for State Dissemination](#node-conditions-for-state-dissemination)
  - [Admission Control Contract](#admission-control-contract)
  - [Resource-Specific Validation Rules](#resource-specific-validation-rules)
  - [Security Considerations](#security-considerations)
  - [Architecture Flow](#architecture-flow)
    - [Path A1: Orchestrated Configuration-Driven Upscale](#path-a1-orchestrated-configuration-driven-upscale)
    - [Path A2: Orchestrated Configuration-Driven Downscale](#path-a2-orchestrated-configuration-driven-downscale)
    - [Path B: Emergency Hardware-Driven Fallback (Ungraceful Downscale)](#path-b-emergency-hardware-driven-fallback-ungraceful-downscale)
    - [Flow Control: Container Swap Limit Recalculation](#flow-control-container-swap-limit-recalculation)
    - [Flow Control: Hardware Degradation and Capacity Starvation](#flow-control-hardware-degradation-and-capacity-starvation)
    - [Compatibility with Cluster Autoscaler](#compatibility-with-cluster-autoscaler)
  - [Step 1: Baseline Assumptions and Pre-requisite Validation](#step-1-baseline-assumptions-and-pre-requisite-validation)
    - [Behaviors We Are Breaking](#behaviors-we-are-breaking)
    - [Step 1 Pre-requisite Tests to be Added](#step-1-pre-requisite-tests-to-be-added)
    - [Proposed Core Code Changes](#proposed-core-code-changes)
  - [Observability and Metrics](#observability-and-metrics)
  - [Test Plan](#test-plan)
      - [Step 1: Baseline API &amp; Scheduler Validation (Pre-requisite Tests)](#step-1-baseline-api--scheduler-validation-pre-requisite-tests)
      - [Unit tests](#unit-tests)
      - [e2e tests](#e2e-tests)
  - [Graduation Criteria](#graduation-criteria)
    - [Phase 1: Alpha (target 1.37)](#phase-1-alpha-target-137)
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
- [Infrastructure Needed (Optional)](#infrastructure-needed-optional)
- [Future Work](#future-work)
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
    - [ ] (R) [all GA Endpoints](https://github.com/kubernetes/community/pull/1806) must be hit by [Conformance Tests](https://github.com/kubernetes/community/blob/master/contributors/devel/sig-architecture/conformance-tests.md)
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

## Glossary

* **In-Place Resource Resize:** Dynamically increasing or decreasing compute resources (CPU, Memory, Swap Capacity, and HugePages) on a node without requiring a node reboot or kubelet restart.
* **Node Compute Resource:** CPU, Memory, Swap Capacity, and HugePages.
* **Physical Capacity:** The raw hardware capacity of a node as reported by the integrated `cAdvisor` subsystem, reflecting the true underlying machine resources (e.g., number of CPU cores).
* **Configured Capacity:** The desired logical capacity declared by an administrator or external controller via `Node.Spec.ConfiguredCapacity`. This is the Kubelet's target and may be less than the Physical Capacity (e.g., to under-report resources intentionally). For Alpha, Configured Capacity must not exceed Physical Capacity.
* **CapacityConfigured Condition:** A `Node.Status.Condition` of type `CapacityConfigured` that the Kubelet uses to expose the current reconciliation state of a capacity resize request. Possible reasons are `Accepted`, `InProgress`, `Infeasible`, and `EmergencyReduced`.

## Summary

This proposal facilitates dynamic native resource resizing (increases and decreases in capacity) on a node to streamline cluster capacity updates, offering a seamless alternative to adding or removing nodes from an existing cluster. The revised node configurations automatically propagate at both the node and cluster levels.

This KEP introduces a new declarative API field, `Node.Spec.ConfiguredCapacity`, allowing external controllers or administrators to declare the node's desired **logical** capacity. The Kubelet is the **sole owner** of `Node.Status.Capacity` and `Node.Status.Allocatable`; external actors write only to `Node.Spec.ConfiguredCapacity`. Driven by this API-first model and validated against physical hardware metrics via cAdvisor, the Kubelet will seamlessly update its internal sub-managers, top-level kubepods cgroups, eviction thresholds, container swap limits, and the Node API object's Capacity and Allocatable fields — all without requiring a Kubelet restart.

The trigger for a capacity change is intentionally decoupled from the physical hardware layer. Both **hardware-driven** events (e.g., a hypervisor hot-plugging additional RAM, detected by cAdvisor) and **configuration-driven** events (e.g., an administrator explicitly setting `ConfiguredCapacity` to 20Gi on a 32Gi machine) are first-class triggers. The Kubelet's reconciliation loop treats both identically: compare `Node.Spec.ConfiguredCapacity` against the physical upper bound from cAdvisor, validate, then actuate.

## Motivation

Currently, a node's resource configurations are recorded solely during the Kubelet bootstrap phase and subsequently cached, operating under the assumption that the node's native capacity remains immutable throughout its lifecycle. In a conventional Kubernetes environment, cluster resources frequently require modifications over time due to inaccurate initial provisioning or escalating/subsiding workloads, traditionally forcing operators to add or remove entire nodes from the cluster.

Contemporarily, modern hypervisors, cloud providers, and kernel capabilities enable the dynamic hot-plugging and hot-unplugging of native resources such as CPU, Memory (e.g., [CPU Hotplug](https://docs.kernel.org/core-api/cpu_hotplug.html), [Memory Hotplug](https://docs.kernel.org/core-api/memory-hotplug.html)), and Ephemeral Storage block devices.

Because Kubernetes is currently unaware of these altered capacities during a live-resize, it retains outdated information, leading to two severe failure modes:

- Cluster-Level Starvation (Upscaling): If capacity is added, the Kubernetes Scheduler and Cluster Autoscaler remain blind to it, rendering the new expensive hardware useless for pending workloads.

- Node-Level Instability (Downscaling): If capacity is removed, the Kubelet's stale top-level cgroups, container swap limits, and Eviction Manager thresholds do not adjust. Because the Kubelet assumes the resources still exist, it fails to evict pods defensively, causing the host's Linux kernel to invoke the OOM killer or exhaust the disk, violently terminating processes and potentially crashing the node.

With the current state of implementation in the Kubernetes realm, the only available workaround to synchronize these capacity changes is to restart the node or at least the Kubelet. This is highly problematic because coupling a Kubelet restart to an infrastructure resize operation (like a Cloud SDK call) is rarely seamless and lacks established best practices.

Furthermore, relying on a Kubelet restart or node reboot carries significant drawbacks:

- Introducing downtime for existing or to-be-scheduled workloads until the node is fully available again.

- For bare-metal clusters, rebooting involves a significant amount of time before nodes return to a Ready state.

- The necessity to reconfigure underlying services post-reboot.

- Triggering a myriad of known edge-case nuances and bugs associated with Kubelet restarts, such as:

    - https://github.com/kubernetes/kubernetes/issues/109595

    - https://github.com/kubernetes/kubernetes/issues/119645

    - https://github.com/kubernetes/kubernetes/issues/125579

    - https://github.com/kubernetes/kubernetes/issues/127793

Therefore, it is necessary to handle capacity updates gracefully across the cluster organically, rather than resetting core cluster components to achieve the same outcome. Enabling the Kubelet to dynamically detect and adapt to underlying capacity changes mitigates manual administrative toil and unlocks several distinct advantages:

- Control Plane Efficiency: Managing resource demands by scaling existing nodes in-place brings significantly less overhead to the control plane compared to provisioning and joining entirely new nodes.

- Speed to Delivery: Expanding the capabilities of current nodes is considerably more time-efficient than the procedure of establishing new virtual machines.

- Network Optimization: Improved inter-pod network latencies, as inter-node traffic is reduced when more pods can be hosted locally on a single scaled-up node.

- Stability: Avoids the aforementioned historical bugs and disruption risks associated with forced Kubelet restarts.

**A declarative API-driven model is essential for safe capacity downscaling.** Without an API, cluster administrators have no safe way to coordinate a hot-unplug operation. The desired workflow is:

1. An external controller invokes the API to reduce logical node capacity.
2. The Kubelet actuates that reduction (graceful eviction, updates cgroups).
3. The external controller observes the Kubelet's completion signal (`CapacityConfigured: Accepted`).
4. The external controller proceeds to physically remove hardware via the hypervisor.

This coordination is impossible with a purely reactive, hardware-first model. If the hypervisor forcefully reclaims RAM (e.g., balloon deflation) without prior API coordination, the kernel OOM killer may fire before the Kubelet can react. By making the API the primary trigger, operators gain deterministic control over the timing and safety of capacity removal.

Implementing this KEP will empower nodes to recognize and adapt to changes in their native configurations instantly, facilitating the safe, efficient, and uninterrupted deployment of workloads.

### Goals

* API Synchronization: Update Node API Capacity and Allocatable fields dynamically without requiring a Kubelet restart or node drain.

* Component Sync: Re-initialize internal Kubelet managers (CPU, Memory, Eviction) to safely align with the altered hardware capacity.

* Cgroup Enforcement: Update the host's top-level /kubepods and QoS cgroup boundaries to physically enforce the resized limits.

* Container Swap: Recalculate and update swap memory limits for actively running containers via the CRI.

* Configured Capacity: Allow the logical capacity of a node to be dynamically configured via `Node.Spec.ConfiguredCapacity`, decoupling the cluster's view of the node from strict physical hardware events. Both hardware-triggered and purely configuration-driven changes (e.g., under-reporting a 32Gi machine as 20Gi) are in-scope.

* Bootstrap Parity: Upon Kubelet restart, the Kubelet reads `Node.Spec.ConfiguredCapacity` as its primary target before falling back to raw cAdvisor hardware discovery. This ensures that a Kubelet restart on a node with an existing `ConfiguredCapacity` spec behaves identically to a live-resize event.

### Non-Goals

* Reserved Adjustments: Dynamically changing --system-reserved and --kube-reserved values (these remain static from bootstrap).

* Infrastructure Orchestration: Executing the physical hardware hot-plug or updating the autoscaler to trigger it.

* Workload Re-balancing: Automatically migrating or re-balancing existing workloads across the cluster to utilize the new space.

* NRI Plugins: Propagating host resource changes to external Node Resource Interface (NRI) plugins.

* OOM Score Updates: Dynamically rewriting oom_score_adj for running processes, due to severe latency and race condition risks.

* Pod Resizing: Dynamically resizing individual Pod resource requests and limits (covered independently by KEP-1287).

* Node Capacity Overcommit: Configuring the Kubelet to report a logical capacity to the API Server that exceeds the raw, physical underlying hardware capacity (e.g., reporting 48Gi on a 32Gi machine relying on swap). For the Alpha phase, `ConfiguredCapacity` is strictly bounded by physical reality (CPU and Memory). This will be explored in Future Work.

* Admission Webhook on Node.Spec: This KEP does not introduce a new admission webhook specifically for capacity changes. Standard Kubernetes `ValidatingWebhookConfiguration` and `MutatingWebhookConfiguration` can be deployed by cluster administrators to intercept mutations to `Node.Spec.ConfiguredCapacity` without any KEP-specific mechanism.

## Proposal

This KEP introduces a declarative, event-driven reconciliation architecture to handle native resource reconfiguration safely. To align with Kubernetes core principles, the reconciliation is decoupled into three distinct phases:

1. Baseline API Validation: Establishing formal support and upstream testing for mutating a Node object's capacity dynamically.

2. Declarative Reconciliation: Updating the Kubelet to gracefully handle discrepancies between `Node.Spec.ConfiguredCapacity` (the desired logical capacity declared via the API) and the physical hardware bounds reported by cAdvisor. `Node.Status.Capacity` is the **output** of this reconciliation — it is written by the Kubelet after validation, not used as an input.

3. Metrics-Based Trigger: Implementing an automated cAdvisor hardware-drift trigger that fires the reconciliation loop when physical capacity changes. The loop re-validates `Node.Spec.ConfiguredCapacity` against the new physical bounds and, only upon successful validation, writes the resolved capacity to `Node.Status`.

By reacting to the difference between what the API states and what the Kubelet knows, the system natively supports webhook admission control, external declarative triggers, and graceful Kubelet restarts across resized hardware.

### User Stories

#### Story 1: Maximizing Specialized Hardware

As a Cluster Administrator, I want to seamlessly add CPU and memory to an existing node equipped with specialized, scarce hardware (e.g., custom ASICs, specific CPU architectures), so that I can maximize the utilization of that specific hardware without draining the node or disrupting the workloads already utilizing it.

#### Story 2: Vertical Scaling for Performance

As a Performance Engineer, I want to dynamically increase the compute capacity of a node running a monolithic database or data-heavy application, so that the application can immediately benefit from larger memory caches and reduced context-switching without suffering the downtime of a pod migration.

#### Story 3: Reducing Operational Complexity (Scale-Up vs. Scale-Out)

As a Site Reliability Engineer (SRE), I want the option to vertically scale existing nodes instead of always horizontally provisioning new VMs, so that I can manage fewer, larger nodes to simplify network topology, monitoring overhead, and overall cluster complexity.

#### Story 4: Instant Capacity Utilization

As a Cluster Administrator, I want the Kubernetes control plane to instantly recognize when a node's capacity is expanded on-the-fly via my cloud provider, so that pending workloads can be scheduled onto that new space immediately without the latency of waiting for a new VM to boot and join the cluster.

#### Story 5: Zero-Disruption Operations

As an Application Owner, I expect my running workloads to experience zero downtime or disruption when the infrastructure administrator adds capacity to the underlying node, entirely avoiding the historical risks and bugs associated with forced Kubelet restarts or node reboots.

#### Story 6: Emergency Hardware Removal (Self-Protection)

As a Node Operator, when my hypervisor unexpectedly reclaims physical memory from a running node (e.g., due to a balloon driver deflation or host pressure), I want the Kubelet to automatically detect the reduced capacity and immediately shrink its cgroup boundaries and eviction thresholds — protecting the node from kernel OOM panics — without requiring any manual intervention or Kubelet restart.

#### Story 7: Dynamic Storage Expansion

As a Storage Administrator, I want to dynamically expand the root block volume of a worker node on the fly, so that the Kubelet instantly recognizes the increased Ephemeral Storage capacity and allows pods to utilize the new space without triggering false disk-pressure evictions.

> **Note:** Ephemeral storage resize follows the same declarative API model as CPU and Memory. The Kubelet's capacity reconciliation loop updates `Node.Status.Capacity[ephemeral-storage]` and the corresponding eviction thresholds when `Node.Spec.ConfiguredCapacity` includes an updated ephemeral storage value. Physical block-device expansion (e.g., resizing the underlying volume via a cloud provider) remains an external operation outside the scope of this KEP.

#### Story 8: Orchestrated Capacity Downscale

As a Cluster Administrator, I want to dynamically reclaim (hot-unplug) underutilized memory or CPU from a node without restarting the Kubelet. I want to orchestrate this via the Kubernetes API first, so workloads are gracefully evicted and the scheduler stops sending pods before I physically remove the hardware, avoiding node crashes and workload scheduling races.

### Notes/Constraints/Caveats (Optional)

* **Linux and cgroup v2:** This feature targets Linux nodes running cgroup v2. On nodes still using cgroup v1, CPU and Memory resize are supported; however, the container swap limit recalculation (which relies on the cgroup v2 `memory.swap.max` interface) is silently skipped. No errors are emitted and node stability is preserved — swap-enabled resize simply has no effect on cgroup v1 nodes.

* **Linux Only:** This feature has no effect on Windows nodes. The Kubelet's capacity reconciliation loop short-circuits immediately on non-Linux platforms.

* **NUMA Topology Lazy Reconciliation:** When a resize changes the available memory or CPU cores per NUMA zone, the Topology Manager's view of NUMA boundaries is updated in its internal state machine. However, **running pods retain their original NUMA pinning** — they are not remapped mid-flight. Only newly admitted pods use the updated NUMA layout. Operators should account for this when sizing a downscale target on NUMA-pinned workloads.

### Risks and Mitigations

1. #### OOMScoreAdjust Drift for Existing Pods

    **Risk**: The Kubelet calculates a container's `oom_score_adj` upon creation using the formula: `1000 - (1000 * containerMemoryRequest) / nodeMemoryCapacity`. 
If a node's memory capacity changes, the OOM scores of existing Burstable pods will mathematically drift compared to newly scheduled Burstable pods, potentially skewing the Linux OOM killer's tie-breaker logic.

    **Mitigation**: We explicitly accept this minor drift. Updating `oom_score_adj` for running containers requires identifying and rewriting the `/proc/<PID>/oom_score_adj` file for every single running thread inside the container. 
This introduces severe latency, high CPU overhead, and dangerous race conditions. The overarching QoS hierarchy (Guaranteed pods remain invincible, BestEffort pods remain first-to-die) is strictly preserved, making the risk of a slightly skewed Burstable tie-breaker acceptable compared to the danger of rewriting thousands of running PIDs.

2. #### Container Swap Limit Re-calculation Overhead

   **Risk**: The proportional swap limit for a container relies on the node's total memory capacity. Upon a resize, ignoring this math leads to stranded swap space (during upscaling) or immediate host kernel panics (during downscaling). However, recalculating and applying this to all active pods introduces overhead to the Container Runtime Interface (CRI).

   **Mitigation**: The Kubelet will leverage the existing, generic `UpdateContainerResources` CRI RPC to push these changes. The Kubelet safely iterates over the active pod cache in memory, recalculates the swap boundary, and issues the update solely for containers currently in a `Running` state. Furthermore, if the node operates with Swap disabled, this entire loop short-circuits instantly, resulting in zero CRI overhead. CRI calls within the loop are best-effort and serialised per-pod: a failure on an individual container is logged and increments `kubelet_node_resize_errors_total{subsystem="container_swap_resize"}`, but does not abort the loop — the remaining containers are still updated. This prevents a single unhealthy container from blocking all swap recalculations across the node.

3. #### Kubelet Sub-Manager Synchronization Failure

   **Risk**: During an upscale, if the internal Kubelet sub-managers (CPU Manager, Memory Manager) fail to synchronize the new capacity, the Kubelet might reject new pod allocations, leading to underutilized hardware and scheduling deadlocks.

   **Mitigation**: The reconciliation loop is self-healing and naturally guarded against Denial of Service (DoS) tight-looping. If a sub-manager fails to sync, the Kubelet aborts the cache update, emits an error, and increments the `kubelet_node_resize_errors_total` metric. Because the internal capacity cache was not updated, the system will naturally re-attempt the reconciliation on the next `cAdvisor` polling cycle (defaulting to every 5 minutes) when the hardware drift is detected again. This strict 5-minute interval acts as a natural rate-limit, protecting the Kubelet's CPU.

4. #### API Status Clamping

    **Risk**: The Kubelet's node status updater relies on a boot-time MachineInfo cache. If the ContainerManager updates its internal Allocatable limits but fails to update this global cache, the status updater will aggressively clamp the new Allocatable value down to the stale boot-time Capacity, hiding the new hardware from the control plane forever.
    
    **Mitigation**: The event-driven signal explicitly forces a refresh of kl.setCachedMachineInfo() before calling syncNodeStatus(). This ensures the status updater evaluates the new limits against the live physical reality, bypassing the clamping safeguard safely.

5. #### Application-Level Hardware Assumptions

    **Risk**: Workloads often read /proc/cpuinfo or /proc/meminfo exactly once during their startup routine. If the node is vertically scaled, applications that spawn fixed per-CPU thread pools or rely on strict NUMA boundary alignments will not organically scale to use the new resources.
    
    **Mitigation**: This is an accepted limitation and is treated as an application-level responsibility. Applications must be written to dynamically poll their limits (e.g., listening to cgroup file changes) or be manually restarted by their controlling Deployment to read the new hardware layout. The Kubelet's responsibility is solely to make the hardware available at the cgroup boundary.

6. #### Coordination with External NRI/Runtime Plugins

   **Risk**: External Node Resource Interface (NRI) plugins or custom runtime wrappers may cache node capacity independently of the Kubelet, leading to split-brain resource tracking after a resize event.

   **Mitigation**: No new CRI or NRI notification call is introduced by this KEP. The runtime implicitly learns of the new capacity boundaries when the Kubelet pushes updated cgroup limits for each running container via the existing `UpdateContainerResources` CRI RPC — the same mechanism used by In-Place Pod Resource Resize (KEP-1287). This means the container runtime and any NRI plugins that subscribe to cgroup changes will see the updated limits without a dedicated capacity-change event. Direct notification of NRI plugins via a new API is explicitly deferred to Future Work (see NRI Integration).

7. #### Capacity Detection Latency (Downscale Risk)

   **Risk**: `cAdvisor` caches and refreshes `MachineInfo` every 5 minutes by default. During a memory hot-unplug (downscale) event, there is up to a 5-minute latency window where physical memory is removed, but the Kubelet remains unaware. If workloads spike during this window, the Linux kernel OOM killer may violently terminate processes before the Kubelet's capacity reconciliation loop detects the hardware drop and triggers a graceful `NodeCapacityExceeded` eviction.

   **Mitigation**: For the Alpha phase, this detection latency is an accepted operational constraint, with the kernel OOM killer serving as the ultimate safety net to protect the node. For future phases (Beta/GA), the detection mechanism will transition to an event-driven architecture to eliminate this polling lag (see Future Work).

8. #### Hardware Metric Jitter and Malformed Signals

   **Risk**: The Kubelet relies on `cAdvisor` to read raw hardware metrics from the host operating system. The OS can occasionally report minor memory capacity jitter (micro-fluctuations in bytes) due to internal kernel allocations, or bugs could result in malformed data (e.g., negative capacities, or `0`). Blindly reacting to these would cause an endless loop of API patches, CRI overhead, and potential mass-evictions.

   **Mitigation**: Before accepting a capacity change, the ContainerManager implements a strict Validation and Jitter Tolerance Filter. Micro-drifts (e.g., changes under 100Mi) are silently dropped. Nonsense values (e.g., 0, negative values, or values that fall below the static Kubelet reservations) are explicitly rejected, an error is logged, and the reconciliation pipeline short-circuits.

## Design Details

### API Changes

To support dynamic configuration of node capacity without requiring a Kubelet restart, a new declarative structure is introduced to the NodeSpec API. This field allows external controllers or administrators to declare the desired logical capacity target for the node.

```go
// 1. New field added to NodeSpec
type NodeSpec struct {
    // ... existing fields ...

    // ConfiguredCapacity defines the desired logical capacity of the node.
    // On startup, the Kubelet checks this field first; if set, it is treated as
    // the desired target and validated against the physical upper bound reported
    // by cAdvisor. If unset, the Kubelet uses self-discovered cAdvisor capacity
    // as both the physical upper bound and the initial target.
    // For Alpha, ConfiguredCapacity must not exceed physical capacity (CPU/Memory).
    // +optional
    ConfiguredCapacity ResourceList `json:"configuredCapacity,omitempty"`
}

// 2. New Condition Type constant
const (
    // ... existing conditions (e.g., NodeReady, NodeMemoryPressure) ...

    // NodeCapacityConfigured indicates the status of the Kubelet's reconciliation 
    // of the desired Node.Spec.ConfiguredCapacity against the physical hardware.
    NodeCapacityConfigured NodeConditionType = "CapacityConfigured"
)
```

### Node Conditions for State Dissemination

To prevent Kubelet reconciliation loops and provide standard Kubernetes observability, the Kubelet exposes its validation and actuation state via a new Node Condition: **CapacityConfigured**. We intentionally mirror the state vocabulary established by In-Place Pod Resource Resize (KEP-1287).

Condition: **CapacityConfigured**

**Status: True**

**Reason Accepted**: The requested `ConfiguredCapacity` is valid, bounded by physical hardware constraints, and has been fully applied to local cgroups and `Node.Status`. The Kubelet considers the node stable and will not retry.

**Status: False**

**Reason InProgress**: The Kubelet has accepted a downscale request and is actively shrinking cgroups or gracefully evicting starved pods. The final `Node.Status` update is pending.

**Reason Infeasible**: The requested `ConfiguredCapacity` exceeds physical hardware limits (overcommit is disallowed in Alpha). The Kubelet has clamped the target to the physical limits, set this condition, and will **not retry** the oversized request. This provides explicit `lockSize` semantics: the Kubelet treats the `Infeasible` state as terminal for the current Spec value. An external controller or administrator must patch `Node.Spec.ConfiguredCapacity` to a valid, in-bounds value to resume normal reconciliation. A dedicated `lockSize` boolean field on `NodeSpec` was considered but rejected in favour of this condition reason — it provides the same terminal semantics without adding a new API field, and remains observable via standard `kubectl get node` condition output.

**Reason EmergencyReduced**: Physical hardware was forcefully removed (e.g., hypervisor-forced reclaim), falling below the current `Node.Spec.ConfiguredCapacity`. The Kubelet bypassed the API and clamped the node to the new physical reality to protect the kernel. Normal reconciliation is suspended until an external actor patches the Spec down to match the new physical bounds.


### Admission Control Contract

**Source of Truth:**
The `Node.Spec.ConfiguredCapacity` field is the authoritative declaration of desired logical capacity. `Node.Status.Capacity` and `Node.Status.Allocatable` are **read-only outputs** of the Kubelet's reconciliation — no external controller or webhook should patch them directly. The Kubelet is the sole component that writes to `Node.Status.Capacity`. This separation of Spec from Status follows the standard Kubernetes controller pattern.

**Path A (Orchestrated):** External controllers or administrators patch `Node.Spec.ConfiguredCapacity`. This mutation is intercepted by the cluster's standard Validating and Mutating Webhooks. If a webhook rejects the resize request, the API Server denies the PATCH, and the Kubelet's informer never receives the event — the node's effective capacity does not change.

**Path B (Emergency):** When physical hardware is forcefully reclaimed (e.g., hypervisor-forced deflation), the Kubelet reacts to cAdvisor directly and patches `Node.Status.Capacity` and `Node.Status.Conditions`. This path utilizes the standard Node Authorizer RBAC, bypassing Spec webhooks to ensure the control plane is immediately notified of physical degradation without needing an external controller to be available.

**Bootstrap Behavior:** Upon restart, the Kubelet reads `Node.Spec.ConfiguredCapacity` from the API Server as the primary capacity target *before* reading the raw cAdvisor hardware data. If a valid `ConfiguredCapacity` exists in the Spec, the Kubelet treats it as the desired state and validates it against live physical hardware. This ensures that a Kubelet restart on a pre-configured node does not accidentally override the declared configuration.

### Resource-Specific Validation Rules

The Alpha constraint (`ConfiguredCapacity <= Physical Capacity`) applies per-resource. The Kubelet validates the Spec against the host using the following resource-specific rules:

**CPU & Memory:** Strictly bounded by the physical hardware limits reported by cAdvisor. The `ConfiguredCapacity` for these resources must not exceed the raw physical quantity.

**Swap:** The `ConfiguredCapacity` for swap is validated against the total swap space currently allocated and active on the host's underlying OS (e.g., via `/proc/swaps`). It is not bounded by the physical RAM quantity. Note: configuring a logical memory capacity that *exceeds* physical RAM by relying on swap (overcommit) is explicitly out of scope for Alpha and is deferred to Future Work.

**Hugepages:** Validated against the pre-allocated hugepage pools configured at the OS kernel level (e.g., via `/sys/kernel/mm/hugepages`), not the total raw memory.

**Ephemeral Storage:** Validated against the available disk capacity as reported by the host OS. The `ConfiguredCapacity` for `ephemeral-storage` must not exceed the actual available disk space on the node's root filesystem.

**Unknown resource types:** Any resource type present in `ConfiguredCapacity` that the Kubelet does not recognise (e.g., custom extended resources) is silently ignored by the validation loop. Only well-known resource types (CPU, Memory, Swap, Hugepages, Ephemeral Storage) are validated and actioned.

### Security Considerations

This section explicitly addresses the security properties of this feature in response to the concern that a compromised node could over-report its capacity to the control plane in order to attract Pod scheduling and gain access to secrets it should not receive.

**Capacity Inflation Attack is Prevented by Design (Alpha):**
The Alpha enforcement rule (`ConfiguredCapacity <= Physical Capacity`) is the primary defense. The Kubelet's `calculateValidatedCapacity()` function enforces this bound locally by reading raw capacity from `cAdvisor`, which reads directly from the host kernel (`/sys`, `/proc`). For a node to successfully inflate its reported capacity above its physical reality, an attacker would need to compromise either:
1. The Kubelet binary itself, or
2. The kernel-level data sources that `cAdvisor` reads.

Both represent a full node compromise, which is already outside the Kubernetes threat model. A cluster-level actor patching `Node.Spec.ConfiguredCapacity` to an inflated value will have that spec clamped by the Kubelet and the `CapacityConfigured` condition set to `False (Reason: Infeasible)` — the API Server will store the spec, but the Kubelet will not act on it.

**Node Authorizer Governs Write Access:**
Mutations to `Node.Spec.ConfiguredCapacity` are governed by the standard Kubernetes Node Authorizer RBAC rules. Only identities with explicit write access to the Node object can set this field. Cluster administrators can additionally deploy a `ValidatingWebhookConfiguration` to restrict which controllers are permitted to set `ConfiguredCapacity` values and within what bounds.

**NRI/Runtime Boundary:**
The Kubelet does not introduce a new trust boundary between itself and the container runtime for capacity data. The runtime's view of resource limits is updated via the existing `UpdateContainerResources` CRI call — the same path used by In-Place Pod Resource Resize (KEP-1287) — which carries no new elevation of privilege.

### Architecture Flow

To safely support both external declarative triggers and physical hardware constraints, the Kubelet utilizes a dual-path reconciliation architecture based on the direction of the capacity change.

#### Path A1: Orchestrated Configuration-Driven Upscale

Spec mutation occurs before physical actuation, consistent with the API-first model.

**1. Spec Mutation:** The external controller patches `Node.Spec.ConfiguredCapacity` to the new higher target value. At this point, cAdvisor has not yet detected new hardware, so the Kubelet's validation check (`ConfiguredCapacity <= Physical Capacity`) will temporarily fail. The Kubelet sets the `CapacityConfigured` condition to `False` (Reason: `Infeasible`) and holds the pending event in the reconciliation channel.

**2. Physical Actuation:** The external controller proceeds to physically increase capacity via the hypervisor hotplug. Once the hardware is online, cAdvisor detects the new physical capacity on its next poll cycle (default: 5 minutes, or triggered immediately by the cAdvisor hardware-change path).

**3. Kubelet Reconciliation:** The cAdvisor drift triggers the reconciliation goroutine. The validation check now passes: `ConfiguredCapacity <= new Physical Capacity`. The Kubelet clears the `Infeasible` hold.

**4. Host Enforcement:** The `ContainerManager` re-initializes sub-managers via the `ResourceResizer` interface, expands the host `/kubepods` cgroups, and updates CRI swap boundaries.

**5. Status Dissemination:** The Kubelet patches `Node.Status.Capacity` and transitions the `CapacityConfigured` condition to `True` (Reason: `Accepted`), advertising the new capacity to the Scheduler.

#### Path A2: Orchestrated Configuration-Driven Downscale

Logical API changes occur prior to physical hardware removal.

**1. Spec Mutation:** The external controller patches `Node.Spec.ConfiguredCapacity` to a lower value before making any physical alterations to the host VM.

**2. Status Dissemination (Scheduler Block):** The Kubelet detects the change. It immediately updates `Node.Status.Capacity` and `Node.Status.Allocatable` to reflect the downscale, physically preventing the Scheduler from assigning new pods to the node. It simultaneously sets the `CapacityConfigured` condition to `False` (Reason: `InProgress`), signaling to external controllers that the node is actively in a transient shrinking state.

**3. Immediate Logic Enforcement:** The `ContainerManager` instantly locks internal state, shrinks the `/kubepods` host cgroups, and recalculates absolute Eviction Manager thresholds against the new logical baseline.

**4. Graceful Workload Degradation:** The Kubelet evaluates active workloads against the newly reduced capacity. If strict request contracts can no longer be met, starved pods are gracefully evicted with `Reason: NodeCapacityExceeded`.

**5. State Transition (Accepted):** Once evictions are complete and the node's boundaries are fully secured, the Kubelet transitions the `CapacityConfigured` condition to `True` (Reason: `Accepted`) and pushes the final status update.

**6. Physical Reclaim:** The external controller observes the `Accepted` condition and safely executes the physical hardware hot-unplug via the hypervisor.

#### Path B: Emergency Hardware-Driven Fallback (Ungraceful Downscale)

When physical hardware (e.g., Memory) is forcefully yanked by a hypervisor without prior API synchronization, physics dictates the timeline. The Kubelet acts as an emergency circuit breaker to secure the host.

**1. Hardware Trigger:** `cAdvisor` detects a drop in physical metrics that falls below the current `Node.Spec.ConfiguredCapacity` value.

**2. Immediate Host Enforcement:** The `ContainerManager` bypasses the API state and instantly shrinks the `/kubepods` cgroups to match the raw physical limits.

**3. Emergency Eviction:** The Kubelet evaluates active workloads against the raw physical bounds and immediately evicts starved pods.

**4. State Override & Alerting:** The Kubelet calculates the clamped target and patches `Node.Status.Capacity`. It transitions the `CapacityConfigured` Condition to `False` (Reason: `EmergencyReduced`) and emits a `Warning` Event (`EmergencyCapacityReduced`). To prevent infinite API loops, the Kubelet caches this clamped target locally and safely ignores the oversized Spec **until the Spec is updated by an external actor to match or fall below the new physical reality**.

**5. Divergence Resolution:** The external controller watches for the `EmergencyReduced` condition. Upon seeing it, the controller is responsible for patching `Node.Spec.ConfiguredCapacity` down to match reality, which clears the split-brain state and resumes normal Kubelet reconciliation behavior.

#### Flow Control: Container Swap Limit Recalculation

If a node is configured with Swap, a container's swap limit is dynamically proportional to the total node memory. Failing to update this during a resize leads to stranded resources or immediate kernel panics.

**Formula**: `(<containerMemoryRequest> / <nodeTotalMemory>) * <totalPodsSwapAvailable>`
```
T=0: Initial Node Resources
- Node Memory: 6G
- Node Swap: 4G
Pod (Running):
- MemoryRequest: 2G
Runtime Cgroup (<cgroup_path>/memory.swap.max): 1.33G

T=1: Hot-plug Memory (Upscale)
- Node Memory: 8G  (+2G)
- Node Swap: 4G
Pod (Running):
- MemoryRequest: 2G
Runtime Cgroup (<cgroup_path>/memory.swap.max): 1.0G (Recalculated via CRI)
```

#### Flow Control: Hardware Degradation and Capacity Starvation

During a hot-unplug (downscale) event, the node's physical capacity may drop below the total resources currently requested by active workloads. Because the Kubelet can no longer fulfill the strict scheduling contract, it must intervene to prevent system lockups or kernel panics.

* **Memory Starvation:** If the newly reduced node memory capacity falls below the aggregate memory requests or active working set of running pods, the Eviction Manager will trigger standard memory-pressure evictions.
* **CPU Starvation (Guaranteed QoS):** If the node's CPU core count drops below the threshold required to fulfill the exclusive core allocations of `Guaranteed` pods (managed by the `static` CPU Manager policy), the Kubelet cannot safely throttle the workload without violating the SLA.

In these starvation scenarios, the Kubelet's eviction manager will gracefully terminate the affected pods with a `Failed` status (Reason: `NodeCapacityExceeded`). This explicitly forces the cluster-wide controllers (e.g., Deployments, StatefulSets) to immediately reschedule the workload onto a capable, healthy node.

**Note on Static Pods:** Static pods are managed directly by the Kubelet via local manifest files and are not subject to eviction manager decisions. They will **not** be evicted during a capacity downscale. Cluster operators must manually account for the resource footprint of any static pods when setting a downscale target in `Node.Spec.ConfiguredCapacity`, ensuring the declared target leaves sufficient headroom above the aggregate static pod requests.

#### Compatibility with Cluster Autoscaler

The Cluster Autoscaler (CA) presently anticipates uniform allocatable values among nodes within the same NodeGroup, using existing nodes as templates for newly provisioned nodes. With In-Place Node Resource Resize, nodes within a single group may horizontally drift in capacity.

If not addressed, the CA could randomly select a dynamically scaled node as a template, assuming identical scaled values for all upcoming new nodes, leading to suboptimal or failed provisioning.

To ensure the Cluster Autoscaler remains stable, we will Capture the Node's Initial Allocatable Values via Annotations:

* During the initial boot, the Kubelet will stamp the node with an annotation representing its baseline boot capacity (e.g., `resize.node.kubernetes.io/initial-capacity: <json_resource_list>`).

* This baseline annotation remains immutable during dynamic resize events.

* The Cluster Autoscaler will be updated to read this annotation (if present) to construct reliable templates for new node provisioning, ignoring the dynamically shifting Node.Status.Capacity fields for template generation. (The corresponding CA changes are tracked separately and are out of scope for this KEP.)

### Step 1: Baseline Assumptions and Pre-requisite Validation

Currently, Kubernetes operates under the unspoken assumption that a `Node` object's capacity is immutable. Before implementing dynamic metric-based triggers (Step 3), we must formally document the behaviors we are breaking and add upstream test coverage to prove the core control plane can survive a Node capacity mutation (Step 1 & 2).

#### Behaviors We Are Breaking
1. **Static Capacity Assumption:** `Node.Status.Capacity` and `Allocatable` will transition from static, boot-time fields to dynamically mutable fields.
2. **Kubelet Restart Admission:** Currently, if a Kubelet is restarted on a machine whose hardware was reduced while offline, the Kubelet may blindly fail pod admission. We are shifting this to a graceful reconciliation and eviction model.
3. **Autoscaler Homogeneity:** Nodes within a single NodeGroup will no longer be guaranteed to have identical capacities, meaning the Autoscaler cannot blindly select any node as a provisioning template.
4. **External Controller Caches:** Third-party operators that cache node sizes indefinitely will become stale. (This is an accepted operational constraint).

#### Step 1 Pre-requisite Tests to be Added
To validate that the API and Control Plane can handle these broken assumptions natively, the following tests will be added before any dynamic cAdvisor triggers are built:

* **Test 1: API Server Mutation Acceptance**
    * *Action:* Manually patch `.status.capacity` and `.status.allocatable` on a `Ready` Node object.
    * *Validation:* Verify the API Server accepts the patch without systemic webhook rejections or validation failures.

* **Test 2: Scheduler Cache Invalidation (Upscale)**
    * *Action:* Create a pending Pod that requires 8Gi of memory on a cluster where the only node has 4Gi. Manually patch the Node object's capacity to 10Gi.
    * *Validation:* Verify the Scheduler detects the mutated Node object, updates its internal cache, and successfully schedules the pending Pod.

* **Test 3: Scheduler Cache Invalidation (Downscale)**
    * *Action:* Manually patch an empty Node's capacity from 10Gi down to 4Gi. Attempt to schedule a Pod requiring 8Gi.
    * *Validation:* Verify the Scheduler respects the mutated smaller capacity and rejects the Pod (leaves it Pending), proving it does not rely on a stale boot-time cache.

* **Test 4: Kubelet Restart on Resized Hardware (Unified Reconciliation)**
    * *Action:* Schedule a Pod. Stop the Kubelet. Mock the underlying machine info to reflect a smaller capacity (simulate offline hot-unplug). Start the Kubelet.
    * *Validation:* Verify the Kubelet boots successfully, recognizes the discrepancy between the API and physical hardware, and handles the change gracefully (e.g., evicting the pod if starved) rather than crashing or permanently locking pod admission.
#### Proposed Core Code Changes

1. **The Dedicated Kubelet Listener** (`pkg/kubelet/kubelet.go`)

   Instead of hijacking the main syncLoopIteration, capacity updates are handled cleanly in the Kubelet's asynchronous `Run()` method:
```go
// 1. Wire up the Node Informer to trigger the reconciliation loop
kl.nodeInformer.AddEventHandler(cache.ResourceEventHandlerFuncs{
    UpdateFunc: func(oldObj, newObj interface{}) {
        oldNode := oldObj.(*v1.Node)
        newNode := newObj.(*v1.Node)
        if !apiequality.Semantic.DeepEqual(oldNode.Spec.ConfiguredCapacity, newNode.Spec.ConfiguredCapacity) {
            kl.capacityReconciliationCh <- struct{}{}
        }
    },
})

// 2. Listen for dynamic node capacity changes if the feature gate is enabled
if utilfeature.DefaultFeatureGate.Enabled(features.InPlaceNodeResourceResize) {
    go wait.Until(func(ctx context.Context) {
        // capacityReconciliationCh fires on Node.Spec updates OR cAdvisor hardware drift
        for range kl.capacityReconciliationCh {
            
            configuredCapacity := kl.getNodeSpecConfiguredCapacity()
            physicalInfo, err := kl.cadvisor.MachineInfo()
            if err != nil { continue }

            // Determine Target Capacity (Alpha rule: Configured <= Physical)
            targetCapacity, condition := kl.calculateValidatedCapacity(configuredCapacity, physicalInfo)

            // Anti-Thrash Guard: Break infinite loops if Spec > Physical
            if kl.requiresReconciliation(targetCapacity) {
                
                // Refresh internal cache to the validated target capacity
                kl.setCachedMachineInfo(targetCapacity)
                
                // CRITICAL DOWNSCALE FIX: Update Status and Allocatable FIRST 
                // to prevent Scheduler livelocks during the eviction window.
                kl.updateNodeCondition(InProgress)
                kl.syncNodeStatus(ctx) 
                
                // Enforce Host Boundaries (cgroups, sub-managers)
                kl.containerManager.SyncCapacity(targetCapacity)

                // Sync Eviction Thresholds for safe downscaling
                kl.evictionManager.SynchronizeThresholds(targetCapacity)

                // Gracefully evict pods starved by the new bounds
                kl.evictStarvedPods(ctx, targetCapacity)

                // Update active container Swap boundaries via standard CRI RPC
                for _, pod := range kl.GetActivePods() {
                    kl.syncContainerSwapLimits(ctx, pod, targetCapacity)
                }

                // Disseminate final resolved state (Accepted, Infeasible, or EmergencyReduced)
                kl.updateNodeCondition(condition)
                kl.syncNodeStatus(ctx)
            }
        }
    }, 0, wait.NeverStop)
}
```

2. **The Sub-Manager Interfaces** (`pkg/kubelet/kubelet.go`)

   Resource managers must implement a new interface to accept dynamic sync events natively, avoiding a full Kubelet restart.
```go
// ResourceResizer defines the interface for sub-managers to accept dynamic capacity changes
type ResourceResizer interface {
    // SyncCapacity safely re-evaluates the sub-manager's internal state against the new boundaries
    SyncCapacity(capacity ResourceList) error
}
```

3. **Resource Manager Synchronization**

When the `ContainerManager` detects a capacity drift, it invokes the `SyncCapacity()` method on its internal sub-managers. This triggers specific state reconciliations:

- **CPU Manager:** 
  * **Upscale (Hot-plug):** 
  
  Newly added CPUs are instantly detected and added to the "shared pool" (the default cpuset). They immediately become available for pods in the Burstable and BestEffort QoS classes, or for new Guaranteed pods requesting exclusive cores.

  * **Downscale (Hot-unplug):**

  If CPUs are removed, the CPU Manager removes them from the shared pool. If the new total CPU core count falls below what is required to fulfill the strict, exclusive core allocations of existing Guaranteed pods, the Kubelet's capacity reconciliation loop intercepts this scheduling contract violation and evicts the affected pods with a `Failed` status (Reason: `NodeCapacityExceeded`). Additionally, if the removed CPU IDs directly overlap with the exclusive cpuset already pinned to a running Guaranteed pod — even if the total remaining core count appears sufficient — those pods are also evicted, because the specific hardware they were guaranteed no longer exists. The CPU Manager validates both the total count and cpuset membership before clearing the violation.

- **Memory Manager:**

  - The Memory Manager recalculates the total memory and hugepages available per NUMA node. It updates its internal state machine so that future TopologyManager admission checks accurately reflect the resized NUMA boundaries.

- **Topology Manager:**

  - While the Topology Manager itself does not store capacity state, the underlying updates to the CPU and Memory managers ensure that any subsequent topology alignment checks (for new pods) use the freshly updated hardware boundaries.

### Observability and Metrics

To ensure cluster operators can monitor resize events and failures, this KEP introduces the following Prometheus metrics within the Kubelet:

- `kubelet_node_resize_requests_total`: Counter tracking the number of successful native resource resize events (labeled by resource name and direction: `increase`/`decrease`).

- `kubelet_node_resize_errors_total`: Counter tracking failures during the reconciliation pipeline (labeled by the failing subsystem, e.g., `cpu_manager_sync`, `cgroup_update`).

### Test Plan

[x] I/we understand the owners of the involved components may require updates to
existing tests to make this code solid enough prior to committing the changes necessary
to implement this enhancement.

##### Step 1: Baseline API & Scheduler Validation (Pre-requisite Tests)
Before dynamic hardware triggers are implemented, upstream test coverage must be added to validate the foundational assumption that Kubernetes natively supports Node object capacity mutation:
1. **API Validation:** Verify that modifying `Node.Status.Capacity` and `Node.Status.Allocatable` on an existing Node object is explicitly permitted by the API server and does not trigger unintended systemic webhook rejections.
2. **Scheduler Validation:** Verify that if a Node's capacity is mutated (simulating an offline/restarted Kubelet resize), the Scheduler correctly recognizes the new capacity and successfully schedules/rejects pending Pods accordingly without requiring the Node object to be deleted and recreated.
3. **Autoscaler Integration:** Verify how the Cluster Autoscaler reacts to a dynamically mutated Node object capacity.

##### Unit tests

1. **cAdvisor Cache Refresh** (`kubelet_node_status_test.go`): Verify that injecting a new `MachineInfo` struct successfully updates the Kubelet's internal cache, and that the subsequent node status sync does not incorrectly clamp the new `Allocatable` values to the old boot-time capacity.

2. **Cgroup Enforcement** (`container_manager_linux_test.go`): Verify that when a capacity change is detected, `enforceNodeAllocatableCgroups` and `UpdateQOSCgroups` are invoked with the newly calculated boundaries, and that the `nodeCapacityUpdateCh` signal is successfully emitted without blocking.

3. **Sub-Manager Re-initialization** (`cpu_manager_test.go`, `memory_manager_test.go`): Verify that the CPU and Memory managers properly implement the `ResourceResizer` interface and cleanly accept `SyncCapacity()` calls without leaking state or crashing.

4. **Eviction Threshold Sync** (`eviction_manager_test.go`): Verify that `SynchronizeThresholds` correctly recalculates absolute byte values (e.g., < `100Mi` vs `10%`) when the underlying capacity increases or decreases.

5. **CRI Swap Limit Recalculation** (`kubelet_test.go`): Verify the math for proportional swap limits. Ensure the capacity reconciliation loop correctly iterates over active pods and invokes the mock CRI `UpdateContainerResources` interface with the newly calculated boundaries.

6. **Bootstrap Parity** (`kubelet_test.go`): Verify that when the Kubelet starts and finds a pre-existing `Node.Spec.ConfiguredCapacity` in the API Server, it uses that value as the desired target rather than silently overwriting it with the raw cAdvisor-discovered capacity. Specifically confirm that if `ConfiguredCapacity` is set to 20Gi on a 32Gi physical node, the Kubelet reports 20Gi in `Node.Status.Capacity` after startup, not 32Gi.

##### e2e tests

These tests will utilize a mock `cAdvisor` interface to inject dynamic hardware capacity changes into a running test Kubelet to validate the end-to-end reconciliation pipeline.

* **Scenario 1: Safe Upscale and Scheduling**

  - **Action**: Inject an upscale event (e.g., `10G` -> `15G` memory).

  - **Validation:** Verify the `/kubepods` host cgroup expands. Verify the Node API object reflects the new capacity. Verify a previously `Pending` pod (due to lack of memory) is successfully scheduled and transitions to `Running`.


* **Scenario 2: Safe Downscale and Eviction (Resource Starvation)**

  - **Action**: Schedule pods that consume 8G of memory. Inject a downscale event reducing the node's total memory to 5G.

  - **Validation**: Verify the Kubelet updates its Eviction Manager thresholds and successfully evicts the lowest-priority pod (e.g., `BestEffort`) to protect the node before the cgroups enforce the 5G limit.


* **Scenario 3: Proportional Swap Recalculation**

  - **Action**: Deploy a pod on a swap-enabled node. Inject a memory upscale event.

  - **Validation**: Inspect the active pod's `memory.swap.max` cgroup file on the host filesystem and verify the limit was proportionally reduced based on the new total node memory.


* **Scenario 4: Upsize -> Downsize -> Upsize (Flapping)**

  - **Action**: Rapidly inject alternating capacity changes.

  - **Validation**: Ensure the `ContainerManager` reconciliation loop does not deadlock, the cgroups settle on the final capacity, and no duplicate capacity update signals block the main Kubelet loop.

### Graduation Criteria

#### Phase 1: Alpha (target 1.37)

* Feature is disabled by default via the `InPlaceNodeResourceResize` feature gate.
* **Step 1 (Baseline Validation):** API, Scheduler, and Autoscaler e2e tests are merged to officially support and document the behavior of mutating an existing Node object's capacity.
* **Step 2 (Unified Reconciliation):** The Kubelet's internal logic is updated to gracefully handle capacity mismatches declaratively. If the Kubelet's internal state differs from the API (for hotplug) or physical hardware (for hot-unplug), it safely reconciles cgroups and evicts starved pods rather than failing admission blindly.
* **Step 3 (Dynamic Trigger):** The `cAdvisor` metrics-based trigger is implemented to automate the reconciliation loop dynamically on running nodes.

### Upgrade / Downgrade Strategy

<!--
If applicable, how will the component be upgraded and downgraded? Make sure
this is in the test plan.

Consider the following in developing an upgrade/downgrade strategy for this
enhancement:
- What changes (in invocations, configurations, API use, etc.) is an existing
  cluster required to make on upgrade, in order to maintain previous behavior?
- What changes (in invocations, configurations, API use, etc.) is an existing
  cluster required to make on upgrade, in order to make use of the enhancement?
-->

##### Upgrade 

To upgrade the cluster to use this feature, the Kubelet must be restarted with the `InPlaceNodeResourceResize` feature gate enabled. Existing clusters do not experience any immediate impact upon upgrade; the Kubelet will simply begin mirroring the existing physical hardware capacity dynamically.

##### Downgrade

It is trivially possible to downgrade by disabling the feature gate and restarting the Kubelet. The Kubelet will simply revert to its legacy behavior: capturing the node capacity once during boot and freezing it. Any subsequent hardware hot-plugs will be safely ignored.

### Version Skew Strategy

<!--
If applicable, how will the component handle version skew with other
components? What are the guarantees? Make sure this is in the test plan.

Consider the following in developing a version skew strategy for this
enhancement:
- Does this enhancement involve coordinating behavior in the control plane and
  in the kubelet? How does an n-2 kubelet without this feature available behave
  when this feature is used?
- Will any other components on the node change? For example, changes to CSI,
  CRI or CNI may require updating that component before the kubelet.
-->

The interaction between the Kubelet and the control plane (specifically the Scheduler) relies entirely on standard Node API update events. When the Kubelet patches the Node.Status.Capacity and Allocatable fields, the scheduler's existing node update event handler seamlessly processes the altered capacity.

Because this leverages pre-existing API contracts, no special coordination or version skew mitigation is required between the Kubelet and the control plane. Similarly, no updates are required for CRI, CNI, or CSI plugins prior to enabling this Kubelet feature.

**Scheduler-Kubelet Race Window:** A capacity change, like a node going `NotReady`, can invalidate an in-flight scheduling decision. This race window is inherent to the Kubernetes scheduler's optimistic concurrency model and is not unique to this KEP. Specifically, for systems using Workload Aware Scheduling (WAS), a capacity downscale occurring between the scheduler's binding decision and the Kubelet's admission check may cause a pod rejection. The Kubelet will return an admission failure, and the pod will be rescheduled by its controlling workload controller. This behavior is consistent with existing failure handling in the scheduler and does not require changes to the scheduler for Alpha.

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

- [x] Feature gate (also fill in values in `kep.yaml`)
    - Feature gate name: `InPlaceNodeResourceResize`
    - Components depending on the feature gate: `kubelet`
    - Will enabling / disabling the feature require downtime of the control plane? **No.** The feature gate is Kubelet-only; the API Server and Scheduler require no changes and no restart.
    - Will enabling / disabling the feature require downtime or reprovisioning of a node? **Yes, a Kubelet restart is required** to toggle the feature gate. However, a Kubelet restart does not disrupt running pods; it only briefly interrupts the Kubelet process itself while existing cgroups and container state are preserved by the container runtime.

###### Does enabling the feature change any default behavior?

No immediate behavior changes occur if the underlying node hardware has not changed.
If the hardware does change, the default behavior changes from "ignoring the hardware change" to:

- Upscale: Dynamically patching the `Node` Allocatable resources and rewriting host cgroups, allowing pending pods to be scheduled.

- Downscale: Dynamically shrinking host cgroups, lowering Eviction Manager thresholds, and potentially triggering pod evictions.

###### Can the feature be disabled once it has been enabled (i.e. can we roll back the enablement)?

Yes. The feature can be disabled by restarting the Kubelet with the feature gate turned off. The Kubelet will freeze its capacity at whatever `cAdvisor` reported during that specific boot cycle.

###### What happens if we re-enable the feature if it was previously rolled back?

The Kubelet will immediately poll the live cAdvisor data, detect any drift that occurred while the feature was disabled, and execute a one-time reconciliation to update the internal cgroups and the API Server's `Node.Status`.

###### Are there any tests for feature enablement/disablement?

Yes, unit tests will validate that the `capacityReconciler` Go routine completely short-circuits and exits if `utilfeature.DefaultFeatureGate.Enabled()` returns false.

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

Rollout failures are isolated to the specific node. If the `ContainerManager` fails to enforce the new top-level `/kubepods` cgroups due to a filesystem error, the main Kubelet loop will not be signaled, and the API Server will not be updated. Existing running workloads are perfectly safe and will continue to operate under their current cgroup limits.

###### What specific metrics should inform a rollback?

<!--
What signals should users be paying attention to when the feature is young
that might indicate a serious problem?
-->
An operator should roll back the feature if there is a sustained spike in the `kubelet_node_resize_errors_total` metric, indicating the Kubelet is deadlocking or failing to write to the host filesystem.

###### Were upgrade and rollback tested? Was the upgrade->downgrade->upgrade path tested?

<!--
Describe manual testing that was done and the outcomes.
Longer term, we may want to require automated upgrade/rollback tests, but we
are missing a bunch of machinery and tooling and can't do that now.
-->

Yes, manual testing of the upgrade -> downgrade -> upgrade path validates that the Kubelet safely falls back to static boot-time caching without disrupting running workloads.

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

Monitor the metrics
- `kubelet_node_resize_requests_total`
- `kubelet_node_resize_errors_total`

###### How can an operator determine if the feature is in use by workloads?

<!--
Ideally, this should be a metric. Operations against the Kubernetes API (e.g.,
checking if there are objects with field X set) may be a last resort. Avoid
logs or events for this purpose.
-->

The enablement of the Kubelet feature gate can be determined via the `kubernetes_feature_enabled` metric. Operational use can be observed when the `kubelet_node_resize_requests_total` counter increments during a hardware change.

###### How can someone using this feature know that it is working for their instance?

<!--
For instance, if this is a pod-related feature, it should be possible to determine if the feature is functioning properly
for each individual pod.
Pick one more of these and delete the rest.
Please describe all items visible to end users below with sufficient detail so that they can verify correct enablement
and operation of this feature.
Recall that end users cannot usually observe component logs or access metrics.
-->

An end-user can verify the feature by executing a hot-plug via their hypervisor, and then running `kubectl get node <node-name> -o yaml`. The `.status.capacity` and `.status.allocatable` fields will natively reflect the newly added hardware within ~10 seconds.

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

For each dynamically resized node:

- **Error rate:** The `kubelet_node_resize_errors_total` counter is expected to remain strictly at `0` during normal operations.
- **Latency (Spec-driven):** For `Node.Spec.ConfiguredCapacity`-triggered resize events, the time from Spec patch to `CapacityConfigured: Accepted` condition should complete within **30 seconds** under normal conditions, bounded by informer propagation latency and cgroup write time.
- **Latency (Hardware-driven):** For physical hardware events detected via cAdvisor polling, the reconciliation completes within one cAdvisor polling cycle (default: **5 minutes**). Operators who require lower latency should configure a shorter cAdvisor polling interval.

###### What are the SLIs (Service Level Indicators) an operator can use to determine the health of the service?

<!--
Pick one more of these and delete the rest.
-->

- [X] Metrics
    - Metric name:
      - `kubelet_node_resize_requests_total`
      - `kubelet_node_resize_errors_total`
   - Components exposing the metric: kubelet

###### Are there any missing metrics that would be useful to have to improve observability of this feature?

<!--
Describe the metrics themselves and the reasons why they weren't added (e.g., cost,
implementation difficulties, etc.).
-->
No

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

**cAdvisor (Internal):** The Kubelet strictly relies on the integrated `cAdvisor` package to successfully read the underlying Linux kernel and hardware capacity.

**Container Runtime (CRI):** The Kubelet relies on the runtime (e.g., containerd, CRI-O) to successfully honor the `UpdateContainerResources` RPC call to propagate recalculated Swap limits.

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
Yes.

The Kubelet's existing NodeInformer will now actively process and react to UPDATE events on the `Node.Spec.ConfiguredCapacity` field.

During a resize event, the Kubelet will issue PATCH calls to the Node status subresource to update `.status.capacity`, `.status.allocatable`, and `.status.conditions`. To prevent API thrashing during emergency hardware fluctuations, the Kubelet uses a jitter tolerance filter and caches the clamped state locally.

###### Will enabling / using this feature result in introducing new API types?

<!--
Describe them, providing:
  - API type
  - Supported number of objects per cluster
  - Supported number of objects per namespace (for namespace-scoped objects)
-->
Yes. This introduces a new optional field `ConfiguredCapacity` within `Node.Spec`, and a new Node Condition type `CapacityConfigured` to track the reconciliation state (Accepted, InProgress, Infeasible, EmergencyReduced).

###### Will enabling / using this feature result in any new calls to the cloud provider?

<!--
Describe them, providing:
  - Which API(s):
  - Estimated increase:
-->
No
###### Will enabling / using this feature result in increasing size or count of the existing API objects?

<!--
Describe them, providing:
  - API type(s):
  - Estimated increase in size: (e.g., new annotation of size 32B)
  - Estimated amount of new objects: (e.g., new Object X for every existing Pod)
-->
Yes.

- API type(s): `Node`
- Estimated increase in size: ~200-500 bytes per Node object. This is due to the addition of the `Node.Spec.ConfiguredCapacity` field, the `CapacityConfigured` Status Condition, and the initial-capacity annotation for the Autoscaler.
- Estimated amount of new objects: 0 (No new objects are created; only the existing Node object is annotated).

###### Will enabling / using this feature result in increasing time taken by any operations covered by existing SLIs/SLOs?

<!--
Look at the [existing SLIs/SLOs].

Think about adding additional work or introducing new steps in between
(e.g. need to do X to start a container), etc. Please describe the details.

[existing SLIs/SLOs]: https://git.k8s.io/community/sig-scalability/slos/slos.md#kubernetes-slisslos
-->

Negligible, In the case of resource reconfiguration the resource manager may take some time to re-sync.

###### Will enabling / using this feature result in non-negligible increase of resource usage (CPU, RAM, disk, IO, ...) in any components?

<!--
Things to keep in mind include: additional in-memory state, additional
non-trivial computations, excessive access to disks (including increased log
volume), significant amount of data sent and/or received over network, etc.
This through this both in small and large cases, again with respect to the
[supported limits].

[supported limits]: https://git.k8s.io/community//sig-scalability/configs-and-limits/thresholds.md
-->

Negligible computational overhead is introduced. The Kubelet utilizes an event-driven Go channel to signal capacity updates, avoiding heavy CPU polling cycles.


###### Can enabling / using this feature result in resource exhaustion of some node resources (PIDs, sockets, inodes, etc.)?

<!--
Focus not just on happy cases, but primarily on more pathological cases
(e.g. probes taking a minute instead of milliseconds, failed pods consuming resources, etc.).
If any of the resources can be exhausted, how this is mitigated with the existing limits
(e.g. pods per node) or new limits added by this KEP?

Are there any tests that were run/should be run to understand performance characteristics better
and validate the declared limits?
-->

Yes, organically. Expanding a node's capacity allows the Scheduler to place more Pods onto the node, consuming more PIDs/sockets. However, this is strictly mitigated by the pre-existing `--max-pods` Kubelet configuration, which enforces a hard ceiling regardless of the underlying hardware size.

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

If the API Server is unavailable during a hardware resize, the Kubelet will successfully update the local host cgroups, sub-managers, and Eviction thresholds to protect the node. However, the `syncNodeStatus` call will fail. The local node will be physically resized and stable, but the cluster Scheduler will remain "blind" to the new capacity until API connectivity is restored.

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

* **Cgroup Enforcement Failure**

  - **Detection**: Spike in `kubelet_node_resize_errors_total` with the label `subsystem="cgroup_update"`.

  - **Mitigations**: Investigate host filesystem or AppArmor/SELinux denials preventing the Kubelet from writing to `/sys/fs/cgroup`.

* **CRI Swap Update Failure**

  - **Detection**: Spike in `kubelet_node_resize_errors_total` with the label `subsystem="container_swap_resize"`.

  - **Mitigations**: Verify the Container Runtime (containerd/CRI-O) is healthy and accepting RPC calls.

* **Downscale Stuck in InProgress (Eviction Stall)**

  - **Scenario**: A downscale was initiated via `Node.Spec.ConfiguredCapacity`. The Kubelet set the `CapacityConfigured` condition to `False (Reason: InProgress)` and began evicting starved pods. However, the eviction never completes — for example, because affected pods have `PodDisruptionBudgets` that block eviction, or the pods are `Guaranteed` QoS with no safe eviction path, or the Kubelet crashed mid-eviction.

  - **Detection**: The `CapacityConfigured` condition remains `False (Reason: InProgress)` for longer than expected. No `kubelet_node_resize_requests_total` increment is observed for the direction `decrease`. The external controller's watch on the `Accepted` condition never fires.

  - **Mitigations**:
    1. Temporarily patch `Node.Spec.ConfiguredCapacity` back to the previous (higher) value to cancel the downscale and unblock the node. The Kubelet will detect the spec revert, restore the cgroup boundaries, and transition the condition to `Accepted`.
    2. Identify and resolve the blocking condition (e.g., adjust PodDisruptionBudgets, force-delete the stalled pod) and re-issue the downscale spec patch.
    3. As a last resort, disable the feature gate and restart the Kubelet to freeze capacity evaluation.

###### What steps should be taken if SLOs are not being met to determine the problem?

Examine Kubelet logs for errors emitted by `container_manager_linux`.go. Disable the feature gate to freeze capacity evaluation until the host-level conflict is resolved.

## Implementation History

- **2023-04-17**: Initial KEP PR ([#3955](https://github.com/kubernetes/enhancements/pull/3955)) opened as *KEP-3953: Dynamic Node Resize* — original scope covering both scale-up and scale-down via cAdvisor polling.
- **2024-01-31**: Scope narrowed to scale-up only (*Node Resource Hot Plug*) following community feedback that a separate CRI-based hardware discovery mechanism was needed before scale-down could be safely addressed.
- **2025-01-13**: KEP retitled to *KEP-3953: Node Resource Hot Plug* to reflect the updated focus on upscaling; Production Readiness Review Questionnaire updated.
- **2025-02-12**: PRR approved for Alpha. Key design additions: swap limit recalculation for existing containers via `UpdateContainerResources`, OOMScoreAdj drift accepted as a known limitation, hot-unplug emergency path outlined in Future Work.
- **2026-02-10**: KEP retitled to *KEP-3953: In-place Node Resource Resize* to reflect the full bidirectional resize scope introduced by the declarative `Node.Spec.ConfiguredCapacity` API field — a major design pivot driven by reviewer feedback.

## Drawbacks

<!--
Why should this KEP _not_ be implemented?
-->

If dynamically removed resources (specifically CPUs or NUMA Memory zones) were exclusively pinned and allocated to specific containers via the CPU Manager or Topology Manager, the underlying hardware backing their strict isolation guarantees no longer exists on the motherboard.

Because the node can no longer fulfill the Pod's strict Requests contract, those affected pods must be forcefully evicted by the Kubelet with a `Failed` status (Reason: `NodeCapacityExceeded`). While this protects the node, it does introduce a disruptive pod termination that would not occur if the node capacity remained static.

## Alternatives
<!--
What other approaches did you consider, and why did you rule them out? These do
not need to be as detailed as the proposal, but should include enough
information to express the idea and why it was not acceptable.
-->

* **Node Replacement (Horizontal Scaling):** Instead of resizing existing nodes, operators can horizontally scale by provisioning entirely new, larger nodes, draining the original nodes, and deleting them.

  * _Why it was rejected:_ This introduces significant workload disruption during the drain process, increases control plane overhead, and takes considerably longer to execute than a simple underlying hypervisor hot-plug.


* **Manual Kubelet Restarts:** Administrators can hot-plug the hardware and then manually restart the `kubelet` systemd service to force it to read the new capacity.

  * _Why it was rejected:_ This causes temporary node `NotReady` states, breaks active `exec`/`port-forward` sessions, and risks triggering historic edge-case bugs associated with Kubelet restarts.

* **Balloon Drivers / Fake Placeholder Resources:** Pre-provisioning massive virtual machines but using memory ballooning or fake placeholder resources to artificially restrict the Kubelet, "inflating" them when capacity is needed.

  * _Why it was rejected:_ This is highly inefficient, complex to manage at the hypervisor level, and confuses the Kubernetes Scheduler, which relies on accurate, native cgroup boundaries.

* **Node Annotation as Configuration Mechanism** (`resize.node.kubernetes.io/configured-capacity`): Using a Node annotation instead of a first-class `NodeSpec` field to carry the desired capacity declaration.

  * _Why it was rejected:_ Annotations are unstructured strings with no API validation, no admission webhook targeting support, and no defaulting semantics. They are effectively a workaround for the absence of a proper API field. Cluster administrators wanting to restrict who can set capacity would have to write brittle label-matching admission webhooks rather than using structured `ValidatingWebhookConfiguration` field selectors. A formal `NodeSpec` field provides schema validation, clean `kubectl diff` output, and correct versioning/defaulting via the API machinery. An annotation is a hack around the right answer.

* **Local Kubelet Configuration File (Static Config):** Expressing the desired logical capacity via a local file on the node host (e.g., a `KubeletConfiguration` field), requiring a Kubelet restart to apply.

  * _Why it was rejected:_ Local configuration fundamentally cannot be driven by external controllers. A cluster-level controller managing capacity across many nodes cannot atomically write a file to a remote node's filesystem and then restart its Kubelet. This approach breaks the API-driven operational model of Kubernetes, makes orchestration of downscale workflows impossible without SSH/node access, and requires a Kubelet restart — defeating the core goal of this KEP.

* **CRI-Based Hardware Discovery (Alternative Trigger):** Using a Container Runtime Interface (CRI) extension to deliver hardware capacity events to the Kubelet instead of relying on cAdvisor polling.

  * _Why it was not chosen for Alpha:_ A CRI-based discovery mechanism would require new CRI API additions and runtime support across containerd, CRI-O, and other runtimes — a multi-org coordination effort that is orthogonal to the Kubelet reconciliation logic this KEP introduces. The cAdvisor-based polling approach is available today on all supported runtimes and platforms. This alternative is tracked as a future evolution path via **KEP-5224** (Node Resource Discovery) and is explicitly called out in the Future Work section.

* **Physical Hot-Unplug as a Considered-But-Deferred Trigger:** Having the Kubelet proactively orchestrate or initiate physical hardware removal (i.e., calling a hypervisor API to perform hot-unplug) as part of a downscale flow.

  * _Why it was deferred:_ The Kubelet has no knowledge of the hypervisor or infrastructure layer. Introducing such a call would violate the single-responsibility principle and couple the Kubelet to provider-specific infrastructure APIs. The correct model is for an external controller (which _does_ understand the infrastructure) to coordinate the physical hot-unplug after observing that the Kubelet has completed its graceful downscale (i.e., `CapacityConfigured` condition reaches `Accepted`). The KEP's Path A2 flow explicitly documents this coordination contract.

## Infrastructure Needed (Optional)

For standard Kubernetes CI (e2e_node tests), no special infrastructure is needed because the tests will utilize a mocked cAdvisor client to simulate hardware capacity events.

However, for provider-specific end-to-end integration testing in the future, underlying infrastructure VMs that natively support CPU and Memory hot-plugging will be required to validate the complete hardware-to-API lifecycle.

## Future Work

* **Dynamic System and Kube Reserved Adjustments**

  * Currently, the `--system-reserved` and `--kube-reserved` values are static configurations defined during Kubelet bootstrap. If a node scales massively (e.g., from 16GB to 128GB of memory), the OS and Kubelet might organically require a dynamically scaled reservation rather than the original static threshold. Future iterations could explore allowing these reservations to be expressed as percentages or dynamic tunables.

* **NRI (Node Resource Interface) Integration**

  * Extending the internal Kubelet resize event broadcaster so that external NRI plugins can natively subscribe to hardware capacity changes. This would allow third-party runtime wrappers and advanced topology managers to react to node upscales and downscales simultaneously with the Kubelet.

* **Event-Driven Hardware Detection ([Node Resource Discovery](https://github.com/kubernetes/enhancements/pull/5319))**

    * Currently, this KEP relies on `cAdvisor` and lightweight host polling to detect physical hardware changes. In the future, as **KEP-5224** matures, the responsibility of hardware discovery will shift toward the Container Runtime Interface (CRI) and external resource plugins. Once the CRI is capable of natively broadcasting dynamic hardware capacity events to the Kubelet, this KEP's capacity reconciliation loop will be updated to subscribe directly to those CRI events.

* **Node Capacity Overcommit (Logical > Physical)**

    * This KEP explicitly defers support for configuring `Node.Spec.ConfiguredCapacity` to a value greater than the raw physical hardware capacity (e.g., reporting 48Gi of memory on a 32Gi machine backed by swap). This is a compelling use-case — particularly for swap-overcommit scenarios where the OS's swap space provides a meaningful backing store for workloads that tolerate memory latency. However, enabling this in Alpha would break the Eviction Manager's absolute threshold math and OOM-killer assumptions, which rely on the invariant that logical ≤ physical. Future work will define how the Eviction Manager, cgroup limits, and memory accounting interact when logical capacity exceeds physical, and will specify per-resource rules (e.g., Swap may be the first resource exempt from the Alpha overcommit restriction, while CPU and raw Memory remain bounded by physical reality).
