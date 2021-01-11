<!--
**Note:** When your KEP is complete, all of these comment blocks should be removed.

To get started with this template:

- [ ] **Pick a hosting SIG.**
  Make sure that the problem space is something the SIG is interested in taking
  up.  KEPs should not be checked in without a sponsoring SIG.
- [ ] **Create an issue in kubernetes/enhancements**
  When filing an enhancement tracking issue, please ensure to complete all
  fields in that template.  One of the fields asks for a link to the KEP.  You
  can leave that blank until this KEP is filed, and then go back to the
  enhancement and add the link.
- [ ] **Make a copy of this template directory.**
  Copy this template into the owning SIG's directory and name it
  `NNNN-short-descriptive-title`, where `NNNN` is the issue number (with no
  leading-zero padding) assigned to your enhancement above.
- [ ] **Fill out as much of the kep.yaml file as you can.**
  At minimum, you should fill in the "title", "authors", "owning-sig",
  "status", and date-related fields.
- [ ] **Fill out this file as best you can.**
  At minimum, you should fill in the "Summary", and "Motivation" sections.
  These should be easy if you've preflighted the idea of the KEP with the
  appropriate SIG(s).
- [ ] **Create a PR for this KEP.**
  Assign it to people in the SIG that are sponsoring this process.
- [ ] **Merge early and iterate.**
  Avoid getting hung up on specific details and instead aim to get the goals of
  the KEP clarified and merged quickly.  The best way to do this is to just
  start with the high-level sections and fill out details incrementally in
  subsequent PRs.

Just because a KEP is merged does not mean it is complete or approved.  Any KEP
marked as a `provisional` is a working document and subject to change.  You can
denote sections that are under active debate as follows:

```
<<[UNRESOLVED optional short context or usernames ]>>
Stuff that is being argued.
<<[/UNRESOLVED]>>
```

When editing KEPS, aim for tightly-scoped, single-topic PRs to keep discussions
focused.  If you disagree with what is already in a document, open a new PR
with suggested changes.

One KEP corresponds to one "feature" or "enhancement", for its whole lifecycle.
You do not need a new KEP to move from beta to GA, for example.  If there are
new details that belong in the KEP, edit the KEP.  Once a feature has become
"implemented", major changes should get new KEPs.

The canonical place for the latest set of instructions (and the likely source
of this file) is [here](/keps/NNNN-kep-template/README.md).

**Note:** Any PRs to move a KEP to `implementable` or significant changes once
it is marked `implementable` must be approved by each of the KEP approvers.
If any of those approvers is no longer appropriate than changes to that list
should be approved by the remaining approvers and/or the owning SIG (or
SIG Architecture for cross cutting KEPs).
-->
# KEP-1687: Hierarchical Namespaces As A Subproject

