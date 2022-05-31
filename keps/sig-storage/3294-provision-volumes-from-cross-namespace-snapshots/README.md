<!--
**Note:** When your KEP is complete, all of these comment blocks should be removed.

To get started with this template:

- [ ] **Pick a hosting SIG.**
  Make sure that the problem space is something the SIG is interested in taking
  up. KEPs should not be checked in without a sponsoring SIG.
- [ ] **Create an issue in kubernetes/enhancements**
  When filing an enhancement tracking issue, please make sure to complete all
  fields in that template. One of the fields asks for a link to the KEP. You
  can leave that blank until this KEP is filed, and then go back to the
  enhancement and add the link.
- [ ] **Make a copy of this template directory.**
  Copy this template into the owning SIG's directory and name it
  `NNNN-short-descriptive-title`, where `NNNN` is the issue number (with no
  leading-zero padding) assigned to your enhancement above.
- [ ] **Fill out as much of the kep.yaml file as you can.**
  At minimum, you should fill in the "Title", "Authors", "Owning-sig",
  "Status", and date-related fields.
- [ ] **Fill out this file as best you can.**
  At minimum, you should fill in the "Summary" and "Motivation" sections.
  These should be easy if you've preflighted the idea of the KEP with the
  appropriate SIG(s).
- [ ] **Create a PR for this KEP.**
  Assign it to people in the SIG who are sponsoring this process.
- [ ] **Merge early and iterate.**
  Avoid getting hung up on specific details and instead aim to get the goals of
  the KEP clarified and merged quickly. The best way to do this is to just
  start with the high-level sections and fill out details incrementally in
  subsequent PRs.

Just because a KEP is merged does not mean it is complete or approved. Any KEP
marked as `provisional` is a working document and subject to change. You can
denote sections that are under active debate as follows:

```
<<[UNRESOLVED optional short context or usernames ]>>
Stuff that is being argued.
<<[/UNRESOLVED]>>
```

When editing KEPS, aim for tightly-scoped, single-topic PRs to keep discussions
focused. If you disagree with what is already in a document, open a new PR
with suggested changes.

One KEP corresponds to one "feature" or "enhancement" for its whole lifecycle.
You do not need a new KEP to move from beta to GA, for example. If
new details emerge that belong in the KEP, edit the KEP. Once a feature has become
"implemented", major changes should get new KEPs.

The canonical place for the latest set of instructions (and the likely source
of this file) is [here](/keps/NNNN-kep-template/README.md).

**Note:** Any PRs to move a KEP to `implementable`, or significant changes once
it is marked `implementable`, must be approved by each of the KEP approvers.
If none of those approvers are still appropriate, then changes to that list
should be approved by the remaining approvers and/or the owning SIG (or
SIG Architecture for cross-cutting KEPs).
-->
# KEP-3294: Provision volumes from cross-namespace snapshots

<!--
This is the title of your KEP. Keep it short, simple, and descriptive. A good
title can help communicate what the KEP is and should be considered as part of
any review.
-->

<!--
A table of contents is helpful for quickly jumping to sections of a KEP and for
highlighting any additional information provided beyond the standard KEP
template.

Ensure the TOC is wrapped with
  <code>&lt;!-- toc --&rt;&lt;!-- /toc --&rt;</code>
tags, and then generate with `hack/update-toc.sh`.
-->

