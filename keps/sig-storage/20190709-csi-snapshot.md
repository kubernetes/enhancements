---
title: CSI Snapshot
authors:
  - "@jingxu97"
  - "@xing-yang"
  - "@yuxiangqian"
owning-sig: sig-storage
participating-sigs:
  - sig-storage
reviewers:
  - "@msau42"
  - "@saad-ali"
  - "@thockin"
approvers:
  - "@msau42"
  - "@saad-ali"
  - "@thockin"
editor: TBD
creation-date: 2019-07-09
last-updated: 2019-07-29
status: implementable
see-also:
  - n/a
replaces:
  - n/a
superseded-by:
  - n/a
---

# Title

CSI Snapshot

## Table of Contents

<!-- toc -->
- [Summary](#summary)
- [Test Plan](#test-plan)
  - [Unit tests](#unit-tests)
  - [E2E tests](#e2e-tests)
- [Graduation Criteria](#graduation-criteria)
  - [Alpha-&gt;Beta](#alpha-beta)
  - [Beta-&gt;GA](#beta-ga)
- [Snapshot Beta](#snapshot-beta)
  - [API Changes](#api-changes)
  - [Controller Split](#controller-split)
  - [Other Changes Implemented](#other-changes-implemented)
  - [Work in Progress](#work-in-progress)
- [Implementation History](#implementation-history)
<!-- /toc -->

## Summary

This KEP is written after the original design doc has been approved and implemented. Design for CSI Volume Snapshot Support in Kubernetes is incorporated as part of the [CSI Volume Snapshot in Kubernetes Design Doc](https://github.com/kubernetes/community/blob/master/contributors/design-proposals/storage/csi-snapshot.md).

The rest of the document includes required information missing from the original design document: test plan and graduation criteria.

## Test Plan

### Unit tests

* Unit tests around snapshot creation and deletion logic.
* Unit tests around VolumeSnapshot and VolumeSnapshotContent binding logic.
* Unit tests for creating volume from snapshot.

### E2E tests

* (P0) e2e tests for creating/deleting snapshot.
* (P0) e2e tests for creating volume from snapshot.
* (P1) e2e tests for delete/retain policy.
* (P1) e2e tests for deleting API objects out of order (snapshot protection).
* (P2) e2e tests for secret fields.
* (P2) e2e tests for metrics.

## Graduation Criteria

### Alpha->Beta

* Feature complete, including:
  * Create/delete volume snapshots
  * Create new volumes from a snapshot
  * SnapshotContent Deletion/Retain Policy
  * Snapshot Object in Use Protection
  * Separate the common controller from the sidecar controller
  * Add secrets field to list-snapshots RPC in the CSI spec. Add “data-source-secret” in create-volume intended for accessing the data source. Implement them in external-snapshotter and external-provisioner.
  * Add metrics support to the external-snapshotter sidecar.
* Unit and e2e tests implemented
* Update snapshot CRDs to v1beta1 and enable VolumeSnapshotDataSource feature gate by default.

### Beta->GA

* Snapshot feature is used as a basic building block in other advanced applications. 
* Feature deployed in production and have gone through at least one K8s upgrade.

## Snapshot Beta

### API Changes

A number of changes were made to the Kubernetes volume snapshot API between alpha to beta. These changes are not backward compatible and the alpha API is no longer supported. The purpose of these changes was to make API definitions more clear and easier to use.

The following changes have been made from the Alpha API:

* DeletionPolicy is now a required field rather than optional in both VolumeSnapshotClass and VolumeSnapshotContent. This way the user has to explicitly specify it, leaving no room for confusion.
* VolumeSnapshotSpec has a new required Source field. Source may be either a PersistentVolumeClaimName (if dynamically provisioning a snapshot) or VolumeSnapshotContentName (if pre-provisioning a snapshot).
* VolumeSnapshotContentSpec also has a new required Source field. This Source may be either a VolumeHandle (if dynamically provisioning a snapshot) or a SnapshotHandle (if pre-provisioning volume snapshots).
* VolumeSnapshotStatus now contains a BoundVolumeSnapshotContentName to indicate the VolumeSnapshot object is bound to a VolumeSnapshotContent.
* VolumeSnapshotContent now contains a Status to indicate the current state of the content. It has a field SnapshotHandle which is the unique identifier of a snapshot on the storage system.

The beta VolumeSnapshot API object is as follows:

```
type VolumeSnapshot struct {
        metav1.TypeMeta
        metav1.ObjectMeta

        Spec VolumeSnapshotSpec
        Status *VolumeSnapshotStatus
}
```

```
type VolumeSnapshotSpec struct {
	Source VolumeSnapshotSource
	VolumeSnapshotClassName *string
}
// Exactly one of its members MUST be specified
type VolumeSnapshotSource struct {
	// +optional
	PersistentVolumeClaimName *string
	// +optional
	VolumeSnapshotContentName *string
}
```

```
type VolumeSnapshotStatus struct {
	BoundVolumeSnapshotContentName *string
	CreationTime *metav1.Time
	ReadyToUse *bool
	RestoreSize *resource.Quantity
	Error *VolumeSnapshotError
}
```

The beta VolumeSnapshotContent API object is as follows:

```
type VolumeSnapshotContent struct {
        metav1.TypeMeta
        metav1.ObjectMeta

        Spec VolumeSnapshotContentSpec
        Status *VolumeSnapshotContentStatus
}
```

```
type VolumeSnapshotContentSpec struct {
         VolumeSnapshotRef core_v1.ObjectReference
         Source VolumeSnapshotContentSource
         DeletionPolicy DeletionPolicy
         Driver string
         VolumeSnapshotClassName *string
}
```

```
type VolumeSnapshotContentSource struct {
	// +optional
	VolumeHandle *string
	// +optional
	SnapshotHandle *string
}
```

```
type VolumeSnapshotContentStatus struct {
  CreationTime *int64
  ReadyToUse *bool
  RestoreSize *int64
  Error *VolumeSnapshotError
  SnapshotHandle *string
}
```

The beta Kubernetes VolumeSnapshotClass API object is as follows:

```
type VolumeSnapshotClass struct {
        metav1.TypeMeta
        metav1.ObjectMeta

        Driver string
        Parameters map[string]string
        DeletionPolicy DeletionPolicy
}
```

### Controller Split

Along with VolumeSnapshot being promoted to Beta in Kubernetes 1.17, the CSI external-snapshotter sidecar controller has been split into two controllers: a snapshot-controller and a CSI external-snapshotter sidecar.

The snapshot controller will be watching the Kubernetes API server for `VolumeSnapshot`, `VolumeSnapshotContent`, and `VolumeSnapshotClass` CRD objects. The CSI `external-snapshotter` sidecar watches the Kubernetes API server for `VolumeSnapshotContent` and `VolumeSnapshotClass` CRD objects.

For dynamic provisioning, the creation of a new `VolumeSnapshot` object referencing a `VolumeSnapshotClass` CRD object corresponding to this driver causes the snapshot controller to trigger the creation of a Kubernetes `VolumeSnapshotContent` object to represent the to-be-created new snapshot.

The creation of a new `VolumeSnapshotContent` object causes the sidecar container to trigger a `CreateSnapshot` operation against the specified CSI endpoint to provision a new snapshot. When a new snapshot is successfully provisioned, the sidecar container updates the status field of the `VolumeSnapshotContent` object to represent the new snapshot.

The snapshot controller will be updating the status field of the `VolumeSnapshot` object accordingly based on the status field of the `VolumeSnapshotContent` object to indicate the new snapshot is ready to be used or failed.

The deletion event of a `VolumeSnapshot` object bound to a `VolumeSnapshotContent` corresponding to this driver with a `delete` deletion policy causes the snapshot controller to start deleting the `VolumeSnapshotContent` object and add an annotation to the object to indicate it is being deleted. Note that both the `VolumeSnapshot` object and the `VolumeSnapshotContent` object will not be deleted immediately due to the finalizers. When the sidecar container detects this update on the `VolumeSnapshotContent` object, it triggers a `DeleteSnapshot` operation against the specified CSI endpoint to delete the snapshot. Once the snapshot is successfully deleted, the sidecar container removes the finalizer on the `VolumeSnapshotContent` object which leads to the deletion of the object from Kubernetes. The snapshot controller then removes the finalizer on the `VolumeSnapshot` object and as a result the object will be deleted from Kubernetes. If a user deletes a bound `VolumeSnapshotContent` object directly, it will have a deletion timestamp set however will persist in API server until its corresponding `VolumeSnapshot` object also gets a deletion timestamp set from a deletion request.

If the deletion policy is `retain` when deleting a `VolumeSnapshot` object bound to a `VolumeSnapshotContent`, the finalizers will be removed from both objects, the `VolumeSnapshot` object will be deleted from Kubernetes, but the `VolumeSnapshotContent` and the snapshot on the storage system will remain.

### Other Changes Implemented

Here are the changes since the original design proposal:

* Renamed `Ready` to `ReadyToUse` in the `Status` field of `VolumeSnapshot` API object.
* Changed type of `RestoreSize` in `CSIVolumeSnapshotSource` from `*resource.Quantity`  to `*int64`.
* Lease based Leader Election support is added.
* Added `VolumeSnapshotContent` deletion policy which is also specified in `VolumeSnapshotClass`.
* Added Finalizer on the snapshot source PVC to prevent it from being deleted when a snapshot is being created from it.
* Added Finalizer on the `VolumeSnapshotContent` object to prevent it from being deleted directly from API server when it is bound to the `VolumeSnapshot` object.
* Added Finalizer on the `VolumeSnapshot` object to prevent it from being deleted when it is being used as a source to create a PVC.
* Added Finalizer on the `VolumeSnapshot` object to prevent it from being deleted when it is bound to the `VolumeSnapshotContent` object.
* Added check to see whether ListSnapshots is supported by the CSI driver. If it is supported, ListSnapshots will be called to find out the status of a snapshot during static binding; otherwise it is assumed the snapshot ID provided by the admin is valid.
* Added deletion secret as annotation to volume snapshot content.
* Added prometheus metrics to CSI external-snapshotter under the /metrics endpoint.
* Removed createSnapshotContentRetryCount and createSnapshotContentInterval
from command line options.
* Added a prefix "external-snapshotter-leader" and the driver name to the snapshotter leader election lock name. Rolling update from an earlier version to v2.0.0 will not work if leader election is enabled because the lock name is changed in v2.0.0.

### Work in Progress

There are other things we are working on before promoting the snapshot feature to GA:

* If snapshot creation times out, VolumeSnapshot status will not be marked as failed so that controller will continue to retry to create until the operation either succeeds or fails. It is up to the user or an upper level application that uses the VolumeSnapshot to determine what to do with the snapshot. This work is on-going.
* Add secrets field to list-snapshots RPC in the CSI spec. Add “data-source-secret” in create-volume intended for accessing the data source. Implement them in external-snapshotter and external-provisioner.
* Add metrics support in the snapshot controller (metrics is already added to the external-snapshotter sidecar).
  * operational end to end latency metrics.
    labels:
    * operation_name, i.e., creation-snapshot, delete-snapshot
    * csi-driver-name
  * operation error count.
    labels:
    * operation_name, i.e., creation-snapshot, delete-snapshot
    * csi-driver-name

## Implementation History

K8s 1.12: alpha
K8s 1.17: beta
Repo: https://github.com/kubernetes-csi/external-snapshotter
