# KEP-4951: Configurable tolerance for HPA

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
    - [Beta](#beta)
    - [Stable](#stable)
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
- [x] (R) Test plan is in place, giving consideration to SIG Architecture and SIG Testing input (including test refactors)
  - [x] e2e Tests for all Beta API Operations (endpoints)
  - [x] (R) Ensure GA e2e tests meet requirements for [Conformance Tests](https://github.com/kubernetes/community/blob/master/contributors/devel/sig-architecture/conformance-tests.md)
  - [x] (R) Minimum Two Week Window for GA e2e tests to prove flake free
- [x] (R) Graduation criteria is in place
  - [x] (R) [all GA Endpoints](https://github.com/kubernetes/community/pull/1806) must be hit by [Conformance Tests](https://github.com/kubernetes/community/blob/master/contributors/devel/sig-architecture/conformance-tests.md)
- [x] (R) Production readiness review completed
- [x] (R) Production readiness review approved
- [x] "Implementation History" section is up-to-date for milestone
- [x] User-facing documentation has been created in [kubernetes/website], for publication to [kubernetes.io]
- [x] Supporting documentation—e.g., additional design documents, links to mailing list discussions/SIG meetings, relevant PRs/issues, release notes

[kubernetes.io]: https://kubernetes.io/
[kubernetes/enhancements]: https://git.k8s.io/enhancements
[kubernetes/kubernetes]: https://git.k8s.io/kubernetes
[kubernetes/website]: https://git.k8s.io/website

## Summary

[Horizontal Pod Autoscaler][] (HPA) regularly estimates how many replicas a given Deployment (or other resource with a `/scale` subresource) should instantiate.
HPAs define one (or more) metrics (e.g. CPU utilization) on which autoscaling is based. The number of replicas is derived from the ratio between the *expected* and *current* value of this metric ([Algorithm details][]).

For example, for a workload with 100 `currentReplicas` and a usage ratio
(`currentMetricValue`/`desiredMetricValue`) of 1.07, the calculated `desiredReplicas`
would be 107  (100 * 1.07).

However, to avoid flapping, scaling actions are skipped if the usage ratio is approximately 1, within a
globally-configurable *tolerance*,  set to 10% by default. In the example above, no scaling action would
take place, since the ratio is within this tolerance.

This proposal adds a parameter to HPAs allowing users to configure this tolerance per HPA resource.
For the example above, we could configure the tolerance in the workload's HPA to 5%, which would
allow the scale-up to 107 replicas to proceed.

[Horizontal Pod Autoscaler]: https://kubernetes.io/docs/tasks/run-application/horizontal-pod-autoscale/
[Algorithm details]: https://kubernetes.io/docs/tasks/run-application/horizontal-pod-autoscale/#algorithm-details

## Motivation

Today the horizontal autoscaling tolerance is a cluster-wide parameter set using the [Kube Control Manager][] `--horizontal-pod-autoscaler-tolerance` parameter. It is by default set to 10%. While this value is often appropriate, it is considered too coarse grained in a number of scenario.

This issue has been raised multiple times ([#116984][], [#125987][], [#62013][],
[#aks-3068][], [#keda-1100][]), with users commenting that:

1. For large deployments, a 10% tolerance translates into very significant resources (i.e. hundreds of pods).
2. This tolerance can slow down scaling operations, hindering responsiveness in
case of surges.
3. Scale-ups are more a problem than scale-downs since typically pods are slower to initialize than to shut down, and since responding to load increase is typically more critical than freeing resources.

Since appropriate tolerance values are workload-dependent, this KEP proposes to let users add custom tolerance values to `HorizontalPodAutoscaler` resources, overriding the existing default value when present.

This solution integrates seamlessly with the existing HPA API since it already allows users to [fine-tune the autoscaler behavior](https://kubernetes.io/docs/tasks/run-application/horizontal-pod-autoscale/#configurable-scaling-behavior).
The exact API recommended here has been previously proposed in [kep-853][] (see [here](https://github.com/kubernetes/enhancements/pull/1234#discussion_r333036990)), but it was then decided to implement it separately.

[#116984]: https://github.com/kubernetes/kubernetes/issues/116984
[#125987]: https://github.com/kubernetes/kubernetes/issues/125987
[#62013]: https://github.com/kubernetes/kubernetes/issues/62013
[#aks-3068]: https://github.com/Azure/AKS/issues/3068
[#keda-1100]: https://github.com/kedacore/keda-docs/issues/1100
[Kube Control Manager]: https://kubernetes.io/docs/reference/command-line-tools-reference/kube-controller-manager/
[kep-853]: https://github.com/kubernetes/enhancements/blob/master/keps/sig-autoscaling/853-configurable-hpa-scale-velocity/README.md

### Goals

- Allow users to optionally override the default workload autoscaling tolerance on a per-HPA bases.

### Non-Goals

- Allow to customize the cluster-wise tolerance given by Kube Control Manager `--horizontal-pod-autoscaler-tolerance` parameter.


## Proposal

We propose to add a new field to the existing [`HPAScalingRules`][] object:

- `tolerance`: (float) the minimum change (from 1.0) in the desired-to-actual metrics ratio for the horizontal pod autoscaler to consider scaling. Must be greater than or equal to 0.

The `tolerance` field is optional, and when not specified the HPA will continue to use the
value of the global `--horizontal-pod-autoscaler-tolerance` as the tolerance for scaling
calculations.

Since there are separate `HPAScalingRules` objects defined for an HPA's
`spec.behavior.scaleUp` and `spec.behavior.scaleDown`, it is possible to specify different
`tolerance` values for scaling up vs. scaling down.

[HPAScalingRules]: https://kubernetes.io/docs/reference/generated/kubernetes-api/v1.31/#hpascalingrules-v2-autoscaling

### Risks and Mitigations

There should be minimal risk introduced by the proposed changes:
- The new field is optional, and its absence results in no changes to the current autoscaling behavior
- When specified, the new value doesn't change the autoscaling algorithm used, but just overrides a single value used during the calculation. This value can already be changed via the `--horizontal-pod-autoscaler-tolerance` option of the `kube-controller-manager`.
- If a change to the new field results in undesirable behavior, the change can be reverted by deploying the previous version of the HPA resource, or removing the `tolerance` field entirely.

## Design Details

The `HorizontalPodAutoscaler` API is updated to add a new `tolerance` field to the `HPAScalingRules` object:

```golang
type HPAScalingRules struct {
  // tolerance is the tolerance on the ratio between the current and desired
  // metric value under which no updates are made to the desired number of
  // replicas.
  // +optional
  Tolerance *resource.Quantity

  // Existing fields.
  StabilizationWindowSeconds *int32
  SelectPolicy *ScalingPolicySelect
  Policies []HPAScalingPolicy
}
```

This new tolerance will be used in the autoscaling controller
[replica_calculator.go][]. The current logic is:

```golang
if math.Abs(1.0-usageRatio) <= c.tolerance { /* ... */ }
```

It will be replaced by:

```diff
- if math.Abs(1.0-usageRatio) <= c.tolerance { /* ... */ }
+ // Down and Up scaling tolerances default to c.tolerance if unset.
+ downTolerance, upTolerance := c.tolerance, c.tolerance
+ if scaleDown.tolerance != nil {
+   downTolerance = scaleDown.tolerance.AsApproximateFloat64()
+ }
+ if scaleUp.tolerance != nil {
+   upTolerance = scaleUp.tolerance.AsApproximateFloat64()
+ }
+
+ if (1.0-downTolerance) <= usageRatio && usageRatio <= (1.0+upTolerance) { /* ... */ }
```

Since the added field is optional and its omission does not change the existing
autoscaling behavior, this feature will only be added to the latest stable API
version `pkg/apis/autoscaling/v2`. Older versions (i.e. `v1`, `v2beta1`,
`v2beta2`) will not include the new field, but converters will be updated where
needed to comply with [round-trip requirements][].

The feature presented in this KEP only allows users to tune an existing parameter, and
as such doesn't require any new HPA Events or modify any Status. The validation logic
will be updated to ensure that the `tolerance` field cannot be set to a negative value.

[replica_calculator.go]: https://github.com/kubernetes/kubernetes/blob/master/pkg/controller/podautoscaler/replica_calculator.go
[round-trip requirements]: https://kubernetes.io/docs/reference/using-api/deprecation-policy/#deprecating-parts-of-the-api

### Test Plan

[x] I/we understand the owners of the involved components may require updates to
existing tests to make this code solid enough prior to committing the changes necessary
to implement this enhancement.

##### Prerequisite testing updates

##### Unit tests

- `/apis/autoscaling/validation`: `2024-11-13` - `95.6`
- `/pkg/controller/podautoscaler`: `2024-11-13` - `96.4`

##### Integration tests

A test will be added to the [podautoscaler integration tests] to ensure HPA
configurable tolerances are correctly taken into account both when scaling up
and down.

- [Test HPA with tolerance](https://github.com/kubernetes/kubernetes/tree/master/test/integration/podautoscaler) ([Testgrid](https://testgrid.k8s.io/sig-release-master-blocking#integration-master&width=20&include-filter-by-regex=sig-autoscaling) [triage search](https://storage.googleapis.com/k8s-triage/index.html?job=ci-kubernetes-integration&test=sig-autoscaling))

[podautoscaler integration tests]: https://github.com/kubernetes/kubernetes/tree/master/test/integration/podautoscaler

##### e2e tests

Existing e2e tests ensure the autoscaling behavior uses the default tolerance when no
configurable tolerance is specified.

The new [e2e autoscaling tests] covering this feature are:

- [Test with large configurable tolerance](https://github.com/kubernetes/kubernetes/blob/07142400ecd02126602ffaa6f91712cd3f1e170c/test/e2e/autoscaling/horizontal_pod_autoscaling_behavior.go#L509): [SIG autoscaling](https://testgrid.k8s.io/sig-autoscaling-hpa-presubmits#gci-gce-autoscaling-hpa-cpu-alpha-beta-pull&include-filter-by-regex=HPAConfigurableTolerance.*large%20configurable%20tolerance), [triage search](https://storage.googleapis.com/k8s-triage/index.html?test=HPAConfigurableTolerance.*large%20configurable%20tolerance)
- [Test with both larger and smaller than default tolerances](https://github.com/kubernetes/kubernetes/blob/96914184bc3fe6d3c0195b5828c07a821ca6e093/test/e2e/autoscaling/horizontal_pod_autoscaling_behavior.go#L549): [SIG autoscaling](https://testgrid.k8s.io/sig-autoscaling-hpa-presubmits#gci-gce-autoscaling-hpa-cpu-alpha-beta-pull&include-filter-by-regex=HPAConfigurableTolerance.*small.*tolerance), [triage search](https://storage.googleapis.com/k8s-triage/index.html?test=HPAConfigurableTolerance.*small.*tolerance)

[e2e autoscaling tests]: https://github.com/kubernetes/kubernetes/tree/master/test/e2e/autoscaling

### Graduation Criteria

#### Alpha

- Feature implemented behind a `HPAConfigurableTolerance` feature flag
- Initial e2e tests completed and enabled

#### Beta

- All tests described in the [`e2e tests` section](#e2e-tests) are implemented
  and linked in this KEP.
- We have monitored for negative user feedback and addressed relevant concerns.

#### Stable

- Observe real-world usage.
- We have monitored for negative user feedback and addressed relevant concerns.

### Upgrade / Downgrade Strategy

#### Upgrade
Existing HPAs will continue to work as they do today, using the global `horizontal-pod-autoscaler-tolerance`
value from the `kube-controller-manager`. Users can use the new feature by enabling the Feature
Gate (alpha only) and setting the new `tolerance` field on an HPA.

#### Downgrade
On downgrade, all HPAs will revert to using the global `horizontal-pod-autoscaler-tolerance`
value from the `kube-controller-manager`, regardless of any configured `tolerance` value on the HPA
itself.

### Version Skew Strategy

1. `kube-apiserver`: More recent instances will accept the new 'tolerance'
   field, while older will ignore it.
2. `kube-controller-manager`: An older version could receive an HPA containing
   the new `tolerance` field from a more recent API server, in which case it
   would ignore it (i.e. scale as if it was not present).

## Production Readiness Review Questionnaire

### Feature Enablement and Rollback

###### How can this feature be enabled / disabled in a live cluster?

- [x] Feature gate (also fill in values in `kep.yaml`)
  - Feature gate name: HPAConfigurableTolerance
  - Components depending on the feature gate: `kube-controller-manager` and
    `kube-apiserver`.

###### Does enabling the feature change any default behavior?

No.

###### Can the feature be disabled once it has been enabled (i.e. can we roll back the enablement)?

The feature cannot be disabled: `tolerance` values set on HPAs are taken into
account.

Users can remove `tolerance` fields to ensure the feature is not used.

###### What happens if we reenable the feature if it was previously rolled back?

When the feature is re-enabled, any HPAs with configured `tolerance` values will use those when calculating replica counts, rather than the global tolerance from the `kube-controller-manager`.

###### Are there any tests for feature enablement/disablement?

[Unit tests have been added](https://github.com/kubernetes/kubernetes/blob/07142400ecd02126602ffaa6f91712cd3f1e170c/pkg/apis/autoscaling/validation/validation_test.go#L1648) to verify that HPAs with and without the new fields are
properly validated, both when the feature gate is enabled or not.

### Rollout, Upgrade and Rollback Planning

###### How can a rollout or rollback fail? Can it impact already running workloads?

This feature does not introduce new failure modes: during rollout/rollback, some
API servers will allow or disallow setting the new 'tolerance' field. The new
field is possibly ignored until the controller manager is fully updated.

###### What specific metrics should inform a rollback?

A high `horizontal_pod_autoscaler_controller_metric_computation_duration_seconds`
metric can indicate a problem related to this feature.

###### Were upgrade and rollback tested? Was the upgrade->downgrade->upgrade path tested?

The upgrade→downgrade→upgrade testing was done manually using a 1.33 cluster with the following steps:

1.  Start the cluster with the HPA enabled:

    ```sh
    kind create cluster --name configurable-tolerance --image kindest/node:v1.33.0 --config config.yaml
    ```
    with the following `config.yaml` file content:
    ```yaml
    kind: Cluster
    apiVersion: kind.x-k8s.io/v1alpha4
    featureGates:
      "HPAConfigurableTolerance": true
    nodes:
    - role: control-plane
    - role: worker
    ```

    Install metrics-server:
    ```sh
    kubectl apply -f https://github.com/kubernetes-sigs/metrics-server/releases/download/v0.7.2/components.yaml
    kubectl patch -n kube-system deployment metrics-server --type=json   -p '[{"op":"add","path":"/spec/template/spec/containers/0/args/-","value":"--kubelet-insecure-tls"}]'
    ```
    
    Create a deployment starting Pods that consume a 50% CPU utilization, and an associated HPA with a very large tolerance:
    ```sh
    kubectl apply -f configurable-tolerance-test.yaml
    ```
    with the following `configurable-tolerance-test.yaml` file content:
    ```yaml
    apiVersion: apps/v1
    kind: Deployment
    metadata:
      name: cpu-stress-deployment
      labels:
        app: cpu-stressor
    spec:
      replicas: 1
      selector:
        matchLabels:
          app: cpu-stressor
      template:
        metadata:
          labels:
            app: cpu-stressor
        spec:
          containers:
            - name: cpu-stressor
              image: alpine:latest
              command: ["/bin/sh"]
              args:  # Load: 1% (10 milliCPU)
                - "-c"
                - "apk add --no-cache stress-ng && stress-ng --cpu 1 --cpu-load 1 --cpu-method=crc16 --timeout 3600s"
              resources:
                requests:
                  cpu: "20m"
    ---
    apiVersion: autoscaling/v2
    kind: HorizontalPodAutoscaler
    metadata:
      name: cpu-stress-hpa
    spec:
      scaleTargetRef:
        apiVersion: apps/v1
        kind: Deployment
        name: cpu-stress-deployment
      minReplicas: 1
      maxReplicas: 5
      metrics:
        - type: Resource
          resource:
            name: cpu
            target:
              type: Utilization
              averageUtilization: 10
      behavior:
        scaleUp:
          tolerance: 20.  # 2000%
    ```

    Check that, after a 5 minutes, `kubectl describe hpa cpu-stress-hpa` displays `ScalingLimited: False` (i.e.
    the HPA doesn't recommend to scale up because of the large tolerance).

2.  Simulate downgrade by disabling the feature for api server and control-plane (update the `config.yaml` file
    to set it to false). Follow the procedure described in step 1, and observe that this time
    `kubectl describe hpa cpu-stress-hpa` displays `ScalingLimited: True`.

4. Simulate downgrade by re-enabling the feature for api server and control-plane. Follow the procedure described
   in step 1, and observe that the HPA description mentions `ScalingLimited: False`, demonstrates that the feature
   is working again.

###### Is the rollout accompanied by any deprecations and/or removals of features, APIs, fields of API types, flags, etc.?

No.

### Monitoring Requirements

###### How can an operator determine if the feature is in use by workloads?

The presence of the new `tolerance` HPA field indicates that the feature is
used.

###### How can someone using this feature know that it is working for their instance?

- [X] Events
  - Event Reason: `SuccessfulRescale`

The tolerance is applied on the ratio between the _current_ and _desired_ metric
values. Users can get both values using
[`kubectl describe`](https://github.com/kubernetes/kubernetes/blob/1b7a0591871772fbbc0fda430b3b73bc24c0e738/staging/src/k8s.io/kubectl/pkg/describe/describe.go#L4109)
and use them to verify that scaling events are triggered when their ratio is out
of tolerance.

The [controller-manager logs have been updated](https://github.com/kubernetes/kubernetes/blob/07142400ecd02126602ffaa6f91712cd3f1e170c/pkg/controller/podautoscaler/horizontal.go#L846) 
to help users understand the behavior of the autoscaler. The data added to the
logs includes the tolerance used for each scaling decision.

###### What are the reasonable SLOs (Service Level Objectives) for the enhancement?

Although the absolute value of the `horizontal_pod_autoscaler_controller_metric_computation_duration_seconds`
metric depends on HPAs configuration, it should be unimpacted by this feature. This metric should not vary
by more than 5%.

###### What are the SLIs (Service Level Indicators) an operator can use to determine the health of the service?

This KEP is not expected to have any impact on SLIs/SLOs as it doesn't introduce
a new HPA behavior, but merely allows users to easily change the value of a
parameter that's otherwise difficult to update.

The standard HPA metric `horizontal_pod_autoscaler_controller_metric_computation_duration_seconds` can
be used to verify the HPA controller health.

###### Are there any missing metrics that would be useful to have to improve observability of this feature?

Users may want to see a signal that autoscaling isn't happening because of the
tolerance, but this is not directly related to this KEP (this problem already
exists today with the hard-coded 10% tolerance), and taking this KEP as an
opportunity to improve the situation is difficult (see
[this thread](https://github.com/kubernetes/enhancements/pull/4954#discussion_r1857098884)).

### Dependencies

###### Does this feature depend on any specific services running in the cluster?

No, this feature does not depend on any specific service.

### Scalability

###### Will enabling / using this feature result in any new API calls?

No.

###### Will enabling / using this feature result in introducing new API types?

No.

###### Will enabling / using this feature result in any new calls to the cloud provider?

No.

###### Will enabling / using this feature result in increasing size or count of the existing API objects?

- This feature adds two new optional integer fields to `HorizontalPodAutoscaler`
  `v2` objects. Users should expect this object to increase in size (5 bytes)
  each time they set this new field.

###### Will enabling / using this feature result in increasing time taken by any operations covered by existing SLIs/SLOs?

No.

###### Will enabling / using this feature result in non-negligible increase of resource usage (CPU, RAM, disk, IO, ...) in any components?

No.

###### Can enabling / using this feature result in resource exhaustion of some node resources (PIDs, sockets, inodes, etc.)?

No.

### Troubleshooting

###### How does this feature react if the API server and/or etcd is unavailable?

API server or etcd issues do not impact this feature.

###### What are other known failure modes?

We do not expect any new failure mode. (While setting `tolerance` below 10% can cause HPAs
to scale up and down as frequently as every 30s, and higher values might stop scaling altogether
if the metric remains within the tolerance band, the feature is still working as intended.
To make HPAs respond faster, decrease the tolerance value. Conversely, to make them respond
slower, increase the tolerance value.)

###### What steps should be taken if SLOs are not being met to determine the problem?

If possible increase the log level for kube-controller-manager and check controller logs:
1. Search for "Proposing desired replicas", verify that the tolerance is set as expected,
   and check (using `kubectl describe hpa`) if the ratio between the _current_ and _desired_
   metric values is in tolerance.
3. Look for warnings and errors which might point where the problem lies.

## Implementation History

2025-01-21: KEP PR merged.
2025-03-24: [Implementation PR](https://github.com/kubernetes/kubernetes/pull/130797) merged.
2025-05-15: Kubernetes v1.33 released (includes this feature).
2025-05-16: This KEP updated for beta graduation.
2026-05-21: This KEP updated for stable graduation.
2026-08-18: This KEP status is set to *implemented*.

## Drawbacks

No major drawbacks have been identified.

## Alternatives

On non-managed Kubernetes instances, users can update the cluster-wide
`--horizontal-pod-autoscaler-tolerance` tolerance parameter,

## Infrastructure Needed (Optional)

N/A.
