# KEP-4540: Add CPUManager policy option to restrict reservedSystemCPUs to system daemons and interrupt processing

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
- [Design Details](#design-details)
  - [Risks and Mitigations](#risks-and-mitigations)
    - [Archived Risk Mitigation (Option 1)](#archived-risk-mitigation-option-1)
    - [Archived Risk Mitigation (Option 2)](#archived-risk-mitigation-option-2)
  - [Test Plan](#test-plan)
      - [Prerequisite testing updates](#prerequisite-testing-updates)
      - [Unit tests](#unit-tests)
      - [Integration tests](#integration-tests)
      - [e2e tests](#e2e-tests)
  - [Graduation Criteria](#graduation-criteria)
    - [Alpha](#alpha)
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
- [Infrastructure Needed (Optional)](#infrastructure-needed-optional)
<!-- /toc -->

## Release Signoff Checklist

Items marked with (R) are required *prior to targeting to a milestone / release*.

- [ ] (R) Enhancement issue in release milestone, which links to KEP dir in [kubernetes/enhancements] (not the initial KEP PR)
- [ ] (R) KEP approvers have approved the KEP status as `implementable`
- [ ] (R) Design details are appropriately documented
- [ ] (R) Test plan is in place, giving consideration to SIG Architecture and SIG Testing input (including test refactors)
  - [ ] e2e Tests for all Beta API Operations (endpoints)
  - [ ] (R) Ensure GA e2e tests meet requirements for [Conformance Tests](https://github.com/kubernetes/community/blob/master/contributors/devel/sig-architecture/conformance-tests.md) 
  - [ ] (R) Minimum Two Week Window for GA e2e tests to prove flake free
- [ ] (R) Graduation criteria is in place
  - [ ] (R) [all GA Endpoints](https://github.com/kubernetes/community/pull/1806) must be hit by [Conformance Tests](https://github.com/kubernetes/community/blob/master/contributors/devel/sig-architecture/conformance-tests.md) 
- [ ] (R) Production readiness review completed
- [ ] (R) Production readiness review approved
- [ ] "Implementation History" section is up-to-date for milestone
- [ ] User-facing documentation has been created in [kubernetes/website], for publication to [kubernetes.io]
- [ ] Supporting documentation—e.g., additional design documents, links to mailing list discussions/SIG meetings, relevant PRs/issues, release notes

[kubernetes.io]: https://kubernetes.io/
[kubernetes/enhancements]: https://git.k8s.io/enhancements
[kubernetes/kubernetes]: https://git.k8s.io/kubernetes
[kubernetes/website]: https://git.k8s.io/website

## Summary

Starting with Kubernetes 1.22, a new `CPUManager` flag has facilitated the use of `CPUManager` Policy options (#2625) which enable users to customize their behavior based on workload requirements without having to introduce an entirely new policy.
These policy options work together to ensure an optimized cpu set is allocated for workloads running on a cluster.
A few policy options that already exist are `full-pcpus-only` (#2625) and `distribute-cpus-across-numa` (#2902) and `align-by-socket` (#3327).
With this KEP, a new `CPUManager` policy option `strict-cpu-reservation` is introduced which ensures that `reservedSystemCPUs` are strictly reserved for system daemons or interrupt processing and are not used by burstable and best-effort pods.

## Motivation

The static policy is used to reduce latency or improve performance. If you want to move system daemons or interrupt processing to dedicated cores, the obvious way is use the `reservedSystemCPUs` option. But in current implementation this isolation is implemented only for guaranteed pods with integer CPU requests not for burstable and best-effort pods (and guaranteed pods with fractional CPU requests).
Admission is only comparing the cpu requests against the allocatable cpus. Since the cpu limit are higher than the request, it allows burstable and best-effort pods to use up the capacity of `reservedSystemCPUs` and cause the OS/Systemd services to starve in real life deployments.
Solutions like Intel's Balloons Policy can be deployed to separate infrastructure and workload into different CPU pools but they require extra software, additional tuning and reduced CPU pool size could affect performance of multi-threaded processes.

### Goals
 * Align scheduler and node view for Node Allocatable (total - reserved).
 * Ensure `reservedSystemCPUs` is only used by system daemons or interrupt processing not by workloads.
 * Ensure no breaking changes for the `static` policy of `CPUManager`.

### Non-Goals

## Proposal

We propose to add a new `CPUManager` policy option called `strict-cpu-reservation` to the `static` policy of `CPUManager`.
When this policy option is enabled, we remove the reserved cores from the list of all available cores at the stage of calculation DefaultCPUSet. As a result, burstable and best-effort containers are launched with a cpuset in which the reserved cores are excluded.

### User Stories (Optional)

#### Story 1
To protect latency of workload, systemd daemons including irqbalance daemon are commonly constrained to the reserved CPUs.
Burstable and best-effort pods (and guaranteed pods with fractional CPU requests) running on the reserved CPUs causes CPU throttling for infrastructure services which results in poor system response time which in turn hits back on workload response time.
This issue is particularly bad in all-in-one deployments where workloads are placed on combined master+worker+storage nodes.

#### Story 2
Silently allowing workloads running on the reserved CPUs makes benchmarking infrastructure and workloads both inaccurate.

## Design Details

In Kubelet, when `strict-cpu-reservation` is enabled as a policy option, we remove the reserved cores from the shared pool at the stage of calculation DefaultCPUSet.

Feature impact can be illustrated as following:

With the following Kubelet configuration:

```yaml
kind: KubeletConfiguration
apiVersion: kubelet.config.k8s.io/v1beta1
featureGates:
  ...
  CPUManagerPolicyOptions: true
  CPUManagerPolicyAlphaOptions: true
cpuManagerPolicy: static
cpuManagerPolicyOptions:
  strict-cpu-reservation: "true"
reservedSystemCPUs: "0,32,1,33,16,48"
...
```

When `strict-cpu-reservation` is disabled:
```console
# cat /var/lib/kubelet/cpu\_manager\_state
{"policyName":"static","defaultCpuSet":"0-79","checksum":1241370203}
```

When `strict-cpu-reservation` is enabled:
```console
# cat /var/lib/kubelet/cpu\_manager\_state
{"policyName":"static","defaultCpuSet":"2-15,17-31,34-47,49-63","checksum":4141502832}
```

### Risks and Mitigations

The feature is isolated to a specific policy option `strict-cpu-reservation` under `cpuManagerPolicyOptions` and is protected by feature gate `CPUManagerPolicyAlphaOptions` or `CPUManagerPolicyBetaOptions` before the feature graduates to `Stable` i.e. enabled by default.

Concern for feature impact on best-effort workloads, the workloads that do not have resource requests, is brought up.

Kube-scheduler schedules pods on node allocatable (total - reserved). For best-effort pods, kube-scheduler uses default request values when scoring the nodes, see https://github.com/kubernetes/kubernetes/blob/master/pkg/scheduler/util/pod_resources.go#L32 and https://github.com/kubernetes/kubernetes/blob/master/pkg/scheduler/framework/plugins/noderesources/resource_allocation.go#L123, but the scheduler does not use the default request values when fitting the nodes i.e. best-effort pods are always admitted.

The concern is, when the feature graduates to `Stable`, it will be enabled by default, best-effort workloads could be starved on the node when the node runs out of CPU cores.

However, this is exactly the feature intent, best-effort workloads have no KPI requirement, they are meant to consume whatever CPU resources left on the node including starving from time to time. Best-effort workloads are not scheduled to run on the `reservedSystemCPUs` so they shall not be run on the `reservedSystemCPUs` to destablize the whole node.

Nevertheless, risk mitigation has been discussed in details (see archived options below) and we agree to start with the following node metrics of cpu pool sizes in Alpha stage to assess the actual impact in real deployment before revisiting if we need risk mitigation.

https://github.com/kubernetes/kubernetes/pull/127506
- report shared pool size, in millicores (e.g. 13500m)
- report exclusively allocated cores, counting full cores (e.g. 16)


#### Archived Risk Mitigation (Option 1)

This option is to add `numMinSharedCPUs` in `strict-cpu-reservation` option as the minimum number of CPU cores not available for exclusive allocation and expose it to Kube-scheduler for enforcement.

In Kubelet, when `strict-cpu-reservation` is enabled as a policy option, we remove the reserved cores from the shared pool at the stage of calculation DefaultCPUSet and remove the `MinSharedCPUs` from the list of available cores for exclusive allocation.

![MinSharedCPUs](./strict-cpu-allocation.png)

When `strict-cpu-reservation` is disabled:
```console
Total CPU cores: 64
ReservedSystemCPUs: 6
defaultCPUSet = Reserved (6) + 58 (available for exclusive allocation)
```

When `strict-cpu-reservation` is enabled:
```console
Total CPU cores: 64
ReservedSystemCPUs: 6
MinSharedCPUs: 4
defaultCPUSet = MinSharedCPUs (4) + 54 (available for exclusive allocation)
```

Prototype PR for the option is created:
https://github.com/kubernetes/kubernetes/pull/123979/commits

Add `numMinSharedCPUs` as part of `strict-cpu-reservation` option in Kubelet configuration:

```yaml
kind: KubeletConfiguration
apiVersion: kubelet.config.k8s.io/v1beta1
featureGates:
  ...
  CPUManagerPolicyOptions: true
  CPUManagerPolicyAlphaOptions: true
cpuManagerPolicy: static
cpuManagerPolicyOptions:
  strict-cpu-reservation: { "enable": "true", "numMinSharedCPUs": 4 }
reservedSystemCPUs: "0,32,1,33,16,48"
...
```

In Node API, we add `exclusive-cpu` in Node Allocatable for Kube-scheduler to consume.

```
  "status": {
    "capacity": {
      "cpu": "64",
      "exclusive-cpu": "64",
      "ephemeral-storage": "832821572Ki",
      "hugepages-1Gi": "0",
      "hugepages-2Mi": "0",
      "memory": "196146004Ki",
      "pods": "110"
    },
    "allocatable": {
      "cpu": "58",
      "exclusive-cpu": "54",
      "ephemeral-storage": "767528359485",
      "hugepages-1Gi": "0",
      "hugepages-2Mi": "0",
      "memory": "186067796Ki",
      "pods": "110"
    },
  ...
```

In kube-scheduler, `ExlusiveMilliCPU` is added in scheduler's `Resource` structure and `NodeResourcesFit` plugin is extended to filter out nodes that can not meet pod's exclusive CPU request.

A new item `ExclusiveMilliCPU` is added in the scheduler `Resource` structure:

```
// Resource is a collection of compute resource.
type Resource struct {
        MilliCPU          int64
        ExclusiveMilliCPU int64    // added
        Memory            int64
        EphemeralStorage  int64
        // We store allowedPodNumber (which is Node.Status.Allocatable.Pods().Value())
        // explicitly as int, to avoid conversions and improve performance.
        AllowedPodNumber int
        // ScalarResources
        ScalarResources map[v1.ResourceName]int64
}
```

A new node fitting failure 'Insufficient exclusive cpu' is added in the `NodeResourcesFit` plugin:

```
        if podRequest.MilliCPU > 0 && podRequest.MilliCPU > (nodeInfo.Allocatable.MilliCPU-nodeInfo.Requested.MilliCPU) {
                insufficientResources = append(insufficientResources, InsufficientResource{
                        ResourceName: v1.ResourceCPU,
                        Reason:       "Insufficient cpu",
                        Requested:    podRequest.MilliCPU,
                        Used:         nodeInfo.Requested.MilliCPU,
                        Capacity:     nodeInfo.Allocatable.MilliCPU,
                })
        }
        if nodeInfo.Allocatable.ExclusiveMilliCPU > 0 {    // added
                if podRequest.ExclusiveMilliCPU > 0 && podRequest.ExclusiveMilliCPU > (nodeInfo.Allocatable.ExclusiveMilliCPU-nodeInfo.Requested.ExclusiveMilliCPU) {
                        insufficientResources = append(insufficientResources, InsufficientResource{
                                ResourceName: v1.ResourceExclusiveCPU,
                                Reason:       "Insufficient exclusive cpu",
                                Requested:    podRequest.ExclusiveMilliCPU,
                                Used:         nodeInfo.Requested.ExclusiveMilliCPU,
                                Capacity:     nodeInfo.Allocatable.ExclusiveMilliCPU,
                        })
                }
        }
```

#### Archived Risk Mitigation (Option 2)

The problem with `MinSharedCPUs` is that it creates another complication like memory and hugpages, new resources vs overlapping resources, exclusive-cpus is a subset of cpu.

Currently the noderesources scheduler plugin does not filter out the best-effort pods in the case there's no available CPU.

Another option is to force the cpu requests for best effort pods to 1 MilliCPU in kubelet for the purpose of resource availability checks (or, equivalently, check there's at least 1 MilliCPU allocatable). This option is meant to be simpler than option-1, but it can create runaway pods similar to that in https://github.com/kubernetes/kubernetes/issues/84869.


### Test Plan

[X] I/we understand the owners of the involved components may require updates to
existing tests to make this code solid enough prior to committing the changes necessary
to implement this enhancement.

##### Prerequisite testing updates

<!--
Based on reviewers feedback describe what additional tests need to be added prior
implementing this enhancement to ensure the enhancements have also solid foundations.
-->

##### Unit tests

- `k8s.io/kubernetes/pkg/kubelet/cm/cpumanager/policy_static.go`: `03-18-2024` - `91.1`

##### Integration tests

- These cases will be added in the existing integration tests:
  - Feature gate enable/disable tests
  - `strict-cpu-reservation` policy option works as expected.
  - `strict-cpu-reservation` policy option works with existing policy options.

##### e2e tests

- These cases will be added in the existing e2e tests:
  - Feature gate enable/disable tests
  - `strict-cpu-reservation` policy option works as expected.
  - `strict-cpu-reservation` policy option works with existing policy options.

### Graduation Criteria

#### Alpha

- [X] Implement the new policy option.
- [X] Ensure proper unit tests are in place.
- [X] Ensure proper e2e node tests are in place.

#### Beta

- [X] Gather feedback from consumers of the new policy option.
- [X] Verify no major bugs reported in the previous cycle.

#### GA

- [X] Allow time for feedback (1 year).
- [X] Make sure all risks have been addressed.

### Upgrade / Downgrade Strategy

The new policy option is opt-in and orthogonal to the existing ones.

### Version Skew Strategy

No changes needed

## Production Readiness Review Questionnaire

### Feature Enablement and Rollback

The /var/lib/kubelet/cpu\_manager\_state needs be removed when changing the value of `strict-cpu-reservation`.

###### How can this feature be enabled / disabled in a live cluster?

- [X] Feature gate (also fill in values in `kep.yaml`)
  - Feature gate name: `CPUManagerPolicyAlphaOptions`
  - Components depending on the feature gate: `kubelet`
- [X] Change the kubelet configuration to set a `CPUManager` policy of `static` and a `CPUManager` policy option of `strict-cpu-reservation`
  - Will enabling / disabling the feature require downtime of the control
    plane? No
  - Will enabling / disabling the feature require downtime or reprovisioning
    of a node?  No -- removing /var/lib/kubelet/cpu\_manager\_state and restarting kubelet are required.


###### Does enabling the feature change any default behavior?

Yes. Reserved CPU cores for system usage will be strictly used for system daemons and interrupt processing no longer available for workloads.

The feature is only enabled when all following conditions are met:
1. The `CPUManagerPolicyAlphaOptions` feature gate must be enabled
2. The `static` `CPUManager` policy must be selected
3. The new `strict-cpu-reservation` policy option must be selected
4. The `reservedSystemCPUs` is not empty

###### Can the feature be disabled once it has been enabled (i.e. can we roll back the enablement)?

Yes, the feature can be disabled by either:
1. Disabling the `CPUManagerPolicyAlphaOptions` feature gate
2. Removing `strict-cpu-reservation` from the list of `CPUManager` policy options

###### What happens if we reenable the feature if it was previously rolled back?

The feature will be enabled regardless it is enabled for the first time or not.

###### Are there any tests for feature enablement/disablement?

- A specific e2e test will demonstrate that the default behaviour is preserved when the feature gate is disabled, or when the feature is not used (2 separate tests)

### Rollout, Upgrade and Rollback Planning

<!--
This section must be completed when targeting beta to a release.
-->

###### How can a rollout or rollback fail? Can it impact already running workloads?

<!--
Try to be as paranoid as possible - e.g., what if some components will restart
mid-rollout?

Be sure to consider highly-available clusters, where, for example,
feature flags will be enabled on some API servers and not others during the
rollout. Similarly, consider large clusters and how enablement/disablement
will rollout across nodes.
-->

###### What specific metrics should inform a rollback?

<!--
What signals should users be paying attention to when the feature is young
that might indicate a serious problem?
-->

###### Were upgrade and rollback tested? Was the upgrade->downgrade->upgrade path tested?

<!--
Describe manual testing that was done and the outcomes.
Longer term, we may want to require automated upgrade/rollback tests, but we
are missing a bunch of machinery and tooling and can't do that now.
-->

###### Is the rollout accompanied by any deprecations and/or removals of features, APIs, fields of API types, flags, etc.?

<!--
Even if applying deprecation policies, they may still surprise some users.
-->

### Monitoring Requirements

###### How can an operator determine if the feature is in use by workloads?

Inspect the `CPUManager` state file /var/lib/kubelet/cpu\_manager\_state.

###### How can someone using this feature know that it is working for their instance?

Inspect the pods' status file -- check the reserved cores are not used by them.

Below is an example:

```console
# kubectl exec cnf1-58446568f4-dr986 -n cnf1-ns -- grep Cpus_allowed /proc/self/status
Cpus_allowed:   fffefffc,fffefffc
Cpus_allowed_list:      2-15,17-31,34-47,49-63
```

###### What are the reasonable SLOs (Service Level Objectives) for the enhancement?

This feature to protect infrastructure services, running on the reserved CPU cores, from bursty workloads.

###### What are the SLIs (Service Level Indicators) an operator can use to determine the health of the service?

Run `top -H` to observe `reservedSystemCPUs` are not used by workloads.

###### Are there any missing metrics that would be useful to have to improve observability of this feature?

None

### Dependencies

None

###### Does this feature depend on any specific services running in the cluster?

No

### Scalability

###### Will enabling / using this feature result in any new API calls?

No

###### Will enabling / using this feature result in introducing new API types?

No

###### Will enabling / using this feature result in any new calls to the cloud provider?

No

###### Will enabling / using this feature result in increasing size or count of the existing API objects?

No

###### Will enabling / using this feature result in increasing time taken by any operations covered by existing SLIs/SLOs?

No

###### Will enabling / using this feature result in non-negligible increase of resource usage (CPU, RAM, disk, IO, ...) in any components?

No

###### Can enabling / using this feature result in resource exhaustion of some node resources (PIDs, sockets, inodes, etc.)?

No

### Troubleshooting

Increase kubelet log level and check kubelet log for errors.

Below is how to check kubelet log when it runs as a systemd service:
```console
journalctl _SYSTEMD_INVOCATION_ID=`systemctl show -p InvocationID --value kubelet.service`
```


###### How does this feature react if the API server and/or etcd is unavailable?

There is no known impact.

###### What are other known failure modes?

There is no known failure mode.

###### What steps should be taken if SLOs are not being met to determine the problem?

You can safely disable the feature.

## Implementation History

- 2023-03-08: Initial KEP created

## Drawbacks

## Alternatives

## Infrastructure Needed (Optional)
