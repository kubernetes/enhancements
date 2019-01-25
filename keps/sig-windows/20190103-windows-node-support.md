---
title: Windows node support
authors:
  - "@astrieanna"
  - "@benmoss"
  - "@patricklang"
  - "@michmike"
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
last-updated: 2019-01-25
status: provisional
---

# Windows node support


## Table of Contents
<!-- TOC -->

- [Table of Contents](#table-of-contents)
- [Summary](#summary)
- [Motivation](#motivation)
    - [Goals](#goals)
    - [Non-Goals](#non-goals)
- [Proposal](#proposal)
    - [What works today](#what-works-today)
    - [What will work eventually](#what-will-work-eventually)
    - [What will never work (without underlying OS changes)](#what-will-never-work-without-underlying-os-changes)
    - [Relevant resources/conversations](#relevant-resourcesconversations)
    - [Risks and Mitigations](#risks-and-mitigations)
- [Graduation Criteria](#graduation-criteria)
- [Implementation History](#implementation-history)
- [Testing Plan](#testing-plan)
    - [Test Dashboard](#test-dashboard)
    - [Test Approach](#test-approach)
        - [Adapting existing tests](#adapting-existing-tests)
        - [Substitute test cases](#substitute-test-cases)
        - [Windows specific tests](#windows-specific-tests)
- [Other references](#other-references)

<!-- /TOC -->

## Summary

There is strong interest in the community for adding support for workloads running on Microsoft Windows. This is non-trivial due to the significant differences in the implementation of Windows from the Linux-based OSes that have so far been supported by Kubernetes.


## Motivation

Windows-based workloads still account for a significant portion of the enterprise software space. While containerization technologies emerged first in the UNIX ecosystem, Microsoft has made investments in recent years to enable support for containers in its Windows OS. As users of Windows increasingly turn to containers as the preferred abstraction for running software and modernizing existing applications, the Kubernetes ecosystem stands to benefit by becoming a cross-platform cluster manager.

### Goals

- Enable users to schedule Windows Server containers in Kubernetes through the introduction of support for Windows compute nodes
- Document the differences and limitations compared to Linux
- Create a test suite in testgrid to maintain high quality for this feature and prevent regression of functionality 

### Non-Goals

- Adding Windows support to all projects in the Kubernetes ecosystem (Cluster Lifecycle, etc)
- Enable the Kubernetes master components to run on Windows

## Proposal

As of 29-11-2018 much of the work for enabling Windows nodes has already been completed. Both `kubelet` and `kube-proxy` have been adapted to work on Windows Server, and so the first goal of this KEP is largely already complete. 

### What works today
- Windows-based containers can be created by kubelet, [provided the host OS version matches the container base image](https://docs.microsoft.com/en-us/virtualization/windowscontainers/deploy-containers/version-compatibility)
    - Pod (single or multiple containers per Pod with process isolation), Deployment, ReplicaSet
    - ConfigMap, Secrets: as environment variables or volumes
    - Resource limits
    - Pod & container metrics
    - Horizontal Pod Autoscaling
- Windows Server 2019 is the only Windows operating system we will support at GA timeframe. Note above that the host operating system version and the container base image need to match. This is a Windows limitation we cannot overcome.
- Customers can deploy a heterogeneous cluster, with Windows and Linux compute nodes side-by-side and schedule Docker containers on both operating systems. Of course, Windows Server containers have to be scheduled on Windows and Linux containers on Linux
- Out-of-tree Pod networking with [Azure-CNI](https://github.com/Azure/azure-container-networking/blob/master/docs/cni.md), [OVN-Kubernetes](https://github.com/openvswitch/ovn-kubernetes), [two CNI meta-plugins](https://github.com/containernetworking/plugins), [Flannel (VXLAN and Host-Gateway)](https://github.com/coreos/flannel) 
- Dockershim CRI
- Many<sup id="a1">[1]</sup> of the e2e conformance tests when run with [alternate Windows-based images](https://hub.docker.com/r/e2eteam/) which are being moved to [kubernetes-sigs/windows-testing](https://www.github.com/kubernetes-sigs/windows-testing)
- Persistent storage: FlexVolume with [SMB + iSCSI](https://github.com/Microsoft/K8s-Storage-Plugins/tree/master/flexvolume/windows), and in-tree AzureFile and AzureDisk providers
- Windows Server containers can take advantage of StatefulSet functionality for stateful applications and distributed systems
- Windows Pods can take advantage of DaemonSet, with the exception that privileged containers are not supported on Windows (more on that below)
 
### What will work eventually
- Group Managed Service Accounts, a way to assign an Active Directory identity to a Windows container, is forthcoming with KEP `Windows Group Managed Service Accounts for Container Identity`
- `kubectl port-forward` hasn't been implemented due to lack of an `nsenter` equivalent to run a process inside a network namespace.
- CRIs other than Dockershim: CRI-containerd support is forthcoming
- Some kubeadm work was done in the past to add Windows nodes to Kubernetes, but that effort has been dormant since. We will need to revisit that work and complete it in the future.
- Calico CNI for Pod networking
- Hyper-V isolation (Currently this is limited to 1 container per Pod and is an alpha feature)
- It is unclear if the RuntimeClass proposal from sig-node will simplify scheduled Windows containers. we will work with sig-node on this.

### What will never work (without underlying OS changes)
- Certain Pod functionality
    - Privileged containers and other Pod security context privilege and access control settings
    - Reservations are not enforced by the OS, but overprovisioning could be blocked with `--enforce-node-allocatable=pods` (pending: tests needed)
    - Certain volume mappings
      - Single file & subpath volume mounting
      - Host mount projection
      - DefaultMode (due to UID/GID dependency)
      - readOnly root filesystem. Mapped volumes still support readOnly
    - Termination Message - these require single file mappings
- CSI plugins, which require privileged containers
- Host networking is not available in Windows
- [Some parts of the V1 API](https://github.com/kubernetes/kubernetes/issues/70604)
- Overlay networking support in Windows Server 1803 is not fully functional using the `win-overlay` CNI plugin. Specifically service IPs do not work on Windows nodes. This is currently specific to `win-overlay`; other CNI plugins (OVS, AzureCNI) work. Since Windows Server 1803 is not supported for GA, this is mostly not applicable. We left it here since it impacts beta

### Relevant resources/conversations

- [sig-architecture thread](https://groups.google.com/forum/#!topic/kubernetes-sig-architecture/G2zKJ7QK22E)
- [cncf-k8s-conformance thread](https://lists.cncf.io/g/cncf-k8s-conformance/topic/windows_conformance_tests/27913232)
- [kubernetes/enhancements proposal](https://github.com/kubernetes/features/issues/116)


### Risks and Mitigations

**Second class support**: Kubernetes contributors are likely to be thinking of Linux-based solutions to problems, as Linux remains the primary OS supported. Keeping Windows support working will be an ongoing burden potentially limiting the pace of development. 

**User experience**: Users today will need to use some combination of taints and node selectors in order to keep Linux and Windows workloads separated. In the best case this imposes a burden only on Windows users, but this is still less than ideal. The recommended approach is outlined below

#### Ensuring OS-specific workloads land on appropriate container host
As you can see below, we plan to document how Windows containers can be scheduled on the appropriate host using Taints and Tolerations. All nodes today have the following default labels
- beta.kubernetes.io/os = [windows|linux]
- beta.kubernetes.io/arch = [amd64|arm64|...]

If a deployment does not specify a nodeSelector like `"beta.kubernetes.io/os": windows`, it is possible the Pods can be scheduled on any host, Windows of Linux. This can be problematic since a Windows container can only land on Windows and a Linux container can only land on Linux. The best practice we will recommend is to use a nodeSelector. 

However, we understand that in certain cases customers have a pre-existing large number of deployments for Linux containers. Since they will not want to change all deployments to add nodeSelectors, the alternative is to use Taints. Because the kubelet can set Taints during registration, it could easily be modified to automatically add a taint when running on Windows only (`“--register-with-taints=’os=Win1809:NoSchedule’” `). By adding a taint to all Windows nodes, nothing will be scheduled on them (that includes existing Linux Pods). In order for a Windows Pod to be scheduled on a Windows node, it would need both the nodeSelector to choose Windows, and a toleration.
```
nodeSelector:
    "beta.kubernetes.io/os": windows
tolerations:
    - key: "os"
      operator: "Equal"
      Value: “Win1809”
      effect: "NoSchedule"
```

## Graduation Criteria
- All features and functionality under `What works today` is fully tested and vetted to be working by SIG-Windows
- SIG-Windows has high confidence to the stability and reliability of Windows Server containers on Kubernetes
- 100% green/passing conformance tests that are applicable to Windows (see the Testing Plan section for details on these tests)
- Comprehensive documentation that includes but is not limited to the following sections. Documentation will reside at https://kubernetes.io/docs
1. Outline of Windows Server containers on Kubernetes
2. Getting Started Guide, including Prerequisites
3. How to deploy Windows nodes in Kubernetes
4. Overview of Networking on Windows
5. Links to documentation on how to deploy and use CNI plugins for Windows (example for OVN - https://github.com/openvswitch/ovn-kubernetes/tree/master/contrib)
6. Links to documentation on how to deploy Windows nodes for public cloud providers or other Kubernetes distributions (example for Rancher - https://rancher.com/docs//rancher/v2.x/en/cluster-provisioning/rke-clusters/windows-clusters/)
7. How to schedule Windows Server containers, including examples
8. Advanced: How to use metrics and the Horizontal Pod Autoscaler
9. Advanced: How to use Group Managed Service Accounts
10. Advanced: How to use Taints and Tolerations for a heterogeneous compute cluster (Windows + Linux)
11. Advanced: How to use Hyper-V isolation (not a stable feature yet)
12. Advanced: How to build Kubernetes for Windows from source
13. Supported functionality (with examples where appropriate)
14. Known Limitations
15. Unsupported functionality
16. Resources for contributing and getting help - Includes troubleshooting help and links to additional troubleshooting guides like https://docs.microsoft.com/en-us/virtualization/windowscontainers/kubernetes/common-problems

## Implementation History
- Alpha was released with Kubernetes v.1.5
- Beta was released with Kubernetes v.1.9

## Testing Plan


### Test Dashboard

All test cases will be built in kubernetes/test/e2e, scheduled through [prow](github.com/kubernetes/test-infra/blob/master/config/jobs/kubernetes-sigs/sig-windows/sig-windows-config.yaml), and published on the [TestGrid SIG-Windows dashboard](https://testgrid.k8s.io/sig-windows) daily. This will be the master list of what needs to pass to be declared stable and will include all tests tagged [SIG-Windows] along with the subset of conformance tests that can pass on Windows.


Additional dashboard pages will be added over time as we run the same test cases with additional CRI, CNI and cloud providers. They reflect work that may be stabilized in v1.15 or later and is not strictly required for v1.14.

- Windows Server 2019 on GCP - this is [in progress](https://k8s-testgrid.appspot.com/google-windows#windows-prototype), but not required for v1.14
- Windows Server 2019 with OVN+OVS & Dockershim
- Windows Server 2019 with OVN+OVS & CRI-ContainerD
- Windows Server 2019 with Azure-CNI & CRI-ContainerD
- Windows Server 2019 with Flannel & CRI-ContainerD

### Test Approach

The testing for Windows nodes will include multiple approaches:

1. [Adapting](#Adapting-existing-tests) some of the existing conformance tests to be able to pass on multiple node OS's. Tests that won't work will be [excluded](https://github.com/kubernetes/test-infra/blob/master/config/jobs/kubernetes-sigs/sig-windows/sig-windows-config.yaml#L69).
2. Adding [substitute](#Substitute-test-cases) test cases where the first approach isn't feasible or would change the tests in a way is not approved by the owner. These will be tagged with `[SIG-Windows]`
3. Last, gaps will be filled with [Windows specific tests](#Windows-specific-tests). These will also be tagged with `[SIG-Windows]`

All of the test cases will be maintained within the kubernetes/kubernetes repo. SIG-Windows specific tests for 2/3 will be in [test/e2e/windows](https://github.com/kubernetes/kubernetes/tree/master/test/e2e/windows)

Additional Windows test setup scripts, container image source code, and documentation will be kept in the [kubernetes-sigs/windows-testing](https://github.com/kubernetes-sigs/windows-testing) repo. One example is that the prow jobs need a list of repos to use for the test containers, and that will be maintained here - see [windows-testing#1](https://github.com/kubernetes-sigs/windows-testing/issues/1).


#### Adapting existing tests

Over the course of v1.12/13, many conformance tests were adapted to be able to pass on either Linux or Windows nodes as long as matching OS containers are run. This was done by creating Windows equivalent containers from [kubernetes/test/images](https://github.com/kubernetes/kubernetes/tree/master/test/images). An additional parameter is needed for e2e.test/kubetest to change the container repos to the one containing Windows versions since they're not part of the Kubernetes build process yet.

These tests are already running and listed on the dashboard above, with a few exceptions:

- [ ] "... should function for node-pod communication: udp" - issue [#72917](https://github.com/kubernetes/kubernetes/issues/72917) has a PR open
- [ ] "should be able to pull image from docker hub" - [PR #72777](https://github.com/kubernetes/kubernetes/pull/72777) open
- [ ] "should provide DNS for the cluster" - [PR #72729](https://github.com/kubernetes/kubernetes/pull/72729) open for issue [#70189](https://github.com/kubernetes/kubernetes/issues/70189)


And also some cleanup to simplify the test exclusions:
 - [ ] Skip Windows unrelated tests - [Issue #69871](https://github.com/kubernetes/kubernetes/issues/69871), [PR#69872](https://github.com/kubernetes/kubernetes/pull/69872)

#### Substitute test cases

These are test cases that follow a similar flow to a conformance test that is dependent on Linux-specific functionality, but differs enough that the same test case cannot be used for both Windows & Linux. Examples include differences in file access permissions (UID/GID vs username, permission octets vs Windows ACLs), and network configuration (`/etc/resolv.conf` is used on Linux, but Windows DNS settings are stored in the Windows registry).

These test cases are in review:


- [ ] [sig-network] [sig-windows] Networking Granular Checks: Pods should function for intra-pod communication: http - [PR#71468](https://github.com/kubernetes/kubernetes/pull/71468)
- [ ] [sig-network] [sig-windows] Networking Granular Checks: Pods should function for intra-pod communication: udp - [PR#71468](https://github.com/kubernetes/kubernetes/pull/71468)
- [ ] [sig-network] [sig-windows] Networking Granular Checks: Pods should function for node-pod communication: udp - [PR#71468](https://github.com/kubernetes/kubernetes/pull/71468)
- [ ] [sig-network] [sig-windows] Networking Granular Checks: Pods should function for node-pod communication: http - [PR#71468](https://github.com/kubernetes/kubernetes/pull/71468)


And these still need to be covered: 

- [ ] DNS configuration is passed through CNI, not `/etc/resolv.conf` [67435](https://github.com/kubernetes/kubernetes/pull/67435)
- [ ] Windows doesn't have CGroups, but nodeReserve and kubeletReserve are [implemented](https://github.com/kubernetes/kubernetes/pull/69960)



#### Windows specific tests

We will also add Windows scenario-specific tests to cover more typical use cases and features specific to Windows. These tests will be in [kubernetes/test/e2e/windows](https://github.com/kubernetes/kubernetes/tree/master/test/e2e/windows). This will also include density and performance tests that are adjusted for Windows apps which have different image sizes and memory requirements.

These areas still need test cases written:

- [ ] System, pod & network stats are implemented in kubelet, not cadvisor [70212](https://github.com/kubernetes/kubernetes/pull/70121), [66427](https://github.com/kubernetes/kubernetes/pull/66427), [62266](https://github.com/kubernetes/kubernetes/pull/62266), [51152](https://github.com/kubernetes/kubernetes/pull/51152), [50396](https://github.com/kubernetes/kubernetes/pull/50396)
- [ ] Windows uses username (string) or SID (binary) to define users, not UID/GID [64009](https://github.com/kubernetes/kubernetes/pull/64009)


## Other references

[Past release proposal for v1.12/13](https://docs.google.com/document/d/1YkLZIYYLMQhxdI2esN5PuTkhQHhO0joNvnbHpW68yg8/edit#)
