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
# KEP-3939: Allow for recreation of pods once fully terminated

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
  - [Job API Definition](#job-api-definition)
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
[kubernetes/website]: https://git.k8s.io/website

## Summary

Currently, Jobs start replacement Pods as soon as previously created Pods are marked for terminating.  Terminating pods are in a transitory state where they are neither active nor really fully terminated.  
This KEP proposes a new field for the Job controller that allows for users to specify if they want to recreate pods once the existing pods are fully terminated.  

## Motivation

Existing Issues:

- [Job Creates Replacement Pods as soon as Pod is marked for deletion](https://github.com/kubernetes/kubernetes/issues/115844)
- [Kueue: Account for terminating pods when doing preemption](https://github.com/kubernetes-sigs/kueue/issues/510)

Many common machine learning frameworks, such as Tensorflow and JAX, require unique pods.  Currently if a pod is killed, a replacement pod is created and can cause undesirable behavior.  This is a rare case but it can provide problems if a job needs to guarantee that the existing pods terminate
before starting new pods.  

In scarce compute environments, these resources can be difficult to obtain so pods can take a long time to find resources and they may only be able to find nodes once the existing pods have been terminated.

If a job is stuck in terminating, it could be possible for autoscaling to kick in and give a new node.  
This is not ideal if you could just wait for the pod to terminating and reuse that node.  

### Goals

- Job controller should allow for flexibility in waiting for pods to be fully terminated before
  creating new ones
- Job controller will have a new status field where we include the number of terminating pods.

### Non-Goals

- Other workload APIs are not included in this proposal.

## Proposal

The Job controller gets a list of active pods.  Active pods usually mean pods that have not been registered for deletion.  
In this KEP, we will consider terminating pods to be separate from active and failed.  
In the job controller, we should include a field that states the number of terminating pods.  

We propose two new API fields:

1) A field in Spec that allows for opt-in behavior of whether to wait for terminating pods to finish before recreating.
2) A new field in Status for tracking the number of terminating pods.

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
See [Kueue: Account for terminating pods when doing preemption](https://github.com/kubernetes-sigs/kueue/issues/510) for an example of this.

### Notes/Constraints/Caveats (Optional)

A focus of this KEP is for nonprogramatic deletion (ie kubelet eviction, preemption, etc) so we want to mention some other cases where the job controller can start deletion.

1) A job is over the `activeDeadlineSeconds` so the child pods are deleted.  
2) With `PodFailurePolicy` active and `FailJob` is set as the action, the children pods would be deleted.

### Risks and Mitigations

One area of contention is how this KEP will work with 3329-retriable-and-non-retriable-failures.  It is important to mention the subtleties here.  
In 3329, there was a decision to make kubelet transition pods to failed before deleting them.  This is feature toggled guarded by `PodDisruptionCondition`.  
This means that when this feature is turned on, the job controller can count pods as failed once they are fully terminated.  This means that the pod is still considered running, as opposed to failed.  So this pod would be considered active until it is fully terminated.  
If `PodDisruptionCondition` is turned off then the job controller considers the pod as failed as soon as it is terminating (has a deletion timestamp), because there is no guarantee that the pod will transition to phase=Failed.
This causes some problems in tracking for active versus failed because we have different behavior based on `PodDisruptionCondition`.  To migitate this, we decided to add a new field called `terminating`.  This will track terminating pods separately from active and failed.  

Another issue is described here: https://github.com/kubernetes/enhancements/pull/3940#discussion_r1180777509.  
If PodDisruptionConditions is disabled, a pod bound to a no-longer-existing node may be stuck in the Running phase. As a consequence, it will never be replaced, so the whole job will be stuck from making progress.  Due to the above issue, we can set phase=Failed when PodDisruptionConditions is enabled OR JobRecreatePodsWhenFailed is enabled. When JobRecreatePodsWhenFailed enabled, but PodDisruptionConditions disabled we would just set the phase, but without adding the condition.  We will need to modify the PodGC (gc_controller.go) in the case of `JobRecreatePodsFailed` being enabled while `PodDisruptionConditions` is disabled.  

## Design Details

### Job API Definition

At the JobSpec level, we are adding a new enum field:

```golang
// This field controls when we recreate pods
// Default will be TerminatingOrFailed ie recreate pods when they are failed
// +enum 
type RecreatePodsWhen string
const (
 // This is a field that recreates pods when they are marked as terminating or failed
 // ie this means that as soon as pods get marked for deletion they will be recreated
 TerminatingOrFailed RecreatePodsWhen = "TerminatingOrFailed"
 // Only recreate pods when they are marked as failed.
 Failed              RecreatePodsWhen = "Failed"
)
```

```golang
type JobSpec struct{
  ...
 // RecreatePodsWhen specifies when pods should be recreated.
 // TerminatingOrFailed means to recreate when a pod is either terminating or failed
 // TerminatingOrFailed is the default.
 // Failed means to wait until pods are fully terminated or failed before recreating
 // +optional
 RecreatePodsWhen *RecreatePodsWhen
}
```

