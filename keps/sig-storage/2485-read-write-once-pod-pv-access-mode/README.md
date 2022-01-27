<!--
**Note:** When your KEP is complete, all of these comment blocks should be removed.

To get started with this template:

- [X] **Pick a hosting SIG.**
  Make sure that the problem space is something the SIG is interested in taking
  up. KEPs should not be checked in without a sponsoring SIG.
- [X] **Create an issue in kubernetes/enhancements**
  When filing an enhancement tracking issue, please make sure to complete all
  fields in that template. One of the fields asks for a link to the KEP. You
  can leave that blank until this KEP is filed, and then go back to the
  enhancement and add the link.
- [X] **Make a copy of this template directory.**
  Copy this template into the owning SIG's directory and name it
  `NNNN-short-descriptive-title`, where `NNNN` is the issue number (with no
  leading-zero padding) assigned to your enhancement above.
- [X] **Fill out as much of the kep.yaml file as you can.**
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
# KEP-2485: ReadWriteOncePod PersistentVolume AccessMode

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
- [Glossary](#glossary)
- [Motivation](#motivation)
  - [Kubernetes Changes](#kubernetes-changes)
  - [CSI Specification Changes](#csi-specification-changes)
  - [Goals](#goals)
  - [Non-Goals](#non-goals)
- [Proposal](#proposal)
  - [User Stories (Optional)](#user-stories-optional)
    - [ReadWriteOncePod PVC Used Twice Fails for Second Consumer](#readwriteoncepod-pvc-used-twice-fails-for-second-consumer)
    - [ReadWriteOnce PVC Continues to Succeed with New Kubernetes, Old CSI Driver](#readwriteonce-pvc-continues-to-succeed-with-new-kubernetes-old-csi-driver)
  - [Notes/Constraints/Caveats (Optional)](#notesconstraintscaveats-optional)
  - [Risks and Mitigations](#risks-and-mitigations)
- [Design Details](#design-details)
  - [Kubernetes Changes, Access Mode](#kubernetes-changes-access-mode)
    - [Scheduler Enforcement](#scheduler-enforcement)
    - [Mount Enforcement](#mount-enforcement)
  - [CSI Specification Changes, Volume Capabilities](#csi-specification-changes-volume-capabilities)
  - [Test Plan](#test-plan)
    - [Validation of PersistentVolumeSpec Object](#validation-of-persistentvolumespec-object)
    - [Mounting and Mapping with ReadWriteOncePod](#mounting-and-mapping-with-readwriteoncepod)
    - [Mounting and Mapping with ReadWriteOnce](#mounting-and-mapping-with-readwriteonce)
    - [Mapping Kubernetes Access Modes to CSI Volume Capability Access Modes](#mapping-kubernetes-access-modes-to-csi-volume-capability-access-modes)
    - [End to End Tests](#end-to-end-tests)
  - [Graduation Criteria](#graduation-criteria)
    - [Alpha](#alpha)
    - [Beta](#beta)
    - [GA](#ga)
  - [Upgrade / Downgrade Strategy](#upgrade--downgrade-strategy)
  - [Version Skew Strategy](#version-skew-strategy)
    - [API Server Version N / Scheduler Version N / Kubelet Version N-1 or N-2](#api-server-version-n--scheduler-version-n--kubelet-version-n-1-or-n-2)
    - [API Server Version N / Scheduler Version N-1 / Kubelet Version N-1 or N-2](#api-server-version-n--scheduler-version-n-1--kubelet-version-n-1-or-n-2)
    - [API Understands ReadWriteOncePod, CSI Sidecars Do Not](#api-understands-readwriteoncepod-csi-sidecars-do-not)
    - [CSI Controller Service Understands New CSI Access Modes, CSI Node Service Does Not](#csi-controller-service-understands-new-csi-access-modes-csi-node-service-does-not)
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

- [X] (R) Enhancement issue in release milestone, which links to KEP dir in [kubernetes/enhancements] (not the initial KEP PR)
- [X] (R) KEP approvers have approved the KEP status as `implementable`
- [X] (R) Design details are appropriately documented
- [X] (R) Test plan is in place, giving consideration to SIG Architecture and SIG Testing input
- [X] (R) Graduation criteria is in place
- [X] (R) Production readiness review completed
- [X] (R) Production readiness review approved
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

This KEP introduces a new ReadWriteOncePod access mode for PersistentVolumes
that restricts access to a single pod on a single node. This access mode
differs from the existing ReadWriteOnce (RWO) access mode, which restricts
access to a single node, but allows simultaneous access from many pods on that
node.

Additionally, this KEP outlines required changes to the CSI spec, drivers, and
sidecars in order to support this new access mode while maintaining backwards
compatibility.

## Glossary

- [Node]
  - A virtual or physical machine in a Kubernetes cluster that runs pods
- [PersistentVolume]
  - A piece of storage in the cluster that has been provisioned by an
    administrator or dynamically provisioned using `StorageClasses`
- Access mode
  - A description of how a PersistentVolume can be accessed
- ReadWriteOnce (RWO)
  - An access mode that restricts PersistentVolume access to a single node
- ReadWriteOncePod (RWOP)
  - A new access mode that restricts PersistentVolume access to a single pod on
    a single node
- [CSI]
  - The Container Storage Interface, a specification for storage provider
    plugins to integrate with cluster orchestrators (like Kubernetes)

[Node]: https://kubernetes.io/docs/concepts/architecture/nodes/
[PersistentVolume]: https://kubernetes.io/docs/concepts/storage/persistent-volumes/
[CSI]: https://github.com/container-storage-interface/spec

## Motivation

<!--
This section is for explicitly listing the motivation, goals, and non-goals of
this KEP.  Describe why the change is important and the benefits to users. The
motivation section can optionally provide links to [experience reports] to
demonstrate the interest in a KEP within the wider Kubernetes community.

[experience reports]: https://github.com/golang/go/wiki/ExperienceReports
-->

### Kubernetes Changes

Kubernetes does not have an access mode for PersistentVolumes that allows users
to restrict access to a single pod on a single node. This can cause problems for
certain workloads. For example, if you had a workload (using ReadWriteOnce)
performing an update of a storage device and the workload scaled to more than
one Pod, you could encounter issues if the second pod landed on the same node
and started simultaneously modifying the device.

For sensitive workloads, users have to work around the lack of a single-workload
access mode in other ways (for example, scheduling only a single pod on a node
and using ReadWriteOnce), which can lead to inefficient use of resources in
their cluster.

See [#30085] and [#26567] for issues related to this.

[#30085]: https://github.com/kubernetes/kubernetes/issues/30085
[#26567]: https://github.com/kubernetes/kubernetes/issues/26567

### CSI Specification Changes

In the CSI spec there are conflicting definitions of the [`SINGLE_NODE_WRITER`]
access mode. By definition, `SINGLE_NODE_WRITER` means "Can only be published
once as read/write on a single node, at any given time." The problem is how this
access mode is used during `NodePublishVolume`, which is typically where volume
mounting is performed.

The CSI spec defines that when [`NodePublishVolume`] is called a second time for
a volume with a non-`MULTI_NODE` access mode and with a different target path,
the plugin should return `FAILED_PRECONDITION`. For CSI plugins that strictly
adhere to the spec, this guarantees that a volume can only be mounted to a
single target path, which means `SINGLE_NODE_WRITER` restricts access to a
single pod on a single node. This behavior conflicts with the original
definition. Due to this conflict, we do not have an access mode that represents
multiple writers on the same node.

[`SINGLE_NODE_WRITER`]: https://github.com/container-storage-interface/spec/blob/v1.4.0/csi.proto#L405-L407
[`NodePublishVolume`]: https://github.com/container-storage-interface/spec/blob/v1.4.0/spec.md#nodepublishvolume

### Goals

<!--
List the specific goals of the KEP. What is it trying to achieve? How will we
know that this has succeeded?
-->

- Outline expected behavior of the ReadWriteOncePod access mode
- Provide a high level design for ReadWriteOncePod access mode support
- Define API changes needed to support this access mode
- Outline changes needed in CSI spec and sidecars to support the
  ReadWriteOncePod access mode
- Outline changes needed in CSI spec and sidecars to continue supporting the
  ReadWriteOnce access mode

### Non-Goals

<!--
What is out of scope for this KEP? Listing non-goals helps to focus discussion
and make progress.
-->

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

See the [version skew strategy] section below for additional scenarios.

[version skew strategy]: #version-skew-strategy

#### ReadWriteOncePod PVC Used Twice Fails for Second Consumer

This scenario asserts a ReadWriteOncePod can only be bind mounted into a single
pod on a single node.

- User creates a PVC with ReadWriteOncePod access mode
- User creates pod 1 using this PVC, scheduled on node 1
- User creates pod 2 using this PVC, also scheduled on node 1
- User observes pod 2 fails to start because the referenced PVC is in-use by
  another pod on the same node

Additionally, for attachment:

- User creates pod 3 using this PVC, scheduled on node 2
- User observes pod 3 fails to start because the referenced PVC is attached to
  another node

#### ReadWriteOnce PVC Continues to Succeed with New Kubernetes, Old CSI Driver

This scenario asserts the existing ReadWriteOnce behavior is preserved for old
CSI drivers. The exact behavior may differ across CSI drivers since not all
drivers conform to the CSI spec, but it should be consistent with how it behaved
before.

- User creates a PVC with ReadWriteOnce access mode
- User creates pod 1 using this PVC, scheduled on node 1
- User observes pod running

### Notes/Constraints/Caveats (Optional)

<!--
What are the caveats to the proposal?
What are some important details that didn't come across above?
Go in to as much detail as necessary here.
This might be a good place to talk about core concepts and how they relate.
-->

### Risks and Mitigations

<!--
What are the risks of this proposal, and how do we mitigate? Think broadly.
For example, consider both security and how this will impact the larger
Kubernetes ecosystem.

How will security be reviewed, and by whom?

How will UX be reviewed, and by whom?

Consider including folks who also work outside the SIG or subproject.
-->

## Design Details

<!--
This section should contain enough information that the specifics of your
change are understandable. This may include API specs (though not always
required) or even code snippets. If there's any ambiguity about HOW your
proposal will be implemented, this is the place to discuss them.
-->

### Kubernetes Changes, Access Mode

In Kubernetes, we should add a new ReadWriteOncePod persistent volume access
mode to PersistentVolumes and PersistentVolumeClaims. This change will require
adding a feature gate to the kube-apiserver, kube-scheduler, and kubelet.
Validation logic will need updating to accept this access mode type if the
feature gate is enabled.

```golang
       // can be mounted read/write mode to exactly 1 pod
       ReadWriteOncePod PersistentVolumeAccessMode = "ReadWriteOncePod"
```

This access mode will be enforced in two places:

#### Scheduler Enforcement

First is at the time a pod is scheduled. When scheduling a pod, if another pod
is found using the same PVC and the PVC uses ReadWriteOncePod, then scheduling
will fail and the pod will be considered UnschedulableAndUnresolvable.

In order to determine if a pod using a ReadWriteOncePod PVC can be scheduled, we
need to enumerate all pods and check if any are already consuming this PVC. This
logic will take place as part of the PreFilter extension point in the [volume
restrictions plugin].

The [node info cache] will be extended to map the PVC name to a reference count
for the PVC. In the PreFilter extension point, if the pod's PVC is using
ReadWriteOncePod, we will query this map for each node checking for references
to the scheduled pod's PVC. If one is found the pod will fail scheduling and be
marked UnschedulableAndUnresolvable.

[volume restrictions plugin]: https://github.com/kubernetes/kubernetes/blob/v1.21.0/pkg/scheduler/framework/plugins/volumerestrictions/volume_restrictions.go#L29
[node info cache]: https://github.com/kubernetes/kubernetes/blob/v1.21.0/pkg/scheduler/framework/types.go#L357

#### Mount Enforcement

As an additional precaution this will also be enforced at the time a volume is
mounted for filesystem devices, and at the time a volume is mapped for block
devices. During the mount operation, kubelet will check the [actual state of the
world cache] to determine if the volume is already in-use by another pod. If it
is, kubelet will fail mounting with an appropriate error message.

[actual state of the world cache]: https://github.com/kubernetes/kubernetes/blob/v1.21.0/pkg/kubelet/volumemanager/cache/actual_state_of_world.go#L46

### CSI Specification Changes, Volume Capabilities

In the CSI spec we should add two new access modes that explicitly state the
number of writers on a single node.

```protobuf
      // Can only be published once as read/write at a single worklad on
      // a single node, at any given time.
      SINGLE_NODE_SINGLE_WRITER = 6;

      // Can be published as read/write at multiple workloads on a
      // single node simultaneously.
      SINGLE_NODE_MULTI_WRITER = 7;
```

These access modes are modeled after the existing `MULTI_NODE_SINGLE_WRITER` and
`MULTI_NODE_MULTI_WRITER` access modes. The reason for making this distinction
is because the `SINGLE_NODE_WRITER` volume capability has conflicting
definitions (see the [motivation](#motivation) section for context).

In order to preserve backwards compatibility, we must be careful about how to
map between Kubernetes access modes and the new CSI access modes. The way we
control this is by maintaining different mappings based on the CSI driver's
capabilities.

Both the controller and node services should have capability bits that
represent that they support the new access modes:

```protobuf
      // Indicates the SP supports the SINGLE_NODE_SINGLE_WRITER and/or
      // SINGLE_NODE_MULTI_WRITER access modes.
      // These access modes are intended to replace the
      // SINGLE_NODE_WRITER access mode to clarify the number of writers
      // for a volume on a single node. Plugins MUST accept and allow
      // use of the SINGLE_NODE_WRITER access mode when either
      // SINGLE_NODE_SINGLE_WRITER and/or SINGLE_NODE_MULTI_WRITER are
      // supported, in order to permit older COs to continue working.
      SINGLE_NODE_MULTI_WRITER = 13;
```

Although it controls support for two access modes, `SINGLE_NODE_MULTI_WRTIER`
is chosen as the capability name because it represents the access mode that is
unsupported.

For ReadWriteOncePod, if the CSI driver supports the `SINGLE_NODE_MULTI_WRTER`
capability, then ReadWriteOncePod will map to `SINGLE_NODE_SINGLE_WRITER`. If
it does not, then ReadWriteOncePod will map to `SINGLE_NODE_WRITER`. This
mapping is chosen because we can safely rely on Kubernetes to enforce the
access mode outside of the CSI driver. It also has the advantage of enabling
existing CSI drivers to start using ReadWriteOncePod.

For ReadWriteOnce, if the CSI driver supports the `SINGLE_NODE_MULTI_WRITER`
capability, then ReadWriteOnce will map to `SINGLE_NODE_MULTI_WRITER`. If it
does not, then ReadWriteOnce will map to `SINGLE_NODE_WRITER`, which is the
existing behavior.

Put more succinctly:

|                  | Driver Supports `SINGLE_NODE_MULTI_WRITER` Capability | Driver Does Not Support `SINGLE_NODE_MULTI_WRITER` Capability |
|------------------|-------------------------------------------------------|---------------------------------------------------------------|
| ReadWriteOncePod | SINGLE_NODE_SINGLE_WRITER                             | SINGLE_NODE_WRITER                                            |
| ReadWriteOnce    | SINGLE_NODE_MULTI_WRITER                              | SINGLE_NODE_WRITER (Existing behavior)                        |

CSI clients that will need updating are kubelet, external-provisioner,
external-attacher, and external-resizer.

### Test Plan

<!--
**Note:** *Not required until targeted at a release.*

Consider the following in developing a test plan for this enhancement:
- Will there be e2e and integration tests, in addition to unit tests?
- How will it be tested in isolation vs with other components?

No need to outline all of the test cases, just the general strategy. Anything
that would count as tricky in the implementation, and anything particularly
challenging to test, should be called out.

All code is expected to have adequate tests (eventually with coverage
expectations). Please adhere to the [Kubernetes testing guidelines][testing-guidelines]
when drafting this test plan.

[testing-guidelines]: https://git.k8s.io/community/contributors/devel/sig-testing/testing.md
-->

#### Validation of PersistentVolumeSpec Object

To test the validation logic of the PersistentVolumeSpec, we need to check the
following cases:

- Validation succeeds when feature gate is enabled and PersistentVolume is created with
  ReadWriteOncePod access mode
- Validation fails when feature gate is disabled and PersistentVolume is created with
  ReadWriteOncePod access mode
- Validation succeeds when feature gate is enabled and PersistentVolumeClaim is created with
  ReadWriteOncePod access mode
- Validation fails when feature gate is disabled and PersistentVolumeClaim is created with
  ReadWriteOncePod access mode

#### Mounting and Mapping with ReadWriteOncePod

To test mount behavior, we need to check the following cases:

- Mounting a volume with ReadWriteOncePod succeeds if the volume isn't already
  mounted
- Mounting a volume with ReadWriteOncePod fails if the volume is already mounted

#### Mounting and Mapping with ReadWriteOnce

Existing unit tests should cover this scenario.

#### Mapping Kubernetes Access Modes to CSI Volume Capability Access Modes

This test involves asserting the behavior in the above table. The volume
capability access mode for ReadWriteOnce will depend on the capabilities of the
CSI driver. A test asserting this behavior will be needed in both Kubernetes as
well as in CSI sidecars.

#### End to End Tests

To test this feature end to end, we will need to check the following cases:

- A ReadWriteOncePod volume will succeed mounting when consumed by a single pod
  on a node
- A ReadWriteOncePod volume will fail to mount when consumed by a second pod on
  the same node
- A ReadWriteOncePod volume will fail to attach when consumed by a second pod on
  a different node

For testing the mapping for ReadWriteOnce, we should update the mock CSI driver
to support the new volume capability access modes and cut a release. The
existing Kubernetes end to end tests will be updated to use this version which
will test the mapping behavior because most storage end to end tests rely on the
ReadWriteOnce access mode.

### Graduation Criteria

<!--
**Note:** *Not required until targeted at a release.*

Define graduation milestones.

These may be defined in terms of API maturity, or as something else. The KEP
should keep this high-level with a focus on what signals will be looked at to
determine graduation.

Consider the following in developing the graduation criteria for this enhancement:
- [Maturity levels (`alpha`, `beta`, `stable`)][maturity-levels]
- [Deprecation policy][deprecation-policy]

Clearly define what graduation means by either linking to the [API doc
definition](https://kubernetes.io/docs/concepts/overview/kubernetes-api/#api-versioning)
or by redefining what graduation means.

In general we try to use the same stages (alpha, beta, GA), regardless of how the
functionality is accessed.

[maturity-levels]: https://git.k8s.io/community/contributors/devel/sig-architecture/api_changes.md#alpha-beta-and-stable-versions
[deprecation-policy]: https://kubernetes.io/docs/reference/using-api/deprecation-policy/

Below are some examples to consider, in addition to the aforementioned [maturity levels][maturity-levels].

#### Alpha -> Beta Graduation

- Gather feedback from developers and surveys
- Complete features A, B, C
- Tests are in Testgrid and linked in KEP

#### Beta -> GA Graduation

- N examples of real-world usage
- N installs
- More rigorous forms of testing—e.g., downgrade tests and scalability tests
- Allowing time for feedback

**Note:** Generally we also wait at least two releases between beta and
GA/stable, because there's no opportunity for user feedback, or even bug reports,
in back-to-back releases.

#### Removing a Deprecated Flag

- Announce deprecation and support policy of the existing flag
- Two versions passed since introducing the functionality that deprecates the flag (to address version skew)
- Address feedback on usage/changed behavior, provided on GitHub issues
- Deprecate the flag

**For non-optional features moving to GA, the graduation criteria must include 
[conformance tests].**

[conformance tests]: https://git.k8s.io/community/contributors/devel/sig-architecture/conformance-tests.md
-->

#### Alpha

- CSI spec supports `SINGLE_NODE_*_WRITER` access modes
- Kubernetes supports ReadWriteOncePod access mode, has unit test coverage, has
  updated CSI spec
- CSI sidecars support `SINGLE_NODE_*_WRITER` access modes and have unit test
  coverage

#### Beta

- Scheduler enforces ReadWriteOncePod access mode by marking pods as
  Unschedulable, preemption logic added
- ReadWriteOncePod access mode has end to end test coverage
- Mock CSI driver supports `SINGLE_NODE_*_WRITER` access modes, relevant end to
  end tests updated to use this driver
- Hostpath CSI driver supports `SINGLE_NODE_*_WRITER` access modes, relevant end
  to end tests updated to use this driver

#### GA

- Kubernetes API and CSI spec changes are stable
- CSI drivers support `SINGLE_NODE_*_WRITER` access modes

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

In order to upgrade a cluster to use this feature, the user will need to restart
the kube-apiserver, kube-scheduler, and kubelet with the ReadWriteOncePod
feature gate enabled.  Additionally they will need to update their CSI drivers
and sidecars to versions that depend on the new Kubernetes API and CSI spec.

When downgrading a cluster to disable this feature, the user will need to
restart the kube-apiserver with the ReadWriteOncePod feature gate disabled. When
disabling this feature gate, any existing volumes with the ReadWriteOncePod
access mode will continue to exist, but can only be deleted. An alternative is
to allow these volumes to be treated as ReadWriteOnce, however that would
violate the intent of the user and so it is not recommended.

If a user downgrades their CSI drivers or sidecars, any existing volumes using
ReadWriteOnce should continue working (switching from `SINGLE_NODE_MULTI_WRITER`
to `SINGLE_NODE_WRITER`). This behavior is ultimately up to each CSI driver, but
they should be designed with this backwards compatibility in mind.

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


#### API Server Version N / Scheduler Version N / Kubelet Version N-1 or N-2

When starting two pods with both using the same PVC with ReadWriteOncePod, one pod
will successfully start, but the other will not be scheduled due to the
ReadWriteOncePod access mode conflict.

When starting the same two pods but also setting `pod.spec.nodeName` to the same
node, kubelet will not enforce the access mode and will proceed with starting
both pods.

For older kubelets, [ReadWriteOncePod will map to access mode `UNKNOWN`]. How
this access mode is used will vary across CSI drivers. By definition, the CSI
spec says ["If ANY of the specified volume capabilities are not supported by the
SP, the call MUST return the appropriate gRPC error code"], see the
`volume_capabilities` field in CreateVolumeRequest. However, not all CSI drivers
strictly adhere to this spec. For example, the EBS CSI driver will [error when
supplied an unsupported access mode]. Other drivers like the mock CSI driver
[won't check the supplied access modes], meaning `UNKNOWN` is valid.

[ReadWriteOncePod will map to access mode `UNKNOWN`]: https://github.com/kubernetes/kubernetes/blob/v1.21.0/pkg/volume/csi/csi_client.go#L512
[error when supplied an unsupported access mode]: https://github.com/kubernetes-sigs/aws-ebs-csi-driver/blob/v1.0.0/pkg/driver/controller.go#L117-L122
[won't check the supplied access modes]: https://github.com/kubernetes-csi/csi-test/blob/v4.2.0/mock/service/controller.go#L44-L46
["If ANY of the specified volume capabilities are not supported by the SP, the call MUST return the appropriate gRPC error code"]: https://github.com/container-storage-interface/spec/blob/v1.4.0/spec.md#createvolume

#### API Server Version N / Scheduler Version N-1 / Kubelet Version N-1 or N-2

When creating a pod using ReadWriteOncePod, the scheduler will not enforce this
access mode during scheduling. It will be possible for two pods using the same
PVC with this access mode to be assigned the same node.

Same as the above case, with an older kubelet ReadWriteOncePod will map to
access mode `UNKNOWN`. How this access mode is used will vary across CSI
drivers.

#### API Understands ReadWriteOncePod, CSI Sidecars Do Not

Both the the [CSI attacher] and the [CSI resizer] will error if they do not
understand ReadWriteOncePod and this access mode is used on a PV.

The CSI provisioner will [map ReadWriteOncePod to a nil access mode]. How this
access mode is used will vary across CSI drivers.

[CSI attacher]: https://github.com/kubernetes-csi/external-attacher/blob/v3.2.0/pkg/controller/util.go#L196-L197
[CSI resizer]: https://github.com/kubernetes-csi/external-resizer/blob/v1.2.0/pkg/resizer/csi_resizer.go#L237-L238
[map ReadWriteOncePod to a nil access mode]: https://github.com/kubernetes-csi/external-provisioner/blob/v2.2.0/pkg/controller/controller.go#L468-L469

#### CSI Controller Service Understands New CSI Access Modes, CSI Node Service Does Not

If the CSI driver running the controller service understands the new access
modes, then volumes will be provisioned and attached using these access modes
(if ReadWriteOncePod or ReadWriteOnce are used). If the CSI driver running the
node service does not understand these access modes, the behavior will depend on
the CSI driver and how it treats unknown access modes. The recommendation is to
upgrade the CSI drivers for the controller and node services together.

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
-->

- [X] Feature gate (also fill in values in `kep.yaml`)
  - Feature gate name: ReadWriteOncePod
  - Components depending on the feature gate:
    - kube-apiserver
    - kube-scheduler
    - kubelet

###### Does enabling the feature change any default behavior?

<!--
Any change of default behavior may be surprising to users or break existing
automations, so be extremely careful here.
-->

No.

###### Can the feature be disabled once it has been enabled (i.e. can we roll back the enablement)?

<!--
Describe the consequences on existing workloads (e.g., if this is a runtime
feature, can it break the existing applications?).

NOTE: Also set `disable-supported` to `true` or `false` in `kep.yaml`.
-->

When the feature gate is disabled, existing ReadWriteOncePod volumes will
continue working. The only allowed operation will be the deletion of
ReadWriteOncePod volumes.

###### What happens if we reenable the feature if it was previously rolled back?

Any existing ReadWriteOncePod and ReadWriteOnce volumes will continue working.
Upon re-enabling of the feature gate, users can begin creating ReadWriteOncePod
volumes again.

###### Are there any tests for feature enablement/disablement?

<!--
The e2e framework does not currently support enabling or disabling feature
gates. However, unit tests in each component dealing with managing data, created
with and without the feature, are necessary. At the very least, think about
conversion tests if API types are being modified.
-->

There will be unit test coverage for API validation and mount behavior with the
feature gate enabled and disabled. There will also be end to end test coverage
for mount behavior (if the the feature gate is enabled).

### Rollout, Upgrade and Rollback Planning

<!--
This section must be completed when targeting beta to a release.
-->

###### How can a rollout fail? Can it impact already running workloads?

<!--
Try to be as paranoid as possible - e.g., what if some components will restart
mid-rollout?
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
-->

###### How can an operator determine if the feature is in use by workloads?

<!--
Ideally, this should be a metric. Operations against the Kubernetes API (e.g.,
checking if there are objects with field X set) may be a last resort. Avoid
logs or events for this purpose.
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

###### What are the reasonable SLOs (Service Level Objectives) for the above SLIs?

<!--
At a high level, this usually will be in the form of "high percentile of SLI
per day <= X". It's impossible to provide comprehensive guidance, but at the very
high level (needs more precise definitions) those may be things like:
  - per-day percentage of API calls finishing with 5XX errors <= 1%
  - 99% percentile over day of absolute value from (job creation time minus expected
    job creation time) for cron job <= 10%
  - 99,9% of /health requests per day finish with 200 code
-->

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

No.

###### Will enabling / using this feature result in introducing new API types?

<!--
Describe them, providing:
  - API type
  - Supported number of objects per cluster
  - Supported number of objects per namespace (for namespace-scoped objects)
-->

No, it will introduce a new "ReadWriteOncePod" value for the
PersistentVolumeAccessMode type, added to the [internal] and [v1] APIs.

[internal]: https://github.com/kubernetes/kubernetes/blob/v1.21.0/pkg/apis/core/types.go#L503-L514
[v1]: https://github.com/kubernetes/kubernetes/blob/v1.21.0/staging/src/k8s.io/api/core/v1/types.go#L556-L565

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

###### Will enabling / using this feature result in non-negligible increase of resource usage (CPU, RAM, disk, IO, ...) in any components?

<!--
Things to keep in mind include: additional in-memory state, additional
non-trivial computations, excessive access to disks (including increased log
volume), significant amount of data sent and/or received over network, etc.
This through this both in small and large cases, again with respect to the
[supported limits].

[supported limits]: https://git.k8s.io/community//sig-scalability/configs-and-limits/thresholds.md
-->

No, the solution will involve using the same ActualStateOfWorld cache in kubelet.

### Troubleshooting

<!--
This section must be completed when targeting beta to a release.

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

- 3/10/2021: Implementation started

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

When it comes to handling ReadWriteOnce, an alternative that was considered was
not introducing a `SINGLE_NODE_MULTI_WRITER` access mode in the CSI spec and
continuing to use `SINGLE_NODE_WRITER`. This solution was ruled out because the
`SINGLE_NODE_WRITER` access mode has conflicting definitions, and since we're
introducing a `SINGLE_NODE_SINGLE_WRITER` access mode we should also address
this issue to reduce confusion for developers.

## Infrastructure Needed (Optional)

<!--
Use this section if you need things from the project/SIG. Examples include a
new subproject, repos requested, or GitHub details. Listing these here allows a
SIG to get the process for these resources started right away.
-->

None.
