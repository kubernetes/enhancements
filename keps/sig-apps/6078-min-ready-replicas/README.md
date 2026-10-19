# KEP-6078: Configurable Available Thresholds for Workload Controllers

<!-- toc -->
- [Summary](#summary)
- [Motivation](#motivation)
  - [Goals](#goals)
  - [Non-Goals](#non-goals)
- [Proposal](#proposal)
  - [Evidence and scope](#evidence-and-scope)
  - [User Stories](#user-stories)
  - [Risks and Mitigations](#risks-and-mitigations)
- [Design Details](#design-details)
  - [API Changes](#api-changes)
  - [Controller Behavior](#controller-behavior)
  - [Interaction with maxUnavailable](#interaction-with-maxunavailable)
  - [Deployment Controller](#deployment-controller)
  - [StatefulSet Controller](#statefulset-controller)
  - [DaemonSet Controller](#daemonset-controller)
  - [Consumer behavior](#consumer-behavior)
  - [Test Plan](#test-plan)
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

## Summary

Add a `minAvailableReplicas` field to Deployment, StatefulSet, and DaemonSet. When set to a positive value, it defines the absolute number of replicas that must be Available before the workload's `Available` condition is True. Deployment keeps its existing condition when unset or zero; StatefulSet and DaemonSet publish the new condition only when a positive threshold is configured. The field controls a status signal only; it does not gate Pod readiness, Service endpoints, HPA decisions, or rollout progression.

## Motivation

The current Deployment `Available` condition does not mean that all replicas are Ready. The Deployment controller sets it when `availableReplicas >= spec.replicas - maxUnavailable`; StatefulSet exposes `availableReplicas`, and DaemonSet exposes `numberAvailable`. These counters and conditions do not provide a common, user-selected absolute threshold across all three workload types.

The motivating [issue #134336] describes traffic imbalance during workload startup and asks for a rule for when a controller's replicas are available. The existing Deployment threshold is coupled to `maxUnavailable`, which also controls rollout behavior; changing it to obtain a status threshold changes the rollout policy too. For StatefulSets and DaemonSets, consumers can inspect the replica counters, but there is no common `Available` condition with an independently configured threshold.

This proposal adds an independent status threshold. It is intentionally limited to reporting workload availability. It does not solve the reported traffic imbalance by itself: Services route according to Pod readiness and EndpointSlice state, not a workload's `Available` condition. A consumer that wants to gate traffic must explicitly watch and act on the condition; that integration is outside this KEP.

### Goals

- Introduce `minAvailableReplicas` on Deployment, StatefulSet, and DaemonSet specs
- Define a common `Available` condition threshold using each controller's existing Available replica count when the field is positive
- Preserve existing behavior when the field is omitted
- Feature gated across alpha/beta/stable graduation

### Non-Goals

- Changing pod-level readiness probes or readiness gates
- Changing Pod readiness, Service endpoint selection, or kube-proxy behavior
- Changing rollout strategy or rollout progression
- Implementing per-zone or per-node availability rules

## Proposal

Add an optional `minAvailableReplicas` field to `DeploymentSpec`, `StatefulSetSpec`, and `DaemonSetSpec`. If omitted, each controller retains its existing status behavior. If set to a positive integer N, the controller sets `Available=True` when its total Available replica count is at least `min(N, desiredReplicas)`. `AvailableReplicas` accounts for `minReadySeconds`; the field does not count merely Ready Pods.

For Deployments the count includes Available replicas across the Deployment's owned ReplicaSets, matching the existing `AvailableReplicas` status field. For StatefulSets and DaemonSets it uses their existing `availableReplicas` and `numberAvailable` counters. The threshold does not select only the newest ReplicaSet and does not affect the rollout controller's decisions.

### Evidence and scope

The current Deployment condition compares `availableReplicas` with
`spec.replicas - MaxUnavailable(deployment)` in the
[Deployment controller]. The existing [`TestMinAvailable`] verifies that 10
replicas with `maxUnavailable: 5` produce a threshold of 5. I ran
`GOTMPDIR=/home/krosci/tmp-go-build go test ./pkg/controller/deployment/util -run '^TestMinAvailable$' -count=1`
against upstream Kubernetes commit `bbf142898f3`; it passed.

The HPA replica calculator counts Pods that are Running and PodReady in
[`getReadyPodsCount`]; it does not consume the Deployment `Available`
condition. kube-proxy builds endpoint state from EndpointSlice/Pod readiness,
not from Deployment conditions. These code paths support the boundary above:
the proposed condition is a status signal only.

The reported user story about delaying traffic until a threshold is met is not demonstrated by this proposal. A `Deployment` condition does not change Pod readiness or Service endpoint eligibility, and the HPA does not consume this condition. The field is therefore proposed only for consumers that explicitly observe the workload status. Before implementation, a concrete consumer must show why reading the existing replica counters or configuring `maxUnavailable` is insufficient; this KEP does not claim a built-in traffic or HPA effect.

### User Stories

**Story 1: Threshold during initial scale-up**

An operator initially scales a 20-replica Deployment from zero and configures
`maxUnavailable: 0` so the rollout strategy retains all desired capacity. A
progressive delivery controller wants the workload's `Available` condition to
become True once 15 replicas are Available. `minAvailableReplicas: 15` supplies
that status threshold. It does not prevent Services from sending traffic to
individual Ready Pods before 15 replicas are Available. A separate
traffic-gating mechanism is required for that behavior. During a revision
rollout, the condition counts old and new replicas together and is not a signal
that 15 replicas of the new revision are Available.

**Story 2: Standard condition for a StatefulSet consumer**

A consumer observes a StatefulSet with five replicas and wants a standard
condition once at least three replicas are Available. `minAvailableReplicas: 3`
provides that signal; it does not prove application quorum or authorize reads.
The application must continue to enforce its own quorum and health rules.

### Risks and Mitigations

**Condition is mistaken for traffic readiness or application quorum:** The field changes only a workload status condition. It does not gate Service endpoints, certify application health, or change the rollout strategy. API comments and documentation must state these limits. Consumers that need traffic gating must use Pod readiness or another explicit traffic mechanism.

**Redundant API:** Deployment already exposes `availableReplicas` and has an `Available` condition; StatefulSet and DaemonSet expose available replica counters. Before implementation, a concrete consumer must establish why those counters and the existing Deployment `maxUnavailable` behavior are insufficient. This KEP does not treat convenience alone as evidence of a core API requirement.

**Confusion with `minReadySeconds`:** The proposed field counts Available replicas, which already accounts for `minReadySeconds`; it does not count Pods that are merely Ready.

**Misinterpreting the condition as traffic readiness**: A lower threshold can make `Available=True` while other Pods are not yet Available. The condition is only a workload-level status signal; documentation and API comments must state that it does not gate Services or certify application quorum.

## Design Details

### API Changes

A new field is added to each workload spec type:

```go
// DeploymentSpec
type DeploymentSpec struct {
    // ... existing fields

    // Minimum number of Available replicas for the workload's Available
    // condition. When unset or zero, the controller's existing behavior applies.
    // +optional
    MinAvailableReplicas int32 `json:"minAvailableReplicas,omitempty"`
}

// StatefulSetSpec
type StatefulSetSpec struct {
    // ... existing fields
    // +optional
    MinAvailableReplicas int32 `json:"minAvailableReplicas,omitempty"`
}

// DaemonSetSpec
type DaemonSetSpec struct {
    // ... existing fields
    // +optional
    MinAvailableReplicas int32 `json:"minAvailableReplicas,omitempty"`
}
```

Validation rules:
- Must be non-negative; zero or omission means the pre-feature behavior.
- Do not reject a value greater than the current desired count. Such validation
  would make a later scale-down require a second spec change. The effective
  threshold is `min(minAvailableReplicas, desiredReplicas)` at reconciliation
  time, including for DaemonSets whose desired count changes with matching
  nodes.

### Controller Behavior

Each controller computes its desired "available threshold" as:

```
if minAvailableReplicas > 0:
    threshold = min(minAvailableReplicas, desiredReplicas)
else:
    threshold = existingControllerThreshold
```

When the controller's current Available replica count reaches this threshold,
the controller sets `Available=True`. For Deployment, the omitted/zero case
continues to use `spec.replicas - maxUnavailable`; it does not mean all desired
replicas must be Available. For StatefulSet and DaemonSet, a positive field
adds a standard `Available` condition based on their existing Available
replica counters. When the field is omitted or zero, no new condition is added,
preserving current status behavior.

Use the existing Deployment reasons where applicable and define the corresponding
stable condition type and reasons for StatefulSet and DaemonSet. Do not emit a
separate Event: condition changes are observable in status, and an additional
event would duplicate that state transition.

### Interaction with maxUnavailable

`minAvailableReplicas` only influences the workload's `Available` condition. It
does not alter rollout strategy or progression.

**Scenario**: Deployment with 10 replicas, `maxUnavailable: 2`, and
`minAvailableReplicas: 5` during initial scale-up from zero.

The rollout proceeds as follows:
- The Deployment controller respects `maxUnavailable` independently. During a rolling update, it ensures at most 2 replicas are unavailable at any time, regardless of `minAvailableReplicas`.
- The `Available` condition transitions to `True` once 5 replicas across the
  Deployment's owned ReplicaSets are Available, even though the rollout may
  still be in progress.
- Rollout progression and completion continue to follow existing Deployment controller rules.

**No stall or deadlock**: Because `minAvailableReplicas` does not gate the rollout logic, setting a high threshold cannot stall the update. The controller continues creating and destroying pods according to `maxSurge`/`maxUnavailable` until the rollout is fully complete. The `Available` condition is merely a status signal — not a rollout gate.

This decoupling is intentional: the condition is a status signal for consumers
that explicitly watch the workload; it is not a throttle or traffic gate.

### Deployment Controller

The Deployment controller currently sets `Available=True` when the total
available replica count meets `spec.replicas - maxUnavailable`. With a positive
`minAvailableReplicas`, it compares the total `AvailableReplicas` count across
owned ReplicaSets with the effective threshold:

```
threshold = min(minAvailableReplicas, desiredReplicas)
```

The `DeploymentAvailable` condition is set based on the total Available replica
count, not only the newest ReplicaSet. When the field is zero, current
`maxUnavailable`-based behavior is preserved.

The existing `MinimumReplicasAvailable` and `MinimumReplicasUnavailable`
reasons remain; messages should include the effective threshold and current
Available count when the field is set.

### StatefulSet Controller

StatefulSet already computes `.status.availableReplicas` and `.status.readyReplicas`. It does not expose an `Available` condition. When `minAvailableReplicas` is positive:

- Add a standard `Available` condition to StatefulSet status.
- Set it True when `.status.availableReplicas >= min(minAvailableReplicas, desiredReplicas)`.
- When the field is omitted or zero, leave StatefulSet conditions unchanged.

### DaemonSet Controller

DaemonSet already tracks `.status.numberAvailable` and has an extensible conditions list. When `minAvailableReplicas` is positive:

- Add a standard `Available` condition to DaemonSet status.
- Set it True when `.status.numberAvailable >= min(minAvailableReplicas, desiredNumberScheduled)`.
- When the field is omitted or zero, leave DaemonSet conditions unchanged.

**Dynamic node count**: A DaemonSet's desired count changes as nodes join, leave, or stop matching its selector. Recompute the effective threshold from the current `desiredNumberScheduled` on every reconciliation. This prevents an unreachable absolute threshold from keeping the condition False after the desired count falls.

An absolute threshold is intentional: the field reports a minimum number, not a percentage of a changing node population. A percentage form is not included in this proposal.

### Consumer behavior

The HPA does not consume a Deployment's `Available` condition, and Service
endpoint selection is based on Pod and EndpointSlice readiness. This feature
does not change either behavior. Consumers that need the new threshold must
explicitly watch the workload condition. The KEP currently has no measured
integration showing that an existing consumer needs this field; identifying
such a consumer is a prerequisite to implementation, not a SIG choice between
different designs.

### Test Plan

[X] I understand the owners of the involved components may require updates to existing tests to make this code solid enough prior to committing the changes necessary to implement this enhancement.

##### Unit tests

- `pkg/controller/deployment`: verify omitted/zero retains the current
  `replicas - maxUnavailable` threshold; positive values use total Available
  replicas across old and new ReplicaSets; cover scale-up/down and values above
  the desired count.
- `pkg/controller/statefulset`: verify the condition against
  `.status.availableReplicas`, including `minReadySeconds`, zero replicas,
  scaling, and thresholds above desired.
- `pkg/controller/daemon`: verify the condition against `.status.numberAvailable`
  as nodes join, leave, become unschedulable, or stop matching the selector.
- API validation/defaulting and feature-gate tests for each served API version;
  verify old controllers and disabled gates handle stored objects safely.
- Verify the field does not change Pod readiness, EndpointSlice contents, HPA
  calculations, `maxUnavailable`, or rollout progression.

##### Integration tests

- Deployment rolling update where `Available` transitions at the configured
  threshold while rollout strategy and completion remain unchanged.
- StatefulSet and DaemonSet updates where conditions match Available counters
  and do not claim application quorum or gate traffic.
- Feature-gate disable/enable across API server and controller-manager skew.

##### e2e tests

- Verify Deployment availability transitions with `minAvailableReplicas`
- Verify StatefulSet availability transitions with `minAvailableReplicas`
- Verify backward compatibility when the field is omitted or zero.

### Graduation Criteria

#### Alpha

- Feature gate `MinAvailableReplicas` disabled by default
- API types accept `minAvailableReplicas`
- Deployment controller respects `minAvailableReplicas`
- StatefulSet `.status.conditions` with `Available` type
- DaemonSet `.status.conditions` with `Available` type
- Unit and integration tests

#### Beta

- Feature gate enabled by default
- Additional e2e tests
- A concrete consumer demonstrates that existing counters and Deployment
  `maxUnavailable` cannot meet its requirement.
- User feedback confirms a common condition is useful independently of
  Service traffic gating.

#### GA

- All tests stable for two releases
- Real-world usage feedback incorporated
- Conformance tests for availability condition

### Upgrade / Downgrade Strategy

On upgrade, omitted fields preserve the existing Deployment threshold. The new
StatefulSet and DaemonSet `Available` condition is additive. Downgrade with a
non-zero field is not assumed safe: older API servers may not preserve fields
they do not know. Before alpha, tests must establish exact API-server and
controller-manager skew behavior. If storage or round trips can drop the
field, the upgrade strategy must require clearing it before downgrade and
document that loss; the KEP does not claim automatic recovery.

### Version Skew Strategy

The feature is controlled by the kube-apiserver (for type validation and storage) and kube-controller-manager (for controller behavior). Both must be at the minimum version for the feature to function. There is no kubelet dependency.

## Production Readiness Review Questionnaire

### Feature Enablement and Rollback

###### How can this feature be enabled / disabled in a live cluster?

- [X] Feature gate
  - Feature gate name: MinAvailableReplicas
  - Components depending on the feature gate: kube-apiserver, kube-controller-manager

###### Does enabling the feature change any default behavior?

For Deployment, zero preserves its existing `maxUnavailable`-based condition.
For StatefulSet and DaemonSet, zero or omission leaves status conditions
unchanged; the new `Available` condition is published only when a positive
threshold is configured.

###### Can the feature be disabled once it has been enabled?

Disabling the controller-manager gate makes the custom threshold inactive.
Deployment returns to its existing `maxUnavailable`-based condition, and
StatefulSet/DaemonSet remove the condition added by this feature. The API-server
gate controls acceptance and round-tripping of the field; this KEP does not
assume that a value survives disabling that gate or downgrading. Tests must
verify the API-server and controller-manager gates independently before alpha.

###### What happens if we reenable the feature if it was previously rolled back?

The values become active again only if API-server handling preserved them.

###### Are there any tests for feature enablement/disablement?

Unit tests cover omitted, zero, positive, and greater-than-desired values with
the feature enabled and disabled. They verify the legacy Deployment threshold,
that StatefulSet/DaemonSet conditions are absent for omitted or zero values,
and that disabling the controller-manager gate removes feature-generated
conditions.

### Rollout, Upgrade and Rollback Planning

###### How can a rollout or rollback fail? Can it impact already running workloads?

If a user sets `minAvailableReplicas` too low, the controller may report
Available earlier than intended; Pods and existing Service routing are not
changed. Rollback safety depends on field preservation across API-server
versions and is an alpha prerequisite, not an assumed property.

###### What specific metrics should inform a rollback?

This status-only feature adds no data-plane behavior. Operators should compare
the condition with the available count and configured threshold; traffic
errors are not a direct signal of this condition's correctness.

###### Were upgrade and rollback tested? Was the upgrade->downgrade->upgrade path tested?

This will be tested during the Beta phase.

###### Is the rollout accompanied by any deprecations and/or removals of features, APIs, fields of API types, flags, etc.?

No.

### Monitoring Requirements

###### How can an operator determine if the feature is in use by workloads?

- Inspect the workload spec for `minAvailableReplicas` and compare it with the
  desired and Available replica counts.

###### How can someone using this feature know that it is working for their instance?

- Controller `.status.conditions` reflects the new threshold
- A consumer can observe the condition and compare it with the configured
  threshold. `kubectl rollout status` is not changed by this feature.

###### What are the reasonable SLOs for the enhancement?

The feature must meet existing controller status reconciliation expectations.
This KEP does not define a new latency SLO.

###### What are the SLIs an operator can use to determine the health of the service?

- Existing Available replica counters and status conditions

###### Are there any missing metrics that would be useful to have to improve observability of this feature?

No feature-specific metric is required for this status-only field; existing
replica counters and conditions expose its behavior.

### Dependencies

No external dependencies. The feature is self-contained in kube-apiserver and kube-controller-manager.

### Scalability

###### Will enabling / using this feature result in any new API calls?

No. The feature only changes when existing condition updates are performed.

###### Will enabling / using this feature result in introducing new API types?

No. Existing API types gain a new field.

###### Will enabling / using this feature result in any new calls to the cloud provider?

No.

###### Will enabling / using this feature result in increasing size or count of the existing API objects?

A single `int32` field (4 bytes) is added to each affected workload spec.

###### Will enabling / using this feature result in increasing time taken by any operations covered by existing SLIs/SLOs?

No. The computation is O(1) and adds negligible overhead.

###### Will enabling / using this feature result in non-negligible increase of resource usage (CPU, RAM, disk, IO, ...) in any components?

No.

###### Can enabling / using this feature result in resource exhaustion of some node resources (PIDs, sockets, inodes, etc.)?

No.

### Troubleshooting

###### How does this feature react if the API server and/or etcd is unavailable?

The controller cannot update the workload status, so condition transitions are delayed until connectivity is restored. This is the same as existing behavior.

###### What are other known failure modes?

- If `minAvailableReplicas` exceeds the desired count, the effective threshold
  is clamped to the current desired count; this is recomputed after scaling or
  DaemonSet node membership changes.
- The condition may be True while some Pods are not Available. It must not be
  interpreted as permission to send traffic to those Pods or proof of
  application-level readiness.

###### What steps should be taken if SLOs are not being met to determine the problem?

Check the workload's `.status.conditions`, its Available replica counter, the
desired replica count, and the effective threshold. Do not compare against
`readyReplicas` when `minReadySeconds` is non-zero.

## Implementation History

- 2026-05-14: Initial KEP draft (provisional)
- 2026-10-20: Corrected existing Deployment availability semantics, removed
  unsupported HPA and traffic-gating claims, and scoped the proposal to an
  independent status threshold.

## Drawbacks

- Adds API surface area and complexity to three workload types
- The field may duplicate information available from replica counters; a
  concrete status consumer is required to justify this core API addition.
- Consumers may mistake workload availability for Service traffic gating.
- The cross-controller condition adds status API surface to StatefulSet and
  DaemonSet.

## Alternatives

**Pod readiness gates or an application-owned readiness controller**: Use
Pod readiness to gate Service endpoints when the actual requirement is to
prevent traffic until a threshold is reached. This addresses the traffic
imbalance in the motivating issue; the proposed workload status condition does
not. The KEP does not reject this alternative for that distinct requirement.

**Consumer computes the threshold from existing status**: A consumer can
compare Deployment `availableReplicas` with its own threshold, and can do the
same with StatefulSet or DaemonSet counters. This avoids a new API field, and
must be evaluated against a concrete consumer before adding a core field.

**Percentage-based threshold**: Not included. The issue asks for an absolute
replica count, and a percentage would be a different API contract.


[Deployment controller]: https://github.com/kubernetes/kubernetes/blob/bbf142898f3/pkg/controller/deployment/sync.go#L523
[`TestMinAvailable`]: https://github.com/kubernetes/kubernetes/blob/bbf142898f3/pkg/controller/deployment/util/deployment_util_test.go#L1378
[`getReadyPodsCount`]: https://github.com/kubernetes/kubernetes/blob/bbf142898f3/pkg/controller/podautoscaler/replica_calculator.go#L323
[issue #134336]: https://github.com/kubernetes/kubernetes/issues/134336
