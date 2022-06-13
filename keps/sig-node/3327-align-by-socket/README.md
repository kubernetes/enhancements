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
  - [Proposed Change](#proposed-change)
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

Starting with Kubernetes 1.22, a new `CPUManager` flag has facilitated the use of `CPUManager` Policy options(#2625) which enable users to customize their behavior based on workload requirements without having to introduce an entirely new policy.
These policy options work together to ensure an optimized cpu set is allocated for workloads running on a cluster.
The two policy options that already exist are `full-pcpus-only`(#2625) and `distribute-cpus-across-numa` (#2902).
With this KEP, a new `CPUManager` policy option is introduced which ensures that all CPUs on a socket are considered to be aligned.
Thus, the `CPUManager` will send a broader set of preferred hints to `TopologyManager`, enabling the increased likelihood of the best hint to be socket aligned with respect to CPU and other devices managed by `DeviceManager`.

## Motivation

With the evolution of CPU architectures, the number of NUMA nodes per socket has increased.
The devices managed by `DeviceManager` may not be uniformly distributed across all NUMA nodes.
Thus there can be scenarios where perfect alignment between devices and CPU may not be possible.
Latency sensitive applications desire resources to be aligned at least within the same socket if NUMA alignment is not possible for optimal performance.
By default, the `CPUManager` prefers CPU allocations which require a minimum number of NUMA nodes.
However, if the NUMA nodes selected for allocation are spread across sockets, it results in degraded performance.
By ensuring the selected NUMA nodes are socket aligned, predictable performance can be achieved.
The best possible alignment of CPUs with other resources(viz. Which are managed by `DeviceManager`) is crucial to guarantee predictable performance for latency sensitive applications.

### Goals
 * Ensure  CPUs are aligned  at socket boundary rather than NUMA node boundary.

### Non-Goals
  * Guarantee optimal NUMA allocation for cpu distribution.

## Proposal

We propose to add a new `CPUManager` policy option called `align-by-socket` to the  `static` policy of `CPUManager`.
With this policy option, the `CPUManager` will prefer those hints in which all CPUs are within same socket in addition to exisiting hints which require minimum NUMA nodes. With this policy option CPUs will be considered aligned at socket boundary instead of NUMA boundary during allocation. Thus if best hint consist of NUMA nodes within one socket, `CPUManager` may try to assign available CPUs from all NUMA nodes of socket.

### Risks and Mitigations

The risks of adding this new feature are quite low.
It is isolated to a specific policy option within the `CPUManager`, and is protected both by the option itself, as well as the `CPUManagerPolicyAlphaOptions` feature gate (which is disabled by default).

| Risk                                             | Impact | Mitigation |
| -------------------------------------------------| -------| ---------- |
| Bugs in the implementation lead to kubelet crash | High   | Disable the policy option and restart the kubelet. The workload will run but CPU allocations can spread across socket in cases when allocation could have been within same socket |

## Design Details

### Proposed Change

When `align-by-socket` is enabled as a policy option,  the `CPUManager`’s function `GetTopologyHints` will prefer hints which are socket aligned in addition to hints which require minimum number of NUMA nodes.

To achieve this, the following updates are needed to the `generateCPUTopologyHints` function of `static` policy of `CPUManager`:

```go
func (p *staticPolicy) generateCPUTopologyHints(availableCPUs cpuset.CPUSet, reusableCPUs cpuset.CPUSet, request int) []topologymanager.TopologyHint {
	...

    // Loop back through all hints and update the 'Preferred' field based on
    // counting the number of bits sets in the affinity mask and comparing it
    // to the minAffinitySize. Those with an equal number of bits set (and
    // with a minimal set of numa nodes) will be considered preferred.
    // If align-by-socket policy option is enabled, socket aligned hints are
    // also considered preferred.
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

During CPU allocation, in function `allocatedCPUs()`, `alignedCPUs` will consist of CPUs which are socket aligned instead of CPUs from NUMA nodes in `numaAffinity` hint when `align-by-socket` policy option is enabled.

```go
 func (p *staticPolicy) allocateCPUs(s state.State, numCPUs int, numaAffinity bitmask.BitMask, reusableCPUs cpuset.CPUSet) (cpuset.CPUSet, error) {
	...
     if numaAffinity != nil {
         alignedCPUs := cpuset.NewCPUSet()

         bits := numaAffinity.GetBits()
         // If align-by-socket policy option is enabled, NUMA based hint is expanded to
         // socket aligned hint. It will ensure that first socket aligned available CPUs are
         // allocated before we try to find CPUs across socket to satify allocation request.
         if p.PolicyOptions.AlignBySocket {
           bits = p.topology.ExpandToFullSocketBits(bits)
         }
         for _, numaNodeID := range bits {
           alignedCPUs = alignedCPUs.Union(assignableCPUs.Intersection(p.topology.CPUDetails.CPUsInNUMANodes(numaNodeID)))
         }

	 ...
}
```
This will ensure that for purpose of allocation, CPUs are considered aligned at socket boundary rather than NUMA boundary

`align-by-socket` policy options will work well for general case where number of NUMA nodes per socket are one or more.
In rare cases like `DualNumaMultiSocketPerNumaHT` where one NUMA can span multiple socket, above option is not applicable.
We will error out in cases when `align-by-socket` is enabled and underlying topology consist of multiple socket per NUMA.
We may address such scenarios in future if there is a usecase for it in real world.

In cases where number of NUMA nodes per socket is one or more as well as `TopologyManager` `single-numa-node` policy is enabled, the policy option of `align-by-socket` is redundant since allocation guarantees within the same NUMA are by definition socket aligned.
Hence, we will error out in case the policy option of `align-by-socket` is enabled along with `TopologyManager` `single-numa-node` policy.

The policyOption `align-by-socket` can work in conjunction with `TopologyManager` `best-effort` and `restricted` policy without any conflict.

### Test Plan

<!--
**Note:** *Not required until targeted at a release.*
The goal is to ensure that we don't accept enhancements with inadequate testing.

All code is expected to have adequate tests (eventually with coverage
expectations). Please adhere to the [Kubernetes testing guidelines][testing-guidelines]
when drafting this test plan.

[testing-guidelines]: https://git.k8s.io/community/contributors/devel/sig-testing/testing.md
-->

[X] I/we understand the owners of the involved components may require updates to
existing tests to make this code solid enough prior to committing the changes necessary
to implement this enhancement.

##### Prerequisite testing updates

<!--
Based on reviewers feedback describe what additional tests need to be added prior
implementing this enhancement to ensure the enhancements have also solid foundations.
-->

##### Unit tests

<!--
In principle every added code should have complete unit test coverage, so providing
the exact set of tests will not bring additional value.
However, if complete unit test coverage is not possible, explain the reason of it
together with explanation why this is acceptable.
-->

<!--
Additionally, for Alpha try to enumerate the core package you will be touching
to implement this enhancement and provide the current unit coverage for those
in the form of:
- <package>: <date> - <current test coverage>
The data can be easily read from:
https://testgrid.k8s.io/sig-testing-canaries#ci-kubernetes-coverage-unit

This can inform certain test coverage improvements that we want to do before
extending the production code to implement this enhancement.
-->

- `k8s.io/kubernetes/pkg/kubelet/cm/cpumanager/policy_static.go`: `06-13-2022` - `91.1`

##### Integration tests

<!--
This question should be filled when targeting a release.
For Alpha, describe what tests will be added to ensure proper quality of the enhancement.

For Beta and GA, add links to added tests together with links to k8s-triage for those tests:
https://storage.googleapis.com/k8s-triage/index.html
-->

- These cases will be added in the existing integration tests:
  - Feature gate enable/disable tests
  - `align-by-socket` policy option works as expected. When policy option is enabled
     - `generateCPUTopologyHints` prefers socket aligned hints in conjunction with hints with minimum NUMA nodes.
	 - `allocateCPUs` allocated CPU at socket boundary.
  - Verify no significant performance degradation

##### e2e tests

<!--
This question should be filled when targeting a release.
For Alpha, describe what tests will be added to ensure proper quality of the enhancement.

For Beta and GA, add links to added tests together with links to k8s-triage for those tests:
https://storage.googleapis.com/k8s-triage/index.html

We expect no non-infra related flakes in the last month as a GA graduation criteria.
-->
- These cases will be added in the existing e2e tests:
  - Feature gate enable/disable tests
  - `align-by-socket` policy option works as expected.

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
1. The `CPUManagerPolicyAlphaOptions` feature gate must be enabled
1. The `static` `CPUManager` policy must be selected
1. The new `align-by-socket` policy option must be selected

###### Can the feature be disabled once it has been enabled (i.e. can we roll back the enablement)?

Yes, the feature can be disabled by either:
1. Disabling the `CPUManagerPolicyAlphaOptions` feature gate
1. Switching the `CPUManager` policy to `none`
1. Removing `align-by-socket` from the list of `CPUManager` policy options

Existing workloads will continue to run uninterrupted, with any future workloads having their CPUs allocated according to the policy in place after the rollback.

###### What happens if we reenable the feature if it was previously rolled back?

No changes. Existing container will not see their allocation changed. New containers will.

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

Inspect the kubelet configuration of a node -- check for the presence of the feature gate and usage of the new policy option.

###### How can someone using this feature know that it is working for their instance?

In order to verify this feature is working, one should:
Pick a node with at least 2 Sockets and multiple NUMA nodes per socket
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

### Dependencies

<!--
This section must be completed when targeting beta to a release.
-->

###### Does this feature depend on any specific services running in the cluster?

<!--
Think about both cluster-level services (e.g. metrics-server) as well
as node-level agents (e.g. specific version of CRI). Focus on external or
optional services that are needed. For example, if this feature depends on
a cloud provider API, or upon an external software-defined storage or network
control plane.

For each of these, fill in the following—thinking about running existing user workloads
and creating new ones, as well as about cluster-level services (e.g. DNS):
  - [Dependency name]
    - Usage description:
      - Impact of its outage on the feature:
      - Impact of its degraded performance or high-error rates on the feature:
-->

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

### Troubleshooting

<!--
This section must be completed when targeting beta to a release.

For GA, this section is required: approvers should be able to confirm the
previous answers based on experience in the field.

The Troubleshooting section currently serves the `Playbook` role. We may consider
splitting it into a dedicated `Playbook` document (potentially with some monitoring
details). For now, we leave it here.
-->

###### How does this feature react if the API server and/or etcd is unavailable?

###### What are other known failure modes?

<!--
For each of them, fill in the following information by copying the below template:
  - [Failure mode brief description]
    - Detection: How can it be detected via metrics? Stated another way:
      how can an operator troubleshoot without logging into a master or worker node?
    - Mitigations: What can be done to stop the bleeding, especially for already
      running user workloads?
    - Diagnostics: What are the useful log messages and their required logging
      levels that could help debug the issue?
      Not required until feature graduated to beta.
    - Testing: Are there any tests for failure mode? If not, describe why.
-->

###### What steps should be taken if SLOs are not being met to determine the problem?

## Implementation History

- 2022-06-02: Initial KEP created
- 2022-06-08: Addressed review comments on KEP

## Drawbacks

<!--
Why should this KEP _not_ be implemented?
-->
`align-by-socket` policy option when enabled, it might result CPU allocation which may not be perfectly aligned by NUMA with other resources managed by `DeviceManager`.

## Alternatives

<!--
What other approaches did you consider, and why did you rule them out? These do
not need to be as detailed as the proposal, but should include enough
information to express the idea and why it was not acceptable.
-->
`align-by-socket` can alternatively be introduced as policy of `TopologyManager` which can choose hints from hint provider which are socket aligned.
However, it is concluded that fuction of `TopologyManager` is to act as arbitrator for hints provider and should make decisions based on hints provided rather than influencing the `BestHint` based on its own policy.
