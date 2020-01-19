---
title: Support Out-of-Tree vSphere Cloud Provider
authors:
  - "@frapposelli"
  - "@andrewsykim"
owning-sig: sig-cloud-provider
reviewers:
  - "@frapposelli"
  - "@cantbewong"
  - "@andrewsykim"
  - "@dvonthenen"
approvers:
  - "@frapposelli"
  - "@cantbewong"
  - "@andrewsykim"
  - "@dvonthenen"
editor: TBD
creation-date: 2019-01-25
last-updated: 2019-01-25
status: implementable
---

# Supporting Out-of-Tree vSphere Cloud Provider

## Table of Contents

<!-- toc -->
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
<!-- /toc -->

## Release Signoff Checklist

- [X] k/enhancements issue in release milestone and linked to KEP (https://github.com/kubernetes/enhancements/issues/670)
- [X] KEP approvers have set the KEP status to `implementable`
- [X] Design details are appropriately documented
- [X] Test plan is in place, giving consideration to SIG Architecture and SIG Testing input
- [X] Graduation criteria is in place
- [X] "Implementation History" section is up-to-date for milestone
- [X] Supporting documentation e.g., additional design documents, links to mailing list discussions/SIG meetings, relevant PRs/issues, release notes

## Summary

Build support for the out-of-tree vSphere cloud provider. This involves a well-tested version of the cloud-controller-manager 
that has feature parity to the kube-controller-manager.  This KEP captures mostly implemented work already completed in the
[Cloud Provider vSphere repository](https://github.com/kubernetes/cloud-provider-vsphere).


## Motivation

Motivation for supporting out-of-tree providers can be found in the [Cloud Controller Manager KEP](/keps/sig-cloud-provider/20180530-cloud-controller-manager.md). 
This KEP is specifically tracking progress for the vSphere cloud provider.

### Goals

* Develop/test/release the vSphere cloud-controller-manager
* Kubernetes clusters running on vSphere should be running the cloud-controller-manager.

### Non-Goals

* Removing in-tree vSphere cloud provider code, this effort falls under the [KEP for removing in-tree providers](https://github.com/kubernetes/enhancements/blob/master/keps/sig-cloud-provider/2019-01-25-removing-in-tree-providers.md).

## Proposal

The vSphere Cloud Provider is implemented, tested and partially documented. The next major steps are to migrate existing users of the in-tree provider to the external provider.

### Implementation Details/Notes/Constraints [optional]

Main provider work is completed and feature parity with in-tree will be achieved with the beta version. Removing in-tree `vsphere_volume` code is underway in the same repo and will piggyback on and be tracked through the
[In-tree Storage Migration to CSI Plugin Migration](https://github.com/kubernetes/community/blob/master/contributors/design-proposals/storage/csi-migration.md)
work. New features will need to follow the community KEP process.

### Risks and Mitigations

None known

## Design Details

### Test Plan

Third-party testing of the vSphere Cloud Provider is handled by VMware and reports to testgrid.

### Graduation Criteria

This feature is complete and will be ready for graduation with release 1.14.

### Upgrade / Downgrade Strategy

Projects that depend on the in-tree cloud provider can retain support for older versions of the provider that ships with Kubernetes for the time being. New deployment tooling is strongly encouraged to migrate to the external provider for future Kubernetes releases. A special consideration was also made for maintaining backward compatibility with in-tree vSphere cloud provider configuration file, an automation tool to upgrade from in-tree to out-of-tree provider was also created: [vcpctl](https://github.com/kubernetes/cloud-provider-vsphere/tree/master/cmd/vcpctl).

### Version Skew Strategy

As such Cloud Provider vSphere has no version skew strategy for migrating from in-tree to out-of-tree providers. Future skew will use the facilities available to the Cloud Controller Manager interface.

## Implementation History

- Implementation and testing completed in 2018.
- Feature parity with in-tree to be achieved with the 1.14 release.
