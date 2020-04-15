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

Metrics emitted by OSS Kubernetes components can have unbounded dimensions, which in turn implies that the metric itself is unbounded in cardinality. Though often innocous, the failure mode of such metrics can be catastrophic, since metrics are held in memory (by default) by the instrumented process (which has a finite amount of memory).

Usually there are triggering conditions which cause community members to realize that a metric is problematic; oftentimes a metric's dimensions increase in number due to a new OSS feature or, alternatively, increasing the useage of an existing feature can also increase metric cardinality.

Historically, we have patched these metrics in backwards incompatible ways (i.e. removing metrics, deleting labels, etc) and cherry-picked the fixes to older releases. This process is manual and has proven to be both laborious and time-consuming, especially when the metric cardinality issue can be weaponized in a security exploit (making the specific incident considerably higher priority).

## Motivation



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