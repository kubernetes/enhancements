---
title: Graduate TaintNodeByCondition to GA
authors:
  - "@draveness"
owning-sig: sig-scheduling
participating-sigs:
  - sig-node
reviewers:
  - TBD
  - "@k82cn"
approvers:
  - TBD
  - "@k82cn"
editor: TBD
creation-date: 2019-10-12
last-updated: 2019-10-12
status: implementable
see-also:
  - "https://github.com/kubernetes/community/blob/master/contributors/design-proposals/scheduling/taint-node-by-condition.md"
---

# Graduate TaintNodeByCondition to GA

## Table of Contents

<!-- toc -->
- [Release Signoff Checklist](#release-signoff-checklist)
- [Summary](#summary)
- [Motivation](#motivation)
  - [Goals](#goals)
  - [Non-Goals](#non-goals)
- [Proposal](#proposal)
  - [Implementation Details/Notes/Constraints](#implementation-detailsnotesconstraints)
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

TaintNodeByCondition has been created in the past to taint node by their conditions. We wish to graduate TaintNodeByCondition feature to make scheduling decisions based on taints instead of node conditions in the scheduler.

## Motivation

TaintNodeByCondition has been beta'ed in 1.12.

### Goals

+ Plan to promote TaintNodeByCondition to the stable version.

### Non-Goals

+ Changing API field or meaning

## Proposal

### Implementation Details/Notes/Constraints

TaintNodeByCondition add taints to nodes based on their conditions in the node lifecycle controller. And it could help the default scheduler to not schedule on specific nodes unless they could tolerate them.

The scheduler will remove condition-based predicates after TaintNodeByCondition was graduated to a stable version. 

## Design Details

### Test Plan

#### Existing Tests

TaintNodeByCondition currently has multiple tests in various components that use the feature.

### Graduation Criteria

**Note:** *Section not required until targeted at a release.*

- [ ] Graduate TaintNodeByCondition to GA
- [ ] Update documents to reflect the changes

## Implementation History

+ TaintNodeByCondition was introduced in Kubernetes 1.8 as an alpha version.
+ TaintNodeByCondition was graduated to beta in Kubernetes 1.12.
