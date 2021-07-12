# Cloud Provider For INSPUR CLOUD

This is a KEP for adding `Cloud Provider For INSPUR CLOUD` into the Kubernetes ecosystem.

## Table of Contents

<!-- toc -->
- [Summary](#summary)
- [Motivation](#motivation)
  - [Goals](#goals)
  - [Non-Goals](#non-goals)
- [Prerequisites](#prerequisites)
  - [Repository Requirements](#repository-requirements)
  - [User Experience Reports](#user-experience-reports)
  - [Testgrid Integration](#testgrid-integration)
  - [CNCF Certified Kubernetes](#cncf-certified-kubernetes)
  - [Documentation](#documentation)
  - [Technical Leads are members of the Kubernetes Organization](#technical-leads-are-members-of-the-kubernetes-organization)
- [Proposal](#proposal)
  - [Subproject Leads](#subproject-leads)
  - [Repositories](#repositories)
  - [Meetings](#meetings)
  - [Others](#others)
<!-- /toc -->

## Summary

[INSPUR CLOUD Controller Manager](https://github.com/OpenInspur/cloud-provider-inspur) is an external cloud 
controller manager for running kubernetes in a INSPUR CLOUD cluster. It's original open sourced project is https://github.com/OpenInspur/cloud-provider-inspur .


## Motivation

### Goals

`Cloud Provider For INSPUR CLOUD` provides an external cloud controller manager for users.

In this project, what we dedicated focus is to provide a reliable and optimized implementation of `Cloud Controller Manager` 
which satisfies [cloudprovider.Interface](https://github.com/kubernetes/kubernetes/blob/master/staging/src/k8s.io/cloud-provider/cloud.go#L43).

### Non-Goals

The networking and storage support for Kubernetes are out of the scope that will be provided by other projects.

## Prerequisites

- Kubernetes version must be v1.14+.
- Go version must be v1.10+.

### Repository Requirements

[INSPUR CLOUD Controller Manager](https://github.com/OpenInspur/cloud-provider-inspur) is an implementation of
[Kubernetes Cloud Controller Manager](https://kubernetes.io/docs/tasks/administer-cluster/running-cloud-controller/). 

### User Experience Reports

`INSPUR` is the founding members of CNCF, and `INSPUR CLOUD` provides customers with stable, reliable, 
secure, and sustainably growing cloud servIOPs. It helps large enterprises address challenges in cloud transformation 
and enables them to take better advantages of potential business opportunities. 
It also helps small- and medium-sized enterprises expand their business growth and rise to challenges. 

The [Inspur Open Platform (IOP)](https://cloud.inspur.com/product/UK8S/) is an application-oriented container management platform based on arm64 architecture. It provides complete Kubernetes clustering capabilities, helping users to quickly deploy flexible and reliable container clusters, easily create and manage container workloads, and provide automatic scaling and resource monitoring. Efficient operation and maintenance capabilities such as log collection and retrieval. 
IOP has been using this project for several releases:
- v1.18 [Conformance Test Report for amd64](https://github.com/cncf/k8s-conformance/blob/master/v1.18/inspur-iop-amd64)
- v1.18 [Conformance Test Report for arm64](https://github.com/cncf/k8s-conformance/blob/master/v1.18/inspur-iop-arm64)
- v1.17 [Conformance Test Report for amd64](https://github.com/cncf/k8s-conformance/blob/master/v1.17/inspur-iop-amd64)
- v1.17 [Conformance Test Report for arm64](https://github.com/cncf/k8s-conformance/blob/master/v1.17/inspur-iop-arm64)
- v1.16 [Conformance Test Report for amd64](https://github.com/cncf/k8s-conformance/blob/master/v1.16/inspur-iop-amd64)
- v1.16 [Conformance Test Report for arm64](https://github.com/cncf/k8s-conformance/blob/master/v1.16/inspur-iop-arm64)
- v1.15 [Conformance Test Report for amd64](https://github.com/cncf/k8s-conformance/blob/master/v1.15/inspur-iop-amd64)
- v1.15 [Conformance Test Report for arm64](https://github.com/cncf/k8s-conformance/blob/master/v1.15/inspur-iop-arm64)
- v1.14 [Conformance Test Report for amd64](https://github.com/cncf/k8s-conformance/blob/master/v1.14/inspur-iop-amd64)
- v1.14 [Conformance Test Report for arm64](https://github.com/cncf/k8s-conformance/blob/master/v1.14/inspur-iop-arm64)

Other usage of this project can be seen from [GitHub issues](https://github.com/OpenInspur/cloud-provider-inspur/issues).

### Testgrid Integration

INSPUR cloud provider is reporting conformance test results to TestGrid as per the [Reporting Conformance Test Results to Testgrid KEP](https://github.com/kubernetes/enhancements/blob/master/keps/sig-cloud-provider/0018-testgrid-conformance-e2e.md).
You can get the result from [TestGrid INSPUR Cloud Dashboard](https://testgrid.k8s.io/conformance-cloud-provider-inspur). 

### CNCF Certified Kubernetes

INSPUR cloud provider is accepted as part of the [Certified Kubernetes Conformance Program](https://github.com/cncf/k8s-conformance).

The conformance test report can be seen from above section `Inspur Open Platform (IOP)` and `INSPUR`. 

### Documentation

INSPUR cloud provider provides [multiple documentation](https://github.com/OpenInspur/cloud-provider-inspur/blob/master/docs) for users as per the [cloud provider documentation KEP](https://github.com/kubernetes/enhancements/blob/master/keps/sig-cloud-provider/20180731-cloud-provider-docs.md).

### Technical Leads are members of the Kubernetes Organization

Technical leads take the responsibility of maintain this project:
- @timyinshi Lead of Kubernetes and Cloud Native Open Source Team at INSPUR. Kubernetes maintainer.
- @ydcool Kubernetes member.

## Proposal

We need a repository under the Kubernetes organization to host our cloud provider specific implementation.
We'd like INSPUR CLOUD provider to become a subproject of Kubernetes community. 

### Subproject Leads

- @timyinshi Lead of Kubernetes and Cloud Native Open Source Team at INSPUR. Kubernetes maintainer.
- @ydcool Kubernetes member.

### Repositories

We would like a repository named `kubernetes/cloud-provider-inspur` to host INSPUR CLOUD specific code.

The owners of the subproject can be the subject leads listed on above.

### Meetings

Recommended Meeting Time: Wednesdays at 18:00 PT (Pacific Time) (biweekly). [Convert to your timezone](http://www.thetimezoneconverter.com/?t=18:00&tz=PT%20%28Pacific%20Time%29).
- Meeting notes and Agenda.(TBD)
- Meeting recordings.(TBD)

### Others