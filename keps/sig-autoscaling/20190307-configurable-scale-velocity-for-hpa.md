---
title: Configurable scale up/down velocity for HPA
authors:
  - "@gliush"
  - "@arjunrn"
owning-sig: sig-autoscaling
participating-sigs:
reviewers:
  - "@mwielgus"
  - "@josephburnett"
approvers:
  - "@mwielgus"
  - "@josephburnett"
editor: TBD
creation-date: 2019-03-07
last-updated: 2020-01-29
status: implemented
superseded-by:
---

# Configurable scale up/down velocity for HPA

## Table of Contents

<!-- toc -->
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
    - [Story 5: Stabilization before scaling down](#story-5-stabilization-before-scaling-down)
    - [Story 6: Avoid false positive signals for scaling up](#story-6-avoid-false-positive-signals-for-scaling-up)
  - [Implementation Details/Notes/Constraints](#implementation-detailsnotesconstraints)
    - [Algorithm Pseudocode](#algorithm-pseudocode)
    - [Introducing <code>stabilizationWindowSeconds</code> Option](#introducing--option)
    - [Default Values](#default-values)
    - [Stabilization Window](#stabilization-window)
    - [API Changes](#api-changes)
    - [HPA Controller State Changes](#hpa-controller-state-changes)
    - [Command Line Options Changes](#command-line-options-changes)
- [Design Details](#design-details)
  - [Test Plan](#test-plan)
  - [Graduation Criteria](#graduation-criteria)
<!-- /toc -->

## Summary

[Horizontal Pod Autoscaler][] (HPA) automatically scales the number of pods in any resource which supports the `scale` subresource based on observed CPU utilization
(or, with custom metrics support, on some other application-provided metrics). This proposal adds scale velocity configuration parameters to the HPA to control the
rate of scaling in both directions.

[Horizontal Pod Autoscaler]: https://kubernetes.io/docs/tasks/run-application/horizontal-pod-autoscale/

## Motivation

Different applications may have different business values, different logic and may require different scaling behaviors.
I can name at least three types of applications:

- Applications that handle business-critical web traffic. They should scale up as fast as possible (false positive signals to scale up are ok), and scale down very slowly (waiting for another traffic spike).
- Applications that process critical data. They should scale up as fast as possible (to reduce the data processing time), and scale down as soon as possible (to reduce the costs). False positives signals to scale up/down are ok.
- Applications that process regular data/web traffic. These are not very critical and may scale up and down in a usual way to minimize jitter.

At the moment, there’s only one cluster-level configuration parameter that influence how fast the cluster is scaled down:

- [--horizontal-pod-autoscaler-downscale-stabilization-window][]   (default to 5 min)

And a couple of hard-coded constants that specify how fast the target can scale up:

- [scaleUpLimitFactor][] = 2.0
- [scaleUpLimitMinimum][] = 4.0

As a result, users cannot influence scale velocity, and that is a problem for many applications. There are several open issues in the tracker about this issue:

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

- Persist the scaling events so that the HPA behavior is consistent even when the controller is restarted.
- Add `tolerance` parameter to the new `behavior` section for both `scaleUp` and `scaleDown`

## Proposal

We need to introduce an algorithm-agnostic HPA object configuration that will allow configuration of individual HPAs.
To customize the scaling behavior we should add a `behavior` object with the following fields:

- `scaleUp` specifies the rules which are used to control scaling behavior while scaling up.
  - `stabilizationWindowSeconds` - this value indicates the amount of time the HPA controller should consider
      previous recommendations to prevent flapping of the number of replicas.
  - `selectPolicy` can be `min` or `max` and specifies which value from the policies should be selected. The `max` value is used by default.
  - `policies` a list of policies which regulate the amount of scaling. Each item has the following fields
    - `type` can have the value `pods` or `percent` which indicates the allowed changed in terms of absolute number of pods or percentage of current replicas.
    - `periodSeconds` the amount of time in seconds for which the rule should hold true.
    - `value` the value for the policy
- `scaleDown` similar to the `scaleUp` but specifies the rules for scaling down.

A user will specify the parameters for the HPA, thus controlling the HPA logic.

The `selectPolicy` field indicates which policy should be applied. By default the `max` policy is chosen or in other words while scaling up the highest
possible number of replicas is used and while scaling down the lowest possible number of replicas is chosen. 

If the user does not specify `policies` for either `scaleUp` or `scaleDown` then default value for that policy is used 
(see the [Default Values] [] section below). Setting the `value` to `0` for `scaleUp` or `scaleDown` disables scaling in that direction.

[Default Values]: #default-values
[Stabilization Window]: #stabilization-window

### User Stories

#### Story 1: Scale Up As Fast As Possible

This mode is essential when you want to respond to a traffic increase quickly.

Create an HPA with the following configuration:

```yaml
behavior:
  scaleUp:
    policies:
    - type: percent
      value: 900%
```

The `900%` implies that 9 times the current number of pods can be added, effectively making the number
of replicas 10 times the current size. All other parameters are not specified (default values are used)

If the application is started with 1 pod, it will scale up with the following number of pods:

    1 -> 10 -> 100 -> 1000

This way the target can reach `maxReplicas` very quickly.

Scale down will be done in the usual way (check stabilization window in the [Stabilization Window][] section below and the [Algorithm details][] in the official HPA documentation)

[Stabilization Window]: #stabilization-window
[Algorithm details]: https://v1-14.docs.kubernetes.io/docs/tasks/run-application/horizontal-pod-autoscale/#algorithm-details

#### Story 2: Scale Up As Fast As Possible, Scale Down Very Gradually

This mode is essential when you do not want to risk scaling down very quickly.

Create an HPA with the following behavior:

```yaml
behavior:
  scaleUp:
    policies:
    - type: percent
      value: 900%
  scaleDown:
    policies:
    - type: pods
      value: 1
      periodSeconds: 600 # (i.e., scale down one pod every 10 min)
```

This `behavior` has the same scale-up pattern as the previous example. However the `behavior` for scaling down is also specified.
The `scaleUp` behavior will be fast as explained in the previous example. However the target will scale down by only one pod every 10 minutes.

    1000 -> 1000 -> 1000 -> … (7 more min) -> 999

#### Story 3: Scale Up Very Gradually, Usual Scale Down Process

This mode is essential when you want to increase capacity, but you want it to be very pessimistic.
Create an HPA with the following behavior:

```yaml
behavior:
  scaleUp:
    policies:
    - type: pods
      value: 1
```

If the application is started with 1 pod, it will scale up very gradually:

    1 -> 2 -> 3 -> 4

Scale down will be done a usual way (check stabilization window in [Algorithm details][])

[Algorithm details]: https://v1-14.docs.kubernetes.io/docs/tasks/run-application/horizontal-pod-autoscale/#algorithm-details

#### Story 4: Scale Up As Usual, Do Not Scale Down

This mode is essential when scale down should not happen or should be controlled by a separate process.

Create an HPA with the following constraints:

```yaml
behavior:
  scaleDown:
    policies:
    - type: pods
      value: 0
```

The cluster will scale up as usual (default values), but will never scale down.

#### Story 5: Stabilization before scaling down

This mode is used when the user expects a lot of flapping or does not want to scale down pods too early expecting some late load spikes.

Create an HPA with the following behavior:

```yaml
behavior:
  scaleDown:
    stabilizationWindowSeconds: 600
    policies:
    - type: pods
      value: 5
```

i.e., the algorithm will:

- gather recommendations for 600 seconds _(default: 300 seconds)_
- pick the largest one
- scale down no more than 5 pods per minute

Example for `CurReplicas = 10` and HPA controller cycle once per a minute:

- First 9 minutes the algorithm will do nothing except gathering recommendations.
  Let's imagine that we have the following recommendations

    recommendations = [10, 9, 8, 9, 9, 8, 9, 8, 9]

- On the 10th minute, we'll add one more recommendation (let it me `8`):

    recommendations = [10, 9, 8, 9, 9, 8, 9, 8, 9, 8]

  Now the algorithm picks the largest one `10`. Hence it will not change number of replicas

- On the 11th minute, we'll add one more recommendation (let it be `7`) and removes the first one to keep the same amount of recommendations:

    recommendations = [9, 8, 9, 9, 8, 9, 8, 9, 8, 7]

  The algorithm picks the largest value `9` and changes the number of replicas `10 -> 9`

#### Story 6: Avoid false positive signals for scaling up

This mode is useful in Data Processing pipelines when the number of replicas depends on the number of events in the queue.
The users want to scale up quickly if they have a high number of events in the queue. 
However, they do not want to react to false positive signals, i.e. to short spikes of events.

Create an HPA with the following behavior:

```yaml
behavior:
  scaleUp:
    stabilizationWindowSeconds: 300
    policies:
    - type: pods
      value: 20
```

i.e., the algorithm will:

- gather recommendations for 300 seconds _(default: 0 seconds)_
- pick the smallest one
- scale up no more than 20 pods per minute

Example for `CurReplicas = 2` and HPA controller cycle once per a minute:

- First 5 minutes the algorithm will do nothing except gathering recommendations.
  Let's imagine that we have the following recommendations

    recommendations = [2, 3, 19, 10, 3]

- On the 6th minute, we'll add one more recommendation (let it me `4`):

    recommendations = [2, 3, 19, 10, 3, 4]

  Now the algorithm picks the smallest one `2`. Hence it will not change number of replicas

- On the 7th minute, we'll add one more recommendation (let it be `7`) and removes the first one to keep the same amount of recommendations:

    recommendations = [7, 3, 19, 10, 3, 4]

  The algorithm picks the smallest value `3` and changes the number of replicas `2 -> 3`

### Implementation Details/Notes/Constraints

To minimize the impact of new changes on existing code the HPA controller will be modified in a such
a way that the scaling calculations will have separate code paths for existing HPA and HPAs with
the `behavior` field set. The new code path will be as shown below.

#### Algorithm Pseudocode

The algorithm to find the number of pods will look like this:

```golang
  for { // infinite cycle inside the HPA controller
    desiredReplicas = AnyAlgorithmInHPAController(...)
    if desiredReplicas > curReplicas {
      replicas = []int{}
      for _, policy := range behavior.ScaleUp.Policies {
        if policy.type == "pods" {
          replicas = append(replicas, CurReplicas + policy.Value)
        } else if policy.type == "percent" {
          replicas = append(CurReplicas * (1 + policy.Value/100))
        }
      }
      if behavior.ScaleUp.selectPolicy == "max" {
        scaleUpLimit = max(replicas)
      } else {
        scaleUpLimit = min(replicas)
      }
      limitedReplicas = min(max, desiredReplicas)
    }
    if desiredReplicas < curReplicas {
      for _, policy := range behaviro.scaleDown.Policies {
        replicas = []int{}
        if policy.type == "pods" {
          replicas = append(replicas, CurReplicas - policy.Value)
        } else if policy.type == "percent" {
          replicas = append(replicas, CurReplicas * (1 - policy.Value /100))
        }
        if behavior.ScaleDown.SelectPolicy == "max" {
          scaleDownLimit = min(replicas)
        } else {
          scaleDownLimit = max(replicas)
        }
        limitedReplicas = max(min, desiredReplicas)
      }
    }
    storeRecommend(limitedReplicas, scaleRecommendations)
    nextReplicas := applyRecommendationIfNeeded(scaleRecommendations)
    setReplicas(nextReplicas)
    sleep(ControllerSleepTime)
  }

```

If no scaling policy is specified then the default policy is chosen(see the [Default Values][] section).

[Default Values]: #default-values

#### Introducing `stabilizationWindowSeconds` Option

Effectively the `stabilizationWindowSeconds` option is a full copy of the current [Stabilization Window][] algorithm extended to cover scale up:

- While scaling down, we should pick the safest (largest) "desiredReplicas" number during last `stabilizationWindowSeconds`.
- While scaling up, we should pick the safest (smallest) "desiredReplicas" number during last `stabilizationWindowSeconds`.

Check the [Algorithm Pseudocode][] section if you need more details.

If the window is `0`, it means that no delay should be used. And we should instantly change the number of replicas.

If no value is specified, the default value is used, see the [Default Values][] section.

The __“Stabilization Window"__ as a result becomes an alias for the `behavior.scaleDown.stabilizationWindowSeconds`.

[Stabilization Window]: #stabilization-window
[Algorithm Pseudocode]: #algorithm-pseudocode
[Default Values]: #default-values

#### Default Values

For smooth transition it makes sense to set the following default values:

- `behavior.scaleDown.stabilizationWindowSeconds = 300`, wait 5 min for the largest recommendation and then scale down to that value.
- `behavior.scaleUp.stabilizationWindowSeconds = 0`, do not gather recommendations, instantly scale up to the calculated number of replicas
- `behavior.scaleUp.policies` has the following policies
   - Percentage policy
      - `policy = percent`
      - `periodSeconds = 60`, one minute period for scaleUp
      - `value = 100` which means the number of replicas can be doubled every minute.
   - Pod policy
      - `policy = pods`
      - `periodSeconds = 60`, one minute period for scaleUp
      - `value = 4` which means the 4 replicas can be added every minute.
- `behavior.scaleDown.policies` has the following policies
  - Percentage Policy
    - `policy = percent`
    - `periodSeconds = 60` one minute period for scaleDown
    - `value = 100`  which means all the replicas can be scaled down in one minute.

Please note that:

`behavior.scaleDown.stabilizationWindowSeconds` value is picked in the following order:

- from the HPA configuration if specified
- from the command-line options for the controller. Check the [Command Line Option Changes][] section.
- from the hardcoded default value `300`.

The `scaleDown` behavior has a single `percent` policy with a value of `100` because
the current scale down behavior is only limited by [Stabilization Window][] which means after
the stabilization window has passed the target can be scaled down to the minimum specified replicas.
In order to replicate the default behavior we set `behavior.scaleDown.stabilizationWindowSeconds` to 300
(the default value for Stabilization window), and let it determine the number of pods.

[Stabilization Window]: #stabilization-window

#### Stabilization Window

Currently the stabilization window ([PR][], [RFC][], [Algorithm Details][]) is used to gather __scale-down-recommendations__
during a fixed interval (default is 5min), and a new number of replicas is set to the maximum of all recommendations
in that interval. This is done to prevent a constant fluctuation in the number of pods if the traffic/load fluctuates
over a short period.

It may be specified by command line option `--horizontal-pod-autoscaler-downscale-stabilization-window`.

[PR]: https://github.com/kubernetes/kubernetes/pull/68122
[RFC]: https://docs.google.com/document/d/1IdG3sqgCEaRV3urPLA29IDudCufD89RYCohfBPNeWIM/edit#heading=h.3tdw2jxiu42f
[Algorithm Details]: https://v1-14.docs.kubernetes.io/docs/tasks/run-application/horizontal-pod-autoscale/#algorithm-details

#### API Changes

The following API changes are needed:

- `HorizontalPodAutoscalerBehavior` should be added as a field to the HPA spec. It will contain fields
  which describe the scaling behavior for both scale up and scale down. In the future other aspects of
  the scaling behavior could be customized by adding fields here.
- `HPAScaleConstraint` specifies the maximum change in the number of pods while scaling up or down
  during autoscaling.

The resulting data structures will look like this:

```golang
type HPAScalingPolicyType string

const (
  PercentPolicy HPAScalingPolicyType = "percent"
  PodsPolicy    HPAScalingPolicyType = "pods"
)

type HPAScalingPolicy struct {
  Type          HPAScalingPolicyType
  Value         int32
  PeriodSeconds int32
}

type HPAScalingRules struct {
  StabilizationWindowSeconds *int32
  Policies     []HpaScalingPolicy
  SelectPolicy *string
}

type HorizontalPodAutoscalerBehavior struct {
    ScaleUp   *HPAScalingRules
    ScaleDown *HPAScalingRules
}

type HorizontalPodAutoscalerSpec struct {
    ScaleTargetRef CrossVersionObjectReference
    MinReplicas    *int32
    MaxReplicas    int32
    Metrics        []MetricSpec
    Behavior      *HorizontalPodAutoscalerBehavior
}
```

#### HPA Controller State Changes

To store the history of scaling events, the HPA controller needs an additional field __(similar to the list of “recommendations”)__

```golang
scaleEvents map[string][]timestampedScaleEvent
```

where `timestampedScaleEvent` is

```golang
type timestampedScaleEvent struct {
    replicaChange int32
    timestamp     time.Time
}
```

It will store last scale events and will be used to make decisions about next scale actions.

Say, if 30 seconds ago the number of replicas was increased by one, and we forbid to scale up for more than 1 pod per minute,
then during the next 30 seconds, the HPA controller will not scale up the target again.

If the controller is restarted, the state information is lost so the behavior is not guaranteed anymore and the
controller may scale a target instantly after the restart.

Though, I don’t think this is a problem, as:

- It should not happen often, or you have some cluster problem
- It requires quite a lot of time to start an HPA pod, for HPA pod to become live and ready, to get and process metrics.
- If you have a large discrepancy between what is a desired number of replicas according to metrics and what is your current number of replicas and you DON’T want to scale - probably, you shouldn’t want to use the HPA. As the HPA goal is the opposite.
- The stabilization algorithm already stores recommendations in memory and this has not yet been reported as an issue
  so far.

As the added parameters have default values, we don’t need to update the API version, and may stay on the same `pkg/apis/autoscaling/v2beta2`.

#### Command Line Options Changes

It should be noted that the current [--horizontal-pod-autoscaler-downscale-stabilization-window][] option defines the default value for the
`behavior.stabilizationWindowSeconds`. As it becomes part of the HPA specification, the option is not needed anymore.
So we should make it obsolete but we should keep the existing flag till user have a chance to migrate.

Check the [Default Values][] section for more information about how to determine the delay (priorities of options).

[--horizontal-pod-autoscaler-downscale-stabilization-window]: https://v1-14.docs.kubernetes.io/docs/tasks/run-application/horizontal-pod-autoscale/#algorithm-details
[DefaultValues]: #default-values

## Design Details

### Test Plan

This feature will include the following unit tests to test the following scenarios:

- [TestGenerateScaleDownRules](https://github.com/kubernetes/kubernetes/blob/928817a26a84d9e3076d110ea30ba994912aa477/pkg/apis/autoscaling/v2beta2/defaults_test.go#L29) and [TestGenerateScaleUpRules](https://github.com/kubernetes/kubernetes/blob/928817a26a84d9e3076d110ea30ba994912aa477/pkg/apis/autoscaling/v2beta2/defaults_test.go#L119) verify that the defaults are populated correctly when only a partial set of fields are specified.
- [TestValidateScale](https://github.com/kubernetes/kubernetes/blob/928817a26a84d9e3076d110ea30ba994912aa477/pkg/apis/autoscaling/validation/validation_test.go#L33) and [TestValidateBehavior](https://github.com/kubernetes/kubernetes/blob/928817a26a84d9e3076d110ea30ba994912aa477/pkg/apis/autoscaling/validation/validation_test.go#L97) ensure sanity of values specified in the various fields during HPA creation.
- [TestScaleDownWithScalingRules](https://github.com/kubernetes/kubernetes/blob/928817a26a84d9e3076d110ea30ba994912aa477/pkg/controller/podautoscaler/horizontal_test.go#L1272) and [TestScalingWithRules](https://github.com/kubernetes/kubernetes/blob/928817a26a84d9e3076d110ea30ba994912aa477/pkg/controller/podautoscaler/horizontal_test.go#L3120) test scale up and scale down in single steps when scaling rules are specified.
- [TestStoreScaleEvents](https://github.com/kubernetes/kubernetes/blob/928817a26a84d9e3076d110ea30ba994912aa477/pkg/controller/podautoscaler/horizontal_test.go#L3598) test the storage of events when scaling happens.

### Graduation Criteria

All the new configuration will be added to the `autoscaling/v2beta2` API which has not yet graduated to GA. So these changes do not need a separate
Graduation Criteria and will be part of the existing beta API.
