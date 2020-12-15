---
title: Online Growing Persistent Volume Size
authors:
  - "@mlmhl"
  - "@wongma7"
owning-sig: sig-storage
participating-sigs:
  - sig-storage
reviewers:
  - "@gnufied"
  - "@jsafrane"
approvers:
  - "@childsb"
editor: TBD
creation-date: 2019-01-25
last-updated: 2019-02-01
status: implementable
see-also:
  - "https://github.com/kubernetes/community/blob/master/contributors/design-proposals/storage/grow-volume-size.md"
  - "https://github.com/kubernetes/community/pull/1535"
replaces:
superseded-by:
---

# Online Growing Persistent Volume Size

## Table of Contents

<!-- toc -->
- [Summary](#summary)
- [Motivation](#motivation)
  - [Goals](#goals)
  - [Non-Goals](#non-goals)
- [Proposal](#proposal)
  - [User Stories](#user-stories)
    - [Story 1](#story-1)
    - [Story 2](#story-2)
  - [Notes](#notes)
  - [Implementation Details](#implementation-details)
  - [Risks and Mitigations](#risks-and-mitigations)
- [Test Plan](#test-plan)
- [Graduation Criteria](#graduation-criteria)
  - [Alpha to Beta](#alpha-to-beta)
  - [Beta to GA](#beta-to-ga)
- [Implementation History](#implementation-history)
<!-- /toc -->

## Summary

This feature enables users to expand a volume's file system by editing a PVC without having to restart a pod using the PVC.

## Motivation

Release 1.10 only supports offline file system resizing for PVCs, as this operation is only executed inside the `MountVolume` operation in kubelet. If a resizing request was submitted after the volume was mounted, it won't be performed. This proposal's intent is to support online file system resizing for PVCs in kubelet.

### Goals

Enable users to increase the size of a PVC which is already in use (mounted). The user will update PVC to request a new size. Underneath we expect that kubelet will resize the file system for the PVC accordingly.

### Non-Goals

* Offline file system resizing is not included. If we find a volume needs file system resizing but is not mounted to the node yet, we will do nothing. This situation will be dealt with by the existing [offline file system resizing handler](https://github.com/kubernetes/community/blob/master/contributors/design-proposals/storage/grow-volume-size.md).

* Extending resize tools: we only support the most common file systems' offline resizing in current release, and we prefer to stay the same for online resizing: ext3, ext4, & xfs.

## Proposal

### User Stories

#### Story 1

* As a user I am running MySQL on a 100GB volume - but I am running out of space. I should be able to increase size of volume MySQL is using without losing all my data. (online and with data)

#### Story 2

* As a user I am running an application with a PVC. I should be able to resize the volume without losing data or mount point. (online and with data and without taking pod offline)

### Notes

- Currently we only support offline resizing for `xfs`, `ext3`, `ext4`. Online resizing of `ext3`,`ext4` was introduced in [Linux kernel-3.3](https://www.ibm.com/developerworks/library/l-33linuxkernel/), and `xfs` has always supported growing mounted partitions (in fact, currently there is no way to expand an unmounted `xfs` file system), so they are all safe for online resizing. If a user tries to expand a volume with other formats an error event will be reported for the pod using it.

- This feature is protected by an alpha feature gate `ExpandOnlinePersistentVolumes` in v1.11. We separate this feature gate from the offline resizing gate `ExpandPersistentVolumes`, if a user wants to enable this feature, `ExpandPersistentVolumes` must be enabled first.

### Implementation Details

The key point of online file system resizing is how kubelet discovers which PVCs need file system resizing. We achieve this goal by reusing the reprocess mechanism of `VolumeManager`'s `DesiredStateOfWorldPopulator`. kubelet synchronizes pods periodically, and during each loop, `DesiredStateOfWorldPopulator` is called to reprocess each pod's volumes, where it ensures they are in `DesiredStateOfWorld`.

When adding a volume to `DesiredStateOfWorld`, the populator will include the PV & PVC fields needed to check whether the volume requires an online resizing operation: one is required if `PVC.Status.Capacity` is less than `PV.Spec.Capacity`.

Later, `VolumeManager`'s `Reconciler` mounts the volume in `DesiredStateOfWorld` to a pod and adds it to `ActualStateOfWorld`. It will copy the `PVC.Status.Capacity` field from the `DesiredStateOfWorld` representation to the `ActualStateOfWorld` one.

The reconciler will periodically check whether any volume that's mounted in a pod (as reported by `ActualStateOfWorld`) requires an online resizing operation by reading its fields and triggering an online file system resize operation if: the volume's `PVC.Status.Capacity` in `ActualStateOfWorld` is less than its `PV.Spec.Capacity` in `DesiredStateOfWorld`, the volume has `volumeMode` `Filesystem`, and the volume is not `readOnly`.

If the file system resizing operation succeeds, `PVC.Status.Capacity` is changed to the desired volume size in `PV.Spec.Capacity`, and the `PersistentVolumeClaimFileSystemResizePending` condition removed from `PVC.Status.Conditions`. The `PVC.Status.Capacity` in `ActualStateOfWorld` is changed too. The `PVC.Status.Capacity` in `DesiredStateOfWorld` will be changed by the populator the next time it reprocesses the pod. (If the same volume is mounted again to another pod before that, a no-op resize may be triggered).

If it fails, the reconciler retries.

It is important to note that:

- The reconciler triggers resize operations only for volumes that are mounted to pods by reading `ActualStateOfWorld`, which should only be read/written to by reconciler and the operations it starts. operationExecutor's goroutinemap prevents resize operations and other operations (e.g. unmount) from happening simultaneously. 

- File system resizing is a global operation for a volume, so if more than one pod mounts the same PVC, the reconciler should avoid triggering unnecessary no-op operations. (The alpha implementation used an in-memory map but it might also be possible to change `PVC.Status.Capacity` for every `mountedPod` of an `attachedVolume`.)

- Since we support only `xfs` and `ext3/4`, we needn't worry about exotic file systems that can be attached/mounted to multiple nodes at the same time and require a resizing operation on a node, such as [gfs2](https://access.redhat.com/documentation/en-us/red_hat_enterprise_linux/7/html/global_file_system_2/s1-manage-growfs).

### Risks and Mitigations

- All in-tree volume plugins that support offline file system expansion support online file system expansion:
  - GCE PD
  - Azure Disk
  - AWS EBS
  - RBD
  - Cinder

However, Azure Disk in particular does not support online "device" expansion, so in practice a pod needs to be restarted and the device reattached for resize to happen regardless of whether this feature is enabled. File system expansion follows "device"/"controller"/"cloud provider" expansion (i.e. file system expansion is the resize2fs call to enlarge a file system to fill a device, and device expansion is the API call to expand the device in the first place). This difference is surfaced to the user via the "FileSystemResizePending" and "Resizing" conditions, for when file system or cloud provider expansion are in progress respectively. No additional conditions or configuration options will be surfaced to account for this case of online device expansion being unsupported: it is left to the volume plugin implementor & cloud provider to document it and return meaningful errors, which are displayed as events on PVCs.

- Offline resize was introduced first and requires a pod restart to take effect, so online resize becoming the default and occurring immediately upon a PVC edit without requiring a restart may come as a surprise to users. But expansion of mounted file systems is supported by the kernel, so all applications will tolerate it. Plus, if there are cases where users would prefer their volumes to not resize immediately, they can still force offline resize to happen by stopping their pod before editing the PVC, then restarting it. Online resizing will be made default.

- Enabling support of volume expansion through stateful sets (https://github.com/kubernetes/enhancements/issues/661) is orthogonal to making online resizing default. Stateful apps/workloads may need to restart to detect file systems' size changes but should have no issue with online resize.

## Test Plan

There will be e2e tests for online resizing. To isolate vs offline resize, the test will edit the PVC while the Pod is started instead of before.

## Graduation Criteria

### Alpha to Beta

- e2e test where the PVC is edited while the Pod is started instead of before
- update existing e2e tests so that they test offline resize correctly whether the online resize feature is enabled or not.
- rename the operation metric to "online_node_expansion"

### Beta to GA

- Time for feedback (at least 1 release)
- stress tests where multiple Pods are started/stopped & PVCs edited simultaneously to find race conditions in volume manager

## Implementation History

- 1.11: alpha
- 1.15: beta
