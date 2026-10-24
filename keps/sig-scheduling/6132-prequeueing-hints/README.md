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
| PreQueueingHintFn incorrectly signals "no pods affected" | Deallocation events and index errors always signal "evaluate all". The periodic flush of unschedulable pods provides an additional safety net. |
| A generated claim referenced by a second pod | Pod informer index looks up all pods referencing the claim by spec, not just the owner. Verified by `TestPreQueueingHint_SharedClaimMultiplePods`. |
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

Core packages touched by this enhancement, with current coverage:

- `k8s.io/kubernetes/pkg/scheduler/backend/queue`: `2026-09-24` - `93.8%`
- `k8s.io/kubernetes/pkg/scheduler/framework/plugins/dynamicresources`: `2026-09-24` - `86.9%`

Notable cases:

- `TestPreQueueingHint`, `TestPreQueueingHint_WithPodInIndexer`,
  `TestPreQueueingHint_PodNotYetIndexed`,
  `TestPreQueueingHint_DuplicateClaimRefs`: pod lookup via index,
  deallocation fallback, empty index (no pods affected), error handling.
- `TestPreQueueingHint_SharedClaimMultiplePods`: multiple pods
  referencing one generated ResourceClaim are all returned by the
  index, which is the correctness case the OwnerReference heuristic
  got wrong.
- `TestPriorityQueue_PreQueueingHint`: gate enabled/disabled, hint
  narrows to specific pods, hint signals all, multiple hints union.
- `TestPreQueueingHint_PerPluginNarrowing`,
  `TestPreQueueingHint_NonNarrowingPluginEvaluatesAllPods`,
  `TestPreQueueingHint_AllPodsAndErrorPreserveNarrowing`: per-plugin
  narrowing, added for beta.
- `TestPreQueueingHint_CompositePodGroupLookup`,
  `TestPreQueueingHint_CompositePodGroupDedup`,
  `TestPreQueueingHint_CompositePodGroupPreCheck`: PodGroup and
  CompositePodGroup root resolution, added for beta.
- `TestPreQueueingHint_Metrics`: the `plugin` and `result` labels on
  `scheduler_pre_queueing_hint_evaluations_total`.

##### Integration tests

