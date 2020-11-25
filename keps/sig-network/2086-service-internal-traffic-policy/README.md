# KEP-2086: Service Internal Traffic Policy

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
  - [EndpointSlice Subsetting](#endpointslice-subsetting)
  - [Bool Field For Node Local](#bool-field-for-node-local)
<!-- /toc -->

## Release Signoff Checklist

Items marked with (R) are required *prior to targeting to a milestone / release*.

- [X] (R) Enhancement issue in release milestone, which links to KEP dir in [kubernetes/enhancements] (not the initial KEP PR)
- [ ] (R) KEP approvers have approved the KEP status as `implementable`
- [ ] (R) Design details are appropriately documented
- [ ] (R) Test plan is in place, giving consideration to SIG Architecture and SIG Testing input
- [ ] (R) Graduation criteria is in place
- [ ] (R) Production readiness review completed
- [ ] Production readiness review approved
- [ ] "Implementation History" section is up-to-date for milestone
- [ ] User-facing documentation has been created in [kubernetes/website], for publication to [kubernetes.io]
- [ ] Supporting documentation—e.g., additional design documents, links to mailing list discussions/SIG meetings, relevant PRs/issues, release notes

<!--
**Note:** This checklist is iterative and should be reviewed and updated every time this enhancement is being considered for a milestone.
-->

[kubernetes.io]: https://kubernetes.io/
[kubernetes/enhancements]: https://git.k8s.io/enhancements
[kubernetes/kubernetes]: https://git.k8s.io/kubernetes
[kubernetes/website]: https://git.k8s.io/website

## Summary

Add a new field `spec.internalTrafficPolicy` to Service that allows node-local routing for Service internal traffic.

## Motivation

Internal traffic routed to a Service is not topology aware today. The [Topolgoy Aware Subsetting](/keps/sig-network/2004-topology-aware-subsetting)
KEP addresses topology aware routing for Services by subsetting endpoints to dedicated EndpointSlices.
While this approach works for the standard zone/region topologies, it wouldn't work for node level
topologies since that would require an EndpointSlice per node. In larger clusters this wouldn't scale well.

This KEP proposes a new field in Service to treat node-local topologies as a first class concept in Service similar
to `externalTrafficPolicy`. This addresses the node-local use-case for Service while avoiding EndpointSlice
subsetting per node.

### Goals

* Allow internal Service traffic to be routed to node-local endpoints.
* Default behavior for internal Service traffic should not change.

### Non-Goals

* Topology aware routing for zone/region topologies.

## Proposal

Introduce a new field in Service `spec.internalTrafficPolicy`. The field will have 3 codified values:
1. All (default): route to all endpoints (or use topology aware subsetting if enabled).
2. PreferLocal: route to node-local endpoints if it exists, otherwise fallback to behavior from All.
3. Local: only route to node-local endpoints, drop otherwise.

A feature gate `ServiceInternalTrafficPolicy` will also be introduced for the alpha stage of this feature.
The `internalTrafficPolicy` field cannot be set on Service during the alpha stage unless the feature gate is enabled.
During the Beta stage, the feature gate will be on by default.

The `internalTrafficPolicy` field will not apply for headless Services or Services of type `ExternalName`.

### User Stories (Optional)

#### Story 1

As an application owner, I would like traffic to cluster DNS servers to always prefer local endpoints to reduce
latency in my application.

#### Story 2

As a platform owner, I want to create a Service that always directs traffic to a logging daemon on the same node.
Traffic should never bounce to a daemon on another node.

### Risks and Mitigations

* When the `Local` policy is set, it is the user's responsibility to ensure node-local endpoints are ready, otherwise traffic will be dropped.
* Using the `Local` or `PreferLocal` policy may result in imbalanced traffic for pods in a Service. It is the user's responsibility to handle this.

## Design Details

Proposed addition to core v1 API:
```go
type ServiceInternalTrafficPolicyType string

const (
	ServiceInternalTrafficPolicyTypeAll         ServiceInternalTrafficPolicyType = "All"
	ServiceInternalTrafficPolicyTypePreferLocal ServiceInternalTrafficPolicyType = "PreferLocal"
	ServiceInternalTrafficPolicyTypeLocal       ServiceInternalTrafficPolicyType = "Local"
)

// ServiceSpec describes the attributes that a user creates on a service.
type ServiceSpec struct {
	...
	...

	// internalTrafficPolicy denotes if the internal traffic for a Service should route
	// to cluster-wide endpoints or node-local endpoints. "Cluster" routes internal traffic
	// to a Service to all cluster-wide endpoints. "PreferLocal" will route internal traffic
	// to node-local endpoints if one exists, otherwise it will fallback to the same behavior
	// as "Cluster". "Local" routes traffic to node-local endpoints only, traffic is dropped
	// if no node-local endpoints are ready.
	InternalTrafficPolicy ServiceInternalTrafficPolicyType `json:"internalTrafficPolicy,omitempty"`
}
```

Proposed changes to kube-proxy:
* when `internalTrafficPolicy=All`, default to existing behavior today.
* when `internalTrafficPolicy=PreferLocal`, route to endpoints in EndpointSlice that matches the local node's topology (topology defined by `kubernetes.io/hostname`),
fall back to "All" behavior if there are no local endpoints.
* when `internalTrafficPolicy=Local`, route to endpoints in EndpointSlice that maches the local node's topology, drop traffic if none exist.

### Test Plan

Unit tests:
* unit tests validating API strategy/validation for when `internalTrafficPolicy` is set on Service.
* unit tests exercising kube-proxy behavior when `internalTrafficPolicy` is set to all possible values.

E2E test:
* e2e tests validating default behavior with kube-proxy did not change when `internalTrafficPolicy` defaults to `All`. Existing tests should cover this.
* e2e tests validating that traffic is preferred to local endpoints when `internalTrafficPolicy` is set to `PreferLocal`.
* e2e tests validating that traffic is only sent to node-local endpoints when `internalTrafficPolicy` is set to `Local`.

### Graduation Criteria

Alpha:
* feature gate `ServiceInternalTrafficPolicy` _must_ be enabled for apiserver to accept values for `spec.internalTrafficPolicy`. Otherwise field is dropped.
* kube-proxy handles traffic routing for 3 initial internal traffic policies `All`, `PreferLocal` and `Local`.
* Unit tests as defined in "Test Plan" section above. E2E tests are nice to have but not required for Alpha.


### Upgrade / Downgrade Strategy

* The `internalTrafficPolicy` field will be off by default during the alpha stage but can handle any existing Services that has the field already set.
This ensures n-1 apiservers can handle the new field on downgrade.
* On upgrade, if the feature gate is enabled there should be no changes in the behavior since the default value for `internalTrafficPolicy` is `Cluster`.

### Version Skew Strategy

Since this feature will be alpha for at least 1 release, an n-1 kube-proxy should handle enablement of this feature if a new apiserver enabled it.

## Production Readiness Review Questionnaire

### Feature Enablement and Rollback

_This section must be completed when targeting alpha to a release._

* **How can this feature be enabled / disabled in a live cluster?**
  - [X] Feature gate (also fill in values in `kep.yaml`)
    - Feature gate name: `ServiceInternalTrafficPolicy`
    - Components depending on the feature gate: kube-apiserver, kube-proxy
  - [ ] Other
    - Describe the mechanism:
    - Will enabling / disabling the feature require downtime of the control
      plane?
    - Will enabling / disabling the feature require downtime or reprovisioning
      of a node? (Do not assume `Dynamic Kubelet Config` feature is enabled).

* **Does enabling the feature change any default behavior?**

No, enabling the feature does not change any default behavior since the default value of `internalTrafficPolicy` is `All`.

* **Can the feature be disabled once it has been enabled (i.e. can we roll back
  the enablement)?**

Yes, the feature gate can be disabled, but Service resource that have set the new field will persist that field unless unset by the user.

* **What happens if we reenable the feature if it was previously rolled back?**

New Services should be able to set the `internalTrafficPolicy` field. Existing Services that have the field set already should not be impacted.

* **Are there any tests for feature enablement/disablement?**

There will be unit tests to verify that apiserver will drop the field when the `ServiceInternalTrafficPolicy` feature gate is disabled.

### Rollout, Upgrade and Rollback Planning

_This section must be completed when targeting beta graduation to a release._

* **How can a rollout fail? Can it impact already running workloads?**

TBD for beta.

* **What specific metrics should inform a rollback?**

TBD for beta.

* **Were upgrade and rollback tested? Was the upgrade->downgrade->upgrade path tested?**

TBD for beta.

* **Is the rollout accompanied by any deprecations and/or removals of features, APIs,
fields of API types, flags, etc.?**

TBD for beta.

### Monitoring Requirements

_This section must be completed when targeting beta graduation to a release._

* **How can an operator determine if the feature is in use by workloads?**

TBD for beta.

* **What are the SLIs (Service Level Indicators) an operator can use to determine
the health of the service?**

TBD for beta.

* **What are the reasonable SLOs (Service Level Objectives) for the above SLIs?**

TBD for beta.

* **Are there any missing metrics that would be useful to have to improve observability
of this feature?**

TBD for beta.

### Dependencies

_This section must be completed when targeting beta graduation to a release._

* **Does this feature depend on any specific services running in the cluster?**
  Think about both cluster-level services (e.g. metrics-server) as well
  as node-level agents (e.g. specific version of CRI). Focus on external or
  optional services that are needed. For example, if this feature depends on
  a cloud provider API, or upon an external software-defined storage or network
  control plane.

TBD for beta.


### Scalability

_For alpha, this section is encouraged: reviewers should consider these questions
and attempt to answer them._

_For beta, this section is required: reviewers must answer these questions._

_For GA, this section is required: approvers should be able to confirm the
previous answers based on experience in the field._

* **Will enabling / using this feature result in any new API calls?**

No, since this is a user-defined field in Service. No extra calls will be required
from EndpointSlice as well since topology information is already stored there.

* **Will enabling / using this feature result in introducing new API types?**

No API types are introduced, only a new field in Service.

* **Will enabling / using this feature result in any new calls to the cloud
provider?**

No

* **Will enabling / using this feature result in increasing size or count of
the existing API objects?**

This feature will (negligibly) increase the size of Service by adding a single field.

* **Will enabling / using this feature result in increasing time taken by any
operations covered by [existing SLIs/SLOs]?**
  Think about adding additional work or introducing new steps in between
  (e.g. need to do X to start a container), etc. Please describe the details.

This feature may slightly increase kube-proxy's sync time for iptable / IPVS rules,
since node topology must be calculated, but this is likely negligible given we
already have many checks like this for `externalTrafficPolicy: Local`.

* **Will enabling / using this feature result in non-negligible increase of
resource usage (CPU, RAM, disk, IO, ...) in any components?**

Any increase in CPU usage by kube-proxy to calculate node-local topology will likely
be offset by reduced iptable rules it needs to sync when using `PreferLocal` or `Local`
internal traffic policies.

### Troubleshooting

The Troubleshooting section currently serves the `Playbook` role. We may consider
splitting it into a dedicated `Playbook` document (potentially with some monitoring
details). For now, we leave it here.

_This section must be completed when targeting beta graduation to a release._

* **How does this feature react if the API server and/or etcd is unavailable?**

TBD for beta.

* **What are other known failure modes?**

TBD for beta.

* **What steps should be taken if SLOs are not being met to determine the problem?**

TBD for beta.

[supported limits]: https://git.k8s.io/community//sig-scalability/configs-and-limits/thresholds.md
[existing SLIs/SLOs]: https://git.k8s.io/community/sig-scalability/slos/slos.md#kubernetes-slisslos

## Implementation History

2020-10-09: KEP approved as implementable in "alpha" stage.

## Drawbacks

Added complexity in the Service API and in kube-proxy to address node-local routing.
This also pushes some responsibility on application owners to ensure pods are scheduled
to work with node-local routing.

## Alternatives

### EndpointSlice Subsetting

EndpointSlice subsetting per node can address the node-local use-case, but this would not be very scalable
for large clusters since that would require an EndpointSlice resource per node.

### Bool Field For Node Local

Instead of `internalTrafficPolicy` field with codified values, a bool field can be used to enable node-local routing.
While this is simpler, it is not expressive enough for the `PreferLocal` use-case where traffic should ideally go
to a local endpoint, but be routed somewhere else otherwise.

