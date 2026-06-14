# KEP-NNNN: Node Lifecycle Conditions

<!-- toc -->
- [Release Signoff Checklist](#release-signoff-checklist)
- [Summary](#summary)
- [Motivation](#motivation)
  - [Goals](#goals)
  - [Non-Goals](#non-goals)
- [Proposal](#proposal)
  - [Use Case Coverage](#use-case-coverage)
    - [DaemonSet Controller and GNS Kubelet Disagree](#daemonset-controller-and-gns-kubelet-disagree)
    - [Jobs Stuck When Nodes Become Unreachable](#jobs-stuck-when-nodes-become-unreachable)
    - [Broken Nodes Can Consume Rollout Budget](#broken-nodes-can-consume-rollout-budget)
    - [DaemonSet Rollouts Aren't Reporting the Node's State](#daemonset-rollouts-arent-reporting-the-nodes-state)
    - [Taints are Insufficient for Signaling Node Drain](#taints-are-insufficient-for-signaling-node-drain)
    - [Reactive vs Proactive Drain](#reactive-vs-proactive-drain)
    - [Coordinating between Controllers and Admins](#coordinating-between-controllers-and-admins)
    - [Coordinating between the Autoscaler and Scheduler during Scale-in](#coordinating-between-the-autoscaler-and-scheduler-during-scale-in)
  - [Risks and Mitigations](#risks-and-mitigations)
- [Design Details](#design-details)
  - [GracefulNodeShutdownInProgress Condition](#gracefulnodeshutdowninprogress-condition)
  - [Lifecycle Conditions](#lifecycle-conditions)
  - [Writer Ownership](#writer-ownership)
  - [Feature Gate](#feature-gate)
  - [Future Extension Points](#future-extension-points)
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

- [ ] (R) Enhancement issue in release milestone, which links to KEP dir in
  [kubernetes/enhancements] (not the initial KEP PR)
- [ ] (R) KEP approvers have approved the KEP status as `implementable`
- [ ] (R) Design details are appropriately documented
- [ ] (R) Test plan is in place
- [ ] (R) Graduation criteria is in place
- [ ] (R) Production readiness review completed
- [ ] (R) Production readiness review approved
- [ ] "Implementation History" section is up-to-date for milestone
- [ ] User-facing documentation has been created in [kubernetes/website], for
  publication to [kubernetes.io]

[kubernetes.io]: https://kubernetes.io/
[kubernetes/enhancements]: https://git.k8s.io/enhancements
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
[KEP-5394](https://github.com/kubernetes/enhancements/pull/5395), which
introduces well-known Node conditions as an observable signal and then updates
core behavior to react to those condition values.

The first version is intentionally narrow. It does not define a general node
maintenance protocol. It establishes the pattern: a well-known condition
consumable on the Node, providing a foundation for future work.

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
then make core controllers use those signals for better status reporting.

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

### Non-Goals

- Define exclusive condition ownership, locking, or handoff.
- Define a contractual relationship between controllers reading and writing the
  condition.
- Define kubelet recovery behavior for lost Graceful Node Shutdown state.
- Define all the well-known conditions for a Node's lifecycle.
- Define a synchronous lifecycle state model.
- Define lifecycle policy or decide what every lifecycle condition means.
- Design how every workload controller consumes these conditions.

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
common signal to consume. The condition type and status carry the observed
lifecycle state.

For this KEP, the new Node conditions are admin managed. An admin, or an
admin-authorized maintenance controller, sets the condition status to `True`
when appropriate. Clearing the lifecycle state is done by setting the condition
status to `False` or removing the condition.

### Use Case Coverage

The conditions in this proposal are intended to provide a shared signal for the use cases
described in
[[public] Node Lifecycle Use Cases](https://docs.google.com/document/d/1EINvuVzEoRra0CKH6uQnOcQJVbd7ZnxT1bcyySN-r7c/edit?usp=sharing)
by publishing important lifecycle state on the Node.

The conditions do not, by themselves, implement every controller behavior needed
to resolve these use cases. All of these use cases should be addressed in their
own issues as follow-up work to this KEP, so the consuming controller behavior
can be designed, reviewed, and owned independently.

#### DaemonSet Controller and GNS Kubelet Disagree
Tracking Issues:
- [kubernetes/kubernetes#122912](https://github.com/kubernetes/kubernetes/issues/122912)
- [kubernetes/kubernetes#137895](https://github.com/kubernetes/kubernetes/issues/137895)

Use case 1a:
`GracefulNodeShutdownInProgress=True` and `DrainInProgress=True` tell
the DaemonSet controller that Graceful Node Shutdown and drain may explain
missing DaemonSet Pods.

Solution:
Before rescheduling a missing Pod, the DaemonSet controller would check the
"well-known" conditions on the node for additional context. If the Node has
`GracefulNodeShutdownInProgress=True` and `DrainInProgress=True`
conditions, do not reschedule the Daemonset Pod.

#### Jobs Stuck When Nodes Become Unreachable
Tracking Issues:
- [kubernetes/kubernetes#134038](https://github.com/kubernetes/kubernetes/issues/134038)

Use case 1b:
`MaintenanceInProgress=True` gives Job and queueing controllers a
Node-level signal that Pods on the Node may need special accounting when an
admin or maintenance controller has identified the Node is being lifecycled.

Solution:
The Job controller configured with `podReplacementPolicy: Failed` should check
the Node for additional context when account for Pods. When the controller sees
a Node condition `MaintenanceInProgress=True`, the controller should
not wait for the Job's Pods to reach Failed/Succeeded phase before moving on.

#### Broken Nodes Can Consume Rollout Budget
Tracking Issues:
- [kubernetes/kubernetes#138240](https://github.com/kubernetes/kubernetes/issues/138240)

Use case 1c:
`MaintenanceInProgress=True` explains to the DaemonSet controller that
some Nodes are not ideal for placement during rollout.

Solution:
When the DaemonSet controller rolls out new Pods, it should check the Node's
conditions for additional lifecycle context. When a Node has
`MaintenanceInProgress=True`, the controller should deprioritize this
node for scheduling.

#### DaemonSet Rollouts Aren't Reporting the Node's State
Tracking Issues:
- [kubernetes/kubernetes#139226](https://github.com/kubernetes/kubernetes/issues/139226)

Use case 1d:
DaemonSet status can surface `GracefulNodeShutdownInProgress` and
`MaintenanceInProgress` attribution under `status.unavailable`, so
rollout verifiers can distinguish rollout failures from lifecycle-related
unavailability.

Solution:
The DaemonSet controller is missing important context about the node's lifecycle
state that would be useful to share with the user. When the DaemonSet controller
schedules or has scheduled Pods to a node with conditions `GracefulNodeShutdownInProgress`
or `MaintenanceInProgress`, the total number of Daemonset Pods on those
nodes can be shared under `status.unavailable`.

#### Taints are Insufficient for Signaling Node Drain
Tracking Issues:
- [kubernetes/kubernetes#25625](https://github.com/kubernetes/kubernetes/issues/25625)
- [kubernetes-sigs/cluster-api#3365](https://github.com/kubernetes-sigs/cluster-api/issues/3365)
- [kubernetes/autoscaler#8157](https://github.com/kubernetes/autoscaler/issues/8157)
- [kubernetes/kubernetes#138719](https://github.com/kubernetes/kubernetes/issues/138719)

Use case 2:
`DrainInProgress=True` or `Drained=True` gives users a clear
Node status signal that is separate from scheduling policy and can attest to
lifecycle progress.

Solution:
There are many solutions that can be implemented for this one, so will focus on
`kubectl drain` for simplicity. When running `kubectl drain`, the Node will receive
condition `DrainInProgress=True`. When `kubectl drain` completes,
`Drained=True` can be set. It will be up to the admin to clear the
state.


#### Reactive vs Proactive Drain
Tracking Issues:
- [rook/rook#4290](https://github.com/rook/rook/issues/4290)
- [rook/rook#16086](https://github.com/rook/rook/issues/16086)
- [rook/rook#16976](https://github.com/rook/rook/issues/16976)
- [Red Hat Bugzilla 1861104](https://bugzilla.redhat.com/show_bug.cgi?id=1861104)

Use case 2a:
`MaintenancePlanned=True` gives storage operators a proactive signal before
drain begins, while `DrainInProgress=True` and `Drained=True` give them progress
signals during the disruption.

Solution:
The admin should set `MaintenancePlanned=True` on the Node before
performing maintenance. Then, rook can adjust the Ceph PDBs to handle disruption
on that node.

#### Coordinating between Controllers and Admins
Tracking Issues:
- [kubernetes/node-problem-detector#1176](https://github.com/kubernetes/node-problem-detector/issues/1176)
- [kubernetes/node-problem-detector#457](https://github.com/kubernetes/node-problem-detector/issues/457)
- [kubernetes/enhancements#1403](https://github.com/kubernetes/enhancements/issues/1403)
- [openshift/enhancements#141](https://github.com/openshift/enhancements/pull/141)

Use case 3a:
`MaintenanceInProgress=True` lets Node Problem Detector and other
automation distinguish a Node actively controlled by an admin from an unexpected
failure.

Solution:
The Node Problem Detector should check the node conditions for additional
context. When NPD sees a Node with `MaintenanceInProgress=True`, the
controller can skip publish conditions on the node.

#### Coordinating between the Autoscaler and Scheduler during Scale-in
Tracking Issues:
- [kubernetes/kubernetes#138718](https://github.com/kubernetes/kubernetes/issues/138718)

Use case 4:
`DrainInProgress=True`, `Drained=True`, `MaintenanceInProgress=True`, or
`MaintenancePlanned=True` provides a Node-local signal that higher-level
scale-in and workload controllers can later consume when deciding where
disruption should be concentrated.

Solution:
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

### Feature Gate

Add the `NodeLifecycleConditions` feature gate.

Components:

- `kube-controller-manager`
- `kube-apiserver`

When the feature gate is disabled:

- Kubernetes controllers do not recognize `GracefulNodeShutdownInProgress`,
  `DrainInProgress`, `Drained`, `MaintenancePlanned`, or
  `MaintenanceInProgress` as well-known conditions for core controller
  consumption.

### Future Extension Points

The use cases and the bugs associated with them should be addressed separately.
Those designs would leverage this KEP to fill the gap around node state that
control plane components don't currently have.

This KEP is a precursor to
[Specialized Lifecycle Management](https://github.com/kubernetes/enhancements/issues/5683).
It creates the well-known Node conditions that can be used to influence core
Kubernetes controller functionality. SLM can later support condition ownership,
coordination, and define a synchronous state model behind an API.

### Test Plan

- [x] I/we understand the owners of the involved components may require updates
  to existing tests to make this code solid enough prior to committing the
  changes necessary to implement this enhancement.

#### Prerequisite testing updates

None.

#### Unit tests

- Unit tests for reading `GracefulNodeShutdownInProgress=True` from Nodes.
- Unit tests for reading lifecycle condition type/status values from Nodes.

#### Integration tests

- Verify that the Node has `GracefulNodeShutdownInProgress`,
  `DrainInProgress`, `Drained`, `MaintenancePlanned`, and
  `MaintenanceInProgress` conditions.

#### e2e tests

For alpha, e2e coverage should verify that the condition is exposed on the Node.
The core controller tests will be tracked in their own issues.

### Graduation Criteria

#### Alpha

- `GracefulNodeShutdownInProgress` condition type is added.
- `DrainInProgress` condition type is added.
- `Drained` condition type is added.
- `MaintenancePlanned` condition type is added.
- `MaintenanceInProgress` condition type is added.
- Initial reason cause categories are documented.
- `NodeLifecycleConditions` feature gate is added.
- Unit tests cover condition reading and feature-gate behavior.

#### Beta

- Define DaemonSet behavior when Node is Graceful Node Shutdown state.
- Define DaemonSet and Job behavior when Node is undergoing maintenance
- Define kubelet recovery behavior for lost Graceful Node Shutdown state.
- Address condition ownership, locking, and coordination behind a new API.
- Gather feedback from rollout tooling and large-cluster operators.
- Decide whether additional lifecycle condition types should be standardized.
- Feature gate defaults to enabled.

#### GA

- All beta feedback is resolved.
- No open questions remain about field semantics.
- Feature gate is removed.

### Upgrade / Downgrade Strategy

This KEP is additive. Anything using these condition types can retain their
current meaning.

On upgrade, clusters that enable the feature gate may see new
`GracefulNodeShutdownInProgress`, `DrainInProgress`, `Drained`,
`MaintenancePlanned`, and `MaintenanceInProgress` conditions on Nodes.

On downgrade or feature disablement, core controllers do not use the conditions.
Existing admin-written conditions may remain on Nodes.

### Version Skew Strategy

If a future controller-manager version supports consuming these conditions but
the current controller-manager does not, the conditions are not consumed.

If `kube-controller-manager` supports the feature but the feature gate is not
enabled, behavior remains unchanged.

If multiple controller-manager versions run during an upgrade, controllers may
temporarily differ in whether they consume these conditions.

## Production Readiness Review Questionnaire

### Feature Enablement and Rollback

###### How can this feature be enabled / disabled in a live cluster?

- [x] Feature gate
  - Feature gate name: `NodeLifecycleConditions`
  - Components depending on the feature gate:
    - `kube-apiserver`
    - `kube-controller-manager`

###### Does enabling the feature change any default behavior?

No scheduling or rollout behavior changes. Enabling the feature adds an
observability signal.

###### Can the feature be disabled once it has been enabled?

Yes. Disabling the feature gate stops core controllers from using these conditions.
It does not remove admin-written Node conditions.

###### What happens if we reenable the feature if it was previously rolled back?

Any core Kubernetes controllers using these conditions will leverage them for
their work.

###### Are there any tests for feature enablement/disablement?

Unit tests will cover feature-gate behavior.

### Rollout, Upgrade and Rollback Planning

###### How can a rollout or rollback fail? Can it impact already running workloads?

The feature is informational and does not affect running workloads. If the Node
already has these conditions, there's still no affect because the admins controls
the values.

Rollout or rollback failures can result in missing or stale conditions.

###### What specific metrics should inform a rollback?

Unexpected increases in Node status updates or API server write latency should
inform rollback.

###### Were upgrade and rollback tested? Was the upgrade->downgrade->upgrade path tested?

TBD before beta.

###### Is the rollout accompanied by any deprecations and/or removals of features, APIs, fields of API types, flags, etc.?

No.

### Monitoring Requirements

###### How can an operator determine if the feature is in use by workloads?

Operators can inspect Nodes for the `GracefulNodeShutdownInProgress`,
`DrainInProgress`, `Drained`, `MaintenancePlanned`, and
`MaintenanceInProgress` conditions.

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
updates when these conditions change.

###### Will enabling / using this feature result in introducing new API types?

No.

###### Will enabling / using this feature result in any new calls to the cloud provider?

No.

###### Will enabling / using this feature result in increasing size or count of the existing API objects?

Yes. Nodes may include five additional conditions.

###### Will enabling / using this feature result in increasing time taken by any operations covered by existing SLIs/SLOs?

No expected impact.

###### Will enabling / using this feature result in non-negligible increase of resource usage?

No. The conditions are stored on existing Node status.

###### Can enabling / using this feature result in resource exhaustion of some node resources?

No.

### Troubleshooting

###### How does this feature react if the API server and/or etcd is unavailable?

Controllers cannot publish updated Node status until API server and
etcd availability returns.

###### What are other known failure modes?

- Stale `DrainInProgress` condition:
  - Detection: check if the Node is Tainted as `Unschedulable`. If not, it's
    likely the condition is stale.

## Implementation History

- 2026-06-04: Initial provisional KEP draft.

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
