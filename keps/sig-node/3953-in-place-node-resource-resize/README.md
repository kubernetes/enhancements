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
  - [The Problem: Live Mutation of Node Capacity Is Unsupported Today](#the-problem-live-mutation-of-node-capacity-is-unsupported-today)
  - [The Need: Dynamic Hardware Requires a Dynamic Capacity Model](#the-need-dynamic-hardware-requires-a-dynamic-capacity-model)
  - [The Two-Layer Solution](#the-two-layer-solution)
    - [Layer 1: Making the Ecosystem Safe for Mutable Capacity](#layer-1-making-the-ecosystem-safe-for-mutable-capacity)
    - [Layer 2: Declarative Capacity Actuation](#layer-2-declarative-capacity-actuation)
  - [Why a Declarative API Is Essential for Safe Downscaling](#why-a-declarative-api-is-essential-for-safe-downscaling)
  - [Benefits Beyond the Workaround](#benefits-beyond-the-workaround)
  - [Goals](#goals)
  - [Non-Goals](#non-goals)
- [Proposal](#proposal)
  - [Layer 1: Ecosystem Tolerance for Mutable Node Capacity](#layer-1-ecosystem-tolerance-for-mutable-node-capacity)
  - [Layer 2: Declarative Capacity Actuation via <code>Node.Spec.ConfiguredCapacity</code> <em>(targeted for v1.39 Alpha)</em>](#layer-2-declarative-capacity-actuation-via-nodespecconfiguredcapacity-targeted-for-v139-alpha)
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
  - [Layer 1 Implementation: Ecosystem Tolerance — Validating That Mutable Capacity Is Safe](#layer-1-implementation-ecosystem-tolerance--validating-that-mutable-capacity-is-safe)
    - [Assumptions Being Changed](#assumptions-being-changed)
    - [Layer 1 Pre-requisite Tests](#layer-1-pre-requisite-tests)
  - [Layer 2 Implementation: Declarative Capacity Actuation](#layer-2-implementation-declarative-capacity-actuation)
    - [Proposed Core Code Changes](#proposed-core-code-changes)
    - [3.1 CPU Manager Synchronization](#31-cpu-manager-synchronization)
    - [3.2 Memory Manager and Memory QoS Synchronization](#32-memory-manager-and-memory-qos-synchronization)
    - [3.3 Topology Manager and NUMA Layout](#33-topology-manager-and-numa-layout)
    - [3.4 Checkpoint State File Consistency](#34-checkpoint-state-file-consistency)
    - [3.5 Feature Scope Progression (Alpha to GA)](#35-feature-scope-progression-alpha-to-ga)
  - [Observability and Metrics](#observability-and-metrics)
  - [Test Plan](#test-plan)
      - [Layer 1: Ecosystem Tolerance Tests (Pre-requisite, no feature gate required)](#layer-1-ecosystem-tolerance-tests-pre-requisite-no-feature-gate-required)
      - [Layer 2: Unit tests (require <code>InPlaceNodeResourceResize</code> feature gate)](#layer-2-unit-tests-require-inplacenoderesourceresize-feature-gate)
      - [Layer 2: e2e tests (require <code>InPlaceNodeResourceResize</code> feature gate)](#layer-2-e2e-tests-require-inplacenoderesourceresize-feature-gate)
  - [Graduation Criteria](#graduation-criteria)
    - [Phase 1: Alpha (v1.38) — Layer 1: Ecosystem Tolerance](#phase-1-alpha-v138--layer-1-ecosystem-tolerance)
    - [Phase 2: Alpha (v1.39) — Layer 2: Declarative Capacity Actuation <em>(planned)</em>](#phase-2-alpha-v139--layer-2-declarative-capacity-actuation-planned)
    - [Phase 3: Beta](#phase-3-beta)
    - [Phase 4: GA](#phase-4-ga)
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
- [Open Questions for Layer 2](#open-questions-for-layer-2)
- [Infrastructure Needed (Optional)](#infrastructure-needed-optional)
- [Future Work](#future-work)
<!-- /toc -->

## Release Signoff Checklist

Items marked with (R) are required *prior to targeting to a milestone / release*.

- [x] (R) Enhancement issue in release milestone, which links to KEP dir in [kubernetes/enhancements] (not the initial KEP PR)
- [ ] (R) KEP approvers have approved the KEP status as `implementable`
- [x] (R) Design details are appropriately documented
- [x] (R) Test plan is in place, giving consideration to SIG Architecture and SIG Testing input (including test refactors)
    - [ ] e2e Tests for all Beta API Operations (endpoints)
    - [ ] (R) Ensure GA e2e tests meet requirements for [Conformance Tests](https://github.com/kubernetes/community/blob/master/contributors/devel/sig-architecture/conformance-tests.md)
    - [ ] (R) Minimum Two Week Window for GA e2e tests to prove flake free
- [x] (R) Graduation criteria is in place
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

## Glossary

* **In-Place Resource Resize:** Dynamically increasing or decreasing compute resources (CPU, Memory, and HugePages) on a live, `Ready` node via a declarative API, with the Kubelet reconciling the change in place. Swap is explicitly out of scope for Alpha — see Non-Goals.
* **Node Compute Resource:** CPU, Memory, and HugePages. Swap is deferred to a later milestone.
* **Physical Capacity:** The raw hardware capacity of a node as reported by the integrated `cAdvisor` subsystem, reflecting the true underlying machine resources (e.g., number of CPU cores).
* **Configured Capacity:** The desired logical capacity declared by an administrator or external controller via `Node.Spec.ConfiguredCapacity`. This is the Kubelet's target and may be less than the Physical Capacity (e.g., to under-report resources intentionally). For Alpha, Configured Capacity must not exceed Physical Capacity.
* **CapacityConfigured Condition:** A `Node.Status.Condition` of type `CapacityConfigured` that the Kubelet uses to expose the current reconciliation state of a capacity resize request. Possible reasons are `Accepted`, `InProgress`, `Infeasible`, and `EmergencyReduced`.
* **Kubelet-Restart Workaround:** The pre-existing, informal practice of restarting the Kubelet after a hardware change so that it re-reads physical capacity from cAdvisor at boot time. This is **not** a supported or safe mechanism for mutating node capacity — it is a workaround with significant drawbacks (downtime, edge-case bugs, no ecosystem coordination).

## Summary

This proposal facilitates dynamic native resource resizing (increases and decreases in capacity) on a node to streamline cluster capacity updates, offering a seamless alternative to adding or removing nodes from an existing cluster. The revised node configurations automatically propagate at both the node and cluster levels.

Today, mutating `Node.Status.Capacity` is not a supported or safe operation in Kubernetes. The Kubernetes ecosystem (Scheduler, Cluster Autoscaler, VPA, Quota system) has never been explicitly designed or tested to tolerate a live change to a Node's capacity values. This KEP addresses the problem in two ordered layers:

**Layer 1 — Ecosystem Tolerance:** Formally define the contract that the broader cluster ecosystem must honour when `Node.Status.Capacity` changes. Establish the API-server mutations, Scheduler cache-invalidation tests, Cluster Autoscaler behaviour validation, and Quota/VPA integration points that make a capacity change safe to propagate — even if, as a first step, that change is initiated by a Kubelet restart. This is the scope of the v1.38 Alpha milestone.

**Layer 2 — Declarative Capacity Actuation:** Introduce `Node.Spec.ConfiguredCapacity`, the declarative API field that allows external controllers to trigger a capacity change. Implement the Kubelet reconciliation loop that updates cgroups, re-initialises sub-managers (CPU Manager, Memory Manager), recalculates container swap limits via the CRI, and synchronises `Node.Status` — all on the running node. The `CapacityConfigured` Node Condition disseminates the live reconciliation state to external orchestrators. This layer is fully described in this KEP and targets a subsequent milestone (v1.39 Alpha or later), pending resolution of the open design questions documented in [Open Questions for Layer 2](#open-questions-for-layer-2).

The full design of Layer 2 — including the `Node.Spec.ConfiguredCapacity` API field, the Kubelet reconciliation loop, and the `CapacityConfigured` condition — is described in this document so that the community can evaluate the complete architecture. However, no Layer 2 code is introduced in v1.38; the initial milestone is deliberately scoped to proving the ecosystem foundation is safe, using the existing Kubelet-restart-on-resized-hardware path as the trigger.

## Motivation

### The Problem: Live Mutation of Node Capacity Is Unsupported Today

Performing a live mutation of `Node.Status.Capacity` on a running node is not a currently supported operation in Kubernetes. A node's resource capacity values (`Node.Status.Capacity`, `Node.Status.Allocatable`) are written by the Kubelet during bootstrap and then held fixed for the entire lifetime of that Kubelet process. When the Kubelet restarts, it re-reads hardware capacity from cAdvisor and rewrites these fields — so a restart *does* cause the values to change. However, this is a full process restart that reinitialises the Kubelet from scratch, not a live mutation of a running node's capacity. No existing API validation, Scheduler logic, Cluster Autoscaler heuristic, VPA algorithm, or ResourceQuota controller has been designed or tested against the assumption that these fields can change for a live, `Ready` node without a Kubelet restart.

The only known workaround today is to restart the Kubelet after a physical hardware change. However, this workaround:

1. **Does not safely mutate capacity on a live node** — the Kubelet reinitialises entirely from scratch with no coordination with the Scheduler, Cluster Autoscaler, or Quota system before or after the change.
2. **Is not a first-class supported operation** — there are no established best practices for coupling a Kubelet restart to a cloud provider or hypervisor API call.
3. **Carries significant operational risk** — it introduces temporary node `NotReady` states, breaks active `exec`/`port-forward` sessions, and has a documented history of triggering edge-case bugs:
    - https://github.com/kubernetes/kubernetes/issues/109595
    - https://github.com/kubernetes/kubernetes/issues/119645
    - https://github.com/kubernetes/kubernetes/issues/125579
    - https://github.com/kubernetes/kubernetes/issues/127793

In summary: a Kubelet restart is a workaround for the absence of a supported capacity-mutation path, not a safe or reliable mechanism for it. This KEP introduces that supported path.

### The Need: Dynamic Hardware Requires a Dynamic Capacity Model

Modern hypervisors, cloud providers, and kernel capabilities enable the dynamic hot-plugging and hot-unplugging of native resources such as CPU and Memory (e.g., [CPU Hotplug](https://docs.kernel.org/core-api/cpu_hotplug.html), [Memory Hotplug](https://docs.kernel.org/core-api/memory-hotplug.html)), and Ephemeral Storage block devices. Without a supported capacity-mutation path, these hardware events lead to two severe failure modes:

- **Cluster-Level Starvation (Upscaling):** If capacity is added, the Kubernetes Scheduler and Cluster Autoscaler remain blind to it, rendering the new hardware useless for pending workloads.

- **Node-Level Instability (Downscaling):** If capacity is removed, the Kubelet's stale top-level cgroups, container swap limits, and Eviction Manager thresholds do not adjust. Because the Kubelet assumes the resources still exist, it fails to evict pods defensively, causing the host's Linux kernel to invoke the OOM killer or exhaust the disk, violently terminating processes and potentially crashing the node.

### The Two-Layer Solution

This KEP decomposes the solution into two ordered, independently valuable layers:

#### Layer 1: Making the Ecosystem Safe for Mutable Capacity

Before any Kubelet actuation can be trusted, the broader cluster ecosystem must be formally validated and — where necessary — updated to tolerate `Node.Status.Capacity` mutations safely. This layer uses the Kubelet restart as its initial boundary: it answers the question *"If the capacity field changes, even as a result of a Kubelet restart on resized hardware, are the API Server, Scheduler, Cluster Autoscaler, VPA, and Quota system safe?"*

The ecosystem contracts that must be established:

- **API Server:** `Node.Status.Capacity` and `Node.Status.Allocatable` must be patchable on a live `Ready` node without triggering systemic webhook rejections or validation failures. This behaviour has never been formally tested or documented.
- **Scheduler:** The Scheduler's internal node cache must correctly invalidate and update its per-node resource view when it receives a `Node` UPDATE event with changed capacity fields. It must not rely on a stale boot-time snapshot.
- **Cluster Autoscaler (CA):** The CA uses existing nodes as templates for provisioning new nodes within the same NodeGroup. If capacity is mutable, the CA must be given a stable, immutable baseline (via a boot-time annotation) to template from, rather than reading the live, potentially-resized `Node.Status.Capacity`.
- **VPA (Vertical Pod Autoscaler):** The VPA's node capacity model must be tolerant of live capacity changes so that it does not issue recommendations that exceed the new node bounds.
- **ResourceQuota:** Namespace-level ResourceQuota controllers operate against allocatable node capacity. A capacity change must not silently bypass quota enforcement.

#### Layer 2: Declarative Capacity Actuation

Once the ecosystem is safe (Layer 1), the second layer provides the supported, declarative mechanism for capacity actuation on a running node. This layer introduces:

- **`Node.Spec.ConfiguredCapacity`:** A new declarative API field allowing external controllers or administrators to declare the desired logical capacity without touching the node or its Kubelet process.
- **Kubelet Reconciliation Loop:** The Kubelet watches for changes to `Node.Spec.ConfiguredCapacity` and for physical hardware drift detected by cAdvisor. It validates the desired value against physical bounds, then actuates the change by updating internal cgroup hierarchies, re-initialising sub-managers (CPU Manager, Memory Manager, Eviction Manager), and recalculating container swap limits via the CRI — all in a running Kubelet.
- **`CapacityConfigured` Node Condition:** A new condition that exposes the live reconciliation state (`Accepted`, `InProgress`, `Infeasible`, `EmergencyReduced`) to external orchestrators, enabling safe, signal-driven coordination for downscale workflows.

### Why a Declarative API Is Essential for Safe Downscaling

Without an API-driven trigger, cluster administrators have no safe way to coordinate a hot-unplug operation. The desired workflow for a graceful downscale is:

1. An external controller invokes the API (`Node.Spec.ConfiguredCapacity`) to reduce logical node capacity.
2. The Kubelet actuates that reduction (graceful eviction, updates cgroups, updates `Node.Status`).
3. The external controller observes the Kubelet's completion signal (`CapacityConfigured: Accepted`).
4. The external controller proceeds to physically remove hardware via the hypervisor.

This coordination is impossible with a purely reactive, hardware-first model. If the hypervisor forcefully reclaims RAM (e.g., balloon deflation) without prior API coordination, the kernel OOM killer may fire before the Kubelet can react. By making the API the primary trigger, operators gain deterministic control over the timing and safety of capacity removal.

### Benefits Beyond the Workaround

Enabling the Kubelet to dynamically detect and adapt to underlying capacity changes mitigates manual administrative toil and unlocks several distinct advantages over the current Kubelet-restart workaround:

- **Control Plane Efficiency:** Managing resource demands by scaling existing nodes in-place brings significantly less overhead to the control plane compared to provisioning and joining entirely new nodes.
- **Speed to Delivery:** Expanding the capabilities of current nodes is considerably more time-efficient than the procedure of establishing new virtual machines.
- **Network Optimization:** Improved inter-pod network latencies, as inter-node traffic is reduced when more pods can be hosted locally on a single scaled-up node.
- **Stability:** Avoids the historical bugs and disruption risks associated with forced Kubelet restarts.

Implementing this KEP will empower nodes to recognize and adapt to changes in their native configurations instantly, facilitating the safe, efficient, and uninterrupted deployment of workloads.

### Goals

* API Synchronization: Update Node API Capacity and Allocatable fields dynamically on a live, `Ready` node via the declarative `Node.Spec.ConfiguredCapacity` API field.

* Component Sync: Re-initialize internal Kubelet managers (CPU, Memory, Eviction) to safely align with the altered hardware capacity.

* Cgroup Enforcement: Update the host's top-level /kubepods and QoS cgroup boundaries to physically enforce the resized limits.

* Configured Capacity: Allow the logical capacity of a node to be dynamically configured via `Node.Spec.ConfiguredCapacity`, decoupling the cluster's view of the node from strict physical hardware events. Both hardware-triggered and purely configuration-driven changes (e.g., under-reporting a 32Gi machine as 20Gi) are in-scope.

* Bootstrap Parity: Upon Kubelet restart, the Kubelet reads `Node.Spec.ConfiguredCapacity` as its primary target before falling back to raw cAdvisor hardware discovery. This ensures that a Kubelet restart on a node with an existing `ConfiguredCapacity` spec behaves identically to a live-resize event.

### Non-Goals

* Reserved Adjustments: Dynamically changing --system-reserved and --kube-reserved values (these remain static from bootstrap).

* Infrastructure Orchestration: Executing the physical hardware hot-plug or updating the autoscaler to trigger it.

* Workload Re-balancing: Automatically migrating or re-balancing existing workloads across the cluster to utilize the new space.

* NRI Plugins: Propagating host resource changes to external Node Resource Interface (NRI) plugins.

* OOM Score Updates: Dynamically rewriting oom_score_adj for running processes, due to severe latency and race condition risks.

* Pod Resizing: Dynamically resizing individual Pod resource requests and limits (covered independently by KEP-1287).

* Swap-Enabled Nodes: In-place node resize is not supported on nodes with Swap enabled for the Alpha phase. The interaction between node capacity resize and per-container swap limit recalculation is non-trivial, and there is ongoing work to align the swap semantics between node resize and pod resize (KEP-1287). To avoid compounding those open questions, resize operations on swap-enabled nodes will be rejected or skipped until a later milestone. This will be revisited in a subsequent Alpha or Beta update.

* Node Capacity Overcommit: Configuring the Kubelet to report a logical capacity to the API Server that exceeds the raw, physical underlying hardware capacity (e.g., reporting 48Gi on a 32Gi machine relying on swap). For the Alpha phase, `ConfiguredCapacity` is strictly bounded by physical reality (CPU and Memory). This will be explored in Future Work.

* Admission Webhook on Node.Spec: This KEP does not introduce a new admission webhook specifically for capacity changes. Standard Kubernetes `ValidatingWebhookConfiguration` and `MutatingWebhookConfiguration` can be deployed by cluster administrators to intercept mutations to `Node.Spec.ConfiguredCapacity` without any KEP-specific mechanism.

## Proposal

This KEP introduces a declarative, event-driven reconciliation architecture to handle native resource reconfiguration safely. It is structured as two ordered layers that build on each other, with each layer being independently valuable and reviewable.

### Layer 1: Ecosystem Tolerance for Mutable Node Capacity

This layer addresses the foundational question that has never been formally answered: can the Kubernetes control plane safely tolerate a change to `Node.Status.Capacity`? Regardless of whether that change originates from a Kubelet restart on resized hardware, a manual patch, or the declarative reconciliation loop introduced in Layer 2, every downstream component must behave correctly.
The requirements for Layer 1 are:


1. **API Server — Capacity Field Mutability:** Formally validate that `Node.Status.Capacity` and `Node.Status.Allocatable` are patchable on a live, `Ready` Node object. Confirm that no system webhook, validation rule, or strategy-merge logic prevents this mutation. This is the unspoken prerequisite that all subsequent work depends on.

2. **Scheduler — Node Cache Invalidation:** Confirm that the Scheduler's internal `NodeInfo` cache is invalidated and refreshed when it receives a `Node` UPDATE event with changed capacity fields. The Scheduler must not rely on a snapshot taken at node registration time. This is validated by: (a) upscaling a node's capacity and verifying a previously unschedulable pod becomes schedulable, and (b) downscaling a node's capacity and verifying the Scheduler correctly rejects pods that no longer fit.

   **Scheduler Capacity View During Downscale:** There is an inherent race condition where the Scheduler schedules a pod to a node concurrently with an in-progress downscale: the Scheduler's view of the node may still reflect the pre-downscale capacity, but by the time the pod reaches Kubelet admission, the Kubelet's capacity has already been reduced — causing Kubelet to reject the pod and leave it in a `Failed` state. This is not a new problem (it is structurally identical to a node going `NotReady` between scheduling and binding), but its window can be minimised by having the Scheduler treat the effective node capacity as `min(Node.Spec.ConfiguredCapacity, Node.Status.Capacity)`. Using the minimum means that as soon as an operator signals a downscale via `Node.Spec.ConfiguredCapacity`, the Scheduler conservatively stops over-committing to the node even before the Kubelet has finished reconciling `Node.Status`. This mirrors the analogous treatment for in-place pod resize, where the Scheduler uses `max(pod.spec.resources, pod.status.resources)` to avoid over-committing against a pod whose resources are still being expanded. This strategy minimises the race window but does not eliminate it entirely; Kubelet admission remains the authoritative gate and the final safety net.

   **Preemption Grace Period Race:** A more specific variant of the above concerns the Scheduler's preemption path. When a high-priority pod arrives and the Scheduler determines it fits on a node only after preempting a lower-priority victim, the victim enters its termination grace period. If node capacity decreases during that grace period, the Scheduler's original preemption calculation becomes stale: the capacity it assumed the high-priority pod would land on no longer exists. The Scheduler must detect this and re-run preemption calculations for the waiting pod against the updated node capacity. The correct mechanism is to add or update a Node UPDATE queueing hint in the Scheduler that re-queues pods in the `WaitingForPreemption` state when `Node.Status.Allocatable` decreases on the node they are targeting. Without this hint, the high-priority pod may attempt to bind to an insufficient node — again falling back to Kubelet admission rejection — or may wait indefinitely for space that now cannot materialise. Determining whether this queueing hint already exists, and adding it if not, is part of the Layer 1 Scheduler contract validation.

3. **Cluster Autoscaler — Stable Provisioning Template:** The CA currently uses an existing node's `Node.Status.Capacity` as the template for provisioning new nodes in the same NodeGroup. With mutable capacity, a dynamically resized node must not corrupt this template. The Layer 1 deliverable for this item is to document the current CA behaviour and identify the gap — not to ship a fix. The correct fix is an open design question: placing a static boot-time value on the Node object (whether as an annotation or a new status field) is problematic because `Node.Status` is meant to reflect live state, annotations are brittle when multiple actors read them, and the edge case where an entire NodeGroup has been uniformly resized means the "initial" capacity is no longer functionally accurate. Alternative approaches — such as having CA read directly from the cloud provider's launch template, or introducing a configurable reference capacity within CA's own NodeGroup configuration — are being considered. This question is tracked as an open design item and must be resolved before this KEP's CA integration is considered complete.

4. **VPA — Recommendation Bounds:** Confirm that the Vertical Pod Autoscaler's node-capacity model is re-read on Node UPDATE events. VPA must not issue container resource recommendations that exceed the new node allocatable bounds.

5. **ResourceQuota — Allocatable Accounting:** Confirm that namespace-level ResourceQuota admission does not cache the node's allocatable value at pod admission time in a way that silently bypasses quota limits after a capacity change.

6. **In-Place Pod Resize Interaction (KEP-1287):** Node capacity changes interact directly with the `Deferred` and `Infeasible` resize states defined by In-Place Pod Vertical Scaling (KEP-1287). Three cases must be explicitly handled:

   - **Upscale → retry `Deferred` pod resizes:** When `Node.Status.Allocatable` increases, pod resize requests that were previously marked `Deferred` (because insufficient node capacity prevented the Kubelet from accepting them) must be re-evaluated. The Kubelet should attempt the deferred resize against the new, larger allocatable values. This mirrors the existing behaviour where a pod resize that is `Deferred` due to resource contention is retried when other pods are evicted and space becomes available — a node capacity upscale is the same signal at a different granularity.

   - **Downscale → `Deferred` becomes `Infeasible`:** If a pod resize was previously `Deferred` (waiting for capacity to become available) and a subsequent node downscale reduces allocatable resources to a point where the desired pod resources can *never* fit, the resize status must transition from `Deferred` to `Infeasible`. Leaving a resize in `Deferred` state against a node that is now provably too small is misleading and prevents the Scheduler and VPA from taking corrective action.

   - **`Infeasible` pod resize cleared by upscale:** Since Kubernetes 1.36, the API server rejects pod resize requests that exceed node capacity at admission. However, the Kubelet can independently mark a resize `Infeasible` when capacity-bound constraints are evaluated at the node level. When `Node.Status.Allocatable` subsequently increases (via a node upscale), previously `Infeasible` pod resize requests that were blocked solely due to node capacity must be re-evaluated and, if now feasible, transitioned out of the `Infeasible` state. Without this re-evaluation, a node upscale would not unblock any waiting workloads. The Kubelet needs a mechanism to track the *reason* a resize was marked `Infeasible` (node-capacity-bound vs. other reasons) in order to know which requests to retry.

   The Scheduler and VPA both react to `Deferred` and `Infeasible` pod resize statuses: the Scheduler may attempt to relocate a pod whose resize is `Infeasible` on its current node, and VPA may lower its recommendation if a resize is persistently `Infeasible`. A node capacity change can therefore trigger a cascade across all three components — node resize → pod resize state update → Scheduler/VPA re-evaluation. This cascade must be documented, and the component interactions validated, as part of the Layer 1 ecosystem contracts.

Layer 1 establishes the baseline: documenting the current ecosystem behaviour when `Node.Status.Capacity` changes and identifying any gaps. Where gaps exist, they are addressed as part of this KEP — either by fixing the relevant component or by adding the missing contract. Layer 2 then builds the supported actuation mechanism on top of that validated foundation.

### Layer 2: Declarative Capacity Actuation via `Node.Spec.ConfiguredCapacity` *(targeted for v1.39 Alpha)*

> **Scope note:** Layer 2 is described here for completeness and community review. It is not part of the v1.38 Alpha scope. Implementation begins once the Layer 1 ecosystem contracts are validated and the open design questions in [Open Questions for Layer 2](#open-questions-for-layer-2) are resolved.

With the ecosystem contracts from Layer 1 established, Layer 2 provides the supported, first-class mechanism for capacity actuation. It is structured as three implementation steps that build incrementally:

**Step 1 — Baseline API & Scheduler Validation (pre-requisite tests):** Add the upstream test coverage described in Layer 1 before any dynamic trigger code is written. These tests form the safety net for all subsequent changes.

**Step 2 — Unified Kubelet Reconciliation:** Introduce `Node.Spec.ConfiguredCapacity` and update the Kubelet to reconcile the difference between what `Node.Status` reflects and what the Kubelet observes from cAdvisor — covering both the live-node case and the Kubelet-restart-on-resized-hardware case. The Kubelet updates internal cgroup hierarchies, re-initialises sub-managers (CPU Manager, Memory Manager, Eviction Manager), and recalculates container swap limits via the CRI. `Node.Status.Capacity` is the output of this reconciliation — written by the Kubelet after validation, not used as an input.

**Step 3 — Hardware-Drift Trigger:** Implement an automated cAdvisor polling trigger that fires the reconciliation loop when physical capacity changes. The loop re-validates `Node.Spec.ConfiguredCapacity` against the new physical bounds and, only upon successful validation, writes the resolved capacity to `Node.Status`. This step extends Step 2 to handle emergency hardware-removal events (Path B) where no prior API signal was sent.

By separating these two layers, the KEP ensures that the requirements for making capacity mutable at the cluster level are addressed before the declarative reconciliation mechanism is built on top of them.

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

* **Linux and cgroup v2:** This feature targets Linux nodes running cgroup v2. cgroup v1 nodes are not supported.

* **Swap-Enabled Nodes Not Supported (Alpha):** In-place node resize is not supported on nodes with Swap enabled in the Alpha phase. If Swap is active on the node, the Kubelet will decline to perform a resize and will set the `CapacityConfigured` condition to `False` (Reason: `Infeasible`) with a message indicating swap is unsupported. This restriction will be revisited in a subsequent milestone once the swap–pod-resize interaction is resolved.

* **Linux Only:** This feature has no effect on Windows nodes. The Kubelet's capacity reconciliation loop short-circuits immediately on non-Linux platforms.

* **NUMA Topology Lazy Reconciliation:** When a resize changes the available memory or CPU cores per NUMA zone, the Topology Manager's view of NUMA boundaries is updated in its internal state machine. However, running pods retain their original NUMA pinning — they are not remapped mid-flight. Only newly admitted pods use the updated NUMA layout. Operators should account for this when sizing a downscale target on NUMA-pinned workloads.

### Risks and Mitigations

1. #### OOMScoreAdjust Drift for Existing Pods

    **Risk**: The Kubelet calculates a container's `oom_score_adj` upon creation using the formula: `1000 - (1000 * containerMemoryRequest) / nodeMemoryCapacity`. 
If a node's memory capacity changes, the OOM scores of existing Burstable pods will mathematically drift compared to newly scheduled Burstable pods, potentially skewing the Linux OOM killer's tie-breaker logic.

    **Mitigation**: We explicitly accept this minor drift. Updating `oom_score_adj` for running containers requires identifying and rewriting the `/proc/<PID>/oom_score_adj` file for every single running thread inside the container. 
This introduces severe latency, high CPU overhead, and dangerous race conditions. The overarching QoS hierarchy (Guaranteed pods remain invincible, BestEffort pods remain first-to-die) is strictly preserved, making the risk of a slightly skewed Burstable tie-breaker acceptable compared to the danger of rewriting thousands of running PIDs.

2. #### Container Swap Limit Re-calculation Overhead

   **Risk**: The proportional swap limit for a container relies on the node's total memory capacity. Upon a resize, failing to update this leads to stranded swap space (upscale) or immediate kernel panics (downscale). Recalculating and applying this to all active pods also introduces CRI overhead.

   **Mitigation**: For the Alpha phase, this risk is eliminated entirely by not supporting resize on swap-enabled nodes (see Non-Goals). The Kubelet will decline to perform a resize if Swap is active, avoiding both the correctness and overhead concerns. The correct approach for swap recalculation — including aligning the semantics with pod resize (KEP-1287) — is deferred to a subsequent milestone.

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

A new declarative structure is introduced to the NodeSpec API, allowing external controllers or administrators to declare the desired logical capacity target for the node.

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

**Reason Infeasible**: The requested `ConfiguredCapacity` exceeds physical hardware limits (overcommit is disallowed in Alpha). The Kubelet has clamped the target to the physical limits, set this condition, and will not retry the oversized request. This provides explicit `lockSize` semantics: the Kubelet treats the `Infeasible` state as terminal for the current Spec value. An external controller or administrator must patch `Node.Spec.ConfiguredCapacity` to a valid, in-bounds value to resume normal reconciliation. A dedicated `lockSize` boolean field on `NodeSpec` was considered but rejected in favour of this condition reason — it provides the same terminal semantics without adding a new API field, and remains observable via standard `kubectl get node` condition output.

**Reason EmergencyReduced**: Physical hardware was forcefully removed (e.g., hypervisor-forced reclaim), falling below the current `Node.Spec.ConfiguredCapacity`. The Kubelet bypassed the API and clamped the node to the new physical reality to protect the kernel. Normal reconciliation is suspended until an external actor patches the Spec down to match the new physical bounds.


### Admission Control Contract

**Source of Truth:**
The `Node.Spec.ConfiguredCapacity` field is the authoritative declaration of desired logical capacity. `Node.Status.Capacity` and `Node.Status.Allocatable` are read-only outputs of the Kubelet's reconciliation — no external controller or webhook should patch them directly. The Kubelet is the sole component that writes to `Node.Status.Capacity`. This separation of Spec from Status follows the standard Kubernetes controller pattern.

**Path A (Orchestrated):** External controllers or administrators patch `Node.Spec.ConfiguredCapacity`. This mutation is intercepted by the cluster's standard Validating and Mutating Webhooks. If a webhook rejects the resize request, the API Server denies the PATCH, and the Kubelet's informer never receives the event — the node's effective capacity does not change.

**Path B (Emergency):** When physical hardware is forcefully reclaimed (e.g., hypervisor-forced deflation), the Kubelet reacts to cAdvisor directly and patches `Node.Status.Capacity` and `Node.Status.Conditions`. This path utilizes the standard Node Authorizer RBAC, bypassing Spec webhooks to ensure the control plane is immediately notified of physical degradation without needing an external controller to be available.

**Bootstrap Behavior:** Upon restart, the Kubelet reads `Node.Spec.ConfiguredCapacity` from the API Server as the primary capacity target *before* reading the raw cAdvisor hardware data. If a valid `ConfiguredCapacity` exists in the Spec, the Kubelet treats it as the desired state and validates it against live physical hardware. This ensures that a Kubelet restart on a pre-configured node does not accidentally override the declared configuration.

### Resource-Specific Validation Rules

`Node.Spec.ConfiguredCapacity` is a `ResourceList` — a map of resource name to quantity — and is entirely optional. Users are not required to set all resource types, or any at all:

- Field absent (`omitempty`): The Kubelet behaves as today — it uses the raw cAdvisor-reported physical capacity for all resources and no reconciliation loop is started.
- Field present, resource key absent: If `ConfiguredCapacity` is set but does not include a particular resource (e.g., CPU is present but memory is not), the Kubelet treats the absent resource as having no declared target and continues to use the cAdvisor-reported physical capacity for that resource. Reconciliation for the absent resource is a no-op.
- Field present, resource key set to zero: A zero value is treated as a malformed signal and rejected — the Kubelet sets `CapacityConfigured` to `False` (Reason: `Infeasible`) and does not actuate the resize for any resource in that request.

`ConfiguredCapacity` is applied per-resource independently: setting a value for CPU does not implicitly affect the declared or effective capacity for memory, hugepages, or any other resource. Each key in the map is validated and reconciled in isolation.

The Alpha constraint (`ConfiguredCapacity <= Physical Capacity`) applies per-resource. The Kubelet validates the Spec against the host using the following resource-specific rules:

**CPU & Memory:** Strictly bounded by the physical hardware limits reported by cAdvisor. The `ConfiguredCapacity` for these resources must not exceed the raw physical quantity.

**Hugepages:** Validated against the pre-allocated hugepage pools configured at the OS kernel level (e.g., via `/sys/kernel/mm/hugepages`), not the total raw memory.

**Ephemeral Storage:** Validated against the available disk capacity as reported by the host OS. The `ConfiguredCapacity` for `ephemeral-storage` must not exceed the actual available disk space on the node's root filesystem.

**Unknown resource types:** Any resource type present in `ConfiguredCapacity` that the Kubelet does not recognise (e.g., custom extended resources) is silently ignored by the validation loop. Only well-known resource types (CPU, Memory, Hugepages, Ephemeral Storage) are validated and actioned in Alpha.

Note on swap-enabled nodes: Swap itself is not a key in `ConfiguredCapacity` and is never set by users. The constraint is more subtle: when a node has Swap enabled and memory is resized, each running container's per-container swap limit (`memory.swap.max`) must be recalculated — it is derived proportionally from the ratio of the container's memory request to total node memory, multiplied by total available node swap. Changing node memory without updating these derived per-container limits leads to stranded swap space or kernel panics. In Alpha, memory resize on swap-enabled nodes is therefore not supported: if the node has Swap active and a memory value is present in `ConfiguredCapacity`, the Kubelet sets `CapacityConfigured` to `False` (Reason: `Infeasible`) and does not actuate the resize. The correct recalculation semantics are deferred to a subsequent milestone.

### Security Considerations

This section explicitly addresses the security properties of this feature in response to the concern that a compromised node could over-report its capacity to the control plane in order to attract Pod scheduling and gain access to secrets it should not receive.

**Capacity Inflation Attack is Prevented by Design (Alpha):**
The Alpha enforcement rule (`ConfiguredCapacity <= Physical Capacity`) is the primary defense. The Kubelet's `calculateValidatedCapacity()` function enforces this bound locally by reading raw capacity from `cAdvisor`, which reads directly from the host kernel (`/sys`, `/proc`). For a node to successfully inflate its reported capacity above its physical reality, an attacker would need to compromise either:
1. The Kubelet binary itself, or
2. The kernel-level data sources that `cAdvisor` reads.

Both represent a full node compromise, which is already outside the Kubernetes threat model. A cluster-level actor patching `Node.Spec.ConfiguredCapacity` to an inflated value will have that spec clamped by the Kubelet and the `CapacityConfigured` condition set to `False (Reason: Infeasible)` — the API Server will store the spec, but the Kubelet will not act on it.

Node Authorizer and write access: Two distinct access-control mechanisms apply to `Node.Spec.ConfiguredCapacity` depending on the identity of the writer.

For Kubelet identities, the standard Node Authorizer enforces that each Kubelet may only write to the Node object that represents itself. This prevents a compromised node from patching `ConfiguredCapacity` on a neighbouring node — a Kubelet attempting to modify another node's spec will be denied at the API Server by the Node Authorizer before the request reaches any webhook or admission plugin.

For non-Kubelet identities (external controllers, cluster administrators, automation), the Node Authorizer is not involved. Access is governed by standard RBAC: any identity with `update` or `patch` permission on the `nodes` resource can write `ConfiguredCapacity`. Cluster administrators who want to restrict which specific controllers are permitted to set this field can deploy a `ValidatingWebhookConfiguration` targeting `Node.Spec.ConfiguredCapacity` mutations.

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

**4. State Override & Alerting:** The Kubelet calculates the clamped target and patches `Node.Status.Capacity`. It transitions the `CapacityConfigured` Condition to `False` (Reason: `EmergencyReduced`) and emits a `Warning` Event (`EmergencyCapacityReduced`). To prevent infinite API loops, the Kubelet caches this clamped target locally and safely ignores the oversized Spec until the Spec is updated by an external actor to match or fall below the new physical reality.

**5. Divergence Resolution:** The external controller watches for the `EmergencyReduced` condition. Upon seeing it, the controller is responsible for patching `Node.Spec.ConfiguredCapacity` down to match reality, which clears the split-brain state and resumes normal Kubelet reconciliation behavior.

#### Flow Control: Container Swap Limit Recalculation

> **Alpha scope note:** Resize on swap-enabled nodes is not supported in the Alpha phase (see Non-Goals). This section describes the intended design for a future milestone when swap support is introduced.

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

**Note on Static Pods:** Static pods are managed directly by the Kubelet via local manifest files and are not subject to eviction manager decisions. They will not be evicted during a capacity downscale. Cluster operators must manually account for the resource footprint of any static pods when setting a downscale target in `Node.Spec.ConfiguredCapacity`, ensuring the declared target leaves sufficient headroom above the aggregate static pod requests.

#### Compatibility with Cluster Autoscaler

The Cluster Autoscaler (CA) presently anticipates uniform allocatable values among nodes within the same NodeGroup, using existing nodes as templates for newly provisioned nodes. With mutable node capacity, nodes within a single group may drift in size over time, which can cause the CA to select a resized node as its provisioning template — causing it to expect new nodes to have the larger capacity, while the cloud provider provisions base-sized nodes, leading to scheduling failures.

This is an open design problem. Placing a static boot-time value on the Node object (as an annotation or a dedicated status field) has limitations: `Node.Status` is intended to reflect live state rather than historical boot state; annotations are brittle when multiple controllers read and react to them; and if every node in a NodeGroup has been uniformly resized, the "initial" capacity is no longer a meaningful reference — the NodeGroup has functionally changed size and operators may reasonably expect that to be reflected.

Alternative approaches include having CA read directly from the cloud provider's launch template (which reflects the true provisioning baseline) or introducing a configurable reference capacity within CA's own NodeGroup configuration. Both decouple the problem from the Kubernetes Node object and place it where it belongs — in the component that understands provisioning semantics.

The Layer 1 deliverable for CA compatibility is to document the current behaviour and the failure mode, and validate that CA correctly observes `Node.Status.Capacity` UPDATE events when capacity changes. The right long-term fix will be agreed upon before any CA integration is standardised in this KEP.

### Layer 1 Implementation: Ecosystem Tolerance — Validating That Mutable Capacity Is Safe

This section implements the Layer 1 contracts described in the Proposal. It formally documents the static-capacity assumptions that Kubernetes currently holds, and adds the upstream test coverage that proves the ecosystem can safely handle a `Node.Status.Capacity` mutation — whether that mutation arrives via a Kubelet restart or via the Layer 2 reconciliation loop.

#### Assumptions Being Changed
1. **Static Capacity Assumption:** `Node.Status.Capacity` and `Allocatable` are currently fixed for the lifetime of a running Kubelet process — they are written at boot and held constant until the next restart. This KEP transitions them to fields that can be mutated on a live, `Ready` node via the declarative reconciliation loop. That live-mutation path has never been formally tested or supported by the ecosystem.
2. **Kubelet Restart Admission:** Currently, if a Kubelet is restarted on a machine whose hardware was reduced while offline, the Kubelet may blindly fail pod admission. We are shifting this to a graceful reconciliation and eviction model, whether the trigger is a restart or the live reconciliation loop.
3. **Autoscaler Homogeneity:** Nodes within a single NodeGroup will no longer be guaranteed to have identical capacities, meaning the Autoscaler cannot blindly select any node as a provisioning template.
4. **External Controller Caches:** Third-party operators that cache node sizes indefinitely will become stale. (This is an accepted operational constraint).

#### Layer 1 Pre-requisite Tests

These tests prove the foundational ecosystem contracts hold. They are deliberately scoped to raw API and control-plane behaviour — no Kubelet reconciliation code is exercised. Passing these tests is the gate for beginning Layer 2 implementation.

* **Test 1: API Server Mutation Acceptance**
    * *Action:* Manually patch `.status.capacity` and `.status.allocatable` on a `Ready` Node object.
    * *Validation:* Verify the API Server accepts the patch without systemic webhook rejections or validation failures.
    * *Layer:* Layer 1 — API Server contract.

* **Test 2: Scheduler Cache Invalidation (Upscale)**
    * *Action:* Create a pending Pod that requires 8Gi of memory on a cluster where the only node has 4Gi. Manually patch the Node object's capacity to 10Gi.
    * *Validation:* Verify the Scheduler detects the mutated Node object, updates its internal cache, and successfully schedules the pending Pod.
    * *Layer:* Layer 1 — Scheduler contract.

* **Test 3: Scheduler Cache Invalidation (Downscale)**
    * *Action:* Manually patch an empty Node's capacity from 10Gi down to 4Gi. Attempt to schedule a Pod requiring 8Gi.
    * *Validation:* Verify the Scheduler respects the mutated smaller capacity and rejects the Pod (leaves it Pending), proving it does not rely on a stale boot-time cache.
    * *Layer:* Layer 1 — Scheduler contract.

* **Test 4: Kubelet Restart on Resized Hardware (Workaround Boundary)**
    * *Action:* Schedule a Pod. Stop the Kubelet. Mock the underlying machine info to reflect a smaller capacity (simulate offline hot-unplug). Start the Kubelet.
    * *Validation:* Verify the Kubelet boots successfully, recognizes the discrepancy between the API and physical hardware, and handles the change gracefully (e.g., evicting the pod if starved) rather than crashing or permanently locking pod admission.
    * *Layer:* Layer 1 — establishes the Kubelet-restart workaround boundary: the minimum safe behaviour that Layer 2 must meet or exceed.

* **Test 5: Deferred Pod Resize Retried on Node Upscale**
    * *Action:* On a node with 4Gi allocatable memory, submit a pod with a pending resize request to 3.5Gi (which is `Deferred` because the node is fully packed by other pods). Manually patch `Node.Status.Allocatable` upward to 8Gi, freeing headroom.
    * *Validation:* Verify the Kubelet re-evaluates the previously `Deferred` resize and transitions it to `InProgress` / `Accepted` now that sufficient allocatable capacity exists. Verify the pod's `resize` status condition reflects the updated state.
    * *Layer:* Layer 1 — KEP-1287 interaction contract (upscale path).

* **Test 6: Deferred Pod Resize Transitions to Infeasible on Node Downscale**
    * *Action:* On a node with 8Gi allocatable memory, submit a pod with a pending resize request to 6Gi (marked `Deferred` due to contention). Manually patch `Node.Status.Allocatable` downward to 3Gi — below the desired resize target.
    * *Validation:* Verify the Kubelet detects that the deferred resize can no longer fit and transitions the pod's resize status from `Deferred` to `Infeasible`. Verify the Scheduler and VPA observe the updated status (they should no longer treat this as a pending retry).
    * *Layer:* Layer 1 — KEP-1287 interaction contract (downscale path).

* **Test 7: Infeasible Pod Resize Cleared on Node Upscale**
    * *Action:* On a node with 4Gi allocatable memory, submit a pod resize request to 6Gi. The Kubelet marks it `Infeasible` due to insufficient node capacity. Manually patch `Node.Status.Allocatable` upward to 8Gi.
    * *Validation:* Verify the Kubelet re-evaluates the `Infeasible` resize, determines the node now has sufficient capacity, and transitions the resize status to `InProgress` / `Accepted`. Confirm the Kubelet correctly distinguishes between a capacity-bound `Infeasible` (retriable) and an `Infeasible` caused by other reasons (not retriable, e.g., resource type not supported).
    * *Layer:* Layer 1 — KEP-1287 interaction contract (capacity-bound Infeasible retry).

* **Test 8: Scheduler Preemption Recalculation on Capacity Decrease During Grace Period**
    * *Action:* On a node with 8Gi allocatable memory running a low-priority pod consuming 6Gi, submit a high-priority pod requiring 7Gi. The Scheduler selects the low-priority pod as a preemption victim and initiates its termination grace period. Before the grace period expires, manually patch `Node.Status.Allocatable` downward to 4Gi.
    * *Validation:* Verify the Scheduler receives the Node UPDATE event, re-queues the waiting high-priority pod for a fresh scheduling cycle, and re-runs preemption calculations against the new (4Gi) allocatable value — rather than proceeding with the stale assumption that 8Gi will be available once the victim terminates. Confirm the queueing hint for `Node.Status.Allocatable` decreases correctly triggers re-evaluation of pods in the `WaitingForPreemption` state.
    * *Layer:* Layer 1 — Scheduler preemption queueing hint contract.

### Layer 2 Implementation: Declarative Capacity Actuation

This section implements the Layer 2 changes described in the Proposal: the `Node.Spec.ConfiguredCapacity` API field, the Kubelet reconciliation loop, and the `CapacityConfigured` condition. These changes are gated on Layer 1 pre-requisite tests passing, since the reconciliation loop produces the same kind of `Node.Status.Capacity` mutation that Layer 1 validates the ecosystem can handle safely.

#### Proposed Core Code Changes

1. **Dedicated Node Capacity syncLoop** (`pkg/kubelet/kubelet.go`)

   A dedicated node capacity syncLoop runs as its own goroutine within `Run()`, analogous to the pod syncLoop. It owns the full lifecycle of a capacity reconciliation event and executes all reconciliation steps serially, ensuring no concurrent capacity mutations. The loop is fed by a single `capacityReconciliationCh` channel written by two sources: the Node informer (when `Node.Spec.ConfiguredCapacity` changes) and the cAdvisor hardware-drift detector (when physical capacity changes).

```go
// 1. Wire up the Node Informer to feed the capacity syncLoop
kl.nodeInformer.AddEventHandler(cache.ResourceEventHandlerFuncs{
    UpdateFunc: func(oldObj, newObj interface{}) {
        oldNode := oldObj.(*v1.Node)
        newNode := newObj.(*v1.Node)
        if !apiequality.Semantic.DeepEqual(oldNode.Spec.ConfiguredCapacity, newNode.Spec.ConfiguredCapacity) {
            kl.capacityReconciliationCh <- struct{}{}
        }
    },
})

// 2. Node capacity syncLoop
if utilfeature.DefaultFeatureGate.Enabled(features.InPlaceNodeResourceResize) {
    go wait.Until(func(ctx context.Context) {
        // capacityReconciliationCh is written by: Node informer OR cAdvisor hardware-drift detector
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

   Resource managers must implement a new interface to accept dynamic sync events natively on the running node.
```go
// ResourceResizer defines the interface for sub-managers to accept dynamic capacity changes
type ResourceResizer interface {
    // SyncCapacity safely re-evaluates the sub-manager's internal state against the new boundaries
    SyncCapacity(capacity ResourceList) error
}
```

3. **Resource Manager Synchronization and State Reconciliation**

When the Container Manager detects a capacity drift, it notifies its sub-managers to synchronize their internal state. Because the Kubelet manages cpusets, memory pages, checkpointed state files, and NUMA alignments, dynamic resize requires coordinated reconciliation across each subsystem:

#### 3.1 CPU Manager Synchronization
The CPU Manager dynamically reconciles capacity changes depending on the configured policy:

* **Policy: `none`**
  - All pods run across the entire machine's cpuset. On upscale or downscale, the Kubelet updates the host and QoS cgroup cpuset hierarchies to match the new root cpuset.

* **Policy: `static`**
  - **Shared Pool Updates for Burstable and BestEffort Pods:**
    - Non-Guaranteed pods (Burstable, BestEffort) and Guaranteed pods with non-integer CPU requests execute within the default shared cpuset pool (all physical CPUs excluding reserved CPUs and active exclusive allocations).
    - When capacity changes, the CPU Manager recalculates the shared cpuset pool and triggers an active reconciliation run to push updated cpuset boundaries to all running Burstable and BestEffort containers through standard container runtime resource updates.
  - **Reserved CPU Invariant:**
    - Reserved CPUs represent an invariant reservation for host and kubelet system daemons.
    - If a hot-unplug event attempts to remove CPU IDs that overlap with the configured reserved CPUs, the Kubelet rejects the downscale as infeasible to protect host system stability.
  - **Guaranteed Pods and Exclusive Core Allocations:**
    - *Upscale:* Newly added CPU IDs expand the shared pool, making more cores available for shared workloads or for subsequent Guaranteed pod admissions.
    - *Downscale:* If CPU core removal reduces total capacity below the count required for active exclusive allocations, or if removed CPU IDs directly overlap with exclusive cores pinned to running Guaranteed containers, the Kubelet evicts the affected pods with a `Failed` status (Reason: `NodeCapacityExceeded`).
  - **Burstable Pod Degradation Semantics:**
    - For Burstable pods, CPU requests establish scheduler bandwidth shares. CPU is a compressible resource: if a downscale reduces node CPU below aggregate Burstable requests, the Kubelet's eviction manager does not proactively evict those pods — they gracefully degrade and share available CPU bandwidth proportionally across the contracted shared cpuset.
    - However, this does not prevent Scheduler-driven preemption. When new higher-priority pods need to be scheduled onto the node after a downscale, the Scheduler sums aggregate requests against the new (smaller) allocatable value and will preempt lower-priority Burstable pods to make room, following standard Kubernetes preemption semantics. This KEP introduces no changes to that behaviour: preemption decisions remain entirely within the Scheduler and are driven by PriorityClass, not by this reconciliation loop.

#### 3.2 Memory Manager and Memory QoS Synchronization
* **NUMA Node Allocation Boundaries:**
  - The Memory Manager re-evaluates available physical memory and hugepages per NUMA node, updating its internal state memory map.
* **Memory QoS:**
  - For nodes running with Memory QoS enabled, resizing node allocatable memory alters the proportional calculation for memory protection and throttling boundaries.
  - If a memory downscale causes aggregate memory usage to exceed new limits, standard Memory QoS throttling triggers, followed by standard Eviction Manager ranking (evicting BestEffort workloads before Burstable).

#### 3.3 Topology Manager and NUMA Layout
* **Machine Topology Refresh:**
  - The Topology Manager queries the refreshed machine topology information to update its internal NUMA cell map, socket counts, and distance matrix.
* **Admission Alignment:**
  - Running pods retain their existing NUMA node and resource pinning. Future pod admissions use the updated NUMA boundaries and refreshed hint providers for single-NUMA or multi-NUMA alignment decisions.

#### 3.4 Checkpoint State File Consistency
* The CPU Manager and Memory Manager persist state across restarts via local checkpoint state files on disk.
* Capacity synchronization ensures that whenever in-memory state (such as the shared pool, allocations, and NUMA memory maps) is modified, the new topology and allocation table are atomically committed to the state files on disk. This prevents topology validation errors during subsequent Kubelet restarts.

#### 3.5 Feature Scope Progression (Alpha to GA)
To ensure safety and manage implementation complexity:
* **Alpha Scope:**
  - **CPU Manager:** Full support for `cpuManagerPolicy: none`. For `cpuManagerPolicy: static`, support upscaling (expanding the shared cpuset pool) and non-destructive downscaling (reclaiming unallocated shared cores). Hot-unplugging cores that conflict with reserved CPUs or allocated exclusive cpusets is rejected.
  - **Memory Manager:** Scoped to `memoryManagerPolicy: None`.
  - **Topology Manager:** Scoped to `topologyManagerPolicy: none` or single-NUMA architectures.
  - **Swap:** Not supported. Resize operations on swap-enabled nodes are rejected in Alpha.
* **Beta Scope:**
  - Swap-enabled node resize, including per-container swap limit recalculation via `UpdateContainerResources` CRI RPC, once swap–pod-resize interaction semantics are aligned.
  - Dynamic multi-NUMA topology cell changes, multi-NUMA memory block redistribution (`memoryManagerPolicy: Static`), and full Topology Manager hint provider recalculation across dynamic NUMA boundaries.

### Observability and Metrics

To ensure cluster operators can monitor resize events and failures, this KEP introduces the following Prometheus metrics within the Kubelet:

- `kubelet_node_resize_requests_total`: Counter tracking the number of successful native resource resize events (labeled by resource name and direction: `increase`/`decrease`).

- `kubelet_node_resize_errors_total`: Counter tracking failures during the reconciliation pipeline (labeled by the failing subsystem, e.g., `cpu_manager_sync`, `cgroup_update`).

### Test Plan

[x] I/we understand the owners of the involved components may require updates to
existing tests to make this code solid enough prior to committing the changes necessary
to implement this enhancement.

##### Layer 1: Ecosystem Tolerance Tests (Pre-requisite, no feature gate required)
These tests must be merged before any Layer 2 Kubelet code is written. They verify that the control plane safely handles a `Node.Status.Capacity` mutation regardless of how it was triggered:
1. **API Validation:** Verify that modifying `Node.Status.Capacity` and `Node.Status.Allocatable` on an existing Node object is explicitly permitted by the API server and does not trigger unintended systemic webhook rejections.
2. **Scheduler Validation:** Verify that if a Node's capacity is mutated (simulating an offline/restarted Kubelet resize), the Scheduler correctly recognizes the new capacity and successfully schedules/rejects pending Pods accordingly without requiring the Node object to be deleted and recreated.
3. **Autoscaler Integration:** Verify how the Cluster Autoscaler reacts to a dynamically mutated `Node.Status.Capacity` and document the current behaviour — specifically whether a resized node gets incorrectly selected as a NodeGroup provisioning template. The correct long-term fix for CA template stability is an open design item (see Cluster Autoscaler compatibility discussion above).

##### Layer 2: Unit tests (require `InPlaceNodeResourceResize` feature gate)

1. **cAdvisor Cache Refresh** (`kubelet_node_status_test.go`): Verify that injecting a new `MachineInfo` struct successfully updates the Kubelet's internal cache, and that the subsequent node status sync does not incorrectly clamp the new `Allocatable` values to the old boot-time capacity.

2. **Cgroup Enforcement** (`container_manager_linux_test.go`): Verify that when a capacity change is detected, `enforceNodeAllocatableCgroups` and `UpdateQOSCgroups` are invoked with the newly calculated boundaries, and that the `nodeCapacityUpdateCh` signal is successfully emitted without blocking.

3. **Sub-Manager Re-initialization** (`cpu_manager_test.go`, `memory_manager_test.go`): Verify that the CPU and Memory managers properly implement the `ResourceResizer` interface and cleanly accept `SyncCapacity()` calls without leaking state or crashing.

4. **Eviction Threshold Sync** (`eviction_manager_test.go`): Verify that `SynchronizeThresholds` correctly recalculates absolute byte values (e.g., < `100Mi` vs `10%`) when the underlying capacity increases or decreases.

5. **CRI Swap Limit Recalculation** (`kubelet_test.go`): Verify the math for proportional swap limits. Ensure the capacity reconciliation loop correctly iterates over active pods and invokes the mock CRI `UpdateContainerResources` interface with the newly calculated boundaries.

6. **Bootstrap Parity** (`kubelet_test.go`): Verify that when the Kubelet starts and finds a pre-existing `Node.Spec.ConfiguredCapacity` in the API Server, it uses that value as the desired target rather than silently overwriting it with the raw cAdvisor-discovered capacity. Specifically confirm that if `ConfiguredCapacity` is set to 20Gi on a 32Gi physical node, the Kubelet reports 20Gi in `Node.Status.Capacity` after startup, not 32Gi.

##### Layer 2: e2e tests (require `InPlaceNodeResourceResize` feature gate)

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

#### Phase 1: Alpha (v1.38) — Layer 1: Ecosystem Tolerance

The v1.38 Alpha milestone is scoped to **Layer 1 only**. The goal is to prove the broader control-plane ecosystem can safely tolerate a live change to `Node.Status.Capacity` — using the Kubelet-restart-on-resized-hardware path as the initial trigger — before any new declarative API or live Kubelet actuation (Layer 2) is introduced.

* **Layer 1 (Ecosystem Tolerance):** API, Scheduler, and Autoscaler e2e tests are merged to officially validate and document that the Kubernetes ecosystem can safely handle `Node.Status.Capacity` mutations. These tests have no feature gate dependency and establish the safety baseline for all Layer 2 work. Specifically:
  * The API Server accepts live patches to `Node.Status.Capacity` on a `Ready` node.
  * The Scheduler's `NodeInfo` cache correctly invalidates and re-evaluates capacity on Node UPDATE events (both upscale and downscale).
  * The Cluster Autoscaler's NodeGroup template behaviour when `Node.Status.Capacity` is mutable is documented and the failure mode is validated. The long-term CA fix is an open design item.
  * VPA recommendations are re-bounded against updated node allocatable values.
  * ResourceQuota enforcement is not bypassed by a capacity change.
* **Kubelet Restart Boundary:** The Kubelet correctly handles a restart on a node whose hardware capacity changed while offline — reconciling gracefully rather than crashing or permanently blocking pod admission (Test 4 in the Layer 1 pre-requisite test plan).
* No new `NodeSpec` API fields are introduced in this milestone. No `InPlaceNodeResourceResize` feature gate is required for any v1.38 deliverable.

#### Phase 2: Alpha (v1.39) — Layer 2: Declarative Capacity Actuation *(planned)*

Layer 2 targets a subsequent milestone once the Layer 1 ecosystem contracts are validated and the open design questions in [Open Questions for Layer 2](#open-questions-for-layer-2) are resolved. The criteria below are provisional.

* Feature is disabled by default via the `InPlaceNodeResourceResize` feature gate (kubelet, kube-apiserver, kube-scheduler).
* The `Node.Spec.ConfiguredCapacity` API field is introduced. The Kubelet reconciles capacity mismatches for CPU and Memory — covering both the live-node and Kubelet-restart-on-resized-hardware cases. The Kubelet updates cgroups, re-initialises sub-managers (`cpuManagerPolicy: none`, `memoryManagerPolicy: None`), and evicts starved pods.
* The cAdvisor metrics-based hardware-drift trigger is implemented, enabling the emergency downscale path (Path B).
* The Scheduler Node UPDATE queueing hint for `WaitingForPreemption` pods is implemented and validated.
* All Layer 1 pre-requisite tests continue to pass.
* Integrations with `cpuManagerPolicy: static`, Swap, and Topology Manager are explicitly deferred to Beta.
* Out-of-tree controller integrations (Cluster Autoscaler, VPA) are not required for this milestone.

#### Phase 3: Beta

* The `InPlaceNodeResourceResize` feature gate is enabled by default.
* Memory resize on swap-enabled nodes is supported: per-container `memory.swap.max` recalculation via `UpdateContainerResources` CRI RPC is implemented and validated.
* `cpuManagerPolicy: static` resize is fully supported for both upscale (expanding the shared cpuset pool) and non-destructive downscale; destructive downscale evicts affected pods with `NodeCapacityExceeded`.
* Topology Manager integration is complete for multi-NUMA configurations: topology cell changes, `memoryManagerPolicy: Static` redistribution, and hint provider recalculation across dynamic NUMA boundaries.
* The Cluster Autoscaler NodeGroup template stability problem is resolved (see [Open Questions for Layer 2](#open-questions-for-layer-2)) and the agreed approach is implemented.
* The Scheduler `min(Node.Spec.ConfiguredCapacity, Node.Status.Capacity)` capacity view, or the equivalent mitigation agreed in Open Question 3, is implemented and validated.
* Rollout, upgrade, and rollback planning is completed (required for Beta PRR).

#### Phase 4: GA

* The `InPlaceNodeResourceResize` feature gate is removed (always on).
* The feature has been enabled by default for at least two release cycles with no regressions.
* All e2e tests are flake-free for a minimum two-week window and meet [Conformance Test](https://github.com/kubernetes/community/blob/master/contributors/devel/sig-architecture/conformance-tests.md) requirements.
* All open design questions from the Alpha/Beta period are resolved and documented.

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

For the v1.38 Alpha (Layer 1), no Kubelet restart or feature gate change is required. The Layer 1 tests operate purely at the control-plane level and are always enabled.

For the v1.39 Alpha (Layer 2), the Kubelet must be restarted with the `InPlaceNodeResourceResize` feature gate enabled. Existing clusters do not experience any immediate impact upon upgrade; the Kubelet will simply begin mirroring the existing physical hardware capacity dynamically.

##### Downgrade

For Layer 2, it is trivially possible to downgrade by disabling the `InPlaceNodeResourceResize` feature gate and restarting the Kubelet. The Kubelet will revert to its legacy behavior: capturing the node capacity once during boot and freezing it. Any subsequent hardware hot-plugs will be safely ignored. The Layer 1 ecosystem tests are always-on and have no rollback requirement.

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

To minimise this window during a downscale, the Scheduler uses `min(Node.Spec.ConfiguredCapacity, Node.Status.Capacity)` as its view of effective node capacity. As soon as an operator writes a reduced value to `Node.Spec.ConfiguredCapacity`, the Scheduler conservatively accounts for the smaller capacity in its scheduling decisions — even before the Kubelet has finished reconciling `Node.Status.Capacity`. This is directly analogous to the treatment for in-place pod resize, where the Scheduler uses `max(pod.spec.resources, pod.status.resources)` to avoid scheduling against a pod whose resources are still being expanded. Together, these two conventions encode a consistent principle: *always assume the worst-case resource footprint for any in-flight change*. The race window is reduced to the interval between the `Node.Spec.ConfiguredCapacity` write and the Scheduler's next cache refresh, rather than the full duration of Kubelet reconciliation. Kubelet admission remains the authoritative gate and the final safety net for correctness.

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
    - Components depending on the feature gate: `kubelet`, `kube-apiserver`, `kube-scheduler` (Layer 2 only; no feature gate is required for the v1.38 Alpha Layer 1 work)
    - Will enabling / disabling the feature require downtime of the control plane? For the v1.38 Alpha (Layer 1), no feature gate exists and no component restart is required — the Layer 1 work consists entirely of test coverage and has no runtime impact. For the v1.39 Alpha (Layer 2), enabling `InPlaceNodeResourceResize` requires a coordinated rollout across all three gated components (kubelet, kube-apiserver, kube-scheduler). The API Server must be updated first so the new `Node.Spec.ConfiguredCapacity` field is recognised before the Kubelet begins writing to it. The Scheduler must be updated to activate the Node UPDATE queueing hint and the `min()` capacity view. Rolling the control plane during the upgrade constitutes the required downtime; it is bounded to the standard control-plane rolling-update window and does not affect running workloads.
    - Will enabling / disabling the feature require downtime or reprovisioning of a node? **Yes, a Kubelet restart is required** to toggle the `InPlaceNodeResourceResize` feature gate on the node. This does not disrupt running pods; existing cgroups and container state are preserved by the container runtime across the restart.

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

**API Server (Layer 2):** The `Node.Spec.ConfiguredCapacity` field introduced in Layer 2 requires the API Server to recognise the updated `NodeSpec` schema. The API Server must be at a version that includes the new field before any Kubelet or external controller can use it.

**Scheduler (Layer 1 validation + Layer 2):** The Scheduler requires changes in two areas identified by this KEP: (a) a Node UPDATE queueing hint to re-queue pods in the `WaitingForPreemption` state when `Node.Status.Allocatable` decreases on their target node (needed for correctness with any mutable-capacity scenario, validated in Layer 1), and (b) using `min(Node.Spec.ConfiguredCapacity, Node.Status.Capacity)` as the effective capacity view during a downscale (Layer 2). If the Scheduler is not updated, the preemption grace period race and the scheduling-binding race window remain unmitigated.

**Cluster Autoscaler (Layer 1 validation):** The Layer 1 deliverable is to document and validate how CA behaves when `Node.Status.Capacity` changes — specifically the NodeGroup template corruption risk when a resized node is selected as the provisioning reference. The correct long-term fix (whether CA reads from the cloud provider's launch template, or uses a configurable reference capacity in its own NodeGroup configuration) is an open design item that must be resolved before this KEP's CA integration is standardised. CA changes are out of scope for this KEP's feature gate.

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
- Estimated increase in size: ~100-300 bytes per Node object. This is due to the addition of the `Node.Spec.ConfiguredCapacity` field and the `CapacityConfigured` Status Condition. No boot-time annotation or additional status field for the Autoscaler is introduced in the current design.
- Estimated amount of new objects: 0 (No new objects are created).

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
- **v1.38**: Alpha milestone scoped to **Layer 1 only** (Ecosystem Tolerance). No new API fields or feature gate. The declarative Layer 2 design is documented in this KEP for community review but deferred to v1.39 pending resolution of the open design questions captured in [Open Questions for Layer 2](#open-questions-for-layer-2).

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

## Open Questions for Layer 2

The following design questions remain open and must be resolved before Layer 2 implementation begins. They are recorded here so that the community can discuss them in parallel with the Layer 1 milestone.

1. **`Node.Spec.ConfiguredCapacity` vs. `NodeDesiredAllocatable`**

   The current Layer 2 design proposes overwriting `Node.Status.Capacity` via `Node.Spec.ConfiguredCapacity`. An alternative approach is to keep `Node.Status.Capacity` anchored strictly to physical reality and instead introduce a separate declarative field — such as `NodeDesiredAllocatable` — that adjusts the *scheduling bound* without touching the raw capacity. This separation would eliminate several of the risks catalogued in the [Risks and Mitigations](#risks-and-mitigations) section (e.g., API Status Clamping, cAdvisor polling latency) because the physical capacity field would remain immutable. The trade-off is a more complex mental model (two writable fields with different semantics) and potential ambiguity for components that currently treat `Capacity` and `Allocatable` as a single authoritative source. The Layer 1 milestone is expected to generate practical evidence that informs which approach is cleaner.

2. **Overcommit and Dense Burst Workloads**

   There is growing interest in supporting highly dense, bursty workloads (AI agents, serverless sandboxes) by restricting scheduling bounds (`NodeAllocatable`) to maximise pod density while keeping parent cgroups intentionally wide, allowing concurrent short-lived workloads to burst into unallocated physical memory. This is in tension with the current Layer 2 design, which ties cgroup boundaries directly to the declared capacity. How the Kubelet should handle a `ConfiguredCapacity` value that implies different cgroup ceiling vs. scheduler-visible capacity needs to be defined before Layer 2 can ship.

3. **Scheduler Race Window Mitigation**

   The current Layer 2 design introduces a race window: an external controller patches `Node.Spec.ConfiguredCapacity`, and subsequently the Kubelet processes the event and changes both `Capacity` and `Allocatable` in `Node.Status`, while the Scheduler only reads `Node.Status.Allocatable`. The proposed mitigation — having the Scheduler use `min(Node.Spec.ConfiguredCapacity, Node.Status.Capacity)` as its effective capacity — requires a Scheduler change and needs community agreement on whether that change belongs in the Scheduler core or in a plugin.

   A related and distinct concern involves the preemption grace period: if node capacity decreases while a victim pod is in its termination grace period (following a preemption decision), the Scheduler's original fit calculation for the preemptor pod is now stale. The correct fix is a Node UPDATE queueing hint that re-queues pods in the `WaitingForPreemption` state when `Node.Status.Allocatable` decreases on their target node. Whether this hint already exists, or needs to be added, must be confirmed as part of the Layer 1 Scheduler contract work. Both the `min()` capacity view change and the queueing hint fix may ultimately be the same Scheduler change, or they may be independent — that needs to be determined before Layer 2 ships.

4. **Cluster Autoscaler NodeGroup Template Stability**

   The Cluster Autoscaler uses existing nodes as templates for provisioning new nodes within the same NodeGroup. When node capacity is mutable, a resized node may be selected as that template, causing newly provisioned nodes to be expected at the larger size while the cloud provider continues to provision at the original base size — resulting in scheduling failures. Three approaches have been considered:

   - A `resize.node.kubernetes.io/initial-capacity` annotation stamped by the Kubelet at first boot, read by CA as its reference baseline. This is low friction but annotations are brittle, lack schema validation, and are awkward when multiple actors read them.
   - A dedicated `Node.Status.InitialCapacity` field written once at first registration. More idiomatic, but introduces a new API field and conflates a historical boot value with live node status.
   - CA reading directly from the cloud provider's launch template, or introducing a configurable reference capacity within CA's own NodeGroup configuration. This decouples the problem from the Kubernetes Node object entirely, which is arguably the cleanest separation of concerns, but requires cloud-provider portability work outside this KEP.

   None of these options is settled. An additional edge case complicates all three: if an entire NodeGroup has been uniformly resized, the "initial" capacity is no longer a meaningful reference and the NodeGroup has functionally changed size. The right approach must handle this case explicitly. This is tracked as a next-phase item to be explored after the Layer 1 milestone establishes the observed behaviour baseline.


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

* **Cluster Autoscaler Integration**

  * Once the open design question in [Open Questions for Layer 2](#open-questions-for-layer-2) is resolved, the agreed approach will be implemented in a subsequent phase. The goal is to ensure CA-managed clusters can safely operate with nodes whose capacity changes over time, without risking NodeGroup template corruption or provisioning failures.
