---
title: Volume Group
authors:
  - "@xing-yang"
  - "@jingxu97"
owning-sig: sig-storage
participating-sigs:
  - sig-storage
reviewers:
  - "@msau42"
  - "@saad-ali"
  - "@thockin"
approvers:
  - "@msau42"
  - "@saad-ali"
  - "@thockin"
editor: TBD
creation-date: 2020-02-12
last-updated: 2020-02-12
status: provisional
see-also:
  - n/a
replaces:
  - n/a
superseded-by:
  - n/a
---

# Title

Volume Group

## Table of Contents

<!-- toc -->
- [Summary](#summary)
- [Motivation](#motivation)
  - [Goals](#goals)
- [Proposal for Consistency Groups and Group Snapshots](#proposal-for-consistency-groups-and-group-snapshots)
  - [Create and Modify VolumeGroup](#create-and-modify-volumegroup)
    - [Create new PVC and add to the VolumeGroup](#create-new-pvc-and-add-to-the-volumegroup)
    - [Modify VolumeGroup with existing PVCs](#modify-volumegroup-with-existing-pvcs)
    - [Create VolumeGroup from VolumeGroupSnapshot](#create-volumegroup-from-volumegroupsnapshot)
    - [Create VolumeGroup with Selector](#create-volumegroup-with-selector)
  - [Create VolumeGroupSnapshot](#create-volumegroupsnapshot)
  - [Delete VolumeGroupSnapshot](#delete-volumegroupsnapshot)
  - [Restore](#restore)
  - [API Definitions](#api-definitions)
  - [Example Yaml Files](#example-yaml-files)
    - [Volume Group Snapshot](#volume-group-snapshot)
  - [CSI Changes](#csi-changes)
    - [CSI Capabilities](#csi-capabilities)
    - [CSI Controller RPC](#csi-controller-rpc)
    - [CreateVolumeGroup](#createvolumegroup)
    - [CreateVolume](#createvolume)
    - [ModifyVolume](#modifyvolume)
    - [DeleteVolumeGroup](#deletevolumegroup)
    - [ModifyVolumeGroup](#modifyvolumegroup)
    - [ControllerGetVolumeGroup](#controllergetvolumegroup)
    - [ListVolumeGroups](#listvolumegroups)
    - [CreateVolumeGroupSnapshot](#createvolumegroupsnapshot)
    - [CreateSnapshot](#createsnapshot)
    - [DeleteVolumeGroupSnapshot](#deletevolumegroupsnapshot)
    - [ControllerGetVolumeGroupSnapshot](#controllergetvolumegroupsnapshot)
    - [ListVolumeGroupSnapshots](#listvolumegroupsnapshots)
  - [Alternatives](#alternatives)
- [Proposal for Volume Placement](#proposal-for-volume-placement)
  - [API Changes](#api-changes)
  - [Example Yaml Files for Volume Placement](#example-yaml-files-for-volume-placement)
<!-- /toc -->

## Summary

This proposal is to introduce a VolumeGroup API to manage multiple volumes together and a VolumeGroupSnapshot API to take a snapshot of a VolumeGroup. It also attempts to address other use cases such as volume placement.

## Motivation

While there is already a KEP (https://github.com/kubernetes/enhancements/pull/1051) that introduces APIs to do application snapshot, backup, and restore, there are other use cases not covered by that KEP.

Use case 1:
A VolumeGroup allows users to manage multiple volumes belonging to the same application together and therefore it is very useful in general. For example, it can be used to group all volumes in the same StatefulSet together.

Use case 2:
For some storage systems, volumes are always managed in a group. For these storage systems, they will have to create a group for a single volume if they need to implement a create volume function in Kubernetes. Providing a VolumeGroup API will be very convenient for them.

Use case 3:
Instead of taking individual snapshots one after another, VolumeGroup can be used as a source for taking a snapshot of all the volumes in the same volume group. This may be a storage level consistent group snapshot if the storage system supports it. In any case, when used together with quiesce hooks, this group snapshot can be application consistent. For this use case, we will introduce another CRD VolumeGroupSnapshot.

Use case 4:
VolumeGroup can be used to manage group replication or consistency group replication if the storage system supports it. Note replication is out of scope for this proposal. It is mentioned here as a potential future use case.

Use case 5:
VolumeGroup can be used to manage volume placement to either spread the volumes across storage pools or stack the volumes on the same storage pool. Related KEPs proposing the concept of storage pool for volume placement is as follows:
  https://github.com/kubernetes/enhancements/pull/1353
  https://github.com/kubernetes/enhancements/pull/1347
We may not really need a VolumeGroup for this use case. A StoragePool is probably enough. This is to be determined.

Use case 6:
VolumeGroup can also be used together with application snapshot. It can be a resource managed by the ApplicationSnapshot CRD.

Use case 7: 
Some applications may not want to use ApplicationSnapshot CRD because they don’t use Kubernetes workload APIs such as StatefulSet, Deployment, etc. Instead, they have developed their own operators. In this case it is more convenient to use VolumeGroup to manage persistent volumes used in those applications.

Use case 8:
Application quiesce is time consuming. Some users may not want to do application quiesce very frequently for that reason. For example, a user may want to run weekly backups with application quiesce and nightly backups without application quiesce but with consistency group support which provides crash consistency across all volumes in the group.

### Goals

* Provide an API to manage multiple volumes together in a group.
* Provide an API to support consistency groups for snapshots, ensuring crash consistency across all volumes in the group.
* Provide an API to take a snapshot of a group of volumes, not ensuring crash consistency.
* Provide a design to facilitate volume placement using the group API (To be determined).
* The group API should be generic and extensible so that it may be used to support other features in the future.
* A VolumeGroup may potentially be used to support consistency group replication or group replication in the future, but providing design on replication group is not in the scope of this KEP. This can be discussed in the future.

## Proposal for Consistency Groups and Group Snapshots

This proposal introduces new CRDs VolumeGroup, VolumeGroupClass, and VolumeGroupSnapshot.

Create new VolumeGroup can be done in several ways:
1. Create an empty group first, then create a new PVC with the group name which will add a volume to the already created group.
2. Create an empty group first, and then add an existing PVC to the group one by one.
3. Create a new volume group from an existing group snapshot.
4. Create a new volume group and add existing PVCs matching the label selector to the group.
5. Non-goal: Create a new empty group and in the same time create new empty PVCs and add to the new group.

Modify an existing VolumeGroup:
Add new volume or remove existing volume from an existing VolumeGroup. Option 2 for create VolumeGroup above falls into this case.

### Create and Modify VolumeGroup

VolumeGroups can be created and/or modified in several ways as described in the following.

#### Create new PVC and add to the VolumeGroup

* Create a new empty VolumeGroup.
* Create a new PVC with existing VolumeGroup name. As a result, new PVC is created and added to VolumeGroup. VolumeGroup is modified so Status has this new PVC in PVCList.
* External-provisioner will be modified so that VolumeGroupName will be passed to the CSI driver when creating a volume.

Only CSI drivers supporting VOLUMEGROUP capability can support the volume group this way.
When a new PVC is created with the existing VolumeGroup name, the VolumeGroup will be modified and the PVC will be added to PVCList in the Status.

#### Modify VolumeGroup with existing PVCs

We can add an existing PVC to the group or remove a PVC from the group without deleting it. A VOLUMEGROUP_ADD_REMOVE_EXISTING_VOLUME capability will be added to CSI Spec. Only CSI drivers supporting both VOLUMEGROUP and VOLUMEGROUP_ADD_REMOVE_EXISTING_VOLUME capabilities can support the volume group this way.
* Create a new empty VolumeGroup.
* Add an existing PVC to an existing VolumeGroup (VolumeGroup can be empty to start with or it can have other PVCs already) by adding VolumeGroup name to the PVC Spec.
* VolumeGroup is modified so the existing PVC is added to the PVCList in the Status.
  * Note: The VolumeGroup controller will be implemented to have a desired state
    of the world and an actual state of the world. The desired state of the world
    contains VolumeGroups with the desired PVCList while the actual state of the
    world contains VolumeGroups with the actual PVCList. The controller will try
    to reconcile the two by handling adding and removing multiple PVCs through a
    single CSI ModifyVolumeGroup RPC call each time.
* External-provisioner will be modified so that modifying PVC by adding VolumeGroupName will trigger a ModifyVolume call (a new CSI controller RPC) to CSI driver.
* VolumeGroup controller will be triggered to update the VolumeGroup Status.
* Deleting a PVC from a VolumeGroup will trigger external-provisioner and the VolumeGroup controller as well.

#### Create VolumeGroup from VolumeGroupSnapshot

Creating a new volume group from an existing group snapshot is supported if the CSI driver supports VOLUMEGROUP capability. As a result, PVCs will be created from source snapshots and placed in a new volume group.

#### Create VolumeGroup with Selector

Creating a new volume group and adding existing PVCs matching the label selector to the group is supported if the CSI driver supports VOLUMEGROUP capability.

CSI drivers that do not have a volume_group_id concept can use the VolumeGroup name stored in Kubernetes API server as the volume_group_id.

### Create VolumeGroupSnapshot

A VolumeGroupSnapshot can be created with a VolumeGroup as the source if the CSI driver supports the GROUPSNAPSHOT capability.
* Create a VolumeGroupSnapshot with a VolumeGroup as the source.
* This will trigger the VolumeGroupSnapshot controller to call the CreateVolumeGroupSnapshot CSI function and also create multiple VolumeSnapshot API objects with VolumeGroupSnapshot name parameter in each VolumeSnapshot Spec. This will trigger the creation of VolumeSnapshotContent API objects in the snapshot controller and calls to the CreateSnapshot CSI function in the CSI snapshotter sidecar. The CSI snapshotter sidecar will pass the new group_snapshot_name parameter to the CSI Driver when calling CreatSnapshot.
* When CSI driver receives CreateSnapshot request for individual snapshots with a VolumeGroupSnapshot name:
  * Case 1: If it knows how to create a group snapshot on the storage system, it returns (nil, nil), and leave it to the CreateVolumeGroupSnapshot function to handle the snapshot creation.
  * Case 2: If it does not know how to create a group snapshot on the storage system, it will create an individual snapshot as usual and return the snapshot_id back.
* CreateVolumeGroupSnapshot CSI function response
  * Case 1: The CreateVolumeGroupSnapshot CSI function should return a list of snapshots (Snapshot message defined in CSI Spec) in its response. The VolumeGroupSnapshot controller can use the returned list of snapshots to update corresponding individual VolumeSnapshotContents, wait for VolumeSnapshots and VolumeSnapshotContents to be bound, and update SnapshotList in the VolumeGroupSnapshot Status.
  * Case 2: The CreateVolumeGroupSnapshot CSI function returns group_snapshot_id and volume_group_id, but leaves snapshots field as empty. The VolumeGroupSnapshot controller watches VolumeSnapshot and VolumeSnapshotContent API objects. When VolumeSnapshot and VolumeSnapshotContent are bound, it saves the VolumeSnapshot API object to SnapshotList in its Status.

### Delete VolumeGroupSnapshot

A VolumeGroupSnapshot can be deleted if the CSI driver supports the GROUPSNAPSHOT capability.
* When a VolumeGroupSnapshot is deleted, the VolumeGroupSnapshot controller will call the DeleteVolumeGroupSnapshot CSI function as well as DeleteSnapshot CSI functions. Just like create snapshot, there are 2 cases.
  * Case 1: Since CSI driver handles individual snapshot creation in CreateVolumeGroupSnapshot, it should handle individual snapshot deletion in DeleteVolumeGroupSnapshot.
  * Case 2: Since CSI driver handles individual snapshot creation in CreateSnapshot, it should handle individual snapshot deletion in DeleteSnapshot.
* DeleteSnapshot on a single snapshot that belongs to a group snapshot is not allowed.

### Restore

Restore can be done as follows:
1. A new empty volume group can be created first, and then a new volume can be created from a snapshot one by one and added to the volume group. This can be repeated for all the snapshots in the VolumeGroupSnapshot.
2. A VolumeGroup can be created from a VolumeGroupSnapshot source in one step. This is the same as what is described in the section `Create VolumeGroup from VolumeGroupSnapshot`.

### API Definitions

API definitions are as follows:

```
type VolumeGroupClass struct {
        metav1.TypeMeta
        // +optional
        metav1.ObjectMeta
 
        // Driver is the driver expected to handle this VolumeGroupClass.
        // This value may not be empty.
        Driver string
 
        // Parameters holds parameters for driver.
        // These values are opaque to the system and are passed directly
        // to the driver.
        // +optional
        Parameters map[string]string

        // This field specifies whether group snapshot is supported.
        // The default is false.
        // +optional
        VolumeGroupSnapshot *bool

        // Specifies whether consistent group snapshot is supported.
	// The default is false.
        // +optional
        ConsistentGroupSnapshot *bool
}

// VolumeGroup is a user's request for a group of volumes
type VolumeGroup struct {
        metav1.TypeMeta
        // +optional
        metav1.ObjectMeta

        // Spec defines the volume group requested by a user
        Spec VolumeGroupSpec

        // Status represents the current information about a volume group
        // +optional
        Status *VolumeGroupStatus
}

// VolumeGroupSpec describes the common attributes of group storage devices
// and allows a Source for provider-specific attributes
Type VolumeGroupSpec struct {
        // +optional
        VolumeGroupClassName *string

        // If InitSource is nil, an empty volume group will be created.
        // Otherwise, a volume group will be created with PVCs.
	// If Selector is set in InitSource, existing PVCs with matching
	// label will be added to the volume group.
	// If SourceVolumeGroupSnapshotName is not nil, the volume group
	// will be created from the source VolumeGroupSnapshot.
	// This field determines what PVCs will be in the volume group
	// when it is initially created. PVCs can be added to or removed
	// from the volume group later if CSI driver supports
	// VOLUMEGROUP_ADD_REMOVE_EXISTING_VOLUME.
        // +optional
        InitSource *VolumeGroupSource
}

// VolumeGroupSource contains 2 options. If VolumeGroupSource is not nil,
// one and only one of the 2 options must be defined.
Type VolumeGroupSource struct {
        // A label query over existing persistent volume claims to be added to the volume group.
        // +optional
        Selector *metav1.LabelSelector

        // If specified, the VolumeGroup will be created from the source
        // VolumeGroupSnapshot.
        // +optional
        SourceVolumeGroupSnapshotName *string
}


type VolumeGroupStatus struct {
	// VolumeGroupId is a unique id returned by the CSI driver
	// to identify the VolumeGroup on the storage system.
	// If a storage system does not provide such an id, the
	// CSI driver can choose to return the VolumeGroup name.
	// +optional
	VolumeGroupId *string

	// +optional
        GroupCreationTime *metav1.Time

        // A list of persistent volume claims
        // +optional
        PVCList []PersistentVolumeClaim

	// +optional
        Ready *bool

        // Last error encountered during group creation
	// +optional
        Error *VolumeGroupError
}

// Describes an error encountered on the group
type VolumeGroupError struct {
    	// time is the timestamp when the error was encountered.
    	// +optional
    	Time *metav1.Time
 
    	// message details the encountered error
    	// +optional
    	Message *string
}

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
        // Source has the information about where the group snapshot is created from.
        // Supported Kind is VolumeGroup
	// Required.
        Source TypedLocalObjectReference `json:"source" protobuf:"bytes,1,opt,name=source"`
}

Type VolumeGroupSnapshotStatus struct {
        // VolumeGroupSnapshotId is a unique id returned by the CSI driver
        // to identify the VolumeGroupSnapshot on the storage system.
        // If a storage system does not provide such an id, the
        // CSI driver can choose to return the VolumeGroupSnapshot name.
	// +optional
	VolumeGroupSnapshotID *string

        // ReadyToUse becomes true when ReadyToUse on all individual snapshots become true
        // +optional
        ReadyToUse *bool

        // List of volume snapshots
	// +optional
        SnapshotList []VolumeSnapshot
}

type PersistentVolumeClaimSpec struct {
	......
	// +optional
        VolumeGroupNames []string
	......
}


type VolumeSnapshotSpec struct{
	......
	// +optional
        VolumeGroupSnapshotName *string
	......
}
```

### Example Yaml Files

#### Volume Group Snapshot

Example yaml files to define a VolumeGroupClass and VolumeGroup are in the following.

A VolumeGroupClass that supports groupSnapshot:
```
apiVersion: volumegroup.storage.k8s.io/v1alpha1
kind: VolumeGroupClass
metadata:
  name: volumeGroupClass1
spec:
  parameters:
     …...
  groupSnapshot: true
```

A VolumeGroup belongs to this VolumeGroupClass:
```
apiVersion: volumegroup.storage.k8s.io/v1alpha1
kind: VolumeGroup
metadata:
  Name: volumeGroup1
spec:
  volumeGroupClassName: volumeGroupClass1
```

A VolumeGroupSnapshot taken from the VolumeGroup:
```
apiVersion: volumegroup.storage.k8s.io/v1alpha1
kind: VolumeGroupSnapshot
metadata:
  name: my-group-snapshot
spec:
  source:
    name: volumeGroup1
    kind: VolumeGroup
    apiGroup: volumegroup.storage.k8s.io
```

A PVC that belongs to the volume group which supports groupSnapshot:
```
apiVersion: v1
kind: PersistentVolumeClaim
metadata:
  name: pvc1
  annotations:
spec:
  accessModes:
  - ReadWriteOnce
  dataSource: null
  resources:
	requests:
  	storage: 1Gi
  storageClassName: storageClass1
  volumeMode: Filesystem
  volumeGroupNames: [volumeGroup1]
```

A new external VolumeGroup controller will handle VolumeGroupClass and VolumeGroup resources.
External provisioner will be modified to read information from volume groups (through volumeGroupNames) and pass them down to the CSI driver.

### CSI Changes

#### CSI Capabilities

New controller capabilities VOLUMEGROUP, VOLUMEGROUP_ADD_REMOVE_EXISTING_VOLUME, GROUPSNAPSHOT, MODIFY_VOLUME, and INDIVIDUAL_SNAPSHOT_RESTORE will be added.

* VOLUMEGROUP
  Indicates that the controller plugin supports creating and deleting a volume group.

* VOLUMEGROUP_ADD_REMOVE_EXISTING_VOLUME
  Indicates that the controller plugin supports adding an existing volume to a
  volume group and removing a volume from a volume group without deleting it.

* GROUPSNAPSHOT
  Indicates that the controller plugin supports creating a snapshot of all volumes
  in a volume group.

* CONSISTENT_GROUPSNAPSHOT
  Indicates that the controller plugin supports creating a consistent snapshot of
  all volumes in a volume group.

* MODIFY_VOLUME
  Indicates that the controller plugin supports modifying a volume.

* INDIVIDUAL_SNAPSHOT_RESTORE
  Indicates whether the controller plugin supports creating a volume from an
  individual volume snapshot if the volume snapshot is part of a
  VolumeGroupSnapshot. Use cases: selective restore, advanced recovery, etc.

#### CSI Controller RPC

```
service Controller {
  …
  rpc CreateVolumeGroup(CreateVolumeGroupRequest)
    returns (CreateVolumeGroupResponse) {
        option (alpha_method) = true;
    }

  rpc CreateVolumeGroupSnapshot(CreateVolumeGroupSnapshotRequest)
    returns (CreateVolumeGroupSnapshotResponse) {
        option (alpha_method) = true;
    }

  rpc ModifyVolumeGroup(ModifyVolumeGroupRequest)
    returns (ModifyVolumeGroupResponse) {
        option (alpha_method) = true;
    }

  rpc DeleteVolumeGroup(DeleteVolumeGroupRequest)
    returns (DeleteVolumeGroupResponse) }
        option (alpha_method) = true;
    }

  rpc DeleteVolumeGroupSnapshot(DeleteVolumeGroupSnapshotRequest)
    returns (DeleteVolumeGroupSnapshotResponse) {
        option (alpha_method) = true;
    }

  rpc ListVolumeGroups(ListVolumeGroupsRequest)
    returns (ListVolumeGroupsResponse) {
        option (alpha_method) = true;
    }

  rpc ListVolumeGroupSnapshots(ListVolumeGroupSnapshotsRequest)
    returns (ListVolumeGroupSnapshotsResponse) {
        option (alpha_method) = true;
    }

  rpc GetVolumeGroup(GetVolumeGroupRequest)
    returns (GetVolumeGroupResponse) {
        option (alpha_method) = true;
    }

  rpc GetVolumeGroupSnapshot(GetVolumeGroupSnapshotRequest)
    returns (GetVolumeGroupSnapshotResponse) {
        option (alpha_method) = true;
    }

  rpc ModifyVolume(ModifyVolumeRequest)
    returns (ModifyVolumeResponse) {
        option (alpha_method) = true;
    }
  …
}
```

#### CreateVolumeGroup

This RPC will be called by the CO to create a new volume group on behalf of a user.
This operation MUST be idempotent. If a volume corresponding to the specified volume name already exists, is compatible with the specified parameters in the CreateVolumeGroupRequest, the Plugin MUST reply 0 OK with the corresponding CreateVolumeGroupResponse.
CSI Plugins MAY create the following types of volume groups:

* Create a new empty volume group.
* At restore time, create a single volume from individual snapshot and then join an existing group.
 * Create an empty group
 * Create a volume from snapshot in the group
* Create a new volume group from a source group snapshot.
* Create a new volume group and add a list of existing volumes to the group.

The following is non-goal:
* Non goal: Create a new group and at the same time create a list of new volumes in the group.

```
message CreateVolumeGroupRequest {
  option (alpha_message) = true;

  // suggested name for volume group (required for idempotency)
  // This field is REQUIRED.
  string name = 1;

  // params passed from VolumeGroupClass
  // This field is OPTIONAL.
  map<string, string> parameters = 2;

  // Secrets required by plugin to complete volume group creation request.
  // This field is OPTIONAL. Refer to the `Secrets Requirements`
  // section on how to use this field.
  map<string, string> secrets = 3 [(csi_secret) = true];

  // If specified, a volume group will be created from the source group snapshot.
  // This field is OPTIONAL.
  VolumeGroupSnapshot source_volume_group_snapshot = 4;

  // If specified, a volume group will be created from a list of existing volumes.
  // This field is OPTIONAL.
  repeated string volume_id = 5;
}

message CreateVolumeGroupResponse {
  option (alpha_message) = true;

  // Contains all attributes of the newly created volume group.
  // This field is REQUIRED.
  VolumeGroup volume_group = 1;
}

message VolumeGroup {
  option (alpha_message) = true;

  // The identifier for this volume group, generated by the plugin.
  // This field is REQUIRED.
  string volume_group_id = 1;

  // Opaque static properties of the volume group.
  // This field is OPTIONAL.
  map<string, string> volume_group_context = 2;

  // Underlying volumes in this group. The same definition in CSI Volume.
  // This field is OPTIONAL.
  repeated .csi.v1.Volume volumes = 3;
}
```

#### CreateVolume

1. When a new volume is created with a volume group id parameter, the volume will be created and added to the existing volume group.
2. A new volume can also be created without a volume group id parameter. It can be added to a volume group later through the ModifyVolumeGroup RPC.

Note that for filesystems based storage systems, only option 1 can be supported. For block based storage systems. Both option 1 and 2 may be supported. However there is a possibility that option 2 will not work for ConsistencyGroups as the volume is created without the consideration of which group the volume will be placed in.

```
message CreateVolumeRequest {
      string name = 1;
      …
      repeated string volume_group_id = 8 [(alpha_field) = true];
}
```

#### ModifyVolume

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

#### DeleteVolumeGroup

```
message DeleteVolumeGroupRequest {
  option (alpha_message) = true;

  // The ID of the volume group to be deprovisioned.
  // This field is REQUIRED.
  string volume_group_id = 1;

  // Secrets required by plugin to complete volume group deletion request.
  // This field is OPTIONAL. Refer to the `Secrets Requirements`
  // section on how to use this field.
  map<string, string> secrets = 2 [(csi_secret) = true];
}

message DeleteVolumeGroupResponse {
  option (alpha_message) = true;
  // Intentionally empty.
}
```

#### ModifyVolumeGroup

This RPC will be called by the CO to modify an existing volumegroup on behalf of a user. volume_ids provided in the ModifyVolumeGroupRequest will be compared to the ones in the existing VolumeGroup. New volume_ids in the modified VolumeGroup will be added to the VolumeGroup. Existing volume_ids not in the modified VolumeGroup will be removed from the VolumeGroup. If volume_ids is empty, the VolumeGroup will be removed of all existing volumes. This operation MUST be idempotent.

To support ModifyVolumeGroup, the Kubernetes VolumeGroup controller will be implemented to have a desired state of the world and an actual state of the world. The desired state of the world contains VolumeGroups with the desired PVCList while the actual state of the world contains VolumeGroups with the actual PVCList. The controller will try to reconcile the two by handling adding and removing multiple PVCs through a single CSI RPC call each time.

Note that filesystems based storage systems may not be able to support this RPC. For block based storage systems, this is a very convenient method. However, it may not satisfy the requirement for consistency as the volume is created without the knowledge of which group it is placed in.

```
message ModifyVolumeGroupRequest {
  option (alpha_message) = true;

  // The ID of the volume group to be modified.
  // This field is REQUIRED.
  string volume_group_id = 1;

  // Specify volume_ids that will be in the modified volume group.
  // This list will be compared with the volume_ids in the existing group.
  // New ones will be added and missing ones will be removed.
  // If no volume_ids are provided, all existing volumes will
  // be removed from the group.
  // This field is OPTIONAL.
  repeated string volume_ids = 3;
}

message ModifyVolumeGroupResponse {
  option (alpha_message) = true;

  // Contains all attributes of the modified volume group.
  // This field is REQUIRED.
  VolumeGroup volume_group = 1;
}
```

#### ControllerGetVolumeGroup

```
message ControllerGetVolumeGroupRequest {
  option (alpha_message) = true;

  // The ID of the volume group to fetch current volume group information for.
  // This field is REQUIRED.
  string volume_group_id = 1;
}

message ControllerGetVolumeGroupResponse {
  option (alpha_message) = true;

  // This field is REQUIRED
  VolumeGroup volume_group = 1;
}
```

#### ListVolumeGroups

```
message ListVolumeGroupsRequest {
  option (alpha_message) = true;

  // If specified (non-zero value), the Plugin MUST NOT return more
  // entries than this number in the response. If the actual number of
  // entries is more than this number, the Plugin MUST set `next_token`
  // in the response which can be used to get the next page of entries
  // in the subsequent `ListVolumeGroups` call. This field is OPTIONAL. If
  // not specified (zero value), it means there is no restriction on the
  // number of entries that can be returned.
  // The value of this field MUST NOT be negative.
  int32 max_entries = 1;

  // A token to specify where to start paginating. Set this field to
  // `next_token` returned by a previous `ListVolumeGroups` call to get the
  // next page of entries. This field is OPTIONAL.
  // An empty string is equal to an unspecified field value.
  string starting_token = 2;
}

message ListVolumeGroupsResponse {
  option (alpha_message) = true;

  message Entry {
    // This field is REQUIRED
    VolumeGroup volume_group = 1;
  }

  repeated Entry entries = 1;

  // This token allows you to get the next page of entries for
  // `ListVolumeGroups` request. If the number of entries is larger than
  // `max_entries`, use the `next_token` as a value for the
  // `starting_token` field in the next `ListVolumeGroups` request. This
  // field is OPTIONAL.
  // An empty string is equal to an unspecified field value.
  string next_token = 2;
}
```

#### CreateVolumeGroupSnapshot

The purpose of this call is to request the creation of a multi-volume snapshot. Group snapshots can be created from existing volume group. Note that calls to this function must be idempotent - the function may be called multiple times for the same name - the group snapshot must only be created once.

```
message CreateVolumeGroupSnapshotRequest {
  option (alpha_message) = true;

  // suggested name for a group snapshot (required for idempotent)
  // This field is REQUIRED.
  string name = 1;

  // identifier indicates which volume group is used to take
  // group snapshot
  // This field is REQUIRED.
  string source_volume_group_id = 2;

  // volume ids of the volumes in the source group. This field is REQUIRED.
  // This is needed because some storage systems does not have a group persisted
  // on the storage system until the time to take a group snapshot
  repeated string volume_ids = 3;

  // secrets required for snapshot creation (pulled from VolumeSnapshotClass)
  // This field is OPTIONAL.
  map<string, string> secrets = 4 [(.csi.v1.csi_secret) = true];

  // params passed from VolumeSnapshotClass
  // This field is OPTIONAL.
  map<string, string> parameters = 5;
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
  // This field is REQUIRED.
  string group_snapshot_id = 1;

  // A list of snapshots created. Snapshot is the same
  // definition as Snapshot definition used in CSI.
  // This field is REQUIRED.
  repeated .csi.v1.Snapshot snapshots = 2;

  // Identity information for the source volume group. Currently, only
  // support the case that source is volume group. This field is REQUIRED.
  string source_volume_group_id = 3;

  // Indicates if a list of group snapshots are ready.
  // This field is REQUIRED.
  bool ready_to_use = 4;

  // Timestamp when the point-in-time consistency group snapshot is taken.
  // This field is REQUIRED.
  .google.protobuf.Timestamp creation_time = 5;

  // Complete total size of the snapshots in group in bytes. The purpose of
  // this field is to give CO guidance on how much space is needed to restore
  // volumes from all snapshots in group. This field is OPTIONAL.
  int64 size_bytes = 6;
}
```

#### CreateSnapshot

```
message CreateSnapshotRequest {
  // The ID of the source volume to be snapshotted.
  // This field is REQUIRED.
  string source_volume_id = 1;
  …
  string group_snapshot_name = 2 [(alpha_field) = true];
}

message CreateSnapshotResponse {
  Snapshot snapshot = 1;
  …
  string group_snapshot_id = 2 [(alpha_field) = true];
}
```

#### DeleteVolumeGroupSnapshot

```
message DeleteVolumeGroupSnapshotRequest {
  option (alpha_message) = true;

  // The ID of the group snapshot to be deprovisioned.
  // This field is REQUIRED.
  string group_snapshot_id = 1;

  // Secrets required by plugin to complete group snapshot deletion request.
  // This field is OPTIONAL. Refer to the `Secrets Requirements`
  // section on how to use this field.
  map<string, string> secrets = 2 [(csi_secret) = true];
}

message DeleteVolumeGroupSnapshotResponse {
  // Intentionally empty.
}
```

#### ControllerGetVolumeGroupSnapshot

```
message ControllerGetVolumeGroupSnapshotRequest {
  option (alpha_message) = true;

  // The ID of the group snapshot to fetch current group snapshot information for.
  // This field is REQUIRED.
  string group_snapshot_id = 1;
}

message ControllerGetVolumeGroupSnapshotResponse {
  option (alpha_message) = true;

  // This field is REQUIRED
  VolumeGroupSnapshot group_snapshot = 1;
}
```

#### ListVolumeGroupSnapshots

```
message ListVolumeGroupSnapshotsRequest {
  option (alpha_message) = true;

  // If specified (non-zero value), the Plugin MUST NOT return more
  // entries than this number in the response. If the actual number of
  // entries is more than this number, the Plugin MUST set `next_token`
  // in the response which can be used to get the next page of entries
  // in the subsequent `ListVolumeGroupSnapshots` call. This field is OPTIONAL. If
  // not specified (zero value), it means there is no restriction on the
  // number of entries that can be returned.
  // The value of this field MUST NOT be negative.
  int32 max_entries = 1;

  // A token to specify where to start paginating. Set this field to
  // `next_token` returned by a previous `ListVolumeGroupSnapshots` call to get the
  // next page of entries. This field is OPTIONAL.
  // An empty string is equal to an unspecified field value.
  string starting_token = 2;
}

message ListVolumeGroupSnapshotsResponse {
  option (alpha_message) = true;

  message Entry {
    // This field is REQUIRED
    VolumeGroupSnapshot group_snapshot = 1;
  }

  repeated Entry entries = 1;

  // This token allows you to get the next page of entries for
  // `ListVolumeGroupSnapshots` request. If the number of entries is larger than
  // `max_entries`, use the `next_token` as a value for the
  // `starting_token` field in the next `ListVolumeGroupSnapshots` request. This
  // field is OPTIONAL.
  // An empty string is equal to an unspecified field value.
  string next_token = 2;
}
```

### Alternatives

During the design discussions, an immutable VolumeGroup was considered but was removed because this would add lots of complexity to the design without much gain.

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

## Proposal for Volume Placement

### API Changes

In order to support Volume Placement, An `AllowedTopologies` field will be added to the VolumeGroupClass API:

```
type VolumeGroupClass struct {
        metav1.TypeMeta
        // +optional
        metav1.ObjectMeta

        // Driver is the driver expected to handle this VolumeGroupClass.
        // This value may not be empty.
        Driver string

        // Parameters holds parameters for driver.
        // These values are opaque to the system and are passed directly
        // to the driver.
        // +optional
        Parameters map[string]string

        // This field specifies whether group snapshot is supported.
        // The default is false.
        // +optional
        VolumeGroupSnapshot *bool

        // Restrict the topologies where a group of volumes can be located.
        // Each driver defines its own supported topology specifications.
        // An empty TopologySelectorTerm list means there is no topology restriction.
        // This field is passed on to the drivers to handle placement of a group of
        // volumes on storage pools.
        // +optional
        AllowedTopologies []api.TopologySelectorTerm
}
```

### Example Yaml Files for Volume Placement

A VolumeGroupClass that supports placement:
```
apiVersion: volumegroup.storage.k8s.io/v1alpha1
kind: VolumeGroupClass
metadata:
  name: placementGroupClass1
spec:
  parameters:
     …...
  allowedTopologies: [failure-domain.example.com/placement: storagePool1]
```
```
apiVersion: volumegroup.storage.k8s.io/v1alpha1
kind: VolumeGroup
metadata:
  Name: placemenGroup1
spec:
  volumeGroupClassName: placementGroupClass1
```

A PVC that belongs to both the volume group with groupSnapshot support and placement.
```
apiVersion: v1
kind: PersistentVolumeClaim
metadata:
  name: pvc1
  annotations:
spec:
  accessModes:
  - ReadWriteOnce
  dataSource: null
  resources:
        requests:
        storage: 1Gi
  storageClassName: storageClass1
  volumeMode: Filesystem
  volumeGroupNames: [volumeGroup1, placementGroup1]
```

If both placement group and volume group with groupSnapshot support are defined, it is possible for the same volume to join both groups. For example, a volume group with groupSnapshot support may include volume members from two placement groups as they belong to the same application.
