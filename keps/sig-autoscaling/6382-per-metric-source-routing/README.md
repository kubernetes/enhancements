# KEP-6382: Per-Metric Source Routing for HPA External Metrics

<!-- toc -->
- [Release Signoff Checklist](#release-signoff-checklist)
- [Summary](#summary)
- [Motivation](#motivation)
  - [Goals](#goals)
  - [Non-Goals](#non-goals)
- [Proposal](#proposal)
  - [User Stories](#user-stories)
    - [Story 1: Two External Providers on a Single HPA](#story-1-two-external-providers-on-a-single-hpa)
    - [Story 2: Migrating Between External Metrics Providers](#story-2-migrating-between-external-metrics-providers)
    - [Story 3: Managed Cluster with a Provider-Managed Add-on](#story-3-managed-cluster-with-a-provider-managed-add-on)
  - [Notes/Constraints/Caveats](#notesconstraintscaveats)
  - [Risks and Mitigations](#risks-and-mitigations)
- [Design Details](#design-details)
  - [API Changes](#api-changes)
    - [autoscaling/v1](#autoscalingv1)
    - [Validation](#validation)
    - [User-Facing Example](#user-facing-example)
  - [Wire Format Constraint](#wire-format-constraint)
  - [Controller Changes](#controller-changes)
  - [Fetching from a Non-Default Group](#fetching-from-a-non-default-group)
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

[kubernetes.io]: https://kubernetes.io/
[kubernetes/enhancements]: https://git.k8s.io/enhancements
[kubernetes/kubernetes]: https://git.k8s.io/kubernetes
[kubernetes/website]: https://git.k8s.io/website

## Summary

Today, the HPA controller fetches every `External` metric from a single,
hardcoded API group: `external.metrics.k8s.io`. Because the aggregation layer
allows only one `APIService` per group version, the whole cluster is locked to
a single external metrics provider. Two HPAs that need metrics from two
different providers (for example adapters from two different vendors) cannot
coexist, and a single HPA cannot mix metrics from two providers.

Kubernetes already has the `APIService` object, which allows registering
aggregated API servers under any API group. This KEP leverages that existing
mechanism by adding an optional `source` field to `MetricIdentifier`. For
`External` metrics, `source.apiGroup` tells the HPA controller which
registered `APIService` to call for that specific metric, so one HPA can fetch
external metrics from several providers.

This KEP covers `External` metrics only. Routing `Pods` and `Object` metrics
(the custom metrics API) requires changes to the custom metrics client library
in `k8s.io/metrics` and is left as future work, see [Non-Goals](#non-goals).

## Motivation

As described in the summary, only one `APIService` can register for the
external metrics API group. This means the entire cluster is locked to a
single provider for external metrics (`external.metrics.k8s.io`). Teams that
need external metrics from multiple sources - for example, a monitoring
vendor's adapter for queue-based scaling and a model-serving adapter for
inference metrics - must either funnel everything through one provider or
give up on one of them entirely.

### Goals

- Allow a cluster to use multiple external metrics providers simultaneously.
- Leverage the existing `APIService` mechanism so that each `External` metric
  in an HPA can be routed to a different registered metrics provider.
- Maintain full backward compatibility: HPAs without a `source` field behave
  exactly as today.
- Keep the provider contract identical to today's `external.metrics.k8s.io`
  contract, only the API group changes.

### Non-Goals

- Changing the API server aggregation layer, for example allowing multiple
  `APIService` objects per group version.
- Routing `Pods` and `Object` metrics (`custom.metrics.k8s.io`). The custom
  metrics client in `k8s.io/metrics` binds both request parameter conversion
  and response decoding to the `custom.metrics.k8s.io` group, so routing those
  types needs a library change and a decision on response decoding. The
  `source` field is placed on `MetricIdentifier` so that this can be added
  later without a new API shape, but in this KEP validation rejects `source`
  on `Pods` and `Object` metrics.
- Routing `Resource` and `ContainerResource` metrics. These always come from
  `metrics.k8s.io`.
- Requiring changes to existing metrics providers that don't want to
  participate.
- Supporting arbitrary wire formats per source. Providers must implement the
  standard external metrics API response type.

## Proposal

Add an optional `source` field to `MetricIdentifier`. For `External` metrics,
`source.apiGroup` tells the HPA controller which registered `APIService` to
call for that metric. When `source` is omitted, the metric is fetched from
`external.metrics.k8s.io`, preserving existing behavior.

### User Stories

#### Story 1: Two External Providers on a Single HPA

As an operator, I run a workload that needs to scale based on metrics from two
different external metrics providers, for example GPU queue depth from a
model-serving adapter and request rate from Datadog. Today I cannot combine
both in a single HPA because only one provider can register for
`external.metrics.k8s.io`. With source routing, I set `source.apiGroup` on the
metric served by the second provider, and the HPA fetches each metric from the
correct source.

#### Story 2: Migrating Between External Metrics Providers

As a platform engineer, my cluster uses one vendor's adapter for external
metrics and I want to move to another vendor's adapter. Today the two cannot
be registered for `external.metrics.k8s.io` at the same time, so the
migration is a single cutover: replacing the `APIService` switches every HPA
in the cluster at once, and any HPA whose metric is not yet served by the new
adapter stops scaling. With source routing, the new adapter is deployed under
its own API group next to the old one. I move HPAs one at a time by setting
`source.apiGroup`, verify each, and can roll back a single HPA if needed.
Once no HPA depends on the old adapter, I remove it.

#### Story 3: Managed Cluster with a Provider-Managed Add-on

As an operator on a managed Kubernetes cluster, a provider-managed add-on
(for example the AKS KEDA add-on) already owns the `external.metrics.k8s.io`
registration. If I install another external metrics provider it takes over
that `APIService` and the managed add-on stops working, which the platform
treats as a misconfiguration. I want to keep the managed add-on and also
scale on metrics served by my own adapter. With source routing, I deploy my
adapter under its own API group and set `source.apiGroup` on the relevant
HPA metrics, leaving the managed registration untouched.

### Notes/Constraints/Caveats

- The provider contract is unchanged from today's external metrics API. A
  provider that serves `external.metrics.k8s.io` today can serve a different
  group by changing the group it registers; see
  [Provider Contract](#provider-contract).
- The HPA controller does not perform discovery for external metrics. The URL
  version is always `v1beta1`, exactly as it is for
  `external.metrics.k8s.io` today.

### Risks and Mitigations

A provider serving metrics under a custom API group may not implement the
correct wire format. This is not a new risk. The controller already trusts
whatever is registered for `external.metrics.k8s.io/v1beta1`, and a
misbehaving provider there produces the same decode or "no metrics returned"
error today. Source routing does not change what the controller accepts; it
only lets a second provider be reached under a different group. The error is
surfaced in the HPA `ScalingActive` condition with the API group name, so an
operator can tell which provider failed, and the provider contract documents
the exact format.

RBAC may be missing for a new API group - the HPA controller only has
bootstrap RBAC for `external.metrics.k8s.io`. A missing grant produces a 403
that is surfaced in the HPA status condition with the API group name, making
it straightforward to diagnose.

The client map could grow if HPAs reference many distinct API groups.
The controller keeps one REST client per API group in a fixed-size LRU
cache. When the cache is full, the least recently used client is evicted and
rebuilt on its next use, so the effect is on efficiency only, not
correctness. An eviction counter makes this observable (see
[Monitoring Requirements](#monitoring-requirements)). A steadily rising
eviction counter tells an operator that HPAs reference more distinct groups
than the cache holds, which is usually a typo in `source.apiGroup`.

Disabling the feature gate changes metric routing - with the gate disabled,
the controller ignores `source` and fetches every external metric from
`external.metrics.k8s.io`, which may not serve the metric. The HPA status
condition surfaces the error, and this is documented as a known rollback
behavior in the PRR section.

## Design Details

### API Changes

`MetricIdentifier` gains a `Source` field:

```go
// MetricIdentifier defines the name and optionally selector for a metric
type MetricIdentifier struct {
    // name is the name of the given metric
    // +required
    Name string `json:"name" protobuf:"bytes,1,name=name"`

    // selector is the string-encoded form of a standard kubernetes label selector for the given metric
    // +optional
    Selector *metav1.LabelSelector `json:"selector,omitempty" protobuf:"bytes,2,name=selector"`

    // source identifies an alternative metrics API group to fetch this metric from.
    // The provider serving that API group must implement the same wire format as
    // the default provider for the metric type.
    // Only supported for External metrics; when unset, the default API group for
    // the metric type is used.
    // +featureGate=HPAMetricSource
    // +optional
    Source *MetricSourceReference `json:"source,omitempty" protobuf:"bytes,3,opt,name=source"`
}

// MetricSourceReference identifies the API group of a metrics provider.
type MetricSourceReference struct {
    // apiGroup is the API group of a registered APIService that serves this
    // metric, for example "metrics.example.com".
    // +required
    APIGroup string `json:"apiGroup" protobuf:"bytes,1,name=apiGroup"`
}
```

`MetricIdentifier` is shared by `PodsMetricSource`, `ObjectMetricSource` and
`ExternalMetricSource` (and their status counterparts). In this KEP the field
is only honored on `ExternalMetricSource`; see [Validation](#validation).

#### autoscaling/v1

`autoscaling/v1` round-trips non-CPU metrics through annotations using its own
metric struct types. The v1 `ExternalMetricSource` and `ExternalMetricStatus`
structs gain the same optional `source` field, and the v1 conversion copies
it in both directions, so a read-modify-write through `autoscaling/v1` does
not drop routing. This follows the existing pattern for `selector`,
`metricSelector` and container resource metrics.

#### Validation

When `source` is set, the following rules apply (feature gate enabled):

- `source.apiGroup` is required and must be a valid DNS-1123 subdomain.
- `source.apiGroup` must not be one of the groups the controller already
  handles implicitly: `external.metrics.k8s.io`, `custom.metrics.k8s.io` or
  `metrics.k8s.io`. Routing to the default group is expressed by omitting
  `source`, so there is exactly one spelling for the default behavior.
- `source` is only allowed on `spec.metrics[*].external.metric`. Setting it
  on `pods.metric` or `object.metric` is rejected with a message stating that
  source routing is only supported for External metrics. This keeps the door
  open for extending routing to custom metrics later without an API change.
- `source` is mutable.

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
  # Metric from the default provider (external.metrics.k8s.io, no source needed)
  - type: External
    external:
      metric:
        name: rabbitmq_queue_messages_ready
        selector:
          matchLabels:
            queue: inference
      target:
        type: AverageValue
        averageValue: "10"
  # Metric from a model-serving metrics adapter registered under its own group
  - type: External
    external:
      metric:
        name: gen_ai_server_request_queued
        source:
          apiGroup: metrics.modelserver.example.com
      target:
        type: AverageValue
        averageValue: "5"
```

### Wire Format Constraint

This KEP changes only the routing. Everything about how the HPA controller
talks to an external metrics provider, the HTTP request it builds, the URL
version, the query parameters, the response type it expects, how it decodes
the body, how it handles errors, and how it turns the returned values into a
replica count, stays exactly as it is today. When `source.apiGroup` is set,
the one and only difference is the group segment of the URL: the controller
calls `/apis/{apiGroup}/v1beta1/...` instead of
`/apis/external.metrics.k8s.io/v1beta1/...`.

Put another way, a provider that correctly serves `external.metrics.k8s.io`
today can be registered under a different group tomorrow and the controller
will consume it without any change to the controller or the provider's
response format. The full contract is restated in
[Provider Contract](#provider-contract) so that providers targeting a
non-default group know what to implement; none of it is new.

### Controller Changes

The HPA controller has one internal code path per metric type for fetching
values. Only the external metric path changes: it gains an API group
argument. An empty value means "use `external.metrics.k8s.io`", which is the
existing behavior. The resource, pods and object paths are untouched.

When the controller evaluates an `External` metric it reads
`source.apiGroup` from the metric spec (empty if `source` is unset) and passes
it down through the replica calculation into the fetch. Everything after the
fetch, including how values are summed or averaged and how the replica
proposal is computed, is unchanged.

Any error from a source-routed fetch is wrapped with the API group before it
reaches the HPA condition and event. This matters because the aggregator's
own messages do not always name the group: a request for an unregistered
group gets a plain "the server could not find the requested resource". With
the wrapping, every `FailedGetExternalMetric` condition and event names the
provider that failed, which the Troubleshooting section relies on.

When the `HPAMetricSource` feature gate is disabled in
`kube-controller-manager`, the controller passes an empty API group regardless
of what the spec says, so `source` has no effect.

### Fetching from a Non-Default Group

Today the controller holds a single external metrics client, built once at
startup against `external.metrics.k8s.io/v1beta1`. This KEP adds a small
factory next to it that can produce an equivalent client for any API group.

The fetch logic becomes:

1. If the API group is empty, use the existing default client. This is the
   current behavior and the common case.
2. Otherwise, ask the factory for a client for that group and issue exactly
   the same request against it, with only the group segment of the URL
   changed: `GET /apis/{apiGroup}/v1beta1/namespaces/{ns}/{metricName}`.
3. If the factory cannot produce a client, return the error. The controller
   never silently falls back to the default group, because that would hide a
   misrouted metric.

The external metrics client library in `k8s.io/metrics` gains a constructor
that is identical to the existing one except that the API group is a
parameter. The version stays `v1beta1`, and the serializer and REST path
construction are unchanged. This is the only change to that library.

The factory keeps one client per API group in a map, created on first use.
Reuse is not required for correctness, but it is required for client-side
rate limiting: each newly built client gets its own token bucket when the
config sets QPS, so a client built on every reconcile would start each call
with a full bucket. TLS transports and connection pools are already shared by
client-go across clients built from the same config, so they are not the
concern.

The number of entries is bounded by the number of distinct `apiGroup` values
across all HPAs, which in practice is a handful, but a typo or a hostile
tenant can add entries. The map is therefore a fixed-size LRU cache holding
100 clients. it is far above any realistic number of providers in one cluster,
and a cached client costs little memory. It can be revisited before beta based on the 
eviction counter users feedback. When the cache is full and a fetch needs 
a client for a new group, the least recently used client is evicted and the new one takes its place;
the evicted client is rebuilt on its next use. Eviction is logged and counted.

### RBAC

No changes to Kubernetes bootstrap RBAC. The existing
`system:controller:horizontal-pod-autoscaler` ClusterRole, which grants
`get`, `list` and `watch` on `*` in `external.metrics.k8s.io`, is untouched.

Providers grant the HPA controller access to their API group via their own
install manifests (Helm chart, operator, etc.):

```yaml
apiVersion: rbac.authorization.k8s.io/v1
kind: ClusterRole
metadata:
  name: hpa-reader:modelserver-metrics
rules:
- apiGroups: ["metrics.modelserver.example.com"]
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

The controller copies `source` from the spec into
`status.currentMetrics[].external.metric` so that the group a value came from
is visible without cross-referencing the spec. Today the controller builds the
status metric identifier from the name and selector only, so copying `source`
is an explicit addition to the status-building logic for external metrics.

```yaml
status:
  currentMetrics:
  - type: External
    external:
      metric:
        name: gen_ai_server_request_queued
        source:
          apiGroup: metrics.modelserver.example.com
      current:
        averageValue: "3"
```

When the feature gate is disabled, the controller does not write `source` to
status.

### Provider Contract

A provider serving external metrics under its own API group MUST satisfy the
following. This is the same contract `external.metrics.k8s.io` providers
satisfy today, with the group name substituted.

Registration:
- Register an `APIService` named `v1beta1.{apiGroup}` for group `{apiGroup}`
  and version `v1beta1`, pointing to a Service backed by the provider's pods.
- Grant the HPA controller identity (by default
  `system:serviceaccount:kube-system:horizontal-pod-autoscaler`) `get` and
  `list` on all resources in `{apiGroup}` via RBAC.

Discovery:
- Respond to `GET /apis/{apiGroup}/v1beta1` with an `APIResourceList`. The
  kube-aggregator uses this endpoint for its availability check. The
  `/apis` and `/apis/{apiGroup}` documents are served by the aggregator
  itself from the `APIService` registration; the HPA controller does not
  call them for external metrics.

Metric endpoint:

```
GET /apis/{apiGroup}/v1beta1/namespaces/{namespace}/{metricName}?labelSelector={selector}
```

Response: a JSON body with the shape of `ExternalMetricValueList`:
`items[].metricName`, `items[].value`, `items[].timestamp`, and optionally
`items[].metricLabels` and `items[].window`. The controller decodes the
body directly into that type and does not validate `apiVersion` or `kind`;
this is how it behaves for `external.metrics.k8s.io` today. Providers are
encouraged to emit `apiVersion: {apiGroup}/v1beta1` and
`kind: ExternalMetricValueList` for clarity.

The URL version is always `v1beta1`. The controller performs no discovery
against the provider's group, so a provider must serve that version.

### Test Plan

[x] I/we understand the owners of the involved components may require updates to
existing tests to make this code solid enough prior to committing the changes necessary
to implement this enhancement.

##### Prerequisite testing updates

None required.

##### Unit tests

- Validation and feature gating:
  - With the gate disabled, the API server clears `source` on create and on
    updates to HPAs that did not already have it, and preserves it on HPAs
    that already use it
  - `source.apiGroup` must be a valid DNS subdomain and not a reserved group
  - `source` is rejected on `pods.metric` and `object.metric`
  - `source` survives the `autoscaling/v1` and `autoscaling/v2` round trips
- MetricsClient routing:
  - Empty `apiGroup` delegates to the default external client (existing
    behavior)
  - Non-empty `apiGroup` uses the factory and the request goes to
    `/apis/{apiGroup}/v1beta1/...`
  - Client cache reuse across calls, and LRU eviction when the cache is full
- Controller plumbing:
  - `source` is extracted from the External metric spec and passed to the
    metrics client
  - `source` is copied into `status.currentMetrics` when the gate is enabled,
    and omitted when it is disabled

- `pkg/apis/autoscaling/validation`: 2026-08-31 - 95.3% of statements
- `pkg/controller/podautoscaler`: 2026-08-31 - 89.0% of statements

##### Integration tests

The existing HPA integration tests in `test/integration/podautoscaler` run
the controller in-process against a real apiserver, with a fake external
metrics client injected in place of the aggregated API. The new tests follow
the same pattern, injecting a fake client factory that returns a distinct
fake client per API group, each serving a different value:

- HPA fetches an external metric from a non-default API group when
  `source.apiGroup` is set
- HPA fetches from `external.metrics.k8s.io` when `source` is not set
- HPA handles mixed metrics (some with `source`, some without) in a single
  spec
- HPA status reflects the API group for each external metric
- HPA surfaces a `FailedGetExternalMetric` condition when no client can be
  obtained for the source API group
- Gate disabled: an HPA with `source` set falls back to
  `external.metrics.k8s.io`

Failure modes that depend on the aggregation layer itself (API group not
registered, backend unavailable, RBAC missing) are covered by the e2e tests
below, where the request actually traverses kube-aggregator.

##### e2e tests

E2e tests with the `agnhost external-metrics` server registered under a
non-default API group will be added as part of beta graduation:

- HPA scales on an external metric served under a non-default API group
- HPA surfaces a clear `FailedGetExternalMetric` condition, naming the API
  group, when the source API group is not registered, when the provider's
  pods are scaled to zero (backend unavailable), when RBAC is missing, and
  when the provider returns an error for the metric (using the server's
  `/fail/` endpoint)

### Graduation Criteria

#### Alpha

- Unit and integration tests passed as designed in the [Test Plan](#test-plan)
- Feature implemented behind the `HPAMetricSource` feature gate (default off)
- E2e tests in place

#### Beta

- Feature gate enabled by default

#### GA

- No negative feedback
- All issues and gaps identified during beta are resolved
- Feedback gathered from at least one real deployment running an external
  metrics provider under a non-default API group alongside the default one

### Upgrade / Downgrade Strategy

#### Upgrade

When the feature gate is enabled:
- Existing HPAs continue to work unchanged
- Metrics without `source` behave as they do today
- Users can add `source` to `External` metrics in their HPAs
- The controller routes those metrics to the specified API group and reflects
  it in status

#### Downgrade

When the feature gate is disabled:
- `kube-apiserver` drops `source` from new HPAs and from updates to HPAs that
  did not already have it; HPAs that already use `source` keep the field so
  that re-enabling the gate restores routing
- `kube-controller-manager` ignores `source` and fetches every external metric
  from `external.metrics.k8s.io`
- HPAs that relied on source routing keep their current replica count and
  surface a `FailedGetExternalMetric` condition if the default provider cannot
  serve the metric
- No pods are restarted or disrupted

### Version Skew Strategy

1. `kube-apiserver`: a newer apiserver with the gate enabled accepts,
   validates and persists `source`. An older apiserver, or one with the gate
   disabled, drops the field on write (unknown or disabled fields are pruned
   on decode), so during a skewed control plane rollout a write that lands on
   an older instance loses routing for that HPA until it is rewritten through
   a newer instance.
2. `kube-controller-manager`: an older controller manager that receives an HPA
   containing `source` from a newer apiserver ignores the field and fetches
   all external metrics from `external.metrics.k8s.io`.

## Production Readiness Review Questionnaire

### Feature Enablement and Rollback

###### How can this feature be enabled / disabled in a live cluster?

- [x] Feature gate (also fill in values in `kep.yaml`)
  - Feature gate name: `HPAMetricSource`
  - Components depending on the feature gate: `kube-apiserver` and `kube-controller-manager`

###### Does enabling the feature change any default behavior?

No. HPAs continue to behave as they do today. The feature only activates when
users explicitly set `source` on an `External` metric. Metrics without
`source` continue to be fetched from `external.metrics.k8s.io`.

###### Can the feature be disabled once it has been enabled (i.e. can we roll back the enablement)?

Yes. When the feature gate is disabled:
- `kube-apiserver` drops `source` from new objects and from updates to objects
  that did not already have it. Objects that already use `source` keep it.
- `kube-controller-manager` ignores `source` and fetches all external metrics
  from `external.metrics.k8s.io`.
- No pods are restarted or disrupted.

###### What happens if we reenable the feature if it was previously rolled back?

HPAs that still have `source` set resume routing on the next reconcile. The
controller rebuilds clients for the referenced API groups on demand. HPAs whose
`source` was dropped while the gate was off must be updated again.

###### Are there any tests for feature enablement/disablement?

Yes. Unit tests will verify:
- `source` is accepted when the gate is enabled
- With the gate disabled, the API server clears `source` except on update of
  an HPA that already uses it
- the controller ignores `source` when the gate is disabled

Integration tests will verify that an HPA with `source` falls back to
`external.metrics.k8s.io` when the controller manager gate is disabled.

### Rollout, Upgrade and Rollback Planning

###### How can a rollout or rollback fail? Can it impact already running workloads?

A rollback makes the controller ignore `source`.
HPAs that were fetching metrics from non-default providers switch to
`external.metrics.k8s.io`. If that provider does not serve the metric, the HPA
stops scaling on that metric and surfaces a `FailedGetExternalMetric`
condition. Running workloads are not disrupted; pods are not restarted and the
current replica count is maintained.

###### What specific metrics should inform a rollback?

- A rise in `horizontal_pod_autoscaler_controller_external_metric_fetches_total{result="error"}`
  for a given `api_group`
- A rise in `horizontal_pod_autoscaler_controller_metric_computation_total{metric_type="External",error!="none"}`
- A rise in `FailedGetExternalMetric` events

###### Were upgrade and rollback tested? Was the upgrade->downgrade->upgrade path tested?

Not yet. Manual upgrade → downgrade → upgrade testing will be performed and
documented before beta.

###### Is the rollout accompanied by any deprecations and/or removals of features, APIs, fields of API types, flags, etc.?

No.

### Monitoring Requirements

###### How can an operator determine if the feature is in use by workloads?

Check whether any HPA has `source` set on an external metric:

```
kubectl get hpa -A -o json | jq '[.items[].spec.metrics[]? | select(.external.metric.source != null)] | length'
```

###### How can someone using this feature know that it is working for their instance?

- [x] API .status
  - Condition name: `ScalingActive` is `True`
  - Other field: `status.currentMetrics[].external.metric.source.apiGroup`
    shows the API group used for each routed metric
- [x] Events
  - Event Reason: `FailedGetExternalMetric` (the message includes the API
    group when source routing fails)

###### What are the reasonable SLOs (Service Level Objectives) for the enhancement?

HPA controller reconcile latency should not be affected. Source routing
changes which endpoint a metric is fetched from; the number of REST calls per
reconcile and the wire format are unchanged.

###### What are the SLIs (Service Level Indicators) an operator can use to determine the health of the service?

- [x] Metrics
  - Existing:
    `horizontal_pod_autoscaler_controller_metric_computation_total` and
    `horizontal_pod_autoscaler_controller_metric_computation_duration_seconds`,
    filtered on `metric_type="External"`
  - New in alpha:
    - `horizontal_pod_autoscaler_controller_external_metric_fetches_total`
      (counter) with labels `api_group` and `result` (`success` or
      `error`): one increment per external metric fetch. `api_group` is
      `external.metrics.k8s.io` when `source` is unset. This is the metric
      that tells an operator which provider is failing. Its cardinality is
      bounded by the client cache capacity.
    - `horizontal_pod_autoscaler_controller_metric_source_client_evictions_total`
      (counter): number of per-group clients evicted because the cache was
      full. A steadily rising value means HPAs reference more distinct API
      groups than the cache holds.
  - Components exposing the metrics: `kube-controller-manager`

###### Are there any missing metrics that would be useful to have to improve observability of this feature?

No. The per-group fetch counter above covers the observability gap this
feature introduces.

### Dependencies

###### Does this feature depend on any specific services running in the cluster?

- External metrics providers registered via `APIService`
  - Usage description: the feature routes metric fetches to API groups
    registered as `APIService`s. Providers must be deployed and serving.
  - Impact of its outage on the feature: metrics from the unavailable provider
    cannot be fetched; the HPA status surfaces the error. Other metrics, from
    other providers or from the default provider, continue to work.
  - Impact of its degraded performance or high-error rates on the feature:
    increased latency in the HPA reconcile loop for affected metrics.

### Scalability

###### Will enabling / using this feature result in any new API calls?

No new calls are added per metric. When `source.apiGroup` is set, the one
GET the controller already makes per external metric per reconcile goes to
`/apis/{apiGroup}/v1beta1/...` instead of
`/apis/external.metrics.k8s.io/v1beta1/...`.

- API call type: GET on an aggregated API endpoint
- Estimated throughput: one GET per source-routed external metric per HPA
  reconcile (unchanged from today)
- Originating component: `kube-controller-manager` (HPA controller)

###### Will enabling / using this feature result in introducing new API types?

No. The feature only adds new fields to existing API types:
- New `MetricSourceReference` struct
- New `source` field in `autoscaling/v2` `MetricIdentifier` (spec and status)
- New `source` field in `autoscaling/v1` `ExternalMetricSource` and
  `ExternalMetricStatus`

###### Will enabling / using this feature result in any new calls to the cloud provider?

No direct calls. If a metrics provider running in-cluster calls a cloud
provider to serve metrics, those calls are unchanged by this feature.

###### Will enabling / using this feature result in increasing size or count of the existing API objects?

- API type: HorizontalPodAutoscaler
- Estimated increase in size: ~50 bytes per metric with `source` set

###### Will enabling / using this feature result in increasing time taken by any operations covered by existing SLIs/SLOs?

No measurable change. The first reconcile that references a new API group
pays a one-time cost to construct a REST client (no network call). Subsequent
reconciles reuse the cached client and make the same single GET as today.

###### Will enabling / using this feature result in non-negligible increase of resource usage (CPU, RAM, disk, IO, ...) in any components?

Minimal increase in `kube-controller-manager` memory for the per-group client
cache, which is bounded. Each cached client is a standard REST client.

###### Can enabling / using this feature result in resource exhaustion of some node resources (PIDs, sockets, inodes, etc.)?

No. All clients built from the controller's config share client-go's
transport cache, so connections are pooled per API server endpoint
regardless of how many API groups are referenced.

### Troubleshooting

###### How does this feature react if the API server and/or etcd is unavailable?

Same as existing HPA behavior: the controller cannot reconcile HPAs and
retries on the next sync period.

###### What are other known failure modes?

- Source API group not registered
  - Detection: `ScalingActive=False` with reason `FailedGetExternalMetric`
    and a 404 ("the server could not find the requested resource") message
    naming the API group
  - Mitigations: verify the `APIService` exists (`kubectl get apiservices`)
  - Diagnostics: HPA events and controller logs include the API group
  - Testing: e2e tests cover the unregistered-group case

- Source API group registered but backend unavailable
  - Detection: `ScalingActive=False` with reason `FailedGetExternalMetric`
    and a 503 ("the server is currently unable to handle the request")
    message
  - Mitigations: check the `APIService` `Available` condition and the
    provider's pods
  - Diagnostics: `kubectl describe apiservice v1beta1.{apiGroup}`
  - Testing: e2e tests cover the unavailable-backend case

- RBAC missing for the source API group
  - Detection: `ScalingActive=False` with reason `FailedGetExternalMetric`
    and a 403 Forbidden message naming the API group
  - Mitigations: add a ClusterRole and ClusterRoleBinding granting the HPA
    controller identity `get` and `list` on the API group
  - Diagnostics: API server audit logs show the 403
  - Testing: e2e tests cover the missing-RBAC case

- Provider serves the wrong wire format
  - Detection: `ScalingActive=False` with reason `FailedGetExternalMetric`
    and either a decode error or "no metrics returned from external metrics
    API"
  - Mitigations: verify the provider implements the
    [Provider Contract](#provider-contract)
  - Diagnostics: query the endpoint directly with
    `kubectl get --raw /apis/{apiGroup}/v1beta1/namespaces/{ns}/{metricName}`
  - Testing: unit tests cover malformed responses

###### What steps should be taken if SLOs are not being met to determine the problem?

Check `horizontal_pod_autoscaler_controller_metric_computation_duration_seconds{metric_type="External"}`
to see whether latency correlates with HPAs that use source routing. Then:

- Check whether the issue only affects HPAs with `source` configured
- Review HPA events with `kubectl describe hpa <name>`
- Check the source API group's `APIService` health with `kubectl get apiservices`
- Verify RBAC grants the HPA controller identity access to the API group

For a problematic HPA, temporarily remove `source` to revert that metric to
`external.metrics.k8s.io`, or query the provider endpoint directly with
`kubectl get --raw` to confirm it is reachable and returns the expected format.

## Implementation History

- 2026-11-04: KEP created

## Drawbacks

Management overhead. Each additional external metrics provider is one more
`APIService`, one more RBAC grant and one more component for operators to
keep healthy, and the HPA controller now depends on every API group that any
HPA references rather than on a single well-known one. A slow or broken
provider affects every HPA that routes to it.

## Alternatives

A fan-out adapter that owns `external.metrics.k8s.io` and proxies to
backends - works today, but every provider must be wired through one
adapter, adds a hop per fetch, and does not help when the default group is
owned by a managed add-on the user cannot reconfigure.

## Infrastructure Needed (Optional)

None.
