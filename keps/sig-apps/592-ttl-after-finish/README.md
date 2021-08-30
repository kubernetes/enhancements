# TTL After Finished Controller

## Table of Contents

<!-- toc -->
- [Summary](#summary)
- [Motivation](#motivation)
  - [Goals](#goals)
- [Proposal](#proposal)
  - [Concrete Use Cases](#concrete-use-cases)
  - [Detailed Design](#detailed-design)
    - [Feature Gate](#feature-gate)
    - [API Object](#api-object)
      - [Validation](#validation)
  - [User Stories](#user-stories)
  - [Implementation Details/Notes/Constraints](#implementation-detailsnotesconstraints)
    - [TTL Controller](#ttl-controller)
    - [Finished Jobs](#finished-jobs)
    - [Owner References](#owner-references)
  - [Risks and Mitigations](#risks-and-mitigations)
  - [Test Plan](#test-plan)
- [Graduation Criteria](#graduation-criteria)
  - [Alpha](#alpha)
  - [Alpha -&gt; Beta](#alpha---beta)
  - [Beta -&gt; GA](#beta---ga)
- [Production Readiness Review Questionnaire](#production-readiness-review-questionnaire)
  - [Feature Enablement and Rollback](#feature-enablement-and-rollback)
  - [Rollout, Upgrade and Rollback Planning](#rollout-upgrade-and-rollback-planning)
  - [Monitoring Requirements](#monitoring-requirements)
  - [Dependencies](#dependencies)
  - [Scalability](#scalability)
  - [Troubleshooting](#troubleshooting)
- [Future Work](#future-work)
- [Implementation History](#implementation-history)
<!-- /toc -->

## Summary

We propose a TTL mechanism to limit the lifetime of finished resource objects,
including Jobs and Pods, to make it easy for users to clean up old Jobs/Pods
after they finish. The TTL timer starts when the Job/Pod finishes, and the
finished Job/Pod will be cleaned up after the TTL expires.

## Motivation

In Kubernetes, finishable resources, such as Jobs and Pods, are often
frequently-created and short-lived. If a Job or Pod isn't controlled by a
higher-level resource (e.g. CronJob for Jobs or Job for Pods), or owned by some
other resources, it's difficult for the users to clean them up automatically,
and those Jobs and Pods can accumulate and overload a Kubernetes cluster very
easily. Even if we can avoid the overload issue by implementing a cluster-wide
(global) resource quota, users won't be able to create new resources without
cleaning up old ones first. See [#64470][].

The design of this proposal can be later generalized to other finishable
frequently-created, short-lived resources, such as completed Pods or finished
custom resources.

[#64470]: https://github.com/kubernetes/kubernetes/issues/64470

### Goals

Make it easy to for the users to specify a time-based clean up mechanism for
finished resource objects. 
* It's configurable at resource creation time and after the resource is created.

## Proposal

[K8s Proposal: TTL controller for finished Jobs and Pods][]

[K8s Proposal: TTL controller for finished Jobs and Pods]: https://docs.google.com/document/d/1U6h1DrRJNuQlL2_FYY_FdkQhgtTRn1kEylEOHRoESTc/edit

### Concrete Use Cases

* [Kubeflow][] needs to clean up old finished Jobs (K8s Jobs, TF Jobs, Argo
  workflows, etc.), see [#718][].

* [Prow][] needs to clean up old completed Pods & finished Jobs. Currently implemented with Prow sinker.

* [Apache Spark on Kubernetes][] needs proper cleanup of terminated Spark executor Pods.

* Jenkins Kubernetes plugin creates slave pods that execute builds. It needs a better way to clean up old completed Pods.

[Kubeflow]: https://github.com/kubeflow
[#718]: https://github.com/kubeflow/tf-operator/issues/718
[Prow]: https://github.com/kubernetes/test-infra/tree/master/prow
[Apache Spark on Kubernetes]: http://spark.apache.org/docs/latest/running-on-kubernetes.html

### Detailed Design 

#### Feature Gate

This will be launched as an alpha feature first, with feature gate
`TTLAfterFinished`.

#### API Object

We will add the following API fields to `JobSpec` (`Job`'s `.spec`).

```go
type JobSpec struct {
 	// ttlSecondsAfterFinished limits the lifetime of a Job that has finished
	// execution (either Complete or Failed). If this field is set, once the Job
	// finishes, it will be deleted after ttlSecondsAfterFinished expires. When
	// the Job is being deleted, its lifecycle guarantees (e.g. finalizers) will
	// be honored. If this field is unset, ttlSecondsAfterFinished will not
	// expire. If this field is set to zero, ttlSecondsAfterFinished expires
	// immediately after the Job finishes.
	// This field is alpha-level and is only honored by servers that enable the
	// TTLAfterFinished feature.
	// +optional
	TTLSecondsAfterFinished *int32
}
```

This allows Jobs to be cleaned up after they finish and provides time for
asynchronous clients to observe Jobs' final states before they are deleted.


##### Validation

Because Job controller depends on Pods to exist to work correctly. In Job
validation, `ttlSecondsAfterFinished` of its pod template shouldn't be set, to
prevent users from breaking their Jobs. Users should set TTL seconds on a Job,
instead of Pods owned by a Job.

It is common for higher level resources to call generic PodSpec validation;
therefore, in PodSpec validation, `ttlSecondsAfterFinished` is only allowed to
be set on a PodSpec with a `restartPolicy` that is either `OnFailure` or `Never`
(i.e. not `Always`).

### User Stories

The users keep creating Jobs in a small Kubernetes cluster with 4 nodes.
The Jobs accumulates over time, and 1 year later, the cluster ended up with more
than 100k old Jobs. This caused etcd hiccups, long high latency etcd requests,
and eventually made the cluster unavailable.

The problem could have been avoided easily with TTL controller for Jobs.

The steps are as easy as:

1. When creating Jobs, the user sets Jobs' `.spec.ttlSecondsAfterFinished` to
   3600 (i.e. 1 hour).
1. The user deploys Jobs as usual.
1. After a Job finishes, the result is observed asynchronously within an hour
   and stored elsewhere.
1. The TTL collector cleans up Jobs 1 hour after they complete.

### Implementation Details/Notes/Constraints

#### TTL Controller
We will add a TTL controller for finished Jobs. We considered
adding it in Job controller, but decided not to, for the following reasons:

1. Job controller should focus on managing Pods based on the Job's spec and pod
   template, but not cleaning up Jobs.
1. We also need the TTL controller to clean up finished Pods in the future, and we consider
   generalizing TTL controller later for custom resources. 

The TTL controller utilizes informer framework, watches all Jobs, and
read Jobs from a local cache.

#### Finished Jobs

When a Job is created or updated:

1. Check its `.status.conditions` to see if it has finished (`Complete` or
   `Failed`). If it hasn't finished, do nothing. 
1. Otherwise, if the Job has finished, check if Job's 
   `.spec.ttlSecondsAfterFinished` field is set. Do nothing if the TTL field is
   not set. 
1. Otherwise, if the TTL field is set, check if the TTL has expired, i.e. 
   `.spec.ttlSecondsAfterFinished` + the time when the Job finishes
   (`.status.conditions.lastTransitionTime`) > now. 
1. If the TTL hasn't expired, delay re-enqueuing the Job after a computed amount
   of time when it will expire. The computed time period is:
   (`.spec.ttlSecondsAfterFinished` + `.status.conditions.lastTransitionTime` -
   now).
1. If the TTL has expired, `GET` the Job from API server to do final sanity
   checks before deleting it.
1. Check if the freshly got Job's TTL has expired. This field may be updated
   before TTL controller observes the new value in its local cache.
   * If it hasn't expired, it is not safe to delete the Job. Delay re-enqueue
     the Job after a computed amount of time when it will expire.
1. Delete the Job if passing the sanity checks. 

#### Owner References

We have considered making TTL controller leave a Job/Pod around even after its
TTL expires, if the Job/Pod has any owner specified in its
`.metadata.ownerReferences`.

We decided not to block deletion on owners, because the purpose of
`.metadata.ownerReferences` is for cascading deletion, but not for keeping an
owner's dependents alive. If the Job is owned by a CronJob, the Job can be
cleaned up based on CronJob's history limit (i.e. the number of dependent Jobs
to keep), or CronJob can choose not to set history limit but set the TTL of its
Job template to clean up Jobs after TTL expires instead of based on the history
limit capacity. 

Therefore, a Job/Pod can be deleted after its TTL expires, even if it still has
owners. 

Similarly, the TTL won't block deletion from generic garbage collector. This
means that when a Job's or Pod's owners are gone, generic garbage collector will
delete it, even if it hasn't finished or its TTL hasn't expired. 

### Risks and Mitigations

Risks:
* Time skew may cause TTL controller to clean up resource objects at the wrong
  time.

Mitigations:
* In Kubernetes, it's required to run NTP on all nodes ([#6159][]) to avoid time
  skew. We will also document this risk.

[#6159]: https://github.com/kubernetes/kubernetes/issues/6159#issuecomment-93844058

### Test Plan

- Units test in kube-controller-manager package to test a variety of scenarios.
- Integration and E2E Tests to validate that jobs get deleted as expected

## Graduation Criteria

### Alpha

 - For alpha graduation, the feature implemented for Job, as future work it can be extended to Pods, but that should happen under a separate feature flag.
 - Unit and e2e tests

### Alpha -> Beta

- Appropriate metrics are agreed on and implemented
- upgrade/rollback manually tested 

### Beta -> GA

- TTL controller will be GA'ed without handling pods. The ability to extend TTL controller to work with pods can be introduced via a feature gate so that we can collect feedback and improve.
- Enabled in Beta for at least two releases without complaints

[umbrella issues]: https://github.com/kubernetes/kubernetes/issues/42752

## Production Readiness Review Questionnaire

### Feature Enablement and Rollback

* **How can this feature be enabled / disabled in a live cluster?**
  - [x] Feature gate (also fill in values in `kep.yaml`)
    - Feature gate name: TTLAfterFinished
    - Components depending on the feature gate: kube-apiserver, kube-controller-manager
  - [ ] Other
    - Describe the mechanism:
    - Will enabling / disabling the feature require downtime of the control
      plane?
    - Will enabling / disabling the feature require downtime or reprovisioning
      of a node? (Do not assume `Dynamic Kubelet Config` feature is enabled).

* **Does enabling the feature change any default behavior?**
  No.

* **Can the feature be disabled once it has been enabled (i.e. can we roll back
  the enablement)?**
  Yes. One caveat here is that Jobs created with TTLSecondsAfterFinished set when 
  the feature was enabled will continue to have that field set when the feature is disabled,
  but will not have any effect.

* **What happens if we reenable the feature if it was previously rolled back?**
  It should work as expected.

* **Are there any tests for feature enablement/disablement?**
  No.

### Rollout, Upgrade and Rollback Planning

* **How can a rollout fail? Can it impact already running workloads?**
  It shouldn't impact already running workloads. This is an opt-in feature since
  users need to explicitly set the TTLSecondsAfterFinished parameter in the job spec,
  if the feature is disabled the field is preserved if it was already set in the
  presisted Job object, otherwise it is silently dropped.

* **What specific metrics should inform a rollback?**
- Unexpected restarts of kube-controller-manager
- Extended 4xx/5xx on the Jobs endpoint from kube-apiserver 

* **Were upgrade and rollback tested? Was the upgrade->downgrade->upgrade path tested?**
  Manually tested. No issues were found.

* **Is the rollout accompanied by any deprecations and/or removals of features, APIs, 
fields of API types, flags, etc.?**
  No

### Monitoring Requirements

_This section must be completed when targeting beta graduation to a release._

* **How can an operator determine if the feature is in use by workloads?**
  - The `workqueue_adds_total{name="ttl_jobs_to_delete"}` tracks the number of 
    finished Jobs with ttlSecondsAfterFinished set.
  - Listing jobs in the cluster and checking if any has ttlSecondsAfterFinished field set.

* **What are the SLIs (Service Level Indicators) an operator can use to determine 
the health of the service?**
 - [x] Metrics
   - Components exposing the metric: `kube-controller-manager`
     - Metric name: `ttl_after_finished_controller_rate_limiter_use`
     - Metric name: `workqueue_adds_total{name="ttl_jobs_to_delete"}`
     - Metric name: `workqueue_depth{name="ttl_jobs_to_delete"}`
     - Metric name: `workqueue_queue_duration_seconds{name="ttl_jobs_to_delete"}`
     - Metric name: `workqueue_retries_total{name="ttl_jobs_to_delete"}`
   - Components exposing the metric: `kube-apiserver`
     - Metric name: `etcd_object_counts{resource="jobs.batch"}`


We will also add the following new histogram metric exposed by kube-controller-manager:
- `ttl_after_finished_controller_time_to_deletion_seconds` which tracks the time it took 
  the delete the job since it became eligible (actual-delete-timestamp - (job-finished-timestamp + ttlAfterFinished)).

* **What are the reasonable SLOs (Service Level Objectives) for the above SLIs?**

99% of the jobs that needs cleanup are deleted within X minutes.

This can be implemented using the `ttl_after_finished_controller_time_to_deletion_seconds` 
histogram.

* **Are there any missing metrics that would be useful to have to improve observability 
of this feature?**

No

### Dependencies

_This section must be completed when targeting beta graduation to a release._

* **Does this feature depend on any specific services running in the cluster?**
  No.

### Scalability

* **Will enabling / using this feature result in any new API calls?**
  - API call type: DELETE jobs
  - Estimated throughput: the upper bound is equal to Job creation rate.
  - originating component(s): kube-controller-manager

* **Will enabling / using this feature result in introducing new API types?**
  No.
  
* **Will enabling / using this feature result in any new calls to the cloud 
provider?**
 No.

* **Will enabling / using this feature result in increasing size or count of 
the existing API objects?**
 Yes. An int field is added to the Job object.

* **Will enabling / using this feature result in increasing time taken by any 
operations covered by [existing SLIs/SLOs]?**
  No.

* **Will enabling / using this feature result in non-negligible increase of 
resource usage (CPU, RAM, disk, IO, ...) in any components?**
  kube-controller-manager may consume more CPU depending on the number of jobs that require deletion in the system.

### Troubleshooting

_This section must be completed when targeting beta graduation to a release._

* **How does this feature react if the API server and/or etcd is unavailable?**
 The controller will not be notified of job updates and it can't deleted existing ones. 

* **What are other known failure modes?**
None.

* **What steps should be taken if SLOs are not being met to determine the problem?**
TBD

[supported limits]: https://git.k8s.io/community//sig-scalability/configs-and-limits/thresholds.md
[existing SLIs/SLOs]: https://git.k8s.io/community/sig-scalability/slos/slos.md#kubernetes-slisslos

## Future Work

As a future work, ttl-after-finished can be added to Pods. The API is similar to the Job's one:

```go
type PodSpec struct {
 	// ttlSecondsAfterFinished limits the lifetime of a Pod that has finished
	// execution (either Succeeded or Failed). If this field is set, once the Pod
	// finishes, it will be deleted after ttlSecondsAfterFinished expires. When
	// the Pod is being deleted, its lifecycle guarantees (e.g. finalizers) will
	// be honored. If this field is unset, ttlSecondsAfterFinished will not
	// expire. If this field is set to zero, ttlSecondsAfterFinished expires
	// immediately after the Pod finishes.
	// This field is alpha-level and is only honored by servers that enable the
	// TTLAfterFinished feature.
	// +optional
	TTLSecondsAfterFinished *int32
}
```

The TTL controller can be changed to watch Pods in addition to Jobs.

When a Pod is created or updated:
1. Check its `.status.phase` to see if it has finished (`Succeeded` or `Failed`).
   If it hasn't finished, do nothing. 
1. Otherwise, if the Pod has finished, check if Pod's
   `.spec.ttlSecondsAfterFinished` field is set. Do nothing if the TTL field is
   not set. 
1. Otherwise, if the TTL field is set, check if the TTL has expired, i.e.
   `.spec.ttlSecondsAfterFinished` + the time when the Pod finishes (max of all
   of its containers termination time
   `.containerStatuses.state.terminated.finishedAt`) > now. 
1. If the TTL hasn't expired, delay re-enqueuing the Pod after a computed amount
   of time when it will expire. The computed time period is:
   (`.spec.ttlSecondsAfterFinished` + the time when the Pod finishes - now).
1. If the TTL has expired, `GET` the Pod from API server to do final sanity
   checks before deleting it.
1. Check if the freshly got Pod's TTL has expired. This field may be updated
   before TTL controller observes the new value in its local cache.
   * If it hasn't expired, it is not safe to delete the Pod. Delay re-enqueue
     the Pod after a computed amount of time when it will expire.
1. Delete the Pod if passing the sanity checks. 

## Implementation History
- 2018-08-16: Initial KEP
- 2021-01-08: KEP updated to 
  - indicate that the feature will be graduated for Jobs, and that Pods will be done as future work under a separate flag
  - add production readiness questionnaire
  - mark the feature for Beta graduation for jobs.
- 2021-07-27: KEP updated to
  - indicate that the feature will be graduated to stable for Jobs
  - Pods will be done as future work if the need arises
