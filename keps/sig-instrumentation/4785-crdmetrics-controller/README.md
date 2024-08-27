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
# KEP-4785: CRDMetrics Controller

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

Items marked with (R) are required *prior to targeting to a milestone / release*.

- [ ] (R) Enhancement issue in release milestone, which links to KEP dir in [kubernetes/enhancements] (not the initial KEP PR)
- [ ] (R) KEP approvers have approved the KEP status as `implementable`
- [ ] (R) Design details are appropriately documented
- [ ] (R) Test plan is in place, giving consideration to SIG Architecture 
  and SIG Testing input (including test refactors)
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

**Note:** This checklist is iterative and should be reviewed and updated every time this enhancement is being considered for a milestone.
-->

N/A since the KEP proposes a controller external to the core Kubernetes codebase.

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

Custom Resource Definition Metrics (`crdmetrics`) is a Kubernetes controller 
that builds on Kube-State-Metrics' Custom Resource State's ideology and
generates metrics for custom resources based on the configuration specified in
its managed resource, `CRDMetricsResource`.

## Motivation

<!--
This section is for explicitly listing the motivation, goals, and non-goals of
this KEP.  Describe why the change is important and the benefits to users. The
motivation section can optionally provide links to [experience reports] to
demonstrate the interest in a KEP within the wider Kubernetes community.

[experience reports]: https://github.com/golang/go/wiki/ExperienceReports
-->

