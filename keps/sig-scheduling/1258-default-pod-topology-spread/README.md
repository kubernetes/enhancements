# Default Pod Topology Spread

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
    - [Feature gate](#feature-gate)
    - [Relationship with &quot;SelectorSpread&quot; plugin](#relationship-with-selectorspread-plugin)
  - [Risks and Mitigations](#risks-and-mitigations)
- [Design Details](#design-details)
  - [API](#api)
  - [Default constraints](#default-constraints)
  - [How user stories are addressed](#how-user-stories-are-addressed)
  - [Implementation Details](#implementation-details)
  - [Test Plan](#test-plan)
  - [Graduation Criteria](#graduation-criteria)
    - [Alpha (v1.19):](#alpha-v119)
- [Implementation History](#implementation-history)
- [Alternatives](#alternatives)
<!-- /toc -->

## Release Signoff Checklist

- [x] kubernetes/enhancements issue in release milestone, which links to KEP (this should be a link to the KEP location in kubernetes/enhancements, not the initial KEP PR)
- [x] KEP approvers have set the KEP status to `implementable`
- [x] Design details are appropriately documented
- [ ] Test plan is in place, giving consideration to SIG Architecture and SIG Testing input
- [ ] Graduation criteria is in place
- [ ] "Implementation History" section is up-to-date for milestone
- [x] User-facing documentation has been created in [kubernetes/website], for publication to [kubernetes.io]
- [x] Supporting documentation e.g., additional design documents, links to mailing list discussions/SIG meetings, relevant PRs/issues, release notes

## Summary

With [Pod Topology Spread](/keps/sig-scheduling/895-pod-topology-spread),
workload authors can define spreading rules for their loads based on the topology of the clusters. 
The spreading rules are defined in the `PodSpec`, thus they are tied to the pod.

We propose the introduction of configurable default spreading constraints, i.e. constraints that
can be defined at the cluster level and are applied to pods that don't explicitly define spreading constraints.
This way, all pods can be spread according to (likely better informed) constraints set by a cluster operator.
Workload authors don't need to know the topology of the cluster they will be running on to have their pods spread.
But if they do, they can still set their own spreading constraints if they have specific needs.

## Motivation

In order for a workload (pod) to use `.spec.topologySpreadConstraints` (known as`PodTopologySpread`
plugin or `EvenPodsSpreadPriority` in the old Policy API):

1. Authors have to have an idea of the underlying topology.
1. PodSpecs become less portable if their spreading constraints are tailored to a specific topology.

On the other hand, cluster operators know the underlying topology of the cluster, which makes
them suitable to provide default spreading constraints for all workloads in their cluster.

### Goals

- Cluster operators can define default spreading constraints for pods that don't provide any
  `pod.spec.topologySpreadConstraints`.
- Workloads are spread with the default constraints if they belong to the same service, replication controller,
  replica set or stateful set, and if they don't define `pod.spec.topologySpreadConstraints`.
- Provide a k8s default for `topologySpreadConstraints` that produces a priority equivalent to
  `DefaultPodTopologySpread`, so that this plugin can be deprecated in the future.

### Non-Goals

- Set defaults for specific namespaces or according to other selectors.
- Removal of `SelectorSpread` plugin.

## Proposal

### User Stories

#### Story 1

As a cluster operator, I want to set default spreading constraints for workloads in the cluster.
Currently, `SelectorSpread` plugin provides a canned scoring that spreads across nodes
and zones (`topology.kubernetes.io/zone`). However, the nodes in my cluster have custom topology
keys (for physical host, rack, etc.).

#### Story 2

As a workload author, I want to spread the workload in the cluster, but:
(1) I don't know the topology of the cluster I'm running on.
(2) I want to be able to run my PodSpec in different clusters (on-prem and cloud).

### Implementation Details/Notes/Constraints


#### Feature gate

Setting a default for `PodTopologySpread` will be guarded with the feature gate
`DefaultPodTopologySpread`.

#### Relationship with "SelectorSpread" plugin

Note that Default `topologySpreadConstraints` has a similar effect to `SelectorSpread`
plugin (`SelectorSpreadingPriority` when using the Policy API).
Given that the latter is not configurable, they could return conflicting priorities, which
may not be the intention of the cluster operator or workload author. On the other hand, a proper
default for `topologySpreadConstraints` can provide the same score as
`SelectorSpread`. Thus, there's no need for the features to co-exist.

When the feature gate is enabled:

- K8s will set Default `topologySpreadConstraints` and remove `SelectorSpread` from the
k8s providers (`DefaultProvider` and `ClusterAutoscalerProvider`). The
[Default constraints](#default-constraints) will produce a similar score.
- When setting plugins in the Component Config API, operators can specify plugins they want to enable.
  Since this is a manual operation, if an operator decides to enable both plugins, this is respected.
- [Beta] When using the Policy API, `SelectorSpreadingPriority` will map to `PodTopologySpread`.

### Risks and Mitigations

The `PodTopologySpread` plugin has some overhead compared to other plugins. We currently ensure that
pods that don't use the feature get minimally affected. After Default `topologySpreadConstraints`
is rolled out, all pods will run through the plugin.
We should ensure that the running overhead is not significantly higher than
`SelectorSpread` with the k8s Default.

## Design Details

### API

A new structure `PodTopologySpreadArgs` is introduced in `pkg/scheduler/apis/config/`.
Values are decoded from the `pluginConfig` slice in the kube-scheduler Component Config and used in
`podtopologyspread.New`.

```go
// pkg/scheduler/apis/config/types_pluginargs.go
type PodTopologySpreadArgs struct {
	// DefaultConstraints defines topology spread constraints to be applied to pods
	// that don't define any in `pod.spec.topologySpreadConstraints`. Pod selectors must
	// be empty, as they are deduced from the resources that the pod belongs to
	// (includes services, replication controllers, replica sets and stateful sets). 
	// If not specified, the scheduler applies the following default constraints:
	// <default rules go here. See next section>
	// +optional
	DefaultConstraints []corev1.TopologySpreadConstraint
}
```

Note the use of `k8s.io/api/core/v1.TopologySpreadConstraint`. During validation, we verify that
selectors are not defined.

### Default constraints

These will be the default constraints for the cluster when the operator doesn't provide any:

```yaml
defaultConstraints:
  - maxSkew: 3
    topologyKey: "kubernetes.io/hostname"
    whenUnsatisfiable: ScheduleAnyway
  - maxSkew: 5
    topologyKey: "topology.kubernetes.io/zone"
    whenUnsatisfiable: ScheduleAnyway
```

An operator can choose to disable the default constraints using:

```yaml
defaultConstraints: []
```

### How user stories are addressed

Let's say we have a cluster that has a topology based on physical hosts and racks.
Then, an operator can set the following configuration for the plugin:

```yaml
defaultConstraints:
  - maxSkew: 5
    topologyKey: "example.com/topology/physical_host"
    whenUnsatisfiable: ScheduleAnyway
  - maxSkew: 15
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

Note that the workload author didn't provide spreading constraints in the `pod.spec`.
The following spreading constraints will be derived from the constraints defined in ComponentConfig,
and will be applied at runtime:

```yaml
topologySpreadConstraints:
  - maxSkew: 5
    topologyKey: "example.com/topology/physical_host"
    whenUnsatisfiable: ScheduleAnyway
    selector:
      matchLabels:
        app: demo
  - maxSkew: 15
    topologyKey: "example.com/topology/rack"
    whenUnsatisfiable: DoNotSchedule
    selector:
      matchLabels:
        app: demo
```

Please note that these constraints get applied internally in the scheduler, but they are NOT
persisted in the PodSpec via API Server.

### Implementation Details

1. Calculate spreading constraints for the pod in the `PreFilter` extension point. Store them
   in the `PluginContext`. The constraints are obtained from `.spec.topologySpreadConstraints`. If
   they are not defined, a default is calculated from the plugin's default constraints, using the
   selectors of the Services, ReplicaSets, StatefulSets or ReplicationControllers the pod belongs to.
1. In the `Filter` and `Score` extension points, use the stored spreading constraints instead of
   the ones defined by the pod.

### Test Plan

To ensure this feature to be rolled out in high quality. Following tests are mandatory:

- **Unit Tests**: All core changes must be covered by unit tests.
- **Integration Tests**: One integration test for the default rules and one for custom rules.
- **Benchmark Tests**: A benchmark test that compare the default rules against `SelectorSpreadingPriority`.
  The performance should be as close as possible.

### Graduation Criteria

#### Alpha (v1.19):

- [x] Args struct for `podtopologyspread.New`.
- [x] Defaults and validation.
- [x] Score extension point implementation. Add support for `maxSkew`.
- [x] Filter extension point implementation.
- [x] Disabling `SelectorSpread` when the feature is enabled.
- [x] Unit, Integration and benchmark test cases mentioned in the [Test Plan](#test-plan).

## Implementation History

- 2019-09-26: Initial KEP sent out for review.
- 2020-01-20: KEP updated to make use of framework's PluginConfig.
- 2020-05-04: Update completed tasks and target alpha for 1.19.

## Alternatives

- Make the topology keys used in `SelectorSpread` configurable.

    While this moves the scheduler in the right direction, there are two problems:
    
    1. We can only support one topology key.
    1. It makes it hard for pods to override the operator-provided spreading rules.

- Implement a mutating controller that sets defaults.

  This approach would likely allow us to provide a more flexible interface that
  can set defaults for specific namespaces or with other selectors. However, that
  wouldn't allow us to replace `SelectorSpread` with `PodTopologySpread`.