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
    - [Beta (v1.20):](#beta-v120)
    - [Stable (v1.24):](#stable-v124)
- [Production Readiness Review Questionnaire](#production-readiness-review-questionnaire)
  - [Feature Enablement and Rollback](#feature-enablement-and-rollback)
  - [Rollout, Upgrade and Rollback Planning](#rollout-upgrade-and-rollback-planning)
  - [Monitoring Requirements](#monitoring-requirements)
  - [Dependencies](#dependencies)
  - [Scalability](#scalability)
  - [Troubleshooting](#troubleshooting)
- [Implementation History](#implementation-history)
- [Alternatives](#alternatives)
<!-- /toc -->

## Release Signoff Checklist

- [x] kubernetes/enhancements issue in release milestone, which links to KEP (this should be a link to the KEP location in kubernetes/enhancements, not the initial KEP PR)
- [x] KEP approvers have set the KEP status to `implementable`
- [x] Design details are appropriately documented
- [x] Test plan is in place, giving consideration to SIG Architecture and SIG Testing input
- [x] Graduation criteria is in place
- [x] "Implementation History" section is up-to-date for milestone
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

Setting a default for `PodTopologySpread` is guarded with the feature gate
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
  // be empty, as they are deduced from pod's membership
  // to Services, ReplicationControllers, ReplicaSets or StatefulSets.
  // If empty, the default constraints prefer to spread Pods across Nodes and Zones.
  DefaultConstraints []corev1.TopologySpreadConstraint
  // DisableDefaultConstraints allows to disable DefaultConstraints. Defaults to false.
  // When set to true, DefaultConstraints must be empty or nil.
  // +optional
  DisableDefaultConstraints bool
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
  [Beta] There should not be any significant degradation in scheduler performance in clusterloader benchmarks
  for vanilla workloads.
- **E2E/Conformance Tests**: Test "Multi-AZ Clusters should spread the pods of a {replication controller, service} across zones" should pass.
  This test is currently broken in 5k nodes.

### Graduation Criteria

#### Alpha (v1.19):

- [x] Args struct for `podtopologyspread.New`.
- [x] Defaults and validation.
- [x] Score extension point implementation. Add support for `maxSkew`.
- [x] Filter extension point implementation.
- [x] Disabling `SelectorSpread` when the feature is enabled.
- [x] Unit and benchmark test cases mentioned in the [Test Plan](#test-plan).

#### Beta (v1.20):

- [X] Finalize implementation:
  - [X] Map `SelectorSpreadingPriority` to `PodTopologySpread` when using Policy API.
  - [X] Provide knob for disabling the k8s default constraints.
- [X] Integration tests.
- [X] Verify conformance tests passing.

#### Stable (v1.24):

- [X] No negative feedback.
  - Issue [#102136](https://github.com/kubernetes/kubernetes/issues/102136) has been fixed and backported.
- [X] [Integration test](https://k8s-testgrid.appspot.com/presubmits-kubernetes-blocking#pull-kubernetes-integration&include-filter-by-regex=TestDefaultPodTopologySpread).
- [X] [E2E test]( https://testgrid.k8s.io/sig-scheduling#sig-scheduling-kind,%20multizone&include-filter-by-regex=Multi-AZ)

## Production Readiness Review Questionnaire

### Feature Enablement and Rollback

* **How can this feature be enabled / disabled in a live cluster?**
  - [x] Feature gate (also fill in values in `kep.yaml`)
    - Feature gate name: `DefaultPodTopologySpread`
    - Components depending on the feature gate: `kube-scheduler`
  - [x] Other
    - Describe the mechanism:

      Explicitly disable default spreading constraints for the `PodTopologySpread` plugin in the kube-scheduler config (passed via `--config` command line flag):

      ```yaml
      apiVersion: kubescheduler.config.k8s.io/v1beta1
      kind: KubeSchedulerConfiguration
      profiles:
        - pluginConfig:
          - name: PodTopologySpread
            args:
              disableDefaultConstraints: true
      ```

    - Will enabling / disabling the feature require downtime of the control
      plane?

      Only kube-scheduler needs to be restarted.

    - Will enabling / disabling the feature require downtime or reprovisioning
      of a node? (Do not assume `Dynamic Kubelet Config` feature is enabled).

      No

* **Does enabling the feature change any default behavior?**

  Yes. Users might experience more spreading of Pods among Nodes and Zones in certain topology distributions.
  In particular, this will be more noticeable in clusters with more than 100 nodes.

  The [default configuration](#default-constraints) was chosen to produce a behavior that closely resembles
  the `SelectorSpread` plugin.
  See [this PR description](https://github.com/kubernetes/kubernetes/pull/91793) for simulation data.

* **Can the feature be disabled once it has been enabled (i.e. can we roll back
  the enablement)?**

  Yes. Once disabled, only scheduling of new Pods will be affected.

* **What happens if we reenable the feature if it was previously rolled back?**

  Only scheduling of new Pods is affected.

* **Are there any tests for feature enablement/disablement?**

  There are unit tests in `pkg/scheduler/algorithmprovider/registry_test.go` that validate the list of default plugins
  of `kube-scheduler` that correspond to the Feature Gate enabled and disabled.

### Rollout, Upgrade and Rollback Planning

* **How can a rollout fail? Can it impact already running workloads?**

  Running workloads are not affected by `kube-scheduler`.

* **What specific metrics should inform a rollback?**

  Primarily scheduling latency metrics, such as `framework_extension_point_duration_seconds`, `scheduling_algorithm_duration_seconds`
  and `e2e_scheduling_duration_seconds`, when they have increased significantly.

  Since spreading is affected, node utilization might change.
  Utilization metrics can be queried in the `/metrics/resource` endpoint exposed by kubelet.

* **Were upgrade and rollback tested? Was the upgrade->downgrade->upgrade path tested?**

  N/A.

* **Is the rollout accompanied by any deprecations and/or removals of features, APIs, 
  fields of API types, flags, etc.?**

  No.

### Monitoring Requirements

* **How can an operator determine if the feature is in use by workloads?**

  All Pods are affected, unless they have explicit spreading constraints (.spec.topologySpreadConstraints).

* **How can someone using this feature know that it is working for their instance?**

  - [ ] Events
    - Event Reason: 
  - [ ] API .status
    - Condition name: 
    - Other field: 
  - [X] Other (treat as last resort)
    - Details: observe the scheduled pods and verify the spreading is satisfied.

* **What are the SLIs (Service Level Indicators) an operator can use to determine 
  the health of the service?**

  - [x] Metrics
    - Metric name: `framework_extension_point_duration_seconds` with label `extension_point` values `PreScore` and/or `Score`.
    - [Optional] Aggregation method:
    - Components exposing the metric: `kube_scheduler`.
  - [ ] Other (treat as last resort)
    - Details:

* **What are the reasonable SLOs (Service Level Objectives) for the above SLIs?**

  For 100 nodes, with a 4-core master:

  - Latency for PreScore+Score less than 60ms for 99% percentile.
  - Latency for PreScore+Score less than 15ms for 95% percentile.

* **Are there any missing metrics that would be useful to have to improve observability 
of this feature?**

  N/A.

### Dependencies

* **Does this feature depend on any specific services running in the cluster?**

  N/A.


### Scalability

* **Will enabling / using this feature result in any new API calls?**

  No.

* **Will enabling / using this feature result in introducing new API types?**

  No.

* **Will enabling / using this feature result in any new calls to the cloud 
provider?**

  No.

* **Will enabling / using this feature result in increasing size or count of 
the existing API objects?**

  No.

* **Will enabling / using this feature result in increasing time taken by any 
operations covered by [existing SLIs/SLOs]?**

  Scheduling time on clusters with more than 100 nodes might increase. Smaller clusters are unaffected.
  This is because `SelectorSpreading` doesn't take into account all the Nodes in big clusters when calculating skew,
  resulting in partial spreading at this scale.
  On the contrary, `PodTopologySpreading` considers all nodes when using topologies bigger than a Node, like a Zone.

  Before graduation, we will ensure that the latency increase is acceptable with Scalability SIG.

* **Will enabling / using this feature result in non-negligible increase of 
resource usage (CPU, RAM, disk, IO, ...) in any components?**

  kube-scheduler might use more CPU to calculate Zone spreading in certain configurations.
  In synthetic benchmarks, the new spreading spends 1.5ms to do PreScore/Score when there are 10k Pods in a 1k Nodes cluster,
  using 16 threads. This is comparable to SelectorSpread.

### Troubleshooting

* **How does this feature react if the API server and/or etcd is unavailable?**

  kube-scheduler won't receive Pods
  The effect is no more than it be without the feature.

* **What are other known failure modes?**

  - Pod scheduling is slow
    - Detection: Pod startup time is too high.
    - Diagnostics: Use the `framework_extension_point_duration_seconds` scheduler metric with label `extension_point` values `PreScore` and/or `Score`.
    - Mitigations: Disable the Feature Gate DefaultPodTopologySpreading in kube-scheduler.
    - Testing: There are performance dashboards.
  - Pods of a Service/ReplicaSet/ReplicationController/StatefulSet are not properly spread: spread is either too weak or too strong.
    - Detection: Too many pods belonging to the same Service/ReplicaSet/ReplicationController/StatefulSet are scheduled in a few nodes or
      are spread in too many nodes.
    - Mitigations: Use [Pod Topology spreading](https://kubernetes.io/docs/concepts/workloads/pods/pod-topology-spread-constraints)
      in your PodSpecs. Or modify the [default constraints](https://kubernetes.io/docs/concepts/workloads/pods/pod-topology-spread-constraints/#cluster-level-default-constraints)
      for the `PodTopologySpread` plugin to your preference.
    - Diagnostics: N/A
    - Testing: E2E tests ensure that Pods are evenly spread in a clusters with only one Service.
* **What steps should be taken if SLOs are not being met to determine the problem?**

If startup latency is in violation, there is the possibility that it's due to this feature.

1. Determine if the scheduler is the culprit: Check for significant latency in `e2e_scheduling_duration_seconds`.
1. The feature only affects scheduling algorithms, thus you can check for significant latency in `scheduling_algorithm_duration_seconds`.
1. To check if this feature is the culprit, look for significant latency in `framework_extension_point_duration_seconds`,
  using label `extension_point` with values `PreScore` and `Score`.
1. Try disabling the Feature Gate `DefaultPodTopologySpreading`.

## Implementation History

- 2019-09-26: Initial KEP sent out for review.
- 2020-01-20: KEP updated to make use of framework's PluginConfig.
- 2020-05-04: Update completed tasks and target alpha for 1.19.
- 2020-09-21: Add Beta graduation criteria and PRR.
- 2022-01-08: Graduate the feature to GA.

## Alternatives

- Make the topology keys used in `SelectorSpread` configurable.

    While this moves the scheduler in the right direction, there are two problems:
    
    1. We can only support one topology key.
    1. It makes it hard for pods to override the operator-provided spreading rules.

- Implement a mutating controller that sets defaults.

  This approach would likely allow us to provide a more flexible interface that
  can set defaults for specific namespaces or with other selectors. However, that
  wouldn't allow us to replace `SelectorSpread` with `PodTopologySpread`.