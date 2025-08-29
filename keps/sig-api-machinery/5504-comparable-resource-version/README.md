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
# KEP-5504: Comparable Resource Versions

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
    - [Story 3](#story-3)
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

Resource version is currently defined as an opaque string from the view of a client, with the only operation that is supported being equality comparisons. This differs from the internal apiserver implementation, where it is clearly defined as a monotonically incrementing integer. There are increasing requirements being required from clients consuming object metadata, where stronger comparisons than just equality are required.

We propose to extend some of the guarantees that the apiserver uses to the client as well, particularly the ability to consume the resource version as an integer, and the ability to compare resource versions to each other for more than equality. Clients can use the new semantics in order to view the "age" of a resource compared to another.

## Motivation

The motivation for this feature comes from the need for certain features to compare resources of the same kind to each other. One strong use case is [storage version migration](https://github.com/kubernetes/enhancements/issues/4192), where in order to tell whether a resource is fully migrated we check the resource version of the last migrated resource until we are sure all resources prior to migration have been migrated. This requires us to do comparisons besides equality on the resource. Since this requires us to "know" what place we are in time to perform the operation it would not be possible with pure equality comparisons. If we had to otherwise, we would likely have to iterate over the entire list of resources, leading to unbounded operation time, especially with objects that are frequently modified.

Another important reason for this is to improve client side informers. Much of the improvements inside of the apiserver internals has been based upon the ability for the apiserver to compare resource version. It is trivial for the apiserver to check cache freshness and order watch and list operations, none of which can currently be done client side, leaving many performance improvements locked to the internal apiserver implementation. With these improvements, libraries that use client-go can take advantage of the ordering properties and implement similar performance improvements so that external controllers can perform just as well as internal apiserver code. See [kubernetes/kubernetes#127693](https://github.com/kubernetes/kubernetes/issues/127693), and (kubernetes/kubernetes#130767)[https://github.com/kubernetes/kubernetes/issues/130767] for motivation here. There is currently no good way for a client to be able to tell whether or not it is far behind the apiserver due to the fact we only have equality comparisons. Implementing something similar to [KEP-2340](https://github.com/kubernetes/enhancements/blob/master/keps/sig-api-machinery/2340-Consistent-reads-from-cache/README.md#monitoring) in client would give us large performance improvements in this aspect.

Lastly, an example of needing to compare resources is a controller that controls deployments. By being able to check the resource version before and after deployment updates, it has the ability to see whether or not a new pod has been created yet. While equality checks are used for that currently, this opens up the floor for subtle correctness bugs and issues when churn is happening in the cluster. Current controllers work most of the time but can run into issues where staleness of the cache and other issues can affect what the controller sees as the current state. By having comparison based RV handling, controllers can directly check whether the objects are actually newer than before. 

### Goals

The goals for this KEP are fairly straightforward, firstly we will expose a utility function that clients can use on the resource version to check comparisons between resource versions. This will take the opaque resource version string and return a boolean and an error if it occurs. Along with that we will update the documentation to specify that a ResourceVersion must be a monotonically increasing integer. 

Second, we will create conformance tests to ensure that a conformant cluster abides by the new constraints to the resource version. This should be essentially every cluster in production but will give us the guarantee that users will be unaffected by the new constraints.

### Non-Goals

Non goals for the KEP are constraining size of the resource version from a client perspective, or adding opinionated ideas of the structure of a resource version besides comparability from a client perspective.

## Proposal

We will expose a helper function in client-go that will compare resource versions. It will have the type definition as follows

```
func CompareResourceVersion(x, y string) int error
```

We expose an error in case the strings are non comparable and return either (-1, 0, 1), following the cmp definition where we return -1 if x is less than y, 0 if they are equal, and 1 if x is greater than y. 

The internal implementation of this is important to ensure compatibility and efficiency:
  - Loop through the string to ensure all characters are 0-9 with no leading 0's
  - Compare length of strings, if they are not equal then the one with larger length is greater
  - If they are equal, perform a lexical comparison per character left to right until we hit a different character, at that point comparison between the characters will give us the larger integer

By doing it via lexical comparison we do not impose a restriction on resource version sizing and is more efficient than [bigint libraries](https://github.com/liggitt/kubernetes/commits/compare-rv-benchmark/) 

Along with this, we will also add a conformance test to kubernetes for all builtin types to ensure they are conformant. On this end we will enforce a 128 bit integer limitation on resource version 

### User Stories (Optional)

<!--
Detail the things that people will be able to do if this KEP is implemented.
Include as much detail as possible so that people can understand the "how" of
the system. The goal here is to make this feel real for users without getting
bogged down.
-->

#### Story 1

I am a controller creator and want to see if my actions have been done to all objects created before a certain object. I use the helper function along with a streamed list to do my work until I hit a greater than or equal object and stop work. Without this I would have to run my work on all objects.

#### Story 2

I am a controller creator and want to see whether my cache has caught up to my actions done on my previous reconcile. I use the helper function to compare the resource version and use it to more efficiently reconcile my objects.

#### Story 3

I am the creator of an extended apiserver, I can decide whether to use a numerical increasing integer as my resource version or not. If I decide to use a numerical resource version, either by using the k8s.io/apiserver or my own, I get the ability for clients to compare resource versions, improving performance and feature richness of my api objects from the client side. If I decide to use a different resource version encoding, clients will have to fall back and not be able to use certain features like storage version migration and possibly have less performant operations with more api calls. 

### Notes/Constraints/Caveats (Optional)

<!--
What are the caveats to the proposal?
What are some important details that didn't come across above?
Go in to as much detail as necessary here.
This might be a good place to talk about core concepts and how they relate.
-->

### Risks and Mitigations

There are certain API objects that may not conform to this specification. Controllers must gracefully handle these objects, which is why we provide an error type to the function signature. These cases will only occur on non conformant clusters and certain aggregated apiservers however, since we will be adding conformance tests to ensure compatibility. 

Some examples of api objects which aren't versioned on a monotonically increasing int are

* (metrics-server)[https://github.com/kubernetes-sigs/prometheus-adapter/blob/c2ae4cdaf160363151f746e253789af89f8b6c49/pkg/resourceprovider/provider.go#L244-L254]
* (calico)[https://github.com/projectcalico/calico/blob/09c0b753c91474e72157818a480165028f620999/libcalico-go/lib/backend/k8s/resources/profile.go#L138]
* (Porch)[https://github.com/nephio-project/porch/blob/4c066b6986533445fb15143507e7ce6470b66c72/pkg/cache/dbcache/dbpackage.go#L97C22-L97C31]

All of these however are non numerical in nature or do not support complex operations which may require listing. Our helper function will just error on these objects and will not support certain features that having a versioned object would have.

## Design Details

<!--
This section should contain enough information that the specifics of your
change are understandable. This may include API specs (though not always
required) or even code snippets. If there's any ambiguity about HOW your
proposal will be implemented, this is the place to discuss them.
-->

### Test Plan

We will add unit tests for the actual helper function. However the more involved piece of work will be adding conformance tests for every built in api type to ensure that the cluster is conforming with the new requirements. This will ensure that a client working on a conformant cluster will not encounter errors.

[x] I/we understand the owners of the involved components may require updates to
existing tests to make this code solid enough prior to committing the changes necessary
to implement this enhancement.

##### Prerequisite testing updates

<!--
Based on reviewers feedback describe what additional tests need to be added prior
implementing this enhancement to ensure the enhancements have also solid foundations.
-->

##### Unit tests

We will add unit tests for the helper function added above.

##### Integration tests

N/A

##### e2e tests

N/A

### Graduation Criteria

This is not a normal feature, rather an additional helper function. There will not be a graduation process function will be able to be used by clients once added to the client-go library. There will not be externally visible changes.

### Upgrade / Downgrade Strategy

N/A

### Version Skew Strategy

N/A

## Production Readiness Review Questionnaire

N/A as this is not a feature being added to K8s, rather a helper function with conformance tests.

## Implementation History

  * 2025-08-28 - KEP sent for review
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

This constrains the possible values of a resource version to a comparable integer, however this is already used in the apiserver code. This just extends the same ability to the client code with what the apiserver and other internal binaries use. 

## Alternatives

Some alternatives may include the use of something akin to Rust traits. We can mark any comparable api object with the comparable trait and use it in order to tell whether it can use the comparability functions. However this adds an opt in approach to comparability when currently effectively every api object already adheres to this constrain besides certain aggregated api objects.
