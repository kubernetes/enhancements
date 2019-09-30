---
title: Supporting CRI-ContainerD on Windows
authors:
  - "@patricklang"
owning-sig: sig-windows
participating-sigs:
  - sig-windows
reviewers:
  - "@yujuhong"
  - "@derekwaynecarr"
  - "@tallclair"
approvers:
  - "@michmike"
editor: TBD
creation-date: 2019-04-24
last-updated: 2019-09-20
status: implementable
---

# Supporting CRI-ContainerD on Windows

## Table of Contents

<!-- TOC -->

- [Supporting CRI-ContainerD on Windows](#supporting-cri-containerd-on-windows)
    - [Table of Contents](#table-of-contents)
    - [Release Signoff Checklist](#release-signoff-checklist)
    - [Summary](#summary)
    - [Motivation](#motivation)
        - [Goals](#goals)
        - [Non-Goals](#non-goals)
    - [Proposal](#proposal)
        - [User Stories](#user-stories)
            - [Improving Kubernetes integration for Windows Server containers](#improving-kubernetes-integration-for-windows-server-containers)
            - [Improved isolation and compatibility between Windows pods using Hyper-V](#improved-isolation-and-compatibility-between-windows-pods-using-hyper-v)
            - [Improve Control over Memory & CPU Resources with Hyper-V](#improve-control-over-memory--cpu-resources-with-hyper-v)
            - [Improved Storage Control with Hyper-V](#improved-storage-control-with-hyper-v)
            - [Enable runtime resizing of container resources](#enable-runtime-resizing-of-container-resources)
        - [Implementation Details/Notes/Constraints](#implementation-detailsnotesconstraints)
            - [Options for supporting multiple OS versions](#options-for-supporting-multiple-os-versions)
                - [Option A - Use RuntimeClass per Windows OS version](#option-a---use-runtimeclass-per-windows-os-version)
                    - [Migrating to Hyper-V from per-version RuntimeClass](#migrating-to-hyper-v-from-per-version-runtimeclass)
                - [Option B: Use RuntimeClass only with ContainerD](#option-b-use-runtimeclass-only-with-containerd)
                - [Option C: Support multiarch os/arch/version in CRI](#option-c-support-multiarch-osarchversion-in-cri)
                    - [Details of Multi-arch images](#details-of-multi-arch-images)
                    - [Proposed changes to CRI to support multi-arch](#proposed-changes-to-cri-to-support-multi-arch)
            - [Proposal: Standardize hypervisor annotations](#proposal-standardize-hypervisor-annotations)
        - [Dependencies](#dependencies)
                - [Windows Server 2019](#windows-server-2019)
                - [CRI-ContainerD](#cri-containerd)
                - [CNI: Flannel](#cni-flannel)
                - [CNI: Kubenet](#cni-kubenet)
                - [CNI: GCE](#cni-gce)
                - [Storage: in-tree AzureFile, AzureDisk, Google PD](#storage-in-tree-azurefile-azuredisk-google-pd)
                - [Storage: FlexVolume for iSCSI & SMB](#storage-flexvolume-for-iscsi--smb)
        - [Risks and Mitigations](#risks-and-mitigations)
            - [CRI-ContainerD availability](#cri-containerd-availability)
    - [Design Details](#design-details)
        - [Test Plan](#test-plan)
        - [Graduation Criteria](#graduation-criteria)
        - [Alpha release (proposed 1.17)](#alpha-release-proposed-117)
        - [Alpha -> Beta Graduation](#alpha---beta-graduation)
                - [Beta -> GA Graduation](#beta---ga-graduation)
        - [Upgrade / Downgrade Strategy](#upgrade--downgrade-strategy)
        - [Version Skew Strategy](#version-skew-strategy)
    - [Implementation History](#implementation-history)
    - [Alternatives](#alternatives)
        - [CRI-O](#cri-o)
    - [Infrastructure Needed](#infrastructure-needed)

<!-- /TOC -->

## Release Signoff Checklist

**ACTION REQUIRED:** In order to merge code into a release, there must be an issue in [kubernetes/enhancements] referencing this KEP and targeting a release milestone **before [Enhancement Freeze](https://github.com/kubernetes/sig-release/tree/master/releases)
of the targeted release**.

For enhancements that make changes to code or processes/procedures in core Kubernetes i.e., [kubernetes/kubernetes], we require the following Release Signoff checklist to be completed.

Check these off as they are completed for the Release Team to track. These checklist items _must_ be updated for the enhancement to be released.

- [x] kubernetes/enhancements issue in release milestone, which links to KEP (this should be a link to the KEP location in kubernetes/enhancements, not the initial KEP PR)
- [x] KEP approvers have set the KEP status to `implementable`
- [x] Design details are appropriately documented
- [x] Test plan is in place, giving consideration to SIG Architecture and SIG Testing input
- [x] Graduation criteria is in place
- [ ] "Implementation History" section is up-to-date for milestone
- [ ] User-facing documentation has been created in [kubernetes/website], for publication to [kubernetes.io]
- [ ] Supporting documentation e.g., additional design documents, links to mailing list discussions/SIG meetings, relevant PRs/issues, release notes

**Note:** Any PRs to move a KEP to `implementable` or significant changes once it is marked `implementable` should be approved by each of the KEP approvers. If any of those approvers is no longer appropriate than changes to that list should be approved by the remaining approvers and/or the owning SIG (or SIG-arch for cross cutting KEPs).

**Note:** This checklist is iterative and should be reviewed and updated every time this enhancement is being considered for a milestone.

[kubernetes.io]: https://kubernetes.io/
[kubernetes/enhancements]: https://github.com/kubernetes/enhancements/issues
[kubernetes/kubernetes]: https://github.com/kubernetes/kubernetes
[kubernetes/website]: https://github.com/kubernetes/website

## Summary

The ContainerD maintainers have been working on CRI support which is stable on Linux, but is not yet available for Windows as of ContainerD 1.2. Currently it’s planned for ContainerD 1.3, and the developers in the Windows container platform team have most of the key work merged into master already. Supporting CRI-ContainerD on Windows means users will be able to take advantage of the latest container platform improvements that shipped in Windows Server 2019 / 1809 and beyond.


## Motivation

Windows Server 2019 includes an updated host container service (HCS v2) that offers more control over how containers are managed. This can remove some limitations and improve some Kubernetes API compatibility. However, the current Docker EE 18.09 release has not been updated to work with the Windows HCSv2, only ContainerD has been migrated. Moving to CRI-ContainerD allows the Windows OS team and Kubernetes developers to focus on an interface designed to work with Kubernetes to improve compatibility and accelerate development.

Additionally, users could choose to run with only CRI-ContainerD instead of Docker EE if they wanted to reduce the install footprint or produce their own self-supported CRI-ContainerD builds.

### Goals

- Improve the matrix of Kubernetes features that can be supported on Windows
- Provide a path forward to implement Kubernetes-specific features that are not available in the Docker API today
- Align with `dockershim` deprecation timelines once they are defined

### Non-Goals

- Running Linux containers on Windows nodes. This would be addressed as a separate KEP since the use cases are different.
- Deprecating `dockershim`. This is out of scope for this KEP. The effort to migrate that code out of tree is in [KEP PR 866](https://github.com/kubernetes/enhancements/pull/866) and deprecation discussions will happen later.

## Proposal

### User Stories

#### Improving Kubernetes integration for Windows Server containers

Moving to the new Windows HCSv2 platform and ContainerD would allow Kubernetes to add support for:

- Mounting single files, not just folders, into containers
- Termination messages (depends on single file mounts)
- /etc/hosts (c:\windows\system32\drivers\etc\hosts) file mapping

#### Improved isolation and compatibility between Windows pods using Hyper-V 

Hyper-V enables each pod to run within it’s own hypervisor partition, with a separate kernel. This means that we can build forward-compatibility for containers across Windows OS versions - for example a container built using Windows Server 1809, could be run on a node running Windows Server 1903. This pod would use the Windows Server 1809 kernel to preserve full compatibility, and other pods could run using either a shared kernel with the node, or their own isolated Windows Server 1903 kernels. Containers requiring 1809 and 1903 (or later) cannot be mixed in the same pod, they must be deployed in separate pods so the matching kernel may be used. Running Windows Server version 1903 containers on a Windows Server 2019/1809 host will not work.

In addition, some customers may desire hypervisor-based isolation as an additional line of defense against a container break-out attack.

Adding Hyper-V support would use [RuntimeClass](https://kubernetes.io/docs/concepts/containers/runtime-class/#runtime-class). 
3 typical RuntimeClass names would be configured in CRI-ContainerD to support common deployments:
- runhcs-wcow-process [default] - process isolation is used, container & node OS version must match
- runhcs-wcow-hypervisor - Hyper-V isolation is used, Pod will be compatible with containers built with Windows Server 2019 / 1809. Physical memory overcommit is allowed with overages filled from pagefile.
- runhcs-wcow-hypervisor-1903 - Hyper-V isolation is used, Pod will be compatible with containers built with Windows Server 1903. Physical memory overcommit is allowed with overages filled from pagefile.

Using Hyper-V isolation does require some extra memory for the isolated kernel & system processes. This could be accounted for by implementing the [PodOverhead](https://kubernetes.io/docs/concepts/containers/runtime-class/#runtime-class) proposal for those runtime classes. We would include a recommended PodOverhead in the default CRDs, likely between 100-200M.


#### Improve Control over Memory & CPU Resources with Hyper-V

The Windows kernel itself cannot provide reserved memory for pods, containers or processes. They are always fulfilled using virtual allocations which could be paged out later. However, using a Hyper-V partition improves control over memory and CPU cores. Hyper-V can either allocate memory on-demand (while still enforcing a hard limit), or it can be reserved as a physical allocation up front. Physical allocations may be able to enable large page allocations within that range (to be confirmed) and improve cache coherency. CPU core counts may also be limited so a pod only has certain cores available, rather than shares spread across all cores, and applications can tune thread counts to the actually available cores.

Operators could deploy additional RuntimeClasses with more granular control for performance critical workloads:
- 2019-Hyper-V-Reserve: Hyper-V isolation is used, Pod will be compatible with containers built with Windows Server 2019 / 1809. Memory reserve == limit, and is guaranteed to not page out.
  - 2019-Hyper-V-Reserve-<N>Core: Same as above, except all but <N> CPU cores are masked out.
- 1903-Hyper-V-Reserve: Hyper-V isolation is used, Pod will be compatible with containers built with Windows Server 1903. Memory reserve == limit, and is guaranteed to not page out.
  - 1903-Hyper-V-Reserve-<N>Core: Same as above, except all but <N> CPU cores are masked out.


#### Improved Storage Control with Hyper-V


Hyper-V also brings the capability to attach storage to pods using block-based protocols (SCSI) instead of file-based protocols (host file mapping / NFS / SMB). These capabilities could be enabled in HCSv2 with CRI-ContainerD, so this could be an area of future work. Some examples could include:

Attaching a "physical disk" (such as a local SSD, iSCSI target, Azure Disk or Google Persistent Disk) directly to a pod. The kubelet would need to identify the disk beforehand, then attach it as the pod is created with CRI. It could then be formatted and used within the pod without being mounted or accessible on the host.

Creating [Persistent Local Volumes](https://kubernetes.io/docs/concepts/storage/volumes/#local) using a local virtual disk attached directly to a pod. This would create local, non-resilient storage that could be formatted from the pod without being mounted on the host. This could be used to build out more resource controls such as fixed disk sizes and QoS based on IOPs or throughput and take advantage of high speed local storage such as temporary SSDs offered by cloud providers.


#### Enable runtime resizing of container resources

With virtual-based allocations and Hyper-V, it should be possible to increase the limit for a running pod. This won’t give it a guaranteed allocation, but will allow it to grow without terminating and scheduling a new pod. This could be a path to vertical pod autoscaling. This still needs more investigation and is mentioned as a future possibility.


### Implementation Details/Notes/Constraints

The work needed will span multiple repos, SIG-Windows will be maintaining a [Windows CRI-Containerd Project Board] to track everything in one place.


#### Options for supporting multiple OS versions




##### Option A - Use RuntimeClass per Windows OS version

With process isolation, Windows containers must be constrained to only run on a matching Windows Server version node.

[RuntimeClass Scheduling] has a few useful features that can make it easier to match a Windows container to a compatible Windows node. Here's an example of steps that could use RuntimeClass to avoid os version mismatches: 

1. A RuntimeClass such as `WindowsServer1809` defined with [RuntimeClass Scheduling] `nodeSelector` for `kubernetes.io/os=windows` and `nodeSelector` for `beta.kubernetes.io/osversion=Windows1809` set.
1. Nodes must be labeled with the `kubernetes.io/os=windows` (already done), as well as the additional label `beta.kubernetes.io/osversion=Windows1809`

This would need to be repeated for each Windows version added to the cluster such as `WindowsServer1809`, `WindowsServer1903` and future releases. The cluster operator would need to ensure that nodes are added and scaled as needed for each Windows version independently.

A user doing a deployment must set an appropriate `runtimeClassName` for their Pod to be scheduled on a compatible Windows node.

```
apiVersion: v1
kind: Pod
metadata:
  name: mypod
spec:
  runtimeClassName: WindowsServer1809
  # ...
```

###### Migrating to Hyper-V from per-version RuntimeClass

Once Hyper-V isolation is supported though, this causes a problem. There may be two different nodes that could run a given container built using Windows Server 1809:

1. One running `Windows Server version 1809` supporting process isolation with `dockershim` or `containerd`
    * Labels
      * `kubernetes.io/os` = `windows`
      * `beta.kubernetes.io/osversion` = `Windows1809`
1. One running `Windows Server version 1903` supporting Hyper-V isolation with `containerd`
    * Labels
      * `kubernetes.io/os` = `windows`
      * `beta.kubernetes.io/osversion` = `Windows1903`
      * `capability.microsoft.com/isolation` = `hyperv`

Using the `nodeSelector` above will only match node #1. Scheduling to node #2 would require defining a separate runtime class, and changing the `Pod` spec `runtimeClassName`.

Migrating off an old Windows Server version would require a multi-step manual process.

1. Set up an admission controller (for example [Gatekeeper]) to prevent new pods from being scheduled using the old `runtimeClassName`
1. Find existing pods (for example - [Gatekeeper] audit rules) using the old `runtimeClassName`, and update them.
1. `kubectl cordon` the old Windows nodes
1. Decommission the old Windows nodes once no pods are left

##### Option B: Use RuntimeClass only with ContainerD

Instead of using RuntimeClass to support multiple Windows Server versions in the same cluster with `process` isolation, it could be used only to enable new features.

Using the same two nodes from above:

1. One running `Windows Server version 1809` supporting process isolation with `dockershim` or `containerd`
    * Labels
      * `kubernetes.io/os` = `windows`
      * `beta.kubernetes.io/osversion` = `Windows1809`
1. One running `Windows Server version 1903` supporting Hyper-V isolation with `containerd`
    * Labels
      * `kubernetes.io/os` = `windows`
      * `beta.kubernetes.io/osversion` = `Windows1903`
      * `capability.microsoft.com/isolation` = `hyperv`

A new RuntimeClass could be created to opt-in to `hyperv` isolation for existing workloads:

```
apiVersion: node.k8s.io/v1beta1
kind: RuntimeClass
metadata:
  name: windows-1809-hyperv
handler: runhcs-wcow-hypervisor-1809  # The name of the corresponding CRI configuration
nodeSelector:
  kubernetes.io/os = 'windows'
  beta.kubernetes.io/osversion = 'Windows1903'
  capability.microsoft.com/isolation = 'hyperv'
```

The new version can be added with a new RuntimeClass

```
apiVersion: node.k8s.io/v1beta1
kind: RuntimeClass
metadata:
  name: windows-1903-hyperv
handler: runhcs-wcow-hypervisor-1903  # The name of the corresponding CRI configuration
nodeSelector:
  kubernetes.io/os = 'windows'
  beta.kubernetes.io/osversion = 'Windows1903'
  capability.microsoft.com/isolation = 'hyperv'
```

Pods would need to be updated to migrate to the new nodes using `hyperv` isolation.

```
apiVersion: v1
kind: Pod
metadata:
  name: mypod
spec:
  runtimeClassName: windows-1809-hyperv
  # ...
```

The same approach of block with [Gatekeeper], audit existing workloads, update deployments, cordon then remove old nodes as in Option A could be followed.

Once the first Hyper-V migration is done, future Windows versions could be used by just updating `RuntimeClass.NodeSelector` to use the new OS version. Cordon the old nodes, wait for the pods to move, then delete the old nodes.


##### Option C: Support multiarch os/arch/version in CRI

The Open Container Initiative specifications for container runtime support specifying the architecture, os, and version when pulling and starting a container. This is important for Windows because there is no kernel compatibility between major versions. In order to successfully start a container with process isolation, the right `os.version` must be pulled and run. Hyper-V can provide backwards compatibility, but the image pull and sandbox creation need to specify `os.version` because the kernel is brought up when the sandbox is created. The same challenges exist for Linux as well because multiple CPU architectures can be supported - for example armv7 with qemu and binfmt_misc.

One way to make the experience uniform for dealing with multi-arch images is to add new optional fields to force a deployment to use a specific os/version/arch. This may be combined with RuntimeClass to simplify node placement if needed.

```
apiVersion: v1
kind: Pod
metadata:
  name: mypod
spec:
  os: windows
  osversion: 1809
  architecture: amd64
  runtimeClassName: windows-hyperv
  containers:
      - name: iis
        image: mcr.microsoft.com/windows/servercore/iis:windowsservercore-ltsc2019
```


###### Details of Multi-arch images

Here's one example of a container image that has a multi-arch manifest with entries for Linux & Windows, multiple CPU architectures, and multiple Windows os versions.

```powershell
(docker manifest inspect mcr.microsoft.com/dotnet/core/sdk:2.1 | ConvertFrom-Json)[0].manifests | Format-Table digest, platform

digest                                                                  platform
------                                                                  --------
sha256:f04192a946a4473d0001ed9c9422a9e3c9c659de8498778d29bfe6c98672ad9f @{architecture=amd64; os=linux}
sha256:b7da46cdbc9a0c0ed154cabbb0591dea596e67d62d84c2a3ef34aaefe98f186d @{architecture=arm; os=linux; variant=v7}
sha256:b73b7b5defbab6beddcf6e9289b25dd37c99c5c79415bf78a8b47f92350fb09b @{architecture=amd64; os=windows; os.version=10.0.17134.1006}
sha256:573625a22a90e684c655f1eed7b0e4f03fbe90a4e94907f1f960f73a9e3092f5 @{architecture=amd64; os=windows; os.version=10.0.17763.737}
sha256:2c0d528344dff960540f500b44ed1c60840138aa60e01927620df59bd63a9dfc @{architecture=amd64; os=windows; os.version=10.0.18362.356}
```

###### Proposed changes to CRI to support multi-arch

Here's the steps needed to pull and start a pod+container matching a specific os/arch/version:

- ImageService.PullImage(PullImageRequest) - PullImageRequest includes a `PodSandboxConfig` reference
- RuntimeService.RunPodSandbox(RunPodSandboxRequest) - RunPodSandboxRequest includes a `PodSandboxConfig` reference
- RuntimeService.CreateContainer(CreateContainerRequest - the same `PodSandboxConfig` is passed as in the previous step

All of these use the same `PodSandboxConfig`, so we're proposing adding os/arch/version there.

From https://github.com/kubernetes/cri-api/blob/24ae4d4e8b036b885ee1f4930ec2b173eabb28e7/pkg/apis/runtime/v1alpha2/api.proto#L310

```
message PodSandboxConfig {
    // Metadata of the sandbox. This information will uniquely identify the
    // sandbox, and the runtime should leverage this to ensure correct
    // operation. The runtime may also use this information to improve UX, such
    // as by constructing a readable name.
    PodSandboxMetadata metadata = 1;
    // Hostname of the sandbox. Hostname could only be empty when the pod
    // network namespace is NODE.
    string hostname = 2;
    // Path to the directory on the host in which container log files are
    // stored.
    // By default the log of a container going into the LogDirectory will be
    // hooked up to STDOUT and STDERR. However, the LogDirectory may contain
    // binary log files with structured logging data from the individual
    // containers. For example, the files might be newline separated JSON
    // structured logs, systemd-journald journal files, gRPC trace files, etc.
    // E.g.,
    //     PodSandboxConfig.LogDirectory = `/var/log/pods/<podUID>/`
    //     ContainerConfig.LogPath = `containerName/Instance#.log`
    //
    // WARNING: Log management and how kubelet should interface with the
    // container logs are under active discussion in
    // https://issues.k8s.io/24677. There *may* be future change of direction
    // for logging as the discussion carries on.
    string log_directory = 3;
    // DNS config for the sandbox.
    DNSConfig dns_config = 4;
    // Port mappings for the sandbox.
    repeated PortMapping port_mappings = 5;
    // Key-value pairs that may be used to scope and select individual resources.
    map<string, string> labels = 6;
    // Unstructured key-value map that may be set by the kubelet to store and
    // retrieve arbitrary metadata. This will include any annotations set on a
    // pod through the Kubernetes API.
    //
    // Annotations MUST NOT be altered by the runtime; the annotations stored
    // here MUST be returned in the PodSandboxStatus associated with the pod
    // this PodSandboxConfig creates.
    //
    // In general, in order to preserve a well-defined interface between the
    // kubelet and the container runtime, annotations SHOULD NOT influence
    // runtime behaviour.
    //
    // Annotations can also be useful for runtime authors to experiment with
    // new features that are opaque to the Kubernetes APIs (both user-facing
    // and the CRI). Whenever possible, however, runtime authors SHOULD
    // consider proposing new typed fields for any new features instead.
    map<string, string> annotations = 7;
    // Optional configurations specific to Linux hosts.
    LinuxPodSandboxConfig linux = 8;
}
```

Today, PodSandboxConfig has an annotation that can be used as a workaround, but it's stated that "Whenever possible, however, runtime authors SHOULD consider proposing new typed fields for any new features instead.".

The proposed additions are:

```
string os = 9;
string architecture = 10;
string version = 11;
```

These correspond to `platform.os`, `platform.architecture`, and `platform.os.version` as describe in the [OCI image spec](https://github.com/opencontainers/image-spec/blob/master/image-index.md)



#### Proposal: Standardize hypervisor annotations

There are large number of [Windows annotations](https://github.com/Microsoft/hcsshim/blob/master/internal/oci/uvm.go#L15) defined that can control how Hyper-V will configure its hypervisor partition for the pod. Today, these could be set in the runtimeclasses defined in the CRI-ContainerD configuration file on the node, but it would be easier to maintain them if key settings around resources (cpu+memory+storage) could be aligned across multiple hypervisors and exposed in CRI.

Doing this would make pod definitions more portable between different isolation types. It would also avoid the need for a "t-shirt size" list of RuntimeClass instances to choose from:
- 1809-Hyper-V-Reserve-2Core-PhysicalMemory
- 1903-Hyper-V-Reserve-1Core-VirtualMemory
- 1903-Hyper-V-Reserve-4Core-PhysicalMemory
- ...




### Dependencies

##### Windows Server 2019

This work would be carried out and tested using the already-released Windows Server 2019. That will enable customers a migration path from Docker 18.09 to CRI-ContainerD if they want to get this new functionality. Windows Server 1903 and later will also be supported once they’re tested.

##### CRI-ContainerD

It was announced that the upcoming 1.3 release would include Windows support, but that release and timeline are still in planning as of early April 2019.

The code needed to run ContainerD is merged, and [experimental support in moby](https://github.com/moby/moby/pull/38541) has merged. CRI is in the process of being updated, and open issues are tracked on the [Windows CRI-Containerd Project Board]

The CRI plugin changes needed to enable Hyper-V isolation are still in a development branch [jterry75/cri](https://github.com/jterry75/cri/tree/windows_port/cmd/containerd) and don’t have an upstream PR open yet.

Code: mostly done
CI+CD: lacking

##### CNI: Flannel 
Flannel isn’t expected to require any changes since the Windows-specific metaplugins ship outside of the main repo. However, there is still not a stable release supporting Windows so it needs to be built from source. Additionally, the Windows-specific metaplugins to support ContainerD are being developed in a new repo [Microsoft/windows-container-networking](https://github.com/Microsoft/windows-container-networking). It’s still TBD whether this code will be merged into [containernetworking/plugins](https://github.com/containernetworking/plugins/), or maintained in a separate repo.
- Sdnbridge - this works with host-gw mode, replaces win-bridge
- Sdnoverlay - this works with vxlan overlay mode, replaces win-overlay

Code: in progress
CI+CD: lacking

##### CNI: Kubenet

The same sdnbridge plugin should work with kubenet as well. If someone would like to use kubenet instead of flannel, that should be feasible.

##### CNI: GCE

GCE uses the win-bridge meta-plugin today for managing Windows network interfaces. This would also need to migrate to sdnbridge.

##### Storage: in-tree AzureFile, AzureDisk, Google PD

These are expected to work and the same tests will be run for both dockershim and CRI-ContainerD.

##### Storage: FlexVolume for iSCSI & SMB
These out-of-tree plugins are expected to work, and are not tested in prow jobs today. If they graduate to stable we’ll add them to testgrid.

### Risks and Mitigations

#### CRI-ContainerD availability

Builds are not yet available, but there is a [ContainerD tracking issue] for support Windows. Tests are currently running in a [prow job with CRITest](https://k8s-testgrid.appspot.com/sig-node-containerd#cri-validation-windows) against the master branch. We will publish the setup steps required to build & test in the kubernetes-sigs/windows-testing repo during the course of alpha so testing can commence.

## Design Details

### Test Plan

The existing test cases running on Testgrid that cover Windows Server 2019 with Docker will be reused with CRI-ContainerD. Testgrid will include results for both ContainerD and dockershim.

- TestGrid: SIG-Windows: [flannel-l2bridge-windows-master](https://testgrid.k8s.io/sig-windows#flannel-l2bridge-windows-master) - this uses dockershim
- TestGrid: SIG-Windows: [containerd-l2bridge-windows-master](https://testgrid.k8s.io/sig-windows#containerd-l2bridge-windows-master) - this uses ContainerD

Test cases that depend on ContainerD and won't pass with Dockershim will be marked with `[feature:windows-containerd]` until `dockershim` is deprecated.

### Graduation Criteria

### Alpha release (proposed 1.17)

- Windows Server 2019 containers can run with process level isolation
- TestGrid has results for Kubernetes master branch. CRI-ContainerD and CNI built from source and may include non-upstream PRs.
- Support RuntimeClass to enable Hyper-V isolation for Windows Server 2019 on 2019


### Alpha -> Beta Graduation

- Feature parity with dockershim, including:
  - Group Managed Service Account support
  - Named pipe & Unix domain socket mounts
- Support RuntimeClass to enable Hyper-V isolation and run Windows Server 2019 containers on Windows Server 1903
- Publically available builds (beta or better) of CRI-ContainerD, at least one CNI
- TestGrid results for above builds with Kubernetes master branch


##### Beta -> GA Graduation

- Stable release of CRI-ContainerD on Windows, at least one CNI
- Master & release branches on TestGrid

### Upgrade / Downgrade Strategy

Because no Kubernetes API changes are expected, there is no planned upgrade/downgrade testing at the cluster level.

Node upgrade/downgrade is currently out of scope of the Kubernetes project, but we'll aim to include CRI-ContainerD in other efforts such as `kubeadm` bootstrapping for nodes.

As discussed in SIG-Node, there's also no testing on switching CRI on an existing node. These are expected to be installed and configured as a prerequisite before joining a node to the cluster.

### Version Skew Strategy

There's no version skew considerations needed for the same reasons described in upgrade/downgrade strategy.

## Implementation History

- 2019-04-24 - KEP started, based on the [earlier doc shared SIG-Windows and SIG-Node](https://docs.google.com/document/d/1NigFz1nxI9XOi6sGblp_1m-rG9Ne6ELUrNO0V_TJqhI/edit)
- 2019-09-20 - Updated with ne wmilestones

## Alternatives

### CRI-O

[CRI-O](https://cri-o.io/) is another runtime that aims to closely support all the fields available in the CRI spec. Currently there aren't any maintainers porting it to Windows so it's not a viable alternative.

## Infrastructure Needed

No new infrastructure is currently needed from the Kubernetes community. The existing test jobs using prow & testgrid will be copied and modified to test CRI-ContainerD in addition to dockershim.


[RuntimeClass Scheduling]: https://kubernetes.io/docs/concepts/containers/runtime-class/#scheduling
[Windows CRI-Containerd Project Board]: https://github.com/orgs/kubernetes/projects/34
[ContainerD tracking issue]: https://github.com/containerd/cri/issues/1257
[Gatekeeper]: https://github.com/open-policy-agent/gatekeeper