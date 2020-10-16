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
    - [Container Images with High IDs](#container-images-with-high-ids)
- [Implementation Phases](#implementation-phases)
  - [Phase 1](#phase-1)
  - [Future Phases](#future-phases)
- [Design Details](#design-details)
  - [Summary of the Proposed Changes](#summary-of-the-proposed-changes)
  - [CRI API Changes](#cri-api-changes)
  - [Add userNamespaceMode Field](#add-usernamespacemode-field)
    - [Option 1: PodSpec](#option-1-podspec)
    - [Option 2: PodSecurityContext](#option-2-podsecuritycontext)
  - [Configuring the Cluster ID Mappings](#configuring-the-cluster-id-mappings)
    - [Option 1: Configure in Kubelet Configuration File](#option-1-configure-in-kubelet-configuration-file)
    - [Option 2: Configure as a Cluster Parameter in kube-apiserver](#option-2-configure-as-a-cluster-parameter-in-kube-apiserver)
  - [1-to-1 Mapping for fsGroup](#1-to-1-mapping-for-fsgroup)
  - [Updating Ownership of Ephemeral Volumes](#updating-ownership-of-ephemeral-volumes)
  - [Test Plan](#test-plan)
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
  - [Differences with Previous Proposal](#differences-with-previous-proposal)
  - [Default Value for userNamespaceMode](#default-value-for-usernamespacemode)
  - [Host Defaulting Mechanishm](#host-defaulting-mechanishm)
- [References](#references)
<!-- /toc -->

## Release Signoff Checklist

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

[kubernetes.io]: https://kubernetes.io/
[kubernetes/enhancements]: https://git.k8s.io/enhancements
[kubernetes/kubernetes]: https://git.k8s.io/kubernetes
[kubernetes/website]: https://git.k8s.io/website

## Summary

Container security consists of many different kernel features that work together
to make containers secure. User namespaces isolate user and group IDs by
allowing processes to run with different IDs in the container and in the host.
Specially, a process running with UID/GID 0 in a container can run with UID/GID != 0 in
the host. This makes it possible to give more capabilities to the containers and
protects the host and other containers from malicious or compromised containers.

This KEP adds a new `userNamespaceMode` field  to `pod.Spec`. It allows users to
place pods in different user namespaces increasing the  pod-to-pod and
pod-to-host isolation. This extra isolation increases the cluster security as it
protects the host and other pods from malicious or compromised processes inside
containers that are able to break into the host. This KEP proposes three
different modes: `Host` uses the host user namespace like the current behaviour,
`Cluster` uses a unique user namespace per pod but the same ID mapping for all
the pods (very similar to the previous [Support Node-Level User Namespaces
Remapping](https://github.com/kubernetes/community/blob/master/contributors/design-proposals/node/node-usernamespace-remapping.md)
proposal) and `Pod` increases pod-to-pod isolation by giving each pod a
different and non-overlapping ID mapping.

## Motivation

From
[user_namespaces(7)](https://man7.org/linux/man-pages/man7/user_namespaces.7.html):
> User namespaces isolate security-related identifiers and attributes, in
particular, user IDs and group IDs, the root directory, keys, and capabilities.
A process's user and group IDs can be different inside and outside a user
namespace. In particular, a process can have a normal unprivileged user ID
outside a user namespace while at the same time having a user ID of 0 inside the
namespace; in other words, the process has full privileges for operations inside
the user namespace, but is unprivileged for operations outside the namespace.

The goal of supporting user namespaces in Kubernetes is to be able to run
processes in pods with a different user and group IDs than in the host.
Specifically, a privileged process in the pod runs as an unprivileged process in
the host. If such a process is able to break out of the container to the host,
it'll have limited impact as it'll be running as an unprivileged user there.

The following security vulnerabilities were mitigated with user namespaces and
it is expected that using them would mitigate against some of the future
vulnerabilities.
- [CVE-2016-8867](https://nvd.nist.gov/vuln/detail/CVE-2016-8867): Privilege escalation inside containers
  - https://github.com/moby/moby/issues/27590
- [CVE-2018-15664](https://nvd.nist.gov/vuln/detail/CVE-2018-15664): TOCTOU race attack that allows to read/write files in the host
  - https://github.com/moby/moby/pull/39252
- [CVE-2019-5736](https://nvd.nist.gov/vuln/detail/CVE-2019-5736): Host runc binary can be overwritten from container
  - https://github.com/opencontainers/runc/commit/0a8e4117e7f715d5fbeef398405813ce8e88558b

### Goals

- Increase node to pod isolation in Kubernetes by mapping user and group IDs
  inside the container to different IDs in the host. In particular, mapping root
  inside the container to unprivileged user and group IDs in the node.
- Make it safer to run workloads that need highly privileged capabilities such
  as `CAP_SYS_ADMIN`, reducing the risk of impacting the host.
- Benefit from the security hardening that user namespaces provide against some
  of the future unknown runtime and kernel vulnerabilities.

### Non-Goals

- Provide a way to run the kubelet process or container runtimes as an
  unprivileged process. Although initiatives like
  [usernetes](https://github.com/rootless-containers/usernetes) and this KEP
  both make use of user namespaces, it is a different implementation for a
  different purpose.

## Proposal

This proposal aims to support user namespaces in Kubernetes by extending the pod
specification with a new `userNamespaceMode` field. This field can have 3
values:

- **Host**:
  The pods are placed in the host user namespace, this is the current Kubernetes
  behaviour. This mode is intended for pods that only work in the initial (host)
  user namespace. It is the default mode when `userNamespaceMode` field is not
  set.

- **Cluster**:
  All the pods in the cluster are placed in a _unique_ user namespace but they
  use the same ID mappings. This mode doesn't provide full pod-to-pod isolation
  as all the pods with `Cluster` mode have the same effective IDs on the host.
  It provides pod-to-host isolation as the IDs are different inside the
  container and in the host. This mode is intended for pods sharing volumes as
  they will run with the same effective IDs on the host, allowing them to read
  and write files in the shared storage.

- **Pod**:
  Each pod is placed in a different user namespace and has a different and
  non-overlapping ID mapping across the node. This mode is intended for stateless pods, i.e.
  pods using only ephemeral volumes like `configMap,` `downwardAPI`, `secret`,
  `projected` and `emptyDir`. This mode not only provides host-to-pod isolation
  but also pod-to-pod isolation as each pod has a different range of effective
  IDs in the host.

### User Stories

#### Story 1

As a cluster admin, I want run some pods with privileged capabilities because
the applications in the pods require it (e.g. `CAP_SYS_ADMIN` to mount a FUSE
filesystem or `CAP_NET_ADMIN` to setup a VPN) but I don't want this extra
capability to grant me any extra privilege on the host.

#### Story 2

As a cluster admin, I want to allow some pods to run in the host user namespace
if they need a feature only available in such user namespace, such as loading a
kernel module with `CAP_SYS_MODULE`.

### Notes/Constraints/Caveats

#### Volumes Support

The Linux kernel uses the effective user and group IDs (the ones the host) to
check the file access permissions. Since with user namespaces IDs are mapped to
a different value on the host, this causes issues accessing volumes if the pod
is run with a different mapping, i.e. the effective user and group IDs on the
host change.

This proposal supports volumes without changing the user and group IDs and
leaves that problem for the user to manage. Developing Linux kernel features
such as shiftfs could allow different pods to see a volume with its own IDs but
it is not available yet. Among the possible future kernel solutions, we can
list:

- [shiftfs: uid/gid shifting filesystem](https://lwn.net/Articles/757650/)
- [A new API for mounting filesystems](https://lwn.net/Articles/753473/)
- [user_namespace: introduce fsid mappings](https://lwn.net/Articles/812221/)

In regard to this proposal, volumes can be divided in ephemeral and
non-ephemeral.

Ephemeral volumes are associated to a **single** pod and their lifecyle is
dependent on that pod. These are `configMap`, `secret`, `emptyDir`,
`downwardAPI`, etc. These kind of volumes can work with any of the three
different modes of `userNamespaceMode` as they are not shared by different pods
and hence all the processes accessing those volumes have the same effective user
and group IDs. Kubelet creates the files for those volumes and it can easily set
the file ownership too.

Non-ephemeral volumes are more difficult to support since they can be persistent
and shared by multiple pods. This proposal supports volumes with two different
strategies:
- The `Cluster` mode makes it easier for pods to share files using volumes when
  those don't have access permissions for `others` because the effective user
  and group IDs on the host are the same for all the pods.
- The semantics of `fsGroup` are respected, if it's specified it's
  assumed to be the correct GID in the host and a 1-to-1 mapping entry for the
  `fsGroup` is added to the GID mappings for the pod.

This KEP doesn't impose any restriction on the different volumes and
`userNamespaceMode` combinations and leaves it to users to chose the correct
combinations based on their specific needs. For instance, if a pod access a
shared volume containing files and folders with permissions for `others`, it can
run in `Pod` mode. On the other hand, a process inside a pod will not be able to
access files with mode `0700` and owned by a user different than the effective
user of that process in a volume that doesn't support the semantics of `fsGroup`
(doesn't support
[`SetVolumeOwnership`](https://github.com/kubernetes/kubernetes/blob/00da04ba23d755d02a78d5021996489ace96aa4d/pkg/volume/volume_linux.go#L42)
that updates permissions and ownership of the files to be accesible by the
`fsGroup` group ID). Such pods should be run in `Host` mode.

#### Container Runtime Support

- **Docker**:
  Docker only supports a [single ID
  mappings](https://docs.docker.com/engine/security/userns-remap/) shared by all
  containers running in the host. There is not support for [multiple ID
  mappings](https://github.com/moby/moby/issues/28593) yet. Updating dockershim
  is out of scope of this KEP as [it's going to be
  deprecated]((https://github.com/kubernetes/enhancements/pull/1985/)) and it
  offers a very limited support for user namespaces.
- **containerd**:
  It's quite straightforward to implement the CRI changes proposed below in
  containerd/cri, we did it in
  [this](https://github.com/kinvolk/containerd-cri/commits/mauricio/userns_poc)
  PoC.
- **cri-o**:
  CRI-O recently [added](https://github.com/cri-o/cri-o/pull/3944) support for
  user namespaces through a pod annotation. The extensions to make it work with
  the CRI changes proposed here are small.
- gVisor, katacontainers: Yet to be investigated.

containerd and cri-o will provide support for the 3 possible values of `userNamespaceMode`.

### Risks and Mitigations

#### Breaking Existing Workloads

Some features that don't work when the host user namespace is not shared are:

- **Some Capabilities**:
  The Linux kernel takes into consideration the user namespace a process is
  running in while performing the capabilities check. There are some
  capabilities that are only available in the initial (host) user namespace such
  as `CAP_SYS_TIME`, `CAP_SYS_MODULE` & `CAP_MKNOD`.

  If a pod is given one of those capabilities it will still be deployed, but the
  capability will be ineffective and processes using those capabilities will
  fail. This is not impacting the implementation in Kubernetes. If users need
  the capability to be effective, they should use `userNamespaceMode=Host`.

  The list of such capabilities is likely to change from one Linux version to
  another. For example, Linux now has [time
  namespaces](https://man7.org/linux/man-pages/man7/time_namespaces.7.html) and
  there are ways to make `CAP_SYS_TIME` work inside a user namespace. There are
  also discussions to make `CAP_MKNOD` work in user namespaces.

- **Sharing Host Namespaces**:
  There are some limitations in the Linux kernel and in the runtimes that
  prevent sharing other host namespaces when the host user namespace is not
  shared.
  - Mounting `mqueue` (`/dev/mqueue`) is not allowed from a process in a user
    namespace that does not own the IPC namespace. Pods with `hostIPC=true` and
    `userNamespaceMode=Pod|Cluster` can fail.
  - Mounting `procfs` (`/proc`) is not allowed from a process in a user
    namespace that does not own the PID namespace. Pods with `hostPID=true` and
    `userNamespaceMode=Pod|Cluster` can fail.
  - Mounting `sysfs` (`/sys`) is not allowed from a process in a user namespace
    that does not own the network namespace. Impact: pods with
    `hostNetwork=true` and `userNamespaceMode=Pod|Cluster` can fail.

  If users specify `userNamespaceMode=Pod|Cluster` and one of these
  `host{IPC,PID,Network}=true` options, runc will currently fail to start the
  container. The kubelet does **not** try to prevent that combination of
  options, in case runc or the kernel make it possible in the future to use that
  combination.

In order to avoid breaking existing workloads `Host` is the default value of
`userNamespaceMode`.

#### Duplicated Snapshots of Container Images

The owners of the files of a container image have to been set accordingly to the
ID mappings used for that container. For example, if the user 0 in the container
is mapped to the host user 100000, then the `/root` directory has to be owned by
user ID 100000 in the host, so it appears to belong to root in the container.
The current implementation in container runtimes is to recursively perform a
`chown` operation over the image snapshot when it's pulled. This presents a risk
as it potentially increases the time and the storage needed to handle the
container images.

[containers/storage](https://github.com/containers/storage/) used by CRI-O
mounts an overlay file system with the
[`metacopy=on`](https://www.kernel.org/doc/html/latest/filesystems/overlayfs.html#metadata-only-copy-up)
flag set, it then chowns all of the lower files in the image to match the user
namespace to which the container will run. This operation is very quick compared
to standard chowning, since none of the files data has to be copied up. If a
second container runs on the same image with the same user namespace, then the
chowned image is shared, eliminating the need to chown again.

[fuse-overlayfs](https://github.com/containers/fuse-overlayfs) is a fuse-based
overlayfs implementation that supports ID shifting. It doesn't require a
recursive chown operation (avoiding the file duplication) but given that it's
implemented in user space has lower I/O performance.

More sophisticated approaches to this problem are being
[discussed](https://lists.linuxfoundation.org/pipermail/containers/2020-September/042230.html)
in the kernel community. This is something that container runtimes should use
once it's available and it does not impact the kubelet nor the CRI gRPC spec.

#### Container Images with High IDs

There are container images designed to run with high user and group IDs. It's
possible that the IDs range assigned to the pod is not big enough to accommodate
these IDs, in this case they will be mapped to the `nobody` user in the host.

It's not a big problem in the `Cluster` case, the users have to be sure that
they provide a range accommodating these IDs. It's more difficult to handle in
the `Pod` case as the logic to allocate the ranges for each pod has to take this
information in consideration. It's likely that this requires some changes to the
CRI and kubelet so the runtimes can inform the kubelet what are the IDs present
on a specific container image.

## Implementation Phases

The implementation of this KEP in a single phase is complicated as there are
many discussions to be done. We learned from previous attempts to bring this
support in that it should be done in small steps to avoid losing the focus on
the discussion. It's also true that a full plan should be agreed at the
beginning to avoid changing the implementation drastically in further phases.

This proposal implementation aims to be divided in the following phases:

### Phase 1

This first phase includes:
 - Extend the PodSpec with the `userNamespaceMode` field.
 - Extend the CRI with user and ID mappings fields.
 - Implement support for `Host` and `Cluster` user namespace modes.

The goal of this phase is to implement some initial user namespace support
providing pod-to-host isolation and supporting volumes. The implementation of
the `Pod` mode is out of scope in this phase because it requires a non
negligible amount of work and we could risk losing the focus failing to deliver
this feature.

### Future Phases

These phases aim to implement the `Pod` mode. After these phases are completed
the full advantanges of user namespaces can be used in some cases (stateless
workloads).

There are some things that have to be studied with more detail for these
phase(s) but are not needed for phase 1, hence they are not discussed in detail:

- **Pod Default Mode**:
  It's not clear yet what should be the process to make this happen as this is a
  potentially non backwards compatible change. It's specially relevant for
  workloads not compatible with user namespaces. A [host defaulting
  mechanishm](#host-defaulting-mechanishm) can help to make this transition
  smoother.
- **Duplicated Container Images Snapshots and Garbage Collection**:
  The `Pod` mode makes it more difficult to handle the [duplicated
  snapshots](#duplicated-snapshots-of-container-images) issue as it's possible
  that a pod uses a unique ID mapping each time it's scheduled. The different
  runtimes will have to use solutions like `metacopy` option of overlayfs or new
  kernel features to overcome it. It's also likely that the kubelet image garbage collection
  algorithm has to be changed as image snapshots shold be deleted as soon as the
  container finishes.
- **ID Mappings Allocation Algorithm**
  The `Pod` mode requires to have each pod in different and non-overlapping ID
  mapping. It requires to implement an algorithm that performs that allocation.
  The default size of the range for each container is `65536`. This can be tuned up
  based on the feedback we receive.
  There are some open questions about it:
    - Should be the size of the range configurable by the operator?
    - How to get the ID mapping range of a running pod when kubelet crashes?
    - Can the user specify the ID mappings for a pod?
- **High IDs in Container Images**:
  The IDs present on the image are not available as image metadata. The runtimes
  would have to perform an image check, such as analysing the `/etc/passwd` file,
  to discover what those IDs are. The kubelet and the CRI will require some
  changes to make this information available to the ID mappings allocator
  algorithm. It would have to be sure that the allocated mappings include those
  IDs and should have some logic to protect against special crafted images to
  perform a kind of DOS allocating too many IDs for a given container.
- **Security Considerations**:
  Once `Pod` is the default mode, it is needed to control who can use `Host` and
  `Cluster` modes. This can be done through Pod Security Policies if they are
  available at the time of implementing this phase.

## Design Details

This section only focuses on phase 1 as specified above.

### Summary of the Proposed Changes

- Extend the CRI to have a user namespace mode and the user and group ID
  mappings.
- Add a `userNamespaceMode` field to the pod spec.
- Add the cluster-wide ID mappings to the kubelet configuration file.
- Add a `UserNamespacesSupport` feature flag to enable / disable the user
  namespaces support.
- Include 1-to-1 mapping for fsGroup.
- Update owner of ephemeral volumes populated by the kubelet.

### CRI API Changes

The CRI is extended to (optionally) specify the user namespace mode and the ID
mappings for a pod.
[`NamespaceOption`](https://github.com/kubernetes/cri-api/blob/1eae59a7c4dee45a900f54ea2502edff7e57fd68/pkg/apis/runtime/v1alpha2/api.proto#L228)
is extended with two new fields:
- A `user` `NamespaceMode` that defines if the pod should run in an independent
  user namespace (`POD`) or if it should share the host user namespace (`NODE`).
- The ID mappings to be used if the user namespace mode is `POD`.

```
// LinuxIDMapping represents a single user namespace ID mapping.
message LinuxIDMapping {
   // container_id is the starting ID for the mapping inside the container.
   uint32 container_id = 1;
   // host_id is the starting ID for the mapping on the host.
   uint32 host_id = 2;
   // size is the number of elements in the mapping.
   uint32 size = 3;
}

// LinuxIDMappings represents the user and group user namespace ID mappings.
message LinuxIDMappings {
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
    // IDs mappings to use if the user NamespaceMode is POD
    LinuxIDMappings id_mappings = 6
}
```

### Add userNamespaceMode Field

The `userNamespaceMode` field can be added in two different places. This
proposal presents the two possibilities to discuss with the community.

<<[UNRESOLVED where to put the userNamespaceMode field ]>>

#### Option 1: PodSpec

Add it to `v1.PodSpec` following the rationale that other fields (`host{Network,
IPC, PID}`) that control namespaces behaviour are defined in this place.

```
const (
	UserNamespaceModeHost    PodUserNamespaceMode = "Host"
	UserNamespaceModeCluster PodUserNamespaceMode = "Cluster"
)

type PodSpec struct {
...
  // UserNamespaceMode controls how user namespaces are used for this Pod.
  // Three modes are supported:
  // "Host": The pod shares the host user namespace. (default value).
  // "Cluster": The pod uses a cluster-wide configured ID mappings.
  // +k8s:conversion-gen=false
  // +optional
  UserNamespaceMode PodUserNamespaceMode `json:"userNamespaceMode,omitempty" protobuf:"bytes,36,opt  name=userNamespaceMode"`
...
```

#### Option 2: PodSecurityContext

Add it to `v1.PodSpec.PodSecurityContext` as this field controls a security
aspect of the pod.

```
const (
	UserNamespaceModeHost    PodUserNamespaceMode = "Host"
	UserNamespaceModeCluster PodUserNamespaceMode = "Cluster"
)

type PodSecurityContext struct {
...
  // UserNamespaceMode controls how user namespaces are used for this Pod.
  // Three modes are supported:
  // "Host": The pod shares the host user namespace. (default value).
  // "Cluster": The pod uses a cluster-wide configured ID mappings.
  // +optional
  UserNamespaceMode PodUserNamespaceMode `json:"userNamespaceMode,omitempty" protobuf:"bytes,11,opt  name=userNamespaceMode"`
...
```
<<[/UNRESOLVED]>>

### Configuring the Cluster ID Mappings

This proposal considers two different ways to configure the ID mappings used for
the `Cluster` mode. This is for discussion with the community and only one will
be considered.

<<[UNRESOLVED where to configure the cluster wide ID mappings ]>>

#### Option 1: Configure in Kubelet Configuration File

The ID mappings used for pods in `Cluster` mode are set in the kubelet
configuration file. The file allows to set different ID mappings for user and
group. Kubelet performs a check looking for configuration mistakes, like
overlapping mappings, to prevent kubelet sending wrong ID mappings to the
runtime.

The following is an example a configuration file:

```
apiVersion: kubelet.config.k8s.io/v1beta1
kind: KubeletConfiguration
clusterIDMappings:
  uidMappings:
  - containerID: 0
    hostID: 500000
    size: 65536
  - containerID: 100000
    hostID: 600000
    size: 65536
  gidMappings:
  - containerID: 0
    hostID: 500000
    size: 65536

```

**Pros**:
 - It's directly accessible by the kubelet. It's not needed to expose through an
   API and there are not concerns if the api server is down.
 - No modifications are needed to other components.

**Cons**:
 - It's more difficult for the user as they have to guarantee the same ID
   mapping is configured in all the nodes.

#### Option 2: Configure as a Cluster Parameter in kube-apiserver

This option considers setting this parameter on the kube-apiserver.

**Pros**:
 - Easier for the user as the parameter is configured once in a single place.

 **Cons**:
 - It's difficult to expose this parameter to the kubelet.
 - The parameter could not be available for the kubelet if the kube-apiserver is
   down.

<<[/UNRESOLVED]>>

### 1-to-1 Mapping for fsGroup

The `fsGroup` is assumed to be the correct group owner of the files present on
the volumes. A 1-to-1 ID mapping is added to ensure that the same GID used in
the container is the effective GID in the host when the fsGroup is defined. If
the `fsGroup` is part of the cluster ID mappings, a hole is "punched", otherwise
an extra one element ID mapping is added.

For instance, if the cluster GID mappings are

```
ContainerID HostID Size
0           1000   0
```
and there is a pod with `fsGroup: 500`, the GID mappings for that specific pod
are

```
ContainerID HostID Size
0           1000   500
501         1501   499
500         500    1
```

On the other hand, if a pod has `fsGroup: 3000`, the GID mappings for that pod are

```
ContainerID HostID Size
0           1000   1000
3000        3000   1
```

### Updating Ownership of Ephemeral Volumes

Ephemeral volumes use
[`AtomicWriter`](https://github.com/kinvolk/kubernetes/blob/master/pkg/volume/util/atomic_writer.go)
to create the files that are mounted to the containers. This component [has some
logic](https://github.com/kinvolk/kubernetes/blob/c94242a7b1d238cc27aea9b6d45ccb9963e814bb/pkg/volume/util/atomic_writer.go#L403)
to update the ownership of those files in some cases. It will be extended to
take the ID mappings into consideration when the pod runs in `Cluster` mode.

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

TBD

### Graduation Criteria

#### Alpha

- [ ] Support for `Cluster` and `Host` modes implemented.
- [ ] Support implemented in CRI-O.
- [ ] Support implemented in containerd.
- [ ] Unit test coverage.
- [ ] Support for `Pod` mode discused and implemented.

#### Beta

- [ ] Feedback from alpha is addressed.
- [ ] E2E test coverage.
- [ ] There are well-documented use cases of this feature.

#### GA

TDB

### Upgrade / Downgrade Strategy

### Version Skew Strategy

The container runtime will have to be updated in the nodes to support this
feature.

The new `user` field in the `NamespaceOption` will be ignored by an old runtime
without user namespaces support. The container will be placed in the host user
namespace. It's a responsibility of the user to guarantee that a runtime
supporting user namespaces is used when this feature is enabled.

An old version of kubelet (without user namespaces support) used with a new
container runtime (with user namespaces support) can cause some issues too. In
this case the runtime can wrongly infer that the `user` field is set to `POD` in
the `NamespaceOption` message. To avoid this problem the runtime should check if
the `mappings` field contains any mappings, an error should be raised otherwise.

## Production Readiness Review Questionnaire

### Feature Enablement and Rollback

* **How can this feature be enabled / disabled in a live cluster?**
  - [X] Feature gate
    - Feature gate name: UserNamespacesSupport
    - Components depending on the feature gate: kubelet

* **Does enabling the feature change any default behavior?**
  The default mode for usernamespaces is `Host`, so the default behaviour is not changed.

* **Can the feature be disabled once it has been enabled (i.e. can we roll back
  the enablement)?**
  Yes, the `UserNamespacesSupport` feature gate has to be disabled and pods
  running in `Cluster` and `Pod` mode have to be recreated. The effective user
  and group IDs of the processes would be different before and after disabling
  the feature for pods running in `Cluster` and `Pod` modes. This can cause
  access issues to pods accessing files saved in volumes.

* **What happens if we reenable the feature if it was previously rolled back?**
  The situation is very similar to the described above. The pod will be able to
  access the files written when the feature was enabled but can have issues to
  access those files written while the feature was disabled.

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
the existing API objects?** Yes. The PodSpec will be increased.

* **Will enabling / using this feature result in increasing time taken by any
operations covered by [existing SLIs/SLOs]?**
  Yes. The startup latency of both stateless and stateful pods is increased as
  the runtime has to set correct ownership for the container image before
  starting them.

* **Will enabling / using this feature result in non-negligible increase of
resource usage (CPU, RAM, disk, IO, ...) in any components?**: No.

### Troubleshooting

Will be added before transition to beta.

* **How does this feature react if the API server and/or etcd is unavailable?**

* **What are other known failure modes?**

* **What steps should be taken if SLOs are not being met to determine the problem?**

## Implementation History

- 2020-10-16: Initial proposal submitted.

## Drawbacks

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
that this decision has the drawback that pods that don't set the
`userNamespaceMode` will not have the security advantages of user namespaces,
however we consider it's more important to keep compatibility with previous
workloads.

### Host Defaulting Mechanishm

Previous proposals like [Node-Level UserNamespace
implementation](https://github.com/kubernetes/kubernetes/pull/64005) had a
mechanism to default to the host user namespace when the pod specification
includes features that could be not compatible with user namespaces (similar to
[Default host user namespace via experimental
flag](https://github.com/kubernetes/kubernetes/pull/31169)). This proposal
doesn't require a similar mechanishm given that the default mode is `Host`.

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
