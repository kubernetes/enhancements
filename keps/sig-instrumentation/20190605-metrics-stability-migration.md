---
title: Control-Plane Metrics Stability Migration
authors:
  - "@logicalhan"
  - "@solodov"
owning-sig:
  - sig-instrumentation
participating-sigs:
  - sig-scheduling
  - sig-node
  - sig-api-machinery
  - sig-cli
  - sig-cloud-provider
reviewers:
  - TBD
approvers:
  - TBD
editor:
  - TBD
creation-date: 2019-06-05
see-also:
  - 20181106-kubernetes-metrics-overhaul
  - 20190404-kubernetes-control-plane-metrics-stability
status: proposal
---

# Control-Plane Metrics Stability Migration

## Table of Contents

* [Table of Contents](#table-of-contents)
* [Summary](#summary)
* [Motivation](#motivation)
    * [Goals](#goals)
    * [Non-Goals](#non-goals)


## Summary

Currently metrics emitted in the Kubernetes control-plane do not offer any stability guarantees. This Kubernetes Enhancement Proposal (KEP) builds off of the framework proposed in the [Kubernetes Control-Plane Metrics Stability KEP](https://github.com/kubernetes/enhancements/blob/master/keps/sig-instrumentation/20190404-kubernetes-control-plane-metrics-stability.md) and proposes a migration strategy for the Kubernetes code base.

## Motivation

We want to start using the metrics stability framework which was based off of [Kubernetes Control-Plane Metrics Stability KEP](https://github.com/kubernetes/enhancements/blob/master/keps/sig-instrumentation/20190404-kubernetes-control-plane-metrics-stability.md).

### Goals

 * Describe a migration strategy for moving metrics onto our metrics stability framework.
 * Maintain backwards compatibility through the migration path. This excludes metrics which are slated to be removed after deprecation due to metrics overhaul.

### Non-Goals

* During migration, we will __*not*__ be determining whether a metric is considered stable. Any metrics which will be promoted to a stable status must be done post-migration.

## Background

The [Kubernetes Control-Plane Metrics Stability KEP](https://github.com/kubernetes/enhancements/blob/master/keps/sig-instrumentation/20190404-kubernetes-control-plane-metrics-stability.md) outlined a framework which we will use to annotate Kubernetes metrics with stability and deprecation metadata. This KEP intends to address our migration strategy.

Kubernetes control-plane binaries can expose one or more metrics endpoints:

* controller-manager - [/metrics](https://github.com/kubernetes/kubernetes/blob/release-1.15/cmd/controller-manager/app/serve.go#L65)
* kube-proxy - [/metrics](https://github.com/kubernetes/kubernetes/blob/release-1.15/cmd/kube-proxy/app/server.go#L570)
* kube-apiserver - [/metrics](https://github.com/kubernetes/kubernetes/blob/release-1.15/staging/src/k8s.io/apiserver/pkg/server/routes/metrics.go#L36)
* kubelet - [four metrics endpoints](https://github.com/kubernetes/kubernetes/blob/release-1.15/staging/src/k8s.io/apiserver/pkg/server/routes/metrics.go#L36)
    * [/metrics](https://github.com/kubernetes/kubernetes/blob/release-1.15/pkg/kubelet/server/server.go#L299)
    * [/metrics/cadvisor](https://github.com/kubernetes/kubernetes/blob/release-1.15/pkg/kubelet/server/server.go#L315)
    * [/metrics/resource](https://github.com/kubernetes/kubernetes/blob/release-1.15/pkg/kubelet/server/server.go#L321-L323)
    * [/metrics/probes](https://github.com/kubernetes/kubernetes/blob/release-1.15/pkg/kubelet/server/server.go#L329-L331)

Ideally, we want to be able to migrate each metrics endpoint individually. Migration in a piecemeal fashion would allow us to avoid having to operate across the entire codebase at once.

However, there are two major shared metric components across binaries:

1. [Kubernetes build info metadata metric](https://github.com/kubernetes/kubernetes/blob/release-1.15/pkg/version/prometheus/prometheus.go#L26-L38) - used by kube-apiserver, controller-manager, hollow-node, kubelet, kube-proxy, scheduler.
2. [Client-go metrics](https://github.com/kubernetes/kubernetes/blob/release-1.15/pkg/util/prometheusclientgo/adapters.go#L20-L24)
    * [client metrics](https://github.com/kubernetes/kubernetes/blob/release-1.15/pkg/client/metrics/prometheus/prometheus.go#L61-L66)
    * [leader-election metrics](https://github.com/kubernetes/kubernetes/blob/release-1.15/pkg/util/prometheusclientgo/leaderelection/adapter.go#L27-L29)
    * [workqueue metrics](https://github.com/kubernetes/kubernetes/blob/release-1.15/pkg/util/workqueue/prometheus/prometheus.go)

This potentially poses a challenge for piecemeal migration. How do we migrate these shared metrics so that we do not have to migrate every component which uses them at the same time?

Consider this:

```golang
func init() {
  leaderelection.SetProvider(prometheusMetricsProvider{})
}

type prometheusMetricsProvider struct{}

func (prometheusMetricsProvider) NewLeaderMetric() leaderelection.SwitchMetric {
  leaderGauge := prometheus.NewGaugeVec(
    prometheus.GaugeOpts{
      Name: "leader_election_master_status",
      Help: "Gauge of if the reporting system is master of the relevant lease, 0 indicates backup, 1 indicates master. 'name' is the string used to identify the lease. Please make sure to group by name.",
    },
    []string{"name"},
  )
  prometheus.Register(leaderGauge)
  return &switchAdapter{gauge: leaderGauge}
}
```

There are two distinct issues:

1. the metrics provider is set in an `init` function which means that the metrics provider is set as a side effect of an import statement like this one:

```golang
import (
  _ "k8s.io/kubernetes/pkg/util/prometheusclientgo/leaderelection" // for leader election metric registration
)
```
This means we cannot use anything in this file without invoking the `init` function (which we would probably want to do in order to migrate metrics).

2. The `NewLeaderMetric` metric creates a new metric and registers that metric to the default global prometheus registry. To enable piecemeal migration, we would want to be able configure a registry and a registerable metric so that we can toggle between using the global prom registry or our internal registry based on whether the component importing this shared metric is migrated or not.

## Proposal

We migrate each endpoint individually, starting with kubelet's metrics/probes endpoint, since it is a relatively thin metrics enpoint.

### Graduation Criteria

TBD

## Drawbacks

TBD

## Alternatives

TBD

## Implementation History

TBD (since this is not yet implemented)

## References
