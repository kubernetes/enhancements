# KEP-6272: LoadBalancer ipMode Router

<!-- toc -->
- [Release Signoff Checklist](#release-signoff-checklist)
- [Summary](#summary)
- [Motivation](#motivation)
  - [Goals](#goals)
  - [Non-Goals](#non-goals)
- [Proposal](#proposal)
  - [User Stories](#user-stories)
  - [Notes/Constraints/Caveats](#notesconstraintscaveats)
  - [Risks and Mitigations](#risks-and-mitigations)
- [Design Details](#design-details)
  - [API Changes](#api-changes)
  - [Semantics](#semantics)
  - [Validation](#validation)
  - [Service Proxy Implementation Guidance](#service-proxy-implementation-guidance)
    - [kube-proxy (iptables)](#kube-proxy-iptables)
    - [kube-proxy (nftables)](#kube-proxy-nftables)
    - [Non-kube-proxy implementations](#non-kube-proxy-implementations)
  - [Health Check Node Port Interaction](#health-check-node-port-interaction)
  - [Gateway API Interaction](#gateway-api-interaction)
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
  - [New externalTrafficPolicy value PreferLocal](#new-externaltrafficpolicy-value-preferlocal)
  - [New spec field externalFailoverPolicy](#new-spec-field-externalfailoverpolicy)
  - [Annotation](#annotation)
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
  - [ ] (R) [all GA Endpoints](https://github.com/kubernetes/community/pull/1806) must be hit by [Conformance Tests](https://github.com/kubernetes/community/blob/master/contributors/devel/sig-architecture/conformance-tests.md)
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

This KEP introduces a new `LoadBalancerIPMode` value, `Router`, for the
existing `status.loadBalancer.ingress[].ipMode` field on Services.

When a LoadBalancer controller sets `ipMode: Router` on an ingress IP, the
Service proxy (kube-proxy or equivalent) programs DNAT rules for that IP when
endpoints are available — same as `VIP` mode. But when no eligible endpoints
exist, the proxy does not install a DROP or REJECT rule. Instead, the packet
follows the node's routing table.

This allows BGP, ECMP, anycast, or other routing mechanisms to redirect traffic
to another node, cluster, or location without requiring coordination between the
Service proxy and the routing system.

The `ipMode` field is set by the LoadBalancer controller (not the user) in
Service status, making it the natural place to express "this IP is a routed
anycast address" rather than "this IP is a cloud load balancer VIP."

## Motivation

With `externalTrafficPolicy: Local`, kube-proxy currently installs DROP or
REJECT rules for external Service destinations when there are no locally
eligible endpoints.

This is appropriate for deployments where an external load balancer stops
sending traffic to a node before the absence of local endpoints matters.
However, in BGP/anycast deployments, endpoint state and route advertisement are
updated independently. Whenever a node has no local endpoints, kube-proxy blocks
traffic to the external VIP even when normal routing could still deliver it:

1. The last locally eligible endpoint disappears.
2. kube-proxy installs a DROP or REJECT rule.
3. Traffic still arriving at the node is blocked by kube-proxy instead of being
   handled by normal routing.

This persists for as long as the block applies (no local endpoints for the DROP,
no endpoints anywhere in the cluster for the REJECT) and is not removed by other
means. The convergence window before a routing controller withdraws the route is
one instance, but the block also applies when no controller withdraws the route
at all, or when the VIP is anycast and still serving from another cluster or
location.

The same issue exists when a Pod in the local cluster accesses the externally
advertised VIP and the cluster has no endpoints at all: kube-proxy rejects that
traffic (via the `FORWARD` hook) instead of letting it follow the external
routing path to another cluster or location. (When other cluster-local
endpoints exist, kube-proxy already load-balances such Pod traffic cluster-wide
without loss, so that case needs no change.)

The Service proxy should therefore be able to abstain from making a negative
reachability decision for an external Service address when the operator has
explicitly delegated failover to the network.

### Goals

- Add a new `LoadBalancerIPMode` value `Router` that suppresses no-endpoint
  DROP/REJECT for the LoadBalancer IP while preserving DNAT when endpoints
  exist.
- The value is set by the LoadBalancer controller in Service status, not by
  the user in Service spec.
- Preserve `healthCheckNodePort` behavior so routing or load-balancer
  integrations can continue to observe local endpoint availability.
- Preserve all existing behavior by default.

### Non-Goals

- BGP route advertisement or withdrawal.
- Creating routes or otherwise implementing failover routing.
- Changing `externalTrafficPolicy` semantics or adding new values.
- Changing `internalTrafficPolicy`.
- Changing ClusterIP or NodePort behavior.
- Defining how a LoadBalancer controller decides to set `Router` — that is an
  implementation decision for Calico, Cilium, MetalLB, or other BGP speakers.

## Proposal

A BGP speaker acting as a LoadBalancer controller sets `ipMode: Router` on
the ingress IP it assigns:

```yaml
apiVersion: v1
kind: Service
metadata:
  name: my-anycast-service
spec:
  type: LoadBalancer
  externalTrafficPolicy: Local
  ports:
  - port: 80
    targetPort: 8080
status:
  loadBalancer:
    ingress:
    - ip: "10.0.0.1"
      ipMode: Router
```

The user does not set `ipMode` — it is a status field managed by the
LoadBalancer controller. From the user's perspective, nothing changes in the
Service spec. The controller signals to kube-proxy how to handle the IP.

### User Stories

**Multi-cluster anycast failover**

As an operator advertising the same Service VIP from multiple clusters, I want
traffic reaching a cluster with no locally eligible endpoints to remain
routable so that the network can redirect it to another location.

This includes the case where the local cluster has no endpoints at all. A
Service proxy in one cluster cannot determine whether the same anycast address
is still serving traffic elsewhere.

**Pod-to-external-VIP failover**

As an application running in the cluster, I may intentionally connect to a
Service through its externally advertised VIP rather than through its ClusterIP.
When the whole cluster has no eligible endpoints, I want that packet to follow
the same external routing and failover path as traffic arriving from outside
the cluster, so it can reach another cluster or location.

The behavior of Pod-originated traffic to the external VIP depends on endpoint
state today. `Router` only changes the case where the traffic would
otherwise be lost:

- When the cluster has no endpoints at all, kube-proxy currently installs a
  filter `REJECT` for the external VIP that is reached from the `FORWARD` hook,
  so Pod-originated traffic to the VIP is rejected. `Router` suppresses
  that reject and lets the packet follow normal node routing, so it can reach
  another cluster or location.
- When the local node has no eligible endpoint but other cluster-local
  endpoints exist, kube-proxy already load-balances this traffic cluster-wide
  ("up-and-out" simulation). It is served without loss, so `Router` leaves
  it unchanged.

### Notes/Constraints/Caveats

- `Router` mode does not provide routing. It only prevents the Service proxy
  from dropping or rejecting the packet because no eligible endpoints are
  reachable. It does not change traffic that kube-proxy already serves, such as
  the Pod/host cluster-wide short-circuit to the external VIP.
- Subsequent behavior depends on the node's routing and networking
  configuration.
- The feature applies even when the local cluster has zero endpoints, because
  the same external address may still be reachable through another cluster or
  location.
- The LB controller decides when to set `Router` vs `VIP`. A BGP-based
  controller would use `Router` for BGP-advertised addresses. A cloud LB
  controller would continue to use `VIP`.

### Risks and Mitigations

| Risk | Mitigation |
|------|------------|
| `Router` set without a usable routing path | Traffic follows the routing table; if no route exists, standard IP routing behavior applies (e.g., ICMP unreachable). Document that the feature delegates to node networking. |
| LB controller sets `Router` incorrectly | The field is in status, not spec. Only controllers with status write access can set it. Incorrect use degrades gracefully. |
| Interaction with Gateway API | Gateway API implementations create a LoadBalancer Service per Gateway. The envoy-proxy pod is always an endpoint, so kube-proxy never triggers the no-endpoint path regardless of `ipMode`. See [Gateway API Interaction](#gateway-api-interaction). |

## Design Details

### API Changes

Add a new constant to the existing `LoadBalancerIPMode` type:

```go
const (
    LoadBalancerIPModeVIP    LoadBalancerIPMode = "VIP"
    LoadBalancerIPModeProxy  LoadBalancerIPMode = "Proxy"

    // LoadBalancerIPModeRouter indicates that the Service proxy should program
    // DNAT rules for this IP when endpoints are available (same as VIP), but
    // should not install DROP or REJECT rules when no eligible endpoints exist.
    // Instead, the packet follows the node's routing table.
    //
    // This mode is intended for BGP/anycast LoadBalancer controllers that
    // advertise the IP as a routed address. The controller manages route
    // advertisement and withdrawal; the proxy should not make independent
    // reachability decisions.
    //
    // +featureGate=LoadBalancerIPModeRouter
    LoadBalancerIPModeRouter LoadBalancerIPMode = "Router"
)
```

No new fields are added. The change extends an existing enum in an existing
status field.

### Semantics

The core contract is:

> When a LoadBalancer ingress IP has `ipMode: Router` and traffic destined to
> that IP would otherwise be dropped or rejected solely because no eligible
> endpoints are reachable, a conforming Service proxy MUST NOT drop or reject
> that traffic. The packet MUST be allowed to continue through normal node
> networking so it can fail over via the network.
>
> `Router` only suppresses no-endpoint DROP/REJECT behavior. It does not
> change cases where the traffic is already served, including the existing
> cluster-wide short-circuit for Pod- and host-originated traffic to the
> external VIP.

The behavior of `ipMode: Router` compared to the existing modes:

|                                            | `VIP` (today)                     | `Proxy` (today)                    | `Router` (new)                                          |
|--------------------------------------------|-----------------------------------|------------------------------------|---------------------------------------------------------|
| Set by                                     | LB controller                     | LB controller                      | LB controller, only for anycast addresses               |
| Packet arrives as                          | dst = LB IP:port                  | dst = nodeIP:nodePort or podIP     | dst = LB IP:port                                        |
| Proxy programs DNAT for the IP             | yes                               | no (IP ignored)                    | yes                                                     |
| Usable endpoints                           | DNAT                              | n/a                                | DNAT (same as VIP)                                      |
| Zero endpoints                             | REJECT                            | n/a                                | no verdict, follows node routing table                  |
| eTP:Local, no local eps, ext. eps          | DROP (external origin)            | n/a                                | open (falls back to the routing table)                  |
| eTP:Local, no local eps, Pod/host origin   | DNAT cluster-wide                 | n/a                                | DNAT cluster-wide (unchanged)                           |
| healthCheckNodePort                        | unhealthy without local eps       | n/a                                | same                                                    |
| BGP / LB controller                        | eTP:Local: nodes with local eps; eTP:Cluster: all nodes | n/a | advertise only while the node would DNAT; never with zero endpoints |

The two key changes from `VIP`:

1. **No-local-endpoints DROP suppressed.** External traffic that would be
   DROPped (because `eTP: Local` has no local endpoints) instead follows the
   routing table.
2. **Zero-endpoints REJECT suppressed.** Traffic that would be REJECTed
   (because no endpoints exist anywhere) instead follows the routing table.

Traffic that kube-proxy already serves — the Pod/host cluster-wide
short-circuit when other endpoints exist — is unchanged.

`Router` never introduces a new forwarding path; it only removes a
no-endpoint DROP/REJECT so normal node networking can take over.

`loadBalancerSourceRanges` and other independent Service policy remain in
force.

NodePort handling is unchanged by this KEP.

### Validation

- `Router` is a valid value for `status.loadBalancer.ingress[].ipMode`.
- The value is set by the LB controller in status, not by users in spec.
- `ipMode: Router` applies **only** to the LoadBalancer ingress IP it is set
  on. ExternalIPs, NodePorts, and ClusterIPs on the same Service retain their
  existing DROP/REJECT behavior regardless of `ipMode`.
- When the `LoadBalancerIPModeRouter` feature gate is enabled,
  `supportedLoadBalancerIPMode` in API validation must include `Router`
  alongside `VIP` and `Proxy`. When the gate is disabled, `Router` is rejected
  on create and update, but existing `Router` values already persisted in etcd
  are preserved and remain visible to kube-proxy (standard drop-on-disable
  pattern for feature-gated API values).

### Service Proxy Implementation Guidance

The KEP specifies observable behavior rather than a required dataplane
mechanism.

kube-proxy's existing `IsVIPMode()` helper gates whether a LoadBalancer IP
is added to kube-proxy's DNAT rules. For `Router`, the IP must be included
(same as `VIP`) but the no-endpoint filter rules must be skipped. The
implementation must ensure both conditions are met.

#### kube-proxy (iptables)

When processing a LoadBalancer ingress IP with `ipMode: Router`:

- Add the IP to `loadBalancerVIPs` (same as `VIP`), so DNAT rules are
  programmed when endpoints exist.
- When `externalTrafficPolicy: Local` and no local endpoints exist: do not
  write a DROP rule in `KUBE-EXTERNAL-SERVICES` for this IP.
- When no endpoints exist anywhere: do not write a REJECT rule in
  `KUBE-EXTERNAL-SERVICES` for this IP.
- Preserve all NAT chains including the Pod/host cluster-wide short-circuit.
- Preserve `loadBalancerSourceRanges` enforcement when endpoints exist.

#### kube-proxy (nftables)

Same observable behavior:

- Include the IP in Service translation maps.
- Do not add the IP to the no-endpoint verdict map when `Router` applies.
- Preserve all Service translation paths that already serve traffic.

#### Non-kube-proxy implementations

Cilium, Calico (BPF), Antrea, OVN-Kubernetes, and other implementations
must satisfy the same observable semantics: program DNAT for `Router` IPs
when endpoints exist, skip no-endpoint DROP/REJECT when they don't.

For BPF implementations, this means: when `ipMode: Router` and no local
endpoints exist, pass the packet to the host networking stack instead of
dropping it or sending an ICMP reject.

### Health Check Node Port Interaction

This KEP does not change `spec.healthCheckNodePort`.

When a LoadBalancer Service using `externalTrafficPolicy: Local` has no locally
eligible endpoints, the health check continues to report the node as unhealthy
according to existing behavior.

A BGP speaker uses this signal (or EndpointSlices) to withdraw the route.
`Router` mode removes the requirement that kube-proxy and that route withdrawal
occur in a particular order.

### Gateway API Interaction

Gateway API implementations (Calico Envoy Gateway, Cilium, etc.) create a
`type: LoadBalancer` Service per Gateway. The envoy-proxy pod is always
running, so the LB Service always has endpoints. kube-proxy never triggers
the no-endpoint DROP/REJECT path for these Services.

This means `ipMode: Router` has no effect on Gateway API Services — the
no-endpoint code path is never reached. This is expected: Gateway API inserts
an always-on proxy between the VIP and the backends. BGP speakers see the
proxy as a healthy endpoint and never withdraw the route, so packets are
consumed by the proxy (returning 503 when backends are gone) rather than
following routing.

For anycast failover, a plain `LoadBalancer` Service with
`externalTrafficPolicy: Local` and `ipMode: Router` is the appropriate
mechanism. See the [reproducer](https://gist.github.com/defo89/99fba7a7b575ccb3175df2164d4a82e1)
for a demonstration.

### Test Plan

[x] I/we understand the owners of the involved components may require updates to
existing tests to make this code solid enough prior to committing the changes
necessary to implement this enhancement.

#### Prerequisite testing updates

Existing tests for `ipMode: VIP` and `ipMode: Proxy` must continue to pass.

#### Unit tests

For both iptables and nftables:

- `ipMode: Router` with local endpoints behaves identically to `VIP`;
- `ipMode: Router` with no local endpoints and other cluster-local endpoints:
  no DROP for externally-originated traffic; Pod/host short-circuit unchanged;
- `ipMode: Router` with zero endpoints: no REJECT; packet follows routing;
- `ipMode: VIP` behavior unchanged;
- `ipMode: Proxy` behavior unchanged;
- `loadBalancerSourceRanges` enforced with `Router`;
- NodePort behavior unchanged;
- cover IPv4 and IPv6;
- cover TCP and UDP.

API tests:

- `Router` accepted as a valid `ipMode` value;
- feature-gate handling;
- update/downgrade compatibility.

#### Integration tests

- `Router` accepted as a valid `ipMode` value in Service status;
- `Router` rejected when feature gate is disabled;
- LB controller can write `ipMode: Router` via status subresource;
- kube-proxy reads and acts on the value;
- feature-gate behavior across API server and kube-proxy.

#### e2e tests

Actual BGP/anycast failover may not be portable across standard Kubernetes e2e
environments.

Alpha testing should therefore focus on deterministic observable behavior:

- no endpoint-availability DROP/REJECT for the LB IP when
  `ipMode: Router` and no endpoints;
- existing Pod/host cluster-wide short-circuit preserved;
- unchanged `VIP` and `Proxy` behavior;
- unchanged NodePort behavior.

A routing-aware e2e test may be added if a portable topology can be built in
Kubernetes CI.

### Graduation Criteria

#### Alpha

- Feature gate `LoadBalancerIPModeRouter` disabled by default.
- New `Router` value accepted behind the feature gate.
- kube-proxy iptables and nftables implementations.
- Unit coverage.
- Documented semantics and limitations.

#### Beta

- Feature gate enabled by default.
- At least one BGP speaker (Calico, Cilium, or MetalLB) sets `Router` in
  its LB controller.
- No unresolved correctness or security issues.
- User documentation.
- Upgrade/downgrade behavior validated.

#### GA

- At least two releases of beta experience.
- Multiple BGP speakers setting `Router`.
- No unresolved API issues.

### Upgrade / Downgrade Strategy

Existing Services are unaffected because `Router` is only set by LB controllers
that explicitly opt in.

On upgrade, Services without `ipMode: Router` retain existing behavior.

On downgrade, the API server is downgraded before node components (standard
Kubernetes ordering). The old API server does not include `Router` in its
validation set, so it rejects status updates that try to set `Router`. The LB
controller falls back to `VIP` and kube-proxy never sees the value.

### Version Skew Strategy

Kubernetes supports kube-proxy being up to one minor version behind the API
server.

- **New API server, old kube-proxy:** old kube-proxy does not recognize
  `Router` and skips the IP entirely — no DNAT and no DROP/REJECT rules are
  installed. This is the same skew behavior that `Proxy` mode had when it
  was introduced. During rolling upgrades, nodes with old kube-proxy do not
  serve the LB IP locally. Full `Router` behavior is available once all
  nodes are updated.
- **New kube-proxy, old API server:** the old API server rejects `Router` in
  validation. The value is never written to status. kube-proxy sees `VIP` or
  `Proxy` and behaves accordingly.
- **Mixed kube-proxy versions:** nodes with new kube-proxy provide `Router`
  behavior; nodes with old kube-proxy skip the IP. This is expected during
  rolling upgrades and resolves once all nodes are updated.

## Production Readiness Review Questionnaire

### Feature Enablement and Rollback

###### How can this feature be enabled / disabled in a live cluster?

- [x] Feature gate (also fill in values in `kep.yaml`)
  - Feature gate name: `LoadBalancerIPModeRouter`
  - Components depending on the feature gate: kube-apiserver, kube-proxy

No downtime beyond the normal component restart. A LoadBalancer controller opts
in by setting `ipMode: Router` in Service status.

###### Does enabling the feature change any default behavior?

No. The feature only takes effect when a LB controller explicitly sets
`ipMode: Router`.

###### Can the feature be disabled once it has been enabled (i.e. can we roll back the enablement)?

Yes. Disabling the gate on the API server prevents new `Router` values from
being written to status. Existing `Router` values already in etcd remain
visible to kube-proxy, which skips those IPs (no DNAT). The feature is fully
rolled back once the LB controller updates the affected Services.

###### What happens if we reenable the feature if it was previously rolled back?

kube-proxy begins programming DNAT for `Router` IPs again on the next sync.

###### Are there any tests for feature enablement/disablement?

Planned unit tests cover gate on/off behavior for API validation and
kube-proxy rule generation with `Router` IPs.

### Rollout, Upgrade and Rollback Planning

###### How can a rollout or rollback fail? Can it impact already running workloads?

Non-opted-in workloads are unaffected. Main risk: `Router` set without a
routing path, so packets are not delivered. This is no worse than DROP/REJECT.
During rolling upgrades, nodes with old kube-proxy skip `Router` IPs (no DNAT);
this resolves once all nodes are updated.

###### What specific metrics should inform a rollback?

Rising connection failures to LB VIPs. Correlate with Services that have
`ipMode: Router` and verify routing table and iptables/nftables state on
affected nodes.

###### Were upgrade and rollback tested? Was the upgrade->downgrade->upgrade path tested?

No. Not yet implemented. Will be tested before beta.

###### Is the rollout accompanied by any deprecations and/or removals of features, APIs, fields of API types, flags, etc.?

No.

### Monitoring Requirements

###### How can an operator determine if the feature is in use by workloads?

Inspect `status.loadBalancer.ingress[].ipMode` for `Router` values.

###### How can someone using this feature know that it is working for their instance?

- [x] Other (treat as last resort)
  - Details: verify kube-proxy does not install DROP/REJECT for the LB IP
    when no endpoints exist. Verify via `iptables-save` / `nft list ruleset`.

###### What are the reasonable SLOs (Service Level Objectives) for the enhancement?

The feature removes kube-proxy as a packet-loss source; it provides no delivery
guarantee itself. Objective: no measurable regression in
`sync_proxy_rules_duration_seconds`.

###### What are the SLIs (Service Level Indicators) an operator can use to determine the health of the service?

- [x] Metrics
  - Metric name: `sync_proxy_rules_duration_seconds`,
    `sync_proxy_rules_last_timestamp_seconds` (existing kube-proxy metrics)
  - Components exposing the metric: kube-proxy
- [x] Other (treat as last resort)
  - Details: application success rate/latency to the external VIP and the
    generated dataplane state above.

###### Are there any missing metrics that would be useful to have to improve observability of this feature?

A counter of LB IPs in `Router` mode could help; deferred past alpha.

### Dependencies

###### Does this feature depend on any specific services running in the cluster?

No Kubernetes component dependency. Successful failover requires a BGP speaker
or equivalent routing controller.

### Scalability

###### Will enabling / using this feature result in any new API calls?

No. No new listers, watches, or reconcile loops are introduced.

###### Will enabling / using this feature result in introducing new API types?

No. Extends an existing enum value.

###### Will enabling / using this feature result in any new calls to the cloud provider?

No.

###### Will enabling / using this feature result in increasing size or count of the existing API objects?

No. Uses an existing field with a new value.

###### Will enabling / using this feature result in increasing time taken by any operations covered by existing SLIs/SLOs?

No. Only a small amount of extra branching in existing rule generation.

###### Will enabling / using this feature result in non-negligible increase of resource usage (CPU, RAM, disk, IO, ...) in any components?

No. Reuses existing endpoint state; may remove rules for opted-in Services.

###### Can enabling / using this feature result in resource exhaustion of some node resources (PIDs, sockets, inodes, etc.)?

No. No additional node resources are consumed.

### Troubleshooting

###### How does this feature react if the API server and/or etcd is unavailable?

kube-proxy enforces the last-synced rules, as today. No new dependency on API
server/etcd availability.

###### What are other known failure modes?

`Router` set without a usable routing path: traffic follows the routing table
but may not reach a destination. No worse than DROP/REJECT. Detect via VIP
connection failures.

###### What steps should be taken if SLOs are not being met to determine the problem?

- Verify `ipMode: Router` is set on the Service status.
- Verify kube-proxy feature gate is enabled.
- Verify no DROP/REJECT in iptables/nftables for the LB IP.
- Verify the node routing table has a path for the LB IP.
- Verify BGP route advertisement and convergence.

## Implementation History

- 2026-08-12: Initial KEP draft proposing `spec.externalFailoverPolicy` field
  based on [kubernetes/kubernetes#139300](https://github.com/kubernetes/kubernetes/issues/139300).
- 2026-10-01: Reworked to use `ipMode: Router` based on sig-network feedback.

## Drawbacks

- Adds a third `ipMode` value. However, `ipMode` is already designed to be
  extensible and the existing two values do not cover the routed-IP case.

## Alternatives

### New externalTrafficPolicy value PreferLocal

Add `externalTrafficPolicy: PreferLocal` — same as `Local` but without
the no-endpoint DROP/REJECT.

Rejected because:
- Mixes two concerns: endpoint selection policy (Local vs Cluster) and VIP
  reachability management (drop vs route).
- `ipMode` is set by the LB controller per-IP, which is the right scope.
  The user sets `eTP: Local` for source IP preservation and the LB controller
  decides how the IP is managed.
- Adding a new eTP value has larger API surface and more complex interaction
  with existing eTP behavior.

### New spec field externalFailoverPolicy

Add `spec.externalFailoverPolicy: Passthrough` — the original proposal in
this KEP.

Rejected because:
- Adds a new spec field that the user must set, but the decision is really
  about how the LB IP is managed (a controller concern, not a user concern).
- `ipMode` already exists as the mechanism for LB controllers to communicate
  IP handling behavior to kube-proxy. No new API fields needed.

### Annotation

`service.kubernetes.io/no-reject-on-no-endpoints: "true"` — the original
proposal in [kubernetes/kubernetes#139300](https://github.com/kubernetes/kubernetes/issues/139300).

Rejected because:
- No validation, no feature lifecycle, not portable across proxy implementations.
- `ipMode` provides a typed, validated, discoverable mechanism.
