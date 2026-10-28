# KEP-6265: Surface Node Lifecycle State for Blocked Rollouts

<!-- toc -->
- [Release Signoff Checklist](#release-signoff-checklist)
- [Summary](#summary)
- [Motivation](#motivation)
  - [Goals](#goals)
  - [Non-Goals](#non-goals)
- [Proposal](#proposal)
  - [User Stories (Optional)](#user-stories-optional)
    - [Story 1 (Optional)](#story-1-optional)
    - [Story 2 (Optional)](#story-2-optional)
  - [Notes/Constraints/Caveats (Optional)](#notesconstraintscaveats-optional)
  - [Risks and Mitigations](#risks-and-mitigations)
- [Design Details](#design-details)
  - [Test Plan](#test-plan)
      - [Prerequisite testing updates](#prerequisite-testing-updates)
      - [Unit tests](#unit-tests)
      - [Integration tests](#integration-tests)
      - [e2e tests](#e2e-tests)
  - [Graduation Criteria](#graduation-criteria)
    - [Alpha](#alpha)
    - [Beta](#beta)
    - [GA](#ga)
    - [Deprecation](#deprecation)
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
  - [ ] (R) Ensure GA e2e tests meet requirements for [Conformance Tests](https://github.com/kubernetes/community/blob/master/contributors/devel/sig-architecture/conformance-tests.md)
  - [ ] (R) Minimum Two Week Window for GA e2e tests to prove flake free
- [ ] (R) Graduation criteria is in place
  - [ ] (R) [all GA Endpoints](https://github.com/kubernetes/community/pull/1806) must be hit by [Conformance Tests](https://github.com/kubernetes/community/blob/master/contributors/devel/sig-architecture/conformance-tests.md) within one minor version of promotion to GA
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

StatefulSets preserve stable Pod identity and roll out Pods in order, so the
controller cannot create a replacement Pod with the same identity while the
old Pod still exists. When a Pod is unavailable because its Node is shutting
down, unreachable, or under maintenance, this can block the rollout with no
indication of why. This KEP proposes surfacing Node lifecycle state (as
reported by [KEP-5683: Node Lifecycle Conditions](/keps/sig-node/5683-lifecycle-conditions))
in StatefulSet status, so it's clear when a rollout is waiting on Node
lifecycle activity rather than the new revision, storage attachment, or
scheduling.

## Motivation

A single Pod stuck on a Node affected by lifecycle activity (e.g. a Node
condition such as `MaintenanceInProgress`) can halt StatefulSet rollout
progress. Today, users and rollout tooling can only see that the
StatefulSet is waiting for Pods to become ready, with no way to tell whether
that's caused by the new revision, storage attachment, scheduling, or
intentional Node maintenance.

### Goals

- Report, in StatefulSet status, when a StatefulSet Pod is unavailable due
  to Node lifecycle state rather than another cause.

### Non-Goals

- Changing StatefulSet rollout, eviction, or Pod replacement behavior.
- ReplicaSet or other controller coordination with Node lifecycle state
  (tracked separately, see KEP-5683).

## Proposal

Extend StatefulSet status reporting so that when a Pod blocking a rollout is
on a Node reporting a lifecycle condition (e.g. `MaintenanceInProgress`),
that context is visible on the StatefulSet, instead of only a generic
"waiting for Pods" signal.

### User Stories (Optional)

- As a cluster operator, when a StatefulSet rollout stalls, I want to know
  whether it's blocked by a Node undergoing maintenance so I don't
  mistakenly treat it as an application or storage problem.

#### Story 1 (Optional)

#### Story 2 (Optional)

### Notes/Constraints/Caveats (Optional)

This KEP depends on Node lifecycle conditions defined in
[KEP-5683](/keps/sig-node/5683-lifecycle-conditions) being available for the
StatefulSet controller to read.

### Risks and Mitigations

TBD.

## Design Details

We add a new condition to `StatefulSetStatus.Conditions`, set when a Pod
blocking the rollout is on a Node reporting `MaintenanceInProgress`:

```yaml
- type: RolloutBlockedByNodeLifecycle
  status: "True"
  reason: NodeMaintenanceInProgress
  message: "pod web-2 is on node n1 reporting MaintenanceInProgress"
```

### Test Plan

##### Prerequisite testing updates

##### Unit tests

##### Integration tests

##### e2e tests

### Graduation Criteria

TBD.

#### Alpha

#### Beta

#### GA

#### Deprecation

### Upgrade / Downgrade Strategy

### Version Skew Strategy

## Production Readiness Review Questionnaire

### Feature Enablement and Rollback

###### How can this feature be enabled / disabled in a live cluster?

###### Does enabling the feature change any default behavior?

###### Can the feature be disabled once it has been enabled (i.e. can we roll back the enablement)?

###### What happens if we reenable the feature if it was previously rolled back?

###### Are there any tests for feature enablement/disablement?

### Rollout, Upgrade and Rollback Planning

###### How can a rollout or rollback fail? Can it impact already running workloads?

###### What specific metrics should inform a rollback?

###### Were upgrade and rollback tested? Was the upgrade->downgrade->upgrade path tested?

###### Is the rollout accompanied by any deprecations and/or removals of features, APIs, fields of API types, flags, etc.?

### Monitoring Requirements

###### How can an operator determine if the feature is in use by workloads?

###### How can someone using this feature know that it is working for their instance?

###### What are the reasonable SLOs (Service Level Objectives) for the enhancement?

###### What are the SLIs (Service Level Indicators) an operator can use to determine the health of the service?

###### Are there any missing metrics that would be useful to have to improve observability of this feature?

### Dependencies

###### Does this feature depend on any specific services running in the cluster?

### Scalability

###### Will enabling / using this feature result in any new API calls?

###### Will enabling / using this feature result in introducing new API types?

###### Will enabling / using this feature result in any new calls to the cloud provider?

###### Will enabling / using this feature result in increasing size or count of the existing API objects?

###### Will enabling / using this feature result in increasing time taken by any operations covered by existing SLIs/SLOs?

###### Will enabling / using this feature result in non-negligible increase of resource usage (CPU, RAM, disk, IO, ...) in any components?

###### Can enabling / using this feature result in resource exhaustion of some node resources (PIDs, sockets, inodes, etc.)?

### Troubleshooting

###### How does this feature react if the API server and/or etcd is unavailable?

###### What are other known failure modes?

###### What steps should be taken if SLOs are not being met to determine the problem?

## Implementation History

- 2026-09-27: Initial KEP created from [enhancement issue #6265](https://github.com/kubernetes/enhancements/issues/6265).

## Drawbacks

## Alternatives

## Infrastructure Needed (Optional)
