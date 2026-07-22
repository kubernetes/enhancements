# KEP-6083: Digest-Aware ImagePullPolicy Defaulting

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
  - [Notes/Constraints/Caveats](#notesconstraintscaveats)
  - [Risks and Mitigations](#risks-and-mitigations)
- [Design Details](#design-details)
  - [Test Plan](#test-plan)
      - [Prerequisite testing updates](#prerequisite-testing-updates)
      - [Unit tests](#unit-tests)
      - [Integration tests](#integration-tests)
      - [e2e tests](#e2e-tests)
  - [Graduation Criteria](#graduation-criteria)
    - [Alpha](#alpha)
    - [Beta](#beta)
    - [GA](#ga)
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
- [Drawbacks](#drawbacks)
- [Alternatives](#alternatives)
<!-- /toc -->

## Release Signoff Checklist

Items marked with (R) are required *prior to targeting to a milestone / release*.

- [ ] (R) Enhancement issue in release milestone, which links to KEP dir in [kubernetes/enhancements] (not the initial KEP PR)
- [ ] (R) KEP approvers have approved the KEP status as `implementable`
- [ ] (R) Design details are appropriately documented
- [ ] (R) Test plan is in place, giving consideration to SIG Architecture and SIG Testing input (including test refactors)
  - [ ] e2e Tests for all Beta API Operations (endpoints)
  - [ ] (R) Ensure GA e2e tests meet requirements for [Conformance Tests](https://github.com/kubernetes/community/blob/master/contributors/devel/sig-architecture/conformance-tests.md)
  - [ ] (R) Minimum Two Week Window for GA e2e tests to prove flake free
- [ ] (R) Graduation criteria is in place
  - [ ] (R) [all GA Endpoints](https://github.com/kubernetes/community/pull/1806) must be hit by [Conformance Tests](https://github.com/kubernetes/community/blob/master/contributors/devel/sig-architecture/conformance-tests.md) within one minor version of promotion to GA
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

When a container image reference includes a digest (e.g.,
`myimage:latest@sha256:abc123`), the image reference is immutable. The digest uniquely
identifies the content regardless of any tag. Today, the `ImagePullPolicy`
defaulting logic only inspects the tag: if the tag is `latest` (or omitted), it
defaults to `Always`, even when a digest is present. This causes unnecessary
image pulls for content that is guaranteed not to change.

This KEP proposes that the defaulting logic also consider the presence of a
digest. When a digest is specified, `ImagePullPolicy` should default to
`IfNotPresent`, regardless of the tag.

## Motivation

The [Kubernetes documentation][image-pull-docs] states:

> if you omit the imagePullPolicy field, and you specify the digest for the
> container image, the imagePullPolicy is automatically set to IfNotPresent.

However, the actual implementation does not match this documented behavior.
The defaulting logic in `SetDefaults_Container` only examines the tag returned
by `ParseImageName` and does not account for the digest. As a result, an image
reference like `myimage:latest@sha256:abc123` gets `ImagePullPolicy: Always`
even though the digest makes the reference immutable.

This discrepancy was introduced unintentionally in commit
[`7796b619fdf`][parseimagename-refactor] (2016), which refactored
`ParseImageName` to return the digest as a separate value. Prior to that
change, `ParseImageName` returned only `(repo, tag, error)` and stored the
digest in the `tag` variable when both were present
(`tag = digested.Digest().String()`). This meant the tag was effectively
ignored if a digest was also specified, and the defaulting logic would see a
digest string (not `"latest"`) as the tag, resulting in `IfNotPresent`. The
refactor split `tag` and `digest` into separate return values but did not
update the defaulting logic to account for the new `digest` field.

[parseimagename-refactor]: https://github.com/kubernetes/kubernetes/commit/7796b619fdf

[image-pull-docs]: https://kubernetes.io/docs/concepts/containers/images/#imagepullpolicy-defaulting

### Goals

- When an image reference contains a digest and `ImagePullPolicy` is not
  explicitly set, default to `IfNotPresent` regardless of the tag.
- Align the implementation with the existing Kubernetes documentation.
- Reduce unnecessary image pulls for digest-pinned images.

### Non-Goals

- Changing the behavior when `ImagePullPolicy` is explicitly set by the user.
- Changing the defaulting behavior for image references without a digest.
- Modifying how admission controllers interact with `ImagePullPolicy`.
- Changing kubelet image-pulling or caching behavior.

## Proposal

Modify the `ImagePullPolicy` defaulting logic in the API server
(`SetDefaults_Container` and `SetDefaults_Volume` in
`pkg/apis/core/v1/defaults.go`) to check for the presence of a digest in
addition to the tag. When a digest is present, default `ImagePullPolicy` to
`IfNotPresent`, regardless of the tag value. Because `SetDefaults_Container`
is invoked for all container types (regular containers, init containers, and
ephemeral containers), this change applies uniformly across all of them.

The change is gated behind a new feature gate,
`DigestAwareImagePullPolicyDefaulting`, to allow safe rollout and rollback.

### User Stories

#### Story 1

As a developer, I pin my container images by digest for reproducibility
(`myimage:latest@sha256:abc123`). I expect Kubernetes to recognize that this
reference is immutable and not re-pull the image on every pod creation. Today,
because the tag is `latest`, Kubernetes defaults `ImagePullPolicy` to `Always`,
causing unnecessary registry round-trips and slowing pod startup.

#### Story 2

As a platform operator, I use admission policies that require all images to be
pinned by digest. I expect that digest-pinned images are not unnecessarily
re-pulled, reducing load on my container registry and improving pod startup
latency in air-gapped or bandwidth-constrained environments.

### Notes/Constraints/Caveats

- When both a tag and a digest are present in an image reference, the container
  runtime resolves the image by digest. The tag is effectively informational.
  This is OCI specification behavior and is implemented consistently across
  container runtimes (containerd, CRI-O).

- The `Always` pull policy with a digest still results in a registry manifest
  fetch (to verify the digest), even though the actual image layers are not
  re-downloaded if already present. Defaulting to `IfNotPresent` eliminates
  this manifest check entirely, which is beneficial in air-gapped or
  bandwidth-constrained environments.

- Image references with a digest but no tag (e.g., `myimage@sha256:abc123`)
  are already handled correctly. `ParseImageName` only sets the default
  `latest` tag when both tag and digest are empty, so a digest-only reference
  has `tag=""` and `digest="sha256:..."`, which already defaults to
  `IfNotPresent` under both the current and proposed logic.

### Risks and Mitigations

**Risk: Breaking existing workflows that rely on `Always` pull with digest references.**

Some users or admission controllers may intentionally use `:latest@sha256:...`
with `Always` to force a manifest verification against the registry on every
pull. With this change, the *default* would change, but users can still
explicitly set `ImagePullPolicy: Always` to preserve the current behavior.

*Mitigation:* The change is gated behind a feature gate so it can be disabled
if it causes issues. The feature gate follows the standard alpha/beta/GA
graduation process, giving users multiple releases to adapt.

**Risk: Multi-tenant security concerns around cached images.**

In shared clusters, `Always` pull is sometimes used to ensure pod access
control to images is verified against the registry. However, this concern is
better addressed by [KEP-2535: Ensure Secret Pulled Images][kep-2535], which
provides proper image access control without relying on pull policy.

*Mitigation:* Document that users relying on `Always` for access control
should explicitly set the pull policy or adopt KEP-2535.

[kep-2535]: https://github.com/kubernetes/enhancements/tree/master/keps/sig-node/2535-ensure-secret-pulled-images

## Design Details

The implementation modifies the defaulting logic in
`pkg/apis/core/v1/defaults.go`. A new helper function `defaultImagePullPolicy`
is introduced that considers both the tag and digest:

```go
func defaultImagePullPolicy(ref string) v1.PullPolicy {
	_, tag, digest, _ := parsers.ParseImageName(ref)
	if tag == "latest" && digest == "" {
		return v1.PullAlways
	}
	return v1.PullIfNotPresent
}
```

This function is used by both `SetDefaults_Container` and `SetDefaults_Volume`
(for image volumes), replacing the current tag-only logic. Since
`SetDefaults_Container` is called for regular containers, init containers, and
ephemeral containers (via generated defaults in `zz_generated.defaults.go`),
all container types benefit from this change.

The new behavior is gated behind the `DigestAwareImagePullPolicyDefaulting`
feature gate. When the gate is disabled, the existing tag-only logic is
preserved. The feature gate will be defined in `pkg/features/kube_features.go`
and checked in `defaultImagePullPolicy`.

### Test Plan

[x] I/we understand the owners of the involved components may require updates to
existing tests to make this code solid enough prior to committing the changes
necessary to implement this enhancement.

##### Prerequisite testing updates

No prerequisite testing updates are required.

##### Unit tests

Unit tests in `pkg/apis/core/v1/defaults_test.go` will cover the following
cases with the feature gate both enabled and disabled:

- Image reference with tag only (e.g., `image:v1`) -> `IfNotPresent`
- Image reference with `latest` tag (e.g., `image:latest`) -> `Always`
- Image reference with implicit `latest` tag (e.g., `image`) -> `Always`
- Image reference with `latest` tag and digest -> `IfNotPresent` (new case)
- Image reference with digest only, no tag (e.g., `image@sha256:...`) -> `IfNotPresent`
- Image volume with `latest` tag and digest -> `IfNotPresent` (new case)
- Feature gate disabled: `latest` tag with digest -> `Always` (preserves old behavior)

Coverage data will be collected prior to alpha targeting.

- `pkg/apis/core/v1`: `<date>` - `<test coverage>`

##### Integration tests

Integration tests in `test/integration/` will verify that the full API server
admission chain correctly applies the new default. Specific scenarios:

- Create a pod with `image:latest@sha256:...` and no explicit pull policy;
  verify the persisted spec has `imagePullPolicy: IfNotPresent`.
- Create a pod with `image:latest` (no digest) and no explicit pull policy;
  verify the persisted spec has `imagePullPolicy: Always` (unchanged).
- Create a pod with an explicit `imagePullPolicy: Always` and a digest;
  verify the explicit policy is preserved.
- Repeat the above for init containers and ephemeral containers.

##### e2e tests

e2e tests will be added under `test/e2e/node/` to verify end-to-end behavior:

- Deploy a pod with `image:latest@sha256:...` and no explicit pull policy;
  verify the pod starts successfully and `imagePullPolicy` is `IfNotPresent`.
- Verify that the image is not re-pulled on subsequent pod recreations when
  the image is already present on the node.
- Test with the feature gate disabled to confirm the old behavior is preserved.

### Graduation Criteria

#### Alpha

- Feature implemented behind `DigestAwareImagePullPolicyDefaulting` feature gate
- Unit tests for the new defaulting behavior
- Feature gate defaults to disabled

#### Beta

- Feature gate defaults to enabled
- Gather feedback from early adopters
- e2e tests in Testgrid and linked in this KEP, stable with no flakes for at
  least two weeks
- Integration tests covering all container types (regular, init, ephemeral)
- Upgrade/downgrade tests verifying correct behavior when the feature gate
  is toggled
- Documentation updated on kubernetes.io

#### GA

- Feature gate is locked to enabled
- At least two releases at beta without reported issues
- Conformance tests covering `ImagePullPolicy` defaulting for digest-pinned
  images

### Upgrade / Downgrade Strategy

On upgrade with the feature gate enabled, pods created after the upgrade with
digest-pinned images and no explicit `ImagePullPolicy` will get
`IfNotPresent` instead of `Always`. Existing pods are not affected since
`ImagePullPolicy` is defaulted at creation time and persisted.

On downgrade or feature gate disable, the behavior reverts to the current
tag-only logic. New pods will again default to `Always` for `:latest` images
even with a digest. Existing pods are not affected.

### Version Skew Strategy

The defaulting is performed by the API server at pod creation time. There is
no coordination needed with the kubelet or other components. In an HA cluster
with mixed API server versions during upgrade, pods may get different defaults
depending on which API server handles the creation request. This is acceptable
because the resulting policy is always valid and the feature gate can be used
to ensure consistent behavior.

## Production Readiness Review Questionnaire

### Feature Enablement and Rollback

###### How can this feature be enabled / disabled in a live cluster?

- [x] Feature gate (also fill in values in `kep.yaml`)
  - Feature gate name: `DigestAwareImagePullPolicyDefaulting`
  - Components depending on the feature gate: `kube-apiserver`

###### Does enabling the feature change any default behavior?

Yes. When a container image reference includes a digest (e.g.,
`myimage:latest@sha256:abc123`) and `ImagePullPolicy` is not explicitly set,
the default changes from `Always` to `IfNotPresent`. Users who want to
preserve the previous behavior can explicitly set `ImagePullPolicy: Always`.

###### Can the feature be disabled once it has been enabled (i.e. can we roll back the enablement)?

Yes. Disabling the `DigestAwareImagePullPolicyDefaulting` feature gate reverts
to the current behavior. Only newly created pods are affected; existing pods
retain their previously defaulted `ImagePullPolicy`.

###### What happens if we reenable the feature if it was previously rolled back?

New pods with digest-pinned images will again default to `IfNotPresent`. No
impact on existing pods.

###### Are there any tests for feature enablement/disablement?

Unit tests will exercise the defaulting behavior with the feature gate both
enabled and disabled.

### Rollout, Upgrade and Rollback Planning

###### How can a rollout or rollback fail? Can it impact already running workloads?

This feature only affects defaulting at pod creation time. Already running
workloads are not affected. The risk is limited to new pods getting a different
default pull policy. Since `IfNotPresent` is a valid and commonly used policy,
the impact is minimal.

###### What specific metrics should inform a rollback?

An unexpected increase in image pull errors or pod startup failures for
digest-pinned images after enabling the feature gate.

###### Were upgrade and rollback tested? Was the upgrade->downgrade->upgrade path tested?

Will be tested during beta.

###### Is the rollout accompanied by any deprecations and/or removals of features, APIs, fields of API types, flags, etc.?

No.

### Monitoring Requirements

###### How can an operator determine if the feature is in use by workloads?

Operators can check if the `DigestAwareImagePullPolicyDefaulting` feature gate
is enabled on the API server. They can also inspect the `ImagePullPolicy` of
newly created pods with digest-pinned images to confirm the default changed.
At beta, we will evaluate whether a metric tracking the number of pods
defaulted to `IfNotPresent` due to digest presence would be valuable for
adoption tracking.

###### How can someone using this feature know that it is working for their instance?

- [x] Other (treat as last resort)
  - Details: Create a pod with `image: myimage:latest@sha256:...` without
    setting `ImagePullPolicy`. Inspect the pod spec; it should show
    `imagePullPolicy: IfNotPresent`.

###### What are the reasonable SLOs (Service Level Objectives) for the enhancement?

This feature does not introduce new SLOs. It may improve pod startup latency
by reducing unnecessary image pulls.

###### What are the SLIs (Service Level Indicators) an operator can use to determine the health of the service?

- [x] Other (treat as last resort)
  - Details: Existing kubelet metrics for image pull duration and count can be
    used to observe changes in pull behavior.

###### Are there any missing metrics that would be useful to have to improve observability of this feature?

No additional metrics are needed. Existing image pull metrics are sufficient.

### Dependencies

###### Does this feature depend on any specific services running in the cluster?

No.

### Scalability

###### Will enabling / using this feature result in any new API calls?

No. This may reduce API calls to container registries by avoiding unnecessary
manifest checks.

###### Will enabling / using this feature result in introducing new API types?

No.

###### Will enabling / using this feature result in any new calls to the cloud provider?

No. This may reduce calls to cloud-hosted container registries.

###### Will enabling / using this feature result in increasing size or count of the existing API objects?

No.

###### Will enabling / using this feature result in increasing time taken by any operations covered by existing SLIs/SLOs?

No.

###### Will enabling / using this feature result in non-negligible increase of resource usage (CPU, RAM, disk, IO, ...) in any components?

No.

###### Can enabling / using this feature result in resource exhaustion of some node resources (PIDs, sockets, inodes, etc.)?

No.

### Troubleshooting

###### How does this feature react if the API server and/or etcd is unavailable?

This feature is part of the API server defaulting logic. If the API server is
unavailable, no pods can be created, so the feature has no additional impact.

###### What are other known failure modes?

None. The change is to defaulting logic only and does not introduce new
failure paths.

###### What steps should be taken if SLOs are not being met to determine the problem?

Disable the `DigestAwareImagePullPolicyDefaulting` feature gate and restart
the API server.

## Implementation History

- 2024-09-16: Initial PR [kubernetes/kubernetes#134092](https://github.com/kubernetes/kubernetes/pull/134092) (without feature gate)
- 2026-05-15: KEP created as provisional

## Drawbacks

- This is technically a change in default behavior, which can surprise users
  who expect `latest` tag to always mean `Always` pull regardless of digest.
- Users relying on the registry manifest check (triggered by `Always` pull)
  as a form of image verification will need to explicitly set the pull policy.

## Alternatives

**Do nothing and document the current behavior.** This would mean accepting
that the documentation is wrong and updating it instead of fixing the code.
However, the documented behavior is more correct (digest implies immutability),
so fixing the code is preferred.

**Treat this as a bug fix without a KEP.** Given the feedback from sig-node
that this is a behavior change that could affect existing workflows, a KEP with
a feature gate is the safer path.
