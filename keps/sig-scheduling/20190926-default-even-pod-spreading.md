---
title: Default Even Pod Spreading
authors:
  - "@alculquicondor"
owning-sig: sig-scheduling
reviewers:
  - "@ahg-g"
  - "@Huang-Wei"
approvers:
  - "@ahg-g"
  - "@k82cn"
creation-date: 2019-09-26
last-updated: 2010-09-26
status: provisional
see-also:
  - "/keps/sig-aaa/20190221-even-pods-spreading.md"
---

# Default Even Pod Spreading

## Table of Contents

<!-- toc -->
- [Release Signoff Checklist](#release-signoff-checklist)
- [Summary](#summary)
- [Motivation](#motivation)
  - [Goals](#goals)
  - [Non-Goals](#non-goals)
- [Proposal](#proposal)
  - [User Stories](#user-stories)
    - [Story 1](#story-1)
    - [Story 2](#story-2)
  - [Implementation Details/Notes/Constraints](#implementation-detailsnotesconstraints)
  - [Risks and Mitigations](#risks-and-mitigations)
- [Design Details](#design-details)
  - [API](#api)
  - [Default rules](#default-rules)
  - [How user stories are addressed](#how-user-stories-are-addressed)
  - [Implementation Details](#implementation-details)
    - [In the metadata/predicates/priorities flow](#in-the-metadatapredicatespriorities-flow)
    - [In the scheduler framework](#in-the-scheduler-framework)
  - [Test Plan](#test-plan)
  - [Graduation Criteria](#graduation-criteria)
- [Implementation History](#implementation-history)
- [Alternatives [optional]](#alternatives-optional)
<!-- /toc -->

## Release Signoff Checklist

- [ ] kubernetes/enhancements issue in release milestone, which links to KEP (this should be a link to the KEP location in kubernetes/enhancements, not the initial KEP PR)
- [ ] KEP approvers have set the KEP status to `implementable`
- [ ] Design details are appropriately documented
- [ ] Test plan is in place, giving consideration to SIG Architecture and SIG Testing input
- [ ] Graduation criteria is in place
- [ ] "Implementation History" section is up-to-date for milestone
- [ ] User-facing documentation has been created in [kubernetes/website], for publication to [kubernetes.io]
- [ ] Supporting documentation e.g., additional design documents, links to mailing list discussions/SIG meetings, relevant PRs/issues, release notes

## Summary

With `EvenPodsSpreading`, workload authors can define spreading rules for their loads based on the
topology of the clusters. 

By introducing configurable default spreading rules, workloads can be spread in the topology of
the cluster according to opinionated rules set by a cluster operator, without having to explicitly
define them. And workloads with specific needs can still override the rules by defining them in
their PodSpec.

## Motivation

In order for a workload (pod) to use `EvenPodsSpreading`:

1. Authors have to have an idea of the underlying topology.
1. PodSpecs become less portable if their spreading rules are tailored to a specific topology.

On the other hand, cluster operators know the underlying topology of the cluster, which makes
them suitable to provide default spreading rules for all workloads in their cluster.

### Goals

- Cluster operators can define default spreading rules for pods that don't provide any.
- Workloads are spread with the default rules if they belong to the same service, controller,
replica set or stateful set,
and if they don't define `TopologySpreadConstraints`.

### Non-Goals

- Removal of `SelectorSpreadPriority`, `ServiceSpreadingPriority` or `ServiceAntiAffinity` priorities.

## Proposal

### User Stories

#### Story 1

As a cluster operator, I want to set default spreading rules for workloads in the cluster.
Currently, `SelectorSpreadPriority` provides a canned priority that spreads across nodes
and zones (`failure-domain.beta.kubernetes.io/zone`). However, the nodes in my cluster have
custom topology keys (for physical host, rack, etc.).

#### Story 2

As a workload author, I want to spread the workload in the cluster, but:
(1) I don't know the topology of the cluster I'm running on.
(2) I want to be able to run my PodSpec in different clusters (on-prem and cloud).

### Implementation Details/Notes/Constraints

Note that a priority given by default `EvenPodsSpreading` rules could conflict with
`SelectorSpreadingPriority`.
Operators can disable `SelectorSpreadingPriority`. But once default rules for `EvenPodsSpreading` is GA,
we can consider removing `SelectorSpreadingPriority` and replacing it by an equivalent
k8s default to the default rules for `EvenPodsSpreading`.

### Risks and Mitigations

`EvenPodsSpreading` has some overhead and we currently ensure that pods that don't use the
feature get minimally affected. After default rules for `EvenPodsSpreading` is rolled out,
all pods will run through the algorithms.
However, we should ensure that the running overhead is not significantly higher than
`SelectorSpreadingPriority` if using the k8s default.

## Design Details

### API

A new structure called `TopologySpreadConstraint` is introduced to `KubeSchedulerConfiguration`.

```go
type KubeSchedulerConfiguration struct {
	....
	// DefaultTopologySpreadConstraints defines spreading constraints to be applied to pods
	// that don't define any.
	// If not specified, the scheduler applies the following default.
	// +optional
	DefaultTopologySpreadConstraints []TopologySpreadConstraint
	....
}

// TopologySpreadConstraint specifies how to spread pods among the given topology.
// Pod selectors are deduced from the resource definitions that the pod belongs to
// (includes services, controllers, replica sets and stateful sets).
type TopologySpreadConstraint struct {
	MaxSkew int32
	TopologyKey string
	WhenUnsatisfiable corev1.UnsatisfiableConstraintAction
}
```

Note that `TopologySpreadConstraint` is similar to `k8s.io/api/core/v1.TopologySpreadConstraint`,
except that it doesn't define selectors.

### Default rules

These will be the default rules for the cluster when the operator doesn't provide any:

```yaml
defaultTopologySpreadConstraints:
  -
    maxSkew: 1
    topologyKey: "kubernetes.io/hostname"
    whenUnsatisfiable: ScheduleAnyway
  -
    maxSkew: 1
    topologyKey: "failure-domain.beta.kubernetes.io/zone"
    whenUnsatisfiable: ScheduleAnyway
```

### How user stories are addressed

Let's say we have a cluster that has a topology based on physical hosts and racks.
Then, an operator can set the following scheduler configuration:

```yaml
apiVersion: componentconfig/v1alpha1
defaultTopologySpreadConstraints:
  -
    maxSkew: 5
    topologyKey: "example.com/topology/physical_host"
    whenUnsatisfiable: ScheduleAnyway
  -
    maxSkew: 15
    topologyKey: "example.com/topology/rack"
    whenUnsatisfiable: DoNotSchedule
```

Then, a workload author could have the following `ReplicaSet`:

```yaml
apiVersion: apps/v1
kind: ReplicaSet
metadata:
  name: replicated_demo
spec:
  replicas: 3
  selector:
    matchLabels:
      app: demo
  template:
    metadata:
      labels:
        app: demo
    spec:
      containers:
      - name: php-redis
        image: example.com/registry/demo:latest
```

Note that the workload author didn't provide a spreading rules.
The following rules will be applied to the pods of this replica set before running the
algorithms for Even Pods Spreading:

```yaml
topologySpreadConstraints:
  -
    maxSkew: 5
    TopologyKey: "example.com/topology/physical_host"
    WhenUnsatisfiable: ScheduleAnyway
    selector:
      matchLabels:
        app: demo
  -
    maxSkew: 15
    TopologyKey: "example.com/topology/rack"
    WhenUnsatisfiable: DoNotSchedule
    selector:
      matchLabels:
        app: demo
```

These rules are internal to the scheduler and they don't get reflected in the apiserver.

### Implementation Details

#### In the metadata/predicates/priorities flow

1. Calculate the spreading constraints for the pod as part of the metadata calculation.
   Use the constraints provided by the pod or calculate the default ones if they don't provide any.
1. When running the predicates or priorities, use the constraints stored in the metadata.

#### In the scheduler framework

1. Calculate spreading constraints for the pod in the `PreFilter` extension point. Store them
   in the `PluginContext`.
1. In the `Filter` and `Score` extension points, use the stored spreading constraints instead of
   the ones defined by the pod.

### Test Plan

To ensure this feature to be rolled out in high quality. Following tests are mandatory:

- **Unit Tests**: All core changes must be covered by unit tests.
- **Integration Tests**: One integration test for the default rules and one for custom rules.
- **Benchmark Tests**: A benchmark test that compare the default rules against `SelectorSpreadingPriority`.
  The performance should be as close as possible.

### Graduation Criteria

Alpha (v1.17):

[ ] Scheduler Component Config API changes.
[ ] Default, validation and generated code.
[ ] Priority Implementation.
[ ] Predicate implementation.
[ ] Test cases mentioned in the [Test Plan](#test-plan).

## Implementation History

- 2019-09-26: Initial KEP sent out for review.

## Alternatives [optional]

- Make the topology keys used in `SelectorSpreadingPriority` configurable.

    While this moves the scheduler in the right direction, there are two problems:
    
    1. We can only support one topology key.
    1. It makes it hard for pods to override the operator-provided spreading rules.