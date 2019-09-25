---
title: Extract k8s.io/code-generator from k/k
authors:
  - "@deads2k"
owning-sig: sig-api-machinery
participating-sigs:
reviewers:
  - "@lavalamp"
  - "@sttts"
approvers:
  - "@lavalamp"
  - "@sttts"
editor: TBD
creation-date: 2019-09-23
last-updated: 2019-09-23
status: implementable
see-also:
replaces:
superseded-by:
---

# Extract k8s.io/code-generator from k/k

We will move k8s.io/code-generator from [k8s.io/kubernetes/staging/src/k8s.io/code-generator](https://github.com/kubernetes/kubernetes/tree/b7003211d5454982401c19705f73bf2820ede855/staging/src/k8s.io/code-generator)
to its own top level repo named k8s.io/code-generator.

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
  - [Upgrade / Downgrade Strategy](#upgrade--downgrade-strategy)
  - [Version Skew Strategy](#version-skew-strategy)
- [Implementation History](#implementation-history)
- [Drawbacks [optional]](#drawbacks-optional)
- [Alternatives [optional]](#alternatives-optional)
- [Infrastructure Needed [optional]](#infrastructure-needed-optional)
<!-- /toc -->

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

k8s.io/code-generator is logically a separate component and one that we have structured as a staging repo before extraction.
After we make the k8s.io/code-generator authoritative, we will vendor the repo into k8s.io/kubernetes to perform our generation.

## Motivation

The k8s.io/code-generator is externally valuable and there is no compelling reason to keep it in tree.
We want to become a standard consumer of generator like all the others. 

### Goals

1. k8s.io/code-generator is the authoritative location of the kube code-generator.

### Non-Goals

1. Changing the k8s.io/code-generator

## Proposal

k8s.io/code-generator only depends on gengo, klog, and kube-openapi (from [go.sum](https://github.com/kubernetes/code-generator/blob/master/go.sum#L106-L114)).
Because it was in staging, its history is up to date.  This means we can simply...

1. Stop publishing k8s.io/code-generator
2. Vendor k8s.io/code-generator as normal using go.mod.
3. Update the readme for k8s.io/code-generator to indicate it as authoritative.
4. For risky changes, a "fake-bump" could be made to k/k pointing to the remote branch to prove that it functions properly.
 This should only be necessary if we find test gaps that need to be addressed, but it's a simple thing for an author
 to put together for a reviewer.

### User Stories [optional]

#### Story 1

#### Story 2

### Implementation Details/Notes/Constraints [optional]

### Risks and Mitigations

1. Changes to k8s.io/code-generator could break k/k.

   Vendoring will break if this happens, so we won't update to a bad level.
   For risky changes, a "fake-bump" could be made to k/k pointing to the remote branch to prove that it functions properly. 

## Design Details

### Test Plan

We aren't changing any code as a part of this.

### Graduation Criteria

The code-generator will continue to be GA.  We will be using exactly the same code afterwards as before.

### Upgrade / Downgrade Strategy

### Version Skew Strategy

## Implementation History

Major milestones in the life cycle of a KEP should be tracked in `Implementation History`.
Major milestones might include

- the `Summary` and `Motivation` sections being merged signaling SIG acceptance
- the `Proposal` section being merged signaling agreement on a proposed design
- the date implementation started
- the first Kubernetes release where an initial version of the KEP was available
- the version of Kubernetes where the KEP graduated to general availability
- when the KEP was retired or superseded

## Drawbacks [optional]

## Alternatives [optional]

## Infrastructure Needed [optional]
