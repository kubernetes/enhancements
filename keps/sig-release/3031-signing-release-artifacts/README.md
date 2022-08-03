# KEP-3031: Signing release artifacts

<!-- toc -->
- [Release Signoff Checklist](#release-signoff-checklist)
- [Summary](#summary)
- [Motivation](#motivation)
  - [Goals](#goals)
  - [Non-Goals](#non-goals)
- [Proposal](#proposal)
  - [Alpha implementation](#alpha-implementation)
  - [Beta graduation](#beta-graduation)
  - [User Stories (Optional)](#user-stories-optional)
  - [Risks and Mitigations](#risks-and-mitigations)
  - [Test Plan](#test-plan)
      - [Prerequisite testing updates](#prerequisite-testing-updates)
      - [Unit tests](#unit-tests)
      - [Integration tests](#integration-tests)
      - [e2e tests](#e2e-tests)
  - [Graduation Criteria](#graduation-criteria)
    - [Alpha](#alpha)
    - [Beta](#beta)
    - [GA](#ga)
- [Production Readiness Review Questionnaire](#production-readiness-review-questionnaire)
  - [Feature Enablement and Rollback](#feature-enablement-and-rollback)
  - [Rollout, Upgrade and Rollback Planning](#rollout-upgrade-and-rollback-planning)
  - [Monitoring Requirements](#monitoring-requirements)
  - [Dependencies](#dependencies)
  - [Scalability](#scalability)
  - [Troubleshooting](#troubleshooting)
- [Drawbacks](#drawbacks)
- [Alternatives](#alternatives)
- [Implementation History](#implementation-history)
<!-- /toc -->

## Release Signoff Checklist

Items marked with (R) are required _prior to targeting to a milestone / release_.

- [x] (R) Enhancement issue in release milestone, which links to KEP dir in [kubernetes/enhancements] (not the initial KEP PR)
- [x] (R) KEP approvers have approved the KEP status as `implementable`
- [x] (R) Design details are appropriately documented
- [x] (R) Test plan is in place, giving consideration to SIG Architecture and SIG Testing input (including test refactors)
- [x] (R) Graduation criteria is in place
- [x] (R) Production readiness review completed
- [x] (R) Production readiness review approved
- [x] "Implementation History" section is up-to-date for milestone
- [x] User-facing documentation has been created in [kubernetes/website], for publication to [kubernetes.io]
- [x] Supporting documentation—e.g., additional design documents, links to mailing list discussions/SIG meetings, relevant PRs/issues, release notes

[kubernetes.io]: https://kubernetes.io/
[kubernetes/enhancements]: https://git.k8s.io/enhancements
[kubernetes/kubernetes]: https://git.k8s.io/kubernetes
[kubernetes/website]: https://git.k8s.io/website

## Summary

Target of this enhancement is to define which technology the Kubernetes
community is using to signs release artifacts.

## Motivation

Signing artifacts provides end users a chance to verify the integrity of the
downloaded resource. It allows to mitigate man-in-the-middle attacks directly on
the client side and therefore ensures the trustfulness of the remote serving the
artifacts.

### Goals

- Defining the used tooling for signing all Kubernetes related artifacts
- Providing a standard signing process for related projects (like k/release)

### Non-Goals

- Discussing not user-facing internal technical implementation details

## Proposal

Every Kubernetes release produces a set of artifacts. We define artifacts as
something consumable by end users. Artifacts can be binaries, container images,
checksum files, documentation, provenance metadata, or the software bill of
materials (SBOM). Only the official Kubernetes container images are signed right
now.

The overall goal of SIG Release is to unify the way how to sign artifacts. This
will be done by relying on the tools of the Linux Foundations digital signing
project [sigstore](https://www.sigstore.dev). This goal aligns with the
[Roadmap and Vision](https://github.com/kubernetes/sig-release/blob/f62149/roadmap.md)
of SIG Release to provide a secure software supply chain for Kubernetes. It also
joins the effort of gaining full SLSA Compliance in the Kubernetes Release
Process ([KEP-3027](https://github.com/kubernetes/enhancements/issues/3027)).
Because of that, the future [SLSA](https://slsa.dev) compliance of artifacts
produced by SIG release will require signing artifacts starting from level 2.

[cosign](https://github.com/sigstore/cosign) will be the tool of our choice when
speaking about the technical aspects of the solution. How we integrate the
projects into our build process in k/release is out of scope of this KEP and
will be discussed in the Release Engineering subproject of SIG Release. A
pre-evaluation of the tool has been done already to ensure that it meets the
requirements.

An [ongoing discussion](https://github.com/kubernetes/release/issues/2227) about
using cosign already exists in k/release. This issue contains technical
discussions about how to utilize the existing Google infrastructure as well as
consider utilizing keyless signing via workload identities. Nevertheless, this
KEP focuses more on the "What" aspects rather than the "How".

### Alpha implementation

The alpha phase of the proposal is about signing the official Kubernetes
container images and providing a minimum infrastructure to achieve that goal.

### Beta graduation

Graduation the KEP to beta means that we will now sign all artifacts which got
created during the release process. This includes binary artifacts, source code
tarballs, documentation and the SBOM.

This explicitly exudes the provenance data, which will be signed into a
different location once we graduate the feature to GA.

### User Stories (Optional)

- As an end user, I would like to be able to verify the Kubernetes release
  artifacts, so that I can mitigate possible resource modifications by the
  network.

### Risks and Mitigations

- **Risk:** Unauthorized access to the signing key or its infrastructure

  **Mitigations:**

  - Storing the credentials in a secure Google Cloud Project with
    limited access for SIG Release.
  - Enabling the cosign [transparency log
    (Rekor)](https://github.com/sigstore/cosign#rekor-support) to make the key
    usage publicly auditable.
  - Working towards [keyless
    signing](https://github.com/sigstore/cosign/blob/3f83940/KEYLESS.md) to
    minimize the attack surface of the supply chain.

### Test Plan

[x] I/we understand the owners of the involved components may require updates to
existing tests to make this code solid enough prior to committing the changes necessary
to implement this enhancement.

##### Prerequisite testing updates

- None

##### Unit tests

Testing of the lower-level signing implementation will be done by writing unit tests
as well as integration tests within the
[release-sdk](https://github.com/kubernetes-sigs/release-sdk) repository. This
implementation is going to be used by
[krel](https://github.com/kubernetes/release/blob/master/docs/krel/README.md)
during the release creation process, which is tested separately. The overall
integration into krel can be tested manually by the Release Managers as well,
while we use the pre-releases of v1.24 as first instance for full end-to-end
feedback.

##### Integration tests

See the unit test section.

##### e2e tests

See the unit test section.

### Graduation Criteria

#### Alpha

- Outline and integrate an example process for signing Kubernetes release
  artifacts.

  Tracking issue: https://github.com/kubernetes/release/issues/2383

#### Beta

- Standard Kubernetes release artifacts (binaries, container images, etc.) are
  signed.

#### GA

- All Kubernetes artifacts are signed. This does exclude everything which gets
  build outside of the main Kubernetes repository.
- Kubernetes owned infrastructure is used for the signing (root trust) and
  verification (transparency log) process.

## Production Readiness Review Questionnaire

### Feature Enablement and Rollback

###### How can this feature be enabled / disabled in a live cluster?

Signed images have not to be verified, so they do not interfere with a running
cluster at all. They can be verified manually or by using the tooling provided
by our documentation.

###### Does enabling the feature change any default behavior?

Not when a manual verification will be done. If the cluster will change its
configuration to only accept signed images, then invalid signatures will cause
the container runtime to refuse the image pull. The same behavior could be
achieved by using an admission webhook which verifies the signature.

###### Can the feature be disabled once it has been enabled (i.e. can we roll back the enablement)?

Yes, depending on how the signature verification will be done.

###### What happens if we reenable the feature if it was previously rolled back?

It will behave in the same way as enabled initially.

###### Are there any tests for feature enablement/disablement?

No, not on a cluster level. We test the signatures during the release process.

### Rollout, Upgrade and Rollback Planning

Not required.

### Monitoring Requirements

Not required.

### Dependencies

Not required.

### Scalability

Not required.

### Troubleshooting

Not required.

## Drawbacks

- The initial implementation effort from the release engineering perspective
  requires adding an additional layer of complexity to the Kubernetes build
  pipeline.

## Alternatives

- Using the [OCI Registry As Storage (ORAS) project](https://github.com/oras-project/oras)

## Implementation History

- 2022-05-30 Graduate to beta
- 2022-01-27 Updated to contain test plan and correct milestones
- 2021-11-29 Initial Draft
