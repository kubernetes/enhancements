<!--
**Note:** When your KEP is complete, all of these comment blocks should be removed.

To get started with this template:

- [ ] **Merge early and iterate.**
  Avoid getting hung up on specific details and instead aim to get the goals of
  the KEP clarified and merged quickly. The best way to do this is to just
  start with the high-level sections and fill out details incrementally in
  subsequent PRs.

Just because a KEP is merged does not mean it is complete or approved. Any KEP
marked as `provisional` is a working document and subject to change. You can
denote sections that are under active debate as follows:

```
<<[UNRESOLVED optional short context or usernames ]>>
Stuff that is being argued.
<<[/UNRESOLVED]>>
```

When editing KEPS, aim for tightly-scoped, single-topic PRs to keep discussions
focused. If you disagree with what is already in a document, open a new PR
with suggested changes.

One KEP corresponds to one "feature" or "enhancement" for its whole lifecycle.
You do not need a new KEP to move from beta to GA, for example. If
new details emerge that belong in the KEP, edit the KEP. Once a feature has become
"implemented", major changes should get new KEPs.

The canonical place for the latest set of instructions (and the likely source
of this file) is [here](/keps/NNNN-kep-template/README.md).

**Note:** Any PRs to move a KEP to `implementable`, or significant changes once
it is marked `implementable`, must be approved by each of the KEP approvers.
If none of those approvers are still appropriate, then changes to that list
should be approved by the remaining approvers and/or the owning SIG (or
SIG Architecture for cross-cutting KEPs).
-->
# KEP-5053: Fallback for HPA on failure to retrieve metrics

<!--
This is the title of your KEP. Keep it short, simple, and descriptive. A good
title can help communicate what the KEP is and should be considered as part of
any review.
-->

<!--
A table of contents is helpful for quickly jumping to sections of a KEP and for
highlighting any additional information provided beyond the standard KEP
template.

Ensure the TOC is wrapped with
  <code>&lt;!-- toc --&rt;&lt;!-- /toc --&rt;</code>
tags, and then generate with `hack/update-toc.sh`.
-->

