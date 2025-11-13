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
# KEP-5053: Fallback for HPA External Metrics on Retrieval Failure

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
  - [User Stories](#user-stories)
    - [Story 1: SaaS Application Scaling on Queue Depth](#story-1-saas-application-scaling-on-queue-depth)
    - [Story 2: E-commerce Site with Multiple External Metrics](#story-2-e-commerce-site-with-multiple-external-metrics)
  - [Risks and Mitigations](#risks-and-mitigations)
- [Design Details](#design-details)
  - [Test Plan](#test-plan)
      - [Prerequisite testing updates](#prerequisite-testing-updates)
      - [Unit tests](#unit-tests)
      - [Integration tests](#integration-tests)
      - [e2e tests](#e2e-tests)
  - [Graduation Criteria](#graduation-criteria)
    - [Alpha](#alpha)
    - [Beta](#beta)
    - [GA](#ga)
  - [Upgrade / Downgrade Strategy](#upgrade--downgrade-strategy)
    - [Upgrade](#upgrade)
    - [Downgrade](#downgrade)
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

- [x] (R) Enhancement issue in release milestone, which links to KEP dir in [kubernetes/enhancements] (not the initial KEP PR)
- [ ] (R) KEP approvers have approved the KEP status as `implementable`
- [x] (R) Design details are appropriately documented
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

## Summary

The Horizontal Pod Autoscaler's reliance on external metrics creates a dependency on systems outside the Kubernetes cluster's control. These external systems (cloud provider APIs, third-party monitoring systems, message brokers, etc.) may experience:

- Network connectivity issues
- Rate limiting
- Service outages
- Authentication/authorization failures
- Degraded performance

When external metrics become unavailable, the HPA cannot make informed scaling decisions, which can lead to:
- Workloads stuck at insufficient scale during traffic spikes
- Inability to respond to critical business metrics (e.g., queue depth, error rates)
- Over-dependence on external system reliability

Unlike in-cluster resource metrics (CPU, memory) served by metrics-server, which are part of the cluster's core infrastructure, external metrics are inherently less reliable and outside the cluster operator's direct control.

## Motivation

The Horizontal Pod Autoscaler (HPA) supports scaling workloads based on external metrics—metrics that originate from systems outside the Kubernetes cluster's control. These external systems include:

- Cloud provider APIs (e.g., AWS CloudWatch, Azure Monitor, GCP Monitoring)
- Third-party monitoring systems (e.g., Datadog, New Relic, Prometheus running externally)
- Message brokers and queues (e.g., AWS SQS, RabbitMQ, Kafka)
- Application-specific metrics services

Unlike in-cluster resource metrics (CPU, memory served by metrics-server) or custom/object metrics (served by in-cluster custom metrics APIs), external metrics are inherently less reliable because they depend on systems outside the cluster operator's direct control. These external systems may experience:

- Network connectivity issues between the cluster and the external service
- Rate limiting or throttling
- Service outages or degraded performance
- Authentication/authorization failures
- Regional or availability zone failures

When external metrics become unavailable, the HPA cannot make informed scaling decisions. Currently, the HPA simply maintains the current replica count and waits for metrics to become available again. This behavior can lead to:

- Workloads stuck at insufficient scale during traffic spikes when metrics are unavailable
- Inability to respond to critical business events (e.g., growing queue depth, increasing error rates)
- Production incidents caused by external metrics provider outages
- Over-dependence on the reliability of external systems for critical autoscaling functionality
Other autoscalers in the ecosystem, such as [KEDA](https://keda.sh/), already provide fallback mechanisms for external metrics to mitigate these availability issues. By allowing users to configure fallback behavior for external metrics in HPA, this proposal aims to:

- Reduce the criticality of external metrics providers on cluster workload scaling
- Improve the overall robustness of autoscaling for workloads that depend on external signals
- Enable users to define safe, conservative scaling actions when external metrics are temporarily unavailable
- Maintain workload availability and performance during external metrics provider disruptions

**Why Duration-Based Instead of Count-Based:**

Different Kubernetes providers and configurations may poll external metrics at different frequencies. The HPA reconciliation loop typically runs every 15 seconds by default (configurable via `--horizontal-pod-autoscaler-sync-period`), but this can vary between clusters. A count-based threshold (e.g., "3 failures") would result in inconsistent behavior:
- In a cluster polling every 15s: 3 failures = 45 seconds
- In a cluster polling every 30s: 3 failures = 90 seconds
- If polling frequency changes, behavior changes unexpectedly

A duration-based threshold provides consistent, predictable behavior regardless of:
- HPA controller reconciliation frequency
- Kubernetes provider configurations
- Cluster-specific settings

The duration is measured from the first consecutive failure, ensuring consistent and understandable semantics: "activate fallback if the metric has been failing for at least X minutes."

This enhancement allows users to specify a desired replica count that the HPA should use after a configurable number of consecutive failures to retrieve an external metric. The fallback replica count is treated as the desired replica count from that metric and combined with other metrics using the HPA's standard multi-metric algorithm (taking the maximum), respecting all configured constraints (min/max replicas, behavior policies, etc.), ensuring predictable and safe scaling decisions even when external metrics are unavailable.

The community has previously expressed interest in addressing this limitation [#109214](https://github.com/kubernetes/kubernetes/issues/109214).

### Goals

- Allow users to optionally define fallback values for external metrics when retrieval fails
- Provide per-metric failure tracking and fallback behavior
- Maintain the HPA's scaling algorithm and respect min/max replica constraints
- Ensure users can determine which specific metrics are using fallback values

### Non-Goals

- Fallback for resource metrics (CPU, memory from metrics-server) - these are in-cluster and should be addressed at the infrastructure level if unavailable
- Fallback for pods/object metrics - these use in-cluster APIs
- Fallback for custom metrics - may be considered in future based on alpha feedback
- Last-known-good metric value caching
- Automatic fallback value calculation
- Changing the HPA scaling algorithm

## Proposal

Add optional fallback configuration to the [ExternalMetricSource](https://github.com/kubernetes/kubernetes/blob/48c56e04e0bc2cdc33eb67ee36ca69eba96b5d0b/staging/src/k8s.io/api/autoscaling/v2/types.go#L343) type, allowing users to specify:

1. A failure duration (how long the metric must be continuously failing before activating fallback)
2. A substitute metric value to use when the threshold is exceeded

This approach:
- **Works with the HPA algorithm**: Fallback provides a desired replica count for that metric, which is combined with other metrics using the standard HPA multi-metric approach (taking the maximum)
- **Is per-metric**: Each external metric can have its own fallback configuration
- **Provides visibility**: Status shows which metrics are in fallback state
- **Is conservative**: Only applies to external metrics, which are inherently out-of-cluster
- **Is consistent**: Duration-based thresholds behave the same across different Kubernetes configurations and reconciliation frequencies

### User Stories

#### Story 1: SaaS Application Scaling on Queue Depth

I run a SaaS application that scales based on a cloud provider's message queue depth (external metric). Occasionally, the cloud provider's metrics API experiences brief outages (5-10 minutes). During these outages, my HPA cannot scale, and customer requests queue up. 

With this feature, I can configure:
```yaml
metrics:
- type: External
  external:
    metric:
      name: queue_depth
    target:
      type: AverageValue
      averageValue: "30"
    fallback:
      failureDuration: 3m # Activate fallback after 3 minutes of consecutive failures
      replicas: 10  # Scale to 10 replicas to handle presumed backlog
```

When the external API fails, the HPA treats this metric as requesting 10 replicas, ensuring sufficient capacity to handle the presumed backlog safely.

#### Story 2: E-commerce Site with Multiple External Metrics

My e-commerce site scales on both external error rates and external request latency from a third-party monitoring system. I want different fallback strategies:

```yaml
metrics:
- type: External
  external:
    metric:
      name: error_rate
    target:
      type: Value
      value: "0.01"  # 1% error rate
    fallback:
      failureDuration: 5m  # Activate after 5 minutes
      replicas: 15  # Scale to 15 replicas assuming higher load
- type: External
  external:
    metric:
      name: p99_latency_ms
    target:
      type: Value
      value: "200"
    fallback:
      failureDuration: 3m  # Activate after 3 minutes
      replicas: 12  # Scale to 12 replicas assuming higher load
```

If only one metric fails, the HPA continues using the healthy metric while treating the failed one as requesting its configured fallback replica count. The HPA takes the maximum of all desired replica counts (standard multi-metric behavior).

### Risks and Mitigations

- Risk: Users configure inappropriate fallback replica counts
  - Mitigation: Documentation with best practices; validation ensures replicas > 0; HPA min/max constraints still apply; users should consider peak load scenarios when setting fallback values

- Risk: Users configure failureDuration too short, causing premature fallback activation
  - Mitigation: Default value of 3 minutes provides reasonable buffer; validation enforces minimum values; documentation recommends considering normal metric provider latency and transient failures
  
- Risk: Users configure failureDuration too long, delaying necessary scaling during outages
  - Mitigation: Documentation provides guidance on balancing between avoiding false positives and responding quickly to genuine outages; recommend 3-5 minutes for most use cases
  
- Risk: Complexity in understanding which metric is in fallback and why
  - Mitigation: Per-metric status clearly shows fallback state, `firstFailureTime` timestamp, and current `fallbackReplicas` value; events are generated when fallback activates with clear messaging including duration and timestamp

## Design Details

Add a new `ExternalMetricFallback` type and include it in `ExternalMetricSource`:

```golang
// ExternalMetricFallback defines fallback behavior when an external metric cannot be retrieved
type ExternalMetricFallback struct {
    // failureDuration is the duration for which the external metric must be continuously
    // failing before the fallback value is used. The duration is measured from the first
    // consecutive failure. Must be greater than 0.
    // +optional
    // +kubebuilder:default="3m"
    FailureDuration *metav1.Duration `json:"failureDuration,omitempty"`
    
    // replicas is the desired replica count to use when the external metric cannot be retrieved.
    // This value is treated as the desired replica count from this metric.
    // When multiple metrics are configured, the HPA controller uses the maximum of all 
    // desired replica counts (standard HPA multi-metric behavior).
    // Must be greater than 0.
    // +required
    Replicas int32 `json:"replicas"`
}

// ExternalMetricSource indicates how to scale on a metric not associated with
// any Kubernetes object (for example length of queue in cloud
// messaging service, or QPS from loadbalancer running outside of cluster).
type ExternalMetricSource struct {
	// metric identifies the target metric by name and selector
	Metric MetricIdentifier `json:"metric" protobuf:"bytes,1,name=metric"`

	// target specifies the target value for the given metric
	Target MetricTarget `json:"target" protobuf:"bytes,2,name=target"`

  // fallback defines the behavior when this external metric cannot be retrieved.
  // If not set, the HPA will not scale based on this metric when it's unavailable.
  // +optional
  Fallback *ExternalMetricFallback `json:"fallback,omitempty"`
}
```

Update `MetricStatus` to include per-metric fallback information:

```golang
// ExternalMetricStatus indicates the current value of a global metric not associated
// with any Kubernetes object.
type ExternalMetricStatus struct {
	// metric identifies the target metric by name and selector
	Metric MetricIdentifier `json:"metric" protobuf:"bytes,1,name=metric"`

	// current contains the current value for the given metric
	Current MetricValueStatus `json:"current" protobuf:"bytes,2,name=current"`
    
  // fallbackActive indicates whether this metric is currently using a fallback value
  // due to retrieval failures.
  // +optional
  FallbackActive bool `json:"fallbackActive,omitempty"`
  
  // firstFailureTime is the timestamp of the first consecutive failure retrieving this metric.
  // Reset to nil on successful retrieval. Used to calculate if failureDuration has been exceeded.
  // +optional
  FirstFailureTime *metav1.Time `json:"firstFailureTime,omitempty"`

  // fallbackReplicas is the replica count being used while fallback is active.
  // Only populated when fallbackActive is true.
  // +optional
  FallbackReplicas *int32 `json:"fallbackReplicas,omitempty"`
}
```

Add a new `HorizontalPodAutoscalerConditionType`:

```golang
const (
    // ExternalMetricFallbackActive indicates that one or more external metrics
    // are currently using fallback values due to retrieval failures.
    // Status will be:
    // - "True" if any external metric is in fallback state
    // - "False" if no external metrics are in fallback state
    // - "Unknown" if the controller cannot determine the state
    ExternalMetricFallbackActive ConditionType = "ExternalMetricFallbackActive"
)
```

### Test Plan

<!--
**Note:** *Not required until targeted at a release.*
The goal is to ensure that we don't accept enhancements with inadequate testing.

All code is expected to have adequate tests (eventually with coverage
expectations). Please adhere to the [Kubernetes testing guidelines][testing-guidelines]
when drafting this test plan.

[testing-guidelines]: https://git.k8s.io/community/contributors/devel/sig-testing/testing.md
-->

[x] I/we understand the owners of the involved components may require updates to
existing tests to make this code solid enough prior to committing the changes necessary
to implement this enhancement.

##### Prerequisite testing updates

<!--
Based on reviewers feedback describe what additional tests need to be added prior
implementing this enhancement to ensure the enhancements have also solid foundations.
-->

None required.

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

- Tests for Fallback Configuration:
  - Verify failureDuration validation (must be > 0)
  - Verify replicas validation (must be > 0)
- Tests for Failure Tracking and Activation:
  - Verify `firstFailureTime` is set on first failure and persists through consecutive failures
  - Verify `firstFailureTime` is cleared on successful metric retrieval
  - Verify fallback activates when current time exceeds `firstFailureTime` + `failureDuration`
  - Verify fallbackActive status field updates correctly
- Tests for Replica Calculation:
  - Verify fallback returns the configured replica count when threshold is exceeded
  - Verify fallback replica count is combined with other metrics using max() (standard multi-metric behavior)
  - Verify replica calculations respect min/max constraints with fallback replica counts
  - Verify correct behavior with multiple external metrics (independent failure tracking and max selection)
  
- `/pkg/controller/podautoscaler`: 05 Nov 2025 - 89.1%
- `/pkg/controller/podautoscaler/metrics`: 05 Nov 2025 - 89.9%

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

N/A, the feature is tested using unit tests and e2e tests.

##### e2e tests

<!--
This question should be filled when targeting a release.
For Alpha, describe what tests will be added to ensure proper quality of the enhancement.

For Beta and GA, add links to added tests together with links to k8s-triage for those tests:
https://storage.googleapis.com/k8s-triage/index.html

We expect no non-infra related flakes in the last month as a GA graduation criteria.
-->

We will add the following e2e autoscaling tests:

- External metric failure triggers fallback after threshold is reached, using configured replica count
- HPA status condition `ExternalMetricFallbackActive` is set to True when fallback activates
- Success in retrieving external metric resets the failure count and resumes normal scaling
- HPA uses max() of healthy metric calculations and fallback replica counts
- Fallback respects HPA min/max replica constraints
- Status correctly reflects which metrics are in fallback state and shows `firstFailureTime`
- With multiple external metrics in fallback, HPA uses the maximum fallback replica count

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

- Feature implemented behind `HPAExternalMetricFallback` feature gate
- Unit and e2e tests passed as designed in [TestPlan](#test-plan).

#### Beta

- Unit and e2e tests passed as designed in [TestPlan](#test-plan).
- Gather feedback from developers and surveys
- All functionality completed
- All security enforcement completed
- All monitoring requirements completed
- All testing requirements completed
- All known pre-release issues and gaps resolved

#### GA

- No negative feedback.
- All issues and gaps identified as feedback during beta are resolved

### Upgrade / Downgrade Strategy

#### Upgrade

When the feature gate is enabled:
- Existing HPAs continue to work unchanged
- External metrics without `fallback` configuration behave as they do today (no scaling when unavailable)
- Users can add `fallback` configuration to external metrics in their HPAs
- The controller begins tracking per-metric `firstFailureTime` for external metrics with fallback configured
  - On the first failure, `firstFailureTime` is set to the current timestamp
  - On subsequent failures, the timestamp is preserved to track failure duration
  - On success, `firstFailureTime` is cleared (set to nil)
- The `fallbackActive`, `firstFailureTime`, and `fallbackReplicas` status fields are populated for external metrics with fallback configured
- Fallback activates when `(current time - firstFailureTime) >= failureDuration`

#### Downgrade

When the feature gate is disabled:
- The `fallback` field in `ExternalMetricSource` is ignored by the controller
- The `fallbackActive`, `firstFailureTime`, and `fallbackReplicas` status fields are not updated (remain at last values but are not used)
- All external metrics revert to current behavior: HPA cannot scale based on them when they're unavailable
- Any HPAs currently using fallback values will:
  - Maintain their current replica count
  - Stop using fallback values
  - Resume normal metric-based scaling when external metrics become available again
- No disruption to running workloads (pods are not restarted)
- The `firstFailureTime` timestamp remains in the status but is not evaluated or updated

All logic related to fallback evaluation, failure counting, and status updates is gated by the `HPAExternalMetricFallback` feature gate.

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

1. `kube-apiserver`:  More recent instances will accept and validate the new `fallback` field in `ExternalMetricSource`, While older instances will ignore it during validation and persist it as part of the HPA object.
2. `kube-controller-manager`: An older version could receive an HPA containing the new `fallback` field from a more recent API server, in which case it would ignore the field (i.e., continue with current behavior where external metrics that fail to retrieve prevent scaling)


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
  - Feature gate name: HPAExternalMetricFallback
  - Components depending on the feature gate: `kube-controller-manager` and `kube-apiserver`

###### Does enabling the feature change any default behavior?

No. By default, HPAs will continue to behave as they do today. The feature only activates when users explicitly configure the `fallback` field on external metrics in their HPA specifications. 
External metrics without fallback configuration will continue to prevent scaling when unavailable, which is the current behavior.
When fallback is configured and activated, the failing metric contributes its configured replica count to the HPA's decision, which is then combined with other metrics using the standard max() approach.

###### Can the feature be disabled once it has been enabled (i.e. can we roll back the enablement)?

Yes. If the feature gate is disabled:
- All `fallback` configurations in HPA specs are ignored by the controller
- External metrics revert to current behavior: HPA cannot scale based on them when they're unavailable
- The `fallbackActive`, `firstFailureTime`, and `fallbackReplicas` status fields stop being updated
  - These fields remain in the HPA status at their last values but are not evaluated or modified
- HPAs maintain their current replica count at the time of rollback
- No pods are restarted or disrupted

To disable, restart `kube-controller-manager` and `kube-apiserver` with the feature gate set to `false`.

###### What happens if we reenable the feature if it was previously rolled back?

When the feature is re-enabled:
- Any HPAs with `fallback` configured on external metrics will resume fallback behavior
- The controller clears any stale `firstFailureTime` timestamps and starts fresh
- If external metrics are failing at re-enablement:
  - On the first failure, `firstFailureTime` is set to the current timestamp
  - The failure duration is calculated as `(current time - firstFailureTime)`
  - Once the configured `failureDuration` has elapsed, fallback values are used
  - The `fallbackActive` status field is set to `true` for affected metrics
- HPAs resume using the static replicas stanza for scaling decisions when external metrics are unavailable and thresholds are exceeded

Existing HPAs without `fallback` configuration are not affected by re-enabling the feature and continue with default behavior.

###### Are there any tests for feature enablement/disablement?

Yes. Unit tests will verify that HPAs with and without the `fallback` field are properly validated both when the feature gate is enabled or disabled, and that the HPA controller correctly applies fallback behavior based on the feature gate status.


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
Rollout failures are unlikely to impact running workloads. If enabled during external metrics failures, HPAs with fallback configured might change scaling decisions after `failureDuration` (default: 3m) has elapsed. This is mitigated by:
- The HPA's min/max replica constraints
- The `failureDuration` buffer before activation
- Gradual HPA scaling behavior
- Scale-up/scale-down stabilization windows

On rollback, HPAs maintain their current replica count and stop using fallback values. No pods are restarted.

###### What specific metrics should inform a rollback?

<!--
What signals should users be paying attention to when the feature is young
that might indicate a serious problem?
-->
- Unexpected scaling events after enabling the feature
- Increased error rate in horizontal_pod_autoscaler_controller_metric_computation_total
- High percentage of HPAs showing fallbackActive: true unexpectedly
- Increased latency in horizontal_pod_autoscaler_controller_reconciliation_duration_seconds

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
No. This feature only adds a new optional field to the HPA API and doesn't deprecate or remove any existing functionality. All current HPA behaviors remain unchanged unless users explicitly opt into the fallback mode.

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
The presence of the `fallback` field in `ExternalMetricSource` specifications indicates that the feature is in use.

###### How can someone using this feature know that it is working for their instance?

<!--
For instance, if this is a pod-related feature, it should be possible to determine if the feature is functioning properly
for each individual pod.
Pick one more of these and delete the rest.
Please describe all items visible to end users below with sufficient detail so that they can verify correct enablement
and operation of this feature.
Recall that end users cannot usually observe component logs or access metrics.
-->
Users can confirm that the feature is active and functioning by inspecting the status fields exposed by the controller. Specifically:
- Check the HPA condition to verify if `ExternalMetricFallbackActive` is currently active
- Check `.status.currentMetrics[].external.fallbackActive` to verify if fallback is currently active
- Check `.status.currentMetrics[].external.firstFailureTime` to see when failures started

Moreover, users can verify the feature is working properly through events on the HPA object:
- When fallback activates: Normal `ExternalMetricFallbackActivated` "Fallback activated for external metric 'queue_depth' after 3m0s of consecutive failures, using fallback replica count: 10"

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
This feature utilizes the existing HPA controller metrics:
- `horizontal_pod_autoscaler_controller_reconciliation_duration_seconds`
- `horizontal_pod_autoscaler_controller_metric_computation_duration_seconds`
- `horizontal_pod_autoscaler_controller_metric_computation_total`

###### What are the SLIs (Service Level Indicators) an operator can use to determine the health of the service?

<!--
Pick one more of these and delete the rest.
-->
This feature doesn't fundamentally change how the HPA controller operates; it adds fallback handling when external metrics fail to be retrieved. Therefore, existing metrics for monitoring HPA controller health remain applicable:
- `horizontal_pod_autoscaler_controller_reconciliation_duration_seconds` - monitors overall HPA reconciliation performance
- `horizontal_pod_autoscaler_controller_metric_computation_duration_seconds` - tracks metric computation time including fallback evaluation
- `horizontal_pod_autoscaler_controller_metric_computation_total` - counts metric computations with error status

###### Are there any missing metrics that would be useful to have to improve observability of this feature?

<!--
Describe the metrics themselves and the reasons why they weren't added (e.g., cost,
implementation difficulties, etc.).
-->
No.

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
No. The feature only adds logic to the existing HPA reconciliation loop. It doesn't introduce new API calls.
The feature tracks failure counts and applies fallback logic in-memory during existing reconciliation cycles.

###### Will enabling / using this feature result in introducing new API types?

<!--
Describe them, providing:
  - API type
  - Supported number of objects per cluster
  - Supported number of objects per namespace (for namespace-scoped objects)
-->
No. The feature only adds new fields to existing API types:
- New `ExternalMetricFallback` struct within `ExternalMetricSource`
- New status fields in `ExternalMetricStatus`

###### Will enabling / using this feature result in any new calls to the cloud provider?

<!--
Describe them, providing:
  - Which API(s):
  - Estimated increase:
-->
No.

###### Will enabling / using this feature result in increasing size or count of the existing API objects?

<!--
Describe them, providing:
  - API type(s):
  - Estimated increase in size: (e.g., new annotation of size 32B)
  - Estimated amount of new objects: (e.g., new Object X for every existing Pod)
-->
Yes, `HorizontalPodAutoscaler` objects will increase in size when fallback is configured:
- Spec increase: ~150 bytes per external metric with fallback configured:
  - `failureDuration`: ~40 bytes (field name + duration string like "3m")
  - `replicas`: ~20 bytes (int32)
- Status increase: ~80 bytes per external metric:
  - `fallbackActive`: ~30 bytes (boolean field)
  - `firstFailureTime`: ~70 bytes (timestamp field + RFC3339 string like "2024-01-15T10:23:45Z")
  - `fallbackReplicas`: ~30 bytes (optional int32 pointer)

###### Will enabling / using this feature result in increasing time taken by any operations covered by existing SLIs/SLOs?

<!--
Look at the [existing SLIs/SLOs].

Think about adding additional work or introducing new steps in between
(e.g. need to do X to start a container), etc. Please describe the details.

[existing SLIs/SLOs]: https://git.k8s.io/community/sig-scalability/slos/slos.md#kubernetes-slisslos
-->

No. The feature adds minimal computational overhead to the existing HPA reconciliation loop. The fallback logic is integrated into the existing metric retrieval and evaluation process:
1. Attempt to retrieve external metric (already happens)
2. On failure: check/update `firstFailureTime` (new, minimal overhead)
3. Evaluate if fallback should activate (new, simple comparison)
4. Return either real metric or fallback replica count (already happens for other metric types)

###### Will enabling / using this feature result in non-negligible increase of resource usage (CPU, RAM, disk, IO, ...) in any components?

<!--
Things to keep in mind include: additional in-memory state, additional
non-trivial computations, excessive access to disks (including increased log
volume), significant amount of data sent and/or received over network, etc.
This through this both in small and large cases, again with respect to the
[supported limits].

[supported limits]: https://git.k8s.io/community//sig-scalability/configs-and-limits/thresholds.md
-->
No. Memory increase in kube-controller-manager is ~100 bytes per HPA for failure count tracking. For 1000 HPAs with 2 external metrics each: ~200 KB total, which is negligible.

###### Can enabling / using this feature result in resource exhaustion of some node resources (PIDs, sockets, inodes, etc.)?

<!--
Focus not just on happy cases, but primarily on more pathological cases
(e.g. probes taking a minute instead of milliseconds, failed pods consuming resources, etc.).
If any of the resources can be exhausted, how this is mitigated with the existing limits
(e.g. pods per node) or new limits added by this KEP?

Are there any tests that were run/should be run to understand performance characteristics better
and validate the declared limits?
-->
No.

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

If the API server and/or etcd becomes unavailable, the entire HPA controller functionality will be impacted, not just this feature. The HPA controller will not be able to:
- Retrieve HPA objects
- Get external metrics (or any metrics)
- Update HPA status (including `fallbackActive`, `firstFailureTime`, and `fallbackReplicas` fields)
- Apply scaling decisions

Therefore, no autoscaling decisions can be made during this period, regardless of whether fallback is configured. The feature itself doesn't introduce any new failure modes with respect to API server or etcd availability - it's dependent on these components being available just like the rest of the HPA controller's functionality.

Once API server and etcd access is restored, the HPA controller will resume normal operation. The in-memory failure counts will reset, if external metrics are still failing and `firstFailureTime` is perseved the controller will use that timestamp to calculate 

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

Check `horizontal_pod_autoscaler_controller_reconciliation_duration_seconds` to identify if issues correlate with HPAs using fallback. If problems are observed:

- Check if the issue only affects HPAs with fallback configured
- Review HPA events: kubectl describe hpa <name> to see fallback activation events
- Check external metrics provider health and connectivity

For problematic HPAs, you can:

- Temporarily remove the fallback field to revert to default behavior (HPA holds current scale on metric failure)
- Adjust `failureThreshold` to prevent premature fallback activation
- Review and adjust fallback values if scaling behavior is inappropriate

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
