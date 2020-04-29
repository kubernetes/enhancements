---
title: graduate-pod-disruption-budget-to-stable
authors:
  - "@bsalamat"
owning-sig: sig-apps
participating-sigs:
  - sig-scheduling
reviewers:
  - "@liggitt"
  - "@kow3ns"
approvers:
  - "@kow3ns"
  - "@liggitt"
editor: TBD
creation-date: 2019-03-18
last-updated: 2019-03-18
status: implementable
see-also:
  - 
replaces:
superseded-by:
---

# Graduate PodDisruptionBudget to stable

## Table of Contents

<!-- toc -->
- [Summary](#summary)
- [Motivation](#motivation)
  - [Goals](#goals)
  - [Non-Goals](#non-goals)
- [Proposal](#proposal)
  - [Implementation Details/Notes/Constraints](#implementation-detailsnotesconstraints)
    - [Mutable PDBs](#mutable-pdbs)
    - [Eviction of non-ready pods](#eviction-of-non-ready-pods)
    - [Make the disruption controller more lenient for pods belonging to non-scale controllers](#make-the-disruption-controller-more-lenient-for-pods-belonging-to-non-scale-controllers)
  - [Risks and Mitigations](#risks-and-mitigations)
  - [Test Plan](#test-plan)
    - [Existing Tests](#existing-tests)
    - [Needed Tests](#needed-tests)
  - [Graduation Criteria](#graduation-criteria)
- [Implementation History](#implementation-history)
<!-- /toc -->

## Summary

[Pod Disruption Budget (PDB)](https://kubernetes.io/docs/tasks/run-application/configure-pdb/)
is a Kubernetes API that limits the number of pods of a collection that are down simultaneously from voluntary disruptions.
[Kubernetes eviction API](https://kubernetes.io/docs/tasks/administer-cluster/safely-drain-node/#the-eviction-api)
takes PDB into account when terminating pods. If PDB is
violated, the eviction API returns failure and does not delete the requested pod.
The feature was introduced in Kubernetes 1.4 and promoted to beta in 1.5.
It has been in beta for a long time. This document lays out the plan to promote
it to stable.

## Motivation

PDB API has been stable and is an important feature that allows users to improve
reliability of their critical workloads. This feature has been in beta for a
long time. We need to promote to stable version given that we plan to support it
long term.

### Goals

* Plan to promote PDB API to stable version.
* List action items to mitigate potential risks of supporting mutable PDB.

### Non-Goals

* Making changes to the API fields or their meanings.

## Proposal

### Implementation Details/Notes/Constraints

#### Mutable PDBs

A mutable PDB object allows its `MinAvailable`, `MaxUnavailable`, and `Selector`
fields to be modified by clients. Components that use PDB must watch such
modifications and use the updated values when making decisions.

This feature is implemented by [this PR](https://github.com/kubernetes/kubernetes/pull/69867).

#### Eviction of non-ready pods

There are a couple of open issues where pods can't be evicted, even if they not Ready and Running 
(https://github.com/kubernetes/kubernetes/issues/72320 and https://github.com/kubernetes/kubernetes/issues/80389).
The root of this issue, is that the rules in the disruption controller for what is a healthy pod, and the rules
in the Eviction API for when a pod can be evicted without looking at the PDB are not the same. This means a pod can
be considered unhealthy by the disruption controller so it does not count as healthy when computing `DisruptionsAllowed`,
but will still require `DisruptionsAllowed` to be larger than 0 for it to be evicted. Some strange situations can
arise from this. For example if we have a PDB with `MinAvailable  = 1` and 10 pods that are all in the CrashLoop state 
(`Running`, but not `Ready`), we will not be allowed to evict any of the pods.

#### Make the disruption controller more lenient for pods belonging to non-scale controllers

The disruption controller is currently taking the very safe route whenever it encounters any
issues with the targeted pods or their controllers. For all configurations of the PDB, except when
`minAvailable` is a number, the PDB requires that it can find the controller and that
the controller implements scale (either by being one of the built-in workloads or a CR where the CRD implements
the scale subresource). If those conditions are not met for all pods targeted by the PDB, the disruption
controller will set `DisruptionsAllowed` to 0, which means none of the pods can be evicted. There is an issue
concerning this behavior: https://github.com/kubernetes/kubernetes/issues/77383.

The current behavior of the disruption controller for the different types of input and the different
types of pods that might be encountered are documented in: 
https://docs.google.com/spreadsheets/d/12HUundBS-slA6axfQYZPRCeIu_Au_wsGD0Vu_oKAnM8/edit?usp=sharing

### Risks and Mitigations

We plan to support mutation of PDB objects that didn't exist in previous versions.
The following needs to be checked and verified before graduation of the API.

- [ ] Ensure that components do not have any logic that relies on immutability
of PDBs. For example, if component builds a cache of pods that match various
PDBs, they must add logic to invalid the cache on updates.
   - Action Item: sweep for in-tree (including kubernetes/* repos) uses to make
   sure components are informer driven and aren’t assuming immutability.
- [ ] Check what watches PDBs and make sure there is no performance concerns.
- [ ] Updating a PDB could cause the state of a cluster to seem incorrect. For
example, a PDB states that at least 10 replicas of a collection must be alive.
Kubernetes control plane evicts some of the existing pods, but keeps at least 10
around. The PDB is updated and states that at least 20 replicas must be kept
alive. It may appear to an observer that the evictions happened before the PDB 
update were incorrect, if they don’t notice the PDB update.
  - Action Item: Update documents and explain mutability of PDB its side effects.

### Test Plan

#### Existing Tests
PodDisruptionBudget currently has tests in various components that use the feature:

* Disruption controller
  - [This test](https://github.com/kubernetes/kubernetes/blob/687d759e362b05dcdf11e336e2799704918e048d/pkg/controller/disruption/disruption_test.go#L140)
  tests PDB MinAvailable, MaxUnavailable, and selector functionality in Disruption controller.
* Kubectl
  - [This test](https://github.com/kubernetes/kubernetes/blob/feature-serverside-apply/pkg/kubectl/generate/versioned/pdb_test.go)
  tests generation of a PDB objects out of given parameters
  - [This test](https://github.com/kubernetes/kubernetes/blob/master/pkg/kubectl/cmd/create/create_pdb_test.go)
  tests creation of PDB objects from cmd parameters
* Scheduler
  - [This test](https://github.com/kubernetes/kubernetes/blob/ac56bd502ab96696682c66ebdff94b6e52471aa3/test/integration/scheduler/preemption_test.go#L731)
  tests effects of PDB on preemption (PDB is honored in a best effort way)
* Eviction integration tests
  - [These tests](https://github.com/kubernetes/kubernetes/blob/master/test/integration/evictions/evictions_test.go) test eviction logic and its interactions with PDB.
* Autoscaler
  - [These tests](https://github.com/kubernetes/kubernetes/blob/master/test/e2e/autoscaling/cluster_size_autoscaling.go) ensure that Autoscaler respects PDB when draining nodes.

#### Needed Tests

As a non-optional feature, there should be a conformance test for
PodDisruptionBudget.


### Graduation Criteria

- [ ] Implement Mutable PDBs
- [ ] Needs a conformance test
- [ ] Update documents to reflect the changes

## Implementation History

- PodDisruptionBudget was introduced in Kubernetes 1.4 as an alpha version.
- PodDisruptionBudget was graduated to beta in Kubernetes 1.5.

