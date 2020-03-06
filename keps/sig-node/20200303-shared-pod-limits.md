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

The `Burstable` QoS level is used for any Pod that defines resource limits for at least one container, and doesn't define the same value for both the `request` and `limit` sections. In these Pods, a container which doesn't define any limit for a resource will effectively not be constrained in any way for that resource. Defining a limit for all of the containers in the pod has the implication that it is not possible to define a pod where the resources are controlled on the pod level. 

This KEP proposes a way to define constraints on the Pod level, to allow for a smoother resource allocation strategy for `Burstable` pods.


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

In some cases deploying a single container with all the tasks is not optimal and not always possible. Kubernetes is not aware of these tasks, and doesn't monitor them for failure, and is not able to manage the resources (cgroups) for each of them. By separating the different tasks to their own containers, the application is able to leverage Kubernetes to monitor the tasks. The current `Burstable` QoS implementation requires the pod to either not limit each individual container at all, or micro-manage the resources allocated to each and every container.

This proposal suggests a middle ground, and suggests a way to make it possible to describe to Kubernetes how to limit the resource consumption of multiple containers in the pod at once, on the Pod level, instead of trying to micro-manage the resource limits on the container level. For workloads where the work performed is burstable, this proposal would make it easier to allow the low-level mechanisms available in the underlying operating-system to manage the resources required for the task.

### Goals

This proposal aims to:

*  Allow a `Burstable` pod to define that memory and CPU resource limits should be set on the level of the Pod.
*  Prevent the developer from having to micro-manage the memory and CPU resource assignments for different containers in the same pod.
*  Keep the current `Burstable` behavior as the default. 

### Non-Goals

*  Providing a general-purpose interface to the full range of possible resource management provided by the Linux cgroup hierarchy.

## Proposal

The Pod QoS enhancement already implemented in Kubernetes manages resources as a hieararchy of cgroups in the following way:

<pre>
  QoS CGroup (one of guaranteed, burstable, or besteffort)
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

Each container level cgroup is limited based on the information provided in the `Container` specification. Currently, if the pod belongs to 
the `Burstable` QoS level, and **all** of the containers specify limits for each resource, then these limits are summed, and that limit is
assigned to the pod level cgroup.

The implications of the current `Burstable` definition is that if any containers belonging to the Pod don't define a limit, then those containers
are effectively not limited by the Linux kernel. Conversely (as happens in the `Guaranteed` QoS level), if all of the containers provided a limit, 
then even though the pod level cgroup is configured with the sum of those limits, there is no significance to this since no container can ever use 
more of the resource than what is defined in the container level cgroup.

The proposal in this KEP is to allow users to opt-in to a slightly modified definition of the `Burstable` QoS level. In this modified definition,
the sum of all defined container limits for each resource are always assigned to the pod-level cgroup. This is done by adding an attribute called 
`shareBurstableLimits` on the Pod specification level. If this attribute is set to false (the default) the feature is disabled, and if it is set to 
true then the feature takes effect.

For example, consider the following Pod:

