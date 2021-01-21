<!--
**Note:** When your KEP is complete, all of these comment blocks should be removed.

To get started with this template:

- [ ] **Pick a hosting SIG.**
  Make sure that the problem space is something the SIG is interested in taking
  up. KEPs should not be checked in without a sponsoring SIG.
- [ ] **Create an issue in kubernetes/enhancements**
  When filing an enhancement tracking issue, please make sure to complete all
  fields in that template. One of the fields asks for a link to the KEP. You
  can leave that blank until this KEP is filed, and then go back to the
  enhancement and add the link.
- [ ] **Make a copy of this template directory.**
  Copy this template into the owning SIG's directory and name it
  `NNNN-short-descriptive-title`, where `NNNN` is the issue number (with no
  leading-zero padding) assigned to your enhancement above.
- [ ] **Fill out as much of the kep.yaml file as you can.**
  At minimum, you should fill in the "Title", "Authors", "Owning-sig",
  "Status", and date-related fields.
- [ ] **Fill out this file as best you can.**
  At minimum, you should fill in the "Summary" and "Motivation" sections.
  These should be easy if you've preflighted the idea of the KEP with the
  appropriate SIG(s).
- [ ] **Create a PR for this KEP.**
  Assign it to people in the SIG who are sponsoring this process.
- [ ] **Merge early and iterate.**
  Avoid getting hung up on specific details and instead aim to get the goals of
  the KEP clarified and merged quickly. The best way to do this is to just
  start with the high-level sections and fill out details incrementally in
  subsequent PRs.

Just because a KEP is merged does not mean it is complete or approved. Any KEP
marked as `provisional` is a working document and subject to change. You can
denote sections that are under active debate as follows:

```
<<[UNRESOLVED optional short context or usernames ]>>
Stuff that is being argued.
<<[/UNRESOLVED]>>
```

When editing KEPS, aim for tightly-scoped, single-topic PRs to keep discussions
focused. If you disagree with what is already in a document, open a new PR
with suggested changes.

One KEP corresponds to one "feature" or "enhancement" for its whole lifecycle.
You do not need a new KEP to move from beta to GA, for example. If
new details emerge that belong in the KEP, edit the KEP. Once a feature has become
"implemented", major changes should get new KEPs.

The canonical place for the latest set of instructions (and the likely source
of this file) is [here](/keps/NNNN-kep-template/README.md).

**Note:** Any PRs to move a KEP to `implementable`, or significant changes once
it is marked `implementable`, must be approved by each of the KEP approvers.
If none of those approvers are still appropriate, then changes to that list
should be approved by the remaining approvers and/or the owning SIG (or
SIG Architecture for cross-cutting KEPs).
-->
# KEP-2307: Job tracking without lingering Pods

