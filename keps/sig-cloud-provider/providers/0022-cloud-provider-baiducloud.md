---
title: Cloud Provider for BaiduCloud
authors:
  - "@tizhou86"
owning-sig: sig-cloud-provider
reviewers:
  - "@andrewsykim"
  - "@cheftako"
approvers:
  - "@andrewsykim"
  - "@cheftako"
  - "@hogepodge"
  - "@jagosan"
editor: TBD
creation-date: 2020-02-25
last-updated: 2020-02-25
status: implementable
---

# Cloud Provider BaiduCloud

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

Baidu is a gold member of CNCF and have a large team working on Kubernetes and related projects like complex scheduling, heterogeneous computing, auto-scaling etc. Baidu build cloud platform to support Baidu emerging business including autonomous driving, deep learning, blockchain by leveraging Kubernetes. Baidu also provide public container services named cloud container engine(CCE) and other services like micro-services management platform(CNAP) and edge computing platform(BEC). In 2019, Baidu ranked 10th globally in Kubernetes contribution.

## Motivation

### Goals

- Integrating and extending Kubernetes with Baidu Cloud Container Engine(CCE), Baidu Private Cloud(BPC), Baidu Micro-Services Platform(CNAP) Baidu Edge Computing(BEC).

- Developing and maintaining cloud resource (node, routing, load balancing, etc.) interface and configurations for full lifecycle management with Kubernetes.

- Developing and maintaining cloud vendor neutral Kubernetes related testing frameworks and tools.

### Non-Goals

- The subproject will not provide any standard technical support for specific cloud vendor.

- The subproject will not work on features or bugs which are not related to Kubernetes integration.

## Prerequisites

### Repository Requirements

[BaiduCloud Controller Manager](https://github.com/baidu/cloud-provider-baiducloud) is a working implementation of the Kubernetes Cloud Controller Manager.

### User Experience Reports

![CCE-ticket-1](http://agroup-bos.su.bcebos.com/c34021571744895b5d9fffd8c22d8409469f47b3)
CCE-ticket-1: User want to get the Kubernetes cluster config file by using account's aksk.

![CCE-ticket-2](http://agroup-bos.su.bcebos.com/756c9463c8487dee9c26d7725e127c5b64975fc4)
CCE-ticket-2: User want to modify the image repository's username.

![CCE-ticket-3](http://agroup-bos.su.bcebos.com/7a4506fcb1fbeeb15c86060cfbb6e69d090c8984)
CCE-ticket-3: User want to have multi-tenant ability in a shared large CCE cluster.

### Testgrid Integration

Baidu applied for the testgrid in 2018, The original report for testgrid is [here](https://k8s-testgrid.appspot.com/conformance-cloud-provider-baiducloud). We are currently working on renewing the testgrid.

### CNCF Certified Kubernetes

For 1.16, The certified Kubernetes link is [here](https://github.com/cncf/k8s-conformance/tree/master/v1.16/baiducloud).
For 1.13, The certified Kubernetes link is [here](https://github.com/cncf/k8s-conformance/tree/master/v1.13/baiducloud).
For 1.11, The certified Kubernetes link is [here](https://github.com/cncf/k8s-conformance/tree/master/v1.11/baiducloud).

### Documentation

BaiduCloud provides documentations for users to build and utilize cloud controller manager. Please refer to this [link](https://github.com/baidu/cloud-provider-baiducloud) for more details.

### Technical Leads are members of the Kubernetes Organization

- @tizhou86 Ti Zhou, Kubernetes Member

## Proposal

### Subproject Leads

The subproject will have 3 leaders at any given time. 

- @tizhou86 Ti Zhou, Kubernetes Member
- @hello2mao Hongbin Mao, Kubernetes Member
- @ZP-AlwaysWin Peng Zhang, Kubernetes Member

### Repositories

The repository we propose at this moment is: kubernetes/cloud-provider-baiducloud or kubernetes-sigs/cloud-provider-baiducloud. I'll be the initial point of contact.

### Meetings

We plan to have bi-weekly online meeting on every next Wednesday 6pm PST. Meeting will have notes and agenda and be recorded.

### Others

NA

