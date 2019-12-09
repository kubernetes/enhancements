---
title: doc-policies-for-third-party-content
authors:
  - "@aimeeu"
  - "@jimangel"
  - "@sftim"
  - "@zacharysarah"
owning-sig: sig-docs
reviewers:
  - TBD
approvers:
  - TBD
editor: TBD
creation-date: 2019-10-20
last-updated: 2019-10-20
status: provisional
---

# doc-policies-for-third-party-content

## Table of Contents

<!-- toc -->
- [Summary](#summary)
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

**Note:** This KEP does not target any release; SIG Docs follows a continuous release process for website content.

## Summary

This KEP seeks consensus on how Kubernetes docs handle two types of content:
1. Content from third-party providers
2. Content hosted on multiple sites ("dual-sourced content")

Kubernetes documentation seeks to teach Kubernetes users about how
Kubernetes works, how to use in-tree Kubernetes features, and how to
build on top of Kubernetes infrastucture.
Feature docs are not a place for vendor pitches. Nevertheless, SIG Docs
receives pull requests to place advertising-like content on the Kubernetes
website. Some PRs clearly do not belong in feature docs, but other
instances are less clear.

Feature docs also contain dual-sourced content. A good practice for code project docs is to host single-sourced content only, and to provide links to other providers’ single-sourced content. This simplifies version management and reduces the work required to maintain content.
This KEP defines a policy on documentation content, so that authors can judge what is appropriate to propose and so that PR approvers can make consistent, fair decisions during the review process.

## Motivation

SIG Docs publishes Kubernetes documentation on kubernetes.io in line with its [charter](https://github.com/kubernetes/community/blob/master/sig-docs/charter.md#scope) and sets standards for website content. Prior to this KEP, there are no clear guidelines or standards for third-party and dual-sourced content.

The Kubernetes documentation is currently a mix of both 1) documentation describing the Kubernetes open source project; and 2) content describing how to install or use Kubernetes on several third party Kubernetes offerings.

