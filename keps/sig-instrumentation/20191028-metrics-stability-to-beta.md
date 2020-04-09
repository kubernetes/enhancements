---
title: Metrics Stability Framework to Beta
authors:
  - "@logicalhan"
  - "@RainbowMango"
owning-sig: sig-instrumentation
participating-sigs:
  - sig-instrumentation
reviewers:
  - "@brancz"
approvers:
  - "@brancz"
editor: "@brancz"
creation-date: 2019-10-28
last-updated: 2019-10-28
status: implementable
see-also:
  - 20181106-kubernetes-metrics-overhaul
  - 20190404-kubernetes-control-plane-metrics-stability
  - 20190605-metrics-stability-migration
  - 20190605-metrics-validation-and-verification
---

# Metrics Stability Framework to Beta

## Table of Contents

<!-- toc -->
- [Summary](#summary)
- [Motivation](#motivation)
  - [Goals](#goals)
  - [Non-Goals](#non-goals)
- [Proposal](#proposal)
  - [Remove Prometheus Registry](#remove-prometheus-registry)
  - [Validated Import Restriction](#validated-import-restriction)
  - [Deprecate Metrics](#deprecate-metrics)
  - [Escape Hatch](#escape-hatch)
- [Graduation Criteria](#graduation-criteria)
- [Post-Beta tasks](#post-beta-tasks)
- [Implementation History](#implementation-history)
  - [Metrics Stability Framework](#metrics-stability-framework)
  - [Metrics Stability Migration](#metrics-stability-migration)
  - [Metrics Validation And Restriction](#metrics-validation-and-restriction)
  - [Deprecate Metrics](#deprecate-metrics-1)
  - [Escape Flag](#escape-flag)
<!-- /toc -->

## Summary

The metrics stability framework has been added to the Kubernetes project as a way to annotate metrics with a supported stability level. Depending on the stability level of a metric, there are some guarantees one can expect as a consumer (i.e. ingester) of a given metric. This document outline required steps to graduate it to Beta.

## Motivation

The metrics stability framework is currently used for defining metrics stability levels for metrics in OSS Kubernetes. The motivation
of this KEP is to address those feature requests and bug reports to move this feature to the Beta level.

### Goals

These are the planned changes for Beta feature graduation:

* No Kubernetes binaries register metrics to prometheus registries directly.
* There is a validated import restriction on all kubernetes binaries (except `component-base/metrics`) such that we will fail, in a precommit phase, a direct import of prometheus in kubernetes. This forces all metrics related code to go through the metrics stability framework.
* All currently deprecated metrics are deprecated using the `DeprecatedVersion` field of metrics options struct.
* All Kubernetes binaries should have a command flag `--show-hidden-metrics` by which cluster admins can show metrics deprecated in last minor release.

### Non-Goals

These are the issues considered and rejected for Beta:

* Being able to individually turn off a metric (this will be a GA feature).

## Proposal

### Remove Prometheus Registry
In order to achieve the first goal: no binaries will register metrics to prometheus registries directly, we must have a plan for migrating metrics which are defined through the `prometheus.Collector` interface. These metrics currently do not have a way to express a stability level. @RainbowMango has a [PR with an implementation of how we may accomplish this](https://github.com/kubernetes/kubernetes/pull/83062/). Alternatively, we can just default all metrics which are defined through a custom `prometheus.Collector` as falling under stability level ALPHA, i.e. they do not offer stability guarantees. This buys us runway in bridging over to a solution like the one @RainbowMango proposes.

### Validated Import Restriction
We also want to validate that direct prometheus imports are no longer possible in Kubernetes outside of component-base/metrics. This will force metric definition to occur within the stability framework and allow us to provide the guarantees that we intend. @serathius has some ideas in a [PR here](https://github.com/kubernetes/kubernetes/pull/84302).

### Deprecate Metrics
The goal merely requires migrating over deprecated metrics from [PR](tdb).

### Escape Hatch
We should add a command flag, such as `--show-hidden-metrics`, to each Kubernetes binaries.
This is to provide cluster admins an escape hatch to properly migrate off of a deprecated metric, if they were not able to react to the earlier deprecation warnings.


## Graduation Criteria

To mark these as complete, all of the above features need to be implemented.
An [umbrella issue](https://github.com/kubernetes/kubernetes/issues/tdb) is tracking all of these changes.
Also there need to be sufficient tests for any of these new features and all existing features and documentation should be completed for all features.

There are still open questions that need to be addressed and updated in this KEP before graduation:

## Post-Beta tasks

These are related Post-GA tasks:

*

## Implementation History

### Metrics Stability Framework
- Setup framework
  - [x] https://github.com/kubernetes/kubernetes/pull/77037 (by @logicalhan)
  - [x] https://github.com/kubernetes/kubernetes/pull/77618 (by @logicalhan)
  - [x] https://github.com/kubernetes/kubernetes/pull/78773 (by @logicalhan)
  - [x] https://github.com/kubernetes/kubernetes/pull/78867 (by @logicalhan)
  - [x] https://github.com/kubernetes/kubernetes/pull/78877 (by @logicalhan)
  - [x] https://github.com/kubernetes/kubernetes/pull/79237 (by @logicalhan)
  - [x] https://github.com/kubernetes/kubernetes/pull/81190 (by @logicalhan)
  - [x] https://github.com/kubernetes/kubernetes/pull/81395 (by @logicalhan)
  - [x] https://github.com/kubernetes/kubernetes/pull/81579 (by @logicalhan)
  - [x] https://github.com/kubernetes/kubernetes/pull/81608 (by @logicalhan)
- Introduce bucket functionality
  - [x] https://github.com/kubernetes/kubernetes/pull/82583 (by @RainbowMango)
- Deal with stability default level
  - [x] https://github.com/kubernetes/kubernetes/pull/82957 (by @RainbowMango)
- Introduce label functionality
  - [x] https://github.com/kubernetes/kubernetes/pull/83019 (by @RainbowMango)
- Introduce test util:
  - [x] https://github.com/kubernetes/kubernetes/pull/83299 (by @RainbowMango)
  - [x] https://github.com/kubernetes/kubernetes/pull/83699 (by @RainbowMango)
- Introduce http handler functionality
  - [x] https://github.com/kubernetes/kubernetes/pull/83722 (by @RainbowMango)
- Introduce GaugeFunc
  - [X] https://github.com/kubernetes/kubernetes/pull/83830 (by @RainbowMango)
- Introduce custom collector
  - [ ] https://github.com/kubernetes/kubernetes/pull/83062 (by @RainbowMango)
- Cleanup
  - [ ] https://github.com/kubernetes/kubernetes/pull/84135 (by @RainbowMango)
  - [x] https://github.com/kubernetes/kubernetes/pull/81432 (by @logicalhan)
- Bug fix
  - [x] https://github.com/kubernetes/kubernetes/pull/84395 (by @RainbowMango)

### Metrics Stability Migration
- General Migration
  - [x] for shared metrics: https://github.com/kubernetes/kubernetes/pull/81173 (by @logicalhan)
  - [x] for apiserver: https://github.com/kubernetes/kubernetes/pull/81531 (by @logicalhan)
  - [x] for kubelet: https://github.com/kubernetes/kubernetes/pull/81534 (by @logicalhan)
  - [x] for scheduler: https://github.com/kubernetes/kubernetes/pull/81576 (by @logicalhan)
  - [x] for controller manager: https://github.com/kubernetes/kubernetes/pull/81624 (by @logicalhan)
  - [x] for kube-proxy: https://github.com/kubernetes/kubernetes/pull/81626 (by @logicalhan)
  - [x] for etcd version monitor: https://github.com/kubernetes/kubernetes/pull/83283 (by @RainbowMango)
  - [ ] for metrics validation framework: https://github.com/kubernetes/kubernetes/pull/84500 (by @RainbowMango)
- Migrate bucket functionality
  - [x] https://github.com/kubernetes/kubernetes/pull/82626 (by @RainbowMango)
  - [x] https://github.com/kubernetes/kubernetes/pull/82630 (by @RainbowMango)
  - [x] https://github.com/kubernetes/kubernetes/pull/82736 (by @RainbowMango)
  - [x] https://github.com/kubernetes/kubernetes/pull/82737 (by @RainbowMango)
  - [x] https://github.com/kubernetes/kubernetes/pull/82741 (by @RainbowMango)
  - [x] https://github.com/kubernetes/kubernetes/pull/82745 (by @RainbowMango)
- Migrate bucket functionality
  - [x] https://github.com/kubernetes/kubernetes/pull/83159 (by @RainbowMango)
  - [x] https://github.com/kubernetes/kubernetes/pull/83220 (by @RainbowMango)
  - [x] https://github.com/kubernetes/kubernetes/pull/83223 (by @RainbowMango)
  - [x] https://github.com/kubernetes/kubernetes/pull/83269 (by @RainbowMango)
  - [x] https://github.com/kubernetes/kubernetes/pull/83278 (by @RainbowMango)
  - [x] https://github.com/kubernetes/kubernetes/pull/83279 (by @RainbowMango)
- Migrate or refactor test case
  - [x] https://github.com/kubernetes/kubernetes/pull/83611 (by @RainbowMango)
  - [x] https://github.com/kubernetes/kubernetes/pull/83678 (by @RainbowMango)
  - [x] https://github.com/kubernetes/kubernetes/pull/83713 (by @RainbowMango)
  - [ ] https://github.com/kubernetes/kubernetes/pull/83664 (by @RainbowMango)
  - [x] https://github.com/kubernetes/kubernetes/pull/84283 (by @serathius)
- Migrate promhttp
  - [ ] https://github.com/kubernetes/kubernetes/pull/84393 (by @wuyafang)
  - [x] https://github.com/kubernetes/kubernetes/pull/84221 (by @wuyafang)

### Metrics Validation And Restriction
- [x] https://github.com/kubernetes/kubernetes/pull/80803 (by @serathius)
- [x] https://github.com/kubernetes/kubernetes/pull/80906 (by @serathius)
- [x] https://github.com/kubernetes/kubernetes/pull/81510 (by @serathius)
- [ ] https://github.com/kubernetes/kubernetes/pull/84302 (by @serathius)
- [ ] https://github.com/kubernetes/kubernetes/pull/84373 (by @serathius)
- [ ] https://github.com/kubernetes/kubernetes/pull/84378 (by @serathius)

### Deprecate Metrics
- [ ] https://github.com/kubernetes/kubernetes/pull/83836 (by @RainbowMango)
- [ ] https://github.com/kubernetes/kubernetes/pull/83837 (by @RainbowMango)
- [ ] https://github.com/kubernetes/kubernetes/pull/83838 (by @RainbowMango)
- [ ] https://github.com/kubernetes/kubernetes/pull/83839 (by @RainbowMango)
- [ ] https://github.com/kubernetes/kubernetes/pull/83841 (by @RainbowMango)

### Escape Flag
- [ ] https://github.com/kubernetes/kubernetes/pull/84292 (by @RainbowMango)