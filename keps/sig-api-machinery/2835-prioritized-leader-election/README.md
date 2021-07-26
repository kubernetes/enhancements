# KEP-2835: Prioritized Leader Election

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
- [Infrastructure Needed (Optional)](#infrastructure-needed-optional)
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

We propose to add an extension to the existing leader election mechanism in client-go to support
prioritizing election participants by user-defined priority comparison functions.

## Motivation

Prioritized leader election has a few key use cases:

* I have multiple versions of an application running concurrently in the
cluster; I want the newest one to become the leader.
* I have multiple replicas with different resources; I want the largest
replica to be the leader.
* I am implementing a cross-region controller (think multicluster
service discovery) and the geographically closest region to be the
leader, to avoid something like a US controller acting on a cluster in
Asia, when I have a controller replica already running in Asia.


### Goals

* Provide a simple mechanism to allow users to opt into prioritized leader election.

### Non-Goals

* User prioritized leader election in Kubernete's maintained controllers
* Require prioritized leader election to be used by all users of leader election

## Proposal


### User Stories (Optional)

#### Story 1

As a controller implementor, I have multiple versions of my controller running concurrently in the
cluster; I want the newest one to become the leader.

#### Story 2

As a controller implementor, I have multiple replicas with different resources; I want the largest
replica to be the leader.

#### Story 3

As a controller implementor of a MCS controller, I want the geographically closest region to be the
leader, to avoid something like a US controller acting on a cluster in
Asia, when I have a controller replica already running in Asia.

### Notes/Constraints/Caveats (Optional)

N/A
### Risks and Mitigations

The largest risk here is that this is a new field. This means that users will need to account for older API versions,
or controllers that do not understand prioritized leader election.

This is mitigated by documenting that users need to take this into account.


## Design Details

A new configuration field will be provided to the leader election configuration:

```go
type KeyComparisonFunc func(existingKey string) bool

type LeaderElectionConfig struct {
...

	// KeyComparison defines a function to compare the existing leader's key to our own.
	// If the function returns true, indicating our key has high precedence, we will take over
	// leadership even if their is another un-expired leader.
	//
	// This can be used to implemented a prioritized leader election. For example, if multiple
	// versions of the same application run simultaneously, we can ensure the newest version
	// will become the leader.
	//
	// It is the responsibility of the caller to ensure that all KeyComparison functions are
	// logically consistent between all clients participating in the leader election to avoid multiple
	// clients claiming to have high precedence and constantly pre-empting the existing leader.
	//
	// KeyComparison functions should ensure they handle an empty existingKey, as "key" is not a required field.
	//
	// Warning: when a lock is stolen (from KeyComparison returning true), the old leader may not
	// immediately be notified they have lost the leader election.
	KeyComparison KeyComparisonFunc
}
```

The API of an election lock will also be changed. Note this is the same change in two places - one defines
the `Lease` API and the other is a mirror of the API used for all leader election types.

```go
type LeaderElectionRecord struct {
...
	// HolderKey is the Key of the lease owner. This may be empty if a key is not set.
	HolderKey            string      `json:"holderKey"`
}

type LeaseSpec struct {
...
	// HolderKey is the Key of the lease owner. This may be empty if a key is not set.
    // +optional
	HolderKey            string      `json:"holderKey,omitempty"`
}
```

### Test Plan

Unit tests will be added to `leaderelection/leaderelection_test.go`. This follows the existing pattern
of not having e2e tests for the leader election library. The tests will ensure to capture edge cases
involving upgrades, including handle the `key` field being unset. They will also test all forms of leader
election (Endpoints, ConfigMap, Lease, etc).

### Graduation Criteria

N/A

### Upgrade / Downgrade Strategy

See version skew strategy.

### Version Skew Strategy

During a version skew, we may have leader election clients that do not have support for this functionality.
As a result, they will not set the `key` field, nor will they read it.

* For not reading the `key` field, this is okay - older clients are just not allowed to pre-empt others, maintaining the previous logic.
* For not setting the `key` field, this is okay as well - `KeyComparisonFunc`s are expected to take an empty `key` into account.

## Production Readiness Review Questionnaire

### Feature Enablement and Rollback

This is explicitly opt-in in client-go library.

###### How can this feature be enabled / disabled in a live cluster?

- [x] Other
  - Describe the mechanism: explicitly opt-in in client-go library
  - Will enabling / disabling the feature require downtime of the control
    plane? No.
  - Will enabling / disabling the feature require downtime or reprovisioning
    of a node? No..

###### Does enabling the feature change any default behavior?

No.

###### Can the feature be disabled once it has been enabled (i.e. can we roll back the enablement)?

Yes, the user can just stop using the option in client-go.

###### What happens if we reenable the feature if it was previously rolled back?

See Version Skew Strategy; the same applies.

###### Are there any tests for feature enablement/disablement?

This is covered in the test plan.

### Rollout, Upgrade and Rollback Planning

N/A

###### How can a rollout or rollback fail? Can it impact already running workloads?

N/A

###### What specific metrics should inform a rollback?

N/A

###### Were upgrade and rollback tested? Was the upgrade->downgrade->upgrade path tested?

This is covered in the test plan.

###### Is the rollout accompanied by any deprecations and/or removals of features, APIs, fields of API types, flags, etc.?

No

### Monitoring Requirements

No additional metrics are added

###### How can an operator determine if the feature is in use by workloads?

They can statically analyse their controller to see if the fields in client-go are set. They could also
look at locks in their cluster for the `key` field.

###### How can someone using this feature know that it is working for their instance?

The user is resonsible for testing their usage of client-go.

###### What are the reasonable SLOs (Service Level Objectives) for the enhancement?

N/A

###### What are the SLIs (Service Level Indicators) an operator can use to determine the health of the service?
N/A

###### Are there any missing metrics that would be useful to have to improve observability of this feature?

N/A

### Dependencies

None.

###### Does this feature depend on any specific services running in the cluster?

No.

### Scalability

###### Will enabling / using this feature result in any new API calls?

No.

###### Will enabling / using this feature result in introducing new API types?

No.

###### Will enabling / using this feature result in any new calls to the cloud provider?

No.

###### Will enabling / using this feature result in increasing size or count of the existing API objects?

Yes, a single new string field. We can limit this to a reasonable size (say 256 characters) if desired.

###### Will enabling / using this feature result in increasing time taken by any operations covered by existing SLIs/SLOs?

No.

###### Will enabling / using this feature result in non-negligible increase of resource usage (CPU, RAM, disk, IO, ...) in any components?

No.

### Troubleshooting

###### How does this feature react if the API server and/or etcd is unavailable?

No changes from existing leader election.

###### What are other known failure modes?

No changes from existing leader election.

###### What steps should be taken if SLOs are not being met to determine the problem?

N/A

## Implementation History

* Implementation PR: https://github.com/kubernetes/kubernetes/pull/103442/files

## Drawbacks


## Alternatives

Controllers can fork client-go and use non-Lease types (which do not have a strongly typed API).

## Infrastructure Needed (Optional)

None.
