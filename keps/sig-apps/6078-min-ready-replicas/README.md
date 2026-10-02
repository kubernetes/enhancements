# KEP-6078: MinReadyReplicas for Workload Controllers

<!-- toc -->
- [Summary](#summary)
- [Motivation](#motivation)
  - [Goals](#goals)
  - [Non-Goals](#non-goals)
- [Proposal](#proposal)
  - [User Stories](#user-stories)
  - [Risks and Mitigations](#risks-and-mitigations)
- [Design Details](#design-details)
  - [API Changes](#api-changes)
  - [Controller Behavior](#controller-behavior)
  - [Interaction with maxUnavailable](#interaction-with-maxunavailable)
  - [Deployment Controller](#deployment-controller)
  - [StatefulSet Controller](#statefulset-controller)
  - [DaemonSet Controller](#daemonset-controller)
  - [HPA Integration](#hpa-integration)
  - [Test Plan](#test-plan)
  - [Graduation Criteria](#graduation-criteria)
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

Add a `minReadyReplicas` field to the spec of Deployment, StatefulSet, and DaemonSet. When set to a positive integer, the controller is considered Available once at least that many replicas are Ready, rather than waiting for all replicas. This gives operators control over the availability signal used by higher-level systems (HPA, network controllers, rollouts).

## Motivation

Controllers today expose a binary Available condition: True only when all replicas are Ready, False otherwise. This all-or-nothing signal is too coarse for operators and higher-level systems that make rollout, rollback, or scaling decisions based on partial availability.

During a rolling update of a controller with many replicas that take time to warm up, there is a long gap between "some replicas Ready" and "all replicas Ready" where no intermediate signal exists. Consumers of the Available condition — such as progressive delivery controllers, automated rollback tools, and rollouts — have no way to distinguish "still progressing, partially available" from "not yet available."

Adding a configurable threshold (`minReadyReplicas`) lets operators define when the controller is considered Available, giving consumers an intermediate signal while preserving the existing all-or-nothing behavior as the default (0).

### Goals

- Introduce `minReadyReplicas` on Deployment, StatefulSet, DaemonSet specs
- Controller Available condition becomes True once at least `minReadyReplicas` replicas are Ready
- Backward compatible: default 0 preserves existing behavior (all replicas must be Ready)
- Feature gated across alpha/beta/stable graduation

### Non-Goals

- Changing pod-level readiness probes or readiness gates
- Changing Service endpoint selection or kube-proxy behavior
- Adding new condition types to existing controllers
- Implementing per-zone or per-node availability rules

## Proposal

Add an optional `minReadyReplicas` field to `DeploymentSpec`, `StatefulSetSpec`, and `DaemonSetSpec`. When the value is zero (default), existing behavior is preserved. When set to a positive integer N, the controller sets its Available condition to True once at least N replicas in the current owner ReplicaSet or controller are Ready.

For Deployments this means the Deployment controller reports Available as soon as the latest ReplicaSet reaches `max(Replicas, minReadyReplicas)` available replicas (clamped to the actual replica count). For StatefulSets and DaemonSets the same threshold applies to the controller's own ready/available replica count.

### User Stories

**Story 1: Gradual rollout with traffic gating**

A user runs a Deployment with 20 replicas serving latency-sensitive traffic. Pods take 30 seconds to warm up caches. Without `minReadyReplicas`, the first pod to become Ready immediately receives traffic, causing overload. With `minReadyReplicas: 15`, the controller does not signal Available until 15 pods are Ready. An external network controller watches the Available condition and only programs Service endpoints once the threshold is met.

**Story 2: StatefulSet quorum-aware availability**

A user operates a StatefulSet with 5 replicas forming a clustered database. The cluster needs at least 3 members to form quorum. The user sets `minReadyReplicas: 3`. During a rolling update, the StatefulSet controller reports Available once 3 of 5 pods are Ready, signaling that the cluster can serve reads.

### Risks and Mitigations

**Premature availability signaling**: Users may set `minReadyReplicas` too low and signal availability before the system is truly ready. Mitigated by documentation and the fact that default is 0 (existing strict behavior).

**Confusion with minReadySeconds**: The two fields have similar names but address different concerns. `minReadySeconds` delays per-pod readiness; `minReadyReplicas` reduces the replica threshold for controller-level availability. Documentation will clarify the distinction.

**HPA metric skew ("yo-yo" effect)**: When `minReadyReplicas` causes the controller to report Available with fewer ready pods, the HPA may see inflated per-pod metrics and trigger an immediate scale-up. As new pods become ready, utilization drops and the HPA may scale back down, creating oscillation. Mitigated by HPA `behavior.scaleUp.stabilizationWindowSeconds` (60-120s recommended), target utilization buffers, and user awareness that this trade-off is inherent to relaxing the availability threshold.

## Design Details

### API Changes

A new field is added to each workload spec type:

```go
// DeploymentSpec
type DeploymentSpec struct {
    // ... existing fields

    // Minimum number of replicas that must be Ready for the Deployment
    // to be considered Available. Defaults to 0 (all replicas must be Ready).
    // +optional
    MinReadyReplicas int32 `json:"minReadyReplicas,omitempty"`
}

// StatefulSetSpec
type StatefulSetSpec struct {
    // ... existing fields
    // +optional
    MinReadyReplicas int32 `json:"minReadyReplicas,omitempty"`
}

// DaemonSetSpec
type DaemonSetSpec struct {
    // ... existing fields
    // +optional
    MinReadyReplicas int32 `json:"minReadyReplicas,omitempty"`
}
```

Validation rules:
- Must be non-negative
- For Deployment and StatefulSet: rejected at admission level if `minReadyReplicas > spec.replicas`
- For DaemonSet: the desired replica count is dynamic (number of matched nodes), so validation only rejects negative values at admission. The controller applies runtime clamping (see DaemonSet section).
- If `spec.replicas` is later scaled below `minReadyReplicas`, the controller auto-clamps: `threshold = min(minReadyReplicas, actualReplicas)`

### Controller Behavior

Each controller computes its desired "available threshold" as:

```
if minReadyReplicas > 0:
    threshold = min(minReadyReplicas, desiredReplicas)
else:
    threshold = desiredReplicas
```

When the controller's current available replica count reaches or exceeds this threshold, the controller sets its Available condition to True. When `minReadyReplicas` is 0, the threshold equals `desiredReplicas` (current behavior).

The `Available` condition status is set to `False` with reason `MinimumReadyReplicasUnavailable` when below the threshold, and `True` with reason `MinimumReplicasAvailable` when at or above it.

When the `Available` condition transitions to `True` due to the `minReadyReplicas` threshold being crossed, the controller emits a Kubernetes Event `TargetAvailabilityReached` on the workload object with a message indicating the threshold and the actual available replica count.

### Interaction with maxUnavailable

This is the most critical design constraint. `minReadyReplicas` only influences the controller's `Available` condition signal. It must NOT alter the rollout strategy itself.

**Scenario**: Deployment with 10 replicas, `maxUnavailable: 2` (at least 8 pods must be running during update), `minReadyReplicas: 5`.

The rollout proceeds as follows:
- The Deployment controller respects `maxUnavailable` independently. During a rolling update, it ensures at most 2 pods are unavailable at any time, regardless of `minReadyReplicas`.
- The `Available` condition transitions to `True` once 5 pods in the new ReplicaSet are Ready, even though the rollout may still be in progress.
- The rollout completes when all 10 pods in the new ReplicaSet are Ready (existing behavior).

**No stall or deadlock**: Because `minReadyReplicas` does not gate the rollout logic, setting a high threshold cannot stall the update. The controller continues creating and destroying pods according to `maxSurge`/`maxUnavailable` until the rollout is fully complete. The `Available` condition is merely a status signal — not a rollout gate.

This decoupling is intentional: `minReadyReplicas` is a signal for external consumers (network controllers, HPA, operators), not a throttle for the controller itself.

### Deployment Controller

The Deployment controller currently computes a `newRS` (new ReplicaSet) and checks if `newRS.Status.AvailableReplicas >= desiredReplicas`. With `minReadyReplicas`:

```
threshold = min(minReadyReplicas, desiredReplicas)
           or desiredReplicas if minReadyReplicas == 0
```

The condition `DeploymentAvailable` is set based on `newRS.Status.AvailableReplicas >= threshold`.

The existing `MinimumReplicasAvailable` and `MinimumReplicasUnavailable` reasons remain; for `minReadyReplicas` the messages are updated to reflect the threshold.

### StatefulSet Controller

StatefulSet already computes `.status.availableReplicas` and `.status.readyReplicas`. The controller currently does not expose an `Available` condition in `.status.conditions`. With `minReadyReplicas`:

- Add `.status.conditions` with `Available` type to StatefulSet (mirroring Deployment)
- The condition is True when `.status.availableReplicas >= min(minReadyReplicas, replicas)`

### DaemonSet Controller

DaemonSet already tracks `.status.numberAvailable`. With `minReadyReplicas`:

- Add `.status.conditions` with `Available` type to DaemonSet (mirroring Deployment)
- The condition is True when `.status.numberAvailable >= minReadyReplicas`

**Dynamic node count**: Unlike Deployment and StatefulSet, a DaemonSet's desired replica count changes as nodes join or leave the cluster. If `minReadyReplicas` is set to an absolute value larger than the current number of matched nodes, the controller clamps: `threshold = min(minReadyReplicas, desiredReplicas)`. This prevents a permanent `Available: False` state when the cluster shrinks.

For DaemonSets, a percentage-based equivalent (e.g. `minReadyPercent`) may be more intuitive. This is discussed in [Alternatives](#alternatives).

### HPA Integration

The Horizontal Pod Autoscaler uses the `Available` condition on Deployments as one signal for scaling decisions. When `minReadyReplicas` lowers the availability threshold, the HPA may observe a reduced ready pod count and compute inflated per-pod metrics, potentially triggering aggressive scale-up.

**Design decision**: The HPA should continue using the existing `Available` condition without special-casing `minReadyReplicas`. Operators who set `minReadyReplicas` must be aware that:
- The HPA sees the same `Available` signal as before (but it may transition to True sooner).
- If per-pod metrics are skewed during the warm-up window (fewer ready pods inflating per-pod utilization), the HPA may trigger an immediate scale-up. When new pods become ready, utilization drops and the HPA may scale back down — creating a "yo-yo" effect.
- **Explicit mitigation**: configure HPA `behavior.scaleUp.stabilizationWindowSeconds` to require the elevated metric to persist before acting. A window of 60-120s is typically sufficient to absorb the warm-up phase. Example:
  ```yaml
  behavior:
    scaleUp:
      stabilizationWindowSeconds: 120
  ```
- Additional mitigations include target utilization buffers and `--horizontal-pod-autoscaler-downscale-stabilization`.
- This is no different from existing scenarios where `minReadySeconds` delays per-pod availability.

Future work may consider an explicit HPA opt-in for `minReadyReplicas` awareness, but this is out of scope for this KEP.

### Test Plan

[X] I understand the owners of the involved components may require updates to existing tests to make this code solid enough prior to committing the changes necessary to implement this enhancement.

##### Unit tests

- `pkg/controller/deployment`: verify `Available` condition threshold with various `minReadyReplicas` values (0, 1, N, > replicas), verify no interaction with `maxUnavailable` rollout logic
- `pkg/controller/statefulset`: verify condition computation and edge cases
- `pkg/controller/daemon`: verify condition computation and edge cases, verify runtime clamping when desired replicas < minReadyReplicas
- `staging/src/k8s.io/api/apps/v1`: verify validation and defaults, verify admission rejection when `minReadyReplicas > spec.replicas`
- Verify `TargetAvailabilityReached` event emission on condition transition

##### Integration tests

- Deployment rollout with `minReadyReplicas` set during update
- StatefulSet rolling update with `minReadyReplicas` quorum threshold
- DaemonSet update with `minReadyReplicas`
- Feature gate toggle during active rollout

##### e2e tests

- Verify Deployment availability transitions with `minReadyReplicas`
- Verify StatefulSet availability transitions with `minReadyReplicas`
- Verify backward compatibility (default 0 produces unchanged behavior)

### Graduation Criteria

#### Alpha

- Feature gate `MinReadyReplicas` disabled by default
- API types accept `minReadyReplicas`
- Deployment controller respects `minReadyReplicas`
- StatefulSet `.status.conditions` with `Available` type
- DaemonSet `.status.conditions` with `Available` type
- Unit and integration tests

#### Beta

- Feature gate enabled by default
- Additional e2e tests
- Metrics for `minReadyReplicas` usage
- User feedback on API shape and behavior

#### GA

- All tests stable for two releases
- Real-world usage feedback incorporated
- Conformance tests for availability condition

### Upgrade / Downgrade Strategy

On upgrade, existing workloads are unaffected (default `minReadyReplicas=0`). On downgrade, the `minReadyReplicas` field is ignored by the older API server and controllers. Cluster must be upgraded to a version supporting the feature before the field has effect.

### Version Skew Strategy

The feature is controlled by the kube-apiserver (for type validation and storage) and kube-controller-manager (for controller behavior). Both must be at the minimum version for the feature to function. There is no kubelet dependency.

## Production Readiness Review Questionnaire

### Feature Enablement and Rollback

###### How can this feature be enabled / disabled in a live cluster?

- [X] Feature gate
  - Feature gate name: MinReadyReplicas
  - Components depending on the feature gate: kube-apiserver, kube-controller-manager

###### Does enabling the feature change any default behavior?

No. The default value of `minReadyReplicas` is 0, which preserves existing controller behavior.

###### Can the feature be disabled once it has been enabled?

Yes. Disable the feature gate and restart the components. Workloads that had `minReadyReplicas` set will have the field ignored, reverting to the default behavior of requiring all replicas to be Ready.

###### What happens if we reenable the feature if it was previously rolled back?

The `minReadyReplicas` values stored in existing workload objects become active again.

###### Are there any tests for feature enablement/disablement?

Unit tests verify that setting `minReadyReplicas` to 0 after having a non-zero value reverts to the all-replicas-ready behavior. Feature gate toggle tests verify proper enable/disable semantics.

### Rollout, Upgrade and Rollback Planning

###### How can a rollout or rollback fail? Can it impact already running workloads?

If a user sets `minReadyReplicas` too low, the controller may report Available prematurely, but this does not affect running pods or existing traffic. Rollback is safe: old controller code ignores the field.

###### What specific metrics should inform a rollback?

A sudden increase in error rates or traffic imbalance correlated with `minReadyReplicas` usage could indicate a misconfiguration. Metrics tracking condition transitions can help.

###### Were upgrade and rollback tested? Was the upgrade->downgrade->upgrade path tested?

This will be tested during the Beta phase.

###### Is the rollout accompanied by any deprecations and/or removals of features, APIs, fields of API types, flags, etc.?

No.

### Monitoring Requirements

###### How can an operator determine if the feature is in use by workloads?

- `kubectl get deployments -o jsonpath='{.items[?(@.spec.minReadyReplicas>0)].metadata.name}'`
- A metric counting workloads with `minReadyReplicas > 0`

###### How can someone using this feature know that it is working for their instance?

- Controller `.status.conditions` reflects the new threshold
- `kubectl rollout status` completes based on the `minReadyReplicas` threshold
- A Kubernetes Event `TargetAvailabilityReached` is emitted on the workload object when the threshold is crossed, viewable via `kubectl describe deployment/<name>`

###### What are the reasonable SLOs for the enhancement?

Same as existing controller SLOs: condition updates within the standard controller loop (5-10 seconds).

###### What are the SLIs an operator can use to determine the health of the service?

- Controller condition transition metrics
- Workload availability metric

###### Are there any missing metrics that would be useful to have to improve observability of this feature?

A gauge for each workload type tracking the configured `minReadyReplicas` value.

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

- Setting `minReadyReplicas > spec.replicas` on Deployment/StatefulSet: rejected at admission level with a clear error message.
- Setting `minReadyReplicas` higher than current matched nodes on DaemonSet: the controller auto-clamps to `desiredReplicas`, so `Available` only when all nodes are ready (equivalent to default behavior).
- Setting `minReadyReplicas` to 0 after a non-zero value: reverts to all-replicas-required behavior.
- During a scale-down of `spec.replicas` below `minReadyReplicas`: the controller auto-clamps via `threshold = min(minReadyReplicas, actualReplicas)`, preventing permanent `Available: False`.

###### What steps should be taken if SLOs are not being met to determine the problem?

Check the workload's `.status.conditions` for the Available condition reason. Check if `minReadyReplicas` is set appropriately relative to the actual ready replica count.

## Implementation History

- 2026-05-14: Initial KEP draft (provisional)
- 2026-05-14: Added maxUnavailable interaction, HPA integration, DaemonSet dynamic node handling, admission validation, Kubernetes Event observability, minAvailableReplicas naming alternative

## Drawbacks

- Adds API surface area and complexity to three workload types
- Potential confusion with the existing `minReadySeconds` field
- The Available condition alone does not gate Service traffic; external controllers must consume it

## Alternatives

**Readiness gate controller**: Inject a readiness gate on pods that is satisfied only when `minReadyReplicas` pods are Ready. This would directly gate Service traffic. Rejected because it introduces a new controller with pod mutation, which is more invasive and harder to roll back.

**External component**: Implement this as an external operator. Rejected because the feature is general-purpose and belongs in the core controllers for broad adoption and consistent behavior.

**Percentage-based threshold**: Use a percentage instead of an absolute number. Rejected as the primary mechanism in favor of absolute values for clarity, matching the pattern of `replicas` and `maxSurge`/`maxUnavailable`. However, a percentage-based option (e.g. `minReadyPercent`) warrants further exploration for DaemonSets where the desired replica count is dynamic. This could be added as an alternative field in a future iteration, or as a conversion helper (`minReadyPercent * desiredReplicas / 100`).

**`minAvailableReplicas` naming**: Consider naming the field `minAvailableReplicas` instead of `minReadyReplicas` to clearly distinguish it from `minReadySeconds`. In Kubernetes terminology, a pod is "Ready" when its readiness probes pass, but "Available" means Ready and stable for `minReadySeconds`. Since the controller-level condition is `Available` and the internal counter used for the threshold is `.status.availableReplicas` (which factors in `minReadySeconds`), `minAvailableReplicas` is semantically more precise. The name `minReadyReplicas` was chosen for brevity and because it mirrors the existing `MinReadySeconds` field name pattern. The naming decision should be resolved based on SIG review feedback.
