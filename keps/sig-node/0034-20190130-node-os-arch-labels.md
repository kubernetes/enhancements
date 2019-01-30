---
kep-number: 34
title: 
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

# Promote Node Operating System & Archtecture labels to GA

## Table of Contents

* [Table of Contents](#table-of-contents)
* [Summary](#summary)
* [Motivation](#motivation)
    * [Goals](#goals)
    * [Non-Goals](#non-goals)
* [Proposal](#proposal)
    * [Risks and Mitigations](#risks-and-mitigations)
* [Graduation Criteria](#graduation-criteria)

## Summary

The kubelet has been been labeling the Node object with operating system (OS)
and archietcture (arch) labels since Kubernetes 1.3. This proposal aims to
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

A new set of GA labels will be added:

```
kubernetes.io/os
kubernetes.io/arch
```

Starting in *v1.14*,
- kubelets will report *both* the beta and the GA labels. 
- the node controller will reconcile a mismatch between a present or missing
  GA label in favor of the beta label
  * This ensure uniform labeling in the presence of pre-1.14 kubelets that
    don't report the GA labels.

Starting in *v1.18*, 
- kubelet will stop reporting the beta labels.
- the node controller will switch to reconciling a mismatch between a present
  beta label and a present GA label in favor of the GA label
  * Since pre-1.14 kubelets are not supported against a 1.18+ control plane.

### Risks and Mitigations

The risk is that nodes may have inconsistent labels after upgrades &
downgrades. This is addrssed by instructing the node controller to reconcile
and ensure uniform labeling.

## Graduation Criteria

None.

