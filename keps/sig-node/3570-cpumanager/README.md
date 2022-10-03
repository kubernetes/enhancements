# CPU Manager

<!-- toc -->
- [Release Signoff Checklist](#release-signoff-checklist)
- [Summary](#summary)
- [Motivation](#motivation)
  - [Goals](#goals)
  - [Non-Goals](#non-goals)
- [Proposal](#proposal)
  - [User Stories (Optional)](#user-stories-optional)
    - [Story 1 : High-performance applications](#story-1--high-performance-applications)
    - [Story 2 : KubeVirt](#story-2--kubevirt)
  - [Notes/Constraints/Caveats (Optional)](#notesconstraintscaveats-optional)
  - [Risks and Mitigations](#risks-and-mitigations)
- [Design Details](#design-details)
  - [Discovering CPU topology](#discovering-cpu-topology)
  - [CPU Manager interfaces (sketch)](#cpu-manager-interfaces-sketch)
  - [Configuring the CPU Manager](#configuring-the-cpu-manager)
    - [Policy 1: &quot;none&quot; cpuset control [default]](#policy-1-none-cpuset-control-default)
    - [Policy 2: &quot;static&quot; cpuset control](#policy-2-static-cpuset-control)
      - [Implementation sketch](#implementation-sketch)
      - [Example pod specs and interpretation](#example-pod-specs-and-interpretation)
      - [Example scenarios and interactions](#example-scenarios-and-interactions)
  - [Test Plan](#test-plan)
      - [Prerequisite testing updates](#prerequisite-testing-updates)
      - [Unit tests](#unit-tests)
      - [Integration tests](#integration-tests)
      - [e2e tests](#e2e-tests)
  - [Graduation Criteria](#graduation-criteria)
    - [Alpha](#alpha)
    - [Beta](#beta)
    - [GA](#ga)
    - [Deprecation](#deprecation)
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
  - [Proposed and not implemented items](#proposed-and-not-implemented-items)
    - [Policy 3: &quot;dynamic&quot; cpuset control](#policy-3-dynamic-cpuset-control)
      - [Implementation sketch](#implementation-sketch-1)
- [Infrastructure Needed (Optional)](#infrastructure-needed-optional)
- [Appendixes](#appendixes)
  - [related issues](#related-issues)
  - [Operations and observability](#operations-and-observability)
  - [Practical challenges](#practical-challenges)
  - [Original implementation roadmap](#original-implementation-roadmap)
    - [Phase 1: None policy [TARGET: Kubernetes v1.8]](#phase-1-none-policy-target-kubernetes-v18)
    - [Phase 2: Static policy [TARGET: Kubernetes v1.8]](#phase-2-static-policy-target-kubernetes-v18)
    - [Phase 3: Beta support [TARGET: Kubernetes v1.9]](#phase-3-beta-support-target-kubernetes-v19)
    - [Later phases [TARGET: After Kubernetes v1.9]](#later-phases-target-after-kubernetes-v19)
  - [cpuset pitfalls](#cpuset-pitfalls)
<!-- /toc -->

## Release Signoff Checklist

Items marked with (R) are required *prior to targeting to a milestone / release*.

- [X] (R) Enhancement issue in release milestone, which links to KEP dir in [kubernetes/enhancements] (not the initial KEP PR)
- [X] (R) KEP approvers have approved the KEP status as `implementable`
- [X] (R) Design details are appropriately documented
- [X] (R) Test plan is in place, giving consideration to SIG Architecture and SIG Testing input (including test refactors)
  - [X] e2e Tests for all Beta API Operations (endpoints)
  - [ ] (R) Ensure GA e2e tests for meet requirements for [Conformance Tests](https://github.com/kubernetes/community/blob/master/contributors/devel/sig-architecture/conformance-tests.md) 
  - [ ] (R) Minimum Two Week Window for GA e2e tests to prove flake free
- [ ] (R) Graduation criteria is in place
  - [ ] (R) [all GA Endpoints](https://github.com/kubernetes/community/pull/1806) must be hit by [Conformance Tests](https://github.com/kubernetes/community/blob/master/contributors/devel/sig-architecture/conformance-tests.md) 
- [ ] (R) Production readiness review completed
- [ ] (R) Production readiness review approved
- [ ] "Implementation History" section is up-to-date for milestone
- [ ] User-facing documentation has been created in [kubernetes/website], for publication to [kubernetes.io]
- [ ] Supporting documentation—e.g., additional design documents, links to mailing list discussions/SIG meetings, relevant PRs/issues, release notes

<!--
**Note:** This checklist is iterative and should be reviewed and updated every time this enhancement is being considered for a milestone.
-->

[kubernetes.io]: https://kubernetes.io/
[kubernetes/enhancements]: https://git.k8s.io/enhancements
[kubernetes/kubernetes]: https://git.k8s.io/kubernetes
[kubernetes/website]: https://git.k8s.io/website

## Summary

The *CPU Manager* is a new software component in Kubelet responsible for
assigning pod containers to sets of CPUs on the local node. In later
phases, the scope will expand to include caches, a critical shared
processor resource.

The kuberuntime notifies the CPU manager when containers come and
go. The first such notification occurs in between the container runtime
interface calls to create and start the container. The second notification
occurs after the container is stopped by the container runtime. The CPU
Manager writes CPU settings for containers using a new CRI method named
[`UpdateContainerResources`](https://github.com/kubernetes/kubernetes/pull/46105).
This new method is invoked from two places in the CPU manager: during each
call to `AddContainer` and also periodically from a separate
reconciliation loop.

This KEP supersedes and replaces `kubernetes/enhancements/keps/sig-node/375-cpumanager/README.md`.

## Motivation

1. Poor or unpredictable performance observed compared to virtual machine
   based orchestration systems. Application latency and lower CPU
   throughput compared to VMs due to cpu quota being fulfilled across all
   cores, rather than exclusive cores, which results in fewer context
   switches and higher cache affinity.
2. Unacceptable latency attributed to the OS process scheduler, especially
   for “fast” virtual network functions (want to approach line rate on
   modern server NICs.)

### Goals

1. Provide an API-driven contract from the system to a user: "if you are a
   Guaranteed pod with 1 or more cores of cpu, the system will try to make
   sure that the pod gets its cpu quota primarily from reserved core(s),
   resulting in fewer context switches and higher cache affinity".
2. Support the case where in a given pod, one container is latency-critical
   and another is not (e.g. auxiliary side-car containers responsible for
   log forwarding, metrics collection and the like.)
3. Do not cap CPU quota for guaranteed containers that are granted
   exclusive cores, since that would be antithetical to (1) above.
4. Take physical processor topology into account in the CPU affinity policy.

### Non-Goals

N/A

## Proposal

![cpu-manager-block-diagram](https://user-images.githubusercontent.com/379372/30137651-2352f4f0-9319-11e7-8be7-0aaeb6ce593a.png)

_CPU Manager block diagram. `Policy`, `State`, and `Topology` types are
factored out of the CPU Manager to promote reuse and to make it easier
to build and test new policies. The shared state abstraction allows
other Kubelet components to be agnostic of the CPU manager policy for
observability and checkpointing extensions._

### User Stories (Optional)

#### Story 1 : High-performance applications

Systems such as real-time trading system or 5G CNFs (User Plane Function, UPF) need to maximize the CPU time; CPU pinning ensure exclusive CPU allocation and allows to avoid performance issues due to core switches, cold caches.
NUMA aware allocation of CPUs, provided by CPU manager cooperating with Topology Manager, is also a critical prerequisite for these applications to meet their performance requirement.
The alignment of resources on the same NUMA node, CPUs first and foremost, prevents performance degradation due to inter-node (between NUMA nodes) communication overhead.

#### Story 2 : KubeVirt

KubeVirt leverages the CPU pinning provided by CPU manager to assign full CPU cores to vCPUs inside the VM to [enhance performance][kubevirt-cpus].
[NUMA support for VMs][kubevirt-numa] is also built on top of the CPU pinning and NUMA-aware CPU allocation.

### Notes/Constraints/Caveats (Optional)

N/A

### Risks and Mitigations

TBD

## Design Details

### Discovering CPU topology

The CPU Manager must understand basic topology. First of all, it must
determine the number of logical CPUs (hardware threads) available for
allocation. On architectures that support [hyper-threading][ht], sibling
threads share a number of hardware resources including the cache
hierarchy. On multi-socket systems, logical CPUs co-resident on a socket
share L3 cache. Although there may be some programs that benefit from
disjoint caches, the policies described in this proposal assume cache
affinity will yield better application and overall system performance for
most cases. In all scenarios described below, we prefer to acquire logical
CPUs topologically. For example, allocating two CPUs on a system that has
hyper-threading turned on yields both sibling threads on the same
physical core. Likewise, allocating two CPUs on a non-hyper-threaded
system yields two cores on the same socket.

**Decision:** Initially the CPU Manager will re-use the existing discovery
mechanism in cAdvisor.

Alternate options considered for discovering topology:

1. Read and parse the virtual file [`/proc/cpuinfo`][procfs] and construct a
   convenient data structure.
1. Execute a simple program like `lscpu -p` in a subprocess and construct a
   convenient data structure based on the output. Here is an example of
   [data structure to represent CPU topology][topo] in go. The linked package
   contains code to build a ThreadSet from the output of `lscpu -p`.
1. Execute a mature external topology program like [`mpi-hwloc`][hwloc] --
   potentially adding support for the hwloc file format to the Kubelet.

### CPU Manager interfaces (sketch)

```go
type State interface {
  GetCPUSet(containerID string) (cpuset.CPUSet, bool)
  GetDefaultCPUSet() cpuset.CPUSet
  GetCPUSetOrDefault(containerID string) cpuset.CPUSet
  SetCPUSet(containerID string, cpuset CPUSet)
  SetDefaultCPUSet(cpuset CPUSet)
  Delete(containerID string)
}

type Manager interface {
  Start(ActivePodsFunc, status.PodStatusProvider, runtimeService)
  AddContainer(p *Pod, c *Container, containerID string) error
  RemoveContainer(containerID string) error
  State() state.Reader
}

type Policy interface {
  Name() string
  Start(s state.State)
  AddContainer(s State, pod *Pod, container *Container, containerID string) error
  RemoveContainer(s State, containerID string) error
}

type CPUSet map[int]struct{} // set operations and parsing/formatting helpers

type CPUTopology // convenient type for querying and filtering CPUs
```

### Configuring the CPU Manager

Kubernetes will ship with CPU manager policies. Only one policy is
active at a time on a given node, chosen by the operator via Kubelet
configuration. The policies are **none** and **static**.


The active CPU manager policy is set through a new Kubelet
configuration value `--cpu-manager-policy`. The default value is `none`.

The CPU manager periodically writes resource updates through the CRI in
order to reconcile in-memory cpuset assignments with cgroupfs. The
reconcile frequency is set through a new Kubelet configuration value
`--cpu-manager-reconcile-period`. If not specified, it defaults to the
same duration as `--node-status-update-frequency` (which itself defaults
to 10 seconds at time of writing.)

Each policy is described below.

#### Policy 1: "none" cpuset control [default]

This policy preserves the existing Kubelet behavior of doing nothing
with the cgroup `cpuset.cpus` and `cpuset.mems` controls. This "none"
policy would become the default CPU Manager policy until the effects of
the other policies are better understood.

#### Policy 2: "static" cpuset control

The "static" policy allocates exclusive CPUs for containers if they are
included in a pod of "Guaranteed" [QoS class][qos] and the container's
resource limit for the CPU resource is an integer greater than or
equal to one. All other containers share a set of CPUs.

When exclusive CPUs are allocated for a container, those CPUs are
removed from the allowed CPUs of every other container running on the
node. Once allocated at pod admission time, an exclusive CPU remains
assigned to a single container for the lifetime of the pod (until it
becomes terminal.)

The Kubelet requires the total CPU reservation from `--kube-reserved`
and `--system-reserved` to be greater than zero when the static policy is
enabled. This is because zero CPU reservation would allow the shared pool to
become empty. The set of reserved CPUs is taken in order of ascending
physical core ID. Operator documentation will be updated to explain how to
configure the system to use the low-numbered physical cores for kube-reserved
and system-reserved cgroups.

Workloads that need to know their own CPU mask, e.g. for managing
thread-level affinity, can read it from the virtual file `/proc/self/status`:

```
$ grep -i cpus /proc/self/status
Cpus_allowed:   77
Cpus_allowed_list:      0-2,4-6
```

Note that containers running in the shared cpuset should not attempt any
application-level CPU affinity of their own, as those settings may be
overwritten without notice (whenever exclusive cores are
allocated or deallocated.)

##### Implementation sketch

The static policy maintains the following sets of logical CPUs:

- **SHARED:** Burstable, BestEffort, and non-integral Guaranteed containers
  run here. Initially this contains all CPU IDs on the system. As
  exclusive allocations are created and destroyed, this CPU set shrinks
  and grows, accordingly. This is stored in the state as the default
  CPU set.

- **RESERVED:** A subset of the shared pool which is not exclusively
  allocatable. The membership of this pool is static for the lifetime of
  the Kubelet. The size of the reserved pool is the ceiling of the total
  CPU reservation from `--kube-reserved` and `--system-reserved`.
  Reserved CPUs are taken topologically starting with lowest-indexed
  physical core, as reported by cAdvisor.

- **ASSIGNABLE:** Equal to `SHARED - RESERVED`. Exclusive CPUs are allocated
  from this pool.

- **EXCLUSIVE ALLOCATIONS:** CPU sets assigned exclusively to one container.
  These are stored as explicit assignments in the state.

When an exclusive allocation is made, the static policy also updates the
default cpuset in the state abstraction. The CPU manager's periodic
reconcile loop takes care of updating the cpuset in cgroupfs for any
containers that may be running in the shared pool. For this reason,
applications running within exclusively-allocated containers must tolerate
potentially sharing their allocated CPUs for up to the CPU manager
reconcile period.

```go
func (p *staticPolicy) Start(s State) {
	fullCpuset := cpuset.NewCPUSet()
	for cpuid := 0; cpuid < p.topology.NumCPUs; cpuid++ {
		fullCpuset.Add(cpuid)
	}
	// Figure out which cores shall not be used in shared pool
	reserved, _ := takeByTopology(p.topology, fullCpuset, p.topology.NumReservedCores)
	s.SetDefaultCPUSet(fullCpuset.Difference(reserved))
}

func (p *staticPolicy) AddContainer(s State, pod *Pod, container *Container, containerID string) error {
  if numCPUs := numGuaranteedCPUs(pod, container); numCPUs != 0 {
    // container should get some exclusively allocated CPUs
    cpuset, err := p.allocateCPUs(s, numCPUs)
    if err != nil {
      return err
    }
    s.SetCPUSet(containerID, cpuset)
  }
  // container belongs in the shared pool (nothing to do; use default cpuset)
  return nil
}

func (p *staticPolicy) RemoveContainer(s State, containerID string) error {
  if toRelease, ok := s.GetCPUSet(containerID); ok {
    s.Delete(containerID)
    s.SetDefaultCPUSet(s.GetDefaultCPUSet().Union(toRelease))
  }
  return nil
}
```

##### Example pod specs and interpretation

| Pod                                        | Interpretation                 |
| ------------------------------------------ | ------------------------------ |
| Pod [Guaranteed]:<br />&emsp;A:<br />&emsp;&emsp;cpu: 0.5 | Container **A** is assigned to the shared cpuset. |
| Pod [Guaranteed]:<br />&emsp;A:<br />&emsp;&emsp;cpu: 2.0 | Container **A** is assigned two sibling threads on the same physical core (HT) or two physical cores on the same socket (no HT.)<br /><br /> The shared cpuset is shrunk to  make room for the exclusively allocated CPUs. |
| Pod [Guaranteed]:<br />&emsp;A:<br />&emsp;&emsp;cpu: 1.0<br />&emsp;B:<br />&emsp;&emsp;cpu: 0.5 | Container **A** is assigned one exclusive CPU and container **B** is assigned to the shared cpuset. |
| Pod [Guaranteed]:<br />&emsp;A:<br />&emsp;&emsp;cpu: 1.5<br />&emsp;B:<br />&emsp;&emsp;cpu: 0.5 | Both containers **A** and **B** are assigned to the shared cpuset. |
| Pod [Burstable] | All containers are assigned to the shared cpuset. |
| Pod [BestEffort] | All containers are assigned to the shared cpuset. |

##### Example scenarios and interactions

1. _A container arrives that requires exclusive cores._
    1. Kuberuntime calls the CRI delegate to create the container.
    1. Kuberuntime adds the container with the CPU manager.
    1. CPU manager adds the container to the static policy.
    1. Static policy acquires CPUs from the default pool, by
       topological-best-fit.
    1. Static policy updates the state, adding an assignment for the new
       container and removing those CPUs from the default pool.
    1. CPU manager reads container assignment from the state.
    1. CPU manager updates the container resources via the CRI.
    1. Kuberuntime calls the CRI delegate to start the container.

1. _A container that was assigned exclusive cores terminates._
    1. Kuberuntime removes the container with the CPU manager.
    1. CPU manager removes the container with the static policy.
    1. Static policy adds the container's assigned CPUs back to the default
       pool.
    1. Kuberuntime calls the CRI delegate to remove the container.
    1. Asynchronously, the CPU manager's reconcile loop updates the
       cpuset for all containers running in the shared pool.

1. _The shared pool becomes empty._
    1. This cannot happen. The size of the shared pool is greater than
       the number of exclusively allocatable CPUs. The Kubelet requires the
       total CPU reservation from `--kube-reserved` and `--system-reserved`
       to be greater than zero when the static policy is enabled. The number
       of exclusively allocatable CPUs is
       `floor(capacity.cpu - allocatable.cpu)` and the shared pool initially
       contains all CPUs in the system.


### Test Plan

[X] I/we understand the owners of the involved components may require updates to
existing tests to make this code solid enough prior to committing the changes necessary
to implement this enhancement.

##### Prerequisite testing updates

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

- `k8s.io/kubernetes/pkg/kubelet/cm/cpumanager`: `20220929` - `86.2%`

##### Integration tests

- TBD

##### e2e tests

- TBD

### Graduation Criteria

#### Alpha

- Feature implemented behind a feature flag
- Initial e2e tests completed and enabled

#### Beta

- Gather feedback from developers and surveys
- Complete features A, B, C
- Additional tests are in Testgrid and linked in KEP

#### GA

- N examples of real-world usage
- N installs
- More rigorous forms of testing—e.g., downgrade tests and scalability tests
- Allowing time for feedback

**Note:** Generally we also wait at least two releases between beta and
GA/stable, because there's no opportunity for user feedback, or even bug reports,
in back-to-back releases.

**For non-optional features moving to GA, the graduation criteria must include
[conformance tests].**

[conformance tests]: https://git.k8s.io/community/contributors/devel/sig-architecture/conformance-tests.md

#### Deprecation

- Announce deprecation and support policy of the existing flag
- Two versions passed since introducing the functionality that deprecates the flag (to address version skew)
- Address feedback on usage/changed behavior, provided on GitHub issues
- Deprecate the flag
-->

### Upgrade / Downgrade Strategy

No impact. It's always possible to trivially downgrade to the previous kubelet

### Version Skew Strategy

Not relevant

## Production Readiness Review Questionnaire

### Feature Enablement and Rollback

###### How can this feature be enabled / disabled in a live cluster?

- [ ] Feature gate (also fill in values in `kep.yaml`)
  - Feature gate name:
  - Components depending on the feature gate:
- [ ] Other
  - Describe the mechanism:
  - Will enabling / disabling the feature require downtime of the control
    plane?
  - Will enabling / disabling the feature require downtime or reprovisioning
    of a node? (Do not assume `Dynamic Kubelet Config` feature is enabled).

###### Does enabling the feature change any default behavior?

No, unless the non-none policy is explicitly configured.

###### Can the feature be disabled once it has been enabled (i.e. can we roll back the enablement)?

Yes, using the kubelet config.

###### What happens if we reenable the feature if it was previously rolled back?

The impact is node-local only.
If the state of a node is steady, no changes.
If a guaranteed pod is admitted, running non-guaranteed pods will have their CPU cgroup changed while running.

###### Are there any tests for feature enablement/disablement?

Yes, covered by e2e tests

### Rollout, Upgrade and Rollback Planning

###### How can a rollout or rollback fail? Can it impact already running workloads?

A rollout can fail if a bug in the cpumanager prevents _new_ pods to start, or existing pods to be restarted.
Already running workload will not be affected if the node state is steady

###### What specific metrics should inform a rollback?

Pod creation errors o a node-by-node basis.

###### Were upgrade and rollback tested? Was the upgrade->downgrade->upgrade path tested?

No to both.
Changes in behavior only affects pods meeting the conditions (guaranteed QoS, integral CPU request) scheduler after the upgrade.
Running pods will be unaffected by any change. This offers some degree of safety in both upgrade->rollback
and upgrade->downgrade->upgrade scenarios.

###### Is the rollout accompanied by any deprecations and/or removals of features, APIs, fields of API types, flags, etc.?

No

### Monitoring Requirements

Monitor the pod admission counter
Monitor the pods not going running after successful schedule

###### How can an operator determine if the feature is in use by workloads?

The operator need to inspect the node and verify the cpu pinning assignment either checking the cgroups on the node
or accessing the podresources API of the kubelet.

###### How can someone using this feature know that it is working for their instance?


- [X] Other (treat as last resort)
  - Details: the containers need to check the cpu set they are allowed to run; in addition, node agents (e.g. node_exporter)
    can report the CPU assignment

###### What are the reasonable SLOs (Service Level Objectives) for the enhancement?

- N/A

###### What are the SLIs (Service Level Indicators) an operator can use to determine the health of the service?

- [ ] Other (treat as last resort)
  - Details:
     a operator should check that pods go running correctly and the cpu pinning is performed. The latter can
     be checked by inspecting the cgroups at node level.

###### Are there any missing metrics that would be useful to have to improve observability of this feature?

No, because all the metrics we were aware of leaked hardware details.
All of the metrics experimented by consumers of the feature so far require to expose hardware details of the
worker nodes, and are dependent on the worker node hardware configuration (e.g. processor core layout).

### Dependencies

None

###### Does this feature depend on any specific services running in the cluster?

No

### Scalability

###### Will enabling / using this feature result in any new API calls?

No, the feature is entirely node-local

###### Will enabling / using this feature result in introducing new API types?

No, the feature is entirely node-local

###### Will enabling / using this feature result in any new calls to the cloud provider?

No, the feature is entirely node-local

###### Will enabling / using this feature result in increasing size or count of the existing API objects?

No, the feature is entirely node-local

###### Will enabling / using this feature result in increasing time taken by any operations covered by existing SLIs/SLOs?

No, the feature is entirely node-local

###### Will enabling / using this feature result in non-negligible increase of resource usage (CPU, RAM, disk, IO, ...) in any components?

No

### Troubleshooting

###### How does this feature react if the API server and/or etcd is unavailable?

No

###### What are other known failure modes?

After changing the CPU manager policy from `none` to `static` or the the other way around, before to start the kubelet again,
you must remove the CPU manager state file(`/var/lib/kubelet/cpu_manager_state`), otherwise the kubelet start will fail.
Startup failures for this reason will be logged in the kubelet log.

###### What steps should be taken if SLOs are not being met to determine the problem?

## Implementation History

- **2022-09-29:** kep translated to the most recent template available at time; proposed to GA; added PRR info.

## Drawbacks

N/A

## Alternatives

### Proposed and not implemented items

#### Policy 3: "dynamic" cpuset control

_TODO: Describe the policy._

Capturing discussions from resource management meetings and proposal comments:

Unlike the static policy, when the dynamic policy allocates exclusive CPUs to
a container, the cpuset may change during the container's lifetime. If deemed
necessary, we discussed providing a signal in the following way. We could
project (a subset of) the CPU manager state into a volume visible to selected
containers. User workloads could subscribe to update events in a normal Linux
manner (e.g. inotify.)


##### Implementation sketch

```go
func (p *dynamicPolicy) Start(s State) {
	// TODO
}

func (p *dynamicPolicy) AddContainer(s State, pod *Pod, container *Container, containerID string) error {
	// TODO
}

func (p *dynamicPolicy) RemoveContainer(s State, containerID string) error {
	// TODO
}
```

## Infrastructure Needed (Optional)

N/A

## Appendixes

Record of information of the original KEP without a clear fit in the latest template

### related issues

* feature: [further differentiate performance characteristics associated
  with pod level qos](https://github.com/kubernetes/features/issues/276)
* feature: [add cpu manager for pod cpuset
  assignment](https://github.com/kubernetes/features/issues/375)

### Operations and observability

* Checkpointing assignments
  * The CPU Manager must be able to pick up where it left off in case the
    Kubelet restarts for any reason.
* Read effective CPU assignments at runtime for alerting. This could be
  satisfied by the checkpointing requirement.

### Practical challenges

1. Synchronizing CPU Manager state with the container runtime via the
   CRI. Runc/libcontainer allows container cgroup settings to be updated
   after creation, but neither the Kubelet docker shim nor the CRI
   implement a similar interface.
    1. Mitigation: [PR 46105](https://github.com/kubernetes/kubernetes/pull/46105)
1. Compatibility with the `isolcpus` Linux kernel boot parameter. The operator
   may want to correlate exclusive cores with the isolated CPUs, in which
   case the static policy outlined above, where allocations are taken
   directly from the shared pool, is too simplistic.
    1. Mitigation: defer supporting this until a new policy tailored for
       use with `isolcpus` can be added.

### Original implementation roadmap

#### Phase 1: None policy [TARGET: Kubernetes v1.8]

* Internal API exists to allocate CPUs to containers
  ([PR 46105](https://github.com/kubernetes/kubernetes/pull/46105))
* Kubelet configuration includes a CPU manager policy (initially only none)
* None policy is implemented.
* All existing unit and e2e tests pass.
* Initial unit tests pass.

#### Phase 2: Static policy [TARGET: Kubernetes v1.8]

* Kubelet can discover "basic" CPU topology (HT-to-physical-core map)
* Static policy is implemented.
* Unit tests for static policy pass.
* e2e tests for static policy pass.
* Performance metrics for one or more plausible synthetic workloads show
  benefit over none policy.

#### Phase 3: Beta support [TARGET: Kubernetes v1.9]

* Container CPU assignments are durable across Kubelet restarts.
* Expanded user and operator docs and tutorials.

#### Later phases [TARGET: After Kubernetes v1.9]

* Static policy also manages [cache allocation][cat] on supported platforms.
* Dynamic policy is implemented.
* Unit tests for dynamic policy pass.
* e2e tests for dynamic policy pass.
* Performance metrics for one or more plausible synthetic workloads show
  benefit over none policy.
* Kubelet can discover "advanced" topology (NUMA).
* Node-level coordination for NUMA-dependent resource allocations, for example
  devices, CPUs, memory-backed volumes including hugepages.

### cpuset pitfalls

1. [`cpuset.sched_relax_domain_level`][cpuset-files]. "controls the width of
   the range of CPUs over  which  the kernel scheduler performs immediate
   rebalancing of runnable tasks across CPUs."
1. Child cpusets must be subsets of their parents. If B is a child of A,
   then B must be a subset of A. Attempting to shrink A such that B
   would contain allowed CPUs not in A is not allowed (the write will
   fail.) Nested cpusets must be shrunk bottom-up. By the same rationale,
   nested cpusets must be expanded top-down.
1. Dynamically changing cpusets by directly writing to the sysfs would
   create inconsistencies with container runtimes.
1. The `exclusive` flag. This will not be used. We will achieve
   exclusivity for a CPU by removing it from all other assigned cpusets.
1. Tricky semantics when cpusets are combined with CFS shares and quota.

[cat]: http://www.intel.com/content/www/us/en/communications/cache-monitoring-cache-allocation-technologies.html
[cpuset-files]: http://man7.org/linux/man-pages/man7/cpuset.7.html#FILES
[kubevirt-cpus]: https://kubevirt.io/user-guide/virtual_machines/dedicated_cpu_resources/
[kubevirt-numa]: https://kubevirt.io/user-guide/virtual_machines/numa/#preconditions
[ht]: http://www.intel.com/content/www/us/en/architecture-and-technology/hyper-threading/hyper-threading-technology.html
[hwloc]: https://www.open-mpi.org/projects/hwloc
[node-allocatable]: /contributors/design-proposals/node/node-allocatable.md#phase-2---enforce-allocatable-on-pods
[procfs]: http://man7.org/linux/man-pages/man5/proc.5.html
[qos]: /contributors/design-proposals/node/resource-qos.md
[topo]: http://github.com/intelsdi-x/swan/tree/master/pkg/isolation/topo
