# KEP-NNNN: Memory Manager Hugepages Availability Verification

<!-- toc -->
- [Release Signoff Checklist](#release-signoff-checklist)
- [Summary](#summary)
- [Motivation](#motivation)
  - [Goals](#goals)
  - [Non-Goals](#non-goals)
- [Proposal](#proposal)
  - [User Stories](#user-stories)
    - [Story 1: DPDK Application Admission Failure](#story-1-dpdk-application-admission-failure)
    - [Story 2: Database Workload with Hugepages](#story-2-database-workload-with-hugepages)
  - [Notes/Constraints/Caveats](#notesconstraintscaveats)
  - [Risks and Mitigations](#risks-and-mitigations)
- [Design Details](#design-details)
  - [Implementation Overview](#implementation-overview)
  - [cadvisor Changes](#cadvisor-changes)
  - [Memory Manager Changes](#memory-manager-changes)
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

The Memory Manager tracks hugepage allocations for Guaranteed QoS pods to provide
NUMA-aware memory and hugepage pinning. However, it operates on its internal
accounting without verifying the actual state of hugepages on the system.

This creates a problem when:
1. Burstable or BestEffort pods consume hugepages (via hugetlbfs mounts or
   `mmap` with `MAP_HUGETLB`) without being tracked by the Memory Manager
2. External processes or other system components consume hugepages
3. The Memory Manager's internal state becomes stale or inconsistent with reality

In these scenarios, a Guaranteed pod requesting hugepages may be admitted based
on the Memory Manager's internal tracking, only to fail at runtime when the
container attempts to use the already-exhausted hugepages.

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

### User Stories

#### Story 1: DPDK Application Admission Failure

As a cluster administrator running DPDK-based network functions, I deploy a
Burstable pod that mounts hugetlbfs and consumes 2GB of 1GB hugepages for packet
buffer pools. Later, I deploy a Guaranteed pod also requesting 2GB of 1GB hugepages.

**Current behavior**: The Guaranteed pod is admitted (Memory Manager shows
hugepages as available) but fails at container startup when DPDK tries to allocate
hugepages that are already consumed.

**Desired behavior**: The Guaranteed pod admission fails immediately with a clear
error indicating insufficient free hugepages, allowing the scheduler to try
another node or the administrator to take corrective action.

#### Story 2: Database Workload with Hugepages

As a database administrator, I run PostgreSQL with hugepages enabled for shared
buffers. If an external monitoring agent or debugging tool temporarily consumes
hugepages, subsequent Guaranteed pods requesting hugepages should not be admitted
until hugepages are freed.

**Current behavior**: Pods are admitted based on Memory Manager tracking and fail
at runtime.

**Desired behavior**: Pods are rejected at admission with informative errors.

### Notes/Constraints/Caveats

- **Race condition window**: A small window exists between verification and actual
  container startup where hugepages could be consumed. This is inherent to any
  admission-time check but significantly reduces the failure window compared to
  no verification.

- **sysfs dependency**: The feature depends on reading from sysfs. If sysfs is
  unavailable or the free_hugepages file cannot be read, the feature gracefully
  degrades to current behavior (no verification).

- **Per-NUMA verification**: Verification is performed per-NUMA node, consistent
  with the Memory Manager's NUMA-aware design.

### Risks and Mitigations

| Risk | Mitigation |
|------|------------|
| sysfs reads add latency to admission | Minimal impact: single file read per hugepage size per NUMA node |
| False rejections due to transient consumption | Acceptable: better to reject than admit and fail at runtime |
| sysfs unavailable in some environments | Graceful degradation: skip verification if sysfs unreadable |

## Design Details

### Implementation Overview

The implementation consists of two parts:

1. **cadvisor**: Add `FreePages *uint64` field to `HugePagesInfo` struct, populated
   from sysfs. Uses pointer with `omitempty` to distinguish between "0 free" and
   "data unavailable".

2. **kubelet Memory Manager**: Add `verifyOSHugepagesAvailability()` function
   called during `Allocate()` that compares requested hugepages against OS-reported
   free hugepages from cadvisor's machine info.

### cadvisor Changes

```go
type HugePagesInfo struct {
    // huge page size (in kB)
    PageSize uint64 `json:"page_size"`
    // number of huge pages
    NumPages uint64 `json:"num_pages"`
    // number of free huge pages (nil if unavailable)
    FreePages *uint64 `json:"free_pages,omitempty"`
}
```

The `FreePages` field is populated by reading from:
```
/sys/devices/system/node/node<N>/hugepages/hugepages-<size>kB/free_hugepages
```

### Memory Manager Changes

During `Allocate()` in the Static policy:

```go
func (p *staticPolicy) verifyOSHugepagesAvailability(
    machineState state.NUMANodeMap,
    pod *v1.Pod,
    container *v1.Container,
) error {
    // For each hugepage size requested by the container:
    // 1. Get the OS-reported free hugepages from cadvisor machine info
    // 2. Compare against the requested amount
    // 3. Return error if insufficient
}
```

The verification:
- Only runs when the Static policy is enabled
- Only checks hugepage resources (not regular memory)
- Aggregates free hugepages across candidate NUMA nodes
- Returns admission error if insufficient free hugepages

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
  - Test graceful handling when FreePages is nil (sysfs unavailable)
  - Test per-NUMA node verification

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

### Version Skew Strategy

The feature is entirely within the kubelet and depends on cadvisor (vendored).
No control plane or cross-component version skew concerns.

When kubelet is upgraded but cadvisor hasn't been updated to provide `FreePages`:
- The field will be `nil`
- Verification will be skipped (graceful degradation)
- Warning logged indicating verification unavailable

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

Unit tests will verify behavior with feature gate enabled and disabled.

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
  - Metric name: `memory_manager_hugepages_verification_failures_total`
  - Components exposing the metric: kubelet
  - Metric name: `memory_manager_hugepages_verification_latency_seconds`
  - Components exposing the metric: kubelet

###### Are there any missing metrics that would be useful to have to improve observability of this feature?

The proposed metrics should provide adequate observability.

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

- sysfs unavailable or unreadable
  - Detection: Warning logs from kubelet, nil FreePages in machine info
  - Mitigations: Feature gracefully degrades to previous behavior
  - Diagnostics: Check kubelet logs for sysfs read warnings
  - Testing: Unit tests cover this scenario

###### What steps should be taken if SLOs are not being met to determine the problem?

1. Check kubelet logs for verification-related messages
2. Verify sysfs is accessible and free_hugepages files exist
3. Compare Memory Manager state with actual sysfs values
4. Check for excessive pod admission rate causing contention

## Implementation History

- 2024-12-24: Initial KEP draft
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
