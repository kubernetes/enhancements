---
title: Support Out-of-Tree Azure Cloud Provider
authors:
  - "@andrewsykim"
  - "@dstrebel"
owning-sig: sig-cloud-provider
participating-sigs:
  - sig-azure
reviewers:
  - "@dstrebel"
  - "@justaugustus"
  - "@khenidak"
  - "@feiskyer"
approvers:
  - "@feiskyer"
  - "@khenidak"
  - "@hogepodge"
  - "@jagosan"
editor: TBD
creation-date: 2019-01-29
last-updated: 2019-01-29
status: provisional
---

# Supporting Out-of-Tree Azure Cloud Provider

## Table of Contents

- [Supporting Out-of-Tree Azure Cloud Provider](#supporting-out-of-tree-azure-cloud-provider)
  - [Table of Contents](#table-of-contents)
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
  - [Technical Leads are members of the Kubernetes Organization](#technical-leads-are-members-of-the-kubernetes-organization)
  - [Subproject Leads](#subproject-leads)
  - [Meetings](#meetings)

## Release Signoff Checklist

- [X] k/enhancements issue in release milestone and linked to KEP (https://github.com/kubernetes/enhancements/issues/667)
- [ ] KEP approvers have set the KEP status to `implementable`
- [ ] Design details are appropriately documentedbs
- [ ] Test plan is in place, giving consideration to SIG Architecture and SIG Testing input
- [ ] Graduation criteria is in place
- [ ] "Implementation History" section is up-to-date for milestone
- [ ] Supporting documentation e.g., additional design documents, links to mailing list discussions/SIG meetings, relevant PRs/issues, release notes

## Summary

Build support for the out-of-tree Azure cloud provider. This involves a well-tested version of the cloud-controller-manager 
that has feature parity to the kube-controller-manager.

## Motivation

Motivation for supporting out-of-tree providers can be found in [KEP-0002](https://github.com/kubernetes/enhancements/blob/master/keps/sig-cloud-provider/0002-cloud-controller-manager.md). 
This KEP is specifically tracking progress for the Azure cloud provider.

### Goals

* Develop/test/release the Azure cloud-controller-manager
* Kubernetes clusters running on Azure should be running the cloud-controller-manager.

### Non-Goals

* Removing in-tree Azure cloud provider code, this effort falls under the [KEP for removing in-tree providers](https://github.com/kubernetes/enhancements/blob/master/keps/sig-cloud-provider/2019-01-25-removing-in-tree-providers.md).

## Proposal
We propose a repository from the Kubernetes organization to host our cloud provider implementation.  The Cloud Provider for Microsoft Azure would be a subproject under Kubernetes community.

### Implementation Details/Notes/Constraints [optional]

TODO for SIG-Azure

### Risks and Mitigations

TODO for SIG-Azure

## Design Details

### Test Plan

Azure Cloud Controller provider is reporting conformance test results to TestGrid as per the [Reporting Conformance Test Results to Testgrid KEP](https://github.com/kubernetes/enhancements/blob/master/keps/sig-cloud-provider/0018-testgrid-conformance-e2e.md).
 See [report](https://testgrid.k8s.io/sig-azure-master#azure-master-conformance) for more details.

### Graduation Criteria

TODO for SIG-Azure

### Upgrade / Downgrade Strategy

TODO for SIG-Azure

### Version Skew Strategy

TODO for SIG-Azure

## Implementation History

TODO for SIG-Azure

## Technical Leads are members of the Kubernetes Organization

The Leads run operations and processes governing this subproject.

* @khenidak
* @feiskyer

## Subproject Leads

* @dstrebel
* @justaugustus

## Meetings

Sig-Azure meetings is expected to have biweekly. SIG Cloud Provider will provide zoom/youtube channels as required. We will have our first meeting after repo has been settled.

Meeting Time: Wednesdays at 09:00 PT (Pacific Time) (biweekly). [Convert to your timezone](http://www.thetimezoneconverter.com/?t=20:00&tz=PT%20%28Pacific%20Time%29).
- Meeting notes and Agenda.
- Meeting recordings.