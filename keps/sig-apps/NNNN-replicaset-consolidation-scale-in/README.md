# KEP-NNNN: ReplicaSet Consolidation-Aware Scale-In Strategy

<!-- toc -->
- [Release Signoff Checklist](#release-signoff-checklist)
- [Summary](#summary)
- [Motivation](#motivation)
  - [Goals](#goals)
  - [Non-Goals](#non-goals)
- [Proposal](#proposal)
  - [User Stories](#user-stories)
    - [Story 1: HPA-Driven Consolidation](#story-1-hpa-driven-consolidation)
    - [Story 2: Respecting Do-Not-Disrupt Signals](#story-2-respecting-do-not-disrupt-signals)
    - [Story 3: Cost Optimization During Off-Peak](#story-3-cost-optimization-during-off-peak)
  - [Risks and Mitigations](#risks-and-mitigations)
- [Design Details](#design-details)
  - [Feature Gate](#feature-gate)
  - [Modified Pod Deletion Ranking](#modified-pod-deletion-ranking)
  - [Node Informer Integration](#node-informer-integration)
  - [Do-Not-Disrupt Annotation Handling](#do-not-disrupt-annotation-handling)
  - [Interaction with Existing Scale-Down Logic](#interaction-with-existing-scale-down-logic)
  - [Test Plan](#test-plan)
    - [Unit Tests](#unit-tests)
    - [Integration Tests](#integration-tests)
    - [End-to-End Tests](#end-to-end-tests)
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
  - [Extend PodDeletionCost (KEP-2255)](#extend-poddeletioncost-kep-2255)
  - [Pluggable Scale-Down Framework](#pluggable-scale-down-framework)
  - [Webhook-Based Pod Selection](#webhook-based-pod-selection)
- [Infrastructure Needed](#infrastructure-needed)
<!-- /toc -->

## Release Signoff Checklist

Items marked with (R) are required *prior to targeting to a milestone / release*.

- [ ] (R) Enhancement issue in release milestone, which links to KEP dir in [kubernetes/enhancements] (not the initial KEP PR)
- [ ] (R) KEP approvers have approved the KEP status as `implementable`
- [ ] (R) Design details are appropriately documented
- [ ] (R) Test plan is in place, giving consideration to SIG Architecture and SIG Testing input (including test refactors)
  - [ ] e2e Tests for all Beta API Operations (CLI, key scenarios, etc.)
  - [ ] (R) Ensure GA://Ensure GA e2e tests meet requirements for [Coverage Onboarding Checklist](https://github.com/kubernetes/community/blob/master/sig-testing/coverage-onboarding-checklist.md)
- [ ] (R) Graduation criteria is in place
  - [ ] (R) [all GA Coverage Onboarding Checklist items](https://github.com/kubernetes/community/blob/master/sig-testing/coverage-onboarding-checklist.md) are met
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

This KEP introduces an opt-in consolidation-aware heuristic to the ReplicaSet
controller's scale-down pod selection algorithm. When the `ConsolidatingScaleDown`
feature gate is enabled, the controller prefers deleting pods on nodes with fewer
total active pods during scale-down, enabling workload consolidation onto fewer
nodes. It also respects do-not-disrupt signals so that pods on protected nodes
are deprioritized for deletion.

## Motivation

The current ReplicaSet scale-down algorithm prefers deleting pods on nodes with
*more* colocated replicas of the same ReplicaSet (a spreading heuristic). While
this promotes even distribution, it actively works against node consolidation.

Node autoscalers such as Karpenter and cluster-autoscaler can reclaim empty or
underutilized nodes, but only when workloads consolidate during scale-down. When
an HPA scales a Deployment from 20 replicas to 10, the current spreading
heuristic distributes deletions evenly across nodes, leaving every node partially
occupied and preventing any node from being reclaimed.

KEP-2255 (PodDeletionCost) provides a mechanism for influencing pod deletion
order via annotations, but it requires an external controller to continuously
update annotations before scale-down events occur. This is operationally complex,
does not integrate with HPA-driven scale-down, and continuously updating
annotations places unnecessary load on the API server.

### Goals

- Provide a feature-gated, consolidation-aware pod deletion heuristic for
  ReplicaSet scale-down that prefers removing pods from nodes with fewer total
  active pods.
- Respect do-not-disrupt signals from node autoscalers via a generic annotation,
  deprioritizing deletion of pods on protected nodes.
- Maintain full backward compatibility: when the feature gate is disabled,
  behavior is identical to the existing spreading heuristic.
- Complement (not replace) KEP-2255 PodDeletionCost — both mechanisms can
  coexist, with PodDeletionCost taking precedence in the existing sort order.

### Non-Goals

- Replace or deprecate KEP-2255 (PodDeletionCost).
- Add new API fields to ReplicaSet, Deployment, or any other workload spec.
- Implement a general-purpose pluggable scale-down framework (see
  [Alternatives](#pluggable-scale-down-framework)).
- Provide resource-aware bin-packing (future enhancement, not in scope for alpha).
- Modify scale-up behavior.

## Proposal

### User Stories

#### Story 1: HPA-Driven Consolidation

As a platform engineer running workloads with HPA on Karpenter-managed nodes, I
want scale-down events to consolidate pods onto fewer nodes so that Karpenter can
reclaim empty nodes and reduce my cloud spend, without requiring me to deploy and
maintain a sidecar controller that manages PodDeletionCost annotations.

#### Story 2: Respecting Do-Not-Disrupt Signals

As a platform engineer, I have nodes running long-lived batch jobs annotated with
do-not-disrupt. When my web-tier Deployment scales down, I want the ReplicaSet
controller to avoid deleting pods from those protected nodes, even if they have
fewer total pods, so that the batch jobs are not indirectly disrupted by
rescheduling churn.

#### Story 3: Cost Optimization During Off-Peak

As a cost-conscious operator, I run a Deployment that scales from 50 replicas
during peak to 10 replicas off-peak. I want the scale-down to preferentially
empty out nodes so that my cluster autoscaler can remove 8 nodes instead of
leaving 20 nodes each running a single pod.

### Risks and Mitigations

**Risk: Coupling to Karpenter-specific annotation.**
The initial implementation checks `karpenter.sh/do-not-disrupt`. This couples
core Kubernetes to a specific autoscaler's annotation.

*Mitigation:* We propose introducing a generic, Kubernetes-native annotation
`controller.kubernetes.io/do-not-disrupt` that any autoscaler or operator can
set. The implementation will check both the generic annotation and the
Karpenter-specific annotation during alpha, with the Karpenter-specific
annotation deprecated in beta. This gives the ecosystem time to migrate.

**Risk: Conflict with pod topology spread constraints.**
The consolidation heuristic may concentrate pods on fewer nodes, potentially
violating `topologySpreadConstraints` configured on the workload.

*Mitigation:* Pod topology spread constraints are enforced at scheduling time,
not at deletion time. When pods are deleted and rescheduled, the scheduler
enforces topology spread. However, during scale-down (net pod reduction), no
rescheduling occurs — pods are simply removed. The consolidation heuristic
affects *which* pods are removed, not where new pods are placed. If a user has
topology spread constraints, the remaining pods may temporarily violate the
desired spread until the next scale-up event. This is the same behavior as the
current spreading heuristic (which also does not guarantee topology spread
compliance during scale-down). We will document this interaction clearly.

**Risk: Unexpected behavior change for existing workloads.**
Users who depend on the current spreading behavior may be surprised if they
enable the feature gate.

*Mitigation:* The feature gate is disabled by default in alpha. Users must
explicitly opt in. Documentation will clearly describe the behavioral change.

## Design Details

### Feature Gate

- Name: `ConsolidatingScaleDown`
- Component: `kube-controller-manager`
- Default: `false` (alpha)
- Disable-supported: `true`

When disabled, the ReplicaSet controller uses the existing spreading heuristic
with no behavioral change.

### Modified Pod Deletion Ranking

The ReplicaSet controller's `ActivePodsWithRanks.Less()` function determines pod
deletion order during scale-down. The existing sort order is:

1. Unassigned pods (no node) before assigned pods
2. Pending pods before running pods
3. Not-ready pods before ready pods
4. Pods with lower deletion cost (KEP-2255) before higher cost
5. **Doubled-up pods before non-doubled-up** (spreading heuristic)
6. Younger pods before older pods

When `ConsolidatingScaleDown` is enabled, steps 4.5 and 5 are modified:

1. Unassigned pods before assigned pods
2. Pending pods before running pods
3. Not-ready pods before ready pods
4. Pods with lower deletion cost before higher cost
5. **[NEW] Pods on non-protected nodes before pods on do-not-disrupt nodes**
6. **[CHANGED] Pods on nodes with fewer total active pods before pods on nodes
   with more total active pods** (consolidation heuristic — inverted from
   spreading)
7. Younger pods before older pods

The rank for each pod is computed by counting all active pods on the same node
(across all namespaces and controllers, not just the current ReplicaSet). This
provides a global view of node utilization for the consolidation decision.

### Node Informer Integration

When `ConsolidatingScaleDown` is enabled, the ReplicaSet controller initializes
a node informer via the shared informer factory. This is used for:

- Validating node existence (future use)
- Future resource-based scoring (not in alpha scope)

The node informer is conditionally initialized — when the feature gate is
disabled, no node informer is created and there is zero additional overhead.

The pod-per-node counting uses the existing pod indexer
(`PodNodeNameKeyIndex`) rather than the node informer, so the node informer
adds minimal memory overhead in alpha.

### Do-Not-Disrupt Annotation Handling

When computing disruption cost ranks, the controller checks whether any active
pod on each candidate node carries a do-not-disrupt annotation. If so, all
candidate pods on that node are deprioritized for deletion.

**Alpha behavior:** Checks `karpenter.sh/do-not-disrupt: "true"`.

**Planned beta behavior:** Checks both `controller.kubernetes.io/do-not-disrupt: "true"`
(new generic annotation) and `karpenter.sh/do-not-disrupt: "true"` (deprecated
but still honored).

**Planned GA behavior:** Only checks `controller.kubernetes.io/do-not-disrupt: "true"`.
The Karpenter-specific annotation is no longer checked by the controller.

### Interaction with Existing Scale-Down Logic

The consolidation heuristic modifies only the ranking step of pod deletion
selection. All other aspects of scale-down remain unchanged:

- **PodDeletionCost (KEP-2255):** Evaluated at step 4, before the consolidation
  rank at step 6. PodDeletionCost takes precedence over the consolidation
  heuristic.
- **Pod phase and readiness:** Steps 1-3 are unchanged. Unassigned, pending, and
  not-ready pods are still preferred for deletion regardless of consolidation.
- **Pod age tiebreaker:** Step 7 is unchanged. Among pods with equal
  consolidation rank, younger pods are still preferred.
- **Burst deletion:** The `BurstReplicas` limit on concurrent deletions is
  unchanged.

### Test Plan

[x] I/we understand the owners of the involved components may require updates to
existing tests to make this code solid enough prior to committing the changes necessary
to implement this enhancement.

#### Unit Tests

- `pkg/controller/controller_utils_test.go`: Tests for `ActivePodsWithRanks.Less()`
  with `ConsolidatingScaleDown` enabled and disabled, covering:
  - Consolidation rank ordering (fewer pods on node = higher deletion priority)
  - Do-not-disrupt deprioritization
  - Interaction with PodDeletionCost
  - Swap correctness for DoNotDisrupt slice
- `pkg/controller/replicaset/replica_set_test.go`: Tests for
  `getPodsRankedByNodeDisruptionCost()` covering:
  - Correct pod-per-node counting via indexer
  - Do-not-disrupt annotation detection
  - Unassigned pods (empty NodeName)
  - Feature gate toggle behavior

Coverage targets:
- `pkg/controller/controller_utils.go`: target 85%+
- `pkg/controller/replicaset/replica_set.go`: target 80%+

#### Integration Tests

- `test/integration/replicaset/replicaset_test.go`: Integration test verifying
  that with `ConsolidatingScaleDown` enabled, scale-down of a ReplicaSet
  preferentially removes pods from nodes with fewer total pods.

#### End-to-End Tests

- E2e test (beta requirement) verifying consolidation behavior in a multi-node
  cluster with HPA-driven scale-down.

### Graduation Criteria

#### Alpha

- Feature gate `ConsolidatingScaleDown` implemented and disabled by default
- Unit tests for consolidation ranking and do-not-disrupt logic
- Integration tests for basic consolidation behavior
- KEP at `implementable` status
- PRR approval

#### Beta

- Address feedback from alpha users
- Introduce generic `controller.kubernetes.io/do-not-disrupt` annotation
- Add metrics for consolidation-aware deletions (see [Monitoring Requirements](#monitoring-requirements))
- E2e tests
- Documentation on kubernetes.io
- PRR re-review

#### GA

- Deprecate and remove Karpenter-specific annotation check
- Conformance tests
- 2-week flake-free test window
- All known bugs fixed
- Lock feature gate to enabled

### Upgrade / Downgrade Strategy

**Upgrade:** Enabling the feature gate changes scale-down pod selection order.
No data migration is required. The change takes effect on the next scale-down
event after the kube-controller-manager restarts with the gate enabled.

**Downgrade:** Disabling the feature gate reverts to the spreading heuristic.
No cleanup is required. Pods previously deleted under the consolidation heuristic
are already gone; the change only affects future scale-down decisions.

### Version Skew Strategy

The feature is entirely within the kube-controller-manager. There is no
version skew concern between control plane components because only one instance
of the ReplicaSet controller runs at a time (leader election). Kubelets and
API servers are unaffected.

## Production Readiness Review Questionnaire

### Feature Enablement and Rollback

###### How can this feature be enabled / disabled in a live cluster?

- [x] Feature gate
  - Feature gate name: `ConsolidatingScaleDown`
  - Components depending on the feature gate: `kube-controller-manager`

###### Does enabling the feature change any default behavior?

Yes. When enabled, ReplicaSet scale-down prefers deleting pods on nodes with
fewer total active pods (consolidation) instead of pods on nodes with more
colocated replicas (spreading). Pods on nodes with do-not-disrupt annotations
are deprioritized for deletion.

###### Can the feature be disabled once it has been enabled (i.e. can we roll back the enablement)?

Yes. Disabling the feature gate and restarting kube-controller-manager reverts
to the spreading heuristic. No state cleanup is required.

###### What happens if we reenable the feature if it was previously rolled back?

The consolidation heuristic resumes on the next scale-down event. No state
is persisted between enablements.

###### Are there any tests for feature enablement/disablement?

Yes. Unit tests verify that `ActivePodsWithRanks.Less()` behaves correctly
with the feature gate both enabled and disabled. Integration tests verify
end-to-end scale-down behavior under both configurations.

### Rollout, Upgrade and Rollback Planning

###### How can a rollout or rollback fail? Can it impact already running workloads?

A rollout cannot fail in a way that impacts running workloads. The feature only
affects the *order* in which pods are selected for deletion during scale-down.
If the feature gate fails to enable (e.g., typo in gate name), the existing
spreading heuristic is used. Running pods are never affected — only future
scale-down decisions change.

###### What specific metrics should inform a rollback?

- Unexpected increase in pod churn or rescheduling events
- Cluster autoscaler unable to consolidate nodes (indicating the heuristic is
  not working as expected)
- Increase in topology spread constraint violations reported by users

###### Were upgrade and rollback tested? Was the upgrade->downgrade->upgrade path tested?

Will be tested during alpha. The feature is stateless — toggling the gate
and restarting kube-controller-manager is sufficient.

###### Is the rollout accompanied by any deprecations and/or removals of features, APIs, machines, permissions, or service account bindings?

No.

### Monitoring Requirements

###### How can an operator determine if the feature is in use by workloads?

- Check if the `ConsolidatingScaleDown` feature gate is enabled on
  kube-controller-manager.
- (Beta) A new metric `replicaset_consolidation_scale_down_total` will count
  the number of scale-down events that used the consolidation heuristic.
- (Beta) A new metric `replicaset_do_not_disrupt_deprioritizations_total` will
  count pods deprioritized due to do-not-disrupt annotations.

###### How can someone using this feature know that it is working for their instance?

- Observe that scale-down events preferentially remove pods from nodes with
  fewer total pods (visible via `kubectl get pods -o wide` before and after
  scale-down).
- Observe that cluster autoscaler or Karpenter reclaims nodes after scale-down
  events (node count decreases).
- (Beta) Check the `replicaset_consolidation_scale_down_total` metric.

###### What are the SLIs (Service Level Indicators) an operator can use to determine the health of the service?

- [x] Metrics
  - Metric name: `replicaset_consolidation_scale_down_total` (beta)
  - Components exposing the metric: `kube-controller-manager`
- [x] Other
  - Node count reduction after scale-down events (observable via cluster
    autoscaler metrics or cloud provider billing)

###### Are there any missing metrics that would be useful to have in this category?

- Per-node pod density distribution after scale-down (future consideration)
- Consolidation efficiency ratio (pods removed from eventually-empty nodes vs
  total pods removed)

###### What are the reasonable SLOs (Service Level Objectives) for the enhancement?

- Scale-down latency should not increase by more than 5% compared to the
  spreading heuristic (the additional pod-per-node counting is O(N) where N is
  the number of candidate pods).
- No increase in ReplicaSet controller error rate.

###### What are the failure modes for the enhancement?

| Failure Mode | Impact | Detection | Mitigation |
|---|---|---|---|
| Pod indexer returns error for node lookup | Falls back to rank 0 for affected pods (treated as empty node) | Controller error logs | Self-healing: next sync cycle retries |
| Node informer fails to sync | `nodeListerSynced` returns false; controller waits for sync | kube-controller-manager startup logs | Restart kube-controller-manager |
| Do-not-disrupt annotation on all nodes | All pods deprioritized equally; falls through to age-based tiebreaker | No pods consolidated | Expected behavior — no mitigation needed |

###### What steps should be taken if SLOs are not being met to determine the problem?

1. Check kube-controller-manager logs for errors related to the ReplicaSet
   controller or node informer.
2. Verify the feature gate is enabled: check kube-controller-manager flags.
3. Check pod indexer health: verify pods are indexed by node name.
4. Disable the feature gate and restart kube-controller-manager to revert to
   spreading heuristic.

### Dependencies

###### Does this feature depend on any specific services running in the cluster?

No. The feature uses only the existing pod informer/indexer and optionally the
node informer, both of which are part of the standard kube-controller-manager
informer factory.

### Scalability

###### Will enabling / using this feature result in any new API calls?

No new API calls. The consolidation ranking uses the existing pod indexer
(in-memory) to count pods per node. The node informer (when enabled) uses the
standard shared informer factory LIST/WATCH, which is already used by other
controllers.

###### Will enabling / using this feature result in introducing new API types?

No.

###### Will enabling / using this feature result in any new calls to the cloud provider?

No.

###### Will enabling / using this feature result in increasing size or count of the existing API objects?

No. No new fields are added to any API objects. The do-not-disrupt annotation
is read-only from the controller's perspective.

###### Will enabling / using this feature result in increasing time taken by any operations?

The pod deletion ranking computation adds an O(N) pass over candidate pods to
count pods per node via the indexer, where N is the number of candidate pods
for deletion. For typical ReplicaSet sizes (tens to hundreds of pods), this
adds negligible latency (microseconds).

###### Will enabling / using this feature result in non-negligible increase of resource usage?

- **Memory:** The node informer adds memory proportional to the number of nodes
  in the cluster. For a 5,000-node cluster, this is approximately 5-10 MB
  additional memory in kube-controller-manager. The `nodePodCounts` and
  `nodeHasDoNotDisrupt` maps are ephemeral (allocated per scale-down event and
  garbage collected).
- **CPU:** Negligible. The pod-per-node counting is O(N) per scale-down event.

###### Can enabling / using this feature result in resource exhaustion of some node resources?

No. The feature only affects which pods are deleted during scale-down. It does
not affect scheduling, resource requests, or resource limits.

###### Will enabling / using this feature result in any new resource claim usage?

No.

### Troubleshooting

###### How does this feature react if the API server and/or etcd is unavailable?

The feature does not make additional API calls. If the API server is unavailable,
the ReplicaSet controller cannot perform any scale-down operations regardless of
this feature. The pod indexer and node informer operate on cached data.

###### What are other known failure modes?

See [failure modes table](#what-are-the-failure-modes-for-the-enhancement) above.

###### What steps should be taken if SLOs are not being met to determine the problem?

See [SLO troubleshooting](#what-steps-should-be-taken-if-slos-are-not-being-met-to-determine-the-problem) above.

## Implementation History

- 2026-03-27: Initial KEP draft (provisional)
- `consolidation-strategy` branch: Reference implementation with feature gate,
  modified ranking logic, node informer integration, and unit/integration tests

## Drawbacks

- **Increased complexity in pod deletion ordering.** The sort comparator gains
  additional conditional branches, making it harder to reason about deletion
  order. Mitigated by clear documentation and comprehensive tests.

- **Node informer memory overhead.** Even though the alpha implementation
  primarily uses the pod indexer, the node informer is initialized when the gate
  is enabled. In very large clusters (5,000+ nodes), this adds non-trivial
  memory. Mitigated by conditional initialization (no overhead when gate is off).

- **Potential for uneven pod distribution.** The consolidation heuristic
  intentionally concentrates pods on fewer nodes, which may conflict with
  availability goals. Users who want both consolidation and spreading must
  rely on pod topology spread constraints at scheduling time.

## Alternatives

### Extend PodDeletionCost (KEP-2255)

Instead of a new heuristic, extend PodDeletionCost with an automated controller
that sets annotations based on node utilization.

**Why not chosen:** This requires deploying and maintaining an external
controller, does not integrate with HPA-driven scale-down (annotations must be
set *before* the scale-down event), and continuously updating annotations places
load on the API server. Previous proposals along these lines (#107598, #123541)
were closed as stale without implementation.

### Pluggable Scale-Down Framework

Add a `scaleConfig` field to ReplicaSetSpec defining a sequence of sorting
methods for scale-down pod selection (as proposed in #107598).

**Why not chosen:** Too broad in scope. Previous attempts at pluggable frameworks
were closed without implementation. A focused, single-heuristic approach behind a
feature gate is more likely to gain SIG approval and be maintainable long-term.
If the community later wants pluggability, this KEP's consolidation heuristic
can become one option in that framework.

### Webhook-Based Pod Selection

Add a webhook extension point that allows external systems to influence or
override pod deletion selection.

**Why not chosen:** Adds latency to the scale-down path, introduces a new
failure mode (webhook unavailability), and significantly increases complexity.
The in-tree heuristic approach is simpler and more reliable.

## Infrastructure Needed

None. The feature is entirely within the existing kube-controller-manager binary
and uses existing informer infrastructure.
