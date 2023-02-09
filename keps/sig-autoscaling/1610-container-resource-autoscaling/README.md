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
# KEP-1610: Container Resource based Autoscaling

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
    - [Multiple containers with different scaling thresholds](#multiple-containers-with-different-scaling-thresholds)
    - [Multiple containers but only scaling for one.](#multiple-containers-but-only-scaling-for-one)
    - [Add container metrics to existing pod resource metric.](#add-container-metrics-to-existing-pod-resource-metric)
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

The Horizontal Pod Autoscaler supports scaling of targets based on the resource usage
of the pods in the target. The resource usage of pods is calculated as the sum
of the individual container usage values of the pod. This is unsuitable for workloads where
the usage of the containers are not strongly correlated or do not change in lockstep. This KEP
suggests that when scaling based on resource usage the HPA also provide an option
to consider the usages of individual containers to make scaling decisions.

## Motivation

An HPA is used to ensure that a scaling target is scaled up or down in such a way that the
specificed current metric values are always maintained at a certain level. Resource based
autoscaling is the most basic approach to autoscaling and has been present in the HPA spec since `v1`.
In this mode the HPA controller fetches the current resource metrics for all the pods of a scaling
target and then computes how many pods should be added or removed based on the current usage to
achieve the target average usage.

For performance critical applications where the resource usage of individual containers needs to
be configured individually the default behavior of the HPA controller may be unsuitable. When
there are multiple containers in the pod their individual resource usages may not have a direct
correlation or may grow at different rates as the load changes. There are several reasons for this:

  - A sidecar container is only providing an auxiliary service such as log shipping. If the
    application does not log very frequently or does not produce logs in its hotpath then the usage of
    the log shipper will not grow.
  - A sidecar container which provides authentication. Due to heavy caching the usage will only
    increase slightly when the load on the main container increases. In the current blended usage
    calculation approach this usually results in the the HPA not scaling up the deployment because
    the blended usage is still low.
  - A sidecar may be injected without resources set which prevents scaling based on utilization. In
    the current logic the HPA controller can only scale on absolute resource usage of the pod when
    the resource requests are not set.

The optimum usage of the containers may also be at different levels. Hence the HPA should offer
a way to specify the target usage in a more fine grained manner.

### Goals

- Make HPA scale based on individual container resources usage

### Non-Goals
- Configurable aggregation for containers resources in pods.
- Optimization of the calls to the `metrics-server`

## Proposal

Currently the HPA accepts multiple metric sources to calculate the number of replicas in the target,
one of which is called `Resource`. The `Resource` type represents the resource usage of the
pods in the scaling target. The resource metric source has the following structure:

```go
type ResourceMetricSource struct {
	Name v1.ResourceName
	Target MetricTarget
}
```

Here the `Name` is the name of the resource. Currently only `cpu` and `memory` are supported
for this field. The other field is used to specify the target at which the HPA should maintain
the resource usage by adding or removing pods. For instance if the target is _60%_ CPU utilization,
and the current average of the CPU resources across all the pods of the target is _70%_ then
the HPA will add pods to reduce the CPU utilization. If it's less than _60%_ then the HPA will
remove pods to increase utilization.

It should be noted here that when a pod has multiple containers the HPA gets the resource
usage of all the containers and sums them to get the total usage. This is then divided
by the total requested resources to get the average utilizations. For instance if there is
a pods with 2 containers: `application` and `log-shipper` requesting `250m` and `250m` of
CPU resources then the total requested resources of the pod as calculated by the HPA is `500m`.
If then the first container is currently using `200m` and the second only `50m` then
the usage of the pod is `250m` which in utilization is _50%_. Although individually
the utilization of the containers are _80%_ and _20%_. In such a situation the performance
of the `application` container might be affected significantly. There is no way to specify
in the HPA to keep the utilization of the first container below a certain threshold. This also
affects `memory` resource based autocaling scaling.

We propose that the a new metric source called `ContainerResourceMetricSource` be introduced
with the following structure:

```go
type ContainerResourceMetricSource struct {
	Container string
	Name v1.ResourceName
	Target MetricTarget
}
```

The only new field is `Container` which is the name of the container for which the resource
usage should be tracked.

### User Stories

#### Multiple containers with different scaling thresholds

Assume the user has a deployment with multiple pods, each of which have multiple containers. A main
container called `application` and 2 others called `log-shipping` and `authnz-proxy`. Two
of the containers are critical to provide the application functionality, `application` and
`authnz-proxy`. The user would like to prevent _OOMKill_ of these containers and also keep
their CPU utilization low to ensure the highest performance. The other container
`log-shipping` is less critical and can tolerate failures and restarts. In this case the
user would create an HPA with the following configuration:

```yaml
apiVersion: autoscaling/v2beta2
kind: HorizontalPodAutoscaler
metadata:
  name: mission-critical
spec:
  scaleTargetRef:
    apiVersion: apps/v1
    kind: Deployment
    name: mission-critical
  minReplicas: 1
  maxReplicas: 10
  metrics:
  - type: ContainerResource
    resource:
      name: cpu
      container: application
      target:
        type: Utilization
        averageUtilization: 30
  - type: ContainerResource
    resource:
      name: memory
      container: application
      target:
        type: Utilization
        averageUtilization: 80
  - type: ContainerResource
    resource:
      name: cpu
      container: authnz-proxy
      target:
        type: Utilization
        averageUtilization: 30
  - type: ContainerResource
    resource:
      name: memory
      container: authnz-proxy
      target:
        type: Utilization
        averageUtilization: 80
  - type: ContainerResource
    resource:
      name: cpu
      container: log-shipping
      target:
        type: Utilization
        averageUtilization: 80
```

The HPA specifies that the HPA controller should maintain the CPU utilization of the containers
`application` and `authnz-proxy` at _30%_ and the memory utilization at _80%_. The `log-shipping`
container is scaled to keep the cpu utilization at _80%_ and is not scaled on memory.

#### Multiple containers but only scaling for one.
Assume the user has a deployment where the pod spec has multiple containers but scaling should
be performed based only on the utilization of one of the containers. There could be several reasons
for such a strategy: Disruptions due to scaling of sidecars may be expensive and should be avoided
or the resource usage of the sidecars could be erratic because it has a different work characteristics
to the main container.

In such a case the user creates an HPA as follows:

```yaml
apiVersion: autoscaling/v2beta2
kind: HorizontalPodAutoscaler
metadata:
  name: mission-critical
spec:
  scaleTargetRef:
    apiVersion: apps/v1
    kind: Deployment
    name: mission-critical
  minReplicas: 1
  maxReplicas: 10
  metrics:
  - type: ContainerResource
    resource:
      name: cpu
      container: application
      target:
        type: Utilization
        averageUtilization: 30
```

The HPA controller will then completely ignore the resource usage in other containers.

#### Add container metrics to existing pod resource metric.
A user who is already using an HPA to scale their application can add the container metric source to the HPA
in addition to the existing pod metric source. If there is a single container in the pod then the behavior
will be exactly the same as before. If there are multiple containers in the application pods then the deployment
might scale out more than before. This happens when the resource usage of the specified container is more
than the blended usage as calculated by the pod metric source. If however in the unlikely case, the usage of
all the containers in the pod change in tandem by the same amount then the behavior will remain as before.

For example consider the HPA object which targets a _Deployment_ with pods that have two containers `application`
and `log-shipper`:

```yaml

apiVersion: autoscaling/v2beta2
kind: HorizontalPodAutoscaler
metadata:
  name: mission-critical
spec:
  scaleTargetRef:
    apiVersion: apps/v1
    kind: Deployment
    name: mission-critical
  minReplicas: 1
  maxReplicas: 10
  metrics:
  - type: ContainerResource
    resource:
      name: cpu
      container: application
      target:
        type: Utilization
        averageUtilization: 50
  - type: PodResource
    resource:
      name: cpu
      target:
        type: Utilization
        averageUtilization: 50
```

If the resource usage of the `application` container increases then the target would be scaled out even if
the usage of the `log-shipper` container does not increase much. If the resource usage of `log-shipper` container
increases then the deployment would only be scaled out if the combined resource usage of both containers increases
above the target.


### Risks and Mitigations

Since the new field `container` in the container resource metric source is not validated against the target it is
possible that the user could specify an invalid value, i.e. a container name which is not part of the pod. The HPA
controller would treat this as invalid configuration and prevent scale down. However scale up would still be possible
based on recommendations from other metric sources.

A similar problem is possible when renaming container names in the HPA. To mitigate this the recommended procedure
is to have both the old and new container names during the deployment. The old container name can be removed from
the HPA when the migration is complete.


## Design Details

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

Most of the tests will follow the same pattern as the tests for the pod resource metric source. The following
unit tests will be added:

- **Replica Calculator:** Verify that the number of replicas calculated is based on the metrics of individual
  containers when a container metric source is specified.
- **REST Metrics Client:** Verify that the resources returned from the REST metric client is the metrics for only
  the containers specified in the metric source.
- **API server validation:** Verify that only valid container metric sources are accepted.
- **kubectl:** Verify that the new metric sources are displayed correctly.

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

- `k8s.io/kubernetes/pkg/controller/podautoscaler`: `2023/02/02` - `87.8%`
- `k8s.io/kubernetes/pkg/controller/podautoscaler/metrics`: `2023/02/02` - `90.2%`
- `k8s.io/kubernetes/pkg/apis/autoscaling/validation`: `2023/02/02` - `95.2%`
- `k8s.io/kubectl/pkg/describe/describe.go`: `2023/02/02` - `68.4%`

##### Integration tests

<!--
This question should be filled when targeting a release.
For Alpha, describe what tests will be added to ensure proper quality of the enhancement.

For Beta and GA, add links to added tests together with links to k8s-triage for those tests:
https://storage.googleapis.com/k8s-triage/index.html
-->

N/A

The HPA behaviors are tested thoroughly in the e2e tests described below,
and the integration tests doesn't add extra value to those e2e tests.

##### e2e tests

<!--
This question should be filled when targeting a release.
For Alpha, describe what tests will be added to ensure proper quality of the enhancement.

For Beta and GA, add links to added tests together with links to k8s-triage for those tests:
https://storage.googleapis.com/k8s-triage/index.html

We expect no non-infra related flakes in the last month as a GA graduation criteria.
-->

**k8s-triage**

- https://storage.googleapis.com/k8s-triage/index.html?sig=autoscaling&job=ci-kubernetes-e2e-gci-gce-autoscaling&test=Container%20Resource

**tests**

- https://github.com/kubernetes/kubernetes/blob/d4750857760ae55802f69989dc2451feeb9a29e5/test/e2e/autoscaling/horizontal_pod_autoscaling.go#L61
- https://github.com/kubernetes/kubernetes/blob/d4750857760ae55802f69989dc2451feeb9a29e5/test/e2e/autoscaling/horizontal_pod_autoscaling.go#L163
- https://github.com/kubernetes/kubernetes/blob/d4750857760ae55802f69989dc2451feeb9a29e5/test/e2e/autoscaling/horizontal_pod_autoscaling.go#L120
- https://github.com/kubernetes/kubernetes/blob/d4750857760ae55802f69989dc2451feeb9a29e5/test/e2e/autoscaling/custom_metrics_stackdriver_autoscaling.go#L323

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

- Feature implemented behind a feature gate
- Initial e2e tests completed and enabled

#### Beta

- The feature gate is enabled by default.
- No negative feedback during alpha for a long-enough time.
- No bug issues reported during alpha.
- Implementing/exposing metrics in HPA so that users can monitor the HPA controller for this feature. 

#### GA

- No negative feedback during beta for a long-enough time.
- No bug issues reported during beta.

### Upgrade / Downgrade Strategy

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

**Upgrade**

The previous HPA behavior will not be broken. Users can continue to use
their HPA specs as it is.

To use this enhancement, 
- [only alpha] users need to enable the feature gate `HPAContainerMetrics`
- add `ContainerResource` type metric on their HPA.
  - https://kubernetes.io/docs/tasks/run-application/horizontal-pod-autoscale/#container-resource-metrics

**Downgrade**

For newly created HPAs, kube-apiserver will drop `ContainerResource` metric 
and thus, HPA controller will also do nothing with it.

For existing HPAs, **the current implementation will continue to work on autoscaling based on `ContainerResource`.**
This behavior will be changed to ignore `ContainerResource` when the feature gate is disabled by the beta.
 ([issue](https://github.com/kubernetes/kubernetes/issues/115467))

### Version Skew Strategy

<!--
If applicable, how will the component handle version skew with other
components? What are the guarantees? Make sure this is in the test plan.

Consider the following in developing a version skew strategy for this
enhancement:
- Does this enhancement involve coordinating behavior in the control plane and
  in the kubelet? How does an n-2 kubelet without this feature available behave
  when this feature is used?
- Will any other components on the node change? For example, changes to CSI,
  CRI or CNI may require updating that component before the kubelet.
-->

N/A

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
  - Feature gate name: `HPAContainerMetrics`
  - Components depending on the feature gate: `kube-apiserver`, `kube-controller-manager`

###### Does enabling the feature change any default behavior?

<!--
Any change of default behavior may be surprising to users or break existing
automations, so be extremely careful here.
-->

No.

###### Can the feature be disabled once it has been enabled (i.e. can we roll back the enablement)?

<!--
Describe the consequences on existing workloads (e.g., if this is a runtime
feature, can it break the existing applications?).

Feature gates are typically disabled by setting the flag to `false` and
restarting the component. No other changes should be necessary to disable the
feature.

NOTE: Also set `disable-supported` to `true` or `false` in `kep.yaml`.
-->

The feature can be disabled in Alpha and Beta versions
by restarting kube-apiserver and kube-controller-manager with the feature-gate off.

As described in [Upgrade / Downgrade Strategy](#upgrade-/-downgrade-strategy),
during the feature-gate off, all existing `ContainerResource` will be ignored by the HPA controller.

In terms of Stable versions, users can choose to opt-out by not setting the
`ContainerResource` type metric in their HPA.

###### What happens if we reenable the feature if it was previously rolled back?

HPA with `ContainerResource` type metric can be created and can be handled by HPA controller.

If there have been HPAs with the `ContainerResource` type metric created before the roll back,
those `ContainerResource` is ignored during the feature gate off, but will be handled by the HPA controller again after reenabling.

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

No. But, the tests to confirm the behavior on switching the feature gate will be added by beta. ([issue](https://github.com/kubernetes/kubernetes/issues/115467))

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

When a rollout fail, it shouldn't impact already running HPAs because it's an opt-in feature,
and users need to set `ContainerResource` metric to use this feature. 

When a rollback fail for kube-controller-manager, HPA controller will continue to handle `ContainerResource` metric in HPAs.
When a rollback fail for kube-apiserver, but success kube-controller-manager, 
HPA controller will just ignore `ContainerResource` metric in HPAs.

###### What specific metrics should inform a rollback?

<!--
What signals should users be paying attention to when the feature is young
that might indicate a serious problem?
-->

- The container resource metric takes much longer time compared to other metrics.
which can be monitored via the 1st metrics described in [What are the SLIs (Service Level Indicators) an operator can use to determine the health of the service?](#what-are-the-slis-service-level-indicators-an-operator-can-use-to-determine-the-health-of-the-service) section.
- Increase the overall performance of HPA controller 
which can be monitored via the 2nd metrics described in [What are the SLIs (Service Level Indicators) an operator can use to determine the health of the service?](#what-are-the-slis-service-level-indicators-an-operator-can-use-to-determine-the-health-of-the-service) section.
- Many error occurrence on the container resource metrics
which can be monitored via the 3rd metrics described in [What are the SLIs (Service Level Indicators) an operator can use to determine the health of the service?](#what-are-the-slis-service-level-indicators-an-operator-can-use-to-determine-the-health-of-the-service) section.

###### Were upgrade and rollback tested? Was the upgrade->downgrade->upgrade path tested?

<!--
Describe manual testing that was done and the outcomes.
Longer term, we may want to require automated upgrade/rollback tests, but we
are missing a bunch of machinery and tooling and can't do that now.
-->

Not yet.
But, as described in [Are there any tests for feature enablement/disablement?](#Are-there-any-tests-for-feature-enablement/disablement?), the tests to confirm the behavior on switching the feature gate will be added. ([issue](https://github.com/kubernetes/kubernetes/issues/115467))

###### Is the rollout accompanied by any deprecations and/or removals of features, APIs, fields of API types, flags, etc.?

<!--
Even if applying deprecation policies, they may still surprise some users.
-->

No.

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

- The operator can observe the execution of the computation for the container metrics through the 1st metrics described in [What are the SLIs (Service Level Indicators) an operator can use to determine the health of the service?](#what-are-the-slis-service-level-indicators-an-operator-can-use-to-determine-the-health-of-the-service) section.
- The operator can query HPAs with `hpa.spec.metrics.containerResource` field set.

###### How can someone using this feature know that it is working for their instance?

<!--
For instance, if this is a pod-related feature, it should be possible to determine if the feature is functioning properly
for each individual pod.
Pick one more of these and delete the rest.
Please describe all items visible to end users below with sufficient detail so that they can verify correct enablement
and operation of this feature.
Recall that end users cannot usually observe component logs or access metrics.
-->

- [x] Events
  - `SuccessfulRescale` event with `memory/cpu/etc resource utilization (percentage of request) above/below target`
    - Note that we cannot know if this reason is due to the `Resource` metric or `ContainerResource` in the current implementation. We'll change this reason for `ContainerResource` to `memory/cpu/etc container resource utilization (percentage of request) above/below target` so that we can distinguish.
- [x] API .status
  - When something wrong with the container metrics, `ScalingActive` condition will be false with `FailedGetContainerResourceMetric` reason.

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

N/A

###### What are the SLIs (Service Level Indicators) an operator can use to determine the health of the service?

<!--
Pick one more of these and delete the rest.
- [ ] Metrics
  - Metric name:
  - [Optional] Aggregation method:
  - Components exposing the metric:
- [ ] Other (treat as last resort)
  - Details:
-->

HPA controller have no metrics in it now. 
The following metrics will be implemented by beta. ([issue](https://github.com/kubernetes/kubernetes/issues/115639))
1. How long does each metric type take to compute the ideal replica num.
  - so that users can confirm the container resource metric doesn't take long time compared to other metrics.
2. How long does the HPA controller take to complete reconcile one HPA object.
  - so that users can confirm the container resource metric doesn't increse the whole time of scaling.
3. Provide the metric to show error occurrence for each metric.
  - so that users can confirm no much error occurrence on the container resource metric.

###### Are there any missing metrics that would be useful to have to improve observability of this feature?

<!--
Describe the metrics themselves and the reasons why they weren't added (e.g., cost,
implementation difficulties, etc.).
-->

Yes. We're planning to implement the metrics described in [What are the SLIs (Service Level Indicators) an operator can use to determine the health of the service?](#what-are-the-slis-service-level-indicators-an-operator-can-use-to-determine-the-health-of-the-service) section.

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

Yes.
The HPA requires the `metrics.k8s.io` APIs to be available in the cluster to operate. This API is served by the Metrics Server,
without Metrics Server autoscaling on container resource metrics will not work. 
If there are multiple metrics defined and one is not available, scale up will
continue but scale down will not (for safety).

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

No.

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

No. 

###### Will enabling / using this feature result in increasing time taken by any operations covered by existing SLIs/SLOs?

<!--
Look at the [existing SLIs/SLOs].

Think about adding additional work or introducing new steps in between
(e.g. need to do X to start a container), etc. Please describe the details.

[existing SLIs/SLOs]: https://git.k8s.io/community/sig-scalability/slos/slos.md#kubernetes-slisslos
-->

No.

###### Will enabling / using this feature result in non-negligible increase of resource usage (CPU, RAM, disk, IO, ...) in any components?

<!--
Things to keep in mind include: additional in-memory state, additional
non-trivial computations, excessive access to disks (including increased log
volume), significant amount of data sent and/or received over network, etc.
This through this both in small and large cases, again with respect to the
[supported limits].

[supported limits]: https://git.k8s.io/community//sig-scalability/configs-and-limits/thresholds.md
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

The autoscaling based on `ContainerResource` is unavailable 
because the HPA controller cannot get HPA object.

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

- Failed to get container resource metric.
  - Detection:  `ScalingActive: false` condition with `FailedGetContainerResourceMetric` reason.
  - Mitigations: remove failed `ContainerResource` in HPAs.
  - Diagnostics: Related errors should be printed as the messages of `ScalingActive: false`. 
  - Testing: https://github.com/kubernetes/kubernetes/blob/0e3818e02760afa8ed0bea74c6973f605ca4683c/pkg/controller/podautoscaler/replica_calculator_test.go#L451

###### What steps should be taken if SLOs are not being met to determine the problem?

## Implementation History

* 2020-04-03 Initial KEP merged
* 2020-10-23 Implementation merged

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

There's an alternative way to scale on container-level metrics without introducing ContainerResource metrics. 

Users can export resource consumption metrics from containers on their own to an external metrics source and then configure HPA based on this external metric. 
However this is cumbersome and results in delayed scaling decisions as using the external metrics path typically adds latency compared to in-cluster resource metrics path.

## Infrastructure Needed (Optional)

<!--
Use this section if you need things from the project/SIG. Examples include a
new subproject, repos requested, or GitHub details. Listing these here allows a
SIG to get the process for these resources started right away.
-->
