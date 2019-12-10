---
title: Enhance ContainerStats message in CRI-API
authors:
  - "@fuweid"
owning-sig: sig-node
participating-sigs:
  - sig-node
  - sig-instrumentation
reviewers:
  - TBD
approvers:
  - TBD
editor: TBD
creation-date: 2019-11-27
last-updated: 2019-12-10
status: provisional
---

# Enhance ContainerStats message in CRI-API

## Table of Contents

<!-- toc -->
- [Summary](#summary)
- [Motivation](#motivation)
  - [Goals](#goals)
- [Proposal](#proposal)
  - [User Stories](#user-stories)
  - [Implementation Details/Notes/Constraints](#implementation-detailsnotesconstraints)
- [Design Details](#design-details)
  - [Test Plan](#test-plan)
  - [Graduation Criteria](#graduation-criteria)
  - [Upgrade / Downgrade Strategy](#upgrade--downgrade-strategy)
  - [Version Skew Strategy](#version-skew-strategy)
- [Implementation History](#implementation-history)
<!-- /toc -->

## Summary

This enhancement is to extend `CpuUsage` and `MemoryUsage` fields in
`ContainerStats`. And also add new `ProcessUsage`/`NetworkUsage` field in
`ContainerStats`. The Container Runtime Engine will cover more metrics about
managed container and let the metric collectors focus on node-level metrics or
monitoring metrics.

## Motivation

Since more and more container runtimes shows up, like `kata-container`/
`gVisor`/ `Firecracker`/ `Windows Containers`, the way to collect metrics from
these containers is quite different from runC. For the third-party metric
collectors, they have to maintain the knowledge about how to collect metrics
from each kind of container runtimes. And more, kubelet codebase focus on
core-metrics export (defined in [Kubernetes monitoring architecture#Terminology](https://github.com/kubernetes/community/blob/master/contributors/design-proposals/instrumentation/monitoring_architecture.md#terminology)),
and there is no powerful standard interface for container level metrics.
It will bring burden to end-users who wants to build powerful monitor pipeline
(defined in [Kubernetes monitoring architecture#Executive Summary](https://github.com/kubernetes/community/blob/master/contributors/design-proposals/instrumentation/monitoring_architecture.md#executive-summary)) for different container runtimes.

Based on this case, we can extend Container Runtime Interface(CRI) to define
the container level metrics. Each container runtime engine, which implemented
CRI, knows lifecycle and metrics about each managed container better than any
external metric collector. For the third-party metric collectors, they just
need to follow the `ContainerStats` API defined by CRI. With standard CRI,
it is easy for the collector from monitor pipeline (defined in [Kubernetes monitoring architecture](https://github.com/kubernetes/community/blob/master/contributors/design-proposals/instrumentation/monitoring_architecture.md#executive-summary)) to collect metrics from different kind of runtimes.

### Goals

Enhance Container Runtime Interface(CRI) to provide more metrics about managed
container to

- Reduce resource usage of kubelet
- Make eaiser integration for Monitoring pipeline

## Proposal

### User Stories

As administrator, I provides sandbox runtime for applications in my cluster. I
am not expertise of the sandbox runtime, but CRI implementation has created by
domain expertise and CRI-API provides useful and powerful metrics for
monitoring.

To expose sandbox container's metrics, I only need to run Daemonset Monitoring
Agent to collect container metrics from CRI Runtime Endpoint and show it in
dashboard. It doesn't bring burden on kubelet and makes eaiser to enhance
monitoring pipeline.

### Implementation Details/Notes/Constraints

Extend `ContainerStats` message in CRI.

`CpuUsage` will cover usage about kernel/user modes and per CPU.

```protobuf
// CpuUsage provides the CPU usage information.
message CpuUsage {
    ...

    // Time spent in kernel space.
    UInt64Value usage_in_kernel_nano_seconds = 3;
    // Time spent in user space.
    UInt64Value usage_in_user_nano_seconds = 4;
    // Per CPU usage.
    repeated UInt64Value per_cpu_usage_nano_seconds = 5;
}
```

`MemoryUsage` will cover usage, cache, rss, pgfault etc.

```protobuf
// MemoryUsage provides the memory usage information.
message MemoryUsage {
    ...

    // Number of bytes of total current memory usage by processes.
    Uint64Value usage_bytes = 3;
    // Number of bytes of the maximum memory used by processes.
    Uint64Value max_usage_bytes = 4;
    // Number of bytes of page cache memory, including tmpfs(shmem).
    UInt64Value cache_bytes = 5;
    // Number of bytes of anonymous and swap cache, not including tmpfs(shmem).
    Uint64Value rss_bytes = 6;
    // Number of bytes of memory-mapped mapped files, including tmpfs(shmem).
    Uint64Value mapped_file_bytes = 7;
    // Number of times that the memory limit has reached.
    Uint64Value failcnt = 8;
    // Number of times that processes triggered page fault.
    Uint64Value pgfault = 9;
    // Number of times that processes triggered major fault.
    Uint64Value pgmajfault = 10;
}
```

New `ProcessUsage` shows current running processes.

```protobuf
// ProcessUsage provides the process usage information.
message ProcessUsage {
    // Timestamp in nanoseconds at which the information were collected. Must be > 0.
    int64 timestamp = 1;
    // Number of current processes.
    Uint64Value current_process = 2;
}
```

`NetworkUsage` shows current network usage information about container.

```protobuf
// NetworkUsage provides the network usage information.
message NetworkUsage {
    // Timestamp in nanoseconds at which the information were collected. Must be > 0.
    int64 timestamp = 1;
    // Per Interface usage.
    repeated NetworkInterfaceUsage interfaces = 2;
}

// NetworkInterfaceUsage provides interface-level usage information.
message NetworkInterfaceUsage {
     // The name of the interface.
    string name = 1;
    // Number of received bytes.
    Uint64Value rx_bytes = 2;
    // Number of received packets.
    Uint64Value rx_packets = 3;
    // Number of received errors.
    Uint64Value rx_errors = 4;
    // Number of dropped packets during receiving.
    Uint64Value rx_dropped = 5;
    // Number of transmitted bytes.
    Uint64Value tx_bytes = 6;
    // Number of transmitted packets.
    Uint64Value tx_packets = 7;
    // Number of transmit errors.
    Uint64Value tx_errors = 8;
    // Number of dropped packets during transmitting.
    Uint64Value tx_dropped = 9;
}
```

And add `ProcessUsage/NetworkUsage` into `ContainerStat` message.

```protobuf
message ContainerStats {
    ...

    // Process usage gathered from the container.
    ProcessUsage process = 5;
    // Network usage gathered from the container.
    NetworkUsage network = 6;
}
```

## Design Details

### Test Plan

<!-- TBD -->

### Graduation Criteria

<!-- TBD -->

### Upgrade / Downgrade Strategy

<!-- TBD -->

### Version Skew Strategy

<!-- TBD -->

## Implementation History

<!-- TBD -->
