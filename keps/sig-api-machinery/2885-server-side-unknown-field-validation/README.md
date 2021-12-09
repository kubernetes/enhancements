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
# KEP-2885: Server Side Unknown Field Validation

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
  - [Notes/Constraints/Caveats (Optional)](#notesconstraintscaveats-optional)
    - [Future Work](#future-work)
  - [Risks and Mitigations](#risks-and-mitigations)
  - [Performance Considerations](#performance-considerations)
- [Design Details](#design-details)
  - [Opt-in API Mechanism (Query Parameter)](#opt-in-api-mechanism-query-parameter)
  - [Create (POST) and Update (PUT)](#create-post-and-update-put)
  - [Patch (PATCH)](#patch-patch)
    - [JSON Patch](#json-patch)
    - [Strategic Merge Patch](#strategic-merge-patch)
    - [Apply Patch](#apply-patch)
    - [Kubectl Flag](#kubectl-flag)
  - [Test Plan](#test-plan)
  - [Graduation Criteria](#graduation-criteria)
    - [Alpha](#alpha)
    - [Beta](#beta)
      - [Alpha Follow Ups](#alpha-follow-ups)
- [Production Readiness Review Questionnaire](#production-readiness-review-questionnaire)
  - [Feature Enablement and Rollback](#feature-enablement-and-rollback)
  - [Rollout, Upgrade and Rollback Planning](#rollout-upgrade-and-rollback-planning)
  - [Monitoring Requirements](#monitoring-requirements)
  - [Dependencies](#dependencies)
  - [Scalability](#scalability)
  - [Troubleshooting](#troubleshooting)
- [Implementation History](#implementation-history)
- [Alternatives](#alternatives)
  - [Content-Type Header](#content-type-header)
  - [Other Alternatives Considered](#other-alternatives-considered)
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
- [x] (R) Design details are appropriately documented
- [x] (R) Test plan is in place, giving consideration to SIG Architecture and SIG Testing input (including test refactors)
  - [ ] e2e Tests for all Beta API Operations (endpoints)
  - [ ] (R) Ensure GA e2e tests for meet requirements for [Conformance Tests](https://github.com/kubernetes/community/blob/master/contributors/devel/sig-architecture/conformance-tests.md) 
  - [ ] (R) Minimum Two Week Window for GA e2e tests to prove flake free
- [x] (R) Graduation criteria is in place
  - [ ] (R) [all GA Endpoints](https://github.com/kubernetes/community/pull/1806) must be hit by [Conformance Tests](https://github.com/kubernetes/community/blob/master/contributors/devel/sig-architecture/conformance-tests.md) 
- [x] (R) Production readiness review completed
- [ ] (R) Production readiness review approved
- [x] "Implementation History" section is up-to-date for milestone
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
As a client sending a create, update, or patch request to the server, I want to
be able to instruct the server to fail when the kubernetes object I send has
fields that are not valid fields of the kubernetes resource.

This will allow us to remove client-side validation from kubectl while
maintaining the same core functionality of erroring out on requests that contain
unknown or invalid fields.

## Motivation

<!--
This section is for explicitly listing the motivation, goals, and non-goals of
this KEP.  Describe why the change is important and the benefits to users. The
motivation section can optionally provide links to [experience reports] to
demonstrate the interest in a KEP within the wider Kubernetes community.

[experience reports]: https://github.com/golang/go/wiki/ExperienceReports
-->
`kubectl –validate=true` is the current mechanism to indicate that a request
should fail if it specifies unknown fields on the object.

There are a few issues with this as highlighted by the [previous
effort](https://docs.google.com/document/d/18nrtJ0gizVHnIhx5NIkGXvSaQ1wT2i6XVDUmmQqc7_4/edit#heading=h.q9616qdce0l9) to do
server-side validation, primarily these issues include:
* Bug Fixes are utilized slower because client-side upgrades are hard to get in
people’s hands.
* Each client needs to implement validation.

This is the last remaining step for removing client-side validation according to
[this
comment](https://github.com/kubernetes/kubernetes/issues/39434#issuecomment-270486105)
* Client-side validation is [very
  painful](https://github.com/kubernetes/kubernetes/issues/39434#issuecomment-270443399)

Additional problems have been highlighted in the relevant github issues:
* https://github.com/kubernetes/kubernetes/issues/39434
* https://github.com/kubernetes/kubernetes/issues/5889
The community has been asking for this feature as recently as [August 8th,
2021](https://github.com/kubernetes/kubernetes/issues/104090#issuecomment-895228540)

### Goals

<!--
List the specific goals of the KEP. What is it trying to achieve? How will we
know that this has succeeded?
-->
* Server should validate that no extra fields are present or invalid (e.g.
misspelled), nor are any fields duplicated (for json and yaml data).
* We must maintain compatibility with all existing clients, thus server side
unknown field validation should be opt-in.
* kubectl should provide the ability for a user to request server-side
  validation.

### Non-Goals

<!--
What is out of scope for this KEP? Listing non-goals helps to focus discussion
and make progress.
-->
* Complete (business-logic) server-side validation of every aspect of an object
(i.e. we’re only focused on mismatched fields between the object in the request
body and its schema).
* Protobuf support. Theoretically, unknown protobuf fields could occur if clients
on version X send a request to the server of version less than X which does not recognize
some of the fields (or a similar situation). We do not think it is worth
supporting this use case initially and will error if clients attempt to validate
schema server-side with protobuf data.


## Proposal

<!--
This is where we get down to the specifics of what the proposal actually is.
This should have enough detail that reviewers can understand exactly what
you're proposing, but should not include things like API designs or
implementation. What is the desired outcome and how do we measure success?.
The "Design Details" section below is for the real
nitty-gritty.
-->

We propose adding server-side schema validation, enabling the API server
to validate that objects received in the request body in POST, PUT, and
PATCH requests contain no unknown or duplicate fields.

We introduce a query parameter that enables the user/client to indicate
that server-side validation should either not be performed at all, provide
warnings for the presence of unknown/duplicate fields, or error out in the
presence of unknown/duplicate fields.

For beta, we propose modifying the kubectl `--validate` flag to enable users
to choose between client-side validation or server-side
validation (either strict or warning)

### Notes/Constraints/Caveats (Optional)

#### Future Work

Upon successfully providing kubectl access to server-side validation via the
`--validate` flag, an open question will remain as to the future of client
side validation.

While we would like kubectl to use server-side validation by default once it
is GA, the question remains whether client-side validation be optionally
available to kubectl users or completely deprecated.

Maintaining client-side validation indefinitely provides the benefit of
allowing users to validate their objects without needing access to a server.
On the other hand, the downside of continuing to maintain client-side
validation is that it uses openapiv2 which will be less actively supported
as openapiv3 becomes standard and will cause client-side validation to
continue to diverge from server-side validation.

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

### Performance Considerations
It is worth noting that checking for unknown fields whether via a json unmarshal
that explicitly breaks if it encounters unknown fields (as in the case of
create, update, or json patch) or extra logic around a merge step (as in
strategic merge patch or apply) will be less performant than not doing so.
[Initial
benchmarks](https://github.com/kubernetes/kubernetes/pull/104433#issuecomment-901398507) estimate ~20% slower 25-30% more memory consumption.

This is deemed acceptable and insignificant because the use case we are
replacing is client side validation that happens from kubectl (i.e. via a human
that should not be too impacted by a slight performance hit). Any existing
automated clients that would be impacted by performance do not currently have a
way of leveraging server side validation and thus will remain opted-out of this
by default (or will be willing to tradeoff the performance if they do choose to
opt-in to server side schema validation).


## Design Details

<!--
This section should contain enough information that the specifics of your
change are understandable. This may include API specs (though not always
required) or even code snippets. If there's any ambiguity about HOW your
proposal will be implemented, this is the place to discuss them.
-->
### Opt-in API Mechanism (Query Parameter)

There are a few ways we could allow a client to opt-in to server-side unknown
field validation, we believe the best option is to introduce a new query
parameter to indicate the level of field validation the server should perform,
such as `?fieldValidation=Strict`.

Primary arguments for using query parameter include:
* Being able version them with the API version.
* Parameters are discoverable in openapi.
* We have precedent for using query parameters for write requests already (via CreateOptions,
  PatchOptions, UpdateOptions)

An alternative to using a query parameter, using content-type header is
discussed in the [#alternatives](#alternatives) section.

For client-go support, we will add a `FieldValidation` field to
CreateOptions, PatchOptions, and UpdateOptions that can supply the query
parameter to the request.

In addition to supporting the options `Strict` (for erroring on unknown fields) and `Ignore`,
we will also support the `?fieldValidation=Warn` option, to warn, rather than error on
unknown fields.

We believe that using a query parameter is the best approach.

### Create (POST) and Update (PUT)

Implementation of this validation for Patch requests differ significantly and
are discussed in the Patch section below.

At a high level, for create and update requests we have the opportunity to
validate the object when we unmarshal the object from wire format into the go
type.

This happens in the [serializer
implementation](https://github.com/kubernetes/kubernetes/blob/master/staging/src/k8s.io/apimachinery/pkg/runtime/serializer/json/json.go#L264-L291), where depending on if a “strict”
option is set (or not), the unmarshalling step will error (or not) if
extra/malformatted fields exist on the data being unmarshalled.

This in turn, gets used by
[create](https://github.com/kubernetes/kubernetes/blob/master/staging/src/k8s.io/apiserver/pkg/endpoints/handlers/create.go#L95-L120) and [update](https://github.com/kubernetes/kubernetes/blob/master/staging/src/k8s.io/apiserver/pkg/endpoints/handlers/update.go#L106-L107)  handlers when they attempt to
decode the request body.

One major advantage of the “Content-Type” header approach is that it requires
little to no change existing code in the request handlers.

When the API server is started and we create the [negotiated
serializer](https://github.com/kubernetes/kubernetes/blob/master/staging/src/k8s.io/apiserver/pkg/server/genericapiserver.go#L616), we
create a [separate serializer](https://github.com/kubernetes/kubernetes/blob/master/staging/src/k8s.io/apimachinery/pkg/runtime/serializer/codec_factory.go#L167) for each content type/media type. To support strict
validation all we would need to do is add [another
serializer](https://github.com/kubernetes/kubernetes/blob/master/staging/src/k8s.io/apimachinery/pkg/runtime/serializer/codec_factory.go#L51-L106) to the
NegotiatedSerializer for strict validation that is identical to the serializer
of “application/json” but has the “strict” option set (as well as ones for
yaml).

The request handler will then proceed unchanged, and the [Decode
step](https://github.com/kubernetes/kubernetes/blob/master/staging/src/k8s.io/apiserver/pkg/endpoints/handlers/create.go#L120) will fail
if it encounters fields in the request body that are not part of the schema.

### Patch (PATCH)

Unlike create and update requests, patch requests are more complicated because
they do not ever unmarshal/decode the request body into a kubernetes object.
Instead patching involves, serializing the existing object to json, then using a
jsonpatch or mergepatch library to combine the patch sent from the client with
the existing object, and finally deserializing the combined object into the
kubernetes object.

During the jsonpatch or mergepatch step we don’t currently check if the patch
sent by the client has any erroneous fields. Now we need to.

This all happens in the
[patchMechanism](https://github.com/kubernetes/kubernetes/blob/master/staging/src/k8s.io/apiserver/pkg/endpoints/handlers/patch.go#L293-L296) used by the patch handler.

There are three types of patchers used by the patch handler: jsonPatcher,
smpPatcher, and applyPatcher that are each discussed individually.

#### JSON Patch

JSON patching is most commonly called when client-side apply looks to patch an
existing custom resource.

For the jsonPatch implementation of
[applyPatchToCurrentObject](https://github.com/kubernetes/kubernetes/blob/master/staging/src/k8s.io/apiserver/pkg/endpoints/handlers/patch.go#L304), This is similar
to create and update where after using the jsonpatch library to generate the
[patchedObjJS](https://github.com/kubernetes/kubernetes/blob/master/staging/src/k8s.io/apiserver/pkg/endpoints/handlers/patch.go#L312) we then call [DecodeInto](https://github.com/kubernetes/kubernetes/blob/master/staging/src/k8s.io/apiserver/pkg/endpoints/handlers/patch.go#L319). 

Based on the presence of the validate query param, we should be able to use a
codec that is or is not doing strict decoding that will fail, in the strict
case, when invalid fields are present on the patched json blob.

Currently, like create and update, the handler generates the [serializer from the
scope](https://github.com/kubernetes/kubernetes/blob/master/staging/src/k8s.io/apiserver/pkg/endpoints/handlers/patch.go#L136-L146) and the codec from the serializer. We just need to ensure that it
conditionally uses a StrictSerializer based on the presence of the validate
query param.

#### Strategic Merge Patch

Strategic merge patching is most commonly called when client-side apply attempts
to patch an existing object of a builtin type.

For the strategic merge patch (SMP) implementation, it calls into the
apimachinery strategicpatch library to update the fields of the original object
with the data from the patch. This goes through a few layers but it eventually
boils down to this
[mergeMap](https://github.com/kubernetes/kubernetes/blob/dadecb2c8932fd28de9dfb94edbc7bdac7d0d28f/staging/src/k8s.io/apimachinery/pkg/util/strategicpatch/patch.go#L1280) call that does the actual merge of the patch into
the original.

`mergeMap` goest field-by-field through the existing object, checking to see if
data should be merged in from the patch. Currently, if there are any
extra/unknown fields in patch, they will never be encountered and will just be
ignored.

`mergeMap` takes a mergeOptions argument that we can update to have some
configuration for strict validation. Within `mergeMap`, we can track the fields
of the patch as they are meged in the existing object. After all fields that can
be merged from the patch are, we will know that any fields on the patch that
were not merged are unknown fields and can error out.

#### Apply Patch

Apply Patching is called whenever we send a server side apply request to the
server.

For apply patch, server side schema validation already occurs and is not
optional, no new behavior is needed here. The exception to this is that one can
create an arbitrary schema that is still a valid structural schema  (see [docs](https://kubernetes.io/docs/tasks/extend-kubernetes/custom-resources/custom-resource-definitions/#specifying-a-structural-schema) on
structural schema). If one manages to do this and permits unknown fields, then
server-side schema validation will not be responsible for invalidating these.

For further context, the applyPatch path of the patch handler calls into the
fieldmanager’s [Apply](https://github.com/kubernetes/kubernetes/blob/9ff3b7e744b34c099c1405d9add192adbef0b6b1/staging/src/k8s.io/apiserver/pkg/endpoints/handlers/patch.go#L441) method in order to generate the newly patched object that
will be stored to etcd

This in turn, calls
[apply](https://github.com/kubernetes/kubernetes/blob/9ff3b7e744b34c099c1405d9add192adbef0b6b1/staging/src/k8s.io/apiserver/pkg/endpoints/handlers/fieldmanager/structuredmerge.go#L106) on it’s own internal fieldmanager (which is what
actually implements the joining of the live object to the patch object to
produce the new object). It attempts to [convert the patch object into a
TypedValue](https://github.com/kubernetes/kubernetes/blob/9ff3b7e744b34c099c1405d9add192adbef0b6b1/staging/src/k8s.io/apiserver/pkg/endpoints/handlers/fieldmanager/structuredmerge.go#L129). This checks the object against its schema and will error if the
object has any unknown or duplicate fields.


#### Kubectl Flag

For beta, we will modify the kubectl `--validate` flag from being a bool flag to
a string flag that accepts the following values:

* `server`: This sends a request with the fieldValidation param set to `Strict`
for strict server-side validation.
* `client`: This performs client-side validation in kubectl that breaks if it
encounters any validation errors before sending a request to the server. If no
client-side validation errors are found, the request sent to the server has
fieldValidation param set to `Ignore`. This will be the default behavior in beta
as it is identical to the current validation behavior pre-1.24
* `ignore`: This performs no client-side validation or server-side validation,
sending a request to the server with fieldValidation param set to `Ignore`.
* `warn`: This sends a request to the server with the fieldValidation param set
  to `Warn`. It performs server-side validation, but validation errors are
  exposed as warnings in the result header rather than failing the request. This
  will be the default once server-side validation goes GA.
* `strict`: same behavior as `--validate=server`.
* `true`: This is the same as `--validate=client`, but throws a warning along
  the lines of "`--validate=true` is being deprecated, use `--validate=client`
  instead". This is also the default behavior if `--validate` flag is passed
  withouth a value set.
* `false`: This is the same as `--validate=ignore`, but throws a warning along
  the lines of "`--validate=false` is being deprecated use `--validate=ignore`
  instead".


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
We can unit test and benchmark the performance of each endpoint in
[apiserver/pkg/endpoints/apiserver_test.go](https://github.com/kubernetes/kubernetes/blob/dadecb2c8932fd28de9dfb94edbc7bdac7d0d28f/staging/src/k8s.io/apiserver/pkg/endpoints/apiserver_test.go). We will need to test that requests
fail with invalid fields for all types of fields (embedding, free-form fields,
etc).

Additionally, we can add integration testing for all endpoints in
[test/integration/apiserver/apiserver_test.go](https://github.com/kubernetes/kubernetes/blob/master/test/integration/apiserver/apiserver_test.go)

We will also have additional testing for changes to the strategic merge patch
logic in
[apimachinery/pkg/util/strategicpatch/patch_test.go](https://github.com/kubernetes/kubernetes/blob/dadecb2c8932fd28de9dfb94edbc7bdac7d0d28f/staging/src/k8s.io/apimachinery/pkg/util/strategicpatch/patch_test.go)

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

-->
#### Alpha

- Feature implemented behind a feature gate
- Integration tests added for all relevant verbs (POST, PUT, and PATCH)

#### Beta

- kubectl validate flag offers ability to perform server-side validation
- All follow-ups from alpha are complete (see below)

##### Alpha Follow Ups

These are some tasks that were initially planned for alpha, but did not make it
and should be addressed before beta is considered complete.
- [ ] endpoints handler unit testing of field validation
- [ ] customresource handler unit testing of field validation
- [ ] field validation integration tests check for exact match of strict errors
- [ ] [sigs.k8s.io/yaml should
  report](https://github.com/kubernetes/kubernetes/pull/105916#discussion_r748530682) strict errors consistent with kjson
- [ ] [make
  use](https://github.com/kubernetes/kubernetes/pull/105916#discussion_r746125180) of the decoded object rather than decoding multiple times
- [ ]
  [decide](https://github.com/kubernetes/kubernetes/pull/105916#discussion_r750769609) whether a fieldValidation=Strict fatal error should return before
  an error from the apply operation.
- [ ] [Look into](https://github.com/kubernetes/kubernetes/issues/106558) strict handling of nested object decoding.


<!--
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

### Upgrade / Downgrade Strategy

If applicable, how will the component be upgraded and downgraded? Make sure
this is in the test plan.

Consider the following in developing an upgrade/downgrade strategy for this
enhancement:
- What changes (in invocations, configurations, API use, etc.) is an existing
  cluster required to make on upgrade, in order to maintain previous behavior?
- What changes (in invocations, configurations, API use, etc.) is an existing
  cluster required to make on upgrade, in order to make use of the enhancement?

### Version Skew Strategy

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
- [x] Feature gate (also fill in values in `kep.yaml`)
  - Feature gate name: ServerSideFieldValidation
  - Components depending on the feature gate: kube-apiserver
- [x] Other
  - Describe the mechanism: query parameter
  - Will enabling / disabling the feature require downtime of the control
    plane? NO
  - Will enabling / disabling the feature require downtime or reprovisioning
    of a node? (Do not assume `Dynamic Kubelet Config` feature is enabled). NO

###### Does enabling the feature change any default behavior?

No, strict validation is false by default.

###### Can the feature be disabled once it has been enabled (i.e. can we roll back the enablement)?

Yes. From the cluster operator's side, they can restart the kube-apiserver without
the ServerSideFieldValidation feature gate set and this will disable the feature
cluster-wide.

For end-users that no longer wish to perform server-side strict validation,
simply not setting the query param will ensure the server is not doing strict
validation.

###### What happens if we reenable the feature if it was previously rolled back?

No harm, requests will resume performing strict validation. The content of
stored data does not change when the feature is used compared to when it is off,
only new requests to the api-server will trigger strict validation.

###### Are there any tests for feature enablement/disablement?

Testing for both presence and absence of query param when the feature is enabled. We also test
that modified codepaths that could be triggered when the feature is enabled are
inert when the feature is disabled.

For example, unknown field path tracking in both the unstructured converter and
structural pruning algorithm affect performance when server-side field
validation is performed, but are tested to ensure that they are never invoked
when server side field validation is disabled.

### Rollout, Upgrade and Rollback Planning

###### How can a rollout or rollback fail? Can it impact already running workloads?

API servers rolled out with field validation enabled could start erroneously
failing requests if there are problems with how field validation is implemented.

Also, performance issues could slow down API server performance or memory usage
if extraneous validation is being invoked.

###### What specific metrics should inform a rollback?

A drastic deterioration in API server performance could indicate an issue with
the feature and a need to rollback (i.e. `apiserver_request_duration_seconds`).

###### Were upgrade and rollback tested? Was the upgrade->downgrade->upgrade path tested?

No

###### Is the rollout accompanied by any deprecations and/or removals of features, APIs, fields of API types, flags, etc.?

For beta we will be warning kubectl users that `--validate={true,false}` is
being deprecated in favor of `--validate={client,server,strict,warn,ignore}`.

### Monitoring Requirements

###### How can an operator determine if the feature is in use by workloads?

One could create a metric to track the number of validation errors on the
server. We will not be implementing this for beta as it seems like overkill.

###### How can someone using this feature know that it is working for their instance?

The easiest way to know that the feature is enabled and working is to query the
API server with the query param set and passing an object with deliberately
invalid, unknown fields. An error from the server will indicate that the feature
is enabled as expected.

Alternatively, and end-user can cross-reference the results of client-side
validation (i.e. `kubectl --validate=true`) with the results of server-side. If
client-side validation indicates an error but server-side does not, then the
feature is disabled.

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

Users should expect to see an increase in request latency and memory usage
(~20-25% increase) for requests that desire server side schema validation.

Cluster operators can also use cpu usage monitoring to determine whether
increased usage of server-side schema validation dramatically increases CPU usage after feature enablement.

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

Pick one more of these and delete the rest.

- [x] Metrics
  - Metric name: apiserver_request_duration_seconds
  - Components exposing the metric: kube-apiserver

###### Are there any missing metrics that would be useful to have to improve observability of this feature?

As mentioned above, one could implement a metric to track the quantity of field validation errors
reported by the apiserver. Unnecessary because it provides insight about how
clients are misusing the API (by sending invalid fields), more so than it
provides any useful insight on the feature itself.

### Dependencies

###### Does this feature depend on any specific services running in the cluster?

No

### Scalability

<!--
For alpha, this section is encouraged: reviewers should consider these questions
and attempt to answer them.

For beta, this section is required: reviewers must answer these questions.

For GA, this section is required: approvers should be able to confirm the
previous answers based on experience in the field.
-->

###### Will enabling / using this feature result in any new API calls?

No

###### Will enabling / using this feature result in introducing new API types?

No

###### Will enabling / using this feature result in any new calls to the cloud provider?

No

###### Will enabling / using this feature result in increasing size or count of the existing API objects?

No

###### Will enabling / using this feature result in increasing time taken by any operations covered by existing SLIs/SLOs?

<!--
Look at the [existing SLIs/SLOs].

Think about adding additional work or introducing new steps in between
(e.g. need to do X to start a container), etc. Please describe the details.

[existing SLIs/SLOs]: https://git.k8s.io/community/sig-scalability/slos/slos.md#kubernetes-slisslos
-->
Mutating API calls that opt-in to validation will be slower.
Benchmarks of alpha changes indicate that performing validation results in
~2-5% slower request latency and ~4-8% more memory usage.

###### Will enabling / using this feature result in non-negligible increase of resource usage (CPU, RAM, disk, IO, ...) in any components?

Non-negligible increase to RAM of kube-apiserver when server-side validation is
heavily utilized.

### Troubleshooting

###### How does this feature react if the API server and/or etcd is unavailable?

Same as without the feature

###### What are other known failure modes?

N/A

###### What steps should be taken if SLOs are not being met to determine the problem?

Opt out of server-side validation if the performance impact of it is not
tolerable.

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
* 2021-11-19 Alpha Implementation
  [PR](https://github.com/kubernetes/kubernetes/pull/105916) of Server Side Field Validation merged.
* 2021-12-07 Kubernetes 1.23 released with alpha implementation present.
* 2021-12-08 Better documentation for fieldValidation parameter
  [merged](https://github.com/kubernetes/kubernetes/pull/106722).

<!--
## Drawbacks

Why should this KEP _not_ be implemented?
-->

## Alternatives

### Content-Type Header

Instead of opting-in to server-side validation via a query parameter, we
considered passing a specific content-type header that would indiciate strict
schema validation.

This could be done by passing a mime-type parameter indicating strict validation
such as `application/json;validation=strict` or we could use an entirely new
header such as `X-Kubernetes-Validation:Strict`.

One could argue that this is the more appropriate way to parameterize opting-in
to server side schema validation because on the server we are fundamentally
treating the content as a different type (“application/json” is json data that
we don’t care if it has extra fields, “application/json;validation=strict” is
json data that is sensitive to extra fields).

A precedent for using a mime-type parameter is from how we [receive resources as
tables](https://kubernetes.io/docs/reference/using-api/_print/#receiving-resources-as-tables).

On the other hand, one could also argue that interpreting input strictly or not
according to the target schema is independent of the content type and that this
is an inappropriate way to parameterize validation opt-in.

Another argument against this method is that for patch, strictness is handled
when decoding the *result* of the patch, so it wouldn’t make sense to use
Content-Type which is providing information about the inbound request body.


### Other Alternatives Considered

* Passing multiple decoders around in the request scope
* Change the Decode signature itself (or adding a new DecodeStrict that Decode
  calls into)
* Instead of erroring on unknown fields, we could warn on unknown fields and
  then more generally have a `TreatWarningsAsErrors` option to fail on requests
  with unknown fields. One drawback here is that it would be difficult to
  flag unknown fields separately from other warnings that we might not want to
  error on (e.g. using a deprecated API). See note in [#proposal](#proposal)
  section around adding an option for warning on unknown fields.

<!--
What other approaches did you consider, and why did you rule them out? These do
not need to be as detailed as the proposal, but should include enough
information to express the idea and why it was not acceptable.
-->

<!--
## Infrastructure Needed (Optional)

Use this section if you need things from the project/SIG. Examples include a
new subproject, repos requested, or GitHub details. Listing these here allows a
SIG to get the process for these resources started right away.
-->
