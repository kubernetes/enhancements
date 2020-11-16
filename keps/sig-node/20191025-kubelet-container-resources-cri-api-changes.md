---
title: Container Resources CRI API Changes for Pod Vertical Scaling
authors:
  - "@vinaykul"
  - "@quinton-hoole"
owning-sig: sig-node
participating-groups:
reviewers:
  - "@Random-Liu"
  - "@yujuhong"
  - "@PatrickLang"
approvers:
  - "@dchen1107"
  - "@derekwaynecarr"
editor: TBD
creation-date: 2019-10-25
last-updated: 2020-01-14
status: implementable
see-also:
  - "/keps/sig-node/20181106-in-place-update-of-pod-resources.md"
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
- [Design Details](#design-details)
  - [Expected Behavior of CRI Runtime](#expected-behavior-of-cri-runtime)
  - [Test Plan](#test-plan)
  - [Graduation Criteria](#graduation-criteria)
    - [Alpha](#alpha)
    - [Beta](#beta)
    - [Stable](#stable)
- [Implementation History](#implementation-history)
<!-- /toc -->

## Summary

This proposal aims to improve the Container Runtime Interface (CRI) APIs for
managing a Container's CPU and memory resource configurations on the runtime.
It seeks to extend UpdateContainerResources CRI API such that it works for
Windows, and other future runtimes besides Linux. It also seeks to extend
ContainerStatus CRI API to allow Kubelet to discover the current resources
configured on a Container.

## Motivation

In-Place Pod Vertical Scaling feature relies on Container Runtime Interface
(CRI) to update the CPU and/or memory limits for Container(s) in a Pod.

The current CRI API set has a few drawbacks that need to be addressed:
1. UpdateContainerResources CRI API takes a parameter that describes Container
   resources to update for Linux Containers, and this may not work for Windows
   Containers or other potential non-Linux runtimes in the future.
1. There is no CRI mechanism that lets Kubelet query and discover the CPU and
   memory limits configured on a Container from the Container runtime.
1. The expected behavior from a runtime that handles UpdateContainerResources
   CRI API is not very well defined or documented.

### Goals

This proposal has two primary goals:
  - Modify UpdateContainerResources to allow it to work for Windows Containers,
    as well as Containers managed by other runtimes besides Linux,
  - Provide CRI API mechanism to query the Container runtime for CPU and memory
    resource configurations that are currently applied to a Container.

An additional goal of this proposal is to better define and document the
expected behavior of a Container runtime when handling resource updates.

### Non-Goals

Definition of expected behavior of a Container runtime when it handles CRI APIs
related to a Container's resources is intended to be a high level guide.  It is
a non-goal of this proposal to define a detailed or specific way to implement
these functions. Implementation specifics are left to the runtime, within the
bounds of expected behavior.

## Proposal

One key change is to make UpdateContainerResources API work for Windows, and
any other future runtimes, besides Linux by making the resources parameter
passed in the API specific to the target runtime.

Another change in this proposal is to extend ContainerStatus CRI API such that
Kubelet can query and discover the CPU and memory resources that are presently
applied to a Container.

To accomplish aforementioned goals:

* A new protobuf message object named *ContainerResources* that encapsulates
LinuxContainerResources and WindowsContainerResources is introduced as below.
  - This message can easily be extended for future runtimes by simply adding a
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
  - For Linux runtimes, Kubelet fills UpdateContainerResourcesRequest.Linux in
    additon to UpdateContainerResourcesRequest.Resources.Linux fields.
    - This keeps backward compatibility by letting runtimes that rely on the
      current LinuxContainerResources continue to work, while enabling newer
      runtime versions to use UpdateContainerResourcesRequest.Resources.Linux,
    - It enables deprecation of UpdateContainerResourcesRequest.Linux field.
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

* ContainerStatus message is extended to return ContainerResources as below.
  - This enables Kubelet to query the runtime and discover resources currently
    applied to a Container using ContainerStatus CRI API.
```
@@ -914,6 +912,8 @@ message ContainerStatus {
     repeated Mount mounts = 14;
     // Log path of container.
     string log_path = 15;
+    // Resource configuration of the container.
+    ContainerResources resources = 16;
 }
```

* ContainerManager CRI API service interface is modified as below.
  - UpdateContainerResources takes ContainerResources parameter instead of
    LinuxContainerResources.
```
--- a/staging/src/k8s.io/cri-api/pkg/apis/services.go
+++ b/staging/src/k8s.io/cri-api/pkg/apis/services.go
@@ -43,8 +43,10 @@ type ContainerManager interface {
        ListContainers(filter *runtimeapi.ContainerFilter) ([]*runtimeapi.Container, error)
        // ContainerStatus returns the status of the container.
        ContainerStatus(containerID string) (*runtimeapi.ContainerStatus, error)
-       // UpdateContainerResources updates the cgroup resources for the container.
-       UpdateContainerResources(containerID string, resources *runtimeapi.LinuxContainerResources) error
+       // UpdateContainerResources updates resource configuration for the container.
+       UpdateContainerResources(containerID string, resources *runtimeapi.ContainerResources) error
        // ExecSync executes a command in the container, and returns the stdout output.
        // If command exits with a non-zero exit code, an error is returned.
        ExecSync(containerID string, cmd []string, timeout time.Duration) (stdout []byte, stderr []byte, err error)
```

* Kubelet code is modified to leverage these changes.

## Design Details

Below diagram is an overview of Kubelet using UpdateContainerResources and
ContainerStatus CRI APIs to set new container resource limits, and update the
Pod Status in response to user changing the desired resources in Pod Spec.

```
   +-----------+                   +-----------+                  +-----------+
   |           |                   |           |                  |           |
   | apiserver |                   |  kubelet  |                  |  runtime  |
   |           |                   |           |                  |           |
   +-----+-----+                   +-----+-----+                  +-----+-----+
         |                               |                              |
         |       watch (pod update)      |                              |
         |------------------------------>|                              |
         |     [Containers.Resources]    |                              |
         |                               |                              |
         |                            (admit)                           |
         |                               |                              |
         |                               |  UpdateContainerResources()  |
         |                               |----------------------------->|
         |                               |                         (set limits)
         |                               |<- - - - - - - - - - - - - - -|
         |                               |                              |
         |                               |      ContainerStatus()       |
         |                               |----------------------------->|
         |                               |                              |
         |                               |     [ContainerResources]     |
         |                               |<- - - - - - - - - - - - - - -|
         |                               |                              |
         |      update (pod status)      |                              |
         |<------------------------------|                              |
         | [ContainerStatuses.Resources] |                              |
         |                               |                              |

```

* Kubelet invokes UpdateContainerResources() CRI API in ContainerManager
  interface to configure new CPU and memory limits for a Container by
  specifying those values in ContainerResources parameter to the API. Kubelet
  sets ContainerResources parameter specific to the target runtime platform
  when calling this CRI API.

* Kubelet calls ContainerStatus() CRI API in ContainerManager interface to get
  the CPU and memory limits applied to a Container. It uses the values returned
  in ContainerStatus.Resources to update ContainerStatuses[i].Resources.Limits
  for that Container in the Pod's Status.

### Expected Behavior of CRI Runtime

TBD

### Test Plan

* Unit tests are updated to reflect use of ContainerResources object in
  UpdateContainerResources and ContainerStatus APIs.

* E2E test is added to verify UpdateContainerResources API with docker runtime.

* E2E test is added to verify ContainerStatus API using docker runtime.

* E2E test is added to verify backward compatibility usign docker runtime.

### Graduation Criteria

#### Alpha

* UpdateContainerResources and ContainerStatus API changes are done and tested
  with dockershim and docker runtime, backward compatibility is maintained.

#### Beta

* UpdateContainerResources and ContainerStatus API changes are completed and
  tested for Windows runtime.

#### Stable

* No major bugs reported for three months.

## Implementation History

- 2019-10-25 - Initial KEP draft created
- 2020-01-14 - Test plan and graduation criteria added

