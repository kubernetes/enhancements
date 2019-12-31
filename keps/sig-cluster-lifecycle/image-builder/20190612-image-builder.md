---
title: Kubernetes Image Builder
authors:
  - "@timothysc"
  - "@moshloop"
owning-sig: sig-cluster-lifecycle
reviewers:
  - "@justinsb"
  - "@luxas" 
  - "@astrieanna"
approvers:
  - "@justinsb"
  - "@timothysc"
  - "@luxas"
editor: "@timothysc"
creation-date: 2019-06-11
last-updated: 2019-07-05
status: provisional
---

# Kubernetes Image Builder

## Table of Contents

<!-- toc -->
- [Summary](#summary)
- [Motivation](#motivation)
  - [Goals](#goals)
  - [Non-Goals](#non-goals)
- [Proposal](#proposal)
  - [Phases](#phases)
    - [Phase 0 (Base Image)](#phase-0-base-image)
    - [Phase 1 (Software Installation / Customization)](#phase-1-software-installation--customization)
    - [Phase 2 (Artifact Generation)](#phase-2-artifact-generation)
  - [Risks and Mitigations](#risks-and-mitigations)
- [Graduation Criteria](#graduation-criteria)
- [Implementation History](#implementation-history)
- [Infrastructure Needed](#infrastructure-needed)
- [Alternatives](#alternatives)
<!-- /toc -->

## Summary
It is common for modern cloud based software deployments to follow immutable patterns. One of the foundational pieces to this idea is the creation of immutable images. There are already several tools that create images in the Kubernetes ecosystem, which include: [Wardroom](https://github.com/heptiolabs/wardroom), [Cluster API AWS](https://github.com/kubernetes-sigs/cluster-api-provider-aws/blob/master/Makefile), [Cluster API vSphere](https://github.com/kubernetes-sigs/cluster-api-provider-vsphere/blob/master/Makefile), [amazon-eks-ami](https://github.com/awslabs/amazon-eks-ami), [talos](https://docs.talos-systems.com/), [LinuxKit](https://github.com/linuxkit/linuxkit),[kube-deploy](https://github.com/kubernetes/kube-deploy/tree/master/imagebuilder), etc. The purpose of this proposal is to distill down the common requirements and provide an image building utility that can be leveraged by the Kubernetes ecosystem.  

The purpose of this document is to request the creation of a sub-project of sig-cluster-lifecycle to address this space. 

## Motivation
There exists a need to be able to create repeatable IaaS images across providers for the explicit purpose of being able to deploy a Kubernetes cluster.

### Goals 
* To build images for Kubernetes-conformant clusters in a consistent way across infrastructures, providers, and business needs.
   * Install all software, containers, and configuration needed to pass conformance tests.
   * Support end users requirements to customize images for their business needs.
* To provide assurances in the binaries and configuration in images for purposes of security auditing and operational stability.
   * Allow introspection of artifacts, software versions, and configurations in a given image.
   * Support repeatable build processes where the same inputs of requested install versions result in the same installed binaries.
* To ensure that the creation of images is performed via well defined phases.  Where users could choose specific phases that they needed.
    * Support incremental usage.

### Non-Goals 
* To publish images to cloud provider marketplaces, or to provide software workflow to automatically upload the built images on the cloud provider infrastructure.
    * For example, it is not the responsibility of *this* utility to publish images to Amazon Marketplace. Each Cluster API Provider may implement its own image publisher. Users should be able to use the provider's publisher with the image output by the image builder.
* To provide upgrade or downgrade semantics.
* To provide guarantees that the software installed provides a fully functional system.
* To prescribe the hardware architecture of the build system.
* To create images from scratch.  
    * The purpose of the tool is to take an existing image and make it Kubernetes ready.

## Proposal 
The Image Builder will start from one image in a supported format and create a new image in the same format specifically for the purpose of creating Kubernetes clusters.  In surveying the landscape of tooling it becomes readily apparent that there are a plethora of tools that provide an opinionated end-to-end user story around image creation, but we’ve observed it can be decomposed into a series of steps, or phases.  By decomposing the problem we can provide a rallying point for different tools to integrate, and provide the Kubernetes ecosystem with a common utility and user experience across those tools.

As a precondition the Image Builder will require a bootable disk image as an input, with native support for the cloud images published by the supported distributions.  However any external process or tool can be used to create the initial disk image from other sources including [ISO](https://packer.io)’s, file trees and [docker](https://github.com/iximiuz/docker-to-linux) images. Existing disk images can also be customized using tools like [virt-customize](http://libguestfs.org/virt-customize.1.html) before being fed into the Image Builder.

**NOTE:** It should be noted that this document is intentionally high level and purposefully omits design choices which should be made at a later date once the subproject is further along in its lifecycle. 

### Phases 
#### Phase 0 (Base Image)

Lay down the initial base image.  Often times this can be some form of certified base image from a vendor or IT team.   **NOTE:** It is not a goal of this project to take on creation of those initial images.

**Input:**

`--disk-image` - A local or remote path to a libvirt/qemu supported image that a user or provider creates. (raw/qcow/vmdk etc.)

Images are verified and cached by looking for a SHA256SUMS, sha256sum.txt file in the same directory as the image

* [Ubuntu](https://cloud-images.ubuntu.com/bionic/current/)
* [Fedora](https://alt.fedoraproject.org/cloud/)
* [Debian](https://cloud.debian.org/images/openstack/) and a [comparison](https://wiki.debian.org/Cloud/SystemsComparison) of the types
* [CentOS](https://cloud.centos.org/centos/7/images/)
* [Amazon Linux](https://cdn.amazonlinux.com/os-images/current/kvm/)

**Output:** Running shell inside the root filesystem

Phase 0 will kickoff Phase 1, for example by chrooting into the disk or using nested virtualization to boot the image and then SSH into it.

#### Phase 1 (Software Installation / Customization)

The purpose of this phase would be the installation of the Kubernetes stack, default account setup, updating packages, config, etc. 

**Input:** / with root/sudo access and a known package manager

**Output:** A modified disk image

#### Phase 2 (Artifact Generation)

Produce output artifacts in their final form, and ideally this should include a BOM.

### Risks and Mitigations
Given that there are already a plethora of existing solutions in the ecosystem the risk to the community is small, and this would allow contributors to help refine the best practices as they see them.  In the case where the subproject does not see traction we will orphan the subproject to the kubernetes-retired org. 

## Graduation Criteria 
alpha: Adoption across Cluster API providers.

(post-alpha criteria will be added post-alpha)

## Implementation History
KEP created - Jun 12 2019 
Vote approved - Jul 02 2019 


## Infrastructure Needed
None at this time, but it's possible this tool could become a critical piece of the test-automation for kubernetes, or Cluster API. 

We are requesting to be a subproject under sig-cluster-lifecycle.

## Alternatives 
Prior to this KEP a Cluster API workstream had written a [document](https://docs.google.com/document/d/1N65N1vCVa5QmU4BJXeSOImgRE8aq7daWhHt7XE9WCeI/edit?ts=5cde5f47#) outlining several options.
