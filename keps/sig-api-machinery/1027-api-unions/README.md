# KEP-1027: Union types

## Table of Contents

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
  - [Discriminator Field](#discriminator-field)
  - [Go Markers](#go-markers)
    - [Discriminator Values](#discriminator-values)
      - [Empty Union Members](#empty-union-members)
    - [Examples](#examples)
  - [OpenAPI](#openapi)
  - [Normalization and Validation](#normalization-and-validation)
    - [Normalization](#normalization)
    - [Validation](#validation)
    - [Ratcheting Validation](#ratcheting-validation)
    - [Migrating existing unions](#migrating-existing-unions)
  - [Test Plan](#test-plan)
      - [Prerequisite testing updates](#prerequisite-testing-updates)
      - [Unit tests](#unit-tests)
      - [Integration tests](#integration-tests)
      - [e2e tests](#e2e-tests)
  - [Graduation Criteria](#graduation-criteria)
    - [Alpha Graduation](#alpha-graduation)
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
<!-- /toc -->

## Release Signoff Checklist

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

Modern data model definitions like OpenAPI v3 and protobuf (versions 2 and 3)
have a keyword to implement “oneof” or "union". They allow APIs to have a better
semantics, typically as a way to say “only one of the given fields can be
set”. We currently have multiple occurrences of this semantics in kubernetes core
types, at least:
- VolumeSource is a structure that holds the definition of all the possible
  volume types, only one of them must be set, it doesn't have a discriminator.
- DeploymentStrategy is a structure that has a discrminator
  "DeploymentStrategyType" which decides if "RollingUpate" should be set

The problem with the lack of solution is that:
- The API is implicit, and people don't know how to use it
- Clients can't know how to deal with that, especially if they can't parse the
  OpenAPI
- Server can't understand the user intent and normalize the object properly

## Motivation

Currently, changing a value in an oneof type is difficult because the semantics
is implicit, which means that nothing can be built to automatically fix unions,
leading to many bugs and issues:
- https://github.com/kubernetes/kubernetes/issues/35345
- https://github.com/kubernetes/kubernetes/issues/24238
- https://github.com/kubernetes/kubernetes/issues/34292
- https://github.com/kubernetes/kubernetes/issues/6979
- https://github.com/kubernetes/kubernetes/issues/33766
- https://github.com/kubernetes/kubernetes/issues/24198
- https://github.com/kubernetes/kubernetes/issues/60340

And then, for other people:
- https://github.com/rancher/rancher/issues/13584
- https://github.com/helm/charts/pull/12319
- https://github.com/EnMasseProject/enmasse/pull/1974
- https://github.com/helm/charts/pull/11546
- https://github.com/kubernetes/kubernetes/pull/35343

This is replacing a lot of previous work and long-standing effort:
- Initially: https://github.com/kubernetes/community/issues/229, then
- https://github.com/kubernetes/community/pull/278
- https://github.com/kubernetes/community/pull/620
- https://github.com/kubernetes/kubernetes/pull/44597
- https://github.com/kubernetes/kubernetes/pull/50296
- https://github.com/kubernetes/kubernetes/pull/70436

Server-side [apply](http://features.k8s.io/555) is what enables this proposal to
become possible.

### Goals

Provide a simple mechanism for API authors to label fields of a resource as
members of a oneOf, in order to receive standardized validation and
normalization, rather than having to author it themeselves per
resource as currently done as a workaround in various validation
functions (e.g. `pkg/apis/<group>/validation/validation.go`).

* Validation - ensuring only one member field is set (or at most one if
  desired).
* Normalization - ensuring the API server can understand the intent of clients
  that are unable to update/modify fields the clients are unaware of due to
  version skew.

### Non-Goals

Migrating all existing unions away from their bespoke validation
logic (e.g validation functions), is an explicit non-goal and will be pursued in
a separate KEP or later release.

## Proposal

### User Stories (Optional)

<!--
Detail the things that people will be able to do if this KEP is implemented.
Include as much detail as possible so that people can understand the "how" of
the system. The goal here is to make this feel real for users without getting
bogged down.

-->


#### Story 1

As a CRD owner, I can use simple semantics (such as openapi tags/go markers), to express the
desired validation of a oneOf (at most one or exactly one field may be set), and
the API server will perform this validation automatically.

#### Story 2

As a client, I can read, modify, and update the union fields of an object, even
if I am not aware of all of the possible fields, and the server will properly
interpret my intent.

### Notes/Constraints/Caveats (Optional)

<!--
What are the caveats to the proposal?
What are some important details that didn't come across above?
Go in to as much detail as necessary here.
This might be a good place to talk about core concepts and how they relate.
-->

### Risks and Mitigations
- We need to ensure we do not break existing union types. This can be done by
  not forcing existing unions to conform to the newly proposed union semantics.
  Integration testing with older types should give us the confidence to be sure
  we have done so.
- There is a lot of risk for errors when there exists skew between clients and
  server. In the section on normalization, we discuss mitigating these risks.

<!--
What are the risks of this proposal, and how do we mitigate? Think broadly.
For example, consider both security and how this will impact the larger
Kubernetes ecosystem.

How will security be reviewed, and by whom?

How will UX be reviewed, and by whom?

Consider including folks who also work outside the SIG or subproject.
-->

## Design Details

### Discriminator Field

We propose that all new unions maintain a "discriminator". This is a field that
points to which of the other "union member" fields is to be respected as the
truly desired union field in the case that there are any conflicts.

In order to demonstrate the need for the discriminator, we developed an
extensive [test matrix](https://docs.google.com/spreadsheets/d/1dIOOqgrgvMI9b2412eVuRSydEaOxhYOoqk6bUjZOY9c/edit?resourcekey=0-wlOfJTC_EX-qpU680STHMA#gid=3601413) that looks at various configurations of the performing
REST operations on a union where the client or server is unaware of a newly
added field to the union (due to version skew).

We present a [guide doc](https://docs.google.com/document/d/1Wruosjo0ELLl1yxauzpsUjgH2fK9KdgXDmOdJ5sG7Kg/edit?resourcekey=0-8Pwzx6EvsFR7VQoXzCTY4Q) on how to interpret the test matrix, but the major
conclusions are as follows (along with the test case number from the test matrix):

* (Case #22 and #27) If an unstructured client (i.e. a client that represents data as raw json maps with no knowledge of the schema)
  is unaware of field on the union, but wants to clear
  the union entirely (assuming the union is optional), it will have no way of doing
  so without a discriminator. With a discriminator, the client can express its
  intention by setting the discriminator to the empty value and the server can
  respect its intentions and clear any fields the client is unaware of.
* (Case #12 and #16) If a structured client is unaware of a field in the union that is set and it
  just wants to echo back the union it received in a get request (such as when
  updating other parts of the object), a client without a discriminator will
  silently drop the currently set field, while a client with the discriminator
  will not change the discriminator value, indicating to the server that no
  changes are desired in the union.
* (Case #34 and #39) If a client sets a union field that the server is not aware of, the server
  will silently drop it and attempt to clear the object of the union field. With
  a discriminator, the server will see the unrecognized discriminator value and
  can fail loudly.
* (Case #23 and #28) When a client goes to set a field it knows of, but a separate field it doesn't
  know about is currently set, the server can simply know to always respect the
  discriminator. Without a discriminator, the server will have to do convoluted
  logic to detect that the previously set field has not been modified and that
  only one of the other union fields has been.

### Go Markers

We're proposing a new type of tags for go types (in-tree types, and also
kubebuilder types):


- `// +unionDiscriminator` before a field means that this field is the
  discriminator for the union. This field MUST be an enum defined as a string (see section on
  discriminator values). This field MUST be required if there is no default
  option, omitempty if the default option is the empty string, or optional and
  omitempty if a default value is specified with the `// +default` marker.
- `// +unionMember=<memberName> before a field means that this
  field is a member of a union. The `<memberName>` is the name of the field that will be set as the discriminator value.
  It MUST correspond to one of the valid enum values of the discriminator's enum
  type. It defaults to the go (i.e `CamelCase`) representation of the field name if not specified.
  `<memberName>` should only be set if authors want to customize how the fields
  are represented in the discriminator field. `<memberName>` should match the
  serialized JSON name of the field case-insensitively.
- `// +unionDiscriminatedBy=<discriminatorName>` before a member field identifies which
  discriminator (and thus which union) the member field belongs to. Optional
  unless there are multiple unions/discriminators in a single struct. If used,
  it must be the go (i.e. `CamelCase`) representation of the field name tagged
  with `unionDiscriminator`.

#### Discriminator Values

Here we present a description of how discriminators and their valid values
should be defined.

As described above, the discriminator field must be a string and required.
Because, there are only a few specific values that the discriminator can be, we
propose that all discriminators should be defined as an enum, and should be
tagged so via the enum go marker `// +enum`.

Required unions will have the number of valid discriminator values equal to the
number of member fields (see exception below on empty union members). Optional
unions will have the number of valid discriminator values equal to the number of
member fields, plus one additional value for when "no member" is desired. By
convention, this "no member" discriminator value should be the empty string.

We define optional unions as union where "at most" one member field of the union
must be non-nil (as opposed to a required union, where "exactly" one member
field of the union must be non-nil).

##### Empty Union Members

In some cases there are more discriminator values than there are member fields
defined in the struct when that specific member requires no configuration. An
example is the `DeploymentStrategy` where it has one member field `rollingUpdate`,
but two valid discriminator values `RollingUpdate` and `Recreate`. By using an
enum as the discriminator value we are able to define values beyond the member
fields in order to accommodate this pattern.

#### Examples

Below is an example of how to define a union based on the above design

```
// +enum
type UnionType string

const (
  FieldA UnionType = "FieldA"
  FieldB UnionType = "FieldB"
  FieldC UnionType = "FieldC"
  FieldD UnionType = "FieldD"
  FieldNone UnionType = ""
)

type Union struct {
  // +unionDiscriminator
  // +required
  UnionType UnionType

  // +unionMember
  // +optional
  FieldA int
  // +unionMember
  // +optional
  FieldB int
}
```


Note unions can't span across multiple go structures (all the fields that are part of a union has to be together in the
same structure), examples of what is allowed:

```
// This will have one embedded union.
type TopLevelUnion struct {
  Name string `json:"name"`

  Union `json:",inline"`
}

// +enum
type UnionType string

const (
  FieldA UnionType = "FieldA"
  FieldB UnionType = "FieldB"
  FieldC UnionType = "FieldC"
  FieldD UnionType = "FieldD"
  FieldNone UnionType = ""
)

// This will generate one union, with two fields and a discriminator.
type Union struct {
  // +unionDiscriminator
  // +required
  UnionType UnionType `json:"unionType"`

  // +unionMember
  // +optional
  FieldA int `json:"fieldA"`
  // +unionMember
  // +optional
  FieldB int `json:"fieldB"`
}

// +enum
type Union2Type string

const (
  Alpha Union2Type = "ALPHA"
  Beta = "BETA"
)

// This will generate one union that can be embedded because the members explicitly define their discriminator.
// Also, the unionMember markers here demonstrate how to customize the names used for
each field in the discriminator.
type Union2 struct {
  // +unionDiscriminator
  // +required
  Type2 Union2Type `json:"type"`
  // +unionMember=ALPHA,
  // +unionDiscriminatedBy=Type2
  // +optional
  Alpha int `json:"alpha"`
  // +unionMember=BETA
  // +unionDiscriminatedBy=Type2
  // +optional
  Beta int `json:"beta"`
}

// +enum
type FieldType string

const (
  Field1 FieldType = "Field1"
  Field2 = "Field2"
  FieldNone = "None"
)

// This has 3 embedded unions:
// One for the fields that are directly embedded, one for Union, and one for Union2.
type InlinedUnion struct {
  Name string `json:"name"`

  // +unionDiscriminator
  // +required
  FieldType FieldType `json:"fieldType"`
  // +unionMember
  // +unionDiscriminatedBy=FieldType
  // +optional
  Field1 *int `json:"field1,omitempty"`
  // +unionMember
  // +unionDiscriminatedBy=FieldType
  // +optional
  Field2 *int `json:"field2,omitempty"`

  // Union does not label its members, so it
  cannot be inlined
  union Union  `json:"union"`
  Union2 `json:",inline"`
}
```

### OpenAPI

OpenAPI v3 already allows a "oneOf" form, which is accepted by CRD validation
(and will continue to be accepted in the future). That oneOf form will be used
for validation, but is "on-top" of this proposal.

A new extension is created in the openapi to describe the behavior:
`x-kubernetes-unions`.

This is a list of unions that are part of this structure/object. Here is what
each list item is made of:
- `discriminator: <discriminator>` is set to the name of the discriminator
  field, if present,
- `fields-to-discriminateBy: {"<fieldName>": "<discriminateName>"}` is a map of
  fields that belong to the union to their discriminated names. The
  discriminatedValue will typically be set to the name of the Go variable.

Conversion between OpenAPI v2 and OpenAPI v3 will preserve these fields.

### Normalization and Validation

#### Normalization

Normalization refers to the process by which the API server attempts to understand
and correct clients which may provide the server with conflicting or incomplete
information about a union in update or patch requests.

Issues primarily arise here because of version skew between a client and a server,
such as when a client is unaware of new fields added to a union and thus doesn't
know how to clear these new fields when trying to set a different field.

For unions that follow this design, normalization is simple: the server should always respect the
discriminator.

This means that when the server receives an update request with a discriminator set to a
given field, and multiple member fields are set it should clear all fields
except the one pointed to by the discriminator _if and only if_ the
discriminator has been modified. Having multiple fields set, and a discriminator
not modified is invalid and caught later by the validation step (see below).

For both custom resources and built-in types, we expect union normalization to be
called by the request handlers shortly after mutating admission occurs.

#### Validation

Objects must be validated AFTER the normalization process.

Some validation situations specific to unions are:
1. When multiple union fields are set and the discriminator is not set we should
   error loudly that the client must change the discriminator if it changes any
   union member fields.
2. When the server receiveds a request with a discriminator set to a given
   field, but that given field is empty, the server should fail with a clear
   error message. Note this does not apply to discriminator values that do not
   correspond to any field (as in the "empty union members case").

For both custom resources and built-in types, validation will occur as part of
the request validation, before validating admission occurs.

For custom resources, union validation will be done at the same point as the
existing structural schema validation that occurs in the custom resource handler.
This ensures that any generic validation changes made to all custom resources (such
as the ratcheting validation discussed below), behaves appropriately with union
validation.

#### Ratcheting Validation

When updating CRDs to support union validation, it is possible that existing CRs
become invalid.

The naive solution is to require existing CRs to be updated to a valid state
before they can be updated again.

This creates many potential landmines, and so ratcheting validation is proposed
as an alternative. Ratcheting validation means that objects will ignore stricter
validation rules if and only if the existing object also fails the stricter
validation for the same reason.

Ratcheting validation for custom resources is a [separate
effort](https://github.com/kubernetes/kubernetes/issues/94060) proposed
outside of this unions effort. For the initial alpha graduation of unions, we
do not propose supporting ratcheting validation. We will require all invalid CRs
to be made valid before they can be updated (the naive solution).

In order to potentially support ratcheting validation in the future, we will
ensure that all callers of union validation retain access to both old and new
objects, so that future ratcheting validation can be implemented within the
union validation library.

#### Migrating existing unions

As mentioned, one of the goals is to migrate at least one existing union to
using the new marker based union validation and normalization. While open
questions remain around the priority and urgency of migrating existing unions,
nonetheless we should be able to come to a consensus on which types to migrate
first.

For discriminated unions, a couple relatively straightforward discriminated types are
`MetricSpec` and `MetricStatus`. These have clearly defined discriminator values
that map one-to-one to a member field, which make them good candidates for
initial migration.

For non-discriminated unions, there are a few relatively straightforward types
that make good candidates for initial migration, such as `ContainerState`

Until migrated, union types without a discriminator (i.e. only existing unions that have not been migrated
to the current desgin), cannot be tagged with the go markers described above and
thus will not be treated as "unions" in the sense of this currently proposed
normalization and validation logic.

These legacy unions must continue to perform normalization and
validation manually, per resource in the validation functions.


### Test Plan

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

- `<package>`: `<date>` - `<test coverage>`

-->
Core functionality will be extensively unit tested in the SMD typed package
(union_test.go). 

Parts of the kubernetes endpoints handlers package that are modified to call
into the SMD code will also be unit tested as appropriate.





##### Integration tests

We will have extensive integration testing of the union code in the
`test/integration/apiserver` package.

We will be testing along the dimensions of:
* Which fields of the union get modified (none, existing fields, newly updated
  fields)
* Type of union (discriminated vs non-discriminated)
* Whether the client is aware of all the fields
* Whether the server is aware of all fields
* Whether the union is optional or required

A fully documented test matrix exists in a [google
spreadsheet](https://docs.google.com/spreadsheets/d/1dIOOqgrgvMI9b2412eVuRSydEaOxhYOoqk6bUjZOY9c/edit?resourcekey=0-wlOfJTC_EX-qpU680STHMA#gid=3601413) along with a
[guide
doc](https://docs.google.com/document/d/1Wruosjo0ELLl1yxauzpsUjgH2fK9KdgXDmOdJ5sG7Kg/edit?resourcekey=0-8Pwzx6EvsFR7VQoXzCTY4Q) on how to read and understand the test matrix.

As part of implementing the test matrix we will be able to prove the viability
of upgrading existing unions by writing tests to mimic using the standardized
union semantics on existing unions (even if actually upgrading these unions is
outside the scope of alpha graduation)

<!--
This question should be filled when targeting a release.
For Alpha, describe what tests will be added to ensure proper quality of the enhancement.

For Beta and GA, add links to added tests together with links to k8s-triage for those tests:
https://storage.googleapis.com/k8s-triage/index.html

- <test>: <link to test coverage>
-->

##### e2e tests

<!--
This question should be filled when targeting a release.
For Alpha, describe what tests will be added to ensure proper quality of the enhancement.

For Beta and GA, add links to added tests together with links to k8s-triage for those tests:
https://storage.googleapis.com/k8s-triage/index.html

We expect no non-infra related flakes in the last month as a GA graduation criteria.

- <test>: <link to test coverage>
-->
 We are considering adding kubectl e2e tests to mimic kubectl users performing
 various operations on objects with union fields.

### Graduation Criteria

#### Alpha Graduation

- CRDs can be created with union fields and are properly validated when
  created/updated.
- Prove the viability of upgrading existing unions to the new semantics by
  mimicking existing unions in e2e tests.
- Existing unions that don't have discriminators do not break when upgraded.

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

Turning the flag on for alpha just enables different runtime codepaths (i.e.
performing the unified union validation and normalization)

Any schema markers (added by CRD authors or propagated from tags on built-in
types) will appear in the schema, but not do anything if the flag is off.

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

See test matrix and commentary about discriminators. It clearly documents how
the server will use the discriminator to understand the client's intention even
if the client is not aware of all union fields because of version skew.

Skew with alpha flag on/off shouldn't make much of a difference.
* Objects created with the union semantics, but applied to a cluster with the
  alpha flag off will simply not perform union validation and normalization.
* Objects created without union semantics will simply not trigger union
  validation and normalization (regardless of whether the server has the alpha
  enabled or disabled).

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
  - Feature gate name: APIUnions
  - Components depending on the feature gate: kube-apiserver

Request handlers in the api server will call into union validation and
normalization function from the structured-merge-diff repo when feature is
enabled.


###### Does enabling the feature change any default behavior?

Enabling the feature could cause existing CRs to fail validation if the
correspond CRD has union fields and the existing CRs have invalid unions that
were unvalidated when initially created in a cluster that had the unions feature
disabled.

These CRs will need to be corrected in order to pass validation (or the feature
disabled).

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

Yes, requests will simply skip union validation and normalization.

###### What happens if we reenable the feature if it was previously rolled back?

Custom resources that were skipping union validation when when the feature was
rolled back may have allowed invalid data to persist.

For alpha, we require that all modifying requests (update/patch) fail unless the
data passes union validation. Retrieving newly invalid CRs should still always
succeed.

In the future, we may require looser "ratcheting validation" which would allow
modifications to ignore union validation if the existing object fails the union
validation for the same reason as the new object (see section on "Ratcheting
Validation" above). This is not a priority for alpha.

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

We will have integration tests demonstrating how CRs with persisted invalid data
will need to be corrected when the feature is re-enabled (and requires more
strict union validation).

### Rollout, Upgrade and Rollback Planning

<!--
This section must be completed when targeting beta to a release.
-->

###### How can a rollout or rollback fail? Can it impact already running workloads?

N/A

<!--
Try to be as paranoid as possible - e.g., what if some components will restart
mid-rollout?

Be sure to consider highly-available clusters, where, for example,
feature flags will be enabled on some API servers and not others during the
rollout. Similarly, consider large clusters and how enablement/disablement
will rollout across nodes.
-->

###### What specific metrics should inform a rollback?

* `apiserver_request_total` could be watched to see if the number of create and
  update requests that are failing increase substantially.

<!--
What signals should users be paying attention to when the feature is young
that might indicate a serious problem?
-->

###### Were upgrade and rollback tested? Was the upgrade->downgrade->upgrade path tested?

N/A
<!--
Describe manual testing that was done and the outcomes.
Longer term, we may want to require automated upgrade/rollback tests, but we
are missing a bunch of machinery and tooling and can't do that now.
-->

###### Is the rollout accompanied by any deprecations and/or removals of features, APIs, fields of API types, flags, etc.?

N/A
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

For builtins in alpha, it won't be possible to break clients since turning on vs
off will validate the same thing via different code paths.

For CRDs, you can see if they have the new union markers. If the CRD has no
other validation mechanism, turning off the flag may result in CRs accepting
invalid input.

<!--
Ideally, this should be a metric. Operations against the Kubernetes API (e.g.,
checking if there are objects with field X set) may be a last resort. Avoid
logs or events for this purpose.
-->

###### How can someone using this feature know that it is working for their instance?

1. Create a new CRD with a union field (and no other validation mechanism)
2. Apply the CRD
3. Create a CR with an invalid union (multiple fields set, no discriminator
   set), see if the CR is rejected via union validation

When we write the e2e test, a standard union CRD and test CR will be obtainable
for users to test on their instance.
<!--
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
-->

###### What are the reasonable SLOs (Service Level Objectives) for the enhancement?

N/A

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

N/A
<!--
Pick one more of these and delete the rest.

- [ ] Metrics
  - Metric name:
  - [Optional] Aggregation method:
  - Components exposing the metric:
- [ ] Other (treat as last resort)
  - Details:
-->

###### Are there any missing metrics that would be useful to have to improve observability of this feature?

N/A
<!--
Describe the metrics themselves and the reasons why they weren't added (e.g., cost,
implementation difficulties, etc.).
-->

### Dependencies


<!--
This section must be completed when targeting beta to a release.
-->

###### Does this feature depend on any specific services running in the cluster?

N/A
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

N/A
<!--
For alpha, this section is encouraged: reviewers should consider these questions
and attempt to answer them.

For beta, this section is required: reviewers must answer these questions.

For GA, this section is required: approvers should be able to confirm the
previous answers based on experience in the field.
-->

###### Will enabling / using this feature result in any new API calls?

No

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

No

<!--
Describe them, providing:
  - API type
  - Supported number of objects per cluster
  - Supported number of objects per namespace (for namespace-scoped objects)
-->

###### Will enabling / using this feature result in any new calls to the cloud provider?

No
<!--
Describe them, providing:
  - Which API(s):
  - Estimated increase:
-->

###### Will enabling / using this feature result in increasing size or count of the existing API objects?

No
<!--
Describe them, providing:
  - API type(s):
  - Estimated increase in size: (e.g., new annotation of size 32B)
  - Estimated amount of new objects: (e.g., new Object X for every existing Pod)
-->

###### Will enabling / using this feature result in increasing time taken by any operations covered by existing SLIs/SLOs?

In GA (maybe beta), we might expect resource reduction/reliability improvement,
since this removes a need for webhooks.

<!--
Look at the [existing SLIs/SLOs].

Think about adding additional work or introducing new steps in between
(e.g. need to do X to start a container), etc. Please describe the details.

[existing SLIs/SLOs]: https://git.k8s.io/community/sig-scalability/slos/slos.md#kubernetes-slisslos
-->

###### Will enabling / using this feature result in non-negligible increase of resource usage (CPU, RAM, disk, IO, ...) in any components?

New validation and normalization logic should be negligible given that the
functions will be in the same SMD path currently used by SSA code.

We will have benchmarking to validate this assumption.

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

For GA, this section is required: approvers should be able to confirm the
previous answers based on experience in the field.

The Troubleshooting section currently serves the `Playbook` role. We may consider
splitting it into a dedicated `Playbook` document (potentially with some monitoring
details). For now, we leave it here.
-->

###### How does this feature react if the API server and/or etcd is unavailable?

N/A objects are not reachable

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
- Unions implemented, but disabled in SMD.

## Drawbacks

<!--
Why should this KEP _not_ be implemented?
-->
An issue that one might have with requiring a discriminator is that it might
seem redundant to have to set a field _and_ set another field indicating to the
server to use the set field. The reasons for doing so are discussed above in the
normalization section.

One other drawback is that our approach does not standardize all existing unions
into a single format. We don't see a way to do so without drastically changing
existing APIs and breaking backwards compatibility

## Alternatives

###### Non-Discrminated

The primary alternative discussed is to not have a discriminator for new union
types. As discussed in the normalization section, requiring a discriminator
allows the server to better understand the intentions of clients that do not
have knowledge of all the fields in a union if newer versions of the server add
new fields to the union.

###### "None" Discriminator Values

A number of strategies were discussed around how to represent the "none" value
of the discriminator (see "Discriminator Values" section above).

* One alternative was to mandate the "none" value always be the empty string.
  The advantage to this is its simplicity and not creating a situation where
  different API authors define there "none" value differently, so that anyone
  could immediately know that a discriminator set to "" (empty string), is not
  selecting any of the member fields. Also, it would allow us to not have to
  define the set of enum values for each discriminator (as we could just use the
  name of the member field). The disadvantage is that by not defining the set of
  enum values, we make it impossible to support the "empty union members" case.
* Another alternative was to make the discriminator a pointer to a string and
  its value nil. The disadvantage here is that this requires more complicated
  union validation logic (first do a nil check, then check the value) and makes
  it harder to determine client intent on patches where the discriminator is not
  set.
* A third alternative is to require all unions be defined in their own separate
  struct. This was rejected because there are many existing unions that define
  random fields that are not members in the union within the same struct as
  fields that do make up the union and we hope to be able to migrate at least
  some of the existing unions to the new semantics.

<!--
What other approaches did you consider, and why did you rule them out? These do
not need to be as detailed as the proposal, but should include enough
information to express the idea and why it was not acceptable.
-->
