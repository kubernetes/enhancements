# KEP-6250: Maintenance-aware DaemonSet rollouts

<!-- toc -->
- [Release Signoff Checklist](#release-signoff-checklist)
- [Summary](#summary)
- [Motivation](#motivation)
  - [Goals](#goals)
  - [Non-Goals](#non-goals)
- [Proposal](#proposal)
  - [Update nodes that have a higher probability of success first](#update-nodes-that-have-a-higher-probability-of-success-first)
  - [Report Node context in DaemonSet status](#report-node-context-in-daemonset-status)
  - [User Stories](#user-stories)
    - [Story 1](#story-1)
    - [Story 2](#story-2)
    - [Story 3](#story-3)
  - [Notes/Constraints/Caveats (Optional)](#notesconstraintscaveats-optional)
  - [Risks and Mitigations](#risks-and-mitigations)
- [Design Details](#design-details)
  - [Node classification](#node-classification)
  - [Candidate selection in <code>rollingUpdate</code>](#candidate-selection-in-rollingupdate)
  - [Node event handling](#node-event-handling)
  - [DaemonSet status](#daemonset-status)
  - [Feature gate](#feature-gate)
  - [Metrics](#metrics)
  - [Open Questions](#open-questions)
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

During a `RollingUpdate`, the DaemonSet controller picks rollout candidates in an unspecified order. It doesn't prefer nodes where the updated Pod is likely to become available. A node that isn't ready, or is being drained or maintained, can be picked early. Its new Pod then gets stuck and holds part of the rollout budget until the node recovers, which slows or stalls the rollout on every other node.

This KEP makes the DaemonSet controller a reader of Node conditions for rollout ordering. During a rollout, the controller targets nodes with a higher probability of success first. Nodes where `Ready` isn't `True`, or where `DrainInProgress=True` or `MaintenanceInProgress=True`, are targeted after the rest.

Only the order changes. Node eligibility, Pod creation and recovery, and `maxUnavailable` and `maxSurge` semantics stay as they are today.

The KEP also adds status fields that report how many unavailable DaemonSet nodes are NotReady, draining, or under maintenance. Rollout tooling can use them to tell a bad revision apart from Node lifecycle activity without listing Nodes.

## Motivation

A DaemonSet rolling update keeps at most `maxUnavailable` nodes unavailable, or with `maxSurge`, at most `maxSurge` extra Pods. Inside that budget, `rollingUpdate` picks candidates while iterating a Go map, so the order is effectively random.

The order matters when some nodes are unlikely to run the updated Pod. Consider `maxUnavailable: 1` and a node under maintenance. If the controller picks it first, it deletes the old Pod, the new Pod can't become available until maintenance ends, and the rollout waits for it. Every other node stays on the old revision the whole time. In a large cluster with a percentage budget, a handful of these nodes can consume the entire budget. [k/k#138240] describes this in clusters with 500+ nodes, where the only workaround today is raising `maxUnavailable` by hand.

Status has a related gap. `numberUnavailable` is an aggregate. It can't tell whether the remaining unavailable Pods are caused by the new revision or by nodes that are NotReady, draining, or under maintenance. [k/k#139226] describes rollout verifiers that list Pods and Nodes to answer that question. Every tool reimplements the attribution, and the extra LIST calls add API server load during periods that are already busy.

[KEP-5683] added well-known Node lifecycle conditions, including `DrainInProgress` and `MaintenanceInProgress`, and deferred how controllers consume them. This KEP defines the DaemonSet controller as one of those readers.

### Goals

- During a `RollingUpdate`, target nodes with a higher probability of success first.
- Keep every other DaemonSet behavior unchanged. This includes node eligibility, Pod creation and recovery, and `maxUnavailable` and `maxSurge` accounting.
- Report, in DaemonSet status, unavailable nodes that are NotReady, draining, or under maintenance.
- React to changes in the relevant Node conditions without reacting to heartbeats.

### Non-Goals

- Changing `maxUnavailable` or `maxSurge` accounting, or adding a separate budget for lifecycle activity.
- Changing which nodes run a DaemonSet Pod, or how missing, failed, or unavailable Pods are recovered.
- Writing or clearing Node lifecycle conditions. Writers are external to this KEP.
- Reading `GracefulNodeShutdownInProgress`. Reading this condition and suppression of Pod creation during Graceful Node Shutdown belong to [KEP-6249].
- Coordinating ownership between condition writers. That's tracked in [#6430].
- Changing `OnDelete` behavior.
- Fairness between nodes with a higher and lower probability of success.

## Proposal

### Update nodes that have a higher probability of success first

For a DaemonSet rollout, the goal is to update nodes in the order of their probability of success. For instance, a node with `Ready=True` that isn't being drained or maintained would be targeted first.

The controller places each node into one of two buckets:

1. **Higher probability of success**
   - `Ready=True`, and
   - `DrainInProgress` isn't `True`, and
   - `MaintenanceInProgress` isn't `True`.
2. **Lower probability of success**
   - `Ready` isn't `True`, or
   - `DrainInProgress=True`, or
   - `MaintenanceInProgress=True`.

For `DrainInProgress` and `MaintenanceInProgress` conditions, only `status: "True"` has an effect. `False` and `Unknown` status, and an absent condition don't have any influence on the daemonset rollout. For the `Ready` condition, `False` and `Unknown` status, and an absent condition all place the node in the lower bucket.

These conditions don't guarantee that an update will succeed or fail. A `Ready` node can still fail the update, and a `NotReady` node can recover. They're signals for which nodes to try first. That's why the bucket only changes order. It never prevents a node from being updated.

The controller selects elective rollout candidates from the higher bucket first. Candidates from the lower bucket only use the budget left over after every candidate in the higher bucket has been selected. This applies to both `maxUnavailable` and `maxSurge` rollouts.

| Behavior | With this KEP |
|---|---|
| Which nodes run a DaemonSet Pod | Unchanged |
| Creating a missing Pod, deleting a failed or succeeded Pod | Unchanged |
| Replacing an old Pod that is already unavailable, with or without `maxSurge` | Unchanged and not subject to ordering |
| `maxUnavailable` and `maxSurge` accounting | Unchanged |
| `OnDelete` | Unchanged |
| Order of elective candidates, meaning nodes with an available old Pod | Higher-probability bucket first |

### Report Node context in DaemonSet status

The controller reports how many unavailable DaemonSet nodes are NotReady, draining, or under maintenance.

Existing status fields keep their current meaning. The new fields report the Node context where unavailability exists. They don't claim that the Node condition caused it.

### User Stories

#### Story 1

As a platform engineer, I roll out a new revision of a CNI DaemonSet with `maxUnavailable: 1`. A maintenance controller has set `MaintenanceInProgress=True` on two nodes that are being reimaged. Today the rollout can pick one of those nodes first and wait on it until reimaging finishes. With this KEP, the rollout updates every other node first and reaches the maintenance nodes last.

#### Story 2

As the owner of a rollout verifier, I need to decide whether a DaemonSet rollout failed. Status shows `numberUnavailable: 5`. Today I list Pods and Nodes to find out whether those five nodes are NotReady or under maintenance. With this KEP, status tells me how many of the five are on NotReady, draining, or maintenance nodes, and I can treat only the remainder as a possible bad revision.

#### Story 3

As an operator of a cluster with 2,000 nodes, I usually have a few NotReady nodes at any time. My DaemonSet rollouts use `maxUnavailable: 5%`. Today the controller can spend that budget on NotReady nodes whose new Pods won't start. With this KEP, the rollout spends the budget on ready nodes first.

### Notes/Constraints/Caveats (Optional)

**Ordering doesn't create budget.** All unavailable Pods still count against `maxUnavailable`, including Pods on NotReady, draining, or maintenance nodes. If existing unavailability already uses the whole budget, the rollout waits as it does today. For example, with 100 nodes, `maxUnavailable: 3`, and three NotReady nodes whose Pods are unavailable, the controller replaces those three old Pods immediately, as it does today. Their nodes stay unavailable until they recover, so no available old Pod on another node is updated in the meantime. This KEP makes sure the budget isn't spent on lower-probability nodes while higher-probability nodes are waiting.

**Graceful Node Shutdown.** This KEP doesn't read `GracefulNodeShutdownInProgress`. During Graceful Node Shutdown the kubelet reports `Ready=False`, so a shutting-down node lands in the lower bucket through `Ready`. This KEP doesn't change Pod recovery on a shutting-down node. That includes today's create/reject churn for DaemonSets that tolerate the `node.kubernetes.io/not-ready:NoSchedule` taint, which is tracked in [KEP-6249].

**Writer-agnostic reader.** The controller reads the condition `type` and `status` only. It doesn't depend on which actor wrote the condition. Ownership and coordination between writers will be handled in [#6430].

**Order within a bucket.** The order of candidates within a bucket stays unspecified, as it is today.

### Risks and Mitigations

| Risk | Mitigation |
|---|---|
| A stale or incorrect `DrainInProgress=True` or `MaintenanceInProgress=True` stays on a node. | The node only moves to the back. It's still updated once the higher-bucket candidates are selected. The worst case is today's behavior. The status fields will count the node under the stale condition, which is visible to the operator. |
| `Ready` flaps during a rollout. | Buckets are computed on every sync. A node can move between buckets, which changes when it's picked but never whether it's picked. |
| Users read the new status fields as a cause. | The field documentation says they report Node context. The controller doesn't claim the condition caused the unavailability. |
| Condition changes cause extra DaemonSet syncs. | The controller compares only the `status` of the three conditions. Heartbeat updates don't trigger syncs. Condition flips are rare compared to heartbeats. |

## Design Details

### Node classification

A single helper classifies a node. The rollout path and the status computation use the same helper so they can't drift.

```go
type rolloutProbability int

const (
	higherProbability rolloutProbability = iota
	lowerProbability
)

func nodeRolloutProbability(node *v1.Node) rolloutProbability {
	if !nodeConditionIsTrue(node, v1.NodeReady) ||
		nodeConditionIsTrue(node, v1.NodeDrainInProgress) ||
		nodeConditionIsTrue(node, v1.NodeMaintenanceInProgress) {
		return lowerProbability
	}
	return higherProbability
}
```

### Candidate selection in `rollingUpdate`

`rollingUpdate` has two branches. Both build a list of nodes that must be acted on and a list of elective candidates, then cap the elective list by the remaining budget.

| Branch | Must act on | Elective candidates | Budget |
|---|---|---|---|
| `maxSurge == 0` | `allowedReplacementPods`, old Pods that are already unavailable | `candidatePodsToDelete`, available old Pods | `maxUnavailable - numUnavailable` |
| `maxSurge > 0` | `allowedNewNodes`, nodes whose old Pod is already unavailable | `candidateNewNodes`, nodes with an available old Pod and no new Pod | `maxSurge - numSurge` |

Today both branches collect elective candidates while iterating `nodeToDaemonPods` (stored in a map), and then select the first ones that fit in the remaining budget. So which candidates are selected depends on map iteration order (which is effectively random). Candidates are only selected while the number of pods unavailable is less than maxUnavailable.

With the feature enabled, each branch stable-sorts the elective candidates by `nodeRolloutProbability`, higher bucket first, then picks as many pods as the budget allows. Everything else stays the same, including the budget computation and how many candidates are selected.

The must-act lists, the `shouldNotRunPodsToDelete` path for nodes that no longer match scheduling constraints, and `manage` are unchanged.

### Node event handling

Today the controller only processes a Node update when the node's labels or taints change (`shouldIgnoreNodeUpdate`), so all condition changes are ignored. With the feature enabled, `updateNode` also queues the node for processing when the `status` of `Ready`, `DrainInProgress`, or `MaintenanceInProgress` changes. Updates that only refresh `lastHeartbeatTime` are still ignored, so heartbeats don't queue anything.

`syncNodeUpdate` enqueues a DaemonSet only when it should create or remove a Pod on the node, or when its misscheduled count is non-zero. A bucket change doesn't match any of those. With the feature enabled, `syncNodeUpdate` also enqueues every DaemonSet that has a Pod on the node. This keeps status attribution current and lets an in-progress rollout reorder on its next sync.

### DaemonSet status

The new fields are additive and follow the existing flat counter model in `DaemonSetStatus`.

```go
type DaemonSetStatus struct {
	// ... existing fields ...

	// numberUnavailableOnNotReadyNodes is the number of nodes counted in
	// numberUnavailable whose Ready condition isn't True.
	// The numberUnavailableOn* counts are mutually exclusive. A node that
	// matches more than one condition is counted under maintenance first,
	// then draining, then NotReady.
	// This reports Node context, not the cause of unavailability.
	// +featureGate=DaemonSetNodeConditionAwareRollout
	// +optional
	NumberUnavailableOnNotReadyNodes *int32

	// numberUnavailableOnDrainingNodes is the number of nodes counted in
	// numberUnavailable with DrainInProgress=True.
	// Mutually exclusive with the other numberUnavailableOn* counts. A node
	// that is also under maintenance is counted only in
	// numberUnavailableOnMaintenanceNodes.
	// +featureGate=DaemonSetNodeConditionAwareRollout
	// +optional
	NumberUnavailableOnDrainingNodes *int32

	// numberUnavailableOnMaintenanceNodes is the number of nodes counted in
	// numberUnavailable with MaintenanceInProgress=True.
	// Mutually exclusive with the other numberUnavailableOn* counts, and
	// takes precedence over them.
	// +featureGate=DaemonSetNodeConditionAwareRollout
	// +optional
	NumberUnavailableOnMaintenanceNodes *int32
}
```

Each unavailable node is counted under at most one reason, so the three `numberUnavailableOn*` fields never sum to more than `numberUnavailable`. When a node matches more than one condition, the precedence is maintenance, then drain, then NotReady. Explicit lifecycle intent is reported before the generic readiness signal.

`updateDaemonSetStatus` computes the fields in its existing per-node loop, using `nodeRolloutProbability` and the same availability check it uses for `numberAvailable`.

With the feature disabled, the controller doesn't set the fields and clears them on its next status update. `storeDaemonSetStatus` skips the API call when the status hasn't changed, and today it only compares the existing fields. Its comparison will also include the new fields, so a controller with the gate off sees a stale value and writes it as unset.

With the gate off, kube-apiserver drops these fields from status updates unless the stored object already has them. This is the standard pattern for feature-gated fields, and matches existing status fields such as `Node.Status.Features`.

### Feature gate

- **Name:** `DaemonSetNodeConditionAwareRollout`
- **Components:** kube-apiserver, kube-controller-manager
- **Depends on:** `NodeLifecycleConditions`, declared in `defaultKubernetesFeatureGateDependencies`

The new gate keeps the DaemonSet reader on its own graduation track, separate from the condition vocabulary in [KEP-5683].

### Metrics

- **`daemonset_controller_rollout_candidates_total`**. A counter incremented when `rollingUpdate` selects an elective candidate.
  - **Label `bucket="higher"|"lower"`**. The bucket of the selected node.

A rising `lower` count while `higher` candidates still exist would indicate a bug in ordering. It also shows how often rollouts reach lower-probability nodes.

### Open Questions

<<[UNRESOLVED status API]>>
The field set and names above are a starting point. Questions for SIG Apps and API review:

- Should we add new fields to the status to indicate the node context?
- If yes,
  - Are the names clear? `numberUnavailableOnNotReadyNodes` follows the proposal in [k/k#139226].
  - Is maintenance, then drain, then NotReady the right precedence?
<<[/UNRESOLVED]>>

<<[UNRESOLVED Ready-only ordering]>>
Ordering by `Ready` alone doesn't depend on Node lifecycle conditions or any API change. Rollout order isn't specified today. Should Ready-based ordering be accepted as a standalone improvement ahead of this KEP, with this KEP adding `DrainInProgress`, `MaintenanceInProgress`, and the status fields?
<<[/UNRESOLVED]>>

<<[UNRESOLVED feature gate name]>>
`DaemonSetNodeConditionAwareRollout` describes what the controller reads, Node conditions, and covers both rollout ordering and the status fields. It differs from the tracking issue title, which only mentions maintenance. Is this name acceptable?
<<[/UNRESOLVED]>>

### Test Plan

[x] I/we understand the owners of the involved components may require updates to
existing tests to make this code solid enough prior to committing the changes necessary
to implement this enhancement.

##### Prerequisite testing updates

None.

##### Unit tests

Coverage measured locally on `master` with `go test -cover`:

- `k8s.io/kubernetes/pkg/controller/daemon`: `2026-10-19` - `71.9%`
- `k8s.io/kubernetes/pkg/controller/daemon/util`: `2026-10-19` - `67.7%`
- `k8s.io/kubernetes/pkg/registry/apps/daemonset`: `2026-10-19` - `17.6%`

New tests:

- `nodeRolloutProbability` for every combination of `Ready`, `DrainInProgress`, and `MaintenanceInProgress`, including `Unknown` and absent conditions.
- `rollingUpdate` with `maxSurge == 0`. Higher-bucket candidates are selected before lower-bucket ones. Lower-bucket candidates use only leftover budget. Already unavailable old Pods are replaced regardless of bucket. The number of selected candidates is unchanged.
- `rollingUpdate` with `maxSurge > 0`. The same cases for surge candidates and for `allowedNewNodes`.
- `shouldIgnoreNodeUpdate`. A `status` change of the conditions relevant to this KEP passes the filter. A heartbeat-only update doesn't. With the gate off, conditions are ignored.
- `syncNodeUpdate`. A condition change enqueues DaemonSets with a Pod on the node.
- `updateDaemonSetStatus`. Each new field, the precedence rule, the invariant that attributed counts don't exceed `numberUnavailable`, and clearing the fields with the gate off.
- `storeDaemonSetStatus`. With the gate off, the existing counts unchanged, and the new fields still set, the controller writes the new fields as unset instead of skipping the update.
- DaemonSet status strategy. With the gate off, new fields are dropped when the stored object doesn't have them, and accepted when it does.
- The metric records the bucket of each selected candidate.

##### Integration tests

New tests in `test/integration/daemonset`:

- A rollout with `maxUnavailable: 1` across several nodes, where one node has `MaintenanceInProgress=True`, updates that node last.
- The same for `DrainInProgress=True` and `Ready=False`.
- The same scenarios with `maxSurge > 0`.
- A node that already has an unavailable old Pod is replaced regardless of its bucket.
- Clearing a condition mid-rollout moves the node back to the higher bucket and updates status.
- A stale `MaintenanceInProgress=True` doesn't block the rollout from completing.
- With the gate off, status fields aren't set and ordering isn't applied.

##### e2e tests

A new test in `test/e2e/apps/daemon_set.go`, run in an alpha job with `NodeLifecycleConditions` and `DaemonSetNodeConditionAwareRollout` enabled. It needs at least three schedulable nodes.

The test sets one condition on one node, starts a rollout, and checks that node is updated last. It then removes the condition. It covers:

- `MaintenanceInProgress=True` with `maxUnavailable: 1`.
- `DrainInProgress=True` with `maxUnavailable: 1`.
- `MaintenanceInProgress=True` with `maxSurge: 1`.
- `DrainInProgress=True` with `maxSurge: 1`.

It also checks that the condition survives kubelet status updates and that `numberUnavailableOn*` reports the node while its Pod is unavailable.

`Ready=False` stays in integration tests. Making a real node NotReady means stopping its kubelet, which would disrupt the test cluster.

### Graduation Criteria

#### Alpha

- Feature implemented behind the `DaemonSetNodeConditionAwareRollout` feature gate, disabled by default.
- Rollout ordering, Node event handling, status fields, and metric implemented.
- Unit and integration tests with the gate on and off.
- e2e tests for maintenance and drain ordering, with `maxUnavailable` and `maxSurge`, running in an alpha job.

#### Beta

- e2e tests running in a periodic job on Testgrid, with no flakes for at least two weeks, and linked in this KEP.
- Feedback from rollout tooling on the status fields.
- A decision on whether additional conditions, such as `MaintenancePlanned`, should affect ordering.
- Upgrade, downgrade, and upgrade again tested, with results recorded in the PRR questionnaire.

#### GA

- All bug reports and feedback from beta resolved.
- No changes to the status field names or meanings since beta.

#### Deprecation

N/A.

### Upgrade / Downgrade Strategy

No change is required to keep the current behavior. Enabling the gate on kube-apiserver and kube-controller-manager enables ordering and status attribution. No objects need migration.

On downgrade or when the gate is disabled, ordering goes back to today's unspecified order. The controller stops setting the new status fields and clears them on its next status update.

### Version Skew Strategy

The feature lives in kube-controller-manager, with status field handling in kube-apiserver. Kubelets aren't involved.

- **Newer kube-controller-manager, older kube-apiserver.** The older API server doesn't know the new fields and drops them. Ordering still works because it depends only on the controller.
- **Older kube-controller-manager, newer kube-apiserver.** The controller doesn't set the new fields and doesn't order candidates. Behavior is the same as today.
- **Condition writers.** If no writer sets `DrainInProgress` or `MaintenanceInProgress`, the feature still orders by `Ready`.

## Production Readiness Review Questionnaire

### Feature Enablement and Rollback

###### How can this feature be enabled / disabled in a live cluster?

- [x] Feature gate (also fill in values in `kep.yaml`)
  - Feature gate name: `DaemonSetNodeConditionAwareRollout`
  - Components depending on the feature gate: kube-apiserver, kube-controller-manager

###### Does enabling the feature change any default behavior?

Yes. During a `RollingUpdate`, the DaemonSet controller targets nodes with a higher probability of success first. Which nodes run a DaemonSet Pod, how Pods are recovered, and `maxUnavailable` and `maxSurge` accounting don't change. DaemonSet status gets new fields.

###### Can the feature be disabled once it has been enabled (i.e. can we roll back the enablement)?

Yes. Disabling the gate returns rollout ordering to today's unspecified order. The controller clears the new status fields on its next status update.

###### What happens if we reenable the feature if it was previously rolled back?

Ordering and status attribution resume on the next sync of each DaemonSet. No state is carried across.

###### Are there any tests for feature enablement/disablement?

Yes. Unit tests cover the controller and status strategy with the gate on and off. An integration test toggles the gate and verifies that ordering and the status fields follow it.

### Rollout, Upgrade and Rollback Planning

###### How can a rollout or rollback fail? Can it impact already running workloads?

Enabling the gate can't affect running Pods outside a DaemonSet rollout. During a rollout it only changes which nodes are picked first, within the same budget. A bug in ordering could pick a different set of nodes than intended, but the budget still limits how many are disrupted at once.

###### What specific metrics should inform a rollback?

`daemonset_controller_rollout_candidates_total{bucket="lower"}` rising while higher-bucket candidates still exist. A rise in `daemonset_controller` sync errors or in DaemonSet rollouts that don't complete.

###### Were upgrade and rollback tested? Was the upgrade->downgrade->upgrade path tested?

Not yet. This will be tested before beta.

###### Is the rollout accompanied by any deprecations and/or removals of features, APIs, fields of API types, flags, etc.?

No.

### Monitoring Requirements

###### How can an operator determine if the feature is in use by workloads?

The new DaemonSet status fields are set, and `daemonset_controller_rollout_candidates_total` increases during rollouts.

###### How can someone using this feature know that it is working for their instance?

- [x] API .status
  - Other field: `numberUnavailableOnNotReadyNodes`, `numberUnavailableOnDrainingNodes`, `numberUnavailableOnMaintenanceNodes`
- [x] Other (treat as last resort)
  - Details: during a rollout, nodes with `Ready` not `True`, `DrainInProgress=True`, or `MaintenanceInProgress=True` are updated after the rest.

###### What are the reasonable SLOs (Service Level Objectives) for the enhancement?

The existing DaemonSet controller SLOs apply. This feature shouldn't change DaemonSet sync latency or error rate.

###### What are the SLIs (Service Level Indicators) an operator can use to determine the health of the service?

- [x] Metrics
  - Metric name: `daemonset_controller_rollout_candidates_total`
  - Components exposing the metric: kube-controller-manager
- [x] Metrics
  - Metric name: `workqueue_depth` and `workqueue_adds_total` for the `daemonset` and `daemonset-node-updates` queues
  - Components exposing the metric: kube-controller-manager

###### Are there any missing metrics that would be useful to have to improve observability of this feature?

Not for alpha.

### Dependencies

###### Does this feature depend on any specific services running in the cluster?

- Node lifecycle conditions from [KEP-5683]
  - Usage description: `DrainInProgress` and `MaintenanceInProgress` are read for ordering and status.
  - Impact of its outage on the feature: if no writer sets them, the feature orders by `Ready` only.
  - Impact of its degraded performance or high-error rates on the feature: stale conditions change ordering, never eligibility.

### Scalability

###### Will enabling / using this feature result in any new API calls?

No new call types. The controller already reads Nodes from its informer. Condition flips on `Ready`, `DrainInProgress`, or `MaintenanceInProgress` cause extra DaemonSet syncs, which can result in status updates. Heartbeats don't.

###### Will enabling / using this feature result in introducing new API types?

No.

###### Will enabling / using this feature result in any new calls to the cloud provider?

No.

###### Will enabling / using this feature result in increasing size or count of the existing API objects?

Yes. `DaemonSetStatus` gets three optional `int32` fields, adding a few tens of bytes per DaemonSet.

###### Will enabling / using this feature result in increasing time taken by any operations covered by existing SLIs/SLOs?

No. Classification is a few condition lookups per node, and sorting adds `O(n log n)` over candidates in a sync that's already `O(n)` over nodes.

###### Will enabling / using this feature result in non-negligible increase of resource usage (CPU, RAM, disk, IO, ...) in any components?

No. The extra syncs are bounded by condition flips, which are rare compared to heartbeats.

###### Can enabling / using this feature result in resource exhaustion of some node resources (PIDs, sockets, inodes, etc.)?

No. The feature runs in the control plane and doesn't change what runs on nodes.

### Troubleshooting

###### How does this feature react if the API server and/or etcd is unavailable?

The same as the DaemonSet controller today. No rollout progress or status updates happen until the API server is back.

###### What are other known failure modes?

- **A stale `DrainInProgress=True` or `MaintenanceInProgress=True`.**
  - Detection: the node shows up in `numberUnavailableOn*` long after the activity ended.
  - Mitigations: clear the condition. The rollout still completes, so no action is required for the rollout itself.
  - Diagnostics: check the Node conditions and their `lastTransitionTime`.
  - Testing: covered by an integration test.

###### What steps should be taken if SLOs are not being met to determine the problem?

Check the `daemonset` and `daemonset-node-updates` queue metrics for unexpected growth. Check whether a node's condition is flapping. Disable the gate on kube-controller-manager to restore today's ordering.

## Implementation History

- 2026-07-20: Enhancement issue [#6250] opened.
- 2026-10-19: Provisional KEP drafted.

## Drawbacks

- Ordering helps only when there's budget to spend. It doesn't help when existing unavailability already uses all of `maxUnavailable`.
- The controller trusts Node conditions it doesn't own. An incorrect condition changes the order of a rollout until someone clears it.
- `DaemonSetStatus` grows by three fields.

## Alternatives

**Exempt lifecycle-affected nodes from `maxUnavailable`.** This changes the meaning of a public field. `maxUnavailable` would no longer bound how many DaemonSet Pods are unavailable during a rollout, so a user's availability limit wouldn't hold during maintenance. A separate budget would let higher-probability nodes progress when lifecycle activity uses the whole budget. That benefit doesn't justify changing what `maxUnavailable` means for every DaemonSet.

**Skip lifecycle-affected nodes until the condition clears.** This turns Node conditions into rules about which nodes get updated, instead of signals for ordering. A stale condition would block a rollout forever. A node under maintenance that doesn't disrupt workloads would never be updated while the condition is set. Ordering avoids both without adding a new way for a rollout to block.

**Read `GracefulNodeShutdownInProgress`.** The kubelet reports `Ready=False` during Graceful Node Shutdown, so these nodes already land in the lower bucket. DaemonSet behavior during Graceful Node Shutdown is owned by [KEP-6249].

**Include `MaintenancePlanned`.** The condition describes future intent and may be set well before maintenance starts. Ordering on it would move a node that's currently fine for an unbounded period, depending on how early each writer sets it. We'll revisit it at beta.

**A reason-keyed list in status.** A list of `{reason, count}` entries would avoid adding a field per reason. It's harder for clients to read and inconsistent with the flat counters in `DaemonSetStatus` today. With a small, known set of reasons, explicit fields are simpler.

## Infrastructure Needed (Optional)

None.

[#6250]: https://github.com/kubernetes/enhancements/issues/6250
[#6430]: https://github.com/kubernetes/enhancements/issues/6430
[KEP-5683]: https://github.com/kubernetes/enhancements/tree/master/keps/sig-node/5683-lifecycle-conditions
[KEP-6249]: https://github.com/kubernetes/enhancements/issues/6249
[k/k#138240]: https://github.com/kubernetes/kubernetes/issues/138240
[k/k#139226]: https://github.com/kubernetes/kubernetes/issues/139226
