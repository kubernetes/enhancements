# Graduate v2beta2 Autoscaling API to GA

**Note:** When your KEP is complete, all of these comment blocks should be removed.

To get started with this template:

- [ x ] **Pick a hosting SIG.**
  Make sure that the problem space is something the SIG is interested in taking
  up. KEPs should not be checked in without a sponsoring SIG.
- [ x ] **Create an issue in kubernetes/enhancements**
  When filing an enhancement tracking issue, please make sure to complete all
  fields in that template. One of the fields asks for a link to the KEP. You
  can leave that blank until this KEP is filed, and then go back to the
  enhancement and add the link.
- [ x ] **Make a copy of this template directory.**
  Copy this template into the owning SIG's directory and name it
  `NNNN-short-descriptive-title`, where `NNNN` is the issue number (with no
  leading-zero padding) assigned to your enhancement above.
- [ x ] **Fill out as much of the kep.yaml file as you can.**
  At minimum, you should fill in the "Title", "Authors", "Owning-sig",
  "Status", and date-related fields.
- [ x ] **Fill out this file as best you can.**
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
## Table of Contents

<!-- toc -->
- [Release Signoff Checklist](#release-signoff-checklist)
- [Summary](#summary)
- [Motivation](#motivation)
  - [Goals](#goals)
  - [Non-Goals](#non-goals)
- [Proposal](#proposal)
  - [Implementation Details](#implementation-details)
  - [Risks and Mitigations](#risks-and-mitigations)
- [Design Details](#design-details)
  - [Renames](#renames)
  - [Test Plan](#test-plan)
  - [Graduation Criteria](#graduation-criteria)
  - [Version Skew Strategy](#version-skew-strategy)
  - [Upgrade/Downgrade Strategy](#upgradedowngrade-strategy)
- [Production Readiness Review Questionnaire](#production-readiness-review-questionnaire)
  - [Requirements for migration](#requirements-for-migration)
  - [Feature Enablement and Rollback](#feature-enablement-and-rollback)
  - [Rollout, Upgrade and Rollback Planning](#rollout-upgrade-and-rollback-planning)
  - [Monitoring Requirements](#monitoring-requirements)
  - [Dependencies](#dependencies)
  - [Scalability](#scalability)
  - [Troubleshooting](#troubleshooting)
- [Implementation History](#implementation-history)
- [Drawbacks](#drawbacks)
- [Alternatives](#alternatives)
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
  - [ ] (R) Ensure GA e2e tests for meet requirements for [Conformance Tests](https://github.com/kubernetes/community/blob/master/contributors/devel/sig-architecture/conformance-tests.md)
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

This document outlines required steps to graduate autoscaling v2beta2 API to GA.

## Motivation
The HPA v2 series APIs were first introduced in November, 2016 (5 years ago).
The primary feature of the v2 series is adding support for multiple and custom metrics. The structure was improved
slightly in the v2beta2 API which became available in May 2018 and has remained largely unchanged since then.
The v2beta2 API has been used extensively and informally treated as stable.
The motivation for this KEP is to push it over the line to make it formally so.

### Goals

* Promote all of v2beta2 to stable
* HPA behavior and container resource targets have E2E tests in order to meet stable requirements.
* Deprecate v2beta2 as soon as v2 stable is landed
* Deprecate v2beta1 immediately
* Container Resource Targets
    * Rename `Resource` to `PodResource` to match new ContainerResource target
* Behavior
    Rename behavior select policy values from `Min` to `MinChange` and `Max` to `MaxChange`
    
### Non-Goals

* Promote scale-to-zero feature as a part of this effort. Since it requires additional effort
  to deprecate the special flag meaning of scaling subresource `replicas=0` (disable autoscaling).
  [Progress](https://github.com/kubernetes/enhancements/issues/2021) has been made. However,
  it is not part of HPA v2 stable effort since APIs cannot introduce breaking changes.
  

## Proposal


### Implementation Details


### Risks and Mitigations

* v1-v2 [conversion loss](https://github.com/kubernetes/kubernetes/issues/80481) of multiple CPU targets
* v2beta1 has the significant amount of boilerplate and overhead in maintaining conversion routines for multiple public APIs.


## Design Details

### Renames

* Rename `Min` and `Max` with respective `MinChange` and `MaxChange` in v2 stable to eliminate confusion with `SelectPolicy` [enumeration](https://github.com/kubernetes/api/blob/2c3c141c931c0ab1ce1396c3152c72852b3d37ee/autoscaling/v2beta2/types.go#L149-L156)
* Rename the value `Disabled` with `ScalingDisabled` for better [understanding](https://github.com/kubernetes/kubernetes/pull/95647#discussion_r507563282)
* Rename Container Resource Targets `Resource` to `PodResource` to match new ContainerResource target

### Test Plan

* Add e2e tests for HPA behavior
* The KEP [test plan](https://github.com/kubernetes/enhancements/blob/15d330a932e0aae220ff719c391f7d815492088f/keps/sig-autoscaling/20190307-configurable-scale-velocity-for-hpa.md#test-plan) includes unit tests
* Add e2e tests for container resource targets
* Add conformance tests

### Graduation Criteria

The following code changes must be made for graduating to GA

* Move API objects to `v2` and support conversion internally

* Add behavior and container target E2E tests.


### Version Skew Strategy

### Upgrade/Downgrade Strategy

All HPA APIs to date are forward and backward conversion without loss by serializing
all unsupported fields to annotations. HPA v2 stable will be the same, verified by unit
tests.

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
### Requirements for migration
All HPA objects are losslessly converted between API versions, which are just a view of the data on disk.
Neither the deprecation of `v2beta1` nor the addition of `v2` requires any changes or conversion on the
server side.  They will continue being stored in disk in v1 format as always, with new v2 fields serialized
to annotiations.

However any HPA objects stored in the user's code repository (all your YAML files) must stop using the
v2beta1 format.  You should migrate all your HPA objects to the v2 format.  See the types.go files or just
run `kubectl get hpa.v2.autoscaling -oyaml` to see your objects in the v2 format.

### Feature Enablement and Rollback
N/A

###### Does enabling the feature change any default behavior?

<!--
Any change of default behavior may be surprising to users or break existing
automations, so be extremely careful here.
-->

###### Can the feature be disabled once it has been enabled (i.e. can we roll back the enablement)?

<!--
Describe the consequences on existing workloads (e.g., if this is a runtime
feature, can it break the existing applications?).

NOTE: Also set `disable-supported` to `true` or `false` in `kep.yaml`.
-->
The feature can be enabled by adding `autoscaling/v2` to the `--runtime-config` flag:
https://github.com/kubernetes/kubernetes/blob/ea0764452222146c47ec826977f49d7001b0ea8c/staging/src/k8s.io/apiserver/pkg/server/options/api_enablement.go#L45

Adding `api/all` will also include `autoscaling/v2`.

The feature can be disabled by removing the `--runtime-config` entry.

###### What happens if we reenable the feature if it was previously rolled back?

###### Are there any tests for feature enablement/disablement?

<!--
The e2e framework does not currently support enabling or disabling feature
gates. However, unit tests in each component dealing with managing data, created
with and without the feature, are necessary. At the very least, think about
conversion tests if API types are being modified.
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
-->
The HPA requires the `metrics.k8s.io` APIs to be available in the cluster to operate. This API is served by the
Metrics Server. An operator can verify the Metrics Server is available to provide resource metrics to the HPA by running
the command `kubectl get apiservices` and looking for the status of `v1beta1.metrics.k8s.io` (version subject to change).
Operators should take care to make sure Metrics Server is up and running to maintain resource autoscaling.

The v2 HPA requires the `custom.metrics.k8s.io` and `external.metrics.k8s.io` APIs as well to retrieve custom and
external metrics. There is no default implementation of these APIs and cluster operators must install an "adapter" for
their metrics backend (e.g. [Prometheus](https://github.com/kubernetes-sigs/prometheus-adapter)).

An operator can verify the adapter is working properly by running the same kubectl for apiservices and looking for the
`v1beta1.custom.metrics.k8s.io` and `v1beta1.external.metrics.k8s.io` APIs (usually served by the same adapter).
Care should be taken to ensure the adapter and specific metrics backend is available to maintain custom metric autoscaling.

###### How can an operator determine if the feature is in use by workloads?

<!--
Ideally, this should be a metric. Operations against the Kubernetes API (e.g.,
checking if there are objects with field X set) may be a last resort. Avoid
logs or events for this purpose.
-->
All HPA objects are stored in v1 format on disk. They are up converted the requested version and down converted upon update.
The document on how to run [HPA](https://kubernetes.io/docs/tasks/run-application/horizontal-pod-autoscale/) includes
quite a bit of background,algorithm details, and some good [operator notes](https://kubernetes.io/docs/tasks/run-application/horizontal-pod-autoscale/#support-for-metrics-apis).

###### How can someone using this feature know that it is working for their instance?

<!--
For instance, if this is a pod-related feature, it should be possible to determine if the feature is functioning properly
for each individual pod.
Pick one more of these and delete the rest.
Please describe all items visible to end users below with sufficient detail so that they can verify correct enablement
and operation of this feature.
Recall that end users cannot usually observe component logs or access metrics.
-->

- [ x ] Events
  - Event Reason: 
  The event type `Normal`, reason `SuccessfulRescale`, note `New size: N; reason: FOO` indicates autoscaling is operating normally.
  Abnormal events type `Warning` include reasons such as `FailedRescale` and `FailedComputeMetricsReplicas` and will
  include details about the error in the note.
- [ x ] API .status
  - Condition name: 
  There are three condition types which indicate the operating status of the HPA.  They are `ScalingEnabled`, `AbleToScale`
  and `ScalingLimited` (see type [comments](https://pkg.go.dev/k8s.io/api/autoscaling/v2beta2#HorizontalPodAutoscalerConditionType))
  Under normal operating circumstances `ScalingEnabled` and `AbleToScale` should be status `true`, indicating the HPA is
  successfully reconciling the scale. `ScalingLimited` indicates user configuration is limiting the "ideal" scale with a
  minimum, maximum, rate or delay. Which limit is the cause will be indicated in the message.
  It's normal for this to be `true` or `false` periodically.
  - Other field: 
- [ x ] Other (treat as last resort)
  - Details:
  The HPA status includes the current observed metric values, one for each given target. Using these
  values an operator can verify the HPA is maintaining the desired target for the dominant metric.
  The operator can also see the number of pods the HPA observed under `status.currentReplicas` and the most
  recent recommendation under `status.desiredReplicas`.
  The latest observed generation is echoed back in status so an operator can verify the HPA is keeping up-to-date with
  configuration changes.

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
The HPA requires the `metrics.k8s.io` APIs to be available in the cluster to operate,This API is served by the Metrics Server,
without Metrics Server autoscaling on resource metrics will not work. Without the a custom metrics adapter and the backing metric store running, custom
and external metrics will not work. If there are multiple metrics defined and one is not available, scale up will
continue but scale down will not (for safety).

### Scalability

<!--
For alpha, this section is encouraged: reviewers should consider these questions
and attempt to answer them.

For beta, this section is required: reviewers must answer these questions.

For GA, this section is required: approvers should be able to confirm the
previous answers based on experience in the field.
-->
The HPA v2 APIs allow users to configure multiple metrics, each with a separate target. A recommendation is calculated
for each metric and the largest recommendation is used.  The more metrics are added to a given HPA the longer it will
take to reconcile. The HPA is single-threaded processing recommendations one-at-a-time. When default reconciliation
period is 15 seconds.  If there is too much work to do reconciliation will slow down and happen less frequently than
every 15 seconds.  This will cause autoscaling to be less responsive at high scale.

Previously v1 scaled along two dimensions, number of HPA and number of pods selected by each HPA (linearly).
Now it will scale with the number of metrics defined in HPAs and the number of pods selected each metric (linearly).

Additionally, v2 adds a behavior structure which allows the user configure that rate and delay of scaling and down.
Enforcing these constraints require storing previous recommendations and scaling events in memory. The longer the
configured interval the more memory is used. The maximum window allows is 60 minutes ([code](https://pkg.go.dev/k8s.io/api/autoscaling/v2beta2#HPAScalingRules))
so 240 recommendations / events per configured metric. Each recommendation is an `int32` and `time.Time`.
Each scaling event is an `int32`, a `time.Time` and a `bool` ([code](https://pkg.go.dev/k8s.io/api/autoscaling/v2beta2#HPAScalingRules))
so the memory footprint is relatively small.
It will scale linearly with the number of metrics defined and the size of the HPA's configured window.

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
No, not in comparison to using the existing v2beta2 APIs, but of course using HPA results in new API calls as described above.

###### Will enabling / using this feature result in introducing new API types?

<!--
Describe them, providing:
  - API type
  - Supported number of objects per cluster
  - Supported number of objects per namespace (for namespace-scoped objects)
-->
Yes.  It will introduce the new autoscaling/v2 API types.

###### Will enabling / using this feature result in any new calls to the cloud provider?

<!--
Describe them, providing:
  - Which API(s):
  - Estimated increase:
-->
Configuring custom metrics (the difference from v1 to v2) will result in API calls to the installed custom metrics adapter
and the backing metrics store (which might be hosted in the cloud provider). These calls will happen every 15 seconds
for each configured metric. Targets of type Value will retrieve for a single metric.
Targets of type AverageValue will retrieve a metric for each pod.

###### Will enabling / using this feature result in increasing size or count of the existing API objects?

<!--
Describe them, providing:
  - API type(s):
  - Estimated increase in size: (e.g., new annotation of size 32B)
  - Estimated amount of new objects: (e.g., new Object X for every existing Pod)
-->
No. Data on disk remains as-in, in v1 format.

###### Will enabling / using this feature result in increasing time taken by any operations covered by existing SLIs/SLOs?

<!--
Look at the [existing SLIs/SLOs].

Think about adding additional work or introducing new steps in between
(e.g. need to do X to start a container), etc. Please describe the details.

[existing SLIs/SLOs]: https://git.k8s.io/community/sig-scalability/slos/slos.md#kubernetes-slisslos
-->
No

###### Will enabling / using this feature result in non-negligible increase of resource usage (CPU, RAM, disk, IO, ...) in any components?

<!--
Things to keep in mind include: additional in-memory state, additional
non-trivial computations, excessive access to disks (including increased log
volume), significant amount of data sent and/or received over network, etc.
This through this both in small and large cases, again with respect to the
[supported limits].

[supported limits]: https://git.k8s.io/community//sig-scalability/configs-and-limits/thresholds.md
-->
No

### Troubleshooting

<!--
This section must be completed when targeting beta to a release.

The Troubleshooting section currently serves the `Playbook` role. We may consider
splitting it into a dedicated `Playbook` document (potentially with some monitoring
details). For now, we leave it here.
-->

###### How does this feature react if the API server and/or etcd is unavailable?
If the API server or etcd are not available the HPA will not reconcile the scale subresource to the target metrics.
This feature depends on other APIs served not from etcd but Metrics Server and custom metrics adapters.
These are referenced in another section for monitoring to keep them alive. When one of the metrics is unavailable
(e.g. a custom metric along side a resource metric) the HPA will continue to scale up if the other metric indicates
to do so,this is for safety. However if one of the metrics is unavailable the HPA will not scale down
in case the unavailable metric would have prevented a scale down. This is again for safety.

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

## Implementation History

* HPA v1
    * HPA v1 proposal [merged](https://github.com/kubernetes/kubernetes/pull/12344) on Aug 13, 2015.
        * [Design](https://github.com/kubernetes/kubernetes/pull/12859)
    * Graduated to [beta](https://github.com/kubernetes/kubernetes/pull/15706) on Oct 15, 2015
    * Graduated to [stable](https://github.com/kubernetes/kubernetes/pull/20501) on Feb 2, 2016 as v1
    
* HPA v2
    * HPA v2 [addition](https://github.com/kubernetes/kubernetes/pull/36033) on Nov 2, 2016 for v2alpha1
        * [Design](https://github.com/kubernetes/enhancements/issues/117)
    * Graduated to [beta](https://github.com/kubernetes/kubernetes/pull/50708) on Aug 15, 2017 as v2beta1
    * Released second beta version [v2beta2](https://github.com/kubernetes/kubernetes/pull/64097) on May 21, 2018
        * [Design](https://github.com/kubernetes/community/pull/2055)
        
* Scale-to-zero
    * scale-to-zero [addition](https://github.com/kubernetes/kubernetes/pull/74526) to external metrics on Jul 16, 2019
      for alpha [feature](https://github.com/kubernetes/kubernetes/issues/69687#issuecomment-467082733) feature
      
* HPA Controls
    * HPA behavior controls [addition](https://github.com/kubernetes/kubernetes/pull/74525) on Dec 11, 2019 for v2beta2
      [API](https://github.com/kubernetes/enhancements/blob/master/keps/sig-autoscaling/20190307-configurable-scale-velocity-for-hpa.md)
      
* Container Resource Targets
    * [Proposed](https://github.com/kubernetes/enhancements/blob/master/keps/sig-autoscaling/0001-container-resource-autoscaling.md) on Mar 30, 2020
      (Implementation is pending)

## Drawbacks

## Alternatives