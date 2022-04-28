# KEP 3000: Image Promotion and Distribution Policy

<!-- toc -->

- [Summary](#summary)
- [Background (from wiki)](#background-from-wiki)
- [Motivation](#motivation)
- [Why a new domain?](#why-a-new-domain)
- [How can we help?](#how-can-we-help)
  - [Goals](#goals)
  - [Non-Goals](#non-goals)
  - [What is not in scope](#what-is-not-in-scope)
  - [What are good goals to shoot for](#what-are-good-goals-to-shoot-for)
- [Proposal](#proposal)
- [What exactly are you doing?](#what-exactly-are-you-doing)
  - [User Stories](#user-stories)
    - [SIG Release - Image Promotion](#sig-release---image-promotion)
    - [Cloud Customer - Installing K8s via kubeadm](#cloud-customer---installing-k8s-via-kubeadm)
  - [Notes/Constraints/Caveats](#notesconstraintscaveats)
  - [Risks and Mitigations](#risks-and-mitigations)
- [Design Details](#design-details)
  - [Release Promotion](#release-promotion)
    - [Policy](#policy)
    - [Process](#process)
  - [Artifact Distribution](#artifact-distribution)
    - [Policy](#policy-1)
    - [Process](#process-1)
- [Alternatives / Background](#alternatives--background)
  - [How much is this going to save us?](#how-much-is-this-going-to-save-us)
- [Infrastructure Needed](#infrastructure-needed)
- [Hack this doc](#hack-this-doc)
<!-- /toc -->

## Summary

The container images and release binaries produced by our community need a clear path to be hosted by multiple service/cloud providers.

The global community should be routed to the appropriate mirror for their country or cloud provider to ensure cost effective worldwide access.

This KEP should cover the policy and distribution mechanisms we will put in place to allow creating a globally distributed, multi-cloud and country solution.

## Background (from wiki)

## Motivation

For a few years now, we have been using k8s.gcr.io in all our repositories as default repository for downloading images from.

The cost of distributing Kubernetes comes at great cost nearing $150kUSD/month (mostly egress) in donations.

Additionally some of our community members are unable to access the official release container images due to country level firewalls that do not them connect to Google services.

Ideally we can dramatically reduce cost and allow everyone in the world to download the container iamges released by our community.

We are now used to using the [image promoter process](https://github.com/kubernetes/enhancements/tree/master/keps/sig-release/1734-k8s-image-promoter) to promote images to the official kubernetes container registry using the infrastructure (GCR staging repos etc) provided by [sig-k8s-infra](https://github.com/kubernetes/k8s.io/tree/main/k8s.gcr.io)

## Why a new domain?

So far we (all kubernetes project) are using GCP as our default infrastructure provider for all things like GCS, GCR, GKE based prow clusters etc. Google has graciously sponsored a lot of our infrastructure costs as well. However for about a year or so we are finding that our costs are sky-rocketing because the community usage of this infrastructure has been from other cloud providers like AWS, Azure etc. So in conjunction with CNCF staff we are trying to put together a plan to host copies of images and binaries nearer to where they are used rather than incur cross-cloud costs.

One part of this plan is to setup a redirecting web service, that can identify where the traffic is coming from and redirect to the nearest image layer/repository. This is why we are setting up a new service using what we call an [oci-proxy](https://github.com/kubernetes-sigs/oci-proxy) for everyone to use. This redirector will identify traffic coming from, for example, a certain AWS region, then will setup a HTTP redirect to a source in that AWS region. If we get traffic from GKE/GCP or we don't know where the traffic is coming from, it will still redirect to the current infrastructure (k8s.gcr.io).

## How can we help?

When Kubernetes master opens up for v1.25 development, we need to update all default urls in our code and test harness to the new registry url. As a team sig-k8s-infra is signing up to ensure that this oci-proxy based registry.k8s.io will be as robust and available as the current setup. As a backup, we will continue to run the current k8s.gcr.io as well. So do not worry about that going away. Turning on traffic to the new url will help us monitor and fix things if/when they break and we will be able to tune traffic and lower our costs of operation.

### Goals

A policy and procedure for use by SIG Release to promote container images and release binaries to multiple registries and mirrors.

A solution to allow redirection to appropriate mirrors to lower cost and allow access from any cloud or country globally.

### Non-Goals

Anything related to creation of artifacts, bom, staging buckets.

### What is not in scope

- Currently we focus on AWS only. We are getting a lot of help from AWS in terms of technical details as well as targeted infrastructure costs for standing up and running this infrastructure

### What are good goals to shoot for

- In terms of cost reduction, monitor GCP infrastructure and get to the point where we fully avoid serving large binary image layers from GCR/GCS
- We can add other AWS regions and clouds as needed in well known documented way
- Seamless transition for the community from the old k8s.gcr.io to registry.k8s.io with same rock solid stability as we now have with k8s.gcr.io

## Proposal

There are two intertwined concepts that are part of this proposal.

First, the policy and procedures to promote/upload our container images to multiple providers. Our existing processes upload only to GCS buckets. Ideally we extend the existing software/promotion process to push directly to multiple providers. Alternatively we use a second process to synchronize container images from our existing production buckets to similar constructs at other providers.

Additionally we require a registry and artifact url-redirection solution to the local cloud provider or country.

## What exactly are you doing?

- We are setting up an AWS account with an IAM role and s3 buckets in AWS regions where we see a large percentage of source image pull traffic
- We will iterate on a sandbox url (registry-sandbox.k8s.io) for our experiments and ONLY promote things to (registry.k8s.io) when we have complete confidence
- both registry and registry-sandbox are serving traffic using oci-proxy on google cloud run
- oci-proxy will be updated to identify incoming traffic from AWS regions based on IP ranges so we can route traffic to s3 buckets in that region. If a specific AWS region do not currently host s3 buckets, we will redirect to the nearest region which does have s3 buckets (tradeoff between storage and network costs)
- We will bulk sync existing image layers to these s3 layers as a starting point (from GCS/GCR)
- We will update image-promoter to push to these s3 buckets as well in addition to the current setup
- We will set up monitoring/reporting to check on new costs we incur on the AWS infrastructure and update what we do in GCP infrastructure as well to include the new components
- We will have a plan in place on how we could add additional AWS regions in the future
- We will have CI jobs that will run against registry-sandbox.k8s.io as well to monitor stability before we promote code to registry
- We will automate the deployment/monitoring and testing of code landing in the oci-proxy repository

### User Stories

#### SIG Release - Image Promotion

```feature
Scenario: images are promoted
  As a SIG Release volunteer
  I want to promote our binaries/images to multiple clouds

Given a promotion / manifest
When my PR is merged
Then the promotion process occurs
```

#### Cloud Customer - pulling an official container image

```feature
Scenario: use Kubernetes container images
  I want to be able to pull and use Kubernetes container images

  Given some compute resources at cloud
  When I pull an official Kubernetes container image from registry.k8s.io
  Then I am redirected to a close-by cloud provider backed bucket (set) / CDN otherwise fall back to k8s.gcr.io
```

### Notes/Constraints/Caveats

The primary purpose of the KEP is getting consensus on the agreed policy and procedure to unblock our community and move forward together.

There has been a lot of activity around the technology and tooling for both goals, but we need shared agreement on policy and procedure first.

### Risks and Mitigations

This is the primary pipeline for delivering Kubernetes worldwide. Ensuring the appropriate SLAs and support as well as artifact integrity is crucial.

## Design Details

### Release Promotion

#### Policy

(more details needed, #sig-release-eng?)

#### Process

Currently the promotion process is primarily driven by the CIP/[promo-tool#kpromo](https://github.com/kubernetes-sigs/promo-tools#kpromo)?

### Artifact Distribution

#### Policy

#### Process

Container images will be written to S3 style storage or CDNs provided by cloud providers through a tool in the promo-tools suite.

## Alternatives / Background

- Original KEP
  - https://github.com/kubernetes/enhancements/tree/master/keps/sig-release/1734-k8s-image-promoter
- Oras
  - https://github.com/oras-project/oras
- KubeCon Talk
  - https://www.youtube.com/watch?v=F2IFjz7sr9Q
- Apache has a widespread mirror network
  - @dims has experince here
  - http://ws.apache.org/mirrors.cgi
  - https://infra.apache.org/mirrors.html
- [Umbrella issue: k8s.gcr.io => registry.k8s.io solution k/k8s.io#1834
  ](https://github.com/kubernetes/k8s.io/issues/1834)
- [ii/registry.k8s.io Implementation proposals](https://github.com/ii/registry.k8s.io#registryk8sio)
- [ii.nz/blog :: Building a data pipline for displaying Kubernetes public artifact traffic
  ](https://ii.nz/post/building-a-data-pipline-for-displaying-kubernetes-public-artifact-traffic/)

### How much is this going to save us?

Cost of K8s Artifact hosting - Data Studio Graphs

![](https://i.imgur.com/LAn4UIE.png)

## Infrastructure Needed

It would be good to request some donations for some larger providers, including one in China, via cncf.io/credits

## Hack this doc

- [![hackmd-github-sync-badge](https://hackmd.io/KjHufZssQR654ShkZFUzyA/badge)](https://hackmd.io/KjHufZssQR654ShkZFUzyA)
- [kubernetes/enhancements!3079](https://github.com/kubernetes/enhancements/pull/3079)
