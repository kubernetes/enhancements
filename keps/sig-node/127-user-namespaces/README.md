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
    - [Story 3](#story-3)
    - [Story 4](#story-4)
    - [Story 5](#story-5)
  - [Notes/Constraints/Caveats](#notesconstraintscaveats)
  - [Risks and Mitigations](#risks-and-mitigations)
- [Design Details](#design-details)
  - [Pod.spec changes](#podspec-changes)
  - [CRI changes](#cri-changes)
  - [Phases](#phases)
    - [Phase 1: pods &quot;without&quot; volumes](#phase-1-pods-without-volumes)
      - [pkg/volume changes for phase I](#pkgvolume-changes-for-phase-i)
    - [Phase 2: pods with volumes](#phase-2-pods-with-volumes)
    - [Phase 3: TBD](#phase-3-tbd)
    - [Unresolved](#unresolved)
  - [Summary of the Proposed Changes](#summary-of-the-proposed-changes)
  - [Test Plan](#test-plan)
      - [Prerequisite testing updates](#prerequisite-testing-updates)
      - [Unit tests](#unit-tests)
      - [Integration tests](#integration-tests)
      - [e2e tests](#e2e-tests)
      - [critests](#critests)
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

Items marked with (R) are required *prior to targeting to a milestone / release*.

- [X] (R) Enhancement issue in release milestone, which links to KEP dir in [kubernetes/enhancements] (not the initial KEP PR)
- [X] (R) KEP approvers have approved the KEP status as `implementable`
- [X] (R) Design details are appropriately documented
- [X] (R) Test plan is in place, giving consideration to SIG Architecture and SIG Testing input
- [X] (R) Graduation criteria is in place
- [X] (R) Production readiness review completed
- [X] Production readiness review approved
- [ ] "Implementation History" section is up-to-date for milestone
- [ ] User-facing documentation has been created in [kubernetes/website], for publication to [kubernetes.io]
- [ ] Supporting documentation—e.g., additional design documents, links to mailing list discussions/SIG meetings, relevant PRs/issues, release notes

[kubernetes.io]: https://kubernetes.io/
[kubernetes/enhancements]: https://git.k8s.io/enhancements
[kubernetes/kubernetes]: https://git.k8s.io/kubernetes
[kubernetes/website]: https://git.k8s.io/website

## Summary

This KEP adds support to use user-namespaces in pods.

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

The following security vulnerabilities are (completely or partially) mitigated
with user namespaces as we propose here and it is expected that it will mitigate
similar future vulnerabilities too.
- [CVE-2019-5736](https://nvd.nist.gov/vuln/detail/CVE-2019-5736): Host runc binary can be overwritten from container. Completely mitigated with userns.
  - Score: [8.6 (HIGH)](https://nvd.nist.gov/vuln/detail/CVE-2019-5736)
  - https://github.com/opencontainers/runc/commit/0a8e4117e7f715d5fbeef398405813ce8e88558b
- [Azurescape](https://unit42.paloaltonetworks.com/azure-container-instances/):
  Completely mitigated with userns, at least as it was found (needs CVE-2019-5736). This is the **first cross-account container takeover in the public cloud**.
- [CVE-2021-25741](https://github.com/kubernetes/kubernetes/issues/104980): Mitigated as root in the container is not root in the host
  - Score: [8.1 (HIGH) / 8.8 (HIGH)](https://nvd.nist.gov/vuln/detail/CVE-2021-25741)
- [CVE-2017-1002101](https://github.com/kubernetes/kubernetes/issues/60813): mitigated, idem
  - Score: [9.6 (CRITICAL) / 8.8 (HIGH)](https://nvd.nist.gov/vuln/detail/CVE-2017-1002101)
- [CVE-2021-30465](https://github.com/opencontainers/runc/security/advisories/GHSA-c3xm-pvg7-gh7r): mitigated, idem
  - Score: [8.5 (HIGH)](https://nvd.nist.gov/vuln/detail/CVE-2021-30465)
- [CVE-2016-8867](https://nvd.nist.gov/vuln/detail/CVE-2016-8867): Privilege escalation inside containers
  - Score: [7.5 (HIGH)](https://nvd.nist.gov/vuln/detail/CVE-2016-8867)
  - https://github.com/moby/moby/issues/27590
- [CVE-2018-15664](https://nvd.nist.gov/vuln/detail/CVE-2018-15664): TOCTOU race attack that allows to read/write files in the host
  - Score: [7.5 (HIGH)](https://nvd.nist.gov/vuln/detail/CVE-2018-15664)
  - https://github.com/moby/moby/pull/39252

### Goals

Here we use UIDs, but the same applies for GIDs.

- Increase node to pod isolation by mapping user and group IDs
  inside the container to different IDs in the host. In particular, mapping root
  inside the container to unprivileged user and group IDs in the node.
- Increase pod to pod isolation by allowing to use non-overlapping mappings
  (UIDs/GIDs) whenever possible. IOW, if two containers runs as user X, they run
  as different UIDs in the node and therefore are more isolated than today.
  See phase 3 for limitations.
- Allow pods to have capabilities (e.g. `CAP_SYS_ADMIN`) that are only valid in
  the pod (not valid in the host).
- Benefit from the security hardening that user namespaces provide against some
  of the future unknown runtime and kernel vulnerabilities.

### Non-Goals

- Provide a way to run the kubelet process or container runtimes as an
  unprivileged process. Although initiatives like [kubelet in user
  namespaces][kubelet-userns] and this KEP both make use of user namespaces, it is
  a different implementation for a different purpose.
- Implement all the very nice use cases that user namespaces allows. The goal
  here is to allow them as incremental improvements, not implement all the
  possible ideas related with user namespaces.

[kubelet-userns]: https://github.com/kubernetes/enhancements/tree/master/keps/sig-node/2033-kubelet-in-userns-aka-rootless

## Proposal

This KEP adds a new `hostUsers` field  to `pod.Spec` to allow to enable/disable
using user namespaces for pods.

This proposal aims to support running pods inside user namespaces. This will
improve the pod to node isolation (phase 1 and 2) and pod to pod isolation
(phase 3) we currently have.

This mitigates all the vulnerabilities listed in the motivation section.

### User Stories

#### Story 1

As a cluster admin, I want run some pods with privileged capabilities because
the applications in the pods require it (e.g. `CAP_SYS_ADMIN` to mount a FUSE
filesystem or `CAP_NET_ADMIN` to setup a VPN) but I don't want this extra
capability to grant me any extra privileges on the host or other pods.

#### Story 2

As a cluster admin, I want to allow some pods to run in the host user namespace
if they need a feature only available in such user namespace, such as loading a
kernel module with `CAP_SYS_MODULE`.

#### Story 3

As a cluster admin, I want to allow users to run their container as root
without that process having root privileges on the host, so I can mitigate the
impact of a compromised container.

#### Story 4

As a cluster admin, I want to allow users to choose the user/group they use
inside the container without it being the user/group they use on the node, so I
can mitigate the impact of a container breakout vulnerability (e.g to read
host files).

#### Story 5

As a cluster admin, I want to use different host UIDs/GIDs for pods running on
the same node (whenever kernel/kube features allow it), so I can mitigate the
impact a compromised pod can have on other pods and the node itself.

### Notes/Constraints/Caveats

### Risks and Mitigations

## Design Details

### Pod.spec changes

The following changes will be done to the pod.spec:

- `pod.spec.hostUsers`: `*bool`.
If true or not present, uses the host user namespace (as today)
If false, a new userns is created for the pod.
By default it is not set, which implies using the host user namespace.

### CRI changes

The following messages will be added:

```
// IDMapping describes host to container ID mappings for a pod sandbox.
message IDMapping {
    // host_id is the id on the host.
    uint32 host_id = 1;
    // container_id is the id in the container.
    uint32 container_id = 2;
    // length is the size of the range to map.
    uint32 length = 3;
}

// UserNamespace describes the intended user namespace configuration for a sandbox.
message UserNamespace {
    // NamespaceMode: `POD` or `NODE`
    NamespaceMode mode = 1;

    // uids specifies the UID mappings for the user namespace.
    repeated IDMapping uids = 2;

    // gids specifies the GID mappings for the user namespace.
    repeated IDMapping gids = 3;
}
```

The existing message `NamespaceOption` will have a `userns_options` field added.
The complete `NamespaceOption` message with the new field is shown here:

```
// NamespaceOption provides options for Linux namespaces.
message NamespaceOption {
    // Network namespace for this container/sandbox.
    // Note: There is currently no way to set CONTAINER scoped network in the Kubernetes API.
    // Namespaces currently set by the kubelet: POD, NODE
    NamespaceMode network = 1;
    // PID namespace for this container/sandbox.
    // Note: The CRI default is POD, but the v1.PodSpec default is CONTAINER.
    // The kubelet's runtime manager will set this to CONTAINER explicitly for v1 pods.
    // Namespaces currently set by the kubelet: POD, CONTAINER, NODE, TARGET
    NamespaceMode pid = 2;
    // IPC namespace for this container/sandbox.
    // Note: There is currently no way to set CONTAINER scoped IPC in the Kubernetes API.
    // Namespaces currently set by the kubelet: POD, NODE
    NamespaceMode ipc = 3;
    // Target Container ID for NamespaceMode of TARGET. This container must have been
    // previously created in the same pod. It is not possible to specify different targets
    // for each namespace.
    string target_id = 4;
    // User namespace for this sandbox.
    // The Kubelet picks the user namespace configuration to use for the sandbox.  The mappings
    // are specified as part of the UserNamespace struct.  If the struct is nil, then the POD mode
    // must be assumed.  This is done for backward compatibility with older Kubelet versions that
    // do not set a user namespace.
    UserNamespace userns_options = 5;
}

```

### Phases

We propose to divide the work in 3 phases. Each phase makes this work with
either more isolation or more workloads. When no support is yet added to handle
some workload, a clear error will be shown.

PLEASE note that only phase 1 is targeted for alpha.

Please note the last sub-section here is a table with the summary of the changes
proposed on each phase. That table is not updated (it is from the initial
proposal, doesn't have all the feedback and adjustments we discussed) but can
still be useful as a general overview.

TODO: move the table to markdown and include it here.


#### Phase 1: pods "without" volumes

This phase makes pods "without" volumes work with user namespaces. This is
activated via the bool `pod.spec.HostUsers` and can only be set to `false`
on pods which use either no volumes or only volumes of the following types:

 - configmap
 - secret
 - downwardAPI
 - emptyDir
 - projected

This list of volumes was chosen as they can't be used to share files with other
pods.

The mapping length will be 65536, mapping the range 0-65535 to the pod. This wide
range makes sure most workloads will work fine. Additionally, we don't need to
worry about fragmentation of IDs, as all pods will use the same length.

The mapping will be chosen by the kubelet, using a simple algorithm to give
different pods in this category ("without" volumes) a non-overlapping mapping.
Giving non-overlapping mappings generates the best isolation for pods.

Furthermore, the node UID space of 2^32 can hold up to 2^16 pods each with a
mapping length of 65536 (2^16) top. This imposes a limit of 65k pods per node,
but that is not an issue we will hit in practice for a long time, if ever (today
we run 110 pods per node by default).

Please note that a mapping is a bijection of host IDs to container IDs. The
settings `RunAsUser`, `RunAsGroup`, `runAsNonRoot` in
`pod.spec.containers.SecurityContext` will refer to the ID used inside the
container. Therefore, the host IDs the kubelet choses for the pod are not
important for the end user changing the pod spec.

The picked range will be stored under a file named `userns` in the pod folder
(by default it is usually located in `/var/lib/kubelet/pods/$POD/userns`). This
way, the Kubelet can read all the allocated mappings if it restarts.

During phase 1, to make sure we don't exhaust the host UID namespace, we will
limit the number of pods using user namespaces to `min(maxPods, 1024)`. This
leaves us plenty of host UID space free and this limits is probably never hit in
practice. See UNRESOLVED for more some UNRESOLVED info we still have on this.

##### pkg/volume changes for phase I

We need the files created by the kubelet for these volume types to be readable
by the user the pod is mapped to in the host. Luckily, Kuberentes already
supports specifying an [fsUser and fsGroup for volumes][volume-mounter-args] for
projected SA tokens so we just need to extend that to the other volume types we
support in phase I.

First, we need to teach the [operation executor][operation-executor] to convert
the container UIDs/GIDs to the host UIDs/GIDs that this pod is mapped to. To do
this, we will modify the [volumeHost interface][volumeHost-interface] adding
functions to transform a container ID to the corresponding host ID, so the
operation executor will just call that function (via
self.volumePluginMgr.Host.GetHostIDsForPod())

This function will call the kubelet, which will use the user namespace manager
created to transform the IDs for that pod. The [volumeHost
interface][volumeHost-interface] will then have this new method:

```
GetHostIDsForPod(pod *v1.Pod, containerUID, containerGID *int64) (hostUID, hostGID *int64, err error)
```

This method will only transform the IDs if the pod is running with userns
enabled. If userns is disabled, the params will not be translated and, also, 
GetHostIDsForPod() will return nil if the containerUID/containerGID received
were nil. This is to keep the current functionality untouched, as [fsUser and
fsGroup can be set to nil][operation-executor] today by the operation executor.

Secondly, we need to modify these volume types to honor the `fsUser` passed in
mounterArgs to their SetUpAt() method. As the [AtomicWriter already knows and
honors the FsUser field][atomic-writer-fsUser] already, just setting this field
is missing when creating the FileProjection (only done when the feature gate is
enabled). For example, configmaps will just need to set FsUser alongside Data
and Mode [here][configmap-fsuser] (and all the other paths in that function that
create the projection, of course). Other volume types are quite similar too.

For this we need to modify the MakePayload() or CollectData() functions of each
supported volume type to handle the fsUser new param, and the corresponding
volume SetUpAt() function to pass this new param when calling AtomicWriter.

We have written already this code and the per-volume diff is 3 lines.

For fsGroup() we don't need any changes, these volume types already honors it.

[volume-mounter-args]: https://github.com/kubernetes/kubernetes/blob/cfd69463deeebfc5ae8a0813d7d2b125033c4aca/pkg/volume/volume.go#L125-L129
[operation-executor]: https://github.com/kubernetes/kubernetes/blob/cfd69463deeebfc5ae8a0813d7d2b125033c4aca/pkg/volume/util/operationexecutor/operation_generator.go#L674-L675
[volumeHost interface]: https://github.com/kubernetes/kubernetes/blob/cfd69463deeebfc5ae8a0813d7d2b125033c4aca/pkg/volume/plugins.go#L360-L361
[atomic-writer-fsUser]: https://github.com/kubernetes/kubernetes/blob/cfd69463deeebfc5ae8a0813d7d2b125033c4aca/pkg/volume/util/atomic_writer.go#L68
[configmap-fsuser]: https://github.com/kubernetes/kubernetes/blob/cfd69463deeebfc5ae8a0813d7d2b125033c4aca/pkg/volume/configmap/configmap.go#L272-L273

#### Phase 2: pods with volumes

This phase makes user namespaces work in pods with volumes too. This is
activated via the bool `pod.spec.HostUsers` too and pods fall into this mode if
some other volume type than the ones listed for phase 1 is used. IOW, when phase
2 is implemented, pods that use volume types not supported in phase 1, fall into
the phase 2.

All pods in this mode will use _the same_ mapping, chosen by the kubelet, with a
length 65536, and mapping the range 0-65535 too. IOW, each pod will have its own user
namespace, but they will map to _the same_ UIDs/GIDs in the host.

Using the same mapping allows for pods to share files and mitigates all the
listed vulnerabilities (as the host is protected from the container). It is also
a trivial next-step to take, given that we have phase 1 implemented: just return
the same mapping if the pod has other volumes.

While these pods do not use a distinct user namespace mapping, they are still
using a new user namespace object in the kernel (so they cannot join/attack
other pods namespaces). Security-wise this is a middle layer between what we
have today (no userns at all) and using a distinct UID/GID mapping for the user
namespace.

#### Phase 3: TBD

#### Unresolved

Here is a list of considerations raised in PRs discussion that hasn't yet
settle. This list is not exhaustive, we are just trying to put the things that
we don't want to forget or want to highlight it. Some things that are obvious we
need to tackle are not listed. Let us know if you think it is important to add
something else to this list:

- We will start with mappings of 64K. Tim Hockin, however, has expressed
  concerns. See more info on [this Github discussion](https://github.com/kubernetes/enhancements/pull/3065#discussion_r781676224)
  SergeyKanzhelev [suggested a nice alternative](https://github.com/kubernetes/enhancements/pull/3065#discussion_r807408134),
  to limit the number of pods so we guarantee enough spare host UIDs in case we
  need them for the future. There is no final decision yet on how to handle this.
  For now we will limit the number of pods, so the wide mapping is not
  problematic, but [there are downsides to this too](https://github.com/kubernetes/enhancements/pull/3065#discussion_r812806223)


- Tim suggested we should try to not use the whole UID space. Something to keep
  in mind for next revisions. More info [here](https://github.com/kubernetes/enhancements/pull/3065#discussion_r797100081)

Idem before, see Sergey idea from previous item.

- While we said that if support is not implemented we will throw a clear error,
  we have not said if it will be API-rejected at admission time or what. I'd
  like to see how that code looks like before promising something. Was raised [here](https://github.com/kubernetes/enhancements/pull/3065#discussion_r798730922)

- Tim suggested that we might want to allow the container runtimes to choose
  the mapping and have different runtimes pick different mappings. While KEP
  authors disagree on this, we still need to discuss it and settle on something.
  This was [raised here](https://github.com/kubernetes/enhancements/pull/3065#discussion_r798760382)

- (will clarify later with Tim what he really means here). Tim [raised this](https://github.com/kubernetes/enhancements/pull/3065#discussion_r798766024): do
  we actually want to ship phase 2 as automatic thing? If we do, any workloads
  that use it can't ever switch to whatever we do in phase 3

- What about windows or VM container runtimes, that don't use linux namespaces?
  We need a review from windows maintainers once we have a more clear proposal.
  We can then adjust the needed details, we don't expect the changes (if any) to be big.
  IOW, in my head this looks like this: we merge this KEP in provisional state if
  we agree on the high level idea, with @giuseppe we do a PoC so we can fill-in
  more details to the KEP (like CRI changes, changes to container runtimes, how to
  configure kubelet ranges, etc.), and then the Windows folks can review and we
  adjust as needed (I doubt it will be a big change, if any). After that we switch
  the KEP to implementable (or if there are long delays to get a review, we might
  decide to do it after the first alpha, as the community prefers and time
  allows). Same applies for VM runtimes.
  UPDATE: Windows maintainers reviewed and [this change looks good to them][windows-review].

[windows-review]: https://github.com/kubernetes/enhancements/pull/3275#issuecomment-1121603308

### Summary of the Proposed Changes

[This table](https://docs.google.com/presentation/d/1z4oiZ7v4DjWpZQI2kbFbI8Q6botFaA07KJYaKA-vZpg/edit#slide=id.gfd10976c8b_1_41) gives you a quick overview of each phase (note it is outdated, but still useful for a general overview).

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

Most of the changes will be in a new file we will create, the userns manager.
Tests are already written for that file.

The rest of the changes are more about hooking it up so it is called.

- `pkg/kubelet/kuberuntime/kuberuntime_container_linux.go`: 24.05.2022 - 90.8%
- `pkg/volume/util/operationexecutor/operation_generator.go`: 24.05.2022 - 8.6%
- `pkg/volume/configmap/configmap.go`: 24.05.2022 - 74.8%
- `pkg/volume/downwardapi/downwardapi.go`: 24.05.2022 - 61.7%
- `pkg/volume/secret/secret.go`: 24.05.2022 - 65.7%
- `pkg/volume/projected/projected.go`: 24.05.2022 - 68.2%

##### Integration tests

<!--
This question should be filled when targeting a release.
For Alpha, describe what tests will be added to ensure proper quality of the enhancement.

For Beta and GA, add links to added tests together with links to k8s-triage for those tests:
https://storage.googleapis.com/k8s-triage/index.html
-->

We will add a test that:

1. Starts the kubelet with the feature gate enabled
1. Creates a pod with user namespaces
1. Restarts the kubelet with the feature gate disabled
1. Deletes the pod

We will check the pod is deleted correctly and the pod folder is indeed deleted.

Same note about container runtime version from the e2e section applies.

- <test>: <link to test coverage>

##### e2e tests

<!--
This question should be filled when targeting a release.
For Alpha, describe what tests will be added to ensure proper quality of the enhancement.

For Beta and GA, add links to added tests together with links to k8s-triage for those tests:
https://storage.googleapis.com/k8s-triage/index.html

We expect no non-infra related flakes in the last month as a GA graduation criteria.
-->

We can do e2e test to verify that userns is disabled when it should.

To test with userns enabled, we need to patch container runtimes. We can either
try to use a patched version or make the alpha longer and add e2e when we can
use container runtime versions that have the needed changes.

##### critests

- For Alpha, the feature is tested for containerd and CRI-O in cri-tools repo using critest to
  make sure the specified user namespace configuration is honored.

- <test>: <link to test coverage>

### Graduation Criteria

Note this section is a WIP yet.

##### Alpha
- Phase 1 implemented

##### Beta

- Make plans on whether, when, and how to enable by default
- Should we reconsider making the mappings smaller by default?
- Should we allow any way for users to for "more" IDs mapped? If yes, how many more and how?
- Should we allow the user to ask for specific mappings?

##### GA

### Upgrade / Downgrade Strategy

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

Some definitions first:
- New kubelet: kubelet with CRI proto files that includes the changes proposed in
this KEP.

- Old kubelet: idem, but CRI proto files don't include this changes.

- New runtime: container runtime with CRI proto files that includes the changes
proposed in this KEP.

- Old runtime: idem, but CRI proto files don't include this changes.

New runtime and old kubelet: containers are created without userns. Kubelet
doesn't request userns (doesn't have that feature) and therefore the runtime
doesn't create them. The runtime can detect this situation as the `user` field
in the `NamespaceOption` will be seen as nil, [thanks to
protobuf][proto3-defaults]. We already tested this with real code.

Old runtime and new kubelet: containers are created without userns. As the
`user` field of the `NamespaceOption` message is not part of the runtime
protofiles, that part is ignored by the runtime and pods are created using the
host userns.


[proto3-defaults]: https://developers.google.com/protocol-buffers/docs/proto3#default

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

- [x] Feature gate (also fill in values in `kep.yaml`)
  - Feature gate name: UserNamespacesSupport
  - Components depending on the feature gate: kubelet, kube-apiserver

###### Does enabling the feature change any default behavior?

No.

<!--
Any change of default behavior may be surprising to users or break existing
automations, so be extremely careful here.
-->

###### Can the feature be disabled once it has been enabled (i.e. can we roll back the enablement)?

Yes. Pods that were created and opted-in to use user namespaces, will have to be
recreated to run again without user namespaces.

Other than that, it is safe to disable once it was enabled.

It is also safe to:

1. Enable the feature gate
1. Create a pod with user namespaces,
1. Disable the feature gate in the kubelet (and restart it)
1. Delete the pod

We won't leak files nor other resources on that scenario either.

<!--
Describe the consequences on existing workloads (e.g., if this is a runtime
feature, can it break the existing applications?).

Feature gates are typically disabled by setting the flag to `false` and
restarting the component. No other changes should be necessary to disable the
feature.

NOTE: Also set `disable-supported` to `true` or `false` in `kep.yaml`.
-->

###### What happens if we reenable the feature if it was previously rolled back?

Pods will have to be re-created to use the feature.

###### Are there any tests for feature enablement/disablement?

We will add.

We will test for when the field pod.spec.HostUsers is set to true, false
and not set. All of this with and without the feature gate enabled.

We will also unit test that, if pods were created with the new field
pod.specHostUsers, then if the featuregate is disabled all works as expected (no
user namespace is used).

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

No.

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

No.

<!--
Describe them, providing:
  - API type
  - Supported number of objects per cluster
  - Supported number of objects per namespace (for namespace-scoped objects)
-->

###### Will enabling / using this feature result in any new calls to the cloud provider?

No.
<!--
Describe them, providing:
  - Which API(s):
  - Estimated increase:
-->

###### Will enabling / using this feature result in increasing size or count of the existing API objects?

Yes. The pod.Spec.HostUsers field is a bool, should be small.

<!--
Describe them, providing:
  - API type(s):
  - Estimated increase in size: (e.g., new annotation of size 32B)
  - Estimated amount of new objects: (e.g., new Object X for every existing Pod)
-->

###### Will enabling / using this feature result in increasing time taken by any operations covered by existing SLIs/SLOs?

Not in any Kubernetes component, it might take more time for the container
runtime to start the pod _if they support old kernels_.

With the info below, we decided that container runtimes _can_ support old
kernels, but the SLO will be excluded in that case.

The SLO that might be affected is:

> Startup latency of schedulable stateless pods, excluding time to pull images and run init containers, measured from pod creation timestamp to when all its containers are reported as started and observed via watch, measured as 99th percentile over last 5 minutes

The rootfs needs to be accessible by the user in the user namespace the pod is.
As every pod might run as a different user, we only know the mapping for a pod
when it is created.

If new kernels are used (5.19+), the container runtime can use idmapped mounts
in the rootfs and the UID/GID shifting needed so the pod can access the rootfs
takes a few ms (it is just a bind mount).

If the container runtimes want to support old kernels too, they will need to
chown the rootfs to the new UIDs/GIDs. This can be slow, although the metacopy
overlayfs parameter helps so it is not so bad.

The options we have for this plumbing to setup the rootfs:
 * Consider it the same way we consider the image pull, and explicitelly exclude
   it from the SLO.
 * Tell runtimes to not support old kernels (KEP authors are working in the
   containerd and CRI-O implementation too, we can easily do this)
 * Tell runtimes to support old kernels if they want, but with a big fat warning
   in the docs about the implications

In any case, the kubernetes components do not need any change.

<!--
Look at the [existing SLIs/SLOs].

Think about adding additional work or introducing new steps in between
(e.g. need to do X to start a container), etc. Please describe the details.

[existing SLIs/SLOs]: https://git.k8s.io/community/sig-scalability/slos/slos.md#kubernetes-slisslos
-->

###### Will enabling / using this feature result in non-negligible increase of resource usage (CPU, RAM, disk, IO, ...) in any components?

Not in any kubernetes component.

The container runtime might use more disk, if the runtime supports this feature
in old kernels, as that means the runtime needs to chown the rootfs (see
previous question for more details).

This is not needed on newer kernels, as they can rely on idmapped mounts for the
UID/GID shifting (it is just a bind mount).

<!--
Things to keep in mind include: additional in-memory state, additional
non-trivial computations, excessive access to disks (including increased log
volume), significant amount of data sent and/or received over network, etc.
This through this both in small and large cases, again with respect to the
[supported limits].

[supported limits]: https://git.k8s.io/community//sig-scalability/configs-and-limits/thresholds.md
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

<!--
What other approaches did you consider, and why did you rule them out? These do
not need to be as detailed as the proposal, but should include enough
information to express the idea and why it was not acceptable.
-->

## Infrastructure Needed (Optional)

<!--
Use this section if you need things from the project/SIG. Examples include a
new subproject, repos requested, or GitHub details. Listing these here allows a
SIG to get the process for these resources started right away.
-->
