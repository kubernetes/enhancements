# KEP 3000: Artifact Distribution Policy

<!-- toc -->
- [Summary](#summary)
- [Motivation](#motivation)
  - [Goals](#goals)
  - [Non-Goals](#non-goals)
- [Proposal](#proposal)
  - [User Stories](#user-stories)
    - [SIG Release - Image Promotion](#sig-release---image-promotion)
    - [Cloud Customer - Installing K8s via kubeadm](#cloud-customer---installing-k8s-via-kubeadm)
  - [Notes/Constraints/Caveats](#notesconstraintscaveats)
  - [Risks and Mitigations](#risks-and-mitigations)
- [Design Details](#design-details)
  - [Artifact Promotion](#artifact-promotion)
    - [Policy](#policy)
    - [Process](#process)
  - [Artifact Distribution](#artifact-distribution)
    - [Policy](#policy-1)
    - [Process](#process-1)
- [Alternatives / Background](#alternatives--background)
  - [How much is this going to save us?](#how-much-is-this-going-to-save-us)
- [Infrastructure Needed](#infrastructure-needed)
- [Hack on this doc](#hack-on-this-doc)
<!-- /toc -->

## Summary

The container images and release binaries produced by our community need a clear path to be hosted by multiple service/cloud providers.

The global community should be routed to the appropriate mirror for their country or cloud provider to ensure cost effective worldwide access.

This KEP should cover the policy and distribution mechanisms we will put in place to allow creating a globally distributed, multi-cloud and country solution.

## Motivation

Currently we push to a single provider, and distributing to the rest of community comes at great cost nearing $150k/month (mostly egress) in donations.

Additionally, some of our community members are unable to access the official release artifacts due to country level firewalls that do not them connect to Google services.

Ideally we can dramatically reduce cost and allow everyone in the world to download the artifacts released by our community.

### Goals

A policy and procedure for use by SIG Release to promote container images and release binaries to multiple registries and mirrors.

A solution to allow redirection to appropriate mirrors to lower cost and allow access from any cloud or country globally.

### Non-Goals

Anything related to creation of artifacts, bom, digital signatures.

## Proposal

There are two intertwined concepts that are part of this proposal.

First, the policy and procedures to promote/upload our artifacts to multiple providers. Our existing processes upload only to GCS buckets. Ideally we extend the existing software/promotion process to push directly to multiple providers. Alternatively we use a second process to synchronize artifacts from our existing production buckets to similar constructs at other providers.

Additionally we require a registry and artifact url-redirection solution to the local cloud provider or country.

### User Stories

#### SIG Release - Image Promotion

```feature
As a SIG Release volunteer
I want to promote our binaries/images to multiple clouds

Given a promotion / manifest
When my PR is merged
Then the promotion process occurs
```

#### Cloud Customer - Installing K8s via kubeadm

```feature
As a CLOUD end-user
I want to install Kubernetes

Given some compute resources at CLOUD
When I use kubeadm to deploy Kubernetes
Then I will be redirected to a local CLOUD registry
```

### Notes/Constraints/Caveats

The primary purpose of the KEP is getting consensus on the agreed policy and procedure to unblock our community and move forward together.

There has been a lot of activity around the technology and tooling for both goals, but we need shared agreement on policy and procedure first.

### Risks and Mitigations

This is the primary pipeline for delivering Kubernetes worldwide. Ensuring the appropriate SLAs and support as well as artifact integrity is crucial.

## Design Details

### Artifact Promotion

#### Policy

(more details needed, #sig-release-eng?)

#### Process

Currently the promotion process is primarily driven by the CIP/[promo-tool#kpromo](https://github.com/kubernetes-sigs/promo-tools#kpromo)?

### Artifact Distribution

#### Policy

#### Process

## Alternatives / Background

- Apache has a widespread mirror network
  - @dims has experience here
  - http://ws.apache.org/mirrors.cgi
  - https://infra.apache.org/mirrors.html
- [Umbrella issue: k8s.gcr.io => registry.k8s.io solution k/k8s.io#1834
](https://github.com/kubernetes/k8s.io/issues/1834)
- [ii/registry.k8s.io Implementation proposals](https://github.com/ii/registry.k8s.io#registryk8sio)
- [ii.nz/blog :: Building a data pipline for displaying Kubernetes public artifact traffic
](https://ii.nz/post/building-a-data-pipline-for-displaying-kubernetes-public-artifact-traffic/)

### How much is this going to save us?

![Cost of K8s Artifact hosting - Data Studio Graphs](https://i.imgur.com/LAn4UIE.png)

## Infrastructure Needed

It would be good to request some donations for some larger providers, including one in China, via [Cloud Native Credits program](https://www.cncf.io/credits/).

## Hack on this doc

[![hackmd-github-sync-badge](https://hackmd.io/KjHufZssQR654ShkZFUzyA/badge)](https://hackmd.io/KjHufZssQR654ShkZFUzyA)
