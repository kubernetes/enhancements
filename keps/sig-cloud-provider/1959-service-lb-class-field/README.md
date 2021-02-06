# KEP-1959: Service Type=LoadBalancer Class Field

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
  - [Generic Annotation](#generic-annotation)
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
implementations in a cluster.

The new [Services APIs](https://github.com/kubernetes-sigs/service-apis) addresses this already
with the GatewayClass resource. However, until Gateway/GatewayClass APIs become mature, we should
support similar functionality for Services of Type=LoadBalancer. Introducing a new resource like
`ServiceClass` is probably not worthwhile given that there are new APIs already in development.
This KEP proposes a light-weight approach for Service Type=LoadBalancer by introducing a Service
field `service.spec.loadBalancerClass`.

## Motivation

The main use-case for this feature is being able to support multiple Service Type=LoadBalancer
implementations in a cluster, as different workloads may want to leverage different loadbalancer
providers based on efficiency, availability, cost and other factors.

For example, a cluster admin may want to use a public load balancer from a cloud provider
for workloads that must be assigned a publically routable address, but they may want to
enable a lower-cost solution for workloads that are only internally accessible.

### Goals

* allow users to opt-out of the Service Type=LoadBalancer implementation by the cloud provider.
* allow multiple implementations of Service Type=LoadBalancer in a cluster.
* prevent every cloud provider from implementing a custom "opt-out" annotation for their load balancer.

### Non-Goals

* performance improvements for Service Type=LoadBalancer.
* changing any other existing behaviors for Service Type=LoadBalancer aside from being able
to disabling it from the cloud provider.

## Proposal

This KEP proposes to add a new field `spec.loadBalancerClass` in Service which allows for
multiple implementations of Service Type=LoadBalancer in a cluster.

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

Many cloud providers today support an "opt-out" annotation for this behavior. The annotation is specific
to the cloud provider. Introduction of the `loadBalancerClass` field at this point would mean that
cloud providers need to start accounting for both existing annotations and the new field.

## Design Details

Introduce a new field to Service `spec.loadBalancerClass`.

If the field `spec.loadBalancerClass` is not set, the existing cloud provider will assume
ownership of the Service Type=LoadBalancer resource. This is required to not break existing clusters
that assume Service Type=LoadBalancer is always managed by the cloud provider.

Required API changes:
```go
// ServiceSpec describes the attributes that a user creates on a service.
type ServiceSpec struct {
	...
	...

	// loadBalancerClass is the name of the load balancer implementation this Service belongs to.
	// This field can only be set when the Service type is 'LoadBalancer'. If not set, the default load
	// balancer implementation is used, today this is typically done through the cloud provider integration,
	// but should apply for any default implementation. If set, it is assumed that a load balancer
	// implementation is watching for Services with a matching class name. Any default load balancer
	// implementation (e.g. cloud providers) should ignore Services that set this field.
	// +optional
	LoadBalancerClass string `json:"loadBalancerClass,omitempty"`
}
```

* `loadBalancerClass` will be immutable only when the Service type is `LoadBalancer`, this way existing and future implementations
do not have to worry about handling Services that change the class name. The class name is mutable and must be cleared when the
type changes.
* `loadBalancerClass` will be validated against label-style format.
* the `loadBalancerClass` field will be feature gated. The field will be dropped during API strategy unless
the feature gate is enabled.
* all external implementations of Service LoadBalancer using a non-empty class name should use the finalizer `service.kubernetes.io/load-balancer-cleanup`
to ensure proper garbage collection of external resources.

Required updates to service controller:
* if the class field is NOT set for a Service, allow the cloud provider to reconcile the load balancer.
* if the class annotation IS set for a Service, skip reconciliation of the Service from the cloud provider.

### Test Plan

Unit tests:
* test that service controller does not call the cloud provider if the class field is set.
* test API strategy to ensure the `loadBalancerClass` field is dropped unless the feature gate is enabled
or an existing Service has the field set.
* test API validation for immutability.

Integration tests:
* test that the class field is propoerly cleared/validated when the Service type changes to and from `LoadBalancer`.

E2E tests:
* test that creating a Service with an unknown class name results in no load balancer being created for a Service.

### Graduation Criteria

Alpha:
* the `loadBalancerClass` field is added to Service with an alpha feature gate.
* when enabled, service controller will ignore Service LBs with a non-empty class name.
* unit tests for service controller.
* unit tests for API strategy (drop disabled fields).

### Upgrade / Downgrade Strategy

* Usage of `loadBalancerClass` will be off by default during the alpha stage but can handle existing Services that
has the field set already. This ensures apiserver can handle the new field on downgrade.
* On upgrade, if the feature gate is enabled, there should be no changes since the default behavior has not changed
(service controller calls the cloud provider to reconcile load balancers).

### Version Skew Strategy

Since this feature will be alpha for at least 1 release, an n-1 kube-controller-manager or cloud-controller-manager should
handle enablement of this feature if a new apiserver enabled it.

## Implementation History

- the `Summary`, `Motivation`, `Proposal` and `Design Details` sections was merged, signaling SIG acceptance

## Drawbacks

* Added complexity to Service.
* In **most** clusters, a single Service Type=LoadBalancer implementation from the cloud provider is sufficient.

## Alternatives

### ServiceClass Resource

Instead of a field specifying the name of the implemmentation, the class name can reference the name of a class resource
similar to GatewayClass and IngressClass. This would enable more expressive configuration per load balancer implementation.

### Generic Annotation

A generic annotation can be used to store the class name. This is avoided since there would be no way to introduce
the annotation in a safe way and we can't enforce immutability for annotations.

### Provider-Specific Annotations

Instead of a generic Kubernetes annotation read by service controller, each cloud provider could implement
their own "skip this Service"-like logic with custom annotations. Given that many cloud providers have been
asking for this feature, a generic field used across all providers may be more beneficial.

## Infrastructure Needed (Optional)

<!--
Use this section if you need things from the project/SIG. Examples include a
new subproject, repos requested, or GitHub details. Listing these here allows a
SIG to get the process for these resources started right away.
-->
