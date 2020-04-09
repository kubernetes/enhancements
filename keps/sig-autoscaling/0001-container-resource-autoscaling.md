---
title: Container Resource based Autoscaling
authors:
  - "@arjunrn"
owning-sig: sig-autoscaling
reviewers:
  - "@josephburnett"
  - "@mwielgus"
approvers:
  - "@josephburnett"
creation-date: 2020-02-18
last-updated: 2020-02-18
status: provisional
---

# Kubernetes Enhancement Proposal Process

## Table of Contents

<!-- toc -->
- [Summary](#summary)
- [Motivation](#motivation)
  - [Goals](#goals)
  - [Non-Goals](#non-goals)
- [Proposal](#proposal)
  - [User Stories](#user-stories)
    - [Multiple containers with different scaling thresholds](#multiple-containers-with-different-scaling-thresholds)
    - [Multiple containers but only scaling for one.](#multiple-containers-but-only-scaling-for-one)
    - [Add container metrics to existing pod resource metric.](#add-container-metrics-to-existing-pod-resource-metric)
  - [Risks and Mitigations](#risks-and-mitigations)
- [Design Details](#design-details)
  - [Test Plan](#test-plan)
  - [Graduation Criteria](#graduation-criteria)
  - [Upgrade/Downgrade Strategy](#upgradedowngrade-strategy)
- [Implementation History](#implementation-history)
<!-- /toc -->

## Summary

The Horizontal Pod Autoscaler supports scaling of targets based on the resource usage
of the pods in the target. The resource usage of pods is calculated as the sum
of the individual container usage values of the pod. This is unsuitable for workloads where
the usage of the containers are not strongly correlated or do not change in lockstep. This KEP
suggests that when scaling based on resource usage the HPA also provide an option
to consider the usages of individual containers to make scaling decisions.

## Motivation

An HPA is used to ensure that a scaling target is scaled up or down in such a way that the
specificed current metric values are always maintained at a certain level. Resource based
autoscaling is the most basic approach to autoscaling and has been present in the HPA spec since `v1`.
In this mode the HPA controller fetches the current resource metrics for all the pods of a scaling
target and then computes how many pods should be added or removed based on the current usage to
achieve the target average usage.

For performance critical applications where the resource usage of individual containers needs to
be configured individually the default behavior of the HPA controller may be unsuitable. When
there are multiple containers in the pod their individual resource usages may not have a direct
correlation or may grow at different rates as the load changes. There are several reasons for this:

  - A sidecar container is only providing an auxiliary service such as log shipping. If the
    application does not log very frequently or does not produce logs in its hotpath then the usage of
    the log shipper will not grow.
  - A sidecar container which provides authentication. Due to heavy caching the usage will only
    increase slightly when the load on the main container increases. In the current blended usage
    calculation approach this usually results in the the HPA not scaling up the deployment because
    the blended usage is still low.
  - A sidecar may be injected without resources set which prevents scaling based on utilization. In
    the current logic the HPA controller can only scale on absolute resource usage of the pod when
    the resource requests are not set.

The optimum usage of the containers may also be at different levels. Hence the HPA should offer
a way to specify the target usage in a more fine grained manner.

### Goals

- Make HPA scale based on individual container resources usage
- Alias the resource metric source to pod resource metric source.

### Non-Goals
- Configurable aggregation for containers resources in pods.
- Optimization of the calls to the `metrics-server`

## Proposal

Currently the HPA accepts multiple metric sources to calculate the number of replicas in the target,
one of which is called `Resource`. The `Resource` type represents the resource usage of the
pods in the scaling target. The resource metric source has the following structure:

```go
type ResourceMetricSource struct {
	Name v1.ResourceName
	Target MetricTarget
}
```

Here the `Name` is the name of the resource. Currently only `cpu` and `memory` are supported
for this field. The other field is used to specify the target at which the HPA should maintain
the resource usage by adding or removing pods. For instance if the target is _60%_ CPU utilization,
and the current average of the CPU resources across all the pods of the target is _70%_ then
the HPA will add pods to reduce the CPU utilization. If it's less than _60%_ then the HPA will
remove pods to increase utilization.

It should be noted here that when a pod has multiple containers the HPA gets the resource
usage of all the containers and sums them to get the total usage. This is then divided
by the total requested resources to get the average utilizations. For instance if there is
a pods with 2 containers: `application` and `log-shipper` requesting `250m` and `250m` of
CPU resources then the total requested resources of the pod as calculated by the HPA is `500m`.
If then the first container is currently using `200m` and the second only `50m` then
the usage of the pod is `250m` which in utilization is _50%_. Although individually
the utilization of the containers are _80%_ and _20%_. In such a situation the performance
of the `application` container might be affected significantly. There is no way to specify
in the HPA to keep the utilization of the first container below a certain threshold. This also
affects `memory` resource based autocaling scaling.

We propose that the following changes be made to the metric sources to address this problem:

1. A new metric source called `ContainerResourceMetricSource` be introduced with the following
structure:

```go
type ContainerResourceMetricSource struct {
	Container string
	Name v1.ResourceName
	Target MetricTarget
}
```

The only new field is `Container` which is the name of the container for which the resource
usage should be tracked.

2. The `ResourceMetricSource` should be aliased to `PodResourceMetricSource`. It will work
exactly as the original. The aliasing is done for the sake of consistency. Correspondingly,
the `type` field for the metric source should be extended to support both `ContainerResource`
and `PodResource` as values.

### User Stories

#### Multiple containers with different scaling thresholds

Assume the user has a deployment with multiple pods, each of which have multiple containers. A main
container called `application` and 2 others called `log-shipping` and `authnz-proxy`. Two
of the containers are critical to provide the application functionality, `application` and
`authnz-proxy`. The user would like to prevent _OOMKill_ of these containers and also keep
their CPU utilization low to ensure the highest performance. The other container
`log-shipping` is less critical and can tolerate failures and restarts. In this case the
user would create an HPA with the following configuration:

```yaml
apiVersion: autoscaling/v2beta2
kind: HorizontalPodAutoscaler
metadata:
  name: mission-critical
spec:
  scaleTargetRef:
    apiVersion: apps/v1
    kind: Deployment
    name: mission-critical
  minReplicas: 1
  maxReplicas: 10
  metrics:
  - type: ContainerResource
    resource:
      name: cpu
      container: application
      target:
        type: Utilization
        averageUtilization: 30
  - type: ContainerResource
    resource:
      name: memory
      container: application
      target:
        type: Utilization
        averageUtilization: 80
  - type: ContainerResource
    resource:
      name: cpu
      container: authnz-proxy
      target:
        type: Utilization
        averageUtilization: 30
  - type: ContainerResource
    resource:
      name: memory
      container: authnz-proxy
      target:
        type: Utilization
        averageUtilization: 80
  - type: ContainerResource
    resource:
      name: cpu
      container: log-shipping
      target:
        type: Utilization
        averageUtilization: 80
```

The HPA specifies that the HPA controller should maintain the CPU utilization of the containers
`application` and `authnz-proxy` at _30%_ and the memory utilization at _80%_. The `log-shipping`
container is scaled to keep the cpu utilization at _80%_ and is not scaled on memory.

#### Multiple containers but only scaling for one.
Assume the user has a deployment where the pod spec has multiple containers but scaling should
be performed based only on the utilization of one of the containers. There could be several reasons
for such a strategy: Disruptions due to scaling of sidecars may be expensive and should be avoided
or the resource usage of the sidecars could be erratic because it has a different work characteristics
to the main container.

In such a case the user creates an HPA as follows:

```yaml
apiVersion: autoscaling/v2beta2
kind: HorizontalPodAutoscaler
metadata:
  name: mission-critical
spec:
  scaleTargetRef:
    apiVersion: apps/v1
    kind: Deployment
    name: mission-critical
  minReplicas: 1
  maxReplicas: 10
  metrics:
  - type: ContainerResource
    resource:
      name: cpu
      container: application
      target:
        type: Utilization
        averageUtilization: 30
```

The HPA controller will then completely ignore the resource usage in other containers.

#### Add container metrics to existing pod resource metric.
A user who is already using an HPA to scale their application can add the container metric source to the HPA
in addition to the existing pod metric source. If there is a single container in the pod then the behavior
will be exactly the same as before. If there are multiple containers in the application pods then the deployment
might scale out more than before. This happens when the resource usage of the specified container is more
than the blended usage as calculated by the pod metric source. If however in the unlikely case, the usage of
all the containers in the pod change in tandem by the same amount then the behavior will remain as before.

For example consider the HPA object which targets a _Deployement_ with pods that have two containers `application`
and `log-shipper`:

```yaml

apiVersion: autoscaling/v2beta2
kind: HorizontalPodAutoscaler
metadata:
  name: mission-critical
spec:
  scaleTargetRef:
    apiVersion: apps/v1
    kind: Deployment
    name: mission-critical
  minReplicas: 1
  maxReplicas: 10
  metrics:
  - type: ContainerResource
    resource:
      name: cpu
      container: application
      target:
        type: Utilization
        averageUtilization: 50
  - type: PodResource
    resource:
      name: cpu
      target:
        type: Utilization
        averageUtilization: 50
```

If the resource usage of the `application` container increases then the target would be scaled out even if
the usage of the `log-shipper` container does not increase much. If the resource usage of `log-shipper` container
increases then the deployment would only be scaled out if the combined resource usage of both containers increases
above the target.


### Risks and Mitigations

In order to keep backward compatibility with the existing API both `ResourceMetricSource` and
`PodResourceMetricSource` will be supported. Existing HPAs will continue functioning like before.
There will be no deprecation warning or internal migrations from `ResourceMetricSource` to
`PodResourceMetricSource`.


## Design Details

### Test Plan
TBD

### Graduation Criteria

Since the feature is being added to the HPA version `v2beta2` there is no further graduation
criteria required because it will graduate when the original API graduates to `stable`

### Upgrade/Downgrade Strategy

For cluster upgrades the HPAs from the previous version will continue working as before. There
is no change in behavior or flags which have to be enabled or disabled.

For clusters which have HPAs which use `ContainerResourceMetricSource` or `PodResourceMetricSource`
a downgrade is possible after HPAs which use this new source have been modified to use
`ResourceMetricSource` instead.

## Implementation History
