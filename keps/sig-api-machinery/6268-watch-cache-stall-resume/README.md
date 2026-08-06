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

The watch cache delivers every event to each watcher through a bounded input
channel. When a client cannot keep up and its channel fills, kube-apiserver
today blocks the shared dispatch path for a small time budget. If the channel is
still full after that, it force-closes the watch as "unresponsive". This has
two costs at scale. The dispatcher's wait delays delivery to every watcher of
that resource. And a watcher that is merely behind (for example, one that just
reconnected and is replaying its history) is guaranteed to fill its channel and
be terminated. The termination triggers an immediate reconnect with a larger
gap, and the loop repeats.

This KEP proposes the `WatchCacheStallResume` feature gate for kube-apiserver.
With the gate on, the watch cache adopts the model etcd uses for its own
watchers. A watcher whose input channel is full becomes "unsynced": the
dispatcher stops offering it events and records the position from which it must
resume. The dispatcher then serves all unsynced watchers from the in-memory
watch cache history in short, bounded sync passes. Watchers that need the same
stretch of history share one history read and one copy of each event, so an
event is serialized once and reused across watchers. A watcher rejoins the live
stream on the same connection once it has accepted every event up to the end of
the history. A watcher is terminated only when its resume position has aged out
of the history. It then receives the same in-stream `410 Gone` an etcd watch
receives on compaction. Slow or stalled clients impose no delivery cost on
healthy watchers, and a healthy but behind client gets a full history window of
grace to recover.

"Alpha" and "opt-in" describe rollout stages, not the end state. The gate
exists so the new path can be proven in production behind a switch. Once it is
proven, the new behavior becomes the behavior of the watch cache and the old
force-close path is removed.

## Motivation

The watch cache dispatches each event to every registered watcher's buffered
channel. A watcher's channel can fill for two very different reasons that the
current code cannot distinguish:

1. **A genuinely slow or dead client.** The consumer is not reading (a stalled
   TCP connection, a wedged or overloaded client, a client behind a slow WAN).
2. **A watcher that is temporarily behind but healthy.** Most importantly, a
   watcher that has just (re)connected and must replay its missed history before
   its consumer loop starts draining new events. During that replay its buffer
   is expected to fill.

Today both lead to the same outcome: after the shared dispatch budget is spent,
the watcher is force-closed. For case (2) this is actively harmful. A re-attached
watcher is force-closed mid-replay, reconnects immediately (clients re-establish
watches without backoff), now has a larger gap to replay, fills its buffer
sooner, and is force-closed again. This is a self-reinforcing loop. At high
write rates it does not converge; the only exits are the write rate dropping or
the watcher's position aging out of the history, which forces a full re-list
that adds yet more load. Because the dispatcher's per-round wait is shared, a
single stuck client also slows delivery to every healthy watcher of the same
resource.

These are not hypothetical. Operators of large clusters with many watch clients
(e.g. per-node agents, or read-through caching proxies fronting the apiserver)
observe both the "one slow client taxes everyone" delay and the
reconnect/replay collapse during events that transiently disconnect many
watchers at once (node churn, zone-connectivity blips).

### Goals

- A watcher that is behind but whose client is still reading is not terminated
  for a full input channel. It catches up from the watch cache history and
  rejoins the live stream on the same connection.
- A slow or stalled watch client imposes no delivery delay or forced close on
  other, healthy watchers of the same resource.
- Catch-up work is shared. Watchers that need the same history share one
  history read and one serialization per event, so the cost of a burst of
  stalls scales with the number of events, not with the number of watchers.
- Preserve watch delivery semantics exactly: every event is delivered to each
  watcher exactly once and in resource version order across the synced and
  unsynced transitions, and no bookmark ever moves a client's last seen
  resource version backwards.
- The only remaining termination for "cannot keep up" is honest: a `410 Gone`
  when the client's resume position has aged out of the watch cache history,
  matching the existing etcd compaction behavior.
- Off by default at alpha and fully reversible via the feature gate, with no
  API, storage, or wire format changes.

### Non-Goals

