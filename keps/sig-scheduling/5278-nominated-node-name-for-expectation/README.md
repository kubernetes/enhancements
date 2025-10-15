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
  - [External components need to know where the pod is going to be bound](#external-components-need-to-know-where-the-pod-is-going-to-be-bound)
  - [Retain the scheduling decision](#retain-the-scheduling-decision)
  - [Goals](#goals)
  - [Non-Goals](#non-goals)
- [Proposal](#proposal)
  - [User Stories (Optional)](#user-stories-optional)
    - [Story 1: Prevent inappropriate scale downs by Cluster Autoscaler](#story-1-prevent-inappropriate-scale-downs-by-cluster-autoscaler)
    - [Story 2: Scheduler can resume its work after restart](#story-2-scheduler-can-resume-its-work-after-restart)
  - [Risks and Mitigations](#risks-and-mitigations)
    - [Confusing semantics of <code>NominatedNodeName</code>](#confusing-semantics-of-nominatednodename)
    - [Node nominations need to be considered together with reserving DRA resources](#node-nominations-need-to-be-considered-together-with-reserving-dra-resources)
    - [Increasing the load to kube-apiserver](#increasing-the-load-to-kube-apiserver)
- [Design Details](#design-details)
  - [The scheduler puts <code>NominatedNodeName</code>](#the-scheduler-puts-nominatednodename)
  - [The scheduler's cache for <code>NominatedNodeName</code>](#the-schedulers-cache-for-nominatednodename)
    - [The scheduler clears <code>NominatedNodeName</code> after scheduling failure](#the-scheduler-clears-nominatednodename-after-scheduling-failure)
  - [Kube-apiserver clears <code>NominatedNodeName</code> when receiving binding requests](#kube-apiserver-clears-nominatednodename-when-receiving-binding-requests)
  - [Handling ResourceClaim status updates](#handling-resourceclaim-status-updates)
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
    - [Introduce a new field](#introduce-a-new-field)
    - [Allow NominatedNodeName to be set by other components](#allow-nominatednodename-to-be-set-by-other-components)
      - [Motivation: External components want to specify a preferred pod placement](#motivation-external-components-want-to-specify-a-preferred-pod-placement)
      - [Goals](#goals-1)
      - [Non-Goals](#non-goals-1)
      - [User stories](#user-stories)
      - [Risks and Mitigations](#risks-and-mitigations-1)
    - [Confusion if <code>NominatedNodeName</code> is different from <code>NodeName</code> after all](#confusion-if-nominatednodename-is-different-from-nodename-after-all)
      - [Design Details](#design-details-1)
      - [Test plan: Integration tests](#test-plan-integration-tests)
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

- [X] (R) Enhancement issue in release milestone, which links to KEP dir in [kubernetes/enhancements] (not the initial KEP PR)
- [X] (R) KEP approvers have approved the KEP status as `implementable`
- [X] (R) Design details are appropriately documented
- [X] (R) Test plan is in place, giving consideration to SIG Architecture and SIG Testing input (including test refactors)
  - [ ] e2e Tests for all Beta API Operations (endpoints)
  - [ ] (R) Ensure GA e2e tests meet requirements for [Conformance Tests](https://github.com/kubernetes/community/blob/master/contributors/devel/sig-architecture/conformance-tests.md) 
  - [ ] (R) Minimum Two Week Window for GA e2e tests to prove flake free
- [X] (R) Graduation criteria is in place
  - [ ] (R) [all GA Endpoints](https://github.com/kubernetes/community/pull/1806) must be hit by [Conformance Tests](https://github.com/kubernetes/community/blob/master/contributors/devel/sig-architecture/conformance-tests.md) 
- [X] (R) Production readiness review completed
- [X] (R) Production readiness review approved
- [X] "Implementation History" section is up-to-date for milestone
- [X] User-facing documentation has been created in [kubernetes/website], for publication to [kubernetes.io]
- [X] Supporting documentation—e.g., additional design documents, links to mailing list discussions/SIG meetings, relevant PRs/issues, release notes

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

Use `NominatedNodeName` to express pod placement, expected by the scheduler.

Besides of using `NominatedNodeName` to indicate ongoing preemption, the scheduler can specify it at the beginning of a binding cycle to show an expected pod placement to other components.

## Motivation

### External components need to know where the pod is going to be bound 

The scheduler reserves the place for the pod when the pod is entering the binding cycle.
This reservation is internally implemented in the scheduler's cache, and is not visible to other components.

The specific problem is, as shown at [#125491](https://github.com/kubernetes/kubernetes/issues/125491),
if the binding cycle takes time before binding pods to nodes (e.g., PreBind takes time to handle volumes)
the cluster autoscaler cannot understand the pod is going to be bound there,
misunderstands the node is low-utilized (because the scheduler keeps the place of the pod), and deletes the node. 

We can expose those internal reservations with `NominatedNodeName` so that external components can take a more appropriate action
based on the expected pod placement.

Please note that the `NominatedNodeName` can express reservation of node resources only, but some resources can be managed by the DRA plugin and be expressed using ResourceClaim allocation. In order to correctly account all the resources needed by a pod, both the nomination and ResourceClaim status update needs to be reflected in the api-server.

### Retain the scheduling decision

At the binding cycle (e.g., PreBind), some plugins could handle something (e.g., volumes, devices) based on the pod's scheduling result.

If the scheduler restarts while it's handling some pods at binding cycles,
kube-scheduler could decide to schedule a pod to a different node. 
If we can keep where the pod was going to go at `NominatedNodeName`, the scheduler likely picks up the same node,
and the PreBind plugins can restart their work from where they were before the restart.

### Goals

- The scheduler will use `NominatedNodeName` to express where the pod is going to go before actually binding them.

### Non-Goals

- External components can suggest a specific node to kube-scheduler using `NominatedNodeName`.
  - This is not in scope of this feature for the time being. See the alternatives section for more details.


## Proposal

### User Stories (Optional)

Here is the all use cases of NominatedNodeNames that we're taking into consideration:
- The scheduler puts it after the preemption (already implemented)
- The scheduler puts it at the beginning of binding cycles (only if the binding cycles involve PreBind or WaitOnPermit phase)

(Possibly, our future initiative around the workload scheduling (including gang scheduling) can also utilize it,
but we don't discuss it here because it's not yet concrete.)

#### Story 1: Prevent inappropriate scale downs by Cluster Autoscaler 

Pod binding may take significant amount of time (even at the order of minutes, e.g. due to volume binding).
During that time, components other than the scheduler don't have the information that such a placement decision
has already been made and is already executed. Without having this information, other components may decide
to take conflicting actions (e.g. ClusterAutoscaler or Karpenter may decide to delete that particular node).

We need a way to share the information about already made scheduling decisions with those components to prevent that. 

#### Story 2: Scheduler can resume its work after restart

Pod binding may take significant amount of time (even at the order of minutes, e.g. due to volume binding).
During that time, scheduler may be restarted, lost its leader lock etc.
Given the placement decision was only stored in schedulers memory, the new incarnation of the scheduler
has no visibility into it and can decide to put a pod on a different node. This would result in wasting
the work that has already been done and increase the end-to-end pod startup latency.

We need a mechanism to be able to resume the already started work in majority of such situations.

### Risks and Mitigations

#### Confusing semantics of `NominatedNodeName`

Up until now, `NominatedNodeName` was expressing the decision made by scheduler to put a given
pod on a given node, while waiting for the preemption. The decision could be changed later so
it didn't have to be a final decision, but it was describing the "current plan of record".

If we add the case of delayed binding, we effectively get a state machine with the following states:

1. pending pod
2. pod nominated to node and waiting for preemption
3. pod allocated to node and waiting for binding
4. pod bound

The important part is that if we decide to use `NominatedNodeName` to store information for both (2) and (3),
we're effectively losing the ability to distinguish between those states.

We may argue that as long as the decision was made by the scheduler, the exact reason and state
probably isn't that important - the content of `NominatedNodeName` can be interpreted as
"current plan of record for this pod from scheduler perspective".

If we look from consumption point of view - these are effectively the same. We want
to expose the information, that as of now a given node is considered as a potential placement
for a given pod. It may change, but for now that's what is being considered.

#### Node nominations need to be considered together with reserving DRA resources

The semantics of node nomination are in fact resource reservation, either in scheduler memory or in external components (after the nomination got persisted to the api-server). Since pods consume both node resources and DRA resources, it's important to persist both at the same (or almost the same) point in time.

This is consistent with the current implementation: ResourceClaim allocation is stored in status in PreBinding phase, therefore in conjunction to node nomination it effectively allows to reserve a complete set of resources (both node and DRA) to enable their correct accounting.

Note that node nomination is set before WaitOnPermit phase, but ResourceClaim status gets published in PreBinding, therefore pods waiting on WaitOnPermit would have only nominations published, and not ResourceClaim statuses. This however is not considered an issue, as long as there are no in-tree plugins supporting WaitOnPermit, and the Gang Scheduling feature is starting in alpha. This means that the fix to this issue will block Gang Scheduling promotion to beta.

#### Increasing the load to kube-apiserver

Setting a NominatedNodeName is an additional API call that then multiple components in the system
need to process. In the extreme case when this is always set before binding the pod, this would
double the number of API calls from scheduler, which isn't really acceptable from scalability and
performance reasons.

To mitigate this problem, we:
- skip setting `NNN` when all `Permit` and `PreBind` plugins have no work to do for this pod.
(We'll discuss how-to in the later section.)

For cases with delayed binding, we make an argument that the additional calls are acceptable, as
there are other calls related to those operations (e.g. PV creation, PVC binding, etc.) - so the
overhead of setting `NNN` is a smaller percentage of the whole e2e pod startup flow.

## Design Details

<!--
This section should contain enough information that the specifics of your
change are understandable. This may include API specs (though not always
required) or even code snippets. If there's any ambiguity about HOW your
proposal will be implemented, this is the place to discuss them.
-->
### The scheduler puts `NominatedNodeName`

The scheduler needs to update `NominatedNodeName` with the node that it determines the pod is going to at the beginning of binding cycles.

As discussed at [Increasing the load to kube-apiserver](#increasing-the-load-to-kube-apiserver), we should set `NominatedNodeName` only when some Permit plugins (at WaitOnPermit) or PreBind plugins work. 

We can know when there is Permit plugins that will work at WaitOnPermit or not by the status returned from Permit() functions.
If one or more Permit() returned `Wait` status, we have to put `NominatedNodeName` at the beginning of binding cycles, before actually starting to wait at WaitOnPermit.

And, for PreBind plugins, we need to add a new function to `PreBindPlugin`.

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

### The scheduler's cache for `NominatedNodeName`

Here, we'll ensure that works for non-existing nodes too and if those nodes won't appear in the future, it won't leak the memory.

The scheduler stores `NominatedNodeName` data at [`nominator`](https://github.com/kubernetes/kubernetes/blob/master/pkg/scheduler/backend/queue/nominator.go).
This `nominator` holds `NominatedNodeName` data even if the node doesn't exist.
So, this caching mechanism should work correctly for non-existing NNN node scenario.

Also, this cached info is cleared 
[`deletePodFromSchedulingQueue`](https://github.com/kubernetes/kubernetes/blob/b2dfba4151b859c31a27fe31f8703f9b2b758270/pkg/scheduler/eventhandlers.go#L199).
This `deletePodFromSchedulingQueue` is called when unscheduled pods are removed, 
or pods are assigned to nodes (EventHandler calls `DeleteFunc` handler when [the condition](https://github.com/kubernetes/kubernetes/blob/b2dfba4151b859c31a27fe31f8703f9b2b758270/pkg/scheduler/eventhandlers.go#L416) is no longer met).

So, as a conclusion, there should be nothing to implement newly around it.
We'll ensure this scenario works correctly via tests.

#### The scheduler clears `NominatedNodeName` after scheduling failure

As of now the scheduler clears the `NominatedNodeName` field at the end of failed scheduling cycle, if it
found the nominated node unschedulable for the pod. This logic remains unchanged.

NOTE: The previous version of this KEP, that allowed external components to set `NominatedNodeName`, deliberately left the `NominatedNodeName` field unchanged after scheduling failure. With the KEP update for v1.35 this logic is being reverted, and scheduler goes back to clearing the field after scheduling failure.
 
### Kube-apiserver clears `NominatedNodeName` when receiving binding requests

We update kube-apiserver so that it clears `NominatedNodeName` when receiving binding requests.

### Handling ResourceClaim status updates

Since ResourceClaim status update is complementary to node nomination (reserves resources in a similar way), it's desired that both will be set at the beginning of the PreBinding phase (before the pod starts waiting for resources to be ready for binding). The order of actions in the device management plugin is correct, however the scheduler performs the prebinding actions of different plugins sequentially. As a result it may happen that e.g. a long lasting PVC provisioning may delay exporting ResourceClaim allocation status. This is not desired, as it allows a gap in time when DRA resources are not reserved - causing problems similar to the ones originally fixed by this KEP - kubernetes/kubernetes#125491

### Test Plan

[x] I/we understand the owners of the involved components may require updates to
existing tests to make this code solid enough prior to committing the changes necessary
to implement this enhancement.

##### Prerequisite testing updates

##### Unit tests

- `k8s.io/kubernetes/pkg/scheduler`: `2025-10-15` - `70.8`
- `k8s.io/kubernetes/pkg/registry/core/pod/storage`: `2025-10-15` - `78.8`
- `k8s.io/kubernetes/pkg/apis/core/validation`: `2025-10-15` - `85.2`

##### Integration tests

We're going to add these integration tests:
- The scheduler prefers to picking up nodes based on NominatedNodeName on pods, if the nodes are available.
- The scheduler ignores NominatedNodeName reservations on pods when it's scheduling higher priority pods.
- The scheduler overwrites NominatedNodeName when it performs the preemption, or when it finds another spot in another node and proceeding to the binding cycle (assuming there's a PreBind plugin).
- The scheduler puts NominatedNodeName at the beginning of binding cycles if Permit or PreBind plugin will do some work.
  - And, the scheduler (actually kube-apiserver, when receiving a binding request) clears NominatedNodeName when the pod is actually bound.

Also, with [scheduler-perf](https://github.com/kubernetes/kubernetes/tree/master/test/integration/scheduler_perf), we'll make sure the scheduling throughputs for pods that go through Permit or PreBind don't get regress too much.
We need to accept a small regression to some extent since there'll be a new API call to set NominatedNodeName. 
But, as discussed, assuming PreBind already makes some API calls for the pods, the regression there should be small.

##### e2e tests

We won't implement any e2e tests because we can test everything with integration tests described above,
and an e2e test wouldn't add any additional value.

### Graduation Criteria

#### Beta 

- The feature is implemented behind the feature gate.
- The tests are implemented. 

## GA

- There are several official components starting to use this:
  - The cluster autoscaler starts to use this feature.
  - Kueue starts to use this feature.
- No negative feedback or bug.

### Upgrade / Downgrade Strategy

**Upgrade**

During the beta period, the feature gates `NominatedNodeNameForExpectation` and `ClearingNominatedNodeNameAfterBinding` are enabled by default, no action is needed.

**Downgrade**

Users need to disable the feature gates, and restart kube-scheduler and kube-apiserver.

On downgrade to the version that doesn't have this feature, there aren't any action that users need to take. For pods that have NominatedNodeName set, scheduler will try to honor it, but:
- if the pod is still not schedulable, it will clear the field
- if the pod is schedulable, but to a different node - it will also clear it (and potentially set it to a different value if preemption is needed)

### Version Skew Strategy

If kube-apiserver's version is older than kube-scheduler,
and doesn't have the implementation change from this KEP,
`NominatedNodeName` won't be cleared at the binding api call. 
But, ideally, users should use the same version of kube-scheduler and kube-apiserver.
For old kube-apiserver, the `NominateNodeName` will not be cleared on binding - this is fine,
because unsetting it is not critical for correctness, it's only done to reduce potential user confusion.

However, it's not that not clearing `NominatedNodeName` will actually cause something wrong in the scheduling flow, 
but, it's just that it might lead to a user's confusion, 
as discussed in [Confusion if `NominatedNodeName` is different from `NodeName` after all](#confusion-if-nominatednodename-is-different-from-nodename-after-all).

So, we can say the risk caused by this version difference would be fairly low.

On the other hand, if kube-scheduler's version is older than kube-apiserver,
and doesn't have the implementation change from this KEP,
nothing goes wrong because kube-apiserver just clears `NominatedNodeName` from the pods at the binding API,
which is fine by the today's scheduler implementation as well.

## Production Readiness Review Questionnaire

### Feature Enablement and Rollback

###### How can this feature be enabled / disabled in a live cluster?

- [x] Feature gate (also fill in values in `kep.yaml`)
  - Feature gate name: NominatedNodeNameForExpectation
  - Components depending on the feature gate: kube-scheduler
- [x] Feature gate
  - Feature gate name: ClearingNominatedNodeNameAfterBinding
  - Components depending on the feature gate: kube-apiserver


###### Does enabling the feature change any default behavior?

Pods that are processed by Permit or PreBind plugins get NominatedNodeName during binding cycles.

###### Can the feature be disabled once it has been enabled (i.e. can we roll back the enablement)?

Yes.
The feature can be disabled in Beta version by restarting the kube-scheduler and kube-apiserver with the feature-gates off.

###### What happens if we reenable the feature if it was previously rolled back?

The scheduler just again starts to put NominatedNodeName at the beginning of binding cycles (if applicable). 

###### Are there any tests for feature enablement/disablement?

No.
This feature is only changing when a NominatedNodeName field will be set - it doesn't introduce a new API. 
However reacting to it is purely in-memory, so enablement/disablement tests wouldn't really differ from regular feature tests.

### Rollout, Upgrade and Rollback Planning

###### How can a rollout or rollback fail? Can it impact already running workloads?

The scheduler and kube-apiserver are involved in this feature.

If upgrading the scheduler fails somehow, new Pods won't be scheduled anymore until rolling back,
while Pods, which are already scheduled, won't be affected.
If upgrading kube-apiserver fails somehow, the whole Kubernetes will not be able to function properly until rolling back.

Even if one of them cannot be upgraded properly somehow, and gets rolled back,
there'll be nothing behaving wrong in the scheduling flow, see [Version Skew Strategy](#version-skew-strategy).

###### What specific metrics should inform a rollback?

- The `schedule_attempts_total` metric with the `error` label is increasing abnormally.
- The `scheduler_pod_scheduling_sli_duration_seconds` or `scheduling_attempt_duration_seconds` gets too long.
  - Although, for pods that have to go through Permit/PreBind plugins, it's expected that their scheduling+binding latency would get higher because of an additional API call for NominatedNodeName.

###### Were upgrade and rollback tested? Was the upgrade->downgrade->upgrade path tested?

TODO: update the test scenario

###### Is the rollout accompanied by any deprecations and/or removals of features, APIs, fields of API types, flags, etc.?

No.

### Monitoring Requirements

###### How can an operator determine if the feature is in use by workloads?

They can check that pods with delayed binding gets `NominatedNodeName`
while waiting for the scheduler to provision their resources at PreBind.

###### How can someone using this feature know that it is working for their instance?

- [x] Other (treat as last resort)
  - Details: `NominatedNodeName` on the pods during its scheduling period.

###### What are the reasonable SLOs (Service Level Objectives) for the enhancement?

We need to make sure the scheduling throughput doesn't get much regressed by this enhancements, especially for pods that go through PreBind or Permit.

The scheduling throughput depends on what types of pods are in your cluster,
and also what types of scheduler customization you add.

So, here we just give a hint of a reasonable SLO, you need to adjust it based on your cluster's usual behaviors.

In the default scheduler, we should see the throughput around 100-150 pods/s ([ref](https://perf-dash.k8s.io/#/?jobname=gce-5000Nodes&metriccategoryname=Scheduler&metricname=LoadSchedulingThroughput&TestName=load)), and this feature shouldn't bring any regression there.

Based on that: 
- `schedule_attempts_total` shouldn't be less than 100 in a second.
- the average of `scheduling_algorithm_duration_seconds` shouldn't be above 10 ms.

###### What are the SLIs (Service Level Indicators) an operator can use to determine the health of the service?

- [x] Metrics
  - `schedule_attempts_total` with `scheduled` label.
  - `scheduler_pod_scheduling_sli_duration_seconds` with `scheduled` label.

###### Are there any missing metrics that would be useful to have to improve observability of this feature?

No.

### Dependencies

No.

###### Does this feature depend on any specific services running in the cluster?

No.

### Scalability

###### Will enabling / using this feature result in any new API calls?

Yes.
- API call type: PATCH pods.
- estimated throughput: Each pod that goes through Permit or PreBind plugins triggers one additional API call.
In the default scheduler, pods with DRA or delayed binding PVC would be those.
- originating component: Kube-scheduler

###### Will enabling / using this feature result in introducing new API types?

No.

###### Will enabling / using this feature result in any new calls to the cloud provider?

No.

###### Will enabling / using this feature result in increasing size or count of the existing API objects?

No.

###### Will enabling / using this feature result in increasing time taken by any operations covered by existing SLIs/SLOs?

No.

###### Will enabling / using this feature result in non-negligible increase of resource usage (CPU, RAM, disk, IO, ...) in any components?

Yes - but it should be negligible impact.
The memory usage in kube-scheduler is supposed to increase because when `NominatedNodeName` is added on the pods, the scheduler's
internal component called `nominator` has to record them so that scheduling cycles can refer to them as necessary.

###### Can enabling / using this feature result in resource exhaustion of some node resources (PIDs, sockets, inodes, etc.)?

No.

### Troubleshooting

###### How does this feature react if the API server and/or etcd is unavailable?

The scheduler itself doesn't work anymore in that case.

###### What are other known failure modes?

Unknown.

###### What steps should be taken if SLOs are not being met to determine the problem?

Since SLOs can be impacted by multiple components and mechanisms in kubernetes, there is not straightforward algorithm to determine the problem. The general approach to investigating issues is described below.

If kube-scheduler SLOs are not being met, we should first check if other components of kubernetes (e.g. kube-apiserver) are experiencing slowdown or increased error rates as well. If that is the case, we should find out whether there is a global issue with an already-determined cause.
A longer turnaround in kube-apiserver handling API requests may result in rising values of `scheduling_algorithm_duration_seconds` and lower values of `schedule_attempts_total`.

If we suspect that there is an ongoing problem inside kube-scheduler and that it is triggered by handling nominated node names, we should check kube-scheduler logs for failed scheduling of pods that had been waiting for preemption of victims, or for failed binding of pods that have nominated node name set - and investigate further.

## Implementation History

- 7th May 2025: The initial KEP is submitted.
- 31st Jul 2025: The enhancement was demoted to alpha, because it haven't met all beta requirements for v1.34.
- 9th Oct 2025: The enhancement was promoted to beta, with the scope narrowed down to allow setting `NominatedNodeName` only in the kube-scheduler, having other components (e.g. Cluster Autoscaler or Karpenter) use the field as read-only.

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
#### Introduce a new field

Instead of using `NominatedNodeName` to let external components to hint scheduler, we considered
introducing a dedicated field for that purpose. However, as discussed above, we don't have any
clear usecases where distinguishing the source of the setting really matters and with multiple
external components it doesn't eliminate the potential races either. If in the future we realize
that distinsuighing that is needed, we believe that we can model such state muchine with an
additional field in a purely additive way.

#### Allow NominatedNodeName to be set by other components

In v1.35 this feature is being narrowed down to one-way communication: only kube-scheduler is allowed to set `NominatedNodeName`,
while for other components this field should be read-only.

The alternative to consider for future releases is that other components can set `NominatedNodeName` in pending pods to indicate
the pod is preferred to be scheduled on a specific node.

##### Motivation: External components want to specify a preferred pod placement

The ClusterAutoscaler or Karpenter internally calculate the pod placement,
and create new nodes or un-gate pods based on the calculation result.
The shape and count of newly added nodes assumes some particular pod placement
and the pods may not fit or satisfy scheduling constraints if placed differently.

By specifying their expectation on `NominatedNodeName`, the scheduler can first check
whether the pod can go to the nominated node, reducing end-to-end scheduling time.

##### Goals

- Make sure external components can use `NominatedNodeName` to express where they prefer the pod is going to.
  - Probably, you can do this with a today's scheduler as well. This proposal wants to discuss/make sure if it actually works, and then add tests etc.

##### Non-Goals

- External components can enforce the scheduler to pick up a specific node via `NominatedNodeName`.
  - `NominatedNodeName` is just a hint for scheduler and doesn't represent a hard requirement

##### User stories

The use case supported by this feature is:
- The ClusterAutoscaler or Karpenter sets `NominatedNodeName` after creating a new node for pending pod(s), so that the scheduler
   can utilize the result of scheduling simulations already calculated by those components

###### Story 1: ClusterAutoscaler or Karpenter can influence scheduling decisions

ClusterAutoscaler or Karpenter perform scheduling simulations to decide what nodes should be
added to make pending pods schedulable. Their decisions assume a certain placement - if pending
pods are placed differently, they may not fit on the newly added nodes or may not satisfy their
scheduling constraints.

In order to improve the end-to-end pod startup latency when cluster scale-up is needed, we need a
mechanism to communicate the results of scheduling simulations from ClusterAutoscaler or Karpenter
to scheduler.

###### Story 2: Kueue specifies `NominatedNodeName` to indicate where it prefers pods being scheduled to

Kueue supports scheduling features that are not (yet) supported in core scheduling, such as topology-aware scheduling. 
When it determines the optimal placement, it needs a mechanism to pass that information to the scheduler.
Currently it is using NodeSelector to enforce placement of pods and only then ungates the pods. Scheduler doesn't take that information into account until pods are ungated and can schedule other pods in those places in the meantime. 
It would be beneficial to pass that information to scheduler sooner, as well as allow scheduler to change the decision if the topology constraints are just the soft ones.

##### Risks and Mitigations

###### NominatedNodeName can be set by other components now.

There aren't any guardrails preventing other components from setting `NominatedNodeName` now.
In such cases, the semantic is not well defined now and the outcome of it may not match user
expectations.

This section is a step towards clarifying this semantic instead of maintaining status-quo.

###### Confusing semantics of `NominatedNodeName`

Up until now, `NominatedNodeName` was expressing the decision made by scheduler to put a given
pod on a given node, while waiting for the preemption. The decision could be changed later so
it didn't have to be a final decision, but it was describing the "current plan of record".

If we put more components into the picture (e.g. ClusterAutoscaler and Karpenter), we effectively
get a more complex state machine, with the following states:

1. pending pod
2. pod proposed to node (by external component) [not approved by scheduler]
3. pod nominated to node (based on external proposal) and waiting for node (e.g. being created & ready)
4. pod nominated to node and waiting for preemption
5. pod allocated to node and waiting for binding
6. pod bound

The important part is that if we decide to use `NominatedNodeName` to store all that information,
we're effectively losing the ability to distinguish between those states.

We may argue that as long as the decision was made by the scheduler, the exact reason and state
probably isn't that important - the content of `NominatedNodeName` can be interpreted as
"current plan of record for this pod from scheduler perspective".

But the `pod proposed to node` state is visibly different. In particular external components
may overallocate the pods on the node, those pods may not match scheduling constraints etc.
We can't claim that it's a current plan of record of the scheduler. It's a hint that we want
scheduler to take into account.

In other words, from state machine perspective, there is visible difference in who sets the
`NominatedNodeName`. If it was scheduler, it may mean that there is already ongoing preemption.
If it was an external component, it's just a hint that may even be ignored.
However, if we look from consumption point of view - these are effectively the same. We want
to expose the information, that as of now a given node is considered as a potential placement
for a given pod. It may change, but for now that's what considered. 

Eventually, we may introduce some state machine, where external components could also approve
schedulers decisions by exposing these states more concretely via the API. But we will be
able to achieve it in an additive way by exposing the information about the state.

However, we don't need this state machine now, so we just introduce the following rules:
- Any component can set `NominatedNodeName` if it is currently unset.
- Scheduler is allowed to overwrite `NominatedNodeName` at any time in case of preemption or
the beginning of the binding cycle.
- No external components can overwrite `NominatedNodeName` set by a different component.
- If `NominatedNodeName` is set, the component who set it is responsible for updating or
clearing it if its plans were changed (using PUT or APPLY to ensure it won't conflict with
potential update from scheduler) to reflect the new hint.

Moreover:
- Regardless of who set `NominatedNodeName`, its readers should always take that into
consideration (e.g. ClusterAutoscaler or Karpenter when trying to scale down nodes).
- In case of faulty components (e.g. overallocation the nodes), these decisions will
simply be rejected by the scheduler (although the `NominatedNodeName` will remain set
for the unschedulability period).

###### Race condition

If an external component adds `NominatedNodeName` to the pod that is going through a scheduling cycle,
`NominatedNodeName` isn't taken into account (of course), and the pod could be scheduled onto a different node.

But, this should be fine because, either way, we're not saying `NominatedNodeName` is something
forcing the scheduler to pick up the node, rather it's just a preference.


###### What if there are multiple components that could set `NominatedNodeName` on the same pod

It's not something newly introduced by this KEP because anyone can set NominatedNodeName today,
but discuss here to form our suggestion. 

Multiple controllers might keep overwriting NominatedNodeName that is set by the others. 
Of course, we can regard that just as user's fault though, that'd be undesired situation.

There could be several ideas to mitigate, or even completely solve by adding a new API.
But, we wouldn't like to introduce any complexity right now because we're not sure how many users would start using this,
and hit this problem.

So, for now, we'll just document it somewhere as a risk, unrecommended situation, 
and in the future, we'll consider something
if we actually observe this problem getting bigger by many people starting using it.

###### Invalid `NominatedNodeName` prevents the pod from scheduling

Currently, `NominatedNodeName` field is cleared at the end of failed scheduling cycle if it found the nominated node
unschedulable for the pod. However, in order to make it work for ClusterAutoscaler and Karpenter, we will remove this
logic, and `NominatedNodeName` could stay on the node forever, despite not being a valid suggestions anymore.
As an example, imagine a scenario, where ClusterAutoscaler created a new node and nominated a pod to it, but
before that pod was scheduled, a new higher-priority pod appeared and used the space on that newly created node.
In such a case, it all worked as expected, but we ended up with `NominatedNodeName` set uncorrectly.

As a mitigation:
- an external component that originally set the `NominatedNodeName` is responsible for clearing or updating
the field to reflect the state
- if it won't happen, given that `NominatedNodeName` is just a hint for scheduler, it will continue to processing
the pod just having a minor performance hit (trying to process a node set via `NNN` first, but falling back to
all nodes anyway). We claim that the additional cost of checking `NominatedNodeName` first is acceptable (even
for big clusters where the performance is critical) because it's just one iteration of Filter plugins
(e.g., if you have 1000 nodes and 16 parallelism (default value), the scheduler needs around 62 iterations of
Filter plugins, approximately. So, adding one iteration on top of that doesn't matter).

#### Confusion if `NominatedNodeName` is different from `NodeName` after all

If an external component adds `NominatedNodeName`, but the scheduler picks up a different node,
`NominatedNodeName` is just overwritten by a final decision of the scheduler.

But, if an external component updates `NominatedNodeName` that is set by the scheduler, 
the pod could end up having different `NominatedNodeName` and `NodeName`.

We will update the logic so that `NominatedNodeName` field is cleared during `binding` call

We believe that ensuring that `NominatedNodeName` can't be set after the pod is already bound
is niche enough feature that doesn't justify an attempt to strengthening the validation.

##### Design Details

If we take into account external components setting `NominatedNodeName`, the design needs to be extended as following:

###### External components put `NominatedNodeName`

There aren't any restrictions preventing other components from setting NominatedNodeName as of now.
However, we don't have any validation of how that currently works.
To support the usecases mentioned above we will adjust the scheduler to do the following:
- if NominatedNodeName is set, but corresponding Node doesn't exist, kube-scheduler will NOT clear it when the pod is unschedulable [assuming that a node might appear soon]
- We will rely on the fact that a pod with NominatedNodeName set is resulting in the in-memory reservation for requested resources. 
Higher-priority pods can ignore it, but pods with equal or lower priority don't have access to these resources. 
This allows us to prioritize nominated pods when nomination was done by external components. 
We just need to ensure that in case when NominatedNodeName was assigned by an external component, this nomination will get reflected in scheduler memory.

We will implement integration tests simulating the above behavior of external components.

###### The scheduler only modifies `NominatedNodeName`, does not clear it in any case

As of now, scheduler clears the `NominatedNodeName` field at the end of failed scheduling cycle if it
found the nominated node unschedulable for the pod. However, this won't work if ClusterAutoscaler or Karpenter
would set it during scale up.

In the most basic case, the node may not yet exist, so clearly it would be unschedulable for the pod.
However, potential mitigation of ignoring non-existing nodes wouldn't work either in the following case:

1. Pods are unschedulable. For the simplicity, let's say all of them are rejected by NodeResourceFit plugin. (i.e., no node has enough CPU/memory for pod's request)
2. CA finds them, calculates nodes necessary to be created
3. CA puts `NominatedNodeName` on each pod
4. The scheduler keeps trying to schedule those pending pods though, here let's say they're unschedulable (no cluster event happens that could make pods schedulable) until the node is created.
5. The nodes are created, and registered to kube-apiserver. Let's say, at this point, nodes have un-ready taints.
6. The scheduler observes `Node/Create` event, `NodeResourceFit` plugin QHint returns `Queue`, and those pending pods are requeued to activeQ.
7. The scheduling cycle starts handling those pending pods.
8. However, because nodes have un-ready taints, pods are rejected by `TaintToleration` plugin.
9. The scheduler clears `NominatedNodeName` because it finds the nominated node (= new node) unschedulable.

In order to avoid the above scenarios, we simply remove the clearing logic. This means that scheduler
will never clear the `NominatedNodeName` - it may update it though if based on its scheduling algorithm
it decides to ignore the current value of `NominatedNodeName` and put it on a different node (either to
signal the preemption, or record the decision before binding as described in the above sections).

##### Test plan: Integration tests

We're going to add these integration tests:
- The scheduler doesn't clear NominatedNodeName when the nominated node isn't available and the pod is unschedulable.
  - And, once the nodes appears, the pod with NNN set is scheduled there (even if there are other equal-priority pending pods).

Also, with [scheduler-perf](https://github.com/kubernetes/kubernetes/tree/master/test/integration/scheduler_perf), we'll make sure the scheduling throughputs for pods that go through Permit or PreBind don't get regress too much.
We need to accept a small regression to some extent since there'll be a new API call to set NominatedNodeName. 
But, as discussed, assuming PreBind already makes some API calls for the pods, the regression there should be small.


## Infrastructure Needed (Optional)

<!--
Use this section if you need things from the project/SIG. Examples include a
new subproject, repos requested, or GitHub details. Listing these here allows a
SIG to get the process for these resources started right away.
-->
