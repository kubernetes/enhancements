---
title: shared-container-limits-in-burstable-pod
authors:
  - "@liorokman"
owning-sig: sig-node
participating-sigs:
reviewers:
  - TBD
approvers:
  - TBD
creation-date: 2020-03-03
last-updated: 2020-03-03
status: proposed
---

# Shared Container Limits in Burstable Pods

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
  - [Risks and Mitigations](#risks-and-mitigations)
- [Design Details](#design-details)
- [Implementation History](#implementation-history)
<!-- /toc -->

## Release Signoff Checklist

For enhancements that make changes to code or processes/procedures in core Kubernetes i.e., [kubernetes/kubernetes], we require the following Release Signoff checklist to be completed.

Check these off as they are completed for the Release Team to track. These checklist items _must_ be updated for the enhancement to be released.

- [ ] kubernetes/enhancements issue in release milestone, which links to KEP (this should be a link to the KEP location in kubernetes/enhancements, not the initial KEP PR)
- [ ] KEP approvers have set the KEP status to `implementable`
- [ ] Design details are appropriately documented
- [ ] Test plan is in place, giving consideration to SIG Architecture and SIG Testing input
- [ ] Graduation criteria is in place
- [ ] "Implementation History" section is up-to-date for milestone
- [ ] User-facing documentation has been created in [kubernetes/website], for publication to [kubernetes.io]
- [ ] Supporting documentation e.g., additional design documents, links to mailing list discussions/SIG meetings, relevant PRs/issues, release notes

**Note:** This checklist is iterative and should be reviewed and updated every time this enhancement is being considered for a milestone.

[kubernetes.io]: https://kubernetes.io/
[kubernetes/enhancements]: https://github.com/kubernetes/enhancements/issues
[kubernetes/kubernetes]: https://github.com/kubernetes/kubernetes
[kubernetes/website]: https://github.com/kubernetes/website

## Summary


Pod resources are currently enforced on a container-by-container case. Each container defines a limit for each managed resource, and the Kubelet translates these limits into the appropriate cgroup definitions. Kubelet creates a three-level deep hierarchy of cgroups for each pod. The top-level is the QoS grouping of the pod (`Guaranteed`, `Burstable`, and `BestEffort`), the second level is the pod itself, and the bottom level are the pod containers. The current Pod API doesn't enable developers to define resource limits at the pod level. This KEP proposes a method for developers to define resource limits on the pod level in addition to the resource limits currently possible on the individual container level.

## Motivation

Some workloads are deployed as pods which are comprised of multiple sidecar containers which are strongly coupled in terms of their task. Such containers communicate either across a shared filesystem, or the localhost network, and orchestrate some common task. 

For example, consider the following Pod with the following structure:

<pre>
  pod
   |
   +-- container1 (main task)
   |
   +-- container2 (second level task)
   |
   +-- container3 (log handler)
   |
   +-- container4 (mesh sidecar)
</pre>

In some cases deploying a single container with all the tasks is not optimal and not always possible. Kubernetes is not aware of these tasks, and doesn't monitor them for failure, and is not able to manage the resources (cgroups) for each of them. By separating the different tasks to their own containers, the application is able to leverage Kubernetes to monitor the tasks. However, splitting a pod into multiple containers also requires a developer to split the resources allocated to the entire pod into slices allocated to each container. For some workloads this is hard to get right, requiring the developer to over-allocate resources to specific containers because the containers can't share their resources easily. 

This proposal suggests a middle ground, and suggests a way to make it possible to describe to Kubernetes how to limit the resource consumption of multiple containers in the pod at the Pod level, instead of trying to micro-manage the resource limits on the container level. For workloads where the work performed is burstable, this proposal would make it easier to allow the low-level mechanisms available in the underlying operating-system to manage the resources required for the task.

### Goals

This proposal aims to:

* Allow developers to define resource limits on the pod level in addition to the individual container level.
* Use as much as possible the Linux kernel's cgroup ability to define limits in hierarchies to enforce the defined limits.

### Non-Goals

*  Providing a general-purpose interface to the full range of possible resource management provided by the Linux cgroup hierarchy.

## Proposal

The Pod QoS enhancement already implemented in Kubernetes manages resources as a hieararchy of cgroups in the following way:

<pre>
  kubepods
   |
   +-- Guaranteed-pod0
   |   |
   |   +-- container0 (pause container)
   |   |
   |   +-- container1 (first container)
   |   |
   |  = ... =
   |   |
   |   +-- containerN (N-th container)
   |
   +-- QoS CGroup (one of burstable, or besteffort)
       |
       \ pod 
          |
          +-- container0 (pause container)
          |
          +-- container1 (first container)
          |
        = ... =
          |
          +-- containerN (N-th container)
</pre>

The current implementation will set the cgroup limits for the memory/CPU resources on the pod level of the hierarchy only in the following cases:
1. The QoS level is `Guaranteed`. 
1. The QoS level is `Burstable` and all of the containers specify a limit for the relevant resource. 

For the current definition of the `Guaranteed` QoS level this proposal would not make any modifications. Since each container is provided with its
own limit, defining a pod-level limit is redundant. If the pod level limit request is smaller than the sum of limit for all the containers, then 
the limit requested by the containers is unreachable. Conversely if the pod level limit request is larger than the sum of the limit for all the 
containers then the excess is not relevant since the individual containers can never use more than their requested share, and even if all of the 
containers use the maximum allowed by their defined limit, it can never amount to the pod-level limit.

For pods in the `Burstable` and `BestEffort` QoS levels specifying a resource limit on the pod level means that resources can be shared across containers.
Sharing resources across containers is possible only if individual containers release the resources that are not being used. For CPU this is trivial.
For memory, this depends on the application being run actually releasing the memory back to the operating system. See [below](#Memory-Reuse-Across-cgroups)
for an analysis.

The proposal in this KEP is to allow users to define resources on the Pod level itself. Only limits will be allowed on this level so that the effect on
the scheduler algorithm will be minimal. The scheduler will continue to use the resource requests defined on the container level as its input. Kubelet 
will configure the pod-level cgroups as defined on the pod level. If the pod level doesn't include any resource limits, then kubelet will continue to
function as it does today.

For example, consider the following Pod:

```yaml
apiVersion: v1
kind: Pod
metadata:
  labels:
    run: nginx
  name: nginx
spec:
  resources:
    limits:
      memory: 384M
      cpu: "2"
  containers:
  - image: proxy
    name: envoy:latest
  - name: nginx
    image: nginx:latest
    command: [ "/usr/bin/tail", "-f", "/dev/null"] 
    resources:
      requests:
        memory: 128M
        cpu: "0.5"
      limits:
        memory: 256M
        cpu: "1"
```

The cgroup hierarchy for each of these resources (memory and cpu) would be this:

<pre>
  QoS CGroup (one of guaranteed, burstable, or besteffort)
   |
   \ pod (memory: 384M limit, CPU: 2 core)
      |
      +-- container0 (pause container, memory: unlimited, cpu: unlimited quota)
      |
      +-- container1 (proxy container, memory: unlimited, CPU: unlimited quota)
      |
      +-- container2 (nginx container, memory: 256M limit, CPU: 1 core)
</pre>

This has the following effect:
1. The nginx container will continue to be constrained to at most 256M of memory and 1 CPU core.
1. The Proxy container will be limited to the amount of resources specified on the Pod cgroup level - no more than 384M memory and 2 CPU core.
1. The pause container will not be affected since it doesn't use any resources anyways.

Because the Linux cgroup implementation doesn't allocate resources to a specific cgroup before the processes in that cgroup actually use the resource,
it is possible for the nginx container to try to allocate the memory too late - after one of the other containers has already allocated more than 128M 
of RAM. In this case, even though the nginx container is technically allowed to allocate more memory, the allocation will fail. This behavior is consistent
with the way Kubernetes acts today as well, even without pod level resources; if a containers with defined resource `requests` can still fail to allocate
the resources if other noisy neighbors are lucky enough to allocate the resources faster.

### Memory Reuse Across cgroups

The memory resource deserves a special discussion due to memory being a non-compressible resource.

The Linux kernel allows sibling cgroups to use memory which was released by a different sibling cgroup. The main difficulty here is understanding the term **released**. It is not enough for a programmer to free the memory, the funtime being used to develop the program must also release the memory back to the kernel. Since allocating memory to a process requires a context switch between userspace and kernelspace, most runtimes cache memory which was allocated to the process, and attempt to reuse memory that the program released so as to minimize the number of memory allocations required.

Different runtimes have different heuristics around the best time to release memory. Here are a few popular runtimes:

* GLibc releases memory opportunisticly and only for large amounts. See the free algorithm used by GLibc: https://sourceware.org/glibc/wiki/MallocInternals#line-286 
* The MUSL libc (used by Alpine) will use the `madvise` system call to [mark freed memory](https://git.musl-libc.org/cgit/musl/tree/src/malloc/malloc.c#n499) ranges with `MADV_DONTNEED`. This allows memory to be reclaimed from the process when memory pressure exists. In this case, memory can be utilized by sibling cgroups.
* Golang uses the `madvise` system call to mark [unused](https://github.com/golang/go/blob/master/src/runtime/malloc.go#L405) memory as such during garbage collector runs. Memory that is marked as unused can move between sibling cgroups, based on memory pressure.
* The OpenJDK Java virtual machine uses the standard free function. It is based on the standard C/C++ library, and calls the standard `free()` function, and as such will work as appropriate for the distribution it is installed on (glibc or musl).
* NodeJS will call `madvise` if the memory reduction feature is used. Follow the code starting at https://github.com/nodejs/node/blob/master/deps/v8/src/heap/sweeper.cc#L322 


### User Stories 

#### Story 1

A development environment implemented as a Kubernetes Pod allows for separation of tools and a (web-based) IDE between multiple side-cars.

The development environment defines a contantainer with the web-server serving the IDE itself, and constrains it to use a certain amount of memory. Additional tools are provided in additional side-car deployments - for example an [LSP](https://langserver.org/) service, a terminal, and more. 

Using this new feature, the containers providing the terminal, the LSP services, and the set of tools being utilizied can share the resource limit defined for the pod. Consider the following pod definition:

```yaml
apiVersion: v1
kind: Pod
metadata:
  labels:
    run: ide
  name: myide
spec:
  resources:
    limits:
      memory: "1024M"
      cpu: "4"
  containers:
  - image: shell
    name: debian:buster
  - image: tool1
    name: first-tool:latest
  - image: tool2
    name: second-tool:latest
  - name: ide
    image: theia:latest
    resources:
      requests:
        memory: 128M
        cpu: "0.5"
      limits:
        memory: 256M
        cpu: "1"
```

Using the pod-level resource definition enables the `tool1` and `tool2` containers to be constrained by the total limit for the pod. 

Without this feature, the developer would need to decide a-priori how much resources to allocate to the tools - and this is not easy to do for this workload.

### Implementation Details/Notes/Constraints 

The proposal is an **opt-in** feature, and will have no effect on existing deployments. Only deployments that explicitly require this functionality should turn it on by specifying the relevant resource section in the Pod specification.

### Interactions With Other Features

#### Support for cgroup v2

This proposal is compatible with both cgroup v1 and cgroup v2. Both versions of cgroup allow specifying limits on all levels of the cgroup hierarchy.

#### ResourceOverhead

As described in the [20190226-pod-overhead](https://github.com/kubernetes/enhancements/blob/6acd3e16806e98fa8545ecf57b02e90384c4bf55/keps/sig-node/20190226-pod-overhead.md) KEP, the pod overhead must be kept separate from the pod resource requirements. However, instead of enforcing the pod overhead on the pod-level cgroup, the pod overhead resources should be assigned to the `pause` container instead, and provided as a `Request` and not as a `Limit`. Since the pause container binary doesn't actually use any resources, doing this would guarentee that the resources required for the pod as overhead are not actually utilized by any other container and are indeed accounted for by Kubelet as reserved for the `RuntimeClass` defined overhead.

#### Memory-based Emptydir volumes

This proposal doesn't change the current status quo in any way. Memory used for files in a `tmpfs` volume is accounted for as shared-memory. The Linux kernel charges the used memory to cgroup that first touched the shared page for the memory. Moving the charge for shared memory pages between cgroups is not currently supported by the Linux kernel. Defining a memory limit on the pod cgroup level would not change the underlying limitation that memory used by files in the volume are accounted to the cgroup that first touched the memory.

#### HugeTLB cgroup

In progress

### PID, and other cgroups

These are out of scope for this proposal. This proposal doesn't remove the ability to set these limits and requests on the container level, if this was already supported by Kubernetes before.

#### Init Containers

The current implementation (before this proposal) sets up the pod-level cgroup using the maximum between the sum of all the sidecars or the biggest limit for any single Init container. 

There are a few options: 
1. Limit the pod-level cgroup only after the `InitContainers` have successfully terminated
1. Require the pod-level cgroup (if it exists) to apply to the `InitContainers` as well.
1. Allow specifying pod-level restrictions separately for Init containers and sidecars.

Since init containers run sequentially, there is no requirement to share resources with any other container. Therefore the first option is the best - the pod-level cgroup should be restricted only after all of the init containers have finished running.

#### NUMA architectures

There should be no issue caused by limiting a non-leaf cgroup for a pod running in a NUMA environment. The Linux cgroup allows memory from any NUMA node to be used in the same cgroup. See section 5.6 of the Lunux memory cgroup [documentation](https://www.kernel.org/doc/Documentation/cgroup-v1/memory.txt).

### Risks and Mitigations

Since this is an **opt-in** feature, there should be no risk to merging the feature. Users which don't use it will not be affected by it.

When users opt to use the feature, the workload must be able to run in a potentially resource-limited environment. 

## Design Details

1. Add a `Resources` section in the `PodSpec` structure.
1. Add a parameter to the pod cgroup setup procedure to distinguish between the lifecycle phase of the pod when `InitContainers` are being run and afterwards.
1. In the initialization lifecycle phase setup the pod-level cgroup should not consider the pod-level definition.
1. Once the init containers have finished running, the pod-level cgroup limits should be established as per the `Resources` section in the `PodSpec` structure.
1. In the memory cgroup configuration, in addition to the `memory.limit_in_bytes` field which is set to the limit specified in each container, the `memory.soft_limit_in_bytes` field should also
   be set to the `request` specified in the container. This field provides some extra information to the memory cgroup controller that long-running processes might benefit from.



## Implementation History

- 2020-03-04 - v1 of the proposal 
- 2020-03-06 - Updates due to suggested review
- 2020-03-15 - More updates due to suggested review
- 2020-03-17 - Reworked the proposal based on the suggested reviews


