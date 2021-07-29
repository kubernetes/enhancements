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
# KEP-NNNN: CRD Validation Expressions

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
  - [Descriptive, self contained CRDs](#descriptive-self-contained-crds)
  - [Webhooks: Development Complexity](#webhooks-development-complexity)
  - [Webhooks: Operational Complexity](#webhooks-operational-complexity)
  - [Overview of existing validation](#overview-of-existing-validation)
  - [Overview of ecosystem](#overview-of-ecosystem)
  - [Goals](#goals)
  - [Non-Goals](#non-goals)
- [Proposal](#proposal)
    - [Field paths and field patterns](#field-paths-and-field-patterns)
    - [Expression lifecycle](#expression-lifecycle)
    - [Function library](#function-library)
  - [User Stories](#user-stories)
  - [Notes/Constraints/Caveats (Optional)](#notesconstraintscaveats-optional)
  - [Risks and Mitigations](#risks-and-mitigations)
    - [Accidental misuse](#accidental-misuse)
    - [Malicious use](#malicious-use)
  - [Future Plan](#future-plan)
    - [Defaulting](#defaulting)
    - [Conversion](#conversion)
- [Design Details](#design-details)
  - [Type Checking](#type-checking)
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
  - [Rego](#rego)
  - [Expr](#expr)
  - [WebAssembly](#webassembly)
  - [Build our own](#build-our-own)
  - [Make it easier to validate CRDs using webhooks](#make-it-easier-to-validate-crds-using-webhooks)
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

CRDs need direct support for non-trivial validation.  While admission webhooks do support
CRDs validation, they significantly complicate the development and
operability of CRDs.

This KEP proposes that an inline expression language be integrated directly into
CRDs such that a much larger portion of validation use cases can
be solved without the use of webhooks. Support for defaulting and CRD conversions via
the expression language will also be added in the future.

This KEP proposes the use the adoption of [Common Expression Language
(CEL)](https://github.com/google/cel-go). It is sufficiently lightweight and
safe to be run directly in the kube-apiserver, has a straight-forward and
unsurprising grammar, and supports pre-parsing and typechecking of expressions,
allowing syntax and type errors to be caught at CRD registration time.

<<[UNRESOLVED @jpbetz @cici37]>>
There are alternatives to CEL that we have not yet ruled out. We are focusing
on CEL because it appears to meet many of our needs, but we still need to
verify that it:

(1) sufficiently covers the validation needs of the community (need more use cases)
(2) integrates well with kubernetes (type checking, support for associative lists...)
(3) has the performance and sandbox properties it claims
<<[/UNRESOLVED]>>

## Motivation

### Descriptive, self contained CRDs

This KEP will make CRDs more self-contained. Instead of having
validation rules coded into webhooks that must be
registered and upgraded independent of a CRD, the rules will be contained within
the CRD object definition, making them easier to author and introspect by
cluster administrators and users, and eliminating version skew issues that can
happen between a CRD and webhook since they can be registered and
upgraded/rolled-back independently.

### Webhooks: Development Complexity

Introducing a production grade webhook is a substantial development task.
Beyond authoring the actual core logic that a webhook must perform, the webhook
must be instrumented for monitoring and alerting and integrated with the
packaging/releases processes for the environments it will be run it.

The developer must also carefully consider the upgrade and rollback ordering
between the webhook and CRD.

### Webhooks: Operational Complexity

Admission webhooks are part of the critical serving path of the kube-apiserver.
Admission webhooks add latency to requests, and large numbers of webhooks
can cause, or contribute to, request timeouts being exceeded.

Webhooks must either be configured as `FailPolicy.Fail` or `FailPolicy.Ignore`. If
`FailPolicy.Ignore` is used, there is potential for requests skip the webhook and
be admitted. If `FailPolicy.Fail` is used, a webhook outage can result in a
localized or widespread Kubernetes control plane outage depending on which
objects the webhook is configured to intercept.

### Overview of existing validation

CRDs currently support two major categories of validation:

- [CRD structural
schemas](https://kubernetes.io/docs/tasks/extend-kubernetes/custom-resources/custom-resource-definitions/#specifying-a-structural-schema):
Provide type checking of custom resources against schemas.

- OpenAPIv3 validation rules: Provide regex ('pattern' property), range
limits ('minimum' and 'maximum' properties) on individual fields and size limits
on maps and lists ('minItems', 'maxItems').

In addition, the API Expression WG is working on KEPs that would improve CRD validation:

- OpenAPIv3 'formats' which could (and I believe should) be leveraged by
Kubernetes to handle validation of string fields for cases where regex is poorly
suited or insufficient.
- Immutability
- Unions

These improvements are largely complementary to expression support and either
are (or should be) addressed by in separate KEPs.


### Overview of ecosystem

<<[UNRESOLVED @cici37 @leilajal]>>
TODO
<<[/UNRESOLVED]>>

### Goals

- Make CRDs more self-contained and declarative
- Simplify CRD development
- Simplify CRD operations

### Non-Goals

- Support for validation formats, immutability or unions. These are all valuable improvements
  but can be addressed orthogonally in separate KEPs.
- Eliminate the need for webhooks entirely. Webhooks will still be needed for
  some use cases. For example, if a validation check requires making a network
  request to some other system, it will still need to be implemented in a webhook.
- Change how Kubernetes built-in types are validated, defaulted and converted.

## Proposal

[Common Expression Language (CEL)](https://github.com/google/cel-go) is an
excellent supplement to the current validation mechanism because it is sufficiently expressive to
satisfy a large set of remaining uses cases that none of the above can solve.
For example, cross-field validation use cases can only be solved using
expressions or webhooks.

`x-kubernetes-validator` extension will be added to CRD structural schemas to allow CEL validation expressions.

```yaml
apiVersion: apiextensions.k8s.io/v1
kind: CustomResourceDefinition
...
  schema:
    openAPIV3Schema:
      type: object
      properties:
        spec:
          x-kubernetes-validator: 
            - rule: "minReplicas <= maxReplicas"
              message: "minReplicas cannot be larger than maxReplicas"
          type: object
          properties:
            minReplicas:
              type: integer
            maxReplicas:
              type: integer
```

- Each validator may have multiple validation rules.

- Each validation rule has an optional 'message' field for the error message that
will be surfaced when the validation rule evaluates to false.

- The validator will be scoped to the location of the `x-kubernetes-validator`
extension in the schema. In the above example, the validator is scoped to the
'spec' field.

- For OpenAPIv3 object types, the expression will have direct access to all the
fields of the object the validator is scoped to.
  
- For OpenAPIv3 scalar types (integer, string & boolean), the expression will have
access to the scalar data element the validator is scoped to.

- For OpenAPIv3 list and map types, the expression will have access to the data
element of the list or map.

<<[UNRESOLVED @cici37]>>
Should the message also be an expression to allow for some basic variable
substitution?
<<[/UNRESOLVED]>>

<<[UNRESOLVED @cici37]>>
Should a 'type' field also be required? If it is not 'cel', the validator
could be skipped to future proof against addition of other ways to validate in
the future, or to allow 3rd party validators to have a way to inline their
validation rules in CRDs.
<<[/UNRESOLVED]>>

#### Field paths and field patterns

A field path is a patch to a single node in the data tree. I.e. it specifies the
exact indices of the list items and the keys of map entries it traverses.

A field *pattern* is a path to all nodes in the data tree that match the pattern. I.e.
it may wildcard list item and map keys.

`this` represents the field `x-kubernetes-validator` scopes to. In above example, the validator is scoped to the
`spec` field. So `this.properties.minReplicas` represents to `minReplicas` field which the rule will apply.
Having `this` will be beneficial for setting up rules like validating if there is a specific field available. 
```yaml
spec:
          x-kubernetes-validator: 
            - rule: this.hasField("properties")
              message: "there has to be properties field under spec"
          type: object
          properties:
```

#### Expression lifecycle

When CRDs are written to the kube-apiserver, all expressions will be [parsed and
typechecked](https://github.com/google/cel-go#parse-and-check) and the resulting
program will be cached for later evaluation (CEL evaluation is thread-safe and
side-effect free). Any parsing or type checking errors will cause the CRD write
to fail with a descriptive error.

#### Function library

The function library available to expressions can be augmented using [extension
functions](https://github.com/google/cel-spec/blob/master/doc/langdef.md#extension-functions).

List of functions to include for the initial release:
- Equality and Ordering
- Regular Expressions
- Some Standard Definitions


Considerations:
- The functions will become VERY difficult to change as this feature matures. We
  should limit ourselves initially to functions that we have a high level of
  confidence will not need to be changed or rethought.
- Support kubernetes specific concepts, like accessing associative lists by key
  may be needed, we should review those cases carefully.


### User Stories

<<[UNRESOLVED @cici37 @leilajal @jpbetz]>>
Find validation use cases representative of community needs and show how they can
be handled using CEL.
<<[/UNRESOLVED]>>

### Notes/Constraints/Caveats (Optional)

<!--
What are the caveats to the proposal?
What are some important details that didn't come across above?
Go in to as much detail as necessary here.
This might be a good place to talk about core concepts and how they relate.
-->

### Risks and Mitigations

#### Accidental misuse

Break the control plane by consuming excessive CPU and/or memory the api-server.

Mitigation: CEL is specifically designed to constrain the running time of expressions
and to limit the memory utilization. We will perform a security review of CEL and how
it is integrated.

#### Malicious use

Breaking out of the sandbox to run untrusted code in the apiserver or exfiltrate data.

Mitigation: CEL is designed to sandbox code execution. We will perform a security review of CEL and how
it is integrated.

<<[UNRESOLVED @jpbetz]>>
Find a security reviewer for CEL.
<<[/UNRESOLVED]>>

### Future Plan

#### Defaulting

The `x-kubernetes-default` extension will be used. The location of the defaulter
determines the scope of defaulter.

The 'field' specifies which field is to be defaulted using a field path that is
relative to the scope root. The defaulter will only be run if 'field' is unset, 
and the result of the expression must match the type of the field to be
defaulted.

```yaml
apiVersion: apiextensions.k8s.io/v1
kind: CustomResourceDefinition
...
  schema:
    openAPIV3Schema:
      type: object
      properties:
        spec:
          x-kubernetes-default:
		    - rule: "!has(spec.type) && spec.Type == 'LoadBalancer' ? 'ServiceExternalTrafficPolicy' : null"
			  field: spec.externalTrafficPolicy
```

- Each defaulter may have multiple rules.

- The 'field' is optional and defaults to the scope if not explicitly set.

If the expression evaluates to null, the field is left unset.
TODO: Any downsides to using null this way?

#### Conversion

Conversion rules will be made available via a ConversionRules strategy. A set of
rules can be provided for each supported source/destination version pair. Each rule will
specify which field it sets using a field path. A 'scope' field *pattern* may
also be used specify all locations in the object that the conversion rule should
be applied.

```yaml
apiVersion: apiextensions.k8s.io/v1
kind: CustomResourceDefinition
...
 conversion:
    strategy: ConversionRules
	converters:
		- fromVersion: v1beta1
		  toVersion: v1
		  rules:
		    - field: spec.xyz.podIPs
			  rule: "has(spec.xyz.podIP) ? [spec.xyz.podIP] : []"
		- fromVersion: v1
		  toVersion: v1beta1
		  rules:
		    - field: xyz.podIP
		      scope: spec
			  rule: "size(xyz.podIPs) > 0 ? xyz.podIPs[0] : null"
```

Expressions that return object types will overwrite all fields that they return and
leave all other fields to be auto-converted.

The 'scope' is optional and defaults to the object root.

The 'field' is optional and defaults to the scope? TODO: is it better to make it required for conversion?

If the expression evaluates to null, the field is left unset.

TODO: demonstrate writing fields into annotations for round trip
TODO: demonstrate string <-> structured (label selector example & ServicePort example)


## Design Details

<!--
This section should contain enough information that the specifics of your
change are understandable. This may include API specs (though not always
required) or even code snippets. If there's any ambiguity about HOW your
proposal will be implemented, this is the place to discuss them.
-->

### Type Checking

CEL type checking requires "declarations" be registered for any types that are to
be type checked.  In our case, the type information of interest is the CRD's
structural schemas. So we need to translate structural schemas to declarations.

"declarations" are registered via the go-genproto "checked" package
(https://github.com/googleapis/go-genproto/blob/master/googleapis/api/expr/v1alpha1/checked.pb.go).

(https://github.com/googleapis/googleapis/blob/master/google/api/expr/v1alpha1/checked.proto).

There are a couple alternative ways to do this. Ideally, we could be able be
able to both dereference into objects and construct objects in a typesafe way.
In otder to construct objects in a typesafe way we need to be able to represent
the structural schema types in CEL, e.g. "v1beta1.Foo{fieldname: value}", this
is complicated by the way CEL relies on protobuf types.

### Test Plan

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
-->

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
-->

###### Can the feature be disabled once it has been enabled (i.e. can we roll back the enablement)?

<!--
Describe the consequences on existing workloads (e.g., if this is a runtime
feature, can it break the existing applications?).

NOTE: Also set `disable-supported` to `true` or `false` in `kep.yaml`.
-->

###### What happens if we reenable the feature if it was previously rolled back?

###### Are there any tests for feature enablement/disablement?

<!--
The e2e framework does not currently support enabling or disabling feature
gates. However, unit tests in each component dealing with managing data, created
with and without the feature, are necessary. At the very least, think about
conversion tests if API types are being modified.
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
-->

###### How can an operator determine if the feature is in use by workloads?

<!--
Ideally, this should be a metric. Operations against the Kubernetes API (e.g.,
checking if there are objects with field X set) may be a last resort. Avoid
logs or events for this purpose.
-->

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

###### Will enabling / using this feature result in introducing new API types?

<!--
Describe them, providing:
  - API type
  - Supported number of objects per cluster
  - Supported number of objects per namespace (for namespace-scoped objects)
-->

###### Will enabling / using this feature result in any new calls to the cloud provider?

<!--
Describe them, providing:
  - Which API(s):
  - Estimated increase:
-->

###### Will enabling / using this feature result in increasing size or count of the existing API objects?

<!--
Describe them, providing:
  - API type(s):
  - Estimated increase in size: (e.g., new annotation of size 32B)
  - Estimated amount of new objects: (e.g., new Object X for every existing Pod)
-->

###### Will enabling / using this feature result in increasing time taken by any operations covered by existing SLIs/SLOs?

<!--
Look at the [existing SLIs/SLOs].

Think about adding additional work or introducing new steps in between
(e.g. need to do X to start a container), etc. Please describe the details.

[existing SLIs/SLOs]: https://git.k8s.io/community/sig-scalability/slos/slos.md#kubernetes-slisslos
-->

###### Will enabling / using this feature result in non-negligible increase of resource usage (CPU, RAM, disk, IO, ...) in any components?

<!--
Things to keep in mind include: additional in-memory state, additional
non-trivial computations, excessive access to disks (including increased log
volume), significant amount of data sent and/or received over network, etc.
This through this both in small and large cases, again with respect to the
[supported limits].

[supported limits]: https://git.k8s.io/community//sig-scalability/configs-and-limits/thresholds.md
-->

### Troubleshooting

<!--
This section must be completed when targeting beta to a release.

The Troubleshooting section currently serves the `Playbook` role. We may consider
splitting it into a dedicated `Playbook` document (potentially with some monitoring
details). For now, we leave it here.
-->

###### How does this feature react if the API server and/or etcd is unavailable?

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

### Rego

See Open Policy Agent (https://github.com/open-policy-agent/opa/tree/main/rego).
The syntax is more extensive than CEL and is designed specifically to work well
with kubernetes objects. It allows larger, multi-line programs and 
includes a package and module system. It does not offer the same sandbox constraints
as CEL, nor does it type check code.

### Expr

See github.com/antonmedv/expr. Has many similarities to CEL: type checking, minimalist syntax,
good performance and sandboxing properties.

This is used by the argo.

### WebAssembly

We looked closely at WebAssembly and created a [proof-of-concept implementation](https://github.com/jpbetz/omni-webhook/blob/main/validators/wasm.go).
The biggest problems with WebAssembly are:
- It doesn't work well as an embedded expression language. With WebAssembly, we would really want to
  have the binaries published somewhere (docker images?) and then referenced in CRD
  declarations so the apiserver could then load and execute them. This would be
  far less convenient for writing simple validation rules than just inlining expressions.
- WebAssembly runtimes require `cgo` to build, something that
  might be difficult to integrate into api-server.
- Passing strings across a WebAssembly boundary is currently dependent on the target language, so any supported
  target language would need a small shim library to be supported. This complicates the developer
  workflow.

See also github.com/chimera-kube/chimera-admission

### Build our own

Given that this would require a much larger engineering investment, we do not plan on entertaining
unless there is strong evidence that none available expression languages are able to support CRD validation
use cases well.

### Make it easier to validate CRDs using webhooks

This has been explored by the community. There are examples in the ecosystem of Rego, Expr and WebAssembly
in the ecosystem.

Kubebuilder can automatically create and manage a webhook to run validation and defaulting code (
https://book.kubebuilder.io/cronjob-tutorial/webhook-implementation.html).

But for a CRD developer that just needs to add a simple validation, having direct access
to an expression language is far simpler than exploring this ecosystem to find the easiest
way to do validation and then investing in it (which may require buy-in to a larger framework)
is a time consuming way to solve what should be a simple problem.

For cluster operators, regardless of what extensions they install in their cluster,
it is to their advantage to install the fewest webhooks possible since.


## Infrastructure Needed (Optional)

<!--
Use this section if you need things from the project/SIG. Examples include a
new subproject, repos requested, or GitHub details. Listing these here allows a
SIG to get the process for these resources started right away.
-->
