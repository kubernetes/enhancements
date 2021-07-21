# KEP-2831: Kubelet Tracing

<!-- toc -->
- [Release Signoff Checklist](#release-signoff-checklist)
- [Summary](#summary)
- [Motivation](#motivation)
  - [Definitions](#definitions)
  - [Goals](#goals)
  - [Non-Goals](#non-goals)
- [Proposal](#proposal)
  - [User Stories](#user-stories)
    - [Continuous Trace Collection](#continuous-trace-collection)
      - [Example Scenarios](#example-scenarios)
  - [Tracing Requests and Exporting Spans](#tracing-requests-and-exporting-spans)
  - [Running the OpenTelemetry Collector](#running-the-opentelemetry-collector)
  - [Kubelet Configuration](#kubelet-configuration)
- [Design Details](#design-details)
  - [Test Plan](#test-plan)
  - [Graduation Requirements](#graduation-requirements)
- [Production Readiness Review Questionnaire](#production-readiness-review-questionnaire)
  - [Feature Enablement and Rollback](#feature-enablement-and-rollback)
      - [Does enabling the feature change any default behavior?](#does-enabling-the-feature-change-any-default-behavior)
      - [Can the feature be disabled once it has been enabled (i.e. can we roll back the enablement)?](#can-the-feature-be-disabled-once-it-has-been-enabled-ie-can-we-roll-back-the-enablement)
      - [Are there any tests for feature enablement/disablement?](#are-there-any-tests-for-feature-enablementdisablement)
  - [Rollout, Upgrade and Rollback Planning](#rollout-upgrade-and-rollback-planning)
  - [Monitoring Requirements](#monitoring-requirements)
      - [What are the SLIs (Service Level Indicators) an operator can use to determine the health of the service?](#what-are-the-slis-service-level-indicators-an-operator-can-use-to-determine-the-health-of-the-service)
      - [Are there any missing metrics that would be useful to have to improve observability](#are-there-any-missing-metrics-that-would-be-useful-to-have-to-improve-observability)
  - [Dependencies](#dependencies)
  - [Scalability](#scalability)
  - [Troubleshooting](#troubleshooting)
- [Implementation History](#implementation-history)
- [Drawbacks](#drawbacks)
- [Alternatives](#alternatives)
  - [Other OpenTelemetry Exporters](#other-opentelemetry-exporters)
<!-- /toc -->

## Release Signoff Checklist

Items marked with (R) are required *prior to targeting to a milestone / release*.

- [ ] (R) Enhancement issue in release milestone, which links to KEP dir in [kubernetes/enhancements] (not the initial KEP PR)
- [ ] (R) KEP approvers have approved the KEP status as `implementable`
- [ ] (R) Design details are appropriately documented
- [ ] (R) Test plan is in place, giving consideration to SIG Architecture and SIG Testing input
- [ ] (R) Graduation criteria is in place
- [ ] (R) Production readiness review completed
- [ ] Production readiness review approved
- [ ] "Implementation History" section is up-to-date for milestone
- [ ] User-facing documentation has been created in [kubernetes/website], for publication to [kubernetes.io]
- [ ] Supporting documentation—e.g., additional design documents, links to mailing list discussions/SIG meetings, relevant PRs/issues, release notes

## Summary

This Kubernetes Enhancement Proposal (KEP) is to enhance the kubelet to allow tracing gRPC and HTTP API requests.
The kubelet is the integration point of a node's operating system and kubernetes. From [control plane-node communication documentation](https://kubernetes.io/docs/concepts/architecture/control-plane-node-communication/#control-plane-to-node),
a primary communication path from the control plane to the nodes is from the apiserver to the kubelet process running on each node.
The kubelet communicates with the container runtime over gRPC, where kubelet is the gRPC client and the Container Runtime Interface is the gRPC server.
The CRI then sends the creation request to the container runtime installed on the node.
Traces gathered from the kubelet will provide critical information to monitor and troubleshoot interactions at the node level.
This KEP proposes using OpenTelemetry libraries, and exports in the OpenTelemetry format.
This is in line with the [API Server enhancement](https://github.com/kubernetes/enhancements/tree/master/keps/sig-instrumentation/647-apiserver-tracing).

## Motivation

Along with metrics and logs, traces are a useful form of telemetry to aid with debugging incoming requests.
The kubelet can make use of distributed tracing to improve the ease of use and enable easier analysis of trace data.
Trace data is structured, providing the detail necessary to debug requests across service boundaries.
As more core components are instrumented, Kubernetes becomes easier to monitor, manage, and troubleshoot.

### Definitions

**Span**: The smallest unit of a trace.  It has a start and end time. Spans are the building blocks of a trace.    
**Trace**: A collection of Spans which represents work being done by a system. A record of the path of requests through a system.    
**Trace Context**: A reference to a Trace that is designed to be propagated across component boundaries.    

### Goals

* The kubelet generates and exports spans for reconcile loops it initiates and for incoming/outgoing requests to the kubelet's authenticated
http servers, as well as the CRI, CNI, and CSI interfaces. An example of a reconcile loop is the creation of a pod. Pod creation involves
pulling the image, creating the pod sandbox and creating the container. With stateful workloads, attachment and mounting of volumes referred by a pod
might be included. Trace data can be used to diagnose latency within such flows.

### Non-Goals

* Tracing in other kubernetes controllers besides the kubelet
* Replace existing logging, metrics
* Change metrics or logging (e.g. to support trace-metric correlation)
* Access control to tracing backends
* Add tracing to components outside kubernetes (e.g. etcd client library).

## Proposal

### User Stories

Since this feature is for diagnosing problems with the kubelet, it is targeted at cluster operators, administrators, and cloud vendors that 
manage kubernetes nodes and core services.

For the following use-cases, I can deploy an OpenTelemetry collector agent as a DaemonSet to collect kubelet trace data from each node's host network.
From there, OpenTelemetry trace data can be exported to a tracing backend of choice.  I can configure the kubelet's optional `TracingConfiguration`
specified within `kubernetes/component-base` with the default URL to make the kubelet send its spans to the collector.
Alternatively, I can point the kubelet at an OpenTelemetry collector listening on a different port or URL if I need to.

#### Continuous Trace Collection

As a cluster administrator or cloud provider, I would like to collect gRPC and HTTP trace data from the transactions between the API server and the 
kubelet and interactions with a node's container runtime (Container Runtime Interface) to debug cluster problems.  I can set the `SamplingRatePerMillion`
in the configuration file to a non-zero number to have spans collected for a small fraction of requests. Depending on the symptoms I need to
debug, I can search span metadata or specific nodes to find a trace which displays the symptoms I am looking to debug.
The sampling rate for trace exports can be configured based on my needs. I can collect each node's kubelet trace data as distinct tracing services
to diagnose node issues.

##### Example Scenarios

* Latency or timeout experienced when:
    * Attach or exec to running containers
        * APIServer to Kubelet, then Kubelet to CRI can determine which component/operation is the culprit
    * Kubelet's port-forwarding functionality
        * APIServer, Kubelet, CRI trace data can determine which component/operation is experiencing issues
    * Container Start/Stop/Create/Remove
        * Kubelet, CRI trace data determines which component/operation is experiencing issues with insight into which is experiencing issues.
    * Attachment and mounting of volumes referred by a pod
        * Trace data from kubelet's interaction with volume plugins will help to quickly diagnose latency issues.
* Kubelet spans can detect node-level latency - each node produces trace data filterable with node hostname
    * Are there issues with particular regions? Cloud providers? Machine-types?

### Tracing Requests and Exporting Spans

The gRPC client in the kubelet's Container Runtime Interface (CRI) [Remote Runtime Service](https://github.com/kubernetes/kubernetes/blob/release-1.21/pkg/kubelet/cri/remote/remote_runtime.go) and [the CRI streaming package](https://github.com/kubernetes/kubernetes/tree/release-1.21/pkg/kubelet/cri/streaming) will be instrumented to export trace data and propagate trace context.
The [Go implementation of OpenTelemetry](https://github.com/open-telemetry/opentelemetry-go) will be used.
An [OTLP exporter](https://github.com/open-telemetry/opentelemetry-go/blob/main/exporters/otlp/otlptrace/otlptracegrpc/exporter.go) and
an [OTLP trace provider](https://github.com/open-telemetry/opentelemetry-go/blob/main/sdk/trace/provider.go) along with a [context propagator](https://opentelemetry.io/docs/go/instrumentation/#propagators-and-context) will be configured.

Also, designated http servers and clients will be wrapped with [otelhttp](https://github.com/open-telemetry/opentelemetry-go-contrib/tree/v0.21.0/instrumentation/net/http)
to generate spans for sampled incoming requests and propagate context with client requests.

OpenTelemetry-Go provides the [propagation package](https://github.com/open-telemetry/opentelemetry-go/blob/main/propagation/propagation.go) with which you can add custom key-value pairs known as [baggage](https://github.com/open-telemetry/opentelemetry-specification/blob/main/specification/baggage/api.md). Baggage data will be propagated across services within contexts.

### Running the OpenTelemetry Collector

Although this proposal focuses on running the [OpenTelemetry Collector](https://github.com/open-telemetry/opentelemetry-collector), note that any
[OTLP-compatible](https://github.com/open-telemetry/opentelemetry-specification/blob/main/specification/protocol/otlp.md)
trace collection pipeline can be used to collect OTLP data from the kubelet. As an example, the OpenTelemetry Collector can be run as a sidecar, a daemonset,
a deployment , or a combination in which the daemonset buffers telemetry and forwards to the deployment for aggregation (e.g. tail-base sampling) and routing to
a telemetry backend.  To support these various setups, the kubelet should be able to send traffic either to a local (on the control plane network) collector,
or to a cluster service (on the cluster network).

### Kubelet Configuration

Refer to [OpenTelemetry Specification](https://github.com/open-telemetry/opentelemetry-specification/blob/ccbfdd9020ba0d581aa5d880235302f8b0eb8669/specification/resource/semantic_conventions/README.md#service)

```golang
// KubeletConfiguration contains the configuration for the Kubelet
type KubeletConfiguration struct {
    metav1.TypeMeta `json:",inline"`
    ------------

    // +optional
    // TracingConfiguration
    TracingConfiguration componentbaseconfig.TracingConfiguration
}

```

**NOTE** The TracingConfiguration will be defined in `kubernetes/component-base` repository
```golang
// TracingConfiguration specifies configuration for opentelemetry tracing
type TracingConfiguration struct {
    // +optional
    // URL of the collector that's running on the node.
    // Defaults to 0.0.0.0:4317
    URL  *string `json:"url,omitempty" protobuf:"bytes,1,opt,name=url"`

    // +optional
    // SamplingRatePerMillion is the number of samples to collect per million spans.
    // Defaults to 0.
    SamplingRatePerMillion int32 `json:"samplingRatePerMillion,omitempty" protobuf:"varint,2,opt,name=samplingRatePerMillion"`
}

```

## Design Details

### Test Plan

We will test tracing added by this feature with an integration test.  The
integration test will verify that spans exported by the kubelet match what is
expected from the request. We will also add an integration test that verifies
spans propagated from kubelet to API server match what is expected from the request.

### Graduation Requirements

Alpha

- [] Implement tracing of incoming and outgoing gRPC, HTTP requests in the kubelet
- [] Integration testing of tracing

Beta

- [] Publish examples of how to use the OT Collector with kubernetes
- [] Allow time for feedback
- [] Revisit the format used to export spans.

GA

## Production Readiness Review Questionnaire

### Feature Enablement and Rollback

###### How can this feature be enabled / disabled in a live cluster?
  - [X] Feature gate (also fill in values in `kep.yaml`)
    - Feature gate name: KubeletTracing
    - Components depending on the feature gate: kubelet
  - [X] Other
    - Describe the mechanism: **KubeletConfiguration TracingConfiguration**
      - To disable tracing entirely, do not enable the feature gate `KubeletTracing` and/or do not provide a TracingConfiguration.
        Tracing will be disabled unless a TracingConfiguration is provided.
      - If disabled (no TracingConfiguration is provided), trace contexts will be propagated in passthrough mode. No spans will
        be generated and no attempt will be made to connect via otlp protocol to export traces. More about passthrough mode in
        [OpenTelemetry documentation](https://github.com/open-telemetry/opentelemetry-go/tree/main/example/passthrough#passthrough-setup-for-opentelemetry), `the default TracerProvider implementation returns a 'Non-Recording' span that keeps the context of the caller but does not record spans`.
      - If feature is enabled and TracingConfiguration sampling rate is 0 (the default):
        - a trace context with sampled=true will still cause traces to be generated.
        - kubelet will attempt to connect via otlp protocol to export traces.
    - Will enabling / disabling the feature require downtime of the control
      plane?  **No. It will require restarting the kubelet service per node.**
    - Will enabling / disabling the feature require downtime or reprovisioning
      of a node? **No, restarting the kubelet with feature-gate disabled will disable tracing**

##### Does enabling the feature change any default behavior?
  No. The feature is disabled unlesss the feature gate is enabled and the TracingConfiguration is populated in Kubelet Configuration.
  When the feature is enabled, it doesn't change behavior from the users' perspective; it only adds tracing telemetry.

##### Can the feature be disabled once it has been enabled (i.e. can we roll back the enablement)?
  Yes.

###### What happens if we reenable the feature if it was previously rolled back?
  It will start generating and exporting traces again.

##### Are there any tests for feature enablement/disablement?
  Unit tests switching feature gates will be added. Manual testing of disabling, reenabling the feature on nodes, ensuring the kubelet comes up w/out error will
  also be performed.

### Rollout, Upgrade and Rollback Planning

_This section must be completed when targeting beta graduation to a release._

###### How can a rollout or rollback fail? Can it impact already running workloads?
  With an improper TracingConfiguration spans will not be exported as expected,
  No impact to running workloads, logs will indicate the problem.

###### What specific metrics should inform a rollback?
  To be determined.

###### Were upgrade and rollback tested? Was the upgrade->downgrade->upgrade path tested?
  Upgrades and rollbacks will be tested while feature-gate is experimental

###### Is the rollout accompanied by any deprecations and/or removals of features, APIs, fields of API types, flags, etc.?
  No

### Monitoring Requirements

_This section must be completed when targeting beta graduation to a release._

###### How can an operator determine if the feature is in use by workloads?

  Operators are expected to have access to and/or control of the OpenTelemetry agent deployment and trace storage backend.
  KubeletConfiguration will show the FeatureGate and TracingConfiguration.

###### How can someone using this feature know that it is working for their instance?

  The tracing backend will display the traces with a service "kubelet".

###### What are the reasonable SLOs (Service Level Objectives) for the enhancement?

  99% expected spans are collected from kubelet
  No indications in logs of failing to export spans
  
##### What are the SLIs (Service Level Indicators) an operator can use to determine the health of the service?

- [] Metrics
  - Metric name: tbd
  - Components exposing the metric: kubelet

##### Are there any missing metrics that would be useful to have to improve observability 
  To be determined.


### Dependencies

_This section must be completed when targeting beta graduation to a release._

###### Does this feature depend on any specific services running in the cluster?**

  Yes.  In the current version of the proposal, users must run the [OpenTelemetry Collector](https://github.com/open-telemetry/opentelemetry-collector)
  as a daemonset and configure a backend trace visualization tool (jaeger, zipkin, etc).


### Scalability

_For alpha, this section is encouraged: reviewers should consider these questions
and attempt to answer them._

_For beta, this section is required: reviewers must answer these questions._

_For GA, this section is required: approvers should be able to confirm the
previous answers based on experience in the field._

###### Will enabling / using this feature result in any new API calls?

  This will not add any additional API calls.

###### Will enabling / using this feature result in introducing new API types?

  This will introduce an API type for the configuration.  This is only for
  loading configuration, users cannot create these objects.

###### Will enabling / using this feature result in any new calls to the cloud provider?

  Not directly.  Cloud providers could choose to send traces to their managed
  trace backends, but this requires them to set up a telemetry pipeline as
  described above.

###### Will enabling / using this feature result in increasing size or count of the existing API objects?

  No.

###### Will enabling / using this feature result in increasing time taken by any operations covered by [existing SLIs/SLOs]?

  It will increase kubelet request latency by a negligible amount (<1 microsecond)
  for encoding and decoding the trace contex from headers, and recording spans
  in memory. Exporting spans is not in the critical path.

###### Will enabling / using this feature result in non-negligible increase of resource usage (CPU, RAM, disk, IO, ...) in any components?

  The tracing client library has a small, in-memory cache for outgoing spans.

### Troubleshooting

The Troubleshooting section currently serves the `Playbook` role. We may consider
splitting it into a dedicated `Playbook` document (potentially with some monitoring
details). For now, we leave it here.

_This section must be completed when targeting beta graduation to a release._

###### How does this feature react if the API server and/or etcd is unavailable?

  No reaction specific to this feature if API server and/or etcd is unavailable.

###### What are other known failure modes?

  - [The controller is misconfigured and cannot talk to the collector or the collector cannot send traces to the backend]
    - Detection: How can it be detected via metrics? Stated another way:
      how can an operator troubleshoot without logging into a master or worker node?
      **kubelet logs, component logs, collector logs**
    - Mitigations: **Disable KubeletTracing, update collector, backend configuration** 
    - Diagnostics: What are the useful log messages and their required logging
      levels that could help debug the issue? **go-opentelemetry sdk provides logs indicating failure**
    - Testing: To be added.

## Implementation History

## Drawbacks

  Small overhead of increased kubelet request latency, will be monitored during experimental phase.

## Alternatives

### Other OpenTelemetry Exporters

This KEP suggests that we utilize the OpenTelemetry exporter format in all components.  Alternative options include:

1. Add configuration for many exporters in-tree by vendoring multiple "supported" exporters. These exporters are the only compatible backends for tracing in kubernetes.
  a. This places the kubernetes community in the position of curating supported tracing backends
2. Support *both* a curated set of in-tree exporters, and the collector exporter
