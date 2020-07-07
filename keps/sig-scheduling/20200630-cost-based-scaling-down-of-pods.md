---
title: Cost based scaling down of pods
authors:
  - "@ingvagabund"
owning-sig: sig-scheduling
participating-sigs:
  - sig-apps
reviewers:
  - TBD
approvers:
  - TBD
editor: TBD
creation-date: 2020-06-30
last-updated: 2020-07-20
status: provisional
---

# Cost based scaling down of pods

## Table of Contents

<!-- toc -->
- [Summary](#summary)
- [Motivation](#motivation)
  - [Goals](#goals)
  - [Non-Goals](#non-goals)
- [Proposal](#proposal)
  - [Examples of strategies](#examples-of-strategies)
    - [Balancing duplicates among topological domains](#balancing-duplicates-among-topological-domains)
    - [Pods not tolerating taints first](#pods-not-tolerating-taints-first)
    - [Minimizing pod anti-affinity](#minimizing-pod-anti-affinity)
  - [Rank normalization and weighted sum](#rank-normalization-and-weighted-sum)
  - [User Stories [optional]](#user-stories-optional)
    - [Story 1](#story-1)
    - [Story 2](#story-2)
    - [Story 3](#story-3)
  - [Implementation Details/Notes/Constraints](#implementation-detailsnotesconstraints)
    - [Phases](#phases)
  - [Option A (field in a pod status)](#option-a-field-in-a-pod-status)
  - [Option B (CRD for a pod group)](#option-b-crd-for-a-pod-group)
  - [Workflow example](#workflow-example)
  - [Risks and Mitigations](#risks-and-mitigations)
- [Design Details](#design-details)
  - [Test Plan](#test-plan)
  - [Graduation Criteria](#graduation-criteria)
  - [Upgrade / Downgrade Strategy](#upgrade--downgrade-strategy)
  - [Version Skew Strategy](#version-skew-strategy)
- [Implementation History](#implementation-history)
- [Alternatives [optional]](#alternatives-optional)
<!-- /toc -->

## Summary

Cost ranking pods through an external component and scaling down pods based
on the cost allows to employ various scheduling strategies to keep a cluster
from diverging from an optimal distribution of resources.
Providing an external solution for selecting the right victim allows to improve ability
to preserve various conditions such us balancing pods among failure domains, keeping
aligned with security requirements or respecting application policies.
Allowing controllers to be free of any scheduling strategy, yet to be aware
of impact of removing pods on the overall cluster scheduling plan, helps to reduce
cost of re-scheduling resources.

## Motivation

Scaling down a set of pods does not always results in optimal selection of victims.
The scheduler relies on filters and scores which may distribute the pods wrt. topology
spreading and/or load balancing constraints (e.g. pods uniformly balanced among zones).
Application specific workloads may prefer to scale down short-running pods and favor long-running pods.
Selecting a victim with a trivial logic can unbalance the topology spreading
or have jobs that accumulated work to be lost in vain.
Given it's a natural property of a cluster to shift workloads in time,
decision made by a scheduler at some time is as good as its ability to predict future demands.
The default kubernetes scheduler was constructed with a goal to provide high throughput
at the cost of being simple. Thus, it is quite easy to diverge from the scheduling plan.
In contrast, descheduler allows to help to re-balance the plan and get closer to
the scheduler constraints. Yet, it is designed to run and adjust the cluster periodically (e.g. each hour).
Therefore, unusable for scaling down purposes (which require immediate action).

On the other hand each controller with a scale down operation has its own
implementation of a victim selection logic.
The decision making logic does not take into account a scheduling plan.
Extending each such controller with additional logic to support various scheduling
constraints is impractical. In cases a proprietary solution for scaling down is required,
it's impossible. Also, controllers do not necessarily have a whole cluster overview
so its decision does not have to be optimal.
Therefore, it's more feasible to locate the logic outside of a controller.

In order to support more informed scaling down operation while keeping scheduling plan in mind,
additional decision logic that can be extended based on applications requirements is needed.

### Goals

- Controllers with scale down operation are allowed to select a victim while still respecting a scheduling plan
- External component is available that can rank pods based on how much they diverge from a scheduling plan when deleted

### Non-Goals

- Allow to employ strategies that require cost re-computation after scaling up/down (with a support from controllers, e.g. backing-off)

## Proposal

Proposed solution is to implement an optional cost-based component that will be watching all
pods (or its subset) and nodes (potentially other objects) present in a cluster.
Assigning each pod a cost based on a set of scheduling constraints.
At the same time extending controllers logic to utilize the pod cost when selecting a victim during scale down operation.

The component will allow to select a different list of scheduling constraints for each targeted
set of pods. Each pod in a set will be given a cost based on how much important it is in the set.
The constraints can follow the same rules as the scheduler (through importing scheduling plugins)
or be custom made (e.g. wrt. to application or proprietary requirements).
The component will implement a mechanism for ranking pods.
<!-- Either by annotating a pod, updating its status, setting a new field in pod's spec
or creating a new CRD which will carry a cost. -->
Each controller will have a choice to either ignore the cost or take it into account
when scaling down.

This way, the logic for selecting a victim for the scaling down operation will be
separated from each controller. Allowing each consumer to provide its own
logic for assigning costs. Yet, having all controllers to consume the cost uniformly.

Given the default scheduler is not a source of truth about how a pod should be distributed
after it was scheduled, scaling down strategies can exercise completely different approaches.

Examples of scheduling constraints:
- choose pods running on a node which have a `PreferNoSchedule` taint first
- choose youngest/oldest pods first
- choose pods minimizing topology skew among failure domains (e.g. availability zones)

The goal of the proposal is not to provide specific strategies for more informed scaling down operation.
The primary goal is to provide a mechanism and have controllers implement the mechanism.
Allowing consumers of the new component to define their own strategies.

### Examples of strategies

Strategies can be divided into two categories:
- scaling down/up a pod group does not require rank re-computation
- scaling down/up a pod group requires rank re-computation

#### Balancing duplicates among topological domains

- Evict pods while minimizing skew between topological domains
- Each pod can be given a cost based on how old/young it is in the same domain:
  - if a pod is the first one in the domain, rank the pod with cost `1`
  - if a pod was created second to the domain, rank the pod with cost `2`
  - continue this way until all pods in all domains are ranked
  - higher rank of a pod, the sooner the pod gets removed

#### Pods not tolerating taints first

- Evict pods that do not tolerate taints before pods that tolerate taints.
- Each pod can be given a cost based on how many taints are not tolerated
  - higher rank of a pod, the sooner the pod gets removed

#### Minimizing pod anti-affinity

- Evict pods maximizing anti-affinity first
- Pod that improves anti-affinity on a node gets higher rank
- Given multiple pod groups can be part of anti-affinity group, scaling down
  a single pod in a group requires re-computation of pod ranks of all pods
  taking part. Also, only a single pod can be scaled down at a time.
  Otherwise, ranks no longer have to provide optimal victim selection.

In the provided examples the first two strategies do not require rank re-computation.

### Rank normalization and weighted sum

In order to allow pod ranking by multiple strategies/constraints, it's important
to normalize ranks. On the other hand, rank normalization requires all strategies
to re-compute all ranks every time a pod is created/deleted. To eliminate the need
to re-compute, each strategy can introduce a threshold where every pod rank
exceeding the threshold gets rounded to the threshold.
E.g. if a topology domain has at least 10 pods, 11-th and other pods get the same
rank as 10-th pod.
With the threshold based normalization multiple strategies can rank a pod group
which can be used to compute weighted rank through all relevant strategies.

### User Stories [optional]

#### Story 1

From [@pnovotnak](https://github.com/kubernetes/kubernetes/issues/4301#issuecomment-328685358):

```
I have a number of scientific programs that I've wrapped with code to talk
to a message broker that do not checkpoint state. The cost of deleting the resource
increases over time (some of these tasks take hours), until it completes the current unit of work.

Choosing a pod by most idle resources would also work in my case.
```

#### Story 2

From [@cpwood](https://github.com/kubernetes/kubernetes/issues/4301#issuecomment-436587548)

```
For my use case, I'd prefer Kubernetes to choose its victims from pods that are running on nodes which have a PreferNoSchedule taint.
```

#### Story 3

From [@barucoh](https://github.com/kubernetes/kubernetes/issues/89922)

```
A deployment with 3 replicas with anti-affinity rule to spread across 2 AZs scaled down to 2 replicas in only 1 AZ.
```

### Implementation Details/Notes/Constraints

Currently, the descheduler does not allow to immediately react on changes in a cluster.
Yet, with some modification another instance of the descheduler (with different set of strategies)
might be ran in watch mode and rank each pod as it comes.
Also, once the scheduling framework gets migrated into its own repository,
scheduling plugins can be vendored as well to provide some of the core scheduling logic.

The pod ranking is best-effort so in case a controller is to delete more than one pod
it selects all the pods with the highest cost and remove those.
In case a pod fails to be deleted during the scale down operation and results in resuming the operation in the next cycle,
it may happen pods get ranked differently and a different set of victim pods gets selected.

Once a pod is removed, ranks of others pods might be required to get re-computed.
Unless strategies that do not require re-computation are deployed.
By default, all pods owned by a controller template has to be ranked.
Otherwise, a controller falls back to each original selection victim logic.
Resp. it can be configured to wait or back-off.
Also, the ranking strategies can be configured to target only selected sets of pods.
Thus, allowing a controller to employ cost based selection only when more sophisticated
logic is required and available.

During alpha phase, each controller utilizing the pod ranking will feature gate the new logic.
Starting by utilizing a pod annotation (e.g. `scheduling.alpha.kubernetes.io/cost`)
which can be eventually promoted to either a field in pod's spec or moved under CRD (see further).

If strategies requiring rank re-computation are employed, it's more practical to define
a CRD for a pod group and have all the costs in a single place to avoid desynchronization
of ranks among pods.

#### Phases

Phase 1:
- add support for strategies which do not need rank re-computation of a pod group
- only a single strategy can be ran to rank pods (unless threshold based normalization is applied)
- use annotations to hold a single pod cost

Phase 2A:
- promote pod cost annotation to a pod status field
- no synchronization of pods in a pod group, harder support of strategies which require rank re-computation

Phase 2B:
- use a CRD to hold costs of all pods in a pod group (to synchronize re-computation of ranks)
- add support for strategies which require rank re-computation

### Option A (field in a pod status)

Store a post cost/rank under pod's status so it can be updated only by component
who has permission to update pod status.

```go
// PodStatus represents information about the status of a pod. Status may trail the actual
// state of a system, especially if the node that hosts the pod cannot contact the control
// plane.
type PodStatus struct {
...
	// More info: https://kubernetes.io/docs/concepts/workloads/pods/pod-lifecycle#pod-cost
	// +optional
	Cost int `json:"cost,omitempty" protobuf:"bytes,...,opt,name=cost"`
...
}
```

Very simple, reading the field directly from a pod status.
No additional apimachinery logic.

### Option B (CRD for a pod group)

```yaml
apiVersion: scheduling.k8s.io/v1alpha1
kind: PodGroupCost
metadata:
  name: rc-guestbook-fronted
  namespace: rc-guestbook-fronted-namespace
spec:
  owner:
    kind: ReplicationController
    name: rc-guestbook-fronted  // may be redundant
  costs:
    "rc-guestbook-fronted-pod1": 4
    "rc-guestbook-fronted-pod2": 8
    ...
    "rc-guestbook-fronted-podn": 2
```

More suitable for keeping all pod costs from a pod group in sync.
Controllers will need to take into account the new CRD (adding informers).
A CR will live in the same namespace as underlying pod group (RC, RC, etc.).

### Workflow example

**Scenario**: pod group of 12 pods, 3 AZs (2 nodes per each AZ), pods are evenly spread among all zones

1. Assuming a pod group is supposed to respect topology spreading and scale down
operation is to minimize topology skew between domains.
1. **Ranking component**: The component is configured to rank pods based on their presence in a topology domain
1. **Ranking component**: The component notices the pods, analyzes the pod group and ranks the pods in the following manner (`{PodName: Rank}`):
   - AZ1: {P1: 1, P2: 2, P3: 3, P4: 4} (P1 getting 1 as it was created first in the domain, P2 getting 2, etc.)
   - AZ2: {P5: 1, P6: 2, P7: 3, P8: 4}
   - AZ3: {P9: 1, P10: 2, P11: 3, P12: 4}
1. **A controller**: Scale down operation of the pod group is requested
1. **A controller**: Scale down logic of a controller selects one of P4, P8 or P12 as a victim (e.g. P8)
1. Topology skew is now `1`
1. **Ranking component**: No need to re-compute ranks since the ranking does not depend on the pod group size
1. **A controller**: Scaling down one more time selects one of {P4, P12}
1. Topology skew is still `1`

### Risks and Mitigations

It may happen the ranking component does not rank all relevant pods in time.
In that case a controller can either choose to ignore the cost. Or, it can back-off
with a configurable timeout and retry the scale down operation once all pods in
a given set are ranked.

From the security perspective a malicious code might assign pod a different cost
with a goal to remove more vital pods to harm a running application.
How much is using annotation safe? Might be better to use pod status
so only clients with pod/status update RBAC are allowed to change the cost.

In case a strategy needs to re-compute costs after scale down operation and
the component stops working (for any reason), a controller might scale down
incorrect pod(s) in the next request. More reasons to constraint strategies
to not need to re-compute pod costs.

In case a scaling down process is too quick, the component may be too slow to
recompute all scores and provide suboptimal/incorrect costs.

In case two or more controllers own a pod group (through labels), scaling down the group by one
controller can result in scale up the same group by another controller.
Entering an endless loop of scaling up and down. Which may result in unexpected
behavior. Leaving a subgroup of pods unranked.

Deployment upgrades might have different expectations when exercising a rolling update.
They could just completely ignore the costs. Unless, it's acceptable to scale down by one
and wait until the costs are recomputed when needed.

## Design Details

### Test Plan

**Scaling down respects pod ranking**:
In the simplest case the component ranks pods in a group.
The goal is to validate all pods are scaled down in an order
respecting ranks of all pods in a pod group.

**A controller ignores ranks if at least one pod is missing a rank**:
Testing the case where not every pod in a pod group is ranked.
A controller falls down to its original behavior if not all pods
in a pod group are ranked after specified timeout (back-off simulation).

**In case a strategy requiring re-computation after a pod group size changed**:
- a controller will not scale down by two (only by one) replicas
- a controller will not scale down by one until pod ranks are re-computed after previous scale down operation

### Graduation Criteria

- Alpha: Initial support for taking pod cost into account when scaling down in controllers. Disabled by default.
- Beta: Enabled by default

### Upgrade / Downgrade Strategy

Scaling down based on a pod cost is optional. If no cost is present, scaling down falls back to the original behavior.

### Version Skew Strategy

A controller either recognizes pod's cost or it does not.

## Implementation History

- KEP Started on 06/30/2020

## Alternatives [optional]

- Controllers might use a webhook and talk to the component directly to select a victim
- Some controllers might improve their decision logic to cover specific use cases (e.g. introduce new policy for sorting pods based on information located in pod objects)
