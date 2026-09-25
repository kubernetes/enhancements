# KEP-6272: Service External Failover Policy

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

This KEP introduces a new Service API field, `spec.externalFailoverPolicy`, for
Services using `externalTrafficPolicy: Local`.

When `externalFailoverPolicy: Passthrough` is set and traffic to the Service's
LoadBalancer or ExternalIP addresses would otherwise be dropped or rejected
solely because no eligible endpoints are reachable, the Service proxy suppresses
that drop/reject. Instead, normal node networking determines subsequent packet
handling.

This allows BGP, ECMP, anycast, or other routing mechanisms to redirect traffic
to another node, cluster, or location without requiring coordination between the
Service proxy and the routing system.

When the cluster has no endpoints at all, this also applies to Pod traffic
addressed to the external VIP: it is no longer rejected and can fail over through
the network. Traffic the proxy already serves (the Pod/host cluster-wide
short-circuit) is unchanged.

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

- Add an opt-in Service API field that suppresses no-endpoint DROP/REJECT
  behavior for LoadBalancer and ExternalIP destinations when
  `externalTrafficPolicy: Local` is used.
- Suppress the external DROP for externally-originated traffic when the node
  has no local endpoint, and the no-endpoint REJECT (external and Pod origin)
  when the cluster has no endpoints at all, while preserving traffic the proxy
  already serves (the Pod/host cluster-wide short-circuit).
- Preserve `healthCheckNodePort` behavior so routing or load-balancer
  integrations can continue to observe local endpoint availability.
- Preserve all existing behavior by default.

### Non-Goals

- BGP route advertisement or withdrawal.
- Creating routes or otherwise implementing failover routing.
- Changing `internalTrafficPolicy`.
- Changing ClusterIP behavior.
- Changing NodePort no-endpoint behavior as part of the initial implementation.
- Defining a cluster-wide default; the feature is opt-in per Service.

## Proposal

Add a new optional field to the Service API:

```yaml
apiVersion: v1
kind: Service
metadata:
  name: my-anycast-service
spec:
  type: LoadBalancer
  externalTrafficPolicy: Local
  externalFailoverPolicy: Passthrough
  ports:
  - port: 80
    targetPort: 8080
```

When the field is unset or set to its default value, existing behavior is
unchanged.

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
state today. `Passthrough` only changes the case where the traffic would
otherwise be lost:

- When the cluster has no endpoints at all, kube-proxy currently installs a
  filter `REJECT` for the external VIP that is reached from the `FORWARD` hook,
  so Pod-originated traffic to the VIP is rejected. `Passthrough` suppresses
  that reject and lets the packet follow normal node routing, so it can reach
  another cluster or location.
- When the local node has no eligible endpoint but other cluster-local
  endpoints exist, kube-proxy already load-balances this traffic cluster-wide
  ("up-and-out" simulation). It is served without loss, so `Passthrough` leaves
  it unchanged.

### Notes/Constraints/Caveats

- `Passthrough` does not provide routing. It only prevents the Service proxy
  from dropping or rejecting the packet because no eligible endpoints are
  reachable. It does not change traffic that kube-proxy already serves, such as
  the Pod/host cluster-wide short-circuit to the external VIP.
- Subsequent behavior depends on the node's routing and networking
  configuration.
- The feature applies even when the local cluster has zero endpoints, because
  the same external address may still be reachable through another cluster or
  location.
- NodePort is intentionally excluded from the initial behavior because
  `nodeIP:nodePort` normally targets a locally owned address and does not
  represent the external VIP routing use case.

### Risks and Mitigations

| Risk | Mitigation |
|------|------------|
| Passthrough is enabled without a usable network path | Document that the feature delegates subsequent handling to normal node networking and does not itself provide failover. |
| Operator expects eTP `Cluster` semantics | Validation only permits the field with `externalTrafficPolicy: Local`. |
| Routing convergence is slow | This feature removes kube-proxy as an additional source of packet loss but does not change routing convergence itself. |

## Design Details

### API Changes

