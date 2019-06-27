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
  - name: "@saad-ali"
editor: TBD
creation-date: 2019-01-30
last-updated: 2019-02-01
see-also:
  - https://github.com/kubernetes/community/blob/master/contributors/design-proposals/storage/raw-block-pv.md
status: implementable
---

# CSI Raw Block Volumes

## Table of Contents

<!-- toc -->

* [Summary](#summary)
* [Motivation](#motivation)
  * [Goals](#goals)
  * [Non-goals](#non-goals)
* [Proposal](#proposal)
* [Test Plan](#test-plan)
   * [Unit tests](#unit-tests)
   * [E2E tests](#e2e-tests)
* [Graduation Criteria](#graduation-criteria)
   * [Alpha -&gt; Beta](#alpha---beta)
   * [Beta -&gt; GA](#beta---ga)
* [Implementation History](#implementation-history)
   * [K8s 1.12: Alpha](#k8s-112-alpha)
   * [K8s 1.13: Alpha](#k8s-113-alpha)

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

## Implementation History

### K8s 1.12: Alpha
* Initial implementation

### K8s 1.13: Alpha
* Multiple third party plugins start to implement the raw block CSI interface
and we can test with them
* Bugs fixed in CSI sidecars 
