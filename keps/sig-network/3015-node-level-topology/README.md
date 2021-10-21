# KEP-3015: Node-Level Service Topology

<!-- toc -->
- [Release Signoff Checklist](#release-signoff-checklist)
- [Summary](#summary)
- [Motivation](#motivation)
  - [Goals](#goals)
  - [Non-Goals](#non-goals)
- [Proposal](#proposal)
  - [User Stories](#user-stories)
    - [Per-Node DNS Pods](#per-node-dns-pods)
- [Design Details](#design-details)
  - [Applying Node-Level Topology](#applying-node-level-topology)
  - [Choosing Services to Apply Node-Level Topology To](#choosing-services-to-apply-node-level-topology-to)
    - [Approach 1: Manual](#approach-1-manual)
    - [Approach 2: Automatic](#approach-2-automatic)
    - [Approach 3: Semi-Automatic or Automatic-With-Opt-Out](#approach-3-semi-automatic-or-automatic-with-opt-out)
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
  - [ ] (R) Ensure GA e2e tests for meet requirements for [Conformance Tests](https://github.com/kubernetes/community/blob/master/contributors/devel/sig-architecture/conformance-tests.md) 
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

The `Local` policy for `service.spec.externalTrafficPolicy` and
`service.spec.internalTrafficPolicy` allows configuring services so
that they will always deliver to local endpoints, avoiding an
unnecessary hop to another node when the request could just as easily
be processed locally.

However, they both fail entirely if there are no endpoints for the
service on the node. This tradeoff is unacceptable for some use cases.

The [Topology Aware Hints feature] provides a way of signaling to
kube-proxy that it should optimize service delivery on a
per-availability zone basis, but does not provide any way to optimize
on any other basis.

This KEP adds a new topology hint, to tell kube-proxy that a Service
is expected to have an endpoint on every node most of the time, and so
it should optimize traffic by always routing to a local endpoint _when
one is available_, but using other topology hints or ordinary
`Cluster` traffic policy when not.

[Topology Aware Hints feature]: ../2433-topology-aware-hints/

## Motivation

### Goals

- Allow configuring a service so that connections will be delivered to
  a local endpoint when possible, and a remote endpoint if not.

```
<<[UNRESOLVED deprecate-iTP]>>

- Deprecate `internalTrafficPolicy`? It's clear that the DNS use case
  given in the Internal Traffic Policy KEP is not actually a good use
  case for Internal Traffic Policy, because no one wants the behavior
  of "I'd rather have DNS requests get dropped than have them go to
  another node". But without the DNS use case, it's not clear that
  there's really a strong argument for Internal Traffic Policy at all.

<<[/UNRESOLVED]>>
```

### Non-Goals

- Providing any improvements over `externalTrafficPolicy: Local` for
  LoadBalancer services. Consensus is that [Proxy Terminating
  Endpoints] should solve the problems that made
  `externalTrafficPolicy: Local` unreliable for some cases.

- Attempting to improve the endpoint selection algorithm for services
  with a number of endpoints that is not (roughly) an integer multiple
  of the number of nodes. If you have a service with 15 endpoints in a
  cluster with 10 nodes, then the service could potentially be
  optimized by making clients more likely to select a local endpoint,
  but they can't _exclusively_ select local endpoints or the endpoints
  on nodes with only a single endpoint would presumably get
  overloaded. But this KEP does not attempt to figure out any new
  behavior there.

[Proxy Terminating Endpoints]: ../1669-graceful-termination-local-external-traffic-policy/

## Proposal

### User Stories

#### Per-Node DNS Pods

As a cluster administrator, I plan to run a DNS pod on each node, and
would like DNS requests from other pods to always go to the local DNS
pod, for efficiency. However, if no local DNS pod is available, DNS
should automatically just go to a remote pod instead.

(This is a modified version of a user story from the [Internal Traffic
Policy KEP]. The original implies (by omission) that is is acceptable
to have DNS lookups fail when there is no local DNS pod.)

[Internal Traffic Policy KEP]: ../2086-service-internal-traffic-policy/

## Design Details

### Applying Node-Level Topology

Given a Service to which node-level topology applies (see next
section), the implementation is simple: if there is at least one
endpoint for the service on the local node, then filter the list of
endpoints to include only local ones. (If there are no local
endpoints, then use the same rules as would have been used if
node-level topology was not in effect.)

As with Topology Aware Hints, Node-Level Topology would only apply to
connections with `Cluster` traffic policy, because
`internalTrafficPolicy: Local` semantically _requires_ local delivery
(and is not allowed to fall back when there are no local endpoints),
and `externalTrafficPolicy: Local` requires source IP preservation
(which is generally not possible when sending to a non-local
endpoint).

### Choosing Services to Apply Node-Level Topology To

```
<<[UNRESOLVED]>>

Pick an approach.

The Automatic approach seems simplest (and would have the effect of
immediately making DNS faster in most(?) clusters). But maybe it would
do bad things in some circumstances?

<<[/UNRESOLVED]>>
```

#### Approach 1: Manual

One approach would be to allow users to explicitly tag services where
they want this behavior.

This could be implemented via the existing
`"service.kubernetes.io/topology-aware-hints"` annotation perhaps.
Instead of setting it to `Auto`, the user could set it to `Node` to
indicate node-level topology rather than zone-level.

(This would imply that a service could not use both node-level
topology and zone-level topology, which is not necessarily terrible;
the zone-level topology would only be useful when the node-level
topology "failed". But if we wanted to cover that case, then we could
have a separate annotation for node-level topology.)

#### Approach 2: Automatic

Alternatively, the EndpointSlice controller could attempt to determine
automatically which services would benefit from node-level topology.
For example, the EndpointSlice controller could look for services
where:

  - The service has an endpoint on every node, or at least "almost
    every" node. (eg, no more than N or N% of nodes are missing an
    endpoint).

  - The pods that make up the service endpoints all have an
    `OwnerReference` pointing to the same `DaemonSet`.

  - ...

If all relevant criteria are met, then it could set a new field in the
`Hints` of the EndpointSlices for the service, and kube-proxy would
use this to know to do node-level topology.

#### Approach 3: Semi-Automatic or Automatic-With-Opt-Out

Some combination of the above two approaches, where the EndpointSlice
controller requires that the `topology-aware-hints` annotation (or
another annotation) either is, or is not, set on the Service, before
investigating the endpoints to decide whether to enable the feature.

("Automatic with opt-out" is probably not a good idea; it implies that
there are some services where users wouldn't want node-level topology,
but also implies that we will break those services on upgrade for
users who forgot to pre-emptively opt out when upgrading to the
version of Kubernetes where this feature is enabled by default.)

### Test Plan

[X] I/we understand the owners of the involved components may require updates to
existing tests to make this code solid enough prior to committing the changes necessary
to implement this enhancement.

##### Prerequisite testing updates

None needed.

##### Unit tests

We will add unit tests to the iptables and ipvs proxiers to confirm
that they generate the expected rules for services using the new
feature, both when a local endpoint is available and when it is not.

- `k8s.io/kubernetes/pkg/proxy/iptables`: 2022-10-04 - 67.7%
- `k8s.io/kubernetes/pkg/proxy/ipvs`: 2022-10-04 - 55.5%

##### Integration tests

Kube-proxy is mostly not tested by integration tests.

##### e2e tests

We will add e2e tests to confirm that the new feature works as
expected, both when a local endpoint is available and when it is not.

(In real-world usage, when all endpoints of the service behave the
same, it is difficult to confirm that traffic really is going to an
endpoint on the same node, but for testing purposes it's easy enough
to just make the endpoints behave slightly differently on each node so
we can tell which endpoint we hit.)

<!--
This question should be filled when targeting a release.
For Alpha, describe what tests will be added to ensure proper quality of the enhancement.

For Beta and GA, add links to added tests together with links to k8s-triage for those tests:
https://storage.googleapis.com/k8s-triage/index.html

We expect no non-infra related flakes in the last month as a GA graduation criteria.
-->

- <test>: <link to test coverage>

### Graduation Criteria

#### Alpha

- Feature implemented behind a feature flag
- Unit tests for api enablement

#### Beta

- Initial e2e tests completed and enabled.

#### GA

- Time passes, no major objections

### Upgrade / Downgrade Strategy

If we use the "Automatic" approach, then on upgrade, some services
that previously had standard `Cluster` semantics would now get
node-level topology. Of course, we would only use the "Automatic"
approach if we believed this was a safe change.

When downgrading, older kube-controller-manager / kube-proxy would not
be aware of node-level topology, and would just fall back to ordinary
`Cluster` semantics for the affected services, resulting in
less-efficient but still correct functioning.

### Version Skew Strategy

For the most part, version skew will result in node-level topology not
happening and the service falling back to ordinary `Cluster`
semantics.

One tricky case is that if kube-controller-manager is downgraded (or
has its feature gate disabled), but kube-apiserver and kube-proxy stay
new/enabled, then a service might get "stuck" with node-level topology
enabled in its EndpointSlices. Even this is unlikely to be a big
problem; services are not likely to flip back and forth between
"suitable for node-level topology" and "not suitable for node-level
topology", so if the previous version of kcm declared the service to
be suitable for node-level topology, then it probably doesn't matter
if the new version doesn't keep checking that it's still suitable.
(And even in the worse case, the result is just that the endpoints get
used in an unbalanced way, not that the service actually breaks.) And
the version skew should eventually get resolved.

## Production Readiness Review Questionnaire

### Feature Enablement and Rollback

###### How can this feature be enabled / disabled in a live cluster?

- [X] Feature gate (also fill in values in `kep.yaml`)
  - Feature gate name: NodeLevelTopology
  - Components depending on the feature gate:
    - kube-apiserver
    - kube-controller-manager
    - kube-proxy

###### Does enabling the feature change any default behavior?

Potentially, depending on UNRESOLVED decisions above.

###### Can the feature be disabled once it has been enabled (i.e. can we roll back the enablement)?

Yes; services that were using node-level topology will just revert
back to ordinary `Cluster` semantics.

###### What happens if we reenable the feature if it was previously rolled back?

It starts working again.

###### Are there any tests for feature enablement/disablement?

Not yet.

### Rollout, Upgrade and Rollback Planning

###### How can a rollout or rollback fail? Can it impact already running workloads?

Assuming we use the "Automatic" semantics, a rollout would affect the
behavior of existing services (such as CoreDNS) that met the criteria
for automatically getting node-local topology. Assuming the feature
works correctly though, it would affect them in a positive way.

If we used the "Manual" semantics then rolling out the feature should
have no effect at all.

###### What specific metrics should inform a rollback?

There are no metrics that would inform anyone that the feature was
failing (discussed more below).

###### Were upgrade and rollback tested? Was the upgrade->downgrade->upgrade path tested?

No

###### Is the rollout accompanied by any deprecations and/or removals of features, APIs, fields of API types, flags, etc.?

No

### Monitoring Requirements

###### How can an operator determine if the feature is in use by workloads?

Assuming the "Manual" model, they could check all services to see if
they were annotated to use the feature. Assuming the "Automatic"
model, there is no easy way to determine thisl

###### How can someone using this feature know that it is working for their instance?

They can know that it hasn't broken anything because their services
won't be broken.

There's no simple way to confirm that kube-proxy is optimizing traffic
correctly based on this feature. The user would need to packet-sniff,
or check individual endpoint logs, or look at the iptables rules
generated by kube-proxy.

###### What are the reasonable SLOs (Service Level Objectives) for the enhancement?

Services using node-level topology should be as reliable as services
_not_ using node-level topology, and should have lower latency.

###### What are the SLIs (Service Level Indicators) an operator can use to determine the health of the service?

- [X] Other

There are no existing metrics for determining the availability or
response time of individual Services, and this KEP does not add any.
In general, the operator cannot know what response times the user
expects from their Services anyway.

The developer may be able to generate their own metrics based on logs
from the endpoint pods and/or logs generated by the service's clients;
but this involves user-owned code, not kubernetes code, on both ends.

###### Are there any missing metrics that would be useful to have to improve observability of this feature?

Not really. We could expose the number of node-local-topology services
on each node, but it's not clear that that would be useful.

### Dependencies

###### Does this feature depend on any specific services running in the cluster?

It depends on a service proxy which understands the feature. We will
update `kube-proxy` ourselves, but network plugins / kubernetes
distributions that ship their own alternative service proxies will
also need to be updated to support the feature before their users can
benefit from it.

### Scalability

###### Will enabling / using this feature result in any new API calls?

No

###### Will enabling / using this feature result in introducing new API types?

No

###### Will enabling / using this feature result in any new calls to the cloud provider?

No

###### Will enabling / using this feature result in increasing size or count of the existing API objects?

It might add a new field to EndpointSliceHints (which would be unset
in most EndpointSlices).

###### Will enabling / using this feature result in increasing time taken by any operations covered by existing SLIs/SLOs?

No. It should _decrease_ the average DNS response time.

###### Will enabling / using this feature result in non-negligible increase of resource usage (CPU, RAM, disk, IO, ...) in any components?

No. It should (slightly) _decrease_ the usage of the network, by
keeping most DNS traffic local rather than cross-node.

### Troubleshooting

###### How does this feature react if the API server and/or etcd is unavailable?

No change from existing service/proxy behavior.

###### What are other known failure modes?

None known

###### What steps should be taken if SLOs are not being met to determine the problem?

If enabling node-level topology makes any services slower then that is
a bug in the feature, so you should file a bug.

## Implementation History

- Initial "PreferLocal traffic policy" proposal: 2021-10-21
- Updated: 2022-01-15
- Initial "Node-Local topology" proposal: 2022-05-03

## Drawbacks

## Alternatives

The [original version] of this proposal suggested adding a
`PreferLocal` option to `internalTrafficPolicy` and
`externalTrafficPolicy`. Discussion on that PR led to an agreement
that this problem was better solved with topology than with traffic
policy, leading to this PR.

[original version]: https://github.com/kubernetes/enhancements/pull/3016
