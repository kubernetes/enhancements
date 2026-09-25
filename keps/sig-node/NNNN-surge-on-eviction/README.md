# KEP-NNNN: Surge capacity on eviction instead of blocking on PodDisruptionBudget

<!-- toc -->
- [Release Signoff Checklist](#release-signoff-checklist)
- [Summary](#summary)
- [Motivation](#motivation)
  - [Goals](#goals)
  - [Non-Goals](#non-goals)
- [Proposal](#proposal)
  - [User Stories](#user-stories)
    - [Story 1: Single-replica Deployment on a drained node](#story-1-single-replica-deployment-on-a-drained-node)
    - [Story 2: Tightly-budgeted service](#story-2-tightly-budgeted-service)
  - [Notes/Constraints/Caveats](#notesconstraintscaveats)
  - [Risks and Mitigations](#risks-and-mitigations)
- [Design Details](#design-details)
  - [The three cooperating pieces](#the-three-cooperating-pieces)
  - [Open design questions](#open-design-questions)
  - [Test Plan](#test-plan)
  - [Graduation Criteria](#graduation-criteria)
  - [Upgrade / Downgrade Strategy](#upgrade--downgrade-strategy)
  - [Version Skew Strategy](#version-skew-strategy)
- [Production Readiness Review Questionnaire](#production-readiness-review-questionnaire)
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
- [ ] (R) Test plan is in place, giving consideration to SIG Architecture and
      SIG Testing input (including test refactors)
- [ ] (R) Graduation criteria is in place
- [ ] (R) Production readiness review completed
- [ ] (R) Production readiness review approved
- [ ] "Implementation History" section is up-to-date for milestone
- [ ] User-facing documentation has been created in [kubernetes/website]
- [ ] Supporting e2e tests documented in the KEP

[kubernetes/enhancements]: https://git.k8s.io/enhancements
[kubernetes/website]: https://git.k8s.io/website

## Summary

When Kubernetes evicts a pod through the API-initiated eviction path, the
eviction is a binary decision against the pod's PodDisruptionBudget (PDB):
either the budget allows a disruption and the pod is deleted, or the budget is
exhausted and the eviction is rejected with `429 TooManyRequests`. There is no
middle ground.

For workloads with no spare capacity relative to their budget — most obviously a
single-replica Deployment with `minAvailable: 1` — this means eviction is
blocked indefinitely. The evictor (node drain, cluster-autoscaler, etc.) is
forced to either wait forever or force-delete the pod, causing an availability
gap until the workload controller schedules a replacement.

This KEP proposes an opt-in behavior where, instead of blocking on an exhausted
budget, the system surges a replacement pod first (temporarily exceeding the
desired replica count, similar to `maxSurge` during a rolling update) and only
completes the eviction once the replacement is Ready — preserving availability
during voluntary disruptions.

## Motivation

Rolling updates already solve the "don't drop below capacity" problem via
`maxSurge`: a new pod comes up before the old one goes away. Voluntary
disruption via eviction offers no equivalent. As a result, the safest budget a
user can express (`minAvailable == replicas`) is also the one that makes their
workload undrainable without an outage. Users are pushed toward either
over-provisioning replicas permanently or accepting downtime during routine
operations like node maintenance.

### Goals

- Allow eviction of a pod whose PDB is currently exhausted to proceed *without*
  an availability drop, by surging a replacement first and releasing the
  eviction once the replacement is Ready.
- Make the behavior opt-in and safe-by-default (existing blocking behavior is
  unchanged unless the feature is enabled and requested).
- Define a clear, well-understood fallback for workloads that cannot surge.

### Non-Goals

- Changing the default eviction semantics for existing PDBs.
- Surging workloads that are not managed by a controller capable of temporary
  over-provisioning (e.g. bare/unmanaged pods, singleton StatefulSets with
  RWO volumes).
- Replacing or superseding [KEP-4212 Declarative Node Maintenance]; this
  proposal is expected to compose with it.
- Autoscaling decisions unrelated to disruption (HPA/VPA behavior is out of
  scope).

[KEP-4212 Declarative Node Maintenance]: /keps/sig-node/4212-declarative-node-maintenance

## Proposal

### User Stories

#### Story 1: Single-replica Deployment on a drained node

A platform operator runs a single-replica Deployment with a PDB of
`minAvailable: 1`. During routine node maintenance they run `kubectl drain`.
Today the eviction is rejected forever because `DisruptionsAllowed == 0`. With
surge-on-eviction enabled, the Deployment controller schedules a second pod on
another node, and once it is Ready the original pod is evicted — the drain
completes with zero downtime.

#### Story 2: Tightly-budgeted service

A latency-sensitive service runs with `replicas: 3` and `minAvailable: 3`
because it cannot tolerate losing a replica. Cluster-autoscaler needs to remove
an underutilized node hosting one of its pods. Instead of the scale-down being
blocked, the service surges a fourth pod, the third pod is evicted once the
fourth is Ready, and the service never drops below three healthy replicas.

### Notes/Constraints/Caveats

- Surging requires schedulable spare capacity in the cluster. Interaction with
  cluster-autoscaler (which may itself need to add a node before the surge can
  schedule) must be defined.
- API-initiated eviction is synchronous today; a surge can take a
  non-trivial amount of time. This proposal turns the exhausted-budget case into
  an asynchronous contract: the evictor keeps retrying and receives `429` with a
  distinct reason (e.g. "surging") until the replacement is Ready.
- Not every workload can safely run an extra replica (RWO volumes, leader-elected
  singletons, ordered StatefulSets). These must fall back to today's blocking
  behavior.

### Risks and Mitigations

- **Runaway surge / cost.** A misbehaving evictor could trigger repeated surges.
  Mitigation: surge is bounded (at most the pods being drained), opt-in, and
  reconciled down once the eviction completes.
- **Stuck surge.** If a surged pod never becomes Ready (no capacity, crash
  loop), the eviction must not complete and the surge must be reclaimed after a
  timeout, leaving the workload in its original state.
- **Semantic confusion with PDBs.** PDBs are intentionally decoupled from
  workload controllers. Overloading them with surge intent risks coupling them.
  The design must decide carefully where surge intent lives (see open questions).

## Design Details

### The three cooperating pieces

A surge-on-eviction flow needs three components to cooperate, none of which
exists end-to-end today:

1. **Signal.** A durable record that "an eviction is pending on pod X, please
   surge." Today a blocked eviction returns a `429` that carries no intent the
   workload controller can observe or act on.
2. **Surge.** The owning controller (Deployment/ReplicaSet/StatefulSet) must be
   willing to temporarily run above its desired replica count *for the purpose
   of draining*, then scale back once the old pod is gone — analogous to
   `maxSurge`, but driven by disruption rather than a spec change.
3. **Release.** The eviction only completes after the surged pod is Ready, so
   observed availability never dips below the budget.

For reference, the current blocking decision is made in the eviction subresource
(`checkAndDecrement` in
`pkg/registry/core/pod/storage/eviction.go`), and `DisruptionsAllowed` is
computed as `currentHealthy - desiredHealthy` by the disruption controller
(`updatePdbStatus` in `pkg/controller/disruption/disruption.go`).

### Open design questions

These are the questions the SIGs need to resolve before this KEP can move to
`implementable`:

<<[UNRESOLVED signal-location]>>
Where does surge intent live? Candidates: a new field/condition on the PDB, a
field/annotation on the Pod, an option on the Eviction subresource, or a
dedicated object. Overloading the PDB is convenient but couples PDBs to workload
controllers, which are intentionally decoupled today.
<<[/UNRESOLVED]>>

<<[UNRESOLVED surge-owner]>>
Who performs the surge? The disruption controller does not manage replica
counts; workload controllers do. This needs a contract so a workload controller
can distinguish "drain-driven surge" from user-driven scaling and from HPA.
<<[/UNRESOLVED]>>

<<[UNRESOLVED non-surgeable]>>
Exact fallback semantics for workloads that cannot surge (singleton
StatefulSets, RWO volumes, unmanaged pods). Block as today, or reject the
surge request explicitly?
<<[/UNRESOLVED]>>

<<[UNRESOLVED eviction-contract]>>
The precise asynchronous contract returned to evictors while a surge is in
flight, and how it interacts with existing drain/eviction timeouts and with
[KEP-4212 Declarative Node Maintenance].
<<[/UNRESOLVED]>>

### Test Plan

[ ] I/we understand the owners of the involved components may require updates to
existing tests to make this code solid enough prior to committing the changes
necessary to implement this enhancement.

Detailed test plan to be added once the design reaches `implementable`. At a
minimum it will cover: unit tests for the eviction subresource decision and the
workload-controller surge/reconcile logic; integration tests for the
end-to-end surge → release flow; and e2e tests draining a single-replica
workload with zero observed downtime.

### Graduation Criteria

To be defined once the design is accepted. Expected shape:

- **Alpha:** feature gate `SurgeOnEviction` (default off); behavior limited to
  Deployment/ReplicaSet-owned pods; unit and integration coverage.
- **Beta:** default off pending feedback; e2e coverage; scalability assessment;
  StatefulSet support decided.
- **GA:** production feedback; no open PRR concerns.

### Upgrade / Downgrade Strategy

The feature is gated and opt-in. Disabling the gate reverts to today's blocking
eviction behavior; any in-flight surge reconciles back to the desired replica
count. Details TBD.

### Version Skew Strategy

To be defined. The primary concern is skew between the kube-apiserver (which
makes the eviction decision) and the kube-controller-manager (which performs the
surge).

## Production Readiness Review Questionnaire

To be completed prior to targeting an alpha milestone. This KEP is currently
`provisional`; the PRR will be filled in once a SIG accepts the design.

## Implementation History

- 2026-07-16: Provisional KEP drafted, based on discussion in
  kubernetes/kubernetes issue for surge-on-eviction behavior.

## Drawbacks

- Adds complexity to the eviction path and to workload controllers.
- Only helps workloads that can tolerate a temporary extra replica.
- Requires spare cluster capacity to be effective.

## Alternatives

- **Do nothing / recommend more replicas.** Users permanently over-provision to
  keep a non-zero budget. Wastes resources and does not help true singletons.
- **Force-delete on drain.** Already possible, but reintroduces the outage this
  KEP aims to avoid.
- **Fold entirely into [KEP-4212 Declarative Node Maintenance].** Possible; this
  KEP explicitly calls out that it should compose with 4212, and the SIGs may
  decide the surge behavior belongs there rather than as a standalone KEP.
