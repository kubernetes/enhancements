<!--
**Note:** When your KEP is complete, all of these comment blocks should be removed.

To get started with this template:

- [x] **Pick a hosting SIG.**
  Make sure that the problem space is something the SIG is interested in taking
  up. KEPs should not be checked in without a sponsoring SIG.
- [x] **Create an issue in kubernetes/enhancements**
  When filing an enhancement tracking issue, please make sure to complete all
  fields in that template. One of the fields asks for a link to the KEP. You
  can leave that blank until this KEP is filed, and then go back to the
  enhancement and add the link.
- [x] **Make a copy of this template directory.**
  Copy this template into the owning SIG's directory and name it
  `NNNN-short-descriptive-title`, where `NNNN` is the issue number (with no
  leading-zero padding) assigned to your enhancement above.
- [x] **Fill out as much of the kep.yaml file as you can.**
  At minimum, you should fill in the "Title", "Authors", "Owning-sig",
  "Status", and date-related fields.
- [x] **Fill out this file as best you can.**
  At minimum, you should fill in the "Summary" and "Motivation" sections.
  These should be easy if you've preflighted the idea of the KEP with the
  appropriate SIG(s).
- [x] **Create a PR for this KEP.**
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
# KEP-5440: Allow updating pod template resources (CPU, memory, GPU, extended resources) of suspended jobs


<!--
This is the title of your KEP. Keep it short, simple, and descriptive. A good
title can help communicate what the KEP is and should be considered as part of
any review.
-->

<!--
A table of contents is helpful for quickly jumping to sections of a KEP and for
highlighting any additional information provided beyond the standard KEP
template.

Ensure the TOC is wrapped with
  <code>&lt;!-- toc --&rt;&lt;!-- /toc --&rt;</code>
