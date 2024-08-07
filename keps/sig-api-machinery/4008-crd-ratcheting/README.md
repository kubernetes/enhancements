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
# KEP-4008: CRD Validation Ratcheting

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
  - [Nested Value Validations](#nested-value-validations)
  - [XListType and XMapKeys](#xlisttype-and-xmapkeys)
  - [Atomic Lists and Maps](#atomic-lists-and-maps)
  - [CEL Rules](#cel-rules)
  - [Advanced Ratcheting](#advanced-ratcheting)
    - [Ratcheting Rules in CEL](#ratcheting-rules-in-cel)
  - [User Stories (Optional)](#user-stories-optional)
    - [CRD Author Tightens a Field](#crd-author-tightens-a-field)
    - [K8s Update Tightens CRD validation](#k8s-update-tightens-crd-validation)
    - [K8s Update Widens CRD Validation](#k8s-update-widens-crd-validation)
  - [Notes/Constraints/Caveats (Optional)](#notesconstraintscaveats-optional)
  - [Risks and Mitigations](#risks-and-mitigations)
    - [Detection of Breaking Schema Changes is Different](#detection-of-breaking-schema-changes-is-different)
      - [Mitigation: Use CRD-Schema-Checker](#mitigation-use-crd-schema-checker)
      - [Mitigation: Use New Objects](#mitigation-use-new-objects)
    - [Not All Rules Can Be Correctly/Easily Ratcheted](#not-all-rules-can-be-correctlyeasily-ratcheted)
      - [Mitigation: Blacklisted Validations](#mitigation-blacklisted-validations)
      - [Mitigation: Conservative Ratcheting Rule](#mitigation-conservative-ratcheting-rule)
- [Design Details](#design-details)
  - [<code>kube-openapi</code> changes](#kube-openapi-changes)
  - [Structural-Schema-based validation changes](#structural-schema-based-validation-changes)
    - [Correlation of Old and New](#correlation-of-old-and-new)
  - [Cel-Validator changes](#cel-validator-changes)
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
  - [CRDs Opt-in to Ratcheting Functionality](#crds-opt-in-to-ratcheting-functionality)
  - [Offline Pass to Flag Invalid Objects at Rest During Upgrade](#offline-pass-to-flag-invalid-objects-at-rest-during-upgrade)
  - [Post-Process Errors](#post-process-errors)
    - [Drawbacks](#drawbacks-1)
      - [Robustness of paths returned by OpenAPI Schema Validator](#robustness-of-paths-returned-by-openapi-schema-validator)
      - [Evaluation of JSON Paths](#evaluation-of-json-paths)
      - [Correlating Errors To Fields](#correlating-errors-to-fields)
  - [Different Ratcheting Rule Per Value Validation](#different-ratcheting-rule-per-value-validation)
    - [Drawbacks](#drawbacks-2)
  - [Ratcheting Within Nested Value Validations](#ratcheting-within-nested-value-validations)
    - [Not](#not)
    - [OneOf](#oneof)
  - [Weaker Ratcheting Rule](#weaker-ratcheting-rule)
    - [Drawbacks](#drawbacks-3)
      - [Allows Arbitrary Invalid Data](#allows-arbitrary-invalid-data)
      - [AllOf](#allof)
      - [AnyOf](#anyof)
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
- [x] (R) Test plan is in place, giving consideration to SIG Architecture and SIG Testing input (including test refactors)
  - [ ] e2e Tests for all Beta API Operations (endpoints)
  - [ ] (R) Ensure GA e2e tests meet requirements for [Conformance Tests](https://github.com/kubernetes/community/blob/master/contributors/devel/sig-architecture/conformance-tests.md) 
  - [ ] (R) Minimum Two Week Window for GA e2e tests to prove flake free
- [x] (R) Graduation criteria is in place
  - [ ] (R) [all GA Endpoints](https://github.com/kubernetes/community/pull/1806) must be hit by [Conformance Tests](https://github.com/kubernetes/community/blob/master/contributors/devel/sig-architecture/conformance-tests.md) 
- [x] (R) Production readiness review completed
- [x] (R) Production readiness review approved
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

Both in this section and below, follow the guidelines of the [documentation
style guide]. In particular, wrap lines to a reasonable length, to make it
easier for reviewers to cite specific portions, and to minimize diff churn on
updates.

[documentation style guide]: https://github.com/kubernetes/community/blob/master/contributors/guide/style-guide.md
-->
The ability to shift left validation logic from controllers to the front-end
is a long-term goal for improving the useability of the Kubernetes project. As
it stands today, our treatment of validation for unchanged fields stands as a 
barrier to both CRD authors and Kubernetes developers to adopting the available
validation features.

## Motivation

<!--
This section is for explicitly listing the motivation, goals, and non-goals of
this KEP.  Describe why the change is important and the benefits to users. The
motivation section can optionally provide links to [experience reports] to
demonstrate the interest in a KEP within the wider Kubernetes community.

[experience reports]: https://github.com/golang/go/wiki/ExperienceReports
-->

CRD authors today are thrown into the deep end when it comes to validation of
the values in their schemas. Generally authors understand to increment version
when adjusting the field layout of a CRD, but to do so when only modifying value 
validations is cumbersome, and more trouble than it is worth.

Modifying a value validation on a CRD today means that you risk breaking the
workflow of all your users, this high price to pay limits adoption, and degrades 
the Kubernetes user experience: we are prevented from shifting validation logic left.

Additionally, this sad state of affairs stunts the advancement of Kubernetes itself.
[KEP-3937](https://github.com/kubernetes/enhancements/pull/3938) proposes to add 
declarative validation to Kubernetes, but to do that would require adding new 
`format` types. This would also break existing Kubernetes user workflows.

See User Stories for more detailed description of
the motivation behind this KEP.

### Goals

<!--
List the specific goals of the KEP. What is it trying to achieve? How will we
know that this has succeeded?
-->
 
1. Remove barriers blocking CRD authors from widening value validations
2. Remove barriers blocking CRD authors from tightening value validations
3. Remove barriers blocking Kubernetes from widening validations
4. Remove barriers blocking Kubernetes from tightening validations
5. Do this automatically for all CRDs installed into clusters with the feature enabled
6. Performance Goals:
  - Constant/negligible persistent overhead
  - Up to 5% time overhead for resource writes (apiserver_request_duration_seconds)
7. Correctness, defined as: not allowing invalid values which would fail a known schema.

### Non-Goals

<!--
What is out of scope for this KEP? Listing non-goals helps to focus discussion
and make progress.
-->
1. Implement complicated logic that can unravel the dependencies of CEL rules ([#116779](https://github.com/kubernetes/kubernetes/pull/116779) is a CEL-specific ratcheting feature)
2. Completeness (not every rule will be able to be ratcheted)

## Proposal

<!--
This is where we get down to the specifics of what the proposal actually is.
This should have enough detail that reviewers can understand exactly what
you're proposing, but should not include things like API designs or
implementation. What is the desired outcome and how do we measure success?.
The "Design Details" section below is for the real
nitty-gritty.
-->

New Feature Flag: `CRDValidationRatcheting`

When enabled, this flag allows updates to custom resources that fail validation to succeed if the validation errors were on unchanged keypaths.

For example:

Take this CRD schema:

```yaml
properties:
    myField:
        type: string
    myOtherField:
        type: string
```

Assume we have applied the following resource to our cluster:

```yaml
apiVersion: stable.example.com/v1
kind: MyCRD
myField: ""
```

Now, lets assume the CRD schema was changed to tighten validation:

```yaml
properties:
    myField:
        type: string
        minLength: 2
    myOtherField:
        type: string
```

When a controller attempts to change one field they get an error:

```yaml
apiVersion: stable.example.com/v1
kind: MyCRD
myField: ""
myOtherField: newly added field
```

```
* myField: Invalid value: "": myField in body should be at least 2 chars long
```

Even more sadly, users of SSA submitting just a patch with the new field get
rejected for fields that were not included in their patch. The following patch
yields the same error for SSA users:
```yaml
myOtherField: newly added field
```


The feature proposed in this KEP would allow this change, since `myField` was
unaltered from the original object. If `myField` was changed, the new value would have to pass the new validation rule to be written.

### Nested Value Validations 

OpenAPI-schemas support nested validations with logical operations:
- not
- oneOf
- anyOf
- allOf

To keep the semantics simple and implementation clear, schemas and validations
nested within the following will be not ratcheted:

- not
- oneOf
- anyOf

These constructs are not supported because its not clear that ratcheting would
yield correct behavior. CRD authors who use these complicated constructs are
encouraged to make use of CEL schemas.

For all these constructs, the conservative `DeepEqual` rule still applies 
to the subobject evaluating the rule, just not its nested fields.

The following will be ratcheted with the conservative ratcheting rule, and
their children will also be ratcheted.

- allOf

Ratcheting may be allowed for `allOf` because a schema with `allOf[a, b, c]` can 
be seen as equivalent to a single schema with the contents of `a, b, c` merged
together. Although allOf schemas may check different forms of the same condition 
multiple times (i.e. `minLength: 2` and `minLength: 5`, or multiple patterns),
the correctness of ratcheting unchanged objects still holds.

### XListType and XMapKeys

Errors thrown due to changing a list to a map-list with `x-list-type: map`or 
changing its `x-kubernetes-map-keys` validations will not be ratched. 

For example:
```yaml
- namespace: myNS
  name: myObj
- name: myObj
```

Changing `x-kubernetes-map-keys: [namespace] to `x-kubernetes-map-keys: [name, namespace]`
would trigger a new `Duplicate key` error not previously seen for this object.

These fields are not ratcheted because changing map keys or list type to a map alters 
the structure of the schema, and old values cannot be correlated to new. 

Additionally Server-Side-Apply does not tolerate updates that violate the map 
keys constraint: these properties are unsafe to change without bumping CRD 
version.

### Atomic Lists and Maps

Following Kubernetes convention that fields annotated `x-kubernetes-list-type`
or `x-kubernetes-map-type` as `atomic` change as a unit, the ratcheting rules
do not apply to values nested inside atomic lists and maps. But a DeepEqual 
comparison on the entire object can still ratchet errors for subfields.

### CEL Rules

Non-transitional CEL Rules (x-kubernetes-validations that do not 
make use of `oldSelf`) will be ratcheted, if the objects in scope to them 
can be correlated, and old is equal to new.

### Advanced Ratcheting

In this KEP we propose a conservative catch-all ratcheting rule that should
 provide a minimum level of ratcheting support for any validation rule in the 
 schema.

`DeepEqual(OldObject, NewObject)` requirement is very strong. But for some 
validation rules it is too strict. Consider the `required` error. This
validation is applied to the object owning the property. Thus, if a new field
becomes required the object cannot have other fields added onto it or changed
without also adding the newly required field.

Kubernetes has precedent for using a weaker form of racheting when validation is
strengthened:

```go
if !Validation(old) {
  // Ignore errors caused by !Validation(new)
}
```

An example of this is the structural schema requirement added when Kubernetes
went GA ([source](https://github.com/kubernetes/kubernetes/blob/12dc19d46fb97cbbfeb1e12b8a10ff7ae73d9515/staging/src/k8s.io/apiextensions-apiserver/pkg/apis/apiextensions/validation/validation.go#L1583-L1590)): 
```go
// requireStructuralSchema returns true if schemas specified must be structural
func requireStructuralSchema(oldCRDSpec *apiextensions.CustomResourceDefinitionSpec) bool {
	if oldCRDSpec != nil && specHasNonStructuralSchema(oldCRDSpec) {
		// don't tighten validation on existing persisted data
		return false
	}
	return true
}
```

It is not possible to generically allow this weaker form of ratcheting on all
types of validations - it may allow data that would have failed a validation
on an older schema and break controller workflows.

To enable schemas to add validations with more complicated ratcheting logic
than a DeepEqual check (e.g. to allow a new `minLength` restriction to allow any
value as long as it is getting closer the new minimum); we turn to CEL.

#### Ratcheting Rules in CEL

CRD Validation Rules currently have two types:

- Normal Rule (Applies to  all values)
- Transition Rule (Applies only on UPDATE in places where old could be correlated to new)

With only these rule types we are not able to express any logic of the form 
"old values must obey one schema, and new values another". We propose to add a 
new change to `ValidationRule` to enable this functionality for ratcheting:

```go
// ValidationRule describes a validation rule written in the CEL expression language.
type ValidationRule struct {
  // ...
	Rule string `json:"rule" protobuf:"bytes,1,opt,name=rule"`
	Message string `json:"message,omitempty" protobuf:"bytes,2,opt,name=message"`
	MessageExpression string `json:"messageExpression,omitempty" protobuf:"bytes,3,opt,name=messageExpression"`
	Reason *FieldValueErrorReason `json:"reason,omitempty" protobuf:"bytes,4,opt,name=reason"`
	FieldPath string `json:"fieldPath,omitempty" protobuf:"bytes,5,opt,name=fieldPath"`
  // ...

  // New
  // Normally, if `rule` references `oldSelf` then it is skipped on create, and
  // when an old value could not be found. If this field is true then the rule is 
  // always evaluated. `oldSelf = nil` on create or when an old value could not 
  // be correlated.
  //
  // Ensure your rule employs a nil check on `oldSelf` before using it in an
  // expression.
  AppliesOnCreate bool `json:"requiresOldValue"`
}
```

With this feature in place, users are able to express complicating ratcheting
logic in CEL, to express racheting conditions based on values they know
their controllers support:


```yaml
x-kubernetes-validations:
- rule: len(self) >= 5 || (oldSelf != nil && len(oldSelf) < len(self))
  appliesOnCreate: true
  message: New objects need names longer or equal to 5.
```

The above rule will allow strings that do not satisfy the new minimum length
requirement if they are longer than the old value; but all new objects must be
created with the new constraint.

### User Stories (Optional)

<!--
Detail the things that people will be able to do if this KEP is implemented.
Include as much detail as possible so that people can understand the ="how" of
the system. The goal here is to make this feel real for users without getting
bogged down.
-->

#### CRD Author Tightens a Field

Consider the current situation that plays when when a CRD author attempts to
shift left validation logic done by their controller by adding a validation to
their CRD:

1. CRD author gets complaints from users that they don't know their CRs have
invalid values until the CR is processed by the controller and updates the status.
2. CRD author notices OpenAPI schemas support adding value validations, and  happily annotates their schema.
3. Since only validation was changed: no fields were removed or had their types changed, the author considers the change too minor to require a new
CRD version.
4. During testing users apply the updated CRD. Some users have pre-existing resources that passed validation before, and now fail.
5. Controllers break since they can't update the Status of the custom resource: it fails validation.
6. User's entire workflow is now broken until they repair all broken custom resources.
7. User downgrades their CRD and complains to the CRD author
8. CRD author abandons their attempt to improve useability of their CR.

CRD authors tightening simple validation rules that their controllers tolerate
shouldn't require a version bump.

With the new feature the story becomes a success:

1. Users complain to CRD author. 
2. CRD author annotates schema
3. CRD author does not bump version
4. Users apply updated CRD schema
5. Controllers can successfully update the status field of the CR
6. User's workflow keeps humming
7. User compliments cunning CRD author for their brilliance
8. CRD author remembers to user value validations in their schemas more in the future.

#### K8s Update Tightens CRD validation

In this case the user has a CRD with a field with the following schema:

```yaml
type: string
format: dns1123subdomain
```

Kubernetes does not recognize this as a format. Today, Kubernetes treats
[unrecognized formats as a no-op](https://github.com/kubernetes/kubernetes/blob/f9f9e7177a328ae47bac6b823d57a99dfbfd310b/staging/src/k8s.io/apiextensions-apiserver/pkg/apiserver/validation/formats.go#L53-L65) and ignores them. So users would be able
to write fields not conforming to this format to custom resources.

If Kubernetes wishes to support this format, users workflows will break upon
upgrade until the offending resources and identified and repaired.

With this CRD Validation Ratcheting feature, Kubernetes would be able to
adopt new format strings safely without breaking most user workflows which 
primarily use UPDATE. Certain workflows using CREATE may still encounter a 
breaking api change upon validation tightening.

#### K8s Update Widens CRD Validation

In this case the user has a CRD with a field with following schema:

```yaml
type: string
format: byte
```

In k8s 1.27 and below, this format disallows empty strings due to an implementation detail. However, in native types like `Secret`, byte is disallowed. If Kubernetes had an update to CRDs to allow empty string for `byte`, this widens the validation for those CRDS which use it.

This is fine when upgrading to new versions if Kubernetes: all existing resources still pass validation. But downgrading from an update that widens validation can be understood 
semantically as an update that tightens validating.

For example if they had unreletated issues with the new Kubernetes update and 
wanted to downgrade:

1. User updates to new Kubernetes version
2. New CRs that were previously disallowed are allowed to be stored
3. User is disappointed with K8s version and downgrades
4. Controllers fail to update some resources, since they fail validation.

If the CRD Validation Ratcheting feature were enabled the following occurs:

1. User updates to new Kubernetes version
2. New CRs that were previously disallowed are allowed to be stored
3. User is disappointed with K8s version and downgrades
4. Controllers are allowed to update their resources, as long as all their changes are valid.

### Notes/Constraints/Caveats (Optional)

<!--
What are the caveats to the proposal?
What are some important details that didn't come across above?
Go in to as much detail as necessary here.
This might be a good place to talk about core concepts and how they relate.
-->

This KEP is scoped to CRDs, but if [KEP-3937](https://github.com/kubernetes/enhancements/pull/3938) is implemented then we should expect it to also affect native type validation.

### Risks and Mitigations

#### Detection of Breaking Schema Changes is Different

Some users might have a process to detect breaking schema changes by applying
their CRD and poking an existing resource. This KEP would make that process
no longer reliable. After this change, authors of CRs will need to do this test 
with a CREATE rather than an UPDATE.

##### Mitigation: Use CRD-Schema-Checker

[CRD-Schema-Checker](https://github.com/openshift/crd-schema-checker) may be
used to lint the CRD schema for breaking changes before applying it to a cluster.

##### Mitigation: Use New Objects

Alternatively, users may create new objects rather than updating old ones to 
test for breaking schema changes. This feature has no effect on the creation 
code path.


#### Not All Rules Can Be Correctly/Easily Ratcheted

Some usages of the OpenAPI schema are not possible to reliably ratchet without
creating confusing exceptions, or rewriting completely the validation code used
by Kubernetes. 

For others it is not clear to see why ratcheting them will yield correct results.

##### Mitigation: Blacklisted Validations

To mitigate this, validation failures in rules which have an
ancestor in the schema of one of these types will not be ratcheted:

- not: Supporting negation of rules for ratcheting would require a complete 
rewrite of the validation system.
- oneOf: Requires EXACTLY one alternative, so it is not allowed for the same reason as `not`  rules.
- anyOf: Blacklisted until we can understand the semantics better and see a large need to add support.
- x-kubernetes-validations which make use of `oldSelf`: To get ratcheting to 
work in an intuitive way with transition  rules would require a copy of the 
resource two states back. This is not feasible. Also, users can write ratcheting 
logic directly in CEL as a  transition rule if that was required.

##### Mitigation: Conservative Ratcheting Rule

For alpha our rule for ratcheting is fairly conservative: 

If a field does not change, validation rules attached to that rule which fail will
be ignored.

This is a correct rule due to the following observations:

1. Each validation rule is attributed to a field.
2. Each rule does not reference fields outside that field to which it is attributed.
This is true by construction for the case of the OpenAPI-schema validation fields,
and validated for the `x-kubernetes-...` CEL rules.
3. Errors raised by validation rules can be attributed to the rules they are attached to
(despite `fieldPath` on CEL Rules we can still determine the true field of the rule)
4. Some rules are attached to a field which captures more data than is referenced
by the rule, but that is OK it just makes the rule more conservative.

From this we can conclude that a given validation rule is completely dependent 
upon a subset of the information contained in the field to which it is attached.
If this subset of data did not change over the update, then the failure of the
rule was definitely due to pre-existing data.

For now we use DeepEqual to check equality since it is conservative. A semantic
check may be employed if it can be done cleanly and performantly.

For beta and beyond we will have an investigation and discussion of a more
permissive semantic.

There are a few cases which, to support ratcheting may require a less strict
ratcheting rule:

1. `maxItems` on an array - should we consider the "size" of the array to remain
unchanged if elements within it change without adding/removing anything? 
The current rule does not allow for this to be racheted.
2. `maxProperties` - similarly to above
3. `required` on an object: this rule is attached to an object, but pertains
to specific keys. Should we allow a non-existing field to ratchet if it is newly
made to be `required`? The current rule does not allow this.
## Design Details

<!--
This section should contain enough information that the specifics of your
change are understandable. This may include API specs (though not always
required) or even code snippets. If there's any ambiguity about HOW your
proposal will be implemented, this is the place to discuss them.
-->

At a high level, CR validation in kubernetes has four components:
1. `kube-openapi`-based `Validate` using `*spec.Schema`
2. StructuralSchema-based Meta, TypeMeta, ListSet/Map validation
3. CEL Validators using a CEL schema interface type
4. Embedded Objects

Each of these validation paths will need to have their `ValidateUpdate(old, new)`
codepath modified to:

1. Take an option to enable the feature
2. Ignore errors which occur both on the `old` and `new` objects
3. Return ignored errors in a separate list

### `kube-openapi` changes

The `kube-openapi` schema validator validates the standard OpenAPI/JSONSchema
elements of the CRD schema. This is basically everything except CEL rules,
and listset/map types.

The current SchemaValidator will be wrapped with a `RatchetingSchemaValidator`
which compares old/new values and ignores all errors on its subpath if they are
equal.

Supporting validation code within kube-openapi which spawn a new `SchemaValidator`
will need to be refactored to find the old value and use `RatchetingSchemaValidator`
when `RatchetingSchemaValidator.ValidateUpdate` is in its callstack.

anyOf, not, oneOf sections of logic will remain using the normal 
`SchemaValidator`.

### Structural-Schema-based validation changes

The two structural-schema based validation methods may need to
be refactored to support `ValidateUpdate` returning separate lists of errors.

[`ValidateListSetsAndMaps(fldPath *field.Path, s *schema.Structural, obj map[string]interface{}) field.ErrorList`](https://github.com/kubernetes/kubernetes/blob/694698ca3801fe5aa258d58c1e27ac7f88c29d35/staging/src/k8s.io/apiextensions-apiserver/pkg/apiserver/schema/listtype/validation.go#L28)

[`objectmeta.Validate(pth *field.Path, obj interface{}, s *structuralschema.Structural, isResourceRoot bool) field.ErrorList`](https://github.com/kubernetes/kubernetes/blob/92042fe6eac61b6dd4092f5c6737050a4a1c43e5/staging/src/k8s.io/apiextensions-apiserver/pkg/apiserver/schema/objectmeta/validation.go#L32)

> Note: ObjectMeta.Validate is a core validation function in kube with high risk
when modifying. For Alpha we will only ratchet errors from this function if the
entire metadata did not change, rather than field-by-field basis. Experimenting will
be done to gauge the feasibility/risk of adding ratcheting to metadata for beta.

#### Correlation of Old and New

For comparison of `old`/`new` `maps`/`lists`/`sets`, we need a method for 
correlation of the old and new elements. Whether that is normalizing the fields,
creating a map representation of the object, or otherwise. This may be tricky
to optimize, so care should be taken to meet performance goals.

### Cel-Validator changes

[`cel.Validate`](https://github.com/kubernetes/kubernetes/blob/2a50ef677ebf0729f176e852d0ad16950e122f6e/staging/src/k8s.io/apiextensions-apiserver/pkg/apiserver/schema/cel/validation.go#L148) is already written to validate an update, 
correlating old and new fields, so would only need to be modified to return an 
ignored error list when applicable.

## Test Plan

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

###### Schema-Changing Tests May Need Refactoring

This change could cause some previously valid tests of invalid schema changes to
fail. Those tests will have to be updated by changing to use create, or by
changing the invalid fields during an update.

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

- `k8s.io/apiextensions-apiserver/pkg/apiserver/schema/cel`: `06/14/2023` - `83.2`
- `k8s.io/apiextensions-apiserver/pkg/apiserver/schema/listtype`: `06/14/2023` - `95.8`
- `k8s.io/apiextensions-apiserver/pkg/apiserver/schema/objectmeta`: `06/14/2023` - `83.9`
- `k8s.io/apiextensions-apiserver/pkg/registry/customresource`: `06/14/2023` - `57.5`
- `k8s.io/apiextensions-apiserver/pkg/apiserver/validation`: `06/14/2023` - `85.8`
- `k8s.io/kube-openapi/pkg/validation/validate` : `06/14/2023` - `97.2`

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

- [k8s.io/apiextensions-apiserver/test/integration.TestRatchetingFunctionality](https://github.com/kubernetes/kubernetes/blob/3ee81787685e47a7a5da22423c8ca4455577ecb3/staging/src/k8s.io/apiextensions-apiserver/test/integration/ratcheting_test.go#L428)


##### e2e tests

<!--
This question should be filled when targeting a release.
For Alpha, describe what tests will be added to ensure proper quality of the enhancement.

For Beta and GA, add links to added tests together with links to k8s-triage for those tests:
https://storage.googleapis.com/k8s-triage/index.html

We expect no non-infra related flakes in the last month as a GA graduation criteria.
-->

- [test/e2e/apimachinery/crd_validation_ratcheting](https://github.com/kubernetes/kubernetes/blob/7353f52bc8ce110c78082690fc9c269465cbbaf9/test/e2e/apimachinery/crd_validation_ratcheting.go#L44): 

### Graduation Criteria


#### Alpha
- Feature Flag added
- Integration Tests implemented and enabled
- Incomplete-but-correct subset of ratcheting rules implemented under flag

#### Beta
- Gather feedback from developers
- Additional Testing added as needed
- CEL Validation Rules Ratcheting Implemented
- Schema ratcheting rules implemented under flag
- Detailed analysis of supported validation rules, whether/how they are ratcheted, and discussion
- Implementation of a weaker form of equality for cases where it is possible after investigation
- CEL expressivity to specify custom ratcheting validation

#### GA
- [x] Upgrade/Downgrade e2e tests
- [ ] Scalability Tests
- [ ] Ratcheting to include `allOf` subschemas
- [x] No non-infra related flakes in the last month

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
No change in how users upgrade/downgrade their clusters. This feature may remove
complexity by removing risk that a tightened validation on Kubernetes' part
does not break workflow.


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
N/A

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
  - Feature gate name: `CRDValidationRatcheting`
  - Components depending on the feature gate: `apiextensions-apiserver`, `kube-apiserver`
- [ ] Other
  - Describe the mechanism:
  - Will enabling / disabling the feature require downtime of the control
    plane?
  - Will enabling / disabling the feature require downtime or reprovisioning
    of a node?

###### Does enabling the feature change any default behavior?

<!--
Any change of default behavior may be surprising to users or break existing
automations, so be extremely careful here.
-->

Yes, enabling the feature will silence errors for unchanged fields.

###### Can the feature be disabled once it has been enabled (i.e. can we roll back the enablement)?

<!--
Describe the consequences on existing workloads (e.g., if this is a runtime
feature, can it break the existing applications?).

Feature gates are typically disabled by setting the flag to `false` and
restarting the component. No other changes should be necessary to disable the
feature.

NOTE: Also set `disable-supported` to `true` or `false` in `kep.yaml`.
-->

Yes, doing so will restore prior functionality. Resources that were successfully updated while the feature was enabled may still be invalid at rest, but this condition is unchanged from before the feature was enabled, and
the system should behave as expected.

###### What happens if we reenable the feature if it was previously rolled back?

Nothing. Future updates to Custom Resources that may have failed validation
before will now succeed if they did not change the failing fields.

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

We will add an integration test to ensure that the feature is disabled when the feature gate is off.
Additionally an integration test that shows CRs which would be ratcheted can 
have their `status` changed when the feature is enabled.

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
This feature will not impact rollouts or already-running workloads.

###### What specific metrics should inform a rollback?

<!--
What signals should users be paying attention to when the feature is young
that might indicate a serious problem?
-->
If `apiextensions_apiserver_update_ratcheting_time` is taking a long time (order of 100ms)

###### Were upgrade and rollback tested? Was the upgrade->downgrade->upgrade path tested?

<!--
Describe manual testing that was done and the outcomes.
Longer term, we may want to require automated upgrade/rollback tests, but we
are missing a bunch of machinery and tooling and can't do that now.
-->

No, a Kubernetes upgrade/downgrade operation is not expected to affect this feature.
For CRDs being upgraded/downgraded, there will be automated tests.

###### Is the rollout accompanied by any deprecations and/or removals of features, APIs, fields of API types, flags, etc.?

<!--
Even if applying deprecation policies, they may still surprise some users.
-->
No.

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

We will have a metric to measure the time performing the ratcheting comparison
for this feature. Operators can check if the time is non-zero.

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
- [x] Other (treat as last resort)
  - Details: User can update a CRD to tighten the requirements of one of its fields. If an update to an object that would now fail the new validation succeeds, the feature is enabled.

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

- [x] Metrics
  - Metric name: apiextensions_apiserver_update_ratcheting_time (histogram, how long we spend in addition to normal validation to perform ratcheting comparisons)
  - [Optional] Aggregation method:
  - Components exposing the metric: kube-apiserver
- [ ] Other (treat as last resort)
  - Details:

###### Are there any missing metrics that would be useful to have to improve observability of this feature?

<!--
Describe the metrics themselves and the reasons why they weren't added (e.g., cost,
implementation difficulties, etc.).
-->
No.

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
No.

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
No.

###### Will enabling / using this feature result in introducing new API types?

<!--
Describe them, providing:
  - API type
  - Supported number of objects per cluster
  - Supported number of objects per namespace (for namespace-scoped objects)
-->
No.

###### Will enabling / using this feature result in any new calls to the cloud provider?

<!--
Describe them, providing:
  - Which API(s):
  - Estimated increase:
-->
No. Possibly only indirectly as a result of writes succeeding instead of failing.

###### Will enabling / using this feature result in increasing size or count of the existing API objects?

<!--
Describe them, providing:
  - API type(s):
  - Estimated increase in size: (e.g., new annotation of size 32B)
  - Estimated amount of new objects: (e.g., new Object X for every existing Pod)
-->
No.

###### Will enabling / using this feature result in increasing time taken by any operations covered by existing SLIs/SLOs?

<!--
Look at the [existing SLIs/SLOs].

Think about adding additional work or introducing new steps in between
(e.g. need to do X to start a container), etc. Please describe the details.

[existing SLIs/SLOs]: https://git.k8s.io/community/sig-scalability/slos/slos.md#kubernetes-slisslos
-->
Performance of any writes to a custom resource may be impacted. Benchmarks must
be taken to measure magniture.

###### Will enabling / using this feature result in non-negligible increase of resource usage (CPU, RAM, disk, IO, ...) in any components?

<!--
Things to keep in mind include: additional in-memory state, additional
non-trivial computations, excessive access to disks (including increased log
volume), significant amount of data sent and/or received over network, etc.
This through this both in small and large cases, again with respect to the
[supported limits].

[supported limits]: https://git.k8s.io/community//sig-scalability/configs-and-limits/thresholds.md
-->
Should have benchmarks before beta, but expecting to see some measureable impact to writes only to CRDs.

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

The same way any write to apiserver would.

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
None.

###### What steps should be taken if SLOs are not being met to determine the problem?

Disable the feature.

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

### CRDs Opt-in to Ratcheting Functionality

One alternative to appling ratcheting generally to all crds when the feature 
is enabled would be to add something like an `x-kubernetes-...` extension to 
CRDs to allow the individual CRDs to decide to ratchet themselves.

This solves one of the goals of this KEP of allowing CRD authors more flexibility
in modifying their schemas. But this does not solve the other goal of enabling
Kubernetes developers from fixing bugs with/extending the supported OpenAPI 
schema.

For example:

1. The discrepency between `byte` as a format for CRDs, and `byte` used by native
types cannot be fixed if this feature is opt-in.
2. New string formats cannot be added like `dns1123subdomain`, since to add it
would break any CRD not opted into ratcheting.

Another drawback is `x-kubernetes-ratcheting: true` cannot be disabled without 
problems for the user if other rules were added at the same time.

One might say to explicitly only ratchet things that Kubernetes' developers have
modified but that increases maintenance burden on Kubernetes developers, and
causes confusing behavior on the user side when parts of your CRD are ratcheting
that you did not enable.

### Offline Pass to Flag Invalid Objects at Rest During Upgrade

An alternative suggested to this process would be to provide a tool to scan
all objects in etcd, and flag to the user which ones are currently failing validation
at rest. This would help operators clean up a broken workflow caused by objects made
invalid by a validation tightening.

This is undesierable for a few reasons related to UX:

1. Workflows are still broken for a period after the update.
2. Updates to CRD Schemas are still very disruptive, remaining a barrier to validation adoption
3. Operators who may not be the original author of a resource will have to correct the objects themselves manually.

### Post-Process Errors

The basic idea is to add a post-processing step to the `Validate(Update)` methods of the CRD strategy. Given a list of `StatusError`, we check the field path associated with the error, if it is unchanged from old to new then the error is changed into a warning.

#### Drawbacks
##### Robustness of paths returned by OpenAPI Schema Validator

This approach would make the field paths returned by the OpenAPI schema validator
important for correctness. Previously they have only been used informationally
to the user. 

##### Evaluation of JSON Paths

This approach also introduces complexity evaluating the JSON paths. It may take
a refactor of schema validator to return paths good enough to use to evaluate
against old and new objects.

##### Correlating Errors To Fields

There are a few problems with using the FieldPath associated with the error
to potentially ignore errors:

1. Prepending item to atomic list.

This case changes the index of all items, and the list is atomic, so it is not possible to ratchet individual items in an atomic list.

2. Field Path containing list-type=map still contains a position index relative to the new object. 

We will have to look up the map key in the new object to find the element's map key. Using the map key we can find the associated element in the old object.

3. Field Path in Errors from CEL Rules may be a lie: https://github.com/kubernetes/kubernetes/pull/118041

The field to which the CEL rule in its `x-kubernetes-validations` list must 
contain all fields referred by the CEL rule; so the CEL error logic can have to 
be modified to also include the rule's original location.

### Different Ratcheting Rule Per Value Validation

An alternative to the weaker form ratcheting described in the proposal would be
to apply the DeepEqual rule to a smaller projection of the object being validated.

i.e. MaxLength would use projection `len(obj)` since it only looks at that value,
and if `len(oldObj) == len(newObj)`, then we would allow ratcheting.

The following projections were considered:

| Field               | Type     | Projection                   
|---------------------|----------|------------------------------
| Format              | string   | Identity                     
| Maximum             | *float64  | Identity                    
| ExclusiveMaximum    | bool     | Identity                    
| Minimum             | *float64  | Identity                    
| ExclusiveMinimum    | bool     | Identity                    
| MaxLength           | *int64   | len(obj) -> int             
| MinLength           | *int64   | len(obj) -> int             
| Pattern             | string   | Identity                  
| MultipleOf          | *float64  | Identity                 
| Enum                | []JSON   | Identity                  
| Nullable            | bool     | Identity                  
| MaxItems            | *int64   | len(obj) -> int             
| MinItems            | *int64   | len(obj) -> int             
| UniqueItems         | bool     | Identity       
| MaxProperties       | *int64   | len(obj) -> int             
| MinProperties       | *int64   | len(obj) -> int             
| Required            | []string | Keys(obj) -> []string (also can choose to ratchet on per-key presence)       


#### Drawbacks

This idea has a surprising UX. It makes no sense for `MaxLength: 3` to ratchet
an update `hello` -> `fives`  but not `hello` -> `longer string`.

The accepted idea has the advantage of being the precedent in Kubernetes for how
we treat validations of persisted data. For example, schemas in CRDs are only
validated to be structural [if the old object was already structural](https://github.com/kubernetes/kubernetes/blob/12dc19d46fb97cbbfeb1e12b8a10ff7ae73d9515/staging/src/k8s.io/apiextensions-apiserver/pkg/apis/apiextensions/validation/validation.go#L1583-L1590).

### Ratcheting Within Nested Value Validations

#### Not

Intuitively, ratcheting for `not` would ignore errors only if we fail 
the same way as the old object. Any failing constraint that depends on a 
changed value should not be racheted.

We cannot apply the weaker ratcheting rule to the top level `not`: `!Not(old) || Not(new)`.
The rule would allow changes that did not fail the same way on the old object
to be ratcheted.

#### OneOf

To ratchet `oneOf`, would require the ability to ratchet `not`. Since that is
not possible, we do not apply a weakened ratcheting rule to oneOf. We also
do not ratchet the errors within children schemas.

<!-- OneOf(obj, S1, S2, S3) = AnyOf(
  AllOf(S1(obj), Not(S2(obj)), Not(S3(obj))), 
  AllOf(S2(obj), Not(S1(obj)), Not(S3(obj))), 
  AllOf(S3(obj), Not(S1(obj)), Not(S2(obj))),
) -->

One way to ratchet OneOf would be to only allow ratcheting within the schema
that passed for the old value. But this seems like awkward UX and no one has
asked for it.

### Weaker Ratcheting Rule

The following fields are all the value validations allowed in our schemas:

- Format
- Maximum
- ExclusiveMaximum
- Minimum
- ExclusiveMinimum
- MaxLength
- MinLength
- Pattern
- MultipleOf
- Enum
- MaxItems
- MinItems
- UniqueItems
- MaxProperties
- MinProperties
- Required
- XValidations (CEL rules, excluding rules using oldSelf)

We will apply a weaker validation rule whenever these are applied: these 
validations will only apply if they had passed for the old value.

For example:
```go
func validateFormat(new, old interface{}, typ, format string) *field.Error {
  if MatchesFormat(typ, format, old) && !MatchesFormat(typ, format, new) {
    return field.Invalid(...)
  }
  return nil
}
```
#### Drawbacks

##### Allows Arbitrary Invalid Data

This rule would default all schemas into allowing artbirary unknown data (of
the correct type). This seems dangerous from a security perspective. Controllers
should at least be able to reasonably expect that all existing objects went 
through validation of a published schema.

##### AllOf

Applying the weakened ratcheting rule would mean to allow the object to pass if
`!AllOf(old) || AllOf(new)`. 

This would ratchet if ANY of the nested validations fail for the old value. 
This is not a viable rule since it is weaker than each nested rule being 
ratcheted individually: it would allow changes that the nested validations did
not permit.

##### AnyOf

AnyOf is the non-exclusive OR of several schemas. Like allOf, to weaken the top
level `anyOf` directive would be weaker than ratcheting its individual children.

## Infrastructure Needed (Optional)

<!--
Use this section if you need things from the project/SIG. Examples include a
new subproject, repos requested, or GitHub details. Listing these here allows a
SIG to get the process for these resources started right away.
-->