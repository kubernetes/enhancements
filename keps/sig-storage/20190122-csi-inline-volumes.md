---
title: Ephemeral Inline CSI Volumes
authors:
  - "@vladimirvivien"
  - "@pohly"
owning-sig: sig-storage
participating-sigs:
  - sig-storage
reviewers:
  - "@msau42"
  - "@jsafrane"
  - "@liggitt"
approvers:
  - "@thockin"
  - "@saad-ali"
creation-date: 2019-01-22
last-updated: 2019-08-30
status: implementable
---

# Ephemeral Inline CSI volumes

## Table of Contents

<!-- toc -->
- [Summary](#summary)
- [Motivation](#motivation)
  - [Goals](#goals)
  - [Non-goals](#non-goals)
- [User stories](#user-stories)
  - [Examples](#examples)
- [Ephemeral inline volume proposal](#ephemeral-inline-volume-proposal)
  - [VolumeHandle generation](#volumehandle-generation)
  - [API updates](#api-updates)
  - [Support for inline CSI volumes](#support-for-inline-csi-volumes)
  - [Secret reference](#secret-reference)
  - [Specifying allowed inline drivers with <code>PodSecurityPolicy</code>](#specifying-allowed-inline-drivers-with-)
  - [Ephemeral inline volume operations](#ephemeral-inline-volume-operations)
- [Test plans](#test-plans)
  - [All unit tests](#all-unit-tests)
  - [Ephemeral inline volumes unit tests](#ephemeral-inline-volumes-unit-tests)
  - [E2E tests](#e2e-tests)
- [Alternatives](#alternatives)
- [Implementation History](#implementation-history)
<!-- /toc -->

## Summary
Currently, volumes that are backed by CSI drivers can only be used with the `PersistentVolume` and `PersistentVolumeClaim` objects. This proposal is to implement support for the ability to nest CSI volume declarations within pod specs for ephemeral-style drivers.

This KEP started life as [feature #2273](https://github.com/kubernetes/community/pull/2273).  Please follow that link for historical context.


## Motivation
Implementing support for embedding volumes directly in pod specs would allow driver developers to create new types of CSI drivers such as ephemeral volume drivers.  They can be used to inject arbitrary states, such as configuration, secrets, identity, variables or similar information, directly inside pods using a mounted volume. 


### Goals 
* Provide a high level design for ephemeral inline CSI volumes support
* Define API changes needed to support this feature
* Outlines how ephemeral inline CSI volumes would work 
* Ensure that inline CSI volumes usage is secure

### Non-goals
The followings will not be addressed by this KEP:
* Introduce new CSI spec changes to support this feature
* Introduce required changes to existing CSI drivers for this feature
* Support for topology or pod placement scheme for ephemeral inline volumes
* Support for PV/PVC related features such as topology, raw block, mount options, and resizing
* Support for inline pod specs backed by a persistent volumes

## User stories
* As a storage provider, I want to use the CSI API to develop drivers that can mount ephemeral volumes that follow the lifecycles of pods where they are embedded.   This feature would allow me to create drivers that work similarly to how the in-tree Secrets or ConfigMaps driver works.  My ephemeral CSI driver should allow me to inject arbitrary data into a pod using a volume mount point inside the pod. 
* As a user I want to be able to define pod specs with embedded ephemeral CSI volumes that are created/mounted when the pod is deployed and are deleted when the pod goes away.

### Examples

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
          driver: some-csi-driver.example.com
          # Passed as NodePublishVolumeRequest.volume_context,
          # valid options depend on the driver.
          volumeAttributes:
              foo: bar
```

## Ephemeral inline volume proposal
A CSI driver may be able to support either PV/PVC-originated or pod spec originated volumes. When a volume definition is embedded inside a pod spec, it is considered to be an `ephemeral inline` volume request and can only participate in *mount/unmount* volume operation calls.  Ephemeral inline volume requests have the following characteristics: 
* The inline volume spec will not contain nor require a `volumeHandle`.
* The CSI Kubelet plugin will internally generate a `volumeHandle` which is passed to the driver.
* Using existing strategy, the volumeHandle will be cached for future volume operations (i.e. unmount).
* The Kubelet will send mount related calls to CSI drivers:
  * Kubelet will have access to both podUID and pod namespace during mount/Setup operations.
  * Secrets references can be fully realized during mount/Setup phase and sent to driver.
* The Kubelet will send unmount related calls to CSI drivers:
  * The cached volumeHandle will be sent to the driver during unmount/Teardown phase.

### VolumeHandle generation
During mount operation, the Kubelet (internal CSI code) will employ a naming strategy to generate the value for the `volumeHandle`.  The generated value will be a combination of `podUID` and `pod.spec.Volume[x].name` to guarantee uniqueness.  The generated value will be stable and the Kubelet will be able to regenerate the value, if needed, during different phases of storage operations.

This approach provides several advantages:
* It makes sure that each pod can use a different volume handle ID for its ephemeral volumes.  
* Each pod will get a uniquely generated volume handle, preventing accidental naming conflicts in pods.
* Each pod created by ReplicaSet, StatefulSet or DaemonSet will get the same copy of a pod template. This makes sure that each pod gets its own unique volume handle ID and thus can get its own volume instance.

Without an auto-generated naming strategy for the `volumeHandle` during an ephemeral lifecycle, a user could guess the volume handle ID of another user causing a security risk. Having a strategy that generates consistent volume handle names, will ensure that drivers obeying idempotency will always return the same volume associated with the podUID. 

### API updates

There are couple of objects needed to implement this feature:
* `VolumeSource` - object that represents a pod's volume. It will be modified to include CSI volume source.
* `CSIVolumeSource` - a new object representing the inline volume data coming from the pod.

```go
type VolumeSource struct {
    // <snip>
    // CSI (Container Storage Interface) represents storage that handled by an external CSI driver (Beta feature).
    // +optional
    CSI *CSIVolumeSource
}

// Represents a source location of a volume to mount, managed by an external CSI driver
type CSIVolumeSource struct {
	// Driver is the name of the driver to use for this volume.
	// Required.
	Driver string

	// Optional: The value to pass to ControllerPublishVolumeRequest.
	// Defaults to false (read/write).
	// +optional
	ReadOnly *bool

	// Filesystem type to mount. Ex. "ext4", "xfs", "ntfs".
	// If not provided, the empty value is passed to the associated CSI driver
	// which will determine the default filesystem to apply.
	// +optional
	FSType *string

	// VolumeAttributes store immutable properties of the volume copied during provision.
	// These attributes are passed back to the driver during controller publish calls.
	// +optional
	VolumeAttributes map[string]string

	// NodePublishSecretRef is a reference to the secret object containing
	// sensitive information to pass to the CSI driver to complete the CSI
	// NodePublishVolume and NodeUnpublishVolume calls.
	// This field is optional, and  may be empty if no secret is required. If the
	// secret object contains more than one secret, all secret references are passed.
	// +optional
	NodePublishSecretRef *LocalObjectReference
}
```

### Support for inline CSI volumes

To indicate that the driver will support ephemeral inline volume requests, the existing `CSIDriver` object will be extended to include attribute `VolumeLifecycleModes`,
a list of strings. That list may contain:
- `persistent` if the driver supports normal, persistent volumes (i.e. the normal CSI API); this is the default if nothing is specified
- `ephemeral` if the driver supports inline CSI volumes

Kubelet will check for support for ephemeral volumes before invoking
the CSI driver as described next. This prevents accidentally using a
CSI driver in a way which it doesn't support. This is important
because using a driver incorrectly might end up causing data loss or
other problems.

When a CSI driver supports it, the following approach is used:
* Volume requests will originate from pod specs.
* The driver will only receive volume operation calls during mount/unmount phase (`NodePublishVolume`, `NodeUnpublishVolume`)
* The driver will not receive separate gRPC calls for provisioning, attaching, detaching, and deleting of volumes.
* The driver is responsible for implementing steps to ensure the volume is created and made available to the pod during mount call.
* The Kubelet may attempt to mount a path, with the same generated volumeHandle, more than once. If that happens, the driver should be able to handle such cases gracefully.
* The driver is responsible for implementing steps to delete and clean up any volume and resources during the unmount call.
* The Kubelet may attempt to call unmount, with the same generated volumeHandle, more than once. If that happens, the driver should be able to handle such cases gracefully.
* `CSIVolumeSource.FSType` is mapped to `NodePublishVolumeRequest.access_type.mount.fs_type`.
* All other parameters that a driver might need (like volume size)
  have to be specified in `CSIVolumeSource.VolumeAttributes` and will be passed in
  `NodePublishVolumeRequest.volume_context`. What those parameters are is entirely
  up to the CSI driver.

A driver that supports both modes may need to distinguish in
`NodePublishVolume` whether the volume is ephemeral or persistent.
This can be done by enabling the "[pod info on
mount](https://kubernetes-csi.github.io/docs/csi-driver-object.html#what-fields-does-the-csidriver-object-have)"
feature which then, in addition to information about the pod, will
also set an entry with this key in the `NodePublishRequest.volume_context`:
* `csi.storage.k8s.io/ephemeral`: `true` for ephemeral inline volumes, `false` otherwise

### Secret reference
The secret reference declared in an ephemeral inline volume can only be used with namespaces from pods where it is referenced.  The `NodePublishSecretRef` is stored in a `LocalObjectReference` value:
* `LocalObjectReference` do not include a namespace reference.  This is to prevent reference to arbitrary namespace values.
* The namespace needed will be extracted from the the pod spec by the Kubelet code during mount.

### Specifying allowed inline drivers with `PodSecurityPolicy`
To control which CSI driver is allowed to be use ephemeral inline volumes within a pod spec, a new `PodSecurityPolicy` called `AllowedCSIDrivers` is introduced as shown below:

```go
  type PodSecurityPolicySpec struct {
	// <snip>

	// AllowedCSIDrivers is a whitelist of allowed CSI drivers used inline in a pod spec.  Empty or nil indicates that no
	// CSI drivers may be used in this way. This parameter is effective only when the usage of the CSI plugin
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

### Ephemeral inline volume operations
Inline volume requests can only participate in mount/unmount volume operations. This phase is handled by the Kubelet which is responsible for mounting/unmounting device and/or filesystem mount points inside a pod. At mount time, the internal API will pass the volume information via parameter of `volume.Spec` which will contain a value of either type `v1.CSIVolumeSource` (for volume originated from pod specs) or `v1.CSIPersistentVolume` for volume originating from PV/PVC.  The code will check for the presence of a `v1.CSIVolumeSource` or `v1.CSIPersistentVolume` value.  If a `v1.CSIPersistentVolume` is found, the operation is considered non-ephemeral and follows regular PV/PVC execution flow.  If, however, the internal volume API passes a `v1.CSIVolumeSource`:
* The Kubelet will create necessary mount point paths
* Kubelet will auto-generate a volumeHandle based on `podUID` and `pod.spec.volume[x].name` (see above for detail).
* CSI driver will receive mount-like calls (NodePublish) with generated paths and generated volumeHandle.

Since ephemeral volume requests will participate in only the mount/unmount volume operation phase, CSI drivers are responsible for implementing all necessary operations during that phase (i.e. create, mount, unmount, delete, etc).  For instance, a driver would be responsible for provisioning any new volume resource during `NodePublish` and for tearing down these resources during the `NodeUnpublish` calls.


## Test plans

### All unit tests
* Volume operation that use CSIVolumeSource can only work with proper feature gate enabled

### Ephemeral inline volumes unit tests
* Ensure required fields are provided: csi.storage.k8s.io/ephemeral (https://github.com/pohly/kubernetes/blob/4bc5d065c919fc239e2c8b40e6a96e409ca011fd/pkg/volume/csi/csi_mounter_test.go#L140-L146)
* Mount/Unmount should be triggered with CSIVolumeSource: https://github.com/kubernetes/kubernetes/blob/10005d2e1e1425904f8c7bf5615e730fb0fea7c9/pkg/volume/csi/csi_mounter_test.go#L386
* Expected generated volumeHandle is created properly: https://github.com/kubernetes/kubernetes/blob/10005d2e1e1425904f8c7bf5615e730fb0fea7c9/pkg/volume/csi/csi_plugin_test.go#L177
* Ensure that CSIDriver.Spec.Mode field is validated properly: https://github.com/kubernetes/kubernetes/pull/80568
* Ensure volumeHandle conforms to resource naming format: TODO
* CSIVolumeSource info persists in CSI json file during mount/unmount: TODO
* Ensure Kubelet skips attach/detach when `CSIDriver.Mode = ephemeral`: TODO
* Ensure Kubelet skips inline logic when `CSIDriver.Mode = persistent` or `CSIDriver.Mode is empty`: covered by existing tests

### E2E tests
* Pod spec with an ephemeral inline volume request can be mounted/unmounted: https://github.com/pohly/kubernetes/blob/4bc5d065c919fc239e2c8b40e6a96e409ca011fd/test/e2e/storage/csi_mock_volume.go#L356-L371, https://github.com/pohly/kubernetes/blob/4bc5d065c919fc239e2c8b40e6a96e409ca011fd/test/e2e/storage/testsuites/ephemeral.go#L110-L115
* Two pods accessing an ephemeral inline volume which has the same attributes in both pods: "should support two pods which share the same data" in `ephemeral.go` (upcoming PR)
* Single pod referencing two distinct inline volume request from the same driver: "should support multiple inline ephemeral volumes" in `ephemeral.go` (upcoming PR)
* CSI Kubelet code invokes driver operations during mount for ephemeral volumes: `checkPodLogs` in `csi_mock_volume.go` (upcoming PR)
* CSI Kubelet code invokes driver operation during unmount of ephemeral volumes: `checkPodLogs` in `csi_mock_volume.go` (upcoming PR)
* CSI Kubelet cleans up ephemeral volume paths once pod goes away: TODO
* Apply PodSecurity settings for allowed CSI drivers: TODO
* Enable testing of an external ephemeral CSI driver: https://github.com/kubernetes/kubernetes/pull/79983/files#diff-e5fc8d9911130b421b74b1ebc273f458
* Enable testing of the csi-host-path-driver in ephemeral mode in Kubernetes-CSI Prow jobs and Kubernetes itself: TODO

## Alternatives

Instead of allowing CSI drivers that support both ephemeral and
persistent volumes and passing the `csi.storage.k8s.io/ephemeral`
hint, a simpler solution would have been to require that a driver gets
deployed twice, once for for each kind of volume. That was rejected
because a driver might manage some resource that is shared between
both kinds of volumes, like local disks (LVM) or persistent memory
(PMEM). Having to deploy the driver twice would have made the driver
implementation more complex.

## Implementation History

1.15:
- Alpha status
- `CSIDriver.Mode` not added yet
- a CSI driver deployment can only be used for ephemeral inline
  volumes or persistent volumes, but not both, because the driver
  cannot determine the mode in its `NodePublishVolume` implementation

1.16:
- Beta status
- the same CSI driver deployment can support both modes by enabling
  the pod info feature and checking the value of
  `csi.storage.k8s.io/ephemeral`
  (https://github.com/kubernetes/kubernetes/pull/79983, merged)
- `CSIDriver.VolumeLifecycleMode` added and checked before calling a CSI driver for
  an ephemeral inline volume
  (https://github.com/kubernetes/kubernetes/pull/80568, merged)
