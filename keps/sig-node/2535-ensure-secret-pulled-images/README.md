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
      - [Integration tests](#integration-tests)
      - [e2e tests](#e2e-tests)
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

We will add support in kubelet for the `pullIfNotPresent` image pull policy, for
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

*** The issue and these changes improving the security posture without requiring the forcing of pull always, will be documented in the kubernetes image pull policy documentation. The new feature gate should also be documented in release notes. ***

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
ensure the images pulled with a secret by `kubelet` since boot. During the
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

For alpha `kubelet` will keep a list, across reboots of host and restart of
kubelet, of container images that required authentication and a list of the
authentications that successfully pulled the image.
For beta an API will be considered to manage the ensure metadata.

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
a timeout will be used to force re-authentication. The timeout could be a
container runtime feature or a `kubelet` feature. If at the container runtime,
images would not be present during the EnsureImagesExist step, thus would have
to be pulled and authenticated if necessary. This timeout feature will be
implemented in alpha.

Since images can be pre-loaded, loaded outside the `kubelet` process, and
garbage collected.. the list of images that required authentication in `kubelet`
will not be a source of truth for how all images were pulled that are in the
container runtime cache. To mitigate, images can be garbage collected at boot.
And we will persist ensure metadata across reboot of host, and restart
of kubelet, and possibly look at a way to add ensure metadata for images loaded
outside of kubelet. In beta we will add a switch to enable re-auth on boot for
admins seeking that instead of having to garbage collect where they do not use
or expect preloaded images since boot.


## Design Details

Kubelet will track, in memory, a hash map for the credentials that were successfully used to pull an image. It has been decided that the hash map will be persisted to disk, in alpha.

The persisted "cache" will undergo cleanup operations on a timely basis (by default once an hour).

The persistence of the on storage cache is mainly for restarting kubelet and/or node reboot.

The max size of the cache will scale with the number of unique cache entries * the number of unique images that have not been garbage collected. It is not expected that this will be a significant number of bytes. Will be verified by actual use in Alpha and subsequent metrics in Beta.

See `/var/lib/kubelet/image_manager_state` in [kubernetes/kubernetes#114847](https://github.com/kubernetes/kubernetes/pull/114847)

> ```
> {
>   "images": {
>     "sha256:eb6cbbefef909d52f4b2b29f8972bbb6d86fc9dba6528e65aad4f119ce469f7a": {
>       "authHash": { ** per review comment use SHA256 here vs hash **
>         "115b8808c3e7f073": {
>           "ensured": true,
>           "dueDate": "2023-05-30T05:26:53.76740982+08:00"
>         }
>       },
>       "name": "daocloud.io/daocloud/dce-registry-tool:3.0.8"
>     }
>   }
> }
> ```

See PR linked above for detailed design / behavior documentation.

### Test Plan

[x] I/we understand the owners of the involved components may require updates to
existing tests to make this code solid enough prior to committing the changes
necessary to implement this enhancement.

##### Prerequisite testing updates


##### Unit tests

For alpha, exhaustive Kubelet unit tests will be provided. Functions affected by the feature gate will be run with the feature gate on and with the feature gate off. Unit buckets will be provided for:
- HashAuth - (new, small) returns a hash code for a CRI pull image auth [link](https://github.com/kubernetes/kubernetes/pull/94899/files#diff-ca08601dfd2fdf846f066d0338dc332beddd5602ab3a71b8fac95b419842da63R704-R751) ** per review comment will use SHA256 **
- shouldPullImage - (modified, large sized change) determines if image should be pulled based on presence, and image pull policy, and now with the feature gate on if the image has been pulled/ensured by a secret. A unit test bucket did not exist for this function. The unit bucket will cover a matrix for:
```
	pullIfNotPresent := &v1.Container{
    ..
   	ImagePullPolicy: v1.PullIfNotPresent,
 	}
 	pullNever := &v1.Container{
    ..
    ImagePullPolicy: v1.PullNever,
 	}
 	pullAlways := &v1.Container{
 		..
    ImagePullPolicy: v1.PullAlways,
 	}
 	tests := []struct {
 		description       string
 		container         *v1.Container
 		imagePresent      bool
 		pulledBySecret    bool
 		ensuredBySecret   bool
 		expectedWithFGOff bool
 		expectedWithFGOn  bool
 	}
```
[TestShouldPullImage link](https://github.com/kubernetes/kubernetes/pull/94899/files#diff-7297f08c72da9bf6479e80c03b45e24ea92ccb11c0031549e51b51f88a91f813R311-R438)

PersistHashMeta() ** will be persisting SHA256 entries vs hash **

Additionally, for Alpha we will update this readme with an enumeration of the core packages being touched by the PR to implement this enhancement and provide the current unit coverage for those in the form of:
- <package>: <date> - <current test coverage>
The data will be read from:
https://testgrid.k8s.io/sig-testing-canaries#ci-kubernetes-coverage-unit

##### Integration tests

At beta we will revisit if integration buckets are warranted for cri-tools/critest, and after gathering feedback.

<!--
Integration tests are contained in k8s.io/kubernetes/test/integration.
Integration tests allow control of the configuration parameters used to start the binaries under test.
This is different from e2e tests which do not allow configuration of parameters.
Doing this allows testing non-default options and multiple different and potentially conflicting command line options.
-->

<!--
This question should be filled when targeting a release.
For Alpha, describe what tests will be added to ensure proper quality of the enhancement.

For Beta and GA, add links to added tests together with links to k8s-triage for those tests:
https://storage.googleapis.com/k8s-triage/index.html
-->

- <test>: <link to test coverage> (TBD)

##### e2e tests

<!--
This question should be filled when targeting a release.
For Alpha, describe what tests will be added to ensure proper quality of the enhancement.

For Beta and GA, add links to added tests together with links to k8s-triage for those tests:
https://storage.googleapis.com/k8s-triage/index.html

We expect no non-infra related flakes in the last month as a GA graduation criteria.
-->
At beta we will revisit if e2e buckets are warranted for e2e node, and after gathering feedback.

- <test>: <link to test coverage> (TBD)

### Graduation Criteria

#### Alpha

- Feature implemented behind a feature flag - KubeletEnsureSecretPulledImages
- Initial e2e tests completed and enabled - No additional e2e identified as yet

#### Deprecation

N/A in alpha
TBD subsequent to alpha

### Upgrade / Downgrade Strategy

### Version Skew Strategy

N/A for alpha
TBD subsequent to alpha

## Production Readiness Review Questionnaire

### Feature Enablement and Rollback
- At Alpha this feature will be disabled by default with a feature gate.
- At Beta this feature will be enabled by default with the feature gate.
- At GA the ability to gate the feature will be removed leaving the feature enabled.

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
TBD

###### How can a rollout or rollback fail? Can it impact already running workloads?

TBD

###### What specific metrics should inform a rollback?

TBD needed for Beta

###### Were upgrade and rollback tested? Was the upgrade->downgrade->upgrade path tested?

TBD

###### Is the rollout accompanied by any deprecations and/or removals of features, APIs, fields of API types, flags, etc.?

TBD

### Monitoring Requirements

TBD

###### How can an operator determine if the feature is in use by workloads?

For alpha can check if images pulled with credentials by a first pod, are also pulled with credentials by a second pod that is
using the pull if not present image pull policy. Will show up as network events. Though only the manifests will be
revalidated against the container image repository, large contents will not be pulled. Thus one could monitor traffic
to the registry.

For beta will add metrics allowing an admin to determine how often an image has been reauthenticated to an image registry because of cache expiration or due to reuse across pods that have different authentication information. Success metrics will also be provided highlighting cache hits.

###### How can someone using this feature know that it is working for their instance?

Can test for an image pull failure event coming from a second pod that does not have credentials to pull the image
where the image is present and the image pull policy is if not present.

- [x] Events
  - Event Reason: "kubelet  Failed to pull image" ... "unexpected status code [manifests ...]: 401 Unauthorized"


###### What are the reasonable SLOs (Service Level Objectives) for the enhancement?

TBD

###### What are the SLIs (Service Level Indicators) an operator can use to determine the health of the service?

TBD

###### Are there any missing metrics that would be useful to have to improve observability of this feature?

TBD needed for Beta

### Dependencies

TBD

###### Does this feature depend on any specific services running in the cluster?

No.

### Scalability

TBD

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

TBD

###### How does this feature react if the API server and/or etcd is unavailable?

TBD

###### What are other known failure modes?

TBD

###### What steps should be taken if SLOs are not being met to determine the problem?

Check logs.

## Implementation History

TBD

## Drawbacks [optional]

Why should this KEP _not_ be implemented. TBD

## Alternatives [optional]

- Make the behavior change enabled by default by changing the feature gate to true by default instead of false by default.
- Discussions went back and forth on whether this should go directly to GA as a fix or alpha as a feature gate. It seems this should be the default security posture for pullIfNotPresent as it is not clear to admins/users that an image pulled by a first pod with authentication can be used by a second pod without authentication. The performance cost should be minimal as only the manifest needs to be re-authenticated. But after further review and discussion with MrunalP we'll go ahead and have a kubelet feature gate with default off for alpha in v1.23.
- Set the flag at some other scope e.g. pod spec (doing it at the pod spec was rejected by SIG-Node).
- For beta/ga we may revisit/replace the in memory hash map in kubelet design, with an extension to the CRI API for having the container runtime
ensure the image instead of kubelet.

## Infrastructure Needed [optional]

TBD
