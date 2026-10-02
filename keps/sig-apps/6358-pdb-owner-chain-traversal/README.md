# KEP-6358: PDB Owner-Chain Traversal for Scalable Ancestor

<!-- toc -->
- [Release Signoff Checklist](#release-signoff-checklist)
- [Summary](#summary)
- [Motivation](#motivation)
  - [Goals](#goals)
  - [Non-Goals](#non-goals)
- [Proposal](#proposal)
  - [User Stories](#user-stories)
    - [Story 1: KubeVirt VMPool](#story-1-kubevirt-vmpool)
    - [Story 2: Argo Rollouts](#story-2-argo-rollouts)
  - [Notes/Constraints/Caveats](#notesconstraintscaveats)
  - [Risks and Mitigations](#risks-and-mitigations)
- [Design Details](#design-details)
  - [Algorithm](#algorithm)
  - [RBAC and Discovery](#rbac-and-discovery)
  - [Test Plan](#test-plan)
      - [Prerequisite testing updates](#prerequisite-testing-updates)
      - [Unit tests](#unit-tests)
      - [Integration tests](#integration-tests)
      - [e2e tests](#e2e-tests)
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

## Release Signoff Checklist

Items marked with (R) are required *prior to targeting to a milestone / release*.

- [ ] (R) Enhancement issue in release milestone, which links to KEP dir in [kubernetes/enhancements] (not the initial KEP PR)
- [ ] (R) KEP approvers have approved the KEP status as `implementable`
- [ ] (R) Design details are appropriately documented
- [ ] (R) Test plan is in place, giving consideration to SIG Architecture and SIG Testing input
- [ ] (R) Graduation criteria is in place
- [ ] (R) Production readiness review completed
- [ ] (R) Production readiness review approved
- [ ] "Implementation History" section is up-to-date for milestone
- [ ] User-facing documentation has been created in [kubernetes/website]

[kubernetes/enhancements]: https://git.k8s.io/enhancements
[kubernetes/kubernetes]: https://git.k8s.io/kubernetes
[kubernetes/website]: https://git.k8s.io/website

## Summary

The PDB disruption controller computes `expectedPods` for `maxUnavailable` PodDisruptionBudgets
by calling the `/scale` subresource on each matched pod's direct controller owner. When that
direct owner does not implement `/scale` — as is the case for KubeVirt's `VirtualMachineInstance`
(VMI) — the controller receives an error, falls back to `expectedPods: 0`, and sets
`disruptionsAllowed: 0`, blocking all evictions even when a scalable ancestor exists higher up
the ownerRef chain (e.g., `VirtualMachinePool` which does implement `/scale`).

This KEP proposes to make the disruption controller walk the ownerRef chain to the **outermost
`/scale`-bearing controller** and use that controller's replica count as `expectedPods`. The
traversal covers two cases in one unified algorithm:

1. **Direct owner lacks `/scale`** (KubeVirt VMI case) — walk up until a scalable ancestor is found.
2. **Direct owner has `/scale` but is itself owned by a higher controller** (Argo Rollouts RS case) —
   continue walking to the outermost scalable controller to avoid double-counting.

The change is gated behind a feature flag (`PDBOwnerChainTraversal`) and is generic — it works
for any multi-level owner hierarchy, not just KubeVirt or Argo.

## Motivation

Two real-world operator ecosystems are blocked today:

1. **KubeVirt VMPool** — launcher pod → `VirtualMachineInstance` [no `/scale`] → `VirtualMachine`
   → `VirtualMachinePool` [has `/scale`]. Any `maxUnavailable` PDB targeting virt-launcher pods
   results in `disruptionsAllowed: 0`, making node drain impossible for VMPool workloads.
   Tracked in https://github.com/kubevirt/kubevirt/issues/18063.

2. **Argo Rollouts** — during a rolling update, a pod is owned by one of two active `ReplicaSets`,
   each of which has `/scale`. The existing hardcoded `getPodReplicaSet` finder skips an RS only
   when it is owned by a `Deployment`; an RS owned by an Argo `Rollout` is not skipped, so both
   RSes' `spec.replicas` are summed, producing an inflated `expectedPods` count and incorrect
   `disruptionsAllowed`.

Both cases share the same root cause: the disruption controller's owner resolution is hardcoded
for the `ReplicaSet → Deployment` path and does not generalize to arbitrary operator hierarchies.

The correct fix — walking to the outermost `/scale`-bearing controller — was validated by
@caesarxuchao (SIG API Machinery) and co-signed by @0xFelix (KubeVirt maintainer).
See https://github.com/kubernetes/kubernetes/issues/139582.

### Goals

- Walk the ownerRef chain from a pod's direct owner to the outermost controller that implements
  `/scale`, and use that controller's replica count as `expectedPods`.
- Fix `maxUnavailable` PDB behavior for KubeVirt VMPool workloads (direct owner lacks `/scale`).
- Fix double-count `expectedPods` for Argo Rollouts during rolling updates (direct owner has
  `/scale` but outermost controller is the correct source of truth).
- Leave the existing `ReplicaSet → Deployment` path unchanged in behavior.
- Gate the change behind a feature flag to allow safe rollout and rollback.

### Non-Goals

- Adding a `/scale` subresource to `VirtualMachineInstance` (singleton; semantically questionable;
  tracked separately as a KubeVirt VEP if appetite exists).
- Changing PDB behavior for workloads whose direct owner is already the outermost scalable
  controller (e.g., standalone `ReplicaSet` not owned by anything — path is unchanged).
- Supporting `minAvailable` percentage-based PDBs (those already use a different code path and
  work correctly today).

## Proposal

Generalize `getScaleController` in `pkg/controller/disruption/disruption.go` to walk the
controllerRef chain rather than only calling `/scale` on the direct owner. Simultaneously,
generalize the hardcoded Deployment skip in `getPodReplicaSet` to handle any higher-level
controller (not just `Deployment`) that owns the ReplicaSet.

The traversal always prefers the **outermost** `/scale`-bearing controller, not the first one
found. This single rule correctly handles both cases:
- VMI (no `/scale`) → continues walking → VMPool (has `/scale`) → uses VMPool replicas.
- Argo RS (has `/scale`) → continues walking → Rollout (has `/scale`) → uses Rollout replicas
  (outermost), not RS replicas (nearest), fixing the double-count.

### User Stories

#### Story 1: KubeVirt VMPool

As a cluster administrator draining a node running KubeVirt VMs managed by a `VirtualMachinePool`,
I want `maxUnavailable` PDBs to correctly limit concurrent evictions so that the pool's desired
availability is maintained during maintenance.

Today: the PDB blocks all evictions (`disruptionsAllowed: 0`) because VMI has no `/scale`.
After: the controller walks VMI → VM → VMPool, reads `VMPool.spec.replicas`, and computes
`disruptionsAllowed` correctly.

#### Story 2: Argo Rollouts

As a platform engineer using Argo Rollouts with a `maxUnavailable` PDB on my application,
I want eviction decisions to reflect the Rollout's desired replica count rather than the sum
of replicas across both active ReplicaSets during a rolling update.

Today: both RSes are counted (neither is skipped since neither is owned by a Deployment),
inflating `expectedPods`.
After: the controller walks RS → Rollout, reads `Rollout.spec.replicas` (outermost), and
computes `disruptionsAllowed` correctly.

### Notes/Constraints/Caveats

- The traversal must be bounded (depth limit, e.g. 10 hops) to protect against pathological
  or adversarial owner graphs.
- Cycle detection is required (track visited UIDs).
- The RBAC and discovery implications of walking arbitrary CRD owner kinds are addressed in
  [RBAC and Discovery](#rbac-and-discovery) below.

### Risks and Mitigations

| Risk | Mitigation |
|------|------------|
| Traversal adds latency to the disruption control loop | Depth-bounded; existing `/scale` calls already do API round-trips |
| Incorrect `expectedPods` for exotic owner graphs | Outermost-wins rule is deterministic; dedup by UID prevents double-count |
| RBAC expansion for arbitrary CRDs | Allowlist/registry approach bounds permission footprint (see below) |
| Regression for existing Deployment/RS path | Existing path preserved as fast path; behavior unchanged when direct owner is outermost scalable controller |

## Design Details

### Algorithm

The unified algorithm handles both the "direct owner lacks `/scale`" and "direct owner has
`/scale` but outermost is correct" cases:

```
func resolveExpectedScale(pod) (replicas int32, err error):
  owner := directControllerOwner(pod)
  visited := {pod.UID}
  outermost := nil

  for owner != nil && depth < maxDepth:
    if owner.UID in visited: break  // cycle protection
    visited.add(owner.UID)

    scale, err := scaleClient.Scales(owner.namespace).Get(owner.GVR, owner.name)
    if err == nil:
      outermost = scale  // do NOT stop — keep walking to find outermost

    owner = controllerOwnerOf(owner)  // walk up via ownerReferences[controller=true]

  if outermost != nil:
    return outermost.Spec.Replicas, nil
  return 0, fmt.Errorf("no scalable ancestor found")
```

Key invariants:
- **Outermost wins**: the loop does not stop at the first `/scale` success; it continues to the
  root and uses the last successful `/scale` result.
- **Direct owner lacking `/scale` is handled naturally**: the loop simply skips it and continues.
- **Dedup by UID**: prevents double-counting when multiple pods share an ancestor.
- **Depth bound**: `maxDepth = 10` (configurable at alpha).

### RBAC and Discovery

Walking arbitrary CRD ancestors requires the disruption controller to `GET` resources of kinds
it does not know about at compile time. Two approaches are under consideration:

**Option A — Allowlist/Registry (preferred for alpha):** The disruption controller only walks
ancestors whose `Group/Kind` appears in a known registry. Initially: `apps/ReplicaSet`,
`apps/Deployment`, `apps/StatefulSet`, plus a configurable extension list
(`--disruption-scalable-owner-kinds`). Operators register their CRDs (e.g., VMPool, Rollout)
via this flag or a future API. Bounds RBAC to explicitly registered kinds.

**Option B — Fully open traversal:** Walk any owner kind that responds to `/scale`. Requires
broader RBAC (`get` on any resource in any group). Simpler for operators but broader permission
footprint for a core controller.

<<[UNRESOLVED sig-apps]>>
Which option (allowlist vs fully-open) does SIG Apps prefer for alpha?
The allowlist approach is proposed as the conservative default.
<<[/UNRESOLVED]>>

### Test Plan

[ ] I/we understand the owners of the involved components may require updates to
existing tests to make this code solid enough prior to committing the changes necessary
to implement this enhancement.

##### Prerequisite testing updates

Existing `pkg/controller/disruption` unit tests must be audited to ensure they cover
the current `getPodReplicaSet` Deployment-skip logic explicitly, so regressions are caught.

##### Unit tests

- `pkg/controller/disruption`: existing coverage baseline to be recorded before changes.
- New unit tests:
  - VMPool chain: pod → VMI (no `/scale`) → VMPool (has `/scale`) → `expectedPods = VMPool.replicas`
  - Argo chain: pod → RS (has `/scale`, owned by Rollout) → Rollout (has `/scale`) →
    `expectedPods = Rollout.replicas` (outermost, not RS)
  - Deployment fast path: pod → RS → Deployment → behavior unchanged
  - Cycle detection: owner graph with a cycle terminates safely
  - Depth bound: chain longer than `maxDepth` terminates safely
  - Feature gate disabled: existing direct-owner-only behavior preserved

##### Integration tests

- PDB with `maxUnavailable` targeting pods in a mock VMPool-style hierarchy correctly
  computes `disruptionsAllowed`.
- PDB with `maxUnavailable` targeting pods in a mock Rollout-style hierarchy avoids
  double-count during rolling update.

##### e2e tests

- Node drain with a `maxUnavailable` PDB on KubeVirt VMPool pods succeeds within
  the disruption budget (requires KubeVirt installed in test cluster).

### Graduation Criteria

#### Alpha

- Feature implemented behind `PDBOwnerChainTraversal` feature gate (disabled by default)
- Unit tests covering VMPool chain, Argo chain, Deployment fast path, cycle, depth, gate off
- Integration tests in place
- Allowlist approach implemented with `--disruption-scalable-owner-kinds` flag

#### Beta

- Feature gate enabled by default
- e2e tests passing in Testgrid
- Feedback gathered from KubeVirt and Argo Rollouts communities
- RBAC/discovery approach finalized (allowlist vs open)
- Scalability benchmarks showing disruption loop latency within SLO

#### GA

- No regressions reported in beta
- Two releases of beta stability
- Conformance test added
- Documentation published at kubernetes.io

### Upgrade / Downgrade Strategy

- **Upgrade:** Feature gate is off by default at alpha; clusters upgrading see no behavior
  change until the gate is explicitly enabled.
- **Downgrade:** Disabling the gate (or downgrading) restores previous behavior. PDBs targeting
  pods with non-scalable direct owners revert to `disruptionsAllowed: 0`.

### Version Skew Strategy

- The disruption controller runs in `kube-controller-manager`. No node-level components are
  affected. No API types are added or changed.
- An n-1 `kube-controller-manager` without this feature uses existing behavior.

## Production Readiness Review Questionnaire

### Feature Enablement and Rollback

###### How can this feature be enabled / disabled in a live cluster?

- [x] Feature gate
  - Feature gate name: `PDBOwnerChainTraversal`
  - Components depending on the feature gate: `kube-controller-manager`

###### Does enabling the feature change any default behavior?

Yes — PDBs targeting pods whose direct owner lacks `/scale` but has a scalable ancestor will
now compute a non-zero `disruptionsAllowed` instead of always returning 0. PDBs targeting pods
in Argo Rollout-style hierarchies will use the outermost controller's replica count instead of
summing intermediate RSes.

###### Can the feature be disabled once it has been enabled?

Yes — disable the feature gate and restart `kube-controller-manager`. The disruption controller
reverts to direct-owner-only `/scale` resolution.

###### Are there any tests for feature enablement/disablement?

Unit tests will cover both gate=true and gate=false code paths.

### Rollout, Upgrade and Rollback Planning

###### How can a rollout or rollback fail?

If the allowlist is misconfigured or a CRD owner kind returns unexpected `/scale` responses,
`expectedPods` could be incorrect. Mitigated by depth bound and the allowlist restricting
traversal to known-safe kinds.

###### What specific metrics should inform a rollback?

- Unexpected increase in eviction failures or PDB violations.
- `disruption_controller_owner_chain_traversal_depth` histogram showing unexpectedly deep chains.

### Monitoring Requirements

###### How can an operator determine if the feature is in use by workloads?

Metric: `disruption_controller_owner_chain_traversal_depth` — a histogram recording the depth
of ownerRef chains traversed. A value > 1 indicates the new traversal path was exercised.

###### What are the reasonable SLOs for the enhancement?

- Disruption control loop reconcile latency p99 should not increase by more than 10ms per
  PDB reconcile when owner chains are ≤ 5 hops deep.

### Dependencies

###### Does this feature depend on any specific services running in the cluster?

No additional services required. The feature uses the existing scale client
(`scale.ScalesGetter`) already present in `kube-controller-manager`.

### Scalability

###### Will enabling / using this feature result in any new API calls?

Yes — additional `/scale` GET calls per ownerRef hop per pod per PDB reconcile. Bounded by
`maxDepth`. In the common case (direct owner is already the outermost scalable controller)
there are zero additional calls beyond today.

###### Will enabling / using this feature result in non-negligible increase of resource usage?

No — the traversal is bounded and the scale client already caches responses within a reconcile
cycle. Memory impact is negligible (visited UID set, bounded by maxDepth).

### Troubleshooting

###### How does this feature react if the API server and/or etcd is unavailable?

Same as today — `/scale` calls will fail, and the disruption controller will fall back to
`disruptionsAllowed: 0` for affected PDBs. No worse than current behavior.

###### What steps should be taken if SLOs are not being met?

1. Check `disruption_controller_owner_chain_traversal_depth` histogram for unexpectedly deep chains.
2. Narrow the allowlist via `--disruption-scalable-owner-kinds` to reduce traversal scope.
3. Disable the feature gate as a last resort.

## Implementation History

- 2026-06-09: Issue filed at https://github.com/kubernetes/kubernetes/issues/139582
- 2026-06-12: @0xFelix (KubeVirt maintainer) co-signed the issue
- 2026-07-08: @caesarxuchao validated "outermost controller" as the correct traversal rule;
  surfaced Argo Rollouts as a second affected downstream
- 2026-09-04: Issue bumped in #sig-apps Slack; agenda item added for Sep 14 SIG Apps meeting
- 2026-09-14: SIG Apps meeting — direction given to write KEP covering both KubeVirt VMI
  (direct owner lacks `/scale`) and general case (outermost controller) in one unified proposal
- 2026-09-14: KEP tracking issue filed at https://github.com/kubernetes/enhancements/issues/6358

## Drawbacks

- Adds API round-trips to the disruption control loop for pods with non-outermost scalable
  owners. Mitigated by depth bound; the common case has zero additional calls.
- Broader RBAC footprint if fully-open traversal is chosen. Allowlist approach mitigates this.

## Alternatives

### Option A: Add `/scale` to VirtualMachineInstance

Add `spec.replicas`, `status.replicas`, and `status.labelSelector` to the VMI type with a
webhook pinning `replicas=1`. This fixes the KubeVirt case without touching the disruption
controller. Rejected by @0xFelix as semantically questionable for a singleton resource.
Does not fix Argo Rollouts.

### Option B: Re-parent launcher pod ownerRef

Change KubeVirt's virt-controller to set the launcher pod's owner to `VirtualMachinePool`
directly. Rejected — breaks Kubernetes garbage collection semantics.

### Option C: Nearest scalable ancestor (not outermost)

Stop traversal at the first `/scale`-bearing owner rather than the outermost. Rejected because
this stops at `ReplicaSet` during Argo rolling updates (two RSes active), causing the same
double-count bug. Outermost is the correct rule as confirmed by @caesarxuchao.
