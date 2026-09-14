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
    - [Why claims rather than devices](#why-claims-rather-than-devices)
  - [Deferred: workload-aware preemption](#deferred-workload-aware-preemption)
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
repeatedly.

Most of this is contained in the `dynamicresources` plugin. The one exception is a new optional
scheduler framework extension point, `PreemptionSettlingPlugin`, which lets a plugin report that a
pod is already waiting for capacity that an earlier preemption freed, so that the preemption logic
does not select further victims for it.

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
* Support preemption of workloads using multi-node or network-attached devices.
* Persist the scheduler's record of in-flight preemptions across a scheduler restart.
* Coordinate held capacity between multiple schedulers.
* Hold only the exact capacity a preemptor needs, rather than the whole of each released claim.

## Proposal
We will implement the `fwk.PreFilterExtensions` interface in the `dynamicresources` scheduler plugin, which
requires implementing the `AddPod` and `RemovePod` methods. These functions are invoked by the core
`DefaultPreemption` plugin to incrementally simulate the removal or recovery (reprieval) of candidate victim
pods during the preemption planning loop.
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
* **Device Binding Conditions**: Pods might be waiting for a resource binding condition, so we want to make
  sure this is handled correctly during preemption simulations.

Features/scenarios that we will not support:
* **ResourceClaims that span multiple nodes, and network-attached devices**: The `dynamicresources`
  plugin will not simulate the release of claims whose allocated devices are on more than one node.
  Separately, preemption enumerates candidate victims from the pods on the node being evaluated, so a
  device held by a pod on a different node is never offered as something that could be freed. Where a
  device's location is decoupled from the pod using it, that is a property of the preemption algorithm
  rather than a restriction this plugin can lift.
* **Pods belonging to a PodGroup**: Preemption for these goes through the workload-aware preemption
  path, which removes victims from the simulation differently. The plugin will not free devices for
  them, whether their ResourceClaims are reserved for the PodGroup or for individual pods. See
  [Deferred: workload-aware preemption](#deferred-workload-aware-preemption).

Simulating the removal of victims is not the whole problem. The devices freed by a preemption do not
become allocatable when the victim pods are deleted, but later, when the resourceclaim controller
deallocates their ResourceClaims. During that interval the scheduler can preempt further pods
unnecessarily, and the freed devices can be taken by an unrelated pod. We introduce a record of the
claims a preemptor is waiting for, and use it both to hold that capacity and to defer any further
preemption by the same pod. This is specified in [Design Details](#design-details).

It requires a small addition to the scheduling framework: an optional extension point through which
a plugin can report that an earlier preemption by a pod has not finished settling. It is not
specific to DRA. Today `DefaultPreemption` answers that question with a hard-coded heuristic: in
`PodEligibleToPreemptOthers` it looks up the pod's nominated node and refuses to start another
preemption while any pod on that node is still terminating because of preemption. That is a proxy
for "the resources I freed are not available yet", and it stops holding as soon as the victims are
gone. That works for resources that are released with the pod, but not for resources that a
controller reclaims afterwards. The extension point lets a plugin answer the question directly.

### Risks and Mitigations

The risks below concern the claim nomination mechanism specified in
[Claim nomination](#claim-nomination), which holds the capacity freed by a preemption until the
preemptor has been scheduled. The timeline referred to as t0 to t4 is defined in
[Preemption settling for DRA](#preemption-settling-for-dra).

* **Capacity is held more coarsely than necessary.** The hold covers everything the victims' claims
  held, which can be more than the preemptor needs: a whole device when the preemptor needs only a
  share of it, or several devices when a claim held more than one. It never covers capacity the
  victims did not hold. Holding only the share that the preemptor's simulated allocation consumed
  would be tighter, but is not generally expressible; see
  [Why claims rather than devices](#why-claims-rather-than-devices). The cost of the coarser
  behavior is bounded: it is never more restrictive than the cluster state just before t3, it lasts
  only until the preemptor is scheduled or the nomination expires, and it does not apply to pods of
  higher priority.
* **Nominations are lost when the scheduler restarts.** Nominations are in-memory, so a restart
  during the settling window returns the affected pods to the behavior they would have without this
  feature: the preemptor may be stolen from, and may cascade once. Persisting a nomination would
  require storing the recorded allocation results, since they no longer exist in the API after t3,
  and therefore an API change. We do not propose one for the initial implementation.
* **Claims that are never deallocated.** If the resourceclaim controller does not deallocate a
  nominated claim, the nomination expires and the preemptor preempts again. This is bounded by the
  expiry and is reported through the metrics below.
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

The scheduler already handles part of this window. `PodEligibleToPreemptOthers` refuses to start
a new preemption for a pod whose nominated node still holds pods that are terminating because of
an earlier preemption. That check assumes the settling window ends when the victim pods are gone,
that is at t1.

For DRA the assumption does not hold, because devices are reclaimed asynchronously by the
resourceclaim controller and only become allocatable at t3. Two problems follow:

* **Cascading preemption.** Between t1 and t3 the terminating-pod check no longer applies, but the
  devices are still allocated. If the preemptor is retried during this interval it fails again and
  preempts a second set of victims that were never needed. In the worst case, a slow or stuck
  resourceclaim controller turns a single preemption into a series of them.
* **Device stealing.** Between t3 and t4 the devices are free, and nothing records that they were
  freed on behalf of the preemptor. An unrelated pod may be allocated them first, after which the
  preemptor has to preempt all over again. Under contention for scarce accelerators this can repeat
  indefinitely, which would defeat the purpose of this KEP.

These are two views of the same gap: the scheduler has no representation of capacity that a
preemptor has earned by preempting but has not yet received. A single mechanism, described below,
addresses both.

DRA is not fundamentally special here. The same gap exists for any resource that is reclaimed by a
controller rather than by the removal of the pod itself; DRA is simply the first case where the
interval is long enough to matter in practice.

### Claim nomination

When the preemption logic selects a winning candidate for a preemptor pod, the `dynamicresources`
plugin records a *claim nomination* for that preemptor, consisting of:

* the nominated node;
* the set of ResourceClaims that the preemption simulation released in order to make the placement
  feasible;
* the allocation result of each of those claims;
* an expiry time.

The set of claims is taken directly from the simulation, which already computes it in order to
evaluate the candidate. Only claims whose every reserving pod was a victim are included. A shared
claim that stays allocated because a pod that is not being preempted still holds it is not part of
the nomination, which is the correct outcome: the preemptor's placement did not depend on it being
released.

The allocation results are recorded because they no longer exist in the API after t3, and the
scheduler needs to know which capacity to hold.

A nomination is discarded when the preemptor is scheduled, when it is deleted, when its
`nominatedNodeName` is cleared, when a subsequent preemption by the same pod replaces it, and when
the capacity it holds is allocated to a pod of higher priority, which invalidates the preemptor's
planned placement. These conditions cover the normal path.

The expiry is a backstop rather than a model of how long settling takes. It exists so that a
nomination cannot hold capacity indefinitely when none of the conditions above is ever observed, for
example because the resourceclaim controller never deallocates one of the claims. We propose a fixed
value of five minutes as a starting point: long enough that it does not fire during ordinary
operation, short enough to bound the cost of a leaked nomination. It is deliberately not derived
from the victims' `terminationGracePeriodSeconds`, which bounds only how long the victim pods take
to terminate. A nomination also has to survive the deallocation of their claims by the resourceclaim
controller and the preemptor's next scheduling attempt, and neither of those is a function of the
grace period.

Erring on the long side is intentional. A nomination that outlives its usefulness holds capacity for
a pod that is still pending and still wants it, and that capacity remains available to pods of
higher priority. A nomination that expires while the preemptor is still waiting removes both the
capacity hold and the deferral at once, so the preemptor preempts a second set of victims and
creates more work for the controller whose slowness caused the expiry. The `expired` outcome on
`scheduler_dra_claim_nominations_total` will show whether five minutes is the right value.

When a nomination expires the plugin requeues the preemptor, so that a pod whose wake-ups were
withheld while the nomination was unsettled does not then wait for the periodic flush of the
unschedulable queue.

Nominations are held in memory in the `dynamicresources` plugin and are keyed by the preemptor's pod
UID. They are not persisted; see [Risks and Mitigations](#risks-and-mitigations).

A claim nomination is distinct from, but paired with, the pod's `nominatedNodeName`. The latter is an
API field recording which node the scheduler intends to place the pod on; the former is scheduler-local
state recording which DRA capacity on that node is being held for it. A claim nomination never exists
without a corresponding `nominatedNodeName`.

A nomination serves two purposes: it holds the freed capacity for the preemptor, and it defers any
further preemption by that pod. The subsections below cover the first, then how it composes with the
preemption simulation, then the second.

#### Holding capacity for the preemptor

While a nomination is live, the `dynamicresources` plugin adds the recorded allocation results back
into the allocated device state that it builds for *other* pods, as though the nominated claims were
still allocated. The freed capacity is therefore not visible to those pods and cannot be allocated
to them.

This is the exact inverse of the operation that the preemption simulation already performs: the
simulation removes a claim's allocation from the allocated state, and a nomination adds it back.
Both use the same accounting, so consumable capacity and partitionable devices are handled exactly
as they were before the claims were deallocated, and no additional logic is needed for either.

Holding this capacity does not make anything unavailable that was previously available. From t0
until t3 the same capacity is already unusable by every pod, because the claims are still allocated.
A nomination extends that existing condition until t4 rather than introducing a new restriction.

The capacity is held only against pods whose priority is not higher than the preemptor's. A pod of
higher priority sees the capacity as free and may be allocated it, after which the preemptor must
preempt again. This keeps the mechanism consistent with pod priority and avoids a lower-priority
preemptor blocking a higher-priority pod.

#### Interaction with the preemption simulation

A nomination holds capacity on behalf of one pod, while the preemption simulation run for a
different pod releases claims by calling `RemovePod` for each of its candidate victims. Both act on
the allocated device state, and they can act on the same claim.

The nomination has to survive the release. `RemovePod` removes a victim's hold on a claim; it does
not remove another pod's nomination of it. Nominated capacity is therefore kept separate from the
state that the simulation mutates, and both are consulted when deciding what is allocatable.

Without this, a nomination could be cancelled by the very pod it is meant to hold capacity against.
Suppose a victim `V` holds a claim with two devices, and a preemption on behalf of `P1`, which needs
one of them, frees it. While the claim is still allocated, `P2` of equal priority fails to schedule
and simulates removing `V`. If that release also dropped `P1`'s nomination, `P2` would see both
devices as free, select `V` as a victim a second time even though it is already terminating, and
record a nomination of its own on the same claim. Both pods would then hold the same capacity and
mask it from each other, and neither would be able to schedule.

With the nomination preserved, removing `V` gains `P2` nothing, so no candidate is selected and `P2`
simply waits. Once `P1` has been scheduled its nomination is discarded, and the remaining device
becomes available to `P2`.

#### Deferring further preemption

While a nomination is live and any of its claims is still allocated, the preemptor must not start a
new preemption. Together with the existing terminating-pod check, which covers t0 to t1, this covers
the settling window up to t3. Once all nominated claims have been deallocated, or once the
nomination expires, the preemptor becomes eligible to preempt again; if its placement no longer
works at that point, preempting again is the correct behavior.

`PodEligibleToPreemptOthers` belongs to the `DefaultPreemption` plugin, and we do not want to make
that plugin aware of DRA. We therefore propose a small and generic addition to the scheduling
framework: an optional extension point through which a plugin can report that an earlier preemption
by a given pod has not finished settling.

```go
// PreemptionSettlingPlugin is an optional interface for plugins that reclaim
// resources asynchronously. DefaultPreemption consults it before starting a new
// preemption for a pod that has already preempted.
type PreemptionSettlingPlugin interface {
    Plugin
    // PreemptionSettling reports whether resources freed by an earlier preemption
    // by this pod are still being reclaimed, and if so why.
    PreemptionSettling(ctx context.Context, pod *v1.Pod) (bool, string)
}
```

The framework gains a `RunPreemptionSettlingPlugins` method that calls each registered plugin
implementing the interface and stops at the first one that reports the pod as settling. Plugins that
do not implement it are skipped, as they are for the other optional extension points.
`PodEligibleToPreemptOthers` calls it alongside its existing terminating-pod check, and the returned
reason is reported in the pod's scheduling condition.

This extension point is not DRA-specific and makes an existing hard-coded heuristic pluggable. The
`dynamicresources` plugin implements it by reporting whether the pod has a live nomination with
claims that are still allocated.

As a secondary optimization, the plugin's queueing hints can withhold a wake-up while a nomination
is unsettled, so that the preemptor is not retried pointlessly. This is not sufficient on its own,
because the pod can also be woken by unrelated events and by the periodic flush of the
unschedulable queue, but it avoids wasted scheduling attempts.

#### Why claims rather than devices

The obvious alternative is to nominate the *devices* that the preemptor is expected to receive,
analogous to how `nominatedNodeName` records a node. We nominate claims instead, because the device
is the wrong unit:

* With **consumable capacity**, the preemptor may need only a share of a device, and that share may
  be freed by several victims. There is no device-level entity corresponding to a share, so a device
  nomination would have to hold the whole device, including capacity that the victims never held and
  that other pods can still use. Recording the victims' claims holds exactly their shares.
* With **partitionable devices**, the released capacity can be recombined and re-partitioned, so the
  partition the preemptor ends up using is generally not one of the partitions the victims held.
  Partitions are enumerated in the ResourceSlice, so there is a name to nominate, but picking one
  commits to a guess: overlapping partitions draw on the same counters, so holding the wrong
  partition fails to protect what the preemptor needs while blocking partitions that others could
  have used. Holding the victims' claims holds the counters that the preemption actually freed,
  without predicting which partition will be built from them.
* A device nomination has no checkable completion condition. There is no observable event that says
  a nominated device has been freed, so there is no reliable point at which to release the hold. A
  claim's `Status.Allocation` being cleared is directly observable.

Claims do not have these problems, because the claim is the unit at which capacity is actually
released. "These claims will be released" remains true and checkable regardless of how the
underlying capacity is subsequently recombined.

### Deferred: workload-aware preemption

Workload-aware preemption ([KEP-5710]) removes candidate victims by mutating the scheduler snapshot
and then reruns the pod group scheduling algorithm with fresh CycleStates, instead of calling
`PreFilterExtensions.RemovePod` for each victim. A plugin that learns about removals only through
`RemovePod`, as specified here, therefore goes on counting the victims' devices as allocated, and
the preemption frees nothing. The pod group stays pending, which is the behavior without this
feature. No state is corrupted and no device is allocated twice.

Supporting that path needs two additions that we prefer to design separately:

* The plugin has to rebuild its allocated state by reconciling the ResourceClaims it reads against
  the pods present in the `NodeInfos` that `PreFilter` receives. Doing so safely requires
  distinguishing a consumer that has been removed from the snapshot from one that the plugin has
  merely not observed yet, such as a pod that another scheduler has reserved a claim for but not
  yet bound.
* A nomination has to be owned by the preempting pod group rather than by a single pod. One
  workload-aware preemption produces a nominated placement for every member of the group, all at
  the same priority, so per-pod nominations covering the same released claims would mask the
  capacity from each other and the group would never schedule.

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
* a nomination expires and the preemptor becomes eligible to preempt again if its claims are never
  deallocated.

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

This feature targets Beta directly, without an Alpha stage.

#### Beta

- Feature implemented behind a feature flag
- Unit, integration and e2e tests completed and enabled, covering the settling window scenarios
  described in the test plan
- Tests are in Testgrid and linked in the KEP
- The cost of the preemption simulation measured with scheduler_perf
- Metrics for claim nominations exposed and documented

#### GA

- 2 examples of real-world usage
- Allowing time for feedback
- A decision on whether claim nominations need to survive a scheduler restart, informed by the
  metrics gathered during Beta


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

Disabling it also discards any claim nominations that are currently held, which releases the
capacity being held for preemptors that have not yet been scheduled. Nothing is persisted, so
there is no state to clean up and no reconciliation is required.

###### What happens if we reenable the feature if it was previously rolled back?

That it was previously rolled back have no impact. Reenabling it just means that
preemption of pods referencing ResourceClaims will again be considered.

###### Are there any tests for feature enablement/disablement?

We will cover this scenario in both unit tests and integration tests.

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

If the `scheduler_preemption_victims` increases significantly when the feature is enabled, but we don't
see a corresponding increase in pods being scheduled, we should investigate. It would suggest
that pods are being preempted incorrectly and the higher-priority pods are not actually
being scheduled.

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
  - Metric name: `scheduler_preemption_victims`. This is not specific to DRA and carries no
    labels, so it can only be compared across the cluster before and after the gate is enabled.
  - Metric name: `scheduler_dra_claim_nominations`, a gauge of the claim nominations currently
    held by the scheduler
  - Metric name: `scheduler_dra_claim_nominations_total`, a counter of resolved nominations,
    labeled by outcome: `scheduled` when the preemptor was scheduled, `expired` when the
    nomination timed out, `preempted` when the held capacity was taken by a pod of higher
    priority, and `discarded` for the remaining cases, such as the preemptor being deleted or
    preempting again
  - Components exposing the metric: kube-scheduler

###### Are there any missing metrics that would be useful to have to improve observability of this feature?

No

### Dependencies

<!--
This section must be completed when targeting beta to a release.
-->

###### Does this feature depend on any specific services running in the cluster?

Yes. The resourceclaim controller in kube-controller-manager deallocates the ResourceClaims of
preempted pods, and preemption for DRA cannot complete until it has done so. If the controller is
unavailable or lagging, preemptors remain unschedulable and their claim nominations eventually
expire; see [What are other known failure modes?](#what-are-other-known-failure-modes).

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
earlier preemption to settle, containing the allocation results of the claims that were released.
The number of such nominations is bounded by the number of in-flight preemptions, and each is
discarded when the preemptor is scheduled or when the nomination expires.

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
be deleted and their ResourceClaims cannot be deallocated, so claim nominations will expire. Once
the API server is available again, preemptors whose nominations expired may preempt a further set
of victims.

###### What are other known failure modes?

- The resourceclaim controller does not deallocate a nominated claim, for example because it is
  unhealthy. The preemptor's nomination expires and it preempts a further set of victims. This is
  visible as a rising `expired` count on `scheduler_dra_claim_nominations_total`, and as a
  persistently non-zero `scheduler_dra_claim_nominations`. Detection is by those metrics; there is no
  mitigation in the scheduler beyond the expiry bound, since the scheduler cannot make the
  controller progress.
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
  workload-aware preemption is deferred to a later revision. Targeted directly at Beta.

## Drawbacks

It complicates the logic in the dynamicresources plugin and can lead
to slower preemption when pods are using DRA.

Holding capacity for a nominated preemptor can also leave devices idle for the duration of the
settling window, and more of a device may be held than the preemptor actually needs. See
[Risks and Mitigations](#risks-and-mitigations).

## Alternatives

Kubernetes has a well-established framework for preemption, and this KEP uses it rather than adding
a separate mechanism for DRA. The alternatives below all concern the settling window described in
[Preemption settling for DRA](#preemption-settling-for-dra).

**Nominating devices rather than claims.** Discussed in
[Why claims rather than devices](#why-claims-rather-than-devices).

**Allocating the preemptor's claims eagerly.** Rather than recording a nomination, the scheduler
could write the allocation computed during preemption to the preemptor's ResourceClaims
immediately and mark it as not yet usable. We rejected this because the victims' claims still hold
the same devices until t3, so the two allocations would conflict; because the allocation was
computed against a simulated state and may no longer be valid once the cluster state converges; and
because it would need an unwind path for the case where the preemptor ultimately cannot be
scheduled. Recording the claims that will be released avoids all three, because it records an input
to the allocator rather than one of its outputs.

**Deallocating the victims' claims from the scheduler.** The scheduler could clear
`Status.Allocation` itself once the victim pods are gone, instead of waiting for the resourceclaim
controller. This shortens the interval from t1 to t3 but does not remove it, does nothing for the
grace period from t0 to t1, and does not prevent device stealing between t3 and t4. It would also
have to cope with the scheduler's informer cache not yet reflecting its own update. It is a
possible later optimization rather than a substitute.

**Waiting for the victims to terminate within the scheduling cycle.** This removes the settling
window altogether, but blocks the scheduler for the length of the victims' grace periods and is
therefore not acceptable.

**Deferring preemption from the plugin's PostFilter rather than through a framework extension
point.** The `dynamicresources` plugin is registered immediately before `DefaultPreemption` at the
PostFilter extension point, and `RunPostFilterPlugins` returns as soon as a plugin reports
`UnschedulableAndUnresolvable`. The plugin could therefore return that status while the pod has an
unsettled nomination, and `DefaultPreemption` would not run for that scheduling attempt. This
achieves the same deferral with no change to the scheduling framework.

We did not choose it as the primary design for three reasons. It depends on the relative ordering of
the two plugins, which is a property of the default configuration rather than a guarantee, so a
profile that reorders the plugins or disables the plugin's PostFilter would lose the protection
silently. It suppresses every PostFilter plugin registered after `dynamicresources`, not only
`DefaultPreemption`. And it leaves the assumption built into `PodEligibleToPreemptOthers`, that a
preemption has finished settling once the victim pods are gone, incorrect for every other resource
that is reclaimed by a controller rather than by the removal of the pod. It remains a workable
fallback if the extension point is not accepted.

**Accepting the behavior and relying on priority.** Preemption is best-effort, so the scheduler
could simply let the preemptor retry. We rejected this because the situation that motivates this
KEP, many pods contending for scarce accelerators, is precisely the one in which a preemptor can be
starved indefinitely while each of its attempts evicts a further set of victims.

## Infrastructure Needed (Optional)

No
