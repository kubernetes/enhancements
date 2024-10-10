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

<!--
 Previously, the KEP had the number 3280, but since the original author no longer has the time to maintain it and the new author unable to modify the original KEP issue, we need to create a new issue and modify the KEP number to allow the new author to track and update everything about this KEP.
-->
# KEP-4664: Guarantee PodDisruptionBudget When Preemption Happens

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
  - [PreemptionPolicy vs AllowDisruptionByPriorityGreaterThanOrEqual](#preemptionpolicy-vs-allowdisruptionbyprioritygreaterthanorequal)
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

This design proposal suggests adding a field `AllowDisruptionByPriorityGreaterThanOrEqual` in the `PriorityClass` API 
to explicitly indicate  that `PodDisruptionBudget` of the pods corresponding to this priorty class can only be 
violated by pods with the priority value greater than or equal to the value of `AllowDisruptionByPriorityGreaterThanOrEqual` 
during the scheduler preemption process, this proposal allows cluster administrators to define PriorityClasses that restrict 
PDB violations during preemption to satisfy the needs for high availability of services during scheduler preemption due to 
some reasons such as high availability of services(https://github.com/kubernetes/kubernetes/issues/91492#issuecomment-1029484252). 

## Motivation

<!--
This section is for explicitly listing the motivation, goals, and non-goals of
this KEP.  Describe why the change is important and the benefits to users. The
motivation section can optionally provide links to [experience reports] to
demonstrate the interest in a KEP within the wider Kubernetes community.

[experience reports]: https://github.com/golang/go/wiki/ExperienceReports
-->

`PodDisruptionBudget` (PDB) is used to limit the number of concurrent disruptions that your application experiences, 
allowing for high availability. Users can set the field `.spec.maxUnavailable` or `.spec.minAvailable` to declare 
the current minimum availability or maximum unavailability to be maintained after eviction. 

However, there is currently an issue where the kube-scheduler does not strictly guarantee PDBs during the preemption 
process. The scheduler supports PDBs when preempting pods, but the adherence to PDBs is best effort. The scheduler 
attempts to select victims whose PDBs are not violated during preemption, but if no such victims are found, preemption 
will still take place, resulting in the removal of lower-priority pods despite their PDBs being violated.

PodDisruptionBudgets (PDBs) are frequently used for stability and the possibility of violating a PDB during the preemption 
process is not acceptable for certain users. As such, it is beneficial to provide users with the option to choose if they 
want the PDB to be guaranteed during preemption or not.

### Goals

<!--
List the specific goals of the KEP. What is it trying to achieve? How will we
know that this has succeeded?
-->
- Provide an option to the cluster administrators to configure whether the scheduler needs to make `PodDisruptionBudget` 
  guaranteed when preemption happens.

### Non-Goals

<!--
What is out of scope for this KEP? Listing non-goals helps to focus discussion
and make progress.
-->
- Let the application developers influence preemption behavior directly.

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
User deployed a service in a cluster and to ensure its high availability, 
User created PDB for this deployment and set `.spec.minAvailable` to 3. User wanted the PDB to be 
guaranteed even in case of scheduler preemption.

#### Story 2
User created a Tensorflow distributed job that requires a minimum of 5 workers running.
User created PDB for the job and set `.spec.minAvailable` to 5. User wanted the PDB to be 
guaranteed even in case of scheduler preemption to ensure the stability of the whole job.

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

If a user sets a PodDisruptionBudget (PDB) for some low-priority pods and sets the `AllowDisruptionByPriorityGreaterThanOrEqual` 
to `PriorityClass`, high-priority (less than the value of allowDisruptionByPriorityGreaterThanOrEqual) pods will not be able 
to violate the PDB and preempt these pods during the scheduling process. This may result in high-priority (less than the value of allowDisruptionByPriorityGreaterThanOrEqual) pods being unable to schedule while low-priority pods continue to run normally. 

Although the above situations may arise, due to the fact that PriorityClass is created and managed by 
cluster administrators with no permission for application owners to perform actions, administrators 
are able to uniformly configure according to the requirements. Additionally, implementation will 
include the addition of additional logging or event descriptions to clearly inform the user of the 
reason why preemption did not occur.

## Design Details

<!--
This section should contain enough information that the specifics of your
change are understandable. This may include API specs (though not always
required) or even code snippets. If there's any ambiguity about HOW your
proposal will be implemented, this is the place to discuss them.
-->

In order to address the issue mentioned above, a new field `AllowDisruptionByPriorityGreaterThanOrEqual` will 
be added to `PriorityClass`. Users will be able to set this field to indicate that the PodDisruptionBudget of 
the pods associated with this priority class can only be violated during scheduler preemption by other pods 
with a priority value greater than or equal to AllowDisruptionByPriorityGreaterThanOrEqual. At the same time, to 
prevent situations where core components or necessary add-ons cannot be scheduled due to the inability to violate 
PDBs, the value of `AllowDisruptionByPriorityGreaterThanOrEqual` cannot be greater than the priority value of 
`system-cluster-critical` and `system-node-critical.priority`.

```go
type PriorityClass struct {
  metav1.TypeMeta
  metav1.ObjectMeta
  Value int32
  GlobalDefault bool
  Description string
  PreemptionPolicy *core.PreemptionPolicy

  // AllowDisruptionByPriorityGreaterThanOrEqual indicates that a PodDisruptionBudget set for pods associated 
  // with this priority class can only be violated by pods with a priority value greater than or equal to 
  // AllowDisruptionByPriorityGreaterThanOrEqual during a preemption process. The value of AllowDisruptionByPriorityGreaterThanOrEqual 
  // cannot be greater than the priority value of system-cluster-critical or system-node-critical.
  // A null value indicates that the PodDisruptionBudget is allow to be disrupted by any other pods with a higher priority.
  // +optional
  AllowDisruptionByPriorityGreaterThanOrEqual *int32
}
```

The `AllowDisruptionByPriorityGreaterThanOrEqual` field in PodSpec will be populated during pod admission, 
similarly to how the PriorityClass Value is populated. Storing the `AllowDisruptionByPriorityGreaterThanOrEqual`
field in the pod spec has several benefits:

1. The scheduler does not need to be aware of PiorityClasses, as all relevant information is in the pod.
2. Mutating PriorityClass objects does not impact existing pods.

```go
// PodSpec is a description of a pod.
type PodSpec struct {
  PriorityClassName string
  Priority *int32
  PreemptionPolicy *PreemptionPolicy

+ // AllowDisruptionByPriorityGreaterThanOrEqual indicates that a PodDisruptionBudget set for pods associated 
+ // with this priority class can only be violated by pods with a priority value greater than or equal to 
+ // AllowDisruptionByPriorityGreaterThanOrEqual during a preemption process. The value of AllowDisruptionByPriorityGreaterThanOrEqual 
+ // cannot be greater than the priority value of system-cluster-critical or system-node-critical. When Priority Admission Controller is 
+ // enabled, it prevents users from setting this field. The admission controller populates this field from PriorityClassName.
+ // A null value indicates that the PodDisruptionBudget is allow to be disrupted by any other pods with a higher priority.
+ // +optiona
+ AllowDisruptionByPriorityGreaterThanOrEqual *int32
}
```

The following is an example of a `PriorityClass` where the user sets the `allowDisruptionByPriorityGreaterThanOrEqual` as 1000
indicating that the pods corresponding to this priorty class can only be violated by the pods with a value of priority 
greater than or equal to 1000 during the scheduler preemption process

```yaml
apiVersion: scheduling.k8s.io/v1
kind: PriorityClass
metadata:
  name: low-priority
value: 100
globalDefault: false
description: "This priority class should be used for XYZ service pods only."
allowDisruptionByPriorityGreaterThanOrEqual: 1000
```

The scheduler plugin `defaultpreemption` needs to check the value set in the `AllowDisruptionByPriorityGreaterThanOrEqual` field 
when selecting victims.
- if the priority of the preemptor is greater than or equal to the value of `AllowDisruptionByPriorityGreaterThanOrEqual` in victim pod, 
  the implementation will remain consistent with the existing behavior, meaning that the scheduler will try to select victims whose PDBs 
  are not violated by preemption, but if no such victims are found, preemption will still happen and lower priority pods will be preempted 
  via the `/evictions` endpoint despite their PDBs being violated. If the `/eviction` endpoint returns a response `429 Too Many Requests`, 
  the scheduler will fallback to deletion as an alternative.
- if the priority of the preemptor is less than the value of `AllowDisruptionByPriorityGreaterThanOrEqual` in victim pod, 
  the scheduler will check if the victim' PDBs will be violated when selecting victims
  - if violate the victims' PDBs, this victim will not be selected as candidates.
  - if not violate the victims' PDBs, scheduler will preempt this pod via the `/evictions` endpoint. 
    - If it responds `200 OK`, it means the eviction is allowed, and the victim is deleted, similar to sending a DELETE request to the Pod URL. 
    - If it responses `429 Too Many Requests`, the scheduler will output an error log and choose another victim among the candidate victims to preempt 
    until it succeeds or there are no more candidates.

### PreemptionPolicy vs AllowDisruptionByPriorityGreaterThanOrEqual

The `PreemptionPolicy` is used to describe the behavior of the pods associated with the PriorityClass during preemption as the preemptor. 
The `AllowDisruptionByPriorityGreaterThanOrEqual` is used to describe the policy of the pods associated with the PriorityClass during 
preemption as the victim. During a preemption process, if the preemptor is configured with the `PreemptionPolicy` and the victim is 
configured with the `AllowDisruptionByPriorityGreaterThanOrEqual`, the `AllowDisruptionByPriorityGreaterThanOrEqual` takes priority over 
the `PreemptionPolicy`.

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

- `pkg/scheduler/apis/config/v1`: `2023-01-19` - `83.9%`
- `pkg/scheduler`: `2023-01-19` - `77.1%`
- `k8s.io/kubernetes/pkg/scheduler/framework/plugins/defaultpreemption`: `2023-01-19` - `85.4%`

##### Integration tests

<!--
This question should be filled when targeting a release.
For Alpha, describe what tests will be added to ensure proper quality of the enhancement.

For Beta and GA, add links to added tests together with links to k8s-triage for those tests:
https://storage.googleapis.com/k8s-triage/index.html
-->

- These cases will be added in the existed integration tests:
  - Feature gate enable/disable tests
  - During scheduling, `AllowDisruptionByPriorityGreaterThanOrEqual` in `PriorityClass` works as expected
  - Verify no significant performance degradation

- `k8s.io/kubernetes/kubernetes/test/integration/scheduler/preemption/preemption_test.go`: https://storage.googleapis.com/k8s-triage/index.html?test=TestPreemption
- `k8s.io/kubernetes/test/integration/scheduler_perf/scheduler_perf_test.go`: https://storage.googleapis.com/k8s-triage/index.html?test=BenchmarkPerfScheduling

##### e2e tests

<!--
This question should be filled when targeting a release.
For Alpha, describe what tests will be added to ensure proper quality of the enhancement.

For Beta and GA, add links to added tests together with links to k8s-triage for those tests:
https://storage.googleapis.com/k8s-triage/index.html

We expect no non-infra related flakes in the last month as a GA graduation criteria.
-->

- These cases will be added in the existed e2e tests in `k8s.io/kubernetes/kubernetes/test/e2e/scheduling/preemption.go`
  - Feature gate enable/disable tests
  - During scheduling, `AllowDisruptionByPriorityGreaterThanOrEqual` in `PriorityClass` works as expected

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
- Feature is enabled by default
- Benchmark tests passed, and there is no performance degradation.
- Update documents to reflect the changes.

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

In the event of an upgrade, kube-apiserver will start to accept and store the field `AllowDisruptionByPriorityGreaterThanOrEqual` in `PriorityClass` and `Pod`.
In the event of a downgrade, kube-scheduler will ignore `AllowDisruptionByPriorityGreaterThanOrEqual` in `PriorityClass` and `Pod` even if it was set.

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
N/A

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
  - Feature gate name: `DisruptionPolicyInPriorityClass`
  - Components depending on the feature gate: `kube-scheduler`, `kube-apiserver`

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
kube-apiserver and kube-scheduler with feature-gate off.
One caveat is that PriorityClasses and Pods that used the feature will continue to have the 
field `AllowDisruptionByPriorityGreaterThanOrEqual` set in `PriorityClass` even after disabling 
the feature gate, however kube-scheduler will not take the field into account.

###### What happens if we reenable the feature if it was previously rolled back?
1. The newly created PriorityClasses and Pods will contain the field `AllowDisruptionByPriorityGreaterThanOrEqual`.
2. The scheduler will check the value in the field `AllowDisruptionByPriorityGreaterThanOrEqual` in `Pod` 
   if preemption occurs during scheduling

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
No. The unit tests that are exercising the `switch` of feature gate itself will be added.

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
- 2023-01-19: Initial KEP
- 2023-01-28: Move the responsibilities from `AllowDisruptionByPriorityGreaterThanOrEqual` to `PriorityClass`
- 2024-05-25: Restart KEP. Update design details.

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
1. Add `PreemptLowerPriorityWithoutViolatePDB` as an option in the `PreemptionPolicy` of the preempter. 
   When the preemptor is set to `PreemptLowerPriorityWithoutViolatePDB`, the pods that would violate 
   PDBs will be excluded when selecting victims. However, if we want to guarantee some pods' PDBs, 
   we need to modify the `PreemptionPolicy` for all other pods to `PreemptLowerPriorityWithoutViolatePDB`. 
   The cost of this operation may be relatively large.

2. A field `PreemptionPolicy` is added to the `PodDisruptionBudget` (PDB) API to indicate whether or not to 
   guarantee the `PodDisruptionBudget` during the scheduler preemption process. Two simple policies are provided, 
   `PreferNotPreempted`, which indicates that the scheduler will try to avoid violating the PDB during preemption, 
   but it cannot be guaranteed, and `RequiredNotPreempted`, which indicates that the PDB will not be violated 
   during scheduler preemption. And, if the preempter has a priority ClassName of `system-cluster-critical` or 
   `system-node-critical`, it may still potentially violate the victim's PDB. But, there is a potential conflict 
   between the creators of PDB and PriorityClass, who may have different priorities (cluster scope and namespace scope), 
   which may result in high-priority pods being blocked from preemption in the cluster.
   
3. Add a new field in the args of preemption plugins to identify if all or some preemptions filtered by Selector 
   cannot violate PDBs. It's cluster-scope or profile-scope. And also to ensure the security and stability of the 
   cluster, we can also add a list called `PriorityClassesAllowViolatePDB` in the configuration to identify that when 
   `PreemptLowerPriorityWithoutViolatePDB` is set to true, the pods with these priority classes can also preempt other 
   pods in violation of the PDBs like `system-cluster-critical` or `system-node-critical` or other priority classes 
   created by users. However, it's too mandatory and inflexible to set it in cluster-scope. And also, schedulers need 
   to restart when we want to update the args. 

## Infrastructure Needed (Optional)

<!--
Use this section if you need things from the project/SIG. Examples include a
new subproject, repos requested, or GitHub details. Listing these here allows a
SIG to get the process for these resources started right away.
-->
