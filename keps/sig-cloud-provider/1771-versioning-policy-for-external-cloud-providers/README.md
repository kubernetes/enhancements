# KEP-1771: Versioning Policy for External Cloud Providers

<!-- toc -->
- [Release Signoff Checklist](#release-signoff-checklist)
- [Summary](#summary)
- [Motivation](#motivation)
  - [Goals](#goals)
  - [Non-Goals](#non-goals)
- [Proposal](#proposal)
  - [Risks and Mitigations](#risks-and-mitigations)
- [Design Details](#design-details)
  - [Test Plan](#test-plan)
  - [Graduation Criteria](#graduation-criteria)
  - [Upgrade / Downgrade Strategy](#upgrade--downgrade-strategy)
  - [Version Skew Strategy](#version-skew-strategy)
- [Implementation History](#implementation-history)
- [Drawbacks](#drawbacks)
- [Alternatives](#alternatives)
<!-- /toc -->

## Release Signoff Checklist

- [ ] Enhancement issue in release milestone, which links to KEP dir in [kubernetes/enhancements] (not the initial KEP PR)
- [ ] KEP approvers have approved the KEP status as `implementable`
- [ ] Design details are appropriately documented
- [ ] Test plan is in place, giving consideration to SIG Architecture and SIG Testing input
- [ ] Graduation criteria is in place
- [ ] "Implementation History" section is up-to-date for milestone
- [ ] User-facing documentation has been created in [kubernetes/website], for publication to [kubernetes.io]
- [ ] Supporting documentation e.g., additional design documents, links to mailing list discussions/SIG meetings, relevant PRs/issues, release notes

[kubernetes.io]: https://kubernetes.io/
[kubernetes/enhancements]: https://git.k8s.io/enhancements
[kubernetes/kubernetes]: https://git.k8s.io/kubernetes
[kubernetes/website]: https://git.k8s.io/website

## Summary

Today we have many external (a.k.a out-of-tree) cloud providers, each responsible for managing releases for components
such as cloud-controller-manger, but possibly others. Thus far there has been no standard on how providers should be
semantically versioning their releases. Likewise, there has been no version skew policy indicating what versions of Kubernetes
is supported by a version of a cloud-controller-manager (though it is assumed based on the vendored version of
k8s.io/kubernetes). This KEP proposes to standardize all external cloud providers on a semantic versioning policy that ensures
the vendored version of the cloud-controller-manager library is likely to match the version of the Kubernetes control plane.
More concretely, releases for external cloud providers should track the major and minor versions of the Kubernetes version they
intend to support. Patch releases are not required to match. For example, release v1.18.X of cloud-controller-manager for provider
Foo should support Kubernetes clusters on v1.18.X.

## Motivation

* improve discovery of version supportability across Kubernetes and external cloud providers.
* promote using the latest cloud-controller-manager library based on the Kubernetes version of the cluster
* prevent wide version skew of Kubernetes control plane and cloud-controller-manager.
* reduce chances of incompatibility issues that occur when a cloud-controller-manager vendoring a really
old version of Kubernetes is running on a much newer version of Kubernetes.

### Goals

* standardize the semantic versioning of all external cloud providers
* reduce chances of incompatibility issues due to version skew between cloud-controller-manager and Kubernetes.

### Non-Goals

* improving the build/release process for publishing arifacts like binaries and docker images.

## Proposal

Semantic versioning used by external cloud providers should track the major and minor releases of the supported Kubernetes
version in that release. Patch releases are not required to match. For example, release v1.18.X of the cloud-controller-manager
for provider Foo should support Kubernetes v1.18.X. The benefits of this versioning policy include but are not limited to:

* the desired version of the cloud-controller-manager is explicit based on the Kubernetes version of the cluster
* resolves incompatibility issues occurring wbsfrom vendoring a really old version of k8s.io/kubernetes
* ensures the vendored version of the cloud-controller-manager is up-to-date

### Risks and Mitigations

* timeline for feature additions are tied to the Kubernetes release cycle (assuming features are only added in minor releases).
* maintainers of external providers will likely have to manage release branches for each minor release of Kubernetes.

## Design Details

Updates to release version for external cloud providers:
* match major and minor versions of releases based on the version of k8s.io/kubernetes that is vendored to
build the cloud-controller-manager
* use patch releases for bug fixes pertaining to specific Kubernetes releases. In many cases bugs will be backported to
all minor versions that Kubernetes supports.
* release a new minor version after every new Kubernetes minor release

### Test Plan

Not applicable.

### Graduation Criteria

Not applicable.

### Upgrade / Downgrade Strategy

Not applicable.

### Version Skew Strategy

The proposed semantic versioning policy would simplify version skew strategy for the cloud-controller-manager since
users can treat versioning the cloud-controller-manager as part of their control plane (if they weren't already).

## Implementation History

- 2020-04-29: the KEP is accepted in the `implementable` state.

## Drawbacks

One motivation for external cloud providers was so to decouple the release cadence for providers from the Kubernetes release.
In practice we've learned that users upgrade the cloud-controller-manager along upgrades to the Kubernetes version. This tends
to be true on managed Kubernetes offerings as well.


## Alternatives

* don't standardize semantic versioning of releases for external cloud provider.

