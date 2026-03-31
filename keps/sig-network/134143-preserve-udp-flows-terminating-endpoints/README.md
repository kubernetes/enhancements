# KEP-134143: Preserve UDP flows for terminating endpoints

<!-- toc -->
- [Release Signoff Checklist](#release-signoff-checklist)
- [Summary](#summary)
- [Motivation](#motivation)
  - [Goals](#goals)
  - [Non-Goals](#non-goals)
- [Proposal](#proposal)
  - [Risks and Mitigations](#risks-and-mitigations)
- [Design Details](#design-details)
  - [Test Plan](#test-plan)
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
- [Implementation History](#implementation-history)
- [Drawbacks](#drawbacks)
- [Alternatives](#alternatives)
<!-- /toc -->

## Release Signoff Checklist

- [ ] (R) Enhancement issue in release milestone, which links to KEP dir in [kubernetes/enhancements] (not the initial KEP PR)
- [ ] (R) KEP approvers have approved the KEP status as `implementable`
- [ ] (R) Design details are appropriately documented
- [ ] (R) Test plan is in place, giving consideration to SIG Architecture and SIG Testing input (including test refactors)
- [ ] (R) Graduation criteria is in place
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

This KEP proposes the preservation of UDP flows for terminating endpoints that are still processing existing connections. It introduces a new field `Processing` to `EndpointConditions` in the EndpointSlice API (v1 and v1beta1). Kube-proxy is updated to respect this field during UDP conntrack cleanup, ensuring that UDP flows are not prematurely cleared for endpoints that are in the process of terminating but still have active traffic.

## Motivation

Currently, when an endpoint is terminating, its UDP flows might be prematurely cleared by kube-proxy's conntrack cleanup logic. This can lead to disrupted connections for UDP-based applications that require some time to finish processing existing traffic even after the endpoint has been marked for termination.

### Goals

- Implement the preservation of UDP flows for terminating endpoints that are still processing traffic.
- Add a `Processing` field to `EndpointConditions` in the EndpointSlice API to signal this state.
- Update kube-proxy to use this field to avoid premature UDP conntrack cleanup.

### Non-Goals

- This KEP does not aim to change how TCP connections are handled, as they already have built-in mechanisms for graceful termination.
- This KEP does not change the general lifecycle of endpoints beyond adding the `Processing` condition.

## Proposal

We propose adding a `Processing` boolean field to the `EndpointConditions` struct in the `discovery.k8s.io` API (both `v1` and `v1beta1`).

```go
type EndpointConditions struct {
    // ... existing fields
    // Processing indicates that an endpoint is terminating but still
    // processing existing connections.
    // +optional
    Processing *bool `json:"processing,omitempty"`
}
```

Kube-proxy will be updated to check this field. If `Processing` is true, kube-proxy will refrain from clearing conntrack entries for that endpoint, even if it is otherwise considered terminating.

### Risks and Mitigations

- **Risk**: Stale UDP flows might persist longer than necessary if the `Processing` field is not correctly managed.
- **Mitigation**: The controller responsible for setting the `Processing` field must ensure it is eventually set to false or the endpoint is removed once processing is complete.

## Design Details

### Test Plan

- Unit tests for API changes and conversions.
- Unit tests for kube-proxy's `EndpointSliceCache` and conntrack cleanup logic.
- Integration tests to verify the propagation of the `Processing` field.
- End-to-end tests to verify that UDP traffic continues to flow to a terminating endpoint while `Processing` is true.

##### Unit tests

- `pkg/proxy/endpointslicecache.go`
- `pkg/proxy/conntrack/cleanup.go`
- `pkg/proxy/conntrack/cleanup_test.go`

##### Integration tests

- EndpointSlice controller tests.

##### e2e tests

- Custom e2e test involving a UDP server that takes time to terminate and continues to receive packets.

### Graduation Criteria

#### Alpha

- Feature implemented behind the `PreserveUDPFlowsTerminatingEndpoints` feature gate.
- Initial unit and integration tests completed.

#### Beta

- Gather feedback from users.
- e2e tests enabled in CI.
- Default enablement of the feature gate.

#### GA

- Feature gate locked to true.
- Conformance tests added.

### Upgrade / Downgrade Strategy

- During upgrade, new kube-proxies will start respecting the `Processing` field if the API server supports it.
- During downgrade, old kube-proxies will ignore the `Processing` field and revert to the old behavior (potentially premature cleanup).

### Version Skew Strategy

- If the API server is older and doesn't support the `Processing` field, kube-proxy will see it as nil/false and behave as before.
- If kube-proxy is older, it will ignore the field even if present in the API.

## Production Readiness Review Questionnaire

### Feature Enablement and Rollback

###### How can this feature be enabled / disabled in a live cluster?

- [x] Feature gate (also fill in values in `kep.yaml`)
  - Feature gate name: PreserveUDPFlowsTerminatingEndpoints
  - Components depending on the feature gate: kube-proxy, kube-apiserver

###### Does enabling the feature change any default behavior?

Yes, it changes how kube-proxy handles UDP conntrack entries for terminating endpoints, potentially keeping them longer than before if the `Processing` field is set.

###### Can the feature be disabled once it has been enabled (i.e. can we roll back the enablement)?

Yes, by disabling the feature gate. Kube-proxy will revert to its previous behavior.

###### What happens if we reenable the feature if it was previously rolled back?

Kube-proxy will resume respecting the `Processing` field for conntrack cleanup.

###### Are there any tests for feature enablement/disablement?

Yes, unit tests will cover behavior with the feature gate both enabled and disabled.

## Implementation History

- 2026-03-31: Initial KEP draft (provisional).

## Drawbacks

- Slightly increased complexity in EndpointSlice API and kube-proxy logic.
- Potential for conntrack table to grow if `Processing` is set for too many endpoints for too long.

## Alternatives

- Increasing UDP conntrack timeouts globally: This would affect all flows and is not as targeted as this proposal.
- Implementing application-level heartbeats: This puts the burden on the application and doesn't solve the network-level cleanup issue.
