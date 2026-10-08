# KEP-6265: StatefulSet Rollouts Respect Node Lifecycle State

## Table of Contents

<!-- toc -->
- [Summary](#summary)
- [Motivation](#motivation)
  - [Goals](#goals)
  - [Non-Goals](#non-goals)
- [Proposal](#proposal)
  - [User Stories](#user-stories)
  - [Notes/Constraints/Caveats](#notesconstraintscaveats)
  - [Risks and Mitigations](#risks-and-mitigations)
- [Design Details](#design-details)
- [Graduation Criteria](#graduation-criteria)
- [Implementation History](#implementation-history)
<!-- /toc -->

## Summary

StatefulSets preserve stable Pod identity and roll out Pods in order, so the
controller cannot create a replacement Pod with the same identity while the
old Pod still exists. When a Pod is unavailable because its Node is shutting
down, unreachable, or under maintenance, this can block the rollout with no
indication of why. This KEP proposes surfacing Node lifecycle state (as
reported by [KEP-5683: Node Lifecycle Conditions](/keps/sig-node/5683-lifecycle-conditions))
in StatefulSet status, so it's clear when a rollout is waiting on Node
lifecycle activity rather than the new revision, storage attachment, or
scheduling.

## Motivation

A single Pod stuck on a Node affected by lifecycle activity (e.g. a Node
condition such as `MaintenanceInProgress`) can halt StatefulSet rollout
progress. Today, users and rollout tooling can only see that the
StatefulSet is waiting for Pods to become ready, with no way to tell whether
that's caused by the new revision, storage attachment, scheduling, or
intentional Node maintenance.

### Goals

- Report, in StatefulSet status, when a StatefulSet Pod is unavailable due
  to Node lifecycle state rather than another cause.

### Non-Goals

- Changing StatefulSet rollout, eviction, or Pod replacement behavior.
- ReplicaSet or other controller coordination with Node lifecycle state
  (tracked separately, see KEP-5683).

## Proposal

Extend StatefulSet status reporting so that when a Pod blocking a rollout is
on a Node reporting a lifecycle condition (e.g. `MaintenanceInProgress`),
that context is visible on the StatefulSet, instead of only a generic
"waiting for Pods" signal.

### User Stories

- As a cluster operator, when a StatefulSet rollout stalls, I want to know
  whether it's blocked by a Node undergoing maintenance so I don't
  mistakenly treat it as an application or storage problem.

### Notes/Constraints/Caveats

This KEP depends on Node lifecycle conditions defined in
[KEP-5683](/keps/sig-node/5683-lifecycle-conditions) being available for the
StatefulSet controller to read.

### Risks and Mitigations

TBD.

## Design Details

TBD - to be filled in as the design is worked out with SIG Apps and SIG
Node.

## Graduation Criteria

TBD.

## Implementation History

- 2026-09-27: Initial KEP created from [enhancement issue #6265](https://github.com/kubernetes/enhancements/issues/6265).