[kubernetes/kube-state-metrics#1710] introduced the Custom Resource State API
to Kube State Metrics, which allowed for generating metrics from Custom
Resources' schemata. This was a highly appreciated and much needed feature-set,
since before this, the only way to generate metrics for a particular resource
was to get that logic merged upstream. It has been two years since the patch
was merged, and yet even today the feature-set is arguably one of the most
contributed-to parts of one of the most active repositories that fall under SIG 
Instrumentation.

However, during the recent releases, the maintainers realized that owing to the
almost zero dependency parsing logic, it became more prone to side effects as
the configuration scaled to include more fields to cater to the needs raised by
the community. After many cycles of trading off maintenance complexity for more
features, and seeing its maturity, the maintainers agreed on putting this on a 
"maintenance-only" mode in favor of a better solution that prioritizes 
scalability in the long run, while ensuring that it meets the same expectations
as its predecessor and more.

Such a "solution" should allow the maintainers to deprecate and drop the Custom
Resource State API from Kube State Metrics, and replace it by the CRDMetrics 
controller which, in addition to its own benefits, would allow Kube State 
Metrics to drop all Custom Resource State API-specific behaviors that can crash
Kube State Metrics, directly affecting the availability of native metrics 
defined in the codebase, and ensuring that native metrics do not experience
significant downtime due to an unavoidable error during the generation of an
entirely different set of metrics. Also, clusters requiring only native metrics
will not experience the additional Custom Resource State API overhead
needlessly when metrics stores' are being built.

The presence of native and custom resource metrics under the same hood also 
encouraged folks to infer implicit expectations that we do not officially 
support, which in turn have driven them to patch the two, sharing logic, when
both anticipate varying degrees of engagement and should not be (and are not
designed to be) interdependent. Kube State Metrics, owing to its original
purpose of native metrics generation, is meant to be comparatively much more
stable, and as such, the two feature-sets should be able to grow and release
independently in contrast to a part of the codebase that the other is not
dependent upon warranting a release.

[kubernetes/kube-state-metrics#1710]: https://github.com/kubernetes/kube-state-metrics/pull/1710

### Goals

<!--
List the specific goals of the KEP. What is it trying to achieve? How will we
know that this has succeeded?
-->

The KEP targets a controller which:
* allows for Custom Resource metrics generation based on their schemata,
* while providing cluster-scoped managed resources (`CRDMetricsResource`) that 
  allows defining the collection configuration for generating metrics 
  on-the-fly, and,
* while being able to accommodate for multiple configuration parsing techniques
  and expression non-turing languages,
* all while conforming to, and improving the existing Custom Resource State API 
  offered by Kube State Metrics without hindering the maintainability and the
  scalability of the controller.

### Non-Goals

<!--
What is out of scope for this KEP? Listing non-goals helps to focus discussion
and make progress.
-->

The KEP does **not** target a controller which:
* overlaps with Kube State Metrics' goals in any way except for the Custom 
  Resource State API, or,
* offers any stability guarantees for the metrics generated using its managed 
  resource(s).

## Proposal

<!--
This is where we get down to the specifics of what the proposal actually is.
This should have enough detail that reviewers can understand exactly what
you're proposing, but should not include things like API designs or
implementation. What is the desired outcome and how do we measure success?.
The "Design Details" section below is for the real
nitty-gritty.
-->

The proposal targets the incubation and incorporation of a controller capable
of essentially doing all that Kube State Metrics' Custom Resource State API
offers, in additional to various benefits of its own owing to the controller
lifecycle it is based upon.

The controller aims to replace the existing Kube State Metrics' Custom Resource 
State feature-set and be significantly much more maintainable and scalable than
its predecessor while enabling the community the freedom to extend the supported
set of DSLs to parse the configuration in a language that they are familiar
with, instead of forcing them up a steep learning-curve for a self-defined DSL
that they have to live with, as is the case with Kube State Metrics' Custom
Resource State feature-set.

### User Stories (Optional)

<!--
Detail the things that people will be able to do if this KEP is implemented.
Include as much detail as possible so that people can understand the "how" of
the system. The goal here is to make this feel real for users without getting
bogged down.
-->

#### Story 1

As a cluster admin, I want to be able to express my Kube State Metrics' Custom 
Resource State configurations in dedicated CRs, so I can benefit from everything
the controller lifecycle has to offer, for e.g., not having to mount volumes and 
redeploy their modifications, event emissions on metrics generation, 
multiple managed resources to isolate configuration logic, etc.

#### Story 2

As a cluster admin, I want to be able to express my Kube State Metrics' Custom
Resource State configurations in a non-turing language that is well-known
throughout the ecosystem, and enables me to get going without learning another
DSL (domain-specific language).

### Notes/Constraints/Caveats (Optional)

<!--
What are the caveats to the proposal?
What are some important details that didn't come across above?
Go in to as much detail as necessary here.
This might be a good place to talk about core concepts and how they relate.
-->

The controller abides by the following principals:
* Garbage in, garbage out: Invalid configurations will generate invalid metrics.
  The exception to this being that certain checks that ensure metric structure
  are still present (for e.g., `value` should be a `float64`).
* Library support: The module is **not** intended to be used as a library, and 
  as such, does not export any functions or types, with `pkg/` being an 
  exception (for e.g., managed resource types).
* Metrics stability: There are **no** metrics stability guarantees, as the
  metrics are dynamically generated.
* No middle-ware: The configuration is `unmarshal`led into a set of stores that
  the codebase directly operates on. Unlike Kube State Metrics, there is **no**
  controller-defined parsing middle-ware that processes the configuration before
  it is used, in order to cut down on unnecessary complexity as much as
  possible.

### Risks and Mitigations

<!--
What are the risks of this proposal, and how do we mitigate? Think broadly.
For example, consider both security and how this will impact the larger
Kubernetes ecosystem.

How will security be reviewed, and by whom?

How will UX be reviewed, and by whom?

Consider including folks who also work outside the SIG or subproject.
-->

N/A since the managed resource offered by the CRDMetrics controller provides the
ability to define metric configurations that are a super-set of the 
expressibility that Kube State Metrics' Custom Resource State configurations
have to offer.

## Design Details

<!--
This section should contain enough information that the specifics of your
change are understandable. This may include API specs (though not always
required) or even code snippets. If there's any ambiguity about HOW your
proposal will be implemented, this is the place to discuss them.
-->

The controller offers a number of improvements over Kube State Metrics' Custom
Resource State API, while maintaining a 3x faster round trip time for metric
generation.

- At its core, the controller relies on its managed resource,
`CRDMetricsResource` to fetch the metric generation configuration. Parts of the
configuration may be defined using different `resolver`s, such as `unstructured`
or `CEL`.
- Once fetched, the controller `unmarshal`s the configuration YAML directly into
`stores` which are a set of metric `families`, which in turn are a set of 
`metrics`.
- Metric `stores` are created based on its respective GVKR (a type that embeds 
`schema.GroupVersionKind`, `schema.GroupVersionResource` to avoid 
[plural ambiguities]), and reflectors for the specified resource are
initialized, and populate the stores on its update.
- `/metrics` pings on `CRDMETRICS_MAIN_PORT` trigger the server to write the
raw metrics, combined with its appropriate header(s), in the response. All
generated metrics are hardcoded to `gauge`s by design, as Prometheus lacks
support for some OpenMetrics-specified metrics' types, such as `Info` and
`StateSets`.

[plural ambiguities]: https://github.com/kubernetes-sigs/kubebuilder/issues/3402

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

TBD.

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

TBD.

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

N/A.

##### e2e tests

<!--
This question should be filled when targeting a release.
For Alpha, describe what tests will be added to ensure proper quality of the enhancement.

For Beta and GA, add links to added tests together with links to k8s-triage for those tests:
https://storage.googleapis.com/k8s-triage/index.html

We expect no non-infra related flakes in the last month as a GA graduation criteria.
-->

- `crdmetrics_test`: https://github.com/rexagod/crdmetrics/tree/main/tests

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

N/A.

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

The controller will follow the same `n-1` release strategy as [Kube State 
Metrics' compatibility matrix] follows.

[Kube State Metrics' compatibility matrix]: https://github.com/kubernetes/kube-state-metrics/tree/main?tab=readme-ov-file#compatibility-matrix

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

N/A.

###### How can this feature be enabled / disabled in a live cluster?

<!--
Pick one of these and delete the rest.

Documentation is available on [feature gate lifecycle] and expectations, as
well as the [existing list] of feature gates.

[feature gate lifecycle]: https://git.k8s.io/community/contributors/devel/sig-architecture/feature-gates.md
[existing list]: https://kubernetes.io/docs/reference/command-line-tools-reference/feature-gates/

- [ ] Feature gate (also fill in values in `kep.yaml`)
  - Feature gate name:
  - Components depending on the feature gate:
- [ ] Other
  - Describe the mechanism:
  - Will enabling / disabling the feature require downtime of the control
    plane?
  - Will enabling / disabling the feature require downtime or reprovisioning
    of a node?
-->

N/A.

###### Does enabling the feature change any default behavior?

<!--
Any change of default behavior may be surprising to users or break existing
automations, so be extremely careful here.
-->

The controller only has RBAC permissions over its managed resources
(`CRDMetricsResource` instances) **only** and does not attempt to modify or
break any existing in-cluster functionality.

###### Can the feature be disabled once it has been enabled (i.e. can we roll back the enablement)?

<!--
Describe the consequences on existing workloads (e.g., if this is a runtime
feature, can it break the existing applications?).

Feature gates are typically disabled by setting the flag to `false` and
restarting the component. No other changes should be necessary to disable the
feature.

NOTE: Also set `disable-supported` to `true` or `false` in `kep.yaml`.
-->

Removing the controller will not remove its managed resources. They will be
dropped if the corresponding `CRDMetricsResource` CRD is deleted. Similarly, 
removing all managed resources will not remove the controller.

The reason why it is so is because cluster-scoped resources cannot have owner
references linking back to namespace-scoped resources, in which case, the
garbage collection is a no-op. See [819a80] for more details.

[819a80]: https://github.com/rexagod/crdmetrics/commit/819a8001200a13c51cb82779c139a9081b4d613b

###### What happens if we reenable the feature if it was previously rolled back?

In the context of the controller, pre-existing managed resources in the cluster
will be picked back up after a (re-)deploy.

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

N/A.

### Rollout, Upgrade and Rollback Planning

<!--
This section must be completed when targeting beta to a release.
-->

Future API version for managed resources will follow the [hub-spoke] 
interconversion model.

[hub-spoke]: https://www.kubebuilder.io/multiversion-tutorial/conversion-concepts

###### How can a rollout or rollback fail? Can it impact already running workloads?

<!--
Try to be as paranoid as possible - e.g., what if some components will restart
mid-rollout?

Be sure to consider highly-available clusters, where, for example,
feature flags will be enabled on some API servers and not others during the
rollout. Similarly, consider large clusters and how enablement/disablement
will rollout across nodes.
-->

In the context of the controller, failed controller deployments do not make any
changes to the existing managed resources' metric configurations.

###### What specific metrics should inform a rollback?

<!--
What signals should users be paying attention to when the feature is young
that might indicate a serious problem?
-->

The controller has a telemetry server to diagnose and query statistics.

###### Were upgrade and rollback tested? Was the upgrade->downgrade->upgrade path tested?

<!--
Describe manual testing that was done and the outcomes.
Longer term, we may want to require automated upgrade/rollback tests, but we
are missing a bunch of machinery and tooling and can't do that now.
-->

Yes, controller deploy, scale down, and scale up works fine. Operations on
managed resources, if any, in the future, will be taken care of using the
aforementioned [hub-spoke] interconversion model.

###### Is the rollout accompanied by any deprecations and/or removals of features, APIs, fields of API types, flags, etc.?

<!--
Even if applying deprecation policies, they may still surprise some users.
-->

The controller aims to completely replace Kube State Metrics' Custom Resource
State feature-set, and as such, cause it to be deprecated once the KEP graduates
to stable.

### Monitoring Requirements

<!--
This section must be completed when targeting beta to a release.

For GA, this section is required: approvers should be able to confirm the
previous answers based on experience in the field.
-->

N/A.

###### How can an operator determine if the feature is in use by workloads?

<!--
Ideally, this should be a metric. Operations against the Kubernetes API (e.g.,
checking if there are objects with field X set) may be a last resort. Avoid
logs or events for this purpose.
-->

N/A.

###### How can someone using this feature know that it is working for their instance?

<!--
For instance, if this is a pod-related feature, it should be possible to determine if the feature is functioning properly
for each individual pod.
Pick one more of these and delete the rest.
Please describe all items visible to end users below with sufficient detail so that they can verify correct enablement
and operation of this feature.
Recall that end users cannot usually observe component logs or access metrics.
-->

- [x] Events: Events are emitted in `EMIT_NAMESPACE` (defaults to ``), for e.g., 
  `OwnerRefInvalidNamespace` in case of an owner reference being defined on
  `CRDMetricsResource` to its controller.
- [x] API .status: The status for a successfully processed `CRDMetricsResource`
looks as follows:
```yaml
  status:                                                                                                     
    conditions:                                                                  
    - lastTransitionTime: "2024-08-27T19:46:13Z"
      message: 'Resource configuration has been processed successfully: Event handler successfully processed event: addEvent'
      observedGeneration: 1                                                                               
      reason: EventHandlerSucceeded                                                                                               
      status: "True                                                                                                           
      type: Processed
 ```
- [x] Other: `http_request_duration_seconds` is a histogram metric exposed on 
  the telemetry port which is useful for observing the trends in requests for
  the generated metrics.

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

TBD.

###### What are the SLIs (Service Level Indicators) an operator can use to determine the health of the service?

<!--
Pick one more of these and delete the rest.
-->

- [x] Metrics: Telemetry metrics exposed on `CRDMETRICS_SELF_PORT`.
- [x] Other: `healthz`, `readyz`, and `livez` endpoints are exposed on
  `CRDMETRICS_SELF_PORT`, `CRDMETRICS_MAIN_PORT`, and `CRDMETRICS_SELF_PORT`,
  respectively.

###### Are there any missing metrics that would be useful to have to improve observability of this feature?

<!--
Describe the metrics themselves and the reasons why they weren't added (e.g., cost,
implementation difficulties, etc.).
-->

TBD.

### Dependencies

<!--
This section must be completed when targeting beta to a release.
-->

The controller __directly__ imports the following `k8s.io` packages:
* `k8s.io/api`
* `k8s.io/apimachinery`
* `k8s.io/client-go`
* `k8s.io/code-generator`
* `k8s.io/klog/v2`
* `k8s.io/utils`

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

The controller relies on core Kubernetes components.

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

Yes, the controller needs to make API calls in order to reconcile the managed
resources in the cluster. There are no resync polls done by the controller
(`ResyncPeriod: 0` for all reflectors). The reflectors will do a
LIST and/or WATCH on the associated resources' modification. The same applies
for managed resource(s), however, their case will be accompanied by additional
GET and UPDATE calls to ensure their `ObjectMeta` and `Status` are synced.

###### Will enabling / using this feature result in introducing new API types?

<!--
Describe them, providing:
  - API type
  - Supported number of objects per cluster
  - Supported number of objects per namespace (for namespace-scoped objects)
-->

Yes, the controller currently has one cluster-scoped managed resource of  
[`CRDMetricsResource` type]. There is currently no upper-limit on the number of
managed resource instances that could be defined.

[`CRDMetricsResource` type]: https://github.com/rexagod/crdmetrics/blob/main/pkg/apis/crdmetrics/v1alpha1/types.go

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

The controller will **not** create or modify any object on its own. Only when a
managed resource is deployed will it try to reconcile it.

###### Will enabling / using this feature result in increasing time taken by any operations covered by existing SLIs/SLOs?

<!--
Look at the [existing SLIs/SLOs].

Think about adding additional work or introducing new steps in between
(e.g. need to do X to start a container), etc. Please describe the details.

[existing SLIs/SLOs]: https://git.k8s.io/community/sig-scalability/slos/slos.md#kubernetes-slisslos
-->

N/A.

###### Will enabling / using this feature result in non-negligible increase of resource usage (CPU, RAM, disk, IO, ...) in any components?

<!--
Things to keep in mind include: additional in-memory state, additional
non-trivial computations, excessive access to disks (including increased log
volume), significant amount of data sent and/or received over network, etc.
This through this both in small and large cases, again with respect to the
[supported limits].

[supported limits]: https://git.k8s.io/community//sig-scalability/configs-and-limits/thresholds.md
-->

The telemetry metrics show no considerable jump in memory or CPU usage under any
supported operation.

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

The controller will log errors but keep reconciling for when `etcd` comes back
up. 

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

It is encouraged to __observe__ any symptoms using the telemetry metrics, while
__monitoring__ any failure using the available set of health probes.

###### What steps should be taken if SLOs are not being met to determine the problem?

The controller provides a `/debug` endpoint (on `CRDMETRICS_SELF_PORT`) which
exposes all available `pprof` data to help diagnose any issue with the binary.
Additionally, the telemetry metrics can provide more details about the runtime
consumptions of the binary. The health probes take into factor the health of the
respective set of components associated with that probe and will fail if any 
such component is not healthy (`?verbose` may be appended when querying the
health probes' endpoint to know exactly which component(s) are not healthy).

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

We considered refactoring the Kube State Metrics' Custom Resource State API, but
that has actually been done multiple times in the past which often amounts to
us ending up in the same position, owing to its limited scalability.

## Infrastructure Needed (Optional)

<!--
Use this section if you need things from the project/SIG. Examples include a
new subproject, repos requested, or GitHub details. Listing these here allows a
SIG to get the process for these resources started right away.
-->

We request a repository (`kubernetes/crdmetrics`) to migrate
`rexagod/crdmetrics` to.