```go
// ServiceExternalFailoverPolicy describes how a Service proxy handles traffic
// destined to a Service's external addresses when externalTrafficPolicy is
// Local and no locally eligible endpoints exist on the node.
type ServiceExternalFailoverPolicy string

const (
    // ServiceExternalFailoverPolicyNone preserves existing behavior.
    ServiceExternalFailoverPolicyNone ServiceExternalFailoverPolicy = "None"

    // ServiceExternalFailoverPolicyPassthrough prevents the Service proxy from
    // dropping or rejecting traffic solely because no eligible endpoints are
    // reachable. Normal node networking then determines subsequent packet
    // handling.
    ServiceExternalFailoverPolicyPassthrough ServiceExternalFailoverPolicy = "Passthrough"
)

// ServiceSpec describes the attributes that a user creates on a Service.
type ServiceSpec struct {
    ...

    // ExternalFailoverPolicy controls handling of traffic destined to the
    // Service's LoadBalancer or ExternalIP addresses when
    // externalTrafficPolicy is "Local" and no locally eligible endpoints exist.
    //
    // "None" (default): preserve existing Service proxy behavior.
    //
    // "Passthrough": do not drop or reject traffic solely because no eligible
    // endpoints are reachable. When the cluster has no endpoints, normal node
    // networking determines subsequent handling instead of a reject. Traffic
    // that the proxy already serves (such as the Pod/host cluster-wide
    // short-circuit to the external address) is unchanged.
    //
    // This field is only valid when externalTrafficPolicy is "Local".
    //
    // +featureGate=ServiceExternalFailoverPolicy
    // +optional
    ExternalFailoverPolicy *ServiceExternalFailoverPolicy `json:"externalFailoverPolicy,omitempty" protobuf:"bytes,24,opt,name=externalFailoverPolicy,casttype=ServiceExternalFailoverPolicy"`
}
```

The exact zero/default representation and feature-gated field handling should
follow the conventions used by current feature-gated Service API fields.

### Semantics

The core contract is:

> When `externalTrafficPolicy` is `Local`,
> `externalFailoverPolicy` is `Passthrough`, and traffic destined to the
> Service's LoadBalancer or ExternalIP addresses would otherwise be dropped or
> rejected solely because no eligible endpoints are reachable, a conforming
> Service proxy MUST NOT drop or reject that traffic. The packet MUST be allowed
> to continue through normal node networking so it can fail over via the
> network.
>
> `Passthrough` only suppresses no-endpoint DROP/REJECT behavior. It does not
> change cases where the traffic is already served, including the existing
> cluster-wide short-circuit for Pod- and host-originated traffic to the
> external VIP.

Today, kube-proxy handles external-VIP traffic differently depending on both
the endpoint state and the packet origin. The following describes the current
(`None`) behavior for kube-proxy iptables/nftables, which `Passthrough` must
reason about explicitly.

Externally-originated traffic (from outside the cluster) for an external
destination under eTP `Local`:

| Local endpoints | Other cluster-local endpoints | `None` (today) | `Passthrough` |
|---|---|---|---|
| yes | any | DNAT to local endpoint | DNAT to local endpoint (unchanged) |
| no | yes | DROP (filter, no local endpoints) | No endpoint-based DROP; normal node networking |
| no | no | REJECT (filter, no endpoints) | No endpoint-based REJECT; normal node networking |

Pod-originated traffic (from a Pod on the node) to the external VIP under eTP
`Local`:

| Local endpoints | Other cluster-local endpoints | `None` (today) | `Passthrough` |
|---|---|---|---|
| yes | any | DNAT to local endpoint | DNAT to local endpoint (unchanged) |
| no | yes | DNAT cluster-wide ("up-and-out" short-circuit) | DNAT cluster-wide (unchanged; already served) |
| no | no | REJECT (via `KUBE-EXTERNAL-SERVICES` from the `FORWARD` hook) | No endpoint-based REJECT; normal node networking |

Host-originated traffic (from the node itself) reaches the EXT chain via the
nat `OUTPUT` hook and is DNAT'd like the Pod short-circuit when endpoints exist
(unchanged). The no-endpoint filter (`KUBE-EXTERNAL-SERVICES`) is only hooked
into `INPUT`/`FORWARD`, so host traffic is not endpoint-rejected today and is
unaffected by `Passthrough`.

The two cases that require the most care:

- **no local / other cluster endpoints exist, Pod or host origin.** kube-proxy
  already load-balances this traffic across all cluster endpoints ("up-and-out"
  simulation). It is served without loss, so `Passthrough` leaves it unchanged.
- **zero endpoints anywhere, external or Pod origin.** Under `None`, the
  external VIP is filter-rejected rather than routed. Under `Passthrough`, no
  endpoint-based reject is installed and the packet follows normal node routing,
  allowing
  anycast failover to another cluster.

`Passthrough` never introduces a new forwarding path; it only removes a
no-endpoint DROP/REJECT so normal node networking can take over. It does not
affect traffic kube-proxy already serves, including the Pod/host short-circuit.

`loadBalancerSourceRanges` and other independent Service policy remain in
force.

