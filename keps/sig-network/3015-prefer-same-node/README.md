# KEP-3015 PreferSameZone and PreferSameNode Traffic Distribution

<!-- toc -->
- [Summary](#summary)
- [Motivation](#motivation)
  - [Goals](#goals)
  - [Non-Goals](#non-goals)
- [Proposal](#proposal)
  - [User Stories](#user-stories)
    - [<code>PreferClose</code> vs <code>PreferSameZone</code>](#preferclose-vs-prefersamezone)
    - [DNS](#dns)
  - [Risks and Mitigations](#risks-and-mitigations)
- [Design Details](#design-details)
  - [Renaming/Deprecation of <code>PreferClose</code>](#renamingdeprecation-of-preferclose)
  - [Addition of <code>PreferSameNode</code>](#addition-of-prefersamenode)
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

## Summary

Discussion about trying to add "prefer same node" behavior to
`TrafficDistribution: PreferClose` ([#4931]) led to the conclusion
that any attempt to change the semantics of `PreferClose` would
inevitably have either too many false positives or too many false
negatives.

Thus, this KEP deprecates `PreferClose` and replaces it with
`PreferSameZone` as a new name for the old behavior. And it adds a new
value, `PreferSameNode`, indicating traffic for a service should
preferentially be routed to endpoints on the same node as the client.

[#4931]: https://github.com/kubernetes/enhancements/pull/4931

## Motivation

### Goals

- Make `TrafficDistribution` less ambiguous.

- Add a new value to allow configuring a service so that connections
  will be delivered to a local endpoint when possible, and a remote
  endpoint if not.

### Non-Goals

- Actually removing `PreferClose` from the API.

## Proposal

### User Stories

#### `PreferClose` vs `PreferSameZone`

As a user, I want to set up a service to have "prefer same zone"
traffic distribution, and I want to know that _it really does have
"prefer same zone" traffic distribution_. I don't want a "smart" proxy
to decide that I actually meant something else, because I didn't.

#### DNS

As a cluster administrator, I plan to run a DNS pod on each node, and
would like DNS requests from other pods to always go to the local DNS
pod, for efficiency. However, if no local DNS pod is available, DNS
should just go to a remote pod instead so it keeps working. There
should never be enough DNS traffic to overload any one endpoint, so
it's safe to use a TrafficDistribution mode that doesn't worry about
endpoint overload.

### Risks and Mitigations

By "locking down" the meaning of `PreferClose`/`PreferSameZone`, we
potentially limit the ability of future proxies to improve traffic
routing on their own.

However, actually _improving_ traffic routing requires the proxy to
have more information than `TrafficDistribution` currently provides.
(For example: does the user want to keep traffic in the same zone
because it's faster, or because it's cheaper?)

However, we are not likely to ever end up with a highly-detailed
system for explaining the user's relative preferences for latency,
bandwidth, cost, server load, etc, and many users have indicated that
they would be very happy with a way to _just_ express "prefer same
zone" without worrying that this might thwart some theoretical future
proxy. (And of course, if it _does_ thwart some theoretical future
proxy, they could just change the Service definition at that point.)

Also, while these alternative traffic distribution modes risk
overloading endpoints if used poorly, the API does not need to assume
that the service proxy is the _only_ component in the system that
cares about balancing endpoint load; the user may be using other APIs
such as Pod Topology Spread Constraints or Horizontal Autoscaling to
ensure that load is handled appropriately, and in those scenarios, the
alternatively traffic distribution modes may be completely safe.

## Design Details

### Renaming/Deprecation of `PreferClose`

KEP-4444 defines `PreferClose` as:

> * `PreferClose`: Indicates a preference for routing traffic to endpoints that
>   are topologically proximate to the client. The interpretation of
>   "topologically proximate" may vary across implementations and could encompass
>   endpoints within the same node, rack, zone, or even region.

We will add a new `TrafficDistribution` value, `PreferSameZone`, which
is defined as follows:

* `PreferSameZone`: Indicates a preference for routing traffic to
  endpoints that are in the same zone as the client. In general, the
  proxy should always route to a same-zone endpoint if any is
  available.

And we will say that henceforth, `PreferClose` is just a deprecated
alias for `PreferSameZone`, and never means anything other than
"prefer same zone".

Note that this does not change the existing implementation of
`PreferClose` at all, it only removes the possibility of future
changes.

(In theory, service proxies other than kube-proxy may already have
been implementing `PreferClose` with semantics other than the ones
kube-proxy used. Those implementations will need to be updated when
they add support for this KEP.)

### Addition of `PreferSameNode`

We will add a new `TrafficDistribution` value, `PreferSameNode`, which
is defined as follows:

* `PreferSameNode`: Indicates a preference for routing traffic to
  endpoints that are on the same node as the client. In general, the
  proxy should always route to a same-node endpoint if any is
  available.

We will add a new field to `discoveryv1.EndpointHints`:

```golang
// EndpointHints provides hints describing how an endpoint should be consumed.
type EndpointHints struct {
        ...

	// forNodes indicates the node(s) this endpoint should be targeted by.
	// +listType=atomic
	ForNodes []ForNode `json:"forNodes,omitempty" protobuf:"bytes,2,name=forNodes"`
}

// ForNode provides information about which nodes should consume this endpoint.
type ForNode struct {
	// name represents the name of the node.
	Name string `json:"name" protobuf:"bytes,1,name=name"`
}
```

(The KEP originally proposed `ForNodes []string` since there are no
use cases for additional information beyond node name. While
implementing it, we decided it was better to have the API be
consistent with `ForZones`.)

When updating EndpointSlices, if the EndpointSlice controller sees a
Service with `PreferSameNode` traffic distribution, then for each
endpoint in the slice, it will add a `ForNodes` hint including the
name of the endpoint's node. (The field is an array for future
extensibility, but initially it will always have either 0 or 1
elements.) In addition, it will set the `ForZones` hint as it would
with `TrafficDistribution: PreferClose`, to allow older service
proxies to fall back to at least same-zone behavior.

Kube-proxy's `CategorizeEndpoints` function will be updated as
follows:

  - If every endpoint has a `ForNodes` hint set, and at least one
    `ForNodes` hint includes the local node, then the set of
    _topologically available_ endpoints is the set of all endpoints
    with a `ForNodes` hint that includes the local node.

  - Otherwise, if every endpoint has a `ForZones` hint set, and at
    least one `ForZones` hint includes the local node's zone, then the
    set of _topologically available_ endpoints is the set of all
    endpoints with a `ForZones` hint that includes the local node's
    zone.

  - Otherwise, all endpoints are _topologically available_.

(The first step is new; the other two steps reflect the current code.)

### Test Plan

[X] I/we understand the owners of the involved components may require updates to
existing tests to make this code solid enough prior to committing the changes necessary
to implement this enhancement.

##### Prerequisite testing updates

N/A

##### Unit tests

Tests of validation, endpointslice-controller, and kube-proxy will be
updated.

- validation: [`TestValidateServiceCreate`](https://github.com/kubernetes/kubernetes/blob/v1.33.0/pkg/apis/core/validation/validation_test.go#L15472), [`TestValidateEndpointSlice`](https://github.com/kubernetes/kubernetes/blob/v1.33.0/pkg/apis/discovery/validation/validation_test.go#L34), [`Test_dropDisabledFieldsOnCreate`](https://github.com/kubernetes/kubernetes/blob/v1.33.0/pkg/registry/discovery/endpointslice/strategy_test.go#L37), [`Test_dropDisabledFieldsOnUpdate`](https://github.com/kubernetes/kubernetes/blob/v1.33.0/pkg/registry/discovery/endpointslice/strategy_test.go#L130)
- endpointslice-controller: [`TestReconcile_TrafficDistribution`](https://github.com/kubernetes/kubernetes/blob/v1.33.0/staging/src/k8s.io/endpointslice/reconciler_test.go#L1976), [`TestReconcileHints`](https://github.com/kubernetes/kubernetes/blob/v1.33.0/staging/src/k8s.io/endpointslice/trafficdist/trafficdist_test.go#L29)
- kube-proxy: [`TestCategorizeEndpoints`](https://github.com/kubernetes/kubernetes/blob/v1.33.0/pkg/proxy/topology_test.go#L48)

##### Integration tests

We will add a test that disabling the feature results in the hints
being removed from the EndpointSlice for Services using the new
`TrafficDistribution` values.

- [`Test_TransitionsForPreferSameTrafficDistribution`](https://github.com/kubernetes/kubernetes/blob/v1.33.0/test/integration/service/service_test.go#L556)

##### e2e tests

E2E tests will be added similar to existing traffic distribution
tests, to cover the new options.

- [`test/e2e/network/traffic_distribution.go`](https://github.com/kubernetes/kubernetes/blob/v1.33.0/test/e2e/network/traffic_distribution.go)

### Graduation Criteria

#### Alpha

- Feature implemented behind a feature flag

- Unit tests for API enablement and endpoint selection.

#### Beta

- E2E and Integration tests completed and enabled.

#### GA

- Time passes, no major objections

### Upgrade / Downgrade Strategy

On (initial) upgrade, no Services will be using the new
`TrafficDistribution` values, so there is nothing to do.

If the user starts using the new values and then downgrades, the
EndpointSlice controller will treat the unrecognized
`TrafficDistribution` value as equivalent to `nil` and remove the
hints from the slice, causing the Service to effectively fall back to
"default" traffic distribution. When they re-upgrade, if
`TrafficDistribution` is still set, the hints will be re-added.

### Version Skew Strategy

In clusters with an updated kube-apiserver but an old
kube-controller-manager, users would be able to set the new
`TrafficDistribution` values on a Service, but those values would not
be recognized. We avoid this by having the feature available in Alpha
for one release before enabling it by default in Beta.

In clusters with an updated kube-apiserver and
kube-controller-manager, but an old kube-proxy (or a third-party
service proxy that doesn't know about the new values),
`TrafficDistribution: PreferSameZone` would still work fine, because
kube-proxy does not look at the actual `TrafficDistribution` value, it
only looks at the hints set by kube-controller-manager, which would be
the same for `PreferSameZone` as they would have been with
`PreferClose`. For Services with `TrafficDistribution:
PreferSameNode`, the service proxy would not know about the `ForNodes`
hint, so it would be unable to provide prefer-same-node semantics, but
in a multi-zone cluster, kcm would also have set the `ForZones` hint,
so the Service would at least fall back to prefer-same-zone.

We are not doing anything to mitigate the kcm/kube-proxy skew problem
other than providing the same-zone fallback. Users for whom this is
not sufficient should just not depend on `PreferSameNode` until they
are sure their cluster is non-skewed.

## Production Readiness Review Questionnaire

### Feature Enablement and Rollback

###### How can this feature be enabled / disabled in a live cluster?

- [X] Feature gate (also fill in values in `kep.yaml`)
  - Feature gate name: PreferSameTrafficDistribution
  - Components depending on the feature gate:
    - kube-apiserver
    - kube-controller-manager
    - kube-proxy

###### Does enabling the feature change any default behavior?

No

###### Can the feature be disabled once it has been enabled (i.e. can we roll back the enablement)?

Yes.

###### What happens if we reenable the feature if it was previously rolled back?

It starts working again.

###### Are there any tests for feature enablement/disablement?

The unit tests in
`pkg/registry/discovery/endpointslice/strategy_test.go` confirm that
the EndpointSlice `forNodes` hint gets dropped from existing objects
on update when the feature gate is disabled. We will add a test

### Rollout, Upgrade and Rollback Planning

###### How can a rollout or rollback fail? Can it impact already running workloads?

An initial rollout cannot fail and won't impact already-running
workloads, because at the time of the initial rollout, there cannot
already be any `PreferSameZone` or `PreferSameNode` services.

A rollback has reasonable fallback behavior (as with downgrades), and
a re-rollout just updates the behavior of existing
`PreferSameZone`/`PreferSameNode` services in the expected way.

###### What specific metrics should inform a rollback?

There are no metrics that would inform anyone that the feature was
failing, but since the feature is opt-in, individual users can simply
stop using the feature if it is not working for them.

###### Were upgrade and rollback tested? Was the upgrade->downgrade->upgrade path tested?

Tested manually in a kind:

- Enabled feature gate, created a Service with `trafficDistribution:
  PreferSameNode`, added a Pod, confirmed that the EndpointSlice
  contained `forNodes` hints.

- Disabled feature gate, restarted apiserver/kcm, confirmed that the
  EndpointSlice still contained the `forNodes` hint. Added another Pod
  to the service and confirmed that the EndpointSlice was rewritten
  with no `forNodes` hint (for either endpoint).

- Re-enabled the feature gate, restarted apiserver/kcm, confirmed that
  the EndpointSlice still contained no `forNodes` hint. Deleted one of
  the Pods and confirmed that the EndpointSlice was rewritten
  with a `forNodes` hint for the remaining endpoint.

###### Is the rollout accompanied by any deprecations and/or removals of features, APIs, fields of API types, flags, etc.?

No

### Monitoring Requirements

###### How can an operator determine if the feature is in use by workloads?

By checking if any Service is using one of the new
`TrafficDistribution` values.

###### How can someone using this feature know that it is working for their instance?

As with other topology features, there is no easy way for an end user
to reliably confirm that it is working correctly other than by
sniffing the network traffic, or else looking at the logs of each
endpoint to confirm that they are receiving the expected connections
and not receiving unexpected connections.

###### What are the reasonable SLOs (Service Level Objectives) for the enhancement?

It inherits the existing SLOs around updating EndpointSlices and
programming the data plane. There should be no changes to either since
the amount of additional work is trivial.

(The effect that the feature has on the performance of end user
workloads that use the feature depends on those workloads.)

###### What are the SLIs (Service Level Indicators) an operator can use to determine the health of the service?

It inherits the existing SLIs around updating EndpointSlices and
programming the data plane.

(User workloads that use the feature may expose SLI information that
the user can examine to determine how well the feature is working for
their workload.)

###### Are there any missing metrics that would be useful to have to improve observability of this feature?

Not really; we don't know how fast the user's services are supposed to
be, so we can't really tell if we are improving them as much as they
hoped or not.

### Dependencies

###### Does this feature depend on any specific services running in the cluster?

Any service proxy that implemented `PreferClose` in the "standard" way
will automatically also handle `PreferSameZone`, because the proxy
does not actually look at `TrafficDistribution` itself; it only looks
at the EndpointSlice hints, which are set by kube-controller-manager.

The `PreferSameNode` semantics will require a service proxy that has
been updated to know about the `ForNodes` hint. We will update
`kube-proxy` ourselves, but network plugins / kubernetes distributions
that ship their own alternative service proxies will also need to be
updated to support the new value before their users can make use of
it. (Until then, `TrafficDistribution: PreferSameNode` would end up
falling back to the semantics of `TrafficDistribution:
PreferSameZone`.)

### Scalability

###### Will enabling / using this feature result in any new API calls?

No

###### Will enabling / using this feature result in introducing new API types?

No

###### Will enabling / using this feature result in any new calls to the cloud provider?

No

###### Will enabling / using this feature result in increasing size or count of the existing API objects?

Using `trafficDistribution: PreferSameNode` will result in each
endpoint in each EndpointSlice for the service gaining a `forNodes`
hint containing the endpoint's node name.

###### Will enabling / using this feature result in increasing time taken by any operations covered by existing SLIs/SLOs?

No

###### Will enabling / using this feature result in non-negligible increase of resource usage (CPU, RAM, disk, IO, ...) in any components?

No

### Troubleshooting

###### How does this feature react if the API server and/or etcd is unavailable?

No change from existing service/proxy behavior.

###### What are other known failure modes?

None known

###### What steps should be taken if SLOs are not being met to determine the problem?

N/A

## Implementation History

- Initial proposal as `InternalTrafficPolicy: PreferLocal`: 2021-10-21
- Initial proposal as "Node-level topology": 2022-01-15
- Initial proposal as `TrafficDistribution: PreferSameNode`: 2025-02-06
- Added `TrafficDistribution: PreferSameZone`: 2025-02-08
- Implemented as Alpha in k8s 1.33: 2025-05-30

## Drawbacks

## Alternatives

Node-level topology was one of many things that was possible with
the old [`TopologyKeys`] API, but that API was deprecated because of
other problems.

This is the third attempt at node-level topology specifically.

The initial proposal ([#3016]) was for `internalTrafficPolicy:
PreferLocal`, but we decided that traffic policy was for
semantically-significant changes to how traffic was distributed,
whereas this is just a hint, like topology.

That led to the second attempt ([#3293]), which never got as far as
defining a specific API, but reframed the problem as being a kind of
topology hint. This eventually fizzled out because of people's
opinions at that time about how topology ought to work in Kubernetes.

However, KEP-4444 (TrafficDistribution) represents an updated
understanding of topology in Kubernetes, which makes the idea of
node-level topology palatable.

[`TopologyKeys`]: /keps/sig-network/536-topology-aware-routing/README.md
[#3016]: https://github.com/kubernetes/enhancements/pull/3016
[#3293]: https://github.com/kubernetes/enhancements/pull/3293
