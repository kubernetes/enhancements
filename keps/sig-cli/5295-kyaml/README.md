<!--
- [X] **Pick a hosting SIG.**
  Make sure that the problem space is something the SIG is interested in taking
  up. KEPs should not be checked in without a sponsoring SIG.
- [X] **Create an issue in kubernetes/enhancements**
  When filing an enhancement tracking issue, please make sure to complete all
  fields in that template. One of the fields asks for a link to the KEP. You
  can leave that blank until this KEP is filed, and then go back to the
  enhancement and add the link.
- [X] **Make a copy of this template directory.**
  Copy this template into the owning SIG's directory and name it
  `NNNN-short-descriptive-title`, where `NNNN` is the issue number (with no
  leading-zero padding) assigned to your enhancement above.
- [X] **Fill out as much of the kep.yaml file as you can.**
  At minimum, you should fill in the "Title", "Authors", "Owning-sig",
  "Status", and date-related fields.
- [X] **Fill out this file as best you can.**
  At minimum, you should fill in the "Summary" and "Motivation" sections.
  These should be easy if you've preflighted the idea of the KEP with the
  appropriate SIG(s).
- [X] **Create a PR for this KEP.**
  Assign it to people in the SIG who are sponsoring this process.
- [ ] **Merge early and iterate.**
  Avoid getting hung up on specific details and instead aim to get the goals of
  the KEP clarified and merged quickly. The best way to do this is to just
  start with the high-level sections and fill out details incrementally in
  subsequent PRs.
-->

# KEP-5295: Introducing KYAML, a safer, less ambiguous YAML subset / encoding

