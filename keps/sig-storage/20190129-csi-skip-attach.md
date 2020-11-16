---
title: Skip attach for non-attachable CSI volumes
authors:
  - "@jsafrane"
owning-sig: sig-storage
participating-groups:
  - sig-storage
reviewers:
  - "@msau42"
  - "@saad-ali"
approvers:
  - "@saad-ali"
editor: TBD
creation-date: 2019-01-29
last-updated: 2020-03-09
status: implemented
see-also:
  - "https://github.com/kubernetes/community/blob/master/contributors/design-proposals/storage/container-storage-interface-skip-attach.md"
replaces:
superseded-by:
---

# Skip attach for non-attachable CSI volumes

## Table of Contents

<!-- toc -->
- [Summary](#summary)
- [Test Plan](#test-plan)
- [Graduation Criteria](#graduation-criteria)
  - [Alpha -&gt; Beta](#alpha---beta)
  - [Beta -&gt; GA](#beta---ga)
- [Implementation History](#implementation-history)
<!-- /toc -->

## Summary

This document presents a design to be able to skip the attach/detach flow in
Kubernetes for CSI plugins that don't support attaching.

The detailed design was originally implemented as a [design
proposal](https://github.com/kubernetes/community/blob/master/contributors/design-proposals/storage/container-storage-interface-skip-attach.md).

This KEP contains details that are missing from the design proposal.

## Test Plan
* Unit tests in attach detach controller
* Integration tests:
   * A VolumeAttachment object is not created for CSI drivers that don't
     support attach
   * A VolumeAttachment object is created for CSI drivers that do
     support attach
   * A VolumeAttachment object is created for CSI drivers that did not
     specify attach support
* E2E tests:
    * Drivers that don't support attach don't need the external-attacher and can
      mount volumes successfully
    * Drivers that don't require attach but has missing CSIDriver objects should successfully
      mount volumes without attachment(and no external-attacher running) if CSIDriver
      object is created later on.

## Graduation Criteria

### Alpha -> Beta
* Basic unit and e2e tests as outlined in the test plan.

### Beta -> GA
* Users deployed in production and have gone through at least one K8s upgrade.

## Implementation History
* K8s 1.12: Alpha implementation
* K8s 1.14: Beta implementation
* K8s 1.18: GA implementation