So we can count terminating pods separately from active or failed we need to include a new field in the JobStatus.

```golang
type JobStatus struct {
  ...
  // Number of terminating pods
  // +optional
  terminating *int32
}
```

We will allow only opt-in behavior for this feature so we will fall back to `TerminatingOrFailed` for `RecreatePodsWhen`.  

### Implementation

The Job Controller filters all pods and classifies the pods as active.  This list is used to verify that the expected number of active pods matches the reality.  

As part of this KEP, we want to include pods that are also terminating (`DeletionTimestamp != nil`).  
The field `Status.terminating` will include the number of terminating pods.  Once the pods are fully terminated, this field will be unset again because we will obtain the count of active and terminating pods and update the status.

We will update the Status field in the same place as when we update the number of active jobs so it should not require any extra API calls.  

In cases where pods are terminating outside of the job controller, we will look at the list of pods that a job has.  And we can classify them as terminating and add that to the status field.

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
- `job`: `April 3rd 2023` - `90.4`

##### Integration tests

<!--
This question should be filled when targeting a release.
For Alpha, describe what tests will be added to ensure proper quality of the enhancement.

For Beta and GA, add links to added tests together with links to k8s-triage for those tests:
https://storage.googleapis.com/k8s-triage/index.html
- <test>: <link to test coverage>
-->

We will add the following integration test for the Job controller:

JobRecreatePodsWhenFailed Feature Toggle On:

  1) Job starts pods that takes a while to terminate
  2) Delete pods
  3) Verify that pod creation only occurs once terminating pods are removed

We should test the above with the FeatureToggle off also.

##### e2e tests

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

#### GA

- Address reviews and bug reports from Beta users
- Write a blog post about the feature
- Graduate e2e tests as conformance tests
- Lock the `JobRecreatePodsWhenFailed` feature-gate
- Declare deprecation of the `JobRecreatePodsWhenFailed` feature-gate in documentation

#### Deprecation

- Remove feature toggle and code using that.

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
  - Feature gate name: JobRecreatePodsWhenFailed
  - Components depending on the feature gate: kube-controller-manager

###### Does enabling the feature change any default behavior?

For enabling JobRecreatePodsWhenFailed:

a) Count the number of terminating pods and populate in JobStatus
b) With `RecreationPodsWhen: Failed` specified, pods will only be recreated when they are fully terminated.

This could potentially make jobs (where pods are terminated) slower due to waiting for terminating pods
to be fully deleted.

###### Can the feature be disabled once it has been enabled (i.e. can we roll back the enablement)?

Yes.
<!--
Describe the consequences on existing workloads (e.g., if this is a runtime
feature, can it break the existing applications?).

Feature gates are typically disabled by setting the flag to `false` and
restarting the component. No other changes should be necessary to disable the
feature.

NOTE: Also set `disable-supported` to `true` or `false` in `kep.yaml`.
-->

###### What happens if we reenable the feature if it was previously rolled back?

Terminating pods will no longer be tracked so terminating pods will be considered deleted and new pods will be created.

###### Are there any tests for feature enablement/disablement?

Yes. Unit tests will include the fields off/on and verify behavior.

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

<!--
Ideally, this should be a metric. Operations against the Kubernetes API (e.g.,
checking if there are objects with field X set) may be a last resort. Avoid
logs or events for this purpose.
-->

#### How can someone using this feature know that it is working for their instance?

If a user terminates pods that are controlled by a deployment/job, then we should wait
until the existing pods are terminated before starting new ones.

When feature is turned on, we will also include a `terminating` field in the Job Status.

We will add e2e test that determine this.

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

NA

#### Are there any missing metrics that would be useful to have to improve observability of this feature?

### Dependencies

This feature is closely related to the 3329-retriable-and-nonretriable-failures but not sure if that is considered a dependency.

#### Does this feature depend on any specific services running in the cluster?

No

### Scalability

Generally, enabling this will slow down job creation if pods take a long time to terminate.  We would wait
to create new pods until the existing ones are terminated

#### Will enabling / using this feature result in any new API calls?

No

#### Will enabling / using this feature result in introducing new API types?

We add `RecreatePodsWhen` to `JobSpec`.  This is a enum of two values.

We add `terminating` to `JobStatus`. This is a pointer to type `int32`.

#### Will enabling / using this feature result in any new calls to the cloud provider?

No

#### Will enabling / using this feature result in increasing size or count of the existing API objects?

For Job API, we are adding a enum field named `RecreatePodsWhen` which takes either a `TerminateOrFailed` or `Failed`

- API type(s): enum
- Estimated increase in size: 8B

We are also added a status field for tracking terminating pods.

- API type(s): int32
- Estimated increase in size: 4B

#### Will enabling / using this feature result in increasing time taken by any operations covered by existing SLIs/SLOs?

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

- Initial KEP

## Drawbacks

Enabling this feature may have rollouts become slower.

## Alternatives

We discussed having this under the PodFailurePolicy but this is a more general idea than the PodFailurePolicy.

## Infrastructure Needed (Optional)

NA