- Changing how the *initial* hydration of a watch (a `sendInitialEvents`
  WatchList, or a from-store list) is produced or paginated. The initial
  interval is still replayed per watcher, as today. This KEP concerns the
  incremental delivery path only.
- Reducing the cost of a full re-list once a client has genuinely aged out of
  the history.
- Any change to client-go, informer, or reflector behavior. This is an
  apiserver-only change and requires no client changes.
- Guaranteeing a bounded watch delay for a client that is permanently slower
  than the write rate; such a client still eventually receives a `410`.
- Changing the watch channel size heuristics. That is follow-up work once the
  feature is proven (see Notes/Constraints/Caveats).

## Proposal

Behind the `WatchCacheStallResume` gate, the watch cache follows the model of
etcd's watchable store. Synced watchers are fed by the live dispatch path.
Unsynced watchers are served from the history by the same dispatcher goroutine
in bounded passes.

- **Dispatch never blocks and never evicts.** When an event does not fit in a
  watcher's input channel, the dispatcher marks the watcher unsynced, sets its
  position to the missed event's resource version minus one, and moves on. It
  never waits on the shared budget for that watcher and never force-closes it
  for a full channel. While a watcher is unsynced the dispatcher offers it
  nothing, neither object events nor bookmarks.
- **The dispatcher catches unsynced watchers up in shared passes.** Between live
  events, the dispatcher runs sync passes. A pass takes up to 512 unsynced
  watchers, lowest position first, and groups them into cohorts by position.
  Each cohort reads the history once, from its lowest position upward. A history
  event that at least one cohort member selects (by its namespace, name, or
  trigger index scope, the same selection the live path applies) is wrapped
  once into the same caching object the live path uses, and the same pointer is
  pushed to every member that selects it. One pass reads at most 2000 history
  events in total. A watcher's position follows the scan, so a scope-filtered
  watcher crosses stretches of history that hold nothing for it. A watcher
  becomes synced again when it has accepted everything it was offered and the
  scan reached the end of the history. From then on the live path feeds it
  again and skips any event at or below its position.
- **Termination is honest and rare.** If a watcher's position has aged out of
  the history before a pass could serve it, the watcher delivers its remaining
  backlog and then one in-stream `410 Gone` (`Expired`) error event. This is
  exactly what a direct etcd watch receives on compaction. The client
  re-establishes and re-lists, as it does today for compaction.
- **Optional cull for truly dead clients.** An off-by-default
  `StalledClientCullAfter` duration terminates an unsynced watcher whose input
  channel is full and whose position has not changed for that long. The check
  runs inside the sync pass; it adds no goroutine. Such a watcher belongs to a
  client that has stopped reading entirely (e.g. a half-open TCP connection). A
  part-filled channel on a quiet resource is not evidence about the client and
  is not counted. With the cull disabled, the request deadline remains the
  backstop, as today.

### User Stories

**Read-through caching proxy fleet.** An operator runs many caching proxies that
each hold a watch on a large collection. A network blip disconnects a subset;
they reconnect and replay. Today the replaying proxies are force-closed, retry
with larger gaps, and can collapse the whole fleet; the healthy proxies are also
slowed. With the gate on, the reconnecting proxies become unsynced, are served
together from one shared history read per pass, and rejoin the live stream on
the same connection while the healthy proxies are unaffected.

**Per-node agents over a WAN.** Some node agents' watch streams stall at the TCP
layer. Today each stalled stream taxes the shared dispatch path and, when it
falls off the history, forces a re-list that adds load. With the gate on, a
stalled agent is unsynced and costs the other watchers nothing; if it never
recovers it receives an honest `410`.

### Notes/Constraints/Caveats

- The unit of tolerance shifts from "how many events fit in a fixed-size
  channel" (a count) to "how far back the watch cache history reaches" (a time
  window, `eventFreshDuration`). The history window, not the channel size, now
  bounds how long a client may lag before it must re-list.
- Follow-up work, once the feature is proven in production: simplify the
  heuristics that size the per-watcher channels, and reduce the memory those
  channels cost. With the history as the unit of tolerance, a large channel no
  longer buys a slow client anything.
