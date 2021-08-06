# Promote Pod Priority and Preemption to GA

## Table of Contents

<!-- toc -->
- [Summary](#summary)
- [Motivation](#motivation)
  - [Goals](#goals)
  - [Non-Goals](#non-goals)
- [Proposal](#proposal)
  - [Risks and Mitigations](#risks-and-mitigations)
- [Graduation Criteria](#graduation-criteria)
- [Production Readiness Review Questionnaire](#production-readiness-review-questionnaire)
  - [Feature enablement and rollback](#feature-enablement-and-rollback)
  - [Rollout, Upgrade and Rollback Planning](#rollout-upgrade-and-rollback-planning)
  - [Monitoring requirements](#monitoring-requirements)
  - [Dependencies](#dependencies)
  - [Scalability](#scalability)
  - [Troubleshooting](#troubleshooting)
- [Testing Plan](#testing-plan)
  - [Unit Tests](#unit-tests)
  - [Integration tests](#integration-tests)
  - [E2E tests](#e2e-tests)
- [Implementation History](#implementation-history)
<!-- /toc -->

## Summary

Pod Priority and Preemption are features introduced in Kubernetes 1.8 as alpha features and 
promoted to beta in 1.11. Pod Priority enables users to specify importance of
a Pod. Pods with higher priority are scheduled ahead of other pods with
lower priority. When a cluster does not have enough capacity for running a high
priority pod, the scheduler preempts and removes lower priority pods in order to
make room for the high priority pod.

## Motivation

Pod Priority and Preemption have existed in the past several releases and some
of our most critical components of Kubernetes, i.e. critical DaemonSet Pods,
rely on this feature for guaranteed scheduling since Kubernetes 1.12.

### Goals

Promote Pod Priority and Preemption to GA.

### Non-Goals

Make any change of functionality to the features.

## Proposal

Create `scheduling.k8s.io/v1` API group and add `PriorityClass` to it.

Make necessary changes to our code base to use `scheduling.k8s.io/v1` instead of
`scheduling.k8s.io/v1beta1`.

Update our documentation to reflect the new version and status of the features.

### Risks and Mitigations

Given that there is no functionality changes, we don't expect any logical errors
caused by this change.

## Graduation Criteria

* The features have been stable and reliable in the past several releases.
* Adequate documentation exists for the features.
* Test coverage of the features is acceptable.

## Production Readiness Review Questionnaire
We'd like priority classes to be mutable going from release 1.23 because of we're technically achieving the
same effect with re-creation. Reasons for making priority classes immutable are mentioned in this 
[proposal](https://github.com/kubernetes/community/blob/master/contributors/design-proposals/scheduling/pod-priority-api.md#drawbacks-of-changing-priority-classes)
but we noticed that people are anyways deleting/creating priority classes despite the reasons mentioned above.

### Feature enablement and rollback

* **How can this feature be enabled / disabled in a live cluster?**

  We'll introduce a new featuregate flag called `AllowPriorityClassUpdates`. By enabling this flag, this feature can be enabled.

* **Does enabling the feature change any default behavior?**

  It changes the behavior of priority classes. Priority class values and names can be changed now.

* **Can the feature be disabled once it has been enabled (i.e. can we rollback
  the enablement)?**

  Yes, by disabling the featuregate

* **What happens if we reenable the feature if it was previously rolled back?**

  Priority classes become immutable again.

* **Are there any tests for feature enablement/disablement?**

  Yes, we will add unit tests at api validation package level.

### Rollout, Upgrade and Rollback Planning

* **How can a rollout fail? Can it impact already running workloads?**

  If the rollout fails, it won't impact the already running workloads.

* **What specific metrics should inform a rollback?**

  N/A.

* **Were upgrade and rollback tested? Was upgrade->downgrade->upgrade path tested?**

  Haven't been tested. Will be tested once this feature graduates.

* **Is the rollout accompanied by any deprecations and/or removals of features,
  APIs, fields of API types, flags, etc.?**
  
  N/A.

### Monitoring requirements

* **How can an operator determine if the feature is in use by workloads?**

  N/A.

* **How can someone using this feature know that it is working for their instance?**

  By updating priority classes. If they're able to do so, it means the feature is working, if not the 
  feature is not working

* **What are the SLIs (Service Level Indicators) an operator can use to
  determine the health of the service?**
  
  N/A.

* **What are the reasonable SLOs (Service Level Objectives) for the above SLIs?**

  N/A.

* **Are there any missing metrics that would be useful to have to improve
  observability if this feature?**

  N/A.

### Dependencies

* **Does this feature depend on any specific services running in the cluster?**

  No.


### Scalability

* **Will enabling / using this feature result in any new API calls?**

  No

* **Will enabling / using this feature result in introducing new API types?**

  No REST API changes.

* **Will enabling / using this feature result in any new calls to cloud
  provider?**
  
  No.

* **Will enabling / using this feature result in increasing size or count
  of the existing API objects?**
  
  No.

* **Will enabling / using this feature result in increasing time taken by any
  operations covered by [existing SLIs/SLOs][]?**
  
  No.

* **Will enabling / using this feature result in non-negligible increase of
  resource usage (CPU, RAM, disk, IO, ...) in any components?**
  
  No.

### Troubleshooting

* **How does this feature react if the API server and/or etcd is unavailable?**

  N/A.

* **What are other known failure modes?**

  Errors will be logged to stderr. 

* **What steps should be taken if SLOs are not being met to determine the problem?**

  N/A.

## Testing Plan
Pod priority and preemption have unit, integration, and e2e tests. These tests
are run regularly as a part of Kubernetes presubmit and CI/CD pipeline.

### Unit Tests
Here is a list of unit tests for various modules of the feature:
* [Priority admission controller tests](https://github.com/kubernetes/kubernetes/blob/master/plugin/pkg/admission/priority/admission_test.go)
* [Priority aware scheduling queue tests](https://github.com/kubernetes/kubernetes/blob/master/pkg/scheduler/internal/queue/scheduling_queue_test.go)
* [Scheduler preemption tests](https://github.com/kubernetes/kubernetes/blob/master/pkg/scheduler/core/generic_scheduler_test.go).
This file includes other tests too.

### Integration tests
Integration tests for priority and preemption are [found here](https://github.com/kubernetes/kubernetes/blob/master/test/integration/scheduler/preemption_test.go).

### E2E tests
End to end tests for priority and preemption are [found here](https://github.com/kubernetes/kubernetes/blob/master/test/e2e/scheduling/preemption.go).

## Implementation History

Pod Priority and Preemption are tracked as part of [enhancement#564](https://github.com/kubernetes/enhancements/issues/564).
The proposal for Pod Priority can be [found here](https://github.com/kubernetes/community/blob/master/contributors/design-proposals/scheduling/pod-priority-api.md)
and Preemption proposal is [here](https://github.com/kubernetes/community/blob/master/contributors/design-proposals/scheduling/pod-preemption.md).
