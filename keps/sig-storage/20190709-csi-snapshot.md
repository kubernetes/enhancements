---
title: CSI Snapshot
authors:
  - "@jingxu97"
  - "@xing-yang"
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
- [Changes](#changes)
  - [Changes Implemented](#changes-implemented)
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
  * Add metrics support
* Unit and e2e tests implemented
* Update snapshot CRDs to v1beta1 and enable VolumeSnapshotDataSource feature gate by default.

### Beta->GA

* Snapshot feature is used as a basic building block in other advanced applications.
* Feature deployed in production and have gone through at least one K8s upgrade.

## Changes

### Changes Implemented

Here are the changes since the original design proposal:

* Renamed `Ready` to `ReadyToUse` in the `Status` field of `VolumeSnapshot` API object.
* Changed type of `RestoreSize` in `CSIVolumeSnapshotSource` from `*resource.Quantity`  to `*int64`.
* Lease based Leader Election support is added.
* Added `VolumeSnapshotContent` deletion policy which is also specified in `VolumeSnapshotClass`.
* Added `VolumeSnapshot` and `VolumeSnapshotContent` in Use Protection using Finalizers.
* Added Finalizer on the snapshot source PVC to prevent it from being deleted when a snapshot is being created from it.
* Added check to see whether ListSnapshots is supported by the CSI driver. If it is supported, ListSnapshots will be called to find out the status of a snapshot during static binding; otherwise it is assumed the snapshot ID provided by the admin is valid.

### Work in Progress

There are other things we are working on for Beta:

* If snapshot creation times out, VolumeSnapshot status will not be marked as failed so that controller will continue to retry to create until the operation either succeeds or fails. It is up to the user or an upper level application that uses the VolumeSnapshot to determine what to do with the snapshot. This work is on-going.
* Investigation is on-going to determine whether we should separate common controller logic from other logic that belongs to the sidecar. Can be in the same external-snapshotter repo. The common controller should not be deployed with the driver. It should be deployed by the cluster deployer, or we can provide a way to deploy it as a separate Statefulset, not together with the driver. CRD installation should also be done by the cluster deployer.
* Add secrets field to list-snapshots RPC in the CSI spec. Add “data-source-secret” in create-volume intended for accessing the data source. Implement them in external-snapshotter and external-provisioner.
* Add metrics support in the snapshot controller.
  * operational end to end latency metrics.
    labels:
    * operation_name, i.e., creation-snapshot, delete-snapshot
    * csi-driver-name
  * operation error count.
    labels:
    * operation_name, i.e., creation-snapshot, delete-snapshot
    * csi-driver-name
* Update snapshot CRDs to v1beta1 and enable VolumeSnapshotDataSource feature gate by default.

## Implementation History

Volume snapshot is implemented as alpha feature in this repo in Kubernetes v1.12:
https://github.com/kubernetes-csi/external-snapshotter

Feature gate is added by this PR:
https://github.com/kubernetes/kubernetes/pull/67087
