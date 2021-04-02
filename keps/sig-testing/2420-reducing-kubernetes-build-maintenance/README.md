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
# KEP-2420: Reducing Kubernetes Build Maintenance

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
  - [Risks and Mitigations](#risks-and-mitigations)
- [Design Details](#design-details)
  - [Test Plan](#test-plan)
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
-->

Items marked with (R) are required *prior to targeting to a milestone / release*.

- [x] (R) Enhancement issue in release milestone, which links to KEP dir in [kubernetes/enhancements] (not the initial KEP PR)
- [x] (R) KEP approvers have approved the KEP status as `implementable`
- [x] (R) Design details are appropriately documented
- [x] (R) Test plan is in place, giving consideration to SIG Architecture and SIG Testing input
- [x] (R) Graduation criteria is in place
- [x] (R) Production readiness review completed
- [x] (R) Production readiness review approved
- [x] "Implementation History" section is up-to-date for milestone
- [x] User-facing documentation has been created in [kubernetes/website], for publication to [kubernetes.io]
  - N/A, this is not user facing
- [x] Supporting documentation—e.g., additional design documents, links to mailing list discussions/SIG meetings, relevant PRs/issues, release notes

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

Kubernetes currently maintains multiple build systems, an ongoing burden and a source of contributor friction and confusion. Much has changed since Bazel was first introduced as an additional build system, upon re-evaluating the project it is clear that we should dedupe this. More details on why can be found in [Motivation](#motivation) and [Drawbacks](#drawbacks).

## Motivation

<!--
This section is for explicitly listing the motivation, goals, and non-goals of
this KEP.  Describe why the change is important and the benefits to users. The
motivation section can optionally provide links to [experience reports] to
demonstrate the interest in a KEP within the wider Kubernetes community.

[experience reports]: https://github.com/golang/go/wiki/ExperienceReports
-->

### Goals

<!--
List the specific goals of the KEP. What is it trying to achieve? How will we
know that this has succeeded?
-->
- Remove the toil of maintaining multiple build-systems from the Kubernetes repo maintainers
- Eliminate the friction of generating `BUILD` files from Kubernetes contributors using Go natively
- Simplify Golang upgrades for SIG Release
- Remove qualification gaps caused by duplicate-but-slightly-different binary builds
- Remove testing gaps caused by duplicate-but-slightly-different test-invocation methods
- Empower broader community maintenance of tests by converging on Golang testing standards

### Non-Goals

<!--
What is out of scope for this KEP? Listing non-goals helps to focus discussion
and make progress.
-->
- Support importing github.com/kubernetes/kubernetes as a library
  - This has never been supported, and is orthogonal to the decision of maintaining duplicate build systems
  - Code under kubernetes/kubernetes/src/staging/k8s.io/foo is intended to be imported as k8s.io/foo from the staged copies at github.com/kubernetes/foo; anything else is internal-only and not supported for import, bazel or otherwise.
- Removing Bazel from sub-projects other than the core repo
  - Bazel comes with tradeoffs, each subproject can make its own decision in this regard. This KEP **only** covers the main Kubernetes repository, and nothing else.
- Improving the previously existing make build
  - We should strongly consider improving the implementation and behavior of this build in the future, but this is largely orthogonal to whether we should consider maintaining two build systems. An anticipated outcome of this KEP is increased bandwidth available to improve our single build system.

## Proposal

<!--
This is where we get down to the specifics of what the proposal actually is.
This should have enough detail that reviewers can understand exactly what
you're proposing, but should not include things like API designs or
implementation. What is the desired outcome and how do we measure success?.
The "Design Details" section below is for the real
nitty-gritty.
-->

For the development (main/master) branch only, **NOT** the existing release branches:

1. Switch remaining CI usage (mostly a few presubmits) to use the make build.
  - Most of CI already uses the make builds, excluding some presubmits, we will need to switch these (generally a flag flip in the CI configuration).
  - Most of periodic testing consumes pre-uploaded binaries from the make builds, and does not build at all. These will require no changes.
  - In areas where the make build generates fewer artifacts or exercises fewer paths than bazel, we will err on the side of parity with artifacts that end up in a kubernetes/kubernetes release
2. Remove the bazel build and associated tooling.
  - There are multiple scripts and LOTS of files related wholly to the bazel build in Kubernetes. Once we are confident that CI is no longer reliant on them we can remove these and relieving the maintenance toil.

No changes should be made to the release branches or their CI.

### Risks and Mitigations

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

### Test Plan

Non-blocking “make build” equivalents to kubernetes/kubernetes “bazel” CI jobs will be introduced (if they don’t already exist).
When the new jobs provide equivalent signal, they will be moved to blocking, and the old jobs will be retired.

This is relevant for at least the following jobs release-blocking and merge-blocking:
 - `pull-kubernetes-bazel-test` (this can be converted to ~ `make test`)
 - `pull-kubernetes-bazel-build` (this largely overlaps with other presubmits, if not for testing ~`bazel build //...` and can  likely be removed)
 - `periodic-bazel-build-<branch>` (this can likely already be removed in favor of `ci-kubernetes-build-<branch>`)
 - `periodic-bazel-test-<branch>`
 - `post-kubernetes-bazel-build` (this can likely already be removed, it’s unclear what depends on this job)

Again, this should not apply to existing release branches.

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

#### Alpha -> Beta Graduation

- Gather feedback from developers and surveys
- Complete features A, B, C
- Tests are in Testgrid and linked in KEP

#### Beta -> GA Graduation

- N examples of real-world usage
- N installs
- More rigorous forms of testing—e.g., downgrade tests and scalability tests
- Allowing time for feedback

**Note:** Generally we also wait at least two releases between beta and
GA/stable, because there's no opportunity for user feedback, or even bug reports,
in back-to-back releases.

#### Removing a Deprecated Flag

- Announce deprecation and support policy of the existing flag
- Two versions passed since introducing the functionality that deprecates the flag (to address version skew)
- Address feedback on usage/changed behavior, provided on GitHub issues
- Deprecate the flag

**For non-optional features moving to GA, the graduation criteria must include 
[conformance tests].**

[conformance tests]: https://git.k8s.io/community/contributors/devel/sig-architecture/conformance-tests.md
-->

This will be declared stable/GA when:
- All kubernetes/kubernetes `master` branch CI jobs use the preexisting make build system
<!-- TODO: as old jobs are rotated out each release anyhow, it may be acceptable to reduce the stable target to just the current development branch -->
- Bazel-related source files and related tooling are removed from the kubernetes/kubernetes repository on currently supported release branches and the current development branch
  - This will only happen for release branches as we phase out support for older releases, rotating in new supported releases that never contained the Bazel build
- Bazel-related configuration/presets are removed from kubernetes/kubernetes CI jobs in kubernetes/test-infra


As bazel-built artifacts are not built or distributed as part of a kubernetes/kubernetes release, there is no deprecation window required.

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

n/a. Not relevant to upgrades. Existing release builds and upgrade CI use make.

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

n/a.

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

_This section must be completed when targeting alpha to a release._

* **How can this feature be enabled / disabled in a live cluster?**

  N/A

* **Does enabling the feature change any default behavior?**
  
  N/A

* **Can the feature be disabled once it has been enabled (i.e. can we roll back
  the enablement)?**

  N/A

* **What happens if we reenable the feature if it was previously rolled back?**

  N/A

* **Are there any tests for feature enablement/disablement?**

  N/A

### Rollout, Upgrade and Rollback Planning

_This section must be completed when targeting beta graduation to a release._

* **How can a rollout fail? Can it impact already running workloads?**

  N/A

* **What specific metrics should inform a rollback?**

  N/A

* **Were upgrade and rollback tested? Was the upgrade->downgrade->upgrade path tested?**

  N/A

* **Is the rollout accompanied by any deprecations and/or removals of features, APIs, 
fields of API types, flags, etc.?**

  N/A

### Monitoring Requirements

_This section must be completed when targeting beta graduation to a release._

* **How can an operator determine if the feature is in use by workloads?**

  N/A

* **What are the SLIs (Service Level Indicators) an operator can use to determine 
the health of the service?**

  N/A

* **What are the reasonable SLOs (Service Level Objectives) for the above SLIs?**

  N/A

* **Are there any missing metrics that would be useful to have to improve observability 
of this feature?**

  N/A

### Dependencies

_This section must be completed when targeting beta graduation to a release._

* **Does this feature depend on any specific services running in the cluster?**

  N/A


### Scalability

_For alpha, this section is encouraged: reviewers should consider these questions
and attempt to answer them._

_For beta, this section is required: reviewers must answer these questions._

_For GA, this section is required: approvers should be able to confirm the
previous answers based on experience in the field._

* **Will enabling / using this feature result in any new API calls?**

  N/A

* **Will enabling / using this feature result in introducing new API types?**

  N/A

* **Will enabling / using this feature result in any new calls to the cloud 
provider?**

  N/A

* **Will enabling / using this feature result in increasing size or count of 
the existing API objects?**

  N/A

* **Will enabling / using this feature result in increasing time taken by any 
operations covered by [existing SLIs/SLOs]?**

  N/A

* **Will enabling / using this feature result in non-negligible increase of 
resource usage (CPU, RAM, disk, IO, ...) in any components?**

  N/A

### Troubleshooting

The Troubleshooting section currently serves the `Playbook` role. We may consider
splitting it into a dedicated `Playbook` document (potentially with some monitoring
details). For now, we leave it here.

_This section must be completed when targeting beta graduation to a release._

* **How does this feature react if the API server and/or etcd is unavailable?**

  N/A

* **What are other known failure modes?**

  N/A

* **What steps should be taken if SLOs are not being met to determine the problem?**

  N/A

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

- 2020-02-04 - Initial KEP draft / provisional [#2421](https://github.com/kubernetes/enhancements/pull/2421)
- 2020-02-08 - KEP Implementable [#2469](https://github.com/kubernetes/enhancements/pull/2469)
- 2020-04-01 - KEP Alpha, Beta in Kubernetes 1.21
  - There is no distinct alpha/beta for this KEP, only alpha/beta (implemented at HEAD) vs stable (all supported branches)

See also PR listing: https://github.com/kubernetes/enhancements/issues/2420#issuecomment-791024902

## Drawbacks

<!--
Why should this KEP _not_ be implemented?
-->

- The make system works best from x86 (though it can cross compile all platforms), largely due to a [bug in the kube-cross image build] and some other small oversights. Improving the existing make system is an explicit non-goal of this KEP, but we support addressing these oversights, and expect the outcome of this KEP will naturally increase available bandwidth to do so. 
  - A few other issues related to CGO / $CC on non-amd64 build hosts have already received pull requests / fixes.
  - Remaining issues largely appear trivial, and should be fixed regardless, especially with the proliferation of non-amd64 developer hardware, where contributors will want to build releases matching the official release (with make).

- Some contributors may be used to the Bazel CLI, however indications are that this is not true for the majority of contributors. Existing discussions on the matter of removing Bazel from the Kubernetes project have received overwhelming support, most (but certainly not all) of the few suggestions against it do not appear to be from [community members] / active upstream contributors. See: [kubernetes/kubernetes#88533]

- In Kubernetes' CI we have bazel [remote caching] enabled which theoretically reduces our resource consumption and improves build times. In practice this gain is reduced currently, due to enabling multiple runs for unit tests to eliminate flakiness. Current measurements show equivalent runtime for the two builds.
  - `go build` has developed high quality caching we can leverage if we need, we have a prototype of this already
  - Kubernetes has enabled repeated runs of "unit" tests to surface flakes, which causes both `go` and Bazel to not cache test results
  - We've disabled caching for large, poorly cachable objects like Docker images anyhow


## Alternatives

<!--
What other approaches did you consider, and why did you rule them out? These do
not need to be as detailed as the proposal, but should include enough
information to express the idea and why it was not acceptable.
-->

1. Continue maintaining both build systems

This has major drawbacks:

- This continues to eat developer time without significant return, for the most part releases, the bulk of CI (periodic tests), and contributor development use the make build(s).

- Key components of our development setup do not easily port to bazel, so it will continue to be an "also":
  - code generators: Kubernetes uses a lot of generated code, while Bazel *can* run code generators just fine without checking in the generated sources, we need to check them in for consumption by non-bazel users (i.e. external projects), and as a result we have never leveraged this. These generators are largely incompatible with bazel "philosophy":
    - much of the k8s codegen uses fake go build tags, which would require an external process (i.e. gazelle extension) to turn into build files
    - additionally, some codegen relies on certain dependencies, which the gazelle extension would need to figure out
    - many of the code generators are optimized to load the entire Go tree, then generate lots of code at once, which is somewhat incompatible with Bazel's approach to working on a package-by-package level. Generating code package-by-package is much slower (due to having to reparse the tree each time).
  - separate "hack/tools" go module for linters etc: not only do linters not work well (because any source change busts the cache anyhow), but rules_go does not do multiple go modules well. Multiple go modules allows us to isolate development dependencies (like linters) from release binary dependencies, easing dependency management

2. Improve bazel integration and drop the make based build

In addition to the points made in 1.) above as to why this is not particularly viable for some of our existing development patterns:
- We would need to improve support for CGO or eliminate CGO in Kubernetes, both of which would be somewhat expensive to develop
  - At minimum kubelet definitely requires CGO for OS integration (e.g. selinux)
  - CGO pkg-config directives do not work in `rules_go`, requiring brittle work-arounds
  - Cross compiling with CGO under bazel is tricky, and despite @ixdy getting close at one point, never shipped in Kubernetes, let alone portably, or capable of shipping a full release.
    - @ixdy suspects this would largely have to be reimplemented now
- This is less likely to grow the amount of potential contributor bandwidth available to improve the kubernetes/kubernetes build system. We have a larger pool of contributors today who have demonstrated experience updating our existing make build system vs. updating bazel's components and our usage of it. Traction with our contributor base is important to ongoing project health.
- Large portions of Kubernetes are intended for dual use internally and as exported go libraries ("staging") to be consumed by other projects, where we'd still need to support use without bazel (i.e. checked-in generated code etc.).


## Infrastructure Needed (Optional)

<!--
Use this section if you need things from the project/SIG. Examples include a
new subproject, repos requested, or GitHub details. Listing these here allows a
SIG to get the process for these resources started right away.
-->

No additional infrastructure is necessary. The existing infrastructure largely uses and hosts make-based builds as-is.

We may in fact consider turning down some of the caching infrastructure if no remaining projects are using it.
At least one subproject does use bazel with a different caching deployment than Kubernetes, but it is not apparent that any other subprojects use the Kubernetes build cache implementation ([greenhouse]).

[bug in the kube-cross image]: https://github.com/kubernetes/kubernetes/issues/75114
[remote caching]: https://docs.bazel.build/versions/master/remote-caching.html
[greenhouse]: https://github.com/kubernetes/test-infra/blob/8c0f54c19923c181244e1bdfa56e5339be497def/greenhouse/README.md
[community members]: https://github.com/kubernetes/community/blob/master/community-membership.md
[kubernetes/kubernetes#88533] https://github.com/kubernetes/kubernetes/issues/88553