<!-- toc -->
- [Release Signoff Checklist](#release-signoff-checklist)
- [Summary](#summary)
- [Motivation](#motivation)
  - [Goals](#goals)
  - [Non-Goals](#non-goals)
- [Proposal](#proposal)
  - [User Stories (Optional)](#user-stories-optional)
    - [Story 1](#story-1)
    - [Story 2](#story-2)
  - [Notes/Constraints/Caveats (Optional)](#notesconstraintscaveats-optional)
    - [Provisioning PVCs from cross-namespace PVCs](#provisioning-pvcs-from-cross-namespace-pvcs)
  - [Risks and Mitigations](#risks-and-mitigations)
    - [Secret Handling](#secret-handling)
    - [Security](#security)
- [Design Details](#design-details)
  - [Example flow of how this proposal works](#example-flow-of-how-this-proposal-works)
  - [API](#api)
  - [Populator implementation](#populator-implementation)
    - [(a) inside the existing CSI external-provisioner](#a-inside-the-existing-csi-external-provisioner)
    - [(b) as a separate populator](#b-as-a-separate-populator)
      - [(1) Populate data from snapshot to provisioned PV](#1-populate-data-from-snapshot-to-provisioned-pv)
      - [(2) Provision PV with data via CSI call](#2-provision-pv-with-data-via-csi-call)
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

<!--
This section is incredibly important for producing high-quality, user-focused
documentation such as release notes or a development roadmap. It should be
possible to collect this information before implementation begins, in order to
avoid requiring implementors to split their attention between writing release
notes and implementing the feature itself. KEP editors and SIG Docs
should help to ensure that the tone and content of the `Summary` section is
useful for a wide audience.

A good summary is probably at least a paragraph in length.

Both in this section and below, follow the guidelines of the [documentation
style guide]. In particular, wrap lines to a reasonable length, to make it
easier for reviewers to cite specific portions, and to minimize diff churn on
updates.

[documentation style guide]: https://github.com/kubernetes/community/blob/master/contributors/guide/style-guide.md
-->

This KEP proposes a method for provisioning volumes from cross-namespace snapshots.

## Motivation

<!--
This section is for explicitly listing the motivation, goals, and non-goals of
this KEP.  Describe why the change is important and the benefits to users. The
motivation section can optionally provide links to [experience reports] to
demonstrate the interest in a KEP within the wider Kubernetes community.

[experience reports]: https://github.com/golang/go/wiki/ExperienceReports
-->

By using [volume snapshots feature](https://kubernetes.io/docs/concepts/storage/volume-snapshots/), users can provision volumes from snapshots.
However, it only works for the `VolumeSnapshot` in the same namespace,
therefore users can't provision a persistent volume claim in one namespace from a `VolumeSnapshot` in the other namespace.
On the other hand, as discussed in other KEP PRs (https://github.com/kubernetes/enhancements/pull/643,
https://github.com/kubernetes/enhancements/pull/1112, and https://github.com/kubernetes/enhancements/pull/2849), there are use cases that require to share the `VolumeSnapshot` across namespaces.
For such use cases, this KEP proposes a method for provisioning volumes from cross-namespace snapshots.

### Goals

<!--
List the specific goals of the KEP. What is it trying to achieve? How will we
know that this has succeeded?
-->
- Provision of PVCs from `VolumeSnapshot`s in other namespaces

### Non-Goals

<!--
What is out of scope for this KEP? Listing non-goals helps to focus discussion
and make progress.
-->
- Provision of PVCs from PVCs in other namespaces (Please also see [Provisioning PVCs from cross-namespace PVCs](#provisioning-pvcs-from-cross-namespace-pvcs))
- Copy or move of `VolumeSnapshot`s to other namespaces (Please also see [Alternatives](#alternatives))
- Clone of `VolumeSnapshotContent`s 

## Proposal

<!--
This is where we get down to the specifics of what the proposal actually is.
This should have enough detail that reviewers can understand exactly what
you're proposing, but should not include things like API designs or
implementation. What is the desired outcome and how do we measure success?.
The "Design Details" section below is for the real
nitty-gritty.
-->

Define an API to specify a cross-namespace `VolumeSnapshot` as a `DataSourceRef` of a PVC and implement a generic populator for the API.

- To specify a non-standard API as a `DataSourceRef` of a PVC, [AnyVolumeDataSource feature](https://kubernetes.io/blog/2021/08/30/volume-populators-redesigned/) is used,
- To specify a cross-namespace `VolumeSnapshot`, a new `VolumeSnapshotLink` CRD is introduced (Please also see [API](#api)),
- To restrict only allowed `VolumeSnapshot` to be consumed from other namespaces, [`ReferencePolicy` CRD](https://gateway-api.sigs.k8s.io/v1alpha2/references/spec/#gateway.networking.k8s.io/v1alpha2.ReferencePolicy) is used,
- To actually populate a PV from a `VolumeSnapshot` referenced from `VolumeSnapshotLink` CRD, a populator for each CSI driver is used,
- As a reference populator implementation, [CSI external provisioner](https://github.com/kubernetes-csi/external-provisioner) is extended to handle the `VolumeSnapshotLink` CRD (Please also see [Populator implementation](#populator-implementation)).

An initial discussion of this idea can be found [here](https://github.com/kubernetes/enhancements/pull/2849#issuecomment-949929595) and PoC implementation can be found [here](https://github.com/kubernetes/enhancements/pull/2849#issuecomment-958208039).

### User Stories (Optional)

<!--
Detail the things that people will be able to do if this KEP is implemented.
Include as much detail as possible so that people can understand the "how" of
the system. The goal here is to make this feel real for users without getting
bogged down.
-->

#### Story 1

`VolumeSnapshot`s for PVCs in prod namespace are taken on a regular basis, PVCs are created from the `VolumeSnapshot`s in other test namespaces for testing.

#### Story 2

The same `VolumeSnapshot`s are expected to be consumed as golden images from multiple namespaces. Using PVs as [VM images for KubeVirt](https://kubevirt.io/2020/KubeVirt-VM-Image-Usage-Patterns.html) is one of the examples of this use case.

### Notes/Constraints/Caveats (Optional)

<!--
What are the caveats to the proposal?
What are some important details that didn't come across above?
Go in to as much detail as necessary here.
This might be a good place to talk about core concepts and how they relate.
-->

#### Provisioning PVCs from cross-namespace PVCs

The conclusion of the original discussion ([here](https://docs.google.com/document/d/17H1k4lqdtJwZSjNRaQue-FhMhyk14JA_MoURpoxha5Q/edit#bookmark=id.nj4e1ocn8b23) and [here](https://docs.google.com/document/d/17H1k4lqdtJwZSjNRaQue-FhMhyk14JA_MoURpoxha5Q/edit#bookmark=id.h1eqongxseo)) on transfer feature was that we should avoid implementing transfer of PVCs, because there will be more race conditions for PVCs than snapshots.
However, we might have a room to reconsider if this cross-namespace-provision approach can solve the issue of race for PVCs, although transfer approach can't seem to resolve the issue easily.

### Risks and Mitigations

<!--
What are the risks of this proposal, and how do we mitigate? Think broadly.
For example, consider both security and how this will impact the larger
Kubernetes ecosystem.

How will security be reviewed, and by whom?

How will UX be reviewed, and by whom?

Consider including folks who also work outside the SIG or subproject.
-->

#### Secret Handling

Unlike transfer feature, this idea doesn't need to involve any transfers of Secert, therefore there will be no issue on Secret handling.
From a populator, Secrets are only referenced through snapshots that exist in the same namespace (As commented [here](https://github.com/kubernetes/enhancements/pull/2849#issuecomment-962168202), depending on the driver implementation, there may be very little chance that some CSI drivers won't work well in a very rare situation. However, such drivers can avoid this issue separately, by turning off this feature, implementing their own populator, and so on).

#### Security

By using [`ReferencePolicy`](https://gateway-api.sigs.k8s.io/concepts/security-model/#2-referencepolicy), only allowed snapshots can be accessed beyond the namespace boundary (Please also see [original  discussion on security](https://github.com/kubernetes/enhancements/pull/2849#issuecomment-919107307)).
Therefore, no malicious user will be able to access to prohibited snapshots.

## Design Details

<!--
This section should contain enough information that the specifics of your
change are understandable. This may include API specs (though not always
required) or even code snippets. If there's any ambiguity about HOW your
proposal will be implemented, this is the place to discuss them.
-->

### Example flow of how this proposal works

Let's use [Story 1](#story-1) as an example and let's assume the following:
- There are two namespaces, prod and test,
- Alice manages the prod namespace and Bob manages the test namespace,
- Alice would like to allow `VolumeSnapshot` foo-backup in the prod namespace to be consumed in the test namespace for testing,
- Bob would like to create a PV for PVC foo-testing in the test namespace from the `VolumeSnapshot` foo-backup in the prod namespace.

Once this proposal is implemented, it can be achieved by doing the following steps:

1. In the prod namespace, Alice creates a `ReferencePolicy` bar that allows referencing to the `VolumeSnapshot` foo-backup in the prod namespace from any `VolumeSnapshotLinks` in the test namespace,
    ```yaml
    apiVersion: gateway.networking.k8s.io/v1alpha2
    kind: ReferencePolicy
    metadata:
      name: bar
      namespace: prod
    spec:
      from:
      - group: snapshot.storage.k8s.io/v1alpha1
        kind: VolumeSnapshotLink
        namespace: test
      to:
      - group: snapshot.storage.k8s.io/v1
        kind: VolumeSnapshot
        name: foo-backup
    ```
2. In the test namespace, Bob creates a `VolumeSnapshotLink` foo-link that references the `VolumeSnapshot` foo-backup in the prod namespace as a source,
    ```yaml
    apiVersion: snapshot.storage.k8s.io/v1alpha1
    kind: VolumeSnapshotLink
    metadata:
      name: foo-link
      namespace: test
    spec:
      source:
        name: foo-backup
        namespace: prod
    ```
3. In the test namespace, Bob creates a `PersistentVolumeClaim` foo-testing that references the `VolumeSnapshotLink` foo-link as a data source,
    ```yaml
    apiVersion: v1
    kind: PersistentVolumeClaim
    metadata:
      name: foo-testing
      namespace: test
    spec:
      accessModes:
      - ReadWriteOnce
      resources:
        requests:
          storage: 10Mi
      dataSourceRef:
        apiGroup: snapshot.storage.k8s.io/v1alpha1
        kind: VolumeSnapshotLink
        name: foo-link
      volumeMode: Filesystem
    ```
4. Once the populator finds a `VolumeSnapshotLink` is specified as `dataSourceRef`, it checks all `ReferencePolicys` in `VolumeSnapshotLink.spec.source.namespace` to see if populating the `VolumeSnapshotLink.spec.source` is allowed. If it is allowed, the populator populates the volume.

### API

A new `VolumeSnapshotLink` CRD is introduced in `snapshot.storage.k8s.io` API group:

```golang
type VolumeSnapshotLink struct {
	metav1.TypeMeta `json:",inline"`
	// +optional
	metav1.ObjectMeta `json:"metadata,omitempty" protobuf:"bytes,1,opt,name=metadata"`

	Spec VolumeSnapshotLinkSpec `json:"spec" protobuf:"bytes,2,opt,name=spec"`
}

type VolumeSnapshotLinkList struct {
	metav1.TypeMeta `json:",inline"`
	// +optional
	metav1.ListMeta `json:"metadata,omitempty" protobuf:"bytes,1,opt,name=metadata"`

	Items []VolumeSnapshotLink `json:"items" protobuf:"bytes,2,rep,name=items"`
}

// VolumeSnapshotLinkSpec describes attributes of a volume snapshot link.
type VolumeSnapshotLinkSpec struct {
	// This field is immutable after creation.
	Source VolumeSnapshotLinkSource `json:"source" protobuf:"bytes,1,opt,name=source"`
}

// VolumeSnapshotLinkSource specifies a reference to VolumeSnapshot.
type VolumeSnapshotLinkSource struct {
	Name string `json:"name" protobuf:"bytes,1,opt,name=name"`
	Namespace string `json:"namespace" protobuf:"bytes,2,opt,name=namespace"`
}
```

### Populator implementation

The populator logic can be implemented either [(a) inside the existing CSI external-provisioner](#a-inside-the-existing-csi-external-provisioner) or [(b) as a separate populator](#b-as-a-separate-populator).
Cluster admins can choose which implementation to be used per CSI driver basis.
As a reference implementation, only (a) will be implemented in the community.

Regardless of the implementation,
- `VolumeSnapshotLink` CRD and `ReferencePolicy` CRD must exist in the cluster before the populator is deployed.
- `VolumePopulator` CRD to allow popluating from `VolumeSnapshotLink` CRD needs to be created to enable this feature, as AnyVolumeDataSource feature defines. The `VolumePopulator` CRD needed for this feature will be as follows:
```yaml
kind: VolumePopulator
apiVersion: populator.storage.k8s.io/v1beta1
metadata:
  name: volumesnapshotlink
sourceKind:
  group: snapshot.storage.k8s.io
  kind: VolumeSnapshotLink
```

#### (a) inside the existing CSI external-provisioner

Once populator is implemented inside the existing CSI external-provisioner, the CSI external provisioner:
- Handles `VolumeSnapshotLink` CRD and `ReferencePolicy` CRD,
- Checks if `VolumeSnapshotLink` is specified as `DataSourceRef`:
  - If specified, check if the access to the `VolumeSnapshot` referenced by the `VolumeSnapshotLink` is allowed by any `ReferencePolicy`s:
    - If allowed, use the `VolumeSnapshot` as a SnapshotSource to pass to the CSI driver for provision.

To enable this feature in CSI external provisioner, `--cross-namespace-snapshot=true`
command line flag needs to be passed to the provisioner for each CSI plugin.

#### (b) as a separate populator

There will be two approaches to implement as a separate populator:
- [(1) Populate data from snapshot to provisioned PV](#1-populate-data-from-snapshot-to-provisioned-pv)
- [(2) Provision PV with data via CSI call](#2-provision-pv-with-data-via-csi-call)

##### (1) Populate data from snapshot to provisioned PV

This is a straightforward implementation that AnyVolumeDataSource feature defines.
Developers will be able to utilize lib-volume-populator to implement this way.
One of the challenges to achieve it will be how to actually copy the data from a snapshot in one namespace to an already provisioned PV that will need to be bound to a PVC in the other namespace.

A naive implementation will be:
1. Create another PV from the snapshot in the snapshot's namespace,
2. Copy the data from the PV to somewhere accessible from any namespaces,
3. Copy the data in step 2 to the originally intended PV,
4. Delete the temporary data in step 1 and step 2.

If the naive implementation is used, unintended transient states, for example a temporary PVC in the snapshot namespace, may be visible to users.
Also, there may be performance issues depending on where and how data is copied.

On the other hand, althoguh it completely depends on the implementation, this approach can have advantages, like the ability to populate volumes from snapshot across different CSI drivers or the ability to efficiently copy data by using CSI driver specific way.

There will be no generic way to implement by using this approach, because the implementations rely too much on backup tools or CSI drivers.
Therefore no community implementation of this approach will be provided.

##### (2) Provision PV with data via CSI call

Current CSI external provisioners provision volume regardless of the data source, therefore populators need to populate data to already provisioned PVs.
However, this behavior may be changed and provisioners may offload provisioning to populators for PV with `VolumeSnapshotLink` CRD data source.

The implementation of provisioner and populator of this approach will be as follows:

- Provisioner:
  - Handles `VolumeSnapshotLink` CRD,
  - Checks if `VolumeSnapshotLink` is specified as `DataSourceRef`:
    - If specified, skip provisioning  the volume

- Populator:
  - Handles `VolumeSnapshotLink` CRD and `ReferencePolicy` CRD,
  - Checks if `VolumeSnapshotLink` is specified as `DataSourceRef`:
    - If specified, check if the access to the `VolumeSnapshot` referenced by the `VolumeSnapshotLink` is allowed by any `ReferencePolicy`s:
      - If allowed, use the `VolumeSnapshot` as a SnapshotSource to pass to the CSI driver for provision.

The above implementation is just separating the logics in approach (a) to two components, and it won't help improve efficiency nor simplify implementations.
Therefore, the description in this section is just for discussion purpose and won't be implemented.

### Test Plan

<!--
**Note:** *Not required until targeted at a release.*
The goal is to ensure that we don't accept enhancements with inadequate testing.

All code is expected to have adequate tests (eventually with coverage
expectations). Please adhere to the [Kubernetes testing guidelines][testing-guidelines]
when drafting this test plan.

[testing-guidelines]: https://git.k8s.io/community/contributors/devel/sig-testing/testing.md
-->

[x] I/we understand the owners of the involved components may require updates to
existing tests to make this code solid enough prior to committing the changes necessary
to implement this enhancement.

##### Prerequisite testing updates

<!--
Based on reviewers feedback describe what additional tests need to be added prior
implementing this enhancement to ensure the enhancements have also solid foundations.
-->

##### Unit tests

<!--
In principle every added code should have complete unit test coverage, so providing
the exact set of tests will not bring additional value.
However, if complete unit test coverage is not possible, explain the reason of it
together with explanation why this is acceptable.
-->

<!--
Additionally, for Alpha try to enumerate the core package you will be touching
to implement this enhancement and provide the current unit coverage for those
in the form of:
- <package>: <date> - <current test coverage>
The data can be easily read from:
https://testgrid.k8s.io/sig-testing-canaries#ci-kubernetes-coverage-unit
This can inform certain test coverage improvements that we want to do before
extending the production code to implement this enhancement.
-->

- external-provisioner/pkg/controller/: 2022/5/31 - 81.1%

##### Integration tests

<!--
This question should be filled when targeting a release.
For Alpha, describe what tests will be added to ensure proper quality of the enhancement.
For Beta and GA, add links to added tests together with links to k8s-triage for those tests:
https://storage.googleapis.com/k8s-triage/index.html
-->

- No integration tests for csi external provisioner.

##### e2e tests

<!--
This question should be filled when targeting a release.
For Alpha, describe what tests will be added to ensure proper quality of the enhancement.
For Beta and GA, add links to added tests together with links to k8s-triage for those tests:
https://storage.googleapis.com/k8s-triage/index.html
We expect no non-infra related flakes in the last month as a GA graduation criteria.
-->

- Verify that PV is provisioned from VS in other namsepace if allowed by ReferencePolicy: <link to test coverage>
- Verify that PV isn't provisioned from VS in other namsepace if not allowed by ReferencePolicy: <link to test coverage>

### Graduation Criteria

<!--
**Note:** *Not required until targeted at a release.*

Define graduation milestones.

These may be defined in terms of API maturity, [feature gate] graduations, or as
something else. The KEP should keep this high-level with a focus on what
signals will be looked at to determine graduation.

Consider the following in developing the graduation criteria for this enhancement:
- [Maturity levels (`alpha`, `beta`, `stable`)][maturity-levels]
- [Feature gate][feature gate] lifecycle
- [Deprecation policy][deprecation-policy]

Clearly define what graduation means by either linking to the [API doc
definition](https://kubernetes.io/docs/concepts/overview/kubernetes-api/#api-versioning)
or by redefining what graduation means.

In general we try to use the same stages (alpha, beta, GA), regardless of how the
functionality is accessed.

[feature gate]: https://git.k8s.io/community/contributors/devel/sig-architecture/feature-gates.md
[maturity-levels]: https://git.k8s.io/community/contributors/devel/sig-architecture/api_changes.md#alpha-beta-and-stable-versions
[deprecation-policy]: https://kubernetes.io/docs/reference/using-api/deprecation-policy/

Below are some examples to consider, in addition to the aforementioned [maturity levels][maturity-levels].
-->

#### Alpha

- Feature implemented behind a non-default command line flag of CSI external-provisioner
- Initial e2e tests completed and enabled

#### Beta

- Gather feedback from developers and surveys
- Additional tests are in Testgrid and linked in KEP

#### GA

- Allowing time for feedback

**Note:** Generally we also wait at least two releases between beta and
GA/stable, because there's no opportunity for user feedback, or even bug reports,
in back-to-back releases.

**For non-optional features moving to GA, the graduation criteria must include
[conformance tests].**

[conformance tests]: https://git.k8s.io/community/contributors/devel/sig-architecture/conformance-tests.md

### Upgrade / Downgrade Strategy

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

- Upgrade:
  - Method: Do both of the below operations:
    - Specify `--cross-namespace-snapshot=true` command line flag of CSI external-provisioner
    - Create `VolumePopulator` CRD to allow popluating from `VolumeSnapshotLink` CRD
  - Behavior:
    - Provisioning volumes from snapshots in other namespaces is enabled.
- Downgrade:
  - Method: Do both of the below operations:
    - Specify `--cross-namespace-snapshot=false` command line flag of CSI external-provisioner
    - Delete `VolumePopulator` CRD to deny popluating from `VolumeSnapshotLink` CRD
  - Behavior:
    - Provisioning volumes from snapshots in other namespaces is disabled.

### Version Skew Strategy

<!--
If applicable, how will the component handle version skew with other
components? What are the guarantees? Make sure this is in the test plan.

Consider the following in developing a version skew strategy for this
enhancement:
- Does this enhancement involve coordinating behavior in the control plane and
  in the kubelet? How does an n-2 kubelet without this feature available behave
  when this feature is used?
- Will any other components on the node change? For example, changes to CSI,
  CRI or CNI may require updating that component before the kubelet.
-->

## Production Readiness Review Questionnaire

<!--

Production readiness reviews are intended to ensure that features merging into
Kubernetes are observable, scalable and supportable; can be safely operated in
production environments, and can be disabled or rolled back in the event they
cause increased failures in production. See more in the PRR KEP at
https://git.k8s.io/enhancements/keps/sig-architecture/1194-prod-readiness.

The production readiness review questionnaire must be completed and approved
for the KEP to move to `implementable` status and be included in the release.

In some cases, the questions below should also have answers in `kep.yaml`. This
is to enable automation to verify the presence of the review, and to reduce review
burden and latency.

The KEP must have a approver from the
[`prod-readiness-approvers`](http://git.k8s.io/enhancements/OWNERS_ALIASES)
team. Please reach out on the
[#prod-readiness](https://kubernetes.slack.com/archives/CPNHUMN74) channel if
you need any help or guidance.
-->

### Feature Enablement and Rollback

<!--
This section must be completed when targeting alpha to a release.
-->

###### How can this feature be enabled / disabled in a live cluster?

<!--
Pick one of these and delete the rest.

Documentation is available on [feature gate lifecycle] and expectations, as
well as the [existing list] of feature gates.

[feature gate lifecycle]: https://git.k8s.io/community/contributors/devel/sig-architecture/feature-gates.md
[existing list]: https://kubernetes.io/docs/reference/command-line-tools-reference/feature-gates/
-->

- [x] Other
  - Describe the mechanism:
  - Will enabling / disabling the feature require downtime of the control
    plane? No, it won't require downtime of the entire control plane. However, it will require a downtime of each provisioner whose enablement is being changed.
  - Will enabling / disabling the feature require downtime or reprovisioning
    of a node? (Do not assume `Dynamic Kubelet Config` feature is enabled). No.

###### Does enabling the feature change any default behavior?

<!--
Any change of default behavior may be surprising to users or break existing
automations, so be extremely careful here.
-->

Yes, `VolumeSnapshotLink` CRD can be used as a `DataSourceRef` for PVC.

###### Can the feature be disabled once it has been enabled (i.e. can we roll back the enablement)?

<!--
Describe the consequences on existing workloads (e.g., if this is a runtime
feature, can it break the existing applications?).

Feature gates are typically disabled by setting the flag to `false` and
restarting the component. No other changes should be necessary to disable the
feature.

NOTE: Also set `disable-supported` to `true` or `false` in `kep.yaml`.
-->

Yes, by specifying `--cross-namespace-snapshot=false` command line flag of CSI external-provisioner, and deleting `VolumePopulator` CRD to deny popluating from `VolumeSnapshotLink` CRD.

###### What happens if we reenable the feature if it was previously rolled back?

`VolumeSnapshotLink` CRD can be used as a `DataSourceRef` for PVC, again.

###### Are there any tests for feature enablement/disablement?

<!--
The e2e framework does not currently support enabling or disabling feature
gates. However, unit tests in each component dealing with managing data, created
with and without the feature, are necessary. At the very least, think about
conversion tests if API types are being modified.

Additionally, for features that are introducing a new API field, unit tests that
are exercising the `switch` of feature gate itself (what happens if I disable a
feature gate after having objects written with the new field) are also critical.
You can take a look at one potential example of such test in:
https://github.com/kubernetes/kubernetes/pull/97058/files#diff-7826f7adbc1996a05ab52e3f5f02429e94b68ce6bce0dc534d1be636154fded3R246-R282
-->

Yes, unit tests cover scenarios where the feature is disabled or enabled.

### Rollout, Upgrade and Rollback Planning

<!--
This section must be completed when targeting beta to a release.
-->

###### How can a rollout or rollback fail? Can it impact already running workloads?

<!--
Try to be as paranoid as possible - e.g., what if some components will restart
mid-rollout?

Be sure to consider highly-available clusters, where, for example,
feature flags will be enabled on some API servers and not others during the
rollout. Similarly, consider large clusters and how enablement/disablement
will rollout across nodes.
-->

###### What specific metrics should inform a rollback?

<!--
What signals should users be paying attention to when the feature is young
that might indicate a serious problem?
-->

###### Were upgrade and rollback tested? Was the upgrade->downgrade->upgrade path tested?

<!--
Describe manual testing that was done and the outcomes.
Longer term, we may want to require automated upgrade/rollback tests, but we
are missing a bunch of machinery and tooling and can't do that now.
-->

###### Is the rollout accompanied by any deprecations and/or removals of features, APIs, fields of API types, flags, etc.?

<!--
Even if applying deprecation policies, they may still surprise some users.
-->

### Monitoring Requirements

<!--
This section must be completed when targeting beta to a release.

For GA, this section is required: approvers should be able to confirm the
previous answers based on experience in the field.
-->

###### How can an operator determine if the feature is in use by workloads?

<!--
Ideally, this should be a metric. Operations against the Kubernetes API (e.g.,
checking if there are objects with field X set) may be a last resort. Avoid
logs or events for this purpose.
-->

###### How can someone using this feature know that it is working for their instance?

<!--
For instance, if this is a pod-related feature, it should be possible to determine if the feature is functioning properly
for each individual pod.
Pick one more of these and delete the rest.
Please describe all items visible to end users below with sufficient detail so that they can verify correct enablement
and operation of this feature.
Recall that end users cannot usually observe component logs or access metrics.
-->

- [ ] Events
  - Event Reason: 
- [x] API .status
  - Condition name: `Bound` for a PV that is provisioned from a PVC referencing `VolumeSnapshotLink`
  - Other field: 
- [x] Other (treat as last resort)
  - Details: Check if a `VolumePopulator` CRD to allow popluating from `VolumeSnapshotLink` CRD exists.

###### What are the reasonable SLOs (Service Level Objectives) for the enhancement?

<!--
This is your opportunity to define what "normal" quality of service looks like
for a feature.

It's impossible to provide comprehensive guidance, but at the very
high level (needs more precise definitions) those may be things like:
  - per-day percentage of API calls finishing with 5XX errors <= 1%
  - 99% percentile over day of absolute value from (job creation time minus expected
    job creation time) for cron job <= 10%
  - 99.9% of /health requests per day finish with 200 code

These goals will help you determine what you need to measure (SLIs) in the next
question.
-->

###### What are the SLIs (Service Level Indicators) an operator can use to determine the health of the service?

<!--
Pick one more of these and delete the rest.
-->

- [x] Metrics
  - Metric name: TBD (Need to discuss if existing metrics for "claims" queue is sufficient).
  - [Optional] Aggregation method: prometheus
  - Components exposing the metric: CSI external-provisioner for each CSI plugin

###### Are there any missing metrics that would be useful to have to improve observability of this feature?

<!--
Describe the metrics themselves and the reasons why they weren't added (e.g., cost,
implementation difficulties, etc.).
-->

Existing metrics only provides number of claims remains in the queue and number of retries for the claims.
To identify what kind of data source was specified for the claim with errors, per data source type metrics may be needed.

### Dependencies

<!--
This section must be completed when targeting beta to a release.
-->

###### Does this feature depend on any specific services running in the cluster?

<!--
Think about both cluster-level services (e.g. metrics-server) as well
as node-level agents (e.g. specific version of CRI). Focus on external or
optional services that are needed. For example, if this feature depends on
a cloud provider API, or upon an external software-defined storage or network
control plane.

For each of these, fill in the following—thinking about running existing user workloads
and creating new ones, as well as about cluster-level services (e.g. DNS):
  - [Dependency name]
    - Usage description:
      - Impact of its outage on the feature:
      - Impact of its degraded performance or high-error rates on the feature:
-->

- Features:
  - CSI feature ([GA](https://kubernetes.io/blog/2019/01/15/container-storage-interface-ga/) in v1.13)
  - AnyVolumeDataSource feature ([Beta](https://github.com/kubernetes/kubernetes/blob/master/CHANGELOG/CHANGELOG-1.24.md#api-change) in v1.24)

- Services:
  - CSI external provisioner and CSI plugins
    - Usage description:
      - Impact of its outage on the feature: The PV isn't provisioned.
      - Impact of its degraded performance or high-error rates on the feature: Provision of PV becomes slow or error.
  - volume-data-source-validator
    - Usage description:
      - Impact of its outage on the feature: The PV isn't populated.
      - Impact of its degraded performance or high-error rates on the feature: Populating PV becomes slow or error.

### Scalability

<!--
For alpha, this section is encouraged: reviewers should consider these questions
and attempt to answer them.

For beta, this section is required: reviewers must answer these questions.

For GA, this section is required: approvers should be able to confirm the
previous answers based on experience in the field.
-->

###### Will enabling / using this feature result in any new API calls?

<!--
Describe them, providing:
  - API call type (e.g. PATCH pods)
  - estimated throughput
  - originating component(s) (e.g. Kubelet, Feature-X-controller)
Focusing mostly on:
  - components listing and/or watching resources they didn't before
  - API calls that may be triggered by changes of some Kubernetes resources
    (e.g. update of object X triggers new updates of object Y)
  - periodic API calls to reconcile state (e.g. periodic fetching state,
    heartbeats, leader election, etc.)
-->
- GET `VolumeSnapshotLink` API call:
  - originating component(s): CSI external-provisioner
  - this API call is triggered once in each [`Provision` call](https://github.com/kubernetes-csi/external-provisioner/blob/master/pkg/controller/controller.go#L719) in CSI external-provisioner when `VolumeSnapshotLink` is referenced from PVC.
- GET(LIST) `ReferencePolicy` API call:
  - originating component(s): CSI external-provisioner
  - this API call is triggered once in each [`Provision` call](https://github.com/kubernetes-csi/external-provisioner/blob/master/pkg/controller/controller.go#L719) in CSI external-provisioner when `VolumeSnapshotLink` is referenced from PVC.

###### Will enabling / using this feature result in introducing new API types?

<!--
Describe them, providing:
  - API type
  - Supported number of objects per cluster
  - Supported number of objects per namespace (for namespace-scoped objects)
-->

- API type: `VolumeSnapshotLink` CRD
- Supported number of objects per namespace (for namespace-scoped objects): TBD
(Estimated maximum number is the number of `VolumeSnapshot`s that should be shared across namespace or
the number of PVs per namespace).

###### Will enabling / using this feature result in any new calls to the cloud provider?

<!--
Describe them, providing:
  - Which API(s):
  - Estimated increase:
-->

No.

###### Will enabling / using this feature result in increasing size or count of the existing API objects?

<!--
Describe them, providing:
  - API type(s):
  - Estimated increase in size: (e.g., new annotation of size 32B)
  - Estimated amount of new objects: (e.g., new Object X for every existing Pod)
-->

No.

###### Will enabling / using this feature result in increasing time taken by any operations covered by existing SLIs/SLOs?

<!--
Look at the [existing SLIs/SLOs].

Think about adding additional work or introducing new steps in between
(e.g. need to do X to start a container), etc. Please describe the details.

[existing SLIs/SLOs]: https://git.k8s.io/community/sig-scalability/slos/slos.md#kubernetes-slisslos
-->

No.

Currently, no SLIs/SLOs are defined for PV provisioning, but no performance change is expected for existing PV provisioning by this feature.
For a new provisioning from cross-namespace snapshot, it may take more time than existing PV provisioning due to the extra API calls.

###### Will enabling / using this feature result in non-negligible increase of resource usage (CPU, RAM, disk, IO, ...) in any components?

<!--
Things to keep in mind include: additional in-memory state, additional
non-trivial computations, excessive access to disks (including increased log
volume), significant amount of data sent and/or received over network, etc.
This through this both in small and large cases, again with respect to the
[supported limits].

[supported limits]: https://git.k8s.io/community//sig-scalability/configs-and-limits/thresholds.md
-->

No.

### Troubleshooting

<!--
This section must be completed when targeting beta to a release.

For GA, this section is required: approvers should be able to confirm the
previous answers based on experience in the field.

The Troubleshooting section currently serves the `Playbook` role. We may consider
splitting it into a dedicated `Playbook` document (potentially with some monitoring
details). For now, we leave it here.
-->

###### How does this feature react if the API server and/or etcd is unavailable?

Existing PV provisioning also fails.

###### What are other known failure modes?

<!--
For each of them, fill in the following information by copying the below template:
  - [Failure mode brief description]
    - Detection: How can it be detected via metrics? Stated another way:
      how can an operator troubleshoot without logging into a master or worker node?
    - Mitigations: What can be done to stop the bleeding, especially for already
      running user workloads?
    - Diagnostics: What are the useful log messages and their required logging
      levels that could help debug the issue?
      Not required until feature graduated to beta.
    - Testing: Are there any tests for failure mode? If not, describe why.
-->

###### What steps should be taken if SLOs are not being met to determine the problem?

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

<!--
Why should this KEP _not_ be implemented?
-->

## Alternatives

<!--
What other approaches did you consider, and why did you rule them out? These do
not need to be as detailed as the proposal, but should include enough
information to express the idea and why it was not acceptable.
-->

- Implement transfer feature for `VolumeSnapshot`:
  - Pros:
    - Can have more control over the transferred `VolumeSnapshot`, like modifying and deleting
    - Can potentially be used to directly clone the `VolumeSnapshot` to other namespaces for [backup use case](https://github.com/kubernetes/enhancements/pull/2849#issuecomment-957693334)
  - Cons:
    - Need to handle [race conditions](https://github.com/kubernetes/enhancements/pull/2849#discussion_r682057570)
    - Need to consider [referenced Secrets](https://github.com/kubernetes/enhancements/pull/2849#discussion_r692459041) from snapshot after transferred

## Infrastructure Needed (Optional)

<!--
Use this section if you need things from the project/SIG. Examples include a
new subproject, repos requested, or GitHub details. Listing these here allows a
SIG to get the process for these resources started right away.
-->
