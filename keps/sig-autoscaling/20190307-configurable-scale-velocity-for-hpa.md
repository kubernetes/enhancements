---
title: Configurable scale up/down velocity for HPA
authors:
  - "@gliush"
owning-sig: sig-autoscaling
participating-sigs:
reviewers:
  - TBD
approvers:
  - TBD
editor: TBD
creation-date: 2019-03-07
last-updated: 2019-03-07
status: provisional
superseded-by: TBD
---

# Configurable scale up/down velocity for HPA

## Table of Contents

- [Configurable scale up/down velocity for HPA](#configurable-scale-updown-velocity-for-hpa)
  - [Table of Contents](#table-of-contents)
  - [Summary](#summary)
  - [Motivation](#motivation)
    - [Goals](#goals)
    - [Non-Goals](#non-goals)
  - [Proposal](#proposal)
    - [User Stories](#user-stories)
      - [Story 1: Scale up as fast as possible](#story-1-scale-up-as-fast-as-possible)
      - [Story 2: Scale up as fast as possible, very gradual scale down](#story-2-scale-up-as-fast-as-possible-very-gradual-scale-down)
      - [Story 3: Scale up very gradually, usual scale down process](#story-3-scale-up-very-gradually-usual-scale-down-process)
      - [Story 4: Scale up as usual, do not scale down](#story-4-scale-up-as-usual-do-not-scale-down)
    - [Implementation Details/Notes/Constraints](#implementation-detailsnotesconstraints)
      - [Algorithm pseudocode](#algorithm-pseudocode)
      - [Default values](#default-values)
      - [Motivation for “pick the largest constraint” concept](#motivation-for-pick-the-largest-constraint-concept)
      - [Stabilization Window](#stabilization-window)
      - [API Changes](#api-changes)
      - [HPA Controller State changes](#hpa-controller-state-changes)

## Summary

[Horizontal Pod Autoscaler][] (HPA) automatically scales the number of pods in a replication controller, deployment or replica set based on observed CPU utilization (or, with custom metrics support, on some other application-provided metrics). This proposal adds scale velocity configuration parameters to the HPA.

[Horizontal Pod Autoscaler]: https://kubernetes.io/docs/tasks/run-application/horizontal-pod-autoscale/

## Motivation

Different applications may have different business values, different logic and may require different scaling behaviours.
I can name at least three types of applications:

- Applications that handles business-critical web traffic. They should scale up as fast as possible (false positive signals to scale up are ok), and scale down very slowly (waiting for another traffic spike).
- Applications that process very important data. They should scale up as fast as possible (to reduce the data processing time), and scale down as soon as possible (to reduce the costs). False positives signals to scale up/down are ok.
- Applications that process other data/web traffic. It is not that important, and may scale up and down in a usual way to minimize jitter.

At the moment, there’s only one cluster-level configuration parameter that influence how fast the cluster is scaled down:

- [--horizontal-pod-autoscaler-downscale-stabilization-window][]   (default to 5 min)

And a couple of hard-coded constants that specify how fast the cluster can scale up:

- [scaleUpLimitFactor][] = 2.0
- [scaleUpLimitMinimum][] = 4.0

As a result users can't influence the scale velocity and that is a problem for a lot of people. There're several open issues in tracker about that:

- [#39090][]
- [#65097][]
- [#69428][]

[--horizontal-pod-autoscaler-downscale-stabilization-window]: https://kubernetes.io/docs/tasks/run-application/horizontal-pod-autoscale/#algorithm-details
[scaleUpLimitFactor]: https://github.com/kubernetes/kubernetes/blob/v1.13.4/pkg/controller/podautoscaler/horizontal.go#L55
[scaleUpLimitMinimum]: https://github.com/kubernetes/kubernetes/blob/v1.13.4/pkg/controller/podautoscaler/horizontal.go#L56
[#39090]: https://github.com/kubernetes/kubernetes/issues/39090
[#65097]: https://github.com/kubernetes/kubernetes/issues/65097
[#69428]: https://github.com/kubernetes/kubernetes/issues/69428

### Goals

- Allow the user to be able to manage the scale velocity

### Non-Goals

TBA

## Proposal

We need to introduce an algorithm-agnostic HPA object configuration that will configure each particular HPA scaling behaviour.
For both direction (scale up and scale down) there should be two parameters:

- Parameter to specify the relative speed, in percentages:
  - `ScaleUpPercent`
    i.e. if ScaleUpPercent = 150 , then we can add 150% more pods (10 -> 25 pods)
  - `ScaleDownPercent`
    i.e. if ScaleDownPercent = 60 , then we can remove 60% of pods (10 -> 4)
- Parameter to specify the absolute speed, in number of pods:
  - `ScaleUpPods`
    i.e. if ScaleUpPods = 5 , then we can add 5 more pods (10 -> 15)
  - `ScaleDownPods`
    i.e. if ScaleDownPods = 7 , then we can remove 7 pods (10 -> 3)

All the parameters are per-minute and allow fraction values by using Quantity type.

A user will specify the parameters for the HPA, thus controlling the HPA logic.

If the user specify two parameters, two constraints are calculated, and the largest is used (see the [Motivation for pick the largest constraint][] section below).

If the user doesn’t want to use some particular parameter (= user doesn’t want to use this particular constraint), the parameter should be set to -1 (or any negative number).

[Motivation for pick the largest constraint]: #motivation-for-pick-the-largest-constraint-concept

### User Stories

#### Story 1: Scale up as fast as possible

This mode is important when you want to quickly respond to a traffic increase.

Create an HPA with the following parameters:

- `ScaleUpPercent = 900`    (i.e. increase number of pods 10 times per minute is ok).

All other parameters are not specified (default values are used)

If the application is started with 1 pod, it will scale up with the following number of pods:

    1 -> 10 -> 100 -> 1000

So, it can reach the `maxReplicas` very fast.

Scale down will be done a usual way (check stabilization window in the [Stabilization Window][] section below and in the [Algorithm details][] in the official HPA documentation)

[Stabilization Window]: #stabilization-window
[Algorithm details]: https://kubernetes.io/docs/tasks/run-application/horizontal-pod-autoscale/#algorithm-details

#### Story 2: Scale up as fast as possible, very gradual scale down

This mode is important when you don’t want to risk scaling down very quickly.

Create an HPA with the following parameters:

- `ScaleUpPercent = 900` (i.e. increase number of pods 10 times per minute is ok).
- `ScaleDownPods = 100m` (=0.1 i.e. scale down one pod every 10 min)

All other parameters are not specified (default values are used)

If the application is started with 1 pod, it will scale up with the following number of pods:

    1 -> 10 -> 100 -> 1000

So, it can reach the `maxReplicas` very fast.

Scaling down will be by one pod each HPA controller cycle:

    1000 -> 1000 -> 1000 -> … (7 more times) 999

#### Story 3: Scale up very gradually, usual scale down process

This mode is important when you want to increase capacity, but you want it to be very pessimistic.

Create an HPA with the following parameters:

- `ScaleUpPods = 1`    (i.e. increase only by one pod)

All other parameters are not specified (default values are used)

If the application is started with 1 pod, it will scale up very gradually:

    1 -> 2 -> 3 -> 4

Scale down will be done a usual way (check stabilization window in [Algorithm details][])

[Algorithm details]: https://kubernetes.io/docs/tasks/run-application/horizontal-pod-autoscale/#algorithm-details

#### Story 4: Scale up as usual, do not scale down

This mode is important when you don’t want to risk scaling down at all.

Create an HPA with the following parameters:

- `ScaleDownPercent = 0`
- `ScaleDownPods = 0`

i.e. set both constraints to 0, so that the HPA controller would never scale the cluster down

All other parameters are not specified (default values are used)

The cluster will scale up as usual (default values), but will never scale down.

### Implementation Details/Notes/Constraints

#### Algorithm pseudocode

The algorithm to find the number of pods will look like this:

```golang
desiredReplicas = 0
calculatedReplicas = AnyAlgorithmInHPAController(...)
if calculatedReplicas > curReplicas:
  constraint1 = CurReplicas * (1 + ScaleUpPercent/100)
  constraint2 = CurReplicas + ScaleUpPods
  scaleUpLimit = max(constraint1, constraint2)
  desiredReplicas = min(scaleUpLimit, calculatedReplicas)
else if calculatedReplicas < CurReplicas:
  constraint1 = curReplicas * (1 - ScaleDownPercent/100)
  constraint2 = CurReplicas - ScaleDownPods
  scaleDownLimit = min(constraint1, constraint2)
  desiredReplicas = max(scaleDownLimit, calculatedReplicas)
```

I.e. from the two provided limit parameters, chosen the most “soft” one (that limit less).
And that limit parameter limits the number of pods for any HPA algorithm.

If you don’t want to scale, you should set both parameters to zero for the appropriate direction (Up/Down).

If only one parameter is given and the other is set to -1 (not defined), the case is obvious: don’t let number of replicas goes beyond the only threshold.

If both parameters are set to -1 (not defined), we assume that no constraints are defined and the calculated number of replicas should be applied immediately.

#### Default values

For smooth transition it makes sense to set the following default values:
- `ScaleUpPercent = 100`
  i.e. allow to twice the number of pods per one HPA controller cycle
- `ScaleDownPercent = -1` (is not defined)
  i.e if the user doesn’t specify it, this parameter is not used
- `ScaleUpPods = 4`
  i.e. allow adding up to 4 pods per one HPA controller cycle
- `ScaleDownPods = 2`
  i.e. allow removing up to 2 pods per one HPA controller cycle

#### Motivation for “pick the largest constraint” concept

Take a look at the example:

- `curReplicas = 10`
- `calculatedReplicas = 20`

The user specifies only one HPA parameter `ScaleUpPods = 5` and expects that number of replicas to be set to 15 during the next HPA controller loop.
But the algorithm picks the largest change instead:

    Constraint1 = 10 * 2 = 20 (as ScaleUpPercent = 100 by default)
    Constraint2 = 10 + 5 = 15 (as ScaleUpPods = 5, set by the user)
    scaleUpLimit = max(20, 15) = 20
    desiredReplicas = 20

The user might expect that the autoscaler would use the smallest constraint (15), not the largest one (20). This is not intuitive, but it does make sense if considered thoroughly.

The main idea of the HPA is to autoscale because of a load increase to avoid request failures. It should work on both small cluster and large ones. For small clusters, the absolute number constraint works better (ScaleUpPods), for large clusters the percentage works better (ScaleUpPercentage).

Example: If the current cluster size is `1` and calculated cluster size for this particular load is `20`, than we should reach it ASAP.

For default values (ScaleUpPercent = 100, ScaleUpPods = 4) and “pick the largest constraint” concept, we’ll increase 1 -> 20 in 3 steps

    1 -> 5 -> 10 -> 20

The first step will use the “ScaleUpPods” limitation, next steps will use “ScaleUpPercent” limitation.

In case of more intuitive “pick the smallest limit” concept, we’ll increase the cluster in 6 steps:

    1 -> 2 -> 4 -> 8 -> 12 -> 16 -> 20

Given that each steps takes [90 sec in worst case], we’ll respond to the load increase in `(6-3)*90 sec = ~ 5 min`.

That’s too much, we should respond faster than that.

[90 sec in worst case]: https://dzone.com/articles/kubernetes-autoscaling-101-cluster-autoscaler-hori-1

#### Stabilization Window

Stabilization window ([PR][], [RFC][]) is used to gather “scale-down-recommendations” during some time (default is 5min),
and new number of replicas is set to the maximum of all recommendations.
It will be applied after all the “scaleDown parameters” described above.
We may want to control stabilization window size in addition to configuring the scale velocity,
but at the moment, the “stabilization window feature” is considered to be an internal feature and is not exposed via any configuration.

I’d suggest allowing to change the window size or turn it off if needed, but it should be discussed in a separate RFC.

[PR]: https://github.com/kubernetes/kubernetes/pull/68122
[RFC]: https://docs.google.com/document/d/1IdG3sqgCEaRV3urPLA29IDudCufD89RYCohfBPNeWIM/edit#heading=h.3tdw2jxiu42f


#### API Changes

The following API changes are needed:

We should add `Scale{Up,Down}{Percent,Pods}` fields into the HPA spec

The resulting solution will look like this:

```golang
type HorizontalPodAutoscalerScaleConstraints struct {
    ScaleUpPercent Quantity
    ScaleDownPercent Quantity
    ScaleUpPods Quantity
    ScaleDownPods Quantity
}

type HorizontalPodAutoscalerSpec struct {
    ScaleTargetRef CrossVersionObjectReference
    MinReplicas *int32
    MaxReplicas int32
    Metrics []MetricSpec
    ScaleConstraints HorizontalPodAutoscalerScaleConstraints
}
```

#### HPA Controller State changes

To store the information about last scale action the HPA need an additional fields (similar to the list of “recommendations”)

```golang
scaleEvents map[string][]timestampedScaleEvent
```

Where

```golang
type timestampedScaleEvent struct {
    replicaChange int32
    timestamp      time.Time
}
```

It will store last scale events, and it will be used to make decisions about next scale events.

Say, if 30 seconds ago the number of replicas was increased by one, and we forbid to scale up for more than 1 pod per minute,
then during the next 30 seconds we won’t scale up again.

If the HPA is restarted, the state information is lost, so it might scale the cluster instantly after the restart.
Though, I don’t think this is a large problem, as:

- It shouldn’t happen often or you have some cluster problem
- It requires quite a lot of time to start an HPA pod, for HPA pod to become live and ready, to get and process metrics.
- If you have a large discrepancy between what is a desired number of replicas according to metrics and what is your current number of replicas and you DON’T want to scale - probably, you shouldn’t want to use the HPA. As the HPA goal is the opposite.

As the added parameters have default values, we don’t need to update the API version, and may stay on the same `pkg/apis/autoscaling/v2beta2`.
