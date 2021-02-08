<!--
**Note:** When your KEP is complete, all of these comment blocks should be removed.

To get started with this template:

- [x] **Pick a hosting SIG.**
  Make sure that the problem space is something the SIG is interested in taking
  up. KEPs should not be checked in without a sponsoring SIG.
- [x] **Create an issue in kubernetes/enhancements**
  When filing an enhancement tracking issue, please make sure to complete all
  fields in that template. One of the fields asks for a link to the KEP. You
  can leave that blank until this KEP is filed, and then go back to the
  enhancement and add the link.
- [x] **Make a copy of this template directory.**
  Copy this template into the owning SIG's directory and name it
  `NNNN-short-descriptive-title`, where `NNNN` is the issue number (with no
  leading-zero padding) assigned to your enhancement above.
- [x] **Fill out as much of the kep.yaml file as you can.**
  At minimum, you should fill in the "Title", "Authors", "Owning-sig",
  "Status", and date-related fields.
- [x] **Fill out this file as best you can.**
  At minimum, you should fill in the "Summary" and "Motivation" sections.
  These should be easy if you've preflighted the idea of the KEP with the
  appropriate SIG(s).
- [x] **Create a PR for this KEP.**
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
# KEP-2458: Resource Fit Scoring Strategy

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
  - [User Stories (Optional)](#user-stories-optional)
    - [Story 1](#story-1)
  - [Risks and Mitigations](#risks-and-mitigations)
- [Design Details](#design-details)
  - [Test Plan](#test-plan)
  - [Graduation Criteria](#graduation-criteria)
    - [Alpha -&gt; Beta Graduation](#alpha---beta-graduation)
    - [Beta -&gt; GA Graduation](#beta---ga-graduation)
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
- [x] (R) Test plan is in place, giving consideration to SIG Architecture and SIG Testing input
- [x] (R) Graduation criteria is in place
- [ ] (R) Production readiness review completed
- [ ] (R) Production readiness review approved
- [x] "Implementation History" section is up-to-date for milestone
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
The default scheduler includes three score plugins (`NodeResourcesLeastAllocated`, 
`NodeResourcesMostAllocated` and `RequestedToCapacityRatio`) that implement different strategies 
for preferred resource allocation. Those plugins are mutually exclusive.

This KEP proposes to deprecate those plugins and combine them under one Score plugin, the same
one used for filtering (namely `NodeResourcesFit`), and add a `ScoringStrategy` parameter
to `NodeResourcesFit` plugin config that allows users to select which exact scoring strategy to run.

<!--
This section is incredibly important for producing high-quality, user-focused
documentation such as release notes or a development roadmap. It should be
possible to collect this information before implementation begins, in order to
avoid requiring implementors to split their attention between writing release
notes and implementing the feature itself. KEP editors and SIG Docs
should help to ensure that the tone and content of the `Summary` section is
useful for a wide audience.

A good summary is probably at least a paragraph in length.

Both in this section and below, follow the guidelines of the [documentation
style guide]. In particular, wrap lines to a reasonable length, to make it
easier for reviewers to cite specific portions, and to minimize diff churn on
updates.

[documentation style guide]: https://github.com/kubernetes/community/blob/master/contributors/guide/style-guide.md
-->

## Motivation

The motivation is two fold:

1) Reduce the complexity of configuring the default scheduler:

Configuring the scheduler plugins is a tedious task. The relatively large number of plugins
that the default scheduler supports doesn't make that task easier: more plugins means larger
number of combinations of enabled/disabled plugins.

Moreover, some combinations don't make sense and potentially harmful. For example, specific
to this KEP, enabling both least and most allocated scoring plugins is not useful.

Finally, we are planning to add more resource fit scoring strategies, like Best/WorstFit which
allows preferring nodes with the least/most amount of available absolute resources that can host the
pod; also, Least/MostAllocatable which allows preferring nodes with the least/most amount of 
allocatable resources that can host the pod. Adding those strategies as separate plugins will
make the scheduler configuration problem even worse.

2) A step towards allowing workloads to express their resource fit scoring strategy via pod spec.

Similar to how we allow workloads to express spread and affinity preferences, we want to allow
pods to express node resource fit preferences. This is important to achieve higher utilization
when running a mix of serving and batch workloads on the same cluster. We believe that the
changes we make here will enable this long term vision.


<!--
This section is for explicitly listing the motivation, goals, and non-goals of
this KEP.  Describe why the change is important and the benefits to users. The
motivation section can optionally provide links to [experience reports] to
demonstrate the interest in a KEP within the wider Kubernetes community.

[experience reports]: https://github.com/golang/go/wiki/ExperienceReports
-->

### Goals
- Allow users to configure resource fit preferences using a single plugin configuration
- Deprecate resource-based scoring plugins that implement individual strategies, specifically: 
  - `NodeResourcesLeastAllocated`
  - `NodeResourcesMostAllocated`
  - `RequestedToCapacityRatio`
- A flexible config API that allows adding new strategies in the future.

<!--
List the specific goals of the KEP. What is it trying to achieve? How will we
know that this has succeeded?
-->

### Non-Goals
- Allow users to express resource fit preferences via the pod spec. This is left as a
  follow up work that will be done under a different KEP.  

<!--
What is out of scope for this KEP? Listing non-goals helps to focus discussion
and make progress.
-->

## Proposal

<!--
This is where we get down to the specifics of what the proposal actually is.
This should have enough detail that reviewers can understand exactly what
you're proposing, but should not include things like API designs or
implementation. What is the desired outcome and how do we measure success?.
The "Design Details" section below is for the real
nitty-gritty.
-->

### User Stories (Optional)

<!--

Detail the things that people will be able to do if this KEP is implemented.
Include as much detail as possible so that people can understand the "how" of
the system. The goal here is to make this feel real for users without getting
bogged down.

-->

#### Story 1
As a cluster operator, I want an easy way to configure the scoring behavior of the scheduler
with respect to node resources. 

<!--

### Notes/Constraints/Caveats (Optional)

What are the caveats to the proposal?
What are some important details that didn't come across above?
Go in to as much detail as necessary here.
This might be a good place to talk about core concepts and how they relate.
-->

### Risks and Mitigations

Potentially reduce the number of allowed combinations too much. This can be mitigated
by adding more configuration strategies wherever applicable. 

<!--
What are the risks of this proposal, and how do we mitigate? Think broadly.
For example, consider both security and how this will impact the larger
Kubernetes ecosystem.

How will security be reviewed, and by whom?

How will UX be reviewed, and by whom?

Consider including folks who also work outside the SIG or subproject.
-->

## Design Details

Define a new `ScoringStrategy` type as follows:

```go

type StrategyType string

const (
    LeastAllocated StrategyType = "LeastAllocated"
    MostAllocated StrategyType = "MostAllocated"
    RequestedToCapacityRatio StrategyType = "RequestedToCapacityRatio"
)

type ScoringStrategy struct {
    metav1.TypeMeta

    // Strategy selects which strategy to run.
    Strategy StrategyType
    
    // Resources to consider when scoring.
    // The default resource set includes "cpu" and "memory" with an equal weight.
    // Allowed weights go from 1 to 100.
    Resources []ResourceSpec

    // Arguments specific to RequestedToCapacityRatio strategy.
    RequestedToCapacityRatio *RequestedToCapacityRatio
}

type RequestedToCapacityRatio struct {
    // Points defining priority function shape
    Shape []UtilizationShapePoint
}

// Note that the two types defined below already exist in the scheduler's component config API.
// ResourceSpec represents a single resource.
type ResourceSpec struct {
    // Name of the resource.
    Name string
    // Weight of the resource.
    Weight int64
}

// UtilizationShapePoint represents a single point of a priority function shape.
type UtilizationShapePoint struct {
    // Utilization (x axis). Valid values are 0 to 100. Fully utilized node maps to 100.
    Utilization int32
    // Score assigned to a given utilization (y axis). Valid values are 0 to 10.
    Score int32
}

```

Add `ScoringStrategy` to the existing `NodeResourcesFitArgs`:

```go

// NodeResourcesFitArgs holds arguments used to configure the NodeResourcesFit plugin.
type NodeResourcesFitArgs struct {
    metav1.TypeMeta

    // IgnoredResources is the list of resources that NodeResources fit filter
    // should ignore.
    IgnoredResources []string
    // IgnoredResourceGroups defines the list of resource groups that NodeResources fit 
    // filter should ignore.
    // e.g. if group is ["example.com"], it will ignore all resource names that begin
    // with "example.com", such as "example.com/aaa" and "example.com/bbb".
    // A resource group name can't contain '/'.
    IgnoredResourceGroups []string
    
    // ScoringStrategy selects the node resource scoring strategy.
    ScoringStrategy *ScoringStrategy
}
```


As of writing this KEP, scheduler component config is v1beta1. The plugins we plan to
deprecate will continue to be configurable in v1beta1, and will not be available in v1beta2. 

<!--
This section should contain enough information that the specifics of your
change are understandable. This may include API specs (though not always
required) or even code snippets. If there's any ambiguity about HOW your
proposal will be implemented, this is the place to discuss them.
-->

### Test Plan

- Unit and integration tests covering the new configuration path. 
  For example, configuring a `MostAllocated` strategy should have the 
  same behavior as configuring a `NodeResourcesMostAllocated` plugin separately.


<!--
**Note:** *Not required until targeted at a release.*

Consider the following in developing a test plan for this enhancement:
- Will there be e2e and integration tests, in addition to unit tests?
- How will it be tested in isolation vs with other components?

No need to outline all of the test cases, just the general strategy. Anything
that would count as tricky in the implementation, and anything particularly
challenging to test, should be called out.

All code is expected to have adequate tests (eventually with coverage
expectations). Please adhere to the [Kubernetes testing guidelines][testing-guidelines]
when drafting this test plan.

[testing-guidelines]: https://git.k8s.io/community/contributors/devel/sig-testing/testing.md
-->

### Graduation Criteria

#### Alpha -> Beta Graduation

- The KEP proposes an API change to scheduler component config to allow 
  expressing an existing behavior in a different way. Since this configuration
  is opt-in, it will not be guarded by a feature flag, and will graduate with
  component config (i.e., will start in beta directly).
- In v1beta1, the default scheduler configuration will continue to use the old plugins. In 
  v1beta2, it will use the new config API.
  
#### Beta -> GA Graduation
    
- Allowing time for feedback to ensure that the new API sufficiently expresses users requirements.

<!--
**Note:** *Not required until targeted at a release.*

Define graduation milestones.

These may be defined in terms of API maturity, or as something else. The KEP
should keep this high-level with a focus on what signals will be looked at to
determine graduation.

Consider the following in developing the graduation criteria for this enhancement:
- [Maturity levels (`alpha`, `beta`, `stable`)][maturity-levels]
- [Deprecation policy][deprecation-policy]

Clearly define what graduation means by either linking to the [API doc
definition](https://kubernetes.io/docs/concepts/overview/kubernetes-api/#api-versioning)
or by redefining what graduation means.

In general we try to use the same stages (alpha, beta, GA), regardless of how the
functionality is accessed.

[maturity-levels]: https://git.k8s.io/community/contributors/devel/sig-architecture/api_changes.md#alpha-beta-and-stable-versions
[deprecation-policy]: https://kubernetes.io/docs/reference/using-api/deprecation-policy/

Below are some examples to consider, in addition to the aforementioned [maturity levels][maturity-levels].

#### Alpha -> Beta Graduation

- Gather feedback from developers and surveys
- Complete features A, B, C
- Tests are in Testgrid and linked in KEP

#### Beta -> GA Graduation

- N examples of real-world usage
- N installs
- More rigorous forms of testing—e.g., downgrade tests and scalability tests
- Allowing time for feedback

**Note:** Generally we also wait at least two releases between beta and
GA/stable, because there's no opportunity for user feedback, or even bug reports,
in back-to-back releases.

#### Removing a Deprecated Flag

- Announce deprecation and support policy of the existing flag
- Two versions passed since introducing the functionality that deprecates the flag (to address version skew)
- Address feedback on usage/changed behavior, provided on GitHub issues
- Deprecate the flag

**For non-optional features moving to GA, the graduation criteria must include 
[conformance tests].**

[conformance tests]: https://git.k8s.io/community/contributors/devel/sig-architecture/conformance-tests.md
-->

### Upgrade / Downgrade Strategy

N/A

### Version Skew Strategy

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

_This section must be completed when targeting alpha to a release._

* **How can this feature be enabled / disabled in a live cluster?**
  - [ ] Feature gate (also fill in values in `kep.yaml`)
    - Feature gate name: 
    - Components depending on the feature gate:
  - [x] Other
    - Describe the mechanism: this is an opt-in scheduler component config parameter. 
      If set, then it will allow configuring an existing scheduler behaviour using 
      this new parameter.
    - Will enabling / disabling the feature require downtime of the control
      plane? yes, requires restarting the scheduler.
    - Will enabling / disabling the feature require downtime or reprovisioning
      of a node? (Do not assume `Dynamic Kubelet Config` feature is enabled). No

* **Does enabling the feature change any default behavior?**
  No.

* **Can the feature be disabled once it has been enabled (i.e. can we roll back
  the enablement)?**
  Yes, this will depend on the component config version used:
  - if using v1beta1, revert to use the legacy plugins directly
  - If using v1beta2, revert to v1beta1 and use the legacy plugins directly 

* **What happens if we reenable the feature if it was previously rolled back?**
 This results in rolling back to an older scheduler configuration. If the old
 configuration is semantically the same, then the pod scheduling behavior should
 not change.

* **Are there any tests for feature enablement/disablement?**
  No, we will do manual testing.


### Rollout, Upgrade and Rollback Planning

_This section must be completed when targeting beta graduation to a release._

* **How can a rollout fail? Can it impact already running workloads?**
  It shouldn't impact already running workloads. This is an opt-in feature to 
  express existing behavior using a different API. Operators need to change the
  scheduler configuration to enable it.

* **What specific metrics should inform a rollback?**
  - A spike on metric `schedule_attempts_total{result="error|unschedulable"}`  


* **Were upgrade and rollback tested? Was the upgrade->downgrade->upgrade path tested?**
No, will be manually tested.

* **Is the rollout accompanied by any deprecations and/or removals of features, APIs, 
fields of API types, flags, etc.?**
  No.

### Monitoring Requirements

_This section must be completed when targeting beta graduation to a release._

* **How can an operator determine if the feature is in use by workloads?**
  This is a scheduler configuration feature that operators themselves opt into.

* **What are the SLIs (Service Level Indicators) an operator can use to determine 
the health of the service?**
  - [x] Metrics
    - Component exposing the metric: kube-scheduler
      - Metric name: `pod_scheduling_duration_seconds`
      - Metric name: `schedule_attempts_total{result="error|unschedulable"}`
  - [ ] Other (treat as last resort)
    - Details:

* **What are the reasonable SLOs (Service Level Objectives) for the above SLIs?**
  - 99% of pod scheduling latency is within x minutes
  - x% of `schedule_attempts_total` are successful

* **Are there any missing metrics that would be useful to have to improve observability 
of this feature?**
No.

### Dependencies

_This section must be completed when targeting beta graduation to a release._

* **Does this feature depend on any specific services running in the cluster?**
No


### Scalability

_For alpha, this section is encouraged: reviewers should consider these questions
and attempt to answer them._

_For beta, this section is required: reviewers must answer these questions._

_For GA, this section is required: approvers should be able to confirm the
previous answers based on experience in the field._

* **Will enabling / using this feature result in any new API calls?**
No. 

* **Will enabling / using this feature result in introducing new API types?**
No.

* **Will enabling / using this feature result in any new calls to the cloud 
provider?**
No.

* **Will enabling / using this feature result in increasing size or count of 
the existing API objects?**
No.

* **Will enabling / using this feature result in increasing time taken by any 
operations covered by [existing SLIs/SLOs]?**
No.

* **Will enabling / using this feature result in non-negligible increase of 
resource usage (CPU, RAM, disk, IO, ...) in any components?**
No.

### Troubleshooting

The Troubleshooting section currently serves the `Playbook` role. We may consider
splitting it into a dedicated `Playbook` document (potentially with some monitoring
details). For now, we leave it here.

_This section must be completed when targeting beta graduation to a release._

* **How does this feature react if the API server and/or etcd is unavailable?**

Running workloads will not be impacted, but pods that are not scheduled yet will
not get assigned nodes.

* **What are other known failure modes?**
N/A

* **What steps should be taken if SLOs are not being met to determine the problem?**
N/A

[supported limits]: https://git.k8s.io/community//sig-scalability/configs-and-limits/thresholds.md
[existing SLIs/SLOs]: https://git.k8s.io/community/sig-scalability/slos/slos.md#kubernetes-slisslos


<!--
## Alternatives

What other approaches did you consider, and why did you rule them out? These do
not need to be as detailed as the proposal, but should include enough
information to express the idea and why it was not acceptable.
-->

## Implementation History
 - 2021-02-08: Initial KEP sent for review

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

<!--

## Drawbacks

Why should this KEP _not_ be implemented?
-->




<!--

## Infrastructure Needed (Optional)

Use this section if you need things from the project/SIG. Examples include a
new subproject, repos requested, or GitHub details. Listing these here allows a
SIG to get the process for these resources started right away.
-->
