# KEP-5710: Workload-aware preemption

<!-- toc -->
- [Release Signoff Checklist](#release-signoff-checklist)
- [Summary](#summary)
- [Motivation](#motivation)
  - [Goals](#goals)
  - [Non-Goals](#non-goals)
- [Proposal](#proposal)
  - [Principles](#principles)
  - [High-level approach](#high-level-approach)
  - [User Stories](#user-stories)
    - [Preemption of AI Training job](#preemption-of-ai-training-job)
    - [Preemption of Multihost Inference](#preemption-of-multihost-inference)
    - [Preemption of Multihost Inference that can run in a degraded mode](#preemption-of-multihost-inference-that-can-run-in-a-degraded-mode)
    - [Preemption cost](#preemption-cost)
  - [Notes/Constraints/Caveats (Optional)](#notesconstraintscaveats-optional)
  - [Risks and Mitigations](#risks-and-mitigations)
- [Design Details](#design-details)
  - [Preemption unit](#preemption-unit)
  - [Pod Group priorities](#pod-group-priorities)
  - [Preemption algorithm](#preemption-algorithm)
  - [Delayed preemption](#delayed-preemption)
  - [Potential future extensions](#potential-future-extensions)
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

- [X] (R) Enhancement issue in release milestone, which links to KEP dir in [kubernetes/enhancements] (not the initial KEP PR)
- [X] (R) KEP approvers have approved the KEP status as `implementable`
- [X] (R) Design details are appropriately documented
- [X] (R) Test plan is in place, giving consideration to SIG Architecture and SIG Testing input (including test refactors)
  - [ ] e2e Tests for all Beta API Operations (endpoints)
  - [ ] (R) Ensure GA e2e tests meet requirements for [Conformance Tests](https://github.com/kubernetes/community/blob/master/contributors/devel/sig-architecture/conformance-tests.md)
  - [ ] (R) Minimum Two Week Window for GA e2e tests to prove flake free
- [X] (R) Graduation criteria is in place
  - [ ] (R) [all GA Endpoints](https://github.com/kubernetes/community/pull/1806) must be hit by [Conformance Tests](https://github.com/kubernetes/community/blob/master/contributors/devel/sig-architecture/conformance-tests.md) within one minor version of promotion to GA
- [X] (R) Production readiness review completed
- [X] (R) Production readiness review approved
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

This KEP describes the changes to kube-scheduler to support workload-aware preemption. We focus
on the API, framework and building blocks, not the ideal algorithm - it can come as a follow up.
We start with simple implementation, that is heavily based on the existing pod preemption algorithm.

The `Workload` and `PodGroup` API introduced in [KEP-4671: Gang Scheduling using Workload Object] is extended to
allow expressing the concept of pod group priority and to define the preemption unit. With those
extensions we make the next step towards our workload-aware scheduling north star.

## Motivation

Tightly-coupled workloads can require ongoing communication between multiple pods to make progress.
While such usecases have always existed (e.g. MPI jobs), in the current AI era the number of such
workloads is much higher as both AI Training and Multihost Inference belong to this category.

With [KEP-4671: Gang Scheduling using Workload Object], we're making the first step towards better
handling this category of workloads. However, that KEP is focused only on the aspect of initial
scheduling of the workload. As mentioned above, the tightly-coupled workload usually requires ongoing
communication across many pods not only on startup, but also across its whole lifetime. This means
that disrupting a single pod of such workload effectively disrupts the whole workload - even if the
rest of the pods are still running, they are not able to make any progress.

In this KEP, we're proposing a solution for the first (but in many cases the primary) reason of such
disruptions - the preemption. Given the supply shortages of the newest and most efficient accelerators
as well as their economics, maximizing their utilization is one of the primary goals for majority of
users. They achieve it by mixing different kinds of workloads in the same Kubernetes cluster and
properly prioritizing these workloads against each other to satisfy the business requirements.
In such capacity-constraint environments, preemption is a critical feature allowing users to balance
the need to satisfy their business needs (often meaning satisfying certain SLOs for serving workloads)
with maximizing utilization of their hardware. However, the currently existing preemption mechanism
doesn't address those needs due its pod-centric nature.

Workload-aware preemption is not a novel thing and was already implemented in other ecosystem projects.
However, similarly to gang scheduling, we believe that workload awareness (which includes workload aware
preemption) is critical enough that deserves standardizing in core Kubernetes. Only this standardization
would allow us tighter integration with other features (managing other types of disruptions, autoscaling
and many others) and bring the true value for every Kubernetes user.

[KEP-4671: Gang Scheduling using Workload Object]: https://github.com/kubernetes/enhancements/tree/master/keps/sig-scheduling/4671-gang-scheduling

### Goals

- Define API for describing units of preemption within a workload
- Define API for describing priority of a preemption unit
- Describe the principles and semantics of workload-aware preemption
- Define the base preemption policies
- Define the scheduler changes needed to implement workload-aware preemption
- Provide full backward compatibility for all existing scheduling features

### Non-Goals

- Change the way how individual pods (not being part of workloads) are preempted
- Provide the most optimal preemption algorithm from day 1
- Address arbitrary preemption policies (more preemption policies will be needed,
  but these should be added in a followup KEPs)
- Introduce workload-awareness for handling different kinds of disruptions
  (e.g. caused by hardware failures) including kubelet eviction
- Design rescheduling for workloads that will be preempted (rescheduling will
  be addressed in a separate dedicated KEP)
- Change the preemption principle of avoiding preemption if a workload/pod can be scheduled without it.
  If we decide to change that it will be addressed in a dedicated KEP.
- Propose any tradeoff between preemption and cluster scale-up.
- Design workload-level preemption triggered by external schedulers

## Proposal

This KEP heavily depends on [KEP-4671: Gang Scheduling using Workload Object] one. It is building
on foundations introduced there and assumes the knowledge of the concepts introduces there.

### Principles

Preemption is critical, but it is causing disruption to workloads that are being preempted.
Disruption itself is never desired and this defines our core principles.

1. We should always minimize (the cost of) preemptions.
1. The cost of preempting a workload should include also the side effects of this preemption.
   In particular, if a preempted workload will immediately be recreated by a controller and
   will result in preempting another workload, the cost of that second preemption should be
   included in the cost of the original preemption.

While cascading preemptions are inevitable in some cases (e.g. if high priority preemptor pod group
has very strict placement requirements), in general if there are multiple options of scheduling
a higher priority pod group with preemptions, with some of them being expected to cause cascading
preemptions and others not, we will try to choose the later. However, due to computational
cost of searching the potential space, this is not a hard rule, but rather a goal that we will
try to optimize for.

### High-level approach

We start with a relatively simple design focusing on extensible APIs and semantics without
targeting very sophisticated algorithms. In this section we just introduce the individual
pieces of the solution and discuss them in more detail in the following sections.

1. We piggy-back on existing `PriorityClass` API to avoid reinventing the concept of priority
   from scratch.
1. We extend `Workload` and `PodGroup` API to allow for defining the preemption unit. Here we also start
   simple and allow for preemption unit to only correspond to a `PodGroup` in a gang mode.
1. We extend `Workload` and `Pod Group` API to allow for defining the priority of a pod group.
1. We start simple by just defining a single static priority used for scheduling and preemption
   While we envision both splitting them into two in the future or making it mutable, both of
   these can be achieved later in backward-compatible way.
1. We start with a simple sub-optimal preemption algorithm that is based on the existing
   pod preemption algorithm used by kube-scheduler.
1. We introduce a mechanism of "delayed preemption" to postpone actuation of preemption
   decisions until we really know that these are necessary. This is to prevent the situation
   when preemption is triggered but the pod/workload that triggered it in the end cannot
   be bound anyway due to inability to schedule the workload as a whole.

The rest of this KEP explains those pieces in more detail.

### User Stories

#### Preemption of AI Training job

When running an AI Training job, I want to ensure that it will not be partially preempted.
If at least one my pods is not running, the others are not making progress anyway and are
just wasting the resources in the cluster.

#### Preemption of Multihost Inference

When running multihost inference using LeaderWorkerSet, I want to ensure that its single
replica (one leader and N workers) will not be partially preempted. If at least one of
such pods is not running, the others are not able to serve and are just wasting resources
in the cluster.

#### Preemption of Multihost Inference that can run in a degraded mode

When running multihost inference using LeaderWorkerSet that can run in a degraded mode,
I don't want to preempt the whole replica (one leader and N workers) if a single worker
would be preempted, because I prefer to serve in a degraded mode than be completely
disrupted.

#### Preemption cost

When a long-running AI Training job, its cost of preemption differs over time and depends
on how long ago it checkpointed its state. I want to be able to somehow influence the
preemption priority of my job based on how much would it really cost me.

This story is treated directionally and will not be addressed at least in Alpha.


### Notes/Constraints/Caveats (Optional)

<!--
What are the caveats to the proposal?
What are some important details that didn't come across above?
Go in to as much detail as necessary here.
This might be a good place to talk about core concepts and how they relate.
-->

### Risks and Mitigations

1. Extensibility - it's obvious that what is proposed in this KEP will not be a final step and we
   will be evolving it. How can we ensure that we will not put ourselves into a corner.

   Mitigation: We enumerate potential extensions after the detailed design and briefly sketch
   how the proposed design can be extended to accommodate these.

1. Incompatible scheduler profiles - different scheduling profiles may enable different sets of
   plugins and if only subset of profiles enable `GangScheduling` plugin (responsible also for
   workload-aware preemption), we may break the expectations.

   Mitigation: We will document that `GangScheduling` plugin has to be enabled in all profiles
   or the logic will need to be reimplemented by other custom plugins. Eventually we may consider
   builtin validation, but we make it out of scope for this KEP.

1. Scalability - finding the optimal set of pod groups/pods to preempt is computationally expensive
   problem, however we need to ensure it can be used even in the largest Kubernetes clusters.

   Mitigation: We propose a simplified algorithm that is computationally feasible at the cost of
   providing "reasonably good" preemption victim candidates.


## Design Details

### Preemption unit

We start with defining what is a unit of preemption. While we can imagine usecases with preemption
unit being an arbitrary group of pods, in majority of real world usecases this is actually aligned
with the scheduling unit. In other words, the group of pods that should be preempted together matches
a group that was initially scheduled together as a gang.

Trying to formalize it, we define `WorkloadPortion` as one of {all pods in a PodGroup replica or
a single pod}. With that definition both scheduling unit and preemption units can only be
`WorkloadPortions`.

In the future, we may want to support usecases when a single scheduling unit consists of multiple
preemption groups, but we leave that usecase as a future extension (it can be addressed when we
decide to extend Workload API with PodSubGroup concept - for more details see
[API Design For Gang and Workload-Aware Scheduling]). However, we never expect preemption unit to
be larger than scheduling unit.

Based on that, we will extend the the existing `GangSchedulingPolicy` as following:

```golang
// DisruptionMode describes the mode in which a PodGroup can be disrupted (e.g. preempted).
// +enum
type DisruptionMode string

const (
  // DisruptionModePod means that individual pods can be disrupted or preempted independently.
  // It doesn't depend on exact set of Pods currently running in this PodGroup.
  DisruptionModePod = "Pod"
  // DisruptionModePodGroup means that the whole PodGroup replica needs to be disrupted or
  // preempted together.
  DisruptionModePodGroup = "PodGroup"
)

type PodGroupSchedulingPolicy struct {
    // Existing field(s).

    // DisruptionMode defines the mode in which a given PodGroup can be disrupted.
    // One of Pod, PodGroup.
    // Defaults to Pod if unset.
    //
    // This field is immutable.
    DisruptionMode *DisruptionMode
}
```

Given that preemption unit shouldn't be larger then the scheduling unit, additional validation
will be added to prevent `PodGroup` disruption mode for PodGroups with BasicSchedulingPolicy.

While the `PreemptionMode` might seem the more natural name here, we envision that the same
concept can be later used in `Eviction` API and other usecases, so we already start with a more
generic name to avoid future confusion.

### Pod Group priorities

Prioritizing pod groups across each other requires answering the question: "What is pod group priority?".
Up until now, only individual pods have assigned priority. But nothing prevents individual pods
forming a `PodGroup` from being heterogeneous and having different priorities.

Intuitively, the priority of a `PodGroup` in such case would be the "minimum of priorities of pods
that belong to it" - any pod with a priority higher than that can e.g. preempt our pod group. But
intuition is not enough here.

As described in user stories above, a simple static priority doesn't seem to be enough. Arguably it is
not even a single priority because a priority used for scheduling can be different than the priority
that should be used for preemption. So in the ideal world a workload owner should be able to:

- define priority used for scheduling a PodGroup
- define priority used for preemption of a PodGroup
- mutate preemption priority during the whole lifecycle of the workload to reflect the importance
  of that workload at a given moment

However, while we believe that all of these are eventually needed, we start simpler by:
- starting with just a single priority for scheduling and preemption.
- starting with static preemption priority (mutability brings additional complexity that is
  purely additive and thus should be added in a follow-up KEP)

In [KEP-4671: Gang Scheduling using Workload Object] we already decided that PodGroup is the scheduling
unit for workload-aware scheduling. Different PodGroups (even if part of the same Workload) are
scheduled independently. As a result, we continue this path and define the priority also at the level
of a scheduling unit.

The proposed `PodGroup` API extensions look as following.

```golang
type PodGroupTemplate struct {
    // Existing field(s).

    // PriorityClassName, if specified, indicates the priority that should be
    // considered when scheduling this pod group. "system-node-critical"
    // and "system-cluster-critical" are two special keywords which indicate the
    // highest priorities with the former being the highest priority. Any other
    // name must be defined by creating a PriorityClass object with that name.
    // If not specified, the priority will be default or zero if there is no
    // default.
    //
    // The authoritative priority for this pod group is expressed via the
    // 'priority' field.
    //
    // This field is immutable.
    PriorityClassName *string
}

type PodGroupSpec struct {
    // Existing field(s).

    // PriorityClassName, if specified, indicates the priority that
    // should be considered when scheduling this pod group. "system-node-critical"
    // and "system-cluster-critical" are two special keywords which indicate the
    // highest priorities with the former being the highest priority. Any other
    // name must be defined by creating a PriorityClass object with that name.
    // If not specified, the priority will be default or zero if there is no
    // default.
    //
    // The authoritative priority for this pod group is expressed via the
    // 'priority' field.
    //
    // This field is immutable.
      PriorityClassName *string
}
```

With that change, when scheduling or preempting a pod that is part of a Pod Group, the
priority defined in the `PodGroup` object will be used (and priority defined in the `Pod`
itself will be ignored, thus not reflecting the actual pod priority).

We acknowledge that it might be misleading to users. For Alpha, we will simply just
describe the possible divergence in the documentation.

For Beta, we will decide if we need additional actions, e.g.:
- expose the information about divergence in the API by introducing a new `Conditions` field
  in the `PodGroup.Status` with dedicated condition like `PodsNotMatchingPriority` that will
  be set by either kube-scheduler or a new workload-controller whenever it observes pods
  referencing a given `PodGroup` object which priority doesn't match the priority of the
  PodGroup object.
- introduce an admission to validate that if a `Pod` is referencing a `PodGroup` object, its
  `pod.Spec.PriorityClassName` equals to `podGroup.Spec.PriorityClassName`. However, we allow
  creating pods before the PodGroup object, and there doesn't seem to be an easy way to avoid
  races.
- making `pod.Spec.PriorityClassName` and `pod.Spec.Priority` mutable fields and having a new
  workload controller responsible for reconciling these. However, that could introduce another
  divergence between the priority of pods and the priority defined in the PodTemplate in true
  workload object (e.g. Job) which would introduce a similar level of confusion to users.
However, decision if we need any of these will be made when graduating this feature to Beta.

It's worth mentioning here, that we want to introduce the same defaulting rules for
`PodGroup.Spec.PriorityClassName` that we have for pods. Namely, if `PriorityClassName` is unset
and there exists PriorityClass marked as `globalDefault`, we default it to that value.
This consistency will allow us to properly handle cases when users set neither pods
nor PodGroup priorities.

Note that, for workload-aware preemption we will support the `preemptionPolicy` being part
of requestion `PriorityClass` - namely both currently existing modes: `PreemptLowerPriority`
and `Never`.

Given that components operate on integer priorities, we will introduce a corresponding fields
that reflect priority of a PodGroup (similarly to how it's done in Pod API).
Since it is effectively a derivative of the field introduced above it would be tempting to
put that into `PodGroup.Status`. However, for the consistency with the Pod API we actually
will put that next to the `PriorityClassName` in the spec:

```golang
type PodGroupTemplate struct {
    // Existing field(s).

    // PriorityClassName, if specified, indicates the priority that should be
    // considered when scheduling this pod group. "system-node-critical"
    // and "system-cluster-critical" are two special keywords which indicate the
    // highest priorities with the former being the highest priority. Any other
    // name must be defined by creating a PriorityClass object with that name.
    // If not specified, the priority will be default or zero if there is no
    // default.
    //
    // The authoritative priority for this pod group is expressed via the
    // 'priority' field.
    //
    // This field is immutable.
    PriorityClassName *string

    // Priority reflects the priority of the pod group.
    // The higher value, the higher the priority.
    // This field is populated from the PriorityClassName.
    Priority *int32
}

type PodGroupSpec struct {
    // Existing field(s).

    // PriorityClassName, if specified, indicates the priority that should be
    // considered when scheduling this pod group. "system-node-critical"
    // and "system-cluster-critical" are two special keywords which indicate the
    // highest priorities with the former being the highest priority. Any other
    // name must be defined by creating a PriorityClass object with that name.
    // If not specified, the priority will be default or zero if there is no
    // default.
    //
    // The authoritative priority for this pod group is expressed via the
    // 'priority' field.
    //
    // This field is immutable.
    PriorityClassName *string

    // Priority reflects the priority of the pod group.
    // The higher value, the higher the priority.
    // This field is populated from the PriorityClassName.
    Priority *int32
}
```

If that appears not being enough, we will similarly extend the `PodSubGroup` API in a follow up.
In such case, the priority of a pod would be the priority of a smallest unit in the `PodGroup`
object (PodGroup, PodSubGroup, ...) corresponding to this pod.


### Preemption algorithm

We start with describing at the high-level how existing pod-level preemption algorithm works.
Below, we will show how to generalize it to workloads.

If a pod P can be scheduled without triggering preemption, we don't consider preemption at all.
To check if a pod P can be scheduled on a given node with preemption we:

1. Identify the list of potential victims - all running pods with priority lower than the new pod P.

1. If removing all these victims would not make the node feasible, the node is infeasible.
   
1. From the list of potential victims, we try to reprieve (remove from the victims list) any pods
   whose eviction would violate PodDisruptionBudget.

1. From remaining potential victims, we start to reprieve pods starting from the highest priority
   and working down until the set of remaining victims still keeps the node feasible.

Once we find enough nodes feasible for preemption and list of victims for them, we score that and
choose the best options.

The above algorithm achieves our principles, as by eliminating highest priority pods first, it
effectively tries to minimize the cascading preemptions later.


We want to generalize the same algorithm to the workload case. However, the difference is not only
moving to the level of `PodGroup`, but also no longer operating at the level of individual nodes.
We need to look at the cluster as a whole. With that in mind, keeping the algorithm efficient
becomes a challenge, thus we modify to the approach below.

At the same time, we need to support four cases:
- individual pod as preemptor, individual pod(s) as victim(s)
- individual pod as preemptor, pod group(s) (and individual pod(s)) as victim(s)
- pod group as preemptor, individual pod(s) as victim(s)
- pod group as preemptor, pod group(s) (and individual pod(s)) as victim(s)

To achieve that, we don't want to multiply preemption algorithms and rather want to have a
unified high-level approach (with potential minor tweaks per option).

To check if a given preemptor (either (gang) PodGroup G or an individual pod P) can be scheduled
with preemption:

1. Split the cluster into mutually-exclusive domains where a preemptor will be put:
   - for pod P, it will always be individual nodes
   - for pod group G, we will start with just one "whole cluster"; eventually once we will have
     topology-aware scheduling, we will most probably inject some domain-based split here

1. For every domain computed above run the following points:

   1. Identify the list of all potential victims in that domain:
      - all running pod groups with (preemption) priority lower then preemptor priority; note that
        some pods from that pod group may be running outside of currently considered domain D - they
        need to contribute to scoring, but they won't contribute to feasibility of domain D.
      - all individual pods with priority lower the preemptor priority

   1. If removing all potential victims would not make the preemptor schedulable, the preemptor
      is unschedulable with preemption in currently considered domain D.

   1. Sort all the potential victims to reflect their "importance" (from the most important to the
      least ones). Tentatively, the function will sort first by their priority, and within a single
      priority prioritizing pod groups over individual pods.

   1. Perform best-effort reprieval of pod groups and pods violating PodDisruptionBudgets. We achieve
      it by scheduling and temporarily adding the preemptor to `nodeInfo` structure (assuming that
      all potential victims are removed), and then iterating over potential victims that would violate
      PodDisruptionBudget to check if these can be placed in the exact same place they are running now.
      If they can we simply leave them where they are running now and remove from the potential victims
      list.

      For domain D being a single node (current pod-based preemption), the above algorithm works
      identically to the current algorithm. For larger domains, different placements of a preemptor
      are potentially possible and may result in potentially different sets of victims violating
      PodDisruptionBudgets to remain feasible. This means that the proposed algorithm is not necessarily
      minimizing the number of victims that would violate their PodDisruptionBudgets.
      However, optimizing for it would be extremely expensive computationally so to not significantly
      hurt performance we propose to accept this limitation (if needed a better algorithm may be
      proposed as a separate KEP).

   1. For the remaining potential victims, using binary search across priorities (not across the list
      of victims) find the minimal priority N for which scheduling the preemptor can be achieved
      without preempting any victims with priority higher than N. This allows to reduce the potential
      cascading preemptions later.

   1. Eliminate all victims from the potential victims list that have priority higher than N.

   1. Schedule and assume the preemptor (assuming that all remaining potential victims are removed).

   1. Iterate over the list of potential victims (in the order achieved with sorting above) checking
      if they can be placed where they are currently running. If so assume it back and remove from
      potential victims list.

   We acknowledge the fact that the above algorithm is not optimal, but (a) is compatible with the
   current pod-based one, (b) is computationally feasible, (c) is simple to reason about. We will
   proceed with it and may consider improvements in a follow-up KEPs in the future.

1. We score scheduling decisions for each of the domains and choose the best one. For Alpha, we will
   reuse (and generalize) the [preemption scoring functions]. We will reconsider the exact criteria
   for Beta.

It's worth noting that as structured, this algorithm addresses all four cases mentioned above that
we want to support and is compatible with the current pod-based preemption algorithm. This means
we will be able to achieve in-place replacement with relatively localized changes.

[preemption scoring functions]: https://github.com/kubernetes/kubernetes/blob/c180d6762d7ac5059d9b50457cafb0d7f4cf74a9/pkg/scheduler/framework/preemption/preemption.go#L702-L714

### Delayed preemption

As part of minimizing preemptions goal, arguably the most important thing to do is to avoid unnecessary
preemptions. However, with the current model of preemption when preemption is triggered immediately
after the victims are decided (in `PostFilter`) doesn't achieve this goal. The reason for that is
that the proposed placement (nomination) can actually appear to be invalid and not be proceeded with.
In such case we will not even proceed to binding and the preemption will be completely unnecessary
disruption.

We're addressing it with what we call `delayed preemption` mechanism described in
[KEP-4671: Introduce Workload Scheduling Cycle].

However, there is one point that requires an update. Namely, how can we avoid or minimize subsequent
preemptions, if the previous scheduling attempt already triggered some preemptions.
We have the following two cases:

   1. If the original nomination is still feasible, we can try finding an alternative placement,
      but can't trigger a new preemption.

   1. If the original nomination is no longer feasible (e.g. some higher priority pods were scheduled
      there in the meantime), we reject the original placement (clear nominations) and start scheduling
      from scratch.

The remaining question is why in the second case we can start from scratch and ignore which exact
preemptions we did instead of trying to first expand the original, now (partially) occupied placement.
The rationale behind that is that we don't really care which exact preemptions were triggered by
the workload - what really matters is the current state of the cluster in which we want to minimize
the cost of additional preemptions. We need to run the preemption algorithm that assumes that all
triggered preemptions are finished and on top of that minimize the cost of additional preemptions.
In some cases it may mean expanding the previously freed up space, but it may not always be the case
(e.g. preempting medium priority small workload to expand the original placement may transitively
preempt low priority large workload elsewhere, preempting which would be just enough for us).

As a result, the only thing we need is additional scoring function(s) for choosing preemption victims
(effectively additional sorting criteria for choosing potential victims in the algorithm). With that,
which is highly desired even for single-step preemption, the subsequent preemption attempt can really
be done from scratch just in the new cluster state.

However, the additional sorting criteria will be added in Beta.

[KEP-4671: Introduce Workload Scheduling Cycle]: https://github.com/kubernetes/enhancements/pull/5730

[API Design For Gang and Workload-Aware Scheduling]: https://tiny.cc/hvhs001

### Potential future extensions

Here we discuss a couple of extensions that we envision just to ensure that we can build them
in an additive and backward-compatible way. The approval of this KEP doesn't mean an approval
for any of those and proceeding with any of these will require dedicated KEP(s) in the future.

1. Improved preemption algorithm.

   Instead of considering a single placement of a preemptor for a given set of victims, we may
   consider multiple different placements. This will have much bigger impact once kube-scheduler
   supports topology-aware scheduling. As a result, we're leaving it as a future extension -
   the algorithm can always be improved and will result in pretty local code changes.

1. Non-uniform priority across PodSubGroups.

   We anticipate that in the future the `PodSubGroup` concept will be introduced. We can envision
   a case where different `PodSubGroups` will require to have different priorities.
   To achieve that, we could introduce `PriorityClassName` field also at the `PodSubGroup` level, with the semantic that lower-level structure
   overwrites the higher-level one (e.g. priority set for `PodSubGroup` overwrites the priority
   for `PodGroup`). So the API and semantics proposed in this KEP would allow for achieving
   it in backward compatible way.

1. Non-uniform PodGroups

   In addition to non-uniform priorities, we may expect other non-uniform behaviors. As an
   example consider `LeaderWorkerSet` and a usecase where we allow for preempting individual
   workers (with a given unit working in a degraded mode), but don't allow for preempting a
   leader. The enum-based `DisruptionMode` allows for introducing more sophisticated policies
   (e.g. only a subset of `PodSubGroups` can be preempted).

1. Dynamic preemption priority

   As described above, the preemption priority of a running workload may actually vary over
   time. In such case, the controller owning a given workload may want to adjust its priority
   over time to reflect its important and cost of preemption.
   There are two primary extensions that we can do to achieve that:

   1. Make `PriorityClassName` mutable over time
   1. Add a new `PreemptionPriorityClassName` field that will be used when considering a given
      PodGroup for preemption (potentially also making it mutable).

   We believe that at least one of these (potentially both) will be needed in the future, but
   these all can be achieved in a purely additive way. Mutability is about relaxing validation
   and defining the semantics for how the mutations are consumed. An `PreemptionPriority` can
   also be added in backward-compatible way - if unset it just defaults to scheduling priority
   but a user has now an ability to overwrite it.

   In the later case, we will also need to avoid preemption cycle, which can be achieved by an
   additional constraint that preemption priority cannot be lower then scheduling priority. This
   ensures that if a given workload X was preempted by workload Y (scheduling(Y) > preemption(X)),
   it will not be able to preempt back workload Y because
   preemption(Y) >= scheduling(Y) > preemption(X) >= scheduling(X). This will work fine even if
   we make preemption priority mutable.

   However, given an ability to achieve both of these in backward compatible way later, we leave
   those for future extensions.

### Test Plan

[X] I/we understand the owners of the involved components may require updates to
existing tests to make this code solid enough prior to committing the changes necessary
to implement this enhancement.

##### Prerequisite testing updates

N/A

##### Unit tests

- `pkg/apis/scheduling/v1alpha1`: `2026-01-29` - `83.3%`
- `pkg/registry/scheduling/workload`: `2026-01-29` - `76.5%`
- `pkg/registry/scheduling/workload/storage`: `2026-01-29` -  `83.3%`
- `pkg/scheduler/framework/plugins/defaultpreemption`: `2026-01-29` - `84.9%`
- `pkg/scheduler/framework/runtime`: `2026-01-29` - `81.5%`

##### Integration tests

<!--
This question should be filled when targeting a release.
For Alpha, describe what tests will be added to ensure proper quality of the enhancement.

For Beta and GA, document that tests have been written,
have been executed regularly, and have been stable.
This can be done with:
- permalinks to the GitHub source code
- links to the periodic job (typically https://testgrid.k8s.io/sig-release-master-blocking#integration-master), filtered by the test name
- a search in the Kubernetes bug triage tool (https://storage.googleapis.com/k8s-triage/index.html)

- [test name](https://github.com/kubernetes/kubernetes/blob/2334b8469e1983c525c0c6382125710093a25883/test/integration/...): [integration master](https://testgrid.k8s.io/sig-release-master-blocking#integration-master?include-filter-by-regex=MyCoolFeature), [triage search](https://storage.googleapis.com/k8s-triage/index.html?test=MyCoolFeature)
-->

We will create integration test(s) to ensure basic functionalities of workload preemption:

- Pods from a single PodGroup with `DisruptionMode=Pod` can be preempted individually by the
  higher priority PodGroup
- Pods from a single PodGroup with `DisruptionMode=PodGroup` are preempted all together even
  when preempting a single pod would be enough to free up the space for the higher priority
  PodGroup
- Pods from a single PodGroup with `DisruptionMode=Pod` can be preempted individuallby by the
  higher priority individual pod.
- Pods from a single PodGroup with `DisruptionMode=PodGroup` are preempted all together even
  when preempting a single pod would be enough to free up the space for the higher priority
  individual pod.

##### e2e tests

Given the new functionality is limited to kube-scheduler change and API extensions,
we will rely on integration tests described above (as easier and faster to run and debug).


### Graduation Criteria

#### Alpha

- The API & feature is implemented behind the feature flag
- Base integration test showing preemption of whole PodGroup in the `PodGroup` mode

#### Beta
- Decision about additional sorting/scoring preemption victims to minimize preemption cost
- Decision about additional mechanisms for detecting/preventing divergence of priorities
  between Workload and its Pods.
- Decision whether we support mutability of Priority for Beta
- Extended performance benchmarks to ensure satisfying scalability & performance
- E2E test that can then be promoted to conformance
- All known issues resolved

#### GA

- TBD in for Beta release


### Upgrade / Downgrade Strategy

Standard procedures for features introducing new API fields should be used:

  - on upgrade, kube-apiservers should be upgraded first before kube-scheduler can
    use the new fields to opt-in for different preemption mode
  - on downgrade, kube-schedulers should be downgraded first (to stop using the new
    fields) before kube-apiservers are downgraded; note that downgrade of
    kube-apiserver(s) and/or disabling the new API fields will not clear their
    contents for objects already stored in the storage (etcd)

### Version Skew Strategy

Once kube-apiserver and kube-scheduler are involved in the feature.
The new API fields are needed to configure preemption behavior, thus kube-apiserver
is required to run in not older version than kube-scheduler.

However, the new preemption algorithm itself is purely in-memory and version skew
is not relevant for it.


## Production Readiness Review Questionnaire

### Feature Enablement and Rollback

###### How can this feature be enabled / disabled in a live cluster?

- [X] Feature gate (also fill in values in `kep.yaml`)
  - Feature gate name: WorkloadAwarePreemption
  - Components depending on the feature gate: kube-apiserver, kube-scheduler
- [ ] Other
  - Describe the mechanism:
  - Will enabling / disabling the feature require downtime of the control
    plane?
  - Will enabling / disabling the feature require downtime or reprovisioning
    of a node?

Note that this feature depends on the Workload API and gang-scheduling, so in addition
to the `WorkloadAwarePreemption` feature gate, the `GenericWorkload` and `GangScheduling`
feature gates also need to be enable. It is represented by adding dependency on those
in the feature-gate framework.

###### Does enabling the feature change any default behavior?

Yes - the preemption victims chosen when scheduling a pod group will be chosen using a slightly
modified version of the algorithm. Thus the exact set of victims may slightly differ.

The bigger changes in preemption victims may appear when pod groups start using `PodGroup`
disruption mode, however that requires an explicit opt-in from the user (or controller)
creating the Workload object.

###### Can the feature be disabled once it has been enabled (i.e. can we roll back the enablement)?

Yes, the preemption algorithm changes can be disabled by simply disabling the feature gate
in kube-scheduler.

The new API changes and admission can also be disabled by disabling the feature gate in
kube-apiserver. However keep in mind that it doesn't result in clearing the new fields
for objects that already have them set in the storage.

###### What happens if we reenable the feature if it was previously rolled back?

The feature starts working again.

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

The scheduler algorithm changes are purely in-memory and doesn't require any dedicated
enablement/disablement tests - the logic will be covered by regular feature tests.

For the newly introduced API fields, dedicated enablement/disablement tests at the
kube-apiserver registry layer will be added in Alpha.

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

###### Will enabling / using this feature result in any new API calls?

Not directly. However, with workload-aware preemption, more pods potentially
needs to be preempted (in PodGroup mode) to free up space for new workloads
to be scheduled.

###### Will enabling / using this feature result in introducing new API types?

No

###### Will enabling / using this feature result in any new calls to the cloud provider?

No

###### Will enabling / using this feature result in increasing size or count of the existing API objects?

Yes - new fields are added to the Workload API.
For `PriorityClassName`, `Priority`, `DisruptionMode` expected increase is O(130B) per PodGroupTemplate object in Workload object.
For `PriorityClassName`, `Priority`, `DisruptionMode` expected increase is O(130B) per PodGroup object.

###### Will enabling / using this feature result in increasing time taken by any operations covered by existing SLIs/SLOs?

Although we designed preemption with performance in mind, the scheduling latency (being part of Pod Startup SLO)
may potentially increase.
We will measure the exact impact using performance benchmarks and scalability tests and update the section based
on the results. The complexity of a single preemption cycle is O(#pods), which is comparable to the current algorithm,
so the benchmarks are primarily to validate the potential inefficiencies of the implementation.

###### Will enabling / using this feature result in non-negligible increase of resource usage (CPU, RAM, disk, IO, ...) in any components?

We don't expect non-negligible CPU increase for kube-scheduler, but it will be confirmed by tests.

###### Can enabling / using this feature result in resource exhaustion of some node resources (PIDs, sockets, inodes, etc.)?

No

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
