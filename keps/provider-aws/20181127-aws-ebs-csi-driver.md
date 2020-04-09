---
title: aws-ebs-csi-driver
authors:
  - "@leakingtapan"
owning-sig: sig-cloud-provider
reviewers:
  - "@d-nishi"
  - "@jsafrane"
approvers:
  - "@d-nishi"
  - "@jsafrane"
editor: TBD
creation-date: 2018-11-27
last-updated: 2019-01-27
status: provisional
---

# AWS Elastic Block Store (EBS) CSI Driver

## Table of Contents

<!-- toc -->
- [Summary](#summary)
- [Motivation](#motivation)
  - [Goals](#goals)
  - [Non-Goals](#non-goals)
- [Proposal](#proposal)
  - [User Stories](#user-stories)
    - [Static Provisioning](#static-provisioning)
    - [Dynamic Provisioning](#dynamic-provisioning)
    - [Volume Scheduling](#volume-scheduling)
    - [Mount Options](#mount-options)
    - [Raw Block Volume](#raw-block-volume)
    - [Offline Volume Resizing](#offline-volume-resizing)
  - [Risks and Mitigations](#risks-and-mitigations)
- [Graduation Criteria](#graduation-criteria)
- [Upgrade/Downgrade Process](#upgradedowngrade-process)
  - [Upgrade](#upgrade)
  - [Downgrade](#downgrade)
- [Implementation History](#implementation-history)
<!-- /toc -->

## Summary
AWS EBS CSI Driver implements [Container Storage Interface](https://github.com/container-storage-interface/spec/tree/master) which is the standard of storage interface for container. It provides the same in-tree AWS EBS plugin features including volume creation, volume attachment, volume mounting and volume scheduling. It is also configurable on what is the EBS volume type to create, what is the file system file should be formatted, which KMS key to use to create encrypted volume, etc.

## Motivation
Similar to CNI plugins, AWS EBS CSI driver will be a stand alone plugin that lives out-of-tree of kuberenetes. Being out-of-tree, it will be benefit from being modularized, maintained and optimized without affecting kubernetes core code base. Aside from those benefits, it could also be consumed by other container orchestrators such as ECS.

### Goals
AWS EBS CSI driver will provide similar user experience as in-tree EBS plugin:
* An application developer will not notice any difference in the operation of EBS CSI driver versus the in-tree volume plugin. His/Her workflow will stay the same as before.
* An infrastructure operator needs to deploy/upgrade the driver and create/update storageclass to let the driver to manage underlying storage backend. The storageclass need not be updated if the name of the csi-driver referenced does not change.

Since EBS CSI Driver is out-of-tree implementation that comes outside of kuberenetes distrubtion, documentations will be provided on how to install, use and upgrade the driver.

List of driver features include volume creation/deletion, volume attach/detach, volume mount/unmount, volume scheduling, create volume configurations, volume snapshotting, mount options, raw block volume, etc.

### Non-Goals
* Supporting non AWS block storage
* Supporting other AWS storage serivces such as Dynamodb, S3, etc.

## Proposal

### User Stories

#### Static Provisioning
Operator creates a pre-created EBS volume on AWS and a CSI PV that refers the EBS volume on cluster. Developer creates PVC and a Pod that uses the PVC. Then developer deploys the Pod during which time the PV will be attached to container inside Pod after PVC bonds to PV successfully.

#### Dynamic Provisioning
Operator creates a storage class that defines EBS CSI driver as provisioner. Developer creates PVC and a Pod that uses the PVC. A new CSI PV will be created dynamically and be bound to the defined PVC. Finally, the PV will be attached to container inside Pod.

#### Volume Scheduling
Operation creates StorageClass with  volumeBindingMode = WaitForFirstConsumer. When developer deploys a Pod that has PVC that is trying to claim for a PV, a new PV will be created, attached, formatted and mounted inside Pod&#39;s container by the EBS CSI driver. Topology information provided by EBS CSI driver will be used during Pod scheduling to guarantee that both Pod and volume are collocated in the same availability zone.

#### Mount Options
Operator creates a storage class that defines mount option of the persistence volume. When a PV is dynamically provisioned, the volume will be mounted inside container using the provided mount option.
Operator creates a PV which is backed by a EBS volume manually. The PV spec defines the mount option (eg. ro) of the volume. When the PV is consumed by the application and the Pod is running, the volume will be mounted inside container with the given mount option.

#### Raw Block Volume
Operator creates PV or PVC with `volumeMode: Block`. When application consumes the volume, it is mounted inside container as raw device (eg. /dev/sdba).

#### Offline Volume Resizing
Operator enables the allowVolumeExpansion feature in storageclass. When there is no Pod consuming the volume and user resizes the volume by editing the requested storage size in PVC, the volume got resized by the driver with the given new size.

### Risks and Mitigations
* *Information disclosure* - AWS EBS CSI driver requires permission to perform AWS operations on behalf of the user. The CSI driver will not log any of the user credentials. We will also provide the user with policies that limit the access of the driver to required AWS services.
* *Escalation of Privileges* - Since EBS CSI driver is formatting and mounting volumes, it requires root privilege to permform the operations. So that driver will have higher privilege than other containers in the cluster. The driver will not execute random commands provided by untrusted user. All of its interfaces are only provided for kuberenetes system components to interact with. The driver will also validate requests to make sure it aligns with its assumption.

## Graduation Criteria
* Static provisioning is implemented.
* Dynamic provisioning is implemented.
* Volume scheduling is implemented.
* Mount options is implmented.
* Raw block volume is implemented .
* Offline volume resizing is implemented.
* Integration test is implemented and integrated with Prow and Testgrid.
* E2E tests are implemented and integrated with Prow and Testgrid.

## Upgrade/Downgrade Process
This assumes user is already using some version of the driver.

### Upgrade
This assumes user is already using Kubernetes 1.13 cluster. Otherwise, the existing cluster needs to be upgraded to 1.13+ in order to install the driver. Formal cluster upgrade process should be followed for upgrading cluster.

Driver upgrade should be performed one version at a time. This means, if the current driver version is 0.1, it can be upgraded to version 0.2 by following the upgrade process. And if the driver version that is required to upgrade to is 0.3, it should be upgraded to 0.2 first.

To upgrade the driver, perform following steps:
1. Delete the old driver controller service and node service along with other resources including cluster roles, cluster role bindings and service accounts.
1. Deploy the new driver controller service and node service along with other resources including cluster roles, cluster role bindings and service accounts.

### Downgrade
Similar to driver upgrade, driver downgrade should be performed one version at a time.

To downgrade the driver, perform following steps:
1. Delete the old driver controller service and node service along with other resources including cluster roles, cluster role bindings and service accounts.
1. Deploy the new driver controller service and node service along with other resources inclluding cluster roles, cluster role bindings and service accounts.

## Implementation History
* 2018-11-26 Initial proposal to SIG
* 2018-11-26 Initial KEP draft
* 2018-12-03 Alpha release with kuberentes 1.13
* 2018-03-25 Beta release with kubernetes 1.14

