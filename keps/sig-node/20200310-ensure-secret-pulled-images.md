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
last-updated: 2020-08-25
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

We will add support in kubelet for the pullIfNotPresent image pull policy, for
ensuring images pulled with pod imagePullSecrets are re-authenticated for other
pods that do not have the same imagePullSecret/auths used to successfully pull
the images in the first place.  

This policy will have no affect on the `pull never` and `pull always` image pull
policies or for images that are preloaded.

This new feature will be enabled by default. This feature improves the security
posture for privacy/security of image contents by forcing images pulled with an
imagePullSecret/auth of a first pod to be re-authenticated for a second pod even
if the image is already present through the secure pull of the first pod.

The new behavior means that if a first pod results in an image pulled with
imagePullSecrets a second pod would have to also have rights to the image in
order to use a present image.

This means that the image pull policy alwaysPull would no longer be required in
every scenario to ensure image access rights by pods.

## Motivation

There have been customer requests for improving upon kubernetes' ability to
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

Modify the current pullIfNotPresent policy management enforced by `kubelet` to
ensure the images pulled with a secret by `kublet` since boot. During the
EnsureImagesExist step `kubelet` will require authentication of present images
pulled with auth since boot.  

Optimize to only force re-authentication for a pod container image when the
secret used to pull the container image is not present. IOW if an image is
pulled with authentication for a first pod, subsequent pods that have the same
authentication information should not need to re-authenticate.

Images already present at boot or loaded externally to `kubelet` or successfully
pulled through `kubelet` with no imagePullSecret/authentication required will
not require authentication.

### Non-Goals

Out of scope for this KEP is an image caching policy that would direct container
runtimes through the CRI wrt. how they should treat the caching of images on a
node. Such as store for public use but only if encrypted. Or Store for private
use un-encrypted...

## Proposal

`kubelet` will keep a list, since boot, of container images that required
authentication and a list of the authentications that successfully pulled the image.

`kubelet` will ensure any image in the list is always pulled if an authentication
used is not present, thus enforcing authentication / re-authentication.


### User Stories
wip

### Risks and Mitigations

Image authentications with a registry may expire. To mitigate expirations a
a timeout could be used to force re-authentication. The timeout could be a
container runtime feature or a `kubelet` feature. If at the container runtime,
images would not be present during the EnsureImagesExist step, thus would have
to be pulled and authenticated if necessary.  

Since images can be pre-loaded, loaded outside the `kubelet` process, and
garbage collected.. the list of images that required authentication in `kubelet`
will not be a source of truth for how all images were pulled that are in the
container runtime cache. To mitigate images can be garbage collected at boot.

## Design Details

See PR.

### Test Plan

tbd

### Graduation Criteria

tbd

#### Examples

tbd

##### Alpha -> Beta Graduation

tbd

##### Beta -> GA Graduation

tbd

## Implementation History

tbd

## Drawbacks [optional]

Why should this KEP _not_ be implemented. N/A

## Alternatives [optional]

- Make the behavior change a `kubelet` configuration switch (This was the SIG-Node suggested option).
However after discussions it seems this should be the default security posture for pullIfNotPresent as it is not clear to admins/users that an image pulled by a first pod with authentication can be used by a second pod without authentication. The performance cost should be minimal as only the manifest needs to be re-authenticated.
- Set the flag at some other scope e.g. pod spec (doing it at the pod spec was rejected by SIG-Node).

## Infrastructure Needed [optional]

tbd
