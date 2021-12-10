# KEP-20210712: Addendum: Add Dellocate and PostStopContainer to [Device Manager API](#reference)

## Table of Contents

<!-- toc -->
- [Release Signoff Checklist](#release-signoff-checklist)
- [Summary](#summary)
- [Motivation](#motivation)
  - [Goals](#goals)
  - [Non-Goals](#non-goals)
- [Use Cases](#use-cases)
- [Proposal](#proposal)
  - [Risks and Mitigations](#risks-and-mitigations)
- [Design Details](#design-details)
  - [Test Plan](#test-plan)
  - [Graduation Criteria](#graduation-criteria)
    - [Alpha -&gt; Beta Graduation](#alpha---beta-graduation)
    - [Beta -&gt; GA Graduation](#beta---ga-graduation)
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
- [Reference: Device Manager Proposal](#reference-device-manager-proposal)
  - [Motivation](#motivation-1)
  - [Use Cases](#use-cases-1)
  - [Objectives](#objectives)
  - [Non Objectives](#non-objectives)
  - [TLDR](#tldr)
  - [Vendor story](#vendor-story)
  - [End User story](#end-user-story)
  - [Device Plugin](#device-plugin)
    - [Introduction](#introduction)
    - [Registration](#registration)
    - [Unix Socket](#unix-socket)
    - [Protocol Overview](#protocol-overview)
    - [API Specification](#api-specification)
  - [HealthCheck and Failure Recovery](#healthcheck-and-failure-recovery)
    - [API Changes](#api-changes)
  - [Upgrading your cluster](#upgrading-your-cluster)
    - [Upgrading Kubelet](#upgrading-kubelet)
    - [Upgrading Device Plugins](#upgrading-device-plugins)
  - [Installation](#installation)
  - [Versioning](#versioning)
  - [References](#references)
<!-- /toc -->

## Release Signoff Checklist

Items marked with (R) are required *before targeting a milestone / release*.

- [ ] (R) Enhancement issue in release milestone, which links to KEP dir in
[kubernetes/enhancements] (not the initial KEP PR)
- [ ] (R) KEP approvers have approved the KEP status as `implementable`
- [ ] (R) Design details are appropriately documented
- [ ] (R) Test plan is in place, giving consideration to SIG Architecture and
SIG Testing input
- [ ] (R) Graduation criteria is in place
- [ ] (R) Production readiness review completed
- [ ] Production readiness review approved
- [ ] "Implementation History" section is up-to-date for milestone
- [ ] User-facing documentation has been created in [kubernetes/website], for
publication to [kubernetes.io]
- [ ] Supporting documentation—e.g., additional design documents, links to
mailing list discussions/SIG meetings, relevant PRs/issues, release notes

[kubernetes.io]: https://kubernetes.io/
[kubernetes/enhancements]: https://git.k8s.io/enhancements
[kubernetes/kubernetes]: https://git.k8s.io/kubernetes
[kubernetes/website]: https://git.k8s.io/website

## Summary

This KEP proposes adding two additional API calls:

- `Deallocate`: (Optional). This API call is the opposite of `Allocate` and is
needed to inform device plugins that some devices are no longer being used.
- `PostStopContainer`: (Optional). This API call allows the device plugins to do
 device cleanup, driver unloading, and any other actions that may be needed.

Since both additions are optional, existing device plugins should continue
functioning correctly with no needed modifications. Only device plugins that
wish to utilize the new API calls will need to be modified.

## Motivation

The following are some use cases and motivations for the proposed change:

- `PostStopContainer`:

  - For use with some devices like FPGAs. Devices like these will need to be
  cleaned up (i.e. de-programmed) after each use. Otherwise, they run the
  possibility of 2 risks:
    - If whatever is programmed on the FPGA is not cleaned up, it will keep
    running and consuming power for no reason, on a large scale (datacenter
    scale) this is unacceptable.
    - If whatever is programmed on the FPGA has network access, it runs the risk
    of continuing to send and respond to packets and pollute the network.
  - For dynamically binding/unbinding drivers for the devices as needed.

- `Deallocate`:

  - For use with complex device plugins that require tracking the state of their
  devices and learning when they are no longer in use. Multi modal devices can
  operate in more than one mode of operation, and thus have to be advertised by
  the device plugin as two separate devices, and the device plugin has to take
  care to stop advertising a device when its being used in the other mode. An
  example of multi modal devices is also using FPGAs in the following 2 modes:
    - Use the entire FPGA as a device
    - Split the FPGA between multiple users, essentially advertising one FPGA as
    multiple smaller FPGAs.
    - A device plugin needs to know when a full FPGA will stop being used so it
    can go back to advertise the FPGA partitions, and vice versa.
  - To maintain the same logical splitting of `Allocate` and `PreStartContainer`

### Goals

- Add the `PostStopContainer` and `Deallocate` API calls to the device plugin
API.
- Make the new added API calls optional, as they are not needed for all devices.
- Maintain compatibility with existing device plugins.

### Non-Goals

- Make any modifications to the main API calls of the device plugin API.
- Make changes specific to one type of devices.

## Use Cases

- Power management: in case of accelerators that are not always depending on
controlling process from userland side (e.g. FPGA which has external
network/GPIO interfaces) it is possible that after stopping of Pod where device
was initially allocated, accelerator part of the code will be still processing
data on external interfaces and consume significant amount of power. As we don't
know when it will be next time allocated, it will be better to reset accelerator
device and put it into power saving mode immediately after workload that
requested it is terminated.

- Security: HW-accelerators often have internal memory and processing data not
always cleared after processing. Doing reset/memory clean-up on next Allocate
call means that between two points of time (termination of workload and next
Allocate) some confidential information can be still present on the device (e.g.
IPsec encryption keys, for crypto accelerators). It is better to have ability to
cleanup device memory as soon as workload that requested it finishes.

- Virtualization: HW-accelerators can be used and resued in several ways in a
VM.
  - A device bound to VFIO and passthrough'ed into the VM
  - A device can be configured as an mediated device and used in a VM
  - A device can be configured with SR-IOV and specific VFs or a PF can be used in
a VM
A device may support all of those modi or a subset. Depending on what is needed
for the workload a device may be bound to VFIO for performance reasons e.g. and
released/deallocated for usage in a different workload as an mediated device.

- Bind/Unbind: Instead of pre-configuring the devices on the node with the
correct drivers etc, interested in letting the device plugin handle the
device configuration (driver bind/unbind etc). Basically, trying to
dynamically switch the drivers bound to the device during Allocate and unbind
the driver once the device is no longer in use. Currently no way of
learning this state and Deallocate interface would be really useful here.

- Shared Devices: The device plugin assumes the allocated devices are local to
the node but that doesn't necessarily have to be the case. If the discovered
devices are shared across nodes, then a `Deallocate(..)` is necessary to
guarantee the resources are quickly freed for others to use
(This explains [Akri](https://github.com/project-akri/akri) usecase)

## Proposal

The device plugin API includes API calls for:

- `Allocate`: Which is used to instruct device plugins to allocate device(s) to
requesting containers.
- `PreStartContainer`: (Optional). Which allows the device plugins to do device
initialization, loading drivers, and any other initialization actions that may
be needed.

This KEP proposes adding two extra API calls, maintaining the same logical
reasoning of the previous two. Those are:

- `Deallocate`: (Optional). Which is the opposite of allocate, and is needed to
inform device plugins that some devices are no longer being used (this used to
happen silently before)
- `PostStopContainer`: (Optional). Which allows the device plugins to do device
cleanup, driver unloading, and any other cleanup actions that may be needed.

Since both additions are optional, existing device plugins should continue
functioning properly with no needed modifications. Only device plugins that wish
to utilize the new API calls will need to be modified.

### Risks and Mitigations

Only risk is breaking existing device plugins by introducing non optional
changes. Can be mitigated by enough test coverage.

## Design Details

- Move `PreStartContainer` in `DeviceManager` to be used as a container
lifecycle hook. (This change isn't truly required, but useful for compatibility
and organization with the next steps).
- Add `PostStopContainer` and `Deallocate` calls in the DevicePlugin API.
- Add `PostStopContainer` as a container lifecycle hook.
- Add `Deallocate` calls in container manager, taking care to only do so for
devices that are no longer in the reuse list.
- Add and modify test cases for both calls.
- Test with existing device plugins to ensure the changes are non-breaking.
- Test with new device plugins utilizing such changes to ensure the changes are
working.

### Test Plan

- Unit tests will be updated to include the new API calls
- E2E tests should be added with a sample device plugin for verification.

### Graduation Criteria

#### Alpha -> Beta Graduation

- Gather feedback from developers and surveys, specially about the device reuse
and possible alternatives.
- Complete implementation for the new API calls.
- Tests (unit, e2e) are in Testgrid and linked in KEP.

#### Beta -> GA Graduation

- More rigorous testing as needed/discussed by developers.
- Larger scale use/testing by interested users with no reported major bugs.

### Upgrade / Downgrade Strategy

As part of the device plugin API, this will follow the same API versioning
system. This means that it is up to an application (a device plugin) to choose
the required API version it wants. As long as the cluster has a recent enough
(to include the required API version) kubernetes, upgrade or downgrades require
no cluster modifications at all, and are decided on the application level.

### Version Skew Strategy

As part of the device plugin API, this will follow the same API versioning
system. This means that it is up to an application (a device plugin) to choose
the required API version it wants. No version skew issues will arise.

## Production Readiness Review Questionnaire

### Feature Enablement and Rollback

#### How can this feature be enabled / disabled in a live cluster?**

- [X] Feature gate (also fill in values in `kep.yaml`)
  - Feature gate name: `DeviceManagerDeallocate`
  - Components depending on the feature gate: `kubelet`

#### Does enabling the feature change any default behavior?**

No. The feature is optional, no default behavior will change.
Feature Gate must be enabled for this new functionality.

#### Can the feature be disabled once it has been enabled (i.e. can we roll back the enablement)?

Yes, the feature can be disabled by disabling the `DeviceManagerDeallocate` feature gate.

#### What happens if we reenable the feature if it was previously rolled back?

No side effects expected.

#### Are there any tests for feature enablement/disablement?

- A specific e2e test will demonstrate that the default behaviour is preserved
when the feature gate is disabled, or when the feature is not used (2 separate
tests)
- Another test will demonstrate that the new feature is working when the feature
gate is enabled.

### Monitoring Requirements

#### How can an operator determine if the feature is in use by workloads?

Inspect the kubelet configuration of a node -- check for the presence of the
feature gate and use a device plugin that uses the new API calls.

#### What are the reasonable SLOs (Service Level Objectives) for the enhancement?

There are no specific SLOs for this feature.

#### What are the SLIs (Service Level Indicators) an operator can use to determine the health of the service?

None

#### Are there any missing metrics that would be useful to have to improve observability of this feature?

None

#### Does this feature depend on any specific services running in the cluster?

No.

### Scalability

#### Will enabling / using this feature result in any new API calls?**

The mere enablement of this feature has no effect. However, using this API by
user applications may result in extra calls to the Device Manager API. These
extra calls only happen at the end of the lifecycle of some containers (those
who have previously requested devices).

The extra API calls originate from the kubelet to the device plugin running on
the same node. Since they are only happening on the node level, there's no
risk of congestion, or a need to measure their throughput, etc.

#### Will enabling / using this feature result in introducing new API types?

No

#### Will enabling / using this feature result in any new calls to the cloud provider?

No

#### Will enabling / using this feature result in increasing size or count of the existing API objects?

No

#### Will enabling / using this feature result in increasing time taken by any operations covered by existing SLIs/SLOs?

The new API calls could delay:

- Pod deletion time, some HW, SW may take longer to do a specific deallocation

#### Will enabling / using this feature result in non-negligible increase of resource usage (CPU, RAM, disk, IO, ...) in any components?

No

## Implementation History

- 2021-07-12: Initial KEP created
- 2021-10-12: Moved to the newer template

---

## Reference: Device Manager Proposal

- [Motivation](#motivation)
- [Use Cases](#use-cases)
- [Objectives](#objectives)
- [Non Objectives](#non-objectives)
- [Vendor story](#vendor-story)
- [End User story](#end-user-story)
- [Device Plugin](#device-plugin)
   -[Introduction](#introduction)
   -[Registration](#registration)
   -[Unix Socket](#unix-socket)
   -[Protocol Overview](#protocol-overview)
   -[API specification](#api-specification)
   -[HealthCheck and Failure Recovery](#healthcheck-and-failure-recovery)
   -[API Changes](#api-changes)
- [Upgrading your cluster](#upgrading-your-cluster)
- [Installation](#installation)
- [Versioning](#versioning)
- [References](#references)

_Authors:_

- @RenaudWasTaken - Renaud Gaubert &lt;rgaubert@NVIDIA.com&gt;
- @jiayingz - Jiaying Zhang &lt;jiayingz@google.com&gt;

### Motivation

Kubernetes currently supports discovery of CPU and Memory primarily to a
minimal extent. Very few devices are handled natively by Kubelet.

It is not a sustainable solution to expect every hardware vendor to add their
vendor specific code inside Kubernetes to make their devices usable.

Instead, we want a solution for vendors to be able to advertise their resources
to Kubelet and monitor them without writing custom Kubernetes code.
We also want to provide a consistent and portable solution for users to
consume hardware devices across k8s clusters.

This document describes a vendor independent solution to:
 -Discovering and representing external devices
 -Making these devices available to the containers, using these devices,
    scrubbing and securely sharing these devices.
 -Health Check of these devices

Because devices are vendor dependent and have their own sets of problems
and mechanisms, the solution we describe is a plugin mechanism that may run
in a container deployed through the DaemonSets mechanism or in bare metal mode.

The targeted devices include GPUs, High-performance NICs, FPGAs, InfiniBand,
Storage devices, and other similar computing resources that require vendor
specific initialization and setup.

The goal is for a user to be able to enable vendor devices (e.g: GPUs) through
the following simple steps:

- `kubectl create -f http://vendor.com/device-plugin-daemonset.yaml`
- When launching `kubectl describe nodes`, the devices appear in the node
    status as `vendor-domain/vendor-device`. Note: naming
    convention is discussed in PR [#844](https://github.com/kubernetes/community/pull/844)

### Use Cases

- I want to use a particular device type (GPU, InfiniBand, FPGA, etc.)
   in my pod.
- I should be able to use that device without writing custom Kubernetes code.
- I want a consistent and portable solution to consume hardware devices
   across k8s clusters.

### Objectives

1. Add support for vendor specific Devices in kubelet:
   - Through an extension mechanism.
   - Which allows discovery and health check of devices.
   - Which allows hooking the runtime to make devices available in containers
      and cleaning them up.
2. Define a deployment mechanism for this new API.
3. Define a versioning mechanism for this new API.

### Non Objectives

1. Handling heterogeneous nodes and topology related problems
2. Collecting metrics is not part of this proposal. We will only solve
   Health Check.

### TLDR

At their core, device plugins are simple gRPC servers that may run in a
container deployed through the pod mechanism or in bare metal mode.

These servers implement the gRPC interface defined later in this design
document and once the device plugin makes itself known to kubelet, kubelet
will interact with the device through two simple functions:

  1. A `ListAndWatch` function for the kubelet to Discover the devices and
     their properties as well as notify of any status change (device
     became unhealthy).
  2. An `Allocate` function which is called before creating a user container
     consuming any exported devices

![Process](device-plugin-overview.png)

### Vendor story

Kubernetes provides to vendors a mechanism called device plugins to:

- advertise devices.
- monitor devices (currently perform health checks).
- hook into the runtime to execute device specific instructions
    (e.g: Clean GPU memory) and
    to take in order to make the device available in the container.

```go
service DevicePlugin {
  // returns a stream of []Device
  rpc ListAndWatch(Empty) returns (stream ListAndWatchResponse) {}
  rpc Allocate(AllocateRequest) returns (AllocateResponse) {}
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

### End User story

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
 -deciding which device to assign to the pod's containers
 -Calling the `Allocate` function with the list of devices

The scheduler is still in charge of filtering the nodes which cannot
satisfy the resource requests.

### Device Plugin

#### Introduction

The device plugin is structured in 3 parts:

1. Registration: The device plugin advertises its presence to Kubelet
2. ListAndWatch: The device plugin advertises a list of Devices to Kubelet
   and sends it again if the state of a Device changes
3. Allocate: When creating containers, Kubelet calls the device plugin's
   `Allocate` function so that it can run device specific instructions (gpu
    cleanup, QRNG initialization, ...) and instruct Kubelet how to make the
    device available in the container.

#### Registration

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

#### Unix Socket

Device Plugins are expected to communicate with Kubelet through gRPC
on an Unix socket.
When starting the gRPC server, they are expected to create a unix socket
at the following host path: `/var/lib/kubelet/device-plugins/`.

For non bare metal device plugin this means they will have to mount the folder
as a volume in their pod spec ([see Installation](#installation)).

Device plugins can expect to find the socket to register themselves on
the host at the following path:
`/var/lib/kubelet/device-plugins/kubelet.sock`.

#### Protocol Overview

When first registering themselves against Kubelet, the device plugin
will send:

- The name of their unix socket
- [The API version against which they were built](#versioning).
- Their `ResourceName` they want to advertise

Kubelet answers with whether or not there was an error.
The errors may include (but not limited to):

- API version not supported
- A device plugin already registered this `ResourceName`

After successful registration, Kubelet will interact with the plugin through
the following functions:

- ListAndWatch: The device plugin advertises a list of Devices to Kubelet
    and sends it again if the state of a Device changes
- `Allocate`: Called when creating a container with a list of devices

![Process](device-plugin.png)

#### API Specification

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

// DevicePlugin is the service advertised by Device Plugins
service DevicePlugin {
  // ListAndWatch returns a stream of List of Devices
  // Whenever a Device state change or a Device disappears, ListAndWatch
  // returns the new list
  rpc ListAndWatch(Empty) returns (stream ListAndWatchResponse) {}

  // Allocate is called during container creation so that the Device
  // Plugin can run device specific operations and instruct Kubelet
  // of the steps to make the Device available in the container
  rpc Allocate(AllocateRequest) returns (AllocateResponse) {}
}

message RegisterRequest {
  // Version of the API the Device Plugin was built against
  string version = 1;
  // Name of the unix socket the device plugin is listening on
  // PATH = path.Join(DevicePluginPath, endpoint)
  string endpoint = 2;
  // Schedulable resource name
  string resource_name = 3;
}

// - Allocate is expected to be called during pod creation since allocation
//   failures for any container would result in pod startup failure.
// - Allocate allows kubelet to exposes additional artifacts in a pod's
//   environment as directed by the plugin.
// - Allocate allows Device Plugin to run device specific operations on
//   the Devices requested
message AllocateRequest {
  repeated string devicesIDs = 1;
}

// Failure Handling:
// if Kubelet sends an allocation request for dev1 and dev2.
// Allocation on dev1 succeeds but allocation on dev2 fails.
// The Device plugin should send a ListAndWatch update and fail the
// Allocation request
message AllocateResponse {
  repeated DeviceRuntimeSpec spec = 1;
}

// ListAndWatch returns a stream of List of Devices
// Whenever a Device state change or a Device disappears, ListAndWatch
// returns the new list
message ListAndWatchResponse {
  repeated Device devices = 1;
}

// The list to be added to the CRI spec
message DeviceRuntimeSpec {
  string ID = 1;

  // List of environment variable to set in the container.
  map<string, string> envs = 2;
  // Mounts for the container.
  repeated Mount mounts = 3;
  // Devices for the container
  repeated DeviceSpec devices = 4;
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

// Mount specifies a host volume to mount into a container.
// where device library or tools are installed on host and container
message Mount {
  // Path of the mount on the host.
  string host_path = 1;
  // Path of the mount within the container.
  string mount_path = 2;
  // If set, the mount is read-only.
  bool read_only = 3;
}

// E.g:
// struct Device {
//    ID: "GPU-fef8089b-4820-abfc-e83e-94318197576e",
//    State: "Healthy",
//}
message Device {
  string ID = 2;
  string health = 3;
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

- In general we expect Kubelet to remove any devices that are owned by the failed
  device plugin from the node capacity. We also expect node allocatable to be
  equal to node capacity.
- We however do not expect Kubelet to fail or restart any pods or containers
  running that are using these devices.
- If Kubelet is in the process of allocating a device, then it should fail
  the container process.

If the Kubelet fails or restarts, we expect the Device Plugins to know about
it through gRPC's Keep alive feature and try to reconnect to Kubelet.

When Kubelet fails or restarts it should know what are the devices that are
owned by the different containers and be able to rebuild a list of available
devices.
We are expecting to implement this through a checkpointing mechanism that Kubelet
would write and read from.

#### API Changes

When discovering the devices, Kubelet will be in charge of advertising those
resources to the API server as part of the kubelet node update current protocol.

We will be using extended resources to schedule, trigger and advertise these
Devices.
When a Device plugin registers two `foo-device` the node status will be
updated to advertise 2 `vendor-domain/foo-device`.

If a user wants to trigger the device plugin he only needs to request this
through the same mechanism as OIRs in his Pod Spec.

### Upgrading your cluster

*TLDR:*
Given that we cannot guarantee that the Device Plugins are not running
a daemon providing a critical service to Devices and when stopped will
crash the running containers, it is up to the vendor to specify the
upgrading scheme of their device plugin.

However, If you are upgrading either Kubelet or any device plugin the safest way
is to drain the node of all pods and upgrade.

Depending on what you are upgrading and what changes happened then it
is completely possible to only restart just Kubelet or just the device plugin.

#### Upgrading Kubelet

This assumes that the Device Plugins running on the nodes fully implement the
protocol and are able to recover from a Kubelet crash.

Then, as long as the Device Plugin API does not change upgrading Kubelet can be done
seamlessly through a Kubelet restart.

*Currently:*
As mentioned in the Versioning section, we currently expect the Device Plugin's
API version to match exactly the Kubelet's Device Plugin API version.
Therefore if the Device Plugin API version change then you will have to change
the Device Plugin too.

*Future:*
When the Device Plugin API becomes a stable feature, versioning should be
backward compatible and even if Kubelet has a different Device Plugin API,

it should not require a Device Plugin upgrade.

Refer to the versioning section for versioning scheme compatibility.

#### Upgrading Device Plugins

Because we cannot enforce what the different Device Plugins will do, we cannot
say for certain that upgrading a device plugin will not crash any containers
on the node.

It is therefore up to the Device Plugin vendors to specify if the Device Plugins
can be upgraded without impacting any running containers.

As mentioned earlier, the safest way is to drain the node before upgrading
the Device Plugins.

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

### Versioning

Currently we require exact version match between Kubelet and Device Plugin.
API version is expected to be increased only upon incompatible API changes.

Follow protobuf guidelines on versioning:

- Do not change ordering
- Do not remove fields or change types
- Add optional fields
- Introducing new fields with proper default values
- Freeze the package name to `apis/device-plugin/v1alpha1`
- Have kubelet and the Device Plugin negotiate versions if we do break the API

### References

- [Adding a proposal for hardware accelerators](https://github.com/kubernetes/community/pull/844)
- [Enable "kick the tires" support for NVIDIA GPUs in COS](https://github.com/kubernetes/kubernetes/pull/45136)
- [Extend experimental support to multiple NVIDIA GPUs](https://github.com/kubernetes/kubernetes/pull/42116)
- [Kubernetes Meeting notes](https://docs.google.com/document/d/1Qg42Nmv-QwL4RxicsU2qtZgFKOzANf8fGayw8p3lX6U/edit#)
- [Better Abstraction for Compute Resources in Kubernetes](https://docs.google.com/document/d/1666PPUs4Lz56TqKygcy6mXkNazde-vwA7q4e5H92sUc)
- [Extensible support for hardware devices in Kubernetes (join Kubernetes-dev@googlegroups.com for access)](https://docs.google.com/document/d/1LHeTPx_fWA1PdZkHuALPzYxR0AYXUiiXdo3S0g2VSlo/edit)
