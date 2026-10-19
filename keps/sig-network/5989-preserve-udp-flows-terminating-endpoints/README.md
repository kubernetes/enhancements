# KEP-5989: Preserve UDP flows while an endpoint drains existing work

<!-- toc -->
- [Release Signoff Checklist](#release-signoff-checklist)
- [Summary](#summary)
- [Motivation](#motivation)
  - [Evidence and reproduction](#evidence-and-reproduction)
  - [Goals](#goals)
  - [Non-Goals](#non-goals)
- [Proposal](#proposal)
  - [Implementation Approach](#implementation-approach)
  - [Kube-proxy Changes](#kube-proxy-changes)
  - [Risks and Mitigations](#risks-and-mitigations)
  - [Design Details](#design-details)
    - [Flow and endpoint semantics](#flow-and-endpoint-semantics)
    - [Test Plan](#test-plan)
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
[kubernetes/kubernetes#134143]: https://github.com/kubernetes/kubernetes/issues/134143
[`CleanStaleEntries`]: https://github.com/kubernetes/kubernetes/blob/bbf142898f3/pkg/proxy/conntrack/cleanup.go#L47
[`TestCleanStaleEntries`]: https://github.com/kubernetes/kubernetes/blob/bbf142898f3/pkg/proxy/conntrack/cleanup_test.go#L89
[`podToEndpoint`]: https://github.com/kubernetes/kubernetes/blob/bbf142898f3/staging/src/k8s.io/endpointslice/utils.go#L33-L54

## Summary

This KEP proposes an explicit endpoint signal for UDP applications that have
stopped accepting new work but can still process datagrams for existing
sessions. A `processingProbe` on one container reports that state. The kubelet
publishes it as a Pod `Processing` condition, the EndpointSlice controller
copies it to an EndpointSlice `processing` condition, and kube-proxy retains
existing UDP conntrack entries only while that condition is true and the
endpoint remains present. Endpoint readiness and new-flow selection are not
changed. Entries are removed when processing ends or the endpoint disappears;
inactive entries can also expire under the kernel's configured timeout. The
feature does not promise a hard maximum age for active flows.

## Motivation

The motivating [kubernetes/kubernetes#134143] report describes a UDP server
whose Pod becomes not Ready through its readiness check while the process is
still completing an existing SIP transaction. The endpoint is not necessarily
terminating. Readiness should stop new traffic, but an application may still
be able to answer packets belonging to an existing UDP exchange.

Current EndpointSlice generation sets `serving` from Pod readiness. Current
kube-proxy removes UDP conntrack entries when that endpoint is no longer
serving. Thus a normal selector-backed Pod cannot currently express
`Ready=false` while still processing existing exchanges. Treating every
non-serving endpoint as drainable would preserve flows to genuinely unhealthy
Pods and can blackhole traffic, so the new behavior requires a separate,
explicit signal from the workload.

The signal is an application contract, not a kube-proxy health judgment. A
successful processing probe means the application can still handle packets for
existing sessions; it does not make the Pod Ready or eligible for ordinary new
flow selection. If an application reports `Processing=true` after it has
stopped handling those sessions, kube-proxy may retain mappings that blackhole
traffic. Probe correctness remains the workload author's responsibility.

### Evidence and reproduction

In the current upstream source, [`CleanStaleEntries`] builds the retained
backend set from `Endpoint.IsServing()`. The existing [`TestCleanStaleEntries`]
exercises removal of flows to a non-serving endpoint and retention of flows to
a serving endpoint. I made that fixture explicit and ran
`GOTMPDIR=/home/krosci/tmp-go-build go test ./pkg/proxy/conntrack -run '^TestCleanStaleEntries$' -count=1`
against upstream Kubernetes commit `bbf142898f3`; it passed with both cases:

- `Ready=false, Serving=true, Terminating=true`: the endpoint's existing UDP
  flow remains. kube-proxy already retains this case; it is not new behavior.
- `Ready=false, Serving=false, Terminating=false`: the endpoint's existing UDP
  flow is removed. This is the behavior the new processing signal would change
  only when the workload explicitly reports `Processing=true`.

The synthetic `Ready=false, Serving=true` fixture does not prove that a
selector-backed Pod can currently produce that state. In current upstream,
[`podToEndpoint`] derives `Serving` from Pod readiness. The proposed container
probe and EndpointSlice propagation are therefore required parts of the design; this
KEP does not present the conntrack unit test as an end-to-end reproduction.

The source reproduction verifies conntrack reconciliation only; it does not
prove an end-to-end SIP transaction survives. A multi-node packet-level test
is required before claiming that result. I could not run such a cluster here
because this environment lacks `kind`, a container runtime, and `CAP_NET_ADMIN`.

### Goals

- Add a container `processingProbe` and propagate its state to EndpointSlices.
- Preserve existing UDP conntrack entries only for endpoints explicitly
  reporting `processing=true`.
- Leave all existing new-flow selection rules unchanged.
- Delete retained entries when the endpoint or Service is removed, or when
  the conntrack entry expires under the kernel's configured idle timeout.
- Preserve current behavior for all Services by default.

### Non-Goals

- This KEP does not change `Ready`, `Serving`, or `Terminating` semantics.
- This KEP does not make processing endpoints eligible for new flows.
- This KEP does not change how TCP connections are handled.
- This KEP does not add a universal hard age limit for conntrack entries. The
  kernel's idle timeout is not an absolute maximum lifetime for a flow whose
  packets continue to refresh it.

## Proposal

Add a `processingProbe` field to `Container`, using existing probe handlers
and thresholds. At most one container per Pod may configure it because
EndpointSlice `processing` is endpoint-wide and this KEP does not define
per-container or per-port aggregation. The probe reports whether the
application can still process datagrams for existing UDP sessions after
readiness becomes false. A successful probe sets the Pod's `Processing`
condition to true; a failed, absent, or unsupported probe means false. The
EndpointSlice controller publishes this as an optional `processing` condition.
Unknown or absent values mean false, preserving current behavior. No ServicePort
field is added: the drain state belongs to a particular workload instance, not
every endpoint behind a Service.

### Implementation Approach

1. **Container API and kubelet**: Add `processingProbe` using the existing
   probe handlers and thresholds. API validation permits it on at most one
   container in a Pod. The kubelet owns and updates only the Pod `Processing`
   condition, preserving conditions written by other actors. The condition
   defaults to false when the probe is absent or failing, and becomes false
   when the container is not running. Tests must cover concurrent kubelet
   status updates so this condition does not overwrite unrelated Pod
   conditions.

2. **EndpointSlice API and controller**: Add an optional `processing`
   condition. For selector-managed Services, the EndpointSlice controller sets
   it true only when the target Pod is Running, its `Processing` condition is
   true, and its Node is Ready. A missing Node or a non-Ready Node forces false
   so stale Pod status cannot retain traffic to an unreachable node. For
   manually managed EndpointSlices, the producer owns the same availability
   responsibility. Unknown or absent values mean false. This condition is
   independent of `ready`, `serving`, and `terminating`.

3. **Kube-proxy conntrack logic**: Retain a UDP conntrack entry when its
   backend endpoint is still present and `processing=true`, even if it is not
   Ready or Serving. Delete it when processing becomes false or the endpoint
   disappears. Processing alone must not make an endpoint eligible for a new
   flow. Existing terminating-endpoint fallback behavior remains unchanged and
   is an explicit exception to ordinary readiness-based selection.

4. **Timeout and removal**: Inactive entries expire under the node's configured
   conntrack timeout. Active traffic may refresh UDP conntrack state; there is
   no absolute maximum age. Endpoint or Service deletion triggers normal
   cleanup.

### Kube-proxy Changes

The implementation adds a probe and Pod status condition, propagates that
condition into EndpointSlices and proxy endpoint state, and changes UDP
conntrack reconciliation. It must preserve backend selection behavior and
cover every proxier mode that claims support.

1. **Pod and EndpointSlice state**:
   - `processingProbe` is configured on at most one container per Pod and
     reports whether the workload
     is still handling existing sessions. Its result is independent of the
     readiness probe. The kubelet writes a Pod `Processing` condition.
   - The EndpointSlice controller propagates that condition to an optional
     EndpointSlice `processing` condition. For manually managed slices, the
     producer sets it. Missing, unknown, and false values all mean not
     processing.
   - The condition applies to the Pod endpoint as a whole, across its UDP
     Service ports. A workload whose UDP ports need independent drain states
     cannot use this API without a future per-port signal.

2. **Kube-proxy conntrack**: Add `processing=true` endpoints to the retained
   backend set for UDP conntrack reconciliation. Keep endpoint selection
   unchanged: readiness and existing terminating-endpoint fallback rules
   determine new-flow eligibility. When processing becomes false or the
   endpoint is removed from the Service's EndpointSlices, the tuple becomes
   stale and is deleted.

3. **Proxier support**: Alpha support is limited to Linux iptables and
   nftables kube-proxy modes after packet-level proof. IPVS and Windows are
   unsupported in alpha; enabling `UDPConnectionDrain` in kube-proxy with
   either mode must fail startup rather than silently ignore the condition.
   Support for either requires an implementation and the same packet-level
   proof. Do not assume conntrack retention alone preserves the backend
   mapping or prescribe an untested platform fallback.

### Risks and Mitigations

- **Stale processing signal:** If the probe reports true after the application
  has stopped processing existing sessions, mappings can blackhole packets.
  The probe contract requires the application to report false before it can no
  longer process them; kube-proxy cannot infer application health.
- **Conntrack pressure:** Active traffic can refresh UDP entries, so preserving
  mappings while an endpoint remains present can extend their lifetime. Do not
  claim a fixed hard timeout based on the kernel's idle timeout. Monitor the
  node's existing conntrack usage and remove or disable the policy if the
  workload exceeds its budget.
- **NetworkPolicy or external routing:** kube-proxy retention alone does not
  guarantee packet delivery. CNI policy, node routing, Pod health, and external
  load balancer behavior are outside this contract.
- **Version skew:** old kubelets, EndpointSlice controllers, API servers, or
  kube-proxy instances may ignore or prune the new probe/condition. During a
  mixed-version rollout, preservation is not uniform; operators must treat
  the feature as unavailable until all relevant components support it.
- **Endpoint removal:** a flow must be cleaned once its endpoint or Service is
  removed, even when `processing=true`, to avoid forwarding to an address that
  may have been reused.

### Design Details

#### Flow and endpoint semantics

When an endpoint reports `processing=true`, kube-proxy retains its existing UDP
conntrack entries while the endpoint remains present in the Service's current
EndpointSlice state. `processing` does not change endpoint selection; the
existing readiness and terminating-endpoint fallback rules continue to govern
new flows. When processing becomes false, the endpoint is removed, or the
Service is deleted, kube-proxy deletes stale entries for that backend.

| Endpoint state | Existing UDP conntrack entry | New-flow selection |
| --- | --- | --- |
| Ready/Serving, Processing false or absent | Retained by existing behavior | Existing behavior |
| Not Ready/Serving false, Processing true, present | Retained until Processing=false, removal, or kernel expiry | Existing selection; Processing adds no eligibility |
| Not Ready/Serving false, Processing false or absent | Cleaned by existing behavior | Existing behavior |
| Node not Ready | Controller-managed Processing is forced false; existing Serving retention is unchanged | Existing behavior |
| Terminating and Serving true | Retained by existing behavior even without Processing | Existing terminating-endpoint fallback remains unchanged |
| Endpoint or Service removed | Cleaned regardless of Processing | Not eligible |

Setting Processing true after an entry has already been cleaned cannot restore
its previous backend mapping. The probe should report true before or when Pod
readiness becomes false; if kube-proxy observes the non-serving state first, it
may clean the entry before Processing propagates. During drain completion, the
workload must keep the process alive while it reports Processing=false and
allows the condition to propagate. Cleanup occurs on the next kube-proxy
reconciliation; the KEP does not promise instantaneous propagation across
nodes or uninterrupted delivery during these transitions.

This specifies kube-proxy's mapping and cleanup behavior, not end-to-end
session continuity. The proposal does not override application failure,
network policy, connection timeouts, route changes, or an external load
balancer's decisions. A successful packet-level test must establish whether
responses to the original UDP tuple still reach the client for every supported
proxier mode.

#### Test Plan

- **Kubelet tests**: verify absent probe defaults to `Processing=false`, probe
  success/failure thresholds update the Pod condition, and processing state is
  independent of Pod readiness. Verify kubelet status updates preserve
  conditions owned by other actors. Verify behavior when Processing becomes
  true before, with, and after readiness becomes false, and when the container
  exits before the probe reports false.
- **EndpointSlice API/controller tests**: verify Pod `Processing` propagation,
  condition transitions, absent/unknown default behavior, Pod phase and Node
  readiness overrides, and manual-slice producer behavior. Verify that Node
  NotReady suppresses only the new Processing retention and does not alter
  existing Serving-based retention.
- **Unit tests** for `pkg/proxy/conntrack`: a
  `Ready=false, Serving=false, Processing=true, Terminating=false` endpoint
  retains an existing UDP mapping; Processing=false, endpoint removal, and
  Service removal clean it. Existing serving and terminating behavior remains
  unchanged when the new condition is absent.
- **API tests**: verify feature-gate behavior, container probe
  validation/defaulting, and rejection of multiple processing probes per Pod,
  Pod status condition handling, and EndpointSlice condition round-tripping.
- **Proxier tests**: verify Processing does not add an endpoint to new-flow
  selection. Cover the existing terminating-endpoint fallback separately;
  do not infer selection behavior from a conntrack unit test.
- Linux integration test with real network namespaces: keep one UDP client
  tuple active while its backend reports Processing=true and Ready=false, then
  verify packets on that tuple still reach the original backend while a new
  tuple follows the existing endpoint selection rules. Change Processing to
  false and verify cleanup; repeat after endpoint removal.
- Run packet-level tests for iptables and nftables before alpha. IPVS and
  Windows are not supported by this KEP's initial implementation; do not claim
  support for either until a follow-up design and tests prove the semantics.
- Do not claim SIP, QUIC, external load balancer, or application-level session
  continuity until a corresponding end-to-end test exercises that behavior.

### Graduation Criteria

#### Alpha

- Container `processingProbe` and EndpointSlice `processing` condition are gated and
  disabled by default; absent condition preserves current behavior.
- Unit tests cover probe state propagation, stale-entry transitions, endpoint
  and Service deletion, and unchanged new-flow selection.
- Packet-level integration tests pass for every proxier mode included in the
  alpha support contract.

#### Beta

- Multi-node e2e coverage verifies an established UDP tuple survives while
  Processing=true and new tuples follow the documented endpoint selection.
- Version-skew and downgrade behavior is documented and tested.
- Operational feedback confirms that workload authors can implement and
  monitor the processing-probe contract safely.

#### GA

- Feature gate locked to true after supported modes meet the same semantics.
- Conformance coverage is considered only after API review defines a portable
  observable contract across supported environments.

### Upgrade / Downgrade Strategy

During upgrade, old kubelets and controllers do not publish or propagate the
Processing condition, and old kube-proxy instances use existing cleanup
behavior. During downgrade, an older API server may prune the new probe or
EndpointSlice condition. Before alpha, test the full API-server, kubelet,
controller-manager, and kube-proxy skew path. If downgrade can drop fields,
document that operators must clear the probe before downgrade and that restoring
the new version will not restore dropped values automatically.

### Version Skew Strategy

New kube-proxies use current cleanup behavior when `processing` is absent or
false. Old kube-proxy instances ignore the condition and clean entries using
their existing rules. Test whether older API servers preserve or prune the Pod
probe and EndpointSlice condition; this KEP does not assume round-trip
compatibility.

## Production Readiness Review Questionnaire

### Feature Enablement and Rollback

###### How can this feature be enabled / disabled in a live cluster?

- [x] Feature gate (also fill in values in `kep.yaml`)
  - Feature gate name: UDPConnectionDrain
  - Components depending on the feature gate: kube-apiserver, kubelet,
    kube-controller-manager, kube-proxy

###### Does enabling the feature change any default behavior?

No. With the gate disabled, current behavior is unchanged. With the gate
enabled, only endpoints explicitly reporting `Processing=true` retain existing
UDP conntrack entries while non-serving.

###### Can the feature be disabled once it has been enabled (i.e. can we roll back the enablement)?

Disabling the feature gate restores current cleanup behavior. API-server
handling of stored values must be tested; an older API server may prune the
probe or EndpointSlice condition it does not recognize.

###### What happens if we reenable the feature if it was previously rolled back?

The feature resumes only if the API server preserved the probe configuration
and EndpointSlice condition.

###### Are there any tests for feature enablement/disablement?

Unit and API tests must cover the probe and condition with the gate enabled and
disabled, including whether API-server round trips preserve them.

### Rollout, Upgrade and Rollback Planning

###### How can a rollout or rollback fail? Can it impact already running workloads?

During a rollout, supporting nodes preserve mappings only when the propagated
condition is true; older instances may clean them using current behavior.
Preservation is not uniform during skew. A stale or incorrect probe can retain
a mapping to an endpoint that cannot answer packets. Downgrade field retention
must be verified before alpha.

###### What specific metrics should inform a rollback?

Use existing node conntrack occupancy (`nf_conntrack_count` relative to
`nf_conntrack_max`) and application behavior. This KEP adds no custom metrics.

###### Were upgrade and rollback tested? Was the upgrade->downgrade path tested?

Before alpha, test supported API-server, kubelet, controller-manager, and
kube-proxy version skew, upgrade, downgrade, and re-upgrade. Verify probe and
condition propagation as well as flow cleanup; do not defer this to beta.

### Monitoring Requirements

###### How can an operator determine if the feature is functioning as expected?

- Verify the Pod's `Processing` condition and the EndpointSlice's Ready,
  Serving, Terminating, and Processing conditions.
- Inspect `conntrack -L -p udp` for the Service tuple and backend while the
  endpoint remains in the EndpointSlice.
- Verify a new client tuple follows existing endpoint selection rules.

###### What are the reasonable SLOs for the feature?

This KEP defines no application packet-loss SLO. The test plan must establish
data-plane behavior for each supported proxier and topology.

###### What are the SLIs that can be used to measure the feature?

Existing node conntrack occupancy and application-level transaction success
are relevant operational signals; neither alone proves the feature's packet
path behavior.

### Dependencies

###### Does this feature depend on any specific services, capabilities, or projects?

- **Kernel/proxier:** Behavior depends on each proxier's flow-tracking and
  endpoint-removal semantics. Support must be tested rather than inferred from
  Linux conntrack behavior alone.
- **CNI and NetworkPolicy:** They can still drop packets; kube-proxy retention
  does not guarantee end-to-end delivery.

### Scalability

###### Will enabling this feature impact the performance of the cluster?

The condition may retain actively refreshed flows while an endpoint remains
present. The reconciliation path already scans conntrack state; measure added
work and table occupancy instead of assuming a fixed cap or negligible cost.

###### How many retained UDP flows per node can this feature sustain?

There is no fixed count or lifetime guarantee. Capacity depends on the node's
conntrack limit, baseline usage, active UDP traffic, and correct processing
probe behavior. Operators must size and monitor that capacity.

### Troubleshooting

###### How does this feature react if the API server becomes unavailable?

Kube-proxy operates on cached Service and EndpointSlice state. If the API
server is unavailable, the last observed policy and endpoint state remain in
effect until caches change or the process restarts. The KEP makes no stronger
guarantee during API-server loss.

###### What are other known failure modes?

- **Stale processing signal:** preserved packets may fail if the probe reports
  true after the application stops processing existing sessions.
- **Conntrack pressure:** active traffic can refresh entries while the endpoint
  remains present; monitor existing node-level conntrack usage.
- **NetworkPolicy or routing drops:** kube-proxy retention cannot override
  those components.

###### What steps can be taken to debug this feature?

1. Check the Pod Processing probe/condition and the EndpointSlice conditions.
2. Run `conntrack -L -p udp` on the node to inspect active conntrack entries.
3. Check kube-proxy logs for the target endpoint's IP and port.
4. Verify that the EndpointSlice processing condition is true and that the
   workload's UDP ports share the endpoint-wide drain state.

## Implementation History

- 2026-03-31: Initial KEP draft (provisional) with API changes (Processing field).
- 2026-05-02: Revised to use Service annotation instead of API changes.
- 2026-05-02: Revised to use conntrack marks and netfilter operations.
- 2026-05-09: Revised to use a Service-level connection type hint (`udpConnectionHandling`).
- 2026-10-20: Corrected the target scenario to non-terminating non-serving
  endpoints; documented current terminating-endpoint behavior and test evidence;
  removed the unverified hard timeout, metrics, and IPVS assumptions.
- 2026-10-20: Reframed the API around a container processing probe and EndpointSlice
  condition after verifying that standard EndpointSlice generation mirrors Pod
  readiness and cannot express the motivating drain state. Removed the
  speculative ServicePort field and clarified the end-to-end proof boundary.

## Drawbacks

- Adds a container probe and EndpointSlice condition, plus kubelet,
  controller-manager, API-server, and kube-proxy version-skew obligations.
- Increases conntrack reconciliation complexity and can keep mappings to an
  unhealthy backend while the endpoint remains present.
- Does not guarantee successful application responses or end-to-end session
  continuity.

## Alternatives

- **Keep current cleanup behavior:** This remains the default and is correct
  when a non-serving endpoint must fail over immediately. It does not let an
  application stop accepting new flows while finishing existing exchanges.
- **Infer draining from `Ready=false` or `Terminating=true`:** Rejected because
  readiness failure also represents unhealthy endpoints, and termination
  alone does not say whether the application can process existing datagrams.
- **Rely only on `Serving=true`:** kube-proxy already retains flows to serving
  endpoints. Standard EndpointSlice generation derives Serving from Pod
  readiness, so a selector-backed Pod cannot use it to independently report
  the motivating drain state.
- **Use a readiness gate:** Rejected because a readiness gate changes Pod
  `Ready`, and the EndpointSlice controller currently derives `Serving` from
  that Pod condition. It therefore cannot simultaneously stop new-flow
  selection and preserve existing UDP conntrack entries.
- **Use a ServicePort connection-type hint:** Rejected because it identifies a
  protocol or port, not whether a particular endpoint can still process
  existing sessions. It would retain mappings for both intentionally draining
  endpoints and endpoints that became unhealthy.
