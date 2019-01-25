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
reviewers:
  - "@Random-Liu"
  - "@bogdandrutu"
approvers:
  - "@brancz"
  - "@piosz"
creation-date: 2018-12-04
last-updated: 2019-01-25
status: provisional
---

# Leveraging Distributed Tracing to Understand Kubernetes Object Lifecycles

## Table of Contents

- [Leveraging Distributed Tracing to Understand Kubernetes Object Lifecycles](#leveraging-distributed-tracing-to-understand-kubernetes-object-lifecycles)
    - [Table of Contents](#table-of-contents)
    - [Summary](#summary)
    - [Motivation](#motivation)
        - [Goals](#goals)
        - [Non-Goals](#non-goals)
    - [Proposal](#proposal)
        - [Architecture](#architecture)
            - [Trace lifecycle](#trace-lifecycle)
            - [Root spans](#root-spans)          
            - [Context propagation](#context-propagation)          
        - [Out-of-tree changes](#out-of-tree-changes)
            - [Mutating admission webhook](#mutating-admission-webhook)
            - [Tracing best-practices documentation](#tracing-best-practices-documentation)
        - [In-tree changes](#in-tree-changes)
            - [Trace utility package](#trace-utility-package)
            - [Basic Kubernetes component trace instrumentation](#basic-kubernetes-component-trace-instrumentation)
    - [Graduation Requirements](#graduation-requirements)
    - [Implementation History](#implementation-history)

## Summary

This Kubernetes Enhancement Proposal (KEP) introduces a model for adding distributed tracing to Kubernetes object lifecycles. The inclusion of this trace instrumentation will mark a significant step in making Kubernetes processes more observable, understandable, and debuggable.


## Motivation

Debugging latency issues in Kubernetes is an involved process. There are existing tools which can be used to isolate these issues in Kubernetes, but these methods fall short for various reasons. For instance:

* **Logs**: are fragmented, and finding out which process was the bottleneck involves digging through troves of unstructured text. In addition, logs do not offer higher-level insight into overall system behavior without an extensive background on the process of interest. 
* **Events**: in Kubernetes are only kept for an hour by default, and don't integrate with visualization of analysis tools. To gain trace-like insights would require a large investment in custom tooling.
* **Latency metrics**: can only supply limited metadata because of cardinality constraints.  They are useful for showing _that_ a process was slow, but don't provide insight into _why_ it was slow.  

Distributed tracing, on the other hand, provides a single window into latency information from across many components and plugins. Trace data is structured, and there are numerous established backends for visualizing and querying over it.

Due to the self-healing nature of Kubernetes, failures are retried until they succeed, causing bugs to manifest as operations that take slightly longer. Our current testing and telemetry, such as e2e tests and SLO monitoring, ensure processes in Kubernetes complete successfully within a bounded amount of time.  While this is effective for catching major regressions and unrecoverable errors, smaller regressions often manifest as flakes in unrelated tests or are otherwise a pain to track down.  Collecting structured trace data allows us to detect such regressions automatically, and quickly determine their root causes.


### Goals

* Make it possible to visualize the progress of objects across distinct Kubernetes components
* Streamline the tedious processes involved in debugging latency issues in Kubernetes
* Make it possible to identify high-level latency regressions, and attribute them back to offending processes


### Non-Goals

* To replace existing logging, metrics, or the events API
* To trace operations from all Kubernetes resource types in a generic manner (i.e. without manual instrumentation)


## Proposal

### Architecture

#### Trace lifecycle

Kubernetes is unique in that it is constantly reconciling its actual state towards some desired state. As a result, it has no definitive concept of an "operation", which breaks the traditional model for distributed tracing. This raises the question of when to begin traces, and when to end them.

In this proposal, we choose to _only_ trace phases of an object's lifecycle wherein it's correcting from an undesired state to its desired state, and to end the trace when it enters this desired state. This means that the same object will export traces for each reconciliation it undergoes. This decision was made because:

1. These reconciliation phases are where Kubernetes performs actual work that can be traced
1. Prevents traces from spanning the entire life of an object, which can be indefinite
1. Provides a consistent, concrete duration for the root span which remains applicable across object types
1. The time required for Kubernetes controllers to reconcile desired and actual state is, by definition, a measurement of either latency experienced by end-users in the fulfillment of their requests, or down-time due to a disruption

Concretely, when a component performs traced work, such as the kubelet creating a container, it uses the trace context associated with the version of the object it is performing the work based on.  For example:

1. The kubelet observes a new pod assigned to it through its APIServer watch with a container using image A.
1. The kubelet begins to perform traced actions, such as pulling image A, using the trace context obtained when it first observed the pod.
1. The kubelet concurrently observes an update to the pod to use image B
1. The kubelet continues the process it began above, and creates the container with image A.  *This action is still traced using the initial trace context from the pod when it was using image A*
1. After concluding the previous process, the kubelet performs traced actions to update the image to image B, such as pulling the image, deleting the old container, and creating the new container.  *These actions are now traced using the new trace context from the pod after the update to image B*

Components should plumb the context through reconciliation functions, rather than storing and looking up trace contexts globally so that each attempt to reconcile desired and actual state uses the context associated with _that_ desired state through the entire attempt.  If multiple components are involved in reconciling a single object, one may act on the new trace context before the other, but each trace is still representative of the work done to reconcile to the corresponding desired state. Given this model, we guarantee that each trace contains the actions taken to reconcile toward a single desired state.

High-level processes, such as starting a pod or restarting a failed container, could be interrupted before completion by an update to the desired state. While this leaves a "partial" trace for the first process, it is the most accurate representation of the work and timing of reconciling desired and actual state.

#### Root spans

In the standard model for distributed tracing, there exists a span in each trace that all other spans are descendents of and which extends the length of the entire trace, called the `root span`. While this is not a hard requirement, it makes traces interact better with visualization tools, and may have implications for analytical tools that expect a root span. For some processes, such as updates, the start time of the process is not stored with the object.  These processes do not have a root span in this version of the proposal.

The Kubernetes component that begins an operation is often not be the same component that ends it. In this proposal, when we are at the point where we want to end a root span, we craft a span to export which acts as the root span for the trace. For example, when the kubelet updates a pod from `Pending` to `Running`, it creates a root span using the start time of the pod as the start, and the current time as the end.  This works for processes where the start time is recorded, such as Creations and Deletions.

#### Context propagation

To correlate work done between components as belonging to the same trace, we must pass span context across process boundaries. In traditional distributed systems, this context can be passed down through RPC metadata or HTTP headers. Kubernetes, however, due to its watch-based nature, requires us to attach trace context directly to the target object. 

In this proposal, we choose to propagate this span context as an encoded string an object annotation called `trace.kubernetes.io/context`. This annotation value is regenerated and replaced when an object's trace ends, to achieve the desired behavior from [section one](#trace-lifecycle). 

This proposal chooses to use annotations as a less invasive alternative to adding a field to object metadata, but as this proposal matures, we should propagate trace context through watch events.

The alpha design adds extra writes to the APIServer for updating the trace context, which will limit the scalability of clusters using this feature.  These extra writes must be eliminated for this feature to move to beta.

### In-tree changes

#### Vendoring the OpenCensus trace framework

This KEP proposes the use of the [OpenCensus tracing framework](https://opencensus.io/) to create and export spans to configured backends. The OpenCensus framework was chosen for various reasons: 

1) Provides concrete, tested implementations for creating and exporting spans to diverse backends, rather than providing an API specification, as is the case with [OpenTracing](https://opentracing.io/specification/)
2) [Provides an agent](https://github.com/census-instrumentation/opencensus-service) which enables lazy configuration for exporters, batching of spans, and other features.  The agent allows importing and configuring exporters to be done out-of-tree.

This KEP suggests that we utilize the OpenCensus agent for the initial implementation to reduce the global changes required for alpha.  Alternative options include:

1. Add configuration for exporters in-tree by vendoring in each "supported" exporter. 
  a. This places the kubernetes community in the position of curating supported tracing backends
  b. This eliminates the requirement to run to OpenCensus agent in order to use tracing
2) Support *both* a curated set of in-tree exporters, and the agent exporter

While this setup is suitable for an alpha stage, it will require further review from Sig-Instrumentation and Sig-Architecture for beta, as it introduces a dependency on the OC Agent.  It is also worth noting that OpenCensus still has many unresolved details on how to run the agent.

#### Adding trace utility package

This package will be able to create spans from the span context embedded in the `trace.kubernetes.io/context` object annotation, in addition to embedding context from spans back into the annotation. This package will facilitate tracing across Kubernetes watches. It will also provide an implementation for exporting the root span for a given reconciliation trace.

This package must be divorced from specific exporters, and allow users to configure the desired tracing backend elsewhere (i.e. on the OC agent).

#### Instrumenting various Kubernetes components with pod startup traces

As we move forward with this KEP, we will use the aforementioned trace utility package to trace pod-related operations across the scheduler and kubelet. In code, this corresponds to creating a span (i.e. `ctx, span := trace.StartSpan(ctx, "Component.SampleSpan")`) at the beginning of an operation, and ending the span afterwards (`span.End()`). All calls to tracing functions will be gated with the `ObjectLifecycleTracing` alpha feature-gate, and will be disabled by default.

OpenCensus ships with plugins to transport trace context across gRPC and HTTP boundaries, which enables us to extend our tracing across the CRI and other internal boundaries.

In OpenCensus' Go implementation, span context is passed down through Go context. This will necessitate the threading of context across more of the Kubernetes codebase, which is a [desired outcome regardless](https://github.com/kubernetes/kubernetes/issues/815).

Following these initial pod-related trace additions, trace instrumentation should be added in an ad-hoc fashion to the various Kubernetes components.

### Out-of-tree changes

#### Mutating admission webhook

For spans to be correlated as part of the same trace, we must generate a `span context`, serialize it, and embed it in target traced objects. To accomplish this, we have introduced an [out-of-tree mutating admission webhook](https://github.com/Monkeyanator/mutating-trace-admission-controller/tree/review).

This mutating admission webhook generates a `span context`, which is the base64 encoded version of [this wire format](https://github.com/census-instrumentation/opencensus-specs/blob/master/encodings/BinaryEncoding.md#trace-context), and embeds it into the `trace.kubernetes.io/context` object annotation. The webhook can be configured to inject context into only target object types.

The decision on whether or not to sample traces from a given object is made in this webhook, upon the generation of the trace context. In addition to this, the aforementioned OpenCensus agent offers ex-post-facto trace sampling.

The proposed in-tree changes will utilize the span context annotation injected into objects with this webhook.

#### Tracing best-practices documentation

This KEP introduces a new form of instrumentation to Kubernetes, which necessiates the creation of guidelines for adding effective, standardized traces to Kubernetes components, [similar to what is found here for metrics](https://github.com/kubernetes/community/blob/master/contributors/devel/instrumentation.md).

This documentation will put forward standards for: 

* How to name spans, attributes, and annotations
* What kind of processes should be wrapped in a span
* When to link spans to other traces
* What kind of data makes for useful attributes

Having these standards in place will ensure that our tracing instrumentation works well with all backends, and that reviewers have concrete criteria to cross-check PRs against. 


### Graduation requirements

Alpha

- [] Alpha-implementation as described above
- [] E2e testing of traces
- [] User-facing documentation

Beta

- [] Security Review, including threat model
- [] Tracing must not increase the number of requests to the APIServer, which likely requires moving trace context generation and propagation in-tree.
- [] Deployment review including whether the [OC Agent](https://github.com/census-instrumentation/opencensus-service#opencensus-agent) is a required component
- [] Benchmark kubernetes components using tracing, and determine resource requirements and scaling for any additional required components (e.g. OC Agent).

GA

- [] Generalize to other kubernetes objects and CRDs
- [] Determine how to handle owner relationships in traces (e.g. tracing a replica set; do we append the associated pod traces to the replica set trace?)

## Implementation History

* [Mutating admission webhook which injects trace context](https://github.com/Monkeyanator/mutating-trace-admission-controller)
* [Instrumentation of Kubernetes components](https://github.com/Monkeyanator/kubernetes/pull/15)
* [Instrumentation of Kubernetes components for 1/24/2019 community demo](https://github.com/kubernetes/kubernetes/compare/master...dashpole:tracing)
