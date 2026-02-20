# KEP-5927: Distributed Tracing in Kube-Scheduler

<!-- toc -->
- [Release Signoff Checklist](#release-signoff-checklist)
- [Summary](#summary)
- [Motivation](#motivation)
  - [Goals](#goals)
  - [Non-Goals](#non-goals)
- [Proposal](#proposal)
  - [User Stories](#user-stories)
    - [Story 1: Debugging High Scheduling Latency](#story-1-debugging-high-scheduling-latency)
    - [Story 2: Identifying Why a Pod Is Stuck in Pending](#story-2-identifying-why-a-pod-is-stuck-in-pending)
    - [Story 3: End-to-End Pod Lifecycle Tracing](#story-3-end-to-end-pod-lifecycle-tracing)
  - [Notes/Constraints/Caveats](#notesconstraintscaveats)
  - [Risks and Mitigations](#risks-and-mitigations)
- [Design Details](#design-details)
  - [Trace Structure](#trace-structure)
  - [Configuration](#configuration)
  - [Context Propagation](#context-propagation)
  - [Implementation](#implementation)
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
  - [Extending utiltrace Instead of OpenTelemetry](#extending-utiltrace-instead-of-opentelemetry)
  - [Generic Automatic Instrumentation](#generic-automatic-instrumentation)
  - [External Profiling Instead of Tracing](#external-profiling-instead-of-tracing)
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

This proposal adds OpenTelemetry distributed tracing to Kube-Scheduler. While the Kubernetes API Server already has tracing support (KEP-647), the Scheduler — the core brain of the cluster responsible for all Pod placement decisions — remains opaque. Current observability is limited to log-based `utiltrace`, which is isolated and cannot be correlated with the wider request lifecycle.

This KEP instruments the `schedulingCycle` and `bindingCycle` with OpenTelemetry spans, generates child spans for individual plugin execution (e.g., `RunFilterPlugins`, `RunScorePlugins`), and ensures trace context is available to custom (out-of-tree) plugins. The implementation reuses the existing `TracingConfiguration` API and `component-base` tracing libraries established by KEP-647, providing a consistent operator experience across all key Kubernetes components.

## Motivation

While the Kubernetes API Server has tracing support (KEP-647), Kube-Scheduler remains a black box from a tracing perspective. Current tooling like `utiltrace` is log-based and isolated — it does not allow operators to visualize the scheduling attempt in the context of the wider request lifecycle (e.g., from an HTTP client request through the API Server and into the Scheduler).

Operators currently struggle to answer critical performance questions:

* **Why is this Pod stuck in Pending?** Is it a specific plugin rejecting it? Which filter plugin is returning `Unschedulable`?
* **Why is scheduling latency high?** Is a custom Score plugin slow? Is the API round-trip slow during binding?
* **How long does the Binding cycle take compared to the Scheduling cycle?** Are there bottlenecks in the preBind or postBind phases?

These questions are fundamental to operating Kubernetes clusters at scale, and today they can only be answered through painstaking log analysis and metrics correlation.

### Goals

* **Instrument the Scheduling Framework:** Add OpenTelemetry spans to the core `schedulingCycle` and `bindingCycle`, providing per-phase latency visibility.
* **Plugin Visibility:** Generate child spans for individual plugin execution (e.g., `RunFilterPlugins`, `RunScorePlugins`) to expose per-plugin latency. Ensure the trace context is available to custom plugins (out-of-tree), allowing platform engineers to trace their own proprietary scheduling logic.
* **Scheduling Queue Observability:** Measure scheduling queue latency — the duration a Pod remains Pending in the active queue before the scheduling cycle begins.
* **Consistency:** Reuse the existing `TracingConfiguration` API and `component-base` tracing libraries established by KEP-647 (API Server Tracing) and KEP-2831 (Kubelet Tracing).
* **Context Propagation:** Link Scheduler traces back to the original Pod creation trace by extracting W3C Trace Context from Pod annotations (as detailed in [KEP-5915](/keps/sig-instrumentation/5915-async-trace-context-propagation/README.md)), enabling full end-to-end lifecycle tracing.

### Non-Goals

* Modifying the Kubernetes API to add tracing context as a first-class field.
* Instrumenting other components (API Server and Kubelet are handled by KEP-647 and KEP-2831 respectively).
* Replacing existing logging, metrics, or the events API.
* Defining how observability backends should visualize traces.
* Implementing automatic instrumentation for out-of-tree scheduler plugins (they can opt into tracing via the context passed to plugin methods).

## Proposal

I propose instrumenting the kube-scheduler with OpenTelemetry tracing. The instrumentation will follow the same patterns and libraries as the established API Server tracing (KEP-647), ensuring consistency across Kubernetes components.

The core idea is to create a trace hierarchy that mirrors the Scheduling Framework's execution phases. Each scheduling attempt for a Pod will produce a root span (`SchedulePod`) with child spans for each phase (`schedulingCycle`, `bindingCycle`) and further child spans for individual plugin invocations. This structure allows operators to drill into exactly which phase or plugin is contributing to latency. Operators can correlate multiple scheduling attempts for the same Pod by filtering on the `k8s.pod.uid` attribute.

### User Stories

#### Story 1: Debugging High Scheduling Latency

As a cluster operator, I notice that Pod scheduling latency (as reported by the `scheduler_scheduling_attempt_duration_seconds` metric) has increased. I open my tracing backend (e.g., Jaeger, Tempo) and search for `SchedulePod` spans with high duration. I can immediately see that the `score` phase is taking 80% of the scheduling cycle time, and drill into the child spans to discover that a custom Score plugin (`MyCustomScorer`) is making an expensive external API call on every invocation. I can then work with the plugin author to optimize or cache the call.

#### Story 2: Identifying Why a Pod Is Stuck in Pending

As a platform engineer, a user reports that their Pod has been Pending for 10 minutes. I search for traces related to this Pod and see multiple scheduling attempts, each ending with a `filter` phase that marks the Pod as Unschedulable. The per-plugin spans show that `NodeResourcesFit` is rejecting every node because the Pod's resource request exceeds available capacity. This immediately tells me the cluster needs to scale up or the resource request needs to be adjusted, without requiring me to manually parse scheduler logs.

#### Story 3: End-to-End Pod Lifecycle Tracing

As an SRE, I want to trace the full lifecycle of a Pod from creation to running. By combining API Server tracing (KEP-647) with Scheduler tracing (this KEP) and Kubelet tracing (KEP-2831), and using W3C Trace Context propagation via annotations ([KEP-5915](/keps/sig-instrumentation/5915-async-trace-context-propagation/README.md)), I can correlate traces across components via Span Links:

1. User creates Pod — API Server injects its trace context into Pod annotations (API Server span)
2. Scheduler picks up Pod, extracts trace context from annotations, and runs scheduling cycle (Scheduler span, linked to the API Server span)
3. Scheduler binds Pod to a node and re-injects its own trace context into Pod annotations (Scheduler span)
4. Kubelet syncs and starts the Pod, extracts trace context from annotations (Kubelet span, linked to the Scheduler span)

### Notes/Constraints/Caveats

* **`utiltrace` Coexistence:** The existing `utiltrace` package in the scheduler provides log-based latency tracking. The `component-base/tracing` package already provides a [`tracing.Start()`](https://github.com/kubernetes/kubernetes/blob/04d87a4b6e72336ee9afb1e5b477223c96a8fcbb/staging/src/k8s.io/component-base/tracing/tracing.go#L33) function that creates **both** an OpenTelemetry span and a `utiltrace` span simultaneously, wrapped in a unified `tracing.Span` type. Calling `span.End(logThreshold)` ends the OTel span and calls `utilSpan.LogIfLong(threshold)` under the hood, preserving existing log-based latency warnings. This unified API will be used throughout, so existing `utiltrace` behavior is preserved automatically — not replaced.
* **Sampling:** At high Pod throughput, generating a trace for every scheduling attempt could be expensive. This proposal relies on the `TracingConfiguration`'s `samplingRatePerMillion` field for head-based sampling (the decision is made at span creation time), consistent with the API Server's approach. Operators who need more sophisticated trace volume management — such as tail-based sampling, where the decision is made after the span is complete based on its content (e.g., only keeping traces for failed scheduling attempts or high-latency outliers) — can configure this in their collection pipeline (e.g., the OpenTelemetry Collector's `tail_sampling` processor). This is outside the scope of the scheduler itself.
* **Performance:** Tracing instrumentation must not add measurable latency to the scheduling critical path. The OpenTelemetry SDK is designed to be low-overhead, and unsampled spans are non-recording, making attribute and event calls no-ops.
* **KEP-5915 Dependency:** The context propagation (Span Links from Pod annotations) depends on [KEP-5915](/keps/sig-instrumentation/5915-async-trace-context-propagation/README.md) landing in `component-base`. However, the core scheduler instrumentation (phase and plugin spans) is independently valuable and does **not** depend on KEP-5915. If KEP-5915 lands first, the scheduler gets full end-to-end tracing from day one. If the scheduler tracing lands first, it still provides per-phase and per-plugin observability — the Span Links are added later when KEP-5915 is available, with no code changes needed beyond importing the helpers.

### Risks and Mitigations

* **Risk: Performance Regression in Scheduling Latency**
  - Mitigations: Use the OpenTelemetry SDK which is designed for low-overhead production use. Spans that are not sampled have negligible cost. Benchmark before and after instrumentation to ensure no measurable regression. The feature gate allows disabling if any issues arise.

* **Risk: Excessive Trace Data Volume**
  - Mitigations: The `TracingConfiguration` sampling rate controls how many scheduling attempts produce traces. Operators can start with a low sampling rate (e.g., 1%) and increase as needed.

* **Risk: Breaking Custom Plugins**
  - Mitigations: Trace context propagation through plugin methods uses the existing Go `context.Context` parameter. Plugins that do not use tracing are unaffected. The context enrichment is purely additive.

* **Risk: Dependency on OpenTelemetry SDK**
  - Mitigations: The kube-scheduler already depends on `component-base` which includes OpenTelemetry dependencies used by [KEP-647](https://github.com/kubernetes/enhancements/blob/master/keps/sig-instrumentation/647-apiserver-tracing/README.md) (API Server Tracing, Stable) and [KEP-2831](https://github.com/kubernetes/enhancements/blob/master/keps/sig-instrumentation/2831-kubelet-tracing/README.md) (Kubelet Tracing, Stable). The OpenTelemetry-Go SDK itself has reached GA. No new external dependencies are introduced.

## Design Details

### Trace Structure

The trace hierarchy mirrors the Scheduling Framework's phases:

```
[Span] SchedulePod (Root)
 ├── [Span] schedulingCycle
 │    ├── [Span] preFilter
 │    ├── [Span] filter
 │    │    ├── [Span] plugin: NodeResourcesFit
 │    │    ├── [Span] plugin: TaintToleration
 │    │    └── ...
 │    ├── [Span] postFilter (if filter fails)
 │    ├── [Span] preScore
 │    ├── [Span] score
 │    │    ├── [Span] plugin: NodeResourcesBalancedAllocation
 │    │    └── ...
 │    ├── [Span] reserve
 │    └── [Span] permit
 └── [Span] bindingCycle
      ├── [Span] preBind
      ├── [Span] bind
      └── [Span] postBind
```

**Span Attributes:** Each span will include relevant attributes:

* `SchedulePod`: `k8s.pod.name`, `k8s.pod.namespace`, `k8s.pod.uid`
* Phase spans: `scheduler.phase`, `scheduler.profile`
* Plugin spans: `scheduler.plugin.name`, `scheduler.plugin.extension_point`
* Result spans: `scheduler.result` (e.g., `Success`, `Unschedulable`, `Error`), `scheduler.selected_node` (on bind)

**Queue Wait Time:** The root `SchedulePod` span will start when the Pod is dequeued from the active queue, and will include an attribute `scheduler.queue_wait_ms` recording how long the Pod waited in the queue. This allows operators to distinguish queue wait time from actual scheduling compute time.

### Configuration

This proposal reuses the existing `TracingConfiguration` struct defined in `k8s.io/component-base/tracing/api/v1` — the same struct used by the API Server ([KEP-647](https://github.com/kubernetes/enhancements/blob/master/keps/sig-instrumentation/647-apiserver-tracing/README.md)) and the Kubelet ([KEP-2831](https://github.com/kubernetes/enhancements/blob/master/keps/sig-instrumentation/2831-kubelet-tracing/README.md)). Both of those implementations have reached Stable, validating this configuration pattern.

The `TracingConfiguration` is embedded in `KubeSchedulerConfiguration`, following the same pattern as the Kubelet's `KubeletConfiguration`:

```go
// KubeSchedulerConfiguration contains the configuration for the Scheduler
type KubeSchedulerConfiguration struct {
    metav1.TypeMeta `json:",inline"`
    // ...existing fields...

    // +optional
    // TracingConfiguration specifies configuration for tracing.
    Tracing tracingapi.TracingConfiguration `json:"tracing,omitempty"`
}
```

The `TracingConfiguration` struct provides two fields:

```go
// TracingConfiguration provides versioned configuration for OpenTelemetry tracing clients.
type TracingConfiguration struct {
    // Endpoint of the collector this component will report traces to.
    // The connection is insecure, and does not currently support TLS.
    // Recommended is unset, and endpoint is the otlp grpc default, localhost:4317.
    // +optional
    Endpoint *string `json:"endpoint,omitempty"`

    // SamplingRatePerMillion is the number of samples to collect per million spans.
    // Recommended is unset. If unset, sampler respects its parent span's sampling
    // rate, but otherwise never samples.
    // +optional
    SamplingRatePerMillion *int32 `json:"samplingRatePerMillion,omitempty"`
}
```

Example scheduler component config:

```yaml
apiVersion: kubescheduler.config.k8s.io/v1
kind: KubeSchedulerConfiguration
...
tracing:
  # OTLP gRPC endpoint
  endpoint: "otel-collector.observability.svc:4317"
  # samplingRatePerMillion controls the sampling rate (10000 = 1%)
  samplingRatePerMillion: 10000
```

This provides a consistent operator experience across all Kubernetes components that support tracing.

**Future: OTel Declarative Configuration.** The current `TracingConfiguration` struct is intentionally minimal (endpoint + sampling rate). The [OpenTelemetry Declarative Configuration](https://github.com/open-telemetry/opentelemetry-specification/blob/main/specification/configuration/README.md#programmatic) format (currently at 1.0 release candidate) would provide significantly more control over the SDK — custom samplers, batch processor tuning, exporter configuration, TLS settings, etc. Introducing a v2 `TracingConfiguration` that adopts this format is a cross-cutting concern affecting all traced components (API Server, Scheduler, Kubelet) and is better addressed in a separate proposal. This KEP uses the existing v1 struct for consistency with the current ecosystem.

### Context Propagation

To link Scheduler traces back to the original creation of the Pod, this proposal extracts the W3C Trace Context from Pod annotations using the helper functions proposed in [KEP-5915: Standardizing Async Trace Context Propagation](/keps/sig-instrumentation/5915-async-trace-context-propagation/README.md). KEP-5915 adds `InjectContext`, `ExtractContext`, and `StartReconcileSpan` primitives to `k8s.io/component-base/tracing` that standardize how controllers propagate trace context across asynchronous boundaries using object annotations and OpenTelemetry Span Links.

**The Scheduler as the first adopter of KEP-5915:** The kube-scheduler is a natural first consumer of this pattern. It is the most prominent asynchronous component in the Pod lifecycle — it watches for unscheduled Pods and reconciles them independently of the original API request. Adopting KEP-5915's `ExtractContext` and Span Link pattern here serves as a concrete, high-value proof point for the async trace context propagation standard. Success in the Scheduler paves the way for adoption in other async consumers (Kubelet, kube-controller-manager, custom operators).

When the API Server handles a Pod creation request, it injects the current trace context (from the HTTP request's `traceparent` header) into the Pod's annotations (e.g., `tracing.k8s.io/traceparent`) using KEP-5915's `InjectContext`.

When the Scheduler picks up the Pod:

1. Extract the trace context from the Pod's annotations.
2. Start a new root span (`SchedulePod`) with a **Span Link** pointing to the extracted trace context.

Span Links (not Child Spans) are used because the scheduling cycle is asynchronous and decoupled from the API request. Using Child Spans would incorrectly imply that the API request is "blocked" until scheduling finishes, which breaks waterfall visualizations. Span Links preserve the causal connection ("This scheduling happened because of that Pod creation") without implying synchronous dependency.

```go
func (sched *Scheduler) schedulePod(ctx context.Context, pod *v1.Pod) {
    // Use KEP-5915's StartReconcileSpan to extract trace context from
    // Pod annotations and create a new root span with a Span Link to
    // the original creation trace.
    ctx, span := tracing.StartReconcileSpan(
        ctx, "SchedulePod", pod, sched.tracer, sched.propagator,
    )
    defer span.End()

    span.SetAttributes(
        attribute.String("k8s.pod.name", pod.Name),
        attribute.String("k8s.pod.namespace", pod.Namespace),
        attribute.String("k8s.pod.uid", string(pod.UID)),
    )

    // Run scheduling cycle with trace context propagated through ctx
    result := sched.schedulingCycle(ctx, pod)
    // ...
}
```

This directly consumes the `StartReconcileSpan` helper from KEP-5915, which handles the `ExtractContext` -> Span Link creation flow internally. If no trace context is present in the Pod's annotations (e.g., the API Server does not yet inject context), the scheduler still creates a useful root span — it just won't have a link to the creation trace.

### Implementation

The implementation hooks into the Scheduling Framework at two levels:

**Level 1: Phase-level spans** — Wrapping each phase method in `pkg/scheduler/framework/runtime` using `tracing.Start()` from `k8s.io/component-base/tracing`. This creates both an OpenTelemetry span and a `utiltrace` span simultaneously:

```go
func (f *frameworkImpl) RunFilterPlugins(ctx context.Context, ...) *fwk.Status {
    ctx, span := tracing.Start(ctx, "filter",
        attribute.String("scheduler.phase", "filter"))
    defer span.End(filterLogThreshold)

    // existing filter logic, with ctx propagated to plugins
    for _, pl := range f.filterPlugins {
        status := f.runFilterPlugin(ctx, pl, ...)
        // ...
    }
}
```

**Level 2: Per-plugin spans** — Wrapping individual plugin calls:

```go
func (f *frameworkImpl) runFilterPlugin(ctx context.Context, pl framework.FilterPlugin, ...) *framework.Status {
    ctx, span := tracing.Start(ctx, "plugin: "+pl.Name(),
        attribute.String("scheduler.plugin.name", pl.Name()),
        attribute.String("scheduler.plugin.extension_point", "filter"),
    )
    defer span.End(pluginLogThreshold)

    status := pl.Filter(ctx, state, pod, nodeInfo)
    if !status.IsSuccess() {
        span.AddEvent("plugin rejected", attribute.String("scheduler.result", status.Code().String()))
    }
    return status
}
```

Note: `span.End(logThreshold)` ends the OpenTelemetry span and also calls `utilSpan.LogIfLong(logThreshold)`, preserving existing log-based latency warnings. The `tracing.Span` type from `component-base/tracing` handles both under the hood.

**Custom Plugin Support:** Because the `context.Context` carrying the trace is passed through to plugin methods (e.g., `Filter(ctx, ...)`), out-of-tree plugins can start their own child spans using either the `component-base/tracing` API or the standard OpenTelemetry API without any changes to the framework.

### Test Plan

[X] I understand the owners of the involved components may require updates to existing tests to make this code solid enough prior to committing the changes necessary to implement this enhancement.

##### Prerequisite testing updates

None.

##### Unit tests

* `pkg/scheduler/framework/runtime`: Tests verifying that spans are created for each scheduling phase and per-plugin invocations.
* `pkg/scheduler/scheduler.go`: Tests verifying that the root `SchedulePod` span is created with correct attributes and Span Links when trace context is present in Pod annotations.

Target coverage for new tracing code: >80%

- `pkg/scheduler/framework/runtime`: `<date>` - `<test coverage>`
- `pkg/scheduler/scheduler.go`: `<date>` - `<test coverage>`

##### Integration tests

* Integration test verifying that when tracing is enabled via `TracingConfiguration`, scheduling a Pod produces trace spans with the expected hierarchy (root -> schedulingCycle -> phase -> plugin).
* Integration test verifying that trace context from Pod annotations results in a Span Link from the root `SchedulePod` span back to the API Server's trace.
* Integration test verifying that the Scheduler re-injects its own trace context into the Pod's annotations after binding, so downstream consumers (e.g., Kubelet) can link back to the Scheduler's trace.

##### e2e tests

* e2e test enabling tracing in the scheduler via component config, scheduling a Pod, and verifying that spans are exported to a test collector with the correct structure and attributes.

### Graduation Criteria

#### Alpha

- [ ] Implement `SchedulerTracing` feature gate.
- [ ] Add OpenTelemetry span instrumentation to `schedulingCycle` and `bindingCycle` in `pkg/scheduler`.
- [ ] Generate per-plugin child spans for Filter, Score, and other extension points.
- [ ] Reuse `TracingConfiguration` for scheduler component config.
- [ ] Extract W3C Trace Context from Pod annotations (using [KEP-5915](/keps/sig-instrumentation/5915-async-trace-context-propagation/README.md)) and create Span Links on the root `SchedulePod` span.
- [ ] Parity with existing `utiltrace` instrumentation in the scheduler (achieved automatically via `component-base/tracing`'s unified `tracing.Start()` API).
- [ ] Ensure consistent span naming and attributes with API Server and Kubelet tracing.
- [ ] Unit tests for all new tracing instrumentation.
- [ ] Integration tests demonstrating end-to-end tracing with a test OpenTelemetry collector.
- [ ] Documentation for enabling and configuring scheduler tracing.

#### Beta

- [ ] OpenTelemetry-Go SDK has reached GA (already satisfied).
- [ ] Gather feedback from early adopters on trace structure and usefulness.
- [ ] Address any performance concerns identified during alpha.
- [ ] Tracing 100% of scheduling attempts does not break scalability tests (this does not necessarily mean trace backends can handle all the data).
- [ ] All monitoring requirements completed.
- [ ] All testing requirements completed.
- [ ] All known pre-release issues and gaps resolved.

#### Stable

- [ ] At least two releases of beta usage and feedback.
- [ ] e2e tests demonstrating full lifecycle tracing across API Server, Scheduler, and Kubelet.
- [ ] Documentation published on kubernetes.io.
- [ ] No significant performance regressions reported during beta.

### Upgrade / Downgrade Strategy

**Upgrade:** When upgrading to a version with this feature:
- Tracing is disabled by default (behind the `SchedulerTracing` feature gate). No behavioral change until explicitly enabled.
- When enabled, the scheduler will begin producing trace spans for scheduling attempts, controlled by the sampling rate.
- No changes to existing scheduling behavior or APIs.

**Downgrade:** When downgrading to a version without this feature:
- Trace spans are no longer produced. No other behavioral change.
- Any tracing configuration in the scheduler component config is ignored by older versions.
- Existing Pods and scheduling behavior are completely unaffected.

### Version Skew Strategy

This feature is entirely internal to the kube-scheduler and does not involve coordination with other components for its core functionality.

* **Context Propagation (Span Links):** If the API Server injects trace context into Pod annotations but the Scheduler does not yet support extracting them (older version), the annotations are simply ignored. If the Scheduler supports extraction but the API Server does not inject annotations (older version), the Scheduler will create traces without Span Links, which is still useful for scheduler-internal observability.
* **No API changes:** This feature does not introduce new API types or modify existing ones. It only adds observability instrumentation.

## Production Readiness Review Questionnaire

### Feature Enablement and Rollback

###### How can this feature be enabled / disabled in a live cluster?

- [X] Feature gate (also fill in values in `kep.yaml`)
  - Feature gate name: `SchedulerTracing`
  - Components depending on the feature gate: `kube-scheduler`
- [X] Other
  - Describe the mechanism: **KubeSchedulerConfiguration TracingConfiguration** (from `k8s.io/component-base`). The behavior depends on the combination of feature gate and configuration:
    - When the `SchedulerTracing` feature gate is **disabled**, the scheduler will:
      - Not generate spans
      - Not initiate an OTLP connection
      - Not propagate context
    - When the feature gate is **enabled**, but no `TracingConfiguration` is provided, the scheduler will:
      - Not generate spans
      - Not initiate an OTLP connection
      - Propagate context in [passthrough](https://github.com/open-telemetry/opentelemetry-go-contrib/tree/main/examples/passthrough) mode
    - When the feature gate is **enabled**, and a `TracingConfiguration` with sampling rate 0 (the default) is provided, the scheduler will:
      - Initiate an OTLP connection
      - Not record or export spans for its own root spans (e.g., `SchedulePod`). Note: because the scheduler creates new root spans via Span Links (not child spans of the API Server trace), the `ParentBasedSampler` treats them as roots and applies the configured rate (0 = no sampling). The sampled flag from the linked API Server trace does not influence this decision, since Span Links do not establish a parent-child relationship.
      - Propagate context normally
    - When the feature gate is **enabled**, and a `TracingConfiguration` with sampling rate > 0 is provided, the scheduler will:
      - Initiate an OTLP connection
      - Record and export spans at the specified sampling rate for root spans (e.g., `SchedulePod`). As above, the sampling decision is independent of the linked API Server trace's sampled flag.
      - Propagate context normally

    The following table summarizes the behavior:

    | Configuration | Span object created? | Data recorded? | Exported to backend? |
    |---|---|---|---|
    | Feature gate disabled | No | No | No |
    | Feature gate enabled, no `TracingConfiguration` | No | No | No |
    | Feature gate enabled, sampling rate 0 | Yes (non-recording) | No | No |
    | Feature gate enabled, sampling rate > 0 (unsampled span) | Yes (non-recording) | No | No |
    | Feature gate enabled, sampling rate > 0 (sampled span) | Yes | Yes | Yes |

    **Note on non-recording spans:** When the sampling decision is "don't sample," the OpenTelemetry SDK still creates a lightweight span object in memory. This object carries trace context for propagation but does not record attributes, events, or timing data. Calls to `SetAttributes`, `AddEvent`, etc. on a non-recording span are no-ops. These spans are never exported to the tracing backend.
  - Will enabling / disabling the feature require downtime of the control plane? Yes, the scheduler must be restarted to change the feature gate or tracing configuration.
  - Will enabling / disabling the feature require downtime or reprovisioning of a node? No, this feature is scheduler-only.

###### Does enabling the feature change any default behavior?

No. The feature is disabled unless both the feature gate is enabled and the `TracingConfiguration` is populated in the `KubeSchedulerConfiguration`. When the feature is enabled, it does not change behavior from the users' perspective; it only adds tracing telemetry based on scheduling operations. Scheduling decisions, latency, and correctness are unaffected.

###### Can the feature be disabled once it has been enabled (i.e. can we roll back the enablement)?

Yes. Disabling the `SchedulerTracing` feature gate and restarting the scheduler will stop all trace production. There are no persistent side effects — traces are exported to an external collector and no state is stored in etcd or the scheduler.

###### What happens if we reenable the feature if it was previously rolled back?

Traces resume being produced for new scheduling attempts. There is no accumulated state, so re-enablement is clean.

###### Are there any tests for feature enablement/disablement?

Unit tests will verify that:
- When the feature gate is disabled, no spans are created.
- When the feature gate is enabled but no tracing config is provided, no spans are created and context is propagated in passthrough mode.
- When the feature gate is enabled and sampling rate is 0, non-recording span objects are created but no spans are recorded or exported.
- When the feature gate is enabled and sampling rate is > 0, sampled spans are recorded and exported with the expected hierarchy; unsampled spans are non-recording.
- When trace context is present in Pod annotations, the root `SchedulePod` span has a Span Link to the API Server's trace.
- After binding, the scheduler re-injects its own trace context into the Pod's annotations.

### Rollout, Upgrade and Rollback Planning

###### How can a rollout or rollback fail? Can it impact already running workloads?

A rollout cannot impact running workloads because this feature only adds observability instrumentation to the scheduler. It does not change scheduling decisions, API behavior, or Pod placement logic.

While a misconfigured OTLP endpoint is the most common failure mode, other possibilities include: exporter resource exhaustion (the SDK's internal buffer, defaulting to 2048 spans, fills up if the backend is down, causing new spans to be silently dropped), invalid configuration causing the OTel SDK to fail fast during initialization (e.g., a malformed endpoint URL), transient network issues causing export timeouts, and the backend rejecting data (e.g., invalid API key, rate limit exceeded).

During normal runtime, the SDK is non-blocking — export happens asynchronously in a background goroutine and does not block the scheduling path. However, two caveats apply: (1) while export is async, span *collection* (creating span objects, allocating memory) is synchronous and adds a small per-span overhead on the hot path; (2) during scheduler shutdown, `Shutdown()` is blocking by design to flush pending spans, and can hang for the duration of the export timeout if the OTLP endpoint is unreachable. Neither of these affects scheduling decisions or correctness.

###### What specific metrics should inform a rollback?

* `scheduler_scheduling_attempt_duration_seconds`: If this metric shows a measurable increase after enabling tracing, it would indicate a performance regression.
* `scheduler_pending_pods`: If Pods start accumulating unexpectedly.
* Process CPU and memory usage of the kube-scheduler process.

###### Were upgrade and rollback tested? Was the upgrade->downgrade->upgrade path tested?

Will be tested during alpha. The feature is stateless, so upgrade/downgrade/upgrade cycles should be straightforward.

###### Is the rollout accompanied by any deprecations and/or removals of features, APIs, fields of API types, flags, etc.?

No.

### Monitoring Requirements

###### How can an operator determine if the feature is in use by workloads?

- [X] Metrics
  - The scheduler already exposes `scheduler_scheduling_attempt_duration_seconds`. When tracing is enabled, operators can verify trace production by checking their tracing backend for `SchedulePod` spans.
  - A new metric `scheduler_tracing_spans_exported_total` will track the number of trace spans successfully exported, with labels for `result` (success/error).

###### How can someone using this feature know that it is working for their instance?

- [ ] Events
  - Event Reason:
- [ ] API .status
  - Condition name:
  - Other field:
- [X] Other
  - Details: Verify that `SchedulePod` spans appear in the configured tracing backend (e.g., Jaeger, Tempo). Spans should include the expected hierarchy (schedulingCycle, bindingCycle, per-plugin spans) and correct attributes (pod name, namespace, UID, plugin names, scheduling result).

###### What are the reasonable SLOs (Service Level Objectives) for the enhancement?

This feature is purely for observability and does not have its own SLOs. It should not impact existing scheduling SLOs:
- Scheduling latency overhead scales with the configured sampling rate (unsampled spans are non-recording and have negligible cost; sampled spans incur synchronous collection overhead on the scheduling path). Concrete latency impact numbers will be established through benchmarking during alpha.
- Zero impact on scheduling correctness.

###### What are the SLIs (Service Level Indicators) an operator can use to determine the health of the service?

- [X] Metrics
  - Metric name: `scheduler_tracing_spans_exported_total`
  - Aggregation method: Rate
  - Components exposing the metric: kube-scheduler
- [X] Metrics
  - Metric name: `scheduler_scheduling_attempt_duration_seconds` (existing — monitor for regression)
  - Components exposing the metric: kube-scheduler

###### Are there any missing metrics that would be useful to have to improve observability of this feature?

A metric tracking silently dropped spans (due to buffer saturation) could be useful to alert operators that trace data is being lost. This will be evaluated during alpha.

### Dependencies

###### Does this feature depend on any specific services running in the cluster?

The feature itself (tracing in the scheduler) does not depend on services running in the cluster. However, collecting traces requires an [OTLP-compatible](https://opentelemetry.io/docs/specs/otlp/) trace collection pipeline. The OpenTelemetry Collector below is one common option, but any OTLP-compatible receiver may be used. The impact of outages is the same regardless of collection pipeline.

* **[OpenTelemetry Collector](https://github.com/open-telemetry/opentelemetry-collector) (optional):**
  - Usage description: Deploy the collector as a sidecar or DaemonSet, and route traces to the backend of choice.
  - Impact of its outage on the feature: Spans will continue to be created and buffered by the kube-scheduler, but may be lost before they reach the trace backend. Scheduling is completely unaffected.
  - Impact of its degraded performance or high-error rates on the feature: Spans may be lost before they reach the trace backend. Scheduling is completely unaffected.

### Scalability

###### Will enabling / using this feature result in any new API calls?

No. The instrumentation adds spans to existing scheduling operations. No new API calls are made.

###### Will enabling / using this feature result in introducing new API types?

No.

###### Will enabling / using this feature result in any new calls to the cloud provider?

No.

###### Will enabling / using this feature result in increasing size or count of the existing API objects?

The re-injection of trace context into Pod annotations (per KEP-5915) adds approximately 55 bytes per Pod for the `tracing.k8s.io/traceparent` annotation. Trace span data itself is exported to an external OTLP endpoint, not stored in Kubernetes objects.

###### Will enabling / using this feature result in increasing time taken by any operations covered by existing SLIs/SLOs?

The overhead of creating and closing OpenTelemetry spans during scheduling phases, and extracting trace context from Pod annotations, depends on whether the span is sampled (attributes are recorded) or unsampled (non-recording, effectively a no-op). Concrete per-span overhead numbers will be established through benchmarking during alpha.

###### Will enabling / using this feature result in non-negligible increase of resource usage (CPU, RAM, disk, IO, ...) in any components?

At reasonable sampling rates and attribute counts, the increase is expected to be minimal. Concrete numbers will be established through alpha benchmarking. Expected impact in the kube-scheduler:
- **CPU:** While export is asynchronous, span collection (creating span objects, recording attributes) is synchronous on the scheduling path. For sampled spans, this adds a small per-span CPU cost. For unsampled spans (non-recording), the cost is negligible. At high throughput with high sampling rates, the cumulative collection overhead may become measurable.
- **RAM:** Increase proportional to the number of buffered spans awaiting export. The SDK's internal buffer defaults to 2048 spans — the actual memory footprint depends on the number of attributes per span. If the OTLP endpoint is unreachable, the buffer fills and new spans are silently dropped (no unbounded memory growth). A full buffer retrying failed exports also contributes to memory pressure.
- **Disk:** None (traces are exported over the network, not written to disk).
- **Network IO:** Proportional to the sampling rate. At 1% sampling, the additional network traffic is negligible.

###### Can enabling / using this feature result in resource exhaustion of some node resources (PIDs, sockets, inodes, etc.)?

No. The scheduler maintains a single gRPC connection to the OTLP endpoint. No per-Pod resources are consumed. If the endpoint is unavailable, the SDK drops spans after the buffer fills, preventing unbounded memory growth.

### Troubleshooting

###### How does this feature react if the API server and/or etcd is unavailable?

If the API Server is unavailable, the scheduler cannot receive Pod events and scheduling stops entirely — this is independent of tracing. The tracing feature itself does not depend on etcd.

If the OTLP endpoint is unavailable, trace exports fail silently and spans are dropped. Scheduling continues normally.

###### What are other known failure modes?

- **Traces not appearing in the backend**
  - Detection: No `SchedulePod` spans in the tracing backend despite Pods being scheduled.
  - Mitigations: Verify the `SchedulerTracing` feature gate is enabled; verify `TracingConfiguration` has a valid OTLP endpoint; verify the collector is running and reachable.
  - Diagnostics: Check scheduler logs for tracing-related errors (e.g., OTLP export failures). Verify network connectivity between the scheduler and the collector.
  - Testing: Integration tests verify span export to a test collector.

- **Incomplete trace hierarchy (e.g., missing plugin spans)**
  - Detection: Root `SchedulePod` span exists but child spans are missing.
  - Mitigations: Verify tracing is configured with a sufficient sampling rate; verify the plugin is passing context correctly.
  - Diagnostics: Increase scheduler log verbosity; check for errors in plugin initialization.
  - Testing: Unit tests verify the complete span hierarchy.

###### What steps should be taken if SLOs are not being met to determine the problem?

1. Check if the `SchedulerTracing` feature gate is enabled.
2. Compare `scheduler_scheduling_attempt_duration_seconds` with and without tracing enabled — if there is a measurable difference, reduce the sampling rate or disable the feature gate.
3. Check scheduler CPU and memory usage for unexpected increases.
4. Verify the OTLP endpoint is responsive and not causing backpressure (check for export errors in scheduler logs).
5. If issues persist, disable the feature gate and restart the scheduler.

## Implementation History

- 2025-12-11: [Proposal document](https://docs.google.com/document/d/1orm174rZRt43TeMCee3cB_RuISyKVGypieKdnDoE_ks) shared for initial feedback.
- 2026-02-19: Initial KEP draft created by @artem-tkachuk and enhancement issue filed (#5927).

## Drawbacks

* **Maintenance Burden:** Adding tracing instrumentation increases code complexity in the scheduling framework. Each new extension point or framework change needs to consider tracing span management.

* **Partial Observability Until Full Adoption:** Until the API Server injects trace context into Pod annotations via [KEP-5915](/keps/sig-instrumentation/5915-async-trace-context-propagation/README.md), Scheduler traces will not be linked to the original creation request. The Scheduler traces are still independently valuable for per-phase and per-plugin observability, but full end-to-end lifecycle tracing (API Server -> Scheduler -> Kubelet) requires KEP-5915 adoption across components. As the first adopter of KEP-5915, the Scheduler validates the pattern and accelerates adoption in other components.

* **Third-Party Dependency:** While the OpenTelemetry SDK is already a dependency via `component-base`, increased usage increases the surface area exposed to SDK bugs or breaking changes. This risk is shared with the API Server and Kubelet tracing implementations.

## Alternatives

### Extending utiltrace Instead of OpenTelemetry

Instead of OpenTelemetry, one could extend the existing `utiltrace` package to provide more detailed timing breakdowns.

**Why rejected:**
- `utiltrace` is log-based and cannot be correlated with traces from other components (API Server, Kubelet).
- It does not support distributed context propagation.
- It cannot export to tracing backends (Jaeger, Tempo, etc.).
- Even within the scheduler itself, `utiltrace` only produces flat log lines. OpenTelemetry traces provide a structured parent-child hierarchy that tracing backends can render as waterfall visualizations, making it easy to see exactly which phase or plugin contributed to latency at a glance.
- OpenTelemetry is the CNCF standard for observability and is already adopted by the API Server and Kubelet.
- The `component-base/tracing` package provides a unified [`tracing.Start()`](https://github.com/kubernetes/kubernetes/blob/04d87a4b6e72336ee9afb1e5b477223c96a8fcbb/staging/src/k8s.io/component-base/tracing/tracing.go#L33) that creates both OTel and `utiltrace` spans simultaneously, so existing `utiltrace` instrumentation is already preserved.

### Generic Automatic Instrumentation

Instead of explicitly instrumenting each phase and plugin in the Scheduling Framework, one could attempt to automatically instrument at a higher level (e.g., wrapping all plugin calls generically).

**Why not now:**
- The Scheduling Framework already has well-defined extension points, making explicit instrumentation straightforward and precise.
- Automatic instrumentation risks producing noisy, less meaningful spans.
- Explicit instrumentation allows adding domain-specific attributes (plugin names, scheduling results, queue wait times) that automatic approaches cannot infer, since they lack semantic understanding of the code being wrapped.
- This approach may be revisited in the future for out-of-tree plugins.

### External Profiling Instead of Tracing

Tools like `pprof` or continuous profiling could be used instead of distributed tracing to understand scheduler performance.

**Why rejected:**
- Profiling shows aggregate CPU/memory usage but not per-Pod or per-scheduling-attempt breakdowns.
- Profiling cannot correlate scheduler activity with the wider request lifecycle (API Server -> Scheduler -> Kubelet).
- Profiling and tracing serve complementary purposes. Tracing provides request-scoped latency breakdowns; profiling provides aggregate resource usage. Both are valuable.

## Infrastructure Needed (Optional)

None. This feature reuses existing Kubernetes infrastructure (`component-base/tracing`, `TracingConfiguration`) and does not require new repositories, subprojects, or CI infrastructure.
