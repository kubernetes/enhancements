# KEP-3476: Volume Group Snapshot

## Table of Contents

<!-- toc -->
- [Release Signoff Checklist](#release-signoff-checklist)
- [Summary](#summary)
- [Motivation](#motivation)
  - [Goals](#goals)
  - [Non Goals](#non-goals)
- [Proposal for VolumeGroupSnapshot](#proposal-for-volumegroupsnapshot)
  - [Create VolumeGroupSnapshot](#create-volumegroupsnapshot)
    - [Dynamic provisioning](#dynamic-provisioning)
    - [Pre-provisioned VolumeGroupSnapshot](#pre-provisioned-volumegroupsnapshot)
  - [Delete VolumeGroupSnapshot](#delete-volumegroupsnapshot)
  - [Restore](#restore)
  - [Risks and Mitigations](#risks-and-mitigations)
- [Design Details](#design-details)
  - [Test Plan](#test-plan)
      - [Prerequisite testing updates](#prerequisite-testing-updates)
      - [Unit tests](#unit-tests)
      - [Integration tests](#integration-tests)
      - [e2e tests](#e2e-tests)
  - [Graduation Criteria](#graduation-criteria)
    - [Alpha](#alpha)
    - [Alpha -&gt; Beta](#alpha---beta)
    - [Beta -&gt; GA](#beta---ga)
    - [Deprecation](#deprecation)
  - [Upgrade / Downgrade Strategy](#upgrade--downgrade-strategy)
  - [Version Skew Strategy](#version-skew-strategy)
  - [API Definitions](#api-definitions)
    - [VolumeGroupSnapshotClass](#volumegroupsnapshotclass)
    - [VolumeGroupSnapshot](#volumegroupsnapshot)
    - [VolumeGroupSnapshotContent](#volumegroupsnapshotcontent)
    - [VolumeSnapshot and VolumeSnapshotContent](#volumesnapshot-and-volumesnapshotcontent)
  - [Example Yaml Files](#example-yaml-files)
    - [Create VolumeGroupSnapshot](#create-volumegroupsnapshot-1)
  - [CSI Changes](#csi-changes)
    - [CSI Capabilities](#csi-capabilities)
    - [CSI Group Controller RPC](#csi-group-controller-rpc)
    - [CreateVolumeGroupSnapshot](#createvolumegroupsnapshot)
    - [DeleteVolumeGroupSnapshot](#deletevolumegroupsnapshot)
    - [GetVolumeGroupSnapshot](#getvolumegroupsnapshot)
- [Production Readiness Review Questionnaire](#production-readiness-review-questionnaire)
  - [Feature enablement and rollback](#feature-enablement-and-rollback)
  - [Rollout, Upgrade and Rollback Planning](#rollout-upgrade-and-rollback-planning)
  - [Monitoring Requirements](#monitoring-requirements)
  - [Dependencies](#dependencies)
  - [Scalability](#scalability)
  - [Troubleshooting](#troubleshooting)
- [Implementation History](#implementation-history)
- [Drawbacks](#drawbacks)
- [Alternatives](#alternatives)
  - [Immutable VolumeGroup](#immutable-volumegroup)
  - [ModifyVolume](#modifyvolume)
  - [VolumeGroup API Definitions](#volumegroup-api-definitions)
<!-- /toc -->

## Release Signoff Checklist

<!--
**ACTION REQUIRED:** In order to merge code into a release, there must be an
issue in [kubernetes/enhancements] referencing this KEP and targeting a release
milestone **before the [Enhancement Freeze](https://git.k8s.io/sig-release/releases)
of the targeted release**.

For enhancements that make changes to code or processes/procedures in core
Kubernetes i.e., [kubernetes/kubernetes], we require the following Release
Signoff checklist to be completed.

Check these off as they are completed for the Release Team to track. These
checklist items _must_ be updated for the enhancement to be released.
-->

Items marked with (R) are required *prior to targeting to a milestone / release*.

- [x] (R) Enhancement issue in release milestone, which links to KEP dir in [kubernetes/enhancements] (not the initial KEP PR)
- [ ] (R) KEP approvers have approved the KEP status as `implementable`
- [x] (R) Design details are appropriately documented
- [x] (R) Test plan is in place, giving consideration to SIG Architecture and SIG Testing input
- [x] (R) Graduation criteria is in place
- [x] (R) Production readiness review completed
- [x] Production readiness review approved
- [x] "Implementation History" section is up-to-date for milestone
- [ ] User-facing documentation has been created in [kubernetes/website], for publication to [kubernetes.io]
- [ ] Supporting documentation e.g., additional design documents, links to mailing list discussions/SIG meetings, relevant PRs/issues, release notes

<!--
**Note:** This checklist is iterative and should be reviewed and updated every time this enhancement is being considered for a milestone.
-->

[kubernetes.io]: https://kubernetes.io/
[kubernetes/enhancements]: https://git.k8s.io/enhancements
[kubernetes/kubernetes]: https://git.k8s.io/kubernetes
[kubernetes/website]: https://git.k8s.io/website

## Summary

This proposal introduces a Kubernetes API that allows users to take a crash consistent snapshot of multiple volumes together. It uses a label selector to group multiple persistent volume claims together for snapshotting. This design is proposed to add the volume group snapshot support for CSI Volume Drivers. The CSI volume group snapshot spec is proposed [here](https://github.com/container-storage-interface/spec/pull/519).

## Motivation

There is already a [VolumeSnapshot API](https://github.com/kubernetes/enhancements/tree/master/keps/sig-storage/177-volume-snapshot) that provides the ability to take a snapshot of a persistent volume to protect against data loss or data corruption. However, there are other snapshotting functionalities not covered by the VolumeSnapshot API.

Some storage systems support consistent group snapshot that allows a snapshot to be taken from multiple volumes at the same point-in-time to achieve write order consistency. This can be useful for applications that contain multiple volumes. For example, an application may have data stored in one volume and logs stored in another volume. If snapshots for the data volume and the logs volume are taken at different times, the application will not be consistent and will not function properly if it is restored from those snapshots when an disaster strikes.

It is true that we can quiesce the application first, take an individual snapshot from each volume that is part of the application one after the other, and then unquiesce the application after all the individual snapshots are taken. This way we will get application consistent snapshots. However, application quiesce is time consuming. Sometimes it may not be possible to quiesce an application. Taking individual snapshots one after another may also take longer time compared to taking a consistent group snapshot. Some users may not want to do application quiesce very frequently for these reasons. For example, a user may want to run weekly backups with application quiesce and nightly backups without application quiesce but with consistency group support which provides crash consistency across all volumes in group.

There is also another KEP (https://github.com/kubernetes/enhancements/pull/1051) that introduces APIs to do application snapshot, backup, and restore, but that KEP has a broader scope. In other words, volume group snapshot proposed in this KEP can be used by the application snapshot proposed in the other KEP.

### Goals

* Provide an API to take a snapshot of multiple volumes together.

### Non Goals

* Provide an API to manage multiple volumes together in a group.
* Provide a generic and extensible group API that may be used to support other features in the future.
* Provide a VolumeGroup API that supports group replication.
* Provide a design to facilitate volume placement using the group API.

## Proposal for VolumeGroupSnapshot

This proposal introduces new CRDs VolumeGroupSnapshot, VolumeGroupSnapshotContent, and VolumeGroupSnapshotClass.

### Create VolumeGroupSnapshot

A VolumeGroupSnapshot can be created from multiple PVCs with a label on the PVCs specified by the labelSelector in the VolumeGroupSnapshot if the CSI driver supports the CREATE_DELETE_GET_VOLUME_GROUP_SNAPSHOT capability.

Note: In the following, we will use VolumeGroupSnapshot Controller to refer to the control logic for VolumeGroupSnapshot. This is not a new controller. It will be new control logic added to the existing Snapshot Controller and the csi-snapshotter sidecar.

#### Dynamic provisioning

* Admin creates a VolumeGroupSnapshotClass.
* User creates a VolumeGroupSnapshot with label selector that matches the label applied to all PVCs to be snapshotted together.
* This will trigger the VolumeGroupSnapshot controller to create a VolumeGroupSnapshotContent API object, and also call the CreateVolumeGroupSnapshot CSI function.
* The controller will retrieve all volumeSnapshotHandles in the Volume Group Snapshot from the CSI CreateVolumeGroupSnapshotResponse, create VolumeSnapshotContents pointing to the volumeSnapshotHandles. Then the controller will create VolumeSnapshots pointing to the VolumeSnapshotContents.
* CreateVolumeGroupSnapshot CSI function response
  * The CreateVolumeGroupSnapshot CSI function should return a list of snapshots (Snapshot message defined in CSI Spec) in its response. The VolumeGroupSnapshot controller can use the returned list of snapshots to construct corresponding individual VolumeSnapshotContents and VolumeSnapshots, wait for VolumeSnapshots and VolumeSnapshotContents to be bound, and update SnapshotRefList in the VolumeGroupSnapshot Status and SnapshotContentList in the VolumeGroupSnapshotContent Status.
 * Individual VolumeSnapshots will be named in this format:
   * <snap>-<hash of VolumeGroupSnapshot UUID+PVC UUID+timestamp>
   * A label with VolumeGroupSnapshot name will also be added to the VolumeSnapshot

```
apiVersion: snapshot.storage.k8s.io/v1
kind: VolumeSnapshot
metadata:
  name: snapshot1
  labels:
    volumeGroupSnapshotName: groupSnapshot1
spec:
  source:
    persistentVolumeClaimName: vsc1
status:
  volumeGroupSnapshotName: groupSnapshot1
```

* An admissions controller or finalizer should be added to prevent an individual snapshot from being deleted that belongs to a VolumeGroupSnapshot. Note that there is a [KEP](https://github.com/kubernetes/enhancements/pull/2840/files) that is proposing the Liens feature which could potentially be used for this purpose.
* In the CSI spec, it is specified that it is required for individual snapshots to be returned along with the group snapshot.

#### Pre-provisioned VolumeGroupSnapshot

Admin can create a VolumeGroupSnapshotContent, specifying an existing VolumeGroupSnapshotHandle in the storage system and specifying a VolumeGroupSnapshot name and namespace. Then the user creates a VolumeGroupSnapshot that points to the VolumeGroupSnapshotContent name.

The controller will call the CSI GetVolumeGroupSnapshot method to retrieve all volumeSnapshotHandles in the Volume Group Snapshot from the storage system, create VolumeSnapshotContents pointing to the volumeSnapshotHandles. Then the controller will create VolumeSnapshots pointing to the VolumeSnapshotContents.

### Delete VolumeGroupSnapshot

A VolumeGroupSnapshot can be deleted if the CSI driver supports the CREATE_DELETE_GET_VOLUME_GROUP_SNAPSHOT capability.
* When a VolumeGroupSnapshot is deleted, the VolumeGroupSnapshot controller will call the DeleteVolumeGroupSnapshot CSI function which will delete individual snapshots as well.
  * Since CSI driver handles individual snapshot creation in CreateVolumeGroupSnapshot, it should handle individual snapshot deletion in DeleteVolumeGroupSnapshot as well. DeleteSnapshot CSI function will not be called.
  * When DeleteVolumeGroupSnapshot CSI function returns success, it is assumed that all individual snapshots on the storage system have been deleted. VolumeGroupSnapshot controller should remove all the finalizers and delete the VolumeSnapshot and VolumeSnapshotContent API objects.
* DeleteSnapshot on a single snapshot that belongs to a group snapshot is not allowed.
* The Snapshot Controller and csi-snapshotter sidecar will be modified to skip the handling of VolumeSnapshot and VolumeSnapshotContent deletion if they are part of a group.

### Restore

Restore can be done as follows:

A new volume can be created from a snapshot. This can be repeated for all the snapshots in the VolumeGroupSnapshot.


### Risks and Mitigations

This feature requires coordination between several controllers including the newly proposed volume group snapshot controller and existing external-snapshotter components. We will introduce this feature as alpha and add tests to make sure it works properly.

## Design Details

### Test Plan

[X] I/we understand the owners of the involved components may require updates to existing tests to make this code solid enough prior to committing the changes necessary to implement this enhancement.

##### Prerequisite testing updates
N/A

##### Unit tests
* Unit tests for external volume group snapshot controller.
* Unit tests for modified code path of external-snapshotter.

##### Integration tests
Integration tests are not needed.

##### e2e tests
* e2e tests for external volume group snapshot controller.
* e2e tests for modified code path of external-snapshotter.
* Add stress and scale tests before moving from beta to GA.

### Graduation Criteria
#### Alpha
* Initial feature implementation, including:
  * Volume group snapshot.
* Sample implementation in the csi-driver-host-path.
* Reviews from vendors whose storage systems can support this feature.
* Add basic unit tests.

#### Alpha -> Beta
* Unit tests and e2e tests outlined in design proposal implemented.

#### Beta -> GA
* Volume group snapshot support is added to multiple CSI drivers.
* Volume group snapshot feature deployed in production and have gone through at least one K8s upgrade.

#### Deprecation
<!--
- Announce deprecation and support policy of the existing flag
- Two versions passed since introducing the functionality that deprecates the flag (to address version skew)
- Address feedback on usage/changed behavior, provided on GitHub issues
- Deprecate the flag
-->
No deprecation plan.

### Upgrade / Downgrade Strategy

<!--
If applicable, how will the component be upgraded and downgraded? Make sure
this is in the test plan.

Consider the following in developing an upgrade/downgrade strategy for this
enhancement:
- What changes (in invocations, configurations, API use, etc.) is an existing
  cluster required to make on upgrade, in order to maintain previous behavior?
- What changes (in invocations, configurations, API use, etc.) is an existing
  cluster required to make on upgrade, in order to make use of the enhancement?
-->
External controller handling volume group snapshot is additional sidecar deployed with the CSI driver. External-snapshotter components will be updated to use the newer version that supports this feature. Upgrade should be fine as long as all the components are updated accordingly. Before downgrade, newly created volume group snapshots which depend on the new CRDs should be deleted.

### Version Skew Strategy

<!--
If applicable, how will the component handle version skew with other
components? What are the guarantees? Make sure this is in the test plan.

Consider the following in developing a version skew strategy for this
enhancement:
- Does this enhancement involve coordinating behavior in the control plane and
  in the kubelet? How does an n-2 kubelet without this feature available behave
  when this feature is used?
- Will any other components on the node change? For example, changes to CSI,
  CRI or CNI may require updating that component before the kubelet.
-->
The enhancement only affects the control plane but there are multiple components involved. If the controllers are updated to support this feature but the CSI driver itself does not support it, the `Ready` status of a new VolumeGroupSnapshot API object will stay `false`.

### API Definitions

API definitions are as follows:

#### VolumeGroupSnapshotClass

```
type VolumeGroupSnapshotClass struct {
        metav1.TypeMeta
        // +optional
        metav1.ObjectMeta

        // Driver is the driver expected to handle this VolumeGroupSnapshotClass.
        // This value may not be empty.
        Driver string

        // Parameters hold parameters for the driver.
        // These values are opaque to the system and are passed directly
        // to the driver.
        // +optional
        Parameters map[string]string

        // +optional
        VolumeGroupSnapshotDeletionPolicy *VolumeGroupSnapshotDeletionPolicy
}

// VolumeGroupSnapshotDeletionPolicy describes a policy for end-of-life maintenance of
// volume group snapshot contents
type VolumeGroupSnapshotDeletionPolicy string

const (
        // VolumeGroupSnapshotContentDelete means the group snapshot will be deleted from the
        // underlying storage system on release from its volume group snapshot.
        VolumeGroupSnapshotContentDelete VolumeGroupSnapshotDeletionPolicy = "Delete"

        // VolumeGroupSnapshotContentRetain means the group snapshot will be left in its current
        // state on release from its volume group snapshot.
        VolumeGroupSnapshotContentRetain VolumeGroupSnapshotDeletionPolicy = "Retain"
)

```

#### VolumeGroupSnapshot

```
// VolumeGroupSnapshot is a user's request for taking a group snapshot.
type VolumeGroupSnapshot struct {
        metav1.TypeMeta `json:",inline"`
        // Standard object's metadata.
        // +optional
        metav1.ObjectMeta `json:"metadata,omitempty" protobuf:"bytes,1,opt,name=metadata"`

        // Spec defines the desired characteristics of a group snapshot requested by a user.
        Spec VolumeGroupSnapshotSpec `json:"spec" protobuf:"bytes,2,opt,name=spec"`

        // Status represents the latest observed state of the group snapshot
        // +optional
        Status *VolumeGroupSnapshotStatus `json:"status,omitempty" protobuf:"bytes,3,opt,name=status"`
}

// VolumeGroupSnapshotSpec describes the common attributes of a group snapshot
type VolumeGroupSnapshotSpec struct {
        // VolumeGroupSnapshotClassName may be left nil to indicate that
        // the default class will be used.
        // Empty string is not allowed for this field.
        // +optional
        VolumeGroupSnapshotClassName *string

        // A label query over persistent volume claims to be grouped together
        // for snapshotting.
        // This labelSelector will be used to match the label added to a PVC.
        // Note that if volumes are added/removed from the label after a group snapshot
        // is created, the existing snapshots won't be modified.
        // Once a VolumeGroupSnapshotContent is created and the sidecar starts to process it,
        // the volume list will not change with retries.
        Selector *metav1.LabelSelector

        // VolumeGroupSnapshotSecretRef is a reference to the secret object containing
        // sensitive information to pass to the CSI driver to complete the CSI
        // calls for VolumeGroupSnapshots.
        // This field is optional, and may be empty if no secret is required. If the
        // secret object contains more than one secret, all secrets are passed.
        // +optional
        VolumeGroupSnapshotSecretRef *SecretReference
}

Type VolumeGroupSnapshotStatus struct {
        // +optional
        BoundVolumeGroupSnapshotContentName *string

        // ReadyToUse becomes true when ReadyToUse on all individual snapshots become true
        // +optional
        ReadyToUse *bool

        // +optional
        CreationTime *metav1.Time

        // +optional
        Error *VolumeSnapshotError

        // List of volume snapshot refs
        // The max number of snapshots in the group is 100
        // +optional
        VolumeSnapshotRefList []core_v1.ObjectReference
}
```

#### VolumeGroupSnapshotContent

```
// VolumeGroupSnapshotContent
type VolumeGroupSnapshotContent struct {
        metav1.TypeMeta `json:",inline"`
        // Standard object's metadata.
        // +optional
        metav1.ObjectMeta `json:"metadata,omitempty" protobuf:"bytes,1,opt,name=metadata"`

        // Spec defines the desired characteristics of a group snapshot content
        Spec VolumeGroupSnapshotContentSpec `json:"spec" protobuf:"bytes,2,opt,name=spec"`

        // Status represents the latest observed state of the group snapshot content
        // +optional
        Status *VolumeGroupSnapshotContentStatus `json:"status,omitempty" protobuf:"bytes,3,opt,name=status"`
}

// VolumeGroupSnapshotContentSpec describes the common attributes of a group snapshot content
type VolumeGroupSnapshotContentSpec struct {
        // Required
        // VolumeGroupSnapshotRef specifies the VolumeGroupSnapshot object
        // to which this VolumeGroupSnapshotContent object is bound.
        VolumeGroupSnapshotRef core_v1.ObjectReference

        // Required
        VolumeGroupSnapshotDeletionPolicy VolumeGroupSnapshotDeletionPolicy

        // Required
        Driver string

        // This field may be unset for pre-provisioned snapshots.
        // For dynamic provisioning, this field must be set.
        // +optional
        VolumeGroupSnapshotClassName *string

        // Required
        Source VolumeGroupSnapshotContentSource
}

// OneOf
type VolumeGroupSnapshotContentSource struct {
        // Dynamic provisioning of VolumeGroupSnapshot
        // A list of PersistentVolume names to be snapshotted together
        // +optional
        PersistentVolumeNames []string

        // Pre-provisioned VolumeGroupSnapshot
        // +optional
        VolumeGroupSnapshotHandle *string
}

Type VolumeGroupSnapshotContentStatus struct {
        // VolumeGroupSnapshotHandle is a unique id returned by the CSI driver
        // to identify the VolumeGroupSnapshot on the storage system.
        // If a storage system does not provide such an id, the
        // CSI driver can choose to return the VolumeGroupSnapshot name.
        // +optional
        VolumeGroupSnapshotHandle *string

        // ReadyToUse becomes true when ReadyToUse on all individual snapshots become true
        // +optional
        ReadyToUse *bool

        // +optional
        CreationTime *int64

        // +optional
        Error *VolumeSnapshotError

        // List of volume snapshot content refs
        // The max number of snapshots in a group is 100
        // +optional
        VolumeSnapshotContentRefList []core_v1.ObjectReference
}
```

#### VolumeSnapshot and VolumeSnapshotContent

For VolumeSnapshot, we cannot request a VolumeSnapshot to be added to be VolumeGroupSnapshot, therefore VolumeGroupSnapshotName is only in the Status but not in the Spec.

```
type VolumeSnapshotStatus struct{
	......
        // +optional
        VolumeGroupSnapshotName *string
	......
}

type VolumeSnapshotContentStatus struct{
	......
        // +optional
        VolumeGroupSnapshotContentName *string
	......
}
```

### Example Yaml Files

#### Create VolumeGroupSnapshot

Create a VolumeGroupSnapshotClass:
```
apiVersion: volumegroup.storage.k8s.io/v1alpha1
kind: VolumeGroupSnapshotClass
metadata:
  name: volumeGroupSnapshotClass1
spec:
  parameters:
     …...
```

A VolumeGroupSnapshot taken from multiple volumes dynamically:
```
apiVersion: volumegroup.storage.k8s.io/v1alpha1
kind: VolumeGroupSnapshot
metadata:
  name: my-group-snapshot
spec:
  selector:
    myapp: postgresql
  volumeGroupSnapshotClassName: volumeGroupSnapshotClass1
```

The new VolumeGroupSnapshot logic will be added to the Snapshot Controller and the csi-snapshotter sidecar to handle VolumeGroupSnapshotClass, VolumeGroupSnapshot, and VolumeGroupSnapshotContent resources accordingly.

Snapshot controller will also be modified so that it will not delete an indiviual VolumeSnapshot that is part of a VolumeGroupSnapshot. External snapshotter sidecar will be modified so that it will not delete an individual VolumeSnapshotContent that is part of a VolumeGroupSnapshotContent.

### CSI Changes

#### CSI Capabilities

A new group controller service will be added with a new controller capability CREATE_DELETE_GET_VOLUME_GROUP_SNAPSHOT.

* CREATE_DELETE_GET_VOLUME_GROUP_SNAPSHOT:
  Indicates that the controller plugin supports creating, deleting, and getting details of a snapshot of
  multiple volumes.

#### CSI Group Controller RPC

```
service Controller {
  …
  rpc CreateVolumeGroupSnapshot(CreateVolumeGroupSnapshotRequest)
    returns (CreateVolumeGroupSnapshotResponse) {
        option (alpha_method) = true;
    }

  rpc DeleteVolumeGroupSnapshot(DeleteVolumeGroupSnapshotRequest)
    returns (DeleteVolumeGroupSnapshotResponse) {
        option (alpha_method) = true;
    }

  rpc GetVolumeGroupSnapshot(GetVolumeGroupSnapshotRequest)
    returns (GetVolumeGroupSnapshotResponse) {
        option (alpha_method) = true;
    }
  …
}
```

#### CreateVolumeGroupSnapshot

The purpose of this call is to request the creation of a multi-volume snapshot. Group snapshots can be created from multiple volumes. Note that calls to this function must be idempotent - the function may be called multiple times for the same name - the group snapshot must only be created once.

```
message CreateVolumeGroupSnapshotRequest {
  option (alpha_message) = true;

  // The suggested name for the group snapshot. This field is REQUIRED
  // for idempotency.
  // Any Unicode string that conforms to the length limit is allowed
  // except those containing the following banned characters:
  // U+0000-U+0008, U+000B, U+000C, U+000E-U+001F, U+007F-U+009F.
  // (These are control characters other than commonly used whitespace.)
  string name = 1;

  // volume ids of the source volumes to be snapshotted together.
  // This field is REQUIRED.
  repeated string source_volume_ids = 2;

  // Secrets required by plugin to complete
  // ControllerCreateVolumeGroupSnapshot request.
  // This field is OPTIONAL. Refer to the `Secrets Requirements`
  // section on how to use this field.
  // The secrets provided in this field SHOULD be the same for
  // all group snapshot operations on the same group snapshot.
  map<string, string> secrets = 3 [(csi_secret) = true];

  // Plugin specific parameters passed in as opaque key-value pairs.
  // This field is OPTIONAL. The Plugin is responsible for parsing and
  // validating these parameters. COs will treat these as opaque.
  map<string, string> parameters = 4;
}

message CreateVolumeGroupSnapshotResponse {
  option (alpha_message) = true;

  // Contains all attributes of the newly created group snapshot.
  // This field is REQUIRED.
  VolumeGroupSnapshot group_snapshot = 1;
}

message VolumeGroupSnapshot {
  option (alpha_message) = true;

  // The identifier for this group snapshot, generated by the plugin.
  // This field MUST contain enough information to uniquely identify
  // this specific snapshot vs all other group snapshots supported by
  // this plugin.
  // This field SHALL be used by the CO in subsequent calls to refer to
  // this group snapshot.
  // The SP is NOT responsible for global uniqueness of
  // group_snapshot_id across multiple SPs.
  // This field is REQUIRED.
  string group_snapshot_id = 1;

  // A list of snapshots created.
  // This field is REQUIRED.
  repeated Snapshot snapshots = 2;

  // Timestamp when the volume group snapshot is taken.
  // This field is REQUIRED.
  .google.protobuf.Timestamp creation_time = 3;

  // Indicates if all individual snapshots in the group snapshot
  // are ready to use as a `volume_content_source` in a
  // `CreateVolumeRequest`. The default value is false.
  // If any snapshot in the list of snapshots in this message have
  // ready_to_use set to false, the SP MUST set this field to false.
  // If all of the snapshots in the list of snapshots in this message
  // have ready_to_use set to true, the SP SHOULD set this field to
  // true.
  // This field is REQUIRED.
  bool ready_to_use = 4;
}
```

#### DeleteVolumeGroupSnapshot

```
message DeleteVolumeGroupSnapshotRequest {
  option (alpha_message) = true;

  // The ID of the group snapshot to be deleted.
  // This field is REQUIRED.
  string group_snapshot_id = 1;

  // A list of snapshot ids that are part of this group snapshot.
  // Some SPs require this list to delete the snapshots in the group.
  // This field is REQUIRED.
  repeated string snapshot_ids = 2;

  // Secrets required by plugin to complete group snapshot deletion
  // request.
  // This field is OPTIONAL. Refer to the `Secrets Requirements`
  // section on how to use this field.
  // The secrets provided in this field SHOULD be the same as
  // the secrets provided in ControllerCreateVolumeGroupSnapshot
  // request for the same group snapshot unless if secrets are rotated
  // after the group snapshot is created.
  // The secrets provided in the field SHOULD be passed to both
  // the group snapshot and the individual snapshot members if needed.
  map<string, string> secrets = 3 [(csi_secret) = true];
}

message DeleteVolumeGroupSnapshotResponse {
  // Intentionally empty.
}
```

#### GetVolumeGroupSnapshot

```
message GetVolumeGroupSnapshotRequest {
  option (alpha_message) = true;

  // The ID of the group snapshot to fetch current group snapshot
  // information for.
  // This field is REQUIRED.
  string group_snapshot_id = 1;

  // Secrets required by plugin to complete
  // GetVolumeGroupSnapshot request.
  // This field is OPTIONAL. Refer to the `Secrets Requirements`
  // section on how to use this field.
  // The secrets provided in this field SHOULD be the same as
  // the secrets provided in ControllerCreateVolumeGroupSnapshot
  // request for the same group snapshot unless if secrets are rotated
  // after the group snapshot is created.
  map<string, string> secrets = 2 [(csi_secret) = true];
}

message GetVolumeGroupSnapshotResponse {
  option (alpha_message) = true;

  // This field is REQUIRED
  VolumeGroupSnapshot group_snapshot = 1;
}
```

## Production Readiness Review Questionnaire

### Feature enablement and rollback

_This section must be completed when targeting alpha to a release._

* **How can this feature be enabled / disabled in a live cluster?**
  - [x] Other
    - Describe the mechanism:
      We don't have a feature gate because this feature is out of tree.
      We will use a flag called enable-volume-group-snapshot to enable this
      feature when the snapshot controller and csi-snapshotter sidecar are started.
    - Will enabling / disabling the feature require downtime of the control
      plane?
      From the controller side, it only affects the external controller sidecars.
    - Will enabling / disabling the feature require downtime or reprovisioning
      of a node?
      No.

* **Does enabling the feature change any default behavior?**
  Yes. Enabling the feature can allow a VolumeSnapshot to be created as part of the VolumeSnapshotGroup.

* **Can the feature be disabled once it has been enabled (i.e. can we rollback
  the enablement)?**
  Yes. In order to disable this feature once it has been enabled, we first need to make sure that all VolumeGroupSnapshot API objects are deleted. Then external-snapshotter controller/sidecar can be restarted without the feature flag.

If we don't delete the VolumeGroupSnapshot API objects and CRDs but just disable the feature and restart Snapshot controller and the csi-snapshotter sidecar, the API objects continue to exist in the API server. User may delete an individual VolumeSnapshot that is associated with a VolumeGroupSnapshot. After that if the user enables the feature again, the pre-existing VolumeGroupSnapshot still has the deleted individual VolumeSnapshots in its status so it is out of sync with the storage system and provides out-dated information to the user. User can still restore individual PVCs from individual VolumeSnapshots that are not deleted, but they cannot restore PVCs from the deleted VolumeSnapshots.

* **What happens if we reenable the feature if it was previously rolled back?**
  We will be able to create new VolumeGroupSnapshot API objects again.

* **Are there any tests for feature enablement/disablement?**
  Unit tests will be added with or without the feature flag enabled.

### Rollout, Upgrade and Rollback Planning

_This section must be completed when targeting beta graduation to a release._

* **How can a rollout fail? Can it impact already running workloads?**
  Try to be as paranoid as possible - e.g., what if some components will restart
   mid-rollout?

* **What specific metrics should inform a rollback?**

* **Were upgrade and rollback tested? Was the upgrade->downgrade->upgrade path tested?**

* **Is the rollout accompanied by any deprecations and/or removals of features, APIs,
fields of API types, flags, etc.?**
  Even if applying deprecation policies, they may still surprise some users.

### Monitoring Requirements

_This section must be completed when targeting beta graduation to a release._

* **How can an operator determine if the feature is in use by workloads?**
  Ideally, this should be a metric. Operations against the Kubernetes API (e.g.,
  checking if there are objects with field X set) may be a last resort. Avoid
  logs or events for this purpose.

* **What are the SLIs (Service Level Indicators) an operator can use to determine
the health of the service?**
  - [ ] Metrics
    - Metric name:
    - [Optional] Aggregation method:
    - Components exposing the metric:
  - [ ] Other (treat as last resort)
    - Details:

* **What are the reasonable SLOs (Service Level Objectives) for the above SLIs?**
  <!--
  At a high level, this usually will be in the form of "high percentile of SLI
  per day <= X". It's impossible to provide comprehensive guidance, but at the very
  high level (needs more precise definitions) those may be things like:
  - per-day percentage of API calls finishing with 5XX errors <= 1%
  - 99% percentile over day of absolute value from (job creation time minus expected
    job creation time) for cron job <= 10%
  - 99,9% of /health requests per day finish with 200 code
  -->

* **Are there any missing metrics that would be useful to have to improve observability
of this feature?**
  <!--
  Describe the metrics themselves and the reasons why they weren't added (e.g., cost,
  implementation difficulties, etc.).
  -->

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
  - [Dependency name]:
    - Usage description:
      - Impact of its outage on the feature:
      - Impact of its degraded performance or high-error rates on the feature:

### Scalability

_For alpha, this section is encouraged: reviewers should consider these questions
and attempt to answer them._

_For beta, this section is required: reviewers must answer these questions._

_For GA, this section is required: approvers should be able to confirm the
previous answers based on experience in the field._

* **Will enabling / using this feature result in any new API calls?**
  Describe them, providing:
  - API call type (e.g. PATCH pods): new APIs VolumeGroup, VolumeGroupContent, VolumeGroupClass, VolumeGroupSnapshot, VolumeGroupSnapshotContent, VolumeGroupSnapshotClass
  - estimated throughput
  - originating component(s) (e.g. Kubelet, Feature-X-controller)
  focusing mostly on:
  - components listing and/or watching resources they didn't before
  - API calls that may be triggered by changes of some Kubernetes resources
    (e.g. update of object X triggers new updates of object Y)

* **Will enabling / using this feature result in introducing new API types?**
  Describe them, providing:
  - API type:
  - Supported number of objects per cluster:
  - Supported number of objects per namespace (for namespace-scoped objects):

* **Will enabling / using this feature result in any new calls to the cloud
provider?**

* **Will enabling / using this feature result in increasing size or count of
the existing API objects?**
  Describe them, providing:
  - API type(s):
  - Estimated increase in size: (e.g., new annotation of size 32B):
  - Estimated amount of new objects: (e.g., new Object X for every existing Pod)

* **Will enabling / using this feature result in increasing time taken by any
operations covered by [existing SLIs/SLOs]?**
  Think about adding additional work or introducing new steps in between
  (e.g. need to do X to start a container), etc. Please describe the details.

* **Will enabling / using this feature result in non-negligible increase of
resource usage (CPU, RAM, disk, IO, ...) in any components?**
  Things to keep in mind include: additional in-memory state, additional
  non-trivial computations, excessive access to disks (including increased log
  volume), significant amount of data sent and/or received over network, etc.
  This through this both in small and large cases, again with respect to the
  [supported limits].

### Troubleshooting

The Troubleshooting section currently serves the `Playbook` role. We may consider
splitting it into a dedicated `Playbook` document (potentially with some monitoring
details). For now, we leave it here.

_This section must be completed when targeting beta graduation to a release._

* **How does this feature react if the API server and/or etcd is unavailable?**

* **What are other known failure modes?**
  For each of them, fill in the following information by copying the below template:
  - [Failure mode brief description]
    - Detection: How can it be detected via metrics? Stated another way:
      how can an operator troubleshoot without logging into a master or worker node?
    - Mitigations: What can be done to stop the bleeding, especially for already
      running user workloads?
    - Diagnostics: What are the useful log messages and their required logging
      levels that could help debug the issue?
      Not required until feature graduated to beta.

    - Testing: Are there any tests for failure mode? If not, describe why.

* **What steps should be taken if SLOs are not being met to determine the problem?**

[supported limits]: https://git.k8s.io/community//sig-scalability/configs-and-limits/thresholds.md
[existing SLIs/SLOs]: https://git.k8s.io/community/sig-scalability/slos/slos.md#kubernetes-slisslos

## Implementation History

N/A

## Drawbacks

Adding more new APIs and more complexities.

## Alternatives

### Immutable VolumeGroup

During the design discussions, an immutable VolumeGroup was considered but was removed because this would add lots of complexity to the design without much gain. It would also make it impossible to support the current way PVCs are provisioned in a Statefulset.

Immutable VolumeGroup - PVCList or PVC Selector in the ImmutableSource field in the Spec (optional field); PVCList is in the Status.
* Create a new VolumeGroup with existing PVCs by PVCList or PVC Selector in the Spec. The PVCList will be in the VolumeGroup Status as well.
* VolumeGroup Status has a boolean Mutable set to false.

```
ImmutableSource struct {
    PVCList
    Selector
}
```

```
// VolumeGroupSpec describes the common attributes of group storage devices
// and allows a Source for provider-specific attributes
Type VolumeGroupSpec struct {
        // +optional
        VolumeGroupClassName *string

        // If ImmutableSource is nil, an empty volume group will be created.
        // Otherwise, a volume group will be created with PVCs (if PVCList or Select is set)
        // If ImmutableSource is not nil, it indicates the VolumeGroup is immutable
        // +optional
        ImmutableSource *VolumeGroupSource
}

// VolumeGroupSource contains 3 options. If VolumeGroupSource is not nil,
// one of the 3 options must be defined.
Type VolumeGroupSource struct {
        // A list of existing persistent volume claims
        // +optional
        PVCList []PersistentVolumeClaim

        // A label query over existing persistent volume claims to be added to the volume group.
        // +optional
        Selector *metav1.LabelSelector
 }

type VolumeGroupStatus struct {
        // VolumeGroupId is a unique id returned by the CSI driver
        // to identify the VolumeGroup on the storage system.
        // If a storage system does not provide such an id, the
        // CSI driver can choose to return the VolumeGroup name.
        VolumeGroupId *string

        GroupCreationTime *metav1.Time

        // A list of persistent volume claims
        // +optional
        PVCList []PersistentVolumeClaim

        Ready *bool

        // Mutable indicates if a VolumeGroup can be modified
        // after it is created. If false, it indicates it cannot be
        // modified once created. If ImmutableSource is not nil
        // in VolumeGroupSpec, Mutable must be false; otherwise
        // it means the driver does not support ImmutableSource.
        // VOLUMEGROUP_IMMUTABLE and VOLUMEGROUP_MUTABLE capability
        // will be added to the CSI spec.
        Mutable *bool

        // If true, it indicates the CSI driver supports adding
        // an existing volume to the VolumeGroup and removing a
        // volume from the VolumeGroup without deleting it.
        // Only mutable VolumeGroup can support AddRemoveExistingPVC.
        // A corresponding VOLUMEGROUP_ADD_REMOVE_EXISTING_VOLUME
        // capability will be added to the CSI spec.
        AddRemoveExistingPVC *bool

        // Last error encountered during group creation
        Error *VolumeGroupError
}
```

VOLUMEGROUP_IMMUTABLE and VOLUMEGROUP_MUTABLE capability will be added to the CSI spec.
If VOLUMEGROUP_IMMUTABLE is supported, a VolumeGroup with an ImmutableSource can be created. Mutable will be false, PVCList will be set, and Ready will be true in the Status.
Otherwise, a VolumeGroup with an ImmutableSource will not be created successfully.

### ModifyVolume

ModifyVolume CSI RPC was considered earlier to add/remove one volume to/from a group at a time but it was removed because ModifyVolumeGroup CSI RPC was added.

A new MODIFY_VOLUME capability will be added to support this.
It indicates that the controller plugin supports modifying a volume.

```
  rpc ModifyVolume(ModifyVolumeRequest)
    returns (ModifyVolumeResponse) {
        option (alpha_method) = true;
    }
```

This RPC is called when an existing volume is added to an existing volume group or when a volume is removed from the volume group.
A volume group id parameter will be in the ModifyVolumeRequest for an add request.
A volume group id parameter will not be in the ModifyVolumeRequest for a delete request.
If user requests to add an existing volume to a consistency group, but the CSI driver cannot fulfill the request because the existing volume is placed on a different storage pool from the consistency group, then the CSI driver MUST return failure.
This RPC MUST be idempotent.

```
message ModifyVolumeRequest {
      string volume_id = 1;

      // This field is OPTIONAL.
      repeated string volume_group_id = 2 [(alpha_field) = true];

      // Secrets required by plugin to complete modify volume request.
      // This field is OPTIONAL. Refer to the `Secrets Requirements`
      // section on how to use this field.
      map<string, string> secrets = 3 [(csi_secret) = true];
}
```
External-provisioner will be modified so that modifying PVC by adding VolumeGroupName will trigger a ModifyVolume call (a new CSI controller RPC) to CSI driver.

### VolumeGroup API Definitions

In an earlier version of this KEP, a VolumeGroup API is introduced to group volumes together. The VolumeGroup is removed from the KEP for a simpler design that supports group snapshot.

For details of the VolumeGroup API design, see [here](https://docs.google.com/document/d/1VlrJGLr6YZvMrhyeQ3mJ-2Kuet9goUyzfg4Bq-NdymE/edit#).
