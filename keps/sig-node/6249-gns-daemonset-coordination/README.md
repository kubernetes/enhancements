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
  - [Node Lifecycle Controller (narrow writer)](#node-lifecycle-controller-narrow-writer)
  - [DaemonSet controller (reader)](#daemonset-controller-reader)
  - [Other writers](#other-writers)
  - [Feature gating](#feature-gating)
  - [Metrics](#metrics)
  - [Priority tiers](#priority-tiers)
    - [How the kubelet terminates Pods today](#how-the-kubelet-terminates-pods-today)
    - [Tier-aware admission](#tier-aware-admission)
    - [The critical-phase condition](#the-critical-phase-condition)
    - [Alpha approximation and graduation path](#alpha-approximation-and-graduation-path)
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
Pods on a node while both conditions are `True`. The Node Lifecycle Controller
sets the conditions to `Unknown` in the single case where the kubelet can no
longer reset them itself.

Two further changes, taken at API review (2026-09-25), keep the reader
consistent with how Graceful Node Shutdown actually terminates Pods. The
kubelet's shutdown admission becomes priority-tier-aware: it rejects only Pods
whose priority falls in the tier being terminated or one already terminated,
instead of every Pod. And the kubelet publishes a third condition,
`GracefulNodeShutdownCriticalPhase=True`, when termination reaches the critical
tier. The DaemonSet controller suppresses non-critical templates while the first
two conditions are `True` and every template once the third is, so that under
the default two-tier configuration it never withholds a Pod the kubelet would
admit. See [Priority tiers](#priority-tiers), including the approximation this
implies for multi-tier `shutdownGracePeriodByPodPriority` ladders.

One new `NodeConditionType` constant is introduced; no new API types or reason
values. The change is the first in-tree writer and the first in-tree reader of
conditions that are, as of v1.37, admin-managed only.

This is **option 2 of the four fixes enumerated by the reporter of
[k/k#122912]** ("teach the DaemonSet controller about `NodeShutdown` status for
a node so that it avoids attempting to schedule Pods there"). The other three
are addressed in [Alternatives](#alternatives).

This KEP is the DaemonSet-controller item of [KEP-5683]'s [alpha-2
milestone][KEP-5683-alpha2] ("consume conditions in controllers"), carried as a
standalone KEP so that it can graduate on its own feature gate.

The DaemonSet **reader** keys on the conditions' `type` and `status` only, not
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
> termination tier. The relationship between this KEP and the kubelet's
> priority-tiered termination, and the admission change this KEP makes, are in
> [Priority tiers](#priority-tiers).

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
  conditions, and a new `GracefulNodeShutdownCriticalPhase` condition when
  termination reaches the critical tier.
- The kubelet's shutdown admission rejects only the Pods its priority-tiered
  termination would terminate, and the DaemonSet controller suppresses only
  what the kubelet would reject — exactly under the default two-tier
  configuration, conservatively under multi-tier ladders.
- The DaemonSet controller stops creating Pods on a node while that node is in
  graceful shutdown, without disturbing Pods the kubelet is already terminating
  and without hiding the resulting unavailability from DaemonSet status.
- Rejected DaemonSet Pod corpses on a shutting-down node are cleaned up without
  being replaced, removing a documented source of manual operator toil.
- The reader's contract is defined in terms of condition `type` and `status`
  only, consistent with [KEP-5683]'s admin-managed condition model, so that it
  remains correct as additional writers are introduced by sibling KEPs.
- The conditions are cleared on every path where the kubelet can clear them
  (shutdown cancelled, kubelet restart) and by the Node Lifecycle Controller in
  the single case where it cannot (kubelet gone, node taken offline).
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
- **Per-tier precision under `shutdownGracePeriodByPodPriority`.** Alpha
  distinguishes two phases, critical and everything below it. Publishing the
  exact tier boundary so the reader is precise for arbitrary tier ladders is a
  [beta criterion](#beta); see [Priority tiers](#priority-tiers). General
  drain ordering across the SLM readers remains with the WG lead's separate
  enhancement.
- **General stuck-Pod cleanup for non-DaemonSet workloads** described in
  [#122912].
- **Multi-writer coordination, ownership, or locking** for node lifecycle
  conditions. Alpha accepts last-writer-wins under the WG's admin-in-control /
  best-effort assumptions, with one rule recorded in [Kubelet
  (writer)](#kubelet-writer): the kubelet never overwrites a
  `DrainInProgress=True` it did not set. The general mechanism, shared by the
  sibling KEPs that need it, is [#6430] (SLM: Coordinate Node lifecycle
  condition writers).
- **Sanctioning controller-driven node teardown as a writer** of
  `GracefulNodeShutdownInProgress`. Whether, and via which conditions, teardown
  orchestrated by a cluster-side controller should trigger DaemonSet suppression
  is a reader question for beta, taken together with [KEP-6250] / [KEP-6251] and
  the coordination work; see [Other writers](#other-writers).
- **Fixing the existing kubelet shutdown-state "amnesia" bug** ([k/k#122674]).
  This KEP adds a new edge case to that bug and commits to handling it on the
  graduation path, but does not attempt the general fix in alpha.
- **Graduating `GracefulNodeShutdown`** (beta since v1.21) or changing its
  termination ordering. The admission change in [Priority
  tiers](#priority-tiers) narrows which Pods are rejected during a shutdown; it
  does not change which Pods are terminated or in what order.

## Proposal

The design has three pieces. All of them follow the WG's alpha assumptions: the
cluster administrator remains in control of node lifecycle conditions, every
write is best-effort, and there is no ownership locking between writers.

1. **Kubelet (writer).** On receiving the shutdown signal, before terminating
   any Pod, the shutdown manager publishes `GracefulNodeShutdownInProgress=True`
   and `DrainInProgress=True` in a single Node status update. If
   `DrainInProgress` is already `True` — set by `kubectl drain` or another
   writer — the kubelet leaves that entry untouched and writes only
   `GracefulNodeShutdownInProgress`. When termination reaches the critical
   tier it publishes `GracefulNodeShutdownCriticalPhase=True`. Its admission
   handler rejects only Pods whose priority falls in the tier being terminated
   or one already terminated. It sets the conditions it wrote back to `False`
   if the shutdown is cancelled, and at startup sets any lingering `True` value
   on any of the three conditions to `False` before reporting `Ready`.
2. **DaemonSet controller (reader).** While a node has
   `GracefulNodeShutdownInProgress=True` and `DrainInProgress=True`, the
   controller creates no non-critical DaemonSet Pods on that node; once
   `GracefulNodeShutdownCriticalPhase` is also `True`, it creates none at all
   (*suppression*, below). It still deletes failed DaemonSet Pods on the node
   without replacing them, and does not touch running ones.
3. **Node Lifecycle Controller (NLC; narrow writer).** Never asserts `True`.
   Sets the conditions to `Unknown` (reason `NodeStatusUnknown`) when the
   kubelet has been lost and the node is taken offline, so a node that dies
   mid-shutdown does not carry stale state into removal or recovery.

Only `status: "True"` has behavioral effect. `False`, `Unknown`, and an absent
condition are all equivalent to "no shutdown in progress" for every reader.

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

*This story is not served by alpha. It is recorded here because it motivates the
beta reader question — which condition combination, if any, should trigger
DaemonSet suppression for controller-driven teardown — deferred in [Other
writers](#other-writers) and carried as a beta graduation criterion.*

### Notes/Constraints/Caveats

- **The kubelet's write must land inside the logind inhibit window.** The
  kubelet holds a delay inhibitor lock; once it releases the lock (after
  `killPods` returns), systemd proceeds regardless. An unreachable API server
  must never delay the shutdown itself, so the write is best-effort with a
  bounded retry, and the Node Lifecycle Controller is the backstop.
- **The kubelet's shutdown state is in-memory only.** `nodeShuttingDownNow` is a
  boolean behind a mutex; a restarted kubelet cannot know it was mid-shutdown.
  Unconditional clearing at startup is what makes this loss a non-issue for
  alpha.
- **Conditions observe; taints enforce.** This KEP uses conditions as an
  observation primitive that a controller reads. The conditions do not affect
  scheduling or admission; the only behavior they drive is whether the
  DaemonSet controller creates a Pod. The admission change in [Priority
  tiers](#priority-tiers) is a change to Graceful Node Shutdown's own handler,
  keyed on the kubelet's in-memory tier state, not on the conditions.
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
| The kubelet's status write does not land before the machine dies (slow or unreachable API server during a rack-wide event). | Best-effort, bounded retry; the DaemonSet controller sees absent conditions and behaves as today. Never worse than status quo. |
| Conditions left `True` after the kubelet is gone (crash, power loss). | Kubelet clears unconditionally at startup on return; Node Lifecycle Controller clears when the node is taken offline. Residual stale state until one of those fires is accepted for alpha; a stale-writer backstop is a graduation item. |
| The NLC clears the conditions while teardown is still in progress, on architectures where the Node object intentionally outlives the kubelet (stop kubelet → post-kubelet cleanup → delete Node). | Accepted for alpha; with no kubelet there is no admission rejection, so Pods sit `Pending` rather than churning, and Node deletion resolves it. The trigger is [Design Decision 2](#design-decisions); see the [NLC section](#node-lifecycle-controller-narrow-writer) for the trade-offs. |
| Another writer flips `DrainInProgress` mid-shutdown (`kubectl drain`, a maintenance operator, an admin). | Last-writer-wins, admin-in-control. Fail-open semantics mean the worst case is today's churn, not incorrect deletion. The kubelet never overwrites a `DrainInProgress=True` it did not set, so the initiating writer's `reason`, `message`, and `lastTransitionTime` survive the shutdown. General cross-writer coordination is [#6430]. |
| A reboot resets a `DrainInProgress=True` that another writer set before the shutdown: the kubelet's startup reset is unconditional and writer-agnostic. | Accepted for alpha. The drain initiator observes the reset and re-asserts if its drain is still in progress; a startup reset that distinguishes writers is an outcome of [#6430]. |
| An admin deletes a condition the kubelet is responsible for while shutdown is in progress (new edge case on [k/k#122674]). | Accepted for alpha. Graduation direction: the kubelet reads level-triggered shutdown state from the OS and continuously reconciles the conditions (WG, 2026-08-17). |
| Under `shutdownGracePeriodByPodPriority` with a tier between the default tier and `SystemCriticalPriority`, a template in a tier the kubelet has not reached yet is suppressed while the kubelet would still admit it. | Over-suppression only: no churn and no rejected Pod; the template's Pod returns when the conditions clear. The window is bounded by the lower tiers' grace periods, which are operator-chosen and can be long ([k/k#122912]'s reproduction configures `shutdownGracePeriod: 3600s`). Exact for the legacy two-tier configuration. Publishing the tier boundary for per-tier precision is a [beta criterion](#beta); see [Priority tiers](#priority-tiers). |
| Tier-aware admission changes Graceful Node Shutdown behavior: a Pod admitted during a lower tier runs only until its own tier is reached. | Intended, and the same fate as a Pod admitted before the shutdown signal. Gated with the rest of the feature, so rollback restores today's reject-all admission. |
| Informer propagation race: the DaemonSet controller may issue one create between the kubelet's write and the controller observing it. | Publishing the conditions before terminating Pods bounds the recreate loop to at most ~one controller sync instead of unbounded. |

## Design Details

### The conditions

`GracefulNodeShutdownInProgress` and `DrainInProgress` already exist: the node
lifecycle `NodeConditionType` constants merged into `k8s.io/api/core/v1` in
v1.37 behind the `NodeLifecycleConditions` feature gate (alpha, default off). As
of v1.37 nothing in core writes or reads them. This KEP adds the first in-tree
writer and reader, plus one new constant of the same kind,
`GracefulNodeShutdownCriticalPhase`; no new API types or reason values.

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

And, in a second update when termination reaches the critical tier:

```yaml
  - type: GracefulNodeShutdownCriticalPhase
    status: "True"
    reason: NodeShutdown
    message: "Graceful node shutdown is terminating critical pods"
```

**Why two conditions.** `DrainInProgress` means Pods are being evicted from the
node. It is set by `kubectl drain` today and by maintenance operators in sibling
KEPs, and DaemonSet Pods are expected to keep running through those drains — a
node-level logging or networking agent must be recreated if it dies mid-drain,
not suppressed. `GracefulNodeShutdownInProgress` means the kubelet has received
a shutdown signal and is rejecting new Pods in the tiers it is terminating, so
recreating such a Pod there can only churn. The DaemonSet reader therefore acts
only when **both** are `True`: the drain that warrants suppression is the one
caused by the node going away. Either condition alone leaves the controller
behaving as it does today. The third condition,
`GracefulNodeShutdownCriticalPhase`, refines *which* templates are suppressed
once the first two are `True`; it never triggers suppression on its own. The
reader-side consequences are in [DaemonSet controller
(reader)](#daemonset-controller-reader) and the rationale in [Priority
tiers](#priority-tiers).

**What "clear" means.** Throughout this document, clearing a condition means
setting `status` on the existing entry, never removing the entry from
`status.conditions`. The kubelet sets `False` (shutdown cancelled; kubelet
startup). The Node Lifecycle Controller sets `Unknown` (kubelet lost). No
component specified by this KEP deletes a condition entry; removal remains an
administrator action under [KEP-5683]'s admin-managed model.

Semantics for every reader in this KEP:

- Only `status: "True"` has behavioral effect (the *fail-open* rule).
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
   left as is; see below). The write is best-effort with a bounded retry: an
   unreachable API server must never delay the shutdown, and the write must
   complete inside the platform's shutdown window (the logind inhibit delay on
   Linux; the SCM preshutdown timeout on Windows). Today the shutdown manager records
   the shutdown in memory and then fires the kubelet's generic status sync in a
   goroutine it does not await (`go m.syncNodeStatus(...)` in
   [`nodeshutdown_manager_linux.go`][nodeshutdown_manager_linux.go]), so
   `Ready=False` may land after Pod termination has already begun. The condition
   write should instead be a dedicated, synchronous, bounded call issued before
   step 2, so that the conditions are ordered ahead of the first Pod termination
   rather than left to the status loop's timing.
2. **Begin Pod termination** per the existing [KEP-2000] priority-tiered
   ordering, with the tier-aware admission and the critical-phase publish
   described in [Priority tiers](#priority-tiers): the shutdown manager records
   the lowest priority it has not yet begun terminating, `Admit()` rejects only
   Pods below that boundary, and `GracefulNodeShutdownCriticalPhase=True` is
   published before the first Pod in the critical tier is terminated.

**Why a single write of both conditions.** Graceful Node Shutdown moves directly
from signal detection to Pod termination; verified against the shutdown manager,
there is no intermediate phase between "shutdown started" and "drain started."
*Resolved in WG (2026-08-24): accepted for alpha; to be corrected with SIG Node
if they see an issue.*

**Invariant (agreed with the issue author).** The signal that gates DaemonSet
Pod creation transitions at the same moment the kubelet begins rejecting Pod
admission. Publishing the conditions before terminating Pods is how the design
honors it, and the same ordering holds per tier: the critical-phase condition is
published before the kubelet begins terminating, and rejecting, critical Pods.

**Pre-existing `DrainInProgress`.** If `DrainInProgress` is already `True` when
the shutdown signal arrives — set by `kubectl drain`, a maintenance operator, or
an administrator — the kubelet writes only `GracefulNodeShutdownInProgress` and
leaves the existing entry untouched, so its `reason`, `message`, and
`lastTransitionTime` continue to record who initiated the drain and when. The
reader's AND holds either way. The kubelet tracks in memory which conditions it
wrote during the current shutdown and, on cancel, resets only those; it does
not inspect `reason` to decide. This is the one ownership rule alpha carries;
the general mechanism is [#6430].

Two more transitions complete the happy path:

- **Shutdown cancelled.** logind emits `PrepareForShutdown(false)`; the shutdown
  manager already handles this by re-acquiring the inhibit lock and resuming
  admission. The kubelet sets the conditions it wrote for this shutdown to
  `False` on this path.
- **Kubelet startup.** The kubelet unconditionally sets any lingering `True`
  value on any of the three conditions to `False` before reporting `Ready`.
  Because shutdown state is in-memory only, a restarted kubelet cannot know it
  was mid-shutdown; an unconditional reset at startup makes that state loss a
  non-issue and covers reboot-after-shutdown, crash recovery, and disablement
  rollback with a single rule. It also resets a `DrainInProgress` another
  writer set before the reboot; see [Risks and
  Mitigations](#risks-and-mitigations). *Resolved in WG (2026-08-17): accepted
  as the alpha mechanism.*

**Windows.** [KEP-4802] (`WindowsGracefulNodeShutdown`, beta since v1.34) gives
Windows nodes the same shutdown manager shape: `ProcessShutdownEvent` in
`nodeshutdown_manager_windows.go` records the shutdown, fires the status sync,
and calls the shared `podManager.killPods`, and `Admit` rejects new Pods for the
duration — so the DaemonSet churn this KEP fixes occurs on Windows today. The
condition publish is therefore implemented as a shared helper in the
`nodeshutdown` package, invoked by both platform managers immediately before
`killPods`, which gives Windows the same ordering guarantee with no
platform-specific code. The tier boundary and the critical-phase publish live
in the shared `podManager` tier walk, so tier-aware admission is likewise
platform-neutral. Two differences are absorbed by the existing design:
[KEP-4802] lists shutdown cancellation as a Non-Goal and the Windows manager has
no cancel path, so the "shutdown cancelled" transition above does not exist on
Windows and kubelet-startup clear is the only recovery rule there; and the
shutdown window is the SCM preshutdown timeout, which the Windows manager
already extends to the configured grace period. GNS on Windows requires the
kubelet to run as a Windows service; where it does not, no conditions are
written and the reader behaves as today.

**New edge case on an existing bug.** The kubelet today does not remember it was
in graceful shutdown across a restart ([k/k#122674]). This KEP adds a new case
to that bug: while a shutdown is in progress, an administrator can remove a
condition the kubelet is responsible for, and the kubelet has no reconciliation
loop to restore it. The WG's agreed graduation direction is that the kubelet
reads level-triggered shutdown state from the OS and continuously reconciles the
conditions to match. Explorations include whether the systemd inhibitor /
`PrepareForShutdown` state is re-observable after a kubelet restart, and a
node-local marker file in a directory that is cleared on reboot. The latter has
precedent: when `GracefulNodeShutdownBasedOnPodPriority` is enabled, the
shutdown manager already persists a small JSON state file
(`graceful_node_shutdown_state`, under the kubelet root directory) recording the
shutdown start and end time for metrics, and reloads it at startup. Neither
exploration is an alpha requirement.

**Implementation shape.** A new nodestatus setter alongside the existing
condition setters in [`pkg/kubelet/nodestatus/`][pkg/kubelet/nodestatus/],
following the established pattern, driven by the shutdown manager's state, and
a shared publish helper in `nodeshutdown` called by both the Linux and Windows
managers. The tier boundary, the tier-aware `Admit()` check, and the
critical-phase publish live in the shared `podManager` tier walk. Unit-testable
against the existing fake `dbusInhibiter` on Linux and against the shared helper
directly on Windows.

### Node Lifecycle Controller (narrow writer)

The Node Lifecycle Controller
([`pkg/controller/nodelifecycle/`][pkg/controller/nodelifecycle/]) never asserts
`True`. Its single alpha responsibility is to clear the conditions when the
kubelet has been lost and the node is taken offline, so a node that dies
mid-shutdown does not carry stale state into removal or recovery flows.

**Trigger.** When the controller transitions the node's `Ready` condition to
`Unknown` (i.e., the kubelet has stopped heartbeating beyond
`nodeMonitorGracePeriod`), it sets all three of this KEP's conditions to
`status=Unknown` with reason `NodeStatusUnknown` — the same reason it writes on
`Ready` for the same event. It does not inspect the existing `reason` first:
under the WG's alpha last-writer-wins model an administrator's condition on a
node whose kubelet has vanished is superseded like any other. `Unknown` rather
than `False` because the control plane cannot distinguish a graceful shutdown
that ran to completion from a plug pulled mid-shutdown from a network partition;
all three look identical from `Ready=Unknown`, `False` would assert one of them,
and `Unknown` says exactly what the controller knows (which is also [KEP-5683]'s
definition of the value). The DaemonSet reader keys on `True` alone, so
suppression lifts either way. Rationale for the trigger: at that point the
writer is definitively gone; the purpose of the suppression — not fighting a
kubelet that is actively rejecting Pods — no longer applies; and reverting an
unreachable node to today's DaemonSet behavior is the fail-open default. Node
object deletion requires no handling (the conditions go away with the object).

**Known limitation of this trigger.** On architectures where the Node object
intentionally outlives the kubelet — teardown flows that stop the kubelet, then
perform post-kubelet cleanup, then delete the Node object — this trigger clears
the conditions while teardown is still underway, and the DaemonSet controller
resumes creating Pods on a node that is going away. The consequence is bounded —
with no kubelet there is no admission rejection, so Pods sit `Pending` rather
than churning until the Node object is deleted — but for the rest of the
teardown the feature has effectively switched itself off: DaemonSet Pods pile up
`Pending` on a dying node, DaemonSet status reports the node as unavailable for
no visible reason, and the "system pods stuck Pending during node removal"
symptom from [#137895] returns. For teardown flows with a long post-kubelet
phase, that can be most of the window. See [Design Decision 2](#design-decisions).

**Alternatives considered for the trigger:** clear only on node deletion (does
not cover a node that stays registered but dead, but *does* correctly serve the
kubelet-outlived-by-Node case — and, since deletion needs no code, would remove
the Node Lifecycle Controller writer from alpha entirely, which is attractive on
the WG's minimal-surface principle); clear on a bounded staleness timeout beyond
`nodeMonitorGracePeriod` (introduces a tunable PRR will ask about, but
accommodates post-kubelet cleanup windows); clear on `Ready` returning `True`
(covered already by the kubelet's own startup clear). Richer stale-writer
detection (e.g., a fresh `Ready` heartbeat alongside a stale condition
heartbeat, indicating a downgraded or gate-disabled kubelet) is a graduation
item.

The reason value `NodeStatusUnknown` is the one the Node Lifecycle Controller
already writes on `Ready` when heartbeats stop, and follows [KEP-5683]'s
convention of a stable, CamelCase cause category (see [Design Decision
3](#design-decisions)).

### DaemonSet controller (reader)

The reader is the DaemonSet controller
([`pkg/controller/daemon/`][pkg/controller/daemon/]). The change touches three
places, all keyed on the same predicate: the node carries
`GracefulNodeShutdownInProgress=True` and `DrainInProgress=True`, and either
the DaemonSet's template is non-critical or the node also carries
`GracefulNodeShutdownCriticalPhase=True`. The implementation factors that
predicate into one helper (working name `nodeShutdownSuppressed(node, ds)`) so
the call sites cannot drift:

1. `podsShouldBeOnNode` — the per-node decision made on every sync.
2. `rollingUpdate` — walks nodes independently of `podsShouldBeOnNode` and
   must apply the same predicate.
3. `shouldIgnoreNodeUpdate` — the informer-side filter that decides whether a
   Node update reaches the controller at all; without a change here the
   controller never observes the conditions changing.

**Critical templates.** A DaemonSet is critical when its
`spec.template.spec.priorityClassName` is `system-node-critical` or
`system-cluster-critical`. Both names are reserved, and their values sit at or
above `SystemCriticalPriority`, which is the boundary the kubelet's
`IsCriticalPod` uses for non-static Pods, so the controller needs no
PriorityClass lister to apply the rule. A critical DaemonSet is suppressed only
while `GracefulNodeShutdownCriticalPhase` is `True`; during the lower tiers the
kubelet admits its Pods and the controller keeps recreating them as it would
during a `kubectl drain`. The two-phase approximation this implies for
`shutdownGracePeriodByPodPriority` ladders is in [Priority
tiers](#priority-tiers).

**`podsShouldBeOnNode`.** While a node is suppressed for a DaemonSet, the
controller does the following:

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

**Why both conditions (the AND is deliberate).** `DrainInProgress` can be set by
other writers — `kubectl drain`, a maintenance operator, an administrator by
hand — and DaemonSet Pods are normally expected to survive a drain. Requiring
`GracefulNodeShutdownInProgress` as well scopes the suppression to the one case
this KEP is about. Read together, the two conditions contextualize what kind of
drain is occurring. *Resolved in WG (2026-08-24): accepted for alpha;
cross-writer coordination of `DrainInProgress` is [#6430].* An
integration test asserts that a node with only `GracefulNodeShutdownInProgress`
(or only `DrainInProgress`) is not suppressed.

**`rollingUpdate`.** The rolling-update path iterates `nodeToDaemonPods` on
its own and would otherwise act on a suppressed node under both strategies.
With `maxSurge > 0`, a node holding an old Pod and no new Pod is a surge
candidate, so the controller would create a new-hash Pod that the kubelet
immediately rejects — the same churn this KEP removes from the core loop. With
`maxSurge == 0`, an old Pod that is still available is a deletion candidate, so
the controller would race the kubelet for a Pod that is already being
terminated and spend `maxUnavailable` budget doing it. The implementation builds
the suppressed-node set for the DaemonSet once from `nodeList` at the top of
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
gate, the filter additionally returns `false` when the `Status` of any of the
three conditions differs between the old and new Node. Only `Status` is compared
— never `LastHeartbeatTime` or `LastTransitionTime` — so heartbeats enqueue
nothing and the controller sees at most three events per shutdown: enter,
critical phase, and exit. From there the existing `syncNodeUpdate` worker does
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
fail-open and bounded to one Pod per DaemonSet per condition transition: one for
a non-critical DaemonSet, at the initial publish; up to two for a critical
DaemonSet, whose suppression begins at the critical-phase publish. The kubelet
publishes each condition before it begins terminating the Pods it governs, and
termination takes at least the Pod's grace period, so the window is narrow in
practice. Alpha accepts it; closing it would require a live read of the Node
before every create and is not worth the API cost.

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
no-locking, last-writer-wins assumptions this KEP applies to the kubelet, and
with the same requirement that both conditions be `True`.

That property is deliberate, and it is what lets the reader stay correct as
sibling KEPs add writers (`kubectl drain` in [KEP-5683] alpha 2; maintenance
operators in [KEP-6250] / [KEP-6251]). It is not a sanction for arbitrary
writers to assert `GracefulNodeShutdownInProgress`. [KEP-5683] defines that
condition as reporting that Graceful Node Shutdown is in progress on the node; a
controller that tears down a node without a shutdown event and sets it anyway
would be asserting something untrue in order to obtain the DaemonSet behavior —
and the whole reason this KEP requires `GracefulNodeShutdownInProgress`
alongside `DrainInProgress` is that GNS *scopes* the suppression.

Controller-driven node teardown (Story 4) is a real environment with the same
churn, and it deserves the same coordination. The correct way to serve it is a
reader question — should the DaemonSet controller also suppress for
`DrainInProgress` together with `MaintenanceInProgress`, or for some other
combination that [KEP-6250] defines — not a writer question about who may set
the GNS condition. That question is deferred to beta, alongside the
coordination mechanism defined by [#6430], and this KEP records it as a
graduation item. An
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
- **Admission change placement.** The tier-aware `Admit()` behavior from
  [Priority tiers](#priority-tiers) ships under this gate for alpha, so a
  rollback restores today's reject-all admission together with the rest of the
  feature. The alternative, recorded for SIG Node, is to land it as a fix under
  `GracefulNodeShutdownBasedOnPodPriority` (beta, default on), which would
  change admission for every cluster in v1.38 and decouple it from the reader
  that depends on it. Recommendation: this gate; decision at SIG Node review.

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

- **`daemonset_controller_node_shutdown_suppression_total`** — counter recorded
  in the node-update worker (`syncNodeUpdate`) when an observed condition
  change flips a DaemonSet's suppression state on that node, incremented once
  per DaemonSet that has a Pod on the node and whose state flipped. Gives PRR an
  observable signal that the feature is active and makes the bounded-loop claim
  verifiable.
  - **Label `transition="enter"|"exit"`** — whether the DaemonSet entered or
    left suppression on the node. The feature is in use only between an `enter`
    and its matching `exit`, so the difference of the two over a window is the
    in-use gauge; no separate gauge is added.
  - **Label `critical="true"|"false"`**, true when the DaemonSet's template
    priority class is `system-node-critical` or `system-cluster-critical`. The
    two populations enter suppression at different moments — non-critical
    DaemonSets when the first two conditions are set, critical ones when
    `GracefulNodeShutdownCriticalPhase` is — so the counter is incremented for
    a DaemonSet when *its* suppression state flips, and the label says which
    rule fired. [#137895] suggests the affected population is predominantly
    critical, so this is also the label that shows whether the critical phase
    is where the churn was. The label is bounded (priority class *names* are
    user-defined and would be unbounded cardinality, so they are not used).

  Recording in the node worker rather than in `podsShouldBeOnNode` is
  deliberate. A counter bumped inside `syncDaemonSet` counts syncs that
  happened to run: once the rejected Pod on a suppressed node has been deleted,
  nothing triggers a further sync until an unrelated Pod or Node event arrives,
  so the value would track cluster noise rather than the feature. The node
  worker runs exactly once per observed condition flip (the
  `shouldIgnoreNodeUpdate` change above guarantees the event is delivered), so
  each transition is counted once per affected DaemonSet regardless of sync
  scheduling. `enter` rising with no matching `exit`, or `enter` events while
  no node is shutting down, are both directly meaningful.

Writer-side metrics belong to the condition-writing milestone and are not
proposed here.

### Priority tiers

*Added at API review (2026-09-25). Supersedes the "exploration" section carried
in earlier drafts; the WG lead concurred with the direction the same day.*

Graceful Node Shutdown terminates Pods in priority tiers, but until this KEP
the kubelet told no one which tier it was in and rejected every new Pod
regardless of tier. The API reviewer's objection to a phase-agnostic reader was
that a critical DaemonSet Pod should remain creatable while the kubelet is still
terminating lower tiers, and that a design which cannot express the tier would
codify the kubelet's reject-all admission rather than fix it. This section
records what the kubelet does today and the two changes that make the reader
consistent with it. Source references are pinned to
[kubernetes/kubernetes@e79a603][k/k-e79a603].

#### How the kubelet terminates Pods today

- Every configuration becomes one sorted list of priority tiers.
  `shutdownGracePeriodByPodPriority` is used as written; the legacy
  `shutdownGracePeriod` / `shutdownGracePeriodCriticalPods` pair is converted
  by `migrateConfig` into two tiers split at `SystemCriticalPriority`
  (2000000000), the same boundary `IsCriticalPod` uses
  ([`migrateConfig`][k/k-migrateconfig]).
- `killPods` buckets running Pods into those tiers and walks them from lowest
  to highest, waiting at most each tier's `shutdownGracePeriodSeconds` before
  moving on; empty tiers are skipped without waiting ([`killPods`][k/k-killpods]).
  A tier can end early, never late.
- The current tier exists only as the loop variable in `killPods` and a V(1)
  log line. It is not in the kubelet's state file, its metrics, or the Node.
- `Admit()` checks only whether shutdown has begun. Every Pod is rejected with
  reason `NodeShutdown` from the first tier to the last, on Linux and Windows
  alike ([`Admit`][k/k-admit]).

The last point is why a phase-agnostic reader did not regress behavior — a
critical DaemonSet Pod created during a lower tier was already rejected and
looped — and also why fixing admission on its own is not enough: once the
kubelet admits critical Pods during lower tiers, a reader that suppresses every
template would withhold Pods the kubelet would take.

#### Tier-aware admission

Under the `DaemonSetGracefulNodeShutdown` gate, the shutdown manager records a
boundary: the `Priority` of the next tier `killPods` has not yet reached.
`Admit()` rejects a Pod only when its `spec.priority` is below that boundary,
which is exactly the set of Pods that fall in the tier being terminated or one
already terminated; once the last tier begins there is no next tier and every
Pod is rejected, as today. Pods at or above the boundary are admitted and run
until their own tier is reached, exactly as Pods admitted before the shutdown
signal do. The boundary is updated in memory at each tier transition and read
by admission under the same lock as the shutdown flag, so the two never
disagree. The tier walk lives in the shared `podManager`, so Windows gets the
same behavior with no platform-specific code.

This is a change to Graceful Node Shutdown's admission behavior, not only to
this KEP's writer, and it is gated for that reason; see [Feature
gating](#feature-gating) for the placement recommendation and the alternative
recorded for SIG Node.

**Implementation order.** The kubelet change — tier-aware admission together
with all three condition writes — lands first, and the DaemonSet reader lands
on top of it. The reader's rule assumes tier-aware admission: a controller
that stops suppressing critical templates during lower tiers is only correct
if the kubelet admits them there. Reviewing the kubelet PR first also lets SIG
Node evaluate the admission change on its own terms before anything depends on
it.

#### The critical-phase condition

The critical tier is the one `groupByPriority` places critical Pods in: the
highest tier whose `Priority` is at or below `SystemCriticalPriority`. Under the
legacy configuration that is the second tier. Under a
`shutdownGracePeriodByPodPriority` ladder whose top tier sits below
`SystemCriticalPriority`, it is the top tier, because that is where critical
Pods are bucketed and terminated; defining it this way guarantees the condition
is published before the first critical Pod is terminated under any ladder. When
`killPods` reaches that tier, and before terminating any Pod in it, the kubelet
publishes `GracefulNodeShutdownCriticalPhase=True` (reason `NodeShutdown`; see
[The conditions](#the-conditions) for the entry as written). It is a new
`NodeConditionType` constant alongside the [KEP-5683] set; the name follows
that set's style and is open to API review. It is cleared with the other two on
cancel and at startup, and set to `Unknown` by the Node Lifecycle Controller
with them. It has no meaning on its own: the reader consults it only when
`GracefulNodeShutdownInProgress` and `DrainInProgress` are both `True`. Under
the legacy two-tier configuration it flips exactly when the kubelet moves from
the regular-Pod phase to the critical-Pod phase.

The reader rule becomes: with the first two conditions `True`, suppress
templates whose `priorityClassName` is neither `system-node-critical` nor
`system-cluster-critical`; once the third is also `True`, suppress every
template. Both names are reserved and their values sit at or above
`SystemCriticalPriority`, so the controller needs no PriorityClass lister to
apply the rule.

#### Alpha approximation and graduation path

The condition expresses two phases. `shutdownGracePeriodByPodPriority` can
define more: a ladder with a tier at, say, priority 1000 between the default
tier and the critical one. While the kubelet is terminating the default tier it
admits a priority-1000 Pod, but the reader, which sees only "not critical",
suppresses it. That is over-suppression — no churn, no rejected Pod, and the
template's Pod returns when the conditions clear — so alpha accepts it and
records it in [Risks and Mitigations](#risks-and-mitigations).

The precise form is for the kubelet to publish the boundary it is terminating
as an integer and for the reader to compare the template's resolved priority
against it. That needs a Node status field (a condition cannot carry an
integer) and a PriorityClass lister in the DaemonSet controller, and it is a
[beta criterion](#beta). When it lands, the reader rule narrows to "suppress
when the resolved priority is below the boundary"; an absent boundary from an
older kubelet falls back to the two-phase rule above, so the change is
additive.

### Design Decisions

*Questions raised during drafting and how each was closed. Nothing here is
open; the list exists so reviewers can see where each answer came from.*

1. **Feature gate name.** `DaemonSetGracefulNodeShutdown`; see [Feature
   gating](#feature-gating). Decided at KEP review.
2. **Node Lifecycle Controller trigger.** Alpha acts on the `Ready`→`Unknown`
   transition. Deletion-only was rejected because it leaves stale `True`
   conditions on a node that stays registered but dead; a bounded staleness
   timeout was rejected for alpha because it adds a tunable with no data yet to
   set it. The known limitation for teardown flows where the Node object
   outlives the kubelet is documented in [Risks and
   Mitigations](#risks-and-mitigations), and revisiting the trigger is a
   [beta criterion](#beta). Decided at KEP review.
3. **Node Lifecycle Controller write.** `status=Unknown`, reason
   `NodeStatusUnknown`, on all three conditions. `Unknown` is the only honest
   value: once heartbeats stop, the control plane cannot tell a shutdown that
   completed from a plug pulled mid-shutdown from a network partition, and
   `False` would assert one of those. It matches [KEP-5683]'s definition
   (Kubernetes cannot determine whether the state is active). The reason is
   the one the controller already writes on `Ready` for the same event, so
   all three conditions on an unreachable node carry the same cause. The
   controller does not inspect the existing `reason` before writing
   (last-writer-wins, per the WG's alpha model). Confirmed with the WG lead,
   2026-09-22.
4. **Single feature gate for writer and reader.** *Resolved in WG
   (2026-08-24)*; see [Feature gating](#feature-gating). The gate depends on
   `NodeLifecycleConditions` via the feature-gate dependency map. Decided at
   SIG Apps review.
5. **Metric labels.** `transition` and `critical` on
   `daemonset_controller_node_shutdown_suppression_total`, both bounded
   two-value labels. Cardinality is reviewed by SIG Instrumentation at the
   implementation PR as usual. Decided at KEP review.
6. **Priority tiers in alpha.** The phase-agnostic reader was replaced by
   tier-aware kubelet admission plus the `GracefulNodeShutdownCriticalPhase`
   condition, with per-tier precision as a [beta criterion](#beta). Raised by
   the API reviewer (a reader that cannot express the tier would codify the
   kubelet's reject-all admission); direction confirmed by the WG lead. See
   [Priority tiers](#priority-tiers). Decided at API review, 2026-09-25.
7. **Reader keys on `type` and `status` only; partial-rollout safety is
   operational.** Keying on `reason` was considered as a way to distinguish
   kubelet-written state from administrator-written or stale state and
   rejected: API conventions reserve `reason` for explanation, not control
   flow (see [Alternatives](#alternatives)). Instead, [Upgrade / Downgrade
   Strategy](#upgrade--downgrade-strategy) makes kubelet-first enablement,
   reader-first disablement, and a preflight listing of nodes carrying the
   conditions hard requirements, and the reader-only enablement test exercises
   the writer-agnostic behavior directly. Decided at PRR review.
8. **Condition name.** `GracefulNodeShutdownCriticalPhase`, matching the
   [KEP-5683] naming style. It is a new `NodeConditionType` constant and the
   name is open to API review on the implementation PR.
9. **Admission change placement.** Under this KEP's gate for alpha; the
   alternative of landing it under `GracefulNodeShutdownBasedOnPodPriority` is
   recorded in [Feature gating](#feature-gating) for SIG Node to decide.

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
- `k8s.io/kubernetes/pkg/controller/nodelifecycle`: `2026-09-21` - `71.6%`

Coverage in `pkg/kubelet/nodeshutdown` is low because the systemd/logind
integration in `nodeshutdown_manager_linux.go` is only partially exercisable
with the fake `dbusInhibiter`; the condition-writing path added by this KEP
will be covered by unit tests against that fake, the tier-aware admission and
critical-phase paths against the shared `podManager` directly, and all of it
end-to-end against real systemd in `test/e2e_node/`.

New and updated tests:

- [`pkg/kubelet/nodeshutdown/`][pkg/kubelet/nodeshutdown/]: condition write on
  shutdown signal, clear on cancel, clear on startup; against the existing fake
  `dbusInhibiter`. Tier-aware admission: a Pod in the current or an
  already-terminated tier is rejected, a Pod at or above the boundary is
  admitted, under both the legacy config and a multi-tier
  `shutdownGracePeriodByPodPriority`; gate off restores reject-all.
  `GracefulNodeShutdownCriticalPhase` is published when the critical tier is
  reached — after the lower tiers finish and before the first critical Pod is
  terminated — and cleared with the other two.
- [`pkg/kubelet/nodestatus/`][pkg/kubelet/nodestatus/]: new setter, gate on/off.
- [`pkg/controller/daemon/`][pkg/controller/daemon/]: `podsShouldBeOnNode` with
  both conditions, one condition, neither; a critical template with and without
  `GracefulNodeShutdownCriticalPhase`; a non-critical template with the third
  condition alone (no effect); failed-Pod deletion without replacement; gate
  on/off. `rollingUpdate`: a suppressed node is neither a
  surge-create candidate (`maxSurge > 0`) nor a delete candidate
  (`maxSurge == 0`), gate on/off. `shouldIgnoreNodeUpdate`: a condition
  `Status` flip passes the filter, a heartbeat-only update does not, gate off
  ignores conditions. `syncNodeUpdate`: metric attribution with `transition`
  and `critical` labels.
- [`pkg/controller/nodelifecycle/`][pkg/controller/nodelifecycle/]: clearing on
  the chosen trigger.

#### Integration tests

[`test/integration/daemonset/`][test/integration/daemonset/] — the clearest
statement of what this KEP does:

- Node with both conditions `True` → non-critical DaemonSet Pod deleted and not
  recreated.
- Node with `GracefulNodeShutdownInProgress=True` only → Pod is recreated
  (proves the AND is deliberate).
- Node with `DrainInProgress=True` only → Pod is recreated.
- Node with both conditions `True` and a `system-node-critical` DaemonSet →
  Pod is recreated until `GracefulNodeShutdownCriticalPhase=True` is added,
  then deleted and not recreated.
- Reader enabled against a node already carrying both conditions `True` →
  suppressed until cleared (the writer-agnostic reader; exercised deliberately
  so the rollout-ordering requirement is backed by a test).
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
  assert the conditions are published before Pod termination begins and cleared
  on kubelet restart; that a critical Pod is admitted while the default tier is
  being terminated and rejected once the critical tier begins; and that
  `GracefulNodeShutdownCriticalPhase` appears at that transition.

### Graduation Criteria

#### Alpha (v1.38)

- Feature gate `DaemonSetGracefulNodeShutdown`, default off, declaring its
  dependency on `NodeLifecycleConditions`.
- Kubelet writer (all three conditions), tier-aware admission, DaemonSet
  reader, and Node Lifecycle Controller clearing implemented behind the gate.
- `GracefulNodeShutdownCriticalPhase` added to `k8s.io/api/core/v1`.
- Unit and integration coverage with the gate on and off, including the
  critical-template cases.
- Suppression counter metric present, with the `critical` label.

#### Beta

*Direction only; criteria to be firmed up with the WG-level plan for how the SLM
writer and reader KEPs graduate together.*

- **Multi-writer coordination.** Adoption of the mechanism defined by [#6430]
  (SLM: Coordinate Node lifecycle condition writers), including cross-writer
  handling of `DrainInProgress` for this KEP's AND and a kubelet startup reset
  that distinguishes writers.
- **Amnesia-bug edge case.** The kubelet reads level-triggered shutdown state
  from the OS and continuously reconciles the conditions.
- **Stale-writer detection.** Node Lifecycle Controller backstop that detects a
  vanished or downgraded writer and clears with a distinct reason.
- **NLC clearing trigger revisited** for architectures where the Node object
  outlives the kubelet (see [Risks and Mitigations](#risks-and-mitigations) and
  [Design Decision 2](#design-decisions)).
- **Per-tier precision.** The kubelet publishes the priority boundary it is
  terminating, and the reader compares the template's resolved priority
  against it, replacing the two-phase approximation of [Priority
  tiers](#priority-tiers); informed by the `critical` metric label collected
  during alpha and aligned with the separate drain-ordering enhancement.
- **Controller-driven teardown.** A decision on whether, and via which condition
  combination, teardown orchestrated by a cluster-side controller triggers
  DaemonSet suppression (see [Other writers](#other-writers)), taken together
  with [KEP-6250] / [KEP-6251].
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

- **Upgrade order (required).** Enable the kubelet gate on all nodes first and
  the kube-controller-manager gate last. The reader is writer-agnostic, so
  enabling it first would act on any pre-existing conditions; kubelet-first
  guarantees that the only conditions the reader sees on enablement are ones a
  kubelet wrote during an actual shutdown. No migration of existing objects; a
  node that is not shutting down carries no kubelet-written condition.
- **Preflight (required).** Before enabling the reader, list nodes already
  carrying the conditions:
  `kubectl get nodes -o jsonpath='{range .items[*]}{.metadata.name}{" "}{.status.conditions[?(@.type=="GracefulNodeShutdownInProgress")].status}{"/"}{.status.conditions[?(@.type=="DrainInProgress")].status}{"\n"}{end}' | grep "True/True"`.
  A listed node that is `Ready` and not shutting down carries stale state;
  clear it before enabling the reader. A listed node an administrator marked
  deliberately will be suppressed once the reader is on, which is the intended
  behavior.
- **Downgrade / disable order (required).** Disable the kube-controller-manager
  gate first, then the kubelet gate or version. Reader-first guarantees that a
  condition a downgraded kubelet can no longer clear has no effect. Conditions
  left `True` on a node mid-shutdown at the moment of disablement are cleared
  by the kubelet's next startup (if it still has the gate) or set `Unknown` by
  the Node Lifecycle Controller; until then they are inert. Manual removal by
  an administrator is always possible.

### Version Skew Strategy

Fail-open semantics make every skew combination safe:

| Kubelet | kube-controller-manager | Result |
|---|---|---|
| New (writes conditions) | Old (ignores them) | Conditions present, unconsumed. Today's churn persists on shutting-down nodes, except that tier-aware admission now admits critical DaemonSet Pods the old controller creates during lower tiers, which is strictly less churn; no new failure mode. Kubelet startup clears the conditions. |
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
recreating non-critical DaemonSet Pods on those nodes for the duration of the
shutdown, and all DaemonSet Pods once the critical tier begins. The kubelet
admits Pods above the tier it is terminating instead of rejecting every Pod;
see [Priority tiers](#priority-tiers). Nodes not in shutdown are unaffected.

###### Can the feature be disabled once it has been enabled (i.e. can we roll back the enablement)?

Yes. With the gate off, no component writes or honors the conditions. Any
condition left `True` at the moment of disablement is cleared by the kubelet at
its next startup or set `Unknown` by the Node Lifecycle Controller when the
node is taken offline, and is inert until then. Disable the
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
reader-only enablement test starts with both conditions already `True` on a
`Ready` node (as an administrator would write them) and asserts suppression,
documenting that the reader is writer-agnostic and that the preflight check is
what protects against stale state.

### Rollout, Upgrade and Rollback Planning

###### How can a rollout or rollback fail? Can it impact already running workloads?

Rollout enables a writer and a reader that are each inert without the other's
conditions, so a partial rollout does not fail in a way that affects running
workloads: the worst case on any component is today's behavior. The same holds
for Pods not yet running. The kubelet's rejection of new Pod admission during
a shutdown is existing Graceful Node Shutdown behavior that this KEP narrows to
the tiers being terminated; with only the kubelet side enabled, DaemonSet Pods
targeted at a shutting-down node are rejected as today except for those above
the tier being terminated, which are admitted, and the node's next kubelet
startup clears the conditions before the node reports `Ready`. A partially
enabled cluster therefore cannot block an upgrade.

The reader is writer-agnostic — it keys on `type` and `status` only, per API
conventions — so a node that already carries both conditions `True` when the
reader is enabled is suppressed immediately for non-critical DaemonSets (and
for critical ones if it also carries `GracefulNodeShutdownCriticalPhase=True`),
whoever wrote them. For an
administrator-written pair that is the intended [KEP-5683] semantics. For stale
state it is not, which is why rollout ordering and the preflight check in
[Upgrade / Downgrade Strategy](#upgrade--downgrade-strategy) are requirements:
kubelets before the reader on the way up, reader off first on the way down, and
a listing of nodes carrying the conditions before the reader is enabled.

Control-plane components run as static Pods are DaemonSet-independent and
unaffected. Control-plane components run as DaemonSets are affected only on a
node carrying both conditions `True`, which after the preflight is a node in
shutdown or one an administrator has deliberately marked;
every path back to `Ready` (kubelet restart, Node Lifecycle Controller) clears
kubelet-written state. Because the reader runs in kube-controller-manager,
suppression can only occur while the control plane is serving, so the
break-glass levers in [Troubleshooting](#troubleshooting) are always reachable
when the feature is active; if the API server is unreachable the controller is
inert, exactly as today.

###### What specific metrics should inform a rollback?

`daemonset_controller_node_shutdown_suppression_total{transition="enter"}`
rising while no node in the cluster is shutting down would indicate stale or
incorrect conditions. `enter` counts with no matching `exit` over a long window
indicate conditions that are not being cleared. With the `critical` label,
`enter` events for critical DaemonSets on a node that never reached the
critical phase would indicate the reader is suppressing without
`GracefulNodeShutdownCriticalPhase`, which is a bug.

###### Were upgrade and rollback tested? Was the upgrade->downgrade->upgrade path tested?

Not yet; alpha. The integration tests cover gate on and gate off; an explicit
upgrade->downgrade->upgrade test is a beta requirement.

###### Is the rollout accompanied by any deprecations and/or removals of features, APIs, fields of API types, flags, etc.?

No.

### Monitoring Requirements

###### How can an operator determine if the feature is in use by workloads?

Presence of `GracefulNodeShutdownInProgress` / `DrainInProgress` (and, during
the critical tier, `GracefulNodeShutdownCriticalPhase`) on Node objects, and a
non-zero suppression counter.

###### How can someone using this feature know that it is working for their instance?

- [ ] Events
- [x] API .status
  - Condition name: `GracefulNodeShutdownInProgress` and `DrainInProgress`
    with reason `NodeShutdown` on a shutting-down Node, joined by
    `GracefulNodeShutdownCriticalPhase` once the critical tier begins.
- [x] Other (treat as last resort)
  - Details: `daemonset_controller_node_shutdown_suppression_total` increments
    with `transition="enter"` when the shutdown begins and `transition="exit"`
    when the node recovers, and the create/reject/delete churn in the audit log
    or `kubectl get events` stops.

###### What are the reasonable SLOs (Service Level Objectives) for the enhancement?

For alpha: no DaemonSet Pod admission is rejected with reason `NodeShutdown` on
a node after the condition governing that DaemonSet was published (the first two
for a non-critical DaemonSet, `GracefulNodeShutdownCriticalPhase` for a critical
one), beyond the one-sync informer propagation window at each transition.
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

One Node status update per shutdown event (setting the first two conditions),
one when termination reaches the critical tier, one on cancel, and one at
kubelet startup if clearing is needed. The startup clear can be coalesced into
the kubelet's initial status update. The Node Lifecycle
Controller issues one additional status update when it clears the conditions
on a node whose kubelet has been lost. Net effect is a **reduction** in API
calls, since the create/reject/delete churn is eliminated — per [#137895], that
churn can reach hundreds of cycles for a single Pod on a single node removal.

###### Will enabling / using this feature result in introducing new API types?

No.

###### Will enabling / using this feature result in any new calls to the cloud provider?

No.

###### Will enabling / using this feature result in increasing size or count of the existing API objects?

Up to three additional entries in `Node.status.conditions` on shutting-down
nodes only; the third appears only once termination reaches the critical tier.

###### Will enabling / using this feature result in increasing time taken by any operations covered by existing SLIs/SLOs?

No. The kubelet's condition write is bounded and does not delay shutdown; the
DaemonSet controller's per-node check is up to three condition lookups on an
object it already holds and a comparison of the template's priority class name
against two constants.

###### Will enabling / using this feature result in non-negligible increase of resource usage (CPU, RAM, disk, IO, ...) in any components?

No. Net resource usage on the API server and kube-controller-manager decreases
because the churn is eliminated.

###### Can enabling / using this feature result in resource exhaustion of some node resources (PIDs, sockets, inodes, etc.)?

No.

### Troubleshooting

###### How does this feature react if the API server and/or etcd is unavailable?

The kubelet's write is best-effort and bounded; it fails and the shutdown
proceeds. The DaemonSet controller sees absent conditions and behaves as today.

###### What are other known failure modes?

- Stale `True` conditions on a node whose kubelet died and has not returned and
  which the Node Lifecycle Controller has not yet processed. DaemonSet Pods are
  not recreated there until cleared. Admin remediation: delete the conditions.
- Stale `True` conditions on a node that returned to `Ready` under a kubelet
  that lacks the startup clear — the node was upgraded with the gate on, shut
  down, and rebooted into a downgraded kubelet. The Node Lifecycle Controller
  covers this whenever the reboot outlasts `nodeMonitorGracePeriod` (it sets
  the conditions `Unknown` before the old kubelet returns); a faster reboot
  leaves them `True` until an administrator clears them. Disabling the
  kube-controller-manager reader before downgrading kubelets (see [Upgrade /
  Downgrade Strategy](#upgrade--downgrade-strategy)) prevents any effect.
- Conditions cleared by the Node Lifecycle Controller while post-kubelet
  teardown is still in progress, on architectures where the Node object outlives
  the kubelet. Symptom: DaemonSet Pods created and left `Pending` on a node
  being torn down, until the Node object is deleted. See [Risks and
  Mitigations](#risks-and-mitigations).
- Conditions set by an administrator or another writer and never cleared.
  Suppression on that node is the intended reading of the [KEP-5683]
  admin-managed model — the writer asserted that the node is shutting down —
  and lasts until the writer clears them, the node's kubelet restarts, or the
  Node Lifecycle Controller takes the node offline. The preflight check in
  [Upgrade / Downgrade Strategy](#upgrade--downgrade-strategy) lists such nodes
  before the reader is enabled.
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
4. For a critical DaemonSet whose Pods are rejected with `NodeShutdown` late in
   the shutdown, confirm `GracefulNodeShutdownCriticalPhase` appeared when the
   critical tier began; if it did not, the kubelet published the first two
   conditions but not the third — check the kubelet's shutdown-manager logs
   for the tier transition.
5. If conditions are stale `True` on a node that is not shutting down, delete
   them, and check whether the kubelet restarted (it should have cleared them
   at startup) or whether the Node Lifecycle Controller has processed the node.

**Break-glass.** Two properties bound the worst case. The reader *is* the
DaemonSet controller, so if the API server is unreachable the controller neither
suppresses nor creates — exactly as today, with or without this feature — and
recovery is the cluster's existing bootstrap path; static Pods are never gated
by the conditions. And the kubelet's rejection of new Pods applies only to a
node actually shutting down ([KEP-2000] behavior, narrowed by tier here but
still confined to a real shutdown); on a healthy node carrying stuck conditions
the kubelet admits normally and the DaemonSet controller is the only thing
holding Pods back. Suppression affects creation only: a crash-looping Pod is
restarted by the kubelet and is untouched. To release a stuck node, in order of
least knowledge required:

- **Restart the kubelet on the node.** Its startup clear sets the conditions
  to `False` and the DaemonSet controller recreates on the next sync.
  Node-local; no API knowledge needed.
- **Clear the conditions on the node's status subresource** (NodeRestriction
  permits this for administrators):
  `kubectl patch node <n> --subresource=status --type=strategic -p '{"status":{"conditions":[{"type":"GracefulNodeShutdownInProgress","status":"False","reason":"AdminRequested"},{"type":"DrainInProgress","status":"False","reason":"AdminRequested"},{"type":"GracefulNodeShutdownCriticalPhase","status":"False","reason":"AdminRequested"}]}}'`.
  The reader keys on `True`, so `False` releases the node on the next sync.
- **Cluster-wide:** disable `DaemonSetGracefulNodeShutdown` on
  kube-controller-manager. Suppression stops on the next sync of every
  DaemonSet; no state needs cleaning up and kubelets need no change.

Automatic detection of the stuck state — a fresh `Ready` heartbeat alongside a
stale condition heartbeat — is the stale-writer backstop listed under
[Beta](#beta).

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
  (kubelet sets `False`, NLC sets `Unknown`, entries never removed); a
  pre-existing `DrainInProgress=True` is left untouched by the kubelet; gate
  depends on `NodeLifecycleConditions`; two-condition rationale moved up front.
- **2026-09-25:** API review (@deads2k), direction confirmed by the WG lead:
  tier-aware kubelet admission and the `GracefulNodeShutdownCriticalPhase`
  condition replace the phase-agnostic alpha; the exploration section becomes
  [Priority tiers](#priority-tiers).

## Drawbacks

- Adds an in-tree writer to conditions whose API doc comment currently says the
  admin owns them; requires an api-approver and a semantics change to that
  comment.
- Alpha accepts residual stale-condition and multi-writer risk under the
  admin-in-control assumption; these are real operational sharp edges until the
  graduation items land.
- The critical-phase condition expresses two phases only. Under
  `shutdownGracePeriodByPodPriority` ladders with intermediate tiers, templates
  in a tier the kubelet has not reached are over-suppressed until per-tier
  precision lands at beta (see [Priority tiers](#priority-tiers)).
- Tier-aware admission is a change to Graceful Node Shutdown's own behavior
  carried under this KEP's gate; it widens the KEP's blast radius from the
  DaemonSet controller to the kubelet's admission path.
- A third condition type is added to the [KEP-5683] vocabulary for a single
  reader.
- The alpha NLC clearing trigger under-serves architectures where the Node
  object outlives the kubelet.

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
  input to the per-tier precision work at beta.

Other alternatives considered:

- **Rely on `Ready=False`.** Indistinguishable from runtime failure; that
  ambiguity is the bug.
- **Rely on `failedPodsBackoff`.** Already exists; bounds the flood but does not
  stop the churn, fix attribution, or help rollout verifiers.
- **Suppress on `GracefulNodeShutdownInProgress` alone.** Would work for the
  DaemonSet case but loses the "what kind of drain" context that lets other
  readers reason about `DrainInProgress` writers uniformly; the WG preferred the
  AND.
- **Node Lifecycle Controller as the asserting writer.** The controller cannot
  observe shutdown; only the kubelet can. Publishing must originate at the
  kubelet to honor the admission-rejection invariant.
- **Exempt `system-node-critical` / `system-cluster-critical` DaemonSet Pods
  from suppression without a phase signal.** Needs no new condition, but the
  kubelet terminates *every* critical Pod, by design, during the critical
  phase; with a bare exemption each one is recreated, rejected, and looped for
  the entire `shutdownGracePeriodCriticalPods` window — reintroducing the churn
  for precisely the population [#137895] reports. A reader without a phase
  signal cannot distinguish "lower tier, critical Pod died, restore it" from
  "critical tier, kubelet is terminating it, leave it". The exemption is
  adopted only in combination with `GracefulNodeShutdownCriticalPhase`, which
  supplies exactly that signal; see [Priority tiers](#priority-tiers).
- **Publish the tier boundary as a Node status field in alpha.** Precise for
  arbitrary `shutdownGracePeriodByPodPriority` ladders, where the critical-phase
  condition over-suppresses intermediate tiers. Deferred to beta: a new status
  field needs API shape review the alpha schedule cannot absorb, the reader
  would need a PriorityClass lister to resolve template priority, and the
  two-phase condition is a strict subset of the eventual rule, so adding the
  field later is additive.
- **Keep reject-all admission and suppress every template.** Behavior-neutral
  today, since no critical Pod is admitted during any tier, but it codifies the
  kubelet's priority-blind admission into a control-plane contract. Rejected at
  API review, 2026-09-25.
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
[pkg/controller/nodelifecycle/]: https://github.com/kubernetes/kubernetes/tree/master/pkg/controller/nodelifecycle
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
