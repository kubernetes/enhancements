# KEP-5925: Enhanced Storage Capacity Tracking

<!-- toc -->
- [Release Signoff Checklist](#release-signoff-checklist)
- [Summary](#summary)
- [Motivation](#motivation)
  - [Goals](#goals)
- [Proposal](#proposal)
  - [User Stories](#user-stories)
    - [Story 1: Multiple homogeneous storage pools and joint PVC evaluation](#story-1-multiple-homogeneous-storage-pools-and-joint-pvc-evaluation)
    - [Story 2: Concurrent pod scheduling with stale capacity data](#story-2-concurrent-pod-scheduling-with-stale-capacity-data)
    - [Story 3: Node drain with local storage](#story-3-node-drain-with-local-storage)
  - [Notes/Constraints/Caveats](#notesconstraintscaveats)
- [Design Details](#design-details)
  - [Enhancement 1: Multi-Pool Capacity Reporting](#enhancement-1-multi-pool-capacity-reporting)
    - [CSI Spec Changes](#csi-spec-changes)
    - [Kubernetes API Changes](#kubernetes-api-changes)
    - [External-Provisioner Changes](#external-provisioner-changes)
    - [Scheduler Changes](#scheduler-changes)
  - [Enhancement 2: Capacity Reservation](#enhancement-2-capacity-reservation)
    - [Data Structure](#data-structure)
    - [Filter Phase](#filter-phase)
    - [Reserve Phase](#reserve-phase)
    - [Unreserve](#unreserve)
    - [Interaction with Enhancement 1](#interaction-with-enhancement-1)
  - [Enhancement 3: Capacity-Aware Rescheduling](#enhancement-3-capacity-aware-rescheduling)
    - [CSI Driver Contract](#csi-driver-contract)
    - [Scheduler Changes: checkBoundClaims](#scheduler-changes-checkboundclaims)
    - [Scheduler Changes: Filter Phase](#scheduler-changes-filter-phase)
    - [Scheduler Changes: PreBind Phase](#scheduler-changes-prebind-phase)
    - [Relationship to KEP-5381](#relationship-to-kep-5381)
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
- [ ] Supporting documentation

[kubernetes.io]: https://kubernetes.io/
[kubernetes/enhancements]: https://git.k8s.io/enhancements
[kubernetes/kubernetes]: https://git.k8s.io/kubernetes
[kubernetes/website]: https://git.k8s.io/website

## Summary

This KEP enhances the existing CSI storage capacity tracking framework ([KEP-1472](https://github.com/kubernetes/enhancements/tree/master/keps/sig-storage/1472-storage-capacity-tracking)) with three capabilities to improve dynamic provisioning reliability:

1. **Multi-pool capacity reporting** — extends the CSI `GetCapacity` RPC response and the `CSIStorageCapacity` Kubernetes
   API to report per-pool available capacity within a topology segment, and changes the scheduler to evaluate all of a
   pod's PVCs jointly against the pool list. This solves two problems: nodes with multiple homogeneous storage pools
   (e.g., multiple identical SSDs) that cannot be distinguished by StorageClass parameters, and the current behavior
   where each PVC is evaluated independently as if it does not affect the others.

2. **Capacity reservation** — the kube-scheduler tracks recently used `CSIStorageCapacity` objects in memory and treats
   them as unavailable until external-provisioner refreshes them. After a pod is scheduled, it takes time for the CSI
   driver to update its internal state and for the `CSIStorageCapacity` object to be refreshed. During this window many
   more pods may use the same stale object, leading to overprovisioning.

3. **Capacity-aware rescheduling** — when a bound PVC's original node is unschedulable, the scheduler reclassifies it as
   a delayed-binding unbound PVC and applies capacity filtering, ensuring the new node has sufficient space for the CSI
   driver to rebuild the volume. This is gated by a new `CSIDriver.spec.volumeRebuilding` capability field.

All three enhancements are controlled by the `CSIStorageCapacityV2` feature gate and are fully backward compatible.

## Motivation

[KEP-1472](https://github.com/kubernetes/enhancements/tree/master/keps/sig-storage/1472-storage-capacity-tracking) introduced `CSIStorageCapacity` objects and scheduler-side capacity filtering, reaching GA in Kubernetes v1.24. While this significantly improved scheduling for storage-aware workloads, several limitations remain:

**Single pool per topology and independent PVC evaluation.** The current `CSIStorageCapacity` object reports a single
`Capacity` value per topology segment, and the scheduler evaluates each PVC independently. This means if a node reports
100Gi available and a pod requests two PVCs of 100Gi each, both PVCs pass the filter individually. The outcome is
ambiguous: the scheduler cannot determine whether provisioning will actually succeed, because the single `Capacity` value
does not convey how many independent volumes the underlying storage can accommodate. In practice, provisioning typically
fails. A reported capacity of 100Gi usually means a single disk that cannot hold two 100Gi volumes.

KEP-1472 acknowledges this limitation explicitly: "Each volume gets checked separately, independently of other volumes
that are needed by the current pod or by volumes that are about to be created for other pods. Those scenarios remain
problematic." A retry mechanism was proposed to revert scheduling decisions when some PVCs fail to provision
([enhancements#1703](https://github.com/kubernetes/enhancements/pull/1703)), but retries alone do not solve the problem.
Without changes to the scheduling logic itself, the scheduler may select the same invalid node repeatedly because the
capacity model still appears to have sufficient space. The capacity model must be made more explicit so that the
scheduler can make correct joint decisions up front rather than relying on post-hoc retries.

The CSI community previously discussed multi-pool support ([CSI spec issue #55](https://github.com/container-storage-interface/spec/issues/55)) and resolved it by adding `parameters` to `GetCapacityRequest` ([CSI spec PR #85](https://github.com/container-storage-interface/spec/pull/85)), allowing the Container Orchestrator to query capacity per StorageClass. This works for **heterogeneous** pools (e.g., SSD vs HDD) where each pool type maps to a different StorageClass. However, it does **not** work for **homogeneous** pools — for example, a node with three identical 100GB SSDs, each managed as a separate storage pool by the CSI driver. All three pools are the same type, so they map to the same StorageClass and `GetCapacity` returns a single aggregated value. Operators must choose between reporting the total sum (300GB, which overstates what a single volume can use) or the largest pool's capacity (100GB, which understates total available resources for multiple volumes). Neither is accurate for scheduling.

**Stale capacity data.** KEP-1472 explicitly states that "no attempts will be made to model how capacity will be
affected by pending volume operations. This would depend on internal driver details that Kubernetes doesn't have." While
precise capacity modeling is indeed impractical, a simpler approach is feasible: a scheduler-side in-memory cache that
tracks recently used `CSIStorageCapacity` objects and treats them as unavailable until external-provisioner refreshes
them. This does not require knowledge of driver internals, yet can significantly reduce the number of provisioning
failures and unnecessary scheduling retries caused by stale data.

**No capacity-aware rescheduling.** When a pod using local storage is deleted and recreated (e.g., due to node drain for
maintenance or node failure), its PVCs are already bound. The scheduler sees bound PVCs and skips capacity checking
entirely. If the CSI driver supports volume rebuilding on a different node, it expects the pod to land on a node with
sufficient capacity. The current scheduler has no mechanism to ensure this.

### Goals

- Enable accurate capacity reporting for topologies with multiple homogeneous storage pools by extending the CSI
  `GetCapacity` RPC response and the `CSIStorageCapacity` Kubernetes API.

- Change the scheduler to evaluate all of a pod's PVCs jointly against the available pool list, rather than checking
  each PVC independently.

- Reduce scheduling failures caused by stale capacity data by reserving used `CSIStorageCapacity` objects in the
  scheduler's memory until they are refreshed by external-provisioner.

- Enable capacity-aware node selection when bound PVCs need to be rebuilt on a different node because the original node
  is unschedulable, for CSI drivers that opt in via a new capability field.

## Proposal

### User Stories

#### Story 1: Multiple homogeneous storage pools and joint PVC evaluation

**Problem A — independent PVC evaluation:** As a cluster administrator, I have a node with a single 100GB disk managed
by a CSI driver. The node reports `Capacity: 100Gi`. A pod requests two PVCs of 100Gi each. The scheduler evaluates
each PVC separately: the first PVC sees 100Gi >= 100Gi and passes, the second PVC also sees 100Gi >= 100Gi and passes.
The pod is scheduled on this node, but there is only one disk — the first PVC provisions successfully but the second
fails.

**Problem B — inaccurate aggregate capacity:** I have another node with three identical 100GB NVMe disks, each managed
as a separate storage pool by the CSI driver. All three disks are the same type and same type. If the node reports
`Capacity: 300Gi` (sum of all disks), a pod requesting a single 120Gi PVC passes the filter. But no individual disk can
hold 120Gi, so provisioning fails.

**Solution:** With multi-pool capacity reporting and joint PVC evaluation, the single-disk node reports
`AvailableCapacities: [100Gi]` and the three-disk node reports `AvailableCapacities: [100Gi, 100Gi, 100Gi]`. The
scheduler groups PVCs by StorageClass and runs bin-packing against the pool list:
- 2×100Gi into 1×100Gi pool → REJECTED (only one can fit)
- 1×120Gi into 3×100Gi pools → REJECTED (no single pool is large enough)
- 2×100Gi into 3×100Gi pools → ACCEPTED (one per pool)
- 3×80Gi into 3×100Gi pools → ACCEPTED (one per pool)
- 4×80Gi into 3×100Gi pools → REJECTED (only 3 disjoint pools)

#### Story 2: Stale capacity data and overprovisioning

As a cluster operator running a batch processing platform, I have nodes with 100GB of local SSD storage managed by a
CSI driver. I submit 10 pods, each requesting 20GB. The scheduler processes them one by one. The first pod is scheduled
on node-A (100GB free). But before the CSI driver provisions the volume and external-provisioner refreshes the
`CSIStorageCapacity` object, the scheduler processes the next 9 pods — they all see the same stale 100GB and are all
directed to node-A. The first 5 pods provision successfully, but pods 6–10 fail because only 100GB was available total.
With capacity reservation, the scheduler marks the `CSIStorageCapacity` object as reserved after scheduling the first
pod. Subsequent pods see no available capacity on that object and are directed to other nodes or wait until the object
is refreshed.

#### Story 3: Node drain with local storage

As a platform engineer running StatefulSet workloads with local storage on a CSI driver that supports volume rebuilding
(e.g., Longhorn), I need to drain node-A for maintenance. I run `kubectl drain node-A`. The node becomes unschedulable
and the pods are evicted. The CSI driver detects the node is unschedulable and unsets PV nodeAffinity for volumes on
that node (via KEP-5381). The StatefulSet controller recreates the pods. The scheduler processes each bound PVC: it sees
the CSI driver supports volume rebuilding and the original node is unschedulable, so it reclassifies the PVC as a
delayed-binding unbound PVC and checks `CSIStorageCapacity` to find a node with enough space. After selecting a new
node, the scheduler sets `pod.spec.nodeName` to the chosen node. The AD controller then creates a `VolumeAttachment`
for that node, triggering the volume attach/mount flow. The CSI driver detects that the volume is requested on a new
node (via `ControllerPublishVolume` if `attachRequired: true`, or via `NodeStageVolume` otherwise), rebuilds the
volume, and sets PV nodeAffinity to the new node.

### Notes/Constraints/Caveats

- **Multi-pool bin-packing is NP-hard in general** but tractable for realistic workloads where the number of PVCs per
  pod is small (typically <10). A first-fit-decreasing greedy algorithm is acceptable for alpha.

- **Capacity reservation is conservative.** Reserving the entire object means that even if the topology has remaining
  capacity, no pod will be scheduled there until external-provisioner refreshes the object. This is intentional — we do
  not know how the CSI driver assigns storage across pools, so partial deduction is not possible. In the worst case
  (infrequent capacity updates), this reduces scheduling throughput to one pod per update interval per
  CSIStorageCapacity object.

- **Rescheduling depends on CSI driver cooperation.** The CSI driver must opt in via
  `CSIDriver.spec.volumeRebuilding`, must manage PV nodeAffinity via KEP-5381, and must handle the volume rebuild when
  the volume is requested on a new node.

- **Interaction with KEP-4049 (Storage Capacity Scoring).** [KEP-4049](https://github.com/kubernetes/enhancements/tree/master/keps/sig-storage/4049-storage-capacity-scoring-of-nodes-for-dynamic-provisioning) adds a Score phase to the VolumeBinding plugin that ranks nodes by available capacity for dynamic provisioning. Its scoring logic reads `provision.Capacity.Capacity.Value()` directly, assuming a single `Capacity` value per `CSIStorageCapacity` object. When `AvailableCapacities` is set, the scoring function will need to be updated to derive an effective capacity from the pool list (e.g., summing all pool entries). To minimize the impact, external-provisioner should populate both `Capacity` (as the sum of all pools) and `AvailableCapacities` (individual pool values), so that the existing scoring logic continues to work without modification for the common case.

## Design Details

### Enhancement 1: Multi-Pool Capacity Reporting

#### CSI Spec Changes

The `GetCapacityResponse` message is extended with a repeated field for per-pool available capacity:

```protobuf
message GetCapacityResponse {
    // ... existing fields ...

    // Per-pool available capacity within this topology segment.
    // Each entry represents the available capacity, in bytes, of one
    // independent storage pool. When this field is non-empty, the CO
    // SHOULD use it instead of available_capacity for scheduling
    // decisions, and SHOULD evaluate all pending volumes jointly
    // against the pool list rather than independently.
    //
    // This field is OPTIONAL. If the SP does not report per-pool
    // capacity, this field MUST be empty, and the CO falls back to
    // available_capacity.
    //
    // Example: a node with three 100GiB SSDs, each a separate pool,
    // would report: available_capacities: [107374182400,
    // 107374182400, 107374182400]
    repeated int64 available_capacities = 4;
}
```

This is a backward-compatible addition. Existing CSI drivers that do not populate the field continue to work as before.
The CO (Kubernetes) falls back to `available_capacity` when the list is empty.

#### Kubernetes API Changes

A new field `AvailableCapacities` is added to `CSIStorageCapacity` in `storage.k8s.io/v1`:

```go
type CSIStorageCapacity struct {
    // ... existing fields ...

    // AvailableCapacities reports per-pool available capacity within this topology segment. Each entry represents the
    // available capacity of one independent storage pool. When set, the scheduler uses this field instead of Capacity
    // and evaluates all of a pod's PVCs jointly against the pool list.
    //
    // This field addresses two problems: (1) multiple homogeneous pools within a topology (e.g., a node with 3
    // identical SSDs) that cannot be distinguished by StorageClass parameters, and (2) the current behavior where each
    // PVC is evaluated independently, allowing a pod to be scheduled on a node that cannot satisfy all of its PVCs.
    //
    // +optional
    // +listType=atomic
    AvailableCapacities []resource.Quantity
}
```

#### External-Provisioner Changes

The external-provisioner is updated to:

1. Call `GetCapacity` as today (per topology segment per StorageClass).
2. If the response contains non-empty `available_capacities`, populate the `AvailableCapacities` field on the
   `CSIStorageCapacity` object and set `Capacity` to the sum of all pool entries. Populating both fields ensures that
   the existing scoring logic in [KEP-4049](https://github.com/kubernetes/enhancements/tree/master/keps/sig-storage/4049-storage-capacity-scoring-of-nodes-for-dynamic-provisioning)
   continues to work without modification, while the Filter phase uses `AvailableCapacities` for accurate bin-packing.

No changes to how external-provisioner determines topology or StorageClass — only the mapping of
`GetCapacityResponse` fields to `CSIStorageCapacity` fields changes.

#### Scheduler Changes

The VolumeBinding plugin is updated to group a pod's PVCs by StorageClass and evaluate each group jointly against its
matching `CSIStorageCapacity` object using bin-packing.

For each StorageClass group:

1. Find the matching `CSIStorageCapacity` object for the topology.
2. Build the pool list:
   - If `AvailableCapacities` is set, use it directly.
   - If not set, fall back to `MaximumVolumeSize` or `Capacity` and treat it as a single-element array (i.e., one pool).
3. Run first-fit-decreasing bin-packing of all PVC requests in the group against the pool list:

```go
func canSatisfyPVCs(pools []int64, requests []int64) bool {
    // Sort both in descending order for first-fit-decreasing
    sort.Sort(sort.Reverse(sort.IntSlice(pools)))
    sort.Sort(sort.Reverse(sort.IntSlice(requests)))
    // Greedy matching
    used := make([]int64, len(pools))
    for _, req := range requests {
        placed := false
        for i, pool := range pools {
            if pool-used[i] >= req {
                used[i] += req
                placed = true
                break
            }
        }
        if !placed {
            return false
        }
    }
    return true
}
```

This replaces the current per-PVC independent check. When `AvailableCapacities` is not set, the single-element fallback
preserves backward-compatible behavior while still catching the case where multiple PVCs exceed a single pool's capacity.

First-fit-decreasing is a well-known bin-packing heuristic. For the typical case of <10 PVCs per pod, even brute-force
would be acceptable, but FFD is chosen for clarity and efficiency.

### Enhancement 2: Capacity Reservation

#### Data Structure

The `ReservedCapacity` struct is modeled after the existing PV/PVC assumed cache in the scheduler and is initialized
with the VolumeBinding plugin. It is a simple map from UID to bool:

```go
type ReservedCapacity struct {
    mu       sync.RWMutex
    reserved map[types.UID]bool
}

func (rc *ReservedCapacity) Reserve(uid types.UID) {
    rc.mu.Lock()
    defer rc.mu.Unlock()
    rc.reserved[uid] = true
}

func (rc *ReservedCapacity) Unreserve(uid types.UID) {
    rc.mu.Lock()
    defer rc.mu.Unlock()
    delete(rc.reserved, uid)
}

func (rc *ReservedCapacity) IsReserved(uid types.UID) bool {
    rc.mu.RLock()
    defer rc.mu.RUnlock()
    return rc.reserved[uid]
}
```

#### Filter Phase

When evaluating whether a `CSIStorageCapacity` object has sufficient capacity, the plugin checks the reserved set
first. If the object is reserved, it is skipped entirely (treated as having zero capacity).

#### Reserve Phase

After the scheduler makes a binding decision, it calls `Reserve(uid)` on the `CSIStorageCapacity` object that was used.

#### Unreserve

The reservation is cleared in two cases:

1. **Object update via informer:** The plugin registers an informer event handler for `CSIStorageCapacity` objects. When
   an object is updated, the handler calls `Unreserve(uid)`. Note that an update does not guarantee it was caused by
   the previous scheduling decision — external-provisioner refreshes capacity when any volume is created, deleted, or
   expanded, so unrelated volume operations can also trigger an update. Conversely, if the previous scheduling decision
   involved multiple PVCs, each PVC's provisioning may cause a separate `CSIStorageCapacity` update, so the first
   update may not fully reflect the capacity consumed by all PVCs.

2. **Scheduling failure:** If the scheduling cycle fails (e.g., the pod is rejected by a later plugin or binding fails),
   the plugin's `Unreserve` callback calls `Unreserve(uid)` to release the reservation.

**Changes required:**

- `pkg/scheduler/framework/plugins/volumebinding/` — add `ReservedCapacity` struct, integrate into Filter, Reserve, and
  Unreserve phases.
- No API changes. No new API calls.

#### Interaction with Enhancement 1

When a `CSIStorageCapacity` object has `AvailableCapacities` set, capacity reservation means that no pool from that
object is used until the capacity data is refreshed. This is conservative but correct — the scheduler cannot predict
which pool the CSI driver will use for a given volume.

### Enhancement 3: Capacity-Aware Rescheduling

#### Kubernetes API Changes

A new field `VolumeRebuilding` is added to `CSIDriverSpec` in `storage.k8s.io/v1`:

```go
type CSIDriverSpec struct {
    // ... existing fields ...

    // VolumeRebuilding indicates that this CSI driver supports rebuilding volumes on a different node when the original
    // node becomes unschedulable. When true, the scheduler may reclassify bound PVCs as delayed-binding unbound PVCs
    // and apply capacity filtering during rescheduling.
    //
    // CSI drivers that set this to true MUST:
    // - Watch Node objects and unset PV nodeAffinity when a node becomes unschedulable (via KEP-5381 Mutable PV Node
    //   Affinity)
    // - Handle volume rebuild when the volume is requested on a new node (move the underlying storage backend)
    // - Set PV nodeAffinity to the new node after rebuild completes
    //
    // +optional
    VolumeRebuilding *bool
}
```

#### CSI Driver Contract

This enhancement requires explicit opt-in from CSI drivers via `CSIDriver.spec.volumeRebuilding: true`. Drivers that
set this field MUST fulfill the following contract:

1. **Manage PV nodeAffinity via KEP-5381.** The CSI driver watches Node objects. When a node becomes unschedulable (or
   is deleted), the driver unsets `PV.spec.nodeAffinity` for all PVs whose volumes reside on that node. This allows the
   scheduler to consider other nodes as candidates. After rebuilding a volume on a new node, the driver sets PV
   nodeAffinity to the new node.

2. **Handle volume rebuild when the volume is requested on a new node.** The signal depends on
   `CSIDriver.spec.attachRequired`:
   - If `attachRequired: true` (default): the attach-detach controller creates a VolumeAttachment on the new node. The
     CSI driver receives a `ControllerPublishVolume` call for the new node and detects that the volume currently resides
     on a different node — this is the rebuild signal.
   - If `attachRequired: false`: no VolumeAttachment is created. The kubelet calls
     `NodeStageVolume`/`NodePublishVolume` on the new node directly, and the CSI driver detects the rebuild need at
     that point.

#### Scheduler Changes: checkBoundClaims

The key change is in `checkBoundClaims` in the VolumeBinding plugin. Currently, bound PVCs are checked only for PV
nodeAffinity match. This enhancement adds per-PVC reclassification logic:

For each bound PVC:

1. Look up the PV and check nodeAffinity against the candidate node (existing behavior — if nodeAffinity doesn't match,
   the PVC fails as before).
2. Look up the CSIDriver for the PVC's StorageClass provisioner.
3. If `CSIDriver.spec.volumeRebuilding` is not true, keep the PVC as a normal bound PVC (existing behavior).
4. Read the `volume.kubernetes.io/selected-node` annotation from the PVC.
5. Look up the original node. If the original node is schedulable (or the annotation is missing), keep the PVC as a
   normal bound PVC.
6. If the original node is unschedulable or deleted, **reclassify this PVC as a delayed-binding unbound PVC**.

```go
func shouldReclassify(pvc *v1.PersistentVolumeClaim,
    pv *v1.PersistentVolume) bool {

    driver := getCSIDriver(pvc)
    if driver == nil || !ptr.Deref(driver.Spec.VolumeRebuilding, false) {
        return false
    }

    selectedNode := pvc.Annotations[annSelectedNode]
    if selectedNode == "" {
        return false
    }

    node, err := getNode(selectedNode)
    if err != nil || node == nil {
        return true // node deleted
    }
    return node.Spec.Unschedulable
}
```

Reclassified PVCs are returned alongside actual unbound PVCs and proceed through the normal delayed-binding capacity
filtering path. This means a pod can have a mix of:
- Normal bound PVCs (nodeAffinity check only)
- Reclassified bound-to-unbound PVCs (capacity filtered)
- Actual unbound PVCs (capacity filtered as today)

#### Scheduler Changes: Filter Phase

Reclassified PVCs enter the same capacity filtering path as actual unbound PVCs. The existing logic for matching
`CSIStorageCapacity` objects by driver, StorageClass, and topology applies. Enhancement 1 (multi-pool) and Enhancement 2
(capacity reservation) also apply to these PVCs.

No additional Filter phase changes are needed — reclassification in `checkBoundClaims` is sufficient.

#### Scheduler Changes: PreBind Phase

No separate handling is needed. Reclassified PVCs join the existing unbound delayed-binding PVC list in
`checkBoundClaims`, so the existing PreBind logic handles them identically — it sets the
`volume.kubernetes.io/selected-node` annotation to the chosen node.

The only difference is that reclassified PVCs are already bound (`PVC.Status.Phase` is `Bound`), so the wait-for-binding
phase completes immediately. Once the pod is scheduled to the new node (i.e., `pod.spec.nodeName` is set), the AD
controller creates a `VolumeAttachment` for that node, triggering the volume attach/mount flow. The CSI driver detects
that the volume is requested on a new node and moves the underlying storage backend accordingly.

### Test Plan

[X] I/we understand the owners of the involved components may require updates to existing tests to make this code solid
enough prior to committing the changes necessary to implement this enhancement.

##### Prerequisite testing updates

No prerequisite testing updates are needed.

##### Unit tests

The following unit tests are planned:

**Enhancement 1 (Multi-Pool Capacity Reporting):**
- Single PVC against multi-pool: passes when at least one pool is large enough
- Single PVC against multi-pool: fails when no pool is large enough
- Multiple PVCs joint evaluation: all PVCs can be placed across pools (bin-packing)
- Multiple PVCs joint evaluation: not enough pools for all PVCs
- Two 100Gi PVCs against single 100Gi pool: correctly rejected (not independently accepted)
- Fall back to Capacity when AvailableCapacities is not set

**Enhancement 2 (Capacity Reservation):**
- Object is skipped in Filter when reserved with matching resourceVersion
- Object is available in Filter when resourceVersion has changed
- Reservation is recorded in Reserve phase
- Reservation is cleared on informer update
- Scheduler restart (fresh state) has no reservations
- Multiple concurrent scheduling decisions: first reserves, second sees reserved

**Enhancement 3 (Capacity-Aware Rescheduling):**
- Bound PVC with volumeRebuilding=false: stays as normal bound PVC
- Bound PVC with volumeRebuilding=true, schedulable original node: stays as normal bound PVC
- Bound PVC with volumeRebuilding=true, unschedulable original node: reclassified as delayed-binding unbound
- Bound PVC with volumeRebuilding=true, deleted original node: reclassified as delayed-binding unbound
- Mixed pod: some PVCs reclassified, some stay bound, some actually unbound
- PV nodeAffinity still checked before reclassification
- Selected-node annotation updated to new node after rescheduling
- Missing selected-node annotation: stays as normal bound PVC

##### Integration tests

Integration tests will be added to `test/integration/volumescheduling/`:

- Multi-pool capacity with joint PVC evaluation correctly filters and admits nodes
- Capacity reservation prevents concurrent overcommit
- Rescheduling path correctly reclassifies bound PVCs and selects nodes with sufficient capacity

##### e2e tests

E2e tests using the CSI hostpath driver:

- Verify multi-pool capacity objects with joint PVC evaluation are respected by the scheduler
- Verify capacity reservation reduces concurrent scheduling conflicts
- Verify rescheduling selects a node with sufficient capacity when original node is cordoned

### Graduation Criteria

#### Alpha

- Feature implemented behind `CSIStorageCapacityV2` feature gate
- CSI spec PR proposed for `available_capacities`
- Unit tests for all three enhancements
- Integration tests for basic scenarios
- At least one CSI driver validates the rescheduling flow

#### Beta

- CSI spec change merged
- Gather feedback from CSI driver maintainers and users
- Benchmarking tests for scheduling latency impact
- At least 2 CSI drivers implement `volumeRebuilding`
- E2e tests in testgrid and linked in KEP

#### GA

- 5+ CSI drivers using multi-pool reporting
- 3+ CSI drivers implementing volume rebuilding
- No regressions reported during beta
- At least 2 releases since beta

### Upgrade / Downgrade Strategy

**Upgrade:** Enabling the feature gate activates the new scheduler logic. `CSIStorageCapacity` objects with the
`AvailableCapacities` field are accepted by the API server. Existing objects without the field continue to work as
before. `CSIDriver` objects with `volumeRebuilding` are accepted. No changes are required to maintain previous behavior.

**Downgrade:** Disabling the feature gate reverts to current behavior. The `AvailableCapacities` field on existing
objects is preserved in etcd but ignored by the scheduler. The `volumeRebuilding` field is preserved but ignored.
In-memory reservation state is lost, which is the same as the current behavior. No impact on running workloads.

### Version Skew Strategy

- **kube-scheduler without feature / kube-apiserver with feature:** The scheduler ignores the new API fields. Scheduling
  works as today.

- **kube-scheduler with feature / kube-apiserver without feature:** The new API fields cannot be stored. Enhancement 2
  (in-memory only) still works. Enhancements 1 and 3 are inactive because the API server does not accept the new fields.

- **Old external-provisioner:** Does not populate `AvailableCapacities`. The scheduler falls back to `Capacity`. No
  behavioral change.

- **Old CSI driver (no available_capacities):** Returns empty field. external-provisioner uses `Capacity` as before.

## Production Readiness Review Questionnaire

### Feature Enablement and Rollback

###### How can this feature be enabled / disabled in a live cluster?

- [X] Feature gate (also fill in values in `kep.yaml`)
  - Feature gate name: `CSIStorageCapacityV2`
  - Components depending on the feature gate: kube-apiserver, kube-scheduler
- [X] CSIDriver.spec.volumeRebuilding field (opt-in per driver for Enhancement 3)

###### Does enabling the feature change any default behavior?

Enabling the feature gate alone does not change default behavior. Behavior changes only when:
- CSI drivers populate `available_capacities` in GetCapacity responses and external-provisioner writes
  `AvailableCapacities` on `CSIStorageCapacity` objects (Enhancement 1)
- Multiple pods with volumes are scheduled concurrently against the same `CSIStorageCapacity` object (Enhancement 2 —
  reservation kicks in)
- CSI drivers set `volumeRebuilding: true` on their `CSIDriver` object (Enhancement 3)

###### Can the feature be disabled once it has been enabled (i.e. can we roll back the enablement)?

Yes. Disabling the feature gate reverts all behavior. In-memory reservation state is discarded on scheduler restart.
Existing API objects with new fields are preserved but ignored. No impact on running workloads.

###### What happens if we reenable the feature if it was previously rolled back?

The feature resumes. In-memory reservation starts from zero (clean state). API fields that were preserved in etcd become
active again.

###### Are there any tests for feature enablement/disablement?

Unit tests will cover behavior with and without the feature gate enabled.

### Rollout, Upgrade and Rollback Planning

###### How can a rollout or rollback fail? Can it impact already running workloads?

Turning the feature gate on/off only affects scheduling decisions for new pods. Running workloads are not impacted. A
rollout could fail if the kube-apiserver and kube-scheduler have different feature gate settings (version skew), but
this only results in the feature being inactive, not in errors.

###### What specific metrics should inform a rollback?

- A spike in `schedule_attempts_total{result="error|unschedulable"}` when the feature gate is enabled.
- Increased `plugin_execution_duration_seconds{plugin="VolumeBinding"}` indicating performance regression.

###### Were upgrade and rollback tested? Was the upgrade->downgrade->upgrade path tested?

Not yet. Will be tested during alpha.

###### Is the rollout accompanied by any deprecations and/or removals of features, APIs, fields of API types, flags, etc.?

No.

### Monitoring Requirements

###### How can an operator determine if the feature is in use by workloads?

- `CSIStorageCapacity` objects with non-empty `AvailableCapacities` indicate Enhancement 1 is in use.
- Non-zero value of new metric `volume_binding_capacity_reservations_total` indicates Enhancement 2 is active.
- Non-zero value of new metric `volume_binding_rescheduling_events_total` indicates Enhancement 3 is active.

###### How can someone using this feature know that it is working for their instance?

- [ ] Events
  - Event Reason: `CapacityAwareRescheduling` — emitted when the scheduler reclassifies a bound PVC as delayed-binding
    unbound due to unschedulable original node.
- [ ] Metrics
  - `volume_binding_capacity_reservations_total` — number of capacity reservations made
  - `volume_binding_capacity_reservation_resets_total` — number of reservations cleared by informer update
  - `volume_binding_rescheduling_events_total` — number of PVC reclassification events

###### What are the reasonable SLOs (Service Level Objectives) for the enhancement?

- `plugin_execution_duration_seconds{plugin="VolumeBinding"}` <= 100ms on 99th percentile (same as current baseline).

###### What are the SLIs (Service Level Indicators) an operator can use to determine the health of the service?

- [X] Metrics
  - Metric name: `plugin_execution_duration_seconds{plugin="VolumeBinding"}`
  - Components exposing the metric: kube-scheduler

###### Are there any missing metrics that would be useful to have to improve observability of this feature?

The new metrics listed above are introduced by this KEP.

### Dependencies

###### Does this feature depend on any specific services running in the cluster?

- **KEP-5381 (Mutable PV Node Affinity):** Required for Enhancement 3. CSI drivers need this to update PV nodeAffinity
  at runtime.
  - Impact of its absence: Enhancement 3 rescheduling still works at the scheduler level, but CSI drivers cannot manage
    nodeAffinity, limiting the workflow to drivers that never set nodeAffinity.

No other new dependencies beyond those already required by KEP-1472 (CSI driver with external-provisioner publishing
`CSIStorageCapacity` objects).

### Scalability

###### Will enabling / using this feature result in any new API calls?

Enhancement 3 (rescheduling path) issues PVC annotation updates (PATCH) when a rescheduling decision is made. This is
expected to be infrequent (only when a node becomes unschedulable and pods are recreated).

###### Will enabling / using this feature result in introducing new API types?

No new types. New fields on existing types:
- `CSIStorageCapacity.AvailableCapacities` (list of quantities)
- `CSIDriverSpec.VolumeRebuilding` (boolean pointer)

###### Will enabling / using this feature result in any new calls to the cloud provider?

No.

###### Will enabling / using this feature result in increasing size or count of the existing API objects?

- `CSIStorageCapacity`: size may increase when `AvailableCapacities` is populated (list of quantities, typically <10
  entries).
- `CSIDriver`: one additional boolean field.

###### Will enabling / using this feature result in increasing time taken by any operations covered by existing SLIs/SLOs?

Enhancement 1 adds a bin-packing check for multi-pool (small for typical PVC counts). Enhancement 2 adds a map lookup
in the Filter phase (negligible). Enhancement 3 adds node lookups in checkBoundClaims (negligible). Overall impact on
scheduling latency is expected to be minimal.

###### Will enabling / using this feature result in non-negligible increase of resource usage (CPU, RAM, disk, IO, ...) in any components?

Enhancement 2 adds an in-memory map in the scheduler. Size is proportional to the number of `CSIStorageCapacity` objects
(one entry per UID). For typical clusters this is negligible.

###### Can enabling / using this feature result in resource exhaustion of some node resources (PIDs, sockets, inodes, etc.)?

No.

### Troubleshooting

###### How does this feature react if the API server and/or etcd is unavailable?

Enhancement 2 continues to work with cached data (reservations are in-memory). Enhancements 1 and 3 cannot read new API
fields but existing cached data is used. Running pods are not impacted.

###### What are other known failure modes?

- **Capacity reservation drift:** If informer updates are delayed, reservations may persist longer than necessary,
  causing false negatives (nodes incorrectly filtered out). Mitigation: informer updates eventually clear reservations;
  scheduler restart clears all state.

- **CSI driver does not handle rebuild:** If a CSI driver sets `volumeRebuilding: true` but does not actually rebuild
  when the volume is requested on a new node, the pod will be scheduled but volume mount will fail. Mitigation: this is
  a driver bug; the capability field is an explicit opt-in contract.

- **CSI driver does not unset nodeAffinity on node failure:** The pod will remain Pending because no other node passes
  the PV nodeAffinity check. Mitigation: this is a safe failure mode; documented as part of the driver contract.

###### What steps should be taken if SLOs are not being met to determine the problem?

Check kube-scheduler logs with increased verbosity for the VolumeBinding plugin. Review the new metrics for anomalies
(e.g., high reservation counts without corresponding resets).

## Implementation History

- 2026-02-16: Initial KEP draft

## Drawbacks

- **CSI spec change required.** Enhancement 1 requires a change to the CSI specification (`GetCapacityResponse`). This
  adds cross-community coordination overhead and may take longer to merge than a Kubernetes-only change.

- **Capacity reservation is conservative.** Reserving the entire `CSIStorageCapacity` object limits scheduling throughput
  to one pod per external-provisioner update interval per object. For clusters with high pod churn and few topology
  segments, this may be too aggressive. However, partial deduction is not possible because the scheduler cannot predict
  how the CSI driver maps storage to pools.

- **Rescheduling requires CSI driver changes.** Drivers must opt in, manage PV nodeAffinity via KEP-5381, and handle
  volume rebuild when the volume is requested on a new node. This limits initial adoption to drivers designed for this
  use case (e.g., Longhorn, replicated storage systems).

## Alternatives

TBD
