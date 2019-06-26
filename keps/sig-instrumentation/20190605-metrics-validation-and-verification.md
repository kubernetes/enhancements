---
title: Metrics Validation and Verification
authors:
  - "@solodov"
  - "@logicalhan"
owning-sig: sig-instrumentation
participating-sigs:
  - sig-instrumentation
  - sig-testing
reviewers:
  - @brancz
approvers:
  - @brancz
editor: TBD
creation-date: 2019-06-05
last-updated: 2019-06-05
status: provisional
see-also:
  - 20190404-kubernetes-control-plane-metrics-stability
  - 20190605-metrics-stability-migration
---

# Metrics Validation and Verification

## Table of Contents

* [Table of Contents](#table-of-contents)
* [Summary](#summary)
* [Motivation](#motivation)
   * [Goals](#goals)
   * [Non-Goals](#non-goals)
* [Proposal](#proposal)
* [Design Details](#design-details)
   * [Graduation Criteria](#graduation-criteria)
* [Implementation History](#implementation-history)

## Summary

This Kubernetes Enhancement Proposal (KEP) builds off of the framework proposed
in the [Kubernetes Control-Plane Metrics Stability
KEP](https://github.com/kubernetes/enhancements/blob/master/keps/sig-instrumentation/20190404-kubernetes-control-plane-metrics-stability.md)
and proposes a strategy for ensuring conformance of metrics with official
stability guarantees.

## Motivation

While the [Kubernetes Control Plane metrics stability KEP](https://github.com/kubernetes/enhancements/blob/master/keps/sig-instrumentation/20190404-kubernetes-control-plane-metrics-stability.md) provides a framework to define stability levels for control-plane metrics, it does not provide a strategy for verifying and validating conformance to stated guarantees. This KEP intends to propose a framework for validating and verifying metric guarantees.

### Goals

* Given a stable metric, validate that we cannot remove or modify it (other than adding deprecation information).
* Given a deprecated but stable metric, validate that we cannot remove or modify it until the deprecation period has elapsed.
* Given an alpha metric which is promoted to be 'stable', automatically include proper instrumentation reviewers (for schema validation and conformance to metrics guidelines).

### Non-Goals

* Conformance testing will not apply to alpha metrics, since alpha metrics do not have stability guarantees.

## Proposal

We will provide validation for metrics under the [new framework](https://github.com/kubernetes/enhancements/blob/master/keps/sig-instrumentation/20190404-kubernetes-control-plane-metrics-stability.md) with static analysis.

## Design Details

Metrics conformance testing will work in a similar (but not identical) fashion to the generic Kubernetes conformance tests.  Sig-instrumentation will own a directory under `test/instrumentation`. There will be a subdirectory `testdata` in which a file `stable-metrics-list.txt` will live.  This file will be owned by sig-instrumentation. Metrics conformance tests will involve a static analysis script which will traverse the entire codebase and look for metrics which are annotated as 'stable'. For each stable metrics, this script will generate a stringified version of metric metadata (i.e. name, type, labels) which will then be appended together in lexographic order. This will be the output of this script.

We will add a pre-submit check, which will run in our CI pipeline, which will run our script with the current changes and compare that to existing, committed file. If there is a difference, the pre-submit check will fail. In order to pass the pre-submit check, the original submitter of the PR will have to run a script `test/instrumentation/update-stable-metrics.sh` which will run our static analysis code and overwrite `stable-metrics-list.txt`. This will cause `sig-instrumentation` to be tagged for approvals on the PR (since they own that file).

### Graduation Criteria

TBD

## Implementation History

TBD
