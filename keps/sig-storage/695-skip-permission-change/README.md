# Skip Volume Ownership Change

## Table of Contents

<!-- toc -->
- [Release Signoff Checklist](#release-signoff-checklist)
- [Summary](#summary)
- [Motivation](#motivation)
  - [Goals](#goals)
  - [Non-Goals](#non-goals)
- [Proposal](#proposal)
  - [Implementation Details/Notes/Constraints [optional]](#implementation-detailsnotesconstraints-optional)
  - [Risks and Mitigations](#risks-and-mitigations)
- [Production Readiness Review Questionnaire](#production-readiness-review-questionnaire)
  - [Feature enablement and rollback](#feature-enablement-and-rollback)
  - [Rollout, Upgrade and Rollback Planning](#rollout-upgrade-and-rollback-planning)
  - [Monitoring requirements](#monitoring-requirements)
  - [Dependencies](#dependencies)
  - [Scalability](#scalability)
  - [Troubleshooting](#troubleshooting)
- [Graduation Criteria](#graduation-criteria)
  - [Test Plan](#test-plan)
  - [Monitoring](#monitoring)
- [Implementation History](#implementation-history)
- [Drawbacks [optional]](#drawbacks-optional)
- [Alternatives [optional]](#alternatives-optional)
- [Infrastructure Needed [optional]](#infrastructure-needed-optional)
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

## Summary

Currently before a volume is bind-mounted inside a container the permissions on
that volume are changed recursively to the provided fsGroup value.  This change
in ownership can take an excessively long time to complete, especially for very
large volumes (>=1TB) as well as a few other reasons detailed in [Motivation](#motivation).

To solve this issue we will add a new field called `pvc.Status.FSGroup` to record last known
fsGroup of the volume managed by PVC and if it matches with `fsGroup` requested in pod's spec - kubelet
will not perform recursive permission and ownership change.

## Motivation

When a volume is mounted on the node, we recursively change permissions of volume
before bind mounting the volume inside container. The reason of doing this is to ensure
that volumes are readable/writable by provided fsGroup.

But this presents following problems:
 - An application(many popular databases) which is sensitive to permission bits changing
   underneath may refuse to start whenever volume being used inside pod gets mounted on
   different node.
 - If volume has a large number of files, performing recursive `chown` and `chmod`
   could be slow and could cause timeout while starting the pod.

### Goals

 - Allow volume ownership and permission change to be skipped during mount

### Non-Goals

 - In some cases if user brings in a large enough volume from outside, the first time ownership and permission change still could take lot of time.
 - On SELinux enabled distributions we will still do recursive chcon whenever applicable and handling that is outside the scope.
 - This proposal does not attempt to fix two pods using same volume with conflicting fsgroup. It also will be only applicable to volume types which support setting fsgroup.

## Proposal

### Implementation Details/Notes/Constraints [optional]


#### Proposed API changes:

##### Changes to PVC:


```go
type PersistentVolumeClaimStatus struct {
     // Phase represents the current phase of PersistentVolumeClaim
     // +optional
     Phase PersistentVolumeClaimPhase
     // AccessModes contains all ways the volume backing the PVC can be mounted
     // +optional
     AccessModes []PersistentVolumeAccessMode
     // Represents the actual resources of the underlying volume
     // +optional
     Capacity ResourceList
     // FSGroup of PVC
     // + optional
     FSGroup *int64     // <------ NEW ------
     // +optional
     Conditions []PersistentVolumeClaimCondition
}
```

##### Changes to CSIDriver

```go
const (
    // ReadWriteOnceWithFSTypeFSGroupPolicy indicates that each volume will be examined
    // to determine if the volume ownership and permissions
    // should be modified. If a fstype is defined and the volume's access mode
    // contains ReadWriteOnce, then the defined fsGroup will be applied.
    // This mode should be defined if it's expected that the
    // fsGroup may need to be modified depending on the pod's SecurityPolicy.
    // This is the default behavior if no other FSGroupPolicy is defined.
    ReadWriteOnceWithFSTypeFSGroupPolicy FSGroupPolicy = "ReadWriteOnceWithFSType"

    // FileFSGroupPolicy indicates that CSI driver supports volume ownership
    // and permission change via fsGroup, and Kubernetes may use fsGroup
    // to change permissions and ownership of the volume to match user requested fsGroup in
    // the pod's SecurityPolicy regardless of fstype or access mode.
    // This mode should be defined if the fsGroup is expected to always change on mount
    FileFSGroupPolicy FSGroupPolicy = "File"

    // OnMountFSGroupPolicy indicates that CSI driver supports changing volume ownership via
    // mount flags and hence fsgroup of pod should be made available to CSI driver in nodePublish
    // and nodeStage CSI RPC calls. fsGroup can be made available via volume attributes of the form:
    //   - csi.storage.k8s.io/pod.fsGroup: 1234
    OnMountFSGroupPolicy FSGroupPolicy = "Mount" <--- new change

    // NoneFSGroupPolicy indicates that volumes will be mounted without performing
    // any ownership or permission modifications, as the CSIDriver does not support
    // these operations.
    // This mode should be selected if the CSIDriver does not support fsGroup modifications,
    // for example when Kubernetes cannot change ownership and permissions on a volume due
    // to root-squash settings on a NFS volume.
    NoneFSGroupPolicy FSGroupPolicy = "None"
)
```

#### Implementation details

For all in-tree plugins when a volume is mounted by the kubelet, the plugin in kubelet will check
if `pvc.Status.FSGroup` match `pod.Spec.SecurityContext.FSGroup` then no recursive permission
change will be performed.

For CSI drivers:

- Don't do any permission and ownership change if `CSIDriver.Spec.FSGroupPolicy` is `None.`
- If `CSIDriver.Spec.FSGroupPolicy` is set to `Mount` then pod's fsGroup will be supplied to the
  CSI driver via volume attributes on nodeStage/nodePublish and it is expected that driver
  will set right permissions on the volume during mount.
- If `CSIDriver.Spec.FSGroupPolicy` is set to `File|ReadWriteOnceWithFSType` and `pvc.Status.FSGroup`
  doesn't match with `pod.Spec.SecurityContext.FSGroup` then a recursive ownership and permission change
  on volume will be performed by kubelet.

After ownership and permissions are recursively changed, pvc.Status.FSGroup will be updated to reflect
latest value.

### Risks and Mitigations

- If somehow users have volumes that have right `fsGroup` recorded in PVC's status but actual permissions on disk is different, the volume
  could become unreadable/unwritable to the pod. This could happen if files inside volume dwerewas changed outside Kubernetes by a different process.
  User/admin can fix the problem by manually changing permission and ownership of volume by same external process that modifies the volume.
- If a process running inside pod removes group's permission from certain subdirectories or files and UID of the pod changes, then those subdirectories
  and files will not be readable/writable by the new pod. To mitigate this - if UID if pod changes then user must change `fsGroup` of the pod too.

## Production Readiness Review Questionnaire

### Feature enablement and rollback

* **How can this feature be enabled / disabled in a live cluster?**
  - [x] Feature gate
    - Feature gate name: SkipFSGroupChange
    - Components depending on the feature gate: kubelet, kube-apiserver

* **Does enabling the feature change any default behavior?**
  Yes enabling the feature gate could skip permission change for PVCs for which recorded `fsGroup` in `pvc.Status` is already correct.

* **Can the feature be disabled once it has been enabled (i.e. can we rollback
  the enablement)?**
  Yes the feature-gate can be disabled once enabled. This will cause all volume mounts that
  require permission change to use recursive permission and ownership change. This is the default
  behaviour without the feature gate.

* **What happens if we reenable the feature if it was previously rolled back?**
  If we reenable feature-gate then for PVCs for which `fsGroup` was already recorded in `pvc.Status` will have recursive permission
  change skipped if it matches requestes `fsGroup` in pod.

* **Are there any tests for feature enablement/disablement?**
  There aren't any e2e but there are unit tests that cover this behaviour.

### Rollout, Upgrade and Rollback Planning

* **How can a rollout fail? Can it impact already running workloads?**
  If somehow users have volumes that have right `fsGroup` recorded in PVC but actual permissions on disk is different, the volume
  could become unreadable/unwritable to the pod.


* **What specific metrics should inform a rollback?**
  If after enabling this feature users notice an increase in volume mount time via `storage_operation_duration_seconds{operation_name=volume_mount}`
  or an increase in mount error count via `storage_operation_errors_total{operation_name=volume_mount}`
  then a cluster-admin may want to rollback the feature.

* **Were upgrade and rollback tested? Was upgrade->downgrade->upgrade path tested?**
  We have not fully tested upgrade and rollback. We have unit tests that cover the scenario
  of feature gate being enabled and then disabled. But we will need to do more upgrade->downgrade->upgrade testing.

* **Is the rollout accompanied by any deprecations and/or removals of features,
  APIs, fields of API types, flags, etc.?**
  This feature deprecates no existing functionality.


### Monitoring requirements

* **How can an operator determine if the feature is in use by workloads?**
  Operator can query `pvc.Status.FSGroup` field and identify if this is being set to non-default values.

* **What are the SLIs (Service Level Indicators) an operator can use to
  determine the health of the service?**
  - [x] Metrics
    - mount operation duration:
        - Metric name: storage_operation_duration_seconds{operation_name=volume_mount}
        - [Optional] Aggregation method: percentile
        - Components exposing the metric: kubelet
    - mount operation errors:
        - Metric name: storage_operation_errors_total{operation_name=volume_mount}
        - [Optional] Aggregation method: cumulative counter
        - Components exposing the metric: kubelet
    - volume ownerhip change timing mtrics: We are also going to add metrics that track time it takes for volume ownerhip change to happen. We will update this section with the name of metrics.



* **What are the reasonable SLOs (Service Level Objectives) for the above SLIs?**
  It is hard to give numbers that an admin could use to determine health of mount operation. In general we expect that after this feature is rolled out
  for volumes managed by PVC `storage_operation_duration_seconds{operation_name=volume_mount}`
  should go report lower values and there should not be an spike in mount error metric (reported via `storage_operation_errors_total{operation_name=volume_mount}`).

* **Are there any missing metrics that would be useful to have to improve
  observability if this feature?**
  As documented above we already have metrics for tracking mount timing. We are planning to add a metric
  for time it takes to change permission of a volume before mount but this is not necessary for observability of this feature.

### Dependencies

* **Does this feature depend on any specific services running in the cluster?**
  Not applicable

### Scalability

* **Will enabling / using this feature result in any new API calls?**
  This feature could trigger additional api call from kubelet to update PVC status after permission/ownership change is complete.

* **Will enabling / using this feature result in introducing new API types?**
  This feature introduces no new API types.

* **Will enabling / using this feature result in any new calls to cloud
  provider?**
  No. This feature has no cloud-provider integration.

* **Will enabling / using this feature result in increasing size or count
  of the existing API objects?**
  Since this feature adds a new field to pvc status, it will increase API size of
  PVC object:
  Describe them providing:
  - API type(s): PVC
  - Estimated increase in size: (e.g. new annotation of size 32B): 7B


* **Will enabling / using this feature result in increasing time taken by any
  operations covered by [existing SLIs/SLOs][]?**
  No. If anything this feature will reduce time it takes for a Pod to start.

* **Will enabling / using this feature result in non-negligible increase of
  resource usage (CPU, RAM, disk, IO, ...) in any components?**
  No. This feature should not cause any increase in memory or CPU usage of the affected component.

### Troubleshooting

Troubleshooting section serves the `Playbook` role as of now. We may consider
splitting it into a dedicated `Playbook` document (potentially with some monitoring
details). For now we leave it here though.

* **How does this feature react if the API server and/or etcd is unavailable?**
  Not applicable

* **What are other known failure modes?**
  - A volume managed by a PVC has different permissions than the one recorded in `pvc.Status.FSGroup`:
    - Detection: If user modifies a volume managed by PVC outside Kubernetes, it can result in different permissions on the disk than what is recorded in `pvc.Status.FSGroup` and if that happens pod may not run correctly.
    - Mitigations: If anything causes creation or modifications of files without group permissions on volume - then user may recreate the pod with different `fsGroup` to force recursive permission and ownership change.
    - Diagnostics: Pod does not run correctly.
    - Testing: If pod can not run correctly after modifying files on volume then `fsGroup` on pod has to be reset.

* **What steps should be taken if SLOs are not being met to determine the problem?**
  If admin notices an increase in mount errors or increase in mount timings as documented in SLIs then an admin could:
      - check number of PVCs that are have `pvc.Status.FSGroup` set in them.
      - Check volume mount and latency metrics (as described in SLI)
      - Check kubelet logs for mount errors or problems.

[supported limits]: https://git.k8s.io/community//sig-scalability/configs-and-limits/thresholds.md
[existing SLIs/SLOs]: https://git.k8s.io/community/sig-scalability/slos/slos.md#kubernetes-slisslos

## Graduation Criteria

* Alpha in 1.20 provided all tests are passing and gated by the feature Gate
   `SkipFSGroupChange` and set to a default of `false`.

* Beta in 1.21 with design validated by at least two customer deployments
  (non-production), with discussions in SIG-Storage regarding success of
  deployments.  A metric will be added to report time taken to perform a
  volume ownership change. Also e2e tests that verify volume permissions
  when PVC is reused by different pods.
* GA in 1.20, with Node E2E tests in place tagged with feature Storage


[umbrella issues]: https://github.com/kubernetes/kubernetes/issues/69699

### Test Plan

A test plan will consist of the following tests

* Basic tests including a permutation of the following values
  - PersistentVolumeClaimStatus.FSGroup (matching, non-matching)
  - Volume Filesystem existing permissions (none, matching, non-matching, partial-matching?)
* E2E tests


### Monitoring

We will add a metric that measures the volume ownership change times.

## Implementation History

- 2020-01-20 Initial KEP pull request submitted
- 2020-09-16 KEP updated to use `pvc.Status.FSGroup`

## Drawbacks [optional]


## Alternatives [optional]

We considered various alternatives before proposing changes mentioned in this proposal.
- We considered using a shiftfs(https://github.com/linuxkit/linuxkit/tree/master/projects/shiftfs) like solution for mounting volumes inside containers without changing permissions on the host. But any such solution is technically not feasible until support in Linux kernel improves.
- We also considered redesigning volume permission API to better support different volume types and different operating systems because `fsGroup` is somewhat Linux specific. But any such redesign has to be backward compatible and given lack of clarity about how the new API should look like, we aren't quite ready to do that yet.

## Infrastructure Needed [optional]
