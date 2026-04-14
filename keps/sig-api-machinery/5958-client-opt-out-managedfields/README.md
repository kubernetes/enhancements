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

This enhancement allows Kubernetes clients to opt-out of receiving `metadata.managedFields` in API responses (GET, LIST, and WATCH) via an HTTP `Accept` parameter. This reduces network bandwidth, API Server CPU serialization costs, and client-side memory allocations and Garbage Collection overhead.

## Motivation

`metadata.managedFields` is used by the API server for Server-Side Apply (SSA) conflict resolution. However, the vast majority of Kubernetes clients do not actively process this data. Many core components, such as `kube-controller-manager` and `kube-scheduler`, currently use client-side transforms to drop managed fields to save memory. 

Relying on client-side transforms still incurs significant system-wide costs:
- **API Server CPU:** The API server still performs expensive serialization of `managedFields`.
- **API Server Memory Savings:** Faster LIST responses and reduced payload sizes over the network allow clients to catch up faster. Since the cost of a full LIST operation is significantly reduced without `managedFields`, the penalty for a watch cache miss (forcing a relist) is lower. This allows the API Server to safely reduce the target time duration for the watch cache window (saving RAM), rather than using the space savings to keep history for longer.
- **Network Bandwidth:** Large `managedFields` arrays are transmitted over the wire.
- **Client-side GC:** Clients must allocate structural objects for `managedFields` before discarding them.

### Goals

- Provide a mechanism for clients to opt-out of receiving `metadata.managedFields` in GET, LIST, and WATCH responses.
- Reduce API Server CPU usage for serialization.
- Reduce network traffic between API Server and clients.
- Reduce client-side memory allocations and GC overhead.
- Potentially allow for a shorter watch cache window due to faster LIST responses.

### Non-Goals

- General-purpose field selection.
- Opting out of fields other than `metadata.managedFields` (though the mechanism may be extensible in the future).
- Opting out of fields in write operations (POST, PUT, PATCH).

## Proposal

### User Stories

#### Story 1
As a cluster operator or developer of a high-traffic controller (like the scheduler), I want to avoid the overhead of receiving `managedFields` because I don't use them for my reconciliation logic. This will save CPU on both the API server and my controller, and reduce network bandwidth.

#### Story 2
As a developer using `client-go`, I want to be able to easily opt-out of `managedFields` for certain informers where the data is redundant.

### Risks and Mitigations

- **Silent Data Loss in Clients:** If a client opts out but later attempts an operation that requires `managedFields` (e.g., using `managedfields.ExtractInto`), it might fail or behave unexpectedly.
  - **Mitigation:** Update `managedfields.ExtractInto` and related utilities to return clear errors when `managedFields` is missing.
- **Watch Cache Memory Overhead:** Mixed opt-in and opt-out watchers will cause the API server to maintain two serialized versions of each object in the `cachingObject` transient cache.
  - **Mitigation:** The `cachingObject` is transient and created per dispatch event. The cost is limited to one additional serialization per unique format. If all watchers opt out, the single cached version is smaller and cheaper to produce.

## Design Details

### Accept Parameter

We propose using an `Accept` header parameter to signal the request to exclude `managedFields`. 

Example:
`Accept: application/json; drop=metadata.managedFields`

This follows Kubernetes API conventions where `Accept` parameters are used for structural transformations (e.g., `as=PartialObjectMetadata`, `as=Table`).

While the `drop` parameter syntax is designed such that it wouldn't look out of place if a general field selection feature were added in the future, this KEP is strictly scoped to `metadata.managedFields`. We are not proposing or pre-approving a general-purpose field filtering mechanism at this time, as that would require a much broader discussion.

### Implementation Details

1.  **API Server Encoder:** Extend the API server's encoders to support a flag for excluding `managedFields`. When this flag is set, the encoder will skip the `managedFields` field during serialization.
2.  **Watch Cache (Cacher):** The `cachingObject` in the watch cache will be updated to include the exclusion flag in its serialization cache key. This ensures that mixed opt-in and opt-out watchers correctly receive their respective serialized forms.
3.  **Discovery:** The capability should be discoverable, likely via the supported media types in the API discovery.
4.  **Client-side Mitigation:** Update `managedfields.ExtractInto` to return an explicit error if `managedFields` is missing or nil. This prevents clients that have opted out from accidentally performing operations that require `managedFields` (like certain "extract-modify-patch" workflows).

### Test Plan

