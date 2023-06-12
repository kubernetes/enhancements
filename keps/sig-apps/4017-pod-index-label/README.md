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
# KEP-4017: Pod Index Label for StatefulSets and Indexed Jobs

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
  - [Test Plan](#test-plan)
      - [Prerequisite testing updates](#prerequisite-testing-updates)
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
  - [Add pod index as annotation](#add-pod-index-as-annotation)
  - [Add pod index as both label and annotation](#add-pod-index-as-both-label-and-annotation)
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
- [X] (R) Design details are appropriately documented
- [X] (R) Test plan is in place, giving consideration to SIG Architecture and SIG Testing input (including test refactors)
  - [ ] e2e Tests for all Beta API Operations (endpoints)
  - [ ] (R) Ensure GA e2e tests meet requirements for [Conformance Tests](https://github.com/kubernetes/community/blob/master/contributors/devel/sig-architecture/conformance-tests.md) 
  - [ ] (R) Minimum Two Week Window for GA e2e tests to prove flake free
- [X] (R) Graduation criteria is in place
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
Currently, StatefulSet pods do not have their pod index as a label or annotation, and
Indexed Jobs only set the pod completion index as an annotation.
This KEP proposes to set the pod index as a label on pods for StatefulSets and Indexed Jobs.

## Motivation

<!--
This section is for explicitly listing the motivation, goals, and non-goals of
this KEP.  Describe why the change is important and the benefits to users. The
motivation section can optionally provide links to [experience reports] to
demonstrate the interest in a KEP within the wider Kubernetes community.

[experience reports]: https://github.com/golang/go/wiki/ExperienceReports
-->

StatefulSets, similar to Indexed Jobs, assign a pod ordinal to each of their pods (this is
analogous to the pod completion index for Indexed Jobs).

Indexed Jobs set the pod index as an annotation, which means it can be utilized by the downward API,
however, it cannot easily be used for selecting pods based on their index or filtering metrics/logs, 
or for doing things like routing network traffic to a pod with a specific index (e.g., index 0). 

StatefulSets include the pod index as part of the pod name. This means that the only way for a 
StatefulSet pod to know its own index is to [parse the pod name](https://github.com/kubernetes/kubernetes/blob/b31774d39b8b493a9468c030e598d8520308443d/pkg/controller/statefulset/stateful_set_utils.go#L63-L75),
which is somewhat hacky solution. Furthermore, the pod index cannot be utilized by the downward API,
nor used for use cases such as filtering metrics/logs or routing traffic to a pod with a specific index.

To address these issues, we propose to set the pod index as a **label** on pods for both StatefulSets and
Indexed Jobs. This will allow the pod index to be utilized via the downward API, so StatefulSet pods
can easily know their own index. In addition, this will allow metrics and logs to more easily be
filtered by pod index, for both StatefulSets and Indexed Jobs.

### Goals

<!--
List the specific goals of the KEP. What is it trying to achieve? How will we
know that this has succeeded?
-->
- Set pod index as label on pods for StatefulSet/Indexed Jobs.
- Adding the label should not be disruptive to existing workloads.

### Non-Goals

<!--
What is out of scope for this KEP? Listing non-goals helps to focus discussion
and make progress.
-->

## Proposal

<!--
This is where we get down to the specifics of what the proposal actually is.
This should have enough detail that reviewers can understand exactly what
you're proposing, but should not include things like API designs or
implementation. What is the desired outcome and how do we measure success?.
The "Design Details" section below is for the real
nitty-gritty.
-->
At a high level, the proposal is to modify the StatefulSet and Job controllers to set the pod index
as a pod label at pod creation time (for jobs, this would only apply to jobs in
Indexed completion mode). The details of this are outlined in the Design Details section below.

StatefulSet pod label: `statefulset.kubernetes.io/pod-index`
Indexed Job pod label: `batch.kubernetes.io/job-completion-index` (same as existing annotation)

### User Stories (Optional)

<!--
Detail the things that people will be able to do if this KEP is implemented.
Include as much detail as possible so that people can understand the "how" of
the system. The goal here is to make this feel real for users without getting
bogged down.
-->


#### Story 1
As a user, I would like to lookup a job's pod logs by its index.

#### Story 2
As a user, I would like to target traffic to a specific pod index (e.g., index 0) in a StatefulSet 
or Indexed Job. Instead of creating a service which matche an entire Job, I'd like to create a
service which matches only the "head" pod, which will be more performant, especially for a large
number of pods.

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

One thing that must be considered is how enabling this new feature will interact with existing workloads. There are a couple of options:

1. Only inject the label on *newly created pods*, so an existing StatefulSet/Indexed Jobs may include pods with the label and some without it. 
This means for the user to utilize the label via the downward API, or to use the label for pod selection, they will need to recreate the StatefulSet so the label is present on all pods.

2. Inject the label only on pods for *newly created StatefulSets/Indexed Jobs*. We can track this by annotating newly created StatefulSets/Indexed Jobs 
to distinguish existing ones from newly created ones. Using this strategy, for a given StatefulSet/IndexedJob, either none of the pods have this label, or all 
of them do, which will provide a more consistent user experience. However, in the case of a cluster downgrade to a version without this feature, new pods 
would start getting created without this label again.

3. Inject the label on *all pods* (pods existing prior to feature enablement and pods created after feature enablement). However, retroactively modifying pods of existing workloads would risk being too disruptive to existing workloads which may have logic depending on pod
labels, so this option should not be considered.

Both options 1 and 2 will not be disruptive to existing workloads. Option 1 is more straightforward and does not risk locking us into adding this somewhat
hacky annotation to StatefulSets/Jobs indefinitely like Option 2 does. On the other hand, outside of the cluster downgrade edge case, Option 2 will
ensure consistency within a single StatefulSet/Indexed Job and therefore a more predictable user experience.

After considering these trade-offs, I propose we move forward with Option 1 for simplicity and to avoid being stuck adding this annotation to 
StatefulSets/Indexed Jobs. In addition, the downside of existing workloads having only a subset of pods with the new label will not cause any serious issues.

## Design Details

<!--
This section should contain enough information that the specifics of your
change are understandable. This may include API specs (though not always
required) or even code snippets. If there's any ambiguity about HOW your
proposal will be implemented, this is the place to discuss them.
-->
The StatefulSet controller will only need a minor update to the [newStatefulSetPod](https://github.com/kubernetes/kubernetes/blob/fb5e9ef3b2a6f2136d54868187431c345e59f55f/pkg/controller/statefulset/stateful_set_utils.go#L458) function, to set the pod ordinal as the label `statefulset.kubernetes.io/pod-index`. This call is downstream from the [newVersionedStatefulSetPod](https://github.com/kubernetes/kubernetes/blob/release-1.27/pkg/controller/statefulset/stateful_set_control.go#LL416C7-L416C7) call, which generates
the StatefulSet pods before creating them as necessary in [CreateStatefulPod](https://github.com/kubernetes/kubernetes/blob/release-1.27/pkg/controller/statefulset/stateful_set_control.go#L433).

Similarly, the Job controller would need to add the completion index as a label [here](https://github.com/kubernetes/kubernetes/blob/release-1.27/pkg/controller/job/job_controller.go#L1480)
where it adds the corresponding annotation.

### Test Plan

<!--
**Note:** *Not required until targeted at a release.*
The goal is to ensure that we don't accept enhancements with inadequate testing.

All code is expected to have adequate tests (eventually with coverage
expectations). Please adhere to the [Kubernetes testing guidelines][testing-guidelines]
when drafting this test plan.

[testing-guidelines]: https://git.k8s.io/community/contributors/devel/sig-testing/testing.md
-->

[X] I/we understand the owners of the involved components may require updates to
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
- `k8s.io/kubernetes/pkg/controller/job`: `05/18/2023` - `90.4%`
- `k8s.io/kubernetes/pkg/controller/statefulset`: `05/18/2023` - `85.7%`

##### Integration tests

<!--
Integration tests are contained in k8s.io/kubernetes/test/integration.
Integration tests allow control of the configuration parameters used to start the binaries under test.
This is different from e2e tests which do not allow configuration of parameters.
Doing this allows testing non-default options and multiple different and potentially conflicting command line options.
-->

<!--
This question should be filled when targeting a release.
For Alpha, describe what tests will be added to ensure proper quality of the enhancement.

For Beta and GA, add links to added tests together with links to k8s-triage for those tests:
https://storage.googleapis.com/k8s-triage/index.html
-->

Unit tests will ensure the new label is correctly added to pods, and
integration tests will verify that the label is only added to pods from newly created StatefulSets and Indexed Jobs, not existing workloads. 

##### e2e tests

<!--
This question should be filled when targeting a release.
For Alpha, describe what tests will be added to ensure proper quality of the enhancement.

For Beta and GA, add links to added tests together with links to k8s-triage for those tests:
https://storage.googleapis.com/k8s-triage/index.html

We expect no non-infra related flakes in the last month as a GA graduation criteria.
-->

E2E tests will not provide any additional coverage that isn't already covered by unit + integration tests, 
since we are simply adding a label, so no e2e tests will be necessary for this change.

### Graduation Criteria

We will release the feature directly in Beta state since there is no benefit in having an alpha release, since we are simply adding a new label so there is very little risk (unlike removing an
existing label which other things may depend on, for example).

#### Beta
- Feature implemented behind the `PodIndexLabel` feature gate.
- Unit and integration tests passing.
- Docs are clear that it is managed by the workload controller(s), and it is NOT guaranteed for every pod.
- Docs are clear about what happens if two pods get the same value (it is set by workload controllers, nothing in the API system will prevent collisions from happening).

#### GA
Fix any potentially reported bugs.

<!--
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

After a user upgrades their cluster to a version which supports this feature (and has the feature gate
enabled) the user will need to redeploy their StatefulSets / Indexed Jobs so that all pods have the pod index label,
since after the upgrade only newly created pods will have this pod index label added. 

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

N/A. This feature doesn't require coordination between control plane components,
the changes to each controller are self-contained. 

If there were version skew between the control plane components and the node components, where the control plane components were at version N where this feature exists, and the node componets were at version N-1 where this feature does not exist, there would be no adverse affects, the new label would simply be added to StatefulSet/Indexed Job pods. 

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
The feature can be safely rolled back.

###### How can this feature be enabled / disabled in a live cluster?

<!--
Pick one of these and delete the rest.

Documentation is available on [feature gate lifecycle] and expectations, as
well as the [existing list] of feature gates.

[feature gate lifecycle]: https://git.k8s.io/community/contributors/devel/sig-architecture/feature-gates.md
[existing list]: https://kubernetes.io/docs/reference/command-line-tools-reference/feature-gates/
-->

- [X] Feature gate (also fill in values in `kep.yaml`)
  - Feature gate name: PodIndexLabel
  - Components depending on the feature gate:
    - StatefulSet controller
    - Job controller
- [ ] Other
  - Describe the mechanism:
  - Will enabling / disabling the feature require downtime of the control
    plane?
  - Will enabling / disabling the feature require downtime or reprovisioning
    of a node?

###### Does enabling the feature change any default behavior?

<!--
Any change of default behavior may be surprising to users or break existing
automations, so be extremely careful here.
-->
Yes - when we start setting a new label, if someone is doing deep-equal comparison, those will start failing.

###### Can the feature be disabled once it has been enabled (i.e. can we roll back the enablement)?

<!--
Describe the consequences on existing workloads (e.g., if this is a runtime
feature, can it break the existing applications?).

Feature gates are typically disabled by setting the flag to `false` and
restarting the component. No other changes should be necessary to disable the
feature.

NOTE: Also set `disable-supported` to `true` or `false` in `kep.yaml`.
-->
Yes. If the feature gate is disabled, the StatefulSet/Job controller will not add the
pod index as a label.

###### What happens if we reenable the feature if it was previously rolled back?

The StatefulSet/Job controller will begin adding the pod index as a label to
pods created while the feature is enabled, and existing pods will be unaffected.

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
We plan to add unit and integration tests. 

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

It will not impact already running workloads.

###### What specific metrics should inform a rollback?

<!--
What signals should users be paying attention to when the feature is young
that might indicate a serious problem?
-->
- Users can monitor queue related metrics (e.g., queue depth and work duration) to make sure they aren't growing.
- For Indexed Jobs, users can also monitor `job_sync_duration_seconds`.
- For StatefulSets: the `kube_statefulset_status_replicas` metric can be monitored against the
`kube_statefulset_replicas` metric to check the expected number of replicas to
the actual number of pods matched by this StatefulSet's selector. If there is
a divergence between these fields during steady state operations, this can
indicate that the number of replicas being created by the StatefulSet do not
match the expected number of replicas.

On a large scale (across a large number of StatefulSets) the distribution of the
ratio of these two metrics should not change when enabling this feature. If this
ratio changes significantly after enabling this feature, it could indicate a problem
and could indicate a rollback is necessary.


###### Were upgrade and rollback tested? Was the upgrade->downgrade->upgrade path tested?

<!--
Describe manual testing that was done and the outcomes.
Longer term, we may want to require automated upgrade/rollback tests, but we
are missing a bunch of machinery and tooling and can't do that now.
-->
It will be tested manually prior to beta launch.

###### Is the rollout accompanied by any deprecations and/or removals of features, APIs, fields of API types, flags, etc.?

<!--
Even if applying deprecation policies, they may still surprise some users.
-->
No.

### Monitoring Requirements

<!--
This section must be completed when targeting beta to a release.

For GA, this section is required: approvers should be able to confirm the
previous answers based on experience in the field.
-->
N/A

###### How can an operator determine if the feature is in use by workloads?

<!--
Ideally, this should be a metric. Operations against the Kubernetes API (e.g.,
checking if there are objects with field X set) may be a last resort. Avoid
logs or events for this purpose.
-->
- Check if StatefulSet pods have the label `statefulset.kubernetes.io/pod-index`.
- Check if Indexed Job pods have the label `batch.kubernetes.io/job-completion-index`.

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
- [X] API .metadata
  - Condition name: 
  - Other field: 
    - `.metadata.labels['statefulset.kubernetes.io/pod-index']` for StatefulSets
    - `.metadata.labels['batch.kubernetes.io/job-completion-index']` for Indexed Jobs
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

These goals will help you determine what you need to measure (SLIs) in the next question.
-->
- Jobs: 99% percentile over day for Job syncs is <= 15s for a client-side 50 QPS
  limit.
- StatefulSets: the ratio of `kube_statefulset_status_replicas`/`kube_statefulset_replicas` should be near 1.0, although as unhealthy replicas are often an application error rather than a problem with the stateful set controller, this will need to be tuned by an operator on a per-cluster basis.

###### What are the SLIs (Service Level Indicators) an operator can use to determine the health of the service?

<!--
Pick one more of these and delete the rest.
-->
Jobs:
  - Metric name: `job_sync_duration_seconds`, `job_sync_total`.
  - Components exposing the metric: `kube-controller-manager`

StatefulSets:
  - Metric name: `statefulset_reconcile_delay`
    - [Optional] Aggregation method: `quantile`
    - Components exposing the metric: `pkg/controller/statefulset`
  - Metric name: `kube_statefulset_replicas`
    - [Optional] Aggregation method: `gauge`
    - Components exposing the metric: `pkg/controller/statefulset`
  - Metric name: `kube_statefulset_status_replicas`
    - [Optional] Aggregation method: `gauge`
    - Components exposing the metric: `pkg/controller/statefulset`
  - Metric name: `kube_statefulset_ordinals_start`
    - [Optional] Aggregation method: `gauge`
    - Components exposing the metric: `pkg/controller/statefulset`

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
No.

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
No.

###### Will enabling / using this feature result in introducing new API types?

<!--
Describe them, providing:
  - API type
  - Supported number of objects per cluster
  - Supported number of objects per namespace (for namespace-scoped objects)
-->
No.

###### Will enabling / using this feature result in any new calls to the cloud provider?

<!--
Describe them, providing:
  - Which API(s):
  - Estimated increase:
-->
No.

###### Will enabling / using this feature result in increasing size or count of the existing API objects?

<!--
Describe them, providing:
  - API type(s):
  - Estimated increase in size: (e.g., new annotation of size 32B)
  - Estimated amount of new objects: (e.g., new Object X for every existing Pod)
-->
New pod label of size 34B plus value of size N where N is the number of digits in the pod ordinal. 
Worst case for N would be the max number of pods per cluster. Per the docs on [large clusters](https://kubernetes.io/docs/setup/best-practices/cluster-large/) this is 150,000 (6 digits). So max label size would be 34 + 6 = 40B.

###### Will enabling / using this feature result in increasing time taken by any operations covered by existing SLIs/SLOs?

<!--
Look at the [existing SLIs/SLOs].

Think about adding additional work or introducing new steps in between
(e.g. need to do X to start a container), etc. Please describe the details.

[existing SLIs/SLOs]: https://git.k8s.io/community/sig-scalability/slos/slos.md#kubernetes-slisslos
-->
No.

###### Will enabling / using this feature result in non-negligible increase of resource usage (CPU, RAM, disk, IO, ...) in any components?

<!--
Things to keep in mind include: additional in-memory state, additional
non-trivial computations, excessive access to disks (including increased log
volume), significant amount of data sent and/or received over network, etc.
This through this both in small and large cases, again with respect to the
[supported limits].

[supported limits]: https://git.k8s.io/community//sig-scalability/configs-and-limits/thresholds.md
-->
No.

###### Can enabling / using this feature result in resource exhaustion of some node resources (PIDs, sockets, inodes, etc.)?

<!--
Focus not just on happy cases, but primarily on more pathological cases
(e.g. probes taking a minute instead of milliseconds, failed pods consuming resources, etc.).
If any of the resources can be exhausted, how this is mitigated with the existing limits
(e.g. pods per node) or new limits added by this KEP?

Are there any tests that were run/should be run to understand performance characteristics better
and validate the declared limits?
-->
No.

### Troubleshooting

<!--
This section must be completed when targeting beta to a release.

For GA, this section is required: approvers should be able to confirm the
previous answers based on experience in the field.

The Troubleshooting section currently serves the `Playbook` role. We may consider
splitting it into a dedicated `Playbook` document (potentially with some monitoring
details). For now, we leave it here.
-->
N/A

###### How does this feature react if the API server and/or etcd is unavailable?

N/A

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
N/A

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
- 2023-05-17: KEP published

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
### Add pod index as annotation
This was discussed but there are use cases where a label is required (filtering metrics, logs).
See [discussion link](https://github.com/kubernetes/enhancements/issues/4017).

### Add pod index as both label and annotation
This was discussed but there are no concrete use cases that a label cannot fulfill and an annotation is
required. See [discussion link](https://github.com/kubernetes/enhancements/issues/4017).

## Infrastructure Needed (Optional)

<!--
Use this section if you need things from the project/SIG. Examples include a
new subproject, repos requested, or GitHub details. Listing these here allows a
SIG to get the process for these resources started right away.
-->