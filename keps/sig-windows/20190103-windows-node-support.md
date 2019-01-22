---
title: Windows node support
authors:
  - "@astrieanna"
  - "@benmoss"
  - "@patricklang"
owning-sig: sig-windows
participating-sigs:
  - sig-architecture
  - sig-node
reviewers:
  - sig-architecture
  - sig-node
approvers:
  - "@bgrant0607"
editor: TBD
creation-date: 2018-11-29
last-updated: 2019-01-21
status: provisional
---

# Windows node support


## Table of Contents
<!-- TOC -->

- [Table of Contents](#table-of-contents)
- [Summary](#summary)
- [Motivation](#motivation)
    - [Goals](#goals)
    - [Non-Goals](#non-goals)
- [Proposal](#proposal)
    - [What works today](#what-works-today)
    - [What will work eventually](#what-will-work-eventually)
    - [What will never work (without underlying OS changes)](#what-will-never-work-without-underlying-os-changes)
    - [Relevant resources/conversations](#relevant-resourcesconversations)
    - [Risks and Mitigations](#risks-and-mitigations)
- [Graduation Criteria](#graduation-criteria)
- [Implementation History](#implementation-history)
- [Testing Plan](#testing-plan)
    - [Adapting existing tests](#adapting-existing-tests)
    - [Test Dashboard](#test-dashboard)
    - [Test Approach](#test-approach)
        - [Adapting existing tests](#adapting-existing-tests-1)
    - [Substitute test cases](#substitute-test-cases)
        - [Substitute test cases](#substitute-test-cases-1)
    - [Windows specific tests](#windows-specific-tests)
        - [Windows specific tests](#windows-specific-tests-1)
- [Other references](#other-references)

<!-- /TOC -->

## Summary

There is strong interest in the community for adding support for workloads running on Microsoft Windows. This is non-trivial due to the significant differences in the implementation of Windows from the Linux-based OSes that have so far been supported by Kubernetes.


## Motivation

Windows-native workloads still account for a significant portion of the enterprise software space. While containerization technologies emerged first in the UNIX ecosystem, Microsoft has made investments in recent years to enable support for containers in its Windows OS. As users of Windows increasingly turn to containers as the preferred abstraction for running software, the Kubernetes ecosystem stands to benefit by becoming a cross-platform cluster manager.

### Goals

- Enable users to run nodes on Windows servers 
- Document the differences and limitations compared to Linux
- Test results added to testgrid to prevent regression of functionality 

### Non-Goals

- Adding Windows support to all projects in the Kubernetes ecosystem (Cluster Lifecycle, etc)

## Proposal

As of 29-11-2018 much of the work for enabling Windows nodes has already been completed. Both `kubelet` and `kube-proxy` have been adapted to work on Windows Server, and so the first goal of this KEP is largely already complete. 

### What works today
- Windows-based containers can be created by kubelet, [provided the host OS version matches the container base image](https://docs.microsoft.com/en-us/virtualization/windowscontainers/deploy-containers/version-compatibility)
    - ConfigMap, Secrets: as environment variables or  volumes
    - Resource limits
    - Pod & container metrics
- Pod networking with [Azure-CNI](https://github.com/Azure/azure-container-networking/blob/master/docs/cni.md), [OVN-Kubernetes](https://github.com/openvswitch/ovn-kubernetes), [two CNI meta-plugins](https://github.com/containernetworking/plugins), [Flannel](https://github.com/coreos/flannel) and [Calico](https://github.com/projectcalico/calico)
- Dockershim CRI
- Many<sup id="a1">[1]</sup> of the e2e conformance tests when run with [alternate Windows-based images](https://hub.docker.com/r/e2eteam/) which are being moved to [kubernetes-sigs/windows-testing](https://www.github.com/kubernetes-sigs/windows-testing)
- Persistent storage: FlexVolume with [SMB + iSCSI](https://github.com/Microsoft/K8s-Storage-Plugins/tree/master/flexvolume/windows), and in-tree AzureFile and AzureDisk providers
 
### What will work eventually
- `kubectl port-forward` hasn't been implemented due to lack of an `nsenter` equivalent to run a process inside a network namespace.
- CRIs other than Dockershim: CRI-containerd support is forthcoming

### What will never work (without underlying OS changes)
- Certain Pod functionality
    - Privileged containers
    - Reservations are not enforced by the OS, but overprovisioning could be blocked with `--enforce-node-allocatable=pods` (pending: tests needed)
    - Certain volume mappings
      - Single file & subpath volume mounting
      - Host mount projection
      - DefaultMode (due to UID/GID dependency)
      - readOnly root filesystem. Mapped volumes still support readOnly
    - Termination Message - these require single file mappings
- CSI plugins, which require privileged containers
- [Some parts of the V1 API](https://github.com/kubernetes/kubernetes/issues/70604)
- Overlay networking support in Windows Server 1803 is not fully functional using the `win-overlay` CNI plugin. Specifically service IPs do not work on Windows nodes. This is currently specific to `win-overlay` - other CNI plugins (OVS, AzureCNI) work.

### Relevant resources/conversations

- [sig-architecture thread](https://groups.google.com/forum/#!topic/kubernetes-sig-architecture/G2zKJ7QK22E)
- [cncf-k8s-conformance thread](https://lists.cncf.io/g/cncf-k8s-conformance/topic/windows_conformance_tests/27913232)
- [kubernetes/enhancements proposal](https://github.com/kubernetes/features/issues/116)


### Risks and Mitigations

**Second class support**: Kubernetes contributors are likely to be thinking of Linux-based solutions to problems, as Linux remains the primary OS supported. Keeping Windows support working will be an ongoing burden potentially limiting the pace of development. 

**User experience**: Users today will need to use some combination of taints and node selectors in order to keep Linux and Windows workloads separated. In the best case this imposes a burden only on Windows users, but this is still less than ideal.

## Graduation Criteria


## Implementation History


## Testing Plan

<<<<<<< HEAD
The testing for Windows nodes will include multiple approaches:

1. [Adapting](#Adapting-existing-tests) some of the existing conformance tests to be able to pass on multiple node OS's
2. Adding [substitute](#Substitute-test-cases) test cases where the first approach isn't feasible or would change the tests in a way is not approved by the owner. These will be tagged with `[SIG-Windows]`
3. Last, gaps will be filled with [Windows specific tests](#Windows-specific-tests). These will also be tagged with `[SIG-Windows]`

All test cases will be built in kubernetes/test/e2e, scheduled through [prow](github.com/kubernetes/test-infra/blob/master/config/jobs/kubernetes-sigs/sig-windows/sig-windows-config.yaml), and published on the [TestGrid SIG-Windows dashboard](https://testgrid.k8s.io/sig-windows) daily.

Windows test setup scripts, container image source, and documentation will be kept in the [kubernetes-sigs/windows-testing](https://github.com/kubernetes-sigs/windows-testing) repo.

### Adapting existing tests
=======

### Test Dashboard

All test cases will be built in kubernetes/test/e2e, scheduled through [prow](github.com/kubernetes/test-infra/blob/master/config/jobs/kubernetes-sigs/sig-windows/sig-windows-config.yaml), and published on the [TestGrid SIG-Windows dashboard](https://testgrid.k8s.io/sig-windows) daily. This will be the master list of what needs to pass to be declared stable and will include all tests tagged [SIG-Windows] along with the subset of conformance tests that can pass on Windows.


Additional dashboard pages will be added over time as we run the same test cases with additional CRI, CNI and cloud providers. They reflect work that may be stabilized in v1.15 or later and is not strictly required for v1.14.

- Windows Server 2019 on GCP - this is [in progress](https://k8s-testgrid.appspot.com/google-windows#windows-prototype), but not required for v1.14
- Windows Server 2019 with OVN+OVS & Dockershim
- Windows Server 2019 with OVN+OVS & CRI-ContainerD
- Windows Server 2019 with Azure-CNI & CRI-ContainerD
- Windows Server 2019 with Flannel & CRI-ContainerD

### Test Approach

The testing for Windows nodes will include multiple approaches:

1. [Adapting](#Adapting-existing-tests) some of the existing conformance tests to be able to pass on multiple node OS's. Tests that won't work will be [excluded](https://github.com/kubernetes/test-infra/blob/master/config/jobs/kubernetes-sigs/sig-windows/sig-windows-config.yaml#L69).
  - [ ] TODO: switch to using a tag instead of a long exclusion list
2. Adding [substitute](#Substitute-test-cases) test cases where the first approach isn't feasible or would change the tests in a way is not approved by the owner. These will be tagged with `[SIG-Windows]`
3. Last, gaps will be filled with [Windows specific tests](#Windows-specific-tests). These will also be tagged with `[SIG-Windows]`

All of the test cases will be maintained within the kubernetes/kubernetes repo. SIG-Windows specific tests for 2/3 will be in [test/e2e/windows](https://github.com/kubernetes/kubernetes/tree/master/test/e2e/windows)

Additional Windows test setup scripts, container image source code, and documentation will be kept in the [kubernetes-sigs/windows-testing](https://github.com/kubernetes-sigs/windows-testing) repo. One example is that the prow jobs need a list of repos to use for the test containers, and that will be maintained here - see [windows-testing#1](https://github.com/kubernetes-sigs/windows-testing/issues/1).


#### Adapting existing tests
>>>>>>> a8dcbdc65241ef129929ef0ac401a884965b2a08

Over the course of v1.12/13, many conformance tests were adapted to be able to pass on either Linux or Windows nodes as long as matching OS containers are run. This was done by creating Windows equivalent containers from [kubernetes/test/images](https://github.com/kubernetes/kubernetes/tree/master/test/images). An additional parameter is needed for e2e.test/kubetest to change the container repos to the one containing Windows versions since they're not part of the Kubernetes build process yet.

> TODO: verify against list of test cases from https://docs.google.com/document/d/1YkLZIYYLMQhxdI2esN5PuTkhQHhO0joNvnbHpW68yg8/edit#

- [k8s.io] Container Lifecycle Hook when create a pod with lifecycle hook should execute poststart exec hook properly [NodeConformance] [Conformance]
- [k8s.io] Container Lifecycle Hook when create a pod with lifecycle hook should execute poststart http hook properly [NodeConformance] [Conformance]
- [k8s.io] Container Lifecycle Hook when create a pod with lifecycle hook should execute prestop exec hook properly [NodeConformance] [Conformance]
- [k8s.io] Container Lifecycle Hook when create a pod with lifecycle hook should execute prestop http hook properly [NodeConformance] [Conformance]
- [k8s.io] Docker Containers should be able to override the image's default arguments (docker cmd) [NodeConformance] [Conformance]
- [k8s.io] Docker Containers should be able to override the image's default command (docker entrypoint) [NodeConformance] [Conformance]
- [k8s.io] Docker Containers should be able to override the image's default command and arguments [NodeConformance] [Conformance]
- [k8s.io] Docker Containers should use the image defaults if command and args are blank [NodeConformance] [Conformance]
- [k8s.io] InitContainer [NodeConformance] should invoke init containers on a RestartAlways pod [Conformance]
- [k8s.io] InitContainer [NodeConformance] should invoke init containers on a RestartNever pod [Conformance]
- [k8s.io] InitContainer [NodeConformance] should not start app containers and fail the pod if init containers fail on a RestartNever pod [Conformance]
- [k8s.io] InitContainer [NodeConformance] should not start app containers if init containers fail on a RestartAlways pod [Conformance]
- [k8s.io] Kubelet when scheduling a busybox command in a pod should print the output to logs [NodeConformance] [Conformance]
- [k8s.io] Kubelet when scheduling a busybox command that always fails in a pod should be possible to delete [NodeConformance] [Conformance]
- [k8s.io] Kubelet when scheduling a busybox command that always fails in a pod should have an terminated reason [NodeConformance] [Conformance]
- [k8s.io] Pods should allow activeDeadlineSeconds to be updated [NodeConformance] [Conformance]
- [k8s.io] Pods should be submitted and removed [NodeConformance] [Conformance]
- [k8s.io] Pods should be updated [NodeConformance] [Conformance]
- [k8s.io] Pods should cap back-off at MaxContainerBackOff [Slow][NodeConformance]
- [k8s.io] Pods should contain environment variables for services [NodeConformance] [Conformance]
- [k8s.io] Pods should get a host IP [NodeConformance] [Conformance]
- [k8s.io] Pods should have their auto-restart back-off timer reset on image update [Slow][NodeConformance]
- [k8s.io] Pods should support remote command execution over websockets [NodeConformance] [Conformance]
- [k8s.io] Pods should support retrieving logs from the container over websockets [NodeConformance] [Conformance]
- [k8s.io] Probing container should *not* be restarted with a /healthz http liveness probe [NodeConformance] [Conformance]
- [k8s.io] Probing container should *not* be restarted with a exec "cat /tmp/health" liveness probe [NodeConformance] [Conformance]
- [k8s.io] Probing container should be restarted with a /healthz http liveness probe [NodeConformance] [Conformance]
- [k8s.io] Probing container should be restarted with a exec "cat /tmp/health" liveness probe [NodeConformance] [Conformance]
- [k8s.io] Probing container should have monotonically increasing restart count [NodeConformance] [Conformance]
- [k8s.io] Probing container should have monotonically increasing restart count [Slow][NodeConformance] [Conformance]
- [k8s.io] Probing container with readiness probe should not be ready before initial delay and never restart [NodeConformance] [Conformance]
- [k8s.io] Probing container with readiness probe that fails should never be ready and never restart [NodeConformance] [Conformance]
- [k8s.io] Security Context When creating a pod with readOnlyRootFilesystem should run the container with writable rootfs when readOnlyRootFilesystem=false [NodeConformance]
- [k8s.io] Variable Expansion should allow composing env vars into new env vars [NodeConformance] [Conformance]
- [k8s.io] Variable Expansion should allow substituting values in a container's args [NodeConformance] [Conformance]
- [k8s.io] Variable Expansion should allow substituting values in a container's command [NodeConformance] [Conformance]
- [k8s.io] [sig-node] Events should be sent by kubelets and the scheduler about pods scheduling and running [Conformance]
- [k8s.io] [sig-node] Pods Extended [k8s.io] Pods Set QOS Class should be submitted and removed [Conformance]
- [k8s.io] [sig-node] PreStop should call prestop when killing a pod [Conformance]
- [sig-api-machinery] CustomResourceDefinition resources Simple CustomResourceDefinition creating/deleting custom resource definition objects works [Conformance]
- [sig-api-machinery] Garbage collector should delete RS created by deployment when not orphaning [Conformance]
- [sig-api-machinery] Garbage collector should delete pods created by rc when not orphaning [Conformance]
- [sig-api-machinery] Garbage collector should keep the rc around until all its pods are deleted if the deleteOptions says so [Conformance]
- [sig-api-machinery] Garbage collector should not be blocked by dependency circle [Conformance]
- [sig-api-machinery] Garbage collector should not delete dependents that have both valid owner and owner that's waiting for dependents to be deleted [Conformance]
- [sig-api-machinery] Garbage collector should orphan RS created by deployment when deleteOptions.PropagationPolicy is Orphan [Conformance]
- [sig-api-machinery] Garbage collector should orphan pods created by rc if delete options say so [Conformance]
- [sig-api-machinery] Namespaces [Serial] should ensure that all pods are removed when a namespace is deleted [Conformance]
- [sig-api-machinery] Namespaces [Serial] should ensure that all services are removed when a namespace is deleted [Conformance]
- [sig-api-machinery] Secrets should be consumable from pods in env vars [NodeConformance] [Conformance]
- [sig-api-machinery] Secrets should be consumable via the environment [NodeConformance] [Conformance]
- [sig-api-machinery] Watchers should be able to restart watching from the last resource version observed by the previous watch [Conformance]
- [sig-api-machinery] Watchers should be able to start watching from a specific resource version [Conformance]
- [sig-api-machinery] Watchers should observe add, update, and delete watch notifications on configmaps [Conformance]
- [sig-api-machinery] Watchers should observe an object deletion if it stops meeting the requirements of the selector [Conformance]
- [sig-apps] Daemon set [Serial] should retry creating failed daemon pods [Conformance]
- [sig-apps] Daemon set [Serial] should run and stop complex daemon [Conformance]
- [sig-apps] Daemon set [Serial] should run and stop simple daemon [Conformance]
- [sig-apps] Daemon set [Serial] should update pod when spec was updated and update strategy is RollingUpdate [Conformance]
- [sig-apps] Deployment RecreateDeployment should delete old pods and create new ones [Conformance]
- [sig-apps] Deployment RollingUpdateDeployment should delete old pods and create new ones [Conformance]
- [sig-apps] Deployment deployment should delete old replica sets [Conformance]
- [sig-apps] Deployment deployment should support proportional scaling [Conformance]
- [sig-apps] Deployment deployment should support rollover [Conformance]
- [sig-apps] ReplicaSet should adopt matching pods on creation and release no longer matching pods [Conformance]
- [sig-apps] ReplicaSet should serve a basic image on each replica with a public image [Conformance]
- [sig-apps] ReplicationController should adopt matching pods on creation [Conformance]
- [sig-apps] ReplicationController should release no longer matching pods [Conformance]
- [sig-apps] ReplicationController should serve a basic image on each replica with a public image [Conformance]
- [sig-apps] StatefulSet [k8s.io] Basic StatefulSet functionality [StatefulSetBasic] Burst scaling should run to completion even with unhealthy pods [Conformance]
- [sig-apps] StatefulSet [k8s.io] Basic StatefulSet functionality [StatefulSetBasic] Scaling should happen in predictable order and halt if any stateful pod is unhealthy [Conformance]
- [sig-apps] StatefulSet [k8s.io] Basic StatefulSet functionality [StatefulSetBasic] Should recreate evicted statefulset [Conformance]
- [sig-apps] StatefulSet [k8s.io] Basic StatefulSet functionality [StatefulSetBasic] should perform canary updates and phased rolling updates of template modifications [Conformance]
- [sig-apps] StatefulSet [k8s.io] Basic StatefulSet functionality [StatefulSetBasic] should perform rolling updates and roll backs of template modifications [Conformance]
- [sig-auth] ServiceAccounts should allow opting out of API token automount [Conformance]
- [sig-auth] ServiceAccounts should mount an API token into pods [Conformance]
- [sig-cli] Kubectl client [k8s.io] Guestbook application should create and stop a working application [Conformance]
- [sig-cli] Kubectl client [k8s.io] Kubectl api-versions should check if v1 is in available api versions [Conformance]
- [sig-cli] Kubectl client [k8s.io] Kubectl cluster-info should check if Kubernetes master services is included in cluster-info [Conformance]
- [sig-cli] Kubectl client [k8s.io] Kubectl describe should check if kubectl describe prints relevant information for rc and pods [Conformance]
- [sig-cli] Kubectl client [k8s.io] Kubectl expose should create services for rc [Conformance]
- [sig-cli] Kubectl client [k8s.io] Kubectl label should update the label on a resource [Conformance]
- [sig-cli] Kubectl client [k8s.io] Kubectl logs should be able to retrieve and filter logs [Conformance]
- [sig-cli] Kubectl client [k8s.io] Kubectl patch should add annotations for pods in rc [Conformance]
- [sig-cli] Kubectl client [k8s.io] Kubectl replace should update a single-container pod's image [Conformance]
- [sig-cli] Kubectl client [k8s.io] Kubectl rolling-update should support rolling-update to same image [Conformance]
- [sig-cli] Kubectl client [k8s.io] Kubectl run --rm job should create a job from an image, then delete the job [Conformance]
- [sig-cli] Kubectl client [k8s.io] Kubectl run default should create an rc or deployment from an image [Conformance]
- [sig-cli] Kubectl client [k8s.io] Kubectl run deployment should create a deployment from an image [Conformance]
- [sig-cli] Kubectl client [k8s.io] Kubectl run job should create a job from an image when restart is OnFailure [Conformance]
- [sig-cli] Kubectl client [k8s.io] Kubectl run pod should create a pod from an image when restart is Never [Conformance]
- [sig-cli] Kubectl client [k8s.io] Kubectl run rc should create an rc from an image [Conformance]
- [sig-cli] Kubectl client [k8s.io] Kubectl version should check is all data is printed [Conformance]
- [sig-cli] Kubectl client [k8s.io] Proxy server should support --unix-socket=/path [Conformance]
- [sig-cli] Kubectl client [k8s.io] Proxy server should support proxy with --port 0 [Conformance]
- [sig-cli] Kubectl client [k8s.io] Update Demo should create and stop a replication controller [Conformance]
- [sig-cli] Kubectl client [k8s.io] Update Demo should do a rolling update of a replication controller [Conformance]
- [sig-cli] Kubectl client [k8s.io] Update Demo should scale a replication controller [Conformance]
- [sig-network] Proxy version v1 should proxy logs on node using proxy subresource [Conformance]
- [sig-network] Proxy version v1 should proxy logs on node with explicit kubelet port using proxy subresource [Conformance]
- [sig-network] Proxy version v1 should proxy through a service and a pod [Conformance]
- [sig-network] Service endpoints latency should not be very high [Conformance]
- [sig-network] Services should provide secure master service [Conformance]
- [sig-network] Services should serve a basic endpoint from pods [Conformance]
- [sig-network] Services should serve multiport endpoints from pods [Conformance]
- [sig-node] ConfigMap should be consumable via environment variable [NodeConformance] [Conformance]
- [sig-node] ConfigMap should be consumable via the environment [NodeConformance] [Conformance]
- [sig-node] Downward API should provide container's limits.cpu/memory and requests.cpu/memory as env vars [NodeConformance] [Conformance]
- [sig-node] Downward API should provide default limits.cpu/memory from node allocatable [NodeConformance] [Conformance]
- [sig-node] Downward API should provide host IP as an env var [NodeConformance] [Conformance]
- [sig-node] Downward API should provide pod UID as env vars [NodeConformance] [Conformance]
- [sig-node] Downward API should provide pod name, namespace and IP address as env vars [NodeConformance] [Conformance]
- [sig-scheduling] SchedulerPredicates [Serial] validates resource limits of pods that are allowed to run [Conformance]
- [sig-scheduling] SchedulerPredicates [Serial] validates that NodeSelector is respected if matching [Conformance]
- [sig-scheduling] SchedulerPredicates [Serial] validates that NodeSelector is respected if not matching [Conformance]
- [sig-storage] ConfigMap binary data should be reflected in volume [NodeConformance] [Conformance]
- [sig-storage] ConfigMap optional updates should be reflected in volume [NodeConformance] [Conformance]
- [sig-storage] ConfigMap should be consumable from pods in volume [NodeConformance] [Conformance]
- [sig-storage] ConfigMap should be consumable from pods in volume with mappings [NodeConformance] [Conformance]
- [sig-storage] ConfigMap should be consumable in multiple volumes in the same pod [NodeConformance] [Conformance]
- [sig-storage] ConfigMap updates should be reflected in volume [NodeConformance] [Conformance]
- [sig-storage] Downward API volume should provide container's cpu limit [NodeConformance] [Conformance]
- [sig-storage] Downward API volume should provide container's cpu request [NodeConformance] [Conformance]
- [sig-storage] Downward API volume should provide container's memory limit [NodeConformance] [Conformance]
- [sig-storage] Downward API volume should provide container's memory request [NodeConformance] [Conformance]
- [sig-storage] Downward API volume should provide node allocatable (cpu) as default cpu limit if the limit is not set [NodeConformance] [Conformance]
- [sig-storage] Downward API volume should provide node allocatable (memory) as default memory limit if the limit is not set [NodeConformance] [Conformance]
- [sig-storage] Downward API volume should provide podname only [NodeConformance] [Conformance]
- [sig-storage] Downward API volume should update annotations on modification [NodeConformance] [Conformance]
- [sig-storage] Downward API volume should update labels on modification [NodeConformance] [Conformance]
- [sig-storage] EmptyDir wrapper volumes should not cause race condition when used for configmaps [Serial] [Conformance]
- [sig-storage] EmptyDir wrapper volumes should not cause race condition when used for configmaps [Serial] [Slow] [Conformance]
- [sig-storage] EmptyDir wrapper volumes should not conflict [Conformance]
- [sig-storage] HostPath should support r/w [NodeConformance]
- [sig-storage] HostPath should support subPath [NodeConformance]
- [sig-storage] Projected combined should project all components that make up the projection API [Projection][NodeConformance] [Conformance]
- [sig-storage] Projected configMap optional updates should be reflected in volume [NodeConformance] [Conformance]
- [sig-storage] Projected configMap should be consumable from pods in volume [NodeConformance] [Conformance]
- [sig-storage] Projected configMap should be consumable from pods in volume with mappings [NodeConformance] [Conformance]
- [sig-storage] Projected configMap should be consumable in multiple volumes in the same pod [NodeConformance] [Conformance]
- [sig-storage] Projected configMap updates should be reflected in volume [NodeConformance] [Conformance]
- [sig-storage] Projected downwardAPI should provide container's cpu limit [NodeConformance] [Conformance]
- [sig-storage] Projected downwardAPI should provide container's cpu request [NodeConformance] [Conformance]
- [sig-storage] Projected downwardAPI should provide container's memory limit [NodeConformance] [Conformance]
- [sig-storage] Projected downwardAPI should provide container's memory request [NodeConformance] [Conformance]
- [sig-storage] Projected downwardAPI should provide node allocatable (cpu) as default cpu limit if the limit is not set [NodeConformance] [Conformance]
- [sig-storage] Projected downwardAPI should provide node allocatable (memory) as default memory limit if the limit is not set [NodeConformance] [Conformance]
- [sig-storage] Projected downwardAPI should provide podname only [NodeConformance] [Conformance]
- [sig-storage] Projected downwardAPI should update annotations on modification [NodeConformance] [Conformance]
- [sig-storage] Projected downwardAPI should update labels on modification [NodeConformance] [Conformance]
- [sig-storage] Projected secret optional updates should be reflected in volume [NodeConformance] [Conformance]
- [sig-storage] Projected secret should be able to mount in a volume regardless of a different secret existing with same name in different namespace [NodeConformance]
- [sig-storage] Projected secret should be consumable from pods in volume [NodeConformance] [Conformance]
- [sig-storage] Projected secret should be consumable from pods in volume with mappings [NodeConformance] [Conformance]
- [sig-storage] Projected secret should be consumable in multiple volumes in a pod [NodeConformance] [Conformance]
- [sig-storage] Secrets optional updates should be reflected in volume [NodeConformance] [Conformance]
- [sig-storage] Secrets should be able to mount in a volume regardless of a different secret existing with same name in different namespace [NodeConformance] [Conformance]
- [sig-storage] Secrets should be consumable from pods in volume [NodeConformance] [Conformance]
- [sig-storage] Secrets should be consumable from pods in volume with mappings [NodeConformance] [Conformance]
- [sig-storage] Secrets should be consumable in multiple volumes in a pod [NodeConformance] [Conformance]

<<<<<<< HEAD
### Substitute test cases
=======
#### Substitute test cases
>>>>>>> a8dcbdc65241ef129929ef0ac401a884965b2a08

These are test cases that follow a similar flow to a conformance test that is dependent on Linux-specific functionality, but differs enough that the same test case cannot be used for both Windows & Linux. Examples include differences in file access permissions (UID/GID vs username, permission octets vs Windows ACLs), and network configuration (`/etc/resolv.conf` is used on Linux, but Windows DNS settings are stored in the Windows registry).

> TODO: include list of test cases from open PRs

<<<<<<< HEAD
### Windows specific tests

We will also add Windows scenario-specific tests to cover more typical use cases and features specific to Windows. These tests will be in [kubernetes/test/e2e/windows](https://github.com/kubernetes/kubernetes/tree/master/test/e2e/windows). This will also include density and performance tests that are adjusted for Windows apps which have different image sizes and memory requirements.

> TODO: new list here
=======

TODO List:
- [ ] DNS configuration is passed through CNI, not `/etc/resolv.conf` [67435](https://github.com/kubernetes/kubernetes/pull/67435)
- [ ] Windows doesn't have CGroups, but nodeReserve and kubeletReserve are [implemented](https://github.com/kubernetes/kubernetes/pull/69960)




#### Windows specific tests

We will also add Windows scenario-specific tests to cover more typical use cases and features specific to Windows. These tests will be in [kubernetes/test/e2e/windows](https://github.com/kubernetes/kubernetes/tree/master/test/e2e/windows). This will also include density and performance tests that are adjusted for Windows apps which have different image sizes and memory requirements.

Here's a list of functionality that needs tests written. 

- [ ] System, pod & network stats are implemented in kubelet, not cadvisor [70212](https://github.com/kubernetes/kubernetes/pull/70121), [66427](https://github.com/kubernetes/kubernetes/pull/66427), [62266](https://github.com/kubernetes/kubernetes/pull/62266), [51152](https://github.com/kubernetes/kubernetes/pull/51152), [50396](https://github.com/kubernetes/kubernetes/pull/50396)
- [ ] Windows uses username (string) or SID (binary) to define users, not UID/GID [64009](https://github.com/kubernetes/kubernetes/pull/64009)

>>>>>>> a8dcbdc65241ef129929ef0ac401a884965b2a08

## Other references

[Past release proposal for v1.12/13](https://docs.google.com/document/d/1YkLZIYYLMQhxdI2esN5PuTkhQHhO0joNvnbHpW68yg8/edit#)
