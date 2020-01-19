---
title: graduate-aws-nlb-to-beta
authors:
  - "@M00nF1sh"
owning-sig: sig-cloud-provider
participating-sigs:
reviewers:
  - "@dnishi"
approvers:
  - "@justinsb"
  - "@dnishi"
editor: TBD
creation-date: 2019-05-01
last-updated: 2019-05-01
status: implementable
see-also:
replaces:
superseded-by:
---

# Graduate AWS Network Load Balancer Support to beta

## Table of Contents

<!-- toc -->
- [Summary](#summary)
- [Motivation](#motivation)
  - [Goals](#goals)
- [Proposal](#proposal)
  - [User Stories](#user-stories)
- [Design](#design)
  - [Test Plan](#test-plan)
    - [Needed Tests](#needed-tests)
  - [Graduation Criteria](#graduation-criteria)
- [Proposed roadmap](#proposed-roadmap)
  - [1.15](#115)
  - [1.16](#116)
  - [1.18](#118)
- [Implementation History](#implementation-history)
<!-- /toc -->

## Summary

[AWS Network Load Balancer](https://docs.aws.amazon.com/elasticloadbalancing/latest/network/introduction.html) is an L4 Load Balancer on AWS platform, that can be used to declare a Kubernetes Service with type: LoadBalancer.

## Motivation

[AWS Network Load Balancer](https://aws.amazon.com/blogs/opensource/network-load-balancer-support-in-kubernetes-1-9/) has been supported in Kubernetes as Alpha feature since v1.9. Since then, the code and API has been stabilized. Therefore, we would like to graduate NLB support from Alpha to Beta.

### Goals

* Promote AWS Network Load Balancer support to beta version.

## Proposal

### User Stories
* An application developer or infrastructure engineer who wants to use AWS Network Load balancer can declare their Kubernetes Service with type: Load Balancer for any Kubernetes clusters running in the AWS cloud. 

## Design

### Test Plan

#### Needed Tests

- Add E2E tests to allow usage of NLB to declare Kubernetes Service with type: LoadBalancer.

### Graduation Criteria
- [x] Support Cross-zone Load Balancing
- [x] Support TLS termination
- [] Have documentation for NLB annotations
- [ ] Have E2E test
- [x] Have roadmap for future development

## Proposed roadmap
### 1.15
* [ ] Graduate AWS Network Load Balancer support to beta
### 1.16
* [ ] Deprecate usage of AWS Classic Load Balancer as the default implementation for Service with LoadBalancerType on AWS
* [ ] Notify users to migrate to use NLB instead
### 1.18
* [ ] Use AWS Network Load Balancer as the default implementation for Service with LoadBalancer Type on aws

## Implementation History

- AWS Network Load Balancer Support was introduced as alpha in kubernetes 1.9
- [support cross-zone load balancing](https://github.com/kubernetes/kubernetes/pull/61064)
- [support TLS termination](https://github.com/kubernetes/kubernetes/pull/74910)
- [Bug fix - SecurityGroup rule removed incorrectly](https://github.com/kubernetes/kubernetes/pull/68422)
- [Bug fix - LoadBalancerSourceRanges not working](https://github.com/kubernetes/kubernetes/pull/74692)
