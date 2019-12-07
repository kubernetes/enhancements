---
title: Support Out-of-Tree AWS Cloud Provider
authors:
  - "@andrewsykim"
owning-sig: sig-cloud-provider
participating-sigs:
  - sig-aws
reviewers:
  - TBD
approvers:
  - TBD
editor: TBD
creation-date: 2019-01-25
last-updated: 2019-01-25
status: provisional
---

# Supporting Out-of-Tree AWS Cloud Provider

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

- [X] k/enhancements issue in release milestone and linked to KEP (https://github.com/kubernetes/enhancements/issues/631)
- [ ] KEP approvers have set the KEP status to `implementable`
- [ ] Design details are appropriately documentedbs
- [ ] Test plan is in place, giving consideration to SIG Architecture and SIG Testing input
- [ ] Graduation criteria is in place
- [ ] "Implementation History" section is up-to-date for milestone
- [ ] Supporting documentation e.g., additional design documents, links to mailing list discussions/SIG meetings, relevant PRs/issues, release notes

## Summary

Build support for the out-of-tree AWS cloud provider. This involves a well-tested version of the cloud-controller-manager 
that has feature parity to the kube-controller-manager. 

## Motivation

Motivation for supporting out-of-tree providers can be found in the [Cloud Controller Manager KEP](/keps/sig-cloud-provider/20180530-cloud-controller-manager.md). 
This KEP is specifically tracking progress for the AWS cloud provider.

### Goals

* Develop/test/release the AWS cloud-controller-manager
* Kubernetes clusters running on AWS should be running the cloud-controller-manager.

### Non-Goals

* Removing in-tree AWS cloud provider code, this effort falls under the [KEP for removing in-tree providers](https://github.com/kubernetes/enhancements/blob/master/keps/sig-cloud-provider/2019-01-25-removing-in-tree-providers.md).

## Proposal

### Implementation Details/Notes/Constraints [optional]

TODO for SIG-AWS

### Risks and Mitigations

TODO for SIG-AWS

## Design Details

### Test Plan

TODO for SIG-AWS

### Graduation Criteria

TODO for SIG-AWS

### Upgrade / Downgrade Strategy

TODO for SIG-AWS

### Version Skew Strategy

TODO for SIG-AWS

## Implementation History

TODO for SIG-AWS
