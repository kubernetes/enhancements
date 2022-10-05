# KEP-1432: Volume Health Monitor

## Table of Contents

<!-- toc -->
- [Release Signoff Checklist](#release-signoff-checklist)
- [Motivation](#motivation)
- [User Experience](#user-experience)
  - [Use Cases](#use-cases)
- [Proposal](#proposal)
- [Implementation](#implementation)
  - [Kubelet Metrics changes](#kubelet-metrics-changes)
  - [CSI changes](#csi-changes)
    - [Add ControllerGetVolume RPC](#add-controllergetvolume-rpc)
    - [Add Node Volume Health Function](#add-node-volume-health-function)
  - [External controller](#external-controller)
    - [CSI interface](#csi-interface)
    - [Node down event](#node-down-event)
  - [Kubelet](#kubelet)
    - [CSI interface](#csi-interface-1)
  - [Alternatives](#alternatives)
    - [Alternative option 1](#alternative-option-1)
    - [Alternative option 2](#alternative-option-2)
    - [Alternative option 3](#alternative-option-3)
    - [Optional HTTP(RPC) service](#optional-httprpc-service)
    - [External node agent](#external-node-agent)
      - [CSI interface](#csi-interface-2)
- [Graduation Criteria](#graduation-criteria)
  - [Alpha](#alpha)
  - [Alpha -&gt; Beta](#alpha---beta)
  - [Beta -&gt; GA](#beta---ga)
- [Test Plan](#test-plan)
  - [Prerequisite testing updates](#prerequisite-testing-updates)
  - [Unit tests](#unit-tests)
  - [Integration tests](#integration-tests)
  - [E2E tests](#e2e-tests)
- [Production Readiness Review Questionnaire](#production-readiness-review-questionnaire)
  - [Feature enablement and rollback](#feature-enablement-and-rollback)
  - [Rollout, Upgrade and Rollback Planning](#rollout-upgrade-and-rollback-planning)
  - [Monitoring Requirements](#monitoring-requirements)
  - [Dependencies](#dependencies)
  - [Scalability](#scalability)
  - [Troubleshooting](#troubleshooting)
- [Implementation History](#implementation-history)
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
- [x] (R) KEP approvers have approved the KEP status as `implementable`
- [x] (R) Design details are appropriately documented
- [x] (R) Test plan is in place, giving consideration to SIG Architecture and SIG Testing input
- [x] (R) Graduation criteria is in place
- [x] (R) Production readiness review completed
- [ ] Production readiness review approved
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

## Motivation

Currently there is no way to monitor Persistent Volumes after they are provisioned in Kubernetes. This makes it very hard to debug and detect root causes when something happens to the volumes. An application may detect that it can no longer write to volumes. However, the root cause happened at the underlying storage system level. This requires application and infrastructure team jointly debug and find out what has triggered the problem. It could be that the volume runs out of capacity and needs an expansion. It could be that the volume was deleted by accident outside of Kubernetes. In the case of local storage, the local PVs may not be accessed any more due to the nodes break down. In either case, it will take lots of effort to find out what has happened at the infrastructure layer. It will also take complicated process to recover. Admin usually needs to intervene and fix the problem.

With health monitoring, unhealthy volumes can be detected and fixed early and therefore could prevent more serious problems to occur. While some problems may be corrected automatically by a Kubernetes controller, most problems may involve manual admin intervention or need to be fixed with specific application knowledge. In any case, volume health monitoring is a very important tool that would be beneficial to Kubernetes users.

## User Experience

### Use Cases

Many things could happen to the underlying storage system after a volume is provisioned in Kubernetes.

* Volume could be deleted by accident outside of Kubernetes.
* The disk that the volume resides on could be removed temporarily for maintenance.
* The disk that the volume resides on could fail.
* There may be configuration issues with the underlying storage system that affect the volume’s health.
* Volume could be out of capacity.
* The disk may be degrading which affects its performance.

If the volume is mounted on a pod and used by an application, the following problems could also happen.
* There may be read/write I/O errors.
* The file system on the volume may be corrupted.
* Filesystem may be out of capacity.
* Volume may be unmounted by accident outside of Kubernetes.

If the CSI driver has implemented the CSI volume health function proposed in this design document, Kubernetes could communicate with the CSI driver to retrieve any errors detected by the underlying storage system. Kubernetes can report an event and log an error about this PVC so that user can inquire this information and decide how to handle them. For example, if the volume is out of capacity, user can request a volume expansion to get more space. In the first phase, volume health monitoring is informational only as it is only reported in the events and logs. In the future, we will also look into how Kubernetes may use volume health information to automatically reconcile.

There could be conditions that cannot be reported by a CSI driver. One or more nodes where the volume is attached to may be down. This can be monitored and detected by the volume health controller so that user knows what has happened.

The Kubernetes components that monitor the volumes and report events with volume health information include the following:

* Extend Kubelet's existing volume monitoring capability to also monitor volume health on each kubernetes worker node.
  * In addition to gathering existing volume stats, Kubelet also watches volume health of the PVCs on that node. If a PVC has an abnormal health condition, an event will to reported on the pod object that is using the PVC. If multiple pods are using the same PVC, events will be reported on multiple pods.
* An external monitoring controller on the master node.
  * Monitoring controller reports events on the PVCs.

Details will be described in the following proposal section.

## Proposal

Volume monitoring is the main focus of this proposal. Reactions are not in the scope of this proposal.

The following area will be the focus of this proposal at first:

* Provide a mechanism for CSI drivers to report volume health problems at the controller and node levels.

Two main parts are involved here in the architecture.

- External Controller:
  - The external controller will be deployed as a sidecar together with the CSI controller driver, similar to how the external-provisioner sidecar is deployed.
  - Trigger controller RPC to check the health condition of the CSI volumes.
  - The external controller sidecar will also watch for node failure events. This component can be enabled via a flag.

- Kubelet:
  - Kubelet already collects volume stats from CSI node plugin by calling CSI function NodeGetVolumeStats.
  - In addition to existing volume stats collected already, Kubelet will also check volume condition collected from the same CSI function and log events to Pods if volume condition is abnormal.
  - Note that currently we do not have CSI support for local storage. When the support is available, we will implement relavant CSI monitoring interfaces as well.
  - Expose Volume Health information as Kubelet VolumeStats Metrics.

The volume health monitoring by Kubelet will be controlled by a new feature gate called `CSIVolumeHealth`.

## Implementation

### Kubelet Metrics changes

Add a new field in the [VolumeStats metrics API](https://github.com/kubernetes/kubernetes/blob/v1.22.1/staging/src/k8s.io/kubelet/pkg/apis/stats/v1alpha1/types.go#L263).

```
// VolumeStats contains data about Volume filesystem usage.
type VolumeStats struct {
	// Embedded FsStats
	FsStats `json:",inline"`
	// Name is the name given to the Volume
	// +optional
	Name string `json:"name,omitempty"`
	// Reference to the PVC, if one exists
	// +optional
	PVCRef *PVCReference `json:"pvcRef,omitempty"`

	// Note: Add the following new field
	// +optional
        // VolumeHealthStats contains data about volume health
        VolumeHealthStats `json:"volumeHealthStats,omitempty"`
}

// VolumeHealthStats contains data about volume health.
type VolumeHealthStats struct {
        // Normal volumes are available for use and operating optimally.
        // An abnormal volume does not meet these criteria.
        Abnormal bool `json:"abnormal,omitempty"`
}
```

Modify [parsePodVolumeStats](https://github.com/kubernetes/kubernetes/blob/v1.22.1/pkg/kubelet/server/stats/volume_stat_calculator.go#L172) to include the new field in the returned `stats.VolumeStats`.

The newly added Volume Health stats will be stored in [persistentStats](https://github.com/kubernetes/kubernetes/blob/v1.22.1/pkg/kubelet/server/stats/volume_stat_calculator.go#L168).

This is returned in [GetPodVolumeStats](https://github.com/kubernetes/kubernetes/blob/v1.22.1/pkg/kubelet/server/stats/fs_resource_analyzer.go#L99).

Since Prometheus does not store string metrics, `volume_health_status` will be stored as either 1 or 0. The `volume_health_status` label could be `status: abnormal`.

```
var volumeHealthMetric = metrics.NewGaugeVec(
	&metrics.GaugeOpts{
		Subsystem:      KubeletSubsystem,
                Name:           "volume_health_status",
                Help:           "Volume health status. The count is either 1 or 0.",
                StabilityLevel: metrics.ALPHA,
                },
                []string{"volume_plugin", "pvc_namespace", "pvc_name", "volume_health_status"},
)
```

### CSI changes

Container Storage Interface (CSI) specification will be modified to provide volume health check leveraging existing RPCs and adding new ones.

- Modify existing ListVolumes controller RPC
 - External controller calls the existing `ListVolumes` RPC to check health condition of volumes if supported by the CSI driver.

- Add new GetVolume controller RPC
  - External controller calls a new `GetVolume` RPC to check health condition of a particular volume if it is supported and ListVolumes is not supported.

- Extend the existing NodeGetVolumeStats RPC
  - For any PV that is mounted, the external node agent calls `NodeGetVolumeStats` to see if volume is still mounted;
  - Calls `NodeGetVolumeStats` to check if volume is usable, e.g., filesystem corruption, bad blocks, etc.

A new `VOLUME_CONDITION` controller capability is added. If a CSI driver supports `LIST_VOLUMES` and `VOLUME_CONDITION` capability, it MUST provide volume health information in `ListVolumesResponse`.

A new `GET_VOLUME` controller capability is added. If a CSI driver supports `GET_VOLUME` capability, it MUST support calling `GetVolume` to provide general volume information in `GetVolumeResponse`. If a driver supports `VOLUME_CONDITION`, it MUST provide additional health information in `GetVolumeResponse`.

A new node capability is added. If CSI driver supports the `VOLUME_CONDITION` node capability, it MUST provide health information in `NodeGetVolumeStats`.

Detailed changes needed in the CSI Spec are described in the following.

#### Add ControllerGetVolume RPC

Add a new `ControllerGetVolume` RPC. `ControllerGetVolumeResponse` should contain current information of a volume if it exists. If the volume does not exist any more, `ControllerGetVolume` should return gRPC error code `NOT_FOUND`.

Add `VolumeCondition` message in `VolumeStatus` field in `ControllerGetVolumeResponse`. This can indicate additional health information for the volume.

Whether the volume is still attached to a node is already handled by `LIST_VOLUMES_PUBLISHED_NODES` controller capability. This capability will be used by Kubernetes to reconcile - re-attach the volume if the volume is not attached any more.

```
message ControllerGetVolumeRequest {
  option (alpha_message) = true;

  // The ID of the volume to fetch current volume information for.
  // This field is REQUIRED.
  string volume_id = 1;
}
```

```
message ControllerGetVolumeResponse {
  option (alpha_message) = true;

  message VolumeStatus{
    // A list of all the `node_id` of nodes that this volume is
    // controller published on.
    // This field is OPTIONAL.
    // This field MUST be specified if the PUBLISH_UNPUBLISH_VOLUME
    // controller capability is supported.
    // published_node_ids MAY include nodes not published to or
    // reported by the SP. The CO MUST be resilient to that.
    repeated string published_node_ids = 1;

    // Information about the current condition of the volume.
    // This field is OPTIONAL.
    // This field MUST be specified if the
    // VOLUME_CONDITION controller capability is supported.
    VolumeCondition volume_condition = 2;
  }

  // This field is REQUIRED
  Volume volume = 1;

  // This field is REQUIRED.
  VolumeStatus status = 2;
}
```

```
// VolumeCondition represents the current condition of a volume.
message VolumeCondition {
  option (alpha_message) = true;

  // Normal volumes are available for use and operating optimally.
  // An abnormal volume does not meet these criteria.
  // This field is REQUIRED.
  bool abnormal = 1;

  // The message describing the condition of the volume.
  // This field is REQUIRED.
  string message = 2;
}
```

Add `GET_VOLUME` and `VOLUME_CONDITION` controller capabilities.  If a driver supports `GET_VOLUME` capability, it MUST implement the `ControllerGetVolume` RPC.  If a driver supports `VOLUME_CONDITION`, it MUST supports the `volume_condition` field in the `status` field in `ControllerGetVolumeResponse`.

If a driver supports `LIST_VOLUMES` and `VOLUME_CONDITION` controller capabilities, it MUST supports the `volume_condition` field in the `status` field in `ListVolumesResponse`.

```
// Specifies a capability of the controller service.
message ControllerServiceCapability {
  message RPC {
    enum Type {
      UNKNOWN = 0;
      CREATE_DELETE_VOLUME = 1;
      PUBLISH_UNPUBLISH_VOLUME = 2;
      LIST_VOLUMES = 3;
      GET_CAPACITY = 4;
      // Currently the only way to consume a snapshot is to create
      // a volume from it. Therefore plugins supporting
      // CREATE_DELETE_SNAPSHOT MUST support creating volume from
      // snapshot.
      CREATE_DELETE_SNAPSHOT = 5;
      LIST_SNAPSHOTS = 6;

      // Plugins supporting volume cloning at the storage level MAY
      // report this capability. The source volume MUST be managed by
      // the same plugin. Not all volume sources and parameters
      // combinations MAY work.
      CLONE_VOLUME = 7;

      // Indicates the SP supports ControllerPublishVolume.readonly
      // field.
      PUBLISH_READONLY = 8;

      // See VolumeExpansion for details.
      EXPAND_VOLUME = 9;

      // Indicates the SP supports the
      // ListVolumesResponse.entry.published_nodes field
      LIST_VOLUMES_PUBLISHED_NODES = 10;

      // Indicates that the Controller service can report volume
      // conditions.
      // An SP MAY implement `VolumeCondition` in only the Controller
      // Plugin, only the Node Plugin, or both.
      // If `VolumeCondition` is implemented in both the Controller and
      // Node Plugins, it SHALL report from different perspectives.
      // If for some reason Controller and Node Plugins report
      // misaligned volume conditions, CO SHALL assume the worst case
      // is the truth.
      // Note that, for alpha, `VolumeCondition` is intended be
      // informative for humans only, not for automation.
      VOLUME_CONDITION = 11 [(alpha_enum_value) = true];

      // Indicates the SP supports the ControllerGetVolume RPC.
      // This enables COs to, for example, fetch per volume
      // condition after a volume is provisioned.
      GET_VOLUME = 12 [(alpha_enum_value) = true];
    }

    Type type = 1;
  }

  oneof type {
    // RPC that the controller supports.
    RPC rpc = 1;
  }
}
```

#### Add Node Volume Health Function

Node Volume Health checks if a volume is still mounted and usable. To check whether a volume is usable, the CSI driver shall check if filesystem is corrupted, whether there are bad blocks, etc. in this RPC.

Instead of adding a new RPC, we can leverage the existing NodeGetVolumeStats RPC.

```
rpc NodeGetVolumeStats (NodeGetVolumeStatsRequest)
    returns (NodeGetVolumeStatsResponse) {}
```

In the NodeGetVolumeStatsRequest, there are `volume_id`, `volume_path`, and `staging_target_path`. CSI driver can figure out whether a volume is still mounted based on these parameters.

In NodeGetVolumeStatsResponse, there is already a `VolumeUsage` which includes information (available, used, and total) of the file system on the volume.

A new message `volume_condition` will be added to `NodeGetVolumeStatsResponse`. A new Node capability `VOLUME_CONDITION` will be added to indicate whether a CSI driver has implemented this function. In a `volume_condition` message, there is a boolean parameter `abnormal` indicating whether the volume is normal or not and a message that describes the details of the volume condition.

```
message NodeGetVolumeStatsRequest {
  // The ID of the volume. This field is REQUIRED.
  string volume_id = 1;

  // It can be any valid path where volume was previously
  // staged or published.
  // It MUST be an absolute path in the root filesystem of
  // the process serving this request.
  // This is a REQUIRED field.
  string volume_path = 2;

  // The path where the volume is staged, if the plugin has the
  // STAGE_UNSTAGE_VOLUME capability, otherwise empty.
  // If not empty, it MUST be an absolute path in the root
  // filesystem of the process serving this request.
  // This field is OPTIONAL.
  string staging_target_path = 3;
}
```

```
message NodeGetVolumeStatsResponse {
  // This field is OPTIONAL.
  repeated VolumeUsage usage = 1;
  // Information about the current condition of the volume.
  // This field is OPTIONAL.
  // This field MUST be specified if the VOLUME_CONDITION node
  // capability is supported.
  VolumeCondition volume_condition = 2 [(alpha_field) = true];
}
```

```
message VolumeUsage {
  enum Unit {
    UNKNOWN = 0;
    BYTES = 1;
    INODES = 2;
  }
  // The available capacity in specified Unit. This field is OPTIONAL.
  // The value of this field MUST NOT be negative.
  int64 available = 1;

  // The total capacity in specified Unit. This field is REQUIRED.
  // The value of this field MUST NOT be negative.
  int64 total = 2;

  // The used capacity in specified Unit. This field is OPTIONAL.
  // The value of this field MUST NOT be negative.
  int64 used = 3;

  // Units by which values are measured. This field is REQUIRED.
  Unit unit = 4;
}
```

```
// VolumeCondition represents the current condition of a volume.
message VolumeCondition {
  option (alpha_message) = true;

  // Normal volumes are available for use and operating optimally.
  // An abnormal volume does not meet these criteria.
  // This field is REQUIRED.
  bool abnormal = 1;

  // The message describing the condition of the volume.
  // This field is REQUIRED.
  string message = 2;
}
```

Add a `VOLUME_CONDITION` node capability. If driver supports this capability, it MUST fetch `volume_condition` information in `NodeGetVolumeStats`.

```
message NodeGetCapabilitiesRequest {
  // Intentionally empty.
}
```

```
message NodeGetCapabilitiesResponse {
  // All the capabilities that the node service supports. This field
  // is OPTIONAL.
  repeated NodeServiceCapability capabilities = 1;
}
```

```
// Specifies a capability of the node service.
message NodeServiceCapability {
  message RPC {
    enum Type {
      UNKNOWN = 0;
      STAGE_UNSTAGE_VOLUME = 1;
      // If Plugin implements GET_VOLUME_STATS capability
      // then it MUST implement NodeGetVolumeStats RPC
      // call for fetching volume statistics.
      GET_VOLUME_STATS = 2;
      // See VolumeExpansion for details.
      EXPAND_VOLUME = 3;
      // Indicates that the Node service can report volume conditions.
      // An SP MAY implement `VolumeCondition` in only the Node
      // Plugin, only the Controller Plugin, or both.
      // If `VolumeCondition` is implemented in both the Node and
      // Controller Plugins, it SHALL report from different
      // perspectives.
      // If for some reason Node and Controller Plugins report
      // misaligned volume conditions, CO SHALL assume the worst case
      // is the truth.
      // Note that, for alpha, `VolumeCondition` is intended to be
      // informative for humans only, not for automation.
      VOLUME_CONDITION = 4 [(alpha_enum_value) = true];
    }

    Type type = 1;
  }

  oneof type {
    // RPC that the controller supports.
    RPC rpc = 1;
  }
}
```

### External controller

#### CSI interface
Call ListVolumes() RPC periodically to check the health condition of volumes themselves. The frequency of the check should be tunable. A configure option will be available in the external controller to adjust this value.

If ListVolumes is not supported but GetVolume is supported, call ControllerGetVolume() RPC for volumes periodically to check the health condition of volumes themselves. The frequency of the check should be tunable. A configure option will be available in the external controller to adjust this value.

The external monitoring controller reports events on the PVCs.

#### Node down event
* External monitoring controller will check if node is marked as unresponsive by the node controller.
* The external monitoring controller will track which pods are using which PVCs and what nodes they got scheduled to.
* In the case that a node goes down, the controller will report an event for all PVCs on that node.
* The external monitoring controller reports node down events on the PVCs. The node down monitoring component can be enabled via a flag.

### Kubelet

#### CSI interface

Kubelet already collects volume stats from CSI node plugin by calling CSI function NodeGetVolumeStats.
https://github.com/kubernetes/kubernetes/blob/v1.21.0-alpha.2/pkg/volume/csi/csi_metrics.go#L71

In addition to volume stats collected already, Kubelet will also check the mounting and other health conditions from NodeGetVolumeStats.

If abnormal volume condition is detected from NodeGetVolumeStats, Kubelet will retrieve all the pods used by the particular volume and report events on the pod objects. If multiple pods are using the same volume, events will be reported on all pods. This can be done by adding logic in csi_client after the NodeGetVolumeStats call to send events to pods if volume condition is abnormal.
https://github.com/kubernetes/kubernetes/blob/v1.21.0-alpha.2/pkg/volume/csi/csi_client.go#L608

This new volume health monitoring by Kubelet will be gated by the `CSIVolumeHealth` feature gate. If enabled, Kubelet will monitor volume health when calling NodeGetVolumeStats CSI function and report events on pods when abnormal volume condition is detected. If not enabled, Kubelet works the same as before and will not check volume health when calling NodeGetVolumeStats CSI function.

### Alternatives

#### Alternative option 1

The Status field in the PVC will be used to mark volumes if they are unhealthy. The external monitoring controller sidecar will be responsible of monitoring the volumes and updating the PVC status field when needed.

```
// PersistentVolumeClaimHealthConditionType defines the health condition of PV claim.
// Valid values are "HealthFailure", "HealthWarning", "HealthUnknown".
type PersistentVolumeClaimHealthConditionType string

// These are valid health conditions of PVC
const (
        // PersistentVolumeClaimHealthFailure - Volume health failure indicates a severe problem that makes the volume unusable
        PersistentVolumeClaimHealthFailure PersistentVolumeClaimHealthConditionType = "HealthFailure"
        // PersistentVolumeClaimHealthWarning - Volume health warning indicates there is a problem but volume is still usable
        PersistentVolumeClaimHealthWarning PersistentVolumeClaimHealthConditionType = "HealthWarning"
        // PersistentVolumeClaimHealthUnknown - Volume health unknown indicates the health condition of the volume is unknown
        PersistentVolumeClaimHealthUnknown PersistentVolumeClaimHealthConditionType = "HealthUnknown"
)

// PersistentVolumeClaimHealthCondition represents the current health condition of PV claim
type PersistentVolumeClaimHealthCondition struct {
        Type   PersistentVolumeClaimHealthConditionType
        Status ConditionStatus
        ErrorCode string
        // +optional
        LastProbeTime metav1.Time
        // +optional
        LastTransitionTime metav1.Time
        // +optional
        Reason string
        // +optional
        Message string
}

// PersistentVolumeClaimStatus represents the status of PV claim
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
        // +optional
        Conditions []PersistentVolumeClaimCondition
        // +optional
        HealthConditions []PersistentVolumeClaimHealthCondition
}

// ConditionStatus defines conditions of resources
type ConditionStatus string

// These are valid condition statuses. "ConditionTrue" means a resource is in the condition;
// "ConditionFalse" means a resource is not in the condition; "ConditionUnknown" means kubernetes
// can't decide if a resource is in the condition or not. In the future, we could add other
// intermediate conditions, e.g. ConditionDegraded.
const (
        ConditionTrue    ConditionStatus = "True"
        ConditionFalse   ConditionStatus = "False"
        ConditionUnknown ConditionStatus = "Unknown"
)
```

To avoid stale information being stored on a PVC, each periodic update will mark the PVC with the latest health information, replacing previous health information added by the external controller. If the PVC becomes healthy after being marked as unhealthy previously, the controller should remove the previous information.

As mentioned earlier, reaction is not in the scope of this proposal but will be considered in the future. Before reacting to any negative health condition, the controller responsible for the reaction should call GetVolume() again to ensure the information is update to date.

Alternative option 1 is not in the main proposal because it requires giving all nodes access to a credential that has the ability to modify all PVCs. This is not desirable as it allows a malicious node to put an unhealthy status on a PVC.

#### Alternative option 2

If the agent on the node cannot be used to modify the PVC status and the monitoring logic cannot be added to Kubelet directly, we can introduce a CRD to represent the volume health. This volume health CRD is in the same namespace as the PVC that it is monitoring. It contains the PVC name and health conditions as defined in the main option. It needs to have a one on one mapping with the PVC. In the PVC status, there will be a `volumeHealthName` field pointing back to the volume health CRD.

Both the controller and the agent can create the volume health CRD for a PVC if it does not exist yet. Only one volume health CRD should be created for a PVC. Only the controller can set the `volumeHealthName` field in the PVC status.

```
type VolumeHealth struct {
        metav1.TypeMeta
        metav1.ObjectMeta
	Spec VolumeHealthSpec
	// +optional
	Status *VolumeHealthStatus
}

type VolumeHealthSpec struct {
	// The PVC name that this VolumeHealth object is monitoring
	PersistentVolumeClaimName string
}

// Note: PersistentVolumeClaimHealthCondition is already defined in the
// main section.
type VolumeHealthStatus struct {
        HealthConditions []PersistentVolumeClaimHealthCondition
}

// PersistentVolumeClaimStatus represents the status of PV claim
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
        // +optional
        Conditions []PersistentVolumeClaimCondition
        // +optional
        VolumeHealthName *string
}
```

Alternative option 2 does not address the concerns from option 1. Assuming the node-level sidecar is running in a pod as a service account, we need to grant that service account permission to modify any instance of the volume health CR, and then the monitoring controller just copies that into the PVC status.

#### Alternative option 3

We can also reuse the PV Taints and introduce a new Taint called PVUnhealthMessage for PV health condition whose key is specific (PVUnhealthMessage) and value can be set differently. 

For example, if the PV is not attached now, we can mark the PV using the PVUnhealthMessage taint like this:
```
Key: “PVUnhealthMessage”
Value: “AttachError - the pv is not attached to node1 now”
VolumeTaintEffect: NoEffect
```

If the volume is deleted, we can mark the PV using the PVUnhealthMessage taint like this:
```
Key: “PVUnhealthMessage” 
Value: “VolumeError - the volume is deleted from backend”
VolumeTaintEffect: NoEffect
```

Note that:

- all the VolumeTaintEffects are NoEffect now at first, we may talk about the reactions later in another proposal;
- the taint Value is string now, it is theoretically possible that several errors are detected for one PV, we may extend the string to cover this situation: combine the errors together and splited by semicolon or other symbols.

This was initially brought up to address the external data populator use case, but there were lots of concerns about this proposal. We do not want our main proposal to depend on something that is not approved.

#### Optional HTTP(RPC) service

Create a HTTP(RPC) service to receive volume health condition reports from other components.

Users can extend the volume health condition monitoring by setting up their own detector and report the result to the service.

This optional service is out of scope for this KEP but may be introduced in a future KEP if needed.

This option is not in the main proposal because it is a push-based method while CSI uses pull-based method.

#### External node agent

In the initial proposal and first alpha implementation of this feature, there is an external node agent component.

* An external monitoring node agent on each kubernetes worker node.
  * The node agent watches volume health of the PVCs on that node. If a PVC has an abnormal health condition, an event will to reported on the pod object that is using the PVC. If multiple pods are using the same PVC, events will be reported on multiple pods.

When moving this feature to beta, we decided to combine this with existing volume stats check in Kubelet as both are calling the same CSI function NodeGetVolumeStats.

Here's the initial proposal:

- The external node agent will be deployed as a sidecar together with the CSI node driver on every Kubernetes worker node.
- The external node agent triggers node RPC to check volume's mounting conditions.
- Note that currently we do not have CSI support for local storage. When the support is available, we will implement relavant CSI monitoring interfaces as well.

##### CSI interface
Call NodeGetVolumeStats() RPC to check the mounting and other health conditions. The frequency of the check should be tunable. A configure option will be available in the external node agent to adjust this value.

The external node agent will go through all the pods on that node and find out all volumes used by those pods. It will watch all those volumes and call NodeGetVolumeStatus() to find out their health status.

The external node agent reports events on the pod object. If multiple pods are using the same volume, report events on all pods.

## Graduation Criteria
### Alpha
* Initial feature implementation, including:
  * External controller volume health monitoring.
  * Kubelet volume health monitoring.
* Implementation in the csi-mock driver.
* Add basic unit tests.

### Alpha -> Beta
* Unit tests and e2e tests outlined in design proposal implemented.

### Beta -> GA
* Volume health support is added to multiple CSI drivers.
* Volume health feature deployed in production and have gone through at least one K8s upgrade.

## Test Plan

[x] I/we understand the owners of the involved components may require updates to
existing tests to make this code solid enough prior to committing the changes necessary
to implement this enhancement.

### Prerequisite testing updates

Volume health is added as a metric in Kubelet. There are already e2e tests that grabs all metrics from Kubelet: https://github.com/kubernetes/kubernetes/blob/master/test/e2e/instrumentation/monitoring/metrics_grabber.go

### Unit tests
* Unit tests for external controller and Kubelet volume health monitoring. The following failure scenario will be simulated in the unit tests:
  * VolumeNotFound
  * OutOfCapacity
  * VolumeUnmounted
  * NodeDown

### Integration tests

A test for volume health metric will be added here: https://github.com/kubernetes/kubernetes/blob/master/test/integration/metrics/metrics_test.go

### E2E tests
* There will not be e2e tests for external controller because volume health is reported as events on PVCs and events can disappear so we do not have a reliable way to write a test for that.
* We need e2e tests for Kubelet volume health monitoring. Volume health is added as a metric in Kubelet. There are already e2e tests that grabs all metrics from Kubelet: https://github.com/kubernetes/kubernetes/blob/master/test/e2e/instrumentation/monitoring/metrics_grabber.go. Hostpath CSI driver will be used for e2e tests. The following failure scenario will be tested in the e2e tests:
  * VolumeNotFound
  * OutOfCapacity
  * VolumeUnmounted
* Add stress and scale tests before moving from beta to GA.

## Production Readiness Review Questionnaire

### Feature enablement and rollback

_This section must be completed when targeting alpha to a release._

* **How can this feature be enabled / disabled in a live cluster?**
  - [x] Other
    - Describe the mechanism:
      This feature has a feature gate called `CSIVolumeHealth` for Kubelet.
      It is enabled when the feature gate in turned on.
      The health monitoring feature in external controller does not have a
      feature gate because it is out of tree.
      It is enabled when the health monitoring controller sidecar is deployed with the CSI driver.
    - Will enabling / disabling the feature require downtime of the control
      plane?
      Enabling the `VolumeHealth` feature gate will require downtime of Kubelet.
      From the controller side, it only affects the health monitoring controller sidecar.
    - Will enabling / disabling the feature require downtime or reprovisioning
      of a node? (Do not assume `Dynamic Kubelet Config` feature is enabled).
      Not for the controller side volume monitoring.
      For Kubelet, enabling/disabling the feature requires downtime of a node.

* **Does enabling the feature change any default behavior?**
  Enabling the `VolumeHealth` feature gate will allow Kubelet to monitor volume health, emit new metric, and
  generate events on Pods so it will change the default behavior.
  Enabling the feature from the controller side will allow events to be reported on PVCs when
  abnormal volume conditions are detected.

* **Can the feature be disabled once it has been enabled (i.e. can we rollback
  the enablement)?**
  Yes. Uninstalling the health monitoring controller sidecar will disable the feature from
  the controller side.
  Disabling the feature gate on Kubelet will prevent Kubelet from monitoring volume health and emitting the new metric.
  Existing events will not be removed but they will disappear after a period of time.
  Disabling the feature should not break an existing application as these events are for humans
  only, not for automation.

* **What happens if we reenable the feature if it was previously rolled back?**
  Events will be added to PVCs or Pods when abnormal volume conditions are
  detected again and the new metric will be emitted by Kubelet again.

* **Are there any tests for feature enablement/disablement?**
  There will be unit tests for the feature `CSIVolumeHealth` enablement/disablement.
  Since there is no feature gate for this feature on the controller side and the only way to
  enable or disable this feature is to install or unistall the sidecar, we cannot write
  tests for feature enablement/disablement on the controller side.

### Rollout, Upgrade and Rollback Planning

_This section must be completed when targeting beta graduation to a release._

* **How can a rollout fail? Can it impact already running workloads?**
  Try to be as paranoid as possible - e.g., what if some components will restart
   mid-rollout?
   This feature does not have a feature gate on the controller side. It is enabled when
   the health monitoring controller sidecar is deployed with the CSI driver.
   So the only way for a rollout to fail is that deploying the health
   monitoring controller sidecar with the CSI driver fails. If
   the health monitoring controller cannot be deployed, no events on volume
   condition will be reported on PVCs.

   If enabling the `CSIVolumeHealth` feature fails, no event on volume condition will be
   reported on the pod and the new `volume_stats_health_abnormal` metric won't be emitted.

* **What specific metrics should inform a rollback?**
  An event will be recorded on the PVC when the controller has successfully retrieved an
  abnormal volume condition from the storage system. When other errors occur in the controller,
  the errors will also be recorded as events. When a rollback happens on the controller side, that means the external health monitor controller is uninstalled. After that we won't see events on the PVC due to abnormal volume conditions.

  In Kubelet, an event will be recorded on the Pod and a `volume_stats_health_abnormal` metric will be emitted when Kubelet has successfully retrieved an
  abnormal volume condition. If the call to `NodeGetVolumeStats` fails for other reasons,
  an error will be returned and whether this will be logged as an event on the Node is up to
  the existing Kubelet logic and will not be changed. When a rollback happens, that means the feature gate is disabled again. The new metric won't be emitted after that.

* **Were upgrade and rollback tested? Was the upgrade->downgrade->upgrade path tested?**
  Describe manual testing that was done and the outcomes.
  Longer term, we may want to require automated upgrade/rollback tests, but we
  are missing a bunch of machinery and tooling and can't do that now.
  Manual testing will be done to upgrade from 1.26 to 1.27 and downgrade from 1.27 back to 1.26.

* **Is the rollout accompanied by any deprecations and/or removals of features, APIs,
fields of API types, flags, etc.?**
  Even if applying deprecation policies, they may still surprise some users.
  No.

### Monitoring Requirements

_This section must be completed when targeting beta graduation to a release._

* **How can an operator determine if the feature is in use by workloads?**
  Ideally, this should be a metric. Operations against the Kubernetes API (e.g.,
  checking if there are objects with field X set) may be a last resort. Avoid
  logs or events for this purpose.

  An operator can check the metric `csi_sidecar_operations_seconds`,
  Container Storage Interface operation duration with gRPC error code status
  total. It is reported from CSI external-health-monitor-controller sidecar.
  For the health monitor controller sidecar, `csi_sidecar_operations_seconds`
  will be measuring `ListVolumes` or `GetVolume` RPC.
  The `csi_sidecar_operations_seconds` metric should be sliced by process after
  they are aggregated to show metrics for different sidecars.

  In Kubelet, an operator can check whether the feature gate `CSIVolumeHealth`
  is enabled and whether the new metric `volume_stats_health_abnormal` is emitted.

* **What are the SLIs (Service Level Indicators) an operator can use to determine
the health of the service?**
  - [ ] Metrics
    - Metric name: csi_sidecar_operations_seconds, volume_stats_health_abnormal
    - [Optional] Aggregation method:
    - Components exposing the metric:
      csi-external-health-monitor-controller exposes the `csi_sidecar_operations_seconds` metric.
      In Kubelet, a call to `NodeGetVolumeStats` is meant to collect volume stats metrics. The new metric name is `volume_stats_health_abnormal`.
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

  The metrics `csi_sidecar_operations_seconds` includes a gRPC status code. If the
  status code is `OK`, the call is successful; otherwise, it is not successful. We
  can look at the ratio of successful vs non-successful statue codes to figure out
  the success/failure ratio.

  In Kubelet, the new metric `volume_stats_health_abnormal` will be emitted. Whether we can successfully retrieve this metric depending on the CSI call 'NodeGetVolumeStats'. This is an existing call in Kubelet. As long as the CSI driver has implemented this capability to provide volume health, it should be in the response of "NodeGetVolumeStats' call.

* **Are there any missing metrics that would be useful to have to improve observability
of this feature?**
  <!--
  Describe the metrics themselves and the reasons why they weren't added (e.g., cost,
  implementation difficulties, etc.).
  -->
  No.

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
  - [Dependency name]: installation of csi-external-health-monitor-controller sidecar
    - Usage description:
      - Impact of its outage on the feature: Installation of csi-external-health-monitor-controller sidecar is required for the feature to work from the controller side. If csi-external-health-monitor-controller is not installed, abnormal volume conditions will not be reported as events on PVCs.
        Note that CSI driver needs to be updated to implement volume health RPCs in controller/node plugins. The minimum kubernetes version should be 1.13: https://kubernetes-csi.github.io/docs/introduction.html#kubernetes-releases. K8s v1.13 is the minimum supported version for CSI driver to work, however, different CSI drivers have different requirements on supported k8s versions so users are supposed to check documentation of the CSI drivers. If the CSI node plugin on one node has been upgraded to support volume health while it is not upgraded on 3 other nodes, then we will only expect to see volume health events on pods running on that one upgraded node.
        In addition, since Kubelet is doing volume health monitoring from the node side, the supported Kubernetes version will have to be the version that supports `CSIVolumeHealth` feature when we moved volume health events report to Kubelet. So the minimum Kubernetes version will be 1.21 for the events to be reported on the pods. In Kubernetes 1.24, we also added volume_stats_health_abnormal to metrics in Kubelet. So 1.24 is the minimum required version for metrics support.
      - Impact of its degraded performance or high-error rates on the feature: If abnormal volume conditions are reported with degraded performance or high-error rates, that would affect how soon or how accurately users could manually react to these conditions.


### Scalability

_For alpha, this section is encouraged: reviewers should consider these questions
and attempt to answer them._

_For beta, this section is required: reviewers must answer these questions._

_For GA, this section is required: approvers should be able to confirm the
previous answers based on experience in the field._

* **Will enabling / using this feature result in any new API calls?**
  Describe them, providing:
  - API call type (e.g. PATCH pods): Events will be reported to PVCs and Pods and metrics will be in Kubelet if this feature is enabled.
  - estimated throughput
  - originating component(s) (e.g. Kubelet, Feature-X-controller)
  focusing mostly on:
  - components listing and/or watching resources they didn't before
    csi-external-health-monitor-controller sidecar.
    There is a monitor interval for the controller to control how often to check the volume health.
    It is configurable with 5 minute as default.
    When scaled out across many nodes, low frequency checks can still produce high volumes of
    events. To control this, we should use options on the eventrecorder to control QPS per key.
    This way we can collapse keys and have a slow update cadence per key.
    From the Kubelet side, since `NodeGetVolumeStats` has already been called by Kubelet, no additional
    call is needed.
  - API calls that may be triggered by changes of some Kubernetes resources
    (e.g. update of object X triggers new updates of object Y)
    We are adding a new `Abnormal` field to the existing Kubelet metrics API. It will be retrieved by the periodic metrics collection call. We are not changing the existing frequency of that call.
  - periodic API calls to reconcile state (e.g. periodic fetching state,
    heartbeats, leader election, etc.)

* **Will enabling / using this feature result in introducing new API types?**
  Describe them, providing:
  - API type: Adding 'Abnormal` field to Kubelet VolumeStats metrics API
  - Supported number of objects per cluster: No
  - Supported number of objects per namespace (for namespace-scoped objects): No

* **Will enabling / using this feature result in any new calls to the cloud
provider?**
  No.

* **Will enabling / using this feature result in increasing size or count of
the existing API objects?**
  Describe them, providing:
  - API type(s): Yes. We are adding new 'Abnormal` field to Kubelet VolumeStats metrics API.
  - Estimated increase in size: (e.g., new annotation of size 32B):
    New string of max length of 128 bytes; new int of 4 bytes.
  - Estimated amount of new objects: (e.g., new Object X for every existing Pod)
    The controller reports events on PVC while Kubelet reports events on Pod. They work independently of each other. It is recommended that CSI driver should not report duplicate information through the controller and Kubelet. For example, if the controller detects a failure on one volume, it should record just one event on one PVC. If Kubelet detects a failure, it should record an event on every pod used by the affected PVC.

    Recovery event will be reported once if the volume condition changes from abnormal back to normal.

* **Will enabling / using this feature result in increasing time taken by any
operations covered by [existing SLIs/SLOs]?**
  Think about adding additional work or introducing new steps in between
  (e.g. need to do X to start a container), etc. Please describe the details.
  On the controller side, this feature will periodically query storage systems to get the latest volume conditions. So this will have an impact on the performance of the operations running on the storage systems.
  In Kubelet, `NodeGetVolumeStats` is an existing call, so it won't have additional performance impact.

* **Will enabling / using this feature result in non-negligible increase of
resource usage (CPU, RAM, disk, IO, ...) in any components?**
  Things to keep in mind include: additional in-memory state, additional
  non-trivial computations, excessive access to disks (including increased log
  volume), significant amount of data sent and/or received over network, etc.
  This through this both in small and large cases, again with respect to the
  [supported limits].
  From the controller side, this will increase load on the storage systems as it periodically queries them.
  From the node side, there will be no change because NodeGetVolumeStats is already being called by Kubelet to retrieve volume stats.

### Troubleshooting

The Troubleshooting section currently serves the `Playbook` role. We may consider
splitting it into a dedicated `Playbook` document (potentially with some monitoring
details). For now, we leave it here.

_This section must be completed when targeting beta graduation to a release._

* **How does this feature react if the API server and/or etcd is unavailable?**
  If API server and/or etcd is unavailable, error messages will be logged and the controller/Kubelet will not be able to report events on PVCs or Pods.

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
      If there are log messages indicating abnormal volume conditions but there are no events reported or new metric emitted, we can check the timestamp of the messages to see if events have disappeared based on TTL or if they are never reported. If there are problems on the storage systems but they are not reported in logs or events, we can check the logs of the storage systems to figure out why this has happened.
    - Testing: Are there any tests for failure mode? If not, describe why.

* **What steps should be taken if SLOs are not being met to determine the problem?**
  If SLOs are not being met, analysis should be made to understand what have caused the problem. Debug level logging should be enabled to collect verbose logs. Look at logs to find out what might have caused the events to be missed. If it indicates an underlying problem on the storage system, then storage admin can be pulled in to help find the root cause.

[supported limits]: https://git.k8s.io/community//sig-scalability/configs-and-limits/thresholds.md
[existing SLIs/SLOs]: https://git.k8s.io/community/sig-scalability/slos/slos.md#kubernetes-slisslos

## Implementation History

- 20210902: Update KEP to add volume health to Kublet metrics.
- 20210117: Update KEP for Alpha

- 20191021: KEP updated

- 20190730: KEP updated

- 20190530: KEP submitted

- Demo implementation (using annotations): 
https://github.com/NickrenREN/kubemonitor/tree/master/build/kube_storage_monitor/local_monitor
https://github.com/NickrenREN/kubemonitor/tree/master/build/kube_storage_monitor/node_watcher
