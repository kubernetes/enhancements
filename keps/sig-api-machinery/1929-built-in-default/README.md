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
  - [CRD Normalization](#crd-normalization)
  - [Marker format](#marker-format)
  - [Built-in Defaulting](#built-in-defaulting)
  - [Examples](#examples)
    - [Non-pointer structs](#non-pointer-structs)
    - [Struct pointers](#struct-pointers)
    - [Non-pointer fields](#non-pointer-fields)
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

- [ ] (R) Enhancement issue in release milestone, which links to KEP dir in [kubernetes/enhancements] (not the initial KEP PR)
- [ ] (R) KEP approvers have approved the KEP status as `implementable`
- [ ] (R) Design details are appropriately documented
- [ ] (R) Test plan is in place, giving consideration to SIG Architecture and SIG Testing input
- [ ] (R) Graduation criteria is in place
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
mechanism.
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
based validation, validation markers for built-in is entirely
out-of-scope for this project.

This proposal also doesn't intend to remove the existing defaulting
mechanism since some defaulting still requires non-static values that
need to be computed and can't be defined declaratively. We could replace
some defaulting that doesn't require to be done as code though.

## Proposal

It is important to us that the OpenAPI definition of a type should
result in the same behavior, whether the type is built-in or a CRD.

In order to do that, we need to change how CRDs are defaulted, and
create a defaulting mechanism for built-in that would behave the same
way.

### CRD Normalization

Null values in CRDs are currently not defaulted. On the other hand,
`null` typically means "Devoid of
value"(link)[https://yaml.org/type/null.html] in json/yaml, though it is
treated as different from not present ("a null value is valid and
different from not having that key in the mapping"). JSon Merge patch
interprets a "null" value for a field as a means to delete it: "Null
values in the merge patch are given special meaning to indicate the
removal of existing values in the
target."(link)[https://tools.ietf.org/rfc/rfc7386.txt#:~:text=Null%20values%20in%20the%20merge]

On the other hand, while `{field: null}` and `{}` probably have the same
meaning, they do not "look" the same and are hard to compare for
clients. They are also currently treated differently by Kubernetes for
defaulting purposes.

We propose the creation of a new CRD flag: `preserveNullValues` which
will default to `true`. This flag will be used as a ratcheting mechanism
as we migrate to this new semantic throughout all Kubernetes types
(which would be how all types eventually work). When false, the
apiserver will use new semantics and remove all null values from CR
objects. When true, the existing and deprecated semantics will be kept.
That new flag is following the same logic as "preserveUnknownFields" and
the new operation will happen using the same mechanisms right after
pruning.

Note that we still consider `{}` (and 0, 0.0, "", false, []) to be
different from `null`, so `{foo: {}}` is different from `{}` or from
`{foo: null}` (but the latter two are considered equal).

As described by @sttts on many occasions, the immutability is blocked
because we don't know how to distinguish between null and
unspecified. We believe that this mechanism will be able to unblock it.

### Marker format

The marker will have the following format:
```
// +default=<any>
```
which describes the value that the server will use if the field has a
zero-value.

`<any>` has to be in one-line YAML format (including its json form),
representing the actual value to use by default. That is different from
the current encoding in kubebuilder, but there seems to be consensus
that they would be able to change that to YAML too for that new marker.

Namely, this will allow to preserve the concise semantics that you want
for most values:

```golang
type Foo struct {
  // +default=32
  Integer int

  // +default=bar
  String string

  // +default=[popcorn, chips]
  StringList []string
}
```
while still allowing more complex expressions:
```
// +default=[{port: 80, name: http}]
```

Forcing the "one-line" subset allows us to sidestep a lot of the YAML
weirdness in syntax, but we can still just use a YAML parser to parse
the expressions if we want.

### Built-in Defaulting

Because built-in types are serialized and deserialized as Go structures,
there are semantics that are imposed on us. The two main ones are:
1. We can't differentiate between null, unspecified and zero-value,
2. non-pointer structs are always present (and hence have to be
recursively defaulted), they also can't be omitEmpty'd.

The first point is solved by the removed nulls described in the previous
section (and invalid zero-values can't be present in objects), and the
second point can be solved by forcing non-pointer structs to have a `{}`
default. To address that, we're proposing that the `default` marker is
forbidden on non-pointer structs (pointer structs can behave as
expected) and the default field of the openapi is automatically set to
`{}`. This only applies to OpenAPI generated from built-in types or from
kubebuilder. CRDs can still have any defaulting they have for
structures, but we strongly suggest that OpenAPI generated from Go types
should follow that semantics.

Computed defaults (through defaulting functions) will not be supported
as a declarative marker and will continue being supported through the
existing defaulting functions.

### Examples

#### Non-pointer structs

Given the following type, defined either in Kubebuilder or as a built-in type:
```golang
type Root struct {
  Entry SubLevel
}

type SubLevel struct {
  // +defaut=default-name
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
      default: {}
      type: object
      properties:
        name:
          type: string
          default: default-name
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
  // +default={name: entry, number: 12}
  Entry SubLevel
}

Type SubLevel struct {
  Name string
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
  // +default={name: pointer-name}
  Entry *SubLevel
}

type SubLevel struct {
  // +defaut=default-name
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
      default: {name: pointer-name}
SubLevel:
  type: object
  properties:
    name:
      type: string
      default: default-name
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

#### Non-pointer fields

Non-pointer fields are also important, since they have a specific unmarshalling mechanism in Go.

For CRDs, we can assume that invalid zero-values are omitted, otherwise
the objects would technically be invalid.

For built-ins, we can differentiate two cases:
1. The zero-value for the field is invalid (e.g. if empty string is not a valid value)
2. The zero-value is valid, but is also the default (e.g. enabled false by default).

Both these cases will be handled properly, here's an example:
```golang
type Object struct {
     // This field has an invalid zero-value, empty string is not a
     // valid string.
     // +default=default-name
     Name string

     // This field is defaulted to 0 with no defaulter.
     // +default=0
     Defaulted int
}
```

This would generate the following OpenAPI:
```yaml
Object:
  type: object
  properties:
    invalid:
      type: string
      default: default-name
    defaulted:
      type: boolean
      default: false
```

Which, for both CRD types and built-in types, would generate the following behavior:
```python
>>> default('{}')
{"entry": {"name": "default-name", "number": 0}}
>>> default('{entry: {name: other-name}}')
{"entry": {"name": "other-name", "number": 0}}
```

Note that the generated code for built-in will look like this:
```golang
func GenerateDefault(obj *Object) {
     if obj.Name == "" {
          obj.Name = "default-name"
     }
     if obj.Number == 0 {
          obj.Number = 0
     }
}
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
existing defaulting logic, right before the existing defaulting
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
3. Implement the new CRD semantics that clears `null` values from
objects when the `x-kubernetes-remove-nulls` flag is present and true at
the top-level (or in the CRD).
4. Add additional "default" field to sigs.k8s.io/structured-merge-diff
internal schema, and parse that field from the OpenAPI, use it when the
map key is missing. The code for this already exists in a
[PR](https://github.com/kubernetes-sigs/structured-merge-diff/pull/83#issuecomment-498410353)
5. Update some of the existing built-in APIs to specify their default
when they are strictly declarative, probably focusing first on defaults
that are part of the associative keys (e.g. `protocol: TCP`).

Note that this is not changing how built-in defaults work, but is merely
adding an extra-layer on top of that which makes type authors life easier.

### Test Plan

As soon as this is implemented and working, we can start removing a lot
of the existing defaulting code and replace it with this values, and we
can re-use the existing defaulting tests, make sure that the behavior
doesn't change when moving from code-based defaults to declarative
default. Also panic in specific places of the defaulters to guarantee
that the defaulting have happened before.

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
