---
kep-number: 23
title: Cloud Provider for Microsoft Azure
authors:
  - "@dstrebel"
owning-sig: sig-cloud-provider
participating-sigs:
  - sig-azure
reviewers:
  - "@dstrebel"
  - "@justaugustus"
  - "@khenidak"
  - "@feiskyer"
approvers:
  - "@feiskyer"
  - "@khenidak"
  - "@hogepodge"
  - "@jagosan"
editor: TBD
creation-date: 2019-01-29
last-updated: 2019-01-29
status: provisional

---

# Cloud Provider for Microsoft Azure

This is a KEP for adding ```Cloud Provider for Azure``` into the Kubernetes ecosystem.

## Table of Contents

* [Table of Contents](#table-of-contents)
* [Summary](#summary)
* [Motivation](#motivation)
    * [Goals](#goals)
    * [Non-Goals](#non-goals)
* [Requirements](#requirements)
* [Proposal](#proposal)

## Summary

Azure provides the Cloud Provider interface implementation as an out-of-tree cloud-controller-manager. It allows Kubernetes clusters to leverage the infrastructure services of Microsoft Azure.
It is original open sourced project is 

## Motivation

### Goals

The Cloud Provider for Microsoft Azure implements interoperability between Kubernetes clusters and Microsoft Azure. This project will be dedicated to:
- Provide reliable, secure and optimized integration with Microsoft Azure for Kubernetes
- Help on the improvement for decoupling cloud provider specifics from Kubernetes core implementation.
- Providing an open community and environment to for developing Kubernetes on Azure 



### Non-Goals

//TODO Need More Info

Kubernetes storage support for Azure will be provided by the Azure CSI interface.

E.g.

* [Azure Kubernetes Storage Drivers](https://github.com/Azure/kubernetes-volume-drivers/tree/master/csi)

## Prerequisites

//TODO

### Repository Requirements

[Azure Cloud Controller Manager](https://github.com/kubernetes/cloud-provider-azure) is a working implementation of the [Kubernetes Cloud Controller Manager](https://kubernetes.io/docs/tasks/administer-cluster/running-cloud-controller/).

The repo requirements is a copy from [cloudprovider KEP](https://github.com/kubernetes/community/blob/master/keps/sig-cloud-provider/0002-cloud-controller-manager.md#repository-requirements). Open the link for more detail.

### User Experience Reports
As a CNCF Platinum member, Microsoft Azure is dedicated in providing users with a highly secure ,stable and efficient cloud service. //TODO

### Testgrid Integration
 Azure Cloud Controller provider is reporting conformance test results to TestGrid as per the [Reporting Conformance Test Results to Testgrid KEP](https://github.com/kubernetes/enhancements/blob/master/keps/sig-cloud-provider/0018-testgrid-conformance-e2e.md).
 See [report](https://k8s-testgrid.appspot.com/conformance-AZURE #NEED INFO) for more details. //TODO

### CNCF Certified Kubernetes
 Microsoft Azure cloud provider is accepted as part of the [Certified Kubernetes Conformance Program](https://github.com/cncf/k8s-conformance).
 * v1.13 https://github.com/cncf/k8s-conformance/tree/master/v1.13/aks
 * v1.12 https://github.com/cncf/k8s-conformance/tree/master/v1.12/aks
 * v1.11 https://github.com/cncf/k8s-conformance/tree/master/v1.11/aks
 * v1.10 https://github.com/cncf/k8s-conformance/tree/master/v1.10/aks


### Documentation
 
//TODO.
 
### Technical Leads are members of the Kubernetes Organization

The Leads run operations and processes governing this subproject.

* @khenidak
* @feiskyer

### Subproject Leads

* @dstrebel
* @justaugustus

## Proposal

We propose a repository from the Kubernetes organization to host our cloud provider implementation.  The Cloud Provider for Microsoft Azure would be a subproject under Kubernetes community.

### Repositories

Cloud Provider for Azure will need a repository under Kubernetes org named ```kubernetes/cloud-provider-azure``` to host any cloud specific code. The initial owners will be indicated in the initial OWNER files.

Additionally, SIG-cloud-provider take the ownership of the repo but Microsoft Azure should have the fully autonomy to operate under this subproject.

### Meetings

Sig-Azure meetings is expected to have biweekly. SIG Cloud Provider will provide zoom/youtube channels as required. We will have our first meeting after repo has been settled.

Meeting Time: Wednesdays at 09:00 PT (Pacific Time) (biweekly). [Convert to your timezone](http://www.thetimezoneconverter.com/?t=20:00&tz=PT%20%28Pacific%20Time%29).
- Meeting notes and Agenda.
- Meeting recordings.


### Others
