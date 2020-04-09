---
title: doc-policies-for-third-party-content
authors:
  - "@aimeeu"
  - "@jimangel"
  - "@sftim"
  - "@zacharysarah"
owning-sig: sig-docs
reviewers:
  - "@jaredbhatti"
  - "@kbarnard10"
approvers:
  - "@cblecker"
  - "@derekwaynecarr"
  - "@dims"
editor: "@zacharysarah"
creation-date: 2019-10-20
last-updated: 2019-10-20
status: provisional
---

# doc-policies-for-third-party-content

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
    - [Graduation Criteria](#graduation-criteria)
- [Drawbacks](#drawbacks)
  - [Alternatives](#alternatives)
<!-- /toc -->

**Note:** This KEP does not target any release; SIG Docs follows a continuous
release process for website content.

## Summary

This KEP seeks consensus on how Kubernetes docs handle two types of content:

1. Content from or about third-party providers ("third-party content")

Minimize and eliminate third-party content except when necessary for Kubernetes 
to function in-project.

2. Content hosted on multiple sites ("dual-sourced content")

Minimize and eliminate dual-sourced content except when necessary for Kubernetes
to function in-project.

**Note:** This KEP defines "in project" to mean projects in the Kubernetes org, 
which includes the [kubernetes](https://github.com/kubernetes) and 
[kubernetes-sigs](https://github.com/kubernetes-sigs) repositories.

## Introduction

Kubernetes documentation teaches Kubernetes users about how
Kubernetes works, how to use in-project Kubernetes features, and how to
build on top of Kubernetes infrastucture.

Feature docs are not a place for vendor pitches. Nonetheless, SIG Docs sometimes
receives pull requests to place advertising-like content on the Kubernetes
website. Some PRs clearly do not belong in feature docs, but other
instances are less clear.

Feature docs also contain dual-sourced content. A good practice for code
project docs is to host single-sourced content only, and to provide
links to other providers’ single-sourced content. This simplifies
version management and reduces the work required to maintain content.

This KEP defines how to handle third-party and dual-sourced content in 
documentation, so that authors can
judge what is appropriate to propose and so that PR approvers can make
consistent, fair decisions during the review process.

## Motivation

SIG Docs publishes Kubernetes
documentation on kubernetes.io in line with its
[charter](https://github.com/kubernetes/community/blob/master/sig-docs/charter.md#scope)
and sets standards for website content. Prior to this KEP, there are no
clear guidelines or standards for third-party and dual-sourced content.

The Kubernetes documentation is currently a mix of both 1) documentation
describing the Kubernetes open source project; and 2) content describing
how to install or use Kubernetes on several third party Kubernetes
offerings.

Some third party content is necessary in order for Kubernetes to
function. For example: container runtimes (containerd, CRI-o, Docker), 
networking policy (CNI plugins), Ingress controllers, and logging all require 
third party components. Pages like [Logging Using Stackdriver](https://kubernetes.io/docs/tasks/debug-application-cluster/logging-stackdriver/) 
are highly specific to a third party offering and seem more like third party 
product documentation than Kubernetes open source documentation.

### Goals

The goal of Kubernetes documentation is to accurately document in-project
functionality for Kubernetes, and to eliminate barriers to effective
contribution and understanding.

The goal of this KEP is to reach and document a consensus on what
types of third-party content are appropriate for inclusion in Kubernetes
documentation; standards for including third-party content; and to create
consistent policies for docs handle third-party and dual-sourced content.

To address its goal, this KEP focuses on the following issues:

<del>

1. What third party content is appropriate for inclusion in the Kubernetes
documentation?

Proposed: Third-party content is permitted if it is required for Kubernetes to
function in-project.

1. Does third party content in sections such as [Getting Started](https://kubernetes.io/docs/setup/)
in the docs provide sufficient value to the reader that they should remain?

Casual consensus says yes, with one modification:
- Eliminate the [production environment table](https://kubernetes.io/docs/setup/#production-environment)
with a link to [certified conformance partners](https://kubernetes.io/partners/#conformance).

1. Is there a list of content pages that are so focused on third party product
usage that they should be removed or updated from the Kubernetes documentation?

See https://github.com/kubernetes/website/issues/15748.

1. When should the Kubernetes documentation host third party content that isn't
maintained by a Kubernetes SIG?

As infrequently as possible, with linking preferred to hosting. 

1. How does the Kubernetes project handle third party content that is not kept
up to date or hosts?

If content isn't refreshed within 180 days, notify stakeholders of 90 days to 
update content or migrate it elsewhere before removing it. Notification
specifically includes:

- Mailing an initial list of affected pages to kubernetes-dev@googlegroups.com
- Announcing the policy change in two Kubernetes community meetings in a row
- Posting a notification of the policy change on the Kubernetes blog
- Notifying SIG PR review aliases on GitHub in PRs that remove affected content

1. Can feature owners flag when third party content is *required*, as opposed to
preferable or common?

Is this capability required for KEP approval?

1. Who decides when to include third-party content?

SIGs responsible for particular features can include third-party content at
their discretion, preferably by linking to the third party's own documentation.

1. What standard of quality and review must be met before docs include
third-party content?

Third-party content must be necessary for Kubernetes to function in-project.

1. To what extent should SIG Docs advocate for third-party content providers to 
host their own content, or decline to host third-party content altogether?

Kubernetes docs publish third-party content only if:

- It's necessary for Kubernetes to function. For example: container runtimes 
(containerd, CRI-o, Docker), networking policy (CNI plugins), Ingress 
controllers, and logging.

- It's an applied example of another project in the Kubernetes GitHub org. This
includes the [kubernetes](https://github.com/kubernetes) and 
[kubernetes-sigs](https://github.com/kubernetes-sigs) repositories.

Third-party content should be linked instead of hosted whenever possible. 

</del>

1. Clearly define what documentation is required so that readers understand
   how to deploy, operate and consume Kubernetes clusters using features from
   in-project code and its mandatory dependencies.

### Non-Goals

1. Outright removal of all content relating to vendors and projects outside the 
   Kubernetes project.

## Proposal

1. Revise the [content guide](https://github.com/kubernetes/website/blob/master/content/en/docs/contribute/style/content-guide.md#contributing-content) to achieve the KEP goal:

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

This KEP originally included language around considering intent of contributors.
Because intent is effectively impossible to judge (and because contributions
are nearly always made with the best intent), this KEP now specifies that 
third-party content is limited to what's required for in-project functionality.

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

Pages to review and possibly revise, if that guideline were in place:
 - [Install a Network Policy Provider](https://kubernetes.io/docs/tasks/administer-cluster/network-policy-provider/) and child pages: how to use Calico, Cilium, Kube-router, Romana, and Weave Net for NetworkPolicy
 - [Audit](https://kubernetes.io/docs/tasks/debug-application-cluster/audit/)
 - [Use fluentd to collect and distribute audit events from log file](https://kubernetes.io/docs/tasks/debug-application-cluster/audit/#use-fluentd-to-collect-and-distribute-audit-events-from-log-file) (dual-sourced)
 - [Use logstash to collect and distribute audit events from webhook backend](https://kubernetes.io/docs/tasks/debug-application-cluster/audit/#use-logstash-to-collect-and-distribute-audit-events-from-webhook-backend) (vendor-specific content)
 - [Auditing with Falco](https://kubernetes.io/docs/tasks/debug-application-cluster/falco/) (dual-sourced)
 - [Events in Stackdriver](https://kubernetes.io/docs/tasks/debug-application-cluster/events-stackdriver/) (vendor-specific content)
 - [Logging Using Elasticsearch and Kibana](https://kubernetes.io/docs/tasks/debug-application-cluster/logging-elasticsearch-kibana/) (vendor-specific content)
 - [Logging using Stackdriver](https://kubernetes.io/docs/tasks/debug-application-cluster/logging-stackdriver/) (vendor-specific content)

### Risks and Mitigations

None known

## Design Details

### Graduation Criteria

**Note:** *this KEP does not target any release*

Once the community have reached consensus, prepare a PR to update the
existing [content guide](https://github.com/kubernetes/website/blob/master/content/en/docs/contribute/style/content-guide.md#contributing-content).

Once the KEP is approved, merge the KEP and then update website content accordingly.

# Drawbacks

SIG Docs identified no meaningful drawbacks.

## Alternatives

The only real alternative&mdash;approving third-party content without a vetting policy&mdash;is unacceptable, and would degrade site outcomes across metrics of quality, searchability, and trust.
