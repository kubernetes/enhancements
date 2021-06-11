# KEP-19: Graduate CronJob to stable

<!-- toc -->
- [Release Signoff Checklist](#release-signoff-checklist)
- [Summary](#summary)
- [Motivation](#motivation)
  - [Goals](#goals)
- [Proposal](#proposal)
  - [Plan for promoting CronJob to GA](#plan-for-promoting-cronjob-to-ga)
  - [Existing controller](#existing-controller)
  - [New controller](#new-controller)
    - [Informers and Caches](#informers-and-caches)
    - [Multiple workers](#multiple-workers)
    - [Handling Cron aspect](#handling-cron-aspect)
  - [Metrics](#metrics)
  - [Add .status.lastSuccessfulTime](#add-statuslastsuccessfultime)
  - [Fix applicable open issues](#fix-applicable-open-issues)
  - [Scale Targets for GA](#scale-targets-for-ga)
    - [CronJob Limits](#cronjob-limits)
    - [Frequency of launched jobs](#frequency-of-launched-jobs)
- [API changes](#api-changes)
  - [CronJob v1 API](#cronjob-v1-api)
  - [CronJob v1beta1 API](#cronjob-v1beta1-api)
  - [Validations](#validations)
- [Tests](#tests)
  - [E2E test](#e2e-test)
    - [Existing test cases](#existing-test-cases)
    - [New test cases](#new-test-cases)
  - [Conformance Tests](#conformance-tests)
  - [Unit Tests](#unit-tests)
- [Implementation plan](#implementation-plan)
    - [Release 1.20](#release-120)
    - [Release 1.21](#release-121)
    - [Release 1.22](#release-122)
- [Graduation Criteria](#graduation-criteria)
- [Production Readiness Review Questionnaire](#production-readiness-review-questionnaire)
  - [Feature Enablement and Rollback](#feature-enablement-and-rollback)
  - [Rollout, Upgrade and Rollback Planning](#rollout-upgrade-and-rollback-planning)
  - [Monitoring Requirements](#monitoring-requirements)
  - [Dependencies](#dependencies)
  - [Scalability](#scalability)
  - [Troubleshooting](#troubleshooting)
- [Implementation History](#implementation-history)
- [Alternatives and Further Reading](#alternatives-and-further-reading)
    - [Cron Aspect](#cron-aspect)
- [Improvements/Considerations](#improvementsconsiderations)
  - [Add .status.nextScheduleTime](#add-statusnextscheduletime)
  - [Add counters](#add-counters)
  - [Add .status.conditions](#add-statusconditions)
  - [Support Jitter for cronjobs](#support-jitter-for-cronjobs)
  - [Support Timezone for cronjobs](#support-timezone-for-cronjobs)
  - [CronJob umbrella issue](#cronjob-umbrella-issue)
<!-- /toc -->

## Release Signoff Checklist

Items marked with (R) are required *prior to targeting to a milestone / release*.

- [x] (R) Enhancement issue in release milestone, which links to KEP dir in [kubernetes/enhancements] (not the initial KEP PR)
- [x] (R) KEP approvers have approved the KEP status as `implementable`
- [x] (R) Design details are appropriately documented
- [x] (R) Test plan is in place, giving consideration to SIG Architecture and SIG Testing input
- [x] (R) Graduation criteria is in place
- [x] (R) Production readiness review completed
- [x] Production readiness review approved
- [x] "Implementation History" section is up-to-date for milestone
- [x] User-facing documentation has been created in [kubernetes/website], for publication to [kubernetes.io]
- [x] Supporting documentation—e.g., additional design documents, links to mailing list discussions/SIG meetings, relevant PRs/issues, release notes

[kubernetes.io]: https://kubernetes.io/
[kubernetes/enhancements]: https://git.k8s.io/enhancements
[kubernetes/kubernetes]: https://git.k8s.io/kubernetes
[kubernetes/website]: https://git.k8s.io/website

## Summary

[CronJob](https://kubernetes.io/docs/concepts/workloads/controllers/cron-jobs/) is a Kubernetes API that creates Job object on a schedule specified by a cron spec. It is in beta status since v1.8, and currently if it does not progress before v1.22, it will be marked deprecated with target release for removal being v1.25. This document lays out the plan to promote it to stable.

## Motivation

CronJob is useful to run periodic tasks using cron like facility in a kubernetes cluster. Its `.spec` has been stable for the last few releases. We feel the API with additional `.status` information is ready to be promoted to Stable and be supported long-term by the community.

### Goals

- Write a new controller which will
  - Use informers instead of polling
  - Address some open issues with the current controller
  - Add metrics exposing controller throughput, latency etc.
- Extend CronJob status field
  - lastSuccessfulTime: tracks the last time the job completed successfully

## Proposal

### Plan for promoting CronJob to GA

To promote to GA we would create `batch/v1/CronJob` in the [batch/v1 API](https://github.com/kubernetes/kubernetes/blob/b58777eda0f0077eba79d063c1c8f31cd7dbb8b9/staging/src/k8s.io/api/batch/v1/types.go).

We shall have dual implementation of controller (old and new) co-exist, until the new controller reaches GA.
The old and new implementation of the controller will be toggled through a feature flag:
- alpha (disabled by default, can be enabled)
- beta (enabled by default, can be disabled)
- GA (enabled by default, cannot be disabled)

After reaching GA we will remove the old controller.

### Existing controller

The current implementation of the CronJob controller is different than the other workload controllers. GA workload controllers use informers and caches to reduce the load on API server. Whereas the cronjob controller does a periodic poll and sweep of all the objects and acts on them. The CronJob controller has only one worker doing this.

1. syncs all CronJob objects [every 10 seconds](https://github.com/kubernetes/kubernetes/blob/b58777eda0f0077eba79d063c1c8f31cd7dbb8b9/pkg/controller/cronjob/cronjob_controller.go#L98).
2. Using pager library, gets all Pods and all CronJobs and [processes them one by one](https://github.com/kubernetes/kubernetes/blob/b58777eda0f0077eba79d063c1c8f31cd7dbb8b9/pkg/controller/cronjob/cronjob_controller.go#L136)

This is not a scalable design and ends up loading the API server. Also this does not follow the [recommended guidelines](https://github.com/kubernetes/community/blob/b58777eda0f0077eba79d063c1c8f31cd7dbb8b9/contributors/devel/sig-api-machinery/controllers.md) for building controllers.

### New controller

With this approach we aim to reduce the potential scale issues (e.g. load on API server and memory usage) when using lots of CronJob objects.

#### Informers and Caches

To reduce the need to list all Jobs and CronJobs frequently to reconcile, we propose to use Informers and WorkQueue based architecture. We would be sharing the [same informer cache](https://github.com/kubernetes/kubernetes/blob/b58777eda0f0077eba79d063c1c8f31cd7dbb8b9/cmd/kube-controller-manager/app/controllermanager.go#L450) as the Job controller uses.

We will follow a controller structure similar to existing workload controllers and as [outlined in the guideline](https://github.com/kubernetes/community/blob/b58777eda0f0077eba79d063c1c8f31cd7dbb8b9/contributors/devel/sig-api-machinery/controllers.md#rough-structure).

```golang
// CronJobController is 2nd generation controller for cronjobs.
// It is using informers and workqueue to improve its performance over
// the old controller.
type CronJobController struct {
	queue	   			workqueue.DelayingInterface
	recorder   			record.EventRecorder

	jobControl 			jobControlInterface
	cronJobControl  		cronJobControlInterface

	jobLister  			batchv1listers.JobLister
	cronJobLister 			CronJobLister

	jobListerSynced 		cache.InformerSynced
	cronJobListerSynced		cache.InformerSynced
}

// CronJobControllerConfiguration contains controller config
type CronJobControllerConfiguration struct {
	// concurrentCronJobSyncs is the number of cronjob objects that are
	// allowed to sync concurrently. Larger number = more responsive
	// but more CPU (and network) load.
	ConcurrentCronJobSyncs int32
}
```

#### Multiple workers

We also propose to have multiple workers controller by a flag similar to [statefulset controller](https://github.com/kubernetes/kubernetes/blob/b58777eda0f0077eba79d063c1c8f31cd7dbb8b9/cmd/kube-controller-manager/app/apps.go#L65). The default would be set to 5 similar to [statefulset](https://github.com/kubernetes/kubernetes/blob/b58777eda0f0077eba79d063c1c8f31cd7dbb8b9/pkg/controller/statefulset/config/v1alpha1/defaults.go#L34)

#### Handling Cron aspect

To detect which CronJob has met its schedule and need to create Jobs we need to implement a timer component. We shall implement a Heap based timer algorithm. We will introduce a separate queue with the [`DelayingInterface`](https://github.com/kubernetes/client-go/blob/b58777eda0f0077eba79d063c1c8f31cd7dbb8b9/util/workqueue/delaying_queue.go#L37) that implements heap based single shot api [`AddAfter`](https://github.com/kubernetes/client-go/blob/b58777eda0f0077eba79d063c1c8f31cd7dbb8b9/util/workqueue/delaying_queue.go#L150). Every time we process an entry from this queue, we will add it back to the queue to simulate a periodic timer.

For further reading:
1. [Reinventing timer wheel](https://lwn.net/Articles/646950/)
2. [Hashed and hierarchical timer wheel](http://www.cs.columbia.edu/~nahum/w6998/papers/sosp87-timing-wheels.pdf)
3. [Golang timers in multi-cpu systems](https://github.com/golang/go/commit/76f4fd8a5251b4f63ea14a3c1e2fe2e78eb74f81)
4. [Go timerwheel](https://github.com/RussellLuo/timingwheel)

### Metrics

We propose to add metrics that could expose the performance and health of the controller including and not limited to:
- Skew (actualJobCreationTime-expectedJobCreationTime) - [Histogram](https://prometheus.io/docs/concepts/metric_types/#histogram)

Queue depth, latency and throughput can be surfaced from existing controller framework.

### Add .status.lastSuccessfulTime

[#issue/75674](https://github.com/kubernetes/kubernetes/issues/75674)
Add `lastSuccessfulTime` to `.status` that tracks the last time the job completed successfully. This will augment the `lastScheduledTime` available in the `.status` in the v1beta1 api. Potential use is in monitoring (e.g. fire an alert if lastSuccessfulTime is more than X ago).

### Fix applicable open issues

These are the [current](https://github.com/kubernetes/kubernetes/issues/82659) list of issues that are being targeted for GA. We will be targeting to address as many as possible, starting with the following:

- [Updating a cronjob causes jobs to be scheduled retroactively](https://github.com/kubernetes/kubernetes/issues/63371)
- [CLI: Updated CronJob Schedule Missing from Dry Run](https://github.com/kubernetes/kubernetes/issues/73613)
- [Kubernetes CronJob pods is not getting clean-up when Job is completed](https://github.com/kubernetes/kubernetes/issues/74741)
- [Infinite ImagePullBackOff CronJob results in resource leak](https://github.com/kubernetes/kubernetes/issues/76570)
- [Cronjob `spec.schedule` cannot be change when `spec.schedule` value not `"` or `’`](https://github.com/kubernetes/kubernetes/issues/78646)
- [Kubelet CPU/Memory Usage linearly increases using CronJob](https://github.com/kubernetes/kubernetes/issues/64137)
- [Stopping cluster overnight prevents scheduled jobs from running after cluster startup](https://github.com/kubernetes/kubernetes/issues/42649)
- [Fix CronJob missed start time handling](https://github.com/kubernetes/kubernetes/pull/81557)

### Scale Targets for GA

The scale targets for CronJob GA API shall conform to existing [SLIs/SLOs of Kubernetes native types](https://github.com/kubernetes/community/blob/b58777eda0f0077eba79d063c1c8f31cd7dbb8b9/sig-scalability/slos/slos.md#kubernetes-slisslos).

The targets are defined by the below suggested maximum limits, which are organized the same way as the [Kubernetes native type thresholds](https://github.com/kubernetes/community/blob/b58777eda0f0077eba79d063c1c8f31cd7dbb8b9/sig-scalability/configs-and-limits/thresholds.md#kubernetes-thresholds).

#### CronJob Limits

There should be nothing in the implementation that limits CronJobs per namespace. Overall cluster-wide limits of CronJob are important. Cluster wide limits for CronJob should be storage bound since it shares the storage space with all other objects. Determining the appropriate storage limit for a cluster is out-of-scope for this document.

#### Frequency of launched jobs

The number of CronJobs is also sensitive to the API server QPS and the schedule of the individual CronJobs. This translates to the frequency of launched jobs. We could have large number of CronJobs with a spread of schedule that doesn't stress the Job API. At the same time we could have a small number of CronJobs that schedule synchronously stressing the Jobs API. The design must be able to easily saturate the API server QPS. The user can setup rate limits for CronJob and Job APIs using [API Server rate limting config](https://kubernetes.io/docs/reference/access-authn-authz/admission-controllers/#eventratelimit).

## API changes

These are the new fields added as part of promotion to stable:
- `.status`
  - `.lastSuccessfulTime`

### CronJob v1 API

```golang
// CronJob represents the configuration of a single cron job.
type CronJob struct {
	metav1.TypeMeta
	// Standard object's metadata.
	// More info: https://git.k8s.io/community/contributors/devel/sig-architecture/api-conventions.md#metadata
	// +optional
	metav1.ObjectMeta

	// Specification of the desired behavior of a cron job, including the schedule.
	// More info: https://git.k8s.io/community/contributors/devel/sig-architecture/api-conventions.md#spec-and-status
	// +optional
	Spec CronJobSpec

	// Current status of a cron job.
	// More info: https://git.k8s.io/community/contributors/devel/sig-architecture/api-conventions.md#spec-and-status
	// +optional
	Status CronJobStatus
}


// CronJobList is a collection of cron jobs.
type CronJobList struct {
	metav1.TypeMeta
	// Standard list metadata.
	// More info: https://git.k8s.io/community/contributors/devel/sig-architecture/api-conventions.md#metadata
	// +optional
	metav1.ListMeta

	// Items is the list of CronJobs.
	Items []CronJob
}

// CronJobSpec describes how the job execution will look like and when it will actually run.
type CronJobSpec struct {
	// The schedule in Cron format, see https://en.wikipedia.org/wiki/Cron.
	Schedule string

	// Optional deadline in seconds for starting the job if it misses scheduled
	// time for any reason.  Missed jobs executions will be counted as failed ones.
	// +optional
	StartingDeadlineSeconds *int64

	// Specifies how to treat concurrent executions of a Job.
	// Valid values are:
	// - "Allow" (default): allows CronJobs to run concurrently;
	// - "Forbid": forbids concurrent runs, skipping next run if previous run hasn't finished yet;
	// - "Replace": cancels currently running job and replaces it with a new one
	// +optional
	ConcurrencyPolicy ConcurrencyPolicy

	// This flag tells the controller to suspend subsequent executions, it does
	// not apply to already started executions.  Defaults to false.
	// +optional
	Suspend *bool

	// Specifies the job that will be created when executing a CronJob.
	JobTemplate JobTemplateSpec

	// The number of successful finished jobs to retain.
	// This is a pointer to distinguish between explicit zero and not specified.
	// +optional
	SuccessfulJobsHistoryLimit *int32

	// The number of failed finished jobs to retain.
	// This is a pointer to distinguish between explicit zero and not specified.
	// +optional
	FailedJobsHistoryLimit *int32
}

// ConcurrencyPolicy describes how the job will be handled.
// Only one of the following concurrent policies may be specified.
// If none of the following policies is specified, the default one
// is AllowConcurrent.
type ConcurrencyPolicy string

const (
	// AllowConcurrent allows CronJobs to run concurrently.
	AllowConcurrent ConcurrencyPolicy = "Allow"

	// ForbidConcurrent forbids concurrent runs, skipping next run if previous
	// hasn't finished yet.
	ForbidConcurrent ConcurrencyPolicy = "Forbid"

	// ReplaceConcurrent cancels currently running job and replaces it with a new one.
	ReplaceConcurrent ConcurrencyPolicy = "Replace"
)

// CronJobStatus represents the current state of a cron job.
type CronJobStatus struct {
	// A list of pointers to currently running jobs.
	// +optional
	Active []api.ObjectReference

	// Information when was the last time the job was successfully scheduled.
	// +optional
	LastScheduleTime *metav1.Time

	// Information when was the last time the job successfully completed.
	// +optional
	LastSuccessfulTime *metav1.Time
}
```

### CronJob v1beta1 API

All the new fields described in the v1 section would be introduced in the v1beta1 API as well.

### Validations

Nothing additional from v1beta1

## Tests

### E2E test

CronJob E2E test code is [located here](https://github.com/kubernetes/kubernetes/blob/b58777eda0f0077eba79d063c1c8f31cd7dbb8b9/test/e2e/apps/cronjob.go). The new controller should pass the current set of E2E.

#### Existing test cases

- ConcurrencyPolicy
  - should schedule multiple jobs concurrently
  - should not schedule new jobs when ForbidConcurrent
  - should replace jobs when ReplaceConcurrent
- Suspend
  - should not schedule jobs when suspended
- SuccessfulJobsHistoryLimit
  - should delete successful finished jobs when above successfulJobsHistoryLimit
- FailedJobsHistoryLimit
  - should delete failed finished jobs when above failedJobsHistoryLimit
- Events Recorder
  - should not emit unexpected warnings
  - should remove from active list jobs that have been deleted

#### New test cases

- Schedule
  - Should not create a cronjob with invalid schedule format
- StartingDeadlineSeconds
  - Should not schedule a job within two minutes when missed the current window if StartingDeadlineSeconds is 0
  - Should schedule a job soon when missed the current window if StartingDeadlineSeconds is long
- JobTemplate
  - Should not schedule a job with invalid job template
- [Endpoints coverage](https://apisnoop.cncf.io/?zoomed=category-beta-batch)
  - Should list cronjobs for all namespaces
  - Should update a cronjob
  - Should patch a cronjob
  - Should delete all cronjobs in a namespace
  - Should get a cronjob status
  - Should update a cronjob status
  - Should patch a cronjob status
- Tests covering Bug fixes
  - [issue/63371 - Should not start “missed” jobs from old cronjob after updating time](https://github.com/kubernetes/kubernetes/issues/63371)
  - [issue/74741 - Should cleanup finished pods when job is completed](https://github.com/kubernetes/kubernetes/issues/74741)
  - [issue/76570 - Should not keep creating new pods when job image has ImagePullBackOff error](https://github.com/kubernetes/kubernetes/issues/76570)
  - [issue/78646 - Should change schedule when schedule value is not wrapped with quotes](https://github.com/kubernetes/kubernetes/issues/78646)
- Scaling
  - Should be able to create and schedule at least 5000 cronjobs
  - Measure scheduling skew, podCreationLatency and check if it meets expectation (To Be Defined)
- Start Stop Tests
  - Schedule cronjobs and randomly stop the controller and start it.
  - Schedule cronjobs and stop the controller and start it after the deadline.

### Conformance Tests

The conformance tests are a subset of e2e tests. We will select test scenarios that we believe are expected from all conforming clusters. Then modify the test case to use the `framework.ConformanceIt()` function rather than the `framework.It()` function.
These e2e tests shall be included in conformance tests:

- ConcurrencyPolicy
  - should schedule multiple jobs concurrently when AllowConcurrent
  - should not schedule new jobs when ForbidConcurrent
  - should replace jobs when ReplaceConcurrent
- Suspend
  - should not schedule jobs when suspended
- SuccessfulJobsHistoryLimit
  - should delete successful finished jobs when above successfulJobsHistoryLimit
- FailedJobsHistoryLimit
  - should delete failed finished jobs when above failedJobsHistoryLimit
- StartingDeadlineSeconds
  - Should schedule a job soon when missed the current window if StartingDeadlineSeconds is long
- Tests covering Bug fixes
  - [issue/63371 - Should not start “missed” jobs from old cronjob after updating time](https://github.com/kubernetes/kubernetes/issues/63371)
- Start Stop Tests
  - Schedule cronjobs and randomly stop the controller and start it.
  - Schedule cronjobs and stop the controller and start it after the deadline.

### Unit Tests

This is subject to the new re-architected controller implementation. Overall these scenarios would be tested.

- [Run or Not](https://github.com/kubernetes/kubernetes/blob/b58777eda0f0077eba79d063c1c8f31cd7dbb8b9/pkg/controller/cronjob/cronjob_controller_test.go#L167) tests the controller under different scenarios to check if the Job is created or not
- [Validates Job cleanup](https://github.com/kubernetes/kubernetes/blob/b58777eda0f0077eba79d063c1c8f31cd7dbb8b9/pkg/controller/cronjob/cronjob_controller_test.go#L371) path of the controller under different conditions.
- [Validates Status](https://github.com/kubernetes/kubernetes/blob/b58777eda0f0077eba79d063c1c8f31cd7dbb8b9/pkg/controller/cronjob/cronjob_controller_test.go#L593) of the CronJob after sync under different conditions.

## Implementation plan

#### Release 1.20

- Alpha: Feature flag for new controller is disabled by default
- Dual controller. Both old and new implementation co-exist

#### Release 1.21

- Beta: Feature flag for new controller is enabled by default. If the distribution chooses it can be disabled
- Introduce CronJob in batch/v1

#### Release 1.22

- GA: The feature flag is deprecated and the old controller code is cleaned up

## Graduation Criteria

- [ ] Implement shared informers to reduce pressure on API Server
- [ ] Pass conformance tests
- [ ] Update documents reflecting the changes
- [ ] Pass CronJob e2e tests
- [ ] Pass CronJob unit-tests
- [ ] Pass scale tests

## Production Readiness Review Questionnaire

### Feature Enablement and Rollback

* **How can this feature be enabled / disabled in a live cluster?**
  - [x] Feature gate (also fill in values in `kep.yaml`)
    - Feature gate name: `CronJobControllerV2`
    - Components depending on the feature gate: `kube-controller-manager`

* **Does enabling the feature change any default behavior?**
  No.

* **Can the feature be disabled once it has been enabled (i.e. can we roll back
  the enablement)?**
  Yes.

* **What happens if we reenable the feature if it was previously rolled back?**
  No changes are expected.

* **Are there any tests for feature enablement/disablement?**
  Not applicable, given we're missing framework allowing switching feature-gates during e2e.

### Rollout, Upgrade and Rollback Planning

* **How can a rollout fail? Can it impact already running workloads?**
  This might fail when leader election for kube-controller-manager ends up assigned to two separate instances. In this case the two controller might end up fighting over the cronjob resource.  Additionally, a bug in the new controller might trigger a panic and thus crashing the whole kube-controller-manager.  Another problem might come from GA-ing batch/v1.CronJob which might cause problems with conversions between different versions of that resource.

* **What specific metrics should inform a rollback?**
  Unexpected restarts of kube-controller-manager.

* **Were upgrade and rollback tested? Was the upgrade->downgrade->upgrade path tested?**
  Manual test is planned once the implementation is finished.

* **Is the rollout accompanied by any deprecations and/or removals of features, APIs,
fields of API types, flags, etc.?**
  batch/v1beta1.CronJob is already deprecated.

### Monitoring Requirements

* **How can an operator determine if the feature is in use by workloads?**
  For the new controller part checking `cronjob_job_creation_skew` and `cronjob_controller_rate_limiter_use` metrics, accompanied with a set of queue-related metrics the old controller is not using.  For the CronJob resource GA verifying if CronJob exists in `batch/v1` API group in the cluster.

* **What are the SLIs (Service Level Indicators) an operator can use to determine
the health of the service?**
  - [x] Metrics
    - Metric name: `cronjob_controller_rate_limiter_use`
    - Metric name: `cronjob_job_creation_skew`
    - Metric name: `workqueue_depth`
    - Metric name: `workqueue_retries`
    - Metric name: `workqueue_adds_total`
    - Components exposing the metric: `kube-controller-manager`
    - Metric name: `etcd_object_counts{resource="jobs.batch"}`
    - Components exposing the metric: `kube-apiserver`.

* **What are the reasonable SLOs (Service Level Objectives) for the above SLIs?**
  99th percentile of cron_job_creation_skew <= X per cluster-day.

* **Are there any missing metrics that would be useful to have to improve observability
of this feature?**
  No.

### Dependencies

* **Does this feature depend on any specific services running in the cluster?**
  No.

### Scalability

* **Will enabling / using this feature result in any new API calls?**
  - WATCH cronjobs, which will replace the current cronjobs polling
  - estimated throughput TBD
  - originating component(s) kube-controller-manager

* **Will enabling / using this feature result in introducing new API types?**
  Graduating CronJob API to GA, iow. CronJob in `batch/v1`.

* **Will enabling / using this feature result in any new calls to the cloud
provider?**
  No.

* **Will enabling / using this feature result in increasing size or count of
the existing API objects?**
  No.

* **Will enabling / using this feature result in increasing time taken by any
operations covered by [existing SLIs/SLOs]?**
  No.

* **Will enabling / using this feature result in non-negligible increase of
resource usage (CPU, RAM, disk, IO, ...) in any components?**
  No.

### Troubleshooting

The Troubleshooting section currently serves the `Playbook` role. We may consider
splitting it into a dedicated `Playbook` document (potentially with some monitoring
details). For now, we leave it here.

* **How does this feature react if the API server and/or etcd is unavailable?**
  The controller will not be able to work.

* **What are other known failure modes?**
  None.

* **What steps should be taken if SLOs are not being met to determine the problem?**
  TBD

[supported limits]: https://git.k8s.io/community/sig-scalability/configs-and-limits/thresholds.md
[existing SLIs/SLOs]: https://git.k8s.io/community/sig-scalability/slos/slos.md#kubernetes-slisslos

## Implementation History

- CronJob was introduced in Kubernetes 1.3 as ScheduledJobs
- In Kubernetes 1.8 it was renamed to CronJob and promoted to Beta
- In Kubernetes 1.19 CronJob was marked deprecated with removal in 1.22 due to not being promoted to GA.

## Alternatives and Further Reading

#### Cron Aspect

To detect which CronJob has met its schedule and need to create Jobs we need to implement a timer component. These are the possible options for implementing the timer:

| algorithm  | how it works | notes |
|:----------|:----------|:-------|
| Unordered timer list | Periodic Sweep from cache | Slower and similar to existing implementation. But improved because we sweep fom the cache instead of API server |
|Ordered timer list| Maintain ordered list of Cronjob keys and next time of expiry. Keep starting a timer with the earliest expiry. | Efficient. Reinsertion to list takes O(n) |
|Timer trees| Instead of ordered list use a sorted binary tree. | More efficient. Insertion is O(log n) |
|Heap based timer|A variant of ordered timer list where heap is used to store the next expiry time | Efficient compared to ordered list. Bookkeeping and insertion is O(log n). |
|Simple Timing wheels| circular buffer of MaxTimeOut slots. List of expiring timers at each slot. | Works for small bounded  MaxTimeOut which is not our case. Insertion and removal is O(1) via indexing |
|Hashed Wheel| Hash expiring time and insert in a hash table with linked list at each index | Bookkeeping is O(1) and worst case insertion is O(n) |
|Hierarchical Wheel| multiple timer wheels for different resolutions (Seconds, minutes, hours, days). When seconds rolls over we grab the next minutes timers and recreate the seconds wheel. similarly for minutes and hours. | Sharding at different hierarchy levels improves insertion and bookkeeping performance. |

## Improvements/Considerations

Below list contains possible future extensions fo both the CronJob resource and its 2nd generation controller. Their completion is not mandatory to promote CronJob to stable.

### Add .status.nextScheduleTime

[#issue/78564](https://github.com/kubernetes/kubernetes/issues/78564)
Add `nextScheduleTime` to `.status` that tracks the next time the job will be scheduled. This may not be accurate with `Forbid` concurrency policy. This only tracks the `Job` creation time and not the actual `Pod` creation time.

| Concurrency policy | notes |
|:----------|:----------|
| Allow   | `nextScheduleTime` would be accurate within a margin of controller scheduling jitter |
| Forbid  | `nextScheduleTime` would not be accurate if the previous `Job` takes longer than the cron interval time |
| Replace | `nextScheduleTime` would be accurate within a margin of controller scheduling jitter along with older concurrent `Job` cleanup if applicable. |

### Add counters

These counters would be added to `.status` section of the CronJob object:
- `SuccessfulRuns` Count of all successful runs
- `FailedRuns` Count of all failed runs
- `FailuresSinceSuccess` Count of failed runs since last successful

### Add .status.conditions

Add a condition array with `Settled` condition type. This would help with the effort of standardizing conditions across all core types. `Settled` is set at the end of every successful reconcile run. The key thing to note here is the notions of `Settled` does not imply the `Job`s are running correctly. It just means that the controller is done processing this object successfully.

NOTE: This should be handled across all of SIG-Apps owned controller.

### Support Jitter for cronjobs

We propose to introduce `.spec.jitter` which is a percentage of the time delta to the next schedule. We propose to cap it to 50%. There is also a [community request](https://github.com/kubernetes/community/issues/2440) for this.

```golang
delta = nextScheduleTime - currentTime
jitter = delta*cronjob.Spec.Jitter/100
nextScheduleTime += jitter
```

### Support Timezone for cronjobs

We propose to introduce `.spec.timezone` which indicates the timezone to be considered when scheduling this cronjob.

Clusters across different environments have different timezones. There is no known recommendation or conformance test to define timezone for the control plane. Different distributions may have different behavior. For CronJob, the timezone is inferred from the control plane (specifically, the timezone as seen by the kube-controller-manager process). Someone with just API access has no idea what that controller manager timezone is. Sometimes cluster operators too, who run the controller manager in a container with a UTC timezone on a host with a non-UTC timezone.

These conditions imply that people with just API access who want to predict when their job will run have no guarantee. By providing a timezone field we provide an absolute value instead of relying on the relative timezone of where the master is running. This ensures more portability of the CronJob configs across clusters as well.

This is an optional field and when not present reverts to existing behavior of using the timezone of the `kube-controller-manager` process.

There is also a [community request](https://github.com/kubernetes/kubernetes/issues/47202) for this. Including a working (external) [implementation, called CronJobber](https://github.com/hiddeco/cronjobber) based off the original controller implementation, also mentioned in the linked issue.

### CronJob umbrella issue

Majority of the problems mentioned earlier are listed in this [umbrella issue](https://github.com/kubernetes/kubernetes/issues/82659). When adding additional capabilities to CronJobs it is advised to review this list and address all the problems identified there.
