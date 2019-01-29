---
title: Support Out-of-Tree OpenStack Cloud Provider
authors:
  - @andrewsykim
owning-sig: sig-cloud-provider
participating-sigs:
  - sig-openstack
reviewers:
  - @dims
approvers:
  - @hogepodge
  - @flaper87
editor: @hogepodge
creation-date: 2019-01-25
last-updated: 2019-01-25
status: provisional

---

# Supporting Out-of-Tree OpenStack Cloud Provider

## Table of Contents

* [Table of Contents](#table-of-contents)
* [Release Signoff Checklist](#release-signoff-checklist)
* [Summary](#summary)
* [Motivation](#motivation)
   * [Goals](#goals)
   * [Non-Goals](#non-goals)
* [Proposal](#proposal)
   * [Implementation Details/Notes/Constraints [optional]](#implementation-detailsnotesconstraints-optional)
   * [Risks and Mitigations](#risks-and-mitigations)
* [Design Details](#design-details)
   * [Test Plan](#test-plan)
   * [Graduation Criteria](#graduation-criteria)
   * [Upgrade / Downgrade Strategy](#upgrade--downgrade-strategy)
   * [Version Skew Strategy](#version-skew-strategy)
* [Implementation History](#implementation-history)

## Release Signoff Checklist

- [X] k/enhancements issue in release milestone and linked to KEP (https://github.com/kubernetes/enhancements/issues/669)
- [X] KEP approvers have set the KEP status to `implementable`
- [X] Design details are appropriately documented
- [X] Test plan is in place, giving consideration to SIG Architecture and SIG Testing input
- [ ] Graduation criteria is in place
- [ ] "Implementation History" section is up-to-date for milestone
- [ ] Supporting documentation e.g., additional design documents, links to mailing list discussions/SIG meetings, relevant PRs/issues, release notes

## Summary

Build support for the out-of-tree OpenStack cloud provider. This involves a well-tested version of the cloud-controller-manager
that has feature parity to the kube-controller-manager. This KEP captures mostly implemented work already completed in the
[Cloud Provider OpenStack repository](https://github.com/kubernetes/cloud-provider-openstack)

## Motivation

Motivation for supporting out-of-tree providers can be found in [KEP-0002](https://github.com/kubernetes/enhancements/blob/master/keps/sig-cloud-provider/0002-cloud-controller-manager.md). 
This KEP is specifically tracking progress for the OpenStack cloud provider.

### Goals

* Develop/test/release the OpenStack cloud-controller-manager
* Kubernetes clusters running on OpenStack should be running the cloud-controller-manager.

### Non-Goals

* Removing in-tree OpenStack cloud provider code, this effort falls under the [KEP for removing in-tree providers](https://github.com/kubernetes/enhancements/blob/master/keps/sig-cloud-provider/2019-01-25-removing-in-tree-providers.md).

## Proposal

The OpenStack Cloud Provider is implemented, tested and documented. The next major steps are to migrate existing users of the in-tree provider to the external provider.

### Implementation Details/Notes/Constraints [optional]

Main provider work is completed. Removing in-tree Cinder code will piggyback on and be tracked through the
[In-tree Storage Migration to CSI Plugin Migration](https://github.com/kubernetes/community/blob/master/contributors/design-proposals/storage/csi-migration.md)
work. New features will need to follow the community KEP process.

### Risks and Mitigations

None known

## Design Details

### Test Plan

Third-party testing of the OpenStack Cloud Provider is handled by Open Lab and reports to testgrid.

### Graduation Criteria

This feature is complete and ready for graduation.

### Upgrade / Downgrade Strategy

Projects like OpenStack Magnum that depend on the in-tree cloud provider can retain support for older versions of the provider that ships with Kubernetes up to 1.13. New deployment tooling must migrate to the external provider for future Kubernetes releases.

### Version Skew Strategy

As such Cloud Provider OpenStack has no version skew strategy for migrating from in-tree to out-of-tree providers. Future skew will use the facilities available to the Cloud Controller Manager interface.

## Implementation History

Implementation and testing completed in 2018.