<!--
This is the title of your KEP.  Keep it short, simple, and descriptive.  A good
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
- [Summary](#summary)
- [Motivation](#motivation)
- [Goals](#goals)
- [Proposal](#proposal)
- [Risks and Mitigations](#risks-and-mitigations)
- [Graduation Criteria](#graduation-criteria)
  - [Intake](#intake)
  - [Beta](#beta)
- [Alternatives](#alternatives)
  - [Absorb Hierarchical Namespace Controller Into Another Project](#absorb-hierarchical-namespace-controller-into-another-project)
  - [Hierarchical Namespace Controller Continues As A Non-Sig Project](#hierarchical-namespace-controller-continues-as-a-non-sig-project)
  - [Abandon Hierarchical Namespace Controller](#abandon-hierarchical-namespace-controller)
- [Implementation History](#implementation-history)
<!-- /toc -->


## Summary

[Hierarchical Namespace Controller](https://github.com/kubernetes-sigs/multi-tenancy/tree/master/incubator/hnc)
makes it easier for you to create and manage namespaces in your cluster.
For example, you can create a hierarchical namespace under your team's namespace,
even if you don't have cluster-level permission to create namespaces, and easily
apply policies like RBAC and Network Policies across all namespaces in your
team (e.g. a set of related microservices).

You can read more about hierarchical namespaces in the
[HNC Concepts doc](https://docs.google.com/document/d/1R4rwTweYBWYDTC9UC-qThaMkPk8gtflr_ycHOOqzBQE/edit).


## Motivation

The Hierarchical Namespace Controller is currently being developed within the
Multi-Tenancy Working Group, (of which Sig-Auth is the sponsor). As Working Groups
are not meant to own code, and Hierarchical Namespace Controller is nearly to
MVP status, a permant home is required. Additionally, having a permanent home
for Hierarchical Namespace Controller prior to officially releasing will prevent
cumbersome migrations of client libraries if a move were to happen at a later time.


## Goals

Establish a new repository and permanent home for Hierarchical Namespace
Controller at github.com/kubernetes-sigs/hierarchical-namespaces to be
maintained by the open source Kubernetes community and governed as a subproject
of sig-auth.


## Proposal

The current multi-tenancy repository will be transferred directly. Subsequent
pull requests will then remove HNC from the multi-tenancy repository and
reorganize the new Hierarchical Namespace Controller repository as needed.
The new source control location will become the authoritative source of truth
for all issues and pull requests. As an independent subproject under sig-auth,
Hierarchical Namespace Controller will continue to maintain the Apache license
hosted [here](https://github.com/kubernetes-sigs).

The following group of community members will serve as initial maintainers of
the new repository:

* @adrianludwin
* @rjbez17

Maintainers will devote time to transfer, maintain, and release the Hierarchical
Namespace Controller code base in a time bound manor. Maintainers will document
features, blog, evangelize, and respond to the community on slack, groups,
forums, etc. Maintainers will serve as the initial owners of the subproject.


## Risks and Mitigations

 There are no obvious risks with this proposal. Hierarchical Namespace
 Controller is currently in pre-alpha and has no apparent adoption.


## Graduation Criteria

### Intake

* API and code quality review completed
* API security review completed
* Experimental warnings on the readme to indicate this is not an officially supported k8s-sigs product

### Beta

* Evidence of usage in the community
* API promoted to v1beta1
* All documentation, source control, tests and project roadmaps are updated and
  inline with sig standards
* Commitment to follow regular [Kubernetes API upgrade standards](https://github.com/kubernetes/community/blob/master/contributors/devel/sig-architecture/api_changes.md)


## Alternatives

### Absorb Hierarchical Namespace Controller Into Another Project

There are no apparent alternative projects to absorb Hierarchical Namespace
Controller. However, our stated goal is to find a permant home, regardless
where it may be.

### Hierarchical Namespace Controller Continues As A Non-Sig Project

The Hierarchical Namespace Controller is currently being developed within the
Multi-Tenancy WG. This poses a problem because working groups are not meant to
own code. As Sig-Auth is the sponsor for the Multi-Tenancy WG, it is fitting for
it to move into a Sig-Auth subproject. All current code has been completed in a
kuberenetes-sig repository and has followed it's governance. Moving Hierarchical
Namespace Controller to a different foundation or non-CNCF owner seems unfitting.

### Abandon Hierarchical Namespace Controller

There is quite a bit of community interest for the Hierarchical Namespace
Controller project to continue on. As we are still pre-alpha, this option would
not affect production workloads. However, with eager maintainers and
contributors, moving to a different foundation is far more preferable.

## Implementation History

- KEP created - April 14 2020
- KEP updated to follow new process - April 14 2020
- KEP updated formatting to make it easier to review - April 23 2020
- KEP updated to include additional graduation criteria - April 28 2020
- KEP updated with grammatical mistakes found in PR - May 5 2020
- KEP updated with graduation criteria requested from reviewers - May 8 2020
- KEP updated with status of `implementable` - Jan 6 2021
