# Cgroups v2

## Table of Contents

<!-- toc -->
- [Summary](#summary)
- [Motivation](#motivation)
- [Goals](#goals)
- [Non-Goals](#non-goals)
- [User Stories](#user-stories)
- [Implementation Details](#implementation-details)
- [Design](#design)
  - [Test Plan](#test-plan)
    - [Needed Tests](#needed-tests)
  - [Graduation Criteria](#graduation-criteria)
  - [Upgrade / Downgrade Strategy](#upgrade--downgrade-strategy)
- [Production Readiness Review Questionnaire](#production-readiness-review-questionnaire)
  - [Feature Enablement and Rollback](#feature-enablement-and-rollback)
  - [Rollout, Upgrade and Rollback Planning](#rollout-upgrade-and-rollback-planning)
  - [Monitoring Requirements](#monitoring-requirements)
  - [Dependencies](#dependencies)
  - [Scalability](#scalability)
  - [Troubleshooting](#troubleshooting)
- [Proposal](#proposal)
  - [Dependencies on OCI and container runtimes](#dependencies-on-oci-and-container-runtimes)
  - [Current status of dependencies](#current-status-of-dependencies)
- [Current cgroups usage and the equivalent in cgroups v2](#current-cgroups-usage-and-the-equivalent-in-cgroups-v2)
  - [cgroup namespace](#cgroup-namespace)
  - [Phase 1: Convert from cgroups v1 settings to v2](#phase-1-convert-from-cgroups-v1-settings-to-v2)
  - [Phase 2: Use cgroups v2 throughout the stack](#phase-2-use-cgroups-v2-throughout-the-stack)
- [Risk and Mitigations](#risk-and-mitigations)
<!-- /toc -->

## Summary

A proposal to add support for cgroups v2 to kubernetes.

## Motivation

The new kernel cgroups v2 API was declared stable more than two years
ago. Newer features in the kernel such as PSI depend upon cgroups
v2. groups v1 will eventually become obsolete in favor of cgroups v2.
Some distros are already using cgroups v2 by default, and that
prevents Kubernetes from working as it is required to run with cgroups
v1.

## Goals

This proposal aims to:

*   Add support for cgroups v2 to the Kubelet

## Non-Goals

*	Expose new cgroup2-only features
*	Dockershim
*	Plugins support

## User Stories

*   The Kubelet can run on a host using either cgroups v1 or v2.
*   Have features parity between cgroup v2 and v1.

## Implementation Details

## Design

### Test Plan

#### Needed Tests

- Run E2E tests on a cgroup v2 enabled host.

### Graduation Criteria

- Alpha: Phase 1 completed and basic support for running Kubernetes on
  a cgroups v2 host,  e2e tests coverage or have a plan for the
  failing tests.
  A good candidate for running cgroup v2 test is Fedora 31 that has
  already switched to default to cgroup v2.

- Beta: e2e tests coverage and performance testing.  Verify that both
  the CPU and Memory Manager work.

- GA: Assuming no negative user feedback based on production
  experience, promote after 2 releases in beta.
  *TBD* whether phase 2 must be implemented for GA.

### Upgrade / Downgrade Strategy

<!--
If applicable, how will the component be upgraded and downgraded? Make sure
this is in the test plan.

Consider the following in developing an upgrade/downgrade strategy for this
enhancement:
- What changes (in invocations, configurations, API use, etc.) is an existing
  cluster required to make on upgrade, in order to maintain previous behavior?
- What changes (in invocations, configurations, API use, etc.) is an existing
  cluster required to make on upgrade, in order to make use of the enhancement?
-->

N/A.  Not relevant to upgrades.  If the host is running with cgroup v2 then
it will be automatically detected and used.

## Production Readiness Review Questionnaire

### Feature Enablement and Rollback

###### How can this feature be enabled / disabled in a live cluster?

- [ ] Feature gate (also fill in values in `kep.yaml`)
  - Feature gate name:
  - Components depending on the feature gate:
- [X] Other
  - Describe the mechanism:
    configure the hosts to use cgroup v2
  - Will enabling / disabling the feature require downtime of the control
    plane?
    No, each host can be restarted to cgroup v2 separately
  - Will enabling / disabling the feature require downtime or reprovisioning
    of a node? (Do not assume `Dynamic Kubelet Config` feature is enabled).
    It requires downtime of a node since it needs to be rebooted

###### Does enabling the feature change any default behavior?

N/A.  It must work in the same way as on cgroup v1

###### Can the feature be disabled once it has been enabled (i.e. can we roll back the enablement)?

yes, it is enough to restart the node on cgroup v1

###### What happens if we reenable the feature if it was previously rolled back?

It should work seamlessly without any difference

###### Are there any tests for feature enablement/disablement?

The same E2E tests that work on cgroup v1 should work on cgroup v2

### Rollout, Upgrade and Rollback Planning

N/A.  Each node can be configured separately.

###### How can a rollout or rollback fail? Can it impact already running workloads?

N/A.  It requires a reboot to be enabled.  If the workload accesses directly the
cgroup file system, then also the workload must be enabled for cgroup v2.

###### What specific metrics should inform a rollback?

Pods not being healthy. One could inspect if the pods are getting the cgroups
set correctly referencing the conversion table in this KEP.

###### Were upgrade and rollback tested? Was the upgrade->downgrade->upgrade path tested?

N/A.  It depends on the node configuration and it is stateless.

###### Is the rollout accompanied by any deprecations and/or removals of features, APIs, fields of API types, flags, etc.?

The cgroup file system inside of the containers will use cgroup v2 instead of cgroup v1.

### Monitoring Requirements

###### How can an operator determine if the feature is in use by workloads?

An operator could run `cat /proc/self/cgroup` on a node to check if it is running in cgroups v2 mode.
If the node is using cgroup v2, then also the pods running on that node are using it.

###### How can someone using this feature know that it is working for their instance?


- [ ] Events
  - Event Reason:
- [ ] API .status
  - Condition name:
  - Other field:
- [X] Other (treat as last resort)
  - Details: pods are healthy.

###### What are the reasonable SLOs (Service Level Objectives) for the enhancement?

N/A.  Same as when running on cgroup v1.

###### What are the SLIs (Service Level Indicators) an operator can use to determine the health of the service?

- [ ] Metrics
  - Metric name:
  - [Optional] Aggregation method:
  - Components exposing the metric:
- [X] Other (treat as last resort)
  - Details: not a service

###### Are there any missing metrics that would be useful to have to improve observability of this feature?

No

### Dependencies

The container runtime must also support cgroup v2

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

### Troubleshooting

###### How does this feature react if the API server and/or etcd is unavailable?

N/A

###### What are other known failure modes?

N/A

###### What steps should be taken if SLOs are not being met to determine the problem?

If SLOs are not being met, reboot the node in cgroup v1 to disable this feature.

## Proposal

The proposal is to implement cgroups v2 in two different phases.

The first phase ensures that any configuration file designed for
cgroups v1 will continue to work on cgroups v2.

The second phase requires changes through the entire stack, including
the OCI runtime specifications.

At startup the Kubelet detects what hierarchy the system is using.  It
checks the file system type for `/sys/fs/cgroup` (the equivalent of
`stat -f --format '%T' /sys/fs/cgroup`).  If the type is `cgroup2fs`
then the Kubelet will use only cgroups v2 during all its execution.

The current proposal doesn't aim at deprecating cgroup v1, that must
still be supported through the stack.

Device plugins that require v2 enablement are out of the scope for
this proposal.

### Dependencies on OCI and container runtimes

In order to support features only available in cgroups v2 but not in
cgroups v1, the OCI runtime specs must be changed.

New features that are not present in cgroup v1 are out of the scope
for this proposal.

The dockershim implementation embedded in the Kubelet won't be
supported on cgroup v2.

### Current status of dependencies

- CRI-O+crun: support cgroups v2

- runc: since [v1.0.0-rc91](https://github.com/opencontainers/runc/tree/v1.0.0-rc91) experimentally, ready for production in [v1.0.0-rc93](https://github.com/opencontainers/runc/releases/tag/v1.0.0-rc93)

- containerd: support cgroup v2 since [v1.4.0](https://github.com/containerd/containerd/releases/tag/v1.4.0)

- Moby: [https://github.com/moby/moby/pull/40174](https://github.com/moby/moby/pull/40174)

- OCI runtime spec: [support cgroup v2 parameters](https://github.com/opencontainers/runtime-spec/pull/1040)

- cAdvisor already supports cgroups v2 ([https://github.com/google/cadvisor/pull/2309](https://github.com/google/cadvisor/pull/2309))

## Current cgroups usage and the equivalent in cgroups v2

|Kubernetes cgroups v1|Kubernetes cgroups v2 behavior|
|---|---|
|CPU stats for Horizontal Pod Autoscaling|No .percpu cpuacct stats.|
|CPU pinning based on integral cores|Cpuset controller available|
|Memory limits|Not changed, different naming|
|PIDs limits|Not changed, same naming|
|hugetlb|Added to linux-next, targeting Linux 5.6|

### cgroup namespace

A cgroup namespace restricts the view on the cgroups.  When
unshare(CLONE_NEWCGROUP) is done, the current cgroup the process
resides in becomes the root.  Other cgroups won't be visible from the
new namespace.  It was not enabled by default on a cgroup v1 system as
older kernel lacked support for it.

Privileged pods will still use the host cgroup namespace so to have
visibility on all the other cgroups.

### Phase 1: Convert from cgroups v1 settings to v2

We can convert the values passed by the k8s in cgroups v1 from to
cgroups v2 so Kubernetes users don’t have to change what they specify
in their manifests.

crun has implemented the conversion as follows:

**Memory controller**

| OCI (x) | cgroup 2 value (y) | conversion  |   comment |
|---|---|---|---|
| limit | memory.max | y = x ||
| swap | memory.swap.max | y = x ||
| reservation | memory.low | y = x ||

**PIDs controller**

| OCI (x) | cgroup 2 value (y) | conversion  |   comment |
|---|---|---|---|
| limit | pids.max | y = x ||

**CPU controller**

| OCI (x) | cgroup 2 value (y) | conversion  |  comment |
|---|---|---|---|
| shares | cpu.weight | y = (1 + ((x - 2) * 9999) / 262142) | convert from [2-262144] to [1-10000]|
| period | cpu.max | y = x| period and quota are written together|
| quota | cpu.max | y = x| period and quota are written together|

**blkio controller**

| OCI (x) | cgroup 2 value (y) | conversion  |   comment |
|---|---|---|---|
| weight | io.bfq.weight | y = (1 + (x - 10) * 9999 / 990) | convert linearly from [10-1000] to [1-10000]|
| weight_device | io.bfq.weight | y = (1 + (x - 10) * 9999 / 990) | convert linearly from [10-1000] to [1-10000]|
|rbps|io.max|y=x||
|wbps|io.max|y=x||
|riops|io.max|y=x||
|wiops|io.max|y=x||

**cpuset controller**

| OCI (x) | cgroup 2 value (y) | conversion  |   comment |
|---|---|---|---|
| cpus | cpuset.cpus | y = x ||
| mems | cpuset.mems | y = x ||

**hugetlb controller**

| OCI (x) | cgroup 2 value (y) | conversion  |   comment |
|---|---|---|---|
| <PAGE_SIZE>.limit_in_bytes | hugetlb.<PAGE_SIZE>.max | y = x ||

With this approach cAdvisor would have to read back values from
cgroups v2 files (already done).

Kubelet PR: [https://github.com/kubernetes/kubernetes/pull/85218](https://github.com/kubernetes/kubernetes/pull/85218)

### Phase 2: Use cgroups v2 throughout the stack

This option means that the values are written directly to cgroups v2
by the runtime. The Kubelet doesn’t do any conversion when setting
these values over the CRI.  We will need to add a cgroups v2 specific
LinuxContainerResources to the CRI.

This depends upon the container runtimes like runc and crun to be able
to write cgroups v2 values directly.

OCI will need support for cgroups v2 and CRI implementations will
write to the cgroups v2 section of the new OCI runtime config.json.

## Risk and Mitigations

Some cgroups v1 features are not available with cgroups v2:

- _cpuacct.usage_percpu_
- network stats from cgroup

Some cgroups v1 controllers such as _device_ and _net_cls_,
_net_prio_ are not available with the new version.  The alternative to
these controllers is to use eBPF.
