# KEP-1669: Graceful Termination for Local External Traffic Policy

<!-- toc -->
- [Release Signoff Checklist](#release-signoff-checklist)
- [Summary](#summary)
- [Motivation](#motivation)
  - [Goals](#goals)
  - [Non-Goals](#non-goals)
- [Proposal](#proposal)
  - [User Stories (optional)](#user-stories-optional)
    - [Story 1](#story-1)
  - [Risks and Mitigations](#risks-and-mitigations)
- [Design Details](#design-details)
  - [Additions to EndpointSlice](#additions-to-endpointslice)
  - [kube-proxy](#kube-proxy)
  - [Test Plan](#test-plan)
    - [Unit Tests](#unit-tests)
    - [E2E Tests](#e2e-tests)
  - [Graduation Criteria](#graduation-criteria)
    - [Alpha](#alpha)
  - [Upgrade / Downgrade Strategy](#upgrade--downgrade-strategy)
  - [Version Skew Strategy](#version-skew-strategy)
- [Implementation History](#implementation-history)
- [Drawbacks](#drawbacks)
- [Alternatives](#alternatives)
<!-- /toc -->

## Release Signoff Checklist

- [X] Enhancement issue in release milestone, which links to KEP dir in [kubernetes/enhancements] (not the initial KEP PR)
- [ ] KEP approvers have approved the KEP status as `implementable`
- [ ] Design details are appropriately documented
- [ ] Test plan is in place, giving consideration to SIG Architecture and SIG Testing input
- [ ] Graduation criteria is in place
- [ ] "Implementation History" section is up-to-date for milestone
- [ ] User-facing documentation has been created in [kubernetes/website], for publication to [kubernetes.io]
- [ ] Supporting documentation e.g., additional design documents, links to mailing list discussions/SIG meetings, relevant PRs/issues, release notes

[kubernetes.io]: https://kubernetes.io/
[kubernetes/enhancements]: https://git.k8s.io/enhancements
[kubernetes/kubernetes]: https://git.k8s.io/kubernetes
[kubernetes/website]: https://git.k8s.io/website

## Summary

Services with externalTrafficPolicy=Local lack the ability to gracefully handle traffic from a loadbalancer when it goes from N to 0 endpoints.
Since terminating pods are never considered "ready" in Endpoints/EndpointSlice, a node with only terminating endpoints would drop traffic even though
it may still be part of a loadbalancer's node pool. Even with loadbalancer health checks, there is usually a delay between when the health check
fails and when a node is completely decommissioned. This KEP proposes changes to gracefully handle traffic to a node that has only terminating endpoints
for a Service with externalTrafficPolicy=Local.

## Motivation

### Goals

* enable zero downtime rolling updates for Services with ExternalTrafficPolicy=Local via nodeports/loadbalancerIPs/externalIPs.

### Non-Goals

* changing the behavior of terminating pods/endpoints outside the scope of Services with ExternalTrafficPolicy=Local via a nodeport/loadbalancerIPs/externalIPs.

## Proposal

This KEP proposes that if all endpoints for a given Service (with externalTrafficPolicy=Local) within the bounds of a node are terminating (i.e pod.DeletionTimestamp != nil),
then all external traffic on this node should be sent to **ready** and **not ready** terminating endpoints, preferring the former if there are any. This ensures that traffic
is not dropped between the time a node fails its health check (has 0 endpoints) and when a node is decommissioned from the loadbalancer's node pool.

The proposed changes in this KEP depend on KEP-1672 and the EndpointSlice API.

### User Stories (optional)

#### Story 1

As a user I would like to do a rolling update of a Deployment fronted by a Service Type=LoadBalancer with ExternalTrafficPolicy=Local.
If a node that has only 1 pod of said deployment goes into the `Terminating` state, all traffic to that node is dropped until either a new pod
comes up or my cloud provider removes the node from the loadbalancer's node pool. Ideally the terminating pod should gracefully handle traffic to this node
until either one of the conditions are satisfied.

### Risks and Mitigations

There are scalability implications to tracking termination state in EndpointSlice. For now we are assuming that the performance trade-offs are worthwhile but
future testing may change this decision. See KEP 1672 for more details.

## Design Details

### Additions to EndpointSlice

This work depends on the `Terminating` condition existing on the EndpointSlice API (see KEP 1672) in order to check the termination state of an endpoint.

### kube-proxy

Updates to kube-proxy when watching EndpointSlice:
* update kube-proxy endpoints info to track terminating endpoints based on endpoint.condition.terminating in EndpointSlice.
* update kube-proxy endpoints info to track endpoint readiness based on endpoint.condition.ready in EndpointSlice
* if externalTrafficPolicy=Local, record all local endpoints that are ready && terminating and endpoints that are !ready && terminating. When there are no local ready endpoints, fall back in the preferred order:
  * local ready & terminating endpoints
  * local not ready & terminating endpoints
  * blackhole traffic
* for all other traffic (i.e. externalTrafficPolicy=Cluster), preserve existing behavior where traffic is only sent to ready && !terminating endpoints.

In addition, kube-proxy's node port health check should fail if there are only `Terminating` endpoints, regardless of their readiness in order to:
* remove the node from a loadbalancer's node pool as quickly as possible
* gracefully handle any new connections that arrive before the loadbalancer is able to remove the node
* allow existing connections to gracefully terminate

### Test Plan

#### Unit Tests

kube-proxy unit tests:

* Unit tests will validate the correct behavior when there are only local terminating endpoints.
* Unit tests will validate the new change in behavior only applies for Services with ExternalTrafficPolicy=Local via nodeports/loadbalancerIPs/externalIPs.
* Existing unit tests will validate that terminating endpoints are only used when there are no ready endpoints AND externalTrafficPolicy=Local, otherwise ready && !terminating endpoints are used.
* Unit tests will validate health check node port succeeds only when there are ready && !terminating endpoints.

#### E2E Tests

E2E tests will be added to validate that no traffic is dropped during a rolling update for a Service with ExternalTrafficPolicy=Local.
This test may be marked "Flaky" as the behavior is largely also dependant on the cloud provider's loadbalancer.

All existing E2E tests for Services should continue to pass.

### Graduation Criteria

#### Alpha

* kube-proxy internally tracks the terminating condition of an endpoint.
* feature is only enabled if the feature gate `EndpointSliceTerminatingCondition` is on.
* unit tests in kube-proxy.

### Upgrade / Downgrade Strategy

Behavioral changes to terminating endpoints will apply once kube-proxy is upgraded to v1.19 and the `EndpointSlice`/`EndpointSliceProxying` feature gates are enabled.
On downgrade, the worse case scenario is that kube-proxy falls back to the existing behavior. See [Version Skew Strategy](#version-skew-strategy) below.

### Version Skew Strategy

The worse case version skew scenario is that kube-proxy falls back to the existing behavior today where traffic does not fall back to terminating endpoints.
This would either happen if a version of the control plane was not aware of the additions to EndpointSlice or if the version of kube-proxy did not know to consume the additions to EndpointSlice.

There's not much risk involved as the worse case scenario is falling back to existing behavior.

## Implementation History

- [x] 2020-04-23: KEP accepted as implementable for v1.19

## Drawbacks

* scalability: this KEP (and KEP 1672) would add more writes per endpoint to EndpointSlice as each terminating endpoint adds at least 1 and at
most 2 additional writes - 1 write for marking an endpoint as "terminating" and another if an endpoint changes it's readiness during termination.
* complexity: an additional corner case is added to kube-proxy adding to it's complexity.

## Alternatives

Some users work around this issue today by adding a preStop hook that sleeps for some duration. Though this may work in some scenarios, better handling from kube-proxy
would alleviate the need for this work around altogether.

