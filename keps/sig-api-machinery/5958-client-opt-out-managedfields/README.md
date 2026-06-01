# KEP-5958: Client Opt-out for managedFields in API Response

<!-- toc -->
- [Release Signoff Checklist](#release-signoff-checklist)
- [Summary](#summary)
- [Motivation](#motivation)
  - [Goals](#goals)
  - [Non-Goals](#non-goals)
- [Proposal](#proposal)
  - [User Stories](#user-stories)
    - [Story 1](#story-1)
    - [Story 2](#story-2)
  - [Risks and Mitigations](#risks-and-mitigations)
- [Design Details](#design-details)
  - [Accept Parameter](#accept-parameter)
  - [Implementation Details](#implementation-details)
    - [In-tree Controller Migration](#in-tree-controller-migration)
    - [Client Configuration](#client-configuration)
  - [Test Plan](#test-plan)
    - [Prerequisite testing updates](#prerequisite-testing-updates)
    - [Unit Tests](#unit-tests)
    - [Integration Tests](#integration-tests)
    - [e2e Tests](#e2e-tests)
  - [Graduation Criteria](#graduation-criteria)
    - [Alpha](#alpha)
    - [Beta](#beta)
    - [GA](#ga)
  - [Upgrade / Downgrade Strategy](#upgrade--downgrade-strategy)
  - [Version Skew Strategy](#version-skew-strategy)
    - [Client Support and Version Skew](#client-support-and-version-skew)
- [Production Readiness Questionnaire](#production-readiness-questionnaire)
  - [Feature Enablement and Rollback](#feature-enablement-and-rollback)
  - [Rollout, Upgrade and Rollback Planning](#rollout-upgrade-and-rollback-planning)
  - [Monitoring Requirements](#monitoring-requirements)
  - [Dependencies](#dependencies)
  - [Scalability](#scalability)
  - [Troubleshooting](#troubleshooting)
- [Implementation History](#implementation-history)
- [Drawbacks](#drawbacks)
- [Alternatives](#alternatives)
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

- [ ] (R) Enhancement issue in release milestone, which links to KEP dir in [kubernetes/enhancements] (not the initial KEP PR)
- [ ] (R) KEP approvers have approved the KEP status as `implementable`
- [ ] (R) Design details are appropriately documented
- [ ] (R) Test plan is in place, giving consideration to SIG Architecture and SIG Testing input (including test refactors)
  - [ ] e2e Tests for all Beta API Operations (endpoints)
  - [ ] (R) Ensure GA e2e tests meet requirements for [Conformance Tests](https://github.com/kubernetes/community/blob/master/contributors/devel/sig-architecture/conformance-tests.md)
  - [ ] (R) Minimum Two Week Window for GA e2e tests to prove flake free
- [ ] (R) Graduation criteria is in place
  - [ ] (R) [all GA Endpoints](https://github.com/kubernetes/community/pull/1806) must be hit by [Conformance Tests](https://github.com/kubernetes/community/blob/master/contributors/devel/sig-architecture/conformance-tests.md) within one minor version of promotion to GA
- [ ] (R) Production readiness review completed
- [ ] (R) Production readiness review approved
- [ ] "Implementation History" section is up-to-date for milestone
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

This enhancement allows Kubernetes clients to opt-out of receiving `metadata.managedFields` in API responses (GET, LIST, WATCH, PUT, POST, and PATCH) via an HTTP `Accept` parameter. This reduces network bandwidth, API Server CPU serialization costs, and client-side memory allocations and Garbage Collection overhead.

## Motivation

`metadata.managedFields` is used by the API server for Server-Side Apply (SSA) conflict resolution. However, the vast majority of Kubernetes clients do not actively process this data. Many core components, such as `kube-controller-manager` and `kube-scheduler`, currently use client-side transforms to drop managed fields to save memory.

As documented in the [Server-Side Apply KEP](https://github.com/kubernetes/enhancements/tree/master/keps/sig-api-machinery/555-server-side-apply#scalability), `managedFields` can represent up to **60% of the total size** of an object. The actual overhead varies with the number of field managers and the fields they touch — objects managed by multiple controllers, webhooks, and users in production clusters carry proportionally larger `managedFields`. Reducing this overhead is important for [supporting larger resource sizes as a scalability dimension](https://github.com/kubernetes/kubernetes/issues/134375).

Relying on client-side transforms still incurs significant system-wide costs:
- **API Server CPU:** The API server still performs expensive serialization of `managedFields`, even when the client will immediately discard it.
- **Network Overhead:** `managedFields` payloads significantly increase transfer time during LIST and WATCH operations. This contributes to request timeouts and limits the API server's ability to handle large resources (see [#134375](https://github.com/kubernetes/kubernetes/issues/134375)).
- **Client-side GC:** Clients must allocate structural objects (string headers, maps, and slice backing arrays) for `managedFields` before discarding them, adding unnecessary memory pressure and GC overhead.

### Goals

- Reduce API server serialization CPU by eliminating the work of encoding `managedFields` for clients that do not need it. The savings are proportional to the `managedFields` share of each object, which varies by the number of field managers.
- Reduce network transfer sizes between the API server and clients proportionally to the `managedFields` overhead (up to 60% per the [SSA KEP](https://github.com/kubernetes/enhancements/tree/master/keps/sig-api-machinery/555-server-side-apply#scalability)), unblocking support for larger resource sizes and reducing LIST/WATCH timeout risk.
- Reduce client-side memory allocations and GC overhead by eliminating the need to allocate and immediately discard `managedFields` data structures.
- Migrate in-tree clients (`kube-controller-manager`, `kube-scheduler`) from client-side informer transforms to server-side opt-out, moving the cost savings from the client to the entire request path.

### Non-Goals

- **General-purpose field selection or opting out of other fields** (though the API design is intended to be extensible to support this in the future without a redesign).
- **Opting out of fields in the request body of write operations.** This KEP drops fields from API *responses* (including responses to write operations like PUT/POST/PATCH), not from request bodies. `kube-controller-manager` and `kube-scheduler` already send objects without `managedFields` on PUT (due to informer transforms). This is safe because the field manager falls back to the stored object's `managedFields` when the request has none.

## Proposal

This KEP proposes using an `Accept` header parameter (`drop=metadata.managedFields`) to allow clients to opt-out of receiving `metadata.managedFields` in API responses. When the API server receives a request with this parameter, it uses an alternate serializer that skips `managedFields` during encoding. This is implemented as a new serializer mode with a distinct `runtime.Identifier`, which allows the watch cache's `cachingObject` to naturally cache both the full and stripped serialized forms as separate entries without any changes to the watch cache itself.

### User Stories

#### Story 1
As a cluster operator or developer of a high-traffic controller (like the scheduler), I want to avoid the overhead of receiving `managedFields` because I don't use them for my reconciliation logic. This will save CPU on both the API server and my controller, and reduce network bandwidth.

#### Story 2
As a developer using `client-go`, I want to be able to easily opt-out of `managedFields` for certain informers where the data is redundant.

### Risks and Mitigations

- **Silent Data Loss in Clients:** If a client opts out but later attempts an operation that requires `managedFields` (e.g., using `managedfields.ExtractInto`), `ExtractInto` will silently return empty results (it already returns `nil` when no matching entry is found).
  - **Mitigation:** Document clearly that clients using the extract-modify-apply workflow must not opt out of `managedFields`. This is an explicit opt-in by the client, so the responsibility lies with the client author.
- **Watch Cache Memory Overhead:** Mixed opt-in and opt-out watchers will cause the API server to maintain two serialized versions of each object in the `cachingObject` transient cache.
  - **Mitigation:** The `cachingObject` is transient and created per dispatch event. The cost is limited to one additional serialization per unique format. If all watchers opt out, the single cached version is smaller and cheaper to produce.

## Design Details

### Accept Parameter

We propose using an `Accept` header parameter to signal the request to exclude `managedFields`. 

Example:
`Accept: application/json; drop=metadata.managedFields`

This follows Kubernetes API conventions where `Accept` parameters are used for structural transformations (e.g., `as=PartialObjectMetadata`, `as=Table`).

While this KEP is strictly scoped to `metadata.managedFields`, the `drop` parameter is designed to be extendable to other fields in the future using `+` as a separator (e.g., `drop=metadata.managedFields+metadata.annotations`). Unknown drop targets are silently ignored for forward compatibility.

### Implementation Details

1.  **API Server Serializer:** Add an `ExcludeManagedFields` option to the JSON, Protobuf, and CBOR serializers. When this option is set, the serializer strips `metadata.managedFields` from the Go object before encoding and exposes this variant as a distinct codec on `runtime.SerializerInfo` with its own `Identifier()`. The content type negotiation layer selects the appropriate serializer based on the `drop` parameter in the `Accept` header. Stripping happens at the Go-object level before encoding, so it is serializer-agnostic and applies uniformly across all formats. Protobuf support is critical since `kube-controller-manager` and `kube-scheduler` use Protobuf by default. CRDs are also in scope — the `apiextensions-apiserver` constructs its own `SerializerInfo` for custom resources and will need to wire in the `ExcludeManagedFields` variant alongside its existing serializers.
2.  **Watch Cache:** No changes are needed to the watch cache or `cachingObject`. The `cachingObject`'s `serializationsCache` is keyed by `runtime.Identifier`. Since the stripped serializer has a different `Identifier` than the full serializer, the cache naturally maintains both forms as separate entries. This means that until all watchers migrate to dropping `managedFields`, each watch event will be serialized twice (once with and once without `managedFields`). Benchmarking shows this dual-serialization adds roughly 62% more time and 83% more memory per event, but this overhead is constant regardless of watcher count and is offset by the smaller payload sizes.
3.  **Discoverability:** The capability should be discoverable via the supported media types in the OpenAPI schema.
4.  **Client-side Mitigation:** `managedfields.ExtractInto` already returns `nil` (no error) when no matching managed fields entry is found, which is a safe no-op. No changes to `ExtractInto` error handling are needed. Clients that opt out should be aware that `ExtractInto` will silently return empty results.

#### In-tree Controller Migration

As part of this KEP, we will migrate `kube-controller-manager` and `kube-scheduler` to use server-side opt-out instead of their current client-side informer transforms:

- Both components currently use `TransformFunc` on their informers to nil out `managedFields` after deserialization. This saves client memory but does not reduce serialization CPU or network transfer costs.
- With this KEP, both components will be updated to send the `drop=metadata.managedFields` parameter in the `Accept` header, moving the field stripping to the API server's serializer. This eliminates the serialization and network costs in addition to the existing client memory savings.
- The migration will be gated behind the `ManagedFieldsOptOutClient` feature gate. When enabled, the informer transforms for `managedFields` stripping become redundant and will be removed.

#### Client Configuration

`drop=metadata.managedFields` is an extension of the existing `Accept`-header content negotiation mechanism (the same family as `as=PartialObjectMetadata` and `as=Table`). The exact client-go surface for opting in (e.g., a client-level default vs. a per-informer option) will be decided during implementation.

The broader question of per-client vs. per-request configuration is a general property of the `Accept` header in client-go (a single client uses one `Accept` header for all requests) and is **out of scope** for this KEP. Because the opt-out is carried in the `Accept` header, the design is forward-compatible with finer-grained configuration if it is added later.

### Test Plan

[x] I/we understand the owners of the involved components may require updates to
existing tests to make this code reproducible.

#### Prerequisite testing updates

None.

#### Unit Tests

- Test API server encoders (JSON, Protobuf, and CBOR) with and without the `managedFields` exclusion flag.
- Test `cachingObject` serialization cache hits and misses with the exclusion flag.

#### Integration Tests

- Verify that GET, LIST, and WATCH requests with the `Accept` parameter correctly return objects without `managedFields`.
- Verify that standard requests (without the parameter) still return `managedFields`.
- Verify mixed watch scenarios with both opt-in and opt-out clients.

#### e2e Tests

- Ensure core components (e.g., scheduler) can successfully opt-out and function correctly.

### Graduation Criteria

The feature uses two independent gates with their own maturities. The server-side serializer is inert unless a client opts in via the `Accept` header, so it carries little risk and is introduced directly at Beta maturity. The client-side gate controls whether in-tree controllers actually drop `managedFields`, which is the part that carries rollout risk, so it starts at Alpha maturity. The milestones below describe the KEP's overall progression.

#### Alpha

- `ManagedFieldsOptOut` (server-side, in `kube-apiserver`): introduced at **Beta** feature-gate maturity, enabled by default. Recognizes the `drop=metadata.managedFields` `Accept` parameter for JSON, Protobuf, and CBOR across GET, LIST, WATCH, PUT, POST, and PATCH.
- `ManagedFieldsOptOutClient` (client-side, in `client-go`): introduced at **Alpha** feature-gate maturity, disabled by default. Controls whether in-tree clients (`kube-scheduler`, `kube-controller-manager`) send the `Accept` header to drop `managedFields`.

#### Beta

- `ManagedFieldsOptOutClient` (client-side) promoted to Beta and enabled by default; in-tree controllers use the opt-out by default.
- Performance benchmarks confirming savings in API server and clients.
- User-facing documentation published in [kubernetes/website].

#### GA

- In-tree controllers have run with the opt-out enabled by default for at least 2 releases with no related regressions.
- Both feature gates locked to their default values (removed 3 releases later).

### Upgrade / Downgrade Strategy

- **Upgrade:** New API servers will recognize the `Accept` parameter. Old clients will continue to work as before (not sending the parameter, thus receiving `managedFields`).
- **Downgrade:** If a client starts using the parameter and the API server is downgraded to a version that doesn't recognize it, the API server will ignore the unknown parameter and return the full object with `managedFields`. The client will receive more data than requested but should be able to handle it (ignoring the field if it doesn't need it).

### Version Skew Strategy

Supported. API servers that do not recognize the `drop` parameter will simply ignore it and return the full object, which is a safe default.

#### Client Support and Version Skew

To ensure consistent behavior and support clients running against older API servers:

- **client-go Modification:** `client-go` will be modified to support field dropping by adding a configuration option to request dropping specific fields. When enabled, it will automatically add the `drop=metadata.managedFields` parameter to the `Accept` header.
- **Informer Configuration:** Informers will inherit this configuration from the client they use. If a client is configured to drop `managedFields`, the informer will receive objects without `managedFields`.
- **Client-side Stripping (Defensive):** If a client requests `drop=metadata.managedFields` but talks to an older API server that does not support the feature, the server will return the full object with `managedFields`. To provide a consistent experience and avoid memory overhead in the client, `client-go` will be updated to strip `managedFields` from the response client-side if the client requested it but the server failed to drop it. This ensures that clients requesting the drop never see `managedFields` in the returned objects, regardless of server version.

## Production Readiness Questionnaire

### Feature Enablement and Rollback

###### How can this feature be enabled / disabled in a live cluster?

- [x] **Feature gate (also fill in values in `kep.yaml`)**
  - Feature gate name: `ManagedFieldsOptOut` and `ManagedFieldsOptOutClient`
  - Components depending on the feature gate: `kube-apiserver` (for `ManagedFieldsOptOut`), `kube-scheduler`, `kube-controller-manager` (for `ManagedFieldsOptOutClient`)

###### Does enabling the feature change any default behavior?

No. The API server behavior only changes when clients explicitly opt-out using the `Accept` header parameter. Enabling the client-side feature gate will cause those specific clients to start omitting the data from their informers and responses.

###### Can the feature be disabled once it has been enabled (i.e. can we roll back the enablement)?

Yes. Disabling `ManagedFieldsOptOut` will cause the API server to ignore the `drop` parameter and always return `managedFields`. Disabling `ManagedFieldsOptOutClient` will cause clients to stop sending the `Accept` header. Feature gates are typically disabled by setting the flag to `false` and restarting the component.

###### What happens if we reenable the feature if it was previously rolled back?

Clients requesting the opt-out will start receiving objects without `managedFields` again. There are no known side effects of reenabling the feature.

###### What happens if the feature is enabled or disabled in the middle of a rollout?

Clients requesting the opt-out might inconsistently receive `managedFields` depending on which API server instance they hit. This is safe as clients are expected to handle the presence of `managedFields`.

###### Are there any tests for feature enablement/disablement?

Yes, integration tests will cover behavior with the feature gate on and off.

### Rollout, Upgrade and Rollback Planning

###### How can a rollout or rollback fail? Can it impact already running workloads?

A rollout failure might lead to API server crashes if there's a bug in the encoder. However, this is unlikely given the targeted nature of the change. It shouldn't impact already running workloads that don't use the new feature.

###### What specific metrics should inform a rollback?

Monitor `kube_apiserver_request_duration_seconds` and network egress. A sudden increase in 5xx errors or unexpected latency spikes should inform a rollback.

###### Were upgrade and rollback tested? Was the upgrade->downgrade->upgrade path tested?

To be tested during implementation.

###### Is the rollout accompanied by any deprecations and/or removals of features, APIs, fields of API types, flags, etc.?

No.

### Monitoring Requirements

###### How can an operator determine if the feature is in use by workloads?

Per-request metrics for `Accept` parameters are not currently available. We may consider adding a metric or logging to track the usage of this specific parameter if required.

###### How can someone using this feature know that it is working for their instance?

- [x] Other (treat as last resort)
  - Details: Clients can verify that `metadata.managedFields` is absent in responses when the `Accept` parameter is used.

###### What are the reasonable SLOs (Service Level Objectives) for the enhancement?

The feature should not introduce any measurable latency increase for API requests. Serialization of objects without `managedFields` should be faster than serialization with them.

###### What are the SLIs (Service Level Indicators) an operator can use to determine the health of the service?

- [x] Metrics
  - Metric name: `kube_apiserver_request_duration_seconds`
  - Components exposing the metric: `kube-apiserver`

###### Are there any missing metrics that would be useful to have to improve observability of this feature?

A metric tracking the number of requests using the `drop=metadata.managedFields` parameter would be useful.

### Dependencies

###### Does this feature depend on any specific services running in the cluster?

No.

### Scalability

###### Will enabling / disabling this feature involve terminating any processes?

No.

###### Will enabling / disabling this feature take a long time?

No.

###### Will the feature increase the size of any objects?

No, it decreases the size of returned objects.

###### Will the feature increase the memory or CPU usage of any component?

It may slightly increase transient memory in `kube-apiserver` if there is a highly mixed population of opt-in and opt-out watchers (due to the extra entries in the `cachingObject` cache). However, the reduction in serialization work and smaller payload sizes are expected to yield an overall decrease in CPU and memory usage.

**Note on Benchmarking:** Standard scalability tests often use minimal pods that do not reflect realistic `managedFields` usage. To properly measure the impact, we plan to create micro-benchmarks in the API server (similar to the prototype explored in the PoC) using mock pods with realistic `managedFields` (simulating multiple managers). For full cluster scalability tests, we will work on defining 'exemplary pods' that reflect realistic production usage.

### Troubleshooting

###### How can an operator determine if the feature is broken?

If clients that have opted out suddenly start receiving `managedFields` despite the header, or if the API server returns errors for requests containing the `Accept` parameter.

## Implementation History

<!--
Major milestones in the lifecycle of a KEP should be tracked in this section.
Major milestones might include:
- the `Summary` and `Motivation` sections being merged, signaling SIG acceptance
- the `Proposal` section being merged, signaling agreement on a proposed design
- the date implementation started
- the first Kubernetes release where an initial version of the KEP was available
- the version of Kubernetes where the KEP graduated to general availability
- when the KEP was retired or superseded
-->

## Drawbacks

- Slight complexity increase in API server encoding logic.
- Potential for two cached serialized versions of the same object in the watch cache.
- Adds to the technical performance debt by introducing another permutation of object representation (e.g., Protobuf without `managedFields`). This increases the matrix of combinations (format + options) that must be tracked, tested, and optimized in the API server over time.

## Alternatives

- **ListOptions/GetOptions flag:** e.g., `excludeManagedFields=true`. Rejected because `Accept` parameters are the standard way in Kubernetes to control object representation.
- **General-purpose field selector:** A much larger undertaking that has been discussed for years. This KEP provides a targeted solution for a high-impact field while leaving room for future generalization. Note that a general solution would likely require a different implementation approach—such as walking the serialized form (Protobuf or CBOR) to dynamically emit the filtered response—rather than modifying Go structs before serialization as done in this targeted KEP.

## Infrastructure Needed (Optional)

None.
