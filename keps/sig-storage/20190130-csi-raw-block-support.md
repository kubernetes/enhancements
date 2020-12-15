---
title: CSI Raw Block Volumes
authors:
  - "@bswartz"
owning-sig: sig-storage
participating-sigs:
  - sig-storage
reviewers:
  - "@msau42"
  - "@saad-ali"
approvers:
  - "@saad-ali"
editor: TBD
creation-date: 2019-01-30
last-updated: 2020-03-09
status: implemented
see-also:
  - "https://github.com/kubernetes/community/blob/master/contributors/design-proposals/storage/raw-block-pv.md"
  - "https://github.com/kubernetes/enhancements/pull/1288"
---

# CSI Raw Block Volumes

## Table of Contents

<!-- toc -->
- [Summary](#summary)
- [Motivation](#motivation)
  - [Goals](#goals)
  - [Non-Goals](#non-goals)
- [Proposal](#proposal)
- [Upgrade/Downgrade Strategy](#upgradedowngrade-strategy)
- [Test Plan](#test-plan)
  - [Unit tests](#unit-tests)
  - [E2E tests](#e2e-tests)
- [Graduation Criteria](#graduation-criteria)
  - [Alpha -&gt; Beta](#alpha---beta)
  - [Beta -&gt; GA](#beta---ga)
- [Implementation History](#implementation-history)
  - [K8s 1.12: Alpha](#k8s-112-alpha)
  - [K8s 1.13: Alpha](#k8s-113-alpha)
  - [K8s 1.14: Beta](#k8s-114-beta)
  - [K8s 1.17: Beta](#k8s-117-beta)
  - [K8s 1.18: GA](#k8s-118-ga)
<!-- /toc -->

## Summary

Raw block support has been added to both the core of Kubernetes (including
API support and support in kubelet for a subset of volume types) as well as
the CSI specification. This feature is specifically to add raw block support
to kubelet for volumes of the "csi" type by taking advantage of the
now-standardized CSI RPCs.

https://github.com/kubernetes/community/blob/master/contributors/design-proposals/storage/raw-block-pv.md

## Motivation

The future of persistent volume support should be mostly CSI-type volumes,
with a few notable exceptions. It's therefore essential that CSI volumes
can implement all of the features that Kubernetes has, including support
for raw block volumes.

### Goals

CSI plugins should be able to create, publish, and attach raw block volumes
in accordance with the raw block support APIs.

### Non-Goals

Raw block support added in any particular CSI driver.

## Proposal

Modify the CSI volume package of kubelet to declare support for raw block
volumes, and make the following changes to the csi mounter plugin:

For raw block volumes:
* Inside MountDevice, invoke NodeStageVolume() without a fsType, but with an
empty directory (just like file system volumes).
* Inside SetUp, invoke NodePublishVolume() with a device file name rather than
a directory name, and do not pass mount options, and expect the device to be
created by the node plugin. Kubelet ensures the parent directory of the file
name exists.
* The volume is passed to the container layer as an actual device rather than
a mount.
* On cleanup kubelet removes the directories it created, and assumes the node
plugin removed any device files it created.

This support is controlled by a feature gate called "CSIBlockVolume".

## Upgrade/Downgrade Strategy

CSIBlockVolume feature gate is used only in CSI volume plugin and only in
the parts that are used by kubelet (`BlockVolumeMapper` interface). A node
must be therefore drained before switching the feature off to remove all
raw devices provided to pods, because newly started kubelet (with the feature
off) won't be able to remove these devices.

When a node is not drained and CSIBlockVolume is disabled while running
a pod, we make sure that the pod is either killed or can continue
using the volume as before. In both cases, kubelet won't touch data
on the volume and can't corrupt it. The volume may not be cleaned after
the pod is deleted, i.e. some leftover symlinks / bind mounts may be
present. It's up to the cluster admin to clean these orphans
(or drain nodes properly before disabling the feature).

## Test Plan

### Unit tests

* Kubelet can generate staging/publish paths to send to the CSI plugins
* Test that fake CSI plugin stages and publishes to the correct paths
* Ensure that paths with and without dots work
* Test that teardown cleans up everything

### E2E tests

There are existing e2e tests for the raw block volume feature. To test CSI raw
block volumes we just need to configure a CSI plugin that has raw block support,
and run the existing raw block tests on that CSI plugin.

## Graduation Criteria

### Alpha -> Beta
* At least 1 CSI plugin that can run in the gate (hostpath or GCE-PD) supporting
raw block volumes so the feature can be regression tested.
* Enable e2e tests for raw block with CSI
* At least 2 third party CSI plugins support raw block with CSI and pass e2e
tests to confirm both that the interface is flexible enough to accommodate
multiple implementations and that it's specific enough to not leave any
ambiguity about how implementations should work.

### Beta -> GA
* Two or more CSI plugins that and run in the gate and test raw block
* At least 5 third party CSI plugins support raw block with CSI and pass e2e
tests
* Manual stress test with at least one real CSI driver is performed with a node
  running non-trivial amount of pods that use a block device (simple `dd`
  should do).
  * Test that the pods are moved to another nodes and data is retained when
    the node is drained.
  * Test that the data is retained when kubelet restarts while the pods are
    running.
  * Test that the data is retained when the node reboots. For pods that were
    moved to another nodes during the outage, test that the newly started node
    cleaned up their devices.

## Implementation History

### K8s 1.12: Alpha
* Initial implementation

### K8s 1.13: Alpha
* Multiple third party plugins start to implement the raw block CSI interface
and we can test with them
* Bugs fixed in CSI sidecars 

### K8s 1.14: Beta
* Bugs fixed

### K8s 1.17: Beta
* Separated NodeStage / NodePublish calls.
* Fixed volume reconstruction after kubelet restart.
* Stress tests as noted above.

### K8s 1.18: GA
* Added block tests to csi-sanity.
* Disruptive testing with block devices in /dev reordered after reboot.
