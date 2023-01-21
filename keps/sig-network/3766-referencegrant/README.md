# KEP-3766: Move ReferenceGrant to sig-auth API Group

<!-- toc -->
- [Release Signoff Checklist](#release-signoff-checklist)
- [Summary](#summary)
- [Motivation](#motivation)
  - [Goals](#goals)
  - [Non-Goals](#non-goals)
- [Proposal](#proposal)
  - [Risks and Mitigations](#risks-and-mitigations)
    - [No Default Implementation](#no-default-implementation)
    - [Potential for Variations Among Implementations](#potential-for-variations-among-implementations)
    - [Cross-Namespace References may Weaken Namespace Boundaries](#cross-namespace-references-may-weaken-namespace-boundaries)
- [Design Details](#design-details)
  - [API Spec](#api-spec)
  - [Test Plan](#test-plan)
      - [Prerequisite testing updates](#prerequisite-testing-updates)
      - [Unit tests](#unit-tests)
      - [Integration tests](#integration-tests)
      - [e2e tests](#e2e-tests)
  - [Graduation Criteria](#graduation-criteria)
    - [GA](#ga)
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
<!-- /toc -->

## Release Signoff Checklist

Items marked with (R) are required *prior to targeting to a milestone / release*.

- [ ] (R) Enhancement issue in release milestone, which links to KEP dir in [kubernetes/enhancements] (not the initial KEP PR)
- [ ] (R) KEP approvers have approved the KEP status as `implementable`
- [ ] (R) Design details are appropriately documented
- [ ] (R) Test plan is in place, giving consideration to SIG Architecture and SIG Testing input (including test refactors)
  - [ ] e2e Tests for all Beta API Operations (endpoints)
  - [ ] (R) Ensure GA e2e tests meet requirements for [Conformance Tests](https://github.com/kubernetes/community/blob/master/contributors/devel/sig-architecture/conformance-tests.md) 
  - [ ] (R) Minimum Two Week Window for GA e2e tests to prove flake free
- [ ] (R) Graduation criteria is in place
  - [ ] (R) [all GA Endpoints](https://github.com/kubernetes/community/pull/1806) must be hit by [Conformance Tests](https://github.com/kubernetes/community/blob/master/contributors/devel/sig-architecture/conformance-tests.md) 
- [ ] (R) Production readiness review completed
- [ ] (R) Production readiness review approved
- [ ] "Implementation History" section is up-to-date for milestone
- [ ] User-facing documentation has been created in [kubernetes/website], for publication to [kubernetes.io]
- [ ] Supporting documentation—e.g., additional design documents, links to mailing list discussions/SIG meetings, relevant PRs/issues, release notes

[kubernetes.io]: https://kubernetes.io/
[kubernetes/enhancements]: https://git.k8s.io/enhancements
[kubernetes/kubernetes]: https://git.k8s.io/kubernetes
[kubernetes/website]: https://git.k8s.io/website

## Summary

[ReferenceGrant](https://gateway-api.sigs.k8s.io/api-types/referencegrant/) was
developed by the [Gateway API subproject](https://gateway-api.sigs.k8s.io/) to
enable certain object references to cross namespaces. More recently, it has also
been [used by
sig-storage](https://kubernetes.io/blog/2023/01/02/cross-namespace-data-sources-alpha/)
to enable cross-namespace data sources.

This KEP proposes moving ReferenceGrant from its current
`gateway.networking.k8s.io` API group to a new `grants.authorization.k8s.io` API
group.

## Motivation

Now that it's clear that ReferenceGrant is useful beyond just Gateway API, it
would be good to formalize this model in a more neutral home, ideally sig-auth.
At this point, each project that wants to enable cross-namespace references has
to choose between introducing a dependency on Gateway API and creating a new
resource that would largely duplicate ReferenceGrant. Both options would lead to
confusion for Kubernetes users.

### Goals

* Move ReferenceGrant to sig-auth API Group
* Clearly define how ReferenceGrant should be used, including both current use
  cases and guidance for future use cases
* Implement a library to ensure that ReferenceGrant is implemented consistently
  by all controllers

### Non-Goals

* Add, remove, or change fields in the ReferenceGrant API
* Develop an authorizer that will automatically implement ReferenceGrant for
  all use cases

## Proposal

Move the existing ReferenceGrant resource into a new
`grants.authorization.k8s.io` API group, defined within the Kubernetes code base
as part of the 1.27 release. We may take this opportunity to clarify
underspecified parts of the API, but will not add, change, or remove any fields
as part of this transition. This resource will start with v1beta1 as the API
version, matching the API version it already has within Gateway API.

### Risks and Mitigations

#### No Default Implementation
Similar to the Ingress and Gateway APIs, this API will be dependent on
implementations by controllers that are not included by default in Kubernetes.
This could lead to confusion for users. We'll need to rely heavily on
documentation for this feature, tracking all uses of official Kubernetes APIs
that support ReferenceGrant in a central place.

#### Potential for Variations Among Implementations
Because this relies on each individual controller to implement the logic,
it is possible that implementations may become inconsistent. To avoid that,
we'll provide a standard library for implementing ReferenceGrant. We'll
also strongly recommend that every API that relies on ReferenceGrant 
includes robust conformance tests covering this functionality. Existing
Gateway API conformance tests can serve as a model for this.

#### Cross-Namespace References may Weaken Namespace Boundaries
Although we believe that the handshake required for cross-namespace references
with ReferenceGrant ensures these references will be safe, it does potentially
weaken existing namespace boundaries. We believe ReferenceGrant will have a net
benefit on the ecosystem as it will allow workloads, secrets, and configuration
to be deployed in separate namespaces that more clearly match up with desired
authorization.

## Design Details

### API Spec

```golang
// ReferenceGrant identifies kinds of resources in other namespaces that are
// trusted to reference the specified kinds of resources in the same namespace
// as the policy.
//
// Each ReferenceGrant can be used to represent a unique trust relationship.
// Additional ReferenceGrants can be used to add to the set of trusted
// sources of inbound references for the namespace they are defined within.
//
// ReferenceGrant is a form of runtime verification allowing users to assert
// which cross-namespace object references are permitted. Implementations that
// support ReferenceGrant MUST NOT permit cross-namespace references which have
// no grant, and MUST respond to the removal of a grant by revoking the access
// that the grant allowed.
//
// Support: Core
type ReferenceGrant struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty"`

	// Spec defines the desired state of ReferenceGrant.
	Spec ReferenceGrantSpec `json:"spec,omitempty"`

	// Note that `Status` sub-resource has been excluded at the
	// moment as it was difficult to work out the design.
	// A `Status` sub-resource may be added in the future.
}

// +kubebuilder:object:root=true
// ReferenceGrantList contains a list of ReferenceGrant.
type ReferenceGrantList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata,omitempty"`
	Items           []ReferenceGrant `json:"items"`
}

// ReferenceGrantSpec identifies a cross namespace relationship that is trusted
// for Gateway API.
type ReferenceGrantSpec struct {
	// From describes the trusted namespaces and kinds that can reference the
	// resources described in "To". Each entry in this list MUST be considered
	// to be an additional place that references can be valid from, or to put
	// this another way, entries MUST be combined using OR.
	//
	// +kubebuilder:validation:MinItems=1
	// +kubebuilder:validation:MaxItems=16
	From []ReferenceGrantFrom `json:"from"`

	// To describes the resources that may be referenced by the resources
	// described in "From". Each entry in this list MUST be considered to be an
	// additional place that references can be valid to, or to put this another
	// way, entries MUST be combined using OR.
	//
	// +kubebuilder:validation:MinItems=1
	// +kubebuilder:validation:MaxItems=16
	To []ReferenceGrantTo `json:"to"`
}

// ReferenceGrantFrom describes trusted namespaces and kinds.
type ReferenceGrantFrom struct {
	// Group is the group of the referent.
	// When empty, the Kubernetes core API group is inferred.
	Group Group `json:"group"`

	// Kind is the kind of the referent.
	Kind string `json:"kind"`

	// Namespace is the namespace of the referent.
	Namespace string `json:"namespace"`
}

// ReferenceGrantTo describes what Kinds are allowed as targets of the
// references.
type ReferenceGrantTo struct {
	// Group is the group of the referent.
	// When empty, the Kubernetes core API group is inferred.
	Group string `json:"group"`

	// Kind is the kind of the referent.
	Kind string `json:"kind"`

	// Name is the name of the referent. When unspecified, this policy
	// refers to all resources of the specified Group and Kind in the local
	// namespace.
	//
	// +optional
	Name *string `json:"name,omitempty"`
}
```


### Test Plan

[x] I/we understand the owners of the involved components may require updates to
existing tests to make this code solid enough prior to committing the changes necessary
to implement this enhancement.

##### Prerequisite testing updates

This is a net new resource to Kubernetes so it will not require any changes or
additions to existing tests.

##### Unit tests

Unit tests will be used to cover:

1. ReferenceGrant validation
2. ReferenceGrant implementation library

More details will be added as the details of the implementation library are
clarified.

##### Integration tests

Because we are not bundling an implementation with this API, no integration
tests are possible.

##### e2e tests

We will strongly encourage every API that uses ReferenceGrant to define
conformance tests for their use of ReferenceGrant.

### Graduation Criteria

#### GA

[x] Almost all of the fields and behavior have conformance test coverage.
[x] Multiple conformant implementations.
[x] Widespread implementation and usage.
[ ] Conformance tests that exercise all ReferenceGrant API calls (not the actual implementation of the API).
[ ] At least 6 months of soak time as a beta API (graduated to beta in Dec 2022).


### Upgrade / Downgrade Strategy

N/A

### Version Skew Strategy

Version skew is a bit different here since it won't be implemented in-tree.
There will be some implementation that support both the API defined by Gateway
API and this API. Since these resources are entirely additive and can be
duplicative, we can copy Gateway API resources to the new API group and delete
the old Gateway API resources as part of a seamless migration. We expect that
many implementations will provide this recommendation to users, and we may even
provide tooling to simplify this process.

## Production Readiness Review Questionnaire

### Feature Enablement and Rollback

###### How can this feature be enabled / disabled in a live cluster?

- [x] Other
  - Describe the mechanism: Enable beta ReferenceGrant API
  - Will enabling / disabling the feature require downtime of the control
    plane?
    No
  - Will enabling / disabling the feature require downtime or reprovisioning
    of a node? (Do not assume `Dynamic Kubelet Config` feature is enabled).
    No

###### Does enabling the feature change any default behavior?

No

###### Can the feature be disabled once it has been enabled (i.e. can we roll back the enablement)?

Yes

###### What happens if we reenable the feature if it was previously rolled back?

The API would become accessible again, implementing controllers may need to be
restarted to pick up the presence of this API.

###### Are there any tests for feature enablement/disablement?

No

### Rollout, Upgrade and Rollback Planning

###### How can a rollout or rollback fail? Can it impact already running workloads?

API enablement may not work, but that would not be unique to this API.

###### What specific metrics should inform a rollback?

N/A, this is just an API

###### Were upgrade and rollback tested? Was the upgrade->downgrade->upgrade path tested?

N/A, this is just an API

###### Is the rollout accompanied by any deprecations and/or removals of features, APIs, fields of API types, flags, etc.?

No

### Monitoring Requirements

N/A, this is just an API

###### How can an operator determine if the feature is in use by workloads?
```
kubectl get referencegrants --all-namespaces
```

###### How can someone using this feature know that it is working for their instance?

This will be dependent on the API that ReferenceGrant is used with. In Gateway API,
each resource has clear status conditions that reflect the validity of a cross-namespace
reference.

###### What are the reasonable SLOs (Service Level Objectives) for the enhancement?

N/A, this is just an API

###### What are the SLIs (Service Level Indicators) an operator can use to determine the health of the service?

N/A, this is just an API

###### Are there any missing metrics that would be useful to have to improve observability of this feature?

No

### Dependencies

N/A, this is just an API

###### Does this feature depend on any specific services running in the cluster?

- API Server

### Scalability

###### Will enabling / using this feature result in any new API calls?

Yes, users may install controllers that watch for changes to ReferenceGrants.
Users may also create ReferenceGrants to enable cross-namespace references.

###### Will enabling / using this feature result in introducing new API types?

API Type: ReferenceGrant
Supported Number of Objects per Cluster: No limit 
Supported Number of Objects per Namespace: No limit

###### Will enabling / using this feature result in any new calls to the cloud provider?

No

###### Will enabling / using this feature result in increasing size or count of the existing API objects?

No

###### Will enabling / using this feature result in increasing time taken by any operations covered by existing SLIs/SLOs?

No

###### Will enabling / using this feature result in non-negligible increase of resource usage (CPU, RAM, disk, IO, ...) in any components?

No

### Troubleshooting

###### How does this feature react if the API server and/or etcd is unavailable?

The API would not be accessible. We would likely recommend that controllers revoke
cross-namespace references if they could not find ReferenceGrants that allow them
so this could result in a disruption for anything that relied on cross-namespace
references.

###### What are other known failure modes?

N/A, this is just an API

###### What steps should be taken if SLOs are not being met to determine the problem?

N/A, this is just an API

## Implementation History

* July 2021: [ReferencePolicy is proposed in Gateway API](https://github.com/kubernetes-sigs/gateway-api/pull/711)
* August 2021: [ReferencePolicy is added to Gateway API](https://github.com/kubernetes-sigs/gateway-api/pulls?page=2&q=is%3Apr+is%3Aclosed+ReferencePolicy)
* June 2022: [ReferencePolicy is renamed to ReferenceGrant](https://github.com/kubernetes-sigs/gateway-api/pull/1179)
* December 2022: [SIG-Storage uses ReferenceGrant for cross-namespace data storage sources](https://kubernetes.io/blog/2023/01/02/cross-namespace-data-sources-alpha/)
* December 2022: [ReferenceGrant graduates to beta in Gateway API v0.6.0](https://github.com/kubernetes-sigs/gateway-api/releases/tag/v0.6.0)

## Drawbacks

N/A

## Alternatives

1. ReferenceGrant could remain as a CRD
This would probably be fine, we just don't really have a good place for it to
live. This could also complicate installation of Gateway API and other APIs that
depended on this.

2. Every API that wanted to support cross-namespace references could maintain their own version of ReferenceGrant
This would be a confusing mess, we should avoid this at all costs.

