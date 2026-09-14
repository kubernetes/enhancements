# KEP-5683: Node Lifecycle Conditions

<!-- toc -->
- [Release Signoff Checklist](#release-signoff-checklist)
- [Summary](#summary)
- [Motivation](#motivation)
  - [Goals](#goals)
  - [Non-Goals](#non-goals)
- [Proposal](#proposal)
  - [User Stories](#user-stories)
    - [Story 1: DaemonSet Controller and GNS Kubelet Disagree](#story-1-daemonset-controller-and-gns-kubelet-disagree)
    - [Story 2: Jobs Stuck When Nodes Become Unreachable](#story-2-jobs-stuck-when-nodes-become-unreachable)
    - [Story 3: Broken Nodes Can Consume Rollout Budget](#story-3-broken-nodes-can-consume-rollout-budget)
    - [Story 4: DaemonSet Rollouts Aren't Reporting the Node's State](#story-4-daemonset-rollouts-arent-reporting-the-nodes-state)
    - [Story 5: Taints are Insufficient for Signaling Node Drain](#story-5-taints-are-insufficient-for-signaling-node-drain)
    - [Story 6: Reactive vs Proactive Drain](#story-6-reactive-vs-proactive-drain)
    - [Story 7: Coordinating between Controllers and Admins](#story-7-coordinating-between-controllers-and-admins)
    - [Story 8: Coordinating between the Autoscaler and Scheduler during Scale-in](#story-8-coordinating-between-the-autoscaler-and-scheduler-during-scale-in)
  - [Risks and Mitigations](#risks-and-mitigations)
- [Design Details](#design-details)
  - [GracefulNodeShutdownInProgress Condition](#gracefulnodeshutdowninprogress-condition)
  - [Lifecycle Conditions](#lifecycle-conditions)
  - [Writer Ownership](#writer-ownership)
  - [Kubectl Drain Condition Writer](#kubectl-drain-condition-writer)
    - [Permissions](#permissions)
    - [Signal Handling](#signal-handling)
  - [Feature Gate](#feature-gate)
  - [Future Extension Points](#future-extension-points)
  - [Test Plan](#test-plan)
      - [Prerequisite testing updates](#prerequisite-testing-updates)
      - [Unit tests](#unit-tests)
      - [Integration tests](#integration-tests)
      - [e2e tests](#e2e-tests)
  - [Graduation Criteria](#graduation-criteria)
    - [Alpha - Introduce Well-Known Conditions](#alpha---introduce-well-known-conditions)
    - [Alpha2 - Publish Drain Conditions from kubectl](#alpha2---publish-drain-conditions-from-kubectl)
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

- [x] (R) Enhancement issue in release milestone, which links to KEP dir in
  [kubernetes/enhancements] (not the initial KEP PR)
- [ ] (R) KEP approvers have approved the KEP status as `implementable`
- [x] (R) Design details are appropriately documented
- [ ] (R) Test plan is in place, giving consideration to SIG Architecture and
  SIG Testing input (including test refactors)
  - [ ] e2e Tests for all Beta API Operations (endpoints)
  - [ ] (R) Ensure GA e2e tests meet requirements for
    [Conformance Tests]
  - [ ] (R) Minimum Two Week Window for GA e2e tests to prove flake free
- [ ] (R) Graduation criteria is in place
  - [ ] (R) [all GA Endpoints] must be hit by [Conformance Tests] within one
    minor version of promotion to GA
- [ ] (R) Production readiness review completed
- [ ] (R) Production readiness review approved
- [ ] "Implementation History" section is up-to-date for milestone
- [ ] User-facing documentation has been created in [kubernetes/website], for
  publication to [kubernetes.io]
- [ ] Supporting documentation, for example additional design documents, links
  to mailing list discussions/SIG meetings, relevant PRs/issues, release notes

[all GA Endpoints]: https://github.com/kubernetes/community/pull/1806
[Conformance Tests]: https://github.com/kubernetes/community/blob/master/contributors/devel/sig-architecture/conformance-tests.md
[kubernetes.io]: https://kubernetes.io/
[kubernetes/enhancements]: https://git.k8s.io/enhancements
[kubernetes/kubernetes]: https://git.k8s.io/kubernetes
[kubernetes/website]: https://git.k8s.io/website

## Summary

Kubernetes has several independent components that need to understand a Node's
lifecycle state to contextualize workload availability. Today, that state is
inferred from a mix of Node readiness, taints, Pod state, controller status,
labels, annotations, and provider-specific APIs. These signals are useful for
their individual purposes, but they do not provide a shared Kubernetes-owned
place for an admin to publish lifecycle state on the Node.

This KEP introduces well-known "lifecycle conditions" to Nodes. These
conditions provide an observable lifecycle signal that core controllers and
ecosystem tooling can consume without each building its own interpretation of
Node, Pod, and controller state. The pattern follows
[KEP-5394](../5394-psi-node-conditions/README.md), which
introduces well-known Node conditions as an observable signal and then updates
core behavior to react to those condition values.

The first version is intentionally narrow. It does not define a general node
maintenance protocol. It establishes the pattern: a well-known condition
consumable on the Node, providing a foundation for future work. The second
alpha introduces `kubectl drain` the first in-tree writer of `DrainInProgress`
and `Drained`, establishing how a lifecycle tool publishes these observations
without changing drain behavior.

## Motivation

Node lifecycle state is a cross-cutting concern. Kubelet, node lifecycle
controller, workload controllers, the scheduler, autoscalers, operators, admins,
and external maintenance systems can all observe pieces of the Node's lifecycle,
but there is no well-known place where an admin can publish "this Node is in x
lifecycle state", which would explain certain Pod behaviors.

The absence of a shared lifecycle signal leads to duplicated and fragile
interpretation. One controller may infer lifecycle state from Node readiness.
Others may look at taints. Some controllers watch Pods, map them back to Nodes,
inspect Node conditions, and apply its own ignore policy. There are operators
that publish annotations or labels that only its own components understand.
These approaches work locally, but they do not create reusable lifecycle data
that can be used cooperatively with core Kubernetes controllers and they do not
provide a stable endpoint for admins or users.

Kubernetes already uses well-known Node conditions for shared Node state such as
`Ready`, `MemoryPressure`, `DiskPressure`, `PIDPressure`, and
`NetworkUnavailable`. This KEP extends that model by proposing new Node
conditions for Node Lifecycle. First, publish well-known Node lifecycle signals,
then follow up KEPs can make core controllers use those signals for better
status reporting.

### Goals

- Add a well-known `GracefulNodeShutdownInProgress` Node condition type.
- Add well-known lifecycle Node condition types for drain and maintenance:
  `DrainInProgress`, `Drained`, `MaintenancePlanned`, and `MaintenanceInProgress`.
- Define condition semantics that let Kubernetes and admins communicate
  lifecycle state relevant to workload availability.
- Establish a Kubernetes-owned lifecycle signal that future core controllers and
  ecosystem tools can consume.
- Provide an asynchronous building block that can be extended without changing
  the initial condition.
- alpha2: Publish `kubectl drain` progress through `DrainInProgress` and `Drained`.

### Non-Goals

- Define exclusive condition ownership, locking, or handoff.
- Define a contractual relationship between controllers reading and writing the
  condition.
- Define kubelet recovery behavior for lost Graceful Node Shutdown state.
- Define all the well-known conditions for a Node's lifecycle.
- Define a synchronous lifecycle state model.
- Define lifecycle policy or decide what every lifecycle condition means.
- Design in this KEP how every workload controller consumes these conditions.

## Proposal

Add Kubernetes-owned Node conditions for well-known lifecycle states.
`GracefulNodeShutdownInProgress` reports that Graceful Node Shutdown is
determined to be in progress on the Node. `DrainInProgress` reports that the
Node is actively being drained. `Drained` reports that the Node has reached the
drain criteria selected by the actor managing the lifecycle.
`MaintenancePlanned` reports that the Node is expected to undergo maintenance.
`MaintenanceInProgress` reports that the Node is actively undergoing
maintenance.

This makes lifecycle state observable on the Node and gives core controllers a
common signal to consume in the future. The condition type and status carry the
observed lifecycle state.

For this KEP, the new Node conditions are admin managed. An admin, or an
admin-authorized maintenance controller, sets the condition status to `True`
when appropriate. Clearing the lifecycle state is done by setting the condition
status to `False` or removing the condition.

In Alpha2, `kubectl drain` becomes the first in-tree lifecycle condition
writer. With condition reporting enabled, it sets `DrainInProgress=True` when
it begins processing a Node and sets `Drained=True` when the drain criteria
selected by that invocation have been met. Reporting is informational and does
not change the existing drain operation.

### User Stories

The conditions in this proposal are intended to provide a shared signal for the use cases
described in
[[public] Node Lifecycle Use Cases](https://docs.google.com/document/d/1EINvuVzEoRra0CKH6uQnOcQJVbd7ZnxT1bcyySN-r7c/edit?usp=sharing)
by publishing important lifecycle state on the Node.

The conditions do not, by themselves, implement every controller behavior needed
to resolve these use cases. All of these use cases should be addressed in their
own issues as follow-up work to this KEP, so the consuming controller behavior
can be designed, reviewed, and owned independently.

#### Story 1: DaemonSet Controller and GNS Kubelet Disagree

Tracking Issues:
- [kubernetes/kubernetes#122912](https://github.com/kubernetes/kubernetes/issues/122912)
- [kubernetes/kubernetes#137895](https://github.com/kubernetes/kubernetes/issues/137895)

As a DaemonSet controller or rollout verifier,
`GracefulNodeShutdownInProgress=True` and `DrainInProgress=True` tell
the DaemonSet controller that Graceful Node Shutdown and drain may explain
missing DaemonSet Pods.

Before rescheduling a missing Pod, the DaemonSet controller would check the
"well-known" conditions on the node for additional context. If the Node has
`GracefulNodeShutdownInProgress=True` and `DrainInProgress=True`
conditions, do not reschedule the Daemonset Pod.

#### Story 2: Jobs Stuck When Nodes Become Unreachable

Tracking Issues:
- [kubernetes/kubernetes#134038](https://github.com/kubernetes/kubernetes/issues/134038)

As a Job or queueing controller,
`MaintenanceInProgress=True` gives Job and queueing controllers a
Node-level signal that Pods on the Node may need special accounting when an
admin or maintenance controller has identified the Node is being lifecycled.

The Job controller configured with `podReplacementPolicy: Failed` should check
the Node for additional context when account for Pods. When the controller sees
a Node condition `MaintenanceInProgress=True`, the controller should
not wait for the Job's Pods to reach Failed/Succeeded phase before moving on.

#### Story 3: Broken Nodes Can Consume Rollout Budget

Tracking Issues:
- [kubernetes/kubernetes#138240](https://github.com/kubernetes/kubernetes/issues/138240)

As a DaemonSet rollout controller,
`MaintenanceInProgress=True` explains to the DaemonSet controller that
some Nodes are not ideal for placement during rollout.

When the DaemonSet controller rolls out new Pods, it should check the Node's
conditions for additional lifecycle context. When a Node has
`MaintenanceInProgress=True`, the controller should deprioritize this
node for scheduling.

#### Story 4: DaemonSet Rollouts Aren't Reporting the Node's State

Tracking Issues:
- [kubernetes/kubernetes#139226](https://github.com/kubernetes/kubernetes/issues/139226)

As a rollout verifier,
DaemonSet status can surface `GracefulNodeShutdownInProgress` and
`MaintenanceInProgress` attribution under `status.unavailable`, so
rollout verifiers can distinguish rollout failures from lifecycle-related
unavailability.

The DaemonSet controller is missing important context about the node's lifecycle
state that would be useful to share with the user. When the DaemonSet controller
schedules or has scheduled Pods to a node with conditions `GracefulNodeShutdownInProgress`
or `MaintenanceInProgress`, the total number of Daemonset Pods on those
nodes can be shared under `status.unavailable`.

#### Story 5: Taints are Insufficient for Signaling Node Drain

Tracking Issues:
- [kubernetes/enhancements#6251](https://github.com/kubernetes/enhancements/issues/6251)
- [kubernetes/kubernetes#25625](https://github.com/kubernetes/kubernetes/issues/25625)
- [kubernetes-sigs/cluster-api#3365](https://github.com/kubernetes-sigs/cluster-api/issues/3365)
- [kubernetes/autoscaler#8157](https://github.com/kubernetes/autoscaler/issues/8157)
- [kubernetes/kubernetes#138719](https://github.com/kubernetes/kubernetes/issues/138719)

As a user or controller observing drain progress,
`DrainInProgress=True` or `Drained=True` gives users a clear
Node status signal that is separate from scheduling policy and can attest to
lifecycle progress.

There are many solutions that can be implemented for this one, so will focus on
`kubectl drain` for simplicity. When running `kubectl drain`, the Node will receive
condition `DrainInProgress=True`. When `kubectl drain` completes,
`Drained=True` can be set. It will be up to the admin to clear the
state.

#### Story 6: Reactive vs Proactive Drain

Tracking Issues:
- [rook/rook#4290](https://github.com/rook/rook/issues/4290)
- [rook/rook#16086](https://github.com/rook/rook/issues/16086)
- [rook/rook#16976](https://github.com/rook/rook/issues/16976)
- [Red Hat Bugzilla 1861104](https://bugzilla.redhat.com/show_bug.cgi?id=1861104)

As a storage operator,
`MaintenancePlanned=True` gives storage operators a proactive signal before
drain begins, while `DrainInProgress=True` and `Drained=True` give them progress
signals during the disruption.

The admin should set `MaintenancePlanned=True` on the Node before
performing maintenance. Then, rook can adjust the Ceph PDBs to handle disruption
on that node.

#### Story 7: Coordinating between Controllers and Admins

Tracking Issues:
- [kubernetes/node-problem-detector#1176](https://github.com/kubernetes/node-problem-detector/issues/1176)
- [kubernetes/node-problem-detector#457](https://github.com/kubernetes/node-problem-detector/issues/457)
- [kubernetes/enhancements#1403](https://github.com/kubernetes/enhancements/issues/1403)
- [openshift/enhancements#141](https://github.com/openshift/enhancements/pull/141)

As Node Problem Detector or other automation,
`MaintenanceInProgress=True` lets Node Problem Detector and other
automation distinguish a Node actively controlled by an admin from an unexpected
failure.

The Node Problem Detector should check the node conditions for additional
context. When NPD sees a Node with `MaintenanceInProgress=True`, the
controller can skip publish conditions on the node.

#### Story 8: Coordinating between the Autoscaler and Scheduler during Scale-in

Tracking Issues:
- [kubernetes/kubernetes#138718](https://github.com/kubernetes/kubernetes/issues/138718)

As an autoscaler or scheduler,
`DrainInProgress=True`, `Drained=True`, `MaintenanceInProgress=True`, or
`MaintenancePlanned=True` provides a Node-local signal that higher-level
scale-in and workload controllers can later consume when deciding where
disruption should be concentrated.

Autoscalers should look at node conditions to gather node lifecycle context.
When the autoscaler sees a node with `MaintenanceInProgress=True` or
`MaintenancePlanned=True`, it should prioritize that node during scale-in.

### Risks and Mitigations

- The Lifecycle Conditions could become too broad if users publish unrelated
  lifecycle signals. This KEP mitigates that by defining the initial condition
  types as `DrainInProgress`, `Drained`, `MaintenancePlanned`, and
  `MaintenanceInProgress`. Core
  controllers must not assign behavioral meaning to additional lifecycle
  conditions without a follow-up KEP.
- Admin-defined values can make state usage inconsistent across clusters. This
  is intentional early on: the purpose is to establish a stable lifecycle
  condition that can carry an admin-provided state, be consumed by core
  Kubernetes controllers, and later be extended by
  [Specialized Lifecycle Management](https://github.com/kubernetes/enhancements/issues/5683).
- A client-side writer can disappear while `DrainInProgress=True`. Kubectl
  attempts a terminal update for handled interrupts, but cannot recover from
  `SIGKILL`, process failure, host failure, or loss of API connectivity.
  Administrators remain responsible for correcting stale state.
- Publishing drain conditions requires `patch` permission on `nodes/status`.
  This KEP does not change default RBAC roles, and condition reporting remains
  best-effort if the user lacks that permission.

## Design Details

### GracefulNodeShutdownInProgress Condition

Add a new `NodeConditionType` constant:

```go
const (
        // GracefulNodeShutdownInProgress reports whether Graceful Node Shutdown
        // is determined to be in progress on this Node.
        GracefulNodeShutdownInProgress NodeConditionType = "GracefulNodeShutdownInProgress"
)
```

The condition has the following semantics:

- `status=True`: Graceful Node Shutdown is determined to be in progress on this
  Node.
- `status=False`: Graceful Node Shutdown is not currently determined to be in
  progress on this Node.
- `status=Unknown`: Kubernetes cannot determine whether Graceful Node Shutdown
  is in progress.

The condition `type` and boolean `status` are the important observed values for
this condition. The `reason` field explains why the condition has the reported
status.

This KEP only defines the condition surface. Integrating this condition
to solve the Graceful Node Shutdown bugs shared in the use cases will be follow
up work.

### Lifecycle Conditions

Add new `NodeConditionType` constants:

```go
const (
        // DrainInProgress reports that this Node is actively being drained.
        DrainInProgress NodeConditionType = "DrainInProgress"

        // Drained reports that this Node has reached the drain criteria
        // selected by the actor managing the lifecycle.
        Drained NodeConditionType = "Drained"

        // MaintenancePlanned reports that this Node is expected to undergo
        // maintenance.
        MaintenancePlanned NodeConditionType = "MaintenancePlanned"

        // MaintenanceInProgress reports that this Node is actively undergoing
        // maintenance.
        MaintenanceInProgress NodeConditionType = "MaintenanceInProgress"
)
```

The conditions have the following semantics:

- `status=True`: the lifecycle state described by the condition type is
  currently observed on this Node.
- `status=False`: the lifecycle state described by the condition type is not
  currently observed on this Node.
- `status=Unknown`: Kubernetes cannot determine whether the lifecycle state
  described by the condition type is active.

The condition `reason` identifies why the condition has the reported status. It
is a machine-readable cause category. Example reasons include:

- `AdminRequested`: an admin or admin-authorized controller explicitly set the
  condition.
- `DrainCompleted`: the actor managing drain observed that its drain criteria
  have been met.
- `MaintenanceWindow`: the condition was set because the Node is in a planned
  maintenance window.
- `NodeShutdown`: the condition was set because Node shutdown was detected.

The transition away from any of these states is controlled by the admin or
admin-authorized maintenance controller that owns the condition. This KEP does
not define where the Node goes after `Drained=True`; it may be
terminated, enter maintenance, or return to service depending on the managing
system. The writer sets the condition status to `False` or removes the
condition when it considers the lifecycle state no longer active.

Future KEPs may standardize additional lifecycle condition types. Those future
condition types must define their own writer ownership and transition semantics
before core controllers assign behavioral meaning to them.

### Writer Ownership

For this KEP, admins write these conditions. An admin may write the conditions
directly or delegate that permission to a maintenance controller. For example,
the writer sets:

- `type=DrainInProgress`
- `status=True`
- `reason=AdminRequested`
- `message=<optional human-readable details>`

For lifecycle conditions, the condition `type` identifies the observed lifecycle
state and `status` reports whether that state is active. The `reason` field
identifies the cause category for the current status. Writers should use stable,
CamelCase.

Any authorized actor can write the Lifecycle Conditions. This avoids introducing
lifecycle ownership or coordination semantics for now.

### Kubectl Drain Condition Writer

Alpha2 adds an opt-in `--report-node-conditions` flag to `kubectl drain`.
When enabled, kubectl publishes drain observations through the Node `status`
subresource.

`Drained=True` means that the criteria selected by that invocation were met. It
does not mean that the Node has zero Pods, that another drain implementation
would select the same Pods, or that Pods cannot be created afterward.

Kubectl uses these reasons:

| Reason | Definition |
|---|---|
| `KubectlDrainStarted` | Kubectl started processing the Pods selected for drain on the Node. |
| `KubectlDrainCompleted` | Kubectl observed that its selected drain criteria were met. |
| `KubectlDrainFailed` | Kubectl could not meet its selected drain criteria because the operation failed or timed out. |
| `KubectlDrainInterrupted` | Kubectl handled an interrupt before its selected drain criteria were met. |

The two conditions are updated together:

| Event | `DrainInProgress` | `Drained` | Reason |
|---|---|---|---|
| Kubectl starts processing the Node | `True` | `False` | `KubectlDrainStarted` |
| Selected criteria are met | `False` | `True` | `KubectlDrainCompleted` |
| Drain fails or times out | `False` | `False` | `KubectlDrainFailed` |
| Kubectl handles an interrupt | `False` | `False` | `KubectlDrainInterrupted` |

For a multi-Node invocation, kubectl publishes `DrainInProgress=True`
immediately before processing each individual Node. It does not publish that
condition for every selected Node when the initial cordon operation begins.

Starting a new drain resets `Drained=False` before processing the Node.
Concurrent drain actors are not coordinated. Any authorized actor can patch
the lifecycle conditions, and the latest successful update determines the
observed value.

Kubectl uses a strategic merge patch containing only `DrainInProgress` and
`Drained`. `NodeCondition` is a merge list keyed by condition type, so unrelated
Node conditions and status fields are preserved. A reporting failure is
printed as a warning and does not change the drain command's result.

#### Permissions

Condition reporting adds one permission to `kubectl drain`: `patch` on the
core `nodes/status` subresource. This KEP does not change default RBAC roles.
Administrators who want condition reporting must grant that permission to
their drain users.

The `k8s.io/kubectl/pkg/drain` package is used by projects other than the
kubectl command. Condition reporting remains disabled by default in the
library, so updating the dependency does not change downstream behavior or
permissions.

#### Signal Handling

Today, `kubectl drain` does not install a signal handler for the drain
operation. The operating system terminates the process immediately after
`SIGINT`, `SIGTERM`, or `SIGHUP`, without a final API request.

With condition reporting enabled, the first of those signals cancels the
active drain and stops processing additional Nodes. Kubectl prints that it is
updating the Node conditions, then makes one best-effort strategic merge patch
for the current Node using a new five-second context. After the patch completes
or times out, kubectl delivers the original signal and preserves signal-based
termination. A second signal terminates immediately. Unhandled process or host
failure can still leave `DrainInProgress=True`.

### Feature Gate

Add the `NodeLifecycleConditions` feature gate.

Components:

- `kube-controller-manager`
- `kube-apiserver`

When the feature gate is disabled:

- No changes

### Future Extension Points

1) The use cases and the bugs associated with them should be addressed separately.
Those designs would leverage this KEP to fill the gap around node state that
control plane components don't currently have.

2) The remaining use cases around condition ownership, locking, and coordination
will be addressed separately. The current thinking is this requires building an API that
allows user to express a synchronous state model. [This KEP](https://github.com/kubernetes/enhancements/pull/5769)
outlines an approach using patterns used in DRA, which would leverage these conditions.

### Test Plan

- [x] I/we understand the owners of the involved components may require updates
  to existing tests to make this code solid enough prior to committing the
  changes necessary to implement this enhancement.

##### Prerequisite testing updates

None.

##### Unit tests

- Unit tests for reading `GracefulNodeShutdownInProgress=True` from Nodes.
- Unit tests for reading lifecycle condition type/status values from Nodes.
- Unit tests in `k8s.io/kubectl/pkg/cmd/drain` and
  `k8s.io/kubectl/pkg/drain` cover:
  - condition reporting disabled
  - successful single-Node and multi-Node drains
  - start, completion, failure, and interruption patch payloads
  - client and server dry-run
  - forbidden and transient status updates
  - repeated drains
  - existing library consumers with no condition reporter

##### Integration tests

- Verify that the Node has `GracefulNodeShutdownInProgress`,
  `DrainInProgress`, `Drained`, `MaintenancePlanned`, and
  `MaintenanceInProgress` conditions.
- Verify that kubectl patches only the drain conditions through
  `nodes/status`, preserves unrelated status, and treats authorization failures
  as non-fatal to the drain operation.

##### e2e tests

- Drain a Node with condition reporting enabled and verify the start and
  completion conditions.
- Interrupt an active drain and verify that both conditions become `False`
  with reason `KubectlDrainInterrupted`.

### Graduation Criteria

#### Alpha - Introduce Well-Known Conditions

- `GracefulNodeShutdownInProgress` condition type is added.
- `DrainInProgress` condition type is added.
- `Drained` condition type is added.
- `MaintenancePlanned` condition type is added.
- `MaintenanceInProgress` condition type is added.
- Initial reason cause categories are documented.
- `NodeLifecycleConditions` feature gate is added.
- Unit tests cover condition reading and feature-gate behavior.

#### Alpha2 - Publish Drain Conditions from kubectl

- Add the opt-in `--report-node-conditions` flag to `kubectl drain`.
- Publish the start, completion, failure, and interruption transitions defined
  in this KEP.
- Keep condition reporting disabled by default for drain library consumers.
- Document the additional `nodes/status` permission.
- Add unit, integration, and initial e2e coverage.

Changes that consume lifecycle conditions in core controllers remain separate
enhancements so their behavioral and production-readiness implications can be
reviewed independently.

#### Beta

- Gather feedback from rollout tooling and large-cluster operators.
- Enable kubectl condition reporting by default, with
  `--report-node-conditions=false` as an opt-out.
- Keep condition reporting opt-in for drain library consumers.
- Resolve known issues with stale observations, concurrent writers, and
  selected drain criteria.
- Decide whether additional lifecycle condition types should be standardized.
- Feature gate defaults to enabled.

#### GA

- All beta feedback is resolved.
- No open questions remain about field semantics.
- Feature gate is removed.

#### Deprecation

N/A. This KEP does not deprecate an existing feature, API, field, or flag.

### Upgrade / Downgrade Strategy

This KEP is additive. Anything using these condition types can retain their
current meaning.

On upgrade, clusters that enable the feature gate may see new
`GracefulNodeShutdownInProgress`, `DrainInProgress`, `Drained`,
`MaintenancePlanned`, and `MaintenanceInProgress` conditions on Nodes.

During Alpha2, upgrading kubectl adds the opt-in
`--report-node-conditions` flag without changing existing invocations. At
beta, reporting becomes the default and users can retain the previous behavior
with `--report-node-conditions=false`.

Downgrading to a kubectl version without the feature stops new drain condition
updates but does not otherwise affect drain. Existing conditions remain on the
Node until an authorized actor updates or removes them.

### Version Skew Strategy

A kubectl version that supports condition reporting can publish the conditions
to any supported API server when the user has `patch` permission on
`nodes/status`.

Older kubectl versions and other drain implementations may not publish the
conditions. Consumers must interpret an absent condition as "no observation
was reported", not as proof that the Node is not being drained.

Future components that consume lifecycle conditions must define their own
version skew behavior in their enhancements.

## Production Readiness Review Questionnaire

### Feature Enablement and Rollback

###### How can this feature be enabled / disabled in a live cluster?

- [x] Feature gate
  - Feature gate name: `NodeLifecycleConditions`
  - Components depending on the feature gate:
    - `kube-apiserver`
    - `kube-controller-manager`
- [x] Other
  - During Alpha2, users enable kubectl reporting with
    `kubectl drain --report-node-conditions`.
  - Starting at beta, users can disable kubectl reporting with
    `--report-node-conditions=false`.

###### Does enabling the feature change any default behavior?

No scheduling or rollout behavior changes. During Alpha2, kubectl reporting is
opt-in. Enabling it adds best-effort Node status writes.

###### Can the feature be disabled once it has been enabled (i.e. can we roll back the enablement)?

Yes. Disable the feature gate and stop passing
`--report-node-conditions`. Existing conditions remain until an authorized
actor clears or replaces them.

###### What happens if we reenable the feature if it was previously rolled back?

The next condition update or drain invocation publishes the current
observation using the normal transition rules.

###### Are there any tests for feature enablement/disablement?

Unit tests cover feature-gate enablement and both enabled and disabled kubectl
reporting paths.

### Rollout, Upgrade and Rollback Planning

###### How can a rollout or rollback fail? Can it impact already running workloads?

The feature is informational and does not directly affect running workloads.
A status update can be forbidden, rejected by admission, conflict, or time out.
Kubectl warns and preserves the drain operation's original result. Rollout or
rollback failures can result in missing or stale conditions.

###### What specific metrics should inform a rollback?

Unexpected increases in Node status update failures, API server write latency,
or stale lifecycle condition reports should inform rollback.

###### Were upgrade and rollback tested? Was the upgrade->downgrade->upgrade path tested?

TBD before beta.

###### Is the rollout accompanied by any deprecations and/or removals of features, APIs, fields of API types, flags, etc.?

No.

### Monitoring Requirements

###### How can an operator determine if the feature is in use by workloads?

Operators can inspect Nodes for the `GracefulNodeShutdownInProgress`,
`DrainInProgress`, `Drained`, `MaintenancePlanned`, and
`MaintenanceInProgress` conditions. A `KubectlDrain*` reason identifies a
condition published by kubectl.

###### How can someone using this feature know that it is working for their instance?

- [x] API .status
  - Condition name: `GracefulNodeShutdownInProgress`
  - Condition name: `DrainInProgress`
  - Condition name: `Drained`
  - Condition name: `MaintenancePlanned`
  - Condition name: `MaintenanceInProgress`

###### What are the reasonable SLOs (Service Level Objectives) for the enhancement?

Share observable state on the Node so Kubernetes controllers can use that state
to clarify Pod behavior.

###### What are the SLIs (Service Level Indicators) an operator can use to determine the health of the service?

- [ ] Metrics
  - Metric name: TBD

###### Are there any missing metrics that would be useful to have to improve observability of this feature?

No.

### Dependencies

###### Does this feature depend on any specific services running in the cluster?

No external services are required.

### Scalability

###### Will enabling / using this feature result in any new API calls?

Yes. Admins or admin-authorized maintenance controllers may issue Node status
updates when these conditions change. With kubectl reporting enabled, each
Node normally receives one patch when drain processing starts and one terminal
patch on completion, failure, or a handled interrupt. Retries can add a bounded
number of requests.

###### Will enabling / using this feature result in introducing new API types?

No.

###### Will enabling / using this feature result in any new calls to the cloud provider?

No.

###### Will enabling / using this feature result in increasing size or count of the existing API objects?

Yes. Nodes may include five additional conditions.

###### Will enabling / using this feature result in increasing time taken by any operations covered by existing SLIs/SLOs?

No expected impact.

###### Will enabling / using this feature result in non-negligible increase of resource usage (CPU, RAM, disk, IO, ...) in any components?

No. The conditions are stored on existing Node status.

###### Can enabling / using this feature result in resource exhaustion of some node resources (PIDs, sockets, inodes, etc.)?

No.

### Troubleshooting

###### How does this feature react if the API server and/or etcd is unavailable?

Controllers cannot publish updated Node status until API server and
etcd availability returns.

###### What are other known failure modes?

- Stale `DrainInProgress` condition:
  - Detection: inspect the condition timestamps, active administration, and
    Pods remaining on the Node.
  - Mitigation: an authorized administrator corrects or clears the condition.
  - Testing: handled interrupts are tested; uncatchable process or host failure
    cannot be cleaned up automatically.
- The kubectl user lacks `nodes/status` permission:
  - Detection: kubectl prints a forbidden warning and audit logs record the
    denied request.
  - Mitigation: grant the permission or run without condition reporting.
- Concurrent writers publish conflicting observations:
  - Detection: inspect condition timestamps, reasons, messages, and audit logs.
  - Mitigation: stop concurrent drain actors and have an administrator publish
    the observed state.

## Implementation History

- 2026-06-04: Initial provisional KEP draft.
- 2026-08-10: Added the Alpha2 kubectl drain condition writer design.

## Drawbacks

This adds five Node conditions, one for Graceful Node Shutdown and four for
broader lifecycle state. The benefit is a stable lifecycle attribution surface
that can be consumed by future controller changes and extended later.

## Alternatives

1. Do nothing and require existing tooling to continue working around the gaps.
   This preserves the status quo but leaves the existing use cases unsolved in
   the core.
2. Use fewer condition types and encode lifecycle states in `reason`. In this
   approach, `Drain=True` could use reasons such as `Draining` and `Drained`,
   while `Maintenance=True` could use reasons such as `MaintenancePlanned` and
   `MaintenanceInProgress`. This keeps the condition type list shorter, but it
   makes `reason` carry the lifecycle state machine. Kubernetes API conventions
   describe conditions as observations rather than state machines, and define
   `reason` as a machine-readable category of cause for the current status. This
   KEP therefore uses condition `type` and `status` for the lifecycle signal and
   reserves `reason` for cause categories.
3. Add a dedicated lifecycle status field to Node, such as
   `Node.status.lifecycle`. This could model lifecycle state more directly than
   conditions and could include structured subfields for state, owner, and
   transition metadata. However, it would introduce a new Node status API surface
   before the broader lifecycle ownership and coordination model is defined.
   Conditions are already the Kubernetes convention for publishing observable
   state, can be consumed by existing clients and controllers.

## Infrastructure Needed (Optional)

No new project infrastructure is needed.
