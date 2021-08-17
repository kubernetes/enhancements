# KEP-2255: ReplicaSet Pod Deletion Cost


<!-- toc -->
- [Release Signoff Checklist](#release-signoff-checklist)
- [Summary](#summary)
- [Motivation](#motivation)
  - [Goals](#goals)
  - [Non-Goals](#non-goals)
- [Proposal](#proposal)
  - [User Stories (optional)](#user-stories-optional)
    - [Story 1](#story-1)
    - [Story 2](#story-2)
  - [Risks and Mitigations](#risks-and-mitigations)
- [Design Details](#design-details)
  - [Test Plan](#test-plan)
  - [Graduation Criteria](#graduation-criteria)
    - [Alpha -&gt; Beta Graduation](#alpha---beta-graduation)
    - [Beta -&gt; GA Graduation](#beta---ga-graduation)
  - [Upgrade / Downgrade Strategy](#upgrade--downgrade-strategy)
  - [Version Skew Strategy](#version-skew-strategy)
- [Production Readiness Review Questionnaire](#production-readiness-review-questionnaire)
  - [Feature enablement and rollback](#feature-enablement-and-rollback)
  - [Rollout, Upgrade and Rollback Planning](#rollout-upgrade-and-rollback-planning)
  - [Monitoring requirements](#monitoring-requirements)
  - [Dependencies](#dependencies)
  - [Scalability](#scalability)
  - [Troubleshooting](#troubleshooting)
- [Implementation History](#implementation-history)
- [Alternatives](#alternatives)
<!-- /toc -->

## Release Signoff Checklist


Items marked with (R) are required *prior to targeting to a milestone / release*.

- [X] (R) Enhancement issue in release milestone, which links to KEP dir in [kubernetes/enhancements] (not the initial KEP PR)
- [X] (R) KEP approvers have approved the KEP status as `implementable`
- [X] (R) Design details are appropriately documented
- [X] (R) Test plan is in place, giving consideration to SIG Architecture and SIG Testing input
- [X] (R) Graduation criteria is in place
- [X] (R) Production readiness review completed
- [X] Production readiness review approved
- [X] "Implementation History" section is up-to-date for milestone
- [X] User-facing documentation has been created in [kubernetes/website], for publication to [kubernetes.io]
- [X] Supporting documentation e.g., additional design documents, links to mailing list discussions/SIG meetings, relevant PRs/issues, release notes


[kubernetes.io]: https://kubernetes.io/
[kubernetes/enhancements]: https://git.k8s.io/enhancements
[kubernetes/kubernetes]: https://git.k8s.io/kubernetes
[kubernetes/website]: https://git.k8s.io/website

## Summary

This feature allows applications to give a hint to the ReplicaSet controller 
as to which pods should be deleted first on scale down.

## Motivation

Currently ReplicaSets are scaled down based on a criteria that on the
limit prioritizes deleting pods with a more recent creation/readiness 
timestamp. This is not ideal for some applications where the cost of 
deleting pods is not related to how recent they were created. 

### Goals

- An API that allows applications to influence the order of deleting 
  pods when scaling down a ReplicaSet


### Non-Goals

- Guarantees on pod deletion order
- A controller that sets the cost of deleting the pods.

## Proposal

Define a known annotation, namely `controller.kubernetes.io/pod-deletion-cost` that 
applications can set to offer a hint on the cost of deleting a pod compared
to other pods belonging to the same ReplicaSet. 

### User Stories (optional)

#### Story 1

The different pods of an application could have different utilization levels. 
On scale down, the application may prefer to remove the pods with lower utilization.
To avoid frequently updating the pods, the application should update pod-deletion-cost
once before issuing a scale down. This works if the application itself controls the down 
scaling (e.g., the driver pod of a Spark deployment).

#### Story 2

On scale down, the application may want to remove pods running on the most expensive 
nodes first. For example, remove pods from nodes running on standard VMs first
then from ones running on preemptible/spot VMs (which can be 80% cheaper than standard VMs).


### Risks and Mitigations

- Users perceive the feature as a guarantee to delete order. Documentation
should stress the fact that this is best effort.

- Users deploy controllers that update the annotation frequently causing a
  significant load on the api server. Documentation should include best 
  practices as to how this feature should be used (e.g., update the
  pod-deletion-cost only before scale down). Moreover, [API priority and fairness](https://kubernetes.io/docs/concepts/cluster-administration/flow-control/) 
  gives operators a new server-side knob that allows them to limit update
  qps issued by such controllers.
  

## Design Details

The pod-deletion-cost range will be from [-MaxInt, MaxInt]. The default value is 0.
Invalid values (like setting the annotation to string) will be rejected by the api-server
with a BadRequest status code.

Having the default value in the middle of the range allows controllers to cutomize
the semantics of the cost of deleting pods that don't have the annotation set: 
controllers can use positive pod-deletion-cost values if they always want uninitialized
pods to be deleted first, or use negative pod-deletion-cost values if they want
uninitialized pods to always be deleted last.

When scaling down a ReplicaSet, controller-manager will prioritize deleting
pods with lower pod-deletion-cost. Specifically, the pod-deletion-cost will be evaluated after
step 3 and before step 4 as they are currently defined in
[ActivePodsWithRanks](https://github.com/kubernetes/kubernetes/blob/cac933934b1301665e6e51a81c66c483f4e16c49/pkg/controller/controller_utils.go#L784-L809),
which means the followig criteria is applied when comparing two pods regardless of their pod-deletion-cost:
- if one is assigned a node and the other is not, then the unassigned pod is deleted first.
- if the two pods are in different phases, then the pod in pending/unknown status is deleted first.
- if the two pods have different readiness status, then the not ready pod is deleted first


If none of the pods set the pod-deletion-cost annotation or all of them have the same value, then the 
scale down behavior is not changed compared to now.

### Test Plan

- Units test in kube-controller-manager package to test a variety of scenarios.
- Integration tests to validate that:
  - Replicas with lower pod-deletion-cost are deleted before replicas with higher pod-deletion-cost
  - No behavior change when pod-deletion-cost is not set or all pods have the same pod-deletion-cost

### Graduation Criteria

#### Alpha -> Beta Graduation
* Implemented feedback from alpha testers

#### Beta -> GA Graduation
* We're confident that no further API changes will be needed to achieve the goals of the KEP
* All known functional bugs have been fixed

### Upgrade / Downgrade Strategy

There is no strategy per se. On upgrade, controller-manager will start taking into account
pod-deletion-cost annotation for new and existing ReplicaSets that set the annotation. On 
downgrade, controller-manager will stop taking into account pod-deletion-cost, and so
reverting to old behavior.

### Version Skew Strategy

N/A

## Production Readiness Review Questionnaire

### Feature enablement and rollback

* **How can this feature be enabled / disabled in a live cluster?**
  - [x] Feature gate (also fill in values in `kep.yaml`)
    - Feature gate name: PodDeletionCost
    - Components depending on the feature gate: kube-controller-manager
  - [ ] Other
    - Describe the mechanism:
    - Will enabling / disabling the feature require downtime of the control
      plane?
    - Will enabling / disabling the feature require downtime or reprovisioning
      of a node? (Do not assume `Dynamic Kubelet Config` feature is enabled).


* **Does enabling the feature change any default behavior?**
 No.


* **Can the feature be disabled once it has been enabled (i.e. can we rollback
  the enablement)?**
Yes.


* **What happens if we reenable the feature if it was previously rolled back?**
It should continue to work as expected.


* **Are there any tests for feature enablement/disablement?**
Unit tests.

### Rollout, Upgrade and Rollback Planning

_This section must be completed when targeting beta graduation to a release._

* **How can a rollout fail? Can it impact already running workloads?**
It shouldn't impact already running workloads. This is an opt-in feature
since users need to explicitly set the annotation.

* **What specific metrics should inform a rollback?**
None.

* **Were upgrade and rollback tested? Was upgrade->downgrade->upgrade path tested?**
Manually tested, worked as expected.

* **Is the rollout accompanied by any deprecations and/or removals of features,
  APIs, fields of API types, flags, etc.?**
No.

### Monitoring requirements

_This section must be completed when targeting beta graduation to a release._

* **How can an operator determine if the feature is in use by workloads?**
  - Search for pod annotations with the exact same pod-cost annotation string.

* **What are the SLIs (Service Level Indicators) an operator can use to
  determine the health of the service?**
N/A

* **What are the reasonable SLOs (Service Level Objectives) for the above SLIs?**
N/A

* **Are there any missing metrics that would be useful to have to improve
  observability if this feature?**
No.

### Dependencies

_This section must be completed when targeting beta graduation to a release._

* **Does this feature depend on any specific services running in the cluster?**
No.

### Scalability

* **Will enabling / using this feature result in any new API calls?**
  - No, not the feature itself. However, users will want to deploy an external controller 
    that updates the pod-deletion-cost, documentation should stress that update frequency
    to be coarse grained.


* **Will enabling / using this feature result in introducing new API types?**
  - No.


* **Will enabling / using this feature result in any new calls to cloud
  provider?**
  - No.


* **Will enabling / using this feature result in increasing size or count
  of the existing API objects?**
  - No.

* **Will enabling / using this feature result in increasing time taken by any
  operations covered by [existing SLIs/SLOs][]?**
  - There are no SLOs covering scale down, but this feature should have negligible 
    impact on scale-down latency since we are adding an additional sorting key.


* **Will enabling / using this feature result in non-negligible increase of
  resource usage (CPU, RAM, disk, IO, ...) in any components?**
  - No.

### Troubleshooting

_This section must be completed when targeting beta graduation to a release._

* **How does this feature react if the API server and/or etcd is unavailable?**
  - The pod annotation can't be set. The normal pod deletion behavior will be used for non-annotated pods in a Deployment.
  
* **What are other known failure modes?**
  - None.

* **What steps should be taken if SLOs are not being met to determine the problem?**
  - N/A

[supported limits]: https://git.k8s.io/community//sig-scalability/configs-and-limits/thresholds.md
[existing SLIs/SLOs]: https://git.k8s.io/community/sig-scalability/slos/slos.md#kubernetes-slisslos

## Implementation History
- 2021-04-11: Promoted the feature to Beta
- 2021-01-13: Initial KEP submitted as provisional
- 2021-01-15: KEP promoted to implementable


## Alternatives

One alternative to using an annotation is adding an explicit API field. If the feature gets
enough traction, we may consider promoting the annotation to a Status field.


