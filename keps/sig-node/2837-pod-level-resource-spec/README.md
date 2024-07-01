# KEP-2837: Pod Level Resource Specifications


<!-- toc -->
- [Release Signoff Checklist](#release-signoff-checklist)
- [Summary](#summary)
- [Motivation](#motivation)
  - [Goals](#goals)
  - [Non Goals](#non-goals)
- [Proposal](#proposal)
  - [User Stories](#user-stories)
    - [Story 1](#story-1)
    - [Story 2](#story-2)
  - [Notes/Constraints/Caveats](#notesconstraintscaveats)
  - [Risks and Mitigations](#risks-and-mitigations)
- [Design Details](#design-details)
  - [Components/Features changes](#componentsfeatures-changes)
    - [PodSpec changes](#podspec-changes)
    - [PodSpec Validation](#podspec-validation)
    - [Init Containers &amp; Sidecar Containers](#init-containers--sidecar-containers)
    - [Scheduler Changes](#scheduler-changes)
    - [QoS Changes](#qos-changes)
    - [OOM Killer Behavior](#oom-killer-behavior)
    - [Admission Controller](#admission-controller)
    - [Eviction Manager](#eviction-manager)
    - [PodOverhead](#podoverhead)
    - [[Scoped for Beta] HugeTLB cgroup](#scoped-for-beta-hugetlb-cgroup)
    - [[Scoped for Beta] Topology Manager](#scoped-for-beta-topology-manager)
    - [[Scoped for Beta] Memory Manager](#scoped-for-beta-memory-manager)
    - [[Scoped for Beta] CPU Manager](#scoped-for-beta-cpu-manager)
    - [[Scoped for Beta] In Place Pod Resize](#scoped-for-beta-in-place-pod-resize)
    - [[Scoped for Beta] VPA](#scoped-for-beta-vpa)
    - [[Scoped for Beta] Cluster Autoscaler](#scoped-for-beta-cluster-autoscaler)
  - [[Scoped for Beta] Support for Windows](#scoped-for-beta-support-for-windows)
  - [Test Plan](#test-plan)
    - [Unit tests](#unit-tests)
    - [e2e tests](#e2e-tests)
  - [Graduation Criteria](#graduation-criteria)
    - [Phase 1: Alpha (target 1.31)](#phase-1-alpha-target-131)
    - [Phase 2:  Beta (target 1.32)](#phase-2--beta-target-132)
    - [GA (stable)](#ga-stable)
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
  - [VPA](#vpa)
<!-- /toc -->


## Release Signoff Checklist


Items marked with (R) are required *prior to targeting to a milestone / release*.

- [] (R) Enhancement issue in release milestone, which links to KEP dir in [kubernetes/enhancements] (not the initial KEP PR)
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

<!--
**options** This checklist is iterative and should be reviewed and updated every time this enhancement is being considered for a milestone.
-->

[kubernetes.io]: https://kubernetes.io/
[kubernetes/enhancements]: https://git.k8s.io/enhancements
[kubernetes/kubernetes]: https://git.k8s.io/kubernetes
[kubernetes/website]: https://git.k8s.io/website


## Summary


Currently resource allocation on PodSpec is container-centric, allowing users to specify resource requests and limits for each container. The scheduler uses 
the aggregate of the requested resources by all containers in a pod to find a suitable node for scheduling the pod. The kubelet then enforces these resource
constraints by translating the requests and limits into corresponding cgroup settings at both the pod and container levels. The existing Pod API doesn’t enable 
defining resource constraints at pod level, limiting the flexibility and ease of resource management for pods as a whole. 

This KEP proposes extending the Pod API to support Pod-level resource limits and requests for non-extended resources (CPU and Memory), in addition to the 
existing container-level settings.


## Motivation


Kubernetes workloads often consist of multiple containers collaborating within a pod. While this separation enables granular monitoring and management of each
container, it can be challenging to accurately estimate and allocate resources for individual containers, especially for workloads with unpredictable or 
bursty resource demands. This often leads to over-allocation of resources to ensure that no container experiences resource starvation, as kubernetes currently
lacks a mechanism for sharing resources between containers easily. 

Introducing pod-level resource requests and limits offers a more simplified resource management for multi-container pods as it is easier to gauge the collective
resource usage of all containers in a pod rather than predicting each container’s individual needs. This proposal aims to support pod-level resource requests 
and limits that would allow Kubernetes to manage the total resource consumption of the pod, freeing users from the burden of meticulously allocating resources
for each container. This simplified approach is particularly valuable for tightly coupled applications, and it will enable Kubernetes to leverage the underlying
cgroup resource management mechanisms more effectively for better resource utilization for bursty workloads. 

This new feature of resource specification at pod-level would complement, not replace, the existing container-level limits, providing users with greater flexibility.


### Goals

This proposal aims to:

1. Extend Pod API to allow specifying resource limits on pod level as an opt-in feature. 
2. Make this feature compatible with existing usage of container level resource/limits, and other features like topology manager, memory manager, etc.

### Non Goals


1. This KEP focuses on the core resource types of CPU and memory. It doesn’t intend to address other resource types (e.g. GPU, storage, network bandwidth)
at pod level in this phase. However, it could be considered in future extensions of the KEP.


## Proposal


### User Stories


#### Story 1

A development environment implemented as a Kubernetes Pod allows for separation of tools and a (web-based) IDE between multiple side-cars.
The development environment defines a container with the web-server serving the IDE itself, and constrains it to use a certain amount of memory.
Additional tools are provided in additional side-car deployments - for example an LSP( https://langserver.org/) service, a terminal, and more. 
Using this new feature, the containers providing the terminal, the LSP services, and the set of tools being utilized can share the resource limit
defined for the pod. Consider the following pod definition:

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
      memory: "1024Mi"
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


#### Story 2

Consider a scenario where a pod has some containers with occasional resource spikes while other containers often have idle resources. With current container 
level resource settings, the peak usage of each container needs to be accounted for when setting limits. This leads to wasted resources when the spikes are 
not occurring. Pod-level resource limits will allow setting a limit for the entire pod, enabling containers to share resources more efficiently without the
need to over-provision for each container’s peak usage. 


### Notes/Constraints/Caveats


1. Community is trying to move cgroupv1 to maintenance mode in future Kubernetes versions (likely 1.31). Hence this feature will be supported only on cgroupv2.
2. This feature is an **opt-in** feature, and will have no effect on existing deployments. Only deployments that explicitly require this functionality should
turn it on by specifying the relevant resource section at pod-level in the Pod specification.


### Risks and Mitigations

Since this is an **opt-in** feature, there should be no risk in merging the changes. Existing workloads that do not use this feature won’t be affected.
Besides, the feature will be introduced in a phased approach, starting with an alpha release for testing and gathering feedback, followed by beta and GA releases
as the feature matures and potential issues are addressed.


## Design Details


### Components/Features changes


#### PodSpec changes

New field in `PodSpec`

```
type PodSpec struct {
...
  // Compute resource requirements.
  // +optional
  Resources ResourceRequirements
}
```


#### PodSpec Validation

<table>
  <tr>
   <td colspan="2" >Pod Level
   </td>
   <td colspan="2" >Container Level
   </td>
   <td colspan="2" >Effective Pod Level
   </td>
   <td colspan="2" >Effective Container Level
   </td>
  </tr>
  <tr>
   <td><strong>Requests</strong>
   </td>
   <td><strong>Limits</strong>
   </td>
   <td><strong>Requests</strong>
   </td>
   <td><strong>Limits</strong>
   </td>
   <td><strong>Requests</strong>
   </td>
   <td><strong>Limits</strong>
   </td>
   <td><strong>Requests</strong>
   </td>
   <td><strong>Limits</strong>
   </td>
  </tr>
  <tr>
   <td>Specified
   </td>
   <td>Specified
   </td>
   <td>Specified
   </td>
   <td>Specified
   </td>
   <td>Specified Pod Request
   </td>
   <td>Specified Pod Limit
   </td>
   <td>Specified Container Request
   </td>
   <td>Specified
<p>
Container Limit
   </td>
  </tr>
  <tr>
   <td>Specified
   </td>
   <td>Specified
   </td>
   <td>Specified
   </td>
   <td>Unspecified
   </td>
   <td>Specified Pod Request
   </td>
   <td>Specified Pod Limit
   </td>
   <td>Specified
<p>
Container Request
   </td>
   <td>Effective Pod Limit
   </td>
  </tr>
  <tr>
   <td>Specified
   </td>
   <td>Specified
   </td>
   <td>Unspecified
   </td>
   <td>Specified
   </td>
   <td>Specified
<p>
Pod Request
   </td>
   <td>Specified Pod Limit
   </td>
   <td>Effective Container Limit
   </td>
   <td>Specified Container Limit
   </td>
  </tr>
  <tr>
   <td>Specified
   </td>
   <td>Specified
   </td>
   <td>Unspecified
   </td>
   <td>Unspecified
   </td>
   <td>Specified
<p>
Pod Request
   </td>
   <td>Specified Pod Limit
   </td>
   <td>Implementation defined minimum
   </td>
   <td>Effective Pod Limit
   </td>
  </tr>
  <tr>
   <td>Specified
   </td>
   <td>Unspecified
   </td>
   <td>Specified
   </td>
   <td>Specified
   </td>
   <td>Specified
<p>
Pod Request
   </td>
   <td>Infinity
   </td>
   <td>Specified Container Request
   </td>
   <td>Specified Container Limit
   </td>
  </tr>
  <tr>
   <td>Specified
   </td>
   <td>Unspecified
   </td>
   <td>Specified
   </td>
   <td>Unspecified
   </td>
   <td>Specified
<p>
Pod Request
   </td>
   <td>Infinity
   </td>
   <td>Specified
<p>
Container Request
   </td>
   <td>Infinity
   </td>
  </tr>
  <tr>
   <td>Specified
   </td>
   <td>Unspecified
   </td>
   <td>Unspecified
   </td>
   <td>Specified
   </td>
   <td>Specified
<p>
Pod Request
   </td>
   <td>Infinity
   </td>
   <td>Effective Container Limit
   </td>
   <td>Specified
<p>
Container Limit
   </td>
  </tr>
  <tr>
   <td>Specified
   </td>
   <td>Unspecified
   </td>
   <td>Unspecified
   </td>
   <td>Unspecified
   </td>
   <td>Specified
<p>
Pod Request
   </td>
   <td>Infinity
   </td>
   <td>Implementation defined minimum
   </td>
   <td>Infinity
   </td>
  </tr>
  <tr>
   <td>Unspecified
   </td>
   <td>Specified
   </td>
   <td>Specified
   </td>
   <td>Specified
   </td>
   <td>Effective Pod Limit
   </td>
   <td>Specified
<p>
Pod Limit
   </td>
   <td>Specified
<p>
Container Request
   </td>
   <td>Specified
<p>
Container Limit
   </td>
  </tr>
  <tr>
   <td>Unspecified
   </td>
   <td>Specified
   </td>
   <td>Specified
   </td>
   <td>Unspecified
   </td>
   <td>Effective Pod Limit
   </td>
   <td>Specified
<p>
Pod Limit
   </td>
   <td>Specified
<p>
Container Request
   </td>
   <td>Effective
<p>
Pod Limit
   </td>
  </tr>
  <tr>
   <td>Unspecified
   </td>
   <td>Specified
   </td>
   <td>Unspecified
   </td>
   <td>Specified
   </td>
   <td>Effective Pod Limit
   </td>
   <td>Specified
<p>
Pod Limit
   </td>
   <td>Effective container limit
   </td>
   <td>Specified Container Limit
   </td>
  </tr>
  <tr>
   <td>Unspecified
   </td>
   <td>Specified
   </td>
   <td>Unspecified
   </td>
   <td>Unspecified
   </td>
   <td>Effective Pod Limit
   </td>
   <td>Specified
<p>
Pod Limit
   </td>
   <td>Implementation based min
   </td>
   <td>Effective Pod Limit
   </td>
  </tr>
  <tr>
   <td>Unspecified
   </td>
   <td>Unspecified
   </td>
   <td>Specified
   </td>
   <td>Specified
   </td>
   <td>Aggregated container requests
   </td>
   <td>Aggregated container limits
   </td>
   <td>Specified
<p>
Container Request
   </td>
   <td>Specified
<p>
Container Limit
   </td>
  </tr>
  <tr>
   <td>Unspecified
   </td>
   <td>Unspecified
   </td>
   <td>Specified
   </td>
   <td>Unspecified
   </td>
   <td>Aggregated container requests
   </td>
   <td>Infinity
   </td>
   <td>Specified
<p>
Container Request
   </td>
   <td>Infinity
   </td>
  </tr>
  <tr>
   <td>Unspecified
   </td>
   <td>Unspecified
   </td>
   <td>Unspecified
   </td>
   <td>Specified
   </td>
   <td>Aggregate of effective container requests
   </td>
   <td>Aggregate of effective container limits
   </td>
   <td>Effective Container Limits
   </td>
   <td>Specified Container Limits
   </td>
  </tr>
  <tr>
   <td>Unspecified
   </td>
   <td>Unspecified
   </td>
   <td>Unspecified
   </td>
   <td>Unspecified
   </td>
   <td>Aggregate of effective containers requests 
   </td>
   <td>Infinity
   </td>
   <td>Implementation defined minimum
   </td>
   <td>Infinity
   </td>
  </tr>
</table>
<br>

  Following validation logic needs to be added:

  1. Only CPU and memory resources are supported in pod level resource specifications, hence invalidate pod-level resource specification for other resources like gpu, storage, etc. 
  2. requests
      * `pod.spec.resources.requests[memory/cpu] >= sum(pod.spec.containers[*].resources.requests[memory/cpu]) + sum(pod.spec.initContainers[*].resources.requests[memory/cpu])` is valid.

        Pod-level resource requests are allowed to be greater than the sum of container level resource requests to account for ephemeral containers that could be added later.Pod-level requests are meant to represent the minimum amount of resources needed for all the containers within a pod to function correctly. If a pod’s request is lower than the sum of its container requests, it creates a logical contradiction: the pod as a whole requires less resources than the sum of its parts. Besides, the scheduler will use pod requests to determine suitable nodes for placement. If a pod’s request underestimates its actual resource needs (represented by the sum of container requests), it could be scheduled on a node that cannot provide enough resources, leading to failures and performance degradation.

  3. limits
      * `pod.spec.resources.limits[memory/cpu] >= sum(pod.spec.containers[*].resources.limits[memory/cpu]) + sum(pod.spec.initContainers[*].resources.limits[memory/cpu])` is valid.

          Pod-level resource limits are allowed to be greater than the sum of container level resource limits to account for ephemeral containers that could be added 
          later. Setting a resource limit on the pod in its cgroup settings will not reserve any resource and hence not affect the other workloads running on the node.

      * `pod.spec.resources.limits[memory/cpu] < sum(pod.spec.containers[*].resources.limits[memory/cpu]) + sum(pod.spec.initContainers[*].resources.limits[memory/cpu])` is valid.
        
        For example: consider a pod with multiple containers:
          1. Container 1: Moderate CPU and memory usage, but occasional spikes.
          2. Container 2: Low and consistent resource usage.


          ```yaml
          apiVersion: v1
          kind: Pod
          ...
          spec:
            resources:
              requests:
                memory: 100Mi
                cpu: 1
              limits:
                memory: 400Mi
                cpu: 2.5
            containers:
            - name: container-1
                resources:
                  requests:
                    memory: 60Mi
                    cpu: 0.5
                  limits:
                    memory: 300Mi
                    cpu: 2  
            - name: container-2
                resources:
                  requests:
                    memory: 40Mi
                    cpu: 0.5
                  limits:
                    memory: 200Mi
                    cpu: 1 
          ```

          The pod-level limit will serve as an aggregate ceiling. This will cover the scenario where the sum of peak usage for individual containers in a pod is high, but these
          peaks don’t necessarily occur simultaneously.

          If we simply set the pod-level limit as the sum of the individual container limits, we might overallocate resources. The pod might never actually use all those resources
          at once, leading to wasted capacity on the node.

      * `pod.spec.resources.limits[memory/cpu] < pod.spec.containers[i].resources.limits[memory/cpu]` is invalid.

        A single container’s limit should not be allowed to exceed the pod level limit.
  4. Validation logic comparing requests and limits

      1. Pod level requests greater than pod level limits is invalid.

      2. Pod level requests greater than sum of limits of all containers is invalid.

      3. Sum of container level requests greater than Pod level limits is invalid.
        
          This is already ensured by other validation rules in 1. and 2. as following:
        
          As per 1. Container level Requests <= Pod level requests is valid<br>
          And as per 3.1 Pod-level requests <= Pod level limits is valid<br>
          Hence, by transitivity these rules will ensure Container level requests <= Pod-level limits.

      4. [Already exists] Sum of container level requests less than container level limits is invalid.

#### Init Containers & Sidecar Containers

With the current implementation (before this KEP), Pod’s effective resource request/limit is the sum of 
pod overhead and the higher of:

* The sum of all non-init containers(app and sidecar containers) request/limit for a resource.
* The highest value resource request or limit among all the init containers.

```
Effective Request/Limit = Max(Max(Init container limit), Sum(Sidecar containers) + Sum (App Containers)) + Pod Overhead
```

With the support of pod-level requests and limits, following are a few options that can be used to determine pod’s effective resource request/limit:

* **[Preferred] Option 1: Static values of effective pod request/limit as supplied in the PodSpec.**

  In this the pod level cgroup will be set using the requests and limits from the resources specified at pod-level.

  Example pod spec:

  ```yaml
  apiVersion: v1
  kind: Pod
  spec:
    resources:
      requests:
        memory: 100Mi
        cpu: 1
      limits:
        memory: 200Mi
        cpu: 2
    containers:
      - name: container-1
      - name: container-2
        resources:
          requests:
            memory: 50Mi
            cpu: 1
  ```

  **Resources requests at pod-level**

  * The scheduler will use the requested resources (CPU and memory) settings specified in the PodSpec to find a suitable node to place the pod. 
  * Kubelet will use the CPU requests from the pod-level resources section, and translate it to cpu.weight in pod-level cgroup.

  **Resources limits at pod-level**

  * All init(Regular and Sidecar) and non-init containers will be capped by the pod level limit which represents the peak usage of all containers.
  * Kubelet will use CPU and memory limits from PodSpec, and translate them to cpu.max and memory.max cgroup settings at pod-level respectively. 

  **Pros**

  * Simple to understand and implement.
  * Guarantees enough resources for all containers.

  **Cons**

  * Can lead to over-provisioning (resource wastage), especially if the init container peak usage is significantly higher than the steady-state usage 
  of the other containers (sidecar and app containers). However this can be mitigated by setting container level resource spec for this case.
  
  Example 1: If your application containers require high resources simultaneously, set pod-level requests/limits such that it is an aggregate of all container-level requests/limits.

  Example 2: If your application has init containers (and not sidecar containers) with high resource requirements and usage, while your application containers require fairly less resources, 
  set pod-level requests/limits as maximum of init and non-init container resources requirements. 

  Example 3: For a Pod with init containers, sidecar containers and app containers, set the pod level requests/limits such that:
  
  ```
  Pod level Request/Limit = Max(Max(Init container limit), Sum(Sidecar containers) + Sum (App Containers))
  ```

* **Option 2: Phased requests/limits - Pod-level cgroups are set after regular init containers are completed.**

  This will require specifying requests/limits for regular init containers at container level if the peak resource requirement/usage of init containers is higher than 
  app containers requirement/usage. Init containers will rely on their own limits. The resources spec set at pod level needs to account for all app and sidecar containers.
  The pod-level cgroup will be set *AFTER* the execution of init containers are completed

  Example pod spec:

  ```yaml
  apiVersion: v1
  kind: Pod
  spec:
    resources:
      requests:
        memory: 100Mi
        cpu: 2
      limits:
        memory: 200Mi
        cpu: 3
    initContainers:
    - name: init-myservice
      resources:
        requests:
          memory: 300Mi
          cpu: 4
        limits:
          memory: 600Mi
          cpu: 6
    containers:
    - name: container-1
      resources:
        requests:
          memory: 50Mi
          cpu: 1
        limits:
          memory: 100Mi
          cpu: 2
      - name: container-2
        resources:
          requests:
            memory: 50Mi
            cpu: 1
          limits:
            memory: 100Mi
            cpu: 2 
  ```

  **Resources requests at pod-level**

  * The scheduler will use the higher of init and pod-level requests values to find the suitable node for pod placement.
    
    From the example above, the scheduler will use:

      max(init container resources requests, pod-level resources requests) 

        = max (memory: 300Mi cpu: 4, memory: 100 Mi cpu: 2)

        = (memory: 300 Mi, cpu: 4) to find a suitable node.

  * The init containers will rely on its own requests specified in the PodSpec. 
  * After the init containers execution is completed, kubelet will use the pod-level CPU request to set cpu.weight at pod-level cgroup.

  **Resources limits at pod-level**

  * The init containers will rely on their own limits specified in the PodSpec.
  * After the init containers execution is completed, kubelet will use CPU and memory limits at pod-level, and translate them to cpu.max and memory.max cgroup settings
  at pod level respectively. 

  **Pros**

  * Reduces the problem of overprovisioning.

  **Cons**

  * Complicates the definitions of QoS classes as this will require considering requests/limits for both initContainer and pod-level requests/limits to attach a 
  QoS class with a pod. It might be worth setting dynamic QoS classes if this option were to be implemented. The QoS class in the initialization phase is based on
  init containers requests/limits, while the QoS class after the initialization phase takes pod-level requests/limits into account. 
  * Complicated to implement as it requires additional logic in kubelet to monitor init containers completion after which it will set/change pod level cgroup settings.

**Note: Option 1 is a *preferred* design option as it is easier to implement, and also easier to understand from UX perspective. 
If there are enough supporting use cases for Option 2, we can explore the implementation in the beta phase of the KEP. Option 2
will also require discussions about dynamic QoS classes as per the execution phase of the pod (init vs non-init phase).**

#### Scheduler Changes

The scheduler will determine pod resources requirements in the following ways:
1. Directly from Pod-Level Resources (pod.spec.resources)
   
   If the pod has pod.spec.resources defined, the scheduler will use these values (requests and limits) as the sole indicator of the pod's resource needs, including init containers, sidecar containers and regular containers.
2. Indirectly from Container-level Resources
   
   If the pod does NOT have pod.spec.resources defined, the scheduler will derive the pod's resource needs by aggregating the requests and limits of all containers (init, sidecar and regular containers) within the pod.
3. Both Pod-level and Container level
   
   If both pod-level and container-level resources are specified, the scheduler will prioritize the pod-level resources and ignore the container-level requests for node placement decisions.

#### QoS Changes

QoS is determined by derived Pod Resources which considers (preferred in order):
1. The values for resources specified in the pod’s `resources` stanza.
2. The value of resources specified in the container’s `resources` stanza.

In simple words, if `resources` is specified at pod-level, the values in pod-level `resources` stanza are given a preference in determining the QoS class, otherwise container level `resources` are considerted for QoS determination.

* **Guaranteed QoS**

    A pod is considered `Guaranteed`, if for each resource type, either of following is true:
    * It is explicitly resourced (pod.spec.resources is set) for that resource type AND the pod's request is equal to its limits.
    * It is implicitly resourced (pod.spec.resources is not set) AND all containers requests are equal to their limits.

  **Note: If the pod spec includes `resources` stanza at pod level, but either requests or limits are missing for a resource, the pod will not be considered Guaranteed QoS, even if container-level requests and limits are present and are equal.**

* **Burstable QoS Criteria**

    A pod is considered `Burstable`, if for each resource type it doesn't meet the criteria for Guaranteed QoS for that resource, and either of following is true:

    * It is explicitly resourced (pod.spec.resources is set) for that resource, AND request (less than or equal to limit) or limit is set.
    * It is implicitly resourced for that resource AND for at least one container in the pod, request (less than or equal to limit) or limit is set. 
  

* **BestEffort QoS Criteria**

    None of pod-level or container-level resources requests or limits are specified.


#### OOM Killer Behavior
With Pod level resources, the OOM Killer will evolve to consider the cumulative resource limits set for the entire Pod. If the aggregated
memory usage of all the containers within a Pod exceeds the effective pod level limits, the entire Pod becomes a candidate for OOM Killing, even if individual containers are within their limits.

Within a pod, the OOM Killer will maintain its existing behavior, considering container level limits (if set) and QoS classes, so a container
exceeding its individual limits can still be terminated even if a pod as a whole is under its limit.

#### Admission Controller

* LimitRanger

  LimitRanger today supports objects of type `Container`, `Pod` and `PersistentVolumeClaim`. It allows following constraints:
  
    * min (this option is to specify min usage constraint on the resource)
    * max (this option is to specify max usage constraints on the resource)
    * defaultRequest (this option is to specify default resource request for the named resource)
    * default (this option is to specify default limit for the named resource)
    * maxLimitRequestRatio (represents the max burst value of the named resource)
  
  default and defaultRequest are not supported by LimitRanger of type `Pod`. To incorporate Pod level resource specifications, LimitRanger of type `Pod` logic will be extended with following changes:
  1. Validation at Pod Admission will derive pod resources from `resources` stanza at pod level if it is set.
  2. Extend `Pod` type LimitRanger to support `defaultRequest` and `default` to allow setting Pod level defaults.

* ResourceQuota

  Quota will derive the effective resource requests and limits for the pod, including the new pod-level resources.

#### Eviction Manager

For memory.available signal, the eviction manager checks the pod's memory usage and compares it with memory requests at pod level to determine the pod eviction order. It aggregates the 
container-level memory requests to calculate the effective pod-level memory request.When pod-level memory requests are specified, the Eviction Manager should use the pod-level request 
directly instead of aggregating container-level requests. This logic should be modified to check if pod-level requests are set. If so, use the pod-level request; otherwise, fall back 
to aggregating container requests.


#### PodOverhead

Since the PodOverhead feature works at pod-level cgroup, pod-level resource specification and PodOverhead will work together seamlessly. When the kube-scheduler is deciding 
which node should run a new Pod, the scheduler considers that Pod's overhead as well as the sum of container requests for that Pod. Scheduler should add pod-level request,
podoverhead to find a suitable node, when pod level request is specfied.

Likewise the kubelet will need to add PodOverhead to the limits specified in the pod-level spec (instead of sum of container level requests) when setting the upper limit for
the pod-level cgroup (i.e. cpu.max and memory.max).


#### [Scoped for Beta] HugeTLB cgroup

Note: This section includes only high level overview; Design details will be added in Beta stage.

To support pod-level resource specifications for hugepages, Kubernetes will need to adjust how it handles hugetlb cgroups. Unlike memory, where an unset limit 
means unlimited, an unset hugetlb limit is the same as setting it to 0.

With the proposed changes, hugepages-2Mi and hugepages-1Gi will be added to the pod-level resources section, alongside CPU and memory. The hugetlb cgroup for the
pod will then directly reflect the pod-level hugepage limits, rather than using an aggregated value from container limits. When scheduling, the scheduler will 
consider hugepage requests at the pod level to find nodes with enough available resources.


#### [Scoped for Beta] Topology Manager

Note: This section includes only high level overview; Design details will be added in Beta stage.


* (Tentative) Only pod level scope for topology alignment will be supported if pod level requests and limits are specified without container-level requests and limits.
* The pod level scope for topology aligntment will consider pod level requests and limits instead of container level aggregates.
* The hint providers will consider pod level requests and limits instead of container level aggregates.


#### [Scoped for Beta] Memory Manager

Note: This section includes only high level overview; Design details will be added in Beta stage.

With the introduction of pod-level resource specifications, the Kubernetes Memory Manager will evolve to track and enforce resource limits at both the pod and container levels. It will need to aggregate memory usage across all containers within a pod to calculate the pod's total memory consumption. The Memory Manager will then enforce the pod-level limit as the hard cap for the entire pod's memory usage, preventing it from exceeding the allocated amount.  While still maintaining container-level limit enforcement, the Memory Manager will need to coordinate with the Kubelet and eviction manager to make decisions about pod eviction or individual container termination when the pod-level limit is breached.


#### [Scoped for Beta] CPU Manager

Note: This section includes only high level overview; Design details will be added in Beta stage.

With the introduction of pod-level resource specifications, the CPU manager in Kubernetes will adapt to manage CPU requests and limits at the pod level rather than solely at the container level. This change means that the CPU manager will allocate and enforce CPU resources based on the total requirements of the entire pod, allowing for more flexible and efficient CPU utilization across all containers within a pod. The CPU manager will need to ensure that the aggregate CPU usage of all containers in a pod does not exceed the pod-level limits.

#### [Scoped for Beta] In Place Pod Resize

TBD. Do not review for the alpha stage.

#### [Scoped for Beta] VPA

TBD. Do not review for the alpha stage.

#### [Scoped for Beta] Cluster Autoscaler

TBD. Do not review for the alpha stage.

### [Scoped for Beta] Support for Windows

Pod-level resource specifications are a natural extension of Kubernetes' existing resource management model. Although this new feature is expected to function with Windows containers, careful testing and consideration are required due to platform-specific differences. As the introduction of pod-level resources is a major change in itself, full support for Windows will be addressed in future stages, beyond the initial alpha release.

### Test Plan

[X] I/we understand the owners of the involved components may require updates to existing tests to make this code solid enough prior to committing the changes
necessary to implement this enhancement.

#### Unit tests

This feature will touch multiple components. For alpha, unit tests coverage for following packages needs to be added:

* Scheduler logic will be updated to consider pod-level requests. Hence pkg/scheduler will require additional coverage.
* pkg/kubelet/cm will be updated to set pod-level cgroups using CPU requests and limits, and memory limits.
* pkg/apis/core/validation/types_test.go and pkg/apis/core/validation since new fields are added in PodSpec and also new validation rules are required for the new fields.
* pkg/kubeapiserver/admission for changes made in the admission logic for LimitRanger and ResourceQuota admission controllers.


#### e2e tests

Following scenarios need to be covered:



* Cgroup settings when pod-level resources are set.
* Validate scheduling and admission.
* Validate the containers with no limits set are throttled on CPU when CPU usage reaches Pod level CPU limits.
* Validate the containers with no limits set are OOMKilled when memory usage reaches Pod level memory limits.


### Graduation Criteria


#### Phase 1: Alpha (target 1.31)


* Feature is disabled by default. It is an opt-in feature which can be enabled by enabling the `PodLevelResources`
feature gate and by setting the new `resources` fields in PodSpec at Pod level.
* Support the basic functionality for scheduler to consider pod-level resource requests to find a suitable node.
* Support the basic functionality for kubelet to translate pod-level requests/limits to pod-level cgroup settings.
* Unit test coverage.
* E2E tests.
* Documentation mentioning high level design.


#### Phase 2:  Beta (target 1.32)


* User Feedback.
* Feature is disabled by default. It is an opt-in feature which can be enabled by setting the new fields in PodSpec.
* Support for all features that are scoped out of alpha phase in the KEP design.
* Unit test coverage for additional features supported in Beta.
* E2E tests for additional features supported in Beta.
* Documentation update and blog post to announce feature in beta phase.
* [Tentative] Benchmark tests for resource usage with and without pod level resources for bursty workloads.
  * Use kube_pod_container_resource_usage metric to check resource utilization.


#### GA (stable)


* TBD


### Upgrade / Downgrade Strategy

The existing workloads will not have any impact specifically because of pod-level resources feature
since they won't be using the new field `resources` in PodSpec at pod-level. However, if pods 
that do utilize this feature have poorly configured limits, they could become noisy neighbours and
negatively impact other workloads on the same node. It's important to nose that this issue of noisy
neighbours due to incorrect limits it not exclusive to pod-level resources; it can also occur
at with container-level limits.

If a cluster is upgraded, the cluster administrator will experience pre-upgrade behavior until 
`resources` field is specified at pod-level. 

If the workloads are not using the new feature, downgrade won't be affected. However, if it is required to
downgrade a cluster that has workloads using the pod-level limits functionality, following steps need to be
performed:

* Translate pod-level resource specifications to container-level specifications.

* Delete existing workloads and recreate them with container-level specifications.

* Drain the nodes one by one to prevent the new pods from being scheduled on them.

* Downgrade the kubelet on the drained nodes to a version that doesn't support pod-level resource specification
fields.

* Downgrade the control plane components.

* Uncordon the nodes, making it available for scheduling again.


### Version Skew Strategy

Pod-level resource specification is an opt-in feature. For this feature to work correctly, it must be enabled
in all parts of the cluster (scheduler, API server, kubelet). However, if a node is running an older version
of kubelet that doesn't support this feature, it will ignore the pod-level resouce specifications and continue
to use the aggregated container-level values (backward compatible). Applications must be prepared for this version
skew implementation.

For users wanting to use this feature, it is user's responsibility to use the correct version of kube-scheduler,
kube-apiserver and kubelet otherwise the feature will not work as expected.

## Production Readiness Review Questionnaire


### Feature Enablement and Rollback

<!--
This section must be completed when targeting alpha to a release.
-->

###### How can this feature be enabled / disabled in a live cluster?

<!--
Pick one of these and delete the rest.

Documentation is available on [feature gate lifecycle] and expectations, as
well as the [existing list] of feature gates.

[feature gate lifecycle]: https://git.k8s.io/community/contributors/devel/sig-architecture/feature-gates.md
[existing list]: https://kubernetes.io/docs/reference/command-line-tools-reference/feature-gates/
-->

- [X] Feature gate (also fill in values in `kep.yaml`)
  - Feature gate name: PodLevelResources
  - Components depending on the feature gate: kubelet, kube-apiserver, kube-scheduler
  - Will enabling / disabling the feature require downtime of the control
    plane? No. `resources` must be set/unset in `spec` at pod-level to enable/disable this feature.
    The workloads need to restart to disable the feature.
  - Will enabling / disabling the feature require downtime or reprovisioning
    of a node? No. Only the workloads need to restart.

###### Does enabling the feature change any default behavior?

No. This feature is guarded by a feature gate, and requires setting Pod level `resource` stanza explicitly.

###### Can the feature be disabled once it has been enabled (i.e. can we roll back the enablement)?

Yes. Once the feature gates are disabled, pods created with pod level resource spec need to be deleted and recreated with only container level resources.

###### What happens if we reenable the feature if it was previously rolled back?

Rollbacks will require disabling the feature, and recreating the pods that were using this new feature.
Once the feature is reenabled, the existing workloads will run as usual since they don't have `resources` set at pod level.
The new `resources` field will become available again for use in the Pod Spec for existing and new workloads.


###### Are there any tests for feature enablement/disablement?

Yes, the tests will be added along with alpha implementation. 
* Validate that scheduler considers pod level resources when the feature is enabled, and falls back to container level resources 
when the feature is disabled. 
* Validate the kubelet is using pod level resources in cgroup settings if the feature is enabled, and uses container level values
when feature is disabled.
* Confirm that the API server gracefully ignores the pod-level resource fields and reverts to the existing container-level 
resource aggregation logic when the feature gate is disabled.
* Confirm Limit Ranger uses defaultRequest and limit when feature is enabled, and errors out for these values when feature is disabled.

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

Since the feature is opt-in feature and requires setting new fields in the PodSpec,
the rollouts won't likely fail because of this feature. For the new workloads
that want to use this feature, there could be unexpected interactions with
the existing features. This is why this feature will be rolled out in phases
to make sure all the cases and interaction with existing features are covered
before making it available in GA.

Rollbacks should be seamless if done after disabling the feature, and recreating 
the running workloads that use the feature.

###### What specific metrics should inform a rollback?

<!--
What signals should users be paying attention to when the feature is young
that might indicate a serious problem?
-->

Any unusual observations in the following metrics should signal rollback:

* `kube_pod_container_resource_requests`: This metric exposes the resource requests (CPU and memory) for each container within a pod. With pod level
  resource spec, this should map to pod level requests. 
* `kube_pod_container_resource_limits`: This metric exposes the resource limits (CPU and memory) for each container within a pod. With pod level
   resource spec, this should map to pod level limits.
* `node_collector_evictions_total`: to check if a pod level resource setting is causing to evict more pods than normal
* `started_pods_errors_total`: exposed by kubelet to check if large number of pods are failing unusually
* `started_containers_errors_total`: exposed by kubelet to check if large number of containers are failing unusually

###### Were upgrade and rollback tested? Was the upgrade->downgrade->upgrade path tested?

<!--
Describe manual testing that was done and the outcomes.
Longer term, we may want to require automated upgrade/rollback tests, but we
are missing a bunch of machinery and tooling and can't do that now.
-->
It will be tested manually as a part of implementation and there will also
be automated tests to cover the scenarios.

###### Is the rollout accompanied by any deprecations and/or removals of features, APIs, fields of API types, flags, etc.?

No

### Monitoring Requirements

<!--
This section must be completed when targeting beta to a release.

For GA, this section is required: approvers should be able to confirm the
previous answers based on experience in the field.
-->

###### How can an operator determine if the feature is in use by workloads?

<!--
Ideally, this should be a metric. Operations against the Kubernetes API (e.g.,
checking if there are objects with field X set) may be a last resort. Avoid
logs or events for this purpose.
-->

This feature will be built into kubelet, API server and scheduler. In order to determine if the feature
is being used by the workloads, check the  `resources` field at pod level in the spec. There's no special
metric planned to track the usage of this feature at the moment.

###### How can someone using this feature know that it is working for their instance?

<!--
For instance, if this is a pod-related feature, it should be possible to determine if the feature is functioning properly
for each individual pod.
Pick one more of these and delete the rest.
Please describe all items visible to end users below with sufficient detail so that they can verify correct enablement
and operation of this feature.
Recall that end users cannot usually observe component logs or access metrics.
-->

- [X] Other Field
  - pod.status.spec.resources[x]
  - Inspect Cgroup Filesystem: cgroup fs for the pod will reflect the requests/limits at pod level in cpu.weight, cpu.max, memory.max cgroup files.


###### What are the reasonable SLOs (Service Level Objectives) for the enhancement?

<!--
This is your opportunity to define what "normal" quality of service looks like
for a feature.

It's impossible to provide comprehensive guidance, but at the very
high level (needs more precise definitions) those may be things like:
  - per-day percentage of API calls finishing with 5XX errors <= 1%
  - 99% percentile over day of absolute value from (job creation time minus expected
    job creation time) for cron job <= 10%
  - 99.9% of /health requests per day finish with 200 code

These goals will help you determine what you need to measure (SLIs) in the next
question.
-->
Measure the overall resource utilization of the cluster with and without pod-level resources to ensure that the feature
does not lead to significant over-provisioning or underutilization of resources.

###### What are the SLIs (Service Level Indicators) an operator can use to determine the health of the service?

<!--
Pick one more of these and delete the rest.
-->

- [X] Metrics
  - Metric name:
  - `apiserver_rejected_requests` will indicate any failures (`Bad Request` code=400) related to translation of new `resources` field in PodSpec. 
  - `schedule_attempts_total{result="error|unschedulable"}`
  - `node_collector_evictions_total`: to check if a pod level resource setting is causing to evict more pods than normal
  - `started_pods_errors_total`: exposed by kubelet to check if large number of pods are failing unusually
  - `started_containers_errors_total`: exposed by kubelet to check if large number of containers are failing unusually
  - Components exposing the metric: apiserver, kubelet, scheduler

###### Are there any missing metrics that would be useful to have to improve observability of this feature?

<!--
Describe the metrics themselves and the reasons why they weren't added (e.g., cost,
implementation difficulties, etc.).
-->
No 

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
No. It only modifies existing API request/response payloads.

###### Will enabling / using this feature result in introducing new API types?

<!--
Describe them, providing:
  - API type
  - Supported number of objects per cluster
  - Supported number of objects per namespace (for namespace-scoped objects)
-->
Yes, the PodSpec object will be modified to include new fields for pod level resources
specifications i.e. pod.spec.resources.requests and pod.spec.resources.limit.


###### Will enabling / using this feature result in any new calls to the cloud provider?

<!--
Describe them, providing:
  - Which API(s):
  - Estimated increase:
-->
No

###### Will enabling / using this feature result in increasing size or count of the existing API objects?

<!--
Describe them, providing:
  - API type(s):
  - Estimated increase in size: (e.g., new annotation of size 32B)
  - Estimated amount of new objects: (e.g., new Object X for every existing Pod)
-->
Negligible.

###### Will enabling / using this feature result in increasing time taken by any operations covered by existing SLIs/SLOs?

<!--
Look at the [existing SLIs/SLOs].

Think about adding additional work or introducing new steps in between
(e.g. need to do X to start a container), etc. Please describe the details.

[existing SLIs/SLOs]: https://git.k8s.io/community/sig-scalability/slos/slos.md#kubernetes-slisslos
-->

The computational overhead introduced in API server, scheduler and kubelet due to additional validation for pod level resource specifications should be negligible. Thorough testing and monitoring will ensure the SLIs/SLOs are not impacted.

###### Will enabling / using this feature result in non-negligible increase of resource usage (CPU, RAM, disk, IO, ...) in any components?

<!--
Things to keep in mind include: additional in-memory state, additional
non-trivial computations, excessive access to disks (including increased log
volume), significant amount of data sent and/or received over network, etc.
This through this both in small and large cases, again with respect to the
[supported limits].

[supported limits]: https://git.k8s.io/community//sig-scalability/configs-and-limits/thresholds.md
-->

No. Pod-level resource information could lead to marginal increase in storage requirements for the new fields, although
its impact is likely to be small compared to other Kubernetes object data.

###### Can enabling / using this feature result in resource exhaustion of some node resources (PIDs, sockets, inodes, etc.)?

<!--
Focus not just on happy cases, but primarily on more pathological cases
(e.g. probes taking a minute instead of milliseconds, failed pods consuming resources, etc.).
If any of the resources can be exhausted, how this is mitigated with the existing limits
(e.g. pods per node) or new limits added by this KEP?

Are there any tests that were run/should be run to understand performance characteristics better
and validate the declared limits?
-->
Yes, it could.

Pod-level resource limits can enable higher pod density on a node, especially when containers within a
pod don't have simultaneous peak resource usage. This is because pod-level limits can more accurately reflect
the pod's overall resource needs, reducing over-provisioning compared to using only container-level limits.
However, this increased density may also increase consumption of other resources like PIDs and network 
sockets. 

This is however be mitigated by `maxPods` kubelet configuration that limits the number of pods on a node.


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

No dependency on etcd availability.

The scheduler relies on the API server to fetch pod specifications and node information.
Without access to the API server, it cannot make informed scheduling decisions based on
pod-level resource requests. This could lead to suboptimal placement.

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

It is suggested that if workloads with pod level resource specifications enabled cause performance 
or stability degradations, those workloads are recrated with only container level resource specs.

## Implementation History

- ** 2020-03-17:** Initial discussion in [SIG Node meeting] (https://www.youtube.com/watch?v=3cU56ZiUZ8w&list=PL69nYSiGNLP1wJPj5DYWXjiArF-MJ5fNG&index=101)
- ** 2020-03-30:** Initial [KEP draft](https://github.com/kubernetes/enhancements/pull/1592) and discussion.
- ** 2021-07-26:** Issue [link](https://github.com/kubernetes/enhancements/issues/2837)
- ** 2024-05-31:** Revised KEP for alpha (#4678)[https://github.com/kubernetes/enhancements/pull/4678]


## Drawbacks

<!--
Why should this KEP _not_ be implemented?
-->

## Alternatives

### VPA

This KEP focusses on addressing the challenge of accurately assessing the container-level resource usage. VPA could be used as an alternative
as it requires setting initial values for requests and limits, and then dynamically adjusts the values based on the actual usage. However,
setting the initial values at container level is still required.

Furthermore, VPA introduces a delay in resource adjustments, as it needs time to gather and analyze usage data. This delay can be problematic
for applications that experience sudden spike in resource usage demands. Additionally, there's always a risk that VPA's requested resource
adjustments might be denied due to limited cluster resources.


<!--
What other approaches did you consider, and why did you rule them out? These do
not need to be as detailed as the proposal, but should include enough
information to express the idea and why it was not acceptable.
-->
