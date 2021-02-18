# Metrics Stability Migration

## Table of Contents

<!-- toc -->
- [Summary](#summary)
- [Motivation](#motivation)
  - [Goals](#goals)
  - [Non-Goals](#non-goals)
- [General Migration Strategy](#general-migration-strategy)
  - [Shared metrics](#shared-metrics)
  - [Deprecation of modified metrics from metrics overhaul KEP](#deprecation-of-modified-metrics-from-metrics-overhaul-kep)
- [Implementation History](#implementation-history)
- [Graduation Criteria](#graduation-criteria)
- [Testing Plan](#testing-plan)
<!-- /toc -->


## Summary

This KEP intends to document and communicate the general strategy for migrating the control-plane metrics stability framework across the Kubernetes codebase. Most of the framework design decisions have been determined and outlined in [an earlier KEP](./kubernetes-control-plane-metrics-stability.md).

## Motivation

We want to start using the metrics stability framework built based off the [Kubernetes Control-Plane Metrics Stability KEP](./kubernetes-control-plane-metrics-stability.md), so that we can define stability levels for metrics in the Kubernetes codebase. These stability levels would provide API compatibility guarantees across version bumps.

### Goals

 * Outline the general strategy for migrating metrics to the stability framework
 * Explicitly define a strategy for handling migration of shared metrics.
 * Maintain backwards compatibility through the migration path. *This excludes metrics which are slated to be removed after deprecation due to metrics overhaul.*
 * Communicate upcoming changes to metrics to respective component owners.

### Non-Goals

* During migration, we will __*not*__ be determining whether a metric is considered stable. Any metrics which will be promoted to a stable status must be done post-migration.
* Kubelet '/metrics/resource' and '/metrics/cadvisor' are out of scope for this migration due to use of non-standard prometheus collectors.

## General Migration Strategy

To keep migration PRs limited in scope (i.e. ideally belonging to a single component owner at a time), we prefer to approach migration in a piecemeal fashion. Each metrics endpoint can be considered an atomic unit of migration. This will allow us to avoid migrating the entire codebase at once.

Kubernetes control-plane binaries can expose one or more metrics endpoints:

* one controller-manager metrics endpoint
    * [/metrics](https://github.com/kubernetes/kubernetes/blob/release-1.15/cmd/controller-manager/app/serve.go#L65)
* one kube-proxy metrics endpoint
    * [/metrics](https://github.com/kubernetes/kubernetes/blob/release-1.15/cmd/kube-proxy/app/server.go#L570)
* one kube-apiserver metrics endpoint
    * [/metrics](https://github.com/kubernetes/kubernetes/blob/release-1.15/staging/src/k8s.io/apiserver/pkg/server/routes/metrics.go#L36)
* one scheduler metrics endpoint
    * [/metrics](https://github.com/kubernetes/kubernetes/blob/release-1.15/cmd/kube-scheduler/app/server.go#L289)
* [four kubelet metrics endpoints](https://github.com/kubernetes/kubernetes/blob/release-1.15/staging/src/k8s.io/apiserver/pkg/server/routes/metrics.go#L36)
    * [/metrics](https://github.com/kubernetes/kubernetes/blob/release-1.15/pkg/kubelet/server/server.go#L299)
    * [/metrics/probes](https://github.com/kubernetes/kubernetes/blob/release-1.15/pkg/kubelet/server/server.go#L329-L331)
    * (out of scope) ~~[/metrics/cadvisor](https://github.com/kubernetes/kubernetes/blob/release-1.15/pkg/kubelet/server/server.go#L315)~~
    * (out of scope) ~~[/metrics/resource](https://github.com/kubernetes/kubernetes/blob/release-1.15/pkg/kubelet/server/server.go#L321-L323)~~

Following our desired approach, each of metrics endpoint above (except those out of scope) will be accompanied by an individual PR for migrating that endpoint.

### Shared metrics

Shared metrics makes piecemeal migration potentially difficult because metrics in shared code will either be migrated or not, and a component which uses the shared metric can be in the opposite state. Currently, there are groups of metrics which are __*shared*__ across binaries:

1. [Kubernetes build info metadata metric](https://github.com/kubernetes/kubernetes/blob/release-1.15/pkg/version/prometheus/prometheus.go#L26-L38) - used by kube-apiserver, controller-manager, hollow-node, kubelet, kube-proxy, scheduler.
2. [Client-go metrics](https://github.com/kubernetes/kubernetes/blob/release-1.15/pkg/util/prometheusclientgo/adapters.go#L20-L24)
    * [client metrics](https://github.com/kubernetes/kubernetes/blob/release-1.15/pkg/client/metrics/prometheus/prometheus.go#L61-L66)
    * [leader-election metrics](https://github.com/kubernetes/kubernetes/blob/release-1.15/pkg/util/prometheusclientgo/leaderelection/adapter.go#L27-L29)
    * [workqueue metrics](https://github.com/kubernetes/kubernetes/blob/release-1.15/pkg/util/workqueue/prometheus/prometheus.go)

Our strategy around shared metrics is to simply duplicate shared metric files and create a version of the shared metrics which uses the new registries. When we migrate an endpoint, we can remove the import statements to the old file and replace it with references to our migrated variant. This has the additional benefit that once migration is complete, we can just delete the old variants of shared metrics.

### Deprecation of modified metrics from metrics overhaul KEP

The [metrics overhaul KEP](https://github.com/kubernetes/enhancements/blob/master/keps/sig-instrumentation/20181106-kubernetes-metrics-overhaul.md) deprecates a number of metrics across the Kubernetes code-base. As a part of this migration, these metrics will be marked as deprecated since 1.16, meaning that they will be hidden automatically for 1.17. These metrics will be in a deprecated but 'hidden' state and will be able to be optionally enabled through command line flags for one release cycle, and they will be permanently deleted in 1.18.

## Implementation History

- [x] [Migrate kubelet metrics to use standard prometheus collectors](https://github.com/kubernetes/kubernetes/issues/79286)
- [x] Create [migrated variants of shared client metrics](https://github.com/kubernetes/kubernetes/pull/81173)
- [x] Create [migrated variants of shared leader-election metrics](https://github.com/kubernetes/kubernetes/pull/81173)
- [x] Create [migrated variants of shared workqueue metrics](https://github.com/kubernetes/kubernetes/pull/81173)
- [x] Migrate [kubelet's /metrics/probes endpoint](https://github.com/kubernetes/kubernetes/pull/81534)
- [x] Migrate [apiserver /metrics endpoint](https://github.com/kubernetes/kubernetes/pull/81531)
- [x] Migrate [scheduler /metrics endpoint](https://github.com/kubernetes/kubernetes/pull/81576)
- [x] Migrate [kube-proxy /metrics endpoint](https://github.com/kubernetes/kubernetes/pull/81626)
- [x] Migrate [controller-manager /metrics endpoint (this include in-tree cloud-provider metrics)](https://github.com/kubernetes/kubernetes/pull/81624)

TBD (since this is not yet implemented)

## Graduation Criteria

- [x] Prior to migrating a [component, automated static analysis testing](./metrics-validation-and-verification.md) is in place to validate and verify API guarantees.
- [x] Adequate [documentation exists for new flags on components](https://kubernetes.io/docs/reference/command-line-tools-reference/kube-apiserver/)
- [x] Update [instrumentation documents to reflect changes](https://github.com/kubernetes/website/pull/17578)

## Testing Plan

- [x] Prior to migrating a metric's endpoint, we will run local tests to verify that the same metrics are populated
- [x] All metrics framework code will have unit/integration tests
- [x] All validation and verification code will have unit/integration tests
