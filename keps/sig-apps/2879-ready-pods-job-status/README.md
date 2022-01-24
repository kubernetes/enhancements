# KEP-2879: Track ready Pods in Job status

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
  - [Changes to the Job controller](#changes-to-the-job-controller)
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

Items marked with (R) are required *prior to targeting to a milestone / release*.

- [x] (R) Enhancement issue in release milestone, which links to KEP dir in [kubernetes/enhancements] (not the initial KEP PR)
- [x] (R) KEP approvers have approved the KEP status as `implementable`
- [x] (R) Design details are appropriately documented
- [ ] (R) Test plan is in place, giving consideration to SIG Architecture and SIG Testing input (including test refactors)
  - [ ] e2e Tests for all Beta API Operations (endpoints)
  - [ ] (R) Ensure GA e2e tests for meet requirements for [Conformance Tests](https://github.com/kubernetes/community/blob/master/contributors/devel/sig-architecture/conformance-tests.md) 
  - [ ] (R) Minimum Two Week Window for GA e2e tests to prove flake free
- [ ] (R) Graduation criteria is in place
  - [ ] (R) [all GA Endpoints](https://github.com/kubernetes/community/pull/1806) must be hit by [Conformance Tests](https://github.com/kubernetes/community/blob/master/contributors/devel/sig-architecture/conformance-tests.md) 
- [x] (R) Production readiness review completed
- [x] (R) Production readiness review approved
- [ ] "Implementation History" section is up-to-date for milestone
- [x] User-facing documentation has been created in [kubernetes/website], for publication to [kubernetes.io]
- [ ] Supporting documentation—e.g., additional design documents, links to mailing list discussions/SIG meetings, relevant PRs/issues, release notes

[kubernetes.io]: https://kubernetes.io/
[kubernetes/enhancements]: https://git.k8s.io/enhancements
[kubernetes/kubernetes]: https://git.k8s.io/kubernetes
[kubernetes/website]: https://git.k8s.io/website

## Summary

The Job status has a field `active` which counts the number of Job Pods that
are in `Running` or `Pending` phases. In this KEP, we add a field `ready` that
counts the number of Job Pods that have a `Ready` condition, with the same
best effort guarantees as the existing `active` field.

## Motivation

Job Pods can remain in the `Pending` phase for a long time in clusters with
tight resources and when image pulls take long. Since the `Job.status.active`
field includes `Pending` Pods, this can give a false impression of progress
to end users or other controllers. This is more important when the pods serve
as workers and need to communicate among themselves.

A separate `Job.status.ready` field can provide more information for users
and controllers, reducing the need to listen to Pod updates themselves.

Note that other workload APIs (such as ReplicaSet and StatefulSet) have a
similar field: `.status.readyReplicas`.

### Goals

- Add the field `Job.status.ready` that keeps a count of Job Pods with the
  `Ready` condition.

### Non-Goals

- Provide strong guarantees for the accuracy of the count. Due to the
  asynchronous nature of k8s, there are can be more or less Pods currently
  ready than what the count provides.

## Proposal

Add the field `.status.ready` to the Job API. The job controller updates the
field based on the number of Pods that have the `Ready` condition.

### Risks and Mitigations

- An increase in Job status updates. To mitigate this, the job controller holds
  the Pod updates that happen in X ms before syncing a Job.
  From experiments using integration tests, X=500ms was found to be a reasonable
  value.

## Design Details

### API

```golang
type JobStatus struct {
	...
	Active    int32
	Ready     *int32  // new field
	Succeeded int32
	Failed    int32
}
```

### Changes to the Job controller

The Job controller already lists the Pods to populate the `active`, `succeeded`
and `failed` fields. To count `ready` pods, the job controller will filter the
pods that have the `Ready` condition.

### Test Plan

- Unit and integration tests covering:
  - Count of ready pods.
  - Feature gate disablement.
- Verify passing existing E2E and conformance tests for Job.

### Graduation Criteria

#### Alpha

- Feature gate disabled by default.
- Unit and [integration] tests passing.

[integration]: https://testgrid.k8s.io/conformance-all#Conformance%20-%20GCE%20-%20master&include-filter-by-regex=sig-apps&include-filter-by-regex=Job&exclude-filter-by-regex=CronJob

#### Beta

- Feature gate enabled by default.
- Existing [E2E] and [conformance] tests passing.
- Scalability tests for Jobs of varying sizes, up to 500 parallelism, that keep
  track of metric `job_sync_duration_seconds`. There should be no significant
  degradation after enabling the feature gate.

[E2E]: https://testgrid.k8s.io/sig-apps#gce&include-filter-by-regex=apps%5C%5D%20Job
[Conformance]: https://testgrid.k8s.io/conformance-all#Conformance%20-%20GCE%20-%20master&include-filter-by-regex=sig-apps&include-filter-by-regex=Job&exclude-filter-by-regex=CronJob

#### GA

- Every bug report is fixed.
- The job controller ignores the feature gate.

#### Deprecation

N/A

### Upgrade / Downgrade Strategy

No changes required for existing cluster to use the enhancement.

### Version Skew Strategy

The feature doesn't affect nodes.

In the first release, a version skew between apiservers might cause the new
field to remain at zero even if there are Pods ready.

## Production Readiness Review Questionnaire

### Feature Enablement and Rollback

###### How can this feature be enabled / disabled in a live cluster?

- [x] Feature gate (also fill in values in `kep.yaml`)
  - Feature gate name: JobReadyPods
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

Yes, the Job controller might upgrade the Job status more frequently to
report ready pods.

###### Can the feature be disabled once it has been enabled (i.e. can we roll back the enablement)?

Yes, the lost of information is acceptable as the field is only informative.

###### What happens if we reenable the feature if it was previously rolled back?

The Job controller will start populating the field again.

###### Are there any tests for feature enablement/disablement?

Yes, there are tests at unit and [integration] level.

### Rollout, Upgrade and Rollback Planning

###### How can a rollout or rollback fail? Can it impact already running workloads?

The field is only informative, it doesn't affect running workloads.

###### What specific metrics should inform a rollback?

- An increase in `job_sync_duration_seconds`.
- A reduction in `job_sync_num`.

###### Were upgrade and rollback tested? Was the upgrade->downgrade->upgrade path tested?

A manual test will be performed, as follows:

1. Create a cluster in 1.23.
1. Upgrade to 1.24.
1. Create long running Job A, ensure that the ready field is populated.
1. Downgrade to 1.23.
1. Verify that ready field in Job A is not lost, but also not updated.
1. Create long running Job B, ensure that ready field is not populated.
1. Upgrade to 1.24.
1. Verify that Job A and B ready field is tracked again.

###### Is the rollout accompanied by any deprecations and/or removals of features, APIs, fields of API types, flags, etc.?

No

### Monitoring Requirements

###### How can an operator determine if the feature is in use by workloads?

The feature applies to all Jobs, unless the feature gate is disabled.

###### How can someone using this feature know that it is working for their instance?

- [x] API .status
  - Other field: `ready`

###### What are the reasonable SLOs (Service Level Objectives) for the enhancement?

The 99% percentile of Job status sync (processing+API calls) is below 2s, when
the controller doesn't create new Pods or tracks finishing Pods.

###### What are the SLIs (Service Level Indicators) an operator can use to determine the health of the service?

- [x] Metrics
  - Metric name: `job_sync_duration_seconds`, `job_sync_total`.
  - Components exposing the metric: `kube-controller-manager`

###### Are there any missing metrics that would be useful to have to improve observability of this feature?

No.

### Dependencies

###### Does this feature depend on any specific services running in the cluster?

No.

### Scalability

###### Will enabling / using this feature result in any new API calls?


- API: PUT Job/status

  Estimated throughput: at most one API call for each Job Pod reaching Ready
  condition.
  
  Originating component: job-controller

###### Will enabling / using this feature result in introducing new API types?

No.

###### Will enabling / using this feature result in any new calls to the cloud provider?

No.

###### Will enabling / using this feature result in increasing size or count of the existing API objects?

- API: Job/status

  Estimated increase in size: New field of less than 10B.

###### Will enabling / using this feature result in increasing time taken by any operations covered by existing SLIs/SLOs?

No.

###### Will enabling / using this feature result in non-negligible increase of resource usage (CPU, RAM, disk, IO, ...) in any components?

No.

### Troubleshooting

###### How does this feature react if the API server and/or etcd is unavailable?

No change from existing behavior of the Job controller.

###### What are other known failure modes?

- When the cluster has apiservers with skewed versions, the `Job.status.ready`
  might remain zero.

###### What steps should be taken if SLOs are not being met to determine the problem?

1. Check reachability between kube-controller-manager and apiserver.
1. If the `job_sync_duration_seconds` is too high, check for the number
   of requests in apiserver coming from the kube-system/job-controller service
   account. Consider increasing the number of inflight requests for
   apiserver or tuning [API priority and fairness](https://kubernetes.io/docs/concepts/cluster-administration/flow-control/)
   to give more priority for the job-controller requests.
1. If the steps above are insufficient disable the `JobTrackingWithFinalizers`
   feature gate from apiserver and kube-controller-manager and [report an issue](https://github.com/kubernetes/kubernetes/issues).

## Implementation History

- 2021-08-19: Proposed KEP starting in alpha status, including full PRR questionnaire.
- 2022-01-05: Proposed graduation to beta.

## Drawbacks

The only drawback is an increase in API calls. However, this is capped by
the number of times a Pod flips ready status. This is usually once for each
Pod created.

## Alternatives

- Add `Job.status.running`, counting Pods with `Running` phase. The `Running`
  phase doesn't take into account preparation work before the worker is ready
  to accept connections. On the other hand, the `Ready` condition is
  configurable through a readiness probe. If the Pod doesn't have a readiness
  probe configured, the `Ready` condition is equivalent to the `Running` phase.
  
  In other words, `Job.status.active` provides as the same behavior as
  `Job.status.running` with the advantage of it being configurable.
