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
# KEP-4639: OCI VolumeSource

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
    - [Story 3](#story-3)
  - [Notes/Constraints/Caveats (Optional)](#notesconstraintscaveats-optional)
  - [Vocabulary: OCI Images, Artifacts, and Objects](#vocabulary-oci-images-artifacts-and-objects)
  - [Risks and Mitigations](#risks-and-mitigations)
- [Design Details](#design-details)
  - [Kubelet and Container Runtime Interface (CRI) support for OCI artifacts](#kubelet-and-container-runtime-interface-cri-support-for-oci-artifacts)
    - [kubelet](#kubelet)
    - [CRI](#cri)
  - [Test Plan](#test-plan)
      - [Prerequisite testing updates](#prerequisite-testing-updates)
      - [Unit tests](#unit-tests)
      - [Integration tests](#integration-tests)
      - [e2e tests](#e2e-tests)
  - [Graduation Criteria](#graduation-criteria)
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
  - [<a href="https://github.com/kubernetes/enhancements/tree/master/keps/sig-storage/1495-volume-populators">KEP 1495: Volume Populators</a>](#kep-1495-volume-populators)
  - [Custom CSI Plugin](#custom-csi-plugin)
  - [Advantages of In-Tree OCI VolumeSource](#advantages-of-in-tree-oci-volumesource)
  - [Conclusion](#conclusion)
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
  - [ ] (R) Ensure GA e2e tests meet requirements for [Conformance Tests](https://github.com/kubernetes/community/blob/master/contributors/devel/sig-architecture/conformance-tests.md) 
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

The proposed enhancement adds a new `VolumeSource` to Kubernetes that supports OCI images and/or OCI artifacts.
This allows users to package files and share them among containers in a pod without including them in the main image,
thereby reducing vulnerabilities and simplifying image creation.

While OCI images are well-supported by Kubernetes and CRI,
extending support to OCI artifacts involves recognizing additional media types, implementing custom lifecycle management,
and ensuring appropriate validation and security measures.

## Motivation

Supporting OCI images and artifacts directly as a `VolumeSource` allows
Kubernetes to focus on OCI standards as well as allows to store and distribute
any content using OCI registries. This allows the project to grow into use cases
which go beyond running particular images.

### Goals

- Introduce a new `VolumeSource` type that allows mounting OCI images and/or artifacts.
- Simplify the process of sharing files among containers in a pod.

### Non-Goals

- This proposal does not aim to replace existing `VolumeSource` types.
- This proposal does not address other use cases for OCI objects beyond file sharing among containers in a pod.

## Proposal

We propose to add a new `VolumeSource` that supports OCI images and/or artifacts. This `VolumeSource` will allow users to mount an OCI object
directly into a pod, making the files within the image accessible to the containers without the need to include them in the main image and to be able to host them in OCI compatible registries.

### User Stories (Optional)

#### Story 1

As a Kubernetes user, I want to share a configuration file among multiple containers in a pod without including the file in my main image, so that I can
minimize security risks and image size. 

Beside that, I want:
- to package this file in an OCI object to take advantage of OCI distribution.
- the image to be downloaded with the same credentials that kubelet using for other images.
- to be able to use image pull secrets when downloading the image if an image is from the registry that requires image pull secrets.

#### Story 2

As a DevOps engineer, I want to package and distribute binary artifacts using OCI images and mount them directly into my Kubernetes pods,
so that I can streamline my CI/CD pipeline. I want to package this file in an OCI object to take advantage of OCI distribution.

#### Story 3

As a data scientist, MLOps engineer, or AI developer, I want to mount large language models or machine learning models in a pod alongside a model-server, so that I can efficiently serve the models
without including them in the container image. I want to package these models in an OCI object to take advantage of OCI distribution and ensure
efficient model deployment. This allows to separate the data from the executables that process them.

### Notes/Constraints/Caveats (Optional)

- This enhancement assumes that the cluster has access to the OCI registry.
- The implementation must handle image pull secrets and other registry authentication mechanisms.
- Performance considerations must be taken into account, especially for large images or artifacts.

### Vocabulary: OCI Images, Artifacts, and Objects

**OCI Image ([spec](https://github.com/opencontainers/image-spec/blob/main/spec.md)):**
   - A container image that conforms to the Open Container Initiative (OCI) Image Specification.
     It includes a filesystem bundle and metadata required to run a container.
   - Consists of multiple layers (each layer being a tarball), a manifest (which lists the layers), and a config file
     (which provides configuration data such as environment variables, entry points, etc.).
   - **Use Case:** Used primarily for packaging and distributing containerized applications.

**OCI Artifact ([guidance](https://github.com/opencontainers/image-spec/blob/main/artifacts-guidance.md)):**
   - An artifact describes any content that is stored and distributed using the OCI image format.
     It includes not just container images but also other types of content like Helm charts, WASM modules, machine learning models, etc.
   - Artifacts use the same image manifest and layer structure but may contain different types of data
     within those layers. The artifact manifest can have media types that differ from those in standard container images.
   - **Use Case:** Allows the distribution of non-container content using the same infrastructure and tools developed for OCI images.

**OCI Object:**
   - Umbrella term encompassing both OCI images and OCI artifacts. It represents any object that conforms to the OCI specifications for storage and distribution.

### Risks and Mitigations

- **Security Risks:** Allowing direct mounting of OCI images introduces potential attack vectors. Mitigation includes thorough security reviews and
  limiting access to trusted registries. Limiting to OCI artifacts (non-runnable content) or read-only mode may lessen the security risk.
- **Compatibility Risks:** Existing webhooks watching for the images used by the pod using some policies will need to be updated to expect the image to be specified as a `VolumeSource`.
- **Performance Risks:** Large images or artifacts could impact performance. Mitigation includes optimizations in the implementation and providing
  guidance on best practices for users.

## Design Details

The new `VolumeSource` will be defined in the Kubernetes API, and the implementation will involve updating components (CRI, Kubelet)
to support this source type. Key design aspects include:

- API changes to introduce the new `VolumeSource` type.
- Modifications to the Kubelet to handle mounting OCI images and artifacts.
- Handling image pull secrets and registry authentication.
- Reuse existing logic from ConfigMaps for:
  - Determining the file location on the host filesystem.
  - Managing garbage collection of unused images and artifacts.
  - Handling updates to the OCI image or artifact, similar to how ConfigMaps update mounted files.

The following code snippet illustrates the proposed API change:

```yaml
apiVersion: v1
kind: Pod
metadata:
  name: example-pod
spec:
  volumes:
  - name: oci-volume
    oci:
      image: "example.com/my-image:latest"
  containers:
  - name: my-container
    image: busybox
    volumeMounts:
    - mountPath: /data
      name: oci-volume
```

### Kubelet and Container Runtime Interface (CRI) support for OCI artifacts

Kubelet and the Container Runtime Interface (CRI) currently handle OCI images. To support OCI artifacts,
potential enhancements may be required:

**Extended Media Type Handling:**
   - Update Kubelet and CRI to recognize and handle new media types associated with OCI artifacts.
   - Ensure that pulling and storing these artifacts is as efficient and secure as with OCI images.

**Lifecycling and Garbage Collection:**
   - Implement custom logic for managing the lifecycle of various OCI artifacts.

**Artifact-Specific Configuration:**
   - Introduce new configuration options or CRD to handle the unique requirements of different types of OCI artifacts.

**Validation:**
   - Extend validation and security checks to cover new artifact types.

**Storage Optimization:**
   - Develop optimized storage solutions tailored for different artifact types,
     potentially integrating with existing storage solutions or introducing new mechanisms.

#### kubelet

Some parts of the existing kubelet code can be reused, for example:

- The logic how to ensure that an image exists on the node:
  https://github.com/kubernetes/kubernetes/blob/39c6bc3/pkg/kubelet/images/image_manager.go#L102
- The retrieval of available secrets for a pod:
  https://github.com/kubernetes/kubernetes/blob/39c6bc3/pkg/kubelet/kubelet_pods.go#L988

We consider to refactor the logic into re-usable bits for both, the kubelet as
well as the volume plugin, because volume plugins run in a dedicated go routine
at the beginning of the kubelet initialization:

https://github.com/kubernetes/kubernetes/blob/39c6bc3/pkg/kubelet/kubelet.go#L1633-L1634

Specifying a pull policy will not be supported in the alpha implementation of the
feature and will align to `PullAlways`.

For registry authentication purposes the same credentials will be used as for
the running pod, if available.

#### CRI

The CRI is already capable of pulling images [via the `PullImage` RPC](https://github.com/kubernetes/cri-api/blob/3a66d9d/pkg/apis/runtime/v1/api.proto#L154).
This does not necessarily include OCI artifacts and is dependent on the used
container runtime (CRI-O, containerd). Beside that, volume plugins usually
require a mount from a local directory. This means we extend the CRI protocol by
adding a two new methods as well as their request and response types:

```protobuf
// ImageService defines the public APIs for managing images.
service ImageService {
    // MountOCIObject pulls and mounts any OCI image or artifact to a local
    // directory on disk
    rpc MountOCIObject(MountOCIObjectRequest) returns (MountOCIObjectResponse) {}

    // UnmountOCIObject unmounts and also removes any OCI image or artifact.
    rpc UnmountOCIObject(UnmountOCIObjectRequest) returns (UnmountOCIObjectResponse) {}

    /// …
}

message MountOCIObject {
    // Reference to be pulled.
    string object = 1;

    // Authentication configuration for pulling the object.
    AuthConfig auth = 2;

    // Local path to be mounted.
    string path = 3;
}


message MountOCIObjectResponse {
    // Reference to the object in use.
    string id = 1;
}

message UnmountOCIObjectRequest {
    // Identifier of the object to remove.
    string id = 1;
}

message UnmountOCIObjectResponse {}
```

### Test Plan

<!--
**Note:** *Not required until targeted at a release.*
The goal is to ensure that we don't accept enhancements with inadequate testing.

All code is expected to have adequate tests (eventually with coverage
expectations). Please adhere to the [Kubernetes testing guidelines][testing-guidelines]
when drafting this test plan.

[testing-guidelines]: https://git.k8s.io/community/contributors/devel/sig-testing/testing.md
-->

[ ] I/we understand the owners of the involved components may require updates to
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

- `<package>`: `<date>` - `<test coverage>`

##### Integration tests

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

- <test>: <link to test coverage>

##### e2e tests

<!--
This question should be filled when targeting a release.
For Alpha, describe what tests will be added to ensure proper quality of the enhancement.

For Beta and GA, add links to added tests together with links to k8s-triage for those tests:
https://storage.googleapis.com/k8s-triage/index.html

We expect no non-infra related flakes in the last month as a GA graduation criteria.
-->

- <test>: <link to test coverage>

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

#### Alpha

- Feature implemented behind a feature flag
- Initial e2e tests completed and enabled

#### Beta

- Gather feedback from developers and surveys
- Complete features A, B, C
- Additional tests are in Testgrid and linked in KEP

#### GA

- N examples of real-world usage
- N installs
- More rigorous forms of testing—e.g., downgrade tests and scalability tests
- Allowing time for feedback

**Note:** Generally we also wait at least two releases between beta and
GA/stable, because there's no opportunity for user feedback, or even bug reports,
in back-to-back releases.

**For non-optional features moving to GA, the graduation criteria must include
[conformance tests].**

[conformance tests]: https://git.k8s.io/community/contributors/devel/sig-architecture/conformance-tests.md

#### Deprecation

- Announce deprecation and support policy of the existing flag
- Two versions passed since introducing the functionality that deprecates the flag (to address version skew)
- Address feedback on usage/changed behavior, provided on GitHub issues
- Deprecate the flag
-->

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

### Version Skew Strategy

<!--
If applicable, how will the component handle version skew with other
components? What are the guarantees? Make sure this is in the test plan.

Consider the following in developing a version skew strategy for this
enhancement:
- Does this enhancement involve coordinating behavior in the control plane and nodes?
- How does an n-3 kubelet or kube-proxy without this feature available behave when this feature is used?
- How does an n-1 kube-controller-manager or kube-scheduler without this feature available behave when this feature is used?
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

- [ ] Feature gate (also fill in values in `kep.yaml`)
  - Feature gate name:
  - Components depending on the feature gate:
- [ ] Other
  - Describe the mechanism:
  - Will enabling / disabling the feature require downtime of the control
    plane?
  - Will enabling / disabling the feature require downtime or reprovisioning
    of a node?

###### Does enabling the feature change any default behavior?

<!--
Any change of default behavior may be surprising to users or break existing
automations, so be extremely careful here.
-->

###### Can the feature be disabled once it has been enabled (i.e. can we roll back the enablement)?

<!--
Describe the consequences on existing workloads (e.g., if this is a runtime
feature, can it break the existing applications?).

Feature gates are typically disabled by setting the flag to `false` and
restarting the component. No other changes should be necessary to disable the
feature.

NOTE: Also set `disable-supported` to `true` or `false` in `kep.yaml`.
-->

###### What happens if we reenable the feature if it was previously rolled back?

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
- [ ] API .status
  - Condition name: 
  - Other field: 
- [ ] Other (treat as last resort)
  - Details:

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

- [ ] Metrics
  - Metric name:
  - [Optional] Aggregation method:
  - Components exposing the metric:
- [ ] Other (treat as last resort)
  - Details:

###### Are there any missing metrics that would be useful to have to improve observability of this feature?

<!--
Describe the metrics themselves and the reasons why they weren't added (e.g., cost,
implementation difficulties, etc.).
-->

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

###### Will enabling / using this feature result in introducing new API types?

<!--
Describe them, providing:
  - API type
  - Supported number of objects per cluster
  - Supported number of objects per namespace (for namespace-scoped objects)
-->

###### Will enabling / using this feature result in any new calls to the cloud provider?

<!--
Describe them, providing:
  - Which API(s):
  - Estimated increase:
-->

###### Will enabling / using this feature result in increasing size or count of the existing API objects?

<!--
Describe them, providing:
  - API type(s):
  - Estimated increase in size: (e.g., new annotation of size 32B)
  - Estimated amount of new objects: (e.g., new Object X for every existing Pod)
-->

###### Will enabling / using this feature result in increasing time taken by any operations covered by existing SLIs/SLOs?

<!--
Look at the [existing SLIs/SLOs].

Think about adding additional work or introducing new steps in between
(e.g. need to do X to start a container), etc. Please describe the details.

[existing SLIs/SLOs]: https://git.k8s.io/community/sig-scalability/slos/slos.md#kubernetes-slisslos
-->

###### Will enabling / using this feature result in non-negligible increase of resource usage (CPU, RAM, disk, IO, ...) in any components?

<!--
Things to keep in mind include: additional in-memory state, additional
non-trivial computations, excessive access to disks (including increased log
volume), significant amount of data sent and/or received over network, etc.
This through this both in small and large cases, again with respect to the
[supported limits].

[supported limits]: https://git.k8s.io/community//sig-scalability/configs-and-limits/thresholds.md
-->

###### Can enabling / using this feature result in resource exhaustion of some node resources (PIDs, sockets, inodes, etc.)?

<!--
Focus not just on happy cases, but primarily on more pathological cases
(e.g. probes taking a minute instead of milliseconds, failed pods consuming resources, etc.).
If any of the resources can be exhausted, how this is mitigated with the existing limits
(e.g. pods per node) or new limits added by this KEP?

Are there any tests that were run/should be run to understand performance characteristics better
and validate the declared limits?
-->

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

### [KEP 1495: Volume Populators](https://github.com/kubernetes/enhancements/tree/master/keps/sig-storage/1495-volume-populators)

The volume-populators API extension allows you to populate a volume with data from an external data source when the volume is created.
This is a good solution for restoring a volume from a snapshot or initializing a volume with data from a database backup. However, it does not
address the desire to use OCI distribution, versioning, and signing for mounted data.

The proposed in-tree OCI VolumeSource provides a direct and integrated approach to mount OCI artifacts, leveraging the existing OCI
infrastructure for packaging, distribution, and security.

### Custom CSI Plugin

See [https://github.com/warm-metal/container-image-csi-driver](https://github.com/warm-metal/container-image-csi-driver)

An out-of-tree CSI plugin can provide flexibility and modularity, but there are trade-offs to consider:

 - Complexity of managing an external CSI plugin. This includes handling the installation, configuration, and updates of the CSI driver, which adds
   an additional operational burden. For a generic, vendor-agnostic, and widely-adopted solution this would not make sense.
 - Supporting the image pull secrets as well as credentials provider will be tricky and needs to be reimplemented with the separate API calls.
 - External CSI plugins implement their own lifecycle management and garbage collection mechanisms,
   yet these already exist in-tree for OCI images.
 - Performance: There is additional overhead with an out-of-tree CSI plugin, especially in scenarios requiring frequent image pulls
   or large volumes of data.

### Advantages of In-Tree OCI VolumeSource

1. **Leverage Existing Mechanisms:**
   - **No New Data Types or Objects:** OCI images are already a core part of the Kubernetes ecosystem. Extending support for OCI artifacts, many of
     the same mechanisms will be reused. This ensures consistency and reduces complexity, as both adhere to the same OCI image format.
   - **Existing Lifecycle Management and Garbage Collection:** Kubernetes has efficient lifecycle management and garbage collection mechanisms for
	 volumes and container images. The in-tree OCI VolumeSource will utilize these existing mechanisms.

2. **Integration with Kubernetes:**
   - **Optimal Performance:** Deep integration with the scheduler and kubelet ensures optimal performance and
     resource management. This integration allows the OCI VolumeSource to benefit from all existing optimizations and features.
   - **Unified Interface:** Users interact with a consistent and unified interface for managing volumes, reducing the learning curve and
     potential for configuration errors.

3. **Simplified Maintenance and Updates:**
   - **Core Project Maintenance:** In-tree features are maintained and updated as part of the core project. It makes sense
     for widely-used and vendor agnostic features to utilize the core testing infrastructure, release cycles, and security updates.

### Conclusion

The in-tree implementation of an OCI VolumeSource offers significant advantages by leveraging existing core mechanisms,
ensuring deep integration, and simplifying management. This approach avoids the complexity, duplication, and other potential inefficiencies
of out-of-tree CSI plugins, providing a more reliable solution for mounting OCI images and artifacts.
