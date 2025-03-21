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
# KEP-3243: Respect PodTopologySpread after rolling upgrades

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
  - [Notes/Constraints/Caveats (Optional)](#notesconstraintscaveats-optional)
  - [Risks and Mitigations](#risks-and-mitigations)
    - [Possible misuse](#possible-misuse)
    - [The update to labels specified at <code>matchLabelKeys</code> isn't supported](#the-update-to-labels-specified-at-matchlabelkeys-isnt-supported)
- [Design Details](#design-details)
  - [[v1.34] design change and a safe upgrade path](#v134-design-change-and-a-safe-upgrade-path)
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
  - [use pod generateName](#use-pod-generatename)
  - [implement MatchLabelKeys in only either the scheduler plugin or kube-apiserver](#implement-matchlabelkeys-in-only-either-the-scheduler-plugin-or-kube-apiserver)
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
- [x] (R) Graduation criteria is in place
  - [ ] (R) [all GA Endpoints](https://github.com/kubernetes/community/pull/1806) must be hit by [Conformance Tests](https://github.com/kubernetes/community/blob/master/contributors/devel/sig-architecture/conformance-tests.md) 
- [ ] (R) Production readiness review completed
- [ ] (R) Production readiness review approved
- [x] "Implementation History" section is up-to-date for milestone
- [x] User-facing documentation has been created in [kubernetes/website], for publication to [kubernetes.io]
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
The pod topology spread feature allows users to define the group of pods over 
which spreading is applied using a LabelSelector. This means the user should 
know the exact label key and value when defining the pod spec.

This KEP proposes a complementary field to LabelSelector named `MatchLabelKeys` in
`TopologySpreadConstraint` which represents a set of label keys only. 
At a pod creation, kube-apiserver will use those keys to look up label values from the incoming pod 
and those key-value labels will be merged with existing `LabelSelector` to identify the group of existing pods over 
which the spreading skew will be calculated.
Note that in case `MatchLabelKeys` is supported in the cluster-level default constraints 
(see https://github.com/kubernetes/kubernetes/issues/129198), kube-scheduler will also handle it separately.


The main case that this new way for identifying pods will enable is constraining 
skew spreading calculation to happen at the revision level in Deployments during 
rolling upgrades.

## Motivation

<!--
This section is for explicitly listing the motivation, goals, and non-goals of
this KEP.  Describe why the change is important and the benefits to users. The
motivation section can optionally provide links to [experience reports] to
demonstrate the interest in a KEP within the wider Kubernetes community.

[experience reports]: https://github.com/golang/go/wiki/ExperienceReports
-->

PodTopologySpread is widely used in production environments, especially in 
service type workloads which employ Deployments. However, currently it has a 
limitation that manifests during rolling updates which causes the deployment to 
end up out of balance ([98215](https://github.com/kubernetes/kubernetes/issues/98215), 
[105661](https://github.com/kubernetes/kubernetes/issues/105661),
[k8s-pod-topology spread is not respected after rollout](https://stackoverflow.com/questions/66510883/k8s-pod-topology-spread-is-not-respected-after-rollout)). 

The root cause is that PodTopologySpread constraints allow defining a key-value 
label selector, which applies to all pods in a Deployment irrespective of their 
owning ReplicaSet. As a result, when a new revision is rolled out, spreading will 
apply across pods from both the old and new ReplicaSets, and so by the time the 
new ReplicaSet is completely rolled out and the old one is rolled back, the actual 
spreading we are left with may not match expectations because the deleted pods from 
the older ReplicaSet will cause skewed distribution for the remaining pods.

Currently, users are given two solutions to this problem. The first is to add a 
revision label to Deployment and update it manually at each rolling upgrade (both 
the label on the podTemplate and the selector in the podTopologySpread constraint),
while the second is to deploy a descheduler to re-balance the pod 
distribution. The former solution isn't user friendly and requires manual tuning,
which is error prone; while the latter requires installing and maintaining an 
extra controller. In this proposal, we propose a native way to maintain pod balance 
after a rolling upgrade in Deployments that use PodTopologySpread.

### Goals

<!--
List the specific goals of the KEP. What is it trying to achieve? How will we
know that this has succeeded?
-->
- Allow users to define PodTopologySpread constraints such that they apply only 
  within the boundaries of a Deployment revision during rolling upgrades.

### Non-Goals

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

When users apply a rolling update to a deployment that uses 
PodTopologySpread,  the spread should be respected only within the new 
revision, not across all revisions of the deployment.

### Notes/Constraints/Caveats (Optional)

<!--
What are the caveats to the proposal?
What are some important details that didn't come across above?
Go in to as much detail as necessary here.
This might be a good place to talk about core concepts and how they relate.
-->
In most scenarios, users can use the label keyed with `pod-template-hash` added 
automatically by the Deployment controller to distinguish between different 
revisions in a single Deployment. But for more complex scenarios 
(eg. topology spread associating two deployments at the same time), users are 
responsible for providing common labels to identify which pods should be grouped. 

### Risks and Mitigations

<!--
What are the risks of this proposal, and how do we mitigate? Think broadly.
For example, consider both security and how this will impact the larger
Kubernetes ecosystem.

How will security be reviewed, and by whom?

How will UX be reviewed, and by whom?

Consider including folks who also work outside the SIG or subproject.
-->
#### Possible misuse

In addition to using `pod-template-hash` added by the Deployment controller, 
users can also provide the customized key in  `MatchLabelKeys` to identify 
which pods should be grouped. If so, the user needs to ensure that it is 
correct and not duplicated with other unrelated workloads.

#### The update to labels specified at `matchLabelKeys` isn't supported

`MatchLabelKeys` is handled and merged into `LabelSelector` at _a pod's creation_.
It means this feature doesn't support the label's update even though a user 
could update the label that is specified at `matchLabelKeys` after a pod's creation.
So, in such cases, the update of the label isn't reflected onto the merged `LabelSelector`,
even though users might expect it to be.
On the documentation, we'll declare it's not recommended to use `matchLabelKeys` with labels that might be updated.

Also, we assume the risk is acceptably low because:
1. It's a fairly low probability to happen because pods are usually managed by another resource (e.g., deployment), 
   and the update to pod template's labels on a deployment recreates pods, instead of directly updating the labels on existing pods. 
   Also, even if users somehow use bare pods (which is not recommended in the first place), 
   there's usually only a tiny moment between the pod creation and the pod getting scheduled, which makes this risk further rarer to happen, 
   unless many pods are often getting stuck being unschedulable for a long time in the cluster (which is not recommended) 
   or the labels specified at `matchLabelKeys` are frequently updated (which we'll declare as not recommended).
2. If it happens, `selfMatchNum` will be 0 and both `matchNum` and `minMatchNum` will be retained.
   Consequently, depending on the current number of matching pods in the domain, `matchNum` - `minMatchNum` might be bigger than `maxSkew`, 
   and the pod(s) could be unschedulable.
   But, it does not mean that the unfortunate pods would be unschedulable forever.

## Design Details

<!--
This section should contain enough information that the specifics of your
change are understandable. This may include API specs (though not always
required) or even code snippets. If there's any ambiguity about HOW your
proposal will be implemented, this is the place to discuss them.
-->

A new optional field named `MatchLabelKeys` will be introduced to `TopologySpreadConstraint`.
Currently, when scheduling a pod, the `LabelSelector` defined in the pod is used 
to identify the group of pods over which spreading will be calculated. 
`MatchLabelKeys` adds another constraint to how this group of pods is identified.
```go
type TopologySpreadConstraint struct {
	MaxSkew           int32
	TopologyKey       string
	WhenUnsatisfiable UnsatisfiableConstraintAction
	LabelSelector     *metav1.LabelSelector

	// MatchLabelKeys is a set of pod label keys to select the pods over which 
	// spreading will be calculated. The keys are used to lookup values from the
	// incoming pod labels, those key-value labels are ANDed with `LabelSelector`
	// to select the group of existing pods over which spreading will be calculated
	// for the incoming pod. Keys that don't exist in the incoming pod labels will
	// be ignored.
	MatchLabelKeys []string
}
```

When a Pod is created, kube-apiserver will obtain the labels from the pod 
by the keys in `matchLabelKeys` and the key-value labels are merged to `LabelSelector` 
of `TopologySpreadConstraint`.

For example, when this sample Pod is created,

```yaml
apiVersion: v1
kind: Pod
metadata:
  name: sample
  labels:
    app: sample
...
  topologySpreadConstraints:
  - maxSkew: 1
    topologyKey: kubernetes.io/hostname
    whenUnsatisfiable: DoNotSchedule
    labelSelector: {}
    matchLabelKeys: # ADDED
    - app
```

kube-apiserver modifies the `labelSelector` like the following:

```diff
  topologySpreadConstraints:
  - maxSkew: 1
    topologyKey: kubernetes.io/hostname
    whenUnsatisfiable: DoNotSchedule
    labelSelector:
+     matchExpressions:
+     - key: app
+       operator: In
+       values:
+       - sample
    matchLabelKeys:
    - app
```

In addition, kube-scheduler will handle `matchLabelKeys` within the cluster-level default constraints 
in the scheduler configuration in the future (see https://github.com/kubernetes/kubernetes/issues/129198).

Finally, the feature will be guarded by a new feature flag. If the feature is 
disabled, the field `matchLabelKeys` and corresponding `labelSelector` are preserved 
if it was already set in the persisted Pod object, otherwise new Pod with the field 
creation will be rejected by kube-apiserver.
Also kube-scheduler will ignore `matchLabelKeys` in the cluster-level default constraints configuration.

### [v1.34] design change and a safe upgrade path
Previously, kube-scheduler just internally handled `matchLabelKeys` before the calculation of scheduling results.
But, we changed the implementation design to the current form to make the design align with PodAffinity's `matchLabelKeys`. 
(See the detailed discussion in [the alternative section](#implement-matchlabelkeys-in-only-either-the-scheduler-plugin-or-kube-apiserver))

However, this implementation change could break `matchLabelKeys` of unscheduled pods created before the upgrade
because kube-apiserver only handles `matchLabelKeys` at pods creation, that is,
it doesn't handle `matchLabelKeys` at existing unscheduled pods.	
So, for a safe upgrade path from v1.33 to v1.34, kube-scheduler would handle not only `matchLabelKeys` 
from the default constraints, but also all incoming pods during v1.34. 
We're going to change kube-scheduler to only concern `matchLabelKeys` from the default constraints at v1.35 for efficiency, 
assuming kube-apiserver handles `matchLabelKeys` of all incoming pods.

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

- `k8s.io/kubernetes/pkg/scheduler/framework/plugins/podtopologyspread`: `2025-01-14 JST (The commit hash: ccd2b4e8a719dabe8605b1e6b2e74bb5352696e1)` - `87.5%`
- `k8s.io/kubernetes/pkg/scheduler/framework/plugins/podtopologyspread/plugin.go`: `2025-01-14 JST (The commit hash: ccd2b4e8a719dabe8605b1e6b2e74bb5352696e1)` - `84.8%`
- `k8s.io/kubernetes/pkg/registry/core/pod/strategy.go`: `2025-01-14 JST (The commit hash: ccd2b4e8a719dabe8605b1e6b2e74bb5352696e1)` - `65%`

##### Integration tests

<!--
This question should be filled when targeting a release.
For Alpha, describe what tests will be added to ensure proper quality of the enhancement.

For Beta and GA, add links to added tests together with links to k8s-triage for those tests:
https://storage.googleapis.com/k8s-triage/index.html
-->
- These cases will be added in the existed integration tests:
  - Feature gate enable/disable tests
  - `MatchLabelKeys` in `TopologySpreadConstraint` works as expected
  - Verify no significant performance degradation

- `k8s.io/kubernetes/test/integration/scheduler/filters/filters_test.go`: https://storage.googleapis.com/k8s-triage/index.html?test=TestPodTopologySpreadFilter
- `k8s.io/kubernetes/test/integration/scheduler/scoring/priorities_test.go`: https://storage.googleapis.com/k8s-triage/index.html?test=TestPodTopologySpreadScoring
- `k8s.io/kubernetes/test/integration/scheduler_perf/scheduler_perf_test.go`: https://storage.googleapis.com/k8s-triage/index.html?test=BenchmarkPerfScheduling

##### e2e tests

<!--
This question should be filled when targeting a release.
For Alpha, describe what tests will be added to ensure proper quality of the enhancement.

For Beta and GA, add links to added tests together with links to k8s-triage for those tests:
https://storage.googleapis.com/k8s-triage/index.html

We expect no non-infra related flakes in the last month as a GA graduation criteria.
-->
- These cases will be added in the existed e2e tests:
  - Feature gate enable/disable tests
  - `MatchLabelKeys` in `TopologySpreadConstraint` works as expected

- `k8s.io/kubernetes/test/e2e/scheduling/predicates.go`: https://storage.googleapis.com/k8s-triage/index.html?sig=scheduling
- `k8s.io/kubernetes/test/e2e/scheduling/priorities.go`: https://storage.googleapis.com/k8s-triage/index.html?sig=scheduling

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
- Feature implemented behind feature gate.
- Unit and integration tests passed as designed in [TestPlan](#test-plan).

#### Beta
- Feature is enabled by default
- Benchmark tests passed, and there is no performance degradation.
- Update documents to reflect the changes.

#### GA
- No negative feedback.
- Update documents to reflect the changes.

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

In the event of an upgrade, kube-apiserver will start to accept and store the field `MatchLabelKeys`.

In the event of a downgrade, kube-apiserver will reject pod creation with `matchLabelKeys` in `TopologySpreadConstraint`. 
But, regarding existing pods, we leave `matchLabelKeys` and generated `LabelSelector` even after downgraded.
kube-scheduler will ignore `MatchLabelKeys` if it was set in the cluster-level default constraints configuration.

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

There's no version skew issue.

We changed the implementation design between v1.34 and v1.35, but we designed the change not to involve any version skew issue
as described at [[v1.34] design change and a safe upgrade path](#v134-design-change-and-a-safe-upgrade-path).

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
  - Feature gate name: `MatchLabelKeysInPodTopologySpread`
  - Components depending on the feature gate: `kube-scheduler`, `kube-apiserver`

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
The feature can be disabled in Alpha and Beta versions by restarting 
kube-apiserver and kube-scheduler with feature-gate off.
One caveat is that pods that used the feature will continue to have the 
MatchLabelKeys field set and the corresponding LabelSelector even after 
disabling the feature gate.
In terms of Stable versions, users can choose to opt-out by not setting 
the matchLabelKeys field.

###### What happens if we reenable the feature if it was previously rolled back?
Newly created pods need to follow this policy when scheduling. Old pods will 
not be affected.

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
No. The unit tests that are exercising the `switch` of feature gate itself  will be added.

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
It won't impact already running workloads because it is an opt-in feature in kube-apiserver 
and kube-scheduler.
But during a rolling upgrade, if some apiservers have not enabled the feature, they will not
be able to accept and store the field "MatchLabelKeys" and the pods associated with these 
apiservers will not be able to use this feature. As a result, pods belonging to the 
same deployment may have different scheduling outcomes.


###### What specific metrics should inform a rollback?

<!--
What signals should users be paying attention to when the feature is young
that might indicate a serious problem?
-->
- If the metric `schedule_attempts_total{result="error|unschedulable"}` increased significantly after pods using this feature are added.
- If the metric `plugin_execution_duration_seconds{plugin="PodTopologySpread"}` increased to higher than 100ms on 90% after pods using this feature are added.  


###### Were upgrade and rollback tested? Was the upgrade->downgrade->upgrade path tested?

<!--
Describe manual testing that was done and the outcomes.
Longer term, we may want to require automated upgrade/rollback tests, but we
are missing a bunch of machinery and tooling and can't do that now.
-->
Yes, it was tested manually by following the steps below, and it was working at intended.
1. create a kubernetes cluster v1.26 with 3 nodes where `MatchLabelKeysInPodTopologySpread` feature is disabled.
2. deploy a deployment with this yaml
```yaml
apiVersion: apps/v1
kind: Deployment
metadata:
  name: nginx
spec:
  replicas: 12 
  selector:
    matchLabels:
      foo: bar
  template:
    metadata:
      labels:
        foo: bar
    spec:
      restartPolicy: Always
      containers:
      - name: nginx
        image: nginx:1.14.2
      topologySpreadConstraints:
        - maxSkew: 1
          topologyKey: kubernetes.io/hostname
          whenUnsatisfiable: DoNotSchedule
          labelSelector:
            matchLabels:
              foo: bar
          matchLabelKeys:
            - pod-template-hash
```
3. pods spread across nodes as 4/4/4
4. update the deployment nginx image to `nginx:1.15.0`
5. pods spread across nodes as 5/4/3
6. delete deployment nginx
7. upgrade kubenetes cluster to v1.27 (at master branch) while `MatchLabelKeysInPodTopologySpread` is enabled.
8. deploy a deployment nginx like step2
9. pods spread across nodes as 4/4/4
10. update the deployment nginx image to `nginx:1.15.0`
11. pods spread across nodes as 4/4/4
12. delete deployment nginx
13. downgrade kubenetes cluster to v1.26  where `MatchLabelKeysInPodTopologySpread` feature is enabled.
14. deploy a deployment nginx like step2
15. pods spread across nodes as 4/4/4
16. update the deployment nginx image to `nginx:1.15.0`
17. pods spread across nodes as 4/4/4

###### Is the rollout accompanied by any deprecations and/or removals of features, APIs, fields of API types, flags, etc.?

<!--
Even if applying deprecation policies, they may still surprise some users.
-->
No.

### Monitoring Requirements

<!--
This section must be completed when targeting beta to a release.
-->

###### How can an operator determine if the feature is in use by workloads?

<!--
Ideally, this should be a metric. Operations against the Kubernetes API (e.g.,
checking if there are objects with field X set) may be a last resort. Avoid
logs or events for this purpose.
-->
Operator can query pods that have the `pod.spec.topologySpreadConstraints.matchLabelKeys` field set to determine if the feature is in use by workloads. 

###### How can someone using this feature know that it is working for their instance?

<!--
For instance, if this is a pod-related feature, it should be possible to determine if the feature is functioning properly
for each individual pod.
Pick one more of these and delete the rest.
Please describe all items visible to end users below with sufficient detail so that they can verify correct enablement
and operation of this feature.
Recall that end users cannot usually observe component logs or access metrics.
-->

- [x] Other (treat as last resort)
  - Details: We can determine if this feature is being used by checking pods that have only `MatchLabelKeys` set in `TopologySpreadConstraint`.

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
Metric plugin_execution_duration_seconds{plugin="PodTopologySpread"} <= 100ms on 90-percentile.

###### What are the SLIs (Service Level Indicators) an operator can use to determine the health of the service?

<!--
Pick one more of these and delete the rest.
-->

- [x] Metrics
  - Component exposing the metric: kube-scheduler
    - Metric name: `plugin_execution_duration_seconds{plugin="PodTopologySpread"}`
    - Metric name: `schedule_attempts_total{result="error|unschedulable"}`

###### Are there any missing metrics that would be useful to have to improve observability of this feature?

<!--
Describe the metrics themselves and the reasons why they weren't added (e.g., cost,
implementation difficulties, etc.).
-->
Yes. It's helpful if we have the metrics to see which plugins affect to scheduler's decisions in Filter/Score phase. 
There is the related issue: https://github.com/kubernetes/kubernetes/issues/110643 . It's very big and still on the way.

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
No.

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
Yes. there is an additional work:
kube-apiserver uses the keys in `matchLabelKeys` to look up label values from the pod, 
and change `LabelSelector` according to them. 
kube-scheduler also handles matchLabelKeys if the cluster-level default constraints has it.
The impact in the latency of pod creation request in kube-apiserver and the scheduling latency 
should be negligible.

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

The Troubleshooting section currently serves the `Playbook` role. We may consider
splitting it into a dedicated `Playbook` document (potentially with some monitoring
details). For now, we leave it here.
-->

###### How does this feature react if the API server and/or etcd is unavailable?
If the API server and/or etcd is not available, this feature will not be available. 
This is because the kube-scheduler needs to update the scheduling results to the pod via the API server/etcd.

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
N/A

###### What steps should be taken if SLOs are not being met to determine the problem?
- Check the metric `plugin_execution_duration_seconds{plugin="PodTopologySpread"}` to determine 
  if the latency increased. If increased, it means this feature may increased scheduling latency. 
  You can disable the feature `MatchLabelKeysInPodTopologySpread` to see if it's the cause of the 
  increased latency.
- Check the metric `schedule_attempts_total{result="error|unschedulable"}` to determine if the number 
  of attempts increased. If increased, You need to determine the cause of the failure by the event of 
  the pod. If it's caused by plugin `PodTopologySpread`, You can further analyze this problem by looking 
  at the kube-scheduler log.


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
 - 2022-03-17: Initial KEP
 - 2022-06-08: KEP merged
 - 2023-01-16: Graduate to Beta
 - 2025-01-23: Change the implementation design to be aligned with PodAffinity's `matchLabelKeys`

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

### use pod generateName
Use `pod.generateName` to distinguish new/old pods that belong to the 
revisions of the same workload in scheduler plugin. It's decided not to 
support because of the following reason: scheduler needs to ensure universal 
and scheduler plugin shouldn't have special treatment for any labels/fields.

### implement MatchLabelKeys in only either the scheduler plugin or kube-apiserver
Technically, we can implement this feature within the PodTopologySpread plugin only;
merging the key-value labels corresponding to `MatchLabelKeys` into `LabelSelector` internally 
within the plugin before calculating the scheduling results.
This is the actual implementation up to 1.33.
But, it may confuse users because this behavior would be different from PodAffinity's `MatchLabelKeys`.

Also, we cannot implement this feature only within kube-apiserver because it'd make it
impossible to handle `MatchLabelKeys` within the cluster-level default constraints 
in the scheduler configuration in the future (see https://github.com/kubernetes/kubernetes/issues/129198).

So we decided to go with the design that implements this feature within both 
the PodTopologySpread plugin and kube-apiserver.
Although the final design has a downside requiring us to maintain two implementations handling `MatchLabelKeys`,
each implementation is simple and we regard the risk of increased maintenance overhead as fairly low.

## Infrastructure Needed (Optional)

<!--
Use this section if you need things from the project/SIG. Examples include a
new subproject, repos requested, or GitHub details. Listing these here allows a
SIG to get the process for these resources started right away.
-->
