---
kep-number: 34
title: Kubernetes Community Artifact Serving
authors:
  - "@brendandburns"
owning-sig: sig-contributor-experience
participating-sigs:
reviewers:
  - "@justinsb"
  - "@dims"
approvers:
  - "@dims"
editor: "@brendandburns"
creation-date: 2019-01-23
last-updated: 2019-01-23
status: provisional
see-also:
---

# Kubernetes Artifact Management

1. **Fill out the "overview" sections.**
  This includes the Summary and Motivation sections.
  These should be easy if you've preflighted the idea of the KEP with the appropriate SIG.
1. **Create a PR.**
  Assign it to folks in the SIG that are sponsoring this process.
1. **Merge early.**
  Avoid getting hung up on specific details and instead aim to get the goal of the KEP merged quickly.
  The best way to do this is to just start with the "Overview" sections and fill out details incrementally in follow on PRs.
  View anything marked as a `provisional` as a working document and subject to change.
  Aim for single topic PRs to keep discussions focused.
  If you disagree with what is already in a document, open a new PR with suggested changes.

The canonical place for the latest set of instructions (and the likely source of this file) is [here](/keps/0000-kep-template.md).

The `Metadata` section above is intended to support the creation of tooling around the KEP process.
This will be a YAML section that is fenced as a code block.
See the KEP process for details on each of these items.

## Table of Contents

A table of contents is helpful for quickly jumping to sections of a KEP and for highlighting any additional information provided beyond the standard KEP template.
[Tools for generating][] a table of contents from markdown are available.

* [Table of Contents](#table-of-contents)
* [Summary](#summary)
* [Motivation](#motivation)
    * [Goals](#goals)
    * [Non-Goals](#non-goals)
* [Proposal](#proposal)
    * [User Stories [optional]](#user-stories-optional)
      * [Story 1](#story-1)
      * [Story 2](#story-2)
    * [Implementation Details/Notes/Constraints [optional]](#implementation-detailsnotesconstraints-optional)
    * [Risks and Mitigations](#risks-and-mitigations)
* [Graduation Criteria](#graduation-criteria)
* [Implementation History](#implementation-history)
* [Drawbacks [optional]](#drawbacks-optional)
* [Alternatives [optional]](#alternatives-optional)

[Tools for generating]: https://github.com/ekalinin/github-markdown-toc

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

## Graduation Criteria

This KEP will graduate when the process is implemented and has been sucessfully used to
manage the images for a Kubernetes release.

## Implementation History

None yet.

