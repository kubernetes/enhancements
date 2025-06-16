<!--
**Note:** When your KEP is complete, all of these comment blocks should be removed.

To get started with this template:

- [ ] **Pick a hosting SIG.**
  Make sure that the problem space is something the SIG is interested in taking
  up. KEPs should not be checked in without a sponsoring SIG.
- [ ] **Create an issue in kubernetes/enhancements**
  When filing an enhancement tracking issue, please make sure to complete all
  fields in that template. One of the fields asks for a link to the KEP. You
  can leave that blank until this KEP is filed, and then go back to the
  enhancement and add the link.
- [ ] **Make a copy of this template directory.**
  Copy this template into the owning SIG's directory and name it
  `NNNN-short-descriptive-title`, where `NNNN` is the issue number (with no
  leading-zero padding) assigned to your enhancement above.
- [ ] **Fill out as much of the kep.yaml file as you can.**
  At minimum, you should fill in the "Title", "Authors", "Owning-sig",
  "Status", and date-related fields.
- [ ] **Fill out this file as best you can.**
  At minimum, you should fill in the "Summary" and "Motivation" sections.
  These should be easy if you've preflighted the idea of the KEP with the
  appropriate SIG(s).
- [ ] **Create a PR for this KEP.**
  Assign it to people in the SIG who are sponsoring this process.
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
# KEP-5625: HPA - Improve pod selection accuracy across workload types

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
  - [Pluggable Pod Filtering](#pluggable-pod-filtering)
  - [Controller Enhancements](#controller-enhancements)
  - [Scope of Support](#scope-of-support)
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

The Horizontal Pod Autoscaler (HPA) has a critical limitation in its pod selection mechanism: it collects metrics from all pods that match the target workload's label selector.
This can lead to incorrect scaling decisions when unrelated pods (such as Jobs, CronJobs, or other Deployments) happen to share the same labels.

This often results in unexpected behavior such as:

* HPAs stuck at maxReplicas despite low actual usage in the target workload
* Unnecessary scaling events triggered by temporary workloads
* Unpredictable scaling behavior that's difficult to diagnose

This proposal adds a parameter to HPAs which ensures the HPA only considers pods that are actually owned by the target workload, through owner references, rather than all pods matching the label selector.

## Motivation

Consider this example:

```yaml
apiVersion: apps/v1
kind: Deployment
metadata:
  name: test-app
spec:
  replicas: 1
  selector:
    matchLabels:
      app: test-app
  template:
    metadata:
      labels:
        app: test-app
    spec:
      containers:
      - name: nginx
        image: nginx
        resources:
          requests:
            cpu: 100m
---
apiVersion: batch/v1
kind: Job
metadata:
  name: test-job
spec:
  template:
    metadata:
      labels:
        app: test-app  # Same label as deployment
        workload: scraper
    spec:
      containers:
      - name: cpu-load
        image: busybox
        command: ["dd", "if=/dev/zero", "of=/dev/null"]
        resources:
          requests:
            cpu: 100m
      restartPolicy: Never
---
apiVersion: autoscaling/v2
kind: HorizontalPodAutoscaler
metadata:
  name: test-app-hpa
spec:
  scaleTargetRef:
    apiVersion: apps/v1
    kind: Deployment
    name: test-app
  minReplicas: 1
  maxReplicas: 5
  metrics:
  - type: Resource
    resource:
      name: cpu
      target:
        type: Utilization
        averageUtilization: 50
```

In this case, the HPA will factor in CPU consumption from the Job's pod, despite it not being part of the Deployment, potentially causing incorrect scaling decisions.

### Goals

* Improve the accuracy of HPA's pod selection to only include pods directly managed by the target workload
* Maintain backward compatibility with existing HPA configurations
* Provide clear visibility into which pods are being considered for scaling decisions
* Allow users to choose between selection strategies based on their needs

### Non-Goals

* Modifying how metrics are collected from pods
* Changing the scaling algorithm itself
* Addressing other HPA limitations not related to pod selection

## Proposal

We propose adding a new field to the HPA specification called `SelectionStrategy` that allows users to specify how pods should be selected for metric collection:

* If set to `LabelSelector` (default): Uses the current behavior of selecting all pods that match the target workload's label selector.
* If set to `OwnerReference`: Only selects pods that are owned by the target workload through owner references.

This enumerated type allows for future extension with additional selection strategies if needed, such as `Annotations` etc.

### Risks and Mitigations

* Backward compatibility: Mitigated by making the new behavior opt-in with the current behavior as default.
* User confusion: We'll provide clear documentation on when and how to use each strategy.

## Design Details

The HPA specification (v2) will be extended with a new field to control additional filtering of pods after the initial label selector matching:

```yaml
apiVersion: autoscaling/v2
kind: HorizontalPodAutoscaler
spec:
  # Existing fields...
  SelectionStrategy: OwnerReference  # Default: LabelSelector
```

Since the added field is optional and its omission does not change the existing
autoscaling behavior, this feature will only be added to the latest stable API
version `pkg/apis/autoscaling/v2`. Older versions (i.e. `v1`, `v2beta1`,
`v2beta2`) will not include the new field, but converters will be updated where
needed to comply with [round-trip requirements][].

Pod Selection Process:

- Initial Label Selection (Always happens):
  * The HPA first selects pods using the target workload's label selector
  * This is the fundamental selection mechanism and remains unchanged
- Additional Filtering (Based on SelectionStrategy):
  * `LabelSelector`(default):
    * No additional filtering
    * All pods that matched the label selector are used for metrics
    * Maintains current behavior for backward compatibility
  * `OwnerReference`:
    * Further filters the label-selected pods
    * Only keeps pods that are owned by the target workload through owner references
    * Follows the ownership chain (e.g., Pods -> ReplicaSet -> Deployment)
    * Excludes pods that matched labels but aren't in the ownership chain

The `HorizontalPodAutoscaler` API updated to add a new `SelectionStrategy` field to the `HorizontalPodAutoscalerSpec` object:

```go
// SelectionStrategy defines how pods are selected for metrics collection
type SelectionStrategy string

const (
    // LabelSelector selects all pods matching the target's label selector
    LabelSelector SelectionStrategy = "LabelSelector"
    
    // OwnerReference only selects pods owned by the target workload
    OwnerReference SelectionStrategy = "OwnerReference"
)

// In HorizontalPodAutoscalerSpec:
type HorizontalPodAutoscalerSpec struct {
    // existing fields...

    // SelectionStrategy determines how pods are selected for metrics collection.
    // Valid values are "LabelSelector" and "OwnerReference".
    // If not set, defaults to "LabelSelector" which is the legacy behavior.
    // +optional
    SelectionStrategy *SelectionStrategy `json:"SelectionStrategy,omitempty"`
}
```

[round-trip requirements]: https://kubernetes.io/docs/reference/using-api/deprecation-policy/#deprecating-parts-of-the-api

### Pluggable Pod Filtering

The HPA controller introduces a pluggable PodFilter interface to encapsulate different filtering strategies:

```go
// PodFilter defines an interface for filtering pods based on various strategies
type PodFilter interface {
	// Filter returns the subset of pods that should be considered for metrics calculation,
	// along with the pods that were filtered out
	Filter(pods []*v1.Pod) (filtered []*v1.Pod, unfiltered []*v1.Pod, err error)
	// Name returns the name of the filter strategy for logging purposes
	Name() string
}
```

Two implementations are provided:

`LabelSelectorFilter`:

* Default implementation
* Passes through all pods that match the label selector
* Maintains existing behavior for backward compatibility

`OwnerReferenceFilter`:

* Validates pod ownership through reference chain
* Only includes pods that are owned by the target workload
* Handles different workload types (Deployments, StatefulSets, etc.)

### Controller Enhancements

The HPA controller caches filters for improved performance:

```go
type HorizontalController struct {
    // ... existing fields ...
    podFilterCache map[string]PodFilter
    podFilterMux   sync.RWMutex
}
```

All metrics collection methods (e.g., `GetResourceReplicas`) are updated to accept a `PodFilter`:

```go
// GetResourceReplicas calculates the desired replica count based on a target resource utilization percentage
// of the given resource for pods matching the given selector in the given namespace, and the current replica count.
// The calculation follows these steps:
// 1. Gets resource metrics for pods in the namespace matching the selector
// 2. Lists all pods matching the selector
// 3. Applies the podFilter to select pods that should be considered for scaling
// 4. Groups considered pods into ready, unready, missing, and ignored pods
// 5. Removes metrics for ignored and unready pods
// 6. Calculates the desired replica count based on the resource utilization of considered pods
//
// Returns:
// - replicaCount: the recommended number of replicas
// - utilization: the current utilization percentage
// - rawUtilization: the raw resource utilization value
// - timestamp: when the metrics were collected
// - err: any error encountered during calculation
func (c *ReplicaCalculator) GetResourceReplicas(ctx context.Context, currentReplicas int32, targetUtilization int32, resource v1.ResourceName, tolerances Tolerances, namespace string, selector labels.Selector, container string, podFilter PodFilter) (replicaCount int32, utilization int32, rawUtilization int64, timestamp time.Time, err error) {
```

Filtered pods are then used as the basis for replica calculations:

```go
  if len(podList) == 0 {
		return 0, 0, 0, time.Time{}, fmt.Errorf("no pods returned by selector while calculating replica count")
	}
  filteredPods, unfilteredPods, err := podFilter.Filter(podList)

  if err != nil {
    // Fall back to default behavior: use all pods
    filteredPods = podList
    unfilteredPods = []*v1.Pod{} // empty slice since we're not filtering out any pods
  }

  unfilteredPodNames := sets.New[string]()
	for _, pod := range unfilteredPods {
		unfilteredPodNames.Insert(pod.Name)
	}
	removeMetricsForPods(metrics, unfilteredPodNames)
	readyPodCount, unreadyPods, missingPods, ignoredPods := groupPods(filteredPods, metrics, resource, c.cpuInitializationPeriod, c.delayOfInitialReadinessStatus)
	removeMetricsForPods(metrics, ignoredPods)
	removeMetricsForPods(metrics, unreadyPods)
```
If filtering fails (e.g., due to RBAC issues), the system defaults to using all pods, ensuring robust behavior.

The HPA controller implements caching to optimize API server queries when checking pod ownership:

```go
type ControllerCache struct {
    mutex         sync.RWMutex
    resources     map[string]*ControllerCacheEntry
    dynamicClient dynamic.Interface
    restMapper    apimeta.RESTMapper
    cacheTTL      time.Duration
}

type ControllerCacheEntry struct {
    Resource    *unstructured.Unstructured
    Error       error
    LastFetched time.Time
}
```
The cache system provides several benefits:

- Reduced API Server Load: Caches controller resources to minimize API server queries
- Improved Performance: Faster pod ownership validation through in-memory lookups
- Configurable TTL: Allows tuning of cache freshness vs performance trade-off
- Automatic Cleanup: Background goroutine removes expired entries

When validating pod ownership, the system first checks the cache
If a valid (non-expired) entry exists, it's returned immediately
Otherwise, the controller fetches from the API server and updates the cache
Expired entries are automatically cleaned up by a background goroutine

### Scope of Support

This enhancement applies consistently across the following supported metric types in the HorizontalPodAutoscaler:

- Resource metrics (e.g., CPU, memory)
- Pods metrics
- Container resource metrics
- Object metrics (only when AverageValue[^1] type is selected with `spec.metrics.object.target.type`)
- External metrics (only when AverageValue[^1] type is selected with `spec.metrics.external.target.type`)

[^1]: With AverageValue, the value returned from the custom metrics API is divided by the number of Pods before being compared to the target, thus requiring improved pod selection. However, Value, the target is compared directly to the returned metric from the API.

Reference: [Kubernetes HPA metric types](https://kubernetes.io/docs/tasks/run-application/horizontal-pod-autoscale/#support-for-resource-metrics)

When a user updates an HPA to change its pod selection strategy:

- The controller detects strategy changes during HPA updates
- The pod filter cache is cleared for the modified HPA
- A new filter is created using the updated strategy
- An event is recorded to notify users of the strategy change:

```bash
Normal  StrategyChanged  Pod selection strategy changed from LabelSelector to OwnerReference
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
Tests for Pod Filters:

- Verify `LabelSelectorFilter` includes all pods matching labels
- Verify `OwnerReferenceFilter` includes only pods owned by target workload
- Verify filters handle edge cases (no owners, broken chains, multiple owners)

Tests for Replica Calculator:
- Verify calculations with `LabelSelectorFilter` match current behavior
- Verify calculations with `OwnerReferenceFilter` only include owned pods
- Verify correct behavior with mixed owned/unowned pods


- `/pkg/controller/podautoscaler`:`16 June 2025`-`88.0%`
- `/pkg/controller/podautoscaler/metrics`:`16 June 2025`-`90.0%`

##### Integration tests

<!--
Integration tests are contained in https://git.k8s.io/kubernetes/test/integration.
Integration tests allow control of the configuration parameters used to start the binaries under test.
This is different from e2e tests which do not allow configuration of parameters.
Doing this allows testing non-default options and multiple different and potentially conflicting command line options.
For more details, see https://github.com/kubernetes/community/blob/master/contributors/devel/sig-testing/testing-strategy.md

If integration tests are not necessary or useful, explain why.
-->

<!--
This question should be filled when targeting a release.
For Alpha, describe what tests will be added to ensure proper quality of the enhancement.

For Beta and GA, document that tests have been written,
have been executed regularly, and have been stable.
This can be done with:
- permalinks to the GitHub source code
- links to the periodic job (typically https://testgrid.k8s.io/sig-release-master-blocking#integration-master), filtered by the test name
- a search in the Kubernetes bug triage tool (https://storage.googleapis.com/k8s-triage/index.html)
-->

N/A, the feature is tested using unit tests and e2e tests.

##### e2e tests

<!--
This question should be filled when targeting a release.
For Alpha, describe what tests will be added to ensure proper quality of the enhancement.

For Beta and GA, document that tests have been written,
have been executed regularly, and have been stable.
This can be done with:
- permalinks to the GitHub source code
- links to the periodic job (typically a job owned by the SIG responsible for the feature), filtered by the test name
- a search in the Kubernetes bug triage tool (https://storage.googleapis.com/k8s-triage/index.html)

We expect no non-infra related flakes in the last month as a GA graduation criteria.
If e2e tests are not necessary or useful, explain why.
-->

We will add the following [e2e autoscaling tests](https://github.com/kubernetes/kubernetes/tree/master/test/e2e/autoscaling):

- For owner references strategy:
  - Workload should not scale up when CPU/Memory usage comes from pods not owned by the target
  - HPA ignores metrics from pods with matching labels but no owner reference to the target
- For label selector strategy:
  - Workload scales up when CPU/Memory usage comes from any pods matching labels (current behavior)
  - HPA considers metrics from all pods with matching labels regardless of ownership
  - Verify backward compatibility when SelectionStrategy is not set

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
- More rigorous forms of testing—e.g., downgrade tests and scalability tests
- All functionality completed
- All security enforcement completed
- All monitoring requirements completed
- All testing requirements completed
- All known pre-release issues and gaps resolved 

**Note:** Beta criteria must include all functional, security, monitoring, and testing requirements along with resolving all issues and gaps identified

#### GA

- N examples of real-world usage
- N installs
- Allowing time for feedback
- All issues and gaps identified as feedback during beta are resolved

**Note:** GA criteria must not include any functional, security, monitoring, or testing requirements.  Those must be beta requirements.

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

- Feature implemented behind a feature flag: `HPASelectionStrategy`
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

Existing HPAs will continue to work as they do today, using the default LabelSelector strategy.
Users can use the new feature by enabling the Feature Gate (alpha only) and setting the SelectionStrategy field to OwnerReference on an HPA.

#### Downgrade

On downgrade, all HPAs will revert to using the LabelSelector strategy, regardless of any configured SelectionStrategy value on the HPA itself.

### Version Skew Strategy

1. `kube-apiserver`: More recent instances will accept the new SelectionStrategy field, while older instances will ignore it during validation and persist it as part of the HPA object.
2. `kube-controller-manager`: An older version could receive an HPA containing the new SelectionStrategy field from a more recent API server, in which case it would ignore it (i.e., continue to use the default LabelSelector strategy regardless of the field's value).

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

<!--
Pick one of these and delete the rest.

Documentation is available on [feature gate lifecycle] and expectations, as
well as the [existing list] of feature gates.

[feature gate lifecycle]: https://git.k8s.io/community/contributors/devel/sig-architecture/feature-gates.md
[existing list]: https://kubernetes.io/docs/reference/command-line-tools-reference/feature-gates/
-->

- [x] Feature gate (also fill in values in `kep.yaml`)
  - Feature gate name: SPASelectionStrategy
  - Components depending on the feature gate: `kube-controller-manager` and
    `kube-apiserver`.

###### Does enabling the feature change any default behavior?

<!--
Any change of default behavior may be surprising to users or break existing
automations, so be extremely careful here.
-->
No. By default, HPAs will continue to use the `LabelSelector` strategy unless the new `SelectionStrategy` field is explicitly set to `OwnerReference`.

###### Can the feature be disabled once it has been enabled (i.e. can we roll back the enablement)?

<!--
Describe the consequences on existing workloads (e.g., if this is a runtime
feature, can it break the existing applications?).

Feature gates are typically disabled by setting the flag to `false` and
restarting the component. No other changes should be necessary to disable the
feature.

NOTE: Also set `disable-supported` to `true` or `false` in `kep.yaml`.
-->
Yes. If the feature gate is disabled, all HPAs will revert to using the `LabelSelector` strategy regardless of the value of the `SelectionStrategy` field.

###### What happens if we reenable the feature if it was previously rolled back?

When the feature is re-enabled, any HPAs with `SelectionStrategy`: `OwnerReference` will resume using the ownership-based pod selection rather than label-based selection.
The HPA controller will immediately begin considering only pods directly owned by the target workload for scaling decisions on these HPAs, potentially changing scaling behavior compared to when the feature was disabled.

Existing HPAs that don't have `SelectionStrategy` explicitly set will continue using the default LabelSelector strategy and won't be affected by re-enabling the feature.

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

We will add a unit test verifying that HPAs with and without the new `SelectionStrategy` field are properly validated, both when the feature gate is enabled or not.
This will ensure the HPA controller correctly applies the pod selection strategy based on the feature gate status and presence of the field.

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
Rollout failures in this feature are unlikely to impact running workloads significantly, but there are edge cases to consider:

If the feature is enabled during a high-traffic period, HPAs with `SelectionStrategy: OwnerReference` might suddenly change their scaling decisions based on the reduced pod set. However, this is mitigated by:

- The HPA's existing behavior specs (minReplicas/maxReplicas) which prevent extreme scaling events
- The gradual nature of HPA scaling decisions
If a kube-controller-manager restarts mid-rollout, some HPAs might temporarily revert to the `LabelSelector` strategy until the controller fully initializes with the new feature enabled. This is mitigated by:

- The HPA's behavior specs which limit the scale of any potential changes
- Normal operation resumes after controller initialization

These issues would only affect HPAs that have explicitly set SelectionStrategy: OwnerReference. Existing HPAs will continue to function with the default LabelSelector strategy.

###### What specific metrics should inform a rollback?

<!--
What signals should users be paying attention to when the feature is young
that might indicate a serious problem?
-->
Operators should monitor these signals that might indicate problems:

- Unexpected scaling events shortly after enabling the feature
- Significant changes in the number of replicas for workloads using HPAs with `SelectionStrategy: OwnerReference`
- Increased latency in the `horizontal_pod_autoscaler_controller_metric_computation_duration_seconds` metric
- Increased error rate in `horizontal_pod_autoscaler_controller_metric_computation_total` with error status
If these metrics show unusual patterns after enabling the feature, operators should consider rolling back.

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
No. This feature only adds a new optional field to the HPA API and doesn't deprecate or remove any existing functionality.
All current HPA behaviors remain unchanged unless users explicitly opt into the new selection mode.

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
The presence of the `SelectionStrategy` field in HPA specifications indicates that the feature is in use.

###### How can someone using this feature know that it is working for their instance?

<!--
For instance, if this is a pod-related feature, it should be possible to determine if the feature is functioning properly
for each individual pod.
Pick one more of these and delete the rest.
Please describe all items visible to end users below with sufficient detail so that they can verify correct enablement
and operation of this feature.
Recall that end users cannot usually observe component logs or access metrics.
-->

Users can confirm that the feature is active and functioning by inspecting the conditions exposed by the controller. Specifically, they can verify the value of `.spec.SelectionStrategy` to ensure the expected behavior is enabled.
Moreover, users can verify the feature is working properly through events on the HPA object:
- When creating or updating an HPA with SelectionStrategy: OwnerReference, an event will be emitted, similar to this: `Normal  SelectionStrategyActive  "Pod selection strategy 'OwnerReference' is active"`
- When switching strategies, an event will indicate the change, similar to this: `Normal  StrategyChanged  "Pod selection strategy changed from 'LabelSelector' to 'OwnerReference'"`

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
  - The new pod filtering should not significantly impact these durations

- `horizontal_pod_autoscaler_controller_metric_computation_duration_seconds`
  - Measures time taken to calculate metrics with labels for action, error, and metric_type
  - The pod filtering logic should work within existing computation time buckets (exponential buckets from 0.001s to ~16s)

- `horizontal_pod_autoscaler_controller_metric_computation_total`
  - Counts number of metric computations with labels for action, error, and metric_type
  - The pod filtering should not introduce new error cases in metric computation

The feature should maintain the current performance characteristics of these metrics.

###### What are the SLIs (Service Level Indicators) an operator can use to determine the health of the service?

<!--
Pick one more of these and delete the rest.
-->
This feature doesn't fundamentally change how the HPA controller operates; it refines which pods are included in metric calculations.
Therefore, existing metrics for monitoring HPA controller health remain applicable.
Standard HPA metrics (e.g. `horizontal_pod_autoscaler_controller_metric_computation_duration_seconds`) can be used to verify the HPA controller health.

###### Are there any missing metrics that would be useful to have to improve observability of this feature?

<!--
Describe the metrics themselves and the reasons why they weren't added (e.g., cost,
implementation difficulties, etc.).
-->
The following metrics should be added to improve cache observability:

- Cache hit counter: Tracks when the controller successfully retrieves data from cache
- Cache miss counter: Tracks when the controller needs to query the API server

These metrics are essential for:
- Monitoring cache effectiveness
- Optimizing cache TTL settings
- Identifying potential performance issues
- Understanding API server query patterns

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
Yes.
Enabling or using this feature will result in new API calls, specifically:

  - API Call Type: GET (read) operations
  - Resources Involved: Deployments, ReplicaSets, and potentially other workload-related resources
  - HPA controller

###### Will enabling / using this feature result in introducing new API types?

<!--
Describe them, providing:
  - API type
  - Supported number of objects per cluster
  - Supported number of objects per namespace (for namespace-scoped objects)
-->
No.

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
Yes, HorizontalPodAutoscaler objects will increase in size by approximately ~39 bytes for the string field when specified

###### Will enabling / using this feature result in increasing time taken by any operations covered by existing SLIs/SLOs?

<!--
Look at the [existing SLIs/SLOs].

Think about adding additional work or introducing new steps in between
(e.g. need to do X to start a container), etc. Please describe the details.

[existing SLIs/SLOs]: https://git.k8s.io/community/sig-scalability/slos/slos.md#kubernetes-slisslos
-->
Yes, enabling this feature may introduce a slight increase in latency due to additional resource checks. For example, in the case of a Deployment, the system may need to perform two extra ownership checks (e.g., Pod → ReplicaSet → Deployment). While this added processing could have some impact, it is expected to be negligible in most scenarios.

###### Will enabling / using this feature result in non-negligible increase of resource usage (CPU, RAM, disk, IO, ...) in any components?

<!--
Things to keep in mind include: additional in-memory state, additional
non-trivial computations, excessive access to disks (including increased log
volume), significant amount of data sent and/or received over network, etc.
This through this both in small and large cases, again with respect to the
[supported limits].

[supported limits]: https://git.k8s.io/community//sig-scalability/configs-and-limits/thresholds.md
-->
Yes, cahing will be implemented for each `podsFilter` strategy, as well as for other resources to reduce the number of API calls to the API server (as described above).

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
- Get pod metrics
- Access workload information
- Update HPA status

Therefore, no autoscaling decisions can be made during this period, regardless of the configured selection strategy. The feature itself doesn't introduce any new failure modes with respect to API server or etcd availability - it's dependent on these components being available just like the rest of the HPA controller's functionality.
Once API server and etcd access is restored, the HPA controller will resume normal operation, including the pod selection strategy specified in the HPA.

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
Check `horizontal_pod_autoscaler_controller_metric_computation_duration_seconds` to identify if the increased latency correlates with HPAs using the OwnerReference selection strategy.
If latency issues are observed:
  - Check if the problem only affects HPAs with `SelectionStrategy: OwnerReference`
  - Verify if the latency increases with deeper ownership chains (e.g., Pod → ReplicaSet → Deployment)
For problematic HPAs, you can:
  - Temporarily revert to the default label-based selection by removing the `SelectionStrategy` field
  - Or explicitly set `SelectionStrategy: LabelSelector` to maintain backward compatibility

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
KEP Published: 05/22/2025

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
