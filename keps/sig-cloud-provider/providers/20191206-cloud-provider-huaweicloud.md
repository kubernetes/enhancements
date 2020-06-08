---
title: Cloud Provider For HUAWEI CLOUD
authors:
  - "@RainbowMango"
owning-sig: sig-cloud-provider
reviewers:
  - "@andrewsykim"
  - "@cheftako"
approvers:
  - "@andrewsykim"
  - "@cheftako"
editor: TBD
creation-date: 2019-12-06
last-updated: 2020-02-26
status: implementable
see-also:
replaces:
superseded-by:
---

# Cloud Provider For HUAWEI CLOUD

This is a KEP for adding `Cloud Provider For HUAWEI CLOUD` into the Kubernetes ecosystem.

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

[HUAWEI CLOUD Controller Manager](https://github.com/huawei-cloudnative/cloud-provider-huaweicloud) is an external cloud
controller manager for running kubernetes in a HUAWEI CLOUD cluster. It's original open sourced project is https://github.com/huawei-cloudnative/cloud-provider-huaweicloud.


## Motivation

### Goals

`Cloud Provider For HUAWEI CLOUD` provides an external cloud controller manager for users.

In this project, what we dedicated focus is to provide a reliable and optimized implementation of `Cloud Controller Manager`
which satisfies [cloudprovider.Interface](https://github.com/kubernetes/kubernetes/blob/919871e86aebf9e0a640a730d01957075d3a29be/staging/src/k8s.io/cloud-provider/cloud.go#L43).

### Non-Goals

The networking and storage support for Kubernetes are out of the scope that will be provided by other projects.

## Prerequisites

- Kubernetes version must be v1.17+.
- Go version must be v1.13+.

### Repository Requirements

[HUAWEI CLOUD Controller Manager](https://github.com/huawei-cloudnative/cloud-provider-huaweicloud) is an implementation of
[Kubernetes Cloud Controller Manager](https://kubernetes.io/docs/tasks/administer-cluster/running-cloud-controller/).

### User Experience Reports

`Huawei Technologies` is the founding members of CNCF, and `HUAWEI CLOUD` provides customers with stable, reliable,
secure, and sustainably growing cloud services. It helps large enterprises address challenges in cloud transformation
and enables them to take better advantages of potential business opportunities.
It also helps small- and medium-sized enterprises expand their business growth and rise to challenges.

The [Cloud Container Engine (CCE)](https://www.huaweicloud.com/en-us/product/cce.html) is a high-performance,
high-reliability service through which enterprises can manage containerized native Kubernetes applications.
CCE has been using this project for several releases:
- v1.15 [Conformance Test Report](https://github.com/cncf/k8s-conformance/tree/843ee84d40962baa07cab9e59a19abe7f778b6b0/v1.15/huawei-cce)
- v1.13 [Conformance Test Report](https://github.com/cncf/k8s-conformance/tree/843ee84d40962baa07cab9e59a19abe7f778b6b0/v1.13/huawei-cce)
- v1.11 [Conformance Test Report](https://github.com/cncf/k8s-conformance/tree/843ee84d40962baa07cab9e59a19abe7f778b6b0/v1.11/huawei-cce)
- v1.9 [Conformance Test Report](https://github.com/cncf/k8s-conformance/tree/843ee84d40962baa07cab9e59a19abe7f778b6b0/v1.9/huawei-cce)

As well as CCE, another certified Kubernetes product which used this project is `Huawei Fusionstage`, and you can refer to
following conformance test reports:
- v1.15 [Conformance Test Report](https://github.com/cncf/k8s-conformance/tree/843ee84d40962baa07cab9e59a19abe7f778b6b0/v1.15/huawei-fusionstage)
- v1.13 [Conformance Test Report](https://github.com/cncf/k8s-conformance/tree/843ee84d40962baa07cab9e59a19abe7f778b6b0/v1.13/huawei-fusionstage)
- v1.11 [Conformance Test Report](https://github.com/cncf/k8s-conformance/tree/843ee84d40962baa07cab9e59a19abe7f778b6b0/v1.11/huawei-fusionstage)
- v1.9  [Conformance Test Report](https://github.com/cncf/k8s-conformance/tree/843ee84d40962baa07cab9e59a19abe7f778b6b0/v1.9/huawei-fusionstage)

Other usage of this project can be seen from [GitHub issues](https://github.com/huawei-cloudnative/cloud-provider-huaweicloud/issues).

### Testgrid Integration

Huawei cloud provider is reporting conformance test results to TestGrid as per the [Reporting Conformance Test Results to Testgrid KEP](https://github.com/kubernetes/enhancements/blob/6427a0becff459815e0e41f72f65ab5f3b8e9c6d/keps/sig-cloud-provider/0018-testgrid-conformance-e2e.md).
You can get the result from [TestGrid Huawei Cloud Dashboard](https://testgrid.k8s.io/conformance-cloud-provider-huaweicloud).

### CNCF Certified Kubernetes

Huawei cloud provider is accepted as part of the [Certified Kubernetes Conformance Program](https://github.com/cncf/k8s-conformance).

The conformance test report can be seen from above section `Cloud Container Engine (CCE)` and `Huawei Fusionstage`.

### Documentation

Huawei cloud provider provides [multiple documentation](https://github.com/huawei-cloudnative/cloud-provider-huaweicloud/tree/70a268bb38183a09b14e3711699d7170a21d317e/docs) for users as per the [cloud provider documentation KEP](https://github.com/kubernetes/enhancements/blob/6427a0becff459815e0e41f72f65ab5f3b8e9c6d/keps/sig-cloud-provider/20180731-cloud-provider-docs.md).

### Technical Leads are members of the Kubernetes Organization

Technical leads take the responsibility of maintain this project:
- @kevin-wangzefeng Lead of Kubernetes and Cloud Native Open Source Team at Huawei. Kubernetes maintainer.
- @RainbowMango Kubernetes member.

## Proposal

We need a repository under the Kubernetes organization to host our cloud provider specific implementation.
We'd like HUAWEI CLOUD provider to become a subproject of Kubernetes community.

### Subproject Leads

- @kevin-wangzefeng Lead of Kubernetes and Cloud Native Open Source Team at Huawei. Kubernetes maintainer.
- @RainbowMango Kubernetes member.

### Repositories

We would like a repository named `kubernetes/cloud-provider-huaweicloud` to host HUAWEI CLOUD specific code.

The owners of the subproject can be the subject leads listed on above.

### Meetings

Recommended Meeting Time: Wednesdays at 20:00 PT (Pacific Time) (biweekly). [Convert to your timezone](http://www.thetimezoneconverter.com/?t=20:00&tz=PT%20%28Pacific%20Time%29).
- Meeting notes and Agenda.(TBD)
- Meeting recordings.(TBD)

### Others
