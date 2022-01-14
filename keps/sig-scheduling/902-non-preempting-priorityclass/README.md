# Add NonPreempting Option For PriorityClasses

## Table of Contents

<!-- toc -->
- [Release Signoff Checklist](#release-signoff-checklist)
- [Summary](#summary)
- [Motivation](#motivation)
  - [Goals](#goals)
  - [Non-Goals](#non-goals)
- [Proposal](#proposal)
  - [User Stories](#user-stories)
  - [Risks and Mitigations](#risks-and-mitigations)
- [Design Details](#design-details)
  - [Testing Plan](#testing-plan)
  - [Graduation Criteria](#graduation-criteria)
    - [Alpha (v1.15):](#alpha-v115)
    - [Beta (v1.19):](#beta-v119)
    - [Stable (v1.24):](#stable-v124)
- [Production Readiness Review Questionnaire](#production-readiness-review-questionnaire)
  - [Feature enablement and rollback](#feature-enablement-and-rollback)
  - [Rollout, Upgrade and Rollback Planning](#rollout-upgrade-and-rollback-planning)
  - [Monitoring Requirements](#monitoring-requirements)
  - [Dependencies](#dependencies)
  - [Scalability](#scalability)
  - [Troubleshooting](#troubleshooting)
- [Implementation History](#implementation-history)
<!-- /toc -->

## Release Signoff Checklist

Items marked with (R) are required *prior to targeting to a milestone / release*.

- [x] (R) Enhancement issue in release milestone, which links to KEP dir in [kubernetes/enhancements] (not the initial KEP PR)
- [x] (R) KEP approvers have approved the KEP status as `implementable`
- [x] (R) Design details are appropriately documented
- [x] (R) Test plan is in place, giving consideration to SIG Architecture and SIG Testing input
- [x] (R) Graduation criteria is in place
- [x] (R) Production readiness review completed
- [ ] Production readiness review approved
- [x] "Implementation History" section is up-to-date for milestone
- [x] User-facing documentation has been created in [kubernetes/website], for publication to [kubernetes.io]
- [x] Supporting documentation e.g., additional design documents, links to mailing list discussions/SIG meetings, relevant PRs/issues, release notes

[kubernetes.io]: https://kubernetes.io/
[kubernetes/enhancements]: https://git.k8s.io/enhancements
[kubernetes/kubernetes]: https://git.k8s.io/kubernetes
[kubernetes/website]: https://git.k8s.io/website

## Summary

[PriorityClasses](https://kubernetes.io/docs/concepts/configuration/pod-priority-preemption/) are a GA feature as on 1.14,
which impact the scheduling and eviction of pods.
Pods are be scheduled according to descending priority.
If a pod cannot be scheduled due to insufficient resources,
lower-priority pods will be preempted to make room.

This proposal makes the preempting behavior optional for a PriorityClass,
by adding a new field to PriorityClasses,
which in turn populates PodSpec.
If a pod is waiting to be scheduled,
and it does not have preemption enabled,
it will not trigger preemption of other pods.

## Motivation

Allowing PriorityClasses to be non-preempting is important for running batch workloads.

Batch workloads typically have a backlog of work,
with unscheduled pods.
Higher-priority workloads can be assigned a higher priority via a PriorityClass,
but this may result in pods with partially-completed work being preempted.
Adding the non-preempting option allows users to prioritize the scheduling queue,
without discarding incomplete work.

### Goals

- Add a boolean flag to PriorityClasses,
to enable or disable preemption for pods of that PriorityClass.

### Non-Goals

* Protecting pods from preemption. PodDisruptionBudget should be used.

## Proposal

Add a Preempting field to both PodSpec and PriorityClass.
This field will default to true,
for backwards compatibility.

If Preempting is true for a pod,
the scheduler will preempt lower priority pods to schedule this pod,
as is current behavior.

If Preempting is false,
a pod of that priority will not preempt other pods.

Setting the Preempting field in PriorityClass provides a straightforward interface,
and allows ResourceQuotas to restrict preemption.

PriorityClass type example:
```
type PriorityClass struct {
	metav1.TypeMeta
	metav1.ObjectMeta
	Value int32
	GlobalDefault bool
	Description string
	Preempting *bool // New option
}
```

The Preempting field in PodSpec will be populated during pod admission,
similarly to how the PriorityClass Value is populated.
Storing the Preempting field in the pod spec has several benefits:
* The scheduler does not need to be aware of PiorityClasses,
as all relevant information is in the pod.
* Mutating PriorityClass objects does not impact existing pods.
* Kubelets can set Preempting on static pods.

PodSpec type example:
```
type PodSpec struct {
    ...
    Preempting *bool
    ...
}
```

This feature should be gated in alpha, provisionally under the gate `NonPreemptingPriority`.

Documentation must be updated to reflect the new feature,
and changes to PriorityClass/PodSpec fields.

### User Stories
A user is running batch workloads on a cluster.
The user has a high-priority job,
that they wish to schedule before other workloads in the queue.
As the user does not want to preempt running batch workloads and discard work,
the user creates the new workload with a high-priority,
non-preempting PriorityClass.
The new workload's pods are scheduled ahead of the queue,
without disrupting running workloads.

* Users are able to run preempting and non-preempting workloads in a stable manner,
and are not requesting additional changes.
* The feature has been stable and reliable in at least 2 releases.
* Adequate documentation exists for preemption and the optional field.
* Test coverage includes non-preempting use cases.
* Conformance requirements for non-preempting PriorityClasses are agreed upon.

### Risks and Mitigations

The new feature may malfuction,
or existing preemption functionality may be impaired.
New tests (covering both nonpreepting workloads and mixed workloads),
and the existing preempting PriorityClass tests should be used to prove stability.

## Design Details
### Testing Plan
Add detailed unit and integration tests for nonpreempting workloads.

Add basic e2e tests, to ensure all components are working together.

Ensure existing tests (for preempting PriorityClasses) do not break.

### Graduation Criteria
#### Alpha (v1.15):

- Support NonPreemptingPriority in PriorityClasses

#### Beta (v1.19):

- Add integration test for NonPreemptingPriority.
- Graduate NonPreemptingPriority to Beta.
- Update documents to reflect the changes.

#### Stable (v1.24):
- No negative feedback.
- Enhance the message of the existing event for scheduling failed to include details about preemption.
- Graduate NonPreemptingPriority to GA.
- Update documents to reflect the changes.

## Production Readiness Review Questionnaire

### Feature enablement and rollback
* **How can this feature be enabled / disabled in a live cluster?**
  - [x] Feature gate
    - Feature gate name: NonPreemptingPriority
    - Components depending on the feature gate: 
      - kube-apiserver 
      - kube-scheduler

* **Does enabling the feature change any default behavior?**
  No

* **Can the feature be disabled once it has been enabled (i.e. can we rollback
  the enablement)?**
  Yes. This feature can be disabled by restarting kube-apiserver and kube-scheduler with feature-gate turned off.

* **What happens if we reenable the feature if it was previously rolled back?**
  If we reenable the feature, the Pod with high priority and NonPreemptionPolicy will be eligible to preempt other pods with low priority when cluster resources are tight.

* **Are there any tests for feature enablement/disablement?**
  No

### Rollout, Upgrade and Rollback Planning
* **How can a rollout fail? Can it impact already running workloads?**
  If a rollout fails, kube-scheduler will keep crashing. Running workloads won't be affected by kube-scheduler.

* **What specific metrics should inform a rollback?**
Check the following indicators to determine if there are any exceptions:
  - pod_preemption_victims
  - total_preemption_attempts
  - scheduling_algorithm_preemption_evaluation_seconds

* **Were upgrade and rollback tested? Was upgrade->downgrade->upgrade path tested?**
  Manually tested successfully. The test environment version is v1.23. We tested enabling and disabling this
  feature. After each change in the feature-gate, 3 separate priorityclasses will be recreated (One
  high-priorityclass with preemptionPolicy as Never, other high-priorityclass with preemptionPolicy not be
  set, one low-priorityclass with preemptionPolicy not be set). Create multiple pods with the above 3
  priorityclasses to verify that the preemption results are as expected.

* **Is the rollout accompanied by any deprecations and/or removals of features?**
  N/A.

### Monitoring Requirements

<!--
This section must be completed when targeting beta to a release.
-->

###### How can an operator determine if the feature is in use by workloads?
The operator can determine if the workload is using the feature by checking if the priorityclass's preemptionPolicy is set to "Never".
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

- [x] Events
  - Event Reason: There is an event sent by kube-scheduler if the pod preempts other pods. If the feature is working and the pod with the priorityclass'preemptionPolicy as Never, there won't be a preemption related event for this pod.
- [ ] API .status
  - Condition name:
  - Other field:
- [x] Other (treat as last resort)
  - Details: Check if pods with preemptionPolicy set to Never can preempt other low-priority pods when the cluster resources cannot be met.  

###### What are the reasonable SLOs (Service Level Objectives) for the enhancement?
N/A

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

- [x] Metrics
  - Metric name: preemption_victims
  - [Optional] Aggregation method:
  - Components exposing the metric: kube-scheduler
- [ ] Other (treat as last resort)
  - Details:

###### Are there any missing metrics that would be useful to have to improve observability of this feature? 
We currently only have events that describe a pod being preempted by another pod. But we don't
have an event that describes why sometimes the preemption is not successful. We can enhance the
message of the existing event for scheduling failed to include details about preemption. This
will help us to improve observability for this feature and other scenarios.

In addition to events, we can add metrics about how many pods have stopped preempting other pods because of this no-preemption option. However, since the probability of this metric being used is likely to be small, it was not added.

<!--
Describe the metrics themselves and the reasons why they weren't added (e.g., cost,
implementation difficulties, etc.).
-->


### Dependencies

<!--
This section must be completed when targeting beta to a release.
-->

###### Does this feature depend on any specific services running in the cluster?
No.

<!--
Think about both cluster-level services (e.g. metrics-server) as well
as node-level agents (e.g. specific version of CRI). Focus on external or
optional services that are needed. For example, if this feature depends on
a cloud provider API, or upon an external software-defined storage or network
control plane.

For each of these, fill in the following—thinking about running existing user workloads
and creating new ones, as well as about cluster-level services (e.g. DNS):
  - [Dependency name]
    - Usage description:
      - Impact of its outage on the feature:
      - Impact of its degraded performance or high-error rates on the feature:
-->

### Scalability
* **Will enabling / using this feature result in any new API calls?**
  No

* **Will enabling / using this feature result in introducing new API types?**
  No

* **Will enabling / using this feature result in any new calls to cloud
  provider?**
  No

* **Will enabling / using this feature result in increasing size or count
  of the existing API objects?**
  No

* **Will enabling / using this feature result in increasing time taken by any
  operations covered by [existing SLIs/SLOs][]?**
  No

* **Will enabling / using this feature result in non-negligible increase of
  resource usage (CPU, RAM, disk, IO, ...) in any components?**
  No

### Troubleshooting
* **How does this feature react if the API server and/or etcd is unavailable?**
  N/A.

* **What are other known failure modes?**
  N/A.

* **What steps should be taken if SLOs are not being met to determine the problem?**
1. Errors for the preempt process are visible in logs.
2. check the metrics below to determine if there is an exception
  - pod_preemption_victims
  - total_preemption_attempts
  - scheduling_algorithm_preemption_evaluation_seconds

## Implementation History
- 2019-03-17: Initial KEP
- 2020-05-19: Graduate the feature to Beta
- 2022-01-15: Graduate the feature to GA
