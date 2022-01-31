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
- [X] (R) KEP approvers have approved the KEP status as `implementable`
- [X] (R) Design details are appropriately documented
- [X] (R) Test plan is in place, giving consideration to SIG Architecture and SIG Testing input
- [X] (R) Graduation criteria is in place
- [X] (R) Production readiness review completed
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

Add a new field `spec.internalTrafficPolicy` to Service that allows node-local and topology-aware routing for Service traffic.

## Motivation

Internal traffic routed to a Service has always been randomly distributed to all endpoints.
This KEP proposes a new API in Service to address use-cases such as node-local and topology aware routing
for internal Service traffic.

### Goals

* Allow internal Service traffic to be routed to node-local or topology-aware endpoints.
* Default behavior for internal Service traffic should not change.

### Non-Goals

* Topology aware routing for zone/region topologies -- while this field enables this feature, this KEP only covers node-local routing.
  See the Topology Aware Hints KEP for more details.

## Proposal

Introduce a new field in Service `spec.internalTrafficPolicy`. The field will have 2 codified values:
1. Cluster (default): route to all cluster-wide endpoints (or use topology aware subsetting if enabled).
2. Local: only route to node-local endpoints, drop otherwise.

A feature gate `ServiceInternalTrafficPolicy` will also be introduced this feature.
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

## Design Details

Proposed addition to core v1 API:
```go
type ServiceInternalTrafficPolicyType string

const (
	ServiceTrafficPolicyTypeCluster     ServiceTrafficPolicyType = "Cluster"
	ServiceTrafficPolicyTypeLocal       ServiceTrafficPolicyType = "Local"
)

// ServiceSpec describes the attributes that a user creates on a service.
type ServiceSpec struct {
	...
	...

	// InternalTrafficPolicy specifies if the cluster internal traffic
	// should be routed to all endpoints or node-local endpoints only.
	// "Cluster" routes internal traffic to a Service to all endpoints.
	// "Local" routes traffic to node-local endpoints only, traffic is
	// dropped if no node-local endpoints are ready.
	// The default value is "Cluster".
	// +featureGate=ServiceInternalTrafficPolicy
	// +optional
	InternalTrafficPolicy *ServiceInternalTrafficPolicyType `json:"internalTrafficPolicy,omitempty" protobuf:"bytes,22,opt,name=internalTrafficPolicy"`
}
```

This field will be independent from externalTrafficPolicy. In other words, internalTrafficPolicy only applies to traffic originating from internal sources.

Proposed changes to kube-proxy:
* when `internalTrafficPolicy=Cluster`, default to existing behavior today.
* when `internalTrafficPolicy=Local`, route to endpoints in EndpointSlice that maches the local node's topology, drop traffic if none exist.

Overlap with topology-aware routing:

| ExternalTrafficPolicy | InternalTrafficPolicy | Topology | External Result | Internal Result |
| - | - | - | - | - |
| - | - | Auto | Topology | Topology |
| Local | - | Auto | Local | Topology |
| Local | Local | Auto | Local | Local |

### Test Plan

Unit tests:
* unit tests validating API strategy/validation for when `internalTrafficPolicy` is set on Service.
* unit tests exercising kube-proxy behavior when `internalTrafficPolicy` is set to all possible values.

E2E test:
* e2e tests validating default behavior with kube-proxy did not change when `internalTrafficPolicy` defaults to `Cluster`. Existing tests should cover this.
* e2e tests validating that traffic is only sent to node-local endpoints when `internalTrafficPolicy` is set to `Local`.

### Graduation Criteria

Alpha:
* feature gate `ServiceInternalTrafficPolicy` _must_ be enabled for apiserver to accept values for `spec.internalTrafficPolicy`. Otherwise field is dropped.
* kube-proxy handles traffic routing for 2 initial internal traffic policies `Cluster`, and `Local`.
* Unit tests as defined in "Test Plan" section above. E2E tests are nice to have but not required for Alpha.

Beta:
* integration tests exercising API behavior for `spec.internalTrafficPolicy` field of Service.
* e2e tests exercising kube-proxy routing when `internalTrafficPolicy` is `Local`.
* feature gate `ServiceInternalTrafficPolicy` is enabled by default.
* consensus on how internalTrafficPolicy overlaps with topology-aware routing.

GA:
* metrics for if a Service is dropping local traffic due to no endpoints (a.k.a black hole)
* consensus on whether or not "PreferLocal" should be included as a new policy type

### Upgrade / Downgrade Strategy

* The `trafficPolicy` field will be off by default during the alpha stage but can handle any existing Services that has the field already set.
This ensures n-1 apiservers can handle the new field on downgrade.
* On upgrade, if the feature gate is enabled there should be no changes in the behavior since the default value for `trafficPolicy` is `Cluster`.

### Version Skew Strategy

Since this feature will be alpha for at least 1 release, an n-1 kube-proxy should handle enablement of this feature if a new apiserver enabled it.

