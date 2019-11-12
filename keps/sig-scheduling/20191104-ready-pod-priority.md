---
title: Ready Pod Priority
authors:
  - "@sparciii"
owning-sig: sig-scheduling
participating-sigs:
  - 
reviewers:
  - TBD
approvers:
  - TBD
editor: 
  - TBD
creation-date: yyyy-mm-dd
last-updated: yyyy-mm-dd
status: provisional
see-also:
  - 
replaces:
  - 
superseded-by:
  - 
---

# Ready Pod Priority

## Table of Contents

<!-- toc -->
- [Release Signoff Checklist](#release-signoff-checklist)
- [Summary](#summary)
- [Motivation](#motivation)
  - [Goals](#goals)
  - [Non-Goals](#non-goals)
- [Proposal](#proposal)
  - [User Stories [optional]](#user-stories-optional)
    - [Story 1](#story-1)
    - [Story 2](#story-2)
  - [Implementation Details/Notes/Constraints [optional]](#implementation-detailsnotesconstraints-optional)
  - [Risks and Mitigations](#risks-and-mitigations)
- [Design Details](#design-details)
  - [Test Plan](#test-plan)
  - [Graduation Criteria](#graduation-criteria)
    - [Examples](#examples)
      - [Alpha -&gt; Beta Graduation](#alpha---beta-graduation)
      - [Beta -&gt; GA Graduation](#beta---ga-graduation)
      - [Removing a deprecated flag](#removing-a-deprecated-flag)
  - [Upgrade / Downgrade Strategy](#upgrade--downgrade-strategy)
  - [Version Skew Strategy](#version-skew-strategy)
- [Implementation History](#implementation-history)
- [Drawbacks [optional]](#drawbacks-optional)
- [Alternatives [optional]](#alternatives-optional)
- [Infrastructure Needed [optional]](#infrastructure-needed-optional)
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

[kubernetes.io]: https://kubernetes.io/
[kubernetes/enhancements]: https://github.com/kubernetes/enhancements/issues
[kubernetes/kubernetes]: https://github.com/kubernetes/kubernetes
[kubernetes/website]: https://github.com/kubernetes/website

## Summary

New scheduler priority `ReadyPodPriority` which adds a score to prefer nodes with least non ready pods.

## Motivation

In our environment we have pods that take a while to pass ready/liveness checks as these pods have caches which need to backfill to memory prior to being ready.

During this extended duration, factors such as spike in memory and cpu resources contribute to very increased load on the node.
When nodes are idle, such as a newly joined node into the cluster, this node would get the majority of the scheduled pods assigned to it, and when multiples of these pods are starting up during the resource intensive period the resulting load would overwhelm the node, evicting pods, and often times become generally unaccessible.

By adding a score to favor nodes with less non ready pods, this helped us prevent deployments of such pods from overwhelming load on node(s).

### Goals

* Avoid node from being overloaded with pods in `NotReady` condition state.

### Non-Goals

## Proposal

### User Stories [optional]

#### Story 1

#### Story 2

### Implementation Details/Notes/Constraints [optional]

### Risks and Mitigations

This introduces a new priority which will affect scoring of nodes during scheduling of pods.

## Design Details

### Test Plan

**Note:** *Section not required until targeted at a release.*

### Graduation Criteria

**Note:** *Section not required until targeted at a release.*

#### Examples

##### Alpha -> Beta Graduation

##### Beta -> GA Graduation

##### Removing a deprecated flag

### Upgrade / Downgrade Strategy

### Version Skew Strategy

## Implementation History

## Drawbacks [optional]

## Alternatives [optional]

## Infrastructure Needed [optional]
