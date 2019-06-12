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

Setup a strategy for migrating metrics to the stability framework. Also:

 * Explicitly define a strategy for handling migration of shared metrics.
 * Maintain backwards compatibility through the migration path. This excludes metrics which are slated to be removed after deprecation due to metrics overhaul.

### Non-Goals

* During migration, we will __*not*__ be determining whether a metric is considered stable. Any metrics which will be promoted to a stable status must be done post-migration.
* Kubelet '/metrics/resource' and '/metrics/cadvisor' are out of scope for this migration.

## General Migration Strategy

To keep migration PRs limited in scope (i.e. ideally belonging to a single component owner at a time), we prefer to approach migration in a piecemeal fashion. Each metrics endpoint can be considered an atomic unit of migration. This will allow us to avoid migrating the entire codebase at once.

Kubernetes control-plane binaries can expose one or more metrics endpoints:

* one controller-manager metrics endpoint
    * [/metrics](https://github.com/kubernetes/kubernetes/blob/release-1.15/cmd/controller-manager/app/serve.go#L65)
* one kube-proxy metrics endpoint
    * [/metrics](https://github.com/kubernetes/kubernetes/blob/release-1.15/cmd/kube-proxy/app/server.go#L570)
* one kube-apiserver metrics endpoint
    * [/metrics](https://github.com/kubernetes/kubernetes/blob/release-1.15/staging/src/k8s.io/apiserver/pkg/server/routes/metrics.go#L36)
* [four kubelet metrics endpoints](https://github.com/kubernetes/kubernetes/blob/release-1.15/staging/src/k8s.io/apiserver/pkg/server/routes/metrics.go#L36)
    * [/metrics](https://github.com/kubernetes/kubernetes/blob/release-1.15/pkg/kubelet/server/server.go#L299)
    * [/metrics/cadvisor](https://github.com/kubernetes/kubernetes/blob/release-1.15/pkg/kubelet/server/server.go#L315)
    * [/metrics/resource](https://github.com/kubernetes/kubernetes/blob/release-1.15/pkg/kubelet/server/server.go#L321-L323)
    * [/metrics/probes](https://github.com/kubernetes/kubernetes/blob/release-1.15/pkg/kubelet/server/server.go#L329-L331)

Following our desired approach, each of metrics endpoint above (except those out of scope) will be accompanied by an individual PR for migrating that endpoint.

## Strategy Around Shared Metrics

Shared metrics makes piecemeal migration potentially difficult because metrics in shared code will either be migrated or not, and a component which uses the shared metric can be in the opposite state. Currently, there are groups of metrics which are __*shared*__ across binaries:

1. [Kubernetes build info metadata metric](https://github.com/kubernetes/kubernetes/blob/release-1.15/pkg/version/prometheus/prometheus.go#L26-L38) - used by kube-apiserver, controller-manager, hollow-node, kubelet, kube-proxy, scheduler.
2. [Client-go metrics](https://github.com/kubernetes/kubernetes/blob/release-1.15/pkg/util/prometheusclientgo/adapters.go#L20-L24)
    * [client metrics](https://github.com/kubernetes/kubernetes/blob/release-1.15/pkg/client/metrics/prometheus/prometheus.go#L61-L66)
    * [leader-election metrics](https://github.com/kubernetes/kubernetes/blob/release-1.15/pkg/util/prometheusclientgo/leaderelection/adapter.go#L27-L29)
    * [workqueue metrics](https://github.com/kubernetes/kubernetes/blob/release-1.15/pkg/util/workqueue/prometheus/prometheus.go)

We should be able to migrate shared metrics in such a way that does not require migrating every component which consumes/uses these shared metrics simulaneously. There are a couple possible ways we can avoid this:

1. We can refactor the shared metrics code so that we can customize the metrics registration codepaths. This likely involves removing the `init` calls to set metricsProviders and reworking of the codepaths which register metrics to the global prometheus registry. This is invasive and potentially complicated.
2. We can simply duplicate the file and create a version of the shared metrics which uses the new registries. When we migrate an endpoint, we can remove the import statements to the old file and replace it with references to our migrated variant. This has the additional benefit that once migration is complete, we can just delete the old variants of shared metrics.

## Implementation History

TBD (since this is not yet implemented)
