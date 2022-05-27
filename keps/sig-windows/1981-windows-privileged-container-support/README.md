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
# KEP-1981: Windows Privileged Containers and Host Networking Mode

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
  - [Use case 1: Privileged Daemon Sets](#use-case-1-privileged-daemon-sets)
  - [Use case 2: Administrative tasks](#use-case-2-administrative-tasks)
  - [Notes/Constraints/Caveats (Optional)](#notesconstraintscaveats-optional)
  - [Risks and Mitigations](#risks-and-mitigations)
  - [Pod Security Implications](#pod-security-implications)
- [Design Details](#design-details)
  - [Overview](#overview)
    - [Networking](#networking)
    - [Resource Limits](#resource-limits)
    - [Container Lifecycle](#container-lifecycle)
    - [Container users](#container-users)
    - [Container Mounts](#container-mounts)
    - [Container Images](#container-images)
    - [Container Image Build/Definition](#container-image-builddefinition)
  - [CRI Implementation Details](#cri-implementation-details)
  - [Kubernetes API updates](#kubernetes-api-updates)
    - [WindowsSecurityContextOptions.HostProcess Flag](#windowssecuritycontextoptionshostprocess-flag)
      - [Alternatives](#alternatives)
    - [Host Network Mode](#host-network-mode)
    - [Example deployment spec](#example-deployment-spec)
  - [Kubelet Implementation Details](#kubelet-implementation-details)
    - [CRI Support Only](#cri-support-only)
    - [Feature Gates](#feature-gates)
  - [Test Plan](#test-plan)
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
- [Alternatives](#alternatives-1)
- [Open Questions](#open-questions)
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
Privileged containers are containers that are enabled with similar access to the host as processes that run on the host directly. With privileged containers, users can package and distribute management operations and functionalities that require host access while retaining versioning and deployment methods provided by containers. Linux privileged containers are currently used for a variety of key scenarios in Kubernetes, including kube-proxy (via kubeadm), storage, and networking scenarios. Support for these scenarios in Windows currently requires workarounds via proxies or other implementations. This proposal aims to extend the Windows container model to support privileged containers. This proposal also aims to enable host network mode for privileged networking scenarios. Enabling privileged containers and host network mode for privileged containers would enable users to package and distribute key functionalities requiring host access.


## Motivation

<!--
This section is for explicitly listing the motivation, goals and non-goals of
this KEP.  Describe why the change is important and the benefits to users. The
motivation section can optionally provide links to [experience reports] to
demonstrate the interest in a KEP within the wider Kubernetes community.

[experience reports]: https://github.com/golang/go/wiki/ExperienceReports
-->
The lack of privileged container support within the Windows container model has resulted in separate workarounds and privileged proxies for Windows workloads that are not required for Linux workloads. These workarounds have provided necessary functionality for key scenarios such as networking, storage, and device access, but have also presented many challenges, including increased available attack surfaces, complex change and update management, and scenario specific solutions. There is significant interest from the community for the Windows container model to support privileged containers and host network mode (which enable pods to be created in the host’s network compartment/namespace, as opposed to getting their own) to transition off such workarounds and align more closely with Linux support and operational models.

Furthermore, since kube-proxy cannot be run as a privileged daemonset, it must either be run with a proxy or directly on the host as a service. In the case that it is run as a service, the admin kube config must be stored on the Windows node which poses a security concern. This is also true for networking daemons such as Flannel.

### Goals

<!--
List the specific goals of the KEP. What is it trying to achieve? How will we
know that this has succeeded?
-->

- To provide a method to build, launch, and run a Windows-based container with privileged access to host resources, including the host network service, devices, disks (including hostPath volumes), etc.
- To enable access to host network resources for **privileged** containers and pods with host network mode

### Non-Goals

<!--
What is out of scope for this KEP? Listing non-goals helps to focus discussion
and make progress.
-->

- To provide access to host network resources for **non-privileged** containers and pods.
- To provide a privileged mode for Hyper-V containers, or a method to run privileged process containers within a Hyper-V isolation boundary. This is a non-goal as running a Hyper-V container in the root namespace from within the isolation boundary is not supported.
- To enable privileged containers for Docker. This will only be for containerd.
- To align privileged containers with pod namespaces - this functionality may be addressed in a future KEP.
- Enabling the ability to mix privileged and non-privileged containers in the same Pod. (Multiple privileged containers running in the same Pod will be supported.)

## Proposal

<!--
This is where we get down to the specifics of what the proposal actually is.
This should have enough detail that reviewers can understand exactly what
you're proposing, but should not include things like API designs or
implementation. The "Design Details" section below is for the real
nitty-gritty.
-->

### Use case 1: Privileged Daemon Sets

Privileged daemon sets are used to deploy networking (CNI), storage (CSI), device plugins, kube-proxy, and other components to Linux nodes. Currently, similar set-up and deployment operations utilize wins or dedicated proxies (i.e. CSI-proxy, HNS-Proxy) or these components are installed as services running on Windows nodes. With Windows privileged containers many of these components could run inside containers increasing consistency between how they are deployed and/or managed on Linux and Windows. For networking scenarios, host network mode will enable these privileged deployments to access and configure host network resources.

Some interesting scenario examples:

- Cluster API
- CSI Proxy
- Logging Daemons

### Use case 2: Administrative tasks

Windows privileged containers would also enable a wide variety of administrative tasks without requiring cluster operations to log onto each Windows nodes. Tasks like installing security patches, collecting specific event logs, etc could all be done via deployments of privileged containers.

### Notes/Constraints/Caveats (Optional)

<!--
What are the caveats to the proposal?
What are some important details that didn't come across above?
Go in to as much detail as necessary here.
This might be a good place to talk about core concepts and how they relate.
-->

- Host network mode support is only targeted for privileged containers and pods.
- Privileged pods can only consist of privileged containers. Standard Windows Server containers or other non-privileged containers will not be supported. This is because containers in a Kubernetes pod share an IP. For the privileged containers with host network mode, this container IP will be the host IP. As a result, a pod cannot consist of a privileged container with the host IP and an unprivileged Windows Server container(s) sharing a vNic on the host with a different IP, or vice versa.
- We are currently investigating service mesh scenarios where privileged containers in a pod will need host networking access but run alongside non-privileged containers in a pod. This would require further investigation and is out of scope for this KEP.

### Risks and Mitigations

<!--
What are the risks of this proposal, and how do we mitigate? Think broadly.
For example, consider both security and how this will impact the larger
Kubernetes ecosystem.

How will security be reviewed, and by whom?

How will UX be reviewed, and by whom?

Consider including folks who also work outside the SIG or subproject.
-->
Most of the fundamental changes to enable this feature for Windows containers is dependent on changes within [hcsshim](https://github.com/microsoft/hcsshim), which serves as the runtime (container creation and management) coordinator and shim layer for containerd on Windows.

However:

- Several upstream changes are required to support this feature in Kubernetes, including changes to containerd, OCI, CRI, and kubelet. The identified changes include (see CRI and Kubelet Implementation Details below for more details on changes):
  - Containerd: enabling host network mode for privileged containers and pods ([working prototype demo](https://drive.google.com/file/d/1WQO2NDX34Z1FPbW7jEymhcPMY4AZWQSE/view)). Prototype is done using containerd runtimehandler but this proposal is to use cri-api.
    - OCI spec: https://github.com/opencontainers/runtime-spec
      - Updates pending decisions made in this KEP regarding namings.
    - CRI-api:
      - Adding `WindowsPodSandboxConfig` and `WindowsSandboxSecurityContext` message
      - Adding `host_process` flag to `WindowsContainerSecurityContext`
      - Pass security context and flag of runtime spec to podsandbox spec (not currently supported, open issue: https://github.com/kubernetes/kubernetes/issues/92963)
  - Kubelet: Pass host_process flag and windows security context options to runtime spec.
- There are risks that changes at each of these levels may not be supported.
  - If containerd changes are not supported, host network mode will not be enabled.This would restrict the scenarios that privileged containers would enable, as CNI plugins, network policy, etc. rely on host network mode to enable access to host network resources.
  - If CRI changes to enable a privileged flag are not supported, there would be a less-ideal workaround via annotations in the pod container spec.
  - The CRI changes may make an annotation in the OCI spec until the OCI updates are included.

### Pod Security Implications

For alpha we will update [Pod Security Standards](https://kubernetes.io/docs/concepts/security/pod-security-standards/) with information on the new `hostProcess` flag.

Additionally, privileged containers may impact other pod security policies (PSPs) outside of allowPrivilegedEscalation. We will provide guidance similar to [Pod Security Standards](https://kubernetes.io/docs/concepts/security/pod-security-standards/) for Windows privileged containers when graduating this feature out of alpha. There is an [analysis for non-privileged containers](https://github.com/kubernetes/kubernetes/issues/64801#issuecomment-668897952) which can be augmented with the details below.The anticipated impacted PSPs include:

<table>
  <tr>
   <td>Use case
   </td>
   <td>Field name
   </td>
   <td>Applicable
   </td>
   <td>Scenario
   </td>
   <td>Priority
   </td>
  </tr>
  <tr>
   <td>Running of privileged containers
   </td>
   <td>privileged
   </td>
   <td>no
   </td>
   <td>Not applicable. Windows privileged containers will be controlled with a new `WindowsSecurityContextOptions.HostProcess` instead of the existing `privileged` field due to fundamental differences in their implementation on Windows.
   </td>
   <td>Alpha
   </td>
  </tr>
  <tr>
   <td>Usage of host namespaces
   </td>
   <td>HostPID, hostIPC
   </td>
   <td>no
   </td>
   <td>Windows does not have configurable PID/IPC namespaces (unlike Linux). Windows containers are always assigned their own process namespace. Job objects always run in the host's process namespace. These behaviors are not configurable. Future plans in this area include improvements to enable scheduling pods that can contain both normal and HostProcess/Job Object containers. These fields would not makes in this scenario because Windows cannot configure PID/IPC namespaces like in Linux.
   </td>
   <td>N/A
   </td>
  </tr>
  <tr>
   <td>Usage of host networking and ports
   </td>
   <td>hostnetwork
   </td>
   <td>yes
   </td>
   <td>Will be in host network by default initially. Support to set network to a different compartment may be desirable in the future.
   </td>
   <td>Beta
   </td>
  </tr>
  <tr>
   <td>Usage of volume types
   </td>
   <td>Volumes
   </td>
   <td>no
   </td>
   <td>Not applicable.
   </td>
   <td>N/A
   </td>
  </tr>
  <tr>
   <td>Usage of the host filesystem
   </td>
   <td>Allowed host paths
   </td>
   <td>no
   </td>
   <td>Job objects have full access to write to the root file system. Current design does not have a way to control access to read only. Instead privileged/job object containers can be ran as users with limited/scoped files system access via RunAsUsername
   </td>
   <td>N/A
   </td>
  </tr>
  <tr>
   <td>Allow specific FlexVolume drivers
   </td>
   <td>Flex volume
   </td>
   <td>no
   </td>
   <td>Not applicable.
   </td>
   <td>N/A
   </td>
  </tr>
  <tr>
   <td>Allocating an FSGroup that owns the pod's volumes
   </td>
   <td>Fsgroup (file system group)
   </td>
   <td>no
   </td>
   <td>The privileged container can be tied to run as a particular user that determines access to different fsgroups.
   </td>
   <td>N/A 
   </td>
  </tr>
  <tr>
   <td>The user and group IDs of the container
   </td>
   <td>Runasuser, runasgroup, supplementalgroup
   </td>
   <td>no
   </td>
   <td>Assigning users to groups would have to occur at node provisioning, or via a privileged container deployment.
   </td>
   <td>N/A
   </td>
  </tr>
  <tr>
   <td>
   </td>
   <td>Allowprivilegedescalation, default
   </td>
   <td>no
   </td>
   <td>Privilege via job objects is not granularly configurable.
   </td>
   <td>N/A
   </td>
  </tr>
  <tr>
   <td>Linux capabilities
   </td>
   <td>Capabilities
   </td>
   <td>no
   </td>
   <td>Windows OS has a concept of “capabilities” (referred to as “privileged constants” but they are not supported in the platform today.
   </td>
   <td>N/A
   </td>
  </tr>
  <tr>
   <td>Restrictions that could be applied to Windows Privileged Containers
   </td>
   <td>Other restrictions for job objects
   </td>
   <td>TBD
   </td>
   <td>There are restrictions that could be enabled via the job object, i.e. <a href="https://docs.microsoft.com/en-us/windows/win32/api/winnt/ns-winnt-jobobject_basic_ui_restrictions">UI restrictions</a>
   </td>
   <td>N/A
   </td>
  </tr>
  <tr>
   <td>Use GMSA with privileged containers
   </td>
   <td>GMSA – would need to implement
   </td>
   <td>yes
   </td>
   <td>Required for auth to domain controller.
   </td>
   <td>GA
   </td>
  </tr>
</table>

## Design Details

<!--
This section should contain enough information that the specifics of your
change are understandable. This may include API specs (though not always
required) or even code snippets. If there's any ambiguity about HOW your
proposal will be implemented, this is the place to discuss them.
-->

### Overview

Windows privileged containers will be implemented with [Job Objects](https://docs.microsoft.com/en-us/windows/win32/procthread/job-objects), a break from the previous container model using server silos. Job objects provide the ability to manage a group of processes as a group, and assign resource constraints to the processes in the job. Job objects have no process or file system isolation, enabling the privileged payload to view and edit the host file system with the correct permissions, among other host resources. The init process, and any processes it launches or that are explicitly launched by the user, are all assigned to the job object of that container. When the init process exits or is signaled to exit, all the processes in the job will be signaled to exit, the job handle will be closed and the storage will be unmounted.

Because Windows privileged containers will work much differently than Linux privileged containers they will be referred to as **HostProcess** containers in kubernetes specs and user-facing documentation. Hopefully this will encourage users to seek documentation to better understand the capabilities and behaviors of these privileged containers.

![Privileged Container Diagram](Privileged.png)

#### Networking

- The container will be in the host’s network namespace (default network compartment) so it will have access to all the host’s network interfaces and have the host's IP as well.

#### Resource Limits

- Resource limits (disk, memory, cpu count) will be applied to the job and will be job wide. For example, with a limit of 10 MB is set for the job, if every process in the jobs memory allocations added up exceeds 10 MB this limit would be reached. This is the same behavior as other Windows container types. These limits would be specified the same way they are currently for whatever orchestrator/runtime is being used.
- Note: HostProcess containers will have access to nodes root filesystem. Disk limits and resource usage will only apply to the scatch volume provisioned for each HostProcess container.

#### Container Lifecycle

- The container's lifecycle will be managed by the container runtime just like other Windows container types.

#### Container users

- By default `hostProcess` containers can run one of the following system accounts:
  - `NT AUTHORITY\SYSTEM`
  - `NT AUTHORITY\Local service`
  - `NT AUTHORITY\NetworkService`
- Running privileged containers as non SYSTEM/admin accounts will be the primary way operators can restrict access to system resources (files, registry, named pipes, WMI, etc).
- To run a `hostProcess` container as a non SYSTEM/admin account, a local users Group must first be created on the host.
Permissions to restrict access to system resources can should be configured to allow/deny access for the Group.
When a new `hostProcess` container is created with the name of a local users Group set as the `runAsUserName` then a temporary user account will be created as a member of the specified group for the container to run as.

  - More information on Windows resource access can be found at <https://docs.microsoft.com/archive/msdn-magazine/2008/november/access-control-understanding-windows-file-and-registry-permissions>
  - Example of configuring non SYSTEM/admin account can be found at <https://github.com/microsoft/hcsshim/pull/1286#issuecomment-1030223306>

#### Container Mounts

- Window's bind-filter driver will be used to create a view that merges the host's OS filesystem with container-local volumes.
When `hostProcess` containers are started a new volume will be created which contains the contents of the contaner image.
This volume will be mounted to `c:\hpc`. The default working directory for `hostProcess` containers will also be set to `c:\hpc`.
- Volume mounts (includinge service account tokens) will be supported for `hostProcess` containers and can be accessed just the same way as regular Windows Server containers.
  - Named Pipe mounts will **not** be supported.
    Instead named pipes should be accessed via their path on the host (\\\\.\\pipe\\*).
    The following error will be returned if `hostProcess` containers attempt to use name pipe mounts -
    https://github.com/microsoft/hcsshim/blob/358f05d43d423310a265d006700ee81eb85725ed/internal/jobcontainers/mounts.go#L40.
  - Unix domain sockets mounts also not not be supported for `hostProces` containers.
    Unix domain sockets can be accessed via their paths on the host like named pipes.
  - Mounting directories from the host OS into `hostProcess` containers will work just like with normal containers but this is not recommend.
    Instead workloads should access the host OS's file-system as if it was not being run in a container.
  - All other volume types supported for normal containers on Windows will work with `hostProcess` containers.
- `HostProcess` Containers will have full access to the host file-system (unless restricted by filed-based ACLs and the run_as_username used to start the container).
- There will be no `chroot` equivalent.

- Note: Behavior of volume mounts will differ between the alpha/beta (old) implementation of this feature and the stable (new) implementation.
Designing/testing/validation of an acceptable solution for handling volume mounts w.r.t. `hostProcess` containers was a primary reason for keeping the featuer in `beta`. (Previous behavior volume mount behavior can be viewed [here](https://github.com/kubernetes/enhancements/blob/cc4052fa88508fcbb20b978b8c181d06acf734b1/keps/sig-windows/1981-windows-privileged-container-support/README.md#container-mounts))
A recording of the behaviors differces from a SIG-Windows community meeting can be found [here](https://youtu.be/8GeZKXgvkdY?t=309).
  - Also note -  In the recording it was mentioned that this functionally might not be supported on WS2019. This functionality will be avaible in WS2019 but will require an OS patch (ETA: July 2022).
- Containerd v1.7+ will be required for new volume mount behavior.

#### Container Images

- `HostProcess` containers can be built on top of existing Windows base images (nanoserver, servercore, etc).
- A new Windows container base image will not be introduced for `HostProcess` containers.
- It is recommended to use nanoserver as the base image for `hostProcess` containers since it has the smallest footprint.
- `HostProcess` containers will not inherit the same [compatibility requirements](https://docs.microsoft.com/en-us/virtualization/windowscontainers/deploy-containers/version-compatibility) as process isolated containers from an OS perspective. Container runtimes like containerd may be able to use fields on `WindowsPodSandboxConfig` to identify `HostProcess` containers and skip OS version checks when pulling/starting these containers in the future.

#### Container Image Build/Definition

- `HostProcess` container images can be built with Docker.
- Only a subset of dockerfile operations will be supported (ADD, COPY, PATH, ENTRYPOINT, etc).
  - Note: The subset of dockerfile operations supported for `HostProcess` containers is very close to the subset of operations supported when building images for other OS's with buildkit (similar to how the [pause image](https://github.com/kubernetes/kubernetes/tree/master/build/pause) is built in kubernetes/kubernetes)
- Documentation on building `HostProcess` containers will be added at either docs.microsoft.com or a new github repository.

### CRI Implementation Details

We will need to add a `hostProcess` field to the runtime spec. We can model this after the Linux pod security context and container security context that is a boolean that is set to `true` for privileged containers. References:

- [LinuxSandboxSecurityConfig](https://github.com/kubernetes/cri-api/blob/master/pkg/apis/runtime/v1alpha2/api.proto#L293)
- [LinuxSandboxSecurityContext](https://github.com/kubernetes/cri-api/blob/master/pkg/apis/runtime/v1alpha2/api.proto#L28)
- [LinuxContainerSecurityContext](https://github.com/kubernetes/cri-api/blob/master/pkg/apis/runtime/v1alpha2/api.proto#L612)

For Windows we are proposing the following updates to CRI-API:

Add WindowsPodSandboxConfig (and it to PodSandboxConfig)

```protobuf
message WindowsPodSandboxConfig {
  WindowsSandboxSecurityContext security_context = 1;
}
```

Add WindowsSandboxSecurityContext:

```protobuf
message WindowsSandboxSecurityContext {
  string run_as_username = 1;
  string credential_spec = 2;
  bool host_process = 3;
}
```

Update WindowsContainerSecurityContext by adding host_process field:

```protobuf
message WindowsContainerSecurityContext {
  string run_as_username = 1;
  string credential_spec = 2;
  bool host_process = 3;
}
```

*Note:* For alpha annotations on RunPodSandbox and CreateContainer CRI calls may be used until a version of containerd with Windows privileged container support is released.

### Kubernetes API updates

#### WindowsSecurityContextOptions.HostProcess Flag

A new `*bool` field named `hostProcess` will be added to [WindowsSecurityContextOptions](https://kubernetes.io/docs/reference/generated/kubernetes-api/v1.20/#windowssecuritycontextoptions-v1-core).

On Windows, all containers in a pod must be privileged. Because of this behavior and because `WindowsSecurityContextOptions` already exists on both `PodSecurityContext` and `Container.SecurityContext` Windows containers will use this new field instead re-using the existing `privileged` field which only exists on `SecurityContext`.
Additionally, the existing `privileged` field does not clearly describe what capabilities the container has (see https://github.com/kubernetes/kubernetes/issues/44503).
Documentation will be added to clearly describe what capabilities these new "hostProcess" containers have.

Current behavior applies `PodSecurityContext.WindowsSecurityContextOptions` settings to all `Container.SecurityContext.WindowsSecurityContextOptions` unless those settings are already specified on the container. To address this the following API validation will be added:

- If `PodSecurityContext.WindowsSecurityContextOptions.HostProcess = true` is set to true then no container in the pod sets `Container.SecurityContext.WindowsSecurityContextOptions.HostProcess = false`
- If `PodSecurityContext.WindowsSecurityContextOptions.HostProcess` is not set then all containers in a pod must set `Container.SecurityContext.WindowsSecurityContextOptions.HostProcess = true`
- If `PodSecurityContext.WindowsSecurityContextOptions.HostProcess = false` no containers may set `Container.SecurityContext.WindowsSecurityContextOptions.HostProcess = true`
- `hostNetwork = true` must be set explicits if the pod contains all hostProcess containers (this value will not be inferred and/or defaulted)

Additionally kube-apiserver will disallow `hostProcess` containers to be scheduled if `--allow-privileged=false` is passed as an argument.
https://github.com/kubernetes/kubernetes/blob/release-1.20/pkg/apis/core/validation/validation.go#L5767-L5771 for reference.

##### Alternatives

Option 1: Re-use `SecurityContext.Privileged` field.

Re-using the existing `SecurityContext.Privileged` field was considering and here are the pros/cons considered:

Pros

- The field already exists and many policy tools already leverage it.

Cons

- Privileged containers on Windows will operate very differently than privileged containers on Linux. Having a new field should help avoid confusion around the differences between the two.
- The privileged field does not have clear meaning for Linux containers today (see comments above).
- `WindowsSecurityContextOptions.RunAsUserName` will the the primary way of restricting access to host/node resources (See [Container users](#container-users)). It is desirable that `RunAsUserName` and `HostProcess` fields live on the same property.
- API validation to ensure all containers are either privileged or not will be difficult because there is no way of definitively knowing that a pod is intended for a Windows node.

#### Host Network Mode

Host Network mode for privileged Windows containers will always be enabled, as the pod will automatically get the host IP.

Privileged Windows containers will be unable to align to pod namespaces due to limitations in the Windows OS. This functionality will likely be enabled in the future through a new KEP.

Because of this we will require that `hostNetwork` is set to `true` when scheduling privileged pods. This will allow existing policy tools to detect and act on privileged Windows containers without any updates. In the future if/when functionality is added to support joining privileged containers to pod networks this validation will be revisited.

#### Example deployment spec

Here are two examples of valid specs each containing two privileged Windows containers:

```yaml
spec:
  hostNetwork: true
  securityContext:
    windowsOptions:
      hostProcess: true
  containers:
  - name: foo
    image: image1:latest
  - name: bar
    image: image2:latest
  nodeSelector:
    "kubernetes.io/os": windows
```

```yaml
spec:
  hostNetwork: true
  containers:
  - name: foo
    image: image1:latest
    securityContext:
      windowsOptions:
        hostProcess: true
  - name: bar
    image: image2:latest
    securityContext:
      windowsOptions:
        hostProcess: true
  nodeSelector:
    "kubernetes.io/os": windows
```

### Kubelet Implementation Details

Kubelet will pass privileged flag from `WindowsSecurityContextOptions` to the appropriate CRI layer calls.

*Note:* For alpha kubelet may add well-known annotations to CRI calls if privileged flags are set.

Add functionality to Kuberuntime_sandbox to:

- Split out the linux sandbox creation and add [windows sandbox creation](https://github.com/kubernetes/kubernetes/blob/a9f1d72e1de6450b890a0c0e451725468f54f515/pkg/kubelet/kuberuntime/kuberuntime_sandbox.go#L136)
- Configure all privileged Windows pods to join the [host network](https://github.com/kubernetes/kubernetes/blob/a9f1d72e1de6450b890a0c0e451725468f54f515/pkg/kubelet/kuberuntime/kuberuntime_sandbox.go#L98)

The following extra validation will be added to the kubelet for Windows. These checks will ensure privileged pods work correctly on Windows if these are not validated by apiserver.

- Ensure all containers in a pod privileged, if any are.
- Ensure `hostNetwork = true` is set if pod contains privileged containers.

#### CRI Support Only

There are no plans to update Docker and/or dockershim to have support for privileged containers due to requirements on HCSv2.
Currently containerd is the only container runtime with a Windows implementation so containerd will be required.

Validation will be added in the kubelet to fail to schedule a pod if the node is configured to use dockershim and the pod contains privileged Windows containers.

#### Feature Gates

Privileged container functionally on windows will be gated behind a new `WindowsHostProcessContainers` feature gate.

https://kubernetes.io/docs/reference/command-line-tools-reference/feature-gates/#feature-stages

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

Alpha

- Unit tests around validation logic for new API fields.
- Add e2e test validating basic privileged container functionality (pod starts and run in a privileged context on the node)
- Update Pod Security Standards doc to dissallow `hostProcess` containers in the baseline/default and restricted policies.

Beta

- Validate running kubeproxy as a daemon set
- Validate CSI-proxy running as a daemon set
- Validate running a CNI implementation as a daemon set
- Validate behaviors of various volume mount types as described in [Container Mounts](#container-mounts) with e2e tests
- Add e2e tests to test different ways to construct paths for container command, args, and workingDir fields for both `hostProcess` and non-hostProcess containers. These tests will include constructing paths with and without `$CONTAINER_SANDBOX_MOUNT_POINT` set and with different combinations of forward and backward slashes.

Graduation

- Add e2e tests to validate running `hostProcess` containers as non SYSTEM/admin accounts
- Update e2e tests for new volume mount behavior as desdribed in [Container Mounts](#container-mounts)

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

Alpha plan

- Version of containerd: Target v1.5
- Version of Kubernetes: Target 1.22
- OS support: Windows 2019 LTSC and all future versions of Windows Server
- Alpha Feature Gate for passing privileged flag **or** annotations to CRI calls.

Graduation to Beta

- Kubernetes Target 1.23
- Set `WindowsHostProcessContainers` feature gate to `beta`
- Go through PSP Linux test (e2e: validation & conformance) and make them relevant for Windows (which apply, which don't and where we need to write new tests).
- Provide guidance similar to Pod Security Standards for Windows privileged containers.
- CRI Support for HostProcess containers.
  - Containerd release is available with HostProcess support (Either v1.6 OR changes backported to a v1.5 patch) - (https://github.com/containerd/containerd/pull/5131)
  - [Windows Host Process annotations](https://github.com/kubernetes/kubernetes/blob/7705b300e2085c3864bb1e49a7302bf17f080219/pkg/kubelet/kuberuntime/labels.go#L46-L50) removed from CRI. (Discussed at (https://github.com/kubernetes/kubernetes/pull/99576#discussion_r635392090))
- OS support: Windows 2019 LTSC and all future versions of Windows Server.
- Documentation for `HostProcess` containers on https://kubernetes.io/.
  - Includes clarification around disk limits mentioned in [Resource Limits](#resource-limits).
  - Documentation on docs.microsoft.com for building `HostProcess` container images.
- Update validation logic for `HostProcess` containers in api-server to handle [ephemeral containers](https://github.com/kubernetes/enhancements/tree/d4aa2b45412bae677e14d44477a73288e3e987fc/keps/sig-node/277-ephemeral-containers)
  - Note: If ephemeral container is also a `HostProcess` container then all containers in the pod must also be `HostProcess` containers (and vise versa).

Graduation to GA:

- Add documentation for running as a non-SYSTEM/admin account to k8s.io
- Update documention on how volume mounts are set up for `hostProcess` containers on k8s.io
- Set `WindowsHostProcessContainers` feature gate to `GA`
- Provide reference images/workloads using the `GA` volume mounting behavior in Cluster-API-Provider-azure (which is used to run the majority of Windows e2e test passes)

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

- Windows: This implementation requires no backports for OS components.
- Kubernetes: No changes required outside of ensuring feature gates are set while feature is in development.
- Containerd: Must run a version of containerd with privileged container support (targeting v1.5+).

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

N/A

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

_This section must be completed when targeting alpha to a release._

* **How can this feature be enabled / disabled in a live cluster?**
  - [x] Feature gate (also fill in values in `kep.yaml`)
    - Feature gate name: WindowsHostProcessContainers
    - Components depending on the feature gate: Kubelet, kube-apiserver
  - [ ] Other
    - Describe the mechanism:
    - Will enabling / disabling the feature require downtime of the control
      plane?
    - Will enabling / disabling the feature require downtime or reprovisioning
      of a node? (Do not assume `Dynamic Kubelet Config` feature is enabled).

* **Does enabling the feature change any default behavior?**
  No

* **Can the feature be disabled once it has been enabled (i.e. can we roll back
  the enablement)?**
  This feature can be disabled.
  If this feature flag is disabled in kube-apiserver than new pods which try to schedule `hostProcess` containers will be rejected by kube-apiserver.
  If this flag is disabled in the kubelet then new `hostProcess` containers are will not be started and an appropriate event will be emitted.

* **What happens if we reenable the feature if it was previously rolled back?**
Newly created privileged Windows containers will run as expected.

* **Are there any tests for feature enablement/disablement?**
No

### Rollout, Upgrade and Rollback Planning

_This section must be completed when targeting beta graduation to a release._

* **How can a rollout fail? Can it impact already running workloads?**
  Try to be as paranoid as possible - e.g., what if some components will restart
   mid-rollout?

* **What specific metrics should inform a rollback?**

* **Were upgrade and rollback tested? Was the upgrade->downgrade->upgrade path tested?**
  Describe manual testing that was done and the outcomes.
  Longer term, we may want to require automated upgrade/rollback tests, but we
  are missing a bunch of machinery and tooling and can't do that now.

* **Is the rollout accompanied by any deprecations and/or removals of features, APIs, 
fields of API types, flags, etc.?**
  Even if applying deprecation policies, they may still surprise some users.

### Monitoring Requirements

_This section must be completed when targeting beta graduation to a release._

* **How can an operator determine if the feature is in use by workloads?**

  Kubelet metrics will be updated to report the number of HostProcess containers started and number of errors started.

  TBD: Confirm the best way to accomplish this is to add new [values/metric labels](https://github.com/kubernetes/kubernetes/blob/fe099b2abdb023b21a17cd6a127e381b846c1a1f/pkg/kubelet/metrics/metrics.go#L96-L99) for `StartedContainersTotal` and `StartedContainersError` counters. Otherwise we could add new counters.



* **What are the SLIs (Service Level Indicators) an operator can use to determine 
the health of the service?**
  - [x] Metrics
    - Metric name: **started_host_process_containers_total** - reports the total number of host-process containers started on a given node
    - Metric name: **started_host_process_containers_errors_total** - reports the total number of host-process containers that have failed to given node.
    - [Optional] Aggregation method:
    - Components exposing the metric: Kubelet
    - Notes: Both metrics were added in v1.23 and are validated with [e2e tests](https://github.com/kubernetes/kubernetes/blob/fdb2d544751adc9fd2f6fa5075e9a16df7d352df/test/e2e/windows/host_process.go#L483-L575)
  - [ ] Other (treat as last resort)
    - Details:

* **What are the reasonable SLOs (Service Level Objectives) for the above SLIs?**
 The same SLOs for starting/stopping non-hostprocess containers would apply here.

* **Are there any missing metrics that would be useful to have to improve observability 
of this feature?**
  N/A

### Dependencies

_This section must be completed when targeting beta graduation to a release._

* **Does this feature depend on any specific services running in the cluster?**

  - [ContainerD]
    - Usage description:
      - `HostProcess` containers support will not be added to dockershim.
      - Containerd v1.5.6+ is required.
      - Impact of its outage on the feature: Containers will fail to start.
      - Impact of its degraded performance or high-error rates on the feature: Containers may behave expectantly and node may go into the NotReady state.

### Scalability

_For alpha, this section is encouraged: reviewers should consider these questions
and attempt to answer them._

_For beta, this section is required: reviewers must answer these questions._

_For GA, this section is required: approvers should be able to confirm the
previous answers based on experience in the field._

* **Will enabling / using this feature result in any new API calls?**
  No

* **Will enabling / using this feature result in introducing new API types?**
  No

* **Will enabling / using this feature result in any new calls to the cloud 
provider?**
  No

* **Will enabling / using this feature result in increasing size or count of 
the existing API objects?**
  A new field is being added so API object size will grow *slightly* larger.

* **Will enabling / using this feature result in increasing time taken by any 
operations covered by [existing SLIs/SLOs]?**
  No

* **Will enabling / using this feature result in non-negligible increase of 
resource usage (CPU, RAM, disk, IO, ...) in any components?**
  No - `HostProcess` containers will honor limits/reserves specified in the specs and will count against node quota just like unprivileged containers.

### Troubleshooting

The Troubleshooting section currently serves the `Playbook` role. We may consider
splitting it into a dedicated `Playbook` document (potentially with some monitoring
details). For now, we leave it here.

_This section must be completed when targeting beta graduation to a release._

* **How does this feature react if the API server and/or etcd is unavailable?**
  This feature will not change any behaviors around Pod scheduling if API server and/or etcd is unavailable.

* **What are other known failure modes?**
  For each of them, fill in the following information by copying the below template:
  N/A

* **What steps should be taken if SLOs are not being met to determine the problem?**

[supported limits]: https://git.k8s.io/community//sig-scalability/configs-and-limits/thresholds.md
[existing SLIs/SLOs]: https://git.k8s.io/community/sig-scalability/slos/slos.md#kubernetes-slisslos

  Kubelet and/or containerd logs will need to be inspected if problems are encountered creating HostProcess containers on Windows nodes.

## Implementation History

- **2020-09-11:** [Issue #1981](https://github.com/kubernetes/enhancements/issues/1981) created.
- **2021-12-17:** Initial KEP draft merged - [#2037](https://github.com/kubernetes/enhancements/pull/2037).
- **2021-02-17:** KEP approved for alpha release - [#2288](https://github.com/kubernetes/enhancements/pull/2288).
- **2021-05-20:** Alpha implementation PR merged - [kubernetes/kubernetes#99576](https://github.com/kubernetes/kubernetes/pull/99576).
- **2021-08-05:** K8s 1.22 released with alpha support for `WindowsHostProcessContainers` feature.
- **2021-08-21:** HostProcessContainers (via CRI) support added to containerd - [containerd/containerd#5131](https://github.com/containerd/containerd/pull/5131).
- **2021-12-07:** K8s 1.23 released with beta support for `WindowsHostProcessContainers` feature.
- **2022-02-15:** Containerd 1.6.0 relased with support for HostProcessContainers.

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

- Use containerd Runtimehandlers and K8s RuntimeClasses - Runtimehandlers are using the prototype. Adding the ability to the CRI provides kubelet to have more control over the security context and and fields that it allows through giving additional checks (such as runasnonroot).

- Use annotations on CRI to pass privileged flag to containerd - Adding the field to the CRI spec allows for the existing CRI calls to work as is. The resulting code is cleaner and doesn’t rely on magic strings.  There is currently a PR for adding the SecurityFields to the CRI API adding Sandbox level security support for window containers.  The Runasusername will be required for privileged containers to make sure every container (including pause) runs as the correct user to limit access to the file system.

## Open Questions
- What’s the future of plug-ins that will be impacted 
- What will be the future CSI-proxy and other plug-ins that will be impacted?
  - CSI-proxy and HNS-proxy are likely to be impacted
- Container base image support
  - Is “from scratch” required
  - Would a slimmer “privileged base image” be more desirable than using stand server core
- Container image build differences with traditional windows server and impacts on image use and distribution
- Should PSP be updated with latest checks or should out-of-tree enforcement tool be used?
  - PSP will be depreciated and documentation and guidance should be produced for [Security Standards](https://kubernetes.io/docs/concepts/security/pod-security-standards/). Implementations in out-of-tree enforcement should be favored and POC/impementation in gatekeeper would be a great way to demonstrate.
- Scheduling checks 
- Privileged containers in the same network compartment as the non-privileged pod, and otherwise init privileged containers may be able to still access the host network
