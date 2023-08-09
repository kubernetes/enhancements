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
# KEP-3990: Pod Topology Spread DoNotSchedule to SchedulingAnyway fallback mode

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
    - [Story 2](#story-2)
  - [Notes/Constraints/Caveats (Optional)](#notesconstraintscaveats-optional)
  - [Risks and Mitigations](#risks-and-mitigations)
    - [the fallback could be done when it's actually not needed.](#the-fallback-could-be-done-when-its-actually-not-needed)
- [Design Details](#design-details)
  - [new API changes](#new-api-changes)
  - [ScaleUpFailed](#scaleupfailed)
    - [How we implement <code>TriggeredScaleUp</code> in the cluster autoscaler](#how-we-implement-triggeredscaleup-in-the-cluster-autoscaler)
  - [PreemptionFalied](#preemptionfalied)
  - [What if are both specified in <code>FallbackCriterion</code>?](#what-if-are-both-specified-in-fallbackcriterion)
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
  - [introduce <code>DoNotScheduleUntilScaleUpFailed</code> and <code>DoNotScheduleUntilPreemptionFailed</code>](#introduce-donotscheduleuntilscaleupfailed-and-donotscheduleuntilpreemptionfailed)
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

A new field `fallbackCriteria` is introduced to `PodSpec.TopologySpreadConstraint[*]` 
to represent when to fallback from DoNotSchedule to ScheduleAnyway.
It can contain two values: `ScaleUpFailed` to fall back when the cluster autoscaler fails to create new Node for Pods,
and `PreemptionFailed` to fall back when the preemption doesn't help to make Pods schedulable.

## Motivation

<!--
This section is for explicitly listing the motivation, goals, and non-goals of
this KEP.  Describe why the change is important and the benefits to users. The
motivation section can optionally provide links to [experience reports] to
demonstrate the interest in a KEP within the wider Kubernetes community.

[experience reports]: https://github.com/golang/go/wiki/ExperienceReports
-->

Pod Topology Spread is designed to enhance high availability by distributing Pods across numerous failure domains. 
However, ironically, it can badly affect the availability of Pods 
if utilized with `WhenUnsatisfiable: DoNotSchedule`. 
Particularly amiss are situations where preemption cannot make a Pod schedulable or the cluster autoscaler is unable to create new Node. 
Notably, under these circumstances, the intended Pod Topology Spread can negatively impact Pod availability.

### Goals

<!--
List the specific goals of the KEP. What is it trying to achieve? How will we
know that this has succeeded?
-->

- A new field `fallbackCriteria` is introduced to `PodSpec.TopologySpreadConstraint[*]` 
  - `ScaleUpFailed` to fallback when the cluster autoscaler fails to create new Node for Pod.
  - `PreemptionFailed` to fallback when preemption doesn't help make Pod schedulable.
- introduce `TriggeredScaleUp` in Pod condition 
  - change the cluster autoscaler to set it `false` when it cannot create new Node for the Pod, `true` when success.

### Non-Goals

<!--
What is out of scope for this KEP? Listing non-goals helps to focus discussion
and make progress.
-->

- reschedule Pods, which are scheduled by the fallback mode, for a better distribution after some time.

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

Your cluster has the cluster autoscaler 
and you widely use Pod Topology Spread with `WhenUnsatisfiable: DoNotSchedule` for zone to strongthen workloads against the zone failure.
And if the cluster autoscaler fails to create new Node for Pods due to the instance stockout, 
you want to fallback from DoNotSchedule to ScheduleAnyway 
because otherwise you'd hurt the availability of workload to achieve a better availability via Pod Topology Spread.
That's putting the cart before the horse.

In this case, you can use `ScaleUpFailed` in `fallbackCriteria`,
to fallback from DoNotSchedule to ScheduleAnyway 

```yaml
topologySpreadConstraints:
  - maxSkew: 1
    topologyKey: topology.kubernetes.io/zone
    whenUnsatisfiable: DoNotSchedule
    fallbackCriteria:
      - ScaleUpFailed
    labelSelector:
      matchLabels:
        foo: bar
```

#### Story 2

Your cluster doesn't have the cluster autoscaler 
and has some low-priority Pods to make space (often called overprovisional Pods, balloon Pods, etc.).
Basically, you want to leverage preemption to achieve the best distribution as much as possible, 
so you have to schedule Pods with `WhenUnsatisfiable: DoNotSchedule`.
But, you don't want to make Pods unschedulable by Pod Topology Spread if the preemption won't make Pods schedulable.

```yaml
topologySpreadConstraints:
  - maxSkew: 1
    topologyKey: topology.kubernetes.io/zone
    whenUnsatisfiable: DoNotSchedule
    fallbackCriteria:
      - PreemptionFailed
    labelSelector:
      matchLabels:
        foo: bar
```

### Notes/Constraints/Caveats (Optional)

<!--
What are the caveats to the proposal?
What are some important details that didn't come across above?
Go in to as much detail as necessary here.
This might be a good place to talk about core concepts and how they relate.
-->

### Risks and Mitigations

<!--
What are the risks of this proposal, and how do we mitigate? Think broadly.
For example, consider both security and how this will impact the larger
Kubernetes ecosystem.

How will security be reviewed, and by whom?

How will UX be reviewed, and by whom?

Consider including folks who also work outside the SIG or subproject.
-->

#### the fallback could be done when it's actually not needed.

Even if the Pod is rejected by plugins other than Pod Topology Spread,
when one of specified criteria is satisfied, the scheduler fallbacks from DoNotSchedule to ScheduleAnyway.

One possible mitigation is to add `UnschedulablePlugins`, which equals to [QueuedPodInfo.UnschedulablePlugins](https://github.com/kubernetes/kubernetes/blob/8a7df727820bafed8cef27e094a0212d758fcd40/pkg/scheduler/framework/types.go#L180), to somewhere in Pod status 
so that Pod Topology Spread can decide to fall back only when the Pod was rejected by Pod Topology Spread.

## Design Details

<!--
This section should contain enough information that the specifics of your
change are understandable. This may include API specs (though not always
required) or even code snippets. If there's any ambiguity about HOW your
proposal will be implemented, this is the place to discuss them.
-->

### new API changes

```go
// FallbackCriterion represents when the scheduler falls back from the required scheduling constraint to the preferred one.
type FallbackCriterion string

const (
  // ScaleUpFailed represents when the Pod has `TriggeredScaleUp: false` in its condition.
  ScaleUpFailed    FallbackCriterion = "ScaleUpFailed"
  // PreemptionFailed represents when the scheduler tried to make space for the Pod by the preemption, but failed.
  // Specifically, when the Pod doesn't have `NominatedNodeName` while having `PodScheduled: false`.
  PreemptionFailed FallbackCriterion = "PreemptionFailed"
)


type TopologySpreadConstraint struct {
......
  // FallbackCriteria is the list of criteria that the scheduler decides when to fall back from DoNotSchedule to ScheduleAnyway.
  // It's valid to set only when WhenUnsatisfiable is DoNotSchedule.
  // If multiple criteria are in this list, the scheduler falls back when ALL criteria in `FallbackCriterion` are satisfied.
  // It's an optional field. The default value is nil, meaning the scheduler never falls back.
  // +optional
  FallbackCriteria []FallbackCriterion
}

// These are valid conditions of pod.
const (
......
  // TriggeredScaleUp indicates that the Pod triggered scaling up the cluster.
  // If it's true, new Node for the Pod was successfully created.
  // Otherwise, new Node for the Pod tried to be created, but failed.
  TriggeredScaleUp PodConditionType = "TriggeredScaleUp"
)
```

### ScaleUpFailed

`ScaleUpFailed` is used to fallback when the Pod doesn't trigger scaling up the cluster.
`TriggeredScaleUp` is a new condition to show whether the Pod triggers scaling up the cluster, 
which creates new Node for Pod typically by the cluster autoscaler.

**fallback scenario**

1. Pod is rejected and stays unschedulable.
2. The cluster autoscaler finds those unschedulable Pod(s) but cannot create Nodes because of stockouts.
3. The cluster autoscaler adds `TriggeredScaleUp: false`. 
4. The scheduler notices `TriggeredScaleUp: false` on Pod and schedules that Pod while falling back to `ScheduleAnyway` on Pod Topology Spread.

#### How we implement `TriggeredScaleUp` in the cluster autoscaler

Basically, we just put `TriggeredScaleUp: false` for Pods in [status.ScaleUpStatus.PodsRemainUnschedulable](https://github.com/kubernetes/autoscaler/blob/109998dbf30e6a6ef84fc37ebaccca23d7dee2f3/cluster-autoscaler/processors/status/scale_up_status_processor.go#L37) every [reconciliation (RunOnce)](https://github.com/kubernetes/autoscaler/blob/109998dbf30e6a6ef84fc37ebaccca23d7dee2f3/cluster-autoscaler/core/static_autoscaler.go#L296).

This `status.ScaleUpStatus.PodsRemainUnschedulable` contains Pods that the cluster autoscaler [simulates](https://github.com/kubernetes/autoscaler/blob/109998dbf30e6a6ef84fc37ebaccca23d7dee2f3/cluster-autoscaler/core/scaleup/orchestrator/orchestrator.go#L536) the scheduling process for and determines that Pods wouldn't be schedulable in any node group. 

So, for a simple example, 
if a Pod has 64 cpu request, but no node group can satisfy 64 cpu requirement,
the Pod would be in `status.ScaleUpStatus.PodsRemainUnschedulable`; get `TriggeredScaleUp: false`.

A complicated scenario could also be covered by this way;
supposing a Pod has 64 cpu request and only a node group can satisfy 64 cpu requirement,
but the node group is running out of instances at the moment.
In this case, the first reconciliation selects the node group to make the Pod schedulable,
but the node group size increase request would be rejected by the cloud provider because of the stockout.
The node group is then considered to be non-safe for a while,
and the next reconciliation happens without taking the failed node group into account.
As said, there's no other node group that can satisfy 64 cpu requirement,
and then the Pod would be finally in `status.ScaleUpStatus.PodsRemainUnschedulable`; get `TriggeredScaleUp: false`.

### PreemptionFalied

`PreemptionFailed` is used to fallback when preemption is failed.
Pod Topology Spread can notice the preemption failure 
by `PodScheduled: false` (the past scheduling failed) and empty `NominatedNodename` (the past postfilter did nothing for this Pod).

**fallback scenario**

1. Pod is rejected in the scheduling cycle.
2. In the PostFilter extension point, the scheduler tries to make space by the preemption, but finds the preemption doesn't help.
3. When the Pod is moved back to the scheduling queue, the scheduler adds `PodScheduled: false` condition to Pod.
4. The scheduler notices that the preemption wasn't performed for Pod by `PodScheduled: false` and empty `NominatedNodeName` on the Pod. 
And, it schedules the Pod while falling back to `ScheduleAnyway` on Pod Topology Spread.

### What if are both specified in `FallbackCriterion`?

The scheduler fallbacks when all criteria in `FallbackCriterion` are satisfied.

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

- `k8s.io/kubernetes/pkg/scheduler/framework/plugins/podtopologyspread`: `2023-08-12` - `87%`
- `k8s.io/kubernetes/pkg/api/pod`: `2023-08-12` - `76.6%`
- `k8s.io/kubernetes/pkg/apis/core/validation`: `2023-08-12` - `83.6%`

##### Integration tests

<!--
Integration tests are contained in k8s.io/kubernetes/test/integration.
Integration tests allow control of the configuration parameters used to start the binaries under test.
This is different from e2e tests which do not allow configuration of parameters.
Doing this allows testing non-default options and multiple different and potentially conflicting command line options.
-->

<!--
This question should be filled when targeting a release.
For Alpha, describe what tests will be added to ensure proper quality of the enhancement.

For Beta and GA, add links to added tests together with links to k8s-triage for those tests:
https://storage.googleapis.com/k8s-triage/index.html
-->

test: https://github.com/kubernetes/kubernetes/blob/6e0cb243d57592c917fe449dde20b0e246bc66be/test/integration/scheduler/filters/filters_test.go#L1066
k8s-triage: https://storage.googleapis.com/k8s-triage/index.html?sig=scheduling&test=TestPodTopologySpreadFilter

##### e2e tests

<!--
This question should be filled when targeting a release.
For Alpha, describe what tests will be added to ensure proper quality of the enhancement.

For Beta and GA, add links to added tests together with links to k8s-triage for those tests:
https://storage.googleapis.com/k8s-triage/index.html

We expect no non-infra related flakes in the last month as a GA graduation criteria.
-->

N/A

--

This feature doesn't introduce any new API endpoints and doesn't interact with other components. 
So, E2E tests doesn't add extra value to integration tests.

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

- [] The feature gate is added, which is disabled by default.
- [] Add a new field `fallbackCriteria` to `TopologySpreadConstraint` and feature gating.
  - [] implement `ScaleUpFailed` to fallback when CA fails to create new Node for Pod.
  - [] implement `PreemptionFailed` to fallback when preemption doesn't help make Pod schedulable.
- [] introduce `TriggeredScaleUp` in Pod condition 
- [] Implement all tests mentioned in the [Test Plan](#test-plan).

Out of Kubernetes, but:
- [] (cluster autoscaler) set `TriggeredScaleUp` after trying to create Node for Pod.

#### Beta

- The feature gate is enabled by default.

#### GA

- No negative feedback.
- No bug issues reported.

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

The previous Pod Topology Spread behavior will not be broken. Users can continue to use
their Pod specs as it is.

To use this enhancement, users need to enable the feature gate (during this feature is in the alpha.),
and add `fallbackCriteria` on their `TopologySpreadConstraint`.

Also, if users want to use `ScaleUpFailed`, they need to use the cluster autoscaler
that supports `TriggeredScaleUp` Pod condition.

**Downgrade**

kube-apiserver will reject Pod creation with `fallbackCriteria` in `TopologySpreadConstraint`.
Regarding existing Pods, we keep `fallbackCriteria`, but the scheduler ignores them.

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

- [ ] Feature gate (also fill in values in `kep.yaml`)
  - Feature gate name: `PodTopologySpreadFallbackMode`
  - Components depending on the feature gate:
      - kube-scheduler
      - kube-apiserver

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
by restarting kube-apiserver and kube-apiserver with the feature-gate off.
In terms of Stable versions, users can choose to opt-out by not setting the
`fallbackCriteria` field.

###### What happens if we reenable the feature if it was previously rolled back?

Scheduling of pods with `fallbackCriteria` is affected.

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

No. 

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

For GA, this section is required: approvers should be able to confirm the
previous answers based on experience in the field.
-->

###### How can an operator determine if the feature is in use by workloads?

<!--
Ideally, this should be a metric. Operations against the Kubernetes API (e.g.,
checking if there are objects with field X set) may be a last resort. Avoid
logs or events for this purpose.
-->

###### How can someone using this feature know that it is working for their instance?

<!--
For instance, if this is a pod-related feature, it should be possible to determine if the feature is functioning properly
for each individual pod.
Pick one more of these and delete the rest.
Please describe all items visible to end users below with sufficient detail so that they can verify correct enablement
and operation of this feature.
Recall that end users cannot usually observe component logs or access metrics.
-->

- [ ] Events
  - Event Reason: 
- [ ] API .status
  - Condition name: 
  - Other field: 
- [ ] Other (treat as last resort)
  - Details:

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

###### Will enabling / using this feature result in introducing new API types?

<!--
Describe them, providing:
  - API type
  - Supported number of objects per cluster
  - Supported number of objects per namespace (for namespace-scoped objects)
-->

###### Will enabling / using this feature result in any new calls to the cloud provider?

<!--
Describe them, providing:
  - Which API(s):
  - Estimated increase:
-->

###### Will enabling / using this feature result in increasing size or count of the existing API objects?

<!--
Describe them, providing:
  - API type(s):
  - Estimated increase in size: (e.g., new annotation of size 32B)
  - Estimated amount of new objects: (e.g., new Object X for every existing Pod)
-->

###### Will enabling / using this feature result in increasing time taken by any operations covered by existing SLIs/SLOs?

<!--
Look at the [existing SLIs/SLOs].

Think about adding additional work or introducing new steps in between
(e.g. need to do X to start a container), etc. Please describe the details.

[existing SLIs/SLOs]: https://git.k8s.io/community/sig-scalability/slos/slos.md#kubernetes-slisslos
-->

###### Will enabling / using this feature result in non-negligible increase of resource usage (CPU, RAM, disk, IO, ...) in any components?

<!--
Things to keep in mind include: additional in-memory state, additional
non-trivial computations, excessive access to disks (including increased log
volume), significant amount of data sent and/or received over network, etc.
This through this both in small and large cases, again with respect to the
[supported limits].

[supported limits]: https://git.k8s.io/community//sig-scalability/configs-and-limits/thresholds.md
-->

###### Can enabling / using this feature result in resource exhaustion of some node resources (PIDs, sockets, inodes, etc.)?

<!--
Focus not just on happy cases, but primarily on more pathological cases
(e.g. probes taking a minute instead of milliseconds, failed pods consuming resources, etc.).
If any of the resources can be exhausted, how this is mitigated with the existing limits
(e.g. pods per node) or new limits added by this KEP?

Are there any tests that were run/should be run to understand performance characteristics better
and validate the declared limits?
-->

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

- 2023-08-12: Initial KEP PR is submitted.

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

### introduce `DoNotScheduleUntilScaleUpFailed` and `DoNotScheduleUntilPreemptionFailed`

Instead of `FallBackCriteria`, introduce `DoNotScheduleUntilScaleUpFailed` and `DoNotScheduleUntilPreemptionFailed` in `WhenUnsatisfiable`.
`DoNotScheduleUntilScaleUpFailed` corresponds to `ScaleUpFailed`, 
and `DoNotScheduleUntilPreemptionFailed` corresponds to `PreemptionFailed`.

We noticed a downside in this way, compared to `FallBackCriteria`.
In other scheduling constraints, we distinguish between preferred and required constraint by where the constraint is written in.
For example, PodAffinity and NodeAffinity, if it's written in `requiredDuringSchedulingIgnoredDuringExecution`, it's required. 
And if it's written in `preferredDuringSchedulingIgnoredDuringExecution`, it's preferred.

In the future, we may want to introduce similar fallback mechanism in such other scheduling constraints, 
but, we couldn't make the similar API design if we went with `DoNotScheduleUntilScaleUpFailed` and `DoNotScheduleUntilPreemptionFailed`, 
as they don't define preferred or required in enum value like `WhenUnsatisfiable`.

On the other hand, `FallBackCriteria` allows us to unify APIs in all scheduling constraints. 
We will just introduce `FallBackCriteria` field in them and there we go.

## Infrastructure Needed (Optional)

<!--
Use this section if you need things from the project/SIG. Examples include a
new subproject, repos requested, or GitHub details. Listing these here allows a
SIG to get the process for these resources started right away.
-->