- Correctness across the unsynced and synced transitions rests on these
  invariants, which the code enforces and documents at their call sites:
  1. Position. Every event a watcher must see with a resource version at or
     below its position is already in its input channel, and every relevant
     event above its position is still to be offered (by the live path if
     synced, by a pass if unsynced). A failed push of event X sets the position
     to at most X minus one; X is in the history because the history is
     appended before dispatch, so a pass re-offers X.
  2. Order and single delivery. One goroutine writes into every input channel,
     history order equals dispatch order, and the channel is FIFO, so a client
     sees increasing resource versions. The live path skips any event at or
     below a watcher's position, so an event a pass already pushed is never
     pushed again after resync.
  3. Bookmarks. No bookmark is delivered while a watcher is unsynced. After
     resync, a bookmark below the watcher's position is skipped, so a client's
     last seen resource version never moves backwards.
  4. Selection. A pass applies the same namespace, name, and trigger index
     selection the live path applies, from a copy of the watcher's scope taken
     at registration, so a watcher receives exactly what live dispatch would
     have offered.
- Catch-up work runs on the dispatcher goroutine and delays live dispatch by a
  bounded amount per pass (see Design Details). This is the trade etcd makes.
- All unsynced cohorts share one scan rate. A cohort catches up only while its
  share of that rate exceeds the write rate. Stalled watchers cluster in
  practice (they stall on the same burst and converge onto one position once
  served together), so the number of active cohorts stays small.

### Risks and Mitigations

- **A delivery correctness bug (missed or duplicated event).** This is the
  primary risk of any change to the delivery path. Mitigations: the gate is off
  by default; the invariants above are covered by unit tests run under the race
  detector, an integration test that overflows a real watch, and a model-based
  correctness harness that validates every watch session against the
  linearized write history across stalls and resyncs; and the honest `410`
  fallback means the worst case for an unhandled edge is a re-list, not silent
  data loss.
- **A watcher that never recovers holds resources.** An unsynced watcher
  retains its channels and goroutine. Mitigations: the request deadline bounds
  its lifetime as today; `StalledClientCullAfter` can reclaim confirmed dead
  clients sooner; the `apiserver_watch_cache_stalled_watchers` gauge makes the
  population visible.
- **Catch-up delays live dispatch.** A pass is bounded (at most 2000 history
  events read, at most 512 watchers, at most one channel's capacity of pushes
  per watcher) and runs at most about every 10 ms while any watcher is
  unsynced, so it takes at most about 10 percent of the dispatcher's time.
  During a busy period the pass period shrinks, but it is scaled so the passes
  stay under about that share. The dispatcher also takes the watch cache read
  lock during a pass, which the previous design never did from that goroutine;
  the budget bounds how often it asks, and the reflector's hold time
  (microseconds per event) bounds how long one wait takes.
- **Metric label change.** The existing `apiserver_terminated_watchers_total`
  gains a `reason` label unconditionally (independent of the gate); queries
  selecting the exact label set must use `sum by (group, resource)` to keep their
  previous shape. Documented in the release note.

## Design Details

The change is contained within
`staging/src/k8s.io/apiserver/pkg/storage/cacher`. It mirrors etcd's
watchable store: a synced group fed by notification, an unsynced group served
from history by a bounded background sync, and a compaction-style termination
for watchers that fall too far behind.

"Alpha" and "opt-in" are rollout stages only. The gate lets the new path be
proven in production next to the old one. Once proven, the new path becomes the
only path and the force-close code is deleted.

**Per-watcher state.** With the gate on, each `cacheWatcher` carries fields that
only the dispatcher goroutine writes: a position (resource version), an
unsynced flag, the count of events pushed by passes since it went unsynced, the
time it went unsynced, a copy of its namespace, name and trigger selection
taken at registration, and a flag that says whether the watcher reached its
live loop (set by the watcher goroutine on its first receive; the dispatcher
reads it to pick the `410` reason).