Some third party content is necessary in order for Kubernetes to function. For example: Docker, networking policy (CNI plugins), Ingress controllers, and logging all require third party components.
Pages like [Logging Using Elasticsearch and Kibana](https://kubernetes.io/docs/tasks/debug-application-cluster/logging-elasticsearch-kibana/) are highly specific to a third party offering and seem more like third party product documentation than Kubernetes open source documentation.

The goal of this KEP is to reach and document a consensus on what types of third party content are appropriate for inclusion in Kubernetes documentation; standards for including third-party content; and to create consistent policies for docs handle third-party and dual-sourced content.
This KEP focuses on the following issues:

1. What third party content is appropriate for inclusion in the Kubernetes documentation?
1. Does third party content in sections such as [Getting Started](https://kubernetes.io/docs/setup/) in the docs provide sufficient value to the reader that they should remain?
1. Is there a list of content pages that are so focused on third party product usage that they should be removed or updated from the Kubernetes documentation?
1. When should the Kubernetes documentation host third party documents that we are not the source/authority of?
1. How does the Kubernetes project handle third party content that is not kept up to date or hosts?
1. Can feature owners flag when third party content is *required*, as opposed to preferable or common?
1. Who decides when to include third-party content?
1. What standard of quality and review must be met before docs include third-party content?
1. To what extent should SIG Docs advocate for third-party content providers to host their own content, or decline to host third-party content altogether?

### Goals

- Create a policy for how to include or exclude third party content in Kubernetes documentation.
- Create a policy for how to handle dual-sourced content.
- Publish clear, transparent guidelines for contributors and reviewers on how to handle third-party and dual-sourced content.
- Published guidelines for contributors and reviewers that clarify requirements for third party content and allow documentation reviewers to reject advertising.


### Non-Goals

- Outright removal of all content relating to vendors and projects outside the CNCF ecosystem.

## Proposal

Revise the [content guide](https://github.com/kubernetes/website/blob/master/content/en/docs/contribute/style/content-guide.md#contributing-content) to address community concerns

### User Stories


#### Story 1 (fictional)

Alice works for ACME, Inc and wants to gain visibility for ACME Cloud Services, which has just launched a managed Kubernetes cluster product. Alice drafts a change to a concept page so that that it mentions ACME Cloud Services’ Kubernetes product, and submits a pull request.
Bob is a documentation approver. Bob explains that Alice’s proposed change does not meet community standards, because it is functionally an advertisement.

#### Story 2 (fictional)

Charlie uses Linux, specifically Ubuntu. Charlie notices that the page about installing `kubethingy` has instructions for installing `kubethingy` on Windows and on CentOS/RHEL but not on Ubuntu. Charlie reads the guidelines on content and sees that this kind of change is acceptable (Ubuntu is one of the most popular Linux distributions, and `kubethingy` documentation is acceptable as it is already documented). Charlie drafts a change and submits a pull request.

#### Story 3 (actual)

Rafael wanted to share a Kubernetes course from an online education provider. He submitted [PR #15962](https://github.com/kubernetes/website/pull/15962) to add the course to [Overview of Kubernetes Online Training](https://kubernetes.io/docs/tutorials/online-training/overview/). The PR was not approved because SIG Docs didn’t want to add a link to third-party content over which SIG Docs have no control.

#### Story 4 (actual)

Website [PR #16203](https://github.com/kubernetes/website/pull/16203) removes Stackdriver and Elasticsearch vendor content. Since logging falls into the external add-ons category, SIG Docs decided to remove this vendor-specific content that had not been meaningfully updated in three years. SIG Docs had buy-in from SIG Instrumentation Bugs for removal; however that PR was held pending the outcome of this KEP.

#### Story 5 (actual)

In [PR #16766](https://github.com/kubernetes/website/pull/16766) @pouledodue proposed adding Hertzner Cloud Controller to the list of vendors that have implemented a cloud controller manager. That PR was held pending the outcome of this KEP.

#### Story 6 (actual)

As [hyperkube transitions to [third-party maintenance](https://github.com/kubernetes/kubeadm/issues/1889), it's unclear how to handle [hyperkube content in the Kubernetes docs](https://github.com/kubernetes/website/search?q=hyperkube&unscoped_q=hyperkube) or re-point related links.

### Implementation Details/Notes/Constraints

While SIG Docs approvers may occasionally need to consider intent, any policies produced by this KEP must be clear enough to minimize cases where intent needs to be considered.

(At the time of writing) Kubernetes requires external software to implement Cluster Networking, Ingress, Persistent Storage, and Logging. Hyperlinking to vendor software and documentation _is_ allowed; creating “how to use” content is not.

Examples of allowed content:
 - Cluster Networking
   - https://kubernetes.io/docs/concepts/cluster-administration/networking/
   - https://kubernetes.io/docs/concepts/cluster-administration/addons/
 - Ingress Controllers
   - https://kubernetes.io/docs/concepts/services-networking/ingress-controllers/#additional-controllers
 - Persistent Volumes
   - https://kubernetes.io/docs/concepts/storage/persistent-volumes/#expanding-persistent-volumes-claims

Examples of content that would not be allowed:
 - [Install a Network Policy Provider](https://kubernetes.io/docs/tasks/administer-cluster/network-policy-provider/) and child pages: how to use Calico, Cilium, Kube-router, Romana, and Weave Net for NetworkPolicy
 - [Audit](https://kubernetes.io/docs/tasks/debug-application-cluster/audit/)
 - [Use fluentd to collect and distribute audit events from log file](https://kubernetes.io/docs/tasks/debug-application-cluster/audit/#use-fluentd-to-collect-and-distribute-audit-events-from-log-file) (dual-sourced)
 - [Use logstash to collect and distribute audit events from webhook backend](https://kubernetes.io/docs/tasks/debug-application-cluster/audit/#use-logstash-to-collect-and-distribute-audit-events-from-webhook-backend) (vendor-specific content)
 - [Auditing with Falco](https://kubernetes.io/docs/tasks/debug-application-cluster/falco/) (dual-sourced)
 - [Events in Stackdriver](https://kubernetes.io/docs/tasks/debug-application-cluster/events-stackdriver/) (vendor-specific content)
 - [Logging Using Elasticsearch and Kibana](https://kubernetes.io/docs/tasks/debug-application-cluster/logging-elasticsearch-kibana/) (vendor-specific content)
 - [Logging using Stackdriver](https://kubernetes.io/docs/tasks/debug-application-cluster/logging-stackdriver/) (vendor-specific content)

### Risks and Mitigations

- Rejecting a change that is ultimately deemed acceptable.
- Accepting a change that is ultimately judged to be unacceptable vendor promotion.
- It may not always be possible to achieve consensus on whether specific third-party content should be included.
  This may lead to resentment/resistance to future third party content.
  *Mitigation*: consider including folks that also work outside the relevant SIG or subproject.

## Design Details

### Graduation Criteria

**Note:** *this KEP does not target any release*

TBD

## Drawbacks

_Why should this KEP _not_ be implemented._

TBD

## Alternatives

_Similar to the `Drawbacks` section the `Alternatives` section is used to highlight and record other possible approaches to delivering the value proposed by a KEP._

TBD
