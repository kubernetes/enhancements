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
# KEP-2862: Fine-grained Kubelet API Authorization 

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
  - [Notes](#notes)
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

We propose a change to the Kubelet API authorization to provide more
fine-grained control so that logging and monitoring agents that interact with the
Kubelet directly can do so while adhering to the least privilege principle.
Currently, the Kubelet API authorization uses a coarse authorization scheme,
where actions like reading health status and the ability to exec into a pod
require the same RBAC permissions.

We also propose that we document the kubelt API endpoints that we are adding
fine-grained authorization for as these were previously undocumented.

## Motivation

<!--
This section is for explicitly listing the motivation, goals, and non-goals of
this KEP.  Describe why the change is important and the benefits to users. The
motivation section can optionally provide links to [experience reports] to
demonstrate the interest in a KEP within the wider Kubernetes community.

[experience reports]: https://github.com/golang/go/wiki/ExperienceReports
-->

Historically the `/healthz` and 
`/pods` endpoints were available on the unauthenticated kubelet read-only port(10255) 
but kubelet read-only port is disabled by default and enabling the port is
considered a security worst practice. This has caused a lot of monitoring and 
logging agents to switch to using the kubelet authenticated port (10250).
However, the Kubelet API on the authenticated(10250) port currently uses a very
coarse scheme for authorizing requests. For example, reading `/healthz` and 
calling `/exec/…` (i.e. execute arbitrary code) require the same `proxy` 
subresource. So, if an application needs to say `list` `pods` on the node, as 
many monitoring and logging agents do, they must be granted the `proxy` 
subresource in RBAC. Doing so grants many other powerful permissions to these 
agents which could be exploited by an attacker to escalate privilege.  

### Goals

<!--
List the specific goals of the KEP. What is it trying to achieve? How will we
know that this has succeeded?
-->
- Introduce new authorization subresources for `configz`, `/healthz` and `/pods/` kubelet endpoints,
allowing for more  granular authorization.
- Officially document the `configz`, `healthz` and `/pods/` endpoints.
- These changes should be backwards compatible.
- These changes should not break users on upgrade.

### Non-Goals

<!--
What is out of scope for this KEP? Listing non-goals helps to focus discussion
and make progress.
-->
- Create a new Kubelet API.
- Remove existing authoriztion subresources.
- Make the kubelet API node-restricted.

## Proposal

<!--
This is where we get down to the specifics of what the proposal actually is.
This should have enough detail that reviewers can understand exactly what
you're proposing, but should not include things like API designs or
implementation. What is the desired outcome and how do we measure success?.
The "Design Details" section below is for the real
nitty-gritty.
-->

Add new authorization subresources for `healthz` and `pods` endpoint while 
supporting the old coarse grained proxy authorization subresources as shown in 
the table below.

| Request path                                     | Existing Subresource | Proposed Subresource(s) |
| ----------------------------------------------- | -------------------- | ------------------------ |
| /configz                                        | proxy                | proxy, configz           |
| /healthz                                        | proxy                | proxy, healthz           |
| /healthz/log                                    | proxy                | proxy, healthz           |
| /healthz/ping                                   | proxy                | proxy, healthz           |
| /healthz/syncloop                               | proxy                | proxy, healthz           |
| /pods/                                          | proxy                | proxy, pods              |
| /runningpods/                                   | proxy                | proxy, pods              |

This way users who were previously using the `proxy` authorization subresource to grant access to the
`/healthz` kubelet endpoint can update their ClusterRole or Role by replacing the
`proxy` subresource with the `healthz` subresource.


### User Stories (Optional)

<!--
Detail the things that people will be able to do if this KEP is implemented.
Include as much detail as possible so that people can understand the "how" of
the system. The goal here is to make this feel real for users without getting
bogged down.
-->

#### Story 1

As a security conscious node monitoring agent owner I want to interact with the
kubelet API to list pods on the node without having to grant the agent the 
`proxy` authorization subresource and use least privilege to grant the exact 
permissions the agent needs like `pods` or `healthz`.

### Notes

If this change were to be implemented as proposed, the `proxy` authorization 
subresource  would still cover `/attach/`, `/exec/`, `/run/`, `/debug/`. 
The reader might be wondering why we didn't break these permissions out into 
their own authorization subresource?

The reason why we did not breakout all endpoints under `proxy` into their own
subresource is because we could not reason about a case where if we allowed
one of these permissions, having any of the other permissions would be considered
worse. If you have `/exec/` then having `/run/` or `/attach/` isn't making it 
worse for an attacker.

We picked `/configz`, `/healthz` and `/pods` endpoints as they are read-only.

### Risks and Mitigations

<!--
What are the risks of this proposal, and how do we mitigate? Think broadly.
For example, consider both security and how this will impact the larger
Kubernetes ecosystem.

How will security be reviewed, and by whom?

How will UX be reviewed, and by whom?

Consider including folks who also work outside the SIG or subproject.
-->

Since we are adding a second SubjectAccessReview request the latency for some
requests to the Kubelet API will increase. However, Kubelet has a 
SubjectAccessReview response cache, so subsequent requests should result in 
cache hits.

The SubjectAccessReview QPS will also increase but this will also be mitigated
by the cache.

## Design Details

<!--
This section should contain enough information that the specifics of your
change are understandable. This may include API specs (though not always
required) or even code snippets. If there's any ambiguity about HOW your
proposal will be implemented, this is the place to discuss them.
-->

To determine if the caller has the required permissions for a particular request
made to the Kubelet API, the Kubelet creates a SubjectAccessReview request to 
`kube-apiserver`. The SubjectAccessReviews are currently populated as follows:

```yaml
apiVersion: authorization.k8s.io/v1
kind: SubjectAccessReview
spec:
  user: user1
  uid: 64167384-10aa-4361-bcef-526ab51d1e4d
  groups:
  - groups1
  resourceAttributes:
    group: ""
    version: "v1"
    resource: "nodes"
    namespace: ""
    name: "node-1"
    subresource: "proxy"
    verb: "GET"
```

The following information is passed into the SubjectAccessReview request
- Requesting user (generally from a TokenReviewRequest or client certificate authentication)
- Request verb
- A resource request with:
  - APIGroup: ""
  - APIVersion: "v1"
  - Resource: "nodes"
  - Name: nodeName (the name of the node the request was against)
  - Subresource: which, before this KEP, can be one of: `proxy`, `log`, `metrics`, `spec`, `stats`

The subresource is determined by the path of the Kubelet API request. `kube-apiserver` upon
receiving this request will check if the user in the SubjectAccessReview is authorized
to access the relevant resource and subresource. For example, if a cluster is using RBAC
then this ClusterRole and ClusterRoleBinding might allow access:

```yaml
apiVersion: rbac.authorization.k8s.io/v1
kind: ClusterRole
metadata:
  name: nodes-proxy
rules:
- apiGroups: [""]
  resources: ["nodes/proxy"]
  verbs: ["get", "create"]
---
apiVersion: rbac.authorization.k8s.io/v1
kind: ClusterRoleBinding
metadata:
  name: nodes-proxy-global
subjects:
- apiGroup: rbac.authorization.k8s.io
  kind: User
  name: user1
roleRef:
  apiGroup: rbac.authorization.k8s.io
  kind: ClusterRole
  name: nodes-proxy
```

We will add a feature-gate called `KubeletFineGrainedAuthz` that will be defaulted to
`false` in `alpha`. When this feature-gate is set to `true` Kubelet will first send
a SubjectAccessReview specifically for the `configz` `healthz` or `pods` endpoint, based on the path of the
request made to the Kubelet.
If that request fails Kubelet will retry with the coarse-grained verb (`proxy`).

When `kube-apiserver` communicates with the Kubelet in a cluster with RBAC 
enabled, then its user is bound to the `system:kubelet-api-admin` ClusterRole. 
For a cluster with the `KubeletFineGrainedAuthz` feature gate enabled, the  
`system:kubelet-api-admin` ClusterRole could be changed to look like:-

```diff
  apiVersion: rbac.authorization.k8s.io/v1
  kind: ClusterRole
  metadata:
    annotations:
      rbac.authorization.kubernetes.io/autoupdate: "true"
    labels:
      kubernetes.io/bootstrapping: rbac-defaults
    name: system:kubelet-api-admin
  rules:
  - apiGroups:
    - ""
    resources:
    - nodes
    verbs:
    - get
    - list
    - watch
  - apiGroups:
    - ""
    resources:
    - nodes
    verbs:
    - proxy

  - apiGroups:
    - ""
    resources:
    - nodes/log
    - nodes/metrics
    - nodes/proxy
    - nodes/stats
+   - nodes/healthz
+   - nodes/pods
+   - nodes/configz
    verbs:
    - '*'
```

Note: Kubelet uses the DelegatingAuthorizerConfig which already implements a
cache for allowed and denied requests. We will rely on this cache to prevent
sending duplicate requests.

Note: We thought about making this behavior controlled via the KubeletConfiguration
by adding a field called mode to the authoriztion.webhook field of type [KubeletWebhookAuthorization](https://github.com/kubernetes/kubernetes/blob/c19d9edfdee7b4ff39041f0254c92ebf66af332f/pkg/kubelet/apis/config/types.go#L529C6-L529C33).
But we could not find a reasonable use case in which someone would want to not do 
fine-grained authorization. We also did not want to support more than one behavior
when webhook authorization is enabled for Kubelet API.

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

- The following unit tests will be added to `nodeAuthorizerAttributesGetter`
  - When feature gate is disabled for `/configz`, `/healthz` and `/pods/` only 1 
  attribute with `proxy` subresource should be returned
  - When feature gate is enabled for `/configz`, `/healthz` and `/pods/` 2 attributes should
  be returned and the first should use subresource `configz`, `healthz` or `pods` and the second should use 
  the `proxy` subresource.

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

We'll add the following integration tests:
- check that when the feature-gate is enabled a request by a user with `nodes/proxy` 
permission to kubelet `/configz` endpoint authorizes successfully
- check that when the feature-gate is enabled a request by a user with `nodes/configz` 
permission to kubelet `/configz` endpoint authorizes successfully
- check that when the feature-gate is enabled a request by a user with `nodes/proxy` 
permission to kubelet `/healthz` endpoint authorizes successfully
- check that when the feature-gate is enabled a request by a user with `nodes/healthz` 
permission to kubelet `/healthz` endpoint authorizes successfully
- check that when the feature-gate is enabled a request by a user with `nodes/proxy` 
permission to kubelet `/pods/` endpoint authorizes successfully
- check that when the feature-gate is enabled a request by a user with `nodes/pods` 
permission to kubelet `/pods/` endpoint authorizes successfully
- check that when the feature-gate is disabled a request by a user with `nodes/proxy`
permission to kubelet `/configz` endpoint authorizes successfully
- check that when the feature-gate is disabled a request by a user with `nodes/configz` 
permission to kubelet `/configz` endpoint authorizes unsuccessfully
- check that when the feature-gate is enabled a request by a user with `nodes/proxy`  
permission to kubelet `/healthz` endpoint authorizes successfully
- check that when the feature-gate is disabled a request by a user with `nodes/healthz` 
permission to kubelet `/healthz` endpoint authorizes unsuccessfully
- check that when the feature-gate is enabled a request by a user with `nodes/proxy` 
permission to kubelet `/pods/` endpoint authorizes successfully
- check that when the feature-gate is enabled a request by a user with `nodes/pods` 
permission to kubelet `/pods/` endpoint authorizes unsuccessfully

##### e2e tests

<!--
This question should be filled when targeting a release.
For Alpha, describe what tests will be added to ensure proper quality of the enhancement.

For Beta and GA, add links to added tests together with links to k8s-triage for those tests:
https://storage.googleapis.com/k8s-triage/index.html

We expect no non-infra related flakes in the last month as a GA graduation criteria.
-->

Unit tests and integration tests should sufficiently cover these changes without
having to introduce new or update existing e2e tests.

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
- Initial unit and integration tests completed and enabled

#### Beta
- Feature gate set to true by default
- e2e tests added

#### GA
- Examples of real-world usage

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
  - Feature gate name: `KubeletFineGrainedAuthz`
  - Components depending on the feature gate: 
    - `kubelet`
    - `kube-apiserver`
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

While there will be no change to use-facing behavior as we will still send a 
SubjectAccessReview with the `proxy` authorization subresource, we will be
sending an extra SubjectAccessReview request.

###### Can the feature be disabled once it has been enabled (i.e. can we roll back the enablement)?

<!--
Describe the consequences on existing workloads (e.g., if this is a runtime
feature, can it break the existing applications?).

Feature gates are typically disabled by setting the flag to `false` and
restarting the component. No other changes should be necessary to disable the
feature.

NOTE: Also set `disable-supported` to `true` or `false` in `kep.yaml`.
-->

Yes, but a workload that is only authorized to use the new authorization subresource
will lose access and will need to update its RBAC Role to use `nodes/proxy`.

Having the feature-gate enabled in kubelet and disabled in kube-apiserver or
vice versa will not impact kube-apiserver's ability to talk to the kubelet API.
This is because whether the feature-gate is enabled or disabled kube-apiserver
will always have `nodes/proxy` permissions in it's RBAC. So either the first or
the second SubjectAccessReview request will authorize kube-apiserver.
 
###### What happens if we reenable the feature if it was previously rolled back?

If the kubelet feature-gate is re-enabled then kubelet will again start sending 
2 SubjectAccessReview requests.

If the kube-apiserver feature-gate is re-enabled then the ClusterRole `system:kubelet-api-admin`
will be updated as described in the (Design Details section)[#design-details].

Readers might wonder if the order in which the feature-gate is disabled matters?
It does not because no matter what the state kube-apiserver will always have `nodes/proxy`
permissions in it's RBAC.

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

Yes.

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

We have designed a fallback mechanism that prevents from failed rollouts or rollbacks
from impacting an already running workloads ability to interact with the kubelet API.

Please see the [Design Details](#design-details) section for more information.

###### What specific metrics should inform a rollback?

<!--
What signals should users be paying attention to when the feature is young
that might indicate a serious problem?
-->

Increase in failed requests to kubelet API from workloads.

###### Were upgrade and rollback tested? Was the upgrade->downgrade->upgrade path tested?

<!--
Describe manual testing that was done and the outcomes.
Longer term, we may want to require automated upgrade/rollback tests, but we
are missing a bunch of machinery and tooling and can't do that now.
-->

We have tested the following upgrade scenarios manually:

|Scenario| Result |
| -------|--------|
| Upgrade both kubelet and kube-apiserver so that feature gate is enabled in both. | workloads and kube-apiserver are able to reach kubelet|
| Upgrade only kubelet to enable the feature-gate | workloads and kube-apiserver are able to reach kubelet |
| Updrade only kube-apiserver to enable the feature-gate | workloads and kube-apiserver are able to reach kubelet |

We have tested the following rollback scenarios manually:

|Scenario| Result |
| -------|--------|
| Rollback both kubelet and kube-apiserver so that feature gate is disabled in both. | workloads and kube-apiserver are able to reach kubelet|
| Rollback only kubelet to disable the feature-gate | workloads and kube-apiserver are able to reach kubelet |
| Rollback only kube-apiserver to disable the feature-gate | workloads and kube-apiserver are able to reach kubelet |

###### Is the rollout accompanied by any deprecations and/or removals of features, APIs, fields of API types, flags, etc.?

<!--
Even if applying deprecation policies, they may still surprise some users.
-->
No.

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

Users can check if this feature is enabled in kube-apiserver by running the
following command:

```sh
kubectl get --raw /metrics | grep kubernetes_feature_enabled | grep KubeletFineGrainedAuthz
```

Users can check if this feature is nabled in the kubelet by running the
following command in a pod that is running on the node:

If readonly port is enabled:
```sh
curl http://<node-ip>:10255/metrics | grep kubernetes_feature_enabled | grep KubeletFineGrainedAuthz
```

If readonly port is not enabled:
```sh
curl -k https://$MY_NODE_IP:10250/metrics | grep kubernetes_feature_enabled | grep KubeletFineGrainedAuthz 
```

NOTE: for port 10250 the pod will need to have the right RBAC bindings (if RBAC is enabled) to view the metrics.

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
- [x] Other (treat as last resort)
  - Details: By replacing `nodes/proxy` permission in RBAC with the fine-grained permissions required by the workload such as `nodes/metrics`, `nodes/pods` etc. and then confirming that the requests to kubelet succeed and don't encounter authorization errors.

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

Same SLOs as the kubelet API currently offers.

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

Same SLIs as the kubelet API currently offers.

###### Are there any missing metrics that would be useful to have to improve observability of this feature?

<!--
Describe the metrics themselves and the reasons why they weren't added (e.g., cost,
implementation difficulties, etc.).
-->

No.

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

This feature only comes into play if kubelet authotization mode is set to Webhook.

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

For some requests, the Kubelet will perform an additional SubjectAccessReview
for the `proxy` authorization subresource when the first request with the 
fine-grained authorization subresource wasn't authorized.

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

The count of SubjectAccessReviews by Kubelet may double if a SubjectAccessReview
request is not cached previously.

###### Will enabling / using this feature result in increasing time taken by any operations covered by existing SLIs/SLOs?

<!--
Look at the [existing SLIs/SLOs].

Think about adding additional work or introducing new steps in between
(e.g. need to do X to start a container), etc. Please describe the details.

[existing SLIs/SLOs]: https://git.k8s.io/community/sig-scalability/slos/slos.md#kubernetes-slisslos
-->
Kubelet API is not covered by existing SLIs/SLOs.

The count of SubjectAccessReviews by Kubelet may double if a SubjectAccessReview
request is not cached previously, which means that the time to authorize a 
request to Kubelet may also double. 

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

Not any different from how it would affect kubelet without this feature. If kube-apiserver 
is unavailable any SAR from kubelet will fail.

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

If requests to kubelet API start failing due to authorization issues users can
disabled the feature-gate.

Users can check the kubernetes Audit logs for SubjectAccessReview requests
created by `system:nodes:*` and check the reason they failed.

###### What steps should be taken if SLOs are not being met to determine the problem?

1. Check that the feature gate is enabled in kube-apiserver and kubelet.
2. Check that the workload has the right permissions. Requesets are expected to
fail if you are using fine-grained subresources but the feature gate is not enabled
in kubelet.
3. Check the audit logs for SubjectAccessReview requests created by `system:nodes:*`
and check the reason these requests failed.
4. Check kubelet logs.

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

2024-09-28: [KEP-2862](https://github.com/kubernetes/enhancements/pull/4760) merged as implementable and PRR approved for ALPHA.
2024-10-17: Alpha Code implementation [PR](https://github.com/kubernetes/kubernetes/pull/126347) merged.
2024-10-22: Alpha Documentation [PR](https://github.com/kubernetes/website/pull/48412) merged.

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

## Infrastructure Needed (Optional)

<!--
Use this section if you need things from the project/SIG. Examples include a
new subproject, repos requested, or GitHub details. Listing these here allows a
SIG to get the process for these resources started right away.
-->