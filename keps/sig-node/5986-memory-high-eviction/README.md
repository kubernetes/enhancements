# KEP-5986: Per-container memory pressure eviction

<!-- toc -->
- [Release Signoff Checklist](#release-signoff-checklist)
- [Summary](#summary)
- [Motivation](#motivation)
  - [Goals](#goals)
  - [Non-Goals](#non-goals)
- [Proposal](#proposal)
  - [Detection: Three Signals](#detection-three-signals)
  - [Why Not a New Eviction Signal](#why-not-a-new-eviction-signal)
  - [Response: Two-Stage Remediation](#response-two-stage-remediation)
  - [Rate Calculation](#rate-calculation)
  - [KubeletConfiguration](#kubeletconfiguration)
  - [User Stories](#user-stories)
    - [Perpetually Throttled Java Workload](#perpetually-throttled-java-workload)
    - [Bursty ML Training Job](#bursty-ml-training-job)
    - [memory.min Contention Across Pods](#memorymin-contention-across-pods)
  - [Notes/Constraints/Caveats](#notesconstraintscaveats)
    - [Feature Interactions](#feature-interactions)
    - [Kernel Version Considerations](#kernel-version-considerations)
    - [CRI Runtime Requirements](#cri-runtime-requirements)
    - [Platform Support](#platform-support)
    - [No-Swap Scenario](#no-swap-scenario)
    - [Eviction-Recreation Loop](#eviction-recreation-loop)
  - [Risks and Mitigations](#risks-and-mitigations)
- [Design Details](#design-details)
  - [Feature Gate](#feature-gate)
  - [Stats API Extension](#stats-api-extension)
  - [Eviction Manager Integration](#eviction-manager-integration)
    - [Stage 1 Dependencies](#stage-1-dependencies)
    - [Detection](#detection)
    - [Candidate Selection](#candidate-selection)
    - [Eviction Observability](#eviction-observability)
  - [Metrics](#metrics)
  - [Prior Art](#prior-art)
  - [Future: EvictionRequest API Integration](#future-evictionrequest-api-integration)
  - [Test Plan](#test-plan)
      - [Prerequisite testing updates](#prerequisite-testing-updates)
      - [Unit tests](#unit-tests)
      - [Integration tests](#integration-tests)
      - [e2e tests](#e2e-tests)
  - [Graduation Criteria](#graduation-criteria)
    - [Alpha (v1.37)](#alpha-v137)
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
  - [Prior Work](#prior-work)
- [FAQ](#faq)
  - [Why are memory.events counters needed when PSI already detects pressure?](#why-are-memoryevents-counters-needed-when-psi-already-detects-pressure)
  - [Why does stage 1 modify cgroup state from the eviction manager?](#why-does-stage-1-modify-cgroup-state-from-the-eviction-manager)
  - [Why do all thresholds default to 0?](#why-do-all-thresholds-default-to-0)
  - [How does the feature behave when MemoryQoS is disabled?](#how-does-the-feature-behave-when-memoryqos-is-disabled)
  - [How is the eviction-recreation loop mitigated?](#how-is-the-eviction-recreation-loop-mitigated)
  - [How does this feature co-exist with EvictionRequest API (KEP-4563)?](#how-does-this-feature-co-exist-with-evictionrequest-api-kep-4563)
  - [How will VPA learn about memory pressure evictions?](#how-will-vpa-learn-about-memory-pressure-evictions)
- [Drawbacks](#drawbacks)
- [Alternatives](#alternatives)
  - [PSI-Only Detection (No memory.events)](#psi-only-detection-no-memoryevents)
  - [Eviction-Only (No Stage 1 Remediation)](#eviction-only-no-stage-1-remediation)
  - [New Eviction Signal in the Signal Framework](#new-eviction-signal-in-the-signal-framework)
  - [systemd-oomd Integration](#systemd-oomd-integration)
  - [Absolute Threshold Instead of Rate-Based](#absolute-threshold-instead-of-rate-based)
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
- [ ] (R) Production readiness review completed
- [ ] (R) Production readiness review approved
- [x] "Implementation History" section is up-to-date for milestone
- [ ] User-facing documentation has been created in [kubernetes/website], for publication to [kubernetes.io]
- [ ] Supporting documentation, e.g., additional design documents, links to mailing list discussions/SIG meetings, relevant PRs/issues, release notes

[kubernetes.io]: https://kubernetes.io/
[kubernetes/enhancements]: https://git.k8s.io/enhancements
[kubernetes/kubernetes]: https://git.k8s.io/kubernetes
[kubernetes/website]: https://git.k8s.io/website

## Summary

When MemoryQoS ([KEP-2570](https://github.com/kubernetes/enhancements/issues/2570))
sets `memory.high` on a container cgroup, the kernel throttles allocations
that exceed the boundary. The kubelet currently has no way to detect this
throttling or act on it. This KEP adds per-container memory pressure
detection using `memory.events` counters and PSI, with a two-stage response:
relax the `memory.high` boundary first, evict only if pressure persists.

## Motivation

cgroup v2 `memory.high` is designed to work with a userspace management
agent that monitors throttling and takes action, either by granting more
memory or terminating the workload
([kernel docs](https://docs.kernel.org/admin-guide/cgroup-v2.html)).
KEP-2570 (MemoryQoS) introduced `memory.high` on container cgroups to
throttle memory overcommit, but the kubelet currently has no mechanism to
detect this throttling or act on it.

When a container exceeds `memory.high`, the kernel throttles its allocations
and tries to reclaim memory. If the container stays between `memory.high`
and `memory.max`, it gets perpetually throttled with no automatic recovery.
Containers that repeatedly approach `memory.max` see severe latency spikes
but never quite trigger OOM kill.
When multiple pods have `memory.low` set and total reserved memory exceeds
what the node has available, the kernel reclaims from the weakest pods,
causing sustained pressure that `memory.events:high` alone cannot detect.

The kernel tracks all of these via `memory.events` counters and PSI
(Pressure Stall Information). cadvisor already reads both. This KEP uses
them to detect sustained pressure and respond.

The existing kubelet eviction manager monitors node-level resource pressure
(memory.available, disk, PIDs) and evicts pods when the node is under stress.
All nine existing eviction signals are quantity-based (capacity/available),
measuring absolute resource levels. Per-container throttle rate and pressure
percentage are different signal types that do not fit this model.
This design gap was discussed in
[KEP-2570](https://github.com/kubernetes/enhancements/issues/2570#issuecomment-3964464265)
and confirmed as requiring a follow-up KEP.

### Goals

- Detect per-container sustained memory pressure using three kernel signals:
  `memory.events:high` rate, `memory.events:max` rate, and PSI
  `memory.pressure` avg10
- Respond with two-stage remediation: relax the `memory.high` throttle
  boundary first, evict only if pressure persists
- Provide configurable per-signal thresholds and grace period via
  `KubeletConfiguration`
- Emit metrics and pod conditions for operator observability

### Non-Goals

- PSI-based node-level protection (system-wide eviction trigger, planned
  for beta)
- Multi-signal weighted scoring (simple OR at alpha, weighted combination
  at beta)
- Per-pod opt-out of memory pressure eviction (beta consideration)
- Direct systemd-oomd integration (at beta, node-level PSI eviction
  provides equivalent functionality within the kubelet)
- New node condition for memory.high pressure (this is per-container, not
  node-level; `NodeMemoryPressure` already covers node-level)

## Proposal

Add a per-container memory pressure check to the kubelet eviction manager
using a two-stage remediation approach. The check runs as a parallel check
in the eviction loop, following the same pattern as the existing local
storage check. If it evicts a pod, signal-based eviction does not run that
cycle. Node-level signals are evaluated on the next cycle (10s default).

### Detection: Three Signals

The eviction manager reads three signals from each container's stats on
every eviction check cycle (~10s):

| Signal | Source | What it detects | When it fires |
|--------|--------|-----------------|---------------|
| `memory.events:high` rate | cgroup v2 `memory.events` high counter | Rapid allocation above `memory.high` | Workloads with continuous alloc/free cycles (GC, training jobs) |
| `memory.events:max` rate | cgroup v2 `memory.events` max counter | Container repeatedly approaching `memory.max` (near-OOM) | Severe latency from repeated near-OOM reclaim |
| PSI `memory.pressure` avg10 | cgroup v2 `memory.pressure` file | Any sustained memory pressure including stable-above-high usage | Workloads with stable working sets above memory.high, memory.min/low contention, page cache exhaustion |

Each signal catches cases the others miss. `memory.events:high` fires only
on new allocation attempts above `memory.high`, so it does not detect
workloads with stable usage above the boundary. PSI detects any ongoing
reclaim pressure regardless of allocation pattern.

All three signals are already available through the existing stats
path in the eviction manager. `memory.events` comes from cadvisor reading
cgroup v2 files. PSI is GA since Kubernetes v1.36 (feature gate locked) and
is available through both cadvisor and CRI-native stats providers.

Any signal exceeding its configured threshold starts a per-pod grace period
timer. The timer tracks that the pod is under pressure, regardless of which
signal triggered it. If the timer was started by memory.events:high and that
signal drops below threshold but PSI rises above threshold, the timer
continues. The timer resets only when all signals for the pod are below their
thresholds. If any signal remains above threshold for longer than the grace
period, the remediation flow triggers.

### Why Not a New Eviction Signal

Existing eviction signals are quantity-based (capacity/available). Rate-based
and percentage-based metrics do not fit that model. This feature uses the
same parallel check pattern as the local storage eviction, which performs
per-container checks outside the signal framework.

### Response: Two-Stage Remediation

**Stage 1. Relax memory.high:**

When sustained throttling is detected via `memory.events:high`, the kubelet
raises the container's `memory.high` to equal `memory.max` via CRI
`UpdateContainerResources`. This removes the throttle boundary while keeping
the hard limit intact. The container can use its full allocation without
being throttled.

Relaxing `memory.high` does not change the container's hard limit
(`memory.max`) or its resource requests/limits in the pod spec. The
container can use up to `memory.max`, which is the same limit it had
before MemoryQoS added the throttle boundary. `memory.high` is computed
locally by the kubelet and is not visible to the API server or scheduler.
The relaxed value persists until the container restarts or until a
subsequent container resource update (e.g., in-place resize) re-applies
the MemoryQoS formula.

At the time remediation triggers, the eviction manager checks which signals
are currently above threshold. If `memory.events:high` is active, stage 1
applies. If only PSI or `memory.events:max` is active, stage 2 applies
directly since `memory.high` is not the bottleneck.

If the CRI call fails, the kubelet logs the error. Pressure is re-evaluated
on the next cycle; if it persists and stage 1 has not succeeded for the
affected container, the eviction manager proceeds to stage 2.

**Stage 2. Evict:**

If pressure continues after stage 1, or if only PSI or `memory.events:max`
is active, the pod is evicted through the existing kubelet pod eviction
path.

### Rate Calculation

For `memory.events` counters:

```
rate = (current_counter - previous_counter) / time_delta_seconds
```

Where `current` and `previous` are values from consecutive eviction check cycles.

For PSI, `avg10` is a percentage computed by the kernel over a 10-second
sliding window and is read directly without rate calculation.

### KubeletConfiguration

Four new fields in `KubeletConfiguration` (internal and v1beta1 versioned
types, with defaults in `v1beta1/defaults.go`):

```go
// MemoryHighEvictionThreshold is the memory.events high rate (events/sec)
// above which a container is considered throttled. Set to 0 to disable.
MemoryHighEvictionThreshold float64

// MemoryMaxEvictionThreshold is the memory.events max rate (events/sec)
// above which a container is considered near-OOM. Set to 0 to disable.
MemoryMaxEvictionThreshold float64

// MemoryPSIEvictionThreshold is the PSI memory.pressure Some avg10
// percentage (0-100) above which a container is considered under memory
// pressure. Set to 0 to disable.
MemoryPSIEvictionThreshold float64

// MemoryPressureEvictionGracePeriod is how long sustained pressure must
// persist before remediation triggers.
MemoryPressureEvictionGracePeriod metav1.Duration
```

All thresholds default to 0 (disabled). When the feature gate is enabled
without explicit configuration, no detection or eviction occurs. Operators
must set at least one threshold to activate the feature, so the gate alone
does not change behavior.

Operators can enable only the signals they care about by setting the others
to 0. Validation rejects negative values for all thresholds and grace
period, and rejects PSI threshold values above 100.

### User Stories

#### Perpetually Throttled Java Workload

A Java application has `requests.memory=512Mi` and `limits.memory=1Gi`.
MemoryQoS sets `memory.high` at approximately 971Mi (with the default 0.9
throttling factor). The JVM frequently allocates and garbage-collects above
`memory.high`, generating sustained `memory.events:high` counter increments
and elevated PSI pressure. The application's latency degrades significantly.
Without this feature, the pod sits in this degraded state indefinitely. With
this feature, the kubelet detects the sustained throttling, relaxes
`memory.high` to `memory.max` (1Gi), and the container stops being throttled.
If the workload stabilizes, no eviction occurs.

#### Bursty ML Training Job

An ML training job periodically spikes above `memory.high` during checkpoint
writes but quickly drops back below. The grace period (e.g., 30s) allows
transient spikes to pass without triggering remediation. Only sustained
pressure triggers the two-stage response.

#### memory.min Contention Across Pods

Three Burstable pods share a node, each with `memory.low` set by MemoryQoS.
The total `memory.low` exceeds available memory. The kernel reclaims from the
pod with the lowest protection, causing sustained PSI pressure. The
`memory.events:high` counter does not increment (the container is not above
its own `memory.high`), but PSI detects the stalling. The kubelet evicts the
lowest-priority pod experiencing pressure, freeing memory for the protected
pods.

### Notes/Constraints/Caveats

#### Feature Interactions

| Feature | Interaction | Handling |
|---------|-------------|----------|
| **MemoryQoS (KEP-2570)** | `memory.high` is only set when MemoryQoS gate is ON. Without it, `memory.events:high` is never incremented. | MemoryQoS is required for memory.events signals and stage 1 remediation. PSI-only detection works without MemoryQoS. Validation logs a warning if MemoryHighEviction is enabled without MemoryQoS and memory.events thresholds are set. |
| **InPlacePodVerticalScaling (KEP-1287)** | Resize DOWN triggers immediate throttling as the new lower `memory.high` is applied. | Two-stage remediation handles this naturally: stage 1 relaxes memory.high, giving the container breathing room during resize. No separate suppress mechanism needed. |
| **VPA** | InPlace mode triggers the same resize interaction as KEP-1287. | Covered by two-stage remediation. |
| **QoS Classes** | Guaranteed pods never get `memory.high` set (request == limit). | Naturally exempt from memory.events-based detection. PSI-based detection also skips Guaranteed pods since they should not be evicted for per-container pressure. |
| **Pod Priority** | Multiple pods under pressure should be evicted in priority order. | Sort candidates by QoS class (BestEffort first, then Burstable), then by priority (lowest first), then by pressure severity. |
| **Sidecar Containers (KEP-753)** | Sidecar container stats are included in `PodStats.Containers`. | Sidecar pressure triggers pod remediation/eviction. |
| **Container Restart** | New cgroup starts with `memory.events:high = 0`. | Track container start time. Reset counters when start time changes. The `high >= lastHigh` guard prevents false positives from counter decrease. |
| **Node-level Memory Eviction** | The memory pressure check runs before signal-based eviction in the eviction manager's main loop. | Different scope: per-container pressure vs node-level exhaustion. Both can fire independently. |
| **EvictionRequest API (KEP-4563)** | Both targeting v1.37 alpha. EvictionRequest provides graceful eviction with interceptors. | Alpha: use the kubelet's pod eviction path directly. Beta: route through EvictionRequest for PDB respect and interceptor support. The two-stage design gives interceptors time to act. |
| **PSI Node Conditions (KEP-4205 Phase 2)** | PSI-based `NodeMemoryPressure` condition. | Complementary: they set node conditions (scheduling avoidance), we evict specific pods (remediation). Different scope, no conflict. |
| **Graceful Node Shutdown (KEP-2000)** | Shutdown manager kills pods directly via pod workers. | Pod worker serializes concurrent kills. Harmless overlap. |
| **PDB** | Kubelet eviction does not check PDBs. | Known limitation for all kubelet evictions. EvictionRequest API integration at beta provides PDB support. |
| **DRA** | Evicted pods with GPU allocations follow normal termination path. | DRA claims released via `NodeUnprepareResources` after containers stop. 60s reconcile safety net catches missed cleanups. No risk. |
| **Swap (KEP-2400)** | When swap is available, kernel reclaim under `memory.high` can push pages to swap. | With swap, reclaim is more effective and throttling may resolve naturally. Rate-based detection adapts since the counter stops incrementing if reclaim succeeds. |
| **PodOverhead** | `memory.high` is set on container cgroups based on container requests/limits, not pod-level overhead. | No interaction. |
| **Cgroup Driver** | `memory.events` and `memory.pressure` are standard cgroup v2 interface files. | No difference between systemd and cgroupfs drivers. |

#### Kernel Version Considerations

The livelock fix (`MEMCG_MAX_HIGH_DELAY_JIFFIES` = 2 seconds max sleep) was
added in kernel 5.9 (commit `b3ff929`). MemoryQoS (KEP-2570) treats 5.9 as
the minimum for safe deployment because without it, containers can stall
indefinitely at `memory.high`. This KEP provides the remediation for that
stall. On pre-5.9 kernels the feature still works but throttle severity is
higher.

#### CRI Runtime Requirements

Stage 1 relies on `UpdateContainerResources` applying the cgroup v2 Unified
map. containerd supports this. CRI-O supports it from v1.35.3 / v1.36.0
([PR #9820](https://github.com/cri-o/cri-o/pull/9820)).

#### Platform Support

This feature requires Linux with cgroup v2. On Windows or other non-cgroup-v2
platforms, all three signals are unavailable and the feature is a no-op.

On Linux, `memory.events` is available through cadvisor. PSI is available
through both cadvisor and CRI-native stats providers. If a future CRI-only
mode on Linux drops cadvisor, `memory.events` signals would be unavailable
but PSI detection and stage 2 eviction would still work.

If the runtime does not apply the Unified update, the container remains
throttled. The kubelet observes continued pressure on subsequent cycles
and proceeds to stage 2.

#### No-Swap Scenario

When swap is disabled (common in Kubernetes), the kernel can only reclaim
file-backed pages under `memory.high`. If the workload is dominated by
anonymous pages (heap allocations), reclaim is ineffective and the process
gets throttled indefinitely with no actual memory freed. The two-stage
remediation is most useful here:
relaxing `memory.high` stops the futile throttling.

#### Eviction-Recreation Loop

An evicted pod's replacement can land on the same node. The two-stage
remediation reduces the loop frequency since stage 1 often resolves the
issue without eviction. When eviction does occur, the grace period bounds
the cycle. Beta targets VPA integration to break the loop by increasing
memory recommendations based on eviction events.

### Risks and Mitigations

| Risk | Mitigation |
|------|------------|
| False positive (remediation triggers from transient pressure) | Configurable grace period. Only sustained pressure past the grace period triggers action. Feature gate defaults to OFF. |
| Relaxing memory.high increases node memory pressure | `memory.max` is unchanged, so the container cannot use more than its limit. Relaxing memory.high returns to pre-MemoryQoS behavior, which is the safe baseline. |
| MemoryQoS gate OFF (no `memory.high` set) | `memory.events:high` and stage 1 remediation are no-ops. PSI signal still functions independently. Validation logs a warning. |
| Kernel < 5.9 (more severe throttling) | Eviction is more beneficial on these kernels because it prevents indefinite livelock. |
| CRI-only stats provider (memory.events unavailable) | PSI signal still works (available through CRI-native path). |
| Eviction-recreation loop | Grace period bounds loop frequency. Two-stage remediation reduces eviction rate. Beta: VPA + InPlacePodResize integration. |

## Design Details

### Feature Gate

`MemoryHighEviction` (alpha, default OFF, v1.37). Requires `MemoryQoS` to
be enabled for `memory.events`-based signals. PSI signal works independently
of MemoryQoS.

### Stats API Extension

Add an `Events` field to `MemoryStats` in the kubelet Summary API to expose
cgroup v2 `memory.events` counters per container. The field contains two
counters: `high` (throttle events) and `max` (near-OOM events). These are
monotonically increasing counters that reset when the container restarts
(new cgroup).

PSI data is already available in the Summary API via `MemoryStats.PSI`
(GA since v1.36). No additional API changes needed for PSI.

### Eviction Manager Integration

The memory pressure check runs as a parallel check in the eviction
manager's main loop, following the same pattern as the existing
per-container local storage check. It runs after local storage eviction
and before signal-based eviction.

On each cycle, the eviction manager reads per-container
`memory.events` counters and PSI data from the existing stats pipeline,
computes rates for `memory.events` counters, and compares all three signals
against their configured thresholds. If any signal exceeds its threshold
and the grace period has elapsed, the two-stage response triggers.

#### Stage 1 Dependencies

Stage 1 needs to call CRI `UpdateContainerResources` to relax
`memory.high`. The eviction manager does not have direct CRI access. A
resource update callback is injected at construction time, following the
same pattern as the existing pod kill callback. The kubelet implements this
callback by reusing the existing resource-generation path
(`generateLinuxContainerResources` with `enforceMemoryQoS=false`), which
already produces the correct Unified map with `memory.high=max`. Container
IDs are resolved by matching container names from the stats pipeline
against pod status (`ContainerStatuses`, `InitContainerStatuses`,
`EphemeralContainerStatuses`), which the eviction manager already receives
through its active pods function.

#### Detection

- `memory.events:high` and `memory.events:max` use rate-based detection:
  the delta between consecutive counter readings divided by elapsed time.
  Counter resets (from container restarts) are detected via container
  start time and handled by resetting the stored baseline.
- PSI `memory.pressure` Some avg10 is a kernel-computed percentage over
  a 10-second window. It is compared directly against the threshold.

#### Candidate Selection

When multiple pods exceed the grace period, candidates are sorted by
QoS class (BestEffort before Burstable), then by pod priority (lowest
first), then by the highest ratio of any active signal value to its
threshold across the pod's containers. Guaranteed pods are naturally
exempt since MemoryQoS does not set `memory.high` on them. Static pods
are skipped.

#### Eviction Observability

Evicted pods receive:
- A `DisruptionTarget` pod condition with reason `TerminationByKubelet`
- A warning event with reason `Evicted` and a message indicating which
  container triggered the eviction and which signal was exceeded (e.g.,
  "Container foo exceeded memory.high throttle threshold")
- Annotations: `OffendingContainersKey` (throttled container name) and
  `StarvedResourceKey` ("memory")

Stage 1 sets a `ContainerMemoryPressure` pod condition (status `True`,
reason `MemoryHighRelaxed`) and emits a warning event. The condition is
queryable and watchable, allowing external controllers (e.g., VPA, custom
operators) to detect pressure and trigger remediation (such as in-place
pod resize) before the kubelet proceeds to stage 2 eviction. The condition
is cleared when pressure subsides before the grace period expires.
The event reflects a successful CRI call, not a confirmed cgroup write.

### Metrics

| Metric | Type | Description |
|--------|------|-------------|
| `kubelet_evictions{eviction_signal="memory.high.pressure"}` | Counter | Pods evicted due to sustained memory pressure (stage 2) |
| `kubelet_memory_high_relaxed_total` | Counter | Containers where `memory.high` relaxation was attempted via CRI (stage 1) |

### Prior Art

| System | Signal | Scope | Deployment | Two-stage? |
|--------|--------|-------|------------|------------|
| [systemd-oomd](https://www.man7.org/linux/man-pages/man8/systemd-oomd.8.html) | PSI avg10, swap usage | cgroup slice | Standalone daemon | No (kill only) |
| [Facebook oomd](https://engineering.fb.com/2018/07/19/production-engineering/oomd/) | PSI, pgscan rate | cgroup | Standalone daemon | No (kill only) |
| [ByteDance Katalyst](https://www.cncf.io/blog/2024/04/25/how-katalyst-guarantees-memory-qos-for-colocated-applications/) | PSI avg10, kswapd steal rate, free memory watermarks | Node/NUMA | Separate DaemonSet | No (kill only) |
| **This KEP** | `memory.events:high/max` rate, PSI avg10 | Per-container | Inside kubelet | **Yes (relax then evict)** |

None of the above use `memory.events` counters for detection or attempt
remediation before killing.

### Future: EvictionRequest API Integration

[KEP-4563 (EvictionRequest API)](https://github.com/kubernetes/enhancements/issues/4563)
is also targeting v1.37 alpha. At alpha, both features ship independently:
this feature uses the kubelet's pod eviction path directly, same as all
evictions. Coupling two alpha features would create fragility since
EvictionRequest's API may change before beta.

At beta, the stage 2 eviction step routes through EvictionRequest, giving
interceptors (checkpoint, migration, connection draining) time to act
before the pod is killed. The two-stage remediation is complementary:
stage 1 relaxes memory.high for immediate relief, while EvictionRequest
handles graceful coordination if stage 2 is needed.

This feature does not set any fields that EvictionRequest interceptors
consume. The `DisruptionTarget` pod condition is for operator
observability, not interceptor triggering. Concurrent `kubectl drain`
operations are safe since pod workers serialize kill requests from
different sources.

### Test Plan

[x] I/we understand the owners of the involved components may require updates
to existing tests to make this code solid enough prior to committing the
changes necessary to implement this enhancement.

##### Prerequisite testing updates

No prerequisite testing updates required.

##### Unit tests

- `pkg/kubelet/eviction/`: eviction manager and helpers
- `pkg/kubelet/stats/`: stats pipeline helpers

##### Integration tests

n/a: plan to use node e2e tests (see below)

##### e2e tests

e2e_node tests on cgroup v2 nodes covering: stage 1 remediation
(memory.high relaxed after sustained throttling), stage 2 eviction
(pod evicted when pressure persists after relaxation), grace period
behavior (transient spikes do not trigger action), and feature gate
disable (no action when gate is OFF).

### Graduation Criteria

#### Alpha (v1.37)

- Feature gate `MemoryHighEviction` implemented (default OFF)
- Three detection signals: `memory.events:high` rate, `memory.events:max`
  rate, PSI `memory.pressure` avg10
- Two-stage remediation: relax `memory.high`, then evict
- `KubeletConfiguration` fields for per-signal thresholds and grace period
- Stats API extension for `memory.events` data
- Container restart detection and counter reset handling
- QoS + priority based candidate ranking
- `ContainerMemoryPressure` pod condition at stage 1 (cleared when pressure subsides)
- `DisruptionTarget` condition and OffendingContainers annotation at stage 2 (eviction)
- Unit tests (10 cases) and e2e_node tests on cgroup v2 nodes
- `kubelet_evictions` and `kubelet_memory_high_relaxed_total` metrics

#### Beta

- Feature gate default ON
- Node-level PSI eviction: evict the most pressured pod when node-wide
  PSI avg10 exceeds a configured threshold, providing kubelet-native
  equivalent of systemd-oomd using the PSI pipeline from KEP-4205
- QoS-tiered stage 1 policy (e.g., skip relaxation for BestEffort)
- Multi-signal weighted scoring (configurable weights instead of simple OR)
- Per-pod opt-out annotation
- EvictionRequest API integration for PDB respect and interceptor support
  (if KEP-4563 is available)
- VPA signal: annotation so VPA can use eviction events to boost memory
  recommendations
- Production feedback from alpha users, data-driven default thresholds

#### GA

- Beta for at least 2 releases
- Production adoption evidence
- PDB-aware eviction via EvictionRequest API
- InPlacePodResize integration: attempt resize UP via API before evicting
  (requires resize controller or VPA)
- Conformance tests if applicable
- Stable defaults informed by production data

### Upgrade / Downgrade Strategy

This is a kubelet-only feature with no API server, scheduler, or controller
manager changes.

**Upgrade**: Enable the `MemoryHighEviction` feature gate and restart the
kubelet. The eviction manager begins monitoring on the next synchronize
cycle. No existing workloads are affected until they exhibit sustained
pressure above the configured thresholds.

**Downgrade**: Disable the feature gate and restart the kubelet. All
monitoring and remediation stops immediately. `memory.high` values that were
relaxed (stage 1) remain at `memory.max` until the container restarts or
is resized. No other persistent state remains.

### Version Skew Strategy

This feature is kubelet-only. No cross-component coordination is needed.
In a heterogeneous cluster, nodes with the gate ON perform pressure-based
remediation; nodes with it OFF do not.

The stats API extension (`memory.events` in the Summary API) is additive.
Older clients will not see the new `Events` field. PSI data in the Summary
API is already GA.

No CRI changes are required. `memory.events` comes from cadvisor reading
cgroup v2 files. PSI is available through both cadvisor and CRI-native
stats providers.

## Production Readiness Review Questionnaire

### Feature Enablement and Rollback

###### How can this feature be enabled / disabled in a live cluster?

- [x] Feature gate (also fill in values in `kep.yaml`)
  - Feature gate name: MemoryHighEviction
  - Components depending on the feature gate: kubelet
  - Will enabling / disabling the feature require downtime of the control
    plane? No.
  - Will enabling / disabling the feature require downtime or reprovisioning
    of a node? Yes, kubelet restart is required to toggle the feature gate.

###### Does enabling the feature change any default behavior?

No, not by default. The feature gate is required, but all thresholds
default to 0 (disabled). No detection or eviction occurs until
the operator explicitly configures at least one threshold. Once thresholds
are set, containers under sustained memory pressure above the configured
values will receive two-stage remediation (stage 1: relax `memory.high`,
stage 2: evict if pressure persists).

###### Can the feature be disabled once it has been enabled (i.e. can we roll back the enablement)?

Yes. Disabling the feature gate and restarting the kubelet stops all
pressure monitoring and remediation immediately. Containers whose
`memory.high` was relaxed (stage 1) retain the relaxed value until the
container restarts or is resized, at which point MemoryQoS (if still
enabled) recomputes the original value from the formula. No other
persistent state exists.

###### What happens if we reenable the feature if it was previously rolled back?

All tracking state starts fresh on kubelet restart. Grace period timers
reset. If a container's `memory.high` was previously relaxed and is still
throttling, the detection flow starts from the beginning. The grace period
must elapse again before any action is taken.

###### Are there any tests for feature enablement/disablement?

Unit tests verify that the memory pressure eviction check is a no-op when the
feature gate is disabled, and that it correctly detects and responds when
enabled. e2e_node tests cover the gate toggle path.

### Rollout, Upgrade and Rollback Planning

###### How can a rollout or rollback fail? Can it impact already running workloads?

If thresholds are misconfigured (negative values, or PSI threshold > 100),
kubelet config validation rejects them at startup. If MemoryQoS is OFF,
`memory.events` signals are no-ops (PSI still works independently). If the
container runtime does not support stage 1 updates, detection and stage 2
still function normally.

Impact on running workloads: only containers under sustained pressure above
the configured thresholds are affected. Containers operating normally are
never touched. Stage 1 (relaxing `memory.high`) is non-destructive.

Rollback: disable the gate, restart kubelet. No active cleanup is required.
Any `memory.high` values that were relaxed during stage 1 remain at the
relaxed level until the container is restarted or resized, at which point
the kubelet recomputes `memory.high` from the MemoryQoS formula without
the eviction feature and the cgroup value returns to its original setting.
In-flight `ContainerMemoryPressure` conditions set during stage 1 are
cleared on the next kubelet sync when pressure is no longer detected.

###### What specific metrics should inform a rollback?

- `kubelet_evictions{eviction_signal="memory.high.pressure"}`: if
  evictions are too frequent, increase thresholds or grace period
- `kubelet_memory_high_relaxed_total`: if stage 1 triggers too often,
  the `memoryThrottlingFactor` in MemoryQoS may need adjustment

###### Were upgrade and rollback tested? Was the upgrade->downgrade->upgrade path tested?

Unit tests cover enable/disable transitions. e2e_node tests cover the
gate toggle path (enable, create pod, disable, verify behavior).

###### Is the rollout accompanied by any deprecations and/or removals of existing features?

No.

### Monitoring Requirements

###### How can an operator determine if the feature is in use by workloads?

Check `kubelet_memory_high_relaxed_total` (stage 1 triggered) and
`kubelet_evictions{eviction_signal="memory.high.pressure"}` (stage 2
triggered). Non-zero values indicate the feature is actively responding
to memory pressure. Pod events with "memory.high" in the message provide
per-pod visibility.

###### How can someone using this feature know that it is working for their instance?

Verify cadvisor exposes `container_memory_events_high_total` (non-zero for
throttled containers). Check Summary API for non-nil `Memory.Events` and
`Memory.PSI` fields. If throttling is occurring and the feature is enabled,
kubelet logs will show stage 1 or stage 2 actions.

###### What are the reasonable SLOs (Service Level Objectives) for the enhancement?

No specific SLO. The feature is operator-tuned via thresholds and grace
period. A reasonable expectation: containers under sustained pressure for
longer than the grace period have `memory.high` relaxed within one
eviction check cycle (~10s) after the grace period expires.

###### What are the SLIs (Service Level Indicators) an operator can use to determine the health of the service?

- `kubelet_memory_high_relaxed_total`: rate of stage 1 remediations
- `kubelet_evictions{eviction_signal="memory.high.pressure"}`: rate of
  stage 2 evictions
- `container_memory_events_high_total`: rate of throttle events per
  container (cadvisor metric)

###### Are there any missing metrics that would be useful to have in this KEP?

For beta, consider:
- `kubelet_memory_pressure_detected_total`: number of containers where
  pressure was detected but within the grace period (useful for threshold
  tuning)

### Dependencies

###### Does this feature depend on any specific services running in the cluster?

- **MemoryQoS feature gate**: Must be enabled for `memory.high` to be set
  and for `memory.events:high/max` signals to function. PSI works
  independently.
- **cgroups v2**: Required for `memory.events` and `memory.pressure` files.
- **cadvisor**: Reads `memory.events` from cgroup v2 files. Vendored
  in-process, not an external dependency. PSI is also available via
  CRI-native stats.

### Scalability

###### Will enabling / using this feature result in any new API calls?

No new Kubernetes API server or cloud-provider calls. Stage 1 uses one CRI
`UpdateContainerResources` call per remediated container. Stage 2 eviction
uses the same pod deletion path as existing eviction signals.

###### Will enabling / using this feature result in introducing new API types?

No.

###### Will enabling / using this feature result in any new calls to the cloud provider?

No.

###### Will enabling / using this feature result in increasing size or count of the existing API objects?

The `MemoryEventsStats` struct adds two optional uint64 fields per container
in the Summary API response. Negligible size increase.

###### Will enabling / using this feature result in increasing time taken by any operations covered by existing SLIs/SLOs?

No. The additional work per eviction check cycle is O(containers): read
counters from existing stats, compute rates (integer arithmetic), compare to
thresholds. Negligible compared to the eviction manager's existing work.

###### Will enabling / using this feature result in non-negligible increase of resource usage (CPU, RAM, disk, IO, ...) in any components?

No. Tracking state is O(containers) for counter baselines and start times,
plus O(pods) for grace period timers. CPU cost is integer arithmetic per
eviction check cycle (default 10 seconds).

###### Can enabling / using this feature result in resource exhaustion of some node resources (PIDs, sockets, inodes, etc.)?

No. The feature reads existing stats and writes to cgroup files only during
stage 1 remediation (one `memory.high` write per affected container).

### Troubleshooting

###### How does this feature react if the API server and/or etcd is unavailable?

Detection and stage 1 remediation are kubelet-local and do not depend on the
API server. Stage 2 eviction needs to update pod status via the API server.
If unavailable, eviction fails and retries on the next eviction check cycle,
same as existing eviction behavior.

###### What are other known failure modes?

| Failure Mode | Detection | Mitigation |
|---|---|---|
| False positive (transient pressure triggers remediation) | `kubelet_memory_high_relaxed_total` increases faster than expected | Increase grace period. Stage 1 is non-destructive since it only relaxes `memory.high`, returning to the pre-MemoryQoS baseline. |
| MemoryQoS gate OFF (memory.events signals are no-ops) | Zero `memory.events`-based actions despite throttled pods. `container_memory_events_high_total` is zero. | Enable MemoryQoS. PSI signal still works independently. |
| Container restart (counter reset) | After restart, throttling in the new container is not detected for one eviction check cycle. | Track container start time. Reset counters on change. |
| PSI persistence after InPlacePodResize | PSI avg10 remains elevated for ~10 seconds after resize. | Grace period absorbs the transient PSI spike. |
| CRI-only stats provider (memory.events unavailable) | PSI signal still works. Feature is partially functional. | Linux only. Only affects future CRI-only mode without cadvisor. |
| Runtime does not apply Unified map on update | Stage 1 appears to succeed but cgroup value is unchanged. Throttling persists, stage 2 triggers on next cycle. | Upgrade CRI-O to v1.35.3+ or use containerd. |
| Kernel < 5.9 (more severe throttling) | Higher throttle rates detected. Stage 1 and 2 trigger more frequently. | Beneficial because it prevents indefinite livelock on older kernels. |
| Eviction-recreation loop | Pod evicted, recreated on same node, same pressure. | Grace period bounds loop frequency. Stage 1 reduces eviction rate. Beta: VPA + InPlacePodResize integration. |

###### What steps should be taken if SLOs are not being met to determine the root cause?

1. Check `kubelet_memory_high_relaxed_total` and
   `kubelet_evictions{eviction_signal="memory.high.pressure"}` for action
   frequency.
2. Check `container_memory_events_high_total` for per-container throttle
   rates.
3. Check kubelet logs for "memory.high" remediation messages.
4. If actions are too aggressive: increase thresholds and/or grace period.
5. If actions are not happening: verify MemoryQoS is ON, verify
   `memory.high` is set on container cgroups, verify cadvisor is reading
   `memory.events` (check Summary API for non-nil `Events` field).

## Implementation History

- 2026-05-28: KEP created, targeting v1.37 alpha
- 2026-05-22: Enhancement issue [#5986](https://github.com/kubernetes/enhancements/issues/5986)
  created, lead opted-in (haircommander), milestone v1.37

### Prior Work

- [google/cadvisor#3870](https://github.com/google/cadvisor/pull/3870):
  Expose `container_memory_events_high_total` and `_max_total` Prometheus
  counters (merged)
- [kubernetes/kubernetes#139157](https://github.com/kubernetes/kubernetes/pull/139157):
  Vendor cadvisor v0.57.0 with memory.events metrics (merged)
- [kubernetes/kubernetes#139178](https://github.com/kubernetes/kubernetes/pull/139178):
  Add memory.events metrics to container_metrics e2e test

## FAQ

### Why are memory.events counters needed when PSI already detects pressure?

PSI cannot distinguish throttling caused by memory.high from normal page
cache reclaim. Stage 1 (relaxing memory.high) would fire on false positives
if PSI alone decided when to relax.

### Why does stage 1 modify cgroup state from the eviction manager?

The value being written (memory.high) was set by the kubelet via MemoryQoS.
Relaxing it undoes our own constraint, not a user-specified limit.

### Why do all thresholds default to 0?

No production data exists yet to pick safe defaults. Beta will ship with
data-driven defaults once alpha feedback is available.

### How does the feature behave when MemoryQoS is disabled?

PSI detection and stage 2 eviction still work. Stage 1 has nothing to
relax since memory.high was never set.

### How is the eviction-recreation loop mitigated?

Stage 1 resolves most cases without eviction. When eviction does happen,
the grace period bounds the loop. Beta targets VPA integration so the
replacement pod gets a higher memory recommendation.

### How does this feature co-exist with EvictionRequest API (KEP-4563)?

This feature decides *when* to evict based on memory pressure signals.
EvictionRequest API will provide the *mechanism* for graceful eviction
with PDB respect and interceptor support. At alpha, both features ship
independently since EvictionRequest's API is not yet stable. At beta,
the stage 2 eviction step will route through EvictionRequest, giving
interceptors time to checkpoint or migrate the workload before the pod
is terminated.

### How will VPA learn about memory pressure evictions?

Evicted pods will receive a DisruptionTarget condition and
OffendingContainers annotation identifying the throttled container.
VPA does not consume these signals today. At beta, we plan to make
these available so VPA can adjust memory recommendations for
replacement pods.

## Drawbacks

- Adds complexity to the eviction manager with a new parallel check and
  two-stage remediation.
- Three detection signals and four configuration fields increase operator
  cognitive load.
- Rate-based and percentage-based detection is new to the eviction
  framework since all existing signals are quantity-based.
- Requires MemoryQoS as a prerequisite for `memory.events` signals.
- Stage 1 (relaxing `memory.high`) modifies container cgroup state, unlike
  existing eviction which is read-only until the kill.

## Alternatives

### PSI-Only Detection (No memory.events)

Use only PSI `memory.pressure` for detection, matching systemd-oomd's
approach.

**Why not**: PSI measures aggregate memory pressure from all causes (page
cache reclaim, NUMA balancing, swap activity), not specifically `memory.high`
breaches. Since this KEP specifically targets MemoryQoS-related throttling,
`memory.events:high` is the precise signal. PSI is included as a
complementary signal to catch memory.min/low contention that `memory.events`
misses.

### Eviction-Only (No Stage 1 Remediation)

Skip relaxing `memory.high` and evict directly when pressure is detected.
This is what Facebook oomd, systemd-oomd, and ByteDance Katalyst do.

**Why not**: The throttling is caused by `memory.high`, a limit the kubelet
set via MemoryQoS, not the user's limit (`memory.max`). Removing our own
constraint before killing the workload avoids unnecessary evictions when
the workload fits within `memory.max` after relaxation.

### New Eviction Signal in the Signal Framework

Add a `memory.high.events` signal to `signalObservations`.

**Why not**: The signal framework uses `signalObservation` with `capacity`
and `available` fields (absolute quantities). Rate-based and percentage-based
metrics do not fit. `localStorageEviction` provides the established precedent
for per-container checks outside the signal framework.

### systemd-oomd Integration

Integrate with systemd-oomd for memory pressure detection and killing.

**Why not**: systemd-oomd sends SIGKILL at the cgroup level with no pod
conditions, events, or remediation. It does not understand pod lifecycle
or rescheduling. At beta, node-level PSI eviction provides equivalent
functionality within the kubelet.

### Absolute Threshold Instead of Rate-Based

Evict when the `memory.events:high` counter exceeds an absolute value.

**Why not**: A counter value does not indicate severity. A container with
1000 events over 24 hours differs from one with 1000 events in 10 seconds.
Rate (events/sec) correlates with throttling intensity.
