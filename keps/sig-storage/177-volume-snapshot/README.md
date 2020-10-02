# KEP-177: CSI Snapshot

## Table of Contents

<!-- toc -->
- [Release Signoff Checklist](#release-signoff-checklist)
- [Summary](#summary)
- [Test Plan](#test-plan)
  - [Unit tests](#unit-tests)
  - [E2E tests](#e2e-tests)
- [Graduation Criteria](#graduation-criteria)
  - [Alpha-&gt;Beta](#alpha-beta)
  - [Beta-&gt;GA](#beta-ga)
- [Snapshot Beta](#snapshot-beta)
  - [API Changes](#api-changes)
  - [Controller Split](#controller-split)
  - [Other Changes Implemented](#other-changes-implemented)
- [Snapshot GA](#snapshot-ga)
  - [Changes Implemented](#changes-implemented)
  - [Additional Changes Planned](#additional-changes-planned)
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
- [Infrastructure Needed (Optional)](#infrastructure-needed-optional)
<!-- /toc -->

## Release Signoff Checklist

<!--
**ACTION REQUIRED:** In order to merge code into a release, there must be an
issue in [kubernetes/enhancements] referencing this KEP and targeting a release
milestone **before the [Enhancement Freeze](https://git.k8s.io/sig-release/releases)
of the targeted release**.

For enhancements that make changes to code or processes/procedures in core
Kubernetes—i.e., [kubernetes/kubernetes], we require the following Release
Signoff checklist to be completed.

Check these off as they are completed for the Release Team to track. These
checklist items _must_ be updated for the enhancement to be released.
-->

Items marked with (R) are required *prior to targeting to a milestone / release*.

- [x] (R) Enhancement issue in release milestone, which links to KEP dir in [kubernetes/enhancements] (not the initial KEP PR)
- [x] (R) KEP approvers have approved the KEP status as `implementable`
- [x] (R) Design details are appropriately documented
- [x] (R) Test plan is in place, giving consideration to SIG Architecture and SIG Testing input
- [x] (R) Graduation criteria is in place
- [ ] (R) Production readiness review completed
- [ ] Production readiness review approved
- [x] "Implementation History" section is up-to-date for milestone
- [x] User-facing documentation has been created in [kubernetes/website], for publication to [kubernetes.io]
- [ ] Supporting documentation—e.g., additional design documents, links to mailing list discussions/SIG meetings, relevant PRs/issues, release notes

<!--
**Note:** This checklist is iterative and should be reviewed and updated every time this enhancement is being considered for a milestone.
-->

[kubernetes.io]: https://kubernetes.io/
[kubernetes/enhancements]: https://git.k8s.io/enhancements
[kubernetes/kubernetes]: https://git.k8s.io/kubernetes
[kubernetes/website]: https://git.k8s.io/website

## Summary

This KEP is written after the original design doc has been approved and implemented. Design for CSI Volume Snapshot Support in Kubernetes is incorporated as part of the [CSI Volume Snapshot in Kubernetes Design Doc](https://github.com/kubernetes/community/blob/master/contributors/design-proposals/storage/csi-snapshot.md).

The rest of the document includes required information missing from the original design document: test plan, graduation criteria, and production readiness review questionnaire.

## Test Plan

### Unit tests

* Unit tests around snapshot creation and deletion logic.
* Unit tests around VolumeSnapshot and VolumeSnapshotContent binding logic.
* Unit tests for creating volume from snapshot.

### E2E tests

* (P0) e2e tests for creating/deleting snapshot.
* (P0) e2e tests for creating volume from snapshot.
* (P1) e2e tests for delete/retain policy.
* (P1) e2e tests for deleting API objects out of order (snapshot protection).
* (P2) e2e tests for secret fields.
* (P2) e2e tests for metrics.

## Graduation Criteria

### Alpha->Beta

* Feature complete, including:
  * Create/delete volume snapshots
  * Create new volumes from a snapshot
  * SnapshotContent Deletion/Retain Policy
  * Snapshot Object in Use Protection
  * Separate the common controller from the sidecar controller
  * Add secrets field to list-snapshots RPC in the CSI spec. Add “data-source-secret” in create-volume intended for accessing the data source. Implement them in external-snapshotter and external-provisioner.
  * Add metrics support to the external-snapshotter sidecar.
* Unit and e2e tests implemented
* Update snapshot CRDs to v1beta1 and enable VolumeSnapshotDataSource feature gate by default.

### Beta->GA

* Snapshot feature is used as a basic building block in other advanced applications. 
* Feature deployed in production and have gone through at least one K8s upgrade.

## Snapshot Beta

### API Changes

A number of changes were made to the Kubernetes volume snapshot API between alpha to beta. These changes are not backward compatible and the alpha API is no longer supported. The purpose of these changes was to make API definitions more clear and easier to use.

The following changes have been made from the Alpha API:

* DeletionPolicy is now a required field rather than optional in both VolumeSnapshotClass and VolumeSnapshotContent. This way the user has to explicitly specify it, leaving no room for confusion.
* VolumeSnapshotSpec has a new required Source field. Source may be either a PersistentVolumeClaimName (if dynamically provisioning a snapshot) or VolumeSnapshotContentName (if pre-provisioning a snapshot).
* VolumeSnapshotContentSpec also has a new required Source field. This Source may be either a VolumeHandle (if dynamically provisioning a snapshot) or a SnapshotHandle (if pre-provisioning volume snapshots).
* VolumeSnapshotStatus now contains a BoundVolumeSnapshotContentName to indicate the VolumeSnapshot object is bound to a VolumeSnapshotContent.
* VolumeSnapshotContent now contains a Status to indicate the current state of the content. It has a field SnapshotHandle which is the unique identifier of a snapshot on the storage system.

The beta VolumeSnapshot API object is as follows:

```
type VolumeSnapshot struct {
        metav1.TypeMeta
        metav1.ObjectMeta

        Spec VolumeSnapshotSpec
        Status *VolumeSnapshotStatus
}
```

```
type VolumeSnapshotSpec struct {
	Source VolumeSnapshotSource
	VolumeSnapshotClassName *string
}
// Exactly one of its members MUST be specified
type VolumeSnapshotSource struct {
	// +optional
	PersistentVolumeClaimName *string
	// +optional
	VolumeSnapshotContentName *string
}
```

```
type VolumeSnapshotStatus struct {
	BoundVolumeSnapshotContentName *string
	CreationTime *metav1.Time
	ReadyToUse *bool
	RestoreSize *resource.Quantity
	Error *VolumeSnapshotError
}
```

The beta VolumeSnapshotContent API object is as follows:

```
type VolumeSnapshotContent struct {
        metav1.TypeMeta
        metav1.ObjectMeta

        Spec VolumeSnapshotContentSpec
        Status *VolumeSnapshotContentStatus
}
```

```
type VolumeSnapshotContentSpec struct {
         VolumeSnapshotRef core_v1.ObjectReference
         Source VolumeSnapshotContentSource
         DeletionPolicy DeletionPolicy
         Driver string
         VolumeSnapshotClassName *string
}
```

```
type VolumeSnapshotContentSource struct {
	// +optional
	VolumeHandle *string
	// +optional
	SnapshotHandle *string
}
```

```
type VolumeSnapshotContentStatus struct {
  CreationTime *int64
  ReadyToUse *bool
  RestoreSize *int64
  Error *VolumeSnapshotError
  SnapshotHandle *string
}
```

The beta Kubernetes VolumeSnapshotClass API object is as follows:

```
type VolumeSnapshotClass struct {
        metav1.TypeMeta
        metav1.ObjectMeta

        Driver string
        Parameters map[string]string
        DeletionPolicy DeletionPolicy
}
```

### Controller Split

Along with VolumeSnapshot being promoted to Beta in Kubernetes 1.17, the CSI external-snapshotter sidecar controller has been split into two controllers: a snapshot-controller and a CSI external-snapshotter sidecar.

The snapshot controller will be watching the Kubernetes API server for `VolumeSnapshot`, `VolumeSnapshotContent`, and `VolumeSnapshotClass` CRD objects. The CSI `external-snapshotter` sidecar watches the Kubernetes API server for `VolumeSnapshotContent` and `VolumeSnapshotClass` CRD objects.

For dynamic provisioning, the creation of a new `VolumeSnapshot` object referencing a `VolumeSnapshotClass` CRD object corresponding to this driver causes the snapshot controller to trigger the creation of a Kubernetes `VolumeSnapshotContent` object to represent the to-be-created new snapshot.

The creation of a new `VolumeSnapshotContent` object causes the sidecar container to trigger a `CreateSnapshot` operation against the specified CSI endpoint to provision a new snapshot. When a new snapshot is successfully provisioned, the sidecar container updates the status field of the `VolumeSnapshotContent` object to represent the new snapshot.

The snapshot controller will be updating the status field of the `VolumeSnapshot` object accordingly based on the status field of the `VolumeSnapshotContent` object to indicate the new snapshot is ready to be used or failed.

The deletion event of a `VolumeSnapshot` object bound to a `VolumeSnapshotContent` corresponding to this driver with a `delete` deletion policy causes the snapshot controller to start deleting the `VolumeSnapshotContent` object and add an annotation to the object to indicate it is being deleted. Note that both the `VolumeSnapshot` object and the `VolumeSnapshotContent` object will not be deleted immediately due to the finalizers. When the sidecar container detects this update on the `VolumeSnapshotContent` object, it triggers a `DeleteSnapshot` operation against the specified CSI endpoint to delete the snapshot. Once the snapshot is successfully deleted, the sidecar container removes the finalizer on the `VolumeSnapshotContent` object which leads to the deletion of the object from Kubernetes. The snapshot controller then removes the finalizer on the `VolumeSnapshot` object and as a result the object will be deleted from Kubernetes. If a user deletes a bound `VolumeSnapshotContent` object directly, it will have a deletion timestamp set however will persist in API server until its corresponding `VolumeSnapshot` object also gets a deletion timestamp set from a deletion request.

If the deletion policy is `retain` when deleting a `VolumeSnapshot` object bound to a `VolumeSnapshotContent`, the finalizers will be removed from both objects, the `VolumeSnapshot` object will be deleted from Kubernetes, but the `VolumeSnapshotContent` and the snapshot on the storage system will remain.

### Other Changes Implemented

Here are the changes since the original design proposal:

* Renamed `Ready` to `ReadyToUse` in the `Status` field of `VolumeSnapshot` API object.
* Changed type of `RestoreSize` in `CSIVolumeSnapshotSource` from `*resource.Quantity`  to `*int64`.
* Lease based Leader Election support is added.
* Added `VolumeSnapshotContent` deletion policy which is also specified in `VolumeSnapshotClass`.
* Added Finalizer on the snapshot source PVC to prevent it from being deleted when a snapshot is being created from it.
* Added Finalizer on the `VolumeSnapshotContent` object to prevent it from being deleted directly from API server when it is bound to the `VolumeSnapshot` object.
* Added Finalizer on the `VolumeSnapshot` object to prevent it from being deleted when it is being used as a source to create a PVC.
* Added Finalizer on the `VolumeSnapshot` object to prevent it from being deleted when it is bound to the `VolumeSnapshotContent` object.
* Added check to see whether ListSnapshots is supported by the CSI driver. If it is supported, ListSnapshots will be called to find out the status of a snapshot during static binding; otherwise it is assumed the snapshot ID provided by the admin is valid.
* Added deletion secret as annotation to volume snapshot content.
* Added prometheus metrics to CSI external-snapshotter under the /metrics endpoint.
* Removed createSnapshotContentRetryCount and createSnapshotContentInterval
from command line options.
* Added a prefix "external-snapshotter-leader" and the driver name to the snapshotter leader election lock name. Rolling update from an earlier version to v2.0.0 will not work if leader election is enabled because the lock name is changed in v2.0.0.

## Snapshot GA

The following changes are either implemented or to be implemented in preparation for moving the snapshot feature to GA.

### Changes Implemented

* If snapshot creation times out, VolumeSnapshot status will not be marked as failed so that controller will continue to retry to create until the operation either succeeds or fails. It is up to the user or an upper level application that uses the VolumeSnapshot to determine what to do with the snapshot.
*  Fixed the re-queue logic so a failed snapshot operation will be added back to a rate limited queue for retries.
* Added secrets field to list-snapshots RPC in the CSI spec. Add “data-source-secret” in create-volume intended for accessing the data source. Implement them in external-snapshotter and external-provisioner.
* Moved snapshot APIs and client lib to a separate sub-module.
* The validation for volume snapshot objects (VolumeSnapshot and VolumeSnapshotContent) is getting more strict. See details in "keps/sig-storage/1900-volume-snapshot-validation-webhook".

### Additional Changes Planned

* Add metrics support in the snapshot controller. Metrics is already added to the external-snapshotter sidecar.
  * snapshot_operation_total_seconds (Snapshot operation end to end duration in number of seconds. Reported from the snapshot controller.)
  * snapshot_operation_count (Total number of operations conducted by the snapshot controller with state changes. Includes an error code to indicate success/failure. Reported from the snapshot controller.)
  * csi_sidecar_operations_seconds (Container Storage Interface operation duration with gRPC error code status total. Reported from CSI external-snapshotter sidecar.)
* Tighten the CRD schema validation to enforce immutability. See details in "keps/sig-storage/1900-volume-snapshot-validation-webhook".

## Production Readiness Review Questionnaire

<!--

Production readiness reviews are intended to ensure that features merging into
Kubernetes are observable, scalable and supportable; can be safely operated in
production environments, and can be disabled or rolled back in the event they
cause increased failures in production. See more in the PRR KEP at
https://git.k8s.io/enhancements/keps/sig-architecture/20190731-production-readiness-review-process.md.

The production readiness review questionnaire must be completed for features in
v1.19 or later, but is non-blocking at this time. That is, approval is not
required in order to be in the release.

In some cases, the questions below should also have answers in `kep.yaml`. This
is to enable automation to verify the presence of the review, and to reduce review
burden and latency.

The KEP must have a approver from the
[`prod-readiness-approvers`](http://git.k8s.io/enhancements/OWNERS_ALIASES)
team. Please reach out on the
[#prod-readiness](https://kubernetes.slack.com/archives/CPNHUMN74) channel if
you need any help or guidance.

-->

### Feature Enablement and Rollback

_This section must be completed when targeting alpha to a release._

* **How can this feature be enabled / disabled in a live cluster?**
  - [x] Feature gate (also fill in values in `kep.yaml`)
    - Feature gate name: VolumeSnapshotDataSource
    - Components depending on the feature gate: kube-apiserver
  - [ ] Other
    - Describe the mechanism:
    - Will enabling / disabling the feature require downtime of the control
      plane?
    - Will enabling / disabling the feature require downtime or reprovisioning
      of a node? (Do not assume `Dynamic Kubelet Config` feature is enabled).

* **Does enabling the feature change any default behavior?**
  Snapshot is an opt-in feature so it should not change any default behavior.
  Enabling the feature gate will enable the volume snapshot as data source to PVC and disabling the feature gate will disable the volume snapshot as data source to PVC. This feature gate can only prevent users from creating a PVC using volume snapshot as a data source. It can't prevent users from creating snapshots because snapshot APIs and controller are out of tree.

* **Can the feature be disabled once it has been enabled (i.e. can we roll back
  the enablement)?**
  Yes. The feature gate can only control creating a new PVC using snapshot as the data source because the snapshot APIs live out of tree. If the feature is disabled after it has been enabled, existing snapshots will still be there but they cannot be used as data source to provision new PVCs and existing PVCs that were created with the snapshot data source will continue to function.

* **What happens if we reenable the feature if it was previously rolled back?**
  If we reenable the feature that was previously rolled back, the existing snapshots can be used again as data source to provision new PVCs.

* **Are there any tests for feature enablement/disablement?**
  The e2e framework does not currently support enabling or disabling feature
  gates. However, unit tests in each component dealing with managing data, created
  with and without the feature, are necessary. At the very least, think about
  conversion tests if API types are being modified.
  Here's a unit test that tests dropping disabled snapshot data source:
  https://github.com/kubernetes/kubernetes/blob/v1.20.0-alpha.1/pkg/api/persistentvolumeclaim/util_test.go#L31
  The original PR for this test is here: https://github.com/kubernetes/kubernetes#72666

### Rollout, Upgrade and Rollback Planning

_This section must be completed when targeting beta graduation to a release._

* **How can a rollout fail? Can it impact already running workloads?**
  Try to be as paranoid as possible - e.g., what if some components will restart
   mid-rollout?
  The feature gate only affects kube-apiserver. If enabling the feature gate fails, already running workloads will not be able to use the snapshot feature to rehydrate PVCs, however, creation of volume snapshot is irrelevant to feature gate.
  Also failed snapshot operations will be added back to a rate limited queue for retries.

* **What specific metrics should inform a rollback?**
  * How many snapshots haven't been taken?
    This is a metric that should be issued by the snapshot controller. This work is in-progress. The metric would cover number of failures, number of snapshot operations finished, and latencies.

  * How many pods using a PVC provisioned from a snapshot datasource are stuck in pending or failing?
    This is not part of the snapshot controller's logic but rather sits in the PV controller and the external provisioner because snapshots can't be mounted directly by a pod but can be used as a datasource to create a new volume. There are metrics around PVC provisioning that shows success/error/time for create volume operations. It currently doesn't distinguish between volumes created from a snapshot vs a blank volume though.

* **Were upgrade and rollback tested? Was the upgrade->downgrade->upgrade path tested?**
  Describe manual testing that was done and the outcomes.
  Longer term, we may want to require automated upgrade/rollback tests, but we
  are missing a bunch of machinery and tooling and can't do that now.
  We plan to do this testing after the code is implemented that moves snapshot API from v1beta1 to v1. We will test the following:
  * Upgrade from v1beta1 to v1. Take a snapshot and create a PVC from a snapshot. Also test delete snapshot and PVC.
  * Downgrade from v1 to v1beta1. Check the snapshot and PVC created above. Take a snapshot and create a PVC from a snapshot. Also test delete snapshot and PVC.
  * Upgrade from v1beta1 to v1 again. Check the snapshot and PVC created earlier. Take a snapshot and create a PVC from a snapshot. Also test delete snapshot and PVC.
  * Also verify snapshots taken with v1beta, can be:
    * restored using v1 + v1beta1
    * deleted using v1 + v1beta1

* **Is the rollout accompanied by any deprecations and/or removals of features, APIs,
fields of API types, flags, etc.?**
  Even if applying deprecation policies, they may still surprise some users.

  We have added a validation webhook that will tighten the API validation. See details in "keps/sig-storage/1900-volume-snapshot-validation-webhook".
  This validating admission webhook should be applied first before going GA to prevent invalid API objects from being created. If the webhook is not applied before going GA, it means there could be invalid API objects created. It also means existing v1beta1 invalid objects will error on any update, including delete. A distribution/admin must not add v1 API until they are sure no invalid v1beta1 objects exist.

  The feature gate is only for creating PVC from a volume snapshot data source. The snapshot controller and validating webhook service are out-of-tree controllers which implement the snapshot feature, and their lifecycle/management/rollback is irrelevant to feature gate. The validation webhook is validating VolumeSnapshot and VolumeSnapshotContent API objects. So if the validation webhook is installed and then the feature gate is disabled, it will just prevent PVC from being created from a snapshot.

### Monitoring Requirements

_This section must be completed when targeting beta graduation to a release._

* **How can an operator determine if the feature is in use by workloads?**
  Ideally, this should be a metric. Operations against the Kubernetes API (e.g.,
  checking if there are objects with field X set) may be a last resort. Avoid
  logs or events for this purpose.
  Metrics support is already added to CSI external-snapshotter sidecar. Metrics support will be added to the snapshot-controller in 1.20.
  Here are the metrics:
  * snapshot_operation_total_seconds (Snapshot operation end to end duration in number of seconds. Reported from the snapshot controller.)
  * snapshot_operation_count (Total number of operations conducted by the snapshot controller with state changes. Includes an error code to indicate success/failure. Reported from the snapshot controller.)
  * csi_sidecar_operations_seconds (Container Storage Interface operation duration with gRPC error code status total. Reported from CSI external-snapshotter sidecar.)

* **What are the SLIs (Service Level Indicators) an operator can use to determine
the health of the service?**
  - [x] Metrics
    - Metric name: snapshot_operation_total_seconds, snapshot_operation_count, csi_sidecar_operations_seconds
      snapshot_operation_count has a status field that shows Error code. So from that we can tell how many failures have occurred in create/delete snapshot operations.
      Another metric we are considering is around the validating webhook itself for conducting validation requests from API server.
      We don't support mounting a snapshot to a pod. Only a volume can be mounted to a pod.
      We also have metrics on volume provisioning, but it doesn't distinguish between datasource. We will consider adding something to differetiate them.
   - [Optional] Aggregation method:
    - Components exposing the metric: snapshot-controller and CSI external-snapshotter sidecar
  - [ ] Other (treat as last resort)
    - Details:

* **What are the reasonable SLOs (Service Level Objectives) for the above SLIs?**
  At a high level, this usually will be in the form of "high percentile of SLI
  per day <= X". It's impossible to provide comprehensive guidance, but at the very
  high level (needs more precise definitions) those may be things like:
  - per-day percentage of API calls finishing with 5XX errors <= 1%
  - 99% percentile over day of absolute value from (job creation time minus expected
    job creation time) for cron job <= 10%
  - 99,9% of /health requests per day finish with 200 code
  We have metrics for failed/successful snapshot operations. We can use that to find out the ratio of failed/successful create/delete snapshot operations.

* **Are there any missing metrics that would be useful to have to improve observability
of this feature?**
  Describe the metrics themselves and the reasons why they weren't added (e.g., cost,
  implementation difficulties, etc.).

### Dependencies

_This section must be completed when targeting beta graduation to a release._

* **Does this feature depend on any specific services running in the cluster?**
  Think about both cluster-level services (e.g. metrics-server) as well
  as node-level agents (e.g. specific version of CRI). Focus on external or
  optional services that are needed. For example, if this feature depends on
  a cloud provider API, or upon an external software-defined storage or network
  control plane.

  For each of these, fill in the following—thinking about running existing user workloads
  and creating new ones, as well as about cluster-level services (e.g. DNS):
  - [Dependency name]: installation of snapshot CRDs, snapshot-controller, snapshot validation webhook, CSI external-snapshotter sidecar
    - Usage description:
      - Impact of its outage on the feature: Installation of snapshot CRDs, snapshot-controller, and CSI external-snapshotter sidecar are required for the volume snapshot feature to work. Snapshot validation webhook is optional. If the validation webhook is not running, API validation will not happen which means there may be invalid snapshot API objects being created.
        The invalid API objects are not usable. We have logic in the controller to check that. That logic was added before the validation web hook was implemented. Since the validation web hook, the snapshot controller, and the snapshot CRDs are all out-of-tree, we don't have a way to make the web hook required. We recommend K8S distro to install them.
      - Impact of its degraded performance or high-error rates on the feature:


### Scalability

_For alpha, this section is encouraged: reviewers should consider these questions
and attempt to answer them._

_For beta, this section is required: reviewers must answer these questions._

_For GA, this section is required: approvers should be able to confirm the
previous answers based on experience in the field._

* **Will enabling / using this feature result in any new API calls?**
  Describe them, providing:
  - API call type (e.g. PATCH pods)

```
    - apiGroups: ["snapshot.storage.k8s.io"]
      resources: ["volumesnapshotclasses"]
      verbs: ["get", "list", "watch"]
    - apiGroups: ["snapshot.storage.k8s.io"]
      resources: ["volumesnapshotcontents"]
      verbs: ["create", "get", "list", "watch", "update", "delete"]
    - apiGroups: ["snapshot.storage.k8s.io"]
      resources: ["volumesnapshots"]
      verbs: ["get", "list", "watch", "update"]
    - apiGroups: ["snapshot.storage.k8s.io"]
      resources: ["volumesnapshots/status"]
      verbs: ["update"]
    - apiGroups: ["snapshot.storage.k8s.io"]
      resources: ["volumesnapshotcontents/status"]
      verbs: ["update"]
```
    There are also calls to existing APIs:
    - list, get, update persistentvolumeclaims
      - update is for adding a finalizer on PVC when creating a VolumeSnapshot from it
    - get persistentvolumes
    - get storageclasses

  - estimated throughput
    We have plan to add stress tests which can help us get maximum throughput achievable.
  - originating component(s) (e.g. Kubelet, Feature-X-controller)
  focusing mostly on:
  - components listing and/or watching resources they didn't before
    snapshot-controller and CSI external-snapshotter sidecar
  - API calls that may be triggered by changes of some Kubernetes resources
    (e.g. update of object X triggers new updates of object Y)
  - periodic API calls to reconcile state (e.g. periodic fetching state,
    heartbeats, leader election, etc.)
    leader election

* **Will enabling / using this feature result in introducing new API types?**
  Describe them, providing:
  - API type: volumesnapshots, volumesnapshotcontents, and volumesnapshotclasses are CRDs
  - Supported number of objects per cluster
  - Supported number of objects per namespace (for namespace-scoped objects)

* **Will enabling / using this feature result in any new calls to the cloud
provider?**
  If a cloud provider has implemented CSI snapshot functionality, it will be called when snapshot operations are triggered.

* **Will enabling / using this feature result in increasing size or count of
the existing API objects?**
  Describe them, providing:
  - API type(s): New CRDs volumesnapshots, volumesnapshotcontents, and volumesnapshotclasses are added
    PVC object also increases because of datasource.
  - Estimated increase in size: (e.g., new annotation of size 32B)
    - Size of VolumeSnapshot:  312 bytes
    - Size of VolumeSnapshotContent:  456 bytes
    - Size of class:  320 bytes
    - Size of DataSource field added to PVC: 48 bytes
    - Size of Finalizer added to PVC: 32 bytes

  - Estimated amount of new objects: (e.g., new Object X for every existing Pod)
    * Each PV can have multiple snapshots. The limit depends on individual storage systems.
    * There's quota support for namespaced custom resources as described here:
https://kubernetes.io/docs/concepts/policy/resource-quotas/#object-count-quota
    * We can have quota support for VolumeSnapshot resource which is namespaced. This is created by a user.
    * VolumeSnapshotContent is cluster scoped and created by the cluster admin, but there is usually a 1 to 1 mapping between the VolumeSnapshot and VolumeSnapshotContent, although admin can create more VolumeSnapshotContents if he/she wants to.
    * VolumeSnapshotClass is cluster scoped and created by cluster-admin.

* **Will enabling / using this feature result in increasing time taken by any
operations covered by [existing SLIs/SLOs]?**
  Think about adding additional work or introducing new steps in between
  (e.g. need to do X to start a container), etc. Please describe the details.
  This is a new feature so it will not increase time taken by existing operations. However, creating volumes from snapshots will probably take longer than an empty disk, especially if the snapshot needs to be downloaded from an object store.

* **Will enabling / using this feature result in non-negligible increase of
resource usage (CPU, RAM, disk, IO, ...) in any components?**
  Things to keep in mind include: additional in-memory state, additional
  non-trivial computations, excessive access to disks (including increased log
  volume), significant amount of data sent and/or received over network, etc.
  This through this both in small and large cases, again with respect to the
  [supported limits].
  This will add new snapshot API objects to the API server.
  The snapshot controller, validation webhook service, and snapshotter sidecar will consume cpu/memory from the control plane.
  Copy on write is typically used by snapshot technologies, but our snapshot API does not make any requirements on that because it depends on individual storage systems.
  The application is usually quiesced before taking a snapshot to ensure consistency, therefore taking a snapshot should not drive more IO on the nodes.
  Some storage systems upload the snapshot to an object store after the snapshot is cut. That could take a very long time.

### Troubleshooting

The Troubleshooting section currently serves the `Playbook` role. We may consider
splitting it into a dedicated `Playbook` document (potentially with some monitoring
details). For now, we leave it here.

_This section must be completed when targeting beta graduation to a release._

* **How does this feature react if the API server and/or etcd is unavailable?**
  If API server is unavailable, an API update will fail due to timeout. Failed operations will be added back to a rate limited queue for retries.
  A CreateSnapshot call to CSI driver is idempotent. So when API server is back and the same request is sent to the CSI driver again, the CSI driver should return the results from the same snapshot.

* **What are other known failure modes?**
  For each of them, fill in the following information by copying the below template:
  - [Failure mode brief description]
    - Detection: How can it be detected via metrics? Stated another way:
      how can an operator troubleshoot without logging into a master or worker node?
      * An operator can check events on the snapshot objects.
      * An operator can also take a look of the metrics of snapshot operations.

    - Mitigations: What can be done to stop the bleeding, especially for already
      running user workloads?
    - Diagnostics: What are the useful log messages and their required logging
      levels that could help debug the issue?
      Not required until feature graduated to beta.
      * Debug level can be enabled to collect logs. Here are examples of enabling verbose logging in the snapshot-controller and CSI snapshotter:
https://github.com/kubernetes-csi/external-snapshotter/blob/v3.0.0/deploy/kubernetes/snapshot-controller/setup-snapshot-controller.yaml#L30
https://github.com/kubernetes-csi/external-snapshotter/blob/master/deploy/kubernetes/csi-snapshotter/setup-csi-snapshotter.yaml#L89

    - Testing: Are there any tests for failure mode? If not, describe why.
      There are negative unit tests and e2e tests.

* **What steps should be taken if SLOs are not being met to determine the problem?**
  If SLOs are not being met, analysis should be made to understand what have caused the problem. Debug level logging should be enabled to collect verbose logs. Look at logs to find out what might have caused the snapshot operation to fail. If it indicates an underlying problem on the storage system, then storage admin can be pulled in to help find the root cause. If the operation times out, check if the underlying storage system is still responding and check if any Kubernetes component goes down. Check if kube-api-server, kube-controller manager, etc. are still up. Check if kubelet is running on the worker nodes and whether worker nodes are down. Check if snapshot-controller, CSI snapshotter sidecar are still running.

[supported limits]: https://git.k8s.io/community//sig-scalability/configs-and-limits/thresholds.md
[existing SLIs/SLOs]: https://git.k8s.io/community/sig-scalability/slos/slos.md#kubernetes-slisslos

## Implementation History

K8s 1.12: alpha
K8s 1.17: beta
K8s 1.20: ga
Repo: https://github.com/kubernetes-csi/external-snapshotter

## Drawbacks

<!--
Why should this KEP _not_ be implemented?
-->

## Alternatives

<!--
What other approaches did you consider, and why did you rule them out? These do
not need to be as detailed as the proposal, but should include enough
information to express the idea and why it was not acceptable.
-->

## Infrastructure Needed (Optional)

<!--
Use this section if you need things from the project/SIG. Examples include a
new subproject, repos requested, or GitHub details. Listing these here allows a
SIG to get the process for these resources started right away.
-->