<!-- toc -->
- [Release Signoff Checklist](#release-signoff-checklist)
- [Summary](#summary)
- [Motivation](#motivation)
  - [Goals](#goals)
  - [Non-Goals](#non-goals)
- [Proposal](#proposal)
  - [Risks and Mitigations](#risks-and-mitigations)
- [Design Details](#design-details)
  - [Test Plan](#test-plan)
      - [Prerequisite testing updates](#prerequisite-testing-updates)
      - [Unit tests](#unit-tests)
      - [Integration tests](#integration-tests)
      - [e2e tests](#e2e-tests)
  - [Graduation Criteria](#graduation-criteria)
    - [Alpha](#alpha)
  - [Upgrade / Downgrade Strategy](#upgrade--downgrade-strategy)
  - [Version Skew Strategy](#version-skew-strategy)
- [Production Readiness Review Questionnaire](#production-readiness-review-questionnaire)
  - [Feature Enablement and Rollback](#feature-enablement-and-rollback)
  - [Rollout, Upgrade and Rollback Planning](#rollout-upgrade-and-rollback-planning)
  - [Monitoring Requirements](#monitoring-requirements)
  - [Dependencies](#dependencies)
  - [Scalability](#scalability)
  - [Troubleshooting](#troubleshooting)
- [Implementation History](#implementation-history)
- [Drawbacks](#drawbacks)
- [Alternatives](#alternatives)
- [Infrastructure Needed (Optional)](#infrastructure-needed-optional)
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

- [ ] (R) Enhancement issue in release milestone, which links to KEP dir in [kubernetes/enhancements] (not the initial KEP PR)
- [ ] (R) KEP approvers have approved the KEP status as `implementable`
- [ ] (R) Design details are appropriately documented
- [ ] (R) Test plan is in place, giving consideration to SIG Architecture and SIG Testing input (including test refactors)
  - [ ] e2e Tests for all Beta API Operations (endpoints)
  - [ ] (R) Ensure GA e2e tests meet requirements for [Conformance Tests](https://github.com/kubernetes/community/blob/master/contributors/devel/sig-architecture/conformance-tests.md) 
  - [ ] (R) Minimum Two Week Window for GA e2e tests to prove flake free
- [ ] (R) Graduation criteria is in place
  - [ ] (R) [all GA Endpoints](https://github.com/kubernetes/community/pull/1806) must be hit by [Conformance Tests](https://github.com/kubernetes/community/blob/master/contributors/devel/sig-architecture/conformance-tests.md) 
- [ ] (R) Production readiness review completed
- [ ] (R) Production readiness review approved
- [ ] "Implementation History" section is up-to-date for milestone
- [ ] User-facing documentation has been created in [kubernetes/website], for publication to [kubernetes.io]
- [ ] Supporting documentation—e.g., additional design documents, links to mailing list discussions/SIG meetings, relevant PRs/issues, release notes

<!--
**Note:** This checklist is iterative and should be reviewed and updated every time this enhancement is being considered for a milestone.
-->

[kubernetes.io]: https://kubernetes.io/
[kubernetes/enhancements]: https://git.k8s.io/enhancements
[kubernetes/kubernetes]: https://git.k8s.io/kubernetes
[kubernetes/website]: https://git.k8s.io/website

## Summary

The [Horizontal Pod Autoscaler (HPA)][] relies on the controller manager to 
fetch metrics from either the resource metrics API (for per-pod resource metrics) 
or the custom metrics API (for other types of metrics). When these APIs experience 
downtime, the HPA becomes unable to make scaling decisions, potentially leaving 
workloads unmanaged.

This proposal introduces a new configuration parameter for the HPA, enabling 
users to define behavior in the event of metric retrieval failures. For example, 
users can opt to scale the target resource to the maximum number of replicas 
specified in the HPA, ensuring safer operation during metrics unavailability.

[Horizontal Pod Autoscaler (HPA)]: https://kubernetes.io/docs/tasks/run-application/horizontal-pod-autoscale/

## Motivation

The Horizontal Pod Autoscaler (HPA) is a critical component for scaling Kubernetes 
workloads based on resource utilization or custom metrics. However, the current 
implementation depends entirely on the availability of the resource metrics API 
or custom metrics API to make scaling decisions. If these APIs experience 
downtime or degradation, the HPA cannot take any scaling actions, leaving 
workloads potentially overprovisioned, underprovisioned, or entirely unmanaged.

In contrast, other autoscalers like [KEDA][] already provide mechanisms to define 
fallback strategies in the event of metric retrieval failures. These strategies 
mitigate the impact of API unavailability, enabling the autoscaler to maintain 
a functional scaling strategy even when metrics are temporarily inaccessible.

By allowing users to configure fallback behavior in HPA, this proposal aims to 
reduce the criticality of the metrics APIs and improve the overall robustness 
of the autoscaling system. This change allows users to define safe scaling 
actions, both as scaling to a predefined maximum or holding the current scale 
(current behavior), ensuring workloads remain operational and better aligned 
with user-defined requirements during unexpected disruptions.

Additionally, the community has also expressed interest in addressing this
limitation in the past. ([#109214][])

[KEDA]: https://keda.sh/docs/2.15/reference/scaledobject-spec/#fallback
[#109214]: https://github.com/kubernetes/kubernetes/issues/109214

### Goals

- Allow users to optionally define the number of replicas to scale in the case of metric retrieval failure.

### Non-Goals

- N/A

## Proposal

Heavily inspired by [KEDA][] propose to add a new field to the existing [`HorizontalPodAutoscalerBehavior`][] object:

- `fallback`: an optional new object containing the following fields:
  - `failureThreshold`: (integer) the number of failures fetching metrics to trigger the fallback behavior. Must be a value greater than 0. This field is optional and defaults to 3 if not specified.
  - `replicas`: (integer) the number of replicas to scale to in case of fallback. Must be greater than 0 and it's mandatory.

To allow for tracking of failures to fetch metrics a new field should be added to the existing [`HorizontalPodAutoscalerStatus`][] object:
- `consecutiveMetricRetrievalFailureCount`: (integer) tracks the number of consecutive failures in retrieving metrics.

When the `behavior` field on the [`HorizontalPodAutoscalerSpec`][] or the `fallback` field in the [`HorizontalPodAutoscalerBehavior`][] 
are not specified, the current behavior is preserved, meaning no scaling operations will occur in the event of a metrics retrieval failure.

[KEDA]: https://keda.sh/docs/2.15/reference/scaledobject-spec/#fallback
[HorizontalPodAutoscalerBehavior]: https://kubernetes.io/docs/reference/generated/kubernetes-api/v1.31/#horizontalpodautoscalerbehavior-v2-autoscaling
[HorizontalPodAutoscalerStatus]: https://kubernetes.io/docs/reference/generated/kubernetes-api/v1.31/#horizontalpodautoscalerstatus-v2-autoscaling
[HorizontalPodAutoscalerSpec]: https://kubernetes.io/docs/reference/generated/kubernetes-api/v1.31/#horizontalpodautoscalerspec-v2-autoscaling

### Risks and Mitigations

There should be minimal risk introduced by the proposed changes:
- The new field is optional, and its absence results in no changes to the current autoscaling behavior
- If a change to the new field results in undesirable behavior, the change can be reverted by deploying the previous version of the HPA resource, or removing the `fallback` field entirely.

## Design Details

The `HorizontalPodAutoscaler` API is updated to have a new object `HPAFallback`:

```golang
type HPAFallback struct {
    // failureThreshold is the number of failures fetching metrics to trigger the 
    // fallback behavior.
    // +optional
    FailureThreshold *int32
    
    // failureThreshold is the number of replicas to scale to in case of fallback.
    Replicas int32
}
```

The `HorizontalPodAutoscaler` API is updated to add a new `fallback` field to the `HorizontalPodAutoscalerBehavior` object:

```golang
type HorizontalPodAutoscalerBehavior struct {
    // fallback specifies the number of replicas to scale the object to during a 
    // fallback state and defines the threshold for errors required to enter the 
    // fallback state.
    //+optional
    Fallback *HPAFallback

    // Existing fields.
    ScaleUp *HPAScalingRules
    ScaleDown *HPAScalingRules
}
```

The `HorizontalPodAutoscaler` API is updated to have a new description of the `behavior` field on the `HorizontalPodAutoscalerSpec` object:

```golang
type HorizontalPodAutoscalerSpec struct {
    // behavior configures the scaling behavior of the target, including 
	// scale-up and scale-down policies, as well as fallback behavior in case
	// of metric retrieval failures. If not set, the default HPAScalingRules 
	// are used for scaling decisions, and no scaling operation will occur 
	// when metrics retrieval fails.
    // +optional
    Behavior *HorizontalPodAutoscalerBehavior

    // Existing fields.
    ScaleTargetRef CrossVersionObjectReference
    MinReplicas *int32
    MaxReplicas int32
    Metrics []MetricSpec
}
```

The `HorizontalPodAutoscaler` API is updated to add a new `fallback` field to the `HorizontalPodAutoscalerStatus` object:

```golang
type HorizontalPodAutoscalerStatus struct {
    // consecutiveMetricRetrievalFailureCount tracks the number of consecutive failures in retrieving metrics. 
    //+optional
    ConsecutiveMetricRetrievalFailureCount int32
    
    // Existing fields.
    ObservedGeneration *int64
    LastScaleTime *metav1.Time
    CurrentReplicas int32
    DesiredReplicas int32
    CurrentMetrics []MetricStatus
    Conditions []HorizontalPodAutoscalerCondition
}
```
The `HorizontalPodAutoscaler` API is updated to introduce a new FallbackActive condition to the `HorizontalPodAutoscalerConditionType`:

```golang
const (
    // FallbackActive indicates that the HPA has entered the fallback state due to repeated
    // metric retrieval failures and is applying the configured fallback behavior.
    FallbackActive HorizontalPodAutoscalerConditionType = "FallbackActive"

    // Existing conditions
    ScalingActive HorizontalPodAutoscalerConditionType = "ScalingActive"
    AbleToScale HorizontalPodAutoscalerConditionType = "AbleToScale"
    ScalingLimited HorizontalPodAutoscalerConditionType = "ScalingLimited"
)
```

The new fallback field will be used in the autoscaling controller
[horizontal.go][]. The current logic is:

```golang
if err != nil && metricDesiredReplicas == -1 {
    a.setCurrentReplicasAndMetricsInStatus(hpa, currentReplicas, metricStatuses)
    if err := a.updateStatusIfNeeded(ctx, hpaStatusOriginal, hpa); err != nil {
        utilruntime.HandleError(err)
    }
    a.eventRecorder.Event(hpa, v1.EventTypeWarning, "FailedComputeMetricsReplicas", err.Error())
    return fmt.Errorf("failed to compute desired number of replicas based on listed metrics for %s: %v", reference, err)
}
```

It will be replaced by:

```golang
if err != nil && metricDesiredReplicas == -1 {
    a.increaseConsecutiveMetricRetrievalFailureCount(hpa)
    a.eventRecorder.Event(hpa, v1.EventTypeWarning, "FailedComputeMetricsReplicas", err.Error())
    
    var inFallback bool
    
    if hpa.Spec.Fallback != nil {
        var failureThreshold int32
        
        if hpa.Spec.Fallback.FailureThreshold != nil {
            failureThreshold = *hpa.Spec.Fallback.FailureThreshold
        } else {
            // Default value
            failureThreshold = 3
        }
        
        if failureThreshold < hpa.Status.ConsecutiveMetricRetrievalFailureCount {
            inFallback = true
            metricDesiredReplicas = hpa.Spec.Fallback.Replicas
            a.eventRecorder.Event(hpa, v1.EventTypeWarning, "FallbackThresholdReached", err.Error())
            setCondition(hpa, autoscalingv2.FallbackActive, v1.ConditionTrue, "FallbackThresholdReached", "%s", err.Error())
        } else {
            setCondition(hpa, autoscalingv2.FallbackActive, v1.ConditionFalse, "FallbackThresholdNotReached", "Threshold is set to %d failures. Current failure count is %d", failureThreshold, hpa.Status.ConsecutiveMetricRetrievalFailureCount)
            inFallback = false
        }
    } else {
        setCondition(hpa, autoscalingv2.FallbackActive, v1.ConditionFalse, "NoFallbackDefined", "No fallback behavior is defined")
        inFallback = false
    }
    
    if !inFallback {
        a.setCurrentReplicasAndMetricsInStatus(hpa, currentReplicas, metricStatuses)
        if err := a.updateStatusIfNeeded(ctx, hpaStatusOriginal, hpa); err != nil {
            utilruntime.HandleError(err)
        }
        return fmt.Errorf("failed to compute desired number of replicas based on listed metrics for %s: %v", reference, err)
    }
}
setCondition(hpa, autoscalingv2.FallbackActive, v1.ConditionFalse, "SucceededToComputeDesiredReplicas", "the HPA controller was able to compute the desired replicas")
```

[horizontal.go]: https://github.com/kubernetes/kubernetes/blob/master/pkg/controller/podautoscaler/horizontal.go

### Test Plan

<!--
**Note:** *Not required until targeted at a release.*
The goal is to ensure that we don't accept enhancements with inadequate testing.

All code is expected to have adequate tests (eventually with coverage
expectations). Please adhere to the [Kubernetes testing guidelines][testing-guidelines]
when drafting this test plan.

[testing-guidelines]: https://git.k8s.io/community/contributors/devel/sig-testing/testing.md
-->

[ ] I/we understand the owners of the involved components may require updates to
existing tests to make this code solid enough prior to committing the changes necessary
to implement this enhancement.

##### Prerequisite testing updates

<!--
Based on reviewers feedback describe what additional tests need to be added prior
implementing this enhancement to ensure the enhancements have also solid foundations.
-->

##### Unit tests

<!--
In principle every added code should have complete unit test coverage, so providing
the exact set of tests will not bring additional value.
However, if complete unit test coverage is not possible, explain the reason of it
together with explanation why this is acceptable.
-->

<!--
Additionally, for Alpha try to enumerate the core package you will be touching
to implement this enhancement and provide the current unit coverage for those
in the form of:
- <package>: <date> - <current test coverage>
The data can be easily read from:
https://testgrid.k8s.io/sig-testing-canaries#ci-kubernetes-coverage-unit

This can inform certain test coverage improvements that we want to do before
extending the production code to implement this enhancement.
-->

- `<package>`: `<date>` - `<test coverage>`

##### Integration tests

<!--
Integration tests are contained in k8s.io/kubernetes/test/integration.
Integration tests allow control of the configuration parameters used to start the binaries under test.
This is different from e2e tests which do not allow configuration of parameters.
Doing this allows testing non-default options and multiple different and potentially conflicting command line options.
-->

<!--
This question should be filled when targeting a release.
For Alpha, describe what tests will be added to ensure proper quality of the enhancement.

For Beta and GA, add links to added tests together with links to k8s-triage for those tests:
https://storage.googleapis.com/k8s-triage/index.html
-->

- <test>: <link to test coverage>

##### e2e tests

<!--
This question should be filled when targeting a release.
For Alpha, describe what tests will be added to ensure proper quality of the enhancement.

For Beta and GA, add links to added tests together with links to k8s-triage for those tests:
https://storage.googleapis.com/k8s-triage/index.html

We expect no non-infra related flakes in the last month as a GA graduation criteria.
-->

Will the follow [e2e autoscaling tests]:

- Failure of retrieving metrics over the threshold scales the resource with the configured replicas
- Success in retrieving metrics should reset the `ConsecutiveMetricRetrievalFailureCount` in the `HorizontalPodAutoscalerStatus`
- When `fallback` is not set the resource should not scale when failing to retrieve metrics

[e2e autoscaling tests]: https://github.com/kubernetes/kubernetes/tree/master/test/e2e/autoscaling

### Graduation Criteria

<!--
**Note:** *Not required until targeted at a release.*

Define graduation milestones.

These may be defined in terms of API maturity, [feature gate] graduations, or as
something else. The KEP should keep this high-level with a focus on what
signals will be looked at to determine graduation.

Consider the following in developing the graduation criteria for this enhancement:
- [Maturity levels (`alpha`, `beta`, `stable`)][maturity-levels]
- [Feature gate][feature gate] lifecycle
- [Deprecation policy][deprecation-policy]

Clearly define what graduation means by either linking to the [API doc
definition](https://kubernetes.io/docs/concepts/overview/kubernetes-api/#api-versioning)
or by redefining what graduation means.

In general we try to use the same stages (alpha, beta, GA), regardless of how the
functionality is accessed.

[feature gate]: https://git.k8s.io/community/contributors/devel/sig-architecture/feature-gates.md
[maturity-levels]: https://git.k8s.io/community/contributors/devel/sig-architecture/api_changes.md#alpha-beta-and-stable-versions
[deprecation-policy]: https://kubernetes.io/docs/reference/using-api/deprecation-policy/

Below are some examples to consider, in addition to the aforementioned [maturity levels][maturity-levels].

#### Alpha

- Feature implemented behind a feature flag
- Initial e2e tests completed and enabled

#### Beta

- Gather feedback from developers and surveys
- Complete features A, B, C
- Additional tests are in Testgrid and linked in KEP

#### GA

- N examples of real-world usage
- N installs
- More rigorous forms of testing—e.g., downgrade tests and scalability tests
- Allowing time for feedback

**Note:** Generally we also wait at least two releases between beta and
GA/stable, because there's no opportunity for user feedback, or even bug reports,
in back-to-back releases.

**For non-optional features moving to GA, the graduation criteria must include
[conformance tests].**

[conformance tests]: https://git.k8s.io/community/contributors/devel/sig-architecture/conformance-tests.md

#### Deprecation

- Announce deprecation and support policy of the existing flag
- Two versions passed since introducing the functionality that deprecates the flag (to address version skew)
- Address feedback on usage/changed behavior, provided on GitHub issues
- Deprecate the flag
-->

#### Alpha

- Feature implemented behind a `HPAFallback` feature flag
- Initial e2e tests completed and enabled

### Upgrade / Downgrade Strategy

When the feature flag is enabled, the `kube-controller-manager` should begin 
counting concurrent failures starting from 0. If the feature flag is disabled, 
the status should always reflect `MetricRetrievalFailureCount` as 0.

All logic related to metric retrieval failure and `MetricRetrievalFailureCount` 
evaluation must be gated by the same feature flag. This means that if the feature 
flag is rolled back, any ongoing metrics retrieval failures will not affect scaling 
behavior, and the resource will continue with the same scale as it did prior to 
the feature being disabled.

<!--
If applicable, how will the component be upgraded and downgraded? Make sure
this is in the test plan.

Consider the following in developing an upgrade/downgrade strategy for this
enhancement:
- What changes (in invocations, configurations, API use, etc.) is an existing
  cluster required to make on upgrade, in order to maintain previous behavior?
- What changes (in invocations, configurations, API use, etc.) is an existing
  cluster required to make on upgrade, in order to make use of the enhancement?
-->

### Version Skew Strategy

<!--
If applicable, how will the component handle version skew with other
components? What are the guarantees? Make sure this is in the test plan.

Consider the following in developing a version skew strategy for this
enhancement:
- Does this enhancement involve coordinating behavior in the control plane and nodes?
- How does an n-3 kubelet or kube-proxy without this feature available behave when this feature is used?
- How does an n-1 kube-controller-manager or kube-scheduler without this feature available behave when this feature is used?
- Will any other components on the node change? For example, changes to CSI,
  CRI or CNI may require updating that component before the kubelet.
-->

## Production Readiness Review Questionnaire

<!--

Production readiness reviews are intended to ensure that features merging into
Kubernetes are observable, scalable and supportable; can be safely operated in
production environments, and can be disabled or rolled back in the event they
cause increased failures in production. See more in the PRR KEP at
https://git.k8s.io/enhancements/keps/sig-architecture/1194-prod-readiness.

The production readiness review questionnaire must be completed and approved
for the KEP to move to `implementable` status and be included in the release.

In some cases, the questions below should also have answers in `kep.yaml`. This
is to enable automation to verify the presence of the review, and to reduce review
burden and latency.

The KEP must have a approver from the
[`prod-readiness-approvers`](http://git.k8s.io/enhancements/OWNERS_ALIASES)
team. Please reach out on the
[#prod-readiness](https://kubernetes.slack.com/archives/CPNHUMN74) channel if
you need any help or guidance.
-->

### Feature Enablement and Rollback

<!--
This section must be completed when targeting alpha to a release.
-->

###### How can this feature be enabled / disabled in a live cluster?

- [x] Feature gate (also fill in values in `kep.yaml`)
  - Feature gate name: HPAFallback
  - Components depending on the feature gate: `kube-controller-manager`

###### Does enabling the feature change any default behavior?

No.

###### Can the feature be disabled once it has been enabled (i.e. can we roll back the enablement)?

Yes.

###### What happens if we reenable the feature if it was previously rolled back?

###### Are there any tests for feature enablement/disablement?

<!--
The e2e framework does not currently support enabling or disabling feature
gates. However, unit tests in each component dealing with managing data, created
with and without the feature, are necessary. At the very least, think about
conversion tests if API types are being modified.

Additionally, for features that are introducing a new API field, unit tests that
are exercising the `switch` of feature gate itself (what happens if I disable a
feature gate after having objects written with the new field) are also critical.
You can take a look at one potential example of such test in:
https://github.com/kubernetes/kubernetes/pull/97058/files#diff-7826f7adbc1996a05ab52e3f5f02429e94b68ce6bce0dc534d1be636154fded3R246-R282
-->

### Rollout, Upgrade and Rollback Planning

<!--
This section must be completed when targeting beta to a release.
-->

###### How can a rollout or rollback fail? Can it impact already running workloads?

<!--
Try to be as paranoid as possible - e.g., what if some components will restart
mid-rollout?

Be sure to consider highly-available clusters, where, for example,
feature flags will be enabled on some API servers and not others during the
rollout. Similarly, consider large clusters and how enablement/disablement
will rollout across nodes.
-->

###### What specific metrics should inform a rollback?

<!--
What signals should users be paying attention to when the feature is young
that might indicate a serious problem?
-->

###### Were upgrade and rollback tested? Was the upgrade->downgrade->upgrade path tested?

<!--
Describe manual testing that was done and the outcomes.
Longer term, we may want to require automated upgrade/rollback tests, but we
are missing a bunch of machinery and tooling and can't do that now.
-->

###### Is the rollout accompanied by any deprecations and/or removals of features, APIs, fields of API types, flags, etc.?

<!--
Even if applying deprecation policies, they may still surprise some users.
-->

### Monitoring Requirements

<!--
This section must be completed when targeting beta to a release.

For GA, this section is required: approvers should be able to confirm the
previous answers based on experience in the field.
-->

###### How can an operator determine if the feature is in use by workloads?

<!--
Ideally, this should be a metric. Operations against the Kubernetes API (e.g.,
checking if there are objects with field X set) may be a last resort. Avoid
logs or events for this purpose.
-->

###### How can someone using this feature know that it is working for their instance?

<!--
For instance, if this is a pod-related feature, it should be possible to determine if the feature is functioning properly
for each individual pod.
Pick one more of these and delete the rest.
Please describe all items visible to end users below with sufficient detail so that they can verify correct enablement
and operation of this feature.
Recall that end users cannot usually observe component logs or access metrics.
-->

- [ ] Events
  - Event Reason: 
- [ ] API .status
  - Condition name: 
  - Other field: 
- [ ] Other (treat as last resort)
  - Details:

###### What are the reasonable SLOs (Service Level Objectives) for the enhancement?

<!--
This is your opportunity to define what "normal" quality of service looks like
for a feature.

It's impossible to provide comprehensive guidance, but at the very
high level (needs more precise definitions) those may be things like:
  - per-day percentage of API calls finishing with 5XX errors <= 1%
  - 99% percentile over day of absolute value from (job creation time minus expected
    job creation time) for cron job <= 10%
  - 99.9% of /health requests per day finish with 200 code

These goals will help you determine what you need to measure (SLIs) in the next
question.
-->

###### What are the SLIs (Service Level Indicators) an operator can use to determine the health of the service?

<!--
Pick one more of these and delete the rest.
-->

- [ ] Metrics
  - Metric name:
  - [Optional] Aggregation method:
  - Components exposing the metric:
- [ ] Other (treat as last resort)
  - Details:

###### Are there any missing metrics that would be useful to have to improve observability of this feature?

<!--
Describe the metrics themselves and the reasons why they weren't added (e.g., cost,
implementation difficulties, etc.).
-->

### Dependencies

<!--
This section must be completed when targeting beta to a release.
-->

###### Does this feature depend on any specific services running in the cluster?

<!--
Think about both cluster-level services (e.g. metrics-server) as well
as node-level agents (e.g. specific version of CRI). Focus on external or
optional services that are needed. For example, if this feature depends on
a cloud provider API, or upon an external software-defined storage or network
control plane.

For each of these, fill in the following—thinking about running existing user workloads
and creating new ones, as well as about cluster-level services (e.g. DNS):
  - [Dependency name]
    - Usage description:
      - Impact of its outage on the feature:
      - Impact of its degraded performance or high-error rates on the feature:
-->

### Scalability

<!--
For alpha, this section is encouraged: reviewers should consider these questions
and attempt to answer them.

For beta, this section is required: reviewers must answer these questions.

For GA, this section is required: approvers should be able to confirm the
previous answers based on experience in the field.
-->

###### Will enabling / using this feature result in any new API calls?

<!--
Describe them, providing:
  - API call type (e.g. PATCH pods)
  - estimated throughput
  - originating component(s) (e.g. Kubelet, Feature-X-controller)
Focusing mostly on:
  - components listing and/or watching resources they didn't before
  - API calls that may be triggered by changes of some Kubernetes resources
    (e.g. update of object X triggers new updates of object Y)
  - periodic API calls to reconcile state (e.g. periodic fetching state,
    heartbeats, leader election, etc.)
-->

###### Will enabling / using this feature result in introducing new API types?

<!--
Describe them, providing:
  - API type
  - Supported number of objects per cluster
  - Supported number of objects per namespace (for namespace-scoped objects)
-->

###### Will enabling / using this feature result in any new calls to the cloud provider?

<!--
Describe them, providing:
  - Which API(s):
  - Estimated increase:
-->

###### Will enabling / using this feature result in increasing size or count of the existing API objects?

<!--
Describe them, providing:
  - API type(s):
  - Estimated increase in size: (e.g., new annotation of size 32B)
  - Estimated amount of new objects: (e.g., new Object X for every existing Pod)
-->

###### Will enabling / using this feature result in increasing time taken by any operations covered by existing SLIs/SLOs?

<!--
Look at the [existing SLIs/SLOs].

Think about adding additional work or introducing new steps in between
(e.g. need to do X to start a container), etc. Please describe the details.

[existing SLIs/SLOs]: https://git.k8s.io/community/sig-scalability/slos/slos.md#kubernetes-slisslos
-->

###### Will enabling / using this feature result in non-negligible increase of resource usage (CPU, RAM, disk, IO, ...) in any components?

<!--
Things to keep in mind include: additional in-memory state, additional
non-trivial computations, excessive access to disks (including increased log
volume), significant amount of data sent and/or received over network, etc.
This through this both in small and large cases, again with respect to the
[supported limits].

[supported limits]: https://git.k8s.io/community//sig-scalability/configs-and-limits/thresholds.md
-->

###### Can enabling / using this feature result in resource exhaustion of some node resources (PIDs, sockets, inodes, etc.)?

<!--
Focus not just on happy cases, but primarily on more pathological cases
(e.g. probes taking a minute instead of milliseconds, failed pods consuming resources, etc.).
If any of the resources can be exhausted, how this is mitigated with the existing limits
(e.g. pods per node) or new limits added by this KEP?

Are there any tests that were run/should be run to understand performance characteristics better
and validate the declared limits?
-->

### Troubleshooting

<!--
This section must be completed when targeting beta to a release.

For GA, this section is required: approvers should be able to confirm the
previous answers based on experience in the field.

The Troubleshooting section currently serves the `Playbook` role. We may consider
splitting it into a dedicated `Playbook` document (potentially with some monitoring
details). For now, we leave it here.
-->

###### How does this feature react if the API server and/or etcd is unavailable?

###### What are other known failure modes?

<!--
For each of them, fill in the following information by copying the below template:
  - [Failure mode brief description]
    - Detection: How can it be detected via metrics? Stated another way:
      how can an operator troubleshoot without logging into a master or worker node?
    - Mitigations: What can be done to stop the bleeding, especially for already
      running user workloads?
    - Diagnostics: What are the useful log messages and their required logging
      levels that could help debug the issue?
      Not required until feature graduated to beta.
    - Testing: Are there any tests for failure mode? If not, describe why.
-->

###### What steps should be taken if SLOs are not being met to determine the problem?

## Implementation History

<!--
Major milestones in the lifecycle of a KEP should be tracked in this section.
Major milestones might include:
- the `Summary` and `Motivation` sections being merged, signaling SIG acceptance
- the `Proposal` section being merged, signaling agreement on a proposed design
- the date implementation started
- the first Kubernetes release where an initial version of the KEP was available
- the version of Kubernetes where the KEP graduated to general availability
- when the KEP was retired or superseded
-->

## Drawbacks

<!--
Why should this KEP _not_ be implemented?
-->

## Alternatives

<!--
What other approaches did you consider, and why did you rule them out? These do
not need to be as detailed as the proposal, but should include enough
information to express the idea and why it was not acceptable.
-->

## Infrastructure Needed (Optional)

<!--
Use this section if you need things from the project/SIG. Examples include a
new subproject, repos requested, or GitHub details. Listing these here allows a
SIG to get the process for these resources started right away.
-->
