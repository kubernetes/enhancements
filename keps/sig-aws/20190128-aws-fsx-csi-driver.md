---
title: aws-fsx-csi-driver
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

# AWS FSx For Lustre CSI Driver

## Table of Contents

* [Table of Contents](#table-of-contents)
* [Summary](#summary)
* [Motivation](#motivation)
    * [Goals](#goals)
    * [Non-Goals](#non-goals)
* [Proposal](#proposal)
    * [User Stories](#user-stories)
        * [Static Provisioning](#static-provisioning)
        * [Dynamice Provisioning](#dynamic-provisioning)
    * [Risks and Mitigations](#risks-and-mitigations)
* [Graduation Criteria](#graduation-criteria)
* [Implementation History](#implementation-history)

## Summary
AWS FSx for Lustre CSI Driver implements [Container Storage Interface](https://github.com/container-storage-interface/spec/tree/master) which is the standard of storage interface for container. Using the driver, developer can access FSx for Lustre filesystem using native Kubernetes primitives such as persistence volume (PV) or persistence volume (PVC).

## Motivation
Users who run Kubernetes workloads on AWS are looking for well integartion with AWS services of different storage backend systems. Being popular distributed parallel filesystem for high performace computing (HPC), AWS FSx for Lustre provides fully managed file system that is optimized for compute-intensive workloads. Without the driver, cluster operator has to manage the FSx for Lustre filesystem lifecycle outside Kubernetes manually or build their own tools to automate the process; application developer has to workaround in order to mount FSx for Lustre volume inside container. In some use cases, FSx for Lustre filesystem is expected to be short lived where it is used as a data caching layer for machines learning traninig job. These situation makes managing FSx for Lustre filesystem manually from challenging to even impossible.

Similar to EBS CSI driver, AWS FSx for Lustre CSI driver will be an out-of-tree volume plugin that lives outside of main core Kubernetes codebase. Being out-of-tree, it benefits from being modularized, flexibility to be maintained and optimized independent of Kubernetes release cadence. Aside from those benefits, it could also be consumed by other container orchestrators such as ECS.

### Goals
AWS FSx for Lustre CSI driver will provide native Kubernetes experience where operator operates the lifecycle of FSx for Lustre volume and where application consumes FSx for Lustre volume inside container workloads.

* As an application developer, he will not even notice any differences between different storage backends.
* As an infrastructure operator, he will need to deploy FSx CSI CSI driver, create storageclass object, create PV or PVC objects for the application to consume.

List of driver features include static provisioning, dynamic provisioning etc.

### Non-Goals
* Supporting non AWS Lustre filesystem
* Supporting other AWS storage serivces such as Dynamodb, S3, etc.

## Proposal

### User Stories

#### Static Provisioning
Operator creates a FSx for Lustre filesystem on AWS and a CSI PV that refers the FSx for Lustre filesystem ID and DNS name as volume attribute. Developer creates PVC and a Pod that uses the PVC. Then developer deploys the Pod. During container creation, after PVC bonds to the PV successfully, the underlying FSx for Lustre volume will be mounted to container inside Pod.

#### Dynamic Provisioning
Operator creates a storageclass that uses FSx for Lustre CSI driver as provisioner. Developer creates PVC and a Pod that uses the PVC. A new PV will be created dynamically along with a FSx for Lustre filesystem. Finally, the create filesystem volume will be attached to container inside Pod.

## Graduation Criteria
* Static provisioning is implemented
* Dynamic provisioning is implemented
* Unit testing

## Implementation History
* 2019-01-11 Initial proposal to SIG
* 2019-01-28 Initial KEP draft

