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

This KEP proposes a mechanism to preserve UDP conntrack entries for stateful UDP protocols during endpoint termination by introducing a "connection type hint" at the Service level. Instead of immediately clearing UDP conntrack entries when an endpoint enters the terminating state, kube-proxy will respect a per-port Service configuration that specifies whether UDP flows should be preserved until the pod actually exits (i.e., until the endpoint is removed from the EndpointSlice).

## Motivation

Certain UDP-based protocols are stateful (e.g., QUIC, TFTP, SIP) and maintain sessions that should not be disrupted when an endpoint is terminating but still capable of processing traffic. Currently, kube-proxy's conntrack cleanup logic clears all UDP conntrack entries as soon as an endpoint is marked as not ready (including when it enters the terminating state). This ensures that new packets are routed to healthy endpoints, but it abruptly breaks existing stateful sessions.

The root cause of this issue is that kube-proxy lacks the information to distinguish between "stateless" UDP (like DNS), where immediate failover is desirable, and "stateful" UDP, where session persistence is required.

By providing a hint at the Service level, users can inform the network layer about the nature of the UDP traffic, allowing kube-proxy to implement the appropriate cleanup strategy.

### Goals

- Introduce a Service-level hint to indicate if UDP flows on a specific port are stateful.
- Update kube-proxy to defer UDP conntrack cleanup for such ports until the endpoint is fully removed from the cluster.
- Ensure that existing "stateless" UDP behavior remains the default.
- Provide a robust mechanism that doesn't rely on application-level marking or racy external helpers.

### Non-Goals

- This KEP does not aim to change the endpoint lifecycle or add new states to EndpointConditions.
- This KEP does not introduce new API fields in EndpointSlice.
- This KEP does not change how TCP connections are handled.
- This KEP does not rely on conntrack marks or netfilter-specific marking operations.

## Proposal

We propose adding a new field (or a standardized hint) to the `ServicePort` API that specifies the UDP persistence strategy. This field, `udpConnectionHandling`, follows the "connection type hint" pattern, allowing the Service owner to inform the network layer about the stateful nature of the traffic.

**Alpha Strategy (Standardized Hint)**: To allow for rapid iteration and gather feedback without immediate core API changes, this feature may initially be implemented as a standardized annotation (e.g., `service.kubernetes.io/udp-connection-handling`). This provides a "fast-track" path for Alpha while the long-term API field is finalized.

### Implementation Approach

1. **Service API Change**: Add a field `udpConnectionHandling` (name to be finalized) to `ServicePort`.
   - `Default`: Current behavior (immediate cleanup on termination).
   - `PreserveDuringTermination`: Defer conntrack cleanup until the endpoint is removed from the EndpointSlice.

2. **Kube-proxy Logic**:
   - Kube-proxy tracks both the Service configuration and the EndpointSlice state.
   - When an endpoint becomes `Ready=False` but is still present in the EndpointSlice (e.g., `Terminating=True`):
     - If the port is configured with `PreserveDuringTermination`, kube-proxy **skips** the conntrack cleanup for that endpoint's IP and port.
     - Existing conntrack entries will remain, ensuring that subsequent packets from the same client are still NATed to the terminating pod.
   - **Dynamic Strategy Change**: If the `udpConnectionHandling` is changed from `PreserveDuringTermination` to `Default` while pods are already in a terminating state, kube-proxy will detect the change during its next sync loop and immediately execute the conntrack cleanup for those terminating endpoints.
   - When the endpoint is **removed** from the EndpointSlice:
     - Kube-proxy performs the standard conntrack cleanup, regardless of the `udpConnectionHandling`.

### Kube-proxy Changes

The implementation will focus on the `EndpointSliceCache` (or `EndpointsChangeTracker`) and the specific proxiers (`iptables`, `ipvs`, `nftables`).

1. **EndpointSliceCache Logic**:
   - The cache will be updated to track the `udpConnectionHandling` for each port.
   - When an endpoint transitions from `Ready` to `NotReady` (e.g., entering `Terminating` state), the cache logic that identifies "stale" UDP connections (`detectStaleConntrackEntries`) will check the strategy.
   - If the strategy is `PreserveDuringTermination`, the endpoint's IP and port will **not** be added to the stale list as long as the endpoint object still exists in the `EndpointSlice`.
   - The IP/port will only be marked as stale and added to the cleanup list when the endpoint is physically removed from the `EndpointSlice` resource.

2. **Iptables Proxier**:
   - In `syncProxyRules`, kube-proxy identifies stale UDP Service IPs and NodePorts.
   - **NAT Persistence**: In `iptables` mode, the `nat` table is only traversed by the first packet of a connection. Subsequent packets of an `ESTABLISHED` flow bypass the `nat` table rules and are handled by `conntrack`. By skipping the conntrack flush, existing flows continue to be correctly NATed to the terminating pod even after the specific `KUBE-SEP-XXX` chain has been unlinked from the service's load-balancing chain.
   - No changes are needed to the low-level `conntrack.ClearEntriesForIP` calls themselves; the logic resides in *when* those IPs are passed to the cleanup loop.

