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

<!--

Production readiness reviews are intended to ensure that features merging into
Kubernetes are observable, scalable and supportable; can be safely operated in
production environments, and can be disabled or rolled back in the event they
cause increased failures in production. See more in the PRR KEP at
https://git.k8s.io/enhancements/keps/sig-architecture/1194-prod-readiness.

The production readiness review questionnaire must be completed and approved
for the KEP to move to `implementable` status and be included in the release.

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

<!--
This section must be completed when targeting alpha to a release.
-->

###### How can this feature be enabled / disabled in a live cluster?

<!--
Pick one of these and delete the rest.
-->

- [ ] Feature gate (also fill in values in `kep.yaml`)
  - Feature gate name:
  - Components depending on the feature gate:
- [ ] Other
  - Describe the mechanism:
  - Will enabling / disabling the feature require downtime of the control
    plane?
  - Will enabling / disabling the feature require downtime or reprovisioning
    of a node? (Do not assume `Dynamic Kubelet Config` feature is enabled).

###### Does enabling the feature change any default behavior?

<!--
Any change of default behavior may be surprising to users or break existing
automations, so be extremely careful here.
-->

###### Can the feature be disabled once it has been enabled (i.e. can we roll back the enablement)?

<!--
Describe the consequences on existing workloads (e.g., if this is a runtime
feature, can it break the existing applications?).

NOTE: Also set `disable-supported` to `true` or `false` in `kep.yaml`.
-->

###### What happens if we reenable the feature if it was previously rolled back?

###### Are there any tests for feature enablement/disablement?

<!--
The e2e framework does not currently support enabling or disabling feature
gates. However, unit tests in each component dealing with managing data, created
with and without the feature, are necessary. At the very least, think about
conversion tests if API types are being modified.
-->

### Rollout, Upgrade and Rollback Planning

<!--
This section must be completed when targeting beta to a release.
-->

###### How can a rollout or rollback fail? Can it impact already running workloads?

<!--
Try to be as paranoid as possible - e.g., what if some components will restart
mid-rollout?

Be sure to consider highly-available clusters, where, for example,
feature flags will be enabled on some API servers and not others during the
rollout. Similarly, consider large clusters and how enablement/disablement
will rollout across nodes.
-->

###### What specific metrics should inform a rollback?

<!--
What signals should users be paying attention to when the feature is young
that might indicate a serious problem?
-->

###### Were upgrade and rollback tested? Was the upgrade->downgrade->upgrade path tested?

<!--
Describe manual testing that was done and the outcomes.
Longer term, we may want to require automated upgrade/rollback tests, but we
are missing a bunch of machinery and tooling and can't do that now.
-->

###### Is the rollout accompanied by any deprecations and/or removals of features, APIs, fields of API types, flags, etc.?

<!--
Even if applying deprecation policies, they may still surprise some users.
-->

### Monitoring Requirements

<!--
This section must be completed when targeting beta to a release.
-->

###### How can an operator determine if the feature is in use by workloads?

<!--
Ideally, this should be a metric. Operations against the Kubernetes API (e.g.,
checking if there are objects with field X set) may be a last resort. Avoid
logs or events for this purpose.
-->

###### How can someone using this feature know that it is working for their instance?

<!--
For instance, if this is a pod-related feature, it should be possible to determine if the feature is functioning properly
for each individual pod.
Pick one more of these and delete the rest.
Please describe all items visible to end users below with sufficient detail so that they can verify correct enablement
and operation of this feature.
Recall that end users cannot usually observe component logs or access metrics.
-->

- [ ] Events
  - Event Reason:
- [ ] API .status
  - Condition name:
  - Other field:
- [ ] Other (treat as last resort)
  - Details:

###### What are the reasonable SLOs (Service Level Objectives) for the enhancement?

<!--
This is your opportunity to define what "normal" quality of service looks like
for a feature.

It's impossible to provide comprehensive guidance, but at the very
high level (needs more precise definitions) those may be things like:
  - per-day percentage of API calls finishing with 5XX errors <= 1%
  - 99% percentile over day of absolute value from (job creation time minus expected
    job creation time) for cron job <= 10%
  - 99.9% of /health requests per day finish with 200 code

These goals will help you determine what you need to measure (SLIs) in the next
question.
-->

###### What are the SLIs (Service Level Indicators) an operator can use to determine the health of the service?

