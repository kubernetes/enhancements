# KEP-2030: EndpointSlice Subsetting and Selection

<!-- toc -->
- [Release Signoff Checklist](#release-signoff-checklist)
- [Summary](#summary)
- [Motivation](#motivation)
  - [Goals](#goals)
  - [Non-Goals](#non-goals)
- [Proposal](#proposal)
  - [Risks and Mitigations](#risks-and-mitigations)
- [Design Details](#design-details)
  - [API Consumption](#api-consumption)
    - [Select EndpointSlices Without Labels](#select-endpointslices-without-labels)
    - [Select EndpointSlices With Matching Zone](#select-endpointslices-with-matching-zone)
    - [Select EndpointSlices With Matching Region](#select-endpointslices-with-matching-region)
    - [Kube-Proxy](#kube-proxy)
  - [Controller Implementation](#controller-implementation)
  - [Backwards Compatibility](#backwards-compatibility)
- [Test Plan](#test-plan)
  - [Unit Tests](#unit-tests)
- [Graduation Criteria](#graduation-criteria)
  - [Alpha Release](#alpha-release)
  - [Alpha -&gt; Beta Graduation](#alpha---beta-graduation)
- [Upgrade / Downgrade Strategy](#upgrade--downgrade-strategy)
- [Version Skew Strategy](#version-skew-strategy)
- [Implementation History](#implementation-history)
- [Drawbacks](#drawbacks)
- [Alternatives](#alternatives)
<!-- /toc -->

## Release Signoff Checklist

Items marked with (R) are required *prior to targeting to a milestone / release*.

- [ ] (R) Enhancement issue in release milestone, which links to KEP dir in [kubernetes/enhancements] (not the initial KEP PR)
- [ ] (R) KEP approvers have approved the KEP status as `implementable`
- [ ] (R) Design details are appropriately documented
- [ ] (R) Test plan is in place, giving consideration to SIG Architecture and SIG Testing input
- [ ] (R) Graduation criteria is in place
- [ ] (R) Production readiness review completed
- [ ] Production readiness review approved
- [ ] "Implementation History" section is up-to-date for milestone
- [ ] User-facing documentation has been created in [kubernetes/website], for publication to [kubernetes.io]
- [ ] Supporting documentation—e.g., additional design documents, links to mailing list discussions/SIG meetings, relevant PRs/issues, release notes

[kubernetes.io]: https://kubernetes.io/
[kubernetes/enhancements]: https://git.k8s.io/enhancements
[kubernetes/kubernetes]: https://git.k8s.io/kubernetes
[kubernetes/website]: https://git.k8s.io/website

## Summary

Segment EndpointSlices into logical chunks enabling API consumers like
kube-proxy to watch only a subset of all EndpointSlices. This provides a natural
starting point for topology aware routing and dramatically increases
scalability.

## Motivation

There has been some discussion that the Alpha Service Topology API requires the
user to do too much. Users generally want the same behavior for all services
where topology is concerned: avoid cross-zone traffic when in-zone endpoints are
available and have enough capacity. This problem is exacerbated by the
introduction of Multi-Cluster services where cross-region services become a
reality and some regions may be more desirable than others as failover
locations.

As clusters and services grow, we’re also seeing that the current proxy
implementations are stretched to their limits as all endpoints for the entire
cluster are tracked independently by each node.

The best user experience seems like it would be to have the platform - with a
first class controller and/or a provider specific implementation - intelligently
prioritize endpoints based on topology, capacity, and any other useful cost
metrics to aid in traffic shaping, or just to reduce the amount of global
resource tracking required by every node. However, we are currently missing a
way to allow endpoints to be targeted to specific subsets of nodes.

### Goals

- Provide the building blocks to allow EndpointSlices to target specific subsets
  of nodes.
- EndpointSlices subsetting will be fully backwards compatible for older
  consumers of the EndpointSlice API.
- Design is flexible enough for multiple implementations and experimentation.
- Minimal duplication of data.
- Room for future enhancements, for example weighted endpoints or Slices.

### Non-Goals

- Define how subsetting should be used.
- Design the controller responsible for subsetting endpoints.
- An API for telling controllers how a service should be subsetted.

Many of these are being tackled by the follow up [KEP
#2004](https://github.com/kubernetes/enhancements/issues/2004).

## Proposal

Two new topology based labels will be introduced for EndpointSlices to support
subsetting:

```
endpointslice.kubernetes.io/for-zone
endpointslice.kubernetes.io/for-region
```

In the future this pattern may be expanded to include other concepts or
topologies. A simple pattern like this will allow EndpointSlices to be delivered
to consumers in a specific zone or region.


### Risks and Mitigations

This approach does not allow a single EndpointSlice to target multiple zones or
regions. Any approach that enabled that would be significantly more complicated.
The initial proposal in [KEP
#2004](https://github.com/kubernetes/enhancements/issues/2004) suggests that
this won't be an issue. 

## Design Details

### API Consumption

EndpointSlice API consumers can use the following selectors to consume
EndpointSlices:

#### Select EndpointSlices Without Labels
This is required to support producers of EndpointSlices that don't set these
labels, including older versions of the EndpointSlice controller.

```
matchExpressions:
  - {key: endpointslice.kubernetes.io/for-zone, operator: DoesNotExist}
  - {key: endpointslice.kubernetes.io/for-region, operator: DoesNotExist}
```

#### Select EndpointSlices With Matching Zone
```
matchLabels:
  endpointslice.kubernetes.io/for-zone: example-zone
```

#### Select EndpointSlices With Matching Region
```
matchLabels:
  endpointslice.kubernetes.io/for-region: example-region
```

#### Kube-Proxy
When the `EndpointSliceSubsetting` feature gate is set to true on `Kube-Proxy`,
it will use these selectors to filter EndpointSlices.

### Controller Implementation
Although a controller implementation is out of scope for this KEP, it is worth
discussing what that might look like. For reference, [KEP
#2004](https://github.com/kubernetes/enhancements/issues/2004) discusses how this
could be implemented for the EndpointSlice controller. That proposal involves 3
potential approaches - Original, PreferZone, and RequireZone.

None of the proposed approaches would involve data duplication. Each
Pod/endpoint would continue to live in a single EndpointSlice. The reason they
might end up with more EndpointSlices would be less efficient packing. Here's
the number of EndpointSlices that would result based on the number of endpoints
a Service has in a 3 zone cluster. In each case, the numbers in parentheses
represent how many endpoints would exist in each slice.

| # endpoints | Original # slices | PreferZone # slices | RequireZone # slices |
|-|-|-|-|
| 6 | 1 (6) | 1 (6) | 3 (2) |
| 90 | 1 (90) | 3 (30) | 3 (30) |
| 270 | 3 (90) | 3 (90) | 3 (90) |

The RequireZone approach requires at least one EndpointSlice per zone per
Service. The PreferZone also has the same requirement unless the minimum
threshold has been met. Before that threshold, a single shared EndpointSlice (no
additional labels) is used. There's some padding involved here to make sure
we're not flapping back and forth between these states.

With this approach, EndpointSlices can be delivered everywhere (no additional
labels), or to a zone (for-zone), or to a region (for-region). None of the
proposed approaches involve a single Service having separate sets of
EndpointSlices for each use case. As defined by [KEP
1659](https://github.com/kubernetes/enhancements/tree/master/keps/sig-architecture/1659-standard-topology-labels),
"region" and "zone" are strictly hierarchical ("zones" are subsets of "regions")
and zone names are unique across regions.

### Backwards Compatibility
We don't need create EndpointSlices without labels for backwards compatibility,
we just need to ensure that consumers always support consuming EndpointSlices
without these labels. Even if we updated the EndpointSlice controller to
consistently label these EndpointSlices with `for-zone` or `for-region`, we
couldn't guarantee that other producers would.

There's nothing in any current consumer implementation that would break if
additional labels like `for-zone` or `for-region` were added to EndpointSlices.
All consumers will need to care about for this or the original approach is the
`kubernetes.io/service-name` label. If they want to support subsetting, they can
update their selectors as described in this KEP, but subsetting won't actually
break any existing functionality.

## Test Plan
This KEP is quite small in scope. The only new functionality being added will be
an adjustment to the EndpointSlices kube-proxy consumes when a feature gate is
enabled. We will need to add more test coverage for when this feature is enabled
or disabled.

### Unit Tests
* Ensure kube-proxy will continue to consume all EndpointSlices when this
  feature is disabled.
* Ensure EndpointSlices delivered to a specific zone will be consumed by
  kube-proxy running in the same zone when this feature is enabled.
* Ensure EndpointSlices delivered to a specific zone will not be consumed by
  kube-proxy running in a different zone when this feature is enabled.
* Ensure EndpointSlices delivered to a specific region will be consumed by
  kube-proxy running in the same region when this feature is enabled.
* Ensure EndpointSlices delivered to a specific region will not be consumed by
  kube-proxy running in a different region when this feature is enabled.

## Graduation Criteria

### Alpha Release

- Proposed labels are added as well known labels in Discovery API types.
- Implement new selectors in kube-proxy.
- Implement test plan.

### Alpha -> Beta Graduation

- EndpointSlice controller supports publishing EndpointSlices in subsets. (See
  [KEP 2004](https://github.com/kubernetes/enhancements/issues/2004) for more
  info).

## Upgrade / Downgrade Strategy

This functionality will be guarded by the `EndpointSliceSubsetting` feature gate
on kube-proxy. This will be fully backwards compatible and will only make a
difference in a cluster if EndpointSlices are being published with the labels
described in this KEP.

## Version Skew Strategy

This is designed with backwards compatibility in mind. Enabling this feature is
not reliant on any other feature being enabled in any other release. [KEP
#2004](https://github.com/kubernetes/enhancements/issues/2004) will be dependent
on this KEP though.

## Implementation History

September 2020: Initial Proposal Submitted

## Drawbacks

Although an optional feature, this adds more complexity to the consumption of
EndpointSlices for anyone that wants to support the feature.

## Alternatives

An alternative would be to use an approach that would allow delivery to multiple
zones. With labels, this would require including the zone name in the label key:

```
endpointslice.kubernetes.io/for-zone-a
endpointslice.kubernetes.io/for-zone-b
```

Unfortunately it would be much more difficult to build backwards compatible
selectors to consume these labels.