# KEP-5989: Preserve UDP flows for stateful UDP protocols during endpoint termination

<!-- toc -->
- [Release Signoff Checklist](#release-signoff-checklist)
- [Summary](#summary)
- [Motivation](#motivation)
  - [Goals](#goals)
  - [Non-Goals](#non-goals)
- [Proposal](#proposal)
  - [Implementation Approach](#implementation-approach)
  - [Conntrack Cleanup Strategy](#conntrack-cleanup-strategy)
  - [Kube-proxy Changes](#kube-proxy-changes)
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

This KEP proposes a mechanism to preserve UDP conntrack entries for stateful UDP protocols during endpoint termination. The approach leverages existing netfilter operations and conntrack marks to indicate which UDP flows are stateful and should not be cleared during endpoint termination. Kube-proxy will be updated to use conntrack marks to track stateful UDP flows and implement a deferred cleanup strategy.

## Motivation

Certain UDP-based protocols are stateful and maintain sessions that should not be disrupted when an endpoint is terminating. Currently, kube-proxy's conntrack cleanup logic clears UDP conntrack entries when endpoints enter the terminating state, which can disrupt existing stateful UDP connections (such as TFCP, QUIC, and other stateful UDP protocols).

The original issue stems from the fact that some UDP protocols behave like connection-oriented protocols and require graceful termination handling similar to TCP, rather than the stateless datagram model that typical UDP applications use.

### Goals

- Provide a mechanism to mark stateful UDP flows using conntrack marks so they are preserved during endpoint termination.
- Update kube-proxy to respect conntrack marks and implement a deferred cleanup strategy for marked flows.
- Solve the problem by leveraging existing netfilter operations, following patterns from secure-conntrack-helpers.
- Define a clear cleanup strategy to avoid stale conntrack entries.

### Non-Goals

- This KEP does not aim to change the endpoint lifecycle or add new states to EndpointConditions.
- This KEP does not introduce new API fields in EndpointSlice or other API types.
- This KEP does not change how TCP connections are handled, as they already have built-in mechanisms for graceful termination.
- This KEP does not introduce new Service annotations (per maintainer feedback, annotations are a poor API).

## Proposal

This proposal leverages existing netfilter operations to handle stateful UDP flows. Instead of using Service annotations (which are a poor API), we use conntrack marks to identify and preserve stateful UDP flows during endpoint termination, following patterns from the [secure-conntrack-helpers](https://github.com/regit/secure-conntrack-helpers/blob/master/secure-conntrack-helpers.rst) project.

### Implementation Approach

The approach uses conntrack marks to identify UDP flows that should be preserved during endpoint termination:

1. **Marking stateful UDP flows**: Kube-proxy (or a userspace component) can use the CT target to mark conntrack entries for stateful UDP protocols. This follows the pattern of using `iptables -t raw -j CT --mark <value>` to set marks on conntrack entries.

2. **Deferred cleanup**: Instead of immediately clearing conntrack entries for terminating endpoints, kube-proxy will:
   - Check if the conntrack entry has a mark indicating it's a stateful UDP flow
   - If marked, defer the cleanup and set a timeout-based cleanup
   - If not marked, proceed with immediate cleanup as before

3. **Cleanup strategy**: Stateful UDP flows will be cleaned up after a configurable timeout (e.g., based on the expected session duration of the stateful protocol) or when the endpoint is fully removed from the cluster.

### Conntrack Cleanup Strategy

The key insight from the maintainer's feedback is that flows must eventually be cleaned up to avoid blackholing future entries. The proposed cleanup strategy:

1. **During endpoint termination**: Kube-proxy identifies stateful UDP flows via conntrack marks and skips immediate cleanup.

2. **Deferred cleanup**: After the endpoint transitions to terminating state, kube-proxy will:
   - Track the termination timestamp for the endpoint
   - Periodically check for conntrack entries that should be cleaned up (e.g., after a grace period)
   - Clean up stateful UDP conntrack entries after the grace period expires

3. **Grace period**: A configurable grace period (default: 30 seconds, matching typical UDP conntrack timeouts for stateful protocols) will be used before forcefully cleaning up marked conntrack entries.

### Kube-proxy Changes

Kube-proxy will be updated to:

1. **Detect conntrack marks**: When performing conntrack cleanup, check for conntrack marks indicating stateful UDP flows.

2. **Deferred cleanup tracking**: Maintain a map of terminating endpoints with stateful UDP flows, including their termination timestamps.

3. **Periodic cleanup**: Implement a periodic cleanup routine that removes conntrack entries for stateful UDP flows after the grace period expires.

4. **Integration with conntrack tools**: Leverage existing conntrack tools and netfilter operations to manage marked flows.

This approach:
- Does not require any API changes (no new fields in EndpointConditions or EndpointSlice).
- Uses existing netfilter operations (CT target, conntrack marks) instead of annotations.
- Is specific to kube-proxy's behavior (conntrack cleanup).
- Defines a clear cleanup strategy to avoid stale entries.
- Keeps the endpoint lifecycle unchanged (endpoints still transition through the same states: ready → terminating → removed).

### Risks and Mitigations

- **Risk**: Stale conntrack entries might persist if the deferred cleanup fails.
- **Mitigation**: The grace period ensures cleanup will happen eventually. Additionally, conntrack entries will still time out based on conntrack timeout settings.

- **Risk**: Complexity in tracking terminating endpoints and their grace periods.
- **Mitigation**: Reuse existing endpoint tracking infrastructure in kube-proxy's EndpointSliceCache.

- **Risk**: Users may not properly mark their stateful UDP flows.
- **Mitigation**: Documentation will explain how to use conntrack marks for stateful UDP protocols. Consider providing examples or helper scripts.

## Design Details

### Test Plan

- Unit tests for kube-proxy's conntrack mark detection logic.
- Unit tests for kube-proxy's deferred cleanup tracking.
- Unit tests for kube-proxy's `EndpointSliceCache` and conntrack cleanup logic to verify stateful UDP flows are preserved during termination and cleaned up after grace period.
- Integration tests to verify conntrack marks are properly respected.
- End-to-end tests to verify that stateful UDP traffic continues to flow to a terminating endpoint and is eventually cleaned up.

##### Unit tests

- `pkg/proxy/endpointslicecache.go`
- `pkg/proxy/conntrack/cleanup.go`
- `pkg/proxy/conntrack/cleanup_test.go`
- `pkg/proxy/conntrack/mark_tracking.go` (new file for mark tracking logic)

##### Integration tests

- Conntrack cleanup tests with marked and unmarked flows.
- Grace period expiration tests.

##### e2e tests

- Custom e2e test involving a stateful UDP server (e.g., QUIC) that continues to receive packets during termination and verify cleanup after grace period.

### Graduation Criteria

#### Alpha

- Feature implemented behind the `PreserveUDPFlowsTerminatingEndpoints` feature gate.
- Initial unit and integration tests completed.
- Support for conntrack mark detection and deferred cleanup.

#### Beta

- Gather feedback from users.
- e2e tests enabled in CI.
- Default enablement of the feature gate.
- Documented guidance on using conntrack marks for stateful UDP protocols.

#### GA

- Feature gate locked to true.
- Conformance tests added.
- Proven reliability of deferred cleanup mechanism.

### Upgrade / Downgrade Strategy

- During upgrade, new kube-proxies will start using conntrack marks to preserve stateful UDP flows during endpoint termination.
- During downgrade, old kube-proxies will ignore conntrack marks and revert to the old behavior (clearing all conntrack entries for terminating endpoints).

### Version Skew Strategy

- If kube-proxy is newer and conntrack marks are present, it will respect them and defer cleanup.
- If kube-proxy is older, it will ignore conntrack marks and clear conntrack entries as before.
- Endpoints may have a mix of marked and unmarked flows during version skew.

## Production Readiness Review Questionnaire

### Feature Enablement and Rollback

###### How can this feature be enabled / disabled in a live cluster?

- [x] Feature gate (also fill in values in `kep.yaml`)
  - Feature gate name: PreserveUDPFlowsTerminatingEndpoints
  - Components depending on the feature gate: kube-proxy

###### Does enabling the feature change any default behavior?

Yes, kube-proxy will preserve UDP conntrack entries marked as stateful during endpoint termination, and implement a deferred cleanup after a grace period, instead of clearing them immediately.

###### Can the feature be disabled once it has been enabled (i.e. can we roll back the enablement)?

Yes, by disabling the feature gate. Kube-proxy will revert to its previous behavior of immediately clearing all conntrack entries for terminating endpoints.

###### What happens if we reenable the feature if it was previously rolled back?

Kube-proxy will resume using conntrack marks to preserve stateful UDP flows and implement deferred cleanup.

###### Are there any tests for feature enablement/disablement?

Yes, unit tests will cover behavior with the feature gate both enabled and disabled.

## Implementation History

- 2026-03-31: Initial KEP draft (provisional) with API changes (Processing field).
- 2026-05-02: Revised to use Service annotation instead of API changes.
- 2026-05-02: Revised to use conntrack marks and netfilter operations instead of annotations, addressing maintainer feedback about poor API and leveraging secure-conntrack-helpers patterns.

## Drawbacks

- More complex implementation compared to annotation-based approach.
- Requires understanding of netfilter/conntrack marks for users who want to mark their stateful UDP flows.
- Potential for conntrack table to grow if deferred cleanup fails or grace period is too long.

## Alternatives

- **Service annotation approach**: Rejected per maintainer feedback (annotations are a poor API).
- **Adding a `Processing` field to EndpointConditions**: Rejected as it requires API changes and adds a new state to the endpoint lifecycle, which is inappropriate for a kube-proxy specific behavior.
- **Increasing UDP conntrack timeouts globally**: This would affect all flows and is not as targeted as this proposal.
- **Implementing application-level heartbeats**: This puts the burden on the application and doesn't solve the network-level cleanup issue.
- **Making this a kube-proxy configuration option**: This would be less flexible, as it would apply to all Services without distinguishing stateful vs stateless UDP protocols.
