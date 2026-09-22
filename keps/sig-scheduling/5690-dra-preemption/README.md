<!--
**Note:** When your KEP is complete, all of these comment blocks should be removed.

Follow the guidelines of the [documentation style guide].
In particular, wrap lines to a reasonable length, to make it
easier for reviewers to cite specific portions, and to minimize diff churn on
updates.

[documentation style guide]: https://github.com/kubernetes/community/blob/master/contributors/guide/style-guide.md

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
# KEP-5690: DRA Preemption

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
  - [Preemption settling for DRA](#preemption-settling-for-dra)
  - [Claim nomination](#claim-nomination)
    - [Holding capacity for the preemptor](#holding-capacity-for-the-preemptor)
    - [Interaction with the preemption simulation](#interaction-with-the-preemption-simulation)
    - [Deferring further preemption](#deferring-further-preemption)
    - [Why simulated allocations and victim claims rather than devices](#why-simulated-allocations-and-victim-claims-rather-than-devices)
  - [Deferred: workload-aware preemption](#deferred-workload-aware-preemption)
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
Dynamic Resource Allocation (DRA) was promoted to stable in v1.35, but one major capability—preemption of pods
utilizing DRA resources—is currently not supported. Under the current state, lower-priority workloads that
happen to be scheduled early can occupy scarce devices indefinitely, leaving higher-priority workloads
pending. By introducing native preemption support in DRA, we equip the scheduler with the necessary
capabilities to ensure that the most critical workloads gain access to scarce hardware resources.

Simulating the removal of victims is not sufficient on its own. The devices held by preempted pods
are reclaimed asynchronously by the resourceclaim controller, so there is a window during which a
preemptor has been promised capacity that it does not yet hold. This KEP therefore also specifies
how the scheduler records and holds that capacity, so that a preemptor is not made to preempt
repeatedly and its promised resources are not taken by other pods while being reclaimed.

Most of this is contained in the `dynamicresources` plugin. The one exception is a new optional
scheduler framework extension point, `PreemptionExtensions`, which notifies plugins when a
preemption candidate is selected or cleared and lets a plugin report whether resources freed by an
earlier preemption are still being reclaimed.

## Motivation
Acquisition and allocation of specialized hardware (e.g., GPUs and TPUs) is a primary operational concern for
AI/ML users. Because accelerator resources are frequently constrained, it is essential that Kubernetes
provides robust tools to run high-priority workloads on the available hardware. Introducing preemption
support for DRA ensures that the scheduler can reclaim resources from lower-priority workloads to satisfy
scheduling requirements of higher-priority tasks.

### Goals
* Enable the scheduler to preempt pods consuming node-local DRA devices, in the pod-by-pod
  preemption path implemented by `DefaultPreemption`.
* Ensure that a preemptor is not made to preempt repeatedly because the devices it freed are
  reclaimed asynchronously, or are taken by another pod before it can be scheduled.

### Non-Goals
* Support preemption in the workload-aware preemption path. Pods belonging to a PodGroup are out of
  scope, whether their ResourceClaims are reserved for the PodGroup or for individual pods. What
  this would require is outlined in
  [Deferred: workload-aware preemption](#deferred-workload-aware-preemption).
* Support preemption of pods using multi-node or network-attached devices.
* Persist the scheduler's record of in-flight preemptions across a scheduler restart or expose it in
  the API for external components such as Cluster Autoscaler.
* Coordinate held capacity between multiple schedulers.

## Proposal
We will implement the `fwk.PreFilterExtensions` interface in the `dynamicresources` scheduler plugin, which
requires implementing the `AddPod` and `RemovePod` methods. These functions are invoked by the core
`DefaultPreemption` plugin to incrementally simulate the removal or recovery (reprieval) of candidate victim
pods during the preemption planning loop, as well as by the scheduling framework when accounting for
nominated pods on a node (`RunFilterPluginsWithNominatedPods`).
Implementing these functions requires updating the internal `stateData` structure in the `dynamicresources`
plugin to track state transitions and maintain local capacity maps transactionally throughout the preemption
simulation.
The scheduler must correctly handle several advanced DRA features during this simulation, or explicitly
bypass preemption if those features are in use.

Features that we will support but require careful implementation:
* **Partitionable Devices**: Freeing up the exact device partition requested by a higher-priority workload
  may require preempting multiple lower-priority pods that collectively occupy fractional partitions
  (represented as counters) on the same physical device, so that the raw capacity can be consolidated
  and re-partitioned.
* **Consumable Capacity**: We might need to free up just a subset of the capacity on a device to satisfy
  the request of a higher-priority pod. We need to make sure the remaining capacity on a device is correctly
  tracked during preemption simulations, and that we don't overcommit.
* **Device Binding Conditions**: A lower-priority pod may be blocked in `PreBind` waiting for device
  binding conditions to be satisfied while already holding an in-flight or allocated ResourceClaim.
  We need to make sure its reserved capacity is accounted for during preemption simulations and that
  preempting the pod immediately aborts its `PreBind` wait so the claim can be deallocated.

Features/scenarios that we will not support:
* **ResourceClaims that span multiple nodes, and network-attached devices**: Every DRA device has an
  associated node selector (`nodeName`, `nodeSelector`, or `allNodes`), either set directly on the
  device or inherited from the `ResourceSlice` in which the device is defined. The `dynamicresources`
  plugin will only simulate the release of claims whose allocated devices all use the explicit
  `nodeName` selector (`Spec.NodeName != nil`). Claims using `nodeSelector` or `allNodes` are ignored
  during preemption simulation. Separately, preemption enumerates candidate victims from the pods on
  the node being evaluated, so a device held by a pod on a different node is never offered as
  something that could be freed.
* **Pods belonging to a PodGroup**: Preemption for these goes through the workload-aware preemption
  path, which removes victims from the simulation differently. The plugin will not free devices for
  them, whether their ResourceClaims are reserved for the PodGroup or for individual pods. See
  [Deferred: workload-aware preemption](#deferred-workload-aware-preemption).

Simulating the removal of victims is not the whole problem. The devices freed by a preemption do not
become allocatable when the victim pods are deleted, but later, when the resourceclaim controller
deallocates their ResourceClaims. During that interval the scheduler can preempt further pods
unnecessarily, and the freed devices can be taken by an unrelated pod. When a preemption candidate is
selected, `dynamicresources` records in memory the simulated `AllocationResult`s computed for the preemptor
together with the victim `ResourceClaim`s being released. It uses the simulated `AllocationResult`s
in `AddPod` to hold the exact capacity the preemptor needs on the nominated node, and uses the victim
`ResourceClaim`s to defer any further preemption by the same pod until those claims have been
deallocated. This is specified in [Design Details](#design-details).

Supporting this requires a small, generic addition to the scheduling framework: an optional
`PreemptionExtensions` interface. Because `DefaultPreemption` evaluates candidate nodes concurrently
using cloned `CycleState`s, a plugin cannot tell from `RemovePod` and `Filter` alone which candidate
won. `PreemptionExtensions` notifies the plugin when a winning preemption candidate is selected
(passing the winning `CycleState` and victim pods) or when a nomination is cleared, and lets a plugin
report in `PodEligibleToPreemptOthers` whether resources freed by an earlier preemption by that pod
are still being reclaimed. Today `DefaultPreemption` answers that eligibility question with a
hard-coded heuristic: if a pod already has `nominatedNodeName` set from an earlier preemption, it
refuses to let that same pod preempt again while any pod on its nominated node is still terminating
(other incoming pods without `nominatedNodeName` set are still free to preempt on that node). That
heuristic works for resources released with the pod, but not for resources that a controller
reclaims afterwards. `PreemptionExtensions` makes both the nomination lifecycle and the settling
check pluggable. For Alpha, `PreemptionExtensions` is provisional (and may be kept internal to the
framework rather than exposed to external plugins) while SIG Scheduling evaluates whether DRA state
and nominations should move into the core scheduler framework for Beta.

### Risks and Mitigations

The risks below concern the claim nomination mechanism specified in
[Claim nomination](#claim-nomination), which holds the capacity promised to a preemptor until the
preemptor has been scheduled. The timeline referred to as t0 to t4 is defined in
[Preemption settling for DRA](#preemption-settling-for-dra).

* **Nominations are in-memory and not visible to external components or across scheduler restarts.**
  Nominations are held in memory in the `dynamicresources` plugin. A scheduler restart during the
  settling window returns the affected pods to the behavior they would have without this feature:
  the preemptor may have its freed devices taken by another pod, and may cascade once. Similarly,
  external components that run scheduling simulations—most notably Cluster Autoscaler and external
  queue controllers such as Kueue—can see `nominatedNodeName` on the preemptor pod in the API, but
  cannot see which specific DRA devices or capacities on that node have been nominated for it, and
  may therefore make conflicting scale-down or placement decisions during the settling window.
  Persisting nominations in the API server (for example in `ResourceClaim.Status`) requires an API
  change; we defer that design to Beta.
* **Claims that are never deallocated.** If the resourceclaim controller is unhealthy and fails to
  deallocate a victim claim, its capacity remains occupied in the API server (which is standard DRA
  behavior whenever a pod is deleted while the controller is down). The preemptor remains waiting
  for its nominated node—matching how `DefaultPreemption` behaves when a victim pod is stuck
  terminating—rather than timing out and evicting additional victims on other nodes while the
  controller is unhealthy. Once the controller recovers and deallocates the claim (or if the
  preemptor is deleted), the nomination resolves.
* **Multiple schedulers.** A nomination is only known to the scheduler that created it. Another
  scheduler may allocate the held capacity. This matches the existing limitation of nominated nodes.

## Design Details

### Preemption settling for DRA

When the scheduler decides to preempt, it deletes the victim pods, sets `nominatedNodeName` on
the preemptor and returns the preemptor to the scheduling queue. There is then a window during
which the preemptor has been promised resources that it does not yet hold:

| Time | Event | Actor |
|------|-------|-------|
| t0 | Victim pods are deleted and graceful termination begins | kube-scheduler |
| t1 | The victim pod objects are gone | kubelet / API server |
| t2 | `ReservedFor` is cleared on the victims' ResourceClaims | resourceclaim controller |
| t3 | `Status.Allocation` is cleared; the devices become allocatable again | resourceclaim controller |
| t4 | The preemptor is retried and scheduled | kube-scheduler |

The scheduler already handles part of this window. When a pod that already has `nominatedNodeName`
set is retried, `PodEligibleToPreemptOthers` refuses to let that pod start a second preemption while
its nominated node still holds pods that are terminating because of preemption (while still allowing
other un-nominated pods to preempt on that node). That check assumes the settling window ends when
the victim pods are gone, that is at t1.

For DRA the assumption does not hold, because devices are reclaimed asynchronously by the
resourceclaim controller and only become allocatable at t3. Two problems follow:

* **Cascading preemption.** Between t1 and t3 the terminating-pod check no longer applies, but the
  devices are still allocated. If the preemptor is retried during this interval it fails again and
  preempts a second set of victims that were never needed. In the worst case, a slow or stuck
  resourceclaim controller turns a single preemption into a series of them.
* **Device stealing.** Between t3 and t4 the devices are free, and nothing in the API records that
  they were freed on behalf of the preemptor. An unrelated pod may be allocated them first, after
  which the preemptor has to preempt all over again. Under contention for scarce accelerators this
  can repeat indefinitely, which would defeat the purpose of this KEP.

These are two views of the same gap: the scheduler has no representation of capacity that a
preemptor has earned by preempting but has not yet received. A single mechanism, described below,
addresses both.

DRA is not fundamentally special here. The same gap exists for any resource that is reclaimed by a
controller rather than by the removal of the pod itself; DRA is simply the first case where the
interval is long enough to matter in practice.

### Claim nomination

When `DefaultPreemption` selects a winning candidate for a preemptor pod, it invokes
`PreemptionExtensions.AddNominatedPod` on registered plugins. In its `AddNominatedPod`
implementation, `dynamicresources` records an internal *claim nomination* for that preemptor,
consisting of:

* the nominated node;
* the simulated `AllocationResult` for each of the preemptor's ResourceClaims on that node;
* the UIDs of the victim ResourceClaims that the simulation released in order to make the placement
  feasible.

During `Filter` in the preemption simulation, `dynamicresources` caches the computed
`AllocationResult`s per candidate node in its own `CycleState` entry. When `AddNominatedPod` is
called with the scheduling cycle's `CycleState`, the winning `nodeName`, and `victims`,
`dynamicresources` looks up the cached `AllocationResult`s for `nodeName` and derives the released
victim claims from `victims` (including only claims whose every reserving pod is in `victims`, not
shared claims still held by a non-preempted pod).

A nomination's lifetime is bound 1-to-1 to the pod's `nominatedNodeName` in the scheduler's
`PodNominator`. It is created by `PreemptionExtensions.AddNominatedPod` and discarded by
`PreemptionExtensions.RemoveNominatedPod` when:

* the preemptor is scheduled and bound;
* the preemptor is deleted;
* the preemptor's `nominatedNodeName` is cleared or replaced by a subsequent preemption.

No separate time-based expiry is needed. Once all victim claims have been deallocated at t3, if the
preemptor is retried and still cannot be scheduled on the nominated node (for example because a
higher-priority pod took the freed device, or the node became unschedulable), `PodEligibleToPreempt`
returns `true` and `DefaultPreemption` immediately re-evaluates the pod—either replacing the
nomination with a new candidate node or clearing `nominatedNodeName`, which removes the nomination.
Before t3, while the victim claims are still allocated, waiting without a timeout matches how
`DefaultPreemption` behaves when a victim pod is stuck terminating: if the resourceclaim controller
is unhealthy, expiring the nomination would only cause the preemptor to evict additional victims on
other nodes while the controller is down.

Because a nomination is held only in memory, removing it when a preemptor is deleted or its
`nominatedNodeName` is cleared or changed does not produce a `ResourceClaim` event in the API server.
Instead, `dynamicresources` registers a `QueueingHint` for nominated-pod removal/update events and
checks the nomination's victim `ResourceClaim`s in its local informer cache: if the victim claims
already have `Status.Allocation == nil` (i.e. t3 has already passed, so no future `ResourceClaim`
deallocation event will arrive), the `QueueingHint` returns `Queue` to wake unschedulable pods that
were blocked by the held capacity; if the victim claims still have `Status.Allocation != nil` (before
t3), it returns `QueueSkip` so those pods remain in the unschedulable queue until the `ResourceClaim`
deallocation event at t3 wakes them.

Nominations are held in memory in the `dynamicresources` plugin and are keyed by the preemptor's pod
UID. They are not persisted; see [Risks and Mitigations](#risks-and-mitigations).

A claim nomination is distinct from, but paired with, the pod's `nominatedNodeName`. The latter is an
API field recording which node the scheduler intends to place the pod on; the former is scheduler-local
state recording which DRA capacity on that node is being held for it and which victim claims it is
waiting on. A claim nomination never exists without a corresponding `nominatedNodeName`.

A nomination serves two purposes: it holds the promised capacity for the preemptor, and it defers
any further preemption by that pod while its victim claims are still being reclaimed. The subsections
below cover the first, then how it composes with the preemption simulation, then the second.

#### Holding capacity for the preemptor

When the scheduler evaluates a node for another pod `Q`—either during normal `Filter` or during a
preemption simulation—the framework's `RunFilterPluginsWithNominatedPods` evaluates `Filter` in up
to two passes whenever nominated pods `P` with priority >= `Q` exist on that node:

* **Pass 1 (with nominated pods):** `RunFilterPluginsWithNominatedPods` applies
  `PreFilterExtensions.AddPod` for each such nominated pod `P` and runs `Filter`.
* **Pass 2 (without nominated pods):** If Pass 1 succeeds,
  `RunFilterPluginsWithNominatedPods` runs `Filter` a second time without `AddPod(P)` applied.

When `AddPod` is called for a nominated preemptor `P` in Pass 1, `dynamicresources` consults `P`'s
live nomination and performs two updates to the allocated device state in `CycleState`:

1. **Subtract still-allocated victim claims (t0 to t3):** For each victim `ResourceClaim` UID
   on that node that still has `Status.Allocation != nil` in the informer cache (and has not already
   been removed in `CycleState`), `dynamicresources` subtracts its `Status.Allocation` from the
   allocated device state, using the same accounting helper as `RemovePod`. This prevents
   double-counting the capacity that `P` itself is taking from its victim claims between t0 and t3:
   in particular, between t1 and t3, the victim Pod object has already been deleted from the API
   server and is no longer in `NodeInfo.Pods` (so `RemovePod` is not called for it), yet its
   `ResourceClaim.Status.Allocation` remains present in the state built by `PreFilter` until t3.
2. **Add the preemptor's simulated `AllocationResult`s (t0 to t4):** `dynamicresources` adds
   `P`'s recorded `AllocationResult`s into the allocated device state, marking the exact devices,
   consumable capacity shares, or partition counters selected for `P` as in use.

When both Pass 1 and Pass 2 run, `dynamicresources.Filter` preserves the `AllocationResult` computed
for `Q` in Pass 1 (which respected `AddPod(P)` and avoided `P`'s nominated devices) rather than
letting Pass 2 overwrite `nodeAllocations[nodeName]`.

The combination of Pass 1 (with nominated pods) and Pass 2 (without nominated pods) ensures accurate
capacity accounting across all phases of the settling window:

* **Surplus victim capacity is never treated as free unless `RemovePod` removed the victim pod or
  t3 has passed:** If a victim claim `Claim-V` holds more capacity than `P` consumes (for example,
  two devices `GPU-0` and `GPU-1` when `P` only needs `GPU-0`), subtracting `Claim-V` and adding
  `P`'s `AllocationResult` in Pass 1 leaves `GPU-1` free in Pass 1, but `Claim-V` (`GPU-0` and
  `GPU-1`) remains allocated in Pass 2 unless `RemovePod(V)` explicitly removed `V`. Thus, during
  normal `Filter` (t0 to t3) or during a preemption simulation between t1 and t3 (when `V` is
  already gone from `NodeInfo.Pods`), Pass 2 rejects any attempt to allocate `GPU-1` while `Claim-V`
  is still allocated in the API server.
* **Once `Claim-V` is deallocated at t3:** `Claim-V` drops out of the state built by `PreFilter`,
  step 1 in `AddPod(P)` becomes a no-op, and `P` continues to hold only `GPU-0` via step 2 until t4,
  while `GPU-1` becomes immediately allocatable in normal `Filter` (passing both Pass 1 and Pass 2).

The capacity is held only against pods whose priority is not higher than the preemptor's. When a pod
of higher priority evaluates the node, `RunFilterPluginsWithNominatedPods` does not call `AddPod`
for the lower-priority preemptor, so the higher-priority pod sees the capacity as free once t3
passes and may be allocated it, after which the preemptor must preempt again.

#### Interaction with the preemption simulation

Two cases illustrate how `AddPod` composes with subsequent preemption simulations on the same node
while an earlier preemption on behalf of `P1` is settling:

* **Sharing a terminating victim (t0 to t1):** Suppose victim `V` on Node `N` holds a claim
  `Claim-V` with two devices (`GPU-0` and `GPU-1`), and `P1` (needing one device) preempts `V` and
  records a nomination for `GPU-0` and victim claim `Claim-V`. While `V` is still terminating in
  `NodeInfo.Pods` (t0 to t1), an equal-priority pod `P2` (also needing one device) runs a preemption
  simulation on Node `N`:
  1. `DefaultPreemption` calls `RemovePod(V)`, removing `Claim-V` (`GPU-0` and `GPU-1`) from the
     simulated state.
  2. In Pass 1 of `RunFilterPluginsWithNominatedPods`, `AddPod(P1)` adds `P1`'s nominated allocation
     (`GPU-0`). `Filter(P2)` sees `GPU-0` occupied by `P1` and `GPU-1` free, and computes a
     simulated allocation of `GPU-1` for `P2`.
  3. Pass 2 (without nominated pods) also succeeds because `RemovePod(V)` removed `Claim-V` from the
     simulated state.
  Both `P1` (`GPU-0`) and `P2` (`GPU-1`) therefore select `V` as a victim, record `Claim-V` in their
  nominations, and schedule once `Claim-V` is deallocated at t3. (If `P2` instead arrives between t1
  and t3 after `V` has already been deleted from `NodeInfo.Pods`, `DefaultPreemption` sees no victim
  pod `V` on Node `N` to evict; `P2` simply waits until `Claim-V` is deallocated at t3, when the
  `ResourceClaim` informer event wakes `P2` and schedules it onto `GPU-1` in normal `Filter`.)

* **Preempting a second victim on the same node after the first victim is deleted (t1 to t3):**
  Suppose an 80Gi device on Node `N` is split between `V1` (`Claim-V1`, 40Gi) and `V2` (`Claim-V2`,
  40Gi). `P1` (needing 40Gi) preempts `V1`, and by t1 `V1` has terminated and been deleted from
  `NodeInfo.Pods` while `Claim-V1` (40Gi) is still allocated in the API server. When an
  equal-priority pod `P2` (needing 40Gi) runs a preemption simulation on Node `N` between t1 and t3,
  `DefaultPreemption` calls `RemovePod(V2)` (freeing `V2`'s 40Gi), but does not call `RemovePod(V1)`
  because `V1` is no longer in `NodeInfo.Pods`. In Pass 1 of `RunFilterPluginsWithNominatedPods`,
  `AddPod(P1)` subtracts `Claim-V1` (40Gi) before adding `P1`'s nominated 40Gi, preventing
  `Claim-V1` and `P1` from being double-counted to 80Gi. Both Pass 1 and Pass 2 therefore see 40Gi
  in use and 40Gi freed by `RemovePod(V2)`, allowing `P2` to preempt `V2`.

#### Deferring further preemption

While a nomination is live and any of its victim ResourceClaims still has `Status.Allocation != nil`
in the informer cache, the preemptor must not start a new preemption. Together with the existing
terminating-pod check, which covers t0 to t1, this covers the settling window up to t3. Once all
victim claims in the nomination have been deallocated (`Status.Allocation == nil`), the preemptor
becomes eligible to preempt again; if its placement no longer works at that point (for example
because a higher-priority pod took the freed device), preempting again is the correct behavior.

`PodEligibleToPreemptOthers` and `prepareCandidate` belong to the `DefaultPreemption` plugin, and we
do not want to make that plugin aware of DRA. We therefore propose a small and generic addition to
the scheduling framework: an optional `PreemptionExtensions` interface that manages the nomination
lifecycle and preemption eligibility check (provisional for Alpha while we evaluate moving DRA state
and nominations into the core framework for Beta):

```go
// PreemptionExtensions is an optional interface for plugins that manage
// resources requiring explicit reservation for nominated pods and/or
// asynchronous reclamation when victim pods are preempted.
type PreemptionExtensions interface {
    Plugin
    // AddNominatedPod is called when preemption selects a winning candidate for a pod.
    // state is the scheduling cycle's CycleState.
    AddNominatedPod(ctx context.Context, state CycleState, pod *v1.Pod, nodeName string, victims []*v1.Pod)
    // RemoveNominatedPod is called when a pod's nomination is cleared or superseded.
    RemoveNominatedPod(ctx context.Context, pod *v1.Pod)
    // PodEligibleToPreempt reports whether a pod with an existing nominatedNodeName
    // is eligible to start a new preemption, or whether resources freed by an earlier
    // preemption on nodeName are still being reclaimed.
    PodEligibleToPreempt(ctx context.Context, pod *v1.Pod, nodeName string) (bool, string)
}
```

The framework invokes registered plugins implementing `PreemptionExtensions`:

* `AddNominatedPod` is called by `DefaultPreemption` when recording the winning preemption
  candidate for a pod.
* `RemoveNominatedPod` is called by the framework's `PodNominator` whenever a pod's nomination is
  cleared or replaced.
* `PodEligibleToPreempt` is called by `DefaultPreemption.PodEligibleToPreemptOthers` alongside its
  existing terminating-pod check, stopping at the first plugin that reports `false` and surfacing
  the returned reason in the pod's scheduling condition.

The `dynamicresources` plugin implements `PodEligibleToPreempt` by checking whether the pod has a
live nomination on `nodeName` with any victim `ResourceClaim` that still has
`Status.Allocation != nil`.

When the `resourceclaim` controller clears `Status.Allocation` on a victim claim at t3, the
scheduler's `ResourceClaim` informer event handler invokes `SchedulingQueue.MoveAllToActiveOrBackoffQueue`,
which calls `dynamicresources`'s `QueueingHint` (`isSchedulableAfterClaimChange`). As a secondary
optimization, the `QueueingHint` withholds a wake-up (`QueueSkip`) for the preemptor while any of
its other nominated victim claims still has `Status.Allocation != nil`, and returns `Queue` as soon
as the last victim claim is deallocated. This avoids pointless scheduling attempts while some victim
claims are still settling.

#### Why simulated allocations and victim claims rather than devices

A claim nomination records both the **preemptor's simulated `AllocationResult`s** (to hold capacity
via `AddPod`) and the **victim `ResourceClaim` UIDs** (to track ongoing deallocation between t0
and t3), rather than nominating raw device names:

* **Why simulated `AllocationResult`s rather than device names:** With **consumable capacity** and
  **partitionable devices**, a bare device name does not express a capacity share or the counter
  draw of a partition. An `AllocationResult` (`*resourceapi.AllocationResult`) is the exact data
  structure produced by the DRA allocator during `Filter` and consumed by `dynamicresources` when
  building its allocated state. Recording the preemptor's simulated `AllocationResult` holds the
  exact share or partition counters the preemptor needs, without blocking the rest of the device.
* **Why victim `ResourceClaim` UIDs are also recorded:** A simulated `AllocationResult` has no
  completion condition of its own in the API before t4, and checking whether a simulated
  `AllocationResult` conflicts with arbitrary deallocating claims in the informer cache would
  require complex counter and partition-overlap math across `ResourceSlice`s (since a preemptor's
  partition name may differ from the partitions held by the victims). Recording the UIDs of the
  victim claims that `RemovePod` released during the winning simulation gives an O(1), directly
  observable completion condition (`claim.Status.Allocation == nil` at t3) and lets `AddPod`
  subtract those exact victim allocations while `Status.Allocation != nil` with zero conflict math.

### Deferred: workload-aware preemption

Workload-aware preemption ([KEP-5710]) removes candidate victims by mutating the scheduler snapshot
and then reruns the pod group scheduling algorithm with fresh CycleStates, instead of calling
`PreFilterExtensions.RemovePod` for each victim. A plugin that learns about removals only through
`RemovePod`, as specified here, therefore goes on counting the victims' devices as allocated, and
the preemption frees nothing. The pod group stays pending, which is the behavior without this
feature. No state is corrupted and no device is allocated twice.

Supporting that path needs three additions that we prefer to design separately:

* The plugin has to rebuild its allocated state by reconciling the ResourceClaims it reads against
  the pods present in the `NodeInfos` that `PreFilter` receives. Doing so safely requires
  distinguishing a consumer that has been removed from the snapshot from one that the plugin has
  merely not observed yet, such as a pod that another scheduler has reserved a claim for but not
  yet bound.
* A nomination has to be owned by the preempting pod group rather than by a single pod. One
  workload-aware preemption produces a nominated placement for every member of the group, all at
  the same priority, so per-pod nominations would need to be coordinated across the group.
* Workload-aware preemption (as well as multi-node DRA claims) changes the scope of preemption and
  nomination from node-local to cluster-global. Currently, the scheduler applies nominations
  per-node during `Filter` (`RunFilterPluginsWithNominatedPods` only invokes `AddPod` for pods whose
  `nominatedNodeName` matches the node being evaluated). Supporting multi-node DRA claims and global
  workload-aware preemption requires applying nominations globally before filtering starts (for
  example during `PreFilter`).

[KEP-5710]: https://github.com/kubernetes/enhancements/issues/5710

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

<!--
Generated with:
go test -cover ./pkg/scheduler/framework/plugins/dynamicresources/... | sed -e 's/.*\(k8s.io[a-z/-]*\).*coverage: \(.*\) of statements/- `\1`: \2/' | sort
-->

- `k8s.io/kubernetes/pkg/scheduler/framework/plugins/dynamicresources`: 82.4%


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

Integration tests will be added that covers the normal functionality of preemption with DRA, with
specific tests covering all the special scenarios called out above. For the settling window in
particular:

* a preemptor does not preempt a second set of victims while the claims freed by its earlier
  preemption are still allocated;
* devices freed by a preemption are not allocated to another pod of equal or lower priority before
  the preemptor has been scheduled;
* a pod of higher priority can be allocated those devices, and the preemptor then preempts again;
* a nomination is removed and held capacity is released when the preemptor is deleted or its
  `nominatedNodeName` is cleared.

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

E2e tests will be added to cover the basic scenarios for preemption with DRA. The more complicated
scenarios will be handled by integration tests.

### Graduation Criteria

#### Alpha

- Feature implemented behind a feature flag
- Unit, integration and e2e tests completed and enabled
- The cost of the preemption simulation measured with scheduler_perf

#### Beta

- Tests are in Testgrid and linked in the KEP
- Alignment on whether DRA state and claim nominations should move from `dynamicresources` into the
  core scheduler framework (replacing `PreemptionExtensions`)
- An agreed-upon design for how claim nominations can be persisted in the API to survive a scheduler
  restart and be visible to external components such as Cluster Autoscaler.
- Metrics for claim nominations exposed and documented


#### GA

- 2 examples of real-world usage
- Allowing time for feedback


### Upgrade / Downgrade Strategy

Standard upgrade/downgrade strategies may be used, no special configuration
changes are needed. The changes are local to the scheduler.

Claim nominations are held only in the scheduler's memory, so a downgrade discards them without
leaving anything behind. Preemptors that were waiting revert to the behavior they would have with
the feature disabled.

### Version Skew Strategy

This feature is implemented entirely in the scheduler and requires no new behavior from any other
component. It does rely on the resourceclaim controller deallocating the ResourceClaims of deleted
pods, but that behavior predates this KEP, so there is no version skew concern between the
scheduler and kube-controller-manager.

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
  - Feature gate name: DRAPreemption
  - Components depending on the feature gate:
    - kube-scheduler

The gate is disabled by default.


###### Does enabling the feature change any default behavior?

<!--
Any change of default behavior may be surprising to users or break existing
automations, so be extremely careful here.
-->

Yes, it will cause the scheduler to preempt pods referencing ResourceClaims to
make room for higher priority pods. There is no API change for this feature, so it
will potentially impact any Pod in the cluster when the feature is enabled.

Because the gate is disabled by default, upgrading to a release that contains this feature does not
by itself change any behavior. An operator has to enable the gate.

###### Can the feature be disabled once it has been enabled (i.e. can we roll back the enablement)?

Yes, disabling the feature will prevent the scheduler from preempting pods that
reference ResourceClaims.

Disabling it also discards any claim nominations that are currently held by the scheduler in its memory, which releases the
capacity being held for preemptors that have not yet been scheduled. Nothing is persisted, so
there is no state to clean up and no reconciliation is required.

###### What happens if we reenable the feature if it was previously rolled back?

That it was previously rolled back have no impact. Reenabling it just means that
preemption of pods referencing ResourceClaims will again be considered.

###### Are there any tests for feature enablement/disablement?

Since this is a purely in-memory feature controlled by a feature gate (which requires a scheduler
restart to change), testing with the feature gate enabled and disabled in unit and integration tests
is sufficient; no separate enablement/disablement transition tests are needed.

### Rollout, Upgrade and Rollback Planning


###### How can a rollout or rollback fail? Can it impact already running workloads?

A rollout will immediately make pods referencing ResourceClaims eligible for
preemption. So if there are higher priority pods that can't be scheduled due
to pending ResourceClaims, the result will be that lower priority pods will
get preempted.

Similarly, a rollback means that pods referencing ResourceClaims will no longer
be eligible for preemption. But already preempted workload will remain in the
pending state until sufficient resources are freed up.

###### What specific metrics should inform a rollback?

The specific signal that would suggest this feature should be rolled back, would
be if pods are being preempted when they shouldn't be. This means there is a bug
somewhere in the implementation.

If `scheduler_preemption_victims` or `scheduler_preemption_attempts_total` increases significantly when the feature is enabled, but we don't
see a corresponding increase in pods being scheduled, we should investigate. It would suggest
that pods are being preempted incorrectly and the higher-priority pods are not actually
being scheduled. A significant increase in `scheduler_plugin_execution_duration_seconds` for
`DefaultPreemption` (`PostFilter`) or `DynamicResources` (`Filter`) would also indicate that DRA
preemption simulations are causing a scheduling latency regression.

###### Were upgrade and rollback tested? Was the upgrade->downgrade->upgrade path tested?

It will be tested by bringing up a KinD cluster and enabling the feature gate.
We will then run a simple workload with higher priority pods and then enable
the feature gate to verify that the lower priority pods are preempted. We will then
disable the feature gate and verify that another pod with an even higher priority
does not preempt the existing pod.

###### Is the rollout accompanied by any deprecations and/or removals of features, APIs, fields of API types, flags, etc.?

No

### Monitoring Requirements

###### How can an operator determine if the feature is in use by workloads?

It is enabled on a cluster-level, so if it is enabled, it is in use. It will only impact
pods that are referencing ResourceClaims.

###### How can someone using this feature know that it is working for their instance?

When a Pod is preempted, it can be observed in the following ways:

- [x] Events
  - Event Reason: Preempted
- [x] API .status
  - Condition name: DisruptionTarget

Seeing this on a Pod that references ResourceClaims shows that the feature is working.

###### What are the reasonable SLOs (Service Level Objectives) for the enhancement?

Kubernetes does not have SLOs for preemption, so neither will this enhancement.

###### What are the SLIs (Service Level Indicators) an operator can use to determine the health of the service?

<!--
Pick one more of these and delete the rest.
-->

- [x] Metrics
  - Metric name: `scheduler_preemption_victims` and `scheduler_preemption_attempts_total`. These are
    not specific to DRA and carry no DRA-specific labels, so they can be compared across the cluster
    before and after the gate is enabled.
  - Metric name: `scheduler_plugin_execution_duration_seconds` (with `plugin="DefaultPreemption"`,
    `extension_point="PostFilter"` and `plugin="DynamicResources"`, `extension_point="Filter"`) to
    monitor the latency cost of DRA preemption simulations.
  - Components exposing the metric: kube-scheduler

###### Are there any missing metrics that would be useful to have to improve observability of this feature?

Dedicated metrics for tracking active DRA claim nominations and how they resolve (e.g. scheduled vs.
cleared) will be added for Beta once we have finalized whether claim nominations remain in
`dynamicresources` or move into the core framework.

### Dependencies

<!--
This section must be completed when targeting beta to a release.
-->

###### Does this feature depend on any specific services running in the cluster?

Yes. The resourceclaim controller in kube-controller-manager deallocates the ResourceClaims of
preempted pods, and preemption for DRA cannot complete until it has done so. If the controller is
unavailable or lagging, preemptors remain unschedulable and wait for their nominated claims to be
deallocated; see [What are other known failure modes?](#what-are-other-known-failure-modes).

### Scalability

<!--
For alpha, this section is encouraged: reviewers should consider these questions
and attempt to answer them.

For beta, this section is required: reviewers must answer these questions.

For GA, this section is required: approvers should be able to confirm the
previous answers based on experience in the field.
-->

###### Will enabling / using this feature result in any new API calls?

The `dynamicresources` plugin itself makes no new API calls. The preemption simulation runs against
state the plugin already maintains.

Enabling the feature does however make pods referencing ResourceClaims eligible for preemption, so
the scheduler will issue pod deletions and `DisruptionTarget` condition updates for those pods
where previously it issued none. The rate is proportional to the rate of preemption, and the cost
per preemption is the same as for pods that do not use DRA.

###### Will enabling / using this feature result in introducing new API types?

No

###### Will enabling / using this feature result in any new calls to the cloud provider?

No

###### Will enabling / using this feature result in increasing size or count of the existing API objects?

No

###### Will enabling / using this feature result in increasing time taken by any operations covered by existing SLIs/SLOs?

No. It might lead to increase time taken to check if lower priority pods can
be preempted to make room for a higher priority pod, since we don't do that
today.

###### Will enabling / using this feature result in non-negligible increase of resource usage (CPU, RAM, disk, IO, ...) in any components?

It can lead to some additional work in the scheduler, since we are enabling
preemption simulation for a new scheduler plugin.

The scheduler additionally holds one claim nomination per preemptor pod that is waiting for an
earlier preemption to settle, containing the simulated allocation results for the preemptor and the
UIDs of the victim claims being released. The number of such nominations is bounded by the number of
in-flight preemptions, and each is discarded when the preemptor is scheduled, deleted, or has its
`nominatedNodeName` cleared.

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

The `dynamicresources` plugin does not add any calls to the API server compared to what it already
does, so the plugin behaves as it would without this feature enabled.

Preemption as a whole cannot make progress while the API server is unavailable: victim pods cannot
be deleted and their ResourceClaims cannot be deallocated, so claim nominations remain active until
the API server recovers and the victim claims are deallocated.

###### What are other known failure modes?

- The resourceclaim controller does not deallocate a nominated claim, for example because it is
  unhealthy. The preemptor remains unschedulable and its nomination continues to hold the simulated
  capacity until the controller recovers and deallocates the claim (or the preemptor is deleted).
  This is visible in the API as the preemptor pod remaining Pending with `.status.nominatedNodeName`
  set while the victim ResourceClaims retain `status.allocation` after their pods have been deleted;
  operator attention is required to restore the controller.
- The scheduler restarts while preemptions are settling. Nominations are in memory and are lost, so
  the affected preemptors may have their devices taken by another pod or may preempt again. The
  effect is limited to preemptions that were in flight at the time of the restart.

###### What steps should be taken if SLOs are not being met to determine the problem?

There are no SLOs for this feature.

## Implementation History

* 1.37: first revision of the KEP and an initial implementation. Deferred to 1.38 over the handling
  of asynchronous device reclamation and the interaction with workload-aware preemption.
* 1.38: revised with the claim nomination and preemption settling designs, which address the
  asynchronous reclamation problem, and scoped to the pod-by-pod preemption path. Support for
  workload-aware preemption is deferred to a later revision. Targeted at Alpha.

## Drawbacks

It complicates the logic in the dynamicresources plugin and can lead
to slower preemption when pods are using DRA.

Holding capacity for a nominated preemptor can also leave devices idle for the duration of the
settling window if the preemptor is deleted or fails before being scheduled. See
[Risks and Mitigations](#risks-and-mitigations).

## Alternatives

Kubernetes has a well-established framework for preemption, and this KEP uses it rather than adding
a separate mechanism for DRA. The alternatives below all concern the settling window described in
[Preemption settling for DRA](#preemption-settling-for-dra).

**Nominating devices rather than simulated allocations and victim claims.** Discussed in
[Why simulated allocations and victim claims rather than devices](#why-simulated-allocations-and-victim-claims-rather-than-devices).

**Holding the victims' claims rather than the preemptor's simulated allocation.** Rather than
recording the simulated `AllocationResult`s computed for the preemptor and adding them via `AddPod`,
the plugin could hold the `AllocationResult`s of the victim claims that were released. We rejected
this because a victim claim may hold more capacity than the preemptor needs—such as multiple devices
when the preemptor needs only one, or a larger share of consumable capacity—which would unnecessarily
block other pods from claiming the surplus capacity during the settling window.

**Allocating the preemptor's claims eagerly.** Rather than recording an in-memory nomination, the
scheduler could write the allocation computed during preemption to the preemptor's ResourceClaims
immediately and mark it as not yet usable. We rejected this because the victims' claims still hold
the same devices until t3, so the two allocations would conflict; because the allocation was
computed against a simulated state and may no longer be valid once the cluster state converges; and
because it would need an unwind path for the case where the preemptor ultimately cannot be
scheduled.

**Deallocating the victims' claims from the scheduler.** The scheduler could clear
`Status.Allocation` itself once the victim pods are gone, instead of waiting for the resourceclaim
controller. This shortens the interval from t1 to t3 but does not remove it, does nothing for the
grace period from t0 to t1, and does not prevent device stealing between t3 and t4. It would also
have to cope with the scheduler's informer cache not yet reflecting its own update. It is a
possible later optimization rather than a substitute.

**Waiting for the victims to terminate within the scheduling cycle.** This removes the settling
window altogether, but blocks the scheduler for the length of the victims' grace periods and is
therefore not acceptable.

**Deferring preemption from the plugin's PostFilter rather than through `PreemptionExtensions`.**
The `dynamicresources` plugin is registered immediately before `DefaultPreemption` at the PostFilter
extension point, and `RunPostFilterPlugins` returns as soon as a plugin reports
`UnschedulableAndUnresolvable`. The plugin could therefore return that status while the pod has an
unsettled nomination, and `DefaultPreemption` would not run for that scheduling attempt. We did not
choose it because it still would not notify the plugin which candidate won `SelectCandidate`, and
because it depends on the relative ordering of the two plugins at `PostFilter`.

**Accepting the behavior and relying on priority.** Preemption is best-effort, so the scheduler
could simply let the preemptor retry. We rejected this because the situation that motivates this
KEP, many pods contending for scarce accelerators, is precisely the one in which a preemptor can be
starved indefinitely while each of its attempts evicts a further set of victims.

## Infrastructure Needed (Optional)

No
