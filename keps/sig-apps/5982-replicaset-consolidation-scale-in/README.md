# KEP-5982: ReplicaSet Consolidation-Aware Scale-In Strategy

<!-- toc -->
<!-- /toc -->

## Summary

Add a `ConsolidatingScaleDown` feature gate to `kube-controller-manager` that changes the ReplicaSet controller's pod deletion sort order during scale-down. When enabled, the controller prefers deleting pods on nodes with fewer total active pods, enabling node autoscalers to reclaim nodes with less disruption.

## Motivation

The ReplicaSet controller's current scale-down heuristic spreads pod deletions evenly across nodes. This works against node autoscalers (Karpenter, cluster-autoscaler) which need nodes to become empty or underutilized before they can remove them. The spreading heuristic leaves every node partially occupied after a scale-down, requiring the autoscaler to evict additional pods to consolidate — increasing total disruption.

### Goals

- Provide an alternative pod deletion heuristic that supports node consolidation during workload scale-down
- Reduce total pod disruption by enabling scale-down events to naturally empty nodes
- Complement KEP-2255 (PodDeletionCost) — both mechanisms coexist, with PodDeletionCost taking precedence

### Non-Goals

- Changing scheduler placement decisions
- Modifying StatefulSet or Job scale-down behavior
- Replacing or deprecating PodDeletionCost (KEP-2255)
- Guaranteeing optimal node packing (this is a heuristic improvement)

## Proposal

*Details to be added as the design evolves. See [kubernetes/enhancements#5982](https://github.com/kubernetes/enhancements/issues/5982) for discussion.*

## Design Details

*To be added.*

## Alternatives

- **KEP-2255 (PodDeletionCost):** External controller sets annotations to influence pod deletion order. Works today but requires continuous annotation management and API server writes.
- **Eviction Request API (KEP-4563):** More general eviction coordination mechanism. Operates at the eviction layer rather than the RS controller's pod selection layer.

## Infrastructure Needed

None.
