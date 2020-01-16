---
title: Inplace Update Pods For Apps
authors:
  - "@skilxn-go"
owning-sig: sig-apps
participating-sigs:

reviewers:
  - "@janetkuo"
  - "@kow3ns"
approvers:
  - "@janetkuo"
  - "@kow3ns"
editor: TBD
creation-date: 2020-01-16
last-updated: 2020-01-16
status: provisional
see-also:
  - n/a
replaces:
superseded-by:
---

# Inplace Update Pod For Apps

## Table of Contents

<!-- toc -->
- [Release Signoff Checklist](#release-signoff-checklist)
- [Summary](#summary)
- [Motivation](#motivation)
  - [Goals](#goals)
  - [Non-Goals](#non-goals)
- [Proposal](#proposal)
  - [User Stories [optional]](#user-stories-optional)
  - [Implementation Details/Notes/Constraints [optional]](#implementation-detailsnotesconstraints-optional)
  - [Risks and Mitigations](#risks-and-mitigations)
- [Design Details](#design-details)
  - [Test Plan](#test-plan)
  - [Graduation Criteria](#graduation-criteria)
  - [Upgrade / Downgrade Strategy](#upgrade--downgrade-strategy)
  - [Version Skew Strategy](#version-skew-strategy)
- [Implementation History](#implementation-history)
<!-- /toc -->

## Release Signoff Checklist

- [ ] kubernetes/enhancements issue in release milestone, which links to KEP (this should be a link to the KEP location in kubernetes/enhancements, not the initial KEP PR)
- [ ] KEP approvers have set the KEP status to `implementable`
- [ ] Design details are appropriately documented
- [ ] Test plan is in place, giving consideration to SIG Architecture and SIG Testing input
- [ ] Graduation criteria is in place
- [ ] "Implementation History" section is up-to-date for milestone
- [ ] User-facing documentation has been created in [kubernetes/website], for publication to [kubernetes.io]
- [ ] Supporting documentation e.g., additional design documents, links to mailing list discussions/SIG meetings, relevant PRs/issues, release notes

## Summary

It's a feature for Statefulset, ReplicaSet, DaemonSet and Deployment pods to do inplace update when people only update the container-image field of pod template spec. It will make update of apps more smoothing and reduce the pressure of scheduler, and it will keep user files in empty-dir during update processes.

## Motivation

Currently, apps updating is done by destroying and recreating pods one by one (statefulsets) or in-batch (deployments). Destroying and recreating method may suit most user cases, but in some scenarios, it hurts the availability of services and applications. For example:

- There are many pods running applications with one or more sidecar containers, such as envoy, nginx, or other services. If people want to update the sidecar, they must pay for restarting there main containers. If it's a huge image, and it's scheduled to another node, it will cost a long time for this pod's recreating.
- Some pods may save cache data in local empty-dirs. An recreating update will cause the lost of these data, and add more overhead to the updating.
- Recreating a pod means the rescheduling of this pod, which will extensive the pressure of etcd, scheduler, and even kubelet.
- Even the service discovery and load balance will be chruned due to the change of distribute of pods.

Given the above use cases, based on the fact that pod's container-image field can be patched for updating docker images and containers, it a better way to update application pods without recreating them.

### Goals

when people use inplace update strategy and ***only*** change the container image field of pod template in statefulsets, replicasets, daemonsets and deployments:

- Support pod inplace update without recreating them.
- Show pod's inplace update status when using **kubectl**.

### Non-Goals

- Change any behavior of other update strategy.
- Support inplace update when update other fields of the pod template spec or workload spec.

## Proposal



### User Stories [optional]



### Implementation Details/Notes/Constraints [optional]



### Risks and Mitigations



## Design Details

### Test Plan



### Graduation Criteria



### Upgrade / Downgrade Strategy



### Version Skew Strategy



## Implementation History

- 2020-01-16: Add the first draft of KEP with overview sections.
