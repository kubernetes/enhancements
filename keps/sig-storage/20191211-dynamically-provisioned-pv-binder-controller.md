---
title: Dynamically Provisioned PersistVolume Binding Controller
authors:
  - "@answer1991"
owning-sig: sig-storage
participating-sigs:
reviewers:
approvers:
creation-date: 2019-12-11
last-updated: 2019-12-11
status: provisional

---

# Dynamically Provisioned PersistVolume Binding Controller

## Table of Contents

<!-- toc -->
- [Summary](#summary)
- [Motivation](#motivation)
  - [Goals](#goals)
  - [Non-Goals](#non-goals)
- [Proposal](#proposal)
  - [User Stories](#user-stories)
    - [Efficiently Binding/Unbinding for CSI PersistVolume](#efficiently-bindingunbinding-for-csi-persistvolume)
  - [Risks and Mitigations](#risks-and-mitigations)
- [Design Details](#design-details)
  - [API Changes](#api-changes)
    - [volume.beta.kubernetes.io/storage-provisioner](#volumebetakubernetesiostorage-provisioner)
  - [Admission](#admission)
  - [Controller](#controller)
- [Implementation History](#implementation-history)
<!-- /toc -->

## Summary

Dynamically Provisioned PersistVolume Binding Controller which cloud be ran in multi-workers is responsible for
binding/unbinding the dynamically provisioned PersistVolume and PersistVolumeClaim efficiently, 
typically for CSI PersistVolume/PersistVolumeClaim.

## Motivation

For now, [Persistent Binding Controller](https://github.com/kubernetes/kubernetes/blob/master/pkg/controller/volume/persistentvolume/pv_controller_base.go) 
is responsible for all PersistVolume and PersistVolumeClaim binding/unbinding. 
However, Persistent Binding Controller is [single-worker](https://github.com/kubernetes/kubernetes/blob/eef4c00ae93bd51125c215918a7b1c632d298610/pkg/controller/volume/persistentvolume/pv_controller_base.go#L304) controller,
we meet the [performance issue](https://github.com/kubernetes/kubernetes/issues/83178) when we implemented a high performance CSI Provisioner. 
Which means even we developed a high performance CSI Provisioner, 
the Persistent Binding Controller will block us delivering PersistVolumes efficiently as the PersistVolume/PersistVolumeClaims' status can not be updated to be `Bound` in time.

PersistVolumeClaim that is supposed to be dynamically provisioned is created to wait for its own PersistVolume provisioned, 
and it has no requirement to select an appropriate PersistVolume from all existing PersistVolumes. 
They are also called pre-bound PersistVolume, and the controller can processing their binding/unbinding in multi-workers.

### Goals

- Adding an annotation to difference the dynamically provisioned PersistVolumeClaim/PersistVolume during creation according to the StorageClass.
- Adding a new multi-workers controller to be responsible for binding/unbinding the dynamically provisioned PersistVolume and PersistVolumeClaim.

### Non-Goals

- Deprecate and remove Persistent Binding Controller is NOT goal.

## Proposal

An Admission will be added, which will add annotation for dynamically provisioned PersistVolumeClaims. 
The annotation key is [`volume.beta.kubernetes.io/storage-provisioner`](https://github.com/kubernetes/kubernetes/blob/eef4c00ae93bd51125c215918a7b1c632d298610/pkg/controller/volume/persistentvolume/util/util.go#L70), 
value is the related StorageClass's field `Provisoner` value.
The provisioner can provision PersistVolumes immediately when PersistVolumeClaims created, 
sometimes provisioner will wait for `volume.kubernetes.io/selected-node` annotation which will be added by Scheduler.

An Controller will be added which can run as multi-works to processing dynamically provisioned PersistVolume/PersistVolumeClaim.
PersistVolumeClaim that is supposed to be dynamically provisioned is annotated with `volume.beta.kubernetes.io/storage-provisioner`,
and dynamically provisioned PersistVolume is annotated with `pv.kubernetes.io/provisioned-by`. 
[Persistent Binding Controller](https://github.com/kubernetes/kubernetes/blob/master/pkg/controller/volume/persistentvolume/pv_controller_base.go) will skip to process dynamically provisioned PersistVolume/PersistVolumeClaim. 

### User Stories

#### Efficiently Binding/Unbinding for CSI PersistVolume

The following tables shows the latency of binding/unbinding 100 groups of CSI PersistVolume/PersistVolumeClaim(s).

|               | Persistent Binding Controller | Dynamically Provisioned PersistVolume Binding Controller(512 workers) |
| ------------  | ----------------------------  | --------------------------------------------------------------------- |
| P95 Binding   |      10.533414s               |                                   2.611919s                           |
| Avg Binding   |      17.308238s               |                                   1.71052218s                         |
| P95 Release   |      1.316980253s             |                                   4.335248ms                          |
| Avg Release   |      1.067353673s             |                                   8.628078ms                          |

### Risks and Mitigations

TBD.

## Design Details

### API Changes

#### volume.beta.kubernetes.io/storage-provisioner

The [`volume.beta.kubernetes.io/storage-provisioner`](https://github.com/kubernetes/kubernetes/blob/eef4c00ae93bd51125c215918a7b1c632d298610/pkg/controller/volume/persistentvolume/util/util.go#L70)
will be added during creation by the Admission to the PersistVolumeClaim if it's supposed to be dynamically provisioned.

### Admission

TBD

### Controller

TBD

## Implementation History

- 2019-12-12: Initial KEP sent out for review.