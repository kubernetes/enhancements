---
title: Pod Evict Policy
authors:
  - "@answer1991"
owning-sig: sig-node
participating-sigs:
  - sig-api-machinery
  - sig-apps
reviewers:
approvers:
creation-date: 2019-04-07
last-updated: 2019-12-06
status: provisional

---

# Pod Evict Policy

## Table of Contents

- [Table of Contents](#table-of-contents)
- [Summary](#summary)
- [Motivation](#motivation)
  - [Goals](#goals)
  - [Non-Goals](#non-goals)
- [Proposal](#proposal)
  - [User Stories](#user-stories)
    - [Story 1](#story-1)
    - [Story 2](#story-2)
  - [Risks and Mitigations](#risks-and-mitigations)
- [Design Details](#design-details)
  - [API](#api)
    - [EvictReason Definition](#evictreason-definition)
    - [EvictRequest Definition](#evictrequest-definition)
    - [EvictPolicyType Definition](#evictpolicytype-definition)
    - [EvictPolicy Definition](#evictpolicy-definition)
  - [Workflow](#workflow)
- [Implementation History](#implementation-history)

## Summary

Pod Evict Policy define the evicting policy to a Pod, Kubernetes will evict Pod by its' value specified.
Thus workload controller may participate in or take control the Pod evicting progress.

## Motivation

For now, Kubernetes evict Pods running on the unhealthy Nodes by deleting them directly. 
We think this behavior is not a good practice which may influence the health of Service which the evicted Pods serve for.
For example: a Service has 3 Pods backend, if these 3 Pods was evicted at same time, then the Service is not healthy during eviction

If a Pod belong to a workload, then the workload controller should take control the whole lifecycle of the Pod, including eviction. 

### Goals

- Define a `EvictPolicy` property to `Pod`, the value means which policy should be used when Kubernetes want to evict this Pod.
- Workload controller could participate in or take control the Pod evicting progress.

### Non-Goals

- When and in which conditions to evict Pods is NOT a goal.

## Proposal

User or workload controller can specified the value of `EvictPolicy` to the Pod. The value option could be `Default` or `ControllerTakeOver`.

- If the `EvictPolicy` value is `Default` or nil, when Kubernetes want to evict Pod, just delete them directly as which we do before.
- If the `EvictPolicy` value is `ControllerTakeOver`, when Kubernetes want to evict Pod, an annotation(or label, or some status filed) will be added to indicate that the Pod should be evicted. 
Then workload controller start to delete these Pod according the workload evict strategy.

### User Stories

#### ReplicaSet Controller Evict Pods according ReplicaSet EvictStrategy

When Pod Evict Policy implemented, ReplicaSet could add `EvictStrategy` field which user could specified the max number of the `rs` in eviction progress like which `updateStrage` did.
 
For example:
```yaml
apiVersion: apps/v1
kind: ReplicaSet
metadata:
  name: foo
spec:
  replica: 3
  evictStrategy:
    maxUnavailable: 1
  ...
```

The ReplicaSet Controller would create 3 Pods by specified the Pod's `EvictPolicy` to be `ControllerTakeOver`.

When the Nodes which these Pods are unhealthy, Kubernetes annotate the Pod state to be eviction wanted. For example:

```yaml
apiVersion: apps/v1
kind: ReplicaSet
metadata:
  name: foo-6b7cb787d
  annotations:
    k8s.io/eviction-wanted: "true"
  ...
```

Then the ReplicaSet Controller will start to evict these Pods by the `rs` *foo* specified `EvictStrategy`. In this example, only one of these `rs` Pod will be in eviction progress.
Thus we can keep the Service which these Pods serve for healthy as much as possible.

### Risks and Mitigations

As the default value of `EvictPolicy` is `Default`, and the `EvictPolicy` value of Pod which created by earlier versions is `nil`, current eviction action will be invoked as before.
Thus risks of compatibility problems is very low, and there is no mitigations issue.

We should consider the risks if workload controller stop working, a timeout should be added if `EvictPolicy` is `ControllerTakeOver`. 
If the workload controller does not finish eviction in a given timeout, Pod eviction polify will fallback to `Default`, which means Kubernetes will do eviction if `ControllerTakeOver` eviction timeout.

## Design Details

### API Changes

#### Adding EvictPolicy Definition in PodSpec

```go
// EvictPolicy describes how the Pod should be evicted.
// Only one of the following evict policies may be specified.
// If none of the following policies is specified, the default one
// is Default.
type EvictPolicy string

const (
    // EvictPolicyDefault indicates Kubernetes take control the whole eviction progress.
    // Which means Kubernetes delete Pod directly when the Node is not healthy which the Pod running on.
	EvictPolicyDefault    EvictPolicy = "Default"

    // EvictPolicyControllerTakeOver indicates the workload controller which Pod belong to will
    // take over the eviction progress.
    // If controller evict timeout, Kubernetes will fallback to policy EvictPolicyDefault.
	EvictPolicyControllerTakeOver EvictPolicy = "ControllerTakeOver"
)

// PodSpec is a description of a pod.
type PodSpec struct {
	...
    
    // Set evict policy for the pod.
    // Defaults to "EvictPolicyDefault".
    // Valid values are 'Default' or 'ControllerTakeOver'.
    // +optional
    EvictPolicy *EvictPolicy
}
```

#### Adding Pod Eviction Wanted Annotation

```go
const (
    // PodEvictionWantedKey is the key of the annotation for Pod if the Pod is eviction wanted.
	PodEvictionWantedKey = "kubernetes.io/pod-evict-wanted"
)
```

### Kubernetes Evict Pod

When Kubernetes decide to evict a Pod, firstly it check the Pod `EvictPolicy`. 

* If the value is `EvictPolicyDefault` or nil, delete Pod directly as current action.
* If the value is `EvictPolicyControllerTakeOver` and controller evict not timeout, add annotation `kubernetes.io/pod-evict-wanted=true` to the Pod.
* If the value is `EvictPolicyControllerTakeOver` and controller evict timeout, fallback to `EvictPolicyDefault` policy, delete Pod directly as current action.

Controller evict timeout could be a Kubernetes start flag, default value could be `1*Minute`.

## Implementation History

- 2019-04-07: Initial KEP sent out for review.
- 2019-12-06: Improve KEP.