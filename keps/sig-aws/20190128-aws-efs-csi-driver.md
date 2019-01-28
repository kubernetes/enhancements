---
title: aws-efs-csi-driver
authors:
  - "@leakingtapan"
owning-sig: sig-aws
reviewers:
  - "@d-nishi"
  - "@jsafrane"
approvers:
  - "@d-nishi"
  - "@jsafrane"
editor: TBD
creation-date: 2019-01-28
last-updated: 2019-01-28
status: provisional
---

# AWS Elastic File System (EFS) CSI Driver

## Table of Contents

* [Table of Contents](#table-of-contents)
* [Summary](#summary)
* [Motivation](#motivation)
    * [Goals](#goals)
    * [Non-Goals](#non-goals)
* [Proposal](#proposal)
    * [User Stories](#user-stories)
        * [Static Provisioning](#static-provisioning)
        * [Encryption in Transit](#encryption-in-transit)
    * [Risks and Mitigations](#risks-and-mitigations)
* [Graduation Criteria](#graduation-criteria)
* [Implementation History](#implementation-history)

## Summary
AWS EFS CSI Driver implements [Container Storage Interface](https://github.com/container-storage-interface/spec/tree/master) which is the standard of storage interface for container.  

## Motivation
Users who run Kubernetes workloads on AWS are looking for well integartion with AWS services of different storage backend systems. Being shared filesystem, AWS EFS provides a simple, scalable, elastic filesystem fo Linux-based workloads. Aside from being a managed cloud service, it provides critical security features such as encryption in transit which guarantees all traffics from the client and the server are encrypted. And it is one of the security compliance requirements for various applications.

Similar to EBS CSI driver, AWS EFS CSI driver will be an out-of-tree volume plugin that lives outside of main core Kubernetes codebase. Being out-of-tree, it benefits from being modularized, flexibility to be maintained and optimized independent of Kubernetes release cadence. Aside from those benefits, it could also be consumed by other container orchestrators such as ECS.

### Goals
AWS EFS CSI driver will provide native Kubernetes experience to operate and consume EFS filesystem in container workloads. 

* As an application developer, he will not even notice any differences between different storage backends. 
* As an infrastructure operator, he will need to deploy the EFS CSI driver and create persistence volume (PV) objects for the application to consume.

List of driver features include static provisioning, encryption in transit, etc.

### Non-Goals
* Supporting non AWS NFS backed storage
* Supporting other AWS storage serivces such as Dynamodb, S3, etc.

## Proposal

### User Stories

#### Static Provisioning
Operator creates an EFS filesystem on AWS and a CSI PV that refers the EFS filesystem ID. Developer creates PVC and a Pod that uses the PVC. Then developer deploys the Pod. During container creation, after PVC bonds to the PV successfully, the underlying EFS volume will be mounted to container inside Pod.

#### Encryption in Transit
Operator creates an EFS filesystem on AWS and a CSI PV that refers the EFS filesystem ID. Operator specifies the PV mount option with `tls` flag. After the volume is consumed by some application and mounted inside container, all the read/write operation to the EFS filesystem will be encrypted.

### Sample Persistence Volume Spec
The following is a sample PV spec that uses EFS as storage backend and the volume will be mounted with encryption in transit enabled (with the tls mount option):
```
apiVersion: v1
kind: PersistentVolume
metadata:
  name: efs-pv
spec:
  capacity:
    storage: 5Gi
  volumeMode: Filesystem
  accessModes:
    - ReadWriteOnce
  persistentVolumeReclaimPolicy: Recycle
  storageClassName: efs-sc
  mountOptions:
    - tls
  csi:
    driver: efs.csi.aws.com
    volumeHandle: fs-xxxxxxxx
```

## Graduation Criteria
* Static provisioning is implemented
* Mount volume with encryption in transit is supported.
* Unit testing

## Implementation History
* 2019-01-11 Initial proposal to SIG
* 2019-01-28 Initial KEP draft

