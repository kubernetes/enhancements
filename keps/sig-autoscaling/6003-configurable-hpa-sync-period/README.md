# KEP-6003: Configurable sync period for HPA

<!-- toc -->
- [Release Signoff Checklist](#release-signoff-checklist)
- [Summary](#summary)
- [Motivation](#motivation)
  - [Goals](#goals)
  - [Non-Goals](#non-goals)
- [Proposal](#proposal)
  - [User Stories](#user-stories)
    - [Story 1](#story-1)
    - [Story 2](#story-2)
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

[kubernetes.io]: https://kubernetes.io/
[kubernetes/enhancements]: https://git.k8s.io/enhancements
[kubernetes/kubernetes]: https://git.k8s.io/kubernetes
[kubernetes/website]: https://git.k8s.io/website

## Summary

[Horizontal Pod Autoscaler][] (HPA) periodically reconciles the desired replica
count for a given Deployment (or other resource with a `/scale` subresource)
based on observed metrics. The frequency at which this reconciliation happens
is governed by a single global flag,
`--horizontal-pod-autoscaler-sync-period`, which defaults to 15 seconds
and applies to every HPA in the cluster.

This proposal adds an optional `syncPeriodSeconds` field to
`HorizontalPodAutoscalerBehavior` that allows users to override the global sync
period on a per-HPA basis. When the field is unset, the global default
continues to apply.

[Horizontal Pod Autoscaler]: https://kubernetes.io/docs/tasks/run-application/horizontal-pod-autoscale/

## Motivation

Today the HPA sync period is a cluster-wide parameter set using the
[Kube Controller Manager][]
`--horizontal-pod-autoscaler-sync-period` flag (default 15s). While this
value is reasonable for many workloads, it is a one-size-fits-all setting
that forces every HPA in the cluster to reconcile at the same frequency.

This creates a tension between two competing goals:

1. **Latency-sensitive workloads** need faster scaling decisions (e.g. every
   1-5 seconds) to respond to rapid traffic spikes.
2. **Stable workloads** are fine with the default or even longer periods, and
   increasing the global frequency for all HPAs unnecessarily increases load
   on the API server and metrics backends.

Since appropriate sync periods are workload-dependent, this KEP proposes to
let users set a custom sync period per `HorizontalPodAutoscaler` resource,
overriding the global default when present.

[Kube Controller Manager]: https://kubernetes.io/docs/reference/command-line-tools-reference/kube-controller-manager/

### Goals

- Allow users to optionally override the default HPA reconciliation frequency
  on a per-HPA basis.
- Maintain full backward compatibility: existing HPAs without the new field
  continue to use the global sync period.

### Non-Goals

- Change the default value of the global
  `--horizontal-pod-autoscaler-sync-period` flag.
- Allow sub-second sync periods.

## Proposal

We propose to add a new field to the existing [`HorizontalPodAutoscalerBehavior`][] object:

- `syncPeriodSeconds`: *(int32)* the period in seconds between each
  reconciliation of this HPA. Must be greater than 0 and less than or equal
  to 3600 (one hour).

The `syncPeriodSeconds` field is optional, and when not specified the HPA will
continue to use the value of the global
`--horizontal-pod-autoscaler-sync-period` flag.

**Interaction with metrics sources**: The effectiveness of reducing the sync
period depends on the freshness of the underlying metrics. For resource metrics
served by [metrics-server][], values are typically collected at a fixed interval
(default 60s) and cached, so syncing the HPA faster than the metrics collection
interval will not yield fresher data -- the HPA will simply re-evaluate the same
cached values. Reducing the sync period is most impactful when used with
custom or external metrics providers that expose rapidly updating values. Users
should consider their metrics pipeline latency when choosing a
`syncPeriodSeconds` value.

This field is placed under `Behavior` rather than at the top-level spec because
it is an operational tuning parameter that controls *how* the HPA operates,
consistent with the existing `scaleUp`/`scaleDown` rules and the `tolerance`
field introduced in [KEP-4951][].

[ValidatingAdmissionPolicy]: https://kubernetes.io/docs/reference/access-authn-authz/validating-admission-policy/
[`HorizontalPodAutoscalerBehavior`]: https://kubernetes.io/docs/reference/generated/kubernetes-api/v1.35/#horizontalpodautoscalerbehavior-v2-autoscaling
[KEP-4951]: /keps/sig-autoscaling/4951-configurable-hpa-tolerance
[metrics-server]: https://github.com/kubernetes-sigs/metrics-server

### User Stories

#### Story 1

As an operator of a latency-sensitive web application, I want my HPA to
reconcile every 2 seconds so that scale-up decisions are made quickly during
sudden traffic spikes, without forcing every other HPA in the cluster to also
reconcile that frequently.

#### Story 2

As a platform engineer managing a multi-tenant cluster, I want different teams
to be able to tune HPA responsiveness for their own workloads independently,
without requiring cluster-admin involvement to change the global flag.

### Risks and Mitigations

There should be minimal risk introduced by the proposed changes:

- **Increased API server and metrics load**: Users could set very low sync
  periods for many HPAs, increasing the rate of metrics queries and scale
  sub-resource calls. This is mitigated by multiple layers:
  - **Validation bounds**: `syncPeriodSeconds` must be >= 1 and <= 3600. We
    may raise the lower bound (e.g. to 5s) before Beta if real-world usage
    data suggests that very short periods cause issues in practice.
  - **Feature gate**: In Alpha, the feature is gated behind
    `HPAConfigurableSyncPeriod`, giving cluster administrators explicit
    control over whether the field is accepted at all.
  - **Best-effort semantics**: The field specifies a *target* interval, not a
    hard guarantee. If a reconciliation cycle takes longer than the configured
    period, the controller will not queue additional work -- it will simply
    start the next cycle once the current one completes. This prevents
    workqueue saturation (see [Design Details](#design-details)).
  - **Policy enforcement**: Cluster administrators can use [ValidatingAdmissionPolicy][]
    (or a webhook) to set a cluster-specific floor. For example, a simple CEL
    rule can reject HPAs with `syncPeriodSeconds` below a chosen threshold
    without needing to reason about aggregate frequency:
    ```
    rule: "!has(object.spec.behavior) ||
           !has(object.spec.behavior.syncPeriodSeconds) ||
           object.spec.behavior.syncPeriodSeconds >= 10"
    ```
- The new field is optional, and its absence results in no changes to the
  current autoscaling behavior.
- If a change to the new field results in undesirable behavior, the change
  can be reverted by removing the `syncPeriodSeconds` field from the HPA,
  restoring the global default.

## Design Details

The `HorizontalPodAutoscaler` API is updated to add a new `syncPeriodSeconds`
field to the `HorizontalPodAutoscalerBehavior` object:

```golang
type HorizontalPodAutoscalerBehavior struct {
  // syncPeriodSeconds is the period in seconds between each reconciliation
  // of this HPA. When unset, the global
  // --horizontal-pod-autoscaler-sync-period value is used (default: 15s).
  // Must be greater than 0 and less than or equal to 3600.
  // +optional
  SyncPeriodSeconds *int32

  // Existing fields.
  ScaleUp *HPAScalingRules
  ScaleDown *HPAScalingRules
}
```

The `syncPeriodSeconds` field specifies a best-effort target interval: the
controller will attempt to reconcile the HPA at least this often, but does not
guarantee a hard real-time deadline. If a reconciliation cycle (including
metrics queries and scale sub-resource calls) takes longer than the configured
period, the controller will start the next cycle immediately after the current
one completes rather than queuing up additional work. This prevents workqueue
saturation even when the metrics backend is slow or the configured period is
shorter than the end-to-end reconciliation latency.

The per-HPA sync frequency is implemented via a new `PerItemIntervalRateLimiter`
in the HPA controller's workqueue. This rate limiter supports per-key interval
overrides with a fallback to the global default. The controller updates the
per-item interval during reconciliation and cleans it up on HPA deletion.

The informer event handlers are updated so that:
- Newly created HPAs and spec changes (detected via `Generation` comparison)
  are enqueued immediately via `queue.Add`, so they are processed without
  waiting for the rate limiter delay. Note: this depends on the HPA
  `Generation` field being properly incremented on spec changes, which is
  being addressed in [kubernetes#138228][].
- Status-only or metadata-only updates continue to use `AddRateLimited` to
  preserve the hot-loop prevention introduced in [#42715][].
- The periodic resync cadence is maintained by the existing `AddRateLimited`
  call in `processNextWorkItem`, which re-enqueues each HPA with its
  configured (or default) sync period delay after every reconciliation.

The informer's `AddEventHandlerWithResyncPeriod` continues to use the global
`resyncPeriod` as a background safety net; the per-HPA sync frequency is
driven entirely by the workqueue rate limiter.

Since the added field is optional and its omission does not change the existing
autoscaling behavior, this feature will only be added to the latest stable API
version `pkg/apis/autoscaling/v2`. Older versions (i.e. `v1`, `v2beta1`,
`v2beta2`) will not include the new field, but converters will be updated where
needed to comply with [round-trip requirements][].

The validation logic will be updated to ensure that the `syncPeriodSeconds`
field is greater than 0 and does not exceed 3600.

[#42715]: https://github.com/kubernetes/kubernetes/pull/42715
[kubernetes#138228]: https://github.com/kubernetes/kubernetes/pull/138228
[round-trip requirements]: https://kubernetes.io/docs/reference/using-api/deprecation-policy/#deprecating-parts-of-the-api

### Test Plan

[x] I/we understand the owners of the involved components may require updates to
existing tests to make this code solid enough prior to committing the changes necessary
to implement this enhancement.

##### Prerequisite testing updates

##### Unit tests

- `pkg/apis/autoscaling/validation`: `<date>` - `<test coverage>`
- `pkg/controller/podautoscaler`: `<date>` - `<test coverage>`

##### Integration tests

<!--
This question should be filled when targeting a release.
For Alpha, describe what tests will be added to ensure proper quality of the enhancement.
-->

An integration test will be added to verify the autoscaling behavior when
custom sync periods are set on an HPA and that HPAs without the field
continue to use the global default.

##### e2e tests

<!--
This question should be filled when targeting a release.
For Alpha, describe what tests will be added to ensure proper quality of the enhancement.
-->

Existing e2e tests ensure the autoscaling behavior uses the default sync period
when no configurable sync period is specified.

New e2e tests will be added to verify:
- An HPA with a short `syncPeriodSeconds` reconciles more frequently than the
  global default.
- An HPA without `syncPeriodSeconds` continues to reconcile at the global
  default rate.

### Graduation Criteria

#### Alpha

- Feature implemented behind a `HPAConfigurableSyncPeriod` feature flag
- Initial e2e tests completed and enabled

#### Beta

- All tests described in the [`e2e tests` section](#e2e-tests) are implemented
  and linked in this KEP.
- We have monitored for negative user feedback and addressed relevant concerns.
- Integration test verifying behavior with and without the field.

### Upgrade / Downgrade Strategy

#### Upgrade
Existing HPAs will continue to work as they do today, using the global
`--horizontal-pod-autoscaler-sync-period` value from the
`kube-controller-manager`. Users can use the new feature by enabling the
Feature Gate (alpha only) and setting the new `syncPeriodSeconds` field in
an HPA's `spec.behavior`.

#### Downgrade
On downgrade, all HPAs will revert to using the global
`--horizontal-pod-autoscaler-sync-period` value from the
`kube-controller-manager`, regardless of any configured `syncPeriodSeconds`
value on the HPA itself.

### Version Skew Strategy

1. `kube-apiserver`: More recent instances will accept the new
   `syncPeriodSeconds` field, while older instances will ignore it.
2. `kube-controller-manager`: An older version could receive an HPA containing
   the new `syncPeriodSeconds` field from a more recent API server, in which
   case it would ignore it (i.e. reconcile at the global default rate).

## Production Readiness Review Questionnaire

### Feature Enablement and Rollback

###### How can this feature be enabled / disabled in a live cluster?

- [x] Feature gate (also fill in values in `kep.yaml`)
  - Feature gate name: HPAConfigurableSyncPeriod
  - Components depending on the feature gate: `kube-controller-manager` and
    `kube-apiserver`.

###### Does enabling the feature change any default behavior?

No.

###### Can the feature be disabled once it has been enabled (i.e. can we roll back the enablement)?

The feature can be disabled by restarting the `kube-controller-manager` with
the feature gate set to `false`.

Any `syncPeriodSeconds` values set on existing HPAs will be ignored by the
`kube-controller-manager` and `kube-apiserver` when the feature gate is off.
All HPAs will revert to using the global sync period.

###### What happens if we reenable the feature if it was previously rolled back?

When the feature is re-enabled, any HPAs with configured `syncPeriodSeconds`
values will use those as their reconciliation interval, rather than the global
sync period from the `kube-controller-manager`.

###### Are there any tests for feature enablement/disablement?

Unit tests will be added to verify that HPAs with and without the new field
are properly validated, both when the feature gate is enabled or not.

### Rollout, Upgrade and Rollback Planning

###### How can a rollout or rollback fail? Can it impact already running workloads?

This feature does not introduce new failure modes: during rollout/rollback,
some API servers will allow or disallow setting the new `syncPeriodSeconds`
field. The new field is possibly ignored until the controller manager is fully
updated.

###### What specific metrics should inform a rollback?

A high `horizontal_pod_autoscaler_controller_metric_computation_duration_seconds`
metric can indicate a problem related to this feature. Additionally, unexpected
increases in API server request rates from the HPA controller may indicate that
sync periods are set too aggressively.

###### Were upgrade and rollback tested? Was the upgrade->downgrade->upgrade path tested?

<!--
Describe manual testing that was done and the outcomes.
Longer term, we may want to require automated upgrade/rollback tests, but we
are missing a bunch of machinery and tooling and can't do that now.
-->

TBD

###### Is the rollout accompanied by any deprecations and/or removals of features, APIs, fields of API types, flags, etc.?

No.

### Monitoring Requirements

###### How can an operator determine if the feature is in use by workloads?

The presence of the new `syncPeriodSeconds` field in the HPA's
`spec.behavior` indicates that the feature is in use.

###### How can someone using this feature know that it is working for their instance?

- [X] Events
  - Event Reason: `SuccessfulRescale`

Users can observe the frequency of `SuccessfulRescale` events and compare it
to their configured `syncPeriodSeconds` value. If the HPA is reconciling at
the expected rate, the feature is working correctly.

###### What are the reasonable SLOs (Service Level Objectives) for the enhancement?

Although the absolute value of the
`horizontal_pod_autoscaler_controller_metric_computation_duration_seconds`
metric depends on HPAs configuration, it should be unimpacted by this feature.
This metric should not vary by more than 5%.

###### What are the SLIs (Service Level Indicators) an operator can use to determine the health of the service?

This KEP is not expected to have any impact on SLIs/SLOs as it doesn't
introduce a new HPA behavior, but merely allows users to easily change the
reconciliation frequency that is otherwise a global parameter.

The standard HPA metric
`horizontal_pod_autoscaler_controller_metric_computation_duration_seconds` can
be used to verify the HPA controller health.

###### Are there any missing metrics that would be useful to have to improve observability of this feature?

A per-HPA metric exposing the effective sync period being used (configured vs.
global default) could be useful for debugging, but is not strictly necessary
for the initial implementation.

### Dependencies

###### Does this feature depend on any specific services running in the cluster?

No, this feature does not depend on any specific service.

### Scalability

###### Will enabling / using this feature result in any new API calls?

No new API call types are introduced. However, setting a lower
`syncPeriodSeconds` on an HPA will increase the frequency of existing API
calls (metrics queries, scale sub-resource reads/updates) for that specific
HPA proportionally.

###### Will enabling / using this feature result in introducing new API types?

No.

###### Will enabling / using this feature result in any new calls to the cloud provider?

No.

###### Will enabling / using this feature result in increasing size or count of the existing API objects?

- This feature adds one new optional int32 field to
  `HorizontalPodAutoscaler` `v2` objects. Users should expect this object to
  increase in size by approximately 4 bytes when the field is set.

###### Will enabling / using this feature result in increasing time taken by any operations covered by existing SLIs/SLOs?

No.

###### Will enabling / using this feature result in non-negligible increase of resource usage (CPU, RAM, disk, IO, ...) in any components?

Setting lower `syncPeriodSeconds` values will proportionally increase
CPU usage and API call volume on the `kube-controller-manager` and metrics
backend for the affected HPAs. The validation bound (minimum 1s, maximum 3600s)
limits the impact, but operators should monitor resource usage when configuring
aggressive sync periods on many HPAs.

###### Can enabling / using this feature result in resource exhaustion of some node resources (PIDs, sockets, inodes, etc.)?

No. The feature only affects the HPA controller's reconciliation frequency
within the control plane. Node-level resources are not directly impacted.

### Troubleshooting

###### How does this feature react if the API server and/or etcd is unavailable?

API server or etcd issues do not impact this feature specifically. The HPA
controller will be unable to reconcile regardless of the sync period
configuration if the API server is unavailable.

###### What are other known failure modes?

We do not expect any new failure modes. Setting a very low `syncPeriodSeconds`
(e.g. 1s) on many HPAs could increase control plane load, but the feature is
still working as intended. To reduce load, increase the `syncPeriodSeconds`
value or remove it to fall back to the global default.

###### What steps should be taken if SLOs are not being met to determine the problem?

If possible increase the log level for kube-controller-manager and check
controller logs:
1. Verify that the `syncPeriodSeconds` value is being respected by observing
   the reconciliation frequency in logs and events.
2. Check if the increased reconciliation frequency is causing excessive load
   on the API server or metrics backend.
3. Look for warnings and errors which might point where the problem lies.

## Implementation History

2026-04-08: Initial KEP created.
2026-04-06: [Implementation PR](https://github.com/kubernetes/kubernetes/pull/138222) opened.

## Drawbacks

Setting aggressive (low) sync periods on a large number of HPAs may increase
control plane load. However, this is an explicit user choice, bounded by
validation, and can be reverted by removing the field.

## Alternatives

- **Change the global flag**: On non-managed Kubernetes instances, users can
  update the cluster-wide `--horizontal-pod-autoscaler-sync-period` flag, but
  this affects all HPAs uniformly and requires cluster-admin access and a
  controller-manager restart.
- **Per-HPA annotation**: An annotation-based approach was considered but
  rejected in favor of a first-class API field, which provides validation,
  documentation, and discoverability.

## Infrastructure Needed (Optional)

N/A.