NodePort handling is unchanged by this KEP.

### Validation

- `externalFailoverPolicy` may only be set when
  `externalTrafficPolicy: Local`.
- Unsupported enum values are rejected.
- The feature affects LoadBalancer and ExternalIP destinations only.
- NodePort behavior is unchanged.
- Feature-gate enablement, disabled-field handling, update preservation, and
  downgrade behavior must follow Kubernetes compatibility rules for
  feature-gated API fields.

### Service Proxy Implementation Guidance

The KEP specifies observable behavior rather than a required dataplane
mechanism.

A conforming implementation must distinguish:

- endpoint availability (any endpoints vs. no endpoints anywhere);
- endpoints usable under the external traffic policy (local endpoints);
- endpoint-availability DROP/REJECT behavior for the external Service address.

With `Passthrough`, the no-endpoint DROP (external origin, no local endpoints)
and the no-endpoint REJECT (no endpoints anywhere) for the external Service
address must not be installed. Traffic that the proxy already serves, including
the Pod/host cluster-wide short-circuit, is left unchanged.

#### kube-proxy (iptables)

The existing kube-proxy implementation derives:

- whether endpoints exist anywhere (`hasEndpoints`);
- whether endpoints are usable under the external traffic policy
  (`hasExternalEndpoints`);
- a DROP or REJECT target for `KUBE-EXTERNAL-SERVICES`.

When `Passthrough` applies, the implementation should:

- suppress the external no-endpoint filter rule in `KUBE-EXTERNAL-SERVICES` for
  LoadBalancer/ExternalIP destinations. This is the DROP that blocks
  externally-originated traffic when `!hasExternalEndpoints`, and the REJECT
  that blocks all traffic (including Pod-originated, via the `FORWARD` hook)
  when `!hasEndpoints`;
- preserve all endpoint calculations and all NAT chains, including the Pod-
  and host-originated "up-and-out" short-circuit rules in the service's
  external (`EXT`) chain. Those rules already serve traffic and are not part of
  the no-endpoint block.

Conceptually the suppression keys off the same conditions kube-proxy already
computes:

```go
// External-origin DROP is suppressed when there are no local endpoints.
suppressExternalDrop :=
    passthrough && !hasExternalEndpoints

// No-endpoint REJECT is suppressed when there are no endpoints anywhere.
suppressNoEndpointReject :=
    passthrough && !hasEndpoints
```

where

```go
passthrough :=
    svcInfo.ExternalPolicyLocal() &&
    svcInfo.ExternalFailoverPolicy() ==
        v1.ServiceExternalFailoverPolicyPassthrough
```

When suppression applies, do not write the corresponding filter rule for the
external Service address and leave all other Service rules, including the
short-circuit NAT rules, unchanged.

#### kube-proxy (nftables)

The nftables implementation should provide the same observable behavior:

- do not add the external Service address to the no-endpoint verdict map when
  `Passthrough` applies (covering both the no-local-endpoints DROP and the
  no-endpoints REJECT cases);
- preserve all Service translation paths that already serve traffic, including
  the Pod/host short-circuit to the cluster policy chain;
- preserve unrelated Service policy.

The implementation should continue using existing endpoint availability state
rather than changing the meaning of `hasEndpoints` or
`hasExternalEndpoints`.

#### Non-kube-proxy implementations

Other Service proxy implementations MAY implement this differently but MUST
satisfy the same observable semantics.

The KEP intentionally does not prescribe an implementation mechanism for eBPF,
IPVS, or other dataplanes.

### Health Check Node Port Interaction

This KEP does not change `spec.healthCheckNodePort`.

When a LoadBalancer Service using `externalTrafficPolicy: Local` has no locally
eligible endpoints, the health check continues to report the node as unhealthy
according to existing behavior.

A routing controller may use this signal, EndpointSlices, or another mechanism
to withdraw a route. `Passthrough` removes the requirement that kube-proxy and
that routing update occur in a particular order.

### Test Plan

[x] I/we understand the owners of the involved components may require updates to
existing tests to make this code solid enough prior to committing the changes
necessary to implement this enhancement.

#### Prerequisite testing updates

Existing tests for `externalTrafficPolicy: Local` must continue to pass
unchanged when the feature gate is disabled or
`externalFailoverPolicy` is unset/defaulted.

#### Unit tests

For both iptables and nftables:

- default behavior remains unchanged;
- `Passthrough` with local endpoints behaves normally;
- `Passthrough` with no local endpoints and other cluster-local endpoints:
  externally-originated traffic is not DROPped and follows normal node
  networking; Pod- and host-originated traffic still uses the existing
  cluster-wide short-circuit (unchanged);
