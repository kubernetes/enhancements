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
# KEP-3939: Count Terminating Pods Separately From Failed/Active

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
    - [Open Questions on Deployment Controller](#open-questions-on-deployment-controller)
    - [Open Questions on Job Controller](#open-questions-on-job-controller)
  - [Risks and Mitigations](#risks-and-mitigations)
- [Design Details](#design-details)
  - [API Name Choices](#api-name-choices)
  - [Job API Definition](#job-api-definition)
  - [Deployment/ReplicaSet API](#deploymentreplicaset-api)
  - [Implementation](#implementation)
  - [Test Plan](#test-plan)
      - [Prerequisite testing updates](#prerequisite-testing-updates)
      - [Unit tests](#unit-tests)
      - [Integration tests](#integration-tests)
      - [e2e tests](#e2e-tests)
  - [Graduation Criteria](#graduation-criteria)
    - [Alpha](#alpha)
    - [Alpha](#alpha-1)
    - [Beta](#beta)
    - [GA](#ga)
    - [Deprecation](#deprecation)
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

Currently, Jobs and Deployments start Pods as soon as they are marked for terminating.  Terminating pods are in a transitory state where they are neither active nor really fully terminated.  
This KEP proposes a new field for the Job, Deployment/ReplicaSet controllers that counts terminating
pods as like they were active.  The goal of this KEP is to allow for opt-in behavior where terminating pods count as active.  This will allow users to see that new pods will be only created once the existing pods have fully terminated.

## Motivation

Existing Issues:

- [Job Creates Replacement Pods as soon as Pod is marked for deletion](https://github.com/kubernetes/kubernetes/issues/115844)
- [Option for acknowledging terminating Pods in Deployment rolling update](https://github.com/kubernetes/kubernetes/issues/107920)
- [Kueue: Account for terminating pods when doing preemption](https://github.com/kubernetes-sigs/kueue/issues/510)

Many common machine learning frameworks, such as Tensorflow, require unique pods.  Terminating pods that count as active pods
can cause errors.  This is a rare case but it can provide problems if a job needs to guarantee that the existing pods terminate
before starting new pods.  

In [Option for acknowledging terminating Pods in Deployment rolling update](https://github.com/kubernetes/kubernetes/issues/107920),
there is a request in the Deployment API to guarantee that the number of replicas should include terminating.  Terminating pods
do utilize resources because resources are still allocated to them and there is potential for a user to be charged for utilizing those resources.

In scarce compute environments, these resources can be difficult to obtain so pods can take a long time to find resources and they may only be able to find nodes once the existing pods have been terminated.

### Goals

- Job Controller should only create new pods once the existing ones are marked as Failed/Succeeded
- Job Controller will have a new status field where we include the number of terminating pods.
- Deployment controller should allow for flexibility in waiting for pods to be fully terminated before
  creating new ones
- Deployment/ReplicaSet will have a new status field where we include the number of terminating replicas.

### Non-Goals

- DaemonSets and StatefulSets are not included in this proposal
  - They were designed to enforce uniqueness from the start so we will not include them in this design.

## Proposal

Both Jobs and the ReplicaSet controller get a list of active pods.  Active pods usually mean pods that have not been registered for deletion.  In this KEP, we will consider terminating pods to be separate from active and failed.  This means that for cases where we track the number of pods, like the Job Controller, we should include a field that states the number of terminating pods.  

We will propose new API fields in Jobs and Deployments/ReplicaSets in this KEP.  

### User Stories (Optional)

#### Story 1

As a machine learning user, ML frameworks allow scheduling of multiple pods.  
The Job controller does not typically wait for terminating pods to be marked as failed.  Tensorflow and other ML frameworks may have a requirement that they only want Pods to be started once the other pods are fully terminated.  The following yaml can fit these needs:

This case was added due to a bug discovered with running IndexedJobs with Tensorflow.  See [Jobs create replacement Pods as soon as a Pod is marked for deletion](https://github.com/kubernetes/kubernetes/issues/115844) for more details.

#### Story 2

As a cloud user, users would want to guarantee that the number of pods that are running is exactly the amount that they specify.  Terminating pods do not relinguish resources so scarce compute resource are still scheduled to those pods.
See [Kueue: Account for terminating pods when doing preemption](https://github.com/kubernetes-sigs/kueue/issues/510) for an example of this.

#### Story 3

As a cloud user, users would want to guarantee that the number of pods that are running includes terminating pods.  In scare compute environments, users may only have a limited amount of nodes and they do not want to try and schedule pods to a new resource.
Counting terminating pods as active allows for the scheduling of pods to wait until pods are terminated.

See [Option for acknowledging terminating Pods in Deployment rolling update](https://github.com/kubernetes/kubernetes/issues/107920)
for more examples.

### Notes/Constraints/Caveats (Optional)

#### Open Questions on Deployment Controller

The Deployment API is open for discussion.  We put the field in Deployment/ReplicaSet because it is related to RolloutStrategy.
It is not clear if `recreate` and/or `rollingupdate` need this API for both rollout options.

Another open question is if we want to include Deployments in the initial release of this feature.  There is some discussion about releasing the Job API first and then follow up with Deployment.  

We decided to define the APIs in this KEP as they can utilize the same implementation.

#### Open Questions on Job Controller

With 3329-retriable-and-non-retriable-failures and PodFailurePolicy enabled, terminating pods are only marked as failed once they have been transitioned to failed.  If PodFailurePolicy is disabled, then we mark a terminating pod as failed as soon as deletion is registered.  

Should we add a new field to the status that reflects terminating pods?


[Job controller should wait for Pods to be in a terminal phase before considering them failed or succeeded](https://github.com/kubernetes/kubernetes/issues/116858) is a relevant issue for this case.  
I am not sure how to handle these two different cases if we want to count terminating pods as active.  

Should we use this feature to help solve 116858?  When this feature toggle is on, then we mark terminating pods only as failed once they are complete regardless of PodFailurePolicy.

### Risks and Mitigations

<!--
What are the risks of this proposal, and how do we mitigate? Think broadly.
For example, consider both security and how this will impact the larger
Kubernetes ecosystem.

How will security be reviewed, and by whom?

How will UX be reviewed, and by whom?

Consider including folks who also work outside the SIG or subproject.
-->

## Design Details

### API Name Choices

- TerminatingAsActive
- ActiveUntilTerminal
- DelayPodRecreationUntilTerminal
- ?

### Job API Definition

At the JobSpec level, we are adding a new BoolPtr field:

```golang
type JobSpec struct{
  ...
 // terminatingAsActive specifies if the Job controller should include terminating pods
 // as active. If the field is true, then the Job controller will include active pods
 // to mean running or terminating pods
 // +optional
 TerminatingAsActive *bool
}
```

So we can count terminating pods separately from active or failed we need to include a new field in the JobStatus.

```golang
type JobStatus struct {
  ...
  // Number of terminating pods
  // +optional
  terminating int32
}
```

### Deployment/ReplicaSet API

```golang
// DeploymentSpec stores information about the strategy and rolling-update
// behavior of a deployment.
type DeploymentSpec struct {
  ... 
  // TerminatingAsActive specifies if the Deployments should include terminating pods
 // as active. If the field is true, then the Deployment controller will include active pods
 // to mean running or terminating pods
 // +optional
 TerminatingAsActive *bool
}
```

So we can count terminating pods separately from active or failed we need to include a new field in the ReplicaSetStatus and DeploymentStatus.

```golang
type DeploymentStatus struct {
  ...
  // Terminating replicas states the number of replicas that are terminating
  // +optional
  TerminatingReplicas int32 
}
```

```golang
type ReplicaSetStatus struct {
  ...
  // Terminating replicas states the number of replicas that are terminating
  // +optional
  TerminatingReplicas int32 
}
```

In [Option for acknowledging terminating Pods in Deployment rolling update](https://github.com/kubernetes/kubernetes/issues/107920)
there was a request to add this as part of the `DeploymentStrategy` field.  Generally, handling terminating pods as active can be useful in both RollingUpdates and Recreating rollouts.  Having this field for both strategies allows for handling of terminating pods in both cases.  

Deployments create ReplicaSets so there is a need to add a field in the ReplicaSet as well.  Since ReplicaSets are not typically
set by users, we should add a field to the ReplicaSet that is set from the DeploymentSpec.  

```golang
// ReplicaSetSpec is the specification of a ReplicaSet.
// As the internal representation of a ReplicaSet, it must have
// a Template set.
type ReplicaSetSpec struct {
  ...
 // TerminatingAsActive specifies if the Deployments should include terminating pods
 // as active. If the field is true, then the Deployment controller will include active pods
 // to mean running or terminating pods
 // +optional
 TerminatingAsActive *bool
}
```

### Implementation

Generally, both the Job controller and ReplicaSets utilize `FilterActivePods` in their reconciliation loop.  `FilterActivePods` gets a list of pods that are not terminating.  This KEP will include terminating pods in this list.

```golang
// FilterActivePods returns pods that have not terminated.
func FilterActivePods(pods []*v1.Pod, terminatingPods bool) []*v1.Pod {
 var result []*v1.Pod
 for _, p := range pods {
  if IsPodActive(p) {
   result = append(result, p)
  } else if IsPodTerminating(p) && terminatingPods {
      result = append(result, p)
  } else {
   klog.V(4).Infof("Ignoring inactive pod %v/%v in state %v, deletion time %v",
    p.Namespace, p.Name, p.Status.Phase, p.DeletionTimestamp)
  }
 }
 return result
}

func IsPodTerminating(p *v1.Pod) bool {
 return v1.PodSucceeded != p.Status.Phase &&
  v1.PodFailed != p.Status.Phase &&
  p.DeletionTimestamp != nil
}
```

The Job Controller uses this list to determine if there is a mismatch of active pods between expected values in the JobSpec.  
Including active pods in this list allows the job controller to wait until these terminating pods.

[Filter Active Pods Usage in Job Controller](https://github.com/kubernetes/kubernetes/blob/master/pkg/controller/job/job_controller.go#L749) filters the active pods.

For the Deployment/ReplicaSet, ReplicaSets [filter out active pods](https://github.com/kubernetes/kubernetes/blob/master/pkg/controller/replicaset/replica_set.go#L692).  The implementation for this should include reading the deployment field and setting the replicaset the same field in the replicaset.  
<!--
This section should contain enough information that the specifics of your
change are understandable. This may include API specs (though not always
required) or even code snippets. If there's any ambiguity about HOW your
proposal will be implemented, this is the place to discuss them.
-->

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

##### Unit tests

- `controller_utils`: `April 3rd 2023` - `56.6`
- `replicaset`: `April 3rd 2023` - `78.5`
- `deployment`: `April 3rd 2023` - `66.4`
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

TerminatingAsActive Feature Toggle On:

  1) NonIndexedJob starts pods that takes a while to terminate
  2) Delete pods
  3) Verify that pod creation only occurs once terminating pods are removed

We should test the above with the FeatureToggle off also.

We will add a similar integration test for Deployment:

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

- Job controller includes terminating pods as active
- Deployment strategy optionally includes terminating pods as active
- Unit Tests
- Initial e2e tests
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

###### How can this feature be enabled / disabled in a live cluster?

- [x] Feature gate (also fill in values in `kep.yaml`)
  - Feature gate name: TerminatingAsActive
  - Components depending on the feature gate: kube-controller-manager

###### Does enabling the feature change any default behavior?

Yes, terminating pods are included in the active pod count for `FilterActivePods`.

This means that deployments/Jobs when field is enabled will only create new pods once the existing pods have terminated.

This could potentially make deployments slower.

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

Terminating pods will now be dropped from active list and we will revert to old behavior.  This means that terminating pods will be considered deleted and new pods will be created.

###### Are there any tests for feature enablement/disablement?

Yes. Unit tests will include the fields off/on and verify behavior.

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

If a user terminates pods that are controlled by a deployment/job, then we should wait
until the existing pods are terminated before starting new ones.

We will add e2e test that determine this.

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

NA

###### Are there any missing metrics that would be useful to have to improve observability of this feature?

### Dependencies

This feature is closely related to the 3329-retriable-and-nonretriable-failures but not sure if that is considered a dependency.

###### Does this feature depend on any specific services running in the cluster?

No

### Scalability

Generally, enabling this will slow down rollouts if pods take a long time to terminate.  We would wait
to create new pods until the existing ones are terminated

###### Will enabling / using this feature result in any new API calls?

No

###### Will enabling / using this feature result in introducing new API types?

We add `TerminatingAsActive` to `JobSpec`, `DeploymentStrategy` and `ReplicaSetSpec`.  This is a boolPtr.

###### Will enabling / using this feature result in any new calls to the cloud provider?

No

###### Will enabling / using this feature result in increasing size or count of the existing API objects?

For Job API, we are adding a BoolPtr field named `TerminatingAsActive` which is a boolPtr of 8 bytes.

- API type(s): boolPtr
- Estimated increase in size: 8B

ReplicaSet and Deployment have two additions:

- API type(s): boolPtr
- `DeploymentStrategy` and ReplicaSetSpec
- Estimated increase in size: 16B (2 x 8B)

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

###### Can enabling / using this feature result in resource exhaustion of some node resources (PIDs, sockets, inodes, etc.)?

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

## Drawbacks

Enabling this feature may have rollouts become slower.

## Alternatives

We discussed having this under the PodFailurePolicy but this is a more general idea than the PodFailurePolicy.

## Infrastructure Needed (Optional)

NA
