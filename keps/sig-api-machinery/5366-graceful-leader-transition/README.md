<!--
**Note:** When your KEP is complete, all of these comment blocks should be removed.

To get started with this template:

- [x] **Pick a hosting SIG.**
  Make sure that the problem space is something the SIG is interested in taking
  up. KEPs should not be checked in without a sponsoring SIG.
- [x] **Create an issue **in** kubernetes/enhancements**
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
- [x] **Merge early and iterate.**
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
# KEP-5366: Graceful Leader Transition

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
  - [Risks and Mitigations](#risks-and-mitigations)
- [Design Details](#design-details)
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
- [Future Work (Stories)](#future-work-stories)
    - [Story 1](#story-1)
    - [Story 2](#story-2)
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
  - [x] (R) [all GA Endpoints](https://github.com/kubernetes/community/pull/1806) must be hit by [Conformance Tests](https://github.com/kubernetes/community/blob/master/contributors/devel/sig-architecture/conformance-tests.md) 
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

This proposal outlines a plan to modify the leader election mechanism for key
Kubernetes components (kube-scheduler, kube-controller-manager,
cloud-controller-manager). The goal is to enable these components to gracefully
release the leader lock and transition back to a follower state without a full
process restart. This change will be introduced behind a new feature gate.

## Motivation

<!--
This section is for explicitly listing the motivation, goals, and non-goals of
this KEP.  Describe why the change is important and the benefits to users. The
motivation section can optionally provide links to [experience reports] to
demonstrate the interest in a KEP within the wider Kubernetes community.

[experience reports]: https://github.com/golang/go/wiki/ExperienceReports
-->

Many high-availability (HA) Kubernetes components, including kube-scheduler and
kube-controller-manager, rely on the leader election library in client-go. The
current implementation mandates that when a component loses its leader lock, it
must shut down immediately. This is typically handled by calling
`klog.FlushAndExit()` in the `OnStoppedLeading` callback.

When the leadership is lost, the component shuts down and waits for the kubelet
to detect that the component is unhealthy and restart it. This has several
significant drawbacks:

- High Overhead: Restarting a component process incurs unnecessary computational
  overhead and increases latency during a leadership transition
- No Graceful Shutdown: The immediate call to `os.Exit()` prevents any graaceful
  shutdown or cleanup operations.

### Goals

<!--
List the specific goals of the KEP. What is it trying to achieve? How will we
know that this has succeeded?
-->

- Allow leader-elected components to transition to a follower state without restarting the process
- Enable graceful handling of leader lock loss

### Non-Goals

<!--
What is out of scope for this KEP? Listing non-goals helps to focus discussion
and make progress.
-->

- This KEP allows work towards graceful shutdown of controllers, but the actual
  mechanisms of how we do graceful shutdown is outside the scope of this KEP.

## Proposal

<!--
This is where we get down to the specifics of what the proposal actually is.
This should have enough detail that reviewers can understand exactly what
you're proposing, but should not include things like API designs or
implementation. What is the desired outcome and how do we measure success?.
The "Design Details" section below is for the real
nitty-gritty.
-->

The main control loops for `kube-controller-manager`, `kube-scheduler`, and
`cloud-controller-manager` will be updated to support graceful leader
transitions. When a leader fails to renew its lease, instead of exiting, the
component will rely on client-go's leader election mechanism to cancel the
context, and stop its internal controllers. It will then immediately return to a
follower state where it will attempt to reacquire the lease.

This change will be guarded by a new feature gate, `GracefulLeaderTransition`.

### Risks and Mitigations

<!--
What are the risks of this proposal, and how do we mitigate? Think broadly.
For example, consider both security and how this will impact the larger
Kubernetes ecosystem.

How will security be reviewed, and by whom?

How will UX be reviewed, and by whom?

Consider including folks who also work outside the SIG or subproject.
-->

Kubernetes leader elected controllers and the scheduler have been running
without graceful shutdown for years. 

Risk 1: Resource exhaustion: Memory leaks may exist in the processes that were
previously masked by doing a full shutdown and restart loop. 

- Severity: Medium high
- Controllers will continue to function (potentially in degraded state due to
  lack of resources), and may be restarted frequently. However, cluster should
  continue to function.

Risk 2: Wedged KCM: There is a risk that controllers and the scheduler are not
properly respecting context shutdowns. This can either result in multiple
instances of controllers running or no instances running despite the lock being
held.

- Severity: High
- Breaking mutual exclusion guarantees can put the cluster into a non-desirable
  state. A manual user intervention is possible but if the problem is triggered
  due to a problematic component, the issue will resurface and the best path for
  mitigation is to turn off the feature.

Risk 3: Futureproofing: An additional risk is that even if all the current code
is safe and respects shutting down gracefully, new controllers/modifications to
kcm or scheduler could create subtle problems in shutdown and transition.

- Severity: High
- Leads to either risk 1 or 2.


Mitigations:

- Audit and add tests for the existing controllers and the scheduler to ensure
  proper handling of context shutdowns. See test plan section for more details.
- Graceful shutdown modifications will not be guarded by a feature gate, but the
  code change to remove the `os.Exit()` line will be guarded by a feature gate.
- Document the new development best practices for graceful shutdown requirements
  for modified components that are leader elected.

## Design Details

<!--
This section should contain enough information that the specifics of your
change are understandable. This may include API specs (though not always
required) or even code snippets. If there's any ambiguity about HOW your
proposal will be implemented, this is the place to discuss them.
-->

The core of this change involves modifying the `OnStoppedLeading` callback to
not forcefully exit.

We will wrap leader election with a `wait.Until()` to retry the leader election
loop similar to how the Coordinated Leader Election controller handles
gracefully transition of leaders
([code](https://github.com/kubernetes/kubernetes/blob/release-1.33/pkg/controlplane/controller/leaderelection/run_with_leaderelection.go#L54))

The `controller-manager` sets up controller level health checks in
non-reversible ways and will need to be modified so that handlers can be
deregistered from the mux when leadership is lost. All resources created after a
KCM becomes leader must be released when it loses leadership. This will be done
through context cancellation and cleanup logic. Some additional refactoring may
be needed to clean up processes gracefully when a leader lock is released. To
verify that individual controllers relinquish the control loop, we can add a
`ValidatingAdmissionPolicy` that warns when a controller that is not the leader
sends a write request to the apiserver, and fails the test. This will help us
identify locations where context cancellations are not respected.

Similarly for scheduler, assumptions that the process will be terminated losing
the leader lock are made. Many scheduler resources are created before the leader
election process. These will be modified to either defer resource creation or
add a resetting mechanism when the leader is lost.


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

To properly test that components can respect context cancellation shutdowns and
not leak memory, we will run both manual tests with profiling, and automated
testing that transitions the leader lock multiple times.

We will also test that things like caches, reflectors, and go funcs are properly
cleared/stopped when the leader lock is lost.

Scenarios:

- Component acquires leader lock should start its control **loop**
- Component releasing leader lock should shut down its control loop
- Component acquiring and releasing the leader lock multiple times should not leak memory
- Component acquiring and releasing the leader lock multiple times should have ONLY ONE control loop running
- If kube-apiserver takes a long time to release client calls, the graceful release will properly wait until the controller has returned
- Releasing a lock will stop the control loop
- Only one control loop should be running at all times
- Components in follower mode should not start control loop or otherwise allocate unnecessary memory

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

This is primarily testing for leader election's interaction with components, and will be tested via integration and e2e tests.


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

See the above scenarios for test plan. kube-scheduler and kcm in particular will be integration tested that they shut down properly.

##### e2e tests

<!--
This question should be filled when targeting a release.
For Alpha, describe what tests will be added to ensure proper quality of the enhancement.

For Beta and GA, add links to added tests together with links to k8s-triage for those tests:
https://storage.googleapis.com/k8s-triage/index.html

We expect no non-infra related flakes in the last month as a GA graduation criteria.
-->

See the above scenarios for test plan.

### Graduation Criteria

#### Alpha

- Feature implemented behind a feature flag
- Runtime detection of leaked goroutines
- Test that controller-manager and scheduler do not leak memory on leadership transitions

#### Beta

- e2e tests
- Address how to minimize risks of putting KCM or scheduler in a "wedged" state

#### GA

- TBD

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

No changes.

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

This is a control plane change. Skew should not affect the feature.

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
  - Feature gate name: GracefulLeaderTransition
  - Components depending on the feature gate: kube-scheduler, kube-controller-manager, cloud-controller-manager
- Will enabling / disabling the feature require downtime of the control plane? Yes, components need to be restarted.
- Will enabling / disabling the feature require downtime or reprovisioning of a node? No.

###### Does enabling the feature change any default behavior?

<!--
Any change of default behavior may be surprising to users or break existing
automations, so be extremely careful here.
-->
Yes. When the `GracefulLeaderTransition` feature gate is enabled, leader-elected
components (kube-scheduler, kube-controller-manager, cloud-controller-manager)
will attempt to gracefully release the leader lock and transition to a follower
state without a full process restart. Previously, these components would shut
down immediately upon losing leadership.

###### Can the feature be disabled once it has been enabled (i.e. can we roll back the enablement)?

<!--
Describe the consequences on existing workloads (e.g., if this is a runtime
feature, can it break the existing applications?).

Feature gates are typically disabled by setting the flag to `false` and
restarting the component. No other changes should be necessary to disable the
feature.

NOTE: Also set `disable-supported` to `true` or `false` in `kep.yaml`.
-->
Yes, the feature can be disabled by setting the `GracefulLeaderTransition`
feature gate to `false` and restarting the affected components (kube-scheduler,
kube-controller-manager, cloud-controller-manager). This will revert to the
previous behavior where components shut down immediately upon losing leadership.
This should not break existing workloads as it restores the prior,
well-understood behavior.

###### What happens if we reenable the feature if it was previously rolled back?

If the feature is re-enabled after being rolled back, the components will once
again use the graceful leader transition mechanism. There are no special
considerations for re-enabling.

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

No.

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
A rollout could fail if:
- Components (kube-scheduler, kube-controller-manager, cloud-controller-manager)
  do not correctly handle context cancellation when losing leadership, leading
  to incomplete shutdown of internal controllers.
- Memory leaks occur in the components because they no longer fully restart on
  leader transition, which previously masked such leaks.

Impact on workloads:
- If a leader component becomes unstable (e.g., due to memory leaks or improper
  shutdown), its ability to perform its duties (scheduling, controller
  management) could be impaired. In an HA setup, another instance should take
  over leadership, but frequent transitions or instability could degrade overall
  cluster performance or reliability.
- If a component fails to release resources correctly upon losing leadership, it
might lead to resource contention or incorrect behavior. Rollback (disabling the
feature gate and restarting components) should revert to the previous stable
behavior.

###### What specific metrics should inform a rollback?

<!--
What signals should users be paying attention to when the feature is young
that might indicate a serious problem?
-->
Some things to look at:
- Component restart counts: An increase in restarts for kube-controller-manager or kube-scheduler.
- Log messages indicating errors during leader transition or resource cleanup.
- General cluster health indicators like API server latency, pod scheduling latency, or controller reconciliation errors.

###### Were upgrade and rollback tested? Was the upgrade->downgrade->upgrade path tested?

<!--
Describe manual testing that was done and the outcomes.
Longer term, we may want to require automated upgrade/rollback tests, but we
are missing a bunch of machinery and tooling and can't do that now.
-->

n/a

###### Is the rollout accompanied by any deprecations and/or removals of features, APIs, fields of API types, flags, etc.?

<!--
Even if applying deprecation policies, they may still surprise some users.
-->
No. This feature introduces new behavior behind a feature gate and does not
deprecate or remove any existing features, APIs, fields, or flags.

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

An operator can determine if the feature is active by inspecting the
command-line flags of the relevant components (kube-scheduler,
kube-controller-manager, cloud-controller-manager) to verify that the
`GracefulLeaderTransition` feature gate is enabled.

Observing component logs for messages indicating graceful leader release (as
opposed to immediate shutdown) would also confirm its use.

###### How can someone using this feature know that it is working for their instance?

<!--
For instance, if this is a pod-related feature, it should be possible to determine if the feature is functioning properly
for each individual pod.
Pick one more of these and delete the rest.
Please describe all items visible to end users below with sufficient detail so that they can verify correct enablement
and operation of this feature.
Recall that end users cannot usually observe component logs or access metrics.
-->

This feature is primarily for cluster operators. Operators can verify its
operation by:

- Observing component logs: Logs for kube-scheduler, kube-controller-manager,
  and cloud-controller-manager should indicate that upon losing leadership, the
  component attempts a graceful shutdown of its internal loops and returns to a
  follower state to re-attempt leader election, rather than exiting.
- Monitoring component behavior: Affected components should not restart (i.e.,
  no new PIDs) immediately after losing leadership if the graceful transition is
  successful. They should continue running and attempt to reacquire leadership.

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

- No increase to leader transition time with the feature enabled.
- No increase to memory usage with feature is enabled

###### What are the SLIs (Service Level Indicators) an operator can use to determine the health of the service?

 Component logs can be used to verify graceful shutdown steps are being executed.

###### Are there any missing metrics that would be useful to have to improve observability of this feature?

<!--
Describe the metrics themselves and the reasons why they weren't added (e.g., cost,
implementation difficulties, etc.).
-->

n/a

### Dependencies

<!--
This section must be completed when targeting beta to a release.
-->

###### Does this feature depend on any specific services running in the cluster?

This is a control plane feature and requires the components that the feature
runs on (kube-scheduler, kube-controller-manager) to be active, as well as the
kube-apiserver and etcd for leader election.

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

No

###### Will enabling / using this feature result in introducing new API types?

<!--
Describe them, providing:
  - API type
  - Supported number of objects per cluster
  - Supported number of objects per namespace (for namespace-scoped objects)
-->

No

###### Will enabling / using this feature result in any new calls to the cloud provider?

<!--
Describe them, providing:
  - Which API(s):
  - Estimated increase:
-->

No

###### Will enabling / using this feature result in increasing size or count of the existing API objects?

<!--
Describe them, providing:
  - API type(s):
  - Estimated increase in size: (e.g., new annotation of size 32B)
  - Estimated amount of new objects: (e.g., new Object X for every existing Pod)
-->

No

###### Will enabling / using this feature result in increasing time taken by any operations covered by existing SLIs/SLOs?

<!--
Look at the [existing SLIs/SLOs].

Think about adding additional work or introducing new steps in between
(e.g. need to do X to start a container), etc. Please describe the details.

[existing SLIs/SLOs]: https://git.k8s.io/community/sig-scalability/slos/slos.md#kubernetes-slisslos
-->

No

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

###### How does this feature react if the API server and/or etcd is unavailable?

Leader election cannot function without apiserver or etcd.

###### What are other known failure modes?

- Memory Leak
  - Detection: kcm or kube-scheduler memory constantly increasing after leader changes.
  - Mitigations: Restart the container, turn off the feature.
    running user workloads?
  - Diagnostics: Looking at memory consumption of KCM and kube-scheduler.
  - Testing: Tests will be done manually.

###### What steps should be taken if SLOs are not being met to determine the problem?

Disable the feature.

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

- 2025-06-03 - KEP was marked as implementable

## Drawbacks

<!--
Why should this KEP _not_ be implemented?
-->

Introduces additional risk of memory leak.

## Alternatives

<!--
What other approaches did you consider, and why did you rule them out? These do
not need to be as detailed as the proposal, but should include enough
information to express the idea and why it was not acceptable.
-->

n/a

## Future Work (Stories)

This feature enables the user stories below, but require additional modification to the kcm and scheduler code that they are outside the scope of this KEP.

#### Story 1

In an HA configuration, cloud provider A wants to balance controllers over
multiple control plane instances. With graceful transitions, multiple locks can
be used by a single KCM instance such that a subset of components run under each
lock.

#### Story 2

A extension developer has controller manager that should dynamically start and
shutdown controllers based on cluster state (such as the registration of
resources that declare how a CRD should be reconciled). The extension developer
requires that controllers shutdown gracefully so that only the controller loops
that SHOULD be running continue to run, and that no resources are leaked over
time as controllers are started and stopped.



## Infrastructure Needed (Optional)

<!--
Use this section if you need things from the project/SIG. Examples include a
new subproject, repos requested, or GitHub details. Listing these here allows a
SIG to get the process for these resources started right away.
-->