- `Passthrough` with zero endpoints does not install the no-endpoint REJECT for
  the external VIP, for both externally- and Pod-originated traffic;
- Pod-originated and externally originated traffic behave per the semantics
  tables: identical when the cluster has no endpoints (no REJECT), and
  divergent when other cluster endpoints exist (external DROP suppressed; Pod
  short-circuit preserved);
- `loadBalancerSourceRanges` remains enforced;
- NodePort behavior remains unchanged;
- cover IPv4 and IPv6;
- cover TCP and UDP.

API tests:

- validate accepted and rejected field combinations;
- verify feature-gate handling;
- verify update/downgrade compatibility behavior.

#### Integration tests

- create and read a Service using the new field;
- verify validation with `externalTrafficPolicy`;
- verify feature-gate behavior;
- verify updates preserve existing stored values according to feature-gated API
  compatibility requirements.

#### e2e tests

Actual BGP/anycast failover may not be portable across standard Kubernetes e2e
environments.

Alpha testing should therefore focus on deterministic observable behavior:

- no endpoint-availability DROP/REJECT for the external VIP when
  `Passthrough` applies;
- existing Pod/host cluster-wide short-circuit preserved;
- unchanged default behavior;
- unchanged NodePort behavior.

A routing-aware e2e test may be added if a portable topology can be built in
Kubernetes CI.

### Graduation Criteria

#### Alpha

- Feature gate disabled by default.
- Service API field available behind the feature gate.
- kube-proxy iptables and nftables implementations.
- API validation.
- Unit coverage for external and Pod-originated traffic.
- IPv4/IPv6 and TCP/UDP coverage.
- Documented semantics and limitations.

#### Beta

- Feature gate enabled by default.
- Implementation complete for supported kube-proxy Linux backends.
- No unresolved correctness or security issues.
- User documentation complete.
- Operational experience from BGP/anycast deployments.
- Upgrade/downgrade behavior validated.

#### GA

- At least two releases of beta experience.
- Evidence of production use.
- No unresolved API-semantic issues.
- Feedback-derived issues resolved.

### Upgrade / Downgrade Strategy

Existing Services are unaffected because the feature is opt-in.

On upgrade, Services without `externalFailoverPolicy` retain existing
DROP/REJECT behavior.

During downgrade or version skew, components that do not understand or act on
the field fall back to existing behavior. Stored-field preservation must follow
normal Kubernetes compatibility requirements for feature-gated API fields.

### Version Skew Strategy

- **New API server, old kube-proxy:** the old proxy ignores the field and retains
  existing behavior.
- **New kube-proxy, API server without the field:** kube-proxy sees the default
  value and retains existing behavior.
- **Mixed kube-proxy versions:** only nodes running an implementation that
  understands and enables the feature provide Passthrough behavior.

Mixed versions therefore degrade toward the existing DROP/REJECT behavior
rather than introducing a new forwarding path.

## Production Readiness Review Questionnaire

### Feature Enablement and Rollback

###### How can this feature be enabled / disabled in a live cluster?

- [x] Feature gate (also fill in values in `kep.yaml`)
  - Feature gate name: `ServiceExternalFailoverPolicy`
  - Components depending on the feature gate: kube-apiserver, kube-proxy

No downtime beyond the normal component restart. A Service also opts in via
`spec.externalFailoverPolicy: Passthrough`.

###### Does enabling the feature change any default behavior?

No. Opt-in per Service; unset/`None` preserves existing DROP/REJECT behavior.

###### Can the feature be disabled once it has been enabled (i.e. can we roll back the enablement)?

Yes (`disable-supported: true`). Disabling the gate or reverting a Service to
`None` restores DROP/REJECT on the next kube-proxy sync. Non-opted-in workloads
are unaffected; opted-in workloads lose network-failover behavior.

###### What happens if we reenable the feature if it was previously rolled back?

Behavior is restored on the next sync. State is derived from the current Service
and gate; nothing is persisted to reconcile.

###### Are there any tests for feature enablement/disablement?

Yes. Planned unit tests cover the gate switch for the new field (drop-on-disabled,
preserve-on-update) and kube-proxy rule generation with the gate on/off.

### Rollout, Upgrade and Rollback Planning

###### How can a rollout or rollback fail? Can it impact already running workloads?

