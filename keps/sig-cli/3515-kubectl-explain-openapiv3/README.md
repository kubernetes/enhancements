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
# KEP-3515: OpenAPI v3 for kubectl explain

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
  - [OpenAPI v3 is a richer API description than OpenAPI v2](#openapi-v3-is-a-richer-api-description-than-openapi-v2)
  - [CRD schemas expressed as OpenAPI v2 are lossy](#crd-schemas-expressed-as-openapi-v2-are-lossy)
  - [Goals](#goals)
  - [Non-Goals](#non-goals)
- [Proposal](#proposal)
  - [Basic Usage](#basic-usage)
  - [Built-in Template Options](#built-in-template-options)
    - [Plaintext](#plaintext)
    - [OpenAPIV3 (raw json)](#openapiv3-raw-json)
    - [HTML](#html)
    - [Markdown](#markdown)
  - [Risks and Mitigations](#risks-and-mitigations)
    - [OpenAPI V3 Not Available](#openapi-v3-not-available)
      - [Risk](#risk)
      - [Mitigation](#mitigation)
    - [OpenAPI serialization time](#openapi-serialization-time)
      - [Risk](#risk-1)
      - [Mitigation](#mitigation-1)
- [Design Details](#design-details)
    - [Current High-level Approach](#current-high-level-approach)
    - [Proposed High-level Approach](#proposed-high-level-approach)
  - [Template rendering](#template-rendering)
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
  - [Implement proto.Models for OpenAPI V3 data](#implement-protomodels-for-openapi-v3-data)
  - [Custom User Templates](#custom-user-templates)
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

This KEP proposes an enhancement to kubectl explain:

1. Switch data source from OpenAPI v2 to OpenAPI v3
2. Replace the hand-written `kubectl explain` printer with a go/template implementation.

## Motivation

<!--
This section is for explicitly listing the motivation, goals, and non-goals of
this KEP.  Describe why the change is important and the benefits to users. The
motivation section can optionally provide links to [experience reports] to
demonstrate the interest in a KEP within the wider Kubernetes community.

[experience reports]: https://github.com/golang/go/wiki/ExperienceReports
-->

### OpenAPI v3 is a richer API description than OpenAPI v2

OpenAPI v3 support in Kubernetes is currently beta since version 1.24.
OpenAPI V3 is a richer representation of the kubernetes API to our users, who have been asking for visibility
into things like:

1. nullable
2. default
3. validation fields like oneOf, anyOf, etc.

To show each of these additional data points by themselves is a strong reason
to switch to using OpenAPI v3.


### CRD schemas expressed as OpenAPI v2 are lossy

Today CRDs specify their schemas in OpenAPI v3 format. To serve the `/openapi/v2`
document used today by kubectl, there is an expensive conversion from the v3 down
to v2 format.

This process is [very lossy](https://github.com/kubernetes/kubernetes/blob/6e0de20fbb4c127d2e45c7a22347c08545fc7a86/staging/src/k8s.io/apiextensions-apiserver/pkg/controller/openapi/v2/conversion.go#L56-L66), so `kubectl explain` when used against CRDs
making use of v3 features does not have a good experience with inaccurate information, or fields removed altogther.

This transformation causes bugs, for example, when attempting to `explain` a field
that is `nullable`, kubectl instead shows nothing, due to the lossy conversion
wiping nullable fields.

### Goals

1. Provide the new richer type information specified by OpenAPI v3 within kubectl explain
2. Have a more maintainable `text/template` based approach to printing
3. Fallback to old `explain` implementation if cluster does not expose OpenAPI v3 data.
4. Provide multiple new output formats for kubectl explain:
    * human-readable plaintext
    * markdown
    * maybe others
5. (Optional?) Allow users to specify their own templates for use with kubectl
  explain (there may be interesting use cases for this)
6. Improve discoverability of API Resources and endpoints, and provide a platform
for richer information to be included in the future.

### Non-Goals

1. "Fix" openapi v3 to openapi v2 conversion
  This is a non-goal for two reasons:
    * These formats are not compatible, and there WILL be data loss and inaccuracy
    * This negates the benefits of using OpenAPI v3 for the richer type information
2. Provide general-purpose OpenAPI visualization.


## Proposal


### Basic Usage

The following user experience should be possible with `kubectl explain`

```shell
kubectl explain pods.spec
```

Output should be familiar to users of today's `kubectl explain`, except new
information from the OpenAPI v3 spec is now populated.

Note: Feature during development will be gated by an experimental flag. The commands
shown here elide the experimental flag for clarity.

### Built-in Template Options
#### Plaintext


```shell
kubectl explain pods
```
or

```shell
kubectl explain pods --output plaintext
```

The plaintext output format is the default and should be crafted to be as close
as the existing `explain` output in use before this KEP.

#### OpenAPIV3 (raw json)

```shell
kubectl explain pods --output openapiv3
```

To get raw OpenAPI v3 data for a certain resource today involves:
1.) setting up kubectl proxy
2.) fetching the correct path at `/openapi/v3/<group>/<version>`
3.) filtering out unwanted results

This command is useful not only for its convenience, but also other visualizations
may be built upon the raw output if we opt not to support a first-class custom
template solution in the future.

#### HTML

```shell
kubectl explain pods --output html
```

Similarly to [godoc](https://pkg.go.dev), we suggest to provide a searchable,
navigable, generated webpage for the kubernetes types of whatever cluster kubectl
is talking to.

Only the fields selected in the command line (and their subfields' types, etc) 
will be included in the resultant page.

If user types `kubectl explain --output html` with no specific target, then all types
in the cluster are included.

#### Markdown

```shell
kubectl explain pods --output md
```

When using the `md` template, a markdown document is printed to stdout, so it
might be saved and used for a documentation website, for example.

Similarly to `html` output, only the fields selected in the command line 
(and their subfields' types, etc) will be included in the resultant page.

If user types `kubectl explain --output md` with no specific target, then all types
in the cluster are included.

### Risks and Mitigations

#### OpenAPI V3 Not Available

##### Risk

OpenAPI v3 data is not available in the current cluster.

##### Mitigation

###### If the user does not provide an --output argument

While this feature is not GA, this case should fallback to the old OpenAPI v2
`kubectl explain` implementation if the server responds with `404` for OpenAPIV3 data.

Once this feature is GA, then `OpenAPIV3` should be available everywhere
so this is not a concern. If a user uses the feature against such a cluster without
OpenAPIV3 after this KEP is GA, an error will be shown.

###### If the user does provide an --output argument

If a user specifies an `--output` argument and the server 404's attempting to
fetch the correct openapi version for the template, a new error message should
be thrown to the effect of: `server missing openapi data for version: %v.%v.%v`.

Internal templates should strive to support the latest OpenAPI version enabled
by default by versions of kubernetes within their skew. With that policy, templates
will always render with the latest spec-version of the data, if it is available.

Other network errors should be handled using normal kubectl error handling.


#### OpenAPI serialization time
##### Risk

Today there is no interactive-speed way to deserialize protobuf or JSON openapi
v3 data into the kube-openapi format.

##### Mitigation

There has been recent progress in this area. To unmarshal kube-OpenAPI v3 is now able
to be done in a performant enough way to do it in the CLI. This KEP's beta release
should be blocked on the merging of this optimization.

## Design Details

#### Current High-level Approach

1. User types `kubectl explain pods`
2. kubectl resolves 'pods' to GVR core v1 pods using cluster discovery information
3. kubectl resolves GVR to its GVK using restmapper
4. kubectl fetches `/openapi/v2` as protobuf
5. kubectl parses the protobuf into `gnostic_v2.Document`
6. kubectl converts `gnostic_v2.Document` into `proto.Models`
7. kubectl searches the document's `Definitions` for a schema with the
extension `x-kubernetes-group-version-kind` matching the interested GVK
8. If a field path was used, kubectl traverses the definition's fields to the subschema
specified by the user's path.
9. kubectl renders the definition using its hardcoded printer
10. If `--recursive` was used, repeat step 9 for the transitive closure of
  object-typed fields of the top-level object. Concat the results together.

#### Proposed High-level Approach

1. User types `kubectl explain pods`
2. kubectl resolves 'pods' to GVR core v1 pods using cluster discovery information
3. kubectl fetches `/openapi/v3/<group>/<version>`
4. kubectl parses the result as kube-openapi spec3
5. kubectl locates the schema of the return type for the Path `/apis/<group>/<version>/<resource>` in kube-openapi
6. If a field path was used, kubectl traverses the definition's fields to the subschema
specified by the user's path.
8. kubectl renders the type using its built-in template for human-readable plaintext
9. If `--recursive` was used, repeat step 8 for the transitive closure of object-typed fields of the top-level object. Concat the results together.

### Template rendering

Go's text/template will be used due to its familiarity, stability, and virtue of being in stdlib.

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

- `k8s.io/kubectl/pkg/explain`: `09/29/2022`-`75.6`

##### Integration tests

<!--
This question should be filled when targeting a release.
For Alpha, describe what tests will be added to ensure proper quality of the enhancement.

For Beta and GA, add links to added tests together with links to k8s-triage for those tests:
https://storage.googleapis.com/k8s-triage/index.html
-->

- <test>: <link to test coverage>

Tests should include

- Expected Output tests
- Show correct OpenAPI v3 endpoints are hit
- Tests that show default/nullability information is being included in plaintext output
- Tests that update the backing openapi in between calls to explain

##### e2e tests

<!--
This question should be filled when targeting a release.
For Alpha, describe what tests will be added to ensure proper quality of the enhancement.

For Beta and GA, add links to added tests together with links to k8s-triage for those tests:
https://storage.googleapis.com/k8s-triage/index.html

We expect no non-infra related flakes in the last month as a GA graduation criteria.
-->

Existing e2e tests should be adapted for the new system.
E2E test that shows every definition in OpenAPI document can be retrieved via explain


- <test>: <link to test coverage>

### Graduation Criteria

Defined using feature gate

#### Alpha

- Feature implemented behind a command line flag `--experimental-output`
- `--experimental-output` flag added
- Existing explain tests are working or adapted for new implementation
- Plaintext output roughly matches explain output
- OpenAPIV3 (raw json) output implemented
- HTML and MD outputs are not target for alpha

#### Beta

- md output implemented (or dropped from design due to continued debate)
  - Table of contents all GVKs grouped by Group then Version.
  - Section for each individual GVK
  - All types hyperlink to specific section
- basic html output  (or dropped from design due to continued debate)
  - Table of contents all GVKs grouped by Group then Version.
  - Page for each individual GVK.
  - All types hyperlink to their specific page
  - Searchable by name, description, field name.
- kube-openAPI v3 JSON deserialization is optimized to take less than 150ms on
  most machines
- OpenAPI V3 is enabled by default on at least one version within kubectl's support window
- Experimental flag is removed/made on by default (thus openapi v3 will always be tried first)
- Old `kubectl explain` implementation for OpenAPI v2 remains as a fallback if v3 is unavailable
(this policy should stand only until kubectl's version skew includes apiserver versions which enabled OpenAPI V3 by default). 

#### GA

- `--experimental-output` renamed to `--output`
- All kube-apiserver releases within version skew of kubectl should have OpenAPIV3 on by default
- Old `kubectl explain` implementation is removed, as is support for OpenAPIV2-backed `kubectl explain`

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

N/A

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

This feature only requires the target cluster has enabled The OpenAPIV3 feature.

OpenAPIV3 is Beta as of Kubernetes 1.24. This feature should not be on-by-default
until it is GA.

Users of the `--output` flag who attempt to use it against a cluster for which
OpenAPI v3 is not enabled will be shown an error informing them of missing openapi
version upon 404.

Built-in templates supported by kubectl should aim to support any OpenAPI
version which is GA.

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

- [ ] Feature gate (also fill in values in `kep.yaml`)
  - Feature gate name:
  - Components depending on the feature gate:
- [x] Other
  - Describe the mechanism: --experimental-output flag usage
    (to be renamed to --output when feature is no longer experimental)
  - Will enabling / disabling the feature require downtime of the control
    plane? No
  - Will enabling / disabling the feature require downtime or reprovisioning
    of a node? (Do not assume `Dynamic Kubelet Config` feature is enabled). No

###### Does enabling the feature change any default behavior?

<!--
Any change of default behavior may be surprising to users or break existing
automations, so be extremely careful here.
-->

Enabling the feature changes the data source of `kubectl explain` to use openapiv3.
The output optimally should be familiar to users, who may be delighted to see new
information populated.

###### Can the feature be disabled once it has been enabled (i.e. can we roll back the enablement)?

Until the feature is stable it will only be enabled when the `--experimental-output` flag is used.
It has no persistent effect on data that is viewewd.

###### What happens if we reenable the feature if it was previously rolled back?

There is no persistence to using the feature. It is only used for viewing data.

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

###### What specific metrics should inform a rollback?

<!--
What signals should users be paying attention to when the feature is young
that might indicate a serious problem?
-->

###### Were upgrade and rollback tested? Was the upgrade->downgrade->upgrade path tested?

<!--
Describe manual testing that was done and the outcomes.
Longer term, we may want to require automated upgrade/rollback tests, but we
are missing a bunch of machinery and tooling and can't do that now.
-->

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

###### How can someone using this feature know that it is working for their instance?

```shell
kubectl explain pods --output openapiv3
```

User should see OpenAPI v3 JSON Schema for `pods` type printed to console.

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

###### Are there any missing metrics that would be useful to have to improve observability of this feature?

<!--
Describe the metrics themselves and the reasons why they weren't added (e.g., cost,
implementation difficulties, etc.).
-->

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

To reap the benefits of this feature, OpenAPI v3 is required, however OpenAPI v2
data can be used as a fallback.

### Scalability

<!--
For alpha, this section is encouraged: reviewers should consider these questions
and attempt to answer them.

For beta, this section is required: reviewers must answer these questions.

For GA, this section is required: approvers should be able to confirm the
previous answers based on experience in the field.
-->

###### Will enabling / using this feature result in any new API calls?

Yes, up feature replaces a single GET of `/openapi/v2` which returns a large (megabytes)
openapi document for all types with a more targeted call to `/openapi/v3/<group>/<version>`

The `/openapi/v3/<group>/<version>` endpoint implements E-Tag caching so that if the document has
not changed the server incurs a cheap, almost negligible cost to serving the request.

The document returned by calls to `/openapi/v3/...` is expected to be far smaller
than the megabytes-scale openapi v2 document, since it only includes information
for a single group-version.

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

###### Will enabling / using this feature result in introducing new API types?

No.

###### Will enabling / using this feature result in any new calls to the cloud provider?

No.

###### Will enabling / using this feature result in increasing size or count of the existing API objects?

No.

###### Will enabling / using this feature result in increasing time taken by any operations covered by existing SLIs/SLOs?

No.

###### Will enabling / using this feature result in non-negligible increase of resource usage (CPU, RAM, disk, IO, ...) in any components?

No, would expect generally same amount of resource usage for kubectl.

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

Using kubectl's normal error handling. There is no lasting effect to data or the
user.

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

## Drawbacks

<!--
Why should this KEP _not_ be implemented?
-->

## Alternatives

### Implement proto.Models for OpenAPI V3 data

The current hard-coded printer is capable of printing any objects in `proto.Models` form.

[We already have a way to express OpenAPI v3 data as `proto.Models`, so this can be
seen as a path of least resistance for plugging OpenAPI v3 into `kubectl explain`.

This approach is undesirable for a few different reasons:

1.) We would like to update the explain printer to include new OpenAPI v3 information,
the current design makes that time consuming and not maintainable.

2.) API-Machinery has desire to deprecate `proto.Models`. We see`proto.Models`
conversion as unnecessary and costly buraucracy, that contributes to high
OpenAPI overhead. We are seeking to deprecate the type in favor of the
kube-openapi types for future usage.

### Custom User Templates
Users might also like to be able to specify a path to a custom template file for
the resource information to be written to:

human-readable plaintext form:
```shell
kubectl explain pods --template /path/to/template.tmpl
```

Since the API surface for this sort of feature remains very unclear and will likely
be very unstable, this sort of feature should be delayed until the internal
templates have proven the API surface to be used. To do otherwise would risk
breaking user's templates.
