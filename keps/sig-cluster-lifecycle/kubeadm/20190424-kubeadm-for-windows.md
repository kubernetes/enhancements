---
title: kubeadm-for-windows
authors:
  - "@benmoss"
  - "@gab-satchi"
  - "@ksubrmnn"
  - "@neolit123"
  - "@patricklang"
owning-sig: sig-windows
participating-sigs:
  - sig-windows
  - sig-cluster-lifecycle
reviewers:
  - "@timothysc"
  - "@michmike"
  - "@fabriziopandini"
  - "@rosti"
approvers:
  - "@timothysc"
editor: "@ksubrmnn"
creation-date: 2019-04-24
last-updated: 2019-04-24
status: implementable
---

# Kubeadm for Windows

## Table of Contents

<!-- toc -->
- [Release Signoff Checklist](#release-signoff-checklist)
- [Summary](#summary)
- [Motivation](#motivation)
  - [Goals](#goals)
  - [Non-Goals](#non-goals)
- [Proposal](#proposal)
  - [User Stories](#user-stories)
    - [Story 1](#story-1)
  - [Implementation Details/Notes/Constraints](#implementation-detailsnotesconstraints)
    - [Wins used to enable privileged/host network DaemonSets](#wins-used-to-enable-privilegedhost-network-daemonsets)
    - [Provisioning script creates a fake host network](#provisioning-script-creates-a-fake-host-network)
    - [Kubeadm manages the kubelet start / stop as a service](#kubeadm-manages-the-kubelet-start--stop-as-a-service)
    - [Kubeadm makes assumptions about systemd and Linux](#kubeadm-makes-assumptions-about-systemd-and-linux)
    - [Windows vs Linux host paths](#windows-vs-linux-host-paths)
    - [Kube-proxy deployment](#kube-proxy-deployment)
    - [CNI plugin deployment](#cni-plugin-deployment)
  - [Risks and Mitigations](#risks-and-mitigations)
- [Design Details](#design-details)
  - [Test Plan](#test-plan)
  - [Graduation Criteria](#graduation-criteria)
      - [Alpha -&gt; Beta Graduation](#alpha---beta-graduation)
      - [Beta -&gt; GA Graduation](#beta---ga-graduation)
  - [Upgrade / Downgrade Strategy](#upgrade--downgrade-strategy)
  - [Version Skew Strategy](#version-skew-strategy)
- [Implementation History](#implementation-history)
- [Drawbacks](#drawbacks)
  - [Non-identical flows for Linux and Windows](#non-identical-flows-for-linux-and-windows)
- [Infrastructure Needed](#infrastructure-needed)
<!-- /toc -->

## Release Signoff Checklist

**ACTION REQUIRED:** In order to merge code into a release, there must be an issue in [kubernetes/enhancements] referencing this KEP and targeting a release milestone **before [Enhancement Freeze](https://github.com/kubernetes/sig-release/tree/master/releases)
of the targeted release**.

For enhancements that make changes to code or processes/procedures in core Kubernetes i.e., [kubernetes/kubernetes], we require the following Release Signoff checklist to be completed.

Check these off as they are completed for the Release Team to track. These checklist items _must_ be updated for the enhancement to be released.

- [ ] kubernetes/enhancements issue in release milestone, which links to KEP (this should be a link to the KEP location in kubernetes/enhancements, not the initial KEP PR)
- [ ] KEP approvers have set the KEP status to `implementable`
- [ ] Design details are appropriately documented
- [ ] Test plan is in place, giving consideration to SIG Architecture and SIG Testing input
- [ ] Graduation criteria is in place
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

In Kubernetes 1.14, official support was added for Windows Containers. However, there is no official tool available to users for joining a Windows node to a cluster. The current solution is to create a set of scripts to download Kubernetes binaries and write their config files, but this has proven to be cumbersome and is a huge pain point for Windows adoption. On Linux, Kubeadm is available to quickly join nodes to a cluster, and the intent of this KEP is to propose a design to implement the same functionality for Windows.

It should be noted that at this time, this document only proposes enablement of support for Windows worker nodes using kubeadm.

## Motivation

The motivation for this KEP is to provide a tool to allow users to take a Windows machine and join it to an existing Kubernetes cluster with a single command. The user should also be able to reset the node with a single command. We will also create a tool to install prerequisites and enable kubeadm to bring the Windows node to a Ready state.

### Goals

* Create and maintain a Powershell script to install kubelet, kubeadm and [wins](https://github.com/rancher/wins/)
* Support kubeadm join, reset and upgrade
* Provide DaemonSets to run kube-proxy and Flannel

### Non-Goals

* Installing the Container Runtime (e.g. Docker or containerd)
* Implement kubeadm init
* Implement kubeadm join --control-plane

## Proposal

### User Stories

#### Story 1

A user will have a Windows machine that they want to join to an existing Kubernetes cluster.

1. The user will download a set of required binaries (kubelet, kubeadm, and wins) using a script.

1. The script will register kubelet as a Windows service.

1. The user will run "kubeadm join ..." to join the node to the cluster. In this step kubeadm will run preflight checks and proceed with the regular join process.

1. kubeadm will restart the kubelet service using flags that kubeadm fed to the Windows service. kubeadm will proceed to bootstrap the node. After this process is finished the node should show with the status of NotReady.

1. The user will then deploy the Flannel and kube-proxy DaemonSets. These will run and initialize container networking.

1. The node status should become Ready.

### Implementation Details/Notes/Constraints

#### Wins used to enable privileged/host network DaemonSets

Windows has no native support for creating privileged containers or attaching them to the host network.
[Wins](https://github.com/rancher/wins/) is a project from Rancher that works around this shortcoming by exposing an API
to run processes on the host. This API can be exposed to containers via a [named pipe](https://docs.microsoft.com/en-us/windows/win32/ipc/named-pipes)
to allow those containers to launch processes that escape the restrictions normally imposed by Windows containerization.
Additional details about this approach can be found [here](https://docs.google.com/document/d/1dXLs2XR8tqueSYWxAb0OGzKqKzx1pR6l2b1JXXT8kqA/edit?usp=sharing).

#### Provisioning script creates a fake host network

In order for the Flannel and kube-proxy DaemonSets to run before CNI has been initialized, they need to be running with `hostNetwork: true` on their Pod specs. This is the established pattern on Linux for bootstrapping CNI, and we are utilizing it here as well. This is in spite of the fact that our containers will not actually need networking at all since the actual Flannel/kube-proxy process will be running outside of the container through wins.

In the provisioning script we create a Docker network named `host` but that is actually of the type `NAT`. This is because the kubelet only checks for this network by name, and Docker does not support networks of type `host` on Windows.

The kubelet on Windows previously would panic when told to run a hostNetwork pod, but changes made in [#84649](https://github.com/kubernetes/kubernetes/pull/84649) allow these pods to run. As a result this means we can only support kubelets from 1.17 onwards.

#### Kubeadm manages the kubelet start / stop as a service

Kubeadm has a “kubelet” phase that can start/stop the [kubelet service](https://github.com/kubernetes/kubernetes/blob/master/cmd/kubeadm/app/phases/kubelet/kubelet.go).

Kubeadm already supports running the kubelet as a Windows service. This support was added by these two PRs: [60144](https://github.com/kubernetes/kubernetes/pull/60144), [53553](https://github.com/kubernetes/kubernetes/pull/53553).

This existing support will be leveraged by this proposal.

Currently kubeadm join downloads a KubeletConfiguration "component config" from a ConfigMap (that is created during kubeadm init) and writes it on all joining nodes. Then each node proceeds to add it's overrides via kubelet flags (that take precedence over config). This process has to be modified so that this KubeletConfiguration is defaulted properly for Windows nodes. One proposed solution is to upload the defaulted KubeletConfiguration for Windows in a separate ConfigMap, but this solution is a subject to change.

#### Kubeadm makes assumptions about systemd and Linux

Kubeadm is quite coupled to systemd and Linux at the moment.
This file shows some of the kubelet-flags that are Linux bound:
[kubelet flags](https://github.com/kubernetes/kubernetes/blob/b8b689aae0d6879de5192b203590330a11c7b9e3/cmd/kubeadm/app/phases/kubelet/flags.go)

The following changes are required::
* Implement proper cri-socket detection in kubeadm for Windows.
* Omit flags such as “cgroup-driver”
* Specify the correct “pod-infra-container-image”
* Also omit writing systemd drop-in files on Windows in general.
* Populate the Windows service dynamically.

To support Windows specific flags for the kubelet, there is a requirement to split kubeadm’s app/phases/kubelet/flags.go files into two:
* app/phases/kubelet/flags_windows.go
* app/phases/kubelet/flags_linux.go

In the case of Windows, we will be updating the registration of the kubelet as a Windows service, by changing its flags based on the flags kubeadm wants to use, complying with Windows specific defaults in kubeadm.

The existing process of passing the following files to the kubelet, will be leveraged:
* /etc/kubernetes/bootstrap-kubelet.conf
* /etc/kubernetes/kubelet.conf
* /var/lib/kubelet/config.yaml

Windows related adjustments to default paths might be required. 

#### Windows vs Linux host paths

Kubeadm makes use of several Linux-specific paths. E.g. “/etc/kubernetes” is a hardcoded path in kubeadm. We intend to use these paths on Windows as well, though we would be open to making them follow Windows path standards later.

Last, for the paths used, restrictive Access Control Lists (ACL) for Windows should be applied. Golang does not convert Posix permissions to an appropriate Windows ACL, so an additional step is needed. See [mkdirWithACL from moby/moby](https://github.com/moby/moby/blob/e4cc3adf81cc0810a416e2b8ce8eb4971e17a3a3/pkg/system/filesys_windows.go#L103) for an example. This step will be performed by the provisioning script.

#### Kube-proxy deployment

On Linux, kube-proxy is deployed as a DaemonSet in the kubeadm init phase. We will supply a DaemonSet that will run on Windows and use the wins API to launch kube-proxy.

#### CNI plugin deployment

On Linux, CNI plugins are deployed via kubectl and run as a DaemonSet. We will
supply DaemonSets that will allow users to run Flannel configured in either
L2Bridge/Host-Gateway or VXLAN/Overlay mode. These ideally can be upstreamed into the Flannel
project.

If users wish to use a different plugin they can create a DaemonSet to be applied in place of the Flannel DaemonSet.

### Risks and Mitigations

**Risk**: Wins proxy introduces new security vector

The same functionality that allows us to now run privileged DaemonSets on Windows could be used maliciously to perform
unwanted behavior on Windows nodes. This brings to Windows problems that already exist on Linux and now require the same
mitigations, namely Pod Security Policies (PSP).

*Mitigation*: Access to the wins named pipe can be restricted using a PSP that either disables
`hostPath` volume mounts or restricts the paths that can be mounted. A sample PSP will be provided.

**Risk**: Versioning of the script can become complicated

Versioning of the script per-Kubernetes version can become a problem if a certain new version diverges in terms of download URLs, flags and configuration.

*Mitigation*: Use git branches to version the script in the repository where it is hosted.

**Risk**: Permissions on Windows paths that kubeadm generates can pose a security hole.

kubeadm creates directories using MakeAll() and such directories are strictly Linux originated for the time being - such as /etc.

On Windows, the creation of such a path can result in sensitive files to be exposed without the right permissions.

*Mitigation*: Create ACLs from the provisioning script that give similar access controls to those on Linux.

## Design Details

### Test Plan

e2e testing for kubeadm on Windows will be performed using Cluster API on AWS.
The CI signal will be owned by SIG Windows.

### Graduation Criteria

This proposal targets *Alpha* support for kubeadm based Windows worker nodes in the release of Kubernetes 1.16.


##### Alpha -> Beta Graduation

Kube-proxy and CNI plugins are run as Kubernetes pods.
The feature is maintained by active contributors.
The feature is tested by the community and feedback is adapted with changes.
e2e tests will be published but may might not be completely green.
Documentation is in a good state. Kubeadm documentation is edited to point to documentation provided by SIG Windows.
Kubeadm upgrade is implemented.

##### Beta -> GA Graduation

The feature is well tested and adapted by the community.
e2e tests are stable and consistent with other SIG-Windows CI signals.
Documentation is complete.

### Upgrade / Downgrade Strategy

The provisioning script will be updated as necessary to support newer versions of kubeadm and kubelet, but ideally will
be parameterized such that it's not heavily tied to specific versions of Kubernetes. Kubeadm doesn't support downgrades and
so that is out of scope for this feature.

### Version Skew Strategy

Kubeadm's version skew policy will apply to this feature as well.

## Implementation History

* April 2019 (1.14) KEP was created. 
* May 1, 2019       KEP was updated to address PR feedback
* May 3, 2019       KEP was updated to address PR feedback
* May 17, 2019      [PR 77989](https://github.com/kubernetes/kubernetes/pull/77989) Remove Powershell dependency for IsPrivilegedUser check on Windows
* May 24, 2019      [PR 78053](https://github.com/kubernetes/kubernetes/pull/78053) Implement CRI detection for Windows
* May 29, 2019      [PR 1136](https://github.com/coreos/flannel/pull/1136) Add net-config-path to FlannelD
* May 31, 2019      [PR 78189](https://github.com/kubernetes/kubernetes/pull/78189) Use Service Control Manager as the Windows Initsystem
* June 3, 2019      [PR 78612](https://github.com/kubernetes/kubernetes/pull/78612) Remove dependency on Kube-Proxy to start after FlannelD
* July 20,2019      KEP was updated to target Alpha for 1.16
* November 1, 2019  [PR 84649](https://github.com/kubernetes/kubernetes/pull/84649) Skip GetPodNetworkStatus when CNI not yet initialized
* January 15th, 2020 KEP was updated to reflect new approach using Wins as a privileged proxy

## Drawbacks 

### Non-identical flows for Linux and Windows
There is overhead to maintaining two different methods for joining a node to a cluster depending on the operating system. However, this is necessary to meet the needs of a typical Windows customer.

## Infrastructure Needed

SIG Windows to provide:
* Azure based infrastructure for testing kubeadm worker nodes.

SIG Windows has provided:
* A kubernetes/ org based repository to host the download / wrapper script: [sig-windows-tools](https://github.com/kubernetes-sigs/sig-windows-tools)
