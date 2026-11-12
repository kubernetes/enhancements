# KEP-6246: Configurable kube-proxy masquerading exclusions

<!-- toc -->
- [Release Signoff Checklist](#release-signoff-checklist)
- [Summary](#summary)
- [Motivation](#motivation)
  - [Evidence and reproduction](#evidence-and-reproduction)
  - [Goals](#goals)
  - [Non-Goals](#non-goals)
- [Proposal](#proposal)
- [Design Details](#design-details)
  - [Configuration and packet scope](#configuration-and-packet-scope)
  - [Compatibility and migration](#compatibility-and-migration)
  - [Test Plan](#test-plan)
  - [Risks and mitigations](#risks-and-mitigations)
  - [Graduation Criteria](#graduation-criteria)
    - [Alpha](#alpha)
    - [Beta](#beta)
    - [GA](#ga)
- [Production Readiness Review Questionnaire](#production-readiness-review-questionnaire)
  - [Feature Enablement and Rollback](#feature-enablement-and-rollback)
  - [Rollout, Upgrade, and Rollback Planning](#rollout-upgrade-and-rollback-planning)
  - [Monitoring Requirements](#monitoring-requirements)
  - [Dependencies](#dependencies)
  - [Troubleshooting](#troubleshooting)
- [Implementation History](#implementation-history)
- [Drawbacks](#drawbacks)
- [Alternatives](#alternatives)
<!-- /toc -->

## Release Signoff Checklist

This KEP is provisional. Release signoff requires SIG Network acceptance,
completion of the implementation and test plan, and Production Readiness Review
approval. No item below is claimed complete by this design draft.

- [ ] (R) Enhancement issue links to this KEP and is in the release milestone
- [ ] (R) KEP approvers approve status `implementable`
- [ ] (R) Design details are complete
- [ ] (R) Test plan is implemented for every supported proxier mode
- [ ] (R) Production Readiness Review is complete and approved
- [ ] User-facing documentation and release note are prepared
- [ ] Implementation history and supporting links are current

## Summary

Add an opt-in `noMasqueradeCIDRs` setting to kube-proxy. For service traffic
from a node-local source that kube-proxy has marked for SNAT, kube-proxy skips
its own SNAT when the post-DNAT endpoint destination matches a configured CIDR.
Non-local Pod-IP sources and external-client traffic retain existing behavior.
HostNetwork Pods use node-local addresses and therefore follow the same rule
as other node-local processes. This lets operators whose routing supports
source preservation avoid kube-proxy replacing the node-local source with the
egress-interface address.

The setting does not guarantee a particular source IP, make a CIDR reachable,
or define NetworkPolicy behavior. Those depend on the node routes and CNI.

## Motivation

In [kubernetes/kubernetes#133024], a multi-interface Cilium native-routing
cluster reports that direct node-to-Pod traffic arrives with the node's
InternalIP, while traffic sent through a ClusterIP arrives with the egress
interface address. The report includes packet captures and a working manual
workaround. kube-proxy's postrouting `MASQUERADE` rule changes the source
address selected before SNAT; this is the narrow behavior this KEP addresses.

This is not a claim that Kubernetes guarantees a universal node-to-Pod source
IP. A cluster that cannot route replies to the preserved address must continue
to masquerade. The operator must opt in only for destinations whose network
supports the unmodified source.

### Evidence and reproduction

The report's reproduction uses Kubernetes 1.33.1, three kind nodes, Cilium
native routing, BGP-advertised Pod routes, and two node interfaces. A
`tcpdump` on the source node shows a request to a remote Pod through a Service
with source `10.16.0.2` before kube-proxy postrouting; after the `MASQUERADE`
rule it leaves with source `10.100.70.14`. A request directly to the Pod leaves
with source `10.16.0.2`. The report also demonstrates that inserting a
`RETURN` for the node source before kube-proxy's masquerade rule restores the
expected source for that setup. See the full [reproducer and packet captures].

I reviewed the reported trace and workaround against the described rule path.
I could not rerun the cluster locally. The published reproducer requires
`kind`, Cilium CLI, and containerlab; none is installed here, and its setup
script invokes `sudo containerlab`. Podman is available in rootless mode and a
container can change its own netfilter rules with `NET_ADMIN`, but the host has
no effective `CAP_NET_ADMIN`, `unshare -n` is denied, and `sudo` requires a
password. That is not the reported three-node topology. The report is direct
evidence for that Cilium setup, not proof that excluding arbitrary Pod CIDRs is
safe for every CNI or routing mode. The integration tests below are required to
establish that boundary for the implementation.

I separately exercised the relevant Linux iptables primitive in a rootless
Podman network namespace: a UDP packet addressed to a synthetic Service IP was
DNATed in `OUTPUT` to a loopback listener; in `POSTROUTING`, a rule matching
the post-DNAT destination and `--src-type LOCAL` returned before a following
`MASQUERADE` rule. The listener received the packet, the DNAT and `RETURN`
counters each increased by one, and the `MASQUERADE` counter stayed at zero.
This verifies the local-source match and post-DNAT destination ordering for a
locally generated packet. It does not exercise kube-proxy's masquerade mark,
remote Pod sources, real node interfaces, or the reported Cilium routing
topology, and is not a substitute for the integration cases below.

The reported source-based workaround and this destination-CIDR proposal are
not interchangeable in all networks. This KEP chooses a destination exclusion
because the rule is evaluated after service DNAT and can be scoped to endpoint
networks. It does not add a source-based allowlist.

### Goals

- Let an operator exempt selected post-DNAT endpoint destination CIDRs from
  kube-proxy's service SNAT.
- Preserve the existing behavior when the setting is empty or absent.
- Define behavior consistently for supported kube-proxy proxier modes, or
  explicitly reject unsupported modes rather than silently diverging.
- Test that a configured exclusion skips only kube-proxy's SNAT and leaves
  unrelated packets and marks unchanged.

### Non-Goals

- Guaranteeing that node-to-Pod traffic uses a node InternalIP.
- Changing CNI routing, NetworkPolicy semantics, or conformance requirements.
- Adding a `snatNodeInternalIP` option; that is a distinct behavior change
  tracked in [kubernetes/kubernetes#138360].
- Recommending that every CNI use a node's primary IP for all node-to-Pod
  traffic.
- Replacing `clusterCIDR`, `detectLocalMode`, or their existing semantics.

## Proposal

Add `NoMasqueradeCIDRs []string` to `KubeProxyConfiguration`, serialized as
`noMasqueradeCIDRs`. Each entry is a valid IPv4 or IPv6 CIDR for service
endpoint destinations. For a packet carrying kube-proxy's masquerade mark, the
proxier first performs its existing mark handling, then skips kube-proxy SNAT
only when both conditions hold: the post-DNAT destination matches a configured
CIDR, and the packet source is classified as local to the node by the
platform's routing table. The source check protects Pod-originated traffic,
including hairpin traffic, and external-client traffic from this exception.
Packets that do not satisfy both conditions retain existing behavior.

This feature is opt-in and is not enabled by default. Alpha support is limited
to Linux iptables and nftables modes that can implement the node-local source
and post-DNAT destination matches. IPVS and Windows are unsupported in alpha.
If a mode cannot honor the same match and mark semantics, kube-proxy must fail
configuration validation for non-empty `noMasqueradeCIDRs`; it must not ignore
the setting.

The field is added to the internal `KubeProxyConfiguration` type and its
`kubeproxy.config.k8s.io/v1alpha1` external type, including conversion,
defaulting, validation, deep-copy generation, and component-config reference
documentation. An omitted or empty list has no effect. The list accepts at
most 64 canonical CIDRs; host bits, malformed prefixes, and duplicates are
rejected. `KubeProxyNoMasqueradeCIDRs` must be enabled before a non-empty list
is accepted.

## Design Details

### Configuration and packet scope

- Parse and validate each CIDR at kube-proxy startup; reject malformed values
  and duplicate canonical prefixes with a clear error. Reject inputs with host
  bits set and reject more than 64 entries so rule expansion has a documented
  upper bound; operators with more destinations must aggregate prefixes.
- Evaluate the destination after service DNAT, so rules match the selected
  endpoint rather than the Service virtual IP.
- Evaluate the source as node-local using the routing-table local-address
  classification (`addrtype --src-type LOCAL` in iptables and the equivalent
  `fib saddr type local` match in nftables). This includes node addresses
  assigned to loopback and other local interfaces, including HostNetwork Pod
  addresses; it does not match remote Pod IPs or external clients.
- Apply the exception only to packets that kube-proxy would otherwise SNAT.
  Do not bypass unrelated CNI or administrator-owned postrouting rules.
- Clear or preserve kube-proxy's masquerade mark exactly as required by the
  current proxier before returning. Tests must verify there is no leaked mark
  and that packets not selected for SNAT are unaffected.
- For dual-stack clusters, IPv4 entries match only IPv4 packets and IPv6
  entries only IPv6 packets. A family with no configured entries is unchanged.
- On a matching packet, perform the existing kube-proxy mark cleanup before
  returning from kube-proxy's postrouting chain. Do not leave the mark set for
  later netfilter chains.
- The source-local check is required even when `masqueradeAll` marks a packet.
  It keeps ordinary Pod-IP and hairpin traffic on the existing SNAT path.
  HostNetwork Pods share node-local addresses with host processes and cannot be
  distinguished by this source match.
- Endpoint selection, locality, `externalTrafficPolicy`, `masqueradeAll`,
  hairpin traffic, NodePort, and LoadBalancer paths must be explicitly covered.
  The setting must not exempt traffic outside both its documented source and
  post-DNAT destination matches.

No CNI behavior is assumed. Before opting in, the administrator must verify
that return traffic to the preserved source address is routable and that the
selected CNI does not require kube-proxy SNAT for the configured destination
range.

### Compatibility and migration

An empty list preserves current behavior. Enabling a non-empty list changes
source-address behavior for matching service traffic and may cause asymmetric
routing or dropped replies if node routes and policy rules do not support the
original source. Operators should start with the narrowest endpoint CIDR that
has been verified in their cluster and roll back by removing the entry.

This configuration is local to kube-proxy and does not change the API server
or workload objects. During a kube-proxy rollout, nodes with and without the
setting can produce different source addresses; operators must account for
that skew when validating network policy and return routing.

### Test Plan

These tests are required before implementation is considered complete:

- Unit tests for CIDR parsing, IPv4/IPv6 matching, canonical duplicates,
  host-bit rejection, the 64-entry limit, empty configuration, invalid
  configuration, and the feature-gate requirement.
- Component-config tests for v1alpha1 to internal conversion, YAML round-trip,
  omitted/empty defaults, and strict rejection of the field when the gate is
  disabled.
- Proxier rule-generation tests showing a matching marked packet bypasses
  kube-proxy SNAT after DNAT; a non-matching marked packet still receives
  existing SNAT; an unmarked packet does not acquire new behavior; and the
  mark is not left set after the exclusion path.
- Rule-generation tests show the exception applies to node-local source
  addresses, including a loopback-assigned node address, but not to ordinary
  Pod-IP sources, hairpin packets, or external-client sources—even when
  `masqueradeAll` is enabled.
- Table-driven coverage for iptables and nftables. IPVS and Windows reject a
  non-empty setting during alpha rather than silently ignoring it.
- Linux integration test with two network namespaces and routes that can
  return to the original source: compare direct Pod-IP and Service-to-Pod
  traffic with the setting absent, with a matching destination CIDR, and with
  a non-matching CIDR. Capture source and destination at the endpoint and
  verify reply delivery.
- Integration cases for same-node and cross-node endpoints, both IP families,
  endpoint changes, and the documented Service traffic policies. Confirm
  ordinary Pod-IP, hairpin, and external-client source paths retain existing
  SNAT behavior, test HostNetwork Pods as node-local sources, and verify
  unrelated service destinations are unaffected.
- A multi-node CNI test is required before claiming compatibility with a
  specific CNI. The original issue's Cilium setup is a candidate, not a
  substitute for the generic proxier tests.

### Risks and mitigations

- **Asymmetric or unroutable replies:** the original source may not be reachable
  from the endpoint. Keep the option empty by default, require explicit narrow
  CIDRs, and document route verification and rollback.
- **Overbroad CIDR:** an exclusion can alter many service flows. Do not default
  to the cluster Pod CIDR; require operator selection and test non-matching
  traffic.
- **Proxier divergence:** alpha only supports iptables and nftables with the
  same local-source and post-DNAT destination semantics. Other modes reject
  the setting until parity is implemented and proven.
- **HostNetwork Pods:** their source is node-local and indistinguishable from
  a host process for this rule. Operators must include them in route and
  policy validation.
- **NetworkPolicy assumptions:** this changes the packet source observed at
  later hooks. Do not promise that policies will identify the source as a
  node; verify the actual CNI enforcement point.
- **Configuration skew:** different kube-proxy settings during rollout can
  produce mixed source addresses. Document the observable transition and
  require rollout verification.

### Graduation Criteria

#### Alpha

- The feature is behind `KubeProxyNoMasqueradeCIDRs`, disabled by default.
- Validation and rule-generation tests pass for each supported proxier.
- Linux integration tests verify matching and non-matching source paths.

#### Beta

- Multi-node tests cover supported proxier modes and dual-stack behavior.
- Operational documentation covers routing prerequisites, NetworkPolicy
  limitations, version skew, and rollback.
- No known proxier mode silently ignores the configuration.

#### GA

- At least two releases of field experience show no unresolved correctness
  issues in supported modes.
- Integration coverage remains enabled and stable in CI.

## Production Readiness Review Questionnaire

### Feature Enablement and Rollback

###### How can this feature be enabled / disabled in a live cluster?

Enable `KubeProxyNoMasqueradeCIDRs` on kube-proxy and configure a non-empty
`noMasqueradeCIDRs` list in its component config. The gate is disabled by
default. This is a node-local setting and takes effect after kube-proxy
restarts; no control-plane downtime is required. With the gate disabled, a
non-empty list must be rejected at startup rather than silently ignored.

###### Does enabling the feature change any default behavior?

Enabling the feature changes source NAT for new node-local service flows whose
post-DNAT destinations match the configured CIDRs. It does not change ordinary
Pod-IP or external-client source paths. HostNetwork Pods use node-local
addresses and follow the same rule as host processes. Existing conntrack
entries keep the NAT mapping created by their first packet; changing the
configuration does not rewrite those entries.

###### Can the feature be disabled once it has been enabled (i.e. can we roll back the enablement)?

Rollback by removing the CIDRs (or disabling the feature gate) and restarting
kube-proxy. New flows return to the existing SNAT behavior. Existing flows
that were created without SNAT may continue using their old conntrack mapping
until they expire or are explicitly removed; rollback does not guarantee
immediate restoration of SNAT for those tuples. The operator must ensure the
preserved source remains routable during that period.

###### What happens if we reenable the feature if it was previously rolled back?

If the CIDRs remain in the config and the gate is re-enabled, new matching
flows again bypass kube-proxy SNAT. Existing conntrack entries retain the
mapping created when they were first established.

###### Are there any tests for feature enablement/disablement?

Before alpha, tests must cover the gate enabled/disabled, invalid configuration,
startup and restart, mark cleanup, and existing conntrack mappings across
enable/rollback.

### Rollout, Upgrade, and Rollback Planning

###### How can a rollout or rollback fail? Can it impact already running workloads?

During a rolling upgrade, nodes with the setting may preserve node-local
sources while nodes without it SNAT them. Do not assume source-address parity
until all kube-proxy instances have the same configuration. Before alpha,
verify how an older kube-proxy decodes a configuration file containing the new
field; the upgrade procedure must not distribute the field to versions that
reject it. Rollback guidance must cover both the process restart and existing
conntrack mappings.

###### What specific metrics should inform a rollback?

No new metric is proposed. Use existing conntrack occupancy, packet captures,
application reachability, and CNI/network-policy telemetry. Conntrack count
alone does not show whether return routing works.

###### Were upgrade and rollback tested? Was the upgrade->downgrade->upgrade path tested?

Not yet. The config-version skew and existing-flow behavior are required
pre-alpha tests; the provisional KEP does not claim these paths have passed.

###### Is the rollout accompanied by any deprecations and/or removals of features, APIs, fields of API types, flags, etc.?

No. The setting is additive and empty by default.

### Monitoring Requirements

###### How can an operator determine if the feature is in use by workloads?

Inspect the kube-proxy config for a non-empty `noMasqueradeCIDRs` list and
confirm the feature gate is enabled. This does not prove any packet matches.

###### How can someone using this feature know that it is working for their instance?

This KEP adds no metric. Operators verify configuration load through kube-proxy
startup logs, inspect packet source/destination at the endpoint, and monitor
existing conntrack occupancy and application reachability. A successful rule
match alone does not prove the return route works.

###### What are the reasonable SLOs (Service Level Objectives) for the enhancement?

This KEP defines no new networking SLO. The operator's existing Service
availability and application SLOs remain applicable.

###### What are the SLIs (Service Level Indicators) an operator can use to determine the health of the service?

Use the existing application success rate and network reachability signals.
There is no kube-proxy metric that directly identifies a successful
no-masquerade return path.

###### Are there any missing metrics that would be useful to have to improve observability of this feature?

No metric is proposed because a rule match does not establish return-path
success and a per-flow metric would add cardinality and synchronization cost.

### Dependencies

###### Does this feature depend on any specific services running in the cluster?

The behavior depends on the node routing table classifying the original source
as local and on the CNI not rewriting it before kube-proxy's postrouting path.
Each configured prefix adds matching state/rules; the list is capped at 64
entries. Performance tests must cover the maximum supported list in both
iptables and nftables modes.

###### Will enabling / using this feature result in any new API calls?

No. kube-proxy uses its existing local configuration and packet-processing
path.

###### Will enabling / using this feature result in introducing new API types?

No new Kubernetes REST API types are introduced. The existing versioned
`KubeProxyConfiguration` component config gains a field.

###### Will enabling / using this feature result in any new calls to the cloud provider?

No.

###### Will enabling / using this feature result in increasing size or count of the existing API objects?

No Kubernetes API objects change. The local config file grows by at most 64
CIDR entries.

###### Will enabling / using this feature result in increasing time taken by any operations covered by existing SLIs/SLOs?

It adds a bounded source-and-destination match to kube-proxy's SNAT path.
Performance tests must measure rule sync and packet processing with 64 prefixes
in iptables and nftables mode before alpha.

###### Will enabling / using this feature result in non-negligible increase of resource usage (CPU, RAM, disk, IO, ...) in any components?

No control-plane usage is added. Node rule and set storage increases with the
configured CIDR list and is bounded at 64 entries; implementation tests must
measure maximum-size configuration.

###### Can enabling / using this feature result in resource exhaustion of some node resources (PIDs, sockets, inodes, etc.)?

No new PIDs, sockets, or conntrack entries are created by the feature. Rule
count is bounded by the 64-entry validation limit.

### Troubleshooting

###### How does this feature react if the API server and/or etcd is unavailable?

The feature does not call the API server or etcd. kube-proxy continues using
its loaded local configuration.

###### What are other known failure modes?

If the source is not node-local, the destination is not in the configured
CIDRs, a CNI rewrites the source earlier, or the endpoint cannot route replies,
the expected source preservation does not occur or traffic fails. In
unsupported proxier modes, startup must reject the non-empty setting.

###### What steps should be taken if SLOs are not being met to determine the problem?

Operators should confirm the feature gate and loaded CIDRs, then verify that
the flow source is node-local, the endpoint destination is post-DNAT and in a
configured CIDR, the kube-proxy mark is cleared, and the endpoint has a valid
return route. If any match prerequisite is absent, existing masquerading
behavior applies. If traffic fails, remove the CIDR and restart kube-proxy;
existing conntrack entries may require natural expiry or explicit cleanup.

## Implementation History

- 2026-07-17: Initial provisional draft combined masquerading exclusions,
  explicit node-IP SNAT, and CNI recommendations.
- 2026-10-20: Re-scoped to one opt-in SNAT exclusion and documented the
  available packet-capture evidence, limitations, and required reproducer.

## Drawbacks

- Adds kube-proxy configuration and proxier-specific rule complexity.
- Requires operators to understand their routing and CNI behavior before
  configuring the exclusion.
- Does not make node-to-Pod source IP behavior portable by itself.

## Alternatives

- **Keep existing masquerading:** preserves current behavior but cannot satisfy
  the demonstrated Cilium routing setup where kube-proxy replaces a usable
  node source with the egress-interface address.
- **Source-based `RETURN` rule:** the original report demonstrates a manual
  source exemption for its node IP. Managing per-node rules outside kube-proxy
  is not durable across kube-proxy syncs and does not provide declarative
  configuration.
- **SNAT explicitly to the node InternalIP:** the separate
  [kubernetes/kubernetes#138360] proposal addresses this behavior. It still
  translates the source and has node-IP discovery/fallback semantics; this KEP
  instead allows selected traffic to retain its pre-SNAT source.
- **CNI source-address recommendations:** not selected because Kubernetes
  does not guarantee one routing model or one source address for node traffic;
  the reproducer shows kube-proxy's own SNAT as the immediate cause.

[kubernetes/kubernetes#133024]: https://github.com/kubernetes/kubernetes/issues/133024
[kubernetes/kubernetes#138360]: https://github.com/kubernetes/kubernetes/pull/138360
[reproducer and packet captures]: https://github.com/naoki9911/kube-proxy-with-loopback
