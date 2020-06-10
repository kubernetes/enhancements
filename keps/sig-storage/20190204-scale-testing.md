---
title: Volume Scale and Performance Testing Plan
authors:
  - "@msau42"
owning-sig: sig-storage
participating-sigs:
  - sig-scalability
reviewers:
  - "@pohly"
  - "@saad-ali"
  - "@wojtek-t"
approvers:
  - "@saad-ali"
  - "@wojtek-t"
editor: TBD
creation-date: 2019-02-04
last-updated: 2019-03-01
status: implementable
see-also:
  - "https://github.com/kubernetes/community/blob/master/sig-scalability/slos/pod_startup_latency.md"
  - "https://github.com/kubernetes/perf-tests/blob/master/clusterloader2/docs/design.md"
replaces:
superseded-by:
---

# Volume Scale and Performance Testing Plan

## Table of Contents

<!-- toc -->
- [Summary](#summary)
- [Motivation](#motivation)
  - [Goals](#goals)
  - [Non-Goals](#non-goals)
- [Proposal](#proposal)
  - [Test Framework](#test-framework)
  - [Test Rollout](#test-rollout)
  - [Test Portability](#test-portability)
  - [Test Cases](#test-cases)
    - [Pod Startup](#pod-startup)
  - [WIP Future Test Cases](#wip-future-test-cases)
    - [Pod Teardown](#pod-teardown)
    - [PV Binding Tests](#pv-binding-tests)
    - [PV Provisioning Tests](#pv-provisioning-tests)
    - [PV Deletion Tests](#pv-deletion-tests)
- [Graduation Criteria](#graduation-criteria)
  - [Phase 1](#phase-1)
  - [Phase 2](#phase-2)
  - [Phase 3](#phase-3)
- [Implementation History](#implementation-history)
<!-- /toc -->

## Summary

This KEP outlines a plan for testing scalability and performance of K8s storage components.

## Motivation

Adding storage scale and performance tests will help:
* Understand the current scale limits of the Kuberentes storage system.
* Set expectations (SLOs) for consumers of the Volume API.
* Determine bottlenecks and influence which need addressing.

### Goals

* Measure the overhead of K8s components owned by sig-storage:
  * K8s volume controllers
  * Kubelet volume manager
  * CSI sidecars
* Stress various dimensions of volume operations to determine:
  * Max volumes per pod
  * Max volumes per node
  * Max volumes per cluster
* Test with the following volume types:
  * EmptyDir
  * Secret
  * Configmap
  * Downward API
  * CSI mock driver
  * TODO: Hostpath?
  * TODO: Local?
* Provide tests that vendors can easily run against their volume drivers.

### Non-Goals

* Test and measure storage provider’s drivers.

## Proposal

### Test Framework

The tests should be developed and run in sig-scalability’s test
[Cluster Loader framework](https://github.com/kubernetes/perf-tests/blob/master/clusterloader2/README.md)
and infrastructure.

The framework already supports measurements for Pod startup latency and can be
used to determine a [Pod startup SLO with
volumes](https://github.com/kubernetes/community/pull/3242). Any changes
necessary to this measurement will be handled by sig-scalability.

Additional measurements can be added to the framework to get informational details for
for volume operations. These are not strictly required but are nice to have:

* Gather existing [volume operation
  metrics](https://github.com/kubernetes/kubernetes/blob/master/pkg/volume/util/metrics.go)
  from kube-controller-manager and kubelet.
  * Caveat: These metrics measure the time of a single operation attempt and not
    the e2e time across retries. These metrics may not be suitable to measure
    latency from a pod's perspective.
* Enhance the scheduler metrics collection to include
  [volume scheduling
  metrics](https://github.com/kubernetes/kubernetes/blob/master/pkg/controller/volume/scheduling/scheduler_binder_cache.go)
* Timing of volume events. However, events should not be considered a reliable
  measurement because they can be dropped or garbage collected. Also there are
  only events for provision/delete, attach/detach. Mount events were removed due
  to the high frequency of remount operations for API volume types (secrets,
  etc). These are still useful to collect for debugging.

Enhancements to Kubernetes itself can also be considered not just for scale
testing but for general improvements to supportability and debuggability of the
system. These are not strictly required and can be considered as a future
enhancement:

* Add e2e metrics for volume operations that can track the time across operation
  retries.
* Add timestamps to the appropriate API object status whenever an operation is
  completed.


### Test Rollout

Before adding these tests to regular CI jobs, a performance baseline needs to be
established to form the basis of a SLO. To accomplish this:

1. Test parameters such as number of pods and number of volumes per pod should be
  configurable.
1. Start with the documented [max limits](https://kubernetes.io/docs/setup/cluster-large/)
  and [stateless pod
  SLO](https://github.com/kubernetes/community/blob/master/sig-scalability/slos/pod_startup_latency.md)
  as a target SLO.
1. Run the new scale tests with the same test parameters as the stateless pod density
  tests.
1. Adjust test parameters and/or target SLO as needed.
1. Repeat until a stable target SLO can be established consistently across runs.

### Test Portability

In order to make it easier for vendors to run these tests against their volume
drivers, the tests for persistent volumes should only use the default
StorageClass configured for the cluster.

Therefore these tests have the following prerequisites:

* A Kubernetes cluster is brought up in an appropriate environment that can
  support the storage provider.
* The storage backend and driver is already installed and configured.
* A default StorageClass for that driver been installed in the cluster.

### Test Cases

#### Pod Startup

These tests should measure how long it takes to start up a pod with unique volumes,
assuming that the volumes have already been provisioned and volumes are not
shared between pods.

The requirement for non-shared pods is so that we can produce an initial
consistent baseline measurement. The performance of pod startup latency when using
shared volumes has a lot of variable factors such as volume type (RWO vs RWX),
and whether or not the scheduler decided to schedule a replacement pod on the
same or different node. We can consider adding more test cases in the future
that can handle this scenario once we establish an initial baseline.

For the initial baseline, the test case can be run for each volume type by changing
various dimensions (X):

* Create 1 pod with X volumes and 1 node. This can help determine a limit for
  max number of volumes in a single Pod.
* Create X pods with 1 volume each on 1 node in parallel. This can help determine a limit
  for max number of volumes on a single node.
* Create X pods with 1 volume each in parallel.

All the test cases should measure pod startup time. A breakdown of time spent in volume scheduling,
attach and mount operations is not strictly required but nice to have.


### WIP Future Test Cases

These test cases are still under development and need to be refined before they
can be implemented in the future.

#### Pod Teardown

These tests should measure how long it takes to delete a pod with volumes.

For each volume type:

* Delete many pods with 1 volume each in parallel.
* Delete 1 pod with many volumes.
  * Measure pod deletion time, with a breakdown of time spent in unmount and detach operations.
    * Note: Detach can only be measured for CSI volumes by the removal of the VolumeAttachment object.

#### PV Binding Tests

These tests should measure the time it takes to bind a PVC to a preprovisioned available PV.

#### PV Provisioning Tests

These tests should measure the time it takes to bind a PVC to a dynamically provisioned PV.

For each volume type that supports provisioning:

* Create many PVCs with immediate binding in parallel.
* Create many PVCs with delayed binding in parallel.
  * Measure volume provisioning time.

#### PV Deletion Tests

These tests should measure the time it takes to delete a PVC and PV.

For each volume type that supports deletion:

* Delete many PVCs with the Delete reclaim policy in parallel.
  * Measure volume deletion time.


## Graduation Criteria

### Phase 1

* Pod startup tests running in scale clusters for all the targeted volume types.
* Pod startup latency and max limits results published.

### Phase 2

* Improve pod startup latency measurements to get finer-grained volume operation
  latencies.
* Establish SLO for pod startup latency with volumes.
* Tests fail if results exceeds SLO.

### Phase 3

* Add additional tests for Pod deletion and volume binding/provisioning.

## Implementation History


