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
    - [AI Training Job as preemptor](#ai-training-job-as-preemptor)
    - [Preemption of Multihost Inference](#preemption-of-multihost-inference)
    - [Preemption of Multihost Inference that can run in a degraded mode](#preemption-of-multihost-inference-that-can-run-in-a-degraded-mode)
    - [Preemption cost](#preemption-cost)
  - [Notes/Constraints/Caveats](#notesconstraintscaveats)
  - [Risks and Mitigations](#risks-and-mitigations)
- [Design Details](#design-details)
  - [Preemption unit](#preemption-unit)
  - [Pod Group priorities](#pod-group-priorities)
  - [Preemption algorithm](#preemption-algorithm)
    - [In place victim reprieval](#in-place-victim-reprieval)
  - [Pod Group Post Filter](#pod-group-post-filter)
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
- [X] "Implementation History" section is up-to-date for milestone
- [X] User-facing documentation has been created in [kubernetes/website], for publication to [kubernetes.io]
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

This KEP is tightly coupled with [KEP-4671: Gang Scheduling using Workload Object] one. It is building
on foundations introduced there and assumes the knowledge of the concepts introduces there. 

We believe that providing a Gang Scheduling capability without a dedicated preemption mechanism for it
is of no use for the high performance batch workloads for which this feature is primarily designed.

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
cost of searching the potential space, especially in big clusters, this is not a hard rule, but rather
a goal that we will try to optimize for.

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
  
The rest of this KEP explains those pieces in more detail.

### User Stories

#### Preemption of AI Training job

When running an AI Training job, I want to ensure that it will not be partially preempted.
If at least one my pods is not running, the others are not making progress anyway and are
just wasting the resources in the cluster.

#### AI Training Job as preemptor

When scheduling an AI Traning Job, I want preemption to ensure that the
whole Training Job can fit on to a cluster. I want to avoid partial preemptions for single pods
from my training job if they cannot guarantee that the whole Job will become schedulable.

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

This story is treated directionally and will not be addressed in Alpha and Beta.


### Notes/Constraints/Caveats

For alpha we defined a Workload Aware Preemption as a separate feature that can be disabled independently of the Gang Scheduling. We acknowledge that for Beta, releasing the Gang Scheduling without Workload Aware Preemption does not provide enough value for the end users. That's why in Beta we merge those two features and progress them together under single `GenericWorkload` feature gate.

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
decide to extend Workload API with CompositePodGroup concept - for more details see
[API Design For Gang and Workload-Aware Scheduling](https://tiny.cc/hvhs001)). However, we never expect preemption unit to
be larger than scheduling unit.

Based on that, we will extend the the existing `GangSchedulingPolicy` as following:

```golang
// DisruptionMode defines how individual entities within a group can be disrupted.
// Exactly one mode can be set.
//
// +union
type DisruptionMode struct {
	// Single specifies that children can be disrupted independently from each other.
	//
	// +optional
	Single *SingleDisruptionMode 

	// All specifies that all children can only be disrupted together.
	//
	// +optional
	All *AllDisruptionMode
}

// SingleDisruptionMode specifies that children can be disrupted independently.
type SingleDisruptionMode struct {
	// Intentionally empty now.
}

// AllDisruptionMode specifies that children can only be disrupted together.
type AllDisruptionMode struct {
	// Intentionally empty now.
}

type PodGroupSpec struct {
    // Existing field(s).
    
    // DisruptionMode defines the mode in which a given PodGroup can be disrupted.
    // Controllers are expected to fill this field by copying it from a PodGroupTemplate.
    // One of Single, All. Defaults to Single if unset.
    // This field is immutable.
    // This field is available only when the GangScheduling feature gate
    // is enabled.
    DisruptionMode *DisruptionMode 
}
```

Given that preemption unit shouldn't be larger then the scheduling unit, additional validation
will be added to prevent `All` disruption mode for PodGroups with BasicSchedulingPolicy. 

While the `PreemptionMode` might seem the more natural name here, we envision that the same
concept can be later used in `Eviction` API and other usecases, so we already start with a more
generic name to avoid future confusion.

### Pod Group priorities

Prioritizing pod groups across each other requires answering the question: "What is pod group priority?".
Up until now, only individual pods have assigned priority. For homogenous `PodGroup` the priority of a PodGroup
should be the same priority as the priority of individual pods that form the group.

We can also think about heterogenous `PodGroup` where individual pods have different priorities. However, that
case rises following questions:
- what should be the priority of the `PodGroup` in the scheduling queue. 
- what should be the priority of the `PodGroup` when it is considered as a preemption victim.

For the first question the natural answer is that each Pod should become a separate scheduling unit, with the
priority. With that, the pods from `PodGroup` should also become a separate PreemptionUnits. In that case, one can ask why should we join such pods in a `PodGroup` in the first place. As we do not find an use case for that, we will continue with the assumption that all pods, even heterogenous, within `PodGroup` should have the same priority. 

Additionally, as described in user stories above, a simple static priority doesn't seem to be enough. Arguably it is
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

We acknowledge that it might be misleading to users. For Alpha, we described
the possible divergence in the documentation.

For Beta and GA, we will disallow the divergence between the priority of the `Pod` and the `PodGroup`. 
This will be done by the scheduler, which will fail scheduling of the `PodGroup`, once it observes such divergence.
This mechanism will follow a similar mechanism already implemented in scheduler that disallows `PodGroup` with pods having different `spec.schedulerName`.
This information will be visible to the user in the description of the `PodScheduled` `Conditions` in the `Pod.Status`

```yaml
status:
  conditions:
  - type: PodScheduled
    status: "False"
    reason: SchedulerError
    message: 'all pods in a single pod group should match the priority of the pod group, got: 1 and 2'
```

We might relax this restriction in the future if there is a strong use cases that justify it. 

It's worth mentioning here, that we want to introduce the same defaulting rules for
`PodGroup.Spec.PriorityClassName` that we have for pods. Namely, if `PriorityClassName` is unset
and there exists PriorityClass marked as `globalDefault`, we default it to that value.
This consistency will allow us to properly handle cases when users set neither pods
nor PodGroup priorities.

Note that, for workload-aware preemption we will support the `preemptionPolicy` being part
of requestion `PriorityClass` - namely both currently existing modes: `PreemptLowerPriority`
and `Never`.

As the `preemptionPolicy` is also a field of the Pod, we will apply the same constraints to this field as we will
for the priority. Namely, all pods within `PodGroup` will have to share the same `preemptionPolicy`. This will be enforced on the scheduler level.

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

If that appears not being enough, we will similarly extend the `CompositePodGroup` API in a follow up.
In such case, we expect that for any pod, the priority of a pod from the perspective of being a preemption victim
would be the priority of a smallest unit encompassing it in the `CompositePodGroup` tree.

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

The above algorithm achieves our principles, as by reprieving the highest priority pods first, it
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
   - for pod group G, we will start with just one "whole cluster"; 

1. For every domain computed above run the following points:

   1. Identify the list of all potential victims in that domain:
      - all running pod groups with (preemption) priority lower then preemptor priority; note that
        some pods from that pod group may be running outside of currently considered domain D - they
        need to contribute to scoring, but they won't contribute to feasibility of domain D.
      - all individual pods with priority lower then preemptor priority

   1. If removing all potential victims would not make the preemptor schedulable, the preemptor
      is unschedulable with preemption in currently considered domain D.

   1. For the [Topology-Aware Scheduling] "checking whether preemptor becomes schedulable" yields a list 
      of potential placements. For the alpha support of TAS, we will only assume the best placement. With the
      progression of TAS to Beta, we might change the algorithm to consider multiple placements.
      We can also see a future where if the algorithm is performant enough, even for the non TAS case,
      we will generate multiple potential placements and consider them. 

   1. For each placement computed above run the following points:

      1.  Sort all the potential victims to reflect their "importance" (from the most important to the
          least ones). Tentatively, the function will sort first by their priority, and within a single
          priority prioritizing pod groups over individual pods.

      1.  Perform best-effort reprieval of pod groups and pods violating PodDisruptionBudgets. We achieve
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

      1.  For the remaining potential victims, using binary search across priorities (not across the list
          of victims) find the minimal priority N for which scheduling the preemptor can be achieved
          without preempting any victims with priority higher than N. This allows to reduce the potential
          cascading preemptions later.

      1.  Eliminate all victims from the potential victims list that have priority higher than N.

      1.  Schedule and assume the preemptor (assuming that all remaining potential victims are removed).

      1.  Iterate over the list of potential victims (in the order achieved with sorting above) checking
          if they can be placed where they are currently running. If so assume it back and remove from
          potential victims list.

      We acknowledge the fact that the above algorithm is not optimal, but (a) is compatible with the
      current pod-based one, (b) is computationally feasible, (c) is simple to reason about. We will
      proceed with it and may consider improvements in a follow-up KEPs in the future.

1. We score scheduling decisions for selected number of the domains/placements and choose the best one. For cluster wide 
   preemption without TAS there will be only one decision. For multiple placments we will reuse (and generalize) the [preemption scoring functions]. During promotion of [Topology Aware Scheduling] to beta we will consider the exact scoring criteria.

We acknowledge that some users might favor more disruptive preemptions if they allow the workload to be placed in a more optimal way, especially when the Topology-Aware Scheduling comes into the picture. This can be addressed by custom scoring criteria for the placements in the future.

It's worth noting that as structured, this algorithm addresses all four cases mentioned above that
we want to support and is compatible with the current pod-based preemption algorithm. This means
we will be able to achieve in-place replacement with relatively localized changes.


[Topology Aware Scheduling]: https://github.com/kubernetes/enhancements/tree/master/keps/sig-scheduling/5732-topology-aware-workload-scheduling
[preemption scoring functions]: https://github.com/kubernetes/kubernetes/blob/c180d6762d7ac5059d9b50457cafb0d7f4cf74a9/pkg/scheduler/framework/preemption/preemption.go#L702-

#### In place victim reprieval

The algorithm described above assumes that we can cheaply check if a potential victim pod can be placed back where they are currently running after preemptor has been `assumed` on the target node. However there is no extension point like this implemented in the scheduler as of right now. During scheduling, the scheduler uses Filter plugins to check whether a pod can run on a given node. This process is heavy and requires to build a `cycleState` object, particularly for plugins that have inter pod constraints such as pod affinity (pod running on node A can fail "Filter" on node B). Running `Filter()` and building a `cycleState` for all of the victims does not seem like a feasible option from performance perspective.

However, one can note that actually do not need a full Filter check. Instead of checking if the victim pod can be scheduled back on its node, we need to check whether keeping the victim pod on a node will break the schedulability of the preemptor that we assumed. Mainly, this means that we do not have to meet the scheduling constraints for victim pod, and we only care about scheduling constraints of the preemptor pod that could be broken. Such check can be implemented by extending Filter plugins with a new method.
We propose to tenantively call it `Reprieve` with following semantics for Beta:

```go 
// ReprieveExtension is an interface that is included in Filter plugins that allow specifying
// reprieve method used by workload aware preemption.
type ReprieveExtension interface {
    // Reprieve is called by the workload aware preemption.
    // All FilterPlugins should return "Success" to declare that
    // the given victim pod can be placed back on the given node
    // without breaking the schedulability of the preemptor pod on this node.
    // If Reprieve does not return "Success", it will return "Unschedulable"
    // or "Error".
    //
    // "Error" aborts preemption.
    //
    // For the node being evaluated Filter plugins should look at the passed
    // nodeInfo reference for this particular node's information instead of
    // looking it up in the NodeInfoSnapshot because during preemption
    // the state of the node can be mutated to evaluate the possibility of  preempting 
    // them to schedule preemptor pods. The same stands for all nodes belonging 
    // to the cluster, as they might be out of sync with NodeInfoSnapshot.

    // Plugins are encouraged to check the context for cancellation.
    // Once canceled, they should return as soon as possible with
    // an Unschedulable status that includes the
    // `context.Cause(ctx)` error explanation.
    Reprieve(ctx context.Context, victimPod *v1.Pod, nodeInfo NodeInfo,  preemptorPods []*v1.Pod clusterNodes []NodeInfo) *Status
}

// FilterPlugin is an interface for Filter plugins. These plugins are called at the
// filter extension point for filtering out hosts that cannot run a pod.
// This concept used to be called 'predicate' in the original scheduler.
// These plugins should return "Success", "Unschedulable" or "Error" in Status.code.
// However, the scheduler accepts other valid codes as well.
// Anything other than "Success" will lead to exclusion of the given host from
// running the pod. Plugins that implement FilterPlugin should
// also implement SignPlugin to enable batching optimizations.
type FilterPlugin interface {
    Plugin
    // Filter is called by the scheduling framework.
    // All FilterPlugins should return "Success" to declare that
    // the given node fits the pod. If Filter doesn't return "Success",
    // it will return "Unschedulable", "UnschedulableAndUnresolvable" or "Error".
    //
    // "Error" aborts pod scheduling and puts the pod into the backoff queue.
    //
    // For the node being evaluated, Filter plugins should look at the passed
    // nodeInfo reference for this particular node's information (e.g., pods
    // considered to be running on the node) instead of looking it up in the
    // NodeInfoSnapshot because we don't guarantee that they will be the same.
    // For example, during preemption, we may pass a copy of the original
    // nodeInfo object that has some pods removed from it to evaluate the
    // possibility of preempting them to schedule the target pod.
    //
    // Plugins are encouraged to check the context for cancellation.
    // Once canceled, they should return as soon as possible with
    // an UnschedulableAndUnresolvable status that includes the
    // `context.Cause(ctx)` error explanation. For example, the
    // context gets canceled when a sufficient number of suitable
    // nodes have been found and searching for more isn't necessary
    // anymore.
    Filter(ctx context.Context, state CycleState, pod *v1.Pod, nodeInfo NodeInfo) *Status 
    // ReprieveExtension returns a ReprieveExtension interface if the plugin implements one,
    // or nil if it does not.
    ReprieveExtension() ReprieveExtension
}
```

We expect that for Filter plugins that do not enforce inter pod constraints the implementation should be trivial. We will
provide an implementation for all the in-tree Filter plugins. We expect that the owners of out-of-tree plugins will 
follow with the implementation of `Reprieve` for their plugins.

For Beta, the workload aware preemption, thus the gang scheduling will be disabled if any of the Filter plugins registered does not support the Reprieve extension.

For GA, we state that the Reprieve method is required for an effective workload aware preemption. That's why we will promote the Reprieve extension to become a method of the FilterPlugin interface as follows:

```go
// FilterPlugin is an interface for Filter plugins. These plugins are called at the
// filter extension point for filtering out hosts that cannot run a pod.
// This concept used to be called 'predicate' in the original scheduler.
// These plugins should return "Success", "Unschedulable" or "Error" in Status.code.
// However, the scheduler accepts other valid codes as well.
// Anything other than "Success" will lead to exclusion of the given host from
// running the pod. Plugins that implement FilterPlugin should
// also implement SignPlugin to enable batching optimizations.
type FilterPlugin interface {
    Plugin
    // Filter is called by the scheduling framework.
    // All FilterPlugins should return "Success" to declare that
    // the given node fits the pod. If Filter doesn't return "Success",
    // it will return "Unschedulable", "UnschedulableAndUnresolvable" or "Error".
    //
    // "Error" aborts pod scheduling and puts the pod into the backoff queue.
    //
    // For the node being evaluated, Filter plugins should look at the passed
    // nodeInfo reference for this particular node's information (e.g., pods
    // considered to be running on the node) instead of looking it up in the
    // NodeInfoSnapshot because we don't guarantee that they will be the same.
    // For example, during preemption, we may pass a copy of the original
    // nodeInfo object that has some pods removed from it to evaluate the
    // possibility of preempting them to schedule the target pod. The same stands for
    // all nodes belonging to the cluster, as they might be out of sync with NodeInfoSnapshot.
    //
    // Plugins are encouraged to check the context for cancellation.
    // Once canceled, they should return as soon as possible with
    // an UnschedulableAndUnresolvable status that includes the
    // `context.Cause(ctx)` error explanation. For example, the
    // context gets canceled when a sufficient number of suitable
    // nodes have been found and searching for more isn't necessary
    // anymore.
    Filter(ctx context.Context, state CycleState, pod *v1.Pod, nodeInfo NodeInfo) *Status

    // Reprieve is called by the preemption.
    // All FilterPlugins should return "Success" to declare that
    // the given victim pod can be placed back on the given node
    // without breaking schedulability of the preemtor pod on this node.
    // If Reprieve does not return "Success", it will return "Unschedulable"
    // or "Error".
    //
    // "Error" aborts preemption.
    //
    // For the node being evaluated Filter plugins should look at the passed
    // nodeInfo reference for this particular node's information instead of
    // looking it up in the NodeInfoSnapshot because during preemption
    // the state of the node can be mutated to evaluate the possibility of  preempting 
    // them to schedule preemptor pods.

    // Plugins are encouraged to check the context for cancellation.
    // Once canceled, they should return as soon as possible with
    // an Unschedulable status that includes the
    // `context.Cause(ctx)` error explanation.
    Reprieve(ctx context.Context, victimPod *v1.Pod, nodeInfo NodeInfo, preemptorPods []*v1.Pod, clusterNodes []NodeInfo) *Status
}
```

### Pod Group Post Filter

As part of minimizing preemptions goal, arguably the most important thing to do is to avoid unnecessary preemptions. However, with the current model of preemption when preemption is triggered immediately after the victims are decided and PostFilter is run per Pod in PodGroup, it doesn't achieve this goal. The reason for that is that the proposed placement (nomination) can actually appear to be invalid and not be proceeded with, if the whole PodGroup fails to schedule. In such case we will not even proceed to binding and the preemption will be completely unnecessary disruption.

For alpha, to avoid triggering unnecessary preemptions, we disabled the default preemption plugin in PostFilter for pods from PodGroup if the workload aware preemption was enabled.

For beta, we acknowledge that the default preemption is not the only PostFilter plugin out there. Other PostFilter plugins can also perform disruptive actions. In the current model, those plugins works only on the outcome of single pod scheduling cycle, within a PodGroup cycle. With that they do not have a full picture of the pod group scheduling outcome and can perform actions that are either not optimal or in the worst case will not make the PodGroup schedulable anwyay.

For beta and GA we propose to disable all PostFilter plugins in the pod group scheduling cycle in favor of the newly added PodGroupPostFilter extension point. This extension point will be called only once after the whole pod group fails to schedule. 
It will provide data about the outcome of whole PodGroup scheduling cycle and allow users to define actions that can be taken to make the PodGroup schedulable. Workload Aware Preemption will be one of the implementations of this extension point. As part of the beta promotion we will provide the implementation for other in tree plugin that implements the PostFilter interface (namely the DRA Plugin). We expect owners of out of tree PostFilters to follow with their own implementations.

```go
// PodGroupPostFilterResult stores information about nominated nodes for a pod group.
type PodGroupPostFilterResult struct {
    NominatedNodeNames map[*v1.Pod]*fwk.NominatingInfo
}

// PodGroupPostFilterPlugin is an interface for "PodGroupPostFilter" plugins. These plugins are called
// after a PodGroup cannot be scheduled.
type PodGroupPostFilterPlugin interface {
    fwk.Plugin

    // PodGroupPostFilter is called by the scheduling framework
    // when the pod group scheduling cycle failed.
    //
    //
    // A PodGroupPostFilter plugin should return one of the following statuses:
    // - Unschedulable: the plugin gets executed successfully but the PodGroup cannot be made schedulable.
    // - Success: the plugin gets executed successfully and the PodGroup can be made schedulable.
    // - Error: the plugin aborts due to some internal error.
    //
    // Informational plugins should be configured ahead of other ones, and always return Unschedulable status.
    // Optionally, a non-nil PodGroupPostFilterResult may be returned along with a Success status. For example,
    // a preemption plugin may choose to return nominatedNodeName, so that framework can reuse that to update the
    // preemptor pod's status.nominatedNodeName field.
    PodGroupPostFilter(ctx context.Context, pg *v1alpha2.PodGroup, pods []*v1.Pod, pgSchedulingFunc PodGroupSchedulingFunc) (*PodGroupPostFilterResult, *fwk.Status)
}
```

### Potential future extensions

Here we discuss a couple of extensions that we envision just to ensure that we can build them
in an additive and backward-compatible way. The approval of this KEP doesn't mean an approval
for any of those and proceeding with any of these will require dedicated KEP(s) in the future.

1. Improved preemption algorithm.

   Instead of considering a single placement of a preemptor for a given set of victims, we may
   consider multiple different placements. This will have much bigger impact once kube-scheduler
   supports topology-aware scheduling. As a result, we're leaving it as a future extension -
   the algorithm can always be improved and will result in pretty local code changes.

1. Non-uniform priority across CompositePodGroups.

   We anticipate that in the future the `CompositePodGroup` concept will be introduced. We can envision
   a case where different `CompositePodGroups` will require to have different preemption priorities.
   To achieve that, we could introduce `PriorityClassName` field also at the `CompositePodGroup` level, with the semantic that lower-level structure overwrites the higher-level one (e.g. priority set for `PodGroup` overwrites the priority
   for `CompositePodGroup`). So the API and semantics proposed in this KEP would allow for achieving
   it in backward compatible way.

1. Non-uniform (Composite)PodGroups

   In addition to non-uniform priorities, we may expect other non-uniform behaviors. As an
   example consider `LeaderWorkerSet` and a usecase where we allow for preempting individual
   workers (with a given unit working in a degraded mode), but don't allow for preempting a
   leader. The struct based `DisruptionMode` allows for introducing more sophisticated policies
   (e.g. only a subset of `CompositePodGroups` can be preempted).

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

1. PodGroupPostFilter extension
   
   The PodGroupPostFilter interface proposed in this KEP contains the most important 
   information about PodGroup that failed scheduling cycle. We can extend this interface in the future
   to allow plugins to have more insights into why a given pod group cannot be scheduled. One example of that
   would be an extension similar to the NodeToStatusReader object passed to a PostFilter plugins.


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

For Alpha we implemented integration tests to ensure basic functionalities of workload preemption:

- Pods from a single PodGroup with `DisruptionMode=Single` can be preempted individually by the
  higher priority PodGroup
- Pods from a single PodGroup with `DisruptionMode=All` are preempted all together even
  when preempting a single pod would be enough to free up the space for the higher priority
  PodGroup
- Pods from a single PodGroup with `DisruptionMode=Single` can be preempted individually by the
  higher priority individual pod.
- Pods from a single PodGroup with `DisruptionMode=All` are preempted all together even
  when preempting a single pod would be enough to free up the space for the higher priority
  individual pod.

Those tests are located at [podgrouppreemption_test.go](https://github.com/kubernetes/kubernetes/blob/e136f39334a72b7d35069a97d373ccaa0211dcae/pkg/scheduler/framework/preemption/podgrouppreemption_test.go).

For Beta we will expand the set of integration tests to cover the new Reprieval method. Namely, we will
make sure that we have scenarios that covers the use of all in tree Filter plugins during the workload aware preemption.
We also aim to have a parity with all the existing test scenarios for the default preemption.

##### e2e tests

For alpha, given the new functionality is limited to kube-scheduler change and API extensions,
we will rely on integration tests described above (as easier and faster to run and debug).

For beta we will add a new e2e test to the sig scheduling tests defined in [test/e2e/scheduling/preemption.go](https://github.com/kubernetes/kubernetes/blob/e136f39334a72b7d35069a97d373ccaa0211dcae/test/e2e/scheduling/preemption.go).
Those tests will cover the four basics basic functionalities describe in previous section.

For GA we will promote those tests to conformance.

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

- Decision about additional sorting/scoring preemption victims to minimize preemption cost
- Decision whether we support mutability of Priority for GA
- E2E test promoted to conformance
- Performance benchmarks have well defined thresholds and are run as part of the scheduler-perf of sig-scalability-benchmarks
- All known issues resolved 


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
  - Feature gate name: GenericWorkload
  - Components depending on the feature gate: kube-apiserver, kube-scheduler

Note that for Alpha this feature was using `WorkloadAwarePreemption` feature gate.
For Beta and GA we decided to merge it together with the `GenericWorkload` feature gate,
with the rationale provided in the rest of the KEP. 

###### Does enabling the feature change any default behavior?

Yes - the preemption victims chosen when scheduling a pod group will be chosen using a slightly
modified version of the algorithm. Thus the exact set of victims may slightly differ.

The bigger changes in preemption victims may appear when pod groups start using `PodGroup`
disruption mode, however that requires an explicit opt-in from the user (or controller)
creating the `PodGroup` object.

###### Can the feature be disabled once it has been enabled (i.e. can we roll back the enablement)?

Yes, the preemption algorithm changes can be disabled by simply disabling the feature gate
in kube-scheduler. However, this requires also disabling a Gang Scheduling feature.

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

The API fields related to Workload Aware Preemption are no longer hidden behind a separate
feature gate and will be promoted with the whole API to beta. There is no need for the
dedicated enablement/disablement tests at the kube-apiserver registry layer

### Rollout, Upgrade and Rollback Planning

<!--
This section must be completed when targeting beta to a release.
-->

###### How can a rollout or rollback fail? Can it impact already running workloads?

Workloads that do not use the `Workload` and `PodGroup `APIs should not be impacted, since the functionality remains unchanged for them. During a rolling upgrade, if the active scheduler instance has the feature disabled, it will schedule pods using the standard pod-by-pod method, falling back to a default PostFilter methods. A default preemption algorithm will treat all pods as single preemption units, even when they are a part of a PodGroup with `All` Disruption Mode.

This results in a fallback to the status quo behavior, meaning that pods will be still scheduled, but PodGroup-level scheduling constraints and preemption behavior won't be applied.

###### What specific metrics should inform a rollback?

- `scheduler_podgroup_preemption_attempts_total{result="error"}`: A sudden spike indicates internal errors or panics within 
the workload aware preemption logic.
- `scheduler_podgroup_preemption_attempt_duration_seconds`: A significant P99 latency would indicate that the performance of the new logic is unacceptable.


###### Were upgrade and rollback tested? Was the upgrade->downgrade->upgrade path tested?

We'll perform manual testing of the upgrade -> downgrade -> upgrade path using the following sequences to
verify workload aware preemption where Pod Group is either preemptor or a victim:

For the Pod Group is a preemptor case we will use the following sequence:

1. Start a local Kubernetes v1.36 cluster with `GenericWorkload` feature gate disabled (default 
behavior).
2. Fill the cluster with low priority idle pods so there is no room for new pods.
3. Attempt to create a Pod with `spec.schedulingGroup` set.
4. The `spec.schedulingGroup` field is dropped by the API server. The pod is created successfully
   but without the `schedulingGroup` reference, resulting in immediate standard scheduling (one-by-one).
5. Restart/Upgrade API Server and Scheduler to v1.37 with feature gate enabled.
6. Create two PodGroup objects: `gang-test-A` and `gang-test-B` (both with `minCount=2`).
6. Create a Pod `test-pod-1` with `spec.schedulingGroup` pointing to `gang-test-A`.
7. The Pod stays in `Pending` state (waiting for the gang). Verify that
   `scheduler_pending_entities{type="podgroup", queue="gated"}` metric is incremented.
8. Create a Pod `test-pod-2` pointing to the same pod group.
9. Both pods are scheduled successfully in the same cycle (Gang Scheduling with Workload Aware Preemption works).
10. Verify that `scheduler_podgroup_preemption_attempts_total` metric is incremented.
10. Downgrade API Server and Scheduler to v1.36 with feature gate disabled.
11. Create `test-pod-3` pointing to `gang-test-B`. Note: We use a pod group created in step 5 because creating new
    PodGroup objects is disabled.
12. The pod is scheduled immediately (PodGroup logic is ignored because the schedulingGroup field is dropped by
    the v1.36 API server). If Gang Scheduling were active, this pod would hang pending waiting for a second member.
13. Verify that `preemption_attempts_total` was increased and the `scheduler_podgroup_preemption_attempts_total` metric
    did not increase.
13. Upgrade API Server and Scheduler back to v1.37 with feature gate enabled.
14. Create `test-pod-4` and `test-pod-5` pointing to `gang-test-B`; verifying that Gang Scheduling functionality is
    restored (these pods wait for `minCount=2` before scheduling).
15. Verify that the `scheduler_podgroup_preemption_attempts_total` metric was increased and `preemption_attempts_total`
    was not increased.

For the Pod Group is a victim case we will use the following sequence:

This sequence assumes that the pods and nodes are define in a way that ensures that only one pod can fit onto a node.

1. Start a local Kubernetes v1.36 cluster with `GenericWorkload` feature gate disabled (default 
behavior).
2. Attempt to create two low priority Pods with `spec.schedulingGroup` set.
3. The `spec.schedulingGroup` field is dropped by the API server. The pod are created successfully
   but without the `schedulingGroup` reference.
4. Create a high priority pod with NodeName set to a node of one of the low priority Pods. The pod is scheduled successfully and it preempts one of the low priority Pods.
5. Restart/Upgrade API Server and Scheduler to v1.37 with feature gate enabled.
6. Create two PodGroup objects: `gang-test-A` and `gang-test-B` (both with `minCount=2` and `PreemptionMode: All` and `Priority: Low`).
7. Create two low piority pods `test-pod-1` and `test-pod-2` with `spec.schedulingGroup` pointing to `gang-test-A`.
8. Create a high priority pod that with NodeName set to a node name of `test-pod-1`. Verify that preemption preempted both `test-pod-1` and -`test-pod-2`. 
9. Downgrade API Server and Scheduler to v1.36 with feature gate disabled.
10. Create `test-pod-3` and `test-pod-4`  pointing to `gang-test-B`. Note: We use a pod group created in step 6 because creating new PodGroup objects is disabled.
11. Create a high priority pod that with NodeName set to a node name of `test-pod-3`. Verify that preemption preempted only `test-pod-3` and not `test-pod-4`. 
12. Upgrade API Server and Scheduler back to v1.37 with feature gate enabled.
13. Create low priority pods `test-pod-5` and `test-pod-6` pointing to `gang-test-B`.
14. Create a high priority pod that with NodeName set to a node name of `test-pod-5`. Verify that preemption preempted `test-pod-5` and `test-pod-6`. 

###### Is the rollout accompanied by any deprecations and/or removals of features, APIs, fields of API types, flags, etc.?

No.

### Monitoring Requirements

<!--
This section must be completed when targeting beta to a release.

For GA, this section is required: approvers should be able to confirm the
previous answers based on experience in the field.
-->

###### How can an operator determine if the feature is in use by workloads?

Operators can check the new `scheduler_podgroup_preemption_attempts_total` metric. A value greater than zero indicates that the scheduler is processing Workload Aware Preemption.

###### How can someone using this feature know that it is working for their instance?

<!--
For instance, if this is a pod-related feature, it should be possible to determine if the feature is functioning properly
for each individual pod.
Pick one more of these and delete the rest.
Please describe all items visible to end users below with sufficient detail so that they can verify correct enablement
and operation of this feature.
Recall that end users cannot usually observe component logs or access metrics.
-->

- [X] Events
  - Event Reason: Preempting
  - The preemption message will have "cluster" set as a node name.

###### What are the reasonable SLOs (Service Level Objectives) for the enhancement?

Since there are no formal SLOs for the kube-scheduler apart from scalability SLOs, we define the objectives for this
feature primarily in terms of non-regression to ensure the workload aware preemption does not degrade the performance of the workload scheduling which in term would degrade the performance of the standard scheduling loop.

- Scheduling Throughput: There should be no significant regression in the system-wide scheduling throughput (pods/s) 
  when scheduling pods attached to a PodGroup that requires preemption compared to scheduling an equivalent number of individual pods that would also require preemption.
  This can be measured by the number of Pod binding API calls arriving to the API server
  (`apiserver_request_total{resource="pods", subresource="binding"}`).
- Scheduling Throughput: There should be no significant regression in the system-wide scheduling throughput (pods/s) 
  when scheduling pods requires preemption of pods grouped in PodGroups (DisruptionMode = All) compared to scheduling an equivalent number of pods in PodGroups that would require preemption of similar number of pods but not grouped in PodGroup.
  This can be measured by the number of Pod binding API calls arriving to the API server
  (`apiserver_request_total{resource="pods", subresource="binding"}`).


###### What are the SLIs (Service Level Indicators) an operator can use to determine the health of the service?

- [X] Metrics
  - Metric name: 
    - scheduler_podgroup_preemption_attempts_total
    - scheduler_podgroup_preemption_attempt_duration_seconds
    - scheduler_podgroup_preemption_victims
    - plugin_execution_duration_seconds{extension_point=Reprieval}
  - Components exposing the metric: kube-scheduler

###### Are there any missing metrics that would be useful to have to improve observability of this feature?

No. 

### Dependencies

<!--
This section must be completed when targeting beta to a release.
-->

###### Does this feature depend on any specific services running in the cluster?

No dependencies other than the components where the feature is implemented (kube-apiserver and kube-scheduler).

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

The behavior is consistent with the status quo. The removal of the victims selected by the Workload-Aware Preemption uses the same code as the standard preemption. Since the scheduler cannot remove pods or update them, any pod removal attempts that are an outcome of preemption will fail, not making space for the preemptor pods. Preemptor pods will not be marked with the NominatedNodeName. 

When the call to delete victim pods fails, the preemptor is moved to active queue (with async preemption) or backoff/unschedulable queue (without async preemption). 

Calls to update NominatedNodeNames for preemptor pods are using PatchPodStatus function which implements a retry mechanism and is shared by all occurrences that require updating pod status from scheduler. 

###### What are other known failure modes?

1. WorkloadAwarePreemption takes too long halting scheduling loop
- Detection: High values for metric: `scheduler_podgroup_preemption_attempt_duration_seconds`
- Mitigations: If intended, delete the PodGroup object and recreate the pods without `schedulingGroup`
  to disable gang scheduling and workload aware preemption (fallback to best-effort scheduling and default preemption) if acceptable.
- Diagnostics: Scheduler logs at V=6 searching for logs from podgrouppreemption.go file to trace where preemption slows down.
- Testing: The scheduler performance benchmarks should catch potential issues with a poor performance of the workload aware preemption

1. WorkloadAwarePreemption does not remove low level pods to make a place for the preemptor
- Detection: Check Pod Events/Status. Expected reason: a message indicating why preemption failed
- Metrics: `scheduler_podgroup_preemption_attempts_total{result=error}`
- Mitigations:
  - Scale up the cluster (add nodes) or delete other real-workloads to free up space.
  - If intended, delete the PodGroup object and recreate the pods without `schedulingGroup`
    to disable gang scheduling (fallback to best-effort scheduling) if acceptable.
- Diagnostics:
  - Scheduler logs at V=6 searching f or logs from podgrouppreemption.go file to see detailed reasons why the workload aware preemption failed.
- Testing:
  - Covered by integration tests

1. WorkloadAwarePreemption removes more pods than necessary 

- Detection: The amount of pods with status `preempted by podgroup X` is higher than expected for a given pod group
- Mitigations: If intended, delete the PodGroup object and recreate the pods without `schedulingGroup`to disable gang  
  scheduling and workload aware preemption (fallback to best-effort scheduling and default preemption) if acceptable.
- Diagnostics: Search for log line (V6) `Pods are potential preemption victims on domain`. This line is outputted after
  each failed victim reprieval.
- Testing: The scheduler performance benchmarks should catch potential issues with a poor performance of the workload aware preemption

###### What steps should be taken if SLOs are not being met to determine the problem?

## Implementation History

2025-11: Initial KEP-5710 proposal.
2026-02: KEP-5710 created for WAP alpha release.
2026-02: KEP-5710 updated to sync with decoupling of PodGroup/Workload API.
2026-05: KEP updated to promote to beta in v1.37.

## Drawbacks

There are already multiple implementations of Gang Scheduling with Gang Preemption in
the kubernetes ecosystem. However, we believe that workload awarness is critical enough
that it deserves standardizing in core Kubernetes.

## Alternatives

 One alternative considered as a short-term workaround for unnecessary preemption was the introduction of "delayed preemption". The proposed delayed preemption mechanism was structured as follows:

1. Modify the `DefaultPreemption` plugin to just compute preemptions, without actuating them.

2. Extend the `PostFilterResult` to include a set of victims (in addition to the existing
   `NominationInfo`). This will allows to clearly decouple the computation from actuation.

3. For individual pods (not being part of a workload), adjust the scheduling framework
   implementation of `schedulingCycle` to actuate preemptions of returned victims if calling
   `PostFilter` plugins resulted in finding a feasible placement.

4. For pods being part of a workload, rely on the Workload Scheduling Cycle.
   There are two subcases here:

   1. In the legacy case (without workload-aware preemption), `PostFilter` is called individually for
      every pod from a PodGroup. However, the victims computed for already the already processed
      pods may affect placement decisions for the next pods.
      To accommodate for that, if a set of victims was returned from a `PostFilter` in addition
      to keeping them for further actuation, they are additionally stored in `CycleState`.
      More precisely, the `CycleState` stores a new entry containing a map from
      a `nodeName` to a list of victims that were already chosen.
      With that, the `DefaultPreemption` plugin is extended to remove all already chosen
      victims from a given node before processing that node.

   2. In the target case (with workload-aware preemption), there is no longer a need to process
      pods individually, so the additional mutations of `CycleState` are not needed.

5. In both above cases, an additional step is introduced to the scheduling algorithm at the
   end. If a feasible placement for the PodGroup is found, all the victims are taken and their
   preemption is actuated. If a feasible placement was not found, the victims are dropped.
   In both cases, the scheduling of the whole PodGroup (all its pods) is marked as unschedulable
   and got back to the scheduling queue.

This alternative was dropped as we decided that workload aware preemption is crucial for the Gang Scheduling effort. As we tied those two efforts together, there is no need for additional alternative approach for minimzing disruptions in Gang Scheduling without WAP.

<!--
What other approaches did you consider, and why did you rule them out? These do
not need to be as detailed as the proposal, but should include enough
information to express the idea and why it was not acceptable.
-->

## Infrastructure Needed (Optional)

N/A
