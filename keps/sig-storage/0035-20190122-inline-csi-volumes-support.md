---
kep-number: 35
title: Inline CSI Volumes Support
authors:
  - "@vladimirvivien"
owning-sig: sig-storage
participating-sigs:
  - sig-storage
reviewers:
  - "@thockin"
  - "@saad-ali"
  - "@msau42"
  - "@jsafrane"
  - "@liggit"
approvers:
  - TBD
creation-date: 2019-01-22
status: implementable
---

# Inline CSI volumes support

## Table of Contents

* [Table of Contents](#table-of-contents)
* [Summary](#summary)
* [Motivation](#motivation)
    * [Goals](#goals)
    * [Non-Goals](#non-goals)
* [User Stories](#user-stories)
    * [Examples](#examples)
* [Proposal](#proposal)
    * [VolumeSource API](#volumesource-api)
    * [Secret references](#secret-references)
    * [Specifying allowed inline drivers with PodSecurityPolicy](#specifying-allowed-inline-drivers-with-podSecuritypolicy)
    * [Persistent vs ephemeral lifecycle setting](#persistent-vs-ephemeral-lifecycle-setting)
    * [VolumeHandle generation](#volumehandle-generation)
    * [Inline CSI volume operation stages](#inline-csi-volume-operation-stages)
    * [Security considerations](#security-considerations)

## Summary
Currently, volumes that are backed by CSI drivers can only be used with the `PersistentVolume` and `PersistentVolumeClaim` objects. This proposal is to implement support for the ability to nest CSI volume declarations within pod specs, as is supported already by existing in-tree storage providers.

This KEP started life as [feature #2273](https://github.com/kubernetes/community/pull/2273).  Please follow that link for historical context.


## Motivation
Implementing support for embedding volumes directly in pod specs would allow external CSI drivers to have feature parity with their in-tree counter parts.  There are several reasons why this is desirable:
* It provides the community a path to move away from in-tree volume plugins to CSI, as designed in a separate proposal https://github.com/kubernetes/community/pull/2199/. 
* This feature would make it possible to create new types of CSI drivers such as ephemeral volume drivers.  They can be used to inject arbitrary states such as configuration, secrets, identity, variables or similar information inside pods. 
* This can also provide a migration path for older Flex-style drivers (allowing the deprecation of Flex.)

### Goals 
* Provide a high level design for inline CSI volumes support
* Define API changes needed to support this feature
* Inline CSI volumes should work with existing CSI drivers
* Design for ephemeral inline CSI volumes support
* Ensure that inline CSI volumes usage is secure

### Non-goals
The followings will not be addressed by this KEP:
* Introduce new CSI spec changes to support this feature
* Introduce required changes to existing CSI drivers for this feature
* Support for topology or pod placement scheme for ephemeral inline volumes
* Support for PV/PVC related features such as topology, raw block, mount options, and resizing

## User stories
* As a storage provider, I want to be able to create CSI drivers that support persistent volumes that can be nested within pod specs.  These volumes would work similarly to how my current in-tree drivers work (with minor limitations addressed later).
* As a storage provider, I want to use the CSI API to develop drivers that can mount ephemeral volumes that follow the lifecycles of pods where they are embedded.   This feature would allow me to create drivers that work similarly to how the in-tree Secrets or ConfigMaps driver works.  My ephemeral CSI driver should allow me to inject arbitrary data into a pod using a volume mount point. 
* As a user, I want to be able to deploy pods, with persistent CSI volumes embedded in their specs, without the use of PV/PVC.  
* As a user I want to be able to define pod specs with embedded ephemeral CSI volumes that are created/mounted when the pod is deployed and are deleted when the pod goes away.

### Examples

**Example 1**
A pod spec with a persistent inline CSI volume.  Note that the `volumeHandle` is required and refers to a pre-existing volume.

```yaml
apiVersion: v1
kind: Pod
metadata:
  name: testpod
spec:
  containers:
    ...
  volumes:
      - name: vol
        csi:
          driver: mock.storage.kubernetes.io
          volumeAttributes:
              name: "Mock Volume 1"
          volumeHandle: "1"
```

**Example 2**
A pod spec with an ephemeral inline CSI volume.  Note that because the volume is expected to be ephemeral, the `volumeHandle` is not provided.  Instead a CSI-generated ID will be submitted to the driver.

```yaml
apiVersion: v1
kind: Pod
metadata:
  name: some-pod
spec:
  containers:
    ...
  volumes:
      - name: vol
        csi:
          driver: inline.storage.kubernetes.io
          volumeAttributes:
              foo: bar
```

## Proposal
This proposal introduces the support of both persistent and ephemeral-style volumes nested within a pod spec definition. The following highlights the major themes of this proposal:

* Support for both persistent and ephemeral volumes
* Propose a way to clearly distinguish between persistent and ephemeral volumes.
* Persistent inline CSI volumes must be pre-provisioned and manually de-provisioned.
* Persistent inline CSI volumes will participate in storage flows including attach, stage, mount, unmount, unstage, and detach.
* Persistent inline volumes must use the handle of pre-provisioned volumes
* An ephemeral volume is created and deleted automatically following its pod's lifecycle
* Inline ephemeral volumes receive a generated volume handle

### Inline volumes
Normally, volume operations can be triggered from PV/PVC or when from a pod spec.  Currently, CSI only supports volumes that originate from PV/PVC. When a volume is embedded inside a pod spec, it will be referred to as `inline` in this document.  Volumes that originates from inline pod specs can support two modes:

* Persistent mode - the volume follows a persistent lifecycle independent of the pod where it is used.
* Ephemeral mode - the volume follows the lifecycle of the pod where it is used.

A driver may be able to support either PV/PVC-originated or pod spec originated volumes (inline).  The following is important to keep in mind:
* When a volume originates from PV/PVC, the driver will receive all volume related CSI calls (normal operation).
* When a volume originates from a pod spec (inline), the CSI driver will only receive a limited number of calls (see below).

### Inline CSI lifecycle setting
A CSI driver will have to indicate the inline volume lifecycle, that it supports, as either `persistent` or `ephemeral`.  This setting will be stored in a [`CSIDriver` configuration CRD](https://github.com/kubernetes/enhancements/issues/594) and will be checked at runtime by Kubelet and external CSI components for enforcement.  The setting is to inform the Kubelet and other CSI components how to process inline volume information for proper setup.

#### Persistent inline mode
For CSI drivers that indicate they support persistent inline volumes, the `CSIDriver.lifecycle` value should be set to `persistent` (default of if not provided).  When a driver is used to handle volumes from an inline context: 
* Volumes are assumed to be originated from an inline pod spec.
* The `volumeHandle` value must be generated ahead of time and manually entered in the pod spec.
* The driver will only receive CSI calls related to storage opearations for *attach*, *mount*, *unmount*, *detach*.
* The driver will not receive CSI calls for *porovision* and *deletion* of volumes.

#### Ephemeral inline mode
Ephemeral inline volumes are more restricted than their persistent counterparts.  The `CSIDriver.lifecycle = ephemeral` must be set to indicate support for inline ephemeral volumes which will follow the lifecycle of its pod.  For drivers indicating support for ephemral mode: 
* Volumes are assumed to be originated from an inline pod spec.
* The `volumeHandle` is not required and is ignored if provided in the pod spec.
* CSI will internally generate a `volumeHandle` value which is passed to the driver.
* Ephemeral volumes will only make CSI calls for *mount* and *unmount* volume operations.

#### Omitting inline mode
Omitting `CSIDriver.lifecycle` is an indication:
* The driver can support PV/PVC originated volume (normal operation).
* If the volume originates from a pod spec, it is assumed that its lifecycle is `persistent`.

### VolumeHandle generation
When a driver has its lifecycle set to `ephemeral` (see above), the Kubelet (internal CSI code) will employ a naming strategy to generate the value for the volumeHandle.  The generated value will be a combination of `podUID` and `pod.spec.Volume[x].name` to guarantee uniqueness.

This approach provides several advantages:
* It makes sure that each pod can use a different volume handle ID for its ephemeral volumes.  
* Each pod will get a uniquely generated volume handle, preventing accidental naming conflicts in pods.
* Each pod created by ReplicaSet, StatefulSet or DaemonSet will get the same copy of a pod template. This makes sure that each pod gets its own unique volume handle ID and thus can get its own volume instance.

Without an auto-generated naming strategy for the `volumeHandle` during an ephemeral lifecycle, a user could guess the volume handle ID of another user causing a security risk. Having a strategy that generates consistent volume handle names, will ensure that drivers obeying idempotency will always return the same volume associated with the podUID. 

### VolumeSource API

The design defines several objects needed to implement this feature:
* `VolumeSource` - Object that represents a pod's volume.  Modified to include CSI volume source.
* `CSIVolumeSource` - object representing the inline volume data


`VolumeSource` needs to be extended with CSI volume source:
```go
type VolumeSource struct {
    // <snip>
    // CSI (Container Storage Interface) represents storage that handled by an external CSI driver (Beta feature).
    // +optional
    CSI *CSIVolumeSource
}

type CSIVolumeSource struct {
	// Driver is the name of the driver to use for this volume.
	// Required.
	Driver string

	// VolumeHandle is the unique volume name returned by the CSI volume
	// plugin’s CreateVolume to refer to the volume on all subsequent calls.
	// If not provided, that handle will be auto-generated.
	//
	// +optional
	VolumeHandle *string

	// ReadOnly is the value to pass to ControllerPublishVolumeRequest.
	// Defaults to false (read/write).
	// 
	// +optional
	ReadOnly *bool

	// Filesystem type to mount. Ex. "ext4", "xfs", "ntfs".
	// If not provided, the empty value is passed to the associated CSI driver
	// which will determine the default filesystem to apply.
	//
	// +optional
	FSType *string

	// VolumeAttributes store immutable properties of the volume copied during provision.
	// These attributes are passed back to the driver during controller publish calls.
	// +optional
	VolumeAttributes map[string]string

	// ControllerPublishSecretRef is a reference to the secret object containing
	// sensitive information to pass to the CSI driver to complete the CSI
	// ControllerPublishVolume and ControllerUnpublishVolume calls.
	// This field is optional, and  may be empty if no secret is required. If the
	// secret object contains more than one secret, all secret references are passed.
	// 
	// +optional
	ControllerPublishSecretRef *LocalObjectReference

	// NodeStageSecretRef is a reference to the secret object containing sensitive
	// information to pass to the CSI driver to complete the CSI NodeStageVolume
	// and NodeStageVolume and NodeUnstageVolume calls.
	// This field is optional, and  may be empty if no secret is required. If the
	// secret object contains more than one secret, all secret references are passed.
	// 
	// +optional
	NodeStageSecretRef *LocalObjectReference

	// NodePublishSecretRef is a reference to the secret object containing
	// sensitive information to pass to the CSI driver to complete the CSI
	// NodePublishVolume and NodeUnpublishVolume calls.
	// This field is optional, and  may be empty if no secret is required. If the
	// secret object contains more than one secret, all secret references are passed.
	// 
	// +optional
	NodePublishSecretRef *LocalObjectReference
}
```

### Secret references
Secret references declared in an inline CSI volume can only be used with namespaces from pods where they are referenced .  For inline usage, secret references are stored in `LocalObjectReference` values:
* `LocalObjectReference` do not include a namespace reference.  This is to prevent reference to arbitrary namespace values.
* The namespace reference will be extracted from the the pod spec at different phases of storage lifecycle by the Kubelet or external CSI compoennts.
* The Kubelet and external CSI components must ensure secret references can only be used with the namespace from inline pod spec.


### Specifying allowed inline drivers with `PodSecurityPolicy`
To control which driver is allowed to be used within a pod spec, this design will update the `PodSecurityPolicy` to introduce `AllowedCSIDrivers` as shown below:

```go
  type PodSecurityPolicySpec struct {
	// <snip>

	// AllowedCSIDrivers is a whitelist of allowed CSI drivers used inline in a pod spec.  Empty or nil indicates that all
	// CSI drivers may be used.  This parameter is effective only when the usage of the CSI plugin
	// is allowed in the "Volumes" field.
	// +optional
	AllowedCSIDrivers []AllowedCSIDriver
  }

  // AllowedCSIDriver represents a single CSI driver that is allowed to be used.
  type AllowedCSIDriver struct {
	// Name of the CSI driver
	Name string
  }  
```

Value `PodSecurityPolicy.AllowedCSIDrivers` must be explicitly set with the names of CSI drivers that are allowed to be embedded within a pod spec.  An empty value means no CSI drivers are allowed to be specified inline inside a pod spec.

### Inline CSI volume operation stages
When a CSI driver is used in an inline context, it works slightly differently then when originated from PV/PVC.  As mentioned earlier, the inline volumes will participate in some, but not all volume operation stages with some limitations discussed here.

#### Provision/deletion
Volume provision and deletion will work differently for inline volumes. For persistent inline volumes, provision/deletion is handled as follows:
* Persistent inline volumes do not participate in the provision and deletion phases.
* Persistent inline volumes expect provision and deletion to be handled outside of the driver.

For ephemeral CSI drivers:
* Ephemeral drivers will not receive provision/deletion API calls (since these stages are driven by a PV/PVC/StorageClass).
* Ephemeral CSI drivers will have to delay or combine any provisioning/deprovisioning operation during different phase.

#### Attach/detach
CSI uses API object `storage.VolumeAttachment` to track and manage attach/detach storage operation. Currently that object contains reference to an associated PV when the driver is not used inline.  However,  it must be extended to support information from `CSIVolumeSource` (see above) when an inline volume is being attached.

```go
// VolumeAttachmentSpec is the specification of a VolumeAttachment request.
type VolumeAttachmentSpec struct {
    // <snip>

	// Source represents the volume that should be attached.
	Source VolumeAttachmentSource
}

// VolumeAttachmentSource represents a volume that should be attached, either
// PersistentVolume or a volume in-lined in a Pod.
// Exactly one member can be set.
type VolumeAttachmentSource struct {
	// Name of the persistent volume to attach.
	// +optional
	PersistentVolumeName *string

	// InlineVolumeSource represents the source location of a in-line volume in a pod to attach.
	// +optional
    InlineVolumeSource *InlineVolumeSource
}

// InlineVolumeSource represents the source location of a in-line volume in a pod.
type InlineVolumeSource struct {
	// VolumeSource is copied from the pod. It ensures that attacher has enough
	// information to detach a volume when the pod is deleted before detaching.
	// Only CSIVolumeSource can be set.
	// Required.
	CSIVolumeSource v1.VolumeSource

	// Namespace of the pod with in-line volume. It is used to resolve
	// references to Secrets in VolumeSource.
	// Required.
	Namespace string
}
```
The following steps outline how the API is used to track attachment and detachment of volumes:
* The external CSI A/D controller **copies whole `VolumeSource`**  from `Pod` into `VolumeAttachment`. This allows external CSI attacher to detach volumes for deleted pods without keeping any internal database of attached VolumeSources.
* Validation of `VolumeSource` will ensure that only `CSIVolumeSource` is being copied.  
* External CSI attacher must be extended to  process either `PersistentVolumeName` or `VolumeSource`.
* **External attacher may need permission to access secrets in any pod namespace** where inline volume is specified.
* CSI `ControllerUnpublishVolume` call (~ volume detach) will require the secrets to be available at detach time. 
* If user deletes secrets in a pod's namespace that was used to attach an inline volume, the external attacher will fail during detach (volume will remain attached), reporting errors about missing secrets to user.

For persistent inline drivers:
* Attachment will require a volumeHandle
* As stated, volumeHandle must refer to a pre-existing provisioned volume

For ephemeral inline drivers:
* Ephemeral inline drivers must ignore attachment requests
* Attachment requires a volumeHandle value which could be generated at that stage by Kubernetes
* For security reason, however, autogenerated volumeHandle must include distinctive values such as `podUID` and `pod.spec.volume[x].name`, which are not available during the Attach stage

### Mount/Unmount
This phase happens in the Kubelet and is responsible for mounting/unmounting device and/or filesystem mount points.  At this stage, volume operations have access to pod information such as podUID.  The volume information will come from `volume.Spec` which contains either `v1.CSIVolumeSource` (for volume originated from pod specs) or `v1.CSIPersistentVolume` for volume originating from PV/PVC.

In-tree CSI volume plugin calls in kubelet, get universal `volume.Spec`, which contains either `v1.VolumeSource` from Pod (for inline volumes) or `v1.PersistentVolume` (originated from PV/PVC). The code will check for presence of `CSIVolumeSource` or `CSIPersistentVolume` and read NodeStage/NodePublish secrets from appropriate source. Kubelet does not need any new permissions, it already can read secrets for pods that it handles. These secrets are needed only for `MountDevice/SetUp` calls and don't need to be cached until `TearDown`/`UnmountDevice`.

For persistent inline drivers:

* The Kubelet code will follow the natural volume operation phases supported by CSI PV/PVC volumes
* The Kubelet code will extract volume information from `v1.CSIVolumeSource` including `volumeHandle`.
* Kubelet will delegate storage operations (mount, unmount, etc) to external CSI driver calls respectively


For ephemeral inline drivers:
* During the Setup/Mount stage, `podUID` and `pod.spec.volume[x].name` will be available
* The Kubelet will create necessary mount point paths
* Kubelet will auto-generate a volumeHandle based on `podUID` and `pod.spec.volume[x].name` (see above for detail)
* Ephemeral drivers will receive mount-like calls (NodePublish) with generated paths and volumeHandle
* Ephemeral drivers are responsible for handling volume operations (create, mount, unmount, delete) during this stage

## Security considerations
As written above, external attacher may require permissions to read Secrets in any namespace. It is up to CSI driver author to document if the driver needs such permission (i.e. access to Secrets at attach/detach time) and up to cluster admin to deploy the driver with these permissions or restrict external attacher to access secrets only in some namespaces.

* Since access to in-line volumes can be configured by `PodSecurityPolicy` (see above), we expect that cluster admin gives access to CSI drivers that require secrets at detach time only to educated users that know they should not delete Secrets used in volumes.
* Number of CSI drivers that require Secrets on detach is probably very limited. No in-tree Kubernetes volume plugin requires them on detach.
* We will provide clear documentation that using in-line volumes drivers that require credentials on detach may leave orphaned attached volumes that Kubernetes is not able to detach. It's up to the cluster admin to decide if using such CSI driver is worth it.