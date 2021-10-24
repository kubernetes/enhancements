# KEP-3017: Pod Healthy Policy for PDBs

<!-- toc -->
- [Release Signoff Checklist](#release-signoff-checklist)
- [Summary](#summary)
- [Motivation](#motivation)
  - [Goals](#goals)
  - [Non-Goals](#non-goals)
- [Proposal](#proposal)
  - [Risks and Mitigations](#risks-and-mitigations)
- [Design Details](#design-details)
  - [API](#api)
  - [Changes to the disruption controller](#changes-to-the-disruption-controller)
  - [Changes to the eviction API](#changes-to-the-eviction-api)
  - [Test Plan](#test-plan)
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
<!-- /toc -->

## Release Signoff Checklist

<!--
**ACTION REQUIRED:** In order to merge code into a release, there must be an
issue in [kubernetes/enhancements] referencing this KEP and targeting a release
milestone **before the [Enhancement Freeze](https://git.k8s.io/sig-release/releases)
of the targeted release**.

For enhancements that make changes to code or processes/procedures in core
Kubernetes—i.e., [kubernetes/kubernetes], we require the following Release
Signoff checklist to be completed.

Check these off as they are completed for the Release Team to track. These
checklist items _must_ be updated for the enhancement to be released.
-->

Items marked with (R) are required *prior to targeting to a milestone / release*.

- [x] (R) Enhancement issue in release milestone, which links to KEP dir in [kubernetes/enhancements] (not the initial KEP PR)
- [x] (R) KEP approvers have approved the KEP status as `implementable`
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

Pod Disruption Budgets currently doesn't provide a way for users to specify how to handle
pods that are Running, but not Ready. In this KEP, we add a new field `podHealthyPolicy` that
allows users to specify whether those pods should be considered healthy and therefore
covered by the constraints of the Pod Disruption Budget, or if they should be considered as
already disrupted and thus not covered by a Pod Disruption Budget. 

## Motivation

Pod Disruption Budgets are currently being used for two different purposes:
 * Provide best-effort constraints on voluntary disruption to preserve 
availability on a set of pods.
 * Prevent data-loss by blocking eviction of pods until any data unique to a
soon-to-be evicted pod has been copied/shared/replicated to other pod(s).

Both use-cases have rough edges with the current implementation. 

For users who only want to make sure a minimum number of pods are available, it is possible
to end up in situations where pods that are Running but not Ready can not be evicted,
even when the total number of pods are higher than the threshold set in the PDB
(https://github.com/kubernetes/kubernetes/issues/72320). This can block automated
tooling like the cluster-autoscaler and draining of nodes.

For users who leverage PDBs to prevent data-loss, the solution is unsafe (racey 
as described in https://github.com/kubernetes/kubernetes/pull/105296#issuecomment-929209150) and
arguably uses the API in a way it was not designed for.

The first use-case if the primary goal of PDBs, but feedback suggests that a sufficient
number of users are leveraging PDBs for the second use-case that changing the behavior
in a way that doesn't support this use-case is not an option. In particular, as Kubernetes
doesn't provide any alternatives solutions for this problem.

### Goals

- Prevent PDBs from deadlocking eviction due to non-Ready pods.
- Make sure users who rely on PDBs for data-safety can continue to do so.

### Non-Goals

- Providing a safe solution for preventing data-loss. Not because this isn't important, but 
it is unclear if PDB is the right tool for this.

## Proposal

The core issue here is whether a pod that is Running but not Ready is already disrupted, 
and thus can be evicted without being constrained by a potential Pod Disruption Budget. Currently,
the disruption controller only considers Running and Ready pods as healthy, thus that is 
the basis for computing `allowedDisruptions`. The disruption API on the other hand, does require
that `allowedDisruptions` is larger than 0 to evict a pod that is Running but not Ready. 

Adding a `podHealthyPolicy` field on the PDB API will allow the user to specify which
behavior that are desired, and this will be consistently handled by the disruption controller, eviction
API, and any other APIs that might use PDBs. If a `podHealthyPolicy` is not provided, the default
will be the current behavior.

### Risks and Mitigations

## Design Details

### API

```golang
// PodDisruptionBudgetSpec is a description of a PodDisruptionBudget.
type PodDisruptionBudgetSpec struct {
	
	...
	
	// PodHealthyPolicy defines the criteria for when the disruption controller
	// should consider a pod to be healthy.
	// If no policy is specified, the legacy behavior will be used. It means
	// only pods that are Running and Ready will be considered when the disruption
	// controller computes "disruptionsAllowed", but all pods in the Running phase
	// will be subject to the PDB on eviction.
	// +optional
	PodHealthyPolicy PodHealthyPolicy `json:"podHealthyPolicy,omitempty" protobuf:"bytes,4,opt,name=podHealthyPolicy"`
}

// PodHealthyPolicy defines the policy when a pod are considered healthy and therefore
// covered by a PodDisruptionBudget.
type PodHealthyPolicy string

const (
	// PodReady policy means that only pods that are both Running and Ready
	// will be considered healthy by the disruption controller. Any pods that
	// are not Ready are considered to already be disrupted and therefore will
	// not be counted when computing "disruptionsAllowed" and can be evicted
	// regardless of whether the criteria in a PDB is met.
	PodReady PodHealthyPolicy = "PodReady"
	
	// PodRunning policy means that pods that are in the Running phase
	// is considered healthy by the disruption controller, regardless of
	// whether they are Ready or not. Any pods that are in the Running
	// phase will be counted when computing "disruptionsAllowed" and
	// will be subject to the PDB for eviction.
	PodRunning PodHealthyPolicy = "PodRunning"
)
```

### Changes to the disruption controller

The disruption controller will be updated to use `podHealthyPolicy` to
determine how it should compute the value of `disruptionsAllowed`.

### Changes to the eviction API

The eviction API will be updated to use `podHealthyPolicy` to determine whether
a pod which is Running but not Ready can be evicted regardless of the value of
`disruptionsAllowed`. This will only be a behavioral change when users have specified
a `podHealthyPolicy`, and will not require the actual API to change.

### Test Plan

- Unit and integration tests covering:
  - The current behavior stays unchanged when the policy is not specified.
  - Correct behavior for both policies in both the disruption controller and
    the eviction API.
  - Feature gate disablement.
- Verify passing existing E2E and conformance tests for PDBs and Eviction.

### Graduation Criteria

#### Alpha

- Feature gate disabled by default.
- Unit and integration tests passing.

#### Beta

- Feature gate enabled by default.
- Existing E2E and conformance tests passing.

#### GA

- Every bug report is fixed.
- The disruption controller and the eviction API ignores the feature gate.

#### Deprecation

N/A

### Upgrade / Downgrade Strategy

No changes required for existing cluster to use the enhancement.

### Version Skew Strategy

This feature doesn't depend on the version for nodes.

## Production Readiness Review Questionnaire

### Feature Enablement and Rollback

###### How can this feature be enabled / disabled in a live cluster?

- [x] Feature gate (also fill in values in `kep.yaml`)
    - Feature gate name: PodDisruptionBudgetPodHealthyPolicy
    - Components depending on the feature gate:
      - kube-controller-manager
      - kube-apiserver
- [ ] Other
    - Describe the mechanism:
    - Will enabling / disabling the feature require downtime of the control
      plane?
    - Will enabling / disabling the feature require downtime or reprovisioning
      of a node? (Do not assume `Dynamic Kubelet Config` feature is enabled).

###### Does enabling the feature change any default behavior?

No, the behavior is only changed when users specify the `podHealthyPolicy` in
the PodDisruptionBudget spec.

###### Can the feature be disabled once it has been enabled (i.e. can we roll back the enablement)?

Yes, in that case the disruption controller and the eviction API will just
use the default behavior.

###### What happens if we reenable the feature if it was previously rolled back?

The disruption controller and the eviction API will again start using
the `PodHealthyPolicy` if provided on a PDB.

###### Are there any tests for feature enablement/disablement?

No, but they will be added for alpha.

### Rollout, Upgrade and Rollback Planning

###### How can a rollout or rollback fail? Can it impact already running workloads?

It does not change the default behavior. Users will have to specify a policy
on the PDB for behavior to be affected.

###### What specific metrics should inform a rollback?

Unexpected controller-manager crashes or significant changes in the latency or depth of the disruption controller
workqueue.

###### Were upgrade and rollback tested? Was the upgrade->downgrade->upgrade path tested?

A manual test will be performed, as follows:

1. Create a cluster in 1.23.
2. Upgrade to 1.24.
3. Create Deployment A and PDB A targeting the pods of Deployment A using the `PodReady` PodHealthyPolicy.
4. Downgrade to 1.23.
5. Verify that the disruption controller and eviction continue to work without using the PodHealthyPolicy.
6. Create another StatefulSet B and PDB B targeting the pods of StatefulSet B.
7. Upgrade to 1.24.
8. Verify that eviction of pods for Deployment A uses the `PodReady` PodHealthyPolicy and eviction of pods for
StatefulSet B uses the default behavior.

###### Is the rollout accompanied by any deprecations and/or removals of features, APIs, fields of API types, flags, etc.?

N/A

### Monitoring Requirements

<!--
This section must be completed when targeting beta to a release.
-->

###### How can an operator determine if the feature is in use by workloads?

<!--
Ideally, this should be a metric. Operations against the Kubernetes API (e.g.,
checking if there are objects with field X set) may be a last resort. Avoid
logs or events for this purpose.
-->

###### How can someone using this feature know that it is working for their instance?

<!--
For instance, if this is a pod-related feature, it should be possible to determine if the feature is functioning properly
for each individual pod.
Pick one more of these and delete the rest.
Please describe all items visible to end users below with sufficient detail so that they can verify correct enablement
and operation of this feature.
Recall that end users cannot usually observe component logs or access metrics.
-->

- [ ] Events
    - Event Reason:
- [ ] API .status
    - Condition name:
    - Other field:
- [ ] Other (treat as last resort)
    - Details:

###### What are the reasonable SLOs (Service Level Objectives) for the enhancement?

<!--
This is your opportunity to define what "normal" quality of service looks like
for a feature.

It's impossible to provide comprehensive guidance, but at the very
high level (needs more precise definitions) those may be things like:
  - per-day percentage of API calls finishing with 5XX errors <= 1%
  - 99% percentile over day of absolute value from (job creation time minus expected
    job creation time) for cron job <= 10%
  - 99.9% of /health requests per day finish with 200 code

These goals will help you determine what you need to measure (SLIs) in the next
question.
-->

###### What are the SLIs (Service Level Indicators) an operator can use to determine the health of the service?

<!--
Pick one more of these and delete the rest.
-->

- [ ] Metrics
    - Metric name:
    - [Optional] Aggregation method:
    - Components exposing the metric:
- [ ] Other (treat as last resort)
    - Details:

###### Are there any missing metrics that would be useful to have to improve observability of this feature?

<!--
Describe the metrics themselves and the reasons why they weren't added (e.g., cost,
implementation difficulties, etc.).
-->

### Dependencies

###### Does this feature depend on any specific services running in the cluster?

No

### Scalability

###### Will enabling / using this feature result in any new API calls?

No, both the disruption controller and the eviction API already fetch the
PDB from the API server.

###### Will enabling / using this feature result in introducing new API types?

No

###### Will enabling / using this feature result in any new calls to the cloud provider?

No

###### Will enabling / using this feature result in increasing size or count of the existing API objects?

- API: PodDisruptionBudget

  Estimated increase in size: New field of about 15B

###### Will enabling / using this feature result in increasing time taken by any operations covered by existing SLIs/SLOs?

No

###### Will enabling / using this feature result in non-negligible increase of resource usage (CPU, RAM, disk, IO, ...) in any components?

No

### Troubleshooting

<!--
This section must be completed when targeting beta to a release.

The Troubleshooting section currently serves the `Playbook` role. We may consider
splitting it into a dedicated `Playbook` document (potentially with some monitoring
details). For now, we leave it here.
-->

###### How does this feature react if the API server and/or etcd is unavailable?

No change from the existing behavior of the disruption controller and the eviction API.

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

- 2021-10-24: Proposed KEP for adding the new behavior in alpha status in 1.24.

## Drawbacks

If the current behavior is sufficient, we should not make this change. However,
the evidence is that it doesn't address the needs of users.

## Alternatives

Changing the default behavior was considered but rejected for two reasons:
* We can't change the behavior of a GA API
* There are two separate use-cases for this feature and changing the behavior
  to support only one of them would create problems for other users.

