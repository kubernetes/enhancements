---
title: Pid Limiting
authors:
  - "@derekwaynecarr"
  - "@dims"
owning-sig: sig-node
participating-sigs:
reviewers:
  - "@dashpole"
approvers:
  - "@dashpole"
  - "@dchen1107"
editor: Derek Carr
creation-date: 2019-01-29
last-updated: 2019-03-05
status: implemented
see-also:
replaces:
superseded-by:
---

# Pid Limiting

## Table of Contents

<!-- toc -->
- [Summary](#summary)
- [Motivation](#motivation)
  - [Goals](#goals)
  - [Non-Goals](#non-goals)
- [Proposal](#proposal)
  - [User Stories [optional]](#user-stories-optional)
  - [Implementation Details/Notes/Constraints [optional]](#implementation-detailsnotesconstraints-optional)
    - [Pod to Pod Isolation](#pod-to-pod-isolation)
    - [Node to Pod Isolation](#node-to-pod-isolation)
    - [Cgroup Enforcement](#cgroup-enforcement)
  - [Risks and Mitigations](#risks-and-mitigations)
- [Graduation Criteria](#graduation-criteria)
  - [Pod to Pod pid isolation](#pod-to-pod-pid-isolation)
  - [Node to Pod pid isolation](#node-to-pod-pid-isolation)
- [Implementation History](#implementation-history)
  - [Version 1.10](#version-110)
  - [Version 1.14](#version-114)
  - [Version 1.15](#version-115)
<!-- /toc -->

## Summary

A proposal to enable isolation of pid resources.  It proposes a mechanism to
enable pod-to-pod PID isolation as well as node-to-pod PID isolation.

## Motivation

Pids are a fundamental resource on Linux hosts.  It is trivial to hit the task
limit without hitting any other resource limits and cause instability to a host
machine.

Administrators require mechanisms to ensure that user pods cannot induce pid
exhaustion that prevents host daemons (runtime, kubelet, etc) from running.  In
addition, it is important to ensure that pids are limited among pods in order to
ensure they have limited impact to other workloads on the node.

### Goals

This proposal aims to the following:
- enable administrator control to provide pod-to-pod pid isolation
- enable administrator control to provide node-to-pod pid isolation

### Non-Goals

This proposal defers the following:
- ability for a user to request additional number of pid resources per pod

It is anticipated we will support that via a policy knob that could be
restricted and/or defaulted via PodSecurityPolicy or LimitRange.  We anticipate
tracking this work under a separate feature gate `GranularPidLimitsPerPod`.  Any
defaulting applied to pods today would only be used if the pod had no local pod
pid limiting policy in future dates.

## Proposal

### User Stories [optional]

1. Administrator can default the number of pids per pod to provide pod-to-pod
   isolation.
1. Administrator can reserve a number of allocatable pids to user pods via node
   allocatable.

### Implementation Details/Notes/Constraints [optional]

#### Pod to Pod Isolation

To enable pid isolation among pods, the `SupportPodPidsLimit` feature gate is
defined.

If enabled, the kubelet argument for `pod-max-pids` will write out the
configured pid limit to the pod level cgroup to the value specified on Linux
hosts.  If -1, the kubelet will default to the node allocatable pid capacity.

#### Node to Pod Isolation

To enable pid isolation from node to pods, the `SupportNodePidsLimit` feature
gate is proposed.  If enabled, pid reservations may be supported at the node
allocatable and eviction manager subsystem configurations.

Node allocatable is a well-established feature concept in the kubelet that
allows isolation of user pod resources from host daemons at the `kubepods`
cgroup level that parents all end-user pods.

The kubelet will be updated to support reservation of pids so the effective pid
limit is enabled as follows:

```
[Allocatable] = [Node Capacity] - 
 [Kube-Reserved] - 
 [System-Reserved] - 
 [Hard-Eviction-Threshold]
```

#### Cgroup Enforcement

To use this feature, the `--cgroups-per-qos` must be enabled.  In addition, the
`pids` cgroup must be mounted.

The `kubepods` cgroup is bounded by the `Allocatable` value.

The QoS level cgroups are left unbounded across all pid pool sizes.

The pod level cgroup sandbox is configured as follows:

1. the pod-max-pids value if positive and is specified on kubelet config
1. the local pod pid limiting policy (future)
1. unbounded (so it is restricted by the `Allocatable` value at `kubepods`)

### Risks and Mitigations

None

## Graduation Criteria

### Pod to Pod pid isolation

The following criteria applies to `SupportPodPidsLimit` feature gate:

Alpha
- basic support integrated in kubelet

Beta
- ensure proper node e2e test coverage is integrated verifying cgroup settings
- see testing:
https://github.com/kubernetes/kubernetes/blob/master/test/e2e_node/pids_test.go
https://k8s-testgrid.appspot.com/sig-node-kubelet#node-kubelet-serial&include-filter-by-regex=Feature%3ASupportPodPidsLimit

GA
- assuming no negative user feedback based on production experience, promote
  after 2 releases in beta.

### Node to Pod pid isolation

Adding support for pid limiting at the Node Allocatable level 

The following criteria applies to `SupportNodePidsLimit`:

Alpha
- basic support integrated via eviction manager and/or node allocatable level

Beta
- ensure proper node e2e testing coverage to ensure a pod is unable to fork-bomb
  a node even when `pod-max-pids` is unbounded.
- see testing:
https://github.com/kubernetes/kubernetes/pull/73651/files#diff-7681b587a8fd514b312fa29c3acc669e


GA
- assuming no negative user feedback, promote after 1 release at beta.

## Implementation History

### Version 1.10

`SupportPodPidsLimit` implemented at Alpha.

### Version 1.14

- Implement `SupportNodePidsLimit` as Alpha.
- Graduate `SupportPodPidsLimit` to Beta by adding node e2e test coverage for
  pid cgroup isolation, ensure PidPressure works as intended.
  
### Version 1.15

- Graduate `SupportNodePidsLimit` to beta by adding node e2e test
  coverage for node cgroup isoation.
