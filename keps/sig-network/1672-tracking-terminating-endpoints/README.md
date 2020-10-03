# KEP-1672: Tracking Terminating Endpoints in the EndpointSlice API

<!-- toc -->
- [Release Signoff Checklist](#release-signoff-checklist)
- [Summary](#summary)
- [Motivation](#motivation)
  - [Goals](#goals)
  - [Non-Goals](#non-goals)
- [Proposal](#proposal)
  - [User Stories (optional)](#user-stories-optional)
    - [Story 1](#story-1)
  - [Notes/Constraints/Caveats (optional)](#notesconstraintscaveats-optional)
  - [Risks and Mitigations](#risks-and-mitigations)
- [Design Details](#design-details)
  - [Test Plan](#test-plan)
  - [Graduation Criteria](#graduation-criteria)
    - [Alpha](#alpha)
  - [Upgrade / Downgrade Strategy](#upgrade--downgrade-strategy)
  - [Version Skew Strategy](#version-skew-strategy)
- [Implementation History](#implementation-history)
- [Drawbacks](#drawbacks)
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

Today, terminating endpoints are considered "not ready" regardless of their actual readiness.
Before any work is done in improving how terminating endpoints are handled, there must be a way
to track whether an endpoint is terminating without having to watch the associated pods. This
KEP proposes a means to track the terminating state of an endpoint via the EndpointSlice API.
This would enable consumers of the API to make smarter decisions when it comes to handling
terminating endpoints (see KEP-1669 as an example).

## Motivation

### Goals

* Provide a mechanism to track whether an endpoint is terminating by only watching the EndpointSlice API.

### Non-Goals

* Consumption of the new API field is out of scope for this KEP but future KEPs will leverage
the work done here to improve graceful terminination of pods in certain scenarios (see issue [85643](https://github.com/kubernetes/kubernetes/issues/85643))

## Proposal

This KEP proposes to keep "terminating" pods in the set of endpoints in EndpointSlice with
additions to the API to indicate whether a given endpoint is terminating or not. If consumers
of the API (e.g. kube-proxy) are required to treat terminating endpoints differently, they
may do so by checking this condition.

The criteria for a ready endpoint (pod phase + readiness probe) will not change based on the
terminating state of pods, but consumers of the API may choose to prefer endpoints that are both ready and not terminating.

### User Stories (optional)

#### Story 1

A consumer of the EndpointSlice API (e.g. kube-proxy) may want to know which endpoints are
terminating without having to watch Pods directly for scalability reasons.

One example would be the IPVS proxier which should set the weight of an endpoint to 0
during termination and finally remove the real server when the endpoint is removed.
Without knowing when a pod is done terminating, the IPVS proxy makes a best-effort guess
at when the pod is terminated by looking at the connection tracking table.

### Notes/Constraints/Caveats (optional)

### Risks and Mitigations

Tracking the terminating state of endpoints poses some scalability concerns as each
terminating endpoint adds additional writes to the API. Today, a terminating pod
results in 1 write in Endpoints (removing the endpoint). With the proposed changes,
each terminating endpoint could result in at least 2 writes (ready -> terminating -> removed)
and possibly more depending on how many times readiness changes during termination.

## Design Details

To track whether an endpoint is terminating, a `terminating` field would be added as part of
the `EndpointCondition` type in the EndpointSlice API.

```go
// EndpointConditions represents the current condition of an endpoint.
type EndpointConditions struct {
    // ready indicates that this endpoint is prepared to receive traffic,
    // according to whatever system is managing the endpoint. A nil value
    // indicates an unknown state. In most cases consumers should interpret this
    // unknown state as ready.
    // +optional
    Ready *bool `json:"ready,omitempty" protobuf:"bytes,1,name=ready"`

    // terminating indicates if this endpoint is terminating. Consumers should assume a
    // nil value indicates the endpoint  is not terminating.
    // +optional
    Terminating *bool `json:"terminating,omitempty" protobuf:"bytes,2,name=terminating"`
}
```

NOTE: A nil value for `Terminating` indicates that the endpoint is not terminating.

Updates to endpointslice controller:
* include pods with a deletion timestamp in endpointslice
* any pod with a deletion timestamp will have condition.terminating = true
* allow endpoint ready condition to change during termination

### Test Plan

endpointslice controller unit tests:
* Unit tests will validate pods with a deletion timestamp are included with condition.teriminating = true
* Unit tests will validate that the ready condition can change for terminating endpoints

There will be no e2e tests since consumption of this new API is out-of-scope for this KEP.
Any future KEP that consumes this API should have e2e tests to ensure behavior for terminating
endpoints is correct.

### Graduation Criteria

#### Alpha

* EndpointSlice API includes `Terminating` condition.
* `Terminating` condition can only be set if feature gate `EndpointSliceTerminatingCondition` is enabled.
* Unit tests in endpointslice controller and API validation/strategy.

### Upgrade / Downgrade Strategy

Since this is an addition to the EndpointSlice API, the upgrade/downgrade strategy will follow that
of the [EndpointSlice API work](/keps/sig-network/20190603-endpointslices/README.md).

### Version Skew Strategy

Since this is an addition to the EndpointSlice API, the version skew strategy will follow that
of the [EndpointSlice API work](/keps/sig-network/20190603-endpointslices/README.md).

## Implementation History

- [x] 2020-04-23: KEP accepted as implementable for v1.19

## Drawbacks

There are some scalability draw backs as tracking terminating endpoints requires at least 1 additional write per endpoint.

