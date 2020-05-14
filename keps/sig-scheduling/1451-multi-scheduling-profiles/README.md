# Multi Scheduling Profiles

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
  - [Implementation Details/Notes/Constraints](#implementation-detailsnotesconstraints)
    - [Component Config API](#component-config-api)
      - [Conversion between API versions](#conversion-between-api-versions)
      - [Defaults](#defaults)
      - [Validation](#validation)
      - [CLI flags binding](#cli-flags-binding)
    - [Kube-Scheduler implementation](#kube-scheduler-implementation)
  - [Risks and Mitigations](#risks-and-mitigations)
- [Design Details](#design-details)
  - [Test Plan](#test-plan)
  - [Graduation Criteria](#graduation-criteria)
    - [Alpha (v1.18):](#alpha-v118)
    - [Beta (v1.19):](#beta-v119)
- [Production Readiness Review Questionnaire](#production-readiness-review-questionnaire)
  - [Feature enablement and rollback](#feature-enablement-and-rollback)
  - [Rollout, Upgrade and Rollback Planning](#rollout-upgrade-and-rollback-planning)
  - [Monitoring requirements](#monitoring-requirements)
  - [Dependencies](#dependencies)
  - [Scalability](#scalability)
  - [Troubleshooting](#troubleshooting)
- [Implementation History](#implementation-history)
- [Alternatives](#alternatives)
<!-- /toc -->

## Release Signoff Checklist

- [x] (R) kubernetes/enhancements issue in release milestone, which links to KEP (this should be a link to the KEP location in kubernetes/enhancements, not the initial KEP PR)
- [x] (R) KEP approvers have set the KEP status to `implementable`
- [x] (R) Design details are appropriately documented
- [x] (R) Test plan is in place, giving consideration to SIG Architecture and SIG Testing input
- [x] (R) Graduation criteria is in place
- [x] (R) Production readiness review completed
- [x] "Implementation History" section is up-to-date for milestone
- [x] User-facing documentation has been created in [kubernetes/website], for publication to [kubernetes.io]
- [x] Supporting documentation e.g., additional design documents, links to mailing list discussions/SIG meetings, relevant PRs/issues, release notes

## Summary

As workloads in clusters become more heterogeneous, it is natural that they have
different scheduling needs.

We propose making the scheduler run different framework plugin configurations,
which we will call profiles and will be associated to a scheduler name.
Pods can choose to be scheduled under a particular configuration by setting the
scheduler name associated to it in its pod spec. They will continue to be
scheduled under the default configuration if they don't specify a scheduler
name (i.e. `.spec.schedulerName`). The scheduler will continue to schedule one
pod at a time.

## Motivation

Clusters run a variety of workloads, which can be *broadly* classified as
services (long-running) and batch jobs (run-to-completion). Some users may
choose to run only one class of workloads in a cluster, so they can provide a
reasonable configuration that suits their scheduling needs.

However, users may choose to run more heterogeneous workloads in a single
cluster. Or they could have a set of fixed nodes and a set that auto-scales,
requiring different scheduling behaviors in each of them.

Pods can influence scheduling decisions with features such as node/pod affinity,
tolerations or (alpha) even pod spreading. But there are 2 problems:

- A single kube-scheduler configuration will weigh scores in such a way that
  doesn't adjust to all types of workloads. For example, the default 
  configuration includes scores that seek high availability of services.
- Authors of the workloads need to be aware of the cluster characteristics and
  the weights of the scores to influence their pods' scheduling in a meaningful
  way.

To serve such heterogeneous types of workloads, some cluster operators choose to
run multiple schedulers, whether those are different binaries or kube-schedulers
with a different configuration. But this setup might cause race conditions
between the multiple schedulers, as they might have a different view of the
cluster resources at a given time. Additionally, more binaries requires more
management effort.

Instead, having a single kube-scheduler run multiple profiles will have the
same benefits of running multiple schedulers without running into race
conditions.

### Goals

- Add support in the kube-scheduler's component config API for multiple
scheduling profiles.
- Make kube-scheduler schedule pods using different profiles given the scheduler
name specified in the pod spec.

### Non-Goals

- Introduce new default scheduling profiles.

## Proposal

### User Stories

#### Story 1

I have two types of workloads that I want to run in two different sets of nodes.
For one type of workload, I want them to spread in the topology. But for the
other type, I prefer them to get scheduled in a few nodes as possible.

### Implementation Details/Notes/Constraints

#### Component Config API

[`v1alpha1` component config](
https://github.com/kubernetes/kubernetes/blob/50deec2/pkg/scheduler/apis/config/types.go#L46)
looks like the following:

```go
type KubeSchedulerConfiguration struct {
   ...
   SchedulerName string
   AlgorithmSource SchedulerAlgorithmSource
   HardPodAffinitySymmetricWeight
   Plugins *Plugins
   PluginConfig []PluginConfig
   ...
}
```

We will introduce `v1alpha2`, with the following structure:

```go
type KubeSchedulerConfiguration struct {
   ...
   Profiles []KubeSchedulerProfile
}

type KubeSchedulerProfile struct {
   SchedulerName string
   Plugins *Plugins
   PluginConfig []PluginConfig
}
```

Note that we remove `AlgorithmSource` from the new API. Its functionality becomes redundant to
what can be configured with `Plugins` and `PluginConfig`.

##### Conversion between API versions

During conversion of `kubescheduler.config.k8s.io` from `v1alpha1` to `v1alpha2`, we will copy all
the necessary parameters from KubeSchedulerConfiguration into one item in the `Profiles` list.

In particular, configurations done by using `AlgorithmSource` will produce different values for
`Plugins` and `PluginConfig`.
This is similar to what we already do internally in [`legacy_registry.go`](
https://github.com/kubernetes/kubernetes/blob/fb66e807cd317254e5c7bf134186ddbfba757ef4/pkg/scheduler/framework/plugins/legacy_registry.go#L149)

`HardPodAffinitySymmetricWeight` would be moved to be a `PluginConfig.Arg` in
the `PluginConfig` slice for the plugin `InterPodAffinity` as `HardPodAffinityWeight`.

##### Defaults

The default configuration will look like:

```yaml
profiles:
  - schedulerName: 'default-scheduler'
```

Note that default plugins are loaded internally from the AlgorithmSource.

`HardPodAffinityWeight` will be set to have a default of `1` in the
`InterPodAffinity` plugin instantiation.

##### Validation

`SchedulerName`, `Plugins` and `PluginConfig` fields for each item in
`Profiles` will be validated according to the same rules as `v1alpha1`. We will
lose the early validation of `HardPodAffinitySymmetricWeight`. However, once we
try to instantiate a framework, the Plugin instantiation will fail, providing a
similar result as the binary is starting.

`SchedulerName` values will be validated to not repeat among the items of
`Profiles`.

Since kube-scheduler has only one queue, we will validate that all `Plugins.QueueSort`
configurations are strictly the same.

##### CLI flags binding

Note that, if component config is used, deprecated flags are currently ignored, which includes
`scheduler-name`, `algorithm-provider` and `hard-pod-affinity-symmetric-weight`. This implies
that we only have to worry about these flags in relationship with the default profile.

Thus, if component config is not used, we will preserve the behavior of the
flags as follows:
- `scheduler-name` will be bound to its counterpart in the default profile.
- `algorithm-provider` will produce different `Plugins` configurations. For examples, it will
produce an empty configuration for `default-scheduler`.
- `hard-pod-affinity-symmetric-weight` will be bound to a new deprecated option
  that will be processed into a `pluginConfig` slice of the default profile,
  like follows:
  
```yaml
profiles:
  - schedulerName: 'default-scheduler'
    pluginConfig:
      - name: 'InterPodAffinity'
      - args:
          hadPodAffinityWeight: <value>
```

#### Kube-Scheduler implementation

1. At startup, kube-scheduler will process all the different profiles,
initialize framework instances for them and store them in a registry.
If no profile is included in the configuration, one will be instantiated with
the name `default-scheduler` using the default plugins.

2. When getting notified about unscheduled pods, kube-scheduler will check the
scheduler name in the registry. If the name is present, they will be added to
the scheduler queue.

3. When a new pod is taken from the queue, it will get scheduled using the
framework instance from the registry corresponding to the specified scheduler
name.

Note that all framework instances will make use of the same shared cache
(for nodes and pods), from which a snapshot is taken for each scheduling cycle.
This is the main advantage over running multiple schedulers in a cluster.

### Risks and Mitigations

Operators could introduce profiles that disable scheduling features exposed in
the Pod Spec. Fortunately, the framework's plugins configuration makes it easy
to [create custom configurations from the default](
https://github.com/kubernetes/kubernetes/blob/50deec2/pkg/scheduler/apis/config/types.go#L156-L160)
through its `enabled` and `disabled` lists. 
However, we should discourage the use of `*` to disable all plugins in
the scheduler documentation.

## Design Details

### Test Plan

The following tests need to be in place:

- **Unit Tests**:
    - Component Config API conversion, validation and defaults
    - Core scheduler implementation. Current tests that use a default scheduler
    (or default framework) should continue passing with no configuration changes.
- **Integration tests**: Current tests with a default scheduler should continue passing with no
configuration changes. We need new tests in `test/integration/scheduler` exercising more than one
profile, in which:
    - Each profile would favor specific nodes, so that we can verify assignment.
    - Pods get binding events for the selected scheduler name.
    - Pods that don't specify a scheduler name continue to be scheduled by the default profile.

*Note on E2E tests*

Due to the proposed architecture, where a single kube-scheduler binary runs all the profiles, E2E
tests wouldn't increase the coverage of this feature over unit and integration tests.
Additionally, profiles can only be provided statically during cluster creation with our current
test infra. This implies that an independent job would be needed for each scheduler configuration.
But, as stated in our goals, this KEP doesn't introduce new default profiles.

### Graduation Criteria

#### Alpha (v1.18):

These are the required changes:

- [x] New `kubescheduler.config.k8s.io/v1alpha2` API.
    - [x] Conversion from `kubescheduler.config.k8s.io/v1alpha1`
    - [x] Validation.
    - [x] Defaults.
- [x] Scheduler can run more than one framework:
    - [x] Scheduler adds unscheduled pods to the pending queue for more than one name.
    - [x] Scheduler uses a framework using the scheduler name specified by the pod.
- [x] Tests from [Test Plan](#test-plan).

Note that we don't require a feature gate as users already have to opt-in by using
`kubescheduler.config.k8s.io/v1alpha2` instead of the previous version.

#### Beta (v1.19):

Scheduling profiles will graduate to beta altogether with the graduation of
the `kubescheduler.config.k8s.io` configuration API to Beta.

See [KEP 785] for more information.

Additionally, we will include the scheduler profile as a field in the
metrics related to scheduling attempts (counts and latency).

[KEP 785]:(https://git.k8s.io/enhancements/keps/sig-scheduling/785-scheduler-component-config-api)

## Production Readiness Review Questionnaire

### Feature enablement and rollback

* **How can this feature be enabled / disabled in a live cluster?**

  - [ ] Feature gate (also fill in values in `kep.yaml`)
    - Feature gate name:
    - Components depending on the feature gate:
  - [x] Other
    - Describe the mechanism:
    
      Modify kube-scheduler configuration file to use a single profile.
    - Will enabling / disabling the feature require downtime of the control
      plane?
      
      Yes, but only kube-scheduler.
    - Will enabling / disabling the feature require downtime or reprovisioning
      of a node? (Do not assume `Dynamic Kubelet Config` feature is enabled)
      
      No

* **Does enabling the feature change any default behavior?**

  No, as long as a profile with the name of `default-scheduler` is kept.

* **Can the feature be disabled once it has been enabled (i.e. can we rollback
  the enablement)?**
  
  Yes.

* **What happens if we reenable the feature if it was previously rolled back?**

  N/A.

* **Are there any tests for feature enablement/disablement?**

  There are tests (unit and integration) exercising single and multiple profiles.

### Rollout, Upgrade and Rollback Planning

* **How can a rollout fail? Can it impact already running workloads?**

  The scheduler errors and exits during start up. Existing workloads are not
  affected.

* **What specific metrics should inform a rollback?**

  Metric "schedule_attempts_total" remaining at zero when new pods are added.
  This would be a symptom of a bad profiles configuration.

* **Were upgrade and rollback tested? Was upgrade->downgrade->upgrade path tested?**

  N/A.

* **Is the rollout accompanied by any deprecations and/or removals of features,
  APIs, fields of API types, flags, etc.?**

  No.

### Monitoring requirements

* **How can an operator determine if the feature is in use by workloads?**

  - The following scheduling metrics will have a sub-field for the profile name
    that can be used for filtering per profile:
    - "schedule_attempts_total"
    - "e2e_scheduling_duration_seconds"
    - "scheduling_algorithm_duration_seconds"
    - "scheduling_algorithm_preemption_evaluation_seconds"
    - "binding_duration_seconds"
  - Pods have scheduling Events with different scheduler names.

* **What are the SLIs (Service Level Indicators) an operator can use to
  determine the health of the service?**
  
  - [x] Metrics
    - Metric name: `schedule_attempts_total`, `e2e_scheduling_duration_seconds`
    - Components exposing the metric: `kube-scheduler`

* **What are the reasonable SLOs (Service Level Objectives) for the above SLIs?**

  No new SLO.
  However, any scheduler related SLOs that can be defined in other KEPs or
  individually by cluster operators, can be split by profile, using the metrics
  discussed above.

* **Are there any missing metrics that would be useful to have to improve
  observability if this feature?**
  
  No.

### Dependencies

* **Does this feature depend on any specific services running in the cluster?**

  No.

### Scalability

* **Will enabling / using this feature result in any new API calls?**

  No.

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
  
  No.

* **Will enabling / using this feature result in non-negligible increase of
  resource usage (CPU, RAM, disk, IO, ...) in any components?**
  
  The overhead is minimal and far lower than running multiple kube-scheduler
  binaries. Here is the detail:
  
  - Memory: each profile is kept in memory with instantiated plugins.
  - CPU: There is a map lookup for each pod to obtain the profile.
  
  Note that using a single profile (default behavior) won't have any overhead.
  
### Troubleshooting

* **How does this feature react if the API server and/or etcd is unavailable?**

  N/A.

* **What are other known failure modes?**

  Malformed profiles cause scheduler to exit.

* **What steps should be taken if SLOs are not being met to determine the problem?**

  Configuration errors in the profiles are visible in logs.

## Implementation History

- 2020-01-14: Initial KEP sent out for review, including Summary, Motivation
and Proposal.
- 2020-01-21: Test Plan and Alpha Graduation criteria in KEP.
- 2020-05-08: Beta graduation criteria.
- 2020-05-14: Updated to new KEP template.

## Alternatives

The existing alternative to multiple profiles in a single scheduler is to run
multiple schedulers with different configurations. The problem with this
approach is the existence of multiple caches and a scheduler taking decisions
that might be incompatible with other schedulers.
