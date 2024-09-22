# KEP-4817: Resource Claim Device Status

<!-- toc -->
- [Release Signoff Checklist](#release-signoff-checklist)
- [Summary](#summary)
- [Motivation](#motivation)
  - [Goals](#goals)
  - [Non-Goals](#non-goals)
- [Proposal](#proposal)
  - [API - ResourceClaim.Status](#api---resourceclaimstatus)
  - [User Stories (Optional)](#user-stories-optional)
    - [Story 1 - Network Device Status for Network Services](#story-1---network-device-status-for-network-services)
    - [Story 2 - Network Device Status for Troubleshooting](#story-2---network-device-status-for-troubleshooting)
  - [Notes/Constraints/Caveats (Optional)](#notesconstraintscaveats-optional)
  - [Risks and Mitigations](#risks-and-mitigations)
- [Design Details](#design-details)
  - [API](#api)
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
  - [Annotations](#annotations)
  - [Pod.Status.PodIPs Enhancement](#podstatuspodips-enhancement)
  - [New Pod.Status Field](#new-podstatus-field)
  - [KEP-4680 extension](#kep-4680-extension)
- [Infrastructure Needed (Optional)](#infrastructure-needed-optional)
<!-- /toc -->

## Release Signoff Checklist

Items marked with (R) are required *prior to targeting to a milestone / release*.

- [ ] (R) Enhancement issue in release milestone, which links to KEP dir in [kubernetes/enhancements] (not the initial KEP PR)
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

This proposal enhances the `ResourceClaim.Status` by adding a new field
`DeviceStatuses`. The new field allows drivers to report driver-specific device
status data for each allocated devices in a resource claim. Allowing the
drivers to report the device statuses will improve both observability and
troubleshooting as well as enabling new functionalities such as, for example,
if the IPs of a network device are reported, network services.

This extension also lays the foundation for a potential future standardization
of specific type device data, such as, for example, networking devices
information.

## Motivation

As of now, when a device is configured in a pod/container, the state and
characteristics of the device set during the configuration stage are invisible.
The lack of this information is then a challenge for the users and developers
to diagnose issues, verify configurations, and integrate the allocated
resources into higher-level services. Reporting this information is then
crucial in environments where device specific configurations are necessary.

For certain types of devices, such as network interfaces, knowing the detailed
characteristics of what has been allocated is particularly useful and crucial.
For example, reporting the interface Name, MAC address and IP addresses of
network interfaces in the status of a ResourceClaim can significantly help in
configuring and managing network services, as well as in debugging network
related issues. By including such device specific information, this proposal
addresses existing gaps in visibility and facilitates better integration and
management of resources.

### Goals

* Allow arbitrary, driver-specific information to be reported from the DRA
  drivers for each allocated device in a ResourceClaim.
* Establish a foundation for potential future standardization (e.g. Network
  Devices status).
* Enable 3rd party implementation of new functionalities based on the Device
  Status (e.g. Secondary Network Services if the IPs of a network device are
reported).

### Non-Goals

* Implement new functionalities based on the Device Status (e.g. Kubernetes
  EndpointSlice Controller supporting IPs reported by networking DRA drivers).
* Modifying the Resource Claim workflow to include the device status report.

## Proposal

### API - ResourceClaim.Status

The API changes define a new `DeviceStatuses` field in the existing
`ResourceClaimStatus` struct. `DeviceStatuses` is a slice of a new struct
`AllocatedDeviceStatus` which holds device specific information. 

A device, identified by `<driver name>/<pool name>/<device name>` can be
represented only once in the `DeviceStatuses` slice and will also mention which
request caused the device to be allocated. The state and characteristics of the
device are reported in the `Conditions`, representing the operational state of
the device and in the `DeviceInfo`, an arbitrary data slice representing device
specific characteristics. Additionally, for networking devices, a field
`NetworkDeviceInfo` can be used to report the IPs, the MAC address and the
interface name.

`DeviceInfo` being a slice of arbitrary data allows the DRA Driver to store
device specific data in different formats. For example, a Network Device being
configured by via a CNI plugin could get its `DeviceInfo` filled with the CNI
result for troubleshooting purpose and with a Network result in a modern
standard format (closer to Pod.Status.PodIPs for example) used by 3rd party
controllers.

For each device, if required, the DRA Driver processing the device allocation
can report the status of it in the `Status.DeviceStatuses` of the ResourceClaim
by using the Kubernetes API.

```golang
// ResourceClaimStatus tracks whether the resource has been allocated and what
// the result of that was.
type ResourceClaimStatus struct {
	...

	// DeviceStatuses contains the status of each device allocated for this
	// claim, as reported by the driver. This can include driver-specific
	// information. Entries are owned by their respective drivers.
	//
	// +optional
	// +listType=map
	// +listMapKey=devicePoolName
	// +listMapKey=deviceName
	DeviceStatuses []AllocatedDeviceStatus `json:"deviceStatuses,omitempty" protobuf:"bytes,4,opt,name=deviceStatuses"`
}


// AllocatedDeviceStatus contains the status of an allocated device, if the
// driver chooses to report it. This may include driver-specific information.
type AllocatedDeviceStatus struct {
	// Request is the name of the request in the claim which caused this
	// device to be allocated. Multiple devices may have been allocated
	// per request.
	//
	// +required
	Request string `json:"request" protobuf:"bytes,1,rep,name=request"`

	// Driver specifies the name of the DRA driver whose kubelet
	// plugin should be invoked to process the allocation once the claim is
	// needed on a node.
	//
	// Must be a DNS subdomain and should end with a DNS domain owned by the
	// vendor of the driver.
	//
	// +required
	Driver string `json:"driver" protobuf:"bytes,2,rep,name=driver"`

	// This name together with the driver name and the device name field
	// identify which device was allocated (`<driver name>/<pool name>/<device name>`).
	//
	// Must not be longer than 253 characters and may contain one or more
	// DNS sub-domains separated by slashes.
	//
	// +required
	Pool string `json:"pool" protobuf:"bytes,3,rep,name=pool"`

	// Device references one device instance via its name in the driver's
	// resource pool. It must be a DNS label.
	//
	// +required
	Device string `json:"device" protobuf:"bytes,4,rep,name=device"`

	// Conditions contains the latest observation of the device's state.
	// If the device has been configured according to the class and claim
	// config references, the `Ready` condition should be True.
	//
	// +optional
	// +listType=atomic
	Conditions []metav1.Condition `json:"conditions" protobuf:"bytes,5,rep,name=conditions"`

	// DeviceInfo contains Arbitrary driver-specific data.
	//
	// +optional
	// +listType=atomic
	DeviceInfo []runtime.RawExtension `json:"deviceInfo,omitempty" protobuf:"bytes,6,rep,name=deviceInfo"`

    // NetworkDeviceInfo contains network-related information specific to the device.
	//
	// +optional
	NetworkDeviceInfo NetworkDeviceInfo `json:"networkDeviceInfo,omitempty" protobuf:"bytes,7,rep,name=networkDeviceInfo"`
}

// NetworkDeviceInfo provides network-related details for the allocated device.
// This information may be filled by drivers or other components to configure
// or identify the device within a network context.
type NetworkDeviceInfo struct {
	// Interface specifies the name of the network interface associated with
	// the allocated device. This might be the name of a physical or virtual
	// network interface.
	//
	// +optional
	Interface string `json:"interface,omitempty" protobuf:"bytes,1,rep,name=interface"`

	// IPs lists the IP addresses assigned to the device's network interface.
	// This can include both IPv4 and IPv6 addresses.
	//
	// +optional
	IPs []string `json:"ips,omitempty" protobuf:"bytes,2,rep,name=ips"`

	// Mac represents the MAC address of the device's network interface.
	//
	// +optional
	Mac string `json:"mac,omitempty" protobuf:"bytes,3,rep,name=mac"`
}
```

### User Stories (Optional)

#### Story 1 - Network Device Status for Network Services

As a Cloud Native Network Function (CNF) vendor, my network services must be
integrated with the network devices configured in Pods. The configuration
properties of these network devices are therefore essential to configure the
network services. For example, the network services must be able to route
traffic to pods over networks attached via the network devices, the IP
addresses of the network device(s) must then be reflected in the
`ResourceClaim.Status` allowing the network service controller(s) to access
them.

#### Story 2 - Network Device Status for Troubleshooting

As a Network Administrator, troubleshooting networking issues can be complex
and time consuming especially when the device characteristics and operational
status are not readily accessible. The `DeviceStatuses` field in the
`ResourceClaim.Status` provides access to comprehensive details regarding
network interfaces helping to quickly and efficiently identify the issues such
as error messages on failed network interface configuration, incorrect IP
assignments or misconfigured network interfaces.

### Notes/Constraints/Caveats (Optional)

The content of `DeviceInfo` is driver specific and not standardized as part of
DRA, the interpretation of this field may then vary between controllers and
users reading it. 

The accuracy of the information depends on the implementation of the DRA
Drivers, the reported status of the device may not always reflect the real time
changes of the state of the device.

### Risks and Mitigations

As stated, 3rd party DRA drivers will set and update the `DeviceStatuses` for
the device they manage. An access control must be set in place to restrict the
write access to the appropriate driver (A device status can only be updated by
the driver which allocated and configured this device).

Adding `DeviceInfo` as an arbitrary data slice may introduce extra processing
and storage overhead which might impact performance in a cluster with many
devices and frequent status updates. In large-scale clusters where many devices
are allocated, this impact must be considered.

## Design Details

### API

The `ResourceClaimStatus` struct in `pkg/apis/resource/types.go` will be
extended to include the slice of `DeviceStatuses`.

`ResourceClaim` validation of the status in
`pkg/apis/resource/validation/validation.go` will be covered to allow a device
to be reported only once in the slice, a device is being identified  by
`<driver name>/<pool name>/<device name>`.

### Test Plan

[x] I/we understand the owners of the involved components may require updates to
existing tests to make this code solid enough prior to committing the changes necessary
to implement this enhancement.

##### Prerequisite testing updates

##### Unit tests

- `ResourceClaim` validation:
    - A device can only be reported once in the `ResourceClaim`.
    - The reported device is allocated in the `ResourceClaim`.
    - Properties set in `AllocatedDeviceStatus` are in the correct format.

##### Integration tests

- Usage of the `DeviceStatuses` field in the `ResourceClaimStatus`:
    * With the feature gate enabled, the field exists in the `ResourceClaim`.
    * With the feature gate disabled, the field does not exist in the
      `ResourceClaim`.

##### e2e tests

TBD

### Graduation Criteria

#### Alpha

- Feature implemented behind feature gates (`ResourceClaimDeviceStatus`).
  Feature Gates are disabled by default.
- Documentation provided.
- Initial unit, integration and e2e tests completed and enabled.

#### Beta

- Feature Gates are enabled by default.
- Authorization implemented to allow only the driver managing the device to
  write the status.
- No major outstanding bugs.
- Feedback collected from the community (developers and users) with adjustments
  provided, implemented and tested.

#### GA

- 2 examples of real-world usage.
- Allowing time for feedback from developers and users.

### Upgrade / Downgrade Strategy

This feature only exposes a new field in the `ResourceClaim.Status`, the field
will either be present or not.

DRA implementation requires DRA interfaces change. DRA is in alpha and in
active development. The feature will follow the DRA upgrade/downgrade strategy.

### Version Skew Strategy

This feature affects only the kube-apiserver, so there is no issue with version
skew with other Kubernetes components.

## Production Readiness Review Questionnaire

### Feature Enablement and Rollback

###### How can this feature be enabled / disabled in a live cluster?

- [x] Feature gate (also fill in values in `kep.yaml`)
  - Feature gate name: ResourceClaimDeviceStatus
  - Components depending on the feature gate: kube-apiserver
- [ ] Other
  - Describe the mechanism:
  - Will enabling / disabling the feature require downtime of the control
    plane?
  - Will enabling / disabling the feature require downtime or reprovisioning of
    a node?

###### Does enabling the feature change any default behavior?

No

###### Can the feature be disabled once it has been enabled (i.e. can we roll back the enablement)?

Yes, with no side effects except for missing the new field in
`ResourceClaim.Status`. Re-enabling this feature will not guarantee to keep
the values written before the feature has been disabled.

###### What happens if we reenable the feature if it was previously rolled back?

###### Are there any tests for feature enablement/disablement?

Enablement/disablement of this feature is tested as part of the integration
tests.

### Rollout, Upgrade and Rollback Planning

No

###### How can a rollout or rollback fail? Can it impact already running workloads?

No

###### What specific metrics should inform a rollback?

N/A

###### Were upgrade and rollback tested? Was the upgrade->downgrade->upgrade path tested?

N/A

###### Is the rollout accompanied by any deprecations and/or removals of features, APIs, fields of API types, flags, etc.?

No

### Monitoring Requirements

###### How can an operator determine if the feature is in use by workloads?

Check the `ResourceClaim.Status.DeviceStatuses`.

###### How can someone using this feature know that it is working for their instance?

- [ ] Events
  - Event Reason: 
- [x] API .status
  - Condition name: 
  - Other field: `ResourceClaim.Status.DeviceStatuses`
- [ ] Other (treat as last resort)
  - Details:

###### What are the reasonable SLOs (Service Level Objectives) for the enhancement?

N/A

###### What are the SLIs (Service Level Indicators) an operator can use to determine the health of the service?

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

[KEP-4381 - DRA Structured Parameters](https://github.com/kubernetes/enhancements/issues/4381)

###### Does this feature depend on any specific services running in the cluster?

No

### Scalability

###### Will enabling / using this feature result in any new API calls?

Yes, DRA Drivers will update the `ResourceClaim.Status` to report the allocated
device status. Depending on the driver the size and frequency of these updates
can vary.

###### Will enabling / using this feature result in introducing new API types?

New field on `ResourceClaim.Status`.

###### Will enabling / using this feature result in any new calls to the cloud provider?

No

###### Will enabling / using this feature result in increasing size or count of the existing API objects?

`ResourceClaim.Status` size will increase.

###### Will enabling / using this feature result in increasing time taken by any operations covered by existing SLIs/SLOs?

No

###### Will enabling / using this feature result in non-negligible increase of resource usage (CPU, RAM, disk, IO, ...) in any components?

Depending on the content of the `DeviceInfo` set by the DRA drivers, the disk
usage could increase.

###### Can enabling / using this feature result in resource exhaustion of some node resources (PIDs, sockets, inodes, etc.)?

No

### Troubleshooting

###### How does this feature react if the API server and/or etcd is unavailable?

N/A

###### What are other known failure modes?

N/A

###### What steps should be taken if SLOs are not being met to determine the problem?

N/A

## Implementation History

- Initial proposal: 2024-08-30

## Drawbacks

If the Network device (network interface) characteristics (e.g. IP) and status
is reported as part of the `Resource.Claim.Status`, it should be ensured the
`ResourceClaim` is not used by several `Pod` at a time. Additionally, if a
controller needs to gather IPs for a specific network to which Pods are
attached via networking devices, it will need to query each `Pod` and then
access the corresponding `ResourceClaim` for every Pod.

## Alternatives

### Annotations

An option the DRA drivers can currently use to report the status of the device
allocated in the `ResourceClaim` is the annotation of the `ResourceClaim` or of
the `Pod` itself. As a reference, the [k8snetworkplumbingwg/Multus-CNI](https://github.com/k8snetworkplumbingwg/multus-cni) project is utilizing annotation to describe the network attachments/interfaces
and report the status.

Here is the API below representing a network attachment. This is stored as a
list in a json format in the annotation of the `Pod`.

[k8snetworkplumbingwg/network-attachment-definition-client/pkg/apis/k8s.cni.cncf.io/v1/types.go](https://github.com/k8snetworkplumbingwg/network-attachment-definition-client/blob/v1.7.3/pkg/apis/k8s.cni.cncf.io/v1/types.go#L103):
```golang
// NetworkStatus is for network status annotation for pod
// +k8s:deepcopy-gen=false
type NetworkStatus struct {
	Name       string      `json:"name"`
	Interface  string      `json:"interface,omitempty"`
	IPs        []string    `json:"ips,omitempty"`
	Mac        string      `json:"mac,omitempty"`
	Mtu        int         `json:"mtu,omitempty"`
	Default    bool        `json:"default,omitempty"`
	DNS        DNS         `json:"dns,omitempty"`
	DeviceInfo *DeviceInfo `json:"device-info,omitempty"`
	Gateway    []string    `json:"gateway,omitempty"`
}
```

### Pod.Status.PodIPs Enhancement

As part of the [Multi-Network (KEP-3698)](https://github.com/kubernetes/enhancements/issues/3698), 
the idea was to use the existing `Pod.Status.PodIPs` and save the data about the
different network interfaces/devices attached to the `Pod`. As part of the
review of the KEP, it has been indicated ([here](https://github.com/kubernetes/enhancements/pull/3700#discussion_r1501690793) and [here](https://github.com/kubernetes/kubernetes/pull/123112#issuecomment-1925957930))
that it would be an API breaking change if the `Pod.Status.PodIPs` contains
more than 1 value per IP family.

### New Pod.Status Field

Still as part of the [KEP-3698 - Multi-Network](https://github.com/kubernetes/enhancements/issues/3698), and in
the continuation of the previous alternative, the idea was to add a new field
`Networks` in the `Pod.Status` so each networking DRA driver could report the
status for each network interface/device directly in the `Pod.Status`.

Here is below the proposed API:
```golang
// PodStatus represents information about the status of a pod. Status may trail the actual
// state of a system, especially if the node that hosts the pod cannot contact the control
// plane.
type PodStatus struct {
[...]
        // Networks is a list of PodNetworks that are attached to the Pod.
        //
        // +optional
        Networks []NetworkStatus `json:"networks,omitempty"`
}

// NetworkStatus provides the status of specific PodNetwork in a Pod.
type NetworkStatus struct {
        // Name is name of PodNetwork
        Name string `json:"name"`

        // InterfaceName is the network interface name inside the Pod for this attachment.
        // Examples: eth1 or net1
        //
        // +optional
        InterfaceName string `json:"interfaceName"`

        // ip is an IP address (IPv4 or IPv6) assigned to the pod
        IP string `json:"ip,omitempty"`

        // IsDefaultGW is a flag indicating that the interface with this IP
        // inside the Pod holds the Default Gateway.
        //
        // +optional
        IsDefaultGW bool `json:"isDefaultGW,omitempty"`
}
```

### KEP-4680 extension

During the WG Device Management meeting on 17th of September 2024 ([Slack
summary](https://kubernetes.slack.com/archives/C0409NGC1TK/p1726679433650409)),
the idea was to extend the [KEP-4680 about resource health status in the
`Pod.Status` ](https://github.com/kubernetes/enhancements/issues/4680) in order
to expose device information and not just the health.

## Infrastructure Needed (Optional)

<!--
Use this section if you need things from the project/SIG. Examples include a
new subproject, repos requested, or GitHub details. Listing these here allows a
SIG to get the process for these resources started right away.
-->
