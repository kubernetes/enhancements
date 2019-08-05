---
title: kubeadm-for-windows
authors:
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

* Create and maintain a Powershell script to install and run Kubernetes prerequisites on Windows
* Support kubeadm join for Windows
* Support kubeadm reset for Windows
* Create and maintain a Powershell script to deploy Windows CNIs

### Non-Goals

* Installing the Container Runtime (e.g. Docker or containerd)
* Implement kubeadm init for Windows (at this time)
* Implement kubeadm join --control-plane for Windows (at this time)
* Supporting upgrades using kubeadm upgrade for Windows (to be revisited for Beta)
* Running kube-proxy as a DaemonSet on Windows (to be revisited for Beta)
* Running Flannel as a DaemonSet on Windows (to be revisited for Beta)

## Proposal

### User Stories

#### Story 1

A user will have a Windows machine that they want to join to an existing Kubernetes cluster.

1. The user will download a set of required binaries (kubelet, kube-proxy, kubeadm, kubectl, Flannel) using a script. The same script will also wrap kubeadm execution.

2. The script will register kubelet and kube-proxy as Windows services.

3. The script will upload a default ConfigMap to the cluster for the default Windows KubeletConfiguration if one does not already exist.

4. The script will run "kubeadm join ..." to join the node to the cluster. In this step kubeadm will run preflight checks and proceed with the regular join process.

5. kubeadm will restart the kubelet service using flags that kubeadm fed to the Windows service. kubeadm will proceed to bootstrap the node. After this process is finished the node should show with the status of NotReady.

6. The same script will then configure FlannelD. The script will (re-)register FlannelD as a service and start it with the correct configuration, optionally using parameters from kubeadm.

7. kube-proxy will do the same steps as FlannelD shortly after.

8. The node status should be Ready.

### Implementation Details/Notes/Constraints

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

Kubeadm makes a number of non-portable assumptions about paths. E.g. “/etc/kubernetes” is a hardcoded path in kubeadm.

We will use "C:\kubernetes" to hold the paths that are normally created for Linux.

We need to evaluate the kubeadm codebase for such instances of non-portable paths - CRI sockets, Cert paths, etc. Such paths need to be defaulted properly in the kubeadm configuration API.

A consolidated list of paths where kubeadm installs files to a set of paths needs to be created and updated to comply with the Windows OS model. At least a single PR against kubeadm will be required to modify the Windows defaults. 