- [`TestPreQueueingHintNarrowsPodSet`](https://github.com/kubernetes/kubernetes/blob/0661ce1bc61c60dcd59903db2401b1e058c504bc/test/integration/scheduler/prequeueinghint/prequeueinghint_test.go#L91):
  [integration master](https://testgrid.k8s.io/sig-release-master-blocking#integration-master?include-filter-by-regex=TestPreQueueingHint),
  [triage search](https://storage.googleapis.com/k8s-triage/index.html?test=TestPreQueueingHint).
  Verifies that PreQueueingHintFn narrows the pod set, and that pods
  not covered by a narrowed hint are still requeued by subsequent
  relevant events.
- Existing scheduler integration tests pass with the feature gate
  enabled and disabled.
- DRA integration tests (`pull-kubernetes-dra-integration`) pass.

##### e2e tests

There is no dedicated e2e test for this enhancement, and none is
planned. The feature changes only which pods the scheduler re-evaluates
on an event, not any scheduling decision or API surface, so the
correctness properties are observable at the unit and integration
levels and are covered there. e2e coverage is by way of the existing
DRA e2e suite running with the gate on:

- [`pull-kubernetes-e2e-kind-alpha-beta-features`](https://testgrid.k8s.io/sig-release-master-blocking#pull-kubernetes-e2e-kind-alpha-beta-features):
  exercises `test/e2e/dra` with all alpha/beta gates enabled, so the
  DRA PreQueueingHintFn is on the path for every claim event in those
  tests.

##### Performance tests

Measured using `test/integration/scheduler_perf` with the
`SchedulingWithResourceClaimTemplate` benchmark:

- Before: ~297 pods/sec
- After: ~711 pods/sec (~2.4x improvement)

This benchmark runs in CI and will detect regressions. Its throughput
threshold (`dra: 56` for the `5000pods_500nodes` workload in
`test/integration/scheduler_perf/dra/performance-config.yaml`) still
reflects pre-feature throughput, so a regression back to the old rate
would pass silently. Raising that threshold to match the improvement is
a follow-up tracked for the beta milestone; it is deliberately not done
in the same PR that enables the gate, so that the threshold bump can be
reverted independently if the gate is disabled.

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
- [x] Unit tests for all code paths
- [x] Unit and integration coverage for shared-claim correctness
  (`TestPreQueueingHint_SharedClaimMultiplePods`,
  `TestPreQueueingHintNarrowsPodSet`)
- [x] `scheduler_perf` benchmark demonstrating improvement
- [x] Existing DRA e2e tests pass with the feature enabled
  (`pull-kubernetes-e2e-kind-alpha-beta-features`)
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

Yes. Restart the scheduler without the feature gate. The scheduler
reverts to O(N) evaluation. No persistent state.

###### What happens if we reenable the feature if it was previously rolled back?

The scheduler resumes using PreQueueingHintFn. No persistent state.

###### Are there any tests for feature enablement/disablement?

Yes. `TestPriorityQueue_PreQueueingHint` runs the queue with the gate
both enabled and disabled and asserts that the disabled path evaluates
all unschedulable pods, matching pre-feature behavior. The scheduler
integration tests are also run with the gate enabled and disabled.
Because the feature keeps no persistent state and touches no API object,
enablement and disablement are fully covered in-process; there is no
upgrade artifact to test separately.

### Rollout, Upgrade and Rollback Planning

###### How can a rollback be performed?

Set `--feature-gates=SchedulerPreQueueingHints=false` on the
scheduler and restart.

###### How can a rollout or rollback fail? Can it impact already running workloads?

Already running workloads are not affected. The feature only changes
which pending pods the scheduler re-evaluates when a cluster event
arrives; it never moves, evicts or re-places a pod that is already bound
to a node.

The failure mode for a rollout is delayed scheduling, not incorrect
scheduling. If a `PreQueueingHintFn` narrows away a pod that a relevant
event should have requeued, that pod stays in `unschedulableEntities`
until the periodic flush moves it back to the active queue, so it is
scheduled late rather than never. Pods are never placed on a node they
would not otherwise have been placed on, because the per-pod
`QueueingHintFn` and the entire scheduling cycle are unchanged — only the
candidate set feeding them is narrowed.

Rollback cannot partially fail. The gate is read at scheduler startup
and the queue holds no persisted state, so a scheduler restarted with the
gate off rebuilds its queue from informers and is immediately back on the
pre-feature O(N) path. In an HA control plane with several scheduler
replicas, only the active leader's behavior matters, so a rolling restart
cannot leave the cluster in a mixed state.

###### What specific metrics should inform a rollback?

- `scheduler_pre_queueing_hint_evaluations_total{result="prequeueing_hint_fn_error"}`
  increasing: a plugin's `PreQueueingHintFn` is erroring. Narrowing is
  skipped for that plugin so scheduling stays correct, but a sustained
  error rate means the hint is not doing its job.
- `scheduler_pod_scheduled_after_flush_total` rate increasing relative
  to the pre-enablement baseline: pods are being rescued by the periodic
  flush timeout rather than by event-driven requeue. This suggests
  PreQueueingHintFn is incorrectly signaling "no pods affected" for
  events that should trigger re-evaluation. Because this counter also
  moves for reasons unrelated to this feature, compare rates before and
  after enablement rather than treating any non-zero value as a defect.
- Pods in `Pending` for longer than the unschedulable flush interval
  while their ResourceClaims are allocated: the user-visible form of a
  missed requeue.

###### Were upgrade and rollback tested? Was the upgrade->downgrade->upgrade path tested?

Yes. Toggling the feature gate between scheduler restarts was tested
in scale environments with no issues.

###### Is the rollout accompanied by any deprecations and/or removals of features, APIs, fields of API types, flags, etc.?

No.

### Monitoring Requirements

###### How can an operator determine if the feature is in use by workloads?

- Scheduler logs at V(5): "PreQueueingHint narrowed pod set"
- Feature gate status visible in
  `kubernetes_feature_enabled{name="SchedulerPreQueueingHints"}`

###### How can someone using this feature know that it is working for their instance?

- For DRA workloads using ResourceClaimTemplates: scheduling
  throughput during burst pod creation improves compared to the
  feature being disabled.
- `scheduler_pre_queueing_hint_evaluations_total` for the
  `dynamicresources` plugin is dominated by `result="narrowed"`.
- `scheduler_pod_scheduled_after_flush_total` does not rise relative to
  its rate before enablement (pods are being requeued by events, not by
  the flush timeout).

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
  - Interpretation: counts pods that reached a node only after the
    periodic flush of `unschedulableEntities` rescued them, which is a
    pre-existing safety net not specific to this feature. It is
    therefore not expected to be exactly zero; compare its rate against
    the pre-enablement baseline for the same workload. A rate that rises
    after enablement indicates PreQueueingHintFn is narrowing away pods
    that a relevant event should have requeued.
- Type: Metric
  - Metric name: `scheduler_pre_queueing_hint_evaluations_total`
  - Components exposing the metric: `kube-scheduler`
  - Interpretation: `result` is `narrowed`, `all_pods`,
    `no_prequeueing_hint_fn`, or `prequeueing_hint_fn_error`. Mostly
    `narrowed` means narrowing works; sustained
    `prequeueing_hint_fn_error` is a bug.

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

The pod informer index adds a small memory overhead proportional to
the number of pods with ResourceClaims. This is negligible compared
to the existing pod informer memory usage.

###### Can enabling / using this feature result in resource exhaustion of some node resources (PIDs, sockets, inodes, etc.)?

No.

### Troubleshooting

###### How does this feature react if the API server and/or etcd is unavailable?

No impact. The feature is internal to the scheduler's in-memory
queue processing.

###### What are other known failure modes?

| Failure Mode | Detection | Mitigation |
|---|---|---|
| PreQueueingHintFn incorrectly signals no pods | `scheduler_pod_scheduled_after_flush_total` rate rises above its pre-enablement baseline; pods stay `Pending` past the flush interval with claims allocated | Disable feature gate |
| A plugin's PreQueueingHintFn errors | `scheduler_pre_queueing_hint_evaluations_total{result="prequeueing_hint_fn_error"}` increases | None needed for correctness: that plugin falls back to evaluating all pods while other plugins keep narrowing. Fix the plugin |
| Pod informer index stale | No impact: empty index means no pod references the claim yet; correct behavior | Self-healing via pod status update event |

###### What steps should be taken if SLOs are not being met to determine the problem?

1. Check `scheduler_pre_queueing_hint_evaluations_total` by `plugin` and
   `result`. A non-trivial `prequeueing_hint_fn_error` rate identifies
   the offending plugin directly.
2. Check `scheduler_pod_scheduled_after_flush_total` — if its rate is
   elevated versus before enablement, a PreQueueingHintFn is missing
   pods that should be re-evaluated.
3. Check scheduler logs at V(5) for "PreQueueingHint narrowed pod
   set" messages to verify the feature is active.
4. Disable the feature gate and restart the scheduler.

## Alternatives

### PodGroup (GenericWorkload) Interaction

When the `GenericWorkload` feature is enabled, the scheduling queue
stores `QueuedPodGroupInfo` entities rather than individual pods.
To resolve which entity a pod belongs to, the framework uses the pod's
`schedulingGroup` field via the pod informer to look up the owning
PodGroup entity in O(1), and evaluates that entity.

This is implemented for beta in v1.38, including resolution to a
`CompositePodGroup` root, and is listed under the beta graduation
criteria above.

### Other Alternatives

1. **OwnerReference heuristic**: Use the ResourceClaim's
   OwnerReference to identify the owning pod. Simpler but incorrect
   when multiple pods reference the same generated claim. Rejected
   in favor of the pod informer index.

2. **Optimize QueueingHintFn directly**: Make per-pod evaluation
   faster. Rejected because the O(N) iteration is the bottleneck,
   not per-pod evaluation cost.

3. **No feature gate**: ship as always-on, on the argument that the
   pod informer index approach is correct by construction. Rejected: the
   v1.37 cycle demoted this feature from beta to alpha after review
   found correctness bugs in the narrowing logic, which is exactly the
   case the gate exists to contain. `SchedulerPreQueueingHints` is
   retained for beta with `disable-supported: true`.

## Implementation History

- 2026-05: Initial KEP proposed, targeting beta in v1.37
  ([#138916](https://github.com/kubernetes/kubernetes/pull/138916)).
- v1.37: Merged, but demoted from beta to **alpha** (default=false) late
  in the cycle after post-merge review surfaced correctness bugs in the
  narrowing logic
  ([#140959](https://github.com/kubernetes/kubernetes/pull/140959)).
- v1.38: Graduated to **beta** (default=true) via
  [#141732](https://github.com/kubernetes/kubernetes/pull/141732),
  which fixed the alpha correctness bugs — per-plugin narrowing so a
  plugin without a `PreQueueingHintFn` no longer drops the pods other
  plugins narrowed, and resolving a hinted pod to its CompositePodGroup
  root — and re-promoted the feature gate to beta.
