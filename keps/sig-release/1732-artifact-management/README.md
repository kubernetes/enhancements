# Kubernetes Artifact Management

<!-- toc -->
- [Summary](#summary)
- [Motivation](#motivation)
  - [Goals](#goals)
  - [Non-Goals](#non-goals)
- [Proposal](#proposal)
  - [HTTP Redirector Design](#http-redirector-design)
    - [Configuring the HTTP Redirector](#configuring-the-http-redirector)
- [Graduation Criteria](#graduation-criteria)
- [Implementation History](#implementation-history)
  - [Milestone 0 (MVP): In progress](#milestone-0-mvp-in-progress)
<!-- /toc -->

## Summary
This document describes how official artifacts (Container Images, Binaries) for the Kubernetes
project are managed and distributed.


## Motivation

The motivation for this KEP is to describe a process by which artifacts (container images, binaries)
can be distributed by the community. Currently the process by which images is both ad-hoc in nature
and limited to an arbitrary set of people who have the keys to the relevant repositories. Standardize
access will ensure that people around the world have access to the same artifacts by the same names
and that anyone in the project is capable (if given the right authority) to distribute images.

### Goals

The goals of this process are to enable:
  * Anyone in the community (with the right permissions) to manage the distribution of Kubernetes images and binaries.
  * Fast, cost-efficient access to artifacts around the world through appropriate mirrors and distribution

This KEP will have succeeded when artifacts are all managed in the same manner and anyone in the community
(with the right permissions) can manage these artifacts.

### Non-Goals

The actual process and tooling for promoting images, building packages or otherwise assembling artifacts
is beyond the scope of this KEP. This KEP deals with the infrastructure for serving these things via
HTTP as well as a generic description of how promotion will be accomplished.

## Proposal

The top level design will be to set up a global redirector HTTP service (`artifacts.k8s.io`) 
which knows how to serve HTTP and redirect requests to an appropriate mirror. This redirector
will serve both binary and container image downloads. For container images, the HTTP redirector
will redirect users to the appropriate geo-located container registry. For binary artifacts, 
the HTTP redirector will redirect to appropriate geo-located storage buckets.

To facilitate artifact promotion, each project, as necessary, will be given access to a
project staging area relevant to their particular artifacts (either storage bucket or image 
registry). Each project is free to manage their assets in the staging area however they feel
it is best to do so. However, end-users are not expected to access artifacts through the
staging area.

For each artifact, there will be a configuration file checked into this repository. When a
project wants to promote an image, they will file a PR in this repository to update their
image promotion configuration to promote an artifact from staging to production. Once this
PR is approved, automation that is running in the k8s project infrastructure (e.g. 
https://github.com/GoogleCloudPlatform/k8s-container-image-promoter) will pick up this new
configuration file and copy the relevant bits out to the production serving locations.

Importantly, if a project needs to roll-back or remove an artifact, the same process will
occur, so that the promotion tool needs to be capable of deleting images and artifacts as
well as promoting them.

### HTTP Redirector Design
To facilitate world-wide distribution of artifacts from a single (virtual) location we will
ideally run a replicated redirector service in the United States, Europe and Asia.
Each of these redirectors
services will be deployed in a Kubernetes cluster and they will be exposed via a public IP
address and a dns record indicating their location (e.g. `europe.artifacts.k8s.io`).

We will use Geo DNS to route requests to `artifacts.k8s.io` to the correct redirector. This is necessary to ensure that we always route to a server which is accessible no matter what region we are in. We will need to extend or enhance the existing DNS synchronization tooling to handle creation of the GeoDNS records.

#### Configuring the HTTP Redirector
THe HTTP Redirector service will be driven from a YAML configuration that specifies a path to mirror
mapping. For now the redirector will serve content based on continent, for example:

```yaml
/kops
  - Americas: americas.artifacts.k8s.io
  - Asia: asia.artifacts.k8s.io
  - default: americas.artificats.k8s.io
```

The redirector will use this data to redirect a request to the relevant mirror using HTTP 302 responses. The implementation of the mirrors themselves are details left to the service implementor and may be different depending on the artifacts being exposed (binaries vs. container images)

## Graduation Criteria

This KEP will graduate when the process is implemented and has been successfully used to
manage the images for a Kubernetes release.

## Implementation History

### Milestone 0 (MVP): In progress

(Described in terms of kops, our first candidate; other candidates welcome!)

* k8s-infra creates a "staging" GCS bucket for each project
  (e.g. `k8s-artifacts-staging-<project>`) and a "prod" GCS bucket for promoted
  artifacts (e.g. `k8s-artifacts`, one bucket for all projects).
* We grant write-access to the staging GCS bucket to trusted jobs / people in
  each project (e.g. kops OWNERS and prow jobs can push to
  `k8s-artifacts-staging-kops`).  We can encourage use of CI & reproducible
  builds, but we do not block on it.
* We grant write-access to the prod bucket only to the infra-admins & the
  promoter process.
* Promotion of artifacts to the "prod" GCS bucket is via a script / utility (as
  we do today).  For v1 we can promote based on a sha256sum file (only copy the
  files listed), similarly to the image promoter.  We will experiment to develop
  that script / utility in this milestone, along with prow jobs (?) to publish
  to the staging buckets, and to figure out how best to run the promoter.
  Hopefully we can copy the image-promotion work closely.
* We create a bucket-backed GCLB for serving, with a single url-map entry for
  `binaries/` pointing to the prod bucket.  (The URL prefix gives us some
  flexibility to e.g. add dynamic content later)
* We create the artifacts.k8s.io DNS name pointing to the GCLB. (Unclear whether
  we want one for staging, or just encourage pulling from GCS directly).
* Projects start using the mirrors e.g. kops adds the
  https://artifacts.k8s.io/binaries/kops mirror into the (upcoming) mirror-list
  support, so that it will get real traffic but not break kops should this
  infrastructure break
* We start to collect data from the GCLB logs.  Questions we would like to
  understand: What are the costs, and what would the costs be for localized
  mirrors?  What is the performance impact (latency, throughput) of serving
  everything from GCLB?  Is GCLB reachable from everywhere (including China)?
  Can we support private mirrors (i.e. non-coordinated mirrors)?
