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
# KEP-3521: Pod Scheduling Readiness

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
    - [Story 4](#story-4)
  - [Notes/Constraints/Caveats (Optional)](#notesconstraintscaveats-optional)
  - [Risks and Mitigations](#risks-and-mitigations)
- [Design Details](#design-details)
  - [API](#api)
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
- [ ] (R) KEP approvers have approved the KEP status as `implementable`
- [x] (R) Design details are appropriately documented
- [x] (R) Test plan is in place, giving consideration to SIG Architecture and SIG Testing input (including test refactors)
  - [ ] e2e Tests for all Beta API Operations (endpoints)
  - [ ] (R) Ensure GA e2e tests meet requirements for [Conformance Tests](https://github.com/kubernetes/community/blob/master/contributors/devel/sig-architecture/conformance-tests.md) 
  - [ ] (R) Minimum Two Week Window for GA e2e tests to prove flake free
- [x] (R) Graduation criteria is in place
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

This KEP aims to add a `.spec.schedulingGates` field to Pod's API, to mark a Pod's schedule readiness.
Integrators can mutate this field to signal to scheduler when a Pod is ready for scheduling.

## Motivation

<!--
This section is for explicitly listing the motivation, goals, and non-goals of
this KEP.  Describe why the change is important and the benefits to users. The
motivation section can optionally provide links to [experience reports] to
demonstrate the interest in a KEP within the wider Kubernetes community.

[experience reports]: https://github.com/golang/go/wiki/ExperienceReports
-->

Pods are currently considered ready for scheduling once created. Kubernetes scheduler does
its due diligence to find nodes to place all pending Pods. However, in a real-world case,
some Pods may stay in a "miss-essential-resources" state for a long period. These pods
actually churn the scheduler (and downstream integrators like Cluster AutoScaler) in an
unnecessary manner.

Lacking the knob to flag Pods as scheduling-paused/ready wastes scheduling cycles on retrying
Pods that are determined to be unschedulable. As a result, it delays the scheduling of other Pods,
and also lowers the overall scheduling throughput. Moreover, it imposes restrictions to vendors to
develop some in-house features (such as hierarchical quota) natively.

On the other hand, a condition `{type:PodScheduled, reason:Unschedulable}` works as canonical info
exposed by scheduler to guide downstream integrators (e.g., ClusterAutoscaler) to supplement cluster
resources. Our solution should not break this contract.

This proposal describes APIs and mechanics to allow users/controllers to control when a pod is
ready to be considered for scheduling.

### Goals

<!--
List the specific goals of the KEP. What is it trying to achieve? How will we
know that this has succeeded?
-->

- Define an API to mark Pods as scheduling-paused/ready.
- Design a new Enqueue extension point to customize Pod's queueing behavior.
- Not mark scheduling-paused Pods as `Unschedulable` by updating their `PodScheduled` condition.
- A default enqueue plugin to honor the new API semantics.

### Non-Goals

<!--
What is out of scope for this KEP? Listing non-goals helps to focus discussion
and make progress.
-->

- Enforce updating the Pod's conditions to expose more context for scheduling-paused Pods.
- Focus on in-house use-cases of the Enqueue extension point.

## Proposal

<!--
This is where we get down to the specifics of what the proposal actually is.
This should have enough detail that reviewers can understand exactly what
you're proposing, but should not include things like API designs or
implementation. What is the desired outcome and how do we measure success?.
The "Design Details" section below is for the real
nitty-gritty.
-->

We propose a new field `.spec.schedulingGates` to the Pod API. The field is defaulted to nil
(equivalent to an empty map).

For Pods carrying non-nil `.spec.schedulingGates`, they will be "parked" in scheduler's internal
unschedulablePods pool, and only get tried when the field is mutated to nil.

Practically, this field can be initialized by a single client and(or) multiple mutating webhooks,
and afterwards each gate entry can be removed by external integrators when certain criteria is met.

### User Stories (Optional)

<!--
Detail the things that people will be able to do if this KEP is implemented.
Include as much detail as possible so that people can understand the "how" of
the system. The goal here is to make this feel real for users without getting
bogged down.
-->

#### Story 1

As an orchestrator developer, such as a dynamic quota manager, I have the full picture to know when
Pods are scheduling-ready; therefore, I want an API to signal to kube-scheduler when to consider a
Pod for scheduling. The pattern for this story would be to use a mutating webhook to force creating
pods in a "not-ready to schedule" state that the custom orchestrator changes to ready at a later
time based on its own evaluations.

#### Story 2

This story is an extension of the previous one in that more than one custom orchestrator could be
deployed on the same cluster, therefore they want an API that enables establishing an agreement om
when a pod is considered ready for scheduling.

#### Story 3

As an advanced scheduler developer, I want to compose a series of scheduler PreEnqueue plugins
to guard the schedulability of my Pods. This enables splitting custom enqueue admission logic into
several building blocks, and thus offers the most flexibility. Meanwhile, some plugin needs to visit
the in-memory state (like waiting Pods) that is only accessible via scheduler framework.

#### Story 4

A custom workload orchestrator may wish to modify the Pod prior to consideration for scheduling,
without having to fork or alter the workload controller. The orchestrator may wish to make 
time-varying (post-creation) decisions on Pod scheduling, perhaps to preserve scheduling constraints,
avoid disruption, or prevent co-existence.

### Notes/Constraints/Caveats (Optional)

<!--
What are the caveats to the proposal?
What are some important details that didn't come across above?
Go in to as much detail as necessary here.
This might be a good place to talk about core concepts and how they relate.
-->

- **Restricted state transition:** The `schedulingGates` field can be initialized only when a Pod is
created (either by the client, or mutated during admission). After creation, each `schedulingGate`
can be removed in arbitrary order, but addition of a new scheduling gate is disallowed.
To ensure consistency, scheduled Pod must always have empty `schedulingGates`.
This means that a client (an administrator or custom controller) cannot create a Pod that has
`schedulingGates` populated in any way, if that Pod also specified a `spec.nodeName`. In this case,
API Server will return a validation error.

    |                                              | non-nil schedulingGates⏸️ | nil schedulingGates▶️ |
    |----------------------------------------------|-------------------------|-------------------------|
    | unscheduled Pod<br>(nil <tt>nodeName</tt>)   | ✅ create<br>❌ update   | ✅ create<br>✅ update    |
    | scheduled Pod<br>(non-nil <tt>nodeName</tt>) | ❌ create<br>❌ update   | ✅ create<br>✅ update    |

- **New field disabled in Alpha but not scheduler extension:** In Alpha, the new Pod field is disabled
by default. However, the scheduler's extension point is activated no matter the feature gate is enabled
or not. This enables scheduler plugin developers to tryout the feature even in Alpha, by crafting
different enqueue plugins and wire with custom fields or conditions.

- **New phase literal in kubectl:** To provide better UX, we're going to add a new phase literal
`SchedulingPaused` to the "phase" column of `kubectl get pod`. This new literal indicates whether it's
scheduling-paused or not.

### Risks and Mitigations

<!--
What are the risks of this proposal, and how do we mitigate? Think broadly.
For example, consider both security and how this will impact the larger
Kubernetes ecosystem.

How will security be reviewed, and by whom?

How will UX be reviewed, and by whom?

Consider including folks who also work outside the SIG or subproject.
-->

- The scheduler doesn't actively clear a Pod's `schedulingGates` field. This means if some controller
is ill-implemented, some Pods may stay in Pending state incorrectly. If you noticed a Pod stays
in Pending state for a long time and it carries non-nil `schedulingGates`, you may find out which
component owns the gate(s) (via `.spec.managedFields`) and report the symptom to the component owner.

- Faulty controllers may forget to remove the Pod's `schedulingGates`, and hence results in
a large number of unschedulable Pods. In Alpha, we don't limit the number of unschedulable Pods caused
by potential faulty controllers. We will evaluate necessary options in the future to mitigate
potential abuse.

- End-users may be confused by no scheduling events in the output of `kubectl describe pod xyz`. We
will provide detailed documentation, along with metrics, tooling (kubectl) and(or) events.

## Design Details

<!--
This section should contain enough information that the specifics of your
change are understandable. This may include API specs (though not always
required) or even code snippets. If there's any ambiguity about HOW your
proposal will be implemented, this is the place to discuss them.
-->

### API

A new API field `SchedulingGates` will be added to Pod's spec:

```go
type PodSpec struct {
    // Each scheduling gate represents a particular scenario the scheduling is blocked upon.
    // Scheduling is triggered only when SchedulingGates is empty.
    // In the future, we may impose permission mechanics to restrict which controller can mutate
    // which scheduling gate. It's dependent on a yet-to-be-implemented fined-grained
    // permission (https://docs.google.com/document/d/11g9nnoRFcOoeNJDUGAWjlKthowEVM3YGrJA3gLzhpf4)
    // and needs to be consistent with how finalizes/liens work today.
    SchedulingGates []PodSchedulingGate
}

type PodSchedulingGate struct {
    // Name of the scheduling gate.
    // Each scheduling gate must have a unique name field.
    Name string
}
```

In the scheduler's ComponentConfig API, we'll add a new type of extension point called `Enqueue`:

```go
type Plugins struct {
    ......
    Enqueue PluginSet
}
```

### Implementation

Inside scheduler, an internal queue called "activeQ" is designed to store all ready-to-schedule Pods.
In this KEP, we're going to wire the aforementioned `Enqueue` extension to the logic deciding whether
or not to add Pods to "activeQ".

Specifically, prior to adding a Pod to "activeQ", scheduler iterates over registered Enqueue plugins.
An Enqueue plugin must implement the `EnqueuePlugin` interface to return a `Status` to tell scheduler
whether this Pod can be admitted or not:

```go
// EnqueuePlugin is an interface that must be implemented by "EnqueuePlugin" plugins.
// These plugins are called prior to adding Pods to activeQ.
// Note: an enqueue plugin is expected to be lightweight and efficient, so it's not expected to
// involve expensive calls like accessing external endpoints; otherwise it'd block other
// Pods' enqueuing in event handlers.
type EnqueuePlugin interface {
    Plugin
    PreEnqueue(ctx context.Context, state *CycleState, p *v1.Pod) *Status
}
```

A Pod can be moved to activeQ only when all Enqueue plugins returns `Success`. Otherwise, it's moved
and parked in the internal unschedulable Pods pool. The pseudo-code is roughly like this:

```go
// pseudo-code
func RunEnqueuePlugins() *Status {
    for _, pl := range enqueuePlugins {
        if status := pl.PreEnqueue(); !status.IsSuccess {
            // Logic: move Pod to unschedulable pod pool.
            return status
        }
    }
    // Logic: move pod to activeQ.
}
```

To honor the semantics of the new `.schedulingGates` API, a default Enqueue plugin will be
introduced. It simply returns `Success` or `Unschedulable` depending on incoming Pod's `schedulingGates`
field.

This `DefaultEnqueue` plugin will also implement the `EventsToRegister` function to claim it's a
`EnqueueExtension` object. So scheduler can move the Pod back to activeQ properly.

```go
func (pl *DefaultEnqueue) EventsToRegister() []framework.ClusterEvent {
	  return []framework.ClusterEvent{
		  {Resource: framework.Pod, ActionType: framework.Update},
	  }
}
```

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

None.

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

All core changes must be covered by unit tests, in both API and scheduler sides:

- **API:** API validation and strategy tests (`pkg/registry/core/pod`) to verify disabled fields
when the feature gate is on/off.

- **Scheduler:** Core scheduling changes which includes Enqueue config API's validation, defaulting,
integration and its implementation.

In particular, update existing UTs or add new UTs
in the following packages:

- `pkg/api/pod`: `10/3/2022` - `70.1%`
- `pkg/apis/core/validation`: `10/3/2022` - `82.3%`
- `pkg/registry/core/pod`: `10/3/2022` - `60.4%`
- `cmd/kube-scheduler/app`: `10/3/2022` - `32.9`
- `pkg/scheduler`: `10/3/2022` - `75.9%`
- `pkg/scheduler/framework/runtime`: `10/3/2022` - `81.9%`

##### Integration tests

<!--
This question should be filled when targeting a release.
For Alpha, describe what tests will be added to ensure proper quality of the enhancement.

For Beta and GA, add links to added tests together with links to k8s-triage for those tests:
https://storage.googleapis.com/k8s-triage/index.html
-->

The following scenarios need to be covered in integration tests:

- Feature gate's enabling/disabling
- Configure an Enqueue plugin via MultiPoint and Enqueue extension point
- Pod carrying nil `.spec.schedulingGates` functions as before
- Pod carrying non-nil `.spec.schedulingGates` will be moved to unscheduledPods pool
- Disable `flushUnschedulablePodsLeftover()`, then verify Pod with non-nil `.spec.schedulingGates`
can be moved back to activeQ when `.spec.schedulingGates` is all cleared
- Ensure no significant performance degradation

- `test/integration/scheduler/queue_test.go`: Will add new tests.
- `test/integration/scheduler/plugins/plugins_test.go`: Will add new tests.
- `test/integration/scheduler/enqueue/enqueue_test.go`: Will add new tests.
- `test/integration/scheduler_perf/scheduler_perf_test.go`: https://storage.googleapis.com/k8s-triage/index.html?test=BenchmarkPerfScheduling

##### e2e tests

<!--
This question should be filled when targeting a release.
For Alpha, describe what tests will be added to ensure proper quality of the enhancement.

For Beta and GA, add links to added tests together with links to k8s-triage for those tests:
https://storage.googleapis.com/k8s-triage/index.html

We expect no non-infra related flakes in the last month as a GA graduation criteria.
-->

Create a test with the following sequences:

- Provision a cluster with feature gate `PodSchedulingReadiness=true` (we may need to setup a testgrid
for when it's alpha)
- Create a Pod with non-nil `.spec.schedulingGates`.
- Wait for 15 seconds to ensure (and then verify) it did not get scheduled.
- Clear the Pod's `.spec.schedulingGates` field.
- Wait for 5 seconds for the Pod to be scheduled; otherwise error the e2e test.

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

- Feature disabled by default.
- Unit and integration tests completed and passed.
- API strategy test (`pkg/registry/*`) to verify disabled fields when the feature gate is on/off.
- Additional tests are in Testgrid and linked in KEP.
- Determine whether any additional state is required per gate.

#### Beta

- Feature enabled by default.
- Permission control on individual schedulingGate is applicable (via 
[fine-grained permissions](https://docs.google.com/document/d/11g9nnoRFcOoeNJDUGAWjlKthowEVM3YGrJA3gLzhpf4)).
- Gather feedback from developers and out-of-tree plugins.
- Benchmark tests passed, and there is no performance degradation.
- Update documents to reflect the changes.
- Identify whether gates can be added post-creation.

#### GA

- Fix all reported bugs.
- Feature enabled and cannot be disabled. All feature gate guarded logic are removed.
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

- Upgrade
  - Enable the feature gate in both API Server and Scheduler, and gate the Pod's scheduling
  readiness by setting non-nil `.spec.schedulingGates`. Next, remove each scheduling gate
  when readiness criteria is met.
- Downgrade
  - Disable the feature gate in both API Server and Scheduler, so that previously configured
  `.spec.schedulingGates` value will be ignored.
  - However, the `.spec.schedulingGates` value of a Pod is preserved if it's previously configured;
  otherwise get silently dropped.

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

The skew between kubelet and control-plane components are not impacted.

If the API Server is at vCurrent and the feature is enabled, while scheduler is at 
vCurrent-n that the feature is not supported, controllers manipulating the new API field won't
get their Pods scheduling gated. The Pod scheduling will behave like this feature is not
introduced.

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
  - Feature gate name: PodSchedulingReadiness
  - Components depending on the feature gate: kube-scheduler, kube-apiserver

###### Does enabling the feature change any default behavior?

<!--
Any change of default behavior may be surprising to users or break existing
automations, so be extremely careful here.
-->

No. It's a new API field, so no default behavior will be impacted.

###### Can the feature be disabled once it has been enabled (i.e. can we roll back the enablement)?

<!--
Describe the consequences on existing workloads (e.g., if this is a runtime
feature, can it break the existing applications?).

Feature gates are typically disabled by setting the flag to `false` and
restarting the component. No other changes should be necessary to disable the
feature.

NOTE: Also set `disable-supported` to `true` or `false` in `kep.yaml`.
-->

Yes. If disabled, kube-apiserver will start rejecting Pod's mutation on `.spec.schedulingGates`.

###### What happens if we reenable the feature if it was previously rolled back?

Mutation on Pod's `.spec.schedulingGates` will be respected again.

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

Appropriate tests will be added in Alpha. See [Test Plan](#test-plan) for more details.

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

Add a label `notReady` to the existing metric `pending_pods` to distinguish unschedulable Pods:

- `unschedulable` (existing): scheduler tried but cannot find any Node to host the Pod
- `notReady` (new): scheduler respect the Pod's present `schedulingGates` and hence not schedule it

Moreover, to explicitly indicate a Pod's scheduling-unready state, a condition
`{type:PodScheduled, reason:WaitingForGates}` is introduced.

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

<!--
- [ ] Events
  - Event Reason:
-->

- [x] API .spec
  - Other field: `schedulingGates`

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

- 2022-09-16 - Initial Proposal

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

- Define a boolean filed `.spec.schedulingPaused`. Its value is optionally initialized to `True` to
indicate it's not scheduling-ready, and flipped to `False` (by a controller) afterwards to trigger
this Pod's scheduling cycle.

  This approach is not chosen because it cannot support multiple independent controllers to control
specific aspect a Pod's scheduling readiness.

## Infrastructure Needed (Optional)

<!--
Use this section if you need things from the project/SIG. Examples include a
new subproject, repos requested, or GitHub details. Listing these here allows a
SIG to get the process for these resources started right away.
-->
