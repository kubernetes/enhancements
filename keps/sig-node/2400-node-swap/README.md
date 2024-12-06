# KEP-2400: Node memory swap support

<!-- toc -->
- [Release Signoff Checklist](#release-signoff-checklist)
- [Summary](#summary)
- [Motivation](#motivation)
  - [Scenarios](#scenarios)
  - [Goals](#goals)
  - [Non-Goals](#non-goals)
- [Proposal](#proposal)
  - [Enable Swap Support only for Burstable QoS Pods](#enable-swap-support-only-for-burstable-qos-pods)
    - [Set Aside Swap for System Critical Daemons](#set-aside-swap-for-system-critical-daemons)
    - [Swap Aware Eviction Manager](#swap-aware-eviction-manager)
    - [Best Practices](#best-practices)
      - [Disable swap for system critical daemons](#disable-swap-for-system-critical-daemons)
      - [Protect system critical daemons for iolatency](#protect-system-critical-daemons-for-iolatency)
      - [Control Plane Swap](#control-plane-swap)
      - [Use of a dedicated disk for swap](#use-of-a-dedicated-disk-for-swap)
      - [Swap as the default](#swap-as-the-default)
  - [Steps to Calculate Swap Limit](#steps-to-calculate-swap-limit)
    - [Example](#example)
  - [User Stories](#user-stories)
    - [Improved Node Stability](#improved-node-stability)
    - [Long-running applications that swap out startup memory](#long-running-applications-that-swap-out-startup-memory)
    - [Memory Flexibility](#memory-flexibility)
    - [Local development and systems with fast storage](#local-development-and-systems-with-fast-storage)
    - [Low footprint systems](#low-footprint-systems)
    - [Virtualization management overhead](#virtualization-management-overhead)
  - [Notes/Constraints/Caveats (Optional)](#notesconstraintscaveats-optional)
  - [Future Extensions of Swap](#future-extensions-of-swap)
  - [Risks and Mitigations](#risks-and-mitigations)
    - [Existing use cases of Swap](#existing-use-cases-of-swap)
    - [Exhausting swap resource](#exhausting-swap-resource)
    - [Security risk](#security-risk)
    - [Cgroupv1 support](#cgroupv1-support)
    - [Memory-backed volumes](#memory-backed-volumes)
      - [Brief technical overview of swap and evictions](#brief-technical-overview-of-swap-and-evictions)
      - [Current eviction limitations](#current-eviction-limitations)
      - [Advanced best-practices for manually setting memory evictions](#advanced-best-practices-for-manually-setting-memory-evictions)
- [Design Details](#design-details)
  - [Enabling swap as an end user](#enabling-swap-as-an-end-user)
  - [API Changes](#api-changes)
    - [KubeConfig addition](#kubeconfig-addition)
    - [CRI Changes](#cri-changes)
    - [Swap Metrics](#swap-metrics)
    - [Add swap support to NFD](#add-swap-support-to-nfd)
    - [Swap Aware Eviction Manager API](#swap-aware-eviction-manager-api)
  - [Test Plan](#test-plan)
      - [Prerequisite testing updates](#prerequisite-testing-updates)
      - [Unit tests](#unit-tests)
      - [Integration tests](#integration-tests)
      - [e2e tests](#e2e-tests)
  - [Graduation Criteria](#graduation-criteria)
    - [Alpha](#alpha)
    - [Alpha2](#alpha2)
    - [Beta 1](#beta-1)
    - [Beta 2](#beta-2)
    - [Beta 3](#beta-3)
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
  - [Just set <code>--fail-swap-on=false</code>](#just-set---fail-swap-onfalse)
- [Infrastructure Needed (Optional)](#infrastructure-needed-optional)
<!-- /toc -->

## Release Signoff Checklist

<!--
**ACTION REQUIRED:** In order to merge code into a release, there must be an
issue in [kubernetes/enhancements] referencing this KEP and targeting a release
milestone **before the [Enhancement Freeze](https://git.k8s.io/sig-release/releases)
of the targeted release**.

For enhancements that make changes to code or processes/procedures in core
Kubernetes—i.e., [kubernetes/kubernetes], we require the following Release
Signoff checklist to be completed.

Check these off as they are completed for the Release Team to track. These
checklist items _must_ be updated for the enhancement to be released.
-->

Items marked with (R) are required *prior to targeting to a milestone / release*.

- [x] (R) Enhancement issue in release milestone, which links to KEP dir in [kubernetes/enhancements] (not the initial KEP PR)
- [x] (R) KEP approvers have approved the KEP status as `implementable`
- [x] (R) Design details are appropriately documented
- [x] (R) Test plan is in place, giving consideration to SIG Architecture and SIG Testing input (including test refactors)
- [x] (R) Graduation criteria is in place
- [x] (R) Production readiness review completed
- [x] (R) Production readiness review approved
- [x] "Implementation History" section is up-to-date for milestone
- [x] User-facing documentation has been created in [kubernetes/website], for publication to [kubernetes.io]
- [x] Supporting documentation—e.g., additional design documents, links to mailing list discussions/SIG meetings, relevant PRs/issues, release notes

<!--
**Note:** This checklist is iterative and should be reviewed and updated every time this enhancement is being considered for a milestone.
-->

[kubernetes.io]: https://kubernetes.io/
[kubernetes/enhancements]: https://git.k8s.io/enhancements
[kubernetes/kubernetes]: https://git.k8s.io/kubernetes
[kubernetes/website]: https://git.k8s.io/website

## Summary

Kubernetes currently does not support the use of [swap
memory](https://en.wikipedia.org/wiki/Paging#Linux) on Linux, as it is
difficult to provide guarantees and account for pod memory utilization when
swap is involved. As part of Kubernetes’ earlier design, [swap support was
considered out of scope](https://github.com/kubernetes/kubernetes/issues/7294).

However, there are a [number of use cases](#user-stories) that would benefit
from Kubernetes nodes supporting swap. Hence, this proposal aims to add swap
support to nodes in a controlled, predictable manner so that Kubernetes users
can perform testing and provide data to continue building cluster capabilities
on top of swap.

This KEP aims to
introduce basic swap enablement and leave further extensions to follow-up KEPs.
This way Kubernetes users / vendors would be able to use swap in a basic manner
quickly while extensions would be brought to discussion in dedicated KEPs that
would progress in the meantime.

For example, to achieve this goal, this KEP does not introduce any APIs
that allow customizing how the feature behaves, but instead only determines 
whether the feature is enabled or disabled.
Instead, the behaviour is automatic and implicit that requires minimum user
intervention (see [proposal below](#steps-to-calculate-swap-limit) for more details).
As mentioned above, in the very near future, follow-up KEPs would bring API extension
and customizability, supporting zswap, and many other extensions to discussion.

## Motivation

There are two distinct types of user for swap, who may overlap:
- node administrators, who may want swap available for node-level performance
  tuning and stability/reducing noisy neighbour issues
- application developers, who have written applications that would benefit from
  using swap memory

There are hence a number of possible ways that one could envision swap use on a
node.

### Scenarios

1. Swap is enabled on a node's host system, but the kubelet does not permit
   Kubernetes workloads to use swap. (This scenario is a prerequisite for the
   following use cases.)
1. Swap is enabled at the node level. The kubelet can permit Kubernetes
   workloads scheduled on the node to use some quantity of swap, depending on
   the configuration.
1. Swap is set on a per-workload basis. The kubelet sets swap limits for each
   individual workload.

This KEP will be limited in scope to the first two scenarios. The third can be
addressed in a follow-up KEP. The enablement work that is in scope for this KEP
will be necessary to implement the third scenario.

### Goals

- On Linux systems, when swap is provisioned and available, Kubelet can start
  up with swap on.
- Configuration is available for kubelet to set swap utilization available to
  Kubernetes workloads, defaulting to 0 swap.
- Cluster administrators can enable and configure kubelet swap utilization on a
  per-node basis.
- Use of swap memory for cgroupsv2.
- Basic swap aware eviction manager

### Non-Goals

- Addressing non-Linux operating systems. Swap support will only be available
  for Linux.
- Provisioning swap. Swap must already be available on the system.
- Setting [swappiness]. This can already be set on a system-wide level outside
  of Kubernetes.
- Allocating swap on a per-workload basis with accounting (e.g. pod-level
  specification of swap), and/or APIs to customize and control the way kubelet
  calculates swap limits, grants swap access, etc. If desired, this should be
  designed and implemented as part of a follow-up KEP. This KEP is a
  prerequisite for that work. Hence, swap will be an overcommitted resource
  in the context of this KEP.
- Supporting zram, zswap, or other memory types like SGX EPC. These could be
  addressed in a follow-up KEP, and are out of scope.
- Use of swap for cgroupsv1.

[swappiness]: https://en.wikipedia.org/wiki/Memory_paging#Swappiness

## Proposal

We propose that, when swap is provisioned and available on a node, cluster
administrators can configure the kubelet such that:

- It can start with swap on.
- It will direct the CRI to allocate Kubernetes workloads 0 swap by default.
- It will have configuration options to configure swap utilization for the
  entire node.

This proposal enables scenarios 1 and 2 above, but not 3.

### Enable Swap Support only for Burstable QoS Pods

Before enabling swap support through the pod API, it is crucial to build confidence in this feature by carefully assessing its impact on workloads and Kubernetes. As an initial step, we propose enabling swap support for Burstable QoS Pods by automatically calculating the appropriate swap values, rather than allowing users to input these values manually. 

Swap access is granted only for pods of Burstable QoS. Guaranteed QoS pods are usually higher-priority pods, therefore we want to avoid swap's performance penalty for them. Best-Effort pods, on the contrary, are low-priority pods that are the first to be killed during node pressures. In addition, they're unpredictable, therefore it's hard to assess how much swap memory is a reasonable amount to allocate for them. 

By doing so, we can ensure a thorough understanding of the feature's performance and stability before considering the manual input of swap values in a subsequent beta release. This cautious approach will ensure the efficient allocation of resources and the smooth integration of swap support into Kubernetes.

Allocate the swap limit equal to the requested memory for each container and adjust the proportion of swap based on the total swap memory available.

#### Set Aside Swap for System Critical Daemons

**Note** In Beta2, we found that having system-critical daemons swapping memory could cause degradation of services.
Therefore, Kubelet will not automatically configure this, although the admin can still manually configure it
this way. In the near future, when a follow-up KEP regarding customizability is presented, this will be considered
to automatically be configured under a dedicated configuration.

System critical daemons (such as Kubelet) are essential for node health. Usually, an appropriate portion of system resources (e.g., memory, CPU) is reserved as system reserved. However, swap doesn't inherently support reserving a portion out of the total available. For instance, in the case of memory, we set `memory.min` on the node-level cgroup to ensure an adequate amount of memory is set aside, away from the pods, and for system critical daemons. But there is no equivalent for swap; i.e., no `memory.swap.min` is supported in the kernel.

Since this proposal advocates enabling swap only for the Burstable QoS pods, this can be done by setting `memory.swap.max` on the cgroups used by the Burstable QoS pods. The value of this `memory.swap.max` can be calculated by:

memory.swap.max = total swap memory available on the system - system reserve (memory)

This is the total amount of swap available for all the Burstable QoS pods; let's call it `TotalPodsSwapAvailable`. This will ensure that the system critical daemons will have access to the swap at least equal to the system reserved memory. This will indirectly act as having support for swap in system reserved.

#### Swap Aware Eviction Manager

While progressing this feature to stable, we found a major gap in this feature. The eviction manager must protect the node from saturation of swap memory. If a node exhausts swap memory, this can take down the node. In some customer cases, OOMKiller could step in to save the node but this should not be the default option.

Due to this, we want to enhance the eviction manager to be aware of swap. The eviction manager must step in when the node becoming unstable and start evicting pods that are swapping first.

At a high level to do this, we will introduce a new eviction signal to track swap usage on the node. Users can toggle this usage based on their risk.
When the swap is above this limit, then the node will evict pods in decreasing order of swap usage.

When a node has Swap pressure, a node condition will be added on the node. 
This allows for admins to at least see that there was swap pressure at one point.

#### Best Practices

This section is a recommendation for how to set up your nodes with swap if using this feature.

##### Disable swap for system critical daemons

As we were testing this feature, we found degration of services if you allow system critical daemons to swap.
This could mean that kubelet is performing slower than normal so if you experience this,
we recommend setting the cgroup for the system slice to avoid swap (ie `memory.swap.max 0`).
While doing this, we found that it is still possible for workloads to impact critical daemons.

##### Protect system critical daemons for iolatency

As we disabled swap for system slice, we saw cases where the system.slice would still be impacted by workloads swapping.
The workloads need to have less priority for IO than the system slice. We found that setting `io.latency` for system.slice fixes these issues.

See [io-control](https://facebookmicrosites.github.io/cgroup2/docs/io-controller.html#protecting-workloads-with-iolatency) for more details.

##### Control Plane Swap

We only recommend enabling swap for the worker nodes. The control plane contains mostly Guaranteed QoS Pods, so swap may be disabled for the most part.
The main concern would be swapping in the critical services on the control plane which can cause a performance impact.

##### Use of a dedicated disk for swap

We recommend using a separate disk for your swap partition. We recommend the separate disk be [encrypted](#security-risk).
If swap is on a partition or the root filesystem, workloads can interfere with system processes needing to write to disk.
If they occupy the same disk, it's possible processes can overwhelm swap and throw off the I/O of kubelet/container runtime/systemd, which would affect other workloads.
See [#protect-system-critical-daemons-for-iolatency] for more details on that.
Swap space is located on a disk so it is imperative to make sure your disk is fast enough for your use cases.

##### Swap as the default

We will turn the feature on for Beta 2 but the default setting will be `NoSwap`.

Enabling Swap on nodes is a pretty advanced feature which requires tuning and knowledge of the kernel.
We do not recommend swap on all nodes so we still suggest `--fail-swap-on=true` for most cases of Kubernetes.

If there is interest in trying out this feature, we suggest provisioning swap space on the worker node along with setting ``--fail-swap-on=false`
and restarting kubelet.

### Steps to Calculate Swap Limit

1. **Calculate the container's memory proportionate to the node's memory:**
  - Divide the container's memory request by the total node's physical memory. Let's call this value `ContainerMemoryProportion`.
  - If a container is defined with memory requests == memory limits, its `ContainerMemoryProportion` is defined as 0. Therefore, as can be seen below, its overall swap limit is also 0.

2. **Multiply the container memory proportion by the available swap memory for Pods:**
  - Meaning: `ContainerMemoryProportion * TotalPodsSwapAvailable`.

#### Example
Suppose we have a Burstable QoS pod with two containers:

- Container A: Memory request 20 GB
- Container B: Memory request 10 GB

Let's assume the total physical memory is 40 GB and the total swap memory available is also 40 GB. Also assume that the system reserved memory is configured at 2GB, 

Step 1: Determine the containers memory proportion:
- Container A: `20G/40G` = `0.5`.
- Container B: `10G/40G` = `0.25`.

Step 2: Determine swap limitation for the containers:
- Container A: `ContainerMemoryProportion * TotalPodsSwapAvailable` = `0.5 * 38G` = `19G`.
- Container B: `ContainerMemoryProportion * TotalPodsSwapAvailable` = `0.25 * 38G` = `9.5G`.

In this example, Container A would have a swap limit of 19 GB, and Container B would have a swap limit of 9.5 GB.

This approach allocates swap limits based on each container's memory request and adjusts the proportion based on the total swap memory available in the system. It ensures that each container gets a fair share of the swap space and helps maintain resource allocation efficiency.

### User Stories

#### Improved Node Stability

cgroupsv2 improved memory management algorithms, such as oomd, strongly
recommend the use of swap. Hence, having a small amount of swap available on
nodes could improve better resource pressure handling and recovery.

- https://man7.org/linux/man-pages/man8/systemd-oomd.service.8.html
- https://www.kernel.org/doc/html/latest/admin-guide/cgroup-v2.html#id1
- https://chrisdown.name/2018/01/02/in-defence-of-swap.html
- https://media.ccc.de/v/ASG2018-175-oomd
- https://github.com/facebookincubator/oomd/blob/master/docs/production_setup.md#swap

This user story is addressed by scenario 1 and 2, and could benefit from 3.

Note: critical / high-priority pods would not be able to access swap, but can
still be configured otherwise to gain swap access. In the future, APIs would
be able to be used to control swap in a more customized way. 

#### Long-running applications that swap out startup memory

- Applications such as the Java and Node runtimes rely on swap for optimal
  performance
  https://github.com/kubernetes/kubernetes/issues/53533#issue-263475425
- Initialization logic of applications can be safely swapped out without
  affecting long-running application resource usage
  https://github.com/kubernetes/kubernetes/issues/53533#issuecomment-615967154

This user story is addressed by scenario 2, and could benefit from 3.

#### Memory Flexibility

This user story addresses cases in which cost of additional memory is
prohibitive, or elastic scaling is impossible (e.g. on-premise/bare metal
deployments).

- Occasional cron job with high memory usage and lack of swap support means
  cloud nodes must always be allocated for maximum possible memory utilization,
  leading to overprovisioning/high costs
  https://github.com/kubernetes/kubernetes/issues/53533#issuecomment-354832960
- Lack of swap support would require provisioning 3x the amount of memory as
  required with swap
  https://github.com/kubernetes/kubernetes/issues/53533#issuecomment-617654228
- On-premise deployment can’t horizontally scale available memory based on load
  https://github.com/kubernetes/kubernetes/issues/53533#issuecomment-637715138
- Scaling resources is technically feasible but cost-prohibitive, swap provides
  flexibility at lower cost
  https://github.com/kubernetes/kubernetes/issues/53533#issuecomment-553713502

This user story is addressed by scenario 2, and could benefit from 3.

#### Local development and systems with fast storage

Local development or single-node clusters and systems with fast storage may
benefit from using available swap (e.g. NVMe swap partitions, one-node
clusters).

- Single node, local Kubernetes deployment on laptop
  https://github.com/kubernetes/kubernetes/issues/53533#issuecomment-361748518
- Linux has optimizations for swap on SSD, allowing for performance boosts
  https://github.com/kubernetes/kubernetes/issues/53533#issuecomment-589275277

This user story is addressed by scenarios 1 and 2, and could benefit from 3.

#### Low footprint systems

For example, edge devices with limited memory.

- Edge compute systems/devices with small memory footprints (\<2Gi)
  https://github.com/kubernetes/kubernetes/issues/53533#issuecomment-751398086
  https://github.com/k0sproject/k0s/issues/3830
- Clusters with nodes \<4Gi memory
  https://github.com/kubernetes/kubernetes/issues/53533#issuecomment-751404417

This user story is addressed by scenario 2, and could benefit from 3.

#### Virtualization management overhead

This would apply to virtualized Kubernetes workloads such as VMs launched by
kubevirt.

Every VM comes with a management related overhead which can sporadically be
pretty significant (memory streaming, SRIOV attachment, gpu attachment,
virtio-fs, …). Swap helps to not request much more memory to deal with short
term worst-case scenarios.

With virtualization, clusters are typically provisioned based on the workloads’
memory consumption, and any infrastructure container overhead is overcommitted.
This overhead could be safely swapped out.

- Required for live migration of VMs
  https://github.com/kubernetes/kubernetes/issues/53533#issuecomment-754878431

This user story is addressed by scenario 2, and could benefit from 3.

### Notes/Constraints/Caveats (Optional)

In updating the CRI, we must ensure that container runtime downstreams are able
to support the new configurations.

We considered adding parameters for both per-workload `memory-swap` and
`swappiness`. These are documented as part of the Open Containers [runtime
specification] for Linux memory configuration. Since `memory-swap` is a
per-workload parameter, and `swappiness` is optional and can be set globally,
we are choosing to only expose `memory-swap` which will adjust swap available
to workloads.

Since we are not currently setting `memory-swap` in the CRI, the current
default behaviour when `--fail-swap-on=false` is set is to allocate the same
amount of swap for a workload as memory requested. We will update the default
to not permit the use of swap by setting `memory-swap` equal to `limit`.

[runtime specification]: https://github.com/opencontainers/runtime-spec/blob/1c3f411f041711bbeecf35ff7e93461ea6789220/config-linux.md#memory

### Future Extensions of Swap

This feature was created so that we iterate on adding swap to Kubernetes. Due to this, workload swap was out of scope for the original implementation.
As this KEP moves towards stability, there is a need to comment on areas that were dropped from support in this KEP.
To make swap more useful for workloads, we acknowledge the need for proper APIs for swap.

- Swap should be opt-in and opt-out at the workload level.
- Workloads can request their own limits for `memory.swap.max`.
- Eviction Manager should be tunability in regards to swap limits.
- Eviction Manager should look at more advanced ways of determining swap pressure.

New functionality must not break this KEP so we think the best approach would be to implement a new Swap

### Risks and Mitigations

Having swap available on a system reduces predictability. Swap's performance is
worse than regular memory, sometimes by many orders of magnitude, which can
cause unexpected performance regressions. Furthermore, swap changes a system's
behaviour under memory pressure, and applications cannot directly control what
portions of their memory usage are swapped out. Since enabling swap permits
greater memory usage for workloads in Kubernetes that cannot be predictably
accounted for, it also increases the risk of noisy neighbours and unexpected
packing configurations, as the scheduler cannot account for swap memory usage.

This risk is mitigated by preventing any workloads from using swap by default,
even if swap is enabled and available on a system. This will allow a cluster
administrator to test swap utilization just at the system level without
introducing unpredictability to workload resource utilization.

Additionally, we will mitigate this risk by determining a set of metrics to
quantify system stability and then gathering test and production data to
determine if system stability changes when swap is available to the system
and/or workloads in a number of different scenarios.

Since swap provisioning is out of scope of this proposal, this enhancement
poses low risk to Kubernetes clusters that will not enable swap.

#### Existing use cases of Swap

As beta2 was being worked on, we discovered use cases where `--fail-swap-on=false` is used but Kubernetes is not utilizing swap.
Kind e2e tests run kubelet with `--fail-swap-on=false` and
the default developer configuration for `hack/local-up-cluster` allows for running developer clusters with swap enabled.

Now, `--fail-swap-on=false` is supported for both cgroup v1 and cgroupv2 although KEP-2400 does not support cgroup v1.
This is achieved by the newly introduced `MemorySwap` called `NoSwap`, which serves as the default swap behavior, that
will disable swap usage on the node while keeping the feature active.
In addition, nodes that support cgroups v1 only would be able to only use `NoSwap`, i.e. in such environments containers
will be restricted from having access to swap. 

This addresses existing use cases where `--fail-swap-on=false` in cgroupv1 and still allow us to turn this feature on.

#### Exhausting swap resource

In previous releases of Swap, we had an `UnlimitedSwap` option for workloads.
This can cause problems where workloads can use up all swap.
If all swap is used up on a node, it can make the node go unhealthy.
To avoid exhausting swap on a node, `UnlimitedSwap` was dropped from the API in beta2.

It was determined that the eviction manager should still be able to protect the node in case of swap memory pressure.
In this case, we will teach the eviction manager to be aware of swap as a resource to avoid exhausting swap resource.

#### Security risk

Enabling swap on a system without encryption poses a security risk, as critical information, such as Kubernetes secrets, may be swapped out to the disk. If an unauthorized individual gains access to the disk, they could potentially obtain these secrets. To mitigate this risk, it is recommended to use encrypted swap. However, handling encrypted swap is not within the scope of kubelet; rather, it is a general OS configuration concern and should be addressed at that level. Nevertheless, it is essential to provide documentation that warns users of this potential issue, ensuring they are aware of the potential security implications and can take appropriate steps to safeguard their system.
The documentation updates are required; there is already a [blog article](https://kubernetes.io/blog/2023/08/24/swap-linux-beta/) that mentions the security implications.

To guarantee that system daemons are not swapped, the kubelet must configure the `memory.swap.max` setting to `0` within the system reserved cgroup. Moreover, to make sure that burstable pods are able to utilize swap space, kubelet should verify that the cgroup associated with burstable pods should not be nested under the cgroup designated for system reserved.

Additionally, end user may decide to disable swap completely for a Pod or a container in beta 1 by making Pod guaranteed or set request == limit for a container. This way, there will be no swap enabled for the corresponding containers and there will be no information exposure risks.

#### Cgroupv1 support

In the early release of this feature, there was a goal to support cgroup v1. As the feature progressed, sig-node realized that supporting swap with cgroup v1 would be very difficult.
Therefore, this feature is limited to cgroupv2 only. The main goal is to deprecate cgroupv1 eventually so this should not be a major inconvience.

#### Memory-backed volumes

Kubernetes guarantees that some volumes' memory would never reside on disk, e.g. Secrets, memory-backed emptyDirs, etc.
Behind the scenes, Kubelet mounts such volumes as tmpfs volumes on the host.

To address this risk, if `--fail-swap-on=false`, the [tmpfs noswap option](https://www.kernel.org/doc/html/latest/filesystems/tmpfs.html)
will be used in order to prevent the volumes' pages from swapping to disk.

Bear in mind that the tmpfs noswap option is fairly new and is supported in kernel versions >= 6.4. However, different
Linux distributions can decide to backport this options to older versions of the kernel. Therefore, when
`--fail-swap-on=false` is being provided on a node:
* If the kernel version equals or is above 6.4, the tmpfs noswap option is being used when necessary.
* Else, kubelet would try to mount a dummy volume with the tmpfs noswap option to understand whether the option is
  backported. If the mount succeeds, the tmpfs noswap option is being used when necessary.
* Else, kubelet would raise a warning about the option not being supported and the possible risk.

In the longer term, when this option would be very widely supported, this would no longer be a concern, hence this logic
could be dropped.

##### Brief technical overview of swap and evictions

With swap being disabled, the memory eviction threshold should be set in a way that will let kubelet's
eviction manager identify memory spikes in time so it can execute custom logic and take care of reclaiming
memory and evicting pods before the kernel's OOM killer is invoked.

However, with swap enabled, the situation is fundamentally different.
In order for the kernel to start swapping memory, either the pod must breach its memory limits
(that are not relevant for kubelet-level evictions) or the node must use all of its physical memory.
Since swapping is a heavy operation in terms of performance, the kernel will try avoiding
it as long as it can, and will start swapping only when it lacks physical memory.

Therefore, when swap is enabled, it is acceptable to use all the available RAM which will result in the node
using some of the swap memory.

##### Current eviction limitations

Because evictions are out-of-scope for this KEP, the eviction manager would remain swap-unaware.

Let's say that on a specific node, the kernel would start swapping when there is less than X free physical memory
available (see more detailed explanation [below](#advanced-best-practices-for-manually-setting-memory-evictions)).
This means either one of the two behaviors will take place:
1. Memory evictions threshold > X: In this scenario, kubelet's eviction manager would kick in before the kernel has
   a chance to swap memory.
   This means that the kernel would generally not be able to swap memory (unless the memory spike
   is fast enough for the eviction manager to handle).
1. Memory evictions threshold <= X: In this scenario, the kernel would start swapping memory before the memory threshold 
   would be met. This practically means that the eviction manager would never kick in, hence it is turned off (unless
   for extreme cases where the memory spike is faster than the kernel's ability to swap memory).
   If the cluster admin sees this as a desirable state, the `--experimental-allocatable-ignore-eviction` kubelet flag
   can be used in order to emphasize that memory evictions are indeed turned off. 

The cluster admin can choose which of the above is the desired behavior.
If the first is applied, kubelet would be able to evict pods but pods would be able to swap only if memory limits
are breached.
If the second is applied, kubelet evictions would be practically disabled but node-level pressure would result in
swapping.

Advanced cluster admins can use the approach outlined [below](#advanced-best-practices-for-manually-setting-memory-evictions).

##### Advanced best-practices for manually setting memory evictions

Let's simplistically explain how the kernel decides to start swapping.
The kernel uses three "watermarks" (i.e. thresholds), which are called "high", "min", and "low", such that high > low > min.
When the "min" threshold is met, the kernel enters an "indirect reclaim" state, which basically means that the `kswapd`
daemon becomes active and asynchronously starts to reclaim memory.
If the "low" threshold is being met, the kernel enters a "direct reclaim" state, in which `kswapd` would aggressively
reclaim memory and memory would be throttled for most of the processes.
Later, when the "min" threshold is reached, the kernel goes back to "indirect reclaim", then when the "high" threshold
is met the node is not considered under pressure anymore.

![swap_watermarks](swap_watermarks.png)

To figure out what's the value of the different watermarks, one can do the following (the output is trimmed
for simplicity. values are in memory pages):
```shell
> cat /proc/zoneinfo
--
Node 0, zone    DMA32
        min      569
        low      1120
        high     1671
--
Node 0, zone   Normal
        min      16322
        low      32138
        high     47954
--
...
```

This is a real example from a 64GB RAM machine with the default kernel configurations. On such a machine:
* The `high` watermark is at `187.32Mi`.
* The `low` watermark is at `125.54Mi`.
* The `min` watermark is at `63.76Mi`.

An advanced cluster-admin can choose to set kubelet's memory eviction threshold according to these values on nodes
with swap enabled.
For example, the threshold can be set to a value between the low and min watermarks.
Bear in mind that the default memory eviction threshold of `100Mi` is between the low and min watermarks as well,
which means that by default evictions are expected to work decently.

## Design Details

We summarize the implementation plan as following:

1. Add a feature gate `NodeSwap` to enable swap support.
1. Leave the default value of kubelet flag `--fail-on-swap` to `true`, to avoid
   changing default behaviour.
1. Introduce a new kubelet config parameter, `MemorySwap`, which configures how
   much swap Kubernetes workloads can use on the node.
1. Introduce a new CRI parameter, `memory_swap_limit_in_bytes`.
1. Ensure container runtimes are updated so they can make use of the new CRI
   configuration.
1. Based on the behaviour set in the kubelet config, the kubelet will instruct
   the CRI on the amount of swap to allocate to each container. The container
   runtime will then write the swap settings to the container level cgroup.
1. Add node stats to report swap usage.
1. Enhance eviction manager to protect against swap memory running out.

### Enabling swap as an end user

Swap can be enabled as follows:

1. Provision swap on the target worker nodes,
1. Enable the `NodeSwap` feature flag on the kubelet,
1. Set `--fail-on-swap` flag to `false`, and
1. (Optional) Allow Kubernetes workloads to use swap by setting
   `MemorySwap.SwapBehavior` to `LimitedSwap` in the kubelet config.

### API Changes

#### KubeConfig addition

We will add an optional `MemorySwap` value to the `KubeletConfig` struct
in [pkg/kubelet/apis/config/types.go] as follows:

[pkg/kubelet/apis/config/types.go]: https://github.com/kubernetes/kubernetes/blob/6baad0a1d45435ff5844061aebab624c89d698f8/pkg/kubelet/apis/config/types.go#L81

```go
// KubeletConfiguration contains the configuration for the Kubelet
type KubeletConfiguration struct {
	metav1.TypeMeta
...
	// Configure swap memory available to container workloads.
	// +featureGate=NodeSwap
	// +optional
	MemorySwap MemorySwapConfiguration
}

type MemorySwapConfiguration struct {
	// swapBehavior configures swap memory available to container workloads. May be one of
	// "", "NoSwap": workloads can not use swap, default option.
	// "LimitedSwap": workload swap usage is limited. The swap limit is proportionate to the container's memory request.
	// +featureGate=NodeSwap
	// +optional
	SwapBehavior string `json:"swapBehavior,omitempty"`}
```

We want to expose common swap configurations based on the [Docker] and open
container specification for the `--memory-swap` flag. Thus, the
`MemorySwapConfiguration.SwapBehavior` setting will have the following effects:

* If `SwapBehavior` is set to `"LimitedSwap"`, containers have limited (or no) swap access.
  See "Steps to Calculate Swap Limit" above.
* If `SwapBehavior` is set to `""` or `"NoSwap"`, no workloads will utilize swap.

[docker]: https://docs.docker.com/config/containers/resource_constraints/#--memory-swap-details
[`memory.swap.max`]: https://www.kernel.org/doc/html/latest/admin-guide/cgroup-v2.html#memory

#### CRI Changes

The CRI requires a corresponding change in order to allow the kubelet to set
swap usage in container runtimes.  We will introduce a parameter
`memory_swap_limit_in_bytes` to the CRI API (found in
[k8s.io/cri-api/pkg/apis/runtime/v1/api.proto]):

[k8s.io/cri-api/pkg/apis/runtime/v1/api.proto]: https://github.com/kubernetes/kubernetes/blob/6baad0a1d45435ff5844061aebab624c89d698f8/staging/src/k8s.io/cri-api/pkg/apis/runtime/v1/api.proto#L563-L580

```go
// LinuxContainerResources specifies Linux specific configuration for
// resources.
message LinuxContainerResources {
...
    // Memory + swap limit in bytes. Default: 0 (not specified).
    int64 memory_swap_limit_in_bytes = 9;
...
}
```

#### Swap Metrics

We added metrics to the summary stats for the Node to report
`SwapAvailableBytes` and `SwapUsageBytes`.

```golang
type NodeStats struct {
  ...
 // Stats pertaining to swap resources. This is reported to non-windows systems only.
 // +optional
Swap *SwapStats `json:"swap,omitempty"`
}
```

```golang
// SwapStats contains data about memory usage
type SwapStats struct {
 // The time at which these stats were updated.
 Time metav1.Time `json:"time"`
 // Available swap memory for use.  This is defined as the <swap-limit> - <current-swap-usage>.
 // If swap limit is undefined, this value is omitted.
 // +optional
 SwapAvailableBytes *uint64 `json:"swapAvailableBytes,omitempty"`
 // Total swap memory in use.
 // +optional
 SwapUsageBytes *uint64 `json:"swapUsageBytes,omitempty"`
}
```

In addition, we've added swap to stats to Summary API and Prometheus endpoints (`/stats/summary` and
`/metrics/resource`):
```shell
> kubectl get --raw "/api/v1/nodes/<NODE-NAME>/proxy/metrics/resource" | grep swap
# HELP container_swap_usage_bytes [ALPHA] Current container amount of swap usage in bytes
# TYPE container_swap_usage_bytes gauge
container_swap_usage_bytes{container="c1",namespace="default",pod="test-pod"} 3.4400333824e+10 1687950863878
container_swap_usage_bytes{container="coredns",namespace="kube-system",pod="coredns-8f5847b64-t9gmr"} 0 1687950855483
# HELP node_swap_usage_bytes [ALPHA] Current node swap usage in bytes
# TYPE node_swap_usage_bytes gauge
node_swap_usage_bytes 1.8446743709127774e+19 1687950863599
# HELP pod_swap_usage_bytes [ALPHA] Current pod amount of swap usage in bytes
# TYPE pod_swap_usage_bytes gauge
pod_swap_usage_bytes{namespace="default",pod="test-pod"} 3.4379333632e+10 1687950858784
pod_swap_usage_bytes{namespace="kube-system",pod="coredns-8f5847b64-t9gmr"} 0 1687950863144
```

```shell
> kubectl get --raw "/api/v1/nodes/localhost/proxy/stats/summary"

node:
 nodeName: localhost
 swap:
   swapAvailableBytes: 407531442176
   swapUsageBytes: 18446743709127774000
pods:
- name: test-pod
  swapUsageBytes: 34379333632
  containers:
  - name: c1
    swapUsageBytes: 34400333824 
- name: coredns-8f5847b64-t9gmr
  swapUsageBytes: 0
  containers:
  - name: coredns
    swapUsageBytes: 0
```

(This output is simplified, for full examples please look at the description of: https://github.com/kubernetes/kubernetes/pull/118865)

#### Add swap support to NFD

Although not directly related to API changes since NFD is out-of-tree, bringing swap support to NFD is extremely
valuable and relevant.
It allows deferring the API changes discussion to the follow-up KEP that will focus on this subject specifically
(for more info on the scope of this KEP and the follow-up KEPs look at the summary section above).

With NFD, the end-user would be able to easily understand which nodes have swap enabled.
In the following example, only worker nodes 1 and 3 have swap enabled.
The user would be possible to easily check which nodes have swap enabled by performing the following:

```shell
> kubectl get nodes
NAME                    STATUS   ROLES           AGE   VERSION
k8s-dev-control-plane   Ready    control-plane   78s   v1.30.0
k8s-dev-worker1         Ready    <none>          66s   v1.30.0
k8s-dev-worker2         Ready    <none>          66s   v1.30.0
k8s-dev-worker3         Ready    <none>          66s   v1.30.0

> kubectl get nodes -o custom-columns=NAME:.metadata.name,LABELS:.metadata.labels | grep memory-swap:true | cut -d" " -f1
k8s-dev-worker1
k8s-dev-worker3
```

#### Swap Aware Eviction Manager API

We will introduce a new condition on the Node conditions that will notify admin of swap pressure.

```golang
type NodeConditionType string

const ( ...
	// NodeSwapPressure means the kubelet is under pressure due to insufficient swap memory.
	NodeSwapPressure NodeConditionType = "SwapPressure"
)
```

A condition goes with a taint so we will introduce a taint for swap also.

```golang
	// TaintNodeSwapPressure will be added when node has swap pressure
	// and removed when node has enough swap.
	TaintNodeSwapPressure = "node.kubernetes.io/swap-pressure"
```

The eviction manager will also have a new signal for swap.

```golang
	// SignalSwapMemoryAvailable is amount of swap available on the node
	SignalSwapMemoryAvailable Signal = "swap.available"
```

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

All existing tests needs to pass with and without swap enabled.

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

This KEP introduces minor additions of memory swap controlling configuration parameters.

- Kubelet configuration parameters are tested in the package `k8s.io/kubernetes/pkg/kubelet/apis/config/validation`
- Passing parameters to runtime is tested in `k8s.io/kubernetes/pkg/kubelet/kuberuntime`

Both packages has near 100% coverage and new functionality was covered.

In alpha2, tests will be extended in these packages to support kube-reserved swap settings.

##### Integration tests

<!--
This question should be filled when targeting a release.
For Alpha, describe what tests will be added to ensure proper quality of the enhancement.
For Beta and GA, add links to added tests together with links to k8s-triage for those tests:
https://storage.googleapis.com/k8s-triage/index.html
-->

NA.

These tasks require e2e test setup so we did not add any integration tests for this.

##### e2e tests

<!--
This question should be filled when targeting a release.
For Alpha, describe what tests will be added to ensure proper quality of the enhancement.
For Beta and GA, add links to added tests together with links to k8s-triage for those tests:
https://storage.googleapis.com/k8s-triage/index.html
We expect no non-infra related flakes in the last month as a GA graduation criteria.
-->

For alpha:

- Swap scenarios are enabled in test-infra for at least two Linux
  distributions. e2e suites will be run against them.
  - Container runtimes must be bumped in CI to use the new CRI.
- Data should be gathered from a number of use cases to guide beta graduation
  and further development efforts.
  - Focus should be on supported user stories as listed above.

Test grid tabs enabled:
- [kubelet-gce-e2e-swap-ubuntu](https://testgrid.k8s.io/sig-node-kubelet#kubelet-gce-e2e-swap-ubuntu): Green
- [kubelet-gce-e2e-swap-ubuntu-serial](https://testgrid.k8s.io/sig-node-kubelet#kubelet-gce-e2e-swap-ubuntu-serial): Green
- [kubelet-gce-e2e-swap-fedora](https://testgrid.k8s.io/sig-node-kubelet#kubelet-gce-e2e-swap-fedora): Green
- [kubelet-gce-e2e-swap-fedora-serial](https://testgrid.k8s.io/sig-node-kubelet#kubelet-gce-e2e-swap-fedora-serial): Green

No new e2e tests introduced.

For alpha2:

- Add e2e tests that exercise all available swap configurations via the CRI.
- Verify MemoryPressure behavior with swap enabled and document any changes
  for configuring eviction.
- Verify new system-reserved settings for swap memory.

For beta 1:

- Add e2e tests that verify pod-level control of swap utilization.
- Add e2e tests that verify swap performance with pods using a tmpfs.

For beta 2:

- Add Node-conformance tests for basic swap validation. To avoid disrupting node conformance lanes, only the
cgroup knobs are validated to be defined as expected with no real memory stress or swap use.

For beta 3:

- We want e2e tests that can confirm that eviction will take in account swap usage

For GA:

- Add a lane dedicated for swap testing, including stress tests and other tests that might be disruptive and intensive.
These lanes are called "swap-conformance", and are (and should remain) consistently green:
  - [kubelet-swap-conformance-fedora-serial](https://testgrid.k8s.io/sig-node-kubelet#kubelet-swap-conformance-fedora-serial): Green.
  - [kubelet-swap-conformance-ubuntu-serial](https://testgrid.k8s.io/sig-node-kubelet#kubelet-swap-conformance-ubuntu-serial): Green.

### Graduation Criteria

#### Alpha

- Kubelet can be started with swap enabled and will support two configurations
  for Kubernetes workloads: `LimitedSwap` and `NoSwap`.
- Kubelet can configure CRI to allocate swap to Kubernetes workloads. By
  default, workloads will not be allocated any swap.
- e2e test jobs are configured for Linux systems with swap enabled.

#### Alpha2

In alpha2 the focus will be on making sure that the feature can be used on
subset of production scenarios to collect more feedback before entering beta.
Specifically, security and test coverage will be increased. As well as the new
setting that will split swap between kubelet and workload will be introduced.

Once functionality part is resolved while in alpha, beta will be more about
performance and feedback on wider range of scenarios.

This will allow to collect feedback from the following scenarios reasonably safe:

- on cgroupv2: allow host system processes to use swap to increase
  system reliability under memory pressure.
- enable swap for the workload in "single large pod per node" scenarios.

Here are specific improvements to be made:

- Address swap impact on memory-backed volumes: https://github.com/kubernetes/kubernetes/issues/105978.
- Investigate swap security when enabling on system processes on the node.
- Improve coverage for appropriate scenarios in testgrid.
- Add the ability to set a system-reserved quantity of swap from what kubelet
  detects on the host.
- Consider introducing new configuration modes for swap, such as a node-wide
  swap limit for workloads.
- Investigate eviction behavior with swap enabled.

#### Beta 1

- Enable Swap Support using Burstable QoS Pods only.
- Enable Swap Support for Cgroup v2 Only.
- Add swap memory to the Kubelet stats api.
- Determine a set of metrics for node QoS in order to evaluate the performance
  of nodes with and without swap enabled.
- Make sure node e2e jobs that use swap are healthy
- Improve coverage for appropriate scenarios in testgrid.

#### Beta 2

- Publish a Kubernetes doc page encouraging users to use encrypted swap if they wish to enable this feature.
- Add [swap specific tests](https://github.com/kubernetes/kubernetes/issues/120798) such as, handling the usage of
  swap during container restart boundaries for writes to tmpfs (which may require pod cgroup change beyond what
  container runtime will do at (container cgroup boundary).
- Fix flaking/failing swap node e2e jobs.
- Address eviction related [issue](https://github.com/kubernetes/kubernetes/issues/120800) in swap implementation.
- Add `NoSwap` as the default setting.
- Remove `UnlimitedSwap` as a supported option.
- Add e2e test confirming that `NoSwap` will actually not swap
- Add e2e test confirming that swap is used for `LimitedSwap`.
- Document [best practices](#best-practices) for setting up Kubernetes with swap

#### Beta 3

- Enhance website documentation
  - Docs should be close to GA quality before promoting to GA.
- Make eviction manager swap aware.

#### GA

- Test a wide variety of scenarios that may be affected by swap support, including tests with aggressive memory stress.
- Address memory-backed backed volumes which should not have access to swap.  
- Remove feature gate.
- Exclude high-priority, static and mirrored pods from gaining access to swap.
- Add documentation regarding encrypted swap.
- Test a wide variety of scenarios in which the node is being memory-stressed with different eviction and swap configurations. 
Ensure the behavior is consistent and reliable.
- Use the presence of the e2e tests to inform documentation changes warning users about the behavior of eviction and swap
with different swap configurations and eviction thresholds.

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

No changes are required on upgrade to maintain previous behaviour.

It is possible to downgrade a kubelet on a node that was using swap, but this
would require disabling the use of swap and setting `swapoff` on the node.

### Version Skew Strategy

<!--
If applicable, how will the component handle version skew with other
components? What are the guarantees? Make sure this is in the test plan.

Consider the following in developing a version skew strategy for this
enhancement:
- Does this enhancement involve coordinating behavior in the control plane and
  in the kubelet? How does an n-2 kubelet without this feature available behave
  when this feature is used?
- Will any other components on the node change? For example, changes to CSI,
  CRI or CNI may require updating that component before the kubelet.
-->

Feature flag will apply to kubelet only, so version skew strategy is N/A.

## Production Readiness Review Questionnaire

<!--

Production readiness reviews are intended to ensure that features merging into
Kubernetes are observable, scalable and supportable; can be safely operated in
production environments, and can be disabled or rolled back in the event they
cause increased failures in production. See more in the PRR KEP at
https://git.k8s.io/enhancements/keps/sig-architecture/1194-prod-readiness.

The production readiness review questionnaire must be completed and approved
for the KEP to move to `implementable` status and be included in the release.

In some cases, the questions below should also have answers in `kep.yaml`. This
is to enable automation to verify the presence of the review, and to reduce review
burden and latency.

The KEP must have a approver from the
[`prod-readiness-approvers`](http://git.k8s.io/enhancements/OWNERS_ALIASES)
team. Please reach out on the
[#prod-readiness](https://kubernetes.slack.com/archives/CPNHUMN74) channel if
you need any help or guidance.
-->

### Feature Enablement and Rollback

<!--
This section must be completed when targeting alpha to a release.
-->

###### How can this feature be enabled / disabled in a live cluster?

<!--
Pick one of these and delete the rest.
-->

- [x] Feature gate (also fill in values in `kep.yaml`)
  - Feature gate name: NodeSwap
  - Components depending on the feature gate: API Server, Kubelet
- [x] Other
  - Describe the mechanism: `--fail-swap-on=false` flag for kubelet must also
    be set at kubelet start
  - Will enabling / disabling the feature require downtime of the control
    plane? Yes. Flag must be set on kubelet start. To disable, kubelet must be
    restarted. Hence, there would be brief control component downtime on a
    given node.
  - Will enabling / disabling the feature require downtime or reprovisioning
    of a node?
    Yes. See above; disabling would require brief node downtime.

###### Does enabling the feature change any default behavior?

<!--
Any change of default behavior may be surprising to users or break existing
automations, so be extremely careful here.
-->

No. If the feature flag is enabled, the user must still set
`--fail-swap-on=false` to adjust the default behaviour.
In addition, since the default "swap behavior" is "NoSwap",
by default containers would not be able to access swap. Instead,
the administrator would need to set a non-default behavior in order
for swap to be accessible.

A node must have swap provisioned and available for this feature to work. If
there is no swap available, but the feature flag is set to true, there will
still be no change in existing behaviour.

###### Can the feature be disabled once it has been enabled (i.e. can we roll back the enablement)?

<!--
Describe the consequences on existing workloads (e.g., if this is a runtime
feature, can it break the existing applications?).

NOTE: Also set `disable-supported` to `true` or `false` in `kep.yaml`.
-->

To turn this off, the kubelet would need to be restarted. If a cluster admin
wants to disable swap on the node without repartitioning the node, they could
stop the kubelet, set `swapoff` on the node, and restart the kubelet with
`--fail-swap-on=true`. The setting of the feature flag will be ignored in this
case.

In Beta2, we realize that we cannot rely on `--fail-swap-on=false`
as a flag for this feature. The flag predates this feature and it has
been used over time. We propose a configuration in `MemorySwap` called `NoSwap`.
Users could also set `NoSwap` in `MemorySwap` to disable all workloads from
using swap without requiring the user to disable swap if that is needed.

In Beta releases of this feature, one could use turn off `NodeSwap` feature toggle
but once this feature is GA, users could use another option to disable swap
for workloads.

###### What happens if we reenable the feature if it was previously rolled back?

As described above, swap can be turned on and off, although kubelet would need to be
restarted.

###### Are there any tests for feature enablement/disablement?

<!--
The e2e framework does not currently support enabling or disabling feature
gates. However, unit tests in each component dealing with managing data, created
with and without the feature, are necessary. At the very least, think about
conversion tests if API types are being modified.
-->

There are extensive tests to ensure that the swap feature as expected.

Unit tests are in place to test that this feature operates as expected with
cgroup v1/v2, the feature gate being on/off, and different swap behaviors defined.

In addition, node e2e tests are added and run as part of the node-conformance
suite. These tests ensure that the underlying cgroup knobs are being configured
as expected.

Furthermore, "swap-conformance" periodic lanes have been introduced for the purpose
testing swap on a stressed environment. These tests ensure that swap kicks in when
expected, tested while stressing both on the node-level and container-level. 

### Rollout, Upgrade and Rollback Planning

<!--
This section must be completed when targeting beta to a release.
-->

###### How can a rollout fail? Can it impact already running workloads?

<!--
Try to be as paranoid as possible - e.g., what if some components will restart
mid-rollout?
-->

If a new node with swap memory fails to come online, it will not impact any
running components.

It is possible that if a cluster administrator adds swap memory to an already
running node, and then performs an in-place upgrade, the new kubelet could fail
to start unless the configuration was modified to tolerate swap. However, we
would expect that if a cluster admin is adding swap to the node, they will also
update the kubelet's configuration to not fail with swap present.

Generally, it is considered best practice to add a swap memory partition at
node image/boot time and not provision it dynamically after a kubelet is
already running and reporting Ready on a node.

###### What specific metrics should inform a rollback?

<!--
What signals should users be paying attention to when the feature is young
that might indicate a serious problem?
-->

Workload churn or performance degradations on nodes. The metrics will be
application/use-case specific, but we can provide some suggestions, based on
the stability metrics identified earlier.

###### Were upgrade and rollback tested? Was the upgrade->downgrade->upgrade path tested?

<!--
Describe manual testing that was done and the outcomes.
Longer term, we may want to require automated upgrade/rollback tests, but we
are missing a bunch of machinery and tooling and can't do that now.
-->

N/A because swap support lacks a runtime upgrade/downgrade path; kubelet must
be restarted with or without swap support.

###### Is the rollout accompanied by any deprecations and/or removals of features, APIs, fields of API types, flags, etc.?

<!--
Even if applying deprecation policies, they may still surprise some users.
-->

No.

### Monitoring Requirements

<!--
This section must be completed when targeting beta to a release.
-->

###### How can someone using this feature know that it is working for their instance?

See #swap-metrics: available by both Summary API (/stats/summary) and Prometheus (/metrics/resource)
which provide how and if swap is utilized in the node, pod and container level.

###### How can an operator determine if the feature is in use by workloads?

<!--
Ideally, this should be a metric. Operations against the Kubernetes API (e.g.,
checking if there are objects with field X set) may be a last resort. Avoid
logs or events for this purpose.
-->

See #swap-metrics: available by both Summary API (/stats/summary) and Prometheus (/metrics/resource)
which provide how and if swap is utilized in the node, pod and container level.

KubeletConfiguration has set `failOnSwap: false`.

The prometheus `node_exporter` will also export stats on swap memory
utilization.

###### What are the SLIs (Service Level Indicators) an operator can use to determine the health of the service?

<!--
Pick one more of these and delete the rest.
-->

See #swap-metrics: available by both Summary API (/stats/summary) and Prometheus (/metrics/resource)
which provide how and if swap is utilized in the node, pod and container level.

- [X] Metrics
  - Metric names:
    - `container_swap_usage_bytes`
    - `pod_swap_usage_bytes`
    - `node_swap_usage_bytes`
    Components exposing the metric: `/metrics/resource` endpoint
  - Metric names:
    - `node.swap.swapUsageBytes`
    - `node.swap.swapAvailableBytes`
    - `node.systemContainers.swap.swapUsageBytes`
    - `pods[i].swap.swapUsageBytes`
    - `pods[i].containers[i].swap.swapUsageBytes`
    Components exposing the metric: `/stats/summary` endpoint
- [ ] Other (treat as last resort)
  - Details:

###### What are the reasonable SLOs (Service Level Objectives) for the above SLIs?

<!--
At a high level, this usually will be in the form of "high percentile of SLI
per day <= X". It's impossible to provide comprehensive guidance, but at the very
high level (needs more precise definitions) those may be things like:
  - per-day percentage of API calls finishing with 5XX errors <= 1%
  - 99% percentile over day of absolute value from (job creation time minus expected
    job creation time) for cron job <= 10%
  - 99,9% of /health requests per day finish with 200 code
-->

Swap is being managed by the kernel, depends on many factors and configurations
that are outside of kubelet's reach like the nature of the workloads running on the node,
swap capacity, memory capacity and other distro-specific configurations. However, generally:

- Nodes with swap enabled -> `node.swap.swapAvailableBytes` should be non-zero.
- Nodes with memory pressure -> `node.swap.swapUsageBytes` should be non-zero.
- Containers that reach their memory limit threshold -> `pods[i].containers[i].swap.swapUsageBytes` should be non-zero.
- Pods with containers that reach their memory limit threshold -> `pods[i].swap.swapUsageBytes` should be non-zero. 

###### Are there any missing metrics that would be useful to have to improve observability of this feature?

<!--
Describe the metrics themselves and the reasons why they weren't added (e.g., cost,
implementation difficulties, etc.).
-->

We added metrics to the node stats to report how much swap is used
and the capacity of swap.

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

No.

### Scalability

<!--
For alpha, this section is encouraged: reviewers should consider these questions
and attempt to answer them.

For beta, this section is required: reviewers must answer these questions.

For GA, this section is required: approvers should be able to confirm the
previous answers based on experience in the field.
-->

###### Can enabling / using this feature result in resource exhaustion of some node resources (PIDs, sockets, inodes, etc.)?

No.

###### Will enabling / using this feature result in any new API calls?

<!--
Describe them, providing:
  - API call type (e.g. PATCH pods)
  - estimated throughput
  - originating component(s) (e.g. Kubelet, Feature-X-controller)
Focusing mostly on:
  - components listing and/or watching resources they didn't before
  - API calls that may be triggered by changes of some Kubernetes resources
    (e.g. update of object X triggers new updates of object Y)
  - periodic API calls to reconcile state (e.g. periodic fetching state,
    heartbeats, leader election, etc.)
-->

No.

###### Will enabling / using this feature result in introducing new API types?

<!--
Describe them, providing:
  - API type
  - Supported number of objects per cluster
  - Supported number of objects per namespace (for namespace-scoped objects)
-->

No.

###### Will enabling / using this feature result in any new calls to the cloud provider?

<!--
Describe them, providing:
  - Which API(s):
  - Estimated increase:
-->

No.

###### Will enabling / using this feature result in increasing size or count of the existing API objects?

<!--
Describe them, providing:
  - API type(s):
  - Estimated increase in size: (e.g., new annotation of size 32B)
  - Estimated amount of new objects: (e.g., new Object X for every existing Pod)
-->

The KubeletConfig API object may slightly increase in size due to new config
fields.

###### Will enabling / using this feature result in increasing time taken by any operations covered by existing SLIs/SLOs?

<!--
Look at the [existing SLIs/SLOs].

Think about adding additional work or introducing new steps in between
(e.g. need to do X to start a container), etc. Please describe the details.

[existing SLIs/SLOs]: https://git.k8s.io/community/sig-scalability/slos/slos.md#kubernetes-slisslos
-->

Yes, enabling swap can affect performance of other critical daemons on the system.
Any scenario where swap memory gets utilized is a result of system running out of physical RAM,
or a container reaching its memory limit threshold.
Hence, to maintain the SLIs/SLOs of critical daemons on the node we highly recommend to disable the swap for the system.slice
along with reserving adequate enough system reserved memory, giving io latency precedence to the system.slice, and more.
See #best practices for more info.

The SLI that could potentially be impacted is [pod startup latency](https://github.com/kubernetes/community/blob/master/sig-scalability/slos/pod_startup_latency.md).
If the container runtime or kubelet are performing slower than expected, pod startup latency would be impacted.
In addition to this SLI, general areas around pod lifecycle (image pulls, sandbox creation, storage) could become slow.

###### Will enabling / using this feature result in non-negligible increase of resource usage (CPU, RAM, disk, IO, ...) in any components?

<!--
Things to keep in mind include: additional in-memory state, additional
non-trivial computations, excessive access to disks (including increased log
volume), significant amount of data sent and/or received over network, etc.
This through this both in small and large cases, again with respect to the
[supported limits].

[supported limits]: https://git.k8s.io/community//sig-scalability/configs-and-limits/thresholds.md
-->

Yes. It will permit the utilization of swap memory (i.e. disk) on nodes. This
is expected, as this enhancement is enabling cluster administrators to access
this resource.

### Troubleshooting

<!--
This section must be completed when targeting beta to a release.

The Troubleshooting section currently serves the `Playbook` role. We may consider
splitting it into a dedicated `Playbook` document (potentially with some monitoring
details). For now, we leave it here.
-->

###### How does this feature react if the API server and/or etcd is unavailable?

No change. Feature is specific to individual nodes.

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


Individual nodes with swap memory enabled may experience performance
degradations under load. This could potentially cause a cascading failure on
nodes without swap: if nodes with swap fail Ready checks, workloads may be
rescheduled en masse.

Thus, cluster administrators should be careful while enabling swap. To minimize
disruption, you may want to taint nodes with swap available to protect against
this problem. Taints will ensure that workloads which tolerate swap will not
spill onto nodes without swap under load.

###### What steps should be taken if SLOs are not being met to determine the problem?

It is suggested that if nodes with swap memory enabled cause performance or
stability degradations, those nodes are cordoned, drained, and replaced with
nodes that do not use swap memory.

## Implementation History

This is a partial list of everything that was done, but contains the most significant implementations.

- **2015-04-24:** Discussed in [#7294](https://github.com/kubernetes/kubernetes/issues/7294).
- **2017-10-06:** Discussed in [#53533](https://github.com/kubernetes/kubernetes/issues/53533).
- **2021-01-05:** Initial design discussion document for swap support and use cases.
- **2021-04-05:** Alpha KEP drafted for initial node-level swap support and implementation (KEP-2400).
- **2021-08-09:** New in Kubernetes v1.22: alpha support for using swap memory: https://kubernetes.io/blog/2021/08/09/run-nodes-with-swap-alpha/.
- **2023-04-17:** KEP update for beta1 [#3957](https://github.com/kubernetes/enhancements/pull/3957).
- **2023-07-18:** Add full cgroup v2 swap support with automatically calculated swap limit for LimitedSwap [#118764](https://github.com/kubernetes/kubernetes/pull/118764).
- **2023-07-18:** Add swap to stats to Summary API and Prometheus endpoints (/stats/summary and /metrics/resource) [#118865](https://github.com/kubernetes/kubernetes/pull/118865).
- **2023-08-15:** Beta1 released in kubernetes 1.28
- **2024-01-12:** Updates to Beta2 KEP.
- **2024-01-08:** Beta2 released in kubernetes 1.30.
- **2024-03-06:** Add no swap as the default option for swap [#122745](https://github.com/kubernetes/kubernetes/pull/122745).
- **2024-03-14:** Add swap-specific (a.k.a. swap conformance) test lanes [#32263](https://github.com/kubernetes/test-infra/pull/32263).
- **2024-05-21:** Add swap serial stress tests, improve NodeConformance tests and adapt NoSwap behavior [#123557](https://github.com/kubernetes/kubernetes/pull/123557).
- **2024-05-23:** Mount tmpfs memory-backed volumes with a noswap option if supported [#124060](https://github.com/kubernetes/kubernetes/pull/124060).
- **2024-07-22:** Restrict access to swap for containers in high priority Pods [#125277](https://github.com/kubernetes/kubernetes/pull/125277).
- **2024-08-28:** Updates to KEP, GA requirements and intention to release in version 1.32.
- **2024-10-15:** Updates to KEPs to reflect eviction and extensions.

## Drawbacks

<!--
Why should this KEP _not_ be implemented?
-->

When swap is enabled, particularly for workloads, the kubelet’s resource
accounting may become much less accurate. This may make cluster administration
more difficult and less predictable.

In general, swap is less predictable and might cause performance degradation.
It also might be hard in certain scenarios to understand why certain workloads
are the chosen candidates for swapping, which could occur for reasons external
to the workload.

In addition, containers with memory limits would be killed less frequently
since with swap enabled the kernel can usually reclaim a lot more memory.
While this can help to avoid crashes, it could also "hide a problem" of a container
reaching its memory limits.

## Alternatives

### Just set `--fail-swap-on=false`

When `--fail-swap-on=false` is provided to Kubelet but swap is not configured
otherwise it is guaranteed that, by default, no Kubernetes workloads would
be able to utilize swap. However, everything outside of kubelet's reach
(e.g. system daemons, kubelet, etc) would be able to use swap.

## Infrastructure Needed (Optional)

<!--
Use this section if you need things from the project/SIG. Examples include a
new subproject, repos requested, or GitHub details. Listing these here allows a
SIG to get the process for these resources started right away.
-->

Added the "swap-conformance" lane for extensive swap testing under node pressure: [kubelet-swap-conformance-fedora-serial](https://testgrid.k8s.io/sig-node-kubelet#kubelet-swap-conformance-fedora-serial),
kubelet-swap-conformance-ubuntu-serial](https://testgrid.k8s.io/sig-node-kubelet#kubelet-swap-conformance-ubuntu-serial).

See #e2e tests above for more information
