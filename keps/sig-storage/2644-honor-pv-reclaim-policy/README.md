# KEP-2644: Honor Persistent Volume Reclaim Policy

<!-- toc -->
- [Release Signoff Checklist](#release-signoff-checklist)
- [Summary](#summary)
- [Motivation](#motivation)
- [Proposal](#proposal)
  - [Risks and Mitigations](#risks-and-mitigations)
- [Design Details](#design-details)
  - [Test Plan](#test-plan)
    - [E2E tests](#e2e-tests)
  - [Graduation Criteria](#graduation-criteria)
    - [Alpha](#alpha)
    - [Alpha -&gt; Beta](#alpha---beta)
    - [Beta -&gt; GA](#beta---ga)
  - [Upgrade / Downgrade Strategy](#upgrade--downgrade-strategy)
  - [Version Skew Strategy](#version-skew-strategy)
- [Production Readiness Review Questionnaire](#production-readiness-review-questionnaire)
  - [Feature Enablement and Rollback](#feature-enablement-and-rollback)
  - [Scalability](#scalability)
- [Implementation History](#implementation-history)
- [Drawbacks](#drawbacks)
- [Alternatives](#alternatives)
<!-- /toc -->

## Release Signoff Checklist

Items marked with (R) are required *prior to targeting to a milestone / release*.

- [ ] (R) Enhancement issue in release milestone, which links to KEP dir in [kubernetes/enhancements] (not the initial KEP PR)
- [ ] (R) KEP approvers have approved the KEP status as `implementable`
- [ ] (R) Design details are appropriately documented
- [ ] (R) Test plan is in place, giving consideration to SIG Architecture and SIG Testing input (including test refactors)
  - [ ] e2e Tests for all Beta API Operations (endpoints)
  - [ ] (R) Ensure GA e2e tests for meet requirements for [Conformance Tests](https://github.com/kubernetes/community/blob/master/contributors/devel/sig-architecture/conformance-tests.md) 
  - [ ] (R) Minimum Two Week Window for GA e2e tests to prove flake free
- [ ] (R) Graduation criteria is in place
  - [ ] (R) [all GA Endpoints](https://github.com/kubernetes/community/pull/1806) must be hit by [Conformance Tests](https://github.com/kubernetes/community/blob/master/contributors/devel/sig-architecture/conformance-tests.md) 
- [ ] (R) Production readiness review completed
- [ ] (R) Production readiness review approved
- [ ] (R) "Implementation History" section is up-to-date for milestone
- [ ] User-facing documentation has been created in [kubernetes/website], for publication to [kubernetes.io]
- [ ] Supporting documentation—e.g., additional design documents, links to mailing list discussions/SIG meetings, relevant PRs/issues, release notes

[kubernetes.io]: https://kubernetes.io/
[kubernetes/enhancements]: https://git.k8s.io/enhancements
[kubernetes/kubernetes]: https://git.k8s.io/kubernetes
[kubernetes/website]: https://git.k8s.io/website

## Summary
Reclaim policy associated with the Persistent Volume is currently ignored under certain circumstance. For a `Bound`
PV-PVC pair the ordering of PV-PVC deletion determines whether the PV delete reclaim policy is honored. The PV honors
the reclaim policy if the PVC is deleted prior to deleting the PV, however, if the PV is deleted prior to deleting the
PVC then the Reclaim policy is not exercised. As a result of this behavior, the associated storage asset in the
external infrastructure is not removed.

## Motivation
Prevent volumes from being leaked by honoring the PV Reclaim policy after the `Bound` PVC is also deleted.

## Proposal
To better understand the existing issue we will initially walk through existing behavior when a Bound PV is deleted prior
to deleting the PVC.


```
kubectl delete pv <pv-name>
```
On deleting a `Bound` PV, a `deletionTimestamp` is added on to the PV object, this triggers an update which is consumed
by the PV-PVC-controller. The PV-PVC-controller observes that the PV is still associated with a PVC, and the PVC hasn't
been deleted, as a result of this, the PV-PVC-controller attempts to update the PV phase to `Bound` state. Assuming that
the PV was already bound previously, this update eventually amounts to a no-op.

The PVC is deleted at a later point of time.
```
kubectl -n <namespace> delete pvc <pvc-name>
```

1. A `deletionTimestamp` is added on the PVC object.
2. The PVC-protection-controller picks the update, verifies if `deletionTimestamp` is present and there are no pods that
are currently using the PVC.
3. If there are no Pods using the PVC, the PVC finalizers are removed, eventually triggering a PVC delete event.
4. The PV-PVC-controller processes delete PVC event, leading to removal of the PVC from it's in-memory cache followed
by triggering an explicit sync on the PV.
5. The PV-PVC-controller processes the triggered explicit sync, here, it observes that the PVC is no longer available
and updates the PV phase to `Released` state.
6. If the PV-protection-controller picks the update, it observes that there is a `deletionTimestamp` and the PV is not
in a `Bound` phase, this causes the finalizers to be removed.
7. This is followed by PV-PVC-controller initiating the reclaim volume workflows.
8. The reclaim volume workflow observes the `persistentVolumeReclaimPolicy` as `Delete` and schedules a volume deletion.
9. Under the event that (6) has occurred and when `deleteVolumeOperation` executes it attempts to retrieve the latest PV
state from the API server, however, due to (6) the PV is removed from the API server, leading to an error state. This
results in the plugin volume deletion not being exercised hence leaking the volume.
```go
func (ctrl *PersistentVolumeController) deleteVolumeOperation(volume *v1.PersistentVolume) (string, error) {
	klog.V(4).Infof("deleteVolumeOperation [%s] started", volume.Name)

	// This method may have been waiting for a volume lock for some time.
	// Previous deleteVolumeOperation might just have saved an updated version, so
	// read current volume state now.
	newVolume, err := ctrl.kubeClient.CoreV1().PersistentVolumes().Get(context.TODO(), volume.Name, metav1.GetOptions{}) //<========== NotFound error thrown
	if err != nil {
		klog.V(3).Infof("error reading persistent volume %q: %v", volume.Name, err)
		return "", nil
	}
    // Remaining code below skipped for brevity...
    // Please refer pkg/controller/volume/persistentvolume/pv_controller.go in kubernetes/kubernetes for the full code. 
```
10. Under the event that (6) has not occurred yet, during execution of  `deleteVolumeOperation` it is observed that the
PV has a pre-existing `deletionTimestamp`, this makes the method assume that delete is already being processed. This
results in the plugin volume deletion not being exercised hence leaking the volume.
```go
func (ctrl *PersistentVolumeController) deleteVolumeOperation(volume *v1.PersistentVolume) (string, error) {
	klog.V(4).Infof("deleteVolumeOperation [%s] started", volume.Name)

	// This method may have been waiting for a volume lock for some time.
	// Previous deleteVolumeOperation might just have saved an updated version, so
	// read current volume state now.
	newVolume, err := ctrl.kubeClient.CoreV1().PersistentVolumes().Get(context.TODO(), volume.Name, metav1.GetOptions{})
	if err != nil {
		klog.V(3).Infof("error reading persistent volume %q: %v", volume.Name, err)
		return "", nil
	}

	if newVolume.GetDeletionTimestamp() != nil {//<==========================DeletionTimestamp set since the PV was deleted first.
		klog.V(3).Infof("Volume %q is already being deleted", volume.Name)
		return "", nil
	}
    // Remaining code below skipped for brevity...
    // Please refer pkg/controller/volume/persistentvolume/pv_controller.go in kubernetes/kubernetes for the full code.
```
11. Meanwhile, the external-provisioner checks if there is a `deletionTimestamp` on the PV, if so, it assumes that its
in a transitory state and returns false for the `shouldDelete` check.
```go
// shouldDelete returns whether a volume should have its backing volume
// deleted, i.e. whether a Delete is "desired"
func (ctrl *ProvisionController) shouldDelete(ctx context.Context, volume *v1.PersistentVolume) bool {
	if deletionGuard, ok := ctrl.provisioner.(DeletionGuard); ok {
		if !deletionGuard.ShouldDelete(ctx, volume) {
			return false
		}
	}

	if ctrl.addFinalizer && !ctrl.checkFinalizer(volume, finalizerPV) && volume.ObjectMeta.DeletionTimestamp != nil {
		return false
	} else if volume.ObjectMeta.DeletionTimestamp != nil { //<========== DeletionTimestamp is set on the PV since it's deleted first.
		return false
	}
    // Remaining code below skipped for brevity...
    // Please refer sig-storage-lib-external-provisioner/controller/controller.go for the full code.
```

The main approach in fixing the issue involves using an existing finalizer already implemented in sig-storage-lib-external-provisioner. It is `external-provisioner.volume.kubernetes.io/finalizer` which is added on the Persistent Volume. Currently it is applied only during provisioning if the feature is enabled. The proposal is to not only add this finalizer to newly provisioned PVs, but also to extend the library to add the finalizer to existing PVs. Adding the finalizer prevents the PV from being removed from the API server. The finalizer will be removed only after the physical volume on the storage system is deleted.

When the PVC is deleted, the PV is moved into `Released` state and following checks are made:

1. Plugin:
If the volume has the finalizer `external-provisioner.volume.kubernetes.io/finalizer`, then, `DeletionTimestamp` checks can be ignored.

3. CSI driver

If when processing the PV update it is observed that it has `external-provisioner.volume.kubernetes.io/finalizer` finalizer and
`DeletionTimestamp` set, then the volume deletion request is forwarded to the driver, provided other pre-defined conditions are met.

### Risks and Mitigations

The primary risk is volume deletion to a user that expect the current behavior, i.e, they do not expect the volume to be
deleted from the underlying storage infrastructure when for a bound PV-PVC pair, the PV is deleted followed by deleting
the PVC. This can be mitigated by initially introducing the fix behind a feature gate, followed by explicitly deprecating
the current behavior.

## Design Details

A feature gate named `HonorPVReclaimPolicy` will be introduced for both `kube-controller-manager` and `external-provisioner`.

An existing finalizer `external-provisioner.volume.kubernetes.io/finalizer` is already implemented in sig-storage-lib-external-provisioner. It is added on the Persistent Volume. Currently it is applied only during provisioning if the feature is enabled. The proposal is to not only add this finalizer to newly provisioned PVs, but also to extend the library to add the finalizer to existing PVs. The existing `AddFinalizer` config option will be used to apply the finalizer. Adding the finalizer prevents the PV from being removed from the API server. The finalizer will be removed only after the physical volume on the storage system is deleted.

When CSI Migration is enabled, external-provisioner adds `external-provisioner.volume.kubernetes.io/finalizer` to all the PVs it creates, including in-tree ones. When CSI Migraiton is disabled, however, these PVs will be deleted by in-tree volume plugin. Therefore, in-tree PV controller needs to be modified to remove the finalizer when the PV is being deleted when CSI Migration is disabled.

```go
// PVDeletionProtectionFinalizer finalizer is added to the PV to prevent PV from being deleted before the physical volume
PVDeletionProtectionFinalizer = "external-provisioner.volume.kubernetes.io/finalizer"
```

In `deleteVolumeOperation` in kubernetes/pkg/controller/volume/persistentvolume/pv_controller.go, PV without `PVDeletionProtectionFinalizer` will be skipped as it is already processed and PV with `PVDeletionProtectionFinalizer` will proceed with volume deletion. This will allow external-provisioner to handle the deletion of the CSI volumes. The finalizer `PVDeletionProtectionFinalizer` will only be deleted after the underlying CSI volume is deleted.

```go
// deleteVolumeOperation deletes a volume. This method is running in standalone
// goroutine and already has all necessary locks.
func (ctrl *PersistentVolumeController) deleteVolumeOperation(volume *v1.PersistentVolume) (string, error) {
	klog.V(4).Infof("deleteVolumeOperation [%s] started", volume.Name)
	newVolume, err := ctrl.kubeClient.CoreV1().PersistentVolumes().Get(context.TODO(), volume.Name, metav1.GetOptions{})
	if err != nil {
		klog.V(3).Infof("error reading persistent volume %q: %v", volume.Name, err)
		return "", nil
	}

	// PV without PVDeletionProtectionFinalizer was already processed and its volume deleted.
	if !ctrl.hasPVDeletionProtectionFinalizer(volume, pvutil.PVDeletionProtectionFinalizer) {
		if newVolume.GetDeletionTimestamp() != nil {
			klog.V(3).Infof("Volume %q is already being deleted", volume.Name)
			return "", nil
		}
	}
	//.....
	//.....
	//.....
	klog.V(4).Infof("deleteVolumeOperation [%s]: success", volume.Name)
        return pluginName, err
    }
// Remaining code below skipped for brevity...
// Please refer pkg/controller/volume/persistentvolume/pv_controller.go in kubernetes/kubernetes for the full code.
```

On the driver side, the library adds the finalizer only to newly provisioned PVs. The proposal is to extend the library to (optionally) add the finalizer to all PVs (that are handled by the external-provisioner), to have all PVs protected once the feature is enabled.

When the `shouldDelete` checks succeed, a delete volume request is initiated on the driver. This ensures that the volume is deleted on the backend.

Once the volume is deleted from the backend, the finalizer can be removed. This allows the pv to be removed from the api server.

Note: This feature should work with CSI Migration disabled or enabled.

### Test Plan

#### E2E tests
An e2e test to exercise deletion of PV prior to deletion of PVC for a `Bound` PV-PVC pair.

### Graduation Criteria

#### Alpha
* Feedback
* Fix of the issue behind a feature flag.
* Unit tests and e2e tests outlined in design proposal implemented.
* Tests should be done with both CSI Migration disabled and enabled.

#### Alpha -> Beta
* Allow the Alpha fix to soak for one release.

#### Beta -> GA
* Deployed in production and have gone through at least one K8s upgrade.

### Upgrade / Downgrade Strategy
* Upgrade from old Kubernetes(1.22) to new Kubernetes(1.23) with `HonorPVReclaimPolicy` flag enabled

In the above scenario, the upgrade will cause change in default behavior as described in the current KEP. Additionally,
if there are PVs that have a valid associated PVC and deletion timestamp set, then a finalizer is added to the PV.

* Downgrade from new Kubernetes(1.23) to old Kubernetes(1.22).

In this case, there may be PVs with the deletion finalizer that the older Kubernetes does not remove. Such PVs will be in the API server forever unless if the user manually removes them.

### Version Skew Strategy
The fix is part of `kube-controller-manager` and `external-provisioner`.

1. `kube-controller-manager` is upgraded, but `external-provisioner` is not:

In this case the drivers would still have the issue since the `external-provisioner` is not updated.

2. `external-provisioner` is upgraded but `kube-controller-manager` is not:

In this case the finalizer will be added and removed by the external-provisioner. PVs backed by the in-tree volume plugin are not protected.

In addition, PVs migrated to CSI have the finalizer. When the CSI migration is disabled, in-tree volume plugin / controller-manager does not remove the finalizer. The finalizer must be manually removed by the cluster admin.

## Production Readiness Review Questionnaire

### Feature Enablement and Rollback

###### How can this feature be enabled / disabled in a live cluster?

- [x] Feature gate (also fill in values in `kep.yaml`)
  - Feature gate name: `HonorPVReclaimPolicy`
  - Components depending on the feature gate: kube-controller-manager, external-provisioner

###### Does enabling the feature change any default behavior?

Enabling the feature will delete the volume from underlying storage when the PV is deleted followed deleting the PVC for
a bound PV-PVC pair where the PV reclaim policy is delete.

###### Can the feature be disabled once it has been enabled (i.e. can we roll back the enablement)?
Yes. Disabling the feature flag will continue previous behavior.

###### What happens if we reenable the feature if it was previously rolled back?
Will pick the new behavior as described before.

###### Are there any tests for feature enablement/disablement?

There will be unit tests for the feature `HonorPVReclaimPolicy` enablement/disablement.

### Scalability

###### Will enabling / using this feature result in any new API calls?
The new finalizer will be added to all existing PVs when the feature is enabled. This will be a one-time sync.

###### Will enabling / using this feature result in introducing new API types?
No.

###### Will enabling / using this feature result in any new calls to the cloud provider?
Yes, previously the delete volume call would not reach the provider under the described circumstances, with this change
the volume delete will be invoked to remove the volume from the underlying storage.

###### Will enabling / using this feature result in increasing size or count of the existing API objects?
When the feature is enabled, we will be adding a new finalizer to existing PVs so this would result in an increase of the PV size.

###### Will enabling / using this feature result in increasing time taken by any operations covered by existing SLIs/SLOs?
We have metric for delete volume operations. So potentially the time to delete a PV could be longer now since we want to make sure volume on the storage system is deleted first before deleting the PV.

###### Will enabling / using this feature result in non-negligible increase of resource usage (CPU, RAM, disk, IO, ...) in any components?
No.

## Implementation History

1.23: Alpha

## Drawbacks
None. The current behavior could be considered as a drawback, the KEP presents the fix to the drawback. The current
behavior was reported in [Issue-546](https://github.com/kubernetes-csi/external-provisioner/issues/546) and [Issue-195](https://github.com/kubernetes-csi/external-provisioner/issues/195)

## Alternatives
Other approaches to fix have not been considered and will be considered if presented during the KEP review.
