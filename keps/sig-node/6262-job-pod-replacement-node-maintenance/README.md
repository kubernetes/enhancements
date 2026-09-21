# KEP-6262: SLM: Job Pod replacement during node maintenance

<!-- toc -->
- [Release Signoff Checklist](#release-signoff-checklist)
- [Summary](#summary)
- [Motivation](#motivation)
  - [Goals](#goals)
  - [Non-Goals](#non-goals)
- [Proposal](#proposal)
  - [User Stories (Optional)](#user-stories-optional)
    - [Story 1: Jobs Stuck When Nodes Become Unreachable](#story-1-jobs-stuck-when-nodes-become-unreachable)
  - [Risks and Mitigations](#risks-and-mitigations)
- [Design Details](#design-details)
  - [Overview](#overview)
  - [Node awareness](#node-awareness)
  - [Feature Gate](#feature-gate)
  - [Reconciling the eventual duplicate](#reconciling-the-eventual-duplicate)
    - [Indexed Jobs](#indexed-jobs)
    - [Non-indexed Jobs](#non-indexed-jobs)
  - [Accounting](#accounting)
  - [Metrics](#metrics)
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
- [Infrastructure Needed (Optional)](#infrastructure-needed-optional)
<!-- /toc -->

## Release Signoff Checklist

Items marked with (R) are required *prior to targeting to a milestone / release*.

- [ ] (R) Enhancement issue in release milestone, which links to KEP dir in [kubernetes/enhancements] (not the initial KEP PR)
- [ ] (R) KEP approvers have approved the KEP status as `implementable`
- [ ] (R) Design details are appropriately documented
- [ ] (R) Test plan is in place, giving consideration to SIG Architecture and SIG Testing input (including test refactors)
  - [ ] e2e Tests for all Beta API Operations (endpoints)
  - [ ] (R) Ensure GA e2e tests meet requirements for [Conformance Tests](https://github.com/kubernetes/community/blob/master/contributors/devel/sig-architecture/conformance-tests.md) 
  - [ ] (R) Minimum Two Week Window for GA e2e tests to prove flake free
- [ ] (R) Graduation criteria is in place
  - [ ] (R) [all GA Endpoints](https://github.com/kubernetes/community/pull/1806) must be hit by [Conformance Tests](https://github.com/kubernetes/community/blob/master/contributors/devel/sig-architecture/conformance-tests.md) within one minor version of promotion to GA
- [ ] (R) Production readiness review completed
- [ ] (R) Production readiness review approved
- [ ] "Implementation History" section is up-to-date for milestone
- [ ] User-facing documentation has been created in [kubernetes/website](https://github.com/kubernetes/website/), for publication to [kubernetes.io](https://kubernetes.io/)
- [ ] Supporting documentation—e.g., additional design documents, links to mailing list discussions/SIG meetings, relevant PRs/issues, release notes

## Summary

When a node becomes unreachable during maintenance, its Pods may never reach a terminal phase (`Failed` or `Succeeded`) that the Job controller can observe. A Job using `podReplacementPolicy: Failed` can then wait indefinitely instead of creating a replacement Pod, leaving the Job and higher-level queueing systems stuck.

The Job controller will use the `MaintenanceInProgress` condition ([KEP-5683](../5683-lifecycle-conditions/README.md), Story 2: Jobs Stuck When Nodes Become Unreachable) as node context when deciding whether a Pod can be replaced. When `MaintenanceInProgress=True` is present on a node, the Job controller trusts this as an authoritative signal from the administrator that the Pods on that node may need special accounting, and does not wait for those Pods to reach a terminal phase before moving on. In accordance with the WG Node Lifecycle consensus reached on 2025-09-14, the Job controller performs no independent detection of node unreachability for this decision; it acts purely on the presence of the condition. This KEP defines the replacement and accounting semantics so Jobs can make progress safely under this explicit admin signal, without unintentionally running duplicate Pods.

This is a narrowly scoped, opt-in change gated behind a feature flag. It does not introduce a new API, taint, or node-death-detection mechanism; it changes only how the Job controller counts and replaces Pods once that condition is set.

## Motivation

A running Pod's phase has exactly one writer: the kubelet, through [`TerminatePod`](https://github.com/kubernetes/kubernetes/blob/e81f39c0e03ce8ed8e2660c9147b391edd9e262b/pkg/kubelet/status/status_manager.go#L721). If a Pod is deleted on a node whose kubelet becomes unreachable, whether during a planned infrastructure upgrade or because an administrator has confirmed a hardware failure and initiated remediation, that write never happens.

The Job controller depends on Pod phase to make two decisions, and gets stuck on both when this happens:

- **No replacement Pod is created.** [`IsPodTerminating`](https://github.com/kubernetes/kubernetes/blob/e81f39c0e03ce8ed8e2660c9147b391edd9e262b/pkg/controller/controller_utils.go#L1091) stays `true` forever for the stuck Pod. Under `podReplacementPolicy: Failed`, the controller subtracts the stuck Pod from its target replacement count, so no new Pod is created to take its place.
- **The Job cannot make progress for that Pod slot.** The Job controller
  continues to treat the terminating, non-terminal Pod as occupying capacity
  for replacement purposes, so the affected work is not retried by the Job
  controller.

This is not a hypothetical failure mode. It is reported in [kubernetes/kubernetes#134038](https://github.com/kubernetes/kubernetes/issues/134038), with concrete consequences, particularly for large parallel and batch workloads (the primary users of `podReplacementPolicy: Failed`):

- Jobs using `podReplacementPolicy: Failed` stop making progress, tying up
  expensive compute resources (for example, GPU quota) indefinitely for work
  that will never finish on the original node.
- Higher-level queueing systems such as Kueue cannot make progress on the
  affected workload while the underlying Job is blocked from creating a
  replacement Pod.
- Ecosystem projects have each built their own workaround for this same gap: [AppWrapper](https://project-codeflare.github.io/appwrapper/arch-fault-tolerance/) force-deletes affected Pods itself after a grace period, [medik8s/self-node-remediation](https://github.com/medik8s/self-node-remediation/blob/main/internal/controller/selfnoderemediation_controller.go) applies the existing `out-of-service` taint through an operator, and Kueue is considering its own mechanism ([kubernetes-sigs/kueue#6757](https://github.com/kubernetes-sigs/kueue/issues/6757)). All of these exist because Kubernetes itself has no primitive for "this Pod is not coming back" that the Job controller can act on directly.

A mechanism already exists to force-fail such Pods: the `node.kubernetes.io/out-of-service` taint, combined with PodGC ([KEP-2268](../../sig-storage/2268-non-graceful-shutdown/README.md)). But that taint is meant for nodes presumed permanently dead, and applying it triggers aggressive cleanup, including forced volume detachment. Routine node maintenance does not warrant treating the node as dead. [`MaintenanceInProgress`](https://github.com/kubernetes/kubernetes/blob/e81f39c0e03ce8ed8e2660c9147b391edd9e262b/staging/src/k8s.io/api/core/v1/types.go#L7260-L7268) is the more appropriate signal for this case: it is a routine, informative condition, not a fencing action. Today, nothing lets the Job controller act on that condition directly.

This KEP closes that gap: the Job controller becomes a consumer of `MaintenanceInProgress` ([KEP-5683](../5683-lifecycle-conditions/README.md)), as described in Summary above.

### Goals

- **Unblock stuck Jobs with `podReplacementPolicy: Failed` only.** Allow Jobs using `podReplacementPolicy: Failed` to make progress when their Pods are stuck on a node the administrator has explicitly marked via `MaintenanceInProgress=True`.
- **Replace without waiting on terminal phase.** Allow the Job controller to create a replacement Pod for a Pod on such a node without waiting for that Pod to reach a terminal phase (`Failed` or `Succeeded`).
- **Allow replacement-driven progress.** Allow the Job to create replacement
  Pods for affected work instead of stalling indefinitely on a Pod stuck in a
  non-terminal phase. Final cleanup of the original Pod, and any Job
  completion behavior that depends on that cleanup, remains outside this
  KEP's scope.
- **Define safe replacement and accounting semantics.** Define replacement and accounting semantics precise enough to avoid unintentionally running duplicate work for the same logical Pod slot.
  - *Alpha caveat:* the Job controller trusts the `MaintenanceInProgress` signal authoritatively and does not independently verify node or kubelet health. The setter of the condition assumes responsibility for ensuring the node is actually isolated, to prevent split-brain execution.
- **Make the behavior opt-in.** Gate this behavior fully behind a feature gate, defaulting to today's behavior when disabled.

### Non-Goals

- **Automatic Pod deletion.** This KEP does not force-delete or force-fail Pods stuck on unreachable nodes. Deletion and cleanup of the original Pod are owned by existing mechanisms (for example, PodGC reacting to the `out-of-service` taint, or a cloud/cluster autoscaler's own node-termination logic, such as [Karpenter's termination controller](https://github.com/kubernetes-sigs/karpenter/tree/main/pkg/controllers/node/termination/terminator)).
- **Guaranteeing final Job completion while the original Pod remains
  non-terminal.** This KEP unblocks replacement creation and lets the
  replacement Pod make progress, but it does not force the original Pod to
  reach a terminal phase or be removed. If final Job completion is still
  blocked by the original non-terminal Pod, cleanup remains the responsibility
  of existing external mechanisms such as PodGC, node remediation, or
  autoscaler cleanup.
- **Node/VM lifecycle management.** This KEP does not drain, terminate, or reboot the underlying node or VM. It treats `MaintenanceInProgress` purely as a signal the Job controller reads.
- **Defining or automating the writer of `MaintenanceInProgress`.** Who may set or clear the condition, and how conflicting writers are reconciled, is owned by [KEP-5683](../5683-lifecycle-conditions/README.md).
- **Other workload APIs.** This KEP does not modify replacement or eviction semantics for other controllers, such as StatefulSets or DaemonSets. Extending this pattern to other controllers would require its own KEP.
- **`podReplacementPolicy: TerminatingOrFailed`.** This KEP is scoped to Jobs using `podReplacementPolicy: Failed`, the policy that waits for a Pod to reach a terminal phase (`Failed` or `Succeeded`) before creating a replacement. Under `TerminatingOrFailed`, the Job controller already creates a replacement as soon as a Pod has `metadata.deletionTimestamp` set, without waiting for the terminal phase, so it is not subject to the stall this KEP addresses.

## Proposal

The Job controller, when deciding whether to create a replacement Pod for a
Job using `podReplacementPolicy: Failed`, will check the `MaintenanceInProgress`
condition on the node hosting a stuck (terminating, non-terminal) Pod. If the
condition is `True`, the Job controller no longer waits for that Pod to reach
a terminal phase before creating its replacement, and adjusts its internal
accounting so the stuck Pod no longer blocks replacement. The mechanism,
including exactly which counters and code paths change for indexed and
non-indexed Jobs, is described in [Design Details](#design-details).

### User Stories (Optional)

#### Story 1: Jobs Stuck When Nodes Become Unreachable

Tracking Issues:
- [kubernetes/kubernetes#134038](https://github.com/kubernetes/kubernetes/issues/134038)

As a Job or queueing controller, `MaintenanceInProgress=True` gives Job and
queueing controllers a node-level signal that Pods on the node may need
special accounting when an admin or maintenance controller has identified
the node as being lifecycled ([KEP-5683](../5683-lifecycle-conditions/README.md),
Story 2).

A Job configured with `podReplacementPolicy: Failed` checks the node for
this additional context when accounting for its Pods. When the Job
controller sees `MaintenanceInProgress=True` on the node hosting one of its
Pods, it does not wait for that Pod to reach a terminal phase before
creating a replacement, unblocking Jobs and higher-level queueing systems
(such as Kueue) that would otherwise stall indefinitely.

### Risks and Mitigations

The primary risk stems from this KEP depending on the admin (or maintenance
controller) correctly managing the node's lifecycle and the
`MaintenanceInProgress` condition. If that responsibility isn't met, the
following can go wrong:

- **Replacing a healthy Pod.** If `MaintenanceInProgress=True` is set while
  the node and its kubelet are actually fine (e.g., a bad script, or a
  condition that's set but never cleared), the Job controller creates a
  replacement Pod while the original is still running, causing duplicate
  work. *Mitigation*: this is an accepted Alpha trade-off (see
  [Goals](#goals)); whoever sets the condition is responsible for the node
  actually being isolated. Writer semantics and validation belong to
  [KEP-5683](../5683-lifecycle-conditions/README.md), not this KEP.

## Design Details

### Overview

**Terminology**: a "stuck Pod," as used throughout this section, is a Pod that
(1) has `metadata.deletionTimestamp` set, and (2) has not reached a terminal
phase (`Failed` or `Succeeded`), i.e.,
[`IsPodTerminating(p)`](https://github.com/kubernetes/kubernetes/blob/e81f39c0e03ce8ed8e2660c9147b391edd9e262b/pkg/controller/controller_utils.go#L1091)
is `true`, as consumed by the Job controller via
[`CountTerminatingPods`](https://github.com/kubernetes/kubernetes/blob/e81f39c0e03ce8ed8e2660c9147b391edd9e262b/pkg/controller/job/job_controller.go#L1039).
This KEP does not assume or care why the Pod was deleted (user
action, a controller, node drain, etc.); it only requires that a deletion
already happened and the kubelet has not been able to follow up with a phase
transition, exactly the precondition described in
[kubernetes/kubernetes#134038](https://github.com/kubernetes/kubernetes/issues/134038)
("Pod stuck in terminating state (with `deletionTimestamp`, but with
`phase=Running`)").

Today, when a Job uses `podReplacementPolicy: Failed`, the Job controller withholds
replacement Pod creation for any Pod that is terminating but has not yet reached a
terminal phase (`Failed` or `Succeeded`). This is implemented via
[`IsPodTerminating`](https://github.com/kubernetes/kubernetes/blob/e81f39c0e03ce8ed8e2660c9147b391edd9e262b/pkg/controller/controller_utils.go#L1091),
which only checks `metadata.deletionTimestamp` and the Pod's phase, with no
awareness of node state. Two independent call sites consume this same primitive
and therefore need to be updated together:

- **Non-indexed Jobs**: `manageJob` currently computes `jobCtx.terminating` via
  [`CountTerminatingPods`](https://github.com/kubernetes/kubernetes/blob/e81f39c0e03ce8ed8e2660c9147b391edd9e262b/pkg/controller/controller_utils.go#L1021)
  and subtracts it from the desired replica count (`diff := wantActive - terminating - active`)
  when `onlyReplaceFailedPods(job)` is true, delaying replacement creation.
- **Indexed Jobs**: `firstPendingIndexes` computes a set of "covered" indexes via
  [`FilterTerminatingPods`](https://github.com/kubernetes/kubernetes/blob/e81f39c0e03ce8ed8e2660c9147b391edd9e262b/pkg/controller/controller_utils.go#L1011)
  and excludes those indexes from the list of indexes needing a new Pod.

Both call sites rely on the same underlying check, `IsPodTerminating`. This
KEP introduces a single new node-aware primitive that both replacement paths
use consistently:

```go
// isPodTerminatingOnMaintenanceNode reports whether a terminating Pod is
// covered by an administrator's MaintenanceInProgress=True assertion on its
// node, and can therefore be replaced without waiting for a terminal phase.
func isPodTerminatingOnMaintenanceNode(p *v1.Pod, nodeLister corelisters.NodeLister) bool {
    if !IsPodTerminating(p) {
        return false
    }
    node, err := nodeLister.Get(p.Spec.NodeName)
    if err != nil {
        return false
    }
    return nodeHasConditionTrue(node, v1.NodeMaintenanceInProgress)
}
```

Pods for which this returns `true` are excluded only from the replacement
blocking decisions: the non-indexed `terminating` value used in
`diff := wantActive - terminating - active`, and the indexed covered-index set
in `firstPendingIndexes`. They remain terminating Pods for normal status and
cleanup purposes. In particular, this KEP does not remove them from
`.status.terminating` and does not make final Job completion independent of
their eventual cleanup. The original Pod is not force-deleted, force-failed,
or phase-modified by the Job controller.

### Node awareness

The Job controller does not currently have a `NodeLister` and does not
watch nodes today. `kube-controller-manager` commonly already runs a shared
node informer, consumed by several existing controllers (for example,
DaemonSet, endpoint, node-lifecycle); this KEP has the Job controller consume
that shared informer rather than performing a live API call per Pod for
`p.Spec.NodeName -> MaintenanceInProgress` lookups. In minimal
`kube-controller-manager` configurations where no enabled controller has
started the shared node informer, enabling this feature may start that shared
informer. This will require updating the Job
controller's RBAC permissions to `get`, `list`, and `watch` nodes, since it
does not have this permission today. To minimize this footprint, when the
feature gate is disabled, the Job controller does not request the node
informer at all, preserving today's RBAC/watch footprint for the Job
controller specifically. The Job controller does not enqueue Jobs directly
from node events; it only reads node state during its existing Job/Pod
syncs, so changes to `MaintenanceInProgress` are observed on the next normal
Job sync or resync rather than immediately on the node update.

Operationally, maintenance controllers should set `MaintenanceInProgress=True`
before or around the Pod deletion or eviction that makes the Pod terminating.
The Pod update then enqueues the owning Job, allowing the Job controller to
observe the condition during that sync. If the condition is set after the Pod
deletion event has already been processed and no further Job or Pod event
occurs, replacement is delayed until a later Job sync or informer resync.

### Feature Gate

This entire behavior is guarded by the disabled-by-default
`JobPodReplacementOnNodeMaintenance` feature gate, enabled on
`kube-controller-manager`. When the gate is off, the Job controller behaves
exactly as it does today:
`IsPodTerminating`/`CountTerminatingPods`/`FilterTerminatingPods` are used
unmodified, no node lister/informer is created, and `MaintenanceInProgress`
is never read. Only when the gate is on does the Job controller consult
`MaintenanceInProgress` and apply the accounting changes described above.
This lets cluster operators adopt the behavior per-cluster without any
change to Job specs, and lets it be rolled back cleanly by disabling the
gate, per the standard feature-gate lifecycle (Alpha: off by default; Beta:
on by default; GA: locked to on, gate removed).

### Reconciling the eventual duplicate

#### Indexed Jobs

Once a replacement Pod is created for a stuck index while the original Pod is
still present (terminating, not yet terminal), the index transiently has two
Pod objects for the same completion index, but only the replacement is
active. The original stuck Pod has `deletionTimestamp` set, so it is already
excluded from `activePods`; the existing
[`appendDuplicatedIndexPodsForRemoval`](https://github.com/kubernetes/kubernetes/blob/e81f39c0e03ce8ed8e2660c9147b391edd9e262b/pkg/controller/job/indexed_job_utils.go#L293)
logic scans active Pods and does not need to choose between the stuck
original and the healthy replacement.

The indexed-specific change is instead in `firstPendingIndexes`: a stuck Pod
on a node with `MaintenanceInProgress=True` must not be added to the
covered-index set. That makes the completion index eligible for a replacement
Pod. The required invariant is that the same node-aware terminating
predicate is used for both non-indexed replacement counts and indexed
covered-index accounting, so the two Job modes make consistent replacement
decisions.

#### Non-indexed Jobs

For non-indexed Jobs, no equivalent duplicate-reconciliation logic is needed:
if the stuck original Pod eventually reaches a terminal phase (e.g., after
being cleaned up by PodGC or an autoscaler), it is handled exactly like any
other concurrently-running Pod in a non-indexed Job today. If it succeeds, it
increments `.status.succeeded`; if it fails, it
consumes a `backoffLimit` retry like any other failure. No new logic is
required for this case.

### Accounting

The table below summarizes how each relevant counter behaves for a Pod stuck
on a node with `MaintenanceInProgress=True`, compared to today's behavior.

| Counter | Today (`podReplacementPolicy: Failed`) | With this KEP |
|---|---|---|
| `.status.active` | The stuck Pod is already excluded from `.status.active` today, since [`FilterActivePods`](https://github.com/kubernetes/kubernetes/blob/e81f39c0e03ce8ed8e2660c9147b391edd9e262b/pkg/controller/job/job_controller.go#L1030)/[`IsPodActive`](https://github.com/kubernetes/kubernetes/blob/e81f39c0e03ce8ed8e2660c9147b391edd9e262b/pkg/controller/controller_utils.go#L1085) treats any Pod with `deletionTimestamp` set as inactive. It blocks replacement solely via `.status.terminating` in the formula `diff := wantActive - terminating - active`. | By excluding this specific Pod from the replacement-blocking `terminating` count (see [Overview](#overview)), `diff` becomes positive again, so a replacement Pod is created and counted in `.status.active`. |
| `.status.terminating` | Includes the stuck Pod (it has `deletionTimestamp` set, not yet terminal). | Unchanged: the stuck Pod still has `deletionTimestamp` set, so it is still reported in `.status.terminating`. This KEP only changes whether that count *blocks* replacement creation, not whether the Pod is reported as terminating. |
| `.status.failed` / `backoffLimit` | Not incremented while the Pod is merely terminating; only incremented once/if the Pod actually reaches the `Failed` phase. | Unchanged. The stuck Pod does not consume `backoffLimit` budget by virtue of this KEP; it is only counted as failed if and when it eventually reaches a terminal phase (e.g., after PodGC or an autoscaler removes it). |
| `.status.succeeded` | Unaffected unless the Pod reaches `Succeeded`. | Unchanged. |
| Completion index (indexed Jobs) | An indexed Job assigns each Pod a completion index (e.g., index 3 of 10). Today, as long as the stuck Pod for index 3 still exists (even though it's terminating and stuck), the controller thinks that index is "handled" and never creates a new Pod for it, so index 3 cannot make progress. | Once `MaintenanceInProgress=True` is observed on the stuck Pod's node, the controller creates a replacement Pod for index 3. If that replacement reaches `Succeeded`, index 3 can be counted as completed using the same indexed-Job rules as today. The original stuck Pod never itself counts as index 3's completion; it only stops blocking the replacement from being created. Final Job completion may still wait for external cleanup of non-terminal Pods, consistent with [Non-Goals](#non-goals). |

**Net effect**: this KEP does not change how or when a Pod is counted as
active, failed, or succeeded, and does not remove the stuck Pod from
`.status.terminating`. It only changes whether the stuck Pod's presence in
`.status.terminating` is allowed to *block creation of a replacement* in the
`diff := wantActive - terminating - active` formula. This keeps the change
narrowly scoped to the replacement-creation decision, and avoids altering
`backoffLimit` semantics or other existing accounting guarantees that SIG
Apps relies on. Because this KEP does not clean up the original Pod, it does
not guarantee that the Job reaches a terminal condition while that Pod remains
non-terminal; external cleanup remains responsible for that part of the
lifecycle.

### Metrics

Alpha adds a Job-controller reader-side metric:

- **`job_controller_pod_replacements_by_node_maintenance_total`** — counter
  of replacement Pods created because the original Pod was terminating,
  non-terminal, and on a node with `MaintenanceInProgress=True`. This gives
  PRR and operators an observable signal that the feature is active and helps
  distinguish this replacement path from ordinary Job Pod replacements.
  - **Proposed label: `completion_mode="Indexed"|"NonIndexed"`**, matching
    existing Job controller metric label conventions and keeping cardinality
    bounded.

Writer-side metrics for setting or clearing `MaintenanceInProgress` belong to
[KEP-5683](../5683-lifecycle-conditions/README.md), not this KEP.

### Test Plan

- [x] I/we understand the owners of the involved components may require updates
  to existing tests to make this code solid enough prior to committing the
  changes necessary to implement this enhancement.

##### Prerequisite testing updates

None.

##### Unit tests

- Unit tests for excluding a terminating Pod on a
  `MaintenanceInProgress=True` node from the replacement-blocking count
  (non-indexed Jobs) and covered-index set (indexed Jobs).
- Unit tests for feature-gate-disabled behavior being unchanged from today.
- Unit tests for node-lookup failures failing closed (treated as no
  `MaintenanceInProgress`).
- Unit tests verifying an indexed Job creates a replacement for an index
  whose original Pod is terminating on a `MaintenanceInProgress=True` node.

##### Integration tests

- Verify a Job with `podReplacementPolicy: Failed` creates a replacement Pod
  without waiting for the stuck Pod's terminal phase, once
  `MaintenanceInProgress=True` is set on its node.
- Verify `.status.active`, `.status.terminating`, `.status.failed`, and
  `.status.succeeded` transition as described in the Accounting table, for
  both indexed and non-indexed Jobs.
- Verify an indexed Job's stuck completion index becomes eligible for a new
  Pod, and the replacement can satisfy that index once it succeeds.
- Verify the stuck Pod later reaching a terminal phase is reconciled
  correctly, without double-counting against `backoffLimit` or
  `.status.succeeded`.

##### e2e tests

Deferred to Beta graduation, once cluster/node-conditions infrastructure for
simulating `MaintenanceInProgress` is available via
[KEP-5683](../5683-lifecycle-conditions/README.md).

### Graduation Criteria

#### Alpha
- Initial implementation of the node-aware replacement logic in the Job controller.
- Introduction of the `JobPodReplacementOnNodeMaintenance` feature gate, disabled by default.
- Unit and integration tests covering the replacement and accounting logic for both non-indexed and indexed Jobs, including the indexed covered-index behavior described in [Reconciling the eventual duplicate](#reconciling-the-eventual-duplicate).
- Metric `job_controller_pod_replacements_by_node_maintenance_total` present.
- SIG Apps sign-off on using the same node-aware terminating predicate for both non-indexed replacement counts and indexed covered-index accounting.
- **Signal scope:** For Alpha, the Job controller explicitly trusts the `MaintenanceInProgress=True` condition as an authoritative signal. It does not perform any secondary checks on the node's `Ready` state or kubelet health, leaving the responsibility of preventing split-brain execution to the administrator or component that sets the condition.

#### Beta
- Feature gate enabled by default.
- **Signal hardening (split-brain prevention):** Evaluate and implement secondary node checks (e.g., node `Ready=Unknown/False` or `node.kubernetes.io/unreachable` taint) to ensure the controller does not act if the node is still healthy.
- Comprehensive e2e tests implemented and passing (covering replacement creation, condition removal/flapping, and indexed Jobs).
- Metrics and dashboards/alerts validated for Job Pod replacements triggered by `MaintenanceInProgress`.
- User-facing documentation published on `kubernetes.io`.
- Community feedback gathered from batch/Kueue workloads confirming stability and no unexpected duplicate executions.

#### GA
- Feature gate locked to `true` (and prepared for removal).
- The feature has baked in Beta for at least two minor Kubernetes releases without major bugs, Job controller regressions, or reported duplicate-execution incidents.
- E2E tests have proven stable and flake-free over the Beta period.
- (If applicable) GA-level e2e tests meet the requirements for Kubernetes Conformance Tests.

### Upgrade / Downgrade Strategy

This KEP is additive and only affects `kube-controller-manager`. Jobs, Pods,
and nodes retain their current meaning; no new API fields are introduced.

On upgrade, clusters that enable the feature gate begin consulting the
`MaintenanceInProgress` condition when deciding whether to replace a Pod for
Jobs using `podReplacementPolicy: Failed`. Jobs already waiting on a stuck
Pod at the time of upgrade are picked up on the next sync: if
`MaintenanceInProgress=True` is already set on the stuck Pod's node, a
replacement Pod is created immediately.

On downgrade or feature disablement, the Job controller reverts to today's
behavior: it stops consulting `MaintenanceInProgress`, and Jobs with a stuck
Pod on a node under maintenance return to waiting for that Pod's terminal
phase, exactly as they do today without this feature.

### Version Skew Strategy

This feature is implemented entirely within `kube-controller-manager` and
does not depend on kubelet or `kube-apiserver` behavior beyond
`MaintenanceInProgress` already being a valid node condition (via
[KEP-5683](../5683-lifecycle-conditions/README.md)).

If `kube-controller-manager` supports this feature but the feature gate is
not enabled, behavior remains unchanged from today.

If `kube-controller-manager` does not yet support this feature (older
version), `MaintenanceInProgress` is simply not consumed for Job Pod
replacement decisions; Jobs behave as they do today.

## Production Readiness Review Questionnaire

### Feature Enablement and Rollback

###### How can this feature be enabled / disabled in a live cluster?

- [x] Feature gate
  - Feature gate name: `JobPodReplacementOnNodeMaintenance`
  - Components depending on the feature gate:
    - `kube-controller-manager`

###### Does enabling the feature change any default behavior?

Yes. For Jobs using `podReplacementPolicy: Failed`, the Job controller no
longer waits for a Pod to reach a terminal phase before creating a
replacement, if that Pod's node has `MaintenanceInProgress=True`. Jobs not
using `podReplacementPolicy: Failed`, and Jobs whose Pods are on nodes
without this condition, are unaffected.

###### Can the feature be disabled once it has been enabled (i.e. can we roll back the enablement)?

Yes.

###### What happens if we reenable the feature if it was previously rolled back?

No change in behavior beyond what is described above. The Job controller
resumes consulting `MaintenanceInProgress` for Pod replacement decisions;
any Pods that accumulated on nodes with the condition set while the feature
was disabled are handled the same as if the condition had just been set.

###### Are there any tests for feature enablement/disablement?

Unit tests to cover turning on/off the feature gate (see [Test Plan](#test-plan)).

### Rollout, Upgrade and Rollback Planning

###### How can a rollout or rollback fail? Can it impact already running workloads?

A rollout failure (for example, a bug in the node-aware replacement check)
could cause a replacement Pod to be created for a Pod that is not actually
stuck, or fail to create one when it should. This could result in duplicate
work (see [Risks and Mitigations](#risks-and-mitigations)) or in a Job
remaining stuck as it does today. It does not affect Jobs that are not
using `podReplacementPolicy: Failed`, or Pods on nodes without
`MaintenanceInProgress` set.

A rollback (disabling the feature gate) returns to today's behavior;
already-created replacement Pods are not affected or removed.

###### What specific metrics should inform a rollback?

An unexpected increase in
`job_controller_pod_replacements_by_node_maintenance_total`, an unexpected
increase in Pod creation rate from the Job controller, or an increase in
duplicate Pods or unexpected failed Pods should inform a rollback.

###### Were upgrade and rollback tested? Was the upgrade->downgrade->upgrade path tested?

TBD before beta.

###### Is the rollout accompanied by any deprecations and/or removals of features, APIs, fields of API types, flags, etc.?

No.

### Monitoring Requirements

###### How can an operator determine if the feature is in use by workloads?

Operators can inspect a Job's Pods and the `MaintenanceInProgress` condition
on their nodes; a replacement Pod created before the original reaches a
terminal phase, or a non-zero
`job_controller_pod_replacements_by_node_maintenance_total`, indicates the
feature is in use.

###### How can someone using this feature know that it is working for their instance?

- [x] Metrics
  - Metric name: `job_controller_pod_replacements_by_node_maintenance_total`

###### What are the reasonable SLOs (Service Level Objectives) for the enhancement?

Jobs using `podReplacementPolicy: Failed` should not be blocked from
progressing solely because a Pod is stuck on a node with
`MaintenanceInProgress=True`.

###### What are the SLIs (Service Level Indicators) an operator can use to determine the health of the service?

- [x] Metrics
  - Metric name: `job_controller_pod_replacements_by_node_maintenance_total`

###### Are there any missing metrics that would be useful to have to improve observability of this feature?

Additional writer-side metrics for `MaintenanceInProgress` would help
operators distinguish condition-writer behavior from Job controller behavior;
those belong to [KEP-5683](../5683-lifecycle-conditions/README.md).

### Dependencies

###### Does this feature depend on any specific services running in the cluster?

This feature depends on `MaintenanceInProgress` being set on nodes by an
admin or admin-authorized maintenance controller, as defined by
[KEP-5683](../5683-lifecycle-conditions/README.md). No other external
services are required.

### Scalability

###### Will enabling / using this feature result in any new API calls?

Yes, but modestly. The Job controller will consume the shared node informer
(see [Node awareness](#node-awareness)). In a typical
`kube-controller-manager` configuration, that informer is already running for
other controllers; in minimal configurations, enabling this feature may start
the shared node informer. The Job controller itself does not have `get`,
`list`, `watch` RBAC permissions on nodes today, so this is also a new
permission. No live node lookup is added per Pod. The feature can cause the
Job controller to create
replacement Pods for Jobs that would otherwise remain stuck, but it does not
introduce a new kind of Pod API operation beyond the create/delete calls the
Job controller already uses.

###### Will enabling / using this feature result in introducing new API types?

No.

###### Will enabling / using this feature result in any new calls to the cloud provider?

No.

###### Will enabling / using this feature result in increasing size or count of the existing API objects?

No.

###### Will enabling / using this feature result in increasing time taken by any operations covered by existing SLIs/SLOs?

No expected impact. `MaintenanceInProgress` lookups are served from the
Job controller's informer cache, not live API calls per Pod.

###### Will enabling / using this feature result in non-negligible increase of resource usage (CPU, RAM, disk, IO, ...) in any components?

No non-negligible increase is expected. In a typical `kube-controller-manager`
configuration, the shared node informer is already running for other
controllers (for example, DaemonSet, endpoint, node-lifecycle), so the Job
controller only becomes another consumer of the same cache. In minimal
configurations where no enabled controller has started that shared informer,
enabling this feature may add the shared node informer cache, proportional to
cluster node count.

###### Can enabling / using this feature result in resource exhaustion of some node resources (PIDs, sockets, inodes, etc.)?

No.

### Troubleshooting

###### How does this feature react if the API server and/or etcd is unavailable?

The Job controller cannot observe `MaintenanceInProgress` changes or create
replacement Pods until API server and etcd availability returns, consistent
with today's Job controller behavior.

###### What are other known failure modes?

- The Job controller's node informer cache is stale or has not yet
  observed a `MaintenanceInProgress` change:
  - Detection: the Pod remains stuck longer than expected after the
    condition was set.
  - Mitigation: wait for the controller cache and Job sync to catch up, or
    trigger a Job or Pod update that re-enqueues the affected Job.
- `MaintenanceInProgress` is set on a node whose Pod is not actually stuck
  (see [Risks and Mitigations](#risks-and-mitigations)):
  - Detection: an unexpected duplicate Pod running for the same Job/index.
  - Mitigation: this is an accepted Alpha-scope trade-off; the entity
    setting the condition is responsible for it being accurate.

###### What steps should be taken if SLOs are not being met to determine the problem?

Check whether `MaintenanceInProgress` is set correctly on the affected
node(s), whether the feature gate is enabled, and whether the Job
controller's node informer is healthy and up to date.

## Implementation History

- 2026-07-29: Provisional enhancement issue [#6262](https://github.com/kubernetes/enhancements/issues/6262) created
- 2026-08-24: WG Node Lifecycle discussed the stuck terminating Job Pod problem and using `MaintenanceInProgress` as additional node context for Job replacement decisions
- 2026-09-14: WG Node Lifecycle aligned on the Alpha direction: the Job controller trusts `MaintenanceInProgress=True` as the admin-provided tie-breaker and does not independently detect node unreachability
- 2026-09-20: Initial KEP PR opened

## Drawbacks

- **Trusts an unverifiable signal.** The Job controller performs no
  independent verification of node or kubelet health; an incorrectly or
  prematurely set `MaintenanceInProgress` condition can cause a replacement
  Pod to be created while the original is still running, causing duplicate
  work (see [Risks and Mitigations](#risks-and-mitigations)). Some
  reviewers in [kubernetes/kubernetes#134038](https://github.com/kubernetes/kubernetes/issues/134038)
  have raised similar concerns about any mechanism that stops waiting for
  a Pod's terminal phase without being certain the Pod has actually
  stopped running.
- **Does not solve the "ghost Pod" problem.** As raised in
  [kubernetes/kubernetes#134038](https://github.com/kubernetes/kubernetes/issues/134038),
  the underlying stuck Pod is not deleted or fenced by this KEP; it remains
  present until an external mechanism removes it. This KEP only unblocks
  the Job's own progress (see [Non-Goals](#non-goals)).
- **Narrow scope.** This KEP only helps Jobs using
  `podReplacementPolicy: Failed`. Other controllers facing the same
  underlying stuck-Pod problem (StatefulSets, DaemonSets, other queueing
  systems) get no benefit from this KEP and would need their own solution.
- **New coupling to node lifecycle.** Even though it consumes the shared
  node informer rather than performing live node lookups (see
  [Node awareness](#node-awareness)), the Job controller becomes newly
  coupled to node state and to a condition semantics owned by another
  KEP ([KEP-5683](../5683-lifecycle-conditions/README.md)). A bug or
  ambiguity in how `MaintenanceInProgress` is set or cleared can now
  affect Job controller behavior, whereas today the Job controller's
  correctness does not depend on node state at all.

## Alternatives

- **Doing nothing and relying on ecosystem workarounds.** As described in
  [Motivation](#motivation), some projects (AppWrapper, medik8s) already
  force-delete or taint Pods/nodes to work around this problem today. This
  KEP was chosen over continuing to rely on this ecosystem fragmentation,
  since it standardizes on a single, already-defined signal
  (`MaintenanceInProgress`) that any of these projects, or an
  administrator directly, can set.
- **Timeout-based termination (`kubernetes/kubernetes#134038`).** Several
  alternatives were discussed in
  [kubernetes/kubernetes#134038](https://github.com/kubernetes/kubernetes/issues/134038),
  including a Pod-level `spec.forcefulDeletionTimeoutSeconds` field, and a
  Job-level `spec.stuckTerminatingTimeout` combined with a new
  `podReplacementPolicy: FailedOrStuckTerminating` value, both of which
  would replace a stuck Pod automatically after a fixed time elapses,
  without requiring any admin signal. These were not adopted for this KEP
  because, as raised in
  [thockin's categorization of causes](https://github.com/kubernetes/kubernetes/issues/134038#issuecomment-3367353880),
  they infer node/kubelet health from elapsed time alone, which
  does not distinguish between a genuinely unreachable node (safe to
  replace) and a temporary network blip or a still-running Pod (unsafe to
  replace). This KEP instead relies on an explicit, admin-asserted
  signal (`MaintenanceInProgress`) rather than an implicit timeout,
  consistent with the WG Node Lifecycle direction of using node conditions
  for this class of problem (see [Motivation](#motivation)).
- **A new Pod-level or node-level fencing/confirmed-dead API.** Also
  discussed in the same issue: a stronger primitive that could
  authoritatively confirm a node is powered off or fenced (for example, via
  cloud-provider integration), which would remove the need to trust an
  unverified admin assertion at all. This is a substantially larger effort
  ("general node repair/fencing," see [Non-Goals](#non-goals)) that is out
  of scope for this KEP, though it could reduce or eliminate the risk
  described in [Drawbacks](#drawbacks) if built in the future.
- **Solving this in Kueue or another queueing/orchestration layer instead
  of the Job controller.** Kueue has already implemented its own fix for a
  related but distinct problem (releasing quota after
  `gracefulTerminationPeriod`, see
  [kubernetes-sigs/kueue#6872](https://github.com/kubernetes-sigs/kueue/pull/6872)).
  This does not help Jobs run without a queueing layer, and does not
  address the Job controller's own accounting (`.status.active`,
  `backoffLimit`, completion index tracking for indexed Jobs) described in
  [Design Details](#design-details), which can only be fixed inside the
  Job controller itself.

## Infrastructure Needed (Optional)

No new project infrastructure is needed.
