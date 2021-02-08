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
    - [How can this feature be enabled / disabled in a live cluster?](#how-can-this-feature-be-enabled--disabled-in-a-live-cluster)
    - [What specific metrics should inform a rollback?](#what-specific-metrics-should-inform-a-rollback)
  - [Monitoring Requirements](#monitoring-requirements)
    - [How can an operator determine if the feature is in use by workloads?](#how-can-an-operator-determine-if-the-feature-is-in-use-by-workloads)
    - [What are the SLIs (Service Level Indicators) an operator can use to determine the health of the service?](#what-are-the-slis-service-level-indicators-an-operator-can-use-to-determine-the-health-of-the-service)
    - [Metrics](#metrics)
  - [Dependencies](#dependencies)
    - [Does this feature depend on any specific services running in the cluster?](#does-this-feature-depend-on-any-specific-services-running-in-the-cluster)
    - [For GA, this section is required: approvers should be able to confirm the previous answers based on experience in the field.](#for-ga-this-section-is-required-approvers-should-be-able-to-confirm-the-previous-answers-based-on-experience-in-the-field)
    - [Will enabling / using this feature result in any new API calls? Describe them, providing:](#will-enabling--using-this-feature-result-in-any-new-api-calls-describe-them-providing)
  - [Troubleshooting](#troubleshooting)
    - [How does this feature react if the API server and/or etcd is unavailable?](#how-does-this-feature-react-if-the-api-server-andor-etcd-is-unavailable)
    - [What are other known failure modes?](#what-are-other-known-failure-modes)
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
- [X] (R) Production readiness review completed
- [X] (R) Production readiness review approved
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

1. [Metrics Stability Framework](https://github.com/kubernetes/enhancements/blob/77a84d2d55b5802a615f3fe98e7e7c9bd26c9efc/keps/sig-instrumentation/1209-metrics-stability/20190404-kubernetes-control-plane-metrics-stability.md)
1. [Metrics Stability Migration](https://github.com/kubernetes/enhancements/blob/77a84d2d55b5802a615f3fe98e7e7c9bd26c9efc/keps/sig-instrumentation/1209-metrics-stability/20190605-metrics-stability-migration.md)
1. [Metrics Validation and Verification](https://github.com/kubernetes/enhancements/blob/77a84d2d55b5802a615f3fe98e7e7c9bd26c9efc/keps/sig-instrumentation/1209-metrics-stability/20190605-metrics-validation-and-verification.md)
1. [Metrics Stability to Beta](https://github.com/kubernetes/enhancements/blob/77a84d2d55b5802a615f3fe98e7e7c9bd26c9efc/keps/sig-instrumentation/1209-metrics-stability/20191028-metrics-stability-to-beta.md)

This document is not net new and ties the four together in order to document the lifecycle of this feature.

[Metrics Stability Framework]: 20190404-kubernetes-control-plane-metrics-stability.md
[Metrics Stability Migration]: 20190605-metrics-stability-migration.md
[Metrics Validation and Verification]: 20190605-metrics-validation-and-verification.md
[Metrics Stability to Beta]: 20191028-metrics-stability-to-beta.md

## Motivation

See:

1. [Metrics Stability Framework#Motivation]
1. [Metrics Stability Migration#Motivation]
1. [Metrics Validation and Verification#Motivation]
1. [Metrics Stability to Beta#Motivation]

[Metrics Stability Framework#Motivation]: 20190404-kubernetes-control-plane-metrics-stability.md#motivation
[Metrics Stability Migration#Motivation]: 20190605-metrics-stability-migration.md#motivation
[Metrics Validation and Verification#Motivation]: 20190605-metrics-validation-and-verification.md#motivation
[Metrics Stability to Beta#Motivation]: 20191028-metrics-stability-to-beta.md#motivation

## Proposal

See:

1. [Metrics Stability Framework#Proposal]
1. [Metrics Stability Migration#General Migration Strategy]
1. [Metrics Validation and Verification#Proposal]
1. [Metrics Stability to Beta#Proposal]

https://github.com/kubernetes/enhancements/blob/77a84d2d55b5802a615f3fe98e7e7c9bd26c9efc/keps/sig-instrumentation/1209-metrics-stability/keps/sig-instrumentation/1209-metrics-stability/20190404-kubernetes-control-plane-metrics-stability.md#implementation-history

[Metrics Stability Framework#Proposal]: 20190404-kubernetes-control-plane-metrics-stability.md#proposal
[Metrics Stability Migration#General Migration Strategy]: 20190605-metrics-stability-migration.md#general-migration-strategy
[Metrics Validation and Verification#Proposal]: 20190605-metrics-validation-and-verification.md#proposal
[Metrics Stability to Beta#Proposal]: 20191028-metrics-stability-to-beta.md#proposal

## Design Details

See:

1. [Metrics Stability Framework#Design Details]
1. [Metrics Validation and Verification#Design Details]

[Metrics Stability Framework#Design Details]: 20190404-kubernetes-control-plane-metrics-stability.md#design-details
[Metrics Validation and Verification#Design Details]: 20190605-metrics-validation-and-verification.md#design-details

### Graduation Criteria

#### Net New -> Alpha Graduation

See:

1. [Metrics Stability Framework#Graduation Criteria]
1. [Metrics Stability Migration#Graduation Criteria]

[Metrics Stability Framework#Graduation Criteria]: 20190404-kubernetes-control-plane-metrics-stability.md#graduation-criteria
[Metrics Stability Migration#Graduation Criteria]: 20190605-metrics-stability-migration.md#graduation-criteria

#### Alpha -> Beta Graduation

See:

1. [Metrics Validation and Verification#Graduation Criteria]
1. [Metrics Stability to Beta#Graduation Criteria]

[Metrics Validation and Verification#Graduation Criteria]: 20190605-metrics-validation-and-verification.md#graduation-criteria
[Metrics Stability to Beta#Graduation Criteria]: 20191028-metrics-stability-to-beta.md#graduation-criteria

#### Beta -> GA Graduation

- Metrics are now eligible to be promoted to STABLE status (we have some candidates in kube-apiserver).
    - [apiserver_storage_object_counts](https://github.com/kubernetes/kubernetes/issues/98270)
    - `apiserver_request_total` will also be promoted (as discussed in biweekly SIG apimachinery meeting)
- Implement the ability to turn off individual metrics (see [here](20191028-metrics-stability-to-beta.md#non-goals))
    - We need this because of stuff like this: [Unbounded valuesets for metric labels](https://github.com/kubernetes/kubernetes/issues/76302)

### Upgrade / Downgrade Strategy

See:

- [Deprecation Lifecycle](20190404-kubernetes-control-plane-metrics-stability.md#deprecation-lifecycle)
- [Deprecation of modified metrics from metrics overhaul KEP](20190605-metrics-stability-migration.md#deprecation-of-modified-metrics-from-metrics-overhaul-kep)
- [Escape Hatch](20191028-metrics-stability-to-beta.md#escape-hatch)

https://github.com/kubernetes/enhancements/blob/0f5bb1138a6dfd7f3d52fa901c2fba7abb7fb731/keps/sig-instrumentation/1209-metrics-stability/keps/sig-instrumentation/1209-metrics-stability/20190404-kubernetes-control-plane-metrics-stability.md#implementation-history

### Version Skew Strategy

N/A

## Production Readiness Review Questionnaire

#### How can this feature be enabled / disabled in a live cluster?

The metrics stability framework adds developer tooling around commit pipelines and is not a user-facing feature per se. The part that is user-facing is the annotation on metrics with a stability level.

This framework intends to increase reliability in control-plane management and so features in the metrics stability framework tend to 'fix' aspects of dev processes which lead to downstream breakages.

Rollout, Upgrade and Rollback Planning
This section must be completed when targeting beta graduation to a release.

N/A, this isn't a feature per se.

#### What specific metrics should inform a rollback?

N/A

### Monitoring Requirements

#### How can an operator determine if the feature is in use by workloads?

N/A

#### What are the SLIs (Service Level Indicators) an operator can use to determine the health of the service?

N/A

#### Metrics

The stability framework applies to all metrics which originate directly from the control-plane.

### Dependencies

This section must be completed when targeting beta graduation to a release.

#### Does this feature depend on any specific services running in the cluster?

N/A

#### For GA, this section is required: approvers should be able to confirm the previous answers based on experience in the field.

#### Will enabling / using this feature result in any new API calls? Describe them, providing:

No.

### Troubleshooting

#### How does this feature react if the API server and/or etcd is unavailable?

N/A (but if the component isn't available, no metrics are being scraped).

#### What are other known failure modes?

At worst, this thing can clog the commit pipeline (since it is effectively a conformance test for ensuring metric stability guarantees). In that case, we can simply turn off the verification and validation mechanism (i.e. the `hack/verify_generated_stable_metrics.sh` script) which effectively puts us back to where we were before the framework. Note that this basically allows developers to commit breaking changes to metrics and violate guarantees though.

## Implementation History

See:

1. [Metrics Stability Framework#Implementation History]
1. [Metrics Stability Migration#Implementation History]
1. [Metrics Validation and Verification#Implementation History]
1. [Metrics Stability to Beta#Implementation History]

[Metrics Stability Framework#Implementation History]: 20190404-kubernetes-control-plane-metrics-stability.md#implementation-history
[Metrics Stability Migration#Implementation History]: 20190605-metrics-stability-migration.md#implementation-history
[Metrics Validation and Verification#Implementation History]: 20190605-metrics-validation-and-verification.md#implementation-history
[Metrics Stability to Beta#Implementation History]: 20191028-metrics-stability-to-beta.md#implementation-history
