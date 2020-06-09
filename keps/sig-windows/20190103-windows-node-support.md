---
title: Windows node support
authors:
  - "@astrieanna"
  - "@benmoss"
  - "@patricklang"
  - "@michmike"
  - "@daschott"
owning-sig: sig-windows
participating-sigs:
  - sig-architecture
  - sig-node
reviewers:
  - sig-architecture
  - sig-node
  - sig-testing
  - sig-release
approvers:
  - "@bgrant0607"
  - "@michmike"
  - "@patricklang"
  - "@spiffxp"
editor: TBD
creation-date: 2018-11-29
last-updated: 2019-03-06
status: implemented
---

# Windows node support


## Table of Contents

<!-- toc -->
- [Summary](#summary)
- [Motivation](#motivation)
  - [Goals](#goals)
  - [Non-Goals](#non-goals)
- [Proposal](#proposal)
  - [What works today](#what-works-today)
  - [Windows Node Roadmap (post-GA work)](#windows-node-roadmap-post-ga-work)
    - [Custom DNS updates for CNI plugins](#custom-dns-updates-for-cni-plugins)
  - [What will never work](#what-will-never-work)
  - [Windows Container Compatibility](#windows-container-compatibility)
  - [Relevant resources/conversations](#relevant-resourcesconversations)
  - [Risks and Mitigations](#risks-and-mitigations)
    - [Ensuring OS-specific workloads land on appropriate container host](#ensuring-os-specific-workloads-land-on-appropriate-container-host)
    - [Memory Overprovisioning](#memory-overprovisioning)
- [Graduation Criteria](#graduation-criteria)
- [Implementation History](#implementation-history)
- [Testing Plan](#testing-plan)
  - [Test Dashboard](#test-dashboard)
  - [Test Environment](#test-environment)
  - [Test Approach](#test-approach)
    - [Adapting existing tests](#adapting-existing-tests)
    - [Substitute test cases](#substitute-test-cases)
    - [Windows specific tests](#windows-specific-tests)
- [Conformance Testing](#conformance-testing)
- [API Reference](#api-reference)
  - [V1.Container](#v1container)
  - [V1.Pod](#v1pod)
  - [V1.PodSecurityContext](#v1podsecuritycontext)
- [Other references](#other-references)
<!-- /toc -->

## Summary

There is strong interest in the community for adding support for workloads running on Microsoft Windows. This is non-trivial due to the significant differences in the implementation of Windows from the Linux-based OSes that have so far been supported by Kubernetes. This KEP will allow Windows nodes to be added to a Kubernetes cluster as compute nodes. With the introduction of Windows nodes, developers will be able to schedule Windows Server containers and run Windows-based applications on Kubernetes.


## Motivation

Windows-based workloads still account for a significant portion of the enterprise software space. While containerization technologies emerged first in the UNIX ecosystem, Microsoft has made investments in recent years to enable support for containers in its Windows OS. As users of Windows increasingly turn to containers as the preferred abstraction for running software and modernizing existing applications, the Kubernetes ecosystem stands to benefit by becoming a cross-platform cluster manager.

### Goals

- Enable users to schedule Windows Server containers in Kubernetes through the introduction of support for Windows compute nodes
- Document the differences and limitations compared to Linux
- Create a test suite in testgrid to maintain high quality for this feature and prevent regression of functionality 

### Non-Goals

- Adding Windows support to all projects in the Kubernetes ecosystem (Cluster Lifecycle, etc)
- Enable the Kubernetes master components to run on Windows
- Support for LCOW (Linux Containers on Windows with Hyper-V Isolation)

## Proposal

As of 29-11-2018 much of the work for enabling Windows nodes has already been completed. Both `kubelet` and `kube-proxy` have been adapted to work on Windows Server, and so the first goal of this KEP is largely already complete. 

### What works today
- Windows-based containers can be created by kubelet, [provided the host OS version matches the container base image](https://docs.microsoft.com/en-us/virtualization/windowscontainers/deploy-containers/version-compatibility). Microsoft will distribute the operating system-dependent `pause` image (mcr.microsoft.com/k8s/core/pause:1.0.0).
    - Pod
      - Single or multiple containers per Pod with process isolation
      - There are no notable differences in Pod status fields between Linux and Windows containers
      - Readiness and Liveness probes
      - postStart & preStop container lifecycle events
      - ConfigMap, Secrets: as environment variables or volumes (Volume subpath does not work)
      - EmptyDir
      - Named pipe host mounts
      - Volumes can be shared between containers in a Pod
      - Resource limits
    - Services types NodePort, ClusterIP, LoadBalancer, and ExternalName. Service environment variables and headless services work.
      - Cross operating system service connectivity
    - Workload controllers ReplicaSet, ReplicationController, Deployments, StatefulSets, DaemonSet, Job, CronJob
    - Scheduler preemption
    - Pod & container metrics
    - Horizontal Pod Autoscaling using all metrics
    - KubeCtl Exec
    - Resource Quotas 
- Windows Server 2019 is the only Windows operating system we will support at GA timeframe. Note above that the host operating system version and the container base image need to match. This is a Windows limitation we cannot overcome.
- Customers can deploy a heterogeneous cluster, with Windows and Linux compute nodes side-by-side and schedule Docker containers on both operating systems. Of course, Windows Server containers have to be scheduled on Windows and Linux containers on Linux
- Out-of-tree Pod networking with [Azure-CNI](https://github.com/Azure/azure-container-networking/blob/master/docs/cni.md), [OVN-Kubernetes](https://github.com/openvswitch/ovn-kubernetes), [two CNI meta-plugins](https://github.com/containernetworking/plugins), [Flannel (VXLAN and Host-Gateway)](https://github.com/coreos/flannel) 
- Dockershim CRI
- Many<sup id="a1">[1]</sup> of the e2e conformance tests when run with [alternate Windows-based images](https://hub.docker.com/r/e2eteam/) which are being moved to [kubernetes-sigs/windows-testing](https://www.github.com/kubernetes-sigs/windows-testing)
- Persistent storage: FlexVolume with [SMB + iSCSI](https://github.com/Microsoft/K8s-Storage-Plugins/tree/master/flexvolume/windows), and in-tree AzureFile and AzureDisk providers
- Kube-Proxy support for L2Bridge and Overlay networks

### Windows Node Roadmap (post-GA work)
- Group Managed Service Accounts, a way to assign an Active Directory identity to a Windows container, is forthcoming with KEP `Windows Group Managed Service Accounts for Container Identity`. This work will be released as alpha in v1.14 and is already merged.
- `kubectl port-forward` hasn't been implemented due to lack of an `nsenter` equivalent to run a process inside a network namespace.
- CRIs other than Dockershim: CRI-containerd support is forthcoming
- Some kubeadm work was done in the past to add Windows nodes to Kubernetes, but that effort has been dormant since. We will need to revisit that work and complete it in the future.
- Calico CNI for Pod networking
- Hyper-V isolation (Currently this is limited to 1 container per Pod and is an alpha feature)
  - This could enable backwards compatibility likely as a RuntimeClass. This would allow running a container host OS that is newer in version than the container OS (visit this link for additional compatibility definitions https://docs.microsoft.com/en-us/virtualization/windowscontainers/deploy-containers/version-compatibility)
- It is unclear if the RuntimeClass proposal from sig-node will simplify scheduled Windows containers. We will work with sig-node on this.
- Properly implement terminationGracePeriodSeconds for Windows (https://github.com/moby/moby/issues/25982 and https://github.com/kubernetes/kubernetes/issues/73434)
- Single file mapping and Termination message will work when we introduce CRI containerD support in Windows
- Design and implement `--enforce-node-allocatable`, hard/soft eviction and `MemoryPressure` conditions. These all depend on cgroups in Linux, and the kubelet will need new work specific to Windows to raise and respond to memory pressure conditions. See [Memory Overprovisioning](#memory-overprovisioning) later in this doc.
- Fix run_as_username for Windows (https://github.com/kubernetes/kubernetes/issues/73387)
- Support for Local Traffic Policy and DSR mode on Windows (https://github.com/kubernetes/kubernetes/issues/62046)



#### Custom DNS updates for CNI plugins

For v1.14, custom pod DNS configuration tests are not running. Some CNI implementations updates are needed to Azure-CNI, win-bridge, OVN, and flannel which are out of the kubernetes/kubernetes tree. Once those are updated, the tests are tracked in [issue 73414](https://github.com/kubernetes/kubernetes/issues/73414)/[pr 74925](https://github.com/kubernetes/kubernetes/pull/74925) will be merged.

As part of Azure-CNI [PR#305](https://github.com/Azure/azure-container-networking/pull/305), manual tests were run with Pod.Spec.DNSPolicy = DNSNone. Hostname, Subdomain, and DNSConfig.Nameservers, and DNSConfig.Searches were set correctly based on the Pod spec.

Tracking Issues:

- win-bridge [#271](https://github.com/containernetworking/plugins/pull/271) - this is also used in the test passes for GCE, see [gce-k8s-windows-testing#7](https://github.com/yujuhong/gce-k8s-windows-testing/pull/7)
- Azure-CNI [PR#305](https://github.com/Azure/azure-container-networking/pull/305)

### What will never work
Note that some features are plain unsupported while some will not work without underlying OS changes
- Certain Pod functionality
    - Privileged containers
    - Pod security context privilege and access control settings. Any Linux Capabilities, SELinux, AppArmor, Seccomp, Capabilities (POSIX Capabilities), and others are not supported 
    - Reservations are not enforced by the OS, but overprovisioning could be blocked with `--enforce-node-allocatable=pods` (pending: tests needed)
    - Certain volume mappings
      - Subpath volume mounting
      - Subpath volume mounting for Secrets
      - Host mount projection
      - DefaultMode (due to UID/GID dependency)
      - readOnly root filesystem. Mapped volumes still support readOnly
      - Block device mapping
    - Expanding the mounted volume (resizefs)
    - HugePages
    - Memory as the storage medium
- CSI plugins, which require privileged containers
- File system features like uui/guid, per-user Linux filesystem permissions, and read-only root filesystems (see note above and also later in the doc about read-only volumes)
- NFS based storage/volume support (https://github.com/kubernetes/kubernetes/issues/56188)
- Host networking is not available in Windows
- ClusterFirstWithHostNet is not supported for DNS. Windows treats all names with a `.` as a FQDN and skips PQDN resolution
- Not all features of shared namespaces are supported. This is clarified in the API section below
- The existing node problem detector is Linux-only and requires privileged containers. In general, we don't expect these to be used on Windows because there's no privileged support
- Overlay networking support in Windows Server 1803 is not fully functional using the `win-overlay` CNI plugin. Specifically service IPs do not work on Windows nodes. This is currently specific to `win-overlay`; other CNI plugins (OVS, AzureCNI) work. Since Windows Server 1803 is not supported for GA, this is mostly not applicable. We left it here since it impacts beta
- Outbound communication using the ICMP protocol via the `win-overlay`, `win-bridge`, and `Azure-CNI` plugin. Specifically, the Windows data plane ([VFP](https://www.microsoft.com/en-us/research/project/azure-virtual-filtering-platform/)) doesn't support ICMP packet transpositions. This means:
    - ICMP packets directed to destinations within the same network (e.g. pod to pod communication via ping) will work as expected and without any limitations
    - TCP/UDP packets will work as expected and without any limitations
    - ICMP packets directed to pass through a remote network (e.g. pod to external internet communication via ping) cannot be transposed and thus will *not* be routed back to their source
      - Since TCP/UDP packets can still be transposed, one can substitute `ping <destination>` with `curl <destination>` to be able to debug connectivity to the outside world.

### Windows Container Compatibility
As noted above, there are compatibility issues enforced by Microsoft where the host OS version must match the container base image OS. Changes to this compatibility policy must come from Microsoft. For GA, since we will only support Windows Server 2019 (aka 1809), both `container host OS` and `container OS` must be running the same version of Windows, 1809. 

Having said that, a customer can deploy Kubernetes v1.14 with Windows 1809.
- We will support Windows 1809 with at least 2 additional Kuberneres minor releases (v1.15 and v.1.16)
- It is possible additional Windows releases (for example Windows 1903) will be added to the support matrix of future Kubernetes releases and they will also be supported for the next 2 versions of Kubernetes after their initial support is announced
- SIG-Windows will announce support for new Windows operating systems at most twice per year, based on Microsoft's published release cycle

Kubernetes minor releases are only supported for 9 months (https://kubernetes.io/docs/setup/version-skew-policy/), which is a smaller support interval than the support interval for Windows bi-annual releases (https://docs.microsoft.com/en-us/windows-server/get-started/windows-server-release-info) 

We don't expect all Windows customers to update the operating system for their apps twice a year. Upgrading your applications is what will dictate and necessitate upgrading or introducing new nodes to the cluster. For the customers that chose to upgrade their operating system for containers running on Kubernetes, we will offer guidance and step-by-step instructions when we add support for a new operating system version. This guidance will include recommended upgrade procedures for upgrading user applications together with cluster nodes.

Windows nodes will adhere to Kubernetes version-skew policy (node to control plane versioning) the same way as Linux nodes do today (https://kubernetes.io/docs/setup/version-skew-policy/)

### Relevant resources/conversations

- [sig-architecture thread](https://groups.google.com/forum/#!topic/kubernetes-sig-architecture/G2zKJ7QK22E)
- [cncf-k8s-conformance thread](https://lists.cncf.io/g/cncf-k8s-conformance/topic/windows_conformance_tests/27913232)
- [kubernetes/enhancements proposal](https://github.com/kubernetes/features/issues/116)


### Risks and Mitigations

**Second class support**: Kubernetes contributors are likely to be thinking of Linux-based solutions to problems, as Linux remains the primary OS supported. Keeping Windows support working will be an ongoing burden potentially limiting the pace of development. 

**User experience**: Users today will need to use some combination of taints and node selectors in order to keep Linux and Windows workloads separated. In the best case this imposes a burden only on Windows users, but this is still less than ideal. The recommended approach is outlined below, with one of its main goals being that we should not break compatibility for existing Linux workloads

#### Ensuring OS-specific workloads land on appropriate container host
As you can see below, we plan to document how Windows containers can be scheduled on the appropriate host using Taints and Tolerations. All nodes today have the following default labels (These labels will be graduating to stable soon)
- beta.kubernetes.io/os = [windows|linux]
- beta.kubernetes.io/arch = [amd64|arm64|...]

If a deployment does not specify a nodeSelector like `"beta.kubernetes.io/os": windows`, it is possible the Pods can be scheduled on any host, Windows or Linux. This can be problematic since a Windows container can only run on Windows and a Linux container can only run on Linux. The best practice we will recommend is to use a nodeSelector. 

However, we understand that in many cases customers have a pre-existing large number of deployments for Linux containers, as well as an ecosystem of off-the-shelf configurations, such as community Helm charts, and programmatic pod generation cases, such as with Operators. Customers will be hesitant to make the configuration change to add nodeSelectors. Our proposal as an alternative is to use Taints. Because the kubelet can set Taints during registration, it could easily be modified to automatically add a taint when running on Windows only (`--register-with-taints='os=Win1809:NoSchedule'`). By adding a taint to all Windows nodes, nothing will be scheduled on them (that includes existing Linux Pods). In order for a Windows Pod to be scheduled on a Windows node, it would need both the nodeSelector to choose Windows, and a toleration.
```
nodeSelector:
    "beta.kubernetes.io/os": windows
tolerations:
    - key: "os"
      operator: "Equal"
      value: "Win1809"
      effect: "NoSchedule"
```

#### Memory Overprovisioning

Windows always treats all user-mode memory allocations as virtual, and pagefiles are mandatory. The net effect is that Windows won't reach out of memory conditions the same way Linux does, and processes will page to disk instead of being subject to out of memory (OOM) termination. There is no way to guarantee a physical memory allocation or reserve for a process - only limits. See [#73417](https://github.com/kubernetes/kubernetes/issues/73417) for more details on the investigation for 1.14.

Keeping memory usage within reasonable bounds is possible with a two-step process. First, use the kubelet parameters `--kubelet-reserve` and/or `--system-reserve` to account for memory usage on the node (outside of containers). This will reduce [NodeAllocatable](https://kubernetes.io/docs/tasks/administer-cluster/reserve-compute-resources/#node-allocatable). As you deploy workloads, use resource limits and reserves on containers. This will also subtract from NodeAllocatable and prevent the scheduler from adding more pods once a node is full. These will be documented as best practices for v1.14. The related kubelet parameters `--eviction-hard`, `--eviction-soft`, and `--enforce-node-allocatable` are invalid for v1.14.

For later releases, we can work on a configurable heuristic to detect memory pressure, report it through the kubelet `MemoryPressure` condition, and implement pod eviction. 

## Graduation Criteria
- All features and functionality under `What works today` is fully tested and vetted to be working by SIG-Windows
- SIG-Windows has high confidence to the stability and reliability of Windows Server containers on Kubernetes
- 100% green/passing conformance tests that are applicable to Windows (see the Testing Plan section for details on these tests). These tests are adequate, non flaky, and continuously run. The test results are publicly accessible, enabled as part of the release-blocking suite
- Compatibility will not be broken, either for existing users/clusters/features or for the new features going forward, and we will adhere to the deprecation policy (https://kubernetes.io/docs/reference/using-api/deprecation-policy/).
- Comprehensive documentation that includes but is not limited to the following sections. Documentation will reside at https://kubernetes.io/docs and will adequately cover end user and admin documentation that describes what the user does and how to use it. Not all of the documentation will be under the Getting Started Guide for Windows. Part of it will reside in its own sections (like the Group Managed Service Accounts) and part will be in the Contributor development guide (like the instructions on how to build your own source code)
1. Outline of Windows Server containers on Kubernetes
2. Getting Started Guide, including Prerequisites
3. How to deploy Windows nodes in Kubernetes and where to find the proper binaries (Listed under the changelog for every release. For example https://github.com/kubernetes/kubernetes/blob/master/CHANGELOG-1.13.md#server-binaries)
4. Overview of Networking on Windows
5. Links to documentation on how to deploy and use CNI plugins for Windows (example for OVN - https://github.com/openvswitch/ovn-kubernetes/tree/master/contrib)
6. Links to documentation on how to deploy Windows nodes for public cloud providers or other Kubernetes distributions (example for Rancher - https://rancher.com/docs//rancher/v2.x/en/cluster-provisioning/rke-clusters/windows-clusters/)
7. How to schedule Windows Server containers, including examples
8. How to use metrics and the Horizontal Pod Autoscaler
9. How to use Group Managed Service Accounts
10. How to use Taints and Tolerations for a heterogeneous compute cluster (Windows + Linux)
11. How to use Hyper-V isolation (not a stable feature yet)
12. How to build Kubernetes for Windows from source
13. Supported functionality (with examples where appropriate)
14. Known Limitations
15. Unsupported functionality
16. Resources for contributing and getting help - Includes troubleshooting help and links to additional troubleshooting guides like https://docs.microsoft.com/en-us/virtualization/windowscontainers/kubernetes/common-problems

## Implementation History
- Alpha was released with Kubernetes v.1.5
- Beta was released with Kubernetes v.1.9

## Testing Plan


### Test Dashboard

All test cases will be built in kubernetes/test/e2e, scheduled through [prow](https://github.com/kubernetes/test-infra/blob/master/config/jobs/kubernetes-sigs/sig-windows/sig-windows-config.yaml), and published on the [TestGrid SIG-Windows dashboard](https://testgrid.k8s.io/sig-windows#aks-engine-azure-windows-master) daily. This will be the master list of what needs to pass to be declared stable and will include all tests tagged [SIG-Windows] along with the subset of conformance tests that can pass on Windows. 

Additional dashboard pages will be added over time as we run the same test cases with additional CRI, CNI and cloud providers. They are running the same test cases, and are not required for v1.14 graduation to stable.

- [Windows Server 2019 on GCE](https://testgrid.k8s.io/sig-windows#gce-windows-master)
- [Windows Server 2019 with Flannel on vSphere](https://testgrid.k8s.io/sig-windows#cfcr-vsphere-windows-master)
- Windows Server 2019 with OVN+OVS & Dockershim
- Windows Server 2019 with OVN+OVS & CRI-ContainerD
- Windows Server 2019 with Azure-CNI & CRI-ContainerD
- Windows Server 2019 with Flannel & CRI-ContainerD

### Test Environment

The primary test environment deployed by [kubetest](https://github.com/kubernetes/test-infra/blob/72c720f29cb43d923ac76b10d25a62c29662683d/kubetest/azure.go#L180) for v1.14 is a group of VMs deployed on Azure:

- 1 Master VM running Ubuntu, size "Standard_D2s_v3"
  - Moby v3.0.1 (https://packages.microsoft.com/ubuntu/16.04/prod/pool/main/m/moby-engine/)
  - Azure CNI
- 3 Windows nodes running Windows Server 2019, size "Standard_D2s_v3"
  - Docker EE-Basic v.18.09
  - Azure CNI

Kubetest uses [aks-engine](https://github.com/Azure/aks-engine) to create the deployment template for each of those VMs that's passed on to Azure for deployment. Once the test pass is complete, kubetest deletes the cluster. The Azure subscription used for this test pass is managed by Lachie Evenson & Patrick Lang. The credentials needed were given to the k8s-infra-oncall team.

### Test Approach

The testing for Windows nodes will include multiple approaches:

1. [Adapting](#Adapting-existing-tests) some of the existing conformance tests to be able to pass on multiple node OS's. Tests that won't work will be [excluded](https://github.com/kubernetes/test-infra/blob/master/config/jobs/kubernetes-sigs/sig-windows/sig-windows-config.yaml#L69).
2. Adding [substitute](#Substitute-test-cases) test cases where the first approach isn't feasible or would change the tests in a way is not approved by the owner. These will be tagged with `[SIG-Windows]`
3. Last, gaps will be filled with [Windows specific tests](#Windows-specific-tests). These will also be tagged with `[SIG-Windows]`

All of the test cases will be maintained within the kubernetes/kubernetes repo. SIG-Windows specific tests for 2/3 will be in [test/e2e/windows](https://github.com/kubernetes/kubernetes/tree/master/test/e2e/windows)

Additional Windows test setup scripts, container image source code, and documentation will be kept in the [kubernetes-sigs/windows-testing](https://github.com/kubernetes-sigs/windows-testing) repo. One example is that the prow jobs need a list of repos to use for the test containers, and that will be maintained here - see [windows-testing#1](https://github.com/kubernetes-sigs/windows-testing/issues/1).
Building these containers for Windows requires a Windows build machine, which isn't part of the Kubernetes PR or official builds. If the SIG is given access to a suitable GCR.io account, images can be pushed there. Otherwise, we'll use continue pushing to Docker Hub.


#### Adapting existing tests

Over the course of v1.12/13, many conformance tests were adapted to be able to pass on either Linux or Windows nodes as long as matching OS containers are run. This was done by creating Windows equivalent containers from [kubernetes/test/images](https://github.com/kubernetes/kubernetes/tree/master/test/images). An additional parameter is needed for e2e.test/kubetest to change the container repos to the one containing Windows versions since they're not part of the Kubernetes build process yet.

These tests are already running and listed on the dashboard above, with a few exceptions:

- [x] "... should function for node-pod communication: udp" - issue [#72917](https://github.com/kubernetes/kubernetes/issues/72917) has a PR open
- [x] "should be able to pull image from docker hub" - [PR #72777](https://github.com/kubernetes/kubernetes/pull/72777) open
- [x] "should provide DNS for the cluster" - [PR #72729](https://github.com/kubernetes/kubernetes/pull/72729) open for issue [#70189](https://github.com/kubernetes/kubernetes/issues/70189)


And also some cleanup to simplify the test exclusions:
 - [x] Skip Windows unrelated tests (those are tagged as `LinuxOnly`) - (https://github.com/kubernetes/kubernetes/pull/73204)

#### Substitute test cases

These are test cases that follow a similar flow to a conformance test that is dependent on Linux-specific functionality, but differs enough that the same test case cannot be used for both Windows & Linux. Examples include differences in file access permissions (UID/GID vs username, permission octets vs Windows ACLs), and network configuration (`/etc/resolv.conf` is used on Linux, but Windows DNS settings are stored in the Windows registry).

These test cases are in review:


- [x] [sig-network] [sig-windows] Networking Granular Checks: Pods should function for intra-pod communication: http - [PR#71468](https://github.com/kubernetes/kubernetes/pull/71468)
- [x] [sig-network] [sig-windows] Networking Granular Checks: Pods should function for intra-pod communication: udp - [PR#71468](https://github.com/kubernetes/kubernetes/pull/71468)
- [x] [sig-network] [sig-windows] Networking Granular Checks: Pods should function for node-pod communication: udp - [PR#71468](https://github.com/kubernetes/kubernetes/pull/71468)
- [x] [sig-network] [sig-windows] Networking Granular Checks: Pods should function for node-pod communication: http - [PR#71468](https://github.com/kubernetes/kubernetes/pull/71468)


And these still need to be covered: 

- [x] DNS configuration is passed through CNI, not `/etc/resolv.conf` [67435](https://github.com/kubernetes/kubernetes/pull/67435)
  - Test cases needed for `dnsPolicy`: Default, ClusterFirst, None
  - Test cases needed for `dnsConfig`
  - Test cases needed for `hostname`
  - Test cases needed for `subdomain`

Tests will be merged once the CNI plugins are updated. See [Custom DNS updates for CNI plugins](#custom-dns-updates-for-cni-plugins) for full details.

- [x] Windows doesn't have CGroups, but nodeReserve and kubeletReserve are [implemented](https://github.com/kubernetes/kubernetes/pull/69960)



#### Windows specific tests

We will also add Windows scenario-specific tests to cover more typical use cases and features specific to Windows. These tests will be in [kubernetes/test/e2e/windows](https://github.com/kubernetes/kubernetes/tree/master/test/e2e/windows). This will also include density and performance tests that are adjusted for Windows apps which have different image sizes and memory requirements.

These areas still need test cases written:

- [x] System, pod & network stats are implemented in kubelet, not cadvisor [70121](https://github.com/kubernetes/kubernetes/pull/70121), [66427](https://github.com/kubernetes/kubernetes/pull/66427), [62266](https://github.com/kubernetes/kubernetes/pull/62266), [51152](https://github.com/kubernetes/kubernetes/pull/51152), [50396](https://github.com/kubernetes/kubernetes/pull/50396)
- [ ] Create a `NodePort` service, and verify it's accessible on both Linux & Windows node IPs on the correct port [tracked as #73327](https://github.com/kubernetes/kubernetes/issues/73327)
- [x] Verify `ExternalPort` works from Windows pods [tracked as #73328](https://github.com/kubernetes/kubernetes/issues/73328)
- [x] Verify `imagePullPolicy` behaviors. The reason behind needing a Windows specific test is because we may need to publish Windows-specific images for this validation. The current tests are pulling Linux images. Long term we will work with the team to use a universal/heterogeneous image if possible.



## Conformance Testing

There were lots of discussions with SIG-Architecture and the Conformance working group on what Windows means for conformance. For the purposes of this KEP - graduating Windows node support to stable does not require conformance testing for v1.14, and will be completed later. This also means that clusters with Windows nodes will not be eligible for the conformance logo. During v1.14, SIG-Windows will be finishing the right set of tests so that we can propose changes to existing tests to make them OS agnostic, and what additional Windows-specific tests are needed. With continued work through the conformance working group, our goal would be to move these into a Windows conformance profile for v1.15. This would mean clusters could be tested and certified with only Linux nodes for 1.15+, no different from how they were run in <= 1.14. Windows nodes could be added and tested against the new conformance profile, but not all clusters will require Windows.


## API Reference

This section provides an API by API list of Windows & Linux differences. Issue [#70604](https://github.com/kubernetes/kubernetes/issues/70604) will be used to track updating the generated API docs with notes on Windows support where needed.

There are no differences in how most of the Kubernetes APIs work. The subtleties around what's different come down to differences in the OS and container runtime. Where a property on a workload API such as Pod or Container was designed with an assumption that it's implemented on Linux, then that may not hold true on Windows.

At a high level, these OS concepts are different:

- Identity - Linux uses userID (UID) and groupID (GID) which are represented as integer types. User and group names are not canonical - they are just an alias in /etc/groups or /etc/passwd back to UID+GID. Windows uses a larger binary security identifier (SID) which is stored in the Windows Security Access Manager (SAM) database. This database is not shared between the host and containers, or between containers.
- File permissions - Windows uses an access control list based on SIDs, rather than a bitmask of permissions and UID+GID
- File paths - convention on Windows is to use `\` instead of `/`. The Go IO libraries typically accept both and just make it work, but when you're setting a path or commandline that's interpreted inside a container, `\` may be needed.
- Signals - Windows interactive apps handle termination differently, and can implement one or more of these:
  - A UI thread will handle well-defined messages including [WM_CLOSE](https://docs.microsoft.com/en-us/windows/desktop/winmsg/wm-close)
  - Console apps will handle `ctrl-c` or `ctrl-break` using a [Control Handler](https://docs.microsoft.com/en-us/windows/console/registering-a-control-handler-function)
  - Services will register a [Service Control Handler](https://docs.microsoft.com/en-us/windows/desktop/Services/service-control-handler-function) function that can accept `SERVICE_CONTROL_STOP` control codes

These conventions are the same:
- Exit Codes mostly follow the same convention where 0 is success, nonzero is failure. The [specific error codes](https://docs.microsoft.com/en-us/windows/desktop/Debug/system-error-codes--0-499-) may differ. Exit codes passed from the Kubernetes components (kubelet, kube-proxy) will be unchanged.


The Windows container runtime also has a few important differences:

- Resource management and process isolation - Linux cgroups are used as a pod boundary for resource controls. Containers are created within that boundary for network, process and filesystem isolation. The cgroups APIs can be used to gather cpu/io/memory stats. Windows uses a Job object per container with a system namespace filter to contain all processes in a container and provide logical isolation from the host.
  - There is no way to run a Windows container without the namespace filtering in place. This means that system privileges cannot be asserted in the context of the host, and privileged containers are not available on Windows. Containers cannot assume an identity from the host because the SAM is separate.
- Filesystems - Windows has a layered filesystem driver to mount container layers and create a copy filesystem based on NTFS. All file paths in the container are resolved only within the context of that container.
  - Volume mounts can only target a directory in the container, and not an individual file.
  - Volume mounts cannot project files or directories back to the host filesystem.  
  - Read-only filesystems are not supported because write access is always required for the Windows registry and SAM database. Read-only volumes are supported
  - Volume user-masks and permissions are not available. Because the SAM is not shared between the host & container, there's no mapping between them. All permissions are resolved within the context of the container.
- Networking - The Windows host networking networking service and virtual switch implement namespacing and can create virtual NICs as needed for a pod or container. However, many configurations such as DNS, routes, and metrics are stored in the Windows registry database rather than /etc/... files as they are on Linux. The Windows registry for the container is separate from that of the host, so concepts like mapping /etc/resolv.conf from the host into a container don't have the same effect they would on Linux. These must be configured using Windows APIs run in the context of that container. Therefore CNI implementations need to call  the HNS instead of relying on file mappings to pass network details into the pod or container.



### V1.Container

- `V1.Container.ResourceRequirements.limits.cpu` and `V1.Container.ResourceRequirements.limits.memory` - Windows doesn't use hard limits for CPU allocations. Instead, a share system is used. The existing fields based on millicores are scaled into relative shares that are followed by the Windows scheduler. [see: kuberuntime/helpers_windows.go](https://github.com/kubernetes/kubernetes/blob/v1.18.0-beta.0/pkg/kubelet/kuberuntime/helpers_windows.go), [see: resource controls in Microsoft docs](https://docs.microsoft.com/en-us/virtualization/windowscontainers/manage-containers/resource-controls)
When using Hyper-V isolation (alpha), the hypervisor also needs a number of CPUs assigned. The millicores used in the limit is divided by 1000 to get the number of cores required. The CPU count is a hard limit.
  - Huge pages are not implemented in the Windows container runtime, and are not available. They require [asserting a user privilege](https://docs.microsoft.com/en-us/windows/desktop/Memory/large-page-support) that's not configurable for containers.
- `V1.Container.ResourceRequirements.requests.cpu` and `V1.Container.ResourceRequirements.requests.memory` - Requests are subtracted from node available resources, so they can be used to avoid overprovisioning a node. However, they cannot be used to guarantee resources in an overprovisioned node. They should be applied to all containers as a best practice if the operator wants to avoid overprovisioning entirely.
- `V1.Container.SecurityContext.allowPrivilegeEscalation` - not possible on Windows, none of the capabilies are hooked up
- `V1.Container.SecurityContext.Capabilities` - POSIX capabilities are not implemented on Windows
- `V1.Container.SecurityContext.privileged` - Windows doesn't support privileged containers
- `V1.Container.SecurityContext.procMount` - Windows doesn't have a `/proc` filesystem
- `V1.Container.SecurityContext.readOnlyRootFilesystem` - not possible on Windows, write access is required for registry & system processes to run inside the container
- `V1.Container.SecurityContext.runAsGroup` - not possible on Windows, no GID support
- `V1.Container.SecurityContext.runAsNonRoot` - Windows does not have a root user. The closest equivalent is `ContainerAdministrator` which is an identity that doesn't exist on the node.
- `V1.Container.SecurityContext.runAsUser` - not possible on Windows, no UID support as int. This needs to change to IntStr, see [64009](https://github.com/kubernetes/kubernetes/pull/64009), to support Windows users as strings, or another field is needed. Work remaining tracked in [#73387](https://github.com/kubernetes/kubernetes/issues/73387)
- `V1.Container.SecurityContext.seLinuxOptions` - not possible on Windows, no SELinux
- `V1.Container.terminationMessagePath` - this has some limitations in that Windows doesn't support mapping single files. The default value is `/dev/termination-log`, which does work because it does not exist on Windows by default.


### V1.Pod

- `V1.Pod.hostIPC`, `v1.pod.hostpid` - host namespace sharing is not possible on Windows
- `V1.Pod.hostNetwork` - There is no Windows OS support to share the host network
- `V1.Pod.dnsPolicy` - ClusterFirstWithHostNet - is not supported because Host Networking is not supported on Windows.
- `V1.Pod.podSecurityContext` - see [V1.PodSecurityContext](#v1podsecuritycontext)
- `V1.Pod.shareProcessNamespace` - this is an beta feature, and depends on Linux namespaces which are not implemented on Windows. Windows cannot share process namespaces or the container's root filesystem. Only the network can be shared.
- `V1.Pod.terminationGracePeriodSeconds` - this is not fully implemented in Docker on Windows, see: [reference](https://github.com/moby/moby/issues/25982). The behavior today is that the ENTRYPOINT process is sent `CTRL_SHUTDOWN_EVENT`, then Windows waits 5 seconds by hardcoded default, and finally shuts down all processes using the normal Windows shutdown behavior. The 5 second default is actually in the Windows registry [inside the container](https://github.com/moby/moby/issues/25982#issuecomment-426441183), so it can be overridden when the container is built. Runtime configuration will be feasible in CRI-ContainerD but not for v1.14. Issue [#73434](https://github.com/kubernetes/kubernetes/issues/73434) is tracking this for a later release.
- `V1.Pod.volumeDevices` - this is an beta feature, and is not implemented on Windows. Windows cannot attach raw block devices to pods.
- `V1.Pod.volumes` - EmptyDir, Secret, ConfigMap, HostPath - all work and have tests in TestGrid
  - `V1.emptyDirVolumeSource` - the Node default medium is disk on Windows. `memory` is not supported, as Windows does not have a built-in RAM disk.
- `V1.VolumeMount.mountPropagation` - is not supported on Windows

### V1.PodSecurityContext

None of the PodSecurityContext fields work on Windows. They're listed here for reference.

- `V1.PodSecurityContext.SELinuxOptions` - SELinux is not available on Windows
- `V1.PodSecurityContext.RunAsUser` - provides a UID, not available on Windows
- `V1.PodSecurityContext.RunAsGroup` - provides a GID, not available on Windows
- `V1.PodSecurityContext.RunAsNonRoot` - Windows does not have a root user. The closest equivalent is `ContainerAdministrator` which is an identity that doesn't exist on the node.
- `V1.PodSecurityContext.SupplementalGroups` - provides GID, not available on Windows
- `V1.PodSecurityContext.Sysctls` - these are part of the Linux sysctl interface. There's no equivalent on Windows.


## Other references

[Past release proposal for v1.12/13](https://docs.google.com/document/d/1YkLZIYYLMQhxdI2esN5PuTkhQHhO0joNvnbHpW68yg8/edit#)
