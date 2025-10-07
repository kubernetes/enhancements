# KEP-4622: New TopologyManager Policy which configure the value of maxAllowableNUMANodes

<!-- toc -->
- [Release Signoff Checklist](#release-signoff-checklist)
- [Summary](#summary)
- [Motivation](#motivation)
  - [Goals](#goals)
  - [Non-Goals](#non-goals)
- [Proposal](#proposal)
  - [User Stories (Optional)](#user-stories-optional)
    - [Story 1](#story-1)
    - [Story 2](#story-2)
    - [Story 3](#story-3)
  - [Notes/Constraints/Caveats (Optional)](#notesconstraintscaveats-optional)
  - [Risks and Mitigations](#risks-and-mitigations)
- [Design Details](#design-details)
  - [Test Plan](#test-plan)
      - [Prerequisite testing updates](#prerequisite-testing-updates)
      - [Unit tests](#unit-tests)
      - [Integration tests](#integration-tests)
      - [e2e tests](#e2e-tests)
  - [Graduation Criteria](#graduation-criteria)
    - [Beta](#beta)
    - [GA](#ga)
  - [Upgrade / Downgrade Strategy](#upgrade--downgrade-strategy)
  - [Version Skew Strategy](#version-skew-strategy)
- [Production Readiness Review Questionnaire](#production-readiness-review-questionnaire)
  - [Feature Enablement and Rollback](#feature-enablement-and-rollback)
  - [Rollout, Upgrade and Rollback Planning](#rollout-upgrade-and-rollback-planning)
  - [Monitoring Requirements](#monitoring-requirements)
  - [Dependencies](#dependencies)
  - [Scalability](#scalability)
  - [Troubleshooting](#troubleshooting)
- [Implementation History](#implementation-history)
- [Drawbacks](#drawbacks)
- [Alternatives](#alternatives)
<!-- /toc -->

## Release Signoff Checklist

Items marked with (R) are required *prior to targeting to a milestone / release*.

- [X] (R) Enhancement issue in release milestone, which links to KEP dir in [kubernetes/enhancements] (not the initial KEP PR)
- [X] (R) KEP approvers have approved the KEP status as `implementable`
- [X] (R) Design details are appropriately documented
- [X] (R) Test plan is in place, giving consideration to SIG Architecture and SIG Testing input (including test refactors)
  - [X] e2e Tests for all Beta API Operations (endpoints)
  - [] (R) Ensure GA e2e tests meet requirements for [Conformance Tests](https://github.com/kubernetes/community/blob/master/contributors/devel/sig-architecture/conformance-tests.md) 
  - [X] (R) Minimum Two Week Window for GA e2e tests to prove flake free
- [X] (R) Graduation criteria is in place
  - [X] (R) [all GA Endpoints](https://github.com/kubernetes/community/pull/1806) must be hit by [Conformance Tests](https://github.com/kubernetes/community/blob/master/contributors/devel/sig-architecture/conformance-tests.md) 
- [X] (R) Production readiness review completed
- [X] (R) Production readiness review approved
- [X] "Implementation History" section is up-to-date for milestone
- [X] User-facing documentation has been created in [kubernetes/website], for publication to [kubernetes.io]
- [X] Supporting documentation—e.g., additional design documents, links to mailing list discussions/SIG meetings, relevant PRs/issues, release notes

<!--
**Note:** This checklist is iterative and should be reviewed and updated every time this enhancement is being considered for a milestone.
-->

[kubernetes.io]: https://kubernetes.io/
[kubernetes/enhancements]: https://git.k8s.io/enhancements
[kubernetes/kubernetes]: https://git.k8s.io/kubernetes
[kubernetes/website]: https://git.k8s.io/website

## Summary

In this KEP, we propose a new TopologyManager Policy Option called `max-allowable-numa-nodes` to configure the value of maxAllowableNUMANodes in the TopologyManager. The current hard-coded value of 8 was added as a stop-gap 4 years ago to mitigate the state explosion that occurs when trying to enumerate the possible NUMA affinities and generating their hints. By making this setting configurable, we give users the ability to increase this limit when appropriate.

## Motivation

### Goals

- This proposal does not aim to modify the existing TopologyManager Policies. It focuses solely on introducing a new policy option to let users configure the maximum supported number of NUMA nodes.
- Support high-end CPUs with more than 8 NUMA nodes.

### Non-Goals

- It does not address other resource allocation or management aspects within Kubernetes.
- It does not attempt to remove the state explosion that still exists in the TopologyManager.

## Proposal

### User Stories (Optional)

#### Story 1

As a developer in the AI space, I want to use AI accelerators "super chips" which expose ARM cores with more than 8 NUMA nodes.  

#### Story 2

As a user in the high-performance-computing space, I want to enable the sub-NUMA or NUMA-per-socket option of my high-end x86 CPU, which will bring  
the count of NUMA nodes to exceed 8.

#### Story 3

As administrator of edge nodes, I want to use power-efficient yet massively parallel ARM chips which expose more than 8 NUMA nodes.

### Notes/Constraints/Caveats (Optional)

Setting values higher than the current default may cause performance degradation at admission time. Users must either be willing to accept this or know that they won't actually be affected by it in their particular setup. Fixing this will require rearchitecting the Topology Manager and it is thus out of scope of this KEP.

### Risks and Mitigations

The risk associated with implementing this new proposal is minimal. It pertains only to a distinct policy option within the `TopologyManager` and is safeguarded by the option's inherent security measures, in addition to the default deactivation of the `TopologyManagerPolicyBetaOptions` feature gate.

| Risk                                             | Impact | Mitigation |
| -------------------------------------------------| -------| ---------- |
| Set a value lower 8 causes kubelet crash         | High   | the minimum value legal value should be the current hardcoded value(8), If not, we should log it and fail |
| Set a value too high                             |  Low   | add a log when starting. If possible, we should mark the node Degraded somehow because allocation performance could be significantly slow |

## Design Details

Users can configure the value of maxAllowableNUMANodes in the TopologyManager when the kubelet starts up, It will fail and abort if the user sets the value is lower than the current hardcoded default (8).

```go
  case MaxAllowableNUMANodes:
   optValue, err := strconv.Atoi(value)
   if err != nil {
    return opts, fmt.Errorf("bad value for option %q: %w", name, err)
   }
   opts.MaxAllowableNUMANodes = optValue
      ...

  if opts.MaxAllowableNUMANodes < defaultMaxAllowableNUMANodes {
    return opts, fmt.Errorf("value for option %q is lower than defaultMaxAllowableNUMANodes: %d", MaxAllowableNUMANodes, opts.MaxAllowableNUMANodes)
  }
```

### Test Plan

[X] I/we understand the owners of the involved components may require updates to
existing tests to make this code solid enough prior to committing the changes necessary
to implement this enhancement.

##### Prerequisite testing updates

##### Unit tests

- `k8s.io/kubernetes/pkg/kubelet/cm/topologymanager`: `20240405` - `91.5%`

##### Integration tests

No new integration tests for kubelet are planned.

##### e2e tests


For beta:

- Verify the input validation with the existing e2e tests(e.g. 9 or 10 or something bigger than the current default but not "too big")

### Graduation Criteria

#### Beta

- Feature implemented behind the existing static policy feature flag
- Initial unit tests completed and coverage is improved
- Documents is improved and enough guidance and examples can be given to potential users.
- Add a e2e test to verify the input validation.

#### GA

- An existing metric: `topology_manager_admission_duration_ms` can be used.

### Upgrade / Downgrade Strategy

We anticipate no repercussions. The new policy option is voluntary and operates independent of the existing options.

### Version Skew Strategy


No changes needed.

## Production Readiness Review Questionnaire

### Feature Enablement and Rollback

1.31:

- enable by default
- allow gate to disable the feature
- release note

1.35:

- promote to GA
- LockToDefault: true (cannot be disabled)
- release note

1.36:

- feature gate removed


###### How can this feature be enabled / disabled in a live cluster?

- [X] Feature gate (also fill in values in `kep.yaml`)
  - Feature gate name: `TopologyManagerPolicyBetaOptions`
  - Components depending on the feature gate: `kubelet`
- [X] Other
  - Describe the mechanism: Change the kubelet configuration to set a TopologyManager policy of static and a TopologyManager policy option of `max-allowable-numa-nodes`
  - Will enabling / disabling the feature require downtime of the control
    plane?
    No
  - Will enabling / disabling the feature require downtime or reprovisioning
    of a node?
    Yes, Kubelet restart is required.

###### Does enabling the feature change any default behavior?

No.

###### Can the feature be disabled once it has been enabled (i.e. can we roll back the enablement)?

Yes, When it is disabled once (i.e. no value is set), this falls back to the default behavior.

###### What happens if we reenable the feature if it was previously rolled back?

Running containers won't be affected by the rollback of the feature, only newly created will.

###### Are there any tests for feature enablement/disablement?

This new `TopologyManager` policy option will start immediately from beta stage. The unit tests will test whether the configured value of `max-allowable-numa-nodes` is as expected and whether it is the default recommended value when it is not configured.

### Rollout, Upgrade and Rollback Planning


When feature a is not enabled or configured, its value is the default value. and the feature is fully contained in the kubelet, has no dependencies and rollback and upgrades both will affect only newly created pods.


###### How can a rollout or rollback fail? Can it impact already running workloads?


This feature has specific hardware dependencies that make rollout considerations unique:

1. This feature is only relevant for machines with more than 8 NUMA nodes AND when using a TopologyManager policy other than 'None'.

2. For such hardware configurations, removing this option (rollback) could prevent the kubelet from starting if the system has more NUMA nodes than the default limit allows.

3. For clusters with standard hardware (8 or fewer NUMA nodes), rollout or rollback has no impact as the default behavior remains unchanged.

4. For already running workloads, there is no impact during rollout or rollback - only new workloads will be affected by changes to this setting.


###### What specific metrics should inform a rollback?


We have an existing metric which records the topology manager admission time: `topology_manager_admission_duration_ms`.


###### Were upgrade and rollback tested? Was the upgrade->downgrade->upgrade path tested?


Rollout or upgrade do not impact already running workloads. We plan to add an e2e test for this in the furture.

###### Is the rollout accompanied by any deprecations and/or removals of features, APIs, fields of API types, flags, etc.?

No.

### Monitoring Requirements

An existing metric: `topology_manager_admission_duration_ms` for kubelet  can be used to check if the setting is causing unacceptable performance drops.


###### How can an operator determine if the feature is in use by workloads?

Examine the kubelet configuration of a node to verify the existence of the feature gate and the utilization of the new policy option. we can use the following command to check the feature if it is enabled:

```
kubectl get --raw "/api/v1/nodes/<nodename>/proxy/configz" | jq '.kubeletconfig.TopologyManagerPolicyOptions'
```

###### How can someone using this feature know that it is working for their instance?

- [ ] Events
  - Event Reason: 
- [ ] API .status
  - Condition name: 
  - Other field: 
- [X] Other (treat as last resort)
  - Details: If their system has more than 8 NUMA nodes, the TopologyManager is turned on and the kubelet is not crashing, then the feature is working.

###### What are the reasonable SLOs (Service Level Objectives) for the enhancement?


The value of max-allowable-numa-nodes does not (in and of itself) affect the latency of pod admission. With the TopologyManager enabled, the time to admit a pod is tied to the number of NUMA nodes on the physical machine. In the past, this was hard-coded at 8 to ensure that pod admission always completed in a reasonable amount of time. If a machine had more than 8 NUMA nodes, the kubelet would crash with a log message stating that the ToplogyManager is unsupported on machines with more than 8 NUMA nodes. With the new max-allowable-numa-nodes option, admins now have the ability to allow nodes with more than 8 NUMA nodes to run with the TopologyManager enabled. However, it is unknown exactly how much this will slow down pod admission on any given system. This feature is therefore to be used at-your-own-risk until we have a proper solution in place to reduce the state explosion that causes pod admission time to slow down as the number of NUMA nodes increases.

###### What are the SLIs (Service Level Indicators) an operator can use to determine the health of the service?

- [X] Metrics
  - Metric name: `topology_manager_admission_duration_ms`
  - [Optional] Aggregation method:
  - Components exposing the metric: kubelet

###### Are there any missing metrics that would be useful to have to improve observability of this feature?

The feature is not used by workloads in any way shape or form. and it only (potentially) impacts how long it takes for the kubelet to start a workload. We can easily check if this feature is enabled by looking at the kubelet config, example:

```shell
kubectl get --raw "/api/v1/nodes/<nodename>/proxy/configz" | jq '.kubeletconfig.TopologyManagerPolicyOptions'
```

### Dependencies

###### Does this feature depend on any specific services running in the cluster?

No. It doesn't rely on other Kubernetes components.

### Scalability

###### Will enabling / using this feature result in any new API calls?

No.

###### Will enabling / using this feature result in introducing new API types?

No.

###### Will enabling / using this feature result in any new calls to the cloud provider?

No.

###### Will enabling / using this feature result in increasing size or count of the existing API objects?

No.

###### Will enabling / using this feature result in increasing time taken by any operations covered by existing SLIs/SLOs?

It will slow down pod admission/start time on the node, and the slowdown occurs because the kubelet's TopoolgyManager now has more combinations it needs to consider when deciding where a cpus and devices can be allocated in an aligned way, and the slowdown affects only node configured with the feature, there is not any cluster impact as the feature is at node-level.

###### Will enabling / using this feature result in non-negligible increase of resource usage (CPU, RAM, disk, IO, ...) in any components?

It will increase the kubelet's CPU usage time. If your system has more than 8 NUMA nodes, then you will not be able to run kubernetes on it without this feature. so
the purpose is then to provide an escape hatch for those that are OK paying the price of increased latency for pod admission (and its associated CPU/RAM costs) in order to allow the kubelet to run on such a node.

###### Can enabling / using this feature result in resource exhaustion of some node resources (PIDs, sockets, inodes, etc.)?

Same answer as above.

### Troubleshooting

###### How does this feature react if the API server and/or etcd is unavailable?

N/A

###### What are other known failure modes?

Keeping the default value will cause the kubelet to fail to start on machines with 9 or more NUMA cells if any but the `none` topology manager policy is also configured. on machines with 9 or more NUMA cells if any but the `none` topology manager policy is also configured.

###### What steps should be taken if SLOs are not being met to determine the problem?

As a cluster administrator you should know the number of NUMA nodes on your nodes and adjust the value of the kubelet's topologyManager options or turn it off.

## Implementation History

- 2024-05-08 - initial KEP draft created
- 2024-06-06 - updates per review feedback
- 2025-10-07 - promote it to GA

## Drawbacks

- increased kubelet's CPU/Memory usage time
- increase in pod start time

Before this feature: the kubelet would crash.
With this feature: you get a potential slowdown, but at least the kubelet will run.

## Alternatives

Adding a new kubelet configuration option.

