---
title: Kube-Scheduler ComponentConfig graduation to v1beta1
authors:
  - "@jfbai"
owning-sig: sig-scheduling
participating-sigs:
  - sig-cluster-lifecycle
  - sig-api-machinery
  - wg-component-standard
reviewers:
  - "@luxas"
  - "@bsalamat"
  - "@k82cn"
  - "@Huang-Wei"
approvers:
  - "@bsalamat"
  - "@k82cn"
editor: "@jfbai"
creation-date: 2019-07-26
last-updated: 2019-07-26
status: implementable
---

# Kube-scheduler ComponentConfig graduation to v1beta1

## Table of Contents

<!-- toc -->
- [Kube-scheduler ComponentConfig graduation to v1beta1](#kube-scheduler-componentconfig-graduation-to-v1beta1)
  - [Table of Contents](#table-of-contents)
  - [Release Signoff Checklist](#release-signoff-checklist)
  - [Summary](#summary)
  - [Motivation](#motivation)
    - [Goals](#goals)
    - [Non-Goals](#non-goals)
  - [Proposal](#proposal)
  - [Design Details](#design-details)
    - [Test Plan](#test-plan)
    - [Graduation Criteria](#graduation-criteria)
  - [Implementation History](#implementation-history)
  - [Alternatives [optional]](#alternatives-optional)
<!-- /toc -->

## Release Signoff Checklist

**ACTION REQUIRED:** In order to merge code into a release, there must be an issue in [kubernetes/enhancements] referencing this KEP and targeting a release milestone **before [Enhancement Freeze](https://github.com/kubernetes/sig-release/tree/master/releases)
of the targeted release**.

For enhancements that make changes to code or processes/procedures in core Kubernetes i.e., [kubernetes/kubernetes], we require the following Release Signoff checklist to be completed.

Check these off as they are completed for the Release Team to track. These checklist items _must_ be updated for the enhancement to be released.

- [ ] kubernetes/enhancements issue in release milestone, which links to KEP (this should be a link to the KEP location in kubernetes/enhancements, not the initial KEP PR)
- [X] KEP approvers have set the KEP status to `implementable`
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

This document is intended to propose a process and desired goals by which kube-scheduler's ComponentConfig is to be graduated to v1beta1.

## Motivation

Kube-scheduler ComponentConfig has been supported in Kubernetes as Alpha feature since v1.12. Since then, the code and API has been stabilized. Therefore, we would like to graduate kube-scheduler ComponentConfig from v1alpha1 to v1beta1.

### Goals

- Introduce v1beta1 for Kube-scheduler ComponentConfig

### Non-Goals

- Do major changes against Kube-scheduler ComponentConfig

## Proposal

- Move `k8s.io/kube-scheduler/config/v1alpha1` to `k8s.io/kube-scheduler/config/v1beta1`

## Design Details

### Test Plan

Existing test cases throughout the kube-scheduler code base should be adapted to use the latest config version. If required, new test cases should also be created.

### Graduation Criteria

The config should be considered graduated to v1beta1 if it:

- is well covered by tests.
- is well documented. Especially with regards of migrating to it from older versions.

## Implementation History

- Kube-scheduler ComponentConfig was introduced in kubernetes 1.12 [#66916](https://github.com/kubernetes/kubernetes/pull/66916)
- Add bind timeout option [#67556](https://github.com/kubernetes/kubernetes/pull/67556)
- Add scheduling framework configuration [77501](https://github.com/kubernetes/kubernetes/pull/77501)

## Alternatives [optional]

- Continue to keep kube-scheduler ComponentConfig v1alpha1