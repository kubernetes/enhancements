<!--
**Note:** When your KEP is complete, all of these comment blocks should be removed.

To get started with this template:

- [x] **Pick a hosting SIG.**
  Make sure that the problem space is something the SIG is interested in taking
  up.  KEPs should not be checked in without a sponsoring SIG.
- [x] **Create an issue in kubernetes/enhancements**
  When filing an enhancement tracking issue, please ensure to complete all
  fields in that template.  One of the fields asks for a link to the KEP.  You
  can leave that blank until this KEP is filed, and then go back to the
  enhancement and add the link.
- [x] **Make a copy of this template directory.**
  Copy this template into the owning SIG's directory and name it
  `NNNN-short-descriptive-title`, where `NNNN` is the issue number (with no
  leading-zero padding) assigned to your enhancement above.
- [x] **Fill out as much of the kep.yaml file as you can.**
  At minimum, you should fill in the "title", "authors", "owning-sig",
  "status", and date-related fields.
- [x] **Fill out this file as best you can.**
  At minimum, you should fill in the "Summary", and "Motivation" sections.
  These should be easy if you've preflighted the idea of the KEP with the
  appropriate SIG(s).
- [x] **Create a PR for this KEP.**
  Assign it to people in the SIG that are sponsoring this process.
- [x] **Merge early and iterate.**
  Avoid getting hung up on specific details and instead aim to get the goals of
  the KEP clarified and merged quickly.  The best way to do this is to just
  start with the high-level sections and fill out details incrementally in
  subsequent PRs.

Just because a KEP is merged does not mean it is complete or approved.  Any KEP
marked as a `provisional` is a working document and subject to change.  You can
denote sections that are under active debate as follows:

```
<<[UNRESOLVED optional short context or usernames ]>>
Stuff that is being argued.
<<[/UNRESOLVED]>>
```

When editing KEPS, aim for tightly-scoped, single-topic PRs to keep discussions
focused.  If you disagree with what is already in a document, open a new PR
with suggested changes.

One KEP corresponds to one "feature" or "enhancement", for its whole lifecycle.
You do not need a new KEP to move from beta to GA, for example.  If there are
new details that belong in the KEP, edit the KEP.  Once a feature has become
"implemented", major changes should get new KEPs.

The canonical place for the latest set of instructions (and the likely source
of this file) is [here](/keps/NNNN-kep-template/README.md).

**Note:** Any PRs to move a KEP to `implementable` or significant changes once
it is marked `implementable` must be approved by each of the KEP approvers.
If any of those approvers is no longer appropriate than changes to that list
should be approved by the remaining approvers and/or the owning SIG (or
SIG Architecture for cross cutting KEPs).
-->
# KEP-1739: kubeadm customization with patches

