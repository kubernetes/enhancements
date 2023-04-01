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

# KEP-3614: Use etcd's learner mode in kubeadm

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
    - [Risk: insufficient testing by kubeadm users](#risk-insufficient-testing-by-kubeadm-users)
      - [Mitigation](#mitigation)
    - [Risk: unstable implementation of learner mode](#risk-unstable-implementation-of-learner-mode)
      - [Mitigation](#mitigation-1)
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

This KEP proposes to enhance kubeadm to start using etcd's learner mode which was introduced in version 3.4. [The release notes for etcd 3.4](https://etcd.io/docs/v3.3/learning/learner/#features-in-v34) suggest a number of benefits of using this method. The proposal aims to add the new mode as a standard kubeadm / Kubernetes feature gate that is graduated over the period of one year or more, while collecting feedback from all kubeadm users.

## Motivation

<!--
This section is for explicitly listing the motivation, goals, and non-goals of
this KEP.  Describe why the change is important and the benefits to users. The
motivation section can optionally provide links to [experience reports] to
demonstrate the interest in a KEP within the wider Kubernetes community.

[experience reports]: https://github.com/golang/go/wiki/ExperienceReports
-->

Kubeadm currently adds all members in the "old way" that etcd supported, that is to add them as voting members from the beginning. If added as learners instead, such members would not disrupt the cluster quorum if they end up being faulty. The "old way" has proven problematic in cases where kubeadm attempts to add a etcd cluster member from a control plane node running on slower infrastructure. In such cases users have to manually interfere and remove the faulty member, by using tools such as etcdctl.

### Goals

<!--
List the specific goals of the KEP. What is it trying to achieve? How will we
know that this has succeeded?
-->

- Add a new code path in kubeadm that can be used to deploy etcd with learner mode enabled.
- Use a new feature gate EtcdLearnerMode that can be used to toggle the feature until graduation to GA.
- Deprecate and remove the "old way" of adding members.
- Keep the number of learners as per the etcd default value (1). Users can customize the value if they wish.

### Non-Goals

- Support both the "old way" and "learner mode" in kubeadm as a toggle in the kubeadm API. Ideally we should support only a single, stable, community approved code path.
- Attempt to improve support for parallel join of control plane nodes. It is currently supported tentatively in the "old way".
- Attempt to improve resilience of control plane node join in case of machine misconfiguration, such as NTP offsets.

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

As a kubeadm user, I wish that my HA cluster is more resilient to etcd member failures during addition of new members at cluster bring up time due to slow infrastructure.

#### Story 2

As a kubeadm user, I wish that my HA cluster is constructed following the recommendation by etcd maintainers and using the latest features - i.e. to use learner mode instead of adding all new members as voting.

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

#### Risk: insufficient testing by kubeadm users

Once the new code path is added and the logic is controlled by a feature gate, the feature gate will be in Alpha state or disabled by default. Even if e2e tests are added we need to notify users that we are making this important change to etcd and that they start testing it ASAP during Alpha, but not in production.

##### Mitigation

Notify users on all possible communication channels: Slack, ML, Reddit, Twitter, etc. Keep umbrella issue as a place for discussion and user feedback. Attempt gathering feedback from parties that build product on top of kubeadm.

#### Risk: unstable implementation of learner mode

Once the new feature is added we need to test the stability of the new code path. The current "old way" of constructing the etcd cluster has proven stable and is used by all kubeadm HA users of the "stacked etcd" topology. It has also proven to allow concurrent join of control plane nodes with their stacked etcd members. With the addition of learner mode we are introducing the potential that once the feature graduates to Beta it would be enabled by default and might cause unforeseen issues.

##### Mitigation

Once again the mitigation here would be to notify all possible channels and ask consumers of kubeadm to test the feature before it moves to Beta. Testing on slow infrastructure might be a key point to mitigate possible issues.

## Design Details

<!--
This section should contain enough information that the specifics of your
change are understandable. This may include API specs (though not always
required) or even code snippets. If there's any ambiguity about HOW your
proposal will be implemented, this is the place to discuss them.
-->

Currently most of the logic of stacked etcd member support in kubeadm is centralized around a couple of files in the source code. These files contain the etcd client wrapped logic and the logic for maintaining a static pod manifest for the etcd server instance.

With the introduction of the new feature gate EtcdLearnerMode a new code path must be created. Preferably the number of "if EtcdLearnerMode" branches in the code should be minimized. Kubeadm currently has some sensitive timeouts while adding etcd members the "old way". Waiting for learners to become voting members would require some modifications in kubeadm in terms of how we wait for a member to be added. Some details can be found in the [official etcd documentation](https://etcd.io/docs/v3.3/learning/learner/#features-in-v34).

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

- `<package>`: `<date>` - `<test coverage>`

-->

New unit tests must be added for all code paths that use the EtcdLearnerMode feature gate. Once the feature graduates to GA, these unit tests must be merged as part of the default unit tests for testing the kubeadm "stacked etcd" logic.

##### Integration tests

<!--
This question should be filled when targeting a release.
For Alpha, describe what tests will be added to ensure proper quality of the enhancement.

For Beta and GA, add links to added tests together with links to k8s-triage for those tests:
https://storage.googleapis.com/k8s-triage/index.html

- <test>: <link to test coverage>
-->

N/A

##### e2e tests

<!--
This question should be filled when targeting a release.
For Alpha, describe what tests will be added to ensure proper quality of the enhancement.

For Beta and GA, add links to added tests together with links to k8s-triage for those tests:
https://storage.googleapis.com/k8s-triage/index.html

We expect no non-infra related flakes in the last month as a GA graduation criteria.

- <test>: <link to test coverage>
-->

A new e2e test must be added as part of the [kubeadm dashboard](https://k8s-testgrid.appspot.com/sig-cluster-lifecycle-kubeadm). All tests in this dashboard use the [kinder](https://github.com/kubernetes/kubeadm/tree/main/kinder) tool.

- During Alpha (disabled by default): add a new e2e test that enables the feature gate EtcdLearnerMode
- During Beta (enabled by default): modify the e2e test to test the feature gate EtcdLearnerMode as disabled
- During GA (locked to enabled): remove the e2e test as the logic will be exercised in all existing kubeadm e2e tests

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

- Feature implemented behind the feature gate EtcdLearnerMode
- Initial unit and e2e tests completed and enabled
- [Document the feature gate](https://kubernetes.io/docs/reference/setup-tools/kubeadm/kubeadm-init/#feature-gates).

#### Beta

- Gather feedback from developers and surveys
- Make unit and e2e test changes
- Update the feature gate documentation

#### GA

- Gather feedback from developers and surveys
- Update unit tests
- Remove e2e tests as this will be the only code path for adding etcd members and it will be tested by all existing kubeadm e2e tests
- Update the feature gate documentation

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

- N/A -> Alpha: users can patch their `ClusterConfiguration` in the `kube-system/kubeadm-config` ConfigMap to before calling `kubeadm upgrade apply` This will allow them to enable learner mode in case they wish to add more etcd members to this cluster. This scenario is anticipated as rare, because usually users maintain a stable control plane with 3 or more members before upgrading it. But it is still plausible and can be documented in the feature gate documentation.
- Alpha -> Beta: similarly to the previous stage users can modify the `ClusterConfiguration` to disable the feature gate during upgrade. This will allow them to use the "old way", in case they wish to add more etcd members to the cluster while the feature gate is enabled by default.
- Beta -> GA: users could no longer patch the `ClusterConfiguration` to opt-out of the feature and it will be locked to default.

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

One important point to make would be that kubeadm must handle a case where the user locked their etcd server version to version < 3.4. This would mean that they must get a sensible error in the lines of "etcd learner mode is not supported by this etcd version" and the control plane with stacked etcd initialization should fail.

All etcd versions that are > 3.4 should be treated as supported by the EtcdLearnerMode feature gate.

If EtcdLearnerMode goes GA, but the user prefers to stay on etcd version < 3.4, their existing cluster will continue to work but they will not be able to add new stacked etcd members. For new clusters the combination of EtcdLearnerMode (GA) and etcd version < 3.4 will not be supported.

## Production Readiness Review Questionnaire

kubeadm is considered an "out of tree" component and PRR is out of scope.

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

- 2022-05-10: KEP draft created
- 2022-12-17(v1.27): add the experimental (alpha) feature gate EtcdLearnerMode and initial implementation <https://github.com/kubernetes/kubernetes/pull/113318>
- 2023-01-28(v1.27): add e2e test for etcd learner mode <https://github.com/kubernetes/kubeadm/pull/2807>

## Drawbacks

<!--
Why should this KEP _not_ be implemented?
-->

The implementation and enablement of EtcdLearnerMode by default hides a number of risks around stability. The "old way" has been tested for years and consumed by many users. By modifying this code path we are introducing potential for user complains about HA cluster creation and maintenance with kubeadm. Sufficient testing and gathering feedback from users would be mandatory.

## Alternatives

<!--
What other approaches did you consider, and why did you rule them out? These do
not need to be as detailed as the proposal, but should include enough
information to express the idea and why it was not acceptable.
-->

N/A

## Infrastructure Needed (Optional)

<!--
Use this section if you need things from the project/SIG. Examples include a
new subproject, repos requested, or GitHub details. Listing these here allows a
SIG to get the process for these resources started right away.
-->

N/A
