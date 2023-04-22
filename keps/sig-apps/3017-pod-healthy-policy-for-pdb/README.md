# KEP-3017: Unhealthy Pod Eviction Policy for PDBs

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
  - [Changes to the eviction API](#changes-to-the-eviction-api)
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
- [Abandoned Alternative Implementation](#abandoned-alternative-implementation)
  - [Changes to the disruption controller](#changes-to-the-disruption-controller)
  - [Changes to the definition of healthy in a PDB according to the policy used.](#changes-to-the-definition-of-healthy-in-a-pdb-according-to-the-policy-used)
- [Future Work](#future-work)
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

Pod Disruption Budgets currently don't provide a way for users to specify how to handle
pods that are Running, but not Healthy (Ready).
In this KEP, we add a new field `unhealthyPodEvictionPolicy` that allows users to specify
what should happen to these not Healthy (Ready) pods. Whether they should be always evicted or kept
in case the application guarded by a Pod Disruption Budget is not available and disrupted.

## Motivation

Pod Disruption Budgets are currently being used for two different purposes:
 * Provide best-effort constraints on voluntary disruption to preserve 
availability on a set of pods.
 * Prevent data-loss by blocking eviction of pods until any data unique to a
soon-to-be evicted pod has been copied/shared/replicated to other pod(s).

Both use-cases have rough edges with the current implementation. 

For users who only want to make sure a minimum number of pods are available, it is possible
to end up in situations where pods that are Running but not Healthy (Ready) can not be evicted,
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

- Prevent PDBs from deadlocking eviction due to non-Healthy (non-Ready) pods.
- Make sure users who rely on PDBs for data-safety can continue to do so.

### Non-Goals

- Providing a safe solution for preventing data-loss. Not because this isn't important, but 
it is unclear if PDB is the right tool for this.
- Allow customization of healthiness detection for pods guarded by a PodDisruptionBudget.

## Proposal

The core issue here is whether a pod that is Running but not Healthy (Ready) is considered disrupted,
and thus should be evicted without being potentially constrained by a Pod Disruption Budget.

Currently, we only allow evicting Running pods in case there are enough pods healthy
(`.status.currentHealthy` is at least equal to `.status.DesiredHealthy`).
This is to give the application best chance to achieve availability and prevent data loss
by disallowing disruption of starting pods that have not become Healthy (Ready yet).

We also want to allow unconditional eviction of Running pods for applications that do not have
such strict constraints. This will allow cluster administrators to evict misbehaving applications
that are guarded by a PDB and proceed with node drain.

Adding a `unhealthyPodEvictionPolicy` field on the PDB API will allow the user to specify which
behavior is desired. This will be consistently handled by the eviction API, and any other APIs
that might use PDBs.  If a `unhealthyPodEvictionPolicy` is not provided, the default will be
the current behavior.

The behavior for pods in Pending, Succeeded or Failed phase will stay the same and such pods will
always be considered for eviction.

### Risks and Mitigations

## Design Details

### API

```golang
// PodDisruptionBudgetSpec is a description of a PodDisruptionBudget.
type PodDisruptionBudgetSpec struct {
	
	...
    // UnhealthyPodEvictionPolicy defines the criteria for when unhealthy pods
    // should be considered for eviction. Current implementation considers healthy pods,
    // as pods that have status.conditions item with type="Ready",status="True".
    //
    // Valid policies are IfHealthyBudget and AlwaysAllow.
    // If no policy is specified, the default behavior will be used,
    // which corresponds to the IfHealthyBudget policy.
    //
    // Additional policies may be added in the future.
    // Clients making eviction decisions should disallow eviction of unhealthy pods
    // if they encounter an unrecognized policy in this field.
    UnhealthyPodEvictionPolicy *UnhealthyPodEvictionPolicyType `json:"unhealthyPodEvictionPolicy,omitempty" protobuf:"bytes,4,opt,name=unhealthyPodEvictionPolicy"`
}

// UnhealthyPodEvictionPolicyType defines the criteria for when unhealthy pods
// should be considered for eviction.
// +enum
type UnhealthyPodEvictionPolicyType string

const (
    // IfHealthyBudget policy means that running pods (status.phase="Running"),
    // but not yet healthy can be evicted only if the guarded application is not
    // disrupted (status.currentHealthy is at least equal to status.desiredHealthy).
    // Healthy pods will be subject to the PDB for eviction.
    IfHealthyBudget UnhealthyPodEvictionPolicyType = "IfHealthyBudget"

    // AlwaysAllow policy means that all running pods (status.phase="Running"),
    // but not yet healthy are considered disrupted and can be evicted regardless
    // of whether the criteria in a PDB is met. This means perspective running
    // pods of a disrupted application might not get a chance to become healthy.
    // Healthy pods will be subject to the PDB for eviction.
    AlwaysAllow UnhealthyPodEvictionPolicyType = "AlwaysAllow"
)
```

### Changes to the eviction API

The eviction API will be updated to use `unhealthyPodEvictionPolicy` of a PDB to determine
whether a pod which is Running but not Ready can be evicted regardless of the value of
`disruptionsAllowed`. This will only be a behavioral change when users have specified
a `unhealthyPodEvictionPolicy`, and will not require the actual API to change.

### Test Plan

<!--
**Note:** *Not required until targeted at a release.*
The goal is to ensure that we don't accept enhancements with inadequate testing.
All code is expected to have adequate tests (eventually with coverage
expectations). Please adhere to the [Kubernetes testing guidelines][testing-guidelines]
when drafting this test plan.
[testing-guidelines]: https://git.k8s.io/community/contributors/devel/sig-testing/testing.md
-->

[x] I/we understand the owners of the involved components may require updates to
existing tests to make this code solid enough prior to committing the changes necessary
to implement this enhancement.

##### Prerequisite testing updates

<!--
Based on reviewers feedback describe what additional tests need to be added prior
implementing this enhancement to ensure the enhancements have also solid foundations.
-->

We assess that the eviction api has adequate test coverage for places which might be impacted by
this enhancement. Thus, no additional tests prior implementing this enhancement
are needed.

##### Unit tests

<!--
In principle every added code should have complete unit test coverage, so providing
the exact set of tests will not bring additional value.
However, if complete unit test coverage is not possible, explain the reason of it
together with explanation why this is acceptable.
-->

Unit tests covering:
  - The current behavior stays unchanged when the policy is not specified.
  - Correct behavior for both policies in the eviction API.
  - Feature gate disablement.

<!--
Additionally, for Alpha try to enumerate the core package you will be touching
to implement this enhancement and provide the current unit coverage for those
in the form of:
- <package>: <date> - <current test coverage>
The data can be easily read from:
https://testgrid.k8s.io/sig-testing-canaries#ci-kubernetes-coverage-unit
This can inform certain test coverage improvements that we want to do before
extending the production code to implement this enhancement.
-->

The core packages (with their unit test coverage) which are going to be modified during the implementation:
- `k8s.io/kubernetes/pkg/apis/policy/validation`: `5 October 2022` - `93%`  <!--(validation of the PodDisruptionBudget configuration with regard to the unhealthyPodEvictionPolicy)-->
- `k8s.io/kubernetes/pkg/apis/policy/v1`: `5 October 2022` - `60%` <!--(extension of PodDisruptionBudgetSpec)-->
- `k8s.io/kubernetes/pkg/registry/policy/poddisruptionbudget`: `8 November 2022` - `62.5%` <!--(create/update logic)-->
- `k8s.io/kubernetes/pkg/registry/core/pod/storage`: `8 November 2022` - `74.2%` <!--(eviction logic)-->

Alpha implementation:
- `k8s.io/kubernetes/pkg/apis/policy/validation`: `7 December 2022` - `93.1%`  <!--(validation of the PodDisruptionBudget configuration with regard to the unhealthyPodEvictionPolicy)-->
- `k8s.io/kubernetes/pkg/apis/policy/v1`: `7 December 2022` - `60%` <!--(extension of PodDisruptionBudgetSpec)-->
- `k8s.io/kubernetes/pkg/registry/policy/poddisruptionbudget`: `7 December 2022` - `75%` <!--(create/update logic)-->
- `k8s.io/kubernetes/pkg/registry/core/pod/storage`: `7 December 2022` - `78%` <!--(eviction logic)-->


##### Integration tests

Integration tests covering:
  - The current behavior stays unchanged when the policy is not specified.
  - Correct behavior for both policies in the eviction API.
  - Feature gate disablement.


<!--
This question should be filled when targeting a release.
For Alpha, describe what tests will be added to ensure proper quality of the enhancement.
For Beta and GA, add links to added tests together with links to k8s-triage for those tests:
https://storage.googleapis.com/k8s-triage/index.html

- <test>: <link to test coverage>
-->

[TestEvictionWithUnhealthyPodEvictionPolicy](https://github.com/kubernetes/kubernetes/blob/c8010537913422cc221cdd784936ff99817f621c/test/integration/evictions/evictions_test.go#L417): https://storage.googleapis.com/k8s-triage/index.html?test=UnhealthyPodEvictionPolicy

##### e2e tests

Verify passing existing E2E and conformance tests for PDBs and Eviction.

<!--
This question should be filled when targeting a release.
For Alpha, describe what tests will be added to ensure proper quality of the enhancement.
For Beta and GA, add links to added tests together with links to k8s-triage for those tests:
https://storage.googleapis.com/k8s-triage/index.html
We expect no non-infra related flakes in the last month as a GA graduation criteria.


- <test>: <link to test coverage>
-->



### Graduation Criteria

#### Alpha

- Feature gate disabled by default.
- Unit and integration tests passing.

#### Beta

- Feature gate enabled by default.
- Integration test which exercises the functionality.
- We want to keep the `spec.unhealthyPodEvictionPolicy` field null by default when not specified.
  This should preserve the original behavior and behave the same as the `IfHealthyBudget` value.
  This should be tested and documented.
- manual test for upgrade->downgrade->upgrade path will be performed once 1.27 is released

#### GA

- Every bug report is fixed.
- Introduce E2E tests for this field and confirm their stability.
- The eviction API ignores the feature gate.

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
    - Feature gate name: PDBUnhealthyPodEvictionPolicy
    - Components depending on the feature gate:
      - kube-apiserver
- [ ] Other
    - Describe the mechanism:
    - Will enabling / disabling the feature require downtime of the control
      plane?
    - Will enabling / disabling the feature require downtime or reprovisioning
      of a node?

###### Does enabling the feature change any default behavior?

No, the behavior is only changed when users specify the `unhealthyPodEvictionPolicy` in
the PodDisruptionBudget spec.

###### Can the feature be disabled once it has been enabled (i.e. can we roll back the enablement)?

Yes, in that case the eviction API will just use the default behavior.

###### What happens if we reenable the feature if it was previously rolled back?

The eviction API will again start using the `unhealthyPodEvictionPolicy` if provided on a PDB.

###### Are there any tests for feature enablement/disablement?

- [TestPodDisruptionBudgetStrategy](https://github.com/kubernetes/kubernetes/blob/06914bdaf51fc1b91501c332bd69d439cd370581/pkg/registry/policy/poddisruptionbudget/strategy_test.go#L96-L114)

### Rollout, Upgrade and Rollback Planning

###### How can a rollout or rollback fail? Can it impact already running workloads?

Bugs could affect `/evictions` endpoint which would return server error in that case.
It cannot directly affect workloads, but could potentially cause node drain to stall,
which would have an effect on the cluster during an upgrade.

When the rollback occurs, existing filled `.spec.unhealthyPodEvictionPolicy` fields will be ignored
and the old eviction behavior will be enforced for these PDBs.

###### What specific metrics should inform a rollback?

Failing eviction requests could be an indicator. `apiserver_request_total{resource = "pods", subresource = "eviction"}` metric
can be observed to detect increased rate of failing evictions.

###### Were upgrade and rollback tested? Was the upgrade->downgrade->upgrade path tested?

A manual test was performed, as follows:

1. Create a cluster in 1.25.
2. Upgrade to 1.26.
3. Create Deployment A and PDB A targeting the pods of Deployment A using the `AlwaysAllow` UnhealthyPodEvictionPolicy.
4. Downgrade to 1.25.
5. Verify that the eviction continue to work without using the UnhealthyPodEvictionPolicy.
6. Create another StatefulSet B and PDB B targeting the pods of StatefulSet B.
7. Upgrade to 1.26.
8. Verify that eviction of pods for Deployment A and StatefulSet B use the default behavior.
   Verify that the `AlwaysAllow` UnhealthyPodEvictionPolicy can be set again to a PDB of Deployment A and test the eviction behavior

TODO:
A manual test will be performed, as follows:

1. Create a cluster in 1.26.
2. Upgrade to 1.27.
3. Create Deployment A and PDB A targeting the pods of Deployment A using the `AlwaysAllow` UnhealthyPodEvictionPolicy.
4. Downgrade to 1.26.
5. Verify that the eviction continue to work without using the UnhealthyPodEvictionPolicy (PDBUnhealthyPodEvictionPolicy feature gate disabled by default).
6. Create another StatefulSet B and PDB B targeting the pods of StatefulSet B.
7. Upgrade to 1.27.
8. Verify that eviction of pods for Deployment A uses the `AlwaysAllow` UnhealthyPodEvictionPolicy and eviction of pods for
   StatefulSet B uses the default behavior.

###### Is the rollout accompanied by any deprecations and/or removals of features, APIs, fields of API types, flags, etc.?

N/A

### Monitoring Requirements

###### How can an operator determine if the feature is in use by workloads?

By checking `.spec.unhealthyPodEvictionPolicy` field of the PodDisruptionBudget.
Pods belonging to this PDB should be evicted according to this policy.

###### How can someone using this feature know that it is working for their instance?

- [x] Other (treat as last resort)
    - Details: kube-apiserver logs and audit logs that track eviction requests can be examined to see
      if the `UnhealthyPodEvictionPolicy` feature is working properly.

###### What are the reasonable SLOs (Service Level Objectives) for the enhancement?

This feature should not have an impact on the eviction request latency or availability.
Eviction requests should follow the [existing latency SLOs](https://github.com/kubernetes/community/blob/master/sig-scalability/slos/slos.md#steady-state-slisslos)
for serving mutating or read-only API calls.

###### What are the SLIs (Service Level Indicators) an operator can use to determine the health of the service?

The following indicators should conform to the existing kube-apiserver SLIs.

- [x] Metrics
    - Metric name: apiserver_request_total
      - [Optional] Aggregation method: resource = "pods", subresource = "eviction"
      - Components exposing the metric: kube-apiserver
    - Metric name: apiserver_request_duration_seconds
      - [Optional] Aggregation method: resource = "pods", subresource = "eviction"
      - Components exposing the metric: kube-apiserver

###### Are there any missing metrics that would be useful to have to improve observability of this feature?

No

### Dependencies

###### Does this feature depend on any specific services running in the cluster?

No

### Scalability

###### Will enabling / using this feature result in any new API calls?

No, the eviction API already fetch the PDB from the API server.

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

No change from the existing behavior of the eviction API.

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
- 2022-11-11: Initial alpha implementation merged into 1.26
- 2022-12-07: KEP rewritten to match the implementation (PodHealthyPolicy was renamed to UnhealthyPodEvictionPolicy)
- 2023-02-06: Update for beta promotion

## Drawbacks

If the current behavior is sufficient, we should not make this change. However,
the evidence is that it doesn't address the needs of users.

## Alternatives

Changing the default behavior was considered but rejected for two reasons:
* We can't change the behavior of a GA API
* There are two separate use-cases for this feature and changing the behavior
  to support only one of them would create problems for other users.

## Abandoned Alternative Implementation

There is a noticeable difference to the original KEP as some behaviours were dropped.

### Changes to the disruption controller

We have removed changes to the disruption controller and computation of `disruptionsAllowed`.
We have kept the scope only to the eviction API. It is better to split these changes into separate features
in order to have simpler (less confusing), and more well-defined behavior for each feature.

You can see possible followups for customizing the definition of healthiness in [Future Work](#future-work)

### Changes to the definition of healthy in a PDB according to the policy used.

We have decided that eviction policy should not change the meaning of a healthy pod as a single powerful field
could introduce more confusion into how it affects the status of PodDisruptionBudget and Eviction API.

`PodRunning` policy was measuring running pods and changing the computation of `disruptionsAllowed`,
and it was removed from the original KEP.

```golang
const (
	// PodRunning policy means that pods that are in the Running phase
	// is considered healthy by the disruption controller, regardless of
	// whether they are Ready or not. Any pods that are in the Running
	// phase will be counted when computing "disruptionsAllowed" and
	// will be subject to the PDB for eviction.
	PodRunning PodHealthyPolicy = "PodRunning"
)
```

## Future Work

The current implementation considers healthy pods, as pods that have `.status.conditions` item with `type="Ready"` and `status="True"`.
These pods are tracked via `.status.currentHealthy` field in the PDB status.

This might not be enough for all use cases. For example the user might want to specifically handle pods that have their PVC
on a specific node's local storage. The pod should block the node from being drained and going down to prevent a possible data loss,
even in all situtations when the pod is not ready ([discussion](https://github.com/kubernetes/kubernetes/pull/105296#issuecomment-929503163))

To support this, a new custom mechanism for defining healthiness needs to be defined to optionally replace the default implementation.
This could be achieved with the help of user defined [Pod Readiness Gates](https://github.com/kubernetes/enhancements/blob/master/keps/sig-network/580-pod-readiness-gates/README.md),
by introducing a new field in a PodDisruptionBudget that could receive either a list of condition types or a logical expression
referencing these condition types to conclude whether the pod is healthy or not.
This field and other options should be explored in an additional KEP.

The disruption controller would update the existing fields in a PodDisruptionBudget status based on the custom healthiness.
The eviction API would react to the existing fields in the same way as it does now,
and in a combination with here proposed `PDBUnhealthyPodEvictionPolicy` feature.