[x] I/we understand the owners of the involved components may require updates to
existing tests to make this code reproducible.

#### Prerequisite testing updates

None.

#### Unit Tests

- Test API server encoders with and without the `managedFields` exclusion flag.
- Test `cachingObject` serialization cache hits and misses with the exclusion flag.
- Test `managedfields.ExtractInto` behavior when `managedFields` is missing.

#### Integration Tests

- Verify that GET, LIST, and WATCH requests with the `Accept` parameter correctly return objects without `managedFields`.
- Verify that standard requests (without the parameter) still return `managedFields`.
- Verify mixed watch scenarios with both opt-in and opt-out clients.

#### e2e Tests

- Ensure core components (e.g., scheduler) can successfully opt-out and function correctly.

### Graduation Criteria

#### Alpha

- Feature implemented. Two feature gates are introduced:
  - `APIServerDropManagedFields` (Server-side): Introduced at **Beta** and enabled by default, as the functionality is purely optional and depends entirely on the client requesting it via the `Accept` header.
  - `ClientDropManagedFields` (Client-side): Introduced at **Alpha** and disabled by default. This gate controls whether internal Kubernetes clients (e.g., `kube-scheduler`, `kube-controller-manager`) send the `Accept` header to drop `managedFields`.
- Support for GET, LIST, and WATCH operations in the API server.
- `Accept` parameter `drop=metadata.managedFields` recognized by the API server.
- `managedfields.ExtractInto` updated with error handling.

#### Beta

- `ClientDropManagedFields` is promoted to Beta and enabled by default.
- Integration with major clients/controllers (e.g., kube-scheduler, kube-controller-manager) to use the opt-out is enabled by default.
- Performance benchmarks confirming savings in API server and clients.

#### GA

- Both feature gates removed.
- Full documentation on usage and benefits.

### Upgrade / Downgrade Strategy

- **Upgrade:** New API servers will recognize the `Accept` parameter. Old clients will continue to work as before (not sending the parameter, thus receiving `managedFields`).
- **Downgrade:** If a client starts using the parameter and the API server is downgraded to a version that doesn't recognize it, the API server will ignore the unknown parameter and return the full object with `managedFields`. The client will receive more data than requested but should be able to handle it (ignoring the field if it doesn't need it).

### Version Skew Strategy

Supported. API servers that do not recognize the `drop` parameter will simply ignore it and return the full object, which is a safe default.

## Production Readiness Questionnaire

### Feature Enablement and Rollback

###### How can this feature be enabled / disabled in a live cluster?

- [x] **Feature gate (also fill in values in `kep.yaml`)**
  - Feature gate name: `APIServerDropManagedFields` and `ClientDropManagedFields`
  - Components depending on the feature gate: `kube-apiserver` (for `APIServerDropManagedFields`), `kube-scheduler`, `kube-controller-manager` (for `ClientDropManagedFields`)

###### Does enabling the feature change any default behavior?

No. The API server behavior only changes when clients explicitly opt-out using the `Accept` header parameter. Enabling the client-side feature gate will cause those specific clients to start omitting the data from their informers and responses.

###### Can the feature be disabled once it has been enabled (i.e. can we roll back the enablement)?

Yes. Disabling `APIServerDropManagedFields` will cause the API server to ignore the `drop` parameter and always return `managedFields`. Disabling `ClientDropManagedFields` will cause clients to stop sending the `Accept` header. Feature gates are typically disabled by setting the flag to `false` and restarting the component.

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

- 2026-03-10: Enhancement issue created.
- 2026-03-30: PoC PR opened.
- 2026-04-14: KEP drafted.

## Drawbacks

- Slight complexity increase in API server encoding logic.
- Potential for two cached serialized versions of the same object in the watch cache.
- Adds to the technical performance debt by introducing another permutation of object representation (e.g., Protobuf without `managedFields`). This increases the matrix of combinations (format + options) that must be tracked, tested, and optimized in the API server over time.

## Alternatives

- **ListOptions/GetOptions flag:** e.g., `excludeManagedFields=true`. Rejected because `Accept` parameters are the standard way in Kubernetes to control object representation.
- **General-purpose field selector:** A much larger undertaking that has been discussed for years. This KEP provides a targeted solution for a high-impact field while leaving room for future generalization. Note that a general solution would likely require a different implementation approach—such as walking the serialized form (Protobuf or CBOR) to dynamically emit the filtered response—rather than modifying Go structs before serialization as done in this targeted KEP.

## Infrastructure Needed (Optional)

None.
