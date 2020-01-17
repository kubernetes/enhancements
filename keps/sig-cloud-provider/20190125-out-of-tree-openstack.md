---
title: Support Out-of-Tree OpenStack Cloud Provider
authors:
  - "@andrewsykim"
  - "@adisky"
owning-sig: sig-cloud-provider
participating-sigs:
  - sig-openstack
reviewers:
  - "@lingxiankong"
  - "@chrigl"
  - "@ramineni"
  - "@kendallnelson"
approvers:
  - TBD
editor: TBD
creation-date: 2019-01-25
last-updated: 2019-01-25
status: provisional
---

# Supporting Out-of-Tree OpenStack Cloud Provider

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

- [X] k/enhancements issue in release milestone and linked to KEP (https://github.com/kubernetes/enhancements/issues/669)
- [X] KEP approvers have set the KEP status to `implementable`
- [X] Design details are appropriately documentedbs
- [X] Test plan is in place, giving consideration to SIG Architecture and SIG Testing input
- [ ] Graduation criteria is in place
- [ ] "Implementation History" section is up-to-date for milestone
- [ ] Supporting documentation e.g., additional design documents, links to mailing list discussions/SIG meetings, relevant PRs/issues, release notes

## Summary

Build support for the out-of-tree OpenStack cloud provider. This involves a well-tested version of the cloud-controller-manager 
that has feature parity to the kube-controller-manager. 

## Motivation

Motivation for supporting out-of-tree providers can be found in the [Cloud Controller Manager KEP](/keps/sig-cloud-provider/20180530-cloud-controller-manager.md). 
This KEP is specifically tracking progress for the OpenStack cloud provider.

### Goals

* Develop/test/release the OpenStack cloud-controller-manager
* Kubernetes clusters running on OpenStack should be running the cloud-controller-manager.

### Non-Goals

* Removing in-tree OpenStack cloud provider code, this effort falls under the [KEP for removing in-tree providers](https://github.com/kubernetes/enhancements/blob/master/keps/sig-cloud-provider/2019-01-25-removing-in-tree-providers.md).

## Proposal
The OpenStack Cloud Provider is implemented, tested and documented, It is being released with matching kubernetes version from [release v1.11](https://github.com/kubernetes/cloud-provider-openstack/releases). cloud-provider-openstack release 1.14, 1.15, 1.16 has been running in production.

### Implementation Details/Notes/Constraints [optional]
OpenStack Cloud Provider is implemented [here](https://github.com/kubernetes/cloud-provider-openstack/releases). This repository also hosts other drivers like CSI, ingress-controller etc. Cloud Provider OpenStack is in feature parity with the intree version. Removal of intree cloud providers is dependent on [In-tree Storage Migration to CSI Plugin Migration](https://github.com/kubernetes/community/blob/master/contributors/design-proposals/storage/csi-migration.md). The migration work for intree OpenStack Volume driver to CSI Driver is in progress.

### Risks and Mitigations

None as of now
## Design Details

### Test Plan
OpenStack Cloud Provider is well tested, CI running at [OpenLab](https://github.com/theopenlab/openlab-zuul-jobs), results are reported to kubernetes test grid.

### Graduation Criteria

This feature is complete, well tested, in parity with intree openstack provider. Documents needs to be updated as per sig-cloud-provider guideline. 

### Upgrade / Downgrade Strategy

TODO for SIG-OpenStack
 
### Version Skew Strategy

TODO for SIG-OpenStack

## Implementation History
- Implementation and testing completed in 2018.
- Matching kubernetes versions released from v1.11, latest version is v1.16.
