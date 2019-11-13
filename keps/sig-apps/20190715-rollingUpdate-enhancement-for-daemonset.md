---
title: RollingUpdate enhancement for Daemonset
authors:
  - "@resouer"
  - "@zhangxiaoyu-zidif"
owning-sig: sig-apps
participating-sigs:
  - sig-apps
reviewers:
  - "@janetkuo"
approvers:
  - TBD
editor: TBD
creation-date: 2019-07-15
last-updated: 2019-11-13
status: provisional
see-also:
  - n/a
replaces:
  - n/a
superseded-by:
  - n/a
---

# RollingUpdate enhancement for Daemonset

## Table of Contents

<!-- toc -->
- [Summary](#summary)
- [Motivation](#motivation)
  - [Goals](#goals)
  - [Non-Goals](#non-goals)
- [Proposal](#proposal)
  - [User Stories](#user-stories)
    - [Story 1](#story-1)
  - [Implementation Details](#implementation-details)
    - [API Changes](#api-changes)
    - [Implementation](#implementation)
  - [Risks and Mitigations](#risks-and-mitigations)
  - [Upgrades/Downgrades](#upgradesdowngrades)
  - [Tests](#tests)
- [Graduation Criteria](#graduation-criteria)
- [Implementation History](#implementation-history)
- [Drawbacks [optional]](#drawbacks-optional)
- [Alternatives](#alternatives)
<!-- /toc -->

## Summary

The purpose of this enhancement is to enhance RollingUpdate for Daemonset.
  - When Daemonset rolling update, it can be `Paused` by user.
  - Add `SurgingRollingUpdate` for Daemonset as a new update strategy.
  - Add two more fields, `partition` and `selector` for `RollingUpdate`
    - `partition` is the number of DaemonSet pods remained to be old version.
    - `selector` is to query over nodes whoes labels are matched by the Daemonset `RollingUpdate`.


## Motivation

Consider the following scenarios:-

1. When executing Daemonset update, even those Pods run well, user need to pause updation to check if  some results of new version image meet expect. If not, user can rollback images to stable version.

1. When rollingUpdate, user should have some chioces to select some specific or random nodes to update pods running on it.

### Goals

- Add field `Paused` for `DaemonSetSpec`.
- Add new fields `Selector` and `Partition` for existing RollingUpdateDaemonSet.

### Non-Goals
- This implement is only implemeted to affect to update strategy of Daemonset. No other behaviors of Daemonset will be changed.

## Proposal

### User Stories

#### Story 1

There are serveral defects of current community design. Such as:
* Pause and Resume. DaemonSet does not has the fetature of Pause/Resume in update。 Especially in large or huge scale cluster, in order to make the cluster high avaiable and certainly correct image version, Kubernetes should server the ability that user can pause DaemonSet update process and check if target version image meet their expect. If users or administrators' test verified new version pass probe test, they can resume DaemonSet update.

* Batch update. As for a mature system, gray update is a necessary ability. However there is no any enhancement on this area. Now the community version do not supports any partion process in update like other workload, such StatefulSet.

### Implementation Details

#### API Changes

Following changes will be made to the Rolling Update Strategy for StatefulSet.

```go
// DaemonSetSpec is the specification of a daemon set.
type DaemonSetSpec struct {
  ...
    // Indicates that the daemonset is paused and will not be processed by the
  // daemonset controller.
  // +optional
  Paused bool
  ...
}
```

```go
type RollingUpdateDaemonSet struct {
	// The maximum number of DaemonSet pods that can be unavailable during the
	// update. Value can be an absolute number (ex: 5) or a percentage of total
	// number of DaemonSet pods at the start of the update (ex: 10%). Absolute
	// number is calculated from percentage by rounding up.
	// This cannot be 0.
	// Default value is 1.
	// Example: when this is set to 30%, at most 30% of the total number of nodes
	// that should be running the daemon pod (i.e. status.desiredNumberScheduled)
	// can have their pods stopped for an update at any given
	// time. The update starts by stopping at most 30% of those DaemonSet pods
	// and then brings up new DaemonSet pods in their place. Once the new pods
	// are available, it then proceeds onto other DaemonSet pods, thus ensuring
	// that at least 70% of original number of DaemonSet pods are available at
	// all times during the update.
  // +optional
  MaxUnavailable *intstr.IntOrString `json:"maxUnavailable,omitempty" protobuf:"bytes,1,opt,name=maxUnavailable"`

	// A label query over nodes that are managed by the daemon set RollingUpdate.
	// Must match in order to be controlled.
	// It must match the node's labels.
	Selector *metav1.LabelSelector `json:"selector" protobuf:"bytes,2,opt,name=selector"`

	// The number of DaemonSet pods remained to be old version.
	// Default value is 0.
	// Maximum value is status.DesiredNumberScheduled, which means no pod will be updated.
	// +optional
	Partition *int32 `json:"partition,omitempty" protobuf:"varint,3,opt,name=partition"`
}
```

#### Implementation
* Add subcommand Pause/Resume for kubectl rollout

```shell
kubectl rollout pause ds <ds name>
```

When users want to update a DaemonSet whose replicas are deployed in a huge cluster, a Pause is a must feature. DaemonSet always serves some basic infrastructures, and the target version image may go wrong, even pass all kinds of tests. At this occasions, the operators should stop updating at once.

Some disadvantages of pause: When DaemonSet's fields paused is set to be true, DaemonSet will not execute any processes. All behaviors, such as extending replicas on new imported nodes, are all stopped. But it will not affect too much because the whole period of pause status is rather short to the whole lifecycle of DaemonSet.

```shell
kubectl rollout resume ds <ds name>
```

After users or operators verified new DaemonSet replicas work well, they could coninue the update process.

* Add new fields for `RollingUpdateDaemonSet`
  * `selector`
    * This field will match nodes' labels, and select label matched nodes to update DaemonSet replicas.
  * `partition`
    * Only if selector is nil, the feild partition works.
    * When partition is specified, the partition value is the count of DaemonSet replcas remaining old version. User should continue to update partition in descreaing order gradually until it turns to be zero.


### Risks and Mitigations

### Upgrades/Downgrades

### Tests

## Graduation Criteria

## Implementation History

- KEP Started on 7/15/2019
- KEP Modified on 11/13/2019
- Implementation PR and UT by TBD


## Drawbacks [optional]

Why should this KEP _not_ be implemented.

## Alternatives


