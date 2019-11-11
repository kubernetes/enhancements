---
title: Multi thresholds based Autoscaling.
authors:
  - "@CharlyF"
owning-sig: sig-autoscaling
participating-sigs:
reviewers:
  - TBD
approvers:
  - TBD
editor: TBD
creation-date: 2019-11-11
last-updated: 2019-11-11
status: provisional
see-also:
  - "/keps/sig-autoscaling/20190307-configurable-scale-velocity-for-hpa.md"
replaces:
superseded-by:
---

# Multi thresholds based Autoscaling

## Table of Contents

<!-- toc -->
- [Summary](#summary)
- [Motivation](#motivation)
  - [Goals](#goals)
  - [Non-Goals](#non-goals)
- [Proposal](#proposal)
  - [User Stories](#user-stories-optional)
    - [Canonical use case of the bounds to control the primary scaling condition.](#Canonical-use-case-of-the-bounds-to-control-the-primary-scaling-condition)
    - [Restriction of the scaling events.](#Restriction-of-the-scaling-events)
  - [Implementation Details/Notes/Constraints](#implementation-detailsnotesconstraints)
    - [The Pseudo Code](#the-pseudo-code)
- [Design Details](#design-details)
  - [Upgrade / Downgrade Strategy](#upgrade--downgrade-strategy)
<!-- /toc -->

## Summary

This is a proposal to extend the Horizontal Pod Autoscaler (or HPA) Controller to allow for a better configurability of the scaling behavior.
Namely, avoiding the sawtooth pattern by introducing two thresholds as low and high bounds.
We (Datadog) have been using a Custom Resource internally, the Watermark Pod Autoscaler or WPA, and we believe the community could benefit from this feature. 
Feel free to check out [the official repository](https://github.com/DataDog/watermarkpodautoscaler) and give us your feedback.

Worth mentioning, we have also included some of the features introduced by @gliush
and @arjunrn in in the [Configurable Scale Velocity KEP](https://github.com/kubernetes/enhancements/blob/master/keps/sig-autoscaling/20190307-configurable-scale-velocity-for-hpa.md) in our controller as we believe those features are must-haves for the next verison of the HPA Controller.


## Motivation

- Set a high and a low bound at the resource level to better control autoscaling events.

### Goals

The goal is to improve the ability to customize the parameters triggering a scaling event. We identified 4 vectors of improvement:

- Specifying multiple thresholds on a per HPA basis.
- Modifying the tolerance (`--horizontal-pod-autoscaler-tolerance`) at the HPA level.
- Specify forbidden windows to prevent scaling events.
- Limit the velocity of scaling.

The two last improvements suggestions are tackled in the [Configurable Scale Velocity KEP](https://github.com/kubernetes/enhancements/blob/master/keps/sig-autoscaling/20190307-configurable-scale-velocity-for-hpa.md)

### Non-Goals

- This does not aim at replacing the HPA Controller. 
- Not addressing the features spec'ed in the [Configurable Scale Velocity KEP](https://github.com/kubernetes/enhancements/blob/master/keps/sig-autoscaling/20190307-configurable-scale-velocity-for-hpa.md).

## Proposal

Getting the tolerance parameter (`--horizontal-pod-autoscaler-tolerance`) from the controller level to the resource level is a good start, and has been suggested in the past (see [#39090](https://github.com/kubernetes/kubernetes/issues/39090#issuecomment-466398426)). 
We would like to suggest going beyond this and be able to specify two thresholds. A high bound and a low bound, which extend the idea of a single threshold +/- the tolerance.

There is some overlap between this KEP and the [Configurable Scale Velocity KEP](https://github.com/kubernetes/enhancements/blob/master/keps/sig-autoscaling/20190307-configurable-scale-velocity-for-hpa.md), we (Datadog) are heavily relying on the features they introduced and strongly believe they should be made generally available.

### User Stories

We have instrumented the custom controller we use internally to expose metrics, so we can best illustrate the use cases.

#### Canonical use case of the bounds to control the primary scaling condition.

In this example we are using the following Spec configuration:
```
    minReplicas: 4
    maxReplicas: 9
    metrics:
    - external:
        highWatermark: 400m
        lowWatermark: 150m
        metricName: custom.request_duration.max
        metricSelector:
          matchLabels:
            kubernetes_cluster: mycluster
            service: billing
            short_image: billing-app
      type: External
    tolerance: 0.01
```

Starting with the thresholds, the value of the metric collected (`watermarkpodautoscaler.wpa_controller_value`) in purple when between the bounds (`watermarkpodautoscaler.wpa_controller_low_watermark` and `watermarkpodautoscaler.wpa_controller_high_watermark`) will instruct the controller to not trigger a scaling event. 

We can use the metric `watermarkpodautoscaler.wpa_controller_restricted_scaling{reason:within_bounds}` to verify that it is indeed restricted. (Nota: the metric was multiplied by 1000 in order to make it look more explicit on the graph that within the red areas, no scaling event could have been triggered by the controller).
<img width="1528" alt="Within Watermarks" src="https://user-images.githubusercontent.com/7433560/63385633-e1a67400-c390-11e9-8fee-c547f1876540.png">

#### Restriction of the scaling events.

This section overlaps with the [Configurable Scale Velocity KEP](https://github.com/kubernetes/enhancements/blob/master/keps/sig-autoscaling/20190307-configurable-scale-velocity-for-hpa.md).
The second set of configuration options pertains to the scaling velocity of the target deployment, controlled by `scaleDownLimitFactor` and `scaleUpLimitFactor`.
They are integers between 0 and 100 and represent the maximum ratio of respectively downscaling and upscaling given the current number of replicas.

```
    scaleDownLimitFactor: 30
    scaleUpLimitFactor: 50
```

In this case, should we have 10 replicas and a recommended to upscale to 14, with a scaleUpFactor of 30 (%), we would be capped to 13 replicas.

In the following graph, we can see that the suggested number of replicas (in purple), represented by the metric `watermarkpodautoscaler.wpa_controller_replicas_scaling_proposal`, is too high compared to the current number of replicas. This will trigger the upscale capping logic, which can be monitored using the metric `watermarkpodautoscaler.wpa_controller_restricted_scaling{reason:upscale_capping}` (Nota: Same as above, the metric was multiplied to make it more explicit on the graph). Thus, the effective number of replicas `watermarkpodautoscaler.wpa_controller_replicas_scaling_effective` will scale up, but limited by the `scaleUpLimitFactor`
<img width="911" alt="Upscale Capping" src="https://user-images.githubusercontent.com/7433560/63385168-f46c7900-c38f-11e9-9e7c-1a7796afd31e.png">

In this similar example, we avoid downscaling by too many replicas, and can use the same set of metrics to guarantee that we only scale down by a reasonable amount of replicas.
<img width="911" alt="Downscale Capping" src="https://user-images.githubusercontent.com/7433560/63385340-44e3d680-c390-11e9-91d4-35b2f8a912ad.png">

The last options available to restrict scaling are `downscaleForbiddenWindowSeconds` and `upscaleForbiddenWindowSeconds` in seconds that represent respectively how much time to wait before scaling down and scaling up after **any scaling event**. We do not compare the `upscaleForbiddenWindowSeconds` to the last time we only upscaled, so a downscale event will prevent an upscale event for a duration of `upscaleForbiddenWindowSeconds`.

```
    downscaleForbiddenWindowSeconds: 60
    upscaleForbiddenWindowSeconds: 30
```

In the following example we can see that the recommended number of replicas is ignored if we are in a cooldown period. The downscale cooldown period can be visualised with `watermarkpodautoscaler.wpa_controller_transition_countdown{transition:downscale}`, and is represented in yellow on the graph below. We can see that it is significantly higher than the upscale cooldown period (`transition:upscale`) in dark orange on our graph. As soon as we are recommended to scale, only if the appropriate cooldown window is over, will we scale. This will reset both countdowns.
<img width="911" alt="Forbidden Windows" src="https://user-images.githubusercontent.com/7433560/63389864-a14cf300-c39c-11e9-9ad5-8308af5442ad.png">

### Implementation Details/Notes/Constraints

First and foremost, upon making a measurement (i.e. querying the Metrics Server or Custom/External Metrics Server) we compare the value to both thresholds, we can only suggest upscaling if the value is at least higher than the high threshold and a downscale event if it is at least lower than the low threshold.

As we add a second threshold, it seems confusing and error prone to keep the logic of the `targetAverageValue` and the `targetValue` nomenclature for those two thresholds. e.g. have the LowThreshold/LowAverageThreshold and deal with cases where a user could set a HighThreshold and a LowAverageThreshold. 
So we decided to implement a separate option in the spec, `algorithm`.

Depending on your use case, you might want to consider one or the other. The naming of the algorithm is from the point of view of the controller.

1. `average`
    The ratio used to compare to the thresholds is `value from the metrics provider` / `current number of replicas`, it is compared to the bounds and the recommended number of replicas is `value from the metrics provider` / `threshold` (low or high depending on the current value).
    - The `average` algorithm is a good fit if you use a metric that does not directly depend on the number of replicas. Typically, the sum of requests received by a Load Balancer can indicate how many webservers we want to have given that we know that a single webserver should handle X rq/s.
    Adding a replica will not increase/decrease the number of requests received.

2. `absolute`
    The default value is `absolute`, we compare the raw metric retrieved as is from the Metrics Provider to the thresholds and it is used as the utilization value. The recommended number of replicas is computed as `current number of replicas` * `value from the metrics provider` / `threshold`.
    - The `absolute` algorithm is the default as it represents the most common use case: You want your application to run between 60% and 80% of CPU, if the metric (avg:cpu.usage) retrieved is at 85%, you need to scale up. The metric has to be correlated to the number of replicas.

This yields the same behavior as the HPA controller where `TargetValue` is equivalent to `absolute` and `TargetAverageValue` is the `average` algorithm.

It is worth noting that in the upstream controller only the `math.Ceil` function is used to compute the recommended number of replicas.
This means that if you have a threshold at 10, you will need to reach a utilization of 8.999... from the Metrics Provider to downscale to 9 replicas but 10.001 will suffice trigger an upscale event to 11 replicas.

In order to guaratee a symetry of scaling, our implementation of this custom controller uses `math.Floor` if the value is under the Lower bound. We are not suggesting porting this to the upstream controller, but we think it is important to bring it up to have the opportunity to discuss if it is and why it would be a desired behavior.

Bounds are specified as `Quantities`, like the threshold so you can use `m | "" | k | M | G | T | P | E` to easily represent the value you want to use.

#### The Pseudo Code

The main change would go in the `replica_calculator.go`

```go
  averaged := 1.0
  if wpa.Spec.Algorithm == "average" {
    averaged = float64(currentReplicas)
  }

  var sum int64
  for _, val := range metrics {
    sum += val
  }

  adjustedUsage := float64(sum) / averaged
  utilizationQuantity := resource.NewMilliQuantity(int64(adjustedUsage), resource.DecimalSI)

  adjustedHM := float64(highMark.MilliValue()) + wpa.Spec.Tolerance*float64(highMark.MilliValue())
  adjustedLM := float64(lowMark.MilliValue()) - wpa.Spec.Tolerance*float64(lowMark.MilliValue())

  // We do not use the abs as we want to know if we are higher than the high mark or lower than the low mark
  switch {
  case adjustedUsage > adjustedHM:
    replicaCount = int32(math.Ceil(float64(currentReplicas) * adjustedUsage / (float64(highMark.MilliValue()))))
  case adjustedUsage < adjustedLM:
    replicaCount = int32(math.Floor(float64(currentReplicas) * adjustedUsage / (float64(lowMark.MilliValue()))))
  default:
    return currentReplicas, utilizationQuantity.MilliValue(), timestamp, nil
  }

  return replicaCount, utilizationQuantity.MilliValue(), timestamp, nil
```
## Design Details
### Upgrade / Downgrade Strategy

Should this suggestion be accepted, we should make sure to have a backward compatible behavior. 
My suggestion would be to fallback to higher bound = lower bound to get the same current behavior.

More details will be added to this section if this moves forward.