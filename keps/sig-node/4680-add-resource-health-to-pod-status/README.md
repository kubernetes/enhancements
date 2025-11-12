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
    - [High-Level Architectural Approach for DRA Health](#high-level-architectural-approach-for-dra-health)
    - [gRPC API for DRA Device Health](#grpc-api-for-dra-device-health)
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
	// Message provides additional human-readable context about the health status.
	// This can include error details, failure reasons, or other diagnostic information.
	// This field is optional and may be empty for healthy resources.
	// +optional
	Message string `json:"message,omitempty" protobuf:"bytes,3,opt,name=message"`
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

Note: Updating Pod Status with device health after the pod has been marked as Failed is **not supported** due to a
race condition in the Kubelet's DRA manager cleanup. See the Known Limitations section for details.


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

- **DRA Device Health Timeout Configuration:** The timeout for marking a DRA device's health as "Unknown" 
  when no updates are received can be configured per device through the `health_check_timeout_seconds` field
  in the `DeviceHealth` message. This allows different hardware types (e.g., GPUs, FPGAs, TPUs, storage devices)
  to specify appropriate timeout values based on their health-reporting characteristics. If not specified,
  Kubelet will use a default timeout of 30 seconds. This addresses 
  [Issue #133118](https://github.com/kubernetes/kubernetes/issues/133118) and the discussion in 
  [PR #130606](https://github.com/kubernetes/kubernetes/pull/130606/files#r2221829511).


- **Failure Message Field:** The `ResourceHealth` struct includes an optional `Message` field that provides
  additional human-readable context about device health status. This field enables Device Plugins and DRA drivers
  to report detailed error information, failure reasons, and diagnostic information beyond the basic health status.
  This enhancement improves troubleshooting capabilities for device-related failures. See
  [Issue #133202](https://github.com/kubernetes/kubernetes/issues/133202) and
  [PR #134506](https://github.com/kubernetes/kubernetes/pull/134506) for implementation details.

- **Known Limitation - Device Health for Terminated Pods:** Device health status is **not** updated in PodStatus
  after a Pod has terminated (e.g., in Failed state). Due to a race condition between pod termination and
  health status updates, the Kubelet's DRA manager cleans up the ClaimInfo from its cache before health updates
  can be applied. The complexity required to fix this (tombstoning terminated ClaimInfo entries) was deemed
  not worth the benefit for this edge case. The core value for long running services (`RestartPolicy: Always`)
  is unaffected. See [Issue #132978](https://github.com/kubernetes/kubernetes/issues/132978) for details on why
  this was closed without implementation.
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

Today DRA does not return the health of the device back to kubelet. The proposal is to extend the
type `BasicDevice` (from [staging/src/k8s.io/dynamic-resource-allocation/api/types.go](https://github.com/kubernetes/kubernetes/blob/master/staging/src/k8s.io/dynamic-resource-allocation/api/types.go#L58)) to include the Health field the same way it is done in the Device Plugin as well as a device ID.

The following design outlines how Kubelet will obtain health information
from DRA plugins and use it to update the PodStatus. This design focuses on an
optional, proactive health reporting mechanism from DRA plugins.

#### High-Level Architectural Approach for DRA Health

1.  **Optional gRPC Stream:** A new, optional gRPC service for health monitoring
    will be defined. DRA plugins can implement this service to proactively send
    health updates for their managed devices to Kubelet. It will expose a
    server-streaming RPC that allows the plugin to send a complete list of
    device health states whenever a change occurs. If a plugin does not
    implement this service, the health of its devices will be reported as "Unknown".

2.  **Health Information Cache:** Kubelet's DRA Manager will maintain a
    persistent cache of device health information. This cache will store the
    latest known health status (e.g., Healthy, Unhealthy, Unknown) and a
    timestamp for each device, keyed by driver and device identifiers. The cache
    will be responsible for reconciling the state reported by the plugin, handling
    timeouts for stale data (marking devices as "Unknown" if not updated
    within a certain period), and persisting this information across Kubelet restarts.
    
    **Note:** The timeout for marking a device's health as "Unknown" can be
    configured per device via the `health_check_timeout_seconds` field in the
    `DeviceHealth` message. If not specified, Kubelet will use a default timeout
    of 30 seconds. This addresses [Issue #133118](https://github.com/kubernetes/kubernetes/issues/133118),
    allowing different hardware types (e.g., GPUs, FPGAs, TPUs, storage) to specify
    appropriate timeout values based on their health-reporting characteristics.

3.  **Kubelet Integration:** The DRA Manager in Kubelet will act as the gRPC client.
    Upon plugin registration, it will attempt to initiate the health monitoring
    stream. If successful, it will consume the health updates, update its
    internal health cache, and identify which Pods are affected by any
    reported health changes. For seamless plugin upgrades, where multiple
    instances of a plugin might run concurrently, the Kubelet will always
    watch the most recently registered plugin for health updates.

4.  **PodStatus Update:** When health changes for a device are detected, the DRA manager
    will trigger an update for the affected Pods. Kubelet's main pod synchronization
    logic will then read the current health status for the Pod's allocated DRA devices
    from the health cache and populate the `AllocatedResourcesStatus` field in the
    PodStatus with the correct health information.

  *Note: Kubelet will only use this health information to update the Pod
  Status. The DRA plugin remains responsible for other actions, such as tainting
  ResourceSlices to prevent scheduling on unhealthy resources.*

#### gRPC API for DRA Device Health

A new gRPC service, `NodeHealth`, will be introduced in a new API group (e.g., `dra-health/v1alpha1`) to keep it separate from the core DRA API and signify its optionality.

The service will define a `WatchResources` RPC:

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
  // Health check timeout duration in seconds for this device.
  // If not specified or zero, Kubelet will use a default timeout.
  // Optional.
  int64 health_check_timeout_seconds = 5;
}
```

### Test Plan

[X] I/we understand the owners of the involved components may require updates to
existing tests to make this code solid enough prior to committing the changes necessary
to implement this enhancement.

##### Prerequisite testing updates

The existing test coverage for Device Manager and DRA will be used as a baseline. New code introduced by this KEP will include thorough unit tests to maintain or improve coverage.

##### Unit tests

Current coverage for the relevant packages (as of June 2025):
- `k8s.io/kubernetes/pkg/kubelet/cm/devicemanager`: `84.8%`
- `k8s.io/kubernetes/pkg/kubelet/cm/dra`: `79.8%`
- `k8s.io/kubernetes/pkg/kubelet/cm/dra/plugin`: `84.0%`
- `k8s.io/kubernetes/pkg/kubelet/cm/dra/state`: `46.2%`

The new DRA health monitoring logic will have thorough unit test coverage, including:

-   **Health Information Cache Logic:**
    -   Cache initialization from scratch and from a checkpoint file.
    -   State reconciliation of device health based on plugin reports.
    -   Correct handling of `LastUpdated` timestamps.
    -   Marking devices as "Unknown" after a timeout period.
    -   Correctly identifying which devices have changed health status.
    -   Accurate retrieval of health status for existing, timed-out, and non-existent devices.
    -   Proper cleanup of a driver's health data upon its deregistration.
    -   Persistence logic for saving to and loading from the checkpoint file.
-   **Plugin Registration and gRPC Stream Handling:**
    -   Verification of successful health stream startup and background processing.
    -   Graceful handling of plugins that do not implement the health monitoring service (`Unimplemented` error).
    -   Correct cancellation of the health stream when a plugin is replaced or deregistered.
    -   Error handling during stream initiation and message reception.
-   **DRA Manager Logic:**
    -   Correct processing of health update messages from the gRPC stream.
    -   Accurate identification of Pods affected by a health change.
    -   Properly sending update notifications for affected Pods.
    -   Correct population of the `AllocatedResourcesStatus` field in the Pod's status object.

##### Integration tests

N/A

##### e2e tests

Planned tests will cover the user-visible behavior of the feature:

-   **Basic Health Reporting:**
    -   Verify that when a DRA plugin reports a device as unhealthy, the PodStatus is updated to reflect this.
    -   Verify that when the device becomes healthy again, the PodStatus is correctly updated.
-   **State Transitions:**
    -   Test rapid health state changes (e.g., unhealthy to healthy and back) to ensure the final PodStatus reflects the latest state.
-   **Failure Scenarios:**
    -   Verify that a Pod in a `CrashLoopBackOff` state due to an unhealthy device correctly shows the device's unhealthy status.
-   **Feature Gate Behavior (for Alpha):**
    -   When the feature gate is disabled, verify that the `AllocatedResourcesStatus` field is not populated by the DRA manager.
    -   When the feature gate is disabled on an existing cluster, verify that existing health information is gracefully ignored or removed on the next Pod update.
    -   When the feature gate is re-enabled, verify that health reporting resumes correctly.

### Graduation Criteria

#### Alpha

- New field is introduced in Pod Status
- Feature implemented in Device Manager behind a feature flag
- Initial e2e tests completed and enabled

#### Alpha2

- Feature implemented in DRA behind a feature flag
- e2e tests completed and enabled for DRA

#### Beta

The following requirements must be met for Beta graduation:

- Complete e2e tests coverage
- **Configurable Device Health Check Timeout** ([Issue #133118](https://github.com/kubernetes/kubernetes/issues/133118), [PR #133752](https://github.com/kubernetes/kubernetes/pull/133752)):
  Verify that the configurable device health check timeout implementation (via `health_check_timeout_seconds` field)
  works correctly across different plugin vendors and hardware types (e.g., GPUs, FPGAs, TPUs, storage devices).
- **Failure Message Field** ([Issue #133202](https://github.com/kubernetes/kubernetes/issues/133202), [PR #134506](https://github.com/kubernetes/kubernetes/pull/134506)):
  Support for a message field in device health reporting to provide additional context about health status and failures,
  enabling better troubleshooting capabilities.

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

Yes, with no side effect except of missing the new field in pod status. When the feature is disabled,
the values of the `AllocatedResourcesStatus` fields will be dropped when serving the API even if they
are written to storage. This prevents clients from acting on potentially stale data when the feature
is off. Values written while the feature was enabled may be wiped on next update request.
Re-enablement of the feature will not guarantee to keep the values written before the
feature was disabled.

###### What happens if we reenable the feature if it was previously rolled back?

The pod status will be updated again. When the feature is re-enabled, there may be a brief period
where stale values from storage reappear in the API before kubelet and controllers actuate and update
the values with current device health information. This period should be kept as short as possible
through normal kubelet reconciliation. Consistency will not be guaranteed for fields written
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

- `v1.31`: KEP is in alpha and imlpemented for Device Plugin

## Drawbacks

- **No post mortem health status for terminated pods:** For batch jobs using `RestartPolicy: Never`,
  device health status will not be updated after the pod terminates. This means "post mortem"
  troubleshooting for batch jobs cannot rely on this field. The race condition between pod termination
  and health updates would require significant complexity to fix (tombstoning ClaimInfo entries in the
  DRA manager), which was deemed not worth the benefit. See [Issue #132978](https://github.com/kubernetes/kubernetes/issues/132978).

## Alternatives

There are a few alternatives to this proposal.

**First**, an API similar to Pod Resources API can be exposed by kubelet to query via kubectl or directly thru some node exposed port. The problem with this approach is:
- it opens up a new API surface
- It will be impossible to get status for Pods that have completed already

**Second**, exposing the status for DRA via claims - this approach leads to a debate on how to ensure security so kubelet is limited to which statuses it can set. With this approach, there are mechanisms in place to ensure that kubelet updates status for Pods scheduled on that node.

## Infrastructure Needed (Optional)

We may need to update sample device plugin. No special infra is needed as emulating real GPU failures or failures in other devices is not practical.
