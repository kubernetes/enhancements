---
title: Image garbage collection whitelist
authors:
  - "@libesz"
  - "@CsatariGergely"
  - "@BenTheElder"
owning-sig: sig-node
participating-sigs:
- sig-node
- sig-testing
reviewers:
  - "@zhangmingld"
  - "@Monkeyanator"
  - "@sjenning"
  - "@Random-Liu"
  - "@timothysc"
approvers:
  - sig-node-leads
editor: "@BenTheElder"
creation-date: 2019-04-26
last-updated: 2020-04-27
status: provisional
see-also:
replaces:
superseded-by:
---

# image-garbage-collector-whitelist

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
  - [Implementation Details/Notes/Constraints](#implementation-detailsnotesconstraints)
    - [Details of Story 1](#details-of-story-1)
    - [Details of Story 2](#details-of-story-2)
  - [Risks and Mitigations](#risks-and-mitigations)
    - [Risks of Story 1](#risks-of-story-1)
    - [Risks of Story 2](#risks-of-story-2)
- [Design Details](#design-details)
  - [Test Plan](#test-plan)
  - [Graduation Criteria](#graduation-criteria)
    - [Examples](#examples)
      - [Alpha -&gt; Beta Graduation](#alpha---beta-graduation)
      - [Beta -&gt; GA Graduation](#beta---ga-graduation)
      - [Removing a deprecated flag](#removing-a-deprecated-flag)
  - [Upgrade / Downgrade Strategy](#upgrade--downgrade-strategy)
  - [Version Skew Strategy](#version-skew-strategy)
- [Implementation History](#implementation-history)
- [Drawbacks [optional]](#drawbacks-optional)
- [Alternatives [optional]](#alternatives-optional)
- [Infrastructure Needed [optional]](#infrastructure-needed-optional)
<!-- /toc -->


## Release Signoff Checklist

- [ ] kubernetes/enhancements issue in release milestone, which links to KEP (this should be a link to the KEP location in kubernetes/enhancements, not the initial KEP PR)
- [ ] KEP approvers have set the KEP status to `implementable`
- [ ] Design details are appropriately documented
- [ ] Test plan is in place, giving consideration to SIG Architecture and SIG Testing input
- [ ] Graduation criteria is in place
- [ ] "Implementation History" section is up-to-date for milestone
- [ ] User-facing documentation has been created in [kubernetes/website], for publication to [kubernetes.io]
- [ ] Supporting documentation e.g., additional design documents, links to mailing list discussions/SIG meetings, relevant PRs/issues, release notes

## Summary

Enhancing Kubelet image garbage collector with a whitelist of container images. 
Whitelisted images will be ignored during garbage collection.
Currently only the dockershim pod sandbox image is whitelisted, this KEP aims
to make the whitelist configurable.

## Motivation

There are some use cases when not all the container images are available from a remote repository, therefore container images are stored locally in the image cache of the Nodes. These container images can be preloaded to the Nodes instead of pulled on demand.
The image garbage collector in Kubelet deletes all unused images from the image cache in case of storage shortage.
When a Pod using such preloaded image scheduled to a Node from where the container image was garbage collected the Pod will go to ImagePullBackOff state forever.

Users of container runtimes other than dockershim may wish to whitelist the
podsandbox image used by their container runtime, which may not be in sync
with dockershim.

### Goals

- Add a Kubelet command line option to list whitelisted container images
- Skip the garbage collection of the whitelisted container images

### Non-Goals

- Further complicating the kubelet image GC algorithm.
  - Kubelet already has this logic for the dockershim pause image, we just need
  to apply it to a configurable list. We do not aim to add any further complexity.

## Proposal

### User Stories

The user stories in the following chapters are alternatives of each other.

#### Story 1

Optimized for small amount of images

I as a Kubernetes administrator would like to define a set of exact image URLs to exclude these images from image garbage collection.

These images may be core system images that would be counter-productive to GC.

#### Story 2

Optimized for big amount of images

I as a Kubernetes administrator have a lots of frequently changing container images and I would like to generally exclude some of them from image garbage collection.

### Implementation Details/Notes/Constraints

What are the caveats to the implementation?
What are some important details that didn't come across above.
Go in to as much detail as necessary here.
This might be a good place to talk about core concepts and how they releate.

#### Details of Story 1

Optimized for small amount of images

It can become difficult to maintain the list of whitelisted images without cluster maintenance in case of changing container images.

This is already implemented in [pr 68549](https://github.com/kubernetes/kubernetes/pull/68549).

#### Details of Story 2

Optimized for big amount of images

There are two alternatives for implementation:
* Define only the hostname part of the image URL in the command line parameter. So that all images tagged to the same (fake) remote repository will be whitelisted.
* Define the whitelist with a regular expression. In this case all images matching the regexp will be whitelisted. A function to verify which images are matching would be useful.

### Risks and Mitigations

#### Risks of Story 1

Optimized for small amount of images

N/A

#### Risks of Story 2

Optimized for big amount of images

It is difficult to verify regular expressions.

## Design Details

### Test Plan

N/A

**Note:** *Section not required until targeted at a release.*

Consider the following in developing a test plan for this enhancement:
- Will there be e2e and integration tests, in addition to unit tests?
- How will it be tested in isolation vs with other components?

No need to outline all of the test cases, just the general strategy.
Anything that would count as tricky in the implementation and anything particularly challenging to test should be called out.

All code is expected to have adequate tests (eventually with coverage expectations).
Please adhere to the [Kubernetes testing guidelines][testing-guidelines] when drafting this test plan.

[testing-guidelines]: https://git.k8s.io/community/contributors/devel/sig-testing/testing.md

### Graduation Criteria

N/A

**Note:** *Section not required until targeted at a release.*

Define graduation milestones.

These may be defined in terms of API maturity, or as something else. Initial KEP should keep
this high-level with a focus on what signals will be looked at to determine graduation.

Consider the following in developing the graduation criteria for this enhancement:
- [Maturity levels (`alpha`, `beta`, `stable`)][maturity-levels]
- [Deprecation policy][deprecation-policy]

Clearly define what graduation means by either linking to the [API doc definition](https://kubernetes.io/docs/concepts/overview/kubernetes-api/#api-versioning),
or by redefining what graduation means.

In general, we try to use the same stages (alpha, beta, GA), regardless how the functionality is accessed.

[maturity-levels]: https://git.k8s.io/community/contributors/devel/sig-architecture/api_changes.md#alpha-beta-and-stable-versions
[deprecation-policy]: https://kubernetes.io/docs/reference/using-api/deprecation-policy/

#### Examples

N/A

These are generalized examples to consider, in addition to the aforementioned [maturity levels][maturity-levels].

##### Alpha -> Beta Graduation

- Gather feedback from developers and surveys
- Complete features A, B, C
- Tests are in Testgrid and linked in KEP

##### Beta -> GA Graduation

- N examples of real world usage
- N installs
- More rigorous forms of testing e.g., downgrade tests and scalability tests
- Allowing time for feedback

**Note:** Generally we also wait at least 2 releases between beta and GA/stable, since there's no opportunity for user feedback, or even bug reports, in back-to-back releases.

##### Removing a deprecated flag

- Announce deprecation and support policy of the existing flag
- Two versions passed since introducing the functionality which deprecates the flag (to address version skew)
- Address feedback on usage/changed behavior, provided on GitHub issues
- Deprecate the flag

**For non-optional features moving to GA, the graduation criteria must include [conformance tests].**

[conformance tests]: https://github.com/kubernetes/community/blob/master/contributors/devel/conformance-tests.md

### Upgrade / Downgrade Strategy

If applicable, how will the component be upgraded and downgraded? Make sure this is in the test plan.

Consider the following in developing an upgrade/downgrade strategy for this enhancement:
- What changes (in invocations, configurations, API use, etc.) is an existing cluster required to make on upgrade in order to keep previous behavior?
- What changes (in invocations, configurations, API use, etc.) is an existing cluster required to make on upgrade in order to make use of the enhancement?

### Version Skew Strategy

This is a per instance kubelet option, there should not be any version skew.

## Implementation History

- [original KEP PR](https://github.com/kubernetes/enhancements/pull/1007)

## Drawbacks [optional]

This does require introducing a new kubelet flag / config field.

## Alternatives [optional]

We could instead handle this in each container runtime using the runtime's own
configuration. This is suboptimal because it fragments solving this problem
while kubelet already has the machinery for this.

It could however be accomplished without any Kubernetes changes.

## Infrastructure Needed [optional]

We should be able to test this using existing CI infrastructure.
We may want some test for this.

The [kind](https://github.com/kubernetes-sigs/kind) project from SIG Testing
would be a first-party consumer of this functionality and could be used to verify
it.

[kubernetes.io]: https://kubernetes.io/
[kubernetes/enhancements]: https://github.com/kubernetes/enhancements/issues
[kubernetes/kubernetes]: https://github.com/kubernetes/kubernetes
[kubernetes/website]: https://github.com/kubernetes/website
