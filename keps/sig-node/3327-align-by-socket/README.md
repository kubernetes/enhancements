# KEP-3327: Add CPUManager policy option to align CPUs by Socket instead of by NUMA node

<!-- toc -->
- [Release Signoff Checklist](#release-signoff-checklist)
- [Summary](#summary)
- [Motivation](#motivation)
  - [Goals](#goals)
  - [Non-Goals](#non-goals)
- [Proposal](#proposal)
  - [Risks and Mitigations](#risks-and-mitigations)
- [Design Details](#design-details)
  - [Test Plan](#test-plan)
  - [Graduation Criteria](#graduation-criteria)
    - [Alpha](#alpha)
    - [Beta](#beta)
    - [GA](#ga)
  - [Upgrade / Downgrade Strategy](#upgrade--downgrade-strategy)
  - [Version Skew Strategy](#version-skew-strategy)
- [Production Readiness Review Questionnaire](#production-readiness-review-questionnaire)
  - [Feature Enablement and Rollback](#feature-enablement-and-rollback)
  - [Monitoring Requirements](#monitoring-requirements)
  - [Scalability](#scalability)
- [Implementation History](#implementation-history)
<!-- /toc -->

## Release Signoff Checklist

Items marked with (R) are required *prior to targeting to a milestone / release*.

- [ ] (R) Enhancement issue in release milestone, which links to KEP dir in [kubernetes/enhancements] (not the initial KEP PR)
- [ ] (R) KEP approvers have approved the KEP status as `implementable`
- [ ] (R) Design details are appropriately documented
- [ ] (R) Test plan is in place, giving consideration to SIG Architecture and SIG Testing input (including test refactors)
  - [ ] e2e Tests for all Beta API Operations (endpoints)
  - [ ] (R) Ensure GA e2e tests for meet requirements for [Conformance Tests](https://github.com/kubernetes/community/blob/master/contributors/devel/sig-architecture/conformance-tests.md) 
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

Starting with Kubernetes 1.22, a new CPUManager flag has facilitated the use of CPUManager Policy options(#2625) which enable users to customize their behavior based on workload requirements without having to introduce an entirely new policy. These policy options work together to ensure an optimized cpu set is allocated for workloads running on cluster. The two policy options that already exist are full-pcpus-only(#2625) and distribute-cpus-across-numa (#2902).  With this KEP, new CPUManager policy option is introduced which ensures that all CPUs on a socket are considered to be aligned. Thus CPUManager will send a broader set of hints to TopologyManger, enabling the increased likelihood of the best hint to be  socket aligned with respect to CPU and other devices managed by DeviceManager


## Motivation

With the evolution of CPU architectures, the number of NUMA nodes per socket has increased. The devices managed by DeviceManager may not be uniformly distributed across all NUMA  nodes. Thus there can be scenarios where perfect alignment between devices and CPU may not be possible. Latency sensitive applications desire resources to be aligned at least within the same socket if NUMA alignment is not possible for optimal performance. By default, CPUManager prefers CPU allocation which requires a minimum number of NUMA nodes. However if NUMA nodes selected for allocation are spread across sockets, it results in degraded performance. By ensuring the selected NUMA nodes to be socket aligned, predictable performance can be achieved. The best possible alignment of CPUs with other resources(viz. Which are managed by device Manager) is crucial to guarantee predictable performance for latency sensitive applications.

### Goals
 * Ensure  CPUs are aligned  at socket boundary which will result in latency sensitive applications and parallel algorithms to run more efficiently in predictable fashion by increasing the probability of hint selection in which NUMA nodes are socket aligned.

### Non-Goals
  * Guarantee optimal NUMA allocation for cpu distribution.

## Proposal

We propose to add a new CPUManager policy option called align-by-socket to the static CPUManager policy. With this policy, the CPUManager will prefer those hints which are within the same socket (as opposed to just within the same NUMA node) if it is possible to have all CPUs allocated from the same socket.

### Risks and Mitigations

The risks of adding this new feature are quite low.
It is isolated to a specific policy option within the `CPUManager`, and is protected both by the option itself, as well as the `CPUManagerPolicyOptions` feature gate (which is disabled by default).

| Risk                                             | Impact | Mitigation |
| -------------------------------------------------| -------| ---------- |
| Bugs in the implementation lead to kubelet crash | High   | Disable the policy option and restart the kubelet. The workload will run but CPU allocations can spread across socket in cases when allocation could have been within same socket |

## Design Details

### Proposed Change 

When align-by-socket is enabled as a policy option,  the CPUManager’s GetTopologyHints() function will generate hints based on the sockets that a group of CPUs  belong to, rather than the NUMA nodes they belong to.

To achieve this, the following updates are needed to the GetTopologyHints() function:
```
func (p *staticPolicy) generateCPUTopologyHints(availableCPUs cpuset.CPUSet, reusableCPUs cpuset.CPUSet, request int) []topologymanager.TopologyHint {
	...

	// Loop back through all hints and update the 'Preferred' field based on
	// counting the number of bits sets in the affinity mask and comparing it
	// to the minAffinitySize. Only those with an equal number of bits set (and
	// with a minimal set of numa nodes) will be considered preferred.
	for i := range hints {
		if p.options.AlignBySocket && isSocketAligned(hints[i].NUMANodeAffinity) {
			hints[i].Preferred = true
			continue
		}
		if hints[i].NUMANodeAffinity.Count() == minAffinitySize {
			hints[i].Preferred = true
		}
	}

	return hints
}

```
At the end, we will have a list of desired hints. These hints will then be passed to the topology manager whose job it is to select the best hint (with an increased likelihood of selecting a hint that has CPUs which are aligned by socket now).

In case TopologyManager “single-numa-node” policy is enabled, the policy option of “align-by-socket” is redundant since allocation guarantees within the same numa are by definition socket aligned. Hence, we will error out in case the policy option of “align-by-socket” is enabled in conjunction with TopologyManager single-numa-node policy.

The policyOption align-by-socket can work in conjunction with TopologyManager “best-effort” and “restricted” policy without any conflict.

### Test Plan

We will extend both the unit test suite and the E2E test suite to cover the new policy option described in this KEP.

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

We expect no impact. The new policy option is opt-in and orthogonal to the existing ones.

### Version Skew Strategy

No changes needed

## Production Readiness Review Questionnaire

### Feature Enablement and Rollback

###### How can this feature be enabled / disabled in a live cluster?

- [X] Feature gate (also fill in values in `kep.yaml`)
  - Feature gate name: `CPUManagerPolicyAlphaOptions`
  - Components depending on the feature gate: `kubelet`
- [X] Change the kubelet configuration to set a `CPUManager` policy of `static` and a `CPUManager` policy option of `align-by-socket`
  - Will enabling / disabling the feature require downtime of the control
    plane? No
  - Will enabling / disabling the feature require downtime or reprovisioning
    of a node? (Do not assume `Dynamic Kubelet Config` feature is enabled).
	Yes -- a kubelet restart is required.

###### Does enabling the feature change any default behavior?

No. In order to trigger any of the new logic, three things have to be true:
1. The `CPUManagerPolicyOptions` feature gate must be enabled
1. The `static` `CPUManager` policy must be selected
1. The new `align-by-socket` policy option must be selected

###### Can the feature be disabled once it has been enabled (i.e. can we roll back the enablement)?

Yes, the feature can be disabled by either:
1. Disabling the `CPUManagerPolicyOptions` feature gate
1. Switching the `CPUManager` policy to `none`
1. Removing `align-by-socket` from the list of `CPUManager` policy options

Existing workloads will continue to run uninterrupted, with any future workloads having their CPUs allocated according to the policy in place after the rollback.

###### What happens if we reenable the feature if it was previously rolled back?

No changes. Existing container will not see their allocation changed. New containers will.

###### Are there any tests for feature enablement/disablement?

- A specific e2e test will demonstrate that the default behaviour is preserved when the feature gate is disabled, or when the feature is not used (2 separate tests)

### Monitoring Requirements

###### How can an operator determine if the feature is in use by workloads?

Inspect the kubelet configuration of a node -- check for the presence of the feature gate and usage of the new policy option.

###### How can someone using this feature know that it is working for their instance?

In order to verify this feature is working, one should:
Pick a node with at least 2 Sockets and 8 NUMA nodes
Ensure no other pods with exclusive CPUs are running on that node
Launch a 2 pods with a nodeSelector to that node that has a single container in it
Run a `sleep infinity` command and request exclusive CPUs for the container in the amount of (4*NUM_CPUS_PER_NUMA_NODE - 8)
Verify that for both pods, all CPU’s are within same socket instead of cpu’s distributed across sockets

To verify the list of CPUs allocated to the container, one can either:
- `exec` into uthe container and run `taskset -cp 1` (assuming this command is available in the container).
- Call the `GetCPUS()` method of the `CPUProvider` interface in the `kubelet`'s [podresources API](https://pkg.go.dev/k8s.io/kubernetes/pkg/kubelet/apis/podresources#CPUsProvider).

###### What are the reasonable SLOs (Service Level Objectives) for the enhancement?

There are no specific SLOs for this feature.
Parallel workloads will benefit from this feature in application specific ways.

###### What are the SLIs (Service Level Indicators) an operator can use to determine the health of the service?

None

###### Are there any missing metrics that would be useful to have to improve observability of this feature?

None

###### Does this feature depend on any specific services running in the cluster?

This feature is `linux` specific, and requires a version of CRI that includes the `LinuxContainerResources.CpusetCpus` field.
This has been available since `v1alpha2`.

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

The algorithm required to implement this feature could delay:
1. Pod admission time
2. The time it takes to launch each container after pod admission

This delay should be minimal.

###### Will enabling / using this feature result in non-negligible increase of resource usage (CPU, RAM, disk, IO, ...) in any components?

No, the algorithm will run on a single `goroutine` with minimal memory requirements.

## Implementation History

- 2022-06-02: Initial KEP created
