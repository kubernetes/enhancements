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
# KEP-2104: kube-proxy library (KEP-2104, breakout)
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

After about a year and a half of testing a new kube-proxy implementation (https://github.com/kubernetes-sigs/kpng/) and
collecting sig-network and community feedback, it became clear that an interface for building new kube proxy's without  an
opinionated implementation is desired by the Kubernetes networking commmunity attempting to build specialied networking
tools.  This KEP distills the goals of such an interface and propose it's lifecycle and official support policy for sig-network.

## Motivation

There have been several presentations, issues, and projects dedicated to reusing kube proxy logic while extending it to embrace
different backend technologies (i.e. NFT, eBPF, openvswitch, and so on).  This KEP attempts to make a library which will facilitate
this type of work.  

A general solution to this problem is explored in the KPNG project (https://github.com/kubernetes-sigs/kpng/), which exhibits many properties
that allow for such goals to be accomplished.  These are enabled by:

- A Generic "*Diff store"* which provides a client side, in memory data model for performant, declarative, generic calculation of differences between the Kubernetes networking state space from one time point to another (i.e. a replacement for the implicit change tracking functionality in k/k/pkg/proxy https://github.com/kubernetes/kubernetes/blob/master/pkg/proxy/endpoints.go#L161).
- A *library that simplifies consumption* of the Kubernetes network and its topology easy to consume for an individual node, abstracting away API Semantics (like topology) from underlying network routing rules.
- *Definition of types* which have a minimal amount of boiler plate, can be created without using the Kubernetes API data model (and thus used to extend proxying behaviour to things outside of the Kubernetes pod, service, and endpoint semantics), and which can describe routing of Kubernetes VIPs (services) to endpoints in a generic way that is understandable to people who don't work on Kubernetes on a day-to-day basis.

### Goals

- Build a vendorable repository "kube-proxy-lib" which can be used to make new kube-proxy's.
- Exemplify the use of such a repository in a mock "backend" which uses the repository to process and respond to changes in the Kubernetes networking state.
- Define a policy around versioning and releasing of "kube-proxy-lib".
- Create a CI system that runs in test-grid which tests kube-proxy-lib compatibility with the latest Kubernetes releases continuously, and which can be used to inform API changes for regressions that break "kube-proxy-lib".

### Non-Goals

- Rewrite or decouple the core , in-tree linux Kubernetes kube-proxy implementations, which are relied on by too many users to be tolerant to major changes.
- Force a new architecture for the standard kube-proxy on to naive users.

## Proposal

We propose to build a kubernetes-sigs/kube-proxy-lib repository.  This repository will be vendored by people wanting to build a new Kube proxy, and provide them with:
- A vendorable golang library that defines a few interfaces that can be easily implemented as a new kube proxy, which respond to endpoint slices and services changes.
- Documentation on how to build a kube proxy, based on https://github.com/kubernetes-sigs/kpng/blob/master/doc/service-proxy.md and other similar documents.
- integration test tooling, similar to the KPNG project's, which allows users to locally implement network routing logic in a small golang program which is not
directly connected to the Kubernetes API server, for local, iterative development of Kubernetes network proxy tooling.

### User Stories (Optional)

#### Story 1

As a networking technology startup I want to make my own kube-proxy implementation but don't want to maintain the logic of talking to the APIServer, caching its data, or calculating an abbreviated/proxy-focused representation of the Kubernetes networking state space.  I'd like a wholesale framework I can simply plug my logic into. 

#### Story 2

As a Kubernetes maintainer, I don't want to have to understand the internals of a networking backend in order to simulate or write core updates to the logic of the kube-proxy locally.

### Notes/Constraints/Caveats (Optional)

- sending the full-state could be resource consuming on big clusters, but it should still be O(1) to
  the actual kernel definitions (the complexity of what the node has to handle cannot be reduced
  without losing functionality or correctness).

### Risks and Mitigations

No risks, because we arent removing the in tree proxy as part of this repo, but rather, proposing a library for kube proxy extenders
to use optionally.  There may be risks, eventually, if we write another KEP to *replatform* sig-networks k/k/pkg/proxy implementations
on top of this library, but that would be in a separate KEP. 

## Design Details

### Test Plan

##### Unit tests

##### Integration tests

##### e2e tests

### Graduation Criteria

We will version "kube-proxy-lib" as 1.0 once it is known to be powering at least one backend proxy which can be run by an end userr, which is able to pass the Kubernetes sig-network (non disruptive) and Conformance suites, including Layer 7 and serviceType=Loadbalancer tests which currently run in the prow sig-network e2es.

## Production Readiness Review Questionnaire

### Dependencies

This project will depend on the Kubernetes client-go library to acquire Service, Endpoints, EndpointSlices, and other
networking API objects.


### Scalability

The ability to scale this service will be equivalent to that of the existing kube-proxy, insofar as the fact that
it will watch the same endpoints (as the existing kube-proxy) and generally then be used to forward to traffic to a single
backend loadbalancing technology (i.e. ebpf, nft, iptables, ...) as does the existing kube-proxy daemonset.

###### Will enabling / using this feature result in any new API calls?

No

###### Will enabling / using this feature result in introducing new API types?

Yes. There will be an in-memory API that is supported by this library, that is incremented overr time
to reflect changes in the Kubernetes API.  Upgrading a verrsion of this library may require users to change
the way they consume its various data structures.  We will provide migration guides when this happens between
versions.

###### Will enabling / using this feature result in any new calls to the cloud provider?

No

###### Will enabling / using this feature result in non-negligible increase of resource usage (CPU, RAM, disk, IO, ...) in any components?

### Troubleshooting

###### How does this feature react if the API server and/or etcd is unavailable?

The APISErver going down will prevent this library from generally working as would be expected in normal cases, where all incoming
Kubernetes networking data is being polled from the APIServer.  However, since this library will be flexible, there are other ways
of providing it with networking information, and thus, APIServer outage doesn't have to result in the library itself being entirely unusable.

###### What are other known failure modes?

###### What steps should be taken if SLOs are not being met to determine the problem?

## Implementation History


"Librarification" PR into KPNG: https://github.com/kubernetes-sigs/kpng/pull/389. 

## Drawbacks

## Alternatives

We could retain the existing kube proxy, but that would require copy and pasting alot of code, and continuing to document a the datastructures and golang maps for diffing which were never originally designed for external consumption.  The exsiting Kube proxy's non-explicit mapping and diffing of Kubernetes API objects inspired the KPNG project, originally.

We could also leverage the KPNG project as an overall framework for solving these problems.  The only drawback of this is that it is opinionated towards a raw GRPC implementaiton and other users (i.e. XDS) want something more decoupled possibly.  This realization has inspired this KEP. 

## Infrastructure Needed (Optional)

None
