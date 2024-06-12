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
- [ ] (R) KEP approvers have approved the KEP status as `implementable`
- [ ] (R) Design details are appropriately documented
- [ ] (R) Test plan is in place, giving consideration to SIG Architecture and SIG Testing input (including test refactors)
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

```
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
```

One field reflects the resource requests and limits and the other actual allocated resources.

This structure will contain standard resources as well as extended resources. As noted in the comment: https://github.com/kubernetes/kubernetes/pull/124227#issuecomment-2130503713, it is only logical to also include the status of those allocated resources. 

The proposal is to keep this structure as-is to simplify parsing of well-known ResourceList data type by various consumers. Typical scenario would be to compare if the `AllocatedResources` match the desired state.

The proposal is to introduce an additional field:

```
// AllocatedResourcesStatus represents the status of various resources
// allocated for this Pod.
AllocatedResourcesStatus ResourcesStatus 

type ResourcesStatus map[ResourceName]ResourceStatus

type ResourceStatus struct {
    // map of unique Resource ID to its health.
    // At a minimum, ResourceID must uniquely identify the Resource
    // allocated to the Pod on the Node for the lifetime of a Pod.
    // See ResourceID type for it's definition.
    Resources map[ResourceID] ResourceHealth
    
    // allow to extend this struct in future with the overall health fields or things like Device Plugin version
}

// ResourceID is calculated based on the source of this resource health information.
// For DevicePlugin:
//   deviceplugin:DeviceID, where DeviceID is from the Device structure of DevicePlugin's ListAndWatchResponse type: https://github.com/kubernetes/kubernetes/blob/eda1c780543a27c078450e2f17d674471e00f494/staging/src/k8s.io/kubelet/pkg/apis/deviceplugin/v1alpha/api.proto#L61-L73
// DevicePlugin ID is usually a constant for the lifetime of a Node and typically can be used to uniquely identify the device on the node.
// For DRA:
//   dra:<driver name>[/<pool name>]/<device name>: such a device can be looked up in the information published by that DRA driver to learn more about it. It is designed to be globally unique in a cluster.
type ResourceID string

type ResourceHealth struct {
    // List of conditions with the transition times
    Conditions []ResourceHealthCondition
}

// This condition type is replicating other condition types exposed by various status APIs
type ResourceHealthCondition struct {
    // can be one of:
    //  - Healthy: operates as normal
    //  - Unhealthy: reported unhealthy. We consider this a temporary health issue
    //               since we do not have a mechanism today to distinguish
    //               temporary and permanent issues.
    //  - Unknown: The status cannot be determined.
    //             For example, Device Plugin got unregistered and hasn't been re-registered since.
    //
    // In future we may want to introduce the PermanentlyUnhealthy Status.
    Type string 

    // Status of the condition, one of True, False, Unknown.
    Status ConditionStatus
    // The last time the condition transitioned from one status to another.
    // +optional
    LastTransitionTime metav1.Time
    // The reason for the condition's last transition.
    // +optional
    Reason string
    // A human readable message indicating details about the transition.
    // +optional
    Message string
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

Today DRA does not return the health of the device back to kubelet. The proposal is to extend the

type `NamedResourcesInstance` (from [pkg/apis/resource/namedresources.go](https://github.com/kubernetes/kubernetes/blob/790dfdbe386e4a115f41d38058c127d2dd0e6f44/pkg/apis/resource/namedresources.go#L29-L37)) to include the Health field the same way it is done in 
the Device Plugin as well as a device ID.

In `1.30` we had a similar `ListAndWatch()` API as in DevicePlugin, from which we could have inferred something very analogous to the above. However, we are removing this in `1.31`, so will need to provide something different.

An optional gRPC interface will be created, so DRA drivers can opt into this by implementing it. The interface will allow a plugin to stream health status information in the form of deviceIDs (of the form `<driver name>/<pool name>/<device name>`) along with extra metadata indicating its health status. Just as before, a device completely disappearing would still need to trigger some state change, but now more detailed information could be attached in the form of metadata when a device isn't necessarily gone, but also isn't operating as it should be.

The API will be limited to "prepared" devices and include the claim `name/namespace/UID`. That should be enough information for kubelet to correlate with the pods for which the claim was prepared and then post that information for those pods.

Kubelet will react on this field the same way as we propose to do it for the Device Plugin.

Specific implementation details will be added for the beta.

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

##### Integration tests

N/A

##### e2e tests

Planned tests:

- Device marked unhealthy - the state is reflected in pod status
- Device marked unhealthy and back to healthy after some time - pod status was changed to unhealthy temporarily
- Device marked as unhealthy and back to healthy in quick succession - pod status reflects the latest health status
- Pod failed due to unhealthy device, earlier than device plugin detected it. Pod status is still updated.
- Pod is in crash loop backoff due to unhealthy device - pod status is updated to unhealthy

Test coverage will be listed once tests are implemented.

- <test>: <link to test coverage>

### Graduation Criteria

#### Alpha

- New field is introduced in Pod Status
- Feature implemented in Device Manager behind a feature flag
- Initial e2e tests completed and enabled

#### Beta

- Feature is implemented in DRA behind the feature flag
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

Yes, with no side effect except of missing the new field in pod status.

###### What happens if we reenable the feature if it was previously rolled back?

The pod status will be updated again.

###### Are there any tests for feature enablement/disablement?

Nothing is planned.

### Rollout, Upgrade and Rollback Planning

###### How can a rollout or rollback fail? Can it impact already running workloads?

No

###### What specific metrics should inform a rollback?

N/A

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

N/A

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
