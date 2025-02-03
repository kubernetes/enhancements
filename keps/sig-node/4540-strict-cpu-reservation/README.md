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

- [x] (R) Enhancement issue in release milestone, which links to KEP dir in [kubernetes/enhancements] (not the initial KEP PR)
- [x] (R) KEP approvers have approved the KEP status as `implementable`
- [x] (R) Design details are appropriately documented
- [ ] (R) Test plan is in place, giving consideration to SIG Architecture and SIG Testing input (including test refactors)
  - [x] e2e Tests for all Beta API Operations (endpoints)
  - [ ] (R) Ensure GA e2e tests meet requirements for [Conformance Tests](https://github.com/kubernetes/community/blob/master/contributors/devel/sig-architecture/conformance-tests.md) 
  - [ ] (R) Minimum Two Week Window for GA e2e tests to prove flake free
- [ ] (R) Graduation criteria is in place
  - [ ] (R) [all GA Endpoints](https://github.com/kubernetes/community/pull/1806) must be hit by [Conformance Tests](https://github.com/kubernetes/community/blob/master/contributors/devel/sig-architecture/conformance-tests.md) 
- [ ] (R) Production readiness review completed
- [ ] (R) Production readiness review approved
- [x] "Implementation History" section is up-to-date for milestone
- [x] User-facing documentation has been created in [kubernetes/website], for publication to [kubernetes.io]
- [x] Supporting documentation—e.g., additional design documents, links to mailing list discussions/SIG meetings, relevant PRs/issues, release notes

[kubernetes.io]: https://kubernetes.io/
[kubernetes/enhancements]: https://git.k8s.io/enhancements
[kubernetes/kubernetes]: https://git.k8s.io/kubernetes
[kubernetes/website]: https://git.k8s.io/website

## Summary

Starting with Kubernetes 1.22, a new `CPUManager` flag has facilitated the use of `CPUManager` Policy options (#2625) which enable users to customize their behavior based on workload requirements without having to introduce an entirely new policy.
These policy options work together to ensure an optimized cpu set is allocated for workloads running on a cluster.
The policy options that already exist are `full-pcpus-only` (#2625) and `distribute-cpus-across-numa` (#2902) and `align-by-socket` (#3327) and `distribute-cpus-across-cores` (#4176).
With this KEP, a new `CPUManager` policy option `strict-cpu-reservation` is introduced which ensures that `reservedSystemCPUs` are strictly reserved for system daemons or interrupt processing and are not used by burstable and best-effort pods.

## Motivation

The static policy is used to reduce latency or improve performance. If you want to move system daemons or interrupt processing to dedicated cores, the obvious way is use the `reservedSystemCPUs` option. But in current implementation this isolation is implemented only for guaranteed pods with integer CPU requests not for burstable and best-effort pods (and guaranteed pods with fractional CPU requests).
Admission is only comparing the cpu requests against the allocatable cpus. Since the cpu limit are higher than the request, it allows burstable and best-effort pods to use up the capacity of `reservedSystemCPUs` and cause host OS services to starve in real life deployments.

### Goals
 * Align scheduler and node view for Node Allocatable (total - reserved).
 * Ensure `reservedSystemCPUs` is only used by system daemons or interrupt processing not by workloads.
 * Ensure no breaking changes for the `static` policy of `CPUManager`.

### Non-Goals
 * Change interface between node and scheduler.

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
  CPUManagerPolicyBetaOptions: true
cpuManagerPolicy: static
cpuManagerPolicyOptions:
  strict-cpu-reservation: "true"
reservedSystemCPUs: "0,32,1,33,16,48"
...
```

When `strict-cpu-reservation` is disabled:
```console
# cat /var/lib/kubelet/cpu_manager_state
{"policyName":"static","defaultCpuSet":"0-64","checksum":1241370203}
```

When `strict-cpu-reservation` is enabled:
```console
# cat /var/lib/kubelet/cpu_manager_state
{"policyName":"static","defaultCpuSet":"2-15,17-31,34-47,49-63","checksum":4141502832}
```

### Risks and Mitigations

The feature is isolated to a specific policy option `strict-cpu-reservation` under `cpuManagerPolicyOptions` and is protected by feature gate `CPUManagerPolicyAlphaOptions` or `CPUManagerPolicyBetaOptions` before the feature graduates to `Stable` i.e. enabled by default.

Concern for feature impact on best-effort workloads, the workloads that do not have resource requests, is brought up.

Kube-scheduler schedules pods on node allocatable (total - reserved). For best-effort pods, kube-scheduler uses default request values when scoring the nodes, see https://github.com/kubernetes/kubernetes/blob/master/pkg/scheduler/util/pod_resources.go#L32 and https://github.com/kubernetes/kubernetes/blob/master/pkg/scheduler/framework/plugins/noderesources/resource_allocation.go#L123, but the scheduler does not use the default request values when fitting the nodes i.e. best-effort pods are always admitted.

The concern is, when the feature graduates to `Stable`, it will be enabled by default, best-effort workloads could be starved on the node when the node runs out of CPU cores.

However, this is exactly the feature intent, best-effort workloads have no KPI requirement, they are meant to consume whatever CPU resources left on the node including starving from time to time. Best-effort workloads are not scheduled to run on the `reservedSystemCPUs` so they shall not be run on the `reservedSystemCPUs` to destablize the whole node.

We agree to start with the following node metrics of cpu pool sizes in Alpha and Beta stage to assess the actual impact in real deployment before revisiting if we need risk mitigation.

https://github.com/kubernetes/kubernetes/pull/127506
- `cpu_manager_shared_pool_size_millicores`: report shared pool size, in millicores (e.g. 13500m), expected to be non-zone otherwise best-effort pods will starve
- `cpu_manager_exclusive_cpu_allocation_count`: report exclusively allocated cores, counting full cores (e.g. 16)


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

No new integration tests for kubelet are planned.

##### e2e tests

- These cases will be added in the existing e2e tests:
  - CPU Manager works with `strict-cpu-reservation` policy option

- Basic functionality
1. Enable `CPUManagerPolicyBetaOptions` feature gate and `strict-cpu-reservation` policy option.
2. Create a simple pod of Burstable QoS type.
3. Verify the pod is not using the reserved CPU cores.
4. Delete the pod.


### Graduation Criteria

#### Alpha

- [X] Implement the new policy option.
- [X] Ensure proper unit tests are in place.

#### Beta

- [X] Gather feedback from consumers of the new policy option.
- [X] Verify no major bugs reported in the previous cycle.
- [X] Ensure proper e2e tests are in place.

#### GA

- [ ] Allow time for feedback (1 year).
- [ ] Make sure all risks have been addressed.

### Upgrade / Downgrade Strategy

The new policy option is opt-in and orthogonal to the existing ones.

### Version Skew Strategy

No changes needed.

## Production Readiness Review Questionnaire

### Feature Enablement and Rollback

The `/var/lib/kubelet/cpu_manager_state` needs to be removed when enabling or disabling the feature.

###### How can this feature be enabled / disabled in a live cluster?

- [X] Feature gate (also fill in values in `kep.yaml`)
  - Feature gate name: `CPUManagerPolicyBetaOptions`
  - Components depending on the feature gate: `kubelet`
- [X] Change the kubelet configuration to set a `CPUManager` policy of `static` and a `CPUManager` policy option of `strict-cpu-reservation`
  - Will enabling / disabling the feature require downtime of the control plane? No
  - Will enabling / disabling the feature require downtime or reprovisioning of a node?  No -- removing `/var/lib/kubelet/cpu_manager_state` and restarting kubelet are enough.


###### Does enabling the feature change any default behavior?

Yes. Reserved CPU cores will be strictly used for system daemons and interrupt processing no longer available for workloads.

The feature is only enabled when all following conditions are met:
1. The `static` `CPUManager` policy must be selected
2. The `CPUManagerPolicyBetaOptions` feature gate must be enabled
3. The new `strict-cpu-reservation` policy option must be selected
4. The `reservedSystemCPUs` is not empty

###### Can the feature be disabled once it has been enabled (i.e. can we roll back the enablement)?

Yes, the feature can be disabled by:
1. Disable feature gate `CPUManagerPolicyBetaOptions` or remove `strict-cpu-reservation` from the list of `CPUManager` policy options
2. Remove `/var/lib/kubelet/cpu_manager_state` and restart kubelet

###### What happens if we reenable the feature if it was previously rolled back?

The feature will be enabled regardless it is enabled for the first time or not.

###### Are there any tests for feature enablement/disablement?

- A specific e2e test will demonstrate that the default behaviour is preserved when the feature gate is disabled, or when the feature is not used (2 separate tests)

### Rollout, Upgrade and Rollback Planning

<!--
This section must be completed when targeting beta to a release.
-->

###### How can a rollout or rollback fail? Can it impact already running workloads?

If the feature rollout fails, burstable and best-efforts continue to run on the reserved CPU cores.
If the feature rollback fails, burstable and best-efforts continue not to run on the reserved CPU cores.
In either case, existing workload will not be affected.

When enabling or disabling the feature, make sure `/var/lib/kubelet/cpu_manager_state` is removed before restarting kubelet otherwise kubelet restart could fail.

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

Best-effort workloads are starved for prolonged time. This indicates you are lacking hardware to use the feature, or you should review the amount of CPU cores reserved.


###### Were upgrade and rollback tested? Was the upgrade->downgrade->upgrade path tested?

<!--
Describe manual testing that was done and the outcomes.
Longer term, we may want to require automated upgrade/rollback tests, but we
are missing a bunch of machinery and tooling and can't do that now.
-->

We use the feature in our internal environment and it works.

###### Is the rollout accompanied by any deprecations and/or removals of features, APIs, fields of API types, flags, etc.?

<!--
Even if applying deprecation policies, they may still surprise some users.
-->

No.

### Monitoring Requirements

###### How can an operator determine if the feature is in use by workloads?

Inspect the `defaultCpuSet` in `/var/lib/kubelet/cpu_manager_state`:
- When the feature is disabled, the reserved CPU cores are included in the `defaultCpuSet`.
- When the feature is enabled, the reserved CPU cores are not included in the `defaultCpuSet`.

###### How can someone using this feature know that it is working for their instance?

Inspect the pods' status file -- check the reserved cores are not used by them.

Below is an example:

```console
# kubectl exec cnf1-58446568f4-dr986 -n cnf1-ns -- grep Cpus_allowed /proc/self/status
Cpus_allowed:   fffefffc,fffefffc
Cpus_allowed_list:      2-15,17-31,34-47,49-63
```

###### What are the reasonable SLOs (Service Level Objectives) for the enhancement?

This feature allows users to protect infrastructure services from bursty workloads.

###### What are the SLIs (Service Level Indicators) an operator can use to determine the health of the service?

Monitor the following kubelet counters:
- `cpu_manager_shared_pool_size_millicores`: report shared pool size, in millicores (e.g. 13500m), expected to be non-zone otherwise best-effort pods will starve
- `cpu_manager_exclusive_cpu_allocation_count`: report exclusively allocated cores, counting full cores (e.g. 16)

###### Are there any missing metrics that would be useful to have to improve observability of this feature?

No.


### Dependencies

No.

###### Does this feature depend on any specific services running in the cluster?

No.

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

No.

###### Will enabling / using this feature result in non-negligible increase of resource usage (CPU, RAM, disk, IO, ...) in any components?

No.

###### Can enabling / using this feature result in resource exhaustion of some node resources (PIDs, sockets, inodes, etc.)?

No.

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

- 2024-03-08: Initial KEP created
- 2024-10-07: KEP gets LGTM and Approval


## Drawbacks

## Alternatives

## Infrastructure Needed (Optional)
