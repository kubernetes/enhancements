---
title: pod-level-resource-limits
authors:
- "@liorokman"
owning-sig: sig-node
participating-sigs:
reviewers:
- "@thockin"
approvers:
- sig-node
creation-date: 2020-03-03
last-updated: 2020-09-22
---

# Pod-Level Resource Limits

## Table of Contents

<!-- toc -->
- [Release Signoff Checklist](#release-signoff-checklist)
- [Summary](#summary)
- [Motivation](#motivation)
  - [Goals](#goals)
  - [Non-Goals](#non-goals)
- [Proposal](#proposal)
  - [Memory Reuse Across cgroups](#memory-reuse-across-cgroups)
  - [User Stories](#user-stories)
    - [Story 1](#story-1)
  - [Implementation Details/Notes/Constraints](#implementation-detailsnotesconstraints)
  - [Interactions With Other Features](#interactions-with-other-features)
    - [Support for cgroup v2](#support-for-cgroup-v2)
    - [ResourceOverhead](#resourceoverhead)
    - [Memory-based Emptydir volumes](#memory-based-emptydir-volumes)
    - [HugeTLB cgroup](#hugetlb-cgroup)
  - [PID, and other cgroups](#pid-and-other-cgroups)
    - [Init Containers](#init-containers)
    - [NUMA architectures](#numa-architectures)
  - [Risks and Mitigations](#risks-and-mitigations)
- [Design Details](#design-details)
- [Proof of concept](#proof-of-concept)
  - [Proof of Concept Architecture](#proof-of-concept-architecture)
  - [Test Workload](#test-workload)
  - [Test Results](#test-results)
    - [Node.JS](#nodejs)
    - [Java](#java)
    - [General Load Results](#general-load-results)
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

Pod resources are currently enforced on a container-by-container case. Each container defines a limit for each managed resource, and the Kubelet translates these limits into the appropriate cgroup definitions. Kubelet creates a three-level deep hierarchy of cgroups for each pod. The top-level is the QoS grouping of the pod (`Guaranteed`, `Burstable`, and `BestEffort`), the second level is the pod itself, and the bottom level are the pod containers. The current Pod API doesn't enable developers to define resource limits at the pod level. This KEP proposes a method for developers to define resource requests and limits on the pod level in addition to the resource requests and limits currently possible on the individual container level.

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

The Pod QoS enhancement already implemented in Kubernetes manages resources as a hierarchy of cgroups in the following way:

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

The proposal in this KEP is to allow users to define resources on the Pod level itself. Both resource requests and limits are allowed. The scheduler
algorithm will consider the pod-level resources if they exist in preference to the container-level resources. If no pod-level resource section is provided,
then the scheduler will continue to operate as it does in the current implementation. The QoS definitions will be updated to consider the pod-level resources
when classifying a pod into one of the QoS levels. If the pod level doesn't include any resource limits, then kubelet will continue to function as it does today.

The current implementation will set the cgroup limits for the memory/CPU resources on the pod level of the hierarchy only in the following cases:
1. The QoS level is `Guaranteed`. 
1. The QoS level is `Burstable` and all of the containers specify a limit for the relevant resource. 

In the current implementation pods are considered `Guaranteed` if and only if all of the included containers define a resource request which is equal
to the resource limit. The current implementation considers a pod to be `Burstable` if at least one container defines resource requests or limits. Any
pods not matching the `Guaranteed` or `Burstable` QoS definitions are placed in the `BestEffort` QoS level.

This proposal suggests modifying these definitions the following: 

A pod is considered `Guaranteed` if either of the following two conditions apply: 
1. All of the containers included in the pod define memory and CPU resource requests equal to resource limits
1. The pod includes a pod-level memory and CPU resource request which is equal to the resource limit, even if one or more of the included containers doesn't define either a CPU or a memory request or limit.

A pod is considered `Burstable` if at least one container has a resource request, or the pod itself has a resource request.

A pod is considered `BestEffort` if there are not resource request or limits settings on any container or on the pod level itself.

If a developer included a request and limit definition at the pod level, the developer's intent was for Kubernetes to attempt to guarantee resources 
as a whole, for all the containers included in the pod. This should apply even if the developer is unable to preemptively split the resources correctly
between all the containers contained in the pod. 

Note that if a pod belongs to the `Guaranteed` QoS level as per the current definition, then defining a pod-level limit is redundant. If the pod level limit
request is smaller than the sum of limit for all the containers, then the limit requested by the containers is unreachable. Conversely if the pod level 
limit request is larger than the sum of the limit for all the containers then the excess is not relevant since the individual containers can never use more 
than their requested share, and even if all of the containers use the maximum allowed by their defined limit, it can never amount to the pod-level limit.

If a there are resource request definition on the pod level, even without any definition on any of the included containers, then it should be considered
`Burstable` since there is at least some knowledge as to the amount of resources the pod should receive. 

Sharing resources across containers is possible only if individual containers release the resources that are not being used. For CPU this is trivial.
For memory, this depends on the application being run actually releasing the memory back to the operating system. See [below](#Memory-Reuse-Across-cgroups)
for an analysis.

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

The Linux kernel allows sibling cgroups to use memory which was released by a different sibling cgroup. The main difficulty here is the meaning of the term **released**. It is not enough for a programmer to free the memory, the runtime being used to develop the program must also release the memory back to the kernel. Since allocating memory to a process requires a context switch between userspace and kernelspace, most runtimes cache memory which was allocated to the process, and attempt to reuse memory that the program released so as to minimize the number of memory allocations required.

Different runtimes have different heuristics around the best time to release memory. Here are a few popular runtimes:

* GLibc releases memory opportunistically and only for large amounts. See the free algorithm used by GLibc: https://sourceware.org/glibc/wiki/MallocInternals#line-286 
* The MUSL libc (used by Alpine) will use the `madvise` system call to [mark freed memory](https://git.musl-libc.org/cgit/musl/tree/src/malloc/malloc.c#n499) ranges with `MADV_DONTNEED`. This allows memory to be reclaimed from the process when memory pressure exists. In this case, memory can be utilized by sibling cgroups.
* Golang uses the `madvise` system call to mark [unused](https://github.com/golang/go/blob/master/src/runtime/malloc.go#L405) memory as such during garbage collector runs. Memory that is marked as unused can move between sibling cgroups, based on memory pressure.
* The OpenJDK Java virtual machine uses the standard free function. It is based on the standard C/C++ library, and calls the standard `free()` function, and as such will work as appropriate for the distribution it is installed on (glibc or musl).
* NodeJS will call `madvise` if the memory reduction feature is used. Follow the code starting at https://github.com/nodejs/node/blob/master/deps/v8/src/heap/sweeper.cc#L322 

The Java runtime auto-configures some garbage collector sizes based on the amount of memory available to it. Since Java 9, the JDK is cgroup aware and will correctly use the amount of memory available to the cgroup that it is running in to perform this initialization. The OpenJDK implementation uses the `hierarchical_memory_limit` field in its `memory.stat` file to determine the amount of memory that is available to it. This field contains the maximum calculated amount of memory available to the cgroup based on the limits set on the entire memory controller hierarchy.

When a process exits, all of the memory that was allocated to it is released back to the operating system and can be used by other processes. In some workloads, tasks are executed by forking and exec-ing child processes from some main process. In this case the memory being used by a short-lived process is made available to the rest of the pod immediately when the short-lived process exits.

### User Stories 

#### Story 1

A development environment implemented as a Kubernetes Pod allows for separation of tools and a (web-based) IDE between multiple side-cars.

The development environment defines a container with the web-server serving the IDE itself, and constrains it to use a certain amount of memory. Additional tools are provided in additional side-car deployments - for example an [LSP](https://langserver.org/) service, a terminal, and more. 

Using this new feature, the containers providing the terminal, the LSP services, and the set of tools being utilized can share the resource limit defined for the pod. Consider the following pod definition:

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

Since the runtime shim is started in the pod-level cgroup, there should be no bad interactions between this proposed feature and the ResourceOverhead feature.

#### Memory-based Emptydir volumes

This proposal doesn't change the current status quo in any way. Memory used for files in a `tmpfs` volume is accounted for as shared-memory. The Linux kernel charges the used memory to cgroup that first touched the shared page for the memory. Moving the charge for shared memory pages between cgroups is not currently supported by the Linux kernel. Defining a memory limit on the pod cgroup level would not change the underlying limitation that memory used by files in the volume are accounted to the cgroup that first touched the memory.

#### HugeTLB cgroup

The hugetlb cgroup is similar to the `memory` cgroup, except that in contrast to memory where unset means unlimited, when `hugetlb` is unset it is essentially the same as enforcing 0. 

The behavior suggested by this KEP then becomes:

1. If no pod-level limits are specified for `hugetlb`, make no change to the current implementation.
1. If pod-level limits are set, apply them at the pod-level cgroup.
    1. for each container in the pod, 
         1. if container-level limits are set, use the values as provided
         1. if container-level limits are NOT set, then set the container-level cgroup to unlimited.

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


The proposed implementation would be along these lines:
1. Add a `Resources` section in the `PodSpec` structure.
1. Update the scheduler to prefer the pod-level resources section to the aggregation of container level resource sections.
1. Add a parameter to the pod cgroup setup procedure to distinguish between the lifecycle phase of the pod when `InitContainers` are being run and afterwards.
1. In the initialization lifecycle phase setup the pod-level cgroup should not consider the pod-level definition.
1. Once the init containers have finished running, the pod-level cgroup limits should be established as per the `Resources` section in the `PodSpec` structure.
1. In the memory cgroup configuration, in addition to the `memory.limit_in_bytes` field which is set to the limit specified in each container, the `memory.soft_limit_in_bytes` field should also
   be set to the `request` specified in the container. This field provides some extra information to the memory cgroup controller that long-running processes might benefit from.

## Proof of concept

As per the recommendation given in the SIG-Node meeting on 2020-03-17, a proof-of-concept was developed to allow assessing if setting pod-level limits provides any benefits.

### Proof of Concept Architecture

The PoC (available [here](https://github.com/liorokman/terminus)) implements a `DaemonSet` that watches `Pod` resources on each node. For each `Pod` that is scheduled to run on the same host as the current worker, if the `Pod` is annotated with pod-level limits, the `DaemonSet` worker modifies the pod-level cgroup as outlined in this proposal. 

### Test Workload

The workload with which this concept was tested is a real-world workload currently used by the [SAP Business Application Studio](https://community.sap.com/topics/business-application-studio) product. The SAP Business Application Studio is a powerful and modern development environment, tailored for efficient development of business applications for the Intelligent Enterprise. Available as a cloud service, SAP Business Application Studio provides desktop-like experience similar to leading IDEs with command line, integrated debugging and optimized code editors. At the heart of SAP business Application Studio are the Dev-Spaces, which are like isolated virtual machines in the cloud containing tailored tools and pre-installed runtimes per business scenario, such as: SAP Fiori, SAP S/4 HANA extensions and more. In SAP Business Application Studio, each Dev-Space is a Kubernetes Pod. The Pod is subdivided (to allow for extensibility) into multiple containers, where each container provides some part of the toolset available to the developer. For example, one container provides the web-based IDE, another provides a full Java toolset, a third provides Node.js tools and runtimes, and so on. 

The tested scenario configuration was:
1. A single `m5.2xlarge` AWS node (8 cores, 32 GB RAM) dedicated to run only the test pods. 
1. 10 Dev-Spaces were launched on the node, each configured to use no more than 3 CPU cores, and 8 GB or RAM. Half of the Dev-spaces were configured for Java development, the other half were configured for Node.JS development.
1. The main test loop ran the following loop for 30 minutes in all of the pods on the node:
  1. Build the project
  1. Sleep for 30 seconds


The project being built was a typical project for SAP workloads, based on the [SAP CDS](https://help.sap.com/viewer/65de2977205c403bbc107264b8eccf4b/Cloud/en-US/855e00bd559742a3b8276fbed4af1008.html) framework, in both Java and Node.JS variants. 

The test was run in two ways:
1. Without the Terminus DaemonSet. The container limits were set to split the CPU and Memory resources evenly between the containers providing the various development tools.
1. With the Terminus DaemonSet. The Pod was configured to have the entire budget (3 cores, 8 GB Ram) allocated at the Pod level, and no limits were set on the container level. 

In both cases, the overall node CPU utilization was monitored, and the response time for each build operation was monitored.

### Test Results

#### Node.JS

The Node.JS project uses the NPM tool to fetch dependencies, and then uses the CDS tool to watch for changes in the source files which trigger rebuilding the project. The project was then run locally using `cds deploy` backed with an SQLite3 database, and packaged to a [Multi-Target Application](https://help.sap.com/viewer/65de2977205c403bbc107264b8eccf4b/Cloud/en-US/d04fc0e2ad894545aebfd7126384307c.html) using the [MBT](https://sap.github.io/cloud-mta-build-tool/) tool.

The `npm install` operation was run once before the entire scenario was run.

| Operation     | Response time (sec) w/o pod-level limits | Response time (sec) with pod-level limits | 
|    :---:      |             :---:                        |        :---:                              |
| `npm install` |   60  |  43   |
| `cds watch`   |   5   |  2    |
| `cds deploy`  |   2   |  0.8  | 
| `mbt build`   |  90   |  29   |

#### Java

The Java project uses Maven to build and run the project.

A setup operation of `mvn clean install` was performed once, and then the project was either run directly as via `mvn`, or via `cds`.

| Operation           | Response time (sec) w/o pod-level limits | Response time (sec) with pod-level limits | 
|    :---:            |             :---:                        |            :---:                      |
| `mvn clean install` | 85 | 42   |
| `mvn run`           | 50 | 13.5 |
| `cds deploy`        | 2  | 0.9  |

#### General Load Results

When pod-level resources were defined, the overall CPU utilization (as monitored by Prometheus) was ~70%.

Without pod-level resources, the overall CPU utilization was ~50%.

The Dev-Space load time, measured as the amount of time between Kubernetes launching all of the containers in the Pod and the Dev Space application becoming ready, was also positively affected. Without pod-level resources, the Dev-Space finished loading within 60 seconds, while when pod-level resources were defined, the Dev-Space finished loading in 10 seconds. The explanation for this huge difference is that some of the containers in the pod need to perform one-time startup operations. With pod-level resources defined, these containers can receive CPU resources that are more adequate for the startup operations, while without the pod-level resources, if more CPU is allocated to these containers, then the CPU is not available to the containers intended to run the main workload in the Dev Space. The fact that pod-level resources are available makes it possible to not have to balance load-time vs. general performance for the Dev-Space.

## Implementation History

- 2020-03-04 - v1 of the proposal 
- 2020-03-06 - Updates due to suggested review
- 2020-03-15 - More updates due to suggested review
- 2020-03-17 - Reworked the proposal based on the suggested reviews
- 2020-03-18 - More updates, after the sig-node meeting on March 17th.
- 2020-06-04 - Updating the KEP after a trial-run on some real-world workloads
- 2020-09-22 - Updating the KEP to allow pod-level resource requests
