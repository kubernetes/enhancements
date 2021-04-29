<!--
**Note:** When your KEP is complete, all of these comment blocks should be removed.

To get started with this template:

- [x] **Pick a hosting SIG.**
  Make sure that the problem space is something the SIG is interested in taking
  up. KEPs should not be checked in without a sponsoring SIG.
- [ ] **Create an issue in kubernetes/enhancements**
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
- [ ] **Fill out this file as best you can.**
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
# KEP-1929: Built-in declarative defaults

<!-- toc -->
- [Release Signoff Checklist](#release-signoff-checklist)
- [Summary](#summary)
- [Motivation](#motivation)
  - [Goals](#goals)
  - [Non-Goals](#non-goals)
- [Proposal](#proposal)
  - [Changes to CRD Normalization](#changes-to-crd-normalization)
  - [Changes to kubebuilder/default-gen](#changes-to-kubebuilderdefault-gen)
  - [Default publishing](#default-publishing)
  - [Marker format](#marker-format)
  - [Examples](#examples)
    - [Non-pointer structs](#non-pointer-structs)
    - [Struct pointers](#struct-pointers)
    - [Non-pointer scalar fields](#non-pointer-scalar-fields)
    - [Lists](#lists)
    - [Lists without default](#lists-without-default)
    - [String Maps](#string-maps)
    - [String maps without default](#string-maps-without-default)
  - [Risks and Mitigations](#risks-and-mitigations)
- [Design Details](#design-details)
  - [Test Plan](#test-plan)
  - [Graduation Criteria](#graduation-criteria)
- [Production Readiness Review Questionnaire](#production-readiness-review-questionnaire)
  - [Feature Enablement and Rollback](#feature-enablement-and-rollback)
  - [Rollout, Upgrade and Rollback Planning](#rollout-upgrade-and-rollback-planning)
  - [Monitoring Requirements](#monitoring-requirements)
  - [Dependencies](#dependencies)
  - [Scalability](#scalability)
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

- [x] (R) Enhancement issue in release milestone, which links to KEP dir in [kubernetes/enhancements] (not the initial KEP PR)
- [x] (R) KEP approvers have approved the KEP status as `implementable`
- [x] (R) Design details are appropriately documented
- [x] (R) Test plan is in place, giving consideration to SIG Architecture and SIG Testing input
- [x] (R) Graduation criteria is in place
- [ ] (R) Production readiness review completed
- [ ] Production readiness review approved
- [ ] "Implementation History" section is up-to-date for milestone
- [ ] User-facing documentation has been created in [kubernetes/website], for publication to [kubernetes.io]
- [ ] Supporting documentation—e.g., additional design documents, links to mailing list discussions/SIG meetings, relevant PRs/issues, release notes

[kubernetes.io]: https://kubernetes.io/
[kubernetes/enhancements]: https://git.k8s.io/enhancements
[kubernetes/kubernetes]: https://git.k8s.io/kubernetes
[kubernetes/website]: https://git.k8s.io/website

## Summary

Currently, built-in API types are written as Go structures annotated
with markers to add additional semantics. This method has inspired
controller-tools who has re-used and improved on that technique for
building CRDs. We're proposing using the `// +default` marker to
declaratively define defaults (and have kubebuilder use that as well)
since that mechanism still doesn't exist for built-in. This proposal
tries to bridge that gap.

## Motivation

Having a new `// +default` marker for built-in types has several
important benefits:

- It's more convenient for types authors.
- The types and their behaviors are easier to understand since they are
described together.
- They are less error-prone, and discourage people from using anti-patterns
like "cross-field dependency defaults" (default depends on another
field), non-deterministic or complicated defaults. These patterns
might be useful in limited circumstances, but it should be easier to do
the right thing in most cases.
- Removes the inconsistency with CRDs that already support this
mechanism, and improves embedding of types from k8s.io/api into CRDs,
which is currently cumbersome.
- Solves bug where associative keys needs to either be required or
defaulted. Default keys will now be able to be specified in the OpenAPI
for built-in types, and SSA will be able to fill in the missing keys.

### Goals

The goal is to add a new `// +default` marker to our current built-in Go
IDL. That marker will be transformed into the OpenAPI `default` field
and then routed to defaulting functions so that defaulting can be done
declaratively. This new marker would also be introduced in kubebuilder,
and we will make the behavior equivalent between built-in types and CRDs
so that built-in types can be directly used in kubebuilder, and the
declarative defaults will be automatically carried along.

### Non-Goals

While this proposal is probably paving the way for additional OpenAPI
based validation, validation markers for built-in types is entirely
out-of-scope for this project.

This proposal also doesn't intend to remove the existing defaulting
mechanism since some defaulting still requires non-static values that
need to be computed and can't be defined declaratively. We could replace
some defaulting that doesn't require to be done as code though.

## Proposal

It is important to us that the OpenAPI definition of a type should
result in the same behavior, whether the type is built-in or a CRD.

In order to do that, we need to change how CRDs are defaulted, and place
specific requirements on definition of Golang types (specifically around
omitempty on non-pointer scalars, and an implicit default of {} on
non-pointer struct fields), and create a defaulting mechanism for
built-in that would behave the same way.

We will depend on tooling (defaulter-gen and kubebuilder) to enforce the
new type requirements (via warnings or errors), to implicitly
generate certain defaults in order to normalize the behaviour of
built-in types and CRDs around zero values and absence/presence.

Rule-of-thumb: when clients use the canonical go serialization, both
clients for CRDs and built-ins should result in the same defaulting
behaviour. We accept drift in the defaulting behaviour for JSON or YAML
manifests that don't roundtrip through Golang types before being sent to
the server (unstructured).

### Changes to CRD Normalization

Null values in CRDs are currently not defaulted. On the other hand,
`null` typically means "Devoid of
value"(link)[https://yaml.org/type/null.html] in json/yaml, though it is
treated as different from not present ("a null value is valid and
different from not having that key in the mapping"). JSon Merge patch
interprets a "null" value for a field as a means to delete it: "Null
values in the merge patch are given special meaning to indicate the
removal of existing values in the
target."(link)[https://tools.ietf.org/rfc/rfc7386.txt#:~:text=Null%20values%20in%20the%20merge]

On the other hand, while `{"field": null}` and `{}` probably have the same
meaning, they do not "look" the same and are hard to compare for
clients. They are also currently treated differently by Kubernetes for
defaulting purposes.

Currently, null values trigger an error when nullable is false and don't
get defaulted if a default is available.

We propose:
1. to start defaulting fields (both properties and additionalProperties)
   with null values if a default is available and nullable unset or
   false, or to remove that field if no default exists.
2. to start defaulting items of arrays with null values if a default is
   available and nullable unset or false, or to leave that null value in
   the arrays and let validation fail if the field is not nullable.

This means that we will start accepting inputs that were rejected in the
past. There is code that would work with a new server but not with an
old server. We assume that existing APIs that use nullable are happy
with null values. Also, people can remove the nullable attribute if they
want null values to start being removed. This will help aligning CRDs
and built-in types (there are no ways to set nullable to true for
built-in types).

Note that we still consider `{}` (and 0, 0.0, "", false, []) to be
different from `null`, so `{"foo": {}}` is different from `{}` or from
`{"foo": null}` (but the latter two are considered equal).

As described by @sttts on many occasions, the immutability is blocked
because we don't know how to distinguish between null and
unspecified. We believe that this mechanism will be able to unblock it.

### Changes to kubebuilder/default-gen

For built-in types defaulting today takes place after marshalling Golang
structs to JSON or ProtoBuf and then unmarshalling, back into Golang
structs. This means that: unset non-pointer structs are always present
after unmarshalling as empty struct ({}) zero valued non-pointer structs
are marshalled as empty struct despite omitempty unset scalars without
omitempty are indistinguishable from zero values after unmarshalling and
are hence defaulted if a default is available. CRDs defaulting does not
default zero values though.

To normalize the behaviour, we propose:
1. we auto-generate `default: {}` for non-pointer struct and forbid the
`+default` marker
2. we forbid or warn (liniting) about omitempty on non-pointer structs
3. we require omitempty on scalars with default different from the
zero-value
4. we auto-generate `default: <zero-value>` for scalars without
omitempty and forbid the `+default` marker

We accept that manifests that don't roundtrip through Golang types
(i.e. omitempty tags cannot lead to removal of zero valued fields) on
the client, but are sent to the server via a dynamic client will have
different defaulting behaviour between CRDs and built-ins (there is an
example further down).

### Default publishing

CRDs do not publish default via the OpenAPI spec on the /openapi/v2
endpoint today, but we have to decide what to do with the new default
values of built-in types. The kube-openapi generated schemas will
include them.

We propose to:
1. to allow the CRDs to publish the defaults field which helps the client to set the <zero-value>.
2. filter out defaults in the built-in type schemas too in a first step
3. but possibly reconsider this behaviour in the future when the
necessity of defaults for client-side merging (e.g. through kustomize)
and influences on the ecosystem of default mis-use is better understood.

### Marker format

The marker will have the following format:
```
// +default=<any>
```
which describes the value that the server will use if the field has a
zero-value.

`<any>` has to be in one-line JSON format (giving us the ability to move
to YAML later while maintaining compatibility), representing the actual
value to use by default.

The JSON value will be unmarshalled by the apiserver when initializing,
and that value will be then deep-copied into the object when defaulting
happens.

Namely, this will allow to preserve the concise semantics that you want
for most values:

```golang
type Foo struct {
  // +default=32
  Integer int

  // +default="bar"
  String string

  // +default=["popcorn", "chips"]
  StringList []string
}
```
while still allowing more complex expressions:
```
// +default=[{"port": 80, "name": "http"}]
```


### Examples

Here are examples of the impact of the changes to CRD normalization,
kubebuilder/openapi schema generation and the suggested semantics of the
+default marker.

#### Non-pointer structs

Given the following type::
```golang
type Root struct {
  Entry SubLevel
}

type SubLevel struct {
  // +default="default-name"
  Name string `json:"name,omitempty"`
  // +default=0
  Number int
}
```

We would generate the following OpenAPI definition, both for CRD types (by kubebuilder)
and built-in types:
```yaml
Root:
  type: object
  default: {}
  properties:
    entry:
      default: {}
      type: object
      properties:
        name:
          type: string
          default: "default-name"
        number:
          type: integer
          default: 0
```

which would have the following behavior, both for CRD types and built-in types:
```python
>>> default('null')
{"entry": {"name": "default-name", "number": 0}}
>>> default('{}')
{"entry": {"name": "default-name", "number": 0}}
>>> default('{entry: null}')
{"entry": {"name": "default-name", "number": 0}}
>>> default('{entry: {}}')
{"entry": {"name": "default-name", "number": 0}}
>>> default('{entry: {name: other-name}}')
{"entry": {"name": "other-name", "number": 0}}
>>> default('{entry: {name: "", number: 0}}') # empty string is different from null or unspecified
{"entry": {"name": "", "number": 0}}
```

On the other hand, this is forbidden:
```golang
Type Root struct {
  // Defaults on non-pointer structs are FORBIDDEN:
  // +default={"name": "entry", "number": 12}
  Entry SubLevel
}

Type SubLevel struct {
  Name string `json:"name,omitempty"`
  Number int
}
```

#### Struct pointers

Struct pointers are allowed to have a specific default since they are distinct
from unspecified non-pointers structs. Modifying the type from the
previous example:
```golang
type Root struct {
  // This is now a pointer, we can provide a default:
  // +default={"name": "pointer-name"}
  Entry *SubLevel
}

type SubLevel struct {
  // +defaut="default-name"
  Name string
  // +default=0
  Number int
}
```

We would generate the following OpenAPI definition, both for CRD types (by kubebuilder)
and built-in types:
```yaml
Root:
  type: object
  default: {}
  properties:
    entry:
      $ref: SubLevel
      default: {"name": "pointer-name"}
SubLevel:
  type: object
  properties:
    name:
      type: string
      default: "default-name"
    number:
      type: integer
      default: 0
```

which would have the following behavior, both for CRD types and built-in types:
```python
>>> default('null')
{"entry": {"name": "pointer-name", "number": 0}}
>>> default('{}')
{"entry": {"name": "pointer-name", "number": 0}}
>>> default('{entry: null}')
{"entry": {"name": "pointer-name", "number": 0}}
>>> default('{entry: {}}')
{"entry": {"name": "default-name", "number": 0}}
>>> default('{entry: {name: other-name}}')
{"entry": {"name": "other-name", "number": 0}}
```

#### Non-pointer scalar fields

Non-pointer scalar fields have a specific unmarshalling mechanism in Go,
since they always have a value, even if the zero value is unset on the
wire. CRD defaulting will not default a zero value though.

```golang
type Object struct {
     // This field is omit-empty, and hence Golang unmarshalling drops a zero-value.
     // +default="default-name"
     Name string `json:"name,omitempty"`

     // Non-pointer scalar fields without omitempty require zero-value default.
     // +default=0
     Defaulted int
}
```

Non-pointer scalar fields that have a default must be omitempty (unless
their default is the zero-value). Kubebuilder and built-in defaulting
generators should both require this. Our type above fulfills this
requirement, in contrast to

```golang
type Invalid struct {
     // This field is NOT omit-empty, and CRD and built-in defaulting would differ,
     // even when going through Golang types on the client
     // +default="default-name"
     Name string `json:"name"`
}
```

The valid Object type would generate the following OpenAPI:
```yaml
Object:
  type: object
  properties:
    name:
      type: string
      default: "default-name"
    defaulted:
      default: 0
      type: integer
```

Which, for both CRD types and built-in types, would generate the following behavior:
```python
>>> default('{}')
{"entry": {"name": "default-name", "number": 0}}
>>> default('{name: other-name}')
{"name": "other-name", "defaulted": 0}
# For built-in, additionally, we would have:
>>> default('{name: ""}')
{"name": "default-name", "defaulted": 0}
```

Note that a simplified version, which deserializes on every call, of the generated code for
built-ins would look like this:
```golang
func GenerateDefault(obj *Object) {
     if obj.Name == "" {
          if err := json.Unmarshal([]byte(`"default-name"`), &obj.Name); err != nil {
               panic(err)
          }
     }
     if obj.Number == 0 {
          if err := json.Unmarshal([]byte(`0`), &obj.Number); err != nil {
               panic(err)
          }
     }
}
```

It's still possible to specify an invalid zero-value in built-in types
embedded into a CRD, but client-go will drop these values, thanks to
`omitempty`. On the other hand, a manifest in JSON

{"entry": {"invalid": "", "number": 0}}

sent to the server would result in different behaviour for built-ins

```python
>>> default('{name: ""}')
{"name": "default-name", "defaulted": 0}
```

and CRDs:

```python
>>> default('{name:"", number:0}')
{"name": "", "defaulted": 0}
```

#### Lists

```golang
type Object struct {
    List []Item
}

// +default="apple"
type Item string
```

Gives the following OpenAPI:
```yaml
Root:
  type: object
  default: {}
  properties:
    list:
      type: array
      items:
        type: string
        default: "apple"
```

And the defaulting will have the following behavior:
```python
>>> default('{list: [null, foo]}')
{"list": ["apple", "foo"]}
```

#### Lists without default

```golang
type Object struct {
    List []Item
}

type Item string
```

Gives the following OpenAPI:
```yaml
Root:
  type: object
  default: {}
  properties:
    list:
      type: array
      items:
        type: string
```

And the defaulting will have the following behavior, null is conserved:
```python
>>> default('{list: [null, foo]}')
{"list": [null, "foo"]}
```

#### String Maps

```golang
type Object struct {
    Mapping map[string]LabelValue
}

// +default="banana"
type LabelValue string
```

Gives the following OpenAPI:
```yaml
Root:
  type: object
  default: {}
  properties:
    mapping:
      type: object
      additionalProperties:
        type: string
        default: "banana"
```

And the defaulting will have the following behavior:
```python
>>> default('{mapping: {foo: null, bar: apple}')
{"mapping": {"foo": "banana", "bar": "apple"}
```

#### String maps without default

```golang
type Object struct {
    Mapping map[string]LabelValue
}

type LabelValue string
```

Gives the following OpenAPI:
```yaml
Root:
  type: object
  default: {}
  properties:
    mapping:
      type: object
      additionalProperties:
        type: string
```

And the defaulting will have the following behavior, this time null is removed:
```python
>>> default('{mapping: {foo: null, bar: apple}')
{"mapping": {"bar": "apple"}
```


### Risks and Mitigations

The main risk is that we could end-up with different defaults for
existing types which would be extremely confusing for our users. This
can be mitigated by double-checking the existing tests and adding new
tests for gap in coverage.

## Design Details

We are proposing to modify the defaulting generator to also generate new
defaulting functions, which will use the marker to generate the default
and apply them to the fields directly if the fields are set to their
zero-value. These new defaulting functions will be called, inside the
existing defaulting logic, right after the existing manual defaulting
functions. The defaulting call will stay unchanged.

Because we also want server-side apply to be aware of these defaults,
the OpenAPI generator will also use this information to fill the
defaults fields in the OpenAPI. Then, Server-side apply will be able to use
that additional information to default associative keys as needed.

In other words, the plan would be the following:
1. Update defaults generator to parse the `+default` marker and generate
the additional defaulting functions, automatically call them prior to
the existing defaulting functions
2. Update kube-openapi to parse the `+default` marker and include it in
the OpenAPI definition.
3. Implement the new CRD semantics for properties with nullable unset or
false (which previously would have resulted in a validation error that
null is not a valid value):
- properties and list items with a default have the default applied,
replacing the explicit null value
- list items without a default let the explicit null value remain,
resulting in a validation error (matches current behavior)
- properties without a default have the explicit null value removed
4. Add additional "default" field to sigs.k8s.io/structured-merge-diff
internal schema, and parse that field from the OpenAPI, use it when the
map key is missing. The code for this already exists in a
[PR](https://github.com/kubernetes-sigs/structured-merge-diff/pull/83#issuecomment-498410353)
5. Update some of the existing built-in APIs to specify their default
when they are strictly declarative, probably focusing first on defaults
that are part of the associative keys (e.g. `protocol: "TCP"`).

Note that we are keeping the existing built-in defaulting
mechanism. This is merely adding an extra-layer on top of that which
makes type authors life easier.

### Test Plan

As soon as this is implemented and working, we can start removing a lot
of the existing defaulting code and replace it with this values, and we
can re-use the existing defaulting tests, make sure that the behavior
doesn't change when moving from code-based defaults to declarative
default. As a one-time test, we could try to panic in specific places of
the defaulters to verify that the defaulting have happened before.

### Graduation Criteria

As an internal feature, this will go straight to stable.

## Production Readiness Review Questionnaire

### Feature Enablement and Rollback

* **How can this feature be enabled / disabled in a live cluster?**
  - [x] Feature gate (also fill in values in `kep.yaml`)
    - Feature gate name: DeclarativeDefaults

* **Does enabling the feature change any default behavior?** No

* **Can the feature be disabled once it has been enabled (i.e. can we roll back
  the enablement)?** Yes

* **What happens if we reenable the feature if it was previously rolled back?** Nothing

* **Are there any tests for feature enablement/disablement?** No

### Rollout, Upgrade and Rollback Planning

* **How can a rollout fail? Can it impact already running workloads?** No

* **What specific metrics should inform a rollback?** None

* **Were upgrade and rollback tested? Was the upgrade->downgrade->upgrade path tested?** No

* **Is the rollout accompanied by any deprecations and/or removals of features, APIs,
fields of API types, flags, etc.?** No

### Monitoring Requirements

* **How can an operator determine if the feature is in use by workloads?** N/A

* **What are the SLIs (Service Level Indicators) an operator can use to determine
the health of the service?** None

* **What are the reasonable SLOs (Service Level Objectives) for the above SLIs?** N/A

* **Are there any missing metrics that would be useful to have to improve observability
of this feature?** No

### Dependencies

* **Does this feature depend on any specific services running in the cluster?** No

### Scalability

* **Will enabling / using this feature result in any new API calls?** No

* **Will enabling / using this feature result in introducing new API types?** No

* **Will enabling / using this feature result in any new calls to the cloud
provider?** No

* **Will enabling / using this feature result in increasing size or count of
the existing API objects?** No

* **Will enabling / using this feature result in increasing time taken by any
operations covered by [existing SLIs/SLOs]?** No

* **Will enabling / using this feature result in non-negligible increase of
resource usage (CPU, RAM, disk, IO, ...) in any components?** No

## Implementation History