<!--
Pick one more of these and delete the rest.
-->

- [ ] Metrics
  - Metric name:
  - [Optional] Aggregation method:
  - Components exposing the metric:
- [ ] Other (treat as last resort)
  - Details:

###### Are there any missing metrics that would be useful to have to improve observability of this feature?

<!--
Describe the metrics themselves and the reasons why they weren't added (e.g., cost,
implementation difficulties, etc.).
-->

### Dependencies

<!--
This section must be completed when targeting beta to a release.
-->

###### Does this feature depend on any specific services running in the cluster?

<!--
Think about both cluster-level services (e.g. metrics-server) as well
as node-level agents (e.g. specific version of CRI). Focus on external or
optional services that are needed. For example, if this feature depends on
a cloud provider API, or upon an external software-defined storage or network
control plane.

For each of these, fill in the following—thinking about running existing user workloads
and creating new ones, as well as about cluster-level services (e.g. DNS):
  - [Dependency name]
    - Usage description:
      - Impact of its outage on the feature:
      - Impact of its degraded performance or high-error rates on the feature:
-->

### Scalability

<!--
For alpha, this section is encouraged: reviewers should consider these questions
and attempt to answer them.

For beta, this section is required: reviewers must answer these questions.

For GA, this section is required: approvers should be able to confirm the
previous answers based on experience in the field.
-->

###### Will enabling / using this feature result in any new API calls?

<!--
Describe them, providing:
  - API call type (e.g. PATCH pods)
  - estimated throughput
  - originating component(s) (e.g. Kubelet, Feature-X-controller)
Focusing mostly on:
  - components listing and/or watching resources they didn't before
  - API calls that may be triggered by changes of some Kubernetes resources
    (e.g. update of object X triggers new updates of object Y)
  - periodic API calls to reconcile state (e.g. periodic fetching state,
    heartbeats, leader election, etc.)
-->

###### Will enabling / using this feature result in introducing new API types?

<!--
Describe them, providing:
  - API type
  - Supported number of objects per cluster
  - Supported number of objects per namespace (for namespace-scoped objects)
-->

###### Will enabling / using this feature result in any new calls to the cloud provider?

<!--
Describe them, providing:
  - Which API(s):
  - Estimated increase:
-->

###### Will enabling / using this feature result in increasing size or count of the existing API objects?

<!--
Describe them, providing:
  - API type(s):
  - Estimated increase in size: (e.g., new annotation of size 32B)
  - Estimated amount of new objects: (e.g., new Object X for every existing Pod)
-->

###### Will enabling / using this feature result in increasing time taken by any operations covered by existing SLIs/SLOs?

<!--
Look at the [existing SLIs/SLOs].

Think about adding additional work or introducing new steps in between
(e.g. need to do X to start a container), etc. Please describe the details.

[existing SLIs/SLOs]: https://git.k8s.io/community/sig-scalability/slos/slos.md#kubernetes-slisslos
-->

###### Will enabling / using this feature result in non-negligible increase of resource usage (CPU, RAM, disk, IO, ...) in any components?

<!--
Things to keep in mind include: additional in-memory state, additional
non-trivial computations, excessive access to disks (including increased log
volume), significant amount of data sent and/or received over network, etc.
This through this both in small and large cases, again with respect to the
[supported limits].

[supported limits]: https://git.k8s.io/community//sig-scalability/configs-and-limits/thresholds.md
-->

### Troubleshooting

<!--
This section must be completed when targeting beta to a release.

The Troubleshooting section currently serves the `Playbook` role. We may consider
splitting it into a dedicated `Playbook` document (potentially with some monitoring
details). For now, we leave it here.
-->

###### How does this feature react if the API server and/or etcd is unavailable?

###### What are other known failure modes?

<!--
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
-->

###### What steps should be taken if SLOs are not being met to determine the problem?

## Implementation History

<!--
Major milestones in the lifecycle of a KEP should be tracked in this section.
Major milestones might include:
- the `Summary` and `Motivation` sections being merged, signaling SIG acceptance
- the `Proposal` section being merged, signaling agreement on a proposed design
- the date implementation started
- the first Kubernetes release where an initial version of the KEP was available
- the version of Kubernetes where the KEP graduated to general availability
- when the KEP was retired or superseded
-->

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
