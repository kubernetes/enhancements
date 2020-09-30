# KEP-1845: Prioritization on volume capacity

<!-- toc -->
- [Release Signoff Checklist](#release-signoff-checklist)
- [Summary](#summary)
- [Motivation](#motivation)
  - [Goals](#goals)
  - [Non-Goals](#non-goals)
- [Proposal](#proposal)
  - [User Stories](#user-stories)
    - [Volumes with close capacity are preferred](#volumes-with-close-capacity-are-preferred)
    - [Different weights for different classes](#different-weights-for-different-classes)
- [Design Details](#design-details)
  - [Configuring the utilization shape points](#configuring-the-utilization-shape-points)
  - [Configuring the weight of storage class](#configuring-the-weight-of-storage-class)
  - [Test Plan](#test-plan)
  - [Graduation Criteria](#graduation-criteria)
    - [Alpha](#alpha)
    - [Beta](#beta)
    - [GA](#ga)
- [Alternatives](#alternatives)
  - [Maintain multiple storage classes for different storage capacities](#maintain-multiple-storage-classes-for-different-storage-capacities)
- [Production Readiness Review Questionnaire](#production-readiness-review-questionnaire)
  - [Feature enablement and rollback](#feature-enablement-and-rollback)
  - [Rollout, Upgrade and Rollback Planning](#rollout-upgrade-and-rollback-planning)
  - [Monitoring requirements](#monitoring-requirements)
  - [Dependencies](#dependencies)
  - [Scalability](#scalability)
  - [Troubleshooting](#troubleshooting)
- [Implementation History](#implementation-history)
- [Drawbacks](#drawbacks)
<!-- /toc -->

## Release Signoff Checklist

Items marked with (R) are required *prior to targeting to a milestone / release*.

- [x] (R) Enhancement issue in release milestone, which links to KEP dir in [kubernetes/enhancements] (not the initial KEP PR)
- [ ] (R) KEP approvers have approved the KEP status as `implementable`
- [ ] (R) Design details are appropriately documented
- [ ] (R) Test plan is in place, giving consideration to SIG Architecture and SIG Testing input
- [ ] (R) Graduation criteria is in place
- [ ] (R) Production readiness review completed
- [ ] Production readiness review approved
- [ ] "Implementation History" section is up-to-date for milestone
- [ ] User-facing documentation has been created in [kubernetes/website], for publication to [kubernetes.io]
- [ ] Supporting documentation e.g., additional design documents, links to mailing list discussions/SIG meetings, relevant PRs/issues, release notes

[kubernetes.io]: https://kubernetes.io/
[kubernetes/enhancements]: https://git.k8s.io/enhancements
[kubernetes/kubernetes]: https://git.k8s.io/kubernetes
[kubernetes/website]: https://git.k8s.io/website

## Summary

With [volume topology-aware
scheduling](https://github.com/kubernetes/community/blob/master/contributors/design-proposals/storage/volume-topology-scheduling.md) in place,
the scheduler filters nodes considering volume topology constraints. However, scheduler picks
the smallest matching volumes in each topology but does not try to prioritize
nodes in different topologies based on volume capacity. This leads to a waste
of resources which we should avoid.

This KEP continues the work proposed [here](https://github.com/kubernetes/community/blob/master/contributors/design-proposals/storage/volume-topology-scheduling.md#priority), to enable the scheduler to favor the node which has the most fit PV.

## Motivation

Because we didn't take volume capacity into account when scheduling a pod that
can run in multiple topologies. A large PV may be used by a PVC with a small
capacity request, even if there are many suitable small PVs in other
topologies. PVCs with a large capacity request may not find feasible PVs to use
if too many large PVs are consumed by PVCs with a small capacity request.

Like CPU, Memory resources, the scheduler should take volume capacity into
account in scheduling pods to ensure balanced resource usage.

### Goals

- Prioritizing nodes based on the best matching size of statically provisioned
  PVs
- Support arbitrary PV topology domain (e.g. hostname, zone, rack)

### Non-Goals

- Prioritizing nodes based on the total available capacity of statically provisioned
  PVs
- Prioritizing nodes based on the total available number of statically provisioned
  PVs
- Prioritizing nodes based on the total available capacity for dynamic
  provisioning PVCs

## Proposal

### User Stories

#### Volumes with close capacity are preferred

As a cluster administrator, I need to prepare some local PVs with different
storage capacities (e.g. 10Gi, 100Gi, 1Ti) as different workloads may have
different storage requirements.

A PV with 1Ti capacity on one node has the same chance as a PV on another node
with 10Gi capacity to be bound with a 10Gi request PVC in scheduling. If nodes
of topology in which volumes with close capacity are always preferred, we can
achieve better storage resource usage.

See https://github.com/kubernetes/kubernetes/issues/83323.

#### Different weights for different classes

Given a cluster which has two kinds of local storage classes:

- hdd
- ssd

Workloads may request two disks, one from `hdd` and one from `ssd`. As
the ssd storage resource may be relatively scarce, the administrator can
configure a relatively larger weight for storage class `ssd` and hope it's more
balanced.

This applies to any other storage with topology constraints.

## Design Details

We have a scheduler plugin called
[VolumeBinding](https://github.com/kubernetes/kubernetes/tree/master/pkg/scheduler/framework/plugins/volumebinding)
to filter the nodes based on volume topology constraints. After the filter
phase, there is a reduced set of nodes that can fit a pod and filtered smallest
statically provisioned PV(s) in each node topology.

We can add  `Score` interfaces to score nodes based on pod's unbound PVCs and
filtered PVs.

The score is calculated in two steps:

1) First, we group pod's unbound PVCs and corresponding statically provisioned PVs
by storage class. For PVCs/PVs in each storage class, we calculate requested to
capacity ratio using the following formula:

```
[UtilizationPerClass] = [MaxUtilization] * ([RequestPerClass] / [CapacityPerClass])
```

The utilization ranges from 0 to 100.

We can tune this formula by applying utilization shape points, which is used in
[`RequestedToCapacityRatio`](https://github.com/kubernetes/kubernetes/blob/master/pkg/scheduler/framework/plugins/noderesources/requested_to_capacity_ratio.go).

By default, we use this shape points to get linear scores between [0,
100] utilization.

```
- utilization: 0
  score: 0
- utilization: 100
  score: config.MaxCustomPriorityScore
```

The score ranges from 0 to `MaxNodeScore` (100).

Optionally, users can configure customized utilization shape points, e.g.

```
- utilization: 50
  score: 0
- utilization: 80
  score: 3
- utilization: 100
  score: 5
```

The score of utilization below 50 will be 0, and the score of utilization of 90
will be 4 (`3 + (5 - 3) * (90 - 80) / (100 - 80)`).

2) Then, we add them together and divides by the number of storage classes.

### Configuring the utilization shape points

We can add a field `Shape` to specify the shape points which defines the score
function shape.

```
// VolumeBindingArgs holds arguments used to configure the VolumeBinding plugin.
type VolumeBindingArgs struct {
		...
    // Shape specifies the shape points defining score function shape
    Shape []UtilizationShapePoint
}
```

The definition of `UtilizationShapePoint` is [here](https://github.com/kubernetes/kubernetes/blob/73fa63a86d1d47d82b435ca85c2f350da06b08b9/pkg/scheduler/apis/config/types_pluginargs.go#L115-L121).

### Configuring the weight of storage class

Optionally, we can allow users to configure the weight of storage class. If
weights of storage classes are evolved, the formula will be:

```
[NodeScore] = Sum([ScorePerClass] * [WeightPerClass]) / Sum([WeightPerClass])
```

In the initial stage, we treat all storage classes equal to avoid introducing
complexity to Kubernetes. If there are real use cases in Beta/GA, we can
consider supporting it.

```
<<[UNRESOLVED]>>

Possible solutions:

1) annotation in storage class object

We can read the weight from the storage class annotation, e.g.

```
storageclass.kubernetes.io/weight: "5"
```

Pros:

- No need to update the configuration of the kube-scheduler
- The change can be applied on the fly

Cons:

- Weights are configured in different objects

2) configure in plugin args:

```
storageClassWeights:
- local-ssd: "5"
- local-hdd: "3"
```

Pros:

- Weights are configured in one place

Cons:

- Need to update and reload the configuration of the kube-scheduler

<<[/UNRESOLVED]>>
```

### Test Plan

- **Unit Tests:** All code will be covered by unit tests.
- **Integration Tests** Typical user cases will be covered in scheduler
  integration tests with this feature enabled.
- **Benchmark Tests** Add benchmarking tests for VolumeBinding with this
  feature enabled. (This is required in Beta and GA)

### Graduation Criteria

#### Alpha

Target: v1.20

- [ ] Add `VolumeCapacityPriority` feature gate
- [ ] Add priority extension point implementation for VolumeBinding plugin
- [ ] Able to prioritizing nodes based on the best matching size of statically
  provisioned PVs
- [ ] Arg struct for `VolumeBindingArgs.Shape`
- [ ] Tests for basic functionalities

#### Beta

Target: v1.21

- [ ] Add benchmarking tests
- [ ] Turn algorithm based on feedback from developers and users
- [ ] Able to configure the weight of storage class (implementation TBD)

#### GA

Target: TBD

- [ ] Code is thoroughly tested
- TBD

## Alternatives

### Maintain multiple storage classes for different storage capacities

System administrator can create multiple storage classes for different storage
capacities, e.g.

- local-storage-1G
- local-storage-10G
- local-storage-100G
- local-storage-1T
- ...

Then workloads can select best fit storage by specifying the desired storage
class in PVCs.

However, this is not flexible as it's a hard scheduling requirement. Cluster
storage are split into pieces, large PVs cannot be consumed by pods that has a
small storage class configured in PVCs.

## Production Readiness Review Questionnaire

### Feature enablement and rollback

* **How can this feature be enabled / disabled in a live cluster?**
  - [x] Feature gate (also fill in values in `kep.yaml`)
    - Feature gate name: VolumeCapacityPriority
    - Components depending on the feature gate: kube-scheduler

* **Does enabling the feature change any default behavior?**

  Basically not, but scheduler may make different decision for the same soft
  (preferred) scheduling requirements, because it takes volume capacity into
  account in scoring nodes.

* **Can the feature be disabled once it has been enabled (i.e. can we rollback
  the enablement)?**

  The feature can be disabled in Alpha and Beta versions. When it's graduated to
  GA, this feature will be enabled by default.

* **What happens if we reenable the feature if it was previously rolled back?**

  N/A.

* **Are there any tests for feature enablement/disablement?**

  Unit tests with and without the feature are necessary to verify we can achieve
  better resource usage with this feature.

### Rollout, Upgrade and Rollback Planning

* **How can a rollout fail? Can it impact already running workloads?**

  When the feature gate is enabled on the kube-scheduler, all unscheduled and
  future pods which reference delay binding PVCs will be applied this priority
  policy.

  This does not impact running workloads.

* **What specific metrics should inform a rollback?**

  - A spike on metric `schedule_attempts_total{result="error|unschedulable"}`
    when this feature gate is enabled.

* **Were upgrade and rollback tested? Was upgrade->downgrade->upgrade path tested?**

  N/A.

* **Is the rollout accompanied by any deprecations and/or removals of features,
  APIs, fields of API types, flags, etc.?**

  No.

### Monitoring requirements

* **How can an operator determine if the feature is in use by workloads?**

  If enabled, this feature applies to all workloads which uses delay binding
  PVCs. Also non-zero value of metric
  `plugin_execution_duration_seconds{plugin="VolumeBinding",extension_point="Score"}`
  is a sign indicating this feature is in use.

* **What are the SLIs (Service Level Indicators) an operator can use to
  determine the health of the service?**

  This feature is implemented in scheduler VolumeBinding plugin. Metric
  `plugin_execution_duration_seconds{plugin="VolumeBinding",extension_point=~"(Score|ScoreExtensionNormalize)"}`
  can used to to indicate the scheduling latency for a pod using this feature.

* **What are the reasonable SLOs (Service Level Objectives) for the above SLIs?**

  - Metric `plugin_execution_duration_seconds{plugin="VolumeBinding",extension_point=~"(Score|ScoreExtensionNormalize)"}`
    <= 100ms on 90-percentile.

* **Are there any missing metrics that would be useful to have to improve
  observability if this feature?**

  N/A.


### Dependencies

* **Does this feature depend on any specific services running in the cluster?**

  No.

### Scalability

* **Will enabling / using this feature result in any new API calls?**

  No

* **Will enabling / using this feature result in introducing new API types?**

  No.

* **Will enabling / using this feature result in any new calls to cloud
  provider?**

  No.

* **Will enabling / using this feature result in increasing size or count
  of the existing API objects?**

  No.

* **Will enabling / using this feature result in increasing time taken by any
  operations covered by [existing SLIs/SLOs][]?**

  This feature needs additional computation, so it's expected to see an
  increased latency on
  `plugin_execution_duration_seconds{plugin="VolumeBinding"}` - comparing to
  other plugin latency. But workloads which does not use delay binding PVCs
  won't get penalties.

* **Will enabling / using this feature result in non-negligible increase of
  resource usage (CPU, RAM, disk, IO, ...) in any components?**

  No.

### Troubleshooting

* **How does this feature react if the API server and/or etcd is unavailable?**

  Running pods won't be impacted. Unscheduled pods using this feature will
  not be scheduled as the API server and/or etcd is unavailable.

* **What are other known failure modes?**

  N/A.

* **What steps should be taken if SLOs are not being met to determine the problem?**

  N/A.

## Implementation History

- 2020-06-04 Initial KEP sent out for review

## Drawbacks

- Users can create PVs with fake capacity or sharing capacity of a filesystem
  volume (e.g. multiple local PVs sharing a filesystem by creating multiple
  subdirectories). In these scenarios, the calculated scores can't reflect the
  actual situation.
