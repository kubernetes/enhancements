
# KEP-85: Graduate PodDisruptionBudget to stable

<!-- toc -->
- [Release Signoff Checklist](#release-signoff-checklist)
- [Summary](#summary)
- [Motivation](#motivation)
  - [Goals](#goals)
- [Proposal](#proposal)
  - [Implementation Details/Notes/Constraints](#implementation-detailsnotesconstraints)
    - [Promote Eviction to policy/v1 without breaking pods/eviction support for policy/v1beta1](#promote-eviction-to-policyv1-without-breaking-podseviction-support-for-policyv1beta1)
    - [Mutable PDBs](#mutable-pdbs)
    - [Eviction of non-ready pods](#eviction-of-non-ready-pods)
    - [Make the disruption controller more lenient for pods belonging to non-scale controllers](#make-the-disruption-controller-more-lenient-for-pods-belonging-to-non-scale-controllers)
    - [Address scalability issues with the disruption controller](#address-scalability-issues-with-the-disruption-controller)
    - [Fix handling of empty selector in disruption controller](#fix-handling-of-empty-selector-in-disruption-controller)
  - [API changes](#api-changes)
    - [PodDisruptionBudget v1 API](#poddisruptionbudget-v1-api)
  - [Test Plan](#test-plan)
    - [Existing Tests](#existing-tests)
    - [Conformance tests](#conformance-tests)
  - [Graduation Criteria](#graduation-criteria)
- [Production Readiness Review Questionnaire](#production-readiness-review-questionnaire)
  - [Feature Enablement and Rollback](#feature-enablement-and-rollback)
  - [Rollout, Upgrade and Rollback Planning](#rollout-upgrade-and-rollback-planning)
  - [Monitoring Requirements](#monitoring-requirements)
  - [Dependencies](#dependencies)
  - [Scalability](#scalability)
  - [Troubleshooting](#troubleshooting)
- [Implementation History](#implementation-history)
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

<!--
**Note:** This checklist is iterative and should be reviewed and updated every time this enhancement is being considered for a milestone.
-->

[kubernetes.io]: https://kubernetes.io/
[kubernetes/enhancements]: https://git.k8s.io/enhancements
[kubernetes/kubernetes]: https://git.k8s.io/kubernetes
[kubernetes/website]: https://git.k8s.io/website

## Summary

[Pod Disruption Budget (PDB)](https://kubernetes.io/docs/tasks/run-application/configure-pdb/)
is a Kubernetes API that limits the number of pods of a collection that are down simultaneously from voluntary disruptions.
[Kubernetes eviction API](https://kubernetes.io/docs/tasks/administer-cluster/safely-drain-node/#the-eviction-api)
takes PDB into account when terminating pods. If PDB is
violated, the eviction API returns failure and does not delete the requested pod.
The feature was introduced in Kubernetes 1.4 and promoted to beta in 1.5.
It has been in beta for a long time. This document lays out the plan to promote
it to stable.

## Motivation

PDB API has been stable and is an important feature that allows users to improve
reliability of their critical workloads. This feature has been in beta for a
long time. We need to promote to stable version given that we plan to support it
long term.

### Goals

* Plan to promote PDB API to stable version.
* Address some open issues with PDBs

## Proposal

### Implementation Details/Notes/Constraints

#### Promote Eviction to policy/v1 without breaking pods/eviction support for policy/v1beta1

Eviction is part of policy/v1beta1, but because it is a subresource of the v1 Pod API,
support for accepting policy/v1beta1 requests should not be dropped.

Luckily, the endpoint only supports Create and returns v1 Status,
so it is possible to let the current endpoint accept both policy/v1 and policy/v1beta1 Evictions.

The following changes will be made:

 * The decoding stack will be adjusted to allow a REST handler to accept multiple GroupVersionKinds
 * Discovery documents will indicate that policy/v1 Eviction kinds are accepted
 * client-go will add Eviction v1 and v1beta1 methods
 * `kubectl drain` will use v1 Eviction if available and fall back to v1beta1 Eviction
 * The Eviction subresource handler will accept policy/v1 and policy/v1beta1 Eviction objects
 * Integration tests will be added to ensure:
   * Get requests continue to be unsupported for this endpoint
   * Patch requests continue to be unsupported for this endpoint
   * Create requests continue to accept policy/v1 and policy/v1beta1 requests and return Status objects

#### Mutable PDBs

A mutable PDB object allows its `MinAvailable`, `MaxUnavailable`, and `Selector`
fields to be modified by clients. Components that use PDB must watch such
modifications and use the updated values when making decisions.

This feature is implemented by [this PR](https://github.com/kubernetes/kubernetes/pull/69867).

#### Eviction of non-ready pods

There are a couple of open issues where pods can't be evicted, even if they not
Ready and Running (https://github.com/kubernetes/kubernetes/issues/72320 and https://github.com/kubernetes/kubernetes/issues/80389).
The root of this issue, is that the rules in the disruption controller for what
is a healthy pod, and the rules in the Eviction API for when a pod can be 
evicted without looking at the PDB are not the same. This means a pod can be
considered unhealthy by the disruption controller so it does not count as
healthy when computing `DisruptionsAllowed`, but will still require
`DisruptionsAllowed` to be larger than 0 for it to be evicted. Some strange
situations can arise from this. For example if we have a PDB with 
`MinAvailable  = 1` and 10 pods that are all in the CrashLoop state (`Running`,
but not `Ready`), we will not be allowed to evict any of the pods.

This issue has been addressed with https://github.com/kubernetes/kubernetes/pull/94381.
It allows eviction of pods that are `Running` but not `Ready` as long as there
is already sufficient healthy pods to satisfy the PDB requirements. Relaxing
the rules further to allow eviction of these pods even if there aren't enough
healthy pods have been discussed, but deemed not safe (details in the discussion
on the PR).

#### Make the disruption controller more lenient for pods belonging to non-scale controllers

The disruption controller is currently taking the very safe route whenever it
encounters any issues with the targeted pods or their controllers. For all
configurations of the PDB, except when `minAvailable` is a number, the PDB
requires that it can find the controller and that the controller implements
scale (either by being one of the built-in workloads or a CR where the CRD
implements the scale subresource). If those conditions are not met for all pods
targeted by the PDB, the disruption controller will set `DisruptionsAllowed` to
0, which means none of the pods can be evicted. There is an issue concerning
this behavior: https://github.com/kubernetes/kubernetes/issues/77383.

The current behavior of the disruption controller for the different types of
input and the different types of pods that might be encountered are documented in: 
https://docs.google.com/spreadsheets/d/12HUundBS-slA6axfQYZPRCeIu_Au_wsGD0Vu_oKAnM8/edit?usp=sharing

This has been addressed by improving the users' visibility into any issues 
encountered by the disruption controller, primarily through the addition of 
conditions to the PDB status: https://github.com/kubernetes/kubernetes/pull/98127.
We also improve the error message provided to users in the case where a controller
resource can not be found with the scale client: https://github.com/kubernetes/kubernetes/pull/98346

We considered changing the current behavior of setting `DisruptionsAllowed` to zero,
but decided against it as it creates issues for the Eviction API and it makes it
harder to understand the behavior of the disruption controller.

#### Address scalability issues with the disruption controller

The disruption controller has some performance issues as reported in 
https://github.com/kubernetes/kubernetes/issues/92826.
https://github.com/kubernetes/kubernetes/pull/92827 was merged to remove the 30s
resync period which should improve performance. The frequency at which the
disruption controller creates events has also been reduced.

#### Fix handling of empty selector in disruption controller

The disruption controller doesn't handle empty selector correctly
https://github.com/kubernetes/kubernetes/issues/95083. We can't fix this
directly in the v1beta1 version of PDBs, but we should fix this as part of
promoting PDBs to v1 using the approach described in
https://github.com/kubernetes/kubernetes/issues/95083#issuecomment-703723763

Fixed as part of https://github.com/kubernetes/kubernetes/pull/99290

### API changes

* Add conditions to the status object for PodDisruptionBudget.
* Change the semantics of an empty selector.

#### PodDisruptionBudget v1 API

```golang
// PodDisruptionBudget is an object to define the max disruption that can be caused to a collection of pods
type PodDisruptionBudget struct {
	metav1.TypeMeta
	// +optional
	metav1.ObjectMeta

	// Specification of the desired behavior of the PodDisruptionBudget.
	// +optional
	Spec PodDisruptionBudgetSpec
	// Most recently observed status of the PodDisruptionBudget.
	// +optional
	Status PodDisruptionBudgetStatus
}

// PodDisruptionBudgetList is a collection of PodDisruptionBudgets.
type PodDisruptionBudgetList struct {
    metav1.TypeMeta
    // +optional
    metav1.ListMeta
    Items []PodDisruptionBudget
}

// PodDisruptionBudgetSpec is a description of a PodDisruptionBudget.
type PodDisruptionBudgetSpec struct {
    // An eviction is allowed if at least "minAvailable" pods selected by
    // "selector" will still be available after the eviction, i.e. even in the
    // absence of the evicted pod.  So for example you can prevent all voluntary
    // evictions by specifying "100%".
    // +optional
    MinAvailable *intstr.IntOrString

    // Label query over pods whose evictions are managed by the disruption
    // budget.
    // +optional
    Selector *metav1.LabelSelector

    // An eviction is allowed if at most "maxUnavailable" pods selected by
    // "selector" are unavailable after the eviction, i.e. even in absence of
    // the evicted pod. For example, one can prevent all voluntary evictions
    // by specifying 0. This is a mutually exclusive setting with "minAvailable".
    // +optional
    MaxUnavailable *intstr.IntOrString
}

// PodDisruptionBudgetStatus represents information about the status of a
// PodDisruptionBudget. Status may trail the actual state of a system.
type PodDisruptionBudgetStatus struct {
    // Most recent generation observed when updating this PDB status. DisruptionsAllowed and other
    // status information is valid only if observedGeneration equals to PDB's object generation.
    // +optional
    ObservedGeneration int64

    // DisruptedPods contains information about pods whose eviction was
    // processed by the API server eviction subresource handler but has not
    // yet been observed by the PodDisruptionBudget controller.
    // A pod will be in this map from the time when the API server processed the
    // eviction request to the time when the pod is seen by PDB controller
    // as having been marked for deletion (or after a timeout). The key in the map is the name of the pod
    // and the value is the time when the API server processed the eviction request. If
    // the deletion didn't occur and a pod is still there it will be removed from
    // the list automatically by PodDisruptionBudget controller after some time.
    // If everything goes smooth this map should be empty for the most of the time.
    // Large number of entries in the map may indicate problems with pod deletions.
    // +optional
    DisruptedPods map[string]metav1.Time

    // Number of pod disruptions that are currently allowed.
    DisruptionsAllowed int32

    // current number of healthy pods
    CurrentHealthy int32

    // minimum desired number of healthy pods
    DesiredHealthy int32

    // total number of pods counted by this disruption budget
    ExpectedPods int32

    // Represents the latest available observations of a pdb's current state.
    Conditions []metav1.Condition
}
```

### Test Plan

#### Existing Tests
PodDisruptionBudget currently has tests in various components that use the feature:

* Unit tests for disruption controller
  - https://github.com/kubernetes/kubernetes/blob/master/pkg/controller/disruption/disruption_test.go
* Integration tests for disruption controller
  - https://github.com/kubernetes/kubernetes/blob/master/test/integration/disruption/disruption_test.go
* Kubectl
  - https://github.com/kubernetes/kubernetes/blob/master/pkg/kubectl/generate/versioned/pdb_test.go
  tests generation of a PDB objects out of given parameters
  - https://github.com/kubernetes/kubernetes/blob/master/pkg/kubectl/cmd/create/create_pdb_test.go
  tests creation of PDB objects from cmd parameters
* Scheduler
  - https://github.com/kubernetes/kubernetes/blob/4b59044b8d2a3502ea490ba2c958008a098511a3/test/integration/scheduler/preemption_test.go#L1053
  tests effects of PDB on preemption (PDB is honored in a best effort way)
* Eviction integration tests
  - https://github.com/kubernetes/kubernetes/blob/master/test/integration/evictions/evictions_test.go test eviction logic and its interactions with PDB.
* Autoscaler
  - https://github.com/kubernetes/kubernetes/blob/master/test/e2e/autoscaling/cluster_size_autoscaling.go ensure that Autoscaler respects PDB when draining nodes.
* Integration tests for disruption controller
  - https://github.com/kubernetes/kubernetes/blob/master/test/e2e/apps/disruption.go

All beta PDB endpoints are covered by e2e tests: https://apisnoop.cncf.io/1.20.0/beta/policy

New tests will be added to test the planned changes. This includes:
* Verify that an empty selector has the correct semantics
* Verify that pods with incompatible controllers are handled correctly and that
events and conditions are set appropriately:
  * Controllers that doesn't implement the scale subresource are always ok if `MinAvailable`
  is an integer.
  * In all other situations (`MinAvailable` is a percentage or `MaxUnavailable` is used)
  any pods with controllers that doesn't implement the scale subresource should be ignored,
  `allowedDisruption` should be computed for the remaining pods with a compatible controller with
  conditions and events set appropriately.

#### Conformance tests

The following e2e tests will be included in the conformance tests:

 * DisruptionController
   * should block an eviction until the PDB is updated to allow it
   * create a PodDisruptionBudget
   * delete collection of PodDisruptionBudget
   * delete a PodDisruptionBudget
   * list or watch objects of kind PodDisruptionBudget
   * partially update the specified PodDisruptionBudget
   * partially update status of the specified PodDisruptionBudget
   * read the specified PodDisruptionBudget
   * read status of the specified PodDisruptionBudget
   * replace the specified PodDisruptionBudget
   * replace status of the specified PodDisruptionBudget

### Graduation Criteria

Graduation to GA:
- [x] Implement Mutable PDBs
- [x] Address performance issues
- [x] Pass conformance tests
- [x] Update documents to reflect the changes

## Production Readiness Review Questionnaire

### Feature Enablement and Rollback

* **How can this feature be enabled / disabled in a live cluster?**
  - [x] Feature gate (also fill in values in `kep.yaml`)
    - Feature gate name: PodDisruptionBudget
    - Components depending on the feature gate: kube-controller-manager, scheduler

* **Does enabling the feature change any default behavior?**
  No

* **Can the feature be disabled once it has been enabled (i.e. can we roll back
  the enablement)?**
  Yes, but any existing PDBs will no longer be updated and the old state might
  still block eviction of pods. Deleting a PDB from the cluster will remove any
  restrictions on evictions for the pods covered by the selector.

* **What happens if we reenable the feature if it was previously rolled back?** 
  PDBs in the cluster will be updated based on the current state of pods in the
  cluster and the pods will again be covered against undesired disruption.

* **Are there any tests for feature enablement/disablement?**
  Not applicable, given we're missing framework allowing switching feature-gates during e2e.

### Rollout, Upgrade and Rollback Planning

* **How can a rollout fail? Can it impact already running workloads?**
  - A panic in the disruption controller will crash the controller-manager

* **What specific metrics should inform a rollback?**
  - Unexpected controller-manager crashes

* **Were upgrade and rollback tested? Was the upgrade->downgrade->upgrade path tested?**
  Manual test is planned once the implementation is finished.

* **Is the rollout accompanied by any deprecations and/or removals of features, APIs, 
fields of API types, flags, etc.?**
  No

### Monitoring Requirements

* **How can an operator determine if the feature is in use by workloads?**
  - Check number of PDBs in cluster with `etcd_object_counts{resource="poddisruptionbudgets.policy"}`
  - Queue-related metrics to make sure the controller is functioning.

* **What are the SLIs (Service Level Indicators) an operator can use to determine 
the health of the service?**
  - [x] Metrics
    - Metric name: `workqueue_depth{name="disruption"}`
    - Metric name: `workqueue_retries_total{name="disruption"}`
    - Metric name: `workqueue_adds_total{name="disruption"}`
    - Components exposing the metric: `kube-controller-manager`

* **What are the reasonable SLOs (Service Level Objectives) for the above SLIs?**
  99th percentile of `workqueue_depth{name="disruption"}` <= X per cluster-day.

* **Are there any missing metrics that would be useful to have to improve observability 
of this feature?**
  No

### Dependencies

* **Does this feature depend on any specific services running in the cluster?**
  No

### Scalability

* **Will enabling / using this feature result in any new API calls?** 
  For graduation to GA, there are no new api calls.
  
  The PDB functionality overall has the following interaction with the API:
   * Controller is watching pods and PDBs. Each change for a pod will trigger a 
     reconcile for the PDB that is selecting the pod.
   * One every PDB reconcile, every pod selected by the PDB will be looked up 
     using the lister. If scale is needed, the controller for each pod is looked
     up to determine the scale. The well-known controllers that implement the
     scale subresource (Deployment, Replicaset, StatefulSet, ReplicaController) 
     are looked up through the lister. If needed the controller is looked up
     through the scale subresource (which can not be done through the lister).
     Each controller is only looked up once for each reconcile loop.
   * The PDB is updated based on the information from the pods. PDBs are only
     updated if some of the information has actually changed.

* **Will enabling / using this feature result in introducing new API types?**
  Graduating PDB API to GA. PodDisruptionBudget in `policy/v1`

* **Will enabling / using this feature result in any new calls to the cloud 
provider?**
  No

* **Will enabling / using this feature result in increasing size or count of 
the existing API objects?**
  We are adding conditions, so that will lead to a small increase in the size
  of the PDB resource.

* **Will enabling / using this feature result in increasing time taken by any 
operations covered by [existing SLIs/SLOs]?**
  No

* **Will enabling / using this feature result in non-negligible increase of 
resource usage (CPU, RAM, disk, IO, ...) in any components?**
  No

### Troubleshooting

The Troubleshooting section currently serves the `Playbook` role. We may consider
splitting it into a dedicated `Playbook` document (potentially with some monitoring
details). For now, we leave it here.


* **How does this feature react if the API server and/or etcd is unavailable?**
  The controller will not work, but in this case evictions will also be unavailable.

* **What are other known failure modes?**
  None

* **What steps should be taken if SLOs are not being met to determine the problem?**
  TBD

[supported limits]: https://git.k8s.io/community//sig-scalability/configs-and-limits/thresholds.md
[existing SLIs/SLOs]: https://git.k8s.io/community/sig-scalability/slos/slos.md#kubernetes-slisslos

## Implementation History

- PodDisruptionBudget was introduced in Kubernetes 1.4 as an alpha version.
- PodDisruptionBudget was graduated to beta in Kubernetes 1.5.
- PodDisruptionBudget was graduated to GA in Kubernetes 1.21.
- Eviction subresource was graduated to GA in Kubernetes 1.22.