3. **IPVS Proxier**:
   - **Quiescent Mode**: When an endpoint enters the terminating state and the policy is `PreserveDuringTermination`, kube-proxy will set the Real Server weight to **0** instead of deleting it. In IPVS, weight 0 (quiescent) prevents new connections from being scheduled to the backend while allowing existing ones to persist.
   - **Conntrack Interaction**: By keeping the Real Server at weight 0 and skipping the conntrack flush, both the IPVS connection table and the Netfilter conntrack table maintain the session state. 
   - **Final Cleanup**: Once the endpoint is removed from the EndpointSlice, the Real Server is finally deleted from the IPVS Virtual Server, and the targeted conntrack flush is executed.

### Risks and Mitigations

- **Risk**: Stale conntrack entries might persist if a pod hangs in terminating state indefinitely (the "Zombie Flow" case).
- **Mitigation**: 
  - The standard `terminationGracePeriodSeconds` still applies. Once the pod is forcefully deleted, the endpoint is removed, and kube-proxy will clear the conntrack.
  - **Safety Valve (Hard Limit)**: To protect the node from conntrack table exhaustion, `kube-proxy` will implement a hard maximum preservation timeout (e.g., 2x the standard `nf_conntrack_udp_timeout_stream`, default 360s). This is a **hard cap**; once reached, the flow is forcefully flushed regardless of the pod's state.
  - **Kernel GC Interaction**: In the event of extreme conntrack table pressure (approaching `nf_conntrack_max`), the Linux kernel's "Early Drop" mechanism will naturally prioritize the eviction of unconfirmed or older entries. This KEP's safety valve serves as a userspace-driven protection layer that triggers before the kernel is forced to drop new legitimate traffic.
  - **Memory Pressure Monitoring**: The new metrics (see below) will allow cluster-level autoscalers or monitoring systems to alert on node-level conntrack pressure caused by deferred cleanups.
- **Risk**: NetworkPolicies might block traffic to terminating pods.
- **Mitigation**: This KEP focuses on `kube-proxy` behavior. For the feature to be fully effective, NetworkPolicies must allow traffic to pods in the `Terminating` state. Cluster administrators should ensure that egress/ingress policies are not so restrictive that they drop packets to unready pods that are still servicing stateful UDP flows. CNI providers should also align their policy enforcement logic with this graceful termination intent.
- **Risk**: Version Skew during rolling updates.
- **Mitigation**: During a cluster upgrade, some nodes may run the new `kube-proxy` while others run the old one. This leads to "best-effort" preservation: flows hitting upgraded nodes will be preserved, while flows on old nodes will be killed. While inconsistent, this is still an improvement over "guaranteed disruption". The metrics provided by this KEP will help operators track the adoption of the feature during the skew period.
- **Risk**: New flows might still be routed to terminating pods if not handled carefully.
- **Mitigation**: This KEP only affects *conntrack cleanup*. The iptables/ipvs rules (weight 0 or chain removal) will still be updated to remove the terminating endpoint from the load-balancing set for **new** flows. Only *existing* flows (tracked in conntrack/IPVS table) will be preserved.

### Design Details

#### Observability and Monitoring

To monitor the health of this feature and protect the node's networking stack, the following metrics will be introduced in `kube-proxy`:
- `kubeproxy_preserved_udp_conntrack_flows_total`: A gauge tracking the total number of UDP flows currently being preserved for terminating endpoints.
- `kubeproxy_conntrack_cleanup_deferred_total`: A counter incremented whenever a conntrack cleanup is deferred due to the `PreserveDuringTermination` strategy.
- `kubeproxy_conntrack_safety_valve_flushes_total`: A counter incremented whenever the hard safety valve timeout triggers a forced cleanup.

#### Coordination with External Load Balancers

