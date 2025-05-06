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
- [x] **Merge early and iterate.**
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
# KEP-4633: Only allow anonymous auth for configured endpoints

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
- [Possible Future Improvements](#possible-future-improvements)
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

By default, requests to the `kube-apiserver` that are not rejected by
configured authentication methods are treated as anonymous requests, and
given a username of `system:anonymous` and a group of `system:unauthenticated`.

This behavior is can be toggled on or off by using the `--anonymous-auth`
boolean flag. By default `anonymous-auth` is set to `true`.

We propose that kubernetes should allow users to configure which endpoints can
be accessed anonymously and disable anonymous auth for all other endpoints.

## Motivation

<!--
This section is for explicitly listing the motivation, goals, and non-goals of
this KEP.  Describe why the change is important and the benefits to users. The
motivation section can optionally provide links to [experience reports] to
demonstrate the interest in a KEP within the wider Kubernetes community.

[experience reports]: https://github.com/golang/go/wiki/ExperienceReports
-->

Many kubernetes users still misconfigure their cluster by creating RoleBindings
and ClusterRoleBindings to powerful rules in their cluster. [This](https://kccncna2023.sched.com/event/1R2tp/rbacdoors-how-cryptominers-are-exploiting-rbac-misconfigs-greg-castle-vinayak-goyal-google)
KubeCon talk covers a real world example of a production kubernetes cluster 
where a  ClusterRoleBinding was created that bound the `cluster-admin` 
ClusterRole to the `system:anonymous` user, allowing for full cluster takeover.

One of the mitigations would be to disable anonymous authentication by setting
the `kube-apiserver` flag `--anonymous-auth` to `false`, but this is not
possible for many deployments that depend on unauthenticated requests (from
clients like load balancers or the kubelet) to check health endpoints of a
kubernetes cluster (`healthz`, `livez` and `readyz`). In order to allow
these health checks a cluster admin has to enable anonymous requests opening the
door for misconfigurations.

### Goals

<!--
List the specific goals of the KEP. What is it trying to achieve? How will we
know that this has succeeded?
-->

Add a way to disable anonymous authentication for all endpoints except a 
set of configured endpoints.

### Non-Goals

<!--
What is out of scope for this KEP? Listing non-goals helps to focus discussion
and make progress.
-->

* Disable anonymous authentication for all endpoints.
* Change kubernetes default behavior around anonymous authentication.

## Proposal

<!--
This is where we get down to the specifics of what the proposal actually is.
This should have enough detail that reviewers can understand exactly what
you're proposing, but should not include things like API designs or
implementation. What is the desired outcome and how do we measure success?.
The "Design Details" section below is for the real
nitty-gritty.
-->

We propose that `anonymous-auth` have 3 states:

1. Disabled: Authentication fails for any anonymous requests
2. Enabled: Authentication succeeds for anonymous requests
3. Enabled for certain endpoints: Authentication succeeds for anonymous requests
   only for the configured endpoints.


### User Stories (Optional)

<!--
Detail the things that people will be able to do if this KEP is implemented.
Include as much detail as possible so that people can understand the "how" of
the system. The goal here is to make this feel real for users without getting
bogged down.
-->

#### Story 1

As a security conscious cluster admin I want to disable anonymous authentication
but still allow unauthenticated access to cluster health endpoints so that I
don't have to reconfigure external services like load balancers. This means that
even if a `RoleBinding` or `ClusterRoleBinding` is added by a user that targets
`system:anonymous` or `system:unauthenticated` then access
to that resource/endpoint would not be possible.

#### Story 2

`kubeadm` requires anonymous access to the
`cluster-info` `ConfigMap` in the `kube-public` namespace during cluster
bootstrapping. As a security conscious `kubeadm` user I would like to configure
just `/api/v1/namespaces/kube-public/configmaps/cluster-info` to be accessed
anonymously.

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

N/A

## Design Details

<!--
This section should contain enough information that the specifics of your
change are understandable. This may include API specs (though not always
required) or even code snippets. If there's any ambiguity about HOW your
proposal will be implemented, this is the place to discuss them.
-->

We will update the `kube-apiserver` [AuthenticationConfiguration](https://github.com/kubernetes/kubernetes/blob/e5a98f837954dc9e686e64229a0ba07f6a7cb200/staging/src/k8s.io/apiserver/pkg/apis/apiserver/v1beta1/types.go#L136)
with a field that allows a user to configure anonymous authentication as shown
below.

```diff
type AuthenticationConfiguration struct {
     JWT []JWTAuthenticator `json:"jwt"`
+
+    // If present --anonymous-auth must not be set.
+    Anonymous *AnonymousConfig `json:"anonymous,omitempty"`
}
+
+type AnonymousConfig struct {
+    Enabled bool `json:"enabled"`
+
+    // If set, anonymous auth is only allowed for requests whose path exactly matches one of the entries.
+    // This can only be set when enabled is true.
+    Conditions []AnonymousAuthCondition `json:"conditions,omitempty"`
}

+type AnonymousAuthCondition struct {
+    // Path for which anonymous auth is allowed.
+    Path string  
+}
```

Using the structure described above a user will be able to do the following:

1. Disable Anonymous Auth.

   ```
   apiVersion: apiserver.config.k8s.io/v1alpha1
   kind: AuthenticationConfiguration
   anonymous:
     enabled: false
   ```

   Note: This is the same as setting `--anonymous-auth` flag to `false`.

2. Enable Anonymous Auth.

   ```
   apiVersion: apiserver.config.k8s.io/v1alpha1
   kind: AuthenticationConfiguration
   anonymous:
     enabled: true
   ```

   Note: This is the same as setting `--anonymous-auth` flag to `true`.

3. Allow Anonymous Auth for certain endpoints only.

   ```
   apiVersion: apiserver.config.k8s.io/v1alpha1
   kind: AuthenticationConfiguration
   anonymous:
     enabled: true
     conditions:
     - path: "/healthz"
     - path: "/readyz"
     - path: "/livez"
   ```

   Note: The path must be an exact case-sensitive match. We do not intend to
   support globbing of paths to keep the surface area here as small as possible.
   Globbing wasn't required for the use cases presented so far, and the intent of
   this feature is to constrain anonymous auth to a well-known set of endpoints.
  
   Note: We expect anyone using this feature to have a small number of (1-3) 
   entries that are very explicit about the paths anonymous auth can be used for
   and for users who want to allow anonymous access to a wider or more
   complicated set of endpoints should lean on authorization policy.

A user will either be able to set `--anonymous-auth` or set the `Anonymous`
field in the `AuthenticationConfiguration`. If nither `--anonymous-auth` nor the
`Anonymous` field in the `AuthenticationConfiguration` are set then the
kubernetes default behavior of anonymous auth being enabled will be observed.

We will gate the ability for a user to configure anonymous auth using the
`AuthenticationConfiguration` behind a feature gate called 
`AnonymousAuthConfigurableEndpoints`.

When a user configures `AuthenticationConfiguration.Anonymous` the following
behavior should be observed:

1. If `AuthenticationConfiguration.Anonymous` is non-nil and
`AnonymousAuthConfigurableEndpoints` is not set to `true` then 
`kube-apiserver` should fail to start with an appropriate error guiding the user
to enable the feature gate.

1. If `AuthenticationConfiguration.Anonymous` is non-nil and `--anonymous-auth` 
flag is set then `kube-apiserver` should fail to start with an appropriate error
guiding the user to either use `--anonymous-auth` or use
`AuthenticationConfiguration.Anonymous`.

1. If `AuthenticationConfiguration.Anonymous.Enabled` is `false` but
`AuthenticationConfiguration.Anonymous.Conditions` is not empty then
`kube-apiserver` should fail to start with an appropriate error guiding the user
to set `AuthenticationConfiguration.Anonymous.Enabled` to `true`.

1. If `AuthenticationConfiguration.Anonymous.Enabled` is `true` but
`AuthenticationConfiguration.Anonymous.Conditions` is empty then
anonymous requests should be able to authenticate for any path.

1. If `AuthenticationConfiguration.Anonymous.Enabled` is `true` and
`AuthenticationConfiguration.Anonymous.Conditions` is not empty then
anonymous requests should be able to authenticate only for the paths specified
in `AuthenticationConfiguration.Anonymous.Conditions`.

Note: Today the authentication config file is dynamically reloaded when the
`jwt` field is updated. However, for the proposed `anonymous` field we plan not
to support dynamic reloading. This behavior is consistent with built-in
authorizers (like Node, RBAC etc.) that are also not reloaded during the dynamic
reloading of the authorization config file. To make this clear to users we will
update the relevant documentation for authentication config.

### Test Plan

<!--
**Note:** *Not required until targeted at a release.*
The goal is to ensure that we don't accept enhancements with inadequate testing.

All code is expected to have adequate tests (eventually with coverage
expectations). Please adhere to the [Kubernetes testing guidelines][testing-guidelines]
when drafting this test plan.

[testing-guidelines]: https://git.k8s.io/community/contributors/devel/sig-testing/testing.md
-->

[X] I/we understand the owners of the involved components may require updates to
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

We will add unit tests for the following scenarios:
1. Validation of the authentication configuration.
2. Making sure that the flag and the config are mutually exclusive.
3. Behavior of the path restricted anonymous authenticator.

Unit tests were added to the following:

* pkg/kubeapiserver/options/authentication_test.go
* staging/src/k8s.io/apiserver/pkg/authentication/request/anonymous/anonymous_test.go 

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
We will add an integration tests that exercise each of the following config file
based authentication scenarios:
1. anonymous auth disabled in the auth-config file.
1. anonymous auth enabled and unrestricted in the auth-config file.
1. anonymous auth enabled and restricted to certain paths in the auth-config
   file.

The following integration tests were added:

test/integration/apiserver/anonymous/anonymous_test.go

##### e2e tests

<!--
This question should be filled when targeting a release.
For Alpha, describe what tests will be added to ensure proper quality of the enhancement.

For Beta and GA, add links to added tests together with links to k8s-triage for those tests:
https://storage.googleapis.com/k8s-triage/index.html

We expect no non-infra related flakes in the last month as a GA graduation criteria.
-->

<!-- - <test>: <link to test coverage> -->
We believe that all scenarios will be sufficiently covered by the unit and 
integration tests so we will not need any additional e2e tests.

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

- Feature implemented behind a feature flag
- Full unit and integration test coverage

#### Beta

- Feature gate set to true by default

#### GA

- Examples of real-world usage
  - GKE and AWS are using this feature to limit anonymous access to /healthz,
  /readyz and /livez endpoints.

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

When the feature-gate is enabled none of the defaults or current settings
regarding anonymous auth are changed. The feature-gate enables the ability for
users to set the `anonymous` field using the `AuthenticationConfiguration` file.

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

This feature only impacts kube-apiserver and does not introduce any changes that
would be impacted by version skews. All changes are local to kube-apiserver and
are controlled by the `AuthenticationConfiguration` file passed to
kube-apiserver as a parameter.

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

- [X] Feature gate (also fill in values in `kep.yaml`)
  - Feature gate name: `AnonymousAuthConfigurableEndpoints`
  - Components depending on the feature gate: `kube-apiserver`
- [ ] Other
  - Describe the mechanism:
  - Will enabling / disabling the feature require downtime of the control
    plane?
  - Will enabling / disabling the feature require downtime or reprovisioning
    of a node?

###### Does enabling the feature change any default behavior?

<!--
Any change of default behavior may be surprising to users or break existing
automations, so be extremely careful here.
-->
Enabling the feature gate does not change the default behavior unless the user
also changes the value of `--anonymous-auth` flag or updates the
`AuthenticationConfiguration`.

###### Can the feature be disabled once it has been enabled (i.e. can we roll back the enablement)?

<!--
Describe the consequences on existing workloads (e.g., if this is a runtime
feature, can it break the existing applications?).

Feature gates are typically disabled by setting the flag to `false` and
restarting the component. No other changes should be necessary to disable the
feature.

NOTE: Also set `disable-supported` to `true` or `false` in `kep.yaml`.
-->
Yes.

###### What happens if we reenable the feature if it was previously rolled back?

Nothing, unless the user also changes the value of `--anonymous-auth` flag or
updates the `AuthenticationConfiguration`.

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

Yes we will add the following tests:

- When `AnonymousAuthConfigurableEndpoints` feature gate is disabled: 
  - If `AuthenticationConfiguration` contains the `Anonymous` stanza then 
  `kube-apiserver` should fail to start with an appropriate error guiding the user
  to enable the feature gate
  - Users should be able to set `--anonymous-auth` to `false`
  - Users should be able to set `--anonymous-auth` to `true`

- When `AnonymousAuthConfigurableEndpoints` feature gate is enabled:
  - If `AuthenticationConfiguration` contains the `Anonymous` stanza then 
  `--anonymous-auth` flag cannot be set
  - If `AuthenticationConfiguration` does not contain the `Anonymous` stanza
  then `--anonymous-auth` flag can be set to `true`
  - If `AuthenticationConfiguration` does not contain the `Anonymous` stanza
  then `--anonymous-auth` flag can be set to `false`

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

Enabling the feature flag alone does not change kube-apiserver defaults. However
if different API servers have different AuthenticationConfiguration for
Anonymous then some requests that would be denied by one API server could be
allowed by another. 

###### What specific metrics should inform a rollback?

<!--
What signals should users be paying attention to when the feature is young
that might indicate a serious problem?
-->

kube-apiserver fails to start when AuthenticationConfiguration file has 
`anonymous` field set.

If audit logs indicate that endpoints other than the ones configured in the
AuthenticationConfiguration file using the `anonymous.conditions` field are 
reachable by anonymous users.

###### Were upgrade and rollback tested? Was the upgrade->downgrade->upgrade path tested?

<!--
Describe manual testing that was done and the outcomes.
Longer term, we may want to require automated upgrade/rollback tests, but we
are missing a bunch of machinery and tooling and can't do that now.
-->

N/A

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

N/A

###### How can someone using this feature know that it is working for their instance?

<!--
For instance, if this is a pod-related feature, it should be possible to determine if the feature is functioning properly
for each individual pod.
Pick one more of these and delete the rest.
Please describe all items visible to end users below with sufficient detail so that they can verify correct enablement
and operation of this feature.
Recall that end users cannot usually observe component logs or access metrics.
-->

If a user sets AuthenticationConfig file and sets the `anonymous.enabled` to
`true` and sets `anonymous.conditions` to allow only certain endpoints. Then 
they can check if the feature is working by:

* making an anonymous request to an endpoint that is not in the list of 
endpoints they allowed. Such a request should fail with http status code 401.

* making an anoymous request to an endpoint that is in the list of endpoints
they allowed. Such a request should either succeed with http status code 200 (if
authz is configured to allow acees to that endpoint) or
fail with http statis code 403 (if authz is not configured to allow access to
that endpoint)

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

SLOs for actual requests should not change in any way compared to the flag-based
Anonymous configuration.

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

N/A

###### Are there any missing metrics that would be useful to have to improve observability of this feature?

<!--
Describe the metrics themselves and the reasons why they weren't added (e.g., cost,
implementation difficulties, etc.).
-->

N/A

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

No.

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

For GA, this section is required: approvers should be able to confirm the
previous answers based on experience in the field.

The Troubleshooting section currently serves the `Playbook` role. We may consider
splitting it into a dedicated `Playbook` document (potentially with some monitoring
details). For now, we leave it here.
-->

###### How does this feature react if the API server and/or etcd is unavailable?

This feature is about for API Server handles Authentication for anonymous
requests. If API server is unavailable then this feature is also unavailable.

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

After observing an issue (e.g. uptick in denied authentication requests),
kube-apiserver logs from the authenticator may be used to debug.

Additionally, manually attempting to exercise the affected codepaths would 
surface information that'd aid debugging. For example, attempting to issue an
anonymous request to an endpoint that is allowed or disallowed based on the
contraints set in the anonymous config in the AuthenticationConfiguration file.

## Implementation History

- [x] 2024-05-13 - KEP introduced
- [x] 2024-06-07 - KEP Accepted as implementable
- [x] 2024-06-27 - Alpha implementation merged https://github.com/kubernetes/kubernetes/pull/124917
- [x] 2024-07-15 - Integration tests merged https://github.com/kubernetes/kubernetes/pull/125967
- [x] 2024-08-13 - First release (1.31) when feature available
- [x] 2024-08-16 - Targeting beta in 1.32

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

- A sidecar proxy could have handled this but would push complexity into all
consumers who are not running side car proxies today, and the complexity of 
allowing the restriction in-tree is minimal.

- A deny authorizer could be added that does this but we think this approach is
better for the following reasons:
  - a deny authorizer is a lot more complex to implement than restricting the
  anonymous authenticator
  - having two decoupled subsystems where a later phase is responsible for 
  locking down over-granted requests from the first phase is not ideal. We 
  already have this with authz/admission, and we don't want to repeat that
  pattern if we don't have to.

## Infrastructure Needed (Optional)

<!--
Use this section if you need things from the project/SIG. Examples include a
new subproject, repos requested, or GitHub details. Listing these here allows a
SIG to get the process for these resources started right away.
-->

## Possible Future Improvements

We decided not to apply any restrictions here to anonymous `userInfo` that comes
back after all authenticators and impersonation have run because we think that
the scope of this KEP is to provide cluster admins with a way to restrict actual
anonymous requests. A request that was considered authenticated and as permitted
to impersonate `system:anonymous` is not actually anonymous.

If we want to allow cluster admins the ability to add  such restrictions we
think its better to give them the capability to configure webhook authenticators
and add `userValidationRules` capabilities. But doing so would expand the scope
of this KEP and it should likely be a separate effort.
