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
  - [Workload priorities](#workload-priorities)
  - [Preemption algorithm](#preemption-algorithm)
  - [Delayed preemption](#delayed-preemption)
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
  - [ ] (R) [all GA Endpoints](https://github.com/kubernetes/community/pull/1806) must be hit by [Conformance Tests](https://github.com/kubernetes/community/blob/master/contributors/devel/sig-architecture/conformance-tests.md) within one minor version of promotion to GA
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

This KEP describes the changes to kube-scheduler to support workload-aware preemption. We focus
on the API, framework and building blocks, not the ideal algorithm - it can come as a follow up.
We start with simple implementation, that is heavily based on the existing pod preemption algorithm.

The `Workload` API introduced in [KEP-4671: Gang Scheduling using Workload Object] is extended to
allow expressing the concept of workload priority and to define the preemption unit. With those
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
In such capacity-contraint environments, preemption is a critical feature allowing users to balance
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
- Define the scheduler changes needed to implement workload-aware preemption
- Provide full backward compatibility for all existing scheduling features

### Non-Goals

- Change the way how individual pods (not being part of workloads) are preempted
- Provide the most optimal preemption algorithm from day 1
- Address arbitrary preemption policies
- Introduce workload-awareness for handling different kinds of disruptions
  (e.g. caused by hardware failures)
- Design rescheduling for workloads that will be preempted (rescheduling will
  be addressed in a separate dedicated KEP)
- Change the preemption principle of avoiding preemption if a workload/pod can be scheduled without it.
  If we decide to change that it will be addressed in a dedicated KEP.
- Propose any tradeoff between preemption and cluster scale-up.
- Design workload-level preemption triggerred by external schedulers

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

The implication of the second one is that we should always try to avoid cascading preemptions.

### High-level approach

We start with a relatively simple design focusing on extensible APIs and semantics without
targeting very sophisticated algorithms. In this section we just introduce the individual
pieces of the solution and discuss them in more detail in the following sections.

1. We piggy-back on existing `PriorityClass` API to avoid reinventing the concept of priority
   from scratch.
1. We extend the `Workload` API to allow for defining the preemption unit. Here we also start
   simple and allow for preemption unit to only correspond to a `PodGroup` in a gang mode.
1. We extend the `Workload API` to allow for defining the priority of a workload. Again, we
   start simple and assume that individual `PodGroups` within a `Workload` has to share the
   same priority. We may decide to relax that assumption in the future follow-up enhancement.
1. However, we start with separating the concepts of scheduling and preemption priorities from
   the very beginning. The first one is simple generalization of pod priority concept. The
   later allow for dynamically adjust the expected cost of preemption of a given workload.
1. We start with a simple sub-optimal preemption algorithm that is based on the existing
   pod preemption algorithm used by kube-scheduler.
1. We introduce a mechanism of "delayed preemption" to postpone actuation of preemption
   decisions until we really know that these are necessary. This is to prevent the situation
   when preemption is triggerred but the pod/workload that triggerred it in the end cannot
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
// PreemptionMode describes the mode in which a PodGroup can be preempted.
// +enum
type PreemptionMode string

const (
  // PreemptionModePod means that individual pods can be preempted independently.
  PreemptionModePod = "Pod"
  // PreemptionModePodGroup means that the whole PodGroup replica needs to be
  // preempted together.
  PreemptionModePodGroup = "PodGroup"
)

type GangSchedulingPolicy struct {
    // Existing field(s).

    // PreemptionMode defines the mode in which a given PodGroup can be preempted.
    // One of Pod, PodGroup.
    // Defaults to Pod if unset.
    PreemptionMode *PreemptionMode
}
```

### Workload priorities

Prioritizing workload across each other requires answering the question: "What is workload priority?".
Up until now, only individual pods have assigned priority. But nothing prevents individual pods
forming a `Workload` or a `PodGroup` from being heterogenuous and having different priorities.

Intuitively, the priority of a `Workload` or `PodGroup` in such case would be the
"minimum of priorities of pods that belong to it" - any pod with a priority higher than that can
e.g. preempt our workload. But intuition is not enough here.

As described in user stories above, a simple static priority doesn't seem to be enough. Arguably it is
not even a single priority because a priority used for scheduling can be different than the priority
that should be used for preemption. So in the ideal world a workload owner should be able to define:

- priority used for scheduling (potentially also separately for every PodGroup)
- priority used for preemption (again potentially also for every PodGroup) and be able to arbitrarily
  mutate it during the whole lifecycle of the workload

We start simpler though and assume that all PodGroups have the same scheduling and preemption
policy by extending the `Workload` API as following:

```golang
type WorkloadSpec struct {
    // Existing field(s).

    // PriorityClassName, if specified, indicates the workload's priority that
    // should be used when scheduling this workload. "system-node-critical" and
	// "system-cluster-critical" are two special keywords which indicate the
	// highest priorities with the former being the highest priority. Any other
	// name must be defined by creating a PriorityClass object with that name.
	// If not specified, the priority will be default or zero if there is no
	// default.
    //
    // This field is immutable.
    PriorityClassName *string

    // PreemptionPriorityClassName, if specified, indicates the workload's
    // priority that should be used when attempting to preempt this workload.
    // If not specified, it will default to PriorityClassName.
    //
    // This field is mutable.
    PreemptionPriorityClassName *string
}
```

If that appears not being enough, we will similarly extend the `PodGroup` API in a follow up.
In such case, the priority of a pod would be the priority of a smallest unit in the `Workload`
object (Workload, PodGroup, PodSubGroup, ...) corresponding to this pod.


There is one direct implication of the above - the `pod.Spec.PriorityClassName` and `pod.Spec.Priority`
may no longer reflect the actual pod priority, which could be misleading to users.

```
<<[UNRESOLVED priority divergence]>>
There are several options we can approach it (from least to most invasive):
- Describe the possible divergence via documentation
- Expose the information about divergence in the API.
  This would require introducing a new `Conditions` field in `workload.Status` and introducing
  a dedicated condition like `PodsNotMatchingPriority` that will be set by either kube-scheduler
  or a new workload-controller whenever it observes pods referencing a given `Workload` object
  which priority doesn't match the priority of the workload object.
- Introducing an admission to validate that if a pod is referencing a workload object, its
  `pod.Spec.PriorityClassName` equals `workload.Spec.PriorityClassName`. However, we allow creating
  pods before the workload object, and there don't see, to be an easy way to avoid races.
- Making `pod.Spec.PriorityClassName` and `pod.Spec.Priority` mutable fields and having a new
  workload controller responsible for reconciling these. However, that could introduce another
  divergence between the priority of pods and the priority defined in the PodTemplate in true
  workload objects which would introduce a similar level of confusion to users.

If we could address the race in validations, that seems like a desired option. However,
I don't see an easy option for it.
Given that, we suggest to proceed with just exposing the information about divergence in the
Workload status (second option) and potentially improving it later.
<<[/UNRESOLVED]>>
```

It's worth mentioning here, that we want to introduce the same defaulting rules for
`workload.Spec.PriorityClassName` that we have for pods. Namely, if `PriorityClassName` is unset
and there exists PriorityClass marked as `globalDefault`, we default it to that value.
This consistency will allow us to properly handle when users are not setting neither pods
nor workload priorities.
Similarly, we will ensure that `PriorityClass.preemptionPolicy` works exactly the same way for
workloads as for pods. Such level of consistency would make adoption of Workload API much easier.

Moving to `PreemptionPriorityClassName`, the same issue of confusion holds (the actual priority
set at the pod level may not reflect priority used for preemption). We argue that its mutable
nature makes it infeasible for reconsiling this information back to pods for scalability reasons
(we can absolutely handle frequent updates to `Workload.Spec.PreemptionPriorityClassName`,
but we can't handle updating potentially hundreds or thousands of pods within that workload
that frequently). So in this case, we limit ourselves to documentation.

```
<<[UNRESOLVED preemption cycles]>>
If we would allow for arbitrary relation between scheduling priority and preemption policy,
we could hit an infinite cycle of preemption. Consider an example when:
- workload A has scheduling priority `high` and preemption policy `low`
- workload B has scheduling priority `high` and preemption policy `low`
In such case, workload A can preempt workload B (`high` > `low`), but then workload B can
also preempt workload A. This is definitely not desired.
We can avoid the infinite cycle by ensuring that `scheduling priority <= preemption priority`.

However, this also opens a question if we should allow for setting arbitrary high preemption
priority for low scheduling priority workloads. Arguably we can claim that scheduling priority
should be the ultimate truth and if there is a workload with higher priority it should be
able to preempt it.
So the alternative model that we can consider is to instead adding the concept of preemption
priority, introduce a concept of "preemption cost". In such a model, the workload with
higher priority can always preempt lower priority ones, but if we need to choose between
two workloads to preempt, such preemption cost may result in choosing the one with higher
priority amongst these two. Consider the following example:
- we want to schedule workload A with scheduling priority `high`
- it needs to preempt one of the already running workloads
- workload B has scheduling priority `med` but preemption cost `low`
- workload C has scheduling priority `low` but preemption cost `high`
In such case, the preemption cost would result in choosing workload B for preemption. But
if it gets recreated, it will preempt workload C causing unnecessary cascading preemption.
This is the reason why a cost-based model was discarded.

So for now, we suggest introducing only additional validation of scheduling priority to be
not higher then preemption policy.
<<[/UNRESOLVED]>>
```

```
<<[UNRESOLVED priority status]>>
We should introduce/describe `workload.status` to reflect:
- actual priority (a number) that is inferred from the priority class
- express the preemption priority that is currently used by scheduler to reflect that
  it aknowledged the change
<<[/UNRESOLVED]>>
```

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
moving to the level of `Workload`, but also no longer operating at the level of individual nodes.
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
      - all running workloads with (preemption) priority lower then preemptor priority; note that
        some pods from that workload may be running outside of currently considered domain D - they
        need to contribute to scoring, but they won't contribute to feasibility of domain D.
      - all individual pods with priority lower the preemptor priority

   1. If removing all potential victims would not make the preemptor schedulable, the preemptor
      is unschedulable with preemption in currently considered domain D.

   1. Sort all the potential victims to reflect their "importance" (from the most important to the
      least ones). Tentatively, the function will sort first by their priority, and within a single
      priority prioritizing workloads over individual pods.

   1. Perform best-effort reprieval of workloads and pods violating PodDisruptionBudgets. We achieve
      it but scheduling and assuming the preemptor (assuming that all potential victims are removed),
      and then iterating over potential victims that would violate PodDisruptionBudget to check if
      these can be placed in the exact same place they are running now. If they can we simply leave
      them where they are running now and remove from the potential victims list.

      ```
      <<[UNRESOLVED PodDisruptionBudget violations]>>
      The above reprieval works identically to current algorithm if the domain D is a single node.
      For larger domains, different placements of a preemptor are potentially possible and may result
      in potentially different sets of victims violating PodDisruptionBudgets to remain feasible.
      This means that the above algorithm is not optimizing for minimizing the number of victims that
      would violate their PodDisruptionBudgets.
      However, we claim that algorithm optimizing for it would be extremely expensive computationally
      and propose to stick with this simple version at least for a foreseable future.
      <<[/UNRESOLVED]
      ```

   1. For the remaining potential victims, using binary search across priorities find the minimal
      priority N for which scheduling the preemptor can be achieved without preempting any victims
      with priority higher than N. This allows to reduce the potential cascaiding preemptions later.

   1. Eliminate all victims from the potential victims list that have priority higher than N.

   1. Schedule and assume the preemptor (assuming that all remaining potential victims are removed).

   1. Iterate over the list of potential victims (in the order achieved with sorting above) checking
      if they can be placed where they are currently running. If so assume it back and remove from
      potential victims list.

   ```
   <<[UNRESOLVED minimizing preemptions]>>
   The above algorithm is definitely non optimal, but is (a) compatible with the current pod-based
   algorithm (b) computationally feasible (c) simple to reason about.
   As a result, I suggest that we proceed with it at least as a starting point.

   As a bonus we may consider few potential placements of the preemptor and choose the one that
   somehow optimizes the number of victims. However, that will appear to be more critical once we
   get to Topology-Aware-Scheduling and I would leave that improvement until then.
   <<[/UNRESOLVED]>>
   ```

1. We score scheduling decisions for each of the domains and choose the best one. The exact criteria
   for that will be figured out during the implementation phase.

It's worth noting that as structured, this algorithm addresses all four cases mentioned above that
we want to support and is compatible with the current pod-based preemption algorithm. This means
we will be able to achieve in-place replacement with relatively localized changes.

### Delayed preemption

```
<<[UNRESOLVED delayed preemption]>>
Should we leave it as part of this KEP or should this be moved to the Gang-Scheduling one?
<<[/UNRESOLVED]>>
```

As part of minimizing preemptions goal, arguably the most important thing to do is to avoid unnecessary
preemptions. However, with the current model of preemption when preemption is triggered immediately
after the victims are decided (in `PostFilter`) doesn't achieve this goal. The reason for that is
that the proposed placement (nomination) can actually appear to be invalid and not be proceeded with.
In such case we will not even proceed to binding and the preemption will be completely unnessary
disruption.
Note that this problem already exists in the current gang scheduling implementation. A given gang may
not proceed with binding if the `minCount` pods from it can't be scheduled. But the preemptions are
currently triggered immediately after choosing a place for individual pods. So similarly as above,
we may end up with completely unnecessary disruptions.

We will address it with what we call `delayed preemption` mechanism as following:

1. We will modify the `DefaultPreemption` plugin to just compute preemptions, without actuating those.
   At this point these should only be stored in kube-scheduler's memory.
   We advice maintainers of custom PostFilter implementations to do the same.

```
<<[UNRESOLVED storing victims]>>
Decide how exactly victims should be stored. Potential options:
1. New field in the pod like `pod.Spec.PreemptionTargets` if it gets introduces by
   [Kubernetes Scheduling Races Handling] (without writing it to kube-apiserver).
1. New field in the workload object (delayed preemption will not bring much value in
   case of scheduling individual pods, though there would be significant benefit from
   unification, so probably this isn't ideal option).
1. Storing it in private kube-scheduler' structures (PodInfo for individual pods and
   WorkloadInfo (new structure) for workload preemption).

The last one would probably allow for the best unification.
<<[/UNRESOLVED]>>
```

1. Introduce a new extension point to the Plugins - tentatively call it `PermitDisruption`.
   The idea behind it can be thought as an extension of scheduling vs binding cycle - we
   want to introduce a clear distinction between places where potentially disruptive decisions
   should be made from where these should be actuated.
   In particular, PostFilter should remain the point where preemption decisions should be
   made, but their actuation should be moved to the new `PermitDisruption`.

1. The framework implementation will be adjusted in a way that `PermitDisruption` plugins
   will be called whenever `schedulingCycle` fails. The input to this plugin will include
   the `nomination` for this pod.

   For individual pod (not being part of a workload) `PermitDisruption` will be implemented
   by the `DefaultPreemption` plugin and will simply actuate the previously computed
   preemption victims for a given pod.

   For pods being part of a workload, `PermitDisruptions` will be implemented by the already
   existing `GangScheduling` plugin. The implemention will be a sibling to the existing
   `Permit` extension point - the plugin will be waiting for at least `gang.minPods` to be
   successfully nominated (or assumed) and only after satisfying this condition the preemptions
   will be actuated.
   Note that as part of this change, we should treat an arbitrary preemption victim as effectively
   blocking any pod to be scheduled. While this isn't a hard requirement, it doesn't introduce
   restrictions of already assumed pods if a different placement can be found in subsequent
   scheduling cycles.

1. To reduce the number of unnessary preemptions, in case a preemption has already been triggerred
   and the already nominated placement remains valid, no new preemptions can be triggerred.
   In other words, a different placement can be chosen in a subsequent scheduling phases only if
   it doesn't require additional preemptions or the previously chosen placements is no longer
   feasible (e.g. because higher priority pods were scheduled in the meantime).

The rationale behind the above design is to maintain the current scheduling property where preemption
doesn't result in a commitment for a particular placement. If a different possible placement appears
in the meantime (e.g. due to other pods terminating or new nodes appearing), subsequent scheduling
attempts may pick it up, improving the end-to-end scheduling latency. Returning pods to scheduling
queue if these need to wait for preemption to become schedulable maintains that property.

[Kubernetes Scheduling Races Handling]: https://docs.google.com/document/d/1VdE-yCre69q1hEFt-yxL4PBKt9qOjVtasOmN-XK12XU/edit?resourcekey=0-KJc-YvU5zheMz92uUOWm4w


[API Design For Gang and Workload-Aware Scheduling]: https://tiny.cc/hvhs001

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
- More rigorous forms of testing—e.g., downgrade tests and scalability tests
- All functionality completed
- All security enforcement completed
- All monitoring requirements completed
- All testing requirements completed
- All known pre-release issues and gaps resolved

**Note:** Beta criteria must include all functional, security, monitoring, and testing requirements along with resolving all issues and gaps identified

#### GA

- N examples of real-world usage
- N installs
- Allowing time for feedback
- All issues and gaps identified as feedback during beta are resolved

**Note:** GA criteria must not include any functional, security, monitoring, or testing requirements.  Those must be beta requirements.

**Note:** Generally we also wait at least two releases between beta and
GA/stable, because there's no opportunity for user feedback, or even bug reports,
in back-to-back releases.

**For non-optional features moving to GA, the graduation criteria must include
[conformance tests].**

[conformance tests]: https://git.k8s.io/community/contributors/devel/sig-architecture/conformance-tests.md

#### Deprecation

<!--
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
