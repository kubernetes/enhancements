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
# KEP-5278: Nominated node name for an expected pod placement

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
  - [External components need to know the pod is going to be bound](#external-components-need-to-know-the-pod-is-going-to-be-bound)
  - [External components want to specify a preferred pod placement](#external-components-want-to-specify-a-preferred-pod-placement)
  - [Retain the scheduling decision](#retain-the-scheduling-decision)
  - [Goals](#goals)
  - [Non-Goals](#non-goals)
- [Proposal](#proposal)
  - [User Stories (Optional)](#user-stories-optional)
    - [Story 1: Prevent inappropriate scale downs by Cluster Autoscaler](#story-1-prevent-inappropriate-scale-downs-by-cluster-autoscaler)
    - [Story 2: Cluster Autoscaler specifies <code>NominatedNodeName</code> to indicate where pods can go after new nodes are created/registered](#story-2-cluster-autoscaler-specifies-nominatednodename-to-indicate-where-pods-can-go-after-new-nodes-are-createdregistered)
  - [Risks and Mitigations](#risks-and-mitigations)
    - [Increasing the load to kube-apiserver](#increasing-the-load-to-kube-apiserver)
    - [Race condition](#race-condition)
    - [Confusion if <code>NominatedNodeName</code> is different from <code>NodeName</code> after all](#confusion-if-nominatednodename-is-different-from-nodename-after-all)
    - [What if there are multiple components that could set <code>NominatedNodeName</code> on the same pod](#what-if-there-are-multiple-components-that-could-set-nominatednodename-on-the-same-pod)
    - [[CA scenario] If the cluster autoscaler puts unexisting node's name on <code>NominatedNodeName</code>, the scheduler clears it](#ca-scenario-if-the-cluster-autoscaler-puts-unexisting-nodes-name-on-nominatednodename-the-scheduler-clears-it)
    - [[CA scenario] A new node's taint prevents the pod from going there, and the scheduler ends up clearing <code>NominatedNodeName</code>](#ca-scenario-a-new-nodes-taint-prevents-the-pod-from-going-there-and-the-scheduler-ends-up-clearing-nominatednodename)
- [Design Details](#design-details)
  - [The scheduler puts <code>NominatedNodeName</code>](#the-scheduler-puts-nominatednodename)
  - [External components put <code>NominatedNodeName</code>](#external-components-put-nominatednodename)
  - [Test Plan](#test-plan)
      - [Prerequisite testing updates](#prerequisite-testing-updates)
      - [Unit tests](#unit-tests)
      - [Integration tests](#integration-tests)
      - [e2e tests](#e2e-tests)
  - [Graduation Criteria](#graduation-criteria)
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

Use `NominatedNodeName` to express an pod placement, expected by the scheduler or expected by other components.

The scheduler puts `NominatedNodeName` at the beginning of binding cycles to show an expected pod placement to other components.
And, also other components can put `NominatedNodeName` on pending pods to indicate the pod is prefered to be scheduled on a specific node.

## Motivation

### External components need to know the pod is going to be bound 

The scheduler reserves the place for the pod when the pod is entering the binding cycle.
This reservation is internally implemented in the scheduler's cache, and is not visible to other components.

The specific problem is, as shown at [#125491](https://github.com/kubernetes/kubernetes/issues/125491),
if the binding cycle takes time before binding pods to nodes (e.g., PreBind takes time to handle volumes)
the cluster autoscaler cannot understand the pod is going to be bound there,
misunderstands the node is low-utilized (because the scheduler keeps the place of the pod), and deletes the node. 

We can expose those internal reservations with `NominatedNodeName` so that external components can take a more appropriate action
based on the expected pod placement.

### External components want to specify a preferred pod placement

The cluster autoscaler or Kueue internally calculates the pod placement,
and create new nodes or un-gate pods based on the calculation result. 

So, they know where those pods are likely going to be scheduled.

By specifing their expectation on `NominatedNodeName`, the scheduler can first check whether the pod can go to the nominated node,
speeding up the filter phase.

### Retain the scheduling decision

At the binding cycle (e.g., PreBind), some plugins could handle something (e.g., volumes, devices) based on the pod's scheduling result.

If the scheduler restarts while it's handling some pods at binding cycles,
kube-scheduler could decide to schedule a pod to a different node. 
If we can keep where the pod was going to go at `NominatedNodeName`, the scheduler likely picks up the same node, and the PreBind plugins can restart their work from where they were before the restart.

### Goals

- The scheduler use `NominatedNodeName` to express where the pod is going to go before actually bound them.
- Make sure external components can use `NominatedNodeName` to express where they prefer the pod is going to.
  - Probably, you can do this with a today's scheduler as well. This proposal wants to discuss/make sure if it actually works, and then add tests etc.

### Non-Goals

- Extenral components can enforce the scheduler to pick up a specific node via `NominatedNodeName`.
  - `NominatedNodeName` is just a hint for scheduler and doesn't represent a hard requirement

## Proposal

### User Stories (Optional)

Here is the all use cases of NominatedNodeNames that we're taking into consideration:
- The scheduler puts it after the preemption (already implemented)
- The scheduler puts it at the beginning of binding cycles (only if the binding cycles invole PreBind phase)
- The cluster autoscaler puts it after creating a new node for pending pod(s) so that the scheduler can find a place faster when the node is created.
- Kueue uses it to determine a prefered node for the pod based on their internal calculation (Topology aware scheduling etc)

(Possibly, our future initiative around the workload scheduling (including gang scheduling) can also utilize it,
but we don't discuss it here because it's not yet concreted at all.)

#### Story 1: Prevent inappropriate scale downs by Cluster Autoscaler 

The scheduler starts to expose where the pod is going to with `NominatedNodeName` at the beginning of binding cycles.
And, the cluster autoscaler takes `NominatedNodeName` into consideration when calculating which nodes they delete.

It helps the scenarios where the binding cycles take time, for example, VolumeBinding plugin takes time at PreBind extension point.

#### Story 2: Cluster Autoscaler specifies `NominatedNodeName` to indicate where pods can go after new nodes are created/registered

Usually, the scheduler scans all the nodes in the cluster when scheduling pods.

When the cluster autoscaler creates instances for pending pods, it calculate which new node might get which pending pod.
If they can put `NominatedNodeName` based on those calculation, it could tell the scheduler that the node can probably picked up for the pod's scheduling,
prevenging the double effort of scanning/calculating all nodes again at the scheduling retries.

#### Story 3: Kueue specifies `NominatedNodeName` to indicate where it prefers pods being scheduled to

When Kueue determines where pods are prefered to being scheduled on, based on their internal scheduling soft constraints (Preferred Topology Aware Scheduling, etc)
currently, they just put the node selector to tell the scheduler about their preference, and then un-gate the pods.

After this proposal, they can specify `NominatedNodeName` instead of a prefered node selector, 
which makes the probability of pods being scheduled onto the node higher.

### Risks and Mitigations

<!--
What are the risks of this proposal, and how do we mitigate? Think broadly.
For example, consider both security and how this will impact the larger
Kubernetes ecosystem.

How will security be reviewed, and by whom?

How will UX be reviewed, and by whom?

Consider including folks who also work outside the SIG or subproject.
-->

#### Increasing the load to kube-apiserver

If we simply implement this, we'd double the API calls during a simple binding cycle (NNN + actual binding),
which would increase the load to kube-apiserver significantly.

To prevent that, we'll skip setting `NominatedNodeName` when all PreBind plugins have nothing to do with the pod.
(We'll discuss how-to in the later section.)
Then, setting `NominatedNodeName` happens only when, for example, a pod has a volume that VolumeBinding plugin needs to handle at PreBind.

Of course, the API calls would still be increasing especially if most of pods have delayed binding. 
However, those cases should actually be ok to have those additional calls because these will have other calls related to those operations (e.g., PV creation, etc.) - so the overhead of an additional call is effectively a smaller percentage of the e2e flow.

#### Race condition

If an external component adds `NominatedNodeName` to the pod that is going through a scheduling cycle,
`NominatedNodeName` isn't taken into account (of course), and the pod could be scheduled onto a different node.

But, this should be fine because, either way, we're not saying `NominatedNodeName` is something forcing the scheduler to pick up the node,
rather it's just a preference.

#### Confusion if `NominatedNodeName` is different from `NodeName` after all

If an external component adds `NominatedNodeName`, but the scheduler picks up a different node,
`NominatedNodeName` is just overwritten by a final decision of the scheduler.

But, if an external component updates `NominatedNodeName` that is set by the scheduler, 
the pod could end up having different `NominatedNodeName` and `NodeName`.

Probably we should clear `NominatedNodeName` when the pod is bound. (at binding api)

#### What if there are multiple components that could set `NominatedNodeName` on the same pod

Multiple controllers might keep overwriting NominatedNodeName that is set by the others. 
Of course, we can regard that just as user's fault though, that'd be undesired situation.

There could be several ideas to mitigate, or even completely solve by adding a new API.
But, we wouldn't like to introduce any complexity right now because we're not sure how many users would start using this,
and hit this problem.

So, for now, we'll just document it somewhere as a risk, unrecommended situation, and in the future, we'll consider something
if we actually observe this problem getting bigger by many people starting using it.

#### [CA scenario] If the cluster autoscaler puts unexisting node's name on `NominatedNodeName`, the scheduler clears it

The current scheduler clears the node name from `NominatedNodeName` if the pod goes through the scheduling cycle,
and the node doesn't exist.

In order for the cluster autoscaler to levarage this feature,
it has to put unexisting node's name, which is supposed to be registered later after its scale up,
so that the scheduler can schedule pending pods on those new nodes as soon as possible after nodes are registered.

So, we need to keep the node's name on `NominatedNodeName` even when the node doesn't exist.
We'll discuss it at [Only modifying `NominatedNodeName`](#only-modifying-nominatednodename) section.

#### [CA scenario] A new node's taint prevents the pod from going there, and the scheduler ends up clearing `NominatedNodeName`

With the current scheduler, what happens if CA puts `NominatedNodeName` is:
1. Pods are unschedulable. For the simplicity, let's say all of them are rejected by NodeResourceFit plugin. (i.e., no node has enough CPU/memory for pod's request)
2. CA finds them, calculates nodes necessary to be created
3. CA puts `NominatedNodeName` on each pod
4. The scheduler keeps trying to schedule those pending pods though, here let's say they're unschedulable (no cluster event happens that could make pods schedulable) until the node is created.
5. The nodes are created, and registered to kube-apiserver. Let's say, at this point, nodes have un-ready taints.
6. The scheduler observes `Node/Create` event, `NodeResourceFit` plugin QHint returns `Queue`, and those pending pods are requeued to activeQ.
7. The scheduling cycle starts handling those pending pods.
8. However, because nodes have un-ready taints, pods are rejected by `TaintToleration` plugin.
9. The scheduler clears `NominatedNodeName` because it finds the nominated node (= new node) unschedulable.

So, after all, `NominatedNodeName` added by CA in this scaling up scenario doesn't add any value, 
unless the taints are removed in a short time (between 6 and 7).

So, we need to keep the node's name on `NominatedNodeName` even when the node doesn't fit right now.
We'll discuss it at [Only modifying `NominatedNodeName`](#only-modifying-nominatednodename) section.

## Design Details

<!--
This section should contain enough information that the specifics of your
change are understandable. This may include API specs (though not always
required) or even code snippets. If there's any ambiguity about HOW your
proposal will be implemented, this is the place to discuss them.
-->
### The scheduler puts `NominatedNodeName`

After the pod is permitted at `WaitOnPermit`, the scheduler needs to update `NominatedNodeName` with the node that it determines the pod is going to.

Also, in order to set `NominatedNodeName` only when some PreBind plugins work, we need to add a new function (or create a new extension point, if we are concerned about the breaking change to the existing PreBind plugins).

```go
type PreBindPlugin interface {
	Plugin
	// **New Function** (or we can have a separate Plugin interface for this, if we're concerned about a breaking change for custom plugins)
	// It's called before PreBind, and the plugin is supposed to return Success, Skip, or Error status.
	// If it returns Skip, it means this PreBind plugin has nothing to do with the pod.
	// This function should be lightweight, and shouldn't do any actual operation, e.g., creating a volume etc
	PreBindPreFlight(ctx context.Context, state *CycleState, p *v1.Pod, nodeName string) *Status

	PreBind(ctx context.Context, state *CycleState, p *v1.Pod, nodeName string) *Status
}
```

The scheduler would run a new function `PreBindPreFlight()` before `PreBind()` functions, 
and if all PreBind plugins return Skip status from new functions, we can skip setting `NominatedNodeName`.

This is a similar approach we're doing with PreFilter/PreScore -> Filter/Score. 
We determine if each plugin is relevant to the pod by Skip status from PreFilter/PreScore, and then determine whether to run Filter/Score function accordingly.

In this way, even if users have some PreBind custom plugins, they can implement `PreBindPreFlight()` appropriately 
so that the scheduler can wisely skip setting `NominatedNodeName`, taking their custom logic into consideration.

### External components put `NominatedNodeName`

There aren't any restrictions preventing other components from setting NominatedNodeName as of now.
However, we don't have any validation of how that currently works.
To support the usecases mentioned above we will adjust the scheduler to do the following:
- if NominatedNodeName is set, but corresponding Node doesn't exist, kube-scheduler will NOT clear it when the pod is unschedulable [assuming that a node might appear soon]
- We will rely on the fact that a pod with NominatedNodeName set is resulting in the in-memory reservation for requested resources. 
Higher-priority pods can ignore it, but pods with equal or lower priority don't have access to these resources. 
This allows us to prioritize nominated pods when nomination was done by external components. 
We just need to ensure that in case when NominatedNodeName was assigned by an external component, this nomination will get reflected in scheduler memory.

We will implement integration tests simulating the above behavior of external components.

#### The scheduler only modifies `NominatedNodeName`, not clears it in any cases

As described at the risk section, there are two problematic scenarios where this use case wouldn't work.
- [[CA scenario] If the cluster autoscaler puts unexisting node's name on `NominatedNodeName`, the scheduler clears it](#ca-scenario-if-the-cluster-autoscaler-puts-unexisting-nodes-name-on-nominatednodename-the-scheduler-clears-it)
- [[CA scenario] A new node's taint prevents the pod from going there, and the scheduler ends up clearing `NominatedNodeName`](#ca-scenario-a-new-nodes-taint-prevents-the-pod-from-going-there-and-the-scheduler-ends-up-clearing-nominatednodename)

Currently, the scheduler clears `NominatedNodeName` at the end of failed scheduling cycles if it found the nominated node unschedulable for the pod.
In order to avoid above two scenarios, we have to remove this clearing logic; change the scheduler not to clear `NominatedNodeName` in any cases.
It means, even if the node on `NominatedNodeName` isn't valid anymore, the scheduler keeps trying the node first.
We regard the additional cost of checking `NominatedNodeName` first unnecessarily isn't reletively big (especially for big clusters, where the performance is critical) because it's just one iteration of Filter plugins.
e.g., if you have 1000 nodes and 16 parallelism (default value), the scheduler needs around 62 iterations of Filter plugins, approximately. So, adding one iteration on top of that doesn't matter.

Also, note that we still allow the scheduler overwrite `NominatedNodeName` when it triggers the preemption for the pod.

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

- `<package>`: `<date>` - `<test coverage>`

##### Integration tests

<!--
Integration tests are contained in https://git.k8s.io/kubernetes/test/integration.
Integration tests allow control of the configuration parameters used to start the binaries under test.
This is different from e2e tests which do not allow configuration of parameters.
Doing this allows testing non-default options and multiple different and potentially conflicting command line options.
For more details, see https://github.com/kubernetes/community/blob/master/contributors/devel/sig-testing/testing-strategy.md

If integration tests are not necessary or useful, explain why.
-->

<!--
This question should be filled when targeting a release.
For Alpha, describe what tests will be added to ensure proper quality of the enhancement.

For Beta and GA, document that tests have been written,
have been executed regularly, and have been stable.
This can be done with:
- permalinks to the GitHub source code
- links to the periodic job (typically https://testgrid.k8s.io/sig-release-master-blocking#integration-master), filtered by the test name
- a search in the Kubernetes bug triage tool (https://storage.googleapis.com/k8s-triage/index.html)
-->

- [test name](https://github.com/kubernetes/kubernetes/blob/2334b8469e1983c525c0c6382125710093a25883/test/integration/...): [integration master](https://testgrid.k8s.io/sig-release-master-blocking#integration-master?include-filter-by-regex=MyCoolFeature), [triage search](https://storage.googleapis.com/k8s-triage/index.html?test=MyCoolFeature)

##### e2e tests

<!--
This question should be filled when targeting a release.
For Alpha, describe what tests will be added to ensure proper quality of the enhancement.

For Beta and GA, document that tests have been written,
have been executed regularly, and have been stable.
This can be done with:
- permalinks to the GitHub source code
- links to the periodic job (typically a job owned by the SIG responsible for the feature), filtered by the test name
- a search in the Kubernetes bug triage tool (https://storage.googleapis.com/k8s-triage/index.html)

We expect no non-infra related flakes in the last month as a GA graduation criteria.
If e2e tests are not necessary or useful, explain why.
-->

- [test name](https://github.com/kubernetes/kubernetes/blob/2334b8469e1983c525c0c6382125710093a25883/test/e2e/...): [SIG ...](https://testgrid.k8s.io/sig-...?include-filter-by-regex=MyCoolFeature), [triage search](https://storage.googleapis.com/k8s-triage/index.html?test=MyCoolFeature)

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

- [ ] Feature gate (also fill in values in `kep.yaml`)
  - Feature gate name:
  - Components depending on the feature gate:
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

###### Can the feature be disabled once it has been enabled (i.e. can we roll back the enablement)?

<!--
Describe the consequences on existing workloads (e.g., if this is a runtime
feature, can it break the existing applications?).

Feature gates are typically disabled by setting the flag to `false` and
restarting the component. No other changes should be necessary to disable the
feature.

NOTE: Also set `disable-supported` to `true` or `false` in `kep.yaml`.
-->

###### What happens if we reenable the feature if it was previously rolled back?

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
