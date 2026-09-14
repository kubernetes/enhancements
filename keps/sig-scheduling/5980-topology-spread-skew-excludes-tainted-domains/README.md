# KEP-5980: TopologySpreadConstraint Skew Excludes Fully-Tainted Domains

<!-- toc -->
- [Release Signoff Checklist](#release-signoff-checklist)
- [Summary](#summary)
- [Motivation](#motivation)
  - [Goals](#goals)
  - [Non-Goals](#non-goals)
- [Proposal](#proposal)
  - [User Stories](#user-stories)
  - [Risks and Mitigations](#risks-and-mitigations)
- [Design Details](#design-details)
  - [Semantics](#semantics)
  - [Implementation Details](#implementation-details)
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

## Release Signoff Checklist

Items marked with (R) are required *prior to targeting to a milestone / release*.

- [ ] (R) Enhancement issue in release milestone, which links to KEP dir in [kubernetes/enhancements] (not the initial KEP PR)
- [ ] (R) KEP approvers have approved the KEP status as `implementable`
- [ ] (R) Design details are appropriately documented
- [ ] (R) Test plan is in place, giving consideration to SIG Architecture and SIG Testing input (including test refactors)
- [ ] (R) Graduation criteria is in place
- [ ] (R) Production readiness review completed
- [ ] (R) Production readiness review approved
- [ ] "Implementation History" section is up-to-date for milestone
- [ ] User-facing documentation has been created in [kubernetes/website], for publication to [kubernetes.io]
- [ ] Supporting documentation—e.g., additional design documents, links to mailing list discussions/SIG meetings, relevant PRs/issues, release notes

## Summary

This KEP modifies the kube-scheduler's `PodTopologySpread` plugin so that when `nodeTaintsPolicy` is set to `Honor`, topology domains whose nodes are **all** tainted with taints not tolerated by the incoming pod—but which already have matching replicas bound—continue to satisfy the `minDomains` requirement while being excluded from the skew calculation.

This allows pods with strict TopologySpreadConstraints to continue scheduling during complete topology domain outages (e.g., zonal failures) without requiring any new user configuration.

## Motivation

Users configure TopologySpreadConstraints (TSCs) with `minDomains` to ensure their workloads are horizontally available across topology domains. A typical high-availability configuration looks like:

```yaml
topologySpreadConstraints:
  - maxSkew: 1
    topologyKey: topology.kubernetes.io/zone
    whenUnsatisfiable: DoNotSchedule
    minDomains: 3
    nodeTaintsPolicy: Honor
```

During normal operations this achieves the desired spread. However, during a complete zonal outage—when all nodes in a zone are tainted (e.g., `node.kubernetes.io/unreachable`)—this configuration prevents new pods from scheduling anywhere:

- The tainted zone's nodes are excluded from consideration (due to `nodeTaintsPolicy: Honor`), reducing the visible domain count below `minDomains`.
- When `domainsNum < minDomains`, the current implementation sets `minMatchNum = 0`, which makes the skew calculation `matchNum + selfMatchNum - 0` for all candidate nodes. Combined with the existing pods in healthy zones, new pods quickly violate `maxSkew` relative to what the scheduler perceives.

**Concrete example:**

Starting state with 3 replicas distributed [1, 1, 1] across zones A, B, C:

```
+--------+--------+--------+
| Zone A | Zone B | Zone C |
+--------+--------+--------+
|   P    |   P    |   P    |
+--------+--------+--------+
```

Zone C experiences a catastrophic failure. All nodes in Zone C are tainted as unreachable. An HPA scales the deployment from 3 to 9 replicas. The scheduler can only place 2 additional pods (one per healthy zone), reaching [2, 2, 1]. The remaining 4 pods are stuck Pending.

The maximum number of additional pods the scheduler can place during such an outage is bounded by:

```
additional pods ≤ (minDomains - 1) × maxSkew
```

At production scale (minDomains=3, maxSkew=1), this means only **2** additional pods can ever be scheduled during an outage—regardless of how many replicas are requested. This fundamentally breaks pod autoscaling during the exact scenarios where scaling out is most critical.

### Goals

- Allow pods with strict TSCs (`minDomains > 1`, `maxSkew` low, `whenUnsatisfiable: DoNotSchedule`, `nodeTaintsPolicy: Honor`) to continue scheduling to healthy topology domains during complete domain outages.
- Achieve this without introducing new API fields or requiring user configuration changes.

### Non-Goals

- Solving autoscaler/scheduler communication for cases where existing nodes are healthy but new nodes cannot be provisioned (e.g., capacity stockouts).
- Handling partial domain failures where some nodes in a domain are healthy and others are tainted.
- Modifying behavior when `nodeTaintsPolicy` is `Ignore` (the default).

## Proposal

When `nodeTaintsPolicy` is set to `Honor`, if a topology domain meets **both** of the following conditions:

1. All of the domain's nodes have taints that the incoming pod does not tolerate.
2. The domain has at least one existing pod that matches the constraint's label selector.

Then that domain:
- **Counts toward `minDomains`** — it is considered an eligible domain for satisfying the minimum domain requirement.
- **Is excluded from skew calculation** — its pod count is not included in `TpValueToMatchNum`, so it does not constrain the placement of new pods in healthy domains.

From the example above, with Zone C fully tainted but containing one existing replica, the scheduler would see:
- Eligible domains for minDomains: 3 (A + B + C) — constraint satisfied
- Domains participating in skew: 2 (A + B only)
- Result: pods can schedule up to [n, n ± maxSkew, 1] — autoscaling is unblocked

This approach requires that the topology domain already exists with replicas. Scaling from zero still requires initial spread across `minDomains` healthy domains, preserving the original intent of the constraint.

### User Stories

**Story 1: Zonal outage with HPA**

As a platform engineer, I configure my Deployments with `minDomains: 3` and `maxSkew: 1` to ensure high availability across 3 zones. When one zone experiences a complete outage (nodes tainted), I expect HPA to be able to scale my workload in the remaining healthy zones to maintain service capacity.

**Story 2: Rolling deployment during zone failure**

As an application owner, I deploy a new version of my application during an ongoing zone outage. The new ReplicaSet's pods should be able to schedule in the 2 healthy zones without being blocked by the TSC, since the old ReplicaSet already has a pod in the impaired zone satisfying the spread intent.

### Risks and Mitigations

| Risk | Mitigation |
|------|------------|
| Imbalanced pod distribution after outage recovery | When nodes are un-tainted, the domain re-enters skew calculation with its existing pod count. New pods will preferentially schedule there until balance is restored. Natural rebalancing occurs over time through rolling updates and scale events. |
| Tainted domain has pods that are terminating (NoExecute) | The implementation counts pods currently bound to the node. Even if those pods are terminating, the domain once had replicas placed there, indicating the original intent of `minDomains` was met. |
| Interaction with `NodeAffinityPolicy: Honor` | A node excluded by node affinity (not taints) does not trigger this behavior. The implementation specifically tracks nodes excluded due to taint policy. |

## Design Details

### Semantics

The documentation for `minDomains` would be updated to include:

> When `nodeTaintsPolicy` is set to `Honor`, topology domains whose nodes are **all** tainted with taints not tolerated by the incoming pod, but which contain at least one existing pod matching the constraint's selector, are considered eligible domains for the purpose of satisfying `minDomains` but are **not** included in the calculation for skew.

### Implementation Details

The change is contained within the `PodTopologySpread` plugin's filtering path (`pkg/scheduler/framework/plugins/podtopologyspread/filtering.go`).

**New state in `preFilterState`:**

```go
type preFilterState struct {
    // ... existing fields ...

    // TaintedDomainCount tracks, per constraint, the number of fully-tainted
    // topology domains that have existing matching pods. These domains count
    // toward minDomains but are excluded from skew.
    TaintedDomainCount []int
}
```

**Modified `calPreFilterState` logic:**

Currently, nodes failing `matchNodeInclusionPolicies` (which checks both `NodeAffinityPolicy` and `NodeTaintsPolicy` together) are skipped entirely. The modified logic separates these checks so that taint-excluded nodes can be tracked independently:

1. For nodes excluded specifically due to taint policy (not node affinity), record their topology value and count matching pods from `nodeInfo.Pods`.
2. After processing all nodes, identify topology values that appear **only** in the tainted tracking map (never in `TpValueToMatchNum`). These are fully-tainted domains.
3. For each fully-tainted domain with matching pod count > 0, increment `TaintedDomainCount` for that constraint.

Note: The existing `matchNodeInclusionPolicies` function evaluates both `NodeAffinityPolicy` and `NodeTaintsPolicy` in a single call. The implementation must decompose this check to distinguish nodes excluded due to taints (which feed the tainted-domain tracking) from nodes excluded due to node affinity (which do not).

```go
// Pseudocode for the modified calPreFilterState
//
// matchingPodsOnTaintedNodes tracks, per constraint, how many matching pods
// exist on nodes excluded due to taints, grouped by topology domain.
// Example: matchingPodsOnTaintedNodes[0]["us-east-1c"] = 3
matchingPodsOnTaintedNodes := make([]map[string]int, numConstraints)

for _, nodeInfo := range allNodes {
    node := nodeInfo.Node()

    if !nodeMatchesAffinityPolicy(node, pod, constraint) {
        // Node excluded by NodeAffinityPolicy — skip entirely (existing behavior)
        continue
    }

    if !nodeMatchesTaintsPolicy(node, pod, constraint) {
        // Node is tainted and pod does not tolerate it — record its domain
        // so we can determine later whether the entire domain is tainted.
        domainValue := node.Labels[topologyKey]
        matchingPodCount := countPodsMatchingSelector(nodeInfo, constraint.LabelSelector)
        matchingPodsOnTaintedNodes[constraintIndex][domainValue] += matchingPodCount
        continue
    }

    // ... existing logic: add to TpValueToMatchNum (healthy domain tracking) ...
}

// Post-processing: a domain is "fully tainted" if it appears ONLY in the
// tainted tracking map (no healthy node contributed it to TpValueToMatchNum).
for constraintIndex, taintedDomainPodCounts := range matchingPodsOnTaintedNodes {
    for domainValue, matchingPodCount := range taintedDomainPodCounts {
        _, domainHasHealthyNodes := TpValueToMatchNum[constraintIndex][domainValue]
        domainIsFullyTainted := !domainHasHealthyNodes
        domainHasExistingPods := matchingPodCount > 0

        if domainIsFullyTainted && domainHasExistingPods {
            state.TaintedDomainCount[constraintIndex]++
        }
    }
}
```

**Modified `minMatchNum`:**

```go
func (s *preFilterState) minMatchNum(constraintIndex int, minDomains int32) (int, error) {
    criticalPaths := s.CriticalPaths[constraintIndex]
    globalMinimumMatchCount := criticalPaths[0].MatchNum

    healthyDomainCount := len(s.TpValueToMatchNum[constraintIndex])
    fullyTaintedDomainCount := s.TaintedDomainCount[constraintIndex]
    totalEligibleDomains := healthyDomainCount + fullyTaintedDomainCount

    if totalEligibleDomains < int(minDomains) {
        globalMinimumMatchCount = 0
    }
    return globalMinimumMatchCount, nil
}
```

**Extension points affected:**
- `PreFilter` — modified to track tainted domain state
- `Filter` — unchanged (uses `minMatchNum` which now includes tainted domains)
- `PreScore` / `Score` — unchanged (applies to `ScheduleAnyway` constraints which don't use `minDomains` enforcement)

**Feature gate:** `TaintedDomainExclusionInPodTopologySpread`

When the feature gate is disabled, the behavior is identical to today. The tracking of tainted domains is skipped entirely.

### Test Plan

[x] I/we understand the owners of the involved components may require updates to existing tests to make this code solid enough prior to committing the changes necessary to implement this enhancement.

##### Prerequisite testing updates

No existing tests combine `NodeTaintsPolicy: Honor` with `minDomains > 1`. New test cases will be added to cover the interaction.

##### Unit tests

- `k8s.io/kubernetes/pkg/scheduler/framework/plugins/podtopologyspread`: `2026-MM-DD` - target ≥ 85%

New unit test cases:
1. Fully-tainted domain with existing pods — verify domain counts toward minDomains, excluded from skew
2. Fully-tainted domain with zero matching pods — verify domain does NOT count toward minDomains
3. Partially-tainted domain (some nodes healthy) — verify normal behavior unchanged
4. Multiple fully-tainted domains — verify correct counting
5. Feature gate disabled — verify no behavior change

##### Integration tests

- `test/integration/scheduler/filters/filters_test.go` — new test case for topology spread with fully-tainted zone
- k8s-triage: https://storage.googleapis.com/k8s-triage/index.html?sig=scheduling&test=TestPodTopologySpreadFilter

Integration test scenarios:
1. 3-zone cluster, 1 zone fully tainted, pods schedule to remaining zones
2. Recovery scenario: un-taint nodes, verify skew rebalancing resumes
3. Scale-from-zero with tainted zone — verify minDomains still requires healthy domains

##### e2e tests

This feature does not introduce new API endpoints and operates entirely within the scheduler plugin. Integration tests provide sufficient coverage. An e2e test simulating a zone failure (taint all nodes in a zone, verify pod scheduling) will be added for graduation to Beta.

### Graduation Criteria

#### Alpha (v1.38)

- [ ] Implementation behind `TaintedDomainExclusionInPodTopologySpread` feature gate.
- [ ] Unit and integration tests as described in the Test Plan.
- [ ] Documentation update for `minDomains` semantics.

#### Beta (v1.39)

- [ ] Feature gate enabled by default.
- [ ] Benchmark tests confirm no performance regression for workloads not using this feature.
- [ ] E2E test simulating zone failure scenario.
- [ ] Gather feedback from SIG Scheduling and SIG Autoscaling.

#### GA (v1.40)

- [ ] No negative feedback after at least one release with the feature enabled by default.
- [ ] Confirmed interoperability with major node autoscalers (Karpenter, Cluster Autoscaler).

### Upgrade / Downgrade Strategy

**No API changes:** There are no new fields. Existing `nodeTaintsPolicy: Honor` semantics are extended. Downgrade reverts to the current (stricter) behavior, which may cause some pods to become unschedulable but does not corrupt state. The upgrade/downgrade path is functionally equivalent to feature enablement/disablement and is covered by the same tests.

### Version Skew Strategy

This change is entirely within kube-scheduler. There is no version skew concern with kube-apiserver or kubelet. If multiple schedulers are running, only schedulers with the feature gate enabled will apply the relaxed behavior.

## Production Readiness Review Questionnaire

### Feature Enablement and Rollback

###### How can this feature be enabled / disabled in a live cluster?

- [x] Feature gate
  - Feature gate name: `TaintedDomainExclusionInPodTopologySpread`
  - Components depending on the feature gate: `kube-scheduler`

###### Does enabling the feature change any default behavior?

Yes, but only to address a gap in scheduling semantics during full domain outages. When a pod's TSC has `nodeTaintsPolicy: Honor` AND a topology domain is fully tainted AND that domain has existing matching pods, the scheduler will now allow scheduling to healthy domains rather than blocking all new pods. Users who don't use `nodeTaintsPolicy: Honor` or who don't experience full domain outages see no change.

###### Can the feature be disabled once it has been enabled (i.e. can we roll back the enablement)?

Yes. Disabling the feature gate reverts to the current behavior. Pods that were scheduled under the relaxed skew may violate the strict skew interpretation, but this is no different from normal scheduler state drift (pods are never evicted for TSC violations after placement).

###### What happens if we reenable the feature if it was previously rolled back?

The relaxed scheduling behavior resumes. No state is persisted between enable/disable cycles.

###### Are there any tests for feature enablement/disablement?

Unit tests will verify that with the feature gate disabled, tainted domains are not tracked and `minMatchNum` behaves identically to today.

### Rollout, Upgrade and Rollback Planning

###### How can a rollout or rollback fail? Can it impact already running workloads?

A rollout cannot fail — it only relaxes scheduling constraints. Already-running workloads are unaffected (the change only impacts scheduling decisions for new pods). A rollback to the stricter behavior may cause some pending pods to remain pending, but cannot disrupt running pods.

###### What specific metrics should inform a rollback?

- A spike in `schedule_attempts_total{result="error"}` when pods using `nodeTaintsPolicy: Honor` are added.
- A spike in `plugin_execution_duration_seconds{plugin="PodTopologySpread"}` indicating performance regression.
- Unexpected scheduling decisions visible via `pod_scheduling_sli_duration_seconds`.

###### Were upgrade and rollback tested? Was the upgrade->downgrade->upgrade path tested?

Yes, as described in the Upgrade/Downgrade Strategy section.

###### Is the rollout accompanied by any deprecations and/or removals of features, APIs, fields of API types, flags, etc.?

No.

### Monitoring Requirements

###### How can an operator determine if the feature is in use by their workload?

- Feature gate `TaintedDomainExclusionInPodTopologySpread` is enabled.
- Pods have TSCs with `nodeTaintsPolicy: Honor` and `minDomains > 1`.
- Some topology domain's nodes are all tainted.
- Scheduler logs at V(5) will indicate when a tainted domain is counted toward minDomains.

###### How can someone using this feature know that it is working for their instance?

- Pods that would previously be stuck Pending during a zone outage now schedule to healthy zones.
- `schedule_attempts_total{result="unschedulable"}` decreases for affected workloads during zone outage events.

###### What are the reasonable SLOs (Service Level Objectives) for the enhancement?

This feature maintains existing SLOs. It does not introduce new latency-sensitive code paths or change the performance characteristics of the PodTopologySpread plugin.

###### What are the SLIs (Service Level Indicators) an operator can use to determine the health of the service?

- [x] Metrics
  - Metric name: `plugin_execution_duration_seconds{plugin="PodTopologySpread"}`
  - Components exposing the metric: kube-scheduler
  - Metric name: `schedule_attempts_total{result="error|unschedulable"}`
  - Components exposing the metric: kube-scheduler

### Dependencies

###### Does this feature depend on any specific services running in the cluster?

No. It depends only on the existing node tainting mechanism, which is a core Kubernetes feature.

### Scalability

###### Will enabling / using this feature result in any new API calls?

No.

###### Will enabling / using this feature result in introducing new API types?

No.

###### Will enabling / using this feature result in any new calls to the cloud provider?

No.

###### Will enabling / using this feature result in increasing size or count of the existing API objects?

No. No API schema changes.

###### Will enabling / using this feature result in increasing time taken by any operations covered by existing SLIs/SLOs?

The additional tracking in PreFilter adds O(N) work where N is the number of tainted nodes — a map insertion per tainted node and a post-processing pass over tainted topology values. In practice this is negligible compared to the existing O(N × P) work (N nodes, P pods) in `calPreFilterState`.

###### Will enabling / using this feature result in non-negligible increase of resource usage (CPU, RAM, storage, network, etc.) in any components?

Additional memory: one `map[string]int` per constraint during PreFilter, plus one `int` per constraint in `preFilterState`. This is negligible (< 1KB for typical configurations).

### Troubleshooting

###### How does this feature react if the API server and/or etcd is unavailable?

No impact. The feature operates entirely within the scheduler's in-memory state during a scheduling cycle.

###### What are other known failure modes?

| Failure Mode | Description | Detection | Mitigation |
|---|---|---|---|
| Incorrect "fully tainted" classification | A domain is incorrectly classified as fully tainted when some nodes are healthy but not yet reflected in the scheduler's cache | Pods schedule to non-optimal zones | Self-correcting on next informer sync; existing scheduler cache consistency mechanisms apply |

###### What steps should be taken if SLOs are not being met to determine the cause of the problem?

1. Check if `plugin_execution_duration_seconds{plugin="PodTopologySpread"}` has increased.
2. Disable the feature gate and restart kube-scheduler.
3. Verify scheduling latency returns to baseline.

## Implementation History

- 2026-06-09: Initial KEP draft created.

## Drawbacks

1. **Requires full domain tainting:** The fallback only applies when ALL nodes in a domain are tainted. Partial failures (some nodes healthy, new capacity cannot be provisioned) are not addressed. This limitation exists in current behavior and represents a broader challenge in autoscaler/scheduler communication that is out of scope.

2. **Post-outage imbalance:** When nodes are un-tainted after an outage, pods remain in an imbalanced state. This is acceptable because the cluster naturally rebalances: future scheduling decisions will favor the previously-tainted domain until skew is restored.

## Alternatives

### Alternative 1: New field `domainUnavailablePolicy`

Introduce a new enum field on TopologySpreadConstraint that explicitly controls behavior when a domain becomes unavailable. This was considered but rejected because:
- It adds API surface area and cognitive load for users.
- The proposed behavior (honor tainted domains for minDomains, exclude from skew) is the only sensible semantic when `nodeTaintsPolicy: Honor` is set.
- Users who already configured `nodeTaintsPolicy: Honor` explicitly opted into taint-aware scheduling.

### Alternative 2: Scheduler/Autoscaler communication channel

A mechanism for node autoscalers to signal kube-scheduler that capacity cannot be provisioned in a given domain. This is complementary but orthogonal:
- It would address stockout scenarios (grey failures) that this KEP does not.
- It would not cover the static case (all existing nodes tainted) that this KEP does cover.
- It requires cross-component protocol design and is significantly more complex.
- This KEP's approach works immediately with existing tainting infrastructure.

### Alternative 3: Treat partially-tainted domains the same way

Extend the behavior to domains where only some nodes are tainted. This was rejected because:
- If healthy nodes exist in a domain, pods can (and should) schedule there.
- The semantic is clear for "100% tainted" but ambiguous for partial cases.
- Partial tainting often indicates node-level issues, not domain-level outages.
