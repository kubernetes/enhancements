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
# KEP-3939: Allow replacement of Pods in a Job when fully terminated

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
  - [Notes/Constraints/Caveats (Optional)](#notesconstraintscaveats-optional)
    - [The default job controller behavior](#the-default-job-controller-behavior)
    - [When Pods enter a terminating state](#when-pods-enter-a-terminating-state)
  - [Exponential Backoff for Pod Failures](#exponential-backoff-for-pod-failures)
  - [Risks and Mitigations](#risks-and-mitigations)
    - [Pods are not guaranteed to transition to a terminal phase](#pods-are-not-guaranteed-to-transition-to-a-terminal-phase)
- [Design Details](#design-details)
  - [Job API Definition](#job-api-definition)
    - [Defaulting and validation](#defaulting-and-validation)
  - [Implementation](#implementation)
  - [Test Plan](#test-plan)
    - [Prerequisite testing updates](#prerequisite-testing-updates)
      - [Unit tests](#unit-tests)
      - [Integration tests](#integration-tests)
      - [e2e tests](#e2e-tests)
  - [Graduation Criteria](#graduation-criteria)
    - [Alpha](#alpha)
    - [Beta](#beta)
    - [GA](#ga)
    - [Deprecation](#deprecation)
  - [Upgrade / Downgrade Strategy](#upgrade--downgrade-strategy)
  - [Version Skew Strategy](#version-skew-strategy)
- [Production Readiness Review Questionnaire](#production-readiness-review-questionnaire)
  - [Feature Enablement and Rollback](#feature-enablement-and-rollback)
    - [How can this feature be enabled / disabled in a live cluster?](#how-can-this-feature-be-enabled--disabled-in-a-live-cluster)
  - [Rollout, Upgrade and Rollback Planning](#rollout-upgrade-and-rollback-planning)
    - [How can a rollout or rollback fail? Can it impact already running workloads?](#how-can-a-rollout-or-rollback-fail-can-it-impact-already-running-workloads)
    - [What specific metrics should inform a rollback?](#what-specific-metrics-should-inform-a-rollback)
    - [Were upgrade and rollback tested? Was the upgrade-&gt;downgrade-&gt;upgrade path tested?](#were-upgrade-and-rollback-tested-was-the-upgrade-downgrade-upgrade-path-tested)
    - [Is the rollout accompanied by any deprecations and/or removals of features, APIs, fields of API types, flags, etc.?](#is-the-rollout-accompanied-by-any-deprecations-andor-removals-of-features-apis-fields-of-api-types-flags-etc)
  - [Monitoring Requirements](#monitoring-requirements)
    - [How can an operator determine if the feature is in use by workloads?](#how-can-an-operator-determine-if-the-feature-is-in-use-by-workloads)
    - [How can someone using this feature know that it is working for their instance?](#how-can-someone-using-this-feature-know-that-it-is-working-for-their-instance)
    - [What are the reasonable SLOs (Service Level Objectives) for the enhancement?](#what-are-the-reasonable-slos-service-level-objectives-for-the-enhancement)
    - [What are the SLIs (Service Level Indicators) an operator can use to determine the health of the service?](#what-are-the-slis-service-level-indicators-an-operator-can-use-to-determine-the-health-of-the-service)
    - [Are there any missing metrics that would be useful to have to improve observability of this feature?](#are-there-any-missing-metrics-that-would-be-useful-to-have-to-improve-observability-of-this-feature)
  - [Dependencies](#dependencies)
    - [Does this feature depend on any specific services running in the cluster?](#does-this-feature-depend-on-any-specific-services-running-in-the-cluster)
  - [Scalability](#scalability)
    - [Will enabling / using this feature result in any new API calls?](#will-enabling--using-this-feature-result-in-any-new-api-calls)
    - [Will enabling / using this feature result in introducing new API types?](#will-enabling--using-this-feature-result-in-introducing-new-api-types)
    - [Will enabling / using this feature result in any new calls to the cloud provider?](#will-enabling--using-this-feature-result-in-any-new-calls-to-the-cloud-provider)
    - [Will enabling / using this feature result in increasing size or count of the existing API objects?](#will-enabling--using-this-feature-result-in-increasing-size-or-count-of-the-existing-api-objects)
    - [Will enabling / using this feature result in increasing time taken by any operations covered by existing SLIs/SLOs?](#will-enabling--using-this-feature-result-in-increasing-time-taken-by-any-operations-covered-by-existing-slisslos)
    - [Will enabling / using this feature result in non-negligible increase of resource usage (CPU, RAM, disk, IO, ...) in any components?](#will-enabling--using-this-feature-result-in-non-negligible-increase-of-resource-usage-cpu-ram-disk-io--in-any-components)
    - [Can enabling / using this feature result in resource exhaustion of some node resources (PIDs, sockets, inodes, etc.)?](#can-enabling--using-this-feature-result-in-resource-exhaustion-of-some-node-resources-pids-sockets-inodes-etc)
  - [Troubleshooting](#troubleshooting)
    - [How does this feature react if the API server and/or etcd is unavailable?](#how-does-this-feature-react-if-the-api-server-andor-etcd-is-unavailable)
    - [What are other known failure modes?](#what-are-other-known-failure-modes)
    - [What steps should be taken if SLOs are not being met to determine the problem?](#what-steps-should-be-taken-if-slos-are-not-being-met-to-determine-the-problem)
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

- [x] (R) Enhancement issue in release milestone, which links to KEP dir in [kubernetes/enhancements] (not the initial KEP PR)
- [x] (R) KEP approvers have approved the KEP status as `implementable`
- [x] (R) Design details are appropriately documented
- [ ] (R) Test plan is in place, giving consideration to SIG Architecture and SIG Testing input (including test refactors)
  - [ ] e2e Tests for all Beta API Operations (endpoints)
  - [ ] (R) Ensure GA e2e tests meet requirements for [Conformance Tests](https://github.com/kubernetes/community/blob/master/contributors/devel/sig-architecture/conformance-tests.md)
  - [ ] (R) Minimum Two Week Window for GA e2e tests to prove flake free
- [x] (R) Graduation criteria is in place
  - [ ] (R) [all GA Endpoints](https://github.com/kubernetes/community/pull/1806) must be hit by [Conformance Tests](https://github.com/kubernetes/community/blob/master/contributors/devel/sig-architecture/conformance-tests.md)
- [x] (R) Production readiness review completed
- [x] (R) Production readiness review approved
- [x] "Implementation History" section is up-to-date for milestone
- [x] User-facing documentation has been created in [kubernetes/website], for publication to [kubernetes.io]
- [x] Supporting documentation—e.g., additional design documents, links to mailing list discussions/SIG meetings, relevant PRs/issues, release notes

<!--
**Note:** This checklist is iterative and should be reviewed and updated every time this enhancement is being considered for a milestone.
-->

[kubernetes.io]: https://kubernetes.io/
[kubernetes/enhancements]: https://git.k8s.io/enhancements
[kubernetes/website]: https://git.k8s.io/website

## Summary

Currently, Jobs start replacement Pods as soon as previously created Pods are terminating (have a `deletionTimestamp`) or fail (`phase=Failed`).
Terminating pods are currently counted as failed in the Job status.
However, terminating pods are actually in a transitory state where they are neither active nor really fully terminated.  
This KEP proposes a new field for the Job API that allows for users to specify if they want replacement Pods as soon as
the previous Pods are terminating (existing behavior) or only once the existing pods are fully terminated (new behavior).

## Motivation

Existing Issues:

- [Job Creates Replacement Pods as soon as Pod is marked for deletion](https://github.com/kubernetes/kubernetes/issues/115844)
- [Kueue: Account for terminating pods when doing preemption](https://github.com/kubernetes-sigs/kueue/issues/510)

Many common machine learning frameworks, such as Tensorflow and JAX, require unique pods per Index.
Currently, if a pod enters a terminating state (due to preemption, eviction or other external factors),
a replacement pod is created and immediately fail to start.

Having a replacement Pod before the previous one fully terminates can also
cause problems in clusters with scarce resources or with tight budgets.
These resources can be difficult to obtain so pods can take a long time to find resources and they may only be able to find nodes once the existing pods have been terminated.
If cluster autoscaler is enabled, the replacement Pods might produce undesired
scale ups.

On the other hand, if a replacement Pod is not immediately created, the Job
status would show that the number of active pods doesn't match the desired
parallelism. To provide better visibility, the job status can have a new field
to track the number of Pods currently terminating.

This new field can also be used by queueing controllers, such as Kueue,
to track the number of terminating pods to calculate quotas.

### Goals

- Job controller should allow for flexibility in waiting for pods to be fully terminated before
  creating replacement Pods
- Job controller will have a new status field where we include the number of terminating pods.

### Non-Goals

- Other workload APIs are not included in this proposal.

## Proposal

The Job controller gets a list of active pods.  Active pods are pods that don't
have a terminal phase (`Succeeded` or `Failed`) and are not terminating 
(have a `deletionTimestamp`)
In this KEP, we will consider terminating pods to be separate from active and failed.  
As an opt-in behavior, the job controller can use the active and terminating
pods to determine whether replacement Pods are needed.

We propose two new API fields:

1. A field in Spec that allows for opt-in behavior of whether to wait for
   terminating pods to finish before creating replacement pods.
2. A new field in Status for tracking the number of terminating pods.

### User Stories (Optional)

#### Story 1

As a machine learning user, ML frameworks allow scheduling of multiple pods.  
The Job controller does not typically wait for terminating pods to be marked as failed.  
Tensorflow and other ML frameworks may have a requirement that they only want Pods to be started once the other pods are fully terminated.

This case was added due to a bug discovered with running IndexedJobs with Tensorflow.  
See [Jobs create replacement Pods as soon as a Pod is marked for deletion](https://github.com/kubernetes/kubernetes/issues/115844) for more details.

#### Story 2

As a cloud user, users would want to guarantee that the number of pods that are running is exactly the amount that they specify.  
Terminating pods do not relinguish resources so scarce compute resource are still scheduled to those pods.
Replacement pods do not produce unnecessary scale ups.

#### Story 3

As a Job-level quota controller, I want to track the number of terminating pods,
in addition to the active pods.

See [Kueue: Account for terminating pods when doing preemption](https://github.com/kubernetes-sigs/kueue/issues/510) for an example of this.

### Notes/Constraints/Caveats (Optional)

#### The default job controller behavior

Based on the [proposed API](#job-api-definition) below, the behavior of the
job controller prior to this KEP is equivalent to
`podReplacementPolicy: TerminatingOrFailed`.

This behavior has the following semantic problems:
- A terminating Pod might gracefully terminate as Succeeded, but it counts
  towards `.status.failed` as soon as it's terminating and it's not reclassified
  upon termination.
- When using podFailurePolicy, the controller might create a replacement Pod
  before being able to evaluate the terminal state of the Pod. The replacement
  Pod might be terminated due to the policy.

In a Job v2 API, we should consider having the default behavior equivalent to
`podReplacementPolicy: Failed`, given the above problems.
We could even consider removing the proposed field `podReplacementPolicy`.

But for backwards compatibility, in v1, we have to introduce a change of
behavior as opt-in.

#### When Pods enter a terminating state

Pods can be marked for termination by several controllers, which we typically
refer to as disruptions, such as: kubelet eviction, scheduler preemption, API eviction, etc.

The job controller itself can delete running Pods, in the following scenarios:

1. A job is over the `activeDeadlineSeconds`.  
1. When the number of Pod failures reaches the `backoffLimit`.
1. With `PodFailurePolicy` active and `FailJob` is set as the action.

In all these situations, the Pod initially gets a `deletionTimestamp`
and we interpret the pod as "terminating". Once the pod terminates, it gets
a terminal `phase` (`Succeeded` or `Failed`).

### Exponential Backoff for Pod Failures

The job controller implements backoff delays to prevent fast recreation of
continuously failing Pods.

This behavior is internal (not configurable through the API) and it's orthogonal
to this KEP. The behavior will be preserved as follows:
- When `podReplacementPolicy: TerminatingOrFailed`, the backoff period counts from
  the time the Pod is terminating or Failed.
- When `podReplacementPolicy: Failed`, the backoff period counts from the time the
  Pod is Failed.

### Risks and Mitigations

#### Pods are not guaranteed to transition to a terminal phase

One area of contention is how this KEP will work with [3329-retriable-and-non-retriable-failures](https://github.com/kubernetes/enhancements/blob/master/keps/sig-apps/3329-retriable-and-non-retriable-failures/README.md).

In 3329, there was a decision to make kubelet transition pods to failed before deleting them.
This is feature toggled guarded by `PodDisruptionCondition`, which in addition to
setting the phase to Failed, it adds a `DisruptionTarget` condition.
This means that when this feature is turned on, the job controller is able to count pods as failed only when they are fully terminated, as it is guaranteed that all pods will reach a terminal state (Failed or Succeeded).
Note that a terminating pod is not considered active either.
If `PodDisruptionCondition` is turned off, then the job controller considers the pod as failed as soon as it is terminating (has a deletion timestamp), because there is no guarantee that the pod will transition to phase=Failed.

Another issue is described [here](https://github.com/kubernetes/enhancements/pull/3940#discussion_r1180777509).
If PodDisruptionConditions is disabled, a pod bound to a no-longer-existing node may be stuck in the Running phase.
As a consequence, it will never be replaced, so the whole job will be stuck from making progress.
When PodDisruptionConditions is enabled, the PodGC transitions the Pod to phase Failed in this scenario.

Due to the above issues, we propose the following mitigation:
- If `PodDisruptionConditions` OR `JobPodReplacementPolicy` are enabled, set
  phase=Failed in kubelet and podGC before deleting a Pod.
- If `JobPodReplacmentPolicy` is enabled, but `PodDisruptionConditions` is
  disabled, the kubelet and podGC only set the phase, but do not add a
  `DisruptionTarget` condition.

## Design Details

### Job API Definition

At the JobSpec level, we are adding a new enum field:

```golang
// This field controls when we recreate pods
// Default will be TerminatingOrFailed ie recreate pods when they are failed
// +enum 
type PodReplacementPolicy string
const (
 // TerminatingOrFailed is a policy that creates replacement pods when they are
 // marked as terminating (have a deletion timestamp) or reach the terminal
 // phase `Failed`.
 // Terminating pods count towards `.status.failed`, even if they later reach
 // the terminal phase `Succeeded`.
 TerminatingOrFailed PodReplacementPolicy = "TerminatingOrFailed"
 // Failed is a policy that creates replacement Pods only when the previously
 // created Pods reach the terminal phase `Failed`.
 Failed PodReplacementPolicy = "Failed"
)
```

```golang
type JobSpec struct{
  ...
 // podReplacementPolicy specifies when to create replacement Pods. Possible values are:
 // - TerminatingOrFailed means to create a replacement Pod when the previously
 //   created Pod is terminating or failed.
 // - Failed means to wait until a previously created Pod is fully terminated
 //   before creating a replacement Pod.
 //
 // When using podFailurePolicy, the default value is Failed and this is the
 // only allowed policy.
 // When not using podFailurePolicy, the default value is TerminatingOrFailed.
 // +optional
 PodReplacementPolicy *PodReplacementPolicy
}
```

In order to offer visibility of the number of terminating pods, we include a new
field in the JobStatus.

```golang
type JobStatus struct {
  ...
  // Number of terminating pods
  // +optional
  terminating *int32
}
```

#### Defaulting and validation

Defaulting of `podReplacementPolicy` will depend on whether `podFailurePolicy`
is in use:
- when `podFailurePolicy` is in use, the default value is `Failed`.
- when `podFailurePolicy` is not in use, the default value is `TerminatingOrFailed`.

When `podFailurePolicy` is in use, the only allowed value for `podFailurePolicy`
is `Failed`.

### Implementation

As part of this KEP, we need to track pods that are terminating (`deletionTimestamp != nil` and `phase` is `Pending` or `Running`).

The following algorithm could be used:

1. Count the number of pods that are active and not terminating.
2. Count the number of terminating pods.
3. In `manageJob` we will count expected pods as:
  - when `podReplacementPolicy: Failed` then `expectedPods = active + terminating`.
  - when `podReplacementPolicy: TerminatingOrFailed` then `expectedPods = active`.
4. Use the expected number of pods to decide whether to recreate.

In Indexed completion mode, the tracking of pods is per index.

The controller updates the field `Status.terminating` with the number of terminating pods.
For backwards compatibility, when `podReplacementPolicy: TerminatingOrFailed`,
the number of failed pods includes the terminating pods.

The controller updates the terminating field in the same API call where it
updates other counters, so it should not require any extra API calls.  

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

#### Prerequisite testing updates

<!--
Based on reviewers feedback describe what additional tests need to be added prior
implementing this enhancement to ensure the enhancements have also solid foundations.
-->

##### Unit tests

- `controller_utils`: `April 3rd 2023` - `56.6`
  - Adding tests to help determine if pods are terminating.
- `job`: `April 3rd 2023` - `90.4`
   a. Verify that terminating pods are in fact counted in the status.
   b. Recreate pods only once pod is fully terminated (ie `Failed`)
   c. Verify existing behavior with `TerminatingOrFailed`
   d. If feature is off verify existing behavior
   e. Count terminating pods even if terminating Pod considered failed when `JobPodReplacementPolicy` is disabled
   f. Count terminating pods even if terminating Pod not considered failed when `JobPodReplacementPolicy` is enabled
- `gc_controller.go`: `April 3rd 2023` - `82.4`
   a. Set `PodPhase` to `failed` when `JobPodReplacementPolicy` true but `PodDisruptionConditions` is false

##### Integration tests

<!--
This question should be filled when targeting a release.
For Alpha, describe what tests will be added to ensure proper quality of the enhancement.

For Beta and GA, add links to added tests together with links to k8s-triage for those tests:
https://storage.googleapis.com/k8s-triage/index.html
- <test>: <link to test coverage>
-->

We will add the following integration test for the Job controller:

Case with `JobPodReplacementPolicy` on and `podReplacementPolicy: Failed`

  1. Job starts pods that takes a while to terminate
  2. Delete pods
  3. Verify that `terminating` is tracked
  4. Verify that pod creation only occurs once pod is fully terminated.

Case with `JobPodReplacementPolicy` on and `podReplacementPolicy: TerminatingOrFailed`

  1. Job starts pods that takes a while to terminate
  2. Delete pods
  3. Verify that `terminating` is tracked
  4. Verify that pod creation only occurs once deletion happens.

Case With `JobPodReplacementPolicy` off

  1. Job starts pods that takes a while to terminate
  2. Delete pods
  3. Verify that `terminating` is not tracked
  4. Verify that pod creation only occurs once deletion happens.

Case for disable and reenable `JobPodReplacementPolicy`

  1. Create Job with `podReplacementPolicy: Failed`
  2. Job starts pods that takes a while to terminate
  3. Restart controller and disable `JobPodReplacementPolicy`
  4. Delete some pods
  5. Verify that terminating pods count as failed and pods are recreated.
  6. Restart controller and reenable `JobPodReplacementPolicy`
  7. Terminate pods with phase Succeeded.
  8. Verify that pods still count as failed.
  9. Delete remaining Pods.
  10. Verify that `terminating` is tracked.
  11. Verify that pod creation only occurs once pod is fully terminated.
  12. Verify that pod creation only occurs once deletion happens.

To cover cases with `PodDisruptionCondition` we really only need to worry about tracking terminating fields.  
Tests will verify counting of terminating fields regardless of `PodDisruptionCondition` being on or off.  

##### e2e tests

Generally the only tests that are useful for this feature are when `PodReplacementPolicy: Failed`.

An example job spec that can reproduce this issue is below:

```yaml
apiVersion: batch/v1
kind: Job
metadata:
  name: job-slow-cleanup-with-pod-recreate-feature
spec:
  completions: 1
  parallelism: 1
  backoffLimit: 2
  podReplacementPolicy: Failed
  template:
    spec:
      restartPolicy: Never
      containers:
      - name: sleep
        image: gcr.io/k8s-staging-perf-tests/sleep
        args: ["-termination-grace-period", "1m", "60s"]
```

A e2e test can verify that deletion will not trigger a new pod creation until the exiting pod is fully deleted.  

If `podReplacementPolicy: TerminatingOrFailed` is specified we would test that pod creation happens closely after deletion.

<!--
This question should be filled when targeting a release.
For Alpha, describe what tests will be added to ensure proper quality of the enhancement.

For Beta and GA, add links to added tests together with links to k8s-triage for those tests:
https://storage.googleapis.com/k8s-triage/index.html

We expect no non-infra related flakes in the last month as a GA graduation criteria.
- <test>: <link to test coverage>
-->

### Graduation Criteria

#### Alpha

- Job controller can consider terminating pods as active
- Job controller counts terminating pods in `JobStatus`.
- Unit Tests
- Integration tests

#### Beta

- Address reviews and bug reports from Alpha users
- E2e tests are in Testgrid and linked in KEP
- The feature flag enabled by default
- `job_pods_creation_total` metric is added.

#### GA

- Address reviews and bug reports from Beta users
- Lock the `JobPodReplacementPolicy` feature-gate to true

#### Deprecation

- Remove `JobPodReplacementPolicy` feature-gate in GA+2.

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

#### How can this feature be enabled / disabled in a live cluster?

- [x] Feature gate (also fill in values in `kep.yaml`)
  - Feature gate name: JobPodReplacementPolicy
  - Components depending on the feature gate:
    - kube-apiserver (for field control)
    - kube-controller-manager (for main functionality)
    - kubelet (for supporting functionality: transition to phase=Failed)

###### Does enabling the feature change any default behavior?

Yes,

a. Count the number of terminating pods and populate in JobStatus
b. Set phase=Failed in kubelet and pod-GC before deleting a Pod object
   (behavior also present when related `PodDisruptionConditions` is enabled)
c. As part of closely related KEP-3329, we will default `podReplacementPolicy`
   to Failed if podFailurePolicy is set which, as described above, will change
   the way of handling terminating pods.

###### Can the feature be disabled once it has been enabled (i.e. can we roll back the enablement)?

Yes.

When the feature is disabled:
- the apiserver:
  - Discards the value of `podReplacementPolicy` for new objects.
  - Preserves the value of `podRepacementPolicy` for existing objects.
- the job controller:
  - processes the Job as `podReplacementPolicy: TerminatingOrFailed` (the existing behavior)
  - stops tracking terminating pods, sets the value of `.status.terminating` to
    `nil` in the next Job sync.

<!--
Describe the consequences on existing workloads (e.g., if this is a runtime
feature, can it break the existing applications?).

Feature gates are typically disabled by setting the flag to `false` and
restarting the component. No other changes should be necessary to disable the
feature.

NOTE: Also set `disable-supported` to `true` or `false` in `kep.yaml`.
-->

###### What happens if we reenable the feature if it was previously rolled back?

The job controller will respect the value of `podReplacementPolicy` for new
events (new Pods becoming terminating or failed).

If `podReplacementPolicy: Failed` and there are currently terminating Pod(s) that
were already considered Failed before reenabling the feature, they won't be
re-evaluated.

###### Are there any tests for feature enablement/disablement?

No, but we will add unit and integration tests for feature enablement and disablement.  

An integration test verifies disable and reenable.
See [integration tests](#integration-tests) for details.

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

#### What specific metrics should inform a rollback?

- job_syncs_total, exposed by kube-controller-manager
  - If the number of syncs increases it could mean that we have an increased number of failures.
<!--
What signals should users be paying attention to when the feature is young
that might indicate a serious problem?
-->

#### Were upgrade and rollback tested? Was the upgrade->downgrade->upgrade path tested?

<!--
Describe manual testing that was done and the outcomes.
Longer term, we may want to require automated upgrade/rollback tests, but we
are missing a bunch of machinery and tooling and can't do that now.
-->

#### Is the rollout accompanied by any deprecations and/or removals of features, APIs, fields of API types, flags, etc.?

No.
<!--
Even if applying deprecation policies, they may still surprise some users.
-->

### Monitoring Requirements

<!--
This section must be completed when targeting beta to a release.

For GA, this section is required: approvers should be able to confirm the
previous answers based on experience in the field.
-->

#### How can an operator determine if the feature is in use by workloads?

During pod terminations, an operator can see that the terminating field is being set.

We will use a new metric:

- `job_pods_creation_total` (new) the `action` label will mention what triggers creation (`new`, `recreateTerminatingOrFailed`, `recreateTerminated`))
This can be used to get the number of pods that are being recreated due to `recreateTerminated`.  Otherwise we would expect to see `new` or `recreateTerminatingOrFailed` as the normal values.  

<!--
Ideally, this should be a metric. Operations against the Kubernetes API (e.g.,
checking if there are objects with field X set) may be a last resort. Avoid
logs or events for this purpose.
-->

#### How can someone using this feature know that it is working for their instance?

If a user terminates pods that are controlled by a job, then we should wait
until the existing pods are terminated before starting new ones.

When feature is turned on, we will also include a `terminating` field in the Job Status if there are any terminating pods.

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

- [x] Metrics
  - Metric name:
    - `job_syncs_total` (existing): can be used to see how much the
feature enablement causes the number of syncs to increase.
  - Components exposing the metric: kube-controller-manager

#### Are there any missing metrics that would be useful to have to improve observability of this feature?

### Dependencies

In [Risks and Mitigations](#risks-and-mitigations) we discuss the interaction with [3329-retriable-and-non-retriable-failures](https://github.com/kubernetes/enhancements/blob/master/keps/sig-apps/3329-retriable-and-non-retriable-failures/README.md).  
We will have to guard against cases if `PodFailurePolicy` is off while this feature is on.  
`PodFailurePolicy` is in beta and is enabled by default but we should guard against cases where `PodDisruptionCondition` is turned off.

#### Does this feature depend on any specific services running in the cluster?

No

### Scalability

Generally, enabling this will slow down pod creation if pods take a long time to terminate.  We would wait
to create new pods until the existing ones are terminated.

#### Will enabling / using this feature result in any new API calls?

No

#### Will enabling / using this feature result in introducing new API types?

No

#### Will enabling / using this feature result in any new calls to the cloud provider?

No

#### Will enabling / using this feature result in increasing size or count of the existing API objects?

For Job API, we are adding a enum field named `PodReplacementPolicy` which takes
either a `TerminatingOrFailed` or `Failed`

- API type(s): enum
- Estimated increase in size: 8B

We are also added a status field for tracking terminating pods.

- API type(s): int32
- Estimated increase in size: 4B

#### Will enabling / using this feature result in increasing time taken by any operations covered by existing SLIs/SLOs?

No, SLI/SLO do not include time taking to create new pods if existing ones are terminated.  
There is an existing one on pod creation but this will not impact that.  

<!--
Look at the [existing SLIs/SLOs].

Think about adding additional work or introducing new steps in between
(e.g. need to do X to start a container), etc. Please describe the details.

[existing SLIs/SLOs]: https://git.k8s.io/community/sig-scalability/slos/slos.md#kubernetes-slisslos
-->

#### Will enabling / using this feature result in non-negligible increase of resource usage (CPU, RAM, disk, IO, ...) in any components?

N/A
<!--
Things to keep in mind include: additional in-memory state, additional
non-trivial computations, excessive access to disks (including increased log
volume), significant amount of data sent and/or received over network, etc.
This through this both in small and large cases, again with respect to the
[supported limits].

[supported limits]: https://git.k8s.io/community//sig-scalability/configs-and-limits/thresholds.md
-->

#### Can enabling / using this feature result in resource exhaustion of some node resources (PIDs, sockets, inodes, etc.)?

N/A
<!--
Focus not just on happy cases, but primarily on more pathological cases
(e.g. probes taking a minute instead of milliseconds, failed pods consuming resources, etc.).
If any of the resources can be exhausted, how this is mitigated with the existing limits
(e.g. pods per node) or new limits added by this KEP?

Are there any tests that were run/should be run to understand performance characteristics better
and validate the declared limits?
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

#### How does this feature react if the API server and/or etcd is unavailable?

No change from existing behavior of the Job controller.

#### What are other known failure modes?

#### What steps should be taken if SLOs are not being met to determine the problem?

## Implementation History

- 2023-04-03: Created KEP
- 2023-05-19: KEP Merged.
- 2023-07-16: Alpha PRs merged.

## Drawbacks

Enabling this feature may have rollouts become slower.

## Alternatives

We discussed having this under the PodFailurePolicy but this is a more general idea than the PodFailurePolicy.

## Infrastructure Needed (Optional)

NA
