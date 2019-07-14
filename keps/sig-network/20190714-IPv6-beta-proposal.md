---
title: graduate-ipv6-to-beta
authors:
  - "@aojea"
owning-sig: sig-network
participating-sigs:
reviewers:
  - "@bentheelder"
  - "@andrewsykim"
  - "@khenidak"
approvers:
  - "@lachie83"
  - "@thockin"
editor: TBD
creation-date: 2019-07-14
last-updated: 2019-07-23
status: implementable
see-also:
replaces:
superseded-by:
---

# Graduate IPv6 to beta

## Table of Contents

<!-- toc -->
- [Summary](#summary)
- [Motivation](#motivation)
  - [Goals](#goals)
  - [Non-Goals](#non-goals)
  - [User Stories](#user-stories)
- [Proposal](#proposal)
- [Design](#design)
  - [Test Plan](#test-plan)
    - [Needed Tests](#needed-tests)
  - [Graduation Criteria](#graduation-criteria)
- [Implementation History](#implementation-history)
<!-- /toc -->

## Summary

Support for IPv6-only clusters was added in Kubernetes 1.9 as an alpha feature, with version 1.13 the Kubernetes default DNS server changed to CoreDNS which has full IPv6 support, there are several CNI plugins with IPv6 support and Dual Stack support is being implemented during the 1.16 cycle.


## Motivation

IPv6 adoption is ramping up with the advances in IoT space and explosion in number of mobile devices, this adoption will continue to grow as we can observe at the Google IPv6 adoption statistics https://www.google.com/intl/en/ipv6/statistics.html.

There are cloud providers that are starting to support IPv6 clusters, the dual stack implementation will facilitate the migration from IPv4 to IPv6 and CI testing was implemented using the kind project.

Therefore, we would like to graduate IPv6 support from Alpha to Beta.

### Goals

* Promote IPv6 to beta version.

### Non-Goals

* IPv6 is NOT Dual-stack  

### User Stories

* A user can deploy, operate and use a kubernetes cluster in an IPv6 only environment.

## Proposal

* Make kind IPv6 e2e jobs mandatory for all PRs
* Have signal on IPv6 e2e jobs running in at least one Cloud Provider

## Design

### Test Plan

#### Needed Tests

- Run a CI with conformance E2E tests on an IPv6 only kubernetes cluster.
- Run a CI with conformance E2E tests on a cloud provider.

### Graduation Criteria
- [x] It has IPv4 feature parity
- [x] It has CI
- [x] It has passed all e2e conformance tests
- [x] It is documented
- [ ] It is being used at least by one Cloud Provider


## Implementation History

- [IPv6 Support was introduced as alpha in kubernetes 1.9](https://github.com/kubernetes/kubernetes/blob/master/CHANGELOG-1.9.md)
- [IPv6 Support enhancement request](https://github.com/kubernetes/enhancements/issues/508)
- [IPv6 implementation tracking issue](https://github.com/kubernetes/kubernetes/issues/1443)
- [IPv6 CI](https://testgrid.k8s.io/conformance-kind#kind%20(IPv6),%20master%20(dev)) 
- [Kind IPv6 support](https://github.com/kubernetes-sigs/kind/pull/636)
