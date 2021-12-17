# KEP-3031: Signing release artifacts

<!-- toc -->

- [Release Signoff Checklist](#release-signoff-checklist)
- [Summary](#summary)
- [Motivation](#motivation)
  - [Goals](#goals)
  - [Non-Goals](#non-goals)
- [Proposal](#proposal)
  - [User Stories (Optional)](#user-stories-optional)
  - [Risks and Mitigations](#risks-and-mitigations)
  - [Graduation Criteria](#graduation-criteria)
    - [Alpha](#alpha)
    - [Beta](#beta)
    - [GA](#ga)
- [Drawbacks](#drawbacks)
- [Alternatives](#alternatives)
- [Implementation History](#implementation-history)
<!-- /toc -->

## Release Signoff Checklist

<!--
**ACTION REQUIRED:** In order to merge code into a release, there must be an
issue in [kubernetes/enhancements] referencing this KEP and targeting a release
milestone **before the [Enhancement Freeze](https://git.k8s.io/sig-release/releases)
of the targeted release**.

For enhancements that make changes to code or processes/procedures in core
Kubernetes—i.e., [kubernetes/kubernetes], we require the following Release
Signoff checklist to be completed.

Check these off as they are completed for the Release Team to track. These
checklist items _must_ be updated for the enhancement to be released.
-->

Items marked with (R) are required _prior to targeting to a milestone / release_.

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

<!--
**Note:** This checklist is iterative and should be reviewed and updated every time this enhancement is being considered for a milestone.
-->

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
materials. None of those end-user resources are signed right now.

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

### Graduation Criteria

#### Alpha

- Outline and integrate an example process for signing Kubernetes release
  artifacts.

#### Beta

- Standard Kubernetes release artifacts (binaries and container images) are
  signed.

#### GA

- All Kubernetes artifacts are signed. This does exclude everything which gets
  build outside of the main Kubernetes repository.

## Drawbacks

- The initial implementation effort from the release engineering perspective
  requires adding an additional layer of complexity to the Kubernetes build
  pipeline.

## Alternatives

- Using the [OCI Registry As Storage (ORAS) project](https://github.com/oras-project/oras)

## Implementation History

- 2021-11-29 Initial Draft
