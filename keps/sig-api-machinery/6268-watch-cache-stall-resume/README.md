# KEP-6268: Stall and resume slow watch streams instead of terminating them

<!-- toc -->
- [Release Signoff Checklist](#release-signoff-checklist)
- [Summary](#summary)
- [Motivation](#motivation)
  - [Goals](#goals)
  - [Non-Goals](#non-goals)
- [Proposal](#proposal)
  - [User Stories](#user-stories)
  - [Notes/Constraints/Caveats](#notesconstraintscaveats)
  - [Risks and Mitigations](#risks-and-mitigations)
- [Design Details](#design-details)
  - [Test Plan](#test-plan)
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
<!-- /toc -->

## Release Signoff Checklist

- [ ] (R) Enhancement issue in release milestone, which links to KEP dir in [kubernetes/enhancements]
- [ ] (R) KEP approvers have approved the KEP status as `implementable`
- [ ] (R) Design details are appropriately documented
- [ ] (R) Test plan is in place, giving consideration to SIG Architecture and SIG Testing input
- [ ] (R) Graduation criteria is in place
- [ ] (R) Production readiness review completed
- [ ] (R) Production readiness review approved
- [ ] "Implementation History" section is up-to-date
- [ ] User-facing documentation created
- [ ] Supporting e2e tests added, or a description of why they are not needed

## Summary

When a watch client cannot keep up and its per-watcher delivery buffer fills, the
apiserver blocks the shared watch-dispatch path for up to a small shared time
budget and, if the buffer is still full, force-closes the watch as
"unresponsive". This has two costs at scale: the dispatcher's wait delays
delivery to every watcher of that resource, and a watcher that is merely *behind*
(for example, one replaying its history right after a reconnect) is guaranteed to
fill its buffer and be terminated — which triggers an immediate reconnect with a
larger gap and repeats.

This KEP proposes an alpha, opt-in `WatchCacheStallResume` feature gate for
kube-apiserver. With it enabled, a watch whose buffer fills is no longer
terminated: the apiserver records the resource version of the first event that
did not fit and parks the watch off the hot dispatch path; when the buffer
drains, the watch's own goroutine delivers what it already holds, re-reads the
events it missed from the in-memory watch cache history starting just below that
missed event, and rejoins the live stream. A watch is terminated only when that
missed event itself has aged out of the history, which yields the same `410 Gone`
an etcd watch receives on compaction. Slow or stalled watch clients then impose no delivery cost on healthy
watchers and are given a full history window of grace to recover.

## Motivation

The watch cache dispatches each event to every registered watcher's buffered
channel. A watcher's channel can fill for two very different reasons that the
current code cannot distinguish:

1. **A genuinely slow or dead client** — the consumer is not reading (a stalled
   TCP connection, a wedged or overloaded client, a client behind a slow WAN).
2. **A watcher that is temporarily behind but healthy** — most importantly, a
   watcher that has just (re)connected and must replay its missed history before
   its consumer loop starts draining new events. During that replay its buffer
   is expected to fill.

Today both lead to the same outcome: after the shared dispatch budget is spent,
the watcher is force-closed. For case (2) this is actively harmful. A re-attached
watcher is force-closed mid-replay, reconnects immediately (clients re-establish
watches without backoff), now has a larger gap to replay, fills its buffer
sooner, and is force-closed again — a self-reinforcing loop. At high write rates
this does not converge; the only exits are the write rate dropping or the
watcher's position aging out of the ring, which forces a full re-list that adds
yet more load. Because the dispatcher's per-round wait is shared, a single stuck
client also slows delivery to every healthy watcher of the same resource.

These are not hypothetical. Operators of large clusters with many watch clients
(e.g. per-node agents, or read-through caching proxies fronting the apiserver)
observe both the "one slow client taxes everyone" delay and the
reconnect/replay collapse during events that transiently disconnect many
watchers at once (node churn, zone-connectivity blips).

### Goals

- A watcher that is behind but whose client is still making progress is not
  terminated for a full input buffer; it catches up from the watch cache history
  and rejoins the live stream on the same connection.
- A slow or stalled watch client imposes no delivery delay or forced-close on
  other, healthy watchers of the same resource.
- Preserve watch delivery semantics exactly: every event is delivered to each
  watcher exactly once and in resource-version order across the stall/resume
  transition, and bookmark semantics are preserved.
- The only remaining termination for "cannot keep up" is honest: a `410 Gone`
  when the client's resume position has aged out of the watch cache history,
  matching existing etcd-compaction behavior.
- Off by default and fully reversible via the feature gate, with no API,
  storage, or wire-format changes.

### Non-Goals

- Changing how the *initial* hydration of a watch (a `sendInitialEvents`
  WatchList, or a from-store list) is produced or paginated. This KEP concerns
  the incremental-delivery path only.
- Reducing the cost of a full re-list once a client has genuinely aged out of
  the history.
- Any change to client-go, informer, or reflector behavior. This is an
  apiserver-only change and requires no client changes.
- Guaranteeing a bounded watch delay for a client that is permanently slower
  than the write rate; such a client still eventually receives a `410`.

## Proposal

Behind the `WatchCacheStallResume` gate, the watch-cache dispatch and per-watcher
processing loops are replaced by a stall-and-resume design:

- **Dispatch never blocks and never evicts.** When an event cannot be enqueued
  into a watcher's input channel without blocking, the dispatcher latches that
  event's resource version on the watcher (a single-slot latch; an O(1),
  non-blocking operation; later misses of the same episode coalesce onto the
  pending value, which is below them) and moves on. It never waits on the shared budget for that watcher and never
  force-closes it for a full buffer.
- **The watcher catches up from history.** The watcher's own goroutine, when it
  observes the latch (it checks it before delivering anything it receives, and
  also wakes on it while its input is quiet), reads the events it missed from
  the in-memory watch cache event history — the same ring buffer already used to serve `resourceVersion`
  resumes. It first delivers, in order, the events already queued in its
  channels below the missed one, then raises its resume position to just below
  the missed resource version and streams the history from there to the client,
  then rejoins the live stream. Because the resume position is the miss itself
  rather than the watcher's last delivery, a scope-filtered watcher (field,
  label, name or trigger-index scoped) that sees little of the collection's
  traffic stays resumable regardless of unrelated churn. A resume-position
  filter guarantees exactly-once, in-order delivery across the transition.
- **Termination is honest and rare.** If the missed event itself has fallen out
  of the history window by the time the watcher catches up (or the history moves
  past a catch-up round while it is being streamed), the watcher delivers one
  in-stream `410 Gone` (`Expired`) error event — exactly what a direct etcd watch
  receives on compaction — and the stream ends. The client re-establishes and
  re-lists, as it does today for compaction.
- **Optional cull for truly-dead clients.** An off-by-default
  `StalledClientCullAfter` duration lets a once-per-second sampler stop a watcher
  whose result buffer is full *and* to whose client nothing has been delivered
  for that long — reclaiming resources from clients that have stopped reading
  entirely (e.g. a half-open TCP connection). A part-filled buffer on a quiet
  resource, or a stall being served by the watcher's own goroutine, is not
  evidence about the client and is not counted. With it disabled, the request deadline
  remains the backstop, as today.

### User Stories

**Read-through caching proxy fleet.** An operator runs many caching proxies that
each hold a watch on a large collection. A network blip disconnects a subset;
they reconnect and replay. Today the replaying proxies are force-closed, retry
with larger gaps, and can collapse the whole fleet; the healthy proxies are also
slowed. With the gate on, the reconnecting proxies catch up from history on the
same connection while the healthy ones are unaffected.

**Per-node agents over a WAN.** Some node agents' watch streams stall at the TCP
layer. Today each stalled stream taxes the shared dispatch path and, when it
falls off the history, forces a re-list that adds load. With the gate on, a
stalled agent is parked and imposes no cost on others; if it never recovers it
receives an honest `410`.

### Notes/Constraints/Caveats

- The unit of tolerance shifts from "how many events fit in a fixed-size channel"
  (a count) to "how far back the watch cache history reaches" (a time window,
  `eventFreshDuration`). The history window, not the channel size, now bounds how
  long a client may lag before it must re-list.
- Correctness across the stall/resume boundary depends on three invariants,
  which are enforced and documented at their call sites: (1) every event whose
  enqueue failed is served by a catch-up round before any event at or above its
  resource version is delivered (the queued backlog below it is delivered
  first); (2) object events are delivered strictly once and in increasing
  resource-version order, filtered by the resume position; (3) bookmark handling
  is the gate-off rule generalized from the fixed starting position to the
  moving resume position: bookmarks above the position are delivered and advance
  it, a bookmark at the position is delivered unless it would repeat one for the
  starting position, and a bookmark below the position (leapt over by a catch-up
  round) is dropped so a client's last-seen resource version never moves
  backwards.

### Risks and Mitigations

- **A delivery-correctness bug (missed or duplicated event).** This is the
  primary risk of any change to the delivery path. Mitigations: the gate is off
  by default; the exactly-once and ordering invariants are covered by unit tests,
  a white-box test that forces stalls and reconnects, and an integration test
  that overflows a real watch and asserts every event arrives in order after the
  client resumes; and the honest `410` fallback means the worst case for an
  un-handled edge is a re-list, not silent data loss.
- **A watcher that never recovers holds resources.** A parked watcher retains its
  channels and goroutine. Mitigations: the request deadline bounds its lifetime
  as today; `StalledClientCullAfter` can reclaim confirmed-dead clients sooner;
  the `apiserver_watch_cache_stalled_watchers` gauge makes the population
  visible.
- **Metric label change.** The existing `apiserver_terminated_watchers_total`
  gains a `reason` label unconditionally (independent of the gate); queries
  selecting the exact label set must use `sum by (group, resource)` to keep their
  previous shape. Documented in the release note.

## Design Details

The change is contained within
`staging/src/k8s.io/apiserver/pkg/storage/cacher`. When the gate is enabled, each
`cacheWatcher` gains stall state (a single-slot latch holding the first missed
resource version, the starting position, and a last-progress timestamp) and a
read-only view over the existing watch cache history; the `Cacher` gains a
stalled-watchers sampler. Dispatch, on a failed non-blocking enqueue, latches the
event's resource version instead of waiting/closing. The watcher's processing
loop is replaced by a variant that (a) when the latch is set, drains the queued
events below the miss, raises the resume position to just below the miss and
serves a catch-up interval from the history, filtering by resume position for
exactly-once delivery, and (b) otherwise delivers live events, dropping any whose
resource version is at or below the resume position. When the gate is disabled,
the dispatch path and the watcher processing loop are the previous behavior (the
only gate-independent changes are the `reason` metric label below and the
formatting of one pre-existing warning log line).

One consequence reaches the initial phase of a watch: with the gate on, a
watcher whose *initial* interval (the history segment or store snapshot it is
hydrated from) is overtaken by the history while it is still being streamed
receives the same in-stream `410` (counted with reason
`resource_expired_initial`) instead of today's silent close; how the initial
state is produced is otherwise unchanged.

New observability:

- `apiserver_watch_cache_watcher_stalls_total` (counter) — stall episodes begun.
- `apiserver_watch_cache_watcher_deferred_events_total` (counter) — events
  deferred by a stall rather than delivered inline.
- `apiserver_watch_cache_watcher_catchup_rounds_total` (counter) — catch-up
  rounds served from history.
- `apiserver_watch_cache_watcher_catchup_events` (histogram) — events per
  catch-up round.
- `apiserver_watch_cache_stalled_watchers` (gauge) — watchers whose result
  buffer is full and that have made no delivery progress within the last
  bookmark period.
- `apiserver_terminated_watchers_total` gains a `reason` label:
  `unresponsive` (today's force-close, gate off), `resource_expired` (gate on:
  a live watcher's missed event or catch-up round aged out of the history),
  `resource_expired_initial` (gate on: the same, before the watcher ever
  reached the live stream, i.e. during or right after its initial hydration),
  `stalled_client` (gate on: stopped by the optional cull).

### Test Plan

[ ] I understand the owning sig for the component(s) is responsible for
maintaining the tests owned by that sig.

##### Prerequisite testing updates

None.

##### Unit tests

- `staging/src/k8s.io/apiserver/pkg/storage/cacher`: existing coverage plus new
  cases for the stall latch and coalescing, catch-up rounds (including a round
  invalidated mid-stream), resume-position filtering with a hand-scripted input,
  scope-filtered and trigger-indexed watchers resuming after unrelated churn
  with and without bookmarks, bookmark parity with the gate off at the starting
  position, initial-list and WatchList watchers stalling during hydration, both
  aged-out `410` reasons, clean closes on context cancellation and Cacher
  shutdown, the sampler and cull driven by a fake clock, a randomized ordering
  test and a multi-watcher chaos oracle; several pre-existing watch cache tests
  now run with the gate both on and off (the gate-off path must match prior
  behavior).

##### Integration tests

- `test/integration/apiserver`: a real kube-apiserver watch whose client stops
  reading while more data than the HTTP/2 stream window plus the watcher buffers
  can hold is written; with the gate on, every event is delivered in order once
  the client resumes and the watch stays open; with the gate off (control), the
  server terminates the watch during the stall.

##### e2e tests

None planned for alpha; the behavior is exercised by the integration test above.

### Graduation Criteria

#### Alpha

- Feature implemented behind the `WatchCacheStallResume` gate, off by default.
- Unit and integration tests as above, passing with the gate on and off.
- Metrics listed above emitted.

#### Beta

- Gathered feedback / soak from clusters running with the gate enabled.
- No unresolved delivery-correctness issues.
- Scalability sign-off that the parked-watcher path does not regress dispatch or
  memory under realistic slow-client populations.
- Decision on whether `StalledClientCullAfter` should have a non-zero default.

#### GA

- The feature has been enabled by default (beta) across at least two releases
  without correctness or scalability regressions.
- e2e coverage as required by SIG API Machinery / SIG Testing.

### Upgrade / Downgrade Strategy

The feature is a single-component (kube-apiserver) behavioral change with no API,
storage, or wire-format impact. Enabling it requires only turning on the gate;
disabling it reverts to the current force-close behavior. No coordinated
upgrade/downgrade of other components is required, and no data is written that a
downgraded apiserver could not read.

### Version Skew Strategy

None. The change is local to a single kube-apiserver process and is invisible to
clients (a caught-up watch and a re-established watch are both valid watch
behaviors clients already handle), to etcd, and to other control-plane
components. In an HA apiserver set, some apiservers may have the gate on and some
off simultaneously; each behaves correctly and independently, and a watch served
by either is spec-compliant.

## Production Readiness Review Questionnaire

### Feature Enablement and Rollback

###### How can this feature be enabled / disabled in a live cluster?

- [x] Feature gate
  - Feature gate name: `WatchCacheStallResume`
  - Components depending on the feature gate: `kube-apiserver`

###### Does enabling the feature change any default behavior?

Yes, for watch delivery under load: a watch whose delivery buffer fills is parked
and resumed from history instead of being force-closed, and a slow watcher no
longer delays delivery to other watchers. No API responses change shape; a
resumed watch delivers the same events it would have, and terminations still
surface as `410 Gone`. The `reason` label added to
`apiserver_terminated_watchers_total` is the one gate-independent change.

###### Can the feature be disabled once it has been enabled?

Yes. Turning the gate off returns kube-apiserver to the current behavior. The
gate is read when each resource's watch cache is constructed, i.e. at
kube-apiserver start, so the change takes effect with the (rolling) restart that
flips the flag; there is no persisted state to migrate.

###### What happens if we reenable the feature if it was previously rolled back?

Nothing special; behavior is determined solely by the current gate value at watch
establishment. No state persists across the toggle.

###### Are there any tests for feature enablement/disablement?

Yes. Unit and integration tests run with the gate both enabled and disabled; the
gate-off runs assert the previous behavior (including that an overflowing watch is
terminated).

### Rollout, Upgrade and Rollback Planning

###### How can a rollout or rollback fail? Can it impact already running workloads?

The feature only affects how kube-apiserver services watch streams; it does not
touch workloads directly. A rollout failure would manifest as a delivery-
correctness bug (a missed or duplicated watch event) or as parked watchers not
being reclaimed. A missed/duplicated event could cause controllers to act on
stale state; this is mitigated by the off-by-default gate, the delivery
invariants and their tests, and the `410` fallback. Rolling the gate back off
immediately restores prior behavior.

###### What specific metrics should inform a rollback?

- A rise in watch-related client errors or controller staleness after enabling.
- `apiserver_watch_cache_stalled_watchers` growing without bound.
- `apiserver_terminated_watchers_total{reason="resource_expired"}` far exceeding
  the pre-enablement `reason="unresponsive"` rate (clients aging out rather than
  catching up), which would indicate the history window is too small for the
  workload.

###### Were upgrade and rollback tested? Was the upgrade->downgrade->upgrade path tested?

To be completed before beta; for alpha the gate on/off tests exercise both
behaviors within one binary.

###### Is the rollout accompanied by any deprecations and/or removals?

No.

### Monitoring Requirements

###### How can an operator determine if the feature is in use by workloads?

The `apiserver_watch_cache_watcher_stalls_total` and
`apiserver_watch_cache_watcher_catchup_rounds_total` counters are non-zero when
watches are being parked and resumed, and `apiserver_watch_cache_stalled_watchers`
shows the current stalled population.

###### How can someone using this feature know that it is working for their instance?

- [x] Metrics
  - Metric names: the `apiserver_watch_cache_watcher_*` series above, the
    `apiserver_watch_cache_stalled_watchers` gauge, and the `reason` label on
    `apiserver_terminated_watchers_total` (a shift from `unresponsive` toward
    zero force-closes-under-load indicates the feature is doing its job).

###### What are the reasonable SLOs for the enhancement?

No new SLO is proposed. The enhancement is expected to *improve* existing watch
availability/latency during slow-client events rather than introduce a new
objective.

###### What are the SLIs an operator can use to determine the health of the feature?

- [x] Metrics
  - `apiserver_watch_cache_stalled_watchers` (should be small and drain, not grow
    unbounded).
  - `apiserver_terminated_watchers_total{reason="resource_expired"}` (rare; a
    spike means clients cannot catch up within the history window).
  - `apiserver_watch_cache_watcher_catchup_events` distribution (bounded per
    round).

###### Are there any missing metrics that would be useful to have?

Not identified for alpha.

### Dependencies

###### Does this feature depend on any specific services running in the cluster?

No. It uses only the existing in-memory watch cache within kube-apiserver.

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

It is expected to *reduce* watch-dispatch delay under slow-client load (a slow
watcher no longer stalls the shared dispatch path). It adds a single per-second
sampler goroutine per resource-cacher and a single-slot latch per watcher, both
negligible.

###### Will enabling / using this feature result in non-negligible increase of resource usage in any components?

A parked watcher retains the same channels/goroutine it holds today; the feature
does not create additional per-watcher buffers. The main change is that a slow
watcher may live longer (until it catches up, ages out, or is culled) than it
would if force-closed — bounded by the request deadline and, optionally,
`StalledClientCullAfter`. CPU/memory of the catch-up path is drawn from the
existing history buffer, not new allocations proportional to state size.

###### Can enabling / using this feature result in resource exhaustion of some node resource?

Not on nodes. On kube-apiserver, a pathological population of never-recovering,
never-culled slow clients could retain more watcher goroutines/channels than the
force-close behavior would. This is bounded by request deadlines, observable via
`apiserver_watch_cache_stalled_watchers`, and reclaimable via
`StalledClientCullAfter`.

### Troubleshooting

###### How does this feature react if the API server and/or etcd is unavailable?

The feature lives entirely in the apiserver's in-memory watch cache and does not
add etcd interactions. If etcd is unavailable, watch behavior is governed by the
existing watch-cache logic; this feature does not change that.

###### What are other known failure modes?

- Clients repeatedly aging out (`reason="resource_expired"`) — the history window
  is too small for the workload's write rate; increase the window or investigate
  the slow clients. Diagnose via the terminated-watchers `reason` label and the
  stalled-watchers gauge.
- Growing `apiserver_watch_cache_stalled_watchers` — a population of clients that
  stall and never recover; consider enabling `StalledClientCullAfter`.

###### What steps should be taken if SLOs are not being met to determine the problem?

Disable the `WatchCacheStallResume` gate to return to the prior behavior, then
inspect the metrics above to characterize the slow-client population.

## Implementation History

- 2026-08-06: KEP drafted; alpha implementation prototyped and validated on a
  scalability rig against a single apiserver + caching-proxy fleet.

## Drawbacks

It adds state and a second delivery path to the watch cache, a component whose
correctness is critical. The complexity is justified by removing a
force-close/reconnect collapse that has no in-place mitigation today, and is
contained behind an off-by-default gate whose disabled path is unchanged.

## Alternatives

- **Do nothing / rely on larger buffers.** Increasing the per-watcher channel
  size raises the count of events a slow watcher can absorb but does not change
  the "buffer full ⇒ force-close, shared wait taxes everyone" structure; it only
  moves the threshold.
- **Only cull zero-progress clients.** Culling clients that read nothing (the
  optional `StalledClientCullAfter` here) addresses dead connections but not the
  reconnect/replay collapse of *healthy-but-behind* watchers, which is the main
  motivation.
- **A hard cap on catch-up work per watcher.** Bounding the number of catch-up
  rounds and terminating past it was considered; it re-introduces a force-close
  for slow-but-live clients and is unnecessary because aging out of the history
  window already provides an honest, time-denominated bound. Kept as a possible
  future safeguard, not required for the core behavior.
