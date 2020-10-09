# KEP-1847: Auto remove PVCs created by StatefulSet

<!-- toc -->
- [Release Signoff Checklist](#release-signoff-checklist)
- [Summary](#summary)
- [Motivation](#motivation)
  - [Goals](#goals)
  - [Non-Goals](#non-goals)
- [Proposal](#proposal)
  - [Background](#background)
  - [Changes required](#changes-required)
  - [User Stories (optional)](#user-stories-optional)
    - [Story 1](#story-1)
    - [Story 2](#story-2)
  - [Notes/Constraints/Caveats (optional)](#notesconstraintscaveats-optional)
  - [Risks and Mitigations](#risks-and-mitigations)
- [Design Details](#design-details)
  - [Volume reclaim policy for the StatefulSet created PVCs](#volume-reclaim-policy-for-the-statefulset-created-pvcs)
- [Cluster role change for statefulset controller](#cluster-role-change-for-statefulset-controller)
  - [Test Plan](#test-plan)
  - [Graduation Criteria](#graduation-criteria)
    - [Alpha release](#alpha-release)
  - [Upgrade / Downgrade Strategy](#upgrade--downgrade-strategy)
  - [Version Skew Strategy](#version-skew-strategy)
- [Production Readiness Review Questionnaire](#production-readiness-review-questionnaire)
  - [Feature Enablement and Rollback](#feature-enablement-and-rollback)
- [Implementation History](#implementation-history)
- [Drawbacks](#drawbacks)
- [Alternatives](#alternatives)
<!-- /toc -->

## Release Signoff Checklist

Items marked with (R) are required *prior to targeting to a milestone / release*.

- [ ] (R) Enhancement issue in release milestone, which links to KEP dir in [kubernetes/enhancements] (not the initial KEP PR)
- [ ] (R) KEP approvers have approved the KEP status as `implementable`
- [ ] (R) Design details are appropriately documented
- [ ] (R) Test plan is in place, giving consideration to SIG Architecture and SIG Testing input
- [ ] (R) Graduation criteria is in place
- [ ] (R) Production readiness review completed
- [ ] Production readiness review approved
- [ ] "Implementation History" section is up-to-date for milestone
- [ ] User-facing documentation has been created in [kubernetes/website], for publication to [kubernetes.io]
- [ ] Supporting documentation e.g., additional design documents, links to mailing list discussions/SIG meetings, relevant PRs/issues, release notes


[kubernetes.io]: https://kubernetes.io/
[kubernetes/enhancements]: https://git.k8s.io/enhancements
[kubernetes/kubernetes]: https://git.k8s.io/kubernetes
[kubernetes/website]: https://git.k8s.io/website

## Summary
The proposal is to add a feature to autodelete the PVCs created by StatefulSet.

## Motivation

Currently, the PVCs created automatically by the StatefulSet are not deleted when 
the StatefulSet is deleted. As can be seen by the discussion in the issue 
[55045](https://github.com/kubernetes/kubernetes/issues/55045) there are several use
cases where the PVCs which are automatically created are deleted as well. In many 
StatefulSet use cases, PVCs have a different lifecycle than the pods of the 
StatefulSet, and should not be deleted at the same time. Because of this, PVC 
deletion will be opt-in for users.

### Goals

Provide a feature to auto delete the PVCs created by StatefulSet. 
Ensure that the pod restarts due to non scale down events such as rolling 
update or node drain does not delete the PVC.

### Non-Goals

This proposal does not plan to address how the underlying PVs are treated on PVC deletion. 
That functionality will continue to be governed by the ReclaimPolicy of the storage class. 

## Proposal

### Background

Controller `garbagecollector` is responsible for ensuring that when a statefulset 
set is deleted the corresponding pods spawned from the StatefulSet is deleted. 
The `garbagecollector` uses `OwnerReference` added to the `Pod` by statefulset controller
to delete the Pod. Similar mechanism is leveraged by this proposal to automatically 
delete the PVCs created by the StatefulSet controller.

### Changes required

The following changes are required:

1. Add `PersistentVolumeClaimReclaimPolicy` entry into StatefulSet spec inorder to make this feature an opt-in.
2. Provide the following PersistentVolumeClaimPolicies:
   * `Retain` - this is the default policy and is considered in cases where no policy is specified. This would be the existing behaviour - when a StatefulSet is deleted, no action is taken with
       respect to the PVCs created by the StatefulSet.
   * `RemoveOnScaledown` - When a pod is deleted on scale down, the corresponding PVC is deleted as well. 
       A scale up following a scale down, will wait till old PVC for the removed Pod is deleted and ensure 
       that the PVC used is a freshly created one.
   * `RemoveOnStatefulSetDeletion` - PVCs corresponding to the StatefulSet are deleted when StatefulSet
       themselves get deleted.
3. Add `patch` to the statefulset controller rbac cluster role for `persistentvolumeclaims`.

### User Stories (optional)

#### Story 1
User environment is such at the content of the PVCs which are created automatically during StatefulSet 
creation need not be retained after the StatefulSet is deleted. User also requires that the scale 
up/down occurs in a fast manner, and leverages any previously existing auto created PVCs within the 
life time of the StatefulSet. An option needs to be provided for the user to auto-delete the PVCs 
once the StatefulSet is deleted. 

User would set the `PersistentVolumeClaimReclaimPolicy` as `RemoveOnStatefulSetDelete` which would ensure that 
the PVCs created automatically during the StatefulSet activation is removed once the StatefulSet 
is deleted.

#### Story 2
User is cost conscious but at the same time can sustain slower scale up(after a scale down) speeds. Needs 
a provision where the PVC created for a pod(which is part of the StatefulSet) is removed when the Pod 
is deleted as part of a scale down. Since the subsequent scale up needs to create fresh PVCs, it will 
be slower than scale ups relying on existing PVCs(from earlier scale ups). 

User would set the `PersistentVolumeClaimReclaimPolicy` as 'RemoveOnScaledown' ensuring PVCs are deleted when corresponding
Pods are deleted. New Pods created during scale  up followed by a scaledown will wait for freshly created PVCs.

### Notes/Constraints/Caveats (optional)

This feature applies to PVCs which are dynamically provisioned from the volumeClaimTemplate of 
a StatefulSet. Any PVC and PV provisioned from this mechanism will function with this feature.

### Risks and Mitigations

Currently the PVCs created by statefulset are not deleted automatically. Using the 
`RemoveOnScaledown` or `RemoveOnStatefulSetDeletion` would delete the PVCs 
automatically. Since this involves persistent data being deleted, users should take 
appropriate care using this feature. Having the `Retain` behaviour as default 
will ensure that the PVCs remain intact by default and only a conscious choice 
made by user will involve any persistent data being deleted. Also, PVCs associated with the StatefulSet will be more 
durable than ephemeral volumes would be, as they are only deleted on scaledown or StatefulSet deletion, and not on other pod lifecycle events 
like being rescheduled to a new node, even with the new retain policies.

## Design Details

### Volume reclaim policy for the StatefulSet created PVCs

When a statefulset spec has a `VolumeClaimTemplate`, PVCs are dynamically created 
using a static naming scheme. A new field named `PersistentVolumeClaimReclaimPolicy` of the 
type `StatefulSetPersistentVolumeClaimReclaimPolicy` will be added to the StatefulSet. This 
field will represent the user indication on whether the associated PVCs can be automatically 
deleted or not. The default policy would be `Retain`. 

If `PersistentVolumeClaimReclaimPolicy` is set to `RemoveOnScaledown`, Pod is set as the owner of the PVCs created
from the `VolumeClaimTemplates` just before the scale down is performed by the statefulset controller. 
When a Pod is deleted, the PVC owned by the Pod is also deleted. When `RemoveOnScaledown` 
policy is set and the Statefulset gets deleted the PVCs also will get deleted 
(similar to `RemoveonStatefulSetDeletion` policy).

Current statefulset controller implementation ensures that the manually deleted pods are restored 
before the scale down logic is run. This combined with the fact that the owner references are set 
only before the scale down will ensure that manual deletions do not automatically delete the PVCs 
in question.

During scale-up, if a PVC has an OwnerRef that does not match the Pod, it 
potentially indicates that the PVC is referred by the deleted Pod and is in the process of 
getting deleted. Controller will exit the current reconcile loop and attempt to reconcile in the 
next iteration. This avoids a race with PVC deletion.

When `PersistentVolumeClaimReclaimPolicy` is set to `RemoveOnStatefulSetDeletion` the owner reference in 
PVC points to the StatefulSet. When a scale up or down occurs, the PVC would remain the same. 
PVCs previously in use before scale down will be used again when the scale up occurs. The PVC deletion 
should happen only after the Pod gets deleted. Since the Pod ownership has `blockOwnerDeletion` set to 
`true` pods will get deleted before the StatefulSet is deleted. The `blockOwnerDeletion` for PVCs will 
be set to `false` which ensures that PVC deletion happens only after the StatefulSet is deleted. This 
chain of ownership ensures that Pod deletion occurs before the PVCs are deleted.

`Retain` `PersistentVolumeClaimReclaimPolicy` will ensure the current behaviour - no PVC deletion is performed as part
of StatefulSet controller.

In alpha release we intend to keep the `PersistentVolumeClaimReclaimPolicy` immutable after creation. 
Based on user feedback we will consider making this field mutable in future releases.

## Cluster role change for statefulset controller
Inorder to update the PVC ownerreference, the `buildControllerRoles` will be updated with 
`patch` on PVC resource.

### Test Plan

1. Unit tests

1. e2e tests
    - RemoveOnScaleDown
      1. Create 2 pod statefulset, scale to 1 pod, confirm PVC deleted
      1. Create 2 pod statefulset, add data to PVs, scale to 1 pod, scale back to 2, confirm PV empty.
      1. Create 2 pod statefulset, delete stateful set, confirm PVCs deleted.
      1. Create 2 pod statefulset, add data to PVs, manually delete one pod, confirm pod comes back and PV still has data (PVC not deleted).
      1. As above, but manually delete all pods in stateful set.
      1. Create 2 pod statefulset, add data to PVs, manually delete one pod, immediately scale down to one pod, confirm PVC is deleted.
      1. Create 2 pod statefulset, add data to PVs, manually delete one pod, immediately scale down to one pod, scale back to two pods, confirm PV is empty.
      1. Create 2 pod statefulset, add data to PVs, perform rolling confirm PVC don't get deleted and PV contents remain intact and useful in the updated pods.
    - RemoveOnStatefulSetDeletion
      1. Create 2 pod statefulset, scale to 1 pod, confirm PVC still exists,
      1. Create 2 pod statefulset, add data to PVs, scale to 1 pod, scale back to 2, confirm PV has data (PVC not deleted).
      1. Create 2 pod statefulset, delete stateful set, confirm PVCs deleted
      1. Create 2 pod statefulset, add data to PVs, manually delete one pod, confirm pod comes back and PV has data (PVC not deleted).
      1. As above, but manually delete all pods in stateful set.
      1. Create 2 pod statefulset, add data to PVs, manually delete one pod, immediately scale down to one pod, confirm PVC exists.
      1. Create 2 pod statefulset, add data to PVs, manually delete one pod, immediately scale down to one pod, scale back to two pods, confirm PV has data.
    - Retain: 
      1. same tests as above, but PVCs not removed in any case and confirm data intact on the PV.
    - Pod restart tests:
      1. Create statefulset, perform rolling update 
1. Upgrade/Downgrade tests
    1. Create statefulset in previous version and upgrade to the version 
       supporting this feature. The PVCs should remain intact.
    2. Downgrade to earlier version and check the PVCs with Retain
       remain intact and the others with set policies before upgrade 
       gets removed based on if the references were already set.
1. Feature disablement/enable test for alpha feature flag `statefulset-autodelete-pvcs`.


### Graduation Criteria

#### Alpha release
- Complete adding the items in the 'Changes required' section.
- Add unit, functional, upgrade and downgrade tests to automated k8s test.

### Upgrade / Downgrade Strategy

There is a new field getting added to the StatefulSet. The upgrade will not 
change the previously expected behaviour of existing Statefulset. 

If the statefulset had been set with the RemoveOnStatefulSetDeletion 
and RemoveOnScaleDown and the version of the kube-controller downgraded,
even though the `PersistentVolumeClaimReclaimPolicy` field will go away, the references
would still be acted upon by the garbage collector and cleaned up 
based on the settings before downgrade. 

### Version Skew Strategy
There is only kubecontroller manager changes involved, hence not applicable for
version skew involving other components.

## Production Readiness Review Questionnaire

### Feature Enablement and Rollback

* **How can this feature be enabled / disabled in a live cluster?**
  - [x] Feature gate (also fill in values in `kep.yaml`)
    - Feature gate name: statefulset-autodelete-pvcs
    - Components depending on the feature gate: kube-controller-manager
  
* **Does enabling the feature change any default behavior?**
  The default behaviour is only changed when user explicitly specifies the `PersistentVolumeClaimReclaimPolicy`. 
  Hence no change in any user visible behaviour change by default.

* **Can the feature be disabled once it has been enabled (i.e. can we roll back the enablement)?**
  Yes, but with side effects for users who already started using the feature by means of 
  specifying non-retain `PersistentVolumeClaimReclaimPolicy`. We will an annotation to the
  PVC indicating that the references have been set from previous enablement. Hence a reconcile
  loop which goes through the required PVCs and removes the references will be added. 
  The side effect is that if there was pod deletion before the references were removed after the
  feature flag was diabled, the PVCs could get deleted.
  
* **What happens if we reenable the feature if it was previously rolled back?** 
The reconcile loop which removes references on disablement will not come into action. Since the 
StatefulSet field would persist through the disablment we will have to ensure that the required
references get set in the next set of reconcile loops.

* **Are there any tests for feature enablement/disablement?**
Feature enablement disablement tests will be added. 

## Implementation History

## Drawbacks
The Statefulset field update is required.

## Alternatives
Users can delete the PVC manually. This is the motivation of the KEP.