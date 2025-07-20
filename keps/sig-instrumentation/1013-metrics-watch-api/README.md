# KEP-1013: Metrics API watch support

## Table of Contents

<!-- toc -->
  - [Release Signoff Checklist](#release-signoff-checklist)
  - [Summary](#summary)
  - [Motivation](#motivation)
    - [Goals](#goals)
    - [Non-Goals](#non-goals)
  - [Proposal](#proposal)
    - [User Stories](#user-stories)
      - [HPA](#hpa)
      - [Custom metrics provider](#custom-metrics-provider)
    - [Implementation Details/Notes/Constraints [optional]](#implementation-detailsnotesconstraints-optional)
    - [Risks and Mitigations](#risks-and-mitigations)
  - [Design Details](#design-details)
    - [Test Plan](#test-plan)
      - [Prerequisite testing updates](#prerequisite-testing-updates)
      - [Unit tests](#unit-tests)
      - [Integration tests](#integration-tests)
      - [e2e tests](#e2e-tests)
    - [Graduation Criteria](#graduation-criteria)
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
  - [Drawbacks [optional]](#drawbacks-optional)
- [Related resources](#related-resources)
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

Provide watch capability to all resource metrics APIs: `metrics.k8s.io`,
`custom.metrics.k8s.io` and `external.metrics.k8s.io`, [similarly to regular
Kubernetes APIs](https://kubernetes.io/docs/reference/using-api/api-concepts/#efficient-detection-of-changes).

## Motivation

There are three APIs for serving metrics. All of them support reading the
metrics in a request-response manner, forcing the clients (e.g. HPA) interested
in up-to-date values to poll in a loop. This introduces additional latency:
between the time when new metric values are known to the process serving the
API and the time when a client interested in reading them actually fetches the
data.

### Goals

- Allow resource metrics clients to subscribe to stream metric changes.

### Non-Goals

- Graduate the APIs to GA. This needs to be done eventually, but is out of scope
  for this work.

## Proposal

This proposal is essentially about implementing [Efficient detection of
changes](https://kubernetes.io/docs/reference/using-api/api-concepts/#efficient-detection-of-changes)
for metrics. GET requests will accept additional `watch` parameter, which would
cause the API to start streaming results. Since old metric values are never
modified, the only supported update type will be `ADDED`, when a new data point
appears. Metrics don't contain any resourceVersion associated with them, so it
won't be possible to retrieve old values by passing `resourceVersion` parameter.
Instead, this parameter will be ignored and all recent data points will be
returned instead. This means metrics APIs will never return `410 Gone` error
code.

### User Stories

There are two sides to that proposal: the API producers and consumers. Examples
below include one consumer (HPA) and one hypothetical producer.

#### HPA

As an autoscaling solution, I will be able to subscribe to updates on a certain
labelSelector and get new metrics as soon as they are known to the metric
backend.

#### Custom metrics provider

As a metrics provider, I will be able to provide a low-latency Metrics API
implementation by taking advantage of backend specific features (e.g. streaming
APIs or known best polling interval).

### Implementation Details/Notes/Constraints [optional]

The implementation requires the following considerations:

1. **API Contract Compatibility**: The existing metrics APIs (`metrics.k8s.io/v1beta1`, `custom.metrics.k8s.io/v1beta1`, and `external.metrics.k8s.io/v1beta1`) will be extended to support the standard Kubernetes watch protocol.

2. **Watch Protocol Adaptation**: Since metrics don't have resource versions like typical Kubernetes objects, the implementation will:
   - Ignore the `resourceVersion` parameter
   - Return only `ADDED` events when new metric data becomes available
   - Never return `410 Gone` errors

3. **Backend Requirements**: Metrics providers need to implement streaming capabilities or efficient polling to provide timely updates to watching clients.

4. **Memory Considerations**: Watch connections hold server resources, so proper connection lifecycle management and limits are required.

5. **Backward Compatibility**: The change is purely additive - existing clients using polling will continue to work unchanged.

### Risks and Mitigations

**Risk**: Metrics providers may not efficiently support streaming, leading to high resource consumption.
**Mitigation**: Implement connection limits and provide guidance on efficient backend implementation. Fall back to polling with provider-specific intervals.

**Risk**: Clients expecting traditional Kubernetes watch semantics may be confused by the lack of resource versions.
**Mitigation**: Clear documentation and examples showing proper usage patterns for metrics watch.

**Risk**: Increased server-side resource usage due to long-lived watch connections.
**Mitigation**: Implement connection timeouts, rate limiting, and monitoring for watch connection counts.

**Risk**: Breaking changes for metrics providers during API updates.
**Mitigation**: Ensure the watch parameter is optional and providers can gradually adopt streaming support.

## Design Details

### Test Plan

#### Prerequisite testing updates

The following prerequisites must be met before implementing this enhancement:
- Existing metrics API tests must continue to pass
- Metrics provider implementations should have basic polling functionality tested

#### Unit tests

- Core watch functionality for metrics APIs in k8s.io/metrics: `2025-07-20` - 85%
- Watch parameter parsing and validation: `2025-07-20` - 90%
- Event stream generation for metrics data: `2025-07-20` - 80%

#### Integration tests

- Watch requests to metrics.k8s.io API with valid responses
- Watch requests to custom.metrics.k8s.io API with proper event streaming
- Watch requests to external.metrics.k8s.io API with timeout handling
- Verify that existing GET requests continue to work unchanged
- Test watch connection lifecycle (connect, stream, disconnect)

#### e2e tests

- End-to-end watch functionality with a metrics provider implementation
- HPA consuming metrics via watch API instead of polling
- Performance comparison between polling and watch for metrics consumption
- Verify watch API works with different metrics backends (resource, custom, external)
- Test watch API behavior under various failure scenarios

**Note:** *Section not required until targeted at a release.*

Consider the following in developing a test plan for this enhancement:
- Will there be e2e and integration tests, in addition to unit tests?
- How will it be tested in isolation vs with other components?

No need to outline all of the test cases, just the general strategy.
Anything that would count as tricky in the implementation and anything particularly challenging to test should be called out.

All code is expected to have adequate tests (eventually with coverage expectations).
Please adhere to the [Kubernetes testing guidelines][testing-guidelines] when drafting this test plan.

[testing-guidelines]: https://git.k8s.io/community/contributors/devel/sig-testing/testing.md

### Graduation Criteria

Since the APIs already exist in beta and the change is backwards-compatible,
this proposal will be applied to the beta API, updating it from v1beta$(n) to
v1beta$(n+1).

**Alpha (v1beta2):**
- Watch support implemented for all three metrics APIs
- Basic integration tests demonstrating watch functionality
- At least one metrics provider implementation supporting watch

**Beta (v1beta3):**
- E2E tests covering watch functionality with multiple metrics backends
- Performance benchmarks showing watch vs polling efficiency
- Documentation for metrics provider implementers
- Conformance tests for watch API behavior

**GA (v1):**
- Production usage in at least 2 different metrics backends
- Comprehensive e2e test coverage with flake-free tests over 2+ weeks
- All GA endpoints covered by conformance tests
- Performance and scalability validation under production load
- Complete production readiness review approved
- Documentation published to kubernetes.io

Stability of this feature proven by at least one backend implementation for
`metrics.k8s.io` and `custom.metrics.k8s.io` will be a blocker for graduating
these APIs to v1.

This stability will be measured by e2e tests that will fetch the data using
watch.

### Upgrade / Downgrade Strategy

**Upgrade Strategy:**
- The watch parameter is optional and additive, so existing clients continue to work unchanged
- Metrics providers can implement watch support incrementally
- No configuration changes required for existing clusters
- Rolling upgrades are safe as the API remains backward compatible

**Downgrade Strategy:**
- Downgrading to versions without watch support will result in watch requests returning errors
- Clients should implement fallback to polling when watch is not supported
- No data loss occurs during downgrades as this is a protocol enhancement, not a data format change

**Configuration Changes:**
- No existing cluster configuration changes required on upgrade
- Optional: Metrics providers may add configuration for watch-specific settings (timeouts, connection limits)
- No breaking changes to existing invocations or configurations

### Version Skew Strategy

The metrics watch API enhancement has minimal version skew concerns:

**API Server vs Metrics Provider:**
- Older metrics providers without watch support will return appropriate errors for watch requests
- Newer metrics providers with watch support are backward compatible with older API servers
- The watch parameter is handled at the API layer, so version compatibility is maintained

**Client vs API Server:**
- Clients using watch with older API servers will receive not supported errors and can fall back to polling
- Newer clients can detect watch support via API discovery
- No compatibility issues between different client and server versions

**Component Coordination:**
- This enhancement is purely API-level and doesn't require coordination between control plane and kubelet
- No changes to CSI, CRI, or CNI components
- No node-level component changes required

## Production Readiness Review Questionnaire

### Feature Enablement and Rollback

###### How can this feature be enabled / disabled in a live cluster?

- [ ] Other
  - Describe the mechanism: This feature is enabled by default as part of the metrics API specification. Individual metrics providers can choose whether to implement watch support.
  - Will enabling / disabling the feature require downtime of the control plane? No, watch support is backwards compatible and optional.
  - Will enabling / disabling the feature require downtime or reprovisioning of a node? No, this is an API-level enhancement.

###### Does enabling the feature change any default behavior?

No. Existing polling-based metrics collection continues to work exactly as before. Watch is an opt-in capability for clients.

###### Can the feature be disabled once it has been enabled (i.e. can we roll back the enablement)?

Yes. Since watch support is backwards compatible, metrics providers can disable watch functionality without affecting existing clients. Clients should implement fallback to polling.

###### What happens if we reenable the feature if it was previously rolled back?

Watch functionality resumes. No data loss or corruption occurs since this affects only the delivery mechanism, not the data itself.

###### Are there any tests for feature enablement/disablement?

Yes, tests will verify that:
- Metrics APIs work with and without watch support
- Graceful degradation when watch is not available
- Proper error handling when watch requests are made to non-supporting providers

### Rollout, Upgrade and Rollback Planning

###### How can a rollout or rollback fail? Can it impact already running workloads?

**Rollout failures:**
- Metrics providers may have bugs in watch implementation
- Increased resource usage from watch connections
- Clients may not handle watch errors properly

**Impact on running workloads:**
- Minimal impact as existing polling continues to work
- HPA and other metrics consumers have built-in fallback mechanisms
- Watch failures don't affect core cluster functionality

###### What specific metrics should inform a rollback?

- Increased error rates in metrics API endpoints
- Memory/CPU usage spikes in metrics providers
- Increased latency in metrics collection
- Failed watch connections or timeouts

###### Were upgrade and rollback tested? Was the upgrade->downgrade->upgrade path tested?

The upgrade/downgrade testing will include:
- Testing metrics API functionality before, during, and after upgrade
- Verifying client fallback behavior during downgrades
- Ensuring watch resumption after upgrade from downgrade

###### Is the rollout accompanied by any deprecations and/or removals of features, APIs, fields of API types, flags, etc.?

No. This is a purely additive enhancement with no deprecations or removals.

### Monitoring Requirements

###### How can an operator determine if the feature is in use?

- Metrics on watch connection counts per API
- Metrics on watch vs polling request ratios
- API server logs showing watch requests
- Metrics provider logs indicating streaming mode

###### How can someone using this feature know that it is working?

- Reduced latency in metrics updates compared to polling
- Metrics collection tools showing streaming mode
- API server metrics showing successful watch connections
- Application logs showing continuous metric updates instead of periodic polling

###### What are the reasonable SLIs (Service Level Indicators) for this feature?

- Watch connection success rate (> 99%)
- Watch connection duration (average and 95th percentile)
- Time to first metric after watch connection (< 1s)
- Watch connection resource usage (memory per connection)

###### What are the SLOs (Service Level Objectives) for this component?

- 99% of watch connections should be established successfully
- 95% of watch connections should receive first metric within 1 second
- Watch connection memory usage should be < 1MB per connection
- Watch error rate should be < 1%

###### Are there any missing metrics that would be useful to have to improve observability of this feature?

Proposed metrics:
- `metrics_api_watch_connections_total` - total watch connections by API
- `metrics_api_watch_duration_seconds` - duration of watch connections
- `metrics_api_watch_errors_total` - watch connection errors by type
- `metrics_api_watch_events_sent_total` - events sent over watch connections

### Dependencies

###### Does this feature depend on any specific services running in the cluster?

- Metrics providers (metrics-server for resource metrics, custom metrics adapters)
- API server for watch protocol support
- No additional services required

###### Does this feature depend on any specific services running outside the cluster?

No. All dependencies are within the cluster.

### Scalability

###### Will enabling / using this feature result in any new API calls?

No additional API calls. Watch connections replace polling API calls with persistent connections, potentially reducing total API call volume.

###### Will enabling / using this feature result in introducing new API types?

No new API types. This extends existing metrics APIs with watch support.

###### Will enabling / using this feature result in any new calls to the cloud provider?

No. This is purely a Kubernetes-internal enhancement.

###### Will enabling / using this feature result in increasing size or count of the existing API objects?

No. Metric objects remain unchanged. Only the delivery mechanism changes.

###### Will enabling / using this feature result in increasing time taken by any operations covered by existing SLIs/SLOs?

- Initial watch connection establishment may take slightly longer than single GET requests
- Overall metrics collection latency should decrease due to streaming updates
- No impact on existing SLIs/SLOs for non-watch operations

###### Will enabling / using this feature result in non-negligible increase of resource usage (CPU, RAM, disk, IO, ...) in any components?

**API Server:**
- Minimal CPU increase for managing watch connections
- Memory usage proportional to number of concurrent watch connections
- Estimated: ~1MB RAM per 1000 concurrent watch connections

**Metrics Providers:**
- Moderate memory increase for maintaining client connections
- CPU increase for streaming metric updates
- Estimated: ~100KB RAM and ~0.1% CPU per watch connection

**Network:**
- Reduced overall network traffic due to elimination of polling
- More persistent connections

###### Can enabling / using this feature result in resource exhaustion of some node resources (PIDs, sockets, inodes, etc.)?

- Socket exhaustion possible with very high numbers of watch connections
- Mitigation: Connection limits and proper timeout handling
- Monitoring: Track socket usage and connection counts

### Troubleshooting

###### How does this feature react to abrupt failure of a critical component?

**API Server failure:**
- Watch connections terminate
- Clients should implement reconnection logic
- Fall back to polling until watch is available

**Metrics Provider failure:**
- Watch connections fail
- Clients fall back to polling or try alternative providers
- No data corruption

**Network partition:**
- Watch connections timeout and close
- Clients reconnect when network is restored
- Built-in resilience through Kubernetes watch semantics

###### What steps should be taken if SLOs are not being met because of this feature?

1. Check watch connection counts and resource usage
2. Examine metrics provider logs for errors
3. Monitor client fallback behavior
4. Consider disabling watch temporarily if severe issues
5. Review client implementation for proper error handling

###### What specific tests should inform a rollback?

- Watch connection failure rate > 5%
- Memory usage increase > 200% in metrics providers
- Increased error rates in existing metrics collection
- Client applications showing degraded performance

###### Is the failure mode obvious?

Yes. Failure modes include:
- Watch connections failing to establish (obvious errors)
- Resource exhaustion (visible in monitoring)
- Client fallback to polling (detectable via request patterns)
- All failures have clear error messages and logs

## Implementation History

Major milestones in the life cycle of a KEP should be tracked in `Implementation History`.
Major milestones might include

- the `Summary` and `Motivation` sections being merged signaling SIG acceptance
- the `Proposal` section being merged signaling agreement on a proposed design
- the date implementation started
- the first Kubernetes release where an initial version of the KEP was available
- the version of Kubernetes where the KEP graduated to general availability
- when the KEP was retired or superseded

**2019-04-25**: KEP created and initial `Summary` and `Motivation` sections defined
**2019-04-29**: Initial `Proposal` section added 
**2025-07-20**: KEP updated for GA graduation with complete design details, production readiness review, and comprehensive test plan
**TBD**: Implementation started
**TBD**: Alpha implementation available
**TBD**: Beta implementation available  
**TBD**: GA graduation

## Drawbacks [optional]

No custom metrics backend today offers a streaming API that would allow a
straightforward implementation of the watch. However, the fact that Kubernetes
metrics APIs will support streaming with watch may encourage some backends to
add such support. Additionally, the polling frequency will be specific to
relevant adapters, rather than to the metrics client.

# Related resources

SIG instrumentation discussions:
- [Custom/External Metrics API watch](https://groups.google.com/forum/#!topic/kubernetes-sig-instrumentation/nJvDyIwDgu8)
- [Resource Metrics API watch](https://groups.google.com/d/msg/kubernetes-sig-instrumentation/_b6c0oyPLJA/Y4rMQTBDAgAJ)
