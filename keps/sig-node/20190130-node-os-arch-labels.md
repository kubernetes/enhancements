---
title: Promote Node Operating System & Architecture labels to GA
authors:
  - "@yujuhong"
owning-sig: sig-node
participating-sigs:
  - sig-node
reviewers:
  - "@liggitt"
approvers:
  - "@dchen1107"
editor: Yu-Ju Hong
creation-date: 2019-01-30
last-updated: 2019-01-30
status: implementable
see-also:
replaces:
superseded-by:
---

# Promote Node Operating System & Architecture labels to GA

## Table of Contents

<!-- toc -->
- [Summary](#summary)
- [Motivation](#motivation)
  - [Goals](#goals)
  - [Non-Goals](#non-goals)
- [Proposal](#proposal)
  - [Risks and Mitigations](#risks-and-mitigations)
- [Graduation Criteria](#graduation-criteria)
<!-- /toc -->

## Summary

The kubelet has been been labeling the Node object with operating system (OS)
and architecture (arch) labels since Kubernetes 1.3. This proposal aims to
promote these labels to GA and ensure a smooth transition with backward
compatibility.

Below lists the labels to promote:
```
beta.kubernetes.io/os
beta.kubernetes.io/arch
```

## Motivation

The labels have existed over two years and are widely in various places.
Promoting these labels to reflect their stable status.

### Goals

Promote the OS and arch labels to GA.

### Non-Goals

Promoting any other label that is not listed in the Goals sections.

## Proposal

A new set of GA labels will be added in 1.14:
```
kubernetes.io/os
kubernetes.io/arch
```
Based on the [deprecation
policy](https://kubernetes.io/docs/reference/using-api/deprecation-policy/#deprecating-a-feature-or-behavior),
kubelet will continue reporting beta labels until 1.18.

In *v1.14*,
- announce the deprecation of the beta OS and GA labels, and the timeframe for removal (v1.18)
- update documentation and examples to the GA labels (noting which version the GA labels were added)

Starting in *v1.14*,
- kubelets will report *both* the beta and the GA labels. 
- the node controller will reconcile a mismatch between a present or missing
  GA label in favor of the beta label to ensure uniform labeling.
  - Pre-1.14 kubelets will only report beta labels.
  - 1.14+ kubelets report both the beta and GA labels.
  - If a GA label does not exist for a node, add one to match the beta label.
  - If the beta label does not match the GA label, modify the GA label to match
    the beta label.

In *v1.15*,
- update in-org use in manifests targeting 1.15+ kubernetes versions to the GA labels

Starting in *v1.18*, 
- kubelet will stop reporting the beta labels.
- the node controller will switch to reconciling a mismatch between a present
  beta label and a present GA label in favor of the GA label, since pre-1.14
  kubelets are not supported against a 1.18+ control plane.
  - All nodes of supported versions should report GA labels.
  - If a beta label does not exist for a node, do nothing.
  - If the beta label exists but does not match the GA label, modfiy the beta
    label to match the GA label.

### Risks and Mitigations

The risk is that nodes may have inconsistent labels after upgrades &
downgrades. This is addressed by instructing the node controller to reconcile
and ensure uniform labeling.

## Graduation Criteria

- GA labels are consistently available, regardless of kubelet version
- Examples and documentation direct users to use the new labels
- In-org manifests use the new labels
