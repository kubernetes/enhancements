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
# KEP-3617: Fine-Grained Authorization

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
  - [User Stories (Optional)](#user-stories-optional)
    - [Story 1](#story-1)
    - [Story 2](#story-2)
  - [Notes/Constraints/Caveats (Optional)](#notesconstraintscaveats-optional)
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

Add fine-grained authorization checks to mutating kubernetes api calls.

"fine-grained" means giving the ability to certain actors (users, groups,
controllers, etc) to edit only particular fields, instead of entire objects.
Additionally, it involves adding the ability to restrict existing board
permissions to existing fields; such that if desired, we can introduce new
fields and not grant broad permissions on them by default.


## Motivation

We wish to [add a new field](https://github.com/kubernetes/enhancements/pull/2840),
but restrict the actors who can write to it by default.

If this ability had existed in the past, we would have used it for `.metadata.finalizers`
and likely for `.metadata.ownerReferences`

The problem is that setting these fields has implications for other objects,
potentially objects which a different set of users have permission on.

Until now we have used subresources to solve this problem, but we have realized
that they have a number of problems (see alternative design section) and it is
best not to add additional uses of that pattern.

### Goals

* Add a mechanism for granting certain users generic ability to set
  (write/update) certain fields, without granting them the ability to set other
  fields in objects.
* Add a mechanism for restricting certain fields from users otherwise having
  general mutating permissions on an object type.

### Non-Goals

* Changing permissions for existing fields. This feature should have no
  observable changes unless configured by administrators.

## Proposal

Let's introduce the user stories before the design:

### User Stories

#### Story 1

I wish to restrict an operator "supersafe" to write only:

* labels with keys beginning with "super.safe.com/"
* annotations with keys beginning with "super.safe.com/"
* finalizers with the prefix "super.safe.com/"
* conditions with the type "SuperSafe"

#### Story 2

I'm adding a new field to pod.spec, "triggerSelfDestructOnSecurityBreach".
Ordinary users must not be able to set this to true. Only admins may set it to
true.

#### List of known fields wanting these features

* .metadata (all objects)
  * .metadata.labels (parameterized by key)
  * .metadata.annotations (parameterized by key)
  * .metadata.finalizers (parameterized by prefix)
  * .metadata.liens (doesn't exist yet; will be parameterized)
  * .metadata.ownerReferences (maybe)
* .status
  * .status.conditions (many objects but not all; parameterized by specific type)
* .spec
  * .spec.replicas (multiple objects)
  * .spec.nodeName (pod only)
  * .spec.schedulePaused (pod only; doesn't exist yet)

### Design

We will describe a system of additional authorization checks apiserver will
perform on mutating requests. First we need to explain some background and state
the rest of the requirements, otherwise the design offered will seem too
complex.

#### Authorization System Background and Requirements

On start-up, apiserver constructs an "authorizor" which is an object that
understands how to correctly make authorization checks (henceforth "authz
checks"). Users / cluster administrators can configure these checks to be done
in a number of different ways. RBAC is the most obvious choice, since it is
provided along with Kubernetes, but many clusters are integrated with external
authz systems (using plugins).

People offering reasonably dangerous extensions / new fields need assurances
that these things can be secured in any environment, and that they can offer
instructions on doing so mostly independently of the backend authz system
(because they may not even be able to identify all implementations).

Therefore, __modifying a single authz system can't solve the problem__.

An authz check can be thought of as sending structured data to an opaque system
which answers yes or no (individual pieces of the authorizor chain may also have
no opinion, but the chain as a whole answers yes or no). Essentially we can't
change things about the system ("opaque") because the system could comprise any
number of implementations, many not even open-source. The data we can give the
system includes:

* The resource type (group, version, resource)
* The object's locator (namespace if any, name)
* The actor's identity (username, group, user.Extra, user.UID)
* The verb, which is a free-form string with some known values

The verb is typically literally the HTTP verb being used, but we have in the
past made up logical verbs for more specific checks. Importantly, all authz
implementations permit administrators to enter arbitrary verbs, since the set of
verbs can't be easily enumerated.

__This KEP will describe a system for producing special authz verbs__.

The authz system is designed to be reasonably high-volume; it is called at least
once per Kubernetes API call already. We can add a few extra calls especially if
they don't need to be done on every request. But it would not be OK to e.g. do
an authz check for every field in an update request, because that could
multiply authz load by a factor of hundreds. So: __we must stick to a reasonable
number of new authz checks__, certainly not more than O(log(number of fields))
and hopefully even smaller than that.

__This KEP will describe the places within API server where the new verbs will
be checked.__ We are not modifying the authz system itself; we are making
additional calls to it, of newly defined verbs.

#### Schema Specification and Requirements

First, note that many of the fields we know we wish to protect exist in
__built-ins and custom resources (CRs); we'll support both__.

There is already precedent for specifying attributes of fields directly in the
schema. See server-side apply tags, CEL validations, documentation, etc. We
propose a similar mechanism for API authors to state required special verbs.
This will permit us to put those verbs in the documentation to make it __easy
for cluster admins to discover__.

The system for specifying custom verbs needs to permit:

* Verbs which cover child fields; e.g. we need to be able to grant "all of
  spec" instead of a (possibly unenumerable) list of verbs for each individual
  field. We need this to keep the number of checks low.
* Verbs which cover specific leaf fields, for example all labels in .metadata.labels.
* Verbs which are parameterized in some way, examples given earlier. The schema
  needs to state how the verb is parameterized.
* Whether the verb is excluded from or included in the generic privilege. This
  is needed to achieve our second goal. Exclusion is expected to be very rare --
  only new fields will be supported. Marking existing fields this way would
  break existing clients unless the corresponding permission is added, which we
  can't universally do (see authz section above).
* If a field name changes between versions of an object, the permission verb
  should not. Otherwise it will be extremely laborious to configure the system
  to ensure permissions don't change when objects are accessed via different
  versions.
* Fields representing the same concept in *different* objects should share the
  *same* permission. The authz system already has a spot for resource type,
  which can be specific or wildcard based. We should not encode resource type
  into the verb also.
* Because the permissions are distinct from fields, we should do our best to
  make the permissions (verb strings) not confusable for field names or paths.

The exact permissions are going to be a shared responsibility between extension
authors and in-tree Kubernetes developers, since anyone can author a CRD.

As part of this KEP, we will update the schemas to supply permission verbs for
the fields listed above. We will document the patterns used so that extension
authors can make their own permission verbs which won't conflict with each other
or with in-tree verbs. We may not know exactly what the best practices are until
we have an alpha or beta behind us, hence we're not ready to specify that in
this KEP yet.

(The exact permission system will be described below because we're not done with
the requirements yet, keep reading!)

#### API Server Additional Plumbing and Checks

Currently apiserver performs the authz check before descending into any detailed
logic for mutating operations. This means that when it is doing the authz check,
it knows the *object* that is being changed, but nothing about *which fields*
are changing.

We will introduce a new permission, "granular". This permission
means that although the caller doesn't have generic update permissions, it may
still change some fields if it has the specific permissions required.

API Server already maintains schema information for all resource types. This
will be watched to also maintain field <-> permission mappings.

Now we can describe the modifications to API Server:

On any mutating call,

1. Check the "PUT" / "PATCH" / "CREATE" permission. If the actor has this,
   proceed to step 3.
2. Otherwise, check the "granular" permission. If yes, place a
   marker in the request context and proceed to step 3, otherwise, fail the
   request with a forbidden error.
3. Compute the change (patch logic, SSA logic, defaulting etc). Compute a list
   of fields which changed. (SSA logic makes this easy.)
4. Pre- or post- (DECISION NEEDED) webhooks, check the list of fields:
5. If the marker is present, every field needs to be covered by some permission.
   Check permissions, stop and fail if some field is not covered. (The order is
   described below.)
6. Otherwise (no marker) ONLY fields having "excluded" permissions need to be
   checked; if such fields are modified, check their permissions as in step 5.
7. If all checks pass, perform the rest of the needed operation.
8. If any check fails, fail the request. The error message will not list ALL
   permissions that are lacked, because it is possible to craft a request that
   requires a large number of authz requests.

When fine-grained field permissions need to be checked, they will be checked
in the order of most general to most specific. When we describe the
representation in the schema it will be clear what that means. The reason for
this is that multiple fields can be covered by a more general permission,
greatly reducing the number of checks needed in the worst case (as long as the
system administrator has made use of the general permission).

### Notes/Constraints/Caveats (Optional)

This is a complex design. See the alternatives for why we propose it anyway.

### Risks and Mitigations

This design avoids the "grant an overly broad permission and then restrict it
later" pattern, which has undesirable failure / misconfiguration patterns.

This design is written to have no impact on existing cluster users for existing
fields.

A risk is that it would be possible to configure permissions which result in a
large number of authz checks. Specifically CRD authors could craft a CRD with
many checks required to do anything.

A runtime risk is an actor with the "granular" permission could make requests
changing many fields it doesn't have permission for, with the aim of driving
many authz checks to overload the authz system. To mitigate this we (a) have the
"granular" permission at all, to reduce the number of actors that could attempt
this, and (b) we will not check all special permissions for a request once we
know at least one has failed.

## Design Details

### Permission Description System

Now that we have described all requirements and given the overall outline, we
can finally state exactly how the mapping from field to permission works and how
it is specified. Additionally, we will describe the parameterization systems. We
will give a series of examples in {goal, built-in specification, CRD
specification, resulting verb} tuplets.

##### Everything in spec

On types:

```go
// +permission-verb:"specification"
type FooSpec struct {
```

Or on fields:

```go
  ...
  // +permission-verb:"specification"
  Spec FooSpec
  ...
```

On CRDs schemas:

```
"x-kubernetes-permission-verb": "specification"
```

The authz system will see the verb "granular:specfication".

This illustrates that the granular permissions can be made hirearchical;
granting "granular" and "granular:specification" to some actor permits it to
modify anything in .spec. API Server will check the permissions from most
general to most specific, so that a single general permission can let it avoid
checking many specific permissions.

#### Pod's nodeName

Given where this appears in Pod, this permission will only be checked for pods
if the agent doesn't have the "specification" permission.

This must be set on the field declaration, since nodeName is a string and not a
struct:

```go
  ...
  // +permission-verb:"nodeAssignment"
  NodeName string
  ...
```

On CRDs schemas (pod is not a CRD, but if a CRD has a field that is the same
concept, you could add to that field this):

```
"x-kubernetes-permission-verb": "nodeAssignment"
```

The authz system will see the verb "granular:nodeAssignment".

##### Everything in metadata

Configured the same as everything in spec, but the verb will be (CHOICE NEEDED)
"metadata" or "objectmeta".

#### All Labels

This permission will only be checked if the agent doesn't have the everything in
metadata permission.

This is set on the field declaration, since it is a map and we are granting
permission on any item in the map.

```go
  ...
  // +permission-verb:"labels"
  Labels map[string]string
  ...
```

The authz system will see the verb "granular:labels".

#### Specific Labels

This permission will only be checked if the agent has neither the general
metadata permission nor the "granular:labels" permission.

This field is currently in ObjectMeta, declared like this:

```go
	...
	// Map of string keys and values that can be used to organize and categorize
	// (scope and select) objects. May match selectors of replication controllers
	// and services.
	// More info: http://kubernetes.io/docs/user-guide/labels
	// +optional
	Labels map[string]string `json:"labels,omitempty" protobuf:"bytes,11,rep,name=labels"`
	...
```

This is very inconvenient as it leaves only one place to declare things in the
schema. Note that we must not make any incompatible changes to the serialization
of this type. This is one way to make this work:

```go
	...
	Labels map[LabelKey]LabelValue `json:"labels,omitempty" protobuf:"bytes,11,rep,name=labels"`
	...
}
.
.
.

// LabelKey is a key for a label; it is a string.
// +permission-parameter-source:"label-key"
type LabelKey string

// LabelValue is a value in a label; it is a string.
// +permission-verb:"label"
// +permission-parameterized-by:"label-key"
// +permission-parameter-treatment:slash-delimited-prefix
type LabelValue string
```

This says to the permission system that in order to change a LabelValue, there
must be something in the field path emitting a "label-key" parameter; that that
value needs to be massaged by a "slash-delimited-prefix" function; and that the
permission should then be checked with that parameter.

Supposing the key is "mycompany.example.com/FooLabel", The authz system will see
the verb "granular:label(mycompany.example.com)".

DECISION NEEDED: would it be better to use a CEL expression to process the
parameter?

#### Specific Finalizers

TODO (similar choice).

#### Specific Conditions

TODO (similar choice).

#### Complete List of parameter treatments

* None (omit the tag completely). The referenced field is used verbatim as the parameter.
* `slash-delimited-prefix`. The referenced field is split by '/' characters and
  the first segment is used as the parameter.
* (in progress)

#### Style Guide / Consistency Help

Rules for making up a permission:

* The permission name SHOULD be a noun.
* The permission name MUST be in lowerCamelCase.
* The permission name SHOULD refer to a conceptual attribute, not a field name.
  This helps the permission be reusable in other contexts with different
  enclosing parent fields.
* The permission MUST NOT reference ancestor fields ("nodeAssigment" not
  "pod.nodeAssignment")

We will update the documentation generation to list the permissions both with
the fields / types they guard, as well as in a standalone list for easy
browsing. This list will help API reviewers keep the permissions on built-in
resources coherent.

DECISION NEEDED:

Option A: we will put a registry file in the kubernetes repository so that 3rd
party extension authors can register their custom permissions and dedup/reuse
rather than create new ones, if possible.

Option B: we will require 3rd party authors to prefix their custom permissions
with the string "ext" (TODO: "mycompany.example.com" if there is likely to be
space).

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

Turning on this feature (via upgrade) should have no effect to an existing cluster.

Turning off this feature (via downgrade) will have no effect unless the feature
is in use (admin has configured some fine-grained permissions); in that case,
fine-grained permissions will stop working (users/groups assigned them will not
be able to use them). Fields marked exclusive would also stop being exclusive
(general permissions would be sufficient to write them). That's a problem, so we
will not add any such fields (and advise CRD authors not to) until this feature
has defaulted on for at least one release.

### Version Skew Strategy

There are no version skew issues, since the design makes permissions independent
from API version, and since the implementation is completely server-side.

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

No default behavior will be directly changed as a result of this KEP. Marking
fields exclusive would be a significant default behavior change and we won't do
it as part of this KEP (and perhaps never).

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

Scan the audit logs to see if any authz checks have been done for any verb with
the prefix "granular".

(Workloads don't use this feature directly, but if configured, they may be
getting permission to do what they want to do via this feature.)

###### How can someone using this feature know that it is working for their instance?

Users can send a SelfSubjectAccessReview request (or a
SubjectAccessReviewRequest if the query is about a service account they don't
have keys for) to see if they have a given granular permission. (Merely being
able to make a change isn't sufficient, because they might have the general
permission.)

But mostly, there is little reason for anyone not the cluster administrator to
want to do this.

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

Potentially; authz calls will increase in cases where the feature applies; heavy
usage of the feature could worst-case-plausibly 3-4x the number of authz calls
on mutating requests. Many clusters check RBAC first before calling the authz
plugin, in such clusters the cloud provider might not see much load. Authz
checks are already cached, so in practice we don't expect this to be too
significant.

The authz path is already intended to have heavy load.

###### Will enabling / using this feature result in increasing size or count of the existing API objects?

None by default. (If RBAC is used to configure this, then there will be a small
increase in Roles, RoleBindings, etc.)

###### Will enabling / using this feature result in increasing time taken by any operations covered by existing SLIs/SLOs?

Yes, mutating API requests will do extra work to see if additional permissions
need to be checked, and perform additional authz checks if needed.

###### Will enabling / using this feature result in non-negligible increase of resource usage (CPU, RAM, disk, IO, ...) in any components?

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

This feature won't make the resulting unavailability worse nor add new failure
modes.

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

This KEP is rather complex. Unfortunately simpler designs are all ruled out by
some criteria (see next section). The best alternative, if we don't want to
implement this, is to wait for CEL based admission to land and just not worry
about that not being enabled universally. (@deads2k will disagree with this.)

## Alternatives

### Subresources

Subresources have some use already, but adding additional ones to cover all the
desired fields was rejected for the following reasons:

* Each one is a lot of work (this KEP is a lot of work too, but it solves the
  whole problem at once).
* Subresources don't solve goal #2 in this KEP.
* Controllers have to be specially written to use a subresource.
  .In contrast, with this KEP, if a particular controller only ever wrote
  .metadata.labels, we could grant it a special permission on just that field and
  remove its existing overly-broad "write on everything" permission, all
  __without changing the controller__.
* Subresources don't compose; if you want to make changes to N fields you have
  to make N API requests.
* Subresources make the client experience much less clear, adding a set of
  Get|Apply|Update|Create|etc functions for every type for every general
  subresource. E.g. 5 new metadata-based subresources times 50 (?) built-in
  types times 6 verbs means 1500 functions that would be added to the generated
  clients.

### Automatically produce permissions from field names

The problem with this approach is around version changes. It is too hard for
system administrators to ensure that users have the same permissions no matter
which API version of an object they access.

Additionally describing the exact parameterization needed for each field still
requires manual attention.

OTOH, the chosen solution is likely to end up with some inconsistent permissions
between different CRs.

### Hard-code permissions instead of putting them in the schema

This is not significantly easier than the given solution, and leaves us with the
problem of how to configure the permissions, and especially how to sustainably
document them.

### Wait for CEL-based admission

This solution is likely 1.5 years or more away from landing. Additionally this
doesn't meet the requirement of being universally available (it might not be
enabled).

## Infrastructure Needed (Optional)

<!--
Use this section if you need things from the project/SIG. Examples include a
new subproject, repos requested, or GitHub details. Listing these here allows a
SIG to get the process for these resources started right away.
-->