**Live path.** For each object event, the dispatcher skips watchers that are
unsynced or whose position is at or above the event's resource version. On a
successful non-blocking push the position becomes the event's resource
version. On a failed push the position becomes the event's resource version
minus one (never lower than it was) and the watcher is marked unsynced. For
each bookmark, the dispatcher skips unsynced watchers and bookmarks below the
watcher's position; an accepted bookmark advances the position.

**Sync pass.** A pass runs on the dispatcher goroutine and does the following.

1. It reads the oldest servable resource version of the history under the
   watch cache read lock, using the same rule the resume path uses. Then, under
   the cacher lock, it drops unsynced watchers that were forgotten or stopped,
   marks every watcher whose position is below that oldest version minus one
   as expired, and selects up to 512 of the rest that have room in their input
   channel, lowest position first. Expired watchers are stopped in drain mode
   so their backlog is still delivered before the `410`.
2. It forms cohorts. The lead is the first unserved candidate at or above a
   cursor that persists across passes (the position where the previous cohort
   stopped), or the first unserved candidate when none is above it. The pass
   opens one history interval from the lead's position and scans it up to the
   remaining budget. The cohort is the lead plus every unserved candidate above
   it whose position is below the last scanned resource version. A candidate
   below the lead is never in the cohort, because it would be offered nothing
   between its position and the lead's.
3. For each scanned history event, it computes the namespace, name, and
   trigger values once, on the immutable history object. The first cohort
   member that selects the event makes the one shallow copy and caching object
   wrapper, exactly as the live path does; later members reuse the pointer.
   Events no member selects are never wrapped.
4. For each member, it pushes every selected event above the member's position
   in order with a non-blocking add. A success advances the position. A failure
   stops the offering to that member for this cohort; the member keeps its last
   accepted resource version. Members that stopped on a full channel are
   re-offered from where they stopped, up to two more rounds within the same
   pass, because the watcher goroutine drains the channel concurrently.
5. A member that accepted everything it was offered has its position advanced
   to the last scanned resource version. This is what lets a scope-filtered
   watcher cross a window that holds nothing for it, and what makes cohort
   members converge onto one position.
6. Under the cacher lock, a member that accepted everything it was offered and
   whose cohort scan reached the end of the history becomes synced. The pass
   then releases the deferred stops, including the expired members' drain mode
   stops.

A pass reads at most 2000 history events across all cohorts, serves at most
512 watchers, pushes at most one channel's capacity per watcher per round, and
wraps each selected event once. The scan rate is about 200,000 history events
per second at the 10 ms period.

**When passes run.** A pass runs after a dispatched event when the unsynced set
is not empty and the current period has elapsed since the last pass, and on a
timer so an idle cacher still drains. The period is 10 ms normally. After a
pass that left a member owed events with a full channel (that client is
draining), the period drops to a busy value of at least 1 ms, scaled so that
passes take under about 10 percent of the dispatcher's time. After a pass that
served and expired nothing, the period rises to 100 ms. A new stall resets it
to 10 ms. The precedent is etcd: its victim retry runs every 10 ms and its
unsynced loop every 100 ms, and both yield after progress.

**Watcher goroutine.** The watcher's processing loop is the existing one, with
two additions. It marks itself live on its first receive. When its input
channel closes and the dispatcher marked it expired, it sends the in-stream
`410` and returns. In drain mode the done channel stays open, so the backlog
and then the `410` are delivered as long as the client reads; a wedged client
is bounded by the request timeout, exactly like today's graceful close path.

**The `410` semantics.** A watcher's position is only ever the missed event's
resource version minus one or later, never a stale "last delivery" resource
version of a scope-filtered watcher. So an idle per-node watcher that stalls
once resumes after one or a few scans instead of receiving a `410`. Expiry is
evaluated for every unsynced watcher on every pass, whether it was a candidate
or not. The reason is `resource_expired` for a watcher that reached its live
loop and `resource_expired_initial` for one that never did. One consequence
reaches the initial phase of a watch: with the gate on, a watcher whose
*initial* interval (the history segment or store snapshot it is hydrated from)
is overtaken by the history while it is still being streamed receives the same
in-stream `410` (counted as `resource_expired_initial`) instead of today's
silent close. How the initial state is produced is otherwise unchanged.

