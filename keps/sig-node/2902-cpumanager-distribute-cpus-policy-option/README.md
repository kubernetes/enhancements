# KEP-2902: Add CPUManager policy option to distribute CPUs across NUMA nodes instead of packing them

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

Kubernetes 1.22 introduced a new framework for adding `CPUManager` policy options (#2625).
These new policy options allow one to tweak the behaviour of a given `CPUManager` policy without the need to introduce an entirely new policy.
Moreover, these policy options can build on one another, such that multiple tweaks can be made to a policy in an additive fashion.
The first option introduced in conjunction with #2625 allows one to ensure that only **full** CPUs are allocated to a container, rather than handing out individual hyperthreads from each CPU to different containers.
This KEP introduces a new `CPUManager` policy option to ensure that CPU allocations are evenly distributed across NUMA nodes in cases where more than one NUMA node is required to satisfy the allocation.

## Motivation

By default, the `CPUManager` will pack CPUs onto one NUMA node until it is filled, with any remaining CPUs simply spilling over to the next NUMA node. 
This can cause undesired bottlenecks in parallel code relying on barriers (and similar synchronization primitivies), as this type of code tends to run only as fast as its slowest worker (which is slowed down by the fact that fewer CPUs are available on at least one NUMA node).
By distributing CPUs evenly across NUMA nodes, application developers can more easily ensure that no single worker suffers from NUMA effects more than any other, improving the overall performance of these types of applications.

### Goals
 * Enable parallel algorithms to run more efficiently when they request more CPUs than can be allocated by a single NUMA node

### Non-Goals
  * Provide a general solution for all types of CPU distributions across NUMA nodes

## Proposal

We propose to add a new `CPUManager` policy option called `distribute-cpus-across-numa` to the `static` `CPUManager` policy.
When enabled, this will trigger the `CPUManager` to evenly distribute CPUs across NUMA nodes in cases where more than one NUMA node is required to satisfy the allocation.

### Risks and Mitigations

The risks of adding this new feature are quite low.
It is isolated to a specific policy option within the `CPUManager`, and is protected both by the option itself, as well as the `CPUManagerPolicyOptions` feature gate (which is disabled by default).

| Risk                                             | Impact | Mitigation |
| -------------------------------------------------| -------| ---------- |
| Bugs in the implementation lead to kubelet crash | High   | Disable the policy option and restart the kubelet. The workload will run but with CPU packing semantics - like it was before this new policy option was added. |

## Design Details

When `distribute-cpus-across-numa` is passed as a policy option, the following algorithm will be run to distribute CPUs across NUMA nodes instead of packing them:

```
Foreach NUMA node:
   * If all requested CPUs can be allocated from this single NUMA node;
   --> do the allocation

For each pair of NUMA nodes:
   * If the set of requested CPUs (modulo 2) can be evenly split across the 2 NUMA nodes; AND
   * Any remaining CPUs (after the modulo operation) can be striped across some subset of the NUMA nodes;
   --> do the allocation

For each 3-tuple of NUMA nodes:
   * If the set of requested CPUs (modulo 3) can be evenly distributed across the 3 NUMA nodes; AND
   * Any remaining CPUs (after the modulo operation) can be striped across some subset of the NUMA nodes;
   --> do the allocation

...

For the set of all NUMA nodes:
   * If the set of requested CPUs (module NUM_NUMA_NODES) can be evenly distributed across all NUMA nodes; AND
   * Any remaining CPUs (after the modulo operation) can be striped across some subset of the NUMA nodes;
   --> do the allocation
```

If none of the above conditions can be met, resort back to a best-effort fit of packing CPUs into NUMA nodes wherever they can fit.

NOTE: The striping operation after all CPUs have been evenly distributed will be performed such that the overall disribution of CPUs across those NUMA nodes remains as balanced as possible.

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
  - Feature gate name: `CPUManagerPolicyOptions`
  - Components depending on the feature gate: `kubelet`
- [X] Change the kubelet configuration to set a `CPUManager` policy of `static` and a `CPUManager` policy option of `distribute-cpus-across-numa`
  - Will enabling / disabling the feature require downtime of the control
    plane? No
  - Will enabling / disabling the feature require downtime or reprovisioning
    of a node? (Do not assume `Dynamic Kubelet Config` feature is enabled).
	Yes -- a kubelet restart is required.

###### Does enabling the feature change any default behavior?

No. In order to trigger any of the new logic, three things have to be true:
1. The `CPUManagerPolicyOptions` feature gate must be enabled
1. The `static` `CPUManager` policy must be selected
1. The new `distribute-cpus-across-numa` policy option must be selected

###### Can the feature be disabled once it has been enabled (i.e. can we roll back the enablement)?

Yes, the feature can be disabled by either:
1. Disabling the `CPUManagerPolicyOptions` feature gate
1. Switching the `CPUManager` policy to `none`
1. Removing `distribute-cpus-across-numa` from the list of `CPUManager` policy options

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
  1. Pick an node with at least 2 NUMA nodes in your cluster
  1. Ensure no other pods with exclusive CPUs are running on that node
  1. Launch a pod with a `nodeSelector` to that node that has a single container in it
  1. Run a ´sleep infinity` command and request exclusive CPUs for the container in the amount of (1 + NUM\_CPUS\_PER\_NUMA\_NODE)
  1. Verify that the list of CPUs allocated to the container are evenly distributed across 2 NUMA nodes instead of packed

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

- 2021-08-26: Initial KEP created
- 2021-08-30: Updates to fill out more sections, answer PRR questions
