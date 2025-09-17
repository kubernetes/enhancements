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
# KEP-4471: Make a control-plane's kubelet point to the local API Server on kubeadm join

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
    - [Story 3](#story-3)
    - [Story 4](#story-4)
  - [Notes/Constraints/Caveats (Optional)](#notesconstraintscaveats-optional)
    - [Caveats](#caveats)
  - [Risks and Mitigations](#risks-and-mitigations)
    - [Risk: Duration for joining a control-plane node may change](#risk-duration-for-joining-a-control-plane-node-may-change)
    - [Risk: control-plane node stability](#risk-control-plane-node-stability)
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
  - [Take the risk of violating Kubernetes version skew policy](#take-the-risk-of-violating-kubernetes-version-skew-policy)
  - [Use external etcd](#use-external-etcd)
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

This KEP proposes to enhance kubeadm to make kubelets on control plane nodes point
to the local kube-apiserver.
Currently the kubelet always points to the load balanced control plane endpoint
which could result in violations of version skew policy.

## Motivation

<!--
This section is for explicitly listing the motivation, goals, and non-goals of
this KEP.  Describe why the change is important and the benefits to users. The
motivation section can optionally provide links to [experience reports] to
demonstrate the interest in a KEP within the wider Kubernetes community.

[experience reports]: https://github.com/golang/go/wiki/ExperienceReports
-->

When a control plane node joins a cluster, the kubelet will bootstrap itself against
the load balanced kube-apiserver endpoint.

This kube-apiserver endpoint may forward requests coming from the local kubelet to
the kube-apiserver running on any of the existing control plane nodes, including
the one running on the same node as the kubelet.

When doing a immutable rolling upgrade (as e.g. done when using [Cluster API]) the
kubelet's minor Kubernetes version could be newer than the version of kube-apiserver
running on the previously existing control plane nodes.

In that case this would lead to a violation of the [version skew policy] rule:

> `kubelet` must not be newer than `kube-apiserver`.

Because of that is not guaranteed to work and this .

[Cluster API]: https://github.com/kubernetes-sigs/cluster-api
[version skew policy]: https://kubernetes.io/releases/version-skew-policy/

### Goals

<!--
List the specific goals of the KEP. What is it trying to achieve? How will we
know that this has succeeded?
-->

- Add a new code path in kubeadm that can be used to join control plane nodes
  without potentially violating the version skew policy, by letting the kubelet
  only communicate with the local kube-apiserver.
- Also adjust init and upgrade to result in the same configuration.
- Use a new feature gate `ControlPlaneKubeletLocalMode` to toggle the feature until
  graduating to GA.

### Non-Goals

<!--
What is out of scope for this KEP? Listing non-goals helps to focus discussion
and make progress.
-->

- Support the "old way" and "new way" indefinitely. Once the proposed feature gate
  graduates to GA it will hardcoded to be active.
- Touch areas of kubeadm different than `kubeadm join`, `kubeadm init` and `kubeadm upgrade`.

## Proposal

<!--
This is where we get down to the specifics of what the proposal actually is.
This should have enough detail that reviewers can understand exactly what
you're proposing, but should not include things like API designs or
implementation. What is the desired outcome and how do we measure success?.
The "Design Details" section below is for the real
nitty-gritty.
-->

The proposal is to implement the required changes to make the kubelet point to the local available kube-apiserver.
This change relates to initializing, joining and upgrading control plane nodes and does not affect worker nodes.

The overall change is:

- for `kubeadm join` to adjust the file `/etc/kubernetes/bootstrap-kubelet.conf` to point to the local kube-apiserver, which gets created by kubeadm during the `KubeletStartJoinPhase` ([xref](https://github.com/kubernetes/kubernetes/blob/caf5311/cmd/kubeadm/app/cmd/phases/join/kubelet.go#L122-L125)). This will also affect the kubelet's kubeconfig.
- for `kubeadm init` to adjust the created kubeconfig to point to the local kube-apiserver, which gets created by kubeadm during the `kubeconfig` phase ([xref](https://github.com/kubernetes/kubernetes/blob/8871513c1b64cae321552abfe9a3a90969637560/cmd/kubeadm/app/cmd/phases/init/kubeconfig.go#L87))
- for `kubeadm upgrade` to edit the kubelet config file to point to the local kube-apiserver.

To make this work for `kubeadm join`, an additional change is required: etcd needs to get started and joined to the etcd cluster before waiting for the kubelet to finish its bootstrap process, instead of the other way around.
This requires reordering some of the operations done in different kubeadm phases by extracting the relevant parts into separate phases and changing their order.

Because reordering the phases can be considered a breaking change to the CLI of kubeadm for some users, this should get done behind a feature gate, while preserving the previous behavior when the feature gate is disabled.

### User Stories (Optional)

<!--
Detail the things that people will be able to do if this KEP is implemented.
Include as much detail as possible so that people can understand the "how" of
the system. The goal here is to make this feel real for users without getting
bogged down.
-->

#### Story 1

As a kubeadm user, I wish no component violates the Kubernetes version skew policy when joining a control plane node.

#### Story 2

As a kubeadm user, I wish the kubelet of a joining control plane node points to the local kube-apiserver.

#### Story 3

As a kubeadm user, I wish the CLI of kubeadm to be stable and breaking changes to it to be announced ahead of time.

#### Story 4

As a kubeadm user, I wish the kubelet of an initializing control plane node points to the local kube-apiserver.

### Notes/Constraints/Caveats (Optional)

<!--
What are the caveats to the proposal?
What are some important details that didn't come across above?
Go in to as much detail as necessary here.
This might be a good place to talk about core concepts and how they relate.
-->

- This change is only relevant for the control plane nodes because worker nodes do
  not have a local kube-apiserver to point to and are not affected by this special
  case of violating the version skew policy violation.
  Worker nodes will still have to follow the documented version skew policy.

#### Caveats

The change needs to get implemented in a way that there is no change for existing users,
when the feature gate is not enabled.

### Risks and Mitigations

<!--
What are the risks of this proposal, and how do we mitigate? Think broadly.
For example, consider both security and how this will impact the larger
Kubernetes ecosystem.

How will security be reviewed, and by whom?

How will UX be reviewed, and by whom?

Consider including folks who also work outside the SIG or subproject.
-->

#### Risk: Duration for joining a control-plane node may change

Joining a node is an asynchronous task which consists of multiple steps where kubeadm also relies on the kubelet to bootstrap its own kubeconfig or start static pods.
It is possible to hit timeouts when waiting for the kubelet to succeed.
This proposal also changes the order of operations done by the kubelet and may change the time required for single steps where kubeadm waits for the kubelet.
At the end the existing time boundaries required for joining a control plane node using kubeadm should not change.

#### Risk: control-plane node stability

Having the kubelet of a control-plane node pointing directly to the local kube-apiserver only may cause some stability issues.
E.g. when the local kube-apiserver is not functional, then the local kubelet won't be able to report its status back to the kube-apiserver and will become `NotReady`.

To mitigate this, a user could adjust the kubeconfig for the kubelet to point to the load balanced API Server after all components got upgraded.

As alternative, kubeadm's configuration in v1beta4 could get an option which makes kubeadm change the endpoint back to the load balanced endpoint after the kubelet bootstrapped itself.
However this could still lead to violations to the skew policy.

## Design Details

<!--
This section should contain enough information that the specifics of your
change are understandable. This may include API specs (though not always
required) or even code snippets. If there's any ambiguity about HOW your
proposal will be implemented, this is the place to discuss them.
-->

As explained above two minor changes are required to implement the required changes.

**1. Make the kubelet point to the local kube-apiserver**

For `kubeadm init` to make the kubelet point to the local apiserver, the kubeconfig
which get's written for the kubelet needs to get adjusted when the kubelet's kubeconfig
file gets written
([xref](https://github.com/kubernetes/kubernetes/blob/caf5311/cmd/kubeadm/app/cmd/phases/init/kubeconfig.go#L135-L163)).

For `kubeadm join` to make the kubelet point to the local apiserver, the file for
kubelet's bootstrap kubeconfig needs to get adjusted, which gets created by kubeadm during the
`KubeletStartJoinPhase`
([xref](https://github.com/kubernetes/kubernetes/blob/caf5311/cmd/kubeadm/app/cmd/phases/join/kubelet.go#L122-L125)).

This creates the following chicken-egg issue:

- After starting the kubelet, kubeadm waits in the `KubeletStartPhase` for it to bootstrap itself, before it would continue to start etcd in the `ControlPlaneJoinPhase`.
- Due to the above change, the kubelet will never be able to bootstrap itself while pointing at the local kube-apiserver, because the local kube-apiserver will never get ready unless etcd is available locally.

The second change address this issue.

**2. Introducing and reordering phases for kubeadm join**

To fix the above chicken-egg issue some operations of kubeadm need to get rearranged during `kubeadm join`.

To be more precise: parts of the `KubeletStartPhase` need to run after the `EtcdLocalSubPhase` (which is part of the `ControlPlaneJoinPhase`).

The current phases of `kubeadm join` are the following:

1. `PreflightPhase`
2. `ControlPlanePreparePhase`
3. `CheckEtcdPhase`
4. `KubeletStartPhase`
5. `ControlPlaneJoinPhase`

First the existing `KubeletStartPhase` gets split up and the code which waits
for the kubelet's bootstrap to complete into a separate phase named `KubeletWaitBootstrapPhase`
gets extracted.

1. `PreflightPhase`
2. `ControlPlanePreparePhase`
3. `CheckEtcdPhase`
4. `KubeletStartPhase`
5. **`KubeletWaitBootstrapPhase`**
6. `ControlPlaneJoinPhase`

Second the `EtcdLocalSubphase` which is the first part of the `ControlPlanJoinPhase` gets moved
to a new phase `ControlPlaneJoinEtcdPhase` and added between the `KubeletStartPhase` and
`KubeletWaitBootstrapPhase`.

1. `PreflightPhase`
2. `ControlPlanePreparePhase`
3. `CheckEtcdPhase`
4. `KubeletStartPhase`
5. **`ControlPlaneJoinEtcdPhase`**
6. `KubeletWaitBootstrapPhase`
7. `ControlPlaneJoinPhase`

Actions to preserve the old behavior when the feature gate is disabled:

- The new phases `ControlPlaneJoinEtcdPhase` and `KubeletWaitBootstrapPhase` should define a `RunIf` function to ensure
  they do not run when the feature gate is disabled.
- The phases `KubeletStartPhase` and `ControlPlaneJoinPhase` should behave as before if the feature gate is disabled.
- There should be no duplication of code, instead the `Run` functions of the new phases should
  get called from the old location if the feature gate is disabled.
- Nothing should be done during the new phases `ControlPlaneJoinEtcdPhase` and `KubeletWaitBootstrapPhase`.
- Add a description to the new phases which explain their functionality and explicitly mark them as `EXPERIMENTAL`.

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

None.

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

New unit tests must be added for all code paths that use the `ControlPlaneKubeletLocalMode` feature gate if applicable.

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

<!-- - <test>: <link to test coverage> -->

NONE

##### e2e tests

<!--
This question should be filled when targeting a release.
For Alpha, describe what tests will be added to ensure proper quality of the enhancement.

For Beta and GA, add links to added tests together with links to k8s-triage for those tests:
https://storage.googleapis.com/k8s-triage/index.html

We expect no non-infra related flakes in the last month as a GA graduation criteria.
-->

A new e2e test will be added in the kubernetes/kubeadm repository by using the kinder
tool. It can be maintained for one or more releases until the feature gate graduates
to beta and will be enabled by default (and because of that be tested already during
other test cases).
It can do the following:

- Create a 3 control plane node cluster
- Call `kubeadm init` on one of them, having the feature gate `ControlPlaneKubeletLocalMode`
  enabled.
- Check that the kubelet is pointing to the local apiserver.
- Call `kubeadm join` on the remaining control plane nodes.
- Check that the kubelet's are pointing to the local apiserver.
- Adjust the kubelet's kubeconfig's to point to the load balanced endpoint.
- Call `kubeadm upgrade` on the nodes.
- Check that all kubelet's are again pointing to the local apiserver.

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

- Feature implemented behind a feature gate `ControlPlaneKubeletLocalMode`.
- Initial unit and e2e tests completed and enabled.
- [Document the feature gate](https://kubernetes.io/docs/reference/setup-tools/kubeadm/kubeadm-init/#feature-gates).
- Document the upcoming breaking change at the release notes.

#### Beta

- Make feature gate to be enabled by default.
- Gather feedback from developers and surveys.
- Make unit and e2e test changes.
- Update the feature gate documentation.
- [Document the new phases](https://kubernetes.io/docs/reference/setup-tools/kubeadm/kubeadm-join-phase/).

#### GA

- Gather feedback from developers and surveys.
- Update unit tests.
- Remove e2e tests as this will be tested by all existing kubeadm e2e tests.
- Update the feature gate documentation.
- Update the phases documentation.

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

- N/A -> Alpha: users can patch their `ClusterConfiguration` in the `kube-system/kubeadm-config` ConfigMap to enable the `ControlPlaneKubeletLocalMode` feature gate, before calling `kubeadm upgrade apply`. This will allow them to join control plane nodes with this feature enabled. This scenario is anticipated as rare, because usually users maintain a stable control plane with 3 or more members before upgrading it. But it is still plausible and can be documented in the feature gate documentation. The `kubelet.conf` on existing nodes can also be edited to match the new behavior.
- Alpha -> Beta: similarly to the previous stage users can modify the `ClusterConfiguration` to disable the feature gate during upgrade. This will allow them to use the "old way", in case they wish to join more control plane nodes to the cluster while the feature gate is enabled by default.
- Beta -> GA: users could no longer patch the `ClusterConfiguration` to opt-out of the feature and it will be locked to be enabled by default.

### Version Skew Strategy

<!--
If applicable, how will the component handle version skew with other
components? What are the guarantees? Make sure this is in the test plan.

Consider the following in developing a version skew strategy for this
enhancement:
- Does this enhancement involve coordinating behavior in the control plane and nodes?
- How does an n-3 kubelet or kube-proxy without this feature available behave when this feature is used?
- How does an n-1 kube-controller-manager or kube-scheduler without this feature available behave when this feature is used?
- Will any other components on the node change? For example, changes to CSI,
  CRI or CNI may require updating that component before the kubelet.
-->

Not applicable.

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

- 01.02.2024: KEP issue created.
- 08.02.2024: KEP draft created.
- 07.05.2024: KEP marked as implementable for 1.31 or later.
- 12.07.2024: KEP adjusted to match discussed implementation.
- 15.07.2024: KEP alpha implementation merged.
- 06.02.2024: PRs to promote the feature gate to Beta.
- 17.09.2025: PRs to promote the feature gate to GA.

## Drawbacks

<!--
Why should this KEP _not_ be implemented?
-->

Moving or introducing new phases are breaking changes to the CLI of kubeadm.
Adding new phases may break users which are executing the single phases manually.
The changes must be well-documented in release notes for users to adapt the changes.

The reordering of phases may change delays or timeouts when control plane nodes are joined.
Modifying this code path may introduce potential for user complains about HA cluster creation and maintenance with kubeadm.
Sufficient testing and gathering feedback from users would be mandatory.

## Alternatives

<!--
What other approaches did you consider, and why did you rule them out? These do
not need to be as detailed as the proposal, but should include enough
information to express the idea and why it was not acceptable.
-->

### Take the risk of violating Kubernetes version skew policy

By not introducing the change, users of kubeadm are on risk to hit a case in the future where a violation of the Kubernetes version skew policy happens and joining a control plane node may fail.
We anticipate such cases to be rare.

### Use external etcd

When the external etcd mode is used nothing changes because the executed code would stay the same.
Just the skipped parts (for self-hosted etcd) will be skipped at other places.

## Infrastructure Needed (Optional)

<!--
Use this section if you need things from the project/SIG. Examples include a
new subproject, repos requested, or GitHub details. Listing these here allows a
SIG to get the process for these resources started right away.
-->

None.
