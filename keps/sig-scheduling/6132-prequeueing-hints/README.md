# KEP-6132: PreQueueingHint Extension Point for Scheduler Event Processing

<!-- toc -->
- [Release Signoff Checklist](#release-signoff-checklist)
- [Summary](#summary)
- [Motivation](#motivation)
  - [Goals](#goals)
  - [Non-Goals](#non-goals)
- [Proposal](#proposal)
  - [User Stories](#user-stories)
  - [Risks and Mitigations](#risks-and-mitigations)
- [Design Details](#design-details)
  - [PreQueueingHintFn Interface](#prequeueinghintfn-interface)
  - [DRA Plugin Implementation](#dra-plugin-implementation)
  - [Feature Gate](#feature-gate)
  - [Test Plan](#test-plan)
    - [Prerequisite testing updates](#prerequisite-testing-updates)
    - [Unit Tests](#unit-tests)
    - [Integration Tests](#integration-tests)
    - [E2E Tests](#e2e-tests)
    - [Performance Tests](#performance-tests)
  - [Graduation Criteria](#graduation-criteria)
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
- [Alternatives](#alternatives)
  - [PodGroup (GenericWorkload) Interaction](#podgroup-genericworkload-interaction)
  - [Other Alternatives](#other-alternatives)
- [Implementation History](#implementation-history)
<!-- /toc -->

## Release Signoff Checklist

Items marked with (R) are required *prior to targeting to a milestone
/ release*.

- [ ] (R) Enhancement issue in release milestone, which links to KEP
  dir in [kubernetes/enhancements]
- [ ] (R) KEP approvers have approved the KEP status as `implementable`
- [ ] (R) Design details are appropriately documented
- [ ] (R) Test plan is in place, giving consideration to SIG
  Architecture and SIG Testing input
- [ ] (R) Graduation criteria is in place
- [ ] (R) Production readiness review completed
- [ ] (R) Production readiness review approved
- [ ] "Implementation History" section is up-to-date for milestone
- [ ] User-facing documentation has been created in
  [kubernetes/website], for publication to [kubernetes.io]
- [ ] Supporting documentation

## Summary

This KEP introduces a `PreQueueingHintFn` extension point to the
Kubernetes scheduling framework. When a cluster event occurs, the
scheduler currently iterates all unschedulable pods to evaluate
per-pod `QueueingHintFn`. The new `PreQueueingHintFn` is called
once per event *before* this iteration, allowing plugins to return
a targeted list of pods to check — or signal that no pods need
checking. This reduces the requeue path from O(N) to O(1) for
events that affect only specific pods.

The DRA plugin implements `PreQueueingHintFn` for ResourceClaim
events using a pod informer index that maps ResourceClaims to the
pods referencing them.
## Motivation

When a burst of pods using DRA ResourceClaimTemplates are created
simultaneously (e.g., a large Deployment scaling up), each pod
generates a ResourceClaim. Each ResourceClaim creation or allocation
event triggers `MoveAllToActiveOrBackoffQueue`, which iterates all
unschedulable pods to evaluate per-pod `QueueingHintFn`. With N pods
in the unschedulable queue and N ResourceClaim events, the total
work is O(N²).

This quadratic behavior becomes the dominant CPU cost in the
scheduler, limiting scheduling throughput during burst workload
creation. The existing `QueueingHintFn` is per-pod (called once per
unschedulable pod per event). There is no mechanism to narrow the
candidate set *before* iterating all pods.

### Goals

- Provide a `PreQueueingHintFn` extension point that plugins can
  implement to narrow the set of pods evaluated on cluster events.
- Reduce ResourceClaim event processing from O(N) to O(1) for
  claims referenced by a known set of pods.
- Maintain correctness: provide an explicit "evaluate all" signal
  for events where the affected pod set cannot be determined.

### Non-Goals

- Replacing or modifying the existing `QueueingHintFn` mechanism.
- Optimizing events other than ResourceClaim (though the framework
  supports it).
- Changing the scheduling algorithm or pod priority ordering.

## Proposal

### User Stories

**Story 1: Large-scale DRA workloads**

As a cluster operator rapidly creating thousands of pods (e.g., 10K)
with DRA ResourceClaimTemplates, I want the scheduler to maintain
high throughput without being bottlenecked by ResourceClaim event
processing.

### Risks and Mitigations

| Risk | Mitigation |
|------|-----------|
| PreQueueingHintFn incorrectly signals "no pods affected" | Deallocation events and index errors always signal "evaluate all". The periodic flush of unschedulable pods provides an additional safety net. |
| A generated claim referenced by a second pod | Pod informer index looks up all pods referencing the claim by spec, not just the owner. Verified by E2E test. |
| Pod informer index returns stale empty result | This is correct: if no pod references the claim yet, no pod can benefit from the event. The pod status update (linking pod to claim) triggers a separate scheduling event. |

## Design Details

### PreQueueingHintFn Interface

A new optional function is added to `ClusterEventWithHint` in the
scheduling framework. When registered, it is invoked once per
matching cluster event inside `moveAllToActiveOrBackoffQueue`,
*before* the loop that iterates unschedulable pods.

The function returns a `PreQueueingHintResult` struct:
- `AllPods bool`: if true, all unschedulable pods should be
  evaluated (existing behavior). The pod list is ignored.
- `Pods []types.NamespacedName`: when `AllPods` is false, only
  the listed pods are evaluated by the per-pod `QueueingHintFn`.
  An empty or nil slice means no pods need evaluation for this
  event.

If no `PreQueueingHintFn` is registered for an event, the framework
falls back to evaluating all pods (existing behavior).

Multiple plugins may register a `PreQueueingHintFn` for the same
event. The framework tracks PreQueueingHint results per-plugin:
each plugin's QueueingHint is only invoked for the pods its own
PreQueueingHintFn identified. Plugins without PreQueueingHintFn
default to "all pods" only for their own QueueingHint evaluation.
If a plugin signals "all pods", the framework evaluates all pods
for that plugin's QueueingHint only.

### DRA Plugin Implementation

The DRA plugin maintains a pod informer index that maps each
ResourceClaim (by namespace/name) to the pods that reference it
in their `spec.resourceClaims`. This index is registered at plugin
initialization via the shared pod informer.

When a ResourceClaim event occurs:

1. **Deallocation**: If the claim's allocation transitions from
   non-nil to nil, signal "evaluate all pods" — freed capacity
   may make any waiting pod schedulable.
2. **Other events**: Look up all pods referencing the claim via
   the informer index. Return their NamespacedNames with
   `allPods=false`. If the index lookup fails, signal
   "evaluate all pods" as a safe fallback.

This avoids the OwnerReference heuristic entirely. The index
correctly handles the case where multiple pods reference the
same generated claim.

### Feature Gate

`SchedulerPreQueueingHints` (beta, default=true):
- When disabled: `PreQueueingHintFn` is ignored, all pods are
  evaluated on every event (existing behavior).
- When enabled: `PreQueueingHintFn` narrows the pod set before
  per-pod QueueingHint evaluation.

### Test Plan

#### Prerequisite testing updates

The existing QueueingHint tests provide baseline coverage for the
requeue path.

#### Unit Tests

- `TestPreQueueingHint`: Tests pod lookup via index, deallocation
  fallback, empty index (no pods affected), error handling.
- `TestPriorityQueue_PreQueueingHint`: Tests gate enabled/disabled,
  hint narrows to specific pods, hint signals all, multiple hints
  union.

#### Integration Tests

- Existing scheduler integration tests pass with feature gate
  enabled and disabled.
- DRA integration tests (`pull-kubernetes-dra-integration`) pass.
- Integration test verifying PreQueueingHintFn correctly narrows
  the pod set and that missed pods are rescued by periodic flush.

#### E2E Tests

- `pull-kubernetes-e2e-kind-alpha-beta-features`: Full e2e with
  all alpha/beta gates enabled.
- Dedicated test: multiple pods sharing one generated ResourceClaim
  all get scheduled after driver starts (verifies the index-based
  approach handles shared claims correctly).

#### Performance Tests

Measured using `test/integration/scheduler_perf` with the
`SchedulingWithResourceClaimTemplate` benchmark:

- Before: ~297 pods/sec
- After: ~711 pods/sec (~2.4x improvement)

This benchmark is part of CI and will detect regressions. Once
this feature is merged, the threshold for expected throughput
needs to be revised to match the improvement.

### Graduation Criteria

#### Beta

- [x] Feature gate `SchedulerPreQueueingHints` (default=true)
- [x] `PreQueueingHintFn` interface added to scheduling framework
- [x] DRA plugin implements `PreQueueingHintFn` using pod informer
  index
- [x] Deallocation handling (signal "evaluate all pods")
- [x] Unit tests for all code paths
- [x] E2E test for shared claim correctness
- [x] `scheduler_perf` benchmark demonstrating improvement
- [x] All e2e tests pass with feature enabled

#### GA

- [ ] Feature gate locked to true, removal planned
- [ ] No reported correctness issues for 2 releases
- [ ] Documentation updated

### Upgrade / Downgrade Strategy

- **Upgrade**: Feature defaults to enabled. No action required.
  Scheduling behavior is unchanged; only performance improves.
- **Downgrade**: Disabling the feature gate reverts to existing
  O(N) behavior. No state is persisted.

### Version Skew Strategy

The feature is entirely within `kube-scheduler`. No version skew
concerns with other components.

## Production Readiness Review Questionnaire

### Feature Enablement and Rollback

###### How can this feature be enabled / disabled in a live cluster?

- Feature gate name: `SchedulerPreQueueingHints`
- Components depending on the feature gate: `kube-scheduler`
- Can be toggled by restarting the scheduler with/without the flag.

###### Does enabling the feature change any default behavior?

No. The feature only changes the *performance* of event processing,
not scheduling decisions. All pods that would have been re-evaluated
are still re-evaluated when `allPods=true` is signaled.

###### Can the feature be disabled once it has been enabled (i.e. can we roll back the enablement)?

Yes. Restart the scheduler without the feature gate. The scheduler
reverts to O(N) evaluation. No persistent state.

###### What happens if we reenable the feature if it was previously rolled back?

The scheduler resumes using PreQueueingHintFn. No persistent state.

### Rollout, Upgrade and Rollback Planning

###### How can a rollback be performed?

Set `--feature-gates=SchedulerPreQueueingHints=false` on the
scheduler and restart.

###### What specific metrics should inform a rollback?

- `scheduler_pod_scheduled_after_flush_total` increasing: indicates
  pods are being rescued by the periodic flush timeout rather than
  by event-driven requeue. This suggests PreQueueingHintFn is
  incorrectly signaling "no pods affected" for events that should
  trigger re-evaluation.

###### Were upgrade and rollback tested? Was the upgrade->downgrade->upgrade path tested?

Yes. Toggling the feature gate between scheduler restarts was tested
in scale environments with no issues.

###### Is the rollout accompanied by any deprecations and/or removals of features, APIs, fields of API types, flags, etc.?

No.

### Monitoring Requirements

###### How can an operator determine if the feature is in use?

- Scheduler logs at V(5): "PreQueueingHint narrowed pod set"
- Feature gate status visible in
  `kubernetes_feature_enabled{name="SchedulerPreQueueingHints"}`

###### How can someone using this feature know that it is working for their instance?

- For DRA workloads using ResourceClaimTemplates: scheduling
  throughput during burst pod creation improves compared to the
  feature being disabled.
- `scheduler_pod_scheduled_after_flush_total` remains at zero or
  near-zero (pods are being requeued by events, not by flush
  timeout).

###### What are the reasonable SLOs (Service Level Objectives) for the enhancement?

- Scheduling latency for DRA pods should not increase.
- No pods should remain stuck in Pending due to missed requeue
  events.

###### What are the SLIs (Service Level Indicators) an operator can use to determine the health of the service?

- Type: Metric
  - Metric name: `scheduler_schedule_attempts_total`
  - Components exposing the metric: `kube-scheduler`
  - Interpretation: an unexpected increase in failed attempts
    may indicate pods being requeued prematurely.
- Type: Metric
  - Metric name: `scheduler_pod_scheduled_after_flush_total`
  - Components exposing the metric: `kube-scheduler`
  - Interpretation: should remain at zero. Non-zero indicates
    pods were missed by event-driven requeue and rescued by
    the periodic flush.
- Type: Metric
  - Metric name: `scheduler_pre_queueing_hint_evaluations_total`
  - Components exposing the metric: `kube-scheduler`
  - Interpretation: a high ratio of `narrowed` vs `all_pods`
    indicates the optimization is effective. If `all_pods`
    dominates, most events are falling back to the full scan.

###### Are there any missing metrics that would be useful to have in this context?

A metric `scheduler_pre_queueing_hint_evaluations_total` with
`plugin` and `result` labels (where result is `all_pods` or
`narrowed`) will be added to track PreQueueingHint invocations
and their outcomes.

### Dependencies

###### Does this feature depend on any specific services running in the cluster?

No. It is internal to kube-scheduler.

###### Does it have a known list of any hard or soft dependencies on other Kubernetes features?

The DRA plugin's PreQueueingHintFn requires DRA to be active.
DRA is GA.

### Scalability

###### Will enabling / using this feature result in any new API calls?

No.

###### Will enabling / using this feature result in introducing new API types?

No.

###### Will enabling / using this feature result in any new calls to the cloud provider?

No.

###### Will enabling / using this feature result in increasing size or count of the existing API objects?

No.

###### Will enabling / using this feature result in increasing time taken by any operations covered by existing SLIs/SLOs?

No. It reduces scheduling latency.

###### Will enabling / using this feature result in non-negligible increase of resource usage (CPU, RAM, disk, IO, ...) in any component?

The pod informer index adds a small memory overhead proportional to
the number of pods with ResourceClaims. This is negligible compared
to the existing pod informer memory usage.

###### Can enabling / using this feature result in resource exhaustion of some node resources (PIDs, sockets, inodes, etc.)?

No.

### Troubleshooting

###### How does the feature react if the API server and/or etcd is unavailable?

No impact. The feature is internal to the scheduler's in-memory
queue processing.

###### What are other known failure modes?

| Failure Mode | Detection | Mitigation |
|---|---|---|
| PreQueueingHintFn incorrectly signals no pods | `scheduler_pod_scheduled_after_flush_total` increases | Disable feature gate |
| Pod informer index stale | No impact: empty index means no pod references the claim yet; correct behavior | Self-healing via pod status update event |

###### What steps should be taken if SLOs are not being met to determine the problem?

1. Check `scheduler_pod_scheduled_after_flush_total` — if elevated,
   the PreQueueingHint is missing pods that should be re-evaluated.
2. Check scheduler logs at V(5) for "PreQueueingHint narrowed pod
   set" messages to verify the feature is active.
3. Disable the feature gate and restart the scheduler.

## Alternatives

### PodGroup (GenericWorkload) Interaction

When the `GenericWorkload` feature is enabled, the scheduling queue
stores `QueuedPodGroupInfo` entities rather than individual pods.
To resolve which entity a pod belongs to, the framework will use
the pod's `schedulingGroup` field via the pod informer to look up
the owning PodGroup entity in O(1).

This will be supported in 1.37. The framework will use the pod's
`schedulingGroup` field to look up the owning PodGroup entity
and evaluate it accordingly.

### Other Alternatives

1. **OwnerReference heuristic**: Use the ResourceClaim's
   OwnerReference to identify the owning pod. Simpler but incorrect
   when multiple pods reference the same generated claim. Rejected
   in favor of the pod informer index.

2. **Optimize QueueingHintFn directly**: Make per-pod evaluation
   faster. Rejected because the O(N) iteration is the bottleneck,
   not per-pod evaluation cost.

3. **No feature gate**: Ship as always-on. Accepted for beta since
   the pod informer index approach is correct by construction and
   verified by E2E tests.

## Implementation History

- 2026-05: Initial KEP for beta proposed
  ([#138916](https://github.com/kubernetes/kubernetes/pull/138916))
