# KEP-6003: Configurable sync period for HPA

<!-- toc -->
- [Release Signoff Checklist](#release-signoff-checklist)
- [Summary](#summary)
- [Motivation](#motivation)
  - [Goals](#goals)
  - [Non-Goals](#non-goals)
- [Proposal](#proposal)
  - [User Stories](#user-stories)
    - [Story 1](#story-1)
    - [Story 2](#story-2)
    - [Story 3](#story-3)
  - [Risks and Mitigations](#risks-and-mitigations)
- [Design Details](#design-details)
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
<!-- /toc -->

## Release Signoff Checklist

Items marked with (R) are required *prior to targeting to a milestone / release*.

- [ ] (R) Enhancement issue in release milestone, which links to KEP dir in [kubernetes/enhancements] (not the initial KEP PR)
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
- [ ] Supporting documentation—e.g., additional design documents, links to mailing list discussions/SIG meetings, relevant PRs/issues, release notes

[kubernetes.io]: https://kubernetes.io/
[kubernetes/enhancements]: https://git.k8s.io/enhancements
[kubernetes/kubernetes]: https://git.k8s.io/kubernetes
[kubernetes/website]: https://git.k8s.io/website

## Summary

[Horizontal Pod Autoscaler][] (HPA) periodically reconciles the desired replica
count for a Deployment (or other resource with a `/scale` subresource) based on
observed metrics. The frequency is governed by a single global flag,
`--horizontal-pod-autoscaler-sync-period`, which defaults to 15 seconds and
applies to every HPA in the cluster.

This proposal adds an optional `syncPeriodSeconds` field to
`HorizontalPodAutoscalerSpec` that overrides the global sync period for a single
HPA. When the field is unset, the global default applies.

[Horizontal Pod Autoscaler]: https://kubernetes.io/docs/tasks/run-application/horizontal-pod-autoscale/

## Motivation

The HPA sync period is a cluster-wide parameter set with the
[Kube Controller Manager][] `--horizontal-pod-autoscaler-sync-period` flag
(default 15s). Forcing every HPA in the cluster to reconcile at the same
frequency creates a tension:

1. **Latency-sensitive workloads** need scaling decisions every few seconds to
   respond to traffic spikes.
2. **Stable workloads** are fine with the default or longer, and raising the
   global frequency on their behalf increases load on the API server and metrics
   backends for no benefit.

The concrete driver is request-driven autoscaling on external metrics. Providers
such as [KEDA][] -- in particular the [KEDA HTTP Add-on][], which exposes the
in-flight and incoming HTTP request counts observed by its interceptor -- serve
values derived from in-memory counters that are as fresh as the query. There the
15 second sync period, not the metrics pipeline, dominates end-to-end reaction
time.

The gap is clearest for teams migrating from a purpose-built request-driven
autoscaler. Knative Serving's Pod Autoscaler recomputes desired replicas every 2
seconds by default (`tick-interval` in its `config-autoscaler` ConfigMap). An HPA
cannot approach that cadence without raising the reconcile frequency for every
HPA in the cluster, which makes HPA-based autoscaling hard to adopt as a
replacement in latency-sensitive, scale-from-idle workloads.

The request predates the KEP process; see [kubernetes#110317][]. Since
appropriate sync periods are workload-dependent, this KEP lets users set one per
`HorizontalPodAutoscaler`, overriding the global default when present.

[Kube Controller Manager]: https://kubernetes.io/docs/reference/command-line-tools-reference/kube-controller-manager/
[KEDA]: https://keda.sh/
[KEDA HTTP Add-on]: https://github.com/kedacore/http-add-on
[kubernetes#110317]: https://github.com/kubernetes/kubernetes/issues/110317

### Goals

- Allow users to optionally override the default HPA reconciliation frequency
  on a per-HPA basis.
- Enable HPAs driven by rapidly updating custom or external metrics to react on
  the order of a few seconds.
- Allow HPAs that do not need the default cadence to reconcile less often,
  reducing load on the control plane and on metrics backends.
- Maintain full backward compatibility: existing HPAs without the new field
  continue to use the global sync period.

### Non-Goals

- Change the default value of the global
  `--horizontal-pod-autoscaler-sync-period` flag.
- Allow sub-second sync periods.
- Improve metric freshness or change how any metrics source collects metrics.
- Guarantee a hard real-time reconciliation deadline.

## Proposal

We propose a new field on [`HorizontalPodAutoscalerSpec`][]:

- `syncPeriodSeconds`: *(int32)* the delay, in seconds, after a reconciliation
  completes before the next periodic reconciliation of this HPA is eligible to
  run. Must be greater than or equal to 3 and less than or equal to 3600 (one
  hour).

The field is optional; when unset the HPA uses the global
`--horizontal-pod-autoscaler-sync-period` value. Creation and spec-change events
may enqueue the HPA immediately, before this delay expires.

**Metrics freshness**: a shorter period only improves reaction time if the
configured metrics refresh more often than the delay. `Resource` and
`ContainerResource` metrics from [metrics-server][] are collected on its own
schedule ([binary default 60s, minimum 10s][ms-options],
[shipped manifests 15s][ms-manifest]), so reconciling faster reuses cached
input. `External`,
`Pods`, and `Object` metrics from adapters such as [KEDA][] can be as fresh as
the query; that is the case this field is for. Field documentation will state
the limitation.

The field is not scoped to metric type. A single HPA may mix types and the
period governs the reconcile loop as a whole; validation must not depend on the
mutable `spec.metrics` list; and silently ignoring a set field is a poor
contract.

**Field placement**: `syncPeriodSeconds` is top-level on
`HorizontalPodAutoscalerSpec`, not on `HorizontalPodAutoscalerBehavior`.
`autoscaling/v2` defaulting keys off `spec.behavior != nil`, so a
behavior-scoped field would materialize a full set of default scaling rules
into any object that only sets a sync period. A top-level field avoids that
and keeps `spec.behavior` meaning "scaling rules". The rejected placement is
in [Alternatives](#alternatives).

[`HorizontalPodAutoscalerSpec`]: https://kubernetes.io/docs/reference/generated/kubernetes-api/v1.35/#horizontalpodautoscalerspec-v2-autoscaling
[KEP-4951]: /keps/sig-autoscaling/4951-configurable-hpa-tolerance
[metrics-server]: https://github.com/kubernetes-sigs/metrics-server
[ms-options]: https://github.com/kubernetes-sigs/metrics-server/blob/master/cmd/metrics-server/app/options/options.go
[ms-manifest]: https://github.com/kubernetes-sigs/metrics-server/blob/master/manifests/base/deployment.yaml

### User Stories

#### Story 1

As the author of an HTTP-request-driven autoscaling stack such as the
[KEDA HTTP Add-on][], I want the HPAs I generate to reconcile every few seconds
so a burst of requests becomes a scale-up quickly. My adapter already serves
fresh request counts; the 15s global period is the bottleneck, and I cannot
change the controller-manager flag.

#### Story 2

As a platform engineer on a multi-tenant cluster, I want teams to tune HPA
responsiveness for their own workloads without changing a cluster-wide flag,
while retaining cluster-level policy to set a minimum delay and observability to
detect aggregate controller saturation.

#### Story 3

As a cluster operator with many HPAs on slow-moving batch workloads, I want a
longer sync period on those HPAs to cut metrics queries and API calls, while
leaving latency-sensitive HPAs at the cluster default. Today the only way to
reduce this load is to slow down every HPA in the cluster.

### Risks and Mitigations

**Increased API server and metrics load.** Short sync periods on many HPAs raise
the rate of metrics queries and scale sub-resource calls. Four layers contain
this:

- **Validation bounds**: `syncPeriodSeconds` must be >= 3 and <= 3600. The
  lower bound is derived from controller safety and reversibility, not from
  metric freshness:
  - With fixed-delay semantics, a delay shorter than the reconciliation
    duration degenerates into continuous back-to-back reconciliation. At 3s,
    one HPA with a 1s reconciliation occupies at most a quarter of one
    worker, so no single HPA can monopolize the pool.
  - The bound can only ever be *relaxed*: lowering a minimum makes previously
    invalid values valid, whereas raising it would invalidate persisted
    values and break subsequent updates. Starting at 3s preserves the option
    to go lower in Beta with usage data; starting at 1s would be
    irreversible.
  - The floor is deliberately above what the fastest sources could drive.
    Knative's Pod Autoscaler, the reference point for request-driven
    scaling, ticks every 2s, and in-memory request counters such as the KEDA
    HTTP Add-on interceptor are as fresh as the query. A 3s delay plus a
    1-2s reconciliation gives a roughly 4-5s start-to-start interval, about
    twice Knative's cadence, so this floor does not fully close the gap for
    the most latency-sensitive workloads. That residual gap is accepted for
    Alpha in exchange for a bound that can be loosened once usage data
    exists.
- **Feature gate**: in Alpha the field is gated behind
  `HPAConfigurableSyncPeriod`.
- **Best-effort semantics**: the next periodic reconciliation is scheduled only
  after the current one completes, and the typed workqueue deduplicates the HPA
  key. This prevents duplicate backlog for one HPA, but does not bound aggregate
  queueing across HPAs.
- **Policy enforcement**: administrators can set a cluster-specific floor with
  [ValidatingAdmissionPolicy][] (or a webhook):
  ```
  rule: "!has(object.spec.syncPeriodSeconds) ||
         object.spec.syncPeriodSeconds >= 5"
  ```
  This is a per-object check and does not bound aggregate load.

**API server cost model.** The lower bound is not what makes API server load
safe; its job is to keep a single HPA from monopolizing a controller worker.
The API server argument rests on the shape of the load, and SIG Scalability
review of this model and of the bound is an Alpha graduation criterion.

- **Per reconciliation**: roughly one etcd-backed read (`GET` on the target's
  `scale` subresource), at most one etcd-backed write (`UpdateStatus`, most
  cycles for an active workload), and one aggregation-layer query per metric
  spec. Pod and HPA reads are informer-cached. Aggregation-layer queries never
  reach etcd but are still API server work -- authentication, authorization,
  APF classification, proxying -- and are counted here.
- **Multiplier**: `15 / period` for the HPAs that set the field -- 1.5x at
  10s, 3x at 5s, at most 5x at the 3s floor. No new call type, no new scaling
  dimension.
- **Expected usage**: the floor is a bound, not the typical value; 5s or 10s
  serve most latency-sensitive workloads. Only HPAs on metrics that refresh
  faster than the global period -- `External`, `Pods`, and `Object` metrics
  from adapters, i.e. request-driven, scale-from-idle, serverless-style
  workloads -- gain anything; the CPU- and memory-driven HPAs that make up the
  bulk of most clusters gain nothing (see [Proposal](#proposal)). In a
  1,000-HPA cluster where 50 request-driven HPAs opt in at 5s, the controller's
  reconciliation rate rises from about 67/s to 74/s, roughly 10%.
- **Upper bound**: 1,000 HPAs all at 3s would be about 333 reconciliations/s,
  roughly 1,000 API server requests/s, two thirds etcd-backed -- the same load
  as 5,000 HPAs at today's default, which is valid today with no bound at all.
  Aggregate load stays linear in HPA count, and administrators control both
  factors: HPA count through `ResourceQuota`, the floor through the admission
  policy above.

Client-side `--kube-api-qps` / `--kube-api-burst` and server-side
[API Priority and Fairness][] are deliberately not part of this argument; a
well-configured deployment should never approach them. They only determine the
failure mode of a misconfigured one: excess demand becomes longer scheduling
delays inside the HPA controller (discussed next), not API server saturation.

**Status writes are the removable half of that cost.** Of the etcd-backed
calls above, the write is a by-product of reporting rather than of scaling.
The controller rebuilds the whole status each cycle, and
`status.currentMetrics` carries raw observed values that differ almost every
cycle on an active workload, so `updateStatusIfNeeded` writes even when
replicas, conditions, and `lastScaleTime` are unchanged. `currentMetrics` is
GA API surface and the primary debugging signal (`kubectl get hpa` `TARGETS`,
`kubectl describe`, kube-state-metrics), so the field stays; how often the
controller persists it does not have to follow the reconcile cadence. If SIG
Scalability review finds the etcd-write rate at the floor unacceptable, the
mitigation held in reserve is to decouple the two: reconcile and decide every
`syncPeriodSeconds`, but persist `currentMetrics` at most once per global sync
period, and immediately whenever replicas or conditions change. That keeps the
per-HPA etcd-write rate at today's level and leaves the fast loop with one read
plus the metrics queries, at the cost of `TARGETS` lagging by up to the global
period on fast HPAs and some additional controller state. It is the same shape
of fix as [KEP-589][] (node heartbeats moved to `Lease`), and since it would
benefit every HPA rather than only those setting the field, it is better
pursued as its own change if needed rather than folded into this one. It is
not part of the Alpha design.

**Worker starvation and cross-HPA interference.** This is the most significant
risk, and it is distinct from workqueue growth. The controller uses a fixed
worker pool sized by `--concurrent-horizontal-pod-autoscaler-syncs` (default 5).
Bounding queue depth does not bound demand for those workers.
`sum(1 / delay_i)` is an upper bound on requested periodic reconciliations per
second; the actual steady-state rate is lower because each delay begins after
reconciliation completes. Worker capacity is roughly
`concurrentSyncs / meanReconcileDuration`. Once offered load exceeds capacity,
every HPA's scheduling delay stretches, including HPAs that never set the field.
A per-object admission floor does not help, because aggregate load depends on
HPA count as much as on any individual delay.

Mitigations and residual risk:

- Administrators expecting short sync periods should raise
  `--concurrent-horizontal-pod-autoscaler-syncs`. A starting size follows from
  the SLIs below: sustaining an aggregate reconcile rate of
  `sum(1 / delay_i)` per second needs roughly
  `concurrentSyncs >= aggregate rate * mean reconciliation duration`, with the
  duration observable in
  `horizontal_pod_autoscaler_controller_reconciliation_duration_seconds`. The
  flag requires a `kube-controller-manager` restart per change, so it should
  be sized with headroom before short delays roll out rather than tuned
  iteratively; the delay-ratio metric then shows whether capacity suffices.
  HPA count per namespace can additionally be bounded with a `ResourceQuota`.
  This will be documented.
- The failure mode is graceful degradation -- longer scheduling delays --
  rather than unbounded queue growth, memory growth, or dropped work.
- The per-HPA delay ratio metric under
  [Monitoring Requirements](#monitoring-requirements) makes the condition
  observable.
- Residual risk is accepted for Alpha, behind the gate. If Alpha feedback shows
  this is a practical problem, workqueue fairness or a controller-level budget
  on aggregate reconcile rate are follow-ups for Beta.

**Overshoot from re-evaluating stale metrics.** With a short period and a slower
metrics source, the controller re-derives the same recommendation from the same
samples. Velocity is still bounded: `behavior.scaleUp` and `behavior.scaleDown`
policies use wall-clock `periodSeconds` windows, so a shorter period reduces
detection latency without raising the scaling ceiling. The HPA can still reach
that ceiling each window on stale data; users pairing a short period with a
slow source should set `behavior.scaleUp` policies.

**Reverting a bad value.** The field is optional; removing it restores current
behavior.

[ValidatingAdmissionPolicy]: https://kubernetes.io/docs/reference/access-authn-authz/validating-admission-policy/
[API Priority and Fairness]: https://kubernetes.io/docs/concepts/cluster-administration/flow-control/
[KEP-589]: /keps/sig-node/589-efficient-node-heartbeats

## Design Details

The `HorizontalPodAutoscaler` API is updated to add `syncPeriodSeconds` to
`HorizontalPodAutoscalerSpec` in both served versions, `autoscaling/v1` and
`autoscaling/v2`, with identical semantics and validation:

```golang
type HorizontalPodAutoscalerSpec struct {
  // syncPeriodSeconds is the delay after a reconciliation completes before
  // the next periodic reconciliation of this HPA is eligible to run.
  // Creation and spec-change events may trigger reconciliation sooner.
  // When unset, the global --horizontal-pod-autoscaler-sync-period is used
  // (default: 15s). Must be in [3, 3600]. Best-effort: worker contention
  // may delay the start. A shorter value only helps if the configured
  // metrics refresh more often than the delay.
  // +optional
  SyncPeriodSeconds *int32

  // Existing fields.
  ScaleTargetRef CrossVersionObjectReference
  MinReplicas *int32
  MaxReplicas int32
  Metrics []MetricSpec
  Behavior *HorizontalPodAutoscalerBehavior
}
```

**Fixed-delay, best-effort semantics**: after a reconciliation completes, the
controller schedules the next periodic reconciliation with the configured delay.
Reconciliation time is therefore additional to the delay; a 3s delay following a
2s reconciliation produces a roughly 5s start-to-start interval when the
controller is otherwise idle. Worker contention can make the interval longer.
Creation and spec-change events bypass the delay and can make it shorter. The
typed workqueue deduplicates each HPA key, so a slow metrics backend does not
create an unbounded duplicate backlog for one HPA. Aggregate capacity is
covered in [Risks and Mitigations](#risks-and-mitigations).

Per-HPA scheduling is implemented with a new `PerItemIntervalRateLimiter` in the
controller's workqueue, supporting per-key delay overrides with a fallback to
the global default. The controller sets the delay during reconciliation and
clears it on HPA deletion. It lives in `pkg/controller/podautoscaler` and
implements `workqueue.TypedRateLimiter`, so there is no staging API change.

The controller also tracks, per HPA, the completion time and requested delay
for the pending periodic requeue. This state distinguishes a scheduled
periodic dequeue from an earlier creation or spec-change dequeue, drives the
`horizontal_pod_autoscaler_controller_reconciliation_delay_ratio` metric, and
is cleared on deletion. When `syncPeriodSeconds` changes, the pending
eligibility deadline is recomputed; a stale delayed workqueue entry must not
cause a periodic reconciliation before the new deadline.

The informer event handlers are updated so that:
- Creations and spec changes (detected via `Generation`) are enqueued
  immediately with `queue.Add`, bypassing the rate limiter delay. This relies
  on `Generation` being incremented on spec changes, fixed in
  [kubernetes#138228][].
- Status-only and metadata-only updates continue to use `AddRateLimited`,
  preserving the hot-loop prevention from [#42715][].
- The periodic schedule comes from the existing `AddRateLimited` call in
  `processNextWorkItem`, which re-enqueues each HPA after every reconciliation
  with its configured (or default) delay.

`AddEventHandlerWithResyncPeriod` continues to use the global `resyncPeriod` as
a background safety net.

**Dependency on `HPAGeneration`**: immediate enqueue on spec changes relies on
`Generation` being incremented, which is gated by `HPAGeneration`. That gate
has been Beta and enabled by default since v1.37, and is planned to be locked
to enabled in v1.38, the release this KEP targets, so only a cluster that
explicitly disables it is affected.

On such a cluster, spec changes are picked up at the next periodic
reconciliation or informer resync rather than immediately. The delay is bounded
by the resync period -- the global sync period, 15s by default -- and not by the
previously configured `syncPeriodSeconds`, because the delaying workqueue only
ever moves an existing entry's ready time earlier: a re-enqueue carrying a
shortened delay takes effect at the next resync, while a lengthened delay is
enforced by the eligibility deadline described above. Should the lock to enabled
slip, the dependency can be removed outright by detecting spec changes in the
controller's update handler with a semantic comparison of `spec`, the same check
the registry strategy already performs in `PrepareForUpdate`. In every case this
is a degradation in responsiveness, not a correctness issue; periodic scheduling
itself does not depend on `Generation`.

Following the SIG convention that new HPA fields are added to both served API
versions rather than exposed through `v1` annotations, the field is added to
the internal type and to `autoscaling/v1` and `autoscaling/v2`. Because every
served version carries the field, `v1` <-> `v2` conversion is a direct copy,
and a client reading and writing back through either version preserves the
value, satisfying [round-trip requirements][]. The unserved `v2beta1` and
`v2beta2` versions are unaffected.

The bounds `3 <= syncPeriodSeconds <= 3600` are expressed as declarative
validation tags (`+k8s:minimum`, `+k8s:maximum`) on the field in both external
types, the same mechanism that already validates `minReplicas` and
`maxReplicas`, so each version's generated validation enforces the identical
range. An equivalent hand-written check on the internal type is marked
`MarkCoveredByDeclarative()` for the duration of the declarative validation
migration.

**Feature gate behavior in the API server**: while `HPAConfigurableSyncPeriod`
is Alpha, the field follows the standard disabled-field pattern in the registry
strategy. It is dropped on create when the gate is disabled, preserved on
update when already set on the existing object -- so disabling the gate does
not strip values during unrelated updates -- and bounds validation applies
only when the field survives the drop step. This is the pattern
`dropDisabledFields` in `pkg/registry/autoscaling/horizontalpodautoscaler`
already implements for `HPAConfigurableTolerance` ([KEP-4951][]); that helper is
restructured so each gated field is dropped independently rather than returning
early on the first enabled gate.

**Interaction with stabilization windows**: a shorter period produces more
recommendations within a given `stabilizationWindowSeconds` (roughly 60 rather
than 20 for a 5s period in a 300s window). Stabilization still selects the
lowest recent recommendation for scale-up and the highest for scale-down, so
more frequent sampling can change which extrema are observed and therefore
the timing of a scaling decision. The windows and selection algorithm
themselves are unchanged.

**Interaction with pod readiness and CPU initialization**: the controller still
ignores CPU samples from pods within
`--horizontal-pod-autoscaler-cpu-initialization-period` (default 5m) or
`--horizontal-pod-autoscaler-initial-readiness-delay` (default 30s). A short
period shortens the time to detect load, not the time for a completed scale-up
to become visible in resource metrics.

[#42715]: https://github.com/kubernetes/kubernetes/pull/42715
[kubernetes#138228]: https://github.com/kubernetes/kubernetes/pull/138228
[kubernetes#138294]: https://github.com/kubernetes/kubernetes/pull/138294
[round-trip requirements]: https://kubernetes.io/docs/reference/using-api/deprecation-policy/#deprecating-parts-of-the-api

### Test Plan

[x] I/we understand the owners of the involved components may require updates to
existing tests to make this code solid enough prior to committing the changes necessary
to implement this enhancement.

##### Prerequisite testing updates

Existing unit tests in `pkg/controller/podautoscaler` build the controller with a
single global sync period and assert behavior without controlling time. They will
be refactored to allow a per-HPA delay and a fake clock to be injected, so
timing can be asserted deterministically. No change to existing test semantics is
expected.

##### Unit tests

- `pkg/apis/autoscaling/validation`: 2026-08-31 - 95.3% of statements
- `pkg/controller/podautoscaler`: 2026-08-31 - 89.0% of statements
- `pkg/registry/autoscaling/horizontalpodautoscaler`: 2026-08-31 - 56.9% of
  statements

Unit tests will cover:

- Validation accepts an unset field and values in `[3, 3600]`, and rejects values
  outside that range.
- Disabled-field behavior with the gate on and off, for both create (dropped) and
  update (preserved when already set).
- `PerItemIntervalRateLimiter` returns the per-key delay when set, falls back
  to the global default when not, and forgets per-key state when a key is
  removed.
- The configured delay starts after reconciliation completes, while creation and
  generation-change events enqueue the HPA immediately.
- The per-HPA scheduling-delay metric is not emitted for the first periodic
  reconciliation, is updated on subsequent periodic reconciliations, ignores
  early event-driven reconciliations, and is removed when the HPA is deleted.

##### Integration tests

Integration tests are the primary place where timing is asserted, because they
can drive a fake clock:

- After reconciliation completes, an HPA with `syncPeriodSeconds` set becomes
  eligible for its next periodic reconciliation after the configured delay; one
  without it uses the global delay.
- Changing `syncPeriodSeconds` on an existing HPA enqueues the spec change
  immediately. Both shorter and longer new delays govern the following periodic
  requeue, without a stale delayed entry using the old value.
- With the gate disabled, an HPA that already carries the field reconciles at the
  global default.

##### e2e tests

Existing e2e tests cover the default sync period when the field is unset. Exact
timing assertions are timing-sensitive and flake-prone, so precise delay
checks live in the integration tests above and e2e coverage stays coarse:

- An HPA with a short `syncPeriodSeconds` scales in response to a step change in
  load measurably sooner than an otherwise identical HPA at the cluster default.
- An HPA with a large `syncPeriodSeconds` scales noticeably later than the
  default, confirming the field is honored in both directions.

### Graduation Criteria

#### Alpha

- SIG Scalability has reviewed the API server cost model and the 3s lower
  bound in [Risks and Mitigations](#risks-and-mitigations)
- Feature implemented behind a `HPAConfigurableSyncPeriod` feature flag
- Unit and integration tests described above implemented and enabled
- Per-HPA reconciliation scheduling-delay metric available, so operators can
  tell whether configured delays are being met
- The e2e tests described in the [`e2e tests` section](#e2e-tests), covering
  both shorter and longer periods than the default, implemented and enabled
- Documentation drafted on kubernetes/website describing the field, its
  best-effort semantics, and when it is and is not effective

#### Beta

- The `HPAConfigurableSyncPeriod` feature gate is enabled by default.
- The e2e tests implemented in Alpha are linked in this KEP and have run
  flake-free for at least two weeks.
- We have monitored for negative user feedback and addressed relevant concerns.
- Real-world usage data -- from the delay-ratio metric on gated clusters,
  early adopters such as the KEDA HTTP Add-on, and feedback on the KEP issue
  and SIG meetings -- reviewed with SIG Scalability, following their Alpha
  review of the bound, to decide whether the lower validation bound should be
  relaxed below 3s, and whether cross-HPA interference warrants workqueue
  fairness or a controller-level budget on aggregate reconcile rate.
- Documentation covering when the field is and is not effective, and the
  interaction with `--concurrent-horizontal-pod-autoscaler-syncs`.

#### GA

- The feature has been Beta for at least 2 releases with no major issues
  reported, so the Beta lower-bound decision and any fairness follow-ups have
  at least one release of production usage behind them before the feature
  gate is locked to enabled.
- The `HPAConfigurableSyncPeriod` feature gate is locked to enabled and
  non-operational, then removed no sooner than two releases later. That floor
  comes from [Rule #9][] of the deprecation policy and from the
  [version skew policy][]: with one minor version of `kube-apiserver` skew
  permitted in HA clusters, the flag must still be accepted in the release
  after the lock.
- Documentation is published on kubernetes.io.

[Rule #9]: https://kubernetes.io/docs/reference/using-api/deprecation-policy/
[version skew policy]: https://kubernetes.io/releases/version-skew-policy/#kube-apiserver

### Upgrade / Downgrade Strategy

#### Upgrade
Existing HPAs continue to work as they do today, using the global
`--horizontal-pod-autoscaler-sync-period` value. Users opt in by enabling the
feature gate (alpha only) and setting `syncPeriodSeconds` on an HPA.

#### Downgrade
On downgrade, all HPAs revert to the global
`--horizontal-pod-autoscaler-sync-period` value, regardless of any configured
`syncPeriodSeconds` on the HPA itself.

### Version Skew Strategy

1. `kube-apiserver`: more recent instances accept `syncPeriodSeconds`; older
   instances do not recognize it. During a rolling control-plane upgrade an
   older API server may reject the field under strict field validation or prune
   it otherwise, so a client that reads, modifies, and writes back an HPA may
   lose a previously set value until every API server is upgraded. The effect is
   a return to the global sync period for that HPA; re-applying the manifest
   after the upgrade restores it.
2. `kube-controller-manager`: an older version receiving an HPA that carries the
   field from a newer API server ignores it and reconciles at the global default.

## Production Readiness Review Questionnaire

### Feature Enablement and Rollback

###### How can this feature be enabled / disabled in a live cluster?

- [x] Feature gate (also fill in values in `kep.yaml`)
  - Feature gate name: HPAConfigurableSyncPeriod
  - Components depending on the feature gate: `kube-controller-manager` and
    `kube-apiserver`.

###### Does enabling the feature change any default behavior?

No.

###### Can the feature be disabled once it has been enabled (i.e. can we roll back the enablement)?

Yes, by restarting the `kube-apiserver` and `kube-controller-manager` with the
gate set to `false`. The API server then drops the field on create and the
controller schedules every HPA with the global sync delay. Values already
persisted are preserved across updates rather than stripped (see
[Design Details](#design-details)), so re-enabling the gate restores the previous
behavior without re-applying manifests.

###### What happens if we reenable the feature if it was previously rolled back?

HPAs with a configured `syncPeriodSeconds` use it again as their periodic
reconciliation delay, in place of the global delay.

###### Are there any tests for feature enablement/disablement?

Unit tests will verify that the field is dropped on create and preserved on
update when the gate is disabled, and that bounds validation applies only when
the field is present. An integration test will verify that an HPA already
carrying the field reconciles at the global default when the gate is disabled in
the `kube-controller-manager`.

### Rollout, Upgrade and Rollback Planning

###### How can a rollout or rollback fail? Can it impact already running workloads?

During rollout or rollback, mixed API server versions may accept, reject, or
prune the field depending on their version and field-validation mode, and older
controller managers ignore it. Once users begin setting short delays, aggregate
reconcile demand can also exceed the worker pool's capacity; see
[Risks and Mitigations](#risks-and-mitigations).

The user-visible consequence of a rollback is that HPAs relying on a short sync
period revert to the global default and become less responsive. That is a
latency regression rather than an outage: replica counts are still computed
from the same metrics with the same algorithm.

###### What specific metrics should inform a rollback?

- `horizontal_pod_autoscaler_controller_reconciliation_delay_ratio` exceeding
  1.2 across many HPAs, meaning configured scheduling delays are not being met.
- HPA controller workqueue metrics (`workqueue_depth`,
  `workqueue_queue_duration_seconds`) rising, meaning the worker pool is at or
  beyond capacity.
- `horizontal_pod_autoscaler_controller_reconciliation_duration_seconds`
  rising, meaning full reconciliations are taking longer and reducing worker
  capacity.
- `horizontal_pod_autoscaler_controller_metric_computation_duration_seconds`
  rising can identify metric retrieval or computation as the cause.
- Unexpected growth in API server request rates from the HPA controller, which
  may mean sync periods are set too aggressively.

###### Were upgrade and rollback tested? Was the upgrade->downgrade->upgrade path tested?

Not yet; this is not required for Alpha, and cannot be exercised before the
field exists in a built binary. The unit and integration coverage for gate
enablement and disablement described in the [Test Plan](#test-plan) lands in
Alpha, and the manual upgrade -> downgrade -> upgrade path will be run on a
local cluster once the Alpha implementation merges, following these steps:

1. Start a cluster with `HPAConfigurableSyncPeriod` enabled on the
   `kube-apiserver` and `kube-controller-manager`. Create an HPA with
   `syncPeriodSeconds` set and confirm from the delay-ratio metric and
   controller logs that the configured delay is used.
2. Restart both components with the gate disabled. Confirm that the persisted
   value is still present in the object, that an unrelated update to the HPA
   does not strip it, that a newly created HPA has the field dropped, and that
   the controller schedules every HPA at the global sync period.
3. Restart both components with the gate enabled again. Confirm the original
   HPA is once more reconciled at its configured delay without re-applying the
   manifest.

The results will be recorded here before the Beta graduation review.

###### Is the rollout accompanied by any deprecations and/or removals of features, APIs, fields of API types, flags, etc.?

No.

### Monitoring Requirements

###### How can an operator determine if the feature is in use by workloads?

The presence of `syncPeriodSeconds` in an HPA's `spec`.

###### How can someone using this feature know that it is working for their instance?

- [X] Metrics
  - Metric name: `horizontal_pod_autoscaler_controller_reconciliation_delay_ratio`
  - Components exposing the metric: `kube-controller-manager`
- [ ] Events
- [X] Other
  - Controller logs at verbosity >= 4 include the configured and observed
    scheduling delay for each HPA.

A new gauge,
`horizontal_pod_autoscaler_controller_reconciliation_delay_ratio{namespace,hpa_name}`,
records the latest periodic scheduling delay for each HPA divided by the delay
that was requested for that requeue. The measured delay starts when the previous
reconciliation completes and ends when the next scheduled periodic
reconciliation starts, so reconciliation duration itself is not included. A
value near 1 means the controller met the requested delay; a sustained value
above 1 means queueing or worker contention made it start late.

The first periodic reconciliation has no prior completion from which to
measure, and creation or spec-change reconciliations are ignored. Metric state
is removed when an HPA is deleted. The `namespace` and `hpa_name` labels add
one series per HPA, the same cardinality class as
`horizontal_pod_autoscaler_controller_desired_replicas`. They let an operator
inspect one HPA or count HPAs over a threshold; an unlabelled histogram would
count samples, not affected HPAs, and would weight short-delay HPAs more
heavily. `SuccessfulRescale` events are not used: they fire only on a scaling
decision, so a stable HPA produces none. Access on a managed control plane
depends on the provider exposing `kube-controller-manager` metrics.

Because the gauge records only the most recent cycle, SLO evaluation and
alerting should aggregate over a time window -- for example the fraction of
scrapes above the threshold, or `avg_over_time` -- rather than reading
instantaneous values, so that a single on-time cycle cannot mask a
persistently late HPA. If Alpha usage shows windowed gauge queries are awkward
in practice, an unlabelled histogram of the same ratio can be added in Beta to
expose the distribution directly.

###### What are the reasonable SLOs (Service Level Objectives) for the enhancement?

At least 99% of HPAs with a recorded periodic requeue should have
`horizontal_pod_autoscaler_controller_reconciliation_delay_ratio` at or below
1.2. Sustained violations are an operator-actionable capacity signal and should
prompt rollback, longer configured delays, or an increase to
`--concurrent-horizontal-pod-autoscaler-syncs`.

Enabling the gate while no HPA sets the field should cause no material regression
in `horizontal_pod_autoscaler_controller_reconciliation_duration_seconds`, since
the feature does not change per-reconcile computation.

###### What are the SLIs (Service Level Indicators) an operator can use to determine the health of the service?

- `horizontal_pod_autoscaler_controller_reconciliation_delay_ratio` -- latest
  per-HPA scheduling delay relative to the requested delay.
- `horizontal_pod_autoscaler_controller_reconciliation_duration_seconds` --
  full per-reconcile execution time and therefore a direct input to worker
  capacity.
- `horizontal_pod_autoscaler_controller_metric_computation_duration_seconds` --
  metric retrieval and computation time, useful for diagnosing an increase in
  full reconciliation duration.
- HPA controller workqueue metrics (`workqueue_depth`,
  `workqueue_queue_duration_seconds`) -- whether the worker pool is keeping up
  with the aggregate requested reconcile rate.

###### Are there any missing metrics that would be useful to have to improve observability of this feature?

The per-HPA scheduling-delay gauge above does not exist today and is required by
this feature; it is part of the Alpha graduation criteria rather than an
outstanding gap. No additional metric is required for Alpha.

### Dependencies

###### Does this feature depend on any specific services running in the cluster?

No.

### Scalability

###### Will enabling / using this feature result in any new API calls?

No new call types. A lower `syncPeriodSeconds` increases the frequency of
existing calls for that HPA. Those calls fall into two classes with materially
different costs:

- **Storage-backed `kube-apiserver` calls**: a `GET` on the target's `scale`
  subresource on every reconciliation; an `UpdateStatus` on the
  `HorizontalPodAutoscaler` whenever the recomputed status differs from the
  previous one, which for an active workload is most reconciliations because
  `status.currentMetrics` tracks live metric values; and an `UPDATE` on the
  `scale` subresource only when the replica count actually changes. These are
  the calls that reach etcd, and their rate is proportional to the configured
  frequency.
- **Aggregation-layer calls**: `metrics.k8s.io`, `custom.metrics.k8s.io`, and
  `external.metrics.k8s.io` queries, which the `kube-apiserver` authenticates,
  authorizes, and proxies to the extension API server registered for them.
  These never reach etcd, but they are still API server work -- authentication,
  authorization, APF classification, and proxying consume apiserver CPU and
  connections -- and the cost of serving them lands on metrics-server or the
  metrics adapter. For the request-driven workloads this feature targets, this
  is where most of the additional traffic goes.

Pod and HPA reads are served from informer caches and are unaffected by the
configured period. The overall increase is approximately proportional when
reconciliation duration and event-driven work are small relative to the
configured delay: `15 / period` per HPA that sets the field, at most 5x at the
3s floor. In expected usage only the request-driven subset of HPAs opts in, and
often at 5s or 10s rather than the floor; 50 such HPAs at 5s in a cluster of
1,000 add roughly 10% to the controller's reconciliation rate. As a worst-case
anchor, 1,000 HPAs all at 3s issue about 333 reconciliations/s, or roughly
1,000 API server requests/s -- equivalent to 5,000 HPAs at the default period.
The full cost model is in [Risks and Mitigations](#risks-and-mitigations).

###### Will enabling / using this feature result in introducing new API types?

No.

###### Will enabling / using this feature result in any new calls to the cloud provider?

No.

###### Will enabling / using this feature result in increasing size or count of the existing API objects?

One optional int32 field is added to `HorizontalPodAutoscaler` objects in both
served versions (`v1` and `v2`). The per-object increase is small but greater
than the raw four-byte integer because of pointer, field, and serialization
overhead.

###### Will enabling / using this feature result in increasing time taken by any operations covered by existing SLIs/SLOs?

Not for any individual operation, since per-reconcile work is unchanged. The
aggregate reconcile rate can increase, and if it exceeds the worker pool's
capacity then scheduling delays grow for all HPAs, including those that do not
set the field; see [Risks and Mitigations](#risks-and-mitigations).

###### Will enabling / using this feature result in non-negligible increase of resource usage (CPU, RAM, disk, IO, ...) in any components?

Lower values proportionally increase CPU usage and API call volume on the
`kube-controller-manager` and the metrics backend for the affected HPAs.
Operators should watch controller-manager resource usage and the SLIs above when
short periods are used on many HPAs, and size
`--concurrent-horizontal-pod-autoscaler-syncs` accordingly.

###### Can enabling / using this feature result in resource exhaustion of some node resources (PIDs, sockets, inodes, etc.)?

No node-level exhaustion is expected. Increased reconciliation can raise CPU,
network, and socket use in the control plane and metrics backend, but the fixed
worker count bounds concurrent calls from the HPA controller.

### Troubleshooting

###### How does this feature react if the API server and/or etcd is unavailable?

No feature-specific impact. The controller cannot reconcile regardless of sync
period configuration while the API server is unavailable.

###### What are other known failure modes?

- **Worker-pool exhaustion.** Short periods on many HPAs can drive the aggregate
  requested reconcile rate above what the controller can sustain. The symptom is
  observed scheduling delays exceeding configured ones for all HPAs, including
  those that do not set the field. Detection is via the SLIs above. Mitigation
  is to raise `--concurrent-horizontal-pod-autoscaler-syncs`, increase the
  affected delays, or remove the field.
- **No improvement on a resource-metric HPA.** Expected when the metrics source
  is slower than the configured period; see [Proposal](#proposal).

###### What steps should be taken if SLOs are not being met to determine the problem?

1. Check `horizontal_pod_autoscaler_controller_reconciliation_delay_ratio` for
   the affected HPA and across the cluster.
2. Check the workqueue metrics to confirm whether the worker pool is
   saturated, then compare
   `horizontal_pod_autoscaler_controller_reconciliation_duration_seconds` and
   aggregate requested reconcile rate against
   `--concurrent-horizontal-pod-autoscaler-syncs`.
3. Check whether the increased frequency is overloading the API server or the
   metrics backend.
4. Raise the `kube-controller-manager` log level and look for warnings and errors
   that point to the cause.

## Implementation History

- 2022-06-01: [kubernetes#110317][] opened, requesting a per-HPA sync period.
  This predates the KEP process and is the original tracking issue.
- 2026-04-06: [Prototype implementation](https://github.com/kubernetes/kubernetes/pull/138222)
  opened, since closed; it will be reopened against the design this KEP settles on.
- 2026-04-08: Initial KEP created.
- 2026-04-24: [kubernetes#138228][] merged -- HPA generation tracking.
- 2026-05-07: [kubernetes#138294][] merged -- immediate enqueue on HPA creation
  and spec changes.
- 2026-08-19: KEP marked `implementable`, targeting alpha in v1.38.

## Drawbacks

Short sync periods on many HPAs can stretch scheduling delays for unrelated
HPAs; see [Risks and Mitigations](#risks-and-mitigations).

The field exposes a controller implementation detail through a stable API: how
often the HPA controller polls is a property of the current poll-based design.
Once at GA it must be honored indefinitely, including by a future event-driven
implementation in its periodic fallback or rate limiting. Best-effort semantics
limit this constraint but do not remove it. The field is also easy to misuse on
CPU/memory HPAs, where a short period does not freshen metric input; that is a
documentation burden, not an API-complexity trade-off we chose to take.

## Alternatives

- **Change the global flag**: affects all HPAs and requires cluster-admin
  access and a controller-manager restart.
- **Per-HPA annotation**: rejected in favor of a first-class field with
  validation, documentation, and discoverability.
- **A `behavior.syncPeriodSeconds` field**:
  `SetDefaults_HorizontalPodAutoscalerBehavior` in `pkg/apis/autoscaling/v2`
  keys off `spec.behavior != nil`. Setting only a sync period would
  materialize default scaling rules (stabilization window, `selectPolicy`, and
  policies for both directions) into the object, pin them in etcd, and produce
  Server-Side Apply / GitOps diffs. Defaulting also runs during decoding,
  before disabled-field dropping, so a gate-disabled cluster would get a fully
  defaulted `behavior` block and no `syncPeriodSeconds`. [KEP-4951][] avoids
  this because setting a tolerance means explicitly touching a scaling rule.
- **Scope the field to custom and external metrics only**: rejected; see
  [Proposal](#proposal).
- **Fixed-rate start-to-start scheduling**: would subtract reconciliation
  duration from the next delay. Rejected because the existing global flag and
  workqueue use a fixed delay after completion, and an overrun would otherwise
  cause an immediate next reconciliation.
- **Adaptive sync period**: reconcile faster when far from target and back off
  when stable. Not chosen because it changes timing for every existing HPA, is
  harder to predict and test, and is still capped by the global period.
  Complementary: an adaptive scheme could treat `syncPeriodSeconds` as its
  floor.
- **Component config or per-namespace defaults**: does not satisfy
  [Story 1](#story-1); an adapter generating HPAs cannot change
  controller-manager configuration.

## Infrastructure Needed (Optional)

N/A.
