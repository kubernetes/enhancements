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
  - [Support for pods](#support-for-pods)
    - [Configuration of ranges](#configuration-of-ranges)
  - [Handling of volumes](#handling-of-volumes)
    - [Example of how idmap mounts work](#example-of-how-idmap-mounts-work)
      - [Example without idmap mounts](#example-without-idmap-mounts)
      - [Example with idmap mounts](#example-with-idmap-mounts)
    - [Regarding the previous implementation for volumes](#regarding-the-previous-implementation-for-volumes)
  - [Pod Security Standards (PSS) integration](#pod-security-standards-pss-integration)
  - [Unresolved](#unresolved)
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
    - [Kubelet and Kube-apiserver skew](#kubelet-and-kube-apiserver-skew)
    - [Kubelet and container runtime skews](#kubelet-and-container-runtime-skews)
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
  - [Don't use idmap mounts and rely chown all the files correctly](#dont-use-idmap-mounts-and-rely-chown-all-the-files-correctly)
  - [64k mappings?](#64k-mappings)
  - [Allow runtimes to pick the mapping](#allow-runtimes-to-pick-the-mapping)
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
- [X] "Implementation History" section is up-to-date for milestone
- [ ] User-facing documentation has been created in [kubernetes/website], for publication to [kubernetes.io]
- [ ] Supporting documentation—e.g., additional design documents, links to mailing list discussions/SIG meetings, relevant PRs/issues, release notes

[kubernetes.io]: https://kubernetes.io/
[kubernetes/enhancements]: https://git.k8s.io/enhancements
[kubernetes/kubernetes]: https://git.k8s.io/kubernetes
[kubernetes/website]: https://git.k8s.io/website

## Summary

This KEP adds support to use user-namespaces.

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
  (UIDs/GIDs) whenever possible. In other words: if two containers runs as user
  X, they run as different UIDs in the node and therefore are more isolated than today.
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
using user namespaces.

This proposal aims to support running pods inside user namespaces.

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

The Mount message will be modified to take the UID/GID mappings. The complete
Mount is shown here:

```
// Mount specifies a host volume to mount into a container.
message Mount {
    // Path of the mount within the container.
    string container_path = 1;
    // Path of the mount on the host. If the hostPath doesn't exist, then runtimes
    // should report error. If the hostpath is a symbolic link, runtimes should
    // follow the symlink and mount the real destination to container.
    string host_path = 2;
    // If set, the mount is read-only.
    bool readonly = 3;
    // If set, the mount needs SELinux relabeling.
    bool selinux_relabel = 4;
    // Requested propagation mode.
    MountPropagation propagation = 5;
    // uid_mappings specifies the UID mappings for this mount.
    repeated IDMapping uid_mappings = 6;
    // gid_mappings specifies the GID mappings for this mount.
    repeated IDMapping gid_mappings = 7;
}
```

The CRI runtime reports what runtime handlers have support for user
namespaces through the `StatusResponse` message, that gains a new
field `runtime_handlers`:

```
message StatusResponse {
    // Status of the Runtime.
    RuntimeStatus status = 1;
    // Info is extra information of the Runtime. The key could be arbitrary string, and
    // value should be in json format. The information could include anything useful for
    // debug, e.g. plugins used by the container runtime.
    // It should only be returned non-empty when Verbose is true.
    map<string, string> info = 2;

    // Runtime handlers.
    repeated RuntimeHandler runtime_handlers = 3;
}
```

Where RuntimeHandler is defined as below:

```
message RuntimeHandlerFeatures {
    // supports_user_namespaces is set to true if the runtime handler supports
    // user namespaces.
    bool supports_user_namespaces = 1;
}

message RuntimeHandler {
    // Name must be unique in StatusResponse.
    // An empty string denotes the default handler.
    string name = 1;
    // Supported features.
    RuntimeHandlerFeatures features = 2;
}
```

### Support for pods

Make pods work with user namespaces. This is activated via the
bool `pod.spec.hostUsers`.

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

#### Configuration of ranges

If no configuration is present, the kubelet will use a sane default:

 * `0-65535`: reserved for the host processes and files
 * `65536-<65536 * maxPods>`: reserved for pods

The kubelet will detect if a configuration is present by using the `getsubids` binary. It will query
for ranges allocated to the "kubelet" user.

If the kubelet user doesn't exist or `getsubids` is not installed, the kubelet will use the sane
default aforementioned. On any other cases, it will return an error.

By using `getsubids` we make sure the kubelet interacts fine with other programs in the host that
also allocate ranges for user namespaces (be it /etc/subuid or centralized systems as freeIPA).

### Handling of volumes

When the volumes used are supported, the kubelet will set the `uid_mappings` and
`gid_mappings` in the CRI `Mount message`. It will use the same mappings the
kubelet allocated for the user namespace (see previous section for more details
on this).

The container runtime, when it receives the `Mount message` with the UID/GID
mappings fields set, it uses the Linux idmap mounts support to create such a
mount. If the kernel/filesystem used doesn't support idmap mounts, a clear error
will be thrown by the OCI runtime at pod creation.

#### Example of how idmap mounts work

Let's say we have a pod A that has this mapping for its user namespace
allocated (same mapping for UIDs than for GIDs):

| Host ID  | Container ID   | Length  |
|----------|----------------|---------|
| 65536    |      0         |  65536  |

This means user 0 inside the container is mapped to user 65536 on the host, user
1 is mapped to 65537 on the host, and so forth. Let' say the pod is running UID
0 inside the container too.

Let's say also that this pod has a configmap with a file named `foo`.
The file `/var/lib/kubelet/.../volumes/configmap/foo` is created by the kubelet,
the owner is host UID 0, and bind-mounted into the container at
`/vol/configmap/foo`.

##### Example without idmap mounts

To understand how idmap mounts work, let's first see an example of what happens
without idmap mounts.

If the pod wants to read who is the owner of file `/vol/configmap/foo`, as host
UID 0 is not mapped in this userns, it will be shown as nobody. Therefore,
process running in the pod's userns won't be able to read the file (unless it
has permissions for others).

Therefore, if we want the pod to be able to read those files and permissions be
what we expect, the kubelet needs to chown the file to a UID that is mapped into
the pod's userns (e.g. UID 65536 in this example), as that is what root inside
the container is mapped to in the user namespace.

We tried this before, but several limitations were hit. See the
[alternatives section](#dont-use-idmap-mounts-and-rely-chown-all-the-files-correctly)
for more details on the limitations we hit.

##### Example with idmap mounts

Now let's say the pod is using its volumes that were mounted using idmap mounts
by the container runtime. All the mappings used (the idmap mounts and the pod
userns) are:

| Host ID  | Container ID   | Length  |
|----------|----------------|---------|
| 65536    |      0         |  65536  |


Keep in mind the file `/var/lib/kubelet/.../volumes/configmap/foo` is created by
the kubelet, the owner is host UID 0, **but** it is bind-mounted into
`/vol/configmap/foo` **using an idmap mount** with the mapping aforementioned.

To achieve this, the kubelet only needs to set the UID/GID mappings in the
`Mount` CRI message.

If the pod wants to read who is the owner of file `/vol/configmap/foo`, now it
will see the owner is root inside the container. This is due to the IDs
transformations that the idmap mount does for us.

In other words: we can make sure the pod can read files instead of chowning them
all using the host IDs the pod is mapped to, by just using an idmap mount that
has the same mapping that we use for the pod user namespace.

#### Regarding the previous implementation for volumes
We previously added to the [KubeletVolumeHost interface][kubeletVolumeHost-interface]
the following method:

```
GetHostIDsForPod(pod *v1.Pod, containerUID, containerGID *int64) (hostUID, hostGID *int64, err error)
```

As this is not needed anymore, it will be deleted from the interface and all
components that implement the interface.

[kubeletVolumeHost-interface]: https://github.com/kubernetes/kubernetes/blob/36450ee422d57d53a3edaf960f86b356578fe996/pkg/volume/plugins.go#L322

### Pod Security Standards (PSS) integration

[Pod Security Standards](https://k8s.io/docs/concepts/security/pod-security-standards)
define three different policies to broadly cover the whole security spectrum of
Kubernetes, while the User Namespaces feature should integrate into them. This
will happen only if the feature is graduated to GA, which _may_ result in
changing the `Restricted` profile to disallow host user namespaces for stateless
Pods.

The Pod Security will relax in a controlled way for pods which enable user
namespaces. This behavior can be controlled by an API Server Feature Gate, which
allows an early opt-in for end users. The overall burden to ensure that all
nodes will honor user namespaces is on the cluster admin, though. The relaxation
in detail means, that if user namespaces are enabled, then the following fields
won't be restricted any more because they always have to refer to the user
inside the container:

- `spec.securityContext.runAsNonRoot`
- `spec.containers[*].securityContext.runAsNonRoot`
- `spec.initContainers[*].securityContext.runAsNonRoot`
- `spec.ephemeralContainers[*].securityContext.runAsNonRoot`
- `spec.securityContext.runAsUser`
- `spec.containers[*].securityContext.runAsUser`
- `spec.initContainers[*].securityContext.runAsUser`
- `spec.ephemeralContainers[*].securityContext.runAsUser`
- `spec.containers[*].securityContext.allowPrivilegeEscalation`
- `spec.initContainers[*].securityContext.allowPrivilegeEscalation`
- `spec.ephemeralContainers[*].securityContext.allowPrivilegeEscalation`
- `spec.containers[*].securityContext.capabilities.drop`
- `spec.initContainers[*].securityContext.capabilities.drop`
- `spec.ephemeralContainers[*].securityContext.capabilities.drop`
- `spec.containers[*].securityContext.capabilities.add`
- `spec.initContainers[*].securityContext.capabilities.add`
- `spec.ephemeralContainers[*].securityContext.capabilities.add`

A serial test will be added to validate the functionality with the enabled
feature gate.

### Unresolved

Here is a list of considerations raised in PRs discussion that hasn't yet
settle. This list is not exhaustive, we are just trying to put the things that
we don't want to forget or want to highlight it. Some things that are obvious we
need to tackle are not listed. Let us know if you think it is important to add
something else to this list:

- What about windows or VM container runtimes, that don't use linux namespaces?
  We need a review from windows maintainers once we have a more clear proposal.
  We can then adjust the needed details, we don't expect the changes (if any) to be big.
  In my head this looks like this: we merge this KEP in provisional state if
  we agree on the high level idea, with @giuseppe we do a PoC so we can fill-in
  more details to the KEP (like CRI changes, changes to container runtimes, how to
  configure kubelet ranges, etc.), and then the Windows folks can review and we
  adjust as needed (I doubt it will be a big change, if any). After that we switch
  the KEP to implementable (or if there are long delays to get a review, we might
  decide to do it after the first alpha, as the community prefers and time
  allows). Same applies for VM runtimes.
  UPDATE: Windows maintainers reviewed and [this change looks good to them][windows-review].


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

- For Beta, the feature is tested for containerd and CRI-O in cri-tools repo using critest to
  make sure the specified user namespace configuration is honored.

- <test>: <link to test coverage>

### Graduation Criteria

##### Alpha

- Support with idmap mounts
- Gather and address feedback from the community
- Add API Server feature flag to integrate into [Pod Security Standards (PSS)](#pod-security-standards-pss-integration)
- Changing restrictions on the what volumes will be allowed

##### Beta

- Gather and address feedback from the community
- Be able to configure UID/GID ranges to use for pods
- Add unit tests that exercise the feature gate switch (see section "Are there
  any tests for feature enablement/disablement?")
- Add cri-tools test
- This feature is not supported on Windows.
- Get review from VM container runtimes maintainers (not blocker, as VM runtimes should just ignore
  the field, but nice to have)

##### GA

- Gather and address feedback from the community
- Fully integrate into [Pod Security Standards (PSS)](#pod-security-standards-pss-integration)

### Upgrade / Downgrade Strategy

Existing pods will still work as intended, as the new field is missing there.

Upgrade will not change any current behaviors.

When the new functionality wasn't yet used, downgrade will not be affected.

On downgrade, when the functionality was used, the pods created with
user namespaces that are running will continue to run with user
namespaces. Pods will need to be re-created to stop using the user
namespace.

Versions of Kubernetes that doesn't have this feature implemented will
ignore the new field `pod.spec.hostUsers`.

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

#### Kubelet and Kube-apiserver skew

The apiserver and kubelet feature gate enablement work fine in any combination:

1. If the apiserver has the feature gate enabled and the kubelet doesn't, then the pod will show
   that field and the kubelet will reject it (see more details about how it is rejected on section
   "What specific metrics should inform a rollback?").
2. If the apiserver has the feature gate disabled and the kubelet enabled, the pod won't show this
   field and therefore the kubelet won't act on a field that isn't shown. The pod is created without
   user namespaces.

The kubelet can still create pods with user namespaces if static-pods are configured with
pod.spec.hostUsers and has the feature gate enabled.

If the kube-apiserver doesn't support the feature at all (< 1.25), the unknown field will be dropped and
the pod will be created without a userns.

If the kubelet doesn't support the feature (< 1.25), it will ignore the pod.spec.hostUsers field.

#### Kubelet and container runtime skews

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

Old runtime and new kubelet: the runtime won't report that it supports
user namespaces through the `StatusResponse` message, so the kubelet
will detect it and return an error if a pod with user namespaces is
created.

We added unit tests for the feature gate disabled, and integration
tests for the feature gate enabled and disabled.

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

We will test for when the field pod.spec.hostUsers is set to true, false
and not set. All of this with and without the feature gate enabled.

We will also unit test that, if pods were created with the new field
pod.specHostUsers, then if the featuregate is disabled all works as expected (no
user namespace is used).

We will add tests exercising the `switch` of feature gate itself (what happens
if I disable a feature gate after having objects written with the new field)

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

If one APIserver is upgraded while other's aren't and you are talking to a not
upgraded one, the pod will be accepted (if the apiserver is >= 1.25, rejected if
< 1.25).

If it is scheduled to a node where the kubelet has the feature flag activated
and the node meets the requirements to use user namespaces, then the pod will be
created with the namespace. If it is scheduled to a node that has the feature
disabled, it will be rejected (see more details about how it is rejected on
section "What specific metrics should inform a rollback?").

On a rollback, pods created while the feature was active (created with user
namespaces) will have to be re-created to run without user namespaces. If those
weren't recreated, they will continue to run in a user namespace.

<!--
Try to be as paranoid as possible - e.g., what if some components will restart
mid-rollout?

Be sure to consider highly-available clusters, where, for example,
feature flags will be enabled on some API servers and not others during the
rollout. Similarly, consider large clusters and how enablement/disablement
will rollout across nodes.
-->

###### What specific metrics should inform a rollback?

On Kubernetes side, the kubelet should start correctly.

On the node runtime side, a pod created with pod.spec.hostUsers=false should be on RUNNING state if
all node requirements are met. If the CRI runtime or the handler do not support the feature, the
kubelet returns an error.

When a pod hits this error returned by the kubelet, the status in `kubectl` is shown as
`ContainerCreating` and the pod events shows:

```
  Warning  FailedCreatePodSandBox  12s (x23 over 5m6s)  kubelet            Failed to create pod sandbox: user namespaces is not supported by the runtime
```

The following kubelet metrics are useful to check:
 - `kubelet_running_pods`: Shows the actual number of pods running
 - `kubelet_desired_pods`: The number of pods the kubelet is _trying_ to run

If these metrics are very different, it means there are desired pods that can't be set to running.
If that is the case, checking the pod events to see if they are failing for user namespaces reasons
(like the errors shown in this KEP) is advised, in which case it is recommended to rollback or
disable the feature gate.

<!--
What signals should users be paying attention to when the feature is young
that might indicate a serious problem?
-->

###### Were upgrade and rollback tested? Was the upgrade->downgrade->upgrade path tested?

Yes, we tested it locally using `./hack/local-up-cluster.sh`.

We tested enabling the feature flag, created a deployment with pod.spec.hostUsers=false, and then disabled
the feature flag and restarted the kubelet and kube-apiserver.

After that, we deleted the deployment pods (not the deployment object), the pods were re-created
without user namespaces just fine, without any modification needed on the deployment yaml.

We then enabled the feature flag on the kubelet and kube-apiserver, and deleted the deployment pod.
This re-created caused the pod to be re-created, this time with user namespaces enabled again.

To validate it, it is necessary to exec into a container in the pod and run the command `cat /proc/self/uid_map`.
When running in a user namespace the output is different than `0 0 4294967295` as it happens when running without
a user namespace.

<!--
Describe manual testing that was done and the outcomes.
Longer term, we may want to require automated upgrade/rollback tests, but we
are missing a bunch of machinery and tooling and can't do that now.
-->

###### Is the rollout accompanied by any deprecations and/or removals of features, APIs, fields of API types, flags, etc.?

No.
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

Check if any pod has the pod.spec.hostUsers field set to false.
<!--
Ideally, this should be a metric. Operations against the Kubernetes API (e.g.,
checking if there are objects with field X set) may be a last resort. Avoid
logs or events for this purpose.
-->

###### How can someone using this feature know that it is working for their instance?

If the runtime doesn't support user namespaces an error is returned by the kubelet and the pod cannot be created.

There are step-by-step examples in the Kubernetes documentation too: https://kubernetes.io/docs/tasks/configure-pod-container/user-namespaces/

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
- [x] Other (treat as last resort)
  - Details: check pods with pod.spec.hostUsers field set to false, and see if they are in RUNNING
    state. Exec into a container and run `cat /proc/self/uid_map` to verify that the mappings are different
    than the mappings on the host.

###### What are the reasonable SLOs (Service Level Objectives) for the enhancement?

If a node meets all the requirements, there should be no change to existing SLO/SLIs.

If a container runtime wants to support old kernels, it can have a performance impact, though. For
more details, see the question:
    "Will enabling / using this feature result in increasing time taken by any operations covered by existing SLIs/SLOs?"

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

No new SLI needed for this feature.
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

No.

This feature is using yet another namespace when creating a pod. If the pod creation fails (by
an error on the kubelet or returned by the container runtime), a clear error is returned to the
user. The feedback on this is very direct to the user actions.

A metric like "errors returned in pods with user namespaces enabled" can be very noisy, as the error
can be completely unrelated (image pull secret errors, configmap referenced and not defined, any
other container runtime error, etc.). We can't see any metric that can be helpful, as the user has a
very direct feedback already.

<!--
Describe the metrics themselves and the reasons why they weren't added (e.g., cost,
implementation difficulties, etc.).
-->

### Dependencies

<!--
This section must be completed when targeting beta to a release.
-->

###### Does this feature depend on any specific services running in the cluster?

Yes:

- [CRI version]
  - Usage description: CRI changes done in k8s 1.27 are needed
    - Impact of its outage on the feature: minimal, feature will be ignored by runtimes using an
      older version (e.g. containerd 1.6) or will return an error (e.g containerd 1.7).
    - Impact of its degraded performance or high-error rates on the feature: N/A.

- [Linux kernel]
  - Usage description: Linux 6.3 or higher
    - Impact of its outage on the feature: pod creation with hostUsers=false will return an error.
    - Impact of its degraded performance or high-error rates on the feature: N/A.

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

> Startup latency of schedulable pods, excluding time to pull images and run init containers, measured from pod creation timestamp to when all its containers are reported as started and observed via watch, measured as 99th percentile over last 5 minutes

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

###### Can enabling / using this feature result in resource exhaustion of some node resources (PIDs, sockets, inodes, etc.)?

<!--
Focus not just on happy cases, but primarily on more pathological cases
(e.g. probes taking a minute instead of milliseconds, failed pods consuming resources, etc.).
If any of the resources can be exhausted, how this is mitigated with the existing limits
(e.g. pods per node) or new limits added by this KEP?
Are there any tests that were run/should be run to understand performance characteristics better
and validate the declared limits?
-->

The kubelet is spliting the host UID/GID space for different pods, to use for
their user namespace mapping. The design allows for 65k pods per node, and the
resource is limited to maxPods per node (currently maxPods defaults to 110, it
is unlikely we will reach 65k).

For container runtimes, they might use more disk space or inodes to chown the
rootfs. This is if they chose to support this feature without relying on new
Linux kernels (or supporting old kernels too), as new kernels allow idmap mounts
and no overhead (space nor inodes) is added with that.

For CRIO and containerd, we are working to incrementally support all variations
(idmap mounts, no overhead;overlyafs metacopy param, that gives us just inode
overhead; and a full rootfs chown, that has space overhead) and document them
appropiately.

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

No changes to current kubelet behaviors. The feature only uses kubelet-local information.

###### What are other known failure modes?

- Some filesystem used by the pod doesn't support idmap mounts on the kernel used.
  - Detection: How can it be detected via metrics? Stated another way:
    how can an operator troubleshoot without logging into a master or worker node?

        See the pod events, it fails with:

    	  Warning  Failed     2s (x2 over 4s)  kubelet, 127.0.0.1  Error: failed to create containerd task: failed to create shim task: OCI runtime create failed: runc create failed: unable to start container process: error during container init: failed to fulfil mount request: failed to set MOUNT_ATTR_IDMAP on /var/lib/kubelet/pods/f037a704-742c-40fe-8dbf-17ed9225c4df/volumes/kubernetes.io~empty-dir/hugepage: invalid argument (maybe the source filesystem doesn't support idmap mounts on this kernel?): unknown

        Note the "maybe the source filesystem doesn't support idmap mounts on this kernel?" part.

  - Mitigations: What can be done to stop the bleeding, especially for already
    running user workloads?

        Remove the pod.spec.hostUsers field or disable the feature gate.

  - Diagnostics: What are the useful log messages and their required logging
    levels that could help debug the issue?
    Not required until feature graduated to beta.

        The Kubelet will get an error from the runtime and will propagate it to the pod (visible on
        the pod events).

        The idmap mount is created by the OCI runtime, not at the kubelet layer. At the kubelet layer, this
        is just another OCI runtime error.

        The container runtime logs will show something as shown before:

            OCI runtime create failed: runc create failed: unable to start container process: error during container init: failed to fulfil mount request: failed to set MOUNT_ATTR_IDMAP on /var/lib/kubelet/pods/f037a704-742c-40fe-8dbf-17ed9225c4df/volumes/kubernetes.io~empty-dir/hugepage: invalid argument (maybe the source filesystem doesn't support idmap mounts on this kernel?): unknown

        This is for runc, crun shows a [very similar error](https://github.com/containers/crun/pull/1382/files#diff-a260cc17fa708d3dc67330596e9d277d05601f8c48621a67379f7e0fb097fbb6R4089).

  - Testing: Are there any tests for failure mode? If not, describe why.

        There are tests in low level container runtimes.

        Testing this at the Kubernetes level is hard to do. We need to choose a file-system that doesn't
        support idmap mounts today, but that might not be the case after a kernel upgrade, as the new
        kernel might support idmap mounts for that file-system. Is very easy to create flaky tests for this
        at the Kubernetes layer.

        The low level container runtimes, that are the ones to do the idmap mount, have unit tests for this
        and return a clear error.



- Error allocating IDs for a pod user namespace

  - Detection: How can it be detected via metrics? Stated another way:
    how can an operator troubleshoot without logging into a master or worker node?

        Errors are returned on pod creation, directly to the user (visible on the pod events). No
        need to use metrics.

        See the pod events, it should contain something like:

          Warning  FailedCreatePodSandBox  3s    kubelet            Failed to create pod sandbox: can't allocate user namespace: could not find an empty slot to allocate a user namespace

        We can add a metric for this on general, but as any pod creation that fails has this direct
        feedback to the user, we don't think it is needed.

  - Mitigations: What can be done to stop the bleeding, especially for already
    running user workloads?

        Remove the pod.spec.hostUsers field or disable the feature gate.

  - Diagnostics: What are the useful log messages and their required logging
    levels that could help debug the issue?
    Not required until feature graduated to beta.

        No extra logs, the error is returned to the user (visible in the pod events).

  - Testing: Are there any tests for failure mode? If not, describe why.

        Yes, there are unit tests for this at the pkg/kubelet/userns package.

- Error to save/read pod mappings file.
The kubelet records the mappings used for a pod in a file. There can be an error while reading or
writing to this file.

  - Detection: How can it be detected via metrics? Stated another way:
    how can an operator troubleshoot without logging into a master or worker node?

        Errors are returned to the operation failed (like pod creation, visible on the pod events),
        no need to see metrics nor logs.

        Errors are returned to the either on:
         * Kubelet initialization: the initialization fails if the feature gate is active and there is a
           fatal error while reading the pod's mapping file.
         * Pod creation: an error is returned to the user (shown in the pod events) if there is an error to
           read or write to this file.

        We can add a metric for this, but as the errors have very direct feedback (kubelet init
        fails with a clear error or a pod creation fails with a clear error), we don't think it is needed.

  - Mitigations: What can be done to stop the bleeding, especially for already
    running user workloads?

        Remove the pod.spec.hostUsers field or disable the feature gate.

  - Diagnostics: What are the useful log messages and their required logging
    levels that could help debug the issue?
    Not required until feature graduated to beta.

        The most likely reason why this can fail is if the kernel doesn't support idmapped mounts.  In this
        case the error reported from the OCI runtime is clear enough, both the runc and crun OCI runtimes were
        instrumented to return a clear error message.

  - Testing: Are there any tests for failure mode? If not, describe why.

        There are extensive unit tests on the function that parses the file content.
        There are no tests for failures to read or write the file, the code-paths just return the errors
        in those cases.

- Error getting the kubelet IDs range configuration
  - Detection: How can it be detected via metrics? Stated another way:
    how can an operator troubleshoot without logging into a master or worker node?

        In this case the kubelet will fail to start with a clear error message.

  - Mitigations: What can be done to stop the bleeding, especially for already
    running user workloads?

        Disable the user namespace support via the feature flag, or fix the configuration.

  - Diagnostics: What are the useful log messages and their required logging
    levels that could help debug the issue?
    Not required until feature graduated to beta.

        The Kubelet will fail to start with a clear error message stating that it could not
        retrieve the range of IDs to use for the user namespaces.

  - Testing: Are there any tests for failure mode? If not, describe why.

        It is part of the system configuration.

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

- 2016: First iterations of this KEP, but code never landed upstream.
- Kubernetes 1.25: Support for stateless pods merged (alpha)
- Kubernetes 1.27: Support for stateless pods rework to rely on idmap mounts (alpha)
- Kubernetes 1.28: Support for stateful pods, renamed feature gate (alpha)

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

Here is a list of considerations raised in PRs discussion that were considered.
This list is not exhaustive.

### Don't use idmap mounts and rely chown all the files correctly

We explored the idea of not using idmap mounts for stateless pods, and instead
make sure we chown each file with the hostID a pod is mapped to (see more
details in the [example section](#example-without-idmap-mounts) of how file
access works with userns and without idmap mounts).

The problems were mostly:

* Changing the owner of files in configmaps/secrets/etc was a concern, those
  volume type today ignore FsUser setting and changing it to honor them was a
  concern for some members of the community due to possibly breaking some
  existing behavior. See discussions [here][fsgroup-1] and [here][fsgroup-2]

* Therefore, we need to rely on `fsGroup`, that always mark files readable for
  the group. This means this solution can't work when we NEED not to have
  permissions for the group (think of ssh keys, for example, that tools enforce no
  permission for the group)

* There are several other files that also need to have the proper permissions,
  like: /dev/termination-log, /etc/hosts, /etc/resolv.conf, /etc/hotname, etc.
  While it is completely possible to do, it adds more complexity to lot of parts
  of the code base and future Kubernetes features will have to take this into
  account too.

To exemplify the complexity of the last point, let's see some concrete examples.
`/etc/hosts` is created by [ensureHostsFile()][ensure-hosts-file] that doesn't
know pod attributes like the mapping. The same happens with
[SetupDNSinContainerizedMounter][dns-in-container] that doesn't know anything
else but the path. The same happens with the other files mentioned, all of them
and live in different "subsystems" of the kubelet, have very long call chains
will need to change to take the mapping or similar. Also, future patches, like
https://github.com/kubernetes/kubernetes/pull/108076 to fix a security bug in
`/dev/termination-log` also will need to be adjusted to take into account the
pod mappings.

If we go this route, while possible, more and more subsystems will have to
special-case if the pod uses userns and chown to a specific ID. Not only
existing places, but future places that create a file that is mounted will have
to know about the mapping of the pod.

Furthermore, some of these files are
[created by containerruntimes][containerd-mounts-files], so complexity creeps
out very easily.

Taking into account the 3 points (can't easily create secret/configmap files
with a specific owner; fsGroup has lot of limitations; and the complexity of
chowing each of these files in the kubelet), this approach was discarded. Future
KEPs can explore this path if they so want to.

[fsgroup-1]: https://github.com/kubernetes/kubernetes/pull/111090#discussion_r934057520
[fsgroup-2]: https://github.com/kubernetes/kubernetes/pull/111090#discussion_r935802376
[dns-in-container]: https://github.com/kubernetes/kubernetes/blob/7a55b76f28eddbbb7abf69038d4bd5abab833b4f/pkg/kubelet/network/dns/dns.go#L452-L479
[ensure-hosts-file]: https://github.com/kubernetes/kubernetes/blob/7a55b76f28eddbbb7abf69038d4bd5abab833b4f/pkg/kubelet/kubelet_pods.go#L327-L347
[containerd-mounts-files]: https://github.com/containerd/containerd/blob/3d32da8f607a2a43d7157499254713f42b3c6701/pkg/cri/server/container_create_linux.go#L61

### 64k mappings?

We discussed using shorter or even allowing for longer mappings in the past. The decision is to use
64k mappings (IDs from 0-65535 are mapped/valid in the pod).

The reasons to consider smaller mappings were valid only before idmap mounts was merged into the
kernel. However, idmap mounts is merged for some years now and we require it, making those reasons
void.

The issues without idmap mounts in previous iterations of this KEP, is that the IDs assigned to a
pod had to be unique for every pod in the cluster, easily reaching a limit when the cluster is "big
enough" and the UID space runs out.  However, with idmap mounts the IDs assigned to a pod just needs
to be unique within the node (and with 64k ranges we have 64k pods possible in the node, so not
really an issue). In other words: by using idmap mounts, we changed the IDs limit to be node-scoped
instead of cluster-wide/cluster-scoped.

Some use cases for longer mappings include:

There are no known use cases for longer mappings that we know of. The 16bit range (0-65535) is what
is assumed by all POSIX tools that we are aware of. If the need arises, longer mapping can be
considered in a future KEP.

### Allow runtimes to pick the mapping

Tim suggested that we might want to allow the container runtimes to choose the
mapping and have different runtimes pick different mappings. While KEP authors
disagree on this, we still need to discuss it and settle on something.  This was
[raised here](https://github.com/kubernetes/enhancements/pull/3065#discussion_r798760382)

Furthermore, the reasons mentioned by Tim Hockin (some nodes having CRIO, some others having containerd,
etc.) are handled correctly now. Different nodes can use different container runtimes, if a custom
range needs to be used by the kubelet, that can be configured per-node.

Therefore, this old concerned is now resolved.

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
