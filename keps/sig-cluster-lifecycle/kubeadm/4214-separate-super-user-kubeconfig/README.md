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
# KEP-4214: Separate super-user kubeconfig for kubeadm

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
    - [Story: Resolving compromised admin credentials](#story-resolving-compromised-admin-credentials)
    - [Story: Keeping the super-user credential in a safe place](#story-keeping-the-super-user-credential-in-a-safe-place)
  - [Notes/Constraints/Caveats (Optional)](#notesconstraintscaveats-optional)
    - [Caveats](#caveats)
  - [Risks and Mitigations](#risks-and-mitigations)
    - [Risk: Implementation complexity during in-place upgrade](#risk-implementation-complexity-during-in-place-upgrade)
    - [Risk: Implementation complexity during re-place upgrade](#risk-implementation-complexity-during-re-place-upgrade)
- [Design Details](#design-details)
  - [Test Plan](#test-plan)
      - [Prerequisite testing updates](#prerequisite-testing-updates)
      - [Unit tests](#unit-tests)
      - [Integration tests](#integration-tests)
      - [e2e tests](#e2e-tests)
  - [Graduation Criteria](#graduation-criteria)
  - [Upgrade / Downgrade Strategy](#upgrade--downgrade-strategy)
  - [Version Skew Strategy](#version-skew-strategy)
- [Production Readiness Review Questionnaire](#production-readiness-review-questionnaire)
- [Implementation History](#implementation-history)
- [Drawbacks](#drawbacks)
- [Alternatives](#alternatives)
  - [Using a feature gate](#using-a-feature-gate)
  - [Integration test vs e2e test](#integration-test-vs-e2e-test)
  - [Signing individual kubeconfig files for control plane nodes](#signing-individual-kubeconfig-files-for-control-plane-nodes)
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
  - [ ] (R) Ensure GA e2e tests meet requirements for [Conformance Tests](https://github.com/kubernetes/community/blob/master/contributors/devel/sig-architecture/conformance-tests.md)
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

During the initial control plane node creation (`kubeadm init`) an `admin.conf` file is generated.
This file currently contains a
[`cluster-admin`](https://kubernetes.io/docs/reference/access-authn-authz/rbac/#user-facing-roles)
credential that is bound to the `system:masters` group. Create two separate files instead -
`admin.conf` containing a regular Kubernetes cluster-admin credential and a `super-admin.conf`
containing a cluster-admin credential bound to the `system:masters` group.

## Motivation

<!--
This section is for explicitly listing the motivation, goals, and non-goals of
this KEP.  Describe why the change is important and the benefits to users. The
motivation section can optionally provide links to [experience reports] to
demonstrate the interest in a KEP within the wider Kubernetes community.

[experience reports]: https://github.com/golang/go/wiki/ExperienceReports
-->

Binding an admin credential to the`system:masters` group means it can
[bypass RBAC](https://github.com/kubernetes/kubeadm/issues/2414#issue-836108390) - i.e.
its permissions cannot be removed.
At the time of writing this KEP, Kubernetes does not support
[certificate revocation](https://github.com/kubernetes/kubernetes/issues/18982). This means
the only way to revoke access of an admin credential bound to the `system:masters` group is to
[rotate the certificate authority](https://kubernetes.io/docs/tasks/tls/manual-rotation-of-ca-certificates/)
of this cluster.

A general purpose `admin.conf` must be created that does not bind to `system:masters`.
This credential can be shared by kubeadm deployed control plane nodes. In case this `admin.conf`
credential is compromised, its permission must be revocable with RBAC.

A separate "break-glass", super-user credential can be managed in `super-admin.conf`.
This credential can be used to restore the cluster to a normal state in case of disruptive
admin user activity.

Both files must not be shared to new admin users and instead new admin credentials should
be signed with the `kubeadm kubeconfig user` command.

### Goals

<!--
List the specific goals of the KEP. What is it trying to achieve? How will we
know that this has succeeded?
-->

- Start managing a separate `super-admin.conf` that contains a super-user credential.
- Continue using a file named `admin.conf` for all its current kubeadm uses today.
- Improve the kubeadm documentation at the k8s.io website for creating
additional admin users.

### Non-Goals

<!--
What is out of scope for this KEP? Listing non-goals helps to focus discussion
and make progress.
-->

- Do not sign the credentials in the `admin.conf` and `super-admin.conf` files
to be with expiration time of more than 1 year.

## Proposal

<!--
This is where we get down to the specifics of what the proposal actually is.
This should have enough detail that reviewers can understand exactly what
you're proposing, but should not include things like API designs or
implementation. What is the desired outcome and how do we measure success?.
The "Design Details" section below is for the real
nitty-gritty.
-->

To preserve the existing behavior where `admin.conf` has full cluster access,
a new ClusterRoleBinding will be created called `kubeadm:cluster-admins`.
It will bind the ClusterRole `cluster-admin` to the `kubeadm:cluster-admins` Group.
The credential stored in `admin.conf` will have the following subject:
`O = kubeadm:cluster-admins, CN = kubernetes-admin`. In case of a compromised
credential, the ClusterRoleBinding `kubeadm:cluster-admins` can be removed or updated.
The Group `kubeadm:cluster-admins` is recommended for internal kubeadm use only.

The new file `super-admin.conf` will contain the following subject:
`O = system:masters, CN = kubernetes-super-admin`. It will act as the "break-glass",
super-user credential that can bypass RBAC. In case this credential is compromised
the cluster certificate authority must be rotated.

Both the `admin.conf` and `super-admin.conf` files will be renewable by `kubeadm upgrade`
and `kubeadm certs renew`. If the `super-admin.conf` is missing it will not cause an error.
That is in case the super-admin has manually moved the file to a safe location -
i.e. not keeping it on the primary control plane node, where `kubeadm init` was called.

The documentation in the
[Generating kubeconfig files for additional users](https://kubernetes.io/docs/tasks/administer-cluster/kubeadm/kubeadm-certs/#kubeconfig-additional-users)
section will be updated.
For each new User or a Group of users the recommendation will be to create a new
ClusterRoleBinding. Use the flags `--client-name` and `--org` of `kubeadm kubeconfig user`
to control what User or Group this credential belongs to. Revoke access of a single User
or a whole Group in case of credential compromise. Using the existing `system:masters` or
`kubeadm:cluster-admins` Groups will not be recommended.

### User Stories (Optional)

<!--
Detail the things that people will be able to do if this KEP is implemented.
Include as much detail as possible so that people can understand the "how" of
the system. The goal here is to make this feel real for users without getting
bogged down.
-->

#### Story: Resolving compromised admin credentials

As a super-user administrator I want to be able to use RBAC to remove access of an
administrator credential that has been compromised. By removing the `kubeadm:cluster-admins`
ClusterRoleBinding all administrators credentials (`admin.conf`) on control plane nodes
that are signed for the `kubeadm:cluster-admins` Group will stop working.
I can proceed to sign new `admin.conf` credentials to be used on all nodes
by using a new custom Group bound to the `cluster-admin` ClusterRole.

Alternatively, the certificate authority of this cluster can be rotated,
which will allow the `kubeadm:cluster-admins` ClusterRoleBinding to be restored
and used again for new `admin.conf` credentials signed using the new certificate authority
and for the Group `kubeadm:cluster-admins`.

#### Story: Keeping the super-user credential in a safe place

As a super-user administrator I want to keep a credential that has super powers
and can override RBAC outside of nodes managed by kubeadm. After `kubeadm init` has
finished, I can move the `super-admin.conf` file to a secure location and only use it in
case of an emergency.

### Notes/Constraints/Caveats (Optional)

<!--
What are the caveats to the proposal?
What are some important details that didn't come across above?
Go in to as much detail as necessary here.
This might be a good place to talk about core concepts and how they relate.
-->

#### Caveats

Removing the `kubeadm:cluster-admins` ClusterRoleBinding will drop access
to all users in the `kubeadm:cluster-admins` Group, rendering all `admin.conf`
files on control plane nodes as invalid. However, it will not cause immediate
downtime as the `admin.conf` on control plane nodes is only used when executing
kubeadm commands, such as `kubeadm upgrade`. In such conditions the `admin.conf`
should be populated with credentials for a safe, temporary User that is bound
to the `cluster-admin` ClusterRole.

The super-user admin can then proceed to rotate the cluster certificate authority
and eventually restore the `kubeadm:cluster-admins` ClusterRoleBinding.

### Risks and Mitigations

<!--
What are the risks of this proposal, and how do we mitigate? Think broadly.
For example, consider both security and how this will impact the larger
Kubernetes ecosystem.

How will security be reviewed, and by whom?

How will UX be reviewed, and by whom?

Consider including folks who also work outside the SIG or subproject.
-->

#### Risk: Implementation complexity during in-place upgrade

During `kubeadm upgrade apply` the new ClusterRoleBinding `kubeadm:cluster-admins`
must be added. The switch to two separate files must be performed on all control plane
nodes. This means that both `kubeadm upgrade apply` and `kubeadm upgrade node` must
handle the migration of the `admin.conf` properly. The creation of the new
`super-admin.conf` will be done only on the node where `kubeadm upgrade apply` is
called. On later upgrades, one release after this feature is added, the certificate
renewal logic of `kubeadm upgrade` must be aware that the `super-admin.conf` file could
be missing and should not be rotated.

The mitigation here is detailed unit tests and e2e tests that ensure that
the migration for in-place upgrades is handled properly.

#### Risk: Implementation complexity during re-place upgrade

Users or higher level tools that manage kubeadm re-place upgrades, by removing old
control plane nodes and adding new control plane nodes, without calling
`kubeadm upgrade apply/node` must handle this transition manually.
The ClusterRoleBinding `kubeadm:cluster-admins` must be created before
the upgrade has started. The `kubeadm join` process for control plane nodes
will create new `admin.conf` files with certificates that bind to the
`kubeadm:cluster-admins` Group.

Again, tests will be required to ensure that the `admin.conf` works
properly and the ClusterRoleBinding `kubeadm:cluster-admins` exists.

The `super-admin.conf` file will not exist at all in such clusters,
that were upgraded from older versions of kubeadm. The administrator can sign
a `super-admin.conf` manually by using the
`kubeadm kubeconfig user --client-name=kubernetes-super-admin --org=system:masters`
command and store it in a safe location.

For new clusters of this kind, the `super-admin.conf` will exist on the node
where `kubeadm init` was called. It can be left untouched or manually moved.

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
The goal is to ensure that we don't accept enhancements with inadequate testing.

All code is expected to have adequate tests (eventually with coverage
expectations). Please adhere to the [Kubernetes testing guidelines][testing-guidelines]
when drafting this test plan.

[testing-guidelines]: https://git.k8s.io/community/contributors/devel/sig-testing/testing.md
-->

[x] I/we understand the owners of the involved components may require updates to
existing tests to make this code solid enough prior to committing the changes necessary
to implement this enhancement.

kubeadm will include new unit tests to ensure the new separate admin files are
generated properly.

One additional e2e test will be added in the kubernetes/kubeadm repository
by using the kinder tool. It can be maintained for one or more releases until
more users upgrade to the first release where this feature is available.
It can do the following:
- Creates a 3 control plane node cluster that has the latest kubeadm installed.
- Calls `kubeadm init` on one of them.
- Verifies that kubeconfig files and RBAC are setup properly.
- Calls `kubeadm join` on the remaining control plane nodes.
- Verifies the kubeconfig files on the remaining control plane nodes.
- Deletes the `super-admin.conf` file from the first control plane node.
- Deletes the `kubeadm:cluster-admins` ClusterRoleBinding.
- Calls `kubeadm upgrade` using the same kubeadm version.
- Ensures that the RBAC and `super-admin.conf` are recreated.

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

<!-- - `<package>`: `<date>` - `<test coverage>` -->

At least the following kubeadm packages will require updates and new unit tests:
- `cmd/kubeadm/app/phases/kubeconfig`
- `cmd/kubeadm/app/phases/certs`
- `cmd/kubeadm/app/phases/upgrade`

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

<!-- - <test>: <link to test coverage> -->

A new e2e test will be added by using the kinder tool.

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

Once released, the feature will come in effect immediately during upgrade
to a particular version or when new cluster creation is done with the
kubeadm release when the feature was added. There are no plans for opt-out
or opt-in with a feature gate as this is considered a security improvement.
The feature will graduate immediately after release.

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

On upgrade, the regular kubeconfig and certificate renewal process will be
performed. The `admin.conf` file will be replaced with a file that has
the de-escalated privileges. If the `super-admin.conf` file is present on
a node during a future N+1 release, the file will be replaced with
updated credentials. If `super-admin.conf` is not present, no errors
will be returned.

The `kubeadm upgrade apply` command will manage the addition of
the new ClusterRoleBinding `kubeadm:cluster-admins`. One release
after the feature was enabled, this logic for the RBAC management
can be removed.

kubeadm does not support downgrades.

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

The target release of kubeadm will be able to upgrade from N-1 nodes that
do not have the feature enabled yet. During cluster creation with the
target kubeadm version the feature will become enabled immediately.

The `kubeadm upgrade apply` command will manage the addition of
the new ClusterRoleBinding `kubeadm:cluster-admins` RBAC. One release
after the feature was enabled, this logic for the RBAC management
can be removed.

## Production Readiness Review Questionnaire

Not applicable for kubeadm. The kubeadm project is considered "out-of-tree".

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

- 18.09.2023: KEP created (1.29).
- 10.10.2023: Address minor feedback. KEP marked as implementable.
- 10.19.2023: Adjust test plan and risk / mitigations.

## Drawbacks

<!--
Why should this KEP _not_ be implemented?
-->

An estimated drawback is the change in user expectations.
The users may expect that the `admin.conf` file will continue to have the
super powers provided by the `system:masters` Group. This feature will affect
this expectation. An action will be required by the same users to sign a new
explicit `super-admin.conf` file with the super powers.

## Alternatives

<!--
What other approaches did you consider, and why did you rule them out? These do
not need to be as detailed as the proposal, but should include enough
information to express the idea and why it was not acceptable.
-->

### Using a feature gate

A feature gate was considered where users can use it to opt-in into this
behavior. The feature could use the standard Alpha-Beta-GA graduation cycle
and have the feature gate enabled for the Beta release.

An argument against this behavior is that the feature would not add disruption for
the average user. The `admin.conf` can continue to work as a cluster-admin
credential. The feature gate would only add complexity and will have no
well established benefits, other than granular feature enablement control.

Another argument is that in practice this is a security improvement
and preferably users should not be able to opt-out of similar features.

### Integration test vs e2e test

The kinder e2e testing tool is quite flexible and the nodes do include tools
such as `openssl` for certificate inspection and `base64` for decoding base64
strings. However, writing such an e2e test must be done in bash
or hardcoded in kinder as Go code.

Instead the option to use an Go integration test included in `cmd/kubeadm/test`
seems preferable. It will allow using the Go standard library and existing
kubeadm utils for parsing kubeconfig files and x509 certificates.

One downside is that the same integration test will be executed on every
change in the kubeadm tree under `kubernetes/kubernetes` instead
of being less frequent - i.e. periodic.

### Signing individual kubeconfig files for control plane nodes

Today, control plane nodes use the same `admin.conf` that is shared
via a `kube-system/kubeadm-certs` Secret and encrypted with a RSA key.
This behavior is expected from kubeadm since it treats control plane
nodes as setup replicas (more or less).

During joining of control plane nodes, this download of the `admin.conf`
can be skipped. Instead the `ca.key` and `ca.cert` pair (already shared)
can be used to create a new `admin.conf` unique for this control-plane node.

For example:
- Control plane node `foo` wishes to join the cluster.
- The `ca.key` and `ca.cert` are downloaded from the Secret.
- The node name is ensured, likely by expecting the kubelet
client certificate.
- A new `admin.conf` is created that has a subject:
`O = kubeadm:cluster-admins, CN = kubernetes-admin-foo`.
- This User `kubernetes-admin-foo` is bound to the `cluster-admin`
ClusterRole with an additional RBAC rule.

Similarly, during `kubeadm init` the `admin.conf` must contain
the CN with the node name. `kubeadm upgrade` for this node
must ensure to properly maintain the new RBAC rule and `CN`
in the `admin.conf`.

This would allow to revoke access of individual control plane
nodes' `admin.conf` users. Since it adds complexity to the current KEP,
it could be done in a separate KEP as additional hardening.

However, it also opens some questions about node security. With disk structure
in mind, if the `admin.conf` of a control plane node has leaked,
that may also mean the `ca.key` has leaked which means the entire cluster
is compromised.

## Infrastructure Needed (Optional)

<!--
Use this section if you need things from the project/SIG. Examples include a
new subproject, repos requested, or GitHub details. Listing these here allows a
SIG to get the process for these resources started right away.
-->

None.
