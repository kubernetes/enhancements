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
  - [This test](https://github.com/kubernetes/kubernetes/blob/master/pkg/kubectl/generate/versioned/pdb_test.go)
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

