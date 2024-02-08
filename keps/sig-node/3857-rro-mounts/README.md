<!--
Last template update: 2023-07-26

https://github.com/kubernetes/enhancements/commit/8ef33fed0c79f80f0cb12df5aae6c5221f90f524
(See https://github.com/kubernetes/enhancements/commits/master/keps/NNNN-kep-template/README.md
to check if there are newer changes)
-->

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
# KEP-3857: Recursive read-only (RRO) mounts

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
  - [Notes/Constraints/Caveats (Optional)](#notesconstraintscaveats-optional)
  - [Risks and Mitigations](#risks-and-mitigations)
- [Design Details](#design-details)
  - [Core API](#core-api)
  - [CRI API](#cri-api)
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

Utilize runc's "rro" bind mount option (https://github.com/opencontainers/runc/pull/3272)
to make read-only bind mounts literally read-only.

The "rro" bind mount options is implemented by calling [`mount_setattr(2)`](https://man7.org/linux/man-pages/man2/mount_setattr.2.html)
with `MOUNT_ATTR_RDONLY` and `AT_RECURSIVE`.

Requires kernel >= 5.12, with one of the following OCI runtimes:
- runc >= 1.1
- crun >= 1.4

## Motivation

<!--
This section is for explicitly listing the motivation, goals, and non-goals of
this KEP.  Describe why the change is important and the benefits to users. The
motivation section can optionally provide links to [experience reports] to
demonstrate the interest in a KEP within the wider Kubernetes community.

[experience reports]: https://github.com/golang/go/wiki/ExperienceReports
-->

The current `readOnly` volumes are not recursively read-only, and may result in compromise of data;
e.g., even if `/mnt` is mounted as read-only, its submounts such as `/mnt/usbstorage` are not read-only.

### Goals

<!--
List the specific goals of the KEP. What is it trying to achieve? How will we
know that this has succeeded?
-->
Support recursive read-only mounts for kernel >= 5.12.

### Non-Goals

<!--
What is out of scope for this KEP? Listing non-goals helps to focus discussion
and make progress.
-->
Support recursive read-only mounts for old runc and old kernel releases.

## Proposal

<!--
This is where we get down to the specifics of what the proposal actually is.
This should have enough detail that reviewers can understand exactly what
you're proposing, but should not include things like API designs or
implementation. What is the desired outcome and how do we measure success?.
The "Design Details" section below is for the real
nitty-gritty.
-->

### User Stories (Optional)

<!--
Detail the things that people will be able to do if this KEP is implemented.
Include as much detail as possible so that people can understand the "how" of
the system. The goal here is to make this feel real for users without getting
bogged down.
-->

#### Story 1

A user wants to mount `/mnt`, includings its submounts such as `/mnt/usbstorage`, as read-only.

### Notes/Constraints/Caveats (Optional)

<!--
What are the caveats to the proposal?
What are some important details that didn't come across above?
Go in to as much detail as necessary here.
This might be a good place to talk about core concepts and how they relate.
-->
Constraints: needs runc >= 1.1 && kernel >= 5.12.

### Risks and Mitigations

<!--
What are the risks of this proposal, and how do we mitigate? Think broadly.
For example, consider both security and how this will impact the larger
Kubernetes ecosystem.

How will security be reviewed, and by whom?

How will UX be reviewed, and by whom?

Consider including folks who also work outside the SIG or subproject.
-->

- Increased API surface but still not secure-by-default, for sake of compatibility.
  - Mitigation: None

- False sense of security when not implemented
  - Mitigation: `VolumeMountStatus` indicating actual RRO setting

## Design Details

<!--
This section should contain enough information that the specifics of your
change are understandable. This may include API specs (though not always
required) or even code snippets. If there's any ambiguity about HOW your
proposal will be implemented, this is the place to discuss them.
-->


### Core API
Add `RecursiveReadOnly: (Disabled|IfPossible|Enabled)` to the [`VolumeMount`](https://github.com/kubernetes/kubernetes/blob/v1.26.1/pkg/apis/core/types.go#L1854-L1880) struct.

A pod manifest will look like this:
```yaml
spec:
  volumes:
    - name: foo
      hostPath:
        path: /mnt
        type: Directory
  containers:
  - volumeMounts:
    - mountPath: /mnt
      name: foo
      mountPropagation: None
      readOnly: true
      # NEW
      recursiveReadOnly: IfPossible
```

See the comment lines in the diff below for the constraints of the `VolumeMount` options:
```diff
diff --git a/pkg/apis/core/types.go b/pkg/apis/core/types.go
index e40b8bfa104..09c88222c2d 100644
--- a/pkg/apis/core/types.go
+++ b/pkg/apis/core/types.go
@@ -1914,6 +1914,31 @@ type VolumeMount struct {
 	// Optional: Defaults to false (read-write).
 	// +optional
 	ReadOnly bool
+	// RecursiveReadOnly specifies recursive-readonly mode.
+	//
+	// 1. If ReadOnly is false, RecursiveReadOnly must be unspecified.
+	// 2. If ReadOnly is true:
+	//   2.1. If RecursiveReadOnly is unspecified:
+	//        2.1.1. if it belongs to a Pod being created, it is initialized to Disabled.
+	//        2.1.2  if it belongs to a PodSpec under Deployment, Job, etc., it remains unspecified
+	//               (and will be set to Disabled eventually, when the Pod is created).
+	//   2.2. If RecursiveReadOnly is set to Disabled, the mount is not made recursively read-only.
+	//   2.3. If RecursiveReadOnly is set to IfPossible, the mount is made recursively read-only,
+	//        if it is supported by the runtime.
+	//        If it is not supported by the runtime, the mount is not made recursively read-only.
+	//        MountPropagation must be None or unspecified (which defaults to None).
+	//   2.4. If RecursiveReadOnly is set to Enabled, the mount is made recursively read-only.
+	//        If it is not supported by the runtime, the Pod will be terminated by kubelet,
+	//        and an error will be generated to indicate the reason.
+	//        MountPropagation must be None or unspecified (which defaults to None).
+	//   2.5. If RecursiveReadOnly is set to unknown value, it will result in an error.
+	//
+	// When this property is recognized by kubelet and kube-apiserver,
+	// VolumeMountStatus.RecursiveReadOnly will be set to either Disabled or Enabled.
+	//
+	// +featureGate=RecursiveReadOnlyMounts
+	// +optional
+	RecursiveReadOnly *RecursiveReadOnlyMode
 	// Required. If the path is not an absolute path (e.g. some/path) it
 	// will be prepended with the appropriate root prefix for the operating
 	// system.  On Linux this is '/', on Windows this is 'C:\'.
@@ -1926,6 +1951,8 @@ type VolumeMount struct {
 	// to container and the other way around.
 	// When not set, MountPropagationNone is used.
 	// This field is beta in 1.10.
+	// When RecursiveReadOnly is set to IfPossible or to Enabled, MountPropagation must be None or unspecified
+	// (which defaults to None).
 	// +optional
 	MountPropagation *MountPropagationMode
 	// Expanded path within the volume from which the container's volume should be mounted.
@@ -1961,6 +1988,18 @@ const (
 	MountPropagationBidirectional MountPropagationMode = "Bidirectional"
 )
 
+// RecursiveReadOnlyMode describes recursive-readonly mode.
+type RecursiveReadOnlyMode string
+
+const (
+	// RecursiveReadOnlyDisabled disables recursive-readonly mode.
+	RecursiveReadOnlyDisabled RecursiveReadOnlyMode = "Disabled"
+	// RecursiveReadOnlyIfPossible enables recursive-readonly mode if possible.
+	RecursiveReadOnlyIfPossible RecursiveReadOnlyMode = "IfPossible"
+	// RecursiveReadOnlyEnabled enables recursive-readonly mode, or raise an error.
+	RecursiveReadOnlyEnabled RecursiveReadOnlyMode = "Enabled"
+)
+
 // VolumeDevice describes a mapping of a raw block device within a container.
 type VolumeDevice struct {
 	// name must match the name of a persistentVolumeClaim in the pod
@@ -2591,6 +2630,10 @@ type ContainerStatus struct {
 	// +featureGate=InPlacePodVerticalScaling
 	// +optional
 	Resources *ResourceRequirements
+	// Status of volume mounts.
+	// +listType=atomic
+	// +optional
+	VolumeMounts []VolumeMountStatus
 }
 
 // PodPhase is a label for the condition of a pod at the current time.
@@ -2664,6 +2707,21 @@ const (
 	PodResizeStatusInfeasible PodResizeStatus = "Infeasible"
 )
 
+// VolumeMountStatus shows status of volume mounts.
+type VolumeMountStatus struct {
+	// Name corresponds to the name of the original VolumeMount.
+	Name string
+	// ReadOnly corresponds to the original VolumeMount.
+	// +optional
+	ReadOnly bool
+	// RecursiveReadOnly must be set to Disabled, Enabled, or unspecified (for non-readonly mounts).
+	// An IfPossible value in the original VolumeMount must be translated to Disabled or Enabled,
+	// depending on the mount result.
+	// +featureGate=RecursiveReadOnlyMounts
+	// +optional
+	RecursiveReadOnly *RecursiveReadOnlyMode
+}
+
 // RestartPolicy describes how the container should be restarted.
 // Only one of the following restart policies may be specified.
 // If none of the following policies is specified, the default one
@@ -4591,6 +4649,24 @@ type NodeDaemonEndpoints struct {
 	KubeletEndpoint DaemonEndpoint
 }
 
+// RuntimeClassFeatures is a set of runtime features.
+type RuntimeClassFeatures struct {
+	// RecursiveReadOnlyMounts is set to true if the runtime class supports RecursiveReadOnlyMounts.
+	// +optional
+	RecursiveReadOnlyMounts *bool
+}
+
+// RuntimeClass is a set of runtime class information.
+type RuntimeClass struct {
+	// Runtime class name.
+	// Empty for the default runtime class.
+	// +optional
+	Name string
+	// Supported features.
+	// +optional
+	Features *RuntimeClassFeatures
+}
+
 // NodeSystemInfo is a set of ids/uuids to uniquely identify the node.
 type NodeSystemInfo struct {
 	// MachineID reported by the node. For unique machine identification
@@ -4701,6 +4777,9 @@ type NodeStatus struct {
 	// Status of the config assigned to the node via the dynamic Kubelet config feature.
 	// +optional
 	Config *NodeConfigStatus
+	// The available runtime classes.
+	// +optional
+	RuntimeClasses []RuntimeClass
 }
 
 // UniqueVolumeName defines the name of attached volume
```

### CRI API

Add `bool recursive_read_only` to the [`Mount`](https://github.com/kubernetes/cri-api/blob/v0.26.1/pkg/apis/runtime/v1/api.proto#L212-L224) message.
CRI implementations will also expose the availability of the feature via the `RuntimeHandlerFeatures` message.

As kubelet can inspect the availability of the feature via the `RuntimeHandlerFeatures` message,
there is no concept of "IfPossible" in the CRI API;
kubelet translates an "IfPossible" value in the Core API into true or false in the CRI API

The `RuntimeHandlerFeatures` message is also propagated to the `NodeSystemInfo` struct of the Core API.

Diff:
```diff
diff --git a/staging/src/k8s.io/cri-api/pkg/apis/runtime/v1/api.proto b/staging/src/k8s.io/cri-api/pkg/apis/runtime/v1/api.proto
index e16688d8386..194d591c27f 100644
--- a/staging/src/k8s.io/cri-api/pkg/apis/runtime/v1/api.proto
+++ b/staging/src/k8s.io/cri-api/pkg/apis/runtime/v1/api.proto
@@ -235,6 +235,15 @@ message Mount {
     repeated IDMapping uidMappings = 6;
     // GidMappings specifies the runtime GID mappings for the mount.
     repeated IDMapping gidMappings = 7;
+    // If set to true, the mount is made recursive read-only.
+    // In this CRI API, recursive_read_only is a plain true/false boolean, although its equivalent
+    // in the Kubernetes core API is a quaternary that can be nil, "Enabled", "IfPossible", or "Disabled".
+    // kubelet translates that quaternary value in the core API into a boolean in this CRI API.
+    // Remarks:
+    // - nil is just treated as false
+    // - when set to true, readonly must be explicitly set to true, and propagation must be PRIVATE (0).
+    // - (readonly == false && recursive_read_only == false) does not make the mount read-only.
+    bool recursive_read_only = 8;
 }
 
 // IDMapping describes host to container ID mappings for a pod sandbox.
@@ -1524,6 +1533,22 @@ message StatusRequest {
     bool verbose = 1;
 }
 
+message RuntimeHandlerFeatures {
+    // recursive_read_only_mounts is set to true if the runtime handler supports
+    // recursive read-only mounts.
+    // For runc-compatible runtimes, availability of this feature can be detected by checking whether
+    // the Linux kernel version is >= 5.12, and,  `runc features | jq .mountOptions` contains "rro".
+    bool recursive_read_only_mounts = 1;
+}
+
+message RuntimeHandler {
+    // Name must be unique in StatusResponse.
+    // An empty string denotes the default handler.
+    string name = 1;
+    // Supported features.
+    RuntimeHandlerFeatures features = 2;
+}
+
 message StatusResponse {
     // Status of the Runtime.
     RuntimeStatus status = 1;
@@ -1532,6 +1557,8 @@ message StatusResponse {
     // debug, e.g. plugins used by the container runtime.
     // It should only be returned non-empty when Verbose is true.
     map<string, string> info = 2;
+    // Runtime handlers.
+    repeated RuntimeHandler runtime_handlers = 3;
 }
 
 message ImageFsInfoRequest {}
diff --git a/staging/src/k8s.io/cri-api/pkg/errors/errors.go b/staging/src/k8s.io/cri-api/pkg/errors/errors.go
index a4538669122..c8e4a18dec5 100644
--- a/staging/src/k8s.io/cri-api/pkg/errors/errors.go
+++ b/staging/src/k8s.io/cri-api/pkg/errors/errors.go
@@ -29,6 +29,9 @@ var (
 
        // ErrSignatureValidationFailed - Unable to validate the image signature on the PullImage RPC call.
        ErrSignatureValidationFailed = errors.New("SignatureValidationFailed")
+
+       // ErrRROUnsupported - Unable to enforce recursive readonly mounts
+       ErrRROUnsupported = errors.New("RROUnsupported")
 )
 
 // IsNotFound returns a boolean indicating whether the error
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

[X] I/we understand the owners of the involved components may require updates to
existing tests to make this code solid enough prior to committing the changes necessary
to implement this enhancement.

##### Prerequisite testing updates

<!--
Based on reviewers feedback describe what additional tests need to be added prior
implementing this enhancement to ensure the enhancements have also solid foundations.
-->

The existing tests will continue to pass.
New tests have to be added to cover the proposed feature.

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

- kubelet unit tests: will take a CRI status and populate the `VolumeMountStatus`.
- [CRI test](https://github.com/kubernetes-sigs/cri-tools):
  will be similar to [e2e tests](#e2e-tests) below but without using Kubernetes Core API.

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

<!--
- <test>: <link to test coverage>
-->

See [e2e tests](#e2e-tests) below.

##### e2e tests

<!--
This question should be filled when targeting a release.
For Alpha, describe what tests will be added to ensure proper quality of the enhancement.

For Beta and GA, add links to added tests together with links to k8s-triage for those tests:
https://storage.googleapis.com/k8s-triage/index.html

We expect no non-infra related flakes in the last month as a GA graduation criteria.
-->

<!--
- <test>: <link to test coverage>
-->

- run a pod in each RecursiveReadOnly mode and verify that the status comes back correctly
- run RecursiveReadOnly="Enabled" on a runtime that does not support it and ensure the error
- run RecursiveReadOnly="Enabled", and verify that the mount is actually recursively read-only
- run RecursiveReadOnly="Disabled", and verify that the mount is actually not recursively read-only

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

#### Alpha
- Feature implemented behind a feature flag
- Unit tests and CRI tests will pass

#### Beta
- e2e tests pass with containerd, CRI-O, and cri-dockerd

#### GA
- (Will be revisited during beta)

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

Upgrade: No action is needed. Existing readonly mounts will remain non-recursively readonly.

Downgrade:
- On downgrading kube-apiserver, the `[]volumeMounts.recursiveReadOnly` property will be lost
  and will not be propagated to kubelet.
  If the mode was set to non-`Disabled`, this will result in producing writable mounts.
  It is the user's responsibility to use the correct version of kube-apiserver
  when they need non-`Disabled` mode.

- On downgrading kubelet, the `[]volumeMounts.recursiveReadOnly` properties will be lost,
  and the `[]containerStatuses.[]volumeMount.recursiveReadOnly` status will not be updated.
  It is the user's responsibility to use the correct version of kubelet when they need to check
  `[]containerStatuses.[]volumeMount.recursiveReadOnly`.

- On downgrading the CRI or OCI runtime, if the `RecursiveReadOnly` mode is set to `Enabled`,
  kubelet will raise an error.
  `IfPossible` will be just treated as `Disabled`.

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

- It is the user's responsibility to use the correct version of kube-apiserver
  when they need non-`Disabled` mode. Otherwise the mode will not be propagated to kubelet.

- It is the user's responsibility to use the correct version of kube-apiserver and kubelet when they need to check
  `[]containerStatuses.[]volumeMount.recursiveReadOnly`.
  Otherwise the property may have an inconsistent value.

- CRI and OCI runtimes have to be updated before kubelet, otherwise kubelet will not be aware whether they
  supports the feature or not, and it will assume that they do not support the feature.

- If only partial nodes supports the feature, `Disabled` and `IfPossible` will continue to work on all the nodes,
  but `Enabled` will fail on a node that does not support the feature.
  kube-scheduler does not care about this, and, it is the user's responsibility to set `nodeSelector`, `nodeAffinity`,
  etc. to avoid scheduling a pod with `Enabled` to a node that does not support the feature.

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

- [X] Feature gate (also fill in values in `kep.yaml`)
  - Feature gate name: `RecursiveReadOnlyMounts`
  - Components depending on the feature gate: kube-apiserver,kubelet
<!--
- [ ] Other
  - Describe the mechanism:
  - Will enabling / disabling the feature require downtime of the control
    plane?
  - Will enabling / disabling the feature require downtime or reprovisioning
    of a node?
-->

###### Does enabling the feature change any default behavior?

<!--
Any change of default behavior may be surprising to users or break existing
automations, so be extremely careful here.
-->

No

###### Can the feature be disabled once it has been enabled (i.e. can we roll back the enablement)?

<!--
Describe the consequences on existing workloads (e.g., if this is a runtime
feature, can it break the existing applications?).

Feature gates are typically disabled by setting the flag to `false` and
restarting the component. No other changes should be necessary to disable the
feature.

NOTE: Also set `disable-supported` to `true` or `false` in `kep.yaml`.
-->

Yes, by unsetting `RecursiveReadOnly=Enabled`.

Components can be downgraded too, but it should be noted that `VolumeMountStatus`
may still see an inconsistent state when kubelet was downgraded.
The pod manifest has to be recreated to get a consistent state in this case.

###### What happens if we reenable the feature if it was previously rolled back?

Works.
Just same as a fresh roll-out, as long as the user has recreated the pod manifests.
(See "Can the feature be disabled once ..." section above)

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

Unit tests will run with and without the feature gate.

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

A rollout may fail when at least one of the following components are too old:

| Component      | `readOnlyRecursive` value that will cause an error |
|----------------|----------------------------------------------------|
| kube-apiserver | any value                                          |
| kubelet        | any value                                          |
| CRI runtime    | `Enabled`                                          |
| OCI runtime    | `Enabled`                                          |
| kernel         | `Enabled`                                          |

For example, an error will be returned like this if kube-apiserver is too old:
```console
$ kubectl apply -f rro.yaml
Error from server (BadRequest): error when creating "rro.yaml": Pod in version "v1" cannot be handled as a Pod:
strict decoding error: unknown field "spec.containers[0].volumeMounts[0].recursiveReadOnly"
```

No impact on already running workloads.

###### What specific metrics should inform a rollback?

<!--
What signals should users be paying attention to when the feature is young
that might indicate a serious problem?
-->

Look for an event saying indicating RRO is not supported by the runtime.
```console
$ kubectl get events -o json -w
...
{
    ...
    "kind": "Event",
    "message": "Error: RRONotSupported",
    ...
}
...
```

###### Were upgrade and rollback tested? Was the upgrade->downgrade->upgrade path tested?

<!--
Describe manual testing that was done and the outcomes.
Longer term, we may want to require automated upgrade/rollback tests, but we
are missing a bunch of machinery and tooling and can't do that now.
-->
(Will be revisited during beta)

###### Is the rollout accompanied by any deprecations and/or removals of features, APIs, fields of API types, flags, etc.?

<!--
Even if applying deprecation policies, they may still surprise some users.
-->
No

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

Yes, the feature is used if the following `jq` command prints non-zero number:

```bash
kubectl get pods -A -o json | jq '[.items[].spec.containers[].volumeMounts[]? | select(.recursiveReadOnly)] | length'
```

###### How can someone using this feature know that it is working for their instance?

<!--
For instance, if this is a pod-related feature, it should be possible to determine if the feature is functioning properly
for each individual pod.
Pick one more of these and delete the rest.
Please describe all items visible to end users below with sufficient detail so that they can verify correct enablement
and operation of this feature.
Recall that end users cannot usually observe component logs or access metrics.
-->

- [X] API .status
  - Condition name:  `volumeMountStatus.recursiveReadOnly`
<!--
- [ ] Other (treat as last resort)
  - Details:
-->

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

- `recursiveReadOnly=Enabled`:
  100% of pods that were scheduled into a node must run with recursive read-only mounts,
  or, 100% of them must fail to run.

- `recursiveReadOnly=IfPossible`:
  100% of pods that were scheduled into a node must run with or without recursive read-only mounts

- `recursiveReadOnly=Disabled`, or unset:
  100% of pods that were scheduled into a node must run without recursive read-only mounts

###### What are the SLIs (Service Level Indicators) an operator can use to determine the health of the service?

<!--
Pick one more of these and delete the rest.
-->

- [X] Metrics
  - Metric name: Event
  - [Optional] Aggregation method: `kubectl get events -o json -w`
  - Components exposing the metric: kubelet -> kube-apiserver

If `recursiveReadOnly` is set to `Enabled` but it is not supported, kubelet will raise an event like this:

```console
$ kubectl get events -o json -w
...
{
    ...
    "kind": "Event",
    "message": "Error: RRONotSupported",
    ...
}
...
```

If the OCI runtime claims that it supports recursive read only mounts but it actually fails to mount them,
the pod will enter CrashLoopBackoff.
The error from the OCI runtime can be inspected by running:
```
kubectl get pod -o json foo | jq .status.containerStatuses[0].lastState.terminated.message
```

###### Are there any missing metrics that would be useful to have to improve observability of this feature?

<!--
Describe the metrics themselves and the reasons why they weren't added (e.g., cost,
implementation difficulties, etc.).
-->
Potentially, kube-scheduler could be implemented to avoid scheduling a pod with `recursiveReadOnly: Enabled`
to a pod running an old kernel.

In this way, the Event metric described above would not happen, and users would instead see `Pending` pods
as an error metric.

However, this is not planned to be implemented in kube-scheduler, as it seems overengineering.
Users may use `nodeSelector`, `nodeAffinity`, etc. to workaround this.

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

Specific version of CRI, OCI, and Linux kernel

### Scalability

<!--
For alpha, this section is encouraged: reviewers should consider these questions
and attempt to answer them.

For beta, this section is required: reviewers must answer these questions.

For GA, this section is required: approvers should be able to confirm the
previous answers based on experience in the field.
-->

A pod with `recursiveReadOnly: Enabled` may be rejected by kubelet with the probablility of $$B/A$$,
where $$A$$ is the number of all the nodes that may potentially accept the pod,
and $$B$$ is the number of the nodes that may potentially accept the pod but does not support RRO.
This may affect scalability.

To evaluate this risk, users may run
`kubectl get nodes -o json | jq '[.items[].status.runtimeClasses[].Features]'`
to see how many nodes support `RecursiveReadOnlyMounts: true`.

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
No

###### Will enabling / using this feature result in introducing new API types?

<!--
Describe them, providing:
  - API type
  - Supported number of objects per cluster
  - Supported number of objects per namespace (for namespace-scoped objects)
-->
No

###### Will enabling / using this feature result in any new calls to the cloud provider?

<!--
Describe them, providing:
  - Which API(s):
  - Estimated increase:
-->
No

###### Will enabling / using this feature result in increasing size or count of the existing API objects?

<!--
Describe them, providing:
  - API type(s):
  - Estimated increase in size: (e.g., new annotation of size 32B)
  - Estimated amount of new objects: (e.g., new Object X for every existing Pod)
-->
A dozen of bytes

###### Will enabling / using this feature result in increasing time taken by any operations covered by existing SLIs/SLOs?

<!--
Look at the [existing SLIs/SLOs].

Think about adding additional work or introducing new steps in between
(e.g. need to do X to start a container), etc. Please describe the details.

[existing SLIs/SLOs]: https://git.k8s.io/community/sig-scalability/slos/slos.md#kubernetes-slisslos
-->
No

###### Will enabling / using this feature result in non-negligible increase of resource usage (CPU, RAM, disk, IO, ...) in any components?

<!--
Things to keep in mind include: additional in-memory state, additional
non-trivial computations, excessive access to disks (including increased log
volume), significant amount of data sent and/or received over network, etc.
This through this both in small and large cases, again with respect to the
[supported limits].

[supported limits]: https://git.k8s.io/community//sig-scalability/configs-and-limits/thresholds.md
-->
No

###### Can enabling / using this feature result in resource exhaustion of some node resources (PIDs, sockets, inodes, etc.)?

<!--
Focus not just on happy cases, but primarily on more pathological cases
(e.g. probes taking a minute instead of milliseconds, failed pods consuming resources, etc.).
If any of the resources can be exhausted, how this is mitigated with the existing limits
(e.g. pods per node) or new limits added by this KEP?

Are there any tests that were run/should be run to understand performance characteristics better
and validate the declared limits?
-->
No

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

A pod cannot be created, just as in other pods.

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
None

###### What steps should be taken if SLOs are not being met to determine the problem?

- Make sure that the node is running Linux kernel v5.12 or later.
- Make sure that `runc features | jq .mountOptions` contains "rro". Otherwise update runc.
- Make sure that `crictl info` (with the latest crictl)
  reports that `RecursiveReadOnlyMounts` is supported.
  Otherwise update the CRI runtime, and make sure that no relevant error is printed in
  the CRI runtime's log.
- Make sure that `kubectl get nodes -o json | jq '[.items[].status.runtimeClasses[].Features]'`
  (with the latest kubectl and control planes)
  reports that `RecursiveReadOnlyMounts` is supported.
  Otherwise update the CRI runtime, and make sure that no relevant error is printed in
  kubelet's log.

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
See "Alternatives" below.

## Alternatives

<!--
What other approaches did you consider, and why did you rule them out? These do
not need to be as detailed as the proposal, but should include enough
information to express the idea and why it was not acceptable.
-->

Plan B is to keep the Kubernetes Core API and the CRI API completely unmodified,
and just let the CRI runtime treat "readonly" as "recursive readonly".

This would be much easier to implement and adopt, however, small portion of users may find this to be a breaking change.

Actually, containerd has once adopted the Plan B (https://github.com/containerd/containerd/pull/9713) in its main branch
(not in any GA release), but it is being reverted in favor of this KEP now (https://github.com/containerd/containerd/pull/9747).

## Infrastructure Needed (Optional)

<!--
Use this section if you need things from the project/SIG. Examples include a
new subproject, repos requested, or GitHub details. Listing these here allows a
SIG to get the process for these resources started right away.
-->

runc >= 1.1 && kernel >= 5.12
