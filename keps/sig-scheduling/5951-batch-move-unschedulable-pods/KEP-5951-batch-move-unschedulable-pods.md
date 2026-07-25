# KEP-5951: Batch Processing for MoveAllToActiveOrBackoffQueue

## Table of Contents

<!-- toc -->
- [Summary](#summary)
- [Motivation](#motivation)
  - [Goals](#goals)
  - [Non-Goals](#non-goals)
- [Proposal](#proposal)
  - [User Stories](#user-stories)
  - [Risks and Mitigations](#risks-and-mitigations)
- [Design Details](#design-details)
  - [Event Buffering Mechanism](#event-buffering-mechanism)
  - [Batch Processing Logic](#batch-processing-logic)
  - [Feature Gate](#feature-gate)
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
  - [Scalability](#scalability)
  - [Troubleshooting](#troubleshooting)
- [Implementation History](#implementation-history)
- [Drawbacks](#drawbacks)
- [Alternatives](#alternatives)
- [Infrastructure Needed](#infrastructure-needed)
<!-- /toc -->

## Summary

This KEP proposes adding a batch processing mechanism for `MoveAllToActiveOrBackoffQueue` events in the Kubernetes scheduler's priority queue. Instead of immediately processing each event to move unschedulable pods, events will be buffered and processed in batch every 1 second. This reduces lock contention and improves scheduling performance in high-throughput scenarios.

## Motivation

In large Kubernetes clusters with high pod churn, the scheduler's priority queue can become a bottleneck. Currently, every cluster event (such as Node addition, Pod deletion, or PVC binding) triggers an immediate call to `MoveAllToActiveOrBackoffQueue`, which:

1. Acquires the queue lock
2. Iterates through all unschedulable pods
3. Evaluates each pod against the event
4. Moves matching pods to activeQ or backoffQ

When many events occur in rapid succession, this creates significant lock contention, as the queue lock is held for the entire duration of each event processing. This can block other critical operations like `Pop()` and `Add()`.

By batching these events and processing them periodically, we can:
- Reduce the frequency of lock acquisitions
- Amortize the cost of iterating through unschedulable pods across multiple events
- Improve overall scheduling throughput

### Goals

- Reduce lock contention in the scheduler's priority queue
- Improve scheduling throughput in high-event-rate scenarios
- Maintain correctness of pod scheduling decisions
- Provide a safe fallback to the existing behavior

### Non-Goals

- Changing the scheduling algorithm or pod selection logic
- Modifying the queueing hint mechanism
- Altering the backoff or active queue behavior
- Adding new APIs or user-facing features

## Proposal

We propose introducing a feature gate `SchedulerBatchMoveUnschedulablePods` that, when enabled, changes how `MoveAllToActiveOrBackoffQueue` events are handled:

1. **Event Buffering**: Instead of immediately processing events, they are buffered in a `pendingMoveEvents` slice protected by a separate mutex (`pendingMoveEventsLock`).

2. **Periodic Flushing**: A background goroutine flushes the buffered events every 1 second via `flushPendingMoveEvents()`.

3. **Batch Processing**: During flush, all buffered events are processed together against all unschedulable pods. A pod matching ANY buffered event will be moved to activeQ or backoffQ.

4. **Backward Compatibility**: When the feature gate is disabled, the scheduler uses the existing immediate processing behavior.

### User Stories

#### Story 1: High-Throughput Cluster

As a cluster administrator running a large cluster with thousands of nodes and high pod churn, I want the scheduler to handle burst events efficiently without lock contention causing scheduling delays.

#### Story 2: Event Burst Scenarios

When many nodes are added simultaneously (e.g., during cluster autoscaling), instead of processing each node addition event separately and holding the queue lock multiple times, the scheduler should batch these events and process them together.

### Risks and Mitigations

| Risk | Impact | Mitigation |
|------|--------|------------|
| Increased latency for pod scheduling (up to 1 second delay) | Medium | The 1-second interval is chosen to balance latency vs. throughput. Users who need immediate processing can disable the feature gate. |
| Memory pressure from buffering too many events | Low | Events are small structs (event type + object references). With 1-second flush interval, memory overhead is minimal. |
| Incorrect scheduling decisions due to delayed event processing | Low | The batch processing logic maintains the same correctness guarantees as immediate processing. A pod matching any event will be moved. |
| Feature gate disablement during operation | Low | The implementation safely handles dynamic feature gate changes. When disabled, new events are processed immediately; buffered events are flushed on next cycle. |

## Design Details

### Event Buffering Mechanism

Two new fields are added to the `PriorityQueue` struct:

```go
// isBatchMoveUnschedulablePodsEnabled indicates whether the feature gate
// SchedulerBatchMoveUnschedulablePods is enabled.
isBatchMoveUnschedulablePodsEnabled bool

// pendingMoveEvents buffers events from MoveAllToActiveOrBackoffQueue calls.
// These events are processed in batch by flushPendingMoveEvents every 1 second.
pendingMoveEventsLock sync.Mutex
pendingMoveEvents     []pendingMoveEvent
```

The `pendingMoveEvent` struct stores the buffered event:

```go
type pendingMoveEvent struct {
    event    fwk.ClusterEvent
    oldObj   interface{}
    newObj   interface{}
    preCheck PreEnqueueCheck
}
```

### Batch Processing Logic

The `flushPendingMoveEvents()` function:

1. **Atomically drains** the pending events slice under `pendingMoveEventsLock`
2. **Acquires the main queue lock** (`p.lock`)
3. **Filters events** to only those that plugins are interested in
4. **Builds a candidate list** from unschedulable pods (to avoid map modification during iteration)
5. **Evaluates each candidate** against all buffered events:
   - Applies `preCheck` filter
   - Checks gating conditions
   - Calls `isPodWorthRequeuing()` to determine queueing strategy
   - A pod matching ANY event with `queueImmediately` strategy is moved immediately
   - Otherwise, if ANY event suggests `queueAfterBackoff`, it's moved to backoffQ
6. **Broadcasts** to waiting Pop() callers if any pods were activated

The key insight is that a pod only needs to match ONE event to be moved, and we take the best queueing strategy across all matching events.

### Feature Gate

A new feature gate `SchedulerBatchMoveUnschedulablePods` is added:

```go
// Enables batch processing of MoveAllToActiveOrBackoffQueue events.
// Instead of immediately moving unschedulable pods, events are buffered
// and processed in batch every 1 second to reduce lock contention.
SchedulerBatchMoveUnschedulablePods featuregate.Feature = "SchedulerBatchMoveUnschedulablePods"
```

Default settings:
- Alpha (v1.36): Disabled by default
- Beta (v1.37): Enabled by default
- GA (v1.39): Locked to enabled

### Test Plan

#### Prerequisite testing updates

- Existing unit tests for `MoveAllToActiveOrBackoffQueue` are updated to call `flushPendingMoveEvents()` after each event to ensure deterministic behavior when testing with the feature enabled.

#### Unit tests

The following test scenarios are covered:

1. **Basic batching**: Verify that events are buffered and flushed correctly
2. **Multiple events**: Verify that a pod matching any of multiple buffered events is moved
3. **Queueing strategy selection**: Verify that `queueImmediately` takes precedence over `queueAfterBackoff`
4. **PreCheck filtering**: Verify that preCheck filters are applied correctly during batch processing
5. **Gated pods**: Verify that gated pods are handled correctly with event matching
6. **Empty flush**: Verify that empty pending events don't cause issues
7. **Feature gate toggle**: Verify behavior when feature gate is dynamically enabled/disabled

Test files updated:
- `pkg/scheduler/backend/queue/scheduling_queue_test.go`

#### Integration tests

- Test scheduler behavior under high event load with and without the feature gate
- Measure scheduling throughput and latency differences
- Verify correctness of scheduling decisions

#### e2e tests

- Run existing scheduler e2e tests with the feature gate enabled
- No new e2e tests are required as this is an internal optimization

### Graduation Criteria

#### Alpha

- [x] Feature implemented behind `SchedulerBatchMoveUnschedulablePods` feature gate
- [x] Unit tests passing
- [x] Basic integration tests passing
- [x] Documentation updated

#### Beta

- [ ] Feature enabled by default
- [ ] Production workloads tested with feature enabled
- [ ] Performance benchmarks showing improvement
- [ ] No critical bugs reported for 2 releases

#### GA

- [ ] Feature used in production for at least 2 releases
- [ ] Comprehensive performance data collected
- [ ] All known issues resolved
- [ ] Feature gate locked to enabled

### Upgrade / Downgrade Strategy

**Upgrade**:
- When upgrading to a version with this feature, the feature gate defaults to disabled (Alpha) or enabled (Beta).
- No user action required.
- Existing scheduler behavior is preserved when feature is disabled.

**Downgrade**:
- When downgrading, if the feature was enabled, the scheduler will revert to immediate event processing.
- Any buffered events at the time of downgrade will be lost, but this is safe as:
  - Unschedulable pods remain in the unschedulable pool
  - Future events will trigger normal processing
  - The `podMaxInUnschedulablePodsDuration` timeout will eventually move pods

### Version Skew Strategy

This feature is internal to the kube-scheduler and does not affect API objects or inter-component communication. Version skew is not a concern.

## Production Readiness Review Questionnaire

### Feature Enablement and Rollback

###### How can this feature be enabled / disabled in a live cluster?

- [x] Feature gate
  - Feature gate name: SchedulerBatchMoveUnschedulablePods
  - Components depending on the feature gate: kube-scheduler

###### Does enabling the feature change any default behavior?

Yes. When enabled, events that would trigger immediate pod movement from unschedulable pool to active/backoff queues are instead buffered and processed in batch every 1 second. This may add up to 1 second of latency for pod scheduling in some cases.

###### Can the feature be disabled once it has been enabled (roll back)?

Yes, by disabling the feature gate. The scheduler will immediately revert to processing events synchronously.

###### What happens if we reenable the feature if it was previously rolled back?

The scheduler will resume buffering events and processing them in batch. No state is lost during the transition.

###### Are there any tests for feature enablement/disablement?

Yes, unit tests verify correct behavior with both feature gate states.

### Rollout, Upgrade and Rollback Planning

###### How can a rollout or rollback fail? Can it impact already running workloads?

Rollout/Rollback is controlled by the feature gate and is safe:
- Enabling: Events start being buffered; existing unschedulable pods are not affected
- Disabling: Buffered events are flushed; new events are processed immediately
- No impact on running workloads or scheduled pods

###### What specific metrics should inform a rollback?

- `scheduler_schedule_attempts_total`: Should not decrease significantly
- `scheduler_pending_pods`: Should not grow unbounded
- `scheduler_pod_scheduling_duration_seconds`: P99 latency should not increase significantly

###### Were upgrade and rollback tested? Was the upgrade->downgrade->upgrade path tested?

This will be tested before Beta graduation.

### Monitoring Requirements

###### How can an operator determine if the feature is in use by workloads?

Check if the feature gate is enabled:
```bash
kubectl get pods -n kube-system -l component=kube-scheduler -o yaml | grep SchedulerBatchMoveUnschedulablePods
```

###### What are the reasonable SLOs (Service Level Objectives) for the enhancement?

- Scheduling throughput should not decrease
- P99 scheduling latency should not increase by more than 1 second (the batch interval)

###### What are the SLIs (Service Level Indicators) an operator can use to determine the health of the service?

- `scheduler_schedule_attempts_total`: Rate of scheduling attempts
- `scheduler_pod_scheduling_duration_seconds`: Scheduling latency distribution
- `scheduler_pending_pods`: Number of pending pods by queue

### Scalability

###### Will enabling / using this feature result in any new API calls?

No. This is an internal scheduler optimization.

###### Will enabling / using this feature result in introducing new API types?

No.

###### Will enabling / using this feature result in any new calls to the cloud provider?

No.

###### Will enabling / using this feature result in increasing size or count of the existing API objects?

No.

###### Will enabling / using this feature result in increasing time taken by any operations covered by existing SLIs/SLOs?

P99 scheduling latency may increase by up to 1 second due to batching. This is acceptable for the throughput improvement gained.

###### Will enabling / using this feature result in non-negligible increase of resource usage (CPU, RAM, disk, IO, ...) in any components?

Minimal increase in memory usage for buffering events (small structs, flushed every second). CPU usage should decrease due to reduced lock contention.

### Troubleshooting

###### How does this feature react if the API server and/or etcd is unavailable?

This feature does not interact with API server or etcd directly. Scheduler behavior regarding API server availability is unchanged.

###### What are other known failure modes?

- Buffered events memory growth: Mitigated by 1-second flush interval
- Delayed pod scheduling: Acceptable trade-off for throughput improvement

###### What steps should be taken if SLOs are not being met to determine the problem?

1. Check if feature gate is enabled
2. Monitor `scheduler_pending_pods` metric
3. If pending pods grow unbounded, disable feature gate
4. Collect scheduler logs and profiles

## Implementation History

- 2025-03-10: KEP created
- 2025-03-10: Alpha implementation merged

## Drawbacks

1. **Scheduling Latency**: Up to 1 second additional latency for some pod scheduling scenarios
2. **Complexity**: Additional code paths and concurrency control
3. **Testing**: More test scenarios needed to cover both feature gate states

## Alternatives

1. **Immediate Processing (status quo)**: Keep current behavior. Simpler but has lock contention issues at scale.

2. **Adaptive Batching**: Instead of fixed 1-second interval, batch based on event rate or queue depth. More complex to implement and tune.

3. **Lock-free Queue**: Redesign the priority queue to use lock-free data structures. Significant refactoring with higher risk.

4. **Per-Event-Type Batching**: Only batch certain event types. Adds complexity without clear benefit over uniform batching.

The chosen approach (fixed 1-second batching with feature gate) provides the best balance of simplicity, performance improvement, and safety.

## Infrastructure Needed

No additional infrastructure needed. This is a code-only change to the kube-scheduler.
