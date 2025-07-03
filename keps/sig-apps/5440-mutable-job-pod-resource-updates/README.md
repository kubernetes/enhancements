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
  - [Risks and Mitigations](#risks-and-mitigations)
- [Design Details](#design-details)
  - [Test Plan](#test-plan)
    - [Unit tests](#unit-tests)
    - [Integration tests](#integration-tests)
    - [e2e tests](#e2e-tests)
  - [Graduation Criteria](#graduation-criteria)
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

- [] (R) Enhancement issue in release milestone, which links to KEP dir in [kubernetes/enhancements] (not the initial KEP PR)
- [] (R) KEP approvers have approved the KEP status as `implementable`
- [] (R) Design details are appropriately documented
- [] (R) Test plan is in place, giving consideration to SIG Architecture and SIG Testing input (including test refactors)
  - [ ] e2e Tests for all Beta API Operations (endpoints)
  - [ ] (R) Ensure GA e2e tests for meet requirements for [Conformance Tests](https://github.com/kubernetes/community/blob/master/contributors/devel/sig-architecture/conformance-tests.md)
  - [ ] (R) Minimum Two Week Window for GA e2e tests to prove flake free
- [] (R) Graduation criteria is in place
  - [ ] (R) [all GA Endpoints](https://github.com/kubernetes/community/pull/1806) must be hit by [Conformance Tests](https://github.com/kubernetes/community/blob/master/contributors/devel/sig-architecture/conformance-tests.md)
- [] (R) Production readiness review completed
- [] (R) Production readiness review approved
- [] "Implementation History" section is up-to-date for milestone
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

To complement the above capability, a queue controller may also want to control the
resource requirements of a job based on current cluster capacity or resource availability.
For example, it may want to adjust CPU, memory, and GPU requests/limits based on available
node capacity, allocate specific extended resources like TPUs or FPGAs, optimize resource
allocation for better cluster utilization, or modify resource requirements based on queue
priority and cluster load.

This is a proposal to relax update validation on suspended jobs to allow mutating
resource specifications in the job's pod template, specifically CPU, memory, GPU,
and other extended resource requests and limits. This enables a higher-level queue
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
a queue controller the ability to optimize resource allocation based on real-time
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

### Risks and Mitigations

- New API calls from queue controllers to update resource specifications. The mitigation
  is for such controllers to make a single API call for both updating resources and
  unsuspending the job.
  
- Potential for resource specification changes to make a job unschedulable if the
  updated requirements exceed available cluster capacity. Queue controllers should
  validate resource availability before making changes.

- A race condition could theoretically happen if a job is unsuspended and then quickly
  suspended again before resource updates, though this is not a typical use case pattern.

## Design Details

The pod template validation logic in the API server needs to be updated to relax the validation
of the Job's Template field. Currently the template is immutable, but we need to make
container resource specifications (CPU, memory, GPU, and extended resources requests and limits) mutable for suspended jobs.

The condition we will check to verify that the job is suspended is `Job.Spec.Suspend=true`.

We will allow updates to the following fields in container specifications within the pod template:
- `resources.requests.cpu`
- `resources.requests.memory`
- `resources.requests.*` (for extended resources like `nvidia.com/gpu`, `amd.com/gpu`, `tpu-v4` etc.)
- `resources.limits.cpu`
- `resources.limits.memory`
- `resources.limits.*` (for extended resources like `nvidia.com/gpu`, `amd.com/gpu`, `tpu-v4` etc.)

### Test Plan

- Unit and integration tests verifying that:
  - Container resource specifications are not mutable for active (non-suspended) jobs.
  - Container resource specifications (CPU, memory, GPU, extended resources) are mutable only for suspended jobs.
  - Job controller observes the resource updates and creates pods with the new resource specifications.
  - Resource validation still applies (e.g., limits >= requests) for all resource types including extended resources.

#### Unit tests

- `k8s.io/kubernetes/pkg/registry/batch/job/`: `1/30/2023` - `76.8%`

#### Integration tests

We will add the following test scenarios to kubernetes/test/integration/jobs.

- When a job is suspended with feature gate enabled, resources are able to be mutated.
- When a job is not suspended and feature gate enabled, resources should not be mutated.
- When feature date is disabled and suspended, mutations are not allowed.

#### e2e tests

Integration tests offer enough coverage.

### Graduation Criteria

We will release the feature directly in Beta state. Because the feature is opt-in and doesn't add
a new field, there is no benefit in having an alpha release.

#### Beta

- Feature implemented behind a feature flag
- Unit and integration tests passing

#### GA

- Fix any potentially reported bugs

<!--
**Note:** *Not required until targeted at a release.*

Define graduation milestones.

These may be defined in terms of API maturity, or as something else. The KEP
should keep this high-level with a focus on what signals will be looked at to
determine graduation.

Consider the following in developing the graduation criteria for this enhancement:
- [Maturity levels (`alpha`, `beta`, `stable`)][maturity-levels]
- [Deprecation policy][deprecation-policy]

Clearly define what graduation means by either linking to the [API doc
definition](https://kubernetes.io/docs/concepts/overview/kubernetes-api/#api-versioning)
or by redefining what graduation means.

In general we try to use the same stages (alpha, beta, GA), regardless of how the
functionality is accessed.

[maturity-levels]: https://git.k8s.io/community/contributors/devel/sig-architecture/api_changes.md#alpha-beta-and-stable-versions
[deprecation-policy]: https://kubernetes.io/docs/reference/using-api/deprecation-policy/

Below are some examples to consider, in addition to the aforementioned [maturity levels][maturity-levels].

#### Alpha

- Feature implemented behind a feature flag
- Initial e2e tests completed and enabled

#### Beta

- Gather feedback from developers and surveys
- Complete features A, B, C
- Additional tests are in Testgrid and linked in KEP

#### GA

- N examples of real-world usage
- N installs
- More rigorous forms of testing—e.g., downgrade tests and scalability tests
- Allowing time for feedback

**Note:** Generally we also wait at least two releases between beta and
GA/stable, because there's no opportunity for user feedback, or even bug reports,
in back-to-back releases.

**For non-optional features moving to GA, the graduation criteria must include
[conformance tests].**

[conformance tests]: https://git.k8s.io/community/contributors/devel/sig-architecture/conformance-tests.md

#### Deprecation

- Announce deprecation and support policy of the existing flag
- Two versions passed since introducing the functionality that deprecates the flag (to address version skew)
- Address feedback on usage/changed behavior, provided on GitHub issues
- Deprecate the flag
-->

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

No. There are unit tests verifying behavior with feature gate on and off.

We have integrations test verifying the behavior for feature on and off.

See [integration-tests](#integration-tests) for more details.

### Rollout, Upgrade and Rollback Planning

###### How can a rollout or rollback fail? Can it impact already running workloads?

The change is opt-in and only affects suspended jobs, so it doesn't impact already
running workloads. However, problems with the updated validation logic may cause
crashes in the apiserver.

###### What specific metrics should inform a rollback?

Crashes in the apiserver because of potential problems with the updated validation logic.

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

N/A

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

###### How does this feature react if the API server and/or etcd is unavailable?

Update requests will be rejected.

###### What are other known failure modes?

In a multi-master setup, when the cluster has skewed apiservers, some update requests
may get accepted and some may get rejected.

###### What steps should be taken if SLOs are not being met to determine the problem?

N/A.

## Implementation History

- July 3th: draft of KEP

## Drawbacks

This allows for more mutability of Jobs, particularly around resource specifications which could impact resource planning and scheduling behavior.

## Alternatives

NA
## Infrastructure Needed (Optional)

NA