Non-opted-in workloads cannot break. Main risk: enabling `Passthrough` where the
node has no network path for the VIP, so traffic may be black-holed instead of
fast-failing. During skew, only enabled kube-proxy nodes provide `Passthrough`;
others keep DROP/REJECT, degrading toward existing behavior. Rollback restores
prior behavior on the next sync.

###### What specific metrics should inform a rollback?

Rising connection failures/timeouts to the affected external VIPs after enabling
`Passthrough`, indicating the network has no usable failover path. kube-proxy
exposes no metric specific to this feature.

###### Were upgrade and rollback tested? Was the upgrade->downgrade->upgrade path tested?

No. Not yet implemented. Will be tested before beta.

###### Is the rollout accompanied by any deprecations and/or removals of features, APIs, fields of API types, flags, etc.?

No.

### Monitoring Requirements

###### How can an operator determine if the feature is in use by workloads?

Inspect Service objects for `spec.externalFailoverPolicy: Passthrough`.

###### How can someone using this feature know that it is working for their instance?

- [x] Other (treat as last resort)
  - Details: kube-proxy installs no endpoint-availability DROP/REJECT for the
    external VIP when no eligible endpoints exist. Verify via the generated
    iptables/nftables state (VIP absent from the no-endpoint reject/drop set).
    End-to-end success also depends on the external routing/anycast path.

###### What are the reasonable SLOs (Service Level Objectives) for the enhancement?

The feature removes kube-proxy as a packet-loss source; it provides no delivery
guarantee itself. Objective: no measurable regression in kube-proxy
`sync_proxy_rules_duration`.

###### What are the SLIs (Service Level Indicators) an operator can use to determine the health of the service?

- [x] Metrics
  - Metric name: `sync_proxy_rules_duration_seconds`,
    `sync_proxy_rules_last_timestamp_seconds` (existing kube-proxy metrics)
  - Components exposing the metric: kube-proxy
- [x] Other (treat as last resort)
  - Details: application success rate/latency to the external VIP and the
    generated dataplane state above.

###### Are there any missing metrics that would be useful to have to improve observability of this feature?

A counter of external VIPs whose no-endpoint DROP/REJECT was suppressed under
`Passthrough` could help; deferred past alpha since the config and dataplane
effect are already inspectable.

### Dependencies

###### Does this feature depend on any specific services running in the cluster?

No Kubernetes component dependency. Successful failover requires an
operator-provided external routing/anycast fabric (e.g. BGP/ECMP + a routing
controller); if it is down, packets allowed by `Passthrough` are not delivered
(no worse than DROP/REJECT, but fast-fail is lost).

### Scalability

###### Will enabling / using this feature result in any new API calls?

No. No new listers, watches, or reconcile loops are introduced.

###### Will enabling / using this feature result in introducing new API types?

No. It adds one optional enum field (`ServiceExternalFailoverPolicy`) to the
existing `ServiceSpec`.

###### Will enabling / using this feature result in any new calls to the cloud provider?

No.

###### Will enabling / using this feature result in increasing size or count of the existing API objects?

Marginally: opted-in Services gain one small string field. No new objects.

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

`Passthrough` enabled without a usable network path for the external VIP: traffic
is black-holed instead of fast-failing. Detect via rising VIP connection
failures; mitigate by reverting the Service to `None` or fixing routing (kube-proxy
restores DROP/REJECT on the next sync).

###### What steps should be taken if SLOs are not being met to determine the problem?

If Passthrough is configured but traffic does not reach another location:

- verify that the Service has no locally eligible endpoints;
- verify that the feature gate is enabled on the relevant components;
- verify that kube-proxy has not installed an endpoint-availability DROP/REJECT
  for the external VIP;
- if the intent is to fail over from a cluster that still has endpoints
  elsewhere, note that Pod/host-originated traffic is still served cluster-wide
  by design; only externally-originated traffic and the zero-endpoint case are
  affected by `Passthrough`;
- verify that the node routing table has a usable path for the external VIP;
- verify upstream route advertisement and convergence.

## Implementation History

- 2026-08-12: Initial KEP draft based on
  [kubernetes/kubernetes#139300](https://github.com/kubernetes/kubernetes/issues/139300).

## Drawbacks

- Adds another field to the Service API.
- Successful failover depends on external routing behavior that Kubernetes does
  not control.

## Alternatives

1. **Annotation instead of a Service API field**

   The original issue proposed
   `service.kubernetes.io/no-reject-on-no-endpoints: "true"`.

   An annotation has a smaller API footprint, but the requested behavior is not
   inherently kube-proxy-specific. A typed field provides validation,
   discoverability, feature lifecycle, and portable semantics for Service proxy
   implementations.