**Memory.** A wrapped event lives while any input or result channel references
it. It is shared, not copied per watcher, and its serialization is cached once
it has been encoded for one watcher. The number of distinct in-flight wrapped
events is bounded by the sum over unsynced watchers of their input and result
channel capacities. Whenever watchers share events, which is the case the
mechanism exists for, the memory is far below one copy per watcher and event.

**Gate off.** With the gate off, the dispatch path and the watcher processing
loop keep their previous control flow on every dispatch and delivery path: no
pass, no pass timer, no extra state read. The gate-independent changes are the
`reason` metric label below, the formatting of one pre-existing warning log
line, and one latent bug fix: a hard stop (a client `Stop` or a
`terminateAllWatchers` on relist or shutdown) can no longer be reopened into
drain mode by a later graceful stop. That path only closes the done channel
where the previous code left it open, which otherwise leaked the goroutine.

**Metrics.** New observability, all alpha, on kube-apiserver:

- `apiserver_watch_cache_watcher_stalls_total` (counter): transitions from
  synced to unsynced.
- `apiserver_watch_cache_watcher_deferred_events_total` (counter): events
  pushed by sync passes rather than by the live path.
- `apiserver_watch_cache_watcher_catchup_rounds_total` (counter): transitions
  from unsynced back to synced. It equals the count of the histogram below.
- `apiserver_watch_cache_watcher_catchup_events` (histogram): events pushed to
  one watcher between going unsynced and resyncing.
- `apiserver_watch_cache_stalled_watchers` (gauge): the number of unsynced
  watchers. It lands with the cull in the follow-up PR.