<!--
This is the title of your KEP.  Keep it short, simple, and descriptive.  A good
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
  - [User Stories (optional)](#user-stories-optional)
    - [Story 1](#story-1)
    - [Story 2](#story-2)
    - [Story 3](#story-3)
  - [Notes/Constraints/Caveats (optional)](#notesconstraintscaveats-optional)
  - [Risks and Mitigations](#risks-and-mitigations)
- [Design Details](#design-details)
  - [Code organization](#code-organization)
  - [Patch formats](#patch-formats)
  - [Patch file naming format](#patch-file-naming-format)
  - [Support for multiple patches per file](#support-for-multiple-patches-per-file)
  - [Test Plan](#test-plan)
  - [Graduation Criteria](#graduation-criteria)
    - [Alpha -&gt; Beta Graduation](#alpha---beta-graduation)
    - [Beta -&gt; GA Graduation](#beta---ga-graduation)
  - [Upgrade / Downgrade Strategy](#upgrade--downgrade-strategy)
  - [Version Skew Strategy](#version-skew-strategy)
- [Production Readiness Review Questionnaire](#production-readiness-review-questionnaire)
  - [Feature enablement and rollback](#feature-enablement-and-rollback)
  - [Rollout, Upgrade and Rollback Planning](#rollout-upgrade-and-rollback-planning)
  - [Monitoring requirements](#monitoring-requirements)
  - [Dependencies](#dependencies)
  - [Scalability](#scalability)
  - [Troubleshooting](#troubleshooting)
- [Implementation History](#implementation-history)
- [Drawbacks](#drawbacks)
- [Alternatives](#alternatives)
- [Infrastructure Needed (optional)](#infrastructure-needed-optional)
<!-- /toc -->

## Release Signoff Checklist

<!--
**ACTION REQUIRED:** In order to merge code into a release, there must be an
issue in [kubernetes/enhancements] referencing this KEP and targeting a release
milestone **before the [Enhancement Freeze](https://git.k8s.io/sig-release/releases)
of the targeted release**.

For enhancements that make changes to code or processes/procedures in core
Kubernetes i.e., [kubernetes/kubernetes], we require the following Release
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
- [ ] User-facing documentation has been created in [kubernetes/website], for publication to [kubernetes.io]
- [ ] Supporting documentation e.g., additional design documents, links to mailing list discussions/SIG meetings, relevant PRs/issues, release notes

<!--
**Note:** This checklist is iterative and should be reviewed and updated every time this enhancement is being considered for a milestone.
-->

[kubernetes.io]: https://kubernetes.io/
[kubernetes/enhancements]: https://git.k8s.io/enhancements
[kubernetes/kubernetes]: https://git.k8s.io/kubernetes
[kubernetes/website]: https://git.k8s.io/website

## Summary

<!--
This section is incredibly important for producing high quality user-focused
documentation such as release notes or a development roadmap.  It should be
possible to collect this information before implementation begins in order to
avoid requiring implementors to split their attention between writing release
notes and implementing the feature itself.  KEP editors, SIG Docs, and SIG PM
should help to ensure that the tone and content of the `Summary` section is
useful for a wide audience.

A good summary is probably at least a paragraph in length.
-->

This KEP proposes a replacement for the feature introduced by the kubeadm KEP
"20190722-Advanced-configurations-with-kubeadm-(Kustomize)".

It has the same scope and goals, allowing the users to amend manifests
generated by kubeadm using patches, but not using Kustomize as the backend
for applying the patches.

The existing Kustomize implementation in kubeadm is in Alpha state.

## Motivation

<!--
This section is for explicitly listing the motivation, goals and non-goals of
this KEP.  Describe why the change is important and the benefits to users.  The
motivation section can optionally provide links to [experience reports][] to
demonstrate the interest in a KEP within the wider Kubernetes community.

[experience reports]: https://github.com/golang/go/wiki/ExperienceReports
-->

The kubeadm team has decided that it is more beneficial if the project moves
to using raw patches instead of Kustomize. Kustomize introduces an undesired
dependency which can be avoided by using patches in a similar way kubectl
uses them.

### Goals

<!--
List the specific goals of the KEP.  What is it trying to achieve?  How will we
know that this has succeeded?
-->

- Implement a solution for applying patches over kubeadm generated manifests
during the common kubeadm commands "init", "join" and "upgrade" and their phases.
- Deprecate and remove the existing Kustomize solution after a grace period
of at least one release.
- To initially allow patching of static Pod manifests only.

### Non-Goals

<!--
What is out of scope for this KEP?  Listing non-goals helps to focus discussion
and make progress.
-->

- Validate if the user patches contain a good practice configuration and
sane security values.
- Allow patching of core addon configuration generated by kubeadm - e.g. CoreDNS,
kube-proxy (until further notice).
- Allowing patches to be added using the kubeadm ComponentConfig (until
further notice).
- Replace the kubeadm ComponentConfig. The sane defaults would still be
generated by kubeadm.

## Proposal

<!--
This is where we get down to the specifics of what the proposal actually is.
This should have enough detail that reviewers can understand exactly what
you're proposing, but should not include things like API designs or
implementation.  The "Design Details" section below is for the real
nitty-gritty.
-->

This KEP proposes introducing a new flag called `--experimental-patches`
next to the existing `--experimental-kustomize`. The flag has the
exact same semantics pointing to a directory with patches.
Once the feature graduates to Beta the flag will be renamed to `--patches`.
As part of its graduation the functionality can also be considered
as an addition to the kubeadm ComponentConfig.

All the relevant kubeadm commands such as "init", "join" and "upgrade"
will support the flag.

A `kustomization.yaml` file will not be required in the directory
and instead it will contain a list of patches named following
a specific format.

Once the flag is introduced, the existing flag `--experimental-kustomize`
will be marked as deprecated.

### User Stories (optional)

(Based on "20190722-Advanced-configurations-with-kubeadm-(Kustomize)")

#### Story 1

As a cluster administrator, I want to add a sidecar to the kube-apiserver
Pod for running an authorization web-hooks serving component.

#### Story 2

As a cluster administrator, I want to set timeouts for the kube-apiserver
liveness probes for edge clusters.

#### Story 3

As a cluster administrator, I want to upgrade my cluster preserving all the
patches that were applied during "init"/"join".

### Notes/Constraints/Caveats (optional)

<!--
What are the caveats to the proposal?
What are some important details that didn't come across above.
Go in to as much detail as necessary here.
This might be a good place to talk about core concepts and how they relate.
-->

NONE

### Risks and Mitigations

<!--
What are the risks of this proposal and how do we mitigate.  Think broadly.
For example, consider both security and how this will impact the larger
kubernetes ecosystem.

How will security be reviewed and by whom?

How will UX be reviewed and by whom?

Consider including folks that also work outside the SIG or subproject.
-->

(Based on "20190722-Advanced-configurations-with-kubeadm-(Kustomize)")

_Confusion between kubeadm ComponentConfig and usage of patches_

kubeadm already offers a way to implement cluster settings using
ComponentConfig. Adding a new feature for supporting customizations using
patches can create confusion in the users.

The kubeadm maintainers need to make it clear to the users in documentation
and release notes what is the scope of the new feature.

- Users are allowed to pass configuration for components using the
kubeadm ComponentConfiguration.
- kubeadm applies sane / secure defaults.
- The users can patch the generated manifests to their liking.

_Misleading expectations on the level of flexibility_

Even if the proposed solution is based on the user feedback/issues,
the kubeadm maintainers want to be sure the implementation is providing
the expected level of flexibility. In order to ensure that, feedback
will be examined before moving forward in graduating the feature to beta.

_Breaking changes post upgrade_

A change in a kubeadm manifest can make a patch apply to fail.

The kubeadm maintainers will work on release notes to make potential
breaking changes more visible. Additionally, upgrade instructions will be
updated adding the recommendation to --dry-run and check expected changes
before upgrades.

_Patch apply errors_

If a patch fails to apply, kubeadm command should exit with a descriptive
error.

The users would have to either remove the problematic patch or amend it.

_Confusion between the old and new customization features_

For a period of time (at least one release) the flags `--experimental-kustomize`
and `--experimental-patches` need to co-exist, which can confuse the users.

By marking `--experimental-kustomize` as deprecated, the flag will
be hidden from command --help` screens. The warning that is printing
when the deprecated flag is used should be amended to denote that
`--experimental-patches` should be used instead.

The `--experimental-patches` patches would apply _after_ the
`--experimental-kustomize` patches are applied.

## Design Details

<!--
This section should contain enough information that the specifics of your
change are understandable.  This may include API specs (though not always
required) or even code snippets.  If there's any ambiguity about HOW your
proposal will be implemented, this is the place to discuss them.
-->

### Code organization

The code organization of the new feature will be done in a similar
fashion to the existing Kustomize feature. The backend will be stored
in a new package called `patches`. Kubeadm commands would have minimal
exposure to the underlying backend and only call a single method that
accepts the directory where the user patches are.

### Patch formats

The proposed design is supporting the following patch formats similarly
to kubectl:
- Strategic Merge Patches (default):
  - Supported as `strategic` by `kubectl patch`.
  - Will be implemented using the `k8s.io/apimachinery` library.
- RFC6902 JSON patches:
  - Supported as `json` by `kubectl patch`.
  - Would be implemented using the `github.com/evanphx/json-patch`
  library.
- RFC7396 JSON merge patches:
  - Supported as `merge` by `kubectl patch`.
  - Would be implemented using the `github.com/evanphx/json-patch`
  library.

Note that kubectl already uses the same backends for the above
patch formats.

Patches defined in both YAML and JSON will be supported, with
the exception of `json` which must be in JSON only.

Conversion from YAML to JSON will be performed using the
`sigs.k8s.io/yaml` library.

### Patch file naming format

Once the user passes a folder with patches to a kubeadm command,
kubeadm must be able to determine what format each file is and what
manifest it is targeting.

Parsing the patch contents to determine the patch format is not possible,
because the formats `strategic` and `merge` are the same, while they do
have different apply mechanisms.

The proposal is to have the following naming format:
`componentname[suffix][+patchtype].{yaml|json}`

- `componentname` must be a known component name - e.g. `kube-apiserver`.
- `suffix` is optional and can be used to ensure an order when applying
patches from multiple files for the same component.
- `+patchtype` is optional and can be used to pass one of the supported
patch types. A missing value implies `+strategic`.
- `.{yaml|json}` defines a patch file extension. It is required and must
be either `.yaml` or `.json`.

Examples:
- `etcd.yaml`
- `etcd+json`
- `etcd+merge.json`
- `etcd2.yaml`
- `etcd2+merge.json`

Alpha-numeric order will be used when applying patches from multiple
files. This allows adjusting the order between patch types based on
`suffix`.

The format:
- Must be documented briefly in `--help` screens.
- Must be fully documented at the kubernetes/website (Beta graduation).
- Would allow for basic usage (using the default `strategic` type),
but also allow for advanced usage with patch ordering and the `json` type.
- Would allow having arbitrary files that are ignored in the patch folder -
e.g. having a `README.md`.

### Support for multiple patches per file

To support multiple patches per file, the proposal is to implement
reading of JSON and YAML multi-documents where the patches
are separated by `---\n`.

Multiple patches in a file will be applied top-first.

This extension is fairly simple to implement and would
allow even higher flexibility.

### Test Plan

<!--
**Note:** *Not required until targeted at a release.*

Consider the following in developing a test plan for this enhancement:
- Will there be e2e and integration tests, in addition to unit tests?
- How will it be tested in isolation vs with other components?

No need to outline all of the test cases, just the general strategy.  Anything
that would count as tricky in the implementation and anything particularly
challenging to test should be called out.

All code is expected to have adequate tests (eventually with coverage
expectations).  Please adhere to the [Kubernetes testing guidelines][testing-guidelines]
when drafting this test plan.

[testing-guidelines]: https://git.k8s.io/community/contributors/devel/sig-testing/testing.md
-->

The feature will be tested extensively by unit tests.

Similarly to the existing Kustomize feature, e2e tests will be added in the
same cycle when the new feature is added in its Alpha state.

For e2e tests the existing kubeadm testing infrastructure (i.e. Prow jobs with kinder)
will be used. The e2e tests will ensure that the kubeadm commands "init", "join" and
"upgrade" support the feature.

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
definition](https://kubernetes.io/docs/concepts/overview/kubernetes-api/#api-versioning),
or by redefining what graduation means.

In general, we try to use the same stages (alpha, beta, GA), regardless how the
functionality is accessed.

[maturity-levels]: https://git.k8s.io/community/contributors/devel/sig-architecture/api_changes.md#alpha-beta-and-stable-versions
[deprecation-policy]: https://kubernetes.io/docs/reference/using-api/deprecation-policy/

Below are some examples to consider, in addition to the aforementioned [maturity levels][maturity-levels].

#### Alpha -> Beta Graduation

- Gather feedback from developers and surveys
- Complete features A, B, C
- Tests are in Testgrid and linked in KEP

#### Beta -> GA Graduation

- N examples of real world usage
- N installs
- More rigorous forms of testing e.g., downgrade tests and scalability tests
- Allowing time for feedback

**Note:** Generally we also wait at least 2 releases between beta and
GA/stable, since there's no opportunity for user feedback, or even bug reports,
in back-to-back releases.

#### Removing a deprecated flag

- Announce deprecation and support policy of the existing flag
- Two versions passed since introducing the functionality which deprecates the flag (to address version skew)
- Address feedback on usage/changed behavior, provided on GitHub issues
- Deprecate the flag

**For non-optional features moving to GA, the graduation criteria must include [conformance tests].**

[conformance tests]: https://git.k8s.io/community/contributors/devel/sig-architecture/conformance-tests.md
-->

#### Alpha -> Beta Graduation

- No major bugs are present.
- No UX complains by the users are received.
- The feature was tested for at least one release in e2e tests and the tests are stable.
- The Alpha flag `--experimental-kustomize` and the underlying backend are removed.
- The Alpha flag `--experimental-patches` is renamed to `--patches`.
- The feature is documented at kubernetes/website under the kubeadm pages.
- The functionality is added as part of the kubeadm ComponentConfig under the
`NodeRegistrationOptions` structure. Requires synchronization with a kubeadm
ComponentConfig version increment, so could land in GA graduation.

#### Beta -> GA Graduation

- The feature is widely used and at least 2 cycles have passed since Beta.
- The `--patches` flag is removed (optional). Depends on user feedback
and the wider kubeadm plan for removing flags in favor of ComponentConfig.

### Upgrade / Downgrade Strategy

<!--
If applicable, how will the component be upgraded and downgraded? Make sure
this is in the test plan.

Consider the following in developing an upgrade/downgrade strategy for this
enhancement:
- What changes (in invocations, configurations, API use, etc.) is an existing
  cluster required to make on upgrade in order to keep previous behavior?
- What changes (in invocations, configurations, API use, etc.) is an existing
  cluster required to make on upgrade in order to make use of the enhancement?
-->

- Downgrade is not supported by kubeadm.
- For upgrades, similarly to the existing Kustomize feature, the new feature
will be supported during the execution of the `kubeadm upgrade` command.
- Once kubeadm generates its upgraded manifest files, the folder with
patches will be processed and the patches will be applied to the manifests.
- In case a patch fails to apply during upgrade, the user will be informed
to fix the issue manually.

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

Once introduced, the feature will support patching at minimum the `corev1.Pod`
object. If at some point the Kubernetes core API graduates to `v2`, kubeadm would
have to support patching both `corev2` and `corev1` for at least one release
cycle.

This is due to the fact kubeadm supports deploying Kubernetes `v1.YY` and `v1.YY-1`.
And if `corev2` is added in `v1.YY`, kubeadm must be able to handle `corev1`
that is supported in `v1.YY-1`.

If the patching mechanic one day supports patching different API types,
it must also apply similar skew strategy to those types.

## Production Readiness Review Questionnaire

<!--

Production readiness reviews are intended to ensure that features merging into
Kubernetes are observable, scalable and supportable, can be safely operated in
production environments, and can be disabled or rolled back in the event they
cause increased failures in production. See more in the PRR KEP at
https://git.k8s.io/enhancements/keps/sig-architecture/20190731-production-readiness-review-process.md

Production readiness review questionnaire must be completed for features in
v1.19 or later, but is non-blocking at this time. That is, approval is not
required in order to be in the release.

In some cases, the questions below should also have answers in `kep.yaml`. This
is to enable automation to verify the presence of the review, and reduce review
burden and latency.

The KEP must have a approver from the
[`prod-readiness-approvers`](http://git.k8s.io/enhancements/OWNERS_ALIASES)
team. Please reach out on the
[#prod-readiness](https://kubernetes.slack.com/archives/CPNHUMN74) channel if
you need any help or guidance.

-->

### Feature enablement and rollback

_This section must be completed when targeting alpha to a release._

* **How can this feature be enabled / disabled in a live cluster?**
  - [ ] Feature gate (also fill in values in `kep.yaml`)
    - Feature gate name:
    - Components depending on the feature gate:
  - [x] Other
    - Describe the mechanism:
      When a new kubeadm control-plane Node joins the cluster it can optionally
      apply custom patches to the static Pods that kubeadm configures.
    - Will enabling / disabling the feature require downtime of the control
      plane?
      No, unless the user created patches result in bad configuration and
      this is done on the primary control-plane Node - i.e. no other control-plane
      Nodes exist yet, which is not a "live cluster" yet.
    - Will enabling / disabling the feature require downtime or reprovisioning
      of a node? (Do not assume `Dynamic Kubelet Config` feature is enabled).
      Potentially, during a mutable `kubeadm upgrade` on a control-plane Node,
      if the user created patches result in bad configuration.

* **Does enabling the feature change any default behavior?**
  Any change of default behavior may be surprising to users or break existing
  automations, so be extremely careful here.

* **Can the feature be disabled once it has been enabled (i.e. can we rollback
  the enablement)?**
  Also set `rollback-supported` to `true` or `false` in `kep.yaml`.
  Describe the consequences on existing workloads (e.g. if this is runtime
  feature, can it break the existing applications?).
Yes. Once the patches are applied for a `kubeadm init|join|upgrade` command,
the user can decide to roll-back the changes which will result in a restart
of the control-plane component static Pods that were patches on this Node.

* **What happens if we reenable the feature if it was previously rolled back?**
This is supported by invoking kubeadm phases. The patches will re-apply
and the kubelet on the Node will pick up the changes and restart the
locally managed control-plane components.

* **Are there any tests for feature enablement/disablement?**
  The e2e framework does not currently support enabling and disabling feature
  gates. However, unit tests in each component dealing with managing data created
  with and without the feature are necessary. At the very least, think about
  conversion tests if API types are being modified.
The feature will include both e2e and unit tests as its Alpha graduation.
Feature gates are not used.

### Rollout, Upgrade and Rollback Planning

_This section must be completed when targeting beta graduation to a release._

* **How can a rollout fail? Can it impact already running workloads?**
  Try to be as paranoid as possible - e.g. what if some components will restart
  in the middle of rollout?
The feature can result in malfunctioning control-plane components, due to the
fact that it targets patching of static Pod manifests with user values.
The user is responsible for configuring the control-plane correctly.

* **What specific metrics should inform a rollback?**
Not applicable.

* **Were upgrade and rollback tested? Was upgrade->downgrade->upgrade path tested?**
  Describe manual testing that was done and the outcomes.
  Longer term, we may want to require automated upgrade/rollback tests, but we
  are missing a bunch of machinery and tooling and do that now.
The feature will include an e2e test for upgrade, but rollback is not planned for
testing.

* **Is the rollout accompanied by any deprecations and/or removals of features,
  APIs, fields of API types, flags, etc.?**
  Even if applying deprecation policies, they may still surprise some users.
The feature will deprecate an existing feature powered by the kubeadm flag
`--experimental-kustomize`. The existing feature will be removed after
one release, even if being Alpha grade. An `action-required` release note
will be filed just in case.

### Monitoring requirements

_This section must be completed when targeting beta graduation to a release._

* **How can an operator determine if the feature is in use by workloads?**
  Ideally, this should be a metrics. Operations against Kubernetes API (e.g.
  checking if there are objects with field X set) may be last resort. Avoid
  logs or events for this purpose.
Not directly applicable as the feature will be used to configure control-plane.

* **What are the SLIs (Service Level Indicators) an operator can use to
  determine the health of the service?**
  - [ ] Metrics
    - Metric name:
    - [Optional] Aggregation method:
    - Components exposing the metric:
  - [x] Other (treat as last resort)
    - Details: If a control-plane component on a Node fails to start,
    the admin must inspect their patches and debug the failure.

* **What are the reasonable SLOs (Service Level Objectives) for the above SLIs?**
  At the high-level this usually will be in the form of "high percentile of SLI
  per day <= X". It's impossible to provide a comprehensive guidance, but at the very
  high level (they needs more precise definitions) those may be things like:
  - per-day percentage of API calls finishing with 5XX errors <= 1%
  - 99% percentile over day of absolute value from (job creation time minus expected
    job creation time) for cron job <= 10%
  - 99,9% of /health requests per day finish with 200 code
Not applicable as the cluster-admin would see failures immediately after kubeadm
commands.

* **Are there any missing metrics that would be useful to have to improve
  observability if this feature?**
  Describe the metrics themselves and the reason they weren't added (e.g. cost,
  implementation difficulties, etc.).
Not applicable.

### Dependencies

_This section must be completed when targeting beta graduation to a release._

* **Does this feature depend on any specific services running in the cluster?**
  Think about both cluster-level services (e.g. metrics-server) as well
  as node-level agents (e.g. specific version of CRI). Focus on external or
  optional services that are needed. For example, if this feature depends on
  a cloud provider API, or upon an external software-defined storage or network
  control plane.

  For each of the dependencies fill in the following, thinking both about running user workloads
  and creating new ones, as well as about cluster-level services (e.g. DNS):
  - [Dependency name]
    - Usage description:
      - Impact of its outage on the feature:
      - Impact of its degraded performance or high error rates on the feature:
No, this feature customizes static Pod manifests for the control-plane.

### Scalability

_For alpha, this section is encouraged: reviewers should consider these questions
and attempt to answer them._

_For beta, this section is required: reviewers must answer these questions._

_For GA, this section is required: approvers should be able to confirms the
previous answers based on experience in the field._

* **Will enabling / using this feature result in any new API calls?**
  Describe them, providing:
  - API call type (e.g. PATCH pods)
  - estimated throughput
  - originating component(s) (e.g. Kubelet, Feature-X-controller)
  focusing mostly on:
  - components listing and/or watching resources they didn't before
  - API calls that may be triggered by changes of some Kubernetes resources
    (e.g. update of object X triggers new updates of object Y)
  - periodic API calls to reconcile state (e.g. periodic fetching state,
    heartbeats, leader election, etc.)
Not applicable.

* **Will enabling / using this feature result in introducing new API types?**
  Describe them providing:
  - API type
  - Supported number of objects per cluster
  - Supported number of objects per namespace (for namespace-scoped objects)
Yes. For the Beta graduation potentially, but that would be a kubeadm
ComponentConfig sub-type.

* **Will enabling / using this feature result in any new calls to cloud
  provider?**
No.

* **Will enabling / using this feature result in increasing size or count
  of the existing API objects?**
  Describe them providing:
  - API type(s):
  - Estimated increase in size: (e.g. new annotation of size 32B)
  - Estimated amount of new objects: (e.g. new Object X for every existing Pod)
No.

* **Will enabling / using this feature result in increasing time taken by any
  operations covered by [existing SLIs/SLOs][]?**
  Think about adding additional work or introducing new steps in between
  (e.g. need to do X to start a container), etc. Please describe the details.
No, unless the user patches the control-plane static Pods with undesired
configuration.

* **Will enabling / using this feature result in non-negligible increase of
  resource usage (CPU, RAM, disk, IO, ...) in any components?**
  Things to keep in mind include: additional in-memory state, additional
  non-trivial computations, excessive access to disks (including increased log
  volume), significant amount of data send and/or received over network, etc.
  This through this both in small and large cases, again with respect to the
  [supported limits][].
No, unless the user patches the control-plane static Pods with undesired
configuration.

### Troubleshooting

Troubleshooting section serves the `Playbook` role as of now. We may consider
splitting it into a dedicated `Playbook` document (potentially with some monitoring
details). For now we leave it here though.

_This section must be completed when targeting beta graduation to a release._

* **How does this feature react if the API server and/or etcd is unavailable?**
This feature can be used to configure an etcd or API server instance, by patching
their static Pods (considering etcd is run as a static Pod too).

* **What are other known failure modes?**
  For each of them fill in the following information by copying the below template:
  - [Failure mode brief description]
    - Detection: How can it be detected via metrics? Stated another way:
      how can an operator troubleshoot without logging into a master or worker node?
      Not applicable.
    - Mitigations: What can be done to stop the bleeding, especially for already
      running user workloads?
      The user needs to debug what patch is causing the failure and rollback the change.
    - Diagnostics: What are the useful log messages and their required logging
      levels that could help debugging the issue?
      Patches that failed to apply will result in overall `kubeadm` command failures,
      causing an `exit > 0` status.
      Not required until feature graduated to Beta.
    - Testing: Are there any tests for failure mode? If not describe why.
      Failure mode testing for patches is out-of-scope for e2e tests, but will be tested
      in unit tests.

* **What steps should be taken if SLOs are not being met to determine the problem?**
Not applicable.

[supported limits]: https://git.k8s.io/community//sig-scalability/configs-and-limits/thresholds.md
[existing SLIs/SLOs]: https://git.k8s.io/community/sig-scalability/slos/slos.md#kubernetes-slisslos

## Implementation History

<!--
Major milestones in the life cycle of a KEP should be tracked in this section.
Major milestones might include
- the `Summary` and `Motivation` sections being merged signaling SIG acceptance
- the `Proposal` section being merged signaling agreement on a proposed design
- the date implementation started
- the first Kubernetes release where an initial version of the KEP was available
- the version of Kubernetes where the KEP graduated to general availability
- when the KEP was retired or superseded
-->

- 2020-05-04: initial provisional KEP created
- 2020-05-14: addressed feedback, filled PRR questionnaire, KEP marked as implementable
- 2021-09-09: marked the feature as graduated to Beta. This was done as part of
the kubeadm v1beta3 API work. The actions under "Alpha -> Beta" are completed.

## Drawbacks

<!--
Why should this KEP _not_ be implemented?
-->

None.

## Alternatives

<!--
What other approaches did you consider and why did you rule them out?  These do
not need to be as detailed as the proposal, but should include enough
information to express the idea and why it was not acceptable.
-->

One alternative is using the existing Kustomize feature, yet as pointed out
in [Motivation](#motivation) the kubeadm maintainers have reservations
for using this approach.

Another alternative is supporting instance specific configuration that is
persisted in the kubeadm created cluster. Yet, to fully replace the
flexibility that patches enable, it requires that the persisted configuration
is stored in the underlying low-level format, such as `corev1.Pod`.

It is not clear whether the kubeadm ComponentConfig will support such
alternatives in the future.

## Infrastructure Needed (optional)

<!--
Use this section if you need things from the project/SIG.  Examples include a
new subproject, repos requested, github details.  Listing these here allows a
SIG to get the process for these resources started right away.
-->

None.