Last, as new paths are created, restrictive Access Control Lists (ACL) for Windows should be applied. Golang does not convert Posix permissions to an appropriate Windows ACL, so an additional step is needed. See [mkdirWithACL from moby/moby](https://github.com/moby/moby/blob/e4cc3adf81cc0810a416e2b8ce8eb4971e17a3a3/pkg/system/filesys_windows.go#L103)) for an example. This step will be performed by kubeadm.

#### Kube-proxy deployment

On Linux, kube-proxy is deployed as a DaemonSet in the kubeadm init phase. However, kube-proxy cannot run as a container in Windows since Windows does not support privileged containers. Kube-proxy should therefore be run as a Windows service so that it is restarted by windows control manager automatically and has lifecycle control.

We need to modify the Linux kube-proxy DaemonSet to not deploy on Windows nodes. A PR is already in flight for that [76327](https://github.com/kubernetes/kubernetes/pull/76327). *Merging this PR is mandatory for this proposal*.

Running kube-proxy as a Windows service from kubeadm is out of scope for this proposal. This is due to the fact that we don’t want the changes in kubeadm to be intrusive to the existing method of running kube-proxy as a DaemonSet on Linux. This can end up requiring an abstraction layer that is far from ideal.

The proposed Windows wrapper script that executes kubeadm will also manage the restart of the kube-proxy Windows service.

Long term and ideally, kube-proxy should be run as a DaemonSet on Windows.

#### CNI plugin deployment

On Linux, CNI plugins are deployed via kubectl and run as a DaemonSet. However, on Windows, CNI plugins need to run on the node, and cannot run in containers (again because Windows does not currently support privileged containers). Azure-CNI, win-bridge (compatible with kubenet), and Flannel all need the binary and config stored on the node.

This proposal plans for FlannelD as the default option. Currently, FlannelD has to be started before the kube-proxy Windows service is started. FlannelD creates an HNS network on the Windows host, and kube-proxy will crash if it cannot find the network. This should be fixed in the scope of this project so that kube-proxy will wait until the network comes up. Therefore, kube proxy can be started at any time. 

However, if FlannelD is deployed in VXLAN (Overlay) mode, then we need to rewrite the KubeProxyConfiguration with the correct Overlay specific values, and kube-proxy will need to read this config again. This is not true for Host-Gateway (L2Bridge) mode. The script will have a flag that allows users to choose between the two networking modes.

If the users wish to use a different plugin they will technically opt-out of the supported setup for kubeadm based Windows worker nodes in the Alpha release.

Long term, any CNI plugin should be supported for kubeadm based Windows worker nodes.

### Risks and Mitigations

**Risk**: Versioning of the wrapper script can become complicated

Versioning of the script per-Kubernetes version can become a problem if a certain new version diverges in terms of download URLs, flags and configuration.

*Mitigation*: Use git branches to version the script in the repository where it is hosted.

**Risk**: The wrapper script is planned to act as both a downloader and runner of the downloader binaries, which might cause scope and maintenance issues.

*Mitigation*: Use separate scripts, the first one downloads the binaries and the wrapper/runner script then setups the environment. The user then executes the wrapper script.

The initial plan is to give the single script method a shot with different arguments that will execute the different stages (downloading, setting up the environment, deploying the CNI).

**Risk**: Flannel or kube-proxy require special configuration that kubeadm does not handle.

*Mitigation*: Allow the user to pass custom configuration files that the wrapper script can feed into the components in question.

**Risk**: Failing or missing preflight checks on Windows

*Mitigation*: The existing kubeadm codebase already has good abstraction in this regard. Still, a PR that makes some non-intrusive adjustments in _windows.go files might be required.

**Risk**: Permissions on Windows paths that kubeadm generates can pose a security hole.

kubeadm creates directories using MakeAll() and such directories are strictly Linux originated for the time being - such as /etc.

On Windows, the creation of such a path can result in sensitive files to be exposed without the right permissions.

*Mitigation*: Provide a SecureMakeAll() func in kubeadm, that ensures secure enough permissions on both Windows & Linux, and replace usage of MakeAll()

## Design Details

### Test Plan

E2e testing for kubeadm on Windows is still being planned.
One available option is to run “kubeadm join” periodically on Azure nodes and federate the reports to test-infra/testgrid.
The CI signal will be owned by SIG Windows.

### Graduation Criteria

This proposal targets *Alpha* support for kubeadm based Windows worker nodes in the release of Kubernetes 1.16.


##### Alpha -> Beta Graduation

Kube-proxy and CNI plugins are run as Kubernetes pods.
The feature is maintained by active contributors.
The feature is tested by the community and feedback is adapted with changes.
Kubeadm join performs complete preflight checks on the host node
E2e tests might not be complete but provide good signal.
Documentation is in a good state. Kubeadm documentation is edited to point to documentation provided by SIG Windows.
Kubeadm upgrade is implemented.

##### Beta -> GA Graduation

The feature is well tested and adapted by the community.
E2e test provide sufficient coverage.
Documentation is complete.

### Upgrade / Downgrade Strategy

Upgrades and downgrades are out of scope for this proposal for 1.16 but will be revisited in future iterations.

### Version Skew Strategy

The existing version skew strategy will apply to Windows worker nodes using kubeadm.
The download scripts will not allow or recommend skewing the version of kube-proxy or the kubelet from the version of kubeadm that is installed by the user.
If the users applies manual skew by diverging from the recommended setup, the node will be claimed as unsupported.

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


## Drawbacks 

### Non-identical flows for Linux and Windows
There is overhead to maintaining two different methods for joining a node to a cluster depending on the operating system. However, this is necessary to meet the needs of a typical Windows customer.

## Infrastructure Needed

SIG Windows to provide:
* Azure based infrastructure for testing kubeadm worker nodes.

SIG Windows has provided:
* A kubernetes/ org based repository to host the download / wrapper script: [sig-windows-tools](https://github.com/kubernetes-sigs/sig-windows-tools)
