# KEP-5759: Memory Manager Hugepages Availability Verification

<!-- toc -->
- [Release Signoff Checklist](#release-signoff-checklist)
- [Summary](#summary)
- [Motivation](#motivation)
  - [The Tracking Gap](#the-tracking-gap)
  - [Real-World Example](#real-world-example)
  - [Goals](#goals)
  - [Non-Goals](#non-goals)
- [Proposal](#proposal)
  - [Current Admission Flow](#current-admission-flow)
  - [User Stories](#user-stories)
    - [Story 1: DPDK Application Admission Failure](#story-1-dpdk-application-admission-failure)
    - [Story 2: Rapid Pod Churn with Hugepages](#story-2-rapid-pod-churn-with-hugepages)
  - [Notes/Constraints/Caveats](#notesconstraintscaveats)
  - [Risks and Mitigations](#risks-and-mitigations)
- [Design Details](#design-details)
  - [Implementation Overview](#implementation-overview)
  - [cadvisor Changes](#cadvisor-changes)
  - [Memory Manager Changes](#memory-manager-changes)
  - [Integration with Topology Manager](#integration-with-topology-manager)
  - [Interaction with CPU Manager](#interaction-with-cpu-manager)
  - [Observability](#observability)
  - [Test Plan](#test-plan)
      - [Prerequisite testing updates](#prerequisite-testing-updates)
      - [Unit tests](#unit-tests)
      - [Integration tests](#integration-tests)
      - [e2e tests](#e2e-tests)
  - [Graduation Criteria](#graduation-criteria)
    - [Alpha](#alpha)
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
- [Drawbacks](#drawbacks)
- [Alternatives](#alternatives)
  - [Alternative 1: Track all pod hugepage usage](#alternative-1-track-all-pod-hugepage-usage)
  - [Alternative 2: Query sysfs directly in Memory Manager](#alternative-2-query-sysfs-directly-in-memory-manager)
  - [Alternative 3: Scheduler-level hugepage awareness](#alternative-3-scheduler-level-hugepage-awareness)
<!-- /toc -->

## Release Signoff Checklist

Items marked with (R) are required *prior to targeting to a milestone / release*.

- [x] (R) Enhancement issue in release milestone, which links to KEP dir in [kubernetes/enhancements] (not the initial KEP PR)
  - Enhancement issue: https://github.com/kubernetes/enhancements/issues/5759
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

[kubernetes.io]: https://kubernetes.io/
[kubernetes/enhancements]: https://git.k8s.io/enhancements
[kubernetes/kubernetes]: https://git.k8s.io/kubernetes
[kubernetes/website]: https://git.k8s.io/website

## Summary

This KEP proposes enhancing the Memory Manager's Static policy to verify OS-reported
free hugepages availability during pod admission. Currently, the Memory Manager only
tracks hugepage allocations for Guaranteed QoS pods but doesn't verify actual
hugepage availability from the operating system. This can lead to pods being admitted
when hugepages aren't actually available, causing runtime failures.

The enhancement adds verification by reading free hugepages from sysfs
(`/sys/devices/system/node/node<N>/hugepages/hugepages-<size>kB/free_hugepages`)
during pod admission, ensuring pods requesting hugepages are only admitted when
sufficient free hugepages exist.

## Motivation

The Memory Manager's Static policy tracks hugepage allocations for Guaranteed QoS
pods to provide NUMA-aware memory and hugepage pinning. However, it operates on
its internal accounting without verifying the actual state of hugepages on the
system.

### The Tracking Gap

The Kubernetes scheduler tracks hugepages at the **node level** - it knows total
hugepage capacity and allocated amounts per node. The Memory Manager's Static
policy tracks hugepages at the **per-NUMA level**, but only for Guaranteed QoS
pods that it manages for NUMA placement.

This creates a tracking gap: **Burstable pods can legitimately request hugepages
through standard Kubernetes resource requests** (e.g., `hugepages-2Mi: 1Gi`).
These requests are:
- Properly validated by the scheduler
- Correctly configured in cgroup limits
- Accounted for at the node level

However, the Memory Manager does not track these Burstable pod allocations for
NUMA placement purposes. When a subsequent Guaranteed pod requests hugepages:
1. The scheduler approves it (node-level accounting shows availability)
2. The Memory Manager's internal state shows hugepages as available
3. But the OS has already allocated those hugepages to the Burstable pod
4. The Guaranteed pod fails at runtime when hugepages are exhausted

### Real-World Example

From [issue #134395](https://github.com/kubernetes/kubernetes/issues/134395),
on an m6id.32xlarge instance with 2 NUMA nodes:

```
Memory Manager internal state: 15.2 GB free hugepages
Actual OS state (sysfs):       3.2 GB free hugepages
```

The 12GB discrepancy was due to Burstable pods consuming hugepages that the
Memory Manager wasn't tracking.

### Goals

- Verify OS-reported free hugepages during pod admission for the Static policy
- Reject pods requesting hugepages when insufficient free hugepages are available
- Provide clear error messages when admission fails due to insufficient hugepages
- Maintain backwards compatibility with existing Memory Manager behavior

### Non-Goals

- Track hugepage usage by Burstable or BestEffort pods in the Memory Manager
- Modify scheduler behavior or add hugepage awareness to the scheduler
- Provide hugepage reservation or preemption mechanisms
- Support platforms other than Linux

## Proposal

Enhance the Memory Manager's Static policy to verify actual hugepage availability
by querying sysfs during pod admission. This involves:

1. **cadvisor enhancement**: Add a `FreePages` field to `HugePagesInfo` struct
   that reports free hugepages per NUMA node, read from sysfs

2. **Memory Manager enhancement**: During `Allocate()` in the Static policy,
   verify that OS-reported free hugepages meet or exceed the requested amount
   before admitting the pod

### Current Admission Flow

Understanding where this enhancement fits in the existing admission flow:

1. **Scheduler**: Checks node-level hugepage capacity and allocations. Ensures
   the node has sufficient total hugepages for the pod's request.

2. **Kubelet Admission**: When a pod is assigned to a node, kubelet performs
   local admission checks including resource availability.

3. **Memory Manager (Static policy)**: For Guaranteed QoS pods, the Memory
   Manager's `Allocate()` function:
   - Checks its internal state for available hugepages per NUMA node
   - Selects NUMA nodes for the allocation
   - Updates its internal tracking
   - **Gap**: Does not verify actual OS-reported free hugepages

4. **Container Runtime**: Creates the container with cgroup limits set. If
   hugepages are not actually available, the container fails at startup.

**This KEP addresses the gap in step 3** by adding OS-level verification before
updating internal tracking.

### User Stories

#### Story 1: DPDK Application Admission Failure

As a cluster administrator running DPDK-based network functions, I deploy a
Burstable pod that requests `hugepages-1Gi: 2Gi` for DPDK packet buffer pools.
Later, I deploy a Guaranteed pod also requesting `hugepages-1Gi: 2Gi`.

**Current behavior**: The Guaranteed pod is admitted (Memory Manager shows
hugepages as available) but fails at container startup when DPDK tries to allocate
hugepages that are already consumed by the Burstable pod.

**Desired behavior**: The Guaranteed pod admission fails immediately with a clear
error indicating insufficient free hugepages, allowing the scheduler to try
another node or the administrator to take corrective action.

#### Story 2: Rapid Pod Churn with Hugepages

As a platform engineer, I run batch jobs that use hugepages. Multiple jobs complete
and new jobs start in quick succession:

1. Node has 8GB of 2MB hugepages total
2. Burstable Job A (requests 4GB hugepages) completes, releasing hugepages
3. Guaranteed Job B (requests 6GB hugepages) is scheduled to this node
4. Before Job B's container starts, Burstable Job C (requests 4GB hugepages) starts
5. Job C's container allocates hugepages from the OS

**Current behavior**: The scheduler approved Job B based on node capacity (8GB).
Memory Manager's internal state (tracking only Guaranteed pods) shows 8GB available.
Job B is admitted, but when its container starts, only 4GB are actually free.
Job B fails at runtime.

**Desired behavior**: Memory Manager reads sysfs during admission and sees only
4GB free. Job B is rejected with error:
`insufficient hugepages-2Mi on NUMA node(s) [0,1]: requested 6Gi, available 4Gi`

Job B can be rescheduled to another node with sufficient hugepages.

### Notes/Constraints/Caveats

- **Race condition window**: A window exists between verification and actual
  container startup where hugepages could be consumed by another process. This is
  inherent to any admission-time check.

  **What happens if verification passes but container still fails?**
  1. Container startup fails with OOM or hugepage allocation error
  2. Kubelet emits `FailedCreatePodContainer` event with details
  3. Pod enters `CrashLoopBackOff` or `Error` state
  4. Scheduler may reschedule to another node (if applicable)

  **Why this is still valuable**: Without verification, the failure window spans
  from pod scheduling to container startup (seconds to minutes). With verification,
  the window is reduced to milliseconds between sysfs read and container start.
  The vast majority of failures are prevented.

- **Linux-only**: This feature is Linux-specific. The sysfs interface for hugepages
  (`/sys/devices/system/node/node<N>/hugepages/`) is a Linux kernel feature.
  On Linux systems where hugepages are configured, this sysfs interface is always
  available.

- **Per-NUMA verification**: Verification is performed per-NUMA node, consistent
  with the Memory Manager's NUMA-aware design and Topology Manager coordination.

- **Static policy only**: Verification only applies when Memory Manager's Static
  policy is enabled. With the "None" policy, Memory Manager doesn't track hugepage
  allocations at all, so there's no internal state to become stale. The scheduler's
  node-level tracking is the only safeguard with the None policy.

### Risks and Mitigations

| Risk | Mitigation |
|------|------------|
| sysfs reads add latency to admission | Minimal impact: single file read per hugepage size per NUMA node; < 1ms typically |
| False rejections due to transient consumption | Acceptable: better to reject than admit and fail at runtime; pod can be rescheduled |
| Verification passes but container still fails (race) | Window is milliseconds vs seconds/minutes without verification; event emitted for debugging |
| Fresh sysfs reads on every Allocate() | Lightweight operation; only triggered for pods requesting hugepages |

## Design Details

### Implementation Overview

The implementation consists of two parts:

1. **cadvisor**: Add `FreePages uint64` field to `HugePagesInfo` struct, populated
   from sysfs. Also expose a method to read current free hugepages on-demand.

2. **kubelet Memory Manager**: Add `verifyOSHugepagesAvailability()` function
   called during `Allocate()` that reads **fresh** hugepage availability from sysfs.

**Important**: cadvisor's `GetMachineInfo()` is called once at startup and cached.
The `FreePages` field in cached machine info would be stale. Therefore, verification
must read sysfs directly during each `Allocate()` call, not rely on cached values.
We will add a `GetCurrentHugepagesInfo()` method to cadvisor's `Manager` interface
that performs a fresh sysfs read.

### cadvisor Changes

**Struct update**:
```go
type HugePagesInfo struct {
    // huge page size (in kB)
    PageSize uint64 `json:"page_size"`
    // number of huge pages
    NumPages uint64 `json:"num_pages"`
    // number of free huge pages
    FreePages uint64 `json:"free_pages"`
}
```

**New method on Manager interface**:
```go
// GetCurrentHugepagesInfo returns fresh hugepage info per NUMA node by reading sysfs.
// This is separate from GetMachineInfo() which returns cached startup data.
func (m *manager) GetCurrentHugepagesInfo() (map[int][]HugePagesInfo, error)
```

The `FreePages` field is populated by reading from:
```
/sys/devices/system/node/node<N>/hugepages/hugepages-<size>kB/free_hugepages
```

**Note on reserved hugepages**: Linux tracks `resv_hugepages` (reserved but not
yet faulted). For this implementation, we use `free_hugepages` directly because:
- Reserved pages are committed to specific processes
- A new pod cannot use reserved pages
- `free_hugepages` accurately reflects what's available for new allocations

**Note**: Since sysfs is always available on Linux systems with hugepages configured,
we use a simple `uint64` rather than a pointer. A value of 0 means zero free
hugepages are available.

### Memory Manager Changes

During `Allocate()` in the Static policy:

```go
func (p *staticPolicy) verifyOSHugepagesAvailability(
    candidateNUMANodes []int,  // NUMA nodes selected by allocation algorithm
    pod *v1.Pod,
    container *v1.Container,
) error {
    // 1. Call cadvisor's GetCurrentHugepagesInfo() to get fresh sysfs data
    // 2. For each hugepage size requested by the container:
    //    a. Sum free hugepages across candidateNUMANodes only
    //    b. Compare against the requested amount
    // 3. Return error if insufficient, with detailed message
}
```

The verification:
- Only runs when the Static policy is enabled and feature gate is on
- Only checks hugepage resources (not regular memory)
- **Respects NUMA node selection**: Only checks the specific NUMA nodes that the
  Memory Manager's allocation algorithm has selected (see Topology Manager section)
- Returns admission error if insufficient free hugepages

**Error message format**:
```
insufficient hugepages-2Mi on NUMA node(s) [0]: requested 4Gi, available 2Gi
```

### Integration with Topology Manager

The Memory Manager works with Topology Manager to coordinate NUMA-aware resource
allocation. The verification must respect Topology Manager's policy:

| Topology Policy | Verification Behavior |
|-----------------|----------------------|
| `none` | Not applicable (Memory Manager Static policy requires topology-aware policies) |
| `best-effort` | Check aggregate across all candidate NUMA nodes |
| `restricted` | Check only NUMA nodes that satisfy topology constraints |
| `single-numa-node` | Check only the single selected NUMA node |

**Critical**: Verification happens **after** the Memory Manager's allocation algorithm
selects candidate NUMA nodes based on topology constraints. We verify against those
specific nodes, not all nodes on the system.

Example with `single-numa-node` policy:
```
Node topology: NUMA0 (2GB free), NUMA1 (3GB free)
Pod requests: 2GB hugepages
Allocation selects: NUMA0 (meets the request)
Verification checks: NUMA0 only → 2GB available ≥ 2GB requested ✓
```

Example where aggregate would be misleading:
```
Node topology: NUMA0 (1GB free), NUMA1 (1GB free)
Pod requests: 2GB hugepages with single-numa-node policy
Allocation fails: Neither NUMA node has 2GB alone
(Verification never reached - allocation algorithm rejects first)
```

### Interaction with CPU Manager

When CPU Manager pins a pod to specific CPUs, those CPUs belong to specific NUMA
nodes. Topology Manager coordinates this to ensure Memory Manager allocates from
the same NUMA node(s). The verification inherits this coordination because it
checks only the candidate NUMA nodes selected by the allocation algorithm.

### Observability

This feature provides explicit signals for operators to monitor hugepage verification:

#### Metrics

| Metric | Type | Description |
|--------|------|-------------|
| `memory_manager_hugepages_verification_total` | Counter | Total verification checks performed. Labels: `result` (success/failure), `hugepage_size` |
| `memory_manager_hugepages_verification_failures_total` | Counter | Pods rejected due to insufficient OS-reported hugepages. Labels: `hugepage_size`, `numa_node` |
| `memory_manager_hugepages_verification_latency_seconds` | Histogram | Time spent performing verification (buckets: 1ms to 100ms) |

#### Events

When a pod is rejected due to insufficient hugepages, a Kubernetes event is generated:

```
Type:    Warning
Reason:  FailedHugepagesVerification
Message: insufficient hugepages-2Mi on NUMA node(s) [0]: requested 4Gi, available 2Gi
```

#### Kubelet Logs

At `--v=4` or higher, kubelet logs verification details:
```
I0127 10:15:32.123456 12345 policy_static.go:XXX] "Verifying OS hugepages availability" pod="default/dpdk-app" container="dpdk"
I0127 10:15:32.123789 12345 policy_static.go:XXX] "Hugepages verification passed" pod="default/dpdk-app" numaNodes=[0] size="hugepages-2Mi" requested=1073741824 available=2147483648
```

#### Alerting Recommendations

Operators should consider alerts for:
- `rate(memory_manager_hugepages_verification_failures_total[5m]) > 0`: Pods being rejected
- `histogram_quantile(0.99, memory_manager_hugepages_verification_latency_seconds) > 0.05`: High verification latency

### Test Plan

[x] I/we understand the owners of the involved components may require updates to
existing tests to make this code solid enough prior to committing the changes necessary
to implement this enhancement.

##### Prerequisite testing updates

- Existing Memory Manager unit tests cover allocation logic
- cadvisor tests cover sysfs reading functionality

##### Unit tests

- `pkg/kubelet/cm/memorymanager`: Add tests for `verifyOSHugepagesAvailability()`
  - Test successful verification when free hugepages >= requested
  - Test rejection when free hugepages < requested
  - Test verification with zero free hugepages (FreePages = 0)
  - Test per-NUMA node verification respects candidate node selection
  - Test multiple hugepage sizes in same request
  - Test with feature gate enabled/disabled

##### Integration tests

- Test Memory Manager with mocked cadvisor returning various FreePages values
- Test admission flow with hugepage verification enabled/disabled

##### e2e tests

- Test pod admission when hugepages are available
- Test pod rejection when hugepages are exhausted
- Test that rejected pods can be admitted after hugepages are freed

### Graduation Criteria

#### Alpha

- Feature implemented behind `MemoryManagerHugepagesVerification` feature gate
- Unit tests for verification logic
- E2e tests demonstrating:
  - Pod admission succeeds when sufficient free hugepages exist
  - Pod admission fails when insufficient free hugepages exist
- Documentation for feature gate and behavior

#### Beta

- E2e tests demonstrating correct behavior
- Metrics for verification failures
- Feedback incorporated from alpha users
- No significant bugs reported

#### GA

- Feature enabled by default
- Conformance tests if applicable
- Documentation updated for stable feature

### Upgrade / Downgrade Strategy

**Upgrade**: No special handling required. The feature is additive and controlled
by a feature gate. Existing pods are unaffected.

**Downgrade**: Disabling the feature gate returns to previous behavior where
OS hugepage availability is not verified. No data migration needed.

**Kubelet restart behavior**: After kubelet restarts, Memory Manager rebuilds its
internal state from checkpoint. Since verification reads fresh sysfs data on each
`Allocate()` call, there's no stale state concern. New pod admissions after restart
will correctly verify against current OS hugepage availability.

### Version Skew Strategy

The feature is entirely within the kubelet and depends on cadvisor (vendored).
No control plane or cross-component version skew concerns.

Since cadvisor is vendored into kubelet, the kubelet and cadvisor versions are
always synchronized. The `FreePages` field will be available when the feature
gate is enabled.

## Production Readiness Review Questionnaire

### Feature Enablement and Rollback

###### How can this feature be enabled / disabled in a live cluster?

- [x] Feature gate
  - Feature gate name: `MemoryManagerHugepagesVerification`
  - Components depending on the feature gate: kubelet

###### Does enabling the feature change any default behavior?

Yes. Pods requesting hugepages may be rejected at admission if the OS reports
insufficient free hugepages, even if the Memory Manager's internal tracking
shows availability. This is the intended behavior to prevent runtime failures.

###### Can the feature be disabled once it has been enabled (i.e. can we roll back the enablement)?

Yes. Disabling the feature gate and restarting kubelet returns to previous
behavior. No persistent state is affected.

###### What happens if we reenable the feature if it was previously rolled back?

The feature resumes verification on new pod admissions. No special handling needed.

###### Are there any tests for feature enablement/disablement?

Yes. Unit tests will verify:
- When feature gate is disabled: verification is skipped, pods are admitted
  based on Memory Manager's internal tracking (existing behavior)
- When feature gate is enabled: verification is performed, pods are rejected
  if OS-reported free hugepages are insufficient

### Rollout, Upgrade and Rollback Planning

###### How can a rollout or rollback fail? Can it impact already running workloads?

The feature only affects pod admission, not running workloads. A rollout cannot
impact already running pods. Rollback simply stops verification on new admissions.

###### What specific metrics should inform a rollback?

- Unexpected increase in pod admission failures
- `memory_manager_hugepages_verification_failures_total` metric (proposed)

###### Were upgrade and rollback tested? Was the upgrade->downgrade->upgrade path tested?

TBD during alpha phase.

###### Is the rollout accompanied by any deprecations and/or removals of features, APIs, fields of API types, flags, etc.?

No.

### Monitoring Requirements

###### How can an operator determine if the feature is in use by workloads?

- Feature gate is enabled
- Pods request hugepages resources

###### How can someone using this feature know that it is working for their instance?

- [ ] Events
  - Event Reason: `FailedHugepagesVerification`
  - When: Pod admission rejected due to insufficient OS-reported free hugepages
- [ ] Other
  - Kubelet logs will indicate verification being performed and results

###### What are the reasonable SLOs (Service Level Objectives) for the enhancement?

- Hugepage verification should add < 10ms to pod admission latency
- 99.9% of pods with sufficient free hugepages should be admitted successfully

###### What are the SLIs (Service Level Indicators) an operator can use to determine the health of the service?

- [x] Metrics
  - Metric name: `memory_manager_hugepages_verification_total`
    - Components exposing the metric: kubelet
    - Description: Total number of hugepages verification checks performed
    - Labels: `result` (success, failure), `hugepage_size` (e.g., 2Mi, 1Gi)
  - Metric name: `memory_manager_hugepages_verification_failures_total`
    - Components exposing the metric: kubelet
    - Description: Total number of pods rejected due to insufficient OS-reported hugepages
    - Labels: `hugepage_size`, `numa_node`
  - Metric name: `memory_manager_hugepages_verification_latency_seconds`
    - Components exposing the metric: kubelet
    - Description: Histogram of time spent performing hugepages verification
    - Buckets: 0.001, 0.005, 0.01, 0.025, 0.05, 0.1 seconds

###### Are there any missing metrics that would be useful to have to improve observability of this feature?

Additional metrics that could be added in Beta:
- `memory_manager_hugepages_discrepancy_bytes`: Gauge showing difference between
  Memory Manager's internal tracking and OS-reported free hugepages (useful for
  detecting drift)

### Dependencies

###### Does this feature depend on any specific services running in the cluster?

- cadvisor (bundled with kubelet)
  - Usage: Provides machine info including hugepage free counts
  - Impact of outage: Verification skipped, graceful degradation
  - Impact of degraded performance: Slightly increased admission latency

### Scalability

###### Will enabling / using this feature result in any new API calls?

No new API calls. The feature reads from local sysfs and cadvisor machine info.

###### Will enabling / using this feature result in introducing new API types?

No.

###### Will enabling / using this feature result in any new calls to the cloud provider?

No.

###### Will enabling / using this feature result in increasing size or count of the existing API objects?

No.

###### Will enabling / using this feature result in increasing time taken by any operations covered by existing SLIs/SLOs?

Minimal impact on pod admission latency (< 10ms for sysfs reads).

###### Will enabling / using this feature result in non-negligible increase of resource usage (CPU, RAM, disk, IO, ...) in any components?

Negligible: periodic sysfs file reads during pod admission.

###### Can enabling / using this feature result in resource exhaustion of some node resources (PIDs, sockets, inodes, etc.)?

No. The feature performs simple file reads.

### Troubleshooting

###### How does this feature react if the API server and/or etcd is unavailable?

No impact. The feature operates entirely within kubelet using local sysfs.

###### What are other known failure modes?

- Verification rejects pods that would have succeeded
  - Detection: Increase in `memory_manager_hugepages_verification_failures_total`
    with pods eventually succeeding on retry
  - Mitigations: This indicates transient hugepage consumption; the feature is
    working correctly by preventing admission during contention
  - Diagnostics: Compare verification failure count with actual runtime failures
  - Testing: E2e tests verify this scenario

###### What steps should be taken if SLOs are not being met to determine the problem?

1. Check kubelet logs for verification-related messages
2. Review `memory_manager_hugepages_verification_latency_seconds` histogram
   for unusually slow verification
3. Compare Memory Manager state with actual sysfs values using:
   `cat /sys/devices/system/node/node*/hugepages/hugepages-*/free_hugepages`
4. Check for excessive pod admission rate causing contention

## Implementation History

- 2024-12-24: Initial KEP draft
- 2024-12-27: KEP updated based on reviewer feedback
- Enhancement issue: https://github.com/kubernetes/enhancements/issues/5759
- Related issue: https://github.com/kubernetes/kubernetes/issues/134395
- cadvisor PR: https://github.com/google/cadvisor/pull/3804

## Drawbacks

- Adds complexity to the admission path
- Small race window still exists between verification and container startup
- May reject pods that would have succeeded if hugepages were freed during startup

## Alternatives

### Alternative 1: Track all pod hugepage usage

Extend Memory Manager to track hugepage usage by Burstable and BestEffort pods.

**Rejected because**:
- Significant refactoring required
- Would not catch external (non-Kubernetes) hugepage consumers
- Changes the scope and purpose of Memory Manager

### Alternative 2: Query sysfs directly in Memory Manager

Read sysfs directly in Memory Manager without cadvisor changes.

**Rejected because**:
- Duplicates sysfs reading logic already in cadvisor
- cadvisor already provides machine info abstraction
- Adding to cadvisor benefits other consumers of machine info

### Alternative 3: Scheduler-level hugepage awareness

Add hugepage availability awareness to the Kubernetes scheduler.

**Rejected because**:
- Much larger scope change
- Scheduler operates on reported capacity, not real-time availability
- Does not solve the admission-time verification problem
