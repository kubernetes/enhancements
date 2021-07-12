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
- [ ] "Implementation History" section is up-to-date for milestone
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
5. The PV-PVC-controller processes the triggered explicit sync, here, it observes that the PVC is not longer available
and updates the PV phase to `Released` state.
6. If the PVC-protection-controller picks the update, it observes that there is a `deletionTimestamp` and the PV is not
in a `Bound` phase, this causes the finalizers to be removed.
7. This is followed by PV-PVC-controller initiating the reclaim volume workflows.
8. The reclaim volume workflow observes the `persistentVolumeReclaimPolicy` as `Delete` and schedules a volume deletion.
9. During delete volume operation, it is observed that there is already a `deletionTimestamp`, and the call is never
forwarded to the CSI driver to delete the underlying volume.
   
As observed from the description above, (9) prevents the volume deletion to be forwarded to the plugin.

Preventing the check should allow the volume deletion.


### Risks and Mitigations

The primary risk is volume deletion for user that expect the current behavior, i.e, they do not expect the volume to be
deleted from the underlying storage infrastructure when for a bound PV-PVC pair, the PV is deleted followed by deleting
the PVC. This can be mitigated by initially introducing the fix behind a feature gate, followed by explicitly deprecating
the current behavior.

## Design Details

The below code is called when `persistentVolumeReclaimPolicy` is `Delete`, as a part of the proposed fix, removing the
`deletionTimestamp` check would allow the volume deletion through the plugin.

```go
// deleteVolumeOperation deletes a volume. This method is running in standalone
// goroutine and already has all necessary locks.
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

	if newVolume.GetDeletionTimestamp() != nil {//<==========================Remove check.
		klog.V(3).Infof("Volume %q is already being deleted", volume.Name)
		return "", nil
	}
	needsReclaim, err := ctrl.isVolumeReleased(newVolume)
	if err != nil {
		klog.V(3).Infof("error reading claim for volume %q: %v", volume.Name, err)
		return "", nil
	}
	if !needsReclaim {
		klog.V(3).Infof("volume %q no longer needs deletion, skipping", volume.Name)
		return "", nil
	}

	pluginName, deleted, err := ctrl.doDeleteVolume(volume)
	// Remaining code below skipped for brevity...
	// Please refer pkg/controller/volume/persistentvolume/pv_controller.go in kubernetes/kubernetes for the full code. 
		
}
```

The proposed fix will not have side effects in existing code flows as the `deleteVolumeOperation` is scheduled with an
identifier generated using the volume name and UID that is being deleted. All duplicate identifiers are skipped.

```go
	case v1.PersistentVolumeReclaimDelete:
		klog.V(4).Infof("reclaimVolume[%s]: policy is Delete", volume.Name)
		opName := fmt.Sprintf("delete-%s[%s]", volume.Name, string(volume.UID))
		// create a start timestamp entry in cache for deletion operation if no one exists with
		// key = volume.Name, pluginName = provisionerName, operation = "delete"
		ctrl.operationTimestamps.AddIfNotExist(volume.Name, ctrl.getProvisionerNameFromVolume(volume), "delete")
		ctrl.scheduleOperation(opName, func() error {
			_, err := ctrl.deleteVolumeOperation(volume)
			if err != nil {
				// only report error count to "volume_operation_total_errors"
				// latency reporting will happen when the volume get finally
				// deleted and a volume deleted event is captured
				metrics.RecordMetric(volume.Name, &ctrl.operationTimestamps, err)
			}
			return err
		})
```

### Test Plan

#### E2E tests
An e2e test to exercise deletion of PV prior to deletion of PVC for a `Bound` PV-PVC pair.


### Graduation Criteria

#### Alpha
* Feedback
* Fix of the issue behind a feature flag.
* Unit tests and e2e tests outlined in design proposal implemented.

#### Alpha -> Beta
* Allow the Alpha fix to soak for one release.

#### Beta -> GA
* Deployed in production and have gone through at least one K8s upgrade.

### Upgrade / Downgrade Strategy
* Upgrade from old Kubernetes(1.21) to new Kubernetes(1.22) with `HonorPvReclaim` flag enabled

In the above scenario, the upgrade will cause change in default behavior as described in the current KEP. Additionally,
no efforts are made to delete volumes that were not previously as a result of old behavior.

* Downgrade from new Kubernetes(1.22) to old Kubernetes(1.21).

In the above scenario, downgrade will retain the current behavior.

### Version Skew Strategy
No Issues, the fix is present on a single component `kube-controller-manager`.

## Production Readiness Review Questionnaire

### Feature Enablement and Rollback

###### How can this feature be enabled / disabled in a live cluster?

- [x] Feature gate (also fill in values in `kep.yaml`)
  - Feature gate name: `HonorPvReclaim`
  - Components depending on the feature gate: kube-controller-manager

###### Does enabling the feature change any default behavior?

Enabling the feature will delete the volume from underlying storage when the PV is deleted followed deleting the PVC for
a bound PV-PVC pair where the PV reclaim policy is delete.

###### Can the feature be disabled once it has been enabled (i.e. can we roll back the enablement)?
Yes. Disabling the feature flag will continue previous behavior.

###### What happens if we reenable the feature if it was previously rolled back?
Will pick the new behavior as described before.

###### Are there any tests for feature enablement/disablement?

There will be unit tests for the feature `HonorPvReclaim` enablement/disablement.

### Scalability

###### Will enabling / using this feature result in any new API calls?
No.

###### Will enabling / using this feature result in introducing new API types?
No.

###### Will enabling / using this feature result in any new calls to the cloud provider?
Yes, previously the delete volume call would not reach the provider under the described circumstances, with this change
the volume delete will be invoked to remove the volume from the underlying storage.

###### Will enabling / using this feature result in increasing size or count of the existing API objects?
No.

###### Will enabling / using this feature result in increasing time taken by any operations covered by existing SLIs/SLOs?
No.

###### Will enabling / using this feature result in non-negligible increase of resource usage (CPU, RAM, disk, IO, ...) in any components?
No.

## Implementation History

1.22 : Alpha

## Drawbacks
None. The current behavior could be considered as a drawback, the KEP presents the fix to the drawback. The current
behavior was reported in [Issue-546](https://github.com/kubernetes-csi/external-provisioner/issues/546) and [Issue-195](https://github.com/kubernetes-csi/external-provisioner/issues/195)

## Alternatives
Other approaches to fix have not been considered and will be considered if presented during the KEP review.