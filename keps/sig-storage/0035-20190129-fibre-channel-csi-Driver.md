---
kep-number: 35
title: Fibre Channel CSI Driver
authors:
  - "@mathu97"
owning-sig: sig-storage
participating-sigs:
  - sig-storage
reviewers:
  - "@rootfs"
  - "@bchilds"
  - "@wongma7"
  - "@screeley44"
  - "@saad-ali"
approvers:
  - "@bchilds"
  - "@saad-ali"
creation-date: 2018-01-29
status: provisional
---

# Fibre Channel CSI Driver

## Table of Contents

* [Table of Contents](#table-of-contents)
* [Summary](#summary)
* [Motivation](#motivation)
    * [Goals](#goals)
    * [User Stories](#user-stories)
* [Graduation Criteria](#graduation-criteria)
* [Implementation History](#implementation-history)

[Tools for generating]: https://github.com/ekalinin/github-markdown-toc

## Summary

Fibre channel CSI driver implements the [Container Storage Interface](https://github.com/container-storage-interface/spec/tree/master). The driver was developed in a personal [repo](https://github.com/mathu97/FC-CSI-Driver), it needs to be 
updated to work with CSI 1.0 and moved to [kubernetes-csi/csi-driver-fibre-channel](https://github.com/kubernetes-csi/csi-driver-fibre-channel).
There is currently a [PR](https://github.com/kubernetes-csi/csi-driver-fibre-channel/pull/1) that addresses this.

## Motivation
The fibre channel CSI driver is an out-of-tree volume plugin. Being out-of-tree gives flexibility for it to be 
maintained independent of Kubernetes. Having a seperate repo under the kubernetes-csi org for the 
driver allows easy access for users, while also enabling contributions and feedback from the community.

### Goals

* The driver should enable the usage of fibre channel based volumes for Pods in kubernetes
* Have unit tests that ensure that the driver is working
* Provide example yaml files that show how to deploy the driver

### User Stories
* As an application developer I should have access to fibre channel storage backends
* As a Kuberenetes Administrator I need to be able to deploy the fibre channel CSI driver and create 
persistent volumes for applications to use

## Graduation Criteria

How will we know that this has succeeded?
* Successfull provisioning 
* Unit Testing

## Implementation History
* 2018-10-01 Completion of basic fibre channel CSI driver for CSI 0.3
* 2019-01-29 Initial KEP draft
