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
# KEP-3786: Migrate Shared kube-proxy Code into Staging 
# Index
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
      - [Unit tests](#unit-tests)
      - [Integration tests](#integration-tests)
      - [e2e tests](#e2e-tests)
  - [Graduation Criteria](#graduation-criteria)
- [Production Readiness Review Questionnaire](#production-readiness-review-questionnaire)
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
    - [ ] (R) Ensure GA e2e tests for meet requirements for [Conformance Tests](https://github.com/kubernetes/community/blob/master/contributors/devel/sig-architecture/conformance-tests.md)
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

After about a year and a half of testing a [new service proxy implementation](https://github.com/kubernetes-sigs/kpng/), and
collecting sig-network and community feedback, it is clear that a shared library (referred to as kube-proxy-lib in this document) designed to make building new service proxies easier is needed. Specifically, it is desired by members of the Kubernetes networking community who are attempting to build specialized networking tools. However, in order to prevent the community from having to maintain
two separate sets of code (the core kube-proxy code and the aforementioned new library) while also ensuring the stability of the existing proxies, it makes sense to work incrementally towards this ultimate goal.

This KEP describes the **first step** towards the ultimate end goal of providing
a shared library of core kube-proxy functionality which can be easily consumed by the community.
Specifically, it will work to move the existing [shared kube-proxy code](https://github.com/kubernetes/kubernetes/tree/master/pkg/proxy) into [staging](https://github.com/kubernetes/kube-proxy).

This initial work will open the door for future improvements to kube-proxy code to be made in both
a safe an incremental way.

## Motivation

There have been several presentations, issues, and projects dedicated to reusing kube-proxy logic while extending it to embrace
different backend technologies (i.e. nftables, eBPF, Open vSwitch, and so on).  This KEP attempts to work towards making a library which will facilitate
this type of work ultimately making it much easier to write and maintain new proxy implementations.

A previous attempt at a broad solution to this problem was explored in the [KPNG project](https://github.com/kubernetes-sigs/kpng/), which exhibits many properties that allow for such goals to be accomplished.  However, because it introduced many new features and would result in large breaking changes if it were
to be incorporated back in tree, it became clear we needed to decompose this large task into smaller pieces. Therefore, we've decided to keep things simple and start by moving the existing shared kube-proxy code into staging where it can be iterated and augmented in an safe, consumable and incremental manner.

### Goals

- Move the [shared kube-proxy code k/k/pkg/proxy](https://github.com/kubernetes/kubernetes/tree/master/pkg/proxy), and relevant
networking utilities (i.e `pkg/util/iptables`) into the kube-proxy [staging repository](https://github.com/kubernetes/kube-proxy).
- Ensure all existing kube-proxy unit and e2e coverage remains the same or is improved.
- Ensure the shared code can be easily consumed by external users to help write new out of tree service proxies.
- Write documentation detailing how external consumers can utilize kube-proxy staging library.

### Non-Goals

- Write any more new "in-tree" service proxies.
- Make any incompatible architectural or deployment changes to the existing kube-proxy implementations.
- Tackle any complex new deployment scenarios (This is solved by [KPNG](https://github.com/kubernetes-sigs/kpng/))

## Proposal

We propose to build a new library in the [kube-proxy staging repository](https://github.com/kubernetes/kube-proxy). This repository will be vendored by the core implementations and developers who want to build new service proxy implementations, providing them with:

- A vendorable golang library that defines a few interfaces which can be easily implemented by a new service proxy, that responds to EndpointSlice and Service changes.
- Documentation on how to build a kube proxy with the library, based on [So You Want To Write A Service Proxy...](https://github.com/kubernetes-sigs/kpng/blob/master/doc/service-proxy.md) and other similar documents.

Not only will this make writing new backends easier, but through incremental changes and optimizations to the new library we hope to also improve the existing proxies, making [legacy bugs](https://github.com/kubernetes/kubernetes/issues/112604) easier to fix in the future.

### User Stories (Optional)

#### Story 1

As a networking technology startup I want to easily make a new service proxy implementation without maintaining the logic of talking to the APIServer, caching its data, or calculating an abbreviated/proxy-focused representation of the Kubernetes networking state space.  I'd like a wholesale framework I can simply plug my custom dataplane oriented logic into.

#### Story 2

As a service proxy maintainer, I don't want to have to understand the in-tree internals of a networking backend in order to simulate or write core updates to the logic of the kube-proxy locally.

#### Story 3

As a Kubernetes developer I want to make maintaining the shared proxy code easier, and allow for updates to that code to be completed in a more incremental and well tested way.

### Notes/Constraints/Caveats (Optional)

TBD

### Risks and Mitigations

Since this KEP involves the movement of core code bits there are some obvious risks, however they will be mitigated by ensuring
all existing unit and e2e test coverage is kept up to date and/or improved througout the process.

## Design Details

**NOTE: This section is still under active development please comment with any further ideas**

The implementation of this kep will begin by moving the various networking utilities (i.e `pkg/util/iptables`, `pkg/util/conntrack`,
`pkg/util/ipset`, `pkg/util/ipvs`) used by `pkg/proxy` to the staging repo using @danwinship's [previous attempt](https://github.com/kubernetes/kubernetes/pull/112886) as a guide. Additionally we will need to [re-look](https://github.com/kubernetes/utils/pull/165) into moving `pkg/util/async` out of `pkg/util/async` and into [`k8s/utils`](https://github.com/kubernetes/utils).

Following this initial work, the [shared kube-proxy code](https://github.com/kubernetes/kubernetes/tree/master/pkg/proxy) as it stands today will be moved into the kube-proxy staging repo. Throughout this process it's crucial that all unit and e2e test coverage
is either maintained or improved to ensure stability of the existing in-tree proxies.

In conclusion, documentation will be written to help users consume the now vendorable kube-proxy code.

Additional steps (most likely described in further detail in a follow-up kep) will include:
    - Building up more tooling around testing and use of the library for external consumers.
    - Analysis possible improvements and updates to the shared library code, using the POC done in KPNG as a reference, to make writing new out of tree service proxy implementations easier.

### Test Plan

##### Unit tests

All existing kube-proxy and associated library unit test coverage **MUST** be maintained or improved.

##### Integration tests

All existing kube-proxy and associated library integration test coverage **MUST** be maintained or improved.

##### e2e tests

All existing kube-proxy and associated library e2e test coverage **MUST** be maintained or improved.


### Graduation Criteria

N/A

## Production Readiness Review Questionnaire

### Dependencies

N/A

### Scalability

N/A the core functionality will remain the same

###### Will enabling / using this feature result in any new API calls?

No

###### Will enabling / using this feature result in introducing new API types?

No, it wont result in new K8s APIs.

###### Will enabling / using this feature result in any new calls to the cloud provider?

No

###### Will enabling / using this feature result in non-negligible increase of resource usage (CPU, RAM, disk, IO, ...) in any components?

No

### Troubleshooting

###### How does this feature react if the API server and/or etcd is unavailable?

The APIServer going down will prevent this library from generally working as would be expected in normal cases, where all incoming
Kubernetes networking data is being polled from the APIServer.  However, since this library will be flexible, there are other ways
of providing it with networking information, and thus, APIServer outage doesn't have to result in the library itself being entirely unusable.

###### What are other known failure modes?

###### What steps should be taken if SLOs are not being met to determine the problem?

## Implementation History

["Librarification" PR into KPNG](https://github.com/kubernetes-sigs/kpng/pull/389).

## Drawbacks

## Alternatives

We could retain the existing kube proxy shared code in core kubernetes and simply better document the datastructures and golang maps used for kubernetes client operations and client side caching. However, that would still require external proxy implementations to copy and paste large amounts of code.  The other option is to not tackle this problem in-tree and to move forward with the singular development of external projects like KPNG as the overall framework for solving these problems.  The drawbacks to this include of this is that it is opinionated towards a raw GRPC implementation and other users (i.e. XDS) want something more decoupled possibly.  This realization has inspired this KEP.

## Infrastructure Needed (Optional)

None