<!-- toc -->
- [Release Signoff Checklist](#release-signoff-checklist)
- [Summary](#summary)
- [Motivation](#motivation)
  - [Goals](#goals)
  - [Non-Goals](#non-goals)
- [Proposal](#proposal)
  - [Notes/Constraints/Caveats](#notesconstraintscaveats)
  - [Risks and Mitigations](#risks-and-mitigations)
- [Design Details](#design-details)
  - [Feature gate](#feature-gate)
  - [Future compatibility](#future-compatibility)
  - [Packaging](#packaging)
  - [Components](#components)
  - [Tooling](#tooling)
    - [How?](#how)
  - [Comments](#comments)
  - [Disambiguation from JSON](#disambiguation-from-json)
  - [Per-type rendering](#per-type-rendering)
    - [Scalars](#scalars)
    - [Mappings (structs and maps)](#mappings-structs-and-maps)
      - [Structs](#structs)
      - [Maps](#maps)
      - [Lists](#lists)
      - [Pointers](#pointers)
      - [Self marshalers](#self-marshalers)
      - [Interfaces](#interfaces)
      - [Anything else](#anything-else)
    - [Multi-line strings](#multi-line-strings)
  - [Commas](#commas)
  - [Brackets](#brackets)
  - [Indents](#indents)
  - [Short lists, maps, and structs](#short-lists-maps-and-structs)
  - [Alignment](#alignment)
  - [Consistency of quoted keys](#consistency-of-quoted-keys)
  - [Examples](#examples)
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

- [X] (R) Enhancement issue in release milestone, which links to KEP dir in [kubernetes/enhancements] (not the initial KEP PR)
- [X] (R) KEP approvers have approved the KEP status as `implementable`
- [X] (R) Design details are appropriately documented
- [X] (R) Test plan is in place, giving consideration to SIG Architecture and SIG Testing input (including test refactors)
  - [ ] e2e Tests for all Beta API Operations (endpoints)
  - [ ] (R) Ensure GA e2e tests meet requirements for [Conformance Tests](https://github.com/kubernetes/community/blob/master/contributors/devel/sig-architecture/conformance-tests.md)
  - [ ] (R) Minimum Two Week Window for GA e2e tests to prove flake free
- [X] (R) Graduation criteria is in place
  - [ ] (R) [all GA Endpoints](https://github.com/kubernetes/community/pull/1806) must be hit by [Conformance Tests](https://github.com/kubernetes/community/blob/master/contributors/devel/sig-architecture/conformance-tests.md)
- [X] (R) Production readiness review completed
- [X] (R) Production readiness review approved
- [X] "Implementation History" section is up-to-date for milestone
- [X] User-facing documentation has been created in [kubernetes/website], for publication to [kubernetes.io]
- [X] Supporting documentation—e.g., additional design documents, links to mailing list discussions/SIG meetings, relevant PRs/issues, release notes

[kubernetes.io]: https://kubernetes.io/
[kubernetes/enhancements]: https://git.k8s.io/enhancements
[kubernetes/kubernetes]: https://git.k8s.io/kubernetes
[kubernetes/website]: https://git.k8s.io/website

## Summary

This KEP proposes to add a new output format for kubectl,
called "KYAML"
(as in `kubectl get -o kyaml ...`).
This format is a strict subset (aka "dialect") of standard YAML,
and so should be parseable by the existing ecosystem.
This dialect seeks to emphasize syntactical choices which avoid
many of the most common traps in YAML.
For example, unlike standard YAML output,
this dialect is not whitespace-sensitive,
which makes it vastly easier to patch correctly in things like Helm charts.

This KEP further proposes to make KYAML the standard format
for all project-owned documentation and examples.

## Motivation

The vast majority of examples of how to use Kubernetes include YAML.
These all use the most "conventional" YAML syntax,
which is what `kubectl get -o yaml` also uses.
Simple YAML is easy to read, but complex YAML can become very tricky.
Some of the things that motivated this KEP are:

Whitespace being significant means that
readers and writers must track the nesting depth.
While this does not sound too onerous,
We have ample evidence that it is, in fact, very hard.
Wrongly indented YAML is syntactically valid but does not marshal properly.
Perhaps worst of all,
template-expanding systems,
such as the widely used `helm`,
force users to get whitespace "just right"
while **outside of** the main YAML context.

String-quoting in YAML is generally optional,
but some things that appear to be strings are in fact coerced to other types.
For example, without quotes
`NO`, `no`, `N`, `YES`, `yes`, `Y`, `On`, and `Off` are all treated as bool
(aka  ["The Norway Bug"](https://hitchdev.com/strictyaml/why/implicit-typing-removed/)),
`_42`, `_4_2_`, and `__________4__________2__________` are all treated as numbers,
and `11:00` is also treated as a number (base 60).

As an alternative to YAML most of Kubernetes also supports JSON,
But JSON has its own drawbacks.
It does not support comments.
It is very particular about trailing commas.
It requires quoted keys.

In addition to `kubectl` output,
most Kubernetes config files are either YAML or JSON.
JSON lacks comments,
which are very valuable for configuration,
so most configuration and tooling is built around YAML.

The most frustrating part of YAML is that it doesn’t have to be this bad.
YAML is a very broad specification.
Inside that specification is a reasonable grammar screaming to be let out.

If we change our tooling and examples to use a less error-prone dialect of YAML,
we can encourage the ecosystem to avoid some of the YAML pitfalls
and reduce common mistakes and frustration.

### Goals

1. Specify a YAML dialect which is 100% compatible with existing parsers and tooling.
2. Provide libraries and tooling for encoding and converting to this formatting, with
   stable (idempotent) conversion.
3. Provide a KYAML output format for kubectl.
4. Update examples and documentation throughout the project.
5. Address common YAML pitfalls through an opinionated formatting approach.
6. Document the design and reasons for KYAML and the 100% compatibility with tooling expecting arbitrary YAML.

### Non-Goals

1. Introduce alternative configuration languages that are not compatible with existing tooling.
2. Server-side support for KYAML distinct from what we support in YAML already.
3. Require projects to migrate to KYAML by default.
4. Change the YAML library used by Kubernetes.

## Proposal

KYAML is primarily a specification of output.
Since KYAML is a subset of YAML,
it should always be accepted as input,
but this KEP does not demand that such input be strict KYAML.
Users can choose to violate this specification,
As long as the result is still valid YAML.

KYAML addresses the problems cited above by:
1. Always double-quoting _value_ strings (no ambiguity around things like "NO").
2. Unquoted keys, unless they are not obviously "safe" (e.g. "no" would always be quoted).
3. Always using `{}` for structs and maps (no more obscure errors about mapping values).
4. Always using `[]` for lists (no more trying to figure out if a dash changes the meaning).

Taken together,
these mean that KYAML is not whitespace-sensitive,
which makes templating and other text-patching much easier.
YAML calls this "flow style",
as opposed to the conventional "block style".

This might sound a lot like JSON,
because it is!
Remember that JSON is valid YAML.
KYAML is friendlier than JSON for humans
to read,
to write,
and to commit to source control.
Unlike JSON, KYAML:
1. Allows comments (when authored by users).
2. Allows trailing commas on the last item in a list or struct/map.
3. Does not require quoted keys.

KYAML also includes a "header" (still valid YAML)
which helps to disambiguate a KYAML document from an ill-formed JSON document
(see: [kubernetes/kubernetes#78946](https://github.com/kubernetes/kubernetes/issues/78946)).

KYAML also tries to economize on vertical space by ["cuddling"](https://blog.gskinner.com/archives/2008/11/curly_braces_to.html) some kinds of brackets together.

Because KYAML is YAML, a KYAML multi-document is a YAML multi-document.

To make this change, we will:
1. Create new code (see details below) which serializes any object into KYAML.
2. Introduce a kubectl output format, `-o=kyaml`, which uses this new package.
3. Create tooling which reformats YAML or JSON as KYAML (see details below).
4. Work with SIG Docs to update examples to this format, both example command-lines and examples of API objects.
5. Look for other examples across kubernetes-affiliated git repositories, and offer PRs to convert to KYAML.
6. Make a lot of noise about it, and try to get more people to use it

The rationale for changing everything everywhere is simple.
Many users start with examples,
which they copy and modify.
By changing the examples we seed the change in the ecosystem.
One success metric might be
that KYAML becomes the preferred format in the most common helm charts.

### Notes/Constraints/Caveats

**Important note:**
This is EXCLUSIVELY about client-side formatting.
This KEP *does not* propose any changes to apiserver
or other components besides kubectl.

Syntax has always been a magnet for arguments in our industry.
We expect this to be no exception.
Reasonable people will disagree with our choices,
which is fine, but we should hold a strong opinion on this.

YAML is a very large and complex specification.
We expect that we will find places where KYAML may be less readable than conventional YAML.
We should revisit this specification when that happens.
Given that KYAML is YAML it should be possible to revise
without breaking anyone.

### Risks and Mitigations

Risk: Our encoder produces some output which is not valid YAML.
Mitigation: That’s a bug. It’s hard to imagine a case that can’t be fixed.

Risk: The result could be net worse than plain YAML.
Mitigation: We will produce many examples and include them in the review
  process. We will solicit feedback from various groups on the result.

Risk: This gives a false sense of safety, but doesn’t SOLVE  the problems with
  YAML, just avoids them.
Mitigation: Docs. Our goal is not to modify YAML format, but rather focus on a
  valid YAML sub-set, which resolves known problems, increasing the success of
  our users.

Risk: Like all APIs, people will come to depend on the details, which may
  prevent us from changing them.
Mitigation: Be as precise as possible where we are confident, but leave room
  where we are not.

See also [Drawbacks](#drawbacks).

## Design Details

Creating an encoder for this format is not very technically challenging,
but there are many details to consider.

### Feature gate

Literal feature gates are not super useful for CLI features, so SIG CLI has a
similar mechanism to gate features via environment variables.  We will use a
variable named `"KUBECTL_KYAML"` to gate this feature.

The output flags for JSON and YAML are handled in
`k8s.io/cli-runtime/pkg/genericclioptions`.  Unfortunately, that package is
already transitively depended on by `k8s.io/kubectl/pkg/cmd/util` where the CLI
gates are defined. Rather than trying to break the import cycle, we will just
check the environment variable directly, without the utils package.


### Future compatibility

KYAML is, and will remain, a strict subset of YAML.  Compliant YAML parsers
should always be able to parse KYAML data and we will endeavour to make sure
they can all retain fidelity with comments.

It may be that, over time, we find some aspects of KYAML's exact rendering that
can be improved, and we will consider those opportunities. The implication of
this is that clients which rely on the byte-exact results of KYAML's rendering
might be broken by future changes. We will try to avoid this, but we will not
guarantee it.

### Packaging

We will create a new `kyaml` package under `sigs.k8s.io/yaml` for the rendering
logic and put the printer under `staging/src/k8s.io/cli-runtime/pkg/printers`,
using the renderer.

### Components

The primary component affected by this KEP is `kubectl`, but any other
component which uses `k8s.io/cli-runtime/pkg/printers` and
`k8s.io/cli-runtime/pkg/genericclioptions` will get KYAML support "for free".

### Tooling

k/k has a CI verifier (hack/verify-yamlfmt.sh) which looks for YAML files and
ensures they are formatted "correctly". The only meaning of "correct" is "block
style", and it corrupts KYAML input today (aside: it also does not handle
multi-documents).  If we intend to convert examples, we need to fix this.

We propose to add flags to it which specify to use "conventional" YAML or KYAML
output, and set the default to KYAML (or to change the parent script to always
specify KYAML).

This tooling will preserve comments as best it can. Unfortunately, go-yaml (our
YAML library) [does not always handle comments properly](https://github.com/yaml/go-yaml/issues/8).
It is possible that some comments will be formatted wrongly, or lost entirely.
The affected comments are not representable in YAML block-style, so this is not
exactly a regression.

A guiding principle for the tooling is that whatever output the tool produces,
when fed back into the same version of the tool, should not be modified again.
As mentioned above, we MIGHT make formatting changes over time, so this
property may not hold across versions of the tooling.

Complex YAML input, with things like anchors, aliases, explicit tags (!!...),
and global tags (%TAG !...) may be simplified (e.g. aliases will be reified)
when rendered to KYAML, as long as the resulting unmarshaled object would be
identical.

#### How?

To ensure maximum fidelity to JSON, KYAML rendering of Go objects will first
marshal them to JSON, then render the JSON as KYAML.  This process means we get
all of the JSON tag logic "for free", and that we can use the same code to
render Go objects and to re-render YAML or JSON input. That extra conversion is
ugly, but this is not generally performance-sensitive code and is not run on
the server.

One side-effect of this is that we cannot reformat some valid YAML documents
into KYAML, because there are things that YAML allows that JSON does not, such
as compound non-string keys (e.g. structs or lists), and things that Go's JSON
encoder quietly converts to strings, such as numbers.

In other words, KYAML is YAML, but not all YAML is KYAML.

As of kubernetes 1.33, we have forked go-yaml.v2 AND go-yaml.v3 into
sigs.k8s.io/yaml.  While KYAML is an output format, our tooling goals require
that we get access to a YAML AST.  Fortunately, go-yaml.v3 offers a way to
decode YAML into such an AST. If we ever change YAML libraries (e.g.
[https://github.com/goccy/go-yaml](https://github.com/goccy/go-yaml)), we will
need to rewrite this code.

### Comments

When reformatting existing YAML with comments, we will attempt to place comments
in "obvious" places.  For example, a line-comment on a scalar field is
obviously placed at the end of the line.  These cases will not be called out in
the subsequent sections of this KEP.  In some cases, the comment placement
may not be obvious, and so will be called out explicitly.

### Disambiguation from JSON

At the top of each KYAML document (or multi-document) we will put a document
separator (`---`) "header". This disambiguates it from JSON, since both KYAML
and JSON begin with an opening curly bracket.  Kubernetes versions 1.33 and
higher can handle KYAML without this, but it is needed for versions before
that.

### Per-type rendering

#### Scalars

Integers and floats are rendered as their natural numeric representation.

Simple strings are rendered as double-quoted and escaped. Multi-line strings
require special consideration, see below.

Booleans are rendered as `true` or `false`.

#### Mappings (structs and maps)

##### Structs

Structs are rendered with curly brackets, and each field on a line of its own.

When reformatting existing YAML, a struct's line-comment will be rendered after
the closing bracket (since this is the only place it will be found by go-yaml
upon re-reading).

Field names are not quoted unless they are not obviously strings or are one of
the known type-ambiguous words (e.g. `no`).  "Obvious strings" include
alphanumeric characters and some punctuation, and includes label-keys with
prefixes. Field values are rendered as per their type.  Embedded structs will
be rendered as inline fields.

KYAML does not guarantee that fields will be rendered in the order they are
defined in the struct.  Implementations may preserve field order, but `kubectl`
itself does not guarantee this.  For example, `kubectl` converts everything to an
unstructured type internally, which loses the field ordering.

Fields which have a `json` struct tag will be rendered according to that tag,
by nature of the fact that we marshal Go objects to JSON first.

The `yaml` struct tag will be ignored (as with our current yaml rendering).

##### Maps

Maps are rendered with curly brackets, and each key on a line of its own.

When reformatting existing YAML, a map's line-comment will be rendered after
the closing bracket (since this is the only place it will be found by go-yaml
upon re-reading).

Because we marshal Go objects to JSON, map keys must be strings. This is more
restrictive than plain YAML, meaning that reformatting YAML with non-string
keys into KYAML will fail.

Keys are sorted and are not quoted unless they are not obviously strings or are
one of the known type-ambiguous words, the same as for struct fields.

Values are rendered as per their type.

##### Lists

Slices and arrays are rendered with square brackets.

When reformatting existing YAML, a list's line-comment will be rendered after
the closing bracket (since this is the only place it will be found by go-yaml
upon re-reading).

List elements are rendered as per their type.

##### Pointers

Pointers are rendered as `null` or dereferenced and rendered as per the
pointee’s type.

##### Self marshalers

Types which implement the Go standard `json.Marshaler` interface
(`MarshalJSON()`) are rendered by marshaling to JSON.

Types which implement the `yaml.Marshaler` interface (`MarshalYAML()`) are
rendered as per their type.  `MarshalYAML()` is not a Go standard and is not
used by KYAML (as with our current YAML rendering).

##### Interfaces

Interfaces are rendered as `null` or their true value type.

##### Anything else

Any other type (as per `reflect.Value.Kind()`) will produce an error.

#### Multi-line strings

For better or worse, multi-line strings are a thing that people do in
Kubernetes, and conventional YAML makes it pretty easy. KYAML has to choose how
we want to handle it.  Consider these examples of conventional YAML.

Example 1:
```
example_1: |
    This
    is a
    multi-line string
```

Example 2:
```
example_2: |
    this:
      is:
      - embedded
      - yaml
      where:
        whitespace:
          matters: true
```

This style is not available in YAML flow-style.

KYAML will use YAML flow-folding and escaped line-breaks to get "close" to the
original without sacrificing the main goals of this KEP.  A flow-fold is a
trailing backslash at the very end of a line, which tells YAML to remove the
newline and any leading whitespace on the next line.

To get newlines we will use `\n` in the string.

To get leading whitespace, we will use an escape character, which tells YAML
that any further whitespace is part of the contents.

We will add an extra space on lines which do not have leading whitespace, to
preserve vertical alignment.

Note: the extra space is only in the rendering, not in the actual content,
since YAML drops all leading whitespace until the first non-whitespace
character.

Example:

```
example_1: "\
     This\n\
     is a\n\
     multi-line string\
    "
example_2: "\
     this:\n\
    \  is:\n\
    \  - embedded\n\
    \  - yaml\n\
    \  where:\n\
    \    whitespace:\n\
    \      matters: true\n\
    "
```

### Commas

Lists, maps, and structs always have a trailing comma after their final
element, unless the closing bracket is cuddled.  As mentioned elsewhere, the
comma is not required, but KYAML always emits it.

Example:

```
labels: {
  app: "foobar",
  foo: "bar",
  something: "12345",
},
```

### Brackets

Paired brackets are always cuddled, unless the next item has a head-comment or
the previous item has a foot-comment.

This is best illustrated with an example:

Cuddled:
```
cuddled: [{
  "f": 1,
}, {
  "f": 2,
}, {
  "f": 3,
}],
```

Partially cuddled, with head-comments.
```
with_head_comments: [
# head1
{
  "f": 1,
},
# head2
{
  "f": 2,
}, {
  "f": 3,
}],
```

Partially cuddled, with line-comments.
```
with_line_comments: [{
  "f": 1,
}, # line1
{
  "f": 2,
}, # line2
{
  "f": 3,
}],
```

Partially cuddled, with foot-comments.
```
with_foot_comments: [{
  "f": 1,
},
# foot1
{
  "f": 2,
},
# foot1
{
  "f": 3,
}],
```

Note that without whitespace a "foot comment" is ambiguous to the the item's
"head comment".  That is an internal issue for go-yaml and should not affect
the rendering, unless we choose to emit blank lines.

### Indents

Indents follow YAML’s two-space convention.

### Short lists, maps, and structs

Empty lists and maps are rendered as `[]` and `{}` respectively.

Other than the special-case above, no affordance is made to render short lists,
maps, or structs on a single line. This is possible but was deemed unnecessary.

### Alignment

No affordance is made to align the values of struct fields or map keys (as gofmt does).

### Consistency of quoted keys

No affordance is made to ensure that keys of a map or struct are all quoted or
not-quoted similarly. Some keys can be quoted while others are not.

### Examples

```
$ kubectl get -o kyaml svc hostnames
---
{
  apiVersion: "v1",
  kind: "Service",
  metadata: {
    creationTimestamp: "2025-05-09T21:14:40Z",
    labels: {
      app: "hostnames",
    },
    name: "hostnames",
    namespace: "default",
    resourceVersion: "37697",
    uid: "7aad616c-1686-4231-b32e-5ec68a738bba",
  },
  spec: {
    clusterIP: "10.0.162.160",
    clusterIPs: [
      "10.0.162.160",
    ],
    internalTrafficPolicy: "Cluster",
    ipFamilies: [
      "IPv4",
    ],
    ipFamilyPolicy: "SingleStack",
    ports: [{
      port: 80,
      protocol: "TCP",
      targetPort: 9376,
    }],
    selector: {
      app: "hostnames",
    },
    sessionAffinity: "None",
    type: "ClusterIP",
  },
  status: {
    loadBalancer: {},
  },
}
```

```
$ kubectl get -o kyaml endpointslices
---
{
  apiVersion: "v1",
  items: [{
    addressType: "IPv4",
    apiVersion: "discovery.k8s.io/v1",
    endpoints: [{
      addresses: [
        "10.64.3.4",
      ],
      conditions: {
        ready: true,
        serving: true,
        terminating: false,
      },
      nodeName: "kubernetes-minion-group-bd63",
      targetRef: {
        kind: "Pod",
        name: "hostnames-77b655d8d-qs6sb",
        namespace: "default",
        uid: "afd43648-4a75-4616-ba72-2915b599fef9",
      },
      zone: "us-central1-b",
    }, {
      addresses: [
        "10.64.1.3",
      ],
      conditions: {
        ready: true,
        serving: true,
        terminating: false,
      },
      nodeName: "kubernetes-minion-group-03hb",
      targetRef: {
        kind: "Pod",
        name: "hostnames-77b655d8d-hsxbf",
        namespace: "default",
        uid: "ce0af71d-8a19-4d36-9435-3ce022f68113",
      },
      zone: "us-central1-b",
    }],
    kind: "EndpointSlice",
    metadata: {
      annotations: {
        "endpoints.kubernetes.io/last-change-trigger-time": "2025-05-09T21:14:42Z",
      },
      creationTimestamp: "2025-05-09T21:14:40Z",
      generateName: "hostnames-",
      generation: 3,
      labels: {
        app: "hostnames",
        "endpointslice.kubernetes.io/managed-by": "endpointslice-controller.k8s.io",
        "kubernetes.io/service-name": "hostnames",
      },
      name: "hostnames-bwpxq",
      namespace: "default",
      ownerReferences: [{
        apiVersion: "v1",
        blockOwnerDeletion: true,
        controller: true,
        kind: "Service",
        name: "hostnames",
        uid: "7aad616c-1686-4231-b32e-5ec68a738bba",
      }],
      resourceVersion: "37714",
      uid: "3c80d3ed-5dc7-4f37-8a60-da23847adec9",
    },
    ports: [{
      name: "",
      port: 9376,
      protocol: "TCP",
    }],
  }, {
    addressType: "IPv4",
    apiVersion: "discovery.k8s.io/v1",
    endpoints: [{
      addresses: [
        "34.60.45.55",
      ],
      conditions: {
        ready: true,
      },
    }],
    kind: "EndpointSlice",
    metadata: {
      creationTimestamp: "2025-05-09T17:17:36Z",
      generation: 1,
      labels: {
        "kubernetes.io/service-name": "kubernetes",
      },
      name: "kubernetes",
      namespace: "default",
      resourceVersion: "198",
      uid: "c818bb83-6b8a-4e30-931f-edbb34ed3dda",
    },
    ports: [{
      name: "https",
      port: 443,
      protocol: "TCP",
    }],
  }],
  kind: "List",
  metadata: {
    resourceVersion: "",
  },
}
```

```
$ kubectl get -o kyaml pods
---
{
  apiVersion: "v1",
  items: [{
    apiVersion: "v1",
    kind: "Pod",
    metadata: {
      annotations: {
        "kubernetes.io/limit-ranger": "LimitRanger plugin set: cpu request for container serve-hostname-xqnl5",
      },
      creationTimestamp: "2025-05-09T21:14:39Z",
      generateName: "hostnames-77b655d8d-",
      generation: 1,
      labels: {
        app: "hostnames",
        pod-template-hash: "77b655d8d",
      },
      name: "hostnames-77b655d8d-hsxbf",
      namespace: "default",
      ownerReferences: [{
        apiVersion: "apps/v1",
        blockOwnerDeletion: true,
        controller: true,
        kind: "ReplicaSet",
        name: "hostnames-77b655d8d",
        uid: "bd95a01f-ad62-4983-a3a4-dd1f9b1ca226",
      }],
      resourceVersion: "37711",
      uid: "ce0af71d-8a19-4d36-9435-3ce022f68113",
    },
    spec: {
      containers: [{
        image: "k8s.gcr.io/serve_hostname:v1.4",
        imagePullPolicy: "IfNotPresent",
        name: "serve-hostname-xqnl5",
        resources: {
          requests: {
            cpu: "100m",
          },
        },
        terminationMessagePath: "/dev/termination-log",
        terminationMessagePolicy: "File",
        volumeMounts: [{
          mountPath: "/var/run/secrets/kubernetes.io/serviceaccount",
          name: "kube-api-access-nc4zj",
          readOnly: true,
        }],
      }],
      dnsPolicy: "ClusterFirst",
      enableServiceLinks: true,
      nodeName: "kubernetes-minion-group-03hb",
      preemptionPolicy: "PreemptLowerPriority",
      priority: 0,
      restartPolicy: "Always",
      schedulerName: "default-scheduler",
      securityContext: {},
      serviceAccount: "default",
      serviceAccountName: "default",
      terminationGracePeriodSeconds: 30,
      tolerations: [{
        effect: "NoExecute",
        key: "node.kubernetes.io/not-ready",
        operator: "Exists",
        tolerationSeconds: 300,
      }, {
        effect: "NoExecute",
        key: "node.kubernetes.io/unreachable",
        operator: "Exists",
        tolerationSeconds: 300,
      }],
      volumes: [{
        name: "kube-api-access-nc4zj",
        projected: {
          defaultMode: 420,
          sources: [{
            serviceAccountToken: {
              expirationSeconds: 3607,
              path: "token",
            },
          }, {
            configMap: {
              items: [{
                key: "ca.crt",
                path: "ca.crt",
              }],
              name: "kube-root-ca.crt",
            },
          }, {
            downwardAPI: {
              items: [{
                fieldRef: {
                  apiVersion: "v1",
                  fieldPath: "metadata.namespace",
                },
                path: "namespace",
              }],
            },
          }],
        },
      }],
    },
    status: {
      conditions: [{
        lastProbeTime: null,
        lastTransitionTime: "2025-05-09T21:14:42Z",
        status: "True",
        type: "PodReadyToStartContainers",
      }, {
        lastProbeTime: null,
        lastTransitionTime: "2025-05-09T21:14:39Z",
        status: "True",
        type: "Initialized",
      }, {
        lastProbeTime: null,
        lastTransitionTime: "2025-05-09T21:14:42Z",
        status: "True",
        type: "Ready",
      }, {
        lastProbeTime: null,
        lastTransitionTime: "2025-05-09T21:14:42Z",
        status: "True",
        type: "ContainersReady",
      }, {
        lastProbeTime: null,
        lastTransitionTime: "2025-05-09T21:14:39Z",
        status: "True",
        type: "PodScheduled",
      }],
      containerStatuses: [{
        allocatedResources: {
          cpu: "100m",
        },
        containerID: "containerd://3a4c16cb0d735fd329b5d9775ee22ca56d824ea1cbe86d722ca9b8c32c3bbd7c",
        image: "k8s.gcr.io/serve_hostname:v1.4",
        imageID: "sha256:b56f22170982bc46dee99f7157be69dce727d3b89ee5698d15fda7a14a124200",
        lastState: {},
        name: "serve-hostname-xqnl5",
        ready: true,
        resources: {
          requests: {
            cpu: "100m",
          },
        },
        restartCount: 0,
        started: true,
        state: {
          running: {
            startedAt: "2025-05-09T21:14:41Z",
          },
        },
        volumeMounts: [{
          mountPath: "/var/run/secrets/kubernetes.io/serviceaccount",
          name: "kube-api-access-nc4zj",
          readOnly: true,
          recursiveReadOnly: "Disabled",
        }],
      }],
      hostIP: "10.40.0.7",
      hostIPs: [{
        ip: "10.40.0.7",
      }],
      phase: "Running",
      podIP: "10.64.1.3",
      podIPs: [{
        ip: "10.64.1.3",
      }],
      qosClass: "Burstable",
      startTime: "2025-05-09T21:14:39Z",
    },
  }, {
    apiVersion: "v1",
    kind: "Pod",
    metadata: {
      annotations: {
        "kubernetes.io/limit-ranger": "LimitRanger plugin set: cpu request for container serve-hostname-xqnl5",
      },
      creationTimestamp: "2025-05-09T21:14:39Z",
      generateName: "hostnames-77b655d8d-",
      generation: 1,
      labels: {
        app: "hostnames",
        pod-template-hash: "77b655d8d",
      },
      name: "hostnames-77b655d8d-qs6sb",
      namespace: "default",
      ownerReferences: [{
        apiVersion: "apps/v1",
        blockOwnerDeletion: true,
        controller: true,
        kind: "ReplicaSet",
        name: "hostnames-77b655d8d",
        uid: "bd95a01f-ad62-4983-a3a4-dd1f9b1ca226",
      }],
      resourceVersion: "37706",
      uid: "afd43648-4a75-4616-ba72-2915b599fef9",
    },
    spec: {
      containers: [{
        image: "k8s.gcr.io/serve_hostname:v1.4",
        imagePullPolicy: "IfNotPresent",
        name: "serve-hostname-xqnl5",
        resources: {
          requests: {
            cpu: "100m",
          },
        },
        terminationMessagePath: "/dev/termination-log",
        terminationMessagePolicy: "File",
        volumeMounts: [{
          mountPath: "/var/run/secrets/kubernetes.io/serviceaccount",
          name: "kube-api-access-g57rz",
          readOnly: true,
        }],
      }],
      dnsPolicy: "ClusterFirst",
      enableServiceLinks: true,
      nodeName: "kubernetes-minion-group-bd63",
      preemptionPolicy: "PreemptLowerPriority",
      priority: 0,
      restartPolicy: "Always",
      schedulerName: "default-scheduler",
      securityContext: {},
      serviceAccount: "default",
      serviceAccountName: "default",
      terminationGracePeriodSeconds: 30,
      tolerations: [{
        effect: "NoExecute",
        key: "node.kubernetes.io/not-ready",
        operator: "Exists",
        tolerationSeconds: 300,
      }, {
        effect: "NoExecute",
        key: "node.kubernetes.io/unreachable",
        operator: "Exists",
        tolerationSeconds: 300,
      }],
      volumes: [{
        name: "kube-api-access-g57rz",
        projected: {
          defaultMode: 420,
          sources: [{
            serviceAccountToken: {
              expirationSeconds: 3607,
              path: "token",
            },
          }, {
            configMap: {
              items: [{
                key: "ca.crt",
                path: "ca.crt",
              }],
              name: "kube-root-ca.crt",
            },
          }, {
            downwardAPI: {
              items: [{
                fieldRef: {
                  apiVersion: "v1",
                  fieldPath: "metadata.namespace",
                },
                path: "namespace",
              }],
            },
          }],
        },
      }],
    },
    status: {
      conditions: [{
        lastProbeTime: null,
        lastTransitionTime: "2025-05-09T21:14:41Z",
        status: "True",
        type: "PodReadyToStartContainers",
      }, {
        lastProbeTime: null,
        lastTransitionTime: "2025-05-09T21:14:39Z",
        status: "True",
        type: "Initialized",
      }, {
        lastProbeTime: null,
        lastTransitionTime: "2025-05-09T21:14:41Z",
        status: "True",
        type: "Ready",
      }, {
        lastProbeTime: null,
        lastTransitionTime: "2025-05-09T21:14:41Z",
        status: "True",
        type: "ContainersReady",
      }, {
        lastProbeTime: null,
        lastTransitionTime: "2025-05-09T21:14:39Z",
        status: "True",
        type: "PodScheduled",
      }],
      containerStatuses: [{
        allocatedResources: {
          cpu: "100m",
        },
        containerID: "containerd://49d6e2a850d3569145de92a1138b377a904954b48b23905af00b15ec5393de77",
        image: "k8s.gcr.io/serve_hostname:v1.4",
        imageID: "sha256:b56f22170982bc46dee99f7157be69dce727d3b89ee5698d15fda7a14a124200",
        lastState: {},
        name: "serve-hostname-xqnl5",
        ready: true,
        resources: {
          requests: {
            cpu: "100m",
          },
        },
        restartCount: 0,
        started: true,
        state: {
          running: {
            startedAt: "2025-05-09T21:14:41Z",
          },
        },
        volumeMounts: [{
          mountPath: "/var/run/secrets/kubernetes.io/serviceaccount",
          name: "kube-api-access-g57rz",
          readOnly: true,
          recursiveReadOnly: "Disabled",
        }],
      }],
      hostIP: "10.40.0.9",
      hostIPs: [{
        ip: "10.40.0.9",
      }],
      phase: "Running",
      podIP: "10.64.3.4",
      podIPs: [{
        ip: "10.64.3.4",
      }],
      qosClass: "Burstable",
      startTime: "2025-05-09T21:14:39Z",
    },
  }],
  kind: "List",
  metadata: {
    resourceVersion: "",
  },
}
```

### Test Plan

The new encoder will include robust tests to prove both its own output format
and its compatibility with standard YAML and JSON.

[X] I/we understand the owners of the involved components may require updates to
existing tests to make this code solid enough prior to committing the changes necessary
to implement this enhancement.

##### Prerequisite testing updates

None.

##### Unit tests

We will add unit tests for things like `omitempty` and `omitzero`, for strings
that YAML coerces into non-string types, self-marshaling, and so forth.

We will include tests which check specific input constructions against known
output, to ensure the subtleties of the specified syntax are correct. We will
unit test the kubectl format option.

We will also include a round-trip fuzz-test which populates a complex structure
(which exercises as many edge-cases as possible) with random data, renders that
to KYAML, parses that KYAML back into the original type (with the standard YAML
decoder), and compares the original and result objects. Doing this enough times
should give us fairly high confidence.

- `sigs.k8s.io/yaml/kyaml`: `2025-07-24` - `89.1%`
- `k8s.io/cli-runtime/pkg/printers/`: `2025-07-25` - `66.4%`

##### Integration tests

N/A

##### e2e tests

N/A

### Graduation Criteria

This KEP introduces a new env-var feature gate to prevent premature
dependencies from forming.

#### Alpha

The new env-var feature gate will be checked before allowing use of the "kyaml"
output option. If `$KUBECTL_KYAML` is not set to "true", the "kyaml" output
option will not be available.

- The `kubectl` output format `kyaml` is implemented
- The gate is disabled by default
- All unit tests are complete
- Gather feedback from developers and surveys

#### Beta

The new env-var feature gate is enabled by default, but still disableable. If
`$KUBECTL_KYAML` is not set to "false", the "kyaml" output option will be
available.

- Resolve any identified issues.

#### GA

The new env-var feature gate is removed, the "kyaml" output option is always
available.

- A proper specification is published
- The majority of examples in the kubernetes GitHub orgs are converted to KYAML
- No new bug reports

This KEP does not propose to add conformance tests.

### Upgrade / Downgrade Strategy

This is a client-side feature.  Users should not use `-o kyaml` in automation
until they are sure their clients are updated, but there is little risk of
breakage from ad hoc use.

KYAML is valid YAML. `kubectl` already accepts KYAML as defined here and with
arbitrary variations, as long as it remains valid YAML.

### Version Skew Strategy

N/A

## Production Readiness Review Questionnaire

### Feature Enablement and Rollback

###### How can this feature be enabled / disabled in a live cluster?

- [X] Other
  - Opt-in
  - The environment variable `KUBECTL_KYAML` can be set to "true" to enable
    the KYAML output format in `kubectl` commands or "false" to disable it.

 - Will enabling / disabling the feature require downtime of the control
    plane?

NO

  - Will enabling / disabling the feature require downtime or reprovisioning
    of a node?

NO

###### Does enabling the feature change any default behavior?

NO

###### Can the feature be disabled once it has been enabled (i.e. can we roll back the enablement)?

N/A - this is always opt-in and is entirely client-side

###### What happens if we reenable the feature if it was previously rolled back?

NOTHING

###### Are there any tests for feature enablement/disablement?

N/A

### Rollout, Upgrade and Rollback Planning

N/A

###### How can a rollout or rollback fail? Can it impact already running workloads?

N/A

###### What specific metrics should inform a rollback?

N/A

###### Were upgrade and rollback tested? Was the upgrade->downgrade->upgrade path tested?

N/A

###### Is the rollout accompanied by any deprecations and/or removals of features, APIs, fields of API types, flags, etc.?

NO

### Monitoring Requirements

N/A

###### How can an operator determine if the feature is in use by workloads?

N/A

###### How can someone using this feature know that it is working for their instance?

N/A

###### What are the reasonable SLOs (Service Level Objectives) for the enhancement?

N/A

###### What are the SLIs (Service Level Indicators) an operator can use to determine the health of the service?

N/A

###### Are there any missing metrics that would be useful to have to improve observability of this feature?

NO

### Dependencies

NONE

###### Does this feature depend on any specific services running in the cluster?

NONE

### Scalability

N/A

###### Will enabling / using this feature result in any new API calls?

NO

###### Will enabling / using this feature result in introducing new API types?

NO

###### Will enabling / using this feature result in any new calls to the cloud provider?

NO

###### Will enabling / using this feature result in increasing size or count of the existing API objects?

NO

###### Will enabling / using this feature result in increasing time taken by any operations covered by existing SLIs/SLOs?

NO

###### Will enabling / using this feature result in non-negligible increase of resource usage (CPU, RAM, disk, IO, ...) in any components?

Using this may incur some client-side resource usage, but marginal.

###### Can enabling / using this feature result in resource exhaustion of some node resources (PIDs, sockets, inodes, etc.)?

NO

### Troubleshooting

If someone thinks the KYAML rendering is wrong they can instead render to plain
YAML or JSON and compare.  Additionally, they can render to KYAML and then use
yamlfmt to convert that to plain YAML.

###### How does this feature react if the API server and/or etcd is unavailable?

N/A

###### What are other known failure modes?

It’s possible we missed some esoteric structure that renders incorrectly.
Those will be handled as bugs.

###### What steps should be taken if SLOs are not being met to determine the problem?

N/A

## Implementation History

May 09, 2025: KEP draft v0
May 26, 2025: KEP draft implementable
May 28, 2025: KEP is set to alpha stage
Aug 28, 2025: KEP is set to beta stage

## Drawbacks

Having a local dialect of YAML could be confusing to people who are used to
conventional YAML.

Trying to text-patch KYAML with plain YAML (as in a helm chart) will almost
certainly not work (mixing flow- and block-styles is a mess).

More serialization "languages" dilutes knowledge.

[Yet another standard](https://xkcd.com/927/). Rebuttal: KYAML is YAML and
users can provide vanilla YAML anywhere that can accept KYAML.

## Alternatives

1. We could just use JSON and deal with it.

1. We could tell users to use JSON with comments and a JSON minifier tool to
   strip those out before feeding to kubernetes.

1. We could "ban" the use of raw JSON (except to the API) and instead use
   JSON-with-comments (via YAML parsing) in all configuration logic.  This
   doesn’t solve for kubectl output.

1. We could add styling hints to go-yaml and call those instead of doing it
   ourselves.  This would require work in go-yaml, which is EOL.  Once the new
   revival-fork of it is alive, we can consider this path.  It is unclear if we
   would still call that "KYAML" or just "YAML".

1. We could adopt one of:
   - [HJSON](https://hjson.github.io/)
   - [JSONC](https://www.npmjs.com/package/jsonc)
   - [JSON5](https://json5.org/)

These would require choosing between them, taking new dependencies, and
learning new grammars.
All of them are very similar to KYAML, which has none of those drawbacks.
