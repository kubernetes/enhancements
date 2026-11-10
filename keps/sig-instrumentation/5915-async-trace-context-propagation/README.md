# KEP-5915: Trace Context Propagation and Distributed Tracing for Controllers

<!-- toc -->
- [Release Signoff Checklist](#release-signoff-checklist)
- [Summary](#summary)
- [Motivation](#motivation)
  - [Goals](#goals)
  - [Non-Goals](#non-goals)
- [Proposal](#proposal)
  - [User Stories (Optional)](#user-stories-optional)
  - [Notes/Constraints/Caveats (Optional)](#notesconstraintscaveats-optional)
  - [Risks and Mitigations](#risks-and-mitigations)
- [Design Details](#design-details)
  - [Annotation Carrier Convention (<code>tracing.k8s.io/*</code>)](#annotation-carrier-convention-tracingk8sio)
  - [Out-of-Tree Context Injection (<code>sig-instrumentation</code> Sub-Project)](#out-of-tree-context-injection-sig-instrumentation-sub-project)
  - [Context Extraction and Span Linking in <code>k8s.io/component-base/tracing</code>](#context-extraction-and-span-linking-in-k8siocomponent-basetracing)
  - [Link-Aware and Consistent Probability Sampling](#link-aware-and-consistent-probability-sampling)
  - [<code>kube-controller-manager</code> Configuration and Alpha Instrumentation](#kube-controller-manager-configuration-and-alpha-instrumentation)
  - [Test Plan](#test-plan)
      - [Prerequisite testing updates](#prerequisite-testing-updates)
      - [Unit tests](#unit-tests)
      - [Integration tests](#integration-tests)
      - [e2e tests](#e2e-tests)
  - [Graduation Criteria](#graduation-criteria)
    - [Alpha](#alpha)
    - [Beta](#beta)
    - [GA](#ga)
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
  - [Parent-Child Spans Across Reconciliations](#parent-child-spans-across-reconciliations)
  - [Controller-Side Annotation Injection](#controller-side-annotation-injection)
  - [In-Tree Admission Plugin in <code>kube-apiserver</code>](#in-tree-admission-plugin-in-kube-apiserver)
  - [First-Class <code>ObjectMeta</code> Trace Context Field](#first-class-objectmeta-trace-context-field)
- [Infrastructure Needed (Optional)](#infrastructure-needed-optional)
<!-- /toc -->

## Release Signoff Checklist

Items marked with (R) are required *prior to targeting to a milestone / release*.

- [x] (R) Enhancement issue in release milestone, which links to KEP dir in [kubernetes/enhancements] (not the initial KEP PR)
- [x] (R) KEP approvers have approved the KEP status as `implementable`
- [x] (R) Design details are appropriately documented
- [x] (R) Test plan is in place, giving consideration to SIG Architecture and SIG Testing input (including test refactors)
  - [ ] e2e Tests for all Beta API Operations (endpoints)
  - [ ] (R) Ensure GA e2e tests meet requirements for [Conformance Tests](https://github.com/kubernetes/community/blob/master/contributors/devel/sig-architecture/conformance-tests.md)
  - [ ] (R) Minimum Two Week Window for GA e2e tests to prove flake free
- [x] (R) Graduation criteria is in place
  - [ ] (R) [all GA Endpoints](https://github.com/kubernetes/community/pull/1806) must be hit by [Conformance Tests](https://github.com/kubernetes/community/blob/master/contributors/devel/sig-architecture/conformance-tests.md) within one minor version of promotion to GA
- [x] (R) Production readiness review completed
- [ ] (R) Production readiness review approved
- [x] "Implementation History" section is up-to-date for milestone
- [ ] User-facing documentation has been created in [kubernetes/website], for publication to [kubernetes.io]
- [x] Supporting documentation—e.g., additional design documents, links to mailing list discussions/SIG meetings, relevant PRs/issues, release notes

[kubernetes.io]: https://kubernetes.io/
[kubernetes/enhancements]: https://git.k8s.io/enhancements
[kubernetes/kubernetes]: https://git.k8s.io/kubernetes
[kubernetes/website]: https://git.k8s.io/website
[KEP-647]: /keps/sig-instrumentation/647-apiserver-tracing/README.md
[KEP-2831]: /keps/sig-instrumentation/2831-kubelet-tracing/README.md

## Summary

This KEP proposes a standard mechanism for propagating OpenTelemetry trace context across asynchronous Kubernetes controllers and adds distributed tracing support to `kube-controller-manager`.

Trace context is stored on Kubernetes objects using `tracing.k8s.io/*` annotations (`tracing.k8s.io/traceparent`, `tracing.k8s.io/tracestate`, and `tracing.k8s.io/baggage`). Context injection into object annotations is performed out-of-tree by a Mutating Admission Webhook maintained as a new `sig-instrumentation` sub-project in `kubernetes-sigs`, which atomically stamps annotations during `.spec`-mutating admission requests while ignoring `/status` subresource updates. Context extraction and OpenTelemetry Span Link creation are provided by shared helpers in `k8s.io/component-base/tracing` (`ObjectCarrier`, `WithObjects`, and link-aware sampling), and `kube-controller-manager` is instrumented behind the `ControllerManagerTracing` feature gate (starting with the `Deployment` and `ReplicaSet` controllers in Alpha).

## Motivation

Kubernetes supports distributed tracing for synchronous API requests in `kube-apiserver` ([KEP-647]) and node-local pod lifecycle operations in `kubelet` ([KEP-2831]). However, control-plane operations in Kubernetes are predominantly asynchronous: a client mutates desired state on an API object in `kube-apiserver`, and controllers (`Deployment`, `ReplicaSet`, `kube-scheduler`, `kubelet`, and custom operators) react asynchronously via watch events and workqueues.

Because watch events carry only the serialized Kubernetes object, the in-memory HTTP `context.Context` ends as soon as `kube-apiserver` finishes serving the write request:

1. Traces break at asynchronous object boundaries, leaving operators unable to correlate a `Deployment` rollout with the resulting `ReplicaSet` and `Pod` creations.
2. Controllers need to agree on common propagation formats and mechanisms so that trace instrumentation from controllers across the Kubernetes ecosystem can interoperate.
3. There are many tricky design aspects—such as level-triggered workqueue coalescing, `/status` update feedback loops, and long-lived resources—that can lead to problematic implementations (for example, naively propagating parent-child spans or mutating annotations from inside controllers can cause infinite watch-reconcile loops, extra API calls, and multi-day trace trees). Solving these in central libraries ensures safe, consistent implementations.

### Goals

- Enable end-to-end trace correlation across asynchronous Kubernetes resource mutations and controller reconciliations (for example, correlating a `Deployment` rollout with the resulting `ReplicaSet` and `Pod` creations).
- Enable trace context interoperability across in-tree controllers (`kube-controller-manager`) and out-of-tree controllers (`controller-runtime` and custom operators) using shared `k8s.io/component-base/tracing` libraries.
- Enable exporting OpenTelemetry reconciliation spans from `kube-controller-manager` behind the `ControllerManagerTracing` feature gate (starting with the `Deployment` and `ReplicaSet` controllers in Alpha).
- Require zero additional Kubernetes API calls for trace context propagation.
- Prevent infinite watch-reconcile loops (including from `/status` updates) and keep individual trace durations bounded even for long-lived objects.
- Avoid changes to `metav1.ObjectMeta` or new in-tree `kube-apiserver` admission plugins, and do not require granting controllers `system:masters` or `system:monitoring` privileges to propagate trace context.

### Non-Goals

- Instrumenting every controller in `kube-controller-manager` in Alpha (additional controllers will be instrumented incrementally in Beta and GA).
- Defining internal span hierarchies for `kube-scheduler` or `kubelet` in this KEP (though both can consume the shared propagation libraries).
- Automatically instrumenting third-party or `controller-runtime` controllers without library adoption.
- Replacing existing metrics, logs, or Kubernetes Events.

## Proposal

End-to-end controller trace propagation works through four cooperating parts:

1. **HTTP Context Propagation to Admission Webhooks (`client-go` and `kube-apiserver`)**:
   When a traced client or controller makes a `Create`, `Update`, or `Patch` call to `kube-apiserver`, `client-go` sends the active span's W3C trace context HTTP headers (`traceparent`, `tracestate`, `baggage`). In [KEP-647], `kube-apiserver` only accepts incoming `traceparent` headers as a parent span for its own internal tracing when the caller belongs to `system:masters` or `system:monitoring`; for all other callers (including controllers running under dedicated ServiceAccounts), `kube-apiserver` starts a new root trace so unprivileged callers cannot force `kube-apiserver` to export internal API server spans. When `kube-apiserver` calls admission webhooks, if `kube-apiserver`'s own span is not sampled (or when forwarding context to webhooks), `kube-apiserver` passes through the incoming request's original `traceparent` header rather than overwriting it with an unsampled internal `kube-apiserver` span header.
2. **Atomic Out-of-Tree Context Injection (`MutatingAdmissionWebhook`)**:
   A mutating admission webhook (hosted as a new `sig-instrumentation` sub-project in `kubernetes-sigs`) receives the `AdmissionReview` HTTP request from `kube-apiserver`, reads the W3C trace context headers from the request, and returns a JSONPatch setting `metadata.annotations["tracing.k8s.io/*"]` on the object. Because injection happens inside admission:
   - The annotation update is persisted atomically alongside the `.spec` mutation, requiring zero extra API calls.
   - The webhook ignores subresource requests (such as `/status`) and `UPDATE` requests where `.spec` did not change, preventing status updates from overwriting trace context or triggering watch loops.
3. **Controller Context Extraction and Span Links (`k8s.io/component-base/tracing`)**:
   When a controller reconciles an object from its informer cache, it starts a new **root span** for the reconciliation pass (named `reconcile <Kind>`, such as `reconcile Deployment`) and attaches an **OpenTelemetry Span Link** pointing to the `SpanContext` extracted from the object's `tracing.k8s.io/*` annotations via `tracing.WithObjects(obj)`.
4. **Multi-Hop Controller Propagation (`kube-controller-manager`)**:
   While the controller's reconcile span is active in `ctx`, any outbound `client-go` calls the controller makes to create or update child objects (for example, the `Deployment` controller creating a `ReplicaSet`, or the `ReplicaSet` controller creating a `Pod`) carry the reconcile span's `traceparent` over HTTP to `kube-apiserver`. `kube-apiserver` forwards that trace context on the `AdmissionReview` call to the mutating admission webhook, which stamps the child object's `metadata.annotations` during admission, forming a linked chain of traces (`kubectl` -> `reconcile Deployment` -> `reconcile ReplicaSet` -> `Pod`).

### User Stories (Optional)

- **Story 1 — Debugging Slow or Failed Workload Rollouts**:
  As a cluster operator, I want to diagnose why a `Deployment` rollout is stalling or taking longer than expected. With `APIServerTracing`, `ControllerManagerTracing`, and the tracing mutating admission webhook enabled, I can inspect the `Deployment` mutation trace and follow Span Links to the `Deployment` controller's `reconcile Deployment` trace and the `ReplicaSet` controller's `reconcile ReplicaSet` trace, pinpointing whether latency occurred in admission, workqueue queuing, or child pod creation.
- **Story 2 — On-Demand Tracing of a Specific Mutation**:
  As a cluster administrator debugging a live cluster where `SamplingRatePerMillion` is set to `0` (or a low baseline rate), I want to send a single API request with a sampled `traceparent` header (`-01`) and have downstream controllers (`Deployment` and `ReplicaSet`) automatically record and export their reconciliation spans for that specific change without restarting `kube-controller-manager`.

### Notes/Constraints/Caveats (Optional)

- **Why OpenTelemetry Span Links Instead of Parent-Child Spans**:
  Kubernetes controllers are level-triggered, coalesce multiple rapid updates into a single workqueue key, and run periodic resyncs over object lifetimes spanning days or weeks. OpenTelemetry specifies [Span Links](https://opentelemetry.io/docs/specs/otel/overview/#links-between-spans) for asynchronous, decoupled, and batched producer-consumer relationships so each reconciliation pass has bounded duration while preserving causal links to the triggering mutation.
- **Multiple Writers Before Reconciliation**:
  Because `metadata.annotations["tracing.k8s.io/traceparent"]` holds the latest `.spec` mutation's context, if two `.spec` updates occur before a controller dequeues the object key, the annotation reflects the most recent `.spec` mutation that produced the current `metadata.generation`.
- **Untraced `.spec` Updates**:
  If an `UPDATE` mutates `.spec` without a sampled trace context, the mutating admission webhook removes any existing `tracing.k8s.io/*` annotations from `metadata.annotations` so the new generation is not falsely linked to an earlier trace.
- **Late `/status` Updates (e.g., Pod Crash Weeks After Creation)**:
  Because `/status` subresource updates are ignored by the admission webhook and controllers do not issue extra API calls to strip `tracing.k8s.io/*` annotations after reconciliation completes, the `tracing.k8s.io/traceparent` annotation from the most recent `.spec` mutation remains on the object until the next `.spec` update. If a status change arrives much later (for example, a Pod crashes a month after creation) or a periodic resync runs, the resulting reconciliation starts a new root trace with current timestamps and attaches a Span Link to the `SpanContext` from that last `.spec` mutation. Because this is a Span Link rather than a parent-child span, it does not reopen the month-old trace (and if the original creation trace has aged out of backend retention, the link simply references an expired trace ID). However, with basic link-aware sampling, if the last `.spec` mutation was sampled (`-01`), subsequent status-driven reconciliations of that object will also be sampled until the next `.spec` mutation. Prior to Beta, we will evaluate bounding link-triggered sampling (for example, only sampling via the link when `metadata.generation != status.observedGeneration` or adopting consistent probability sampling).

### Risks and Mitigations

1. **Infinite Reconciliation Loops from Annotation Updates**:
   - *Risk*: If a controller updated annotations on the object it was reconciling, or if `/status` updates updated annotations, controllers watching that object would re-enqueue it indefinitely.
   - *Mitigation*: Controllers never write `tracing.k8s.io/*` annotations to objects. Injection occurs exclusively in the mutating admission webhook on `CREATE` and `.spec`-mutating `UPDATE` requests, and `/status` subresource updates are ignored.
2. **Control-Plane Disruption if Webhook is Unavailable**:
   - *Risk*: A failing mutating admission webhook could block workload mutations.
   - *Mitigation*: The webhook is deployed with `failurePolicy: Ignore`, `sideEffects: None`, and `timeoutSeconds: 2`. If the webhook is unreachable, mutations succeed without trace annotations.
3. **Span Volume / DoS from Forced Sampling**:
   - *Risk*: Unprivileged clients sending requests with `traceparent` sampled (`-01`) could force control-plane components to export excessive spans.
   - *Mitigation*: `kube-apiserver` continues to restrict parent-child trace sampling of internal API server spans to callers in `system:masters` and `system:monitoring`. Passing the caller's original `traceparent` header through to admission webhooks only allows callers who are already authorized by RBAC to `CREATE` or `UPDATE` a resource (and who could already write `metadata.annotations` directly) to annotate that resource. In `kube-controller-manager`, the OpenTelemetry SDK's bounded batch span processor caps in-memory span buffering and drops spans if the queue fills.

## Design Details

### Annotation Carrier Convention (`tracing.k8s.io/*`)

Rather than hardcoding specific header names, `ObjectCarrier` and the admission webhook dynamically map whatever keys are used by the installed OpenTelemetry `TextMapPropagator` (`tracing.Propagators()`, which configures W3C `TraceContext` and `Baggage`) into `metadata.annotations` under the `tracing.k8s.io/` prefix:

| OpenTelemetry Propagator Key | Kubernetes Annotation Key | Example Value |
| --- | --- | --- |
| `traceparent` | `tracing.k8s.io/traceparent` | `00-4bf92f3577b34da6a3ce929d0e0e4737-00f067aa0ba902b7-01` |
| `tracestate` | `tracing.k8s.io/tracestate` | `ot=th:8` |
| `baggage` | `tracing.k8s.io/baggage` | `key1=val1` |

### Out-of-Tree Context Injection (`sig-instrumentation` Sub-Project)

Context injection is implemented out-of-tree as a Mutating Admission Webhook maintained under a new `sig-instrumentation` sub-project in `kubernetes-sigs`.

When `kube-apiserver` receives an API request from a caller outside `system:masters` and `system:monitoring` (such as a controller authenticating with a ServiceAccount), `kube-apiserver` treats the request as a public endpoint and starts a new root trace for its own internal spans. When `--tracing-config-file` is not configured on `kube-apiserver`, its `NoopTracerProvider` already passes the caller's W3C trace context headers through to admission webhooks. However, when `--tracing-config-file` is configured (`APIServerTracing`, GA in `v1.34`), if `kube-apiserver` did not sample that new root trace, outbound `AdmissionReview` HTTP calls to webhooks previously propagated `kube-apiserver`'s unsampled internal span header, overwriting the caller's original `traceparent`. To make webhook context propagation consistent and allow controllers and other authorized callers to propagate trace context to webhooks without granting them `system:masters` or `system:monitoring` permissions, `k8s.io/apiserver` records the incoming request's remote `SpanContext` on the request context (`k8s.io/apiserver/pkg/endpoints/request`) when `kube-apiserver`'s own server span is unsampled and injects it on outbound admission webhook calls (`k8s.io/apiserver/pkg/util/webhook`).

When the tracing mutating admission webhook receives an `AdmissionReview` request, it behaves as follows:

1. **Ignore Subresources and Non-Desired-State Updates**: Requests targeting subresources (such as `/status`, `/scale`, or `/binding`) or `UPDATE` requests where desired state (top-level fields excluding `metadata` and `status`) did not change are allowed without modification.
2. **Inject Propagated Context or Clear Stale Annotations**: On `CREATE` or desired-state-mutating `UPDATE` requests (which in admission include HTTP `PATCH` requests after patch application), the webhook clears any pre-existing `tracing.k8s.io/*` propagator annotations from prior generations and, if the incoming `AdmissionReview` HTTP request carries a valid (and, when `requireSampled` is enabled, sampled) trace context, returns a JSONPatch setting the propagated `tracing.k8s.io/*` annotations on top-level `metadata.annotations` (never mutating nested pod templates such as `.spec.template.metadata.annotations`, which would alter template hashes and trigger unnecessary Pod rollouts).

### Context Extraction and Span Linking in `k8s.io/component-base/tracing`

`k8s.io/component-base/tracing` provides three primitives for controller instrumentation:
- `ObjectCarrier`: implements OpenTelemetry's `propagation.TextMapCarrier` interface over `metav1.Object` annotations using the `tracing.k8s.io/` prefix for whatever keys the installed `TextMapPropagator` requests.
- `WithObjects`: a stateless `oteltrace.SpanStartOption` helper that extracts valid `SpanContext`s from the current annotations of one or more objects and attaches them as OpenTelemetry Span Links.
- `LinkTracker`: a bounded tracker hooked into a controller's Informer event handlers (`Record` on `Add`/`Update`, `Forget` on `Delete`) so that when multiple updates to the same object are coalesced by the controller's workqueue before a single reconciliation runs, `linkTracker.WithObjects(obj)` links the resulting reconciliation span to **all** coalesced mutations rather than only the latest snapshot:

```go
package tracing

const AnnotationPrefix = "tracing.k8s.io/"

// ObjectCarrier adapts metav1.Object annotations to OpenTelemetry's
// propagation.TextMapCarrier interface using the "tracing.k8s.io/" key prefix.
type ObjectCarrier struct {
	// contains filtered or unexported fields
}

func NewObjectCarrier(obj metav1.Object) ObjectCarrier
func (c ObjectCarrier) Get(key string) string
func (c ObjectCarrier) Set(key, value string)
func (c ObjectCarrier) Keys() []string

// WithObjects extracts SpanContexts from the annotations of the provided
// Kubernetes objects and returns an oteltrace.SpanStartOption that attaches
// them as OpenTelemetry Span Links on the started span.
func WithObjects(objs ...metav1.Object) oteltrace.SpanStartOption

// LinkTracker accumulates SpanContexts observed by Informer event handlers for
// Kubernetes objects so that multiple updates coalesced by a controller's
// workqueue before a single reconciliation are all linked to the resulting
// reconciliation span.
type LinkTracker struct {
	// contains filtered or unexported fields
}

func NewLinkTracker() *LinkTracker
func (t *LinkTracker) Record(obj metav1.Object)
func (t *LinkTracker) Forget(obj metav1.Object)
func (t *LinkTracker) WithObjects(objs ...metav1.Object) oteltrace.SpanStartOption
```

### Link-Aware and Consistent Probability Sampling

When a controller reconciles an object, it starts a new root trace (`ParentContext` is empty and a new `TraceID` is generated) and attaches the triggering mutation's `SpanContext` via `SamplingParameters.Links`. Standard OpenTelemetry samplers—including both `ParentBased` and the consistent probability samplers (`ComposableParentThreshold` and `ComposableProbability`)—only inspect `ParentContext` and the new span's `TraceID`, ignoring `Links`. Without a link-aware sampler:

1. **Multi-hop sampling probability degrades as $p^N$**: If each controller hop samples independently using its own newly generated `TraceID` at rate $p$ (`SamplingRatePerMillion`), a 3-hop chain (`Deployment` -> `ReplicaSet` -> `Pod`) at $1\%$ sampling is captured end-to-end only $0.0001\%$ of the time.
2. **On-demand tracing (`SamplingRatePerMillion: 0`) is dropped**: When `SamplingRatePerMillion` is `0`, `ParentBased` sees an empty `ParentContext` on the controller's root span and falls back to `NeverSample()`, dropping the trace even when the triggering mutation was explicitly sampled (`-01`).

To solve both problems, `k8s.io/component-base/tracing` wraps the root sampler with a link-aware sampler for internal controller reconciliation spans (excluding `SpanKindServer` public-endpoint HTTP server spans in `kube-apiserver`, where `otelhttp` attaches untrusted incoming HTTP headers as links): when starting an internal root span (where `ParentContext` has no valid span), if any linked `SpanContext` in `SamplingParameters.Links` is sampled, the new root span is sampled; otherwise, it delegates to the configured `SamplingRatePerMillion` root sampler.

When multiple sampled links are attached to a single reconciliation span (either from multiple objects or from coalesced updates tracked via `LinkTracker`), the link-aware sampler combines their `TraceState` deterministically rather than picking an arbitrary link:
- If all sampled links share a single `TraceID`, that trace's `TraceState` is preserved.
- If sampled links originate from multiple distinct `TraceID`s, vendor-specific `TraceState` entries are not copied from an arbitrary trace onto the new root span (each `Link` retains its own `TraceState`).
- When sampled links carry W3C/OpenTelemetry Consistent Probability Sampling thresholds (`ot=th:...`), the sampler combines them by taking the **minimum rejection threshold** (`min(th)`, representing the maximum sampling probability among the linked traces) for the new root span's `ot=th:...` entry.

Once OpenTelemetry's consistent probability samplers stabilize in `go.opentelemetry.io/otel/sdk/trace`, the link-aware sampler can evaluate the W3C `ot` `tracestate` sub-key (`th` threshold and `rv` randomness value) from linked `SpanContext`s when present so components configured with different nonzero `SamplingRatePerMillion` rates make consistent sampling decisions across Span Links, while still honoring explicit on-demand sampled links (`-01` / `th:0`) when `SamplingRatePerMillion` is `0` without requiring the admission webhook to stamp `rv` annotations on unsampled cluster writes.

### `kube-controller-manager` Configuration and Alpha Instrumentation

1. **Flag and Component Configuration**:
   `kube-controller-manager` adds a `--tracing-config-file` flag and `TracingConfiguration` in `GenericControllerManagerConfiguration` (matching `kube-apiserver` in [KEP-647] and `kubelet` in [KEP-2831]). When `ControllerManagerTracing` is enabled, `kube-controller-manager` initializes its OpenTelemetry `TracerProvider` and wraps its Kubernetes client transports (`k8s.io/client-go`) so outbound API requests automatically propagate the active span context in HTTP headers.
2. **Span Naming and Semantic Conventions**:
   Controller reconciliation spans follow OpenTelemetry span naming guidelines (`reconcile <Kind>`, such as `reconcile Deployment` and `reconcile ReplicaSet`), reuse existing OpenTelemetry `k8s.*` resource semantic conventions (`k8s.namespace.name`, `k8s.deployment.name`, `k8s.deployment.uid`, `k8s.replicaset.name`, `k8s.replicaset.uid`, etc. from `go.opentelemetry.io/otel/semconv`), and will adopt `k8s.controller.*` semantic conventions as they are standardized in `open-telemetry/semantic-conventions`.
3. **Alpha Controller Instrumentation (`Deployment` and `ReplicaSet`)**:
   For Alpha (`v1.39`), the `Deployment` and `ReplicaSet` controllers in `kube-controller-manager` record incoming object updates in a `tracing.LinkTracker` within their Informer event handlers (`Record` on add/update, `Forget` on delete) and start a reconciliation span at their sync entrypoints:
   ```go
   ctx, span := dc.tracer.Start(ctx, "reconcile Deployment",
       dc.linkTracker.WithObjects(d),
       oteltrace.WithAttributes(
           semconv.K8SDeploymentUID(string(d.UID)),
           semconv.K8SDeploymentName(d.Name),
           semconv.K8SNamespaceName(d.Namespace),
       ),
   )
   defer span.End()
   ```
   When the controller creates or updates child `ReplicaSet` or `Pod` objects using `ctx`, `client-go` propagates the active reconcile span's `traceparent` header to `kube-apiserver`.

### Test Plan

[x] I/we understand the owners of the involved components may require updates to
existing tests to make this code solid enough prior to committing the changes necessary
to implement this enhancement.

##### Prerequisite testing updates

None.

##### Unit tests

- `k8s.io/component-base/tracing`:
  - Unit tests for `ObjectCarrier` (`Get`, `Set`, `Keys`, nil-object handling, and propagator round-tripping without hardcoded header names).
  - Unit tests for `WithObjects`, `LinkTracker` (verifying that multiple updates coalesced before a single reconcile are all linked and deduplicated), and link-aware sampling (verifying that root spans with sampled links are sampled when `SamplingRatePerMillion` is `0`, `SpanKindServer` public-endpoint links are excluded, and multi-link `TraceState` / `ot=th:...` minimum thresholds are combined deterministically).
- `k8s.io/apiserver`:
  - Unit tests in `k8s.io/apiserver/pkg/endpoints/filters` and `k8s.io/apiserver/pkg/util/webhook` verifying that incoming trace context headers from non-privileged callers are passed through to admission webhooks without forcing internal `kube-apiserver` sampling.
- `pkg/controller/deployment` and `pkg/controller/replicaset`:
  - Unit tests verifying that `reconcile Deployment` and `reconcile ReplicaSet` root spans are created with Span Links matching `tracing.k8s.io/traceparent` on the reconciled objects.

Current unit test coverage of touched packages:

- `k8s.io/component-base/tracing`: `2026-11-09` - `78.4%`
- `k8s.io/apiserver/pkg/endpoints/filters`: `2026-11-09` - `75.8%`
- `k8s.io/apiserver/pkg/util/webhook`: `2026-11-09` - `71.4%`
- `k8s.io/kubernetes/cmd/kube-controller-manager/app`: `2026-11-09` - `52.1%`
- `k8s.io/kubernetes/pkg/controller/deployment`: `2026-11-09` - `74.2%`
- `k8s.io/kubernetes/pkg/controller/replicaset`: `2026-11-09` - `76.1%`

##### Integration tests

- `test/integration/controllermanager/tracing`:
  - Integration test running `kube-apiserver` (`APIServerTracing=true`), the mutating admission webhook handler, and `kube-controller-manager` (`ControllerManagerTracing=true`) with an in-process OTLP test collector.
  - Verifies that creating a `Deployment` with a sampled `traceparent` header stamps `tracing.k8s.io/traceparent` on the `Deployment`, `ReplicaSet`, and `Pod` objects, exports linked `reconcile Deployment` and `reconcile ReplicaSet` traces (even when controllers authenticate with non-privileged ServiceAccounts), and ignores `/status` updates without triggering extra reconciliations.

##### e2e tests

- E2E test under `test/e2e/instrumentation` verifying `kube-controller-manager` health and `Deployment` -> `ReplicaSet` -> `Pod` rollout with `ControllerManagerTracing` enabled.

### Graduation Criteria

#### Alpha

- `ObjectCarrier`, `WithObjects`, `LinkTracker`, and link-aware sampling implemented in `k8s.io/component-base/tracing` with unit tests.
- Incoming caller `SpanContext` passthrough to admission webhooks implemented in `k8s.io/apiserver`.
- `ControllerManagerTracing` feature gate and `--tracing-config-file` implemented in `kube-controller-manager`.
- `Deployment` and `ReplicaSet` controllers instrumented and verified via integration tests.
- Out-of-tree mutating admission webhook sub-project established under `sig-instrumentation` in `kubernetes-sigs`.

#### Beta

- Instrument additional controllers in `kube-controller-manager` (`StatefulSet`, `DaemonSet`, `Job`) and demonstrate end-to-end linked traces across at least two components (e.g., `kube-controller-manager` and `kubelet`).
- Evaluate consistent probability sampling and bounding link-triggered sampling for late `/status` updates on long-lived objects.
- Verify in scalability benchmarks that enabling `ControllerManagerTracing` and the mutating admission webhook stays within Kubernetes scalability thresholds.
- Publish user-facing documentation and deployment examples for the mutating admission webhook and OpenTelemetry Collector.

#### GA

- Feedback from production users collected and addressed across at least two releases after Beta.
- Align controller span attributes with OpenTelemetry `k8s.controller.*` semantic conventions.
- Graduate `ControllerManagerTracing` feature gate to GA and lock it to default-enabled (with span export remaining opt-in via `--tracing-config-file`).

### Upgrade / Downgrade Strategy

- **Upgrade**: Enabling `ControllerManagerTracing` without `--tracing-config-file` leaves span export disabled. Enabling both starts exporting spans to the configured OTLP endpoint. Deploying the optional mutating admission webhook begins stamping `tracing.k8s.io/*` annotations on new `.spec` mutations.
- **Downgrade**: Disabling `ControllerManagerTracing` or removing `--tracing-config-file` stops span creation and export upon `kube-controller-manager` restart. Any `tracing.k8s.io/*` annotations already persisted on objects are inert metadata and are ignored by un-instrumented or downgraded controllers.

### Version Skew Strategy

- `tracing.k8s.io/*` annotations use standard W3C Trace Context strings inside `metadata.annotations`. A newer `kube-apiserver` or webhook and an older `kube-controller-manager` (or vice versa) interoperate safely: older components ignore the annotations, while newer components extract them when present.

## Production Readiness Review Questionnaire

### Feature Enablement and Rollback

###### How can this feature be enabled / disabled in a live cluster?

- [x] Feature gate (also fill in values in `kep.yaml`)
  - Feature gate name: `ControllerManagerTracing`
  - Components depending on the feature gate: `kube-controller-manager`
- [x] Other
  - Describe the mechanism: Pass `--tracing-config-file` to `kube-controller-manager` (or configure `TracingConfiguration` in `GenericControllerManagerConfiguration`), matching [KEP-647] and [KEP-2831]:
    - When `ControllerManagerTracing` is disabled: `kube-controller-manager` does not generate spans, initiate an OTLP connection, or propagate context.
    - When `ControllerManagerTracing` is enabled without `--tracing-config-file`: `kube-controller-manager` uses a no-op `TracerProvider` and does not initiate an OTLP connection or export spans.
    - When `ControllerManagerTracing` is enabled with `--tracing-config-file` and `samplingRatePerMillion: 0` (the default): `kube-controller-manager` initiates an OTLP connection and exports spans only when a linked or parent span context has the sampled flag set.
    - When `ControllerManagerTracing` is enabled with `--tracing-config-file` and `samplingRatePerMillion > 0`: `kube-controller-manager` exports spans at the configured sampling rate as well as whenever a linked or parent span context is sampled.
    - Out-of-tree context injection is enabled or disabled independently by applying or deleting the `MutatingWebhookConfiguration` for the `sig-instrumentation` tracing webhook.
  - Will enabling / disabling the feature require downtime of the control plane? Restarting `kube-controller-manager` is required to toggle the feature gate or `--tracing-config-file` flag (in HA control planes, standby replicas take over leadership). Enabling or disabling the mutating admission webhook requires no control-plane restart.
  - Will enabling / disabling the feature require downtime or reprovisioning of a node? No.

###### Does enabling the feature change any default behavior?

No. When `ControllerManagerTracing` is enabled without `--tracing-config-file`, no spans are recorded or exported. When `--tracing-config-file` is provided, `kube-controller-manager` exports trace telemetry asynchronously without changing reconciliation semantics.

###### Can the feature be disabled once it has been enabled (i.e. can we roll back the enablement)?

Yes, by removing `--tracing-config-file` or setting `--feature-gates=ControllerManagerTracing=false` and restarting `kube-controller-manager`, and/or deleting the `MutatingWebhookConfiguration`.

###### What happens if we reenable the feature if it was previously rolled back?

`kube-controller-manager` resumes generating and exporting spans.

###### Are there any tests for feature enablement/disablement?

Unit and integration tests will verify behavior with `ControllerManagerTracing` enabled and disabled, as well as with and without `--tracing-config-file`.

### Rollout, Upgrade and Rollback Planning

###### How can a rollout or rollback fail? Can it impact already running workloads?

- If `kube-controller-manager` is given a malformed `--tracing-config-file`, it will fail validation on startup and log a configuration error; existing running workloads are unaffected.
- If the OTLP collector endpoint is unreachable, the OpenTelemetry SDK drops spans after filling its bounded in-memory queue without blocking reconciliation loops.
- If the out-of-tree mutating admission webhook is unreachable, `failurePolicy: Ignore` ensures API requests proceed normally without trace annotations.

###### What specific metrics should inform a rollback?

- Existing controller workqueue and API server metrics: `workqueue_depth`, `workqueue_queue_duration_seconds`, `workqueue_work_duration_seconds`, `apiserver_request_duration_seconds`, and `apiserver_admission_webhook_admission_duration_seconds`.
- Process resource metrics (`process_resident_memory_bytes`, `process_cpu_seconds_total`) on `kube-controller-manager`.

###### Were upgrade and rollback tested? Was the upgrade->downgrade->upgrade path tested?

Will be tested on a local cluster prior to Beta graduation by enabling `ControllerManagerTracing` and the mutating admission webhook, verifying linked spans are exported during a `Deployment` rollout, disabling the feature gate, and re-enabling it.

###### Is the rollout accompanied by any deprecations and/or removals of features, APIs, fields of API types, flags, etc.?

No.

### Monitoring Requirements

###### How can an operator determine if the feature is in use by workloads?

Check the `kubernetes_feature_enabled{name="ControllerManagerTracing"}` metric on `kube-controller-manager` and observe spans with `service.name="kube-controller-manager"` in the configured tracing backend.

###### How can someone using this feature know that it is working for their instance?

- [x] Other (treat as last resort)
  - Details: Inspect the configured OTLP tracing backend for `reconcile Deployment` and `reconcile ReplicaSet` spans from `kube-controller-manager` with Span Links to the triggering mutation spans, and check `metadata.annotations["tracing.k8s.io/traceparent"]` on mutated objects when the webhook is deployed.

###### What are the reasonable SLOs (Service Level Objectives) for the enhancement?

N/A (match [KEP-647]).

###### What are the SLIs (Service Level Indicators) an operator can use to determine the health of the service?

- [x] Other (treat as last resort)
  - Details: Presence of exported spans in the tracing backend, alongside existing `workqueue_work_duration_seconds` and `apiserver_admission_webhook_admission_duration_seconds` metrics.

###### Are there any missing metrics that would be useful to have to improve observability of this feature?

No new custom Kubernetes metrics are required; OpenTelemetry Go SDK self-observability metrics (`otel.sdk.span.*` / exporter metrics) cover span generation and export health.

### Dependencies

###### Does this feature depend on any specific services running in the cluster?

- **OpenTelemetry Collector or OTLP-compatible backend (optional)**:
  - Usage description: Receives exported OTLP gRPC spans from `kube-controller-manager`.
    - Impact of its outage on the feature: Spans are buffered up to the SDK batch queue limit and then dropped; controller reconciliation continues normally.
    - Impact of its degraded performance or high-error rates on the feature: Spans may be delayed or dropped before reaching the tracing backend.
- **Tracing Mutating Admission Webhook (optional, out-of-tree `sig-instrumentation` sub-project)**:
  - Usage description: Injects `tracing.k8s.io/*` annotations during `.spec`-mutating admission requests.
    - Impact of its outage on the feature: With `failurePolicy: Ignore`, API requests succeed without `tracing.k8s.io/*` annotations; `kube-controller-manager` still exports spans when `SamplingRatePerMillion > 0`, but without Span Links to the prior hop.
    - Impact of its degraded performance or high-error rates on the feature: Bounded by `timeoutSeconds: 2` and `failurePolicy: Ignore`.

### Scalability

###### Will enabling / using this feature result in any new API calls?

No. Because context injection is performed by the mutating admission webhook during the existing `CREATE`/`UPDATE` request admission phase, zero additional Kubernetes API calls are made. (When the mutating admission webhook is deployed, `kube-apiserver` makes a webhook `AdmissionReview` call on matching `CREATE`/`UPDATE` requests, excluding `/status` updates.)

###### Will enabling / using this feature result in introducing new API types?

No new cluster API resources are introduced (`TracingConfiguration` in `k8s.io/component-base/tracing/api/v1` already exists from [KEP-647] and [KEP-2831]).

###### Will enabling / using this feature result in any new calls to the cloud provider?

Not directly. Operators may configure their OpenTelemetry Collector to export traces to a cloud provider's tracing backend.

###### Will enabling / using this feature result in increasing size or count of the existing API objects?

- API type(s): Objects created or updated while a sampled trace context is active and the mutating admission webhook is deployed.
- Estimated increase in size: ~83 bytes per sampled object for `"tracing.k8s.io/traceparent": "00-<32-hex>-<16-hex>-01"` (plus optional `tracestate` / `baggage`). Unsampled requests do not add annotations.
- Estimated amount of new objects: 0.

###### Will enabling / using this feature result in increasing time taken by any operations covered by existing SLIs/SLOs?

In `kube-controller-manager`, extracting annotations and recording an in-memory span adds <1 microsecond per reconciliation pass; span export happens asynchronously via the OpenTelemetry SDK's batch span processor. When the optional mutating admission webhook is deployed, `.spec`-mutating `CREATE`/`UPDATE` requests incur a local cluster webhook round-trip (~0.5–2ms).

###### Will enabling / using this feature result in non-negligible increase of resource usage (CPU, RAM, disk, IO, ...) in any components?

`kube-controller-manager` uses the same bounded batch span processor queue as `kube-apiserver` ([KEP-647]), which consumes at most a few megabytes of memory under full queue load.

###### Can enabling / using this feature result in resource exhaustion of some node resources (PIDs, sockets, inodes, etc.)?

No. `kube-controller-manager` maintains a single gRPC client connection to the configured OTLP collector endpoint.

### Troubleshooting

###### How does this feature react if the API server and/or etcd is unavailable?

`kube-controller-manager` tracing operates in memory and exports to the configured OTLP collector; it does not add new failure modes when `kube-apiserver` or `etcd` is unavailable.

###### What are other known failure modes?

- **OTLP Collector endpoint unreachable or misconfigured**:
  - Detection: Absence of `kube-controller-manager` spans in the tracing backend and exporter connection warnings in `kube-controller-manager` logs.
  - Mitigations: Fix `--tracing-config-file` endpoint or disable `ControllerManagerTracing`.
  - Diagnostics: `kube-controller-manager` logs emitted by the OpenTelemetry OTLP exporter.
  - Testing: Covered by `k8s.io/component-base/tracing` configuration validation unit tests.

###### What steps should be taken if SLOs are not being met to determine the problem?

1. If admission latency (`apiserver_admission_webhook_admission_duration_seconds`) increases, remove the `MutatingWebhookConfiguration` for the tracing admission webhook.
2. If controller reconciliation latency (`workqueue_work_duration_seconds`) or memory usage increases, remove `--tracing-config-file` or set `--feature-gates=ControllerManagerTracing=false` on `kube-controller-manager`.

## Implementation History

- `2026-02-12`: Initial KEP tracking issue [#5915](https://github.com/kubernetes/enhancements/issues/5915) and draft PR [#5916](https://github.com/kubernetes/enhancements/pull/5916) opened.
- `2026-11-10`: Updated KEP design to use out-of-tree mutating admission webhook injection (`sig-instrumentation` sub-project), original caller `traceparent` passthrough to webhooks, `tracing.WithObjects`, OpenTelemetry `reconcile <Kind>` span naming and `semconv` attributes, and Alpha instrumentation of `Deployment` and `ReplicaSet` controllers in `kube-controller-manager` targeting `v1.39`.

## Drawbacks

- Depending on the configured sampling rate, recording and exporting spans adds a small CPU and memory overhead to `kube-controller-manager`, and deploying the mutating admission webhook adds ~83 bytes of annotation data to sampled objects.

## Alternatives

### Parent-Child Spans Across Reconciliations

Instead of starting a root span per reconciliation pass with an OpenTelemetry Span Link (`tracing.WithObjects`), controllers could extract `tracing.k8s.io/traceparent` as a remote parent and continue the original `TraceID`. This was rejected because controllers coalesce multiple mutations in workqueues (where a single parent cannot represent multiple causes) and run asynchronous loops over hours or days, which exceeds trace backend retention/tail-sampling windows.

### Controller-Side Annotation Injection

Instead of using an out-of-tree Mutating Admission Webhook, each controller could manually inject `tracing.k8s.io/traceparent` into child objects or patch watched objects in controller code. This was rejected because:
- External clients (`kubectl`, GitOps controllers) and un-instrumented controllers would still require an admission webhook to stamp initial mutations.
- Controllers operating via subresources (such as `pods/binding`) lack RBAC permissions or API support to patch `metadata.annotations` in the same request, which would require an extra API call per reconciliation.
- Multiple controllers patching the same annotation on shared objects can cause Server-Side Apply (`managedFields`) conflicts.

### In-Tree Admission Plugin in `kube-apiserver`

Annotation injection could be implemented as a built-in admission plugin inside `kube-apiserver`. This was rejected because SIG API Machinery discourages new in-tree admission plugins when an out-of-tree `MutatingAdmissionWebhook` can achieve the same behavior using `kube-apiserver` webhook trace propagation.

### First-Class `ObjectMeta` Trace Context Field

Adding a dedicated `traceContext` field to `metav1.ObjectMeta` was rejected because `ObjectMeta` changes affect every Kubernetes API schema and wire serialization, whereas `metadata.annotations` is the standard extension point for non-identifying operational metadata.

## Infrastructure Needed (Optional)

- A new `sig-instrumentation` sub-project repository in `kubernetes-sigs` (e.g., `kubernetes-sigs/tracing-admission-webhook`) to host the out-of-tree mutating admission webhook and its container image build/release configuration.
