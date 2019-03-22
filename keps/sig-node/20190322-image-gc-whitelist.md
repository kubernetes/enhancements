---
title: Image garbage collection whitelist
authors:
  - "@libesz"
  - "@CsatariGergely"
owning-sig: sig-node
participating-sigs:
reviewers:
  - "@zhangmingld"
  - "@Monkeyanator"
  - "@sjenning"
  - "@Random-Liu"
  - "@timothysc"
approvers:
  - sig-node-leads
editor: "@CsatariGergely"
creation-date: 2019-04-26
last-updated: 2019-04-26
status: implemented
see-also:
replaces:
superseded-by:
---

# image-garbage-collector-whitelist

To get started with this template:
1. **Pick a hosting SIG.**
  Make sure that the problem space is something the SIG is interested in taking up.
  KEPs should not be checked in without a sponsoring SIG.
1. **Make a copy of this template.**
  Copy this template into the owning SIG's directory (or KEP root directory, as appropriate) and name it `YYYYMMDD-my-title.md`, where `YYYYMMDD` is the date the KEP was first drafted.
1. **Fill out the "overview" sections.**
  This includes the Summary and Motivation sections.
  These should be easy if you've preflighted the idea of the KEP with the appropriate SIG.
1. **Create a PR.**
  Assign it to folks in the SIG that are sponsoring this process.
1. **Create an issue in kubernetes/enhancements, if the enhancement will be targeting changes to kubernetes/kubernetes**
  When filing an enhancement tracking issue, please ensure to complete all fields in the template.
1. **Merge early.**
  Avoid getting hung up on specific details and instead aim to get the goal of the KEP merged quickly.
  The best way to do this is to just start with the "Overview" sections and fill out details incrementally in follow on PRs.
  View anything marked as a `provisional` as a working document and subject to change.
  Aim for single topic PRs to keep discussions focused.
  If you disagree with what is already in a document, open a new PR with suggested changes.

The canonical place for the latest set of instructions (and the likely source of this file) is [here](/keps/YYYYMMDD-kep-template.md).

The `Metadata` section above is intended to support the creation of tooling around the KEP process.
This will be a YAML section that is fenced as a code block.
See the KEP process for details on each of these items.

## Table of Contents

