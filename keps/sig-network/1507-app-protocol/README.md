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
    - [Proposed Roadmap](#proposed-roadmap)
    - [Graduation Criteria](#graduation-criteria)
- [Alpha -&gt; Beta](#alpha---beta)
- [Beta -&gt; GA](#beta---ga)
    - [Test plan](#test-plan)
  - [Production Readiness Review Questionnaire](#production-readiness-review-questionnaire)
    - [Feature Enablement and Rollback](#feature-enablement-and-rollback)
    - [Rollout, Upgrade and Rollback Planning](#rollout-upgrade-and-rollback-planning)
    - [Monitoring Requirements](#monitoring-requirements)
    - [Dependencies](#dependencies)
    - [Scalability](#scalability)
    - [Troubleshooting](#troubleshooting)
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

### Proposed Roadmap
**Kubernetes 1.18**: New field is added but gated behind new alpha
`ServiceAppProtocol` feature gate.
**Kubernetes 1.19**: `ServiceAppProtocol` feature gate graduates to beta and is
enabled by default.
**Kubernetes 1.20**: `ServiceAppProtocol` feature gate graduates to GA.
**Kubernetes 1.21**: `ServiceAppProtocol` feature gate is removed.

### Graduation Criteria

This adds a new optional attribute to 2 existing stable APIs. This will follow
the traditional approach for adding new fields initially guarded by a feature
gate.

# Alpha -> Beta
- `ServiceAppProtocol` has been supported for at least 1 minor release.
- `ServiceAppProtocol` feature gate is enabled by default.

# Beta -> GA
- `ServiceAppProtocol` has been enabled by default for at least 1 minor release.

### Test plan

This will replicate the existing validation tests for the AppProtocol field that
already exists on EndpointSlice. Additionally, it will add tests that ensure
that both the Endpoints and EndpointSlice controllers appropriately set the
AppProtocol field on Endpoints and EndpointSlices when it is set on the
corresponding Service.

## Production Readiness Review Questionnaire

### Feature Enablement and Rollback

* **How can this feature be enabled / disabled in a live cluster?**
  This was previously enabled with the `ServiceAppProtocol` feature gate. That
  will be removed in Kubernetes 1.21.

* **Does enabling the feature change any default behavior?**
  No.

* **Can the feature be disabled once it has been enabled (i.e. can we roll back
  the enablement)?**
  Not anymore.

* **What happens if we reenable the feature if it was previously rolled back?**
  N/A.

* **Are there any tests for feature enablement/disablement?**
  N/A.

### Rollout, Upgrade and Rollback Planning

* **How can a rollout fail? Can it impact already running workloads?**
  If the `ServiceAppProtocol` gate is manually enabled on Kubernetes components
  it will no longer be recognized in Kubernetes 1.21. Users should stop using
  this feature gate.

* **What specific metrics should inform a rollback?**
  N/A.

* **Were upgrade and rollback tested? Was the upgrade->downgrade->upgrade path tested?**
  N/A.

* **Is the rollout accompanied by any deprecations and/or removals of features,
  APIs, fields of API types, flags, etc.?**
  The v1.21 rollout will include the removal of the `ServiceAppProtcol` feature
  gate.

### Monitoring Requirements

* **How can an operator determine if the feature is in use by workloads?**
  If this field is set on any Services, it may be used by applications that
  consume those Services. No core Kubernetes components consume this field.

* **What are the SLIs (Service Level Indicators) an operator can use to
  determine the health of the service?**
  N/A.

* **What are the reasonable SLOs (Service Level Objectives) for the above SLIs?**
  N/A.

* **Are there any missing metrics that would be useful to have to improve
  observability of this feature?**
  No.

### Dependencies

* **Does this feature depend on any specific services running in the cluster?**
  No.


### Scalability

* **Will enabling / using this feature result in any new API calls?**
  No.

* **Will enabling / using this feature result in introducing new API types?**
  No.

* **Will enabling / using this feature result in any new calls to the cloud
  provider?**
  No.

* **Will enabling / using this feature result in increasing size or count of the
  existing API objects?**
  Describe them, providing:
  - API type(s): Service
  - Estimated increase in size: 10B
  - Estimated amount of new objects: This field could be specified on each port
    in each Service in a cluster although that is unlikely.

* **Will enabling / using this feature result in increasing time taken by any
  operations covered by existing SLIs/SLOs?**
  No.

* **Will enabling / using this feature result in non-negligible increase of
  resource usage (CPU, RAM, disk, IO, ...) in any components?**
  No

### Troubleshooting

The Troubleshooting section currently serves the `Playbook` role. We may consider
splitting it into a dedicated `Playbook` document (potentially with some monitoring
details). For now, we leave it here.

* **How does this feature react if the API server and/or etcd is unavailable?**
  N/A

* **What are other known failure modes?**
  N/A

* **What steps should be taken if SLOs are not being met to determine the problem?**
  N/A
