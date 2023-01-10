# KEP-3720: Freeze `k8s.gcr.io` image registry

The change proposed by this KEP is very unusual as the engineering work will be done on release-tools. However, this is a major change to the project hence the KEP.

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

The Kubernetes project has created and runs `registry.k8s.io` image registry which is fully controlled by the community.
This registry has been [GA for several months](https://kubernetes.io/blog/2022/11/28/registry-k8s-io-faster-cheaper-ga/) now and we need to freeze the old image registry.


## Motivation

Running image registries is very expensive and eats up a significant chunk of the projects Infrastructure costs. We built `registry.k8s.io` image registry to serve images from various origins around the world depending on the location of the user. For example, an AWS Cluster API cluster in eu-west-1 can pull images from an AWS S3 bucket in the same region which is very fast and more importantly very cheap for the Kubernetes project.

There was a plan to redirect `k8s.gcr.io` to `registry.k8s.io` but it didn't work out so we backported the image registry defaults to 1.26.x, 1.25, 1.24, 1.23, and 1.22 so all the patch releases from December 2022 will using the new registry by default.

We are currently exceeding our budget as it will take quite a while for end users to upgrade Kubernetes to v1.25 so we want to incentivise our end users to move to the new registry as fast as possible by freezing the registry by 1.27. This would mean that all subsequent image releases will not be available on the old registry.
<!--
This section is for explicitly listing the motivation, goals, and non-goals of
this KEP.  Describe why the change is important and the benefits to users. The
motivation section can optionally provide links to [experience reports] to
demonstrate the interest in a KEP within the wider Kubernetes community.

[experience reports]: https://github.com/golang/go/wiki/ExperienceReports
-->

### Goals

Freeze `k8s.gcr.io` image registry and push all new images and tags to the `registry.k8s.io` registry.

### Non-Goals

- `registry.k8s.io` internal implementations details. That is handled separately by sig-k8s-infra

## Proposal

Freeze the `k8s.gcr.io` image by not pushing any new digests or tags after 1.27 release. The 1.27 release itself won't be available on `k8s.gcr.io`

### Risks and Mitigations

There are no risks. The old registry will still be available and you can pull the images before 1.27 on there. This change will also
affect other users of k8s.gcr.io who should have already updated their helm charts and manifests to use the new registry.

## Design Details

The image promotion process is described [here](https://github.com/kubernetes/k8s.io/tree/main/k8s.gcr.io). Please read it for full details.

This is the planned technical changes(grouped by repos).

- k-sigs/promo-tools
  - merge https://github.com/kubernetes-sigs/promo-tools/pull/669
- k/k8s.io
  - Fix https://github.com/kubernetes/k8s.io/issues/4611
  - clean up the contents of the `registry.k8s.io` folder. Most of the content should be in k/registry.k8s.io repository
  - duplicate the top level `k8s.gcr.io` and call the new folder `registry.k8s.io`
- k/test-infra
  - blockade the k8s.gcr.io folder in k/k8s.io repository
  - update the ProwJobs [post-k8sio-image-promo](https://github.com/kubernetes/test-infra/blob/master/config/jobs/kubernetes/sig-k8s-infra/trusted/releng/releng-trusted.yaml) and [pull-k8sio-cip](https://github.com/kubernetes/test-infra/blob/master/config/jobs/kubernetes/sig-k8s-infra/releng/artifact-promotion-presubmits.yaml) `thin-manifest-dir` flags to point to the new folder


### Test Plan

This is not applicable.

<!--
**Note:** *Not required until targeted at a release.*
The goal is to ensure that we don't accept enhancements with inadequate testing.

All code is expected to have adequate tests (eventually with coverage
expectations). Please adhere to the [Kubernetes testing guidelines][testing-guidelines]
when drafting this test plan.

[testing-guidelines]: https://git.k8s.io/community/contributors/devel/sig-testing/testing.md
-->

### Graduation Criteria

This is not applicable.

### Upgrade / Downgrade Strategy

When users upgrade to various kubernetes versions that use the new image registry, they will be able to pull images from the new
registry.

<!--
If applicable, how will the component be upgraded and downgraded? Make sure
this is in the test plan.

Consider the following in developing an upgrade/downgrade strategy for this
enhancement:
- What changes (in invocations, configurations, API use, etc.) is an existing
  cluster required to make on upgrade, in order to maintain previous behavior?
- What changes (in invocations, configurations, API use, etc.) is an existing
  cluster required to make on upgrade, in order to make use of the enhancement?
-->

### Version Skew Strategy

This is not applicable.

## Communication Plan

This is a major change and requires an appropiate communication plan.


## Production Readiness Review Questionnaire

This is not applicable.

## Implementation History

<!--
Major milestones in the lifecycle of a KEP should be tracked in this section.
Major milestones might include:
- the `Summary` and `Motivation` sections being merged, signaling SIG acceptance
- the `Proposal` section being merged, signaling agreement on a proposed design
- the date implementation started
- the first Kubernetes release where an initial version of the KEP was available
- the version of Kubernetes where the KEP graduated to general availability
- when the KEP was retired or superseded
-->

## Drawbacks

This is not applicable.

<!--
Why should this KEP _not_ be implemented?
-->

## Alternatives

We keep pushing new images to the old registry.
<!--
What other approaches did you consider, and why did you rule them out? These do
not need to be as detailed as the proposal, but should include enough
information to express the idea and why it was not acceptable.
-->

## Infrastructure Needed (Optional)

None as it has already been deployed.
