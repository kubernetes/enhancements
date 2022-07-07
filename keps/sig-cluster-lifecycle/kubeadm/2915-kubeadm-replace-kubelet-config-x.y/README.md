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
# KEP-2915: Replace usage of the kubelet-config-x.y naming in kubeadm

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
    - [First release N (when the deprecation happens)](#first-release-n-when-the-deprecation-happens)
    - [Release N+1](#release-n1)
    - [Release N+3 (two releases after Beta)](#release-n3-two-releases-after-beta)
    - [Release N+4](#release-n4)
  - [User Stories (Optional)](#user-stories-optional)
    - [Story 1](#story-1)
    - [Story 2](#story-2)
  - [Notes/Constraints/Caveats (Optional)](#notesconstraintscaveats-optional)
  - [Risks and Mitigations](#risks-and-mitigations)
    - [Risk 1](#risk-1)
    - [Risk 2](#risk-2)
    - [Risk 3](#risk-3)
- [Design Details](#design-details)
  - [Test Plan](#test-plan)
  - [Graduation Criteria](#graduation-criteria)
    - [Alpha](#alpha)
    - [Beta](#beta)
    - [GA](#ga)
    - [Post GA](#post-ga)
  - [Upgrade / Downgrade Strategy](#upgrade--downgrade-strategy)
    - [Steps per release:](#steps-per-release)
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

- [x] (R) Enhancement issue in release milestone, which links to KEP dir in [kubernetes/enhancements] (not the initial KEP PR)
- [ ] (R) KEP approvers have approved the KEP status as `implementable`
- [x] (R) Design details are appropriately documented
- [x] (R) Test plan is in place, giving consideration to SIG Architecture and SIG Testing input (including test refactors)
  - [ ] e2e Tests for all Beta API Operations (endpoints)
  - [ ] (R) Ensure GA e2e tests for meet requirements for [Conformance Tests](https://github.com/kubernetes/community/blob/master/contributors/devel/sig-architecture/conformance-tests.md)
  - [ ] (R) Minimum Two Week Window for GA e2e tests to prove flake free
- [x] (R) Graduation criteria is in place
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

During `kubeadm init`, a ConfigMap is written under `kube-system` called `kubelet-config-x.y`
where `x` is the Kubernetes MAJOR version and `y` the MINOR version. It includes a KubeletConfiguration
object Kind. The same ConfigMap is read during `kubeadm join` and downloaded to a file so that the joining
kubelet can use it via `--config`. The access to this ConfigMap for joining nodes, is made possible
via RBAC rules created during `kubeadm init` using a similar versioned naming format - Role
`kubeadm:kubelet-config-x.y` and RoleBinding `kubeadm:kubelet-config-x.y`, both under `kube-system`.

Simplify the naming of the default kubelet configuration ConfigMap and related RBAC rules that kubeadm manages
to the format `kubelet-config`, which does not include a version in the name. Given the stored configuration
object is versioned (has GroupVersionKind), it would be up to the running kubeadm instance that GETs
it to determine if this GroupVersionKind is supported.

## Motivation

<!--
This section is for explicitly listing the motivation, goals, and non-goals of
this KEP.  Describe why the change is important and the benefits to users. The
motivation section can optionally provide links to [experience reports] to
demonstrate the interest in a KEP within the wider Kubernetes community.

[experience reports]: https://github.com/golang/go/wiki/ExperienceReports
-->

Currently the overall design is faulty as `kubeadm init`, despite supporting a version skew with the kubelet
of N-1, writes a single ConfigMap `kubelet-config-x.y` where `x.y` is the version of the control plane
and not the version of the kubelet on the host where `kubeadm init` is called.

Later, during `join` operations the version `x.y` is determined by looking up the control plane version
of the cluster, via the `kubernetesVersion` field of the `ClusterConfiguration` object stored
in the `kube-system/kubeadm-config` ConfigMap.

During `upgrade` a new `kubelet-config-x.y+1` ConfigMap and RBAC rules are written where `y+1` is the target
MINOR upgrade version. The old `kubelet-config-x.y` are not cleaned up.

Encoding the version in the API object names is redundant and creates unnecessary complications for both
kubeadm maintainers and users that wish to modify the object in the cluster.

### Goals

<!--
List the specific goals of the KEP. What is it trying to achieve? How will we
know that this has succeeded?
-->

- Simplify the name of the kubelet related ConfigMap and RBAC rules that kubeadm manages to exclude
encoded version information in the API object names.
- Remove confusion around the version that is encoded in the name - control plane vs kubelet version?
- Simplify user access to the ConfigMap, allowing them to always rely on `kube-system/kubelet-config`
as the default location.
- Simplify the kubeadm logic to not manage the `x.y` version during PUT/GET operations.
- Gradually apply the change to allow users to adapt using a feature gate.

### Non-Goals

<!--
What is out of scope for this KEP? Listing non-goals helps to focus discussion
and make progress.
-->

- Apply the change within a single release of kubeadm, breaking the users.

## Proposal

<!--
This is where we get down to the specifics of what the proposal actually is.
This should have enough detail that reviewers can understand exactly what
you're proposing, but should not include things like API designs or
implementation. What is the desired outcome and how do we measure success?.
The "Design Details" section below is for the real
nitty-gritty.
-->

The proposal is to declare the default ConfigMap and RBAC object format of `kubelet-config-x.y`
deprecated but support it for a GA deprecation period of 1 year (3 kubeadm releases). In parallel,
introduce objects named `kubelet-config` (without `x.y`) that will become the source of truth.

The switch will be controlled by a new kubeadm specific feature gate (not a core one,
stored in kubeadm's ClusterConfiguration) called `UnversionedKubeletConfigMap`. Using a feature gate
for similar changes is common in changes in core Kubernetes components and is familiar mechanism
to users.

Note that depending on feedback from users the N+x release schedule might be adjusted - e.g.
extending or shortening the Alpha or Beta periods.

#### First release N (when the deprecation happens)

- The feature gate is added as Alpha and disabled by default.
- Users can continue using the old naming format or opt-in into the new naming format
by enabling the feature gate.

#### Release N+1

- The feature gate becomes Beta and is enabled by default.
- Users can opt-out of the new feature by disabling the feature gate.

#### Release N+3 (two releases after Beta)

- The feature gate becomes GA and is locked to enabled.
- Users can no longer opt-out and must adapt their infrastructure that reads/writes from/to the ConfigMap.
- Users should remove usage of the feature gate from their infrastructure. Alternatively they can do it
in the next release.

#### Release N+4

- The feature gate is removed.

### User Stories (Optional)

<!--
Detail the things that people will be able to do if this KEP is implemented.
Include as much detail as possible so that people can understand the "how" of
the system. The goal here is to make this feel real for users without getting
bogged down.
-->

#### Story 1

As a kubeadm user that wishes to modify the KubeletConfiguration that kubeadm stores in the cluster
before joining new nodes, I wish to have easy access to a static location such as `kube-system/kubelet-config`
without looking up the control plane version first.

#### Story 2

As a kubeadm maintainer, I wish to simplify the logic in kubeadm that manages the kubelet configuration
read/write operations.

### Notes/Constraints/Caveats (Optional)

<!--
What are the caveats to the proposal?
What are some important details that didn't come across above?
Go in to as much detail as necessary here.
This might be a good place to talk about core concepts and how they relate.
-->

NONE

### Risks and Mitigations

<!--
What are the risks of this proposal, and how do we mitigate? Think broadly.
For example, consider both security and how this will impact the larger
Kubernetes ecosystem.

How will security be reviewed, and by whom?

How will UX be reviewed, and by whom?

Consider including folks who also work outside the SIG or subproject.
-->

#### Risk 1

**Users do not have sufficient visibility of the change**

Communicate the change on the following channels to ensure as much visibility as possible:
- kubeadm release notes
- kubernetes-dev mailing list
- #kubeadm on k8s slack
- Blog posts, Twitter, Reddit, other

#### Risk 2

**Users that do not use `kubeadm upgrade` would not be able to join new nodes**

In the Alpha stage of the change, the command `kubeadm upgrade apply` will be used to create
a ConfigMap and the related RBAC rules that follow the new naming format. Nodes that are joined
to the cluster later would be able to use this new ConfigMap. During Beta the presence of these
new API objects will become a requirement, unless the users opt-out by disabling the feature gate
stored in the `ClusterConfiguration`.

Cluster API is an example of a project that uses kubeadm for cluster creation, but does not use the
`kubeadm upgrade` command for upgrades. These upgrades are called "immutable node upgrades", where new
version nodes join the cluster and replace the old ones. Similar projects and users would have to
implement their own means to support the Alpha->Beta transition of this feature by populating the required
ConfigMap and RBAC rules before joining new nodes during an immutable upgrade.

#### Risk 3

**The kubelet deprecates the API version stored in the ConfigMap**

Ideally, when the kubelet introduces a new API version (e.g. v1beta2) and deprecates the API currently
used by kubeadm (e.g. v1beta1), kubeadm has to have a way to upconvert the configuration in the ConfigMap,
so that users don't have to do it manually. Currently the kubelet does not expose a way to perform
this upconversion and the process is delegated as manual for the user to perform.

To continue supporting its version skew against the kubelet (N, N-1), kubeadm will:
- Delay the switch from the old to new kubelet API by one or more releases for new clusters
- For existing clusters during "join" and "upgrade", kubeadm will continue using the old API for
as long as possible.

This would give more time for the users to update the contents of the ConfigMap
and also to properly support the kubeadm / kubelet skew.

## Design Details

<!--
This section should contain enough information that the specifics of your
change are understandable. This may include API specs (though not always
required) or even code snippets. If there's any ambiguity about HOW your
proposal will be implemented, this is the place to discuss them.
-->

### Test Plan

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

The new feature gate will be tested in a specific e2e test jobs managed by the existent tooling
that kubeadm has established.

**Unit tests**

Unit tests will be included in the kubeadm source code to ensure that the feature gate correctly toggles
the used names of API objects.

**Upgrade e2e test job**

This upgrade job will ensure that `kubeadm upgrade apply` correctly creates API objects
with the new name format and that `kubeadm upgrade node` correctly consumes them.

This test job would not be required for the Beta->GA upgrade stage, since at that point
the feature gate will be enabled by default and the existing kubeadm tests will provide the same coverage.

**Join e2e test job**

This test job will be similar to the existing regular kubeadm e2e test jobs, except that
it would have the feature gate enabled. Its purpose would be to ensure that `kubeadm join` to an
existing cluster when the feature gate is enabled would GET the correct ConfigMap.

This test job would be viable only during the Alpha period, as during Beta the feature will be
enabled by default. Not having the feature enabled during Beta would be tested by existing e2e test jobs.

**Other kubeadm e2e tests**

Tests under `kubernetes/kubernetes/test/e2e_kubeadm` will be updated to correctly consume the correct
ConfigMap and RBAC depending on the feature gate state.

These tests would be relevant for the Alpha period and once the feature gate is enabled by default (Beta)
the checks for the legacy naming can be removed from them.

**Future e2e tests**

In the future, when the kubelet introduces a new API version, we should add kubeadm e2e tests that
ensure that kubeadm continues to operate using its preferred kubelet API version stored
in the `kubelet-config` ConfigMap.

### Graduation Criteria

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

- The feature is implemented behind a feature flag.
- Announce the feature to users.
- E2e tests are enabled.

#### Beta

- The feature is enabled by default, but users can opt out.
- Gather more feedback from users if needed.
- E2e tests are updated.
- Documentation at the k8s.io website is updated.

#### GA

- The feature is locked to enabled.
- Documentation at the k8s.io website is updated.
- E2e tests are updated.
- Gather more feedback from users if needed.

#### Post GA

- The feature gate is removed.

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

#### Steps per release:

- First release (N): during `kubeadm upgrade` to a kubeadm version N (feature gate as Alpha and disabled
by default) kubeadm must write the new ConfigMap and RBAC rules to allow the feature gate to work
if the user wishes to switch it to enabled.
- Second release (N+1): during `kubeadm upgrade` to a kubeadm version N+1 (feature gate is Beta and
enabled by default) the nodes can start using the new ConfigMap and RBAC rules by default, unless the
user has explicitly disabled the feature gate before upgrade.
- Third release (N+2): `kubeadm upgrade` and `join` will continue to honor the value of the feature gate.
- Fourth release (N+3): `kubeadm upgrade` and `join` will now only use the new format.
Cleanup of the old ConfigMaps and RBAC rules is out of scope but kubeadm can perform it.
- Fifth release (N+4): kubeadm no longer uses the feature gate and the feature is enabled by default.

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

Non-applicable, since the feature gate will be specific to the kubeadm binary and its operation
and components that kubeadm can be skewed against such as the kubelet, kube-proxy and control
plane will not be affected.

Note that kubeadm does not support conversion of the kubelet KubeletConfiguration Kind and this is left
as a manual operation to the user, unless the kubelet exposes conversion on an API endpoint or via
the CLI, eventually.

## Production Readiness Review Questionnaire

While the some of the questions in the PRR could be qualified as relevant to this proposal, KEPs
for kubeadm have been established as "out-of-tree" and the PRR does not apply to them.

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

- 2021-08-30: Initial KEP draft for 1.23 (Alpha).
- 2022-01-10: Update KEP for 1.24 (Beta).
- 2022-05-18: Update KEP for 1.25 (GA).

## Drawbacks

<!--
Why should this KEP _not_ be implemented?
-->

Deprecation of GA features is always disruptive and hard for the users to adapt to. The usage
of the old naming format is documented in a number of places in the kubeadm documentation
and a percentage of the users have already integrated it in scripts and tooling.

This KEP proposes a standard deprecation procedure for a GA feature in a Kubernetes component.
Users should already be familiar with similar procedures in other components, yet the likelihood
of this change causing disruptions is relatively high.

## Alternatives

<!--
What other approaches did you consider, and why did you rule them out? These do
not need to be as detailed as the proposal, but should include enough
information to express the idea and why it was not acceptable.
-->

Not using a feature gate has been considered, but the alternative of performing a hard switch
to the new format after a GA deprecation period is less desired. A feature gate gives control
to the users that are slower to adapt to opt-out during the Beta period and is a standard
practice in Kubernetes.

## Infrastructure Needed (Optional)

<!--
Use this section if you need things from the project/SIG. Examples include a
new subproject, repos requested, or GitHub details. Listing these here allows a
SIG to get the process for these resources started right away.
-->

NONE
