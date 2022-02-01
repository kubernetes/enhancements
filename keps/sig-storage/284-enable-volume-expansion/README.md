# Growing Persistent Volume size

## Table of Contents

<!-- toc -->
- [Release Signoff Checklist](#release-signoff-checklist)
- [Goals](#goals)
- [Non Goals](#non-goals)
- [Use Cases](#use-cases)
- [Volume Plugin Matrix](#volume-plugin-matrix)
- [Implementation Design](#implementation-design)
  - [Prerequisite](#prerequisite)
  - [Admission Control and Validations](#admission-control-and-validations)
  - [Controller Manager resize](#controller-manager-resize)
  - [File system resize on kubelet](#file-system-resize-on-kubelet)
    - [Prerequisite of File system resize](#prerequisite-of-file-system-resize)
    - [Steps for resizing file system available on Volume](#steps-for-resizing-file-system-available-on-volume)
    - [Reduce coupling between resize operation and file system type](#reduce-coupling-between-resize-operation-and-file-system-type)
- [API and UI Design](#api-and-ui-design)
- [API Changes](#api-changes)
  - [PVC API Change](#pvc-api-change)
  - [StorageClass API change](#storageclass-api-change)
  - [Other API changes](#other-api-changes)
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

- [x] (R) Enhancement issue in release milestone, which links to KEP dir in [kubernetes/enhancements] (not the initial KEP PR)
- [x] (R) KEP approvers have approved the KEP status as `implementable`
- [x] (R) Design details are appropriately documented
- [x] (R) Test plan is in place, giving consideration to SIG Architecture and SIG Testing input
- [x] (R) Graduation criteria is in place
- [ ] (R) Production readiness review completed
- [ ] Production readiness review approved
- [x] "Implementation History" section is up-to-date for milestone
- [x] User-facing documentation has been created in [kubernetes/website], for publication to [kubernetes.io]
- [ ] Supporting documentation e.g., additional design documents, links to mailing list discussions/SIG meetings, relevant PRs/issues, release notes

## Goals

Enable users to increase size of PVs that their pods are using. The user will update PVC for requesting a new size. Underneath we expect that - a controller will apply the change to PV which is bound to the PVC.

## Non Goals

* Reducing size of Persistent Volumes: We realize that, reducing size of PV is way riskier than increasing it. Reducing size of a PV could be a destructive operation and it requires support from underlying file system and volume type. In most cases it also requires that file system being resized is unmounted.

* Rebinding PV and PVC: Kubernetes will only attempt to resize the currently bound PV and PVC and will not attempt to relocate data  from a PV to a new PV and rebind the PVC to newly created PV.

## Use Cases

* As a user I am running Mysql on a 100GB volume - but I am running out of space, I should be able to increase size of volume mysql is using without losing all my data. (*online and with data*)
* As a user I created a PVC requesting 2GB space. I am yet to start a pod with this PVC but I realize that I probably need more space. Without having to create a new PVC, I should be able to request more size with same PVC. (*offline and no data on disk*)
* As a user I was running a rails application with 5GB of assets PVC. I have taken my application offline for maintenance but I would like to grow asset PVC to 10GB in size. (*offline but with data*)
* As a user I am running an application on glusterfs. I should be able to resize the gluster volume without losing data or mount point. (*online and with data and without taking pod offline*)
* In the logging project we run on dedicated clusters, we start out with 187Gi PVs for each of the elastic search pods. However, the amount of logs being produced can vary greatly from one cluster to another and its not uncommon that these volumes fill and we need to grow them.

## Volume Plugin Matrix


| Volume Plugin   | Supports Resize   | Requires File system Resize | Supported in 1.8 Release |
| ----------------| :---------------: | :--------------------------:| :----------------------: |
| EBS             | Yes               | Yes                         | Yes                      |
| GCE PD          | Yes               | Yes                         | Yes                      |
| GlusterFS       | Yes               | No                          | Yes                      |
| Cinder          | Yes               | Yes                         | Yes                      |
| Vsphere         | Yes               | Yes                         | No                       |
| Ceph RBD        | Yes               | Yes                         | No                       |
| Host Path       | No                | No                          | No                       |
| Azure Disk      | Yes               | Yes                         | No                       |
| Azure File      | No                | No                          | No                       |
| Cephfs          | No                | No                          | No                       |
| NFS             | No                | No                          | No                       |
| Flex            | Yes               | Maybe                       | No                       |
| LocalStorage    | Yes               | Yes                         | No                       |


## Implementation Design

For volume type that requires both file system expansion and a volume plugin based modification, growing persistent volumes will be two
step process.


For volume types that only require volume plugin based api call, this will be one step process.

### Prerequisite

* `pvc.spec.resources.requests.storage` field of pvc object will become mutable after this change.
* #sig-api-machinery has agreed to allow pvc's status update from kubelet as long as pvc and node relationship
  can be validated by node authorizer.
* This feature will be protected by an alpha feature gate, so as API changes needed for it.


### Admission Control and Validations

* Resource quota code has to be updated to take into account PVC expand feature.
* In case volume plugin doesn’t support resize feature. The resize API request will be rejected and PVC object will not be saved. This check will be performed via an admission controller plugin.
* In case requested size is smaller than current size of PVC. A validation will be used to reject the API request. (This could be moved to admission controller plugin too.)
* Not all PVCs will be resizable even if underlying volume plugin allows that. Only dynamically provisioned volumes
which are explicitly enabled by an admin will be allowed to be resized. A plugin in admission controller will forbid
size update for PVCs for which resizing is not enabled by the admin.
* The design proposal for raw block devices should make sure that, users aren't able to resize raw block devices.


### Controller Manager resize

A new controller called `volume_expand_controller` will listen for pvc size expansion requests and take action as needed. The steps performed in this
new controller will be:

* Watch for pvc update requests and add pvc to controller's work queue if a increase in volume size was requested. Once PVC is added to
  controller's work queue - `pvc.Status.Conditions` will be updated with `ResizeStarted: True`.
* For unbound or pending PVCs - resize will trigger no action in `volume_expand_controller`.
* If `pv.Spec.Capacity` already is of size greater or equal than requested size, similarly no action will be performed by the controller.
* A separate goroutine will read work queue and perform corresponding volume resize operation. If there is a resize operation in progress
  for same volume then resize request will be pending and retried once previous resize request has completed.
* Controller resize in effect will be level based rather than edge based. If there are more than one pending resize request for same PVC then
  new resize requests for same PVC will replace older pending request.
* Resize will be performed via volume plugin interface, executed inside a goroutine spawned by `operation_executor`.
* A new plugin interface called `volume.Expander` will be added to volume plugin interface. The `Expander` interface
  will also define if volume requires a file system resize:

  ```go
    type Expander interface {
      // ExpandVolume expands the volume
      ExpandVolumeDevice(spec *Spec, newSize resource.Quantity, oldSize resource.Quantity) error
      RequiresFSResize() bool
    }
  ```

* The controller call to expand the PVC will look like:

```go
func (og *operationGenerator) GenerateExpandVolumeFunc(
    pvcWithResizeRequest *expandcache.PvcWithResizeRequest,
    resizeMap expandcache.VolumeResizeMap) (func() error, error) {

    volumePlugin, err := og.volumePluginMgr.FindExpandablePluginBySpec(pvcWithResizeRequest.VolumeSpec)
    expanderPlugin, err := volumePlugin.NewExpander(pvcWithResizeRequest.VolumeSpec)


    expandFunc := func() error {
        expandErr := expanderPlugin.ExpandVolumeDevice(pvcWithResizeRequest.ExpectedSize, pvcWithResizeRequest.CurrentSize)

        if expandErr != nil {
            og.recorder.Eventf(pvcWithResizeRequest.PVC, v1.EventTypeWarning, kevents.VolumeResizeFailed, expandErr.Error())
            resizeMap.MarkResizeFailed(pvcWithResizeRequest, expandErr.Error())
            return expandErr
        }

        // CloudProvider resize succeeded - lets mark api objects as resized
        if expanderPlugin.RequiresFSResize() {
            err := resizeMap.MarkForFileSystemResize(pvcWithResizeRequest)
            if err != nil {
                og.recorder.Eventf(pvcWithResizeRequest.PVC, v1.EventTypeWarning, kevents.VolumeResizeFailed, err.Error())
                return err
            }
        } else {
            err := resizeMap.MarkAsResized(pvcWithResizeRequest)

            if err != nil {
                og.recorder.Eventf(pvcWithResizeRequest.PVC, v1.EventTypeWarning, kevents.VolumeResizeFailed, err.Error())
                return err
            }
        }
        return nil

    }
    return expandFunc, nil
}
```

* Once volume expand is successful, the volume will be marked as expanded and new size will be updated in `pv.spec.capacity`. Any errors will be reported as *events* on PVC object.
* If resize failed in above step, in addition to events - `pvc.Status.Conditions` will be updated with `ResizeFailed: True`. Corresponding error will be added to condition field as well.
* Depending on volume type next steps would be:

    * If volume is of type that does not require file system resize, then `pvc.status.capacity` will be immediately updated to reflect new size. This would conclude the volume expand operation. Also `pvc.Status.Conditions` will be updated with `Ready: True`.
    * If volume is of type that requires file system resize then a file system resize will be performed on kubelet. Read below for steps that will be performed for file system resize.

* If volume plugin is of type that can not do resizing of attached volumes (such as `Cinder`) then `ExpandVolumeDevice` can return error by checking for
  volume status with its own API (such as by making Openstack Cinder API call in this case). Controller will keep trying to resize the volume until it is
  successful.

* To consider cases of missed PVC update events, an additional loop will reconcile bound PVCs with PVs. This additional loop will loop through all PVCs
  and match `pvc.spec.resources.requests` with `pv.spec.capacity` and add PVC in `volume_expand_controller`'s work queue if `pv.spec.capacity` is less
  than `pvc.spec.resources.requests`.

* There will be additional checks in controller that grows PV size - to ensure that we do not make volume plugin API calls that can reduce size of PV.

### File system resize on kubelet

A File system resize will be pending on PVC until a new pod that uses this volume is scheduled somewhere. While theoretically we *can* perform
online file system resize if volume type and file system supports it - we are leaving it for next iteration of this feature.

#### Prerequisite of File system resize

* `pv.spec.capacity` must be greater than `pvc.status.spec.capacity`.
* A fix in pv_controller has to made to fix `claim.Status.Capacity` only during binding. See comment by jan here - https://github.com/kubernetes/community/pull/657#discussion_r128008128
* A fix in attach_detach controller has to be made to prevent fore detaching of volumes that are undergoing resize.
This can be done by checking `pvc.Status.Conditions` during force detach. `AttachedVolume` struct doesn't hold a reference to PVC - so PVC info can either be directly cached in `AttachedVolume` along with PV spec or it can be fetched from PersistentVolume's ClaimRef binding info.

#### Steps for resizing file system available on Volume

* When calling `MountDevice` or `Setup` call of volume plugin, volume manager will in addition compare `pv.spec.capacity` and `pvc.status.capacity` and if `pv.spec.capacity` is greater
  than `pvc.status.spec.capacity` then volume manager will additionally resize the file system of volume.
* The call to resize file system will be performed inside `operation_generator.GenerateMountVolumeFunc`.  `VolumeToMount` struct will be enhanced to store PVC as well.
* The flow of file system resize will be as follow:
    * Perform a resize based on file system used inside block device.
    * If resize succeeds, proceed with mounting the device as usual.
    * If resize failed with an error that shows no file system exists on the device, then log a warning and proceed with format and mount.
    * If resize failed with any other error then fail the mount operation.
* Any errors during file system resize will be added as *events* to Pod object and mount operation will be failed.
* If there are any errors during file system resize `pvc.Status.Conditions` will be updated with `ResizeFailed: True`. Any errors will be added to
  `Conditions` field.
* File System resize will not be performed on kubelet where volume being attached is ReadOnly. This is similar to pattern being used for performing formatting.
* After file system resize is successful, `pvc.status.capacity` will be updated to match `pv.spec.capacity` and volume expand operation will be considered complete. Also `pvc.Status.Conditions` will be updated with `Ready: True`.

#### Reduce coupling between resize operation and file system type

A file system resize in general requires presence of tools such as `resize2fs` or `xfs_growfs` on the host where kubelet is running. There is a concern
that open coding call to different resize tools directly in Kubernetes will result in coupling between file system and resize operation. To solve this problem
we have considered following options:

1. Write a library that abstracts away various file system operations, such as - resizing, formatting etc.

   Pros:
   * Relatively well known pattern

   Cons:
   * Depending on version with which Kubernetes is compiled with, we are still tied to which file systems are supported in which version
     of kubernetes.
2. Ship a wrapper shell script that encapsulates various file system operations and as long as the shell script supports particular file system
   the resize operation is supported.
   Pros:
   * Kubernetes Admin can easily replace default shell script with her own version and thereby adding support for more file system types.

   Cons:
   * I don't know if there is a pattern that exists in kube today for shipping shell scripts that are called out from code in Kubernetes. Flex is
     different because, none of the flex scripts are shipped with Kubernetes.
3. Ship resizing tools in a container.


Of all options - #3 is our best bet but we are not quite there yet. Hence, I would like to propose that we ship with support for
most common file systems in current release and we revisit this coupling and solve it in next release.

## API and UI Design

Given a PVC definition:

```yaml
kind: PersistentVolumeClaim
apiVersion: v1
metadata:
 name: volume-claim
 annotations:
   volume.beta.kubernetes.io/storage-class: "generalssd"
spec:
 accessModes:
   - ReadWriteOnce
 resources:
   requests:
     storage: 1Gi
```

Users can request new size of underlying PV by simply editing the PVC and requesting new size:

```
~> kubectl edit pvc volume-claim
kind: PersistentVolumeClaim
apiVersion: v1
metadata:
 name: volume-claim
 annotations:
   volume.beta.kubernetes.io/storage-class: "generalssd"
spec:
 accessModes:
   - ReadWriteOnce
 resources:
   requests:
     storage: 10Gi
```

## API Changes

### PVC API Change

`pvc.spec.resources.requests.storage` field of pvc object will become mutable after this change.

In addition to that PVC's status will have a `Conditions []PvcCondition` - which will be used
to communicate the status of PVC to the user.

The API change will be protected by Alpha feature gate and api-server will not allow PVCs with
`Status.Conditions` field if feature is not enabled. `omitempty` in serialization format will
prevent presence of field if not set.

So the `PersistentVolumeClaimStatus` will become:

```go
type PersistentVolumeClaimStatus struct {
    Phase PersistentVolumeClaimPhase
    AccessModes []PersistentVolumeAccessMode
    Capacity ResourceList
    // New Field added as part of this Change
    Conditions []PVCCondition
}

// new API type added
type PVCCondition struct {
   Type PVCConditionType
   Status ConditionStatus
   LastProbeTime metav1.Time
   LastTransitionTime metav1.Time
   Reason string
   Message string
}

// new API type
type PVCConditionType string

// new Constants
const (
   PVCReady PVCConditionType = "Ready"
   PVCResizeStarted PVCConditionType = "ResizeStarted"
   PVCResizeFailed  PVCResizeFailed = "ResizeFailed"
)
```

### StorageClass API change

A new field called `AllowVolumeExpand` will be added to StorageClass. The default of this value
will be `false` and only if it is true - PVC expansion will be allowed.

```go
type StorageClass struct {
    metav1.TypeMeta
    metav1.ObjectMeta
    Provisioner string
    Parameters map[string]string
    // New Field added
    // +optional
    AllowVolumeExpand bool
}
```

### Other API changes

This proposal relies on ability to update PVC status from kubelet. While updating PVC's status
a PATCH request must be made from kubelet to update the status.

## Production Readiness Review Questionnaire

### Feature Enablement and Rollback

###### How can this feature be enabled / disabled in a live cluster?

Volume expansion as a feature has been in beta for too long and as a result has gathered
different feature gates that control various aspects of expansion.

- [x] Feature gate (also fill in values in `kep.yaml`)
  - Feature gate name: ExpandPersistentVolumes
    - description: |
        This feature is required for `pvc.Spec.Resources` to be editable and must be
        enabled for other expansion related feature gates to work.
    - Components depending on the feature gate:
        - kube-apiserver
        - kubelet
        - kube-controller-manager
  - Feature gate name: ExpandInUsePersistentVolumes
    - description: Enables online expansion. Requires ExpandPersistentVolumes feature gate.
    - Components depending on the feature gate:
        - kube-apiserver
        - kubelet
        - kube-controller-manager
  - Feature gate name: ExpandCSIVolumes
    - description: Enables CSI expansion.
    - Components depending on the feature gate:
        - kube-apiserver
        - kubelet
        - kube-controller-manager
- [ ] Other
  - Describe the mechanism:
  - Will enabling / disabling the feature require downtime of the control
    plane?
    Enabling/Disabling this feature does not require complete downtime of control-plane
    and feature gates can be enabled progressively on different control-plane nodes.
  - Will enabling / disabling the feature require downtime or reprovisioning
    of a node? (Do not assume `Dynamic Kubelet Config` feature is enabled).
    Enabling this feature can be enabled progressively on nodes and as expansion is enabled
    on the node then volume expansion will happen on kubelet.

###### Does enabling the feature change any default behavior?

Enabling the feature gate allows users to increase size of pvc by editing `pvc.Spec.Resources` which results
in Kubernetes trying to fulfill the request by actually expanding the volume in controller and then performing
file system or any other kind of expansion needed on the node.

###### Can the feature be disabled once it has been enabled (i.e. can we roll back the enablement)?

Yes - it can be disabled. It just means users can no longer expand their PVCs.

###### What happens if we reenable the feature if it was previously rolled back?

It should be safe to do that. It will just re-enable the feature.

###### Are there any tests for feature enablement/disablement?

There aren't any e2e but there are unit tests that cover this behaviour.

### Rollout, Upgrade and Rollback Planning

###### How can a rollout or rollback fail? Can it impact already running workloads?

The feature gate should not impact existing workloads but since we try to expand the
file system(or perform node-expansion) during volume mount and if expansion fails with
some kind of terminal error then it may prevent mount operation from succeeding.

###### What specific metrics should inform a rollback?

The `volume_mount` operation failure metric - `storage_operation_duration_seconds{operation_name=volume_mount, status=fail-unknown}`
combined with `storage_operation_duration_seconds{operation_name=volume_fs_resize, status=fail-unknown}` should tell us
if expansion is failing on the node and if it is causing mount failures.

Also `csi_sidecar_operations_seconds` and `csi_operations_seconds` metrics with high failure rates for expansion operation should indicate
that expansion is not working in the cluster and hence feature should be rolled back.

###### Were upgrade and rollback tested? Was the upgrade->downgrade->upgrade path tested?

There are no e2e for upgrade-downgrade-upgrade tests for this specific feature but since volume expansion has been
in beta since 1.11, we have tested the feature manually.

###### Is the rollout accompanied by any deprecations and/or removals of features, APIs, fields of API types, flags, etc.?

This feature does not deprecate any existing features.

### Monitoring Requirements

###### How can an operator determine if the feature is in use by workloads?

A PVC that is being expanded should have `pvc.Status.Conditions` set.

###### How can someone using this feature know that it is working for their instance?

- [x] Events
  - Resizing (on PVC)
    - Event Reason: External resizer is resizing volume pvc-a71483ed-a5bc-48fa-9151-ca41e7e6634e
  - VolumeResizeSuccessful (on PVC)
    - Event Reason: Volume resize is successful
  - FileSystemResizeSuccessful (on PVC)
    - Event Reason: Volume resize is successful. This event is emitted when resizing finishes on kubelet.
- [x API .status
  - Condition name:
  - Other field:
- [x] Other (treat as last resort)
  - Details: `pvc.Status.Capacity` should reflect user requested size after expansion is complete.

###### What are the reasonable SLOs (Service Level Objectives) for the enhancement?

Enabling this feature should not negatively impact volume mount timings in general cases and hence percentile determined by `storage_operation_duration_seconds{operation_name=volume_mount}` metric should not change.

Having said that if file system requires expansion during mount then it is obviously going to take longer for mount operation to finish.

###### What are the SLIs (Service Level Indicators) an operator can use to determine the health of the service?

- [ ] Metrics
  - controller expansion operation duration:
    - Metric name: storage_operation_duration_seconds{operation_name=expand_volume, status=success|fail-unknown}
    - [Optional] Aggregation method: percentile
    - Components exposing the metric: kube-controller-manager
  - node expansion operation duration:
    - Metric name: storage_operation_duration_seconds{operation_name=volume_fs_resize, status=success|fail-unknown}
    - [Optional] Aggregation method: percentile
    - Components exposing the metric: kubelet
  - CSI operation metrics in controller:
    - Metric name: csi_sidecar_operations_seconds
    - [Optional] Aggregation method: percentile
    - Components exposing the metric: external-resizer
  - CSI operation metrics in kubelet:
    - Metric Name: csi_operations_seconds
    - [Optional] Aggregation method: percentile
    - Components exposing the metric: kubelet

- [ ] Other (treat as last resort)
  - Details:

###### Are there any missing metrics that would be useful to have to improve observability of this feature?
    All the intree operations from control plane emit `storage_operation_duration_seconds{operation_name=expand_volume, status=success|fail-unknown}` metrics but CSI equivalent from external-resizer is `csi_sidecar_operations_seconds` which will be
    documented as alternative if CSI migration is enabled or driver being used is CSI driver.
    We don't need to emit new metrics but we do need to document the naming change in metric names.

### Dependencies

<!--
This section must be completed when targeting beta to a release.
-->

###### Does this feature depend on any specific services running in the cluster?

This feature requires external-resizer running in the cluster for CSI volume expansion.

### Scalability

###### Will enabling / using this feature result in any new API calls?

Yes enabling this feature requires new API calls.

- Updates to PVs
  - API operations
    - PATCH PV
    - GET PV
    - List PVs
  - originating components: kubelet, kube-controller-manager, external-resizer
  - resync duration: 10mins (also user configurable)
- Update to PVCs:
  - API operations
    - PATCH PVC
    - GET PVC
    - List PVC
  - originating components: kubelet, kube-controller-manager, external-resizer
  - resync duration: 10mins (also user configurable)

If user enables protection for not expanding PVCs that are in-use, external-resizer will
also watch *all* pods in the cluster. This is an optional flag in external-resizer and generally
only needed when some CSI drivers don't want to handle expansion calls for volumes which are potentially in-use by a pod.

###### Will enabling / using this feature result in introducing new API types?

No

###### Will enabling / using this feature result in any new calls to the cloud provider?

Yes, we expect new calls to modify existing volume objects.

###### Will enabling / using this feature result in increasing size or count of the existing API objects?

Describe them, providing:
  - API type(s): PVC
    - Estimated increase in size: A PVC with conditions could have its size increased by anywhere between 100 to 250B.
    - Estimated amount of new objects: (e.g., new Object X for every existing Pod)
  - API type(s): StorageClass
    - Estimated increase in size: A StorageClass with `AllowVolumeExpansion` has its size increased by 26bytes almost.
    - Estimated amount of new objects: (e.g., new Object X for every existing Pod)

###### Will enabling / using this feature result in increasing time taken by any operations covered by existing SLIs/SLOs?

If expansion happens because of pending file system during mount, then it would increase mount time of volume.

###### Will enabling / using this feature result in non-negligible increase of resource usage (CPU, RAM, disk, IO, ...) in any components?

Enabling this feature should not result in resource usage by significant margin, but since we are talking about new controller and an external resize controller for CSI, the resource usage is not negligible either. Having said that - this feature has been in beta since 1.11 and enabled by default(and used in production) - we do not expect resource usage to be a problem.

### Troubleshooting

###### How does this feature react if the API server and/or etcd is unavailable?
    Since this feature is user driven and API server or etcd becomes unavailable then users won't be able to expand the PVC.
    But if API server becomes unavailable midway through the expansion process then the expansion controller may not be able
    save updated PVC in api-server but control-flow is designed to retry and recover from such failures.

###### What are other known failure modes?

  - Expansion can be permanently stuck:
    - Detection: Check conditions on `pvc.status`
    - Mitigations: If expansion is stuck permanently because of issues in backend and can not be recovered then, it requires manual intervention. Steps to recover from expansion failures are documented in - https://kubernetes.io/docs/concepts/storage/persistent-volumes/#recovering-from-failure-when-expanding-volumes
    - Diagnostics: Conditions on `pvc.Status` and events on PVC should clearly indicate that expansion is failing.
    - Testing: There are some unit tests for failure mode but no e2e.


###### What steps should be taken if SLOs are not being met to determine the problem?

If expansion is affecting pod startup time or causing other issues. It can be disabled by editing storageclass and setting `allowVolumeExpansion` to `false`.

## Implementation History

- 1.8: Alpha
- 1.11: Beta
- 1.24 GA

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
