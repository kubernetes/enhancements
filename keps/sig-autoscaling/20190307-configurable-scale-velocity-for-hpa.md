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

<!-- toc -->

- [Configurable scale up/down velocity for HPA](#configurable-scale-updown-velocity-for-hpa)
  - [Table of Contents](#table-of-contents)
  - [Summary](#summary)
  - [Motivation](#motivation)
    - [Goals](#goals)
    - [Non-Goals](#non-goals)
  - [Proposal](#proposal)
    - [User Stories](#user-stories)
      - [Story 1: Scale Up As Fast As Possible](#story-1-scale-up-as-fast-as-possible)
      - [Story 2: Scale Up As Fast As Possible, Scale Down Very Gradually](#story-2-scale-up-as-fast-as-possible-scale-down-very-gradually)
      - [Story 3: Scale Up Very Gradually, Usual Scale Down Process](#story-3-scale-up-very-gradually-usual-scale-down-process)
      - [Story 4: Scale Up As Usual, Do Not Scale Down](#story-4-scale-up-as-usual-do-not-scale-down)
      - [Story 5: Delay Before Scaling Down](#story-5-delay-before-scaling-down)
    - [Implementation Details/Notes/Constraints](#implementation-detailsnotesconstraints)
      - [Algorithm Pseudocode](#algorithm-pseudocode)
      - [Introducing `delay` Option](#introducing-delay-option)
      - [Default Values](#default-values)
      - [The Motivation To “Pick The Largest Constraint” Concept](#the-motivation-to-pick-the-largest-constraint-concept)
      - [Stabilization Window](#stabilization-window)
      - [API Changes](#api-changes)
      - [HPA Controller State Changes](#hpa-controller-state-changes)
      - [Command Line Options Changes](#command-line-options-changes)

<!-- /toc -->

## Summary

[Horizontal Pod Autoscaler][] (HPA) automatically scales the number of pods in a replication controller, deployment or replica set based on observed CPU utilization (or, with custom metrics support, on some other application-provided metrics). This proposal adds scale velocity configuration parameters to the HPA.

[Horizontal Pod Autoscaler]: https://kubernetes.io/docs/tasks/run-application/horizontal-pod-autoscale/

## Motivation

Different applications may have different business values, different logic and may require different scaling behaviors.
I can name at least three types of applications:

- Applications that handle business-critical web traffic. They should scale up as fast as possible (false positive signals to scale up are ok), and scale down very slowly (waiting for another traffic spike).
- Applications that process critical data. They should scale up as fast as possible (to reduce the data processing time), and scale down as soon as possible (to reduce the costs). False positives signals to scale up/down are ok.
- Applications that process regular data/web traffic. It is not that important and may scale up and down in a usual way to minimize jitter.

At the moment, there’s only one cluster-level configuration parameter that influence how fast the cluster is scaled down:

- [--horizontal-pod-autoscaler-downscale-stabilization-window][]   (default to 5 min)

And a couple of hard-coded constants that specify how fast the cluster can scale up:

- [scaleUpLimitFactor][] = 2.0
- [scaleUpLimitMinimum][] = 4.0

As a result, users can't influence scale velocity, and that is a problem for many people. There're several open issues in tracker about that:

- [#39090][]
- [#65097][]
- [#69428][]

[--horizontal-pod-autoscaler-downscale-stabilization-window]: https://v1-14.docs.kubernetes.io/docs/tasks/run-application/horizontal-pod-autoscale/#algorithm-details
[scaleUpLimitFactor]: https://github.com/kubernetes/kubernetes/blob/release-1.14/pkg/controller/podautoscaler/horizontal.go#L55
[scaleUpLimitMinimum]: https://github.com/kubernetes/kubernetes/blob/release-1.14/pkg/controller/podautoscaler/horizontal.go#L56
[#39090]: https://github.com/kubernetes/kubernetes/issues/39090
[#65097]: https://github.com/kubernetes/kubernetes/issues/65097
[#69428]: https://github.com/kubernetes/kubernetes/issues/69428

### Goals

- Allow the user to be able to manage the scale velocity

### Non-Goals

TBA

## Proposal

We need to introduce an algorithm-agnostic HPA object configuration that will configure each particular HPA scaling behavior.
For both direction (scale up and scale down) there should be a `Constraint` field with the following parameters:

- `periodSeconds` - a parameter to specify the time period for the constraint, in seconds
- `percent` - a parameter to specify the relative speed, in percentages:
    i.e., if ScaleUpPercent = 150 , then we can add 150% more pods (10 -> 25 pods)
- `pods` - a parameter to specify the absolute speed, in the number of pods:
    i.e., if ScaleUpPods = 5 , then we can add 5 more pods (10 -> 15)
- `delay` - a parameter to specify the window over which the max (or min) recommendation is used. It behaves the same as the [Stabilization Window][].

A user will specify the parameters for the HPA, thus controlling the HPA logic.

If the user specifies two parameters, two constraints are calculated, and the largest is used (see the [The motivation to pick the largest constraint][] section below).

If the user doesn't specify any parameter, the default value for that parameter is used (see the [Default Values][] section below).

If the user set both `percent` and `pods` to `0`, it means that the corresponding resource (e.g. `deploy`, `rs`) should not be scaled in that direction.

[The motivation to pick the largest constraint]: #the-motivation-to-pick-the-largest-constraint-concept
[Default Values]: #default-values
[Stabilization Window]: #stabilization-window

### User Stories

#### Story 1: Scale Up As Fast As Possible

This mode is essential when you want to respond to a traffic increase quickly.

Create an HPA with the following configuration:

- `constraints`:
  - `scaleUp`:
    - `percent = 900`    (i.e., to increase the number of pods 10 times per minute is ok).

All other parameters are not specified (default values are used)

If the application is started with 1 pod, it will scale up with the following number of pods:

    1 -> 10 -> 100 -> 1000

So, it can reach `maxReplicas` very fast.

Scale down will be done a usual way (check stabilization window in the [Stabilization Window][] section below and the [Algorithm details][] in the official HPA documentation)

[Stabilization Window]: #stabilization-window
[Algorithm details]: https://v1-14.docs.kubernetes.io/docs/tasks/run-application/horizontal-pod-autoscale/#algorithm-details

#### Story 2: Scale Up As Fast As Possible, Scale Down Very Gradually

This mode is essential when you don’t want to risk scaling down very quickly.

Create an HPA with the constraints:

- `constraints`:
  - `scaleUp`:
    - `percent = 900` (i.e. increase number of pods 10 times per minute is ok).
  - `scaleDown`:
    - `pods = 1`
    - `periodSeconds = 600` (i.e., scale down one pod every 10 min)

All other parameters are not specified (default values are used)

If the application is started with 1 pod, it will scale up with the following number of pods:

    1 -> 10 -> 100 -> 1000

So, it can reach `maxReplicas` very fast.

Scaling down will be by one pod each 10 min:

    1000 -> 1000 -> 1000 -> … (7 more min) -> 999

#### Story 3: Scale Up Very Gradually, Usual Scale Down Process

This mode is essential when you want to increase capacity, but you want it to be very pessimistic.

Create an HPA with the constraints:

- `constraints`:
  - `scaleUp`:
    - `pods = 1`    (i.e., increase only by one pod per minute)

All other parameters are not specified (default values are used)

If the application is started with 1 pod, it will scale up very gradually:

    1 -> 2 -> 3 -> 4

Scale down will be done a usual way (check stabilization window in [Algorithm details][])

[Algorithm details]: https://v1-14.docs.kubernetes.io/docs/tasks/run-application/horizontal-pod-autoscale/#algorithm-details

#### Story 4: Scale Up As Usual, Do Not Scale Down

This mode is essential when you don’t want to risk scaling down at all.

Create an HPA with the following constraints:

- `constraints`:
  - `scaleDown`:
    - `percent = 0`
    - `pods = 0`

i.e., set both constraints to 0, so that the HPA controller would never scale the cluster down

All other parameters are not specified (default values are used)

The cluster will scale up as usual (default values), but will never scale down.

#### Story 5: Delay Before Scaling Down

This mode is used when the user expects a lot of flapping
or doesn't want to turn off pods too early expecting some late load spikes.

Create an HPA with the following constraints:

- `constraints`:
  - `scaleDown`:
    - `pods = 5`
  - `delaySeconds = 600`

i.e., the algorithm will:

- gather recommendations for 600 seconds (by default: 300)
- pick the largest one
- turn off no more than 5 pods per minute

Example for `CurReplicas = 10` and HPA controller cycle once per a minute:

- First 9 minutes the algorithm will do nothing except gathering recommendations.
  Let's imagine that we have the following recommendations

    recommendations = [10, 9, 8, 9, 9, 8, 9, 8, 9]

- On the 10th minute, we'll add one more recommendation (let it me `8`):

    recommendations = [10, 9, 8, 9, 9, 8, 9, 8, 9, 8]

  Now the algorithm picks the largest one `10`. Hence it will not change number of replicas

- On the 11th minute, we'll add one more recommendation (let it be `7`) and removes the first one to keep the same amount of recommendations:

    recommendations = [9, 8, 9, 9, 8, 9, 8, 9, 8, 7]

  The algorithm picks the largest value `9` and change the number of replicas `10 -> 9`

### Implementation Details/Notes/Constraints

#### Algorithm Pseudocode

The algorithm to find the number of pods will look like this:

```golang
  for { // infinite cycle inside the HPA controller
    desiredReplicas = AnyAlgorithmInHPAController(...)
    if desiredReplicas > curReplicas:
      constraint1 = CurReplicas * (1 + ScaleUpPercent/100)
      constraint2 = CurReplicas + ScaleUpPods
      scaleUpLimit = max(constraint1, constraint2)
      limitedReplicas = min(scaleUpLimit, desiredReplicas)
      storeRecommendation(limitedReplicas, scaleUpRecommendations)
      recommendations = getLastRecommendations(scaleUpRecommendations, ScaleUpDelaySeconds) // get recommendations for the last ScaleUpDelaySeconds
      nextReplicas = min(recommendations)
    if desiredReplicas < curReplicas:
      constraint1 = curReplicas * (1 - ScaleDownPercent/100)
      constraint2 = CurReplicas - ScaleDownPods
      scaleDownLimit = max(constraint1, constraint2)
      limitedReplicas = max(scaleDownLimit, desiredReplicas)
      storeRecommendation(limitedReplicas, scaleDownRecommendations)
      recommendations = getLastRecommendations(scaleDownRecommendations, ScaleDownDelaySeconds) // get recommendations for the last ScaleDownDelaySeconds
      nextReplicas = max(recommendations)
    setReplicas(nextReplicas)
    sleep(ControllerSleepTime)
  }

```

I.e., from the two provided constraints the larger one is always used.

If you don’t want to scale, you should set both parameters to zero for the appropriate direction (Up/Down).

If only one parameter is given and the other is 0, then use the only defined constraint.

If no value is given, the default one is chosen, see the [Default Values][] section.

[Default Values]: #default-values

#### Introducing `delay` Option

Effectively, the `delay` option is a full copy of the current [Stabilization Window][] algorithm:

- While scaling up, we should pick the safest (smallest) "desiredReplicas" number during last `delaySeconds`.
- While scaling down, we should pick the safest (largest) "desiredReplicas" number during last `delaySeconds`.

Check the [Algorithm Pseudocode][] section if you need more details.

If delay is `0`, it means that no delay should be used. And we should instantly change the number of replicas.

If no delay is specified, the default value is used, see the [Default Values][] section.

The “Stabilization Window" as a result becomes an alias for the `constraints.scaleDown.delaySeconds`.

[Stabilization Window]: #stabilization-window
[Algorithm Pseudocode]: #algorithm-pseudocode
[Default Values]: #default-values

#### Default Values

For smooth transition it makes sense to set the following default values:

- `constraints.scaleUp.delaySeconds = 0`, the delay is not used, instantly scale up
- `constraints.scaleDown.delaySeconds = 300`, wait 5 min for the largest recommendation and then scale down to that value.
- `constraints.scaleUp.rate.periodSeconds = 60`, one minute period for scaleUp
- `constraints.scaleDown.rate.periodSeconds = 60`, one minute period for scaleDown
- `constraints.scaleUp.rate.percent = 100`, allow to twice the number of pods
- `constraints.scaleUp.rate.pods = 4`, i.e. allow adding up to 4 pods
- `constraints.scaleDown.rate.percent = nil`, allow to remove all the pods
- `constraints.scaleDown.rate.pods = nil`, allow to remove all the pods

Please note that:

`constraints.ScaleDown.delaySeconds` value is picked in the following order:

- from the HPA configuration, use that value
- from the command-line options. Check the [Command Line Option Changes][] section.
- from the hardcoded default value `300`.

For the `scaleDown` constraint both `pods` and `percent` default values are set to maximum possible values.
Because currently (as of k8s-1.14) the scale down is only limited by [Stabilization Window][].
In order to repeat the default behavior we set `constraints.scaleDown.delaySeconds` to 5min
(the default value for Stabilization window), and let it rule the number of pods.

We should differentiate "not given" value and `0`-value.
For `pods` and `percent`, `0`-value means that we shouldn't scale.
While "not given" value means that we should use the default.
Hence we should use pointers `*int32` ("nillable" type) instead of just `int32` for all the introduced values.

[Stabilization Window]: #stabilization-window

#### The Motivation To “Pick The Largest Constraint” Concept

Take a look at the example:

- `curReplicas = 10`
- `calculatedReplicas = 20`

The user specifies only one HPA parameter `constraints.scaleUp.pods = 5` and expects that number of replicas to be set to 15 during the next HPA controller loop.
But the algorithm picks the largest change instead:

    Constraint1 = 10 * 2 = 20 (as constraints.scaleUp.percent = 100 by default)
    Constraint2 = 10 + 5 = 15 (as constraints.scaleUp.pods = 5, set by the user)
    scaleUpLimit = max(20, 15) = 20
    desiredReplicas = 20

The user might expect that the autoscaler would use the smallest constraint (15), not the largest one (20). This behavior is not intuitive, but it does make sense if considered thoroughly.

The main idea of the HPA is to autoscale because of a load increase to avoid request failures. It should work on both small clusters and large ones. For small clusters, the absolute number constraint works better (ScaleUpPods), for large clusters the percentage works better (ScaleUpPercentage).

Example: If the current cluster size is `1` and calculated cluster size for this particular load is `20`, then we should reach it ASAP.

For default values (ScaleUpPercent = 100, ScaleUpPods = 4) and “pick the largest constraint” concept, we’ll increase 1 -> 20 in 3 steps

    1 -> 5 -> 10 -> 20

The first step will use “scaleUp.pods” constraint; the next steps will use “scaleUp.percent” one.

In case of more intuitive “pick the hardest limit” concept, we’ll increase the cluster in 6 steps:

    1 -> 2 -> 4 -> 8 -> 12 -> 16 -> 20

Given that each step takes [90 sec in worst case], we’ll respond to the load increase in `(6-3)*90 sec = ~ 5 min`.

That’s too much, we should respond faster than that.

[90 sec in worst case]: https://dzone.com/articles/kubernetes-autoscaling-101-cluster-autoscaler-hori-1

The scale down constraints are a method to prevent too rapid loss of capacity. Hence, it makes sense to pick the maximum of two constraints.

#### Stabilization Window

Currently stabilization window ([PR][], [RFC][], [Algorithm Details][]) is used to gather “scale-down-recommendations” during some time (default is 5min),
and a new number of replicas is set to the maximum of all recommendations.

It may be defined by command line option `--horizontal-pod-autoscaler-downscale-stabilization-window`.

[PR]: https://github.com/kubernetes/kubernetes/pull/68122
[RFC]: https://docs.google.com/document/d/1IdG3sqgCEaRV3urPLA29IDudCufD89RYCohfBPNeWIM/edit#heading=h.3tdw2jxiu42f
[Algorithm Details]: https://v1-14.docs.kubernetes.io/docs/tasks/run-application/horizontal-pod-autoscale/#algorithm-details

#### API Changes

The following API changes are needed:

We should add `Scale{Up,Down}{Percent,Pods}` fields into the HPA spec

The resulting solution will look like this:

```golang
type HPAScaleConstraintValue struct {
    Rate         *HPAScaleConstraintRateValue
    DelaySeconds *int32

type HPAScaleConstraintRateValue struct {
    Pods          *int32
    Percent       *int32
    PeriodSeconds *int32
}

type HPAScaleConstraints struct {
    ScaleUp   *HPAScaleConstraintValue
    ScaleDown *HPAScaleConstraintValue
}

type HorizontalPodAutoscalerSpec struct {
    ScaleTargetRef CrossVersionObjectReference
    MinReplicas    *int32
    MaxReplicas    int32
    Metrics        []MetricSpec
    Constraints    *HPAScaleConstraints
}
```

#### HPA Controller State Changes

To store not only scale down recommendations, we need to replace

```golang
    recommendations map[string][]timestampedRecommendation
```

with

```golang
    scaleUpRecommendations   map[string][]timestampedRecommendation
    scaleDownRecommendations map[string][]timestampedRecommendation
```

To store the information about last scale action, the HPA need an additional field (similar to the list of “recommendations”)

```golang
scaleEvents map[string][]timestampedScaleEvent
```

Where

```golang
type timestampedScaleEvent struct {
    replicaChange int32
    timestamp     time.Time
}
```

It will store last scale events, and it will be used to make decisions about next scale events.

Say, if 30 seconds ago the number of replicas was increased by one, and we forbid to scale up for more than 1 pod per minute,
then during the next 30 seconds, we won’t scale up again.

If the HPA is restarted, the state information is lost so that it might scale the cluster instantly after the restart.
Though, I don’t think this is a problem, as:

- It shouldn’t happen often, or you have some cluster problem
- It requires quite a lot of time to start an HPA pod, for HPA pod to become live and ready, to get and process metrics.
- If you have a large discrepancy between what is a desired number of replicas according to metrics and what is your current number of replicas and you DON’T want to scale - probably, you shouldn’t want to use the HPA. As the HPA goal is the opposite.

As the added parameters have default values, we don’t need to update the API version, and may stay on the same `pkg/apis/autoscaling/v2beta2`.

#### Command Line Options Changes

It should be noted that the
current [--horizontal-pod-autoscaler-downscale-stabilization-window][] option
defines the default value for the `constraints.scaleDown.delaySeconds`
As it becomes part of the HPA specification, the option is not needed anymore.
So we should make it obsolete.

Check the [Default Values][] section for more information about how to determine the delay (priorities of options).

[--horizontal-pod-autoscaler-downscale-stabilization-window]: https://v1-14.docs.kubernetes.io/docs/tasks/run-application/horizontal-pod-autoscale/#algorithm-details
[DefaultValues]: #default-values
