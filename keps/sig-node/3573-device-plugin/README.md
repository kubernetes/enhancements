# KEP-3573: Device Manager Proposal

<!-- toc -->
- [Release Signoff Checklist](#release-signoff-checklist)
- [Summary](#summary)
- [Motivation](#motivation)
  - [Use Cases](#use-cases)
  - [Goals](#goals)
  - [Non-Goals](#non-goals)
- [Proposal](#proposal)
  - [User Stories](#user-stories)
    - [Vendor story](#vendor-story)
    - [End User story](#end-user-story)
  - [Notes/Constraints/Caveats (Optional)](#notesconstraintscaveats-optional)
  - [Risks and Mitigations](#risks-and-mitigations)
- [Design Details](#design-details)
  - [Introduction](#introduction)
  - [Registration](#registration)
  - [Unix Socket](#unix-socket)
  - [Protocol Overview](#protocol-overview)
  - [API Specification](#api-specification)
  - [HealthCheck and Failure Recovery](#healthcheck-and-failure-recovery)
  - [API Changes](#api-changes)
  - [Installation](#installation)
  - [Test Plan](#test-plan)
      - [Prerequisite testing updates](#prerequisite-testing-updates)
      - [Unit tests](#unit-tests)
      - [Integration tests](#integration-tests)
      - [e2e tests](#e2e-tests)
  - [Graduation Criteria](#graduation-criteria)
    - [Alpha](#alpha)
    - [Beta](#beta)
    - [GA](#ga)
    - [Deprecation](#deprecation)
  - [Upgrade / Downgrade Strategy](#upgrade--downgrade-strategy)
    - [Upgrading your cluster](#upgrading-your-cluster)
      - [Upgrading Kubelet](#upgrading-kubelet)
      - [Upgrading Device Plugins](#upgrading-device-plugins)
  - [Version Skew Strategy](#version-skew-strategy)
- [Production Readiness Review Questionnaire](#production-readiness-review-questionnaire)
  - [Feature Enablement and Rollback](#feature-enablement-and-rollback)
  - [Rollout, Upgrade and Rollback Planning](#rollout-upgrade-and-rollback-planning)
  - [Monitoring Requirements](#monitoring-requirements)
  - [Dependencies](#dependencies)
  - [Scalability](#scalability)
  - [Troubleshooting](#troubleshooting)
- [Implementation History](#implementation-history)
- [References](#references)
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
  - [X] e2e Tests for all Beta API Operations (endpoints)
  - [ ] (R) Ensure GA e2e tests meet requirements for [Conformance Tests](https://github.com/kubernetes/community/blob/master/contributors/devel/sig-architecture/conformance-tests.md) 
  - [ ] (R) Minimum Two Week Window for GA e2e tests to prove flake free
- [X] (R) Graduation criteria is in place
  - [ ] (R) [all GA Endpoints](https://github.com/kubernetes/community/pull/1806) must be hit by [Conformance Tests](https://github.com/kubernetes/community/blob/master/contributors/devel/sig-architecture/conformance-tests.md) 
- [X] (R) Production readiness review completed
- [ ] (R) Production readiness review approved
- [X] "Implementation History" section is up-to-date for milestone
- [X] User-facing documentation has been created in [kubernetes/website], for publication to [kubernetes.io]
- [ ] Supporting documentation—e.g., additional design documents, links to mailing list discussions/SIG meetings, relevant PRs/issues, release notes


[kubernetes.io]: https://kubernetes.io/
[kubernetes/enhancements]: https://git.k8s.io/enhancements
[kubernetes/kubernetes]: https://git.k8s.io/kubernetes
[kubernetes/website]: https://git.k8s.io/website

## Summary

Device Manager Proposal is for a user to be able to enable vendor devices (e.g: GPUs) through
the following simple steps:
  * `kubectl create -f http://vendor.com/device-plugin-daemonset.yaml`
  * When launching `kubectl describe nodes`, the devices appear in the node
    status as `vendor-domain/vendor-device`. Note: naming
    convention is discussed in PR [#844](https://github.com/kubernetes/community/pull/844)

## Motivation

Kubernetes currently supports discovery of CPU and Memory primarily to a
minimal extent. Very few devices are handled natively by Kubelet.

It is not a sustainable solution to expect every hardware vendor to add their
vendor specific code inside Kubernetes to make their devices usable.

Instead, we want a solution for vendors to be able to advertise their resources
to Kubelet and monitor them without writing custom Kubernetes code.
We also want to provide a consistent and portable solution for users to
consume hardware devices across k8s clusters.

This document describes a vendor independent solution to:
  * Discovering and representing external devices
  * Making these devices available to the containers, using these devices,
    scrubbing and securely sharing these devices.
  * Health Check of these devices

Because devices are vendor dependent and have their own sets of problems
and mechanisms, the solution we describe is a plugin mechanism that may run
in a container deployed through the DaemonSets mechanism or in bare metal mode.

The targeted devices include GPUs, High-performance NICs, FPGAs, InfiniBand,
Storage devices, and other similar computing resources that require vendor
specific initialization and setup.

### Use Cases

 * I want to use a particular device type (GPU, InfiniBand, FPGA, etc.)
   in my pod.
 * I should be able to use that device without writing custom Kubernetes code.
 * I want a consistent and portable solution to consume hardware devices
   across k8s clusters.

### Goals

1. Add support for vendor specific Devices in kubelet:
    * Through an extension mechanism.
    * Which allows discovery and health check of devices.
    * Which allows hooking the runtime to make devices available in containers
      and cleaning them up.
1. Define a deployment mechanism for this new API.
1. Define a versioning mechanism for this new API.

### Non-Goals

1. Handling heterogeneous nodes and topology related problems
1. Collecting metrics is not part of this proposal. We will only solve
   Health Check.

## Proposal

At their core, device plugins are simple gRPC servers that may run in a
container deployed through the pod mechanism or in bare metal mode.

These servers implement the gRPC interface defined later in this design
document and once the device plugin makes itself known to kubelet, kubelet
will interact with the device through simple functions. In v1.8, the two
functions were introduced:
  1. A `ListAndWatch` function for the kubelet to Discover the devices and
     their properties as well as notify of any status change (device
     became unhealthy).
  2. An `Allocate` function which is called before creating a user container
     consuming any exported devices

![Process](device-plugin-overview.png)

In the subsequent releases, additional gRPC functions were added:
  1. `GetDevicePluginOptions` is used by device plugins to communicate
     options to the `DeviceManager`.
  1. `GetPreferredAllocation` allows a device plugin to forward allocation
     preferrence to the `DeviceManager` so it can incorporate this information
     into its allocation decisions. The `DeviceManager` will call out to a
     plugin at pod admission time asking for a preferred device allocation
     of a given size from a list of available devices to make a more informed
     decision.
  1. `PreStartContainer` is called before each container start if indicated by
     device plugins during registeration phase. It allows Device Plugins to run device
     specific operations on the Devices requested.


### User Stories

#### Vendor story

Kubernetes provides to vendors a mechanism called device plugins to:
  * advertise devices.
  * monitor devices (currently perform health checks).
  * hook into the runtime to execute device specific instructions
    (e.g: Clean GPU memory) and 
    to take in order to make the device available in the container.

```go
service DevicePlugin {
  rpc GetDevicePluginOptions(Empty) returns (DevicePluginOptions) {}
	rpc ListAndWatch(Empty) returns (stream ListAndWatchResponse) {}
	rpc Allocate(AllocateRequest) returns (AllocateResponse) {}
  rpc GetPreferredAllocation(PreferredAllocationRequest) returns (PreferredAllocationResponse) {}
	rpc PreStartContainer(PreStartContainerRequest) returns (PreStartContainerResponse) {}
}
```

The gRPC server that the device plugin must implement is expected to
be advertised on a unix socket in a mounted hostPath (e.g:
`/var/lib/kubelet/device-plugins/nvidiaGPU.sock`).

Finally, to notify Kubelet of the existence of the device plugin,
the vendor's device plugin will have to make a request to Kubelet's
own gRPC server.
Only then will kubelet start interacting with the vendor's device plugin
through the gRPC apis.

#### End User story

When setting up the cluster the admin knows what kind of devices are present
on the different machines and therefore can select what devices to enable.

The cluster admin knows his cluster has NVIDIA GPUs therefore he deploys
the NVIDIA device plugin through:
`kubectl create -f nvidia.io/device-plugin.yml`

The device plugin lands on all the nodes of the cluster and if it detects that
there are no GPUs it terminates (assuming `restart: OnFailure`). However, when
there are GPUs it reports them to Kubelet and starts its gRPC server to
monitor devices and hook into the container creation process.

Devices reported by Device Plugins are advertised as Extended resources of
the shape `vendor-domain/vendor-device`.
E.g., Nvidia GPUs are advertised as `nvidia.com/gpu`

Devices can be selected using the same process as for OIRs in the pod spec.
Devices have no impact on QOS. However, for the alpha, we expect the request
to have limits == requests.

1. A user submits a pod spec requesting X GPUs (or devices) through
   `vendor-domain/vendor-device`
2. The scheduler filters the nodes which do not match the resource requests
3. The pod lands on the node and Kubelet decides which device
   should be assigned to the pod
4. Kubelet calls `Allocate` on the matching Device Plugins
5. The user deletes the pod or the pod terminates

When receiving a pod which requests Devices kubelet is in charge of:
  * deciding which device to assign to the pod's containers 
  * Calling the `Allocate` function with the list of devices

The scheduler is still in charge of filtering the nodes which cannot
satisfy the resource requests.

### Notes/Constraints/Caveats (Optional)

N/A

### Risks and Mitigations

 In case of upgrades, bugs in the `DeviceManager` or device plugin can
 prevent new pods from starting and/or already running pods from restarting.
 This can be mitigated by comprehensive testing both in `DeviceManager`
 and device plugin where the latter being the responsiblility of the device
 plugin vendor.

## Design Details

### Introduction

The device plugin is structured in 3 parts:
1. Registration: The device plugin advertises its presence to Kubelet
2. ListAndWatch: The device plugin advertises a list of Devices to Kubelet
   and sends it again if the state of a Device changes
3. Allocate: When creating containers, Kubelet calls the device plugin's
   `Allocate` function so that it can run device specific instructions (gpu
    cleanup, QRNG initialization, ...) and instruct Kubelet how to make the
    device available in the container.

### Registration

When starting the device plugin is expected to make a (client) gRPC call
to the `Register` function that Kubelet exposes.

The communication between Kubelet is expected to happen only through Unix
sockets and follow this simple pattern:
1. The device plugins sends a `RegisterRequest` to Kubelet (through a
   gRPC request)
2. Kubelet answers to the `RegisterRequest` with a `RegisterResponse`
   containing any error Kubelet might have encountered
3. The device plugin start its gRPC server if it did not receive an
   error

### Unix Socket

Device Plugins are expected to communicate with Kubelet through gRPC
on an Unix socket.
When starting the gRPC server, they are expected to create a unix socket
at the following host path: `/var/lib/kubelet/device-plugins/`.

For non bare metal device plugin this means they will have to mount the folder
as a volume in their pod spec ([see Installation](#installation)).

Device plugins can expect to find the socket to register themselves on
the host at the following path:
`/var/lib/kubelet/device-plugins/kubelet.sock`.

### Protocol Overview

When first registering themselves against Kubelet, the device plugin
will send:
  * The name of their unix socket
  * [The API version against which they were built](#versioning).
  * Their `ResourceName` they want to advertise

Kubelet answers with whether or not there was an error.
The errors may include (but not limited to):
  * API version not supported
  * A device plugin already registered this `ResourceName`

After successful registration, Kubelet will interact with the plugin through
the following functions:
  * ListAndWatch: The device plugin advertises a list of Devices to Kubelet
    and sends it again if the state of a Device changes
  * `Allocate`: Called when creating a container with a list of devices

![Process](device-plugin.png)


### API Specification

```go
// Registration is the service advertised by the Kubelet
// Only when Kubelet answers with a success code to a Register Request
// may Device Plugins start their service
// Registration may fail when device plugin version is not supported by
// Kubelet or the registered resourceName is already taken by another
// active device plugin. Device plugin is expected to terminate upon registration failure
service Registration {
	rpc Register(RegisterRequest) returns (Empty) {}
}

message DevicePluginOptions {
	// Indicates if PreStartContainer call is required before each container start
	bool pre_start_required = 1;
	// Indicates if GetPreferredAllocation is implemented and available for calling
	bool get_preferred_allocation_available = 2;
}

message RegisterRequest {
	// Version of the API the Device Plugin was built against
	string version = 1;
	// Name of the unix socket the device plugin is listening on
	// PATH = path.Join(DevicePluginPath, endpoint)
	string endpoint = 2;
	// Schedulable resource name. As of now it's expected to be a DNS Label
	string resource_name = 3;
	// Options to be communicated with Device Manager
	DevicePluginOptions options = 4;
}

message Empty {
}

// DevicePlugin is the service advertised by Device Plugins
service DevicePlugin {
	// GetDevicePluginOptions returns options to be communicated with Device
	// Manager
	rpc GetDevicePluginOptions(Empty) returns (DevicePluginOptions) {}

	// ListAndWatch returns a stream of List of Devices
	// Whenever a Device state change or a Device disappears, ListAndWatch
	// returns the new list
	rpc ListAndWatch(Empty) returns (stream ListAndWatchResponse) {}

	// GetPreferredAllocation returns a preferred set of devices to allocate
	// from a list of available ones. The resulting preferred allocation is not
	// guaranteed to be the allocation ultimately performed by the
	// devicemanager. It is only designed to help the devicemanager make a more
	// informed allocation decision when possible.
	rpc GetPreferredAllocation(PreferredAllocationRequest) returns (PreferredAllocationResponse) {}

	// Allocate is called during container creation so that the Device
	// Plugin can run device specific operations and instruct Kubelet
	// of the steps to make the Device available in the container
	rpc Allocate(AllocateRequest) returns (AllocateResponse) {}

	// PreStartContainer is called, if indicated by Device Plugin during registeration phase,
	// before each container start. Device plugin can run device specific operations
	// such as resetting the device before making devices available to the container
	rpc PreStartContainer(PreStartContainerRequest) returns (PreStartContainerResponse) {}
}

// ListAndWatch returns a stream of List of Devices
// Whenever a Device state change or a Device disappears, ListAndWatch
// returns the new list
message ListAndWatchResponse {
	repeated Device devices = 1;
}

message TopologyInfo {
	repeated NUMANode nodes = 1;
}

message NUMANode {
	int64 ID = 1;
}

/* E.g:
* struct Device {
*    ID: "GPU-fef8089b-4820-abfc-e83e-94318197576e",
*    Health: "Healthy",
*    Topology:
*      Node:
*        ID: 1
*} */
message Device {
	// A unique ID assigned by the device plugin used
	// to identify devices during the communication
	// Max length of this field is 63 characters
	string ID = 1;
	// Health of the device, can be healthy or unhealthy, see constants.go
	string health = 2;
	// Topology for device
	TopologyInfo topology = 3;
}

// - PreStartContainer is expected to be called before each container start if indicated by plugin during registration phase.
// - PreStartContainer allows kubelet to pass reinitialized devices to containers.
// - PreStartContainer allows Device Plugin to run device specific operations on
//   the Devices requested
message PreStartContainerRequest {
	repeated string devices_ids = 1 [(gogoproto.customname) = "DevicesIDs"];
}

// PreStartContainerResponse will be send by plugin in response to PreStartContainerRequest
message PreStartContainerResponse {
}

// PreferredAllocationRequest is passed via a call to GetPreferredAllocation()
// at pod admission time. The device plugin should take the list of
// `available_deviceIDs` and calculate a preferred allocation of size
// 'allocation_size' from them, making sure to include the set of devices
// listed in 'must_include_deviceIDs'.
message PreferredAllocationRequest {
	repeated ContainerPreferredAllocationRequest container_requests = 1;
}

message ContainerPreferredAllocationRequest {
	// List of available deviceIDs from which to choose a preferred allocation
	repeated string available_deviceIDs = 1;
	// List of deviceIDs that must be included in the preferred allocation
	repeated string must_include_deviceIDs = 2;
	// Number of devices to include in the preferred allocation
	int32 allocation_size = 3;
}

// PreferredAllocationResponse returns a preferred allocation,
// resulting from a PreferredAllocationRequest.
message PreferredAllocationResponse {
	repeated ContainerPreferredAllocationResponse container_responses = 1;
}

message ContainerPreferredAllocationResponse {
	repeated string deviceIDs = 1;
}

// - Allocate is expected to be called during pod creation since allocation
//   failures for any container would result in pod startup failure.
// - Allocate allows kubelet to exposes additional artifacts in a pod's
//   environment as directed by the plugin.
// - Allocate allows Device Plugin to run device specific operations on
//   the Devices requested
message AllocateRequest {
	repeated ContainerAllocateRequest container_requests = 1;
}

message ContainerAllocateRequest {
	repeated string devices_ids = 1 [(gogoproto.customname) = "DevicesIDs"];
}

// AllocateResponse includes the artifacts that needs to be injected into
// a container for accessing 'deviceIDs' that were mentioned as part of
// 'AllocateRequest'.
// Failure Handling:
// if Kubelet sends an allocation request for dev1 and dev2.
// Allocation on dev1 succeeds but allocation on dev2 fails.
// The Device plugin should send a ListAndWatch update and fail the
// Allocation request
message AllocateResponse {
	repeated ContainerAllocateResponse container_responses = 1;
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
}

// Mount specifies a host volume to mount into a container.
// where device library or tools are installed on host and container
message Mount {
	// Path of the mount within the container.
	string container_path = 1;
	// Path of the mount on the host.
	string host_path = 2;
	// If set, the mount is read-only.
	bool read_only = 3;
}

// DeviceSpec specifies a host device to mount into a container.
message DeviceSpec {
	// Path of the device within the container.
	string container_path = 1;
	// Path of the device on the host.
	string host_path = 2;
	// Cgroups permissions of the device, candidates are one or more of
	// * r - allows container to read from the specified device.
	// * w - allows container to write to the specified device.
	// * m - allows container to create device files that do not yet exist.
	string permissions = 3;
}
```

### HealthCheck and Failure Recovery

We want Kubelet as well as the Device Plugins to recover from failures
that may happen on any side of this protocol.

At the communication level, gRPC is a very strong piece of software and
is able to ensure that if failure happens it will try its best to recover
through exponential backoff reconnection and Keep Alive checks.

The proposed mechanism intends to replace any device specific handling in
Kubelet. Therefore in general, device plugin failure or upgrade means that
Kubelet is not able to accept any pod requesting a Device until the upgrade
or failure finishes.

If a device fails, the Device Plugin should signal that through the
`ListAndWatch` gRPC stream. We then expect Kubelet to fail the Pod.

If any Device Plugin fails the behavior we expect depends on the task Kubelet
is performing:
* In general we expect Kubelet to remove any devices that are owned by the failed
  device plugin from the node capacity. We also expect node allocatable to be
  equal to node capacity.
* We however do not expect Kubelet to fail or restart any pods or containers
  running that are using these devices.
* If Kubelet is in the process of allocating a device, then it should fail
  the container process.

If the Kubelet fails or restarts, we expect the Device Plugins to know about
it through gRPC's Keep alive feature and try to reconnect to Kubelet.

When Kubelet fails or restarts it should know what are the devices that are
owned by the different containers and be able to rebuild a list of available
devices.
We are expecting to implement this through a checkpointing mechanism that Kubelet
would write and read from.


### API Changes

When discovering the devices, Kubelet will be in charge of advertising those
resources to the API server as part of the kubelet node update current protocol.

We will be using extended resources to schedule, trigger and advertise these
Devices.
When a Device plugin registers two `foo-device` the node status will be
updated to advertise 2 `vendor-domain/foo-device`.

If a user wants to trigger the device plugin he only needs to request this
through the same mechanism as OIRs in his Pod Spec.


### Installation

The installation process should be straightforward to the user, transparent
and similar to other regular Kubernetes actions.
The device plugin should also run in containers so that Kubernetes can
deploy them and restart the plugins when they fail.
However, we should not prevent the user from deploying a bare metal device
plugin.

Deploying the device plugins through DemonSets makes sense as the cluster
admin would be able to specify which machines it wants the device plugins to
run on, the process is similar to any Kubernetes action and does not require
to change any parts of Kubernetes.

Additionally, for integrated solutions such as `kubeadm` we can add support
to auto-deploy community vetted Device Plugins.
Thus not fragmenting once more the Kubernetes ecosystem.

For users installing Kubernetes without using an integrated solution such
as `kubeadm` they would use the examples that we would provide at:
`https://github.com/vendor/device-plugin/tree/master/device-plugin.yaml`

YAML example:

```yaml
apiVersion: extensions/v1beta1
kind: DaemonSet
metadata:
spec:
    template:
        metadata:
            labels:
                - name: device-plugin
        spec:
            containers:
                name: device-plugin-ctr
                image: NVIDIA/device-plugin:1.0
                volumeMounts:
                  - mountPath: /device-plugin
                  - name: device-plugin
           volumes:
             - name: device-plugin
               hostPath:
                   path: /var/lib/kubelet/device-plugins
```


### Test Plan

[X] I/we understand the owners of the involved components may require updates to
existing tests to make this code solid enough prior to committing the changes necessary
to implement this enhancement.

##### Prerequisite testing updates

##### Unit tests

- `k8s.io/kubernetes/pkg/kubelet/cm/devicemanager`: `<20221002>` - `83.5%`

##### Integration tests

Not Applicable.

##### e2e tests

Device Manager and Device plugin node e2e tests:
* https://github.com/kubernetes/kubernetes/blob/master/test/e2e_node/device_manager_test.go
* https://github.com/kubernetes/kubernetes/blob/master/test/e2e_node/device_plugin_test.go

### Graduation Criteria

#### Alpha

- Feature implemented behind a feature flag
- Initial e2e tests completed and enabled

#### Beta

- Gather feedback from developers and surveys
- Additional tests are in Testgrid and linked in KEP

#### GA

- N examples of real-world usage
- N installs
- More rigorous forms of testing—e.g., downgrade tests and scalability tests
- Allowing time for feedback

**Note:** Generally we also wait at least two releases between beta and
GA/stable, because there's no opportunity for user feedback, or even bug reports,
in back-to-back releases.

#### Deprecation

- Announce deprecation and support policy of the existing flag
- Two versions passed since introducing the functionality that deprecates the flag (to address version skew)
- Address feedback on usage/changed behavior, provided on GitHub issues
- Deprecate the flag

### Upgrade / Downgrade Strategy

#### Upgrading your cluster

*TLDR:* 
Given that we cannot guarantee that the Device Plugins are not running
a daemon providing a critical service to Devices and when stopped will
crash the running containers, it is up to the vendor to specify the
upgrading scheme of their device plugin.

However, If you are upgrading either Kubelet or any device plugin the safest way
is to drain the node of all pods and upgrade.

Depending on what you are upgrading and what changes happened then it
is completely possible to only restart just Kubelet or just the device plugin.

##### Upgrading Kubelet

This assumes that the Device Plugins running on the nodes fully implement the
protocol and are able to recover from a Kubelet crash.

Then, as long as the Device Plugin API does not change upgrading Kubelet can be done
seamlessly through a Kubelet restart.

Upgrading Kubelet can be done seamlessly through a Kubelet restart and does not
require deployment of a different version of the device plugin as the API has been
mostly stable since its graduation to Beta in v1.10.

Note that graduation of Device Plugin API to GA is out of scope of this KEP and
will be covered in a separate KEP in the future.

Refer to the versioning section for versioning scheme compatibility.

##### Upgrading Device Plugins

Because we cannot enforce what the different Device Plugins will do, we cannot
say for certain that upgrading a device plugin will not crash any containers
on the node.

It is therefore up to the Device Plugin vendors to specify if the Device Plugins
can be upgraded without impacting any running containers.

As mentioned earlier, the safest way is to drain the node before upgrading
the Device Plugins.

### Version Skew Strategy

Prior to v1.10, the versioning scheme required the Device Plugin's API version to
match exactly the Kubelet's version. With the graduation of this feature to Beta
and device plugin API has been stable and this is no longer a hard requirement.

## Production Readiness Review Questionnaire

### Feature Enablement and Rollback

###### How can this feature be enabled / disabled in a live cluster?

- [X] Feature gate (also fill in values in `kep.yaml`)
  - Feature gate name: `DevicePlugins`
  - Components depending on the feature gate: Device Manager

###### Does enabling the feature change any default behavior?

No, in order to tap into this feature and for a container to be allocated devices as resources:
* feature gate has to be enabled
* device plugin must be deployed
* the container needs to explicitly request resources provided by plugins

###### Can the feature be disabled once it has been enabled (i.e. can we roll back the enablement)?

Yes, using `DevicePlugins` feature gate this feature can be disabled.
Note that disabling the feature gate requires kubelet restart for the changes
to take effect. In case no device plugin daemonset pods or pods consuming
devices are running on the node, disabling feature gate shouldn't cause any issue.

If the feature gate is being disabled on a node where such pods are running,
it is the responsibliity of the cluster admin to ensure that the node is
appropriately drained.


###### What happens if we reenable the feature if it was previously rolled back?

No impact on running pods in the cluster.
When enabled, this feature provides the ability to expose and consume custom
devices in a Kubernetes cluster. These device plugins are typically deployed
as daemonset which can be subsequently requested by the users. Refer to
[Installation](#installation) section for more details.

###### Are there any tests for feature enablement/disablement?

Yes, covered by node e2e tests:
* https://github.com/kubernetes/kubernetes/blob/master/test/e2e_node/device_manager_test.go
* https://github.com/kubernetes/kubernetes/blob/master/test/e2e_node/device_plugin_test.go

### Rollout, Upgrade and Rollback Planning

###### How can a rollout or rollback fail? Can it impact already running workloads?

* A rollout can fail in case a bug is introduced in device manager preventing already
  running pods from restarting or new pods to start.

###### What specific metrics should inform a rollback?

`device_plugin_registration_total` and `device_plugin_alloc_duration_seconds` metrics can
determine the heath of the feature and can be be used to detrmine if a rollback needs to
be performed.

`device_plugin_registration_total` is an indicator of number of device plugin
registrations. This metric can be observed both before and after an upgrade. If the
number of device plugin registrations remain the same, it can be assumed that
the upgrade process was successful. If however, the number of registrations are not
same, upgrade process hasn't been successful and a rollback is required.

`device_plugin_alloc_duration_seconds` can be used as an indicator to determine
whether or not device plugin allocations are taking place successfully. If devices
were being allocated before upgrade and are not after an update a rollback is
required.

###### Were upgrade and rollback tested? Was the upgrade->downgrade->upgrade path tested?

Yes

When upgrading device plugins, it is recommended that the cluster admin drain workloads
requesting devices and conform to the device plugin API.

This has been validated by several device plugins. Here is an example: https://github.com/NVIDIA/k8s-device-plugin#upgrading-kubernetes-with-the-device-plugin. 

###### Is the rollout accompanied by any deprecations and/or removals of features, APIs, fields of API types, flags, etc.?

No

### Monitoring Requirements

###### How can an operator determine if the feature is in use by workloads?

1. The operator can look at `device_plugin_registration_total` and `device_plugin_alloc_duration_seconds`
   metrics to gather information about successful device plugin registrations and device
   allocations to pods respectively.
1. Kubelet PodResource API endpoints (`List` and `GetAllocatableResources`) can be used
   to obtain information on devices allocated to running pods. Refer to
   [keps/sig-node/606-compute-device-assignment]([keps/sig-node/606-compute-device-assignment]).
1. Status of the nodes  specifically device `capacity` and `allocatable` can be inspected
   to see if device plugins were successfully deployed and are available to be allocated
   on the node.
   In addition to that, status of the running pods can be inspected to determine if the
   pod requesting device(s) was running successfully and allocated device(s) as per its
   request.
1. Checking that the pod spec contains request for resources provided by plugins which
   can be identified as non-native resources (i.e resources other than cpu, memory,
   hugepages).


###### How can someone using this feature know that it is working for their instance?

- [X] API .status
  - Field: Node.status
    - Property: [capacity,allocatable]

###### What are the reasonable SLOs (Service Level Objectives) for the enhancement?

The metric `device_plugin_alloc_duration_seconds` can be used to determine if device allocation
is taking longer than expected.

###### What are the SLIs (Service Level Indicators) an operator can use to determine the health of the service?

- [X] Metrics
  - Metric name:
    - device_plugin_registration_total
    - device_plugin_alloc_duration_seconds

###### Are there any missing metrics that would be useful to have to improve observability of this feature?

No.

### Dependencies

###### Does this feature depend on any specific services running in the cluster?

No.

### Scalability

###### Will enabling / using this feature result in any new API calls?

Yes, node status is updated to reflect the device capacity and allocatable.
At a node level, device plugins and device Manager communicate over gRPC.

###### Will enabling / using this feature result in introducing new API types?

No, devices are exposed as extended resources.

###### Will enabling / using this feature result in any new calls to the cloud provider?

No.

###### Will enabling / using this feature result in increasing size or count of the existing API objects?

No.

###### Will enabling / using this feature result in increasing time taken by any operations covered by existing SLIs/SLOs?

No.

###### Will enabling / using this feature result in non-negligible increase of resource usage (CPU, RAM, disk, IO, ...) in any components?

No reported or known increase in resource usage.

### Troubleshooting

###### How does this feature react if the API server and/or etcd is unavailable?

If Kubernetes control plane is down, no new Pods including device plugin
daemonset pods can be deployed.


###### What are other known failure modes?

No known failure modes.

###### What steps should be taken if SLOs are not being met to determine the problem?

## Implementation History
- **2020-10-02:** KEP ported to the most recent template and GA graduation.
- **2020-10-04:** Updates based on review comments, added device plugin metrics.


## References

  * [Extension to support new compute resources](https://github.com/kubernetes/enhancements/issues/368)
  * [Device Plugin Design Proposal](https://github.com/kubernetes/community/pull/695)
  * [Adding a proposal for hardware accelerators](https://github.com/kubernetes/community/pull/844)
  * [Enable "kick the tires" support for NVIDIA GPUs in COS](https://github.com/kubernetes/kubernetes/pull/45136)
  * [Extend experimental support to multiple NVIDIA GPUs](https://github.com/kubernetes/kubernetes/pull/42116)
  * [Kubernetes Meeting notes](https://docs.google.com/document/d/1Qg42Nmv-QwL4RxicsU2qtZgFKOzANf8fGayw8p3lX6U/edit#)
  * [Better Abstraction for Compute Resources in Kubernetes](https://docs.google.com/document/d/1666PPUs4Lz56TqKygcy6mXkNazde-vwA7q4e5H92sUc)
  * [Extensible support for hardware devices in Kubernetes (join Kubernetes-dev@googlegroups.com for access)](https://docs.google.com/document/d/1LHeTPx_fWA1PdZkHuALPzYxR0AYXUiiXdo3S0g2VSlo/edit)


## Drawbacks

Not Applicable.

## Alternatives

In Kubernetes v1.25, [Dynamic Resource Allocation](https://github.com/kubernetes/enhancements/tree/master/keps/sig-node/3063-dynamic-resource-allocation) enhancement proposal was approved for development as an alpha feature. This feature was primarility designed for **dynamic** allocation of resources and is expected to co-exist with the device plugin API. 


## Infrastructure Needed (Optional)

Not Applicable.
