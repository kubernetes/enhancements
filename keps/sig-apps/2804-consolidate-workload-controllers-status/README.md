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
# KEP-2804: Consolidate Workload controllers life cycle state

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
  - [Overview of all conditions](#overview-of-all-conditions)
  - [Proposed Conditions](#proposed-conditions)
    - [Progressing](#progressing)
    - [Complete](#complete)
    - [Failed](#failed)
    - [Available](#available)
    - [Batch Workloads Conditions: Waiting &amp; Running](#batch-workloads-conditions-waiting--running)
  - [Notes/Constraints/Caveats (Optional)](#notesconstraintscaveats-optional)
  - [Risks and Mitigations](#risks-and-mitigations)
- [Design Details](#design-details)
  - [Test Plan](#test-plan)
  - [Graduation Criteria](#graduation-criteria)
    - [Alpha](#alpha)
    - [Alpha -&gt; Beta Graduation](#alpha---beta-graduation)
    - [Beta -&gt; GA Graduation](#beta---ga-graduation)
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

The main goal of this enhancement is to compare all the workload conditions and consolidate the workload controller lifecycle
state. The secondary goal is to identify and expose other conditions that could bring benefit to the users.
This includes defining conditions (Waiting, Running, Progressing, Available) for:
- Deployment
- DaemonSet
- ReplicaSet & ReplicationController
- StatefulSet
- Job

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

## Motivation

<!--
This section is for explicitly listing the motivation, goals, and non-goals of
this KEP.  Describe why the change is important and the benefits to users. The
motivation section can optionally provide links to [experience reports] to
demonstrate the interest in a KEP within the wider Kubernetes community.

[experience reports]: https://github.com/golang/go/wiki/ExperienceReports
-->

Today only deployment controller has a [status](https://kubernetes.io/docs/concepts/workloads/controllers/deployment/#deployment-status) to fully reflect the state during its lifecycle.
This enhancement extends the scope of those and other conditions to other controllers (DaemonSet, Job, ReplicaSet & ReplicationController, StatefulSet).

### Goals

The current status of a workload can be depicted via its conditions. It can be a subset of:
- Progressing
- Available
- ReplicaFailure
- Suspended
- Complete
- Failed (to progress)
- Waiting (Job only)
- Running (Job only)

Workload controllers should have above conditions (when applicable) to reflect their states.

### Non-Goals

<!--
What is out of scope for this KEP? Listing non-goals helps to focus discussion
and make progress.
-->

- Modifying the existing states of deployment controller
- Changing the definition of States
- Introduce new api for existing conditions
- To properly implement Progressing condition, `.spec.progressDeadlineSeconds` field has to be introduced in 
  DaemonSet and StatefulSet to describe the time when the controllers should declare the workload as `failed`.

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
As an end-user of Kubernetes, I'd like all my workload controllers to have consistent states

#### Story 2
As an developer building Kubernetes Operators, I'd like all my operators deployed to have
consistent states.


### Overview of all conditions

The following table gives an overview on what conditions each of the workload resources support.

|                                    | Progressing | Available |  ReplicaFailure | Suspended | Failed | Complete |
| ---------------------------------  | ----------- | --------- | --------------- | --------- | ------ | -------- |
| ReplicaSet & ReplicationController | -           | -         |  failed to create / delete pod (FailedCreate, FailedDelete)  | -         | -           | -        |
| Deployment                         | True when scaling replicas / creating-updating new ReplicaSet / successfully finished progressing (Pods ready or available for MinReadySeconds). False when failed creating ReplicaSet / reached progressDeadlineSeconds. Unknown when rollout paused | True if if required number of replicas is available (takes MaxSurge and MaxUnavailable into account) | failure propagated from new or old ReplicaSet | -         | -*        | -*          |
| StatefulSet                        | -           | -         | -               | -         | -      | -        |
| DaemonSet                          | -           | -         | -               | -         | -      | -        |
| Job                                | -           | -         | -               | True / False when suspended / resumed | failed execution  (BackoffLimitExceeded, DeadlineExceeded)| completed execution |
| CronJob**                          | -           | -         | -               | -         | -      | -        |

**\* Success of the rollout is instead represented by a Progressing condition (status and reason)**

**\*\* CronJob does not even have Conditions field in its Status**

### Proposed Conditions

<!--
What are the caveats to the proposal?
What are some important details that didn't come across above?
Go in to as much detail as necessary here.
This might be a good place to talk about core concepts and how they relate.
-->
We are proposing 4 new conditions to be added to the following controllers:
- Available (DaemonSet, ReplicaSet & ReplicationController, StatefulSet)
- Progressing (DaemonSet, StatefulSet)
- Waiting (Job)
- Running (Job)

The definitions for Progressing condition (including Failed/Complete) are similar to what we have for [Deployment controller](https://kubernetes.io/docs/concepts/workloads/controllers/deployment/#deployment-status).


The following table is indicating what conditions are currently available (`X`) and what conditions will be added (`A`).

|                                    | Waiting | Running | Progressing | Available |  ReplicaFailure | Suspended | Failed | Complete |
| ---------------------------------  | --------| ------- | ----------- | --------- | --------------- | --------- | ------ | -------- |
| ReplicaSet & ReplicationController | -       | -       | -           | A         | X               | -         | -      | -        |
| Deployment                         | -       | -       | X           | X         | X               | -         | -      | -        |
| StatefulSet                        | -       | -       | A           | A         | -               | -         | -      | -        |
| DaemonSet                          | -       | -       | A           | A         | -               | -         | -      | -        |
| Job                                | A       | A       | -           | -         | -               | X         | X      | X        |
| CronJob                            | -       | -       | -           | -         | -               | -         | -      | -        |

#### Progressing
Kubernetes marks a DaemonSet or Stateful as `progressing` when:
- The DaemonSet or StatefulSet is created
- The DaemonSet or StatefulSet is scaling up or scaling down
- New DaemonSet or StatefulSet pods become Ready or available

#### Complete
Kubernetes marks a DaemonSet, ReplicaSet or Stateful as `complete` when it has the following characteristics:

- All of the replicas/pods associated with the DaemonSet or StatefulSet have been updated to the latest version you've specified, meaning any updates you've requested have been completed.
- All of the replicas/pods associated with the DaemonSet, ReplicaSet or StatefulSet are available.
- No old or mischeduled replicas/pods for the DaemonSet, ReplicaSet or Stateful are running.

#### Failed
In order to introduce this condition we need to come up with a new field called `.spec.progressDeadlineSeconds` which denotes the number of seconds the controller waits before indicating(in the workload controller status) that the controller progress has stalled.

There are many factors that can cause failure to happen like:
- Insufficient quota
- Readiness probe failures
- Image pull errors
- Failed to create/delete pod

See the [Kubernetes API Conventions](https://git.k8s.io/community/contributors/devel/sig-architecture/api-conventions.md#typical-status-properties) for more information on status conditions

Because of the number of changes that are involved as part of this effort, we are thinking of a phased approach where we introduce these conditions to DaemonSet controller behind a featuregate flag first in one release and then make similar changes to ReplicaSet and StatefulSet controller.
This also needs some code refactoring of existing conditions for Deployment controller.

#### Available
Kubernetes marks a ReplicaSet, StatefulSet as `available` when number of available replicas reaches number of replicas.
- This could be confusing in ReplicaSets a bit since Deployment could get available sooner than its ReplicaSet due `maxUnavailable`.
- Available replicas is alpha feature guarded by StatefulSetMinReadySeconds gate in StatefulSets, but the value defaults to ReadyReplicas when the feature gate is disabled so using it shouldn't be an issue.

Kubernetes marks DaemonSet as `available` when `numberUnavailable` or `desiredNumberScheduled - numberAvailable` is zero.

#### Batch Workloads Conditions: Waiting & Running

Batch workloads behaviour does not properly map to the other workloads that are expected to be always running (eg. `Progressing` condition and its behaviour).
- Jobs are indicating a `Failed`/`Complete` state in a standalone condition compared to `Progressing` condition in other workloads.
- `.spec.activeDeadlineSeconds` variable, is similar to `progressDeadlineSeconds`, but does not have a default value.
  It also resets on suspension, so its behaviour is a bit different.


Kubernetes marks a Job as `waiting` if one of the following conditions is true:

- There are no Pods with phase `Running` and there is at least one Pod with phase `Pending`.
- The Job is suspended.

Kubernetes marks a Job as `running` if there is at least one Pod with phase `Running`.

This KEP does not introduce CronJob conditions as it is difficult to define conditions that would describe CronJob behaviour in useful manner.
In case the user is interested if there are any running Jobs, `.status.active` field should be sufficient.

### Notes/Constraints/Caveats (Optional)

As observed in some issues (https://github.com/kubernetes/website/pull/31226) and talking to the users, there is a misunderstanding about the meaning of the `Progressing` condition. These include:
- Thinking that the `Progressing` condition reflects the state of the current Deployment instead of the last rollout. Which includes expectation that the `Progressing` condition will keep checking availability of replicas and revert to `progressing`/`failed` state even after the `complete` state is reached. And that the progressing condition will thus also reflect any changes in scaling.
- Confusion that ProgressDeadlineExceeded does not occur after the Deployment rollout completes when the availability of pods degrades before the  `.spec.progressDeadlineSeconds` times out.

To summarize, the name of the `Progressing` condition doesn't indicate its true meaning that its main responsibility is the indication of rollouts, and it confuses the users.

### Risks and Mitigations

<!--
What are the risks of this proposal, and how do we mitigate? Think broadly.
For example, consider both security and how this will impact the larger
Kubernetes ecosystem.

How will security be reviewed, and by whom?

How will UX be reviewed, and by whom?

Consider including folks who also work outside the SIG or subproject.
-->
We are proposing a new field called `.spec.progressDeadlineSeconds` to DaemonSet and StatefulSet whose default value will be set to the max value of int32 (i.e. 2147483647) by default, which means "no deadline".
In this mode, DaemonSet and StatefulSet controllers will behave exactly like its current behavior but with no `Failed` mode as they're either `Progressing` or `Complete`.
This is to ensure backward compatibility with current behavior. We will default to reasonable values for the controllers in a future release.
Since a DaemonSet can make progress no faster than "healthy but not ready nodes", the default value for `progressDeadlineSeconds` can be set to 30 minutes but this value can vary depending on the node count in the cluster.
The value for StatefulSet can be longer than 10 minutes since it also involves provisioning storage and binding. This default value can be set to 15 minutes in case of StatefulSet.

It is possible that we introduce a bug in the implementation. The bug can cause:
- DaemonSet and StatefulSet controllers can be marked `Failed` even though rollout is in progress
- The states could be misrepresented, for example a DaemonSet or StatefulSet can be marked `Progressing` when actually it is complete

The mitigation currently is that these features will be in Alpha stage behind featuregates for people to try out and give feedback. In Beta phase when
these are enabled by default, people will only see issues or bugs when `progressDeadlineSeconds` is set to something greater than 0 and we choose default values for that field.
Since people would have tried this feature in Alpha, we would have had time to fix issues.
The featuregates we use are `DaemonSetConditions` for DaemonSet controller,  `StatefulSetConditions` for StatefulSet controller.




## Design Details

<!--
This section should contain enough information that the specifics of your
change are understandable. This may include API specs (though not always
required) or even code snippets. If there's any ambiguity about HOW your
proposal will be implemented, this is the place to discuss them.
-->

### Test Plan
Unit and E2E tests will be added to cover the
API validation, behavioral change of DaemonSet and StatefulSet with feature gate enabled and disabled.

- Validating all possible states of old and new conditions. Checking that the changes in underlying Pod statuses correspond to the conditions.
- Testing `progressDeadlineSeconds` and feature gates.

<!--
**Note:** *Not required until targeted at a release.*

Consider the following in developing a test plan for this enhancement:
- Will there be e2e and integration tests, in addition to unit tests?
- How will it be tested in isolation vs with other components?

No need to outline all of the test cases, just the general strategy. Anything
that would count as tricky in the implementation, and anything particularly
challenging to test, should be called out.

All code is expected to have adequate tests (eventually with coverage
expectations). Please adhere to the [Kubernetes testing guidelines][testing-guidelines]
when drafting this test plan.

[testing-guidelines]: https://git.k8s.io/community/contributors/devel/sig-testing/testing.md
-->

### Graduation Criteria

#### Alpha
- Complete feature behind featuregates
- Have proper unit, integration and e2e tests 

#### Alpha -> Beta Graduation
- Gather feedback from end users
- Tests are in Testgrid and linked in KEP

#### Beta -> GA Graduation
- 2 examples of end users using this field
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
- Upgrades
 When upgrading from a release without this feature, to a release with 
 `progressDeadlineSeconds`, we will set `progressDeadlineSeconds` to max value of int32 (i.e. 2147483647). This would give users 
 the same default behavior.
- Downgrades
 When downgrading from a release with this feature, to a release without 
 `progressDeadlineSeconds`, there are two cases
  - If `progressDeadlineSeconds` is greater than 0 -- in this case kube-apiserver
    clears the `progressDeadlineSeconds` and the existing StatefulSet and DaemonSet controllers wouldn't honor 
    `progressDeadlineSeconds` which is expected.
  - If `progressDeadlineSeconds` is equal to 0 -- in this case user wont see any 
    difference in behavior.

We will ensure that the `progressDeadlineSeconds` field is properly validated
before persisting. The validation includes checking for positive number for the `progressDeadlineSeconds` field.

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

None identified yet.

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
-->

- [x] Feature gate (also fill in values in `kep.yaml`)
  - Feature gate name: DaemonSetConditions, StatefulSetConditions
  - Components depending on the feature gate:
    - kube-controller-manager
    - kube-apiserver

###### Does enabling the feature change any default behavior?
No. The default behavior won't change.
The default values for newly introduced `progressDeadlineSeconds` fields might change when graduating it after getting feedback from users.
Only new conditions will be added with no effect on existing conditions.

<!--
Any change of default behavior may be surprising to users or break existing
automations, so be extremely careful here.
-->

###### Can the feature be disabled once it has been enabled (i.e. can we roll back the enablement)?
 Yes. Using the featuregates is the only way to enable/disable this feature

<!--
Describe the consequences on existing workloads (e.g., if this is a runtime
feature, can it break the existing applications?).

NOTE: Also set `disable-supported` to `true` or `false` in `kep.yaml`.
-->

###### What happens if we reenable the feature if it was previously rolled back?

The DaemonSet, StatefulSet controller starts respecting the `progressDeadlineSeconds` again.

###### Are there any tests for feature enablement/disablement?

Yes, unit, integration and e2e tests for feature enabled, disabled

<!--
The e2e framework does not currently support enabling or disabling feature
gates. However, unit tests in each component dealing with managing data, created
with and without the feature, are necessary. At the very least, think about
conversion tests if API types are being modified.
-->

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
Yes
  - Update of DaemonSet, Job, ReplicaSet, ReplicationController, StatefulSet status
  - Controllers could potentially make additional update calls when syncing the resources
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

No.
<!--
Describe them, providing:
  - API type
  - Supported number of objects per cluster
  - Supported number of objects per namespace (for namespace-scoped objects)
-->

###### Will enabling / using this feature result in any new calls to the cloud provider?

No.
<!--
Describe them, providing:
  - Which API(s):
  - Estimated increase:
-->

###### Will enabling / using this feature result in increasing size or count of the existing API objects?
Yes.
API type(s): DaemonSet, Deployment, Job, ReplicaSet, ReplicationController, StatefulSet
  - Estimated increase in size:
    - New field in DaemonSet and StatefulSet spec about 4 bytes
    - Add new conditions in Deployment, DaemonSet, Job, ReplicaSet, ReplicationController, StatefulSet status
<!--
Describe them, providing:
  - API type(s):
  - Estimated increase in size: (e.g., new annotation of size 32B)
  - Estimated amount of new objects: (e.g., new Object X for every existing Pod)
-->

###### Will enabling / using this feature result in increasing time taken by any operations covered by existing SLIs/SLOs?

TBD.
<!--
Look at the [existing SLIs/SLOs].

Think about adding additional work or introducing new steps in between
(e.g. need to do X to start a container), etc. Please describe the details.

[existing SLIs/SLOs]: https://git.k8s.io/community/sig-scalability/slos/slos.md#kubernetes-slisslos
-->

###### Will enabling / using this feature result in non-negligible increase of resource usage (CPU, RAM, disk, IO, ...) in any components?

No.
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
Adds more complexity to Deployment, DaemonSet, Job, ReplicaSet, ReplicationController, StatefulSet controllers in terms of checking conditions and updating the conditions continuously.

<!--
Why should this KEP _not_ be implemented?
-->

## Alternatives
Continue to check AvailableReplicas, Replicas and other fields instead of having explicit conditions. This is not always foolproof and can cause problems.
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
