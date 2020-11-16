---
title: Remove ClusterStatus from kubeadm-config
authors:
  - "@fabriziopandini"
owning-sig: sig-cluster-lifecycle
participating-groups:
  - sig-cluster-lifecycle
reviewers:
  - "@neolit123"
  - "@rosti"
  - "@ereslibre"
  - "@ncdc"
approvers:
  - "@timothysc"
editor: "@fabriziopandini"
creation-date: 2019-11-25
last-updated: 2019-11-25
status: implementable
---

# Remove ClusterStatus from kubeadm-config

## Table of Contents

<!-- TOC -->
- [Release Signoff Checklist](#release-signoff-checklist)
- [Summary](#summary)
- [Motivation](#motivation)
  - [Goals](#goals)
  - [Non-Goals](#non-goals)
- [Proposal](#proposal)
  - [Implementation Details/Notes/Constraints [optional]](#implementation-detailsnotesconstraints-optional)
  - [Risks and Mitigations](#risks-and-mitigations)
- [Design Details](#design-details)
  - [Test Plan](#test-plan)
  - [Graduation Criteria](#graduation-criteria)
  - [Upgrade / Downgrade Strategy](#upgrade--downgrade-strategy)
  - [Version Skew Strategy](#version-skew-strategy)
- [Implementation History](#implementation-history)
<!-- /TOC -->

## Release Signoff Checklist

- [x] kubernetes/enhancements issue in release milestone, which links to KEP (this should be a link to the KEP location in kubernetes/enhancements, not the initial KEP PR)
- [x] KEP approvers have set the KEP status to `implementable`
- [x] Design details are appropriately documented
- [x] Test plan is in place, giving consideration to SIG Architecture and SIG Testing input
- [x] Graduation criteria is in place
- [x] "Implementation History" section is up-to-date for milestone
- [ ] User-facing documentation has been created in [kubernetes/website], for publication to [kubernetes.io]
- [ ] Supporting documentation e.g., additional design documents, links to mailing list discussions/SIG meetings, relevant PRs/issues, release notes

**Note:** Any PRs to move a KEP to `implementable` or significant changes once it is marked `implementable` should be approved by each of the KEP approvers. If any of those approvers is no longer appropriate than changes to that list should be approved by the remaining approvers and/or the owning SIG (or SIG-arch for cross cutting KEPs).

**Note:** This checklist is iterative and should be reviewed and updated every time this enhancement is being considered for a milestone.

[kubernetes.io]: https://kubernetes.io/
[kubernetes/enhancements]: https://github.com/kubernetes/enhancements/issues
[kubernetes/kubernetes]: https://github.com/kubernetes/kubernetes
[kubernetes/website]: https://github.com/kubernetes/website

## Summary

This KEP is proposing a new mode for tracking the list of the API endpoints in a cluster, thus allowing to remove the  `ClusterStatus` entry in the `kubeadm-config` ConfigMap and solve the problems that arise when, for any reasons, such entry does not reflect anymore the real status of the cluster.

## Motivation

In order to manage HA cluster properly, kubeadm requires to have access to the list of API endpoints in a cluster.

Currently this feature is implemented by adding an entry in the list of API endpoints that is stored in the `ClusterStatus` entry of the `kubeadm-config` ConfigMap.

There are well known problem related to the management of this list e.g. when a control-plane node dies or is deleted without invoking `kubeadm reset`, the list gets stale and the user is required to manually cleanup the list in order to avoid any kubeadm operation that relies on such list might incur into errors.

This KEP is going to propose a different mode for tracking the list of the API endpoints in a cluster, based on the inspection of the current Pods.

This approach does not require the maintenance of a separated list, and implicitly always reflect the current status of the cluster.

This allows to remove the `ClusterStatus` entry in the `kubeadm-config` ConfigMap and to clean-up all the related code in `kubeadm init`, `kubeadm join` and `kubeadm reset`

### Goals

- To introduce a new method for tracking the list of API endpoints in a cluster
- To allow removal of the `ClusterStatus` entry in the `kubeadm-config` ConfigMap and clean-up of the related code.

### Non-Goals

- To change any user facing behavior in `kubeadm`.
  However, considering that we are going to remove the `ClusterStatus` in the `kubeadm-config` ConfigMap and potentially some user/higher level tools can rely on it, we are going issue and "Action Required" warning in the release note an respect the Beta API deprecation policy before actual deletion (9 months/3 release).

## Proposal

### Implementation Details/Notes/Constraints [optional]

As of today, the `ClusterStatus` entry in the `kubeadm-config` ConfigMap contains a map that stores the `LocalAPIEndpoint` for each control-plane node.

The `LocalAPIEndpoint` primary usage is for the `advertise-address`:`bindPort` flag in the `kube-apiserver` pod. Having this value in a flag is not ideal for the purpose of this proposal, so, we are going to echo the same value into a new annotation in the `kube-apiserver` pod manifest.
The annotation will be named `kubeadm.kubernetes.io/kube-apiserver.advertise-address.endpoint` and the value will be `advertise-address`:`bindPort`.

Once the annotation will be in place, it will be possible to easily retrieve the local advertise address for each control plane node by querying the corresponding `kube-apiserver` pod.

The `LocalAPIEndpoint` is also used in the stacked `etcd` pod manifest for composing the `peer-urls` and the `client-urls`; the latter is used by kubeadm when accessing etcd in an existing cluster, e.g. when doing `join --control-plane`.

We are going to echo the `client-urls` value into a new annotation named `kubeadm.kubernetes.io/etcd.advertise-client-urls`. Once the annotation will be in place, it will be possible to easily retrieve the etcd client urls by querying the `etcd` pods. Note: Every `etcd` pod's annotation will list only its own client urls.

### Risks and Mitigations

R. The list of API endpoints in a cluster is crucial to all the kubeadm workflows
M. The proper function of those workflows and of the underlying codes is already covered by E2E tests; on top of that, we are going to try to implement this change at the beginning of the v1.18 cycle, thus ensuring as much
test cycles as possible.

R. We are relying on the kubelet signal to check if a static pod goes down.
M. The fact that there is a long-running agent doing this check is already an improvement vs current state; However, during the implementation, we should try to make this smarter as possible, e.g checking the pod status or any type of feedback the kubelet provides.

## Design Details

### Test Plan

All the affected behaviors are already covered by existing E2E test; however, we should consider if to add new destructive tests in order to exercise properly all the possible conditions.

Additional unit test are required only for the new function implementing the inspection of the current Pods.

### Graduation Criteria

Alpha - Does not apply (this is part of how kubeadm handles it's configuration API, and currently this is already in Beta).
Beta - Initial status
GA - This will graduate as part of the Kubeadm Configuration API.

### Upgrade / Downgrade Strategy

During upgrades:

- The new annotations `kubeadm.kubernetes.io/kube-apiserver.advertise-address` and `kubeadm.kubernetes.io/etcd.advertise-client-urls` will be generated during the upgrade of the static pod manifests.
- The `ClusterStatus` entry will be cleaned up during the upgrade of the `kubeadm-config` ConfigMap.

Downgrade are not supported by kubeadm.

### Version Skew Strategy

NA

## Implementation History

- the `Summary` and `Motivation` sections being merged signaling SIG acceptance
- the `Proposal` section being merged signaling agreement on a proposed design
- the date implementation started
