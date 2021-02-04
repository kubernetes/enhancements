# KEP-1209: Metrics Stability Framework

<!-- toc -->
- [Release Signoff Checklist](#release-signoff-checklist)
- [Summary](#summary)
- [Motivation](#motivation)
- [Proposal](#proposal)
- [Design Details](#design-details)
  - [Graduation Criteria](#graduation-criteria)
    - [Net New -&gt; Alpha Graduation](#net-new---alpha-graduation)
    - [Alpha -&gt; Beta Graduation](#alpha---beta-graduation)
    - [Beta -&gt; GA Graduation](#beta---ga-graduation)
  - [Upgrade / Downgrade Strategy](#upgrade--downgrade-strategy)
  - [Version Skew Strategy](#version-skew-strategy)
- [Production Readiness Review Questionnaire](#production-readiness-review-questionnaire)
- [Implementation History](#implementation-history)
<!-- /toc -->

## Release Signoff Checklist

<!--
**ACTION REQUIRED:** In order to merge code into a release, there must be an
issue in [kubernetes/enhancements] referencing this KEP and targeting a release
milestone **before the [Enhancement Freeze](https://git.k8s.io/sig-release/releases)
of the targeted release**.

For enhancements that make changes to code or processes/procedures in core
Kubernetes—i.e., [kubernetes/kubernetes], we require the following Release
Signoff checklist to be completed.

Check these off as they are completed for the Release Team to track. These
checklist items _must_ be updated for the enhancement to be released.
-->

Items marked with (R) are required *prior to targeting to a milestone / release*.

- [X] (R) Enhancement issue in release milestone, which links to KEP dir in [kubernetes/enhancements] (not the initial KEP PR)
- [X] (R) KEP approvers have approved the KEP status as `implementable`
- [X] (R) Design details are appropriately documented
- [X] (R) Test plan is in place, giving consideration to SIG Architecture and SIG Testing input
- [X] (R) Graduation criteria is in place
- \[Predates\] (R) Production readiness review completed
- \[Predates\] (R) Production readiness review approved
- [X] "Implementation History" section is up-to-date for milestone
- [X] User-facing documentation has been created in [kubernetes/website], for publication to [kubernetes.io]
- [X] Supporting documentation—e.g., additional design documents, links to mailing list discussions/SIG meetings, relevant PRs/issues, release notes

[kubernetes.io]: https://kubernetes.io/
[kubernetes/enhancements]: https://git.k8s.io/enhancements
[kubernetes/kubernetes]: https://git.k8s.io/kubernetes
[kubernetes/website]: https://git.k8s.io/website

## Summary

This proposal covers the implementation of metrics stability in the kubernetes/kubernetes repo (and anywhere else that consumes `component-base/metrics`).

Historically, the implementation was split into four documents:

1. [Metrics Stability Framework]
1. [Metrics Stability Migration]
1. [Metrics Validation and Verification]
1. [Metrics Stability to Beta]

This document is not net new and ties the four together in order to document the lifecycle of this feature.

[Metrics Stability Framework]: keps/sig-instrumentation/1209-metrics-stability/20190404-kubernetes-control-plane-metrics-stability.md
[Metrics Stability Migration]: keps/sig-instrumentation/1209-metrics-stability/20190605-metrics-stability-migration.md
[Metrics Validation and Verification]: keps/sig-instrumentation/1209-metrics-stability/20190605-metrics-validation-and-verification.md
[Metrics Stability to Beta]: keps/sig-instrumentation/1209-metrics-stability/20191028-metrics-stability-to-beta.md

## Motivation

See:

1. [Metrics Stability Framework#Motivation]
1. [Metrics Stability Migration#Motivation]
1. [Metrics Validation and Verification#Motivation]
1. [Metrics Stability to Beta#Motivation]

[Metrics Stability Framework#Motivation]: keps/sig-instrumentation/1209-metrics-stability/20190404-kubernetes-control-plane-metrics-stability.md#motivation
[Metrics Stability Migration#Motivation]: keps/sig-instrumentation/1209-metrics-stability/20190605-metrics-stability-migration.md#motivation
[Metrics Validation and Verification#Motivation]: keps/sig-instrumentation/1209-metrics-stability/20190605-metrics-validation-and-verification.md#motivation
[Metrics Stability to Beta#Motivation]: keps/sig-instrumentation/1209-metrics-stability/20191028-metrics-stability-to-beta.md#motivation

## Proposal

See:

1. [Metrics Stability Framework#Proposal]
1. [Metrics Stability Migration#General Migration Strategy]
1. [Metrics Validation and Verification#Proposal]
1. [Metrics Stability to Beta#Proposal]

[Metrics Stability Framework#Proposal]: keps/sig-instrumentation/1209-metrics-stability/20190404-kubernetes-control-plane-metrics-stability.md#proposal
[Metrics Stability Migration#General Migration Strategy]: keps/sig-instrumentation/1209-metrics-stability/20190605-metrics-stability-migration.md#general-migration-strategy
[Metrics Validation and Verification#Proposal]: keps/sig-instrumentation/1209-metrics-stability/20190605-metrics-validation-and-verification.md#proposal
[Metrics Stability to Beta#Proposal]: keps/sig-instrumentation/1209-metrics-stability/20191028-metrics-stability-to-beta.md#proposal

## Design Details

See:

1. [Metrics Stability Framework#Design Details]
1. [Metrics Validation and Verification#Design Details]

[Metrics Stability Framework#Design Details]: keps/sig-instrumentation/1209-metrics-stability/20190404-kubernetes-control-plane-metrics-stability.md#design-details
[Metrics Validation and Verification#Design Details]: keps/sig-instrumentation/1209-metrics-stability/20190605-metrics-validation-and-verification.md#design-details

### Graduation Criteria

#### Net New -> Alpha Graduation

See:

1. [Metrics Stability Framework#Graduation Criteria]
1. [Metrics Stability Migration#Graduation Criteria]

[Metrics Stability Framework#Graduation Criteria]: keps/sig-instrumentation/1209-metrics-stability/20190404-kubernetes-control-plane-metrics-stability.md#graduation-criteria
[Metrics Stability Migration#Graduation Criteria]: keps/sig-instrumentation/1209-metrics-stability/20190605-metrics-stability-migration.md#graduation-criteria

#### Alpha -> Beta Graduation

See:

1. [Metrics Validation and Verification#Graduation Criteria]
1. [Metrics Stability to Beta#Graduation Criteria]

[Metrics Validation and Verification#Graduation Criteria]: keps/sig-instrumentation/1209-metrics-stability/20190605-metrics-validation-and-verification.md#graduation-criteria
[Metrics Stability to Beta#Graduation Criteria]: keps/sig-instrumentation/1209-metrics-stability/20191028-metrics-stability-to-beta.md#graduation-criteria

#### Beta -> GA Graduation

- Promote (some) metrics to STABLE status
    - [apiserver_storage_object_counts](https://github.com/kubernetes/kubernetes/issues/98270)
    - `apiserver_request_total` will also be promoted (as discussed in biweekly SIG apimachinery meeting)
- Implement the ability to turn off individual metrics (see [here](keps/sig-instrumentation/1209-metrics-stability/20191028-metrics-stability-to-beta.md#non-goals))
    - [Unbounded valuesets for metric labels](https://github.com/kubernetes/kubernetes/issues/76302)

### Upgrade / Downgrade Strategy

See:

- [Deprecation Lifecycle](keps/sig-instrumentation/1209-metrics-stability/20190404-kubernetes-control-plane-metrics-stability.md#deprecation-lifecycle)
- [Deprecation of modified metrics from metrics overhaul KEP](keps/sig-instrumentation/1209-metrics-stability/20190605-metrics-stability-migration.md#deprecation-of-modified-metrics-from-metrics-overhaul-kep)
- [Escape Hatch](keps/sig-instrumentation/1209-metrics-stability/20191028-metrics-stability-to-beta.md#escape-hatch)

### Version Skew Strategy

N/A

## Production Readiness Review Questionnaire

N/A - this KEP predates PRR. @logicalhan to fill this in later if desired.

## Implementation History

See:

1. [Metrics Stability Framework#Implementation History]
1. [Metrics Stability Migration#Implementation History]
1. [Metrics Validation and Verification#Implementation History]
1. [Metrics Stability to Beta#Implementation History]

[Metrics Stability Framework#Implementation History]: keps/sig-instrumentation/1209-metrics-stability/20190404-kubernetes-control-plane-metrics-stability.md#implementation-history
[Metrics Stability Migration#Implementation History]: keps/sig-instrumentation/1209-metrics-stability/20190605-metrics-stability-migration.md#implementation-history
[Metrics Validation and Verification#Implementation History]: keps/sig-instrumentation/1209-metrics-stability/20190605-metrics-validation-and-verification.md#implementation-history
[Metrics Stability to Beta#Implementation History]: keps/sig-instrumentation/1209-metrics-stability/20191028-metrics-stability-to-beta.md#implementation-history
