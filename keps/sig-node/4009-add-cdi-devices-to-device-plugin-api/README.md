# KEP-4009: Add CDI devices to device plugin API

<!-- toc -->
- [Release Signoff Checklist](#release-signoff-checklist)
- [Summary](#summary)
- [Motivation](#motivation)
  - [Goals](#goals)
- [Proposal](#proposal)
- [Design Details](#design-details)
  - [Test Plan](#test-plan)
      - [Prerequisite testing updates](#prerequisite-testing-updates)
      - [Unit tests](#unit-tests)
      - [Integration tests](#integration-tests)
      - [e2e tests](#e2e-tests)
  - [Graduation Criteria](#graduation-criteria)
    - [Alpha](#alpha)
    - [Alpha to Beta Graduation](#alpha-to-beta-graduation)
    - [Beta to G.A Graduation](#beta-to-ga-graduation)
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
<!-- /toc -->

## Release Signoff Checklist

Items marked with (R) are required *prior to targeting to a milestone / release*.

- [X] (R) Enhancement issue in release milestone, which links to KEP dir in [kubernetes/enhancements] (not the initial KEP PR)
- [X] (R) KEP approvers have approved the KEP status as `implementable`
- [X] (R) Design details are appropriately documented
- [ ] (R) Test plan is in place, giving consideration to SIG Architecture and SIG Testing input (including test refactors)
  - [X] e2e Tests for all Beta API Operations (endpoints)
  - [ ] (R) Ensure GA e2e tests meet requirements for [Conformance Tests](https://github.com/kubernetes/community/blob/master/contributors/devel/sig-architecture/conformance-tests.md)
  - [ ] (R) Minimum Two Week Window for GA e2e tests to prove flake free
- [X] (R) Graduation criteria is in place
  - [ ] (R) [all GA Endpoints](https://github.com/kubernetes/community/pull/1806) must be hit by [Conformance Tests](https://github.com/kubernetes/community/blob/master/contributors/devel/sig-architecture/conformance-tests.md)
- [x] (R) Production readiness review completed
- [x] (R) Production readiness review approved
- [X] "Implementation History" section is up-to-date for milestone
- [X] User-facing documentation has been created in [kubernetes/website], for publication to [kubernetes.io]
- [ ] Supporting documentation—e.g., additional design documents, links to mailing list discussions/SIG meetings, relevant PRs/issues, release notes

[kubernetes.io]: https://kubernetes.io/
[kubernetes/enhancements]: https://git.k8s.io/enhancements
[kubernetes/kubernetes]: https://git.k8s.io/kubernetes
[kubernetes/website]: https://git.k8s.io/website

## Summary

This KEP proposes extending the Device Plugin API, adding a field to specify
Container Device Interface (CDI) device IDs in the `AllocateResponse`. This
supplements the existing fields such as annotations and allows device plugin
implementations to uniquely specify devices using their fully-qualified CDI
devices names.

The recent addition of CDI device IDs to the CRI structures in [#3731](https://github.com/kubernetes/enhancements/pull/3731) allow these IDs to be forwarded to the CRI runtimes in a secure manner. Although
these changes were motivated by [KEP-3063](https://github.com/kubernetes/enhancements/issues/3063), adding support for these fields to the
existing device plugin API allows this mechanism to also be used for devices
supported by these plugins.

## Motivation

The Container Device Inteface (CDI) provides a standard mechanism for device
vendors to describe what is required to provide access to a specific resource
such as a GPU. These resources can be uniquely identified using a
fully-qualified CDI device name.

The changes proposed in [#3731]((https://github.com/kubernetes/enhancements/pull/3731)) extend the CRI to provide a well-defined mechanism for forwarding such
requests to CRI runtimes such as Containerd and Cri-o. These have already
been extended to accept CDI device requests, and to use the associated CDI
specifications to ensure that the required
modifications are made to the OCI runtime specification for a container being
launched.

The addition of an explicit field for specifying CDI device names to the Device
Plugin API allows this CRI field to be used to indicate which devices should be
injected. This removes the need to use workarounds such as container annotations
to pass this information to the runtimes and allows Device Plugin authors to
adopt CDI to inject devices without requiring that users move to a Dynamic
Resource Allocation (DRA) based implementation.

### Goals

* Allow Device Plugin authors to forward device requests to CRI runtimes as a CRI field.
* Allow Device Plugin authors to use CDI to define the modifications required for containerised environments.

## Proposal

We propose a mechanism for device plugin authors to specify devices using Container Device Interface (CDI) names. The names of the requested devices are passed down as CRI fields to CRI runtimes which are ultimately responsible for making the requested devices accessible from a container.

## Design Details

This adds a repeated `CDIDevice` field to the exiting `ContainerAllocateResponse` returned as part of the
`AllocateResponse` in the Device Plugin API. This matches the modifications made to the Dynamic Resource Allocation API in [#3731](https://github.com/kubernetes/enhancements/pull/3731).

The values contained in this field are then used to populate the corresponding field in the CRI
which is passed to the container runtimes. In addition, annotations with a `cdi.k8s.io` prefix will be
added to the CRI to allow for consumption in container runtimes that do not yet support the
CRI field directly, but do support device requests through annotations.

```protobuf
// CDIDevice specifies a CDI device information.
message CDIDevice {
	// Fully qualified CDI device name
	// for example: vendor.com/gpu=gpudevice1
	// see more details in the CDI specification:
	// https://github.com/container-orchestrated-devices/container-device-interface/blob/main/SPEC.md
	string name = 1;
}

message ContainerAllocateResponse {
	// List of environment variable to be set in the container to access one of more devices.
	map<string, string> envs = 1;
	// Mounts for the container.
	repeated Mount mounts = 2;
	// Devices for the container.
	repeated DeviceSpec devices = 3;
	// Container annotations to pass to the container runtime
	map<string, string> annotations = 4;
	// CDI devices for the container.
	repeated CDIDevice cdi_devices = 5;
}
```

### Test Plan

[x] I/we understand the owners of the involved components may require updates to
existing tests to make this code solid enough prior to committing the changes necessary
to implement this enhancement.

##### Prerequisite testing updates

##### Unit tests

- `devicemanager`: `2023-06-15` - `85.1%`

##### Integration tests

There are currently no integration tests for device plugins.
We do not plan to add any for this feature.

However, these cases will be added in the existing integration tests:
  - Feature gate enable/disable tests

##### e2e tests

This test case has been added to the existing `e2e_node` tests:
  - DevicePlugin can make a CDI device accessible in a container

Links to test grid:
- https://testgrid.k8s.io/sig-node-cri-o#ci-crio-cdi-device-plugins

Links to k8s-triage for tests:
- https://storage.googleapis.com/k8s-triage/index.html?job=ci-crio-cdi-device-plugins

### Graduation Criteria

#### Alpha
- [X] Add the CDIDevices field to the device plugin API
- [X] Implement the logic to pass the CDIDevices into the CRI
- [X] Add proper `e2e_node` tests

#### Alpha to Beta Graduation
- [X] No major bugs reported in the previous cycle

#### Beta to G.A Graduation
- [X] Gather feedback from at least 2 device plugin vendors that CDI support works for them

### Upgrade / Downgrade Strategy

We expect no impact on upgrades.
On downgrades, we expect no impact to Kubernetes and minimal impact to device
plugin developers.

We are not bumping the device plugin API version, but simply adding a field to
its protobuf. On upgrades this means that older device plugins will simply
continue to work as they always have, since they will need to opt-in to using
this new field.

For downgrades, if a plugin has not opted to use the new field, there will be
no impact since a downgraded kubelet won't support it anyway. If a device
plugin has opted-in to use the new field, a downgraded kubelet will simply
silently ignore it. This would have no impact to Kubernetes itself, but the
plugin developer would need to be aware of this if they are confused as to why
their new CDI support is suddenly not working anymore.

### Version Skew Strategy

The kubelet will always be backwards compatible, so going forward existing
plugins are not expected to break.

## Production Readiness Review Questionnaire

### Feature Enablement and Rollback

###### How can this feature be enabled / disabled in a live cluster?

- [x] Feature gate (also fill in values in `kep.yaml`)
  - Feature gate names:
    - `DevicePluginCDIDevices`
  - Components depending on the feature gate: kubelet
- [x] Pass CDI devices to the kubelet over the new field in the device plugin API
  - Will enabling / disabling the feature require downtime of the control
    plane?
    No.
  - Will enabling / disabling the feature require downtime or reprovisioning
    of a node?
    No.


###### Does enabling the feature change any default behavior?

No. Device Plugins need to be updated to make use of the new field.

###### Can the feature be disabled once it has been enabled (i.e. can we roll back the enablement)?

- Yes, disabling the `DevicePluginCDIDevices` feature gate shuts down the feature completely.
- Yes, by not sending CDI devices over the device plugin API (and falling back to the old way of passing device info).

###### What happens if we reenable the feature if it was previously rolled back?

Nothing bad will happen, new containers will simply be able to be started with
CDI devices again.

###### Are there any tests for feature enablement/disablement?

There will be e2e tests demonstrating that CDI devices are attached as expected
when the feature is enabled, and silently ignored if the feature is disabled.

### Rollout, Upgrade and Rollback Planning

###### How can a rollout or rollback fail? Can it impact already running workloads?

The failure of the kubelet would mean that fields from new device allocations
will not be processed.

However, CDI device themselves are only interpereted at container start.
Existing containers that were started with support for CDI devices will not be
impacted if the feature gate is enabled or disabled during the lifetime of a
running container. Only new containers will be impacted by the presence or
absence of the feature gate.

###### What specific metrics should inform a rollback?

N/A

###### Were upgrade and rollback tested? Was the upgrade->downgrade->upgrade path tested?

N/A

###### Is the rollout accompanied by any deprecations and/or removals of features, APIs, fields of API types, flags, etc.?

No

### Monitoring Requirements

###### How can an operator determine if the feature is in use by workloads?

This depends on Device Plugin vendor implementations making use of the required
field and cannot be directly determined.

###### How can someone using this feature know that it is working for their instance?

End-users are not aware that this feature exists. Device plugin developers can
ensure that this feature is working by passing CDI devices to workloads
requesting them, and ensuring that the workloads come up successfully with
access to the devices they asked for.

###### What are the reasonable SLOs (Service Level Objectives) for the enhancement?

N/A

###### What are the SLIs (Service Level Indicators) an operator can use to determine the health of the service?

N/A

###### Are there any missing metrics that would be useful to have to improve observability of this feature?

N/A

### Dependencies

###### Does this feature depend on any specific services running in the cluster?

- The container runtime (e.g. containerd, crio-o, etc.) must support CDI.
- A Device Plugin must be implemented to use the field.

### Scalability

###### Will enabling / using this feature result in any new API calls?

No

###### Will enabling / using this feature result in introducing new API types?

No

###### Will enabling / using this feature result in any new calls to the cloud provider?

No

###### Will enabling / using this feature result in increasing size or count of the existing API objects?

No

###### Will enabling / using this feature result in increasing time taken by any operations covered by existing SLIs/SLOs?

No

###### Will enabling / using this feature result in non-negligible increase of resource usage (CPU, RAM, disk, IO, ...) in any components?

No. The additional field will replace existing usages where used.

###### Can enabling / using this feature result in resource exhaustion of some node resources (PIDs, sockets, inodes, etc.)?

No

### Troubleshooting

###### How does this feature react if the API server and/or etcd is unavailable?

N/A

###### What are other known failure modes?

The change to Kubernetes to support this feature is very minimal. The CDI
device list passed from the plugin to the kubelet is opaquely forwarded to the
underlying container runtime without affecting the overall logic of the kubelet
in any significant way. As such, the only known failure scenarios result from
plugins themselves doing something incorrectly (not the kubelet). For example,
sending back a list of CDI devices that are not included in any CDI spec
visible to the underlying container runtime. However, such failure scenarios do
not affect the proper functioning of kubernetes itself, and are therefore out
of scope for this KEP. We recommend you check the device plugin and container
runtime logs instead.

###### What steps should be taken if SLOs are not being met to determine the problem?

N/A

## Implementation History

- 2023-05-15: KEP created
- 2023-09-25: KEP updated to mark transition to beta
- 2024-01-24: KEP updated to mark transition to stable

## Drawbacks

There is no reason this KEP should not be implemented. CDI is the new standard
for device support in containerized environments, and this enhancement now
makes this possible through a simple addition to the device plugin API.

## Alternatives

None
