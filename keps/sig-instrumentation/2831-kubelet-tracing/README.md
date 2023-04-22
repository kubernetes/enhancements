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
    - [Continuous trace collection](#continuous-trace-collection)
      - [Example scenarios](#example-scenarios)
  - [Tracing Requests and Exporting Spans](#tracing-requests-and-exporting-spans)
  - [Connected Traces with Nested Spans](#connected-traces-with-nested-spans)
  - [Running the OpenTelemetry Collector](#running-the-opentelemetry-collector)
  - [Kubelet Configuration](#kubelet-configuration)
- [Design Details](#design-details)
  - [Test Plan](#test-plan)
      - [Prerequisite testing updates](#prerequisite-testing-updates)
      - [Unit tests](#unit-tests)
      - [Integration tests](#integration-tests)
      - [e2e tests](#e2e-tests)
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

- [X] (R) Enhancement issue in release milestone, which links to KEP dir in [kubernetes/enhancements] (not the initial KEP PR)
- [X] (R) KEP approvers have approved the KEP status as `implementable`
- [X] (R) Design details are appropriately documented
- [X] (R) Test plan is in place, giving consideration to SIG Architecture and SIG Testing input
- [X] (R) Graduation criteria is in place
- [X] (R) Production readiness review completed
- [X] Production readiness review approved
- [X] "Implementation History" section is up-to-date for milestone
- [X] User-facing documentation has been created in [kubernetes/website], for publication to [kubernetes.io]
- [X] Supporting documentation—e.g., additional design documents, links to mailing list discussions/SIG meetings, relevant PRs/issues, release notes

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

#### Continuous trace collection

As a cluster administrator or cloud provider, I would like to collect gRPC and HTTP trace data from the transactions between the API server and the 
kubelet and interactions with a node's container runtime (Container Runtime Interface) to debug cluster problems.  I can set the `SamplingRatePerMillion`
in the configuration file to a non-zero number to have spans collected for a small fraction of requests. Depending on the symptoms I need to
debug, I can search span metadata or specific nodes to find a trace which displays the symptoms I am looking to debug.
The sampling rate for trace exports can be configured based on my needs. I can collect each node's kubelet trace data as distinct tracing services
to diagnose node issues.

##### Example scenarios

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

### Connected Traces with Nested Spans

With the initial implementation of this proposal, kubelet tracing produced disconnected spans, because context was not wired through kubelet CRI calls.
With [this PR](https://github.com/kubernetes/kubernetes/pull/113591), context is now plumbed between CRI calls and kubelet.
It is now possible to connect spans for CRI calls. Nested spans with top-level traces in the kubelet will connect CRI calls together.
Nested spans will be created for the following:
* Sync Loops (e.g. syncPod, eviction manager, various gc routines) where the kubelet initiates new work.
    * [top-level traces for pod sync and GC](https://github.com/kubernetes/kubernetes/pull/114504)
* Incoming requests (exec, attach, port-forward, metrics endpoints, podresources)
* Outgoing requests (CNI, CSI, device plugin, k8s API calls)

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

[x] I/we understand the owners of the involved components may require updates to
existing tests to make this code solid enough prior to committing the changes necessary
to implement this enhancement.

##### Prerequisite testing updates

An integration test will verify that spans exported by the kubelet match what is
expected from the request. We will also add an integration test that verifies
spans propagated from kubelet to API server match what is expected from the request.

##### Unit tests

- https://github.com/kubernetes/kubernetes/blob/master/pkg/kubelet/apis/config/validation/validation_test.go#L503-#L532
- https://github.com/kubernetes/kubernetes/blob/master/pkg/kubelet/cri/remote/remote_runtime_test.go#L65-#L97
- https://github.com/kubernetes/kubernetes/blob/master/staging/src/k8s.io/apiserver/pkg/server/options/tracing_test.go
- https://github.com/kubernetes/kubernetes/blob/master/staging/src/k8s.io/component-base/tracing/api/v1/config_test.go

##### Integration tests

Integration tests verify that spans exported by the kubelet match what is
expected from the request. Also an integration test that verifies
spans propagated from kubelet to API server match what is expected from the request.

- _component-base tracing/api/v1 integration test_ https://github.com/kubernetes/kubernetes/blob/master/test/integration/apiserver/tracing/tracing_test.go

##### e2e tests

- A test with kubelet-tracing & apiserver-tracing enabled to ensure no issues are introduced, regardless
of whether a tracing backend is configured.

### Graduation Requirements

Alpha

- [X] Implement tracing of incoming and outgoing gRPC, HTTP requests in the kubelet
- [X] Integration testing of tracing
- [X] Unit testing of kubelet tracing and tracing configuration

Beta

- [X] OpenTelemetry reaches GA
- [X] Publish examples of how to use the OT Collector with kubernetes
- [X] Allow time for feedback
- [ ] Test and document results of upgrade and rollback while feature-gate is enabled.
- [ ] Add top level traces to connect spans in sync loops, incoming requests, and outgoing requests.
- [ ] Unit/integration test to verify connected traces in kubelet.
- [ ] Revisit the format used to export spans.
- [ ] Parity with the old text-based Traces
- [ ] Connecting traces from container runtimes via the Container Runtime Interface
  - https://github.com/kubernetes/kubernetes/pull/114504

GA

## Production Readiness Review Questionnaire

### Feature Enablement and Rollback

###### How can this feature be enabled / disabled in a live cluster?
  - [X] Feature gate (also fill in values in `kep.yaml`)
    - Feature gate name: KubeletTracing
    - Components depending on the feature gate: kubelet
  - [X] Other
    - Describe the mechanism: **KubeletConfiguration TracingConfiguration**
      - When the `KubeletTracing` feature gate is disabled, the kubelet will:
        - Not generate spans
        - Not initiate an OTLP connection
        - Not Propagate context
      - When the feature gate is enabled, but no TracingConfiguration is provided, the kubelet will:
        - Not generate spans
        - Not initiate an OTLP connection
        - Propagate context in (https://github.com/open-telemetry/opentelemetry-go/tree/main/example/passthrough#passthrough-setup-for-opentelemetry)[passthrough] mode
      - When the feature gate is enabled, and a TracingConfiguration with sampling rate 0 (the default) is provided, the kubelet will:
        - Initiate an OTLP connection
        - Generate spans only if the incoming (if applicable) trace context has the sampled flag set
        - Propagate context normally
      - When the feature gate is enabled, and a TracingConfiguration with sampling rate > 0 is provided, the kubelet will:
        - Initiate an OTLP connection
        - Generate spans at the specified sampling rate, or if the incoming context has the sampled flag set
        - Propagate context normally
    - Will enabling / disabling the feature require downtime of the control
      plane?  **No. It will require restarting the kubelet service per node.**
    - Will enabling / disabling the feature require downtime or reprovisioning
      of a node? **No, restarting the kubelet with feature-gate disabled will disable tracing**

##### Does enabling the feature change any default behavior?
  No. The feature is disabled unless the feature gate is enabled and the TracingConfiguration is populated in Kubelet Configuration.
  When the feature is enabled, it doesn't change behavior from the users' perspective; it only adds tracing telemetry.

##### Can the feature be disabled once it has been enabled (i.e. can we roll back the enablement)?
  Yes.

###### What happens if we reenable the feature if it was previously rolled back?
  It will start generating and exporting traces again.

##### Are there any tests for feature enablement/disablement?
  Enabling and disabling kubelet tracing is an in-memory switch. Explicit enablement/disablement tests will not provide value so will not be added.
  Manual testing of disabling, reenabling the feature on nodes, ensuring the kubelet comes up w/out error will be performed and documented.

### Rollout, Upgrade and Rollback Planning

_This section must be completed when targeting beta graduation to a release._

###### How can a rollout or rollback fail? Can it impact already running workloads?
  With an improper TracingConfiguration spans will not be exported as expected,
  No impact to running workloads, logs will indicate the problem.

###### What specific metrics should inform a rollback?

  * This KEP is following the [opentelemetry-go issue #2547](https://github.com/open-telemetry/opentelemetry-go/issues/2547).

  ```
  ...using the OTLP trace exporter, it isn't currently possible to monitor (with metrics) whether or not spans are being successfully collected and exported.
  For example, if my SDK cannot connect to an opentelemetry collector, and isn't able to send traces, I would like to be able to measure how many traces are collected,
  vs how many are not sent. I would like to be able to set up SLOs to measure successful trace delivery from my applications.
  ```

  * Pod Lifecycle and Kubelet [SLOs](https://github.com/kubernetes/community/tree/master/sig-scalability/slos) are the signals that should guide a rollback.  In particular, the [`kubelet_pod_start_duration_seconds_count`, `kubelet_runtime_operations_errors_total`, and `kubelet_pleg_relist_interval_seconds_bucket`] would surface issues affecting kubelet performance.


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
  - Metric name: tbd [opentelemetry-go issue #2547](https://github.com/open-telemetry/opentelemetry-go/issues/2547)
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

- 2021-07-20: KEP opened
- 2022-07-22: KEP merged, targeted at Alpha in 1.24
- 2022-03-29: KEP deemed not ready for Alpha in 1.24
- 2022-06-09: KEP targeted at Alpha in 1.25
- 2023-01-09: KEP targeted at Beta in 1.27

## Drawbacks

  Small overhead of increased kubelet request latency, will be monitored during experimental phase.

## Alternatives

### Other OpenTelemetry Exporters

This KEP suggests that we utilize the OpenTelemetry exporter format in all components.  Alternative options include:

1. Add configuration for many exporters in-tree by vendoring multiple "supported" exporters. These exporters are the only compatible backends for tracing in kubernetes.
  a. This places the kubernetes community in the position of curating supported tracing backends
2. Support *both* a curated set of in-tree exporters, and the collector exporter
