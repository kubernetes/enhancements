---
title: Adding AppProtocol to Services and Endpoints
authors:
  - "@robscott"
owning-sig: sig-network
reviewers:
  - "@thockin"
  - "@dcbw"
approvers:
  - "@thockin"
  - "@dcbw"
creation-date: "2019-12-27"
last-updated: "2019-12-27"
status: implementable
see-also:
  - "/keps/sig-network/20190603-EndpointSlice-API.md"
  - "https://github.com/kubernetes/kubernetes/issues/40244"
---

# Adding AppProtocol to Services and Endpoints

## Table of Contents

<!-- toc -->
- [Summary](#summary)
- [Motivation](#motivation)
  - [Goals](#goals)
- [Proposal](#proposal)
    - [Services:](#services)
    - [Endpoints:](#endpoints)
  - [Risks and Mitigations](#risks-and-mitigations)
  - [Graduation Criteria](#graduation-criteria)
  - [Test plan](#test-plan)
<!-- /toc -->

## Summary

Kubernetes does not have a standardized way of representing application
protocols. When a protocol is specified, it must be one of TCP, UDP, or SCTP.
With the EndpointSlice beta release in 1.17, a concept of AppProtocol was added
that would allow application protocols to be specified for each port. This KEP
proposes adding support for that same attribute to Services and Endpoints.

## Motivation

The lack of direct support for specifying application protocols for ports has
led to widespread use of annotations, providing a poor user experience and
general frustration (https://github.com/kubernetes/kubernetes/issues/40244).
Unfortunately annotations are cloud specific and simply can't provide the ease
of use of a built in attribute like `AppProtocol`. Since application protocols
are specific to each port specified on a Service or Endpoints resource, it makes
sense to have a way to specify it at that level.

### Goals

Add AppProtocol field to Ports in Services and Endpoints.

## Proposal

In both Endpoints and Services, a new `AppProtocol` field would be added. In
both cases, constraints validation would directly mirror what already exists
with EndpointSlices.

#### Services:
```go
// ServicePort represents the port on which the service is exposed
type ServicePort struct {
    ...
    // The application protocol for this port.
    // This field follows standard Kubernetes label syntax.
    // Un-prefixed names are reserved for IANA standard service names (as per
    // RFC-6335 and http://www.iana.org/assignments/service-names).
    // Non-standard protocols should use prefixed names such as
    // mycompany.com/my-custom-protocol.
    // +optional
    AppProtocol *string
}
```

#### Endpoints:
```go
// EndpointPort is a tuple that describes a single port.
type EndpointPort struct {
    ...
    // The application protocol for this port.
    // This field follows standard Kubernetes label syntax.
    // Un-prefixed names are reserved for IANA standard service names (as per
    // RFC-6335 and http://www.iana.org/assignments/service-names).
    // Non-standard protocols should use prefixed names such as
    // mycompany.com/my-custom-protocol.
    // +optional
    AppProtocol *string
}
```

### Risks and Mitigations

It may take some time for cloud providers and other consumers of these APIs to
support this attribute. To help with this, we will work to communicate this
change well in advance of release so it can be well supported initially.

### Graduation Criteria

This adds a new optional attribute to 2 existing stable APIs. There is no need
for feature gating or a graduation process, this will be added to the existing
API versions.

### Test plan

This will replicate the existing validation tests for the AppProtocol field that
already exists on EndpointSlice. Additionally, it will add tests that ensure
that both the Endpoints and EndpointSlice controllers appropriately set the
AppProtocol field on Endpoints and EndpointSlices when it is set on the
corresponding Service.
