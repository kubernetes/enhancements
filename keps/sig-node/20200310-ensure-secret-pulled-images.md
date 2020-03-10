---
title: Ensure Secret Pulled Images
authors:
  - "@mikebrow"
owning-sig: sig-node
participating-sigs:
  - sig-node
reviewers:
  - "@Random-Liu"
  - "@yujuhong"
approvers:
  - "@dchen1107"
editor: N/A
creation-date: 2020-03-10
last-updated: 2020-03-10
status: provisional|implementable|implemented|deferred|rejected|withdrawn|replaced
see-also:
  - N/A
replaces:
  - N/A
superseded-by:
  - N/A
---

# Ensure Secret Pulled Images

## Table of Contents

<!-- toc -->
- [Release Signoff Checklist](#release-signoff-checklist)
- [Summary](#summary)
- [Motivation](#motivation)
  - [Goals](#goals)
  - [Non-Goals](#non-goals)
- [Proposal](#proposal)
  - [User Stories](#user-stories)
  - [Risks and Mitigations](#risks-and-mitigations)
- [Design Details](#design-details)
  - [Test Plan](#test-plan)
  - [Graduation Criteria](#graduation-criteria)
    - [Examples](#examples)
      - [Alpha -&gt; Beta Graduation](#alpha---beta-graduation)
      - [Beta -&gt; GA Graduation](#beta---ga-graduation)
- [Implementation History](#implementation-history)
- [Drawbacks [optional]](#drawbacks-optional)
- [Alternatives [optional]](#alternatives-optional)
- [Infrastructure Needed [optional]](#infrastructure-needed-optional)
<!-- /toc -->

## Release Signoff Checklist

- [ ] kubernetes/enhancements issue in release milestone, which links to KEP (this should be a link to the KEP location in kubernetes/enhancements, not the initial KEP PR)
- [ ] KEP approvers have set the KEP status to `implementable`
- [ ] Design details are appropriately documented
- [ ] Test plan is in place, giving consideration to SIG Architecture and SIG Testing input
- [ ] Graduation criteria is in place
- [ ] "Implementation History" section is up-to-date for milestone
- [ ] User-facing documentation has been created in [kubernetes/website], for publication to [kubernetes.io]
- [ ] Supporting documentation e.g., additional design documents, links to mailing list discussions/SIG meetings, relevant PRs/issues, release notes

## Summary

We will add support for ensuring images pulled with pod imagePullSecrets are
always authenticated even if cached. We will add a new boolean field
`ensureSecretPulledImages` to the pod spec. The default to false
means that if a first pod results in an image pulled with imagePullSecrets a
second pod would have to be using always pull to ensure rights to use the
previously pulled image. When set to true always pull would not be required,
instead kublet will check if the image was pulled with an image pull secret and
if so would force a pull of the image to ensure the image pulled with the
secret is not used by another pod unless that pod also has the proper auth.

## Motivation

There have been customer requests for improving upon kubernetes ability to
secure images pulled with auth. on a node. Issue
[#18787](https://github.com/kubernetes/kubernetes/issues/18787) has been around
for a while.

To secure images one currently needs to inject `AllwaysPullImages` into pod
specs via an admission plugin. As @liggitt [notes](https://github.com/kubernetes/kubernetes/issues/18787#issuecomment-532280931)
the `pull` does not re-pull already-pulled layers of the image, but simply
resolves/verifies the image manifest has not changed in the registry (which
incidentally requires authenticating to private registries, which enforces the
image access). That means in the normal case (where the image has not changed
since the last pull), the request size is O(kb). However, the `pull` does put
the registry in the critical path of starting a container, since an unavailable
registry will fail the pull image manifest check (with or without proper
authentication.)


### Goals

Add a flag processed by `kubelet` for `ensureSecretPulledImages` (or something
similarly named) that, if true, would force `kubelet` to attempt to pull every
image that was pulled with image pulled secret based authentication, regardless
of the container image's pull policy.

Optimize to only force re-authentication for a pod when the secret used to pull
the container image is not present.

### Non-Goals

Out of scope for this KEP is an image caching policy that would direct container
runtimes through the CRI wrt. how they should treat the caching of images on a
node. Such as store for public use but only if encrypted. Or Store for private
use unencrypted...

## Proposal

When `ensureSecretPulledImages` is set, `kublet` will check keep a list of
container images that required authentication. `kublet` will ensure any image
in the list is always pulled thus enforcing authentication / re-authentication
with the exception of pods with secrets containing an auth that has been
authenticated.

### User Stories
wip

### Risks and Mitigations

With the default being false, devops engineers may not know to set the flag to
true in new/old pod specs that are using secrets for pull authentication with
registries.

A mitigation would be an admission plugin to inject `ensureSecretPulledImages.`

Images authentications with a registry may expire. To mitigate expirations a
a timeout could be used to force re-authentication.

## Design Details

### Test Plan

tbd

### Graduation Criteria

tbd

#### Examples

These are generalized examples to consider, in addition to the aforementioned [maturity levels][maturity-levels].

##### Alpha -> Beta Graduation

tbd

##### Beta -> GA Graduation

tbd

## Implementation History

tbd

## Drawbacks [optional]

Why should this KEP _not_ be implemented. N/A

## Alternatives [optional]

Default the ensure secrets rule to true and don't introduce a new pod spec flag.
Instead of a pod spec flag make the option a kublet configuration switch or
set the flag at some other scope.

## Infrastructure Needed [optional]

tbd
