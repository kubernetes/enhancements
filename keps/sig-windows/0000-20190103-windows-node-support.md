---
kep-number: 0
title: Windows node support
authors:
  - "@astrieanna"
  - "@benmoss"
  - "@patricklang"
owning-sig: sig-windows
participating-sigs:
  - sig-architecture
reviewers:
  - TBD
approvers:
  - TBD
editor: TBD
creation-date: 2018-11-29
last-updated: 2019-01-03
status: provisional
---

# Windows node support


## Table of Contents

   * [Windows node support](#windows-node-support)
      * [Table of Contents](#table-of-contents)
      * [Summary](#summary)
      * [Motivation](#motivation)
         * [Goals](#goals)
         * [Non-Goals](#non-goals)
      * [Proposal](#proposal)
         * [What works today](#what-works-today)
         * [What will work eventually](#what-will-work-eventually)
         * [What will never work (without underlying OS changes)](#what-will-never-work-without-underlying-os-changes)
         * [Relevant resources/conversations](#relevant-resourcesconversations)
         * [Risks and Mitigations](#risks-and-mitigations)
      * [Graduation Criteria](#graduation-criteria)
      * [Implementation History](#implementation-history)
      * [Other references](#other-references)


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
 
<sup id="a1">1</sup> This list should be available at https://k8s-testgrid.appspot.com/sig-windows but this test setup is not currently working. https://k8s-testgrid.appspot.com/google-windows#windows-prototype is also running against a Windows cluster.

### What will work eventually
- `kubectl port-forward` hasn't been implemented due to lack of an `nsenter` equivalent to run a process inside a network namespace.
- CRIs other than Dockershim: CRI-containerd support is forthcoming


### What will never work (without underlying OS changes)
- Certain Pod functionality
    - Privileged containers
    - Reservations are not enforced by the OS, but overprovisioning could be blocked with `--enforce-node-allocatable=pods` (pending: tests needed)
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

The testing for Windows nodes will include multiple approaches:

1. [Adapting](#Adapting-existing-tests) some of the existing conformance tests to be able to pass on multiple node OS's
2. Adding [substitute](#Substitute-test-cases) test cases where the first approach isn't feasible or would change the tests in a way is not approved by the owner. These will be tagged with `[SIG-Windows]`
3. Last, gaps will be filled with [Windows specific tests](#Windows-specific-tests). These will also be tagged with `[SIG-Windows]`

All test cases will be built in kubernetes/test/e2e, scheduled through [prow](github.com/kubernetes/test-infra/blob/master/config/jobs/kubernetes-sigs/sig-windows/sig-windows-config.yaml), and published on the [TestGrid SIG-Windows dashboard](https://testgrid.k8s.io/sig-windows) daily.

Windows test setup scripts, container image source, and documentation will be kept in the [kubernetes-sigs/windows-testing](https://github.com/kubernetes-sigs/windows-testing) repo.

### Adapting existing tests

Over the course of v1.12/13, many conformance tests were adapted to be able to pass on either Linux or Windows nodes as long as matching OS containers are run. This was done by creating Windows equivalent containers from [kubernetes/test/images](https://github.com/kubernetes/kubernetes/tree/master/test/images). An additional parameter is needed for e2e.test/kubetest to change the container repos to the one containing Windows versions since they're not part of the Kubernetes build process yet.

> TODO: include list of test cases from https://docs.google.com/document/d/1YkLZIYYLMQhxdI2esN5PuTkhQHhO0joNvnbHpW68yg8/edit#


### Substitute test cases

These are test cases that follow a similar flow to a conformance test that is dependent on Linux-specific functionality, but differs enough that the same test case cannot be used for both Windows & Linux. Examples include differences in file access permissions (UID/GID vs username, permission octets vs Windows ACLs), and network configuration (`/etc/resolv.conf` is used on Linux, but Windows DNS settings are stored in the Windows registry).

> TODO: include list of test cases from open PRs

### Windows specific tests

We will also add Windows scenario-specific tests to cover more typical use cases and features specific to Windows. These tests will be in [kubernetes/test/e2e/windows](https://github.com/kubernetes/kubernetes/tree/master/test/e2e/windows). This will also include density and performance tests that are adjusted for Windows apps which have different image sizes and memory requirements.

> TODO: new list here

## Other references

[Past release proposal for v1.12/13](https://docs.google.com/document/d/1YkLZIYYLMQhxdI2esN5PuTkhQHhO0joNvnbHpW68yg8/edit#)
