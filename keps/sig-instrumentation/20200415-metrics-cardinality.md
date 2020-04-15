---
title: Metrics Cardinality
authors:
  - "@logicalhan"
owning-sig: sig-instrumentation
participating-sigs:
  - sig-instrumentation
reviewers:
  - todo
approvers:
  - todo
editor: todo
creation-date: 2020-04-15
last-updated: 2020-04-15
status: proposal
---

# Metrics Cardinality

## Table of Contents

<!-- toc -->
- [Summary](#summary)
- [Motivation](#motivation)
  - [Goals](#goals)
  - [Non-Goals](#non-goals)
- [Proposal](#proposal)
- [Graduation Criteria](#graduation-criteria)
- [Post-Beta tasks](#post-beta-tasks)
- [Implementation History](#implementation-history)
<!-- /toc -->

## Summary

Metrics emitted by OSS Kubernetes components can have unbounded dimensions, which in turn implies that the metric itself is unbounded in cardinality. Though often innocous, the failure mode of such metrics can be catastrophic, since metrics are held in memory (by default) by the instrumented process (which has a finite amount of memory). This KEP proposes introducing a mechanism through which we can specify a finite set of label values for a metric label.


## Motivation

Metric with unbounded dimensions can cause OSS Kubernetes components to OOM (not a desirable outcome). Historically, we have patched these metrics in backwards incompatible ways (i.e. removing metrics, deleting labels, etc) and cherry-picked the fixes to older releases (todo add links to issues/PRs). This process is manual and has proven to be both laborious and time-consuming, especially when the metric cardinality issue can be weaponized in a security exploit (todo add links to issues/PRs), since that greatly increases the urgency to apply a fix.

Usually there are triggering conditions which cause community members to realize that a metric is problematic; oftentimes a metric's dimensions increase in number due to a new OSS feature or, alternatively, increasing the useage of an existing feature can also increase metric cardinality.

### Goals

Provide a mechanism for enabling bounded dimensions for labels.

### Non-Goals

This is a purely SIG Instrumentation effort. We will expose the ability to bound labels. Individual component owners can opt to feed data into this during startup through some config loading mechanism (i.e. command line flags or reading from a file).

## Proposal

todo


## Graduation Criteria

todo


## Post-Beta tasks

todo

## Implementation History

todo