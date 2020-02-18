---
title: Coscheduling plugin based on scheduler framework
authors:
  - "@denkensk"
owning-sig: sig-scheduling
reviewers:
  - "@Huang-Wei"
  - "@ahg-g"
  - "@alculquicondor"
  - "k82cn"
  - "@resouer"
  - "@hex108"
  - "@everpeace"
approvers:
  - "@Huang-Wei"
creation-date: 2020-01-16
last-updated: 2020-01-16
status: provisional
---

# Coscheduling plugin based on scheduler framework

## Table of Contents

<!-- toc -->
- [Motivation](#motivation)
- [Goals](#goals)
- [Non-Goals](#non-goals)
- [Use Cases](#use-cases)
- [Terms](#terms)
- [Proposal](#proposal)
- [Design Details](#design-details)
  - [PodGroup](#podgroup)
  - [Coscheduling](#coscheduling)
  - [Extension points](#extension-points)
    - [QueueSort](#queuesort)
    - [Pre-Filter](#pre-filter)
    - [Permit](#permit)
    - [UnReserve](#unreserve)
- [Alternatives considered](#alternatives-considered)
- [Graduation Criteria](#graduation-criteria)
- [Testing Plan](#testing-plan)
- [Implementation History](#implementation-history)
- [References](#references)
<!-- /toc -->

## Motivation
Kubernetes has become a popular solution for orchestrating containerized workloads. Due to limitation of Kubernetes scheduler, some offline workloads (ML/DL) are managed in a different system. To improve cluster utilization and operation efficiency, we'd like to treat Kubneretes as a unified management platform. But ML jobs are All-or-Nothing: they require all tasks of a job to be scheduled at the same time. If the job only start part of tasks, it will wait for other tasks to be ready to begin to work. In the worst case, all jobs are pending leading to a deadlock. To solve this problem, co-scheduling is needed for the scheduler. The new scheduler framework makes the goal possible.
 
## Goals
Use scheduler plugin, which is the most Kubernetes native way, to implement coscheduling.
 
## Non-Goals
Discuss the API definition of `PodGroup`.
 
## Use Cases
when running a Tensorflow/MPI job, all tasks of a job must be start together; otherwise, did not start anyone of tasks. If the resource is enough to run all 'tasks', everything is fine; but it's not true for most of case, especially in the on-premises environment. In worst case, all jobs are pending here because of deadlock: every job only start part of tasks, and waits for the other tasks to start. In worst case, all jobs are pending leading to a deadlock.
 
## Terms

- **pgPod**: pod belongs to some `PodGroup`.
- **regularPod**: a regular `Pod` (which doesn't have `PodGropuName` set).

## Proposal

In order to implement coscheduling, we developed plugins in different extension points. In `QueueSort`  we ensure that the Pods belonging to the same PodGroup are queued back-to-back. For example, suppose PodGroup A owns Pod-A1, Pod-A2, Pod-A3, while PodGroup B owns Pod-B1, Pod-B2. The pods of the two PodGroups should not interleave - it should be always <PodGroup-A, PodGroup-B> or the other way around; but never <Pod-A1, Pod-B1, Pod-A2, ...>. In `Permit` phase we put the pod that doesn't meet min-available into the WaitingMap and reserve resources until min-available are met or timeout is triggered. In `Unreserve` phase，clean up the pods that timed-out.

![image](./20200116-coscheduling-plugin-based-on-scheduler-framework-extensions.png)


## Design Details

### PodGroup

We use a special label named ```pod-group.scheduling.sigs.k8s.io/name``` to define a PodGroup. Pods that set this label and use the same value belong to the same PodGroup. This is a short term solution, in the future if the definition of `PodGroup` is accepted by the community, we will define it directly through the CRD of `PodGroup`. This is not the focus of this proposal. 

```yaml 
labels:
     pod-group.scheduling.sigs.k8s.io/name: nginx
     pod-group.scheduling.sigs.k8s.io/min-available: "2"
``` 
`Pods` in the same `PodGroup` with different priorities might lead to unintended behavior, so need to ensure `Pods` in the same `PodGroup` with the same priority.

### Coscheduling
```go
// Coscheduling is a plugin that implements the mechanism of gang scheduling.
type Coscheduling struct {
    FrameworkHandle     	framework.FrameworkHandle
    PodLister               corelisters.PodLister
    // Key is the name of PodGroup.
    PodGroupInfos       	map[string]PodGroupInfo
    // Name of the last scheduled podgroup
    LastPodGroup        	string
}
 
type PodGroupInfo struct {
    // LastFailureTimestamp stores the timestamp of last scheduling failure.
    LastFailureTimestamp 	time.Time
    UID           			types.UID
    MinAvailable  		int
    Name                                 String
}
```

1.  `PodGroupInfo` is initialized the first time the pod belongs to the PodGroup is encountered, and LastFailureTimestamp is updated every time the PodGroup fails to schedule.
2.  `LastPodGroup` records which `PodGroup` the last scheduled pod belongs to.
3.  `UID` is the unique identification value used to distinguish different podgroups.
 

### Extension points

#### QueueSort
In order to make the pods which belongs to the same `PodGroup` to be scheduled together as much as possible, implement a strategy in `QueueSort` phase. 

```go
  func  Less(podA *PodInfo, podB *PodInfo) bool
```
1. Trying to order by pod priority (i.e. .spec.priorityValue), pod with higher priority is scheduled ahead of other pod with lower priority. When priorities are the same, they are operated according to the following process. 
   
2. When podA and podB are both regularPods (we will check it by their labels), it follows the same logic of default in-tree [PrioritySort](https://github.com/kubernetes/kubernetes/blob/master/pkg/scheduler/framework/plugins/queuesort/priority_sort.go#L41-L45) plugin.
   
3. When podA is regularPod, podB is pgPod, trying to order by podA's `Timestamp` (the time pod added to the scheduling queue) and podB’s `LastFailureTimestamp` (we get the PodGroupInfo from the cache). Pod with earlier timestamp is scheduled ahead of other pod.   
   
4. When podA and podB are both pgPods. 
   1. Trying to order by the `LastFailureTimestamp` of the podGroup. Pod with earlier timestamp is scheduled ahead of other pod.
   
   2. If `LastFailureTimestampA` is equal to `LastFailureTimestampB`, trying to order by the `UID` of `PodGroup`,`Pod` with lexicographically greater `UID` is scheduled ahead of other pod. (The purpose is to distinguish different `PodGroup` with the same `LastFailureTimestamp` and to keep the pods of the same `PodGroup` together)

**Note1**: There are different `LastFailureTimestamp` (even if they are the same, the UID will be different). So when the pods enter the queue, the pods that belongs to the same PodGroup will be together.

#### Pre-Filter
1. `PreFilter` validates that if the total number of pods belonging to the same `PodGroup` is less than `minAvailable`. If so, the scheduling process will be interrupted directly.

2. `PreFilter` validates that if we should reject `WaitingPods` in advance. When the pod belongs to a new `PodGroup` (we can check it by the `LastPodGroup` in cache), it indicates that the new PodGroup scheduling cycle has been entered. Then we will check if the last `PodGroup` meets the minAvailable, if not, we will reject it directly in advance. The purpose is 1.To avoid invalid waiting, 2.To make the pod belongs to the same `PodGroup` fail together, rather than waiting partially.

#### Permit
In `Permit` phase, we put the pod that doesn't meet min-available into the WaitingMap and reserve resources until min-available are met or timeout is triggered.
1. Get the number of Running pods that belong to the same PodGroup
2. Get the number of WaitingPods (used to record pods in waiting status) that belong to the same PodGroup
3. If Running + WaitingPods + 1 >= min-available(1 means the pod itself), approve the waiting pods that  belong to the same PodGroup. Otherwise, put the pod into WaitingPods and set the timeout (eg: the timeout is dynamic value depends on the size of the `PodGroup`.)

#### UnReserve
After a pod which belongs to a PodGroup times out in the permit phase.  UnReserve ```Rejects``` the pods that belong to the same PodGroup to avoid long-term invalid reservation of resources.

## Alternatives considered
1. Using `PodGroup` as a scheduling unit. This requires major refactoring, which only supports Pods as scheduling unit today.


## Graduation Criteria

## Testing Plan
1.  Add detailed unit and integration tests for workloads.
2.  Add basic e2e tests, to ensure all components are working together.
 
## Implementation History
## References
- [Coscheduling in Kubernetes](https://docs.google.com/document/d/1AUwcvTtULNvow5M9e428FnlvINO1uQ7ojRoTGuTp4DA/edit#heading=h.ckn8nv2jj0xv)
- [Schedule a group of pods all at once](https://github.com/kubernetes/kubernetes/issues/16845)
- [kubeflow/tf-operator: Prevent scheduling deadlocks](https://github.com/kubeflow/tf-operator/issues/165)
- [Added PodGroup Phase in Status](https://github.com/kubernetes-sigs/kube-batch/pull/533)
