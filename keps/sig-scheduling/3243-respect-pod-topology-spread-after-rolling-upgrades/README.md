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
  - [[v1.38] GA design](#v138-ga-design)
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

- [ ] (R) Enhancement issue in release milestone, which links to KEP dir in [kubernetes/enhancements] (not the initial KEP PR)
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
The pod topology spread feature allows users to define the group of pods over 
which spreading is applied using a LabelSelector. This means the user should 
know the exact label key and value when defining the pod spec.

This KEP proposes a complementary field to LabelSelector named `MatchLabelKeys` in
`TopologySpreadConstraint` which represents a set of label keys only. 
At a pod creation, kube-apiserver will use those keys to look up label values from the incoming pod 
and those key-value labels will be merged with existing `LabelSelector` to identify the group of existing pods over 
which the spreading skew will be calculated.
Cluster-level default constraints in the scheduler configuration do not support
`matchLabelKeys`; adding that support is outside the scope of this KEP.


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

- Adding `matchLabelKeys` to cluster-level default constraints in the
  kube-scheduler configuration.

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

Cluster-level default constraints in the scheduler configuration do not support
`matchLabelKeys`. Adding that support is outside the scope of this KEP and was
discussed separately in [kubernetes/kubernetes#129198].

During Alpha and Beta, the feature is guarded by the
`MatchLabelKeysInPodTopologySpread` feature gate. If the feature is disabled,
the `matchLabelKeys` field and corresponding `labelSelector` are preserved when
they already exist in a persisted Pod object; otherwise, kube-apiserver rejects
creation of a Pod that sets the field. At GA, the feature gate is locked on.

[kubernetes/kubernetes#129198]: https://github.com/kubernetes/kubernetes/issues/129198

### [v1.34] design change and a safe upgrade path
Previously, kube-scheduler just internally handled `matchLabelKeys` before the calculation of scheduling results.
But, we changed the implementation design to the current form to make the design align with PodAffinity's `matchLabelKeys`. 
(See the detailed discussion in [the alternative section](#implement-matchlabelkeys-in-only-either-the-scheduler-plugin-or-kube-apiserver))

However, this implementation change could break `matchLabelKeys` for
unscheduled Pods created before the upgrade because kube-apiserver only applies
the mutation at Pod creation. For a safe upgrade from v1.33 to v1.34,
kube-scheduler retained its legacy merge for all incoming Pods. That
compatibility path was originally planned for removal in v1.35, but remained in
place throughout the Beta period.

Also, in case of bugs in this new design, users can disable this feature through a new feature flag, 
`MatchLabelKeysInPodTopologySpreadSelectorMerge` (enabled by default).
(See more details in [Feature Enablement and Rollback](#feature-enablement-and-rollback))

### [v1.38] GA design

Both `MatchLabelKeysInPodTopologySpread` and
`MatchLabelKeysInPodTopologySpreadSelectorMerge` graduate to GA together and
are locked on. The API behavior introduced in v1.34 is the stable behavior:
kube-apiserver resolves `matchLabelKeys` once, when a Pod is created, and
persists the resulting requirements in `labelSelector`.

The scheduler-side compatibility merge for explicit Pod constraints is removed
at GA. The scheduler consumes the persisted `labelSelector` and does not
re-resolve `matchLabelKeys` from the Pod's current labels. This completes the
transition to the API-server-owned behavior and avoids adding a second,
potentially different requirement if a label named by `matchLabelKeys` is
updated after Pod creation. Label updates still do not rewrite the persisted
selector, as described in [Risks and Mitigations](#risks-and-mitigations).

The compatibility path has been enabled by default since v1.34. A cluster using
the supported upgrade order upgrades kube-apiserver before kube-scheduler, so
new Pods have persisted selectors before a GA scheduler relies on them. The
handling of older pending Pods and explicitly disabled Beta gates is described
in [Version Skew Strategy](#version-skew-strategy).

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

Existing unit tests cover:

- API-server mutation and the behavior when either Beta feature gate is
  disabled in
  [`pkg/registry/core/pod/strategy_test.go`](https://github.com/kubernetes/kubernetes/blob/ca0942e6fbf0b562bd230c9fff6f7048439d0e65/pkg/registry/core/pod/strategy_test.go#L2733-L3147).
- dropping the field when disabled, and selecting the validation behavior for
  old Pods, in
  [`pkg/api/pod/util_test.go`](https://github.com/kubernetes/kubernetes/blob/ca0942e6fbf0b562bd230c9fff6f7048439d0e65/pkg/api/pod/util_test.go#L2550-L3131).
- old and new validation rules in
  [`pkg/apis/core/validation/validation_test.go`](https://github.com/kubernetes/kubernetes/blob/ca0942e6fbf0b562bd230c9fff6f7048439d0e65/pkg/apis/core/validation/validation_test.go#L26123-L26640).
- filter and score behavior in
  [`pkg/scheduler/framework/plugins/podtopologyspread`](https://github.com/kubernetes/kubernetes/tree/ca0942e6fbf0b562bd230c9fff6f7048439d0e65/pkg/scheduler/framework/plugins/podtopologyspread).

The GA implementation will update the scheduler unit tests to verify that
explicit Pod constraints use the persisted selector without re-resolving
`matchLabelKeys` from current Pod labels.

##### Integration tests

<!--
This question should be filled when targeting a release.
For Alpha, describe what tests will be added to ensure proper quality of the enhancement.

For Beta and GA, add links to added tests together with links to k8s-triage for those tests:
https://storage.googleapis.com/k8s-triage/index.html
-->
Existing integration tests cover `matchLabelKeys` in both filtering and
scoring:

- [`TestPodTopologySpreadFilter`](https://github.com/kubernetes/kubernetes/blob/ca0942e6fbf0b562bd230c9fff6f7048439d0e65/test/integration/scheduler/filters/filters_test.go#L2190-L2234):
  [triage results](https://storage.googleapis.com/k8s-triage/index.html?test=TestPodTopologySpreadFilter)
- [`TestPodTopologySpreadScoring`](https://github.com/kubernetes/kubernetes/blob/ca0942e6fbf0b562bd230c9fff6f7048439d0e65/test/integration/scheduler/scoring/priorities_test.go#L1073-L1126):
  [triage results](https://storage.googleapis.com/k8s-triage/index.html?test=TestPodTopologySpreadScoring)

Before GA, an integration test will create a scheduling-gated Pod, verify that
kube-apiserver persisted the selector derived from `matchLabelKeys`, update the
corresponding Pod label, remove the scheduling gate, and verify that scheduling
continues to use the selector persisted at creation time. This test covers the
removal of the scheduler-side compatibility merge.

The GA change removes per-cycle selector construction from kube-scheduler and
does not add a new scheduling operation. Existing
[`scheduler_perf`](https://github.com/kubernetes/kubernetes/tree/master/test/integration/scheduler_perf)
results will be monitored for regression.

##### e2e tests

<!--
This question should be filled when targeting a release.
For Alpha, describe what tests will be added to ensure proper quality of the enhancement.

For Beta and GA, add links to added tests together with links to k8s-triage for those tests:
https://storage.googleapis.com/k8s-triage/index.html

We expect no non-infra related flakes in the last month as a GA graduation criteria.
-->
A conformance test will be added to
[`test/e2e/scheduling/predicates.go`](https://github.com/kubernetes/kubernetes/blob/master/test/e2e/scheduling/predicates.go).
It will create an intentionally skewed old revision, then create Pods for a new
revision using `matchLabelKeys` and verify that the new revision is spread
independently across two topology domains. The test must run for at least two
weeks without a non-infrastructure flake before GA code freeze.

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
- Both `MatchLabelKeysInPodTopologySpread` and
  `MatchLabelKeysInPodTopologySpreadSelectorMerge` have been enabled by default
  for at least two releases; the selector-merge behavior has been enabled by
  default since v1.34.
- Remove the scheduler-side compatibility merge for explicit Pod constraints
  and verify that the persisted selector is the single source of truth.
- Unit and integration tests cover API-server mutation, validation, Beta gate
  transitions, and scheduling after a label named by `matchLabelKeys` changes.
- A conformance test covers spreading each rollout revision independently and
  has no non-infrastructure flakes for at least two weeks.
- No unresolved correctness or scalability regressions attributable to this
  feature. In particular, monitor the fix for empty non-nil selectors from
  [kubernetes/kubernetes#141340] for at least two weeks.
- Production Readiness Review is approved for GA.
- User-facing documentation is updated for the stable behavior and removal of
  the Beta feature-gate opt-out.

[kubernetes/kubernetes#141340]: https://github.com/kubernetes/kubernetes/pull/141340

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

No action is required for clusters that use the default feature-gate settings.
Both gates are enabled by default in all supported source releases. Follow the
standard control-plane order and upgrade kube-apiserver before kube-scheduler.

If either Beta gate was explicitly disabled, enable both gates on every
kube-apiserver before upgrading kube-scheduler to the GA release. Recreate any
still-pending Pod whose `matchLabelKeys` values have not been materialized as
`In` requirements in its persisted `labelSelector`.

**Downgrade**

A downgrade to a supported Beta release preserves `matchLabelKeys` and the
generated `labelSelector`; both gates are enabled by default in that release.
An older scheduler may merge the same key-value requirements again, which is
idempotent. If an administrator disables the feature after downgrade, new Pods
that set `matchLabelKeys` are rejected, while existing Pods retain both the
field and the selector that was persisted at creation time.

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

The supported order requires kube-apiserver to be upgraded before
kube-scheduler and does not allow kube-scheduler to be newer than
kube-apiserver. Therefore, when a GA scheduler stops resolving
`matchLabelKeys`, every supported kube-apiserver version has already persisted
the generated selector by default.

A Beta scheduler running with a GA kube-apiserver remains compatible. It may
merge requirements that are already in the persisted selector, but identical
requirements are idempotent and do not change the selected Pods.

Pods created by a pre-v1.34 kube-apiserver, or while either Beta gate was
disabled, may not contain the generated requirements. Such Pods can live longer
than the supported component-skew window. Administrators using those
configurations must enable both gates and recreate any affected pending Pods
before upgrading kube-scheduler to the GA version. Running Pods are unaffected.

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

- `MatchLabelKeysInPodTopologySpread` enables the `matchLabelKeys` field in
  `TopologySpreadConstraint`.
- `MatchLabelKeysInPodTopologySpreadSelectorMerge` enables the API-server
  mutation and validation behavior described in
  [[v1.34] design change and a safe upgrade path](#v134-design-change-and-a-safe-upgrade-path).
  During Beta, disabling this gate while leaving
  `MatchLabelKeysInPodTopologySpread` enabled selects the legacy scheduler-owned
  behavior. Enabling the selector-merge gate alone has no effect.

The `MatchLabelKeysInPodTopologySpreadSelectorMerge` feature flag has been added in v1.34 and enabled by default.
This flag can be disabled to revert [the implementation design change in v1.34](#v134-design-change-and-a-safe-upgrade-path) 
and go back to the previous behavior in case of bug.

At GA, both feature gates are locked on and the legacy scheduler-owned behavior
is removed. Users opt out by omitting `matchLabelKeys` from their Pod template.

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
  - Components depending on the feature gate: `kube-apiserver`
- [x] Feature gate (also fill in values in `kep.yaml`)
  - Feature gate name: `MatchLabelKeysInPodTopologySpreadSelectorMerge`
  - Components depending on the feature gate: `kube-apiserver`

###### Does enabling the feature change any default behavior?

<!--
Any change of default behavior may be surprising to users or break existing
automations, so be extremely careful here.
-->
There is no change for clusters using the default settings because both gates
were enabled by default before GA. A cluster that explicitly disabled either
Beta gate will no longer be able to keep it disabled after upgrading to GA.

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
kube-apiserver and kube-scheduler with the feature gates off. Pods that already
used the feature retain `matchLabelKeys` and the corresponding
`labelSelector`.

The feature cannot be disabled after GA because both feature gates are locked
on. Users can opt out for new Pods by not setting `matchLabelKeys`.

###### What happens if we reenable the feature if it was previously rolled back?
In Alpha and Beta, newly created Pods use the feature again. Existing Pods are
not remutated. This question is not applicable after GA because the gates are
locked on.

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
Yes. Unit tests cover mutation with both gates enabled and disabled, retaining
persisted data across gate transitions, field dropping, and validation of old
Pods:

- [`pkg/registry/core/pod/strategy_test.go`](https://github.com/kubernetes/kubernetes/blob/ca0942e6fbf0b562bd230c9fff6f7048439d0e65/pkg/registry/core/pod/strategy_test.go#L2733-L3147)
- [`pkg/api/pod/util_test.go`](https://github.com/kubernetes/kubernetes/blob/ca0942e6fbf0b562bd230c9fff6f7048439d0e65/pkg/api/pod/util_test.go#L2550-L3131)
- [`pkg/apis/core/validation/validation_test.go`](https://github.com/kubernetes/kubernetes/blob/ca0942e6fbf0b562bd230c9fff6f7048439d0e65/pkg/apis/core/validation/validation_test.go#L26123-L26640)

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
The feature and selector-merge behavior have been enabled by default since
v1.27 and v1.34 respectively, so a default-configured rolling upgrade does not
change behavior for existing or new workloads.

The GA scheduler stops performing the legacy merge for explicit Pod
constraints. In the supported upgrade order, all kube-apiservers are upgraded
first and have persisted the selector before the scheduler sees a new Pod. If
either gate was explicitly disabled on a Beta kube-apiserver, the administrator
must follow the additional steps in
[Upgrade / Downgrade Strategy](#upgrade--downgrade-strategy). Already running
Pods are unaffected in either case.


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
The v1.26 to v1.27 upgrade and rollback path was tested manually as follows:
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

Before GA, the integration test described in [Test Plan](#test-plan) will cover
the v1.34 transition boundary: the API server persists the selector, a label is
updated while scheduling is gated, and the scheduler uses the persisted value
after the gate is removed. The conformance test will cover independent spreading
across rollout revisions.

###### Is the rollout accompanied by any deprecations and/or removals of features, APIs, fields of API types, flags, etc.?

<!--
Even if applying deprecation policies, they may still surprise some users.
-->
Yes. Both feature gates graduate to GA and are locked on, and the temporary
scheduler-side merge for explicit Pod constraints is removed. No API type,
field, or user-facing capability is removed.

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
An operator can query Pods whose
`spec.topologySpreadConstraints[*].matchLabelKeys` field is non-empty. No
feature-specific metric is needed because usage is explicitly recorded in each
Pod spec.

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
  - Details: Get the created Pod and verify that each key present in both the
    Pod labels and `matchLabelKeys` also appears as an `In` requirement in the
    persisted `labelSelector`. The user can then compare the distribution of
    matching Pods across the topology domains.

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
The 90th percentile of
`plugin_execution_duration_seconds{plugin="PodTopologySpread"}` should remain
below 100ms.

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
No feature-specific metric is missing. General PodTopologySpread observability
improvements were tracked in
[#110643](https://github.com/kubernetes/kubernetes/issues/110643) and
implemented by [#115082](https://github.com/kubernetes/kubernetes/pull/115082)
and [#118025](https://github.com/kubernetes/kubernetes/pull/118025).

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
Yes. For every key in `matchLabelKeys` that is present on the incoming Pod,
kube-apiserver persists one additional `In` requirement in that Pod's
`labelSelector`. The size increase is proportional to the total length of the
matched label keys and values. No additional API objects are created.

###### Will enabling / using this feature result in increasing time taken by any operations covered by existing SLIs/SLOs?

<!--
Look at the [existing SLIs/SLOs].

Think about adding additional work or introducing new steps in between
(e.g. need to do X to start a container), etc. Please describe the details.

[existing SLIs/SLOs]: https://git.k8s.io/community/sig-scalability/slos/slos.md#kubernetes-slisslos
-->
Yes. On Pod creation, kube-apiserver looks up each key in `matchLabelKeys` and
adds the corresponding requirement to `labelSelector`. This work is linear in
the number of specified keys and only occurs once per Pod. At GA,
kube-scheduler performs no additional `matchLabelKeys` processing and evaluates
the persisted selector through the existing PodTopologySpread path. The impact
on Pod creation and scheduling latency is expected to be negligible.

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
- A pending Pod created before v1.34, or while either Beta gate was disabled,
  may have `matchLabelKeys` without the corresponding requirements in its
  persisted `labelSelector`.
  - Detection: inspect the Pod spec and compare the Pod labels and
    `matchLabelKeys` with the persisted selector requirements.
  - Mitigation: recreate the pending Pod after all kube-apiservers have both
    gates enabled. Already running Pods are unaffected.
  - Diagnostics: scheduler events show the result of the effective
    PodTopologySpread constraint; there is no dedicated log message because
    the scheduler intentionally treats the persisted selector as authoritative.
  - Testing: the GA integration test described in [Test Plan](#test-plan)
    covers the persisted-selector boundary.

###### What steps should be taken if SLOs are not being met to determine the problem?
- Check `plugin_execution_duration_seconds{plugin="PodTopologySpread"}` for a
  latency increase correlated with newly created Pods that use the feature.
  During Beta, an administrator can disable the gates to compare behavior. At
  GA, test with a newly created equivalent workload that omits
  `matchLabelKeys`; do not edit immutable constraints on an existing Pod.
- Check `schedule_attempts_total{result="error|unschedulable"}` and the events
  of affected Pods. If PodTopologySpread rejected a Pod, inspect the persisted
  selector, the labels of matching Pods, and their topology-domain
  distribution, then review kube-scheduler logs if necessary.


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
 - 2025-04-07: Add a new feature flag `MatchLabelKeysInPodTopologySpreadSelectorMerge` and update milestone
 - 2026-09-02: Target both feature gates for GA in v1.38

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

From v1.34, kube-apiserver also resolves the keys and persists the resulting
selector. The scheduler implementation was retained temporarily so that pending
Pods created before that transition continued to work during an upgrade.

At GA, kube-apiserver is the only component that resolves `matchLabelKeys` for
explicit Pod constraints. The scheduler consumes the persisted selector. This
keeps PodTopologySpread aligned with PodAffinity and avoids maintaining two
sources of truth. Supporting `matchLabelKeys` in cluster-level default
constraints would require separate scheduler behavior and is outside the scope
of this KEP.

## Infrastructure Needed (Optional)

<!--
Use this section if you need things from the project/SIG. Examples include a
new subproject, repos requested, or GitHub details. Listing these here allows a
SIG to get the process for these resources started right away.
-->
