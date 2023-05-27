# Kubelet Resource Metrics Endpoint

## Table of Contents

<!-- toc -->
- [Release Signoff Checklist](#release-signoff-checklist)
- [Summary](#summary)
- [Motivation](#motivation)
  - [Background](#background)
  - [Goals](#goals)
  - [Non-Goals](#non-goals)
- [Proposal](#proposal)
  - [Risks and Mitigations](#risks-and-mitigations)
- [Design Details](#design-details)
  - [API](#api)
  - [Future Improvements](#future-improvements)
  - [Benchmarking](#benchmarking)
    - [Round 1](#round-1)
      - [Methods](#methods)
      - [Results](#results)
    - [Round 2](#round-2)
      - [Methods](#methods-1)
      - [Results](#results-1)
  - [Test Plan](#test-plan)
      - [Prerequisite testing updates](#prerequisite-testing-updates)
      - [Unit tests](#unit-tests)
      - [Integration tests](#integration-tests)
      - [e2e tests](#e2e-tests)
  - [Graduation Criteria](#graduation-criteria)
  - [Upgrade / Downgrade Strategy](#upgrade--downgrade-strategy)
  - [Version Skew Strategy](#version-skew-strategy)
- [Production Readiness Review Questionnaire](#production-readiness-review-questionnaire)
  - [Feature Enablement and Rollback](#feature-enablement-and-rollback)
  - [Rollout, Upgrade and Rollback Planning](#rollout-upgrade-and-rollback-planning)
  - [Monitoring Requirements](#monitoring-requirements)
  - [Dependencies](#dependencies)
  - [Scalability](#scalability)
  - [Troubleshooting](#troubleshooting)
- [Implementation History](#implementation-history)
- [Drawbacks](#drawbacks)
- [Alternatives Considered](#alternatives-considered)
  - [gRPC API](#grpc-api)
- [Infrastructure Needed (Optional)](#infrastructure-needed-optional)
<!-- /toc -->

## Release Signoff Checklist

<!--
**ACTION REQUIRED:** In order to merge code into a release, there must be an
issue in [kubernetes/enhancements] referencing this KEP and targeting a release
milestone **before the [Enhancement Freeze](https://git.k8s.io/sig-release/releases)
of the targeted release**.

For enhancements that make changes to code or processes/procedures in core
Kubernetes—i.e., [kubernetes/kubernetes], we require the following Release
Signoff checklist to be completed.

Check these off as they are completed for the Release Team to track. These
checklist items _must_ be updated for the enhancement to be released.
-->

Items marked with (R) are required *prior to targeting to a milestone / release*.

- [x] (R) Enhancement issue in release milestone, which links to KEP dir in [kubernetes/enhancements] (not the initial KEP PR)
- [x] (R) KEP approvers have approved the KEP status as `implementable`
- [x] (R) Design details are appropriately documented
- [x] (R) Test plan is in place, giving consideration to SIG Architecture and SIG Testing input (including test refactors)
  - [x] e2e Tests for all Beta API Operations (endpoints)
  - [x] (R) Ensure GA e2e tests meet requirements for [Conformance Tests](https://github.com/kubernetes/community/blob/master/contributors/devel/sig-architecture/conformance-tests.md) 
  - [ ] (R) Minimum Two Week Window for GA e2e tests to prove flake free
- [x] (R) Graduation criteria is in place
  - [x] (R) [all GA Endpoints](https://github.com/kubernetes/community/pull/1806) must be hit by [Conformance Tests](https://github.com/kubernetes/community/blob/master/contributors/devel/sig-architecture/conformance-tests.md) 
- [x] (R) Production readiness review completed
- [ ] (R) Production readiness review approved
- [x] "Implementation History" section is up-to-date for milestone
- [ ] User-facing documentation has been created in [kubernetes/website], for publication to [kubernetes.io]
- [ ] Supporting documentation—e.g., additional design documents, links to mailing list discussions/SIG meetings, relevant PRs/issues, release notes

<!--
**Note:** This checklist is iterative and should be reviewed and updated every time this enhancement is being considered for a milestone.
-->

[kubernetes.io]: https://kubernetes.io/
[kubernetes/enhancements]: https://git.k8s.io/enhancements
[kubernetes/kubernetes]: https://git.k8s.io/kubernetes
[kubernetes/website]: https://git.k8s.io/website

## Summary

The Kubelet Resource Metrics Endpoint is a new kubelet metrics endpoint which serves metrics required by the cluster-level [Resource Metrics API](https://github.com/kubernetes/metrics#resource-metrics-api).  The proposed design uses the prometheus text format, and provides the minimum required metrics for serving the [Resource Metrics API](https://github.com/kubernetes/metrics#resource-metrics-api).

## Motivation

The Kubelet Summary API is a source of both Resource and Monitoring Metrics.  Because of it’s dual purpose, it does a poor job of both.  It provides much more information than required by the Metrics Server, as demonstrated by [kubernetes/kubernetes#68841](https://github.com/kubernetes/kubernetes/pull/68841).  Additionally, we have pushed back on adding metrics to the Summary API for monitoring, such as DiskIO or tcp/udp metrics, because they are expensive to collect, and not required by all users.

This proposal deals with the first problem, which is that the Summary API is a poor provider of Resource Metrics.  It proposes a purpose-built API for supplying Resource Metrics.

### Background

The [Monitoring Architecture](https://github.com/kubernetes/design-proposals-archive/blob/master/instrumentation/monitoring_architecture.md) proposal established separate pipelines for Resource Metrics, and for Monitoring Metrics.  The [Core Metrics](https://github.com/kubernetes/design-proposals-archive/blob/master/instrumentation/core-metrics-pipeline.md#core-metrics-in-kubelet) proposal describes the set of metrics that we consider core, and their uses.  Note that the term “core” is overloaded, and this document will refer to these as Resource Metrics, since they are for first class kubernetes resources and are served by the [Resource Metrics API](https://github.com/kubernetes/metrics#resource-metrics-api) at the cluster-level.

A [previous proposal](https://docs.google.com/document/d/1_CdNWIjPBqVDMvu82aJICQsSCbh2BR-y9a8uXjQm4TI/edit?usp=sharing) by @DirectXMan12 also proposed a prometheus endpoint.  The [kubernetes metrics overhaul KEP](https://github.com/kubernetes/enhancements/blob/master/keps/sig-instrumentation/20181106-kubernetes-metrics-overhaul.md#export-less-metrics) acknowledges the need to export fewer metrics from the kubelet.  This new API is a step in that direction, as it eliminates the Metric Server's dependency on the Summary API.

For the purposes of this document, I will use the following definitions:

* Resource Metrics: Metrics for the consumption of first-class resources (CPU, Memory, Ephemeral Storage) which are aggregated by the [Metrics Server](https://github.com/kubernetes-incubator/metrics-server#kubernetes-metrics-server), and served by the [Resource Metrics API](https://github.com/kubernetes/metrics#resource-metrics-api)
* Monitoring Metrics: Metrics for observability and introspection of the cluster, which are used by end-users, operators, devs, etc. 


The Kubelet’s [JSON Summary API](https://github.com/kubernetes/kubernetes/blob/master/pkg/kubelet/apis/stats/v1alpha1/types.go) is currently used by the [Metrics Server](https://github.com/kubernetes-incubator/metrics-server#kubernetes-metrics-server).  It contains far more metrics than are required by the Metrics Server.

[Prometheus](https://prometheus.io/) is commonly used for exposing metrics for kubernetes components, and the [Prometheus Operator](https://github.com/coreos/prometheus-operator#prometheus-operator), which Sig-Instrumentation works on, is commonly used to deploy and manage metrics collection.

[OpenMetrics](https://openmetrics.io/) is a new prometheus-based metric standard which supports both text and protobuf.  

[GRPC](https://grpc.io/) is commonly used for interfaces between components in kubernetes, such as the [Container Runtime Interface](https://github.com/kubernetes/kubernetes/blob/master/pkg/kubelet/apis/cri/runtime/v1alpha2/api.proto).  GRPC uses [protocol-buffers](https://developers.google.com/protocol-buffers/docs/overview) (protobuf) for serialization and deserialization, which is more performant than other formats.

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

The metrics in this endpoint will make use of the [Kubernetes Metrics Stability framework](https://github.com/kubernetes/enhancements/blob/master/keps/sig-instrumentation/1209-metrics-stability/kubernetes-control-plane-metrics-stability.md) for stability and deprecation policies.

### Risks and Mitigations

## Design Details

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

### Future Improvements

[OpenMetrics](https://openmetrics.io/) is an upcoming prometheus-based standard which has support for protocol buffers.  By using this format when it becomes available, we can further improve the efficiency of the Resource Metrics Pipeline, while maintaining compatibility with other monitoring pipelines.

### Benchmarking

#### Round 1

This experiment compares the current JSON Summary API to prometheus and GRPC at 1s and 30s scrape intervals.  Prometheus uses basic text parsing, and grpc uses a basic `Get()` API.

##### Methods

The setup has 10 nodes, 500 pods, and 6500 containers (running pause).  Nodes have 1 CPU core, and 3.75Gb memory.  The same cluster was used for all benchmarks for consistency, with a different Metrics Server running.  The values below are the maximum values reported during a 10 minute period.

##### Results

We can see that GRPC has the lowest CPU usage of all formats tested, and is an order-of-magnitude improvement over the current JSON Summary API.  Memory Usage for both GRPC and Prometheus are similarly lower than the JSON Summary API.

<img src="https://user-images.githubusercontent.com/3262098/51704936-1dca4f80-1fcf-11e9-9485-b4c765a5a1c9.png" width="600" height="375">

<img src="https://user-images.githubusercontent.com/3262098/51704931-1acf5f00-1fcf-11e9-93aa-8004b43e6770.png" width="600" height="375">

#### Round 2

After learning that the prometheus server achieves better performance with caching, I performed an additional round of tests.  These used a metrics-server which caches metric descriptors it has parsed before, and tested with larger numbers of container metrics.

This experiment compares basic prometheus, optimized prometheus parsing and GRPC at 1s scrape intervals with higher numbers of container metrics.  "Unoptimized Prometheus" uses basic text parsing, "Prometheus w/ Caching" borrows [caching logic from the prometheus server](https://github.com/prometheus/prometheus/blob/master/scrape/scrape.go#L991) to avoid re-parsing metric descriptors it has already parsed and grpc uses a basic `Get()` API.

##### Methods

The setup has 10 nodes, and up to 40,000 containers (running pause).  Nodes have 2 CPU core, and 7.5Gb memory.  The same cluster was used for all benchmarks for consistency, with a different Metrics Server running.  The values below are the maximum values reported during a 10 minute period.

This experiment "fakes" large numbers of containers by having the kubelet return 100 container metrics for each actual container run on the node.

##### Results

Both gRPC and the optimized prometheus were able to scale to 40k containers.  The gRPC implementation was more efficient by a factor of approx. 3.  

<img src="https://user-images.githubusercontent.com/3262098/51704923-173bd800-1fcf-11e9-910d-3fd6606550f3.png" width="600" height="375">

<img src="https://user-images.githubusercontent.com/3262098/51704880-02f7db00-1fcf-11e9-8034-c64f971a2204.png" width="600" height="375">

### Test Plan

[X] I/we understand the owners of the involved components may require updates to
existing tests to make this code solid enough prior to committing the changes necessary
to implement this enhancement.

##### Prerequisite testing updates


##### Unit tests

- `<package>`: `<date>` - `<test coverage>`

##### Integration tests

- <test>: <link to test coverage>

##### e2e tests

Test the new endpoint with a node-e2e test similar to the current summary API test.
Testgrid: https://k8s-testgrid.appspot.com/sig-node-kubelet#node-kubelet-features-master&include-filter-by-regex=ResourceMetricsAPI

### Graduation Criteria

Alpha:

- [X] Implement the kubelet resource metrics endpoint as described above

Beta:

- [X] Modify the metrics server to consume the kubelet resource metrics endpoint 3 releases after it is added to the kubelet

GA:

- [X] Add [node-e2e test](https://github.com/kubernetes/kubernetes/pull/116897/files#diff-3859a7587ac4b3d1e162a2360b1fd2d3e88d4589be9b0bf19029fa7489294796R59-R70)

### Upgrade / Downgrade Strategy

The kubelet can be upgraded or downgraded normally with respect to this feature. Users of the metrics endpoint, such as the metrics server, should use other kubelet metrics endpoints (such as the summary api) before downgrading.

### Version Skew Strategy

This feature affects only the kubelet - in that it will expose the resource metrics for kubelet in a new endpoint, so there is no issue with version skew with other components.

## Production Readiness Review Questionnaire

### Feature Enablement and Rollback

###### How can this feature be enabled / disabled in a live cluster?

- [ ] Feature gate (also fill in values in `kep.yaml`)
  - Feature gate name:
  - Components depending on the feature gate:
- [x] Other
  - Describe the mechanism: This feature exposes the /metrics/resource endpoint for kubelet, with all metrics annotated as STABLE. **Note:** Because this feature was built before the PRR process was established, it unfortunately does not adhere to the best practices of feature enablement/disablement 
  - Will enabling / disabling the feature require downtime of the control
    plane? No
  - Will enabling / disabling the feature require downtime or reprovisioning
    of a node? No

###### Does enabling the feature change any default behavior?

It will expose the /metrics/resource endpoint for kubelet by default

###### Can the feature be disabled once it has been enabled (i.e. can we roll back the enablement)?

No, this feature can not be disabled once it has been enabled since we do not have a feature flag for this. To rollback, one will have to downgrade the kubernetes version. Note: This version was added in v1.14, so to disable this feature, one would need to switch back to a version older than v1.14 

###### What happens if we reenable the feature if it was previously rolled back?

/metrics/resource endpoint for kubelet will become available

###### Are there any tests for feature enablement/disablement?

Since there is no feature gate involved for this, there are no feature enablement/disablement test

### Rollout, Upgrade and Rollback Planning

###### How can a rollout or rollback fail? Can it impact already running workloads?

A rollback can impact running workloads if clients, such as the metrics server, are relying on metrics provided by the endpoint. The rollback could break cluster functions, such as HPA, if the metrics were no longer available.

###### What specific metrics should inform a rollback?

The following metrics exposed by /kubelet/resource endpoint could be used:
- node_memory_working_set_bytes
- pod_memory_working_set_bytes

We could compute node_memory_working_set_bytes - sum(pod_memory_working_set_bytes) to know if there's a memory leak. 

###### Were upgrade and rollback tested? Was the upgrade->downgrade->upgrade path tested?

No, because the feature was enabled (with no way to disable) since v1.14.

###### Is the rollout accompanied by any deprecations and/or removals of features, APIs, fields of API types, flags, etc.?

No

### Monitoring Requirements

###### How can an operator determine if the feature is in use by workloads?

By checking kubelet's /metrics/resource endpoint

###### How can someone using this feature know that it is working for their instance?

- [ ] Events
  - Event Reason: 
- [ ] API .status
  - Condition name: 
  - Other field: 
- [X] Other (treat as last resort)
  - Details: /metrics/resource endpoint for kubelet should show resource metrics

###### What are the reasonable SLOs (Service Level Objectives) for the enhancement?

This feature introduces a metrics endpoint that can used to establish SLOs

###### What are the SLIs (Service Level Indicators) an operator can use to determine the health of the service?

This feature introduces a metrics endpoint that can be used to determine health of kubelet.

###### Are there any missing metrics that would be useful to have to improve observability of this feature?

No

### Dependencies

###### Does this feature depend on any specific services running in the cluster?

Kubelet

### Scalability

###### Will enabling / using this feature result in any new API calls?

No

###### Will enabling / using this feature result in introducing new API types?

No

###### Will enabling / using this feature result in any new calls to the cloud provider?

No

###### Will enabling / using this feature result in increasing size or count of the existing API objects?

No

###### Will enabling / using this feature result in increasing time taken by any operations covered by existing SLIs/SLOs?

No

###### Will enabling / using this feature result in non-negligible increase of resource usage (CPU, RAM, disk, IO, ...) in any components?

No, infact CPU usage is reduced as compared to the Summary API's usage which was previously used my the metrics server.

###### Can enabling / using this feature result in resource exhaustion of some node resources (PIDs, sockets, inodes, etc.)?

No

### Troubleshooting

###### How does this feature react if the API server and/or etcd is unavailable?

No impact

###### What are other known failure modes?

/metrics/resource endpoint is not available

###### What steps should be taken if SLOs are not being met to determine the problem?

Memory leaks should be checked by looking at node_memory_working_set_bytes - sum(pod_memory_working_set_bytes)
If the problem is severe, kubernetes version should be downgraded so that the /metrics/resource endpoint is not exposed for kubelet. Keep in mind, users of these metrics should use other metrics endpoints (such as the summary api) before downgrading.

## Implementation History

- 2019-01-24: Initial KEP published.
- 2019-01-29: Presentation to Sig-Node
- 2019-02-04: KEP gets LGTM and Approval
- 2019-02-07: Presentation to Sig-Instrumentation
- 2020-01-14: [1.18] Endpoint copied from /metrics/resource/v1alpha1 to /metrics/resource, and adopting the metrics stability framework: https://github.com/kubernetes/kubernetes/pull/86282
- 2020-09-01: [1.20] /metrics/resource/v1alpha1 removed: https://github.com/kubernetes/kubernetes/pull/94272
- 2021-06-28: Use kubelet's /metrics/resource endpoint in metrics-server: https://github.com/kubernetes-sigs/metrics-server/pull/777
- 2023-08-23: [1.29] GA graduation, non conformance test added https://github.com/kubernetes/kubernetes/pull/116897
- 2023-09-08: [1.29] Promoted test to conformance test https://github.com/kubernetes/kubernetes/pull/120473

## Drawbacks

<!--
Why should this KEP _not_ be implemented?
-->

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

## Infrastructure Needed (Optional)

<!--
Use this section if you need things from the project/SIG. Examples include a
new subproject, repos requested, or GitHub details. Listing these here allows a
SIG to get the process for these resources started right away.
-->
