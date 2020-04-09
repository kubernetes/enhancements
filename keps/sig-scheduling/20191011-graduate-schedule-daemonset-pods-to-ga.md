---
title: Graduate ScheduleDaemonSetPods to GA
authors:
  - "@draveness"
owning-sig: sig-scheduling
participating-sigs:
  - sig-apps
reviewers:
  - "@k82cn"
  - "@janetkuo"
approvers:
  - "@k82cn"
editor: TBD
creation-date: 2019-10-11
last-updated: 2019-10-11
status: implemented
see-also:
  - "https://docs.google.com/document/d/10Ch3dhD88mnHYTq9q4jtX3e9e6gpndC78g5Ea6q4JY4/edit#heading=h.dtxm02f9bgaw"
---

# Graduate ScheduleDaemonSetPods to GA

## Table of Contents

<!-- toc -->
- [Release Signoff Checklist](#release-signoff-checklist)
- [Summary](#summary)
- [Motivation](#motivation)
  - [Goals](#goals)
  - [Non-Goals](#non-goals)
- [Proposal](#proposal)
  - [Implementation Details/Notes/Constraints](#implementation-detailsnotesconstraints)
  - [Risks and Mitigations](#risks-and-mitigations)
- [Design Details](#design-details)
  - [Test Plan](#test-plan)
    - [Existing Tests](#existing-tests)
  - [Graduation Criteria](#graduation-criteria)
- [Implementation History](#implementation-history)
<!-- /toc -->

## Release Signoff Checklist

- [x] kubernetes/enhancements issue in release milestone, which links to KEP (this should be a link to the KEP location in kubernetes/enhancements, not the initial KEP PR)
- [x] KEP approvers have set the KEP status to `implementable`
- [x] Design details are appropriately documented
- [x] Test plan is in place, giving consideration to SIG Architecture and SIG Testing input
- [x] Graduation criteria is in place
- [x] "Implementation History" section is up-to-date for milestone
- [x] User-facing documentation has been created in [kubernetes/website], for publication to [kubernetes.io]
- [x] Supporting documentation e.g., additional design documents, links to mailing list discussions/SIG meetings, relevant PRs/issues, release notes

## Summary

ScheduleDaemonSetPods has been created in the past to schedule DaemonSet Pods by the default scheduler. We wish to graduate ScheduleDaemonSetPods feature to make scheduling decisions only in the scheduler and remove the scheduling related code in the DaemonSetController.

## Motivation

ScheduleDaemonSetPods has been beta'ed in 1.12.

### Goals

+ Plan to promote ScheduleDaemonSetPods to the stable version.
+ Remove scheduling related codes in DaemonSetController.

### Non-Goals

+ Changing API field or meaning

## Proposal

### Implementation Details/Notes/Constraints

ScheduleDaemonSetPods attaches node affinity to DaemonSet pods, which let the default kubernetes scheduler to schedule pods on specific nodes. The DaemonSet controller uses several scheduler predicates to calculate the nodes which need to schedule DaemonSet pods and create pods with specific NodeAffinity.

### Risks and Mitigations

The major concern for graduating ScheduleDaemonSetPods to the stable version could be the overhead to the scheduler and the startup time of daemons. After we graduate this feature, the scheduler would select nodes for all of the DaemonSet pods, which may cause a lot of pods with NodeAffinity to be processed.

## Design Details

### Test Plan

#### Existing Tests

ScheduleDaemonSetPods currently has multiple tests in various components that use the feature.

### Graduation Criteria

**Note:** *Section not required until targeted at a release.*

- [x] Graduate ScheduleDaemonSetPods to GA
- [x] Remove suspenedDaemonPods which handles Pod deleted on the nodes
- [x] Refactor nodeShouldRunDaemonPod to remove useless return values
- [x] Update documents to reflect the changes

## Implementation History

+ ScheduleDaemonSetPods was introduced in Kubernetes 1.11 as an alpha version.
+ ScheduleDaemonSetPods was graduated to beta in Kubernetes 1.12.
