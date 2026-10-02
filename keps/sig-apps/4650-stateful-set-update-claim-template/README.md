# KEP-4650: StatefulSet Support for Updating Volume Claim Template

<!-- toc -->
- [Release Signoff Checklist](#release-signoff-checklist)
- [Summary](#summary)
- [Motivation](#motivation)
  - [Goals](#goals)
  - [Non-Goals](#non-goals)
- [Proposal](#proposal)
  - [Kubernetes API Changes](#kubernetes-api-changes)
  - [Kubernetes Controller Changes](#kubernetes-controller-changes)
  - [User Stories (Optional)](#user-stories-optional)
    - [Story 1: Batch Expand Volumes](#story-1-batch-expand-volumes)
    - [Story 2: Asymmetric Replicas](#story-2-asymmetric-replicas)
  - [Notes/Constraints/Caveats (Optional)](#notesconstraintscaveats-optional)
  - [Risks and Mitigations](#risks-and-mitigations)
- [Design Details](#design-details)
  - [Test Plan](#test-plan)
      - [Prerequisite testing updates](#prerequisite-testing-updates)
      - [Unit tests](#unit-tests)
      - [Integration tests](#integration-tests)
      - [e2e tests](#e2e-tests)
  - [Graduation Criteria](#graduation-criteria)
    - [Alpha](#alpha)
    - [Beta](#beta)
    - [GA](#ga)
  - [Upgrade / Downgrade Strategy](#upgrade--downgrade-strategy)
  - [Version Skew Strategy](#version-skew-strategy)
- [Production Readiness Review Questionnaire](#production-readiness-review-questionnaire)
  - [Feature Enablement and Rollback](#feature-enablement-and-rollback)
  - [Rollout, Upgrade and Rollback Planning](#rollout-upgrade-and-rollback-planning)
  - [Monitoring Requirements](#monitoring-requirements)
  - [Dependencies](#dependencies)
  - [Scalability](#scalability)
  - [Troubleshooting](#troubleshooting)
- [Implementation History](#implementation-history)
- [Drawbacks](#drawbacks)
- [Alternatives](#alternatives)
  - [Extensively validate the updated <code>volumeClaimTemplates</code>](#extensively-validate-the-updated-volumeclaimtemplates)
  - [Support for updating arbitrary fields in <code>volumeClaimTemplates</code>](#support-for-updating-arbitrary-fields-in-volumeclaimtemplates)
  - [Patch PVC size regardless of the immutable fields](#patch-pvc-size-regardless-of-the-immutable-fields)
  - [Support for automatically skip not managed PVCs](#support-for-automatically-skip-not-managed-pvcs)
  - [Reconcile all PVCs regardless of Pod revision labels](#reconcile-all-pvcs-regardless-of-pod-revision-labels)
  - [Treat all incompatible PVCs as unavailable replicas](#treat-all-incompatible-pvcs-as-unavailable-replicas)
  - [Integrate with RecoverVolumeExpansionFailure feature](#integrate-with-recovervolumeexpansionfailure-feature)
  - [Order of Pod / PVC updates](#order-of-pod--pvc-updates)
  - [When to track <code>volumeClaimTemplates</code> in <code>ControllerRevision</code>](#when-to-track-volumeclaimtemplates-in-controllerrevision)
- [Infrastructure Needed (Optional)](#infrastructure-needed-optional)
<!-- /toc -->

## Release Signoff Checklist

Items marked with (R) are required *prior to targeting to a milestone / release*.

- [x] (R) Enhancement issue in release milestone, which links to KEP dir in [kubernetes/enhancements] (not the initial KEP PR)
- [ ] (R) KEP approvers have approved the KEP status as `implementable`
- [x] (R) Design details are appropriately documented
- [x] (R) Test plan is in place, giving consideration to SIG Architecture and SIG Testing input (including test refactors)
  - [ ] e2e Tests for all Beta API Operations (endpoints)
  - [ ] (R) Ensure GA e2e tests meet requirements for [Conformance Tests](https://github.com/kubernetes/community/blob/master/contributors/devel/sig-architecture/conformance-tests.md)
  - [ ] (R) Minimum Two Week Window for GA e2e tests to prove flake free
- [x] (R) Graduation criteria is in place
  - [ ] (R) [all GA Endpoints](https://github.com/kubernetes/community/pull/1806) must be hit by [Conformance Tests](https://github.com/kubernetes/community/blob/master/contributors/devel/sig-architecture/conformance-tests.md)
- [ ] (R) Production readiness review completed
- [ ] (R) Production readiness review approved
- [x] "Implementation History" section is up-to-date for milestone
- [ ] User-facing documentation has been created in [kubernetes/website], for publication to [kubernetes.io]
- [ ] Supporting documentation—e.g., additional design documents, links to mailing list discussions/SIG meetings, relevant PRs/issues, release notes

[kubernetes.io]: https://kubernetes.io/
[kubernetes/enhancements]: https://git.k8s.io/enhancements
[kubernetes/kubernetes]: https://git.k8s.io/kubernetes
[kubernetes/website]: https://git.k8s.io/website

## Summary

Kubernetes does not support the modification of the `volumeClaimTemplates` of a StatefulSet currently.
This enhancement proposes relaxing validation of StatefulSet's VolumeClaim template.
Specifically, we will allow modifying the following fields of `spec.volumeClaimTemplates`:
* increasing the requested storage size (`spec.volumeClaimTemplates.spec.resources.requests.storage`)
* modifying Volume AttributesClass used by the claim (`spec.volumeClaimTemplates.spec.volumeAttributesClassName`)
* modifying VolumeClaim template's labels (`spec.volumeClaimTemplates.metadata.labels`)
* modifying VolumeClaim template's annotations (`spec.volumeClaimTemplates.metadata.annotations`)

When `volumeClaimTemplates` is updated, the StatefulSet controller will reconcile the
PersistentVolumeClaims in the StatefulSet's pods.
The behavior of updating PersistentVolumeClaim is similar to updating Pod.
The updates to PersistentVolumeClaim will be coordinated with Pod updates to honor any dependencies between them.

## Motivation

Currently there are very few things that users can do to update the volumes of
their existing StatefulSet deployments.
They can only expand the volumes, or modify them with VolumeAttributesClass
by updating individual PersistentVolumeClaim objects as an ad-hoc operation.
When the StatefulSet scales up, the new PVC(s) will be created with the old
config and this again needs manual intervention.
This brings many headaches in a continuously evolving environment.

### Goals

* Allow users to update some fields of `volumeClaimTemplates` of a `StatefulSet`, specifically:
  * increasing the requested storage size (`spec.volumeClaimTemplates.spec.resources.requests.storage`)
  * modifying Volume AttributesClass used by the claim (`spec.volumeClaimTemplates.spec.volumeAttributesClassName`)
  * modifying VolumeClaim template's labels (`spec.volumeClaimTemplates.metadata.labels`)
  * modifying VolumeClaim template's annotations (`spec.volumeClaimTemplates.metadata.annotations`)
* Add `.spec.volumeClaimUpdateStrategy` allowing users to decide how the volume claim will be updated: in-place or on PVC deletion.

### Non-Goals

* Support automatic re-creating of PersistentVolumeClaim. We will never delete a PVC automatically.
* Validate the updated `volumeClaimTemplates` as how PVC patch does.
* Update ephemeral volumes.
* Patch PVCs that are different from the template, e.g. StatefulSet adopts the pre-existing PVCs.
* Support for volumes that only support offline expansion.

## Proposal

### Kubernetes API Changes

Change API server to allow specific updates to `volumeClaimTemplates` of a StatefulSet:
   * `spec.volumeClaimTemplates.spec.resources.requests.storage` (increase only)
   * `spec.volumeClaimTemplates.spec.volumeAttributesClassName`
     * Note that this field is currently disabled by default. But should not affect the progress of this KEP.
   * `spec.volumeClaimTemplates.metadata.labels`
   * `spec.volumeClaimTemplates.metadata.annotations`

Introduce a new field in StatefulSet `spec`: `volumeClaimUpdateStrategy` to
specify how to coordinate the update of PVCs and Pods.
It is defined as a struct to allow future extensions.
Possible types are:
- `OnClaimDelete`: the default value, only update the PVC when the the old PVC is deleted.
- `InPlace`: patch the PVC in-place if possible. Also includes the `OnClaimDelete` behavior.

```golang
type StatefulSetSpec struct {
    // volumeClaimUpdateStrategy indicates how PersistentVolumeClaims should be
    // updated to match the volumeClaimTemplates.
    // +optional
    VolumeClaimUpdateStrategy StatefulSetVolumeClaimUpdateStrategy
}

// StatefulSetVolumeClaimUpdateStrategy indicates the strategy that the StatefulSet
// controller will use to update PersistentVolumeClaims. It includes any additional parameters
// necessary to perform the update for the indicated strategy.
type StatefulSetVolumeClaimUpdateStrategy struct {
    // Type indicates the type of the StatefulSetVolumeClaimUpdateStrategy.
    Type StatefulSetVolumeClaimUpdateStrategyType
}

// StatefulSetVolumeClaimUpdateStrategyType is a string enumeration type that enumerates
// all possible update strategies for the PersistentVolumeClaims managed by StatefulSet.
type StatefulSetVolumeClaimUpdateStrategyType string

const (
    // InPlaceStatefulSetVolumeClaimUpdateStrategy indicates that the updates to
    // volumeClaimTemplate will be propagated to the managed PersistentVolumeClaims
    // before updating the Pods. Claims are recreated at the same revision as the corresponding Pod.
    // The update is in-place without interruption or data loss.
    InPlaceStatefulSetVolumeClaimUpdateStrategy StatefulSetVolumeClaimUpdateStrategyType = "InPlace"
    // OnClaimDeleteStatefulSetVolumeClaimUpdateStrategy triggers the legacy behavior.
    // Updates to volumeClaimTemplate only affects the new claims. Version
    // tracking and ordered rolling updates are disabled. Claims are recreated
    // from the StatefulSetSpec when they are manually deleted.
    OnClaimDeleteStatefulSetVolumeClaimUpdateStrategy StatefulSetVolumeClaimUpdateStrategyType = "OnClaimDelete"
)
```

Additionally collect the status of managed PVCs, and show them in the StatefulSet status.
Some fields in the `status` are updated to reflect the status of the PVCs:
- currentRevision, updateRevision, currentReplicas, updatedReplicas
  are updated to reflect the status of PVCs.

```diff
 // StatefulSetStatus represents the current state of a StatefulSet.
 type StatefulSetStatus struct {
-    // currentReplicas is the number of Pods created by the StatefulSet controller from the StatefulSet version
+    // currentReplicas is the number replicas with PersistentVolumeClaims updated to and Pods created from the StatefulSet version
     // indicated by currentRevision.
     CurrentReplicas int32
 
-    // updatedReplicas is the number of Pods created by the StatefulSet controller from the StatefulSet version
+    // updatedReplicas is the number replicas with PersistentVolumeClaims updated to and Pods created from the StatefulSet version
     // indicated by updateRevision.
     UpdatedReplicas int32
 
-    // currentRevision, if not empty, indicates the version of the StatefulSet used to generate Pods in the
+    // currentRevision, if not empty, indicates the version of the StatefulSet used to generate PersistentVolumeClaims and Pods in the
     // sequence [0,currentReplicas).
     CurrentRevision string
 
-    // updateRevision, if not empty, indicates the version of the StatefulSet used to generate Pods in the sequence
+    // updateRevision, if not empty, indicates the version of the StatefulSet used to generate PersistentVolumeClaims and Pods in the sequence
     // [replicas-updatedReplicas,replicas)
     UpdateRevision string
 }
```

We will decrease `currentReplicas` when we start to update the PVCs, and increase `updatedReplicas` when we create the new Pods.
We update `currentRevision` to `updateRevision` when all Pods and PVCs are ready.

With these changes, user can still use `kubectl rollout status` to monitor the update process,
both for automated patching and for the PVCs that need manual intervention.

A PVC is considered ready if:
* PVC's `status.capacity.storage` is greater than or equal to min(template spec, PVC spec).
  If the template is 10Gi, PVC is 10Gi and is expanding to 100Gi but failed, we still consider it ready.
* PVC's `status.currentVolumeAttributesClassName` equals to `spec.volumeAttributesClassName`.

A new label `controller-revision-hash` is added to the PVCs,
to ensure we have the correct version of PVC in cache when determining whether the PVC is ready.

### Kubernetes Controller Changes

Additionally watch for events from PVCs, in order to kickoff the update process when the PVC becomes ready.

If the `volumeClaimUpdateStrategy` field is set to `OnClaimDelete`, nothing changes.
To opt in to the new behavior, the `inPlace` policy should be used.
This new behaviour is described below.

Include `volumeClaimTemplates` in the `ControllerRevision`.

Since modifying `volumeClaimTemplates` will change the hash,
add support for updating `controller-revision-hash` label of the Pod without deleting and recreating the Pod,
if the pod template is not changed.

Before deleting an old Pod, or, if the Pod template is not changed, updating the label,
use server-side apply to update the PVCs used by the Pod.

The patch used in server-side apply is the volumeClaimTemplates in the StatefulSet, except:
* `spec.resources.requests.storage` is set to max(template `spec.resources.requests.storage`, PVC `spec.resources.requests.storage`),
  so that we will never decrease the storage size.
* `controller-revision-hash` label is added to the PVCs.

Naturally, most of the update control logic also applies to PVCs.
If `updateStrategy` is `RollingUpdate`, update the PVCs in the order from the largest ordinal to the smallest.
However, `minReadySeconds` is not considered when only PVCs are updated,
because it is hard to determine when the PVC becomes ready.
And updating PVCs is unlikely to disrupt workloads, so it should be unnecessary to inject delay into the update process.

If `updateStrategy` is `OnDelete`, we do not update the PVCs automatically.

When creating new PVCs, use the `volumeClaimTemplates` from the same revision that is used to create the Pod.

### User Stories (Optional)

#### Story 1: Batch Expand Volumes

We're running a CI/CD system and the end-to-end automation is desired.
To expand the volumes managed by a StatefulSet,
we can just use the same pipeline that we are already using to update the Pod.
All the test, review, approval, and rollback process can be reused.

#### Story 2: Asymmetric Replicas

The storage requirement of different replicas are not identical,
so we still want to update each PVC manually and separately.
Possibly we also update the `volumeClaimTemplates` for new replicas,
but we don't want the controller to interfere with the existing replicas.

### Notes/Constraints/Caveats (Optional)

When designing the `InPlace` update strategy, we want to reuse the infrastructures controlling Pod rollout.
We apply the changes to the PVCs before we set new `controller-revision-hash` label.
New invariance established about PVCs:
If the Pod has revision A label, all its PVCs are either not existing yet, or updated to revision A and ready.

We introduce `controller-revision-hash` label on PVCs to:
* Record where have progressed, to ensure each PVC is only updated once per rollout.
* When waiting for PVCs to become ready, we can check the label to ensure we got the correct version in the informer cache.

The rational of using server-side apply to update PVCs:
Avoid interference with other controllers or human operators that operate on PVCs.
* If additional annotations/labels are added to the PVCs by others, do not remove them.
* If storage class is not set in the template, we should not care the storage class of the PVCs.

### Risks and Mitigations

Since we don't allow decreasing the storage size of `volumeClaimTemplates`,
it is not possible to run `kubectl rollout undo` after increasing it.
This may surprise users already working with StatefulSets, maybe a breaking change.
We may loosen this restriction in the future.
But unfortunately, since volume expansion cannot be fully cancelled,
undoing StatefulSet changes may not be enough to revert the system to the previous state,
but should be enough to unblock StatefulSet rollout.

The user who can update the StatefulSet gains implicit permission to update the PVCs.
This can incur extra fee to cloud providers.
Cluster administrators should setup appropriate quota or validation to mitigate this.

Interfering with other controllers or human operators.
Over the years, the user may have deployed third-party controllers to e.g., expand the volume automatically.
We should not interfere with them. Like Pods, we use `controller-revision-hash` label to record whether we have updated the PVCs.
If the `controller-revision-hash` label on either Pod or PVC is already matched, we will not touch the PVCs again.
So we will not interfere with them as long as the `controller-revision-hash` label is preserved by them.

New Pod may still see old PVC configuration.
We already ensure that the PVC is updated before the new Pod is created.
However, the operation on PVCs can be asynchronous. And expansion may not finish without a running Pod.

## Design Details

When `volumeClaimUpdateStrategy` is `OnClaimDelete`, APIServer should accept the changes to `volumeClaimTemplates`,
but StatefulSet controller should not touch the PVCs and preserve the current behaviour.
Following describes the workflow when `volumeClaimUpdateStrategy` is `InPlace`.

When updating volumeClaimTemplates along with pod template, we will go through the following steps:
1. Apply the changes to the PVCs used by this replica.
2. Wait for the PVCs to be ready.
3. Delete the old pod.
4. Create the new pod with new `controller-revision-hash` label.
5. Wait for the new pod to be ready.
6. Advance to the next replica and repeat from step 1.

When only updating the volumeClaimTemplates:
1. Apply the changes to the PVCs used by this replica.
2. Wait for the PVCs to be ready.
3. Update the pod with new `controller-revision-hash` label.
4. Advance to the next replica and repeat from step 1.

Assuming we are updating a replica from revision A to revision B:

| # | Pod | PVC | Action |
| --- | --- | --- | --- |
| 1 | not existing | not existing | create PVC at revision B |
| 2 | not existing | at revision A | create Pod at revision B |
| 3 | not existing | at revision B | create Pod at revision B |
| 4 | at revision A | not existing | create PVC at revision B |
| 5 | at revision A | at revision A | update PVC to revision B |
| 6 | at revision A | at revision B | wait for PVC to be ready, then delete Pod or update Pod label |
| 7 | at revision B | not existing | create PVC at revision B |
| 8 | at revision B | at revision A | update PVC to revision B |
| 9 | at revision B | at revision B | wait for Pod and PVCs to be ready, then advance to next replica |

A normal rollout should be like: 5 -> 6 (-> 3) -> 9.

Normally, when Pod is at revision B, PVCs will be at revision B and already ready, unless:
* when user set `volumeClaimUpdateStrategy` to `InPlace` when the feature-gate of KCM is disabled,
  or disable the previously enabled feature-gate.
* When the Pod is deleted externally, e.g. be evicted or deleted manually.

In such cases, we will still update PVCs at 8 and wait for the PVCs to be ready at 9.

When `volumeClaimUpdateStrategy` is updated from `OnClaimDelete` to `InPlace`,
StatefulSet controller will begin to add claim templates to ControllerRevision,
which will change its hash and trigger a rollout.
The rollout works like a volumeClaimTemplates only rollout above.
In this case, step 3 will be no-op if PVC is not changed actually (apart from adding the new controller-revision-hash label),
so the rollout should proceed really fast.

When `volumeClaimUpdateStrategy` is updated from `InPlace` to `OnClaimDelete`,
StatefulSet controller will begin to remove claim templates from ControllerRevision,
which will change its hash and trigger a rollout.
PVCs will not be touched and Pods will be updated with new `controller-revision-hash` label.

Failure cases: don't leave too many PVCs being updated in-place. We expect to update the PVCs in order.

- If the PVC update fails, we should block the StatefulSet rollout process.
  We should retry and report events for this.
  The events and status should look like those when the Pod creation fails.
  We update PVC before deleting the old Pod, so failure of PVC update should not disrupt running Pods,
  and user should have enough time to fix this manually.
  The failure cases of this kind includes (but not limited to):
  - immutable fields mismatch (e.g. storageClassName)
  - webhook
  - [storage quota](https://kubernetes.io/docs/concepts/policy/resource-quotas/#storage-resource-quota)
  - [VAC quota](https://kubernetes.io/docs/concepts/policy/resource-quotas/#resource-quota-per-volumeattributesclass)
  - StorageClass.allowVolumeExpansion not set to true

- While waiting for the PVC to become ready,
  we should update status, just like what we do when waiting for Pod to be ready.
  We should block the StatefulSet rollout process if the PVC is never ready.

- When individual PVC failed to become ready, the user can update that PVC manually to bring it back to ready.
  - If the PVC cannot become ready because of the old Pod (e.g. unable to schedule),
    user can delete the Pod and the StatefulSet controller will create a new Pod at new revision.

- If the `volumeClaimTemplates` is updated again when the previous rollout is blocked,
  similar to [Pods](https://kubernetes.io/docs/concepts/workloads/controllers/statefulset/#forced-rollback),
  user may need to manually deal with the blocking PVCs (update or delete them).

In all cases, if the user determines the failure of updating PVCs is not critical,
he can change `volumeClaimUpdateStrategy` back to `OnClaimDelete` to unblock normal Pod rollout immediately.

### Test Plan

[x] I/we understand the owners of the involved components may require updates to
existing tests to make this code solid enough prior to committing the changes necessary
to implement this enhancement.

##### Prerequisite testing updates

N/A

##### Unit tests

For alpha, the core package we will be touching:
- `pkg/controller/statefulset`: `2025-05-25` - `86.5%`
- `pkg/controller/history`: `2025-05-25` - `84.5%`
- `pkg/apis/apps/validation`: `2025-05-25` - `92.5%`

##### Integration tests

- When the feature gate is enabled, existing StatefulSets gains a default `volumeClaimUpdateStrategy` of `OnClaimDelete`, and can be updated to `InPlace`.
  Then disable the feature gate, `volumeClaimUpdateStrategy` field should remain unchanged, but user can clear it manually.

- When the feature gate is disabled in the mid of the PVC rollout, we should not update or wait for the PVCs anymore.
  `volumeClaimTemplate` should remain in the controllerRevision. And the current rollout should finish successfully.

##### e2e tests

- When feature gate is enabled, update the StatefulSet `volumeClaimTemplates` with `volumeClaimUpdateStrategy: InPlace` can successfully expand the PVCs.
  And running Pods are not restarted.

### Graduation Criteria

#### Alpha

- Feature implemented behind a feature flag
- Initial unit, integration and e2e tests completed

#### Beta

- Gather feedback from developers and surveys
- Complete features: StatefulSet status reporting and `kubectl rollout status` support.
- Additional tests are in Testgrid and linked in KEP
- Downgrade tests and scalability tests
- All functionality completed
- All security enforcement completed
- All monitoring requirements completed
- All testing requirements completed
- All known pre-release issues and gaps resolved

#### GA

- 3 examples of real-world usage
- Allowing time for feedback
- All issues and gaps identified as feedback during beta are resolved

### Upgrade / Downgrade Strategy

No changes required to maintain previous behavior.

To make use of the enhancement, user can update `volumeClaimTemplates` of existing StatefulSets.
One can also update `volumeClaimUpdateStrategy` to `InPlace` in order to rollout the changes automatically.

### Version Skew Strategy

No coordinating between the control plane and nodes are required, since this KEP does not involve nodes.

Should enable this feature for APIServer before kube-controller-manager.
An n-1 kube-controller-manager should ignore the `volumeClaimUpdateStrategy` field and never touch PVCs.
It should always create PVCs with the latest `volumeClaimTemplates`.

If `volumeClaimUpdateStrategy` is set to `InPlace` when the feature-gate of kube-controller-manager is disabled,
kube-controller-manager should still update the controllerRevision and label on Pods.
After that, when the feature-gate of kube-controller-manager is enabled,
updates to PVCs will be picked up and rollout will start automatically.

## Production Readiness Review Questionnaire

### Feature Enablement and Rollback

###### How can this feature be enabled / disabled in a live cluster?

- [x] Feature gate (also fill in values in `kep.yaml`)
  - Feature gate name: StatefulSetUpdateVolumeClaimTemplate
  - Components depending on the feature gate:
    - kube-apiserver
    - kube-controller-manager

###### Does enabling the feature change any default behavior?

The update to StatefulSet `volumeClaimTemplates` will be accepted by the API server while it is previously rejected.
StatefulSets gains a new field `volumeClaimUpdateStrategy` with default value `OnClaimDelete`.

Otherwise No.
If `volumeClaimUpdateStrategy` is `OnClaimDelete` (the default value),
the behavior of StatefulSet controller is almost the same as before.

###### Can the feature be disabled once it has been enabled (i.e. can we roll back the enablement)?

Yes. Since the `volumeClaimTemplates` can already differ from the actual PVCs now,
disabling this feature gate should not leave any inconsistent state.

The `volumeClaimUpdateStrategy` field will not be cleared automatically.
When it is set to `InPlace`, `volumeClaimTemplates` also remains in the controllerRevision.
User can rollback each StatefulSet manually by deleting the `volumeClaimUpdateStrategy` field.

###### What happens if we reenable the feature if it was previously rolled back?

If the `volumeClaimUpdateStrategy` is already set to `InPlace`,
user needs to update the `volumeClaimTemplates` again to trigger a rollout.

###### Are there any tests for feature enablement/disablement?

Will add unit tests for the StatefulSet controller with and without the feature gate,
`volumeClaimUpdateStrategy` set to `InPlace` and `OnClaimDelete` respectively.

Will add unit tests for exercising the switch of feature gate when `volumeClaimUpdateStrategy` already set.

### Rollout, Upgrade and Rollback Planning

<!--
This section must be completed when targeting beta to a release.
-->

###### How can a rollout or rollback fail? Can it impact already running workloads?

###### What specific metrics should inform a rollback?

###### Were upgrade and rollback tested? Was the upgrade->downgrade->upgrade path tested?

###### Is the rollout accompanied by any deprecations and/or removals of features, APIs, fields of API types, flags, etc.?

### Monitoring Requirements

<!--
This section must be completed when targeting beta to a release.
-->

###### How can an operator determine if the feature is in use by workloads?

###### How can someone using this feature know that it is working for their instance?

- [ ] Events
  - Event Reason:
- [ ] API .status
  - Condition name:
  - Other field:
- [ ] Other (treat as last resort)
  - Details:

###### What are the reasonable SLOs (Service Level Objectives) for the enhancement?

###### What are the SLIs (Service Level Indicators) an operator can use to determine the health of the service?

- [ ] Metrics
  - Metric name:
  - [Optional] Aggregation method:
  - Components exposing the metric:
- [ ] Other (treat as last resort)
  - Details:

###### Are there any missing metrics that would be useful to have to improve observability of this feature?

### Dependencies

###### Does this feature depend on any specific services running in the cluster?

CSI drivers with in-place ExpandVolume or ModifyVolume capabilities,
when `spec.resources.requests.storage` or `spec.volumeAttributesClassName` of `volumeClaimTemplates` is updated respectively.

### Scalability

###### Will enabling / using this feature result in any new API calls?

- PATCH StatefulSet
  - kubectl or other user agents
- PATCH PersistentVolumeClaim (server-side apply)
  - 1 per PVC in the StatefulSet (number of updated claim template * replica)
  - StatefulSet controller (in KCM)
  - triggered by the StatefulSet spec update

StatefulSet controller will watch PVC updates.
(although statefulset controller does not watch PVCs before, KCM does)

###### Will enabling / using this feature result in introducing new API types?

No

###### Will enabling / using this feature result in any new calls to the cloud provider?

Not directly. The cloud provider may be called when the PVCs are updated, by CSI.

###### Will enabling / using this feature result in increasing size or count of the existing API objects?

StatefulSet:
- `spec`: 1 new enum field, ~10B

PersistentVolumeClaim:
- new label `controller-revision-hash` of size 32B

###### Will enabling / using this feature result in increasing time taken by any operations covered by existing SLIs/SLOs?

No.

###### Will enabling / using this feature result in non-negligible increase of resource usage (CPU, RAM, disk, IO, ...) in any components?

The logic of StatefulSet controller is more complex, more CPU will be used.
TODO: measure the actual increase.

###### Can enabling / using this feature result in resource exhaustion of some node resources (PIDs, sockets, inodes, etc.)?

No.

### Troubleshooting

###### How does this feature react if the API server and/or etcd is unavailable?

Not very different from the current StatefulSet controller workflow.

If the API server and/or etcd is unavailable, we either cannot apply the update to PVCs, or cannot gather status of PVCs.
In both cases, the rollout will be blocked until the API server and/or etcd is available again.

###### What are other known failure modes?

- Rollout of the StatefulSet blocked due to failing to update PVCs
  - Detection: apiserver_request_total{resource="persistentvolumeclaims",verb="patch",code!="200"} increased. Events on StatefulSet.
  - Mitigations:
    - Undo `volumeClaimTemplates` changes
    - Set `volumeClaimUpdateStrategy` to `OnClaimDelete`
  - Diagnostics: Events on StatefulSet
  - Testing: Will test the Event is emitted

- Rollout of the StatefulSet blocked due to PVCs never becomes ready, expansion or modify volume failed
  - Detection: Events on PVC. controller_{modify,expand}_volume_errors_total metrics on external-resizer
  - Mitigations:
    - Undo `volumeClaimTemplates` changes
    - Set `volumeClaimUpdateStrategy` to `OnClaimDelete`
    - Edit PVC manually to correct the issue
  - Diagnostics: Events on PVC, logs of external-resizer
  - Testing: No. the error is already reported on the PVC, by external-resizer.

###### What steps should be taken if SLOs are not being met to determine the problem?

When SLOs are not being met, events of PVC or StatefulSet are emitted.
If problem is not determined from events, operator should check whether the PVC spec is updated correctly.
If so, follow the troubleshooting instructions of expanding or modifying volume.
If not, look into the KCM log to determine why the PVC is not updated, raising the log level if necessary.

## Implementation History

- 2024-05-17: Initial version by @huww98 and @vie-serendipity
- 2025-06-09: Targeting v1.34 for alpha
- 2026-06-03: Re-submitted targeting v1.38 for alpha; @darshansreenivas taking over as primary contributor

## Drawbacks

## Alternatives

### Extensively validate the updated `volumeClaimTemplates`

[KEP-0661] proposes that we should do extensive validation on the updated `volumeClaimTemplates`.
e.g., prevent decreasing the storage size, preventing expand if the storage class does not support it.
However, this has several drawbacks:
* If we disallow decreasing, we make the editing a one-way road.
  If a user edited it then found it was a mistake, there is no way back.
  The StatefulSet will be broken forever. If this happens, the updates to pods will also be blocked. This is not acceptable.
* To mitigate the above issue, we will want to prevent the user from going down this one-way road by mistake.
  We are forced to do way more validations on APIServer, which is very complex, and fragile (please see KEP-0661).
  For example: check storage class allowVolumeExpansion, check each PVC's storage class and size,
  basically duplicate all the validations we have done to PVC.
  And even if we do all the validations, there are still race conditions and async failures that we are impossible to catch.
  I see this as a major drawback of KEP-0661 that I want to avoid in this KEP.
* Validation means we should disable rollback of storage size. If we enable it later, it can surprise users, if it is not called a breaking change.
* The validation is conflict to RecoverVolumeExpansionFailure feature.
* `volumeClaimTemplates` is also used when creating new PVCs, so even if the existing PVCs cannot be updated,
  a user may still want to affect new PVCs.
* It violates the high-level design.
  The template describes a desired final state, rather than an immediate instruction.
  A lot of things can happen externally after we update the template.
  For example, I have an IaaC platform, which tries to `kubectl apply` one updated StatefulSet + one new StorageClass to the cluster to trigger the expansion of PVs.
  We don't want to reject it just because the StorageClass is applied after the StatefulSet.

### Support for updating arbitrary fields in `volumeClaimTemplates`

No technical limitations. Just that we want to be careful and keep the changes small, so that we can move faster.
This is just an extra validation in APIServer. We may remove it later if we find it is not needed.

### Patch PVC size regardless of the immutable fields

We propose to patch the PVC as a whole, so it can only succeed if the immutable fields matches.

If only expansion is supported, patching regardless of the immutable fields can be a logical choice.
But this KEP also integrates with volumeAttributesClass (VAC). VAC is closely coupled with storage class.
Only patching VAC if storage class matches is a very logical choice.
And we'd better follow the same operation model for all mutable fields.

### Support for automatically skip not managed PVCs

Introduce a new field in StatefulSet `spec.updateStrategy.rollingUpdate`: `volumeClaimSyncStrategy`.
If it is set to `Async`, then we skip patching the PVCs that are not managed by the StatefulSet (e.g. StorageClass does not match).

The rules to determine what PVCs are managed are a little bit tricky.
We have to check each field, and determine what to do for each field.
This makes us deeply coupled with the PVC implementation.

And still, we want to keep the changes small.

### Reconcile all PVCs regardless of Pod revision labels

Like Pods, we only update the PVCs if the Pod revision labels is not the update revision.

We need to unmarshal all revisions used by Pods to determine the desired PVC spec.
Even if we do so, we don't want to send a apply request for each PVC at each reconcile iteration.
We also don't want to replicate the SSA merging/extraction and validation logic, which can be complex and CPU-intensive.

### Treat all incompatible PVCs as unavailable replicas

Currently, incompatible PVCs only blocks the rolling update, not scaling up or down.
Only the update revision is used for checking.

We need to unmarshal all revisions used by Pods to determine the compatibility.
Even if we do so, old StatefulSets do not have claim info in its history.
If we just use the latest version, then all replicas may suddenly become unavailable,
and all operations are blocked.

[KEP-0661]: https://github.com/kubernetes/enhancements/pull/3412

### Integrate with RecoverVolumeExpansionFailure feature

We may decrease the size in PVC spec automatically to help recover from a failed expansion
if `RecoverVolumeExpansionFailure` feature gate is enabled.
However, when reducing the spec size of PVC, it must still be greater than its status (not equal to).
So we don't know what to set if `volumeClaimTemplates` is smaller than PVC status.

User can still update PVC manually.

### Order of Pod / PVC updates

We've considered delete the Pod while/before updating the PVC, but realized several issues:
* The admission of PVC update is fairly complex, it can fail for many reasons.
  We want to make sure the Pod is still running if we cannot update the PVC.
* As described in [KEP-5381], we want to allow affinity change when the VolumeAttributesClass is updated.
  Updating PVC and Pod concurrently may trigger a race condition where the Pod can be scheduled to wrong node.
* Pod may depend on PVC updates, e.g. when the volume is full. So we should not wait for Pod to be ready before updating PVC.

That left us with two options:
1. Wait for PVC ready before delete old Pod.
2. Wait for new Pod to be scheduled, with all volumes attached before update PVC.

We choose 1 currently. This has an extra advantage:
When Pod is ready, PVCs will almost always be ready too.
So any existing tools to monitor StatefulSet rollout process does not need to change.
But this is not guaranteed. If the Pod is deleted before the PVC is ready (be evicted, or manually),
we still want to ensure maximum Pod availability, so we will still create the Pod.
In this case, the Pod may be ready before PVCs are ready.

We can choose to create Pod at current revision (instead of update revision) if PVCs are not ready.
But there may be some case where the PVCs depends on the new Pod (e.g. old Pod is not schedulable).
We don't want to block them.

This downside is that the concurrency is lower, so the rolling update may take longer.

[KEP-5381]: https://github.com/kubernetes/enhancements/blob/0602a5f744b8e4e201d7bd90eb69e67f1b9baf62/keps/sig-storage/5381-mutable-pv-affinity/README.md#notesconstraintscaveats-optional

### When to track `volumeClaimTemplates` in `ControllerRevision`

The current design tracks volumeClaimTemplates in ControllerRevision only when `volumeClaimUpdateStrategy` is set to `InPlace`.

There are two reasons:
1. We want a new revision to trigger the rollout when `volumeClaimUpdateStrategy` is changed from `OnClaimDelete` to `InPlace`.
2. We want to avoid updating all the Pods under any StatefulSet at once when the feature-gate is enabled, to avoid overloading the control-plane.

If we track volumeClaimTemplates whenever the feature-gate is enabled, we violate all these reasons.

Or we can make this tri-state:
* empty/nil: the default and preserve the current behavior.
* `OnClaimDelete`: Add volumeClaimTemplate to the history, but don't update PVCs
* `InPlace`: Add volumeClaimTemplate to the history, and also update PVCs in-place

While this resolves reason 2, it still violates reason 1.

We can add volumeClaimUpdateStrategy to ControllerRevision to resolve reason 1.
But all the policies we already have does not present in ControllerRevision. So this is not ideal either.

The down-side of the current design is that `kubectl rollout undo` may not work as expected sometimes.

* If `volumeClaimUpdateStrategy` is set to `OnClaimDelete`, `kubectl rollout undo` will not undo the `volumeClaimTemplates`.
* When changing `volumeClaimUpdateStrategy` from `OnClaimDelete` to `InPlace` to trigger the rollout, `kubectl rollout undo` will be no-op.
* Consider the following history:
  1. Pod Rev1 + PVC Rev1 + `OnClaimDelete`
  2. Pod Rev2 + PVC Rev1 + `InPlace`
  3. Pod Rev2 + PVC Rev2 + `InPlace`

  Now if user revert to history 1 directly, `volumeClaimTemplates` will not be reverted.
  But if the user revert to history 2, then history 1, `volumeClaimTemplates` will be reverted.

While somewhat surprising, `kubectl rollout undo` is just a convenient method to update the StatefulSet.
User can always do the update manually. So this is not a big problem.

## Infrastructure Needed (Optional)