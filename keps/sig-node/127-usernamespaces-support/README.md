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
# KEP-127: Support User Namespaces

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
    - [Volumes Support](#volumes-support)
    - [Container Runtime Support](#container-runtime-support)
  - [Risks and Mitigations](#risks-and-mitigations)
    - [Breaking Existing Workloads](#breaking-existing-workloads)
    - [Duplicated Snapshots of Container Images](#duplicated-snapshots-of-container-images)
- [Implementation Phases](#implementation-phases)
  - [Phase 1](#phase-1)
  - [Phase 2](#phase-2)
  - [Phase 2+](#phase-2-1)
- [Design Details](#design-details)
  - [Summary of the Proposed Changes](#summary-of-the-proposed-changes)
  - [PodSpec Changes](#podspec-changes)
  - [CRI API Changes](#cri-api-changes)
  - [Test Plan](#test-plan)
  - [Graduation Criteria](#graduation-criteria)
    - [Alpha -&gt; Beta](#alpha---beta)
    - [Beta -&gt; GA](#beta---ga)
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
  - [Differences with Previous Proposal](#differences-with-previous-proposal)
  - [Default Value for userNamespaceMode](#default-value-for-usernamespacemode)
  - [Host Defaulting Mechanishm](#host-defaulting-mechanishm)
- [References](#references)
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
- [ ] (R) Test plan is in place, giving consideration to SIG Architecture and SIG Testing input
- [ ] (R) Graduation criteria is in place
- [ ] (R) Production readiness review completed
- [ ] Production readiness review approved
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

Container security consists of many different kernel features that work together
to make containers secure. User namespaces isolate user and group IDs by
allowing processes to run with different IDs in the container and in the host.
Specially, a process running as privileged in a container can be seen as
unprivileged in the host. This gives more capabilities to the containers and
protects the host from malicious or compromised containers.

This KEP is a continuation of the work initiated in the [Support Node-Level User Namespaces Remapping](https://github.com/kubernetes/community/blob/master/contributors/design-proposals/node/node-usernamespace-remapping.md) proposal.

## Motivation

From [user_namespaces(7)](https://man7.org/linux/man-pages/man7/user_namespaces.7.html):
> User namespaces isolate security-related identifiers and attributes, in
particular, user IDs and group IDs, the root directory, keys, and capabilities.
A process's user and group IDs can be different inside and outside a user
namespace. In particular, a process can have a normal unprivileged user ID
outside a user namespace while at the same time having a user ID of 0 inside
the namespace; in other words, the process has full privileges for operations
inside the user namespace, but is unprivileged for operations outside the
namespace.

The goal of supporting user namespaces in Kubernetes is to be able to run
processes in pods with a different user and group IDs than in the host.
Speficically, a privileged process in the pod runs as an unprivileged process in the
host. If such a process is able to break into the host, it'll have limited
impact as it'll be running as an unprivileged user there.

There have been some security vulnerabilities that could have been mitigated by
user namespaces. Some examples are:
- CVE-2016-8867: Privilege escalation inside containers
  - https://github.com/moby/moby/issues/27590
- CVE-2018-15664: TOCTOU race attack that allows to read/write files in the host
  - https://github.com/moby/moby/pull/39252
- CVE-2019-5736: Host runc binary can be overwritten from container
  - https://github.com/opencontainers/runc/commit/0a8e4117e7f715d5fbeef398405813ce8e88558b

### Goals

- Increase node to pod isolation in Kubernetes by mapping user and group IDs
  inside the container to different IDs in the host. In particular, mapping root
  inside the container to unprivileged user and group IDs in the node.

### Non-Goals

TODO(Mauricio)

## Proposal

This proposal aims to support user namespaces in Kubernetes by extending the pod
specification with a new `userNamespaceMode` field. This proposal aims to
support three modes.

- **Host**:
  The pods are placed in the host user namespace, this is the current Kubernetes
  behaviour. This mode is intended for pods that only work in the root (host)
  user namespace. It is the default mode when `userNamespaceMode` field is not
  set.

- **Cluster**:
  All the pods in the cluster are placed in a different user namespace but they
  use the same ID mappings. This mode doesn't provide full pod-to-pod isolation
  as all the pods with `Cluster` mode have the same effective IDs on the host.
  It provides pod-to-host isolation as the IDs are different inside the
  container and in the host. This mode is intended for pods sharing volumes as
  they will run with the same effective IDs on the host, allowing them to read
  and write files in the shared storage.

- **Pod**:
  Each pod is placed in a different user namespace and has a different and
  non-overlapping ID mappings. This mode is intended for stateless pods, i.e.
  pods using only ephemeral volumes like `configMap,` `downwardAPI`, `secret`,
  `projected` and `emptyDir`. This mode not only provides host-to-pod isolation
  but also  pod-to-pod isolation as each pod has a different range of effective
  IDs in the host.

### User Stories

#### Story 1

As a cluster admin, I want run some pods with privileged capabilities because the applications in the pods require it (e.g. `CAP_SYS_ADMIN` to mount a FUSE filesystem or `CAP_NET_ADMIN` to setup a VPN) but I don't want this extra capability to give any extra privilege on the host.

#### Story 2

As a cluster admin, I want to allow some pods to share the host user namespace if they need a feature only available in such user namespace.

### Notes/Constraints/Caveats

#### Volumes Support

The Linux kernel uses the effective user and group IDs (the ones the host) to
check the file access permissions. Since with user namespaces IDs are mapped to
a different value on the host, this could cause issues accessing volumes if the
pod is run with a different mapping, i.e. the effective user and group IDs on
the host change.

This proposal supports volume without changing the user and group IDs and leaves
that problem to the user to manage. Future Linux kernel features such as shiftfs
could allow to different pods to see a volume with its own IDs but it is out of
scope of this proposal. Among the possible future kernel solutions, we can list:

- [shiftfs: uid/gid shifting filesystem](https://lwn.net/Articles/757650/)
- [A new API for mounting filesystems](https://lwn.net/Articles/753473/)
- [user_namespace: introduce fsid mappings](https://lwn.net/Articles/812221/)

In regard to this proposal, volumes can be divided in ephemeral and non-ephemeral.

Ephemeral volumes are associated to a **single** pod and their lifecyle is
dependent on that pod. These are `configMap`, `secret`, `emptyDir`,
`downwardAPI`, etc. These kind of volumes are easy to handle as they are not
shared by different pods and hence all the process accessig those volumes have
the same effective user and group IDs. Kubelet creates the files for those
volumes and it can easily set the file ownership too.

Non-ephemeral volumes more difficult to support since they can be persistent and
also can be shared by multiple pods. This proposal supports volumes with two
different strategies:
- The `Cluster` makes it easier for pods to share files using volumes when those
  don't have access permissions for `others` because the effective user and
  group IDs on the host are the same for all the pods.
- The semantics of semantics of `fsGroup` are respected, if it's specified it's
  assumed to be the correct GID in the host and an 1-to-1 mapping entry for the
  `fsGroup` is added to the GID mappings for the pod.

There are some cases that require special attention from the user. For instance, a process
inside a pod will not be able to access files with mode `0700` and owned by a
user different than the effective user of that process in a volume that
doesn't support the semantics of `fsGroup` (doesn't support
[`SetVolumeOwnership`](https://github.com/kubernetes/kubernetes/blob/00da04ba23d755d02a78d5021996489ace96aa4d/pkg/volume/volume_linux.go#L42)
that updates permissions and ownership of the files to be accesible by the `fsGroup`
group ID). These pods should be run in `Host` mode.

#### Container Runtime Support

- **Docker**:
  It only supports a [single IDs
  mapping](https://docs.docker.com/engine/security/userns-remap/) shared by all
  containers running in the host. There is not support for [multiple IDs
  mapping](https://github.com/moby/moby/issues/28593) yet. Dockershim runtime is
  only compatible with pods running in `Host` and `Cluster` modes. The user has
  to guarantee that the ID mappings configured in Docker through the
  `userns-remap` and the cluster-wide range configured in the Kubelet are the
  same. The dockershim implementation includes a check to verify that the IDs
  mapping received from the Kubelet are equal to the ones configured in Docker,
  returning an error otherwise.
- **containerd**:
  It's quite straigtforward to implement the CRI changes proposed below in
  containerd/cri, we did it in
  [this](https://github.com/kinvolk/containerd-cri/commits/mauricio/userns_poc)
  PoC.
- **cri-o**:
  It recently [added](https://github.com/cri-o/cri-o/pull/3944) support for
  user namespaces through a pod annotation. The extensions to make it work with
  the current CRI changes are small.
- TODO(Mauricio): gVisor, katacontainers?

containerd and cri-o will provide support for the 3 possible values of `userNamespaceMode`.

### Risks and Mitigations

#### Breaking Existing Workloads

Some features that could not work when the host user namespace is not shared are:

- **Some Capabilities**:
  The Linux kernel takes into consideration the user namespace a process is
  running in while performing the capabilities check. There are some
  capabilities that are only available in the root (host) user namespace such as
  `CAP_SYS_TIME`, `CAP_SYS_MODULE` & `CAP_MKNOD`.

- **Sharing Host Namespaces**:
  There are some limitations in the Linux kernel and in the runtimes that
  prevents sharing other host namespaces when the host user namespace is not
  shared.
  TODO(Mauricio): Put links to those limitations?

In order to avoid breaking existing workloads `Host` is the default value of `userNamespaceMode`.

#### Duplicated Snapshots of Container Images

The owners of the files of a container image have to been set accordingly to the
ID mappings used for that container. The runtimes perform a `chown` operation
over the image snapshot when it's pulled. This presents a risk as it potentially
increases the time and the storage needed to handle the container images.

There is not immediate mitigation for it, [we talked](https://lists.linuxfoundation.org/pipermail/containers/2020-September/042230.html) to the Linux kernel community and [they replied](https://lists.linuxfoundation.org/pipermail/containers/2020-September/042230.html) they are working on a solution for it.

Another risk is exausting the disk space on the nodes if pods are repeativily started and stopped while using `Pod` mode.

TODO(Mauricio): How to mitigate it?
- could kubelet ask for garbage collecting the snapshots of the images?

## Implementation Phases

The implemenation of this KEP in a single phase is complicated as there are many
discussions to be done. We learned from previous attempts to bring this support in
that it should be done in small steps to avoid losing the focus on the
discussion. It's also true that a full plan should be agreed at the beginning to
avoid changing the implementation drastically in further phases.

This proposal implementation aims to be divided in the following phases:

### Phase 1

The first phase includes:
 - Extend the PodSpec with the `userNamespaceMode` field.
 - Extend the CRI with user and ID mappings fields.
 - Implement support for `Host` and `Cluster` user namespace modes.

The goal of this phase is to implement some initial user namespace support
providing pod-to-host isolation and supporting volumes. The implementation of
the `Pod` mode is out of scope in this phase because it requires a non
negligible amount of work and we could risk losing the focus failing to deliver
this feature.

### Phase 2

This phase aims to implement the `Pod` mode. After this phase is completed the
full advantanges of user namespaces could be used in some cases (stateless
workloads). Once this phase is completed the user namespaces support could be
moved to beta.

### Phase 2+

The default value of `userNamespaceMode` should be set to `Pod` so pods that
don't set this field can also get the security benefits of user namespaces. It's
not clear yet what should be the process to make this happen as this is a
potentially non backwards compatible change. It's specially relevant for
workloads not compatible with user namespaces.

## Design Details

This section only focuses on phase 1 as specified above.

### Summary of the Proposed Changes

- Extend the CRI to have a user namespace mode and the user and group ID mappings.
- Add a `userNamespaceMode` field to the pod spec.
- Add the cluster-wide ID mappings to the kubelet configuration file.
- Add a `UserNamespacesSupport` feature flag to enable / disable the user.
  namespaces support.
- Update owner of ephemeral volumes populated by the kubelet.

### PodSpec Changes

`v1.PodSpec` is extended with a new `UserNamesapceMode` field:

```
const (
	UserNamespaceModeHost    PodUserNamespaceMode = "Host"
	UserNamespaceModeCluster PodUserNamespaceMode = "Cluster"
	UserNamespaceModePod     PodUserNamespaceMode = "Pod"
)

type PodSpec struct {
...
  // UserNamespaceMode controls how user namespaces are used for this Pod.
  // Three modes are supported:
  // "Host": The pod shares the host user namespace. (default value).
  // "Cluster": The pod uses a cluster-wide configured ID mappings.
  // "Pod": The pod gets a non-overlapping ID mappings range.
  // +k8s:conversion-gen=false
  // +optional
  UserNamespaceMode PodUserNamespaceMode `json:"userNamespaceMode,omitempty" protobuf:"bytes,36,opt  name=userNamespaceMode"`
...
```

### CRI API Changes

The CRI has to be extended to allow kubelet to specify the user namespace mode
and the ID mappings for a pod.
[`NamespaceOption`](https://github.com/kubernetes/cri-api/blob/1eae59a7c4dee45a900f54ea2502edff7e57fd68/pkg/apis/runtime/v1alpha2/api.proto#L228)
is extended with two new fields:
- A `user` `NamespaceMode` that defines if the pod should run in an independent
  user namespace (`POD`) or if it should share the host user namespace
  (`NODE`).
- The ID mappings to be used if the user namespace mode is `POD`.

```
// LinuxIDMapping represents a single user namespace mapping in Linux.
message LinuxIDMapping {
   // container_id is the starting ID for the mapping inside the container.
   uint32 container_id = 1;
   // host_id is the starting ID for the mapping on the host.
   uint32 host_id = 2;
   // number of IDs in this mapping.
   uint32 length = 3;
}

// LinuxUserNamespaceConfig represents the user and group ID mapping in Linux.
message LinuxUserNamespaceConfig {
   // uid_mappings is an array of user ID mappings.
   repeated LinuxIDMapping uid_mappings = 1;
   // gid_mappings is an array of group ID mappings.
   repeated LinuxIDMapping gid_mappings = 2;
}

// NamespaceOption provides options for Linux namespaces.
message NamespaceOption {
    ...
    // User namespace for this container/sandbox.
    // Note: There is currently no way to set CONTAINER scoped user namespace in the Kubernetes API.
    // Namespaces currently set by the kubelet: POD, NODE
    Namespace user = 5;
    // ID mappings to use when the user namespace mode is POD.
    LinuxUserNamespaceConfig mappings = 6;
}
```

### Test Plan

TBD

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

### Graduation Criteria

TBD
Mauricio: Should we require Pod mode to be implemented to switch to Beta?

#### Alpha -> Beta

- Future Complete:
  - `Pod` mode implemented

#### Beta -> GA

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

The container runtime will have to be updated in the nodes to support this feature.

The new `user` field in the `NamespaceOption` will be ignored by an old runtime
without user namespaces support. The container will be placed in the host user
namespace. It's a responsibility of the user to guarante that a runtime
supporting user namespaces is used.

If an old version of kubelet without user namespaces support could cause some
issues too. In this case the runtime could wrongly infer that the `user` field
is set to `POD` in the `NamespaceOption` message. To avoid this problem the
runtime should check if the `mappings` field contains any mappings, an error
should be raised otherwise.

## Production Readiness Review Questionnaire

<!--

Production readiness reviews are intended to ensure that features merging into
Kubernetes are observable, scalable and supportable; can be safely operated in
production environments, and can be disabled or rolled back in the event they
cause increased failures in production. See more in the PRR KEP at
https://git.k8s.io/enhancements/keps/sig-architecture/20190731-production-readiness-review-process.md.

The production readiness review questionnaire must be completed for features in
v1.19 or later, but is non-blocking at this time. That is, approval is not
required in order to be in the release.

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

* **How can this feature be enabled / disabled in a live cluster?**
  - [X] Feature gate
    - Feature gate name: UserNamespacesSupport
    - Components depending on the feature gate: kubelet

* **Does enabling the feature change any default behavior?**
  The default mode for usernamespaces is `Host`, so the default behaviour is not changed.

* **Can the feature be disabled once it has been enabled (i.e. can we roll back
  the enablement)?**
  Yes, by disabling the `UserNamespacesSupport` feature gate.
  Pods that were running in `Cluster` mode could have problems to access files saved in permanement storage as the effective user ID in the host would be the one the process is using in the container.

* **What happens if we reenable the feature if it was previously rolled back?**
  Pods using the `Cluster` mode will have a different effective user and group IDs on the host. The same issues accessing files in permanent storage are valid for this case.

* **Are there any tests for feature enablement/disablement?**
  TBD

### Rollout, Upgrade and Rollback Planning

Will be added before transition to beta.

* **How can a rollout fail? Can it impact already running workloads?**

* **What specific metrics should inform a rollback?**

* **Were upgrade and rollback tested? Was the upgrade->downgrade->upgrade path tested?**

* **Is the rollout accompanied by any deprecations and/or removals of features, APIs,
fields of API types, flags, etc.?**


### Monitoring Requirements

Will be added before transition to beta.

* **How can an operator determine if the feature is in use by workloads?**

* **What are the SLIs (Service Level Indicators) an operator can use to determine
the health of the service?**

* **What are the reasonable SLOs (Service Level Objectives) for the above SLIs?**

* **Are there any missing metrics that would be useful to have to improve observability
of this feature?**

### Dependencies

* **Does this feature depend on any specific services running in the cluster?**: No.

### Scalability

* **Will enabling / using this feature result in any new API calls?** No.

* **Will enabling / using this feature result in introducing new API types?** No.

* **Will enabling / using this feature result in any new calls to the cloud
provider?** No.

* **Will enabling / using this feature result in increasing size or count of
the existing API objects?** Yes. The PodSpec will be increased. TODO(Mauricio): what is the increased size?

* **Will enabling / using this feature result in increasing time taken by any
operations covered by [existing SLIs/SLOs]?** Yes. The runtime will have to set the correct owner on the container image before starting it. TODO(Mauricio): check it out

* **Will enabling / using this feature result in non-negligible increase of
resource usage (CPU, RAM, disk, IO, ...) in any components?**: No.

### Troubleshooting

Will be added before transition to beta.

* **How does this feature react if the API server and/or etcd is unavailable?**

* **What are other known failure modes?**

* **What steps should be taken if SLOs are not being met to determine the problem?**

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

TBD:
Some ideas
- another configuration knob is added
- user namespaces could make troubleshooting difficult
- volumes are really trickly to handle
- any performance issues?

## Alternatives

### Differences with Previous Proposal
Even if this KEP is heavily based on the previous [Support Node-Level User
Namespaces
Remapping](https://github.com/kubernetes/community/blob/master/contributors/design-proposals/node/node-usernamespace-remapping.md)
proposal there are some big differences:
- The previous proposal aimed to configure the ID mappings in the container
  runtime instead of the kubelet. In this proposal this decision is made in the
  kubelet because:
  - It has knowledge of Kubernetes elements like volumes, pods, etc.
  - Runtimes will be more simple as they don't have to implement logic to
    allocate non-overlapping ID mappings.
  - We keep the behaviour consistent among runtimes as kubelet will be the one
    ordering what to do.
- That proposal didn't consider having different ID mappings for each pod. Even
  if it's not planned for the first phase, this KEP takes that into
  consideration performing the needed changes in the CRI from the beginning.

### Default Value for userNamespaceMode

This proposal intends to have `Host` instead of `Pod` as default value for the
user namespace mode. The rationale behind this decision is that it avoids
breaking existing workloads that don't work with user namespaces. We are aware
that this decision has the drawback that pods that have the `userNamespaceMode`
set will not have the security advantages of user namespaces but we consider
it's more important to keep compatibility with previous workloads.

### Host Defaulting Mechanishm

Previous proposals like [Node-Level UserNamespace
implementation](https://github.com/kubernetes/kubernetes/pull/64005) had a
mechanism to default to the host user namespace when the pod specification includes
features that could be not compatible with user namespaces (similar to [Default host user
namespace via experimental
flag](https://github.com/kubernetes/kubernetes/pull/31169)).

This proposal doesn't require a similar mechanishm given that the default mode
is `Host` that works with all current existing workloads.

## References

- Support Node-Level User Namespaces Remapping design proposal document.
  - https://github.com/kubernetes/community/blob/master/contributors/design-proposals/node/node-usernamespace-remapping.md
- Node-Level UserNamespace implementation
  - https://github.com/kubernetes/kubernetes/pull/64005
- Support node-level user namespace remapping
  - https://github.com/kubernetes/enhancements/issues/127
- Default host user namespace via experimental flag
  - https://github.com/kubernetes/kubernetes/pull/31169
- Add support for experimental-userns-remap-root-uid and
  experimental-userns-remap-root-gid options to match the remapping used by the
  container runtime
  - https://github.com/kubernetes/kubernetes/pull/55707
- Track Linux User Namespaces in the pod Security Policy
  - https://github.com/kubernetes/kubernetes/issues/59152
