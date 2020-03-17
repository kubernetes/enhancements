---
title: Add Request-ID to each Kubernetes component log
authors:
 - "@hase1128"
 - "@sshukun"
 - "@furukawa3"
 - "@vanou"
owning-sig: sig-instrumentation
participating-sigs:
reviewers:
 - TBD
approvers:
 - TBD
editor: TBD
creation-date: 2019-11-01
last-updated: 2020-03-17
status: provisional
---

# Add Request-ID to each Kubernetes component log

## Table of Contents

<!-- toc -->
 - [Summary](#summary)
 - [Motivation](#motivation)
   - [Target User](#target-user)
   - [Target User's objective](#target-users-objective)
   - [Case 1](#case-2)
   - [Case 2](#case-1)
   - [Goals](#goals)
   - [Non-Goals](#non-goals)
 - [Proposal](#proposal)
   - [Basic policy of Request-ID](#basic-policy-of-request-id)
   - [Design Overview](#design-overview)
     - [Design of Propagate Request-ID](#design-of-propagate-request-id)
     - [Design of Export Request-ID](#design-of-export-request-id)
     - [Design overview of Control Request-ID](#design-overview-of-control-request-id)
     - [Detail design of Control Request-ID](#detail-design-of-control-request-id)
 - [Test Plan](#test-plan)
 - [Migration / Graduation Criteria](#migration--graduation-criteria)
   - [Alpha](#alpha)
   - [Beta](#beta)
   - [GA](#ga)

<!-- /toc -->

## Summary

This KEP proposes a new unique logging meta-data into all Kubernetes logs. It makes us
more easy to identify specific logs related to a single user operation (such as
`kubectl apply -f <my-pod.yaml>`). This feature is similar to
[Global request ID](https://docs.openstack.org/api-guide/compute/faults.html) for
OpenStack. It greatly reduces investigation cost.

## Motivation

### Target User

Support team in k8s Service Provider

### Target User's objective

We'd like to resolve quickly for end users' problem.

Tracking logs among each Kubernetes component related to specific an user operation is very tough work. It is necessary to match logs by basically using timestamps and object's name as hints. If multiple users throw many API requests at the same time, it is very difficult to track logs across each Kubernetes component log. 

It is difficult that support team in k8s Service Provider resolve end user's problem quickly in the above. Therefore, we'd like to add a new identifier which is unique to each user operation. This feature is useful for the following use cases:

#### Case 1

In case of insecure or unauthorized operation happens, it is necessary to
identify what effect that operation caused. This proposed feature helps identify
what happened at each component or server by each insecure / unauthorized API
request. We can collect these logs as an evidence. This is similar to the 
[Auditing](https://kubernetes.io/docs/tasks/debug-application-cluster/audit/), 
except for the following points.

 - Audit only collects information about http request sending and receiving in kube-apiserver, so it can't track internal work of each component.
 - Audit logs can't be associated to logs related to user operation (kubectl operation), because auditID is different for each http request.


#### Case 2

If the container is terminated by OOM killer, there is a case to break down the
issue into parts(Pod or Kubernetes) from the messages related OOM killer on host logs
and the API processing just before OOM killer. If the cause is that some unknown
pod creations, it is helpful to detect the root API request and who called this
request.

### Goals

 - Adding a Request-ID into each K8s component log.
 - The Request-ID is unique to a kubectl operation.
   - (One kubectl operation by user causes multiple API requests and klog calls. Request-ID has same value in these klog calls.)
 - Control enabled/disabled Request-ID feature(Request-ID feature is disabled on default to avoid an impact for existing user).

### Non-Goals

 - To centrally manage the logs of each Kubernetes component with Request-ID (This can
be realized with existing OSS such as Kibana, so no need to implement into Kubernetes
components).
 - We don't associate Request-ID to all of operations(Our target is important operations such as `kubectl create/delete/etc.`).

## Proposal

### Basic policy of Request-ID

 - Minimize the impact to existing users who are retrieving logs and analyzing with existing log format.
   - So we disabled Request-ID feature on default.
 - Collaborate with related KEPs to avoid unnecessary conflict to them regarding implementation and feature.
   - Use existing KEP's feature as much as possible.
   - Therefore, we will merge Request-ID feature after related KEP features are merged.

### Design Overview

Request-ID feature consists the three features.
 - Propagate Request-ID to related objects.
 - Export Request-ID to each Kubernetes component log.
 - Control enabled/disabled exporting Request-ID.

#### Design of Propagate Request-ID

There is an idea to use `distributed context` of the existing KEP([Distributed Tracing](https://github.com/kubernetes/enhancements/blob/master/keps/sig-instrumentation/0034-distributed-tracing-kep.md)). We use `Distributed Tracing` feature for propagation.

 - We just use Tracing codes provided by `Distributed Tracing` feature.
 - We use Trace-ID as Request-ID (This does not interfere Tracing codes).

The target functions that we add Request-id codes are the following tables. The target functions include the klog call which called via important `kubectl` operations. We prioritize `kubectl` operations and target resources by the following criteria.

 - Operations that are frequently used by user
 - Resources that are user's application and asset

Operations against Pod(Deployment) are the most frequently used by user. PV / PVC stores user assets. In addition to this, network is relatively troublesome for orchestration. These operations are listed in [Migration / Graduation Criteria](#migration--graduation-criteria) section. As a result, we can propagate Request-ID regarding important `kubectl` operations.

Here is the target functions that we add Request-id codes in **Alpha implementation**.

**kube-apiserver.log**

| source file | function name |
| ------ | ------ |
| staging/src/k8s.io/apiserver/pkg/endpoints/handlers/get.go | ListResource |
| staging/src/k8s.io/apiserver/pkg/server/httplog/httplog.go | WithLogging |

**kube-controller-manager.log**

| source file | function name |
| ------ | ------ |
| pkg/controller/deployment/deployment_controller.go | syncDeployment |
| pkg/controller/deployment/deployment_controller.go | handleErr |
| pkg/controller/garbagecollector/garbagecollector.go | attemptToDeleteItem |
| pkg/controller/replicaset/replica_set.go | manageReplicas |

**kubelet.log**

| source file | function name |
| ------ | ------ |
| pkg/volume/emptydir/empty_dir.go | setupTmpfs |
| pkg/kubelet/kubelet.go | HandlePodAdditions |
| pkg/kubelet/kubelet.go | HandlePodUpdates |
| pkg/kubelet/kubelet.go | HandlePodRemoves |
| pkg/kubelet/kubelet.go | HandlePodReconcile |
| pkg/kubelet/kubelet_pods.go | cleanupOrphanedPodCgroups |  |
| pkg/kubelet/kubelet_pods.go | makeMounts |
| pkg/kubelet/kubelet_pods.go | PodResourcesAreReclaimed |
| pkg/kubelet/kubelet_pods.go | podKiller |
| pkg/kubelet/kubelet_pods.go | generateAPIPodStatus |
| pkg/kubelet/kubelet_pods.go | cleanupOrphanedPodCgroups |
| pkg/kubelet/kubelet_volumes.go | cleanupOrphanedPodDirs |
| pkg/kubelet/kuberuntime/kuberuntime_container.go | killContainer |
| pkg/kubelet/kuberuntime/kuberuntime_image.go | PullImage |
| pkg/kubelet/kuberuntime/kuberuntime_manager.go | podSandboxChanged |
| pkg/kubelet/kuberuntime/kuberuntime_manager.go | killPodWithSyncResult |
| pkg/kubelet/kuberuntime/kuberuntime_sandbox.go | createPodSandbox |
| pkg/kubelet/dockershim/docker_service.go | GenerateExpectedCgroupParent |
| pkg/kubelet/dockershim/libdocker/kube_docker_client.go | start |
| pkg/volume/util/operationexecutor/operation_generator.go | GenerateMountVolumeFunc |
| pkg/volume/util/operationexecutor/operation_generator.go | GenerateUnmountVolumeFunc |
| pkg/kubelet/dockershim/network/plugins.go | SetUpPod |
| pkg/kubelet/dockershim/network/plugins.go | TearDownPod |
| pkg/kubelet/pod_container_deletor.go:75 | getContainersToDeleteInPod |
| pkg/kubelet/volumemanager/reconciler/reconciler.go | unmountVolumes |
| pkg/kubelet/volumemanager/reconciler/reconciler.go | mountAttachVolumes |
| pkg/kubelet/volumemanager/reconciler/reconciler.go | unmountDetachDevices |
| pkg/kubelet/remote/remote_runtime.go | ContainerStatus |
| pkg/volume/secret/secret.go | SetUpAt |
| pkg/kubelet/status/status_manager.go | updateStatusInternal |
| pkg/kubelet/status/status_manager.go | syncPod |
| pkg/volume/util/util.go | UnmountViaEmptyDir |
| pkg/kubelet/volumemanager/volume_manager.go | WaitForAttachAndMount |

**kube-proxy**

```
NONE
```

**kube-scheduler**

| source file | function name |
| ------ | ------ |
| pkg/scheduler/framework/plugins/defaultbinder/default_binder.go | Bind |
| pkg/scheduler/eventhandlers.go | addPodToSchedulingQueue |
| pkg/scheduler/eventhandlers.go | updatePodInSchedulingQueue |
| pkg/scheduler/eventhandlers.go | deletePodFromSchedulingQueue |
| pkg/scheduler/eventhandlers.go | addPodToCache |
| pkg/scheduler/eventhandlers.go | updatePodInCache |
| pkg/scheduler/eventhandlers.go | deletePodFromCache |
| pkg/scheduler/scheduler.go | scheduleOne |

**NOTE1:**

We need context or object or http request to get Trace-ID(This is specification of `Distributed Tracing` feature). So, we may exclude function which does not have such resource(context, object, http request) from our target.

**NOTE2:**

We use Tracing codes by `Distributed Tracing` KEP for propagation. So the following implementation is required.
 - Case1. The function which contains Tracing codes by `Distributed Tracing` KEP
   - We just add Request-ID codes (Request-ID codes don't interfere Tracing codes).
 - Case2. The function which does not contain Tracing codes by `Distributed Tracing` KEP
   - We add both of Tracing codes and Request-id codes (Request-ID codes don't interfere Tracing codes).

#### Design of Export Request-ID

We get Trace-ID by using `Distributed Tracing` feature in each Kubernetes function that we add Tracing codes. And then, we add Trace-ID information into **each klog call** as Request-ID. Note that we don't associate Request-ID to all of klog calls. Our target is important operations such as `kubectl create/delete/etc.`, and our target klog calls are the only klogs which is called via such important operations. Request-ID feature does not change existing klog function/method, but changes each klog calls and their log format. Currently, there is [Structured logging](https://github.com/serathius/enhancements/blob/structured-logging/keps/sig-instrumentation/20191115-structured-logging.md) KEP, and this KEP also change specific klog calls. We will merge Request-ID feature after Structured logging KEP is merged.

**Examples (We quote some parts from `Structured logging` KEP)**

Source Code

Original
```go
klog.Infof("Updated pod %s status to ready", pod.name)
```

Structured format
```go
klog.InfoS("Pod status updated", "pod", pod, "status", "ready")
```
Request-ID with structured format
```go
klog.InfoS("Pod status updated", "pod", pod, "status", "ready", "Request-ID", trace-id)
```

Expected Log

Structured format
```json
{
   "ts": 1580306777.04728,
   "v": 4,
   "msg": "Pod status updated",
   "pod":{
      "name": "nginx-1",
      "namespace": "default"
   },
   "status": "ready"
}

Request-ID with structured format
```json
{
   "ts": 1580306777.04728,
   "v": 4,
   "msg": "Pod status updated",
   "pod":{
      "name": "nginx-1",
      "namespace": "default"
   },
   "status": "ready"
   "request-id": 5acf2a4d258157e06402fb734186b684
}
```
#### Design overview of Control Request-ID

We should control Request-ID feature to avoid an impact to existing users who are retrieving logs and analyzing with existing log format. So we introduce `--request-id` parameter which enables/disables Request-ID feature. We also manage the range of operations which are added Request-ID. The effect of each parameter of `--request-id` is as follows.

| parameter | efficient |
| ------ | ------ |
| --request-id=0 | Request-ID feature is disabled (Default) |
| --request-id=1 | Request-ID feature is enabled, and Request-ID is added to klogs related to the `Alpha` target operations | 
| --request-id=2 | Request-ID feature is enabled, and Request-ID is added to klogs related to the `Alpha and Beta` target operations | 

Alpha and Beta target operations are described in [Migration / Graduation Criteria](#migration--graduation-criteria) section.

#### Detail design of Control Request-ID

TBD. I will write down the following things.
 - How to realize `--request-id` parameter in each Kubernetes component.
 - Sample codes which is used with `--request-id` option.

### Test Plan

 - test against the combination of following patterns.
   - --request-id(0/1/2)

### Migration / Graduation Criteria

#### Alpha

 - Add Request-ID against the following operations:
   - kubectl create/apply/delete
     - target resources: pod/deployment
   - kubectl drain
     - target resources: node
 - Implement `--request-id` parameter
 - E2e testing 
 - User-facing documentation

#### Beta

 - Add Request-ID against the following operations:
   - kubectl create/apply/delete
     - target resources: daemonset/pv/pvc/svc
   - kubectl scale/rollout
 - Update E2e testing
 - Update documentation

#### GA

 - All feedback is addressed.

