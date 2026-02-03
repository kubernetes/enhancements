<!--
**Note:** When your KEP is complete, all of these comment blocks should be removed.

Follow the guidelines of the [documentation style guide].
In particular, wrap lines to a reasonable length, to make it
easier for reviewers to cite specific portions, and to minimize diff churn on
updates.

[documentation style guide]: https://github.com/kubernetes/community/blob/master/contributors/guide/style-guide.md

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
# KEP-5589: Remove gogo protobuf dependency for Kubernetes API types

<!-- toc -->
- [Release Signoff Checklist](#release-signoff-checklist)
- [Summary](#summary)
- [Motivation](#motivation)
  - [Goals](#goals)
  - [Non-Goals](#non-goals)
- [Proposal](#proposal)
  - [User Stories](#user-stories)
    - [Story 1](#story-1)
    - [Story 2](#story-2)
    - [Story 3](#story-3)
  - [Notes/Constraints/Caveats](#notesconstraintscaveats)
  - [Risks and Mitigations](#risks-and-mitigations)
- [Design Details](#design-details)
  - [Test Plan](#test-plan)
      - [Prerequisite testing updates](#prerequisite-testing-updates)
      - [Unit tests](#unit-tests)
      - [Integration tests](#integration-tests)
      - [e2e tests](#e2e-tests)
  - [Graduation Criteria](#graduation-criteria)
    - [Deprecation](#deprecation)
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
- [Possible Future Work](#possible-future-work)
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
  - [ ] (R) [all GA Endpoints](https://github.com/kubernetes/community/pull/1806) must be hit by [Conformance Tests](https://github.com/kubernetes/community/blob/master/contributors/devel/sig-architecture/conformance-tests.md) within one minor version of promotion to GA
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

This KEP describes a path to remove runtime and generation-time dependencies on gogo protobuf,
while preserving existing behavior and minimizing impact on consumers of Kubernetes API Go types.

## Motivation

The toolchain to generate protobuf marshal / unmarshal support for Kubernetes REST API types
uses https://github.com/gogo/protobuf. This dependency is unmaintained since 2021.

Reliance on an unmaintained and frozen dependency for complex code generation means
that addressing any future vulnerability, bug, or feature requirement will require
switching out that dependency first, possibly in a time-critical scenario.

The gogo-generated code also includes run-time references to gogo packages.
These package dependencies are propagated to all Kubernetes module consumers,
and make an unmaintained module appear in their dependency graph.
Using core Kubernetes libraries should not require consumers to inherit unmaintained dependencies.

The generated code also *appears* to implement interfaces to interoperate with standard Go protobuf libraries
(google.golang.org/protobuf and github.com/golang/protobuf), but these libraries are only intended to interoperate
with types generated by the official protoc-gen-go tool.
Using gogo-generated types with standard protobuf libraries has led to [runtime panics in other projects](https://github.com/etcd-io/etcd/pull/12398#discussion_r505322880),
and currently only partially works via [best-effort tolerance of "aberrant messages"](https://github.com/protocolbuffers/protobuf-go/commit/23ccb359e1a18a9fc77be08aeff34f00d79f4f11).
Kubernetes client and server machinery do not actually use these libraries for Kubernetes API protobuf handling,
but the apparent ability to use these types as "standard" protobuf messages is misleading.
Reducing the generated protobuf code for Kubernetes API types to *only* what is used for client / server machinery
will minimize accidental and possibly dangerous misuse of these types in combination with standard protobuf libraries.

### Goals

1. Leave protobuf serialization / deserialization code used by API machinery unchanged
2. Leave non-protobuf usage of Kubernetes API Go types unchanged
3. Reduce the possibility of incorrectly using Kubernetes API objects with standard protobuf libraries
4. Eliminate runtime use of unmaintained gogo packages in Kubernetes API protobuf-handling
5. Eliminate generation-time use of unmaintained gogo packages by Kubernetes API protobuf-handling
6. Ensure all tooling involved in producing Kubernetes protobuf-handling code is able to be updated,
   if needed, to fix vulnerabilities, bugs, improve performance, or add capabilities

### Non-Goals

1. Switch to generating Kubernetes REST API Go types using standard `protoc-gen-go` tooling
2. Enable use of Kubernetes Go types with standard protobuf libraries

## Proposal

Step 1: Identify the exact methods used by Kubernetes Go clients and servers
required to encode and decode Kubernetes API objects to and from protobuf.

Step 2: Define interfaces within Kubernetes for *exactly* those methods, without referencing gogo packages.

Step 3: Update non-generated code within Kubernetes to use those interfaces instead of referring to gogo packages.
This accomplishes goal 4 (no runtime use of gogo packages for the Kubernetes API) for non-generated code.

Step 4: Update the post-gogo-generation Go-rewriting step already present in
[k8s.io/code-generator/cmd/go-to-protobuf](https://github.com/kubernetes/code-generator/tree/v0.34.0/cmd/go-to-protobuf)
to remove or isolate via build tags generated protobuf code that is unused, removing all runtime references to gogo packages,
while still satisfying the interfaces required for apimachinery protobuf handling.
This accomplishes goal 3 by removing generated aspects of Kubernetes API types like proto descriptors
which *appeared* to allow standard proto libraries to interact with those objects,
but were unused within Kubernetes and not actually supported by the standard proto libraries.
This also accomplishes goal 4 (no runtime use of gogo packages for the Kubernetes API) for generated code.

Step 5: Fork the gogo generator (either to a sigs.k8s.io repo or to a subpackage of k8s.io/code-generator/cmd/go-to-protobuf),
modify it to produce identical output to the post-gogo-generation Go-rewriting step of go-to-protobuf,
and drop the Go-rewriting logic from k8s.io/code-generator/cmd/go-to-protobuf.
This accomplishes goal 5 and 6 (no code-generation use of gogo packages,
all generation tooling is in a location where it can be modified / fixed if needed).

All of these steps leave the methods used to marshal / unmarshal Kubernetes API types either untouched
or trivially inspectable to be equivalent (like changing a pass-through call to stdlib `sort.Strings` to a direct call).
That accomplishes goal 1 of leaving serialization / deserialization code unchanged.

All of these steps can be accomplished without changing non-protobuf aspects of Kubernetes API Go types,
which accomplishes goal 2 of leaving the Go types unchanged from previous releases.

### User Stories

#### Story 1

As a consumer of Kubernetes API types and client libraries (k8s.io/api and k8s.io/client-go),
I have no runtime dependencies on the unmaintained github.com/gogo/protobuf module.

Upgrading to a Kubernetes release including changes from proposal steps 1-4
removes the gogo dependency from all client modules.

#### Story 2

As a consumer of Kubernetes API types and client libraries like k8s.io/api, k8s.io/client-go,
and k8s.io/apimachinery, I can upgrade Kubernetes modules without modifying how I construct
Kubernetes API Go instances or access fields.

The changes in this proposal do not modify fields of Kubernetes API types.

#### Story 3

As a maintainer of Kubernetes code-generator components, I can merge changes to
address vulnerabilities, bugs, and feature requirements in protobuf-handling.

Once the protobuf generating code is forked from gogo/protobuf to k8s.io/code-generator
or a sigs.k8s.io repo, it can be modified if needed to address vulnerabilities, bugs,
and feature requirements.

### Notes/Constraints/Caveats

**Constraint: Compatibility with existing marshal / unmarshal behavior**

Kubernetes API objects use protobuf wire encoding, but differ from typical protobuf behavior in two key ways:

1. Canonical output for a given version is supported.
   This is a requirement of Kubernetes API object serialization,
   and is relied on by the API server storage layer to ensure repeated
   no-op write requests do not re-persist different bytes to storage.
   Canonical output is an [explicit non-goal](https://protobuf.dev/programming-guides/serialization-not-canonical/) of the standard protobuf libraries.

2. Preservation of unknown fields is intentionally avoided.
   Only fields which are known to the current API server version are allowed to be persisted and returned.
   Preservation of unknown fields when decoding / round-tripping through the server is a non-goal for Kubernetes,
   but is a non-configurable default behavior for `protoc`-generated marshaling / unmarshaling.

These two key differences preclude switching to using `protoc` / `protoc-gen-go` generated marshaling / unmarshaling.

**Constraint: Verifiable compatibility**

Any changes in marshal / unmarshal implementation must be able to be verified to behave identically.
Due to the complexity of protobuf marshaling / unmarshaling logic, any non-inspectable change would be difficult to prove behaves identically.

**Constraint: Avoiding dangerous or disruptive changes to Kubernetes API types**

Kubernetes API Go types are *widely* used by non-generated Go code to:
* construct objects
* read fields
* write fields

Changing to `protoc`/`protoc-gen-go`-generated objects would:
1. change all fields to pointers, making every direct access of existing non-pointer fields risk a panic
2. add additional unexported fields to objects to handle things like unknown field preservation and size caching
3. add shallow-copy-unsafe fields to objects, making any code passing non-pointer instances no longer pass vet checks
4. change the Go names of API fields containing initialisms (like `URL`)
5. remove the ability to customize JSON tags for backwards compatibility

These changes would impact non-protobuf usage of Kubernetes API types,
and preclude switching to using `protoc` / `protoc-gen-go` generated marshaling / unmarshaling.

### Risks and Mitigations

**Risk: consumers of removed generated code will have to adjust usage.**

Removed methods and their replacement are:
* `XXX_Unmarshal` --> `Unmarshal`
* `XXX_Marshal` --> `MarshalToSizedBuffer`
* `XXX_Merge` --> No replacement ([no public evidence](https://github.com/search?q=language%3AGo+-org%3Agogo+-repo%3Acosmos%2Fgogoproto+-path%3A%2Fvendor%7Ctest%7Cexample%7Cgogo%2F+%22.XXX_Merge%28%22&type=code) of use of this method with Kubernetes types)
* `XXX_Size` --> `Size`
* `XXX_DiscardUnknown` --> Remove, was a no-op
* `Descriptor` --> No replacement
* `ProtoMessage` --> Remove, was a no-op (or re-enable via a `kubernetes_protomessage_one_more_release` build tag for one release while eliminating use)

Mitigation: there is no use of these methods across all
[kubernetes](https://github.com/search?q=language%3AGo+-org%3Agogo+-repo%3Acosmos%2Fgogoproto+-path%3A%2Fvendor%7Ctest%7Cexample%7Cgogo%2F+%2F%5C.%28XXX_Unmarshal%7CXXX_Marshal%7CXXX_Merge%7CXXX_Size%7CXXX_DiscardUnknown%7CDescriptor%7CProtoMessage%29%5C%28%2F+org%3Akubernetes&type=code)
and
[kubernetes-sigs](https://github.com/search?q=language%3AGo+-org%3Agogo+-repo%3Acosmos%2Fgogoproto+-path%3A%2Fvendor%7Ctest%7Cexample%7Cgogo%2F+%2F%5C.%28XXX_Unmarshal%7CXXX_Marshal%7CXXX_Merge%7CXXX_Size%7CXXX_DiscardUnknown%7CDescriptor%7CProtoMessage%29%5C%28%2F+org%3Akubernetes-sigs&type=code)
repos, and scans of public github did not uncover use of these with Kubernetes API types specifically.

**Risk: global registration of Kubernetes API types into gogo-proto global registries is removed.**

Two `init()`-time registrations of Kubernetes API types into gogo-proto global registries will be removed:
- github.com/gogo/protobuf/proto#RegisterType
  - calls to github.com/gogo/protobuf/proto#MessageType will now return nil reflect.Type
  - calls to github.com/gogo/protobuf/proto#MessageName will now return an empty string
- github.com/gogo/protobuf/proto#RegisterFile
  - calls to github.com/gogo/protobuf/proto#FileDescriptor will now return an empty string

Mitigation: there is no use of these methods across all
[kubernetes](https://github.com/search?q=language%3AGo+-org%3Agogo+-repo%3Acosmos%2Fgogoproto+-path%3A%2Fvendor%7Ctest%7Cexample%7Cgogo%2F+%2F%5C.%28MessageType%7CMessageName%7CFileDescriptor%29%5C%28%2F+org%3Akubernetes&type=code)
and
[kubernetes-sigs](https://github.com/search?q=language%3AGo+-org%3Agogo+-repo%3Acosmos%2Fgogoproto+-path%3A%2Fvendor%7Ctest%7Cexample%7Cgogo%2F+%2F%5C.%28MessageType%7CMessageName%7CFileDescriptor%29%5C%28%2F+org%3Akubernetes-sigs&type=code)
repos, and scans of public github did not uncover use of these with Kubernetes API types specifically.

**Risk: marshal / unmarshal code is unintentionally modified**

Mitigations:
* This KEP prioritizes leaving the in-use marshal / unmarshal code paths untouched,
  focusing on straight removal of unused code paths and imports instead,
  or modification of generators that leave generated code unmodified.
* Existing compatibility tests ensure protobuf data from previous releases is round-tripped losslessly

## Design Details

**Truncate exported methods**

Only the exported methods used by `k8s.io/apimachinery` protobuf handling or with other widespread use will be kept in default builds:

* Unmarshaling:
    ```go
    Reset()
    Unmarshal([]byte) error
    ```
* Marshaling:
    ```go
    Size() int
    Marshal() ([]byte, error)
    MarshalTo(data []byte) (int, error)
    MarshalToSizedBuffer(data []byte) (int, error)
    ```

* Other
    ```go
    String() string
    ```

We will truncate the exported methods from protobuf generation to just those methods.

The `ProtoMessage()` marker method will be relocated to build-tagged files,
which library consumers can enable with a `kubernetes_protomessage_one_more_release`
build tag for a single release as a build mitigation if they rely on this method.
After one minor release, this will be removed as well.

**Truncate imports**

The only gogo packages used in generated code for anything other than a type assertion are:

* The `sortkeys` package, which is a pass-through to the stdlib `sort.Strings`.
  This will be replaced with a direct call to stdlib `sort.Strings`.
* The `proto` package, used for `init()`-time registration of types and file descriptors.
  The `init()` time global registrations are not required by apimachinery and will be dropped.

**Truncate exported variables**

Only the exported variables used by retained methods will be kept:
* `ErrInvalidLengthGenerated`
* `ErrIntOverflowGenerated`
* `ErrUnexpectedEndOfGroupGenerated`

**Truncate unused private variables and methods**

After the truncation of exported methods, variables, and `init()`-time registration,
other imports, variables, and methods which are now unused will be dropped.

**Fork gogo generator**

The subset of the gogo code generation used by `k8s.io/code-generator/cmd/go-to-protobuf`
will be forked to a location within the `kubernetes` or `kubernetes-sigs` project,
and pruned / modified to only output the subset of generated code kept by `go-to-protobuf`.

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

- `k8s.io/apimachinery/pkg/runtime/serializer/protobuf`: `2025-09-29` - `53.1%`

##### Integration tests

There is no feature enablement / disablement, configurable behavior, or runtime behavior change associated with this KEP.

##### e2e tests

There is no feature enablement / disablement, configurable behavior, or runtime behavior change associated with this KEP.

### Graduation Criteria

First phase:
  * Complete steps 1-4: Removal of runtime use of gogo packages by API types and clients
  * Result: No gogo packages are referenced from `k8s.io/api`, `k8s.io/apimachinery`, and `k8s.io/client-go` .go / go.mod files

Second phase:
  * Complete step 5: Removal of code-generation use of gogo packages
  * Result: No gogo packages are referenced from any staging .go file, or as a direct dependency from any Kubernetes go.mod file

#### Deprecation

n/a

### Upgrade / Downgrade Strategy

The changes proposed in this KEP do not modify built code or runtime behavior.
There is no difference in serialized data from the client or server relative to other versions.

### Version Skew Strategy

The changes proposed in this KEP do not modify built code or runtime behavior.
There is no difference in serialized data from the client or server relative to other versions.

## Production Readiness Review Questionnaire

### Feature Enablement and Rollback

###### How can this feature be enabled / disabled in a live cluster?

The changes in this KEP are to code generation, and primarily result in removal of unused generated code.
There is no runtime enablement / disablement of these changes.

###### Does enabling the feature change any default behavior?

No.

###### Can the feature be disabled once it has been enabled (i.e. can we roll back the enablement)?

The code-generator changes could be reverted, restoring deleted generated protobuf code,
but that is a development / build-time change, not a runtime / configuration change.

###### What happens if we reenable the feature if it was previously rolled back?

Restoring unused deleted generated code is not expected to have any effect on runtime behavior.

###### Are there any tests for feature enablement/disablement?

No.

### Rollout, Upgrade and Rollback Planning

###### How can a rollout or rollback fail? Can it impact already running workloads?

Failures related to the changes at this KEP are expected to surface at build time, not runtime.

###### What specific metrics should inform a rollback?

There are no runtime code changes or behavior changes associated with this proposal.

###### Were upgrade and rollback tested? Was the upgrade->downgrade->upgrade path tested?

No, there is no runtime code change or behavior change associated with this proposal.
Cross-version compatibility tests for protobuf marshaling / unmarshaling ensure
marshal / unmarshal code at HEAD is compatible with protobuf data from previous versions.

###### Is the rollout accompanied by any deprecations and/or removals of features, APIs, fields of API types, flags, etc.?

No REST API fields, types, or command-line flags are deprecated or removed.
Several unused generated protobuf-related methods will be removed:
* `XXX_Unmarshal`
* `XXX_Marshal`
* `XXX_Merge`
* `XXX_Size`
* `XXX_DiscardUnknown`
* `Descriptor`

The `ProtoMessage` marker method for use with standard protobuf libraries
will be isolated in build-tagged files, which can be temporarily reenabled with a
`kubernetes_protomessage_one_more_release` build tag for one minor release.

### Monitoring Requirements

###### How can an operator determine if the feature is in use by workloads?

Protobuf read / write paths are exercised by default on every kube-apiserver startup
for all built-in API types by the watch cache.

###### How can someone using this feature know that it is working for their instance?

If kube-apiserver builds and starts successfully, then protobuf writing to etcd is working properly.

###### What are the reasonable SLOs (Service Level Objectives) for the enhancement?

Status quo for protobuf marshaling / unmarshaling.
The changes proposed in this KEP do not modify built code or runtime behavior.

###### What are the SLIs (Service Level Indicators) an operator can use to determine the health of the service?

The changes proposed in this KEP do not modify built code or runtime behavior.

###### Are there any missing metrics that would be useful to have to improve observability of this feature?

Content-type of incoming requests could be a useful tool in determining
whether errors are correlated with a particular content type, like protobuf.
However, request metrics have historically had a problematically large number of labels,
adding one more for content-type was discussed and discarded in the past for that reason.

### Dependencies

###### Does this feature depend on any specific services running in the cluster?

No

### Scalability

###### Will enabling / using this feature result in any new API calls?

No

###### Will enabling / using this feature result in introducing new API types?

No

###### Will enabling / using this feature result in any new calls to the cloud provider?

No

###### Will enabling / using this feature result in increasing size or count of the existing API objects?

No

###### Will enabling / using this feature result in increasing time taken by any operations covered by existing SLIs/SLOs?

No

###### Will enabling / using this feature result in non-negligible increase of resource usage (CPU, RAM, disk, IO, ...) in any components?

No

###### Can enabling / using this feature result in resource exhaustion of some node resources (PIDs, sockets, inodes, etc.)?

No

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

There is no runtime behavior change associated with this KEP.

###### What steps should be taken if SLOs are not being met to determine the problem?

## Implementation History

* v1.2: original protobuf support proposal [PR](https://github.com/kubernetes/kubernetes/pull/22600)
([final archive location](https://github.com/kubernetes/design-proposals-archive/blob/main/api-machinery/protobuf.md)).
* v1.2: generate .proto files from Go types ([#17854](https://github.com/kubernetes/kubernetes/pull/17854)),
        write back proto tags to Go types ([#19426](https://github.com/kubernetes/kubernetes/pull/19426))
* v1.3: add protobuf serializer ([#20377](https://github.com/kubernetes/kubernetes/pull/20377))
* v1.4: customize String() ([#28815](https://github.com/kubernetes/kubernetes/pull/28815))
* v1.7: make map serialization deterministic ([#47701](https://github.com/kubernetes/kubernetes/pull/47701))
* 2020: gogo / golang/protobuf incompatibilities stall dependency updates, panic etcd libraries, get resolved in a best-effort way
* 2021-2022: gogo transitions to deprecated / unmaintained
* v1.35: This KEP proposed
* v1.35: code-generator updated to remove runtime dependencies from generated code ([#134256](https://github.com/kubernetes/kubernetes/pull/134256))

## Drawbacks

Taking on responsibility for complex protobuf generation code is not ideal.
However, I would claim that the Kubernetes project already has implicit responsibility
for the *output* of the unmaintained generator, so I don't think a maintenance-mode fork
whose only consumer is the Kubernetes API protobuf generator is significantly different
than the current state.

## Alternatives

Two alternatives were considered:

1. Switching to standard `protoc`/`protoc-gen-go` generation.
   The impact to non-protobuf use of Kubernetes API go types,
   and the mismatch between Kubernetes requirements around canonical serialization
   and dropping of unknown fields ruled this out. See the [Notes/Constraints/Caveats](#notesconstraintscaveats)
   section for more details.

2. Switching to another non-standard protobuf generation (e.g. `vtprotobuf`).
   This would have required significantly more work to achieve,
   and would have been significantly more difficult to verify compatibility.
   There were not clear benefits that would justify this amount of time investment.
   The approach described in this KEP does not preclude such a change in the future,
   if it becomes apparent there are benefits that would justify a change of that magnitude,
   and a plan is proposed to make that change safely.

## Possible Future Work

The following are *possible* future changes that could be made to the protobuf generator
once it is in a location that can be modified in ways scoped to Kubernetes API types.
These changes are not planned as part of this KEP, but are captured here for reference.

**Add omitzero support**

Current protobuf serialization *always* outputs non-pointer scalar and struct fields.
This means that feature-gate-disabled fields must be created as pointers so they will not be serialized unless populated.
Adding support for an `omitzero` tag to protobuf serialization would allow for non-pointer fields which can still be feature-gate-disabled.

**Simplify String() implementation**

There is widespread reliance on Kubernetes API objects implementing `String()`,
so preserving the existence of `String()` implementations is necessary to minimize disruption.

However, this only *happens* to be done as part of protobuf generation,
and the output is a pseudo-Go structure that requires large amounts of code to construct,
and is not particularly meaningful from an API perspective.

After the code-generation is forked and modifiable, the `String()` implementation could be
switched to a simpler implementation, for example returning a JSON encoding of the object.

## Infrastructure Needed (Optional)

n/a
