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
# KEP-3370: Native PodGroup API

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
  - [Notes/Constraints/Caveats (Optional)](#notesconstraintscaveats-optional)
  - [Risks and Mitigations](#risks-and-mitigations)
- [Design Details](#design-details)
  - [PodGroup API](#podgroup-api)
  - [User workflow](#user-workflow)
  - [Scheduler](#scheduler)
    - [QueueSort](#queuesort)
    - [PreFilter](#prefilter)
    - [Permit](#permit)
    - [UnReserve](#unreserve)
  - [Cluster Autoscaler](#cluster-autoscaler)
  - [Test Plan](#test-plan)
      - [Prerequisite testing updates](#prerequisite-testing-updates)
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

- [ ] (R) Enhancement issue in release milestone, which links to KEP dir in [kubernetes/enhancements] (not the initial KEP PR)
- [ ] (R) KEP approvers have approved the KEP status as `implementable`
- [ ] (R) Design details are appropriately documented
- [ ] (R) Test plan is in place, giving consideration to SIG Architecture and SIG Testing input (including test refactors)
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

<!--
This section is incredibly important for producing high-quality, user-focused
documentation such as release notes or a development roadmap. It should be
possible to collect this information before implementation begins, in order to
avoid requiring implementors to split their attention between writing release
notes and implementing the feature itself. KEP editors and SIG Docs
should help to ensure that the tone and content of the `Summary` section is
useful for a wide audience.

A good summary is probably at least a paragraph in length.

Both in this section and below, follow the guidelines of the [documentation
style guide]. In particular, wrap lines to a reasonable length, to make it
easier for reviewers to cite specific portions, and to minimize diff churn on
updates.

[documentation style guide]: https://github.com/kubernetes/community/blob/master/contributors/guide/style-guide.md
-->
Propose a native PodGroup API in Kubernetes to represent an enforced constraint to 
schedule a group of pods altogether.

## Motivation

<!--
This section is for explicitly listing the motivation, goals, and non-goals of
this KEP.  Describe why the change is important and the benefits to users. The
motivation section can optionally provide links to [experience reports] to
demonstrate the interest in a KEP within the wider Kubernetes community.

[experience reports]: https://github.com/golang/go/wiki/ExperienceReports
-->

Kubernetes provides a number of scheduling directives to abstract workloads’ 
placement requirements. Till now, they’re mostly resource requests or topological
inter-pod hard/soft restrictions, and executed on a single pod’s basis, in a
stateless manner. This works great for most workloads. However, if we look at the
workload level, it’s not good enough because sometimes the entire workload needs
to be scheduled altogether or do nothing.

Lacking this "all-or-nothing" semantics can cause a workload to be partially
scheduled. This, from an application’s perspective, leads to an undetermined SLO
since the workload cannot be guaranteed to be ready within an expected period.
Sometimes it can be worse: workloads may be partially scheduled simultaneously to
reserve their resources and then cause deadlocks.

This "all-or-nothing" semantics, also known as coscheduling or gang-scheduling, 
is actually a typical requirement for batch (run-to-completion) workloads. We’re
proposing a fundamental Kubernetes PodGroup API to achieve this. This new API,
which served as a scheduling primitive, can be used along with other scheduling 
building blocks, to accommodate more workloads to run in Kubernetes natively and
effortlessly. Moreover, this enforced API will help achieve efficient cluster
utilization and ease users to define their application SLOs.


### Goals

<!--
List the specific goals of the KEP. What is it trying to achieve? How will we
know that this has succeeded?
-->

- Define PodGroup API in Kubernetes (k/k)
- Enforce PodGroup as a scheduling primitive
- Manage the lifecycle of PodGroup
- Compatible with CA (Cluster Autoscaler)

### Non-Goals

<!--
What is out of scope for this KEP? Listing non-goals helps to focus discussion
and make progress.
-->

- Restrict the PodGroup implementation to a specific scheduler
- Support PodGroup based preemption for alpha implementation.

## Proposal

<!--
This is where we get down to the specifics of what the proposal actually is.
This should have enough detail that reviewers can understand exactly what
you're proposing, but should not include things like API designs or
implementation. What is the desired outcome and how do we measure success?.
The "Design Details" section below is for the real
nitty-gritty.
-->

### User Stories (Optional)

<!--
Detail the things that people will be able to do if this KEP is implemented.
Include as much detail as possible so that people can understand the "how" of
the system. The goal here is to make this feel real for users without getting
bogged down.
-->

#### Story 1
As a batch workload user (e.g., MPI/TensorFlow/Spark), I want all pods of a 
Job to be either started altogether; otherwise, don’t start any of them.

#### Story 2
As a service workload user, I want some related deployments to run in the same zone 
(with other scheduling directives like PodAffinity) at the same time (using PodGroup); 
otherwise, don’t start any of them.

### Notes/Constraints/Caveats (Optional)

<!--
What are the caveats to the proposal?
What are some important details that didn't come across above?
Go in to as much detail as necessary here.
This might be a good place to talk about core concepts and how they relate.
-->

### Risks and Mitigations

<!--
What are the risks of this proposal, and how do we mitigate? Think broadly.
For example, consider both security and how this will impact the larger
Kubernetes ecosystem.

How will security be reviewed, and by whom?

How will UX be reviewed, and by whom?

Consider including folks who also work outside the SIG or subproject.
-->

- Incompatibility with CA : The failure reason of some pods in a pod group is not that the
resources cannot be met, but that the podgroup cannot meet `all-or-nothing` of
gang-scheduling. But CA will provision machines for all pods. For this part of pods in pod 
group, we will add different failure message in condition.reason. Cluster Autoscaler check the
failure reasons in pod.status.condition to decide whether it's possible to resolve the failure 
by provisioning new machines. Details are described below.

- Short-time resource deadlock in the memory of kube-scheduler: base on the alpha version 
implementation of pod group scheduling, it may happen that two pod groups reserve some
resources at the same time, and hence neither of them can be scheduled successfully. This is different
from the kube-scheduler without group scheduling, where the reserved resources are released by
timeout. And with QueueSort and PodsToActivate mechanism, the chances would be mitigated significantly 
(see [data](https://github.com/kubernetes-sigs/scheduler-plugins/pull/299)).

## Design Details

### PodGroup API

<!--
This section should contain enough information that the specifics of your
change are understandable. This may include API specs (though not always
required) or even code snippets. If there's any ambiguity about HOW your
proposal will be implemented, this is the place to discuss them.
-->

Most batch jobs consist of multiple (usually two) roles of tasks. For example, a
typical Tensorflow job has some parameter servers and workers. Each role
requires a quorum to be ready to run. With that being considered, the following
API designs all employ a 2-tier specification (Subsets) inside PodGroup. 

The PodGroup API will be defined in group scheduling.k8s.io, like PriorityClass.

```go
// group scheduling.k8s.io
type PodGroup struct {
	Spec   PodGroupSpec
	Status PodGroupStatus
}

// PodGroupSpec is a description of a PodGroupSpec.
type PodGroupSpec struct {
	// Subsets consist of various kinds of pod sets.
	// A PodGroup is schedulable if all subsets can be scheduled.
	Subsets []Subset

	// ScheduleTimeoutSeconds defines the timeout threshold to abort
	// an in-progress PodGroup-level scheduling attempt.
	// Duration Time is calculated from the time the first Pod in this
	// PodGroup gets a resource. If the timeout is reached, the resources
	// occupied by the PodGroup will be released to avoid long-term resource waste.
	ScheduleTimeoutSeconds *int32
}

// Subset represents a collection of pods with the same role.
type Subset struct {
	// Name is used to distinguish pods with different roles in the
	// same pod group. Optional if there is only one subset in the 
	// pod group.
	Name string

	// MinMember specifies the minimum number of pods required for a 
	// subset to be operational.
	MinMember int32
}
```

We then need to associate a Pod with its target PodGroup. To prevent potential race conditions
between PodGroup create events and Pod add events, we choose to add a field called `PodGroup`
to the podSpec instead of `label selector` to declare which PodGroup and Subset the Pod
belongs to.

```go
// PodSpec is a description of a pod.
type PodSpec struct {
	...
	PodGroup PodGroupRef
	...
}

// PodGroupRef contains enough information to let you identify an owning
// pod group.
// - pod.spec.podGroup.name = foo
// - pod.spec.podGroup.subset = (bar/baz)
type PodGroupRef struct {
	// Name is the name of PodGroup.
	Name   string
	// Subset is the name of Subset.
	Subset string
}
```

Pods in the same PodGroup with different priorities might lead to unintended behavior, 
so need to ensure Pods in the same PodGroup with the same priority.

In this design, a PodGroupStatus is needed to record the latest status. This
enables users to quickly know what’s going on with this PodGroup; also some
integrators can rely on it to make pending PodGroup schedulable. For this, we 
need to add a new PodGroup Controller in controller-manager.

```go
// PodGroupStatus represents the current state of a pod group.
type PodGroupStatus struct {
	// Represents the latest available observations of 
	// a pod group's current state.
	Conditions []PodGroupCondition

	// Subsets is a list of SubsetStatus which represents 
	// the current state of a single subset.
	Subsets []SubsetStatus
}

// PodGroupCondition describes the state of a pod group 
// at a certain point.
type PodGroupCondition struct {
	// Type of podGroup condition.
	Type PodGroupConditionType

	// Status of the condition, one of True, False, Unknown.
	Status v1.ConditionStatus

	// The last time this condition was updated.
	LastUpdateTime metav1.Time

	// Last time the condition transitioned from one status to another.
	LastTransitionTime metav1.Time
	
	// The reason for the condition's last transition.
	Reason string
	
	// A human readable message indicating details about 
	// the transition.
	Message string
}

type PodGroupConditionType string

const (
	// PodGroupScheduled represents status of the scheduling 
	// process for this pod group.
	PodGroupScheduled PodGroupConditionType = "PodGroupScheduled"
)

// SubsetStatus represents the current state of a subset.
type SubsetStatus struct {
	Name string

	// Total number of non-terminated pods targeted by this subset. (
	// their labels match the selector)
	Total int32

	// ScheduledAvailable is the number of pods in the subset that can be scheduled
	// successfully, but the scheduling fails because minMember cannot be met.
	ScheduledAvailable int32

	// ScheduledUnavailable is the number of pods in the subset that can't be 
	// scheduled successfully.
	ScheduledUnavailable int32

	// Running is the number of pods targeted by this subset
	// with a Running phase.
	Running int32
}
```

 After the pod group is scheduled for the first time, the condition `PodGroupScheduled` is
 added by the `pod-group-controller` in kube-controller-manager. Then `pod-group-controller`
 will reconcile the status of pod group. The workload operator can watch the pod group status,
 for example, when a pod is deleted and the total running count is less than `MinMember`. The
 workload operator can determine if it needs to evict the whole group or just create a new pod.


### User workflow
1. The user creates a PodGroup object.
2. The user creates a workload (Deployment, Job, etc.) and associates .spec.podGroup to the previously created PodGroup name.
3. The user watched the status of PodGroup. Pods associated with this PodGroup are expected to be co-scheduled if resources are adequate.
4. The user waits for the PodGroup to be complete and then deletes it.

### Scheduler
#### QueueSort
In order to maximize the chance that the pods which belong to the same `PodGroup` to be scheduled 
consecutively, we need to implement a customized `QueueSort` plugin to sort the Pods properly.

Firstly, we will inherit the default in-tree PrioritySort plugin so as to honor .spec.priority to 
ensure high-priority Pods are always sorted ahead of low-priority ones.

Secondly, if two Pods hold the same priority, the sorting precedence is described as below:

- If they are both regular Pods (without pod.spec.podGroup specified), compare their 
`InitialAttemptTimestamp` field: the Pod with earlier `InitialAttemptTimestamp` is positioned 
ahead of the other.
  
- If one is regularPod and the other is pgPod (with pod.spec.podGroup), compare regularPod's
`InitialAttemptTimestamp` with the `creationtime` of pgPod's PodGroup: the Pod with earlier 
timestamp is positioned ahead of the other.
  
- If they are both pgPods:
  - Compare their pod group's `CreationTimestamp`: the Pod in pod group which has earlier timestamp is positioned ahead of the other.
  - If their `CreationTimestamp` is identical, order by their UID of PodGroup: a Pod with lexicographically greater UID is scheduled ahead of the other Pod. (The purpose is to tease different PodGroups with the same `CreationTimestamp` apart, while also keeping Pods belonging to the same PodGroup back-to-back)

#### PreFilter
In PreFilter, it does some preliminary checks in the following sequences:

- If the pod doesn't carry .spec.podGroup, returns Success immediately. Optionally, plumb a key-value pair in CycleState so as to be leveraged in Permit phase later.
- If the pod carries .spec.podGroup, verify if the PodGroup exists or not:
  - If not, return UnschedulableAndUnresolvable.
  - If yes, verify if the quorum (MinMember) has been met. If met, return Success; otherwise return UnschedulableAndUnresolvable.


#### Permit
In Permit, it firstly leverages the pre-plumbed CycleState variable to quick return if it's a Pod not associated with any PodGroup.

Next, it counts the number of "cohorted" Pods that belong to the same PodGroup. Under the hood, it reads the scheduler framework's NodeInfos to include internally assumed/reserved Pods. The evalution formula is:

```go
 Running + Assumed Pods + 1 >= MinMember(1 means the pod itself)
```

- If the evaluation result is false, return Wait: which will hold the pod in the internal waitingPodsMap, and timed out based on the PodGroup's timeout setting. Meanwhile, proactively move the "cohorted" Pods back to the head of activeQ, so they can be retried immediately. This is a critical optimization to avoid potential deadlocks among PodGroups.

- If the evaluation result is true, iterate over its "cohorted" Pods to unhold them (as mentioned above, they where holded in the internal waitingPodsMap), and return Success.

#### UnReserve
After a pod which belongs to a PodGroup times out in the permit phase. UnReserve Rejects the pods that belong to the same PodGroup to avoid long-term invalid reservation of resources. The rejected pod will have the different failure message in condition with other scheduled failed pods.

```go
  // These are reasons for a pod's transition to a condition.
  const (
    // PodReasonUnschedulable reason in PodScheduled PodCondition means that the scheduler
    // can't schedule the pod right now, for example due to insufficient resources in the cluster.
    PodReasonUnschedulable = "Unschedulable"

    // PodReasonCoschedulingNotMeetMinMember reason in PodScheduled PodCondition means 
    // that the pod can be scheduled successfully, but the scheduling fails because 
    // minMember cannot be met.
    PodReasonCoschedulingNotMeetMinMember = "CoschedulingNotMeetMinMember"
  )


  pod.status.conditions[*].reason = PodReasonCoschedulingNotMeetMinMember
```
### Cluster Autoscaler

Cluster Autoscaler check the failure reasons in pod.status.condition to decide whether it's possible to resolve the failure by provisioning new machines. If the reason is `CoschedulingNotMeetMinMember`, CA don't need to provision new machines.

### Test Plan

<!--
**Note:** *Not required until targeted at a release.*
The goal is to ensure that we don't accept enhancements with inadequate testing.

All code is expected to have adequate tests (eventually with coverage
expectations). Please adhere to the [Kubernetes testing guidelines][testing-guidelines]
when drafting this test plan.

[testing-guidelines]: https://git.k8s.io/community/contributors/devel/sig-testing/testing.md
-->

[x] I/we understand the owners of the involved components may require updates to
existing tests to make this code solid enough prior to committing the changes necessary
to implement this enhancement.

##### Prerequisite testing updates

<!--
Based on reviewers feedback describe what additional tests need to be added prior
implementing this enhancement to ensure the enhancements have also solid foundations.
-->
No.

##### Unit tests

<!--
In principle every added code should have complete unit test coverage, so providing
the exact set of tests will not bring additional value.
However, if complete unit test coverage is not possible, explain the reason of it
together with explanation why this is acceptable.
-->

<!--
Additionally, for Alpha try to enumerate the core package you will be touching
to implement this enhancement and provide the current unit coverage for those
in the form of:
- <package>: <date> - <current test coverage>
The data can be easily read from:
https://testgrid.k8s.io/sig-testing-canaries#ci-kubernetes-coverage-unit

This can inform certain test coverage improvements that we want to do before
extending the production code to implement this enhancement.
-->

- `k8s.io/kubernetes/pkg/scheduler/internal/cache/cache.go`: `06-17` - `76.1%`
- `k8s.io/kubernetes/pkg/scheduler/internal/cache/snapshot.go`: `06-17` - `35.1%`

##### Integration tests

<!--
This question should be filled when targeting a release.
For Alpha, describe what tests will be added to ensure proper quality of the enhancement.

For Beta and GA, add links to added tests together with links to k8s-triage for those tests:
https://storage.googleapis.com/k8s-triage/index.html
-->

- These cases will be added in the existed integration tests:
  - Feature gate enable/disable tests
  - gang-scheduling in kube-scheduler works as expected
  - pod-group-controller is kube-controller-manager works as expected.

- `k8s.io/kubernetes/test/integration/scheduler/filters/filters_test.go`: https://storage.googleapis.com/k8s-triage/index.html?test=TestPodTopologySpreadFilter
- `k8s.io/kubernetes/test/integration/scheduler/scoring/priorities_test.go`: https://storage.googleapis.com/k8s-triage/index.html?test=TestPodTopologySpreadScoring
- `k8s.io/kubernetes/test/integration/scheduler_perf/scheduler_perf_test.go`: https://storage.googleapis.com/k8s-triage/index.html?test=BenchmarkPerfScheduling

##### e2e tests

<!--
This question should be filled when targeting a release.
For Alpha, describe what tests will be added to ensure proper quality of the enhancement.

For Beta and GA, add links to added tests together with links to k8s-triage for those tests:
https://storage.googleapis.com/k8s-triage/index.html

We expect no non-infra related flakes in the last month as a GA graduation criteria.
-->

- These cases will be added in the existed e2e tests:
  - Feature gate enable/disable tests
  - gang-scheduling in kube-scheduler works as expected
  - pod-group-controller is kube-controller-manager works as expected.

- `k8s.io/kubernetes/test/e2e/scheduling/predicates.go`: https://storage.googleapis.com/k8s-triage/index.html?sig=scheduling
- `k8s.io/kubernetes/test/e2e/scheduling/priorities.go`: https://storage.googleapis.com/k8s-triage/index.html?sig=scheduling

### Graduation Criteria

<!--
**Note:** *Not required until targeted at a release.*

Define graduation milestones.

These may be defined in terms of API maturity, [feature gate] graduations, or as
something else. The KEP should keep this high-level with a focus on what
signals will be looked at to determine graduation.

Consider the following in developing the graduation criteria for this enhancement:
- [Maturity levels (`alpha`, `beta`, `stable`)][maturity-levels]
- [Feature gate][feature gate] lifecycle
- [Deprecation policy][deprecation-policy]

Clearly define what graduation means by either linking to the [API doc
definition](https://kubernetes.io/docs/concepts/overview/kubernetes-api/#api-versioning)
or by redefining what graduation means.

In general we try to use the same stages (alpha, beta, GA), regardless of how the
functionality is accessed.

[feature gate]: https://git.k8s.io/community/contributors/devel/sig-architecture/feature-gates.md
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

#### Alpha
- Feature implemented behind feature gate.
- Unit and integration tests passed as designed in [TestPlan](#test-plan).

#### Beta
- Feature is enabled by default.
- Update documents to reflect the changes.
- Add official benchmarking of pod group scheduling.

#### GA
- No negative feedback.
- Update documents to reflect the changes.

### Upgrade / Downgrade Strategy

<!--
If applicable, how will the component be upgraded and downgraded? Make sure
this is in the test plan.

Consider the following in developing an upgrade/downgrade strategy for this
enhancement:
- What changes (in invocations, configurations, API use, etc.) is an existing
  cluster required to make on upgrade, in order to maintain previous behavior?
- What changes (in invocations, configurations, API use, etc.) is an existing
  cluster required to make on upgrade, in order to make use of the enhancement?
-->

In the event of an upgrade, kube-controller-manager will start to reconcile and keep the status of PodGroup in date. kube-scheduler will start to listAndWatch the object of pod groups and support gang-scheduling.

In the event of a downgrade，kube-controller-manager will stop reconciling and the status of PodGroup will be out of date. kube-scheduler won't listAndWatch the object of pod groups and does't support gang-scheduling.

### Version Skew Strategy

<!--
If applicable, how will the component handle version skew with other
components? What are the guarantees? Make sure this is in the test plan.

Consider the following in developing a version skew strategy for this
enhancement:
- Does this enhancement involve coordinating behavior in the control plane and
  in the kubelet? How does an n-2 kubelet without this feature available behave
  when this feature is used?
- Will any other components on the node change? For example, changes to CSI,
  CRI or CNI may require updating that component before the kubelet.
-->

## Production Readiness Review Questionnaire

<!--

Production readiness reviews are intended to ensure that features merging into
Kubernetes are observable, scalable and supportable; can be safely operated in
production environments, and can be disabled or rolled back in the event they
cause increased failures in production. See more in the PRR KEP at
https://git.k8s.io/enhancements/keps/sig-architecture/1194-prod-readiness.

The production readiness review questionnaire must be completed and approved
for the KEP to move to `implementable` status and be included in the release.

In some cases, the questions below should also have answers in `kep.yaml`. This
is to enable automation to verify the presence of the review, and to reduce review
burden and latency.

The KEP must have a approver from the
[`prod-readiness-approvers`](http://git.k8s.io/enhancements/OWNERS_ALIASES)
team. Please reach out on the
[#prod-readiness](https://kubernetes.slack.com/archives/CPNHUMN74) channel if
you need any help or guidance.
-->

### Feature Enablement and Rollback

<!--
This section must be completed when targeting alpha to a release.
-->

###### How can this feature be enabled / disabled in a live cluster?

<!--
Pick one of these and delete the rest.

Documentation is available on [feature gate lifecycle] and expectations, as
well as the [existing list] of feature gates.

[feature gate lifecycle]: https://git.k8s.io/community/contributors/devel/sig-architecture/feature-gates.md
[existing list]: https://kubernetes.io/docs/reference/command-line-tools-reference/feature-gates/
-->

- [x] Feature gate (also fill in values in `kep.yaml`)
  - Feature gate name: `PodGroup`
  - Components depending on the feature gate: kube-apiserver, kube-controller-manager, kube-scheduler
- [ ] Other
  - Describe the mechanism:
  - Will enabling / disabling the feature require downtime of the control
    plane?
  - Will enabling / disabling the feature require downtime or reprovisioning
    of a node? (Do not assume `Dynamic Kubelet Config` feature is enabled).

###### Does enabling the feature change any default behavior?

<!--
Any change of default behavior may be surprising to users or break existing
automations, so be extremely careful here.
-->
No.

###### Can the feature be disabled once it has been enabled (i.e. can we roll back the enablement)?

<!--
Describe the consequences on existing workloads (e.g., if this is a runtime
feature, can it break the existing applications?).

Feature gates are typically disabled by setting the flag to `false` and
restarting the component. No other changes should be necessary to disable the
feature.

NOTE: Also set `disable-supported` to `true` or `false` in `kep.yaml`.
-->

The feature can be disabled in Alpha and Beta versions by restarting 
kube-apiserver, kube-controller-manager and kube-scheduler with feature-gate off.

###### What happens if we reenable the feature if it was previously rolled back?

- kube-controller-manager will start to reconcile the pod status in pod-group controller.
- kube-scheduler will support gang scheduling if you created related pod group for the exist pending pods.

###### Are there any tests for feature enablement/disablement?

<!--
The e2e framework does not currently support enabling or disabling feature
gates. However, unit tests in each component dealing with managing data, created
with and without the feature, are necessary. At the very least, think about
conversion tests if API types are being modified.

Additionally, for features that are introducing a new API field, unit tests that
are exercising the `switch` of feature gate itself (what happens if I disable a
feature gate after having objects written with the new field) are also critical.
You can take a look at one potential example of such test in:
https://github.com/kubernetes/kubernetes/pull/97058/files#diff-7826f7adbc1996a05ab52e3f5f02429e94b68ce6bce0dc534d1be636154fded3R246-R282
-->
No, unit and integration tests will be added.

### Rollout, Upgrade and Rollback Planning

<!--
This section must be completed when targeting beta to a release.
-->

###### How can a rollout or rollback fail? Can it impact already running workloads?

<!--
Try to be as paranoid as possible - e.g., what if some components will restart
mid-rollout?

Be sure to consider highly-available clusters, where, for example,
feature flags will be enabled on some API servers and not others during the
rollout. Similarly, consider large clusters and how enablement/disablement
will rollout across nodes.
-->

###### What specific metrics should inform a rollback?

<!--
What signals should users be paying attention to when the feature is young
that might indicate a serious problem?
-->

###### Were upgrade and rollback tested? Was the upgrade->downgrade->upgrade path tested?

<!--
Describe manual testing that was done and the outcomes.
Longer term, we may want to require automated upgrade/rollback tests, but we
are missing a bunch of machinery and tooling and can't do that now.
-->

###### Is the rollout accompanied by any deprecations and/or removals of features, APIs, fields of API types, flags, etc.?

<!--
Even if applying deprecation policies, they may still surprise some users.
-->

### Monitoring Requirements

<!--
This section must be completed when targeting beta to a release.

For GA, this section is required: approvers should be able to confirm the
previous answers based on experience in the field.
-->

###### How can an operator determine if the feature is in use by workloads?

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

- [ ] Events
  - Event Reason: 
- [ ] API .status
  - Condition name: 
  - Other field: 
- [ ] Other (treat as last resort)
  - Details:

###### What are the reasonable SLOs (Service Level Objectives) for the enhancement?

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

- [ ] Metrics
  - Metric name:
  - [Optional] Aggregation method:
  - Components exposing the metric:
- [ ] Other (treat as last resort)
  - Details:

###### Are there any missing metrics that would be useful to have to improve observability of this feature?

<!--
Describe the metrics themselves and the reasons why they weren't added (e.g., cost,
implementation difficulties, etc.).
-->

### Dependencies

<!--
This section must be completed when targeting beta to a release.
-->

###### Does this feature depend on any specific services running in the cluster?

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

<!--
For alpha, this section is encouraged: reviewers should consider these questions
and attempt to answer them.

For beta, this section is required: reviewers must answer these questions.

For GA, this section is required: approvers should be able to confirm the
previous answers based on experience in the field.
-->

###### Will enabling / using this feature result in any new API calls?

<!--
Describe them, providing:
  - API call type (e.g. PATCH pods)
  - estimated throughput
  - originating component(s) (e.g. Kubelet, Feature-X-controller)
Focusing mostly on:
  - components listing and/or watching resources they didn't before
  - API calls that may be triggered by changes of some Kubernetes resources
    (e.g. update of object X triggers new updates of object Y)
  - periodic API calls to reconcile state (e.g. periodic fetching state,
    heartbeats, leader election, etc.)
-->

###### Will enabling / using this feature result in introducing new API types?

<!--
Describe them, providing:
  - API type
  - Supported number of objects per cluster
  - Supported number of objects per namespace (for namespace-scoped objects)
-->

###### Will enabling / using this feature result in any new calls to the cloud provider?

<!--
Describe them, providing:
  - Which API(s):
  - Estimated increase:
-->

###### Will enabling / using this feature result in increasing size or count of the existing API objects?

<!--
Describe them, providing:
  - API type(s):
  - Estimated increase in size: (e.g., new annotation of size 32B)
  - Estimated amount of new objects: (e.g., new Object X for every existing Pod)
-->

###### Will enabling / using this feature result in increasing time taken by any operations covered by existing SLIs/SLOs?

<!--
Look at the [existing SLIs/SLOs].

Think about adding additional work or introducing new steps in between
(e.g. need to do X to start a container), etc. Please describe the details.

[existing SLIs/SLOs]: https://git.k8s.io/community/sig-scalability/slos/slos.md#kubernetes-slisslos
-->

###### Will enabling / using this feature result in non-negligible increase of resource usage (CPU, RAM, disk, IO, ...) in any components?

<!--
Things to keep in mind include: additional in-memory state, additional
non-trivial computations, excessive access to disks (including increased log
volume), significant amount of data sent and/or received over network, etc.
This through this both in small and large cases, again with respect to the
[supported limits].

[supported limits]: https://git.k8s.io/community//sig-scalability/configs-and-limits/thresholds.md
-->

### Troubleshooting

<!--
This section must be completed when targeting beta to a release.

For GA, this section is required: approvers should be able to confirm the
previous answers based on experience in the field.

The Troubleshooting section currently serves the `Playbook` role. We may consider
splitting it into a dedicated `Playbook` document (potentially with some monitoring
details). For now, we leave it here.
-->

###### How does this feature react if the API server and/or etcd is unavailable?

###### What are other known failure modes?

<!--
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
-->

###### What steps should be taken if SLOs are not being met to determine the problem?

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
- 2022-06-09: Initial KEP

## Drawbacks

<!--
Why should this KEP _not_ be implemented?
-->

## Alternatives

<!--
What other approaches did you consider, and why did you rule them out? These do
not need to be as detailed as the proposal, but should include enough
information to express the idea and why it was not acceptable.
-->

## Infrastructure Needed (Optional)

<!--
Use this section if you need things from the project/SIG. Examples include a
new subproject, repos requested, or GitHub details. Listing these here allows a
SIG to get the process for these resources started right away.
-->
