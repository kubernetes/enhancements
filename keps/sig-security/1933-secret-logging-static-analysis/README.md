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

# KEP-1933: Defend Against Logging Secrets via Static Analysis

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
  - [Notes/Constraints/Caveats](#notesconstraintscaveats)
  - [Risks and Mitigations](#risks-and-mitigations)
- [Design Details](#design-details)
  - [Test Plan](#test-plan)
  - [Graduation Criteria](#graduation-criteria)
    - [Alpha (1.20)](#alpha-120)
    - [Alpha -&gt; Beta Graduation](#alpha---beta-graduation)
    - [Beta](#beta)
    - [Beta -&gt; Stable Graduation](#beta---stable-graduation)
    - [Stable](#stable)
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
- [x] Production readiness review approved
- [x] "Implementation History" section is up-to-date for milestone
- [x] User-facing documentation has been created in [kubernetes/website], for publication to [kubernetes.io]
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

*Taint propagation analysis* can provide insight into how data spreads and is consumed within a program.
It can be use used to harden the boundaries for those data which require special handling.

This Kubernetes Enhancement Proposal (KEP) proposes such analysis to be used
during testing to prevent various types of sensitive information from leaking via logs.
For a complimentary efforts at runtime, see [KEP-1753: Kubernetes system components logs sanitization](https://github.com/kubernetes/enhancements/pull/1754).

## Motivation

<!--
This section is for explicitly listing the motivation, goals and non-goals of
this KEP.  Describe why the change is important and the benefits to users. The
motivation section can optionally provide links to [experience reports] to
demonstrate the interest in a KEP within the wider Kubernetes community.

[experience reports]: https://github.com/golang/go/wiki/ExperienceReports
-->

The 2019 [Trail of Bits security audit](https://github.com/kubernetes/community/blob/master/wg-security-audit/findings/Kubernetes%20Final%20Report.pdf)
contained several issues centered on the exposure of secrets to logs or execution environment.

* Trail of Bits issue 6, Issue [\#81114](https://github.com/kubernetes/kubernetes/issues/81114): Bearer tokens are revealed in logs.
* Trail of Bits issue 9, Issue [\#81117](https://github.com/kubernetes/kubernetes/issues/81117) :Environment variables expose sensitive data.
* Trail of Bits issue 22. Issue [\#81130](https://github.com/kubernetes/kubernetes/issues/81130): iSCSI volume storage cleartext secrets in logs.

In light of these issues, the audit authors' long-term suggestions regarding secret management includes:

> **Ensure that sensitive data cannot be trivially stored in logs.**
> Prevent dangerous logging actions with improved code review policies.
> Redact sensitive information with logging filters.
> Together, these actions can help to prevent sensitive data from being exposed in the logs.

This KEP represents part of "improved code review policies."

### Goals

<!--
List the specific goals of the KEP. What is it trying to achieve? How will we
know that this has succeeded?
-->

- Detect logging of secrets during testing.
- Block pull requests that would introduce such violations.

### Non-Goals

<!--
What is out of scope for this KEP? Listing non-goals helps to focus discussion
and make progress.
-->

- Detect *all* instances of potential secret logging.
- Replace meaningful review of how secrets are handled.

## Proposal

<!--
This is where we get down to the specifics of what the proposal actually is.
This should have enough detail that reviewers can understand exactly what
you're proposing, but should not include things like API designs or
implementation. The "Design Details" section below is for the real
nitty-gritty.
-->

[`go-flow-levee`](https://github.com/google/go-flow-levee) is a highly configurable
taint propagation analysis tool for Go code.
Independent use successfully identified a potential risk of secret logging,
remedied in [PR \#90413](https://github.com/kubernetes/kubernetes/pull/90413).
This KEP proposes that `go-flow-levee` be used as part of Prow's testing
 of pull requests to limit the introduction of new secret logging.

As a taint propagation analysis tool, `go-flow-levee` examines potential
 code paths of data *sources* and emits a report if any reaches a *sink*.
In use within Kubernetes, a source would consist of any field that contains a secret, e.g., `iscsi.iscsiDisk.secret`,
 or particular method calls that would return a secret, e.g., `os.Getenv("CFSSL_CA_PK_PASSWORD")`.
A sink consists of any `klog` logging method.

Taint propagation analysis gives additional consideration is given to how data
 may be transformed or extracted to other values via *propagators*,
 and how source data may be processed for safe consumption via *sanitizers*.
See the `go-flow-levee` documentation for details.

While configuration of source identification can be done via manually configured regexp,
this KEP would benefit from a set of standard Kubernetes go lang struct tags indicating which fields are expected to contain secrets,
as proposed in [KEP-1753](https://github.com/kubernetes/enhancements/pull/1754).

### Notes/Constraints/Caveats

<!--
What are the caveats to the proposal?
What are some important details that didn't come across above?
Go in to as much detail as necessary here.
This might be a good place to talk about core concepts and how they relate.
-->

`go-flow-levee` is in active development and has not yet reached a `v1.0` release.
It is, however, being developed with specific interest in application to Kubernetes.


### Risks and Mitigations
<!--
What are the risks of this proposal, and how do we mitigate? Think broadly.
For example, consider both security and how this will impact the larger
Kubernetes ecosystem.

How will security be reviewed, and by whom?

How will UX be reviewed, and by whom?

Consider including folks who also work outside the SIG or subproject.
-->

While the test will not initially be blocking, if it produces many false positives, 
it may train developers to ignore issues.
Additionally, *because* it is not initially blocking, it risks being overlooked
as unimportant even when findings are relevant.

Should issues reach a branch, either by the happenstance of a merge, overridden warnings, or analysis flakiness,
reported findings may include those out of scope for the change set in a given PR.
Such incidents would provide confusion and toil for developers, but could be quickly corrected, suppressed via configuration, or the offending commit reverted.

Changes to `test-infra` carry with them the potential for inconvenience,
should they introduce any instability to wider testing.  While diligent review
mitigates this, it does not remove the concern completely.

## Design Details

<!--
This section should contain enough information that the specifics of your
change are understandable. This may include API specs (though not always
required) or even code snippets. If there's any ambiguity about HOW your
proposal will be implemented, this is the place to discuss them.
-->

In-depth analysis interacts with build artifacts, and as a result is quickly
stymied by Bazel's sandboxing of build and test environments.
It is similar in this regard to linters, and indeed a standard execution path for
analysis in Go is via `go vet -vettool=...`.

This KEP would introduce a execution script to `hack/` which handles passing
the analyzer and config to our vet process.
Within Kubernetes, `go vet` is executed via `make vet`.
The analyzer can be passed to this in the introduced `hack/` script via
`make vet WHAT="..."`.
Dependency on `go-flow-levee` will be introduced to `hack/tools` to avoid polluting
the main Kubernetes dependencies.

This KEP will also introduce execution of this script as a Prow presubmit job.
Any findings by the analysis will constitute a test failure.
Failure will be initially non-blocking, per the graduation criteria below.
Analysis findings reporting the position of both source and sink,
 indicating to a developer which log call must be corrected and/or which argument must be sanitized.

### Test Plan

As a testing target, testing will consist of examples of Kubernetes-specific
cases that we expect static analysis to detect.  No full-scale Kubernetes
integration/e2e tests will be necessary.

For analysis based on `go-flow-levee`, this will consist of sample Kubernetes code based on the same configuration used in presubmit scanning.
This test will be examined via the `analysistest` package to ensure analysis produces expected diagnostics.
As part of testing of our testing process, these tests should belong to `kubernetes/test-infra`.

### Graduation Criteria

#### Alpha (1.20)
- Analysis is manually triggered to run in Prow against a sampling of PRs.

#### Alpha -> Beta Graduation
- Test is validated as running soundly.
- Analysis configuration consumes fields tags introduced by [KEP-1753](https://github.com/kubernetes/enhancements/blob/master/keps/sig-instrumentation/1753-logs-sanitization/README.md) for identification of material that should not be logged.
  Configuration should also identify other sensitive data not bound to fields, if any are identified during KEP-1753's investigation.

#### Beta
- Analysis runs as a non-blocking presubmit check, warning developers of any findings in their changes.

#### Beta -> Stable Graduation
- Test is validated as running soundly at scale.
- No false positives, test failures, or other concerning issues are raised for 1-2 weeks.

#### Stable
- Analysis runs as a blocking presubmit test.

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
- Does this enhancement involve coordinating behavior in the control plane and
  in the kubelet? How does an n-2 kubelet without this feature available behave
  when this feature is used?
- Will any other components on the node change? For example, changes to CSI,
  CRI or CNI may require updating that component before the kubelet.
-->

## Production Readiness Review Questionnaire

<!--

Production readiness reviews are intended to ensure that features merging into
Kubernetes are observable, scalable and supportable; can be safely operated in
production environments, and can be disabled or rolled back in the event they
cause increased failures in production. See more in the PRR KEP at
https://git.k8s.io/enhancements/keps/sig-architecture/20190731-production-readiness-review-process.md.

The production readiness review questionnaire must be completed for features in
v1.19 or later, but is non-blocking at this time. That is, approval is not
required in order to be in the release.

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

As part of Prow, enablement is managed by configuration in `kubernetes/test-infra`.
As the test target and tool version are fixed in `kubernetes/kubernetes/hack/tools`,
rollback can be handled by reverting any offending commit to `hack/tools`.

### Rollout, Upgrade and Rollback Planning

As a third-party dependency, analyzer upgrading is handled by upgrading
the version targeted by `kubernetes/kubernetes/hack/tools/`.
Tool configuration at `kubernetes/kubernetes/hack/testdata/levee` can be updated
independently, though may be required during tool upgrading.

### Monitoring Requirements

As a Prow job, visibility is provided by existing Prow monitoring via dashboards and alerts.

### Dependencies

This KEP introduces a third-party dependency on `github.com/google/go-flow-levee`.

### Scalability

_For alpha, this section is encouraged: reviewers should consider these questions
and attempt to answer them._

_For beta, this section is required: reviewers must answer these questions._

_For GA, this section is required: approvers should be able to confirm the
previous answers based on experience in the field._

* **Will enabling / using this feature result in any new API calls?**

The Prow dashboard will serve an additional test dashboard.

* **Will enabling / using this feature result in introducing new API types?**

No.

* **Will enabling / using this feature result in any new calls to the cloud 
provider?**

No.

* **Will enabling / using this feature result in increasing size or count of 
the existing API objects?**

With the additional dashboard page, there may be a marginal increase in traffic served.

* **Will enabling / using this feature result in increasing time taken by any 
operations covered by [existing SLIs/SLOs]?**

While adding a new test will increase cycle-time taken to perform taken,
parallelization of test tasks will prevent increase in wall-time.
At time of writing, analysis of Kubernetes takes ~5 minutes.

* **Will enabling / using this feature result in non-negligible increase of 
resource usage (CPU, RAM, disk, IO, ...) in any components?**
  Things to keep in mind include: additional in-memory state, additional
  non-trivial computations, excessive access to disks (including increased log
  volume), significant amount of data sent and/or received over network, etc.
  This through this both in small and large cases, again with respect to the
  [supported limits].

There may be non-negligible increase in testing resource usage.
While analysis is much simpler than tests that require a cluster for testing,
implementation should benchmark to estimate actual computational costs of analysis.

### Troubleshooting

While this KEP does not change cluster behavior, any reported finding should be
communicated clearly such that developer correction can proceed as smoothly as possible.
During non-blocking release stages, this should include instructions for reporting false-positives if the PR author believes the findings are incorrect.
During blocking release stages, this should include instructions for escalating possible false-positives to avoid blocking other PRs and how to contact contributors with `/override` permissions to approve bypass of analysis.

Assistance in resolving issues identified by the analyzer can be found in the [Verification Tests Documentation](https://github.com/kubernetes/community/blob/master/contributors/devel/sig-testing/verify-tests.md)

Analyzer failures or bugs should be reported to [`go-flow-levee` Issues](http://github.com/google/go-flow-levee/issues).

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

* 2020-08-13: Initial Proposal Merged (#1936)
* 2020-09-10: Alpha state - Non-blocking, manually triggered test added to Prow (kubernetes/test-infra/pull/19181)
* 2020-12-16: Beta state - Prow test converted to automatically trigger (kubernetes/test-infra/pull/20164)
* 2020-02-11: Stable state - Prow test is now blocking (kubernetes/test-infra/pull/20836)

## Drawbacks

<!--
Why should this KEP _not_ be implemented?
-->

As a blocking test, there is a risk for developer toil in the event of any
false-positive or test flakiness.
This can be mitigated by any contributor with `/override` permissions.

In the unexpected event that Prow-bot merges two PR without first rebasing one to the HEAD of the target branch, it could be possible for an analysis violation to reach a given branch.
Like any other failing test that could reach  `master`, all subsequent PRs would be blocked by spurious failure.
This could be mitigated if analysis first executes a baseline against the target branch without the changes introduced by a PR.
However, such additional testing has not proven necessary given the rarity of both such Prow-bot misbehavior and the sort of PR diffs necessary to introduce a new violation.

As this analysis depends on project-specific considerations of what constitutes
a secret or a sink, periodic review is required to ensure configuration is kept up-to-date.
This is mitigated somewhat with a consistent use of field tags,
as proposed in [KEP-1753](https://github.com/kubernetes/enhancements/pull/1754),
though correct application of field tags would also be subject to periodic review.


## Alternatives

<!--
What other approaches did you consider, and why did you rule them out? These do
not need to be as detailed as the proposal, but should include enough
information to express the idea and why it was not acceptable.
-->

[GitHub's CodeQL](https://securitylab.github.com/tools/codeql) includes taint analysis
and permits general SSA graph queries.  While CodeQL may provide similar testing, [its own documentation](https://lgtm.com/help/lgtm/about-automated-code-review) indicates that any findings would not be blocking.
Given the intended scope of this KEP as a means to block potential security concerns, blocking on detection is of heightened interest.
CodeQL could be used to augment coverage in the future, however.

While other static analysis tools exist for Go, these tend towards more general linters.
[`gosec`](https://github.com/securego/gosec), for instance, can be used to detect
hard-coded tokens or use of cryptographically broken packages, e.g., `crypto/md5`.
However, such linters are insufficient for our use, as they do not allow for project-specific configuration
to identify sources, sinks, etc.  Additionally, linters like `gosec` or even `go vet` can change their specification with new versions,
resulting in new findings breaking CI.

There are few other developer analyzers that provide depth greater than a linter.
[`gotcha`](https://github.com/akwick/gotcha), the <ins>go</ins> <ins>t</ins>aint <ins>ch</ins>ecker <ins>a</ins>nalyzer,
provides a basic taint propagation analysis.
However, it does not provide the ability to specify sanitizers or other fine-tuning.

## Infrastructure Needed (Optional)

<!--
Use this section if you need things from the project/SIG. Examples include a
new subproject, repos requested, or GitHub details. Listing these here allows a
SIG to get the process for these resources started right away.
-->

This KEP will introduce a new Prow test.
No additional infrastructure beyond that which already exists should be necessary.
