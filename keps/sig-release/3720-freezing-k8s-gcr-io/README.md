# KEP-3720: Freeze `k8s.gcr.io` image registry

The change proposed by this KEP is very unusual as the engineering work will **not** be done in the k/k repository. However, this is a major change to the project hence the KEP.

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
- [Communication Plan](#communication-plan)
- [Production Readiness Review Questionnaire](#production-readiness-review-questionnaire)
- [Implementation History](#implementation-history)
- [Drawbacks](#drawbacks)
- [Alternatives](#alternatives)
- [Infrastructure Needed (Optional)](#infrastructure-needed-optional)
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

Items marked with (R) are required *prior to targeting to a milestone / release*.

This KEP is unusual and doesn't actually make/propose changes to the Kubernetes codebase. It does propose a major change to how images of the Kubernetes are consumed hence the KEP.

- [X] (R) Enhancement issue in release milestone, which links to KEP dir in [kubernetes/enhancements] (not the initial KEP PR)
- [X] (R) KEP approvers have approved the KEP status as `implementable`
- [X] (R) Design details are appropriately documented
- [X] (N/A) Test plan is in place, giving consideration to SIG Architecture and SIG Testing input (including test refactors)
  - [X] (N/A) e2e Tests for all Beta API Operations (endpoints)
  - [X] (N/A) Ensure GA e2e tests meet requirements for [Conformance Tests](https://github.com/kubernetes/community/blob/master/contributors/devel/sig-architecture/conformance-tests.md) 
  - [X] (N/A) Minimum Two Week Window for GA e2e tests to prove flake free
- [X] (N/A) Graduation criteria is in place
  - [X] (N/A) [all GA Endpoints](https://github.com/kubernetes/community/pull/1806) must be hit by [Conformance Tests](https://github.com/kubernetes/community/blob/master/contributors/devel/sig-architecture/conformance-tests.md) 
- [X] (N/A) Production readiness review completed
- [X] (N/A) Production readiness review approved
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

The Kubernetes project has created and runs the `registry.k8s.io` image registry which is fully controlled by the community.
This registry has been [GA for several months](https://kubernetes.io/blog/2022/11/28/registry-k8s-io-faster-cheaper-ga/) now and we need to freeze the old image registry.

## Motivation

Running public image registries is very expensive and eats up a significant chunk of the project's Infrastructure budget. We built `registry.k8s.io` image registry to serve images from various origins around the world depending on the location of the user. For example, an a kops Kubernetes cluster in eu-west-1 can pull images from an AWS S3 bucket in the same region which is very fast and more importantly very cheap for the Kubernetes project.

There was a plan to redirect `k8s.gcr.io` to `registry.k8s.io` but it didn't [work out](https://kubernetes.slack.com/archives/CCK68P2Q2/p1666725317568709) so we backported the image registry defaults to 1.24, 1.23, and 1.22 so all the patch releases from December 2022 will using the new registry by default.

We are currently exceeding our budget as it will take quite a while for end users to upgrade Kubernetes to v1.25 so we want to incentivise our end users to move to the new registry as fast as possible by freezing the registry by 1.27. This would mean that all subsequent image releases will not be available on the old registry.

### Goals

Freeze `k8s.gcr.io` image registry and push all new images and tags exclusively to the `registry.k8s.io` image registry.

### Non-Goals

- `registry.k8s.io` internal implementations details. That is handled separately by sig-k8s-infra.

## Proposal

Freeze the `k8s.gcr.io` image by not pushing any new digests or tags after 1.27 release. The 1.27 release itself won't be available on `k8s.gcr.io`.

I'm proposing that on the 1st of April 2023 (10 days before 1.27 is released):

- `k8s.gcr.io` is frozen and no new images will be published by any subproject.
- last 1.23 release on `k8s.gcr.io` will be 1.23.18 (goes EoL before the freeze)
- last 1.24 release on `k8s.gcr.io` will be 1.24.12
- last 1.25 release on `k8s.gcr.io` will be 1.25.8
- last 1.26 release on `k8s.gcr.io` will be 1.26.3
- 1.27.0 will not be available `k8s.gcr.io`

### Risks and Mitigations

There are no risks. The old registry will still be available and you can pull the images before 1.27 on there. This change will also
affect other users of k8s.gcr.io who should have already updated their helm charts and manifests to use the new registry.

## Design Details

The image promotion process is described [here](https://github.com/kubernetes/k8s.io/tree/main/k8s.gcr.io). Please read it for full details.

This is the planned technical changes(grouped by repos):

- k-sigs/promo-tools
  - merge https://github.com/kubernetes-sigs/promo-tools/pull/669
- k/k8s.io
  - Fix https://github.com/kubernetes/k8s.io/issues/4611
  - clean up the contents of the `registry.k8s.io` folder. Most of the content should be in k/registry.k8s.io repository
  - duplicate the top level folder `k8s.gcr.io` in the repo and call it `registry.k8s.io`
- k/test-infra
  - blockade the k8s.gcr.io folder in k/k8s.io repository. blockade is a prow plugin that rejects PRs that modify specific folders/files.
  - update the ProwJobs [post-k8sio-image-promo](https://github.com/kubernetes/test-infra/blob/master/config/jobs/kubernetes/sig-k8s-infra/trusted/releng/releng-trusted.yaml) and [pull-k8sio-cip](https://github.com/kubernetes/test-infra/blob/master/config/jobs/kubernetes/sig-k8s-infra/releng/artifact-promotion-presubmits.yaml) `thin-manifest-dir` flags to point to the new folder


### Test Plan

This is not applicable.

### Graduation Criteria

This is not applicable.

### Upgrade / Downgrade Strategy

When users upgrade to various kubernetes versions that use the new image registry, they will be able to pull images from the new
registry.

### Version Skew Strategy

This is not applicable.

## Communication Plan

This is a major change and requires an appropriate communication plan.

We plan on communicating this change via:
- an email to k-dev
- an email to k-announce
- a blog post on kubernetes.io

## Production Readiness Review Questionnaire

This is not applicable.

## Implementation History

## Drawbacks

This is not applicable.

<!--
Why should this KEP _not_ be implemented?
-->

## Alternatives

We keep pushing new images to the old registry.

## Infrastructure Needed (Optional)

None as it has already been deployed.
