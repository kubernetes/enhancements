# KEP-4800: Split UncoreCache Toplogy Awareness in CPU Manager

<!-- toc -->
- [Release Signoff Checklist](#release-signoff-checklist)
- [Summary](#summary)
- [Motivation](#motivation)
  - [Goals](#goals)
  - [Non-Goals](#non-goals)
- [Proposal](#proposal)
  - [User Stories (Optional)](#user-stories-optional)
    - [Story 1](#story-1)
    - [Story 2](#story-2)
  - [Notes/Constraints/Caveats (Optional)](#notesconstraintscaveats-optional)
  - [Risks and Mitigations](#risks-and-mitigations)
- [Design Details](#design-details)
  - [Test Plan](#test-plan)
      - [Prerequisite testing updates](#prerequisite-testing-updates)
      - [Unit tests](#unit-tests)
      - [Integration tests](#integration-tests)
      - [e2e tests](#e2e-tests)
  - [Graduation Criteria](#graduation-criteria)
    - [Alpha](#alpha)
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
- [Infrastructure Needed (Optional)](#infrastructure-needed-optional)
<!-- /toc -->

## Release Signoff Checklist

Items marked with (R) are required *prior to targeting to a milestone / release*.

- [x] (R) Enhancement issue in release milestone, which links to KEP dir in [kubernetes/enhancements] (not the initial KEP PR)
- [x] (R) KEP approvers have approved the KEP status as `implementable`
- [x] (R) Design details are appropriately documented
- [ ] (R) Test plan is in place, giving consideration to SIG Architecture and SIG Testing input (including test refactors)
  - [ ] e2e Tests for all Beta API Operations (endpoints)
  - [ ] (R) Ensure GA e2e tests meet requirements for [Conformance Tests](https://github.com/kubernetes/community/blob/master/contributors/devel/sig-architecture/conformance-tests.md) 
  - [ ] (R) Minimum Two Week Window for GA e2e tests to prove flake free
- [ ] (R) Graduation criteria is in place
  - [x] (R) [all GA Endpoints](https://github.com/kubernetes/community/pull/1806) must be hit by [Conformance Tests](https://github.com/kubernetes/community/blob/master/contributors/devel/sig-architecture/conformance-tests.md) 
- [x] (R) Production readiness review completed
- [ ] (R) Production readiness review approved
- [x] "Implementation History" section is up-to-date for milestone
- [ ] User-facing documentation has been created in [kubernetes/website], for publication to [kubernetes.io]
- [x] Supporting documentation—e.g., additional design documents, links to mailing list discussions/SIG meetings, relevant PRs/issues, release notes

[kubernetes.io]: https://kubernetes.io/
[kubernetes/enhancements]: https://git.k8s.io/enhancements
[kubernetes/kubernetes]: https://git.k8s.io/kubernetes
[kubernetes/website]: https://git.k8s.io/website

## Summary

This KEP proposes to introduce a new CPU Manager static policy option, "prefer-align-cpus-by-uncorecache", that groups CPU resources by uncore cache where possible.
The opt-in feature changes the cpu assignment algorithm to add sorting by uncore cache and then taking cpus aligned to the same uncore cache, where possible.
In cases where numbers of cpu requested exceeds number of cpus grouped in the same uncore cache, the algorithm attempts best-effort to reduce assignments of cpus across minimal numbers of uncore caches.
If the cpumanager cannot align optimally, it will still admit the workload as before. Uncore cache assignment will be preferred but not a requirement for this feature.
This feature does not introduce the requirement of aligned by uncorecache but the preference to alignment of uncorecache.

## Motivation

Main motivation for this proposal is to reduce noisy neighbor scenarios that occur on systems with split uncore cache, which is available on both x86 and ARM architecture.
The challenge with current kubelet’s cpu manager is that it is unaware of split uncore architecture and spreads CPU assignments across distributed uncore caches. This creates a noisy neighbor problem where multiple pods/containers are sharing the same uncore cache. In addition, pods spread against multiple uncore caches see higher latency and reduced performance due to inter-cache access latency. For workload use cases that are sensitive to latency and are performance deterministic, minimizing the noisy neighbor condition in uncore cache can have significant improvements in performance.

![Figure to the motivation](./images/uncore-alignment-motivation.png)

The figure below highlights performance gain when pod placement is aligned to an uncore cache group. In this example HammerDB TPROC-C/My-SQL is the workload deployed. There was a 18% uplift in performance when uncore cache aligned compared to default behavior. Other workloads may see higher gains.

![Figure to the results of uncore cache alignment](https://github.com/ajcaldelas/enhancements/blob/split_l3_cache/keps/sig-node/4800-cpumanager-split-uncorecache/images/results.PNG) ![Figure of results](https://github.com/ajcaldelas/enhancements/blob/c7e423439608001cc837e2f5629dc87935353611/keps/sig-node/4800-cpumanager-split-l3cache/images/chart.png)

### Goals

- Introduce a new CPU Manager policy option that assigns CPU within the same grouping of uncore cache to container scope.  
- Minimize the number of CPUs assignments from different uncore cache grouping to pods and container scope.

### Non-Goals

- This proposal does not aim to modify CPU assignments for CPU Manager policy set to none.
- This does not alter the behavior of existing static policy options such as full-pcpus-only.

## Proposal

- Add a new static policy option to fine-tune of CPU assignments to have a preference to align by uncore cache grouping.

- Topology struct already contains CPUDetails which is a map of CPUInfo.  CPUInfo knows about NUMA, socket, and core IDs associated with a CPU.  We just need to add a new member called uncorecacheId into the CPUInfo struct that tracks whether the CPU is part of a split uncore caches. Functionality to support this was merged in cadvisor with these pull-requests [cadvisor/pr-2849](https://github.com/google/cadvisor/pull/2849) and [cadvisor/pr-2847](https://github.com/google/cadvisor/pull/2847/)

- Handle enablement of the policy option in pkg/kubelet/cm/cpumanager/policy_options.go and check the validity of the user flag and capability of the system.  If topology does not support split uncore caches, grouping by uncore cache static policy will be ignored.  

- Modify the "Allocate" static policy to check for the option, prefer-align-cpus-by-uncorecache. For platforms where SMT is enabled, prefer-align-cpus-by-uncorecache will continue to follow default behaviour and try to allocate full cores when possible. prefer-align-cpus-by-uncorecache can be enabled along with full-pcpus-only and enforce full core assignment with uncorecache alignment.

- prefer-align-cpus-by-uncorecache will be compatible with the default CPU allocation logic with future plans to be compatible with the options distribute-cpus-across-numa and distribute-cpus-across-cores.

### User Stories (Optional)

#### Story 1

- As a HPC user, I want to extract the best performance possible and reduce latency, so that I can extract the most value when deploying applications in Kubernetes. I can use the static CPU Manager Policy option, but on split uncore cache processors, my application can experience latency when assigned CPUs across multiple uncore caches. In order to maximize performance, I want to minimize the distribution across uncore caches to minimize the latency. I want a feature I can enable so the CPU allocation logic inside the kubelet will automatically handle this for me for larger clusters for simple deployment and efficiency.

#### Story 2

- As a Networking/Telco Engineer, my application is latency sensitive, to which uncore cache misalignment is a contributing factor. I want my application to be statically assigned to CPUs that correspond to one single uncore cache so that I can get the maximum throughput from my application. Additionally, I have applications that are also static and of different CPU size requirements. The default Static CPU Manager Policy will result in my applications being assigned to two or more uncore caches. I have a large multi-node Kuberentes cluster. I want a cpu allocation logic to automatically assign CPUs for the best performance across the fewest amount of uncore cahces based on available CPUs without having to worry about what order/size I deploy my applications so I can ensure I get the best performance possible. 

### Notes/Constraints/Caveats (Optional)

- The name UncoreCache is directly derived from cAdvisor that is being used as a package. 
- Using the flow in `CPUManagerPolicy{Alpha,Beta}Options` from [xref](https://github.com/kubernetes/kubernetes/blob/af879aebb1a866a2f0e45bb33c09a1cc8f7acc45/pkg/features/kube_features.go#L110C1-L117C83) which is used to avoid proliferation of feature gates. 

### Risks and Mitigations

- Risk: Enabling or disabling the feature might lead to unexpected behavior
  - Mitigation: Feature is enabled by a static policy option flag
- Risk: The new feature might interfere with existing functionality
  - Mitigation: It does not change behavior of non-static policy and preserves the behavior of other static options
- Risk: Inconsistent configuration could cause scheduling issues
  - Mitigation: Failure will occur during runtime if mismatch between options occurs, preventing the Pod from being scheduled incorrectly or leading a non-optimal aligment

## Design Details

To make Kubelet be uncore cache aware for CPU sensitive workloads the CPU allocation, `UncoreCacheID` has been added to the `CPUInfo` and `CPUTopology` structures that are provided by `cAdvisor` during the CPU discovery process.

A new CPUManager policy option, prefer-align-cpus-by-uncorecache, will be introduced to enable this feature. When enabled, the existing default scheduling function will be modified to account for UncoreCache as a preference. The allocation process will first attempt to allocate CPUs from an entire socket. Next, it will attempt to allocate CPUs within the logical boundary of a NUMA node within the socket. Finally, the allocation will consider the subset of CPUs associated with each UncoreCache within the previously selected CPUs. While UncoreCache allocation is preferred, it is not strictly required this policy will not fail if alignment is not possible. Since this policy extends the CPUManager’s Static policy, it will only apply to guaranteed pods with whole CPU requests.

The algorithm for prefer-align-cpus-by-uncorecache will be implemented to follow the default packed behavior of the existing static CPUManager allocation with the introduction of a uncorecache hierarchy. This means that when a guaranteed container requires CPUs that are equal to or greater than the size of a NUMA or socket, the CPU allocation will behave as usual and schedule the full socket or NUMA. The scheduled CPUs will be subtracted from the quantity of CPUs required for the container.

When/once the required CPUs for a container are less than the number of CPUs within a NUMA, the algorithm will be implemented as follows:
1. Scan each socket in numerical order. If required CPUs are less than the size of available CPUs on the socket, pick this socket of CPUs.
2. Within the chosen socket, scan each NUMA in numerical order. If the required CPUs are less than the number of the available CPUs on the NUMA, pick the subset of CPUs that correspond to this NUMA.
3. Within the chosen NUMA, scan through every uncore cache index in numerical order:
   - If the required CPUs are greater than or equal the size of the **total** amount of CPUs to an uncore cache, assign the full uncore cache CPUs if they are all **unscheduled** within the group. Subtract the amount of scheduled CPUs from the quantity of required CPUs. 
     - Continue scanning uncore caches in numerical order and assigning full uncore cache CPU groups until the required quantity is less than the **total** number of CPUs to a uncore cache group.
   - If the required CPUs are less than the size of the **total** amount of CPUs to an uncore cache, scan through each uncore cache index in numerical order starting from the first index, and assign CPUs if there are enough **available** CPUs within the uncore cache group.
4. If the required amount of CPUs cannot fit within an uncore cache group and there are enough schedulable CPUs on the node, assign cores in numerical order.
5. Container will not be scheduled on the node if there are not enough CPUs to satisfy the container requirements. 

In the case where the NUMA boundary is larger than a socket (setting NPS0 on a dual-socket system), the allocatable pool of CPUs does not exapnd beyond the respective socket when NUMA hints are provided in the above scheduling policy. The filtered subset of CPUs will still remain within the socket. 

In the case where the NUMA boundary is smaller than an uncore cache (enabling Sub-NUMA clustering on a monolithic cache system), the allocatable pool of CPUs does not expand beyong the respective NUMA boundary when uncore cache hints are provided in the above scheduling policy. The filtered subset of CPUs will still remain within the NUMA boundary.

This scheduling policy will minimize the distribution of containers across uncore caches to improve performance while still maintaining the default packed logic. The scope will be initially be narrowed to implement uncore cache alignment to the default static scheduling behavior. The table below summarizes future enhancement plans to implement uncore cache alignment to be compatible with the distributed scheduling policies to reduce contention/noisy neighbor effects.


| Compatibility | alpha | beta | GA |
| --- | --- | --- | --- |
| full-pcpus-only | x | x | x |
| distribute-cpus-across-numa  |   | x | x |
| align-by-socket | x | x | x |
| distribute-cpus-across-cores |   | x | x |

### Test Plan

The different configurations can help show the above scheduling policy behavior of how containers are assigned CPUs when the prefer-align-cpus-by-uncorecache feature is enabled with containers of different CPU sizes and order. These different configurations can also help outline unit and e2e test cases. Below are a few examples of the scheduling policy across different processors with different NUMA Per Socket (NPS) settings.

HW Test Matrix

- 1P AMD EPYC 7303 32C (smt-on) NPS=1 
- 1P AMD EPYC 7303 32C (smt-off)  NPS=1 
- 2P AMD EPYC 9754 128C (smt-on) NPS=1   
- 2P AMD EPYC 9654 96C (smt-off) NPS=2   
- 2P Intel Xeon Platinum 8490H 60c  (hyperthreading off)
- 2P Intel Xeon Platinum 8490H 60c (hyperthreading on) with Sub-NUMA Clustering
- 1P Intel Core i7-12850HX
- 1P ARM Ampere Altra 128c
- AWS Graviton

Case #1. 1P AMD EPYC 7303 32C (smt-off) NPS=1

```
NumCores:              32,
NumCPUs:               32,
NumSockets:            1,
NumCPUsPerUncoreCache: 8,
NPS:                   1
UncoreCache Topology: {
    0: {CPUSet: 0-7,    NUMAID: 0,  SocketID: 0},
    1: {CPUSet: 8-15,   NUMAID: 0,  SocketID: 0},
    2: {CPUSet: 16-23,  NUMAID: 0,  SocketID: 0},
    3: {CPUSet: 24-31,  NUMAID: 0,  SocketID: 0}
}
CPUAssignments: { //pod deployment size, expected result
    Reserved:           {CPUs: 1,   UncoreCacheID: 0,        CPUSet: 0}
    Container1:         {CPUs: 2,   UncoreCacheID: 0,        CPUSet: 1-2},
    Container2:         {CPUs: 20,  UncoreCacheID: 0-2,      CPUSet: 3-6,8-15,16-23},
    Container3:         {CPUs: 6,   UncoreCacheID: 3,        CPUSet: 24-31}
}
```

Case #2. 1P AMD EPYC 7303 32C (smt-on) NPS=1 
```
NumCores:                   32,
NumCPUs:                    64,
NumSockets:                 1,
NumCPUsPerUncoreCache:      16,
NPS:                        1,
ReservedCPUs:               4

UncoreCache Topology: {
    0: {CPUSet: 0-7,32-39    NUMAID: 0,  SocketID: 0},
    1: {CPUSet: 8-15,40-47   NUMAID: 0,  SocketID: 0},
    2: {CPUSet: 16-23,48-55  NUMAID: 0,  SocketID: 0},
    3: {CPUSet: 24-31,56-63  NUMAID: 0,  SocketID: 0}
}

CPUAssignments: { //pod deployment size, expected result
    Reserved:         {CPUs: 4,   UncoreCacheID: 0,     CPUSet: 0-1,32-33},
    Container1:       {CPUs: 2,   UncoreCacheID: 0,     CPUSet: 2,34},
    Container2:       {CPUs: 22,  UncoreCacheID: 0-1,   CPUSet: 3-5,8-15,35-37,40-47},
    Container3:       {CPUs: 22,  UncoreCacheID: 2-3,   CPUSet: 16-26,48-58},
    Container4:       {CPUs: 12,  UncoreCacheID: 0,3,   CPUSet: 6-7,27-30,38-39,59-62},
    Container5:       {CPUs: 2,   UncoreCacheID: -1,    CPUSet: -1}
}
```

Case#3: 2P AMD EPYC 9754 128C (smt-on) NPS=1

```
NumCoresPerSocket:  128,
NumSockets:         2,
NumCPUs:            512,
NumCPUsPerUncoreCache:      16,
NPS:                1,
ReservedCPUs:       2

UncoreCache Topology: {
    0:  {CPUSet: 0-7,256-263,        NUMAID: 0,  SocketID: 0},
    1:  {CPUSet: 8-15,264-271,       NUMAID: 0,  SocketID: 0},
    2:  {CPUSet: 16-23,272-279,      NUMAID: 0,  SocketID: 0},
    3:  {CPUSet: 24-31,280-287,      NUMAID: 0,  SocketID: 0},
    4:  {CPUSet: 32-39,288-295,      NUMAID: 0,  SocketID: 0},
    5:  {CPUSet: 40-47,296-303,      NUMAID: 0,  SocketID: 0},
    6:  {CPUSet: 48-55,304-311,      NUMAID: 0,  SocketID: 0},
    7:  {CPUSet: 56-63,312-319,      NUMAID: 0,  SocketID: 0},
    8:  {CPUSet: 64-71,320-327,      NUMAID: 0,  SocketID: 0},
    9:  {CPUSet: 72-79,328-335,      NUMAID: 0,  SocketID: 0},
    10: {CPUSet: 80-87,336-343,      NUMAID: 0,  SocketID: 0},
    11: {CPUSet: 88-95,344-351,      NUMAID: 0,  SocketID: 0},
    12: {CPUSet: 96-103,352-359,     NUMAID: 0,  SocketID: 0},
    13: {CPUSet: 104-111,360-367,    NUMAID: 0,  SocketID: 0},
    14: {CPUSet: 112-119,368-375,    NUMAID: 0,  SocketID: 0},
    15: {CPUSet: 120-127,376-383,    NUMAID: 0,  SocketID: 0},
    16: {CPUSet: 128-135,384-391,    NUMAID: 1,  SocketID: 1},
    17: {CPUSet: 136-143,392-399,    NUMAID: 1,  SocketID: 1},
    18: {CPUSet: 144-151,400-407,    NUMAID: 1,  SocketID: 1},
    19: {CPUSet: 152-159,408-415,    NUMAID: 1,  SocketID: 1},
    20: {CPUSet: 160-167,416-423,    NUMAID: 1,  SocketID: 1},
    21: {CPUSet: 168-175,424-431,    NUMAID: 1,  SocketID: 1},
    22: {CPUSet: 167-183,432-439,    NUMAID: 1,  SocketID: 1},
    23: {CPUSet: 184-191,440-447,    NUMAID: 1,  SocketID: 1},
    24: {CPUSet: 192-199,448-455,    NUMAID: 1,  SocketID: 1},
    25: {CPUSet: 200-207,456-463,    NUMAID: 1,  SocketID: 1},
    26: {CPUSet: 208-215,464-471,    NUMAID: 1,  SocketID: 1},
    27: {CPUSet: 216-223,472-479,    NUMAID: 1,  SocketID: 1},
    28: {CPUSet: 224-231,480-487,    NUMAID: 1,  SocketID: 1},
    29: {CPUSet: 232-239,488-495,    NUMAID: 1,  SocketID: 1},
    30: {CPUSet: 240-247,496-503,    NUMAID: 1,  SocketID: 1},
    31: {CPUSet: 248-255,504-511,    NUMAID: 1,  SocketID: 1}
}

CPUAssignments: {
    Reserved:         {CPUs: 2,   UncoreCacheID: 0,         CPUSet: 0,256,                      NUMAID: 0,  SocketID: 0},
    Container1:       {CPUs: 2,   UncoreCacheID: 0,         CPUSet: 1,257,                      NUMAID: 0,  SocketID: 0},
    Container2:       {CPUs: 22,  UncoreCacheID: 0-1,       CPUSet: 2-4,8-15,258-260,264-271,   NUMAID: 0,  SocketID: 0},
    Container3:       {CPUs: 8,   UncoreCacheID: 2,         CPUSet: 16-19,272-275,              NUMAID: 0,  SocketID: 0},
    Container4:       {CPUs: 22,  UncoreCacheID: 0,3,       CPUSet: 5-7,24-31,261-263,280-287,  NUMAID: 0,  SocketID: 0},
    Container5:       {CPUs: 12,  UncoreCacheID: 4,         CPUSet: 32-35,288-291,              NUMAID: 0,  SocketID: 0},
    Container6:       {CPUs: 8,   UncoreCacheID: 2,         CPUSet: 20-23,276-279,              NUMAID: 0,  SocketID: 0},
    Container7:       {CPUs: 12,  UncoreCacheID: 5,         CPUSet: 120-125,376-381,            NUMAID: 0,  SocketID: 0}
}

Case#4: 2P AMD EPYC 9654 96C (smt-off) NPS=2

NumCoresPerSocket:          96,
NumSockets:                 2,
NumCPUs:                    192,
NumCPUsPerUncoreCache:      8,
NPS:                        2,
ReservedCPUs:               4

UncoreCache Topology: {
    0:  {CPUSet: 0-7,       NUMAID: 0,  SocketID: 0},
    1:  {CPUSet: 8-15,      NUMAID: 0,  SocketID: 0},
    2:  {CPUSet: 16-23,     NUMAID: 0,  SocketID: 0},
    3:  {CPUSet: 24-31,     NUMAID: 0,  SocketID: 0},
    4:  {CPUSet: 32-39,     NUMAID: 0,  SocketID: 0},
    5:  {CPUSet: 40-47,     NUMAID: 0,  SocketID: 0},
    6:  {CPUSet: 48-55,     NUMAID: 1,  SocketID: 0},
    7:  {CPUSet: 56-63,     NUMAID: 1,  SocketID: 0},
    8:  {CPUSet: 64-71,     NUMAID: 1,  SocketID: 0},
    9:  {CPUSet: 72-79,     NUMAID: 1,  SocketID: 0},
    10: {CPUSet: 80-87,     NUMAID: 1,  SocketID: 0},
    11: {CPUSet: 88-95,     NUMAID: 1,  SocketID: 0},
    12: {CPUSet: 96-103,    NUMAID: 2,  SocketID: 1},
    13: {CPUSet: 104-111,   NUMAID: 2,  SocketID: 1},
    14: {CPUSet: 112-119,   NUMAID: 2,  SocketID: 1},
    15: {CPUSet: 120-127,   NUMAID: 2,  SocketID: 1},
    16: {CPUSet: 128-135,   NUMAID: 2,  SocketID: 1},
    17: {CPUSet: 136-143,   NUMAID: 2,  SocketID: 1},
    18: {CPUSet: 144-151,   NUMAID: 3,  SocketID: 1},
    19: {CPUSet: 152-159,   NUMAID: 3,  SocketID: 1},
    20: {CPUSet: 160-167,   NUMAID: 3,  SocketID: 1},
    21: {CPUSet: 168-175,   NUMAID: 3,  SocketID: 1},
    22: {CPUSet: 167-183,   NUMAID: 3,  SocketID: 1},
    23: {CPUSet: 184-191,   NUMAID: 3,  SocketID: 1}
}

CPUAssignments: {
    Reserved:         {CPUs: 4,   UncoreCacheID: 0,         CPUSet: 0-4         NUMAID: 0,      SocketID: 0},
    Container1:       {CPUs: 2,   UncoreCacheID: 0,         CPUSet: 5-6,        NUMAID: 0,      SocketID: 0},
    Container2:       {CPUs: 22,  UncoreCacheID: 1-3,       CPUSet: 8-29,       NUMAID: 0,      SocketID: 0},
    Container3:       {CPUs: 8,   UncoreCacheID: 4,         CPUSet: 32-39,      NUMAID: 0,      SocketID: 0},
    Container4:       {CPUs: 12,  UncoreCacheID: 6-7,       CPUSet: 48-59,      NUMAID: 1,      SocketID: 0},
    Container5:       {CPUs: 22,  UncoreCacheID: 8-10,      CPUSet: 64-85,      NUMAID: 1,      SocketID: 0},
    Container6:       {CPUs: 22,  UncoreCacheID: 12-14,     CPUSet: 96-117,     NUMAID: 2,      SocketID: 1}
}
```

Case#5: 2P Intel Xeon Platinum 8490H
```
NumCoresPerSocket:          60,
NumSockets:                 2,
NumCPUs:                    120,
NumCPUsPerUncoreCache:      60,
NPS:                        1,
ReservedCPUs:               4

UncoreCache Topology: {
    0: {CPUSet: 0-59,   NUMAID: 0,  SocketID: 0},
    1: {CPUSet: 60-119, NUMAID: 1,  SocketID: 1}
}

CPUAssignments: {
    Reserved:         {CPUs: 4,   UncoreCacheID: 0,     CPUSet: 0-3,        NUMAID: 0,      SocketID: 0},
    Container1:       {CPUs: 6,   UncoreCacheID: 0,     CPUSet: 4-9,        NUMAID: 0,      SocketID: 0},
    Container2:       {CPUs: 22,  UncoreCacheID: 0,     CPUSet: 10-31,      NUMAID: 0,      SocketID: 0},
    Container3:       {CPUs: 8,   UncoreCacheID: 0,     CPUSet: 32-39,      NUMAID: 0,      SocketID: 0},
    Container4:       {CPUs: 12,  UncoreCacheID: 0,     CPUSet: 40-51,      NUMAID: 0,      SocketID: 0},
    Container5:       {CPUs: 4,   UncoreCacheID: 0,     CPUSet: 52-55,      NUMAID: 0,      SocketID: 0}
}
```

<!--
**Note:** *Not required until targeted at a release.*
The goal is to ensure that we don't accept enhancements with inadequate testing.

All code is expected to have adequate tests (eventually with coverage
expectations). Please adhere to the [Kubernetes testing guidelines][testing-guidelines]
when drafting this test plan.

[testing-guidelines]: https://git.k8s.io/community/contributors/devel/sig-testing/testing.md
-->

[x] I/we understand the owners of the involved components may require updates to
existing tests to make this code solid enough prior to committing the changes necessary
to implement this enhancement.

##### Prerequisite testing updates

<!--
Based on reviewers feedback describe what additional tests need to be added prior
implementing this enhancement to ensure the enhancements have also solid foundations.
-->

##### Unit tests

We plan on adding/modifying functions to the following files to create the uncorecache alignment feature:
- pkg/kubelet/cm/cpumanager/cpu_assignment.go
- /pkg/kubelet/cm/cpumanager/cpu_manager.go
- /pkg/kubelet/cm/cpumanager/policy_options.go
- /pkg/kubelet/cm/cpumanager/topology/topology.go

Existing topoology test cases will be modified to include uncorecache topology. All modified and added functions will have new test cases.


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
- `/pkg/kubelet/cm/cpumanager/cpu_assignment.go`: `2024-08-26` - `93.5%`
- `/pkg/kubelet/cm/cpumanager/cpu_manager.go`: `2024-08-26` - `74.5%`
- `/pkg/kubelet/cm/cpumanager/policy_static.go`: `2024-08-26` - `89.5%`
- `/pkg/kubelet/cm/cpumanager/topology/topology.go`: `2024-9-30` - `93.2%`

##### Integration tests

N/A. This feature requires a e2e test for testing.

##### e2e tests

- For e2e testing, checks will be added to determine if the node has a split uncore cache topology. If node does not meet the requirement to have multiple uncore caches, the added tests will be skipped. 
- e2e testing should cover the deployment of a pod that is following uncore cache alignment. CPU assignment can be determined by podresources API and programatically cross-referenced to syfs topology information to determine proper uncore cache alignment.
- For e2e testing, guaranteed pods will be deployed with various CPU size requirements on our own baremetal instances across different vendor architectures and confirming the CPU assignments to uncore cache core groupings. This feature is intended for baremetal only and not cloud instances.
- Update CI to test GCP instances of different architectures utilizing uncore cache alignment feature.


### Graduation Criteria

#### Alpha

- Feature implemented behind a feature gate flag option
- E2E Tests will be skipped until nodes with uncore cache can be provisioned within CI hardware. Work is ongoing to add required systems (https://github.com/kubernetes/k8s.io/issues/7339). E2E testing will be required to graduate to beta.

### Upgrade / Downgrade Strategy

N/A


### Version Skew Strategy

N/A


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

To enable this feature requires enabling the feature gates for static policy in the Kubelet configuration file for the CPUManager feature gate and add the policy option for uncore cache alignment


###### How can this feature be enabled / disabled in a live cluster?

For `CPUManager` it is a requirement going from `none` to `static` policy cannot be done dynamically because of the `cpu_manager_state file`. The node needs to be drained and the policy checkpoint file (`cpu_manager_state`) need to be removed before restarting Kubelet. This feature specifically relies on the `static` policy being enabled.

- [x] Feature gate (also fill in values in `kep.yaml`)
  - Feature gate name: `CPUManagerAlphaPolicyOptions`
  - Components depending on the feature gate: `kubelet`
- [x] Other
  - Describe the mechanism: Change the `kubelet` configuration to set a `CPUManager` policy of static then setting the policy option of `prefer-align-cpus-by-uncorecache`
  - Will enabling / disabling the feature require downtime of the control
    plane? No.
  - Will enabling / disabling the feature require downtime or reprovisioning
    of a node? Yes, a `kubelet` restart is required for changes to take place.

###### Does enabling the feature change any default behavior?

No, to enable this feature, it must be explicitly set in the `CPUManager` static policy and the policy option `prefer-align-cpus-by-uncorecache` must be set.

###### Can the feature be disabled once it has been enabled (i.e. can we roll back the enablement)?

Yes this feature can be disabled it will just require a restart of `kubelet`. The Kubelet configuration will need to be set with the static policy option and prefer-align-cpus-by-uncorecache flag removed.


###### What happens if we reenable the feature if it was previously rolled back?

Feature will be enabled. Proper drain of node and restart of kubelet required. Feature is not intended to be enabled/disabled dynamically, similar to static policy.

###### Are there any tests for feature enablement/disablement?

Option is not enabled dynamically. To enable/disable option, cpu_manager_state must be removed and kubelet must be restarted.
Unit tests will be implemented to test if the feature is enabled/disabled.
E2e node serial suite can be use to test the enablement/disablement of the feature since it allows the kubelet to be restarted.


### Rollout, Upgrade and Rollback Planning

<!--
This section must be completed when targeting beta to a release.
-->

###### How can a rollout or rollback fail? Can it impact already running workloads?

Kubelet restarts are not expected to impact existing CPU assignments to already running workloads


###### What specific metrics should inform a rollback?

Increased pod startup time/latency 

###### Were upgrade and rollback tested? Was the upgrade->downgrade->upgrade path tested?

N/A, because way to upgrade and rollback would be the same process of removing the CPU Manager state file and drain the node of pods then restarting kubelet.  


###### Is the rollout accompanied by any deprecations and/or removals of features, APIs, fields of API types, flags, etc.?

No


### Monitoring Requirements

Reference CPUID info in podresources API to be able to verify assignment.

###### How can an operator determine if the feature is in use by workloads?

Reference podresources API to determine CPU assignment and CacheID assignment per container.
Use proposed 'container_aligned_compute_resources_count' metric which reports the count of containers getting aligned compute resources. See PR#127155 (https://github.com/kubernetes/kubernetes/pull/127155).

###### How can someone using this feature know that it is working for their instance?

Reference podresources API to determine CPU assignment.

- [x] Other
  - Metric: container_aligned_compute_resource_count
  - Other field: CPUID from podresources API

###### What are the reasonable SLOs (Service Level Objectives) for the enhancement?

Measure the time to deploy pods under default settings and compare to the time to deploy pods with align-by-uncorecache enabled. Time difference should be negligible.

###### What are the SLIs (Service Level Indicators) an operator can use to determine the health of the service?

- Metrics
  - `topology_manager_admission_duration_ms`: Which measures the the duration of the admission process performed by Topology Manager.

###### Are there any missing metrics that would be useful to have to improve observability of this feature?

Utilized proposed 'container_aligned_compute_resources_count' in PR#127155 to be extended for uncore cache alignment count.

<!--
Describe the metrics themselves and the reasons why they weren't added (e.g., cost,
implementation difficulties, etc.).
-->

### Dependencies

<!--
This section must be completed when targeting beta to a release.
-->

###### Does this feature depend on any specific services running in the cluster?

No


### Scalability

<!--
For alpha, this section is encouraged: reviewers should consider these questions
and attempt to answer them.

For beta, this section is required: reviewers must answer these questions.

For GA, this section is required: approvers should be able to confirm the
previous answers based on experience in the field.
-->

###### Will enabling / using this feature result in any new API calls?

All of the housekeeping for this feature is node internal, and thus will not require the kubelet request anything new of the apiserver


###### Will enabling / using this feature result in introducing new API types?

No

###### Will enabling / using this feature result in any new calls to the cloud provider?

NA. 


###### Will enabling / using this feature result in increasing size or count of the existing API objects?

No


###### Will enabling / using this feature result in increasing time taken by any operations covered by existing SLIs/SLOs?

Pod startup time can directly affected because CPUManager will have to do a few extra steps when scheduling a Pod. This extra steps would be negligible as all the computation is done on RAM.


###### Will enabling / using this feature result in non-negligible increase of resource usage (CPU, RAM, disk, IO, ...) in any components?

No


###### Can enabling / using this feature result in resource exhaustion of some node resources (PIDs, sockets, inodes, etc.)?

No


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

There is no known consequence since this component is dedicated for CPU allocation for Pods which does not directly interact with any API server.

###### What are other known failure modes?

- Feature is best effort, resulting in potential for non-optimal uncore cache alignment when node is highly utilized.
  - Detection: Reference proposed metric in podresource API
  - Mitigation: Feature is preferred/best-effort
  - Diagnostics: Reference podresource API
  - Testing for failure mode not required as alignment is preferred and not a requirement
    
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

- The outlined sections were filled out was created 2024-08-27.

## Drawbacks

N/A


## Alternatives

Uncore cache affinity scheduling is possible by delgating CPU allocation from the Kubelet to the container runtime and plugins. However as a consequence of using a different implementation, the topology alignment granted by the Topology Manager within the kubelet is not compatible. 
Existing Static CPU Manager can be used, but requires manual assignment and for user to only run guaranteed pods with CPU sizes matching the corresponding uncore cache CPU group of the specific node.


## Infrastructure Needed (Optional)

To be able to do e2e testing it would be required that CI machines with CPUs with Split L3 Cache (UncoreCache) exist to be able to use this static policy option properly.
