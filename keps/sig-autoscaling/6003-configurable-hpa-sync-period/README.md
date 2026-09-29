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
    - [Story 3](#story-3)
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

Items marked with (R) are required *prior to targeting to a milestone / release*.

- [ ] (R) Enhancement issue in release milestone, which links to KEP dir in [kubernetes/enhancements] (not the initial KEP PR)
- [ ] (R) KEP approvers have approved the KEP status as `implementable`
- [x] (R) Design details are appropriately documented
- [x] (R) Test plan is in place, giving consideration to SIG Architecture and SIG Testing input (including test refactors)
  - [ ] e2e Tests for all Beta API Operations (endpoints)
  - [ ] (R) Ensure GA e2e tests meet requirements for [Conformance Tests](https://github.com/kubernetes/community/blob/master/contributors/devel/sig-architecture/conformance-tests.md)
  - [ ] (R) Minimum Two Week Window for GA e2e tests to prove flake free
- [x] (R) Graduation criteria is in place
  - [ ] (R) [all GA Endpoints](https://github.com/kubernetes/community/pull/1806) must be hit by [Conformance Tests](https://github.com/kubernetes/community/blob/master/contributors/devel/sig-architecture/conformance-tests.md)
- [ ] (R) Production readiness review completed
- [ ] (R) Production readiness review approved
- [x] "Implementation History" section is up-to-date for milestone
- [ ] User-facing documentation has been created in [kubernetes/website], for publication to [kubernetes.io]
- [ ] Supporting documentation—e.g., additional design documents, links to mailing list discussions/SIG meetings, relevant PRs/issues, release notes

[kubernetes.io]: https://kubernetes.io/
[kubernetes/enhancements]: https://git.k8s.io/enhancements
[kubernetes/kubernetes]: https://git.k8s.io/kubernetes
[kubernetes/website]: https://git.k8s.io/website

## Summary

[Horizontal Pod Autoscaler][] (HPA) regularly checks metrics and calculates the desired replica count for a Deployment or another resource with a `/scale` subresource. This process is called reconciliation. Periodic reconciliations use the cluster-wide `--horizontal-pod-autoscaler-sync-period` value, which defaults to 15 seconds. Other events may trigger reconciliation sooner.

This proposal adds an optional `syncPeriodSeconds` field to `HorizontalPodAutoscalerSpec`. The field sets the sync period for one HPA. When it is unset, the HPA uses the global value.

[Horizontal Pod Autoscaler]: https://kubernetes.io/docs/tasks/run-application/horizontal-pod-autoscale/

## Motivation

Today, the [Kube Controller Manager][] sets one sync period for all HPAs with the `--horizontal-pod-autoscaler-sync-period` flag. The default is 15 seconds. One value does not work well for every HPA:

1. **Latency-sensitive workloads** need scaling decisions every few seconds to respond to traffic spikes.
2. **Stable workloads** can use the default or a longer period. Making the global period shorter increases load on the API server and metrics backends without helping these workloads.

The main use case is request-driven autoscaling based on external metrics. Providers such as [KEDA][], especially the [KEDA HTTP Add-on][], can return the current in-memory request count for each query. In this case, the 15-second sync period can be the main avoidable delay before the next scaling decision.

This matters most to teams moving from an autoscaler built for request-driven scaling. Knative Serving's Pod Autoscaler calculates the desired replica count every 2 seconds by default (`tick-interval` in its `config-autoscaler` ConfigMap). An HPA can reconcile that often only if every HPA in the cluster also reconciles more often. This makes HPA-based autoscaling harder to adopt for latency-sensitive workloads that scale from idle.

KEDA is tracking this gap in [kedacore/keda#7801][]. The issue suggests adaptive polling inside KEDA as a temporary workaround. This workaround can be removed when Kubernetes provides a stable per-HPA sync period.

Users requested this before this KEP; see [kubernetes#110317][]. Because different workloads need different sync periods, this KEP lets users set one for each `HorizontalPodAutoscaler`.

[Kube Controller Manager]: https://kubernetes.io/docs/reference/command-line-tools-reference/kube-controller-manager/
[KEDA]: https://keda.sh/
[KEDA HTTP Add-on]: https://github.com/kedacore/http-add-on
[kedacore/keda#7801]: https://github.com/kedacore/keda/issues/7801
[kubernetes#110317]: https://github.com/kubernetes/kubernetes/issues/110317

### Goals

- Allow users to set a sync period for each HPA.
- Enable HPAs with rapidly changing custom or external metrics to make scaling decisions every few seconds.
- Allow HPAs that do not need frequent checks to reconcile less often, reducing load on the control plane and metrics backends.
- Maintain full backward compatibility: existing HPAs without the new field continue to use the global sync period.

### Non-Goals

- Change the default value of the global `--horizontal-pod-autoscaler-sync-period` flag.
- Allow sub-second sync periods.
- Improve metric freshness or change how any metrics source collects metrics.
- Guarantee a hard real-time reconciliation deadline.

## Proposal

We propose a new field on [`HorizontalPodAutoscalerSpec`][]:

- `syncPeriodSeconds`: *(int32)* the delay in seconds after a reconciliation finishes. After this delay, the next periodic reconciliation of this HPA is eligible to run. Valid values are 3 to 3600 (one hour).

The field is optional. When it is unset, the HPA uses the global `--horizontal-pod-autoscaler-sync-period` value. Creating an HPA or changing its spec may enqueue it immediately, before this delay expires.

**Metrics freshness**: a shorter sync period does not refresh metrics. It helps only when the HPA can get a new metric value each time it reconciles.

For instance, [metrics-server][] collects CPU and memory metrics on its own schedule. The binary default is 60 seconds, with a minimum of 10 seconds ([options][ms-options]), and the provided manifests use 15 seconds ([manifests][ms-manifest]). If an HPA reconciles every 3 seconds, several reconciliations may therefore use the same CPU or memory data.

`External`, `Pods`, and `Object` metrics may behave differently. Adapters such as [KEDA][] can calculate them when the HPA requests them, so a shorter sync period can provide a faster response. This is the main use case for the field. The field documentation will explain this limitation.

`syncPeriodSeconds` applies to the whole HPA, not to individual metrics. An HPA can use several metric types at the same time. For this reason, validation does not limit the field to specific metric types, and the controller does not silently ignore it when resource metrics are present.

**Field placement**: `syncPeriodSeconds` is a top-level field in `HorizontalPodAutoscalerSpec`. Keeping it outside `spec.behavior` avoids defaulting scaling rules; see [Alternatives](#alternatives).

[`HorizontalPodAutoscalerSpec`]: https://kubernetes.io/docs/reference/generated/kubernetes-api/v1.35/#horizontalpodautoscalerspec-v2-autoscaling
[KEP-4951]: /keps/sig-autoscaling/4951-configurable-hpa-tolerance
[metrics-server]: https://github.com/kubernetes-sigs/metrics-server
[ms-options]: https://github.com/kubernetes-sigs/metrics-server/blob/master/cmd/metrics-server/app/options/options.go
[ms-manifest]: https://github.com/kubernetes-sigs/metrics-server/blob/master/manifests/base/deployment.yaml

### User Stories

#### Story 1

As a user of an HTTP autoscaler such as the [KEDA HTTP Add-on][], I want its HPAs to check request metrics every few seconds, so they can respond to a traffic spike sooner. The metrics adapter can provide a current request count on each query, but by default the HPA checks it only once every 15 seconds. I do not want to change the sync period for every HPA in the cluster.

#### Story 2

As a platform engineer managing a cluster shared by several teams, I want each team to choose a suitable sync period for its HPAs. Their workloads need different response times, but cluster administrators still need to enforce a minimum value and monitor whether the HPA controller is overloaded.

#### Story 3

As a cluster operator running many HPAs for batch workloads, I want those HPAs to reconcile less often because their metrics change slowly. This reduces metrics queries and API calls without slowing down HPAs that need a faster response. Today, I can only reduce the frequency for all HPAs at the same time.

### Risks and Mitigations

**Increased API server and metrics load.** Each HPA reconciliation gets the current scale of the target, makes one request for every configured metric, and may update the HPA status. A shorter period makes these calls more often. Compared with the default 15 seconds, an HPA reconciles 1.5 times as often at 10 seconds, 3 times as often at 5 seconds, and at most 5 times as often at the 3-second minimum.

Only HPAs with a period shorter than the global value generate extra traffic; longer periods reduce it. Ignoring reconciliation time and worker wait, 1,000 HPAs at the 15-second default produce about 67 reconciliations per second (`1000 / 15`). If 50 of them use a 5-second period, the estimate becomes 73 per second (`950 / 15 + 50 / 5`), an increase of roughly 10 percent. These are rate estimates, not benchmark results.

With one metric and a changed HPA status, each reconciliation makes three API requests:

- one scale `GET`
- one metrics request
- one status update

This gives about 200 requests per second for 1,000 HPAs at the 15-second default, and about 220 for the example with 50 HPAs at 5 seconds. In the extreme case, 1,000 HPAs at 3 seconds produce up to 333 reconciliations and about 1,000 requests per second. Each additional metric adds one request, and a scaling decision adds a scale update.

Controller-manager API clients default to 50 QPS with a burst of 100. The limit applies separately to each client, so it does not cap total HPA traffic. When a client reaches its limit, requests wait and the HPA may reconcile later than requested.

A shorter period does not change how often Prometheus scrapes existing metrics. The new delay-ratio histogram has only two fixed label values. However, more frequent status updates can increase API server and etcd writes. Storage effects in a metrics adapter or its backend depend on the provider.

The following safeguards reduce this risk:

- The feature gate is disabled by default in Alpha, and the field is optional for each HPA.
- The 3-second minimum supports metrics that change every few seconds while limiting one HPA to five times the default rate. It is an Alpha risk limit, not a guarantee that every cluster can support that rate. The minimum can be lowered later, while raising it could invalidate values already stored in the API.
- Administrators can require a higher minimum with a [ValidatingAdmissionPolicy][] or an admission webhook. A `ResourceQuota` can limit the total number of HPAs, but not specifically HPAs with short periods.

These controls do not calculate total demand or enforce an overall reconciliation limit for the cluster. Administrators must choose the minimum period and HPA quotas based on their cluster capacity.

For HPAs that use standard resource metrics, the documentation will recommend keeping the period at 15 seconds or longer. The following `autoscaling/v2` policy rejects shorter periods when any `Resource` or `ContainerResource` metric is present:

```yaml
apiVersion: admissionregistration.k8s.io/v1
kind: ValidatingAdmissionPolicy
metadata:
  name: hpa-resource-metric-sync-period
spec:
  failurePolicy: Fail
  matchConstraints:
    matchPolicy: Exact
    resourceRules:
    - apiGroups: ["autoscaling"]
      apiVersions: ["v2"]
      operations: ["CREATE", "UPDATE"]
      resources: ["horizontalpodautoscalers"]
  validations:
  - expression: >-
      !has(object.spec.syncPeriodSeconds) ||
      object.spec.syncPeriodSeconds >= 15 ||
      (size(object.spec.metrics) > 0 &&
       object.spec.metrics.all(
         m, m.type != "Resource" && m.type != "ContainerResource"))
    message: syncPeriodSeconds below 15 requires non-resource metrics
---
apiVersion: admissionregistration.k8s.io/v1
kind: ValidatingAdmissionPolicyBinding
metadata:
  name: hpa-resource-metric-sync-period
spec:
  policyName: hpa-resource-metric-sync-period
  validationActions: [Deny]
```

An `autoscaling/v1` HPA exposes CPU resource autoscaling, so an equivalent v1 policy should require `syncPeriodSeconds >= 15` whenever the field is set. The Alpha documentation will include both complete examples.

**Delays for other HPAs.** The HPA controller has a fixed worker pool, configured by `--concurrent-horizontal-pod-autoscaler-syncs` (default 5). If many HPAs use short periods, they can keep all workers busy. This delays other HPAs too, including those that use the default period.

The workqueue still keeps only one pending entry for each HPA, so one HPA cannot create a growing list of duplicate work. This does not protect the shared worker pool from too much total work.

Operators can detect this problem with the delay-ratio metric and the existing workqueue depth, queue duration, and reconciliation duration metrics. They can restrict short periods through admission policy, limit the total HPA count with a `ResourceQuota`, or increase `--concurrent-horizontal-pod-autoscaler-syncs`. Increasing the worker count should be done carefully because it also increases concurrent traffic to the API server and metrics adapters.

There is no single correct worker count. Administrators can estimate it from the expected reconciliation rate and the observed reconciliation duration, then allow some extra capacity. Changing the flag requires a controller-manager restart, so it should be planned before short periods are used broadly.

**Repeated use of stale metrics.** A short sync period does not make the metrics source update faster. If the source is slow, the HPA may calculate the same recommendation several times from the same data. Existing `behavior.scaleUp` and `behavior.scaleDown` policies still limit how quickly the replica count can change. Users should combine short periods with fresh metrics and suitable scaling policies.

Metric responses include timestamps, but the controller receives them only after querying the metrics API. Checking the timestamp therefore does not reduce query traffic.

The controller also cannot skip reconciliation only because the timestamp is unchanged. The replica count, pod readiness, another metric, or the stabilization window may have changed. If the resulting HPA status is unchanged, the controller already skips the status write.

[ValidatingAdmissionPolicy]: https://kubernetes.io/docs/reference/access-authn-authz/validating-admission-policy/

## Design Details

The `HorizontalPodAutoscaler` API adds `syncPeriodSeconds` to `HorizontalPodAutoscalerSpec`:

```golang
type HorizontalPodAutoscalerSpec struct {
  // syncPeriodSeconds is the delay after a reconciliation completes before
  // the next periodic reconciliation of this HPA is eligible to run.
  // Other events may trigger reconciliation before the delay expires.
  // When unset, the global --horizontal-pod-autoscaler-sync-period is used.
  // The value must be in the range [3, 3600]. Timing is best-effort.
  // +featureGate=HPAConfigurableSyncPeriod
  // +optional
  // +k8s:minimum=3
  // +k8s:maximum=3600
  SyncPeriodSeconds *int32
}
```

**Scheduling behavior.** The delay starts when a reconciliation finishes. The time between two reconciliation starts therefore includes the previous reconciliation, the configured delay, and any time spent waiting for a worker. Other controller events may trigger reconciliation before the delay expires. The workqueue keeps only one pending entry for each HPA.

**Workqueue implementation.** A new `PerItemIntervalRateLimiter` provides a per-HPA delay with the global period as its fallback. The controller refreshes the delay from the current HPA during reconciliation and before handling an informer update. It removes the override when the field is unset, the feature gate is disabled, or the HPA is deleted. After each reconciliation, `processNextWorkItem` uses the existing `AddRateLimited` call to schedule the next one.

The controller also records the completion time, requested delay, and whether the delay came from the global setting or the HPA spec for each pending periodic requeue. This state provides the delay-ratio histogram described in [Monitoring Requirements](#monitoring-requirements) and is removed when the HPA is deleted. Event-driven reconciliations are not included in the histogram.

**Updating the period.** Refreshing the delay in the update handler makes a new value take effect without depending on the `HPAGeneration` feature gate. Queue behavior otherwise stays the same. Creations always use `queue.Add`; spec changes also use it when `HPAGeneration` is enabled. Other updates use `AddRateLimited`, which prevents status updates from creating the hot loop fixed by [#42715][].

The delaying workqueue can move a waiting item earlier, but not later:

- If the period becomes shorter, the waiting item moves earlier when needed. It may run sooner if it was already scheduled before the new deadline.
- If the period becomes longer, one reconciliation may still run at the old delay. Later reconciliations use the new delay.

An informer resync refreshes the stored delay if the normal update event is missed. With `HPAGeneration` enabled, a period change also causes immediate reconciliation, like any other spec change.

**API versions and validation.** The field is added to the internal type and to both served versions, `autoscaling/v1` and `autoscaling/v2`. Conversion copies the value directly, so clients preserve it when reading and writing through either version, as required by the [round-trip requirements][]. Declarative validation tags enforce the range `[3, 3600]` in both versions.

**Feature gate behavior.** While `HPAConfigurableSyncPeriod` is disabled, the API server drops the field from new objects but preserves it when an existing object already contains it. This prevents an unrelated update from removing a value written while the gate was enabled. The controller ignores a stored value while its gate is disabled and uses the global period instead. This follows the same pattern as `HPAConfigurableTolerance` ([KEP-4951][]). The disabled-field helper is updated to handle each gated field independently.

**Interaction with existing HPA behavior.** A shorter period adds more recommendations to the same stabilization window. The stabilization algorithm does not change, but having more samples can change which recommendation it selects. Existing pod readiness and CPU initialization rules are also unchanged; a shorter period does not bypass them.

[#42715]: https://github.com/kubernetes/kubernetes/pull/42715
[round-trip requirements]: https://kubernetes.io/docs/reference/using-api/deprecation-policy/#deprecating-parts-of-the-api

### Test Plan

[x] I/we understand the owners of the involved components may require updates to existing tests to make this code solid enough prior to committing the changes necessary to implement this enhancement.

##### Prerequisite testing updates

No prerequisite changes are required. Unit tests will create the workqueue with the fake clock already supported by client-go. This allows them to advance time without sleeping.

##### Unit tests

- `pkg/apis/autoscaling/validation`: 2026-08-31 - 95.3% of statements
- `pkg/controller/podautoscaler`: 2026-08-31 - 89.0% of statements
- `pkg/registry/autoscaling/horizontalpodautoscaler`: 2026-08-31 - 56.9% of statements

Unit tests will cover:

- API and feature-gate behavior: both API versions accept values in `[3, 3600]` and reject values outside that range. When the gate is disabled, the field is dropped on create and preserved on update if it was already set.
- Rate limiting and scheduling: the per-HPA delay overrides the global value and is removed when the field is unset or the HPA is deleted. Fake-clock tests cover the fixed delay and changes to shorter and longer periods.
- Informer updates: the handler refreshes the delay after normal updates and resyncs, with `HPAGeneration` enabled and disabled. Status-only updates remain rate-limited.
- Metrics: the first periodic reconciliation and event-driven reconciliations are ignored. Later periodic reconciliations update the correct histogram buckets and `period_source` label. Deleting the HPA removes its internal timing state.

##### Integration tests

Integration tests run the real API server and HPA controller with a real clock. Exact queue timing is covered by the unit tests above. Integration tests will verify that:

- `syncPeriodSeconds` survives storage and conversion between `autoscaling/v1` and `autoscaling/v2`.
- An HPA with the field set uses its own period, while an HPA without it uses the global period.
- An HPA that already contains the field uses the global period when the controller feature gate is disabled.

##### e2e tests

Existing HPA e2e tests exercise HPAs without `syncPeriodSeconds`, but do not test a custom period. One new e2e scenario will create three separate workloads with a short period, the default period, and a long period. The test will use a controlled metric source and verify each HPA against a broad expected time window rather than require strict ordering between workloads.

### Graduation Criteria

#### Alpha

- The API server load estimate and the 3-second minimum in [Risks and Mitigations](#risks-and-mitigations) have been shared with SIG Scalability.
- The feature is implemented behind the `HPAConfigurableSyncPeriod` feature gate.
- The unit and integration tests described above are implemented and enabled.
- The scheduling-delay histogram is available, so operators can tell whether configured delays are being met for HPAs using the global period and those using `syncPeriodSeconds`.
- The e2e scenario described in the [`e2e tests` section](#e2e-tests), covering both shorter and longer periods than the default, is implemented and enabled.
- Documentation is drafted in [kubernetes/website]. It describes the field, its best-effort timing, when a shorter period is useful, and example admission policies for resource-metric HPAs in `autoscaling/v1` and `autoscaling/v2`.

#### Beta

- Feedback from Alpha users is reviewed and blocking issues are resolved.
- Before Beta, the KEP authors record a summary in the KEP tracking issue. It covers delay-ratio observations from opt-in clusters, early adopters such as the KEDA HTTP Add-on and various KEDA scalers, and other user feedback. The summary is used to decide whether to lower the 3-second minimum or add controls for worker contention. Any significant control-plane impact is shared with SIG Scalability.
- Monitoring guidance is updated using Alpha data, including a numeric delay-ratio threshold if the data supports one.
- The e2e test implemented in Alpha is linked in this KEP and has run without non-infrastructure flakes for at least two weeks.
- The upgrade, downgrade, and re-enable procedure is described in detail in this KEP, and its test results are recorded.
- Documentation is published on kubernetes.io. It explains when the field is useful and how it interacts with `--concurrent-horizontal-pod-autoscaler-syncs`.
- The `HPAConfigurableSyncPeriod` feature gate is enabled by default.

#### GA

- The feature has remained Beta for at least two releases with no major unresolved issues. This provides at least one full release to observe the lower-bound decision and worker contention before the gate is locked at GA.
- The `HPAConfigurableSyncPeriod` feature gate is locked to enabled and deprecated at GA. It is removed only after the deprecation period required by [Rule #9][].
- Documentation is reviewed and updated for GA.

[Rule #9]: https://kubernetes.io/docs/reference/using-api/deprecation-policy/

### Upgrade / Downgrade Strategy

#### Upgrade
Existing HPAs continue to work as they do today, using the global `--horizontal-pod-autoscaler-sync-period` value. During Alpha, administrators enable `HPAConfigurableSyncPeriod` on every `kube-apiserver` and `kube-controller-manager` instance. They should finish the rolling control-plane upgrade before users set `syncPeriodSeconds` on an HPA.

#### Downgrade
After a downgrade, all HPAs revert to the global `--horizontal-pod-autoscaler-sync-period` value, regardless of any configured `syncPeriodSeconds` on the HPA itself. An older API server may remove the unknown field if the HPA is updated. Users should reapply the HPA manifest after upgrading again.

### Version Skew Strategy

1. `kube-apiserver`: more recent instances accept `syncPeriodSeconds`; older instances do not recognize it. During a rolling control-plane upgrade an older API server may reject the field under strict field validation or prune it otherwise, so a client that reads, modifies, and writes back an HPA may lose a previously set value until every API server is upgraded. The effect is a return to the global sync period for that HPA; re-applying the manifest after the upgrade restores it.
2. `kube-controller-manager`: an older controller ignores the field even when a newer API server provides it. A newer controller connected to an older API server receives the HPA without the field. In both cases, the controller uses the global period.

## Production Readiness Review Questionnaire

### Feature Enablement and Rollback

###### How can this feature be enabled / disabled in a live cluster?

- [x] Feature gate (also fill in values in `kep.yaml`)
  - Feature gate name: HPAConfigurableSyncPeriod
  - Components depending on the feature gate: `kube-controller-manager` and `kube-apiserver`.

###### Does enabling the feature change any default behavior?

No. An HPA without `syncPeriodSeconds` continues to use the global period.

###### Can the feature be disabled once it has been enabled (i.e. can we roll back the enablement)?

Yes, by restarting the `kube-apiserver` and `kube-controller-manager` with the gate disabled. The API server drops the field from new objects, and the controller uses the global period for every HPA. Values already stored in existing objects are preserved.

###### What happens if we reenable the feature if it was previously rolled back?

HPAs with a stored `syncPeriodSeconds` use it again. Manifests do not need to be reapplied.

###### Are there any tests for feature enablement/disablement?

Unit tests will cover dropping and preserving the field. An integration test will cover an existing HPA using the global period while the controller gate is disabled.

### Rollout, Upgrade and Rollback Planning

###### How can a rollout or rollback fail? Can it impact already running workloads?

Enabling the gate alone does not affect running workloads. If users set `syncPeriodSeconds` before the rolling upgrade finishes, older API servers may reject or remove the field, and older controller managers ignore it.

After rollback, HPAs that used the field return to the global period and may react more slowly. They still calculate replicas from the same metrics and algorithm.

###### What specific metrics should inform a rollback?

- A sustained shift toward larger values in `horizontal_pod_autoscaler_controller_reconciliation_delay_ratio`, introduced by this feature, compared with a baseline recorded after installing the new binaries and enabling the gate but before HPAs begin using `syncPeriodSeconds`.
- Growth in the existing `workqueue_depth` or `workqueue_queue_duration_seconds` metrics for the HPA controller.
- Growth in the existing `horizontal_pod_autoscaler_controller_reconciliation_duration_seconds` or `horizontal_pod_autoscaler_controller_metric_computation_duration_seconds` metrics.
- API server errors reported by `apiserver_request_total`, and adapter-specific request-error metrics where available.

###### Were upgrade and rollback tested? Was the upgrade->downgrade->upgrade path tested?

Not yet because the Alpha implementation does not exist. Unit and integration tests cover feature-gate changes. Before Beta, the full path will also be tested on a local cluster:

1. Enable the gate and verify that an HPA uses its configured period.
2. Disable the gate and verify that the value is preserved but ignored.
3. Enable the gate again and verify that the value is used without reapplying the HPA.

The results will be recorded here before the Beta graduation review.

###### Is the rollout accompanied by any deprecations and/or removals of features, APIs, fields of API types, flags, etc.?

No.

### Monitoring Requirements

###### How can an operator determine if the feature is in use by workloads?

The presence of `syncPeriodSeconds` shows that an HPA is configured to use the feature. Controller logs show the effective and observed delay for a specific HPA.

###### How can someone using this feature know that it is working for their instance?

- [ ] Metrics
- [ ] Events
- [X] Other
  - Controller logs at verbosity >= 4 include the configured and observed scheduling delay for each HPA.

A new histogram, `horizontal_pod_autoscaler_controller_reconciliation_delay_ratio{period_source}`, records the observed delay divided by the requested delay for every periodic reconciliation. Measurement starts when one reconciliation finishes and ends when the next periodic reconciliation starts, so reconciliation time is excluded. A value near 1 is on time; a value above 1 is late.

The `period_source` label has two values: `global` when the HPA uses the controller-wide period and `spec` when it uses `syncPeriodSeconds`. Using only these two values limits the number of metric series while allowing operators to compare the two groups. Proposed buckets are `1`, `1.1`, `1.25`, `1.5`, `2`, `3`, `5`, and `10`. The first periodic reconciliation and event-driven reconciliations are ignored. Per-HPA debugging uses controller logs instead of metric labels. Access in a managed control plane depends on the provider exposing controller-manager metrics and logs.

###### What are the reasonable SLOs (Service Level Objectives) for the enhancement?

There is no existing HPA scheduling SLO, so Alpha does not define a numeric threshold. The histogram distribution should stay close to 1 and should not regress from the cluster's baseline. Alpha data will be used to improve the guidance for Beta.

Enabling the gate without setting the field should not affect reconciliation duration.

###### What are the SLIs (Service Level Indicators) an operator can use to determine the health of the service?

- The new `horizontal_pod_autoscaler_controller_reconciliation_delay_ratio` histogram shows the scheduling-delay distribution for HPAs using the global period and those using the field.
- The existing `workqueue_depth` and `workqueue_queue_duration_seconds` metrics show whether the worker pool is keeping up.
- The existing `horizontal_pod_autoscaler_controller_reconciliation_duration_seconds` and `horizontal_pod_autoscaler_controller_metric_computation_duration_seconds` metrics help identify slow controller or metrics-adapter operations.

###### Are there any missing metrics that would be useful to have to improve observability of this feature?

No. The new delay-ratio histogram is part of the Alpha implementation.

### Dependencies

###### Does this feature depend on any specific services running in the cluster?

No. `HPAGeneration` is not required; see [Design Details](#design-details).

### Scalability

###### Will enabling / using this feature result in any new API calls?

No new call types. A lower `syncPeriodSeconds` increases the frequency of existing calls for that HPA. The cost of that increase is discussed in [Risks and Mitigations](#risks-and-mitigations).

###### Will enabling / using this feature result in introducing new API types?

No.

###### Will enabling / using this feature result in any new calls to the cloud provider?

No.

###### Will enabling / using this feature result in increasing size or count of the existing API objects?

One optional int32 field is added to each HPA object. No new objects are created.

###### Will enabling / using this feature result in increasing time taken by any operations covered by existing SLIs/SLOs?

No individual operation changes. More reconciliations overall can increase queueing delays for all HPAs; see [Risks and Mitigations](#risks-and-mitigations).

###### Will enabling / using this feature result in non-negligible increase of resource usage (CPU, RAM, disk, IO, ...) in any components?

Lower values increase CPU, network traffic, and API calls in the `kube-controller-manager`, API server, and metrics adapter for the affected HPAs.

###### Can enabling / using this feature result in resource exhaustion of some node resources (PIDs, sockets, inodes, etc.)?

No node-level exhaustion is expected. The fixed worker count bounds concurrent calls from the HPA controller, although control-plane resource use can increase.

### Troubleshooting

###### How does this feature react if the API server and/or etcd is unavailable?

No feature-specific impact. The controller cannot reconcile regardless of sync period configuration while the API server is unavailable.

###### What are other known failure modes?

Too many HPAs with short periods can keep all controller workers busy and delay every HPA. Operators can increase the configured periods or remove the field. They can also increase `--concurrent-horizontal-pod-autoscaler-syncs` if the API server and metrics adapters can handle the additional concurrency.

###### What steps should be taken if SLOs are not being met to determine the problem?

1. Check the delay-ratio and workqueue metrics.
2. Check reconciliation and metric computation duration.
3. Check API server and metrics-adapter latency and errors.
4. Check the `kube-controller-manager` logs.

## Implementation History

- 2022-06-01: [kubernetes#110317][] opened, requesting a per-HPA sync period. This is the original request that led to this KEP.
- 2026-04-08: Initial KEP created.
- 2026-08-19: KEP updated to target Alpha in v1.38.

## Drawbacks

Adding this field to the stable `autoscaling/v1` and `autoscaling/v2` APIs creates a long-term commitment. Once released, the field must continue to round-trip correctly, and its documented behavior cannot be removed or significantly changed.

Users may set a short period for CPU or memory metrics and expect faster scaling, even when metrics-server does not refresh that often. This adds controller and API traffic without improving response time, so the documentation must explain when the field is useful.

## Alternatives

- **Change the global flag**: affects every HPA and requires cluster-admin access and a controller-manager restart.
- **Use a per-HPA annotation**: rejected in favor of a validated, documented, and discoverable API field.
- **Put `syncPeriodSeconds` under `spec.behavior`**: rejected because creating a `behavior` block causes Kubernetes to fill in and store all default scaling rules. If the feature gate is disabled, the sync period would be removed but the defaulted `behavior` block would remain. This would create unexpected object changes and GitOps diffs. `HPAConfigurableTolerance` is different because tolerance modifies an existing scaling rule.
- **Allow the field only with custom or external metrics**: rejected because one HPA can use several metric types, while the period applies to the whole reconciliation.
- **Use fixed-rate scheduling**: rejected because the existing HPA loop waits after each reconciliation. With fixed-rate scheduling, a slow reconciliation could cause the next one to start immediately and create a continuous loop.
- **Choose the period automatically**: an adaptive controller could run faster during a traffic spike and slower when stable. This is harder to predict and test, and does not give users explicit per-HPA control.

## Infrastructure Needed (Optional)

N/A.
