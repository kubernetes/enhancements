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
last-updated: 2019-07-27
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
  - [Graduation Criteria](#graduation-criteria)
- [Implementation History](#implementation-history)
<!-- /toc -->

## Summary

Support for IPv6-only clusters was added in Kubernetes 1.9 as an alpha feature, allowing full Kubernetes capabilities using IPv6 networking instead of IPv4 networking. It also included support for Kubernetes IPv6 cluster deployments using kubeadm and support for the iptables kube-proxy backend using ip6tables. With version 1.13 the Kubernetes default DNS server changed to CoreDNS which has full IPv6 support.

## Motivation

IPv6 adoption is ramping up with the advances in IoT space and explosion in number of mobile devices, this adoption will continue to grow as we can observe at the [Google IPv6 adoption statistics](https://www.google.com/intl/en/ipv6/statistics.html).

There are cloud providers that already support IPv6 and can deploy kubernetes IPv6 only clusters, a new CI with e2e testing is running all conformance tests using [kind](https://kind.sigs.k8s.io/) and dual stack support will be added during this release cycle (1.16), that will facilitate the migration from IPv4 to IPv6.

Therefore, we would like to graduate IPv6 support from Alpha to Beta.

### Goals

* Promote IPv6 to beta.

### Non-Goals

* IPv6 is NOT Dual-stack.

### User Stories

* A user can deploy, operate and use an IPv6-only kubernetes cluster.

## Proposal

The IPv6 support introduced in 1.9 allowed full Kubernetes capabilities using IPv6 networking instead of IPv4 networking. For beta we will provide enough e2e testing to guarantee it:

* CI jobs with e2e and conformance testing will be implemented, using a virtual test environment with Kind and at least in one Cloud Provider.

* The CI jobs will publish the results on testgrid.

* The CI jobs running e2e conformance tests will be promoted to release-blokcing following the standard process, proving that they are stable and contacting SIG-release.

* To ensure there are no regression, an IPv6 e2e job will be added as a presubmit job.

## Design

### Test Plan

There is no need to develop new tests, IPv6 only clusters should pass the same e2e tests that IPv4 only clusters, guaranteeing the feature parity.

- Run E2E tests on an IPv6 only kubernetes cluster using kind.
- Run E2E tests on an IPv6 only kubernetes on at least one cloud provider.

### Graduation Criteria
- [x] It has IPv4 feature parity
- [x] It has CI using a kubernetes testing environment
- [ ] It has CI using at least one Cloud Provider
- [x] It has passed all e2e conformance tests
- [x] It is documented

## Implementation History

- [IPv6 Support was introduced as alpha in kubernetes 1.9](https://github.com/kubernetes/kubernetes/blob/master/CHANGELOG-1.9.md#ipv6)
- [IPv6 Support enhancement request](https://github.com/kubernetes/enhancements/issues/508)
- [IPv6 implementation tracking issue](https://github.com/kubernetes/kubernetes/issues/1443)
- [IPv6 CI](https://testgrid.k8s.io/conformance-kind#kind%20(IPv6),%20master%20(dev)) 
- [Kind IPv6 support](https://github.com/kubernetes-sigs/kind/pull/636)
