---
title: Even Pods Spreading
authors:
  - "@Huang-Wei"
owning-sig: sig-scheduling
reviewers:
  - "@bsalamat"
  - "@lavalamp"
  - "@krmayankk"
  - "@ahg-g"
  - "@alculquicondor"
approvers:
  - "@ahg-g"
  - "@alculquicondor"
creation-date: 2019-02-21
last-updated: 2020-01-21
status: implementable
---

# Pod Topology Spread

## Table of Contents

<!-- toc -->
- [Release Signoff Checklist](#release-signoff-checklist)
- [Terms](#terms)
- [Summary](#summary)
- [Motivation](#motivation)
  - [Goals](#goals)
  - [Non-Goals](#non-goals)
- [Proposal](#proposal)
  - [User Stories](#user-stories)
    - [Story 1](#story-1)
    - [Story 2](#story-2)
  - [Risks and Mitigations](#risks-and-mitigations)
- [Design Details](#design-details)
  - [API](#api)
    - [Option 1](#option-1)
    - [Option 2 (preferred)](#option-2-preferred)
  - [MaxSkew](#maxskew)
  - [How User Stories are Addressed](#how-user-stories-are-addressed)
  - [Pros/Cons](#proscons)
  - [Test Plan](#test-plan)
  - [Graduation Criteria](#graduation-criteria)
- [Alternatives](#alternatives)
- [Impact to Other Features](#impact-to-other-features)
- [Implementation History](#implementation-history)
<!-- /toc -->

## Release Signoff Checklist

- [x] kubernetes/enhancements issue in release milestone, which links to KEP (this should be a link to the KEP location in kubernetes/enhancements, not the initial KEP PR)
- [x] KEP approvers have set the KEP status to `implementable`
- [x] Design details are appropriately documented
- [x] Test plan is in place, giving consideration to SIG Architecture and SIG Testing input
- [x] Graduation criteria is in place
- [x] "Implementation History" section is up-to-date for milestone
- [x] User-facing documentation has been created in [kubernetes/website], for publication to [kubernetes.io]
- [ ] Supporting documentation e.g., additional design documents, links to mailing list discussions/SIG meetings, relevant PRs/issues, release notes

## Terms

- **Topology:** describe a series of worker nodes which belongs to the same
  region/zone/rack/hostname/etc. In terms of Kubernetes, they're defined and
  grouped by node labels.
- **Affinity**: if not specified particularly, "Affinity" refers to
  `NodeAffinity`, `PodAffinity` and `PodAntiAffinity`.
- **CA**: Cluster Autoscaler.
  [CA](https://github.com/kubernetes/autoscaler/tree/master/cluster-autoscaler)
  is a tool that automatically adjusts the size of the Kubernetes cluster upon
  specific conditions.

## Summary

The `PodTopologySpread` feature gives users more fine-grained control on
distribution of pods scheduling, so as to achieve better high availability and
resource utilization.

## Motivation

In Kubernetes, "Affinity" related directives are aimed to control how pods are
scheduled - more packed or more scattering. But right now only limited options
are offered: for `PodAffinity`, infinite pods can be stacked onto qualifying
topology domain(s); for `PodAntiAffinity`, only one pod can be scheduled onto a
single topology domain.

This is not an ideal situation if users want to put pods evenly across different
topology domains - for the sake of high availability or saving cost. And regular
rolling upgrade or scaling out replicas can also be problematic. See more
details in [user stories](#user-stories).

### Goals

- Pod Topology Spread Constraints is calculated among pods instead of apps API (such as
  Deployment, ReplicaSet).
- Pod Topology Spread Constraints can be either a predicate (hard requirement) or a priority
  (soft requirement).

### Non-Goals

- Pod Topology Spread Constraints is NOT calculated on an application basis. In other words, it's
  not only applied within replicas of an application, but also applied to
  replicas of other applications if appropriate.
- "Max number of pods per topology" is NOT a goal.
- Scale-down on an application is not guaranteed to achieve desired pods spreading
  in the initial implementation.

## Proposal

### User Stories

#### Story 1

As an application developer, I want my application pods to be scheduled onto
specific topology domains as even as possible. Current status is that pods may
be stacked onto a specific topology domain. (see
[#68981](https://github.com/kubernetes/kubernetes/issues/68981))

#### Story 2

As an application developer, I want my application pods not to co-exist with
specific pods (via PodAntiAffinity). But in some cases, it'd be favorable to
tolerate "violating" pods in a manageable way. For example, suppose an app
(replicas=2) is using PodAntiAffinity and deployed onto a 2-nodes cluster, and
next the app needs to perform a rolling upgrade, then a third replacement pod is
created, but it failed to be placed due to lack of resource. In this case,

- if CA is enabled, a new machine will be provisioned to hold the new pod
  (although old replicas will be deleted afterwards) (see
  [#40358](https://github.com/kubernetes/kubernetes/issues/40358))
- if CA is not enabled, it's a deadlock since the replacement pod can't be
  placed. The only workaround at this moment is to update app strategyType from
  "RollingUpdate" to "Recreate".

Neither of them is an ideal solution. A promising solution is to give user an
option to trigger "toleration" mode when the cluster is out of resource. Then in
aforementioned example, a third pod is "tolerated" to be put onto node1 (or
node2). But keep it in mind, this behavior is only triggered upon resource
shortage. For a 3-nodes cluster, the third pod will still be placed onto node3
(if node3 is capable).

### Risks and Mitigations

The feature requires additional processing for pods that use it and it is ok to
have some performance overhead. But we will make sure our implementation will
not have any performance penalty for pods that do not use this feature.

## Design Details

### API

A new structure called `TopologySpreadConstraint` is introduced which acts as a
standalone spec and is applied to `pod.spec`. It's only effective when it's not
nil.

```go
type PodSpec struct {
    ......
    // TopologySpreadConstraints describes how a group of pods are spread
    // If specified, scheduler will enforce the constraints
    // +optional
    TopologySpreadConstraints []TopologySpreadConstraint
    ......
}
```

#### Option 1

Inside `TopologySpreadConstraint`, we need hard affinityTerms (similar with
`PodAffinityTerm`) and soft affinityTerms (similar with
`WeightedPodAffinityTerm`). This describes when we perform even distribution,
which pods are considered as a group.

```go
type TopologySpreadConstraint struct {
    // MaxSkew describes the degree of imbalance of pods spreading.
    // It's the max difference between the number of matching pods in any two
    // topology domains of a given topology type.
    // Default value is 1 and 0 is not allowed.
    MaxSkew int32
    // TopologyKey defines where pods are placed evenly
    TopologyKey string
    // Similar with the same field in PodAffinity/PodAntiAffinity
    // +optional
    RequiredDuringSchedulingIgnoredDuringExecution []PodAffinityTerm
    // Similar with the same field in PodAffinity/PodAntiAffinity
    // +optional
    PreferredDuringSchedulingIgnoredDuringExecution []WeightedPodAffinityTerm
}
```

#### Option 2 (preferred)

Another option is to flatten "required" and "preferred" podAffinityTerms, and
eliminate embedded "TopologyKey":

```go
type UnsatisfiableConstraintResponse string

const (
    // do not schedule a pod in all circumstances
    DoNotSchedule UnsatisfiableConstraintResponse = "DoNotSchedule"
    // schedule a pod despite of any circumstance
    ScheduleAnyway UnsatisfiableConstraintResponse = "ScheduleAnyway"
)

type TopologySpreadConstraint struct {
    // MaxSkew describes the degree of imbalance of pods spreading.
    // It's the max difference between the number of matching pods in any two
    // topology domains of a given topology type.
    // For example, in a 3-zone cluster, currently pods with the same labelSelector
    // are spread as 1/1/0:
    // - if MaxSkew is 1, incoming pod can only be scheduled to zone3 to become 1/1/1;
    // schedule it onto zone1(zone2) will make the ActualSkew(2) violates MaxSkew(1)
    // - if MaxSkew is 2, incoming pod can be scheduled to any zone.
    // Default value is 1 and 0 is not allowed.
    MaxSkew int32
    // TopologyKey is the key such that we consider each value as a "bucket";
    // we try to put balanced number of pods into each bucket.
    TopologyKey string
    // WhenUnsatisfiable indicates how to deal with a pod if it doesn't satisfy
    // the spreading constraint.
    // - DoNotSchedule (default) tells the scheduler not to schedule it
    // - ScheduleAnyway tells the scheduler to still schedule it
    // Note: it's considered as "Unsatisfiable" only when actual skew on all nodes
    // exceeds "MaxSkew".
    WhenUnsatisfiable UnsatisfiableConstraintResponse
    // Label selector for pods. This's enforced by scheduler to check which pods
    // should be recognized as a group to satisfy the spreading constraint.
    Selector *metav1.LabelSelector
}
```

### MaxSkew

`MaxSkew` is the core of this KEP, so the exact semantics are clarified as below:

- how Skew is calculated and enforced

Suppose we have a 3-zone cluster, currently pods with the same labelSelector are
spread as 1/1/0. Internally we compute an "ActualSkew" for each topology
domain representing "matching pods in this topology domain" minus "minimum
matching pods in any topology domain", so for this 1/1/0 cluster, the ActualSkew
for each zone is 1(1-0)/1(1-0)/0(0-0). (If the spreading is 3/2/1, the
ActualSkew for each zone will be 2(3-1)/1(2-1)/0(1-1))

The internal computation logic would be to find nodes satisfying "ActualSkew <=
MaxSkew". Let's go back to the 1/1/0 example:

If MaxSkew is 1, incoming pod can only be scheduled to zone3 to become 1/1/1;
because schedule it onto zone1(zone2) will make the ActualSkew(2) violates
MaxSkew(1).

If MaxSkew is 2, incoming pod can be scheduled to any zone.

**NOTE:** If NodeAffinity or NodeSelector is defined, spreading is applied to
nodes that pass those filters. For example, if NodeAffinity chooses zone1 and
zone2 and there are 10 zones in the cluster, pods are spread in zone1 and zone2
only and MaxSkew is enforced only on these two zones.

- chicken/egg problem

Let's say we have a 3-zone cluster, and there is no pod in any node yet. Here
comes a pod, and it wants to be scheduled to a zone which has pods with label
`foo`. Obviously, there is no qualified node. However, we don't stop here;
instead, we proceed to check if the incoming pod matches itself on its labels.
If it does, we would think any node is a fit.

This is actually an existing implication in PodAffinity algorithm. I just want
to put here again to avoid confusion. And **below examples are all based the
assumption that incoming pod matches itself on its labels**.

- matching number and min matching number

"matching" number is the number of pods matched on topology domain (defined by
the global topologyKey). Suppose we have a 3-zone cluster, and there are 3 pods
in zone1, 2 pods in zone2, 1 pod in zone3. And all pods carry label `foo`:

```
+----------------------------+----------------------------+--------+
|            zone1           |            zone2           |  zone3 |
+----------------------------+----------------------------+--------+
| node1a |  node1b  | node1c |  node2a  | node2b | node2c | node3a |
+--------+----------+--------+----------+--------+--------+--------+
|   pod  | pod, pod |        | pod, pod |        |        |   pod  |
+--------+----------+--------+----------+--------+--------+--------+
```

Now let's say there comes a pod, it wants to be placed along with pods which
carries label `foo` in zones.

If global topologyKey is "zone" and maxSkew is "1", then incoming pod can only
be put into zone3 because for zone1, it violates `matching num (3) - min
matching num (1) < maxSkew (1)`. Zone2 violate the formula the same way.

If global topologyKey is "node" and maxSkew is "1", things are slightly
different. Min matching num becomes 0 now, and hence only node1c, node2b and
node2c are qualified candidates.

- what if a topology domain is infeasible

Suppose we have pods distribution in a 3-zone cluster as 3/3/0, and all pods
have label `foo`:

```
+-------------+-------------+--------------------+
|    zone1    |    zone2    | zone3 (infeasible) |
+-------------+-------------+--------------------+
| pod,pod,pod | pod,pod,pod |                    |
+-------------+-------------+--------------------+
```

And we have an incoming pod which wants to be scheduled with pods which carry
label `foo` in zones. And suppose all nodes in zone3 are infeasible, e.g. due to
taints or lack of resources. In this case:

If it's a hard requirement, we treat the `min matching num` as 0, which means
incoming pod would fail to be scheduled.

If it's a soft requirement, we treat the `min matching num` as 3 instead of 0,
which means incoming pod can be placed onto zone1 or zone2.

- (more cases) when a topology domain is infeasible

    > Suppose maxSkew is 1: (~~zone~~ means the zone is infeasible)

    - for a "1/1/~~0~~" cluster, pod can't be placed onto any zone if it's a
      Predicate; zone1 and zone2 are equally preferred if it's a Priority
    - for a "2/1/~~0~~" cluster, pod can't be placed onto any zone if it's a
      Predicate; zone2 is preferable over zone1 if it's a Priority
    - for a "1/1/~~1~~" cluster, pod can be placed onto zone1 or zone2 if it's a
      Predicate; zone1 and zone2 are equally preferred if it's a Priority
    - for a "2/1/~~1~~" cluster, pod can be placed onto zone2 if it's a
      Predicate; zone2 is preferable over zone1 if it's a Priority

- when formula check is enforced

We only enforce the formula check upon new pod scheduling. In other words, if
pods become imbalanced (due to explicit taints, lack of resources, or node
lost), we don't do proactive re-scheduling. Our goal is to not make things
worse.

### How User Stories are Addressed

In terms of story 1, users can define a `TopologySpreadConstraint` to achieve an
even pods distribution:

```yaml
spec:
  topologySpreadConstraint:
    maxSkew: 1
    topologyKey: k8s.io/zone
    whenUnsatisfiable: DoNotSchedule
    selector:
      matchLabels:
        app: foo
```

And it can work together with NodeSelector/NodeAffinity. (check
[MaxSkew](#maxskew) for more details)

Similarly, story 2 can also be addressed using above solution.

And the pseudo algorithms below explain the processing flow in a nutshell.

- Predicate

```bash
for each candidate node; do
    if "TopologySpreadConstraint" is enabled for the pod being scheduled; then
        # minMatching num is globally calculated
        count number of matching pods on the topology domain this node belongs to
        if "matching num - minMatching num" < "MaxSkew"; then
            approve it
        fi
    fi
done
```

- Priority

```bash
for each candidate node; do
    if "TopologySpreadConstraint" is enabled for the pod being scheduled; then
        # minMatching num is calculated across node list filtered by Predicate phase
        count number of matching pods on the topology domain this node belongs to
        calculate the value of "matching num - minMatching num" minus "MaxSkew"
        the lower, the higher score this node is ranked
    fi
done
```

### Pros/Cons

**Pros:**

- Independent design, so can work independently with Affinity API
- Support both predicate and priority

**Cons:**

- Work for Story 2 without the presence of PodAntiAffinity
- More API changes
- More code changes, and some efforts of refactoring code to ensure Affinity
  related structure/logic can be reused gracefully

### Test Plan

To ensure this feature to be rolled out in high quality. Following tests are mandatory:

- **Unit Tests:** All core changes must be covered by unit tests.
- **Integration Tests / E2E Tests:** All user cases discussed in this KEP must
  be covered by either integration tests or e2e tests.
- **Benchmark Tests:** We can bear with slight performance overhead if users are
  using this feature, but it shouldn't impose penalty to users who are not using
  this feature. We will verify it by designing some benchmark tests.

### Graduation Criteria

Alpha:

- [x] This feature will be rolled out as an Alpha feature in v1.15.
- [x] API changes and feature gating.
- [x] Necessary defaulting, validation and generated code.
- [x] Predicate implementation.
- [x] Priority implementation.
- [x] Implementation of all scenarios discussed in this KEP.
- [x] Minimum viable test cases mentioned in [Test Plan](#test-plan) section.

Beta:

- [ ] This feature will be enabled by default as a Beta feature in v1.18.
- [ ] Replace of the term "Even Pods Spreading" with "Pod Topology Spread
  Constraints" in docs, KEP and source code. However, keep the feature gate name
  "EvenPodsSpread" as is.
- [ ] Migrate predicate implementation to preFilter / filter plugins.
- [ ] Migrate priority implementation to postFilter / score plugins.
- [ ] Calculate "preFilterState" if it's not pre-calculated in preFilter plugin.
  This is particularly for some extended usage such as Cluster Autoscaler.
- [ ] Add necessary end-to-end tests.

GA:

- [ ] Ensure feature documentation is clear and complete.

## Alternatives

- mixin new fields into `pod.spec.affinity` to act as a"sub feature" of Affinity

    ```go
    type TopologySpreadConstraint struct {
        // MaxSkew describes the degree of imbalance of pods spreading.
        // Default value is 1 and 0 is not allowed.
        MaxSkew int32
        // TopologyKey defines where pods are placed evenly
        TopologyKey string
    }

    type NodeAffinity struct {
        TopologySpreadConstraint *TopologySpreadConstraint
        ......
    }

    type PodAffinity struct {
        TopologySpreadConstraint *TopologySpreadConstraint
        ......
    }

    type PodAntiAffinity struct {
        TopologySpreadConstraint *TopologySpreadConstraint
        ......
    }
    ```

    - Pros:
        - Less API changes
        - Less code changes (code can be built on existing InterPodPredicate, as
          well as the internal data structures)
    - Cons:
        - The support on NodeAffinity is vague
        - Current API design only supports predicate

## Impact to Other Features

The motivation of this KEP is to resolve limitations of existing features, but
it won't replace them.

Comparing to this feature, PodAffinity has the most expressive APIs such like
multiple podAffinityTerms and multiple topologyKeys, hence still fits for the
complex scenarios; PodAntiAffinity still fits for the scenario which needs to
place up to one pod to one topology domain.

However there are some notices worth mentioning for efficient cooperation with
existing features.

- NodeAffinity/NodeSelector

As aforementioned, it's a reasonable assumption that evenness should be applied
among the filtered nodes specified by NodeAffinity/NodeSelector. So be aware of
implicit assumption.

- PodAffinity

PodAffinity can work seamlessly with this feature. But a tip here is that if
your requirement on PodAffinity only applies to one topology, and cares about
evenness, you can simply put the podAffinityTerm in the manner of `selector` and
`topologyKey` of `TopologySpreadConstraint`. This can achieve the same
scheduling goal efficiently.

- PodAntiAffinity

(not specific to this KEP, but worth mentioning here)

Currently PodAntiAffinity supports arbitrary topology domain, but sadly this
causes a slow down in scheduling (see [Rethink pod
affinity/anti-affinity](https://github.com/kubernetes/kubernetes/issues/72479)).
We're evaluating solutions such as limit topology domain to node, or internally
implement a fast/slow path handling that. If this KEP gets implemented, we can
simply achieve the semantics of "PodAntiAffinity in zones" via a combination of
"Even pods spreading in zones" plus "PodAntiAffinity in nodes" which could be an
extra benefit of this KEP.

## Implementation History

- 2019-02-21: Initial KEP sent out for review.
- 2019-04-16: Initial KEP approved.
- 2019-05-01: First [KEP implementation PR](https://github.com/kubernetes/kubernetes/pull/77327) sent out for review.
- 2020-01-21: KEP updated to meet the criteria of promoting to beta.
  - NOTE: The term "Even Pods Spreading" is replaced with "Pod Topology Spread",
    to be consistent with the [official doc](https://kubernetes.io/docs/concepts/workloads/pods/pod-topology-spread-constraints/), but
    the featuregate name "EvenPodsSpread" remains unchanged.
