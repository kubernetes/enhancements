# KEP-4680: Add Resource Health Status to the Pod Status for Device Plugin and DRA

<!-- toc -->
- [Release Signoff Checklist](#release-signoff-checklist)
- [Summary](#summary)
- [Motivation](#motivation)
  - [Goals](#goals)
  - [Non-Goals](#non-goals)
- [Proposal](#proposal)
  - [PodStatus.AllocatedResourcesStatus](#podstatusallocatedresourcesstatus)
  - [User Stories (Optional)](#user-stories-optional)
    - [Story 1](#story-1)
  - [Notes/Constraints/Caveats (Optional)](#notesconstraintscaveats-optional)
  - [Risks and Mitigations](#risks-and-mitigations)
- [Design Details](#design-details)
  - [Device Plugin implementation details](#device-plugin-implementation-details)
  - [DRA implementation details](#dra-implementation-details)
  - [Test Plan](#test-plan)
      - [Prerequisite testing updates](#prerequisite-testing-updates)
      - [Unit tests](#unit-tests)
      - [Integration tests](#integration-tests)
      - [e2e tests](#e2e-tests)
  - [Graduation Criteria](#graduation-criteria)
    - [Alpha](#alpha)
    - [Alpha2](#alpha2)
    - [Beta](#beta)
    - [GA](#ga)
  - [Upgrade / Downgrade Strategy](#upgrade--downgrade-strategy)
  - [Version Skew Strategy](#version-skew-strategy)
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

Items marked with (R) are required *prior to targeting to a milestone / release*.

- [X] (R) Enhancement issue in release milestone, which links to KEP dir in [kubernetes/enhancements] (not the initial KEP PR)
- [X] (R) KEP approvers have approved the KEP status as `implementable`
- [X] (R) Design details are appropriately documented
- [X] (R) Test plan is in place, giving consideration to SIG Architecture and SIG Testing input (including test refactors)
  - [ ] e2e Tests for all Beta API Operations (endpoints)
  - [ ] (R) Ensure GA e2e tests meet requirements for [Conformance Tests](https://github.com/kubernetes/community/blob/master/contributors/devel/sig-architecture/conformance-tests.md) 
  - [ ] (R) Minimum Two Week Window for GA e2e tests to prove flake free
- [ ] (R) Graduation criteria is in place
  - [ ] (R) [all GA Endpoints](https://github.com/kubernetes/community/pull/1806) must be hit by [Conformance Tests](https://github.com/kubernetes/community/blob/master/contributors/devel/sig-architecture/conformance-tests.md) 
- [ ] (R) Production readiness review completed
- [ ] (R) Production readiness review approved
- [ ] "Implementation History" section is up-to-date for milestone
- [ ] User-facing documentation has been created in [kubernetes/website], for publication to [kubernetes.io]
- [ ] Supporting documentation—e.g., additional design documents, links to mailing list discussions/SIG meetings, relevant PRs/issues, release notes

<!--
**Note:** This checklist is iterative and should be reviewed and updated every time this enhancement is being considered for a milestone.
-->

[kubernetes.io]: https://kubernetes.io/
[kubernetes/enhancements]: https://git.k8s.io/enhancements
[kubernetes/kubernetes]: https://git.k8s.io/kubernetes
[kubernetes/website]: https://git.k8s.io/website

## Summary

Today it is difficult to know when a Pod is using a device that has failed or is temporarily unhealthy. This makes troubleshooting of Pod crashes hard or impossible. This KEP will fix this by exposing device health via Pod Status. This KEP is intentionally scoped small, but can be extended later to expose more device information to troubleshoot Pod devices placement issues (for example, validating that related Pods are allocated on connected devices).

## Motivation

Device Plugin and DRA do not have a good failure handling strategy defined. With proliferation of workloads using devices (like GPU), variable quality of devices, and overcommitting of data centers on power, there are cases when devices can fail temporarily or permanently and k8s need to handle this natively.

Today, the typical design is for jobs consuming a failing device to fail with a specific error code whenever possible. For long running workloads, K8s will keep restarting the workload without reallocating it on a different device. So the container will be in crash loop backoff with limited information on why it is crashing.

Exposing unhealthy devices in Pod Status will provide a generic way to understand that the failure is related to the unhealthy device, and be able to respond to this properly.

### Goals

- Expose device health information (served by Device Plugin or DRA) in Pod Status and events.

### Non-Goals

- Expose any other device information beyond the health.
- Expose CPU assignment of the pod by CPU manager or any other resources assignment by other managers.

## Proposal

### PodStatus.AllocatedResourcesStatus

As part of the InPlacePodVerticalScaling KEP, the two new fields were introduced in Pod Status to reflect the currently allocated resources for the Pod:

```go
type ContainerStatus struct {
	...

	// AllocatedResources represents the compute resources allocated for this container by the
	// node. Kubelet sets this value to Container.Resources.Requests upon successful pod admission
	// and after successfully admitting desired pod resize.
	// +featureGate=InPlacePodVerticalScaling
	// +optional
	AllocatedResources ResourceList `json:"allocatedResources,omitempty" protobuf:"bytes,10,rep,name=allocatedResources,casttype=ResourceList,castkey=ResourceName"`

	// Resources represents the compute resource requests and limits that have been successfully
	// enacted on the running container after it has been started or has been successfully resized.
	// +featureGate=InPlacePodVerticalScaling
	// +optional
	Resources *ResourceRequirements `json:"resources,omitempty" protobuf:"bytes,11,opt,name=resources"`

	...
}
```

One field reflects the resource requests and limits and the other actual allocated resources.

This structure will contain standard resources as well as extended resources. As noted in the comment: https://github.com/kubernetes/kubernetes/pull/124227#issuecomment-2130503713, it is only logical to also include the status of those allocated resources. 

The proposal is to keep this structure as-is to simplify parsing of well-known ResourceList data type by various consumers. Typical scenario would be to compare if the `AllocatedResources` match the desired state.

The proposal is to introduce an additional field:

```go
type ContainerStatus struct {
	...

	// AllocatedResourcesStatus represents the status of various resources
	// allocated for this Container. In case of DRA, the same resource health
	// can be reported multiple times if it is associated with the multiple containers.
	// +featureGate=ResourceHealthStatus
	// +optional
	// +patchMergeKey=name
	// +patchStrategy=merge
	// +listType=map
	// +listMapKey=name
	AllocatedResourcesStatus []ResourceStatus `json:"allocatedResourcesStatus,omitempty" patchStrategy:"merge" patchMergeKey:"name" protobuf:"bytes,14,rep,name=allocatedResourcesStatus"`

	...
}
```

The `ResourceStatus` is defined as:

```go
type ResourceStatus struct {
	// Name of the resource. Must be unique within the pod and in case of non-DRA resource, match one of the resources from the pod spec.
	// For DRA resources, the value must be claim:claim_name/request.
	// The claim_name must match one of the claims from resourceClaims field in the podSpec.
	// +required
	Name ResourceName `json:"name" protobuf:"bytes,1,opt,name=name"`
	// List of unique Resources health. Each element in the list contains an unique resource ID and resource health.
	// At a minimum, ResourceID must uniquely identify the Resource
	// allocated to the Pod on the Node for the lifetime of a Pod.
	// See ResourceID type for it's definition.
	// +listType=map
	// +listMapKey=resourceID
	Resources []ResourceHealth `json:"resources,omitempty" protobuf:"bytes,2,rep,name=resources"`
}

type ResourceHealthStatus string

const (
	ResourceHealthStatusHealthy   ResourceHealthStatus = "Healthy"
	ResourceHealthStatusUnhealthy ResourceHealthStatus = "Unhealthy"
	ResourceHealthStatusUnknown   ResourceHealthStatus = "Unknown"
)

// ResourceID is calculated based on the source of this resource health information.
// For DevicePlugin:
//
//	DeviceID, where DeviceID is from the Device structure of DevicePlugin's ListAndWatchResponse type: https://github.com/kubernetes/kubernetes/blob/eda1c780543a27c078450e2f17d674471e00f494/staging/src/k8s.io/kubelet/pkg/apis/deviceplugin/v1alpha/api.proto#L61-L73
//
// DevicePlugin ID is usually a constant for the lifetime of a Node and typically can be used to uniquely identify the device on the node.
// For DRA:
//
//	<driver name>/<pool name>/<device name>: such a device can be looked up in the information published by that DRA driver to learn more about it. It is designed to be globally unique in a cluster.
type ResourceID string

// ResourceHealth represents the health of a resource. It has the latest device health information.
// This is a part of KEP https://kep.k8s.io/4680 and historical health changes are planned to be added in future iterations of a KEP.
type ResourceHealth struct {
	// ResourceID is the unique identifier of the resource. See the ResourceID type for more information.
	ResourceID ResourceID `json:"resourceID" protobuf:"bytes,1,opt,name=resourceID"`
	// Health of the resource.
	// can be one of:
	//  - Healthy: operates as normal
	//  - Unhealthy: reported unhealthy. We consider this a temporary health issue
	//               since we do not have a mechanism today to distinguish
	//               temporary and permanent issues.
	//  - Unknown: The status cannot be determined.
	//             For example, Device Plugin got unregistered and hasn't been re-registered since.
	//
	// In future we may want to introduce the PermanentlyUnhealthy Status.
	Health ResourceHealthStatus `json:"health,omitempty" protobuf:"bytes,2,name=health"`
}
```

In alpha2 in order to support pod level DRA resources, the following field will be added to the PodStatus:

```go
// PodStatus represents information about the status of a pod. Status may trail the actual
// state of a system.
type PodStatus struct {

	...

	// Status of resource claims.
	// +featureGate=DynamicResourceAllocation
	// +optional
	ResourceClaimStatuses []PodResourceClaimStatus

	...

	// AllocatedResourcesStatus represents the status of various resources
	// allocated for this Pod, but not associated with any of containers.
	// +featureGate=ResourceHealthStatus
	// +optional
	// +patchMergeKey=name
	// +patchStrategy=merge
	// +listType=map
	// +listMapKey=name
	AllocatedResourcesStatus []ResourceStatus `json:"allocatedResourcesStatus,omitempty" patchStrategy:"merge" patchMergeKey:"name" protobuf:"bytes,14,rep,name=allocatedResourcesStatus"`
}
```

***Is there any guarantee that the AllocatedResourcesStatus will be updated before Container crashed and unscheduled?***

No, there is no guarantee that the Device Plugin/DRA will detect device going unhealthy earlier than the Pod. Once device got unhealthy, container may crash and being marked as Failed already (if `restartPolicy=Never`, in other cases Pod will enter crash loop backoff).

The proposal is to update the Pod Status with the device status even if the pod has been marked as Failed already, but still known to the kubelet.

This use case is important to explore so this status may inform the retry policy introduced by the KEP: [Retriable and non-retriable Pod failures for Jobs](https://github.com/kubernetes/enhancements/issues/3329).


***Do we need the CheckDeviceHealth call introduced to the Device Plugin to work around the limitation above?***

We may consider this as a future improvement. 


***Should we introduce a permanent failure status?***

We may consider this as a future improvement. 

### User Stories (Optional)

#### Story 1

- User scheduled a Pod using the GPU device
- When GPU device fails, user sees the Pod is in crash loop backoff
- User checks the Pod Status using `kubectl describe pod`
- User sees the pod status indicating that the GPU device is not healthy
- User or some (custom for now) controller deletes the Pod and replicaset reschedules it on another available GPU

### Notes/Constraints/Caveats (Optional)

<!--
What are the caveats to the proposal?
What are some important details that didn't come across above?
Go in to as much detail as necessary here.
This might be a good place to talk about core concepts and how they relate.
-->

### Risks and Mitigations

There is not many risks of this KEP. The biggest risk is that Device Plugins will not be
able to detect device health reliably and fast enough to assign this status to the
Pods, marked as `restartPolicy=Never`. End users will expect this field and the
missing health information will confuse them.

## Design Details

### Device Plugin implementation details

Kubelet already keeps track of healthy and unhealthy devices as well as the mapping of those devices to Pods.

One improvement will be needed is to distinguish unhealthy devices (marked unhealthy explicitly) and when device plugin was unregistered.

NVIDIA device plugin has the checkHealth implementation: https://github.com/NVIDIA/k8s-device-plugin/blob/eb3a709b1dd82280d5acfb85e1e942024ddfcdc6/internal/rm/health.go#L39 that has more information than simple “Unhealthy”.

We should consider introducing another field to the Status that will be a free form error information as a future improvement.

### DRA implementation details

Today DRA does not return the health of the device back to kubelet. In `1.30` we had a `ListAndWatch()` API similar to DevicePlugin, from which we could have inferred device health. However, this API is being removed in `1.31`, necessitating a new approach for health monitoring.

The following design outlines how Kubelet will obtain health information from DRA plugins and use it to update the PodStatus. This design focuses on an optional, proactive health reporting mechanism from DRA plugins.

**High-Level Architectural Approach for DRA Health**

1.  **Optional gRPC Stream:** A new, optional gRPC service (`NodeHealth`) is defined within a dedicated API group (`dra-health/v1alpha1`). DRA plugins can implement this service. It exposes a server-streaming RPC (`WatchResources`) allowing the plugin to proactively send health updates for its managed devices to Kubelet.
2.  **Health Information Cache:** Kubelet's DRA Manager will maintain a persistent cache (`healthInfoCache`) storing the latest known health status (Healthy, Unhealthy, Unknown) and a `LastUpdated` timestamp for each device, keyed by driver name and device identifiers. This cache handles state reconciliation, timeouts for stale data, and persists across Kubelet restarts.
3.  **Kubelet Integration:** Kubelet's DRA Manager (`pkg/kubelet/cm/dra/manager.go`) acts as the gRPC client. Upon plugin registration, it attempts to initiate the `WatchResources` stream. If successful, it consumes the health stream, updates the `healthInfoCache`, identifies which Pods are affected by reported health changes (using its existing `claimInfoCache`), and signals the Kubelet status manager via a dedicated update channel.
4.  **Update Channel Aggregation:** The Container Manager (`pkg/kubelet/container_manager_linux.go`) merges update signals from the DRA Manager's update channel (if the DRA health feature is enabled) and the Device Manager's existing update channel into a single channel (`Updates()`) monitored by the Kubelet status manager.
5.  **PodStatus Update:** Kubelet's pod synchronization logic, upon receiving an update signal or during other Pod updates, calls the Container Manager's `UpdateAllocatedResourcesStatus` method. This, in turn, calls the DRA Manager's `UpdateAllocatedResourcesStatus` method, which reads the current health status for the Pod's allocated DRA devices from `healthInfoCache` and populates the `pod.Status.ContainerStatuses[].AllocatedResourcesStatus` field (and `pod.Status.AllocatedResourcesStatus` for pod-level resources in alpha2) with the correct health information.

*Note: Kubelet will ONLY currently use this health information to update the Pod Status. The DRA plugin remains responsible for updating ResourceSlices to taint resources or mark them unhealthy through other mechanisms if needed.*

**Core Components and Logic for DRA Health**

1.  **DRA Health Information Cache (`pkg/kubelet/cm/dra/healthinfo.go`)**
    *   **Purpose:** To maintain the source of truth for device health within Kubelet, reconciling reported data and handling missing/stale information.
    *   **Data Structure:**
        *   `healthInfoCache`: Holds `HealthInfo` (e.g., `*state.DevicesHealthMap`) and state file persistence information.
        *   `state.DevicesHealthMap`: `map[string]DriverHealthState` (keyed by driver name).
        *   `state.DriverHealthState`: Contains `Devices []DeviceHealth`.
        *   `state.DeviceHealth`: Contains `PoolName string`, `DeviceName string`, `Health state.DeviceHealthString`, `LastUpdated time.Time`.
        *   `state.DeviceHealthString`: A string type with values "Healthy", "Unhealthy", "Unknown".
    *   **Key Features & Functions:**
        *   `newHealthInfoCache(stateFile)`: Initializes cache, loads from checkpoint file if it exists.
        *   `updateHealthInfo(driverName, devices)`: Performs full-state reconciliation for a driver based on a `WatchResourcesResponse`. Updates `LastUpdated` timestamps. Marks devices not present in the update as "Unknown" if `time.Since(LastUpdated) > healthTimeout`. Preserves existing health status if within timeout. Returns information about changed devices.
        *   `getHealthInfo(driverName, poolName, deviceName)`: Returns current health, considering `healthTimeout`. If `time.Since(device.LastUpdated) > healthTimeout`, it returns "Unknown".
        *   `clearDriver(driverName)`: Removes all data for a driver (e.g., on deregistration or stream end).
        *   `saveToCheckpoint()` / `loadFromCheckpoint()`: Handles persistence via JSON marshalling.
        *   `healthTimeout`: A constant (e.g., 30s) for marking stale devices "Unknown".
    *   **Handling Timeouts and 'Unknown' State:**
        *   "Unknown" status due to timeout is determined on-demand by `getHealthInfo`.
        *   `updateHealthInfo` also actively sets "Unknown" for devices previously known but not in the latest update from the plugin, if their `LastUpdated` timestamp exceeds `healthTimeout`.
        *   No separate background expiration worker is required; Kubelet's existing status manager update triggers will naturally reflect timeouts when `getHealthInfo` is called.
        *   If a plugin stops sending updates for all its devices (e.g., plugin crashes, stream ends), `HandleWatchResourcesStream` calls `clearDriver`, and subsequent calls to `getHealthInfo` for these devices will return "Unknown".

2.  **gRPC API for DRA Device Health (`staging/src/k8s.io/kubelet/pkg/apis/dra-health/v1alpha1/api.proto`)**
    *   A new API group `dra-health/v1alpha1` is introduced.
    *   Defines a `NodeHealth` service:
        ```proto
        service NodeHealth {
          // WatchResources allows a DRA plugin to stream health updates for its devices to Kubelet.
          // Kubelet calls this method, and the plugin streams responses.
          // This method is optional; if not implemented by a plugin, Kubelet will assume
          // devices managed by that plugin have an "Unknown" health status.
          rpc WatchResources(WatchResourcesRequest) returns (stream WatchResourcesResponse) {}
        }

        message WatchResourcesRequest {
          // Reserved for future use, e.g., filtering or options.
        }

        message WatchResourcesResponse {
          // A list of all devices managed by the plugin for which health is being reported.
          // This should be a complete list for the driver; Kubelet will reconcile this state.
          repeated DeviceHealth devices = 1;
        }

        message DeviceHealth {
          // The name of the resource pool this device belongs to.
          // Required.
          string pool_name = 1;
          // The unique name of the device within the pool.
          // Required.
          string device_name = 2;
          // Health status of the device.
          // Expected values: "Healthy", "Unhealthy", "Unknown".
          // Required.
          string health_status = 3;
          // Timestamp of when this health status was last determined by the plugin, as a Unix timestamp (seconds).
          // Required.
          int64 last_updated_timestamp = 4;
          // Optional: A globally unique resource name, e.g., <driver>/<pool>/<device>.
          // Useful for correlating with other systems.
          string resource_name = 5;
        }
        ```
    *   This gRPC service is kept separate from the core DRA API (`Node` service in `k8s.io/kubelet/pkg/apis/dra/v1alphaX`) to signify its optionality.
    *   If a plugin does not implement this service, plugin registration will continue. If device health is queried for such a plugin (e.g., during a call to `UpdateAllocatedResourcesStatus`), its devices will report an "Unknown" health state.

3.  **Kubelet DRA Plugin Client and Registration for Health Monitoring (`pkg/kubelet/cm/dra/plugin/plugin.go`, `pkg/kubelet/cm/dra/plugin/registration.go`)**
    *   **Plugin Client Logic:**
        *   The `Plugin` struct will hold a `healthClient` (e.g., `drahealthv1alpha1.NodeHealthClient`), `healthStreamCtx (context.Context)`, and `healthStreamCancel (context.CancelFunc)`.
        *   `getOrCreateGRPCConn()`: Initializes the `healthClient` after gRPC connection establishment.
        *   `WatchResources(ctx)`: A method on the plugin client, called by the registration handler, to initiate the gRPC stream via `healthClient.WatchResources()`.
        *   `SetHealthStream`, `HealthStreamCancel`: Manage the context and cancellation for the health stream goroutine.
    *   **Registration:**
        *   The `RegistrationHandler` will be extended to handle the health stream.
        *   `RegisterPlugin`: After successful registration with the core DRA API, it will attempt to call `pluginInstance.WatchResources()`.
            *   If an `Unimplemented` error is received, Kubelet logs this and continues registration without health monitoring for that plugin. PodStatus will not be populated with health for this driver's resources (will default to "Unknown").
            *   On Success: It stores the stream context/cancel func using `pluginInstance.SetHealthStream()` and starts a background goroutine (e.g., `go h.streamHandler.HandleWatchResourcesStream(streamCtx, stream, pluginName)` via an interface).
            *   On Replacement of a plugin: The `HealthStreamCancel()` function for the old plugin instance is called to stop its health stream.
    *   **Deregistration:**
        *   `DeRegisterPlugin`: Calls `p.HealthStreamCancel()` to stop the health stream goroutine and cleans up related plugin resources.

4.  **Kubelet DRA Manager and Container Manager Integration (`pkg/kubelet/cm/dra/manager.go`, `pkg/kubelet/cm/container_manager_linux.go`)**
    *   **DRA Manager (`pkg/kubelet/cm/dra/manager.go`):**
        *   `ManagerImpl` will implement a `plugin.StreamHandler` interface (or similar). It holds the `healthInfoCache`, a `healthInfoMutex`, and an `update chan resourceupdates.Update`.
        *   `HandleWatchResourcesStream(ctx context.Context, stream drahealthv1alpha1.NodeHealth_WatchResourcesClient, pluginName string)`: This function runs in a goroutine for each plugin that supports health monitoring. It receives health updates from the stream, calls `healthInfoCache.updateHealthInfo()`. If `updateHealthInfo` indicates devices changed state, it finds affected Pod UIDs from its `claimInfoCache` and sends these UIDs to its `update` channel (non-blocking). It logs errors/cancellation and exits. A `defer` function will be added to clean the cache for the specific driver if the stream ends or an error occurs (e.g., call `healthInfoCache.clearDriver(pluginName)`).
        *   `Updates() <-chan resourceupdates.Update`: Implemented method that returns the DRA manager's `m.update` channel.
        *   `UpdateAllocatedResourcesStatus(pod, status)`: Reads health for the pod's allocated DRA devices from `healthInfoCache` (using `getHealthInfo`), formats it into `v1.ResourceStatus` (with Name = claim name, Resources = slice of `v1.ResourceHealth`), and updates the input `PodStatus`.
        *   `GetWatcherHandler()`: Instantiates `plugin.NewRegistrationHandler`, passing the `ManagerImpl` instance (which implements the stream handling logic) as the `plugin.StreamHandler`.
    *   **Container Manager (`pkg/kubelet/cm/container_manager_linux.go`):**
        *   `Updates()`: Merges the channel returned by `m.draManager.Updates()` (if DRA health is enabled) with the `deviceManager`'s update channel into a single output channel consumed by the main Kubelet sync loop.
        *   `UpdateAllocatedResourcesStatus()`: Calls the device manager's update method, then calls the DRA manager's `UpdateAllocatedResourcesStatus` method (guarded by the feature gate).


### Test Plan

[X] I/we understand the owners of the involved components may require updates to
existing tests to make this code solid enough prior to committing the changes necessary
to implement this enhancement.

##### Prerequisite testing updates

Device Plugin and DRA are relatively new features and have a reasonable test coverage.

##### Unit tests

- `k8s.io/kubernetes/pkg/kubelet/cm/devicemanager`: `5/31/2024` - `84.1`
- `k8s.io/kubernetes/pkg/kubelet/cm/dra`: `5/31/2024` - `59.2`
- `k8s.io/kubernetes/pkg/kubelet/cm/dra/plugin`: `5/31/2024` - `34`
- `k8s.io/kubernetes/pkg/kubelet/cm/dra/state`: `5/31/2024` - `98`

  The following areas will have thorough unit test coverage:
-   `pkg/kubelet/cm/dra/healthinfo.go`:
    -   Cache initialization (new, load from checkpoint, file not found).
    -   `updateHealthInfo`:
        -   Correct reconciliation of device lists.
        -   `LastUpdated` timestamp updates.
        -   Marking devices not in update as "Unknown" after `healthTimeout`.
        -   Preserving health status within `healthTimeout`.
        -   Correct identification of `changedDevices`.
    -   `getHealthInfo`:
        -   Returning correct status for existing devices.
        -   Returning "Unknown" for timed-out devices.
        -   Returning "Unknown" for non-existent devices/drivers.
    -   `clearDriver`: Correct removal of driver data.
    -   Persistence: `saveToCheckpoint` and `loadFromCheckpoint` logic.
-   `pkg/kubelet/cm/dra/plugin/registration.go` (and related client logic in `plugin.go`):
    -   Successful health stream start and goroutine launch.
    -   Handling of `Unimplemented` error from plugin for `WatchResources`.
    -   Correct cancellation of health stream on plugin replacement or deregistration.
    -   Error handling during stream initiation.
-   `pkg/kubelet/cm/dra/manager.go`:
    -   `HandleWatchResourcesStream`:
        -   Correctly processing messages from the stream.
        -   Calling `healthInfoCache.updateHealthInfo`.
        -   Finding affected Pod UIDs from `claimInfoCache` when health changes.
        -   Sending Pod UIDs to the update channel.
        -   Error handling, stream termination, and cache cleanup (`clearDriver`).
    -   `UpdateAllocatedResourcesStatus`:
        -   Correctly reading from `healthInfoCache`.
        -   Formatting `v1.ResourceStatus` accurately.
        -   Populating `PodStatus` for both container-level and pod-level DRA resources.
    -   `GetWatcherHandler` and `Updates` channel logic.


##### Integration tests

N/A

##### e2e tests

Planned tests:

- Device marked unhealthy - the state is reflected in pod status
- Device marked unhealthy and back to healthy after some time - pod status was changed to unhealthy temporarily
- Device marked as unhealthy and back to healthy in quick succession - pod status reflects the latest health status
- Pod failed due to unhealthy device, earlier than device plugin detected it. Pod status is still updated.
- Pod is in crash loop backoff due to unhealthy device - pod status is updated to unhealthy

For alpha rollout and rollback:

- Fields dropped on update when feature gate is disabled
- Field is not populated after the feature gate is disabled
- Field is populated again when the feature gate is enabled

Test coverage will be listed once tests are implemented.

- <test>: <link to test coverage>

### Graduation Criteria

#### Alpha

- New field is introduced in Pod Status
- Feature implemented in Device Manager behind a feature flag
- Initial e2e tests completed and enabled

#### Alpha2

- Feature implemented in DRA behind a feature flag
- e2e tests completed and enabled for DRA

#### Beta

- Complete e2e tests coverage

#### GA

- Feedback is collected on usability of the field
- Example of real-world usage with one of the device plugin. For example, NVIDIA Device Plugin

### Upgrade / Downgrade Strategy

The feature exposes a new field based on information the Device Plugin already exposes. There will be no dependency on upgrade/downgrade, feature will either work or not.

DRA implementation requires DRA interfaces change. DRA is in alpha and in active development. The feature will follow the DRA ugrade/downgrade strategy.

### Version Skew Strategy

There is no issue with the version skew. Kubelet that will expose this flag will
always be the same version of behind the API, which introduced this new field.

## Production Readiness Review Questionnaire

### Feature Enablement and Rollback

Simple change of a feature gate will either enable or disable this feature.

###### How can this feature be enabled / disabled in a live cluster?


- [X] Feature gate (also fill in values in `kep.yaml`)
  - Feature gate name: `ResourceHealthStatus`
  - Components depending on the feature gate: `kubelet` and `kube-apiserver`

###### Does enabling the feature change any default behavior?

No

###### Can the feature be disabled once it has been enabled (i.e. can we roll back the enablement)?

Yes, with no side effect except of missing the new field in pod status. Values written 
while the feature was enabled will continue to have it and may be wiped on next update request.
They also may be ignored on reads.
Re-enablement of the feature will not guarantee to keep the values written before the
feature was disabled.

###### What happens if we reenable the feature if it was previously rolled back?

The pod status will be updated again. Consistency will not be guaranteed for fields written
before the last enablement. 

###### Are there any tests for feature enablement/disablement?

Yes, see in e2e tests section.

### Rollout, Upgrade and Rollback Planning

###### How can a rollout or rollback fail? Can it impact already running workloads?

No

###### What specific metrics should inform a rollback?

API server error rate increase. `apiserver_request_total` filtered by `code` to be non `2xx`.
API validation error is the most likely indication of an error.

Potential errors on kubelet would likely be exposed as error logs and events on Pods.

###### Were upgrade and rollback tested? Was the upgrade->downgrade->upgrade path tested?

Will be tested, but we do not expect any issues.

###### Is the rollout accompanied by any deprecations and/or removals of features, APIs, fields of API types, flags, etc.?

No

### Monitoring Requirements

###### How can an operator determine if the feature is in use by workloads?

Check the Pod Status.

###### How can someone using this feature know that it is working for their instance?

- [X] API pod.status

###### What are the reasonable SLOs (Service Level Objectives) for the enhancement?

N/A

###### What are the SLIs (Service Level Indicators) an operator can use to determine the health of the service?

N/A

###### Are there any missing metrics that would be useful to have to improve observability of this feature?

There are a few error modes for this feature:
1. API issues accepting the new field - for example kubelet is writing the field in a format not acceptable by the API server
2. kubelet fails while populating this field

First error mode can be observer with the metric `apiserver_request_total` filtered by `code` to be non `2xx`.

There is no good metric for the second error mode because it will not be clear what part of processing may fail.
The most likely indication of an error would be the increased number of error events on the Pod.

### Dependencies

DRA implementation.

###### Does this feature depend on any specific services running in the cluster?

No

### Scalability

###### Will enabling / using this feature result in any new API calls?

Pod Status size will increase insignificantly.

###### Will enabling / using this feature result in introducing new API types?

New field on Pod Status.

###### Will enabling / using this feature result in any new calls to the cloud provider?

No

###### Will enabling / using this feature result in increasing size or count of the existing API objects?

Pod Status size will increase insignificantly.

###### Will enabling / using this feature result in increasing time taken by any operations covered by existing SLIs/SLOs?

No

###### Will enabling / using this feature result in non-negligible increase of resource usage (CPU, RAM, disk, IO, ...) in any components?

Not significantly. We already keep all the collection in memory, just need to connect dots.

###### Can enabling / using this feature result in resource exhaustion of some node resources (PIDs, sockets, inodes, etc.)?

No

### Troubleshooting

###### How does this feature react if the API server and/or etcd is unavailable?

N/A

###### What are other known failure modes?

Not applicable.

###### What steps should be taken if SLOs are not being met to determine the problem?

## Implementation History

- `v1.31`: KEP is in alpha

## Drawbacks

Not that we can think of.

## Alternatives

There are a few alternatives to this proposal.

**First**, an API similar to Pod Resources API can be exposed by kubelet to query via kubectl or directly thru some node exposed port. The problem with this approach is:
- it opens up a new API surface
- It will be impossible to get status for Pods that have completed already

**Second**, exposing the status for DRA via claims - this approach leads to a debate on how to ensure security so kubelet is limited to which statuses it can set. With this approach, there are mechanisms in place to ensure that kubelet updates status for Pods scheduled on that node.

## Infrastructure Needed (Optional)

We may need to update sample device plugin. No special infra is needed as emulating real GPU failures or failures in other devices is not practical.
