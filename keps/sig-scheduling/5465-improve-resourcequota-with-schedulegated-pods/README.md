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
# KEP-5465: Improve ResourceQuota Enforcement with Schedule-Gated Pods

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
  - [Admission Controller Change](#admission-controller-change)
  - [ResourceQuota Accounting](#resourcequota-accounting)
  - [Scheduler Behavior](#scheduler-behavior)
  - [User Stories (Optional)](#user-stories-optional)
    - [Story 1: Deferred Workload Admission in Research Pipelines](#story-1-deferred-workload-admission-in-research-pipelines)
    - [Story 2: Multi-tenant CI/CD System with Queueing Logic](#story-2-multi-tenant-cicd-system-with-queueing-logic)
    - [Story 3: High-Throughput Batch Scheduling with Kueue](#story-3-high-throughput-batch-scheduling-with-kueue)
  - [Notes/Constraints/Caveats (Optional)](#notesconstraintscaveats-optional)
  - [Risks and Mitigations](#risks-and-mitigations)
- [Design Details](#design-details)
  - [API Compatibility](#api-compatibility)
  - [Feature Gate](#feature-gate)
  - [Admission Controller Logic](#admission-controller-logic)
  - [Quota Controller Modifications](#quota-controller-modifications)
  - [Scheduler Adjustments](#scheduler-adjustments)
    - [Scheduler Behavior](#scheduler-behavior-1)
      - [PreEnqueue Context](#preenqueue-context)
      - [PostBind Context](#postbind-context)
  - [Schedule Behavior Summary](#schedule-behavior-summary)
  - [Test Plan](#test-plan)
    - [Prerequisite testing updates](#prerequisite-testing-updates)
    - [Unit tests](#unit-tests)
    - [Integration tests](#integration-tests)
    - [e2e tests](#e2e-tests)
    - [Soak Testing](#soak-testing)
  - [Graduation Criteria](#graduation-criteria)
    - [Alpha](#alpha)
    - [Beta](#beta)
    - [GA](#ga)
  - [Upgrade / Downgrade Strategy](#upgrade--downgrade-strategy)
  - [Version Skew Strategy](#version-skew-strategy)
    - [Specifically:](#specifically)
    - [Resulting issues:](#resulting-issues)
    - [Mitigation:](#mitigation)
- [Production Readiness Review Questionnaire](#production-readiness-review-questionnaire)
  - [Feature Enablement and Rollback](#feature-enablement-and-rollback)
    - [How can this feature be enabled / disabled in a live cluster?](#how-can-this-feature-be-enabled--disabled-in-a-live-cluster)
    - [Does enabling the feature change any default behavior?](#does-enabling-the-feature-change-any-default-behavior)
    - [Can the feature be disabled once it has been enabled (i.e. can we roll back the enablement)?](#can-the-feature-be-disabled-once-it-has-been-enabled-ie-can-we-roll-back-the-enablement)
      - [<strong>Consequences of disabling:</strong>](#consequences-of-disabling)
    - [What happens if we reenable the feature if it was previously rolled back?](#what-happens-if-we-reenable-the-feature-if-it-was-previously-rolled-back)
      - [Specifically:](#specifically-1)
      - [Existing pods:](#existing-pods)
      - [Important Note:](#important-note)
    - [Are there any tests for feature enablement/disablement?](#are-there-any-tests-for-feature-enablementdisablement)
  - [Rollout, Upgrade and Rollback Planning](#rollout-upgrade-and-rollback-planning)
    - [How can a rollout or rollback fail? Can it impact already running workloads?](#how-can-a-rollout-or-rollback-fail-can-it-impact-already-running-workloads)
      - [Possible failure modes:](#possible-failure-modes)
      - [Mitigation:](#mitigation-1)
    - [What specific metrics should inform a rollback?](#what-specific-metrics-should-inform-a-rollback)
      - [🔍 Additional Indicators](#-additional-indicators)
    - [How can someone using this feature know that it is working for their instance?](#how-can-someone-using-this-feature-know-that-it-is-working-for-their-instance)
    - [What are the reasonable SLOs (Service Level Objectives) for the enhancement?](#what-are-the-reasonable-slos-service-level-objectives-for-the-enhancement)
    - [What are the SLIs (Service Level Indicators) an operator can use to determine the health of the service?](#what-are-the-slis-service-level-indicators-an-operator-can-use-to-determine-the-health-of-the-service)
    - [Are there any missing metrics that would be useful to have to improve observability of this feature?](#are-there-any-missing-metrics-that-would-be-useful-to-have-to-improve-observability-of-this-feature)
  - [Dependencies](#dependencies)
    - [Does this feature depend on any specific services running in the cluster?](#does-this-feature-depend-on-any-specific-services-running-in-the-cluster)
      - [Details:](#details)
      - [Internal dependency considerations:](#internal-dependency-considerations)
      - [Summary:](#summary-1)
  - [Scalability](#scalability)
    - [Will enabling / using this feature result in any new API calls?](#will-enabling--using-this-feature-result-in-any-new-api-calls)
      - [Affected API Call Patterns:](#affected-api-call-patterns)
      - [🔍 Summary:](#-summary)
    - [Will enabling / using this feature result in increasing time taken by any operations covered by existing SLIs/SLOs?](#will-enabling--using-this-feature-result-in-increasing-time-taken-by-any-operations-covered-by-existing-slisslos)
      - [📊 Summary:](#-summary-1)
  - [Troubleshooting](#troubleshooting)
    - [How does this feature react if the API server and/or etcd is unavailable?](#how-does-this-feature-react-if-the-api-server-andor-etcd-is-unavailable)
      - [⚠️ 1. <strong>Inconsistent Quota Accounting (partial rollout)</strong>](#-1-inconsistent-quota-accounting-partial-rollout)
      - [⚠️ 3. <strong>Pods remain stuck in Pending with unclear reason</strong>](#-3-pods-remain-stuck-in-pending-with-unclear-reason)
      - [⚠️ 5. <strong>Pod misbehavior after rollback</strong>](#-5-pod-misbehavior-after-rollback)
- [Implementation History](#implementation-history)
- [Drawbacks](#drawbacks)
  - [Summary:](#summary-2)
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
  - [ ] (R) Ensure GA e2e tests meet requirements for [Conformance Tests](https://github.com/kubernetes/community/blob/master/contributors/devel/sig-architecture/conformance-tests.md) 
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

This KEP proposes changes to the Kubernetes `ResourceQuota` enforcement mechanism to improve alignment with established scheduling paradigms. Specifically, it introduces a more lenient admission strategy for pods requesting CPU and memory resources that would exceed the namespace-scoped `ResourceQuota`, while relying on the scheduler to determine actual feasibility. The proposal includes:

* Allowing admission of such overcommitting pods if they are blocked by a `PodSchedulingReadiness` gate.
* Excluding these gated pods from `ResourceQuota` usage calculation.
* Enhancing the scheduler to respect `ResourceQuota` limits and leave pods in the Pending phase if scheduling would cause quota overcommit.


## Motivation

<!--
This section is for explicitly listing the motivation, goals, and non-goals of
this KEP.  Describe why the change is important and the benefits to users. The
motivation section can optionally provide links to [experience reports] to
demonstrate the interest in a KEP within the wider Kubernetes community.

[experience reports]: https://github.com/golang/go/wiki/ExperienceReports
-->

Current `ResourceQuota` enforcement is strict: if a pod's resource request exceeds the remaining quota, it is rejected during admission. This is inconsistent with other Kubernetes paradigms, such as node resource availability, where unschedulable pods are allowed to remain in a Pending state.

In addition, there is a notable misalignment between quota usage accounting and pod readiness state. Pods that are gated by mechanisms like `PodSchedulingReadiness` cannot yet be scheduled or consume actual cluster resources, yet they are still counted against quota usage. This leads to inefficiencies and unnecessary admission rejections, particularly for systems managing asynchronous or staged workload execution.

The strict admission approach:

* Breaks compatibility with asynchronous or deferred scheduling patterns.
* Prevents advanced workload orchestrators (e.g., Kueue) from managing workloads that temporarily exceed quota.
* Impedes resource overcommit strategies even if final scheduling may be feasible after other pods complete.

This proposal introduces a more natural, flexible alternative.

### Goals

<!--
List the specific goals of the KEP. What is it trying to achieve? How will we
know that this has succeeded?
-->

* Enable pod admission despite temporary `ResourceQuota` overcommitment for compute resources.
* Prevent such pods from being scheduled unless `ResourceQuota` constraints are met.
* Preserve current behavior for `resource.count` types and for non-gated pods.

### Non-Goals

<!--
What is out of scope for this KEP? Listing non-goals helps to focus discussion
and make progress.
-->

* Changing enforcement behavior for `resource.count/*` types.
* Making quotas globally scoped or changing quota reconciliation mechanics.

## Proposal

<!--
This is where we get down to the specifics of what the proposal actually is.
This should have enough detail that reviewers can understand exactly what
you're proposing, but should not include things like API designs or
implementation. What is the desired outcome and how do we measure success?.
The "Design Details" section below is for the real
nitty-gritty.
-->

### Admission Controller Change

Modify the `ResourceQuota` admission controller to:

1. Check if a pod includes one or more `PodSchedulingReadiness` gates.
2. If present, exclude the pod's compute resource usage (e.g., `cpu`, `memory`) from the namespace quota accounting.
3. Permit the pod's admission even if its resource requests exceed the current quota usage.
4. Maintain current rejection behavior for pods that do not include readiness gates.

### ResourceQuota Accounting

**Update quota controller logic:**

* When reconciling usage, exclude pods that are blocked by one or more `PodSchedulingReadiness` gates.
* For pods gated post-scheduling, continue excluding their resource requests from quota usage until they are successfully scheduled.
* Resume accounting for their resource usage only once all scheduling gates are removed and the pod is scheduled.

### Scheduler Behavior

**Update [scheduler context](https://kubernetes.io/docs/concepts/scheduling-eviction/scheduling-framework/#interfaces) for previously schedule-gated pods:**

To support deferred ResourceQuota enforcement, we will introduce a new `resourcequota` scheduler plugin responsible for deferred quota validation and usage accounting. Key behavior is outlined below:

* The plugin will be **feature-gated** and only active when the `ResourceQuotaDeferredEnforcement` feature is enabled.

* It will only apply to **previously schedule-gated pods**, which are identified by the following `PodScheduled` status condition, or
  * `PodScheduled` condition type and `ResourceQuotaValidated` reason.
 
* Upon successful scheduling quota will be updated reflecting scheduled pod resources after deferred validation.


### User Stories (Optional)

<!--
Detail the things that people will be able to do if this KEP is implemented.
Include as much detail as possible so that people can understand the "how" of
the system. The goal here is to make this feel real for users without getting
bogged down.
-->

#### Story 1: Deferred Workload Admission in Research Pipelines

A machine learning research team submits jobs to a shared Kubernetes cluster where resource quotas are used to prevent overutilization. 
A job might require a significant amount of `GPU` or `CPU` resources, but is designed to wait for availability using `PodSchedulingReadiness` gates. 
With the current `ResourceQuota` enforcement, these jobs are rejected at admission time due to exceeding quota, even though the scheduler may have handled them in sequence.
With the proposed change, such jobs' pods are admitted, tracked appropriately, and only scheduled when feasible.

#### Story 2: Multi-tenant CI/CD System with Queueing Logic

CI/CD system orchestrates pipelines for many tenants using Kueue. 
These pipelines submit multiple workload stages that depend on dynamic scheduling windows. 
Some stages are configured with scheduling gates to delay execution until earlier stages complete or quota becomes available. 
Current strict quota checks prevent these staged pods from being admitted, breaking pipeline logic. 
With the proposed change, all stages can be admitted upfront and scheduling can proceed incrementally.

#### Story 3: High-Throughput Batch Scheduling with Kueue

A research institution runs thousands of short-lived batch jobs using Kueue across multiple teams, each with its own namespace and associated ResourceQuota. 
These jobs are submitted into queues managed by Kueue, and often gated to ensure they are only scheduled when sufficient capacity becomes available. 
Currently, if the initial pods of a Kueue-managed job exceed quota, the admission controller rejects them, breaking the workflow and requiring the job controller to recreate pods, causing unnecessary load and potential delays.

With the proposed change, pods from Kueue-managed jobs with scheduling gates are admitted regardless of quota pressure, allowing Kueue to manage scheduling according to availability while avoiding unnecessary admission rejections and resubmissions.

### Notes/Constraints/Caveats (Optional)

<!--
What are the caveats to the proposal?
What are some important details that didn't come across above?
Go in to as much detail as necessary here.
This might be a good place to talk about core concepts and how they relate.
-->

### Risks and Mitigations

- **Risk**: Misuse by users resulting in flooding cluster with Pending pods.
  - **Mitigation**:
    - `resource.count/*` quota enforcement is still in place and used to control the number of admitted pods.
    - Quota still applies to non-gated pods; scheduler prevents actual overload by withholding scheduling if the total usage would exceed quota limits.

## Design Details

<!--
This section should contain enough information that the specifics of your
change are understandable. This may include API specs (though not always
required) or even code snippets. If there's any ambiguity about HOW your
proposal will be implemented, this is the place to discuss them.
-->

### API Compatibility

This KEP does not introduce new API types or fields. Instead, it reuses the existing `PodSchedulingReadiness` mechanism and interprets its presence as an indication that the pod is not yet eligible for scheduling.

This KEP introduces new value for the existing field: pod status condition reason `ResourceQuotaValidated` 

### Feature Gate

The behavior will be guarded by a new feature gate:

```yaml
ResourceQuotaDeferredEnforcement: true
```

* When disabled: default `ResourceQuota` admission behavior is preserved.
* When enabled: pods with scheduling gates are admitted even if they overcommit compute quotas.

### Admission Controller Logic

* Add check for presence of `PodSchedulingReadiness` gates.
* If found, compute resource requests (e.g., CPU and memory) are not counted towards quota usage.
* Proceed with pod admission even if quota usage would be exceeded.
* Maintain normal rejection behavior for pods without scheduling gates or for resources of type `resource.count/*`.

### Quota Controller Modifications

* Modify the quota controller to exclude pods with scheduling gates from usage calculation.
* Recalculate quota usage dynamically once gates are removed.

### Scheduler Adjustments

* This adjustment applies **only** to previously schedule-gated pods.
* Before scheduling, simulate pod admission against the current namespace `ResourceQuota` usage.
* If scheduling the pod would exceed quota, do **not** proceed with scheduling, by updating `PodScheduled` condition with `ResourceQuotaValidated` reason.
* Allow the scheduler to retry later as other pods complete or quota becomes available.

#### Scheduler Behavior

**Update [scheduler context](https://kubernetes.io/docs/concepts/scheduling-eviction/scheduling-framework/#interfaces) for previously schedule-gated pods:**

To support deferred ResourceQuota enforcement, we will introduce a new `resourcequota` scheduler plugin responsible for deferred quota validation and usage accounting. Key behavior is outlined below:

* The plugin will be **feature-gated** and only active when the `ResourceQuotaDeferredEnforcement` feature is enabled.

* It will only apply to **previously schedule-gated pods**, which are identified by the following `PodScheduled` status condition:

  ```yaml
  status:
    conditions:
      - type: PodScheduled
        status: "False"
        reason: SchedulingGated
  ```

* If deferred quota validation fails, the plugin will:
  * update `PosdScheduled` condition with new Reason `ResourceQuotaValidated`
  * return a `PreEnqueue` result of `UnschedulableAndUnresolvable`.

* The scheduler will then mark the pod’s `PodScheduled` condition with:

  ```yaml
  status:
    conditions:
      - type: PodScheduled
        status: "False"
        reason: ResourceQuotaValidated
        message: '0/1 nodes are available: quota exceeded:
          p1, requested: cpu=1, used: cpu=2, limited: cpu=2. preemption: 0/1 nodes are
          available: 1 Preemption is not helpful for scheduling.'
  ```

* Since the plugin relies on the `SchedulingGated` condition to identify eligible pods, we must avoid losing track of pods that fail deferred validation. To that end:

  * A new `PodScheduled` condition reason will be introduced: `ResourceQuotaValidated`.
  * Pods failing deferred quota checks will retain `status: "False"` for `PodScheduled` but with the updated reason `ResourceQuotaValidated`.

* Upon successful scheduling, the pod’s `PodScheduled` condition is updated to reflect success:

  ```yaml
  - type: PodScheduled
    status: "True"
    lastTransitionTime: "2025-08-01T06:43:37Z"
  ```

##### PreEnqueue Context

The deferred quota validation will be implemented in the `PreEnqueeu` extension point of the scheduling cycle. The logic is as follows:

* **For `SchedulingGated` pods**:

  * Confirm the pod is schedule-gated:

    ```yaml
    status:
      conditions:
        - type: PodScheduled
          status: "False"
          reason: SchedulingGated
    ```
  * Validate that scheduling the pod does not exceed any `ResourceQuota` in its namespace.

    * **If quota is exceeded**:

      * Retain `status: "False"` for `PodScheduled`, but update the reason to `ResourceQuotaValidated`.
    * **If quota check passes**:

      * Continue with the scheduling flow (see PostBind section below).

* **For `ResourceQuotaValidated` pods**:

  * Detect by presence of:

    ```yaml
    status:
      conditions:
        - type: PodScheduled
          status: "False"
          reason: ResourceQuotaValidated
          message: '...'
    ```
  * Retry quota validation:

    * **If quota is still exceeded**:

      * Keep `status: "False"` and reason as `ResourceQuotaValidated`.
    * **If quota check passes**:

      * Continue with the scheduling flow (see PostBind section).

##### PostBind Context

In the `PostBind` extension point, we will update the namespace’s `ResourceQuota` usage to account for the resources requested by the now-scheduled pod. This ensures proper quota accounting is re-established after deferred enforcement.

### Schedule Behavior Summary

| Event                            | Pod Counted in Quota | Resource Counted in Quota? | Admitted?  | Eligible for Scheduling?                                                               |
|----------------------------------|----------------------|-------------------------| ---------- |----------------------------------------------------------------------------------------|
| Normal pod created               | ✅ Yes                | ✅ Yes                   | ✅ If fits | ✅ If fits quota and node                                                               |
| Gated pod created (new behavior) | ✅ Yes  | ❌ No                    | ✅ Always                   | ❌ - No, all scheduling gates must be removed first.                                    |
| Gate removed (not scheduled)     | ✅ Yes | ❌ No                    | Already in                 | ❌ - If fails deferred quota validation <br/> ✅ - If succeeds deferred quota validation |
| Gate removed (scheduled)     | ✅ Yes | ✅ Yes                       | Already in                 | Already scheduled                                                                      |


### Test Plan

<!--
**Note:** *Not required until targeted at a release.*
The goal is to ensure that we don't accept enhancements with inadequate testing.

All code is expected to have adequate tests (eventually with coverage
expectations). Please adhere to the [Kubernetes testing guidelines][testing-guidelines]
when drafting this test plan.

[testing-guidelines]: https://git.k8s.io/community/contributors/devel/sig-testing/testing.md
-->

[ ] I/we understand the owners of the involved components may require updates to
existing tests to make this code solid enough prior to committing the changes necessary
to implement this enhancement.

#### Prerequisite testing updates

<!--
Based on reviewers feedback describe what additional tests need to be added prior
implementing this enhancement to ensure the enhancements have also solid foundations.
-->

#### Unit tests

- Admission controller logic for gated vs non-gated pods.
- Quota controller usage computation logic with readiness gates.

#### Integration tests
 
- Create pods with readiness gates that exceed quota and verify admission.
- Verify quota usage updates after gate removal.
- Simulate full scheduling cycle for gated workloads.

#### e2e tests

- Submit a gated pod exceeding quota and verify:
  - It is admitted
  - It is not counted in quota
  - It stays in Pending until gate removed and quota becomes available
  - It schedules successfully once eligible

#### Soak Testing
- Run workloads via Kueue that depend on this feature to test stability in high-churn scenarios.

### Graduation Criteria

#### Alpha

* Feature gate introduced
* Basic implementation and tests

#### Beta

* Proven stability via soak testing
* Widespread usage in deferred scheduling systems (e.g., Kueue)

#### GA

* Enabled by default
* Documented and adopted across workloads

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

### Version Skew Strategy

<!--
If applicable, how will the component handle version skew with other
components? What are the guarantees? Make sure this is in the test plan.

Consider the following in developing a version skew strategy for this
enhancement:
- Does this enhancement involve coordinating behavior in the control plane and nodes?
- How does an n-3 kubelet or kube-proxy without this feature available behave when this feature is used?
- How does an n-1 kube-controller-manager or kube-scheduler without this feature available behave when this feature is used?
- Will any other components on the node change? For example, changes to CSI,
  CRI or CNI may require updating that component before the kubelet.
-->

An `n-1` (older version) `kube-controller-manager` or `kube-scheduler` **without support for `ResourceQuotaDeferredEnforcement`** will behave as if the feature is **disabled**, even if it is enabled on the API server.

#### Specifically:

* The **kube-scheduler** will not perform quota validation before binding previously schedule-gated pods. It will schedule them as long as they fit on the node, potentially **bypassing quota constraints**, resulting in **quota overcommit**.
* The **kube-controller-manager**, which runs the `ResourceQuota` controller, will **continue to include gated pods in quota usage accounting**, because it lacks logic to exclude `PodSchedulingReadiness`-gated pods. This may cause **inconsistent or inaccurate quota usage reports**.

#### Resulting issues:

* **Quota inconsistencies**: The API server may admit pods assuming their usage won’t count yet, while the older controller counts them anyway.
* **Possible scheduler misbehavior**: Pods that should be blocked due to quota limits may be scheduled if the scheduler doesn’t re-check quota before binding.
* **Unintended admission rejections or binding decisions** if different components interpret quota state differently.

#### Mitigation:

This feature **requires version alignment** between the API server, quota controller (kube-controller-manager), and scheduler for correct behavior. Partial upgrades (e.g., API server on `n`, others on `n-1`) are **not recommended** when the feature is enabled. Feature should be enabled only when all relevant components support it.

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

#### How can this feature be enabled / disabled in a live cluster?

<!--
Pick one of these and delete the rest.

Documentation is available on [feature gate lifecycle] and expectations, as
well as the [existing list] of feature gates.

[feature gate lifecycle]: https://git.k8s.io/community/contributors/devel/sig-architecture/feature-gates.md
[existing list]: https://kubernetes.io/docs/reference/command-line-tools-reference/feature-gates/
-->

- [ ] Feature gate (also fill in values in `kep.yaml`)
  - Feature gate name: ResourceQuotaDeferredEnforcement
  - Components depending on the feature gate: apiserver, controller-manager, scheduler
- [ ] Other
  - Describe the mechanism:
  - Will enabling / disabling the feature require downtime of the control
    plane? - No.
  - Will enabling / disabling the feature require downtime or reprovisioning
    of a node? - No.

#### Does enabling the feature change any default behavior?

<!--
Any change of default behavior may be surprising to users or break existing
automations, so be extremely careful here.
-->

No, enabling the feature does **not** change any default behavior unless the feature gate `ResourceQuotaDeferredEnforcement` is explicitly set to `true`.

By default, (with the feature gate disabled) the `ResourceQuota` admission controller continues to reject pods whose compute resource requests exceed quota, regardless of whether they include `PodSchedulingReadiness` gates.

When the feature gate is enabled, the behavior changes **only** for pods with one or more `PodSchedulingReadiness` gates:

* These gated pods are allowed to be admitted even if they exceed compute resource quotas.
* Their resource requests are temporarily excluded from quota usage until the gates are removed.

All other pods, including those without gates or those affecting `resource.count/*` quotas, continue to follow existing behavior.

In summary: **default behavior is preserved unless the feature gate is enabled.**

#### Can the feature be disabled once it has been enabled (i.e. can we roll back the enablement)?

<!--
Describe the consequences on existing workloads (e.g., if this is a runtime
feature, can it break the existing applications?).

Feature gates are typically disabled by setting the flag to `false` and
restarting the component. No other changes should be necessary to disable the
feature.

NOTE: Also set `disable-supported` to `true` or `false` in `kep.yaml`.
-->

Yes, the feature **can be disabled** once it has been enabled by setting the `ResourceQuotaDeferredEnforcement` feature gate to `false` and restarting the relevant control plane components (primarily the API server and scheduler).

##### **Consequences of disabling:**

* Pods that were previously admitted under the deferred quota enforcement rule (i.e., gated pods that exceeded compute quota) will remain in the cluster.
* New pods that exceed quota, even if they include `PodSchedulingReadiness` gates, will be rejected during admission.
* The `ResourceQuota` controller will resume counting all pods (gated or not) toward quota usage as before.
* The scheduler will no longer perform quota revalidation for previously gated pods before scheduling.
* **Side effect**: There is a potential for **overcommitted resource requests**, since previously admitted schedule-gated pods (not counted against quota at the time) may now bypass quota checks entirely if the scheduler does not revalidate, leading to aggregate quota usage exceeding the intended limits.

To avoid such inconsistencies, it is recommended to avoid disabling the feature while previously gated pods admitted under the relaxed rules are still present in the cluster.

#### What happens if we reenable the feature if it was previously rolled back?

If the `ResourceQuotaDeferredEnforcement` feature is reenabled after being previously rolled back, the system resumes the deferred quota enforcement behavior **immediately** for newly admitted pods.

##### Specifically:

* **Newly created pods** that include `PodSchedulingReadiness` gates will once again be admitted even if their compute resource requests exceed the namespace quota.
* These newly gated pods will **not** be counted against quota usage until their gates are removed.
* The scheduler will resume performing quota validation **before scheduling** such pods.

##### Existing pods:

* Pods admitted during the time the feature was disabled will have been subject to strict quota checks and quota usage accounting.
* Reenabling the feature does **not retroactively adjust** how those existing pods were treated.
* However, future behavior for all newly submitted gated pods will follow the deferred enforcement logic.

##### Important Note:

If there are existing schedule-gated pods from before the rollback (i.e., still in Pending with gates intact), reenabling the feature may change how quota usage is computed going forward, for example, such pods may now be **excluded** from usage until the gates are removed.

In short: reenabling the feature is safe and restores the relaxed admission + deferred quota accounting logic for gated pods without disrupting already-admitted workloads.


#### Are there any tests for feature enablement/disablement?

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

### Rollout, Upgrade and Rollback Planning

<!--
This section must be completed when targeting beta to a release.
-->

#### How can a rollout or rollback fail? Can it impact already running workloads?

<!--
Try to be as paranoid as possible - e.g., what if some components will restart
mid-rollout?

Be sure to consider highly-available clusters, where, for example,
feature flags will be enabled on some API servers and not others during the
rollout. Similarly, consider large clusters and how enablement/disablement
will rollout across nodes.
-->

A rollout or rollback of the `ResourceQuotaDeferredEnforcement` feature can fail or cause inconsistencies **if the cluster components are not upgraded or reverted uniformly**, or if **admitted workloads are not accounted for properly during the transition**.

##### Possible failure modes:

1. **Partial rollout**

* If the feature is enabled only on the API server but not on the `kube-scheduler` or `kube-controller-manager`:

  * The **API server may admit schedule-gated pods** that exceed quota.
  * The **quota controller (kube-controller-manager)** may still count these pods toward quota usage.
  * The **scheduler** may fail to check quota before scheduling such pods.
  * **Impact**: Inconsistent quota accounting, quota overcommit, or blocked scheduling of other pods.

2. **Partial rollback**

* If the feature is disabled only on the API server but **existing gated pods** (previously admitted under relaxed rules) are still present:

  * The **quota controller will start counting these pods** toward quota again.
  * But the **scheduler won’t know that these pods previously bypassed quota checks**, possibly leading to:

    * Over-counting quota usage (causing unexpected scheduling delays).
    * Pods being scheduled that now exceed quota (if scheduler doesn’t revalidate).

3. **Workload-level impact**

* **Already admitted pods** (while the feature was enabled) will remain in the cluster.

  * These pods may consume resources even if their usage now exceeds quota due to gate removal or rollback.
  * **Impact**: Other pods may be rejected or blocked due to quota exhaustion.

**Summary of rollout/rollback risks:**

| Scenario                                | Risk                                                     | Impact on Workloads                         |
| --------------------------------------- | -------------------------------------------------------- | ------------------------------------------- |
| Feature enabled only on API server      | Controller and scheduler mismatch                        | Inconsistent quota tracking and enforcement |
| Feature disabled while gated pods exist | Already admitted pods may exceed quota post-gate removal | Resource overcommit                         |
| Components on mixed versions            | Feature logic applied inconsistently                     | Unpredictable admission or scheduling       |

##### Mitigation:

* **Coordinate rollout/rollback across all control plane components**.
* **Ensure feature gate is enabled or disabled cluster-wide** (API server, scheduler, controller manager).
* **Avoid rollback while previously gated pods are still Pending** under the feature’s relaxed admission behavior.
* Consider draining or deleting these pods before rollback to prevent quota miscalculations.

This ensures safe transitions without disrupting already-running workloads or compromising scheduling integrity.

#### What specific metrics should inform a rollback?

<!--
What signals should users be paying attention to when the feature is young
that might indicate a serious problem?
-->

The following **metrics and signals** should be monitored to determine whether a rollback of the `ResourceQuotaDeferredEnforcement` feature is necessary:

---

##### 🚨 Primary Metrics to Watch

1. **Quota Overcommit Indicators**

* **`scheduler_resource_quota_violations_total`** (proposed custom metric)

  * Counts the number of times the scheduler blocked a pod from binding due to resource quota constraints.
  * **Spike in this metric** could indicate quota accounting inconsistencies or unexpected quota pressure.

2. **Admission Rejections**

* **`apiserver_admission_step_admission_latencies_seconds`** (existing)

  * Track latencies and rejection spikes in the `ResourceQuota` admission step.
* **`apiserver_request_total{code="Forbidden",resource="pods"}`**

  * Increased `403` errors with reasons like `exceeded quota` may indicate regressions in admission logic.

3. **Quota Usage Anomalies**

* **Namespace quota usage suddenly jumping or becoming inaccurate** after enabling or disabling the feature.

  * Can be inspected via:

    ```sh
    kubectl describe quota -n <namespace>
    ```

4. **Unexpected Scheduler Behavior**

* **Pods stuck in Pending phase indefinitely**

  * Especially if they have readiness gates and are not progressing even when gates are removed and resources are available.
* **`scheduler_bind_latency_seconds`** may show unexpected increases if the scheduler is overloading with failed re-attempts.

---

##### 🔍 Additional Indicators

###### Controller Logs or Events

* Look for logs or events in the `kube-controller-manager` or scheduler mentioning:

  * "pod exceeds quota"
  * "failed to account pod due to scheduling gate"
  * "quota exceeded on binding attempt"

###### User Feedback

* Workload operators or systems like Kueue reporting:

  * Higher rate of job failures or retries
  * Inconsistent scheduling behavior
  * Admission rejections for expected-eligible pods

---
##### When to Roll Back

A rollback should be considered if:

* You see sustained quota usage mismatches across the system.
* There is an increase in rejected or stuck pods that appears tied to gating or deferred quota logic.
* You are unable to explain scheduler or controller behavior with existing logs or metrics.

In such cases, disabling the feature gate and reverting to strict admission enforcement provides a known-good baseline.

#### Were upgrade and rollback tested? Was the upgrade->downgrade->upgrade path tested?

<!--
Describe manual testing that was done and the outcomes.
Longer term, we may want to require automated upgrade/rollback tests, but we
are missing a bunch of machinery and tooling and can't do that now.
-->

#### Is the rollout accompanied by any deprecations and/or removals of features, APIs, fields of API types, flags, etc.?

<!--
Even if applying deprecation policies, they may still surprise some users.
-->

No, the rollout of the `ResourceQuotaDeferredEnforcement` feature is **not** accompanied by any deprecations or removals of:

* Existing features,
* API versions or fields,
* Flags,
* Admission behavior for non-gated pods,
* Quota types (e.g., `resource.count/*`).

**Summary:**

* ✅ All existing behavior remains intact.
* ✅ No API surface is changed or removed.
* ✅ The feature is fully opt-in via a feature gate.
* ❌ No user-facing changes are forced upon upgrade.

This ensures backward compatibility and a safe adoption path.

### Monitoring Requirements

<!--
This section must be completed when targeting beta to a release.

For GA, this section is required: approvers should be able to confirm the
previous answers based on experience in the field.
-->

#### How can an operator determine if the feature is in use by workloads?

<!--
Ideally, this should be a metric. Operations against the Kubernetes API (e.g.,
checking if there are objects with field X set) may be a last resort. Avoid
logs or events for this purpose.
-->

An operator can determine if the `ResourceQuotaDeferredEnforcement` feature is in use by workloads by observing the following indicators:

✅ 1. **Pods have one or more `PodSchedulingReadiness` gates**

Look for:

```yaml
spec:
  schedulingGates:
    - name: example.com/ready
```

---

✅ 2. **Pods request compute resources exceeding namespace `ResourceQuota`**, yet are admitted

These pods would normally be rejected, but are accepted if the feature is enabled.

---

✅ 3. **Pods are not counted in quota usage while their gates are active**

Check quota usage via:

```bash
kubectl describe quota -n <namespace>
```

If admitted gated pods don’t impact usage, the feature is active.

---

✅ 4. **Pods remain in `Pending` with `PodScheduled` condition set to `SchedulingGated`**

This reflects that the scheduler is waiting for the gate to be removed.

---

✅ 5. **Pods remain in `Pending` *after* the scheduling gate is removed, with status condition indicating resource quota constraint**

Example condition:

```yaml
status:
  conditions:
    - type: PodScheduled
      status: "False"
      reason: ResourceQuotaExceeded
```

This indicates that the scheduler has revalidated the pod and found that scheduling it would violate quota constraints, a key behavior of this feature.

---

These five combined signals allow operators to infer that the feature is active and being exercised by admitted workloads.


#### How can someone using this feature know that it is working for their instance?

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

#### What are the reasonable SLOs (Service Level Objectives) for the enhancement?

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

#### What are the SLIs (Service Level Indicators) an operator can use to determine the health of the service?

<!--
Pick one more of these and delete the rest.
-->

- [ ] Metrics
  - Metric name:
  - [Optional] Aggregation method:
  - Components exposing the metric:
- [ ] Other (treat as last resort)
  - Details:

#### Are there any missing metrics that would be useful to have to improve observability of this feature?

<!--
Describe the metrics themselves and the reasons why they weren't added (e.g., cost,
implementation difficulties, etc.).
-->

### Dependencies

<!--
This section must be completed when targeting beta to a release.
-->

#### Does this feature depend on any specific services running in the cluster?

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

No, this feature does **not** depend on any external or optional services running in the cluster.

##### Details:

* It **does not require** any node-level agents, cloud provider APIs, or third-party components.
* It **uses only core Kubernetes components**, including:

  * **API Server** (for feature gate logic and admission control),
  * **kube-controller-manager** (for quota reconciliation),
  * **kube-scheduler** (for quota-aware scheduling decisions).

##### Internal dependency considerations:

* All relevant behavior is confined to Kubernetes core components.
* The feature is designed to be **self-contained**, relying on the existing Pod API, ResourceQuota API, and standard scheduling and admission pathways.

##### Summary:

* ✅ No external services or dependencies required
* ✅ Fully functional in vanilla Kubernetes
* ❌ No impact from the absence of metrics-server, CRI, CSI, or CNI plugins

This makes the feature safe and portable across a wide variety of cluster setups.

### Scalability

<!--
For alpha, this section is encouraged: reviewers should consider these questions
and attempt to answer them.

For beta, this section is required: reviewers must answer these questions.

For GA, this section is required: approvers should be able to confirm the
previous answers based on experience in the field.
-->

#### Will enabling / using this feature result in any new API calls?

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

Enabling and using the `ResourceQuotaDeferredEnforcement` feature does **not introduce any new API types or endpoints**, but it **does affect the frequency and context** of some existing API calls, particularly in the **scheduler** and **controller manager**.

##### Affected API Call Patterns:

✅ 1. **Admission Control**

* **No change** in API call type, admission control remains part of the API server request path (`POST /api/v1/namespaces/{ns}/pods`).
* **Behavioral change**: Admission controller may allow pods that would otherwise be rejected.

✅ 2. **Scheduler – Simulated Quota Admission**

* Before binding a pod that was previously gated, the **scheduler will simulate a quota check** by querying:

  * **ResourceQuota objects** in the pod’s namespace via:

    ```
    GET /api/v1/namespaces/{ns}/resourcequotas
    ```
* This is done **per previously gated pod at scheduling time**, introducing additional **read** traffic, especially under high pod churn.

✅ 3. **Quota Controller – Usage Recalculation**

* The quota controller may:

  * Skip usage accounting for gated pods.
  * Recalculate usage when gates are removed.
* This affects **internal quota accounting logic**, not external API traffic.

---

##### Summary:

| API Call             | Type  | Originating Component  | Behavior Change                               |
| -------------------- | ----- | ---------------------- | --------------------------------------------- |
| `GET resourcequotas` | READ  | kube-scheduler         | Used before scheduling gated pods             |
| `POST pods`          | WRITE | API server (admission) | Admission allowed if gated even if over quota |
| `LIST/GET pods`      | READ  | quota controller       | Filters out gated pods for usage accounting   |

---

##### Impact:

* The increase in API traffic is minor and proportional to:

  * The number of schedule-gated pods
  * The scheduler’s binding rate
* No periodic polling or high-frequency API calls are introduced.

This ensures the feature is scalable and in line with existing control plane patterns.


#### Will enabling / using this feature result in introducing new API types?

<!--
Describe them, providing:
  - API type
  - Supported number of objects per cluster
  - Supported number of objects per namespace (for namespace-scoped objects)
-->

No, enabling or using the `ResourceQuotaDeferredEnforcement` feature will not introduce any new API types.

#### Will enabling / using this feature result in any new calls to the cloud provider?

<!--
Describe them, providing:
  - Which API(s):
  - Estimated increase:
-->

No, enabling or using the `ResourceQuotaDeferredEnforcement` feature will not result in any new calls to the cloud provider.

#### Will enabling / using this feature result in increasing size or count of the existing API objects?

<!--
Describe them, providing:
  - API type(s):
  - Estimated increase in size: (e.g., new annotation of size 32B)
  - Estimated amount of new objects: (e.g., new Object X for every existing Pod)
-->

Enabling or using the `ResourceQuotaDeferredEnforcement` feature will **not significantly increase the size or count** of existing API objects.

📌 Details:

1. **No new fields or annotations are added**

* The feature **does not modify** the schema of `Pod`, `ResourceQuota`, or any other existing object.
* It uses the existing `schedulingGates` field in the `Pod.spec` and existing quota tracking mechanisms.

2. **Pod objects**

* **Size**: Slight increase in size **only if users include `schedulingGates`**, typically a few bytes (e.g., one or two short strings).
* **Count**: The number of Pods admitted might increase slightly if previously-rejected over-quota pods are now allowed due to deferred enforcement. This is bounded by existing `resource.count/pods` quotas.

3. **ResourceQuota objects**

* No size or structural changes. Internals of usage accounting logic are updated, but this does not reflect in object size or frequency.

---

##### 🔍 Summary:

| Resource        | Size Increase               | Count Increase                      | Notes                                     |
| --------------- | --------------------------- | ----------------------------------- | ----------------------------------------- |
| `Pod`           | Minimal (if gates are used) | Possible (more gated pods admitted) | Only if workloads were previously blocked |
| `ResourceQuota` | ❌ No                        | ❌ No                                | No new fields or tracking logic exposed   |
| Other objects   | ❌ No                        | ❌ No                                | No new object types introduced            |

The increase in pod count is **bounded by existing resource.count quotas** and is not unbounded. Overall, the impact on API object size and count is negligible.


#### Will enabling / using this feature result in increasing time taken by any operations covered by existing SLIs/SLOs?

<!--
Look at the [existing SLIs/SLOs].

Think about adding additional work or introducing new steps in between
(e.g. need to do X to start a container), etc. Please describe the details.

[existing SLIs/SLOs]: https://git.k8s.io/community/sig-scalability/slos/slos.md#kubernetes-slisslos
-->

Enabling or using the `ResourceQuotaDeferredEnforcement` feature may result in **minor increases** in the time taken by some control plane operations, but **not to a degree that would meaningfully impact existing SLIs/SLOs** under normal conditions.

📈 Potentially Affected Operations:

1. **Pod Admission (`POST /pods`)**

   * **Change**: Slightly more logic in the `ResourceQuota` admission controller to check for `schedulingGates` and conditionally exclude resource usage.
   * **Impact**: Negligible latency increase (sub-millisecond), no full quota recalculation.

2. **Scheduler Binding Decision**

   * **Change**: When scheduling a previously schedule-gated pod, the scheduler must simulate quota usage before binding.
   * **Impact**: One additional **`GET resourcequotas`** per such pod.
   * **Expected latency**: A few milliseconds per pod, measurable at high pod churn but still within typical scheduler SLOs.

3. **Quota Controller Reconciliation**

   * **Change**: Filtering out gated pods from usage calculations.
   * **Impact**: Slightly more filtering logic per reconciliation loop; unlikely to exceed existing thresholds for controller latency.

---

##### 🔍 Summary Table:

| Operation               | Latency Increase  | Likely to Violate SLOs? |
| ----------------------- | ----------------- | ----------------------- |
| Pod admission           | Negligible (<1ms) | ❌ No                    |
| Scheduler bind decision | Minor (1–5ms)     | ❌ No (bounded rate)     |
| Quota controller sync   | Negligible        | ❌ No                    |

---

⚠️ Worst-case scenarios (unlikely but possible):

* **High volumes of gated pods admitted at once**, combined with scheduler retries under tight quota constraints, may lead to:

  * Slightly elevated scheduler latency.
  * Increased load on quota reads.
  * Risk of exceeding tail latency SLOs in large clusters **only** if left unthrottled.

These scenarios can be mitigated with existing scheduler queue throttling and pod admission rate controls.

Final Verdict:

* ✅ No significant impact on core Kubernetes SLIs/SLOs.
* ✅ All expected increases are bounded, localized, and scale with pod count.
* ❌ No broad regressions in user-facing API responsiveness or scheduler throughput.


#### Will enabling / using this feature result in non-negligible increase of resource usage (CPU, RAM, disk, IO, ...) in any components?

<!--
Things to keep in mind include: additional in-memory state, additional
non-trivial computations, excessive access to disks (including increased log
volume), significant amount of data sent and/or received over network, etc.
This through this both in small and large cases, again with respect to the
[supported limits].

[supported limits]: https://git.k8s.io/community//sig-scalability/configs-and-limits/thresholds.md
-->

Enabling and using the `ResourceQuotaDeferredEnforcement` feature will result in **only minimal and localized increases** in resource usage (CPU, memory, I/O) within core control plane components. These increases are **not expected to be significant** or exceed existing resource allocation assumptions for those components.

---

🔍 Component-by-Component Breakdown:

1. **API Server**

   * **CPU/RAM**: Minimal increase in admission controller logic to check for `schedulingGates`.
   * **Disk/IO**: No additional write or read amplification.
   * **Impact**: Negligible. The logic is simple and constant-time.

2. **kube-controller-manager (ResourceQuota controller)**

   * **CPU**: Slight increase due to filtering out schedule-gated pods during quota usage reconciliation.
   * **RAM**: Unchanged; no new state is stored.
   * **Impact**: Low. The additional logic is simple and scales linearly with pod count, bounded by existing informer caches.

3. **kube-scheduler**

   * **CPU/RAM**: Small increase when evaluating previously gated pods, it performs a simulated admission by reading `ResourceQuota` objects.
   * **IO**: Additional `GET` requests to the API server for `ResourceQuota` objects.
   * **Impact**: Moderate in clusters with very high volumes of gated pods being scheduled concurrently, but still within typical scheduler resource envelopes.

---

🧠 Observability Note:

There is **no increase in long-lived memory**, background goroutines, or persistent disk usage introduced by this feature.

---

##### 📊 Summary:

| Component               | Resource | Impact Level | Notes                                                                   |
| ----------------------- | -------- | ------------ | ----------------------------------------------------------------------- |
| API Server              | CPU/RAM  | Low          | Lightweight gate check at admission time                                |
| kube-controller-manager | CPU      | Low          | Filters out gated pods from usage loop                                  |
| kube-scheduler          | CPU/IO   | Low–Moderate | Performs per-pod quota simulation when scheduling previously gated pods |
| Nodes (kubelet)         | None     | None         | Feature is control-plane only                                           |

---

##### Final Verdict:

* ✅ No persistent resource pressure
* ✅ No new background loops or caches
* ❌ No disk or memory bloat
* ⚠️ Only scenario requiring attention: high churn of gated pods may briefly elevate scheduler CPU/IO, but still remains within safe limits in well-provisioned clusters.


#### Can enabling / using this feature result in resource exhaustion of some node resources (PIDs, sockets, inodes, etc.)?

<!--
Focus not just on happy cases, but primarily on more pathological cases
(e.g. probes taking a minute instead of milliseconds, failed pods consuming resources, etc.).
If any of the resources can be exhausted, how this is mitigated with the existing limits
(e.g. pods per node) or new limits added by this KEP?

Are there any tests that were run/should be run to understand performance characteristics better
and validate the declared limits?
-->

No, enabling or using the `ResourceQuotaDeferredEnforcement` feature **will not result in resource exhaustion on nodes** (e.g., PIDs, sockets, inodes, etc.), because:

✅ Feature is control-plane only:

* All logic related to this feature is confined to the **API server**, **kube-scheduler**, and **kube-controller-manager**.
* It does **not affect the kubelet**, container runtime, or any node-local resource management.

✅ Gated pods do not run on nodes:

* Schedule-gated pods remain in the **Pending** phase until:

  * All `schedulingGates` are removed, **and**
  * The scheduler verifies quota constraints and assigns a node.
* Until then, **they are not pulled, started, or counted against node-level limits** (PIDs, inodes, CPU shares, etc.).

🧱 Cluster-wide controls remain in place:

* `resource.count/pods` quota types still apply during admission and protect against unbounded pod creation.
* Pod-level limits (e.g., `maxPods` per node) continue to constrain resource usage even if this feature is enabled.

---

⚠️ Theoretical risk:

If an operator misconfigures quotas or removes `resource.count/*` constraints entirely:

* It’s possible to admit a **large number of gated pods**, which could eventually all become schedulable.
* If those pods land on nodes in large bursts, resource exhaustion (e.g., PIDs, CPU) is possible, but this risk exists independently of this feature and is mitigated by:

  * Normal scheduling limits
  * Kubelet pod density limits
  * PodDisruptionBudgets and priority policies

---

##### Final Verdict:

| Resource Type | Risk Introduced by Feature? | Mitigation                       |
| ------------- | --------------------------- | -------------------------------- |
| PIDs          | ❌ No                        | Pods not started until scheduled |
| Inodes        | ❌ No                        | No file creation on nodes        |
| Sockets       | ❌ No                        | No node-level workloads launched |
| Network Ports | ❌ No                        | No containers created            |

This feature introduces **no new node-level resource consumption paths**, and therefore does **not pose a resource exhaustion risk** on nodes when used as intended.


### Troubleshooting

<!--
This section must be completed when targeting beta to a release.

For GA, this section is required: approvers should be able to confirm the
previous answers based on experience in the field.

The Troubleshooting section currently serves the `Playbook` role. We may consider
splitting it into a dedicated `Playbook` document (potentially with some monitoring
details). For now, we leave it here.
-->

#### How does this feature react if the API server and/or etcd is unavailable?

If the **API server or etcd becomes unavailable**, the `ResourceQuotaDeferredEnforcement` feature behaves similarly to existing core features, it **degrades gracefully** along with the rest of the Kubernetes control plane. Below is a breakdown of behavior during such outages:

---

1. **API Server Unavailability**

  * Effects:

    * **Pod admission** is blocked, all `POST /pods` requests fail, regardless of whether the feature is enabled.
    * No new gated or non-gated pods can be created.
    * **No impact on already-admitted pods**, existing pods, including gated ones, remain in `Pending`, `Running`, or `Terminating` as appropriate.
    * The scheduler and controller loops will eventually pause due to inability to talk to the API server.

  * Feature-specific impact: 
    * No unique impact, the feature does not introduce any special reliance on the API server beyond normal pod lifecycle interactions.

---

2. **etcd Unavailability**

  * Effects:

    * The API server cannot persist or retrieve objects (e.g., Pods, ResourceQuotas).
    * Pod admission, status updates, and scheduling will all be stalled.
    * Existing in-memory state in kube-scheduler or controller-manager is frozen but may remain active until TTLs or retries expire.

  * Feature-specific impact:

    * No additional degradation or risk beyond standard etcd-related downtime effects.
    * Quota usage reconciliation and scheduling decisions (including deferral logic) are paused along with all other cluster activity.

---

##### Summary Table:

| Component Down | Cluster Behavior              | Impact on Feature                          |
| -------------- | ----------------------------- | ------------------------------------------ |
| API Server     | Pod admission blocked         | Gated pod admission deferred like all pods |
| etcd           | All state persistence blocked | Quota usage and scheduling logic suspended |
| Scheduler      | No change unless API down     | No re-evaluation of quota for pending pods |

---

##### Final Notes:

* The feature introduces **no new failure modes** under API server or etcd outages.
* Its reliance on Kubernetes core components ensures **failure behavior is well understood and consistent** with existing SLOs and expectations.

In short: the system will **pause, not break**, and will **resume safely** once the control plane is restored.


#### What are other known failure modes?

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

Here are the **known failure modes** for the `ResourceQuotaDeferredEnforcement` feature, along with how they can be detected, mitigated, and debugged:

---

##### ⚠️ 1. **Inconsistent Quota Accounting (partial rollout)**

* **Description**: If the API server has the feature enabled, but the `kube-controller-manager` or `kube-scheduler` does not, components may disagree on whether a gated pod counts toward quota.

* **Detection**:

  * Mismatch between actual admitted pods and quota usage reported by `kubectl describe quota`.
  * Scheduler binds a pod that puts total usage over quota.

* **Mitigations**:

  * Ensure all control plane components are upgraded and have the feature gate aligned.
  * Do not enable the feature until the entire control plane is uniformly rolled out.

* **Diagnostics**:

  * Look for logs in the controller manager like:

    ```
    "Including gated pod in quota usage"
    ```
  * Scheduler logs may lack quota validation step.

* **Testing**:

  * Integration tests should verify behavior under misaligned feature gate configurations.

---

##### ⚠️ 2. **Scheduler skips quota validation on previously gated pods**

* **Description**: If the scheduler does not correctly simulate quota usage when re-evaluating a pod after scheduling gate removal, it may schedule a pod that exceeds quota.

* **Detection**:

  * Watch for quota overcommit where total resource usage exceeds quota limits.
  * Pods scheduled despite quota exhaustion.

* **Mitigations**:

  * Ensure feature gate is enabled on the scheduler and logic for deferred enforcement is active.
  * Validate scheduler logs for quota simulation behavior.

* **Diagnostics**:

  * Check for log entries like:

    ```
    "Quota exceeded for pod <name>, deferring scheduling"
    ```

* **Testing**:

  * Unit and e2e tests must verify quota checks on previously gated pods.

---

##### ⚠️ 3. **Pods remain stuck in Pending with unclear reason**

* **Description**: If a pod's scheduling gate is removed but it still can’t be scheduled due to quota limits, the user may be confused why it is still Pending.

* **Detection**:

  * Pods in `Pending` with no obvious events.
  * Pod status condition:

    ```yaml
    - type: PodScheduled
      status: "False"
      reason: ResourceQuotaExceeded
    ```

* **Mitigations**:

  * Ensure this condition is **explicitly set** and that an event is emitted.
  * Educate users that deferred enforcement will reapply quota at scheduling time.

* **Diagnostics**:

  * Look at pod status and events:

    ```bash
    kubectl describe pod <name>
    ```

* **Testing**:

  * e2e tests should verify status conditions and event emission.

---

##### ⚠️ 4. **Admission of excessive gated pods leading to API server pressure**

* **Description**: If `resource.count/pods` quotas are not in place, users could flood the cluster with many gated pods that are not rejected due to deferred enforcement.

* **Detection**:

  * Rapid growth of Pending pods.
  * Increased memory or list/watch pressure on API server.

* **Mitigations**:

  * Ensure `resource.count/pods` quota is enforced even when feature is enabled.
  * Optionally use `LimitRange` to constrain per-pod requests.

* **Diagnostics**:

  * Monitor API server metrics like `apiserver_storage_objects` and `etcd_object_counts`.

* **Testing**:

  * Soak testing with heavy gated pod workloads should validate system resilience.

---

##### ⚠️ 5. **Pod misbehavior after rollback**

* **Description**: If the feature is disabled after gated pods have been admitted under relaxed rules, and the scheduler or controller resumes strict behavior, quota inconsistencies may occur.

* **Detection**:

  * Pods fail to schedule unexpectedly.
  * Quota usage appears inflated or unexpectedly blocks new pods.

* **Mitigations**:

  * Avoid rollback while deferred pods are still in Pending.
  * Clean up or drain such pods before disabling the feature.

* **Diagnostics**:

  * Review admission and quota controller logs post-rollback.

* **Testing**:

  * Upgrade–rollback–upgrade scenarios should be part of integration test coverage.

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

## Drawbacks

<!--
Why should this KEP _not_ be implemented?
-->

Here are several reasons why this KEP *might not* be implemented, or at least why it warrants caution and debate:

---

❌ 1. **Increased Complexity in Quota Semantics**

* Introducing deferred enforcement adds a conditional and time-dependent dimension to `ResourceQuota`, making it harder for operators and users to reason about when resources are truly "used."
* This breaks the existing invariant that quota usage is fully determined at admission time.
* Documentation, tooling, and user expectations would need updates to explain the distinction between "admitted but not counted" pods.

---

❌ 2. **Risk of Resource Overcommit or Abuse**

* If `resource.count/pods` quotas are not configured correctly, users could flood the system with schedule-gated pods that bypass quota limits at admission.
* While such pods won’t run immediately, they still consume API server and controller resources, which could lead to denial-of-service conditions or degraded scheduler performance.

---

❌ 3. **Cross-Component Coupling**

* The feature requires tight coordination across **API server**, **scheduler**, and **controller-manager**. This increases the chance of inconsistent behavior if version skew or partial rollouts occur.
* Rollbacks are not always clean: previously admitted gated pods may require a cleanup after disabling the feature.

---

❌ 4. **Departing from Kubernetes Principles**

* Kubernetes has generally favored **strict admission with conservative enforcement** to ensure predictability and avoid overload.
* Allowing pods that knowingly violate quota at admission, even if gated, might be seen as a philosophical shift — blurring the contract between resource request and admission guarantees.

---

❌ 5. **Limited Use Case Scope**

* The primary motivation is to support workload orchestration tools like **Kueue** and advanced batch systems.
* For general-purpose Kubernetes users, the benefits may not outweigh the added complexity and risk, especially in clusters without scheduling gates or deferred workloads.

---

### Summary:

This KEP introduces powerful scheduling flexibility, but at the cost of quota clarity, admission consistency, and broader system simplicity. 
At the same time, this KEP introduces a **measured, bounded enhancement** to Kubernetes that enables advanced workload orchestration without sacrificing core principles — as long as it is implemented carefully. It unlocks critical capabilities for platforms like **Kueue**, aligns quota behavior with scheduling primitives, and provides a clean migration path by being fully opt-in.

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
