# KEP-2535: Ensure Secret Pulled Images

## Table of Contents

<!-- toc -->
- [Release Signoff Checklist](#release-signoff-checklist)
- [Summary](#summary)
- [Motivation](#motivation)
  - [Goals](#goals)
  - [Non-Goals](#non-goals)
- [Proposal](#proposal)
  - [User Stories](#user-stories)
    - [Story 1](#story-1)
    - [Story 2](#story-2)
  - [Notes/Constraints/Caveats (Optional)](#notesconstraintscaveats-optional)
  - [Risks and Mitigations](#risks-and-mitigations)
- [Design Details](#design-details)
  - [Test Plan](#test-plan)
  - [Graduation Criteria](#graduation-criteria)
    - [Alpha](#alpha)
    - [Deprecation](#deprecation)
  - [Upgrade / Downgrade Strategy](#upgrade--downgrade-strategy)
  - [Version Skew Strategy](#version-skew-strategy)
- [Production Readiness Review Questionnaire](#production-readiness-review-questionnaire)
  - [Feature Enablement and Rollback](#feature-enablement-and-rollback)
  - [Rollout, Upgrade and Rollback Planning](#rollout-upgrade-and-rollback-planning)
  - [Monitoring Requirements](#monitoring-requirements)
  - [Dependencies](#dependencies)
  - [Scalability](#scalability)
  - [Troubleshooting](#troubleshooting)
- [Implementation History](#implementation-history)
- [Drawbacks [optional]](#drawbacks-optional)
- [Alternatives [optional]](#alternatives-optional)
- [Infrastructure Needed [optional]](#infrastructure-needed-optional)
<!-- /toc -->

## Release Signoff Checklist

Items marked with (R) are required *prior to targeting to a milestone / release*.

- [ ] (R) Enhancement issue in release milestone, which links to KEP dir in [kubernetes/enhancements] (not the initial KEP PR)
- [ ] (R) KEP approvers have approved the KEP status as `implementable`
- [ ] (R) Design details are appropriately documented
- [ ] (R) Test plan is in place, giving consideration to SIG Architecture and SIG Testing input (including test refactors)
  - [ ] e2e Tests for all Beta API Operations (endpoints)
  - [ ] (R) Ensure GA e2e tests for meet requirements for [Conformance Tests](https://github.com/kubernetes/community/blob/master/contributors/devel/sig-architecture/conformance-tests.md)
  - [ ] (R) Minimum Two Week Window for GA e2e tests to prove flake free
- [ ] (R) Graduation criteria is in place
  - [ ] (R) [all GA Endpoints](https://github.com/kubernetes/community/pull/1806) must be hit by [Conformance Tests](https://github.com/kubernetes/community/blob/master/contributors/devel/sig-architecture/conformance-tests.md)
- [ ] (R) Production readiness review completed
- [ ] (R) Production readiness review approved
- [ ] "Implementation History" section is up-to-date for milestone
- [ ] User-facing documentation has been created in [kubernetes/website], for publication to [kubernetes.io]
- [ ] Supporting documentation—e.g., additional design documents, links to mailing list discussions/SIG meetings, relevant PRs/issues, release notes


[kubernetes.io]: https://kubernetes.io/
[kubernetes/enhancements]: https://git.k8s.io/enhancements
[kubernetes/kubernetes]: https://git.k8s.io/kubernetes
[kubernetes/website]: https://git.k8s.io/website


## Summary

We will add support in kubelet for the pullIfNotPresent image pull policy, for
ensuring images pulled with pod imagePullSecrets are re-authenticated for other
pods that do not have the same imagePullSecret/auths used to successfully pull
the images in the first place.

This policy change will have no affect on the `pull always` image pull
policy or for images that are preloaded.

However, for the `pull never` policy if a first pod successfully pulled an image
with credential and then a second pod with pull never tried to use the image,
when the feature gate is on the second pod will receive an error message, where
before and with the feature gate off the second pod would be able to use the image
pulled with credentials by the first pod.

This new feature will be enabled with a feature gate in alpha. This feature
improves the security posture for privacy/security of image contents by forcing
images pulled with an imagePullSecret/auth of a first pod to be re-authenticated
for a second pod even if the image is already present through the secure pull of
the first pod.

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

#### Story 1
User with multiple tenants will be able to support all image pull policies without
concern that one tenant will gain access to an image that they don't have rights to.

#### Story 2
User will will no longer have to inject the Pull Always Image Pull Policy to
ensure all tenants have rights to the images that are already present on a host.


### Notes/Constraints/Caveats (Optional)

With the default of the feature gate being off, users / cloud providers will have
to set the feature gate to true to gain these this Secure by Default benefit.

### Risks and Mitigations

Image authentications with a registry may expire. To mitigate expirations a
a timeout could be used to force re-authentication. The timeout could be a
container runtime feature or a `kubelet` feature. If at the container runtime,
images would not be present during the EnsureImagesExist step, thus would have
to be pulled and authenticated if necessary.

Since images can be pre-loaded, loaded outside the `kubelet` process, and
garbage collected.. the list of images that required authentication in `kubelet`
will not be a source of truth for how all images were pulled that are in the
container runtime cache. To mitigate, images can be garbage collected at boot.


## Design Details

Kubelet will track, in memory, a hash map for the credentials that were successfully used to pull an image. The hash map
will not be persisted to disk, in alpha. For alpha explicitly, we will not reuse or add other state manager concepts to kubelet.

See PR for detailed design / behavior documentation.

### Test Plan

See PR (exhaustive unit tests added for alpha covering feature gate on and off for new and modified functions)

### Graduation Criteria

#### Alpha

- Feature implemented behind a feature flag - KubeletEnsureSecretPulledImages
- Initial e2e tests completed and enabled - No additional e2e identified as yet

#### Deprecation

N/A in alpha

### Upgrade / Downgrade Strategy

### Version Skew Strategy

N/A for alpha

## Production Readiness Review Questionnaire

### Feature Enablement and Rollback

###### How can this feature be enabled / disabled in a live cluster?

- [x] Feature gate (also fill in values in `kep.yaml`)
  - Feature gate name: KubeletEnsureSecretPulledImages
  - Components depending on the feature gate: kubelet


###### Does enabling the feature change any default behavior?

Yes, see discussions above.

###### Can the feature be disabled once it has been enabled (i.e. can we roll back the enablement)?

Yes.

###### What happens if we reenable the feature if it was previously rolled back?

Will go back to working as designed.

###### Are there any tests for feature enablement/disablement?

Yes, tests run both enabled and disabled.

### Rollout, Upgrade and Rollback Planning
N/A

###### How can a rollout or rollback fail? Can it impact already running workloads?

N/A

###### What specific metrics should inform a rollback?

N/A

###### Were upgrade and rollback tested? Was the upgrade->downgrade->upgrade path tested?

N/A

###### Is the rollout accompanied by any deprecations and/or removals of features, APIs, fields of API types, flags, etc.?

N/A

### Monitoring Requirements

N/A

###### How can an operator determine if the feature is in use by workloads?

Can check if images pulled with credentials by a first pod, are also pulled with credentials by a second pod that is
using the pull if not present image pull policy. Will show up as network events. Though only the manifests will be
revalidated against the container image repository, large contents will not be pulled. Thus one could monitor traffic
to the registry.

###### How can someone using this feature know that it is working for their instance?

Can test for an image pull failure event coming from a second pod that does not have credentials to pull the image
where the image is present and the image pull policy is if not present.

- [x] Events
  - Event Reason: "kubelet  Failed to pull image" ... "unexpected status code [manifests ...]: 401 Unauthorized"


###### What are the reasonable SLOs (Service Level Objectives) for the enhancement?

N/A

###### What are the SLIs (Service Level Indicators) an operator can use to determine the health of the service?

N/A

###### Are there any missing metrics that would be useful to have to improve observability of this feature?

N/A

### Dependencies

N/A for alpha

###### Does this feature depend on any specific services running in the cluster?

No.

### Scalability

N/A

###### Will enabling / using this feature result in any new API calls?

No.

###### Will enabling / using this feature result in introducing new API types?

No.

###### Will enabling / using this feature result in any new calls to the cloud provider?

No.

###### Will enabling / using this feature result in increasing size or count of the existing API objects?

Yes. When enabled, and when container images have been pulled with image pull secrets (credentials), subsequent image
pulls for pods that do not contain the image pull secret that successfully pulled the image will have to authenticate
by trying to pull the image manifests from the registry. The image layers do not have to be re-pulled, just the
manifests for authentication purposes.

###### Will enabling / using this feature result in increasing time taken by any operations covered by existing SLIs/SLOs?

When switched on see above.

###### Will enabling / using this feature result in non-negligible increase of resource usage (CPU, RAM, disk, IO, ...) in any components?

When switched on see above.

### Troubleshooting

N/A

###### How does this feature react if the API server and/or etcd is unavailable?

N/A

###### What are other known failure modes?

N/A

###### What steps should be taken if SLOs are not being met to determine the problem?

Check logs.

## Implementation History

tbd

## Drawbacks [optional]

Why should this KEP _not_ be implemented. N/A

## Alternatives [optional]

- Make the behavior change enabled by default by changing the feature gate to true by default instead of false by default.
- Discussions went back and forth on whether this should go directly to GA as a fix or alpha as a feature gate. It seems this should be the default security posture for pullIfNotPresent as it is not clear to admins/users that an image pulled by a first pod with authentication can be used by a second pod without authentication. The performance cost should be minimal as only the manifest needs to be re-authenticated. But after further review and discussion with MrunalP we'll go ahead and have a kubelet feature gate with default off for alpha in v1.22.
- Set the flag at some other scope e.g. pod spec (doing it at the pod spec was rejected by SIG-Node).
- For beta/ga we may revisit/replace the in memory hash map in kubelet design, with an extension to the CRI API for having the container runtime
ensure the image instead of kubelet.

## Infrastructure Needed [optional]

tbd
