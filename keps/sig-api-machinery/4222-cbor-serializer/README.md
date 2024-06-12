<!--
**Note:** When your KEP is complete, all of these comment blocks should be removed.

To get started with this template:

- [x] **Pick a hosting SIG.**
  Make sure that the problem space is something the SIG is interested in taking
  up. KEPs should not be checked in without a sponsoring SIG.
- [x] **Create an issue in kubernetes/enhancements**
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
- [x] **Fill out this file as best you can.**
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
# KEP-4222: CBOR Serializer

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
  - [Format](#format)
  - [Negotiation](#negotiation)
  - [Client Enablement](#client-enablement)
  - [User Stories (Optional)](#user-stories-optional)
    - [Story 1](#story-1)
    - [Story 2](#story-2)
  - [Notes/Constraints/Caveats (Optional)](#notesconstraintscaveats-optional)
  - [Risks and Mitigations](#risks-and-mitigations)
    - [Phased Implementation](#phased-implementation)
    - [Library Dependency](#library-dependency)
- [Design Details](#design-details)
  - [Why CBOR?](#why-cbor)
  - [Duplicate Map Keys and Unrecognized or Duplicate Field Names](#duplicate-map-keys-and-unrecognized-or-duplicate-field-names)
  - [Encoding Determinism](#encoding-determinism)
  - [Unicode](#unicode)
  - [Libraries](#libraries)
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

Under this proposal, Kubernetes API servers and clients will support the Concise
Binary Object Representation (CBOR) data format. CBOR will be available to
clients as an alternative to JSON for serializing resources in request and
response bodies. It will supersede JSON in apiextensions-apiserver for storage
serialization of custom resources.

## Motivation

<!--
This section is for explicitly listing the motivation, goals, and non-goals of
this KEP.  Describe why the change is important and the benefits to users. The
motivation section can optionally provide links to [experience reports] to
demonstrate the interest in a KEP within the wider Kubernetes community.

[experience reports]: https://github.com/golang/go/wiki/ExperienceReports
-->

In the course of processing a single request to the Kubernetes API, various
representations of a resource may be encoded and decoded several times by both
the client (encode request body, decode response body) and the server (decode
request body, decode from storage, encode response body, encode to storage). For
years, Kubernetes has supported a Protobuf format requiring dramatically less
CPU time and heap churn than its JSON (or YAML) format. The reduction in codec
overhead resulting from the adoption of Protobuf has made Kubernetes clusters
more efficient and able to handle increasingly heavy API traffic.

The Kubernetes community has embraced CustomResourceDefinitions (CRDs) as a
declarative extension mechanism for the Kubernetes API. Unlike native types,
custom resources can not trivially be serialized as Protobuf for serving or
storage. Protobuf is dependent on code generation, requires careful schema
evolution, and requires clients and servers to have compilation-time knowledge
of any Protobuf definitions they will use.

High-object-count and high-traffic custom resources are at a serious efficiency
disadvantage versus comparable native resources. Benchmarks suggest that custom
resource and dynamic client encode and decode operations can be made up to
approximately 8x and 2x faster, respectively, with a substantial reduction in
heap allocations, by adopting CBOR as a self-describing binary format.

### Goals

<!--
List the specific goals of the KEP. What is it trying to achieve? How will we
know that this has succeeded?
-->

- Reduce CPU time and heap churn of encode and decode along the request-response
  path when Protobuf can not be used, especially:
  - custom resource storage
  - custom resource serving
  - dynamic clients
  - apply configurations
  - strategic merge patches

### Non-Goals

<!--
What is out of scope for this KEP? Listing non-goals helps to focus discussion
and make progress.
-->

- Replace existing usage of Protobuf.
- Substantially reduce the size of encoded objects (a modest size reduction is
  anticipated).
- Replace all usage of YAML or JSON.

## Proposal

<!--
This is where we get down to the specifics of what the proposal actually is.
This should have enough detail that reviewers can understand exactly what
you're proposing, but should not include things like API designs or
implementation. What is the desired outcome and how do we measure success?.
The "Design Details" section below is for the real
nitty-gritty.
-->

### Format

The output of the CBOR encoder is a single tagged data item as specified in
“[Self-Described
CBOR](https://www.rfc-editor.org/rfc/rfc8949.html#name-self-described-cbor)”,
with no additional envelope. Self-described CBOR -- a tagged data item with tag
number 55799 -- has the same semantics as the same data item with no tag, with
the convenient property that its encoded form is always prefixed by
`0xd9d9f7`. By design, this prefix is never found at the beginning of a JSON
text and can be used as a “magic number” to distinguish the data format of a
stored object at rest.

To support decoding custom resources that have been stored as a mixture of JSON
and CBOR, the CBOR serializer will implement RecognizingDecoder by checking for
the prefix `0xd9d9f7`.

Streaming responses (i.e. watches) will be serialized as CBOR Sequences. A [CBOR
Sequence](https://www.rfc-editor.org/rfc/rfc8742.html) is a concatenation of
zero or more CBOR data items, with no additional framing. This is effectively
equivalent to the existing JSON stream serialization behavior and takes
advantage of the property that, like JSON objects – and unlike Protobuf messages
or non-object JSON documents, e.g. numbers – CBOR data items are
self-delimiting.

At the time of writing, watch events are encoded to a temporary buffer before
being passed to the frame writer. Frame writers can also assume that the byte
slice passed to each call of `Write` represents the complete contents of one
frame. The Protobuf frame writer takes advantage of both in order to determine a
frame's length prefix "for free". If this proposal were to require encoding
events using the effectively length-prefixed approach described in [Optimizing
CBOR Sequences for Skipping
Elements](https://www.rfc-editor.org/rfc/rfc8742.html#name-optimizing-cbor-sequences-f),
the CBOR frame writer would similarly need to know each event's encoded size.

One useful property of a self-delimiting encoding is described [in the CBOR
standard](https://www.rfc-editor.org/rfc/rfc8949.html#section-4.2.1-3.1):

> the self-delimiting nature of the CBOR encoding means that there are no two
> well-formed CBOR encoded data items where one is a prefix of the other

In other words, CBOR (and the existing JSON framing) can stream directly to and
from the wire without incurring additional copies on both sides of the
connection. If an encoding fails or is otherwise not completely received on the
other end, the fragment that _is_ received will not be well-formed and will
produce a decode error.

### Negotiation

Proactive content negotiation will be supported for clients that want to receive
CBOR-encoded responses using the MIME type “application/cbor” in the Accept
request header. For compatibility with API servers that don’t support CBOR,
clients should also accept “application/json” (with a lower quality factor) and
choose the appropriate decoder based on the Content-Type response header.

Streaming requests should use the MIME type for CBOR Sequences,
“application/cbor-seq”.

A new "+cbor" suffix will be accepted for the existing Server-Side Apply media
type "application/apply-patch" and identifies a CBOR-encoded apply
configuration. Similarly, "application/strategic-merge-patch+cbor" will be
accepted as the content type of a CBOR-encoded strategic merge patch.

CBOR will not be a supported encoding for JSON Patches or JSON Merge Patches
because both types are JSON documents by definition; supporting them would
require either defining parallel CBOR variants of each patch type, or
sacrificing the efficiency benefit of CBOR by transcoding to JSON on the server
side.

Clients can send CBOR-encoded request bodies with the appropriate Content-Type
to API servers that support CBOR. API servers that don’t support CBOR will
return status 415 (Unsupported Media Type). In client-go, for alpha, when a
RESTClient configured to encode requests with CBOR receives a 415, it will
permanently (for the life of the RESTClient) fall back to JSON for subsequent
requests. For GA, this fallback behavior will be changed to operate on a
per-(method, target resource) basis, and to consider acceptable fallback
content-types based on the value of the Accept header in a 415 response, [as
described in RFC 9110](https://httpwg.org/specs/rfc9110.html#status.415).

The client's mapping of (method, target resource) pairs to acceptable request
content type can be pre-populated from the request media types in OpenAPI
documents. This allows clients to bypass the initial request in the content-type
fallback mechanism, but is not required.

### Client Enablement

Clients can be explicitly configured to prefer CBOR as a request encoding as
they can today be configured to prefer Protobuf or JSON. In client-go, this
involves setting the `ContentType` field of `rest.ClientContentConfig`. The
default request content-type will remain JSON for a period of time post-GA; a
minimum of two minor versions, so that the oldest kube-apiserver within the
supported kubectl version skew will have CBOR support. The supported version
skew for aggregated API servers is much wider (infinite?). Encoding and decoding
resources from aggregated API servers that don't support CBOR will rely on the
content-type negotiation mechanisms described above.

Two client-side gates will be added as follows, using a common client-go gating
mechanism with specific details to be agreed by sig-api-machinery:

1. AllowCBOR: If disabled, clients configured to accept "application/cbor" will
   instead accept "application/json" with the same preference. Clients
   configured to write "application/cbor" will instead write
   "application/json". Patch requests with content types
   "application/apply-patch+cbor" or "application/strategic-merge-patch+cbor"
   will instead use "application/apply-patch+yaml" and
   "application/strategic-merge-patch+json", respectively.
1. PreferCBOR: If enabled _and_ AllowCBOR is enabled, The default request
   content-type (if not explicitly configured) becomes "application/cbor" and
   the dynamic client's request content-type becomes "application/cbor".

### User Stories (Optional)

<!--
Detail the things that people will be able to do if this KEP is implemented.
Include as much detail as possible so that people can understand the "how" of
the system. The goal here is to make this feel real for users without getting
bogged down.
-->

#### Story 1

#### Story 2

### Notes/Constraints/Caveats (Optional)

<!--
What are the caveats to the proposal?
What are some important details that didn't come across above?
Go in to as much detail as necessary here.
This might be a good place to talk about core concepts and how they relate.
-->

### Risks and Mitigations

<!--
What are the risks of this proposal, and how do we mitigate? Think broadly.
For example, consider both security and how this will impact the larger
Kubernetes ecosystem.

How will security be reviewed, and by whom?

How will UX be reviewed, and by whom?

Consider including folks who also work outside the SIG or subproject.
-->

#### Phased Implementation

Introducing a new data format comes with risks to most API endpoints. Errors
that lose or modify parts of a resource during encode/decode are a special
concern, as is the risk of being unable to decode an object from its encoded
form. Additionally, as soon as it becomes possible for users to enable the new
encoding, it must always remain possible to decode any custom resource that may
have been persisted (barring a forced storage encoding migration).

Before allowing it to be enabled in kube-apiserver at all, there will be a
phased implementation to establish confidence in the safety and correctness of
the serializer.

1. Make it a fatal error if kube-apiserver starts with support for CBOR (same
   with apiextensions-apiserver storage codec?).
1. Add CBOR library dependency and incrementally implement all unit and fuzz
   tests enumerated in the alpha criteria.
1. Make it possible, by code injection only, to allow CBOR in
   kube-apiserver. Keep the fatal error condition.
1. Implement all integration tests.
1. Complete other alpha criteria.
1. Expose using feature gate.

#### Library Dependency

Kubernetes will take a new dependency on a CBOR library, with associated risks:

- The library may become unmaintained or undermaintained, or our use cases may
  require a change/addition to the library that its maintainers are unwilling to
  accept.
  - Mitigation: Contribute features, fixes, and testing upstream. If necessary,
    accept owning a fork.
- Since the library will be used to decode untrusted input, it is a potential
  source of security vulnerabilities.
  - Mitigation: New fuzz tests.
  - Mitigation: Manual review of library source.

## Design Details

<!--
This section should contain enough information that the specifics of your
change are understandable. This may include API specs (though not always
required) or even code snippets. If there's any ambiguity about HOW your
proposal will be implemented, this is the place to discuss them.
-->

### Why CBOR?

CBOR is a binary data format initially developed in 2013, specified in [RFC
8949](https://www.rfc-editor.org/rfc/rfc8949.html), and assigned Internet
Standard number 94 by the IETF.

In addition to its mature specification, the stated design objectives of CBOR
are interesting to Kubernetes. In particular:

*All JSON data types are convertible to and from CBOR.* It should be possible to
represent all existing API objects in a CBOR encoding.

*Decoding does not require a schema ("self-describing").* No need to build
supporting machinery to generate and manage schemas, distribute them to clients,
and associate them with persisted objects.

*Encoding and decoding is "reasonably frugal" in CPU usage.* Not efficiency at
all costs, but suitability for "high-volume" applications is an explicit goal.

*Serialization is "reasonably compact".* Smaller than JSON, but not at the
expense of codec implementation complexity. Exploratory testing showed a fuzzed
v1 Pod was nearly 20% smaller than JSON. Like JSON, field names are present in
the encoded form due to the self-describing nature of CBOR.

### Duplicate Map Keys and Unrecognized or Duplicate Field Names

Existing serializers handle decoding of duplicate fields / map keys differently.

The JSON serializer:

1. keeps the last duplicate entry
1. records the duplicated key
1. continues decoding
1. returns a strict decoding error along with the decoded object
   1. the recognizing decoder treats data as recognized on strict decoding error
   1. field validation configures the handling of strict decoding errors
      encountered while decoding request bodies

The generated Protobuf marshalers keep the last duplicate entry (for both fields
and map entries) without producing a strict decoding error.

As a text format, JSON (or YAML) is more commonly edited by hand and so is more
prone to this sort of error. And although Kubernetes consistently decodes JSON
objects containing duplicate keys, the presence of duplicate keys indicates a
mistake. Protobuf is typically machine-generated, and decoders are expected to
be “last one wins” in the case of duplicated fields. So while it would be
unexpected for a Protobuf-encoded object to contain duplicate fields, the
interpretation of such an object is unambiguous.

A map containing duplicate keys is well-formed but invalid according to the CBOR
specification. Decoding a map containing duplicate keys will produce a decode
error.

Decoding a map with unrecognized fields (map keys that do not not correspond to
the name of a struct field's `json` tag name) is expected in cases where the
client is newer than the server, or where an object containing an unrecognized
field was transcoded from YAML or JSON to CBOR. A strict decoding error (as in
JSON) will be generated in this case. In the custom resource path, where objects
are decoded into `unstructured.Unstructured`, a schema-aware decoder wrapper is
responsible for reporting unknown fields as strict decoding errors.

Note that clients (e.g. kubectl) may choose to decode an object from a JSON or
YAML text representation containing duplicate keys, then encode to CBOR to
populate the body of an API request. Since the text-encoded content (potentially
containing duplicates) is not literally transcoded to CBOR, this use case is
supported. Depending on the strictness mode, duplicate keys would either be
removed or produce an error at decode time.

### Encoding Determinism

It is possible for a single object to be encodable as multiple distinct but
valid and semantically-equivalent CBOR byte strings. The CBOR specification does
not require encoder implementations to produce deterministic output, although it
does include recommendations for implementing deterministic encoding.

The etcd3 storage implementation of GuaranteedUpdate relies on deterministic
encoding to skip writes if the stored bytes would not change. The existing JSON
and Protobuf encoders produce deterministic output.

Other potential use cases for deterministic encoding of response bodies might
include:

- caching
  - The existing WithCacheControl filter sets the response header
    “Cache-Control: no-cache, private” to prevent shared caches from storing
    responses (since requests are subject to authn/authz), and to prevent
    responses in non-shared caches from being reused without
    validation. Deterministic encoding could allow an API server to generate
    strong ETags by hashing the encoded form of the resource.
  - Even for the existing data formats, there should be no caching proxies
    storing API responses.
- diffing
  - The human-readable text formats (JSON and YAML) are not changing under this
    proposal.

Encode benchmarks for the two evaluated Go CBOR libraries show a 2.4x speedup
and a 1.8x speedup by disabling map key sorting. According to the spec, “the
CBOR data model for maps does not allow ascribing semantics to the order of the
key/value pairs in the map representation.”  And since the CBOR decoder will
reject maps containing duplicate keys, a CBOR map represents exactly the same
set of key-value pairs regardless of the order they are encoded.

In order to take advantage of the available speedup, the CBOR encoder will
support separate deterministic and nondeterministic modes. The deterministic
mode will be used for storage serialization only. The nondeterministic mode
should introduce randomness into the order of map item encoding (as with map
iteration in Go) to make it easier to detect invalid assumptions about the
order, but not in a way that adds significant overhead.

To further mitigate the risk that the output of the nondeterministic encoder
mode will be accidentally used in cases that require determinism (bytewise
equality, hashing, etc.), and because output determinism is implicitly part of
the contract of `runtime.Encoder`, the CBOR encoder will also implement a new
interface:

```go
type NondeterministicEncoder interface {
  NondeterministicEncode(runtime.Object, io.Writer) error
}
```

Callers that don't require output determinism will perform a conditional type
assertion and invoke `NondeterministicEncode` in place of `Encode`.

### Unicode

CBOR supports distinct major types for text strings and byte strings. Text
strings that do not contain a valid UTF-8 sequence are well-formed but invalid
CBOR. Unlike JSON strings, CBOR text strings do not support any escape
sequences.

The JSON serializer replaces invalid UTF-8 sequences with the Unicode
replacement character (u+fffd) during both encode and decode. This is consistent
with the behavior of encoding/json in the Go standard library. Generated
Protobuf marshal and unmarshal code neither validates nor coerces strings; the
byte sequence is directly copied on both encode and decode.

To avoid accepting invalid CBOR, the decoder will produce an error if a text
string is not a valid UTF-8 sequence. Strings will follow the precedent
established by Protobuf and be encoded using CBOR's byte string type, except in
cases where the encoder can be sure that the string is a valid UTF-8
sequence. This ensures the serializer will not encode an object to a byte
sequence that it will not successfully decode.

### Libraries

|                                      | github.com/ugorji/go/codec | github.com/fxamacker/cbor/v2 |
|--------------------------------------|----------------------------|------------------------------|
| license                              | MIT                        | MIT                          |
| text string utf-8 coercion           | none                       | none                         |
| decode: text string utf-8 validation | {error, ignore}            | {error, ignore}              |
| decode: duplicate map key            | ignore                     | {error, ignore}              |
| decode: unknown field name           | {error, ignore}            | {error, ignore}              |
| decode: case-sensitivity             | yes                        | no                           |
| unsafe                               | yes (disable by build tag) | no                           |
| fuzzed                               | no                         | maybe                        |

[Benchmarks](https://docs.google.com/spreadsheets/d/1yi8cHrnlbmCUY2Vo7Sknrf87WDOuGUswYsyqJfEUwls/edit#gid=0) TODO: inline



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

Tests for the following behaviors will be added:

- decoding a map containing duplicate keys into a Go map produces an error
- decoding a map containing duplicate keys into a Go struct produces an error
- roundtripping preserves the distinction between integers and floating-point
  numbers
- decoding a text string containing an invalid UTF-8 sequence produces an error
- decoding a map into a Go struct matches `json` field tag names
  case-sensitively
- when decoding a map into a Go struct, a case-insensitive match between a key
  and a `json` field tag name is treated the same as no match
- encoding a struct with duplicate field names (`json` tag names) does not
  result in a map containing duplicate keys ([https://go.dev/issue/17913])
- pooled buffers should not grow and be retained forever
  ([https://go.dev/issue/23199])
- decoding into a Go interface{} stores only either `nil` concrete values or
  concrete values of type `bool`, `string`, `int64`, `float64`, `[]interface{}`,
  or `map[string]interface{}` (no special treatment of tagged content producing
  time.Time, math/big.Int, etc.)
- conformance to CBOR specification (adopt existing suite and/or develop as
  necessary)
  - this should be demonstrated to run against implementations in at least some
    of the non-Go client languages
- Go strings that are not valid UTF-8 sequences can be roundtripped through CBOR
  without error
- decoding a map into a Go struct produces a strict decoding error if the map
  contains a key that does not correspond to JSON tag name of one of the
  struct's fields
- roundtripping preserves the distinction between absent, present-but-null, and
  present-and-empty for slices and maps
- `runtime.RawExtension`
  - re-encoding preserves the original raw bytes
  - encoding a runtime.Object with existing no raw bytes defaults to JSON
  - decoding JSON-in-CBOR, JSON-in-Protobuf, CBOR-in-JSON, and CBOR-in-Protobuf
    is supported

As well as fuzz tests covering:

- for all native types, native-to-JSON-to-unstructured and
  native-to-CBOR-to-unstructured is identical
- the number of bytes allocated per decode does not exceed a reasonable upper limit
- roundtrip JSON-to-CBOR-to-JSON and CBOR-to-JSON-to-CBOR
- roundtrip through implementations in at least some of the non-Go client
  languages

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

- custom resources storage encoding is CBOR with feature gate enabled
- custom resources storage encoding is JSON with feature gate disabled
- response content-type negotiation works and honors indicated preference
  (Protobuf > CBOR > JSON)
- get, list, watch, update, delete, deletecollection, and scale support CBOR
  using dynamic and generated clients for all native types
- mixed CBOR and JSON encodings in storage for a single custom resource can be
  retrieved with feature gate disabled
- client gating mechanism:
  - can force clients otherwise configured with a CBOR request encoding to use JSON
  - can change the default request encoding to CBOR if not explicitly configured
  - can be disabled programmatically
- request content-type falls back to JSON and does not try CBOR again for a
  given (method, target resource) pair

##### e2e tests

<!--
This question should be filled when targeting a release.
For Alpha, describe what tests will be added to ensure proper quality of the enhancement.

For Beta and GA, add links to added tests together with links to k8s-triage for those tests:
https://storage.googleapis.com/k8s-triage/index.html

We expect no non-infra related flakes in the last month as a GA graduation criteria.
-->

- request and response content-type negotiation with 1.17 sample API server

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

#### Alpha

- All new tests enumerated in "Test Plan" are implemented.
- Feature gate wired to kube-apiserver.
- Dynamic client updated to support CBOR behind client-side gates.
- Client generation updated to support CBOR behind client-side gates.
- Runtime gating mechanism added to client-go.
- Maintenance of CBOR library is understood.

#### Beta

- Review of nondeterministic encoding mode and final decision on whether to keep
  or remove it.

#### GA

- Granular content-type fallback behavior on HTTP 415.
- Ability to bypass content-type fallback behavior using OpenAPI.

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
- Does this enhancement involve coordinating behavior in the control plane and nodes?
- How does an n-3 kubelet or kube-proxy without this feature available behave when this feature is used?
- How does an n-1 kube-controller-manager or kube-scheduler without this feature available behave when this feature is used?
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

- [x] Feature gate (also fill in values in `kep.yaml`)
  - Feature gate name: CBORSerializer
  - Components depending on the feature gate:
    - kube-apiserver

###### Does enabling the feature change any default behavior?

<!--
Any change of default behavior may be surprising to users or break existing
automations, so be extremely careful here.
-->

Enabling the feature changes the default storage encoding of custom resources to
CBOR, but this should be invisible to clients.

###### Can the feature be disabled once it has been enabled (i.e. can we roll back the enablement)?

<!--
Describe the consequences on existing workloads (e.g., if this is a runtime
feature, can it break the existing applications?).

Feature gates are typically disabled by setting the flag to `false` and
restarting the component. No other changes should be necessary to disable the
feature.

NOTE: Also set `disable-supported` to `true` or `false` in `kep.yaml`.
-->

Yes, with the exception of support for CBOR decoding of custom resources from
storage. That cannot be disabled because it must remain possible to decode any
resource that has already been persisted.

With CBOR is disabled on the server side, resources that have been persisted
using the CBOR encoding can be replaced with their JSON encoding by retrieving
the resource as JSON and writing it back unaltered. This is the same process
used for storage version migrations and can be automated using the Storage
Version Migrator.

###### What happens if we reenable the feature if it was previously rolled back?

No additional considerations. Custom resource storage will support recognition
and decoding of both JSON and CBOR whether the feature is enabled or disabled.

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

There will be integration tests that ensure custom resources that have been
stored with a mixture of CBOR and JSON encodings continue to be accessible with
the feature gate disabled, and integration tests for client
enablement/disablement.

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

If a client is configured to encode a request body using CBOR, and that request
is handled by an API server that does not have CBOR enabled, the API server will
send response status 415 (Unsupported Media Type) and the client will repeat the
request using JSON. This is not expected to produce a substantial number of
additional requests because:

1. the default request encoding for clients will not be modified until CBOR
   support is widespread (beyond GA and accounting for version skew)
1. individual clients will limit failed attempts at using CBOR as request
   content-type for any given verb and target resource

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

No.

###### Will enabling / using this feature result in increasing size or count of the existing API objects?

<!--
Describe them, providing:
  - API type(s):
  - Estimated increase in size: (e.g., new annotation of size 32B)
  - Estimated amount of new objects: (e.g., new Object X for every existing Pod)
-->

No. Objects counts will not be affected. Storage and most serving of native
types will continue to use Protobuf and will be unaffected. Traffic from dynamic
clients, and storage of custom resources, should be modestly more
compact. Although not a goal of this proposal, pods encoded as part of
benchmarking were approximately 20% smaller with CBOR than with JSON.

###### Will enabling / using this feature result in increasing time taken by any operations covered by existing SLIs/SLOs?

<!--
Look at the [existing SLIs/SLOs].

Think about adding additional work or introducing new steps in between
(e.g. need to do X to start a container), etc. Please describe the details.

[existing SLIs/SLOs]: https://git.k8s.io/community/sig-scalability/slos/slos.md#kubernetes-slisslos
-->

No.

###### Will enabling / using this feature result in non-negligible increase of resource usage (CPU, RAM, disk, IO, ...) in any components?

<!--
Things to keep in mind include: additional in-memory state, additional
non-trivial computations, excessive access to disks (including increased log
volume), significant amount of data sent and/or received over network, etc.
This through this both in small and large cases, again with respect to the
[supported limits].

[supported limits]: https://git.k8s.io/community//sig-scalability/configs-and-limits/thresholds.md
-->

No.

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
