# KEP-6265: Surface Node Lifecycle State for Blocked Rollouts

<!-- toc -->
- [Release Signoff Checklist](#release-signoff-checklist)
- [Summary](#summary)
- [Motivation](#motivation)
  - [Goals](#goals)
  - [Non-Goals](#non-goals)
- [Proposal](#proposal)
  - [User Stories (Optional)](#user-stories-optional)
    - [Story 1: Operator Debugging a Stalled Rollout](#story-1-operator-debugging-a-stalled-rollout)
    - [Story 2: Rollout Automation](#story-2-rollout-automation)
    - [Story 3: Platform Team Coordinating Maintenance](#story-3-platform-team-coordinating-maintenance)
  - [Notes/Constraints/Caveats (Optional)](#notesconstraintscaveats-optional)
  - [Risks and Mitigations](#risks-and-mitigations)
- [Design Details](#design-details)
  - [API](#api)
  - [When the condition is set](#when-the-condition-is-set)
  - [Controller changes](#controller-changes)
  - [Test Plan](#test-plan)
      - [Prerequisite testing updates](#prerequisite-testing-updates)
      - [Unit tests](#unit-tests)
      - [Integration tests](#integration-tests)
      - [e2e tests](#e2e-tests)
  - [Graduation Criteria](#graduation-criteria)
    - [Alpha](#alpha)
    - [Beta](#beta)
    - [GA](#ga)
    - [Deprecation](#deprecation)
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
- [ ] User-facing documentation has been created in [kubernetes/website], for publication to [kubernetes.io]
- [ ] Supporting documentation—e.g., additional design documents, links to mailing list discussions/SIG meetings, relevant PRs/issues, release notes

[kubernetes.io]: https://kubernetes.io/
[kubernetes/enhancements]: https://git.k8s.io/enhancements
[kubernetes/kubernetes]: https://git.k8s.io/kubernetes
[kubernetes/website]: https://git.k8s.io/website

## Summary

StatefulSets preserve stable Pod identity and roll out Pods in order, so the
controller cannot create a replacement Pod with the same identity while the
old Pod still exists. When a Pod is unavailable because its Node is shutting
down, unreachable, or under maintenance, this can block the rollout with no
indication of why. This KEP proposes surfacing Node lifecycle state (as
reported by [KEP-5683: Node Lifecycle Conditions](/keps/sig-node/5683-lifecycle-conditions))
in StatefulSet status, so it's clear when a rollout is waiting on Node
lifecycle activity rather than the new revision, storage attachment, or
scheduling.

## Motivation

A single Pod stuck on a Node affected by lifecycle activity (e.g. a Node
condition such as `MaintenanceInProgress`) can halt StatefulSet rollout
progress. Today, users and rollout tooling can only see that the
StatefulSet is waiting for Pods to become ready, with no way to tell whether
that's caused by the new revision, storage attachment, scheduling, or
intentional Node maintenance.

The StatefulSet controller blocks by design. It never creates a replacement
Pod until the old Pod with the same name is fully gone from the API, and with
`podManagementPolicy: OrderedReady` it updates and waits on one Pod at a time
in ordinal order. A single Pod that is unavailable because its Node is being
drained, shut down, or maintained therefore stalls everything behind it. Node
lifecycle activity is the one cause of a stall that is intentional and
temporary, and it is also the one the StatefulSet status gives no hint about.

### Goals

- Report, in StatefulSet status, when a StatefulSet Pod that the controller is
  waiting on during a rollout is unavailable and its Node reports a lifecycle
  condition from KEP-5683, rather than another cause.
- Make that report machine-readable, so rollout tooling can tell a
  lifecycle-caused stall from a revision-caused one without cross-referencing
  Pods and Nodes.
- Add no new API calls to the write path and no new fields to the API schema.

### Non-Goals

- Changing StatefulSet rollout, eviction, or Pod replacement behavior. The
  controller still waits exactly as it does today.
- Reporting stalls caused by Node `Ready=False` or `Ready=Unknown`, or by the
  `node.kubernetes.io/unreachable` and `not-ready` taints. Those are failure
  signals, not the lifecycle state defined by KEP-5683. They could be a
  follow-up.
- Reporting stalls outside a rollout, such as an `OrderedReady` scale-up
  waiting on a Pod. Only rollouts are covered.
- Defining or setting the Node lifecycle conditions. That belongs to KEP-5683.
- ReplicaSet or other controller coordination with Node lifecycle state
  (tracked separately, see KEP-5683).
- Changing `kubectl rollout status` output. It could read the new condition
  later.

## Proposal

Extend StatefulSet status reporting so that when a Pod blocking a rollout is
on a Node reporting a lifecycle condition (e.g. `MaintenanceInProgress`),
that context is visible on the StatefulSet, instead of only a generic
"waiting for Pods" signal.

### User Stories (Optional)

The Node lifecycle conditions this KEP reads are defined in
[KEP-5683](/keps/sig-node/5683-lifecycle-conditions). Story 4 of that KEP
covers the same gap for DaemonSets, tracked in
[kubernetes/kubernetes#139226](https://github.com/kubernetes/kubernetes/issues/139226).
This KEP is the StatefulSet counterpart.

<!--
TODO: add StatefulSet-specific tracking issues to each story below. Candidates
to search in kubernetes/kubernetes and kubernetes/enhancements: StatefulSet
rolling update stuck on an unready Pod, `kubectl rollout status` hanging, and
StatefulSet Pods stuck Terminating on a drained Node.
-->

#### Story 1: Operator Debugging a Stalled Rollout

As a cluster operator, I update a StatefulSet and the rollout stops at `web-2`.
`kubectl rollout status` only reports that it is waiting for Pods to be ready.
I check the new image, the probes, and the PVC before finding that the Node was
cordoned and drained for maintenance. I want the StatefulSet to say so, so I
don't treat node maintenance as an application or storage problem.

#### Story 2: Rollout Automation

As a rollout tool (a GitOps controller or an operator) that watches a
StatefulSet, I see no progress and hit my own timeout. I can't tell a stall
caused by a bad revision, where I should roll back, from one caused by a Node in
maintenance, where I should wait. I want a condition I can read to decide.

#### Story 3: Platform Team Coordinating Maintenance

As a platform engineer running node maintenance, I want to see which
StatefulSets have a rollout waiting on a Node I have marked
`MaintenanceInProgress` or `DrainInProgress`, so I can finish or reschedule the
maintenance instead of leaving rollouts stuck behind it. This is the StatefulSet
case of the coordination problem in KEP-5683 stories 5 and 6.

### Notes/Constraints/Caveats (Optional)

- This KEP depends on the Node lifecycle conditions defined in
  [KEP-5683](/keps/sig-node/5683-lifecycle-conditions) being set by an admin or
  maintenance controller. Without them the new condition is never set, which is
  harmless.
- The condition reports correlation, not proof of cause. A Pod can be
  unavailable for several reasons at once. The condition means "the Pod the
  rollout is waiting on is unavailable and its Node reports lifecycle state",
  and the message says so.
- A Node `Drained` or `MaintenancePlanned` condition does not set the
  condition. `MaintenancePlanned` means disruption has not started, so it
  explains no unavailable Pod. `Drained` means Pods have already left, so it
  explains nothing about a Pod that is still waiting.

### Risks and Mitigations

- **Misleading attribution.** A Pod can fail for an unrelated reason while its
  Node is in maintenance. The condition is set only when the blocking Pod is
  itself unavailable, and its message uses neutral wording ("is on node n1
  reporting MaintenanceInProgress"). It never claims to be the sole cause.
- **Stale Node conditions.** If an admin never clears a lifecycle condition, the
  StatefulSet condition stays set. It is recomputed from live Node state on
  every sync and never latched, so it clears as soon as the Node condition
  clears or the Pod moves.
- **New Node watch in kube-controller-manager.** The StatefulSet controller does
  not read Nodes today. See Design Details for how Node events are scoped to
  keep this cheap. It needs read-only access to Nodes, added to the controller's
  RBAC.
- **Status churn.** The message names a Pod and a Node, so a changing blocking
  Pod rewrites the message. Status is written only when the condition changes,
  and the existing status update path is reused, so there are no extra writes.

## Design Details

### API

No new fields or types are added to the API schema. `StatefulSetStatus` already
has a `conditions` list of `StatefulSetCondition` (`type`, `status`,
`lastTransitionTime`, `reason`, `message`), keyed by `type`. This KEP adds one
well-known condition type constant, `RolloutBlockedByNodeLifecycle`, and three
reason constants to `k8s.io/api/apps/v1`.

```yaml
status:
  conditions:
  - type: RolloutBlockedByNodeLifecycle
    status: "True"
    reason: NodeMaintenanceInProgress
    message: "pod web-2 is on node n1 reporting MaintenanceInProgress"
    lastTransitionTime: "2026-10-01T12:00:00Z"
```

The condition is present with `status: "True"` while a rollout is blocked as
defined below, and is removed otherwise. It is never written as `False`. This
keeps StatefulSets that never see Node lifecycle activity free of extra status.

| Node condition (KEP-5683) | `reason` |
|---|---|
| `GracefulNodeShutdownInProgress=True` | `NodeGracefulShutdownInProgress` |
| `DrainInProgress=True` | `NodeDrainInProgress` |
| `MaintenanceInProgress=True` | `NodeMaintenanceInProgress` |

If a Node reports more than one, the reason follows the order in the table and
the message lists all of them. `MaintenancePlanned` and `Drained` are
deliberately excluded (see Notes/Constraints/Caveats).

### When the condition is set

The condition is set during a sync of the StatefulSet when all of the following
hold:

1. A rollout is in progress: `status.currentRevision` differs from
   `status.updateRevision`, so the controller still has Pods to update.
2. The controller is waiting on one or more Pods. These are the same Pods it
   already waits on today before moving to the next ordinal: a Pod that is not
   Running and Ready (or not Available when `minReadySeconds` is set), or a Pod
   that is terminating. Under `Parallel` pod management, or when
   `maxUnavailable` allows several, there can be more than one.
3. At least one of those Pods is assigned to a Node (`spec.nodeName` is set)
   that reports one of the lifecycle conditions in the table with
   `status: "True"`.

The message names the lowest-ordinal Pod that meets (3) and its Node. If other
waiting Pods also meet it, the message appends "and N other pods".

A Pod that is Ready on a Node in maintenance blocks nothing, so it does not set
the condition. A Pod with no `spec.nodeName` (still Pending) does not set it,
because the scheduler is the cause there.

The controller does nothing new with the result. The existing wait, update, and
Pod replacement logic is untouched.

### Controller changes

Changes are confined to `pkg/controller/statefulset`:

- `stateful_set_control.go`: while computing `StatefulSetStatus`, evaluate the
  rules above for the Pods the controller is waiting on and set or remove the
  condition on the status it already writes. No extra API call is made.
- `stateful_set.go`: add a Node informer and a Node lister to the controller.
  The informer's event handler enqueues StatefulSets only when a Node's
  lifecycle conditions in the table change, and only the StatefulSets that own
  a Pod on that Node. Ordinary Node heartbeat and status updates are filtered
  out by comparing the old and new condition sets. Finding the Pods on a Node
  uses a `spec.nodeName` index on the existing Pod informer, so a Node event
  does not scan every Pod.
- The condition is recomputed from live Pod and Node state on each sync. It
  keeps no state of its own.
- An Event (`RolloutBlockedByNodeLifecycle`, type `Normal`) is emitted when the
  condition transitions to `True`, and not on later syncs, so the signal is also
  visible in `kubectl describe`.
- The `system:controller:statefulset-controller` RBAC role gains
  `get`, `list`, and `watch` on `nodes`.

Everything is behind the `StatefulSetNodeLifecycleStatus` feature gate in
kube-controller-manager. With the gate off, the Node informer handler is not
registered and the condition is never written.

### Test Plan

[x] I/we understand the owners of the involved components may require updates to
existing tests to make this code solid enough prior to committing the changes
necessary to implement this enhancement.

##### Prerequisite testing updates

None are required. The existing StatefulSet controller unit tests already cover
the "waiting on a Pod" paths this KEP reads from.

##### Unit tests

All in `pkg/controller/statefulset`. Current coverage will be recorded when the
implementation PR opens.

- Condition set for each of the three Node conditions, with the matching reason.
- Reason precedence when a Node reports several conditions.
- Not set when no rollout is in progress.
- Not set when the blocking Pod is Ready, or has no `spec.nodeName`.
- Not set for `Drained`, `MaintenancePlanned`, and `Ready=Unknown` Nodes.
- Under `Parallel` and `maxUnavailable > 1`: lowest-ordinal Pod named, and the
  "and N other pods" suffix.
- Condition removed when the Node condition clears, the Pod becomes Ready, or the
  Pod is recreated on another Node.
- Feature gate off: no condition and no Node informer handler.
- Node event filter: heartbeat-only updates enqueue nothing, and lifecycle
  condition changes enqueue only owners of Pods on that Node.

##### Integration tests

In `test/integration/statefulset`:

- Create a StatefulSet, start a rollout, set `MaintenanceInProgress=True` on the
  Node of the blocking Pod, and verify the condition appears. Clear the Node
  condition and verify it is removed.
- Verify rollout behavior is identical with the gate on and off, which covers the
  non-goal of not changing it.

##### e2e tests

In `test/e2e/apps/statefulset.go`: a rollout blocked by a Pod on a Node that has
a lifecycle condition set shows the condition, and the rollout completes after
the Node condition is cleared. Added for Beta.

### Graduation Criteria

#### Alpha

- `StatefulSetNodeLifecycleStatus` feature gate, default off.
- Condition, reasons, controller changes, and RBAC implemented as above.
- Unit and integration tests in place.

#### Beta

- KEP-5683 conditions are at Beta or later, so the signal is expected to be
  populated in real clusters.
- e2e test in place and passing in CI without flakes.
- Feedback from at least one rollout tool or operator that reads the condition.
- Decision on follow-ups: stalls from `Ready=Unknown` or taints, a
  `kubectl rollout status` message, and a metric.
- Feature gate default on.

#### GA

- At least two releases at Beta with no open correctness issues.
- Feature gate locked on and then removed per the deprecation policy.

#### Deprecation

N/A. The feature adds a status condition and has no behavior to deprecate. If
the condition is replaced by a cross-controller mechanism, it goes through the
usual API deprecation process.

### Upgrade / Downgrade Strategy

- **Upgrade:** enabling the gate adds the condition to StatefulSets with a
  blocked rollout. No migration is needed. Nothing else changes.
- **Gate disabled:** the controller removes any existing
  `RolloutBlockedByNodeLifecycle` condition on its next sync of each StatefulSet,
  so no stale value is left behind.
- **Downgrade to a version without the feature:** the older controller does not
  know the condition and leaves it in place. It is inert, because nothing
  reads it, and it is recomputed on the next upgrade.

### Version Skew Strategy

Only kube-controller-manager changes. The API schema is unchanged, so
kube-apiserver and kubelet versions do not matter. Node conditions are free-form
strings to the API server, so the controller can read KEP-5683 conditions
regardless of apiserver version. An older kube-controller-manager ignores the
Node conditions and does not set the condition. Clients that do not know the
condition type ignore it.

## Production Readiness Review Questionnaire

### Feature Enablement and Rollback

###### How can this feature be enabled / disabled in a live cluster?

- [x] Feature gate (also fill in values in `kep.yaml`)
  - Feature gate name: `StatefulSetNodeLifecycleStatus`
  - Components depending on the feature gate: kube-controller-manager

###### Does enabling the feature change any default behavior?

No. It adds a status condition and an Event. No rollout, eviction, or Pod
replacement behavior changes. Tooling that treats any unknown StatefulSet
condition as an error would see a new condition type, which is unlikely.

###### Can the feature be disabled once it has been enabled (i.e. can we roll back the enablement)?

Yes. Disabling the gate stops the controller from setting the condition and
removes it on the next sync. No data or workloads are affected.

###### What happens if we reenable the feature if it was previously rolled back?

The condition is recomputed from live state on the next sync of each StatefulSet.
Nothing is persisted that could be out of date.

###### Are there any tests for feature enablement/disablement?

Yes. The unit tests above cover the gate on and off, including removal of an
existing condition when the gate is turned off.

### Rollout, Upgrade and Rollback Planning

###### How can a rollout or rollback fail? Can it impact already running workloads?

The new code reads Pod and Node state and writes only a status condition. It
cannot change running Pods. The failure to watch for is the extra Node watch
adding load to kube-controller-manager, and a bug in the event filter that
enqueues too many StatefulSets on Node updates. Neither affects workloads, and
both are removed by disabling the gate.

###### What specific metrics should inform a rollback?

Rising `workqueue_depth` or `workqueue_queue_duration_seconds` for the
`statefulset` queue, and elevated kube-controller-manager CPU or memory after
enabling the gate.

###### Were upgrade and rollback tested? Was the upgrade->downgrade->upgrade path tested?

Will be covered by the enablement/disablement unit tests and a manual
upgrade, downgrade, upgrade run before Beta.

###### Is the rollout accompanied by any deprecations and/or removals of features, APIs, fields of API types, flags, etc.?

No.

### Monitoring Requirements

###### How can an operator determine if the feature is in use by workloads?

Run `kubectl get statefulset -A -o json` and look for StatefulSets with a
`RolloutBlockedByNodeLifecycle` condition. A `RolloutBlockedByNodeLifecycle`
Event is also recorded on each transition.

###### How can someone using this feature know that it is working for their instance?

- [x] Events
  - Event Reason: `RolloutBlockedByNodeLifecycle`
- [x] API .status
  - Condition name: `RolloutBlockedByNodeLifecycle`
  - Other field: `reason` identifies which Node condition applied.

###### What are the reasonable SLOs (Service Level Objectives) for the enhancement?

The condition appears or clears within one StatefulSet sync of a Node lifecycle
condition changing, in practice a few seconds under normal controller load. No
existing StatefulSet SLO is affected.

###### What are the SLIs (Service Level Indicators) an operator can use to determine the health of the service?

- [x] Other (treat as last resort)
  - Details: the existing `workqueue_*` metrics for the `statefulset` queue. No
    new metric is added for Alpha.

###### Are there any missing metrics that would be useful to have to improve observability of this feature?

A gauge of StatefulSets currently reporting the condition, by reason, would help
fleet operators. It is proposed for Beta rather than Alpha.

### Dependencies

###### Does this feature depend on any specific services running in the cluster?

- Node lifecycle conditions (KEP-5683)
  - Usage description: the signal that triggers the condition.
  - Impact of its outage on the feature: the condition is never set. Rollouts
    behave as today.
  - Impact of its degraded performance or high-error rates on the feature: a
    delayed or missing Node condition means a delayed or missing StatefulSet
    condition.

### Scalability

###### Will enabling / using this feature result in any new API calls?

A new list and watch of Nodes from kube-controller-manager. This is a shared
informer, so no per-StatefulSet calls are made. Status writes reuse the existing
status update and happen only when the condition changes. At most one extra
Event is created per transition.

###### Will enabling / using this feature result in introducing new API types?

No.

###### Will enabling / using this feature result in any new calls to the cloud provider?

No.

###### Will enabling / using this feature result in increasing size or count of the existing API objects?

Only StatefulSets with a blocked rollout, and only while blocked: one condition
of roughly 300 bytes. No new objects apart from Events.

###### Will enabling / using this feature result in increasing time taken by any operations covered by existing SLIs/SLOs?

No. The evaluation is a lookup of at most a few Pods and their Nodes from local
caches during a sync that already runs.

###### Will enabling / using this feature result in non-negligible increase of resource usage (CPU, RAM, disk, IO, ...) in any components?

A small amount of memory in kube-controller-manager for the Node cache, which
other controllers (the node lifecycle and daemonset controllers) already hold
through the shared informer factory. The `spec.nodeName` Pod index adds a small
amount of memory proportional to the Pod count.

###### Can enabling / using this feature result in resource exhaustion of some node resources (PIDs, sockets, inodes, etc.)?

No.

### Troubleshooting

###### How does this feature react if the API server and/or etcd is unavailable?

Like the rest of the StatefulSet controller: it cannot read or write, so the
condition stops updating. It is recomputed from the caches once the API server
returns.

###### What are other known failure modes?

- **Condition set but the rollout is blocked by something else as well.**
  - Detection: the Pod's own status and Events show another cause, such as an
    image pull failure.
  - Diagnostics: the condition is correlation only. Check the blocking Pod named
    in the message.
  - Mitigations: fix the other cause. Clear the Node condition if it is stale.
- **Condition stays set after maintenance ends.**
  - Detection: the Node still reports a lifecycle condition.
  - Diagnostics: `kubectl get node <n> -o jsonpath='{.status.conditions}'`.
  - Mitigations: the actor that set the Node condition clears it. The StatefulSet
    condition follows.
- **Condition never appears for a stalled rollout.**
  - Detection: the rollout is stalled, but the condition is absent.
  - Diagnostics: confirm the gate is on, the Node has one of the three
    conditions set to `True`, the blocking Pod has a `spec.nodeName`, and a
    rollout is in progress. A stall caused by `Ready=Unknown` or a taint is out
    of scope.
  - Mitigations: none needed. This is expected outside the defined cases.

###### What steps should be taken if SLOs are not being met to determine the problem?

Check kube-controller-manager logs for the StatefulSet controller, and the
`statefulset` workqueue metrics. If load is the issue, disable the gate.

## Implementation History

- 2026-09-27: Initial KEP created from [enhancement issue #6265](https://github.com/kubernetes/enhancements/issues/6265).

## Drawbacks

- The StatefulSet controller becomes aware of Nodes, which it deliberately was
  not before. That adds a watch and a coupling to KEP-5683 condition names.
- The signal is only as good as the actor setting the Node conditions. In
  clusters where nobody sets them, the feature does nothing.
- It addresses visibility only. A stalled rollout still needs a human or a
  tool to act.

## Alternatives

- **Events only.** Easy, but Events expire, are not a stable API, and are hard
  for tooling to query. A condition is queryable and persists. This KEP emits an
  Event as well, for `kubectl describe`.
- **A condition on the Pod**, set by the node lifecycle controller. That is a
  more general fix, but it sits in a different SIG's controller and leaves the
  question "why is my StatefulSet rollout stuck" unanswered at the StatefulSet
  level. It does not exclude this KEP, which could read such a Pod condition
  later.
- **A cross-controller shared library or API** so Deployments, DaemonSets, and
  StatefulSets all report lifecycle-caused unavailability the same way. The
  better long-term shape, but larger. DaemonSet is already going its own way
  (`status.unavailable`, see KEP-5683 story 4). Starting with StatefulSet gives
  concrete input for such a design.
- **A count field in status, like the DaemonSet proposal.** StatefulSet has no
  `unavailable` count to attach it to, and a condition carries a reason and a
  message that a count cannot.
- **Resolve it client-side in `kubectl rollout status`.** Helps humans but not
  controllers or GitOps tools, and every client would repeat the Pod-to-Node
  lookup.
- **Change behavior**, such as force-replacing Pods on Nodes in maintenance.
  Rejected. It breaks StatefulSet's at-most-one guarantee, and is a non-goal.

## Infrastructure Needed (Optional)

None.
