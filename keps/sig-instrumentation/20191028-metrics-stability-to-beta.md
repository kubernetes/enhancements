---
title: Metrics Stability Framework to Beta
authors:
  - "@logicalhan"
owning-sig:
  - sig-instrumentation
participating-sigs:
  - sig-instrumentation
reviewers:
  - @brancz
approvers:
creation-date: 2019-10-28
see-also:
  - 20181106-kubernetes-metrics-overhaul
status: proposed
---

# Metrics Stability Framework to Beta

## Table of Contents

* [Table of Contents](#table-of-contents)
* [Summary](#summary)

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

### Non-Goals

These are the issues considered and rejected for Beta:

* Being able to individually turn off a metric (this will be a GA feature).

## Proposal

In order to achieve the first goal: no binaries will register metrics to prometheus registries directly, we must have a plan for migrating metrics which are defined through the `prometheus.Collector` interface. These metrics currently do not have a way to express a stability level. @RainbowMango has a [PR with an implementation of how we may accomplish this](https://github.com/kubernetes/kubernetes/pull/83062/). Alternatively, we can just default all metrics which are defined through a custom `prometheus.Collector` as falling under stability level ALPHA, i.e. they do not offer stability guarantees. This buys us runway in bridging over to a solution like the one @RainbowMango proposes.

We also want to validate that direct prometheus imports are no longer possible in Kubernetes outside of component-base/metrics. This will force metric definition to occur within the stability framework and allow us to provide the guarantees that we intend. @serathius has some ideas in a [PR here](https://github.com/kubernetes/kubernetes/pull/84302).

The last goal merely requires migrating over deprecated metrics from [PR](tdb).

## Graduation Criteria

To mark these as complete, all of the above features need to be implemented.
An [umbrella issue](https://github.com/kubernetes/kubernetes/issues/tdb) is tracking all of these changes.
Also there need to be sufficient tests for any of these new features and all existing features and documentation should be completed for all features.

There are still open questions that need to be addressed and updated in this KEP before graduation:

## Post-Beta tasks

These are related Post-GA tasks:

*

## Implementation History

