---
title: Node Maintenance Controller
authors:
  - "@yanirq"
owning-sig: sig-node
participating-sigs:
  - sig-node
  - sig-scheduling
reviewers:
  - TBD
approvers:
  - TBD
editor: "@yanirq"
creation-date: 2019-05-28
last-updated: 2019-06-02
status: provisional

---

# Node Maintenance Controller

## Table of Contents

A table of contents is helpful for quickly jumping to sections of a KEP and for highlighting any additional information provided beyond the standard KEP template.
[Tools for generating][] a table of contents from markdown are available.

- [Title](#title)
  - [Release Signoff Checklist](#release-signoff-checklist)
  - [Summary](#summary)
  - [Motivation](#motivation)
    - [Goals](#goals)
    - [Non-Goals](#non-goals)
  - [Proposal](#proposal)
    - [User Stories](#user-stories)
      - [Story 1](#story-1)
      - [Story 2](#story-2)
      - [Story 3](#story-2)
      - [Story 4](#story-2)
      - [Story 5](#story-2) 
      - [Story 6](#story-2)                       
    - [Implementation Details/Notes/Constraints](#implementation-detailsnotesconstraints-optional)
    - [Risks and Mitigations](#risks-and-mitigations)
  - [Design Details](#design-details)
    - [Test Plan](#test-plan)
    - [Graduation Criteria](#graduation-criteria)
    - [Upgrade / Downgrade Strategy](#upgrade--downgrade-strategy)
    - [Version Skew Strategy](#version-skew-strategy)
  - [Implementation History](#implementation-history)
  - [Drawbacks [optional]](#drawbacks-optional)
  - [Alternatives [optional]](#alternatives-optional)
  - [Infrastructure Needed [optional]](#infrastructure-needed-optional)

[Tools for generating]: https://github.com/ekalinin/github-markdown-toc

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

Node maintenance controller that performs node drain - server side using custom resources calls to the API in order to perform the drain (cordon + evictions)

## Motivation

Move kubectl drain into the server to enable execution of node drain using custom resources calls to the API in order to perform the drain.
This would also reduce the proliferation of drain behaviours in other projects that re-implement the drain logic to accustom their needs.

(https://github.com/kubernetes/kubernetes/issues/25625)

### Goals

- A new controller that implements the drain logic that is currently [implemented](https://github.com/kubernetes/kubernetes/blob/v1.14.0/pkg/kubectl/cmd/drain/drain.go#L307) in [kubectl](https://kubernetes.io/docs/reference/generated/kubectl/kubectl-commands#drain).
- The controller watches and acts upon maintenance CRs to put/remove a node on/from maintenance.
- Status report of the drain (embodied in the maintenance CR).
- Custom taints and labels added to the node going under maintenance.
- Configurable time-out(time buffer) for the drain process.

### Non-Goals
- "Dry run" mode to detect what is the best node / N nodes to drain

## Proposal

Node maintenance controller watches for new or deleted custom resources e.g. `NodeMaintenance` which indicate that a node in the cluster should either:
 - `NodeMaintenance` CR created: move node into maintenance, cordon the node - set it as unscheduleble and evict the pods (which can be evicted) from that node.
  - `NodeMaintenance` CR deleted: remove pod from maintenance and uncordon the node - set it as scheduleble.

Current implementation and design can be found under Kubevirt:
https://github.com/kubevirt/node-maintenance-operator 

### User Stories

#### Story 1
As a third party developer/operator I want to evacuate my workloads appropriately when a node goes into maintenance mode, but I do not want to evacuate them when the node is cordoned.

- Today we can not differentiate between cordon and drain using cluster information

- Adding taints/labels as part of the evacuation process can help performing additional work before a pod is removed from a node.

#### Story 2
As a third party developer/operator I would like to activate the node back simply by removing the maintenance mode CR.


#### Story 3
As an operator I need a reliable way to request to put a node into maintenance mode and understand if it is - or not - in order to perform i.e. hardware maintenance work.

- CR needs to report pending/running/success/failure as well as informative conditions in the CR’s Status section. The conditions should provide information about what is blocking maintenance node in the event maintenance mode is can not proceed.

#### Story 4
As an operator I need to put a number of nodes i.e. a rack of nodes into node maintenance.

- As a cluster admin i would like to shutdown a rack (due to overheating problem) to do that I would need to move hosts to maintenance.

#### Story 5
- As a cluster admin, i would like to perform a firmware update on the nodes.

- As a cluster admin i would like to perform a bios update for the nodes in the cluster.

#### Story 6
- As a developer/operator I would like a way to invoke node maintenance (drain) not only from CLI but also from UI.

### Implementation Details/Notes/Constraints

- Custom taints added to the node by the controller might be overun by other controllers such as the node life cycle controller for the `TaintNodesByCondition` feature or the node controller for the `TaintBasedEvictions` feature.

### Risks and Mitigations

TBD

## Design Details

TBD

### Test Plan

To ensure this feature to be rolled out in high quality. Following tests are mandatory:

- **Unit Tests:** All core changes must be covered by unit tests.
- **Integration Tests / E2E Tests:** All user cases discussed in this KEP must be covered by either integration tests or e2e tests.

### Graduation Criteria

TBD

### Upgrade / Downgrade Strategy

TBD

### Version Skew Strategy

TBD

## Implementation History

- 2019-06-02: Initial KEP sent out for review.

## Drawbacks [optional]

TBD

## Alternatives [optional]

TBD

## Infrastructure Needed [optional]

TBD

