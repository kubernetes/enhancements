# KEP-647: APIServer Tracing

<!-- toc -->
- [Release Signoff Checklist](#release-signoff-checklist)
- [Summary](#summary)
- [Motivation](#motivation)
  - [Definitions](#definitions)
  - [Goals](#goals)
  - [Non-Goals](#non-goals)
- [Proposal](#proposal)
  - [Tracing API Requests](#tracing-api-requests)
  - [Exporting Spans](#exporting-spans)
  - [Running the OpenTelemetry Collector](#running-the-opentelemetry-collector)
  - [APIServer Configuration and EgressSelectors](#apiserver-configuration-and-egressselectors)
  - [Controlling use of the OpenTelemetry library](#controlling-use-of-the-opentelemetry-library)
  - [Test Plan](#test-plan)
- [Graduation requirements](#graduation-requirements)
- [Production Readiness Survey](#production-readiness-survey)
- [Implementation History](#implementation-history)
- [Alternatives considered](#alternatives-considered)
  - [Introducing a new EgressSelector type](#introducing-a-new-egressselector-type)
  - [Other OpenTelemetry Exporters](#other-opentelemetry-exporters)
<!-- /toc -->

## Release Signoff Checklist

Items marked with (R) are required *prior to targeting to a milestone / release*.

- [X] (R) Enhancement issue in release milestone, which links to KEP dir in [kubernetes/enhancements] (not the initial KEP PR)
- [X] (R) KEP approvers have approved the KEP status as `implementable`
- [X] (R) Design details are appropriately documented
- [X] (R) Test plan is in place, giving consideration to SIG Architecture and SIG Testing input
- [X] (R) Graduation criteria is in place
- [x] (R) Production readiness review completed
- [ ] Production readiness review approved
- [X] "Implementation History" section is up-to-date for milestone
- [ ] User-facing documentation has been created in [kubernetes/website], for publication to [kubernetes.io]
- [ ] Supporting documentation—e.g., additional design documents, links to mailing list discussions/SIG meetings, relevant PRs/issues, release notes

## Summary

This Kubernetes Enhancement Proposal (KEP) proposes enhancing the API Server to allow tracing requests.  For this, it proposes using OpenTelemetry libraries, and exports in the OpenTelemetry format.

## Motivation

Along with metrics and logs, traces are a useful form of telemetry to aid with debugging incoming requests.  The API Server currently uses a poor-man's form of tracing (see [github.com/kubernetes/utils/trace](https://github.com/kubernetes/utils/tree/master/trace)), but we can make use of distributed tracing to improve the ease of use and enable easier analysis of trace data.  Trace data is structured, providing the detail necessary to debug requests, and context propagation allows plugins, such as admission webhooks, to add to API Server requests.

### Definitions

**Span**: The smallest unit of a trace.  It has a start and end time, and is attached to a single trace.
**Trace**: A collection of Spans which represents a single process.
**Trace Context**: A reference to a Trace that is designed to be propagated across component boundaries.  Sometimes referred to as the "Span Context".  It is can be thought of as a pointer to a parent span that child spans can be attached to.

### Goals

* The API Server generates and exports spans for incoming and outgoing requests.
* The API Server propagates context from incoming requests to outgoing requests.

### Non-Goals

* Tracing in kubernetes controllers
* Replace existing logging, metrics, or the events API
* Trace operations from all Kubernetes resource types in a generic manner (i.e. without manual instrumentation)
* Change metrics or logging (e.g. to support trace-metric correlation)
* Access control to tracing backends
* Add tracing to components outside kubernetes (e.g. etcd client library).

## Proposal

### Tracing API Requests

We will wrap the API Server's http server and http clients with [otelhttp](https://github.com/open-telemetry/opentelemetry-go-contrib/tree/master/instrumentation/net/http/otelhttp) to get spans for incoming and outgoing http requests.  This generates spans for all sampled incoming requests and propagates context with all client requests.  For incoming requests, this would go below [WithRequestInfo](https://github.com/kubernetes/kubernetes/blob/9eb097c4b07ea59c674a69e19c1519f0d10f2fa8/staging/src/k8s.io/apiserver/pkg/server/config.go#L676) in the filter stack, as it must be after authentication and authorization, before the panic filter, and is closest in function to the WithRequestInfo filter.

Note that some clients of the API Server, such as webhooks, may make reentrant calls to the API Server.  To gain the full benefit of tracing, such clients should propagate context with requests back to the API Server.  One way to do this is to use the wrap the webhook's http server using otelhttp, and use the request's context when making requests to the API Server.

**Webhook Example**

Wrapping the http server, which ensures context is propagated from http headers to the requests context:
```golang
mux := http.NewServeMux()
handler := otelhttp.NewHandler(mux, "HandleAdmissionRequest")
```
Use the context from the request in reentrant requests:
```golang
ctx := req.Context()
client.CoreV1().Pods("").List(ctx, metav1.ListOptions{})
```

Note: Even though the admission controller uses the otelhttp handler wrapper, that does _not_ mean it will emit spans.  OpenTelemetry has a concept of an SDK, which manages the exporting of telemetry.  If no SDK is registered, the NoOp SDK is used, which only propagates context, and does not export spans.  In the webhook case in which no SDK is registered, the reentrant API call would appear to be a direct child of the original API call.  If the webhook registers an SDK and exports spans, there would be an additional span from the webhook between the original and reentrant API Server call.

Note: OpenTelemetry has a concept of ["Baggage"](https://github.com/open-telemetry/opentelemetry-specification/blob/master/specification/baggage/api.md#baggage-api), which is akin to annotations for propagated context.  If there is any additional metadata we would like to attach, and propagate along with a request, we can do that using Baggage.

### Exporting Spans

This KEP proposes the use of the [OpenTelemetry tracing framework](https://opentelemetry.io/) to create and export spans to configured backends.

The API Server will use the [OpenTelemetry exporter format](https://github.com/open-telemetry/opentelemetry-proto), and the [OTlp exporter](https://github.com/open-telemetry/opentelemetry-go/tree/master/exporters/otlp#opentelemetry-collector-go-exporter) which can export traces.  This format is easy to use with the [OpenTelemetry Collector](https://github.com/open-telemetry/opentelemetry-collector), which allows importing and configuring exporters for trace storage backends to be done out-of-tree in addition to other useful features.

### Running the OpenTelemetry Collector

The [OpenTelemetry Collector](https://github.com/open-telemetry/opentelemetry-collector) can be run as a sidecar, a daemonset, a deployment , or a combination in which the daemonset buffers telemetry and forwards to the deployment for aggregation (e.g. tail-base sampling) and routing to a telemetry backend.  To support these various setups, the API Server should be able to send traffic either to a local (on the control plane network) collector, or to a cluster service (on the cluster network).

### APIServer Configuration and EgressSelectors

The API Server controls where traffic is sent using an [EgressSelector](https://github.com/kubernetes/enhancements/blob/master/keps/sig-api-machinery/20190226-network-proxy.md), and has separate controls for `Master`, `Cluster`, and `Etcd` traffic.  As described above, we would like to support either sending telemetry to a url using the `Master` egress, or a service using the `Cluster` egress.  To accomplish this, we will introduce a flag, `--opentelemetry-config-file`, that will point to the file that defines the opentelemetry exporter configuration.  That file will have the following format:

```golang
// +k8s:deepcopy-gen:interfaces=k8s.io/apimachinery/pkg/runtime.Object

// OpenTelemetryClientConfiguration provides versioned configuration for opentelemetry clients.
type OpenTelemetryClientConfiguration struct {
  metav1.TypeMeta `json:",inline"`

  // +optional
  // URL of the collector that's running on the master.
  // if URL is specified, APIServer uses the egressType Master when sending data to the collector.
  URL *string `json:"url,omitempty" protobuf:"bytes,3,opt,name=url"`

  // +optional
  // Service that's the frontend of the collector deployment running in the cluster.
  // If Service is specified, APIServer uses the egressType Cluster when sending data to the collector.
  Service *ServiceReference `json:"service,omitempty" protobuf:"bytes,1,opt,name=service"`
}

// ServiceReference holds a reference to Service.legacy.k8s.io
type ServiceReference struct {
  // `namespace` is the namespace of the service.
  // Required
  Namespace string `json:"namespace" protobuf:"bytes,1,opt,name=namespace"`
  // `name` is the name of the service.
  // Required
  Name string `json:"name" protobuf:"bytes,2,opt,name=name"`

  // If specified, the port on the service.
  // Defaults to 55680.
  // `port` should be a valid port number (1-65535, inclusive).
  // +optional
  Port *int32 `json:"port,omitempty" protobuf:"varint,3,opt,name=port"`
}
```

If `--opentelemetry-config-file` is not specified, the API Server will not send any telemetry.

### Controlling use of the OpenTelemetry library

As the community found in the [Metrics Stability Framework KEP](https://github.com/kubernetes/enhancements/blob/master/keps/sig-instrumentation/20190404-kubernetes-control-plane-metrics-stability.md#kubernetes-control-plane-metrics-stability), having control over how the client libraries are used in kubernetes can enable maintainers to enforce policy and make broad improvements to the quality of telemetry.  To enable future improvements to tracing, we will restrict the direct use of the OpenTelemetry library within the kubernetes code base, and provide wrapped versions of functions we wish to expose in a utility library.

### Test Plan

We will e2e test this feature by deploying an OpenTelemetry Collector on the master, and configure it to export traces using the [stdout exporter](https://github.com/open-telemetry/opentelemetry-go/tree/master/exporters/stdout), which logs the spans in json format.  We can then verify that the logs contain our expected traces when making calls to the API Server.

## Graduation requirements

Alpha

- [] Implement tracing of incoming and outgoing http/grpc requests in the kube-apiserver
- [] Tests are in testgrid and linked in KEP

Beta

- [] Tracing 100% of requests does not break scalability tests (this does not necessarily mean trace backends can handle all the data).
- [] OpenTelemetry reaches GA
- [] Publish examples of how to use the OT Collector with kubernetes
- [] Allow time for feedback
- [] Revisit the format used to export spans.

GA

- [] Tracing e2e tests are promoted to conformance tests

## Production Readiness Survey

* Feature enablement and rollback
  - How can this feature be enabled / disabled in a live cluster?  **Feature-gate: APIServerTracing.  The API Server must be restarted to enable/disable exporting spans.**
  - Can the feature be disabled once it has been enabled (i.e., can we roll
    back the enablement)?  **Yes, the feature gate can be disabled in the API Server**
  - Will enabling / disabling the feature require downtime for the control
    plane?  **Yes, the API Server must be restarted with the feature-gate disabled.**
  - Will enabling / disabling the feature require downtime or reprovisioning
    of a node?  **No.**
  - What happens if a cluster with this feature enabled is rolled back? What happens if it is subsequently upgraded again?  **Rolling back this feature would break the added tracing telemetry, but would not affect the cluster.**
  - Are there tests for this?  **No.  The feature hasn't been developed yet.**
* Scalability
  - Will enabling / using the feature result in any new API calls? **No.**
    Describe them with their impact keeping in mind the [supported limits][]
    (e.g. 5000 nodes per cluster, 100 pods/s churn) focusing mostly on:
     - components listing and/or watching resources they didn't before
     - API calls that may be triggered by changes of some Kubernetes
       resources (e.g. update object X based on changes of object Y)
     - periodic API calls to reconcile state (e.g. periodic fetching state,
       heartbeats, leader election, etc.)
  - Will enabling / using the feature result in supporting new API types? **No**
    How many objects of that type will be supported (and how that translates
    to limitations for users)?
  - Will enabling / using the feature result in increasing size or count
    of the existing API objects?  **No.**
  - Will enabling / using the feature result in increasing time taken
    by any operations covered by [existing SLIs/SLOs][] (e.g. by adding
    additional work, introducing new steps in between, etc.)? **Yes.  It will increase API Server request latency by a negligible amount (<1 microsecond) for encoding and decoding the trace contex from headers, and recording spans in memory.  Exporting spans is not in the critical path.**
    Please describe the details if so.
  - Will enabling / using the feature result in non-negligible increase
    of resource usage (CPU, RAM, disk IO, ...) in any components?
    Things to keep in mind include: additional in-memory state, additional
    non-trivial computations, excessive access to disks (including increased
    log volume), significant amount of data sent and/or received over
    network, etc. Think through this in both small and large cases, again
    with respect to the [supported limits][].  **The tracing client library has a small, in-memory cache for outgoing spans.**
* Rollout, Upgrade, and Rollback Planning
* Dependencies
  - Does this feature depend on any specific services running in the cluster
    (e.g., a metrics service)? **Yes.  In the current version of the proposal, users can run the [OpenTelemetry Collector](https://github.com/open-telemetry/opentelemetry-collector) to configure which backend (e.g. jager, zipkin, etc.) they want telemetry sent to.**
  - How does this feature respond to complete failures of the services on
    which it depends?  **Traces will stop being exported, and components will store spans in memory until the buffer is full.  After the buffer fills up, spans will be dropped.**
  - How does this feature respond to degraded performance or high error rates
    from services on which it depends? **If the bi-directional grpc streaming connection to the collector cannot be established or is broken, the controller retries the connection every 5 minutes (by default).**
* Monitoring requirements
  - How can an operator determine if the feature is in use by workloads?  **Operators are generally expected to have access to the trace backend.**
  - How can an operator determine if the feature is functioning properly?
  - What are the service level indicators an operator can use to determine the
    health of the service?  **Error rate of sending traces in the API Server and OpenTelemetry collector.**
  - What are reasonable service level objectives for the feature?  **Not entirely sure, but I would expect at least 99% of spans to be sent successfully, if not more.**
* Troubleshooting
  - What are the known failure modes?  **The API Server is misconfigured, and cannot talk to the collector.  The collector is misconfigured, and can't send traces to the backend.**
  - How can those be detected via metrics or logs?  Logs from the component or agent based on the failure mode.
  - What are the mitigations for each of those failure modes?  **None.  You must correctly configure the collector for tracing to work.**
  - What are the most useful log messages and what logging levels do they require? **All errors are useful, and are logged as errors (no logging levels required). Failure to initialize exporters (in both controller and collector), failures exporting metrics are the most useful.  Errors are logged for each failed attempt to establish a connection to the collector.**
  - What steps should be taken if SLOs are not being met to determine the
    problem?  **Look at API Server  and collector logs.**

## Implementation History

* [Mutating admission webhook which injects trace context](https://github.com/Monkeyanator/mutating-trace-admission-controller)
* [Instrumentation of Kubernetes components](https://github.com/Monkeyanator/kubernetes/pull/15)
* [Instrumentation of Kubernetes components for 1/24/2019 community demo](https://github.com/kubernetes/kubernetes/compare/master...dashpole:tracing)
* KEP merged as provisional on 1/8/2020, including controller tracing
* KEP scoped down to only API Server traces on 5/1/2020

## Alternatives considered

### Introducing a new EgressSelector type

Instead of a configuration file to choose between a url on the `Master` network, or a service on the `Cluster` network, we considered introducing a new `OpenTelemetry` egress type, which could be configured separately.  However, we aren't actually introducing a new destination for traffic, so it is more conventional to make use of existing egress types.  We will also likely want to add additional configuration for the OpenTelemetry client in the future.

### Other OpenTelemetry Exporters

This KEP suggests that we utilize the OpenTelemetry exporter format in all components.  Alternative options include:

1. Add configuration for many exporters in-tree by vendoring multiple "supported" exporters. These exporters are the only compatible backends for tracing in kubernetes.
  a. This places the kubernetes community in the position of curating supported tracing backends
2. Support *both* a curated set of in-tree exporters, and the collector exporter
