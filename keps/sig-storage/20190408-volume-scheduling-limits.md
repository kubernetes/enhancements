---
title: KEP Template
authors:
  - "@jsafrane"
  - "@gnufied"
owning-sig: sig-storage
participating-sigs:
  - sig-scheduling
reviewers:
  - "@bsalamat"
  - "@thockin"
  - "@msau42"
approvers:
  - "@bsalamat"
  - "@msau42"
  - "@thockin"
editor: TBD
creation-date: 2019-04-08
last-updated: 2019-04-08
status: implemented
see-also:
  - "https://github.com/kubernetes/enhancements/blob/master/keps/sig-storage/20190129-csi-migration.md"
replaces:
superseded-by:
---

# Volume Scheduling Limits

## Table of Contents

<!-- toc -->
  - [Release Signoff Checklist](#release-signoff-checklist)
  - [Summary](#summary)
  - [Motivation](#motivation)
    - [Goals](#goals)
    - [Non-Goals](#non-goals)
  - [Proposal](#proposal)
    - [API Change](#api-change)
    - [Implementation](#implementation)
      - [Implementation Detail for all CSI Drivers](#implementation-detail-for-all-csi-drivers)
      - [Implementation detail for in-tree Drivers with CSI migration disabled](#implementation-detail-for-in-tree-drivers-with-csi-migration-disabled)
        - [When no CSI driver for same underlying storage type is installed on the node.](#when-no-csi-driver-for-same-underlying-storage-type-is-installed-on-the-node)
        - [When CSI driver for same underlying storage type is installed on the node.](#when-csi-driver-for-same-underlying-storage-type-is-installed-on-the-node)
      - [Implementation detail for in-tree Drivers with CSI migration enabled](#implementation-detail-for-in-tree-drivers-with-csi-migration-enabled)
    - [User Stories](#user-stories)
    - [Implementation Details/Notes/Constraints](#implementation-detailsnotesconstraints)
    - [Risks and Mitigations](#risks-and-mitigations)
  - [Design Details](#design-details)
    - [Test Plan](#test-plan)
    - [Graduation Criteria](#graduation-criteria)
        - [Alpha -&gt; Beta Graduation](#alpha---beta-graduation)
        - [Beta -&gt; GA Graduation](#beta---ga-graduation)
    - [Upgrade / Downgrade / Version Skew Strategy](#upgrade--downgrade--version-skew-strategy)
      - [Interaction with old <code>AttachVolumeLimit</code> implementation](#interaction-with-old--implementation)
  - [Implementation History](#implementation-history)
- [Alternatives](#alternatives)
<!-- /toc -->

## Release Signoff Checklist

- [ ] kubernetes/enhancements issue in release milestone, which links to KEP (this should be a link to the KEP location in kubernetes/enhancements, not the initial KEP PR)
- [ ] KEP approvers have set the KEP status to `implementable`
- [ ] Design details are appropriately documented
- [ ] Test plan is in place, giving consideration to SIG Architecture and SIG Testing input
- [ ] Graduation criteria is in place
- [ ] "Implementation History" section is up-to-date for milestone
- [ ] User-facing documentation has been created in [kubernetes/website], for publication to [kubernetes.io]
- [ ] Supporting documentation e.g., additional design documents, links to mailing list discussions/SIG meetings, relevant PRs/issues, release notes

**Note:** Any PRs to move a KEP to `implementable` or significant changes once it is marked `implementable` should be approved by each of the KEP approvers. If any of those approvers is no longer appropriate than changes to that list should be approved by the remaining approvers and/or the owning SIG (or SIG-arch for cross cutting KEPs).

## Summary

Number of volumes of certain type that can be attached to a node should be configurable easily and should be based on node type. This proposal implements dynamic attachable volume limits on a per-node basis rather than cluster global defaults that exist today. This proposal also implements a way of configuring volume limits for CSI volumes.

This proposal replaces [#730](https://github.com/kubernetes/enhancements/pull/730) and integrates volume limits for in-tree volumes (AWS EBS, GCE PD, AZURE DD, OpenStack Cinder) and CSI into one predicate. As result, in-tree volumes and corresponding CSI driver can share the same volume limit.

## Motivation

Current scheduler predicates for scheduling of pods with volumes is based on `node.status.capacity` and `node.status.allocatable`. It works well for hardcoded predicates for volume limits on AWS (`MaxEBSVolumeCount`), GCE(`MaxGCEPDVolumeCount`), Azure (`MaxAzureDiskVolumeCount`) and OpenStack (`MaxCinderVolumeCount`).

It is problematic for CSI (`MaxCSIVolumeCountPred`) outlined in [#730](https://github.com/kubernetes/enhancements/pull/730)

- `ResourceName` is limited to 63 characters. We must prefix `ResourceName` with unique string (such as `attachable-volumes-csi-<driver name>`) so it cannot collide with existing resources like `cpu` or `memory`. But `<driver name>` itself is up to 63 character long, so we ended up with using SHA-sums of driver name to keep the `ResourceName` unique, which is not user readable.
- CSI driver cannot share its limits with in-tree volume plugin e.g. when running pods with AWS EBS in-tree volumes and `ebs.csi.aws.com` CSI driver on the same node.

### Goals

- When CSI Driver is installed on the node, for in-tree drivers which are being considered for migration to CSI - same predicate will be used to handle Volume limit counting for in-tree as well as CSI Volumes. Similarly same limit will be used when user is using CSI or in-tree volumes on the node.

- Existing predicates for in-tree volumes `MaxEBSVolumeCount`, `MaxGCEPDVolumeCount`, `MaxAzureDiskVolumeCount` and `MaxCinderVolumeCount`(now deprecated) will be removed when in-tree to CSI migration is GA and enabled by default for that particular volume plugin.
  - When both deprecated in-tree predicate and CSI predicate are enabled, only `MaxCSIVolumeCountPred` does useful work and the other is NOOP to save CPU. This requires CSI Driver to be installed on the node.

- Scheduler does not put pods that require CSI volumes to nodes that don't have the CSI driver installed.

- Scheduler does not increase its CPU consumption. Any regression must be approved by sig-scheduling.
  - Scheduler benchmark must be extended to schedule pods with volumes as part of this KEP.

Note: Although we are saying existing predicates will become `NOOP` in this section and elsewhere, existing predicates still have to look up `CSINode`
object and return early as applicable.

### Non-Goals

- Heterogenous clusters, i.e. clusters where access to storage is limited only to some nodes. Existing `PV.spec.nodeAffinity` handling, not modified by this KEP, will filter out nodes that don't have access to the storage, so predicates changed in this KEP don't need to worry about storage topology and can be simpler.

- Multiple plugins sharing the same volume limits. We expect that every CSI driver will have its own limits, not shared with other CSI drivers. In this KEP we support only in-tree volume plugins sharing its limits with one hard-coded CSI driver each.

- Multiple "units" per single volume. Each volume used on a node takes exactly 1 unit from `allocatable.volumes`, regardless of the volume size, its replica count, number of connections to remote servers or other underlying resources needed to use the volume. For example, multipath iSCSI volume with three paths (and thus three iSCSI connections to three different servers) still takes 1 unit from `CSINode.spec.drivers[xyz].allocatable.volumes`.

- Maximum capacity per node. Some cloud environments limit both number of attached volumes (covered in this KEP) and total capacity of attached volumes (not covered in this KEP). For example, this KEP will ensure that scheduler puts max. 128 GCE PD volumes to a [typical GCE node](https://cloud.google.com/compute/docs/machine-types#predefined_machine_types), but it won't ensure that the total capacity of the volumes is less than 64 TB.

- Volume limits does not yet integrate with cluster autoscaler if all nodes in the cluster are running at maximum volume limits.

## Proposal

Track volume limits for CSI driver in `CSINode` object instead of `Node` and update scheduler to use `CSINode` object to determining volume limits and availability of CSI driver.

Limit in `CSINode` is used instead of limit coming from `Node` object whenever available for same in-tree volume type. This mean scheduler will always try to translate in-tree driver name to CSI driver name whenever `CSINode` object has same in-tree volume type (even if migration is off).

  * To get rid of prefix + SHA for `ResourceName` of CSI volumes.
  * So in-tree volume plugin can share limits with CSI driver that uses the same storage backend.

### API Change

CSINode is split into `spec` and `status`. `spec` contains list of drivers installed to the node and their properties that do not change during lifetime of a driver. `status` is missing right now, but it may be used later e.g. for driver health that changes in time.

We expect that limits of a CSI driver does not change during lifetime of a driver and therefore we put the resource limits into `CSINodeSpec`. The only way for a driver to change the limits is to deregister and register again, e.g. by restarting its container.

```go
// Until further notice, this is existing API to introduce full context.

type CSINode struct {
	...

	// spec is the specification of CSINode
	Spec CSINodeSpec `json:"spec" protobuf:"bytes,2,opt,name=spec"`}
}

// CSINodeSpec holds information about the specification of all CSI drivers installed on a node
type CSINodeSpec struct {
	// drivers is a list of information of all CSI Drivers existing on a node.
	// If all drivers in the list are uninstalled, this can become empty.
	// +patchMergeKey=name
	// +patchStrategy=merge
	Drivers []CSINodeDriver `json:"drivers" patchStrategy:"merge" patchMergeKey:"name" protobuf:"bytes,1,rep,name=drivers"`
}

// CSINodeDriver holds information about the specification of one CSI driver installed on a node
type CSINodeDriver struct {
    // ...

    // NEW API STARTS HERE
    // Allocatable represents the resources of a node that are available for scheduling for volumes of this driver.
    Allocatable VolumeLimits
}

// VolumeLimits is a set of resource limits for scheduling of volumes.
type VolumeLimits struct {
	// Count is maximum number of volumes provided by the CSI driver that can be used by the node
	// "nil" represents no limits - the node can handle any number of volumes of the driver.
	Count *int32 `json:"count,omitempty" protobuf:"varint,1,opt,name=count`

	// Future proof: max. total size of volumes on the node can be added later
}
```

CSINode example:

```yaml
apiVersion: storage.k8s.io/v1beta1
kind: CSINode
metadata:
  name: ip-172-18-4-112.ec2.internal
spec:
  drivers:
    - name: ebs.csi.aws.com
      # Already existing fields
      nodeID: ip-172-18-4-112.ec2.internal
      topologyKeys:
         # ...
      # New API:
      allocatable:
        # AWS node can attach max. 40 volumes, 1 is reserved for the system
        count: 39
    - name: org.kernel.nfs
      allocatable:
        # NFS does not impose any limits of volumes mounted to the node
        count:    # nil means "no limit"
```

### Implementation

Section below describes behaviour of old predicates, CSI predicate and scheduler after the proposal has been implemented.
For brevity - "old predicates" refers to now deprecated cloudprovider specific predicates - `MaxEBSVolumeCount`, `MaxGCEPDVolumeCount`, `MaxAzureDiskVolumeCount` and `MaxCinderVolumeCount`.

#### Implementation Detail for all CSI Drivers

* Kubelet will create `CSINode` instance during initial CSI Driver registration.
  * Limits of each CSI volume plugin will be added to `CSINode.spec.drivers[xyz].allocatable`.
  * User may NOT change `CSINode.spec.drivers[xyz].allocatable` to override volume plugin / CSI driver values, e.g. to "reserve" some attachment to the operating system. Kubelet will periodically reconcile `CSINode` and overwrite the value.
    * Especially, `kubelet --kube-reserved` or `--system-reserved` cannot be used to "reserve" volumes for kubelet or the OS. It is not possible with existing kubelet and this KEP does not change it. We expect that CSI drivers will have configuration options / cmdline arguments to reserve some volumes and they will report their limit already reduced by that reserved amount.
* Scheduler will respect `Node.status.allocatable` and `Node.status.capacity` for CSI volumes if `CSINode` object is not available or has missing entry in `CSINode.spec.drivers[xyz].allocatable` during a deprecation period but kubelet will stop populating `Node.status.allocatable` and `Node.status.capacity` for CSI volumes.
  * After deprecation period for CSI volumes, limits coming from `Node.status.allocatable` and `Node.status.capacity` will be completely ignored by the scheduler.
* Scheduler won't schedule pods with volumes (in-tree or CSI) for which migration has been enabled and driver is not installed on the node yet.
  * Volumes for which there is no in-tree to CSI migration plan will follow a deprecation cycle before.
  * Important: this can only be implemented once volume limits are integrated with the cluster autoscaler.

#### Implementation detail for in-tree Drivers with CSI migration disabled

##### When no CSI driver for same underlying storage type is installed on the node.

* For Azure, GCEPD, AWS and Cinder - in-tree volume plugins will keep reporting their limits via `Node` object and old predicates will work as expected until CSI migration has been enabled (and GA) for given volume plugin.

##### When CSI driver for same underlying storage type is installed on the node.

* For Azure, GCEPD, AWS and Cinder - in-tree volume plugins will report their limits via `Node` object same as before.

#### Implementation detail for in-tree Drivers with CSI migration enabled

* For Azure, GCEPD, AWS and Cinder - in-tree volume plugins will report their limits via `Node` object same as before.
* Old predicates will be modified to perform an additional check of `CSINode`. If they detect that the CSI migration has been enabled for the volume, the old predicate will return early (with success) and `MaxCSIVolumeCountPred` will be responsible for counting both CSI and in-tree volumes of same type.

### User Stories

### Implementation Details/Notes/Constraints

[CSI migration library](https://github.com/kubernetes/kubernetes/tree/master/staging/src/k8s.io/csi-translation-lib) is used to find CSI driver name for in-tree volume plugins + its `VolumeHandle`. This CSI driver name is used as key in `CSINode.CSINode.spec.drivers[xyz].allocatable` list. The `VolumeHandle` is unique for each volume and will be used to de-duplicate volumes used by multiple pods on the same node.

### Risks and Mitigations

* This KEP depends on [CSI migration library](https://github.com/kubernetes/kubernetes/tree/master/staging/src/k8s.io/csi-translation-lib). It can happen that CSI migration is redesigned / cancelled.
  * Countermeasure: [CSI migration](https://github.com/kubernetes/enhancements/blob/master/keps/sig-storage/20190129-csi-migration.md) and this KEP should graduate together.

* This KEP depends on [CSI migration library](https://github.com/kubernetes/kubernetes/tree/master/staging/src/k8s.io/csi-translation-lib) ability to handle in-line in-tree volumes. Scheduler will need to get CSI driver name + `VolumeHandle` from them to count them towards the limit.

## Design Details

Existing feature gate `AttachVolumeLimit` will be re-used for implementation of this KEP. The feature is already beta and is enabled by default.

### Test Plan

* [Scheduler benchmark](https://github.com/kubernetes/community/blob/master/contributors/devel/sig-scheduling/scheduler_benchmarking.md) must be extended to run pods with volumes as part of this KEP. Following matrix will be tested:
  * Predicates:
    * All volume predicates enabled.
    * Only deprecated `MaxEBSVolumeCount`, `MaxGCEPDVolumeCount`, `MaxAzureDiskVolumeCount` and `MaxCinderVolumeCount` predicates enabled.
    * Only `MaxCSIVolumeCountPred` predicate enabled.
  * API objects:
    * Both CSINode and Node containing `spec/status.allocatable` for a volume plugin (to simulate kubelet during deprecation period).
    * Only CSINode containing `spec.drivers[xyz].allocatable` for a volume plugin (to simulate kubelet after deprecation period).
    * Only Node containing `status.allocatable` for a volume plugin (to simulate old kubelet).
  * Test results should be ideally the same as before the KEP.
    * Any deviation needs to be approved by sig-scheduling.

* Run e2e tests and kubelet version skew tests to check that scheduler picks the right values from CSINode or Node.

* Add e2e test that runs pods with both in-tree volumes and CSI driver for the same storage backend and check that they share the same volume limits.

### Graduation Criteria

##### Alpha -> Beta Graduation

N/A (`AttachVolumeLimit` feature is already beta).

##### Beta -> GA Graduation

It must graduate together with CSI migration. We can enable caching of in-use volumes on a node to improve performance before going GA.

### Upgrade / Downgrade / Version Skew Strategy

During upgrade, downgrade or version skew, kubelet may be older that scheduler. Kubelet will not fill `CSINode.spec` with volume limits and it will fill volume limits into `Node.status`. Scheduler must fall back to `Node.status` when `CSINode` is not available or its `spec` does not contain a volume plugin / CSI driver.

#### Interaction with old `AttachVolumeLimit` implementation

Due to version skew, following situations are possible (scheduler is always with `AttachVolumeLimit` enabled and with this KEP implemented):

* Kubelet has `AttachVolumeLimit` off:
  * Scheduler does not see any volume limits in `CSINode` nor `Node`.
  * In-tree volumes: since `CSINode` is missing, scheduler falls back to `MaxEBSVolumeCount`, `MaxGCEPDVolumeCount`, `MaxAzureDiskVolumeCount` and `MaxCinderVolumeCount` predicates and schedules in-tree volumes the old way with hardcoded limits.
  * CSI: from scheduler point of view, the node can handle any number of CSI volumes.

* Kubelet has old implementation of `AttachVolumeLimit` and the feature is on (kubelet fills `Node.status.available`):
  * Scheduler does not see any volume limits in `CSINode`.
  * In-tree: Since `CSINode` is missing, scheduler falls back to `MaxEBSVolumeCount`, `MaxGCEPDVolumeCount`, `MaxAzureDiskVolumeCount` and `MaxCinderVolumeCount` predicates and schedules in-tree volumes the old way.
  * CSI: Scheduler falls back to told implementation of `MaxCSIVolumeCountPred` for CSI volumes and uses limits from `Node.status`.

* Kubelet has new implementation of `AttachVolumeLimit` and the feature is on (kubelet fills `CSINode`):
  * No issue here, see this KEP.
  * Since `CSINode` is available, scheduler uses new implementation of `MaxCSIVolumeCountPred`.

As implied by the above, the scheduler needs to have both old and new implementation of `MaxCSIVolumeCountPred` and switch between them based on `CSINode` availability for a particular node until the old implementation is deprecated and removed (2 releases).

## Implementation History

* K8s 1.11: Alpha
* K8s 1.12: Beta
* K8s 1.17: GA

# Alternatives

In https://github.com/kubernetes/enhancements/pull/730 we tried to merge volume limits in `Node.status.capacity` and `Node.status.attachable`. We discovered these issues:

* We cannot use plain CSI driver name as resource name `Node.status.attachable`, as it could collide with other resources (e.g. "memory"), so we added volume specific prefix.
* Since CSI driver name can be [up to 63 character long](https://github.com/container-storage-interface/spec/blob/master/spec.md#getplugininfo), the prefix + driver name it cannot fit 64 character resource name limit. We ended up hashing the driver name to save space.

By moving volume limit to CSINode we fix both issues.
