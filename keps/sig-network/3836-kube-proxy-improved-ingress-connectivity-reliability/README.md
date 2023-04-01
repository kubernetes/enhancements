# KEP-3836: Kube-proxy improved ingress connectivity reliability

<!-- toc -->
- [Release Signoff Checklist](#release-signoff-checklist)
- [Summary](#summary)
- [Motivation](#motivation)
  - [Goals](#goals)
  - [Non-Goals](#non-goals)
- [Proposal](#proposal)
    - [Risk](#risk)
    - [Mitigations](#mitigations)
- [Design Details](#design-details)
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

The service controller in the Kubernetes cloud controller manager (KCCM)
configures the load balancer and its corresponding health check (HC) following
service related events. The configured health check is then used by the load
balancer as to determine which instances are candidates for traffic load
balancing. For certain cloud providers (GCP): the KCCM configures the HC to
target the Kubernetes service proxy for this information. In this KEP we will
focus on Kube-proxy, since it's the only service proxy under the responsibility
of the Kubernetes project.

We can define two classes of services for what concerns the HC:

- `externalTrafficPolicy: Cluster` / `eTP:Cluster` (default)
- `externalTrafficPolicy: Local` / `eTP:Local`

For class `eTP:Cluster` services: Kube-proxy currently returns an answer
following its `healthz` state (specifically, whether the data-plane programming
is known to be stale). For `eTP:Local` services: Kube-proxy only reports if the
service for which the load balancer was created for, has a `Ready` endpoint
running on the node. This KEP will only focus on the former case.

This KEP proposes three changes:

1. That Kube-proxy provides a mechanism for load balancers to do connection
  draining for terminating Nodes. This is to be done by Kube-proxy by inspecting
  a field on the Node object which indicates that the "node is
  terminating/deleting" and when seen, starts failing its `healthz` and
  subsequently the LB HC. When discussing this scenario, the primary case this
  applied to was: downscaling by the cluster autoscaler (CA). Given that the CA
  taints the Node which is to be downscaled and deleted, with the taint:
  `ToBeDeletedByClusterAutoscaler` it would seem most appropriate to use that
  here. It is unfortunate to have this taint spread around the code base, but
  for now: no better indicator has been thought of. The `.spec.unschedulable`
  field has been discussed as well.  Setting that field is usually followed by
  eviction and Node termination, but it doesn't have as strong of a direct link
  to the termination/deletion of the Node, as the taint does. Users can for
  example decide to cordon all nodes during at given moment in time. Using
  `.spec.unschedulable` as this signal would cause ingress traffic for all
  `eTP:Cluster` services to break.

2. That Kube-proxy adds a `/livez` path to its health check server
  (`proxier_health.go`) corresponding to the old healthz semantics (i.e: whether
  the data-plane programming is known to be stale).

3. This KEP does not attempt to align cloud providers for health checking
   `eTP:Cluster` services. It recognizes cloud providers have valid reasons for
   doing this differently depending on their implementations. However, it does
   want to recommend ways for cloud providers to do health checking in a better
   way on Kubernetes clusters. The KEP hence proposes that a document be added
   to https://kubernetes.io/docs/concepts/ which can act as a formal guide and
   be utilized as knowledge sharing with cloud providers.

## Motivation

The motivation for each change is:

1. Nodes used as intermediate nexthop for `eTP:Cluster` services would allow all
  connections passing through the node while it is being terminated, to
  gracefully shutdown.

2. Adding this new `/livez` path will allow vendors / users of Kube-proxy to
   specify a `livenessProbe` which isn't impacted by any node termination
   indicator. It will indicate Kube-proxy health, only, just as is the case
   today. This is a low-hanging fruit which requires modifying the Kube-proxy
   DeamonSet spec to opt-in on.

3. Cloud providers have very different ways of ascertaining if a load balancer
   should target a specific node for `eTP:Cluster` services. We would like to
   highlight the benefits of certain methods and pitfalls of others, in a formal
   document, so that this is known. The hope is that this allows the information
   to act as a source of knowledge for how to adapt their implementations to the
   mechanics of a Kubernetes cluster.

### Goals

- Offer a better capability of connection draining terminating Nodes, for load
  balancers which support that.

### Non-Goals

- Aligning cloud provider HCs for `eTP:Cluster` services. Cloud providers like
  Azure/AWS do not configure their HCs to point to the service proxy. Instead
  they connect to the `NodePort` defined for the service. As to have them
  benefit from the proposals of this KEP, they would need to change their
  implementation.

- That Kube-proxy includes its `healthz` state AND its current answer w.r.t the
  local endpoints, when it answers to the HC for `eTP:Local` services.
  Kube-proxy is currently defined as "unhealthy" when `2 * syncPeriod` passes in
  which it knows that it needs to update the data plane (iptables/ipvs), but has
  not actually done so. Not including the `healthz` state can cause Kube-proxy
  to indicate to a load balancer that it should send traffic to a Node simply
  because the endpoint is scheduled there, even though Kube-proxy might not be
  healthy and successfully managed to write the rules required for actually
  being able to forward traffic to the endpoint. This has however been agreed is
  a bug and will be treated as such, as opposed to following the KEP cycle for
  it.

## Proposal

#### Risk

The risk are:

1. Vendors of Kubernetes which deploy Kube-proxy and specify a `livenessProbe`
   targeting `/healthz` are expected to start seeing a CrashLooping Kube-proxy
   when the Node gets tainted with `ToBeDeletedByClusterAutoscaler`. This is
   because: if we modify `/healthz` to fail when this taint gets added on the
   Node, then the `livenessProbe` will fail, causing the Kubelet to restart the
   Pod until the Node is deleted. As far as we can tell, no vendor set
   `livenessProbe`, nor does kubeadm, so the risk is low.

2. By not being able to watch the Node object (while failing to read from the
   API server, for example) we might have all Kube-proxy start failing the HCs
   at once. That being said: Kube-proxy currently watches the Node object and is
   susceptible to this risk.

#### Mitigations

1. Such problems are expected to surface during the Beta phase when the feature
   gate will be enabled by default. The mitigation at that point would be to set
   the feature gate to "off" and default back to current behavior.
   Alternatively, to start using the `/livez` path which will keep the old
   semantics. We will also make the graduation criteria _to_ Beta be the
   document we would like to write and mention this as an explicit
   recommendation of what not to do when deploying Kube-proxy. As such any
   vendor doing this would get a heads-up during Alpha.

2. If Kube-proxy starts failing when reading from the API server, it should just
   assume that the last state seen continues. For Kube-proxy to be made aware of
   this, it needs to invoke
   `serviceInformer.Informer().SetWatchErrorHandler(DefaultWatchErrorHandler)`
   when initializing its informers. Any errors observed by client-go when
   watching from the API server will be reported on `DefaultWatchErrorHandler`.

3. Metrics should inform on Kube-proxy health and include information about its
   `healthz`/`livez` state, this can then be used to correlate to networking
   metrics surrounding new/established connections on the node. Ex: a failing
   `healthz` should correlate to a total drop in the count of new connections
   and with a zero-or-negative rate of established connections. E2E tests should
   also be designed with this specific goal in mind, i.e: validating the impact
   of a failing kube-proxy on ingress connectivity. Kube-proxy currently has a
   lot of metrics regarding how its health is doing, but no direct red/green
   indicator of what the end result of its health is. A couple of such metric
   could be `proxy_healthz_200_count` /
   `proxy_healthz_503_count`/`proxy_livez_200_count` / `proxy_livez_503_count`

4. The feature could be disabled for user who is dependent upon such behavior by
   means of flipping the feature flag to off.

## Design Details

1. Implement Kube-proxy change invoking client-go's `SetWatchErrorHandler`  on
   watch errors from the API server. This addresses the second point in
   [Mitigations](#mitigations)

2. Implement change in Kube-proxy which will react to changes on the Node object
   and once the taint `ToBeDeletedByClusterAutoscaler` is placed on the Node
   object: start failing it's `healthz` state.

3. Write document to be published at: https://kubernetes.io/docs/concepts/ which
   details: a) how determining node/instance health can best be done for
   Kubernetes clusters b) how Kube-proxy will do it once the changes proposed in
   this KEP are merged c) what some pitfalls with other methods might be.

### Test Plan

#### Prerequisite testing updates

#### Unit tests

Update the Kube-proxy unit tests to include the `healthz` answer for the
healthcheck server test suite.

- `k8s.io/kubernetes/pkg/proxy/healthcheck`: `09/Feb/2023` - `68.8%`

#### Integration tests

This feature is not readily integration tested, so we will use unit and E2E.

#### e2e tests

- Add E2E tests for connectivity to services on terminating nodes and
  validate graceful termination of TCP connections.

### Graduation Criteria

#### Alpha

- E2E tests coded before any feature implementation is made which highlights the
  existing problem.
- Feature implemented behind a feature flag.
- Document written at https://kubernetes.io/docs/concepts/

#### Beta

- No issues reported.
- Decision on final field on Node object to be used as an indicator of "node is
  terminating/deleting"

#### GA

- No issues reported during two releases.

### Upgrade / Downgrade Strategy

Any upgrade to a version enabling the feature, succeeded by a downgrade to a
version disabling it, is not expected to be impact ingress in any way, given
that Kube-proxy is healthy on all cluster nodes. Should any Kube-proxy not be
healthy: then ingress for `eTP:Cluster` services won't be using that node as a
nexthop for ingress traffic. This would have been the case in the preceding
version

### Version Skew Strategy

This doesn't touch load balancer / HC API, so even though an old Kube-proxy
might talk to a newer control plane, there's no real concern.

## Production Readiness Review Questionnaire

### Feature Enablement and Rollback

###### How can this feature be enabled / disabled in a live cluster?

- [X] Feature gate (also fill in values in `kep.yaml`)
  - Feature gate name: `KubeProxyDrainingTerminatingNodes`
  - Components depending on the feature gate: Kube-proxy

###### Does enabling the feature change any default behavior?

Yes. For `eTP:Cluster` services: Kube-proxy currently doesn't include any logic
about terminating / deleting nodes when determining if it's healthy. This will
be the case going forward, whereby the addition of the taint
`ToBeDeletedByClusterAutoscaler` will cause Kube-proxy to fail its `healthz`.

###### Can the feature be disabled once it has been enabled (i.e. can we roll back the enablement)?

Yes, by resetting the feature gate back.

###### What happens if we reenable the feature if it was previously rolled back?

Behavior will be restored back immediately.

###### Are there any tests for feature enablement/disablement?

Not needed, since the feature is purely in-memory thing with no consequences for
any persistent data.

### Rollout, Upgrade and Rollback Planning

###### How can a rollout or rollback fail? Can it impact already running workloads?

This change is localized to Kube-proxy only. On upgrades Kube-proxy will
restart, so client connectivity is impacted in any case. If applications are
running on Nodes which are tainted with `ToBeDeletedByClusterAutoscaler`, but
which are experiencing delay for draining: then ingress SLAs might be impacted,
whereby ingress connectivity for new connections experience a drop below what's
accepted. But all application pods should be running on these terminating Nodes
in that case.

###### What specific metrics should inform a rollback?

The metric: `proxy_healthz_503_count` mentioned in [Monitoring
requirements](#monitoring-requirements) will inform on red `healthz`.
`proxy_livez_503_count` will inform on red `livez` state. If the `healthz` count
is increasing but the `livez` does not: then a problem might have occurred with
the node related reconciliation logic.

###### Were upgrade and rollback tested? Was the upgrade->downgrade->upgrade path tested?

Once the change is implemented: the author will work with Kubernetes vendors to
test the upgrade/downgrade scenario in a cloud environment.

###### Is the rollout accompanied by any deprecations and/or removals of features, APIs, fields of API types, flags, etc.?

No

### Monitoring Requirements

Four new metrics
`proxy_healthz_200_count`/`proxy_healthz_503_count`/`proxy_livez_200_count`/`proxy_livez_503_count`
which will count the amount of reported successful/unsuccessful health check
invocations. A drop in this metric can then be correlated to impacted ingress
connectivity, for endpoints running on those nodes.

###### How can an operator determine if the feature is in use by workloads?

- By connecting to service of `type: LoadBalancer` and `eTP:Cluster` through a
  terminating/tainted Node and validating that any new connections are blocked,
  and established connections are fine.

###### How can someone using this feature know that it is working for their instance?

For `eTP:Cluster`: their connections will terminate gracefully when the node
used as a nexthop for their connection is terminating or tainted.

###### What are the reasonable SLOs (Service Level Objectives) for the enhancement?

No

###### What are the SLIs (Service Level Indicators) an operator can use to determine the health of the service?

- [X] Metrics
  - Metric name: `proxy_healthz_200_count`
  - Metric name: `proxy_healthz_503_count`
  - Metric name: `proxy_livez_200_count`
  - Metric name: `proxy_livez_503_count`

###### Are there any missing metrics that would be useful to have to improve observability of this feature?

N/A

### Dependencies

No

###### Does this feature depend on any specific services running in the cluster?

No

### Scalability

###### Will enabling / using this feature result in any new API calls?

No

###### Will enabling / using this feature result in introducing new API types?

No

###### Will enabling / using this feature result in any new calls to the cloud provider?

No

###### Will enabling / using this feature result in increasing size or count of the existing API objects?

No

###### Will enabling / using this feature result in increasing time taken by any operations covered by existing SLIs/SLOs?

No

###### Will enabling / using this feature result in non-negligible increase of resource usage (CPU, RAM, disk, IO, ...) in any components?

No

###### Can enabling / using this feature result in resource exhaustion of some node resources (PIDs, sockets, inodes, etc.)?

No

### Troubleshooting

###### How does this feature react if the API server and/or etcd is unavailable?

Not any different than today.

###### What are other known failure modes?

###### What steps should be taken if SLOs are not being met to determine the problem?

## Implementation History

- 2023-02-03: Initial proposal

## Drawbacks

## Alternatives
