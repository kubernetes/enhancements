---
title: Leveraging Distributed Tracing to Understand Kubernetes Object Lifecycles
authors:
  - "@Monkeyanator"
editors:
  - "@dashpole"
owning-sig: sig-instrumentation
participating-sigs:
  - sig-architecture
  - sig-node
  - sig-api-machinery
  - sig-scalability
  - sig-cli
reviewers:
  - "@Random-Liu"
  - "@bogdandrutu"
approvers:
  - "@brancz"
  - "@piosz"
creation-date: 2018-12-04
last-updated: 2019-11-07
status: provisional
---

# Leveraging Distributed Tracing to Understand Kubernetes Object Lifecycles

## Table of Contents

- [Leveraging Distributed Tracing to Understand Kubernetes Object Lifecycles](#leveraging-distributed-tracing-to-understand-kubernetes-object-lifecycles)
    - [Table of Contents](#table-of-contents)
    - [Summary](#summary)
    - [Motivation](#motivation)
        - [Definitions](#definitions)
        - [Goals](#goals)
        - [Non-Goals](#non-goals)
    - [Proposal](#proposal)
        - [Architecture](#architecture)
            - [Tracing API Requests](#tracing-api-requests)
            - [Propagating Traces Through Objects](#propagating-context-through-objects)
            - [Controller Behavior](#controller-behavior)
            - [End-User Behavior](#end-user-behavior)
        - [In-tree changes](#in-tree-changes)
            - [Vendor the Tracing Framework](#vendor-the-tracing-framework)
            - [Trace Utility Package](#trace-utility-package)
            - [Tracing Pod Lifecycle](#tracing-pod-lifecycle)
        - [Out-of-tree changes](#out-of-tree-changes)
            - [Tracing best-practices documentation](#tracing-best-practices-documentation)
    - [Graduation Requirements](#graduation-requirements)
    - [Production Readiness Survey](#production-readiness-survey)
    - [Implementation History](#implementation-history)

## Summary

This Kubernetes Enhancement Proposal (KEP) introduces a model for adding distributed tracing to Kubernetes object lifecycles. The inclusion of this trace instrumentation will mark a significant step in making Kubernetes processes more observable, understandable, and debuggable.


## Motivation

Debugging latency issues in Kubernetes is an involved process. There are existing tools which can be used to isolate these issues in Kubernetes, but these methods fall short for various reasons. For instance:

* **Logs**: are fragmented, and finding out which process was the bottleneck involves digging through troves of unstructured text. In addition, logs do not offer higher-level insight into overall system behavior without an extensive background on the process of interest. 
* **Events**: in Kubernetes are only kept for an hour by default, and don't integrate with visualization of analysis tools. To gain trace-like insights would require a large investment in custom tooling.
* **Latency metrics**: can only supply limited metadata because of cardinality constraints.  They are useful for showing _that_ a process was slow, but don't provide insight into _why_ it was slow.
* **Latency Logging**: is a "poor man's" version of tracing that only works within a single binary and outputs log messages.  See [github.com/kubernetes/utils/trace](https://github.com/kubernetes/utils/tree/master/trace).

Distributed tracing provides a single window into latency information from across many components and plugins. Trace data is structured, and there are numerous established backends for visualizing and querying over it.

### Definitions

**Span**: The smallest unit of a trace.  It has a start and end time, and is attached to a single trace.
**Trace**: A collection of Spans which represents a single process.
**Trace Context**: A reference to a Trace that is designed to be propagated across component boundaries.

### Goals

* Make it possible to visualize the progress of objects across distinct Kubernetes components
* Streamline the tedious processes involved in debugging latency issues in Kubernetes
* Make it possible to identify high-level latency regressions, and attribute them back to offending processes


### Non-Goals

* Replace existing logging, metrics, or the events API
* Trace operations from all Kubernetes resource types in a generic manner (i.e. without manual instrumentation)

## Proposal

### Architecture

#### Tracing API Requests

In the traditional tracing model, a client sends a request to a server and recieves a response back.  Even though Kubernetes "controllers" don't follow this model (more on that later), the kube-apierver and backing storage (e.g. etcd3) do.  To enable traces to be collected for API requests, the following must be true:

1. The apiserver must propagate the http context of incoming requests through its function stack to the backing storage
1. Kubernetes client libraries must allow passing a context with API requests

To actually add traces to API requests, owners of the kube-apiserver and backing storage may add Spans to incoming requests, and configure sampling as they see fit.

#### Propagating Context Through Objects

While API requests follow the traditional RPC client-server tracing model, kubernetes controllers don't.  Instead of controller actions being driven by incoming RPCs, their actions are driven by observations of desired and actual state.  This is the primary reason why the kubernetes community hasn't agreed on how to integrate tracing into kubernetes thus far.

In the traditional RPC client-server tracing model, a trace context is attached to a single incoming request, and is propagated with all requests the server makes to other servers required to fulfill the initial single request.  Conceptually, this proposal suggests treating a kubernetes cluster as a single RPC server.  The difference is that we attach context to objects, and propagate this context to objects modified as a result of the initial object modification.  For example, if a user creates a ReplicaSet, the kube-controller-manager will create many Pod objects as a result, and will propagate the context used to create the ReplicaSet to Pod objects as well.  This ensures that all actions taken by kubernetes controllers as a result of the initial user action are linked by the same context.

For the alpha phase, we choose to propagate this span context as an encoded string an object annotation called `trace.kubernetes.io/context`.  As noted in [Tracing API Requests](#tracing-api-requests) above, storing the trace context with the context is _in addition_ to attaching a context to http requests to the apiserver.  The reason for this is explained in the [Controller Behavior](#controller-behavior) section below.  In some scenarios, controllers will want to update the trace context from A -> B, but want to associated that Update request with context A.

This means two trace contexts are sent in different forms with Create/Patch/Update requests to the apiserver.  A trace context is around 32 bytes (16 bytes for the trace ID, 8 bytes for the span ID, and some metadata). See the [w3c spec](https://w3c.github.io/trace-context/#tracestate-field) for details.


This annotation value is removed when an object's trace ends, to achieve the desired behavior from [section one](#trace-lifecycle).  For core kubernetes components, this must be done in the same request to the API Server as the status update which updates the object to its desired state.  This is a requirement to ensure tracing does not affect the scalability of kubernetes.  For other components, it is recommended, but not required to update the trace annotation in the same request.

This proposal chooses to use annotations to store the SpanContext associated with an object.  This mirrors how trace context propagation is done with golang context.Context and http headers, which are both key/value stores.

#### Controller Behavior

When reconciling an object `Foo` a Controller must:

1. Send the trace context stored in `Foo` in the http request context for all API requests. See [Tracing API Requests](#tracing-api-requests)
1. Store the trace context of `Foo` in object `Bar` when updating the Spec of `Bar`. See [Propagating Context Through Objects](#propagating-context-through-objects)
1. Export a span around work that attempts to drive the actual state of an object towards its desired state
1. Replace the trace context of `Foo` when updating `Foo`'s status to the desired state

Controllers must _only_ export Spans around work that it is correcting from an undesired state to its desired state.  To avoid exporting pointless spans, controllers must not export spans around reconciliation loops that do not perform actual work.  For example, the kubelet must not export a span around syncPod, which is a generic Reconcile function.  Instead, it should export spans around CreateContainer, or other functions that move the system towards its desired state. 

This proposal is grounded on the principle that a trace context is attached to and propagated with end-user intent.  When the status of an object is updated to its desired state, the end-user's intent for that object has been fulfilled.  Controllers must "end" tracing for an object when it reaches its desired state.  To accomplish this, Controllers must update the trace context of an object when updating the status of an object from an undesired to a desired state.  For objects that report a status that can reach a desired state, this limits traces to just the actions taken by controllers in the fulfillment of the end-user's intent, and prevents traces from spanning an indefinite period of time.

Components should plumb the context through reconciliation functions, rather than storing and looking up trace contexts globally so that each attempt to reconcile desired and actual state uses the context associated with _that_ desired state through the entire attempt.  If multiple components are involved in reconciling a single object, one may act on the new trace context before the other, but each trace is still representative of the work done to reconcile to the corresponding desired state. Given this model, we guarantee that each trace contains the actions taken to reconcile toward a single desired state.

High-level processes, such as starting a pod or restarting a failed container, could be interrupted before completion by an update to the desired state. While this leaves a "partial" trace for the first process, it is the most accurate representation of the work and timing of reconciling desired and actual state.

#### End-User Behavior

Add a new `--trace` argument to `kubectl`, which generates a new trace context, sets the trace context to be sampled, attaches the context to all modified objects, and uses the context when sending requests to the API Server.  The option is disabled by default.  Note that by attaching a trace context to the initial object creation, this will cause all object modification done by controllers to propagate the context through to all changes made by the system that are driven by the initial user action.

Add `context.Context` arguments to k8s.io/client-go client functions.  This will allow users and components to associate API calls with the context of the involved object.  In some cases, such as object creation, we can automatically attach the SpanContext of the provided context to the created object, making propagation simpler.

This also enables kubernetes to be a composable part of a larger system. For example, if an end-user's service creates a pod as part of handing a request, it could do:
```golang
ctx, span := trace.StartSpan(preexistingContext, “create-my-pod”)
defer span.End()
pod, err := c.CoreV1().Pod(myPod.Namespace).Create(ctx, myPod)
if err != nil {
    return err
}
waitForPodToBeRunning(ctx, myPod)
return nil
```

A previous iteration of this proposal suggested controllers should export a "Root Span" when ending a trace (described in [Controller Behavior](#controller-behavior) above).  However, that would limit a trace to being associated with a single object, since a "Root Span" defines the scope of the trace.  More generally, we shouldn't assume that the creation or update of a single object represents the entirety of end-user intent.  The user or system using kubernetes determines what the user intent is, not kubernetes controllers.

Tracing in a kubernetes cluster must be a composable component within a larger system, and allow external users or systems to define the "Root Span" that defines and bounds the scope of a trace.

### In-tree changes

#### Vendor the Tracing Framework

This KEP proposes the use of the [OpenTelemetry tracing framework](https://opentelemetry.io/) to create and export spans to configured backends.

While in alpha, controllers should use the OpenTelemetry exporter, which exports traces to the [OpenTelemetry Collector](https://github.com/open-telemetry/opentelemetry-collector). The OpenTelemetry collector allows importing and configuring exporters for trace storage backends to be done out-of-tree in addition to other useful features.

This KEP suggests that we utilize the OpenTelemetry collector for the initial implementation to reduce the global changes required for alpha.  Alternative options include:

1. Add configuration for exporters in-tree by vendoring in each "supported" exporter. These exporters are the only compatible backends for tracing in kubernetes.
  a. This places the kubernetes community in the position of curating supported tracing backends
  b. This eliminates the requirement to run to OpenTelemetry collector in order to use tracing
2. Support *both* a curated set of in-tree exporters, and the collector exporter

While this setup is suitable for an alpha stage, it will require further review from Sig-Instrumentation and Sig-Architecture for beta, as it introduces a dependency on the OT Collector.  It is also worth noting that OpenTelemetry still has many unresolved details on how to run the collector.

#### Trace Utility Package

This package will be able to create spans from the span context embedded in the `trace.kubernetes.io/context` object annotation, in addition to embedding context from spans back into the annotation. This package will facilitate propagating traces through kubernetes objects.  The exported functions include:

```golang
// InitializeExporter initializes the trace exporting service with the provided service name.
// Components should use this initializer to ensure common behavior.
func InitializeExporter(service string)

// StartSpanFromObject constructs a new Span using the context attached to the object as the parent SpanContext.  It mirrors trace.StartSpan, but for kubernetes objects.
func StartSpanFromObject(ctx context.Context, obj meta.Object, spanName string) (context.Context, *trace.Span, error)

// EncodeContextIntoObject encodes the SpanContext contained in the context into the object
func EncodeContextIntoObject(ctx context.Context, obj meta.Object)

// RemoveSpanContextFromObject removes the SpanContext attached to an object, if one exists
func RemoveSpanContextFromObject(obj meta.Object) 
```

#### Tracing Pod Lifecycle

As we move forward with this KEP, we will use the aforementioned trace utility package to trace pod-related operations across the scheduler and kubelet. In code, this corresponds to creating a span (i.e. `ctx, span := trace.StartSpan(ctx, "Component.SampleSpan")`) at the beginning of an operation, and ending the span afterwards (`span.End()`). All calls to tracing functions will be gated with the `ObjectLifecycleTracing` alpha feature-gate, and will be disabled by default.

OpenTelemetry ships with plugins to transport trace context across gRPC and HTTP boundaries, which enables us to extend our tracing across the CRI and other internal boundaries.

In OpenTelemetry's Go implementation, span context is passed down through Go context. This will necessitate the threading of context across more of the Kubernetes codebase, which is a [desired outcome regardless](https://github.com/kubernetes/kubernetes/issues/815).

While adding tracing to Pods is a good first step to demonstrate the viability of object lifecycle tracing in kubernetes, we expect component owners to add tracing to their components in an ad-hoc fashion.

### Out-of-tree changes

#### Tracing best-practices documentation

This KEP introduces a new form of instrumentation to Kubernetes, which necessiates the creation of guidelines for adding effective, standardized traces to Kubernetes components, [similar to what is found here for metrics](https://github.com/kubernetes/community/blob/master/contributors/devel/instrumentation.md).

This documentation will put forward standards for: 

* How to name spans, attributes, and annotations
* What kind of processes should be wrapped in a span
* When to link spans to other traces
* What kind of data makes for useful attributes
* How to propagate trace context as proposed above

Having these standards in place will ensure that our tracing instrumentation works well with all backends, and that reviewers have concrete criteria to cross-check PRs against. 

## Graduation requirements

Alpha

- [] Alpha-implementation as described above
- [] E2e testing of traces
- [] User-facing documentation
- [] Tracing must not increase the number of requests to the APIServer

Beta

- [] Security Review, including threat model
- [] Deployment review including whether the [OT Collector](https://github.com/open-telemetry/opentelemetry-collector) is a required component
- [] Benchmark kubernetes components using tracing, and determine resource requirements and scaling for any additional required components (e.g. OT Collector).

GA

- [] Versioning for span naming and backwards-compatibility guarantees

## Production Readiness Survey

* Feature enablement and rollback
  - How can this feature be enabled / disabled in a live cluster?  **Feature-gate: ComponentTracing.  All components that are instrumented with tracing must be restarted to enable/disable exporting spans from that component.  Initial components that will emit spans are: kube-apiserver, kube-scheduler, kube-controller-manager, kubelet.  Others may be added in later stages.**
  - Can the feature be disabled once it has been enabled (i.e., can we roll
    back the enablement)?  **Yes, the feature gate can be disabled in all relevant components**
  - Will enabling / disabling the feature require downtime for the control
    plane?  **Yes, control-plane components must be restarted with the feature-gate disabled.**
  - Will enabling / disabling the feature require downtime or reprovisioning
    of a node?  **No, it just requires restarting the kubelet with the feature-gate disabled**
  - What happens if a cluster with this feature enabled is rolled back? What happens if it is subsequently upgraded again?  **No adverse effects in either case.**
  - Are there tests for this?  **No.  The feature hasn't been developed yet.**
* Scalability
  - Will enabling / using the feature result in any new API calls? **No (there was a recent change in the KEP to that effect).**
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
    of the existing API objects?  **Yes.  It adds an annotation to "traced" objects.  The value is a trace context, which is ~32 bytes.  Traced objects will initially include pods, replicasets, and deployments, but may expand to include others over time.  Notably, this annotation should not be added to Events.**
  - Will enabling / using the feature result in increasing time taken
    by any operations covered by [existing SLIs/SLOs][] (e.g. by adding
    additional work, introducing new steps in between, etc.)? **No**
    Please describe the details if so.
  - Will enabling / using the feature result in non-negligible increase
    of resource usage (CPU, RAM, disk IO, ...) in any components?
    Things to keep in mind include: additional in-memory state, additional
    non-trivial computations, excessive access to disks (including increased
    log volume), significant amount of data sent and/or received over
    network, etc. Think through this in both small and large cases, again
    with respect to the [supported limits][].  **The tracing client library has an in-memory cache for outgoing spans.  I believe this is limited to 1Mb by default.  This overhead would apply to all controllers that export spans.  Note that this applies to the kubelet as well, since the kubelet is one of the initial components that will be instrumented.**
* Rollout, Upgrade, and Rollback Planning
* Dependencies
  - Does this feature depend on any specific services running in the cluster
    (e.g., a metrics service)? **Yes.  In the current version of the proposal, users must run the [OpenTelemetry Collector](https://github.com/open-telemetry/opentelemetry-collector) as a daemonset to configure which backend (e.g. jager, zipkin, etc.) they want telemetry sent to.**
  - How does this feature respond to complete failures of the services on
    which it depends?  **Traces will stop being exported, and components will store spans in memory until the buffer is full.  After the buffer fills up, spans will be dropped.**
  - How does this feature respond to degraded performance or high error rates
    from services on which it depends? **If the bi-directional grpc streaming connection to the collector cannot be established or is broken, the controller retries the connection every 5 minutes (by default).**
* Monitoring requirements
  - How can an operator determine if the feature is in use by workloads?  **The operator can check for the presence of the trace context annotation.  Generally, operators are expected to have access to (and likely control over) the OpenTelemetry agent deployment and trace storage backend.**
  - How can an operator determine if the feature is functioning properly?  **TODO: does the client library add metrics about trace exporting?**
  - What are the service level indicators an operator can use to determine the
    health of the service?  **Error rate of sending traces in controllers and OpenTelemetry collector.**
  - What are reasonable service level objectives for the feature?  **Not entirely sure, but I would expect at least 99% of spans to be sent successfully, if not more.**
* Troubleshooting
  - What are the known failure modes?  **The controller is misconfigured, and cannot talk to the collector.  The collector is misconfigured, and can't send traces to the backend.**
  - How can those be detected via metrics or logs?  Logs from the component or agent based on the failure mode.
  - What are the mitigations for each of those failure modes?  **None.  You must correctly configure the collector for tracing to work.**
  - What are the most useful log messages and what logging levels do they require? **All errors are useful, and are logged as errors (no logging levels required). Failure to initialize exporters (in both controller and collector), failures exporting metrics are the most useful.**
  - What steps should be taken if SLOs are not being met to determine the
    problem?  **Look at controller and collector logs.**

## Implementation History

* [Mutating admission webhook which injects trace context](https://github.com/Monkeyanator/mutating-trace-admission-controller)
* [Instrumentation of Kubernetes components](https://github.com/Monkeyanator/kubernetes/pull/15)
* [Instrumentation of Kubernetes components for 1/24/2019 community demo](https://github.com/kubernetes/kubernetes/compare/master...dashpole:tracing)
