# KEP-2214: Indexed Job

<!-- toc -->
- [Release Signoff Checklist](#release-signoff-checklist)
- [Summary](#summary)
- [Motivation](#motivation)
  - [Goals](#goals)
  - [Non-Goals](#non-goals)
- [Proposal](#proposal)
  - [User Stories (Optional)](#user-stories-optional)
    - [Story 1](#story-1)
    - [Story 2](#story-2)
  - [Notes/Constraints/Caveats (Optional)](#notesconstraintscaveats-optional)
  - [Risks and Mitigations](#risks-and-mitigations)
- [Design Details](#design-details)
  - [JobSpec API](#jobspec-api)
  - [Pod detail](#pod-detail)
  - [Job completion and restart policy](#job-completion-and-restart-policy)
    - [Track completed indexes in Job status](#track-completed-indexes-in-job-status)
  - [Job parallelism](#job-parallelism)
  - [Test Plan](#test-plan)
  - [Graduation Criteria](#graduation-criteria)
    - [Alpha](#alpha)
    - [Alpha -&gt; Beta Graduation](#alpha---beta-graduation)
    - [Beta -&gt; GA Graduation](#beta---ga-graduation)
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
- [x] (R) Test plan is in place, giving consideration to SIG Architecture and SIG Testing input (including test refactors)
  - [x] e2e Tests for all Beta API Operations (endpoints)
  - [x] (R) Ensure GA e2e tests for meet requirements for [Conformance Tests](https://github.com/kubernetes/community/blob/master/contributors/devel/sig-architecture/conformance-tests.md) 
  - [x] (R) Minimum Two Week Window for GA e2e tests to prove flake free
- [x] (R) Graduation criteria is in place
  - [x] (R) [all GA Endpoints](https://github.com/kubernetes/community/pull/1806) must be hit by [Conformance Tests](https://github.com/kubernetes/community/blob/master/contributors/devel/sig-architecture/conformance-tests.md) 
- [x] (R) Production readiness review completed
- [x] (R) Production readiness review approved
- [x] "Implementation History" section is up-to-date for milestone
- [x] User-facing documentation has been created in [kubernetes/website], for publication to [kubernetes.io]
- [x] Supporting documentation—e.g., additional design documents, links to mailing list discussions/SIG meetings, relevant PRs/issues, release notes

[kubernetes.io]: https://kubernetes.io/
[kubernetes/enhancements]: https://git.k8s.io/enhancements
[kubernetes/kubernetes]: https://git.k8s.io/kubernetes
[kubernetes/website]: https://git.k8s.io/website

## Summary

This KEP extends kubernetes with user-friendly support for running parallel jobs.

Here, parallel means multiple pods per Job. Jobs can be:
- Embarrassingly parallel, where the pods have no dependencies between each other.
- Tightly coupled, where the Pods communicate among themselves to make progress
  (kubernetes/kubernetes#99497)[https://github.com/kubernetes/kubernetes/issues/99497]

We propose the addition of completion indexes into the Pods of a *Job
[with fixed completion count]* to support running embarrassingly parallel
programs, with a focus on ease of use for workload partitioning.
We call this new Job pattern an *Indexed Job*, because each Pod of the Job
specializes to work on a particular index, as if the Pods where elements of an
array.
With the addition of a headless Service, Pods can address another Pod with a
specific index with a DNS lookup, because the index is part of the hostname.

[with fixed completion count]: https://kubernetes.io/docs/concepts/workloads/controllers/job/#parallel-jobs

## Motivation

Users can use some [Job patterns] to run embarrassingly parallel Jobs, but those
approaches have downsides:

- The queue patterns require setting up an external queue service and modifying
  the Job binary to be able to connect to the queue.
  Depending on the implementation, it is prone to race conditions when
  coordinating which Pod works on which item.
- The template pattern doesn't scale when the parallelism level is too high,
  in terms of job creation and querying status.

Due to these reasons, workloads where each Pod just needs a unique and
ordered completion index, are hard to adapt to the existing Job patterns.

The lack of support for this pattern in k8s forces users to implement their
own APIs and controllers or adopt third party implementations. Each
implementation splits the ecosystem, making it harder for higher level systems
for Job queueing or workflows to support all of them.

Additionally, the Pods within a Job can't easily address and communicate with
each other, making it hard to run tightly coupled parallel Jobs using the Job
API.

Third-party operators cover these use cases by defining their own APIs, leading
to fragmentation of the ecosystem. The operators use mainly two networking
patterns: (1) fronting each index with a Service or (2) creating Pods with
stable hostnames based on their index.

Using a Service per index has scalability problems. Other than the Service
objects themselves, the control plane creates an Endpoint object.

Creating Pods with stable hostnames mitigates this problem. The control plane
requires only one headless Service and one Endpoint (or a few EndpointSlices) to
inform the DNS programming. Pods can address each other with a DNS lookup and
communicate directly using Pod IPs.

A popular operator chose to use a StatefulSet to handle Pod creation and
management with these characteristics. Due to limitations, the operator now
manages plain pods. These limitations of StatefulSet were:
- Pods are created serially.
- Pods can be replaced without leaving notice of failures.
- Pods cannot run to completion (containers restart on success or failure).

[Job patterns]: https://kubernetes.io/docs/concepts/workloads/controllers/job/#job-patterns

### Goals

- Support the *indexed Job* pattern by adding completion indexes to each Pod
  of a Job in *fixed completion count* mode.
- Add stable hostnames to Pods based on the index to simplify communication 
  among themselves.

### Non-Goals

- Support for work lists, where each Pod receives a different element of a
  static list. This can be implemented by users from completion indexes.
- Support for completion index in non-parallel Jobs or Jobs with a work queue.
- Network programming for indexed Jobs. This is left to headless Services.
- All-or-nothing scheduling.

## Proposal

### User Stories (Optional)

#### Story 1

As a Job author, I can create an Indexed Job where each Pod receives an ordered
completion index. I can use the index in my binary through an environment
variable or a file to statically select the load the Pod should work on.

```yaml
apiVersion: batch/v1
kind: Job
metadata:
  name: my-job
spec:
  completions: 100
  parallelism: 100
  completionMode: Indexed
  template:
    spec:
      containers:
      - name: task
        image: registry.example.com/processing-image
        command: ["./process",  "--index", "$JOB_COMPLETION_INDEX"]
```

#### Story 2

As a Job author, I can create an Indexed Job where pods can address each other
by the hostname that can be built from the index.

```yaml
apiVersion: batch/v1
kind: Job
metadata:
  name: my-job
spec:
  completions: 100
  parallelism: 100
  completionMode: Indexed
  template:
    metadata:
      labels:
        job: my-job
    spec:
      subdomain: my-job-svc
      containers:
      - name: task
        image: registry.example.com/processing-image
        command: ["./process",  "--index", "$JOB_COMPLETION_INDEX", "--hosts-pattern", "my-job-{{.id}}.my-job-svc"]
```

```yaml
apiVersion: v1
kind: Service
metadata:
  name: my-job-svc
spec:
  clusterIP: None
  selector:
    job: my-job
```

### Notes/Constraints/Caveats (Optional)

* An earlier proposal for [indexed Job]
suggested the support for work lists, i.e. passing different parameters to each
Pod. We decided to leave this out of the proposal to keep it simple and
because work lists can be implemented in a startup script using the completion
index as building block.
* The semantics of an indexed Job are similar to a StatefulSet, in the sense
that Pods have an associated index.
However, the APIs have major differences:
  - a StatefulSet doesn't have completion semantics, as opposed to Jobs.
  - a StatefulSet creates pods serially, whereas Job creates all Pods in
    parallel.
  - a StatefulSet gives Pods stable hostnames, a Job doesn't.

[indexed Job]: https://github.com/kubernetes/community/blob/b21d1b27c8c748bf81283c2d89cde2becb5f2709/contributors/design-proposals/apps/indexed-job.md

### Risks and Mitigations

- More than one pod per index

  Jobs have a known issue in which more than one Pod can be started even if
  parallelism and completion are set to 1 ([reference]). In the case of indexed
  Jobs, this translates to more than one Pod having the same index.

  Just like for existing Job patterns, workloads have to handle duplicates at the
  application level.

- Jobs with a high number of parallelism produce starvation on small jobs
  
  This problem is not unique to Indexed Jobs, but the new API might motivate
  use cases with higher degree of parallelism.
  
  In a Job sync, the controller will be limited to create or delete up to 500
  Pods. The controller processes the remaining operations in subsequent syncs,
  which it schedules with no delay.

- Scalability and latency of DNS programming, if users choose to pair the
  Indexed Job with a headless service.

  DNS programming requires the update of Endpoint or EndpointSlices by the
  control plane and updating DNS records by the DNS provider.
  This might not scale well for short-lived Jobs with high number of
  parallelism.
  
  Thus, Pods need to be prepared to:
  - Retry lookups, when the control plane didn't have time to update the records.
  - Handle the IPs for a CNAME to change, in the case of a Pod failure.
  - Handle more than one IP for the CNAME. This might happen temporarily when
    the job controller creates more than one pod per index. The controller
    corrects this in the next sync, deleting the Pod that started last, which
    should correspond to the last IP added to the record.
  In short, Pods are ephemeral and resolutions might change, so users shouldn't
  rely on DNS caches.
  
  However, DNS programming is opt-in (users need to create a matching
  headless Service). Moreover, workloads have other means of obtaining IPs,
  such as querying/watching the API server. Vendors can also choose to implement
  alternate DNS programming tailored for Jobs.

[reference]: https://kubernetes.io/docs/concepts/workloads/controllers/job/#handling-pod-and-container-failures

## Design Details

### JobSpec API

The JobSpec gets the field `completionMode` to control whether the Job should	
be treated as an Indexed Job.	

```golang
// CompletionMode specifies how Pod completions of a Job are tracked.
type CompletionMode string

const (
	// NonIndexedCompletion means that Pod completions of a Job are
	// indistinguishable from each other.
	NonIndexedCompletion CompletionMode = "NonIndexed"

	// IndexedCompletion means that each Pod completion of a Job is tracked
	// individually, being associated to a completion index.
	IndexedCompletion CompletionMode = "Indexed"
)

type JobSpec struct {	
  ...	
  // CompletionMode specifies how Pod completions are tracked. It can be
  // `NonIndexed` (default) or `Indexed`.
  //
  // `NonIndexed` means that the Job is considered complete when there have
  // been .spec.completions successfully completed Pods. Each Pod completion is
  // homologous to each other.
  //
  // `Indexed` means that the Pods of a
  // Job get an associated completion index from 0 to (.spec.completions - 1),
  // available in the annotation batch.kubernetes.io/job-completion-index.
  // The Pod hostnames are set to $(job-name)-$(index) and the names to
  // $(job-name)-$(index)-$(random-suffix).
  // The Job is considered complete when there is one successfully completed Pod
  // for each index.
  // When value is `Indexed`, .spec.completions must be specified and
  // `.spec.parallelism` must be less than or equal to 10^5.
  // More completion modes can be added in the future. If the Job controller
  // observes a mode that it doesn't recognize, the controller skips updates
  // for the Job.
  CompletionMode *CompletionMode
}	

type JobStatus struct {
  ...

  // CompletedIndexes holds the completed indexes when .spec.completionMode =
  // "Indexed" in a text format. The indexes are represented as decimal integers
  // separated by commas. The numbers are listed in increasing order. Three or
  // more consecutive numbers are compressed and represented by the first and
  // last element of the series, separated by a hyphen.
  // For example, if the completed indexes are 1, 3, 4, 5 and 7, they are
  // represented as "1,3-5,7".
  CompletedIndexes string
}
```

As the comment describes, when `.spec.completionMode = "Indexed"`:

- `.spec.completions` must be a non-zero positive value. This is to trigger Job
  management strategy for *fixed completion count*. That is, `Indexed` mode
  cannot be used for work queue patterns.	
- `.spec.parallelism` must be less than or equal to `10^5`. This is to guarantee
  that we can keep track of completions per-index in the Job status.

### Pod detail

The Pod and PodSpec APIs don't get any new fields. However, Pods created for
Indexed Jobs get the annotation `batch.kubernetes.io/job-completion-index`
with a value equal to its completion index. The annotation is immutable.

The annotation can be accessed through the downward API as a file or environment
variable.

For user convenience, the Job controller adds the completion index as an
environment variable through the downward API. That is, the Job controller
creates Pods like so:

```yaml
apiVersion: v1
kind: Pod
spec:
  containers:
    - name: test-container
      env:
        - name: JOB_COMPLETION_INDEX
          valueFrom:
            fieldRef:
              fieldPath: metadata.annotations['batch.kubernetes.io/job-completion-index'] 
```

The Job controller doesn't add the environment variable if there is a name
conflict with an existing environment variable. Users can specify other
environment variables for the same annotation.

The Pod name takes the form `$(job-name)-$(index)-$(random-string)`,
which can be used for quickly identifying Pods for a specific index when listing
pods or looking at logs.

The Pod hostname takes the form `$(job-name)-$(index)` which can be used to
address the Pod from others, when the Job is used in combination with a headless
Service.

### Job completion and restart policy

When dealing with Indexed Jobs, the Job controller keeps track of Pod
completions for each index from 0 to `.spec.completions - 1`.
Once the controller notices that a Pod finishes successfully, it will not create
another Pod with the same index.

The Job controller considers a Job completed when there is at least one
successful Pod for each completion index.

If an entire Pod fails, such as when the Pod gets kicked off the node or if a
container of the Pod fails and `.spec.template.spec.restartPolicy = "Never"`,
the Job controller gets the completion index for the failed Pod and creates a
new Pod with the same index. The application needs to handle restarts in a
different Pod.

The Pod might not be immediately replaced, as the number of active Pods could
have hit the parallelism limit. Once another Pod finishes, the Job controller
syncs the Job, scanning for unsatisfied indexes and creates the missing Pods for
them.

The kubelet handles container restarts as usual, according to the
`spec.template.spec.restartPolicy`.

#### Track completed indexes in Job status

The Job controller keeps track of completed indexes in
`.status.completedIndexes`, a string that represents a list of numbers in a
compressed format. For example, if a Job has completed indexes 2, 3, 4, 6 and 7,
the list looks like:

```golang
CompletedIndexes: "2-4,6-7"
```

The `kubectl describe` command crops the list of indexes if it's too long:

```
Completed Indexes: 1-25,28,30-32,...
``` 

### Job parallelism

A user can change the number of active Pods for a Job changing `.spec.parallelism`
(note that `.spec.completions` is an immutable field).

When starting a Job or increasing the parallelism, the Job controller creates
Pods with lower completion index first, as long as there is no other completed
or running Pod with the same index. This is to make the controller behavior more
predictable. We do not offer guarantees on creation order based on completion
index.

Reducing parallelism is unaffected by completion index.

### Test Plan

Unit, integration and E2E tests cover the following Indexed Job mechanics:

  - Creation with index annotations and indexed pod hostnames.
  - Scale up and down.
  - Pod failures.
  
Additionally, we add unit tests for API defaulting and validation, with feature
gate enabled and disabled.
  
### Graduation Criteria

#### Alpha

- Complete features:
  - Completion index in annotation
  - Restart policy

#### Alpha -> Beta Graduation

- Complete features:
  - Index as part of the pod name and hostname.
  - Indexed Jobs when tracking completion with finalizers.
    [kubernetes/enhancements#2307](https://github.com/kubernetes/enhancements/issues/2307).
    
    Keeping the size of .status.completedIndexes is desirable to reduce load
    on watchers. We will evaluate holding of from counting completed Pods that
    have an outlying index. That is, contiguous indexes would be counted first.
    This allows to keep the size of the compressed list small.
  - Add metrics.
- Enable feature gate IndexedJob by default.
- Tests are in Testgrid and linked in KEP

#### Beta -> GA Graduation

- [E2E](https://testgrid.k8s.io/sig-apps#gce&include-filter-by-regex=Indexed) test graduates to conformance
- Scalability tests for Jobs of varying sizes, up to 500 parallelism, that keep
  track of metric `job_sync_duration_seconds`.

  Using a [clusterloader2 test](https://github.com/kubernetes/perf-tests/pull/1998)
  that creates 101 jobs of varying sizes (total of 1200 pods) on a 20 nodes cluster, 
  with 100 QPS for the job controller, I obtained the following completion times (averaged for 5 runs):
    - NonIndexed jobs: 34.2s
    - Indexed jobs: 33.4s
  
  The slight improvement for Indexed Jobs can be attributed to one less API call
  necessary to track job status with finalizers.

### Upgrade / Downgrade Strategy

In the event of a kube-controller-manager upgrade, there should not be any
existing Indexed Jobs with running Pods.

In the event of a downgrade, existing Indexed Jobs will run as NonIndexed Jobs.
The controller will track the existing Pods ignoring the completion index. New
Pods will be created without a completion index. Existing workloads that
expected the completion index will fail. But this is expected in a downgrade.

In the event of an upgrade after a downgrade, the controller will remove
existing Pods without completion index for existing Indexed Jobs, without they
counting towards failures or completions. The controller will create new Pods
with indexes. This might cause a load spike in the cluster.

If, instead of a downgrade, the cluster administrator disables the feature gate:

  - kube-apiserver clears `.spec.completionMode` for new Jobs at creation time.
    That is, all new Jobs are interpreted as `NonIndexed`.
  - kube-controller-manager skips syncing existing Indexed Jobs and emits a
    warning event. More specifically, the controller does not create new Pods,
    track completion nor update status of the Job.
  
The above guarantees that the controller never creates Pods for Indexed Jobs
without a completion index.

### Version Skew Strategy

This feature has no node runtime implications.

## Production Readiness Review Questionnaire

### Feature Enablement and Rollback

###### How can this feature be enabled / disabled in a live cluster?


  - [x] Feature gate (also fill in values in `kep.yaml`)
    - Feature gate name: IndexedJob
    - Components depending on the feature gate:
      - kube-apiserver
      - kube-controller-manager

###### Does enabling the feature change any default behavior?

  No. Jobs need to opt-in with `.spec.completionMode=Indexed`.

###### Can the feature be disabled once it has been enabled (i.e. can we roll back the enablement)?
  
  Yes. Using the feature gate is the recommended way.

###### What happens if we reenable the feature if it was previously rolled back?

  The Job controller starts managing Indexed Jobs again.
  More details covered in [Upgrade / Downgrade Strategy](#upgrade--downgrade-strategy).

###### Are there any tests for feature enablement/disablement?

  Yes, unit and integration test for the feature enabled, disabled and
  transitions.

### Rollout, Upgrade and Rollback Planning

###### How can a rollout or rollback fail? Can it impact already running workloads?

  If the new kube-controller-manager crashes, it's possible that an older
  version of it would pick it up. In 1.21, when the IndexedJob feature is
  disabled (default), the controller would not sync Indexed Jobs, that is: the
  controller doesn't create or delete Pods and doesn't update Job status.
  Running Pods are not affected.

###### What specific metrics should inform a rollback?

  - job_sync_duration_seconds shows significantly more latency for label
    completion_mode=Indexed Jobs than completion_mode=NonIndexed.
  - job_sync_total shows more errors for completion_mode=Indexed than
    completion_mode=NonIndexed.
  - job_finished_total shows that Jobs with completion_mode=Indexed don't finish.

###### Were upgrade and rollback tested? Was the upgrade->downgrade->upgrade path tested?

  Manual test performed:
  
  1. Deploy k8s 1.21 cluster
  1. Upgrade to 1.22
  1. Create Indexed Job with big number of completions and pods that run for ~10min.
  1. Downgrade to 1.21. Verify that no new pods are created for the Indexed Job.
  1. Upgrade to 1.22. Verify that new pods are created for Indexed Job.

###### Is the rollout accompanied by any deprecations and/or removals of features, APIs, fields of API types, flags, etc.?

  No

### Monitoring Requirements

###### How can an operator determine if the feature is in use by workloads?

  - job_sync_total has values for the label completion_mode=Indexed.

###### How can someone using this feature know that it is working for their instance?

- [x] Events
  - Event Reason: SuccessfulCreate
  - The message includes the pod name.
- [x] API .status
  - Condition name: 
  - Other field: `completedIndexes` will not be empty as pods terminate.

###### What are the reasonable SLOs (Service Level Objectives) for the enhancement?

  - per-day percentage of job_sync_total with label result=error <= 1%
  - 99% percentile over day for job_sync_duration_seconds is <= 15s, assuming
    a client-side QPS limit of 50 calls per second. Note that this is the
    expected SLO for NonIndexed jobs as well.

###### What are the SLIs (Service Level Indicators) an operator can use to determine the health of the service?

  - [x] Metrics
    - Metric name (all new):
      - job_sync_duration_seconds: tracks the latency of a Job sync.
      - job_sync_total: tracks the number of Job syncs.
      - job_finished_total: tracks the number of Jobs that finish as
        result=failed/succeeded
    - Components exposing the metric: kube-controller-manager

###### Are there any missing metrics that would be useful to have to improve observability of this feature?

  No

### Dependencies

###### Does this feature depend on any specific services running in the cluster?

  No, the feature only involves kube-apiserver and kube-controller-manager.


### Scalability

<!--
For alpha, this section is encouraged: reviewers should consider these questions
and attempt to answer them.

For beta, this section is required: reviewers must answer these questions.

For GA, this section is required: approvers should be able to confirm the
previous answers based on experience in the field.
-->

###### Will enabling / using this feature result in any new API calls?

  No.

###### Will enabling / using this feature result in introducing new API types?

  No.

###### Will enabling / using this feature result in any new calls to the cloud provider?

  No.

###### Will enabling / using this feature result in increasing size or count of the existing API objects?

  Yes.
  
  - API type(s): Job
  - Estimated increase in size:
    - New field in Spec about 30 bytes.
    - New field in Status. In the worst case scenario, completed indexes are
      non-consecutive. Since the API limits parallelism to 10^5, we could have
      up to 5*10^4 non-consecutive numbers, which can be represented in less
      than 1MB.
  
  - API type(s): Pod, only when created with the new completion mode.
  - Estimated increase in size: new annotation of about 50 bytes and hostname
    which includes the index.

###### Will enabling / using this feature result in increasing time taken by any operations covered by existing SLIs/SLOs?

  No.

###### Will enabling / using this feature result in non-negligible increase of resource usage (CPU, RAM, disk, IO, ...) in any components?

  Additional CPU and memory increase in the controller-manager is negligible
  and restricted to Jobs using the new completion mode.

### Troubleshooting

###### How does this feature react if the API server and/or etcd is unavailable?

  The job controller can't create or delete pods nor update job status.
  The metric job_sync_total increases for label result=error.
  Existing pods continue to run.

###### What are other known failure modes?

  None.

###### What steps should be taken if SLOs are not being met to determine the problem?

  1. Check job_sync_total with label result=error. See if it varies for
     different completion modes.
  1. Verify if kube-apiserver is healthy. If not, the Job controller can't operate.
  1. Check job_sync_duration_seconds. If the latency is increased, verify if it
     varies for different completion modes.
     Note that latency increases linearly with the Job's parallelism.

[supported limits]: https://git.k8s.io/community//sig-scalability/configs-and-limits/thresholds.md
[existing SLIs/SLOs]: https://git.k8s.io/community/sig-scalability/slos/slos.md#kubernetes-slisslos

## Implementation History

* 2021-01-08: First version of the KEP in provisional status. Design Details
  completed.
* 2021-03-09: Feature implemented under feature gate disabled by default.
* 2021-04-09: KEP updated for graduation to beta.
* 2022-01-06: KEP updated for graduation to stable.

## Drawbacks

* Adds more complexity to the Job controller in terms of Pod and Pod status
  management, as it introduces a new mode.

## Alternatives

- **Leave Indexed Job to third-party implementations**

  The major painpoint is that this leaves Pod management to the third-party
  implementation. With different implementations, the ecosystem is split, making
  it harder for higher level Job orchestration frameworks to support all of
  them.
  
  On the other hand, with the Indexed Job native support in core k8s,
  third-party implementations can focus on application level APIs, using the Job
  API as their underlying Pod management mechanism.
  
- **Completion Index in the Pod Name**

  Completion indexes could also be part of the Pod name, leading to stable Pod
  names. This allows 2 things:
  - Uniqueness for each completion index. This frees applications from having to
    handle duplicated indexes. When used along with a headless Service, there
    are less chances for a DNS record to refer to more than one Pod.
  
  Stable pod names require the Job controller to remove failed Pods before
  creating a new one with the same index. This has some downsides:
  - Removing Job Pods is a breaking change. But this can be done if it's a new
    Job execution mode accessible through a JobSpec field.
  - Currently, the Job controller uses the tombstones of failed Pods to track
    the status of the Job, affecting retry backoffs and backoff limit. This
    needs to change before stable Pod names can be implemented
    [#28486](https://github.com/kubernetes/kubernetes/issues/28486).
  - Reduced availability of Job Pods per completion index as, in addition to
    the time necessary to create a new Pod, we need to account for the time of
    deleting the failed Pod.
    
  However, stable Pod names can be offered later as a new value for
  `.spec.completionMode` for Jobs.
