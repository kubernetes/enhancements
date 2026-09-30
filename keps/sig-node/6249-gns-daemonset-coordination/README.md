# KEP-6249: Publish Graceful Node Shutdown state for DaemonSet coordination

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
    - [Story 4](#story-4)
  - [Notes/Constraints/Caveats](#notesconstraintscaveats)
  - [Risks and Mitigations](#risks-and-mitigations)
- [Design Details](#design-details)
  - [The conditions](#the-conditions)
  - [Kubelet (writer)](#kubelet-writer)
  - [DaemonSet controller (reader)](#daemonset-controller-reader)
  - [Other writers](#other-writers)
  - [Feature gating](#feature-gating)
  - [Metrics](#metrics)
  - [Design Decisions](#design-decisions)
  - [Test Plan](#test-plan)
      - [Prerequisite testing updates](#prerequisite-testing-updates)
    - [Unit tests](#unit-tests)
    - [Integration tests](#integration-tests)
    - [e2e tests](#e2e-tests)
  - [Graduation Criteria](#graduation-criteria)
    - [Alpha (v1.38)](#alpha-v138)
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
- [Infrastructure Needed (Optional)](#infrastructure-needed-optional)
<!-- /toc -->

This KEP is a standalone follow-up to [KEP-5683: Node Lifecycle
Conditions](https://github.com/kubernetes/enhancements/issues/5683) and one of
several consumers of the conditions that KEP introduced. The `SLM:` prefix on
the tracking issue signals that umbrella context.

## Release Signoff Checklist

Items marked with (R) are required prior to targeting to a milestone / release.

- [ ] (R) Enhancement issue in release milestone, which links to KEP dir in
  [kubernetes/enhancements] (not the initial KEP PR)
- [ ] (R) KEP approvers have approved the KEP status as `implementable`
- [x] (R) Design details are appropriately documented
- [x] (R) Test plan is in place, giving consideration to SIG Architecture and
  SIG Testing input (including test refactors)
  - [ ] e2e Tests for all Beta API Operations (endpoints)
  - [ ] (R) Ensure GA e2e tests meet requirements for [Conformance Tests]
  - [ ] (R) Minimum Two Week Window for GA e2e tests to prove flake free
- [x] (R) Graduation criteria is in place
  - [ ] (R) [all GA Endpoints] must be hit by [Conformance Tests]
- [ ] (R) Production readiness review completed
- [ ] (R) Production readiness review approved
- [x] "Implementation History" section is up-to-date for milestone
- [ ] User-facing documentation has been created in [kubernetes/website], for
  publication to [kubernetes.io]
- [ ] Supporting documentation—e.g., additional design documents, links to
  mailing list discussions/SIG meetings, relevant PRs/issues, release notes

## Summary

When a node undergoes Graceful Node Shutdown ([KEP-2000]; GNS below), the
kubelet knows it is terminating Pods but the control plane does not. The
DaemonSet controller observes a missing DaemonSet Pod on an otherwise eligible
node, creates a replacement, the kubelet rejects it at admission, the controller
deletes the rejected Pod and creates another, and the cycle repeats for the
duration of the shutdown window. The result is sustained create/reject/delete
churn against the API server — hundreds of cycles for a single Pod on a single
node in production reports — in which no Pod ever runs. This KEP is scoped to
stopping that churn.

This KEP makes the kubelet publish the node's shutdown state on the Node object,
using the `GracefulNodeShutdownInProgress` and `DrainInProgress` conditions
introduced by [KEP-5683], and teaches the DaemonSet controller to stop creating
Pods on a node while `GracefulNodeShutdownInProgress` is `True`. The kubelet
clears what it wrote when a shutdown is cancelled and when it starts up. If the
kubelet never comes back, the conditions stay until an administrator clears
them or the Node object is deleted.

No new API types, constants, or reason values are introduced. The change is the
first in-tree writer and the first in-tree reader of conditions that are, as of
v1.37, admin-managed only.

The kubelet's shutdown admission is not changed. Today it rejects every new Pod
for the whole shutdown, whatever the Pod's priority. Making it priority-aware is
a separate bug, tracked in [k/k#142521]; see [Alternatives](#alternatives).

This is **option 2 of the four fixes enumerated by the reporter of
[k/k#122912]** ("teach the DaemonSet controller about `NodeShutdown` status for
a node so that it avoids attempting to schedule Pods there"). The other three
are addressed in [Alternatives](#alternatives).

This KEP is the DaemonSet-controller item of [KEP-5683]'s [alpha-2
milestone][KEP-5683-alpha2] ("consume conditions in controllers"), carried as a
standalone KEP so that it can graduate on its own feature gate.

The DaemonSet **reader** keys on the condition's `type` and `status` only, not
on which component wrote them; see [Other writers](#other-writers) for what that
does and does not imply.

## Motivation

During Graceful Node Shutdown, shutdown is node-local knowledge. The kubelet's
shutdown manager receives a `PrepareForShutdown` signal from systemd-logind,
records the shutdown in memory, triggers a node status sync, and begins
terminating Pods in priority order while rejecting new Pod admission with reason
`NodeShutdown`. The status sync is the only thing the control plane sees: the
kubelet's `Ready` setter consults the shutdown manager and reports `Ready=False`
with the message "node is shutting down." The shutdown manager itself applies no
taints. The `node.kubernetes.io/not-ready` taints (`NoSchedule` and `NoExecute`)
that appear on the node are added by the Node Lifecycle Controller on its next
monitor pass, as they are for any `Ready=False` node. Crucially, none of this
distinguishes the *cause*: shutdown is only an input to the `Ready` condition,
so a gracefully shutting-down node and a node whose container runtime has
crashed both surface as `Ready=False` and acquire the same taints.

The DaemonSet controller is level-triggered. It sees an eligible node missing
its DaemonSet Pod and creates a replacement. The kubelet rejects the replacement
at admission (`Admit()` returns `NodeShutdown`), the Pod goes to `Failed`, the
controller deletes it and creates another. The loop is unbounded for the
duration of the shutdown grace period.

Nothing the node publishes today stops it. As [k/k#122912] puts it, the ability
to ignore node status, the `Unschedulable` flag, and most default taints is "the
key feature of a DaemonSet" — and it is exactly that feature which "can
interfere with the Graceful Node Shutdown feature." `Ready=False` is not a
scheduling input. The `not-ready:NoExecute` taint is tolerated by every
DaemonSet Pod by default. The `not-ready:NoSchedule` taint is not in the default
set, but the system DaemonSets at issue here — CNI, CSI, kube-proxy, service
mesh agents — almost universally declare a blanket `operator: Exists` toleration
precisely so that they run on cordoned, pressured, and not-ready nodes, and so
no taint gates them ([#122912]'s own reproduction uses exactly that toleration).
Between the kubelet's `Ready=False` and the controller's taint pass there is
also a window in which even a DaemonSet with only default tolerations is
eligible.

Two issues document the consequences, and they carry different evidence:

**[k/k#122912]** (open; listed in [KEP-2000]'s tracking issue #2000) is the
canonical analysis. Filed 2024-01-22 with a reproduction, logs, and video, it
identifies the disagreement precisely: the controller "sees that its previously
managed workloads were removed, assumes that this is an invalid state and
attempts to reconcile it," while the shutdown manager's admission callback
"makes no attempts to handle different types of workloads; everything in unison
will be rejected." **This KEP resolves [#122912]'s DaemonSet scope.**

**[k/k#137895]** is an independent production report from an operator on EKS
1.34, and supplies magnitude and blast radius: roughly 300 DaemonSet Pods
created during a single node deletion, with one `ztunnel` Pod going "through the
'Create-Reject-Delete' cycle hundreds of times during a single node's removal,"
and `istio-cni-node`, `ztunnel`, `aws-node`, `kube-proxy`, `ebs-csi-node`,
`efs-csi-node`, `s3-csi-node`, `eks-node-monitoring-agent`, and
`eks-pod-identity-agent` all left Pending for the duration. The reporter
describes the churn as "parasitic load on etcd, the kube-apiserver, and Istio
control plane" and asks whether "a temporary patch or a specific Feature Gate"
exists to resolve it; none does today. The report perceives a regression from
1.33, but [#122912] — filed against 1.29 — shows the underlying disagreement is
long-standing. (The issue is also a tracking issue for [KEP-5683] Story 4.)

> **Note the composition of that list: it is dominated by `system-node-critical`
> networking and storage agents.** Today the kubelet's shutdown admission treats
> them like any other Pod: `Admit()` checks only whether shutdown has begun, not
> priority, so no new critical Pod runs on a shutting-down node during any
> termination tier. That is a kubelet bug in its own right, tracked in
> [k/k#142521]. This KEP does not change admission; it stops the DaemonSet
> controller from fighting it.

Taken together, the reported consequences are:

- **API server flooded** with Pod create/reject/delete churn, amplified when
  many nodes shut down simultaneously (rolling upgrades, spot fleets, rack
  maintenance) and, per [#137895], reaching hundreds of cycles for a single Pod
  on a single node removal.
- **Misleading alerts** on Pod failures that are intentional.
- **Distorted availability and SLA metrics** for DaemonSet workloads, and
  DaemonSet rollout verifiers unable to distinguish lifecycle unavailability
  from a bad revision.
- **Rejected-Pod corpses accumulating**, which per [#122912] are "left … to have
  to be cleaned manually," including Pods "permanently stuck in the `Pending` or
  `Terminating` states."
- **Impeded cluster scale-down**: per [#137895], "System pods remain in Pending
  indefinitely while the node is dying, preventing clean cluster scaling."

*(Scope note: [#122912] also describes non-DaemonSet workloads stranded in
`Error`/`Completed`/`Terminating` states requiring manual cleanup. This KEP
addresses the DaemonSet-owned corpses only — see [DaemonSet controller
(reader)](#daemonset-controller-reader) — and does not attempt general stuck-Pod
cleanup.)*

The DaemonSet controller's own code concedes the fight exists: the failed-Pod
backoff path in `podsShouldBeOnNode` carries a comment noting that the
controller "is often fighting with kubelet that rejects pods." Backoff bounds
the flood but fixes neither the churn nor the attribution.

The information needed to break the cycle exists inside the kubelet at the
moment shutdown begins. Publishing it on the Node object, and teaching the
DaemonSet controller to respect it, fixes the problem at its source rather than
at its symptom.

### Goals

- The kubelet publishes Graceful Node Shutdown state on the Node object using
  the existing `GracefulNodeShutdownInProgress` and `DrainInProgress`
  conditions.
- The DaemonSet controller stops creating Pods on a node while that node is in
  graceful shutdown, without disturbing Pods the kubelet is already terminating
  and without hiding the resulting unavailability from DaemonSet status.
- Rejected DaemonSet Pod corpses on a shutting-down node are cleaned up without
  being replaced, removing a documented source of manual operator toil.
- The reader's contract is defined in terms of condition `type` and `status`
  only. It does not care who wrote the condition: an administrator (the only
  writer as of v1.37), the kubelet (this KEP), or writers added by sibling
  KEPs.
- The kubelet clears the conditions on every path where it can (shutdown
  cancelled, kubelet startup) and clears only the conditions it wrote. If the
  kubelet never returns, the conditions are cleared by an administrator or go
  away with the Node object.
- Every partial-enablement, rollback, and version-skew combination degrades to
  today's behavior; nothing is ever worse than the status quo.

### Non-Goals

The following are explicitly out of scope for this KEP. Several are owned by
sibling enhancements under the same umbrella.

- **Job / maintenance-aware controller behavior.** Out of scope per initial
  scoping with the issue author.
- **ReplicaSet / Deployment coordination.** The same create/reject/delete churn
  affects ReplicaSet-managed Pods on a shutting-down node; that is tracked
  separately in [#6265](https://github.com/kubernetes/enhancements/issues/6265).
  This KEP changes no ReplicaSet or Deployment controller behavior.
- **Rolling-update budget accounting.** Whether nodes suppressed by this KEP
  should be exempt from `maxUnavailable` during a DaemonSet rolling update, and
  more generally how DaemonSet status should attribute lifecycle unavailability,
  is deferred to [KEP-6250] (resolved in WG, 2026-08-24). This KEP changes no
  update semantics.
- **`kubectl drain` as a writer of `DrainInProgress` / `Drained`.** Folded into
  the [KEP-5683] alpha-2 milestone (WG, 2026-08-17).
- **`MaintenancePlanned` / `MaintenanceInProgress` semantics.** [KEP-6250] /
  [KEP-5683] territory.
- **Priority-aware behavior during shutdown.** The kubelet rejects every new
  Pod during a shutdown today, whatever its priority, and this KEP's reader
  holds back every DaemonSet Pod on such a node. Making either side
  priority-aware is tracked in [k/k#142521] and is not on this KEP's
  graduation path; see [Alternatives](#alternatives). General drain ordering
  across the SLM readers remains with the WG lead's separate enhancement.
- **General stuck-Pod cleanup for non-DaemonSet workloads** described in
  [#122912].
- **Multi-writer coordination, ownership, or locking** for node lifecycle
  conditions. Alpha accepts last-writer-wins under the WG's admin-in-control /
  best-effort assumptions, with one rule recorded in [Kubelet
  (writer)](#kubelet-writer): the kubelet sets and clears only the conditions
  it wrote. The general mechanism, shared by the
  sibling KEPs that need it, is [#6430] (SLM: Coordinate Node lifecycle
  condition writers).
- **Controller-driven node teardown.** This KEP's reader keys on
  `GracefulNodeShutdownInProgress`, which only the kubelet writes. Teardown
  driven by a cluster-side controller is served by [KEP-6250], whose reader
  keys on `MaintenanceInProgress`; the two KEPs do not depend on each other
  (WG lead, 2026-09-29). Nothing here sanctions another writer of the GNS
  condition; see [Other writers](#other-writers).
- **Fixing the existing kubelet shutdown-state "amnesia" bug** ([k/k#122674]).
  A kubelet that restarts in the middle of a shutdown does not know it. [Kubelet
  (writer)](#kubelet-writer) says what alpha does and does not handle because
  of that.
- **Graduating `GracefulNodeShutdown`** (beta since v1.21) or changing its
  termination ordering or its admission behavior.

## Proposal

The design has two pieces. Both follow the WG's alpha assumptions: the cluster
administrator remains in control of node lifecycle conditions, every write is
best-effort, and there is no ownership locking between writers.

1. **Kubelet (writer).** On receiving the shutdown signal, before terminating
   any Pod, the shutdown manager publishes `GracefulNodeShutdownInProgress=True`
   and `DrainInProgress=True` in a single Node status update. If
   `DrainInProgress` is already `True` — set by `kubectl drain` or another
   writer — the kubelet leaves that entry alone and writes only
   `GracefulNodeShutdownInProgress`. The kubelet sets and clears only the
   conditions it wrote. It sets them back to `False` if the shutdown is
   cancelled. At startup it sets `GracefulNodeShutdownInProgress` to `False`,
   and sets `DrainInProgress` to `False` only if its shutdown state file says
   the kubelet wrote it.
2. **DaemonSet controller (reader).** While a node has
   `GracefulNodeShutdownInProgress=True`, the controller creates no DaemonSet
   Pods on that node (*suppression*, below). It still deletes failed DaemonSet
   Pods on the node without replacing them, and does not touch running ones.

No other component is specified by this KEP to write or clear the conditions.
If the kubelet never comes back, the conditions stay `True` until an
administrator clears them or the Node object is deleted; see [Kubelet
(writer)](#kubelet-writer).

Only `status: "True"` on `GracefulNodeShutdownInProgress` has behavioral
effect. `False`, `Unknown`, and an absent condition are all equivalent to "no
shutdown in progress" for the reader. The reader ignores `DrainInProgress`; the
kubelet writes it because the WG requires it during Graceful Node Shutdown and
other readers key off it.

**Why a drain does not stop DaemonSet Pods, but a shutdown does.** A drain and
a shutdown are different things for a DaemonSet. `kubectl drain` evicts
workload Pods but not DaemonSet Pods; it refuses to run unless told to ignore
them. That is because a drained node is still a running node: its kubelet is
up, Pods not yet evicted still need networking and storage, and the node may be
uncordoned and reused without a reboot. So a node-level agent that dies during
a drain must be recreated, and the DaemonSet controller keeps creating on a
node with `DrainInProgress=True`. A shutdown is the opposite. The node is going
away, and the kubelet rejects every new Pod until it does, so creating there
only churns. That is why the reader keys on the shutdown condition and not on
the drain one.

The reader in (2) is specified against condition values only; see [Other
writers](#other-writers).

### User Stories

#### Story 1

As a cluster operator running a rolling OS upgrade across a node pool, I want
the DaemonSet controller to stop fighting the kubelet on nodes that are shutting
down, so that my API server is not flooded with create/reject/delete churn and
my on-call is not paged for Pod failures that are intentional.

#### Story 2

As an operator of a DaemonSet-delivered agent (logging, CNI, CSI, security), I
want DaemonSet status and my rollout verifier to reflect that a node is
unavailable because it is shutting down, not because my new revision is broken,
so that I can distinguish lifecycle unavailability from a bad release.

#### Story 3

As an author of a controller that reacts to node lifecycle, I want a
distinguishable, level-triggered signal on the Node object that says "this node
is deliberately going away and its workloads are being drained," so that I do
not have to infer it from `Ready`, taints, and Pod phase.

*This is [KEP-5683] Story 1, which this KEP implements for the DaemonSet
controller.*

#### Story 4

As an operator whose node decommission is driven by a cluster-side controller
rather than a host-level shutdown signal — so that the kubelet's
`GracefulNodeShutdown` path never fires — I experience the same DaemonSet churn
during teardown, and I want the DaemonSet controller's coordination to be
expressed in terms of node conditions so that a future writer for my teardown
path can participate without a second mechanism.

*This story is not served by this KEP. It is recorded here because it is the
same churn with a different trigger. [KEP-6250] serves it with its own
condition, `MaintenanceInProgress`, and its own reader; see [Other
writers](#other-writers).*

### Notes/Constraints/Caveats

- **The kubelet's write must land inside the logind inhibit window.** The
  kubelet holds a delay inhibitor lock; once it releases the lock (after
  `killPods` returns), systemd proceeds regardless. An unreachable API server
  must never delay the shutdown itself, so the write is issued first but never
  waited on, and the kubelet re-asserts the condition on its later status
  updates while the shutdown is in progress.
- **Whether a shutdown is in progress is in-memory only.** `nodeShuttingDownNow`
  is a boolean behind a mutex; a restarted kubelet cannot know it was
  mid-shutdown. (The state file records only what the kubelet wrote, not
  whether a shutdown is still under way.)
  A kubelet that is starting and has received no shutdown signal treats itself
  as not in shutdown and clears `GracefulNodeShutdownInProgress`. If it
  restarted in the middle of a shutdown, that clear is early. Alpha accepts
  that; see [Kubelet (writer)](#kubelet-writer).
- **Conditions observe; taints enforce.** This KEP uses conditions as an
  observation primitive that a controller reads. The conditions do not affect
  scheduling or admission; the only behavior they drive is whether the
  DaemonSet controller creates a Pod.
- **Reasons are cause categories, not phase encodings.** Per [KEP-5683]
  conventions (and the WG position for the `kubectl drain` writer), the `reason`
  field is informational and is not an API for readers to key behavior off. This
  KEP introduces no new reason vocabulary; it uses `NodeShutdown`, already
  reserved by [KEP-5683]. No reader specified by this KEP keys behavior off any
  reason value.
- **The reader keys on `type` and `status` only.** `podsShouldBeOnNode` has no
  knowledge of, and takes no dependency on, which component performed a write.
  This is the [KEP-5683] admin-managed model; it is not an invitation for
  arbitrary writers to assert `GracefulNodeShutdownInProgress` — see [Other
  writers](#other-writers).
- **API doc-comment change.** The condition constants in
  [`staging/src/k8s.io/api/core/v1/types.go`][staging/src/k8s.io/api/core/v1/types.go]
  currently state that "the admin is responsible for setting and clearing this
  condition." Adding an in-tree writer is a semantic change to that comment,
  which the implementation PR updates.

### Risks and Mitigations

| Risk | Mitigation (alpha) |
|---|---|
| The kubelet's status write does not land before the machine dies (slow or unreachable API server during a rack-wide event). | The write is best-effort and never waited on, so it cannot delay the shutdown; the kubelet re-asserts the condition on its later status updates while the shutdown is in progress. If nothing lands, the DaemonSet controller sees absent conditions and behaves as today. Never worse than status quo. |
| Conditions left `True` after the kubelet is gone (crash, power loss). | Cleared by the kubelet at startup when the node returns. If the kubelet never returns, the conditions stay `True` until an administrator clears them or the Node object is deleted. Nothing can run on that node in the meantime, so holding DaemonSet Pods back costs nothing. Accepted for alpha; automatic detection of a vanished writer is a graduation item. |
| A stale `True` condition on a node whose kubelet is alive (for example, the cancel write failed). | The kubelet keeps `GracefulNodeShutdownInProgress` in step with its own state on every node status update, so a stale value lasts one heartbeat. |
| The kubelet restarts in the middle of a shutdown. | It does not know a shutdown is in progress ([k/k#122674], the amnesia bug) and clears `GracefulNodeShutdownInProgress` at startup. The DaemonSet controller may create Pods on that node for the few seconds until the machine goes down; those Pods are rejected or left `Pending`, as today. Accepted for alpha. |
| Another writer flips `DrainInProgress` mid-shutdown (`kubectl drain`, a maintenance operator, an admin). | The reader ignores `DrainInProgress`, so suppression is unaffected. The kubelet never overwrites a `DrainInProgress=True` it did not set, so the initiating writer's `reason`, `message`, and `lastTransitionTime` survive the shutdown. General cross-writer coordination is [#6430]. |
| A reboot could reset a `DrainInProgress=True` that another writer set before the shutdown. | The kubelet records in its shutdown state file that it wrote `DrainInProgress`, and at startup clears `DrainInProgress` only when that record is present. If the kubelet is stopped before the record is written, or the file is lost, the kubelet leaves `DrainInProgress` alone; a kubelet-written value then stays `True` until an administrator clears it. Accepted for alpha; [#6430] covers the general case. |
| An admin sets `GracefulNodeShutdownInProgress` to `False`, or removes it, while a shutdown is in progress (new edge case on [k/k#122674]). | The shutdown itself continues: the condition is something the kubelet publishes, not an input the shutdown manager reads, and Pod termination and admission rejection carry on unchanged. The DaemonSet controller may create Pods on the node until the kubelet's next status update, which sets the condition back to `True`; the kubelet rejects those Pods, as today. If the kubelet restarts mid-shutdown it no longer knows, and does not restore the condition. Accepted for alpha; the graduation direction is that the kubelet reads level-triggered shutdown state from the OS (WG, 2026-08-17). |
| A control-plane component runs as a DaemonSet on a node carrying a stale `True` condition. | During a real shutdown nothing changes for it: the kubelet already rejects every new Pod. Outside a real shutdown, a live kubelet corrects the condition on its next status update; a dead kubelet runs no Pods regardless. See the rollout answer in the PRR questionnaire. |
| Informer propagation race: the DaemonSet controller may issue one create between the kubelet's write and the controller observing it. | Publishing the conditions before terminating Pods bounds the recreate loop to at most ~one controller sync instead of unbounded. |

## Design Details

### The conditions

`GracefulNodeShutdownInProgress` and `DrainInProgress` already exist: the node
lifecycle `NodeConditionType` constants merged into `k8s.io/api/core/v1` in
v1.37 behind the `NodeLifecycleConditions` feature gate (alpha, default off). As
of v1.37 nothing in core writes or reads them. This KEP adds the first in-tree
writer and reader; no new API types, constants, or reason values.

As published by the kubelet during shutdown, in a single status update:

```yaml
status:
  conditions:
  - type: GracefulNodeShutdownInProgress
    status: "True"
    reason: NodeShutdown
    message: "Kubelet received a shutdown signal and is terminating pods"
  - type: DrainInProgress
    status: "True"
    reason: NodeShutdown
    message: "Graceful node shutdown is draining pods on this node"
```

**Why the kubelet writes two conditions and the reader uses one.** The WG
requires the kubelet to set `DrainInProgress=True` during Graceful Node
Shutdown ([KEP-5683] Story 1; confirmed by the WG lead, 2026-09-29). It is the
common signal that Pods are leaving a node, and other readers key off it. `GracefulNodeShutdownInProgress` says *why*: the kubelet has
received a shutdown signal and is rejecting new Pods. The DaemonSet reader needs
only the why. It acts on `GracefulNodeShutdownInProgress=True` alone and ignores
`DrainInProgress`. So a plain `kubectl drain` or a maintenance drain, which sets
`DrainInProgress` without the GNS condition, does not stop DaemonSet Pods. A
drained node is still a running node, and its node-level agents must keep
running and be recreated if they die; see the [Proposal](#proposal) for the
full reasoning. Only the kubelet writes
`GracefulNodeShutdownInProgress`, so the reader has one writer to reason about.
*Reader rule agreed with SIG Apps, SIG Node, and the WG lead at KEP review,
2026-09-29.*

**What "clear" means.** Throughout this document, clearing a condition means
setting `status` on the existing entry, never removing the entry from
`status.conditions`. The kubelet sets `False` (shutdown cancelled; kubelet
startup; a later status update while not in shutdown). No component specified
by this KEP sets `Unknown` or deletes a
condition entry; removal remains an administrator action under [KEP-5683]'s
admin-managed model.

Semantics for the reader in this KEP:

- Only `status: "True"` on `GracefulNodeShutdownInProgress` has behavioral
  effect (the *fail-open* rule).
- `False`, `Unknown`, and an absent condition are equivalent to "no shutdown in
  progress."
- Conditions are level-triggered observations. Readers must behave correctly
  whether they observe every transition or only the current value.
- Readers key on `type` and `status` only. `reason` and `message` are
  informational.

This fail-open rule is what makes partial enablement, rollback, and version skew
safe without additional machinery.

### Kubelet (writer)

The writer is the kubelet's Graceful Node Shutdown manager
([`pkg/kubelet/nodeshutdown/`][pkg/kubelet/nodeshutdown/]). When the OS signals
an impending shutdown — `PrepareForShutdown(true)` from systemd-logind on Linux,
`SERVICE_CONTROL_PRESHUTDOWN` from the Service Control Manager on Windows
([KEP-4802]) — the manager does two things, in this order:

1. **Publish the conditions** — a single Node status update setting
   `GracefulNodeShutdownInProgress=True` and `DrainInProgress=True`, both with
   reason `NodeShutdown` (a `DrainInProgress=True` that is already present is
   left as is; see below). The write is best-effort and never waited on: it
   is issued before step 2 begins, in its own goroutine, and step 2 starts
   without waiting for it to land. A slow or unreachable API server therefore
   never delays the shutdown, which matters on nodes with seconds to live, such
   as spot instances. Today the shutdown manager records the shutdown in memory
   and then fires the kubelet's generic status sync in a goroutine it does not
   await (`go m.syncNodeStatus(...)` in
   [`nodeshutdown_manager_linux.go`][nodeshutdown_manager_linux.go]), so
   `Ready=False` may land after Pod termination has already begun. The
   condition write is a dedicated call issued ahead of that, so in the common
   case the conditions reach the API server before the first Pod termination
   does, rather than being left to the status loop's timing. If the write does
   not land, the DaemonSet controller behaves as today for that shutdown; see
   [Risks and Mitigations](#risks-and-mitigations). After that first write, the
   kubelet keeps `GracefulNodeShutdownInProgress` in step with its own state on
   every later node status update: `True` while it is in shutdown, `False` when
   it is not. That is what makes a failed first write, a failed cancel write,
   and an administrator removing the condition mid-shutdown all self-correct
   within one status update while the same kubelet process is running.
2. **Begin Pod termination** per the existing [KEP-2000] priority-tiered
   ordering. Admission is unchanged: today `Admit()` rejects every new Pod once
   shutdown has begun, whatever its priority ([`Admit`][k/k-admit]). Making it
   priority-aware is [k/k#142521].

**Why a single write of both conditions.** Graceful Node Shutdown moves directly
from signal detection to Pod termination; verified against the shutdown manager,
there is no intermediate phase between "shutdown started" and "drain started."
*Resolved in WG (2026-08-24): accepted for alpha; to be corrected with SIG Node
if they see an issue.*

**Invariant (agreed with the issue author).** The signal that gates DaemonSet
Pod creation transitions at the same moment the kubelet begins rejecting Pod
admission. Issuing the condition write before terminating Pods is how the
design honors it, without making termination wait for the write.

**What the kubelet touches: one rule.** The kubelet sets and clears only the
conditions it wrote. If `DrainInProgress` is already `True` when the shutdown
signal arrives — set by `kubectl drain`, a maintenance operator, or an
administrator — the kubelet writes only `GracefulNodeShutdownInProgress` and
leaves the existing entry untouched, so its `reason`, `message`, and
`lastTransitionTime` keep saying who started the drain and when. On cancel it
resets only what it wrote. Across a restart it knows what it wrote from its
shutdown state file (see "Kubelet startup" below). It never inspects `reason`
to decide. This is the one ownership rule alpha carries; the general mechanism
is [#6430].

Two more transitions complete the happy path:

- **Shutdown cancelled.** logind emits `PrepareForShutdown(false)`; the shutdown
  manager already handles this by re-acquiring the inhibit lock and resuming
  admission. The kubelet sets the conditions it wrote for this shutdown to
  `False` on this path.
- **Kubelet startup.** A kubelet that is starting and has received no shutdown
  signal is not in shutdown. So, before reporting `Ready`, it sets
  `GracefulNodeShutdownInProgress` to `False` if it is `True`. This covers the
  common case: the shutdown completed, the machine rebooted, and the Node
  object still carries the old value. For `DrainInProgress` the kubelet
  consults its shutdown state file. When the shutdown signal arrived it
  recorded there whether it wrote `DrainInProgress`; at startup it sets
  `DrainInProgress` to `False` only if that record is present, then removes
  the record. A `DrainInProgress` set by `kubectl drain` or an operator before
  the reboot is left alone. The file is the `graceful_node_shutdown_state`
  file the shutdown manager already keeps under the kubelet root directory
  ([kubelet files][kubelet-files-gns]); this KEP adds one field to it.
  *Startup clear resolved in WG (2026-08-17); the state-file record was agreed
  with SIG Node at KEP review, 2026-09-28.*

  Two gaps in this rule are accepted for alpha. The kubelet can be stopped
  before the record is written, or the file can be lost; then the kubelet
  leaves `DrainInProgress` alone, and a value the kubelet wrote stays `True`
  until an administrator clears it. And a kubelet that restarts in the middle
  of a shutdown does not know it ([k/k#122674], the amnesia bug), so it clears
  `GracefulNodeShutdownInProgress` a few seconds before the machine goes down.
  Neither gap is solved in alpha; see [Risks and
  Mitigations](#risks-and-mitigations).

**If the kubelet never comes back.** Nothing in alpha clears the conditions for
a node whose kubelet has died and not returned. They stay `True` until an
administrator sets them to `False` or removes them, or until the Node object is
deleted, which takes them with it. That is deliberate. There is no signal that
says a shutdown has finished, so a controller that cleared on a guess — for
example when `Ready` goes `Unknown` — would sometimes clear while the node was
still being torn down and release DaemonSet Pods onto it. In the meantime
nothing can run on that node, so holding DaemonSet Pods back costs nothing.
Node deletion is the end state for a node that does not return. *Agreed with
the WG lead at KEP review, 2026-09-29.*

**Windows.** [KEP-4802] (`WindowsGracefulNodeShutdown`, beta since v1.34) gives
Windows nodes the same shutdown manager shape: `ProcessShutdownEvent` in
`nodeshutdown_manager_windows.go` records the shutdown, fires the status sync,
and calls the shared `podManager.killPods`, and `Admit` rejects new Pods for the
duration — so the DaemonSet churn this KEP fixes occurs on Windows today. The
condition publish is therefore implemented as a shared helper in the
`nodeshutdown` package, invoked by both platform managers immediately before
`killPods`, which gives Windows the same ordering guarantee with no
platform-specific code. Two differences are absorbed by the existing design:
[KEP-4802] lists shutdown cancellation as a Non-Goal and the Windows manager has
no cancel path, so the "shutdown cancelled" transition above does not exist on
Windows and kubelet-startup clear is the only recovery rule there; and the
shutdown window is the SCM preshutdown timeout, which the Windows manager
already extends to the configured grace period. GNS on Windows requires the
kubelet to run as a Windows service; where it does not, no conditions are
written and the reader behaves as today.

**The amnesia bug.** The kubelet today does not remember it was in graceful
shutdown across a restart ([k/k#122674]). Alpha does not fix that. Two
consequences are recorded above: a restart mid-shutdown clears
`GracefulNodeShutdownInProgress` early, and a condition an administrator
removes during a shutdown is restored only while the same kubelet process is
running. The WG's agreed graduation direction is that the kubelet reads
level-triggered shutdown state from the OS — for example, whether the systemd
inhibitor / `PrepareForShutdown` state is re-observable after a restart — so
that a restarted kubelet knows where it is.

**Implementation shape.** A new nodestatus setter alongside the existing
condition setters in [`pkg/kubelet/nodestatus/`][pkg/kubelet/nodestatus/],
following the established pattern, driven by the shutdown manager's state, and
a shared publish helper in `nodeshutdown` called by both the Linux and Windows
managers. The state-file record, the startup clear, and the per-status-update
reconciliation live in the shutdown manager. Unit-testable against the existing
fake `dbusInhibiter` on Linux and against the shared helper directly on
Windows.

### DaemonSet controller (reader)

The reader is the DaemonSet controller
([`pkg/controller/daemon/`][pkg/controller/daemon/]). The change touches three
places, all keyed on the same predicate: the node carries
`GracefulNodeShutdownInProgress=True`. The implementation factors that
predicate into one helper (working name `nodeShutdownSuppressed(node)`) so the
call sites cannot drift:

1. `podsShouldBeOnNode` — the per-node decision made on every sync.
2. `rollingUpdate` — walks nodes independently of `podsShouldBeOnNode` and
   must apply the same predicate.
3. `shouldIgnoreNodeUpdate` — the informer-side filter that decides whether a
   Node update reaches the controller at all; without a change here the
   controller never observes the conditions changing.

**`podsShouldBeOnNode`.** While a node is suppressed, the controller does the
following:

- **Create nothing on the node** — no replacement Pods, no first-time
  placements.
- **Still delete failed DaemonSet Pods on the node** (the rejected-Pod corpses),
  without replacing them. This is the change that breaks the cycle, and it
  directly addresses [#122912]'s report that these failed attempts are "left …
  to have to be cleaned manually."
- **Do not touch running DaemonSet Pods.** The kubelet owns their termination.
- **Status stays honest.** The change is scoped to `podsShouldBeOnNode` rather
  than `nodeShouldRunDaemonPod`, so the node remains in `desiredNumberScheduled`
  and the missing Pod surfaces in `numberUnavailable`. The unavailability is
  real and intentional. Whether it should be exempt from rolling-update
  `maxUnavailable` budgets is deferred to [KEP-6250].

**Why not `DrainInProgress` too.** `DrainInProgress` can be set by other
writers — `kubectl drain`, a maintenance operator, an administrator by hand —
and DaemonSet Pods are expected to survive those drains. Only the kubelet's
own shutdown warrants suppression, and only the kubelet writes
`GracefulNodeShutdownInProgress`, so that one condition is the whole predicate.
Earlier drafts required both conditions; see [Alternatives](#alternatives) for
why that was dropped. An integration test asserts that a node with only
`DrainInProgress=True` is not suppressed and a node with only
`GracefulNodeShutdownInProgress=True` is.

**`rollingUpdate`.** The rolling-update path iterates `nodeToDaemonPods` on
its own and would otherwise act on a suppressed node under both strategies.
With `maxSurge > 0`, a node holding an old Pod and no new Pod is a surge
candidate, so the controller would create a new-hash Pod that the kubelet
immediately rejects — the same churn this KEP removes from the core loop. With
`maxSurge == 0`, an old Pod that is still available is a deletion candidate, so
the controller would race the kubelet for a Pod that is already being
terminated and spend `maxUnavailable` budget doing it. The implementation builds
the suppressed-node set once from `nodeList` at the top of
`rollingUpdate` and skips those nodes as surge-create and delete candidates in
both branches. This
removes the controller as an *actor* on the node; it does not change
accounting. A suppressed node's missing or terminating Pod continues to count
as unavailable, consistent with the status treatment above, and whether it
should be exempt from the `maxUnavailable` budget remains [KEP-6250]'s question.
`updatedDesiredNodeCounts` is unchanged.

**Recovery.** One filter change, then existing machinery. Today
`shouldIgnoreNodeUpdate` compares only `Labels` and `Spec.Taints`, so a change
to `Node.Status.Conditions` never reaches the node-update worker. Under the
gate, the filter additionally returns `false` when the `Status` of
`GracefulNodeShutdownInProgress` differs between the old and new Node. Only
`Status` is compared — never `LastHeartbeatTime` or `LastTransitionTime` — so
heartbeats enqueue nothing and the controller sees two events per shutdown:
enter and exit. From there the existing `syncNodeUpdate` worker does
the right thing without modification. On exit, `NodeShouldRunDaemonPod` is
`true` and no Pod is scheduled on the node, which is already an enqueue
condition, and the Pod returns on the next sync. On enter, the running Pod is
still scheduled, nothing is enqueued, and the kubelet's own termination drives
the subsequent Pod events. Without the filter change, recovery after a reboot
would only *happen* to work because the `node.kubernetes.io/not-ready` taint
flips on the way through; an aborted shutdown that never goes `NotReady`
(inhibitor released, kubelet clears the conditions) would leave the node without
its DaemonSet Pods until an unrelated event arrived.

**Informer ordering.** The Node condition write and the kubelet's first Pod
deletion arrive on separate informers with no ordering guarantee between them.
If a Pod-delete event is processed before the Node event carrying the
conditions, `podsShouldBeOnNode` sees an unsuppressed node and creates one
replacement, which the kubelet rejects and the next sync deletes. This is
fail-open and bounded to one Pod per DaemonSet per shutdown. The kubelet
publishes the conditions before it begins terminating Pods, and termination
takes at least the Pod's grace period, so the window is narrow in practice.
Alpha accepts it; closing it would require a live read of the Node before every
create and is not worth the API cost.

**Expectations.** The early return that skips a create must not leave a dangling
creation expectation on the controller; the implementation must ensure
`podsShouldBeOnNode` decides before any expectation is recorded.

### Other writers

The kubelet is the writer this KEP *implements*. It is not the only writer the
conditions *admit*: [KEP-5683] introduced them as admin-managed, and the doc
comment on the constants states that "the admin is responsible for setting and
clearing this condition." Because the DaemonSet reader keys on `type` and
`status` only, a condition set by an administrator or by another controller is
honored exactly as a kubelet-written one is — under the same best-effort,
no-locking, last-writer-wins assumptions this KEP applies to the kubelet. One
consequence of the kubelet owning `GracefulNodeShutdownInProgress`: on a node
whose kubelet is running with the gate, the kubelet corrects that condition on
its next status update, so a value written there by someone else lasts one
heartbeat. Administrator-written values persist only on nodes whose kubelet
lacks the gate or is not running.

That property is deliberate, and it is what lets the reader stay correct as
sibling KEPs add writers (`kubectl drain` in [KEP-5683] alpha 2; maintenance
operators in [KEP-6250] / [KEP-6251]). It is not a sanction for arbitrary
writers to assert `GracefulNodeShutdownInProgress`. [KEP-5683] defines that
condition as reporting that Graceful Node Shutdown is in progress on the node; a
controller that tears down a node without a shutdown event and sets it anyway
would be asserting something untrue in order to obtain the DaemonSet behavior —
and the whole reason this KEP keys on `GracefulNodeShutdownInProgress` rather
than `DrainInProgress` is that GNS *scopes* the suppression.

Controller-driven node teardown (Story 4) is a real environment with the same
churn, and it deserves the same coordination. It gets it from [KEP-6250]: that
KEP's reader keys on `MaintenanceInProgress`, this KEP's reader keys on
`GracefulNodeShutdownInProgress`, and neither depends on the other (WG lead,
2026-09-29). No combination rule is planned here. An
external writer that chooses to set these conditions in the meantime does so
under the admin-managed model as it stands in v1.37, inherits the
stale-condition risks above without the kubelet's startup-clear safety net, and
is responsible for its own clearing.

### Feature gating

*Resolved in WG (2026-08-24):* this KEP does not reuse `NodeLifecycleConditions`
for its behavior. It introduces its own feature gate so that the
DaemonSet-coordination behavior matures on its own track while
`NodeLifecycleConditions` and the `kubectl drain` writer graduate separately.
The new gate does depend on `NodeLifecycleConditions`, since it writes and reads
the conditions that gate introduced.

- **Gate name:** `DaemonSetGracefulNodeShutdown` (see [Design
  Decisions](#design-decisions))
- **Components:** kubelet, kube-controller-manager
- **Stage:** alpha, default off, v1.38
- **Depends on:** `NodeLifecycleConditions`, declared in
  `defaultKubernetesFeatureGateDependencies` in
  [`pkg/features/kube_features.go`][pkg/features/kube_features.go]. Feature-gate
  validation refuses to start a component with `DaemonSetGracefulNodeShutdown`
  enabled and `NodeLifecycleConditions` disabled.

The working assumption is a single gate covering both this KEP's kubelet writer
and its DaemonSet reader, to minimize gate count per the WG's alpha philosophy.
Splitting into separate writer/reader gates remains a fallback if SIG Node
prefers decoupled enablement.

Layout relative to the existing gate:

| `NodeLifecycleConditions` | `DaemonSetGracefulNodeShutdown` | Result |
|---|---|---|
| off | off | v1.37 behavior; conditions admin-managed only. |
| on | off | Conditions remain admin-managed; today's churn persists. Safe. |
| off | on | Rejected at component startup by feature-gate dependency validation; this combination cannot run. |
| on | on | Full behavior; churn loop broken. |

Fail-open semantics make every partial combination safe: any component without
the gate sees absent conditions and behaves as today.

### Metrics

Metrics are defined per consuming KEP (per prior scoping with the issue author).
Alpha adds, at minimum, on the DaemonSet reader:

- **`daemonset_controller_node_shutdown_suppression_total`** — counter
  recorded in the DaemonSet controller's Node update handler (`updateNode`),
  which receives both the old and the new Node, when the `Status` of
  `GracefulNodeShutdownInProgress` differs between them. Incremented once per
  node per transition. Gives PRR an observable signal that the feature is
  active and makes the bounded-loop claim verifiable.
  - **Label `transition="enter"|"exit"`** — whether the node entered or left
    suppression. The feature is in use only between an `enter` and its
    matching `exit`, so the difference of the two over a window is the in-use
    gauge; no separate gauge is added.

  Recording in the update handler rather than in `podsShouldBeOnNode` or the
  node-update worker is deliberate. A counter bumped inside `syncDaemonSet`
  counts syncs that happened to run: once the rejected Pod on a suppressed
  node has been deleted, nothing triggers a further sync until an unrelated
  Pod or Node event arrives, so the value would track cluster noise rather
  than the feature. The node-update worker (`syncNodeUpdate`) receives only a
  node name and reads the current Node from the lister, so it has no old value
  to detect a flip from. The update handler is the one place that sees both
  values; it runs once per observed condition change (the
  `shouldIgnoreNodeUpdate` change above guarantees the event is delivered), so
  each transition is counted once regardless of sync scheduling. `enter` rising
  with no matching `exit`, or `enter` events while no node is shutting down,
  are both directly meaningful. The counter is per node, not per DaemonSet:
  counting affected DaemonSets would need Pod lookups in the informer handler.

Writer-side metrics belong to the condition-writing milestone and are not
proposed here.

### Design Decisions

*Questions raised during drafting and how each was closed. Nothing here is
open; the list exists so reviewers can see where each answer came from.*

1. **Feature gate name.** `DaemonSetGracefulNodeShutdown`; see [Feature
   gating](#feature-gating). Decided at KEP review.
2. **Reader keys on `GracefulNodeShutdownInProgress` alone.** Earlier drafts
   required `DrainInProgress=True` as well, following [KEP-5683] Story 1. The
   kubelet writes both in one update, so for the DaemonSet controller the
   second condition adds no information, and the kubelet-only condition gives
   the reader a single writer. The kubelet still writes `DrainInProgress`, as
   the WG requires. Decided with SIG Apps, SIG Node, and the WG lead at KEP
   review, 2026-09-29.
3. **No control-plane writer; Node deletion is the end state.** Earlier drafts
   had the Node Lifecycle Controller set the conditions to `Unknown` when
   `Ready` went `Unknown`. Dropped: there is no signal that a shutdown has
   finished, so that clear was a guess, and on architectures where the Node
   object outlives the kubelet it guessed wrong. In alpha the kubelet clears
   what it can, and a node that never returns keeps its conditions until an
   administrator clears them or the Node is deleted. Decided with the WG lead
   at KEP review, 2026-09-29.
4. **Single feature gate for writer and reader.** *Resolved in WG
   (2026-08-24)*; see [Feature gating](#feature-gating). The gate depends on
   `NodeLifecycleConditions` via the feature-gate dependency map. Decided at
   SIG Apps review.
5. **Metric label.** `transition` on
   `daemonset_controller_node_shutdown_suppression_total`, a bounded two-value
   label, recorded in the Node update handler. Cardinality is reviewed by SIG
   Instrumentation at the implementation PR as usual. Decided at SIG Apps and
   KEP review.
6. **Priority handling is out of scope.** At API review (2026-09-25) the KEP
   briefly carried tier-aware kubelet admission plus a third condition,
   `GracefulNodeShutdownCriticalPhase`. SIG Node asked for the smallest alpha
   and no special case for one priority stage, and the admission problem was
   filed as its own bug, [k/k#142521]. Withdrawn at SIG Node review,
   2026-09-28; see [Alternatives](#alternatives).
7. **Reader keys on `type` and `status` only; partial-rollout safety is
   operational.** Keying on `reason` was considered as a way to distinguish
   kubelet-written state from administrator-written or stale state and
   rejected: API conventions reserve `reason` for explanation, not control
   flow (see [Alternatives](#alternatives)). Instead, [Upgrade / Downgrade
   Strategy](#upgrade--downgrade-strategy) asks for a preflight listing of
   nodes carrying the condition, with kubelet-first enablement only when that
   listing is non-empty, and the reader-only enablement test exercises the
   writer-agnostic behavior directly. A heartbeat rule that makes enablement
   order-agnostic is a [beta item](#beta). Decided at PRR review.
8. **Startup clear of `DrainInProgress` uses the shutdown state file.** The
   kubelet records whether it wrote `DrainInProgress` and clears it at startup
   only when that record is present, so a drain another writer started
   survives a reboot. Agreed with SIG Node at KEP review, 2026-09-28; the WG
   lead concurred with the caveat that it does not cover a kubelet stopped
   before the record is written or a kubelet restarting mid-shutdown, both
   recorded in [Risks and Mitigations](#risks-and-mitigations).
9. **The kubelet reconciles `GracefulNodeShutdownInProgress` on every status
   update.** Raised at PRR review (2026-09-29): a stale `True` on a live node
   would hold back control-plane components some platforms run as DaemonSets.
   The kubelet owns the condition, so it sets it to match its own state on
   every node status update; a stale value lasts one heartbeat.

**Follow-up (not a design question).** Multi-writer coordination of node
lifecycle conditions is tracked in [#6430]. General drain ordering across the
SLM readers remains with the WG lead's separate enhancement; [KEP-5683] is the
umbrella reference until it is filed.

### Test Plan

- [x] I/we understand the owners of the involved components may require updates
  to existing tests to make this code solid enough prior to committing the
  changes necessary to implement this enhancement.

##### Prerequisite testing updates

None identified. The existing fake `dbusInhibiter` in
[`pkg/kubelet/nodeshutdown/`][pkg/kubelet/nodeshutdown/] and the DaemonSet
controller test fixtures are sufficient to exercise the new paths.

#### Unit tests

Coverage at time of writing (`go test -cover`):

- `k8s.io/kubernetes/pkg/kubelet/nodeshutdown`: `2026-09-21` - `34.5%`
- `k8s.io/kubernetes/pkg/kubelet/nodestatus`: `2026-09-21` - `89.8%`
- `k8s.io/kubernetes/pkg/controller/daemon`: `2026-09-21` - `69.4%`

Coverage in `pkg/kubelet/nodeshutdown` is low because the systemd/logind
integration in `nodeshutdown_manager_linux.go` is only partially exercisable
with the fake `dbusInhibiter`; the condition-writing path added by this KEP
will be covered by unit tests against that fake and end-to-end against real
systemd in `test/e2e_node/`.

New and updated tests:

- [`pkg/kubelet/nodeshutdown/`][pkg/kubelet/nodeshutdown/]: condition write on
  shutdown signal; a pre-existing `DrainInProgress=True` left untouched; the
  state-file record written with the conditions; re-assert on a later status
  update while in shutdown; clear on cancel; clear on startup, with
  `DrainInProgress` cleared only when the record is present; a stale `True`
  corrected on the next status update when not in shutdown; gate on/off.
  Against the existing fake `dbusInhibiter`.
- [`pkg/kubelet/nodestatus/`][pkg/kubelet/nodestatus/]: new setter, gate on/off.
- [`pkg/controller/daemon/`][pkg/controller/daemon/]: `podsShouldBeOnNode` with
  `GracefulNodeShutdownInProgress=True`, with `DrainInProgress=True` only, with
  neither; failed-Pod deletion without replacement; gate on/off.
  `rollingUpdate`: a suppressed node is neither a surge-create candidate
  (`maxSurge > 0`) nor a delete candidate (`maxSurge == 0`), gate on/off.
  `shouldIgnoreNodeUpdate`: a `GracefulNodeShutdownInProgress` `Status` flip
  passes the filter, a `DrainInProgress`-only flip does not, a heartbeat-only
  update does not, gate off ignores conditions. `updateNode`: metric
  attribution with the `transition` label.

#### Integration tests

[`test/integration/daemonset/`][test/integration/daemonset/] — the clearest
statement of what this KEP does:

- Node with `GracefulNodeShutdownInProgress=True` and `DrainInProgress=True`
  (as the kubelet writes them) → DaemonSet Pod deleted and not recreated.
- Node with `GracefulNodeShutdownInProgress=True` only → Pod deleted and not
  recreated (proves the reader needs only the one condition).
- Node with `DrainInProgress=True` only → Pod is recreated (a plain drain does
  not suppress).
- Reader enabled against a node already carrying
  `GracefulNodeShutdownInProgress=True` → suppressed until cleared (the
  writer-agnostic reader; exercised deliberately so the rollout-ordering
  requirement is backed by a test).
- Rolling update while a node is suppressed → no new-hash Pod is created there
  (`maxSurge > 0`) and the controller does not delete the old Pod there
  (`maxSurge == 0`); the rollout completes on every other node.
- Conditions cleared after the node went `NotReady` → Pods return on the next
  sync.
- Conditions cleared without the node ever leaving `Ready` (aborted shutdown)
  → Pods return on the next sync. This is the case the `shouldIgnoreNodeUpdate`
  change exists for.
- Feature gate off → today's behavior, unchanged. This is the path every real
  cluster will run.

#### e2e tests

- [`test/e2e_node/`][test/e2e_node/] — extend the existing graceful node
  shutdown suite (the only place GNS is exercised against real systemd) to
  assert that the conditions are published before Pod termination begins, that
  `GracefulNodeShutdownInProgress` is cleared on kubelet restart, and that a
  `DrainInProgress` set by another writer before the shutdown survives the
  restart.

### Graduation Criteria

#### Alpha (v1.38)

- Feature gate `DaemonSetGracefulNodeShutdown`, default off, declaring its
  dependency on `NodeLifecycleConditions`.
- Kubelet writer (both conditions, the state-file record, the startup clear,
  and per-status-update reconciliation) and DaemonSet reader implemented
  behind the gate.
- Unit and integration coverage with the gate on and off.
- Suppression counter metric present.

#### Beta

*Direction only; criteria to be firmed up with the WG-level plan for how the SLM
writer and reader KEPs graduate together.*

- **Multi-writer coordination.** Adoption of the mechanism defined by [#6430]
  (SLM: Coordinate Node lifecycle condition writers), including cross-writer
  handling of `DrainInProgress` and a startup clear that does not depend on
  the kubelet's state file.
- **Amnesia-bug edge case.** The kubelet reads level-triggered shutdown state
  from the OS, so a kubelet that restarts mid-shutdown knows it and keeps the
  conditions correct.
- **Vanished-writer detection.** A way to clear the conditions of a node whose
  kubelet never returns without waiting for an administrator or Node deletion,
  once there is a signal that says a shutdown has finished; with a distinct
  reason.
- **Order-agnostic rollout.** The reader ignores a `True` condition whose
  `lastHeartbeatTime` is older than the node's `Ready` heartbeat. A live
  kubelet with the gate refreshes both on every status update; a live kubelet
  without the gate refreshes only `Ready`, so a stale condition on it is
  ignored; a dead kubelet refreshes neither, so suppression stays, which is
  harmless. With that rule the preflight and the enablement order above become
  unnecessary. Raised at PRR review, 2026-09-29.
- e2e coverage in [`test/e2e_node/`][test/e2e_node/]; version-skew matrix
  documented and tested.
- **Windows e2e.** Condition publish verified against a real SCM preshutdown
  event in the SIG Windows node suite, coordinated with [KEP-4802]'s own e2e
  work. Alpha covers Windows through the shared helper's unit tests only.
- **Evidence of use from at least one production environment.**

#### GA

- Condition contract stable for additional readers.
- Conformance considerations for any endpoint behavior changes.

### Upgrade / Downgrade Strategy

The order of enablement matters only if some nodes already carry
`GracefulNodeShutdownInProgress=True` before the reader is turned on. In v1.37
nothing in core writes that condition, so on a cluster that has never run this
feature, and where no administrator has set the condition by hand, the gate can
be enabled on the kubelet and kube-controller-manager in any order. The
preflight below is how to tell which case a cluster is in.

- **Preflight.** Before enabling the reader, list nodes already carrying the
  condition:
  `kubectl get nodes -o jsonpath='{range .items[*]}{.metadata.name}{" "}{.status.conditions[?(@.type=="GracefulNodeShutdownInProgress")].status}{"\n"}{end}' | grep " True"`.
  If nothing is listed, enable the gates in any order. A listed node that is
  `Ready` and not shutting down carries stale state: if its kubelet is running
  with the gate, the kubelet corrects it on its next status update; otherwise
  clear it before enabling the reader. A listed node whose kubelet is gone will
  be suppressed once the reader is on, which is harmless because nothing runs
  there; such nodes are normally deleted by the cloud controller or the
  autoscaler, and deletion clears them.
- **Upgrade order (only if the preflight lists nodes).** Enable the kubelet gate
  on all nodes first and the kube-controller-manager gate last. Every live
  kubelet with the gate corrects its own node on its next status update, so by
  the time the reader is on, the only `True` values left are on nodes that are
  shutting down or whose kubelet is gone.
- **Downgrade / disable order.** Disable the kube-controller-manager
  gate first, then the kubelet gate or version. Reader-first guarantees that a
  condition a downgraded kubelet can no longer clear has no effect. Conditions
  left `True` on a node mid-shutdown at the moment of disablement are cleared
  by the kubelet's next startup (if it still has the gate); otherwise they stay
  until an administrator clears them or the Node is deleted, and are inert
  while the reader is off.

### Version Skew Strategy

Fail-open semantics make every skew combination safe:

| Kubelet | kube-controller-manager | Result |
|---|---|---|
| New (writes conditions) | Old (ignores them) | Conditions present, unconsumed. Today's churn persists on shutting-down nodes; no new failure mode. Kubelet startup clears the conditions. |
| Old (never writes) | New (would consume) | No kubelet-written conditions exist. DaemonSet behavior unchanged unless conditions pre-exist (see preflight); no new failure mode. |
| Both new, gate off | — | No change. |
| Both new, gate on | — | Feature works. |

The table covers the supported n-3 kubelet skew: an older kubelet never writes
the conditions, so a newer kube-controller-manager observes absent conditions
and the DaemonSet controller behaves exactly as today. A newer kubelet against
an older control plane publishes conditions that nothing consumes. "No new
failure mode" is the claim in every row, not that the status quo is acceptable
— the status quo is the churn this KEP exists to fix, and it persists in every
partial combination.

## Production Readiness Review Questionnaire

### Feature Enablement and Rollback

###### How can this feature be enabled / disabled in a live cluster?

- [x] Feature gate (also fill in values in `kep.yaml`)
  - Feature gate name: `DaemonSetGracefulNodeShutdown`
  - Components depending on the feature gate: kubelet, kube-controller-manager

###### Does enabling the feature change any default behavior?

Yes, on nodes undergoing graceful shutdown only. The DaemonSet controller stops
recreating DaemonSet Pods on those nodes for the duration of the shutdown. The
kubelet's admission behavior is unchanged. Nodes not in shutdown are
unaffected.

###### Can the feature be disabled once it has been enabled (i.e. can we roll back the enablement)?

Yes. With the gate off, no component writes or honors the conditions. Any
condition left `True` at the moment of disablement is inert. It is cleared by
the kubelet at its next startup if the kubelet still has the gate; otherwise by
an administrator or by Node deletion. Disable the
kube-controller-manager reader before downgrading kubelets; see [Upgrade /
Downgrade Strategy](#upgrade--downgrade-strategy).

###### What happens if we reenable the feature if it was previously rolled back?

Behavior resumes on the next shutdown event. No state migration.

###### Are there any tests for feature enablement/disablement?

Yes. Unit tests in `pkg/controller/daemon/` and `pkg/kubelet/nodeshutdown/`
exercise each path with the gate enabled and disabled via
`featuregatetesting.SetFeatureGateDuringTest`. The integration test in
`test/integration/daemonset/` runs the suppression scenario with the gate on
and asserts today's behavior with it off. A gate off → on → off transition test
on the DaemonSet reader verifies that toggling the gate leaves no dangling
creation expectations and that suppression stops immediately on disable. A
reader-only enablement test starts with `GracefulNodeShutdownInProgress`
already `True` on a `Ready` node (as an administrator would write it) and
asserts suppression,
documenting that the reader is writer-agnostic and that the preflight check is
what protects against stale state.

### Rollout, Upgrade and Rollback Planning

###### How can a rollout or rollback fail? Can it impact already running workloads?

Rollout enables a writer and a reader that are each inert without the other's
conditions, so a partial rollout does not fail in a way that affects running
workloads: the worst case on any component is today's behavior. The same holds
for Pods not yet running. The kubelet's rejection of new Pod admission during
a shutdown is existing Graceful Node Shutdown behavior that this KEP does not
change; with only the kubelet side enabled, DaemonSet Pods targeted at a
shutting-down node are rejected exactly as today, and the node's next kubelet
startup clears the conditions before the node reports `Ready`. A partially
enabled cluster therefore cannot block an upgrade.

The reader is writer-agnostic — it keys on `type` and `status` only, per API
conventions — so a node that already carries `GracefulNodeShutdownInProgress`
`True` when the reader is enabled is suppressed immediately, whoever wrote it.
For stale state that is not wanted, which is why [Upgrade / Downgrade
Strategy](#upgrade--downgrade-strategy) asks for a preflight listing before the
reader is enabled. On a cluster where nothing has written the condition, which
is every cluster today, the listing is empty and the gates can be enabled in
any order. Where it is not empty, kubelets go first, and every live kubelet
with the gate corrects its own node on its next status update, so no node needs
hand-clearing except one whose kubelet is gone, and those are normally deleted
by the cloud controller or the autoscaler.

Control-plane components run as static Pods are DaemonSet-independent and
unaffected. Control-plane components run as DaemonSets — some platforms do
this — are affected only on a node carrying `GracefulNodeShutdownInProgress`
`True`. During a real shutdown that changes nothing for them: the kubelet
already rejects every new Pod, so they cannot come back until the node does.
Outside a real shutdown the condition is stale. On a node whose kubelet is
alive, the kubelet corrects it on its next status update, so the suppression
lasts one heartbeat. On a node whose kubelet is dead, no Pod runs regardless.
The preflight catches stale state that predates enablement. Because the reader
runs in kube-controller-manager,
suppression can only occur while the control plane is serving, so the
break-glass levers in [Troubleshooting](#troubleshooting) are always reachable
when the feature is active; if the API server is unreachable the controller is
inert, exactly as today.

###### What specific metrics should inform a rollback?

`daemonset_controller_node_shutdown_suppression_total{transition="enter"}`
rising while no node in the cluster is shutting down would indicate stale or
incorrect conditions. `enter` counts with no matching `exit` over a long window
indicate conditions that are not being cleared.

###### Were upgrade and rollback tested? Was the upgrade->downgrade->upgrade path tested?

Not yet; alpha. The integration tests cover gate on and gate off; an explicit
upgrade->downgrade->upgrade test is a beta requirement.

###### Is the rollout accompanied by any deprecations and/or removals of features, APIs, fields of API types, flags, etc.?

No.

### Monitoring Requirements

###### How can an operator determine if the feature is in use by workloads?

Presence of `GracefulNodeShutdownInProgress` (with `DrainInProgress`) on Node
objects, and a non-zero suppression counter.

###### How can someone using this feature know that it is working for their instance?

- [ ] Events
- [x] API .status
  - Condition name: `GracefulNodeShutdownInProgress` and `DrainInProgress`
    with reason `NodeShutdown` on a shutting-down Node.
- [x] Other (treat as last resort)
  - Details: `daemonset_controller_node_shutdown_suppression_total` increments
    with `transition="enter"` when the shutdown begins and `transition="exit"`
    when the node recovers, and the create/reject/delete churn in the audit log
    or `kubectl get events` stops.

###### What are the reasonable SLOs (Service Level Objectives) for the enhancement?

For alpha: no DaemonSet Pod admission is rejected with reason `NodeShutdown` on
a node after `GracefulNodeShutdownInProgress=True` was published there, beyond
the one-sync informer propagation window.
Formal SLOs will be set at beta once alpha metrics establish a baseline for
suppression counts and propagation latency.

###### What are the SLIs (Service Level Indicators) an operator can use to determine the health of the service?

- [x] Metrics
  - Metric name: `daemonset_controller_node_shutdown_suppression_total`
  - Components exposing the metric: kube-controller-manager
- [ ] Other (treat as last resort)

###### Are there any missing metrics that would be useful to have to improve observability of this feature?

A writer-side metric (count and latency of the kubelet's condition write, and
whether it landed inside the inhibit window) would help; it belongs to the
condition-writing milestone and is deferred.

### Dependencies

###### Does this feature depend on any specific services running in the cluster?

- The `NodeLifecycleConditions` API constants (v1.37+).
- **For the kubelet writer:** the node OS's Graceful Node Shutdown gate must be
  enabled and functional — `GracefulNodeShutdown` (beta since v1.21) on Linux
  with systemd-logind reachable and the inhibitor lock acquired, or
  `WindowsGracefulNodeShutdown` (beta since v1.34) on Windows with the kubelet
  running as a Windows service.
- **The DaemonSet reader** depends only on the conditions being present on the
  Node object.

### Scalability

###### Will enabling / using this feature result in any new API calls?

One Node status update per shutdown event (setting the conditions), one on
cancel, and one at kubelet startup if clearing is needed. The startup clear can
be coalesced into the kubelet's initial status update, and the per-status-update
reconciliation adds no calls because it rides on updates the kubelet already
makes. No control-plane component writes the conditions. Net effect is a
**reduction** in API
calls, since the create/reject/delete churn is eliminated — per [#137895], that
churn can reach hundreds of cycles for a single Pod on a single node removal.

###### Will enabling / using this feature result in introducing new API types?

No.

###### Will enabling / using this feature result in any new calls to the cloud provider?

No.

###### Will enabling / using this feature result in increasing size or count of the existing API objects?

Up to two additional entries in `Node.status.conditions` on shutting-down
nodes only, plus one field in the kubelet's local shutdown state file.

###### Will enabling / using this feature result in increasing time taken by any operations covered by existing SLIs/SLOs?

No. The kubelet's condition write is never waited on and cannot delay
shutdown; the DaemonSet controller's per-node check is one condition lookup on
an object it already holds.

###### Will enabling / using this feature result in non-negligible increase of resource usage (CPU, RAM, disk, IO, ...) in any components?

No. Net resource usage on the API server and kube-controller-manager decreases
because the churn is eliminated.

###### Can enabling / using this feature result in resource exhaustion of some node resources (PIDs, sockets, inodes, etc.)?

No.

### Troubleshooting

###### How does this feature react if the API server and/or etcd is unavailable?

The kubelet's write is best-effort and never waited on; it fails and the
shutdown proceeds. The DaemonSet controller sees absent conditions and behaves as today.

###### What are other known failure modes?

- Stale `True` conditions on a node whose kubelet died and has not returned.
  DaemonSet Pods are not recreated there until cleared, and nothing else runs
  there either. Cleared by the kubelet when it returns, by an administrator, or
  by Node deletion. This is the alpha assumption for a node that never comes
  back; see [Kubelet (writer)](#kubelet-writer).
- Stale `True` conditions on a node that returned to `Ready` under a kubelet
  that lacks the startup clear — the node was upgraded with the gate on, shut
  down, and rebooted into a downgraded kubelet. The condition stays `True`
  until an administrator clears it or the node's kubelet is upgraded again.
  Disabling the kube-controller-manager reader before downgrading kubelets (see
  [Upgrade / Downgrade Strategy](#upgrade--downgrade-strategy)) prevents any
  effect.
- Stale `True` condition on a live node with the gate (for example, the cancel
  write failed, or someone set the condition by hand). The kubelet corrects it
  on its next status update. If it stays, check that the kubelet has the gate
  and is reporting status.
- Conditions set by an administrator or another writer on a node whose kubelet
  lacks the gate or is not running. Suppression on that node lasts until the
  writer clears them. The preflight check in [Upgrade / Downgrade
  Strategy](#upgrade--downgrade-strategy) lists such nodes before the reader is
  enabled.
- A kubelet-written `DrainInProgress` left `True` after a reboot because the
  kubelet was stopped before it recorded the write in its state file. Admin
  remediation: clear the condition. The reader ignores `DrainInProgress`, so
  DaemonSet behavior is unaffected.
- `GracefulNodeShutdown` misconfigured (e.g. empty priority list degrades GNS to
  a no-op): no shutdown signal reaches the manager, no conditions are written,
  today's behavior.

###### What steps should be taken if SLOs are not being met to determine the problem?

1. Confirm the conditions are present on the shutting-down Node:
   `kubectl get node <name> -o jsonpath='{.status.conditions[?(@.type=="GracefulNodeShutdownInProgress")]}'`.
   If absent, check that the gate is enabled on the kubelet and that
   `GracefulNodeShutdown` is active on the node (systemd-logind reachable,
   inhibitor lock held) — see kubelet logs for the shutdown manager.
2. Confirm the gate is enabled on kube-controller-manager.
3. Check that `daemonset_controller_node_shutdown_suppression_total{transition="enter"}`
   incremented when the shutdown began; if it did not, the DaemonSet controller
   is not observing the conditions (informer lag or gate off).
4. If `GracefulNodeShutdownInProgress` is stale `True` on a node that is not
   shutting down: if the kubelet is running with the gate, it should correct
   the value on its next status update, so check the kubelet's gate and logs.
   If the kubelet is gone, clear the condition by hand or delete the Node.

**Break-glass.** Two properties bound the worst case. The reader *is* the
DaemonSet controller, so if the API server is unreachable the controller neither
suppresses nor creates — exactly as today, with or without this feature — and
recovery is the cluster's existing bootstrap path; static Pods are never gated
by the conditions. And the kubelet's rejection of new Pods applies only to a
node actually shutting down ([KEP-2000] behavior, unchanged here); on a healthy
node carrying a stuck condition the kubelet admits normally and the DaemonSet
controller is the only thing holding Pods back. Suppression affects creation
only: a crash-looping Pod is
restarted by the kubelet and is untouched. To release a stuck node, in order of
least knowledge required:

- **Restart the kubelet on the node, if it has stopped reporting status.** A
  running kubelet with the gate corrects the condition on its next status
  update by itself; if it is wedged, a restart runs the startup clear, which
  sets `GracefulNodeShutdownInProgress` to `False`, and the DaemonSet
  controller recreates on the next sync. Node-local; no API knowledge needed.
- **Clear the condition on the node's status subresource** (NodeRestriction
  permits this for administrators):
  `kubectl patch node <name> --subresource=status --type=strategic -p '{"status":{"conditions":[{"type":"GracefulNodeShutdownInProgress","status":"False","reason":"AdminRequested"}]}}'`.
  The reader keys on `True`, so `False` releases the node on the next sync.
  Clear `DrainInProgress` the same way if the kubelet wrote it and did not get
  to clear it.
- **Cluster-wide:** disable `DaemonSetGracefulNodeShutdown` on
  kube-controller-manager. Suppression stops on the next sync of every
  DaemonSet; no state needs cleaning up and kubelets need no change.

Automatic clearing for a node whose kubelet never returns is the
vanished-writer item listed under [Beta](#beta).

## Implementation History

- **2026-07:** Enhancement issue #6249 filed and scoped in WG Node Lifecycle;
  standalone KEP linked to [KEP-5683]; both kubelet and NLC as writers agreed.
- **2026-08-10:** WG agrees process (short design draft first) and alpha
  assumptions (admin in control, best-effort writes, no locking).
- **2026-08-17:** WG resolves startup-clear (Q1) and scope of drain ordering
  (Q2).
- **2026-08-24:** WG resolves single write + AND (Q3), separate feature gate
  (Q4), `maxUnavailable` out of scope (Q5); requires priorities/ordering
  section.
- **2026-09-08:** First KEP draft; WG lead approves opening the draft PR.
- **2026-09-11:** KEP PR [kubernetes/enhancements#6351](https://github.com/kubernetes/enhancements/pull/6351)
  opened for SIG Node, SIG Apps, and PRR review.
- **2026-09-21:** SIG Apps review (@janetkuo): reader extended to
  `rollingUpdate` and `shouldIgnoreNodeUpdate`; metric moved to the
  node-update worker; @janetkuo added as approver.
- **2026-09-24:** WG lead files [#6430] for multi-writer coordination; this
  KEP's coordination placeholders now point there.
- **2026-09-25:** SIG Apps review (@soltysh): clearing semantics made explicit
  (kubelet sets `False`, entries never removed); a pre-existing
  `DrainInProgress=True` is left untouched by the kubelet; gate depends on
  `NodeLifecycleConditions`.
- **2026-09-25:** API review (@deads2k), direction confirmed by the WG lead:
  tier-aware kubelet admission and a third condition,
  `GracefulNodeShutdownCriticalPhase`, added so critical DaemonSet Pods could
  be recreated during lower tiers.
- **2026-09-28:** SIG Node review (@SergeyKanzhelev, @mrunalp): the third
  condition and the admission change are withdrawn in favor of the two
  existing conditions and no special case for one priority stage; the startup
  clear of `DrainInProgress` is made ownership-aware through the shutdown
  state file.
- **2026-09-29:** SIG Apps (@soltysh, @atiratree), PRR (@kannon92), and the
  WG lead (@rthallisey): reader keys on `GracefulNodeShutdownInProgress`
  alone; the Node Lifecycle Controller writer is dropped and Node deletion is
  the end state for a node that never returns; the kubelet reconciles its
  condition on every status update; priority-aware admission is tracked as
  [k/k#142521] and kept off this KEP's graduation path.

## Drawbacks

- Adds an in-tree writer to conditions whose API doc comment currently says the
  admin owns them; requires an api-approver and a semantics change to that
  comment.
- Alpha accepts residual stale-condition and multi-writer risk under the
  admin-in-control assumption; these are real operational sharp edges until the
  graduation items land.
- A node whose kubelet never returns keeps its conditions until an
  administrator clears them or the Node is deleted. Alpha has no automatic
  backstop for that case.
- The startup clear of `DrainInProgress` depends on a record in the kubelet's
  state file that can be missing if the kubelet was stopped before writing it.
- Critical DaemonSet Pods get no special treatment. That matches the kubelet's
  reject-all admission today; if [k/k#142521] changes admission, the reader
  will need to follow.

## Alternatives

The reporter of [k/k#122912] enumerated four candidate fixes. This KEP
implements **option 2**. The others:

- **Option 1 — "Do not remove DaemonSets from the node that is due to be shut
  down."** This is already what several operators do out of tree: drain
  implementations that hard-skip DaemonSet Pods so they are never evicted in the
  first place, sometimes paired with a host reboot after drain to force
  DaemonSet teardown. It does not solve the problem. The controller is still
  told nothing, so any DaemonSet Pod that dies for an unrelated reason during
  the shutdown window — OOM, crash, rollout — re-enters the churn loop;
  attribution in DaemonSet status is unimproved; It also cannot help the
  rejected-corpse cleanup that [#122912] asks for.
- **Option 3 — Add a new taint type and teach the DaemonSet controller to
  respect it.** The DaemonSets that churn during shutdown carry blanket
  `operator: Exists` tolerations by convention (see [Motivation](#motivation));
  [#122912] itself notes that ignoring node status and "most of the default node
  taints" is "the key feature of a DaemonSet." A new shutdown taint would be
  tolerated by exactly the workloads it needs to stop, unless the DaemonSet
  controller were taught to ignore that toleration for that key — which is a
  change to the toleration contract, not a taint. Taints are also policy
  (repel), whereas the need here is observation. Note the Node Lifecycle
  Controller already applies `not-ready` taints during shutdown today and they
  do not help, for the same reason.
- **Option 4 — Teach the Shutdown Manager to ignore DaemonSets until the very
  last moment before eviction.** Reduces the window but does not close it — the
  controller still recreates once those Pods are terminated — and moves the fix
  into the kubelet, which cannot stop the controller from creating. Retained as
  input to the priority work in [k/k#142521].

Other alternatives considered:

- **Rely on `Ready=False`.** Indistinguishable from runtime failure; that
  ambiguity is the bug.
- **Rely on `failedPodsBackoff`.** Already exists; bounds the flood but does not
  stop the churn, fix attribution, or help rollout verifiers.
- **Require `DrainInProgress=True` as well (the AND).** The first drafts did
  this, following [KEP-5683] Story 1, so that a plain drain would never
  suppress. Dropped at KEP review (2026-09-29): a plain drain never sets
  `GracefulNodeShutdownInProgress`, so that condition alone already scopes the
  suppression; the kubelet writes both in one update, so the second condition
  adds no information for the DaemonSet controller; and keying on the
  kubelet-only condition gives the reader a single writer. The kubelet still
  writes `DrainInProgress` for other readers.
- **Node Lifecycle Controller as the asserting writer.** The controller cannot
  observe shutdown; only the kubelet can. Publishing must originate at the
  kubelet to honor the admission-rejection invariant.
- **Node Lifecycle Controller clears the conditions when `Ready` goes
  `Unknown`.** Carried in earlier drafts, writing `status=Unknown`. Dropped at
  KEP review (2026-09-29). There is no signal that a shutdown has finished, so
  the controller was guessing, and on architectures where the Node object
  outlives the kubelet it guessed wrong and released DaemonSet Pods onto a node
  still being torn down. `Unknown` was also not useful to readers, which need
  `True` or `False`. Node deletion is the end state instead; see [Kubelet
  (writer)](#kubelet-writer).
- **Exempt `system-node-critical` / `system-cluster-critical` DaemonSet Pods
  from suppression.** Not adopted: the kubelet rejects every new Pod during a
  shutdown today, so an exempted critical Pod would be created, rejected, and
  looped for the whole window, reintroducing the churn for precisely the
  population [#137895] reports.
- **Priority-aware admission and reader.** At API review (2026-09-25) the KEP
  briefly carried a tier-aware kubelet admission change plus a third condition,
  `GracefulNodeShutdownCriticalPhase`, so that critical DaemonSet Pods could be
  recreated while the kubelet was still terminating lower tiers. Withdrawn at
  SIG Node review (2026-09-28): it special-cased one priority stage, it invited
  a per-stage condition scheme that every reader would have to learn, and the
  critical stage can be so short that the write never lands. The kubelet's
  reject-all admission is now tracked as its own bug, [k/k#142521]. A
  kubelet-owned Node status field carrying the priority boundary
  (`HighestPriorityAllowed`, [proposed at KEP review][atiratree-field]) is a
  candidate design for that work. It is not on this KEP's graduation path:
  this KEP stays on the condition from alpha through GA rather than switching
  APIs between stages, and if the priority work needs a new Node API it should
  be its own KEP. For reference, the kubelet today, pinned to
  [kubernetes/kubernetes@e79a603][k/k-e79a603]: [`migrateConfig`][k/k-migrateconfig]
  turns the configuration into a sorted list of priority tiers,
  [`killPods`][k/k-killpods] walks them from lowest to highest, and
  [`Admit`][k/k-admit] checks only whether shutdown has begun.
- **Reuse the `NodeLifecycleConditions` gate.** Changes the meaning of a gate
  that already shipped and couples this behavior's maturity to the admin-managed
  conditions and the `kubectl drain` writer. *Rejected in WG (2026-08-24).*
- **Separate writer and reader gates.** Retained as a fallback if SIG Node wants
  decoupled enablement; not the default, to minimize gate count.
- **A kubelet configuration setting rather than a feature gate**, as floated in
  [#122912] ("some of the above could be controlled via a new setting for
  kubelet, like the behaviour that controls whether to evict DaemonSets and
  when"). A gate is preferred for an alpha behavior change; a durable setting
  can be revisited if operators need per-node control after graduation.
- **Change `nodeShouldRunDaemonPod` instead of `podsShouldBeOnNode`.** Would
  move `desiredNumberScheduled` and hide the unavailability from status; status
  attribution is [KEP-6250]'s territory.
- **Key the reader on the condition `reason` (e.g. require `NodeShutdown`) so
  that only kubelet-written state suppresses.** Rejected: API conventions
  reserve `reason` for a machine-readable explanation of the current status,
  not for control flow; a controller that branches on it turns `reason` into a
  state machine. The reader keys on `type` and `status` only. Partial-rollout
  and admin-written-state safety are handled operationally instead (see
  [Upgrade / Downgrade Strategy](#upgrade--downgrade-strategy)).
- **Additionally require the node's `Ready` condition not be `True` before
  suppressing**, as a second signal that the kubelet is really in shutdown.
  Not adopted for alpha: it reintroduces the asynchronous `Ready=False`
  ordering window the writer design closes, and it changes the meaning of an
  admin-written condition on a `Ready` node. Recorded as a candidate beta
  hardening for stale-state safety if the alpha metric shows suppression on
  nodes that are not shutting down.

## Infrastructure Needed (Optional)

None.

[k/k#122674]: https://github.com/kubernetes/kubernetes/issues/122674
[k/k#142521]: https://github.com/kubernetes/kubernetes/issues/142521
[kubelet-files-gns]: https://kubernetes.io/docs/reference/node/kubelet-files/#graceful-node-shutdown
[atiratree-field]: https://github.com/kubernetes/enhancements/pull/6351#discussion_r4132862002
[k/k#122912]: https://github.com/kubernetes/kubernetes/issues/122912
[k/k#137895]: https://github.com/kubernetes/kubernetes/issues/137895
[KEP-5683]: https://github.com/kubernetes/enhancements/issues/5683
[KEP-5683-alpha2]: https://github.com/kubernetes/enhancements/blob/master/keps/sig-node/5683-lifecycle-conditions/README.md#alpha2---consume-conditions-in-controllers
[#6430]: https://github.com/kubernetes/enhancements/issues/6430
[k/k-e79a603]: https://github.com/kubernetes/kubernetes/tree/e79a603500e18bfb713af5dc4e57f8fe39514ab1
[k/k-migrateconfig]: https://github.com/kubernetes/kubernetes/blob/e79a603500e18bfb713af5dc4e57f8fe39514ab1/pkg/kubelet/nodeshutdown/nodeshutdown_manager.go#L237-L259
[k/k-killpods]: https://github.com/kubernetes/kubernetes/blob/e79a603500e18bfb713af5dc4e57f8fe39514ab1/pkg/kubelet/nodeshutdown/nodeshutdown_manager.go#L129-L226
[k/k-admit]: https://github.com/kubernetes/kubernetes/blob/e79a603500e18bfb713af5dc4e57f8fe39514ab1/pkg/kubelet/nodeshutdown/nodeshutdown_manager_linux.go#L117-L128
[pkg/features/kube_features.go]: https://github.com/kubernetes/kubernetes/blob/master/pkg/features/kube_features.go
[KEP-6250]: https://github.com/kubernetes/enhancements/issues/6250
[KEP-6251]: https://github.com/kubernetes/enhancements/issues/6251
[#122912]: https://github.com/kubernetes/kubernetes/issues/122912
[#137895]: https://github.com/kubernetes/kubernetes/issues/137895
[pkg/kubelet/nodeshutdown/]: https://github.com/kubernetes/kubernetes/tree/master/pkg/kubelet/nodeshutdown
[pkg/kubelet/nodestatus/]: https://github.com/kubernetes/kubernetes/tree/master/pkg/kubelet/nodestatus
[pkg/controller/daemon/]: https://github.com/kubernetes/kubernetes/tree/master/pkg/controller/daemon
[test/integration/daemonset/]: https://github.com/kubernetes/kubernetes/tree/master/test/integration/daemonset
[test/e2e_node/]: https://github.com/kubernetes/kubernetes/tree/master/test/e2e_node
[staging/src/k8s.io/api/core/v1/types.go]: https://github.com/kubernetes/kubernetes/blob/master/staging/src/k8s.io/api/core/v1/types.go
[nodeshutdown_manager_linux.go]: https://github.com/kubernetes/kubernetes/blob/master/pkg/kubelet/nodeshutdown/nodeshutdown_manager_linux.go
[KEP-2000]: https://github.com/kubernetes/enhancements/issues/2000
[KEP-4802]: https://github.com/kubernetes/enhancements/issues/4802
[kubernetes/enhancements]: https://github.com/kubernetes/enhancements
[kubernetes/website]: https://github.com/kubernetes/website
[kubernetes.io]: https://kubernetes.io
[Conformance Tests]: https://git.k8s.io/community/contributors/devel/sig-architecture/conformance-tests.md
[all GA Endpoints]: https://github.com/kubernetes/community/pull/1806
