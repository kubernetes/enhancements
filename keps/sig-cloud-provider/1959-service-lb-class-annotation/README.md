# KEP-1959: Service Type=LoadBalancer Class Annotations

<!-- toc -->
- [Release Signoff Checklist](#release-signoff-checklist)
- [Summary](#summary)
- [Motivation](#motivation)
  - [Goals](#goals)
  - [Non-Goals](#non-goals)
- [Proposal](#proposal)
  - [User Stories (Optional)](#user-stories-optional)
    - [Story 1](#story-1)
    - [Story 2](#story-2)
  - [Risks and Mitigations](#risks-and-mitigations)
- [Design Details](#design-details)
  - [Test Plan](#test-plan)
  - [Graduation Criteria](#graduation-criteria)
  - [Upgrade / Downgrade Strategy](#upgrade--downgrade-strategy)
  - [Version Skew Strategy](#version-skew-strategy)
- [Implementation History](#implementation-history)
- [Drawbacks](#drawbacks)
- [Alternatives](#alternatives)
  - [ServiceClass Resource](#serviceclass-resource)
  - [Provider-Specific Annotations](#provider-specific-annotations)
- [Infrastructure Needed (Optional)](#infrastructure-needed-optional)
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

When Service Type=LoadBalancer is enabled by a Kubernetes cloud provider, it is a global
configuration that applies for all Service Type=LoadBalancer resources in a given cluster.
This becomes problematic if users want to leverage multiple Service Type=LoadBalancer
implementations in a given cluster.

The new [Services APIs](https://github.com/kubernetes-sigs/service-apis) addresses this already
with the GatewayClass resource. However, until Gateway/GatewayClass APIs become mature, we should
support similar functionality for Services of Type=LoadBalancer. Introducing a new resource like
`ServiceClass` is probably not worthwhile given that there are new APIs already in development.
This KEP proposes a light-weight approach for Service Type=LoadBalancer by introducing a Service
annotation `service.kubernetes.io/load-balancer-class`.

## Motivation

The main use-case for this feature is being able to support multiple Service Type=LoadBalancer
implementations in a cluster, as different workloads may want to leverage different loadbalancer
providers based on efficiency, availability, cost and other factors.

For example, a cluster admin may want to use a public load balancer from a cloud provider
for workloads that must be assigned a publically routable address, but they may want to
enable a lower-cost solution for workloads that are only internally accessible.

### Goals

* allow users to opt-out of the Service Type=LoadBalancer implementation by the cloud provider.
* allow multiple implementations of Service Type=LoadBalancer in a given cluster.

### Non-Goals

* performance improvements for Service Type=LoadBalancer.
* changing any other existing behaviors for Service Type=LoadBalancer aside from being able
to disabling it from the cloud provider.

## Proposal

This KEP proposes to add a new Service annotation `service.kubernetes.io/load-balancer-class`
that allows for multiple implementations of Service Type=LoadBalancer in a cluster.

### User Stories (Optional)

#### Story 1

As a cluster admin:
* I want to use my cloud provider's public load balancer service for applications that require
public ingress.
* I want to use my own load balancing solution for any applications that only talk internally
within my own network because I want to save costs.

#### Story 2

As an application developer:
* I MUST use a hardware-based loadbalancer for certain applications due to specific protocols
only available there.
* I want to use the cloud provider's default load balancer for any applications that do not
rely on protocols from hardware load balancers.

### Risks and Mitigations

Many Service Type=LoadBalancer implementations today support a lot of knobs via annotations already.
Introducing yet another annotation for Service Type=LoadBalancer is not ideal, but this is better than
every cloud provider supporting their own "skip this Service" annotation.

## Design Details

Introduce a new Service annotation `service.kubernetes.io/load-balancer-class`.

If the loadbalancer class annotation is not set, the existing cloud provider
will assume ownership of the Service Type=LoadBalancer resource. This is required
to not break existing clusters that assume Service Type=LoadBalancer is always
managed by the cloud provider.

Required updates to service controller:
* if the class annotation is NOT set for a Service, allow the cloud provider
to reconcile the load balancer.
* if the class annotation IS set for a Service, skip reconciliation of the Service
by the cloud provider.

### Test Plan

Unit tests:
* test that service controller does not call the cloud provider if the class annotation is set.
* the annotation `service.kubernetes.io/load-balancer-class` is not accepted when the feature gate `ServiceLoadBalancerClass` is disabled.

E2E tests:
* test that creating a Service with an unknown class annotation results in no load balancer being created for a Service.

### Graduation Criteria

N/A since we can't apply alpha/beta/GA criteria for annotations.

### Upgrade / Downgrade Strategy

On upgrade, use of this annotation will be allowed. On downgrade, service controller
may ignore existing Services with the annotation, leading to multiple implementations
trying to create load balancers. On downgrade, if the class annotation is used
and there are multiple implementations of Service Type=LoadBalancer, a user must ensure
there is only 1 implementation of Service Type=LoadBalancer in the cluster.

Though the downgrade scenario isn't ideal, it is assumed if that a cluster was upgraded to v1.20,
and already has multiple Service Type=LoadBalancer implementations enabled, it will likely not be
downgrading to v1.19 anytime soon.

### Version Skew Strategy

N/A since this only impacts one component.

## Implementation History

- the `Summary`, `Motivation`, `Proposal` and `Design Details` sections was merged, signaling SIG acceptance

## Drawbacks

* Annotations are a clunky way to implement "Class" semantics to Service Type=LoadBalancer.
* In **most** clusters, a single Service Type=LoadBalancer implementation from the cloud provider is sufficient.
* The potential risks during downgrade can cause outages if Service Status is updated by the wrong load balancer implementation.

## Alternatives

### ServiceClass Resource

Similar to GatewayClass and IngressClass, we could introduce a new resource so that multiple implementations of
Service Type=LoadBalancer can exist, however, a new resource just for Service Type=LoadBalancer seems unnecessary,
especially if GatewayClass will satisfy this use-case better in the near future.

### Provider-Specific Annotations

Instead of a generic Kubernetes annotation read by service controller, each cloud provider could implement
their own "skip this Service"-like logic with custom annotations. Given that many cloud providers have been
asking for this feature, a generic well-known annotation used across all providers may be more beneficial.

## Infrastructure Needed (Optional)

<!--
Use this section if you need things from the project/SIG. Examples include a
new subproject, repos requested, or GitHub details. Listing these here allows a
SIG to get the process for these resources started right away.
-->
