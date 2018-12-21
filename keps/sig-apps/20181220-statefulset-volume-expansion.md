---
title: StatefulSet Volume Expansion
authors:
  - "@sidakm"
owning-sig: sig-apps
participating-sigs:
  - sig-storage
reviewers:
  - "@janetkuo"
  - "@gnufied"
approvers:
  - "@kow3ns" 
editor: TBD
creation-date: 2018-12-20
last-updated: 2019-08-02
status: provisional
see-also:
  - https://github.com/kubernetes/enhancements/issues/531
  - https://github.com/kubernetes/enhancements/pull/737
  - https://github.com/kubernetes/enhancements/issues/284
replaces:
  - n/a
superseded-by:
  - n/a
---

# StatefulSet Volume Expansion

## Table of Contents

* [Table of Contents](#table-of-contents)
* [Summary](#summary)
* [Motivation](#motivation)
    * [Goals](#goals)
* [Proposal](#proposal)
    * [Implementation Details/Notes/Constraints](#implementation-detailsnotesconstraints)
    * [Risks and Mitigations](#risks-and-mitigations)
* [Graduation Criteria](#graduation-criteria)
* [Implementation History](#implementation-history)

[Tools for generating]: https://github.com/ekalinin/github-markdown-toc

## Summary

The purpose of this enhancement is to allow for the expansion of persistent volume claims created by StatefulSets. This entails propagating increases to storage requests in `StatefulSets.volumeClaimTemplates` to associated persistent volume claims.

## Motivation
In Kubernetes v1.11 the persistent volume expansion feature was promoted to beta. This allowed users to expand volumes by editing storage requests in persistent volume claim objects.

Kubernetes creates a persistent volume for each `volumeClaimTemplate` in the `volumeClaimTemplates` component of a StatefulSet. However, it is not possible to expand persistent volumes, created by StatefulSets, by editing the source `volumeClaimTemplate` in the StatefulSet object. Therefore, it is necessary for the user to individually modify all pods' persistent volume claims, by increasing their storage requests, to expand the underlying persistent volumes. This would have to be repeated each time the number of replicas in the StatefulSet object is increased, since new persistent volumes would be created with the original storage request specified in `volumeClaimTemplates` component.

It would be easier and expected to allow for changes to storage requests in the `volumeClaimTemplates` component of a StatefulSet to propagate to all associated persistent volume claims.

Relevant Issues:

* https://github.com/kubernetes/kubernetes/issues/71477
* https://github.com/kubernetes/kubernetes/issues/72198

### Goals

Allow for increases to storage requests in the `volumeClaimTemplates` component of a StatefulSet to propagate to all associated persistent volume claims.

## Proposal

### Implementation Details/Notes/Constraints

The apiserver will allow for increases to storage requests in the `volumeClaimTemplates` component of a StatefulSet. Additionally, for each volumeClaimTemplate being expanded, it will be necessary to check if its associated StorageClass has volume expansion enabled. This will be achieved by updating the `PersistentVolumeClaimResize` admission controller to incorporate validating updates to `volumeClaimTemplates` within a StatefulSet. Specifically the admission controller will now also check if every `volumeClaimTemplate` in a StatefulSet, which is being resized, is associated with a StorageClass that has `allowVolumeExpansion = true`.

During the StatefulSet update process, the StatefulSet controller will detect an update to a `volumeClaimTemplate` by comparing the updated and current revision of the StatefulSet. This requires the `VolumeClaimTemplates` component of the StatefulSet to be recorded in the StatefulSet's `ControllerRevision` object. 

While updating a pod, the StatefulSet controller will update a referenced persistent volume claim object if its storage request in the associated `volumeClaimTemplate` has been increased.

Not all volumes support online control-plane expansion so this design aims to support both online and offline control-plane volume expansion.

The functionality provided by this enhancement will be gated by the `StatefulSetVolumeExpansion` feature gate.

For the initial version of this feature we will simply be expanding all volumes when the associated pod is offline. Below is the outline of how a resize will be propagated to each PVC:

1. User updates a storage request in the `volumeClaimTemplates` component of a StatefulSet
2. Apiserver validation validates that `StatefulSetVolumeExpansion` is enabled and that the storage request has not been decreased.
3. The `PersistentVolumeClaimResize` admission controller verifies that the associated StorageClass for the `volumeClaimTemplate` being updated has `allowVolumeExpansion = true`
4. Each PVC will be resized after the associated pod has been deleted by the StatefulSet controller. The controller will wait for expansion to complete before the pod is recreated. The controller will also wait for FileSystemExpansion to complete before concluding.

In step 4 if volume expansion continues to fails we will document the following workaround for users. Note this is based off of the documented workaround to recover from a failing volume expansion.

1. Delete the StatefulSet. Note PVCs and PVs are preserved.
2. For the PV associated with the offending PVC, edit it to have the `Retain` reclaim policy.
3. Delete the offending PVC.
4. Recreate the PVC with the old spec and rebind to the same PV.
5. Recreate the StatefulSet with the old spec.

Note this issue can occur after some portion of the volumes have already been successfully resized. So after following the workaround the user can update the StatefulSet again to attempt to expand the remaining volumes. 

#### Optimization for volumes that can be expanded online

Note this is an optimization that may be added in later versions of this feature.

Prerequisite: The StatefulSet controller must be able to determine if a volume supports online expansion.

If a volume is deemed to support online expansion it will be expanded before the pod is terminated for the update. The StatefulSet controller will wait for the file system resize to complete on all such volumes before terminating the associated pod.

Otherwise, the volume will be expanded offline as outlined above after the pod is terminated and before it is recreated.

This design minimizes the time a StatefulSet pod is unavailable if all volumes support online expansion while still supporting volumes that can only be expanded offline.

### Risks and Mitigations

Since changes to `VolumeClaimTemplates` will be recorded in a statefulset's revisions, rolling back a volume expansion would imply the client is attempting to shrink volumes which is unsupported and would be rightfully invalidated by the apiserver. However, this also means that it is possible for a client to attempt to rollback another change in a revision X but be prevented from doing so as a volume was expanded in a revision Y where Y >= X. The implications and potential alternatives to this should be further discussed.

## Graduation Criteria

Move to Alpha after initial implementation and approvals.

Consider optimizing for volumes that support online expansion for the Beta.

## Implementation History

* Initial implementation [PR](https://github.com/kubernetes/kubernetes/pull/71384/files)
