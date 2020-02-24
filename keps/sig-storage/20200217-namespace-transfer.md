---
title: Namespace Transfer for Storage Resources
authors:
  - "@mhenriks"
owning-sig: sig-storage
participating-sigs:
  - sig-storage
reviewers:
  - "@saad-ali"
  - "@j-griffith"
approvers:
  - TBD
editor: TBD
creation-date: 2020-02-13
last-updated: 2020-02-13
status: provisional
see-also:
  - "/keps/sig-storage/20190709-csi-snapshot.md"
  - "/keps/sig-storage/20181111-extend-datasource-field"
replaces:
superseded-by:
---

# PVC Namespace Transfer

## Table of Contents

<!-- toc -->
- [Release Signoff Checklist](#release-signoff-checklist)
- [Summary](#summary)
- [Motivation](#motivation)
  - [Goals](#goals)
  - [Non-Goals](#non-goals)
- [Proposal](#proposal)
  - [User Stories](#user-stories)
    - [Story 1](#story-1)
    - [Story 2](#story-2)
  - [Implementation Details/Notes/Constraints](#implementation-detailsnotesconstraints)
  - [Risks and Mitigations](#risks-and-mitigations)
- [Design Details](#design-details)
  - [Test Plan](#test-plan)
  - [Graduation Criteria](#graduation-criteria)
    - [Examples](#examples)
      - [Alpha -&gt; Beta Graduation](#alpha---beta-graduation)
      - [Beta -&gt; GA Graduation](#beta---ga-graduation)
  - [Upgrade / Downgrade Strategy](#upgrade--downgrade-strategy)
  - [Version Skew Strategy](#version-skew-strategy)
- [Implementation History](#implementation-history)
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

This KEP proposes an API to securely transfer Kubernetes storage resources between namespaces.
The initial implementation will focus on PersistentVolumeClaims.  But the API could be applied
to other resource types as compelling use cases come up.

`StorageTransferRequest` resources are created in the target namespace.
`StorageTransferApproval` resources are created in the source namespace.
When a controller detects matching request/approval resources, a transfer in initiated.

When transferring a PersistentVolumeClaim, the associated PersistentVolume will be updated,
the source PersistentVolumeClaim will be deleted, and the target PersistentVolumeClaim will be created.
The transfer simply deals with Kubernetes API resources.
No data on the physical PersistentVolume is accessed/copied. 

## Motivation

Give Kubernetes users the flexibility to easily and securely share data between namespaces.

### Goals

- Define an API for transferring persistent storage resources between namespaces.
- The API should be compatible with as many existing storage provisioners as possible.

### Non-Goals

- Transfer of resources other than PersistentVolumeClaims will not be discussed.
- Implementation details only discussed in the context of influencing the API definition.

## Proposal

`StorageTransferRequest` resource:

```yaml
apiVersion: v1alpha1
kind: StorageTransferRequest
metadata:
  name: transfer-foo
  namespace: target-namespace
spec:
  source:
    name: foo
    namespace: source-namespace
    kind: PersistentVolumeClaim
  targetName: bar #this is optional
```

`StorageTransferApproval` resource:

```yaml
apiVersion: v1alpha1
kind: StorageTransferApproval
metadata:
  name: approve-foo
  namespace: source-namespace
spec:
  source:
    name: foo
    kind: PersistentVolumeClaim
  targetNamespace: target-namespace
```

When matching `StorageTransferRequest` and `StorageTransferApproval` resources are detected, the transfer process is initiated.

Matching requests/approvals is similar to the process for binding PersistentVolumes with PersistentVolumeClaims.
If they are not explicitly bound by the user, a controller will attempt to pair them by looking for matching resource types/names.

Once matching is complete, a controller will begin the transfer process which does the following (not necessarily in order):

- Make sure the associated PersistentVolume is not deleted/recycled during the transfer by setting reclaim policy to `Retain` (if not already).
- Delete the PVC `source-namespace\foo`.
- Bind The PersistentVolume to `target-namespace\bar`.
- Create the PVC `target-namespace\bar` by copying the spec of `source-namespace\foo` and setting `spec.volumeName` appropriately.

The last step makes it possible for the API to be compatible with the most provisioners
as they will ignore a PVC that is already bound.  In [previous](https://github.com/kubernetes/enhancements/pull/1112) namespace transfer KEPs, the target PVC is created by the user.  Although, this will probably work with the CSI external provisioner, it will not work with others (like static local volume).

Users can monitor the status of a transfer by querying for the existence of the bound target PVC or checking the `status.complete` property of `StorageTransferRequest`.

`StorageTransferRequest` object:

```golang
type StorageTransferRequest struct {
  metav1.TypeMeta
  metav1.ObjectMeta

  Spec StorageTransferRequestSpec
  Status *StorageTransferRequestStatus
}
```

```golang
type StorageTransferRequestSpec struct {
  Source *corev1.ObjectReference

  // ApprovalName is set by the controller when requests/approvals are matched
  // or by the user when manual binding is preferred
  ApprovalName string

  // TargetName allows for the source and target resources to have different names
  // It is optional
  TargetName *string
}
```

```golang
type StorageTransferRequestStatus struct {
  Complete bool
  Error *StorageTransferError
}
```

`StorageTransferApproval` object:

```golang
type StorageTransferApproval struct {
  metav1.TypeMeta
  metav1.ObjectMeta

  Spec StorageTransferApprovalSpec
  Status *StorageTransferApprovalStatus
}
```

```golang
type StorageTransferApprovalSpec struct {
  Source *corev1.TypedLocalObjectReference

  TargetNamespace string

  // RequestName is set by the controller when requests/approvals are matched
  // or by the user when manual binding is preferred
  RequestName string
}
```

```golang
type StorageTransferApprovalStatus struct {
  Complete bool
  Error *StorageTransferError
}
```

### User Stories

When combined with VolumeSnapshots and PVC cloning, users are given the power to easily share/experiment with persistent data.

#### Story 1

The `prod` namespace contains the production database which is stored on the `db1` PersistentVolumeClaim.
VolumeSnapshots of `db1` are taken at regular intervals.  DBAs want to test a schema update script before running it in `prod`.
A new PVC named `db1-test` is created in `prod` from a VolumeSnapshot of `db1`.  The `db1-test` PVC is then transferred to the `stage` namespace where it can safely be mounted and modified in an isolated testing environment that mimics `prod`.

#### Story 2

Someone wants to build an Infrastructure as a Service provider with Kubernetes and [KubeVirt](https://github.com/kubevirt).
All the default virtual machine images can be stored on PersistentVolumeClaims in a single namespace called `golden-images`.
When virtual machines are created, the PVC containing the appropriate virtual machine image is cloned and transferred to the target namespace.
The virtual machine is booted from the cloned/transferred PVC.

### Implementation Details/Notes/Constraints

I believe the API can be implemented as CRDs and an external controller.  If so, is a KEP really required?  Would there be advantages to implementing in-tree?

What is the best way to deal with unbound PVCs (WaitForFirstConsumer)?  Transfer the unbound PVC?  Wait until it's bound?  May be race conditions with the former.

Should there be a way (with the API) for a user to automatically approve all transfers from a namespace?

### Risks and Mitigations

Are all security concerns handled by request having separate request/approval resources?
Should `sig-auth` be included in the review?

I think the transfer phase (for bound PVCs) can be done by an external controller in a failure resistant way
but issues may be discovered in the implementation phase.

## Design Details

### Test Plan

Functionality should be tested/verified with all in-tree and multiple external provisioners.

**Note:** *Section not required until targeted at a release.*

### Graduation Criteria

**Note:** *Section not required until targeted at a release.*

#### Examples

These are generalized examples to consider, in addition to the aforementioned [maturity levels][maturity-levels].

##### Alpha -> Beta Graduation

##### Beta -> GA Graduation

**Note:** Generally we also wait at least 2 releases between beta and GA/stable, since there's no opportunity for user feedback, or even bug reports, in back-to-back releases.

### Upgrade / Downgrade Strategy

### Version Skew Strategy

## Implementation History

- [Initial](https://github.com/kubernetes/enhancements/pull/643) PVC transfer proposal by John Griffith (@j-griffith) in Dec 2018
- [Updated](https://github.com/kubernetes/enhancements/pull/1112) PVC transfer proposal by John Griffith (@j-griffith) in Jun 2019
