# KEP-6412: Expose cgroup v2 memory.events OOM-kill counters in Kubelet/CRI Stats

<!-- toc -->
- [Release Signoff Checklist](#release-signoff-checklist)
- [Summary](#summary)
- [Motivation](#motivation)
  - [Goals](#goals)
  - [Non-Goals](#non-goals)
- [Proposal](#proposal)
  - [User Stories](#user-stories)
    - [Story 1: authoritative OOM-kill accounting for a controller](#story-1-authoritative-oom-kill-accounting-for-a-controller)
    - [Story 2: distinguishing a whole-cgroup kill from per-process kills](#story-2-distinguishing-a-whole-cgroup-kill-from-per-process-kills)
  - [Notes/Constraints/Caveats](#notesconstraintscaveats)
  - [Risks and Mitigations](#risks-and-mitigations)
- [Design Details](#design-details)
  - [Field set and roles](#field-set-and-roles)
  - [Why hierarchical <code>memory.events</code>](#why-hierarchical-memoryevents)
  - [Deferred fields](#deferred-fields)
  - [API change](#api-change)
  - [Stats provider scope](#stats-provider-scope)
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

Items marked with (R) are required *prior to targeting to a milestone / release*.

- [ ] (R) Enhancement issue in release milestone, which links to KEP dir in [kubernetes/enhancements] (not the initial KEP PR)
- [ ] (R) KEP approvers have approved the KEP status as `implementable`
- [ ] (R) Design details are appropriately documented
- [ ] (R) Test plan is in place, giving consideration to SIG Architecture and SIG Testing input (including test refactors)
  - [ ] e2e Tests for all Beta API Operations (endpoints)
  - [ ] (R) Ensure GA e2e tests meet requirements for [Conformance Tests](https://git.k8s.io/community/contributors/devel/sig-architecture/conformance-tests.md)
  - [ ] (R) Minimum Two Week Window for GA e2e tests to prove flake free
- [ ] (R) Graduation criteria is in place
  - [ ] (R) [all GA Endpoints](https://github.com/kubernetes/community/pull/1806) must be hit by [Conformance Tests](https://git.k8s.io/community/contributors/devel/sig-architecture/conformance-tests.md)
- [ ] (R) Production readiness review completed
- [ ] (R) Production readiness review approved
- [ ] "Implementation History" section is up-to-date for milestone
- [ ] User-facing documentation has been created in [kubernetes/website], for publication to [kubernetes.io]
- [ ] Supporting documentation e.g., additional design documents, links to mailing list discussions/SIG meetings, relevant PRs/issues, release notes

## Summary

cgroup v2 maintains a per-cgroup `memory.events` file: a set of kernel-maintained counters
(`low`, `high`, `max`, `oom`, `oom_kill`, and, on kernels 5.17+, `oom_group_kill`) recording how a cgroup
has interacted with its memory limits. The kubelet Summary API (the structured stats surface that
*in-cluster* consumers, including the kubelet's own decision loop, read) exposes none of them.
cAdvisor exposes some of these keys, but only as Prometheus metrics on `/metrics/cadvisor`, which is
reachable by an external scraper and not by anything running inside the kubelet (see
[Motivation](#motivation)).

This KEP adds two of those counters, `oom_kill` and `oom_group_kill`, to the kubelet
Summary API (`stats/v1alpha1`) per container, and a corresponding `MemoryEvents` message to CRI so
both the cAdvisor and CRI stats providers can supply them. It is read-only and on-demand: the fields
are computed when `/stats/summary` is served, nothing is written to etcd, and no kubelet scheduling,
eviction or lifecycle behaviour changes.

The two counters are the subset with an identified reader on the structured stats path:

- `oom_kill`: authoritative per-cgroup OOM-kill count (one per process killed), replacing cAdvisor's
  lossy kmsg-derived `container_oom_events_total`.
- `oom_group_kill`: cgroup-level (group) OOM-kill count, pairing with `memory.oom.group`; a whole-
  cgroup kill counts as one event rather than N.

`high`, `max`, `low` and `oom` are deferred until a stats-path consumer exists; `high`/`max` are
already on cAdvisor's `/metrics/cadvisor` for raw-counter needs (see [Deferred fields](#deferred-fields)).

The feature is guarded by the `KubeletMemoryEvents` feature gate (kubelet, default off) and enters
at **Alpha**. This mirrors [KEP-4205 (PSI)](https://github.com/kubernetes/enhancements/tree/master/keps/sig-node/4205-psi-metric),
which added `PsiStats psi = 8` to the same CRI `MemoryUsage` message and took PSI through
Alpha, Beta, and GA under the `KubeletPSI` gate. `memory.events` is the counting complement to PSI's
time-based pressure stalls, and `events = 9` is the next field on that message.

## Motivation

**The kubelet's own decision loop cannot read cAdvisor's Prometheus endpoint.** Everything the
kubelet acts on, eviction today and per-container memory-pressure remediation tomorrow
([KEP-5986](https://github.com/kubernetes/enhancements/pull/5986), the parent this KEP split from),
is driven off the Summary/CRI stats pipeline, not `/metrics/cadvisor`. The eviction manager gets its
data from `summaryProvider.Get(...)` in `pkg/kubelet/eviction/eviction_manager.go` and makes every
threshold decision through `makeSignalObservations(summary)` over the resulting `statsFunc`; the only
cAdvisor reference on that path is a housekeeping-timing comment, never a metric read. The Summary it
consumes is built from `ListPodCPUAndMemoryStats` into `statsapi.MemoryStats`
(`pkg/kubelet/server/stats/summary.go`), the struct that already carries `PSI` and would carry
`Events`.

So `/metrics/cadvisor` (where cAdvisor exposes `memory.events` today) is reachable only by an external
scraper and is invisible to the kubelet loop. For any in-cluster consumer to act on `memory.events`,
the counters must be on the CRI/Summary surface, exactly why PSI lives there (KEP-4205) though cAdvisor
also exposes `container_pressure_memory_*` from the same kernel file.

The OOM-kill counters have an identified reader:

- **`oom_kill` / `oom_group_kill`: authoritative kill counts the structured path lacks.** The number
  of times a container's processes were OOM-killed is a first-class reliability signal for alerting,
  crash-loop diagnosis and right-sizing. The only structured source today is cAdvisor's
  `container_oom_events_total`, produced from an OOM watcher that parses the kernel log; log parsing is
  lossy: under pressure the kernel rate-limits its own log, messages rotate out of a bounded ring
  buffer, and the text varies across kernel versions and cgroup drivers, so a consumer can silently
  undercount, the worst failure mode for a reliability signal. The kernel already keeps an exact count
  in `memory.events`, with no log involved. This also fills a gap the existing autoscaling path has:
  VPA's OOM observer
  (`vertical-pod-autoscaler/pkg/recommender/input/oom/observer.go`) fires only when a container's
  status reaches `Terminated.Reason == "OOMKilled"` with a restart, so it *misses* per-process and
  group kills that do not terminate the container; `memory.events` counts those.

These are counts, complementary to PSI, which measures the *time* a workload spent stalled on memory.
A controller uses PSI for severity and these counters for "did a kill happen, and how many times".

### Goals

- Expose the `oom_kill` and `oom_group_kill` counters from cgroup v2 `memory.events`
  per container through the kubelet Summary API, from both the cAdvisor and the CRI stats providers,
  so the fields do not depend on how a node is configured to source container stats.
- Give an in-cluster consumer on the structured stats path, the kubelet's own decision loop
  included, an authoritative, non-log-derived OOM-kill count that the Summary surface does not
  provide today.

### Non-Goals

- Any eviction, throttle detection, or remediation behaviour in the kubelet. This KEP exposes data
  and nothing else. Per-container memory-pressure eviction is
  [kubernetes/enhancements#5986](https://github.com/kubernetes/enhancements/pull/5986), out of
  scope here; the conclusion on
  [kubernetes/enhancements#6141](https://github.com/kubernetes/enhancements/pull/6141) is that
  detection and response belong in a controller outside the kubelet. If such eviction is built, it
  should key on PSI (which measures severity), with `memory.events` as the corroborating/audit
  layer, never the trigger.
- Exposing `high`, `max`, `low` or `oom` from `memory.events`. Deferred until a concrete stats-path
  consumer exists (see [Deferred fields](#deferred-fields)).
- Exposing the effective `memory.high` value. Useful, but a separate change with a different owner
  surface; discussed in [Alternatives](#alternatives).
- Windows. `memory.events` is a Linux cgroup v2 interface. On Windows nodes the fields are never
  populated and no Windows-side equivalent is proposed.
- cgroup v1. The `memory.events` file is a cgroup v2 interface; on cgroup v1 nodes the fields are
  never populated.

## Proposal

Add an optional `Events` field to `MemoryStats` in
`staging/src/k8s.io/kubelet/pkg/apis/stats/v1alpha1`, holding the two OOM-kill counters as `*uint64`
(the `Events` object is nil when the source reports no `memory.events`). The cAdvisor stats provider populates them from the cAdvisor `MemoryEvents` struct
(extended by this KEP to read the `oom_kill` / `oom_group_kill` keys of the same `memory.events` file
it already reads for `high`/`max`); the CRI stats provider populates them from a new optional
`MemoryEvents` message on the CRI `MemoryUsage`, reported by the container runtime. Everything is guarded by the `KubeletMemoryEvents` gate, mirroring
exactly where `KubeletPSI` guards PSI on the same paths.

### User Stories

#### Story 1: authoritative OOM-kill accounting for a controller

A reliability controller tracks how often each container's processes are OOM-killed to drive
alerting and right-sizing. Today it must scrape cAdvisor's kmsg-derived
`container_oom_events_total`, which can miss kills when kernel log messages are rate-limited or
rotated, so its counts drift low exactly when the node is under the most pressure. Reading
`Events.OOMKill` from the Summary API gives it the kernel's own exact count, on the same stats path
it already consumes, with no log parsing. The value is exact within a container's lifetime; because
it resets on restart and the stats path reports only running containers, the controller accumulates
across container generations rather than treating one sample as a lifetime total (see
[Notes/Constraints/Caveats](#notesconstraintscaveats)).

#### Story 2: distinguishing a whole-cgroup kill from per-process kills

A pod with `memory.oom.group` set is killed as a unit when it exceeds its limit. An operator wants
to distinguish "this pod was group-OOM-killed once" from "several of its processes were individually
killed", because the two imply different root causes and remediations. `Events.OOMGroupKill` counts
the group kills and `Events.OOMKill` counts the individual process kills, so the operator can tell
them apart directly from `/stats/summary`.

### Notes/Constraints/Caveats

**`oom_kill` counts process kills; `oom_group_kill` counts group kills.** The kernel increments
`oom_kill` once per process killed, so a single OOM event that kills several processes in a cgroup
increments it by more than one. When `memory.oom.group` is set, the same event increments
`oom_group_kill` by one. A consumer should read `oom_group_kill` when it wants "how many times was
the workload killed" and `oom_kill` when it wants "how many processes died".

**`oom_group_kill` requires a recent kernel; an absent key reads as zero.** The `oom_group_kill` key
was added to `memory.events` in Linux 5.17; on older kernels the key is absent. Both producer
libraries collapse a missing key to zero (containerd reads `memory.events` into a `map[string]uint64`
and takes `m["oom_group_kill"]`; `opencontainers/cgroups` `GetValueByKey` returns `(0, nil)`), and
containerd's CRI plugin sees only the already-collapsed cgroup2 metrics proto (plain `uint64`), so
neither runtime can distinguish "key absent" from "genuine zero" without an upstream
`containerd/cgroups` change. The KEP therefore reports `oom_group_kill` as the kernel's value, which
is `0` on a pre-5.17 kernel. A consumer that must tell "unsupported" from "genuinely zero" checks the
node kernel version. `oom_kill` is the backstop here: a `memory.oom.group` kill also increments
`oom_kill`, and `oom_kill` is present on every cgroup v2 kernel Kubernetes runs on, so a group kill on
an old kernel is still counted there.

**Counters are per-container, observed only while the container runs, and reset on restart.** Each
counter lives on the container's own cgroup and starts at zero on restart; the stats providers report
only running containers (`removeTerminatedContainers` keeps `CONTAINER_RUNNING` and drops the rest).
So a consumer polling `/stats/summary` must reset its accumulator when the value drops (a new cgroup),
can miss the final increment of a container that exits between samples, and cannot reconstruct a
complete cross-restart total from the sampled values alone; within a running container's lifetime the
value is the kernel's exact count. This KEP adds no pod-level aggregate; a pod total is the consumer's
sum across container generations.

**A zero `oom_kill` means "no kill"; a zero `oom_group_kill` may also mean "old kernel".** On a
cgroup v2 node with the feature enabled, a zero `oom_kill` genuinely means no OOM kill has occurred.
A zero `oom_group_kill` means either no group kill or a pre-5.17 kernel where the key is absent (see
above); a consumer distinguishes the two by the node kernel version. Absence of the whole `Events`
object (nil), by contrast, means the node is cgroup v1, the runtime does not report it (strict-CRI
path), or the gate is off.

### Risks and Mitigations

**Risk: the CRI field lands before runtimes populate it.** A kubelet running strictly from CRI
(`PodAndContainerStatsFromCRI` enabled) leaves the fields unset until containerd or CRI-O reports the
new `MemoryEvents` message. That gate is off by default; on every default configuration the cAdvisor
overlay supplies the values, so the exposure is limited to clusters that deliberately opted into
strict CRI stats. Consumers must already tolerate absent optional fields here (the same happens on
cgroup v1 and when the gate is off), so this degrades rather than breaks.

**Risk: a consumer cannot tell whether the runtime supports the field.** There is no capability
discovery for CRI stats fields today, so an unset value is ambiguous between "runtime does not
report it", "cgroup v1", and "old kernel" (for `oom_group_kill`). This is the same ambiguity that
already applies to PSI on the CRI path. Mitigation is documentation: the fields are advisory; a
consumer needing certainty on the default path can cross-check the node's cgroup file or
`/metrics/cadvisor`.

There is no security or privilege risk. The data is a read-only kernel counter for a cgroup the
kubelet already owns, served to the same authenticated and authorized callers that already reach
`/stats/summary`.

## Design Details

### Field set and roles

Each exposed counter earns its place by having an identified reader on the structured stats surface.
This is the whole design: the message is not a dump of `memory.events`, it is the subset with a
justified reader on the CRI/Summary path.

| Counter | Role | Consumer / justification |
|---|---|---|
| `oom_kill` | **Authoritative** per-cgroup OOM-kill count (per process killed). | Reliability controllers / right-sizing / VPA; replaces cAdvisor's lossy kmsg-derived `container_oom_events_total` and catches kills VPA's `OOMKilled`-status path misses. |
| `oom_group_kill` | **Authoritative** cgroup-level (group) OOM-kill count; pairs with the `memory.oom.group` cgroup setting. | Same consumer, distinguishing one whole-cgroup kill from N process kills; newer-kernel (5.17+), absent-tolerant. |

`high`, `max`, `low` and `oom` are deferred (see below).

### Why hierarchical `memory.events`

The counters are read from the hierarchical `memory.events`, not `memory.events.local`. The two
differ only when a cgroup has descendants; a container's cgroup is a leaf, so they are identical for
it and the choice changes no value. Hierarchical is also what every existing collector already reads
(cAdvisor's `setMemoryEvents`, containerd's `readMemoryEvents`, CRI-O's `OOMKillCount`), so none has
to change which file it opens; implementing this KEP only adds the `oom_kill` / `oom_group_kill` keys
to reads they already perform. cAdvisor's `MemoryEvents` struct currently carries only `High` and
`Max`, so the OOM keys are genuine new collection, the same reason PSI (KEP-4205) entered at Alpha.
Specifying `.local` would force all three to change for an identical value; if pod-level aggregation
is ever added it will sum per-container values explicitly, a separate design not needed here.

### Deferred fields

`high`, `max`, `low` and `oom` from `memory.events` are intentionally not added now. They are cheap to
add later (one more `UInt64Value` on the message and one more key read from the same file), but adding
a field with no reader is the "dump the cgroup file" anti-pattern this design avoids. Each will be
added when a concrete stats-path consumer exists, for example:

- `high` / `max`: throttle- and limit-breach counters, already on cAdvisor's `/metrics/cadvisor` as
  `container_memory_events_{high,max}_total`. Nothing on the structured stats path reads them:
  MemoryQoS (KEP-2570) *writes* `memory.high` and never reads the counter; the eviction loop and
  [#5986](https://github.com/kubernetes/enhancements/pull/5986) use PSI for severity; and VPA's
  reactive AEP ([AEP-9926](https://github.com/kubernetes/autoscaler/issues/9926)) makes `memory.events`
  a Non-Goal because `events:high` inverts under distress. Add them if a stats-path reader appears.
- `low`: signals that protected memory (`memory.low`) was reclaimed under pressure. It becomes the
  health signal for the emerging MemoryQoS reservation policies (`Soft` / `TieredReservation`) and
  should land alongside that work, which gives it a reader.
- `oom`: the OOM *condition* count (not necessarily a kill), the weakest of the counters; only
  worth adding if a consumer needs the condition/kill distinction that `oom_kill` alone does not
  give.

Adding a field is a backward-compatible proto/struct change (a new optional field), so deferring
costs nothing in compatibility.

### API change

Summary API. Add an optional `Events` field to `MemoryStats`, mirroring the existing
`PSI *PSIStats` field:

```go
// MemoryStats contains data about memory usage.
type MemoryStats struct {
	// ... existing fields ...

	// Memory PSI stats.
	// +optional
	PSI *PSIStats `json:"psi,omitempty"`

	// Selected cgroup v2 memory.events counters for this container.
	// Only populated on cgroup v2 nodes when the KubeletMemoryEvents feature is enabled.
	// +optional
	Events *MemoryEvents `json:"events,omitempty"`
}

// MemoryEvents holds selected cgroup v2 memory.events counters. The enclosing Events pointer is nil
// when the source reports no memory.events (cgroup v1, or a runtime that does not report it).
type MemoryEvents struct {
	// OOMKill counts memory.events "oom_kill" (one per process killed); authoritative, unlike the
	// log-derived container_oom_events_total.
	// +optional
	OOMKill *uint64 `json:"oomKill,omitempty"`

	// OOMGroupKill counts memory.events "oom_group_kill" (one per whole-cgroup kill, with
	// memory.oom.group). Requires Linux 5.17+; reads as 0 on older kernels.
	// +optional
	OOMGroupKill *uint64 `json:"oomGroupKill,omitempty"`
}
```

CRI. `MemoryUsage` in `staging/src/k8s.io/cri-api` currently ends at `PsiStats psi = 8` (verified
in `pkg/apis/runtime/v1/api.proto`). This KEP adds `MemoryEvents events = 9`, using `UInt64Value`
wrappers so a counter a runtime does not populate stays distinct from a real zero:

```protobuf
// MemoryEvents carries selected cgroup v2 memory.events counters. Each field is a
// UInt64Value so a runtime can omit a counter it does not populate, distinct from a real zero.
message MemoryEvents {
    // oom_kill: authoritative kernel count of OOM kills in this cgroup (one per process killed).
    // Preferred over log-derived OOM counts, which can undercount.
    UInt64Value oom_kill = 1;
    // oom_group_kill: cgroup-level (group) OOM kills, paired with memory.oom.group (one per
    // whole-cgroup kill). Requires Linux 5.17+; reads as 0 on older kernels where the key is absent.
    UInt64Value oom_group_kill = 2;
}
```

```protobuf
message MemoryUsage {
    // ... existing fields through PsiStats psi = 8 ...
    // Selected cgroup v2 memory.events counters.
    MemoryEvents events = 9;
}
```

Populated in the kubelet at the same sites that already handle PSI, all guarded on
`KubeletMemoryEvents` exactly as they are guarded on `KubeletPSI` (verified for PSI):

- cAdvisor path: `cadvisorInfoToCPUandMemoryStats` in `pkg/kubelet/stats/helper.go`, inside the
  existing `info.Spec.HasMemory && cstat.Memory != nil` block, alongside
  `memoryStats.PSI = cadvisorPSIToStatsPSI(...)`.
- CRI path: `makeContainerCPUAndMemoryStats` in `pkg/kubelet/stats/cri_stats_provider.go`, in the
  `stats.Memory != nil` block alongside `makePSIStats(stats.Memory.Psi)`, plus the Linux CRI
  pod-stats path in `pkg/kubelet/stats/cri_stats_provider_linux.go`.
- Summary surfacing: `pkg/kubelet/server/stats/summary.go`, which is where PSI is gated for the
  node-level surface.

The conversion follows the existing `makePSIStats` / `valueOfUInt64Value` shape (verified): both
return nil when the source proto message or wrapper is nil, which is how an absent field stays nil
rather than becoming zero.

### Stats provider scope

Both stats providers are in scope. A field present under one stats configuration and absent under
another is not something a consumer can rely on, so this KEP covers the CRI path as well as the
cAdvisor path, as PSI, the closest analogue in the same struct, already does.

The CRI stats provider is hybrid today. `listPodStats` branches on `useCRIPodSandboxStats`, set from
the `PodAndContainerStatsFromCRI` feature gate (Beta since v1.37, default **off**):

- Gate off, the default: the CRI provider overlays cAdvisor data through
  `addCadvisorContainerCPUAndMemoryStats`, which calls the same `cadvisorInfoToCPUandMemoryStats`
  conversion as the cAdvisor provider (verified in `cri_stats_provider.go`). These fields are
  therefore populated on this path by the cAdvisor change alone.
- Gate on: the provider runs strictly from CRI with no cAdvisor overlay. This is the configuration
  that requires the CRI `MemoryEvents` message, and the direction the cAdvisor-less work
  ([KEP-2371](https://github.com/kubernetes/enhancements/issues/2371)) is moving toward.

So coverage is complete for the default configuration once the cAdvisor change lands; the CRI
message closes the strict-CRI configuration.

**Runtime implementation is a dependency, not a blocker.** Adding the field to CRI does not by
itself make runtimes report it. The runtime populates the `MemoryEvents` message on cgroup v2 nodes
(nil on cgroup v1): `oom_kill` always, and `oom_group_kill` as the kernel's value, which is `0` on a
pre-5.17 kernel where the key is absent. It does no gating of its own; it has no knowledge of
`KubeletMemoryEvents`. All gating lives in the kubelet's consumption path, the same split as PSI:
containerd sets `PsiStats psi = 8` with no feature check, and the kubelet gates the read behind
`KubeletPSI`. The current runtime state (verified):

- containerd's CRI plugin builds `MemoryUsage` from the cgroup2 metrics proto (`cg2.Metrics`)
  returned by the task shim (`container_stats_list.go`), whose `MemoryEvents` sub-message already
  carries `oom_kill` and, with `containerd/cgroups v3.1.3`, `oom_group_kill`, as plain `uint64`.
  `v3.1.0` and earlier (still vendored by some containerd releases) lack the `oom_group_kill` proto
  field and need the cgroups-lib bump. Populating the new CRI `MemoryEvents` message is a field copy
  in the CRI stats conversion; the shim's `readMemoryEvents` has already collapsed an absent key to
  `0`, so containerd reports that value.
- CRI-O builds `MemoryUsage` in its stats server directly from cgroup files (`criMemStats` in
  `stats_server_linux.go`); it reads `oom_kill` today only for its Prometheus `/metrics` endpoint
  (`OOMKillCount`, a separate path), not for CRI stats, so it adds a read of `memory.events` at the
  CRI stats site for both keys. `opencontainers/cgroups` `GetValueByKey` returns `(0, nil)` for a
  missing key, so `oom_group_kill` reads as `0` on a pre-5.17 kernel, matching containerd.

Both changes are small (containerd needs the cgroups-lib bump only if it vendors pre-`v3.1.3`). Until
a runtime reports the message, a strict-CRI kubelet leaves the fields nil, the same handling a cgroup
v1 node already requires; skew in both directions is safe (see [Version Skew Strategy](#version-skew-strategy)).

### Test Plan

[x] I/we understand the owners of the involved components may require updates to existing tests to
make this code solid enough prior to committing the changes necessary to implement this enhancement.

##### Prerequisite testing updates

None.

##### Unit tests

- `pkg/kubelet/stats`: `cadvisorInfoToCPUandMemoryStats` populates `Events.OOMKill` and
  `Events.OOMGroupKill` from the cAdvisor struct, and leaves `Events` nil when the source is absent.
- `pkg/kubelet/stats`: a guard test asserting the shape of the cAdvisor `MemoryEvents` struct, so a
  change to it fails the build rather than silently changing the conversion (mirrors the existing
  `TestCadvisorPSIStructs`).
- `pkg/kubelet/stats`: with `KubeletMemoryEvents` disabled, `Events` is nil on both providers.
- `pkg/kubelet/stats`: `makeContainerCPUAndMemoryStats` populates the fields when the fake runtime
  reports the CRI `MemoryEvents` message, and leaves the whole `Events` object nil when it does not,
  including the case where the runtime reports `oom_kill` with `oom_group_kill` at `0` (pre-5.17
  kernel).

`pkg/kubelet/stats` unit coverage figure to be recorded at implementation time.

##### Integration tests

Not applicable. The kubelet Summary API is covered by node e2e rather than integration tests.

##### e2e tests

- `test/e2e_node/summary_test.go`: extend the Summary API structure test so the new fields are
  matched. The existing matchers use `ptrMatchAllFields`, which is exhaustive, so this is required
  rather than optional.
- `test/e2e_node`: on a cgroup v2 node, a pod whose container exceeds its memory limit and is
  OOM-killed reports a non-zero `Events.OOMKill` through the Summary API; with `memory.oom.group`
  in effect, `Events.OOMGroupKill` is non-zero as well (skipped on pre-5.17 kernels).

### Graduation Criteria

This feature enters at **Alpha** behind `KubeletMemoryEvents` (default off), following the KEP-4205
(PSI) trajectory for the same `MemoryUsage` stats field.

#### Alpha

- `KubeletMemoryEvents` gate added (kubelet, default off).
- `Events` (`oom_kill`, `oom_group_kill`) populated by the cAdvisor stats provider on
  cgroup v2 nodes, from the hierarchical `memory.events`.
- CRI `MemoryEvents events = 9` message added and populated by the CRI stats provider when the
  runtime reports it.
- Unit tests for both providers, including gate-disabled (`Events` nil) and a pre-5.17 kernel where
  `oom_group_kill` reads as `0`.
- Node e2e for the cAdvisor path (`oom_kill`).

#### Beta

- At least one container runtime (containerd or CRI-O) reports the CRI `MemoryEvents` message, so the
  strict-CRI path is exercised in practice.
- Node e2e covers the strict-CRI path and `oom_group_kill` on a 5.17+ kernel.
- Gate defaults to on; monitoring/SLI guidance filled in (see (Beta) stubs below).

#### GA

- No outstanding bugs against the fields for two releases.
- Both containerd and CRI-O report the CRI message.
- Conformance considered for the Summary API surface if applicable.

### Upgrade / Downgrade Strategy

No action is required on upgrade or downgrade, and no node drain. On upgrade with the gate enabled
the fields begin appearing in Summary API responses on cgroup v2 nodes. On downgrade, or with the
gate disabled, they are omitted. Toggling the gate only changes whether the kubelet reads a live
kernel counter when serving `/stats/summary`; nothing is persisted, so nothing is lost on downgrade.
Consumers must already tolerate absent optional fields in this API.

### Version Skew Strategy

Two skew surfaces, neither requiring coordination.

**Kubelet against its consumers.** The Summary API is served by the kubelet and consumed out of
process. A consumer written against a newer kubelet sees the fields omitted when talking to an older
one, the same handling already required for a cgroup v1 node or a node with the gate disabled.

**Kubelet against the container runtime** (only when `PodAndContainerStatsFromCRI` is enabled).
`MemoryEvents events = 9` is an optional proto3 message on an existing message:

- New kubelet, old runtime: field 9 is absent on the wire and decodes to a nil `MemoryEvents`
  (proto3 message absence is nil, not zero; verified: the kubelet's `makePSIStats`-style
  conversion returns nil on a nil message, and `valueOfUInt64Value` returns nil on a nil wrapper).
  The kubelet leaves the Summary fields nil.
- Old kubelet, new runtime: field 9 is an unknown field and is ignored by the old kubelet's decoder
  (proto3 forward compatibility).

Both directions are safe and no minimum runtime version is required. With the gate off (the default)
the cAdvisor overlay supplies the values in-process and there is no runtime skew at all.

## Production Readiness Review Questionnaire

### Feature Enablement and Rollback

###### How can this feature be enabled / disabled in a live cluster?

- [x] Feature gate (also fill in values in `kep.yaml`)
  - Feature gate name: `KubeletMemoryEvents`
  - Components depending on the feature gate: kubelet
- [ ] Other

###### Does enabling the feature change any default behavior?

No. When enabled, an optional `Events` object begins appearing on `MemoryStats` in Summary API
responses on cgroup v2 nodes. Nothing in the kubelet reads it, and no kubelet behaviour (eviction,
scheduling, lifecycle) changes.

###### Can the feature be disabled once it has been enabled (i.e. can we roll back the enablement)?

Yes, by disabling the gate and restarting the kubelet. The fields are then omitted. Nothing persists
and no state is left behind; rollback loses nothing because nothing is written anywhere.

###### What happens if we reenable the feature if it was previously rolled back?

The fields reappear. The underlying counters are maintained by the kernel for the lifetime of each
cgroup and are unaffected by kubelet configuration, so values are continuous across the disable
window for containers that were not restarted.

###### Are there any tests for feature enablement/disablement?

Yes, unit tests asserting `Events` is nil on both stats providers when the gate is disabled, and
populated when it is enabled.

### Rollout, Upgrade and Rollback Planning

###### How can a rollout or rollback fail? Can it impact already running workloads?

It cannot impact running workloads. The change is confined to serialization of a read-only stats
response; toggling the gate only changes whether the kubelet copies an already-collected kernel
counter into the response. The kernel maintains the counter and cAdvisor or the runtime reads it
regardless of the gate, so the gate controls consumption, not collection. The one
plausible failure is a consumer that cannot tolerate a new optional field in the Summary API
response, which would be a pre-existing defect in that consumer, since this API already gains fields
between releases.

###### What specific metrics should inform a rollback?

None specific to this feature. A kubelet that fails to serve `/stats/summary` would show up in
existing kubelet health signals and in consumers of that endpoint such as metrics-server.

###### Were upgrade and rollback manually tested? Is the upgrade/rollback failure impact tested?

To be completed prior to Beta. The test is to enable the gate, confirm the fields appear on a cgroup
v2 node, disable it, and confirm they are omitted.

###### Is the rollout accompanied by any deprecations and/or removals of features, APIs, fields of API types, flags, etc.?

No.

### Monitoring Requirements

###### How can an operator determine if the feature is in use by workloads?

Query a node's `/stats/summary` and observe whether `memory.events` is present on container entries.
Presence indicates the gate is enabled and the node is cgroup v2.

###### How can someone using this feature know that it is working for their instance?

- [x] Other (treat as manual verification for Alpha)
  - Query `/stats/summary` on a cgroup v2 node with the gate enabled and confirm the container's
    `memory.events.oomKill` field is present, then cross-check against the kernel by running
    `cat /sys/fs/cgroup/<container-cgroup-path>/memory.events` on the node; the Summary value
    should match the `oom_kill` (and, on 5.17+ kernels, `oom_group_kill`) key. Deliberately force an
    OOM kill (e.g. a pod that allocates past its limit) and confirm the counter increments.

###### What are the reasonable SLOs (Service Level Objectives) for the enhancement?

_(Beta)_ To be defined at Beta. The values are read on the existing stats-collection path, so the
objective will be expressed as agreement with the underlying kernel counter and no added
`/stats/summary` latency, rather than a separate availability target.

###### What are the SLIs (Service Level Indicators) an operator can use to determine the health of the service?

_(Beta)_ For Alpha, the Summary value can be compared against `memory.events` on the node and,
once cAdvisor exposes it, against the new authoritative OOM-kill metric (see `kep.yaml`). Dedicated
SLIs are Beta work.

###### Are there any missing metrics that would be useful to have to improve observability of this feature?

Beta adds two Prometheus counters (`container_memory_events_oom_kill_total` and
`container_memory_events_oom_group_kill_total`) sourced from `memory.events`, so they are
authoritative unlike the existing kmsg-derived `container_oom_events_total`. The counters and their
meaning are fixed (they mirror the Summary fields); the open Beta decision is only the *emitter*:
extending cAdvisor's existing `container_memory_events_{high,max}_total` family (it already reads
`memory.events`, so this is the natural home) versus a kubelet-native metric off the Summary stats.
cAdvisor is the default choice unless its metrics are being retired by then. Alpha ships the Summary
field; the Prometheus counter is Beta work.

### Dependencies

###### Does this feature depend on any specific services running in the cluster?

No cluster services. Dependencies are node-local:

- cgroup v2. `memory.events` is a cgroup v2 interface; on cgroup v1 the fields are never populated.
- On the cAdvisor path: the in-process cAdvisor library the kubelet already vendors (extended by
  this KEP to read the `oom_kill` / `oom_group_kill` keys alongside the `high` / `max` it already
  reads).
- On the strict-CRI path: a container runtime (containerd / CRI-O) that reports the CRI
  `MemoryEvents` message. Until a runtime does, the fields are simply unset; a runtime outage or
  degradation has no impact beyond the existing impact of a runtime outage on stats.
- `oom_group_kill` additionally requires Linux 5.17+; on older kernels the key is absent and the
  field reads as `0`.

Absent on cgroup v1, an old runtime, or an old kernel is "unknown by design", not an error (see
[Notes/Constraints](#notesconstraintscaveats)).

###### Are there any feature interactions worth documenting?

`oom_group_kill` is meaningful only when `memory.oom.group` is set on the cgroup; otherwise it stays
zero while `oom_kill` still counts individual process kills. Beyond that, the fields are read-only and
derived from a kernel counter
for the cgroup the kubelet already owns, so they do not interact with the kubelet's isolation or
security features: user namespaces (KEP-127), seccomp and AppArmor do not change which cgroup a
container is in or what the kernel writes to `memory.events`, and Pod Security Standards are not
relevant since nothing here is settable by a workload author.

### Scalability

###### Will enabling / using this feature result in any new API calls?

No. There are no apiserver reads, writes, or watches; the data is served by the kubelet on the
existing `/stats/summary` path.

###### Will enabling / using this feature result in introducing new API types?

Yes, two new stats-only types, neither apiserver- nor etcd-backed: a `MemoryEvents` struct on the
kubelet Summary API (`statsapi.MemoryStats.Events`) and a `MemoryEvents` protobuf message on the CRI
API (embedded in `MemoryUsage.events`). Both hold the same two optional counters, `oom_kill` and
`oom_group_kill`, attached to the existing per-container memory stats.

###### Will enabling / using this feature result in any new calls to the cloud provider?

No.

###### Will enabling / using this feature result in increasing size or count of the existing API objects?

No etcd-backed API objects change. The kubelet Summary API response grows by one optional object of
two `uint64` fields per container (on the order of tens of bytes of JSON per container). The CRI
`ListContainerStats` response grows by one optional message of two `UInt64Value` fields per
container.

###### Will enabling / using this feature result in increasing time taken by any operations covered by existing SLIs/SLOs?

Negligibly. The collector already reads `memory.events` for `high`/`max`, so the OOM keys are parsed
from a file it already opens, with no additional file read per container, plus assigning a couple of
pointers during serialization. There are no apiserver operations involved.

###### Will enabling / using this feature result in non-negligible increase of resource usage (CPU, RAM, disk, IO, ...) in any components?

No. The `memory.events` file is already read on the stats path; this parses additional keys from it.

###### Can enabling / using this feature result in resource exhaustion of some node resources (PIDs, sockets, inodes, etc.)?

No.

### Troubleshooting

_(Beta: full Failure Modes / Troubleshooting is Beta work; the Alpha verification path is under
[Monitoring Requirements](#monitoring-requirements).)_

###### How does this feature react if the API server and/or etcd is unavailable?

Not applicable. The Summary API is served by the kubelet and does not depend on the API server.

###### What are other known failure modes?

_(Beta)_ The expected-absence cases are already documented above: fields absent on cgroup v1 / old
runtime / old kernel (`oom_group_kill`) are by design, not failures. If a value is present but
disagrees with `memory.events` on the node, the fault is in the conversion in
`pkg/kubelet/stats/helper.go`; this will be expanded into a full Failure Modes table at Beta.

###### What steps should be taken if SLOs are not being met to determine the problem?

_(Beta)_ Compare the Summary value against `cat /sys/fs/cgroup/<path>/memory.events` on the
node. Detailed steps are Beta work.

## Implementation History

- 2026-03-16: first attempt to expose `memory.events` counters through CRI, closed unmerged
  ([kubernetes/kubernetes#137760](https://github.com/kubernetes/kubernetes/pull/137760)). It
  proposed a CRI field with neither a cAdvisor path nor a KEP; both gaps are closed here.
- 2026-05-13: cAdvisor exposes the `high`/`max` keys as Prometheus metrics
  ([google/cadvisor#3870](https://github.com/google/cadvisor/pull/3870)).
- 2026-05-20: cAdvisor bump brings those metrics into Kubernetes v1.37
  ([kubernetes/kubernetes#139157](https://github.com/kubernetes/kubernetes/pull/139157)).
- 2026-05-26: Summary API implementation opened
  ([kubernetes/kubernetes#139310](https://github.com/kubernetes/kubernetes/pull/139310)).
- 2026-09-16: KEP opened, split out of
  [kubernetes/enhancements#6141](https://github.com/kubernetes/enhancements/pull/6141) so the
  observability change is decoupled from the per-container eviction design proposed there
  ([#5986](https://github.com/kubernetes/enhancements/pull/5986)).
- 2026-09-22: scope set to Alpha behind `KubeletMemoryEvents`; field set fixed to the two OOM-kill
  counters with an identified stats-path reader (`oom_kill`, `oom_group_kill`), reading the
  hierarchical `memory.events` (matching all three existing collectors); `high`, `max`, `low` and
  `oom` deferred.

## Drawbacks

The CRI half of this KEP cannot be fully delivered by Kubernetes alone: the field can be added to
the API, but containerd and CRI-O each have to report the `MemoryEvents` message before a
CRI-sourced kubelet reports anything. The runtime work is small (see
[Stats provider scope](#stats-provider-scope)), but until both do it, the Summary API is uniform in
schema but not in population.

The feature adds a second structured source of OOM-kill counts alongside cAdvisor's existing
kmsg-derived metric. That is deliberate, the kernel counter is authoritative and the kmsg one is
lossy (see [Motivation](#motivation)), but two sources that usually agree and occasionally do not
is itself something operators have to understand, which is a documentation cost.

## Alternatives

**Consume `container_oom_events_total` from `/metrics/cadvisor` instead.** The status quo, and what
this KEP improves on: it's kmsg-derived so it undercounts under pressure, and `/metrics/cadvisor`
isn't reachable by the kubelet's own decision loop or other stats-path consumers
(see [Motivation](#motivation)).

**Expose more of the `memory.events` counters: all six, or just `high`/`max`.** Rejected as a "dump
the cgroup file" design: none of `high`/`max`/`low`/`oom` has a stats-path reader today, and
`high`/`max` are already on `/metrics/cadvisor`. The per-field reasoning, including why VPA's
AEP-9926 rejects `events:high`, is in [Deferred fields](#deferred-fields); each is added when a
reader exists.

**Expose the effective `memory.high` value.** Orthogonal to `memory.events` counting and a separate
change with a different owner surface (better proposed as a cAdvisor gauge alongside the existing
`container_spec_memory_*` series and listed in KEP-2570's metrics). Discussed here only to scope it
out.

**Add a pod condition indicating an OOM kill.** Cheaper for a controller to consume, but rejected
for now: a permanent core/v1 API addition should follow a demonstrated consumer rather than precede
one, and the counter on the existing stats path is enough for the consumers identified here.

**Fold this into KEP-2570.** KEP-2570 already lists cAdvisor memory-events metrics among its own and
`memory.events` is the observability surface for the feature it defines. It is kept separate because
KEP-2570 is at Beta and adding API surface to a Beta feature would be new scope on a graduating KEP.
KEP-2570's Monitoring Requirements can reference this work without owning it.