<!-- toc -->
- [Release Signoff Checklist](#release-signoff-checklist)
- [Summary](#summary)
- [Motivation](#motivation)
  - [Goals](#goals)
  - [Non-Goals](#non-goals)
- [Proposal](#proposal)
  - [Notes/Constraints/Caveats (Optional)](#notesconstraintscaveats-optional)
  - [Risks and Mitigations](#risks-and-mitigations)
    - [New API calls](#new-api-calls)
    - [Bigger Job status](#bigger-job-status)
    - [Unprotected Job status endpoint](#unprotected-job-status-endpoint)
- [Design Details](#design-details)
  - [API changes](#api-changes)
  - [Algorithm](#algorithm)
  - [Deleted Pods](#deleted-pods)
  - [Pod adoption](#pod-adoption)
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
- [x] (R) Test plan is in place, giving consideration to SIG Architecture and SIG Testing input
- [x] (R) Graduation criteria is in place
- [x] (R) Production readiness review completed
- [x] (R) Production readiness review approved
- [ ] "Implementation History" section is up-to-date for milestone
- [ ] User-facing documentation has been created in [kubernetes/website], for publication to [kubernetes.io]
- [ ] Supporting documentation—e.g., additional design documents, links to mailing list discussions/SIG meetings, relevant PRs/issues, release notes

[kubernetes.io]: https://kubernetes.io/
[kubernetes/enhancements]: https://git.k8s.io/enhancements
[kubernetes/kubernetes]: https://git.k8s.io/kubernetes
[kubernetes/website]: https://git.k8s.io/website

## Summary

The current Job controller currently relies on completed Pods to not be removed
in order to track the Job completion status. This proposal presents an
alternative implementation for Job tracking that does not have this dependency.

## Motivation

The current approach of relying on the Pods existence is problematic for Jobs
that require a big number of completions or for clusters with too many Jobs
running at the same time. The finished Pods cannot be removed until the entire
Job completes, even if the Pods failed.

Furthermore, once the number of finished Pods reaches a threshold, the Pod
garbage collection controller starts removing Pods. Today, the Job controller
relies on the garbage collector having a big threshold.

### Goals

- Perform Job completion and failures tracking without relying on lingering
  Pods.

### Non-Goals

- Remove Pods once they have been accounted for.

## Proposal

The Job controller creates Pods with a finalizer to prevent finished Pods from
being removed by the garbage collector. The Job controller removes the finalizer
from the finished Pods once it has accounted for them. In subsequent Job syncs,
the controller ignores finished Pods that don't have the finalizer.

### Notes/Constraints/Caveats (Optional)

Due to the lack of support of atomic changes across kubernetes objects, an
intermediate state is necessary. Before removing the Pod finalizers,
the controller adds finished Pods to a list in the Job status. After removing
the Pod finalizers, the controller clears the list and updates the counters.

### Risks and Mitigations

#### New API calls

The new algorithm introduces new API calls:
- one per Pod lifecycle to remove finalizers
- one for each Job sync, to track the intermediate state.

Note that there no new calls in the Pod creation path.

On the other hand, to update the Job status once a Pod finishes, we need a total
2 new API calls compared to the legacy algorithm. If more than one Pod finish at
a given time, we add `n + 1` API calls, where `n` is the number finished Pods.

Consider a Job with multiple Pods. With a 50 QPS limit in the job controller
client, the controller should be able to process between 2000 to 3000 Pods.

The increase in API calls is justified for the following reasons:
- The legacy tracking cannot handle a big number of terminated Pods, across
  any number of Jobs, at a given point.
- In the entirety of its lifecycle, a Pod requires at least 8 API calls,
  including status updates and events.
  
However, in order to prevent Jobs with big number of Pods from starving Jobs
with fewer Pods, the Job controller might skip status updates until enough
Pods have accumulated or enough time has passed. See [Algorithm](#algorithm)
below for more details.
   
#### Bigger Job status

Job status can temporarily grow if too many Pods finish at the same time or
if the Job controller is down for some time. In this case, we do partial
status updates. See [Algorithm](#algorithm) below.

#### Unprotected Job status endpoint

Changes in the status not produced by the Job controller in
kube-controller-manager could affect the Job tracking. Cluster administrators
should make sure to protect the Job status endpoint via RBAC.

## Design Details

### API changes

The Job status gets a new struct to hold the uncounted Pods before they are
added to the counters.

```golang
type JobStatus struct {
    Succeeded int32
    Failed    int32
    ...

    // UncountedTerminatedPods holds UIDs of Pods that have finished but
    // haven't been accounted in the counters.
    // If nil, Job tracking doesn't make use of this structure.
    // +optional
    UncountedTerminatedPods *UncountedTerminatedPods
}

// UncountedTerminatedPods holds UIDs of Pods that have finished but haven't
// been accounted in Job status counters.
type UncountedTerminatedPods struct {
    // Succeeded holds UIDs of succeeded Pods.
    Succeeded []types.UID
    // Succeeded holds UIDs of failed Pods.
    Failed    []types.UID
}
```

Note: the final name of the field `uncountedTerminatedPods` will be decided
during API review.

### Algorithm

The following algorithm updates the status counters without relying on finished
Pods to be present indefinitely. The algorithm assumes that the Job controller
could be stopped at any point and executed again from the first step without
losing information. Generally, all the steps happen in a single Job sync
cycle.

1. The Job controller calculates the number of succeeded Pods as the sum of:
   - `.status.succeeded`,
   - the size of `job.status.uncountedTerminatedPods.succeeded` and
   - the number of finished Pods that are not in `job.status.uncountedTerminatedPods.succeeded`.
   This number informs the creation of missing Pods to reach `.spec.completions`.
   The controller creates Pods for a Job with the finalizer
   `batch.kubernetes.io/job-completion`.
2. The Job controller adds Pod UIDs to the `.status.uncountedTerminatedPods.succeeded`
   and `.status.uncountedTerminatedPods.failed` lists if the Pod:
    - has the `batch.kubernetes.io/job-completion` finalizer, and
    - the Pod is on Succeeded or Failed phase, respectively.
    The controller sends a status update.
3. The Job controller removes the `batch.kubernetes.io/job-completion` finalizer
   from all Pods on Succeeded or Failed phase that were added to the lists in
   `.status.uncountedTerminatedPods` in the previous step.
4. The Job controller counts the Pods in the `.status.uncountedTerminatedPods` lists
   that:
   - have no finalizer, or
   - were removed from the system.
   The counts increment the `.status.failed` and `.status.succeeded` and clears
   counted Pods from `.status.uncountedPodsUIDs` lists. The controller sends a
   status update.

Steps 2 to 4 might deal with a potentially big number of Pods. Thus, status
updates can potentially stress the kube-apiserver. For this reason, the Job
controller repeats steps 2 to 4, capping the number of Pods each time, until
the controller processes all the Pods. The number of Pods is caped by:
- time: in each iteration, the job controller removes all the Pods' finalizers
  it can in a unit of time in the order of tens of seconds. This allows to
  throttle the number of Job status updates.
- count: Preventing big writes to etcd. We limit the number of UIDs to the order
  of hundreds, keeping the size of the slice under 20kb.

If any Pod finalizer removal fails in step 3, the controller manager still
executes step 4 with the Pods that succeeded.

Steps 2 to 4 might be skipped in the scenario where a status update happened
too recently and the number of uncounted Pods is a small percentage of
parallelism.

### Deleted Pods
   
In the case where a user or another controller removes a Pod, which sets a
deletion timestamp, the Job controller treats it the same as any other Pod.
That is, once it reaches Failed status, the controller accounts for the Pod and
then removes the finalizer.

This is different from the legacy tracking, where the Job controller does not
account for deleted Pods. This is a limitation that this KEP also wants to
solve.

However, if the Job controller deletes the Pod (when parallelism is decreased,
for example), the controller removes the finalizer before deleting it. Thus,
these deletions don't count towards the failures.
   
### Pod adoption

If a Job with `.status.uncountedTerminatedPods != nil` can adopt a Pod
(according to the existing adoption criteria), this Pod might not have a
finalizer.

The job controller adds the finalizer in the same patch request that modifies
the owner reference.

### Test Plan

- Unit tests:
  - Job sync with feature gate enabled.
  - Removal of finalizers when feature gate is disabled.
  - Tracking of terminating Pods.
- Integration tests:
  - Job tracking with feature enabled.
  - Tracking of terminating Pods.
  - Transition from feature enabled to disabled and enabled again.
  - Tracking Jobs with big number of Pods, making sure the status is eventually
    consistent.
- E2E test:
  - Job tracking with feature enabled.

### Graduation Criteria

#### Alpha

- Implementation:
  - Job tracking without lingering Pods
  - Removal of finalizer when feature gate is disabled.
- Tests: unit, integration, E2E

#### Alpha -> Beta Graduation

- Support for [Indexed Jobs](https://git.k8s.io/enhancements/keps/sig-apps/2214-indexed-job)
- Processing 5000 Pods per minute across any number of Jobs, with Pod creation
  having higher priority than status updates. This might depend on
  [Priority and Fairness](https://git.k8s.io/enhancements/keps/sig-api-machinery/1040-priority-and-fairness).
- Metrics:
  - latency
  - errors
- Tests are in Testgrid and linked in KEP

#### Beta -> GA Graduation

- E2E test graduates to conformance.
- Job tracking scales to 10^5 completions per Job processed within an order of
  minutes.

### Upgrade / Downgrade Strategy

When the feature `JobTrackingWithoutLingeringPods` is enabled for the first
time, the cluster can have Jobs whose Pods don't have the
`batch.kubernetes.io/job-completion` finalizer. It would be hard to add the
finalizer to all Pods while preventing race conditions.

We use `.status.uncountedTerminatedPods != nil` to indicate whether the Job
was created after the feature was enabled. If this field is nil, the Job
controller tracks Pods using the legacy tracking.

The kube-apiserver sets `.status.uncountedTerminatedPods` to an empty struct
when the feature gate `JobTrackingWithoutLingeringPods` is enabled, at Job
creation. In alpha, apiserver leaves `.status.uncountedTerminatedPods = nil`
for [Indexed Jobs](https://git.k8s.io/enhancements/keps/sig-apps/2214-indexed-job)

When the feature is disabled after being enabled for some time, the next time
the Job controller syncs a Job:
1. It removes finalizers from all Pods owned by the Job.
2. Sets `.status.uncountedTerminatedPods` to nil.

### Version Skew Strategy

No implications to node runtime.

## Production Readiness Review Questionnaire

### Feature Enablement and Rollback

_This section must be completed when targeting alpha to a release._

* **How can this feature be enabled / disabled in a live cluster?**
  - [x] Feature gate (also fill in values in `kep.yaml`)
    - Feature gate name: JobTrackingWithoutLingeringPods
    - Components depending on the feature gate:
      - kube-apiserver
      - kube-controller-manager
  - [ ] Other
    - Describe the mechanism:
    - Will enabling / disabling the feature require downtime of the control
      plane?
    - Will enabling / disabling the feature require downtime or reprovisioning
      of a node? (Do not assume `Dynamic Kubelet Config` feature is enabled).

* **Does enabling the feature change any default behavior?**

  Yes.
  
  - Removing terminated Pods doesn't affect Job status.
  - Pods removed by the user or other controllers count towards failures or
    completions.

* **Can the feature be disabled once it has been enabled (i.e. can we roll back
  the enablement)?**
  
  Yes.
  The job controller removes finalizers in this case.
  Since some succeeded Pods might have been removed, the job controller will
  create new Pods to fulfill completions. But this is no different from
  existing behavior with the legacy tracking.

* **What happens if we reenable the feature if it was previously rolled back?**

  Existing Jobs are tracked with legacy Pods.

* **Are there any tests for feature enablement/disablement?**

  Yes, we plan to add integration tests.

### Rollout, Upgrade and Rollback Planning

_This section must be completed when targeting beta graduation to a release._

* **How can a rollout fail? Can it impact already running workloads?**
  Try to be as paranoid as possible - e.g., what if some components will restart
   mid-rollout?

* **What specific metrics should inform a rollback?**

* **Were upgrade and rollback tested? Was the upgrade->downgrade->upgrade path tested?**
  Describe manual testing that was done and the outcomes.
  Longer term, we may want to require automated upgrade/rollback tests, but we
  are missing a bunch of machinery and tooling and can't do that now.

* **Is the rollout accompanied by any deprecations and/or removals of features, APIs, 
fields of API types, flags, etc.?**
  Even if applying deprecation policies, they may still surprise some users.

### Monitoring Requirements

_This section must be completed when targeting beta graduation to a release._

* **How can an operator determine if the feature is in use by workloads?**
  Ideally, this should be a metric. Operations against the Kubernetes API (e.g.,
  checking if there are objects with field X set) may be a last resort. Avoid
  logs or events for this purpose.

* **What are the SLIs (Service Level Indicators) an operator can use to determine 
the health of the service?**
  - [ ] Metrics
    - Metric name:
    - [Optional] Aggregation method:
    - Components exposing the metric:
  - [ ] Other (treat as last resort)
    - Details:

* **What are the reasonable SLOs (Service Level Objectives) for the above SLIs?**
  At a high level, this usually will be in the form of "high percentile of SLI
  per day <= X". It's impossible to provide comprehensive guidance, but at the very
  high level (needs more precise definitions) those may be things like:
  - per-day percentage of API calls finishing with 5XX errors <= 1%
  - 99% percentile over day of absolute value from (job creation time minus expected
    job creation time) for cron job <= 10%
  - 99,9% of /health requests per day finish with 200 code

* **Are there any missing metrics that would be useful to have to improve observability 
of this feature?**
  Describe the metrics themselves and the reasons why they weren't added (e.g., cost,
  implementation difficulties, etc.).

### Dependencies

_This section must be completed when targeting beta graduation to a release._

* **Does this feature depend on any specific services running in the cluster?**
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


### Scalability

_For alpha, this section is encouraged: reviewers should consider these questions
and attempt to answer them._

_For beta, this section is required: reviewers must answer these questions._

_For GA, this section is required: approvers should be able to confirm the
previous answers based on experience in the field._

* **Will enabling / using this feature result in any new API calls?**

  - PATCH Pods, to remove finalizers.
    - estimated throughput: one per Pod created by the Job controller, when Pod
      finishes or is removed.
    - originating component: kube-controller-manager
  - PUT Job status, to keep track of uncounted Pods.
    - estimated throughput: at least one per Job sync. The job controller
      throttles additional calls at 1 per a few seconds (precise throughput TBD
      from experiments).
    - originating component: kube-controller-manager.

* **Will enabling / using this feature result in introducing new API types?**

  No.

* **Will enabling / using this feature result in any new calls to the cloud 
provider?**

  No.

* **Will enabling / using this feature result in increasing size or count of 
the existing API objects?**

  - Pod
    - Estimated increase: new finalizer of 33 bytes.
  - Job status
    - Estimated increase: new array temporarily containing terminated Pod UIDs.
      The job controller caps the size of the array to less than 20kb.

* **Will enabling / using this feature result in increasing time taken by any 
operations covered by [existing SLIs/SLOs]?**

  No existing SLIs/SLOs for Jobs.

* **Will enabling / using this feature result in non-negligible increase of 
resource usage (CPU, RAM, disk, IO, ...) in any components?**

  Additional memory to hold terminated Pods and the status of removing their
  finalizers.

### Troubleshooting

The Troubleshooting section currently serves the `Playbook` role. We may consider
splitting it into a dedicated `Playbook` document (potentially with some monitoring
details). For now, we leave it here.

_This section must be completed when targeting beta graduation to a release._

* **How does this feature react if the API server and/or etcd is unavailable?**

* **What are other known failure modes?**
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

* **What steps should be taken if SLOs are not being met to determine the problem?**

[supported limits]: https://git.k8s.io/community//sig-scalability/configs-and-limits/thresholds.md
[existing SLIs/SLOs]: https://git.k8s.io/community/sig-scalability/slos/slos.md#kubernetes-slisslos

## Implementation History

<!--
Major milestones in the lifecycle of a KEP should be tracked in this section.
Major milestones might include:
- the `Summary` and `Motivation` sections being merged, signaling SIG acceptance
- the `Proposal` section being merged, signaling agreement on a proposed design
- the date implementation started
- the first Kubernetes release where an initial version of the KEP was available
- the version of Kubernetes where the KEP graduated to general availability
- when the KEP was retired or superseded
-->

## Drawbacks

- Extra API calls and temporarily bigger Job status. However, without them
  it's impossible to ever scale the Job controller to deal with greater amount
  of Jobs or Jobs with greater amount of Pods.

## Alternatives

- Keep a list of created Pod UIDs, clearing them when they have been accounted.
  This has the benefit of requiring less Job status updates. On the other hand,
  the size of the updates is unbounded.