- [Title](#image-garbage-collector-whitelist)
  - [Table of Contents](#table-of-contents)
  - [Release Signoff Checklist](#release-signoff-checklist)
  - [Summary](#summary)
  - [Motivation](#motivation)
    - [Goals](#goals)
    - [Non-Goals](#non-goals)
  - [Proposal](#proposal)
    - [User Stories](#user-stories)
      - [Story 1: Optimized for small amount of images](#story-1)
      - [Story 2: Optimized for big amount of images](#story-2)
    - [Implementation Details/Notes/Constraints](#implementation-detailsnotesconstraints)
      - [Details of Story 1](details-of-story-1)
      - [Details of Story 2](details-of-story-2)
    - [Risks and Mitigations](#risks-and-mitigations)
      - [Risks of Story 1](risks-of-story-1)
      - [Risks of Story 2](risks-of-story-2)
  - [Design Details](#design-details)
    - [Test Plan](#test-plan)
    - [Graduation Criteria](#graduation-criteria)
      - [Examples](#examples)
        - [Alpha -> Beta Graduation](#alpha---beta-graduation)
        - [Beta -> GA Graduation](#beta---ga-graduation)
        - [Removing a deprecated flag](#removing-a-deprecated-flag)
    - [Upgrade / Downgrade Strategy](#upgrade--downgrade-strategy)
    - [Version Skew Strategy](#version-skew-strategy)
  - [Implementation History](#implementation-history)
  - [Drawbacks [optional]](#drawbacks-optional)
  - [Alternatives [optional]](#alternatives-optional)
  - [Infrastructure Needed [optional]](#infrastructure-needed-optional)

[Tools for generating]: https://github.com/ekalinin/github-markdown-toc

## Release Signoff Checklist

**ACTION REQUIRED:** In order to merge code into a release, there must be an issue in [kubernetes/enhancements] referencing this KEP and targeting a release milestone **before [Enhancement Freeze](https://github.com/kubernetes/sig-release/tree/master/releases)
of the targeted release**.

For enhancements that make changes to code or processes/procedures in core Kubernetes i.e., [kubernetes/kubernetes], we require the following Release Signoff checklist to be completed.

Check these off as they are completed for the Release Team to track. These checklist items _must_ be updated for the enhancement to be released.

- [ ] kubernetes/enhancements issue in release milestone, which links to KEP (this should be a link to the KEP location in kubernetes/enhancements, not the initial KEP PR)
- [ ] KEP approvers have set the KEP status to `implementable`
- [ ] Design details are appropriately documented
- [ ] Test plan is in place, giving consideration to SIG Architecture and SIG Testing input
- [ ] Graduation criteria is in place
- [ ] "Implementation History" section is up-to-date for milestone
- [ ] User-facing documentation has been created in [kubernetes/website], for publication to [kubernetes.io]
- [ ] Supporting documentation e.g., additional design documents, links to mailing list discussions/SIG meetings, relevant PRs/issues, release notes

**Note:** Any PRs to move a KEP to `implementable` or significant changes once it is marked `implementable` should be approved by each of the KEP approvers. If any of those approvers is no longer appropriate than changes to that list should be approved by the remaining approvers and/or the owning SIG (or SIG-arch for cross cutting KEPs).

**Note:** This checklist is iterative and should be reviewed and updated every time this enhancement is being considered for a milestone.

[kubernetes.io]: https://kubernetes.io/
[kubernetes/enhancements]: https://github.com/kubernetes/enhancements/issues
[kubernetes/kubernetes]: https://github.com/kubernetes/kubernetes
[kubernetes/website]: https://github.com/kubernetes/website

## Summary

Enhancing Kubelet image garbage collector with a whitelist of container images. Whitelisted images will be ignored during garbage collection.

## Motivation

There are some use cases when not all the container images are available from a remote repository, therefore container images are stored locally in the image cache of the Nodes. These container images can be preloaded to the Nodes instead of pulled on demand.
The image garbage collector in Kubelet deletes all unused images from the image cache in case of storage shortage.
When a Pod using such preloaded image scheduled to a Node from where the container image was garbage collected the Pod will go to ImagePullBackOff state forever.

### Goals

- Add a Kubelet command line option to list whitelisted container images
- Skip the garbage collection of the whitelisted container images

### Non-Goals

N/A

## Proposal

### User Stories

The user stories in the following chapters are alternatives of each other.

#### Story 1

Optimized for small amount of images

I as a Kubernetes administrator would like to define a set of exact image URLs to exclude these images from image garbage collection.

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

#### Details of Story 2

Optimized for big amount of images

There are two alternatives for implementation:
* Define only the hostname part of the image URL in the command line parameter. So that all images tagged to the same (fake) remote repository will be whitelisted.
* Define the whitelist with a regular expression. In this case all images matching the regexp will be whitelisted. A function to verify which images are matching would be usefull.

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

If applicable, how will the component handle version skew with other components? What are the guarantees? Make sure
this is in the test plan.

Consider the following in developing a version skew strategy for this enhancement:
- Does this enhancement involve coordinating behavior in the control plane and in the kubelet? How does an n-2 kubelet without this feature available behave when this feature is used?
- Will any other components on the node change? For example, changes to CSI, CRI or CNI may require updating that component before the kubelet.

## Implementation History

N/A

Major milestones in the life cycle of a KEP should be tracked in `Implementation History`.
Major milestones might include

- the `Summary` and `Motivation` sections being merged signaling SIG acceptance
- the `Proposal` section being merged signaling agreement on a proposed design
- the date implementation started
- the first Kubernetes release where an initial version of the KEP was available
- the version of Kubernetes where the KEP graduated to general availability
- when the KEP was retired or superseded

## Drawbacks [optional]

N/A

## Alternatives [optional]

N/A

## Infrastructure Needed [optional]

N/A