- `apiserver_terminated_watchers_total` gains a `reason` label:
  `unresponsive` (today's force-close, gate off), `resource_expired` (gate on:
  a live watcher's position aged out of the history),
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
  cases, all run under the race detector: the synced to unsynced transition
  and the position it records; cohort formation and the single wrap per
  selected event; the scan budget and the member cap; position following the
  scan for scope-filtered and trigger-indexed watchers after unrelated churn;
  the resync condition; no bookmarks while unsynced and the bookmark skip after
  resync; both `410` reasons and the delivery of the backlog before the `410`;
  a watcher stopped by its client between the stall and the pass; clean closes
  on context cancellation and Cacher shutdown; the hard stop fix; the cull
  driven by a fake clock; and gate-off parity, where several pre-existing watch
  cache tests run with the gate both on and off and the gate-off path must
  match prior behavior.

##### Integration tests

- `test/integration/apiserver`: a real kube-apiserver watch whose client stops
  reading for 6 seconds while more data than the HTTP/2 stream window plus the
  watcher channels can hold is written. With the gate on, every event is
  delivered in order once the client resumes and the watch stays open. With
  the gate off (control), the server terminates the watch during the stall.

##### e2e tests

None planned for alpha; the behavior is exercised by the integration test and
the two validations below.

- Benchmark (`kubernetes/kubernetes#141475`, merged): a healthy watcher next
  to slow ones. With the gate on there are zero force-closes and the healthy
  watcher's p99 delivery latency is under 100 microseconds. With the gate off
  the same run shows 16 to 24 ms and 20 force-closes.
- Model-based correctness harness (`kubernetes/kubernetes#142381`) stacked on
  the change with slow readers: every watch session is validated against the
  linearized write history across stalls and resyncs.

### Graduation Criteria

"Alpha" and "opt-in" are rollout stages. The end state is that the new path is
the only path.

#### Alpha

- Feature implemented behind the `WatchCacheStallResume` gate, off by default.
- Unit and integration tests as above, passing with the gate on and off.
- The benchmark and the model-based correctness harness pass with the gate on.
- Metrics listed above emitted.

#### Beta

- Feedback and soak from clusters running with the gate enabled.
- No unresolved delivery correctness issues.
- Scalability sign-off that the sync pass does not regress live dispatch or
  memory under realistic slow-client populations.
- Gate enabled by default.
- Decision on whether `StalledClientCullAfter` should have a non-zero default.

#### GA

- The feature has been enabled by default (beta) across at least two releases
  without correctness or scalability regressions.
- The old force-close path is removed and the gate is locked on.
- The watch channel size heuristics are revisited, since the history window
  now bounds tolerance.
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

Yes, for watch delivery under load: a watch whose input channel fills becomes
unsynced and is served from history instead of being force-closed, and a slow
watcher no longer delays delivery to other watchers. No API responses change
shape; a resynced watch delivers the same events it would have, and
terminations still surface as `410 Gone`. The `reason` label added to
`apiserver_terminated_watchers_total` is the one gate-independent change to
the metrics.

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
touch workloads directly. A rollout failure would manifest as a delivery
correctness bug (a missed or duplicated watch event), as unsynced watchers not
being reclaimed, or as sync passes taking more of the dispatcher's time than
their bound. A missed or duplicated event could cause controllers to act on
stale state; this is mitigated by the off-by-default gate, the delivery
invariants and their tests, the model-based harness, and the `410` fallback.
Rolling the gate back off immediately restores prior behavior.

###### What specific metrics should inform a rollback?

- A rise in watch-related client errors or controller staleness after enabling.
- `apiserver_watch_cache_stalled_watchers` growing without bound.
- `apiserver_terminated_watchers_total{reason="resource_expired"}` far exceeding
  the pre-enablement `reason="unresponsive"` rate (clients aging out rather than
  catching up), which would indicate the history window is too small for the
  workload.
- A rise in the existing watch dispatch latency metrics for healthy watchers
  while `apiserver_watch_cache_stalled_watchers` is non-zero, which would mean
  sync passes cost more than their bound.

###### Were upgrade and rollback tested? Was the upgrade->downgrade->upgrade path tested?

To be completed before beta; for alpha the gate on/off tests exercise both
behaviors within one binary.

###### Is the rollout accompanied by any deprecations and/or removals?

No.

### Monitoring Requirements

###### How can an operator determine if the feature is in use by workloads?

The `apiserver_watch_cache_watcher_stalls_total` and
`apiserver_watch_cache_watcher_catchup_rounds_total` counters are non-zero when
watches go unsynced and resync, and `apiserver_watch_cache_stalled_watchers`
shows the current unsynced population.

###### How can someone using this feature know that it is working for their instance?

- [x] Metrics
  - Metric names: the `apiserver_watch_cache_watcher_*` series above, the
    `apiserver_watch_cache_stalled_watchers` gauge, and the `reason` label on
    `apiserver_terminated_watchers_total` (a shift from `unresponsive` toward
    zero force-closes under load indicates the feature is doing its job).

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
    stall episode by the history window).

###### Are there any missing metrics that would be useful to have?

Not identified for alpha. A histogram of sync pass duration may be added
before beta if the scalability sign-off needs it.

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

It is expected to *reduce* watch dispatch delay under slow-client load (a slow
watcher no longer stalls the shared dispatch path; the benchmark shows a
healthy watcher's p99 delivery latency drop from 16 to 24 ms to under 100
microseconds). It adds sync passes on the dispatcher goroutine, each bounded
(at most 2000 history events read, at most 512 watchers, one channel's capacity
of pushes per watcher per round) and run at most about every 10 ms while any
watcher is unsynced, so at most about 10 percent of the dispatcher's time. It
adds no goroutine.

###### Will enabling / using this feature result in non-negligible increase of resource usage in any components?

An unsynced watcher retains the same channels and goroutine it holds today; the
feature does not create additional per-watcher buffers or per-watcher copies of
events. Events pushed by a pass are shared between the watchers that select
them, and their serialization is cached, as on the live path. The main change is
that a slow watcher may live longer (until it resyncs, ages out, or is culled)
than it would if force-closed. That is bounded by the request deadline and,
optionally, `StalledClientCullAfter`. The catch-up path reads the existing
history buffer; it makes no allocations proportional to state size.

###### Can enabling / using this feature result in resource exhaustion of some node resource?

Not on nodes. On kube-apiserver, a pathological population of never-recovering,
never-culled slow clients could retain more watcher goroutines and channels than
the force-close behavior would. This is bounded by request deadlines, observable
via `apiserver_watch_cache_stalled_watchers`, and reclaimable via
`StalledClientCullAfter`.