tags, and then generate with `hack/update-toc.sh`.
-->

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
    - [Story 3](#story-3)
  - [Risks and Mitigations](#risks-and-mitigations)
- [Design Details](#design-details)
  - [DRA Support](#dra-support)
  - [Resuming on running workloads](#resuming-on-running-workloads)
  - [Related changes](#related-changes)
  - [Test Plan](#test-plan)
    - [Unit tests](#unit-tests)
    - [Integration tests](#integration-tests)
    - [e2e tests](#e2e-tests)
  - [Graduation Criteria](#graduation-criteria)
    - [Alpha](#alpha)
    - [Beta](#beta)
    - [GA](#ga)
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
  - [Delete and Recreate Jobs](#delete-and-recreate-jobs)
- [Infrastructure Needed (Optional)](#infrastructure-needed-optional)
<!-- /toc -->

## Release Signoff Checklist

<!--
**ACTION REQUIRED:** In order to merge code into a release, there must be an
issue in [kubernetes/enhancements] referencing this KEP and targeting a release
milestone **before the [Enhancement Freeze](https://git.k8s.io/sig-release/releases)
of the targeted release**.

For enhancements that make changes to code or processes/procedures in core
Kubernetes—i.e., [kubernetes/kubernetes], we require the following Release
Signoff checklist to be completed.

Check these off as they are completed for the Release Team to track. These
checklist items _must_ be updated for the enhancement to be released.
-->

Items marked with (R) are required *prior to targeting to a milestone / release*.

- [x] (R) Enhancement issue in release milestone, which links to KEP dir in [kubernetes/enhancements] (not the initial KEP PR)
- [x] (R) KEP approvers have approved the KEP status as `implementable`
- [x] (R) Design details are appropriately documented
- [x] (R) Test plan is in place, giving consideration to SIG Architecture and SIG Testing input (including test refactors)
  - [ ] e2e Tests for all Beta API Operations (endpoints)
  - [ ] (R) Ensure GA e2e tests for meet requirements for [Conformance Tests](https://github.com/kubernetes/community/blob/master/contributors/devel/sig-architecture/conformance-tests.md)
  - [ ] (R) Minimum Two Week Window for GA e2e tests to prove flake free
- [ ] (R) Graduation criteria is in place
  - [ ] (R) [all GA Endpoints](https://github.com/kubernetes/community/pull/1806) must be hit by [Conformance Tests](https://github.com/kubernetes/community/blob/master/contributors/devel/sig-architecture/conformance-tests.md)
- [ ] (R) Production readiness review completed
- [ ] (R) Production readiness review approved
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

In [#2232](https://github.com/kubernetes/enhancements/issues/2232) we added a new flag
to allow suspending jobs to control when the Pods of a Job get created by controller-manager.
This was proposed as a primitive to allow a higher-level queue controller to implement
job queuing: the queue controller unsuspends the job when resources become available.

To complement the above capability, a secondary controller may also want to control the
resource requirements of a job based on current cluster capacity or resource availability.
For example, it may want to adjust CPU, memory, and GPU requests/limits based on available
node capacity, allocate specific extended resources like TPUs or FPGAs, optimize resource
allocation for better cluster utilization, or modify resource requirements based on queue
priority and cluster load.

This is a proposal to relax update validation on suspended jobs to allow mutating
resource specifications in the job's pod template, specifically CPU, memory, GPU,
and other extended resource requests and limits. This enables a higher-level
controller to optimize resource allocation before un-suspending a job based on
current cluster conditions and resource availability.

## Motivation

Most kubernetes batch workloads have dynamic resource requirements that may not be
known at job creation time. The optimal resource allocation for a job often depends
on current cluster conditions, available capacity, and queue priorities that change
over time. This is especially true for GPU and other specialized hardware resources
which are expensive and have limited availability.

We made the first step towards achieving better resource management by introducing the
`suspend` flag to the Job API, which allowed a queue controller to decide when a job
should start. However, once a job's resource requirements are set at creation time,
there's no way to optimize them based on actual cluster conditions when the job is
ready to run.

Adding the ability to mutate a job's resource requirements while it's suspended gives
a controller the ability to optimize resource allocation based on real-time
cluster conditions, improve overall cluster utilization, and ensure jobs are sized
appropriately for current capacity constraints.

### Goals

- Allow mutating CPU, memory, GPU, and extended resource requests and limits of a container within a PodTemplate of a suspended jobs.
- Enable queue controllers to optimize resource allocation based on cluster conditions.
- Improve cluster resource utilization through dynamic resource sizing, especially for expensive GPU and specialized hardware.

### Non-Goals

- Implement a queue controller.
- Allow mutating resource specifications of jobs that are currently running. This could
  disrupt running workloads and complicate resource management.
- Allow mutating resource specifications of pods directly.
- Allow mutating other job specifications beyond container resource requirements.
- Support in-place pod resource updates (this is covered by separate KEPs).
- Allow mutating of Pod Resources.
- Allow mutating of ResourceClaims.

## Proposal

The proposal is to relax update validation for container resource specifications
(CPU, memory, GPU, and extended resource requests and limits) in the pod template of suspended jobs.

This change has minimal impact on the job-controller, as the job controller will
use the updated resource specifications when creating new pods for the job.

### User Stories (Optional)

#### Story 1

I want to build a controller that implements job queueing with dynamic resource optimization.
Users create v1.Job objects, and to control when the job can run, I have a webhook that
forces the jobs to be created in a suspended state. The controller analyzes current
cluster capacity and adjusts job resource requirements to optimize cluster utilization
before unsuspending them.

At job creation time, users may specify conservative resource estimates or may not know
the optimal resource allocation for current cluster conditions. The queue controller can
analyze available capacity, other queued jobs, and cluster utilization patterns to
determine optimal CPU, memory, and GPU allocations. For example, it might adjust the number of GPUs based on current
availability. By updating the job's resource requirements before unsuspending it, the
controller ensures efficient resource utilization and better cluster throughput.

#### Story 2

As an user, I want to be able to submit a job to a queueing solution. If my job cannot be admitted due to quota limitations, it should be possible to change my request so that the workload will be admitted.

#### Story 3

As a cluster administrator, I want to monitor existing workloads and see how many resources they are actually using. 
If a workload is oversubscribed and their actual utilization is lower, I want to checkpoint the workload via external APIs and then suspend the workload.
This will terminate the pods.
I would then lower the request requirements to match the actual utilization and resume my job.

### Risks and Mitigations

New API calls from queue controllers to update resource specifications.
The mitigation is for such controllers to make a single API call for both updating resources and
unsuspending the job.
  
Potential for resource specification changes to make a job unschedulable if the
updated requirements exceed available cluster capacity.
Queue controllers should validate resource availability before making changes.

Potential "leaks" for terminating pods can occur on suspended Jobs. The pods can be left in terminating and if `podReplacementPolicy: TerminatingOrFailed` (default option) is set, then the pods will be created and there is potential overlap of terminating pods and created pods.
Some frameworks, like TensorFlow, may not like this. The recommendation would be to set `podReplacementPolicy: Failed` so one can wait for pods to fully terminated before recreation.

Resources have always been immutable on Jobs so there is a risk that external controllers will read the Job controller spec and a separate controller could suspend and resize.
If the external controller acts on the stale information there could be a mismatch of expected versus reality.
Right now, the best approach would be to make sure controllers look at suspended state and if that state changes they should recheck the resources.

## Design Details

The pod template validation logic in the API server needs to be updated to relax the validation
of the Job's Template field. Currently the template is immutable, but we need to make
container resource specifications (CPU, memory, GPU, and extended resources requests and limits) mutable for suspended jobs.

The condition we will check to verify that the job is suspended is `Job.Spec.Suspend=true`.

We will allow updates to the following fields in container specifications within the pod template:
- `resources.requests`
- `resources.limits`

### DRA Support

DRA does not allow changing ResourceClaimTemplates once they are created.
At the moment, relaxing mutability constraints of ResourceClaimTemplates or ResourceClaims is not in scope.
To add support for this feature with DRA, the recommendation is to recreate ResourceClaimTemplates that match the
desired resources.

One does not have to modify claims in the PodTemplate so one can still assume claims are immutable also.

### Resuming on running workloads

When mutable pod scheduling requirements KEP was implemented, Jobs can only have their scheduling contraints changed on Job that is created in a suspended state.

If a workload is resumed, then the workload is assumed to be immutable from then on. So users are not able to change scheduling constraints or update resources after a Job starts.

In this work, we want to relax this solution to enable #story-3.
Users would be able to suspend a running workload, and change the resources on the suspended job.
It is important to note that when a running Job is suspended, any of its active Pods will be terminated.
This is a critical detail for any user or controller implementing this workflow.

For that reason we only allow mutability of the PodTemplate when all Pods are already marked for deletion,
ie. the Job has the "Suspended" condition and the "status.Active" equals 0.

### Related changes

As part of this KEP we also modify the condition for the mutability of the suspended Jobs which check that
`Job.Status.StartTime=nil`. While this check has the similar intention of making sure that there are no
Pods running with the old template, it is not ideal as it needs to be workaround by Kueue [here](https://github.com/kubernetes-sigs/kueue/blob/a5ce091a74e6e46e91a0c49e8a5942e64154d90b/pkg/controller/jobs/job/job_controller.go#L185-L192).

Finally, the changes above allow to also clear that "status.startTime" when suspending a Job, avoiding the
need to clean the field explicitly in the Kueue project.

### Test Plan

The following unit and integrations tests will be added.

- Container resource specifications are not mutable for active (non-suspended) jobs.
- Container resource specifications (CPU, memory, GPU, extended resources) are mutable only for suspended jobs.
- Job controller observes the resource updates and creates pods with the new resource specifications.
- Resource validation still applies (e.g., limits >= requests) for all resource types including extended resources.
- A job that is suspended once it went running is still able to change resources while it is suspended.

#### Unit tests

- `k8s.io/kubernetes/pkg/registry/batch/job/`: `9/30/2025` - `93.6%`
- `k8s.io/kubernetes/pkg/apis/batch/job/`: `9/30/2025` - `86.3%`

#### Integration tests

We will add the following test scenarios to kubernetes/test/integration/jobs.

- When a job is suspended with feature gate enabled, resources are able to be mutated.
- When a job is not suspended and feature gate enabled, resources should not be mutated.
- When feature gate is disabled and a job is suspended, mutations are not allowed.
- When a running job is suspended, mutations will be allowed.

An integration test will be added to verify the behavior of PodReplacementPolicy with `Failed` and this feature.
In this case, a running job will be suspended and the pods will go to a terminating state. If `PodReplacementPolicy` is set to `False`, a user should be able to change the resources on the Job but a pod will only be created once the terminating pods are removed. The new pods should have the resized resource.

#### e2e tests

Integration tests offer enough coverage.

### Graduation Criteria

#### Alpha

- Implement feature.
- Implement all the test cases.

#### Beta

- Enable feature by default
- Verify behavior in external controllers like Kueue or JobSet

#### GA

- Fix any potentially reported bugs

### Upgrade / Downgrade Strategy

No changes required to existing cluster to use this feature.

### Version Skew Strategy

N/A. This feature doesn't impact nodes.

## Production Readiness Review Questionnaire

### Feature Enablement and Rollback

###### How can this feature be enabled / disabled in a live cluster?

- [X] Feature gate (also fill in values in `kep.yaml`)
  - Feature gate name: MutableJobPodResourcesForSuspendedJobs
  - Components depending on the feature gate: kube-apiserver
- [ ] Other
  - Describe the mechanism:
  - Will enabling / disabling the feature require downtime of the control
    plane?
  - Will enabling / disabling the feature require downtime or reprovisioning
    of a node?

###### Does enabling the feature change any default behavior?

Yes, it relaxes validation of updates to jobs while they are suspended. Specifically, it will allow
mutating the container resource specifications (CPU, memory, GPU, and extended resource
requests and limits) in the pod template of suspended jobs.

###### Can the feature be disabled once it has been enabled (i.e. can we roll back the enablement)?

Yes. If disabled, kube-apiserver will start rejecting updates to container resource
specifications in job pod templates.

###### What happens if we reenable the feature if it was previously rolled back?

kube-apiserver will accept container resource specification updates for suspended jobs.

###### Are there any tests for feature enablement/disablement?

Yes. We have unit tests and integration tests verifying behavior with feature gate on and off.

See [integration-tests](#integration-tests) for more details.

### Rollout, Upgrade and Rollback Planning

###### How can a rollout or rollback fail? Can it impact already running workloads?

The change is opt-in and only affects suspended jobs, so it doesn't impact already
running workloads. However, problems with the updated validation logic may cause
crashes in the apiserver.

###### What specific metrics should inform a rollback?

If the SLOs for apiserver `apiserver_request_sli_duration_seconds` and `apiserver_request_duration_seconds` are performing poorly,
one can rollback this feature.

Another metric is `apiserver_request_total[resource=job, group=batch, verb=UPDATE, code=400].
This could also be used to monitor the rollout of this feature.

###### Were upgrade and rollback tested? Was the upgrade->downgrade->upgrade path tested?

Will be done after beta. In 1.36, we will perform the following test:

- create a kind cluster with feature gate off
  - verify suspend and patching of resources is forbidden

- create a kind cluster with feature gate on
  - verify suspend and patching of resources is allowed.

###### Is the rollout accompanied by any deprecations and/or removals of features, APIs, fields of API types, flags, etc.?

No.

### Monitoring Requirements

###### How can an operator determine if the feature is in use by workloads?

N/A. This is not a feature that workloads use directly.

###### How can someone using this feature know that it is working for their instance?


- [ ] Events
  - Event Reason:
- [ ] API .status
  - Condition name:
  - Other field:
- [X] Other (treat as last resort)
  - Details: Create a suspended job then update the container resource specifications (CPU/memory/GPU/extended resource requests/limits) of the pod template.

###### What are the reasonable SLOs (Service Level Objectives) for the enhancement?

N/A

###### What are the SLIs (Service Level Indicators) an operator can use to determine the health of the service?

<!--
Pick one more of these and delete the rest.
-->

- [x] Metrics
  - Metric name: apiserver_request_total[resource=job, group=batch, verb=UPDATE, code=400]
  - [Optional] Aggregation method:
  - Components exposing the metric: kube-apiserver
- [ ] Other (treat as last resort)
  - Details:

###### Are there any missing metrics that would be useful to have to improve observability of this feature?

No, there are no missing metrics. A metric is not necessary for this feature as users can patch workloads to resize.

This would be a one time request and monitoring provides little value.

### Dependencies

###### Does this feature depend on any specific services running in the cluster?

No.

### Scalability

###### Will enabling / using this feature result in any new API calls?

The feature itself doesn't generate API calls. But it will allow the
apiserver to accept update requests to mutate container resource specifications
(CPU, memory, GPU, and extended resources) in job pod templates, which will
encourage implementing controllers that do this.

###### Will enabling / using this feature result in introducing new API types?

No.

###### Will enabling / using this feature result in any new calls to the cloud provider?

No.

###### Will enabling / using this feature result in increasing size or count of the existing API objects?

No.

###### Will enabling / using this feature result in increasing time taken by any operations covered by existing SLIs/SLOs?

No.

###### Will enabling / using this feature result in non-negligible increase of resource usage (CPU, RAM, disk, IO, ...) in any components?

No.

### Troubleshooting

For general troubleshooting of API server issues, see [kubernetes.io/docs/tasks/debug/](https://kubernetes.io/docs/tasks/debug/).

###### How does this feature react if the API server and/or etcd is unavailable?

Update requests will be rejected.

###### What are other known failure modes?

In a multi-master setup, when the cluster has skewed apiservers, some update requests
may get accepted and some may get rejected.

###### What steps should be taken if SLOs are not being met to determine the problem?

N/A.

## Implementation History

- July 3rd: draft of KEP

## Drawbacks

This allows for more mutability of Jobs, particularly around resource specifications which could impact resource planning and scheduling behavior.

## Alternatives

### Delete and Recreate Jobs

One option is to keep this immutability and any modification of a Job should require a delete and create.

If there is a higher level actor controlling this Job with its own Owner References, then deletion is required on all resources that this job originally referenced.
This is a common use case for JobSet which manages multiple jobs and services.
Recreation would require deleting all existing jobs on an update if JobSet wanted to add support on updating JobTemplates while suspended.

In a multicluster scenario, deletion and recreation may require dispatching among different clusters.
Think of a scenario where you have 1 hub cluster and many worker clusters. If uses the hub to dispatch to a worker cluster, then this would require one to delete
the Job and propagate that deletion to the clusters. A patch is only a single operation so this would be faster.

## Infrastructure Needed (Optional)

NA
