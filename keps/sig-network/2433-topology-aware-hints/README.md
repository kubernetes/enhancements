# KEP: Topology Aware Hints
<!-- toc -->
- [Release Signoff Checklist](#release-signoff-checklist)
- [Summary](#summary)
- [Motivation](#motivation)
  - [Goals](#goals)
  - [Non-Goals](#non-goals)
- [Proposal](#proposal)
  - [Risks and Mitigations](#risks-and-mitigations)
- [Design Details](#design-details)
  - [Assumptions](#assumptions)
  - [Identifying Zones](#identifying-zones)
  - [Excluding Control Plane Nodes](#excluding-control-plane-nodes)
  - [Configuration](#configuration)
    - [Interoperability](#interoperability)
    - [Feature Gate](#feature-gate)
  - [API](#api)
    - [Future API Expansion](#future-api-expansion)
  - [Kube-Proxy](#kube-proxy)
  - [EndpointSlice Controller](#endpointslice-controller)
    - [Example](#example)
    - [Overload](#overload)
    - [Handling Node Updates](#handling-node-updates)
  - [Future Expansion](#future-expansion)
  - [Test Plan](#test-plan)
    - [Unit tests](#unit-tests)
      - [Controller Unit Tests](#controller-unit-tests)
      - [Kube-Proxy Unit Tests](#kube-proxy-unit-tests)
    - [Integration tests](#integration-tests)
    - [e2e tests](#e2e-tests)
  - [Observability](#observability)
  - [Events](#events)
    - [Logic](#logic)
    - [Sample Events](#sample-events)
    - [Documentation](#documentation)
    - [Limitations](#limitations)
  - [Graduation Criteria](#graduation-criteria)
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

- [x] (R) Enhancement issue in release milestone, which links to KEP dir in [kubernetes/enhancements] (not the initial KEP PR)
- [x] (R) KEP approvers have approved the KEP status as `implementable`
- [x] (R) Design details are appropriately documented
- [x] (R) Test plan is in place, giving consideration to SIG Architecture and SIG Testing input
- [x] (R) Graduation criteria is in place
- [x] (R) Production readiness review completed
- [x] (R) Production readiness review approved
- [ ] "Implementation History" section is up-to-date for milestone
- [ ] User-facing documentation has been created in [kubernetes/website], for publication to [kubernetes.io]
- [ ] Supporting documentation—e.g., additional design documents, links to mailing list discussions/SIG meetings, relevant PRs/issues, release notes

[kubernetes.io]: https://kubernetes.io/
[kubernetes/enhancements]: https://git.k8s.io/enhancements
[kubernetes/kubernetes]: https://git.k8s.io/kubernetes
[kubernetes/website]: https://git.k8s.io/website

## Summary

Kubernetes clusters are increasingly deployed in multi-zone environments but
unfortunately network routing has not caught up with that. This KEP proposes an
automatic topology aware hinting mechanism that would provide a way for
EndpointSlice producers to indicate where consumers should use specific
endpoints. Even in scenarios where endpoints are not balanced evenly across
zones, EndpointSlice producers could use these hints to allocate endpoints
from zones with extra endpoints to zones with insufficient endpoints.

This would enable EndpointSlice consumers such as Kube-Proxy to implement simple
topology aware routing. This proposal is currently focused on topology aware
routing at zone level but could be expanded to include region.

In the short term, this is taking the place of two closely related KEPs that
were never implemented. These KEPs relate to EndpointSlice subsetting and are
still relevant, just deferred to a later point in time. This
[doc](https://docs.google.com/document/d/1ZzUoFY1SrdjVefl7gVOJZJLt1I1LHttw8pcX95nlgMY/edit)
has more info on this transition.

## Motivation

Kubernetes clusters are increasingly deployed in multi-zone environments.
Network traffic is routed randomly to any endpoint matching a Service. Some
users might want the traffic to stay in the same zone for the following
reasons:
- Cost savings: Keeping traffic within a zone can limit cross-zone networking
  costs.
- Performance: Traffic within a zone usually has less latency and bandwidth
  constraints, having a better performance than traffic leaving the zone.

In this KEP we are going to focus on avoiding cross-zone traffic when in-zone
endpoints would suffice. We're attempting to provide a simple and more automatic
approach to topology aware routing. This API will still allow users to indicate
that they prefer to keep traffic in the same zone if there's sufficient
capacity. With this approach users won't have to configure anything by default
for most use cases.

### Goals
- Provide a simple way for users to indicate their preference for keeping
  traffic in zone.
- Use the standard topology label `topology.kubernetes.io/zone` to derive the
  zones of nodes and endpoints.
- Use EndpointSlice hints as the primary mechanism for topology aware routing.
- Minimize churn of EndpointSlices while doing topology aware distribution.
- Minimize the number of new EndpointSlices required.
- Provide a simple API that requires minimal configuration for most users.

### Non-Goals
- Real-time distribution rebalancing based on traffic load or distribution
  feedback or metrics.
- Multi-cluster topology aware routing (this same pattern may be useful there
  though).
- Region based topology aware routing (this may come later).
- Ensuring that Pods are distributed evenly across zones.

## Proposal

When this feature is enabled, the EndpointSlice controller will be updated to
provide hints for each endpoint. These hints will initially be limited to a
single zone per-endpoint. Kube-Proxy will then use these hints to filter the
endpoints they should route to.

For example, for a Service with 3 endpoints, the EndpointSlice controller may
create an EndpointSlice with endpoints that look like this:

```
- addresses: ["10.1.2.3"]
  zone: "zone-a"
  hints:
    zone: "zone-a"
- addresses: ["10.1.2.4"]
  zone: "zone-b"
  hints:
    zone: "zone-b"
- addresses: ["10.1.2.5"]
  zone: "zone-a"
  hints:
    zone: "zone-c"
```

In the above example, 2 endpoints are in zone-a and 1 endpoint is in zone-b. The
hints help ensure that each zone will have a single endpoint to consume by
adding a hint to the third endpoint that it should be consumed by "zone-c".

This functionality will be enabled by a `TopologyAwareHints` feature gate along
with a new Service annotation.

### Risks and Mitigations

- In a scenario where all traffic originates from a single zone there is a
  chance that endpoints in that zone will be overloaded while endpoints in other
  zones receive little to no traffic. Without some sort of feedback (out of
  scope) this will not self-rectify.
- Autoscaling will not behave well if only a single zone is receiving large
  amounts of traffic. This could potentially be mitigated by separating
  deployments and HPAs per zone.
- Services with ExternalTrafficPolicy=local will need special treatment here.
  This approach could result in a situation where an endpoint on a Node is
  delivered to a separate underprovisioned zone. The simplest approach would be
  to disable this functionality altogether.
- When this feature is transitioning between enabled and disabled states, there
  will be a brief point in time where only some EndpointSlices have hints. That
  could temporarily result in traffic being routed to a small subset of
  endpoints. To avoid this, we only filter out endpoints that have a hint set
  to a different zone. If a hint is not set for an endpoint, it will be included
  by all instances of kube-proxy.

## Design Details

### Assumptions

- Incoming traffic is proportional to the number of allocatable CPU cores in a
  zone. Although this is an imperfect metric, it is the best available way of
  predicting how much traffic will be received in a zone. If we are unable to
  derive the number of allocatable cores in a zone we will fall back to the
  number of nodes in that zone.
- Service capacity is proportional to the number of endpoints in a zone. This
  assumes that each endpoint has equivalent capacity. Although this is not
  always true, it usually is. We can explore ways to deal with variable capacity
  endpoints in the future.

### Identifying Zones

The EndpointSlice controller reads the standard `topology.kubernetes.io/zone`
label on Nodes to determine which zone a Pod is running in. Kube-Proxy would be
updated to read the same information to identify which zone it is running in.

### Excluding Control Plane Nodes

Any Nodes with the following labels (set to any value) will be excluded when
calculating allocatable cores in a zone:

* `node-role.kubernetes.io/control-plane`
* `node-role.kubernetes.io/master`

### Configuration

A new `service.kubernetes.io/topology-aware-routing` annotation  can be used to
enable or disable Topology Aware Routing (and by extension, hints) for a
Service. This may be set to "Auto" or "Disabled". Any other value is treated as
"Disabled".

The previous `service.kubernetes.io/topology-aware-hints` annotation will
continue to be supported as a means of configuring this feature.

#### Interoperability

Topology hints will be ignored if the TopologyKeys field has at least one entry.
This field is deprecated and will be removed soon.

Both ExternalTrafficPolicy and InternalTrafficPolicy will be given precedence
over topology aware routing. For example, if `ExternalTrafficPolicy=Local` and
topology was enabled, external traffic would be routed using the
ExternalTrafficPolicy configuration while internal traffic would be routed with
topology.

#### Feature Gate

This functionality will be guarded by the `TopologyAwareHints` feature gate.
This gate also interacts with 2 other feature gates:
- It is dependent on the `ServiceTrafficPolicy` feature gate.
- It is not compatible with the deprecated `ServiceTopology` feature gate.

### API

A new `EndpointHints` struct would be added to the `EndpointSlice.Endpoint`
struct:

```go
type Endpoint struct {
  ...
  // hints contains information associated with how an endpoint should be
  // consumed.
  // +optional
  Hints EndpointHints `json:"hints,omitempty" protobuf:"bytes,7,opt,name=hints"`
}
```

```go
// EndpointHints provides hints describing how an endpoint should be consumed.
type EndpointHints struct {
  // forZones indicates the zone(s) this endpoint should be consumed by to
  // enable topology aware routing.
  forZones []ForZone `json:"forZone,omitempty" protobuf:"bytes,1,name=forZones"`
}
```

```go
// ForZone provides information about which zones should consume this endpoint.
type ForZone struct {
  // name represents the name of the zone.
  name string `json:"name" protobuf:"bytes,1,name=name"`
}
```

#### Future API Expansion
This approach would allow for future API expansion that enabled specifying
multiple zones per endpoint with weights. That level of complexity may never be
necessary, but it will be possible. For example:

```yaml
hints:
  forZones:
  - name: example-1a
    weight: 50
  - name: example-2a
    weight: 50
```

Additionally we could easily expand this API to include support for region
hints. Although it is unclear if either expansion will be necessary, the API is
designed in a way to make expansions straightforward.

### Kube-Proxy

When the `TopologyAwareHints` feature gate is enabled, Kube-Proxy will be
updated to filter endpoints based on topology hints when the following
conditions are true:

- Kube-Proxy is able to determine the zone it is running within (likely based
  on node labels).
- The annotation is set to `Auto`.
- At least one endpoint for the Service has a hint pointing to the zone
  Kube-Proxy is running within.
- All endpoints for the Service have zone hints.

When the above conditions are true, kube-proxy will only route traffic to
endpoints with a hint referring to the zone Kube-Proxy is running within.

This means that if any endpoints for a Service do not have a hint, kube-proxy
will ignore all hints. This is to provide safer transitions between enabled
and disabled states. Without this fallback, endpoints could easily get
overloaded as hints were being added or removed from some EndpointSlices but
had not yet propagated to all of them.

### EndpointSlice Controller

When the `TopologyAwareHints` feature gate is enabled and the annotation is set
to `Auto` for a Service, the EndpointSlice controller will add hints to
EndpointSlices. These hints will indicate where an endpoint should be consumed
by proxy implementations to enable topology aware routing.

The EndpointSlice controller will determine how many endpoints should be
available for each zone based on the proportion of CPU cores in each zone. If
it is not possible to determine the number CPU cores, 1 core per node will be
assumed for calculations.

#### Example

zone-a: 20 CPU cores
zone-b: 16 CPU cores
zone-c: 14 CPU cores

In this scenario, the following proportion of endpoints would be allocated for
each Service:

zone-a: 40%
zone-b: 32%
zone-c: 28%

When allocating endpoints to meet this distribution, keeping endpoints in the
same zone will be prioritized. When same-zone endpoints are exhausted, endpoints
will be taken from zones that have excess capacity.

#### Overload

Overload is a key concept for this proposal. This occurs when there are less
endpoints for a zone than a perfect distribution would result in. For example,
in a 3-zone cluster where each zone has an equivalent size, an EndpointSlice for
a 4 endpoint service would not receive any zone hints. The expected number of
endpoints per zone would be 1.33, and 2 of the 3 zones would only have 1
endpoint allocated. This means that endpoints for these zones would be likely to
receive 33% more traffic than a perfectly balanced scenario. In this case, the
"Overload" for those zones would be 33%.

Overload Threshold represents the maximum acceptable overload for this algorithm
before changes are required. If the overload threshold is reached, the
controller will attempt to redistribute endpoints to get below this threshold.
If this is impossible, hints will be removed from the endpoints.

As a starting point, an Overload Threshold of 30% will be used. Hints will not
be added for a Service unless the expected initial overload is below 20%. This
difference exists to prevent flapping between approaches.

#### Handling Node Updates

This approach results in a new potential reason to update EndpointSlices. As
nodes are added or removed, the proportion of endpoints that should be allocated
to each zone will change. This will be especially common in autoscaling
scenarios.

To mitigate the number of changes resulting from these events, EndpointSlices
will only be updated if a Node addition or removal results in a transition above
or below the overload threshold. For example, syncs would be triggered in either
of the following scenarios:

1. A deleted Node results in a Service exceeding the overload threshold.
2. A new Node results in a Service that is able to achieve an endpoint
   distribution below 20% for the first time.

### Future Expansion

In the future we may expand this functionality if needed. This could include:

- A new `RequireZone` algorithm that would keep endpoints in EndpointSlices for
  the same zone they are in.
- A new option to specify a minimum threshold for the `Auto` (PreferZone)
  approach.
- Support for region based hints.

### Test Plan

[x] I/we understand the owners of the involved components may require updates to
existing tests to make this code solid enough prior to committing the changes necessary
to implement this enhancement.

#### Unit tests

- `k8s.io/pkg/controller/endpointslice`: `2022-10-05` - `73.1`
- `k8s.io/pkg/controller/endpointslice/topologycache`: `2022-10-05` - `75.4`

##### Controller Unit Tests
| Test Description | Expected Result |
| :--- | :--- |
| Feature On, 2+ zones | Hints set |
| Feature Off, 2+ zones | No hints |
| Feature On, 1 zone | No hints set |
| Feature On, ExternalTrafficPolicy == 'Local', 2+ zones | No hints |
| 2 endpoints, 3 zones | No hints |
| 3 endpoints, 3 zones | Hints set |
| 4 endpoints, 3 zones | No hints |
| 4 endpoints, 2 zones | Hints set |
| 4 endpoints all from 1 zone, 2 zones | Hints set |
| 4 endpoints, 3 zones, 1 zone with 2x cores | Hints set |
| 400 endpoints, 4 zones with slightly different cores | Hints set |
| Node removal that does not trigger threshold transition | No EndpointSlice changes |
| Node removal that triggers threshold transition | EndpointSlice updates |
| Node without way to determine cores | All Nodes treated equally |
| Endpoint additions that require redistribution | Hints updated |
| Endpoint removals that require redistribution | Hints updated |

##### Kube-Proxy Unit Tests
| Test Description | Expected Result |
| :--- | :--- |
| Feature On, hints matching zone | Endpoints filtered |
| Feature On, ExternalTrafficPolicy == 'Local', hints matching zone | Endpoints not filtered |
| Feature Off, hints matching zone | Endpoints not filtered |
| Feature On, no hints matching zone | Endpoints not filtered |

#### Integration tests

N/A

#### e2e tests

This feature has e2e test coverage with the ["Topology Hints"
test](https://github.com/kubernetes/kubernetes/blob/fbb6ccc0c62d2431dc3a18a4130d7fbae5c05acd/test/e2e/network/topology_hints.go#L43).
This is currently limited to a periodic run due to the nature of requiring a
multizone cluster to run. It has been [remarkably
stable](https://testgrid.k8s.io/sig-network-kind#sig-network-kind,%20multizone)
with 100% green runs.

As a prerequisite for GA, we will ensure that this test runs as a presubmit
if any code changes in kube-proxy or the EndpointSlice controller.

### Observability
We can reuse some of the metrics of EndpointSlice Controller that we already
have in the current version to observe the changes of endpoints (addition,
deletion and update). Meanwhile we can add more metrics to have a glimpse of
different approaches.

- `endpoint_slice_controller/endpointslices_changed_per_sync`
- `endpoint_slice_controller/syncs`

```
const SubSystem = "endpoint_slice_controller"

// This metric observes churn of EndpointSlices per sync
EPSChangedPerSync = metrics.NewHistogramVec(
  &metrics.HistogramOpts{
	  Subsystem: Subsystem,
    Name: "endpointslices_changed_per_sync",
    Help: "Number of EndpointSlices be changed on each Service sync",
  },
  []string{"approach"}, // either "random" or "auto"
)

// EndpointSliceSyncs tracks the number of sync operations the controller runs along with their result.
EndpointSliceSyncs = metrics.NewCounterVec(
  &metrics.CounterOpts{
    Subsystem:      EndpointSliceSubsystem,
    Name:           "syncs",
    Help:           "Number of EndpointSlice syncs",
    StabilityLevel: metrics.ALPHA,
  },
  []string{"result"}, // either "success" or "failure"
)

```

### Events
A common point of frustration among initial users of this feature was how
difficult it was to tell if this feature was enabled and working as intended.
Due to the nature of this design, even when a user opts in to the `Auto` mode
that is no guarantee that the controller logic will determine that there are
a sufficient number of endpoints to allocate them proportionally to each zone
in the cluster.

To make this feature easier to understand and use, the EndpointSlice controller
will publish events for a Service to describe if the feature has been enabled,
and if not, why not.

#### Logic

The EndpointSlice controller will track the known state of this feature for
each Service. When that state or the reason for it changes, the EndpointSlice
controller will publish a new Event to reflect the updated status of this
feature.

#### Sample Events

| Type | Reason | Message | 
|-|-|-|
| Normal | TopologyAwareRoutingEnabled | Topology Aware Routing has been enabled |
| Normal | TopologyAwareRoutingDisabled | Topology Aware Routing configuration was removed |
| Warning | TopologyAwareRoutingDisabled | Insufficient number of Endpoints (n), impossible to safely allocate proportionally |
| Warning | TopologyAwareRoutingDisabled | 1 or more Endpoints do not have a Zone specified |
| Warning | TopologyAwareRoutingDisabled | 1 or more Nodes do not have allocatable CPU specified |
| Warning | TopologyAwareRoutingDisabled | Nodes only ready in 1 zone |

#### Documentation

The Topology Aware Hints documentation will be updated to describe the reason
each of these events may have been triggered, along with steps that can be taken
to recover from that state.

#### Limitations

Although the events described above should dramatically simplify the use of this
feature, there is a tiny edge case that will not be covered. If any
EndpointSlices for a Service do not include Hints, Kube-Proxy will not implement
this feature. This would happen if a user created custom EndpointSlices _and_
enabled Topology Aware Hints _and_ failed to set Hints on their custom
EndpointSlices. This seems very unlikely, but is mentioned here for the sake of
completeness.

### Graduation Criteria
**Alpha:**
- Basic functionality covered with unit tests described above.

**Beta:**
- Tests expanded to include e2e coverage described above.

**GA:**
- Feedback from real world usage shows that feature is working as intended
- Events are triggered on each Service to provide users with clear information
  on when the feature transitioned between enabled and disabled states.
- Test coverage in EndpointSlice strategy to ensure that the Hints field is
  dropped when the feature gate is not enabled.
- Test coverage in EndpointSlice controller for the transition from enabled to
  disabled.
- Ensure that existing Topology Hints e2e test runs as a presubmit if any code
  changes in kube-proxy or the EndpointSlice controller.
- Topology Hints e2e tests will graduate to conformance tests.
- Autoscaling and Scheduling SIGs have a plan to provide zone aware autoscaling
  (and scheduling) that allows users to proportionally distribute endpoints
  across zones.

### Version Skew Strategy
This KEP requires updates to both the EndpointSlice Controller and kube-proxy.
Thus there could be two potential version skew scenarios:
1. EndpointSlice Controller falls back to current behavior that does not
   support labeling EndpointSlices. In this case, kube-proxy will still work
   because EndpointSlices will not include topology hints.
2. Kube-Proxy falls back to current behavior that does not support topology
   hints in EndpointSlices. In this case, kube-proxy will continue to consume
   all endpoints. This will not be an issue, it simply won't be taking advantage
   of the new controller functionality.

Each scenario described above will end up behaving as if this feature is not
enabled even if the annotation has been set on the Service.

## Production Readiness Review Questionnaire

### Feature Enablement and Rollback

* **How can this feature be enabled / disabled in a live cluster?**
  - [x] Feature gate (also fill in values in `kep.yaml`)
    - Feature gate name: TopologyAwareHints
    - Components depending on the feature gate:
      - kube-controller-manager
      - kube-proxy

* **Does enabling the feature change any default behavior?**
  No.

* **Can the feature be disabled once it has been enabled (i.e. can we roll back
  the enablement)?**
  Yes. It can easily be disabled universally by turning off the feature gate or
  setting the annotation to some other value for a Service.

* **What happens if we reenable the feature if it was previously rolled back?**
  EndpointSlices hints will be added again resulting in changes to existing
  EndpointSlices for Services that have this feature enabled.

* **Are there any tests for feature enablement/disablement?**
  Per Service enablement and disablement is covered in depth by unit tests. As a
  prerequisite for graduation to GA, we will also add the following:

  - Test coverage in EndpointSlice strategy to ensure that the Hints field is
    dropped when the feature gate is not enabled.
  - Test coverage in EndpointSlice controller for the transition from enabled to
    disabled.

### Rollout, Upgrade and Rollback Planning

* **How can a rollout fail? Can it impact already running workloads?**
  The biggest risk here is that EndpointSlices may be created with hints some
  but not all zones. This will be covered by kube-proxy falling back to all
  endpoints if none have hints.

* **What specific metrics should inform a rollback?**
  If the proportion of `endpoint_slice_controller/syncs` with a "failure" result
  is greater than 10%, a rollback may be considered. It is worth noting that
  other issues can cause sync failures such as an out of date informer cache.
  The key indicator should be a significantly elevated error rate when compared
  with before the feature was enabled.

* **Were upgrade and rollback tested? Was the upgrade->downgrade->upgrade path tested?**
  Per-Service enablement/disablement is covered in depth and feature gate
  enablement and disablement will be covered before the feature graduates to GA.
  In addition, manual testing covering combinations of
  upgrade->downgrade->upgrade cycles will be completed prior to GA graduation.

* **Is the rollout accompanied by any deprecations and/or removals of features,
  APIs, fields of API types, flags, etc.?**
  Yes, this represents a replacement to the approach tracked with KEP 536. This
  KEP included an alpha implementation but did not graduate beyond that.

### Monitoring Requirements

* **How can an operator determine if the feature is in use by workloads?**
  If the `endpointslices_changed_per_sync` metric has a non-zero value for the
  `auto` approach, this feature is in use.

* **What are the SLIs (Service Level Indicators) an operator can use to
  determine the health of the service?**
  - [x] Metrics
    - Metric name: `endpoint_slice_controller/syncs`
    - [Optional] Aggregation method: Counter
    - Components exposing the metric: EndpointSlice Controller
    - The relative failure rate over time can be used to track the health of
      this controller.

* **What are the reasonable SLOs (Service Level Objectives) for the above SLIs?**
  As a starting point, it is likely reasonable for the EndpointSlice controller
  to experience up to a 10% sync failure rate. This is largely related to it
  trying to update stale EndpointSlices. When we are able to find a solution for
  that issue the expected sync failure rate should be significantly lower. This
  specific problem is most notable for large Services that have rapidly updating
  endpoints.

* **Are there any missing metrics that would be useful to have to improve
  observability of this feature?**
  None that I can think of.

### Dependencies

* **Does this feature depend on any specific services running in the cluster?**
  No new dependencies.

### Scalability

* **Will enabling / using this feature result in any new API calls?**
  Kube-Proxy will include a Node informer when this feature is enabled. This is
  also the case for a couple other Kube-Proxy features, including the previous
  `ServiceTopology` feature gate. This would also require a watch that was
  covering the node the instance is running on. This may result in some
  additional calls to the EndpointSlice API, but expect the increase to be
  minimal.

  The EndpointSlice controller will begin publishing Events for each Service
  that has opted in to this feature when this transitions between enablement
  states.

* **Will enabling / using this feature result in introducing new API types?**
  No.

* **Will enabling / using this feature result in any new calls to the cloud
  provider?**
  No.

* **Will enabling / using this feature result in increasing size or count of the
  existing API objects?**
  Yes, a new EndpointHints field will be added to the EndpointSlice API. This
  could add 20 bytes for each endpoint.

* **Will enabling / using this feature result in increasing time taken by any
  operations covered by [existing SLIs/SLOs]?**
  Although the EndpointSlice controller may take slightly longer to create
  EndpointSlices, kube-proxy performance should also be slightly improved. I do
  not anticipate any impact on existing SLIs or SLOs.

* **Will enabling / using this feature result in non-negligible increase of
  resource usage (CPU, RAM, disk, IO, ...) in any components?**
  This could result in increased CPU utilization for kube-controller-manager
  (specifically  the EndpointSlice controller). Profiling will be performed to
  ensure that this increase is minimal.

### Troubleshooting

* **How does this feature react if the API server and/or etcd is unavailable?**
  The EndpointSlice controller will stop functioning.

* **What are other known failure modes?**
  - The API server is unavailable. This is not specific to this controller and
    detections and mitigations are likely already widely covered.

* **What steps should be taken if SLOs are not being met to determine the problem?**
  This feature should be disabled. It is easy to leave this enabled for a single
  Service for debugging, but if SLOs are not being met the fastest solution is
  likely to disable this feature for any critical Services.

[supported limits]: https://git.k8s.io/community//sig-scalability/configs-and-limits/thresholds.md
[existing SLIs/SLOs]: https://git.k8s.io/community/sig-scalability/slos/slos.md#kubernetes-slisslos

## Implementation History

- KEP Merged: February 2021
- Alpha release: Kubernetes 1.21
- Beta Release: Kubernetes 1.23[^1]
- Feature Gate on-by default, feature available by default: 1.24

[^1]: This was intended to also flip the feature gate to enabled by default, but
    unfortunately that part was missed in 1.23. See
    [#108747](https://github.com/kubernetes/kubernetes/pull/108747) for more
    information.

## Drawbacks
1. Increased complexity in EndpointSlice controller
2. No immediate plans to support region

## Alternatives
1. Conduct topology aware routing at node level with specified topology keys,
   refer to the previous [Topology Aware Routing
   KEP](https://github.com/kubernetes/enhancements/blob/master/keps/sig-network/20181024-service-topology.md).
   As drawbacks described above, we could do some improvement i.e. fix the
   topology keys. But it still requires API and controller additions which
   introduces more complexity meanwhile cannot offer an easy policy decision at
   service level.
2. Implement this proposal with EndpointSlice subsetting. This was the original
   plan here but it resulted in too many compromises on both sides. We ended up
   with weaker approaches for subsetting and topology aware routing than if we
   separated them.