## Production Readiness Review Questionnaire

### Feature Enablement and Rollback

_This section must be completed when targeting alpha to a release._

* **How can this feature be enabled / disabled in a live cluster?**
  - [X] Feature gate (also fill in values in `kep.yaml`)
    - Feature gate name: `ServiceInternalTrafficPolicy`
    - Components depending on the feature gate: kube-apiserver, kube-proxy

* **Does enabling the feature change any default behavior?**

No, enabling the feature does not change any default behavior since the default value of `internalTrafficPolicy` is `Cluster`.

* **Can the feature be disabled once it has been enabled (i.e. can we roll back
  the enablement)?**

Yes, the feature gate can be disabled, but Service resource that have set the new field will persist that field unless unset by the user.

* **What happens if we reenable the feature if it was previously rolled back?**

New Services should be able to set the `internalTrafficPolicy` field. Existing Services that have the field set will begin to apply the policy again.

* **Are there any tests for feature enablement/disablement?**

There will be unit tests to verify that apiserver will drop the field when the `ServiceInternalTrafficPolicy` feature gate is disabled.

### Rollout, Upgrade and Rollback Planning

_This section must be completed when targeting beta graduation to a release._

* **How can a rollout fail? Can it impact already running workloads?**

Rollout should have minimal impact because the default value of `internalTrafficPolicy` is `Cluster`, which is the default behavior today.

* **What specific metrics should inform a rollback?**

Metrics representing Services being black-holed will be added. This metric can inform rollback.

* **Were upgrade and rollback tested? Was the upgrade->downgrade->upgrade path tested?**

No, but this will be manually tested prior to beta. Automated testing will be done if the test tooling is available.

* **Is the rollout accompanied by any deprecations and/or removals of features, APIs,
fields of API types, flags, etc.?**

No.

### Monitoring Requirements

_This section must be completed when targeting beta graduation to a release._

* **How can an operator determine if the feature is in use by workloads?**

* Check Service to see if `internalTrafficPolicy` is set to `Local`.
* A per-node "blackhole" metric will be added to kube-proxy which represent Services that are being intentionally dropped (internalTrafficPolicy=Local and no endpoints).

TODO: add metric name once it's decided

* **What are the SLIs (Service Level Indicators) an operator can use to determine
the health of the service?**

They can check the "blackhole" metric when internalTrafficPolicy=Local and there are no endpoints.

* **What are the reasonable SLOs (Service Level Objectives) for the above SLIs?**

This will depend on Service topology and whether `internalTrafficPolicy=Local` is being used.

* **Are there any missing metrics that would be useful to have to improve observability
of this feature?**

A new metric will be added to represent Services that are being "blackholed" (internalTrafficPolicy=Local and no endpoints).

### Dependencies

_This section must be completed when targeting beta graduation to a release._

* **Does this feature depend on any specific services running in the cluster?**
  Think about both cluster-level services (e.g. metrics-server) as well
  as node-level agents (e.g. specific version of CRI). Focus on external or
  optional services that are needed. For example, if this feature depends on
  a cloud provider API, or upon an external software-defined storage or network
  control plane.

No.


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
traffic policies.

### Troubleshooting

The Troubleshooting section currently serves the `Playbook` role. We may consider
splitting it into a dedicated `Playbook` document (potentially with some monitoring
details). For now, we leave it here.

_This section must be completed when targeting beta graduation to a release._

* **How does this feature react if the API server and/or etcd is unavailable?**

Services will not be able to update their internal traffic policy.

* **What are other known failure modes?**

A Service `internalTrafficPolicy` is set to `Local` but there are no node-local endpoints.

* **What steps should be taken if SLOs are not being met to determine the problem?**

* check Service for internal traffic policy
* check EndpointSlice to ensure nodeName is set correctly
* check iptables/ipvs rules on kube-proxy

[supported limits]: https://git.k8s.io/community//sig-scalability/configs-and-limits/thresholds.md
[existing SLIs/SLOs]: https://git.k8s.io/community/sig-scalability/slos/slos.md#kubernetes-slisslos

## Implementation History

2020-10-09: KEP approved as implementable in "alpha" stage.
2021-03-08: alpha implementation merged for v1.21
2021-05-12: KEP approved as implementable in "beta" stage.

## Drawbacks

Added complexity in the Service API and in kube-proxy to address node-local routing.
This also pushes some responsibility on application owners to ensure pods are scheduled
to work with node-local routing.

## Alternatives

### EndpointSlice Subsetting

EndpointSlice subsetting per node can address the node-local use-case, but this would not be very scalable
for large clusters since that would require an EndpointSlice resource per node.

### Bool Field For Node Local

Instead of `trafficPolicy` field with codified values, a bool field can be used to enable node-local routing.
While this is simpler, it is not expressive enough for the `PreferLocal` use-case where traffic should ideally go
to a local endpoint, but be routed somewhere else otherwise.

