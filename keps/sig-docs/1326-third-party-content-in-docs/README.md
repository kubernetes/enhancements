# Doc policies for third party content

## Table of Contents

<!-- toc -->
  - [Summary](#summary)
  - [Introduction](#introduction)
  - [Motivation](#motivation)
    - [Goals](#goals)
    - [Non-Goals](#non-goals)
  - [Proposal](#proposal)
    - [User Stories](#user-stories)
      - [Story 1 (fictional)](#story-1-fictional)
      - [Story 2 (fictional)](#story-2-fictional)
      - [Story 3 (actual)](#story-3-actual)
      - [Story 4 (actual)](#story-4-actual)
      - [Story 5 (actual)](#story-5-actual)
      - [Story 6 (actual)](#story-6-actual)
    - [Implementation Details/Notes/Constraints](#implementation-detailsnotesconstraints)
    - [Risks and Mitigations](#risks-and-mitigations)
  - [Design Details](#design-details)
- [Drawbacks](#drawbacks)
  - [Alternatives](#alternatives)
<!-- /toc -->

**Note:** This KEP does not target any release; SIG Docs follows a continuous
release process for website content.

## Summary

This KEP defines the Kubernetes project consensus on in-project documentation
should handle two types of content:

1. Content from or about third-party providers ("third-party content")

   Minimize and eliminate third-party content except when necessary for Kubernetes
   to function in-project.

2. Content hosted on multiple sites ("dual-sourced content")

   Minimize and eliminate dual-sourced content except when necessary for Kubernetes
   to function in-project.

**Note:** This KEP defines "in project" to mean projects in the Kubernetes organization;
on GitHub, this covers all [kubernetes](https://github.com/kubernetes) and
[kubernetes-sigs](https://github.com/kubernetes-sigs) repositories.

## Introduction

Kubernetes documentation teaches Kubernetes users about how
Kubernetes works, how to use in-project Kubernetes features, and how to
build on top of Kubernetes infrastucture.

Feature docs also contain _dual-sourced content_: explanations about how to use thing
1 from project A with thing 2 from project or vendor B.

A good practice for code project docs is to host single-sourced content only, and to provide
links to other providers’ single-sourced content. This simplifies version management and
reduces the work required to maintain content.

This KEP defines how to handle third-party and dual-sourced content in
documentation, so that authors can judge what is appropriate to propose and so that PR
approvers can make consistent, fair decisions during the review process.

## Motivation

SIG Docs publishes Kubernetes documentation on kubernetes.io in line with its
[charter](https://github.com/kubernetes/community/blob/master/sig-docs/charter.md#scope)
and sets standards for website content. Prior to this KEP, there were no
clear guidelines or standards for third-party and dual-sourced content.

Feature docs are not a place for vendor pitches. Nonetheless, SIG Docs (along with 
other teams) sometimes receives pull requests to place advertising-like content on
the Kubernetes website. Some changes proposed in those PRs clearly do not belong in
feature docs, but other instances are less clear; this PR sets a policy to provide
that clarity.

The Kubernetes documentation is a mix of both 1) documentation
describing the Kubernetes open source project; and 2) content describing
how to install or use Kubernetes whilst relying on several third party Kubernetes
offerings.

Some third party content is necessary in order for Kubernetes to
function. For example: you need an operating system. You also typically
need or want: container runtimes (such as containerd or CRI-O),
NetworkPolicy (CNI plugins), Ingress or Gateway controllers, and logging.
Those listed outcomes all require third party components.

Before this KEP, the docs had several pages that explained how to do a relevant task,
but in a way that was too narrow in scope and too tied to details outside of Kubernetes
(such as explaining how to ship logs to a particular vendor solution). Contributors
struggled to maintain these pages and vendors hoping to add explanations of integration
with rival offerings may have felt there was an advantage to the docs that happened to
have landed first.

### Goals

The goal of Kubernetes documentation is to accurately document in-project
functionality for Kubernetes, and to eliminate barriers to effective
contribution and understanding.

The goals of this KEP are to:

* formally document a consensus on what types of third-party
  content are appropriate for inclusion in Kubernetes documentation
* define consistent policies for how Kubernetes
  and its subprojects should handle third-party and dual-sourced content.
* ensure that there are published standards for including third-party content


### Non-Goals

* Outright removal of all content relating to vendors and projects outside the
  Kubernetes project.

## Proposal

Clearly define what documentation is required so that readers understand
how to deploy, operate and consume Kubernetes clusters using features from
in-project code and its mandatory dependencies.

1. Revise the [content guide](https://kubernetes.io/docs/contribute/style/content-guide/) to achieve the KEP goal:

- Specify that Kubernetes docs are limited to content required for Kubernetes to
function in-project. Docs may include third-party OSS content for components that
require a third-party solution to function. Docs may include content for
other projects in the Kubernetes org, and content from other OSS projects that
are necessary for Kubernetes to function. Third-party content must be linked
whenever possible, rather than duplicated or hosted in k/website.

2. Revise the documentation when the KEP is approved:

- **Third-party content:** Notify stakeholders of all affected content via
GitHub issues and via a single message containing a summary of all affected
content to kubernetes-dev@googlegroups.com that non-conforming content will be
removed after 90 days.

This limits the impact to out-of-project content and gives current stakeholders
approximately one Kubernetes release cycle to migrate
third-party content to an alternate platform before removing content from
Kubernetes docs.

- **Dual-sourced content:** Where sourcing is obvious, replace dual-sourced
content with links to an authoritative single source. Where sourcing is unclear,
notify stakeholders via GitHub issues in k/website and via a single message
containing a summary of all affected content to kubernetes-dev@googlegroups.com
that non-conforming content will be removed after 90 days.

In all cases where content would be removed, provide adequate time for the
relevant SIG to review changes and notify stakeholders.

### User Stories

#### Story 1 (fictional)

Alice works for ACME, Inc and wants to gain visibility for ACME
Cloud Services, which has just launched a managed Kubernetes cluster
product. Alice drafts a change to a concept page so that that it mentions
ACME Cloud Services’ Kubernetes product, and submits a pull request.

Bob is a documentation approver. Bob explains that Alice’s proposed
change does not meet community standards, because it is functionally
an advertisement.

#### Story 2 (fictional)

Charlie uses Linux, specifically Ubuntu. Charlie notices that the page
about installing `kubethingy` has instructions for installing `kubethingy`
on Windows and on CentOS/RHEL but not on Ubuntu. Charlie reads the
guidelines on content and sees that this kind of change is acceptable
(Ubuntu is one of the most popular Linux distributions, and `kubethingy`
documentation is acceptable as it is already documented).

Charlie drafts a change and submits a pull request.

#### Story 3 (actual)

Rafael wanted to share a Kubernetes course from
an online education provider. Rafael submitted
[PR #15962](https://github.com/kubernetes/website/pull/15962)
to add the course to [Overview of Kubernetes Online
Training](https://kubernetes.io/docs/tutorials/online-training/overview/).

The PR was not approved because SIG Docs didn’t want to add a link to
third-party content over which SIG Docs have no control.

#### Story 4 (actual)

[Website PR #16203](https://github.com/kubernetes/website/pull/16203)
removes Stackdriver and Elasticsearch vendor content. Since logging
falls into the external add-ons category, SIG Docs decided to remove this
vendor-specific content that had not been meaningfully updated in three
years.

SIG Docs had buy-in from SIG Instrumentation Bugs for removal; however,
that PR was held pending the outcome of this KEP and later closed.

#### Story 5 (actual)

In [PR #16766](https://github.com/kubernetes/website/pull/16766)
@pouledodue proposed adding Hertzner Cloud Controller to the list of
vendors that have implemented a cloud controller manager. That PR was
held pending the outcome of this KEP, then later merged.

#### Story 6 (actual)

As [hyperkube transitions to third-party maintenance](https://github.com/kubernetes/kubeadm/issues/1889), it's unclear how to handle [hyperkube content in the Kubernetes docs](https://github.com/kubernetes/website/search?q=hyperkube&unscoped_q=hyperkube) or re-point related links.

### Implementation Details/Notes/Constraints

_This KEP originally included language around considering intent of contributors.
Because intent is effectively impossible to judge (and because contributions
are nearly always made with the best intent), this KEP now specifies that
third-party content is limited to what's required for in-project functionality._

SIG Docs may add its own guidelines for writing and reviewing ambiguous content.

For example:
> Kubernetes requires out-of-tree software and tools to implement: cluster
> networking, Ingress, persistent storage, and logging. Hyperlinking to vendor software
> and documentation _is_ allowed; creating “how to use” content for a specific vendor
> is discouraged.

Pages that fit with that example guideline:
 - Cluster Networking
   - https://kubernetes.io/docs/concepts/cluster-administration/networking/
   - https://kubernetes.io/docs/concepts/cluster-administration/addons/
 - Ingress Controllers
   - https://kubernetes.io/docs/concepts/services-networking/ingress-controllers/#additional-controllers
 - Persistent Volumes
   - https://kubernetes.io/docs/concepts/storage/persistent-volumes/#expanding-persistent-volumes-claims

### Risks and Mitigations

None known

## Design Details

Once the community have reached consensus, prepare a PR to update the
existing [content guide](https://github.com/kubernetes/website/blob/master/content/en/docs/contribute/style/content-guide.md#contributing-content).

Once the KEP is approved, merge the KEP and then update website content accordingly.

# Drawbacks

SIG Docs identified no meaningful drawbacks.

## Alternatives

The only real alternative&mdash;approving third-party content without a vetting policy&mdash;is unacceptable, and would degrade site outcomes across metrics of quality, searchability, and trust.
