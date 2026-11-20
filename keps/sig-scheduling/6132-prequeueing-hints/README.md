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
      - [Unit tests](#unit-tests)
      - [Integration tests](#integration-tests)
      - [e2e tests](#e2e-tests)
      - [Performance tests](#performance-tests)
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
- [x] (R) KEP approvers have approved the KEP status as `implementable`
- [x] (R) Design details are appropriately documented
- [x] (R) Test plan is in place, giving consideration to SIG
  Architecture and SIG Testing input
- [x] (R) Graduation criteria is in place
- [x] (R) Production readiness review completed
- [ ] (R) Production readiness review approved
- [x] "Implementation History" section is up-to-date for milestone
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
| PreQueueingHintFn incorrectly signals "no pods affected" | Deallocation and deletion of an allocated claim signal "evaluate all"; an index lookup error makes the framework evaluate all pods for the DRA plugin. The periodic flush of unschedulable pods provides an additional safety net. |
| A generated claim referenced by a second pod | Pod informer index looks up all pods referencing the claim by spec, not just the owner. `TestPreQueueingHint_SharedClaimMultiplePods` checks that every pod referencing one claim is returned; it uses a claim named in `spec.resourceClaims`, not a generated one. |
| Pod informer index returns stale empty result | This is correct: if no pod references the claim yet, no pod can benefit from the event. The pod status update (linking pod to claim) triggers a separate scheduling event. |

## Design Details

### PreQueueingHintFn Interface

A new optional function is added to `ClusterEventWithHint` in the
scheduling framework. When registered, it is invoked once per
matching registration (in each profile) for a cluster event, inside
`moveAllToActiveOrBackoffQueue` and *before* the loop that iterates
unschedulable pods.

The function returns a `PreQueueingHintResult` struct:
- `AllPods bool`: if true, the plugin is not narrowed: its
  `QueueingHintFn` is evaluated for all unschedulable pods (existing
  behavior). The pod list is ignored.
- `Pods []types.NamespacedName`: when `AllPods` is false, the
  plugin's `QueueingHintFn` is evaluated only for the listed pods.
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
initialization via the shared pod informer, only when the
`SchedulerPreQueueingHints` feature gate is enabled. All scheduler
profiles share one index.

When a ResourceClaim event occurs:

1. **Deallocation**: If the claim's allocation transitions from
   non-nil to nil, or an allocated claim is deleted, signal "evaluate
   all pods" — freed capacity may make any waiting pod schedulable.
2. **Other events**: Look up all pods referencing the claim via
   the informer index. Return their NamespacedNames with
   `allPods=false`. If the index lookup fails, return an error;
   the framework then evaluates all pods for the DRA plugin and
   counts the result as `prequeueing_hint_fn_error`.

This avoids the OwnerReference heuristic entirely. The index
correctly handles the case where multiple pods reference the
same generated claim.

### Feature Gate

`SchedulerPreQueueingHints`:
- v1.37: alpha, default=false. (Initially targeted beta in v1.37, but
  demoted to alpha late in the cycle after correctness bugs were found
  in review — see Implementation History.)
- v1.38: beta, default=true, once those bugs were fixed.
- When disabled: `PreQueueingHintFn` is ignored, all pods are
  evaluated on every event (existing behavior).
- When enabled: `PreQueueingHintFn` narrows the pod set before
  per-pod QueueingHint evaluation.

### Test Plan

[x] I/we understand the owners of the involved components may require updates to
existing tests to make this code solid enough prior to committing the changes necessary
to implement this enhancement.

##### Prerequisite testing updates

The existing QueueingHint tests provide baseline coverage for the
requeue path.

##### Unit tests

- `k8s.io/kubernetes/pkg/scheduler/backend/queue`: `2026-09-24` - `93.8%`
- `k8s.io/kubernetes/pkg/scheduler/framework/plugins/dynamicresources`: `2026-09-24` - `86.9%`

Unit tests cover the DRA index lookup, including a claim shared by
several pods (`TestPreQueueingHint_SharedClaimMultiplePods`), the
deallocation and error fallbacks, per-plugin narrowing, PodGroup and
CompositePodGroup resolution, the metric labels, and the gate on and off
(`TestPriorityQueue_PreQueueingHint`).

##### Integration tests

- [`TestPreQueueingHintNarrowsPodSet`](https://github.com/kubernetes/kubernetes/blob/0661ce1bc61c60dcd59903db2401b1e058c504bc/test/integration/scheduler/prequeueinghint/prequeueinghint_test.go#L91):
  [integration master](https://testgrid.k8s.io/sig-release-master-blocking#integration-master?include-filter-by-regex=scheduler.prequeueinghint),
  [triage search](https://storage.googleapis.com/k8s-triage/index.html?test=prequeueinghint).
  With the gate on, only hinted pods are evaluated; with it off, all pods
  are evaluated.
  [#141732](https://github.com/kubernetes/kubernetes/pull/141732) adds a
  check that a pod left out of one hint is still requeued by a later event.
- Existing scheduler and DRA integration tests (`test/integration/scheduler`,
  `test/integration/dra`), run by `pull-kubernetes-integration` and
  integration-master, pass with the gate on. So does the DRA
  upgrade/downgrade suite `test/e2e_dra` (`pull-kubernetes-dra-integration`).

##### e2e tests

No dedicated e2e test: the feature changes only which pending pods are
re-evaluated, which the unit and integration tests check directly. The
gate is on by default, so the DRA e2e jobs, such as
[`ci-kind-dra`](https://testgrid.k8s.io/sig-node-dynamic-resource-allocation#ci-kind-dra),
exercise it on every claim event.

##### Performance tests

Measured using `test/integration/scheduler_perf` with the
`SchedulingWithResourceClaimTemplate` benchmark:

- Before: ~297 pods/sec
- After: ~711 pods/sec (~2.4x improvement)

The benchmark runs in CI, but its threshold (`dra: 56` for
`5000pods_500nodes`) predates this feature and would not catch a
regression to the old rate. Raising it is a v1.38 follow-up.

### Graduation Criteria

#### Beta

- [x] Feature gate `SchedulerPreQueueingHints` (default=true)
- [x] `PreQueueingHintFn` interface added to scheduling framework
- [x] DRA plugin implements `PreQueueingHintFn` using pod informer
  index
- [x] Deallocation handling (signal "evaluate all pods")
- [x] Per-plugin narrowing: a plugin without a `PreQueueingHintFn`
  (or one returning "evaluate all pods", or erroring) forces all pods
  to be evaluated for that plugin, while other plugins' narrowing is
  preserved
- [x] CompositePodGroup support: a hinted pod is resolved to its root
  entity (PodGroup or CompositePodGroup)
- [x] Unit tests for all code paths, including a claim shared by
  several pods
- [x] Integration test for narrowing (`TestPreQueueingHintNarrowsPodSet`)
- [x] `scheduler_perf` benchmark demonstrating improvement
- [x] Existing DRA e2e tests pass with the feature enabled
- [x] Correctness bugs found during the v1.37 alpha review addressed
  (see Implementation History)

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

Yes. Restart the scheduler with
`--feature-gates=SchedulerPreQueueingHints=false`. The scheduler
reverts to O(N) evaluation. No persistent state.

###### What happens if we reenable the feature if it was previously rolled back?

The scheduler resumes using PreQueueingHintFn. No persistent state.

###### Are there any tests for feature enablement/disablement?

Yes. `TestPriorityQueue_PreQueueingHint` and
`TestPreQueueingHintNarrowsPodSet` run with the gate on and off, and
check that with it off all unschedulable pods are evaluated. The
feature persists no state, so there is nothing else to test.

### Rollout, Upgrade and Rollback Planning

###### How can a rollout or rollback fail? Can it impact already running workloads?

Running workloads are not affected: the feature only changes which
pending pods are re-evaluated on an event.

A rollout can only delay scheduling, not change placement. If a
`PreQueueingHintFn` misses a pod, the pod waits for another event that
requeues it, or at most until the periodic flush moves pods that have
been unschedulable for `podMaxInUnschedulablePodsDuration` (5 minutes by
default). The per-pod
`QueueingHintFn` and the scheduling cycle are unchanged.

A rollback cannot partially fail. The gate is read at startup and
nothing is persisted, so a restarted scheduler is back on the O(N) path.
Only the leader schedules, so an HA control plane has no mixed state.

###### What specific metrics should inform a rollback?

`scheduler_pod_scheduled_after_flush_total` rising above its rate before
enablement: a `PreQueueingHintFn` is missing pods that an event should
have requeued. The counter is not specific to this feature, so compare
rates rather than expecting zero.

###### Were upgrade and rollback tested? Was the upgrade->downgrade->upgrade path tested?

Yes. Toggling the feature gate between scheduler restarts was tested
in scale environments with no issues.

###### Is the rollout accompanied by any deprecations and/or removals of features, APIs, fields of API types, flags, etc.?

No.

### Monitoring Requirements

###### How can an operator determine if the feature is in use by workloads?

- `scheduler_pre_queueing_hint_evaluations_total` is non-zero.
- Feature gate status visible in
  `kubernetes_feature_enabled{name="SchedulerPreQueueingHints"}`

###### How can someone using this feature know that it is working for their instance?

- For DRA workloads using ResourceClaimTemplates: scheduling
  throughput during burst pod creation improves compared to the
  feature being disabled.
- `scheduler_pre_queueing_hint_evaluations_total{plugin="DynamicResources"}`
  is mostly `result="narrowed"`, and
  `scheduler_pod_scheduled_after_flush_total` does not rise (see SLIs).

###### What are the reasonable SLOs (Service Level Objectives) for the enhancement?

- Scheduling latency for DRA pods should not increase.
- No pods should remain stuck in Pending due to missed requeue
  events.

###### What are the SLIs (Service Level Indicators) an operator can use to determine the health of the service?

- Type: Metric
  - Metric name: `scheduler_schedule_attempts_total`
  - Components exposing the metric: `kube-scheduler`
  - Interpretation: narrowing only ever evaluates a subset of the pods
    evaluated with the gate off, so it cannot requeue pods prematurely.
    Fewer `result="scheduled"` attempts while DRA pods stay pending,
    together with a rising `scheduler_pod_scheduled_after_flush_total`,
    may indicate missed requeues.
- Type: Metric
  - Metric name: `scheduler_pod_scheduled_after_flush_total`
  - Components exposing the metric: `kube-scheduler`
  - Interpretation: pods scheduled only after the periodic flush of
    unschedulable pods. This safety net predates the feature, so the
    counter is not expected to be zero; a rise after enablement means a
    PreQueueingHintFn is missing pods.
- Type: Metric
  - Metric name: `scheduler_pre_queueing_hint_evaluations_total`
  - Components exposing the metric: `kube-scheduler`
  - Interpretation: per `plugin`, `result` is `narrowed`, `all_pods` or
    `prequeueing_hint_fn_error`. Mostly `narrowed` means narrowing works;
    sustained errors are a plugin bug.

###### Are there any missing metrics that would be useful to have to improve observability of this feature?

No.

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

###### Will enabling / using this feature result in non-negligible increase of resource usage (CPU, RAM, disk, IO, ...) in any components?

No. The pod informer index (only with the gate on, shared by all
profiles) adds memory proportional to the number of pods with
ResourceClaims, negligible next to the pod informer itself.

###### Can enabling / using this feature result in resource exhaustion of some node resources (PIDs, sockets, inodes, etc.)?

No.

### Troubleshooting

###### How does this feature react if the API server and/or etcd is unavailable?

No impact. The feature is internal to the scheduler's in-memory
queue processing.

###### What are other known failure modes?

| Failure Mode | Detection | Mitigation |
|---|---|---|
| PreQueueingHintFn misses pods | `scheduler_pod_scheduled_after_flush_total` rises after enablement | Disable feature gate |
| A plugin's PreQueueingHintFn errors | `scheduler_pre_queueing_hint_evaluations_total{result="prequeueing_hint_fn_error"}` increases | None needed: that plugin falls back to all pods. Fix the plugin |
| Pod informer index stale | No impact: empty index means no pod references the claim yet; correct behavior | Self-healing via pod status update event |

###### What steps should be taken if SLOs are not being met to determine the problem?

1. Check `scheduler_pre_queueing_hint_evaluations_total` by `plugin`
   and `result`; errors name the failing plugin.
2. Check whether `scheduler_pod_scheduled_after_flush_total` rose after
   enablement.
3. Check scheduler logs at V(5) for "PreQueueingHint narrowed pod
   set" messages to verify the feature is active.
4. Disable the feature gate and restart the scheduler.

## Alternatives

### PodGroup (GenericWorkload) Interaction

When the `GenericWorkload` feature is enabled, the scheduling queue
stores `QueuedPodGroupInfo` entities rather than individual pods.
To resolve which entity a pod belongs to, the framework uses the pod's
`schedulingGroup` field via the pod informer to look up the owning
PodGroup entity in O(1), and evaluates that entity. This shipped with
the v1.37 alpha. v1.38 also resolves a hinted pod to its
`CompositePodGroup` root; in v1.37, narrowing was skipped when
`CompositePodGroup` was enabled.

### Other Alternatives

1. **OwnerReference heuristic**: Use the ResourceClaim's
   OwnerReference to identify the owning pod. Simpler but incorrect
   when multiple pods reference the same generated claim. Rejected
   in favor of the pod informer index.

2. **Optimize QueueingHintFn directly**: Make per-pod evaluation
   faster. Rejected because the O(N) iteration is the bottleneck,
   not per-pod evaluation cost.

3. **No feature gate**: ship as always-on. Rejected: correctness bugs
   found in the v1.37 review demoted the feature to alpha, which is the
   case the gate exists to contain.

## Implementation History

- 2026-05: Initial KEP proposed, targeting beta in v1.37
  ([#138916](https://github.com/kubernetes/kubernetes/pull/138916)).
- v1.37: Merged, but demoted from beta to **alpha** (default=false) late
  in the cycle after post-merge review surfaced correctness bugs in the
  narrowing logic
  ([#140959](https://github.com/kubernetes/kubernetes/pull/140959)).
- 2026-09: KEP merged as `implementable`, targeting beta in v1.38
  ([kubernetes/enhancements#6374](https://github.com/kubernetes/enhancements/pull/6374)).
- v1.38: Graduated to **beta** (default=true) via
  [#141732](https://github.com/kubernetes/kubernetes/pull/141732).
  It fixes the alpha correctness bug where another plugin's narrowing
  dropped pods that only a plugin without a `PreQueueingHintFn` would
  requeue, keeps other plugins' narrowing when one plugin signals "all
  pods" or errors, enables narrowing with CompositePodGroup by resolving a
  hinted pod to its root, and re-promotes the feature gate to beta.
