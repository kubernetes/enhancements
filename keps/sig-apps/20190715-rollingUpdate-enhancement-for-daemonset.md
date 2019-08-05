---
title: RollingUpdate enhancement for Daemonset
authors:
  - "@resouer"
  - "@zhangxiaoyu-zidif"
  - "@answer1991"
owning-sig: sig-apps
participating-sigs:
  - sig-apps
reviewers:
  - "@janetkuo"
approvers:
  - TBD
editor: TBD
creation-date: 2019-07-15
last-updated: 2019-08-05
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
  - Add two more fields, `partition` and `selector` for `RollingUpdate` and `SurgingRollingUpdate`.
    - `partition` is the number of DaemonSet pods remained to be old version.
    - `selector` is to query over nodes whoes labels are matched by the Daemonset `RollingUpdate` or `SurgingRollingUpdate`.


## Motivation

Consider the following scenarios:-

1. When executing Daemonset update, even those Pods run well, user need to pause updation to check if  some results of new version image meet expect. If not, user can rollback images to stable version.

1. In some case, user do not want to stop some services when rolling update, so SurgingRollingUpdate is a must choice.

1. When rollingUpdate or surgeRollingUpdate, user should have some chioce to select some specific or random nodes to update pods running on it.

### Goals

- Add field `Paused` for `DaemonSetSpec`.
- Add a new  `DaemonSetUpdateStrategy`, i.e.: `SurgingRollingUpdate`. `SurgingRollingUpdate` has three fileds, i.e.: `MaxSurge`,`Selector`, and `Partition`.
- Add new fields `Selector` and `Partition` for existing RollingUpdateDaemonSet.

### Non-Goals
- This implement is only implemeted to affect to update strategy of Daemonset. No other behaviors of Daemonset will be changed.

## Proposal

### User Stories

#### Story 1
In some end users' clusters, they deploy DNS resolver and other containers serving infrastructure by daemonset in every nodes. To an online e-commercial services which serve 200K+ QPS, it must cause cascading disaster to stop those infrastructure services even in a very short time by current Daemonset RollingUpdate strategy. So surgingRollingUpdate is a necessary strategy for our production environment. 
As a solution for above case, if end users are not sure whether SurgeingRollingUpdate are suitable for their production environment or cause other resource conflicts, e.g. hostport, they can use 'Selector' or 'Partition' to do some experimental update.

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
// DaemonSetUpdateStrategy is a struct used to control the update strategy for a DaemonSet.
type DaemonSetUpdateStrategy struct {
  ...
	// Surging rolling update config params. Present only if type = "SurgingRollingUpdate".
	SurgingRollingUpdate *SurgingRollingUpdateDaemonSet `json:"surgingRollingUpdate,omitempty" protobuf:"bytes,3,opt,name=surgingRollingUpdate"`
}
```

```go
// Spec to control the desired behavior of a daemon set surging rolling update.
type SurgingRollingUpdateDaemonSet struct {
	// The maximum number of DaemonSet pods that can be scheduled above the desired number of pods
	// during the update. Value can be an absolute number (ex: 5) or a percentage of the total number
	// of DaemonSet pods at the start of the update (ex: 10%). The absolute number is calculated from
	// the percentage by rounding up. This cannot be 0. The default value is 1. Example: when this is
	// set to 30%, at most 30% of the total number of nodes that should be running the daemon pod
	// (i.e. status.desiredNumberScheduled) can have 2 pods running at any given time. The update
	// starts by starting replacements for at most 30% of those DaemonSet pods. Once the new pods are
	// available it then stops the existing pods before proceeding onto other DaemonSet pods, thus
	// ensuring that at most 130% of the desired final number of DaemonSet  pods are running at all
	// times during the update.
	// +optional
	MaxSurge *intstr.IntOrString `json:"maxSurge,omitempty" protobuf:"bytes,1,opt,name=maxSurge"`

	// A label query over pods that are managed by the daemon set SurgingRollingUpdate.
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

```go
type RollingUpdateDaemonSet struct {
  ...
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


### Risks and Mitigations

### Upgrades/Downgrades

### Tests

## Graduation Criteria

## Implementation History

- KEP Started on 7/15/2019
- Implementation PR and UT by TBD

## Drawbacks [optional]

Why should this KEP _not_ be implemented.

## Alternatives


