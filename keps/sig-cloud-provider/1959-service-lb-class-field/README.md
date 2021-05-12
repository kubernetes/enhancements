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
    - [Alpha:](#alpha)
    - [Beta:](#beta)
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
  - [ServiceClass Resource](#serviceclass-resource)
  - [Generic Annotation](#generic-annotation)
  - [Provider-Specific Annotations](#provider-specific-annotations)
- [Infrastructure Needed (Optional)](#infrastructure-needed-optional)
<!-- /toc -->

## Release Signoff Checklist

Items marked with (R) are required *prior to targeting to a milestone / release*.

- [X] (R) Enhancement issue in release milestone, which links to KEP dir in [kubernetes/enhancements] (not the initial KEP PR)
- [X] (R) KEP approvers have approved the KEP status as `implementable`
- [X] (R) Design details are appropriately documented
- [X] (R) Test plan is in place, giving consideration to SIG Architecture and SIG Testing input
- [X] (R) Graduation criteria is in place
- [X] (R) Production readiness review completed
- [X] (R) Production readiness review approved
- [ ] "Implementation History" section is up-to-date for milestone
- [X] (R) User-facing documentation has been created in [kubernetes/website], for publication to [kubernetes.io]
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

* `loadBalancerClass` will be immutable when the Service type is `LoadBalancer`, this way existing and future implementations
do not have to worry about handling Services that change the class name. The class name is mutable only when the type is not LoadBalancer and
must be cleared when the type changes.
* `loadBalancerClass` will be validated against label-style format.
* the `loadBalancerClass` field will be feature gated. The field will be dropped during API strategy unless
the feature gate is enabled.
* all external implementations of Service LoadBalancer using a non-empty class name should use the finalizer `service.kubernetes.io/load-balancer-cleanup`
to ensure proper garbage collection of external resources.

Required updates to service controller:
* if the class field is NOT set for a Service, allow the cloud provider to reconcile the load balancer.
* if the class field IS set for a Service, skip reconciliation of the Service from the cloud provider.

### Test Plan

Unit tests:
* test that service controller does not call the cloud provider if the class field is set.
* test API strategy to ensure the `loadBalancerClass` field is dropped unless the feature gate is enabled
or an existing Service has the field set.
* test API validation for immutability.

Integration tests:
* test that the class field is properly cleared/validated when the Service type changes to and from `LoadBalancer`.

E2E tests:
* test that creating a Service with an unknown class name results in no load balancer being created for a Service.

### Graduation Criteria

#### Alpha:

* the `loadBalancerClass` field is added to Service with an alpha feature gate.
* when enabled, service controller will ignore Service LBs with a non-empty class name.
* unit tests for service controller.
* unit tests for API strategy (drop disabled fields).

#### Beta:

* Feature gate is on by default.
* E2E tests checking that default load balancer implementation ignores LoadBalancer type of Services when `loadBalancerClass` set.

### Upgrade / Downgrade Strategy

* Usage of `loadBalancerClass` will be off by default during the alpha stage but can handle existing Services that
has the field set already. This ensures apiserver can handle the new field on downgrade.
* On upgrade, if the feature gate is enabled, there should be no changes since the default behavior has not changed
(service controller calls the cloud provider to reconcile load balancers).

### Version Skew Strategy

Since this feature will be alpha for at least 1 release, an n-1 kube-controller-manager or cloud-controller-manager should
handle enablement of this feature if a new apiserver enabled it.

## Production Readiness Review Questionnaire

### Feature Enablement and Rollback

_This section must be completed when targeting alpha to a release._

* **How can this feature be enabled / disabled in a live cluster?**
  - [X] Feature gate (also fill in values in `kep.yaml`)
    - Feature gate name: ServiceLoadBalancerClass
    - Components depending on the feature gate: kube-apiserver, kube-controller-manager
  - [ ] Other
    - Describe the mechanism:
    - Will enabling / disabling the feature require downtime of the control
      plane?
    - Will enabling / disabling the feature require downtime or reprovisioning
      of a node? (Do not assume `Dynamic Kubelet Config` feature is enabled).

* **Does enabling the feature change any default behavior?**
  No, the default service controller in Kubernetes will continue to watch and implement
  any Services with an empty class name. Behavior is only changed when the class name is set.

* **Can the feature be disabled once it has been enabled (i.e. can we roll back
  the enablement)?**
  Yes, the feature can be disabled, but any existing Services using the new field will
  continue to have the field set. External controllers watching a specific class name
  will continue to watch and reconcile those Services.

* **What happens if we reenable the feature if it was previously rolled back?**
  New Services can continue to use the field. Existing Services with the field always had
  the field set so no behavior is changed when the feature is re-enabled.


* **Are there any tests for feature enablement/disablement?**
  Yes, there will be unit tests in Service strategy, validation and defaulting to ensure
  the field cannot be used when the feature is disabled.

### Rollout, Upgrade and Rollback Planning

_This section must be completed when targeting beta graduation to a release._

* **How can a rollout fail? Can it impact already running workloads?**

	* By default this should not impact any existing Services since we are not changing any default behaviors.
	* Enabling this feature on new clusters can impact workloads only if the user wants to use a custom load balancer implementation and sets `service.spec.loadBalancerClass`.


* **What specific metrics should inform a rollback?**
	* None, if the user doesn't want to use a custom load balancer implementation, user can simply ignore this field.

* **Were upgrade and rollback tested? Was the upgrade->downgrade->upgrade path tested?**

   * No, upgrade->downgrade->upgrade has not been tested yet. Like any new API field, on downgrade any existing Services using the field will continue to have the field set. For these LoadBalancer type of Services, any default load balancer implementation (e.g. cloud providers) would ignore them. New Services cannot use the new field unless the feature gate is enabled in the old version when the feature was alpha.

* **Is the rollout accompanied by any deprecations and/or removals of features, APIs,
fields of API types, flags, etc.?**

   * No.

### Monitoring Requirements

_This section must be completed when targeting beta graduation to a release._

* **How can an operator determine if the feature is in use by workloads?**

  * Service should have `spec.loadBalancerClass` set.

* **What are the SLIs (Service Level Indicators) an operator can use to determine
the health of the service?**

	* N/A

* **What are the reasonable SLOs (Service Level Objectives) for the above SLIs?**

   * N/A

* **Are there any missing metrics that would be useful to have to improve observability
of this feature?**

   * N/A

### Dependencies

_This section must be completed when targeting beta graduation to a release._

* **Does this feature depend on any specific services running in the cluster?**

  * This feature is dependent on the Service LoadBalancer implementation of a cluster. This feature should only be used if the user doesn't want to use default load balancer implementation.

### Scalability

_For alpha, this section is encouraged: reviewers should consider these questions
and attempt to answer them._

_For beta, this section is required: reviewers must answer these questions._

_For GA, this section is required: approvers should be able to confirm the
previous answers based on experience in the field._

* **Will enabling / using this feature result in any new API calls?**

  Introduction of this feature enables multiple implementations of Service LoadBalancer
  for a single cluster. New API calls will be introduced by new controllers operating against
  Services with a non-empty class name. This feature does not introduce new API calls from
  core Kubernetes components.

* **Will enabling / using this feature result in introducing new API types?**

  No

* **Will enabling / using this feature result in any new calls to the cloud
provider?**

  Yes, introduction of new load balancer "classes" can introduce new calls to the cloud provider.

* **Will enabling / using this feature result in increasing size or count of
the existing API objects?**

  Yes, Service will (negligibly) increase with the addition of 1 new field.

* **Will enabling / using this feature result in increasing time taken by any
operations covered by [existing SLIs/SLOs]?**

  No

* **Will enabling / using this feature result in non-negligible increase of
resource usage (CPU, RAM, disk, IO, ...) in any components?**

  This change should not impact any core Kubernetes components.

### Troubleshooting

The Troubleshooting section currently serves the `Playbook` role. We may consider
splitting it into a dedicated `Playbook` document (potentially with some monitoring
details). For now, we leave it here.

_This section must be completed when targeting beta graduation to a release._

* **How does this feature react if the API server and/or etcd is unavailable?**

   * N/A

* **What are other known failure modes?**

   * If `service.spec.loadBalancerClass` set, but there is no load balancer implementation watches the set value. 

* **What steps should be taken if SLOs are not being met to determine the problem?**

   * N/A

[supported limits]: https://git.k8s.io/community//sig-scalability/configs-and-limits/thresholds.md
[existing SLIs/SLOs]: https://git.k8s.io/community/sig-scalability/slos/slos.md#kubernetes-slisslos

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