### Troubleshooting

###### How does this feature react if the API server and/or etcd is unavailable?

The feature lives entirely in the apiserver's in-memory watch cache and does not
add etcd interactions. If etcd is unavailable, watch behavior is governed by the
existing watch cache logic; this feature does not change that.

###### What are other known failure modes?

- Clients repeatedly aging out (`reason="resource_expired"`): the history window
  is too small for the workload's write rate; increase the window or investigate
  the slow clients. Diagnose via the terminated-watchers `reason` label and the
  stalled-watchers gauge.
- Growing `apiserver_watch_cache_stalled_watchers`: a population of clients that
  stall and never recover; consider enabling `StalledClientCullAfter`.
- Many cohorts active at once on a resource with a very high write rate: the
  shared scan rate is split between them and the slowest cohort may not outrun
  the history. It then ages out with an honest `410` and re-lists.

###### What steps should be taken if SLOs are not being met to determine the problem?

Disable the `WatchCacheStallResume` gate to return to the prior behavior, then
inspect the metrics above to characterize the slow-client population.

## Implementation History

- 2026-08-06: KEP drafted; alpha implementation prototyped and validated on a
  scalability rig against a single apiserver + caching-proxy fleet.
- 2026-09-25: design revised to server-driven batched resync at reviewer
  request; the dispatcher serves unsynced watchers from history in bounded,
  shared passes, following etcd's watchable store.

## Drawbacks

It adds state and a second delivery path to the watch cache, a component whose
correctness is critical, and it puts catch-up work on the dispatcher goroutine.
The complexity is justified by removing a force-close/reconnect collapse that
has no in-place mitigation today. It follows a model etcd has run for years, it
is contained behind an off-by-default gate whose disabled path keeps its
control flow, and the intent is to delete the old path once the new one is
proven.

## Alternatives

- **Do nothing / rely on larger buffers.** Increasing the per-watcher channel
  size raises the count of events a slow watcher can absorb but does not change
  the "buffer full, so force-close, and the shared wait taxes everyone"
  structure; it only moves the threshold.
- **Remove the watcher from the dispatcher and let it re-initialize itself.**
  On a failed push, record the missed resource version, unregister the
  `cacheWatcher` from the Cacher so live dispatch never touches it again, and,
  once the watcher has drained its input channel, have it re-initialize exactly
  as a new watch does: replay from the recorded resource version through the
  existing initial interval code, then re-register. This reuses the existing
  replay path and keeps the faulty watcher fully isolated. It was not chosen
  because of cost at scale. Each re-initializing watcher walks the history on
  its own and serializes each object separately, so a burst that stalls many
  watchers (the case this KEP exists for) costs one history walk and one
  serialization per watcher and event. Watcher resync was one of the main
  allocation sources on the bad path found in
  `kubernetes/kubernetes#142223`. The server-driven batched resync adopted
  here, which is what etcd does, serves many watchers from one history read
  and one cached serialization per event, so it scales to many watchers. It
  also keeps the scope of the change smaller than committing to permanently
  cached objects for per-watcher replay.
- **Per-watcher catch-up on the watcher's own goroutine.** The first draft of
  this KEP had each stalled watcher re-read the events it missed from the
  history on its own goroutine. It has the same per-watcher cost as the option
  above (one history walk, one deep copy and one serialization per watcher and
  event), plus a large body of per-watcher ordering rules to review. It was
  replaced by the shared passes described here at reviewer request.
- **Only cull zero-progress clients.** Culling clients that read nothing (the
  optional `StalledClientCullAfter` here) addresses dead connections but not the
  reconnect/replay collapse of healthy-but-behind watchers, which is the main
  motivation.
- **A hard cap on catch-up work per watcher.** Bounding the number of catch-up
  rounds and terminating past it was considered; it re-introduces a force-close
  for slow-but-live clients and is unnecessary because aging out of the history
  window already provides an honest, time-denominated bound. Kept as a possible
  future safeguard, not required for the core behavior.