For Services of `type: LoadBalancer`, traffic management is split between the Cloud Provider's Load Balancer (external) and `kube-proxy` (internal).
- **Primary Benefit**: This KEP is most beneficial for **East-West traffic (ClusterIP)** and **North-South traffic** where the external load balancer supports "Graceful Termination" or has inherent delays in health check propagation.
- **External LB Behavior**: Most Cloud Controllers (CCM) remove backends from the external LB as soon as they become `Ready=False`. If the external LB cuts traffic immediately, the benefits of this KEP for North-South traffic are limited.
- **Call to Action**: This KEP serves as the technical foundation for Cloud Providers to support graceful UDP termination. By providing a consistent node-level preservation mechanism, we enable Cloud Providers to implement matching "Graceful Shutdown" logic in their external LBs, ensuring end-to-end session persistence.
- **Node-level Protection**: This KEP ensures that if a packet *does* reach the node (either via internal traffic or because the external LB hasn't updated its state yet), the node's network stack will continue to route it to the terminating pod.


#### Conntrack Cleanup Granularity and Performance

To avoid "blackholing" while preserving sessions, the cleanup must be targeted:
- **During Termination**: Kube-proxy removes the endpoint from the load-balancing backend list (iptables chains or IPVS real servers). This stops **new** flows.
- **Conntrack Preservation**: By NOT calling the conntrack flush, the kernel continues to use the existing NAT mapping for any packet that matches an existing conntrack entry.
- **Performance**: Modern kube-proxy implementations use the Netlink API for conntrack management. Targeted deletions using specific filters (Protocol, Source/Destination IP/Port) are efficient and avoid the overhead of full table scans or spawning sub-processes like `conntrack -D`.
- **Final Cleanup**: Once the pod has exited and the endpoint is removed from the API, kube-proxy performs a targeted conntrack flush for that specific `{ServiceIP, ServicePort, Protocol, BackendIP, BackendPort}` tuple.

#### Test Plan

- **Unit tests** for `pkg/proxy/endpointslicecache.go`:
  - Verify that `detectStaleConntrackEntries` excludes terminating endpoints for `PreserveDuringTermination` ports.
  - Verify that `detectStaleConntrackEntries` includes them once the endpoint is deleted.
  - **Dynamic Change Test**: Verify that changing the strategy from `PreserveDuringTermination` to `Default` triggers immediate cleanup for already-terminating pods.
- **Unit tests** for proxiers:
  - Verify that `syncProxyRules` handles the transition correctly.
- **Integration tests**:
  - Service creation with `udpConnectionHandling: PreserveDuringTermination`.
  - Endpoint transition from Ready to Terminating.
  - Verification that `conntrack -L` still shows the entry during termination.
- **End-to-end tests**:
  - **Session Persistence**: A QUIC client remains connected to a terminating pod until the pod exits.
  - **Timeout Handling**: Verify that if a client stops sending packets for longer than the kernel's `nf_conntrack_udp_timeout`, the flow is naturally expired even if the pod is still in termination (expected behavior).
  - **Safety Valve Test**: Verify that a flow is forcefully flushed after the hard safety valve timeout even if the pod remains in termination state.
  - **Scalability Test**: Perform a rolling update of a Service with 5,000 pods and verify that the deferred conntrack cleanup does not cause excessive memory pressure on the node's conntrack table or significant sync delays in kube-proxy.

### Graduation Criteria

#### Alpha

- Feature implemented behind the `PreserveUDPFlowsTerminatingEndpoints` feature gate.
- Initial unit and integration tests completed.
- Support for `udpConnectionHandling` in `ServicePort` (or as a hint).

#### Beta

- Gather feedback from users.
- e2e tests enabled in CI.
- Default enablement of the feature gate.

#### GA

- Feature gate locked to true.
- Conformance tests added.

### Upgrade / Downgrade Strategy

- During upgrade, new kube-proxies will start respecting the `udpConnectionHandling`.
- During downgrade, old kube-proxies will ignore the new field/hint and revert to the old behavior of clearing all conntrack entries for terminating endpoints.

### Version Skew Strategy

- New kube-proxies will work with old/new API servers (if the field is missing, it defaults to `Default`).
- Old kube-proxies will work with new API servers (they will just ignore the new field).

## Production Readiness Review Questionnaire

### Feature Enablement and Rollback

###### How can this feature be enabled / disabled in a live cluster?

- [x] Feature gate (also fill in values in `kep.yaml`)
  - Feature gate name: PreserveUDPFlowsTerminatingEndpoints
  - Components depending on the feature gate: kube-proxy, kube-apiserver (for API changes)

###### Does enabling the feature change any default behavior?

No, the default strategy is `Default`, which matches the current behavior.

###### Can the feature be disabled once it has been enabled (i.e. can we roll back the enablement)?

Yes, by disabling the feature gate. Kube-proxy will revert to its previous behavior.

###### What happens if we reenable the feature if it was previously rolled back?

Kube-proxy will resume respecting the `udpConnectionHandling`.

###### Are there any tests for feature enablement/disablement?

Yes, unit tests will cover behavior with the feature gate both enabled and disabled.

## Implementation History

- 2026-03-31: Initial KEP draft (provisional) with API changes (Processing field).
- 2026-05-02: Revised to use Service annotation instead of API changes.
- 2026-05-02: Revised to use conntrack marks and netfilter operations.
- 2026-05-09: Revised to use a Service-level connection type hint (`udpConnectionHandling`).

## Drawbacks

- Adds a new field to the Service API (if implemented as a field).
- Increases complexity in kube-proxy's endpoint tracking logic.

## Alternatives

- **Conntrack mark approach**: Rejected per maintainer feedback (racy, inefficient, requires app awareness).
- **Service annotation approach**: Rejected per maintainer feedback (annotations are a poor API).
- **Adding a `Processing` field to EndpointConditions**: Rejected as it changes the endpoint lifecycle.
