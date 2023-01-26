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

**Note:** Any PRs to move a KEP to `implementable`, or significant changes once
it is marked `implementable`, must be approved by each of the KEP approvers.
If none of those approvers are still appropriate, then changes to that list
should be approved by the remaining approvers and/or the owning SIG (or
SIG Architecture for cross-cutting KEPs).
-->

# KEP-3784: A database of Kubernetes-owned names

<!-- toc -->
- [Release Signoff Checklist](#release-signoff-checklist)
- [Summary](#summary)
- [Motivation](#motivation)
  - [Goals](#goals)
    - [Maybe later](#maybe-later)
  - [Non-Goals](#non-goals)
- [Proposal](#proposal)
  - [Primary index](#primary-index)
  - [Secondary indices](#secondary-indices)
  - [Data](#data)
    - [Why YAML](#why-yaml)
    - [Examples](#examples)
  - [Risks and Mitigations](#risks-and-mitigations)
- [Design Details](#design-details)
  - [Test Plan](#test-plan)
      - [Prerequisite testing updates](#prerequisite-testing-updates)
      - [Unit tests](#unit-tests)
      - [Integration tests](#integration-tests)
      - [e2e tests](#e2e-tests)
  - [Graduation Criteria](#graduation-criteria)
  - [Upgrade / Downgrade Strategy](#upgrade--downgrade-strategy)
  - [Version Skew Strategy](#version-skew-strategy)
- [Production Readiness Review Questionnaire](#production-readiness-review-questionnaire)
- [Implementation History](#implementation-history)
- [Drawbacks](#drawbacks)
- [Alternatives](#alternatives)
- [Infrastructure Needed (Optional)](#infrastructure-needed-optional)
<!-- /toc -->

## Release Signoff Checklist

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

[kubernetes.io]: https://kubernetes.io/
[kubernetes/enhancements]: https://git.k8s.io/enhancements
[kubernetes/kubernetes]: https://git.k8s.io/kubernetes
[kubernetes/website]: https://git.k8s.io/website

## Summary

Kubernetes has many extensible fields which can carry either "standard"
(contextual) or user-defined values.  Examples include (but are not limited to)
label keys, annotation keys, node taints, pod resource names, conditions, etc.

In most cases we define a "reserved" block, which is set aside for the project
to use.  Labels and annotations which are prefixed with a `k8s.io` or
`kubernetes.io` domain are reserved, and should not be used by end-users,
whereas unprefixed names are reserved for end-users and should not be used by
Kubernetes or any other project.

We have made great use of these reserved names in myriad ways, all over the
project.  What we have not had is a common way to know WHICH names we are
using, what they mean, where they apply, or who is responsible for them.  In
fact, the only way to find them is to read human-oriented documentation or to
grep the codebase(s).

This KEP proposes to store these names in a way that is searchable, machine
parseable, but also relatively human-friendly.

## Motivation

We have too many KEPs and PRs that add annotation- or label-keys without any
sort of collision detection or governance or documentation.

### Goals

1. Publish a list of known values.
1. Have a place where end-users can go to find information.
1. Have a place we can send developers to get some modicum of review of these
names.

#### Maybe later

1. Add this to the KEP template
1. Extend this DB to highly-coupled projects (e.g. containerd, crio)
1. Extend this DB to loosely-coupled projects (e.g. GKE, EKS, AKS)
1. Use in-code metadata to auto-generate these DB records

### Non-Goals

1. Completeness.  We know that getting 100% of the worms back into the can may
   never happen.  That doesn't mean we should not try.
1. Enforcement.  We do not intend to inject this into code-review processes.
   We do intend to make people **aware** of it, so they can voluntarily contribute.

## Proposal

The "database" should live in a new git repo (proposed name:
"kubernetes/namesdb").  In this repo we will establish a files-and-directories
structure which allows for easy searching and "a priori" direct indexing by
name.  We will populate it with some of the known names from the codebase, and
then crowd-source more.  This represents the "primary index".  We will use
OWNERS files to delegate ownership.

Given that primary index, we can programmatically produce secondary indices
which allow easy searching by other criteria (e.g. by SIG or by API resource).

We can also use this data to generate docs, if we want to make the
"description" parts of this proposal fairly robust.

This repo should probably live under k/k/staging to make PRs easier to add
names and code which uses them at the same time.  Staging "repos" are
automatically synced to "real" repos for easier consumption.

### Primary index

The primary index would look like:

```
github.com/kubernetes/namesdb/
    OWNERS
    README.md
    by-name/
        io/
            k8s/
                OWNERS
                names.yaml
                network/
                    OWNERS
                    names.yaml
                storage/
                    OWNERS
                    names.yaml
                <more...>
            kubernetes/
                OWNERS
                names.yaml
```

### Secondary indices

Secondary indices could be like:

```
github.com/kubernetes/namesdb/
    by-sig/
        network/
            names.yaml
        storage/
            names.yaml
    by-context/
        annotation/
            names.yaml
        app-protocol/
            names.yaml
        condition/
            names.yaml
        label/
            names.yaml
        taint/
            names.yaml
    by-resource/
        names.yaml
        pod/
            names.yaml
        deployment/
            names.yaml
        endpoints/
            names.yaml
        service/
            names.yaml
```

These secondary indices can be created by a simple program which we validate
has been run before merging changes to the primary index, or even by a bot
running after merge.

### Data

At the "leaf" directory of each name is a YAML file which follows a defined
schema.  The schema will certainly evolve over time, but we can start with the
following:

```go
// RecordList is the type represented in each names.yaml file.
type RecordList []Record

// Record describes a single reserved name.
type Record struct {
    // Key is the full name being claimed, including the prefix.
    // Required
    Key string

    // SIG specifies which SIG owns this name.  Multiple SIGs might use a name,
    // but only one can own it.
    // Required
    SIG SIG

    // Description provides human-friendy information about what this name
    // means and how/when to use it.
    // Required
    Description string

    // DocsLink provides an HTTP link to more information about this name.
    // Optional
    DocsLink string

    // Context specifies where this name might be used.  Names which can be
    // used in more than one place should define multiple name records.
    // Subsequent fields' requiredness depend on the context.
    Context Context

    // ValueType specifies what "inner" type(s) of value are expected for
    // contexts which require a value.  For example, annotation values are
    // always strings, but the content of those strings might be expected to be
    // integers, booleans, or even JSON-encoded structs.  Some names accept
    // multiple types.
    // Required if context demands
    ValueType []ValueType

    // Resources specifies which API resource(s) this name might be used with,
    // in "apigroup/resource" syntax.
    // Required if context demands
    Resources []string
}

// SIG is a Kubernetes special interest group name.
type SIG string

const (
    SIGNetwork = SIG("Network")
    SIGNode = SIG("Node")
    SIGApps = SIG("Apps")
    SIGStorage = SIG("Storage")
    SIGArchitecture = SIG("Architecture")
    // more...
)

// Context describes where a name might be used.
type Context string

const (
    ContextLabel = Context("Label")             // key+value
    ContextAnnotation = Context("Annotation")   // key+value
    ContextTaint = Context("Taint")             // key
    ContextCondition = Context("Condition")     // key
    ContextAppProtocol = Context("AppProtocol") // key
    ContextAPIGroup = Context("APIGroup")       // key
    // more...
)

// ValueType describes the datatype of a value for a key name.
type ValueType string

const (
    ValueTypeInt = ValueType("Int")
    ValueTypeString = ValueType("String")
    ValueTypeBool = ValueType("Bool")
    ValueTypeJSONObject = ValueType("JSONObject")
    ValueTypeJSONList = ValueType("JSONList")
    // more...
)
```

We can publish this schema as Go code or something else if there is a better or
more standard notation.  We can validate all names against this schema.

<<[UNRESOLVED enumerate resources ]>>
Should we enumerate resources?  Good for validation, but a bit tedious.
<<[/UNRESOLVED]>>

#### Why YAML

YAML is the lingua franca of Kubernetes.  We propose to use a subset of YAML
which is closer to JSON and easier to avoid whitespace errors, while still
allowing comments.

#### Examples

/by-name/io/kubernetes/names.yaml:
```yaml
[{
  key: "kubernetes.io/hostname",
  sig: "Node",
  context: "Label",
  valueType: "String",
  resources: [ "core/nodes" ],
  description: "The name of the node.",
}, {
  key: "kubernetes.io/arch",
  sig: "Node",
  context: "Label",
  valueType: "String",
  resources: [ "core/nodes" ],
  description: "The architecture of the node.",
}, {
  key: "kubernetes.io/os",
  sig: "Node",
  context: "Label",
  valueType: "String",
  resources: [ "core/nodes" ],
  description: "The OS of the node.",
}]
```


/by-name/io/kubernetes/topology/names.yaml:
```yaml
[{
  key: "topology.kubernetes.io/zone",
  sig: "Architecture",
  context: "Label",
  valueType: "String",
  resources: [ "core/nodes", core/persistentvolumes ],
  description: "The availability zone of the resource.",
  docsLink: "https://kubernetes.io/docs/reference/labels-annotations-taints/#topologykubernetesiozone"
}, {
  key: "topology.kubernetes.io/region",
  sig: "Architecture",
  context: "Label",
  valueType: "String",
  resources: [ "core/nodes", core/persistentvolumes ],
  description: "The availability region of the resource.",
  docsLink: "https://kubernetes.io/docs/reference/labels-annotations-taints/#topologykubernetesioregion"
}]
```

<<[UNRESOLVED key collisions ]>>
Should we have a YAML per key - easier lookups and collision avoidance, but
lots of files.  Would maybe need Context to be a list.
<<[/UNRESOLVED]>>

### Risks and Mitigations

Risk: Once we have a list, people will consider it authoritative and, even if
we say otherwise, complete.  This does not absolve people of the need to "grep
around" and make sure a name they want to use isn't quietly in use elsewehere.

## Design Details

See Proposal above.

### Test Plan

##### Prerequisite testing updates

N/A

##### Unit tests

Tooling around this would be unit-tested.

##### Integration tests

N/A

##### e2e tests

N/A

### Graduation Criteria

CI must flag changes that do not conform to schema.

### Upgrade / Downgrade Strategy

N/A

### Version Skew Strategy

N/A

## Production Readiness Review Questionnaire

N/A

## Implementation History

Jan 26, 2023: First draft.

## Drawbacks

It's YET ANOTHER thing to update.

## Alternatives

We could NOT do this and maintain status quo - no obvious way to enumerate
these names other than [exclusively-human-oriented docs](https://kubernetes.io/docs/reference/labels-annotations-taints/).

## Infrastructure Needed (Optional)

A github repo and CI integration.
