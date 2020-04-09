---
title: Promote Taint Based Evictions to GA
authors:
  - "@damemi"
owning-sig: sig-node
participating-sigs:
  - sig-scheduling
reviewers:
  - "@Huang-Wei"
approvers:
  - sig-node
  - sig-cloud-provider
editor:
creation-date: 2020-01-14
last-updated: 2020-03-23
status: implemented
see-also:
replaces:
superseded-by:
---

# Promote Taint Based Evictions to GA

## Table of Contents

<!-- toc -->
- [Summary](#summary)
- [Motivation](#motivation)
  - [Goals](#goals)
  - [Non-Goals](#non-goals)
- [Proposal](#proposal)
  - [Risks and Mitigations](#risks-and-mitigations)
- [Graduation Criteria](#graduation-criteria)
- [Testing Plan](#testing-plan)
  - [Unit tests](#unit-tests)
  - [Integration tests](#integration-tests)
  - [e2e tests](#e2e-tests)
- [Implementation History](#implementation-history)
<!-- /toc -->

## Summary

Taint Based Evictions was introduced as an alpha feature in Kubernetes 1.6 and was promoted to
beta in 1.13. The feature automatically taints nodes with `NoExecute` when they become unready or
unreachable.

## Motivation

The TaintNodesByCondition feature has worked to ensure nodes are tainted with `NoSchedule` effect
upon different node conditions. However, it's also required to taint nodes with `NoExecute` automatically
upon some node conditions such as node gets not ready or unreachable.

### Goals

Ensure nodes are tainted properly with a NoExecute effect when it's not ready or unreachable, so that
scheduler can use taints to make scheduling decisions consistently.

### Non-Goals

It is not the goal of taint based evictions to make any scheduling or removal decisions for pods, but rather
to monitor the nodes and ensure that the proper `NoExecute` effect is applied.

## Proposal

Ensure test coverage is sufficient.

Update feature gate logic around Taint Based Evictions to enable it by default.

Update documentation to reflect the status of the feature.


### Risks and Mitigations

There is no proposed change to the functionality of this feature, and it has functioned 
well since its promotion to beta in 1.13, so no risks are expected.


## Graduation Criteria

* The feature has been stable and reliable in the past several releases.
* Adequate documentation exists for the feature.
* Test coverage of the feature is acceptable. This includes moving existing tests to be under the appropriate sigs.

## Testing Plan

Taint based evictions is comprised of node lifecycle functions taints and
evictions, as well as the feature itself, all of which have stable unit, e2e,
and integration tests that are run regularly as part of the Kubernetes CI/CD pipeline.

### Unit tests
* [Node lifecycle controller eviction tests](https://github.com/kubernetes/kubernetes/blob/47d5c3ef8d/pkg/controller/nodelifecycle/node_lifecycle_controller_test.go#L196)

### Integration tests
* [Taint based evictions integration test](https://github.com/kubernetes/kubernetes/blob/47d5c3ef8df2b1b26da739aec0ada15d41f20cf3/test/integration/scheduler/taint_test.go#L580) (note that prior to 1.17, this test existed as an [end-to-end test](https://github.com/kubernetes/kubernetes/blob/001f2cd2b553d06028c8542c8817820ee05d657f/test/e2e/scheduling/taint_based_evictions.go)

### e2e tests
* [Scheduler taints e2e tests](https://github.com/kubernetes/kubernetes/blob/master/test/e2e/scheduling/taints.go)

## Implementation History

The original implementation of taint based evictions predates the KEP process, so discussion on it can be found here: https://github.com/kubernetes/enhancements/issues/166

