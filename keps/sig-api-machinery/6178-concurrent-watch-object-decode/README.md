# KEP-6178: Concurrent Watch Object Decode

<!-- toc -->
- [Release Signoff Checklist](#release-signoff-checklist)
- [Summary](#summary)
- [Motivation](#motivation)
  - [Goals](#goals)
  - [Non-Goals](#non-goals)
- [Proposal](#proposal)
  - [Risks and Mitigations](#risks-and-mitigations)
- [Design Details](#design-details)
  - [Choosing the concurrency level](#choosing-the-concurrency-level)
  - [Test Plan](#test-plan)
      - [Prerequisite testing updates](#prerequisite-testing-updates)
      - [Unit tests](#unit-tests)
      - [Integration tests](#integration-tests)
      - [e2e tests](#e2e-tests)
  - [Graduation Criteria](#graduation-criteria)
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
  - [Scale the pool with GOMAXPROCS](#scale-the-pool-with-gomaxprocs)
  - [Make the pool size configurable](#make-the-pool-size-configurable)
<!-- /toc -->

## Release Signoff Checklist

Items marked with (R) are required *prior to targeting to a milestone / release*.

- [X] (R) Enhancement issue in release milestone, which links to KEP dir in [kubernetes/enhancements] (not the initial KEP PR)
- [ ] (R) KEP approvers have approved the KEP status as `implementable`
- [X] (R) Design details are appropriately documented
- [X] (R) Test plan is in place, giving consideration to SIG Architecture and SIG Testing input (including test refactors)
- [X] (R) Graduation criteria is in place
- [ ] (R) Production readiness review completed
- [ ] (R) Production readiness review approved
- [X] "Implementation History" section is up-to-date for milestone
- [ ] User-facing documentation has been created in [kubernetes/website], for publication to [kubernetes.io]
- [ ] Supporting documentation—e.g., additional design documents, links to mailing list discussions/SIG meetings, relevant PRs/issues, release notes

[kubernetes.io]: https://kubernetes.io/
[kubernetes/enhancements]: https://git.k8s.io/enhancements
[kubernetes/kubernetes]: https://git.k8s.io/kubernetes
[kubernetes/website]: https://git.k8s.io/website

## Summary

The kube-apiserver decodes and transforms watch events from etcd serially, so one
slow per-event transform (notably a CRD conversion webhook call) blocks all later
events on that watch and can stall watch cache initialization past the etcd
compaction interval, leaving the cache unable to converge.
`ConcurrentWatchObjectDecode` runs the transforms across a bounded goroutine pool
while preserving delivery order. The gate has been Beta
(off by default) since v1.31. This KEP tracks graduating it to Beta on by default
in v1.37.

## Motivation

Stored objects are decoded and transformed before they enter the watch cache, and
today this runs serially per watch event, so cache initialization for a
high-cardinality resource takes time proportional to the object count times the
per-object cost. Parallelizing it helps two cases:

- General speedup for built-in resources. The per-object cost is base decode plus
  any storage transform (for example protobuf decode and at-rest decryption), with
  no webhook involved. Sweeping the gate in `BenchmarkCacherInit` over 150k pods
  shows concurrent decode alone cuts cache initialization about 40 percent, and
  about 55 percent combined with `EtcdRangeStream`
  ([kubernetes/kubernetes#139619](https://github.com/kubernetes/kubernetes/pull/139619)).
- Worst case for CRDs. When the per-object cost is a conversion webhook round trip
  (a CRD whose served version differs from its stored version), serially converting
  a cold cache for a large resource can take minutes (about 8 minutes was estimated
  in [kubernetes/kubernetes#136950](https://github.com/kubernetes/kubernetes/issues/136950)).
  If that exceeds the etcd compaction interval (default 5 minutes), the revision the
  cache started from is compacted before initialization finishes, so the watch
  cannot resume from it and init restarts, never converging for a large enough
  resource. The cache stays uninitialized and every client listing or watching the
  resource gets errors. Concurrent decode keeps that conversion inside the
  compaction window, which is what lets large CRDs serve a non-storage version and
  migrate storage versions safely at scale.

### Goals

- Decode and transform watch events concurrently.
- Preserve watch event ordering and delivery semantics exactly.

### Non-Goals

N/A

## Proposal

Flip the existing `ConcurrentWatchObjectDecode` gate to default-on in v1.37. The
implementation has shipped behind the gate since v1.31.

### Risks and Mitigations

The change is internal to the apiserver, has no persisted state, and produces the
same ordered event stream as serial decoding, so toggling the default or running
mixed versions across apiservers does not affect correctness.

The one behavioral change worth noting is conversion webhook load. With the feature
off, the watch decode path converts objects one at a time, so the conversion
webhook gets one call at a time. With it on, up to 10 conversions run concurrently
(the `processEventConcurrency` limit), per cacher per apiserver and only during
cache initialization. The total number of calls is unchanged, only how many run at
once. Two dimensions are worth checking, request concurrency and memory:

- Request handling. A webhook that limits or rejects concurrent requests (for
  example returning 503 above a fixed in-flight count) can drop calls under the
  burst. Most webhook servers do not limit concurrency, so this only affects
  webhooks that set their own limit below 10.
- Memory. Cache initialization sends one object per call, so the webhook holds at
  most 10 objects in flight regardless of how many are stored. The list path
  already sends far more in a single ConversionReview, carrying a whole list page
  (hundreds of objects by default, or the entire collection for an unpaginated
  list), so the feature adds no new memory or OOM risk.

The increase is small (up to 10), happens only during initialization, and does not
change total call volume. Webhooks already field concurrent calls from normal
traffic, so this is within their usual range.

## Design Details

When enabled, the etcd watch event processing path decodes and transforms events
across a bounded pool of worker goroutines instead of a single goroutine. A
collector reassembles the transformed events in their original order before
delivery, so event ordering is preserved. Concurrency is capped by a fixed
internal bound, and a bounded queue applies backpressure so a fast event stream
cannot grow unbounded goroutines or memory.

### Choosing the concurrency level

10 is a tuning value for the decode pool. A pool-size sweep showed init speedup
flattening out around 8 to 12 with no gain above that, so 10 is a reasonable
default. The exact value will be refined through scalability testing.

### Test Plan

[X] I/we understand the owners of the involved components may require updates to
existing tests to make this code solid enough prior to committing the changes
necessary to implement this enhancement.

##### Prerequisite testing updates

None.

##### Unit tests

`k8s.io/apiserver/pkg/storage/etcd3` (`watcher_test.go`) runs the watch suite with
the gate on and off and asserts equivalent ordered output.

##### Integration tests

Existing watch and storage integration tests cover this path.

##### e2e tests

None required.

### Graduation Criteria

- Beta (v1.31): gate introduced, disabled by default.
- Beta on by default (v1.37): default flipped to enabled.
- GA: TBD in a later release.

### Upgrade / Downgrade Strategy

Controlled solely by the feature gate with no persisted state. Enable or disable
by setting the gate and restarting the kube-apiserver.

### Version Skew Strategy

Internal to the kube-apiserver, no cross-component coordination. Mixed enablement
across apiservers is safe.

## Production Readiness Review Questionnaire

### Feature Enablement and Rollback

###### How can this feature be enabled / disabled in a live cluster?

- [X] Feature gate (also fill in values in `kep.yaml`)
  - Feature gate name: `ConcurrentWatchObjectDecode`
  - Components depending on the feature gate: kube-apiserver

###### Does enabling the feature change any default behavior?

Watch events are decoded concurrently instead of serially, with the same ordering
and content.

###### Can the feature be disabled once it has been enabled (i.e. can we roll back the enablement)?

Yes, set the gate to false and restart the kube-apiserver. No persisted state.

###### What happens if we reenable the feature if it was previously rolled back?

Same as first enablement, no state to reconcile.

###### Are there any tests for feature enablement/disablement?

Yes, unit tests run the watch suite with the gate on and off.

### Rollout, Upgrade and Rollback Planning

###### How can a rollout or rollback fail? Can it impact already running workloads?

No. There is no persisted state and both paths produce the identical event stream,
so toggling the gate or mixed-version apiservers do not affect running workloads.

###### What specific metrics should inform a rollback?

Rising `apiserver_watch_cache_initialization_errors_total` or increased WATCH
`apiserver_request_duration_seconds`.

###### Were upgrade and rollback tested? Was the upgrade->downgrade->upgrade path tested?

Covered by unit tests that toggle the gate. With no persisted state the path is
equivalent to enabling and disabling the gate.

###### Is the rollout accompanied by any deprecations and/or removals of features, APIs, fields of API types, flags, etc.?

No.

### Monitoring Requirements

###### How can an operator determine if the feature is in use by workloads?

It applies to all watches once enabled. Confirm via the apiserver feature gate
configuration.

###### How can someone using this feature know that it is working for their instance?

No feature-specific metric. Watch cache initialization for a large or
conversion-heavy resource is faster with the gate on, seen as a shorter WATCH
`apiserver_request_duration_seconds` during init.

###### What are the reasonable SLOs (Service Level Objectives) for the enhancement?

No regression in existing list and watch latency SLOs.

###### What are the SLIs (Service Level Indicators) an operator can use to determine the health of the service?

- [X] Metrics
  - Metric name: `apiserver_watch_cache_initializations_total`,
    `apiserver_watch_cache_initialization_errors_total`,
    `apiserver_request_duration_seconds`
  - Components exposing the metric: kube-apiserver

###### Are there any missing metrics that would be useful to improve observability of this feature?

`apiserver_watch_decode_inflight`, a gauge of watch objects currently being
decoded concurrently, to track how much of the per-stream decode budget is in use.

### Dependencies

###### Does this feature depend on any specific services running in the cluster?

No.

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

No, it reduces watch cache initialization and watch event processing time.

###### Will enabling / using this feature result in non-negligible increase of resource usage (CPU, RAM, disk, IO, ...) in any components?

Up to `processEventConcurrency` (10) transform goroutines per watch plus a bounded
queue. Total work is unchanged.

###### Can enabling / using this feature result in resource exhaustion of some node resources (PIDs, sockets, inodes, etc.)?

No, concurrency and queue depth are bounded per watch.

### Troubleshooting

###### How does this feature react if the API server and/or etcd is unavailable?

No change. The feature only affects how received events are decoded.

###### What are other known failure modes?

A conversion webhook unable to handle the increased concurrency may error or
throttle, surfacing as watch cache initialization errors for that resource
(`apiserver_watch_cache_initialization_errors_total`).

###### What steps should be taken if SLOs are not being met to determine the problem?

Correlate with the feature being enabled and check conversion webhook health and
watch cache initialization metrics for the affected resource.

## Implementation History

- v1.31: gate introduced as Beta, disabled by default, with the concurrent
  implementation.
- v1.37: KEP created, gate proposed to default-on.

## Drawbacks

Adds bounded concurrency to a previously serial path, making ordering an invariant
that must be maintained.

## Alternatives

The worker pool is a fixed size (`processEventConcurrency = 10`). Two other ways to
pick it were considered.

### Scale the pool with GOMAXPROCS

Size the pool to the apiserver's CPU count instead of a constant.

Pros:

- Scales automatically with the machine, more parallelism on bigger apiservers with
  no tuning.
- Follows the common Go pattern of sizing worker pools to GOMAXPROCS.

Cons:

- The conversion-webhook burst would scale with CPU count (32, 64, and up), which is
  much harder to bound and document than a flat 10.
- Conversion is usually bound by the webhook round trip, not apiserver CPU, so
  GOMAXPROCS is a poor proxy for the right concurrency.
- Behavior varies by apiserver size, which is harder to reason about and test.

### Make the pool size configurable

Expose the pool size as an apiserver flag.

Pros:

- Operators can lower it for a capacity-limited webhook or raise it for a large CRD
  whose webhook can take it.
- A tuning option short of disabling the feature.

Cons:

- Another apiserver flag most operators will never set and cannot easily reason
  about, since the right value depends on webhook capacity, object size, and CPU.
- More API surface to support and deprecate, especially if the pool later becomes
  self-tuning.
- A sensible default still has to be chosen, so it does not remove the core decision.

A fixed 10 keeps the added webhook concurrency predictable and easy to document, and
is the simplest to test. It can be revisited if it proves too rigid.
