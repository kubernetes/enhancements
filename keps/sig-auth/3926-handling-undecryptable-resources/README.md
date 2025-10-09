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
# KEP-3926: Handling undecryptable resources

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
  - [User Stories (Optional)](#user-stories-optional)
    - [Story 1](#story-1)
    - [Story 2](#story-2)
  - [Notes/Constraints/Caveats (Optional)](#notesconstraintscaveats-optional)
  - [Risks and Mitigations](#risks-and-mitigations)
- [Design Details](#design-details)
  - [Background](#background)
  - [Proposed Solution](#proposed-solution)
    - [New Error Status for Read Failures](#new-error-status-for-read-failures)
    - [New Delete Option for Corrupt Objects](#new-delete-option-for-corrupt-objects)
    - [Admission Control for Unconditional Deletion](#admission-control-for-unconditional-deletion)
  - [Implementation Considerations](#implementation-considerations)
    - [Watch Event Propagation and Client Recovery](#watch-event-propagation-and-client-recovery)
    - [Design Principles](#design-principles)
    - [Alternative Approaches Considered](#alternative-approaches-considered)
  - [Test Plan](#test-plan)
      - [Prerequisite testing updates](#prerequisite-testing-updates)
      - [Unit tests](#unit-tests)
      - [Integration tests](#integration-tests)
      - [e2e tests](#e2e-tests)
  - [Graduation Criteria](#graduation-criteria)
    - [Alpha](#alpha)
    - [Beta](#beta)
    - [Beta](#beta)
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

- [x] (R) Enhancement issue in release milestone, which links to KEP dir in [kubernetes/enhancements] (not the initial KEP PR)
- [x] (R) KEP approvers have approved the KEP status as `implementable`
- [x] (R) Design details are appropriately documented
- [x] (R) Test plan is in place, giving consideration to SIG Architecture and SIG Testing input (including test refactors)
  - [ ] e2e Tests for all Beta API Operations (endpoints)
  - [ ] (R) Ensure GA e2e tests meet requirements for [Conformance Tests](https://github.com/kubernetes/community/blob/master/contributors/devel/sig-architecture/conformance-tests.md)
  - [ ] (R) Minimum Two Week Window for GA e2e tests to prove flake free
- [ ] (R) Graduation criteria is in place
  - [ ] (R) [all GA Endpoints](https://github.com/kubernetes/community/pull/1806) must be hit by [Conformance Tests](https://github.com/kubernetes/community/blob/master/contributors/devel/sig-architecture/conformance-tests.md)
- [x] (R) Production readiness review completed
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
Encryption at rest for API resources has been a stable part of Kubernetes for a long time.
Every now and then there had been cases where, be it by improper handling or external system
failures, the cluster encryption got into a broken state.

If a single object of a resource type cannot be decrypted, listing resources of that
type in a path prefix containing the object always fails, even if the rest of
the resource instances is accessible.

Currently, removing a resource that causes such failures is not possible.
A cluster administrator must access etcd directly and remove the malformed data manually.

This KEP proposes a way to identify resources that fail to decrypt or fail to be decoded
into an object, and introduces a new delete option to ignore any storage checks in case
such a read error occurs. This is done in order to be able to delete such a failing
resource by using just Kubernetes API.

## Motivation

<!--
This section is for explicitly listing the motivation, goals, and non-goals of
this KEP.  Describe why the change is important and the benefits to users. The
motivation section can optionally provide links to [experience reports] to
demonstrate the interest in a KEP within the wider Kubernetes community.

[experience reports]: https://github.com/golang/go/wiki/ExperienceReports
-->

### Goals

<!--
List the specific goals of the KEP. What is it trying to achieve? How will we
know that this has succeeded?
-->
- provide a way to identify persisted resources that failed decryption or that cannot be
  decoded
- provide an option to delete a resource independently of its contents, if those
  cannot be reached due to data transformation or data corruption

### Non-Goals

<!--
What is out of scope for this KEP? Listing non-goals helps to focus discussion
and make progress.
-->
- implementing system for ignoring different types of storage errors
- give clients control over skipping other steps of a delete request flow than decoding errors

## Proposal

<!--
This is where we get down to the specifics of what the proposal actually is.
This should have enough detail that reviewers can understand exactly what
you're proposing, but should not include things like API designs or
implementation. What is the desired outcome and how do we measure success?.
The "Design Details" section below is for the real
nitty-gritty.
-->

Improve resource retrieval errors to include more information about the object that
failed transformation while it was being retrieved from the storage.

Introduce a new `DeleteOption` that would allow deleting a resource even
if we cannot retrieve its data.

### User Stories (Optional)

<!--
Detail the things that people will be able to do if this KEP is implemented.
Include as much detail as possible so that people can understand the "how" of
the system. The goal here is to make this feel real for users without getting
bogged down.
-->

#### Story 1

I accidentally removed my encryption key but only a few resources were encrypted
with it. I know that these will either be recreated by a controller, or I can
manually recreate them. I would like a simple way to figure out which resources
fail decryption and I would like a way to remove them via Kubernetes API.

#### Story 2

I would like to remove a namespace I no longer need.  However, some of the resources
inside of the namespace were encrypted before the encryption at
rest configuration broke, which blocks a successful namespace delete.

### Notes/Constraints/Caveats (Optional)

<!--
What are the caveats to the proposal?
What are some important details that didn't come across above?
Go in to as much detail as necessary here.
This might be a good place to talk about core concepts and how they relate.
-->
An unconditional delete of a malformed resource may break garbage collection, would ignore finalizers and
would disregard any underlying system processes that might be tied to the given resource
(e.g. pods).

### Risks and Mitigations

<!--
What are the risks of this proposal, and how do we mitigate? Think broadly.
For example, consider both security and how this will impact the larger
Kubernetes ecosystem.

How will security be reviewed, and by whom?

How will UX be reviewed, and by whom?

Consider including folks who also work outside the SIG or subproject.
-->
We need to make sure that a user that is trying to perform an unconditional delete of
a malformed resource is well informed about the impact of what they are doing. This should be handled
by one or more prompts from the `kubectl` client when the `DeleteOption` from this enhancement is set.

Gate the deletion with an additional admission layer on server.

## Design Details

<!--
This section should contain enough information that the specifics of your
change are understandable. This may include API specs (though not always
required) or even code snippets. If there's any ambiguity about HOW your
proposal will be implemented, this is the place to discuss them.
-->
### Background

The encryption/decryption for encryption at rest is implemented via transformers
that get applied to a resource in code that handles resource read/write from etcd3
databases.

The storage handling does not change with KMSv2, a resource transformer is provided
in that case, too.

References:
1. [Encryption transformers creation](https://github.com/kubernetes/kubernetes/blob/8decaf3ae7f410ab3f3774f3895b9f3124b8a4c6/staging/src/k8s.io/apiserver/pkg/server/options/encryptionconfig/config.go#L185)
2. [Example of a resource retrieval with the transformer](https://github.com/kubernetes/kubernetes/blob/8decaf3ae7f410ab3f3774f3895b9f3124b8a4c6/staging/src/k8s.io/apiserver/pkg/storage/etcd3/store.go#L155-L157)

The code example 2. above shows that currently, when reading a resource fails, we
lose all the context about the resource and a non-wrapping, generic internal error
is returned.

### Proposed Solution

#### New Error Status for Read Failures

The current API errors don't appear to include an error status specific to storage. Therefore
a new status should be introduced - `StatusReasonStoreReadError`.

```go
  // StatusReasonStoreReadError means that the server encountered an error while
  // retrieving resources from the backend object store.
  // This may be due to backend database error, or because processing of the read
  // resource failed.
  // Details:
  //   "kind" string - the kind attribute of the resource being acted on.
  //   "name" string - the prefix where the reading error(s) occurred
  //   "causes" []StatusCause
  //      - (optional):
  //        - "type" CauseType - CauseTypeUnexpectedServerResponse
  //        - "message" string - the error message from the store backend
  //        - "field" string - the full path with the key of the resource that failed reading
  //
  // Status code 500
  StatusReasonStoreReadError StatusReason = "StorageReadError"
```

This error will also include full paths to the resources that cannot be read in
an unstructured, human-readable message.

In cases where the number of malformed resources would be too great (> 100), only
the first 100 will be shown in the `causes` slice. The 101st element of the slice
takes the following form:

```go
StatusCause{
  type: CauseTypeTooMany
  message: "too many errors, the list is truncated"
}
```


#### New Delete Option for Corrupt Objects

Deleting a resource is a rather complicated process:
1. a resource might represent an actual process running on a host (Pod)
2. there might be other resources with owner references to the resource that's being deleted
3. a resource might contain finalizers that safeguard the deletion of the given resource
   before other dependent resources are deleted (typically - namespaces and the `kubernetes` finalizer)

An unconditional deletion should try to do best effort on all of the above, but
in case of an undecryptable resource, all the above would be ignored.

For case 1., ignoring an underlying process may not be an issue as kubelet is supposed
to take care [of unused containers](https://kubernetes.io/docs/concepts/architecture/garbage-collection/#containers-images).

In case 2., there might be issues with setting related objects as `orphans`, which
could potentially cause an unwanted cascade deletion of objects.

3. has a potential of becoming rather serious. Finalizers are typically set to
safeguard other objects, and so if e.g. an aggregated API server is removed, its
API objects might be scattered around the etcd database without and API to remove them.

To allow unconditional deletion, a new `DeleteOption` should be introduced - `IgnoreStoreReadErrorWithClusterBreakingPotential`

```go
type DeleteOptions struct {
  ...
  // IgnoreStoreReadErrorWithClusterBreakingPotential will try to perform the normal
  // deletion flow but if the data of the resource being deleted cannot be read from
  // the store, either because it failed to be decrypted or the data is
  // otherwise corrupted and cannot be decoded, it will disregard these errors
  // and still perform the deletion.
  // WARNING: This will break the cluster if the resource has dependencies beyond
  //          the caller's comprehension. Use only if you REALLY know what you are
  //          doing.
  // WARNING: Vendors will most likely consider using this option to be breaking the
  //          support of their product.
  IgnoreStoreReadErrorWithClusterBreakingPotential bool
}
```

#### Admission Control for Unconditional Deletion

A "delete" verb on a resource is not usually considered a privileged action. As the previous
section explains, deletion of a resource might carry unexpected consequences. Unconditional
deletions should therefore have their own extra admission.

The unconditional deletion admission:
1. checks if a "delete" request contains the `IgnoreStoreReadErrorWithClusterBreakingPotential` option
2. if it does, it checks the RBAC of the request's user for the `delete-ignore-read-errors` verb of the given resource

### Implementation Considerations

#### Watch Event Propagation and Client Recovery

When a corrupt object is deleted from etcd, the kube-apiserver's watch cache
cannot transform or decode the object's previous value. This triggers a
deliberate recovery sequence:

1. **Error Detection**: The etcd3 watcher fails to transform/decode the deleted
   object's data and generates a `watch.Error` event with `StatusReasonStoreReadError`.

2. **Cacher Reset**: The Cacher's internal Reflector receives this error, causing
   `ListAndWatch()` to stop. After a brief delay, the Cacher reinitializes by
   calling `terminateAllWatchers()` followed by a fresh LIST from etcd.

3. **Client Disconnection**: All active watch connections for that resource type
   are terminated. Clients see their watch channels close without receiving the
   original error event.

4. **Client Recovery**: Disconnected clients attempt to resume watching from their
   last known `resourceVersion`. The server rejects this with a "too old resource
   version" error, forcing clients to perform a fresh LIST and rebuild their
   local caches.

#### Design Principles

The following principles, agreed upon by SIG API Machinery, guide this enhancement:

1. **Watch history cannot be preserved** when a corrupt object exists. Since the
   object's data cannot be decrypted or decoded, we have no access to the correct
   previous object state required for a semantically valid DELETE event.

2. **Performance degradation is acceptable** during the remediation window. The
   temporary increase in API server load from client re-lists is an accepted
   tradeoff for restoring cluster health.

3. **Enable admin remediation**: The admin must be able to identify corrupt
   objects and delete them, even if one by one. Once all corrupt objects are
   removed, the kube-apiserver and client informers recover automatically.

This approach favors eventual consistency and cluster recovery over preserving
individual watch streams during an inherently abnormal situation.

#### Alternative Approaches Considered

We considered using shallow object representations to enhance error or delete events,
enabling targeted removal of the corrupt object from client caches without triggering
a full re-list:

1. **`DeletedFinalStateUnknown`**: A client-go type used when the final state of
   a deleted object is unknown. This approach failed because `DeletedFinalStateUnknown`
   does not implement `runtime.Object`, which is required by the watch cache.

2. **`PartialObjectMetadata`**: A Kubernetes type containing only object metadata.
   This failed because the watch cache's `getAttrsFunc` performs type assertions
   to the specific resource type (e.g., `*api.Secret`), which `PartialObjectMetadata`
   cannot satisfy.

3. **Type Identity Object**: Creating an empty object of the correct type via
   `newFunc()` and copying only essential metadata (namespace, name, resourceVersion,
   UID). While technically feasible, the added complexity was not justified given
   the design principles outlined above.

### Test Plan

<!--
**Note:** *Not required until targeted at a release.*
The goal is to ensure that we don't accept enhancements with inadequate testing.

All code is expected to have adequate tests (eventually with coverage
expectations). Please adhere to the [Kubernetes testing guidelines][testing-guidelines]
when drafting this test plan.

[testing-guidelines]: https://git.k8s.io/community/contributors/devel/sig-testing/testing.md
-->

- [x] I/we understand the owners of the involved components may require updates to
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

- `k8s.io/apiserver/pkg/storage/etcd3`: `28.9.2023` - `77%`
- `k8s.io/apimachinery/pkg/apis/meta/v1`: `28.9.2023` - `48.1%`

##### Integration tests

<!--
This question should be filled when targeting a release.
For Alpha, describe what tests will be added to ensure proper quality of the enhancement.

For Beta and GA, add links to added tests together with links to k8s-triage for those tests:
https://storage.googleapis.com/k8s-triage/index.html
-->

Alpha: 
- [TestAllowUnsafeMalformedObjectDeletionFeature](https://github.com/kubernetes/kubernetes/blob/506e4fed14e38d3dd84ac043dfe66bbc16993fa7/test/integration/controlplane/transformation/secrets_transformation_test.go#L137): [testgrid](https://testgrid.k8s.io/sig-release-master-blocking#integration-master&include-filter-by-regex=AllowUnsafeMalformedObjectDeletion), [triage](https://storage.googleapis.com/k8s-triage/index.html?test=TestAllowUnsafeMalformedObjectDeletionFeature)
  - Verifies corrupt secrets can be deleted with feature enabled, the new option set and proper RBAC
  - Verifies that normal deletion deletion fails with new `StorageError: corrupt object`
  - Verifies that normal secrets can still be deleted with the feature enabled, even with corrupt objects in the database
  - Verifies deletion of corrupt objects is blocked when feature is disabled and there is a lack of option and RBAC.
- [TestListCorruptObjects](https://github.com/kubernetes/kubernetes/blob/506e4fed14e38d3dd84ac043dfe66bbc16993fa7/test/integration/controlplane/transformation/secrets_transformation_test.go#L426): [testgrid](https://testgrid.k8s.io/sig-release-master-blocking#integration-master&include-filter-by-regex=AllowUnsafeMalformedObjectDeletion), [triage](https://storage.googleapis.com/k8s-triage/index.html?test=TestListCorruptObjects)
  - Verifies LIST returns errors for corrupt objects when feature is enabled
  - Verifies error truncation when too many corrupt objects exist

Beta:
- test that LIST operation is capable of returning multiple corrupt objects
- test delete handler with unsafe deletion flow
- test deletion of bit-flip corrupted objects (deserialization failure, not transformer failure)
- test deletion of corrupt CRs
- validate kube-apiserver transition to healthy state after cleanup

##### e2e tests

<!--
This question should be filled when targeting a release.
For Alpha, describe what tests will be added to ensure proper quality of the enhancement.

For Beta and GA, add links to added tests together with links to k8s-triage for those tests:
https://storage.googleapis.com/k8s-triage/index.html

We expect no non-infra related flakes in the last month as a GA graduation criteria.
-->

Integration tests are functionally equivalent to e2e tests for this feature.
They exercise the full kube-apiserver stack with a real etcd backend. The
integration test framework is preferred because it allows direct manipulation of
etcd contents, encryption configuration during test execution and they are more
stable to handle such manipulation.

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

- Error type is implemented
- Deletion of malformed etcd objects and its admission can be enabled via a feature flag

#### Beta

- Feature enabled by default
- Dry-run support for unsafe corrupt object deletion
- Comprehensive test coverage as outlined in the [Integration tests > Beta](#integration-tests) section.

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

This feature is contained entirely within kube-apiserver with no persistent state changes:

- **Upgrade:** Enabling the feature gate makes the `IgnoreStoreReadErrorWithClusterBreakingPotential` delete option functional. No configuration migration required.
- **Downgrade:** Disabling the feature gate makes the delete option non-functional. The option is silently ignored. No cleanup required.
- **Mixed version clusters:** During rolling updates, some apiservers may have the feature enabled while others don't. Requests with the unsafe delete option will only succeed on apiservers with the feature enabled. This is acceptable for an emergency recovery feature.

No special upgrade or downgrade procedures are required.

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

This feature is entirely within kube-apiserver with no node component interaction:

- **API server to API server:** In HA setups, some apiservers may have the feature enabled while others don't during rollout. The unsafe delete option only works on apiservers with the feature enabled. This is acceptable behavior.
- **Kubelet:** No interaction. This feature doesn't affect pod lifecycle or node operations.
- **Other components:** No interaction. The feature only affects DELETE requests with the specific option set.

No version skew concerns exist because:
1. The feature doesn't introduce new API fields that need coordination
2. The DeleteOption is ignored by apiservers without the feature
3. No persistent state changes that could cause inconsistency

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
  - Feature gate name: AllowUnsafeMalformedObjectDeletion
  - Components depending on the feature gate: kube-apiserver
- [x] Other
  - Describe the mechanism: The new error type will always be present once implemented
  - Will enabling / disabling the feature require downtime of the control
    plane? **No**
  - Will enabling / disabling the feature require downtime or reprovisioning
    of a node? **No**

###### Does enabling the feature change any default behavior?

<!--
Any change of default behavior may be surprising to users or break existing
automations, so be extremely careful here.
-->
No.

###### Can the feature be disabled once it has been enabled (i.e. can we roll back the enablement)?

<!--
Describe the consequences on existing workloads (e.g., if this is a runtime
feature, can it break the existing applications?).

Feature gates are typically disabled by setting the flag to `false` and
restarting the component. No other changes should be necessary to disable the
feature.

NOTE: Also set `disable-supported` to `true` or `false` in `kep.yaml`.
-->
The feature can be safely enabled and disabled at will.

###### What happens if we reenable the feature if it was previously rolled back?

There should be no side-effects.

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
Yes, the integration tests explicitly toggle the feature gate to verify enablement/disablement:

- [TestAllowUnsafeMalformedObjectDeletionFeature](https://github.com/kubernetes/kubernetes/blob/master/test/integration/controlplane/transformation/secrets_transformation_test.go#L137) - [feature gate toggle at L198](https://github.com/kubernetes/kubernetes/blob/master/test/integration/controlplane/transformation/secrets_transformation_test.go#L198): Parametrized test running with `featureEnabled: true` and `featureEnabled: false`. Verifies deletion is blocked when disabled, works when enabled with proper RBAC.
- [TestListCorruptObjects](https://github.com/kubernetes/kubernetes/blob/master/test/integration/controlplane/transformation/secrets_transformation_test.go#L426) - [feature gate toggle at L512](https://github.com/kubernetes/kubernetes/blob/master/test/integration/controlplane/transformation/secrets_transformation_test.go#L512): Parametrized test verifying LIST returns `StatusReasonStoreReadError` when enabled, `StatusReasonInternalError` when disabled.

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
Rollout and rollback cannot fail because:

1. **No persistent state changes:** The feature doesn't write new data to etcd or modify existing objects (except deleting them when explicitly requested).
2. **Contained within kube-apiserver:** No coordination with kubelet, controllers, or other components required.
3. **Opt-in behavior:** The feature only activates when a client explicitly sets the `IgnoreStoreReadErrorWithClusterBreakingPotential` option AND has RBAC permission for the `unsafe-delete-ignore-read-errors` verb.

Impact on running workloads: None. The feature doesn't affect normal cluster operations.

###### What specific metrics should inform a rollback?

<!--
What signals should users be paying attention to when the feature is young
that might indicate a serious problem?
-->

**Important context:** This feature is for emergency cluster recovery. During remediation,
temporary performance degradation is expected and acceptable. The following metrics will
spike when corrupt objects are deleted - this is the feature working correctly, not a problem.

Rollback should only be considered if:

1. **Unexpected cache resets** — `apiserver_watch_cache_initializations_total` spikes occur
   when no corrupt object deletion was performed. This would indicate the feature gate
   enablement itself is causing unintended side effects.

2. **Recovery does not complete** — After corrupt object deletion, the system should stabilize
   within minutes. If `apiserver_storage_list_total` remains elevated for an extended period
   (>10 minutes for typical clusters), clients may be stuck in reconnection loops.

###### Were upgrade and rollback tested? Was the upgrade->downgrade->upgrade path tested?

<!--
Describe manual testing that was done and the outcomes.
Longer term, we may want to require automated upgrade/rollback tests, but we
are missing a bunch of machinery and tooling and can't do that now.
-->
No testing of upgrade->downgrade->upgrade necessary because:

1. **No new persisted state changes**: Either the corrupt object is deleted or not.
2. **"Atomic" behavior**: Either the feature is enabled and the user can perform unsafe deletes (with proper RBAC), or it's disabled and they cannot.
3. **Version skew is handled gracefully**: The interpretation of a deletion event of a corrupt object is added to k8s 1.32.
4. **Rollback is trivial**: Disabling the feature gate simply makes the `DeleteOption` non-functional. No cleanup or migration required.

###### Is the rollout accompanied by any deprecations and/or removals of features, APIs, fields of API types, flags, etc.?

<!--
Even if applying deprecation policies, they may still surprise some users.
-->
No deprecations.

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

This feature is for cluster administrators performing emergency recovery, not for workload automation.

To detect actual usage (i.e., unsafe deletions being performed):

- Audit logs: Search for annotation: `apiserver.k8s.io/unsafe-delete-ignore-read-error`.
- RBAC: Check RoleBindings/ClusterRoleBindings granting `unsafe-delete-ignore-read-errors` verb.

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
- Details:
    1. Attempt to delete a corrupt object with the delete option set but without
        RBAC permission for `unsafe-delete-ignore-read-errors` verb. Receiving
        403 Forbidden (instead of 500 StorageReadError) confirms the feature is
        enabled and recognizing the option.
    2. Without the delete option, attempting to delete a corrupt object returns
        the original 500 StorageReadError.
    3. Use dry-run to safely verify the behavior with various combinations of
        option and RBAC permissions.
    4. With proper RBAC permission and the option set, the corrupt object
        deletion succeeds.

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

This feature targets emergency cluster recovery scenarios where corrupt objects
are blocking normal operations. Temporary performance degradation during
remediation is acceptable - the priority is restoring cluster functionality.

The deletion itself is faster as it bypasses preconditions and finalizers,
but there are cache resets at the kube-apiserver and its watching clients
(informers) that may cause performance degradation.

###### What are the SLIs (Service Level Indicators) an operator can use to determine the health of the service?

<!--
Pick one more of these and delete the rest.
-->

- [x] Metrics

  **Note:** During corrupt object deletion remediation, temporary metric spikes are expected
  and acceptable. The priority is restoring cluster functionality, not maintaining SLOs.

  - Metric name: `apiserver_watch_cache_initializations_total`
    - Labels: `group`, `resource`
    - Components exposing the metric: kube-apiserver
    - Details: Increments when watch cache rebuilds. A spike correlating with corrupt
      object deletion confirms the expected recovery flow triggered. After remediation
      completes, this should return to baseline (typically zero or very low).
  
  - Metric name: `apiserver_storage_list_total`
    - Labels: `group`, `resource`
    - Components exposing the metric: kube-apiserver
    - Details: Tracks LIST operations hitting etcd storage. Expect a transient spike
      as clients reconnect and rebuild caches. Recovery is complete when this returns
      to pre-remediation levels.

- [ ] Other (treat as last resort)
    - Details:

###### Are there any missing metrics that would be useful to have to improve observability of this feature?

<!--
Describe the metrics themselves and the reasons why they weren't added (e.g., cost,
implementation difficulties, etc.).
-->

The existing metrics provide sufficient observability for tracking cache rebuilds and recovery:

- `apiserver_watch_cache_initializations_total` — confirms cache rebuild occurred
- `apiserver_storage_list_total` — tracks recovery progress (client re-lists)

**Known gap:** `apiserver_storage_decode_errors_total` only covers decode errors in `store.go`
operations (GET, LIST, etc.), not in `watcher.go` transform/decode failures. This means the
metric won't increment specifically for the corrupt object deletion watch flow. This is
acceptable because:

1. The feature is for emergency recovery where detailed decode error counts are less
   critical than successful deletion.
2. The cache rebuild metrics above provide sufficient signal that the flow completed.
3. Adding watcher-specific decode error metrics would require broader consensus in
   sig-instrumentation.

For tracking actual feature usage (unsafe deletions performed), operators should use audit logs
and search for the `apiserver.k8s.io/unsafe-delete-ignore-read-error` annotation.

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
No

### Scalability

<!--
For alpha, this section is encouraged: reviewers should consider these questions
and attempt to answer them.

For beta, this section is required: reviewers must answer these questions.

For GA, this section is required: approvers should be able to confirm the
previous answers based on experience in the field.
-->
The feature itself should not bring any concerns in terms of performance at scale.
In particular as its usage is supposed to run on potentially broken clusters.

An issue in terms of scaling comes with the error that attempts to list all
resources that appeared to be malformed while reading from the storage. A limit
of 100 presented resources was arbitrarily picked to prevent huge HTTP responses.

Another issue in terms of scaling happens when the corrupt objects are deleted.
Client reflectors re-list to recover, this causes temporarily increased load on
the client-side and the kube-apiserver.

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
No.

###### Will enabling / using this feature result in increasing size or count of the existing API objects?

<!--
Describe them, providing:
  - API type(s):
  - Estimated increase in size: (e.g., new annotation of size 32B)
  - Estimated amount of new objects: (e.g., new Object X for every existing Pod)
-->
DeleteOptions gets a new boolean field, but it is transient: no persistence in
etcd.

###### Will enabling / using this feature result in increasing time taken by any operations covered by existing SLIs/SLOs?

<!--
Look at the [existing SLIs/SLOs].

Think about adding additional work or introducing new steps in between
(e.g. need to do X to start a container), etc. Please describe the details.

[existing SLIs/SLOs]: https://git.k8s.io/community/sig-scalability/slos/slos.md#kubernetes-slisslos
-->

DELETE operations:

- Unsafe DELETE path is faster (skips preconditions, validation, finalizers)
- Decreases latency for the unsafe delete itself

LIST operations:

- Client-side reflectors re-list when their watch breaks (after corrupt object deletion ERROR event)
- Temporarily increases LIST request volume to apiserver
- Latency increase depends on: number of watching clients × object count × apiserver resources

Expected impact:

- Negligible under the circumstance that the cluster is in a potentially broken
  state.
- Potentially noticeable if: popular resource (many watchers) × many objects × resource-constrained apiserver

###### Will enabling / using this feature result in non-negligible increase of resource usage (CPU, RAM, disk, IO, ...) in any components?

<!--
Things to keep in mind include: additional in-memory state, additional
non-trivial computations, excessive access to disks (including increased log
volume), significant amount of data sent and/or received over network, etc.
This through this both in small and large cases, again with respect to the
[supported limits].

[supported limits]: https://git.k8s.io/community//sig-scalability/configs-and-limits/thresholds.md
-->

Temporary increase during cleanup, dependent on object and resource type
popularity:

- apiserver: CPU / network during re-lists
- client-side: CPU / memory / network during re-lists / rebuilding cache

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

If the API server is unavailable, no DELETE requests can be processed (including unsafe deletes). This is standard Kubernetes behavior.

If etcd is unavailable, DELETE requests fail with storage errors, including the unsafe delete feature.

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

1. **Missing RBAC permission**
   - Detection: 403 Forbidden responses when using the unsafe delete option
   - Mitigation: Grant `unsafe-delete-ignore-read-errors` verb permission to the user
   - Diagnostics: Audit logs show RBAC denial; API server logs show "forbidden" at verbosity 3+
   - Testing: Covered by TestAllowUnsafeMalformedObjectDeletionFeature

2. **Feature gate disabled**
   - Detection: Unsafe delete option silently ignored; corrupt object still returns 500 StorageReadError
   - Mitigation: Enable AllowUnsafeMalformedObjectDeletion feature gate
   - Diagnostics: Check feature gate status via /healthz or metrics
   - Testing: Covered by TestAllowUnsafeMalformedObjectDeletionFeature

3. **Object not actually corrupt**
   - Detection: Normal delete succeeds without needing the option
   - Mitigation: None needed - use normal delete
   - Diagnostics: Object is readable via GET
   - Testing: Covered by integration tests

###### What steps should be taken if SLOs are not being met to determine the problem?

During corrupt object deletion, temporary SLO degradation is expected (see Monitoring Requirements section). If degradation persists:

1. **Check apiserver_watch_cache_initializations_total** - should return to baseline within minutes
2. **Check apiserver_storage_list_total** - elevated counts indicate clients are still rebuilding caches
3. **Review audit logs** - confirm the unsafe delete completed successfully
4. **If recovery doesn't complete** - restart kube-apiserver to force fresh state

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

- 2023-03-27: KEP created
- 2023-10-05: KEP merged as provisional
- v1.32: Alpha implementation:
  - Deletion of corrupt objects, with client option and RBAC.
  - Extended listing of corrupt objects
  - Integration tests
- v1.36: Targeting beta
  - Cache reset deemed acceptable in sig-api-machinery bi-weekly meeting
  - Dry-Run
  - Additional integration tests for CRs and serialization failures.

## Drawbacks

<!--
Why should this KEP _not_ be implemented?
-->

1. **Potential for misuse:** The unsafe delete option bypasses safety mechanisms (finalizers, garbage collection). Misuse could orphan resources or break cluster state.

2. **Vendor support concerns:** Using this feature may void support from Kubernetes distributions/vendors, as it allows bypassing normal API guarantees.

3. **No undo:** Unsafe deletion is permanent. If used incorrectly, the only recovery is restoring from etcd backup.

These drawbacks are intentional - the feature is designed for emergency recovery where the alternative (direct etcd manipulation) is worse.

## Alternatives

<!--
What other approaches did you consider, and why did you rule them out? These do
not need to be as detailed as the proposal, but should include enough
information to express the idea and why it was not acceptable.
-->

**Direct etcd manipulation (status quo)**

Requires etcd access, bypasses all Kubernetes abstractions, risky, not audited.

## Infrastructure Needed (Optional)

<!--
Use this section if you need things from the project/SIG. Examples include a
new subproject, repos requested, or GitHub details. Listing these here allows a
SIG to get the process for these resources started right away.
-->
