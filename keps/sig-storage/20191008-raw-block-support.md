---
title: Raw Block Volumes
authors:
  - "@jsafrane"
owning-sig: sig-storage
participating-sigs:
  - sig-storage
reviewers:
  - "@msau42"
  - "@saad-ali"
approvers:
  - "@saad-ali"
editor: TBD
creation-date: 2019-10-08
last-updated: 2019-15-10
status: implementable
see-also:
  - "https://github.com/kubernetes/community/blob/master/contributors/design-proposals/storage/raw-block-pv.md"
---

# Raw Block Volumes

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
  - [K8s 1.9: Alpha](#k8s-19-alpha)
  - [K8s 1.13: Beta](#k8s-113-beta)
  - [K8s 1.17: GA](#k8s-117-ga)
<!-- /toc -->

## Summary

This document presents a proposal for managing raw block storage in Kubernetes
using the persistent volume source API as a consistent model of consumption.

Note that is has been designed & merged before KEP process was introduced:
https://github.com/kubernetes/community/blob/master/contributors/design-proposals/storage/raw-block-pv.md

## Motivation

By extending the API for volumes to specifically request a raw block device,
we provide an explicit method for volume consumption, whereas previously any
request for storage was always fulfilled with a formatted filesystem, even when
the underlying storage was block. In addition, the ability to use a raw block
device without a filesystem will allow Kubernetes better support of high
performance applications that can utilize raw block devices directly for their
storage. Block volumes are critical to applications like databases (MongoDB,
Cassandra) that require consistent I/O performance and low latency. For mission
critical applications, like SAP, block storage is a requirement.

### Goals

* Enable durable access to block storage
* Provide flexibility for users/vendors to utilize various types of storage devices
* Agree on API changes for block
* Provide a consistent security model for block devices 
* Provide a means for running containerized block storage offerings as non-privileged container

### Non-Goals

* Support all storage devices natively in upstream Kubernetes. Non-standard storage devices are expected to be managed using extension
  mechanisms.
* Provide a means for full integration into the scheduler based on non-storage related requests (CPU, etc.)
* Provide a means of ensuring specific topology to ensure co-location of the data 
* CSI volume plugin changes - CSI block volumes are tracked as a separate KEP.

## Proposal

See original design proposal at
https://raw.githubusercontent.com/kubernetes/community/master/contributors/design-proposals/storage/raw-block-pv.md

## Upgrade/Downgrade Strategy

These situations can happen when various Kubernetes components run with raw block volume feature gate on/off:

* API server on, controller-manager on, kubelet off: kubelet does not see
  `volumeDevices` section in pods and thus it will run the pods as if the pods
  did not use the volume at all. Kubelet should not touch block volumes at all,
  especially should not format / mount / resize them.
* API server on, controller-manager off: PV controller will bind / provision
  filesystem PVs to PVCs that request block volumes.
* API server off: all newly created PVs / PVCs / Pods that refer to a block
  volume are rejected by validation. Older objects may keep the field,
  however, the field is ignored by controller-manager and kubelet as in the
  previous case.

As result, cluster admins are responsible for deleting any pods that use block
volumes and all block PV/PVCs before downgrading to an older release / disabling
the feature gate.

## Test Plan

### Unit tests

* Kubelet VolumeManager can provide a block volume using a (fake) volume plugin
  to (fake) container runtime.
* PV controller can bind block PVs to block PVCs.
* PV controller can provision block PVs to block PVCs.

### E2E tests

* Implement the same e2e tests as we have for filesystem volumes.

## Graduation Criteria

### Alpha -> Beta
Already happened.

### Beta -> GA
* Implement missing e2e tests:
  * Mismatched usage of filesystem / block volumes (#79796).
  * Block volume reconstruction after kubelet restart (#83451).
* Implement reconstruction of local volumes.
* Manual stress test with at least block volume plugin is performed with a node
  running non-trivial amount of pods that use a block device (simple `dd`
  should do).
  * Test that the pods are moved to another nodes and data is retained when
    the node is drained.
  * Test that the data is retained when kubelet restarts while the pods are
    running.
  * Test that the data is retained when the node reboots. For pods that were
    moved to another nodes during the outage, test that the newly started node
    cleaned up their devices.
* Manual test with API server with enabled block volume feature gate and
  kubelet with the gate disabled.

## Implementation History

### K8s 1.9: Alpha
* Initial implementation.

### K8s 1.13: Beta
* Enhanced e2e tests.
* Most block-based volume plugins implemented block volume plugin interfaces.

### K8s 1.17: GA
* Fixed block volume reconstruction.
* Stress tests as noted above.
