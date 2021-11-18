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
# KEP-3027: SLSA Compliance in the Kubernetes Release Process 

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
  - [User Stories](#user-stories)
    - [Artifact Verification](#artifact-verification)
    - [Provenance Metadata](#provenance-metadata)
    - [Release Completeness](#release-completeness)
  - [Risks and Mitigations](#risks-and-mitigations)
    - [Leaving Common Threats Unaddressed](#leaving-common-threats-unaddressed)
    - [Failure to Provide Certainty Downstream](#failure-to-provide-certainty-downstream)
- [Design Details](#design-details)
  - [Graduation Criteria](#graduation-criteria)
  - [Graduation Milestones](#graduation-milestones)
    - [SLSA Level 1: Documentation of the Build Process (not user impacting)](#slsa-level-1-documentation-of-the-build-process-not-user-impacting)
    - [SLSA Level 2: Tamper Resistance of the Build Service](#slsa-level-2-tamper-resistance-of-the-build-service)
    - [SLSA Level 3: Extra Resistance to Specific Threats](#slsa-level-3-extra-resistance-to-specific-threats)
    - [SLSA Level 4: Highest Levels of Confidence and Trust](#slsa-level-4-highest-levels-of-confidence-and-trust)
- [Implementation History](#implementation-history)
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
  - [ ] (R) Ensure GA e2e tests for meet requirements for [Conformance Tests](https://github.com/kubernetes/community/blob/master/contributors/devel/sig-architecture/conformance-tests.md) 
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

This document proposes a plan to harden the Kubernetes releases by
making the necessary adjustments to comply with the SLSA Framework.
[SLSA (Supply-chain Levels for Software Artifacts)](https://slsa.dev/)
is a framework to harden software supply currently being defined by the 
[OpenSSF](https://openssf.org/)'s 
[Supply Chain Integrity WG](https://github.com/ossf/wg-supply-chain-integrity).

The framework provides requirements and recommendations to software
build systems to harden their environments and the processes that drive
them. It also defines the metadata that needs to be produced to trace the 
origins of every item in a software release.

The main goal of this enhancement is to provide downstream consumers of our
artifacts the highest assurance about the integrity of each Kubernetes release.

SLSA defines several levels of hardening, each touching more aspects of the
release process that go beyond its technical implementation. This document is
meant to serve as a guide to reach the highest possible levels after 
consensus has been reached about their viability.

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

## Motivation

Release Kubernetes in a zero-trust environment.

Kubernetes releases represent key links in many software supply chains, not
just for the project itself but also for consumers that derive, repackage and
distribute our artifacts downstream. The project releases end-user artifacts
like binaries and container images, but also source code that is actively reused
further down the distribution stream.

All current work done by SIG Release on the Kubernetes supply chain centers
around three focus areas: Artifact Consumption, Introspection, and Security.
These areas are explained in more detail in our [_Roadmap for 2021 and Beyond_ 
document](https://github.com/kubernetes/sig-release/blob/master/roadmap.md).

As the world hardens its software distribution methods, the Kubernetes project needs
to do its part to achieve a secure supply chain from end to end: from the
top base images to the final artifacts downloaded by end users. This proposal
provides a path to achieve increasing levels of security, integrity, and availability
in our releases by engineering new features to our processes. The objective is
to achieve the highest possible compliance with the
[SLSA framework](https://slsa.dev/) (Supply-chain Levels for Software Artifacts).

We consider SLSA compliance to be an effort in line with the three objectives outlined
in our roadmap: Artifacts can be consumed easier and with more trust.
Improvements to code and process will secure the supply chain and each release
will produce software bills of materials, provenance attestations and signatures
which will yield much better introspection to the journey from code to binary.

The SLSA framework is a project under the [OpenSSF](https://openssf.org/)'s [Digital Identity 
Attestation Working Group](https://github.com/ossf/wg-digital-identity-attestation).
The framework defines numbered levels of compliance that harden software supply
chains by recommending concrete steps to address, each of increasing technical
complexity, number of stakeholders, and process adjuments necessary to reach them. 


<!--
This section is for explicitly listing the motivation, goals, and non-goals of
this KEP.  Describe why the change is important and the benefits to users. The
motivation section can optionally provide links to [experience reports] to
demonstrate the interest in a KEP within the wider Kubernetes community.

[experience reports]: https://github.com/golang/go/wiki/ExperienceReports
-->

### Goals

The main goal of this KEP is to provide a plan to achieve the highest possible
SLSA compliance in the Kubernetes release process. While providing a roadmap
this KEP can be considered complete when a SLSA level is deemed as not
implementable by the community at large (see Graduation Criteria).

### Non-Goals

This KEP does not aim to propose concrete technical implementation details
or process improvements to achieve each SLSA level. As a framework,
SLSA provides an incomplete roadmap to guide our implementation while it leaves
the rest to the project. Some of these changes need a discussion and KEP by
themselves, and we will be working on them as we advance.

The changes that need to be conducted include technical improvements (signing
of artifacts, metadata generation, etc), setting up cryptographic key management
systems and policy, VCS (GitHub) management, code review practices, and more.
None of those concrete enhancements are under the scope of this document.

## Proposal

We propose to work on SLSA compliance in a predictable manner. As we move up the
SLSA hardening levels, more and more stakeholders will be required to weigh in and
to provide perspective about the required enhancements. Starting with SLSA 1,
which is mostly a documentation effort, we plan to start having the discussions
and presenting the necessary KEPs to work our way up.

The ideal outcome of this KEP would be to have a Kubernetes release process 
that complies with SLSA Level 4, meaning it is shielded from the most common threats
to software supply chains. However, changes to reach level 4 may not be feasible. 

If the changes that need to be conducted are deemed too disruptive or even destructive
to other areas of the project (development velocity, contributor experience, policy,
etc), the community may declare a specific SLSA level to be unimplementable. In that
scenario, we would work on the rest of the SLSA requirements and consider this KEP
complete.

### User Stories

#### Artifact Verification

Through signed artifacts, downstream users will be able to check the integrity of
binaries, container images, documents, and other files that form part of a
Kubernetes release.

#### Provenance Metadata

Once the KEP is implemented, end-users will be able to verify the provenance of
the artifacts we release. This means that the origin of each artifact will be
traceable to its precise origin in the build process, with all parameters,
inputs, code points, and other metadata available in a verifiable and 
non-repudiable format.    

#### Release Completeness

Signed supply chain metadata files, like the software bill of materials and the
provenance attestations, will allow downstream consumers to ensure the integrity and
completeness of a Kubernetes release. This means users can be sure that all expected
artifacts are there, untampered, and also check that no extra items filtered into
the release buckets, registries, etc.

<!--
### Notes/Constraints/Caveats (Optional)

SLSA requirements involve changes to software that releases Kubernetes, to 
-->

<!--
What are the caveats to the proposal?
What are some important details that didn't come across above?
Go in to as much detail as necessary here.
This might be a good place to talk about core concepts and how they relate.
-->

### Risks and Mitigations

Failure to pursue this enhancement poses three problems for the project:


#### Leaving Common Threats Unaddressed

SLSA requirements call attention to the most vulnerable areas of Software 
Supply Chains. Failure to pursue this enhancement leaves some of those areas 
exposed.

#### Failure to Provide Certainty Downstream

As projects up and down the stream harden their supply chains, Kubernetes
cannot afford to be the weakest link. Our base images, the [Distroless Images are SLSA 
2 compliant now](https://security.googleblog.com/2021/09/distroless-builds-are-now-slsa-2.html),
for example. If we want to be good team players, our turn is up.


<!--
What are the risks of this proposal, and how do we mitigate? Think broadly.
For example, consider both security and how this will impact the larger
Kubernetes ecosystem.

How will security be reviewed, and by whom?

How will UX be reviewed, and by whom?

Consider including folks who also work outside the SIG or subproject.
-->

## Design Details

<!--
This section should contain enough information that the specifics of your
change are understandable. This may include API specs (though not always
required) or even code snippets. If there's any ambiguity about HOW your
proposal will be implemented, this is the place to discuss them.
-->
<!--

### Test Plan

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

### Graduation Criteria

This KEP can be considered complete when one of two scenarios is reached:

1. All SLSA levels have been successfully complied with 
1. The community determines one of the levels as not implementable. This
may be concluded if the nature of necessary changes proves to be too
disruptive or implies altering aspects of technical environments
and/or systems, contributor experience, policy, and other domains beyond what
deems to be acceptable.

Tracking issues will be opened to track and discuss the viability of the
required enhancements to reach each SLSA level.

### Graduation Milestones

The following is a rough, non-comprehensive outline of the work required
to achieve each of the SLSA levels. It is important to note that some
items and/or their specific implementations (like digital signing) 
will warrant a KEP of their own.

#### SLSA Level 1: Documentation of the Build Process (not user impacting)

SLSA level 1 calls for consumer availability of build and release process
information. The metadata provides a better overview of how software gets
built. Only provenance attestations need be published to reach compliance.

#### SLSA Level 2: Tamper Resistance of the Build Service

Level 2 calls for digital signatures of the metadata captured and passed
around the release process in the provenance attestations. 

Work to reach SLSA 2 will center around three key areas:

1. Laying the required groundwork to enable SIG Release access to
produce digital signatures. This enhancement is being proposed in 
[KEP-3031](https://github.com/kubernetes/enhancements/issues/3031).

2. Add the required improvements to sign the container images that are part
of a Kubernetes release.

3. Add the required improvements to allow the release process to
sign and verify artifacts as they travel through staging and release.

With those improvements in place, releases will produce signatures 
for their artifacts (images, binaries, tarballs, etc) as well as the
release metadata (provenance attestations, SBOMs, etc).

#### SLSA Level 3: Extra Resistance to Specific Threats

The enhancements needed to reach SLSA Level 3 involve modifying the release process
so that builds are controlled from configuration code checked into the VCS. Key
areas for SLSA level 3 include:

1. Modifying the release process to run from configuration files that determine
the build's outcome (relevant issue: 
[k/release#1836](https://github.com/kubernetes/release/issues/1836))

2. The build process needs to be modified to ensure the parameters for
running builds are accessible and recorded in the provenance statements.

#### SLSA Level 4: Highest Levels of Confidence and Trust

Reaching SLSA level 4 demands hardening and reviewing of access controls and
permissions to the build infrastructure, possibly a review of the PR approval
process, and making the build process hermetic. Some requirements:

1. Modifying the release process to make available all dependencies before 
the build starts (tracking issue:
[k/sig-release#1720](https://github.com/kubernetes/sig-release/issues/1720))

2. Ensuring that the release process meets a security standard. The framework
does not require a specific one. This topic shall be proposed and discussed in
a KEP when the time comes.

3. Ensuring that only an extremely limited number of individuals can override the
guarantees provided by SLSA. This is mostly true at the moment but more
transparency is needed to ensure risks and policies are understood by the
community.


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
<!--

### Upgrade / Downgrade Strategy

If applicable, how will the component be upgraded and downgraded? Make sure
this is in the test plan.

Consider the following in developing an upgrade/downgrade strategy for this
enhancement:
- What changes (in invocations, configurations, API use, etc.) is an existing
  cluster required to make on upgrade, in order to maintain previous behavior?
- What changes (in invocations, configurations, API use, etc.) is an existing
  cluster required to make on upgrade, in order to make use of the enhancement?
-->

<!--

### Version Skew Strategy

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

<!--

## Production Readiness Review Questionnaire


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

<!--

### Feature Enablement and Rollback

This section must be completed when targeting alpha to a release.
-->

<!--
###### How can this feature be enabled / disabled in a live cluster?

Pick one of these and delete the rest.


- [ ] Feature gate (also fill in values in `kep.yaml`)
  - Feature gate name:
  - Components depending on the feature gate:
- [ ] Other
  - Describe the mechanism:
  - Will enabling / disabling the feature require downtime of the control
    plane?
  - Will enabling / disabling the feature require downtime or reprovisioning
    of a node? (Do not assume `Dynamic Kubelet Config` feature is enabled).

###### Does enabling the feature change any default behavior?

<!--
Any change of default behavior may be surprising to users or break existing
automations, so be extremely careful here.

###### Can the feature be disabled once it has been enabled (i.e. can we roll back the enablement)?

<!--
Describe the consequences on existing workloads (e.g., if this is a runtime
feature, can it break the existing applications?).

NOTE: Also set `disable-supported` to `true` or `false` in `kep.yaml`.
-->
<!--
###### What happens if we reenable the feature if it was previously rolled back?

###### Are there any tests for feature enablement/disablement?

<!--
The e2e framework does not currently support enabling or disabling feature
gates. However, unit tests in each component dealing with managing data, created
with and without the feature, are necessary. At the very least, think about
conversion tests if API types are being modified.
-->


<!--

### Rollout, Upgrade and Rollback Planning
This section must be completed when targeting beta to a release.
-->


<!--
###### How can a rollout or rollback fail? Can it impact already running workloads?

Try to be as paranoid as possible - e.g., what if some components will restart
mid-rollout?

Be sure to consider highly-available clusters, where, for example,
feature flags will be enabled on some API servers and not others during the
rollout. Similarly, consider large clusters and how enablement/disablement
will rollout across nodes.
-->


<!--
###### What specific metrics should inform a rollback?

What signals should users be paying attention to when the feature is young
that might indicate a serious problem?
-->

<!--
###### Were upgrade and rollback tested? Was the upgrade->downgrade->upgrade path tested?

Describe manual testing that was done and the outcomes.
Longer term, we may want to require automated upgrade/rollback tests, but we
are missing a bunch of machinery and tooling and can't do that now.
-->

<!--
###### Is the rollout accompanied by any deprecations and/or removals of features, APIs, fields of API types, flags, etc.?

Even if applying deprecation policies, they may still surprise some users.
-->

<!--
### Monitoring Requirements

This section must be completed when targeting beta to a release.
-->

<!--
###### How can an operator determine if the feature is in use by workloads?

Ideally, this should be a metric. Operations against the Kubernetes API (e.g.,
checking if there are objects with field X set) may be a last resort. Avoid
logs or events for this purpose.
-->

<!--
###### How can someone using this feature know that it is working for their instance?

For instance, if this is a pod-related feature, it should be possible to determine if the feature is functioning properly
for each individual pod.
Pick one more of these and delete the rest.
Please describe all items visible to end users below with sufficient detail so that they can verify correct enablement
and operation of this feature.
Recall that end users cannot usually observe component logs or access metrics.


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
<!--

###### What are the SLIs (Service Level Indicators) an operator can use to determine the health of the service?

Pick one more of these and delete the rest.

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

<!--

### Dependencies

This section must be completed when targeting beta to a release.
-->

<!--

###### Does this feature depend on any specific services running in the cluster?

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
<!--

### Scalability

For alpha, this section is encouraged: reviewers should consider these questions
and attempt to answer them.

For beta, this section is required: reviewers must answer these questions.

For GA, this section is required: approvers should be able to confirm the
previous answers based on experience in the field.
-->
<!--

###### Will enabling / using this feature result in any new API calls?

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

<!--
###### Will enabling / using this feature result in introducing new API types?

Describe them, providing:
  - API type
  - Supported number of objects per cluster
  - Supported number of objects per namespace (for namespace-scoped objects)
-->

<!--
###### Will enabling / using this feature result in any new calls to the cloud provider?

Describe them, providing:
  - Which API(s):
  - Estimated increase:
-->

<!--
###### Will enabling / using this feature result in increasing size or count of the existing API objects?

Describe them, providing:
  - API type(s):
  - Estimated increase in size: (e.g., new annotation of size 32B)
  - Estimated amount of new objects: (e.g., new Object X for every existing Pod)
-->
<!--

###### Will enabling / using this feature result in increasing time taken by any operations covered by existing SLIs/SLOs?

Look at the [existing SLIs/SLOs].

Think about adding additional work or introducing new steps in between
(e.g. need to do X to start a container), etc. Please describe the details.

[existing SLIs/SLOs]: https://git.k8s.io/community/sig-scalability/slos/slos.md#kubernetes-slisslos
-->

<!--
###### Will enabling / using this feature result in non-negligible increase of resource usage (CPU, RAM, disk, IO, ...) in any components?

Things to keep in mind include: additional in-memory state, additional
non-trivial computations, excessive access to disks (including increased log
volume), significant amount of data sent and/or received over network, etc.
This through this both in small and large cases, again with respect to the
[supported limits].

[supported limits]: https://git.k8s.io/community//sig-scalability/configs-and-limits/thresholds.md
-->
<!--

### Troubleshooting

This section must be completed when targeting beta to a release.

The Troubleshooting section currently serves the `Playbook` role. We may consider
splitting it into a dedicated `Playbook` document (potentially with some monitoring
details). For now, we leave it here.
-->
<!--

###### How does this feature react if the API server and/or etcd is unavailable?

###### What are other known failure modes?

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


###### What steps should be taken if SLOs are not being met to determine the problem?
-->

## Implementation History

- 2021-10-31 Initial Draft
- 2021-11-17 Broader descriptions of required work for each SLSA level

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


<!--

## Drawbacks

Why should this KEP _not_ be implemented?
-->

<!--

## Alternatives

What other approaches did you consider, and why did you rule them out? These do
not need to be as detailed as the proposal, but should include enough
information to express the idea and why it was not acceptable.
-->
<!--

## Infrastructure Needed (Optional)



Use this section if you need things from the project/SIG. Examples include a
new subproject, repos requested, or GitHub details. Listing these here allows a
SIG to get the process for these resources started right away.
-->
