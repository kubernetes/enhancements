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
  - [Initial values](#initial-values)
  - [Target design (Beta)](#target-design-beta)
  - [How we measure success](#how-we-measure-success)
  - [Rollout](#rollout)
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
still full after that, it force-closes the watch as "unresponsive". The
dispatcher's wait delays delivery to every watcher of that resource. A watcher
that is merely behind is guaranteed to fill its channel and be terminated. The
termination triggers an immediate reconnect with a larger gap, and the loop
repeats. A re-established watch is costly for the apiserver as well: a watch
opened at a resource version replays its catch-up without the serialization
cache, as described in
[kubernetes/kubernetes#142223](https://github.com/kubernetes/kubernetes/issues/142223).

This KEP replaces that mechanism with the synced and unsynced watcher model
that etcd uses in its own watchable store. This is cross-pollination between
the two projects, not a new design: etcd borrowed the watch cache from
Kubernetes, and Kubernetes now borrows watcher synchronization from etcd. A
watcher whose input channel is full becomes unsynced. The dispatcher stops
offering it events and records the position from which it must resume. Unsynced
watchers are served from the in-memory watch cache history, and watchers that
need the same stretch of history share one history read and one serialized
copy of each event. A watcher rejoins the live stream on the same connection
once it has accepted every event the history holds for it. A watcher is
terminated only when the history no longer holds the events it needs.

The properties are: a slow or stalled client imposes no delivery cost on
healthy watchers; a client that is behind but reading gets the full history
window to recover; and the cost of catching up scales with the number of
events, not with the number of watchers.

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

Each termination is also costly on the server side. A watch re-established at
a resource version replays its catch-up without the serialization cache
(`cachedObject`), so every object is serialized again for that one watcher.
Watcher resync was one of the main allocation sources on the bad path found in
[kubernetes/kubernetes#142223](https://github.com/kubernetes/kubernetes/issues/142223).
A burst of terminations therefore turns into an allocation storm that is hard
to recover from.

The client is not the only cause of a full channel. A short halt in the
apiserver's own processing, for example an allocation storm or a goroutine
storm in the Go runtime, stops the dispatcher long enough for channels to fill
and trips the time-based termination, even when the apiserver is not heavily
loaded
([kubernetes/kubernetes#142223](https://github.com/kubernetes/kubernetes/issues/142223)).
Removing that delicate time-based mechanism matters for the reliability of the
apiserver itself, not only for slow clients. This is reproducible in write
throughput benchmarks: the moment one watcher breaks, the apiserver degrades
out of proportion to the load.

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
  interval is still replayed per watcher, as today. Serving it through the
  same resync path as a stalled watcher is the target design and a Beta
  criterion, not part of alpha.
- Reducing the cost of a full re-list once a client has genuinely aged out of
  the history.
- Any change to client-go, informer, or reflector behavior. This is an
  apiserver-only change and requires no client changes.
- Guaranteeing a bounded watch delay for a client that is permanently slower
  than the write rate; such a client still eventually receives a `410`.
- Changing the watch channel size heuristics. That is follow-up work once the
  feature is proven (see Notes/Constraints/Caveats).
- Terminating a client that never reads (a cull). That belongs with the
  handling of misbehaving clients, which crosses into API Priority and
  Fairness. It is out of scope for alpha. Such a watcher stays bounded by the
  request deadline, as today, and the stalled-watchers gauge measures that
  population before any decision about it.

## Proposal

Behind the `WatchCacheStallResume` gate, the watch cache follows the model of
etcd's watchable store. Synced watchers are fed by the live dispatch path.
Unsynced watchers are served from the history in shared passes. The proposal
is defined by the properties and the trade-off below. The concrete parameters
are initial values, listed in one place in Design Details, that benchmarks and
the simulator will revise.

- **A stalled watcher never blocks or delays a healthy one.** When an event
  does not fit in a watcher's input channel, the dispatcher marks the watcher
  unsynced, records the position from which it must resume, and moves on. It
  never waits for that watcher and never force-closes it for a full channel.
  While a watcher is unsynced the dispatcher offers it nothing.
- **One serialization per event per pass, shared by every watcher that needs
  it.** A pass serves the unsynced watchers that need the same stretch of
  history from one history read. Each event is wrapped once per pass, and the
  same object with its cached serialization is pushed to every watcher that
  selects it. The cost of a burst of stalls scales with the number of events,
  not with the number of watchers.
- **Every unsynced watcher with room in its channel is served within a bounded
  number of passes.** A pass takes the watchers that have room and rotates
  among them across passes (round robin from a cursor that persists between
  passes), so no watcher is starved by its position. A watcher that cannot
  accept an event stays unsynced, keeps its last accepted resource version,
  and is offered again in a later pass.
- **Termination only when the history no longer holds the needed events.** A
  watcher whose resume position has aged out of the history delivers its
  remaining backlog and then one in-stream `410 Gone` (`Expired`) error event.
  This is the same in-stream `410` an etcd watch receives on compaction. The
  client re-establishes and re-lists, as it does today for compaction. There is
  no other termination for a watcher that cannot keep up.
- **No bookmarks while unsynced.** A bookmark carries the latest resource
  version the dispatcher has processed. Delivered to a watcher that is still
  behind, or a bookmark below the watcher's position after resync, it would
  move the client's last seen resource version backwards relative to what the
  client has actually received. A reconnect from that resource version would
  then replay events the client already received. So no bookmark is delivered
  while a watcher is unsynced, and after resync a bookmark below the watcher's
  position is skipped. This differs from etcd, where a watch can be opened at a
  future revision, so etcd's synced set also holds watchers that legitimately
  skip events. Kubernetes watches never start in the future, so a skip on the
  live path here is a safety check, not a mechanism (see Design Details).
- **The trade-off: event reuse first, fairness by rotation second.** A pass
  groups watchers by position so that one history read and one serialization
  serve as many watchers as possible. That is efficiency. Rotation across
  passes guarantees that every watcher with room is served within a bounded
  number of passes. That is fairness. When the two conflict inside one pass,
  the pass favors reuse, and fairness comes from rotation across passes. The
  simulator described under How we measure success is the tool to revisit this
  choice.

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

**An apiserver that is itself overloaded.** In
[kubernetes/kubernetes#142223](https://github.com/kubernetes/kubernetes/issues/142223)
the apiserver, not the client, is the slow party. An allocation storm or a
goroutine storm in the Go runtime stalls watch processing for a short time.
Channels fill, the time-based termination fires, and the terminated watchers
reconnect at once and replay without the serialization cache. That replay adds
allocations and deepens the storm. With the gate on, the watchers become
unsynced during the halt and are served from the history with shared
serializations once the dispatcher runs again. No watch is re-established, and
no replay runs without the cache.

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
     pushed again after resync. That skip is a safety check with a counter;
     the target resync point makes it unnecessary (see Design Details).
  3. Bookmarks. No bookmark is delivered while a watcher is unsynced. After
     resync, a bookmark below the watcher's position is skipped, so a client's
     last seen resource version never moves backwards and a reconnect never
     replays events the client already received. Unlike etcd, no watcher here
     starts at a future resource version.
  4. Selection. A pass applies the same namespace, name, and trigger index
     selection the live path applies, from a copy of the watcher's scope taken
     at registration, so a watcher receives exactly what live dispatch would
     have offered.
- In alpha, catch-up work runs on the dispatcher goroutine and delays live
  dispatch by a bounded amount per pass. The target design moves it to its own
  goroutine (see Design Details).
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
  its lifetime as today, and the `apiserver_watch_cache_stalled_watchers` gauge
  makes the population visible. That measurement comes before any decision on
  reclaiming such watchers sooner.
- **Catch-up delays live dispatch.** In alpha a pass runs on the dispatcher
  goroutine. It is bounded in watchers served, history events read, and pushes
  per watcher, and its period is scaled so passes stay under about a tenth of
  the dispatcher's time (initial values in Design Details). The dispatcher also
  takes the watch cache read lock during a pass, which the previous design
  never did from that goroutine; the budget bounds how often it asks, and the
  reflector's hold time (microseconds per event) bounds how long one wait
  takes. At the boundary of the apiserver's watch throughput this bound is
  still a cost paid by live dispatch; the target design removes it by serving
  unsynced watchers from their own goroutine, which is a Beta criterion.
- **Starvation of new or synced watchers by stale ones.** A large population of
  stale watchers could take every pass. Mitigations: candidates are chosen by
  free room and rotated from a persistent cursor, so a watcher that can catch
  up is served first and every watcher with room is served within a bounded
  number of passes; the simulator measures this fairness before Beta.

## Design Details

The change is contained within
`staging/src/k8s.io/apiserver/pkg/storage/cacher`. It mirrors etcd's
watchable store: a synced group fed by notification, an unsynced group served
from history by a bounded sync, and a compaction-style termination for
watchers that fall too far behind. In alpha the sync runs on the dispatcher
goroutine in bounded passes. The target design, described below, serves the
unsynced group from its own goroutine.

**Per-watcher state.** With the gate on, each `cacheWatcher` carries fields that
only the dispatcher goroutine writes: a position (resource version), an
unsynced flag, the count of events pushed by passes since it went unsynced, a
copy of its namespace, name and trigger selection taken at registration, and a
flag that says whether the watcher reached its live loop (set by the watcher
goroutine on its first receive; the dispatcher reads it to pick the `410`
reason). The cacher carries the set of unsynced watchers and the pass cursor.

**Live path.** For each object event, the dispatcher skips watchers that are
unsynced or whose position is at or above the event's resource version. The
second skip is counted in `apiserver_watch_cache_watcher_skipped_events_total`,
which lands in the follow-up PR with the gauge (see the resync point below). On a successful non-blocking push the position
becomes the event's resource version. On a failed push the position becomes
the event's resource version minus one (never lower than it was) and the
watcher is marked unsynced. For each bookmark, the dispatcher skips unsynced
watchers and bookmarks below the watcher's position; an accepted bookmark
advances the position.

**Sync pass.** A pass runs on the dispatcher goroutine and does the following.

1. It reads the oldest servable resource version of the history under the
   watch cache read lock, using the same rule the resume path uses. Then, under
   the cacher lock, it drops unsynced watchers that were forgotten or stopped,
   and marks every watcher whose position is below that oldest version minus
   one as expired. Expired watchers are stopped in drain mode so their backlog
   is still delivered before the `410`. The candidates are the remaining
   unsynced watchers with room in their input channel. Candidates are ordered
   by three keys: first by the free share of their channel (a share, so a
   watcher with a small channel and one with a large channel compare fairly),
   then by whether the watcher is new to the unsynced set or drained its
   channel since its last service,
   then by rotation from a cursor that persists across passes (a position at
   or above the cursor comes first). When there are more candidates than the
   member cap, the first ones in that order are kept. A watcher that can catch
   up is served first, laggards take turns across passes, and no watcher is
   starved by its position.
2. It forms cohorts. The lead is the first unserved candidate by the same
   keys. The pass opens one history interval from the lead's position and
   scans it up to the remaining budget. Followers join as the scan passes
   their position: the cohort is the lead plus every unserved candidate above
   it whose position is below the last scanned resource version. A candidate
   below the lead is never in the cohort, because it would be offered nothing
   between its position and the lead's. After the pass, the cursor is the last
   scanned resource version of the last cohort served.
3. For each scanned history event, it computes the namespace, name, and
   trigger values once, on the immutable history object. The first cohort
   member that selects the event makes the one shallow copy and caching object
   wrapper, exactly as the live path does; later members reuse the pointer.
   Events no member selects are never wrapped.
4. For each member, it pushes every selected event above the member's position
   in order with a non-blocking add. A success advances the position. A
   failure ends the offering to that member for this cohort; the member keeps
   its last accepted resource version and records the scanned events it is
   still owed. At the end of the pass, after the other cohorts ran, the owed
   events are offered once more to the fast-draining members whose channel has
   room again, in up to two rounds, because the watcher goroutine drains the channel
   concurrently. A member that still rejects stays unsynced with its last
   accepted resource version and is offered again in a later pass.
5. A member that accepted everything it was offered has its position advanced
   to the last scanned resource version. This is what lets a scope-filtered
   watcher cross a window that holds nothing for it, and what makes cohort
   members converge onto one position.
6. Under the cacher lock, a member that accepted everything it was offered and
   whose cohort scan reached the resync point becomes synced. The pass then
   releases the deferred stops, including the expired members' drain mode
   stops.

**The resync point.** The target resync point is the last resource version the
dispatcher has dispatched. A watcher that resyncs there has received exactly
what the live path has delivered so far, so the live path never has an event
to skip for it. Alpha resyncs at the end of the history instead. The history
runs ahead of the dispatcher's incoming channel by up to that channel's
capacity, so an alpha watcher can resync at a position the live path has not
dispatched yet, and the live path then skips the events at or below its
position. That skip stays as a safety check in both cases, and
`apiserver_watch_cache_watcher_skipped_events_total` counts it. Once the
resync point is the last dispatched resource version, a non-zero value means
the resync point is wrong. Moving the resync point is a Beta criterion.

**Bounds.** A pass is bounded in four ways: the number of watchers it serves,
the scan units it spends (history events read plus interval opens), the pushes
it makes, and one wrap per selected event. The scan and push budgets are
totals for the whole pass across all cohorts, not per watcher. The initial
values are listed below.

**When passes run.** A pass runs after a dispatched event when the unsynced set
is not empty and the current period has elapsed since the last pass, and on a
timer so an idle cacher still drains. The period has three values. Normal is
the default. After a pass that left a member owed events with a full channel
(that client is draining), or that spent its push budget, the period drops to a busy value, scaled from the
measured duration of the pass so that passes take under about a tenth of the
dispatcher's time. After a pass that served and expired nothing, the period
rises to an idle value. A new stall resets it to normal. The precedent is
etcd: its victim retry and its unsynced loop both yield after progress.

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
  watchers. It lands in a follow-up PR.
- `apiserver_watch_cache_watcher_skipped_events_total` (counter): events the
  live path skipped because they were at or below a watcher's position. It
  lands in the follow-up PR with the gauge. Once
  the resync point is the last dispatched resource version, a non-zero value
  means the resync point is wrong.
- `apiserver_terminated_watchers_total` gains a `reason` label:
  `unresponsive` (today's force-close, gate off), `resource_expired` (gate on:
  a live watcher's position aged out of the history), and
  `resource_expired_initial` (gate on: the same, before the watcher ever
  reached the live stream, i.e. during or right after its initial hydration).
  `apiserver_terminated_watchers_total` is an ALPHA metric under the
  Kubernetes metrics stability framework, so adding a label is within its
  guarantees. The label is added independent of the gate and is noted in the
  release note.

### Initial values

The bounds and periods below are starting points, not tuned values. They were
validated only by the benchmark and by a 512-watcher scenario, not by a
large-cluster test. etcd's tuning does not carry over: a large kube-apiserver
serves hundreds of thousands of watchers, where etcd serves hundreds to
thousands. The simulator described under How we measure success is the tool
to revise them.

- Watchers per pass: 512.
- Scan budget: 2000 scan units per pass, in total across all cohorts, not per
  watcher. A scanned history event costs one unit. An interval open costs 64
  units, because an open takes the watch cache read lock and allocates a
  buffer, so many watchers with tiny windows cannot multiply the per-open
  cost.
- Push budget: 2000 pushes per pass across all cohorts, plus the same
  allowance again for the end-of-pass retry rounds, and at most one channel's
  capacity per watcher per cohort.
- Pass period: 10 ms normal. Busy: scaled from the measured duration of the
  pass to keep passes under about a tenth of the dispatcher's time, at least
  1 ms and at most 10 ms. Idle: 100 ms.
- Measured on the benchmark host, a full pass costs about a millisecond, so
  the scan rate is about 200,000 history events per second at the 10 ms
  period. A pass never dispatches 512 times 2000 events; the budgets are
  totals for the pass.

### Target design (Beta)

Alpha runs the passes on the dispatcher goroutine. That keeps one writer per
input channel and made the first implementation reviewable, and the pass is
bounded so it stays under about a tenth of that goroutine's time. That bound
is still a cost paid by live dispatch. In an environment with many stalled
watchers, at the boundary of the apiserver's watch throughput, serving them
from the dispatcher goroutine makes the healthy watchers lag, which works
against the goals. The pass also walks the unsynced set once per pass, which
is linear in that population. The target design removes both costs from the
dispatcher goroutine:

- **The unsynced group is served from its own goroutine**, as etcd's
  `syncWatchersLoop` does. The hand-off to the live stream is synchronized
  with the dispatcher only for the last few events, when a watcher approaches
  the present. A watcher that is further behind is served without
  synchronizing with live dispatch, so live dispatch pays nothing for it.
- **One resync path.** A watch that starts at an old resource version and a
  watch that fell behind are the same case: an unsynced watcher with a
  position. Both are served by the one resync path, instead of the per-watcher
  initial replay and the shared passes side by side. Alpha keeps the initial
  replay per watcher, as today, to keep the first change small and
  gate-isolated.
- **Separate sets.** The synced and unsynced watchers are kept in separate
  sets, as in etcd's cache demux
  ([etcd `cache/demux.go`](https://github.com/etcd-io/etcd/blob/7583cc6e7e2756bb4166d646fe48d2fa1863b4b6/cache/demux.go#L30-L31)).
  Live dispatch iterates only the synced set and the sync goroutine only the
  unsynced set, so each can be scanned independently, and the linear walk of
  the unsynced population moves off the dispatcher goroutine with the rest.
- **Resync at the last dispatched resource version**, so the live path never
  has an event to skip and `apiserver_watch_cache_watcher_skipped_events_total`
  stays at zero.

These are Beta graduation criteria.

### How we measure success

The mechanism is judged by four measurements, taken with the gate on:

1. Healthy watcher delivery latency is unchanged with N stalled watchers
   present, for N up to the populations seen in large clusters.
2. Serializations per event stay close to one per pass while many watchers
   catch up together.
3. The time for a slow watcher to resync, as a function of its read rate and
   the write rate.
4. No starvation: every unsynced watcher with room in its channel is served
   within a bounded number of passes, including a watcher that just became
   unsynced next to long-standing laggards.

Two tools produce these measurements. The model-based correctness harness
([kubernetes/kubernetes#142381](https://github.com/kubernetes/kubernetes/pull/142381))
covers correctness: it validates every watch session against the linearized
write history across stalls and resyncs. A simulator covers efficiency and
fairness. Its inputs are an event rate with a label distribution (how events
spread over namespaces and names, which decides how many watchers select each
event) and a model of watcher behavior (read rates, stalls, reconnects,
approximating production). Its outputs are which watchers were terminated and
when, the per-watcher event latency, and the number of serializations per
event. The simulator is the tool to revisit the initial values (the member
cap, the scan and push budgets, the periods) and the candidate order, and to
compare algorithm variants on the same input. The simulator and measured
values for the four measurements above are Beta criteria.

### Rollout

The mechanism ships behind the `WatchCacheStallResume` feature gate on
kube-apiserver, off by default at alpha. "Alpha" and "opt-in" are rollout
stages, not the end state. The gate exists so the new path can be proven in
production next to the old one. Once proven, the new path becomes the behavior
of the watch cache and the old force-close path is removed.

**Gate off.** With the gate off, the dispatch path and the watcher processing
loop keep their previous control flow on every dispatch and delivery path: no
pass, no pass timer, no extra state read. The gate-independent changes are the
`reason` metric label above, the formatting of one pre-existing warning log
line, and one latent bug fix: a hard stop (a client `Stop` or a
`terminateAllWatchers` on relist or shutdown) can no longer be reopened into
drain mode by a later graceful stop. That path only closes the done channel
where the previous code left it open, which otherwise leaked the goroutine.

### Test Plan

[ ] I understand the owning sig for the component(s) is responsible for
maintaining the tests owned by that sig.

##### Prerequisite testing updates

None.

##### Unit tests

- `staging/src/k8s.io/apiserver/pkg/storage/cacher`: existing coverage plus new
  cases, all run under the race detector: the synced to unsynced transition
  and the position it records; the candidate order (free share of the
  channel, drained since the last service, rotation from the cursor); cohort
  formation and the single wrap per selected event; the scan budget and the
  member cap; the owed events re-offered at the end of the pass; position
  following the scan for scope-filtered and trigger-indexed watchers after
  unrelated churn; the resync condition and the skipped-events counter; no
  bookmarks while unsynced and the bookmark skip after resync; both `410`
  reasons and the delivery of the backlog before the `410`; a watcher stopped
  by its client between the stall and the pass; clean closes on context
  cancellation and Cacher shutdown; the hard stop fix; and gate-off parity,
  where several pre-existing watch cache tests run with the gate both on and
  off and the gate-off path must match prior behavior.

##### Integration tests

- `test/integration/apiserver`: a real kube-apiserver watch whose client stops
  reading while more data than the HTTP/2 stream window plus the watcher
  channels can hold is written. With the gate on, every event is
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
- The passes run off the dispatcher goroutine, with the hand-off to the live
  stream synchronized only for the last few events (Target design).
- One resync path serves both a watch that starts at an old resource version
  and a watch that fell behind, with the synced and unsynced sets kept
  separate (Target design).
- The resync point is the last dispatched resource version, and
  `apiserver_watch_cache_watcher_skipped_events_total` is zero in the tests.
- The simulator exists, the four measurements under How we measure success
  have measured values, and the initial values are revised from them.
- Scalability sign-off, based on those measurements, that the sync does not
  regress live dispatch or memory under realistic slow-client populations.
- The failure modes section of the PRR questionnaire is restructured into
  detection, mitigation, diagnostics, and testing.
- Gate enabled by default.

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

There are no separate enablement tests. The change is purely in memory, so the
feature tests themselves run with the gate on and off. The gate-off runs assert
the previous behavior, including that an overflowing watch is terminated.

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
  - `apiserver_watch_cache_watcher_skipped_events_total` (zero once the resync
    point is the last dispatched resource version; a non-zero value then means
    the resync point is wrong).

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
microseconds). In alpha it adds sync passes on the dispatcher goroutine, each
bounded in watchers served, history events read, and pushes, and run on a
period scaled so that passes take at most about a tenth of the dispatcher's
time (initial values in Design Details). It adds no goroutine in alpha. The
target design moves the passes off the dispatcher goroutine, which is a Beta
criterion.

###### Will enabling / using this feature result in non-negligible increase of resource usage in any components?

An unsynced watcher retains the same channels and goroutine it holds today; the
feature does not create additional per-watcher buffers or per-watcher copies of
events. Events pushed by a pass are shared between the watchers that select
them, and their serialization is cached, as on the live path. The main change is
that a slow watcher may live longer (until it resyncs or ages out) than it
would if force-closed. That is bounded by the request deadline. The catch-up
path reads the existing history buffer; it makes no allocations proportional
to state size.

###### Can enabling / using this feature result in resource exhaustion of some node resource?

Not on nodes. On kube-apiserver, a pathological population of never-recovering
slow clients could retain more watcher goroutines and channels than the
force-close behavior would. This is bounded by request deadlines and
observable via `apiserver_watch_cache_stalled_watchers`. That gauge measures
the population before any decision on reclaiming such watchers sooner.

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
  stall and never recover. Each is bounded by its request deadline; investigate
  the clients.
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
- 2026-09-30: review round; cull removed from alpha scope; target design and
  success metrics added as Beta criteria.

## Drawbacks

It adds state and a second delivery path to the watch cache, a component whose
correctness is critical, and in alpha it puts catch-up work on the dispatcher
goroutine. The complexity is justified by removing a force-close/reconnect
collapse that has no in-place mitigation today. It follows a model etcd has run
for years, it is contained behind an off-by-default gate whose disabled path
keeps its control flow, and the intent is to delete the old path once the new
one is proven.

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
- **Terminate clients that read nothing (a cull).** Culling a client whose
  channel is full and whose position never moves addresses dead connections
  but not the reconnect/replay collapse of healthy-but-behind watchers, which
  is the main motivation. It also adds a third watcher state, which etcd found
  costly in complexity for little gain and which etcd's cache library skips.
  It is out of scope for alpha; the request deadline bounds such a client as
  today, and the stalled-watchers gauge measures the population first.
- **A hard cap on catch-up work per watcher.** Bounding the number of catch-up
  rounds and terminating past it was considered; it re-introduces a force-close
  for slow-but-live clients and is unnecessary because aging out of the history
  window already provides an honest, time-denominated bound. Kept as a possible
  future safeguard, not required for the core behavior.
