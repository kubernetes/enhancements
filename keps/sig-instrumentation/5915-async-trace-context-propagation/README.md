# KEP-5915: Standardizing Async Trace Context Propagation

<!-- toc -->
- [Release Signoff Checklist](#release-signoff-checklist)
- [Summary](#summary)
- [Motivation](#motivation)
  - [Goals](#goals)
  - [Non-Goals](#non-goals)
- [Proposal](#proposal)
  - [The "Span Link" Pattern](#the-span-link-pattern)
    - [Why Span Links?](#why-span-links)
    - [The Mechanism](#the-mechanism)
  - [User Stories](#user-stories)
    - [Story 1: Debugging Pod Creation Latency](#story-1-debugging-pod-creation-latency)
    - [Story 2: Custom Controller Observability](#story-2-custom-controller-observability)
  - [Notes/Constraints/Caveats](#notesconstraintscaveats)
  - [Risks and Mitigations](#risks-and-mitigations)
- [Design Details](#design-details)
  - [Annotation Standard](#annotation-standard)
  - [Library Changes (k8s.io/component-base)](#library-changes-k8siocomponent-base)
  - [Usage Examples](#usage-examples)
    - [Example 1: API Server Creating a Pod ✅ Inject](#example-1-api-server-creating-a-pod--inject)
    - [Example 2: Deployment Controller Creating ReplicaSet ✅ Inject](#example-2-deployment-controller-creating-replicaset--inject)
    - [Example 3: Kubelet Updating Pod Status ❌ Don't Inject](#example-3-kubelet-updating-pod-status--dont-inject)
    - [Example 4: Scheduler Assigning Node ⚠️ Special Case](#example-4-scheduler-assigning-node--special-case)
  - [Test Plan](#test-plan)
      - [Prerequisite testing updates](#prerequisite-testing-updates)
      - [Unit tests](#unit-tests)
      - [Integration tests](#integration-tests)
      - [e2e tests](#e2e-tests)
  - [Graduation Criteria](#graduation-criteria)
    - [Alpha](#alpha)
    - [Beta](#beta)
    - [Stable](#stable)
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
- [Alternatives](#alternatives)
  - [First-Class API Field](#first-class-api-field)
  - [Child Spans Instead of Span Links](#child-spans-instead-of-span-links)
  - [Mutating Admission Webhook](#mutating-admission-webhook)
  - [Automatic Instrumentation via Controller Runtime](#automatic-instrumentation-via-controller-runtime)
- [Infrastructure Needed (Optional)](#infrastructure-needed-optional)
<!-- /toc -->

## Release Signoff Checklist

Items marked with (R) are required *prior to targeting to a milestone / release*.

- [ ] (R) Enhancement issue in release milestone, which links to KEP dir in [kubernetes/enhancements] (not the initial KEP PR)
- [ ] (R) KEP approvers have approved the KEP status as `implementable`
- [ ] (R) Design details are appropriately documented
- [ ] (R) Test plan is in place, giving consideration to SIG Architecture and SIG Testing input (including test refactors)
  - [ ] e2e Tests for all Beta API Operations (endpoints)
  - [ ] (R) Ensure GA e2e tests meet requirements for [Conformance Tests](https://github.com/kubernetes/community/blob/master/contributors/devel/sig-architecture/conformance-tests.md)
  - [ ] (R) Minimum Two Week Window for GA e2e tests to prove flake free
- [ ] (R) Graduation criteria is in place
  - [ ] (R) [all GA Endpoints](https://github.com/kubernetes/community/pull/1806) must be hit by [Conformance Tests](https://github.com/kubernetes/community/blob/master/contributors/devel/sig-architecture/conformance-tests.md)
- [ ] (R) Production readiness review completed
- [ ] (R) Production readiness review approved
- [ ] "Implementation History" section is up-to-date for milestone
- [ ] User-facing documentation has been created in [kubernetes/website], for publication to [kubernetes.io]
- [ ] Supporting documentation—e.g., additional design documents, links to mailing list discussions/SIG meetings, relevant PRs/issues, release notes

[kubernetes.io]: https://kubernetes.io/
[kubernetes/enhancements]: https://git.k8s.io/enhancements
[kubernetes/kubernetes]: https://git.k8s.io/kubernetes
[kubernetes/website]: https://git.k8s.io/website

## Summary

This proposal aims to standardize how Distributed Tracing context is propagated across asynchronous boundaries within Kubernetes. Currently, traces often fragment when operations move from the synchronous API Server plane to asynchronous controllers (e.g., Scheduler, Kubelet, Controller Manager).

I propose adding a standardized mechanism to `k8s.io/component-base` that allows controllers to persist W3C Trace Context into object annotations. Downstream components can then consume this context to create Span Links, preserving the causal relationship of operations without implying synchronous dependencies.

## Motivation

Kubernetes is fundamentally an event-driven, asynchronous system. A user's request (e.g., "Create Pod") triggers a cascade of reconciliation loops that may happen milliseconds or minutes later.

**The Problem:** Current tracing approaches often treat these asynchronous operations as entirely separate traces. There is no standard way for a controller to know *why* it is reconciling an object (i.e., which user request triggered this state).

**The Gap:** While developers can manually patch annotations, there is no agreed-upon standard for the annotation key or the semantic relationship (Child vs. Link) between the trigger and the reconciliation.

**The Impact:** Observability backends display disjointed traces. Users cannot see the End-to-End (E2E) lifecycle of a resource.

### Goals

* **Standardize the Storage:** Define a reserved annotation key (e.g., `tracing.k8s.io/traceparent`) for storing W3C TraceParent data on Kubernetes objects.
* **Standardize the Semantics:** Explicitly define Span Links (not Child Spans) as the correct OpenTelemetry primitive for asynchronous Kubernetes reconciliation.
* **Library Implementation:** Implement reusable `Inject` and `Extract` primitives in `k8s.io/component-base/tracing` to handle this logic for all controllers.

### Non-Goals

* Modifying the Kubernetes API to add tracing context as a first-class field (this uses annotations instead).
* Automatically instrumenting all controllers (adoption is opt-in per component).
* Replacing existing logging, metrics, or the events API.
* Defining how observability backends should visualize Span Links.

## Proposal

### The "Span Link" Pattern

#### Why Span Links?

In synchronous systems, Child Spans are used to represent dependency. However, in Kubernetes, the downstream action (e.g., Kubelet syncing a Pod) is asynchronous and decoupled from the API request.

Treating reconciliation as a Child Span of the API request is **incorrect** because:

1. **Latency Distortion:** It implies the API request is "blocked" until the reconciliation finishes, which breaks waterfall visualizations if the reconciliation happens minutes later.
2. **Independence:** The reconciliation loop has its own lifecycle and may be triggered by multiple events.

Therefore, I propose using [Span Links](https://opentelemetry.io/docs/concepts/signals/traces/#span-links) (as defined in the OpenTelemetry Specification). The reconciler starts a new root span but adds a Link pointing to the Trace ID stored in the object's annotation. This preserves the causal connection ("This ran because of that") without implying a synchronous parent-child structure.

#### The Mechanism

The workflow consists of two phases: **Injection (Producer)** and **Extraction (Consumer)**.

**Phase 1: Injection (Producer)**

When a component creates or updates an object's **spec** based on an incoming request, it is responsible for "stamping" the object with the current trace context.

**When to inject:**
- Creating a new object (e.g., API Server creates a Pod from user request)
- Updating the **spec** (e.g., changing a Deployment's replica count, updating a label)
- Making semantic changes that represent user intent or control plane decisions

**When NOT to inject:**
- Updating **status** fields (e.g., Pod phase transitions, condition updates)
- Heartbeats or leader election updates
- Internal housekeeping that doesn't represent a new "action"

The guideline: **Inject when the update represents a new action or decision**, not for routine status reporting.

**Who is responsible for injection:**
- **API Server**: When handling user requests that create objects (e.g., `kubectl create pod`)
- **Controllers**: When creating child objects or making control plane decisions (e.g., Deployment Controller creating ReplicaSets, ReplicaSet Controller creating Pods)
- **Mutating Admission Webhooks**: Could inject context when modifying objects (though not recommended as primary approach - see Alternatives section)

Each component that creates or updates an object's spec should call `tracing.InjectContext()` with its current context before persisting the object.

* **Trigger:** Incoming API Request (with W3C `traceparent` header).
* **Action:**
  1. OpenTelemetry middleware extracts `traceparent` from HTTP headers and propagates it into the Go `context.Context`.
  2. Component code (API Server, Controller) calls `tracing.InjectContext(ctx, obj, propagator)`.
  3. The `InjectContext()` function extracts trace context from the Go `context.Context` and writes it to the object's annotations (`tracing.k8s.io/traceparent`).
  4. Persist the object to etcd with the annotation.

**Phase 2: Extraction (Consumer)**

When a controller (Scheduler, Kubelet, Custom Operator) picks up an object from a generic informer/watch:

* **Trigger:** Reconcile Loop starts.
* **Action:**
  1. Controller code calls `tracing.StartReconcileSpan(ctx, "ReconcilePod", pod, tracer, propagator)`.
  2. The `StartReconcileSpan()` function checks the object's annotations for `tracing.k8s.io/traceparent`.
  3. If present: The function extracts the `traceparent` value and creates a **New Root Span** with a **Span Link** to the stored trace ID.
  4. If not present: The function creates a regular root span without a link.
  5. (Optional) When creating child objects, the controller calls `tracing.InjectContext(ctx, childObj, propagator)` to propagate the tracing chain.

**Important Note on HTTP vs. Annotation Propagation:**

Once the initial HTTP request with `traceparent` header reaches the API Server and gets injected into the object's annotations, **subsequent propagation happens via annotations only**. Controllers (Scheduler, Kubelet, Controller Manager) do not receive HTTP requests with trace headers - they read objects from their watch/informer and extract trace context from the stored annotations. This is why annotations are the critical propagation mechanism for async boundaries in Kubernetes.

**Flow:**
```
User → HTTP (traceparent header) → API Server → Annotation → etcd
                                                      ↓
                                              (watch/informer)
                                                      ↓
Scheduler/Controller → Reads Annotation → Creates Linked Span
```

### User Stories

#### Story 1: Debugging Pod Creation Latency

As a cluster operator, I want to trace the full lifecycle of a Pod from the initial `kubectl apply` through the API Server, Scheduler, and Kubelet. With this feature, I can:

1. Make a request to create a Pod with tracing enabled.
2. The API Server injects the trace context into the Pod's annotations.
3. The Scheduler picks up the Pod, extracts the trace context, and creates a linked span for the scheduling decision.
4. The Kubelet picks up the Pod, extracts the trace context, and creates a linked span for the Pod sync operation.
5. In my observability backend, I can see the complete causal chain across all components, even though they operated asynchronously.

#### Story 2: Custom Controller Observability

As a developer of a custom Kubernetes operator, I want to understand why my controller is reconciling objects. By using the standard library functions:

1. My controller extracts trace context from incoming objects.
2. I can create spans linked to the original user request.
3. I gain visibility into the relationship between user actions and my controller's behavior.

### Notes/Constraints/Caveats

* **Multiple Writers:** Multiple components may update an object, potentially overwriting the trace context annotation. Since this KEP uses Span Links (not parent-child), this is acceptable—each update can represent a new triggering context. A future enhancement could support appending multiple trace contexts (e.g., `tracing.k8s.io/traceparent.0`, `tracing.k8s.io/traceparent.1`) to preserve the full lineage, though this adds complexity.
* **Annotation Size:** The W3C TraceParent format is compact (approximately 55 bytes), so annotation size impact is minimal.
* **Backwards Compatibility:** Controllers that don't implement this feature will simply ignore the annotation.
* **Baggage Support:** The proposal focuses on `traceparent` but the design can be extended to support W3C Baggage headers in the future using a `tracing.k8s.io/baggage` annotation.

### Risks and Mitigations

| Risk | Mitigation |
|------|------------|
| Annotation clobbering by multiple writers | Using Span Links instead of parent-child relationships makes this acceptable semantically |
| Performance overhead from annotation processing | The overhead is negligible (string parsing); benchmarks will be provided |
| Confusion about when to inject context | Clear documentation and guidelines distinguish spec updates (inject) vs. status updates (don't inject) |
| Adoption friction | Providing easy-to-use library functions in component-base reduces friction |

## Design Details

### Annotation Standard

I propose reserving a specific annotation key to avoid collisions and allow standard collectors to recognize the data.

* **Proposed Key:** `tracing.k8s.io/traceparent`
* **Value Format:** W3C TraceContext String (`Version-TraceID-ParentID-TraceFlags`)
* **Example:** `00-4bf92f3577b34da6a3ce929d0e0e4737-00f067aa0ba902b7-01`

Future extension for baggage:
* **Key:** `tracing.k8s.io/baggage`
* **Value Format:** W3C Baggage format

### Library Changes (k8s.io/component-base)

I propose adding the following utilities to the `component-base/tracing` package:

```go
package tracing

import (
    "context"

    "go.opentelemetry.io/otel/propagation"
    "go.opentelemetry.io/otel/trace"
    metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

const (
    // TracingAnnotationPrefix is the prefix for tracing-related annotations.
    TracingAnnotationPrefix = "tracing.k8s.io/"
    
    // TraceparentAnnotation is the annotation key for W3C traceparent.
    TraceparentAnnotation = TracingAnnotationPrefix + "traceparent"
    
    // BaggageAnnotation is the annotation key for W3C baggage.
    BaggageAnnotation = TracingAnnotationPrefix + "baggage"
)

// AnnotationCarrier implements propagation.TextMapCarrier for Kubernetes object annotations.
// This allows OpenTelemetry propagators to read/write trace context directly to/from
// Kubernetes object annotations using a standard interface.
//
// The carrier uses a fixed prefix (TracingAnnotationPrefix = "tracing.k8s.io/") and
// appends the propagator's keys (e.g., "traceparent", "baggage") to form the full
// annotation keys (e.g., "tracing.k8s.io/traceparent"). This makes the specific
// headers/keys pluggable via the propagator while maintaining a consistent namespace.
type AnnotationCarrier struct {
    object metav1.Object
}

// NewAnnotationCarrier creates a new AnnotationCarrier for the given object.
func NewAnnotationCarrier(obj metav1.Object) *AnnotationCarrier {
    return &AnnotationCarrier{object: obj}
}

// Get returns the value for the given key from annotations.
// The key from the propagator (e.g., "traceparent") is prefixed with TracingAnnotationPrefix.
func (c *AnnotationCarrier) Get(key string) string {
    annotations := c.object.GetAnnotations()
    if annotations == nil {
        return ""
    }
    return annotations[TracingAnnotationPrefix+key]
}

// Set stores the key-value pair in annotations.
// The key from the propagator (e.g., "traceparent") is prefixed with TracingAnnotationPrefix.
func (c *AnnotationCarrier) Set(key, value string) {
    annotations := c.object.GetAnnotations()
    if annotations == nil {
        annotations = make(map[string]string)
    }
    annotations[TracingAnnotationPrefix+key] = value
    c.object.SetAnnotations(annotations)
}

// Keys returns all keys in the carrier (without the prefix).
// Returns the propagator keys (e.g., "traceparent", "baggage") for annotations
// matching TracingAnnotationPrefix.
func (c *AnnotationCarrier) Keys() []string {
    annotations := c.object.GetAnnotations()
    var keys []string
    for k := range annotations {
        if len(k) > len(TracingAnnotationPrefix) && k[:len(TracingAnnotationPrefix)] == TracingAnnotationPrefix {
            keys = append(keys, k[len(TracingAnnotationPrefix):])
        }
    }
    return keys
}

// InjectContext injects the trace context from ctx into the object's annotations.
func InjectContext(ctx context.Context, obj metav1.Object, propagator propagation.TextMapPropagator) {
    carrier := NewAnnotationCarrier(obj)
    propagator.Inject(ctx, carrier)
}

// ExtractContext extracts trace context from the object's annotations.
func ExtractContext(ctx context.Context, obj metav1.Object, propagator propagation.TextMapPropagator) context.Context {
    carrier := NewAnnotationCarrier(obj)
    return propagator.Extract(ctx, carrier)
}

// StartReconcileSpan starts a new root span for reconciliation, linked to any
// trace context stored in the object's annotations.
func StartReconcileSpan(
    ctx context.Context,
    name string,
    obj metav1.Object,
    tracer trace.Tracer,
    propagator propagation.TextMapPropagator,
) (context.Context, trace.Span) {
    opts := []trace.SpanStartOption{
        trace.WithSpanKind(trace.SpanKindConsumer),
    }

    // Extract stored context and create a link if present
    extractedCtx := ExtractContext(context.Background(), obj, propagator)
    remoteSpanCtx := trace.SpanContextFromContext(extractedCtx)
    
    if remoteSpanCtx.IsValid() {
        // Create a new root span with a link to the original context
        opts = append(opts,
            trace.WithLinks(trace.Link{SpanContext: remoteSpanCtx}),
            trace.WithNewRoot(),
        )
    }

    return tracer.Start(ctx, name, opts...)
}
```

### Usage Examples

To clarify when to inject trace context, here are concrete examples:

#### Example 1: API Server Creating a Pod ✅ Inject

```go
// User creates Pod via API
func (s *PodStorage) Create(ctx context.Context, obj runtime.Object) {
    pod := obj.(*v1.Pod)
    
    // Inject trace context - this represents user's intent
    tracing.InjectContext(ctx, pod, propagator)
    
    s.storage.Create(ctx, pod)
}
```

#### Example 2: Deployment Controller Creating ReplicaSet ✅ Inject

```go
func (dc *DeploymentController) syncDeployment(ctx context.Context, deployment *appsv1.Deployment) {
    // Controller decides to create a new ReplicaSet (spec change)
    rs := constructReplicaSet(deployment)
    
    // Inject - this is a control plane decision
    tracing.InjectContext(ctx, rs, propagator)
    
    dc.client.Create(ctx, rs)
}
```

#### Example 3: Kubelet Updating Pod Status ❌ Don't Inject

```go
func (kl *Kubelet) updatePodStatus(pod *v1.Pod) {
    // Updating status - don't inject, this is status reporting
    pod.Status.Phase = v1.PodRunning
    pod.Status.Conditions = append(pod.Status.Conditions, readyCondition)
    
    // NO InjectContext call here
    kl.client.UpdateStatus(ctx, pod)
}
```

#### Example 4: Scheduler Assigning Node ⚠️ Special Case

```go
func (s *Scheduler) bind(ctx context.Context, pod *v1.Pod, node string) {
    // This is tricky - it's updating spec (nodeName) but arguably
    // should link to the scheduling decision context, not overwrite
    // the original creation context
    
    // Start linked span for scheduling decision
    ctx, span := tracing.StartReconcileSpan(ctx, "SchedulePod", pod, tracer, propagator)
    defer span.End()
    
    // Option 1: Don't inject (preserve creation context)
    pod.Spec.NodeName = node
    s.client.Update(ctx, pod)
    
    // Option 2: Inject (track scheduling as new action)
    // tracing.InjectContext(ctx, pod, propagator)
    // s.client.Update(ctx, pod)
}
```

**Guideline:** When in doubt, ask "Does this update represent a new decision or action?" If yes, inject. If it's just status reporting, don't inject.

### Test Plan


[X] I understand the owners of the involved components may require updates to
existing tests to make this code solid enough prior to committing the changes necessary
to implement this enhancement.

##### Prerequisite testing updates

None.

##### Unit tests

- `staging/src/k8s.io/component-base/tracing/annotation_carrier_test.go`: Tests for the AnnotationCarrier implementation
- `staging/src/k8s.io/component-base/tracing/context_propagation_test.go`: Tests for InjectContext, ExtractContext, and StartReconcileSpan functions

Target coverage: >80%

##### Integration tests

- Integration tests verifying that trace context is properly propagated through the API Server to object annotations.
- Integration tests verifying that controllers can extract and link to stored trace context.

##### e2e tests

- e2e test creating a Pod and verifying trace context flows through Scheduler and Kubelet with proper Span Links.

### Graduation Criteria

#### Alpha

- [ ] Define the annotation key standard (`tracing.k8s.io/traceparent`).
- [ ] Implement helper functions in `k8s.io/component-base/tracing`.
- [ ] Add feature gate `AsyncTraceContextPropagation`.
- [ ] Unit tests for all new library functions.
- [ ] Documentation for library usage.

#### Beta

- [ ] Adoption by at least one core component (e.g., Scheduler or Controller Manager).
- [ ] Integration tests demonstrating end-to-end context propagation.
- [ ] Gather feedback from early adopters.
- [ ] Address any issues discovered during alpha.

#### Stable

- [ ] Widespread adoption across Kubernetes system components, including the Kubelet.
- [ ] e2e tests demonstrating full lifecycle tracing.
- [ ] At least two releases of beta feedback incorporated.
- [ ] Documentation in kubernetes.io.

### Upgrade / Downgrade Strategy

**Upgrade:** When upgrading to a version with this feature:
- Existing objects will not have tracing annotations (this is expected).
- New or updated objects will have annotations injected if the feature is enabled.
- Controllers will start creating Span Links for objects with annotations.

**Downgrade:** When downgrading to a version without this feature:
- Tracing annotations on existing objects are ignored (they're just regular annotations).
- No Span Links will be created, but the system continues to function normally.
- Annotations can be garbage collected manually if desired, but this is not required.

### Version Skew Strategy

This feature is tolerant of version skew:

- **Older API Server + Newer Controller:** The controller won't find trace annotations and will operate normally without Span Links.
- **Newer API Server + Older Controller:** The annotations will be present but ignored by the older controller.
- **Mixed Controller Versions:** Each controller independently decides whether to inject/extract context based on its version.

The feature does not require coordination between components at different versions.

## Production Readiness Review Questionnaire

### Feature Enablement and Rollback

###### How can this feature be enabled / disabled in a live cluster?

- [X] Feature gate
  - Feature gate name: `AsyncTraceContextPropagation`
  - Components depending on the feature gate:
    - kube-apiserver
    - kube-controller-manager
    - kube-scheduler
    - kubelet
- [X] Other
  - Describe the mechanism: Components must explicitly call the library functions to inject/extract context. The feature gate controls whether these calls have any effect.
  - Will enabling / disabling the feature require downtime of the control plane? No, but a restart of components is required to change the feature gate.
  - Will enabling / disabling the feature require downtime or reprovisioning of a node? No.

###### Does enabling the feature change any default behavior?

Yes, when enabled:
- Objects created or updated by instrumented components will have `tracing.k8s.io/traceparent` annotations added.
- Controllers will create Span Links to stored trace context when reconciling.

This does not affect the functional behavior of the system, only observability.

###### Can the feature be disabled once it has been enabled (i.e. can it be rolled back)?

Yes. Disabling the feature gate will:
- Stop new annotations from being injected.
- Stop Span Links from being created.
- Existing annotations remain but are ignored.

###### What happens if the feature is reenabled after it was previously rolled back?

The feature resumes normal operation:
- New annotations are injected for new/updated objects.
- Span Links are created for objects with existing annotations.

###### Are there any tests for feature enablement/disablement?

Yes, unit tests will verify that:
- When disabled, `InjectContext` is a no-op.
- When disabled, `StartReconcileSpan` creates a normal span without links.

### Rollout, Upgrade and Rollback Planning

###### How can a rollout or rollback fail? Can it impact already running workloads?

The feature only affects observability and does not impact workload scheduling, execution, or networking. A rollout or rollback cannot cause workload failures.

Potential issues:
- If annotation size limits are approached (unlikely given the small size of trace context), object updates could fail. This is mitigated by the compact W3C format.

###### What specific metrics should inform a rollback?

- `trace_context_annotations_injected_total{result="error"}`: High error rates when injecting trace context.
- `trace_context_annotations_extracted_total{result="error"}`: High error rates when extracting trace context.
- `trace_context_annotations_injected_total` filtered by `component` or `controller`: Unexpectedly high injection rates from specific components indicating a bug or misconfiguration.
- `apiserver_request_duration_seconds`: If annotation processing adds noticeable latency.
- Object update failures related to annotation size (monitored via API Server error metrics).

###### Were upgrade and rollback tested? Was the upgrade->downgrade->upgrade path tested?

Will be tested during alpha development.

###### Is the rollout accompanied by any deprecations and/or removals of features, APIs, fields of API types, flags, etc.?

No.

### Monitoring Requirements

###### How can an operator determine if the feature is in use by workloads?

- Check for `tracing.k8s.io/traceparent` annotations on objects.
- Monitor the `trace_context_annotations_injected_total` metric, filtering by `component`, `controller`, `node`, or `resource_kind` labels to identify specific sources of issues.
- Look for Span Links in trace data from the observability backend.

###### How can someone using this feature know that it is working for their instance?

- [X] Metrics
  - `trace_context_annotations_injected_total`: Non-zero values indicate components are injecting trace context
  - `trace_context_annotations_extracted_total`: Non-zero values indicate components are extracting and linking trace context
  - Filter by `component`, `controller`, or `resource_kind` labels to verify specific adoption
- [X] Other
  - Details: Check that objects have `tracing.k8s.io/traceparent` annotations after creation. Verify Span Links appear in the trace backend.

###### What are the reasonable SLOs (Service Level Objectives) for the enhancement?

This feature is for observability and does not have direct SLOs. It should not impact existing API Server SLOs.

###### What are the SLIs (Service Level Indicators) an operator can use to determine the health of the service?

- [X] Metrics
  - Metric name: `trace_context_annotations_injected_total`
  - Components exposing the metric: kube-apiserver, kube-controller-manager
  - Labels:
    - `component`: Component name (e.g., kube-apiserver, kube-controller-manager)
    - `controller`: Specific controller name within kube-controller-manager (e.g., deployment-controller, replicaset-controller)
    - `resource_kind`: Kubernetes resource kind (e.g., Pod, Deployment, ReplicaSet)
    - `result`: Operation result (success, error)
- [X] Metrics
  - Metric name: `trace_context_annotations_extracted_total`
  - Components exposing the metric: kube-scheduler, kubelet, kube-controller-manager
  - Labels:
    - `component`: Component name (e.g., kube-scheduler, kubelet, kube-controller-manager)
    - `controller`: Specific controller name within kube-controller-manager
    - `node`: Node name (for kubelet only)
    - `resource_kind`: Kubernetes resource kind being reconciled
    - `result`: Operation result (success, error)

###### Are there any missing metrics that would be useful to have to improve observability of this feature?

No. The metrics above with their labels provide sufficient observability. The `result` label on both metrics allows tracking of success and error rates. The `component`, `controller`, `node`, and `resource_kind` labels enable granular analysis of feature adoption and usage patterns across the cluster.

### Dependencies

###### Does this feature depend on any specific services running in the cluster?

No. The feature works independently but provides value only when combined with distributed tracing infrastructure:

- **Tracing enabled in Kubernetes components**: Any W3C TraceContext-compliant tracing implementation. Kubernetes currently uses OpenTelemetry (see KEP-647 for API Server, KEP-2831 for Kubelet).
- **Trace collection pipeline**: Any system that can collect and forward traces (e.g., OpenTelemetry Collector, Jaeger Agent, Zipkin Collector).
- **Trace storage backend**: Any backend that supports W3C TraceContext and Span Links (e.g., Jaeger, Tempo, Lightstep, Honeycomb).

Note: This KEP uses the W3C TraceContext standard, making it compatible with any compliant tracing system, not just OpenTelemetry.

### Scalability

###### Will enabling / using this feature result in any new API calls?

No additional API calls. The annotation is added during existing create/update operations.

###### Will enabling / using this feature result in introducing new API types?

No.

###### Will enabling / using this feature result in any new calls to the cloud provider?

No.

###### Will enabling / using this feature result in increasing size or count of the existing API objects?

Yes, a small increase in object size:
- API type(s): All objects that have trace context injected.
- Estimated increase in size: ~100 bytes per object (annotation key + W3C traceparent value).
- Estimated amount of new objects: None.

###### Will enabling / using this feature result in increasing time taken by any operations covered by existing SLIs/SLOs?

Negligible increase (<1 microsecond) for:
- Encoding trace context into annotations on object create/update.
- Extracting trace context from annotations during reconciliation.

###### Will enabling / using this feature result in non-negligible increase of resource usage (CPU, RAM, disk, IO, ...) in any components?

No. The overhead is minimal:
- CPU: Negligible (string formatting/parsing).
- RAM: Negligible (no additional caching).
- Disk: Small increase in etcd storage for annotations (~100 bytes per object).
- IO: No additional IO.

###### Can enabling / using this feature result in resource exhaustion of some node resources (PIDs, sockets, inodes, etc.)?

No. The feature only adds annotations to objects and creates trace spans, which are handled by the existing tracing infrastructure.

### Troubleshooting

###### How does this feature react if the API server and/or etcd is unavailable?

The feature depends on the API Server and etcd for normal operation (storing objects with annotations). If they are unavailable, no objects can be created/updated, so no trace context can be propagated. This is expected behavior.

###### What are other known failure modes?

- **Trace context not appearing in annotations**
  - Detection: Objects lack `tracing.k8s.io/traceparent` annotation.
  - Mitigations: Verify feature gate is enabled; verify tracing is configured in the component.
  - Diagnostics: Check component logs for tracing-related messages.
  - Testing: Unit tests verify injection behavior.

- **Span Links not appearing in traces**
  - Detection: Trace backend shows disconnected spans without links.
  - Mitigations: Verify controllers are using `StartReconcileSpan`; verify annotations exist on objects.
  - Diagnostics: Check controller logs; inspect object annotations.
  - Testing: Integration tests verify link creation.

###### What steps should be taken if SLOs are not being met to determine the problem?

1. Check if the feature gate is enabled.
2. Verify tracing is configured in the affected components.
3. Check component logs for errors related to trace context injection/extraction.
4. Disable the feature gate if it's suspected to cause issues (unlikely given the minimal overhead).

## Implementation History

- 2026-02-11: Initial KEP draft created by @artem-tkachuk and enhancement issue filed (#5915).

## Drawbacks

- **Annotation Pollution:** Objects will have additional annotations, which some users may find undesirable. However, the annotation is small and follows Kubernetes conventions.
- **Partial Adoption:** Until all components adopt this pattern, traces will have gaps. This is expected during the rollout period.
- **No Automatic Instrumentation:** Developers must explicitly use the library functions to participate in trace propagation. 
  - Future enhancements could explore automatic injection/extraction for common controller patterns, though this requires careful consideration of which reconciliation events warrant trace propagation.

## Alternatives

### First-Class API Field

Instead of using annotations, trace context could be added as a first-class field in `ObjectMeta` (e.g., `metadata.traceContext`).

**Benefits:**
- Direct API access without annotation parsing
- Type-safe structure instead of string values
- Could support multiple parent contexts natively (e.g., `[]TraceContext`)
- No SDK dependency for basic read/write operations

**Why rejected:**
- **API change scope**: Requires modifying the core API across all resource types, which is a much higher bar than adding library utilities
- **API bloat**: ObjectMeta is already complex; adding observability-specific fields sets a precedent for other metadata types
- **Annotations are idiomatic**: Kubernetes uses annotations for operational metadata that doesn't affect resource semantics
- **SDK is minimal**: The proposed library functions are thin wrappers that simply adapt annotations to W3C format
- **Backward compatibility**: Annotations can be read by any Kubernetes client; a new field requires client library updates

This remains a valid alternative for future consideration if annotation-based approach proves insufficient.

### Child Spans Instead of Span Links

Using child spans instead of links was considered but rejected because:
- It incorrectly implies synchronous dependency.
- It distorts latency visualizations.
- It doesn't represent the true asynchronous nature of Kubernetes.

### Mutating Admission Webhook

A mutating admission webhook could inject trace context, decoupling the logic from components. This was considered but:
- It adds latency to all requests.
- It requires additional infrastructure.
- It doesn't help with the extraction/linking side.

The component-base library approach provides a cleaner, more integrated solution.

### Automatic Instrumentation via Controller Runtime

Instead of requiring explicit calls to `InjectContext` and `ExtractContext`, the controller runtime (e.g., controller-runtime or client-go's `SharedInformer`) could automatically inject/extract trace context.

**Potential Implementation:**
- Automatically inject trace context when creating/updating objects via the client
- Automatically extract trace context and start linked spans in reconciliation loops

**Why not now:**
- **Semantic ambiguity**: Not all updates warrant trace propagation (e.g., status updates, leader election heartbeats)
- **Opt-in vs. opt-out**: Starting with explicit opt-in ensures controlled rollout and clearer semantics
- **Flexibility**: Different controllers may have different trace propagation needs

**Future consideration:** The explicit library functions in this KEP provide a foundation that could be wrapped by automatic instrumentation in the future. Feedback from early adopters will help determine if/when automatic instrumentation is appropriate and how it should be scoped.

## Infrastructure Needed (Optional)

None. This feature uses existing Kubernetes infrastructure (annotations, component-base).
