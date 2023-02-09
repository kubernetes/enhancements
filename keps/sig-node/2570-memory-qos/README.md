# KEP-2570: Support Memory QoS with cgroups v2
<!-- toc -->
- [Release Signoff Checklist](#release-signoff-checklist)
- [Summary](#summary)
- [Motivation](#motivation)
  - [Goals](#goals)
  - [Non-Goals](#non-goals)
- [Proposal](#proposal)
    - [Alpha v1.22](#alpha-v122)
    - [Alpha v1.27](#alpha-v127)
  - [User Stories (Optional)](#user-stories-optional)
    - [Memory Sensitive Workload](#memory-sensitive-workload)
    - [Node Availability](#node-availability)
  - [Comparison with Memory Manager](#comparison-with-memory-manager)
  - [memory.low vs memory.min](#memorylow-vs-memorymin)
  - [Notes/Constraints/Caveats (Optional)](#notesconstraintscaveats-optional)
  - [Risks and Mitigations](#risks-and-mitigations)
- [Design Details](#design-details)
  - [Prerequisite](#prerequisite)
  - [Feature Gate](#feature-gate)
  - [Mapping Rules](#mapping-rules)
    - [Container/Pod](#containerpod)
    - [Node](#node)
  - [Interactive](#interactive)
  - [Workflow](#workflow)
    - [Container](#container)
    - [Pod](#pod)
    - [QoS](#qos)
    - [Node](#node-1)
  - [Cgroup Hierarchy](#cgroup-hierarchy)
  - [Cgroup v2 Support](#cgroup-v2-support)
  - [Container Runtime Interface (CRI) Changes](#container-runtime-interface-cri-changes)
  - [Test Plan](#test-plan)
      - [Prerequisite testing updates](#prerequisite-testing-updates)
      - [Unit tests](#unit-tests)
      - [Integration tests](#integration-tests)
      - [e2e tests](#e2e-tests)
  - [Graduation Criteria](#graduation-criteria)
    - [Alpha Graduation](#alpha-graduation)
    - [Beta Graduation](#beta-graduation)
    - [GA Graduation](#ga-graduation)
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
- [x] (R) Enhancement issue in release milestone, which links to KEP dir in [kubernetes/enhancements] (not the initial KEP PR)
- [x] (R) KEP approvers have approved the KEP status as `implementable`
- [x] (R) Design details are appropriately documented
- [x] (R) Test plan is in place, giving consideration to SIG Architecture and SIG Testing input (including test refactors)
- [x] (R) Graduation criteria is in place
- [x] (R) Production readiness review completed
- [x] (R) Production readiness review approved
- [x] "Implementation History" section is up-to-date for milestone
- [ ] User-facing documentation has been created in [kubernetes/website], for publication to [kubernetes.io]
- [ ] Supporting documentation—e.g., additional design documents, links to mailing list discussions/SIG meetings, relevant PRs/issues, release notes

## Summary
Support memory qos with cgroups v2.

## Motivation
In traditional cgroups v1 implement in Kubernetes, we can only limit cpu resources, such as `cpu_shares / cpu_set / cpu_quota / cpu_period`, memory qos has not been implemented yet. cgroups v2 brings new capabilities for memory controller and it would help Kubernetes enhance memory isolation quality.

### Goals
- Provide guarantees around memory availability for pod and container memory requests and limits
- Provide guarantees around memory availability for node resource
- Make use of new cgroup v2 memory knobs(`memory.min/memory.high`) for pod and container level cgroup
- Make use of new cgroup v2 memory knobs(`memory.min`) for node level cgroup

### Non-Goals
- Additional qos design
- Support other resources qos
- Consider QOSReserved feature

## Proposal
This proposal uses memory controller of cgroups v2 to support memory qos for guaranteeing pod/container memory requests/limits and node resource.

Currently we only use `memory.limit_in_bytes=sum(pod.spec.containers.resources.limits[memory])` with cgroups v1 and `memory.max=sum(pod.spec.containers.resources.limits[memory])` with cgroups v2 to limit memory usage. `resources.requests[memory]` is not yet used neither by cgroups v1 nor cgroups v2 to protect memory requests. About memory protection, we use `oom_scores` to determine order of killing container process when OOM occurs. Besides, kubelet can only reserve memory from node allocatable at node level, there is no other memory protection for node resource.

So there are missing some memory protection, it may cause:
- Pod/Container memory requests can't be fully reserved, page cache is at risk of being recycled
- Pod/Container memory allocation is not well protected, there may occur allocation latency frequently when node memory nearly runs out
- Memory overcommit of container is not throttled, there may increase risk of node memory pressure
- Memory resource of node can't be fully retained and protected

Cgroups v2 introduces a better way to protect and guarantee memory quality.

| File | Description |
| -------- | -------- |
| memory.min | memory.min specifies a minimum amount of memory the cgroup must always retain, i.e., memory that can never be reclaimed by the system. If the cgroup's memory usage reaches this low limit and can’t be increased, the system OOM killer will be invoked. **We map it to `requests.memory`.** |
| memory.max | memory.max is the memory usage hard limit, acting as the final protection mechanism: If a cgroup's memory usage reaches this limit and can't be reduced, the system OOM killer is invoked on the cgroup. Under certain circumstances, usage may go over the memory.high limit temporarily. When the high limit is used and monitored properly, memory.max serves mainly to provide the final safety net. The default is max. **We map it to `limits.memory` as consistent with existing `memory.limit_in_bytes` for cgroups v1.** |
| memory.low | memory.low is the best-effort memory protection, a "soft guarantee" that if the cgroup and all its descendants are below this threshold, the cgroup's memory won't be reclaimed unless memory can’t be reclaimed from any unprotected cgroups. Not yet considered for now. |
| memory.high | memory.high is the memory usage throttle limit. This is the main mechanism to control a cgroup's memory use. If a cgroup's memory use goes over the high boundary specified here, the cgroup’s processes are throttled and put under heavy reclaim pressure. The default is max, meaning there is no limit. **We use a formula to calculate `memory.high` depending on `limits.memory/node allocatable memory` and a memory throttling factor.** |

This proposal sets `requests.memory` to `memory.min` for protecting container memory requests. `limits.memory` is set to `memory.max` (this is consistent with existing `memory.limit_in_bytes` for cgroups v1, we do nothing because [cgroup_v2](https://github.com/kubernetes/enhancements/tree/master/keps/sig-node/2254-cgroup-v2) has implemented for that).  

We also introduce `memory.high` for container cgroup to throttle container memory overcommit allocation. 
***Note***: memory.high is set for container-level cgroup, and not for pod-level cgroup. If a container in a pod sees a spike in memory usage, it could result in total pod-level memory usage to reach memory.high level set at pod-level cgroup. This will induce throttling in other containers as the pod-level memory.high was hit. Hence to avoid containers from affecting each other, we set memory.high for only container-level cgroup.

#### Alpha v1.22
It is based on a formula:
```
memory.high=(limits.memory or node allocatable memory) * memory throttling factor, 
where default value of memory throttling factor is set to 0.8
```

e.g. If a container has `requests.memory=50, limits.memory=100`, and we have a throttling factor of .8, `memory.high` would be 80. If a container has no memory limit specified, we substitute `limits.memory` for `node allocatable memory` and apply the throttling factor of .8 to that value.
It must be ensured that `memory.high` is always greater than `memory.min`.

Node reserved resources(kube-reserved/system-reserved) are either considered. It is tied to `--enforce-node-allocatable` and `memory.min` will be set properly.

Brief map as follows:
| type | memory.min | memory.high |
| -------- | -------- | -------- | 
| container | requests.memory | limits.memory/node allocatable memory * memory throttling factor |
| pod | sum(requests.memory) | N/A |
| node | pods, kube-reserved, system-reserved | N/A |

#### Alpha v1.27
The formula for memory.high for container cgroup is modified in Alpha stage of the feature in K8s v1.27. It will be set based on formula:
```
memory.high=floor[(requests.memory + memory throttling factor * (limits.memory or node allocatable memory - requests.memory))/pageSize] * pageSize, where default value of memory throttling factor is set to 0.9
```
Note: If a container has no memory limit specified, we substitute `limits.memory` for `node allocatable memory` and apply the throttling factor of .9 to that value.


The table below runs over the examples with different values requests.memory and 1Mi pageSize:

| limits.memory (1000) | memory throttling factor (0.9)|
| ---------------------- | ----------------------------- |
| request 0    | 900  |
| request 100  | 910  |
| request 200  | 920  |
| request 300  | 930  |
| request 400  | 940  |
| request 500  | 950  |
| request 600  | 960  |
| request 700  | 970  |
| request 800  | 960  |
| request 900  | 980  |
| request 1000 | 1000 |


Node reserved resources(kube-reserved/system-reserved) are either considered. It is tied to `--enforce-node-allocatable` and `memory.min` will be set properly.

Brief map as follows:
| type | memory.min | memory.high |
| -------- | -------- | -------- | 
| container | requests.memory | floor[(requests.memory + memory throttling factor * (limits.memory or node allocatable memory - requests.memory))/pageSize] * pageSize |
| pod | sum(requests.memory) | N/A |
| node | n/a | pods, kube-reserved, system-reserved | N/A |

###### Reasons for changing the formula of memory.high calculation in Alpha v1.27

The formula for memory.high has changed in K8s v1.27 as the Alpha v1.22 implementation has following problems:
1. It fails to throttle when requested memory is closer to memory limits (or node allocatable) as it results in memory.high being less than requests.memory.

   For example, if `requests.memory = 85, limits.memory=100`, and we have a throttling factor of 0.8, then as per the Alpha implementation memory.high =  memory throttling factor * limits.memory i.e. memory.high = 80. In this case the level at which throttling is supposed to occur i.e. memory.high is less than requests.memory. Hence there won't be any throttling as the Alpha v1.22 implementation doesn't allow memory.high to be less than requested memory. 

2. It could result in early throttling putting the processes under early heavy reclaim pressure. 
  
    For example, 
    * `requests.memory` = 800Mi

      `memory throttling factor` = 0.8

      `limits.memory` = 1000Mi

      As per Alpha v1.22 implementation,

      `memory.high` = memory throttling factor * limits.memory  = 0.8 * 1000Mi = 800Mi

      This results in early throttling and puts the processed under heavy reclaim pressure at 800Mi memory usage levels. There's a significant difference of 200Mi between the memory throttling limit (800Mi) and memory usage hard limit (1000Mi). 

    * `requests.memory` = 500Mi

      `memory throttling factor` = 0.6

      `limits.memory` = 1000Mi
      
      As per Alpha v1.22 implementation,

      `memory.high` = memory throttling factor * limits.memory = 0.6 * 1000Mi = 600Mi
      
      Throttling occurs at 600Mi which is just a 100Mi over the requested memory. There's a significant difference of 400Mi between the memory throttle limit (600Mi) and memory usage hard limit (1000Mi).
  

3. Default throttling factor of 0.8 may be too aggressive for some applications that are latency sensitive and always use memory close to memory limits.

   For example, there are some known Java workloads that use 85% of the memory will start to get throttled once this feature is enabled by default. Hence the default 0.8 MemoryThrottlingFactor value may not be a good value for many applications due to inducing throttling too early.

<br>
Some more examples to compare memory.high using Alpha v1.22 and Alpha v1.27 are listed below:

| Limit 1000Mi <br /> Request, factor | Alpha v1.22: memory.high = memory throttling factor \* memory.limit (or node allocatable if memory.limit is not set) | Alpha v1.27: memory.high = floor[(requests.memory + memory throttling factor * (limits.memory or node allocatable memory - requests.memory))/pageSize] * pageSize assuming 1Mi pageSize
| -------------------------------- | ------------------------------------------------------- | ------------------------------------------------
| request 500Mi, factor 0.6        | 600Mi (very early throttling when memory usage is just 100Mi above requested memory; 400Mi unused) | 800Mi
| request 800Mi, factor 0.6        | no throttling (600 < 800 i.e. memory.high < memory.request => no throttling) | 920Mi
| request 1Gi, factor 0.6          | max | max
| request 500Mi, factor 0.8        | 800Mi (early throttling at 800Mi, when 200Mi is unused) | 900Mi
| request 850Mi, factor 0.8        | no throttling (800 < 850 i.e. memory.high < memory.request => no throttling) | 970Mi
| request 500Gi, factor 0.4        | no throttling (800 < 400  i.e. memory.high < memory.request => no throttling) | 700Mi

***Note***: As seen from the examples in the table, the formula used in Alpha v1.27 implementation eliminates the cases of memory.high being less than memory.request. However, it still can result in early throttling if memory throttling factor is set low. Hence, it is recommended to set a high memory throttling factor to avoid early throttling.

###### Quality of Service for Pods

In addition to the change in formula for memory.high, we are also adding the support for memory.high to be set as per `Quality of Service(QoS) for Pod` classes. Based on user feedback in Alpha v1.22, some users would like to opt-out of MemoryQoS on a per pod basis to ensure there is no early memory throttling. By making user's pods guaranteed, they will be able to do so. Guaranteed pod ,by definition, are not overcommitted, so memory.high does not provide significant value. 

Following are the different cases for setting memory.high as per QOS classes:
1. Guaranteed
Guaranteed pods by their QoS definition require memory requests=memory limits and are not overcommitted. Hence MemoryQoS feature is disabled on those pods by not setting memory.high. This ensures that Guaranteed pods can fully use their memory requests up to their set limit, and not hit any throttling.

2. Burstable
Burstable pods by their QoS definity require at least one container in the Pod with CPU or memory request or limit set. 

    Case I: When requests.memory and limits.memory are set, the forumula is used as-is: 
    ```
    memory.high = floor[ (requests.memory + memory throttling factor * (limits.memory - requests.memory)) / pageSize ] * pageSize
    ```

    Case II. When requests.memory is set, limits.memory is not set, we substitute limits.memory for node allocatable memory in the formula:
    ```
    memory.high = floor[ (requests.memory + memory throttling factor * (node allocatable memory - requests.memory))/ pageSize ] * pageSize
    ```

    Case III. When requests.memory is not set and  limits.memory is set, we set `requests.memory = 0` in the formula:
    ```
    memory.high = floor[ (memory throttling factor * limits.memory) / pageSize) ] * pageSize
    ```

3. BestEffort 
The pod gets a BestEffort class if limits.memory and requests.memory are not set. We set `requests.memory = 0` and substitute limits.memory for node allocatable memory in the formula:
    ```
    memory.high = floor[ (memoryThrottlingFactor * node allocatable memory) / pageSize) * pageSize
    ```

###### Alternative solutions for implementing memory.high
Alternative solutions that were discussed (but not preferred) before finalizing the implementation for memory.high are:
1. Allow customers to set memoryThrottlingFactor for each pod in annotations.

   Proposal: Add a new annotation for customers to set memoryThrottlingFactor to override kubelet level memoryThrottlingFactor.
	  * Pros
        * Allows more flexibility.
        * Can be quickly implemented.
	  * Cons
        * Customers might not need per pod memoryThrottlingFactor configuration.
        * It is too low-level detail to expose to customers.

2. Allow customers to set MemoryThrottlingFactor in pod yaml.

   Proposal: Add a new field in API for customers to set memoryThrottlingFactor to override kubelet level memoryThrottlingFactor.
    * Pros
        * Allows more flexibility.
    * Cons
        * Customers might not need per pod memoryThrottlingFactor configuration.
        * API changes take a lot of time, and we might eventually realize that the customers don’t need per pod level setting.
        * It is too low-level detail to expose to customers, and it is highly unlikely to get an API approval.

***[Preferred Alternative]***: Considering the cons of the alternatives mentioned above, adding support for memory QoS looks more preferrable over other solutions for following reasons:
  * Memory QOS complies with QOS which is a wider known concept. 
  * It is simple to understand as it requires setting only 1 kubelet configuration for setting memory throttling factor.
  * It doesn't involve API changes, and doesn't expose low-level detail to customers.

### User Stories (Optional)
#### Memory Sensitive Workload
Some workloads are sensitive to memory allocation and availability, slight delays may cause service outage. In this case, a mechanism is needed to ensure the quality of memory.
We must provide guarantee in both of the following aspects:
- Retain memory requests to reduce allocation latency
- Protect memory requests from being reclaimed

#### Node Availability
The stability of kubelet node is very important to users. As the key resource of the node, the availability of memory is the key factor for the stability of the node. We should do something to protect node reserved memory.

### Comparison with Memory Manager
The Memory Manager is a new component of kubelet ecosystem proposed to enable single-NUMA and multi-NUMA guaranteed memory allocation at topology level. Memory QoS proposal mainly uses cgroups v2 to improve the quality of memory requests, thereby improving the memory qos of `Guaranteed` and `Burstable` pods and even entire node. 
See also https://github.com/kubernetes/enhancements/tree/master/keps/sig-node/1769-memory-manager

### memory.low vs memory.min
In cgroups v2, `memory.low` is designed for best-effort memory protection which is more like "soft guarantee" and won't be reclaimed unless memory can't be reclaimed from any unprotected cgroups. `memory.min` is a bit aggressive. It will always retain specified amount of memory and it can be never reclaimed. When requirement is not satisfied, system OOM killer will be invoked. 


### Notes/Constraints/Caveats (Optional)

<!--
What are the caveats to the proposal?
What are some important details that didn't come across above?
Go in to as much detail as necessary here.
This might be a good place to talk about core concepts and how they relate.
-->

n/a

### Risks and Mitigations

<!--
What are the risks of this proposal, and how do we mitigate? Think broadly.
For example, consider both security and how this will impact the larger
Kubernetes ecosystem.

How will security be reviewed, and by whom?

How will UX be reviewed, and by whom?

Consider including folks who also work outside the SIG or subproject.
-->

The main risk of this proposal is too avoid throttling applications to early.
We intend to mitigate this by (1) setting a `memory.high` that is closer to the
limit and (2) only throttling when usage > request.

## Design Details
### Prerequisite
1. Kernel enables cgroups v2 unified hierarchy 
2. CRI runtime supports [cgroups v2 Unified Spec](https://github.com/opencontainers/runtime-spec/blob/7c549cb0939af03d5a2a8b271e2ad6871309e228/specs-go/config.go#L376) for container level
3. Kubelet enables `--enforce-node-allocatable=<pods, kube-reserved, system-reserved>` 

### Feature Gate
Set `--feature-gates=MemoryQoS=true` to enable the feature.

### Mapping Rules
#### Container/Pod
![](./memory-high.png)
1. If container sets `requests.memory`, we set `memory.min=pod.spec.containers[i].resources.requests[memory]` for container level cgroup
2. If any containers in pod sets `requests.memory`, we set `memory.min=sum(pod.spec.containers[i].resources.requests[memory])` for pod level cgroup
3. If container sets `limits.memory`, we set `memory.high=pod.spec.containers[i].resources.limits[memory] * memory throttling factor` for container level cgroup if `memory.high>memory.min` 
4. If container does't set `limits.memory`, we set `memory.high=node allocatable memory * memory throttling factor` for container level cgroup
5. If kubelet sets `--cgroups-per-qos=true`, we set `memory.min=sum(pod[i].spec.containers[j].resources.requests[memory])` to make ancestor cgroups propagation effective
6. There are no changes regarding memory limit, that is `memory.max=memory_limits` (same as existing cgroup v2 implementation)
#### Node
1. If kubelet sets `--enforce-node-allocatable=kube-reserved`, `--kube-reserved=[a]` and `--kube-reserved-cgroup=[b]`, we set `memory.min=[a]` for node level cgroup `[b]`
2. If kubelet sets `--enforce-node-allocatable=system-reserved`, `--system-reserved=[a]` and `--system-reserved-cgroup=[b]`, we set `memory.min=[a]` for node level cgroup `[b]`
3. If kubelet sets `--enforce-node-allocatable=pods`, we set `memory.min=sum(pod[i].spec.containers[j].resources.requests[memory])` for kubepods cgroup

### Interactive
New `Unified` field will be added in both CRI and QoS Manager for cgroups v2 extra parameters. It is recommended to has same semantics with opencontainers/runtime-spec#1040

- container level: `Unified` added in `LinuxContainerResources`
- pod/node level: `Unified` added in `cm.ResourceConfig` 

### Workflow
#### Container 
![](./container-level.png)

#### Pod 
![](./pod-level.png)

#### QoS 
![](./qos-level.png)

#### Node 
![](./node-level.png)

### Cgroup Hierarchy 
Container/Pod:
```
// Container
/cgroup2/kubepods/pod<UID>/<container-id>/memory.min=pod.spec.containers[i].resources.requests[memory]
/cgroup2/kubepods/pod<UID>/<container-id>/memory.high=(pod.spec.containers[i].resources.limits[memory]/node allocatable memory)*memory throttling factor // Burstable
// Pod
/cgroup2/kubepods/pod<UID>/memory.min=sum(pod.spec.containers[i].resources.requests[memory])
// QoS ancestor cgroup
/cgroup2/kubepods/burstable/memory.min=sum(pod[i].spec.containers[j].resources.requests[memory]) 
```

Node:
```
/cgroup2/kubepods/memory.min=sum(pod[i].spec.containers[j].resources.requests[memory])
/cgroup2/<kube-reserved-cgroup,system-reserved-cgroup>/memory.min=<kube-reserved,system-reserved>
```

### Cgroup v2 Support
After Kubernetes v1.19, kubelet can identify cgroups v2 and do the convention. Since [v1.0.0-rc93](https://github.com/opencontainers/runc/releases/tag/v1.0.0-rc93), runc supports `Unified` to pass through cgroups v2 parameters. So we use this variable to pass `memory.min` when cgroups v2 mode is detected.

### Container Runtime Interface (CRI) Changes
We need add new field `Unified` in CRI api which is basically passthrough for OCI spec Unified field and has same semantics: opencontainers/runtime-spec#1040

```
type LinuxContainerResources struct {
    ...
    Unified map[string]string `json:"unified,omitempty"
}
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

Overall Test plan:

For `Alpha`, unit tests were added to test functionality for container/pod/node level cgroup with containerd and CRI-O.

For second alpha iteration, (1.27), we plan to add new E2E node e2e tests to
validate the MemoryQoS settings are applied correctly.

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

- `pkg/kubelet/cm`: `02/09/2023` - `65.6`

##### Integration tests

<!--
This question should be filled when targeting a release.
For Alpha, describe what tests will be added to ensure proper quality of the enhancement.

For Beta and GA, add links to added tests together with links to k8s-triage for those tests:
https://storage.googleapis.com/k8s-triage/index.html
-->

n/a: plan to use node e2e tests (see below)

##### e2e tests

<!--
This question should be filled when targeting a release.
For Alpha, describe what tests will be added to ensure proper quality of the enhancement.

For Beta and GA, add links to added tests together with links to k8s-triage for those tests:
https://storage.googleapis.com/k8s-triage/index.html

We expect no non-infra related flakes in the last month as a GA graduation criteria.
-->

As part of alpha, we plan to add a new node e2e test to validate that the MemoryQoS settings will be correctly set on both pods as well as node allocatable.
The test will be reside in `test/e2e_node`.

### Graduation Criteria

#### Alpha Graduation
- [cgroup_v2](https://github.com/kubernetes/enhancements/tree/master/keps/sig-node/2254-cgroup-v2) is in `Alpha`
- Memory QoS is implemented for new feature gate
- Memory QoS is covered by proper tests
- Memory QoS supports containerd, cri-o

#### Beta Graduation
- [cgroup_v2](https://github.com/kubernetes/enhancements/tree/master/keps/sig-node/2254-cgroup-v2) is in `Beta`
- Metrics and graphs to show the amount of reclaim done on a cgroup as it moves from below-request to above-request to throttling
- Memory QoS is covered by unit and e2e-node tests
- Memory QoS supports containerd, cri-o and dockershim

#### GA Graduation
- [cgroup_v2](https://github.com/kubernetes/enhancements/tree/master/keps/sig-node/2254-cgroup-v2) is in `GA`
- Memory QoS has been in beta for at least 2 releases
- Memory QoS sees use in 3 projects or articles
- Memory QoS is covered by conformance tests

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
  - Feature gate name: MemoryQoS
  - Components depending on the feature gate: kubelet

###### Does enabling the feature change any default behavior?

<!--
Any change of default behavior may be surprising to users or break existing
automations, so be extremely careful here.
-->
Yes, the kubelet will set `memory.min` for Guaranteed and Burstable pod/container level cgroup. It also will set `memory.high` for burstable container, which may cause memory allocation throttle. `memory.min` for qos or node level cgroup will be set when `--cgroups-per-qos` or `--enforce-node-allocatable` is satisfied.

###### Can the feature be disabled once it has been enabled (i.e. can we roll back the enablement)?

<!--
Describe the consequences on existing workloads (e.g., if this is a runtime
feature, can it break the existing applications?).

NOTE: Also set `disable-supported` to `true` or `false` in `kep.yaml`.
-->
Yes, related cgroups can be rolled back, `memory.min/memory.high` will reset to default value.

###### What happens if we reenable the feature if it was previously rolled back?
The kubelet will reconcile `memory.min/memory.high` with related cgroups.

###### Are there any tests for feature enablement/disablement?

<!--
The e2e framework does not currently support enabling or disabling feature
gates. However, unit tests in each component dealing with managing data, created
with and without the feature, are necessary. At the very least, think about
conversion tests if API types are being modified.
-->
Yes, some unit tests are exercised with the feature both enabled and disabled to verify proper behavior in both cases. When enabled, we test `memory.min/memory.high` for workloads and node cgroups whether it is proper value. When transitioning from enabled to disabled happens, we verify `memory.min/memory.high` whether be reset to default value.

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

###### How can someone using this feature know that it is working for their instance?

<!--
For instance, if this is a pod-related feature, it should be possible to determine if the feature is functioning properly
for each individual pod.
Pick one more of these and delete the rest.
Please describe all items visible to end users below with sufficient detail so that they can verify correct enablement
and operation of this feature.
Recall that end users cannot usually observe component logs or access metrics.
-->

- [ ] Events
  - Event Reason: 
- [ ] API .status
  - Condition name: 
  - Other field: 
- [ ] Other (treat as last resort)
  - Details:

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

###### What are the SLIs (Service Level Indicators) an operator can use to determine the health of the service?

<!--
Pick one more of these and delete the rest.
-->

- [ ] Metrics
  - Metric name:
  - [Optional] Aggregation method:
  - Components exposing the metric:
- [ ] Other (treat as last resort)
  - Details:

###### Are there any missing metrics that would be useful to have to improve observability of this feature?

<!--
Describe the metrics themselves and the reasons why they weren't added (e.g., cost,
implementation difficulties, etc.).
-->

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

<!--
For alpha, this section is encouraged: reviewers should consider these questions
and attempt to answer them.

For beta, this section is required: reviewers must answer these questions.

For GA, this section is required: approvers should be able to confirm the
previous answers based on experience in the field.
-->

###### Will enabling / using this feature result in any new API calls?

No, new API calls will be generated.

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

###### Will enabling / using this feature result in introducing new API types?

<!--
Describe them, providing:
  - API type
  - Supported number of objects per cluster
  - Supported number of objects per namespace (for namespace-scoped objects)
-->

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

No.

###### Will enabling / using this feature result in increasing time taken by any operations covered by existing SLIs/SLOs?

<!--
Look at the [existing SLIs/SLOs].

Think about adding additional work or introducing new steps in between
(e.g. need to do X to start a container), etc. Please describe the details.

[existing SLIs/SLOs]: https://git.k8s.io/community/sig-scalability/slos/slos.md#kubernetes-slisslos
-->

No.

###### Will enabling / using this feature result in non-negligible increase of resource usage (CPU, RAM, disk, IO, ...) in any components?

<!--
Things to keep in mind include: additional in-memory state, additional
non-trivial computations, excessive access to disks (including increased log
volume), significant amount of data sent and/or received over network, etc.
This through this both in small and large cases, again with respect to the
[supported limits].

[supported limits]: https://git.k8s.io/community//sig-scalability/configs-and-limits/thresholds.md
-->

No.

###### Can enabling / using this feature result in resource exhaustion of some node resources (PIDs, sockets, inodes, etc.)?

<!--
Focus not just on happy cases, but primarily on more pathological cases
(e.g. probes taking a minute instead of milliseconds, failed pods consuming resources, etc.).
If any of the resources can be exhausted, how this is mitigated with the existing limits
(e.g. pods per node) or new limits added by this KEP?

Are there any tests that were run/should be run to understand performance characteristics better
and validate the declared limits?
-->

No, resources like PIDs, sockets, inodes will not be affected. However,
additional memory throttling can be experienced which is intended by this
feature.

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
- 2020/03/14: initial proposal
- 2020/05/05: target Alpha to v1.22

## Drawbacks

<!--
Why should this KEP _not_ be implemented?
-->

The main drawbacks are concerns about unintended memory throttling and
additional complexity due to to utilization of several new cgroupv2
based memory controls (i.e memory.low, memory.high, etc).

However, we believe that impact of unintended throttling will be minimized due
to a high throttling factor (see above) and the additional complexity is
justified due to the additional resource management benefits

## Alternatives

<!--
What other approaches did you consider, and why did you rule them out? These do
not need to be as detailed as the proposal, but should include enough
information to express the idea and why it was not acceptable.
-->

Please refer to alternatives mentioned above in the [proposal](#proposal) section,
which discusses the alternatives and changes from the original alpha design to
the newly updated alpha design.

## Infrastructure Needed (Optional)

<!--
Use this section if you need things from the project/SIG. Examples include a
new subproject, repos requested, or GitHub details. Listing these here allows a
SIG to get the process for these resources started right away.
-->

n/a, not new infrastructure is needed, this KEP aims to reuse the existing node e2e jobs and framework.