```yaml
apiVersion: v1
kind: Pod
metadata:
  labels:
    run: nginx
  name: nginx
spec:
  containers:
  - image: shell
    name: debian:buster
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

This pod belongs to the `Burstable` QoS level because it defines multiple containers, of which only one has resource limits, and there is a 
difference between the `Request` and the `Limit` for both these resources. Effectively, the containers which don't define any limits are
not constrained in any way. They are provided the same level of resources as a container in a `BestEffort` pod would, except that they belong to a pod with a higher priority, so they are more proofed from causing evictions.

The cgroup hierarchy for each of these resources (memory and cpu) would be this:

<pre>
  QoS CGroup (one of guaranteed, burstable, or besteffort)
   |
   \ pod (memory: unlimited, cpu: unlimited quota)
      |
      +-- container0 (pause container, memory: unlimited, cpu: unlimited quota)
      |
      +-- container1 (shell container, memory: unlimited, cpu: unlimited quota)
      |
      +-- container2 (proxy container, memory: unlimited, CPU: unlimited quota)
      |
      +-- container3 (nginx container, memory: 256M limit, CPU: 1 core)
</pre>

By setting the `ShareBurstableLimits` attribute on the Pod spec to `true`, the following cgroup hierarchy would be configured:

<pre>
  QoS CGroup (one of guaranteed, burstable, or besteffort)
   |
   \ pod (memory: 256M limit, CPU: 1 core)
      |
      +-- container0 (pause container, memory: unlimited, cpu: unlimited quota)
      |
      +-- container1 (shell container, memory: unlimited, cpu: unlimited quota)
      |
      +-- container2 (proxy container, memory: unlimited, CPU: unlimited quota)
      |
      +-- container3 (nginx container, memory: 256M limit, CPU: 1 core)
</pre>

This has the following effect:
1. No change will be noticed for the nginx container
1. The shell and proxy containers will be limited to the amount of resources specified on the Pod cgroup level - no more than 256M memory and 1 CPU core.
1. The pause container will not be affected since it doesn't use any resources anyways.
1. The total resource usage for this Pod will be more predictable as far as the Kubelet is concerned, since the shell container can't consume an unlimited amount of resources.

Effectively, if the shell isn't being used, all of the currently unused resources which are allowed to be consumed would be usable also in the proxy container.

In this example the pod memory limit is set by manipulating the limit on a single container. There is still the possibility to constrain usage for any specific container by specifying the limits in that container's section. For example, consider the following Pod specification:

```yaml
apiVersion: v1
kind: Pod
metadata:
  labels:
    run: nginx
  name: nginx
spec:
  shareBurstableLimits: true
  containers:
  - image: shell
    name: debian:stable
  - image: proxy
    name: envoy:latest
    resources:
      requests:
        memory: 128M
        cpu: "0.5"
      limits:
        memory: 256M
        cpu: "1"
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

In this pod, the cgroups would be set up in the following way:

<pre>
  QoS CGroup (one of guaranteed, burstable, or besteffort)
   |
   \ pod (memory: 512 limit, CPU: 2 cores)
      |
      +-- container0 (pause container, memory: unlimited, cpu: unlimited quota)
      |
      +-- container1 (shell container, memory: unlimited, cpu: unlimited quota)
      |
      +-- container2 (proxy container, memory: 256M limit, CPU: 1 core)
      |
      +-- container3 (nginx container, memory: 256M limit, CPU: 1 core)
</pre>

Both the proxy and the nginx container are constrained, and the entire pod is still limited to the sum of the specified limits of all the containers in the pod.

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
  shareBurstableLimits: true
  containers:
  - image: shell
    name: debian:buster
    resources:
      limits:
        memory: "1024M"
        cpu: "4"
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

Using the `ShareBurstableLimits` feature enables the `tool1` and `tool2` containers to be constrained by the total limit for the pod. 

Without this feature, the developer would need to decide a-priori how much resources to allocate to the tools - and this is not easy to do for this workload.

### Implementation Details/Notes/Constraints 

This implementation proposal doesn't try to enable a developer to specify a Resources section in the Pod spec itself (above the containers definition), so that there would be no need to reconcile the Pod level definition with the definition provided as the sum of the container limits. The basic premise is that the current Kubernetes approach where resource limits are best set (where possible) in the lowest level possible.

The proposal is an **opt-in** feature, and will have no effect on existing deployments. Only deployments that explicitly require this functionality should turn it on by specifying the `sharedBurstableLimit` attribute on the Pod specification.

This proposal is compatible with both cgroup v1 and cgroup v2. Both versions of cgroup allow specifying limits on all levels of the cgroup hierarchy.

### Risks and Mitigations

Since this is an **opt-in** feature, there should be no risk to merging the feature. Users which don't use it will not be affected by it.

When users opt to use the feature, the workload must be able to run in a potentially resource-limited environment. 

## Design Details

1. An attribute called `shareBurstableLimit` is added to the Pod specification, defaulting to `false`.
1. In the code defining the [resources configuration](https://github.com/kubernetes/kubernetes/blob/master/pkg/kubelet/cm/helpers_linux.go#L109) for the pod, check if the attribute is set and act accordingly.


See a PoC implementation in PR [#88899](https://github.com/kubernetes/kubernetes/pull/88899).

## Implementation History

- 2020-03-04 - v1 of the proposal 
- 2020-03-06 - Updates due to suggested review


