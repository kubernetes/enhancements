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
      - [Prerequisite testing updates](#prerequisite-testing-updates)
      - [Unit tests](#unit-tests)
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
[kubernetes/website]: https://git.k8s.io/website

## Summary

We will add support in the kubelet for an admin to enable the ability to ensure an image that is already present on a node because
a pod with `ImagePullSecrets` previously pulled it is reauthenticated when a new pod with different `ImagePullSecrets` attempts to use the same image,
when the `ImagePullPolicy` is `IfNotPresent`.

In other words: ensure the pull secrets are rechecked for each new set of credentials, and ensure a pod has access to those images.

For the `Never` policy, the behavior also must change. Otherwise, a user who wishes to use the image of another pod could just use `Never` and hope
another pod have pulled it. Functionally from a security standpoint, we must account for this.
Thus, `Never` `ImagePullPolicy` images will be allowed past the ensure image stage of the pod lifecyle if the image has previously been pulled
by an `IfNotPresent` pod successfully: either with no auth, or with the same auth as the `Never` policy. The image will continue to never be pulled
for this pod.

This will be enforced for both policies regardless of whether the image is already present when the kubelet starts. For an image to be allowed to be used,
the kubelet must be aware of its credentials.

This policy change will have no affect on the `Always` `ImagePullPolicy`.

This new feature will be enabled with a feature gate in alpha, as well as a kubelet configuration
field `pullImageSecretRecheck`. Another kubelet configuration field `pullImageSecretRecheckPeriod` will be added
to allow an admin to configure the recheck period. A recheck period may be used to periodically clean the cache, or ensure
expiring credentials are still valid.

*** The issue and these changes improving the security posture without requiring the forcing of pull always, will be documented in the kubernetes image pull policy documentation. The new feature gate should also be documented in release notes. ***

## Motivation

There have been customer requests for improving upon kubernetes' ability to
secure images pulled with auth on a node. Issue
[#18787](https://github.com/kubernetes/kubernetes/issues/18787) has been around
for a while.

To secure images one currently needs to inject `Always` `ImagePullPolicy` into pod
specs via an admission plugin. As @liggitt [notes](https://github.com/kubernetes/kubernetes/issues/18787#issuecomment-532280931)
the `pull` does not re-pull already-pulled layers of the image, but simply
resolves/verifies the image manifest has not changed in the registry (which
incidentally requires authenticating to private registries, which enforces the
image access). That means in the normal case (where the image has not changed
since the last pull), the request size is O(kb).

However, the `pull` does put the registry in the critical path of starting a container,
since an unavailable registry will fail the pull image manifest check (with or without proper authentication.)

Thus, the motivation is to allow users to ensure the kubelet requires an image pull auth check for each new set of credentials,
regardless of whether the image is already present on the node.

### Goals

Modify the current behavior of images with an `IfNotPresent` and `Never` `ImagePullPolicy` enforced by the kubelet to
ensure the images pulled with a secret by the kubelet are authenticated by the CRI implementation. During the
EnsureImagesExist step the kubelet will require authentication of present images pulled with auth since boot.

Optimize to only force re-authentication for a pod container image when the
`ImagePullSecrets` used to pull the container image has not already been authenticated.
IOW if an image is pulled with authentication for a first pod, subsequent pods that have the same
authentication information should not need to re-authenticate, unless the kubelet's `pullImageSecretRecheckPeriod` has passed.

Images already present at boot or loaded externally to the kubelet or successfully
pulled through the kubelet with no `ImagePullSecrets`/authentication required will
not require authentication.

### Non-Goals

Out of scope for this KEP is an image caching policy that would direct container
runtimes through the CRI wrt. how they should treat the caching of images on a
node. Such as store for public use but only if encrypted. Or Store for private
use un-encrypted...

This feature will not change the behavior of pod with `ImagePullPolicy` `Always`.

## Proposal

For alpha the kubelet will keep a list, across reboots of host and restart of
kubelet, of container images that required authentication and a list of the
authentications that successfully pulled the image.
For beta an API will be considered to manage the ensure metadata.

The kubelet will ensure any image in the list is always pulled if an authentication
used is not present, thus enforcing authentication / re-authentication.

There will be two different kubelet configuration options added, as well as a feature
gate to gate their use:
- `pullImageSecretRecheck`
    - A boolean that toggles this behavior. If `false`, the kubelet will fallback to the
      old behavior: only pull an image if it's not present.
- `pullImageSecretRecheckPeriod`
    - the period after which the kubelet's cache will be invalidated,
      thus causing rechecks for all `IfNotPresent` images that are recreated.
    - If set to `0s`, or `0`, but `pullImageSecretRecheck` is `true`, then
      the kubelet will never invalidate its cache, but will maintain one.


### User Stories

#### Story 1

User with multiple tenants will be able to support all image pull policies without
concern that one tenant will gain access to an image that they don't have rights to.

#### Story 2

User will no longer have to inject the `PullAlways` imagePullPolicy to
ensure all tenants have rights to the images that are already present on a host.

### Notes/Constraints/Caveats (Optional)

With the default of the feature gate being off, users / cloud providers will have
to set the feature gate to true to gain these this Secure by Default benefit.

### Risks and Mitigations

- Image authentications with a registry may expire.
  - To mitigate expirations a timeout will be used to force re-authentication.
    This timeout will be configured as a kubelet configuration field `pullImageSecretRecheckPeriod`.
    This timeout feature will be implemented in alpha.

- Images can be "pre-loaded", or pulled behind the kubelet's back before it starts.
  In this case, the kubelet is not managing the credentials for these images.
  - To mitigate, metadata will be persisted across reboot. The kubelet will compare previously
    cached credentials against the images that exist. On a new image pull, the kubelet will use
    its saved cache and revalidate as necessary.
    In other words: even if the images are already cached, if new images are present that have not
    previously been authenticated against a pods credentials, then the image will be revalidated.


## Design Details

The kubelet will track, in memory, a pulled image auth cache for the credentials that were successfully used to pull an image.
This cache will be persisted to disk, to allow nodes that are "disconnected", or unable to reach the registry to boot up, assuming
they have previously been able to access a registry and authenticated the images present.

The persisted cache will be cleaned up every `pullImageSecretRecheckPeriod`.

The max size of the cache will scale with the number of unique cache entries * the number of unique images that have not been garbage collected.
It is not expected that this will be a significant number of bytes. Will be verified by actual use in Alpha and subsequent metrics in Beta.

See `/var/lib/kubelet/image_manager_state` in [kubernetes/kubernetes#114847](https://github.com/kubernetes/kubernetes/pull/114847)

```
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
```

See PR linked above for detailed design / behavior documentation.

Note: using the tag `:latest` is equivalent to using the image pull policy `Always.`

### Test Plan

[x] I/we understand the owners of the involved components may require updates to
existing tests to make this code solid enough prior to committing the changes
necessary to implement this enhancement.

##### Prerequisite testing updates

##### Unit tests

For alpha, exhaustive Kubelet unit tests will be provided. Functions affected by the feature gate will be run with the feature gate on and with the feature gate off. Unit buckets will be provided for:

- HashAuth - (new, small) returns a hash code for a CRI pull image auth [link](https://github.com/kubernetes/kubernetes/pull/94899/files#diff-ca08601dfd2fdf846f066d0338dc332beddd5602ab3a71b8fac95b419842da63R704-R751) ** per review comment will use SHA256 **
- shouldPullImage - (modified, large sized change) determines if image should be pulled based on presence, and image pull policy, and now with the feature gate on if the image has been pulled/ensured by a secret. A unit test bucket did not exist for this function. The unit bucket will cover a matrix for:

```golang
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

PersistMeta() ** will be persisting SHA256 entries vs hash **

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
- Discussions went back and forth as to whether to persist the cache across reboots. It was decided to do so.
- `Never` could be always allowed to use an image on the node, regardless of its presence on the node. However, this would functionally disable this feature from a security standpoint.

## Infrastructure Needed [optional]

TBD
