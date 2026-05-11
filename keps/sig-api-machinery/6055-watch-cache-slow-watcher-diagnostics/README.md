# KEP-6055: Watch Cache Slow Watcher Diagnostics

<!-- toc -->
- [Release Signoff Checklist](#release-signoff-checklist)
- [Summary](#summary)
- [Motivation](#motivation)
  - [Goals](#goals)
  - [Non-Goals](#non-goals)
- [Proposal](#proposal)
  - [Metrics](#metrics)
    - [<code>apiserver_watch_cache_watcher_terminated_total</code>](#apiserver_watch_cache_watcher_terminated_total)
    - [<code>apiserver_watch_cache_init_events_duration_seconds</code>](#apiserver_watch_cache_init_events_duration_seconds)
    - [Existing <code>apiserver_init_events_total</code>](#existing-apiserver_init_events_total)
  - [Structured Logging](#structured-logging)
  - [Reason Classification](#reason-classification)
  - [User Stories](#user-stories)
    - [Story 1: Identify client backpressure by resource](#story-1-identify-client-backpressure-by-resource)
    - [Story 2: Separate watch-list initialization cost from client reads](#story-2-separate-watch-list-initialization-cost-from-client-reads)
    - [Story 3: Diagnose bookmark-delivery pressure](#story-3-diagnose-bookmark-delivery-pressure)
- [Risks and Mitigations](#risks-and-mitigations)
- [Design Details](#design-details)
  - [Implementation Notes](#implementation-notes)
  - [Test Plan](#test-plan)
  - [Graduation Criteria](#graduation-criteria)
    - [Alpha](#alpha)
    - [Beta](#beta)
    - [Stable](#stable)
- [Production Readiness Review Questionnaire](#production-readiness-review-questionnaire)
  - [Feature Enablement and Rollback](#feature-enablement-and-rollback)
  - [Rollout, Upgrade and Rollback Planning](#rollout-upgrade-and-rollback-planning)
  - [Monitoring Requirements](#monitoring-requirements)
  - [Dependencies](#dependencies)
  - [Scalability](#scalability)
  - [Troubleshooting](#troubleshooting)
- [Drawbacks](#drawbacks)
- [Alternatives](#alternatives)
  - [Add a <code>reason</code> label to the existing metric](#add-a-reason-label-to-the-existing-metric)
  - [Only improve logs](#only-improve-logs)
  - [Add watcher identifiers as metric labels](#add-watcher-identifiers-as-metric-labels)
  - [Change buffer sizes or termination behavior](#change-buffer-sizes-or-termination-behavior)
- [Implementation History](#implementation-history)
<!-- /toc -->

## Release Signoff Checklist

- [ ] (R) Enhancement issue in release milestone, which links to KEP dir in kubernetes/enhancements
- [ ] (R) KEP approvers have approved the KEP status as `implementable`
- [ ] (R) Design details are appropriately documented
- [ ] (R) Test plan is in place
- [ ] (R) Graduation criteria is in place
- [ ] (R) Production readiness review completed
- [ ] (R) Production readiness review approved
- [ ] "Implementation History" section is up-to-date for milestone
- [ ] User-facing documentation has been created in kubernetes/website

## Summary

Improve kube-apiserver watch cache diagnostics for slow or blocked watch clients
by adding low-cardinality metrics and structured logs around cache watcher
termination and initial event processing.

The watch cache currently closes a cache watcher when the watcher cannot accept
events quickly enough. The apiserver already exposes a counter for these
terminations and logs the watcher identifier, buffer lengths, and whether the
watcher is closed gracefully. However, the metric does not classify why the
watcher was terminated, and the initialization path only logs slow initial event
processing after a fixed threshold.

This KEP proposes diagnostic-only observability improvements:

- classify forced cache watcher terminations by a bounded `reason` label;
- record initial event processing duration as a histogram;
- keep the existing initial event counter;
- convert slow watcher and slow initialization logs to structured logs; and
- avoid any change to watch semantics, buffer sizing, termination policy, or
  client-visible API behavior.

## Motivation

Large Kubernetes clusters can have many long-running watches and watch-list
requests. When a watch client is slow, disconnected, or unable to read quickly
enough, the watch cache must avoid blocking dispatch to other watchers. Today
the apiserver protects itself by terminating unresponsive cache watchers, but
operators have limited visibility into whether the pressure was caused by:

- a watcher's input buffer filling while dispatching watch cache events;
- the watcher's result buffer being full while the processing goroutine tries to
  deliver events to the client;
- a watcher waiting for the initial-events-end bookmark to reach the client; or
- expensive initial event processing before the watcher begins processing new
  incoming events.

The current signal is too coarse for fleet-wide diagnosis. Operators can see
that watch cache watchers were terminated for a resource, but not whether this
correlates with client backpressure, watch-list bookmark delivery, or slow
initial event processing.

### Goals

- Add bounded, low-cardinality metrics for forced watch cache watcher
  terminations.
- Add a duration histogram for initial event processing in `cacheWatcher`.
- Preserve the existing initial event counter and avoid duplicate counters for
  the same event count.
- Improve logs with structured key-value fields that are already available in
  the cache watcher code path.
- Keep all new labels bounded and independent of object names, namespaces,
  users, clients, request URIs, object keys, or error strings.
- Avoid changes to watch behavior, watch cache dispatch behavior, termination
  policy, bookmark semantics, and buffer sizes.

### Non-Goals

- Change when or how cache watchers are terminated.
- Change the watch cache input or result buffer sizes.
- Add per-client, per-user, per-namespace, per-object, or per-request metrics.
- Change the watch API, List API, or watch-list semantics.
- Add tracing spans or distributed tracing integration.
- Replace existing apiserver watch metrics.
- Guarantee that a metric reason identifies the remote client-side root cause.
  The metrics classify the server-side condition observed by the watch cache.

## Proposal

Add ALPHA apiserver metrics and structured logs in the watch cache watcher path.

### Metrics

#### `apiserver_watch_cache_watcher_terminated_total`

- Type: Counter
- Stability: ALPHA
- Labels:
  - `group`
  - `resource`
  - `reason`
- Description: Number of watch cache watchers forcibly closed by the watch
  cache because the watcher could not accept events quickly enough.

`reason` values:

- `input_buffer_full`: the watcher input channel remained full until the
  dispatch timeout budget expired, or the dispatch path attempted an immediate
  close after the timeout had already fired.
- `result_buffer_full`: the watcher input channel remained full and the
  watcher's result channel was also full at close time. This indicates the
  processing goroutine was likely blocked delivering events to the watch result
  channel.
- `bookmark_pending`: the watcher had received the bookmark that satisfies
  `bookmarkAfterResourceVersion`, but that bookmark had not yet been delivered
  to the client at close time. The watcher is closed with input-buffer draining
  enabled to allow the bookmark to make progress.

The `result_buffer_full` and `bookmark_pending` reasons are server-side
classifications based on cache watcher state at the time of forced closure. They
do not assert a client-side root cause.

The existing `apiserver_terminated_watchers_total{group,resource}` metric
remains registered and continues to be incremented for compatibility. The new
metric provides a watch-cache-scoped name and a bounded reason label. This KEP
does not add a `reason` label to the existing metric to avoid changing its label
set.

#### `apiserver_watch_cache_init_events_duration_seconds`

- Type: Histogram
- Stability: ALPHA
- Labels:
  - `group`
  - `resource`
- Description: Time spent processing initial events for a cache watcher before
  it starts processing incoming watch cache events.

Initial event processing duration starts before reading the first event from the
watch cache interval and ends after the initial interval is exhausted. The
duration does not include asynchronously writing buffered watch results to the
client after the event has been placed into the watcher's result channel.

Suggested buckets:

- `0.005`
- `0.025`
- `0.05`
- `0.1`
- `0.2`
- `0.4`
- `0.6`
- `0.8`
- `1.0`
- `1.25`
- `1.5`
- `2`
- `3`

These buckets intentionally match the existing
`apiserver_watch_cache_read_wait_seconds` bucket shape to make watch cache
latency dashboards easier to compare.

#### Existing `apiserver_init_events_total`

The existing `apiserver_init_events_total{group,resource}` metric remains the
counter of initial events processed by resource. This KEP does not introduce a
second initial event counter with the watch cache subsystem prefix, because that
would duplicate the same count under a different name.

### Structured Logging

Convert slow watcher forced-close logs and slow initial event processing logs to
structured `klog.InfoS` logs.

Forced watcher close log fields:

- `group`
- `resource`
- `watcher`
- `reason`
- `inputBufferLength`
- `inputBufferCapacity`
- `resultBufferLength`
- `resultBufferCapacity`
- `graceful`
- `bookmarkState`

Slow initial event processing log fields:

- `group`
- `resource`
- `watcher`
- `initEventCount`
- `duration`
- `threshold`

The `watcher` field uses the existing human-readable cache watcher identifier.
It must not be added as a metric label.

### Reason Classification

The implementation should classify forced termination in the cache watcher close
path using only state already available to the watcher:

1. If the watcher is in the state where the required bookmark has been received
   but not sent, record `reason="bookmark_pending"` and close gracefully.
2. Else if `len(result) == cap(result)` at close time, record
   `reason="result_buffer_full"`.
3. Else record `reason="input_buffer_full"`.

This classification keeps cardinality bounded and reflects observable
server-side conditions without adding timers or changing channel behavior. The
classification is intentionally ordered. A watcher may have a full result
channel while also having a required bookmark pending, but the implementation
records `bookmark_pending` first because preserving delivery of that bookmark is
the behaviorally significant state that determines graceful draining. Operators
should treat the reason as the server-side condition selected at forced-close
time, not as an exclusive client-side root cause.

### User Stories

#### Story 1: Identify client backpressure by resource

A cluster operator observes high watch latency and increasing watch reconnects.
They query:

```promql
sum by (group, resource, reason) (
  rate(apiserver_watch_cache_watcher_terminated_total[5m])
)
```

They see `reason="result_buffer_full"` for pods. This indicates that cache
watchers are likely blocked while delivering events to clients, and the
operator can focus on watch clients consuming pod streams.

#### Story 2: Separate watch-list initialization cost from client reads

During a large workload rollout, operators see increased watch cache watcher
termination for a custom resource. They compare it with:

```promql
histogram_quantile(
  0.99,
  sum by (le, group, resource) (
    rate(apiserver_watch_cache_init_events_duration_seconds_bucket[5m])
  )
)
```

If initial event duration is also high, they can investigate large initial
watch-list result sets, expensive filtering, or CPU pressure on apiserver
instances.

#### Story 3: Diagnose bookmark-delivery pressure

An operator sees termination reason `bookmark_pending` for resources using
watch-list. This indicates that the apiserver received the bookmark that would
allow the client to resume from a newer resource version, but the bookmark had
not yet reached the client when the watcher was forced closed. That points to
result delivery pressure rather than lack of watch cache progress.

## Risks and Mitigations

Metric cardinality is the primary risk. The proposed metrics only use
`group`, `resource`, and a bounded `reason` label. They do not include watcher
identifier, request URI, user, namespace, object key, client address, or error
strings.

The reason classification is intentionally conservative. `result_buffer_full`
is a server-side observation at close time and may not be the original root
cause. Documentation and metric help text should avoid implying that the remote
client is always at fault.

The structured `watcher` log field contains the same human-readable watcher
identifier that is already logged today. That identifier includes the watch key
and selectors, so it must remain a log field only and must not be copied into
metric labels. The implementation should keep the existing log verbosity and
avoid adding new per-event logs, so log volume only increases for the existing
forced-close and slow-initialization diagnostic paths.

The existing `apiserver_terminated_watchers_total` metric remains unchanged to
avoid breaking dashboards that rely on its current name and labels. The new
watch-cache-scoped metric duplicates the total count with more detail. During
graduation, SIG API Machinery and SIG Instrumentation should decide whether the
old metric remains indefinitely or is deprecated under the metric stability
policy.

## Design Details

### Implementation Notes

The primary implementation area is:

- `staging/src/k8s.io/apiserver/pkg/storage/cacher/cache_watcher.go`
- `staging/src/k8s.io/apiserver/pkg/storage/cacher/metrics/metrics.go`

`cacheWatcher.add()` already detects when an event cannot be added to a watcher
input channel before the dispatch timeout expires. The implementation can
classify the forced close in the existing close function before calling
`forget(graceful)`.

`cacheWatcher.processInterval()` already records the count of initial events
and measures processing duration for threshold logging. The implementation can
observe the same duration in the new histogram for every watcher, not only when
the threshold is exceeded.

### Test Plan

Unit tests should cover:

- forced watcher termination increments the legacy
  `apiserver_terminated_watchers_total` metric;
- forced watcher termination increments
  `apiserver_watch_cache_watcher_terminated_total` with
  `reason="input_buffer_full"`;
- termination while the result buffer is full increments the new metric with
  `reason="result_buffer_full"`;
- termination after the required bookmark was received but before it was sent
  increments the new metric with `reason="bookmark_pending"` and preserves
  graceful draining behavior;
- initial event processing observes
  `apiserver_watch_cache_init_events_duration_seconds`; and
- metric labels do not include unbounded values.

Existing cache watcher draining tests should continue to pass without behavior
changes.

### Graduation Criteria

#### Alpha

- New metrics are registered at ALPHA stability.
- Structured logs are emitted on forced watcher close and slow initial event
  processing.
- Unit tests cover metric recording and reason classification.
- Existing watch behavior and existing termination counter remain unchanged.

#### Beta

- Operators have used the metrics in large-cluster or scale-test environments.
- Documentation includes example PromQL queries and reason semantics.
- SIG API Machinery and SIG Instrumentation agree that the reason labels are
  bounded and useful.
- The compatibility plan for `apiserver_terminated_watchers_total` is
  documented.

#### Stable

- Metrics have remained useful and low-cardinality for at least two releases.
- No reason labels were removed or renamed after Beta.
- Documentation and runbooks use the metrics for slow watcher diagnosis.

## Production Readiness Review Questionnaire

### Feature Enablement and Rollback

###### How can this feature be enabled / disabled in a live cluster?

This feature is enabled by running a kube-apiserver binary that includes the new
instrumentation. It does not introduce a feature gate because it only adds ALPHA
metrics and structured logs on existing code paths.

The feature can be disabled by rolling back or replacing the kube-apiserver
binary with a version that does not include the instrumentation. That requires a
normal control plane rollout of kube-apiserver instances. It does not require
node downtime or node reprovisioning.

###### Does enabling the feature change any default behavior?

No. Watch semantics, watch cache dispatch behavior, termination policy, bookmark
semantics, buffer sizes, and client-visible API behavior are unchanged. The only
default-visible changes are additional metrics and structured log fields on
existing diagnostic log paths.

###### Can the feature be disabled once it has been enabled?

Yes, by rolling back the kube-apiserver binary. Existing watch behavior and
existing metrics remain unchanged. The new metrics disappear after rollback, so
dashboards and alerts should tolerate missing time series during rollback or
skewed control plane upgrades.

###### What happens if we reenable the feature if it was previously rolled back?

The new metrics and structured logs appear again on upgraded kube-apiserver
instances. Counter values restart with the process lifetime as usual for
Prometheus counters, and Prometheus handles stale time series from rolled-back
instances.

###### Are there any tests for feature enablement/disablement?

There are no feature-gate enablement or disablement tests because there is no
feature gate and no persisted state. Unit tests should verify that the new
instrumentation records the expected metrics and preserves the existing watcher
close behavior.

### Rollout, Upgrade and Rollback Planning

###### How can a rollout fail? Can it impact already running workloads?

A rollout should not affect already running workloads because it does not change
API behavior or watch cache behavior. Possible failure modes are limited to
instrumentation mistakes, such as metric registration errors, unexpectedly high
metric series count, increased kube-apiserver CPU or memory usage from metric
observation, or unexpected log volume on clusters with many forced watcher
closures.

During a skewed control plane upgrade, only upgraded kube-apiserver instances
emit the new metrics and structured logs. Operators should aggregate by instance
carefully and tolerate missing metrics from older instances.

###### What specific metrics should inform a rollback?

Rollback should be considered if the upgraded kube-apiserver instances show a
clear regression in existing control plane health metrics, for example:

- increased `apiserver_request_total{code=~"5.."}`;
- increased high-percentile `apiserver_request_duration_seconds`;
- increased kube-apiserver CPU or resident memory usage;
- increased Prometheus scrape samples or scrape failures for kube-apiserver; or
- materially increased log volume from the new structured diagnostic logs.

The new `apiserver_watch_cache_watcher_terminated_total` metric is diagnostic.
A non-zero value should not by itself trigger rollback, because it reports an
existing behavior with more detail.

###### Were upgrade and rollback tested? Was the upgrade->downgrade->upgrade path tested?

No manual upgrade or rollback testing has been performed for this KEP draft.
Because the feature has no persisted state, no API changes, and no configuration
changes, targeted unit tests for metric recording and behavior preservation are
expected for Alpha. Upgrade and rollback validation can be covered by normal
kube-apiserver rollout testing when implementation PRs are prepared.

###### Is the rollout accompanied by any deprecations and/or removals of features, APIs, fields of API types, flags, etc.?

No. The existing `apiserver_terminated_watchers_total{group,resource}` metric
continues to be registered and incremented. This KEP does not remove or
deprecate any API, field, flag, or metric in Alpha.

### Monitoring Requirements

###### How can an operator determine if the feature is in use by workloads?

The feature is passive instrumentation, so "in use" means that kube-apiserver is
serving watch cache watches and the relevant code paths are exercised. Operators
can check for the presence of:

```promql
apiserver_watch_cache_watcher_terminated_total
```

and:

```promql
apiserver_watch_cache_init_events_duration_seconds_count
```

The first metric appears only when forced watcher closures occur. The histogram
count appears when cache watchers process initial events.

###### What are the SLIs (Service Level Indicators) an operator can use to determine the health of the service?

The new metrics are diagnostic signals, not standalone service health SLIs.
Operators should continue to use existing kube-apiserver SLIs, including request
latency, request error rate, watch request behavior, and kube-apiserver resource
usage.

The new diagnostic queries are:

Example diagnostic query:

```promql
sum by (group, resource, reason) (
  rate(apiserver_watch_cache_watcher_terminated_total[5m])
)
```

Example initial event latency query:

```promql
histogram_quantile(
  0.99,
  sum by (le, group, resource) (
    rate(apiserver_watch_cache_init_events_duration_seconds_bucket[5m])
  )
)
```

Recommended alerting should avoid firing solely on a small non-zero rate of
watcher terminations. Operators should correlate these diagnostics with
apiserver request latency, watch reconnects, CPU usage, and client behavior.

###### What are the reasonable SLOs (Service Level Objectives) for the above SLIs?

This KEP does not define new SLOs. The change must not regress existing
kube-apiserver SLOs. The new metrics help explain existing watch cache behavior
when those SLOs or operator expectations are not met.

###### Are there any missing metrics that would be useful to have to improve observability of this feature?

Per-client, per-user, per-namespace, per-object, per-request, and watcher
identifier labels would make individual incidents easier to correlate, but they
are intentionally not added because they would introduce unbounded cardinality.
The watcher identifier remains available in structured logs only.

### Dependencies

###### Does this feature depend on any specific services running in the cluster?

No. The feature does not depend on any new in-cluster, cloud-provider, node, or
external service. Operators need their normal metrics and log collection systems
to observe the new diagnostics, but kube-apiserver behavior does not depend on
those systems.

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

The implementation adds metric observations and structured logging to existing
watch cache code paths. It should not add I/O, blocking calls, timers, goroutines,
or additional channel operations to the watch dispatch path beyond the metric
updates and existing log paths.

The forced-close counter is incremented only when a watcher is already being
terminated for unresponsiveness. The initial-event duration histogram is observed
once per cache watcher initial interval. Unit tests should verify that existing
watcher behavior and draining behavior are unchanged.

###### Will enabling / using this feature result in non-negligible increase of resource usage (CPU, RAM, disk, IO, ...) in any components?

The primary scalability cost is additional metric time series in kube-apiserver
and the monitoring backend. For each kube-apiserver instance and each observed
`group,resource` pair:

- `apiserver_watch_cache_watcher_terminated_total` adds up to 3 counter series,
  one per bounded reason;
- `apiserver_watch_cache_init_events_duration_seconds` uses 13 explicit buckets,
  plus the implicit `+Inf` bucket, `_sum`, and `_count`, for 16 histogram series.

The upper bound is therefore approximately 19 new series per observed
`group,resource` pair per kube-apiserver instance. For example, 300 observed
resources across 3 kube-apiserver instances would add up to roughly 17,100 new
series if every resource exercised both metrics. Actual usage depends on the
resources served from watch cache and whether forced watcher closures occur.

The labels do not include object names, namespaces, users, clients, request
URIs, watcher identifiers, or error strings. Clusters with many CRDs can
increase the number of `group,resource` pairs, but cardinality remains bounded by
resource types rather than by objects or requests.

Structured log volume can increase only on the existing forced watcher close and
slow initial event processing paths. The implementation should preserve the
current log verbosity levels so operators can control collection through the
existing kube-apiserver logging configuration.

### Troubleshooting

###### How does this feature react if the API server and/or etcd is unavailable?

The feature does not change kube-apiserver or etcd availability behavior. If a
kube-apiserver instance is unavailable, its metrics cannot be scraped. If etcd
or the watch cache is unhealthy, the new metrics may help diagnose watch cache
symptoms after kube-apiserver is reachable, but they do not provide recovery
behavior.

###### What are other known failure modes?

New metrics are absent after upgrade:

- Detection: `apiserver_watch_cache_watcher_terminated_total` and
  `apiserver_watch_cache_init_events_duration_seconds_count` are absent from an
  upgraded kube-apiserver target.
- Mitigation: verify the kube-apiserver version, scrape configuration, and that
  watch cache metrics are being scraped from the upgraded instance.
- Diagnostics: kube-apiserver startup logs and `/metrics` output from the target
  instance.
- Testing: unit tests should cover metric registration and recording.

Unexpected metric series growth:

- Detection: monitoring backend reports increased scrape samples, increased
  head series, or scrape failures for kube-apiserver targets.
- Mitigation: verify that only `group`, `resource`, and bounded `reason` labels
  are present. Roll back the kube-apiserver binary if the implementation adds
  unintended high-cardinality labels.
- Diagnostics: inspect the exported metric labels and compare observed
  `group,resource` pairs with the resources served by the apiserver.
- Testing: unit tests should verify that metric labels do not include unbounded
  values.

Unexpected log volume:

- Detection: log backend volume increases on upgraded kube-apiserver instances,
  especially for forced watcher close or slow initial event processing messages.
- Mitigation: preserve the existing log verbosity levels, reduce collected log
  verbosity if appropriate, or roll back the kube-apiserver binary if the
  implementation logs more frequently than intended.
- Diagnostics: inspect structured log fields `group`, `resource`, `reason`,
  `initEventCount`, and `duration`.
- Testing: unit tests can verify log call paths where practical; behavior is
  primarily guarded by keeping logs on existing diagnostic paths.

If `input_buffer_full` dominates, the watcher input channel is filling before
the processing goroutine can drain it.

If `result_buffer_full` dominates, the watcher result channel is full at close
time, indicating pressure between the cache watcher processing goroutine and the
watch result consumer.

If `bookmark_pending` appears, the watcher was closed while trying to preserve
delivery of the required bookmark. This points to result delivery pressure after
the watch cache had already observed the bookmark resource version.

###### What steps should be taken if SLOs are not being met to determine the problem?

Operators should first use existing kube-apiserver SLO and health metrics to
confirm whether the problem is API latency, errors, resource pressure, or watch
reconnect behavior. They can then use the new reasoned termination counter to
identify affected resources and server-side forced-close conditions, and use the
initial event duration histogram to check whether watch-list initialization is
contributing to the issue. Structured logs can be used to correlate individual
forced-close events with watcher key and selector information without adding
those unbounded values to metrics.

## Drawbacks

The new detailed counter duplicates the existing watcher termination count. This
is intentional for compatibility, but it means operators may need to learn which
metric to use for detailed diagnosis.

The reason labels describe server-side observations, not guaranteed root causes.
Operators may still need logs, traces, or client-side investigation to identify
the specific slow client.

## Alternatives

### Add a `reason` label to the existing metric

The existing metric is ALPHA, so changing its labels may be technically
possible. However, adding a label changes the Prometheus time series shape and
can break existing dashboards or alerts. This KEP proposes a new metric instead
and keeps the existing metric unchanged.

### Only improve logs

Structured logs are useful for individual incidents, but metrics are needed for
fleet-wide alerting, trend analysis, and correlation with apiserver load. Logs
may be sampled, rotated, or unavailable in centralized systems.

### Add watcher identifiers as metric labels

This would make it easier to connect a metric to a request, but it would create
unbounded cardinality. Watcher identifiers remain log fields only.

### Change buffer sizes or termination behavior

Changing behavior may reduce terminations in some cases, but it is outside the
scope of this diagnostic KEP. The purpose of this proposal is to understand the
existing behavior before changing policy.

## Implementation History

- 2026-05-01: Initial draft.
