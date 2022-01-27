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
# KEP-2891: Simplified Scheduler Config

<!-- toc -->
- [Release Signoff Checklist](#release-signoff-checklist)
- [Summary](#summary)
- [Motivation](#motivation)
  - [Goals](#goals)
  - [Non-Goals](#non-goals)
- [Proposal](#proposal)
  - [User Stories (Optional)](#user-stories-optional)
    - [Story 1](#story-1)
    - [Story 2](#story-2)
    - [Story 3](#story-3)
  - [Risks and Mitigations](#risks-and-mitigations)
- [Design Details](#design-details)
  - [Test Plan](#test-plan)
  - [Graduation Criteria](#graduation-criteria)
    - [Alpha](#alpha)
    - [Beta](#beta)
    - [GA](#ga)
    - [Deprecation](#deprecation)
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

- [x] (R) Enhancement issue in release milestone, which links to KEP dir in [kubernetes/enhancements] (not the initial KEP PR)
- [x] (R) KEP approvers have approved the KEP status as `implementable`
- [x] (R) Design details are appropriately documented
- [x] (R) Test plan is in place, giving consideration to SIG Architecture and SIG Testing input (including test refactors)
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

This KEP defines a simplified field for end-users to configure Scheduler plugins
which use multiple extension points easily. It is based on a new extension point that
plugin developers can implement in their plugins. This extension point will automatically
register a plugin for all (or some) of the scheduling cycle extension points that the
plugin implements. Thus simplifying the configuration required for cluster administrators
while preserving the development benefits of the scheduling framework.


## Motivation

The [Scheduling Framework](https://github.com/kubernetes/enhancements/tree/master/keps/sig-scheduling/624-scheduling-framework) was designed with
developers of custom schedulers in mind to make the scheduler more extendable. This design
provided several extension points used by developers to implement their custom plugins into
the scheduling cycle. Some extension points, like `PreFilter` and `PreScore`, are only
used for data processing or other internal functions not directly relevant to the end user.

However these "behind-the-scenes" endpoints must still be configured in the `KubeSchedulerConfiguration`
provided by the user for every plugin that uses them. For the average user this is an unnecessary
and confusing step -- they only care that the plugin works, not how it works, and so
internal processing steps are usually irrelevant to them.

Therefore, a way for users to simply toggle all of a plugin's extension points
(or a subset of the extension points as defined necessary by the plugin developer)
would remove complexity and the potential for misconfiguration.

### Goals

* Provide an option for simplified plugin config from the end-user's perspective
* Enable this config in all in-tree default scheduler plugins

### Non-Goals

* Refactor or introduce any breaking changes to the scheduling framework

## Proposal

A new extension point, `MultiPoint` (final name TBD) will be made available to
users and developers which handles plugin registration for multiple extension
points simultaneously.

From the user's perspective, this will be a new (optional) field in the scheduler's
config to enable or disable plugins across multiple extension points.


### User Stories (Optional)

#### Story 1

As a cluster administrator, I wish to enable plugin `XYZ` for all applicable extension points (ie, `Filter` 
and `Score`). I do not care about internal implementation or extension points such as 
`PreFilter`, nor do I want to have to update my config should plugin `XYZ` change its 
implementation within the scheduling framework to use different extension points.

```yaml
apiVersion: kubescheduler.config.k8s.io/v1beta3
kind: KubeSchedulerConfiguration
profiles:
  - schedulerName: default-scheduler
    plugins:
      multiPoint:
        enabled:
        - name: "XYZ"
          weight: 3
```

#### Story 2

As a cluster administrator, I wish to enable plugin `XYZ` for only *some* of its applicable extension 
points (ie `Filter` but *not* `Score`). I still do not care about extension points such 
as `PreFilter`. I want to have control over which subsets of extension points are enabled.

```yaml
apiVersion: kubescheduler.config.k8s.io/v1beta3
kind: KubeSchedulerConfiguration
profiles:
  - schedulerName: default-scheduler
    plugins:
      multiPoint:
        enabled:
        - name: "XYZ"
      preScore:
        disabled:
        - "*"
      score:
        disabled:
        - "*"
```

#### Story 3

As a plugin developer, I want users to be able to easily enable my plugin for all applicable 
extension points (or a subset of extension points as defined either by me or my users' needs). 
I want this available with minimal code updates required from me.

### Risks and Mitigations

* out-of-tree plugins may be developed by anyone, so the information that is 
passed to them and received from them should be kept to a minimum. However these are 
unsupported so security is ultimately up to users to be aware of.
* UX will need to be carefully examined, as a new config field can introduce confusing 
and unexpected outcomes if different combinations of options are not well-defined.

## Design Details

An example of the new user-facing profile config would be:

```yaml
apiVersion: kubescheduler.config.k8s.io/v1beta3
kind: KubeSchedulerConfiguration
profiles:
  - schedulerName: default-scheduler
    plugins:
      multiPoint:
        enabled:
        - name: "PodTopologySpread" ## PreFilter/Filter/PreScore/Score
          weight: 3
        - name: "VolumeBinding" ## PreFilter/Filter/Reserve/PreBind
        - name: "PrioritySort" ## QueueSort
```

Specific sections (eg, `Score`) could still be entirely disabled like so:

```yaml
apiVersion: kubescheduler.config.k8s.io/v1beta3
kind: KubeSchedulerConfiguration
profiles:
  - schedulerName: default-scheduler
    plugins:
      multiPoint:
        enabled:
        - name: "PodTopologySpread" ## PreFilter/Filter
          weight: 3
        - name: "VolumeBinding" ## PreFilter/Filter/Reserve/PreBind
        - name: "PrioritySort" ## QueueSort
      preScore:
        disabled:
        - "*"
      score:
        disabled:
        - "*"
```
(Note that in the above, now `PodTopologySpread` will not be registered for `Score` or `PreScore`)

Default plugins will be updated in the kube-scheduler API to be defined via MultiPoint. This 
will then translate to users seeing the MultiPoint configuration when printing the default config 
with the `--write-config-to` flag.

To do this, we must first define a new extension point `MultiPoint` in [`Plugins`](https://github.com/kubernetes/kube-scheduler/blob/bb2d43a/config/v1beta2/types.go#L154-L159)
```go
type Plugins struct {
  MultiPoint PluginSet `json:"multiPoint,omitEmpty"`
...
}
```

We define the `MultiPoint` extension as transparently applicable to all plugins, using 
type casting to automatically initialize every plugin for every extension point they implement 
by checking each one. The benefit to this approach is that all plugins are capable of being registered 
this way without any additional plugin code. The drawbacks of this are less defined control for 
developers and minor changes to config semantics for users.

Modify [`f.pluginsNeeded`](https://github.com/kubernetes/kubernetes/blob/dc079acc2be603fbc2065e2748c79bb5e2c3453d/pkg/scheduler/framework/runtime/framework.go#L1164) 
to additionally parse MultiPoint plugins so the function 
properly builds the map of `pluginName->New function`. Otherwise, we will not have a 
reference to any plugins that are only enabled through MultiPoint later when we 
try to unroll them to their individual extension points.

Either before or after [regular extension point plugin initialization](https://github.com/kubernetes/kubernetes/blob/dc079acc2be603fbc2065e2748c79bb5e2c3453d/pkg/scheduler/framework/runtime/framework.go#L349-L353) attempt 
to initialize all `MultiPoint` plugins as all extension points (this is the section of 
code that utilizes the factory function stored in the `frameworkImpl` struct)

This code is similar to the existing [`updatePluginList`](https://github.com/kubernetes/kubernetes/blob/dc079acc2be603fbc2065e2748c79bb5e2c3453d/pkg/scheduler/framework/runtime/framework.go#L418)
function except for 2 notable differences:

1. It does not return an error if a plugin doesn't implement an extension point, it just continues
2. It is only looping through the MultiPoint config field for each internal extension point

With this implementation, there is no option for plugins or users to define a subset of extension 
points to enable or disable should we deprecate the existing regular extension points. This is because 
the plugins themselves do not have control over when their functions are called, and we have cast them 
into each extension point. At that point the framework, not the plugin, takes over.

Disabling of individual subsets could be preserved using modified config semantics as below:

|   |   |enabled: xyz|disabled: xyz|enabled: *|disabled: *|
|---|---|------------|-------------|----------|-----------|
|default|regular|reorder plugins/set weight|disable plugin xyz for this extension point|no-op|disable all plugins of the point|\
|   |multi-point|enable plugin xyz for all extension points|disable plugin xyz for all extension points|no-op|disable all default plugins for all extension points|
|non-default|regular|enable plugin xyz|disable plugin xyz for this extension point|no-op|disable all plugins of this extension point|
|   |multi-point|enable plugin xyz for all extension points|no-op|no-op|no-op|

The key changes are in `non-default/regular`, where now `disabled` would take effect on 
non-default plugins. Currently, non-default plugins are not run unless they are `enabled`, so `disabled` 
is a no-op. But to provide subset control of multipoint plugins, we could modify this to allow disabling 
a particular extension point. This would need additional code changes to the internal processing of 
`disabled` fields, as they are currently only parsed against default plugin initialization.


### Test Plan

* Unit and integration tests to ensure multiPoint plugin configs are 
translated to their expected expanded config
* Existing scheduler e2es will ensure continued functioning of the 
scheduler framework and default plugins

### Graduation Criteria

#### Alpha

- N/A (this feature is targeted directly at beta in the `v1beta3` scheduler API, see https://github.com/kubernetes/enhancements/blob/master/keps/sig-scheduling/785-scheduler-component-config-api/README.md)

#### Beta

- Developer and user feedback on the implementation details
- Test coverage
- API is fully defined and implemented

#### GA

- Deprecation timeline for `v1beta3` in line with the GA release of scheduler `v1` API
- Additional feedback from users and developers

#### Deprecation

- Deprecation of the `v1beta3` API will match the scheduler's [ComponentConfig graduation to GA](https://github.com/kubernetes/enhancements/issues/785)

### Upgrade / Downgrade Strategy

Upgrades will be unaffected for as long as the existing component config fields are supported

Downgrades may be more difficult, since there will be no inherent mapping of the simplified config to the
multi-point extensions, especially for out-of-tree plugins. However the scheduler is capable of writing its
internal config to a file usable by the user post-downgrade

### Version Skew Strategy

N/A - this does not affect other components

## Production Readiness Review Questionnaire

### Feature Enablement and Rollback

###### How can this feature be enabled / disabled in a live cluster?

- [ ] Feature gate (also fill in values in `kep.yaml`)
  - Feature gate name:
  - Components depending on the feature gate:
- [x] Other
  - Describe the mechanism: This will be a new field in the config which encapsulates a
  subset of existing fields. Setting it implies enablement, leaving it unset does not.
  - Will enabling / disabling the feature require downtime of the control
    plane? Momentary downtime for the scheduler to load a new config, but this is not
    specific to the feature (any config change requires a re-deployment of the scheduler)
  - Will enabling / disabling the feature require downtime or reprovisioning
    of a node? (Do not assume `Dynamic Kubelet Config` feature is enabled). No

###### Does enabling the feature change any default behavior?

No

###### Can the feature be disabled once it has been enabled (i.e. can we roll back the enablement)?

The field option can be unset and the scheduler re-deployed with existing config options.

###### What happens if we reenable the feature if it was previously rolled back?

N/A

###### Are there any tests for feature enablement/disablement?

Scheduler unit tests will confirm translation and functioning of the config options as intended

### Rollout, Upgrade and Rollback Planning

###### How can a rollout or rollback fail? Can it impact already running workloads?

Misconfigured plugins could cause the scheduler to CrashLoop as it fails to initialize the framework.

This does not have any effect on running workloads.

###### What specific metrics should inform a rollback?

Scheduler CrashLoops could indicate a failure to properly initialize the plugin registry

###### Were upgrade and rollback tested? Was the upgrade->downgrade->upgrade path tested?

This will be manually tested

###### Is the rollout accompanied by any deprecations and/or removals of features, APIs, fields of API types, flags, etc.?

No

### Monitoring Requirements

###### How can an operator determine if the feature is in use by workloads?

This feature has no functional change to workloads. The scheduler's functionality can be
measured by its throughput and startup time.

###### How can someone using this feature know that it is working for their instance?

- [ ] Events
  - Event Reason: 
- [ ] API .status
  - Condition name: 
  - Other field: 
- [x] Other (treat as last resort)
  - Details: Pods are scheduled as expected indicates functioning kube-scheduler.
  Users can measure this with high instances of pods being marked "Unschedulable" and/or 
  with no `NodeName` assigned. The `schedule_attempts_total` metric can be used to reference 
  how successful the scheduling attempts have been.

###### What are the reasonable SLOs (Service Level Objectives) for the enhancement?

Relevant metrics would only be related to the scheduler's startup time, as this enhancement is a 
one-time operator in the scheduler's lifecycle. This may have an effect on rolling restarts and 
upgrades/downgrades.

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

###### Does this feature depend on any specific services running in the cluster?

No, this API is only consumed by the scheduler. Thus it is inherent that the scheduler is already running.

### Scalability

###### Will enabling / using this feature result in any new API calls?

No

###### Will enabling / using this feature result in introducing new API types?

No

###### Will enabling / using this feature result in any new calls to the cloud provider?

No

###### Will enabling / using this feature result in increasing size or count of the existing API objects?

One new field in the `KubeSchedulerConfig.spec.profiles` object, with sub-lists for enable/disable containing
at most several dozen strings of plugin names.

###### Will enabling / using this feature result in increasing time taken by any operations covered by existing SLIs/SLOs?

Scheduler startup time may be affected as the new multi-point extension needs to be translated to the necessary
existing extension points. This could be controlled through efficient design of the translation.

###### Will enabling / using this feature result in non-negligible increase of resource usage (CPU, RAM, disk, IO, ...) in any components?

No

### Troubleshooting

<!--
This section must be completed when targeting beta to a release.

The Troubleshooting section currently serves the `Playbook` role. We may consider
splitting it into a dedicated `Playbook` document (potentially with some monitoring
details). For now, we leave it here.
-->

###### How does this feature react if the API server and/or etcd is unavailable?

The scheduler would not function anyway, so this is irrelevant

###### What are other known failure modes?

Invalid parsing of the config could lead to a scheduler crash, or improper pod placement.

###### What steps should be taken if SLOs are not being met to determine the problem?

## Implementation History

- 2021-08-25: KEP created

## Drawbacks

* **Additional code**: This proposal requires additional code (and with it additional
maintenance) on top of the existing scheduling framework and all of its components
* **Confusion between existing fields**: Users may be confused by the coexistence of the
new simple config and the current set of extension points.

## Alternatives

* **Removal of the existing extension points from the user's config**: This will significantly
reduce the complexity of the config from the user's perspective, but is also a breaking change
in the API and removes specificity where some users and developers may desire it.
* **Optional implementation with `Register()` interface**: This design proposed making multi-point
registration optional from the developer side through an interface that dictated the extension 
points to register. However, this involved code changes and additional work for plugins to 
take advantage of.

## Infrastructure Needed (Optional)

<!--
Use this section if you need things from the project/SIG. Examples include a
new subproject, repos requested, or GitHub details. Listing these here allows a
SIG to get the process for these resources started right away.
-->
