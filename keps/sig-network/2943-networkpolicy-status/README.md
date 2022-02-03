# KEP-2943: Network Policy Status subresource

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
  - [Notes/Constraints/Caveats (Optional)](#notesconstraintscaveats-optional)
  - [Risks and Mitigations](#risks-and-mitigations)
- [Design Details](#design-details)
  - [Test Plan](#test-plan)
  - [Graduation Criteria](#graduation-criteria)
    - [Alpha](#alpha)
    - [Beta](#beta)
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
  - [ ] (R) Ensure GA e2e tests for meet requirements for [Conformance Tests](https://github.com/kubernetes/community/blob/master/contributors/devel/sig-architecture/conformance-tests.md) 
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


NetworkPolicy objects does not contain a status subresource. This KEP 
proposes to add this subresource in NetworkPolicy objects, allowing Network Policy providers to provide 
feedback to users whether a NetworkPolicy and its features has been properly parsed.

## Motivation

While implementing new features in Network Policy, we've realized that some Network Policy providers were 
not able to implement parsing those new features. From a user perspective this is bad, 
as some user can create a NetworkPolicy containing a range of ports (`endPort`) for example
and the rule not being created properly but no feedback is provided for the user.

The same applies for other cases where protocols may or may not be supported (SCTP, for 
instance) or some limits (as an example, Cilium supports an array of up to 40 ports, 
over that it does not create the policy).

While Network Policy providers should give users feedback on invalid 
Network Policy as soon as possible (via webhooks, as an example), 
using status to provide feedback on whether a Network Policy could be parsed on a generic way is a desired 
feature that can help users, admins and the providers to understand 
why some network policy is not working.


### Goals

* Allow Network Policy providers to add a feedback/status to users on whether 
a Network Policy was properly parsed with its features

### Non-Goals

* Provide a feedback to users on whether a Network Policy was applied on all the cluster
nodes

## Proposal

The proposal of this KEP is to add/enable the status subresource in NetworkPolicy 
resources.

This subresource will contain only a single field with a tuple of (controller, 
status, description), aka a Condition.

### User Stories (Optional)

#### Story 1
Mary, the developer, created an application that communicates with an application in another
Kubernetes cluster, via NodePorts. Because the company implements restrictive Network Policies, 
Mary needs to create an Egress Network Policy allowing the communication with the other cluster 
on the range between 30000 to 32000. Mary knows that Kubernetes now supports the `endPort` field
and implemented a new Network Policy, but apparently it didn't worked and she doesn't know why. 

Mary then issues a `kubectl get netpol myegress -o yaml` to check if there is a feedback, but no
feedback is provided, so Mary needs the cluster to tell her what's wrong with her policy.

#### Story 2
Samantha, the cluster admin of a Kubernetes cluster is watching the complaints about Network 
Policies not working in the cluster raise after she upgraded the cluster Network Policy provider to the latest 
version.

Users decided to create Network Policies with some fancy new features that were announced 
in the release notes of a random NetPol provider, not knowing that this provider used in Samantha's cluster 
is different and still cannot implement the new fields.

Samantha needs to know exactly what are the complaints and why the provider cannot parse the 
new rules, so she can explain to the users and also plan their monthly cluster upgrade to 
add the required features.


### Notes/Constraints/Caveats (Optional)
The implementation of a status field depends on the Network Policy provider to return this feedback. It's 
not up to Kubernetes to know what happened, but to the controller of that object.

If decided to use a "generic" approach of "controller / status / description" we cannot 
guarantee that the Network Policy provider providers will follow the same standard/defaulting of the messages, 
and this might be frustrating for users.

### Risks and Mitigations
Wrong RBAC in .status subresource field may lead to someone adding wrong status to users.

The lack of the Network Policy Provider to provide a status to users may 
lead to some confusion on whether the network policy could be properly parsed.

It's up to the Network Policy Provider to decide how this feature is going to be 
implemented, and this can be even an early admission webhook, making this feature
useless.

## Design Details
API changes to NetworkPolicy:
* Add the `status` inside `NetworkPolicy` struct as following:
```go
// NetworkPolicy describes what network traffic is allowed for a set of Pods
type NetworkPolicy struct {
	metav1.TypeMeta `json:",inline"`
	// Standard object's metadata.
	// More info: https://git.k8s.io/community/contributors/devel/sig-architecture/api-conventions.md#metadata
	// +optional
	metav1.ObjectMeta `json:"metadata,omitempty" protobuf:"bytes,1,opt,name=metadata"`

	// Specification of the desired behavior for this NetworkPolicy.
	// +optional
	Spec NetworkPolicySpec `json:"spec,omitempty" protobuf:"bytes,2,opt,name=spec"`

  // Status is the current state of the NetworkPolicy.
	// More info: https://git.k8s.io/community/contributors/devel/sig-architecture/api-conventions.md#spec-and-status
	// +optional
	Status NetworkPolicyStatus `json:"status,omitempty" protobuf:"bytes,3,opt,name=status"`
}
```

* Add the `NetworkPolicyStatus` struct as following:
```go
// NetworkPolicyStatus describe the current state of the NetworkPolicy.
type NetworkPolicyStatus struct {
  // Conditions holds an array of metav1.Condition that describe the state of the NetworkPolicy.
 Conditions []metav1.Conditon
}
```
being metav1.Condition:

```go
type Condition struct {
	Type string `json:"type" protobuf:"bytes,1,opt,name=type"`
	// status of the condition, one of True, False, Unknown.
	Status ConditionStatus `json:"status" protobuf:"bytes,2,opt,name=status"`
	// observedGeneration represents the .metadata.generation that the condition was set based upon.
	// +optional
	// +kubebuilder:validation:Minimum=0
	ObservedGeneration int64 `json:"observedGeneration,omitempty" protobuf:"varint,3,opt,name=observedGeneration"`
	// lastTransitionTime is the last time the condition transitioned from one status to another.
	// This should be when the underlying condition changed.  If that is not known, then using the time when the API field changed is acceptable.
	// +required
	LastTransitionTime Time `json:"lastTransitionTime" protobuf:"bytes,4,opt,name=lastTransitionTime"`
	// reason contains a programmatic identifier indicating the reason for the condition's last transition.
	// Producers of specific condition types may define expected values and meanings for this field,
	// and whether the values are considered a guaranteed API.
	// The value should be a CamelCase string.
	// This field may not be empty.
	Reason string `json:"reason" protobuf:"bytes,5,opt,name=reason"`
	// message is a human readable message indicating details about the transition.
	// This may be an empty string.
	// +required
	// +kubebuilder:validation:Required
	// +kubebuilder:validation:MaxLength=32768
	Message string `json:"message" protobuf:"bytes,6,opt,name=message"`
}
```

The above is an extraction from https://github.com/kubernetes/apimachinery/blob/master/pkg/apis/meta/v1/types.go#L1475-L1521

```
<<[UNRESOLVED default conditions ]>>
Should Network Policy Status cover multiple network policy provider conditions? 
If so, how can this be solved? Prefixing the Type field with the implementation
name as `myprovider.com/PartialFailure?`

Also, what kind of polatiry should we use in a default condition Type? Positive polarity 
like "Accepted=bool" or negative polarity like PartialFailure=true?

See the original discussion in https://github.com/kubernetes/enhancements/pull/2947#discussion_r796974848
<<[/UNRESOLVED]>>
```

### Test Plan

<!--
**Note:** *Not required until targeted at a release.*

Consider the following in developing a test plan for this enhancement:
- Will there be e2e and integration tests, in addition to unit tests?
- How will it be tested in isolation vs with other components?

No need to outline all of the test cases, just the general strategy. Anything
that would count as tricky in the implementation, and anything particularly
challenging to test, should be called out.

All code is expected to have adequate tests (eventually with coverage
expectations). Please adhere to the [Kubernetes testing guidelines][testing-guidelines]
when drafting this test plan.

[testing-guidelines]: https://git.k8s.io/community/contributors/devel/sig-testing/testing.md
-->

Unit tests:
* test API validation logic
* test API strategy to ensure disabled fields
* test random insertions into the conditions field

E2E tests:
* Add e2e tests exercising only the API operations for the policy condition fields

### Graduation Criteria

#### Alpha

- Add a feature gated new status subresource to NetworkPolicy
- Initial e2e tests completed
- Communicate Network Policy providers about the new field
- Add validation tests in API

#### Beta

- Status has been supported for at least 1 minor release
- **two** commonly used NetworkPolicy providers implement the new field,
with generally positive feedback on its usage.
- Feature Gate is enabled by Default.

#### GA

- At least **three** NetworkPolicy providers support the status subresource and the conditions
- Status has been enabled by default for at least 2 minor release

### Upgrade / Downgrade Strategy

If upgraded no impact should happen as this is a new field.

If downgraded the Network Policy provider wont be able to write into 
this new field, which may lead to errors and inconsistency.

The provider implementing this feature should deal with errors 
returned from the API Server and bypass them in case of the field not existing in the current version.

### Version Skew Strategy

N/A

## Production Readiness Review Questionnaire

### Feature Enablement and Rollback

###### How can this feature be enabled / disabled in a live cluster?

<!--
Pick one of these and delete the rest.
-->

- [X] Feature gate (also fill in values in `kep.yaml`)
  - Feature gate name: NetworkPolicyStatus
  - Components depending on the feature gate:
    - API Server
    - Network Policy Providers


###### Does enabling the feature change any default behavior?

Yes, after enabling the feature gate network policy providers will start setting some status 
if implemented on the provider

###### Can the feature be disabled once it has been enabled (i.e. can we roll back the enablement)?

 Yes. Rolling back after Status has been added will keep 
 the status persisted on ETCD but the field wont be accessible on the API Server for GET/PUT operations.

Network Policy Providers wont be able to update the status field and 
this may lead to this providers returning errors when trying to 
access / change this field.

###### What happens if we reenable the feature if it was previously rolled back?

Network Policy Providers will be able again to add the status conditions.
###### Are there any tests for feature enablement/disablement?

Tests should be added in Alpha

### Rollout, Upgrade and Rollback Planning

###### How can a rollout or rollback fail? Can it impact already running workloads?

A rollout can cause unexpected crashes of kube-apiserver. Running workloads will not be affected

A rollback can be problematic from the Network Policy Provider perspective, 
as if decided to use this field, the provider will have to deal with 
failures if the Feature Gate is disabled or rolled back and the provider 
already relies on this field to provide its status.

###### What specific metrics should inform a rollback?

The increase of 5xx code in metric `apiserver_request_total`, on `resource=networkpolicies`, `subresource=status` 

###### Were upgrade and rollback tested? Was the upgrade->downgrade->upgrade path tested?

Should be targeted in Beta

###### Is the rollout accompanied by any deprecations and/or removals of features, APIs, fields of API types, flags, etc.?

No

### Monitoring Requirements


###### How can an operator determine if the feature is in use by workloads?

Check the increase of 2xx code in metric `apiserver_request_total`, on `resource=networkpolicies`, `subresource=status`

###### How can someone using this feature know that it is working for their instance?

- [X] API .status
  - Condition name: All
  - Other field: All

The .status field will be created together with the Conditions

###### What are the reasonable SLOs (Service Level Objectives) for the enhancement?

 - per-day percentage of API calls finishing with 5XX errors <= 1%

###### What are the SLIs (Service Level Indicators) an operator can use to determine the health of the service?

- [X] Metrics
  - Metric name: apiserver_request_total
    - filtered by `resource=networkpolicies`, `subresource=status`, `group=networking.k8s.io`, `version=v1`
  - Components exposing the metric: APIServer
- [X] Other 
  - Details: Network Policy Providers can expose metrics of communication with APIServer and 
  specifically Status operations. This is a recommendation and not a requirement.

###### Are there any missing metrics that would be useful to have to improve observability of this feature?

N/A

### Dependencies


###### Does this feature depend on any specific services running in the cluster?


  - [Network Policy Provider]
    - Usage description: Provide status on policies parsing
      - Impact of its outage on the feature: No status will be available for users
      - Impact of its degraded performance or high-error rates on the feature: Inconsistent status may be reported to users

### Scalability


###### Will enabling / using this feature result in any new API calls?

  - API call type: GET/POST/PUT/PATCH in networkpolicies/status
  - estimated throughput: Depends on the amount of existing Network Policies on cluster
  - originating component(s): Network Policy provider

###### Will enabling / using this feature result in introducing new API types?

No

###### Will enabling / using this feature result in any new calls to the cloud provider?

No

###### Will enabling / using this feature result in increasing size or count of the existing API objects?

Yes:

  - API type(s): NetworkPolicy / NetworkPolicyStatus
  - Estimated increase in size: New array of conditions with at least 512 bytes each (Status, Type, Message)
  - Estimated amount of new objects: New Status field for every existing Network Policy object


###### Will enabling / using this feature result in increasing time taken by any operations covered by existing SLIs/SLOs?

No

###### Will enabling / using this feature result in non-negligible increase of resource usage (CPU, RAM, disk, IO, ...) in any components?

It may get some resource usage increase in disk of Etcd (due to constant patching of status field), and in the 
Network Policy provider due to policies parsing and update into api server.

kube-apiserver would have also some increase in resource consumption as the network policy providers will post and watch 
the new subresources.

### Troubleshooting

###### How does this feature react if the API server and/or etcd is unavailable?

It won't be possible for Network Policy providers to update the policy status, and for users to get its status
###### What are other known failure modes?

<!--
For each of them, fill in the following information by copying the below template:
  - [Failure mode brief description]
    - Detection: How can it be detected via metrics? Stated another way:
      how can an operator troubleshoot without logging into a master or worker node?
    - Mitigations: What can be done to stop the bleeding, especially for already
      running user workloads?
    - Diagnostics: What are the useful log messages and their required logging
      levels that could help debug the issue?
      Not required until feature graduated to beta.
    - Testing: Are there any tests for failure mode? If not, describe why.
-->

###### What steps should be taken if SLOs are not being met to determine the problem?

## Implementation History

- 2022-02-01 Initial KEP

## Drawbacks

Network Policy Providers may not want to use this field. They already provide this 
feature as part of their CRDs, and implementing this may lead to 
additional overhead in the implementation that may not be desired.

## Alternatives

Network Policy Providers may want to implement an additional validation webhook 
instead of relying on an API field to let users know about 
the correctness of a Network Policy

