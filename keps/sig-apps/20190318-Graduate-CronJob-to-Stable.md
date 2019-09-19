---
title: graduate-cronjob-to-stable
authors:
  - "@barney-s"
owning-sig: sig-apps
participating-sigs:
  - sig-scaling
reviewers:
  - "@liggitt"
  - "@kow3ns"
  - "@janetkuo"
  - “@mortent”
approvers:
  - "@kow3ns"
  - "@liggitt"
  - "@soltysh"
editor: TBD
creation-date: 2019-04-18
last-updated: 2019-04-18
status: provisional
see-also:
  - 
replaces:
superseded-by:
---

# Graduate CronJob to stable

## Table of Contents

- [Summary](#summary)
- [Motivation](#motivation)
  - [Goals](#goals)
  - [Non-Goals](#non-goals)
- [Proposal](#proposal)
  - [Implementation Notes](#implementation-notes)
  - [Constraints](#constraints)
  - [Test Plan](#test-plan)
  - [Graduation Criteria](#graduation-criteria)
- [Implementation History](#implementation-history)

## Summary

[CronJob](https://kubernetes.io/docs/concepts/workloads/controllers/cron-jobs/) is a Kubernetes API that creates Job object on a schedule specified by a cron spec. It is in beta status since v1.8. This document lays out the plan to promote it to stable.

## Motivation

CronJob definition has been stable for the last few releases and is useful to run periodic tasks in kubernetes cluster. This API adds the ability to add cron facility to a cluster. We feel the API is ready to be promoted to Stable and be supported longterm by the community.

### Goals

* Promote CronJob API to stable version - batch/v1/CronJob
* Deprecate older versions of CronJob API (batch/v1beta1, batch/v2alpha1) following the [deprecation policy](https://kubernetes.io/docs/reference/using-api/deprecation-policy/) 
* The CronJob controller would be alpha along with the current beta controller. It would then be promoted to beta and then to GA.

### Non-Goals

* Changing API field or meaning

## Proposal

### Implementation Notes

In the current implementation, the controller: 

1. syncs all CronJob objects [every 10 seconds](https://github.com/kubernetes/kubernetes/blob/30165e40ddfbe75fddc575c14294c6b540361078/pkg/controller/cronjob/controller.go#L98). 
2. Using pager library, gets all Pods and all CronJobs and [processes them one by one](https://github.com/kubernetes/kubernetes/blob/30165e40ddfbe75fddc575c14294c6b540361078/pkg/controller/cronjob/controller.go#L144)

The current implementation can be improved to reduce the need to list all Jobs and CronJobs frequently to reconcile. Instead we should replace it with an Informers and WorkQueue based architecture. Preferable we should be sharing the same informer cache as the Job controller uses.

This is required to:

1. Reduce the potential scale issues when using lots of pods and CronJob objects (not measured).  
2. Reduce load on API server in such cases.
3. Reduce memory usage when listing all Jobs and CronJobs in every sync loop.

We will implement the new controller to handle batch/v2alpha1 version initially. Then it will be promoted to handle batch/v1beta1 replacing the current implementation.

### Constraints

We need to verify the ability to share informer cache across controllers. 

### Test Plan

#### Existing Tests
- [Run or Not](https://github.com/kubernetes/kubernetes/blob/30165e40ddfbe75fddc575c14294c6b540361078/pkg/controller/cronjob/controller_test.go#L167) tests the controller under different scenarios to check if the Job is created or not
- [Validates Job cleanup](https://github.com/kubernetes/kubernetes/blob/30165e40ddfbe75fddc575c14294c6b540361078/pkg/controller/cronjob/controller_test.go#L371) path of the controller under different conditions.
- [Validates Status](https://github.com/kubernetes/kubernetes/blob/30165e40ddfbe75fddc575c14294c6b540361078/pkg/controller/cronjob/controller_test.go#L593) of the CronJob after sync under different conditions.

#### Needed Tests

- Conformance tests need to be added for CronJob

### Graduation Criteria

- [ ] Implement shared informers to reduce pressure on API Server
- [ ] Needs a conformance test
- [ ] Update documents to reflect the changes
- [ ] Pass current e2e tests for CronJob
- [ ] Pass existing and new unit-tests that are applicable

## Implementation History

- CronJob was introduced in Kubernetes 1.3 as ScheduledJobs
- In Kuberenetes 1.8 it was renamed to CronJob and promoted to Beta
