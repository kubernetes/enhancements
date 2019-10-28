---
title: Container Resources CRI API Changes for Pod Vertical Scaling
authors:
  - "@vinaykul"
  - "@quinton-hoole"
owning-sig: sig-node
participating-sigs:
reviewers:
  - TBD
approvers:
  - TBD
editor: TBD
creation-date: 2019-10-25
last-updated: 2019-10-25
status: provisional
see-also:
  - "/keps/sig-autoscaling/20181106-in-place-update-of-pod-resources.md"
replaces:
superseded-by:
---

# Container Resources CRI API Changes for Pod Vertical Scaling

## Table of Contents

<!-- toc -->
- [Summary](#summary)
- [Motivation](#motivation)
  - [Goals](#goals)
  - [Non-Goals](#non-goals)
- [Proposal](#proposal)
  - [Implementation Details/Notes/Constraints [optional]](#implementation-detailsnotesconstraints-optional)
  - [Risks and Mitigations](#risks-and-mitigations)
- [Design Details](#design-details)
  - [Test Plan](#test-plan)
  - [Graduation Criteria](#graduation-criteria)
  - [Upgrade / Downgrade Strategy](#upgrade--downgrade-strategy)
  - [Version Skew Strategy](#version-skew-strategy)
- [Implementation History](#implementation-history)
<!-- /toc -->

## Summary

This proposal aims to improve the Container Runtime Interface (CRI) APIs for
managing CPU and memory resource configurations for Containers. It seeks to
extend UpdateContainerResources CRI API such that it works for Windows, and
other future runtimes besides Linux. It also seeks to add a new CRI API that
allows Kubelet to query the current resources configuration for a Container.

## Motivation

In-Place Pod Vertical Scaling feature relies on Container Runtime Interface
(CRI) to update the CPU and/or memory limits for Container(s) in a Pod.

The current API set have a few issues that need to be addressed:
1. UpdateContainerResources CRI API takes a parameter that describes container
   resources to update for Linux Containers, and this may not work for Windows
   Containers or other potential non-Linux runtimes in the future.
1. There is no CRI mechanism that lets Kubelet query and discover the CPU and
   memory limits configured on a running Container.
1. The expected behavior from a runtime that handles UpdateContainerResources
   CRI API is not very well defined.

### Goals

There are two primary goals of this proposal:
  - Modify UpdateContainerResources to allow it to work for Windows Containers,
    as well as Containers managed by other runtimes in the future,
  - Define a new CRI API to query CPU and memory resource configurations that
    are currently applied to a running Container.

An additional goal of this proposal is to better define the expected behavior
from a Container runtime when handling the above APIs.

### Non-Goals

Definition of expected behavior of a Container runtime when it handles CRI APIs
related to a Container's resources APIs is intended to be a high-level guide.
It is a non-goal of this proposal to define a detailed or specific way to
implement these APIs. Implementation specifics are left to the runtime, within
the bounds of expected behavior.

## Proposal

This is where we get down to the nitty gritty of what the proposal actually is.

One key change is to make UpdateContainerResources API work for Windows, and
any other runtimes that may need to pass resources information in the future.

Another change in this proposal is to add a new GetContainerResources CRI API
that Kubelet can use to query and discover the CPU and memory resources that
are presently configured for a Container.

* A new protobuf message object named *ContainerResources* that encapsulates
LinuxContainerResources and WindowsContainerResources is introduced as below.
  - This message can be easily extended for future runtimes by simply adding a
    new runtime-specific resources struct to the ContainerResources message.
```
// ContainerResources holds resource configuration for a container.
message ContainerResources {
    // Resource configuration specific to Linux container.
    LinuxContainerResources linux = 1;
    // Resource configuration specific to Windows container.
    WindowsContainerResources windows = 2;
}
```

* UpdateContainerResourcesRequest message is extended to carry
  ContainerResources field as below.
  - This allows backward compatibility where runtimes that rely on
    LinuxContainerResources continue to work.
  - Kubelet fills both UpdateContainerResourcesRequest.Linux and
    UpdateContainerResourcesRequest.Resources.Linux fields, allowing
    newer runtimes to use UpdateContainerResourcesRequest.Resources.Linux
    field.
  - This enables deprecation of UpdateContainerResourcesRequest.Linux field.
```
message UpdateContainerResourcesRequest {
    // ID of the container to update.
    string container_id = 1;
    // Resource configuration specific to Linux container.
    LinuxContainerResources linux = 2;
    // Resource configuration for the container.
    ContainerResources resources = 3;
}
```

* A new CRI API, GetContainerResources, is introduced, and RuntimeService is
  modified as shown below.
  - This API enables Kubelet to query and discover currently configured
    resources for a Container.
```
message GetContainerResourcesRequest {
    // ID of the container whose resource config is queried.
    string container_id = 1;
}

message GetContainerResourcesResponse {
    // Resource configuration of the container.
    ContainerResources resources = 1;
}

// GetContainerResources returns resource configuration of the container.
rpc GetContainerResources(GetContainerResourcesRequest) returns (GetContainerResourcesResponse) {}
```

* ContainerManager CRI API service interface is modified as follows:
  - UpdateContainerResources takes ContainerResources parameter instead of
    LinuxContainerResources.
  - GetContainerResources is introduced that allows Kubelet to query current
    resource configurations of a Container.
  - Kubelet code is modified to implement and use these methods.
```
--- a/staging/src/k8s.io/cri-api/pkg/apis/services.go
+++ b/staging/src/k8s.io/cri-api/pkg/apis/services.go
@@ -43,8 +43,10 @@ type ContainerManager interface {
        ListContainers(filter *runtimeapi.ContainerFilter) ([]*runtimeapi.Container, error)
        // ContainerStatus returns the status of the container.
        ContainerStatus(containerID string) (*runtimeapi.ContainerStatus, error)
-       // UpdateContainerResources updates the cgroup resources for the container.
-       UpdateContainerResources(containerID string, resources *runtimeapi.LinuxContainerResources) error
+       // GetContainerResources returns resource configuration applied to the container.
+       GetContainerResources(containerID string) (*runtimeapi.ContainerResources, error)
+       // UpdateContainerResources updates resource configuration for the container.
+       UpdateContainerResources(containerID string, resources *runtimeapi.ContainerResources) error
        // ExecSync executes a command in the container, and returns the stdout output.
        // If command exits with a non-zero exit code, an error is returned.
        ExecSync(containerID string, cmd []string, timeout time.Duration) (stdout []byte, stderr []byte, err error)
```

### Expected Behavior of UpdateContainerResources CRI API

TBD

### Expected Behavior of GetContainerResources CRI API

TBD

## Design Details

TODO: Add a section/flow-diagram on how Kubelet will use these APIs

### Implementation Details/Notes/Constraints [optional]

TBD

### Risks and Mitigations

TBD

### Test Plan

Unit tests: TBD
E2E tests: TBD

### Graduation Criteria

TBD

### Upgrade / Downgrade Strategy

Is this applicable? - TBD

### Version Skew Strategy

Is this applicable? - TBD

## Implementation History

- 2019-10-25 - Initial KEP draft created

