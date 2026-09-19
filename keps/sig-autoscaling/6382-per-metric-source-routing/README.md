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
# KEP-6382: Per-Metric Source Routing for HPA

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
    - [Story 1: Multi-Provider Metrics on a Single HPA](#story-1-multi-provider-metrics-on-a-single-hpa)
    - [Story 2: Model-Serving Metrics Alongside Existing Custom Metrics Provider](#story-2-model-serving-metrics-alongside-existing-custom-metrics-provider)
    - [Story 3: Managed Cluster with Vendor-Specific Adapter](#story-3-managed-cluster-with-vendor-specific-adapter)
  - [Risks and Mitigations](#risks-and-mitigations)
- [Design Details](#design-details)
  - [API Changes](#api-changes)
    - [User-Facing Example](#user-facing-example)
  - [Wire Format Constraint](#wire-format-constraint)
    - [Version Constraints](#version-constraints)
  - [MetricsClient Interface Changes](#metricsclient-interface-changes)
  - [Client Implementation](#client-implementation)
  - [RBAC](#rbac)
  - [Status Reporting](#status-reporting)
  - [Provider Contract](#provider-contract)
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
- [x] (R) KEP approvers have approved the KEP status as `implementable`
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

[kubernetes.io]: https://kubernetes.io/
[kubernetes/enhancements]: https://git.k8s.io/enhancements
[kubernetes/kubernetes]: https://git.k8s.io/kubernetes
[kubernetes/website]: https://git.k8s.io/website

## Summary

Today, the Kubernetes API server allows only one metrics provider per metrics
API group. Only one `APIService` can register for `custom.metrics.k8s.io` at a
time, and only one for `external.metrics.k8s.io`, which means the entire
cluster is locked to a single custom metrics provider and a single external
metrics provider. Two different HPAs that need metrics from two different
providers (e.g. one using KEDA and another using a vendor-specific adapter)
cannot coexist — nor can a single HPA mix metrics from multiple providers.
This creates vendor lock-in, forcing all custom and external metrics through
their respective single provider regardless of where the metrics actually
originate.

Kubernetes already has the `APIService` object, which allows registering
aggregated API servers on the fly. This KEP leverages that existing mechanism
by adding an optional `source` field to `MetricIdentifier`, which allows each
HPA metric to specify a `source.apiGroup` pointing to a different registered
`APIService`. This lets the HPA controller fetch metrics from multiple
providers within a single HPA - for Pods, Object, and External metric types.

## Motivation

As described in the summary, only one `APIService` can register per metrics
API group. This means the entire cluster is locked to a single provider for
custom metrics (`custom.metrics.k8s.io`) and a single provider for external
metrics (`external.metrics.k8s.io`). Teams that need metrics from multiple
sources - for example, KEDA for queue-based scaling and a model-serving
adapter for inference metrics - must either funnel everything through one
provider or give up on one of them entirely.

### Goals

- Break the vendor lock-in that prevents clusters from using multiple
  external or custom metrics providers simultaneously.
- Leverage the existing `APIService` mechanism so that each HPA metric
  can be routed to a different registered metrics provider.
- Maintain full backward compatibility: HPAs without a `source` field
  behave exactly as today.

### Non-Goals

- Changing the API server aggregation layer.
- Requiring changes to existing metrics providers that don't want to
  participate.
- Supporting arbitrary wire formats per source (providers must implement
  the standard metrics API response types).

## Proposal

Add an optional `source` field to `MetricIdentifier`. The
`source.apiGroup` tells the HPA controller which registered `APIService`
to call for that specific metric, allowing each metric in an HPA to be
fetched from a different provider. When `source` is omitted, the metric
is fetched from the default provider for its type - preserving existing
behavior.

### User Stories

#### Story 1: Multi-Provider Metrics on a Single HPA

As an operator, I run a workload that needs to scale based on metrics from
two different external metrics providers - for example, GPU queue depth from
one adapter and request rate from another. Today, I cannot combine both in a
single HPA because only one provider can register for
`external.metrics.k8s.io`. With source routing, I set `source.apiGroup` on
each metric to point to its respective provider's `APIService`, and the HPA
fetches each metric from the correct source.

#### Story 2: Model-Serving Metrics Alongside Existing Custom Metrics Provider

As a platform engineer running an AI inference workload, I need to scale
based on model-specific metrics (e.g. request queue depth, batch size) served
by a model-serving metrics adapter, while the cluster already has an existing
custom metrics provider registered for `custom.metrics.k8s.io`. Today, only
one provider can own that API group, so I must choose between my
model-serving adapter and the existing provider - I cannot use both. With
source routing, I configure `type: Pods` metrics with
`source.apiGroup: modelserver.example.com` to fetch from the model-serving
adapter, while the existing provider continues to serve metrics through the
default `custom.metrics.k8s.io` API group.

#### Story 3: Managed Cluster with Vendor-Specific Adapter

As an operator on a managed Kubernetes cluster (GKE, EKS, AKS), the cloud
provider has already registered their own adapter for
`custom.metrics.k8s.io`. I want to also scale based on application-specific
metrics served by my own custom metrics adapter, but I cannot replace the
cloud provider's registration. With source routing, I deploy my adapter under
its own API group and set `source.apiGroup` on the relevant HPA metrics,
leaving the cloud provider's default registration untouched.

### Risks and Mitigations

A provider serving metrics under a custom API group may not implement the
correct wire format. The controller returns a clear error in HPA status
conditions with the API group name, and the provider contract documents the
exact format requirements.

RBAC may be missing for a new API group. The HPA status condition surfaces the
specific API group and the 403 error, making it straightforward to diagnose.

Discovery calls for many distinct API groups could add overhead. The factory
lazily discovers and caches clients per group, following the same pattern as
existing custom metrics discovery.

The client cache could grow unbounded if HPAs reference many arbitrary API
groups. The cache is bounded to 20 entries with LRU eviction and a warning
logged when eviction occurs.

Disabling the feature gate after it has been enabled silently changes metric
routing — HPAs fall back to the default provider, which may return different
values or not have the metric at all. The HPA status condition surfaces the
error, and this is documented as a known downgrade risk.

## Design Details

### API Changes

`MetricIdentifier` gains a `Source` field:

```go
type MetricIdentifier struct {
    Name     string                `json:"name" protobuf:"bytes,1,name=name"`
    Selector *metav1.LabelSelector `json:"selector,omitempty" protobuf:"bytes,2,opt,name=selector"`
    // +featureGate=HPAMetricSource
    // +optional
    Source   *MetricSourceReference `json:"source,omitempty" protobuf:"bytes,3,opt,name=source"`
}

// MetricSourceReference identifies the API group of a metrics provider.
// The provider must serve the same wire format as the default provider
// for the metric type (MetricValueList for Pods/Object, ExternalMetricValueList
// for External).
type MetricSourceReference struct {
    APIGroup string `json:"apiGroup" protobuf:"bytes,1,name=apiGroup"`
}
```

The `source` field changes where the metric is fetched from. The
`type` field continues to determine how the metric is fetched and
calculated:

| type     | Wire format provider must serve | REST path pattern                          | HPA calculation     |
|----------|--------------------------------|--------------------------------------------|---------------------|
| Pods     | `MetricValueList`              | `.../namespaces/{ns}/pods/*/{metricName}`  | Average per pod     |
| Object   | `MetricValueList`              | `.../namespaces/{ns}/{resource}/{name}/{metricName}` | Single object value |
| External | `ExternalMetricValueList`      | `.../namespaces/{ns}/{metricName}`         | Raw sum of values   |

#### User-Facing Example

```yaml
apiVersion: autoscaling/v2
kind: HorizontalPodAutoscaler
metadata:
  name: inference-server
spec:
  scaleTargetRef:
    apiVersion: apps/v1
    kind: Deployment
    name: inference-server
  minReplicas: 2
  maxReplicas: 50
  metrics:
  # Metric from a model-serving metrics adapter (not KEDA)
  - type: Pods
    pods:
      metric:
        name: gen_ai_server_request_queued
        source:
          apiGroup: modelserver.example.com
      target:
        type: AverageValue
        averageValue: "5"
  # Metric from KEDA (default external.metrics.k8s.io, no source needed)
  - type: External
    external:
      metric:
        name: datadog_request_rate
      target:
        type: Value
        value: "100"
  # Metric from a vendor-specific adapter
  - type: Object
    object:
      metric:
        name: active_connections
        source:
          apiGroup: monitoring.example.com
      describedObject:
        apiVersion: networking.k8s.io/v1
        kind: Ingress
        name: main-ingress
      target:
        type: Value
        value: "1000"
```

### Wire Format Constraint

The `type` field determines the wire format:

| type     | Expected response type       | Default API group              |
|----------|------------------------------|--------------------------------|
| Pods     | `MetricValueList`            | `custom.metrics.k8s.io`       |
| Object   | `MetricValueList`            | `custom.metrics.k8s.io`       |
| External | `ExternalMetricValueList`    | `external.metrics.k8s.io`     |

When `source.apiGroup` is set, the controller calls that API group
instead of the default, but expects the **same response type**. The
provider at `modelserver.example.com` must serve `MetricValueList`
responses if the HPA uses `type: Pods` or `type: Object`.

#### Version Constraints

This KEP does not change the existing version or decoding behavior of
the metrics clients. The constraints below are pre-existing — they
apply identically to `custom.metrics.k8s.io` and
`external.metrics.k8s.io` today and are inherited unchanged by
source-routed API groups.

**Custom metrics (Pods / Object):** Both the URL version and the
response body `apiVersion` must be `v1beta1` or `v1beta2`. The URL
version is discovered via `GET /apis`, but only versions known to the
`custom_metrics` scheme (`custom.metrics.k8s.io/v1beta1` and
`custom.metrics.k8s.io/v1beta2`) are accepted — unknown versions are
rejected at discovery time. The response body must use
`custom.metrics.k8s.io` as the group in its `apiVersion` field,
because the client-side decoder only recognizes that group. This
applies identically to the default group and source-routed groups.

**External metrics:** Both the URL version and the response body
`apiVersion` must be `v1beta1`. The external metrics client hardcodes
this version — there is no discovery. This is how
`external.metrics.k8s.io` works today; source-routed API groups
inherit the same constraint.

### MetricsClient Interface Changes

Add an `apiGroup string` parameter to the three existing methods that
support source routing. Empty string means "use default" (current
behavior).

```go
type MetricsClient interface {
    GetResourceMetric(ctx context.Context, resource v1.ResourceName,
        namespace string, selector labels.Selector,
        container string) (PodMetricsInfo, time.Time, error)

    GetRawMetric(metricName string, namespace string,
        selector labels.Selector, metricSelector labels.Selector,
        apiGroup string) (PodMetricsInfo, time.Time, error)

    GetObjectMetric(metricName string, namespace string,
        objectRef *autoscaling.CrossVersionObjectReference,
        metricSelector labels.Selector,
        apiGroup string) (int64, time.Time, error)

    GetExternalMetric(metricName string, namespace string,
        selector labels.Selector,
        apiGroup string) ([]int64, time.Time, error)
}
```

Note: `GetResourceMetric` is unchanged — resource metrics always come from
`metrics.k8s.io`.

### Client Implementation

`restMetricsClient` gains a client factory:

```go
type restMetricsClient struct {
    *resourceMetricsClient
    *customMetricsClient
    *externalMetricsClient
    clientFactory MetricsClientFactory // NEW
}

type MetricsClientFactory interface {
    CustomClientForGroup(apiGroup string) (customclient.CustomMetricsClient, error)
    ExternalClientForGroup(apiGroup string) (externalclient.ExternalMetricsClient, error)
}
```

When `apiGroup` is empty, delegate to the embedded default client (existing
behavior). When set, call `clientFactory.CustomClientForGroup(apiGroup)` or
`ExternalClientForGroup(apiGroup)` to get a client for that API group, then
make the same REST call.

The factory lazily constructs and caches clients keyed by API group.
Creating a REST client involves a discovery call to resolve the preferred
API version and setting up HTTP transport - doing that on every HPA
reconcile cycle (default 15 seconds) for every source-routed metric would
be wasteful. The cache stores already-constructed clients so subsequent
reconciles reuse them. The cache is bounded to 20 entries with LRU eviction
to prevent unbounded memory growth (e.g., if a user creates HPAs referencing
many arbitrary API groups).

### RBAC

No changes to Kubernetes bootstrap RBAC. The existing HPA controller
ClusterRole is untouched.

Providers grant the HPA controller access to their API group via their
own install manifests (Helm chart, operator, etc.):

```yaml
apiVersion: rbac.authorization.k8s.io/v1
kind: ClusterRole
metadata:
  name: hpa-reader:modelserver-metrics
rules:
- apiGroups: ["modelserver.example.com"]
  resources: ["*"]
  verbs: ["get", "list"]
---
apiVersion: rbac.authorization.k8s.io/v1
kind: ClusterRoleBinding
metadata:
  name: hpa-reader:modelserver-metrics
roleRef:
  apiGroup: rbac.authorization.k8s.io
  kind: ClusterRole
  name: hpa-reader:modelserver-metrics
subjects:
- kind: ServiceAccount
  name: horizontal-pod-autoscaler
  namespace: kube-system
```

### Status Reporting

The HPA status surfaces which API group was used for each metric,
to aid debugging:

```yaml
status:
  currentMetrics:
  - type: Pods
    pods:
      metric:
        name: gen_ai_server_request_queued
        source:
          apiGroup: modelserver.example.com
      current:
        averageValue: "3"
```

### Provider Contract

A provider serving metrics under a custom API group MUST satisfy
the following requirements. This is the contract that `source.apiGroup`
implies.

Registration:
- Register an `APIService` with the API server for its API group and version.
- The APIService must point to a Service backed by the provider's pods.
- Grant the HPA controller ServiceAccount
  (`system:serviceaccount:kube-system:horizontal-pod-autoscaler`) read
  access to its API group via RBAC (ClusterRole + ClusterRoleBinding).

Discovery:
- Respond to `GET /apis/{apiGroup}` with an `APIGroup` resource listing
  at least one version.
- Respond to `GET /apis/{apiGroup}/{version}` with an `APIResourceList`.

Wire Format and Versioning:

The provider must serve responses using the same wire format and
versions that existing metrics providers use today. This KEP does not
introduce new response types or versions — it only changes which API
group the controller calls.

| HPA `type` | URL version | Response body `apiVersion` | Response body `kind` |
|------------|-------------|---------------------------|----------------------|
| Pods       | `v1beta1` or `v1beta2` | `custom.metrics.k8s.io/v1beta1` or `v1beta2` | `MetricValueList` |
| Object     | `v1beta1` or `v1beta2` | `custom.metrics.k8s.io/v1beta1` or `v1beta2` | `MetricValueList` |
| External   | `v1beta1` | `external.metrics.k8s.io/v1beta1` | `ExternalMetricValueList` |

For custom metrics (Pods/Object), the provider must register a URL
version of `v1beta1` or `v1beta2` for its API group. The controller
discovers the preferred version via `GET /apis/{apiGroup}` and only
accepts versions known to the `custom_metrics` scheme. The response
body must use `custom.metrics.k8s.io` as the group in its `apiVersion`
field, because the client-side decoder only recognizes that group.

For external metrics, both the URL version and response body must use
`v1beta1`. This is the same constraint as `external.metrics.k8s.io`
today - there has only ever been one version of the external metrics
protocol.

REST Path Structure:

The provider must serve the same REST path patterns as the default
provider for the metric type:

External metrics:
```
GET /apis/{apiGroup}/v1beta1/namespaces/{namespace}/{metricName}
    ?labelSelector={selector}
```

Pods metrics (custom):
```
GET /apis/{apiGroup}/{v1beta1|v1beta2}/namespaces/{namespace}/pods/*/{metricName}
    ?labelSelector={podSelector}&metricLabelSelector={metricSelector}
```

Object metrics (custom):
```
GET /apis/{apiGroup}/{v1beta1|v1beta2}/namespaces/{namespace}/{resource}/{name}/{metricName}
    ?metricLabelSelector={metricSelector}
```

### Test Plan

[x] I/we understand the owners of the involved components may require updates to
existing tests to make this code solid enough prior to committing the changes necessary
to implement this enhancement.

##### Prerequisite testing updates

None required.

##### Unit tests

- Tests for Validation and Feature Gating:
  - Verify `source` field is accepted/rejected based on feature gate state
  - Verify `source` field is dropped by strategy when feature gate is disabled
  - Verify `source.apiGroup` must be a valid DNS subdomain and not a default group
- Tests for MetricsClient Routing:
  - Verify empty `apiGroup` delegates to default client (existing behavior)
  - Verify non-empty `apiGroup` uses factory to get a client for that group
  - Verify client cache behavior and bounded eviction
- Tests for Controller Plumbing:
  - Verify `source` is extracted from metric spec and passed through to the metrics client
  - Verify HPA status reflects the actual API group used for each metric

- `pkg/apis/autoscaling/validation`: 2026-08-31 - 95.3% of statements
- `pkg/controller/podautoscaler`: 2026-08-31 - 89.0% of statements

##### Integration tests

Integration tests with a mock metrics server registered under a custom API
group to verify end-to-end routing:

- HPA fetches metrics from a non-default API group when `source.apiGroup` is set
- HPA fetches metrics from the default API group when `source` is not set
- HPA handles mixed metrics (some with source, some without) in a single spec
- HPA status reflects the actual API group used for each metric
- HPA surfaces clear error when the source API group is unreachable or RBAC is missing
- Downgrade: HPA falls back to default provider when `source` field is stripped

##### e2e tests

E2e tests with a real metrics adapter on a non-default API group will be
added as part of beta graduation criteria.

### Graduation Criteria

#### Alpha

- Feature implemented behind `HPAMetricSource` feature gate (default off)
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
- Metrics without `source` configuration behave as they do today
- Users can add `source` to `MetricIdentifier` in their HPA metrics
- The controller begins routing metrics with `source.apiGroup` set to the specified API group
- Status reflects the actual API group used for each metric

#### Downgrade

When the feature gate is disabled:
- The `source` field ignored by the controller field and fetches all metrics from the default provider for their type (`custom.metrics.k8s.io` for Pods/Object, `external.metrics.k8s.io` for External)
- Any HPAs that were using source routing will:
  - Maintain their current replica count
  - Stop routing metrics to non-default API groups
  - Fetch all metrics from the default provider instead
  - Surface errors in HPA status conditions if the default provider cannot serve the metric
- No disruption to running workloads

All logic related to source routing, client factory usage, and status reporting of the API group is gated by the `HPAMetricSource` feature gate.

### Version Skew Strategy

1. `kube-apiserver`: More recent instances will accept and validate the new `source` field in `MetricIdentifier`. Older instances will ignore it during validation and persist it as part of the HPA object.
2. `kube-controller-manager`: An older version could receive an HPA containing the new `source` field from a more recent API server, in which case it would ignore the field (i.e., continue with current behavior where all metrics are fetched from the default provider for their type).

## Production Readiness Review Questionnaire

### Feature Enablement and Rollback

###### How can this feature be enabled / disabled in a live cluster?

- [x] Feature gate (also fill in values in `kep.yaml`)
  - Feature gate name: HPAMetricSource
  - Components depending on the feature gate: `kube-controller-manager` and `kube-apiserver`

###### Does enabling the feature change any default behavior?

No. By default, HPAs will continue to behave as they do today. The feature only
activates when users explicitly set the `source` field on metrics in their HPA
specifications. Metrics without `source` continue to be fetched from the default
provider for their type.

###### Can the feature be disabled once it has been enabled (i.e. can we roll back the enablement)?

Yes. If the feature gate is disabled:
- The `source` field in HPA specs is ignored by the controller
- All metrics revert to being fetched from the default provider for their type
- The `source` field remains in the HPA spec at its last value but is not evaluated or used for routing
- No pods are restarted or disrupted

To disable, restart `kube-controller-manager` and `kube-apiserver` with the feature gate set to `false`.

###### What happens if we reenable the feature if it was previously rolled back?

When the feature is re-enabled:
- Any HPAs with `source` configured on metrics will resume source routing
- The controller creates new clients for the referenced API groups on demand
- Metrics are fetched from the specified API groups as configured

###### Are there any tests for feature enablement/disablement?

Yes. Unit tests will verify:
- `source` field is accepted when feature gate is enabled
- `source` field is rejected when feature gate is disabled
- `source` field is dropped by strategy when feature gate is disabled
- HPAs with `source` fall back to default provider when gate is disabled

### Rollout, Upgrade and Rollback Planning

<!--
This section must be completed when targeting beta to a release.
-->

###### How can a rollout or rollback fail? Can it impact already running workloads?

A rollback (disabling the feature gate) causes the `source` field to be stripped.
HPAs that were fetching metrics from non-default providers will silently switch to
the default provider. If the default provider doesn't serve the metric, the HPA
will stop scaling based on that metric and surface an error in status conditions.
Running workloads are not disrupted — pods are not restarted.

###### What specific metrics should inform a rollback?

- Increased error rate in HPA status conditions
- Increased HPA controller reconcile errors

###### Were upgrade and rollback tested? Was the upgrade->downgrade->upgrade path tested?

<!--
Describe manual testing that was done and the outcomes.
Longer term, we may want to require automated upgrade/rollback tests, but we
are missing a bunch of machinery and tooling and can't do that now.
-->

###### Is the rollout accompanied by any deprecations and/or removals of features, APIs, fields of API types, flags, etc.?

No.

### Monitoring Requirements

<!--
This section must be completed when targeting beta to a release.
-->

###### How can an operator determine if the feature is in use by workloads?

Check if any HPA objects have the `source` field set on their metrics:
```
kubectl get hpa -A -o json | jq '[.items[].spec.metrics[]? | select(.pods.metric.source != null or .object.metric.source != null or .external.metric.source != null)] | length'
```

###### How can someone using this feature know that it is working for their instance?

- [ ] API .status
  - Condition name: ScalingActive
  - Other field: `status.currentMetrics[].*.metric.source.apiGroup` shows the API group used
- [ ] Events
  - Event Reason: FailedGetMetric (includes API group in message when source routing fails)

###### What are the reasonable SLOs (Service Level Objectives) for the enhancement?

- HPA controller reconcile latency should not be affected. Source routing changes which endpoint a metric is fetched from, but the number of REST calls and wire format remain the same.

###### What are the SLIs (Service Level Indicators) an operator can use to determine the health of the service?

- [ ] Metrics
  - Metric name: `horizontal_pod_autoscaler_controller_metric_computation_duration_seconds`
  - Components exposing the metric: `kube-controller-manager`

###### Are there any missing metrics that would be useful to have to improve observability of this feature?

A metric tracking the number of source-routed metric fetches (by API group) and
their success/failure rate would be useful. TBD whether this should be added in
alpha or beta.

### Dependencies

###### Does this feature depend on any specific services running in the cluster?

- Metrics providers registered via APIService
  - Usage description: The feature routes metric fetches to API groups registered
    as APIServices. Providers must be deployed and serving.
  - Impact of its outage on the feature: Metrics from the unavailable provider
    cannot be fetched. HPA status surfaces the error. Other metrics (from other
    providers or default providers) continue to work.
  - Impact of its degraded performance or high-error rates on the feature:
    Increased latency in HPA reconcile loop for affected metrics.

### Scalability

###### Will enabling / using this feature result in any new API calls?

Yes. When `source.apiGroup` is set, the HPA controller makes discovery calls
(`GET /apis/{apiGroup}`) and metric fetch calls to the specified API group.
- API call type: GET on aggregated API endpoints
- Estimated throughput: One discovery call per unique API group (cached), one
  metric fetch per HPA reconcile cycle per source-routed metric
- Originating component: `kube-controller-manager` (HPA controller)

###### Will enabling / using this feature result in introducing new API types?

No. The feature only adds new fields to existing API types:
- New `MetricSourceReference` struct within `MetricIdentifier`
- New `source` field in `MetricIdentifier` (used in both spec and status)

###### Will enabling / using this feature result in any new calls to the cloud provider?

No direct calls. However, if a metrics provider running in-cluster makes calls
to a cloud provider to serve metrics, those calls are unchanged by this feature.

###### Will enabling / using this feature result in increasing size or count of the existing API objects?

- API type: HorizontalPodAutoscaler
- Estimated increase in size: ~50 bytes per metric with `source` set (one
  `MetricSourceReference` struct with an `apiGroup` string)

###### Will enabling / using this feature result in increasing time taken by any operations covered by existing SLIs/SLOs?

HPA reconcile may take slightly longer when source-routed metrics are configured,
due to discovery and client creation for non-default API groups. Discovery is
cached; client creation is lazy and cached. The incremental cost per reconcile
is one additional REST call per source-routed metric (same as the existing call,
just to a different endpoint).

###### Will enabling / using this feature result in non-negligible increase of resource usage (CPU, RAM, disk, IO, ...) in any components?

Minimal increase in `kube-controller-manager` memory for the client cache
(bounded to 20 entries). Each cached client is a standard REST client with
a discovery cache — similar to the existing custom metrics client.

###### Can enabling / using this feature result in resource exhaustion of some node resources (PIDs, sockets, inodes, etc.)?

No.

### Troubleshooting

###### How does this feature react if the API server and/or etcd is unavailable?

Same as existing HPA behavior — the controller cannot reconcile HPAs and
retries on the next sync cycle.

###### What are other known failure modes?

- Source API group not registered
  - Detection: HPA status condition `ScalingActive=False` with message
    "no metrics API group X registered"
  - Mitigations: Verify APIService is registered (`kubectl get apiservices`)
  - Diagnostics: HPA controller logs at V(4) include API group in error messages
  - Testing: Unit tests verify error surfacing for unregistered API groups

- RBAC missing for source API group
  - Detection: HPA status condition `ScalingActive=False` with 403 Forbidden
    message including the API group name
  - Mitigations: Add ClusterRole + ClusterRoleBinding granting HPA controller
    SA access to the API group
  - Diagnostics: API server audit logs show 403 for the HPA controller SA
  - Testing: Integration tests verify RBAC error surfacing

- Provider serves wrong wire format
  - Detection: HPA status condition `ScalingActive=False` with deserialization
    error message
  - Mitigations: Verify provider implements the correct wire format per the
    provider contract
  - Diagnostics: HPA controller logs include the expected vs received type
  - Testing: Unit tests verify error handling for wire format mismatches

###### What steps should be taken if SLOs are not being met to determine the problem?

Check `horizontal_pod_autoscaler_controller_metric_computation_duration_seconds` to identify if issues correlate with HPAs using source routing. If problems are observed:

- Check if the issue only affects HPAs with `source` configured
- Review HPA events: `kubectl describe hpa <name>` to see source routing errors
- Check the source API group's APIService health: `kubectl get apiservices`
- Verify RBAC grants the HPA controller SA access to the API group

For problematic HPAs, you can:

- Temporarily remove the `source` field to revert to the default provider for that metric type
- Verify the provider is reachable and serving the correct wire format
- Check HPA controller logs for reconcile errors related to the API group

## Implementation History

- 2026-09-17: KEP created (provisional)

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
