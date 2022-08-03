# KEP-596: Ephemeral Inline CSI volumes

## Table of Contents

<!-- toc -->
- [Release Signoff Checklist](#release-signoff-checklist)
- [Summary](#summary)
- [Motivation](#motivation)
  - [Goals](#goals)
  - [Non-goals](#non-goals)
- [Proposal](#proposal)
  - [User Stories](#user-stories)
    - [Examples](#examples)
  - [Risks and Mitigations](#risks-and-mitigations)
    - [Security Considerations](#security-considerations)
- [Design Details](#design-details)
  - [VolumeHandle generation](#volumehandle-generation)
  - [API updates](#api-updates)
  - [Support for inline CSI volumes](#support-for-inline-csi-volumes)
  - [Secret reference](#secret-reference)
  - [Ephemeral inline volume operations](#ephemeral-inline-volume-operations)
  - [Read-only volumes](#read-only-volumes)
  - [Test Plan](#test-plan)
      - [Prerequisite testing updates](#prerequisite-testing-updates)
      - [Unit tests](#unit-tests)
      - [Integration tests](#integration-tests)
      - [e2e tests](#e2e-tests)
  - [Graduation Criteria](#graduation-criteria)
    - [Alpha](#alpha)
    - [Beta](#beta)
    - [GA](#ga)
- [Production Readiness Review Questionnaire](#production-readiness-review-questionnaire)
  - [Feature Enablement and Rollback](#feature-enablement-and-rollback)
  - [Rollout, Upgrade and Rollback Planning](#rollout-upgrade-and-rollback-planning)
  - [Monitoring Requirements](#monitoring-requirements)
  - [Dependencies](#dependencies)
  - [Scalability](#scalability)
  - [Troubleshooting](#troubleshooting)
- [Alternatives](#alternatives)
- [Implementation History](#implementation-history)
<!-- /toc -->

## Release Signoff Checklist

Items marked with (R) are required *prior to targeting to a milestone / release*.

- [x] (R) Enhancement issue in release milestone, which links to KEP dir in [kubernetes/enhancements] (not the initial KEP PR)
- [x] (R) KEP approvers have approved the KEP status as `implementable`
- [x] (R) Design details are appropriately documented
- [x] (R) Test plan is in place, giving consideration to SIG Architecture and SIG Testing input (including test refactors)
  - [x] e2e Tests for all Beta API Operations (endpoints)
  - [x] (R) Ensure GA e2e tests for meet requirements for [Conformance Tests](https://github.com/kubernetes/community/blob/master/contributors/devel/sig-architecture/conformance-tests.md)
  - [x] (R) Minimum Two Week Window for GA e2e tests to prove flake free
- [x] (R) Graduation criteria is in place
  - [ ] (R) [all GA Endpoints](https://github.com/kubernetes/community/pull/1806) must be hit by [Conformance Tests](https://github.com/kubernetes/community/blob/master/contributors/devel/sig-architecture/conformance-tests.md) 
- [x] (R) Production readiness review completed
- [x] (R) Production readiness review approved
- [ ] "Implementation History" section is up-to-date for milestone
- [ ] User-facing documentation has been created in [kubernetes/website], for publication to [kubernetes.io]
- [ ] Supporting documentation—e.g., additional design documents, links to mailing list discussions/SIG meetings, relevant PRs/issues, release notes

[kubernetes.io]: https://kubernetes.io/
[kubernetes/enhancements]: https://git.k8s.io/enhancements
[kubernetes/kubernetes]: https://git.k8s.io/kubernetes
[kubernetes/website]: https://git.k8s.io/website


## Summary

Currently, volumes that are backed by CSI drivers can only be used with the `PersistentVolume` and `PersistentVolumeClaim` objects. This proposal is to implement support for the ability to nest CSI volume declarations within pod specs for ephemeral-style drivers.

This KEP started life as [feature #2273](https://github.com/kubernetes/community/pull/2273).  Please follow that link for historical context.

## Motivation

Implementing support for embedding volumes directly in pod specs would allow driver developers to create new types of CSI drivers such as ephemeral volume drivers.  They can be used to inject arbitrary states, such as configuration, secrets, identity, variables or similar information, directly inside pods using a mounted volume. 

### Goals 

* Provide a high level design for ephemeral inline CSI volumes support
* Define API changes needed to support this feature
* Outlines how ephemeral inline CSI volumes would work 

### Non-goals

The followings will not be addressed by this KEP:
* Introduce required changes to existing CSI drivers for this feature
* Support for topology or pod placement scheme for ephemeral inline volumes
* Support for PV/PVC related features such as topology, raw block, mount options, and resizing
* Support for inline pod specs backed by a persistent volumes

## Proposal

A CSI driver may be able to support either PV/PVC-originated or pod spec originated volumes. When a volume definition is embedded inside a pod spec, it is considered to be an `ephemeral inline` volume request and can only participate in *mount/unmount* volume operation calls.  Ephemeral inline volume requests have the following characteristics: 
* The inline volume spec will not contain nor require a `volumeHandle`.
* The CSI Kubelet plugin will internally generate a `volumeHandle` which is passed to the driver.
* Using existing strategy, the volumeHandle will be cached for future volume operations (i.e. unmount).
* The Kubelet will send mount related calls to CSI drivers:
  * Kubelet will have access to both podUID and pod namespace during mount/Setup operations.
  * Secrets references can be fully realized during mount/Setup phase and sent to driver.
* The Kubelet will send unmount related calls to CSI drivers:
  * The cached volumeHandle will be sent to the driver during unmount/Teardown phase.

### User Stories

* As a storage provider, I want to use the CSI API to develop drivers that can mount ephemeral volumes that follow the lifecycles of pods where they are embedded.   This feature would allow me to create drivers that work similarly to how the in-tree Secrets or ConfigMaps driver works.  My ephemeral CSI driver should allow me to inject arbitrary data into a pod using a volume mount point inside the pod. 
* As a user I want to be able to define pod specs with embedded ephemeral CSI volumes that are created/mounted when the pod is deployed and are deleted when the pod goes away.

#### Examples

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

### Risks and Mitigations

#### Security Considerations

CSI driver vendors that support inline volumes will be responsible for secure handling of volumes.

For example, `csi-driver-nfs` allows anybody who can create a pod to mount any NFS volume into that pod, when the cluster admin deploys the driver with csi driver instance allowing ephemeral use. This option is not on by default, but may be surprising to some admins that this allows mounting of any NFS volume and could be unsafe without additional authorization checks.

Downstream distributions and cluster admins that wish to exercise fine-grained control over which CSI drivers are allowed to use ephemeral inline volumes within a pod spec should do so with a 3rd party pod admission plugin or webhook (not part of this KEP).

The [Kubernetes docs](https://kubernetes.io/docs/concepts/storage/ephemeral-volumes/) and [CSI docs](https://kubernetes-csi.github.io/docs/ephemeral-local-volumes.html) have been updated to include the security aspects of inline CSI volumes and recommend CSI driver vendors not implement inline volumes for persistent storage unless they also provide a 3rd party pod admission plugin.

This is consistent with the proposal by sig-auth in [KEP-2579](https://github.com/kubernetes/enhancements/blob/787515fbfa386bed95ff4d21e472474f61d1c536/keps/sig-auth/2579-psp-replacement/README.md?plain=1#L512-L519) regarding how inline CSI volumes should be handled.

## Design Details

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
* The namespace needed will be extracted from the pod spec by the Kubelet code during mount.

### Ephemeral inline volume operations

Inline volume requests can only participate in mount/unmount volume operations. This phase is handled by the Kubelet which is responsible for mounting/unmounting device and/or filesystem mount points inside a pod. At mount time, the internal API will pass the volume information via parameter of `volume.Spec` which will contain a value of either type `v1.CSIVolumeSource` (for volume originated from pod specs) or `v1.CSIPersistentVolume` for volume originating from PV/PVC.  The code will check for the presence of a `v1.CSIVolumeSource` or `v1.CSIPersistentVolume` value.  If a `v1.CSIPersistentVolume` is found, the operation is considered non-ephemeral and follows regular PV/PVC execution flow.  If, however, the internal volume API passes a `v1.CSIVolumeSource`:
* The Kubelet will create necessary mount point paths
* Kubelet will auto-generate a volumeHandle based on `podUID` and `pod.spec.volume[x].name` (see above for detail).
* CSI driver will receive mount-like calls (NodePublish) with generated paths and generated volumeHandle.

Since ephemeral volume requests will participate in only the mount/unmount volume operation phase, CSI drivers are responsible for implementing all necessary operations during that phase (i.e. create, mount, unmount, delete, etc).  For instance, a driver would be responsible for provisioning any new volume resource during `NodePublish` and for tearing down these resources during the `NodeUnpublish` calls.

### Read-only volumes

It is possible for a CSI driver to provide its volumes to Pods as read-only, while keeping them read-write to the driver and to kubelet / the container runtime. This allows the CSI driver to dynamically update content of their Secrets-like volumes and at the same time users can't expose issues like [CVE-2017-1002102](https://github.com/kubernetes/kubernetes/issues/60814), because the volume is read-only to them. In addition, it allows kubelet to apply `fsGroup` to the volume and the container runtime to change SELinux context of files on the volume, so the Pod gets the volume with expected owner and SELinux label.

To benefit from this behavior, the following should be implemented in the CSI driver:

* The driver should provide an admission plugin that sets `ReadOnly: true` to all volumeMounts of such volumes.
  * We can't trust users to do that correctly in all their pods.
  * Presence of `ReadOnly: true` tells kubelet to mount the volume as read-only to the Pod.
* The driver should check in all NodePublish requests that readonly flag is set.
  * We can't trust cluster admins that they deploy the admission webhook mentioned above.
* When both conditions above are satisfied, the driver MAY ignore the `readonly` flag in [NodePublish](https://github.com/container-storage-interface/spec/blob/5b0d4540158a260cb3347ef1c87ede8600afb9bf/csi.proto#L1375) and set up the volume as read-write. Kubelet then can apply fsGoup if needed. Seeing `ReadOnly: true` in the Pod spec, kubelet then tells CRI to bind-mount the volume to the container as read-only, while it's read-write on the host / in the CSI driver. This behavior is already implemented in Kubelet for all projected volumes (Secrets, ConfigMap, Projected and DownwardAPI), we're allowing ephemeral CSI driver to reuse it for their Secrets-like volumes.

This behavior is documented in the [CSI docs](https://kubernetes-csi.github.io/docs/ephemeral-local-volumes.html). Ignoring the `readonly` flag in [NodePublish](https://github.com/container-storage-interface/spec/blob/5b0d4540158a260cb3347ef1c87ede8600afb9bf/csi.proto#L1375) of in-line CSI volumes will be supported as a valid CSI driver behavior.

Examples where this is used by the Secrets Store CSI driver:
- [NodePublish ReadOnly check](https://github.com/kubernetes-sigs/secrets-store-csi-driver/blob/d32ca72038650c79561092dab26bf6d5a9c9e40a/pkg/secrets-store/nodeserver.go#L174-L177)
- [Ignoring readonly and providing read-write volume](https://github.com/kubernetes-sigs/secrets-store-csi-driver/blob/d32ca72038650c79561092dab26bf6d5a9c9e40a/pkg/secrets-store/nodeserver.go#L202)


### Test Plan

[x] I/we understand the owners of the involved components may require updates to
existing tests to make this code solid enough prior to committing the changes necessary
to implement this enhancement.

##### Prerequisite testing updates

<!--
Based on reviewers feedback describe what additional tests need to be added prior
implementing this enhancement to ensure the enhancements have also solid foundations.
-->

##### Unit tests

<!--
In principle every added code should have complete unit test coverage, so providing
the exact set of tests will not bring additional value.
However, if complete unit test coverage is not possible, explain the reason of it
together with explanation why this is acceptable.
-->

<!--
Additionally, for Alpha try to enumerate the core package you will be touching
to implement this enhancement and provide the current unit coverage for those
in the form of:
- <package>: <date> - <current test coverage>
The data can be easily read from:
https://testgrid.k8s.io/sig-testing-canaries#ci-kubernetes-coverage-unit

This can inform certain test coverage improvements that we want to do before
extending the production code to implement this enhancement.
-->

- `k8s.io/kubernetes/pkg/volume/csi`: `6/9/22` - `76.3`
  - `k8s.io/kubernetes/pkg/volume/csi/csi_attacher.go`: `6/9/22` - `78.2`
  - `k8s.io/kubernetes/pkg/volume/csi/csi_client.go`: `6/9/22` - `76.4`
  - `k8s.io/kubernetes/pkg/volume/csi/csi_mounter.go`: `6/9/22` - `82.1`
  - `k8s.io/kubernetes/pkg/volume/csi/csi_plugin.go`: `6/9/22` - `75.1`
  - `k8s.io/kubernetes/pkg/volume/csi/csi_util.go`: `6/9/22` - `93.2`

Specific unit tests that were implemented for this feature:
- [Volume operations that use CSIVolumeSource can only work with proper feature gate enabled](https://github.com/kubernetes/kubernetes/blob/163aab43d7d1a279dfa4a261202a8f424933e7dd/pkg/volume/csi/csi_plugin_test.go#L668))
- [Ensure required fields are provided: csi.storage.k8s.io/ephemeral](https://github.com/kubernetes/kubernetes/blob/163aab43d7d1a279dfa4a261202a8f424933e7dd/pkg/volume/csi/csi_mounter_test.go#L154-L160)
- [Mount/Unmount should be triggered with CSIVolumeSource](https://github.com/kubernetes/kubernetes/blob/163aab43d7d1a279dfa4a261202a8f424933e7dd/pkg/volume/csi/csi_mounter_test.go#L504)
- [driverPolicy is ReadWriteOnceWithFSTypeFSGroupPolicy with CSI inline volume](https://github.com/kubernetes/kubernetes/blob/163aab43d7d1a279dfa4a261202a8f424933e7dd/pkg/volume/csi/csi_mounter_test.go#L1205)
- [Expected generated volumeHandle is created properly](https://github.com/kubernetes/kubernetes/blob/163aab43d7d1a279dfa4a261202a8f424933e7dd/pkg/volume/csi/csi_plugin_test.go#L280)
- [CanSupport works with CSI inline volumes](https://github.com/kubernetes/kubernetes/blob/163aab43d7d1a279dfa4a261202a8f424933e7dd/pkg/volume/csi/csi_plugin_test.go#L372)
- [ConstructVolumeSpec works with CSI inline volumes](https://github.com/kubernetes/kubernetes/blob/163aab43d7d1a279dfa4a261202a8f424933e7dd/pkg/volume/csi/csi_plugin_test.go#L506)
- [NewMounter works with CSI inline volumes](https://github.com/kubernetes/kubernetes/blob/163aab43d7d1a279dfa4a261202a8f424933e7dd/pkg/volume/csi/csi_plugin_test.go#L757)
- [CanAttach works with CSI inline volumes](https://github.com/kubernetes/kubernetes/blob/163aab43d7d1a279dfa4a261202a8f424933e7dd/pkg/volume/csi/csi_plugin_test.go#L995)
- [FindAttachablePlugin works with CSI inline volumes](https://github.com/kubernetes/kubernetes/blob/163aab43d7d1a279dfa4a261202a8f424933e7dd/pkg/volume/csi/csi_plugin_test.go#L1049)
- [CanDeviceMount works with CSI inline volumes](https://github.com/kubernetes/kubernetes/blob/163aab43d7d1a279dfa4a261202a8f424933e7dd/pkg/volume/csi/csi_plugin_test.go#L1125)
- [FindDeviceMountablePluginBySpec works with CSI inline volumes](https://github.com/kubernetes/kubernetes/blob/163aab43d7d1a279dfa4a261202a8f424933e7dd/pkg/volume/csi/csi_plugin_test.go#L1177)
- [Ensure that CSIDriver.VolumeLifecycleModes field is validated properly](https://github.com/kubernetes/kubernetes/pull/80568)

##### Integration tests

<!--
This question should be filled when targeting a release.
For Alpha, describe what tests will be added to ensure proper quality of the enhancement.

For Beta and GA, add links to added tests together with links to k8s-triage for those tests:
https://storage.googleapis.com/k8s-triage/index.html
-->

See E2E tests below.

##### e2e tests

<!--
This question should be filled when targeting a release.
For Alpha, describe what tests will be added to ensure proper quality of the enhancement.

For Beta and GA, add links to added tests together with links to k8s-triage for those tests:
https://storage.googleapis.com/k8s-triage/index.html

We expect no non-infra related flakes in the last month as a GA graduation criteria.
-->

- [TestPattern: CSI Ephemeral-volume (default fs)](https://github.com/kubernetes/kubernetes/blob/7c127b33dafc530f7ca0c165ddb47db86eb45880/test/e2e/storage/framework/testpattern.go#L98-L102): [test coverage](https://storage.googleapis.com/k8s-triage/index.html?test=.*CSI%20Ephemeral-volume%20%5C%28default%20fs%5C%29.*)
- [should create read-only inline ephemeral volume](https://github.com/kubernetes/kubernetes/blob/7c127b33dafc530f7ca0c165ddb47db86eb45880/test/e2e/storage/testsuites/ephemeral.go#L175): [test coverage](https://storage.googleapis.com/k8s-triage/index.html?test=should%20create%20read-only%20inline%20ephemeral%20volume)
- [should create read/write inline ephemeral volume](https://github.com/kubernetes/kubernetes/blob/7c127b33dafc530f7ca0c165ddb47db86eb45880/test/e2e/storage/testsuites/ephemeral.go#L196): [test coverage](https://storage.googleapis.com/k8s-triage/index.html?test=should%20create%20read%2Fwrite%20inline%20ephemeral%20volume)
- [should support two pods which have the same volume definition](https://github.com/kubernetes/kubernetes/blob/7c127b33dafc530f7ca0c165ddb47db86eb45880/test/e2e/storage/testsuites/ephemeral.go#L277): [test coverage](https://storage.googleapis.com/k8s-triage/index.html?test=should%20support%20two%20pods%20which%20have%20the%20same%20volume%20definition)
- [should support multiple inline ephemeral volumes](https://github.com/kubernetes/kubernetes/blob/7c127b33dafc530f7ca0c165ddb47db86eb45880/test/e2e/storage/testsuites/ephemeral.go#L315): [test coverage](https://storage.googleapis.com/k8s-triage/index.html?test=should%20support%20multiple%20inline%20ephemeral%20volumes)
- [contain ephemeral=true when using inline volume](https://github.com/kubernetes/kubernetes/blob/7c127b33dafc530f7ca0c165ddb47db86eb45880/test/e2e/storage/csi_mock_volume.go#L495): [test coverage](https://storage.googleapis.com/k8s-triage/index.html?test=contain%20ephemeral%3Dtrue%20when%20using%20inline%20volume)


### Graduation Criteria

#### Alpha

- Feature implemented behind `CSIInlineVolume` feature flag
- Initial unit tests and e2e tests implemented

#### Beta

- Feature flag enabled by default
- CSI drivers can support both ephemeral inline volumes and persistent volumes

#### GA

- [x] Remove dependency on deprecated `PodSecurityPolicy` and document new strategy
- [x] Fix for [#89290 - CSI Inline volume panic when calling applyFSGroup](https://github.com/kubernetes/kubernetes/issues/89290)
- [x] Fix for [#79980 - CSI volume reconstruction](https://github.com/kubernetes/kubernetes/issues/79980)
- [x] Updated documentation as described in [Security Considerations](#security-considerations) and [Read-only volumes](#read-only-volumes)
- [ ] Upgrade / downgrade manual testing, document results in the [upgrade / rollback section](#rollout-upgrade-and-rollback-planning).
- [ ] Provide measurements for the [Scalability section](#scalability) (time taken to start a pod)
- [ ] Ensure our sponsored [NFS](https://github.com/kubernetes-csi/csi-driver-nfs) and [SMB](https://github.com/kubernetes-csi/csi-driver-smb) CSI drivers align with the new guidance in [Security Considerations](#security-considerations)
- [ ] Conformance tests implemented / promoted
- [ ] Feature flag set to GA


## Production Readiness Review Questionnaire

### Feature Enablement and Rollback

###### How can this feature be enabled / disabled in a live cluster?

- [x] Feature gate (also fill in values in `kep.yaml`)
  - Feature gate name: CSIInlineVolume
  - Components depending on the feature gate:
    - kubelet
    - kube-apiserver

###### Does enabling the feature change any default behavior?

  Enabling the feature gate will **not** change the default behavior.
  Each CSI driver must opt-in to support ephemeral inline volumes, and this
  feature needs to be explicitly requested in the PodSpec (see [Examples](#examples)).

###### Can the feature be disabled once it has been enabled (i.e. can we roll back the enablement)?

  Yes. Disabling the feature gate will disable support for inline CSI volumes.
  It can also be disabled on a specific CSI driver by removing the "Ephemeral"
  VolumeLifecycleMode from `CSIDriver.VolumeLifecycleModes`.

###### What happens if we reenable the feature if it was previously rolled back?

  Reenabling the feature will again allow CSI drivers that opt-in to be used
  as ephemeral inline volumes in the pod spec.

###### Are there any tests for feature enablement/disablement?

  Yes, the unit tests will test with the feature gate enabled and disabled.

### Rollout, Upgrade and Rollback Planning

###### How can a rollout or rollback fail? Can it impact already running workloads?

  A rollout should not impact running workloads. For pods that are already running,
  there should be no effect. Any workloads that need to make use of this feature
  would require pod spec updates, along with a CSI driver that supports it.

  A rollback (or disabling the feature flag) will not impact already running
  workloads, however starting new workloads using this feature (as well as
  potentially restarting ones that failed for some reason) would fail.
  Additionally, kubelet may fail to fully cleanup after pods that were
  running and taking advantage of inline volumes.

  *It is highly recommended to delete any pods using this feature before disabling it*.

###### What specific metrics should inform a rollback?

  An increased failure rate of volume mount operations can be used as an indication
  of a problem. In particular, the following metrics:

  - Metric name: storage_operation_duration_seconds
    - [Optional] Aggregation method: filter by `operation_name='volume_mount',status='fail-unknown'`
      - Components exposing the metric: kubelet
      - Shows failure rate of volume mount operations

  - Metric name: csi_operations_seconds
    - [Optional] Aggregation method: filter by `method_name='NodePublishVolume',grpc_status_code!='0'`
      - Components exposing the metric: kubelet
      - May also filter by `driver_name` to narrow down to a specific driver
      - Shows failure rate of CSI NodePublishVolume requests

###### Were upgrade and rollback tested? Was the upgrade->downgrade->upgrade path tested?

  TODO: To be documented as part of manual GA testing.

<!--
Describe manual testing that was done and the outcomes.
Longer term, we may want to require automated upgrade/rollback tests, but we
are missing a bunch of machinery and tooling and can't do that now.
-->

###### Is the rollout accompanied by any deprecations and/or removals of features, APIs, fields of API types, flags, etc.?

  No, this feature does not deprecate any existing functionality.

### Monitoring Requirements

###### How can an operator determine if the feature is in use by workloads?

  Check if any CSIDriver specs have `VolumeLifecycleModes` including an
  "Ephemeral" `VolumeLifecycleMode` and if any pod specs are using this driver inline.

###### How can someone using this feature know that it is working for their instance?

For individual pods using an inline CSI driver (as in [Examples](#examples)),
review the status of the pod:

- [x] API Pod.status
  - Condition name: Ready
  - Other field: Pod.status.phase = Running

For an aggregated view at the cluster level, check the failure rate of
CSI NodePublishVolume requests:

- [x] Metric: csi_operations_seconds
  - Aggregation method: filter by `method_name='NodePublishVolume',grpc_status_code!='0'`
    - Components exposing the metric: kubelet
    - May also filter by `driver_name` to narrow down to a specific driver

###### What are the reasonable SLOs (Service Level Objectives) for the enhancement?

- No increased failure rates during mount operations.
- Mount times should be expected to be less than or equal to the default behavior.

###### What are the SLIs (Service Level Indicators) an operator can use to determine the health of the service?

- [x] Metrics
  - Metric name: storage_operation_duration_seconds
    - [Optional] Aggregation method: filter by `operation_name='volume_mount',status='fail-unknown'`
      - Components exposing the metric: kubelet
      - Shows failure rate of volume mount operations
  - Metric name: storage_operation_duration_seconds
    - [Optional] Aggregation method: filter by `operation_name='volume_mount'`
      - Components exposing the metric: kubelet
      - Shows time taken by volume mount operations
  - Metric name: csi_operations_seconds
    - [Optional] Aggregation method: filter by `method_name='NodePublishVolume',grpc_status_code!='0'`
      - Components exposing the metric: kubelet
      - May also filter by `driver_name` to narrow down to a specific driver
      - Shows failure rate of CSI NodePublishVolume requests

###### Are there any missing metrics that would be useful to have to improve observability of this feature?

  No additional metrics needed.

### Dependencies

###### Does this feature depend on any specific services running in the cluster?

  - [CSI Drivers]
    - Usage description: In order for this feature to be used by a pod, there
      must be a CSI Driver deployed that supports ephemeral inline volumes.
      - Impact of its outage on the feature:
        If the CSI Driver is unavailable, pods attempting to mount a volume with
        that driver will be unable to start.
      - Impact of its degraded performance or high-error rates on the feature:
        Degraded performance or high-error rates of the CSI Driver will delay
        or prevent pods using that driver from starting.

### Scalability

###### Will enabling / using this feature result in any new API calls?

  There will be no new API calls.

###### Will enabling / using this feature result in introducing new API types?

  The only new API type is the `VolumeLifecycleMode` string that is used
  to specify supported modes in the CSIDriver spec.

###### Will enabling / using this feature result in any new calls to the cloud provider?

  No, ephemeral inline volumes can only participate in mount/unmount volume operation
  calls to the CSI driver.

###### Will enabling / using this feature result in increasing size or count of the existing API objects?

  There is a new `VolumeLifecycleModes` field in CSIDriverSpec that may slightly increase
  the size by adding 2 new strings (`Persistent` and `Ephemeral`).

  There is also `pod.spec.volumes` that can contain `csi` items now. Its size is comparable
  to the size of Secrets/ConfigMap volumes inline in Pods. The CSI volume definitions will
  be bigger, as they contain a generic map of `volumeAttributes`, which contains opaque
  parameters for the CSI driver.

###### Will enabling / using this feature result in increasing time taken by any operations covered by existing SLIs/SLOs?

  Compared to CSI persistent volumes, there should be no increase in the amount of time
  taken to mount / unmount, as there will be fewer CSI calls required for inline volumes.
  Compared to Secrets and ConfigMaps, inline CSI volumes will be slower mount / unmount,
  since an external CSI driver is responsible for providing the actual volume.
  Note that mount time is in the critical path for [pod startup latency](https://github.com/kubernetes/community/blob/1181fb0266a01d1dfd170ff437817eb7de24b624/sig-scalability/slos/pod_startup_latency.md) and the choice to use CSI inline volumes may affect the SLI/SLO, since this is still a type of volume that needs to be mounted.

  TODO: Provide a measurements showing the time it takes to start a pod with 5 secrets,
  compared to mounting those secrets via a CSI inline volume. This will be driver
  dependent, but it would be useful to set some baseline expectations.

###### Will enabling / using this feature result in non-negligible increase of resource usage (CPU, RAM, disk, IO, ...) in any components?

  Kubernetes itself should not see any noticeable increase in resource consumption,
  but CSI driver pods will need to be deployed on all the nodes in order to make
  use of this feature.

### Troubleshooting

###### How does this feature react if the API server and/or etcd is unavailable?

  Pods will not start and volumes for them will not get provisioned.

###### What are other known failure modes?

  If the storage system fails to provision inline volumes, there will
  be an event in the affected pod indicating what went wrong when
  mounting the CSI volume.

  Failure modes related to individual CSI drivers may require examining
  the CSI driver logs.

###### What steps should be taken if SLOs are not being met to determine the problem?

  The kubelet and CSI driver logs should be inspected to look for errors mounting / unmounting the volume, or to help explain why operations are taking longer than expected. Pay special attention to `mounter.SetUpAt` and `NodePublishVolume` messages.

  Example kubelet log messages:
  - `mounter.SetUpAt failed to get CSI client`
  - `mounter.SetupAt failed to get CSI persistent source`
  - `CSIInlineVolume feature required`
  - `unexpected volume mode`

  CSI driver log messages will vary between drivers, but some common failure scenarios are:
  - `Volume capability missing in request`
  - `Target path missing in request`
  - `Failed to create target path`
  - `Failed to mount ...`

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
- `CSIDriver.VolumeLifecycleModes` not added yet
- a CSI driver deployment can only be used for ephemeral inline
  volumes or persistent volumes, but not both, because the driver
  cannot determine the mode in its `NodePublishVolume` implementation

1.16:
- Beta status
- the same CSI driver deployment can support both modes by enabling
  the pod info feature and checking the value of
  `csi.storage.k8s.io/ephemeral`
  (https://github.com/kubernetes/kubernetes/pull/79983, merged)
- `CSIDriver.VolumeLifecycleModes` added and checked before calling a CSI driver for
  an ephemeral inline volume
  (https://github.com/kubernetes/kubernetes/pull/80568, merged)

1.24:
- Remove dependency on deprecated `PodSecurityPolicy` and document new strategy
- Fix for [#89290 - CSI Inline volume panic when calling applyFSGroup](https://github.com/kubernetes/kubernetes/issues/89290)
- Updated documentation as described in [Security Considerations](#security-considerations) and [Read-only volumes](#read-only-volumes)

1.25:
- Fix for [#79980 - CSI volume reconstruction](https://github.com/kubernetes/kubernetes/issues/79980)
- GA status

