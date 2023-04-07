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
# KEP-3937: Declarative Validation of Kubernetes Native Types

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
  - [Unions](#unions)
  - [CEL Validation](#cel-validation)
  - [Formats](#formats)
  - [IDL tags](#idl-tags)
- [Performance considerations](#performance-considerations)
- [Analysis of existing validation rules](#analysis-of-existing-validation-rules)
- [Migration](#migration)
  - [User Stories (Optional)](#user-stories-optional)
    - [Story 1](#story-1)
    - [Story 2](#story-2)
  - [Notes/Constraints/Caveats (Optional)](#notesconstraintscaveats-optional)
  - [Risks and Mitigations](#risks-and-mitigations)
- [Design Details](#design-details)
  - [Request handling](#request-handling)
  - [Prototype Notes](#prototype-notes)
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

With this enhancement, Kubernetes Developers will declare validation rules using
IDL tags in the `types.go` files that to define the Kubernetes native API types.
For example:

```go
// staging/src/k8s.io/api/core/v1/types.go

// +valdiation=rule:"!self.hostNetwork || self.containers.all(c, c.containerPort.all(cp, cp.hostPort == cp.containerPort))"
type PodSpec struct {
  Containers []Container `json:...`
}

type Container struct {
  // ...

  Ports []ContainerPort `json:...`

  // ...
}

type ContainerPort struct {
  // ...

  // ...
  //
  // +minimum=1
  // +maximum=65535
  HostPort int32 `json:...`

  // ...
  //
  // +minimum=1
  // +maximum=65535
  ContainerPort int32 `json:...`

  // ...
}
```

In this example, both `+valdiation`, `+minimum` and `+maximum` are IDL tags.

The declarative validation rules will be used by the kube-apiserver to
validate API requests.

The declarative validation rules will also be included in the published OpenAPI:

```json
// Pod Spec
"openAPIV3Schema": {
  "type": "object",
  "x-kubernetes-validations": [
    {
      "rule": "!self.hostNetwork || self.containers.all(c, c.containerPort.all(cp, cp.hostPort == cp.containerPort))"
    }
  ],
}

...

// Container Port
"openAPIV3Schema": {
  "type": "object",
  "properties": {
    "hostPort": {
      "minimum": 1,
      "maximum": 65535
      ...
    }
    "containerPort": {
      "minimum": 1,
      "maximum": 65535
      ...
    }
  }
}
```

It is important to note that the declarative validation will use the same set of
validation options available today for CRDs. namely:

- [JSON Schema value validations](https://datatracker.ietf.org/doc/html/draft-bhutton-json-schema-validation-00) (e.g. `format`, `maxItems`)
- [CEL Validation Rules](https://kubernetes.io/docs/tasks/extend-kubernetes/custom-resources/custom-resource-definitions/#validation-rules) (e.g. `self.replicas <= self.maxReplicas`)
- Relevant [merge strategy declarations](https://kubernetes.io/docs/reference/using-api/server-side-apply/#merge-strategy) (e.g. `listType=set`)

## Motivation

Kubernetes API validation rules are currently written by hand, which makes them
difficult for users to access directly.

Declarative validation will benefit Kubernetes maintainers:

- It will make it easier to develop, maintain and review APIs.
- It will make it easier programatically inspect and analyze the API, enabling
  new tools and improved documentation.
- It will enable improvements to the API machinery. For example, a feature like
  ratcheting validation will become more tractable to implement because the
  feature can be implemented once in the declarative validation subsystem
  rather than piecemeal across the 15k lines of hand written validation code.

Declarative validation will also benefit Kubernetes users:

- It will give users direct access to the actual API validation rules, which
  are currently only available to developers willing and able to find and read
  the hand written validation rules.
- It will enable clients to perform validation of native types earlier in
  development worflows ("shift-left"), such as at with a Git pre-submit linter.
- It will improve API composition. In particular CRDs that embed native types
  (such as PodTemplate), which gain validation of the native type automatically.
  This has the potential to simplify controller development and improve end
  user experiences when using CRDs.

Please feel free to try out the
[prototype](https://github.com/jpbetz/kubernetes/blob/cel-for-native/staging/src/k8s.io/sample-apiserver/pkg/apis/wardle/validation/README.md)
to get hands on experience with this proposed enhancement.

### Goals

- Vast majority (95%+) of hand written validation are replaced with declarative
validation. The remaining hand written rules will primarily be rules that _should
not_ be published as part of the API, usually because server side state is
involved in the validation decision.

- `types.go` files become the de-facto IDL of Kubernetes for native types.
It is worthing noting that `+enum` support, `+default` support and similar
enhancements all moved our API development forward in this direction. This
enhancement is an attempt to finish that story arc.

- CRDs and native types are validated and published in OpenAPI in a consistent
and uniform way. Improvement we make to CEL to enable declarative validation of
native types will be made available to CRD authors and vis-versa.

### Non-Goals

- It is not a goal for _all_ validation rules to be published to OpenAPI. There
  exist complex, stateful rules that should not be published in OpenAPIv3. These
  will remain "server side only" validation rules.
- It is not a goal to require that _all_ validation rules be written in CEL. We
  will continue to support hand written validation rules indefinitely for use
  cases that are a poor fit using the existing validation mechanism (or
  something very similar).


## Proposal

Declarative validation will be performed against versioned APIs. This differs from the hand
written validation, which is evaluated against internal types.

Go IDL tags will be added to support the following declarative validation rules:

| Type of valiation      | Go IDL tag                                                       | OpenAPI validation field                                                           |
| ---------------------- | ---------------------------------------------------------------- | ---------------------------------------------------------------------------------- |
| string format          | `+format={format name}`                                          | `format`                                                                           |
| size limits            | `+min{Length,Properties,Items}`, `+max{Length,Properties,Items}` | `min{Length,Properties,Items}`, `max{Length,Properties,Items}` |
| numeric limits         | `+minimum`, `+maximum`, `+exclusiveMinimum`, `+exclusiveMaximum` | `minimum`, `maximum`, `exclusiveMinimum`, `exclusiveMaximum`                       |
| required fields        | `+optional` (exists today)                                       | `required`                                                                         |
| enum values            | `+enum` (exists today)                                           | `enum`                                                                             |
| uniqueness             | `listType=set` (sets and map keys)                               | `x-kubernetes-list-type`                                                           |
| regex matches          | `+pattern`                                                       | `pattern`                                                                          |
| cross field validation | `cel=rule:"{CEL expression}"`                                     | `x-kubernetes-validations`                                                         |
| [transition rules](https://kubernetes.io/docs/tasks/extend-kubernetes/custom-resources/custom-resource-definitions/#transition-rules)   | `cel=rule:"{CEL expression using oldSelf}"`   | `x-kubernetes-validations`                                                   |
| special case: metadata name format   | `+metadataNameFormat={format name}`                | `x-kubernetes-validations` (see "format" section below for details) |

### Unions

Unions are informally defined but common in built-in types.

We will convert most union validation to CEL expressions. For example:

```go
// staging/src/k8s.io/api/apps/v1/types.go

type DeploymentSpec struct {
  // ...
  //
  // +minimum=0
  Replicas string `json...`
  
  // ...
  // 
  // +validation=rule:"!(self.type == 'Recreate') || !has(self.rollingUpdate)",message="may not be specified when strategy `type` is Recreate",reason=Forbidden,field="rollingUpdate"
  // +validation=rule:"!(self.type == 'RollingUpdate') || has(self.rollingUpdate)",message="this should be defaulted and never be nil",reason=Required,field="rollingUpdate"
  Strategy DeploymentStrategy `json:...`
  
  // ...
}
```

For backward compatibility with existing validations, `reason` and `field` will
be added to make it possible to declare a validation rule in CEL that matches
not just the logic of the hand written validation is replaces, but also the
exact field path and reason type.

### CEL Validation

CEL will play a major role in declarative validation by offering a way to
declare validation rules for use cases that are too complex to be declared using
JSON Schema value validations. Using CEL to validate APIs has been successfully
demonstrated by CRD Validation Rules, which are on track for GA in 1.28.

CEL rules will typically be placed on struct or field declarations and access
multiple fields nested below the level of the type or field where the CEL
expression is declared, e.g.:

```go
  
  //+validation=rule:"!self.widgetType == 'Component' || !['badname2', 'badname2'].exists(notAllowed, self.componentName.contains(notAllowed))",reason=Forbidden
  type FizzBuzzSpec struct {
    Type WidgetType `json...`
    ComponentName string `json...`
  }
```

We will need to extend our CEL libraries to make it possible to migrate all the
validation rules that exist in the Kubernetes API today.

- `isFormat() <bool>` and `validateFormat() <list<string>>` will be added to allow formats to be checked in CEL
  expression and for format violations to be reported using `messageExpression: "self.validateFormat('ipv6')"`
- IP and CIDR libraries will be added that allow for a wide range of IP (v4 and v6) checks to be performed.

TODO: Flesh out the exact library functions we will to add.

### Formats

A significant portion of all validation rules in the API check that a field
value conforms to a particular "format". A prominent example is `metadata.name`
and `metadata.generateName` validation.

We will extend the [available list of
formats](https://github.com/kubernetes/kube-openapi/blob/7fbd8d59e5b89f2ca43a5dcececbffc0bb186c37/pkg/validation/strfmt/default.go#L128)
to cover formats heavily used by the Kubernetes API, namely:

| Format                       | Primary validate uses              |
| ---------------------------- | ---------------------------------- |
| 'dns1123subdomain'           | metadata name and generateName     |
| 'dns1123label'               | metadata name and generateName     |
| 'dns1035label'               | Scoped names and keys              |
| 'quantity'                   | various fields                     |

We will add all of these to the supported list of formats in kube-openapi.
We will also document all supported formats on the Kubernetes website.

Other candidate format types are:

- Qualified name
- Fully qualified name
- Fully qualified domain name

### IDL tags

IDL tags may be used directly on type declarations and indirectly on field and
type aliases. For example:

```go
type Widget struct {
  // +cel=rule:"self.matches('[a-z][1-9]+')"
  Component PartId `json...`
}

// +maxLength=20
type PartId = Identifier

// +format=dns1123label
type Identifier string

type Contraption struct {
  Component Identifier `json...`
}
```

In the above example, the `widget.component` field is validated against all three
IDL tags. But `contraption.component` is only validated against `+format=dns1123label`.

Shared types present a challenge. For example, different Kubernetes resources
have different validation rules for `metadata.name` and `metadata.generateName`.
But all resources share the `ObjectMeta` type.

We can support these uses cases with CEL validation rules:

```go
type ExampleSpec struct {

  // +cel=rule:"!has(self.name) || self.name.isFormat('dns1123subdomain')"
  // +cel=rule:"!has(self.generateName) || self.generateName.replace('-$', 'a').isFormat('dns1123subdomain')"
  metav1.ObjectMeta `json...`
}
```

Because the above use case is so common, we plan to offer a special tag to make
declaring the above rules convenient:

```go
// +metadataNameFormat='dns1123subdomain'
metav1.ObjectMeta `json...`
```

Another example: Pod `container` and `initContainer` fields share the same
`Container` type but have different validation rules. The rules that are the
same for both can be declared on `Container` but any rules that are different
can be declared on the `container` and `initContainer` fields using CEL. E.g.:

```go
// +validation=rule:"!has(self.RestartPolicy)"
Containers []Container `json...`
```

## Performance considerations

The design decision to declaratively validate API versions has performance
implications.

Today, the apiserver's request handler decodes incoming versioned API requests
to the versioned type and then immediately performs an "unsafe" conversion to
the internal type when possible. All subsequent request processing uses the
internal type as the "hub" type. Validation is written for the internal
type and so are per-resource strategies.

With this change, the internal type will no longer be responsible for
validation.

If we were to convert from the internal back to the version of the API request
for validation, we would introduce one additional conversion. If we make this
an "unsafe" conversion, than is will be low cost for the vast majority of requests.

We will benchmark this approach and plan to use it for alpha.

Long term, we could do better:

Since the internal type will no longer be used for validation, it becomes a lot
less important. It is still important to have a hub type. But why not pick one
of the versioned types to be the hub type? The vast majority of APIs only have
one version anyway. The obvious candidate version to choose for the hub version
would the perferred storage version.

Switching to a versioned type for the hub type would have a few implications:

- We would eliminate the need for internal versions.
- We would reduce the need for "unsafe" conversions (a small security win) since
  most APIs only have a single v1 version and so the entire request handling process
  would simply use that version.
- We would introduce more conversion when API request version differs from the
  hub version. But beta APIs are off-by-default and we expect a lot less mixed
  version API usage than in the past.

This "hub version" change feels like something that could be made somewhat
independant of this KEP.


## Analysis of existing validation rules

At the time of writing this document, there are 1181 validation rules written in about
15k lines of go code in [kubernetes/kubernetes/pkg/apis](https://github.com/kubernetes/kubernetes/commit/0c62b122c02bff9131b6db960042150a3638d3f3).

Roughly 30% of validation rules may require CEL expressions. 15% of these are
forbidden rules which primarily check which fields are allowed when a union
discriminator is set. 10% are object name validations, and the remaining 10% are
cross field validation checks, mainly mutual exclusion and some "transition
rules" (e.g. immutability checks).

The remaining 70% of validation rules can be represented using JSON Schema value
validations. `optional`, `format` and `enum` will the the most frequently used.

![Validation types](validation-types.svg)

![Object metadata name validations](metadata-name-types.svg)

![Requires CEL](requires-cel.svg)

## Migration

High level plan:

- Declarative validation rules will be added to APIs (but hand written
  validation rules will not be removed for quite some time).
- Feature flag: `DeclarativeValidationInOpenAPI` will allow clients to access
  the validation rules in via the OpenAPI endpoints.
- Feature flag: `DeclarativeValidationEnabled`: Will turn
  on declarative validation enforcement. TODO: Do we need per-API group gating?
- Feature flag: `LegacyValidationEnabled` (defaults to true):
  will be available to turn off legacy validation. May only be set to `false`
  if `DeclarativeValidationEnabled` is set to true.
- When both `DeclarativeValidationEnabled` and `LegacyValidationEnabled` are
  enabled, any difference in the validation errors caught by the two validators
  will be logged in a special way making them easy to find and analyze, but only
  the validation errors caught by legacy validation will be reported to the
  client.
- e2e tests will be instrumented to detect any difference in validation rules
  between declarative validation and legacy validation, using the log
  information mentioned above, and to fail if a PR introduces a difference.
- `validation_test.go` unit tests will be improved to make it easy to verify
  that declarative validation rules are agreeing with legacy validation rules.
- A linter will be added to ensure that all versions of an API have the same
  validation rules unless an exception is made (via a line in an exceptions
  file). In practice we expect the number of exceptions to be very small and
  so this linter will primarily ensure that we keep API validation in sync.
- We will benchmark all validation changes. We will make sure there is a
  framework in place for this so that it is easy to do.

Migration steps for each API group:

- In a single PR:
  - Add declarative validation rules using Go struct tags.
  - Validate that both declarative validation and legacy validation produce
    the same errors for validation_test.go.
  - Enable e2e testing testing of the API group so that each e2e test run includes
    the API group in `DeclarativeValidationEnabled` and checks for any differences
    reported in logs between declarative validation and legacy validation and
    fails the e2e test run if any are found.
  - API reviewers approve the change.
- ...Time passes. All API validation changes are deamed stable...
- In a single PR:
  - The `DeclarativeValidationEnabled` is defaulted to true.
  - The `LegacyValidationEnabled` is defaulted to false.
- ...Deprecation of legacy validation is announced...
- ...Deprecation wait period passes...
- The legacy API is deleted and can no longer be enabled/disabled via the
  feature gates.

### User Stories (Optional)

<!--
Detail the things that people will be able to do if this KEP is implemented.
Include as much detail as possible so that people can understand the "how" of
the system. The goal here is to make this feel real for users without getting
bogged down.
-->

#### Kubernetes developer wishes to add a field to a existing API version

1. Developer add the field to the Go struct
2. Developer adds needed IDL tags to Go struct
3. Developer adds validation_test.go cases (same as today)
4. API reviewers review IDL tags along with Go struct change

#### Kubernetes devekoper adds a v1beta2 version of an API

1. Develop copies over v1beta1 API and creates v1beta2
2. Linter verifies that IDL tags match for both version of API (unless
   exceptions are put in exception file)
3. API reviewer can review change knowing that validation is consistent unless
   there are lines added to exception file

#### User wishes to validate the YAML of a kubernetes native type as a Git pre-submit check

1. OpenAPI of native types is downloaded from kube-apiserver
2. Tool that checks YAML against OpenAPI schema is used to validate YAML in a
   Git pre-submit (there are tools being written now that will also handle
   x-kubernetes-validations)

#### User wishes to use Kubebuilder to define a custom resource that embeds PodTemplate

1. Go struct that declares custom resource references the go struct of v1.PodTemplate.
2. Kubebuilder recognizes the Go IDL tags introduced by this KEP (Note that
   Kubebuidler already has IDL tags:
   https://book.kubebuilder.io/reference/markers/crd-validation.html, but they
   are different than what we propose here)
3. Kubebuilder generates appropriate OpenAPI for the CRD resulting in full
   validation of the PodTemplate

### Notes/Constraints/Caveats (Optional)

<!--
What are the caveats to the proposal?
What are some important details that didn't come across above?
Go in to as much detail as necessary here.
This might be a good place to talk about core concepts and how they relate.
-->

### Risks and Mitigations

- Risk: Declarative validation adds significant latency to API request handling.
  - Potential causes:
    - Validation versioned types introduces extra conversions.
      - Why this might be OK: Requests are received as the versioned type, so it
        should be feasible to avoid extra conversions.
    - CEL expressions are significantly more expensive to run.
      - Why this might be OK:
        - CEL expressions are expected to account for 30% of all
          validation rules, with the rest expressed as simple value validations
          (required checks, enum value checks) that do not require CEL
          evaluation.
        - CEL evaluation is relatively efficient, we will gather detailed
          benchmarks baselined against native types to demonstrate this.
        - If CEL becomes a bottleneck, we have potential mitigations:
          - Generate Go code from CEL cel expressions.
          - Hand write validation code for select validations and call those
            directly (or from CEL).
    - A general purpose validator must (a) convert to unstructured or (b) use
      reflection.
      - Why this might be OK: SMD's
        [value](https://github.com/kubernetes-sigs/structured-merge-diff/tree/master/value)
        wrappers have already been benchmarked and shown to have low overhead.
- Risk: Migration introduces breaking change to API validation.
  - Mitigations: See above migration plan, which includes numerous cross checks
    to prevent mistakes from slipping through.

## Design Details

### Request handling

Write requests are decoded, converted to internal, processed by any mutating admission
plugins, processed any before create handlers, validated, then converted to the storage
version and stored.

Our plan:

- Early Alphas:  If feature flag is set, convert from internal back to versioned
  type (via unsafe where possible) for declarative validation. Because all CEL
  evaluation is currently written for unstructured values, we will convert to
  unstructured before declarative validation.
- Before graduation to Beta: Benchmark performance. Decide if we should invest
  in the "Switch hub version to be an API version" idea proposed in the
  performance section of this KEP. Also decide if we should invest in using
  using SMD value wrappers to avoid conversion to unstructured.

### Prototype Notes

The core changes proposed here can be implement with only a small set of code
changes. A working
[prototype](https://github.com/jpbetz/kubernetes/blob/cel-for-native/staging/src/k8s.io/sample-apiserver/pkg/apis/wardle/validation/README.md)
was built quite quickly by leveraging code that was merged recently as part of
other CEL work. The main changes were:

- Extend kube-openapi to add the needed Go tag support (Draft PR:
  https://github.com/kubernetes/kube-openapi/pull/381)
- Reuse the same validator as used today for CRDs, but generalize it to work with native types
  - This was be done by leveraging
    https://github.com/kubernetes/kubernetes/pull/113312 and
    https://github.com/kubernetes/kubernetes/pull/116267 to handle the
    resolution and conversion of native type schemas

TODO:

- Leverage SMD's
  [value](https://github.com/kubernetes-sigs/structured-merge-diff/tree/master/value)
  wrappers to minimize conversion costs? Benchmark this.
- Agree on and publish the needed format additions.
- Design and implement the CEL library changes. This will be quite a bit of
  work. But the changes can be rolled out safely using
  https://github.com/kubernetes/kubernetes/pull/116779.
- Integrate declarative validation into the admission chain in a way that is
  general purpose, feature gated, and minimizes conversions.
- Implement all feature gating, cross checking and logging of declarative
  validation and legacy validation.
- Set up e2e to use the feature gating, cross checking and logging of
  declarative validation and legacy validation. Any differences should result in
  a test failure.
- Add support for `reason` to `+validation`. (also allow CRDs to use this flag?
  Will require a feature flag and rollout)
- Set up `validation_test.go` files to be able to test both declarative and
  legacy validation and compare the results of both.
- Prepare a detailed migration guide.

### Test Plan

<!--
**Note:** *Not required until targeted at a release.*
The goal is to ensure that we don't accept enhancements with inadequate testing.

All code is expected to have adequate tests (eventually with coverage
expectations). Please adhere to the [Kubernetes testing guidelines][testing-guidelines]
when drafting this test plan.

[testing-guidelines]: https://git.k8s.io/community/contributors/devel/sig-testing/testing.md
-->

[ ] I/we understand the owners of the involved components may require updates to
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

- `<package>`: `<date>` - `<test coverage>`

##### Integration tests

<!--
This question should be filled when targeting a release.
For Alpha, describe what tests will be added to ensure proper quality of the enhancement.

For Beta and GA, add links to added tests together with links to k8s-triage for those tests:
https://storage.googleapis.com/k8s-triage/index.html
-->

- <test>: <link to test coverage>

##### e2e tests

<!--
This question should be filled when targeting a release.
For Alpha, describe what tests will be added to ensure proper quality of the enhancement.

For Beta and GA, add links to added tests together with links to k8s-triage for those tests:
https://storage.googleapis.com/k8s-triage/index.html

We expect no non-infra related flakes in the last month as a GA graduation criteria.
-->

- <test>: <link to test coverage>

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

Documentation is available on [feature gate lifecycle] and expectations, as
well as the [existing list] of feature gates.

[feature gate lifecycle]: https://git.k8s.io/community/contributors/devel/sig-architecture/feature-gates.md
[existing list]: https://kubernetes.io/docs/reference/command-line-tools-reference/feature-gates/
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

Feature gates are typically disabled by setting the flag to `false` and
restarting the component. No other changes should be necessary to disable the
feature.

NOTE: Also set `disable-supported` to `true` or `false` in `kep.yaml`.
-->

###### What happens if we reenable the feature if it was previously rolled back?

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

###### Can enabling / using this feature result in resource exhaustion of some node resources (PIDs, sockets, inodes, etc.)?

<!--
Focus not just on happy cases, but primarily on more pathological cases
(e.g. probes taking a minute instead of milliseconds, failed pods consuming resources, etc.).
If any of the resources can be exhausted, how this is mitigated with the existing limits
(e.g. pods per node) or new limits added by this KEP?

Are there any tests that were run/should be run to understand performance characteristics better
and validate the declared limits?
-->

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
