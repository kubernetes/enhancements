---
title: Kubelet Resource Metrics Endpoint
authors:
  - "@dashpole"
owning-sig: sig-node
participating-sigs:
  - sig-instrumentation
reviewers:
  - DirectXMan12
  - tallclair
approvers:
  - dchen1107
  - brancz
creation-date: 2019-01-24
last-updated: 2019-02-21
status: implementable
---

# Kubelet Resource Metrics Endpoint

## Table of Contents

<!-- toc -->
- [Summary](#summary)
- [Background](#background)
- [Motivation](#motivation)
  - [Goals](#goals)
  - [Non-Goals](#non-goals)
- [Proposal](#proposal)
  - [API](#api)
- [Future Improvements](#future-improvements)
- [Benchmarking](#benchmarking)
  - [Round 1](#round-1)
    - [Methods](#methods)
    - [Results](#results)
  - [Round 2](#round-2)
    - [Methods](#methods-1)
    - [Results](#results-1)
- [Alternatives Considered](#alternatives-considered)
  - [gRPC API](#grpc-api)
  - [Test Plan](#test-plan)
- [Graduation Criteria](#graduation-criteria)
- [Implementation History](#implementation-history)
<!-- /toc -->

## Summary

The Kubelet Resource Metrics Endpoint is a new kubelet metrics endpoint which serves metrics required by the cluster-level [Resource Metrics API](https://github.com/kubernetes/metrics#resource-metrics-api).  The proposed design uses the prometheus text format, and provides the minimum required metrics for serving the [Resource Metrics API](https://github.com/kubernetes/metrics#resource-metrics-api).

## Background

The [Monitoring Architecture](https://github.com/kubernetes/community/blob/master/contributors/design-proposals/instrumentation/monitoring_architecture.md) proposal established separate pipelines for Resource Metrics, and for Monitoring Metrics.  The [Core Metrics](https://github.com/kubernetes/community/blob/master/contributors/design-proposals/instrumentation/core-metrics-pipeline.md#core-metrics-in-kubelet) proposal describes the set of metrics that we consider core, and their uses.  Note that the term “core” is overloaded, and this document will refer to these as Resource Metrics, since they are for first class kubernetes resources and are served by the [Resource Metrics API](https://github.com/kubernetes/metrics#resource-metrics-api) at the cluster-level.

A [previous proposal](https://docs.google.com/document/d/1_CdNWIjPBqVDMvu82aJICQsSCbh2BR-y9a8uXjQm4TI/edit?usp=sharing) by @DirectXMan12 also proposed a prometheus endpoint.  The [kubernetes metrics overhaul KEP](https://github.com/kubernetes/enhancements/blob/master/keps/sig-instrumentation/20181106-kubernetes-metrics-overhaul.md#export-less-metrics) acknowledges the need to export fewer metrics from the kubelet.  This new API is a step in that direction, as it eliminates the Metric Server's dependency on the Summary API.

For the purposes of this document, I will use the following definitions:

* Resource Metrics: Metrics for the consumption of first-class resources (CPU, Memory, Ephemeral Storage) which are aggregated by the [Metrics Server](https://github.com/kubernetes-incubator/metrics-server#kubernetes-metrics-server), and served by the [Resource Metrics API](https://github.com/kubernetes/metrics#resource-metrics-api)
* Monitoring Metrics: Metrics for observability and introspection of the cluster, which are used by end-users, operators, devs, etc. 


The Kubelet’s [JSON Summary API](https://github.com/kubernetes/kubernetes/blob/master/pkg/kubelet/apis/stats/v1alpha1/types.go) is currently used by the [Metrics Server](https://github.com/kubernetes-incubator/metrics-server#kubernetes-metrics-server).  It contains far more metrics than are required by the Metrics Server.

[Prometheus](https://prometheus.io/) is commonly used for exposing metrics for kubernetes components, and the [Prometheus Operator](https://github.com/coreos/prometheus-operator#prometheus-operator), which Sig-Instrumentation works on, is commonly used to deploy and manage metrics collection.

[OpenMetrics](https://openmetrics.io/) is a new prometheus-based metric standard which supports both text and protobuf.  

[GRPC](https://grpc.io/) is commonly used for interfaces between components in kubernetes, such as the [Container Runtime Interface](https://github.com/kubernetes/kubernetes/blob/master/pkg/kubelet/apis/cri/runtime/v1alpha2/api.proto).  GRPC uses [protocol-buffers](https://developers.google.com/protocol-buffers/docs/overview) (protobuf) for serialization and deserialization, which is more performant than other formats.

## Motivation

The Kubelet Summary API is a source of both Resource and Monitoring Metrics.  Because of it’s dual purpose, it does a poor job of both.  It provides much more information than required by the Metrics Server, as demonstrated by [kubernetes/kubernetes#68841](https://github.com/kubernetes/kubernetes/pull/68841).  Additionally, we have pushed back on adding metrics to the Summary API for monitoring, such as DiskIO or tcp/udp metrics, because they are expensive to collect, and not required by all users.

This proposal deals with the first problem, which is that the Summary API is a poor provider of Resource Metrics.  It proposes a purpose-built API for supplying Resource Metrics.

### Goals

* [Primary] Provide the minimum set of metrics required to serve the Resource Metrics API
* [Secondary] Minimize the CPU and Memory footprint of the metrics server due to collecting metrics
  * Perform efficiently at frequent (sub-second) rates of metrics collection
* [Secondary] Use a format that is familiar to the kubernetes community, which can be consumed by common monitoring pipelines, and is interoperable with commonly-used monitoring pipelines.

### Non-Goals

* Deprecate or remove the Summary API
* Add new Resource Metrics to the metrics server (e.g. Ephemeral Storage)
* Detail how the kubelet will collect metrics to support this API.
* Determine what the pipeline for “Monitoring” metrics will look like

## Proposal

The kubelet will expose an endpoint at `/metrics/resource` in prometheus text exposition format using the prometheus client library.

The metrics in this endpoint will make use of the [Kubernetes Metrics Stability framework](https://github.com/kubernetes/enhancements/blob/master/keps/sig-instrumentation/20190404-kubernetes-control-plane-metrics-stability.md) for stability and deprecation policies.


### API

```
# Cumulative cpu time consumed by a container in seconds
Name: container_cpu_usage_seconds_total
Labels: container, pod, namespace

# Current working set of a container in bytes
Name: container_memory_working_set_bytes
Labels: container, pod, namespace

# Cumulative cpu time consumed by the node in seconds
Name: node_cpu_usage_seconds_total
Labels: 

# Current working set of the node in bytes
Name: node_memory_working_set_bytes
Labels:
```

Explicit timestamps (see the [prometheus exposition format docs](https://github.com/prometheus/docs/blob/master/content/docs/instrumenting/exposition_formats.md#comments-help-text-and-type-information)) will be added to metrics because metrics are (currently) collected out-of-band and cached.  We make no guarantees about the age of metrics, but include the timestamp to allow readers to correctly calculate rates, etc.  Timestamps are currently required because metrics are collected out-of-band by cAdvisor.  This deviates from the [prometheus best practices](https://prometheus.io/docs/instrumenting/writing_exporters/#scheduling), and we should attempt to migrate to synchronous collection during each scrape in the future.

Use separate metrics for node and containers to avoid “magic” container names, such as “machine”.

Currently the Metrics Server uses a 10s average of CPU usage provided by the kubelet summary API.  The kubelet should provide the raw cumulative CPU usage so the metrics server can determine the time period over which it wants to take the rate.

Labels are named in accordance with the [kubernetes instrumentation guidelines](https://github.com/kubernetes/community/blob/master/contributors/devel/sig-instrumentation/instrumentation.md#resource-referencing), and thus are named `pod`, rather than `pod_name`.

Example implementation: https://github.com/kubernetes/kubernetes/compare/master...dashpole:prometheus_core_metrics

## Future Improvements

[OpenMetrics](https://openmetrics.io/) is an upcoming prometheus-based standard which has support for protocol buffers.  By using this format when it becomes available, we can further improve the efficiency of the Resource Metrics Pipeline, while maintaining compatibility with other monitoring pipelines.

## Benchmarking

### Round 1

This experiment compares the current JSON Summary API to prometheus and GRPC at 1s and 30s scrape intervals.  Prometheus uses basic text parsing, and grpc uses a basic `Get()` API.

#### Methods

The setup has 10 nodes, 500 pods, and 6500 containers (running pause).  Nodes have 1 CPU core, and 3.75Gb memory.  The same cluster was used for all benchmarks for consistency, with a different Metrics Server running.  The values below are the maximum values reported during a 10 minute period.

#### Results

We can see that GRPC has the lowest CPU usage of all formats tested, and is an order-of-magnitude improvement over the current JSON Summary API.  Memory Usage for both GRPC and Prometheus are similarly lower than the JSON Summary API.

<img src="https://user-images.githubusercontent.com/3262098/51704936-1dca4f80-1fcf-11e9-9485-b4c765a5a1c9.png" width="600" height="375">

<img src="https://user-images.githubusercontent.com/3262098/51704931-1acf5f00-1fcf-11e9-93aa-8004b43e6770.png" width="600" height="375">

### Round 2

After learning that the prometheus server achieves better performance with caching, I performed an additional round of tests.  These used a metrics-server which caches metric descriptors it has parsed before, and tested with larger numbers of container metrics.

This experiment compares basic prometheus, optimized prometheus parsing and GRPC at 1s scrape intervals with higher numbers of container metrics.  "Unoptimized Prometheus" uses basic text parsing, "Prometheus w/ Caching" borrows [caching logic from the prometheus server](https://github.com/prometheus/prometheus/blob/master/scrape/scrape.go#L991) to avoid re-parsing metric descriptors it has already parsed and grpc uses a basic `Get()` API.

#### Methods

The setup has 10 nodes, and up to 40,000 containers (running pause).  Nodes have 2 CPU core, and 7.5Gb memory.  The same cluster was used for all benchmarks for consistency, with a different Metrics Server running.  The values below are the maximum values reported during a 10 minute period.

This experiment "fakes" large numbers of containers by having the kubelet return 100 container metrics for each actual container run on the node.

#### Results

Both gRPC and the optimized prometheus were able to scale to 40k containers.  The gRPC implementation was more efficient by a factor of approx. 3.  

<img src="https://user-images.githubusercontent.com/3262098/51704923-173bd800-1fcf-11e9-910d-3fd6606550f3.png" width="600" height="375">

<img src="https://user-images.githubusercontent.com/3262098/51704880-02f7db00-1fcf-11e9-8034-c64f971a2204.png" width="600" height="375">

## Alternatives Considered

### gRPC API

As demonstrated in the benchmarks above, the proto-based gRPC endpoint is the most efficient in terms of CPU and Memory usage.  Such an endpoint could potentially be improved by using streaming, rather than scraping to be even more efficient at high rates of collection.

However, given the prevalence of the Prometheus format within the kubernetes community, gRPC is not as compatible with common monitoring pipelines.  The endpoint would _only_ be useful for supplying metrics for the Metrics Server, or monitoring components that integrate directly with it.

When using caching in the Metrics Server, the prometheus text format performs _well enough_ for us to prefer prometheus over gRPC given the prevalence of prometheus in the community.  When the OpenMetrics format becomes stable, we can get even closer to the performance of gRPC by using the proto-based format.

```
// Usage is a set of resources consumed
message Usage {
  int64 time = 1;
  uint64 cpu_usage_core_nanoseconds_total = 2;
  uint64 memory_working_set_bytes = 3;
}
// ContainerUsage is the resource usage for a single container
message ContainerUsage {
  string name = 1;
  Usage usage = 2;
}
// PodUsage is the resource usage for a pod
message PodUsage {
  string name = 1;
  string namespace = 2;
  repeated ContainerUsage containers = 3;
}
// MetricsResponse is sent by plugin to kubelet in response to MetricsRequest RPC
message MetricsResponse {
  Usage node = 1;
  repeated PodUsage pods = 2;
}
// MetricsRequest is the empty request message for Kubelet
message MetricsRequest {}
// ResourceMetrics is the service advertised by the kubelet for usage metrics.
service ResourceMetrics {
  rpc Get(MetricsRequest) returns (MetricsResponse) {}
}
```

### Test Plan

Test the new endpoint with a node-e2e test similar to the current summary API test.
Testgrid: https://k8s-testgrid.appspot.com/sig-node-kubelet#node-kubelet-features-master&include-filter-by-regex=ResourceMetricsAPI

## Graduation Criteria

Alpha:

- [X] Implement the kubelet resource metrics endpoint as described above

Beta:

- [ ] Modify the metrics server to consume the kubelet resource metrics endpoint 3 releases after it is added to the kubelet

GA:

- [ ] Add node-e2e test to the node conformance tests

## Implementation History

- 2019-01-24: Initial KEP published.
- 2019-01-29: Presentation to Sig-Node
- 2019-02-04: KEP gets LGTM and Approval
- 2019-02-07: Presentation to Sig-Instrumentation
- 2020-01-14: [1.18] Endpoint copied from /metrics/resource/v1alpha1 to /metrics/resource, and adopting the metrics stability framework: https://github.com/kubernetes/kubernetes/pull/86282
- 2020-09-01: [1.20] /metrics/resource/v1alpha1 removed: https://github.com/kubernetes/kubernetes/pull/94272
