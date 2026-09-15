# KEP-4885: Windows CPU and Memory Affinity

<!-- toc -->
- [Release Signoff Checklist](#release-signoff-checklist)
- [Summary](#summary)
- [Motivation](#motivation)
  - [Goals](#goals)
  - [Non-Goals](#non-goals)
- [Proposal](#proposal)
  - [User Stories (Optional)](#user-stories-optional)
  - [Notes/Constraints/Caveats (Optional)](#notesconstraintscaveats-optional)
  - [Risks and Mitigations](#risks-and-mitigations)
- [Design Details](#design-details)
  - [Windows CPU Discovery](#windows-cpu-discovery)
  - [Windows Memory considerations](#windows-memory-considerations)
    - [Kubelet memory management](#kubelet-memory-management)
  - [Windows Topology manager considerations](#windows-topology-manager-considerations)
  - [Test Plan](#test-plan)
      - [Prerequisite testing updates](#prerequisite-testing-updates)
      - [Unit tests](#unit-tests)
      - [Integration tests](#integration-tests)
      - [Node e2e tests](#node-e2e-tests)
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
- [Infrastructure Needed (Optional)](#infrastructure-needed-optional)
<!-- /toc -->

## Release Signoff Checklist

Items marked with (R) are required *prior to targeting to a milestone / release*.

- [x] (R) Enhancement issue in release milestone, which links to KEP dir in [kubernetes/enhancements] (not the initial KEP PR)
- [ ] (R) KEP approvers have approved the KEP status as `implementable`
- [x] (R) Design details are appropriately documented
- [x] (R) Test plan is in place, giving consideration to SIG Architecture and SIG Testing input (including test refactors)
  - [x] e2e Tests for all Beta API Operations (endpoints) (N/A: this KEP introduces no Kubernetes API endpoints.)
  - [x] (R) Ensure GA e2e tests meet requirements for [Conformance Tests](https://github.com/kubernetes/community/blob/master/contributors/devel/sig-architecture/conformance-tests.md) (N/A: this KEP introduces no Kubernetes API endpoints.)
  - [x] (R) Minimum Two Week Window for [GA e2e tests](https://testgrid.k8s.io/sig-windows-signal#windows-e2e-node-master) to prove flake free
- [x] (R) Graduation criteria is in place
  - [x] (R) [all GA Endpoints](https://github.com/kubernetes/community/pull/1806) must be hit by [Conformance Tests](https://github.com/kubernetes/community/blob/master/contributors/devel/sig-architecture/conformance-tests.md) within one minor version of promotion to GA (N/A: this KEP introduces no Kubernetes API endpoints.)
- [ ] (R) Production readiness review completed
- [ ] (R) Production readiness review approved
- [ ] "Implementation History" section is up-to-date for milestone
- [x] User-facing documentation has been created in [kubernetes/website], for publication to [kubernetes.io]
- [ ] Supporting documentation—e.g., additional design documents, links to mailing list discussions/SIG meetings, relevant PRs/issues, release notes

[kubernetes.io]: https://kubernetes.io/
[kubernetes/enhancements]: https://git.k8s.io/enhancements
[kubernetes/kubernetes]: https://git.k8s.io/kubernetes
[kubernetes/website]: https://git.k8s.io/website

## Summary

This kep outlines how to add support for the CPU, Memory and Topology Managers in kubelet for Windows.  
The Managers are already available and support in kubelet on Linux and there have been requests to sig-windows
to add support on Windows to help with workloads that require co-location.  The goal of the KEP is to
add Windows support without significant changes to the Managers logic while providing the same feature sets available
on Linux today.

The existing KEPS are:

https://github.com/kubernetes/enhancements/tree/master/keps/sig-node/3570-cpumanager
https://github.com/kubernetes/enhancements/tree/master/keps/sig-node/1769-memory-manager
https://github.com/kubernetes/enhancements/tree/master/keps/sig-node/693-topology-manager

## Motivation

Currently enabling low latency workloads co-hosted on the same nodes in Windows Server create noisy neighbor behaviors 
preventing them from achieving their expected performance goals. 
The CPU, Memory and Topology Managers feature is needed to add the necessary isolation to accomplish both high performance and co-hosting efficiency.  
The feature is enabled and available in Linux and Windows users are asking for the same features on Windows.

### Goals

- Enable CPU manager for Windows allowing for CPU affinity for configured pods
- Enable Memory Manager for Windows allowing for memory affinity for configured pods
- Enable Topology Manager for Windows allowing for coordination of Memory and CPU affinity at the node level for scheduled pods

### Non-Goals

- We do not wish to create new managers and instead re-use the existing logic provided 
- Modify or bypass any existing feature gated features.  Existing Policy features gates will still be used to progress specific policies related to the managers.

## Proposal

The proposal requires very little changes to the code for the managers and instead extends the [Windows](https://learn.microsoft.com/en-us/windows/win32/procthread/processor-groups) concepts to a CAdvisor mapping to enable the [topology structure in kubelet](https://github.com/kubernetes/kubernetes/blob/cede96336a809a67546ca08df0748e4253ec270d/pkg/kubelet/cm/cpumanager/topology/topology.go#L34-L39).

There are no plans to change the core logic for selecting CPU's and NUMA nodes in the CPU/Memory/Tolopology managers from the existing KEPS ([memory-manager](keps/sig-node/1769-memory-manager)/[cpu-manager](keps/sig-node/3570-cpu-manager)/[topology-manager](keps/sig-node/693-topology-manager")).  The logic is currently in platform agnostic
structures so the selection process does not require changes for adoption on Windows.  The Windows specific considerations for each of the managers will be covered in separate sections in this document.


### User Stories (Optional)

The User stories on Windows are similar to Linux:

https://github.com/kubernetes/enhancements/tree/master/keps/sig-node/3570-cpumanager#user-stories-optional
https://github.com/kubernetes/enhancements/tree/master/keps/sig-node/1769-memory-manager#user-stories
https://github.com/kubernetes/enhancements/tree/master/keps/sig-node/693-topology-manager#user-stories-optional

### Notes/Constraints/Caveats (Optional)

Windows does not have an API to constrain workloads to a specific NUMA node.  This is addressed in the Memory Manager section below.

### Risks and Mitigations


The technical risks are the same from existing KEP's: 
 - https://github.com/kubernetes/enhancements/tree/master/keps/sig-node/3570-cpumanager#risks-and-mitigations
 - https://github.com/kubernetes/enhancements/tree/master/keps/sig-node/1769-memory-manager#risks-and-mitigations
 - https://github.com/kubernetes/enhancements/tree/master/keps/sig-node/693-topology-manager#risks-and-mitigations

For sig-windows, we also see a risk to enabling a feature that has already Stable or fully featured on Linux.  To mitigate this risk we have opted to create a 
separate KEP with a feature flag so we can communicate our status effectively. 

## Design Details

### Windows CPU Discovery

The Windows Kubelet provides an implementation for the [cadvisor api](https://github.com/kubernetes/kubernetes/blob/fbaf9b0353a61c146632ac195dfeb1fbaffcca1e/pkg/kubelet/cadvisor/cadvisor_windows.go#L50) 
in order to provide Windows stats to other components without modification.  
The ability to provide the `cadvisorapi.MachineInfo` api is already partially mapped
in on the Windows client.  By mapping the Windows specific topology API's to 
cadvisor API, no changes are required to the CPU Manager.

The [Windows concepts](https://learn.microsoft.com/windows/win32/procthread/processor-groups) are mapped to [Linux concepts](https://github.com/kubernetes/kubernetes/blob/cede96336a809a67546ca08df0748e4253ec270d/pkg/kubelet/cm/cpumanager/topology/topology.go#L34-L39) with the following:

| Kubelet Term | Description | Cadvisor term | Windows term |
| --- | --- | --- | --- |
| CPU | logical CPU | thread | Logical processor |
| Core | physical CPU | Core | Core |
| Socket | socket | Socket | Physical Processor |
| NUMA Node | NUMA cell | Node | Numa node |

The result of this mapping  gives the following output from CPU manager after the conversion into kubelet's memory structure:

```json
"Detected CPU topology" 
topology={"NumCPUs":8,"NumCores":4,"NumSockets":1,"NumNUMANodes":1,"CPUDetails":{
"0":{"NUMANodeID":0,"SocketID":1,"CoreID":0},
"1":{"NUMANodeID":0,"SocketID":1,"CoreID":0},
"2":{"NUMANodeID":0,"SocketID":1,"CoreID":2},
"3":{"NUMANodeID":0,"SocketID":1,"CoreID":2},
"4":{"NUMANodeID":0,"SocketID":1,"CoreID":4},
"5":{"NUMANodeID":0,"SocketID":1,"CoreID":4},
"6":{"NUMANodeID":0,"SocketID":1,"CoreID":6},
"7":{"NUMANodeID":0,"SocketID":1,"CoreID":6}}}
```

The Windows API's used will be
-	[getlogicalprocessorinformationex](https://learn.microsoft.com/windows/win32/api/sysinfoapi/nf-sysinfoapi-getlogicalprocessorinformationex)
-	[nf-winbase-getnumaavailablememorynodeex](https://learn.microsoft.com/windows/win32/api/winbase/nf-winbase-getnumaavailablememorynodeex) 

One difference between the Windows API and Linux is the concept of [Processor groups](https://learn.microsoft.com/windows/win32/procthread/processor-groups).
On Windows systems with more than 64 cores the CPU's will be split into groups, 
each processor is identified by its group number and its group-relative processor number. 

In CRI we will add the following structure to the `WindowsContainerResources` in CRI:

```protobuf
message WindowsCpuGroupAffinity {
    // CPU mask relative to this CPU group.
    uint64 cpu_mask = 1;
    // CPU group that this CPU belongs to.
    uint32 cpu_group = 2;
}
```

Since the Kubelet API's are looking for a distinct ProcessorId, the processorid's will be calculated by looping 
through the mask and calculating the ids with `(group *64) + procesorid` resulting in unique processor id's from `group 0` as `0-63` and 
processor Id's from `group 1` as `64-127` and so on. This translation will be done only in kubelet, the `cpu_mask` will be used when 
communicating with the container runtime.

```golang
for i := 0; i < 64; i++ {
		if groupaffinity.Mask&(1<<i) != 0 {
			processors = append(processors, i+(int(a.Group)*64))
		}
	}
}
```

Using this logic, a cpu bit mask of `0000111` (leading zero's removed) would result in cpu's: 

- `0,1,2` in `group 0` 
- `64,65,66` in `group 1`.

When converting back to the Windows Group Affinity we will divide the cpu number by 64 to get the group number then 
use mod of 64 to calculate the location of the cpu in mask:

```golang
group := cpu / 64
mask := 1 << (cpu % 64)
groupaffinity.Mask |= mask
```

There are some scenarios where cpu count might be greater than 64 cores but in each group it is less
than 64. For instance, you could have 2 CPU groups with 35 processors each.  The unique ID's using the strategy 
above would give you: 

- CPU group 0 : 0 to 34
- CPU group 2: 64 to 99

### Windows Memory considerations

[Numa nodes](https://learn.microsoft.com/en-us/windows/win32/procthread/numa-support) can not be directly assigned or guaranteed via the Windows API but the windows sub system attempts to use memory assigned to the CPU to improve performance.  
It is possible to indicate to a process which Numa node is preferred but a limitation of the Windows API's is that [PROC_THREAD_ATTRIBUTE_PREFERRED_NODE](https://learn.microsoft.com/windows/win32/api/processthreadsapi/nf-processthreadsapi-updateprocthreadattribute)
does not support setting multiple Numa nodes for a single Job object (i.e. Container) so is not usable in the context of Windows containers which have multiple processes.  

Since the existing Memory Manager Policy `Static` on Linux has semantic meaning that ensures that only the memory from a 
NUMA node selected is used. We can not re-use this policy on Windows given that there is no way to ensure only the memory 
on the Node that the memory manager selects.  For these reason if the `Static` policy is chosen on Windows kubelet will fail
to start with an error message that states it can use the `Static` policy.  Instead we will create a new Windows only Policy called
`BestEffort` which will initially only be implemented on Windows and Linux will fail to start if the Policy is set.  
We do not have any use cases for this policy to be implemented on Linux at this time and so we will avoid adding a feature that isn't 
applicable to that platform.

The main purpose of the `BestEffort` policy on Windows will be to ensure that at the time of pod start up there is enough Memory on a given NUMA node to meet the memory requests of the pod.  The intent here is to make sure if CPU's are selected that there is enough memory to also support the request to avoid cross CPU/NUMA node processing. On Windows, even though we cannot guarantee NUMA node selection, the Windows Schedule will do the right thing in most cases.  By using Kubelet's existing [Memory Mapping strategy](https://github.com/kubernetes/enhancements/tree/master/keps/sig-node/1769-memory-manager#memory-map) we can ensure NUMA nodes have enough memory at the time of scheduling.  It is important to note that this does not mean that it is guaranteed (hence the policy name change)

Since Windows does not have an API to directly assign NUMA nodes, the kubelet uses CPU Group affinity in the CRI field to influence memory locality. The final CPU affinity depends on whether CPU Manager has allocated exclusive CPUs:

- Memory Manager is enabled and CPU Manager has not allocated exclusive CPUs: kubelet looks up all CPUs associated with the NUMA nodes selected by Memory Manager and assigns them to the CPU Group affinity. For example, if Memory Manager selects NUMA node 0 and its first four CPUs are in Windows CPU group 0, the result is `cpu affinity: 0000001111, group 0`.
- CPU Manager has allocated exclusive CPUs: kubelet always uses exactly the CPU Manager allocation for CPU Group affinity. Memory Manager derives its NUMA affinity from those CPUs and uses that affinity without extending it to additional NUMA nodes. The CPU Manager allocation is authoritative because it has already considered Topology Manager hints; expanding it could include CPUs exclusively assigned to other containers. See [#139684](https://github.com/kubernetes/kubernetes/pull/139684).

This NUMA-node-to-CPU mapping steers the container toward memory local to its assigned CPUs. However, Windows does not guarantee NUMA-local memory allocation, so a container can still access memory from a remote NUMA node and experience
reduced performance. Windows does not expose reliable per-container NUMA memory placement information, so kubelet cannot determine whether a container's memory is physically allocated on a remote NUMA node.

Kubelet's logical CPU and memory allocation decisions are retained in the manager checkpoint state and exposed through the Pod Resources API. Kubelet emits V(4) diagnostic logs when exclusive CPUs cannot be mapped to NUMA nodes and it falls back to the stored Topology Manager hint, or when the CPU-derived NUMA affinity differs from the stored hint. It emits V(5) logs when no
exclusive CPUs are assigned or when the CPU-derived and stored hints match. These logs describe kubelet's allocation decisions; they do not indicate physical memory placement or misalignment.

No metric or Pod condition for physical memory misalignment is proposed for beta because kubelet cannot observe that condition reliably on Windows. Additional persistent observability for kubelet-detectable allocation-plan differences may be evaluated based on feedback during beta.

Operators that want to minimize cross-NUMA memory access should configure the Topology Manager `single-numa-node` policy together with CPU Manager. This restricts admission to workloads for which sufficient CPU and memory resources are available on one NUMA node, but it does not change the best-effort nature of physical memory placement on Windows. A Windows-specific policy for workloads spanning multiple NUMA nodes may be considered in a separate KEP.

#### Kubelet memory management 

Windows support for [kubelet's memory eviction](https://github.com/kubernetes/kubernetes/pull/122922) was enabled in 1.31 and would follow the same patterns
as [Mechanism I](#mechanism-i-pod-eviction-by-kubelet).
Windows does not have an OOM killer and so Mechanisms II and III are out of scope in the section 
related to the [Kubernetes Node Memory Management](#kubernetes-nodes-memory-management-mechanisms-and-their-relation-to-the-memory-manager).

### Windows Topology manager considerations

Topology manager is already enabled on Windows in order to support the device manager.  Enabling the CPU and Memory manager as 
hint providers will be behind a feature flag. The CPU manager and Memory Manager can independently be enabled or disabled to support cases where the features needs to be shut off.  

### Test Plan

[x] I/we understand the owners of the involved components may require updates to
existing tests to make this code solid enough prior to committing the changes necessary
to implement this enhancement.

The Windows `e2e_node` suite now covers [CPU affinity behavior](https://github.com/kubernetes/kubernetes/blob/5a7afc5d7d11afa35ed0a88f8fb37b6acc47a637/test/e2e_node_windows/cpu_manager_test.go), [CPU manager metrics](https://github.com/kubernetes/kubernetes/blob/5a7afc5d7d11afa35ed0a88f8fb37b6acc47a637/test/e2e_node_windows/cpu_manager_metrics_test.go), [memory manager metrics](https://github.com/kubernetes/kubernetes/blob/5a7afc5d7d11afa35ed0a88f8fb37b6acc47a637/test/e2e_node_windows/memory_manager_metrics_test.go), [topology manager coordination](https://github.com/kubernetes/kubernetes/blob/5a7afc5d7d11afa35ed0a88f8fb37b6acc47a637/test/e2e_node_windows/topology_manager_test.go), and [topology manager metrics](https://github.com/kubernetes/kubernetes/blob/5a7afc5d7d11afa35ed0a88f8fb37b6acc47a637/test/e2e_node_windows/topology_manager_metrics_test.go). These tests configure the feature gate for both enabled and disabled scenarios and run in the periodic [ci-kubernetes-e2enode-windows-master](https://testgrid.k8s.io/sig-windows-signal#windows-e2e-node-master?include-filter-by-regex=Feature%3A(CPUManager%7CMemoryManager%7CTopologyManager)) job.

##### Prerequisite testing updates

##### Unit tests

- pkg/kubelet/cm/container_manager_windows.go
- pkg/kubelet/cm/internal_container_lifecycle_windows.go
- pkg/kubelet/winstats/cpu_topology_test.go

##### Integration tests

Kubernetes integration tests do not run on Windows. Windows functionality is covered by unit tests and Windows node e2e tests.

##### Node e2e tests

- Windows CPU, memory, and topology manager coverage runs in [ci-kubernetes-e2enode-windows-master](https://testgrid.k8s.io/sig-windows-signal#windows-e2e-node-master?include-filter-by-regex=Feature%3A(CPUManager%7CMemoryManager%7CTopologyManager)). The periodic job runs every eight hours.

### Graduation Criteria

#### Alpha

- Feature implemented behind a feature flag
- Initial basic e2e tests in Windows e2e suite are added
- unit tests for Windows specific components are added

#### Beta

- [x] Gather feedback from developers and users, and address feedback that identifies a correctness or usability issue. Developer feedback identified an affinity-coordination issue, resolved in [kubernetes/kubernetes#139684](https://github.com/kubernetes/kubernetes/pull/139684).
- [ ] Promote `WindowsCPUAndMemoryAffinity` to beta and enable it by default in Kubernetes v1.38.
- [x] Complete CPU, memory, and topology manager support on Windows behind the `WindowsCPUAndMemoryAffinity` feature gate, including validation with the supported containerd and runhcs versions documented in the Version Skew Strategy. See [CPU and Topology Manager support](https://github.com/kubernetes/kubernetes/pull/125296), [Memory Manager BestEffort support](https://github.com/kubernetes/kubernetes/pull/128560), [multi-group NUMA support](https://github.com/kubernetes/kubernetes/pull/137416), and the [CPU and memory affinity coordination fix](https://github.com/kubernetes/kubernetes/pull/139684).
- [ ] Complete security review and resolve identified security issues. Security review details: `TBD`.
- [x] Provide the CPU, memory, and topology manager metrics documented in this KEP through kubelet metrics.
- [x] Windows `e2e_node` tests for CPU affinity, memory manager metrics, topology manager coordination, and topology manager metrics run regularly and are green in [Testgrid](https://testgrid.k8s.io/sig-windows-signal#windows-e2e-node-master?include-filter-by-regex=Feature%3A(CPUManager%7CMemoryManager%7CTopologyManager)).
- [x] Complete testing requirements, including upgrade, downgrade, and re-upgrade validation.
- [ ] Provide beta-level documentation for configuration, supported runtime versions, monitoring, and recovery from manager state changes. Website PR: `TBD`.
  - Update the `WindowsCPUAndMemoryAffinity` feature-gate reference for beta and default-on in v1.38.
  - Update the Windows support sections for CPU Manager, Memory Manager, and Topology Manager to describe the v1.38 beta behavior, supported container runtime, and rollback through the feature gate.
- [x] Resolve all known prerelease issues and gaps. No open Kubernetes issues are specific to `WindowsCPUAndMemoryAffinity`; a Windows-specific topology policy for workloads spanning multiple NUMA nodes remains out of scope for this KEP and requires a separate KEP.

#### GA

- 2 examples of real-world usage
- Allowing time for feedback

**Note:** Generally we also wait at least two releases between beta and
GA/stable, because there's no opportunity for user feedback, or even bug reports,
in back-to-back releases.

**For non-optional features moving to GA, the graduation criteria must include
[conformance tests].**

[conformance tests]: https://git.k8s.io/community/contributors/devel/sig-architecture/conformance-tests.md

#### Deprecation

N/A

### Upgrade / Downgrade Strategy

There is no interaction with out Kubernetes components and upgrade / downgrade strategy is the same as the existing CPU/Memory/Topology manager.
https://github.com/kubernetes/enhancements/tree/master/keps/sig-node/1769-memory-manager#upgrade--downgrade-strategy
https://github.com/kubernetes/enhancements/tree/master/keps/sig-node/693-topology-manager#upgrade--downgrade-strategy
https://github.com/kubernetes/enhancements/tree/master/keps/sig-node/3570-cpumanager#upgrade--downgrade-strategy

### Version Skew Strategy

This feature requires updated to CRI-API (see above) and containerd in order to set CPU affinity when running Windows containers.
If the kubelet requests CPU affinity for a container and the container runtime does not support it, the container will be started without CPU affinity. This follows the same behavior as other kubelet enhancements that require container runtime support. Cluster operators that wish to use this feature are responsible to ensuring they have a container runtime that respects the CPU affinity settings since the kubelet doesn't perform minimum version checks for the container runtime or query the container runtime for its capabilities.
CPU affinity requires the CRI v1 `WindowsCpuGroupAffinity` field, added in Kubernetes v1.31, and containerd `v2.3.0` or later. The containerd `release/2.3` branch pairs with [runhcs v0.15.0-rc.4](https://github.com/containerd/containerd/blob/release/2.3/script/setup/runhcs-version); earlier containerd release branches use older runhcs versions and do not propagate affinity updates and status through CRI.
Multi-processor-group NUMA support requires Windows Server 2022 or later (build 20348 or later), which provides the `GetNumaNodeProcessorMask2` API used by kubelet.

## Production Readiness Review Questionnaire

This KEP discusses the changes required to enable for the various managers for Windows. 
This means many of the PRR questions for these features have already been covered and implemented 
as part of those KEPs.  We try to give details relevant to Windows but do not plan to change any of the
details of the features enablement in the KEP unless it is required because of a difference in Windows.  

https://github.com/kubernetes/enhancements/tree/master/keps/sig-node/1769-memory-manager#production-readiness-review-questionnaire
https://github.com/kubernetes/enhancements/tree/master/keps/sig-node/693-topology-manager#production-readiness-review-questionnaire
https://github.com/kubernetes/enhancements/tree/master/keps/sig-node/3570-cpumanager#production-readiness-review-questionnaire

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

- [x] Feature gate (also fill in values in `kep.yaml`)
  - Feature gate name: WindowsCPUAndMemoryAffinity
  - Components depending on the feature gate: Kubelet
  - Will enabling / disabling the feature require downtime of the control
    plane?
    No
  - Will enabling / disabling the feature require downtime or reprovisioning
    of a node?
    This is behavior is is the same as the features is implemented today in existing KEPs:

    https://github.com/kubernetes/enhancements/tree/master/keps/sig-node/3570-cpumanager#troubleshooting
    https://github.com/kubernetes/enhancements/tree/master/keps/sig-node/1769-memory-manager#feature-enablement-and-rollback

    Yes it uses a feature gate. Memory and CPU managers have a state file that requires cleanup.  After changing the CPU manager policy from none to static or the the other way around, before to start the kubelet again, you must remove the CPU manager state file(/var/lib/kubelet/cpu_manager_state), otherwise the kubelet start will fail. Startup failures for this reason will be logged in the kubelet log.

    Details for the steps to reset a state file are in https://kubernetes.io/docs/tasks/administer-cluster/cpu-management-policies/#changing-the-cpu-manager-policy. Memory manager has the same steps for resetting.

###### Does enabling the feature change any default behavior?

No. In v1.38, `WindowsCPUAndMemoryAffinity` will be beta and enabled by default. The default CPU and Memory Manager policies remain `None`, so the gate alone does not change workload behavior. CPU or memory affinity is applied only when an administrator configures a supported non-default manager policy.

See feature details in:

https://github.com/kubernetes/enhancements/tree/master/keps/sig-node/3570-cpumanager#feature-enablement-and-rollback
https://github.com/kubernetes/enhancements/tree/master/keps/sig-node/1769-memory-manager#feature-enablement-and-rollback
https://github.com/kubernetes/enhancements/tree/master/keps/sig-node/693-topology-manager#feature-enablement-and-rollback

###### Can the feature be disabled once it has been enabled (i.e. can we roll back the enablement)?

<!--
Describe the consequences on existing workloads (e.g., if this is a runtime
feature, can it break the existing applications?).

Feature gates are typically disabled by setting the flag to `false` and
restarting the component. No other changes should be necessary to disable the
feature.

NOTE: Also set `disable-supported` to `true` or `false` in `kep.yaml`.
-->

Yes.  A rolling restart (delete or delete and redeploy) of the pods will be required to remove the CPU/Memory affinity
from running pods.  Restarting kubelet after changing the feature will not affect any running pods but new pods created will be 
affected by the changes.

###### What happens if we reenable the feature if it was previously rolled back?

The Memory Manager and CPU managers utilize a state file to track assignments. If State file is not valid, it must be removed and kubelet restarted. E.g., State file might become invalid when kube/system reserved have changed (increased), which may lead to a situation when some containers cannot be started.

###### Are there any tests for feature enablement/disablement?

Yes. Unit tests cover manager state-file validation. Windows node e2e tests exercise the feature gate enabled and disabled and verify that a Guaranteed container does not receive exclusive CPU affinity when the gate is disabled.

<!--
The e2e framework does not currently support enabling or disabling feature
gates. However, unit tests in each component dealing with managing data, created
with and without the feature, are necessary. At the very least, think about
conversion tests if API types are being modified.

Additionally, for features that are introducing a new API field, unit tests that
are exercising the `switch` of feature gate itself (what happens if I disable a
feature gate after having objects written with the new field) are also critical.
You can take a look at one potential example of such test in:
https://github.com/kubernetes/kubernetes/pull/97058/files#diff-7826f7adbc1996a05ab52e3f5f02429e94b68ce6bce0dc534d1be636154fded3R246-R282
-->

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

Impact is node local, and doesn't affect rest of the cluster.

It is possible that the state file from the memory/cpu manager will have inconsistent data during the rollout, because of the kubelet restart, but you can easily to fix it by removing memory manager state file and run kubelet restart. It should not affect any running workloads.


###### What specific metrics should inform a rollback?

<!--
What signals should users be paying attention to when the feature is young
that might indicate a serious problem?
-->

The pod may fail with the admission error because the kubelet can not provide all resources. You can see the error messages under the pod events.

Monitor the CPU, memory, and topology manager metrics listed in the Monitoring Requirements section.

###### Were upgrade and rollback tested? Was the upgrade->downgrade->upgrade path tested?

<!--
Describe manual testing that was done and the outcomes.
Longer term, we may want to require automated upgrade/rollback tests, but we
are missing a bunch of machinery and tooling and can't do that now.
-->

The following node-local upgrade, downgrade, and state-recovery sequence was manually validated for beta.
It is not included in the periodic Windows `e2e_node` suite because the suite does not currently support
replacing the running kubelet binary, restarting it with different feature-gate configurations, or modifying
manager state files on the node.

1. Start a Windows node running kubelet v1.37.0 with CPU Manager `static`, Memory Manager `BestEffort`, and `WindowsCPUAndMemoryAffinity=true`, using containerd `v2.3.5` or later and its paired runhcs release. Verify a
  Guaranteed pod receives the expected CPU affinity.
2. Upgrade the kubelet to v1.38.0 without changing the manager policies or feature-gate configuration. Verify existing workloads continue running and a newly created Guaranteed pod receives the expected CPU affinity.
3. Downgrade the kubelet to v1.37.0 without changing the manager policies or feature-gate configuration. Verify existing workloads continue running and a newly created Guaranteed pod receives the expected CPU affinity.
4. Re-upgrade the kubelet to v1.38.0 without changing the manager policies or feature-gate configuration. Verify existing workloads continue running and a newly created Guaranteed pod receives the expected CPU affinity.
5. Disable `WindowsCPUAndMemoryAffinity` and restart the kubelet. Verify existing workloads continue running and newly created Guaranteed pods do not receive exclusive CPU affinity.
6. Re-enable `WindowsCPUAndMemoryAffinity` and restart the kubelet. Verify a newly created Guaranteed pod receives the expected CPU affinity.
7. When changing a CPU or memory manager policy, drain the node and remove the corresponding manager state file before restarting the kubelet. Verify the kubelet starts successfully and workloads receive the expected affinity.


| Evidence | Result |
| --- | --- |
| Kubernetes version | `v1.38.0` |
| containerd version | `2.3.5` |
| runhcs version | `v0.15.0-rc.4` |
| Windows Server version and build | `Windows Server 2022 Datacenter   10.0.20348.5256 (amd64)` |
| Test date | `2026-09-09` |
| Test job or Testgrid link | `Manual` |
| Upgrade, downgrade, and re-upgrade result | `Pass, upgraded kubelet from v1.37.0 to v1.38.0, downgraded it to v1.37.0, then re-upgraded to v1.38.0. Existing Pods continued running across each transition, and newly scheduled Guaranteed Pods showed the expected affinity behavior for the active feature-gate state.` |
| State-file recovery result | `Pass. After draining the node and removing the CPU and Memory Manager state files, kubelet restarted successfully and a new Guaranteed Pod received the expected CPU affinity` |

###### Is the rollout accompanied by any deprecations and/or removals of features, APIs, fields of API types, flags, etc.?

<!--
Even if applying deprecation policies, they may still surprise some users.
-->

No.

### Monitoring Requirements

<!--
This section must be completed when targeting beta to a release.

For GA, this section is required: approvers should be able to confirm the
previous answers based on experience in the field.
-->

The kubelet exposes the following metrics for CPU, memory, and topology manager behavior:

```text
# CPU Manager
cpu_manager_pinning_requests_total
cpu_manager_pinning_errors_total
cpu_manager_shared_pool_size_millicores
cpu_manager_exclusive_cpus_allocation_count

# Memory Manager
memory_manager_pinning_requests_total
memory_manager_pinning_errors_total

# Topology Manager
topology_manager_admission_requests_total
topology_manager_admission_errors_total
topology_manager_admission_duration_ms
```

These metrics are also listed in `kep.yaml` for beta PRR validation.

###### How can an operator determine if the feature is in use by workloads?

<!--
Ideally, this should be a metric. Operations against the Kubernetes API (e.g.,
checking if there are objects with field X set) may be a last resort. Avoid
logs or events for this purpose.
-->

Operators can use the CPU, memory, and topology manager metrics listed below to identify allocation and admission activity. CPU allocations are also visible through the kubelet Pod Resources API.

###### How can someone using this feature know that it is working for their instance?

- [x] Other (treat as last resort)
  - Details: check the kubelet metric `cpu_manager_pinning_requests_total`
  - check the kubelet metric `memory_manager_pinning_requests_total`

###### What are the reasonable SLOs (Service Level Objectives) for the enhancement?

No feature-specific SLOs are introduced. Existing kubelet SLOs continue to apply.

###### What are the SLIs (Service Level Indicators) an operator can use to determine the health of the service?

The CPU, memory, and topology manager pinning and admission request and error metrics listed above are the SLIs for this feature.

###### Are there any missing metrics that would be useful to have to improve observability of this feature?

No additional feature-specific metrics are required for beta. The CPU, memory, and topology manager metrics listed in `kep.yaml` provide the required observability.

### Dependencies


###### Does this feature depend on any specific services running in the cluster?

This feature requires the CRI v1 `WindowsCpuGroupAffinity` field and containerd `v2.3.0` or later with its paired runhcs release, as documented in the Version Skew Strategy.

### Scalability

<!--
For alpha, this section is encouraged: reviewers should consider these questions
and attempt to answer them.

For beta, this section is required: reviewers must answer these questions.

For GA, this section is required: approvers should be able to confirm the
previous answers based on experience in the field.
-->

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

We will monitor for cpu consumption to query the CPU topology.  If required we may wish to implement a caching strategy while also
supporting any new support for dynamic node resizing.

###### Can enabling / using this feature result in resource exhaustion of some node resources (PIDs, sockets, inodes, etc.)?

Memory and CPU's could be exhausted resulting in Pods not being scheduled.

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

N/a

###### What are other known failure modes?

The failure modes for pods on the node are the same as in CPU/Memory/topology Manager

###### What steps should be taken if SLOs are not being met to determine the problem?

No feature-specific SLOs are defined. Operators should inspect pod events for admission errors and monitor the CPU, memory, and topology manager pinning and admission error metrics listed above.

## Implementation History

<!--
Major milestones in the lifecycle of a KEP should be tracked in this section.
Major milestones might include:
- the `Summary` and `Motivation` sections being merged, signaling SIG acceptance
- the `Proposal` section being merged, signaling agreement on a proposed design
- the date implementation started
- the first Kubernetes release where an initial version of the KEP was available
- the version of Kubernetes where the KEP graduated to general availability
- when the KEP was retired or superseded
-->

- 2024-09-03: KEP created.
- v1.32: Alpha implementation introduced.
- v1.38: Beta graduation targeted.

## Drawbacks

<!--
Why should this KEP _not_ be implemented?
-->

## Alternatives

<!--
What other approaches did you consider, and why did you rule them out? These do
not need to be as detailed as the proposal, but should include enough
information to express the idea and why it was not acceptable.
-->

## Infrastructure Needed (Optional)

<!--
Use this section if you need things from the project/SIG. Examples include a
new subproject, repos requested, or GitHub details. Listing these here allows a
SIG to get the process for these resources started right away.
-->

n/a Windows will use existing testing infrastructure
