---
title: Windows node support
authors:
  - "@astrieanna"
  - "@benmoss"
  - "@patricklang"
owning-sig: sig-windows
participating-sigs:
  - sig-architecture
  - sig-node
reviewers:
  - sig-architecture
  - sig-node
approvers:
  - "@bgrant0607"
editor: TBD
creation-date: 2018-11-29
last-updated: 2019-01-21
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
- [API Reference](#api-reference)
    - [Volumes](#volumes)
    - [V1.Pod.Resources & V1.Container.ResourceRequirements](#v1podresources--v1containerresourcerequirements)
    - [Networking features](#networking-features)
    - [IPC & Pid](#ipc--pid)
    - [Security](#security)
    - [User Mapping](#user-mapping)

<!-- /TOC -->

## Summary

There is strong interest in the community for adding support for workloads running on Microsoft Windows. This is non-trivial due to the significant differences in the implementation of Windows from the Linux-based OSes that have so far been supported by Kubernetes.


## Motivation

Windows-native workloads still account for a significant portion of the enterprise software space. While containerization technologies emerged first in the UNIX ecosystem, Microsoft has made investments in recent years to enable support for containers in its Windows OS. As users of Windows increasingly turn to containers as the preferred abstraction for running software, the Kubernetes ecosystem stands to benefit by becoming a cross-platform cluster manager.

### Goals

- Enable users to run nodes on Windows servers 
- Document the differences and limitations compared to Linux
- Test results added to testgrid to prevent regression of functionality 

### Non-Goals

- Adding Windows support to all projects in the Kubernetes ecosystem (Cluster Lifecycle, etc)

## Proposal

As of 29-11-2018 much of the work for enabling Windows nodes has already been completed. Both `kubelet` and `kube-proxy` have been adapted to work on Windows Server, and so the first goal of this KEP is largely already complete. 

### What works today
- Windows-based containers can be created by kubelet, [provided the host OS version matches the container base image](https://docs.microsoft.com/en-us/virtualization/windowscontainers/deploy-containers/version-compatibility)
    - ConfigMap, Secrets: as environment variables or  volumes
    - Resource limits
    - Pod & container metrics
- Pod networking with [Azure-CNI](https://github.com/Azure/azure-container-networking/blob/master/docs/cni.md), [OVN-Kubernetes](https://github.com/openvswitch/ovn-kubernetes), [two CNI meta-plugins](https://github.com/containernetworking/plugins), [Flannel](https://github.com/coreos/flannel) and [Calico](https://github.com/projectcalico/calico)
- Dockershim CRI
- Many<sup id="a1">[1]</sup> of the e2e conformance tests when run with [alternate Windows-based images](https://hub.docker.com/r/e2eteam/) which are being moved to [kubernetes-sigs/windows-testing](https://www.github.com/kubernetes-sigs/windows-testing)
- Persistent storage: FlexVolume with [SMB + iSCSI](https://github.com/Microsoft/K8s-Storage-Plugins/tree/master/flexvolume/windows), and in-tree AzureFile and AzureDisk providers
 
### What will work eventually
- `kubectl port-forward` hasn't been implemented due to lack of an `nsenter` equivalent to run a process inside a network namespace.
- CRIs other than Dockershim: CRI-containerd support is forthcoming

### What will never work (without underlying OS changes)
- Certain Pod functionality
    - Privileged containers
    - Reservations are not enforced by the OS, but overprovisioning could be blocked with `--enforce-node-allocatable=pods` (pending: tests needed)
    - Certain volume mappings
      - Single file & subpath volume mounting
      - Host mount projection
      - DefaultMode (due to UID/GID dependency)
      - readOnly root filesystem. Mapped volumes still support readOnly
    - Termination Message - these require single file mappings
- CSI plugins, which require privileged containers
- [Some parts of the V1 API](https://github.com/kubernetes/kubernetes/issues/70604)
- Overlay networking support in Windows Server 1803 is not fully functional using the `win-overlay` CNI plugin. Specifically service IPs do not work on Windows nodes. This is currently specific to `win-overlay` - other CNI plugins (OVS, AzureCNI) work.

### Relevant resources/conversations

- [sig-architecture thread](https://groups.google.com/forum/#!topic/kubernetes-sig-architecture/G2zKJ7QK22E)
- [cncf-k8s-conformance thread](https://lists.cncf.io/g/cncf-k8s-conformance/topic/windows_conformance_tests/27913232)
- [kubernetes/enhancements proposal](https://github.com/kubernetes/features/issues/116)


### Risks and Mitigations

**Second class support**: Kubernetes contributors are likely to be thinking of Linux-based solutions to problems, as Linux remains the primary OS supported. Keeping Windows support working will be an ongoing burden potentially limiting the pace of development. 

**User experience**: Users today will need to use some combination of taints and node selectors in order to keep Linux and Windows workloads separated. In the best case this imposes a burden only on Windows users, but this is still less than ideal.

## Graduation Criteria


## Implementation History


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



## API Reference

This section provides an API by API list of Windows & Linux differences. Issue [#70604](https://github.com/kubernetes/kubernetes/issues/70604) will be used to track updating the generated API docs with notes on Windows support where needed.


### Volumes

`V1.Pod.Volumes`

Out of the various volume types, these should all be possible on Windows but tests are lacking:

- EmptyDirVolumeSource
- Secret
- hostPath

The main gaps in Windows Server 2016 & 1709 are that symlinks are pretty much broken. The only ones that work are SMB/CIFS mount points. Workarounds need to be investigated.

`V1.Container.volumeMounts`
Mounting volumes across some (but not all) containers will need changes to Windows. Not ready in Windows Server 2016/1709.


References: 

- [FlexVolume does not work on Windows node](https://github.com/kubernetes/kubernetes/issues/56875)
- [feature proposal add SMB(cifs) volume plugin](https://github.com/kubernetes/kubernetes/issues/56005)
- [add NFS volume support for Windows](https://github.com/kubernetes/kubernetes/issues/56188)

### V1.Pod.Resources & V1.Container.ResourceRequirements

`V1.Container.ResourceRequirements.limits.cpu`
`V1.Container.ResourceRequirements.limits.memory`

Windows schedules CPU based on CPU count & percentage of cores. We need this represented because it can help optimize app performance. CPU count is immutable once set but you can change % of core allocations.

`V1.Container.ResourceRequirements.requests.cpu`
`V1.Container.ResourceRequirements.requests.memory`

Also of note, requests aren't supported. Will pod eviction policies in the kubelet ensure reserves are met by not overprovisioning the node?

Windows can either expose a NUMA topology matching the host (best performance) or fake it to be 1 big NUMA node (suboptimal). We should think of a way to turn this on/off later - probably q2 2018


References:

- [Kubernetes Container Runtime Interface (CRI) doesn't support WindowsContainerConfig and WindowsContainerResources](https://github.com/kubernetes/kubernetes/issues/56734)


### Networking features

`V1.Pod.dnsPolicy` - I think only ClusterFirst is implemented

`V1.Pod.hostNetwork` - Not feasible on Windows Server 2016 / 1709


### IPC & Pid

`V1.Pod.hostIPC`, `v1.pod.hostpid`

How important are these? They're not implemented in Windows Server 2016 / 1709, and I'm not too sure if they'd be helpful or not.

For cases where a pod/container need to talk to the host docker / containerd daemon we could map a named pipe as a volume which would offer the same functionality as the unix socket to the Linux daemons. It works in moby but isn't hooked up in the kubelet yet.

### Security

- `V1.Container.SecurityContext.Capabilities`
- `V1.Container.SecurityContext.seLinuxOptions`

These don't have Windows equivalents since the permissions model is substantially different

`V1.Container.SecurityContext.readOnlyRootFilesystem`

This is probably doable if needed but not possible in Windows Server 2016 / 1709.

### User Mapping

There are a few fields that refer to uid/gid. These probably need to be supplemented with a Windows SID (string) and username (string)

`V1.podSecurityContext.runAsUser` provides a UID
`V1.podSecurityContext.supplementalGroups` provides GID