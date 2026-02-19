<!--
**Note:** When your KEP is complete, all of these comment blocks should be removed.

Follow the guidelines of the [documentation style guide].
In particular, wrap lines to a reasonable length, to make it
easier for reviewers to cite specific portions, and to minimize diff churn on
updates.

[documentation style guide]: https://github.com/kubernetes/community/blob/master/contributors/guide/style-guide.md

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
# KEP-5905: Migrate `kubernetes-mixin` from `kubernetes-monitoring` to `kubernetes-sigs` organization

**Please note that quite a few sections from the template were dropped in this
KEP, as they are not applicable to the nature of the proposal.**

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
      - [Prerequisite testing updates](#prerequisite-testing-updates)
      - [Unit tests](#unit-tests)
      - [Integration tests](#integration-tests)
      - [e2e tests](#e2e-tests)
  - [Graduation Criteria](#graduation-criteria)
    - [ALPHA](#alpha)
    - [BETA](#beta)
    - [STABLE](#stable)
  - [Upgrade / Downgrade Strategy](#upgrade--downgrade-strategy)
  - [Version Skew Strategy](#version-skew-strategy)
  - [Dependencies](#dependencies)
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
- [X] (R) Graduation criteria is in place
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
-->

This KEP aims to migrate the `kubernetes-monitoring/kubernetes-mixin`
repository under the `kubernetes-sigs` organization, to better reflect its
Kubernetes-specific nature and to align with the organizational structure of
Kubernetes-related projects. The mixins provided by the repository are a set of
monitoring rules and dashboards that are fulfilled by CNCF graduated projects,
but do not, and will not, engage any in-tree Kubernetes components directly, in
any way.

This migration will help clarify the purpose and scope of the repository, and
facilitate better collaboration and maintenance within the Kubernetes
community.

## Motivation

<!--
This section is for explicitly listing the motivation, goals, and non-goals of
this KEP.  Describe why the change is important and the benefits to users. The
motivation section can optionally provide links to [experience reports] to
demonstrate the interest in a KEP within the wider Kubernetes community.

[experience reports]: https://github.com/golang/go/wiki/ExperienceReports
-->

`kubernetes-monitoring/kubernetes-mixin` has provided the de-facto set of
monitoring mixins tailered for the Kubernetes ecosystem for a long time, and
has been widely adopted by users and vendors alike. The migration not only
reflects the Kubernetes-specific nature of the repository, but also aims to
foster better [collaboration and maintenance] within the Kubernetes community.

[collaboration and maintenance]: https://kubernetes.slack.com/archives/C20HH14P7/p1768490389495989

### Goals

<!--
List the specific goals of the KEP. What is it trying to achieve? How will we
know that this has succeeded?
-->

* Migrate `kubernetes-monitoring/kubernetes-mixin` repository to `kubernetes-sigs/kubernetes-mixin`.
* Continue operations with the same expectations as they were with `kubernetes-monitoring/kubernetes-mixin`.

### Non-Goals

<!--
What is out of scope for this KEP? Listing non-goals helps to focus discussion
and make progress.
-->

* Any changes to the mixins themselves (as a part of the migration), or to the way they are maintained.

## Proposal

<!--
This is where we get down to the specifics of what the proposal actually is.
This should have enough detail that reviewers can understand exactly what
you're proposing, but should not include things like API designs or
implementation. What is the desired outcome and how do we measure success?.
The "Design Details" section below is for the real
nitty-gritty.
-->

There isn't much to say here, as the proposal is straightforward: we want to
migrate the `kubernetes-monitoring/kubernetes-mixin` repository to
`kubernetes-sigs/kubernetes-mixin`, and continue operations as they were
before, with the same expectations. Users should be able to reference the new
repository without any issues, and the mixins should continue to work as they
did before, as the migration is mostly a change in ownership and location,
rather than in functionality. The migration should be seamless for users, and
the mixins should continue to be maintained and updated as they were before,
with the same level of quality and reliability. To not break existing
references, CI automations will be setup in the `kubernetes-monitoring`
counterpart, to enable mirroring changes from the `kubernetes-sigs` repository.

Also, as mentioned earlier, and most importantly, the migration should also
help clarify the purpose and scope of the repository, and facilitate better
collaboration and maintenance within the Kubernetes community.

### Risks and Mitigations

<!--
What are the risks of this proposal, and how do we mitigate? Think broadly.
For example, consider both security and how this will impact the larger
Kubernetes ecosystem.

How will security be reviewed, and by whom?

How will UX be reviewed, and by whom?

Consider including folks who also work outside the SIG or subproject.
-->

The mixins build over signals exposed by the involved metric sources. False
postives are absolutely possible, however, vulnerabilities in the mixins
themselves are unlikely, as they are mostly static definitions of monitoring
rules and dashboards. The mixins are also not directly involved in any critical
path of Kubernetes components, so their failure should not impact the cluster's
stability or security.

## Design Details

<!--
This section should contain enough information that the specifics of your
change are understandable. This may include API specs (though not always
required) or even code snippets. If there's any ambiguity about HOW your
proposal will be implemented, this is the place to discuss them.
-->

N/A

### Test Plan

<!--
**Note:** *Not required until targeted at a release.*
The goal is to ensure that we don't accept enhancements with inadequate testing.

All code is expected to have adequate tests (eventually with coverage
expectations). Please adhere to the [Kubernetes testing guidelines][testing-guidelines]
when drafting this test plan.

[testing-guidelines]: https://git.k8s.io/community/contributors/devel/sig-testing/testing.md
-->

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

Unit tests for the mixins live under the [`tests` directory].

[`tests` directory]: https://github.com/kubernetes-monitoring/kubernetes-mixin/tree/master/tests

##### Integration tests

<!--
Integration tests are contained in https://git.k8s.io/kubernetes/test/integration.
Integration tests allow control of the configuration parameters used to start the binaries under test.
This is different from e2e tests which do not allow configuration of parameters.
Doing this allows testing non-default options and multiple different and potentially conflicting command line options.
For more details, see https://github.com/kubernetes/community/blob/master/contributors/devel/sig-testing/testing-strategy.md

If integration tests are not necessary or useful, explain why.
-->

<!--
This question should be filled when targeting a release.
For Alpha, describe what tests will be added to ensure proper quality of the enhancement.

For Beta and GA, document that tests have been written,
have been executed regularly, and have been stable.
This can be done with:
- permalinks to the GitHub source code
- links to the periodic job (typically https://testgrid.k8s.io/sig-release-master-blocking#integration-master), filtered by the test name
- a search in the Kubernetes bug triage tool (https://storage.googleapis.com/k8s-triage/index.html)
-->

Integration tests for the mixins live under the [`tests` directory].

[`tests` directory]: https://github.com/kubernetes-monitoring/kubernetes-mixin/tree/master/tests

##### e2e tests

<!--
This question should be filled when targeting a release.
For Alpha, describe what tests will be added to ensure proper quality of the enhancement.

For Beta and GA, document that tests have been written,
have been executed regularly, and have been stable.
This can be done with:
- permalinks to the GitHub source code
- links to the periodic job (typically a job owned by the SIG responsible for the feature), filtered by the test name
- a search in the Kubernetes bug triage tool (https://storage.googleapis.com/k8s-triage/index.html)

We expect no non-infra related flakes in the last month as a GA graduation criteria.
If e2e tests are not necessary or useful, explain why.
-->

While no end-to-end tests exist currently, they may be bootstrapped over the
already available [local development] functionality, building over
[ContainerSolutions/prom-metrics-check] in the future, to establish greater
confidence between the dependency versions mentioned in the compatibility
matrix.

[local development]: https://github.com/kubernetes-monitoring/kubernetes-mixin/tree/master#local-development
[ContainerSolutions/prom-metrics-check]: https://github.com/ContainerSolutions/prom-metrics-check

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
- More rigorous forms of testing—e.g., downgrade tests and scalability tests
- All functionality completed
- All security enforcement completed
- All monitoring requirements completed
- All testing requirements completed
- All known pre-release issues and gaps resolved

**Note:** Beta criteria must include all functional, security, monitoring, and testing requirements along with resolving all issues and gaps identified

#### GA

- N examples of real-world usage
- N installs
- Allowing time for feedback
- All issues and gaps identified as feedback during beta are resolved

**Note:** GA criteria must not include any functional, security, monitoring, or testing requirements.  Those must be beta requirements.

**Note:** Generally we also wait at least two releases between beta and
GA/stable, because there's no opportunity for user feedback, or even bug reports,
in back-to-back releases.

**For non-optional features moving to GA, the graduation criteria must include
[conformance tests].**

[conformance tests]: https://git.k8s.io/community/contributors/devel/sig-architecture/conformance-tests.md

#### Deprecation

<!--
- Announce deprecation and support policy of the existing flag
- Two versions passed since introducing the functionality that deprecates the flag (to address version skew)
- Address feedback on usage/changed behavior, provided on GitHub issues
- Deprecate the flag
-->

#### ALPHA

- KEP merged to indicate SIG acceptance of the proposal.
- `kubernetes-mixin` repository created under `kubernetes-sigs` organization.

#### BETA

- CI automations setup in the `kubernetes-monitoring` counterpart, to enable
  mirroring changes from the `kubernetes-sigs` repository, so as it not break
  existing references.
- Any additional changes necessary to ensure the migration is smooth, that
  surface during ALPHA and BETA reviews, are implemented.

#### STABLE

- Announce migration in SIG Instrumentation mailing list and
  [#sig-instrumentation] Slack channel.
- Setup a recommendation on both repository's README files for users to
  reference the `kubernetes-sigs` repository in their mixins. This is
  encouraged, but not doing this will never break any functionality whatsoever.

[#sig-instrumentation]: https://kubernetes.slack.com/archives/C20HH14P7

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

While the repository tries its best to maintain backward compatibility and keep
up with the latest versions of the metric sources, upgrading or downgrading
should ideally be preceded by a review of the [compatibility matrix] to ensure
that the mixins being used (from the set of releases) are compatible with the
versions of the metric sources in use.

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

The version skews of the repository, in general, are dictated by the metric
sources (see below) that the mixins rely on. The [compatibility matrix]
documents the relationship between the mixins and the metric sources.

[compatibility matrix]: https://github.com/kubernetes-monitoring/kubernetes-mixin/#releases

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

- [Prometheus]
  - Usage description: Kubernetes-specific monitoring rules and alerts, defined
    in `PrometheusRules` provided by the mixins repository, rely on the
    presence of [Prometheus] for their enforcement and application.
    - Impact of its outage on the feature: `PrometheusRules` no longer work, although alternatives such as [VictoriaMetrics] exist.
    - Impact of its degraded performance or high-error rates on the feature: `PrometheusRules` may no longer work.

- [Alertmanager]
  - Usage description: Kubernetes-specific monitoring alerts, defined in
    `PrometheusRules` provided by the mixins repository, rely on the presence
    of [Alertmanager] for their notification and alerting capabilities.
    - Impact of its outage on the feature: Alerts no longer get delivered.
    - Impact of its degraded performance or high-error rates on the feature: Alerts may not get delivered.

- [Grafana]
  - Usage description: Kubernetes-specific monitoring dashboards, defined using
    the [`grafonnet`] Jsonnet library, rely on the presence of [Grafana] for
    their visualization.
    - Impact of its outage on the feature: Dashboards no longer work, although alternatives such as [Perses] exist.
    - Impact of its degraded performance or high-error rates on the feature: Dashboards may no longer work.

Additionally, the mixins themselves are sourced from metrics exposed by
components within the CNCF landscape, which may or may not be in-tree to
Kubernetes. These include:
- [`kubernetes/kube-state-metrics`]
- [`prometheus/node_exporter`]
- [`kubernetes/kubernetes`] (`kube-scheduler`, `kube-controller-manager`, `kube-apiserver`, `kube-proxy`, `kubelet`)

[Prometheus]: https://github.com/prometheus/prometheus
[VictoriaMetrics]: https://github.com/VictoriaMetrics/VictoriaMetrics
[Alertmanager]: https://github.com/prometheus/alertmanager
[Grafana]: https://github.com/grafana/grafana
[grafonnet]: https://github.com/grafana/grafonnet
[Perses]: https://github.com/perses/perses
[kubernetes/kube-state-metrics]: https://github.com/kubernetes/kube-state-metrics
[prometheus/node_exporter]: https://github.com/prometheus/node_exporter

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

TBD

## Drawbacks

<!--
Why should this KEP _not_ be implemented?
-->

Maintenance efforts will require bandwidth allocations from the SIG from time
to time.

## Alternatives

<!--
What other approaches did you consider, and why did you rule them out? These do
not need to be as detailed as the proposal, but should include enough
information to express the idea and why it was not acceptable.
-->

The only other alternative is to not migrate the repository. This directly
works against the proposal and isn't ideal for the reasons discussed above.

## Infrastructure Needed (Optional)

<!--
Use this section if you need things from the project/SIG. Examples include a
new subproject, repos requested, or GitHub details. Listing these here allows a
SIG to get the process for these resources started right away.
-->

SIG Instrumentation requests a `kubernetes-mixin` repository under the
`kubernetes-sigs` organization.
