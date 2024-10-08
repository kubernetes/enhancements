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
- [Infrastructure Needed (Optional)](#infrastructure-needed-optional)
<!-- /toc -->

## Release Signoff Checklist

Items marked with (R) are required *prior to targeting to a milestone / release*.

- [x] (R) Enhancement issue in release milestone, which links to KEP dir in [kubernetes/enhancements] (not the initial KEP PR)
- [ ] (R) KEP approvers have approved the KEP status as `implementable`
- [x] (R) Design details are appropriately documented
- [x] (R) Test plan is in place, giving consideration to SIG Architecture and SIG Testing input (including test refactors)
  - [ ] e2e Tests for all Beta API Operations (endpoints)
  - [x] (R) Ensure GA e2e tests meet requirements for [Conformance Tests](https://github.com/kubernetes/community/blob/master/contributors/devel/sig-architecture/conformance-tests.md) 
  - [ ] (R) Minimum Two Week Window for GA e2e tests to prove flake free
- [x] (R) Graduation criteria is in place
  - [ ] (R) [all GA Endpoints](https://github.com/kubernetes/community/pull/1806) must be hit by [Conformance Tests](https://github.com/kubernetes/community/blob/master/contributors/devel/sig-architecture/conformance-tests.md) 
- [ ] (R) Production readiness review completed
- [ ] (R) Production readiness review approved
- [ ] "Implementation History" section is up-to-date for milestone
- [ ] User-facing documentation has been created in [kubernetes/website], for publication to [kubernetes.io]
- [ ] Supporting documentation—e.g., additional design documents, links to mailing list discussions/SIG meetings, relevant PRs/issues, release notes

[kubernetes.io]: https://kubernetes.io/
[kubernetes/enhancements]: https://git.k8s.io/enhancements
[kubernetes/kubernetes]: https://git.k8s.io/kubernetes
[kubernetes/website]: https://git.k8s.io/website

## Summary

This kep outlines how to add support for the CPU, Memory and Topology Managers in kubelet for Windows.  
The Managers are already available and support in kubelet on Linux and there have been requests to sig-windows
to add support on Windows to help with workloads that require co-located workloads.  The goal of the KEP is to 
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
structures so the selection process is does not require changes for adoption on Windows.  The Windows specific considerations for each of the managers will be covered in separate sections in this document.  


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

Another risk is the testing implementation for these features is mostly in e2e_node which doesn't currently support Windows.  As a mitigation there was [some exploration](https://github.com/jsturtevant/kubernetes/tree/e2e_node-windows) to see if these tests could be enabled on Windows so we can progress this feature with confidence in the testing suite.

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

To work around these limitations, the kubelet will query the OS to get the affinity masks associated with each of the Numa nodes selected by the memory manager and update the CPU Group affinity accordingly in the CRI field. This will result in the memory from the Numa node being used. There are a couple scenarios that need to be considered:

- Memory manager is enabled, cpu manager is not: kubelet will look up all the cpu's associated with the selected Numa nodes and assign the CPU Group affinity.  For example if NumaNode 0 is selected by memory manager, and NumaNode 0 has the first four CPU's in Windows CPU group 0 the result would be `cpu affinity: 0000001111, group 0`.  
- Memory manager is enabled, CPU manager is enabled
  - cpu manager selects fewer CPU's than Numa nodes and CPU's fall with in Numa node: Kubelet will only set only the CPU's selected by the cpu-manager as the memory from the memory manager will be used by default.  
  - cpu manager selects more CPU's than Numa nodes and CPU's fall within/or outside Numa node: kubelet will set selected only CPU's from cpu-manager
  - cpu manager selects fewer CPU's than the CPU's associated with the Numa nodes selected by the memory manager: Kubelet would set the CPU's by cpu-manager plus all the CPU's associated with the Numa node.  

Using Memory manager's internal mapping this should provide the desired behavior in most cases. Since Memory affinity isn't guaranteed, It is possible that a CPU could access memory from a different Numa 
Node than it is currently in, resulting in decreased performance. For this reason, we will add documentation, a log warning message in kubelet, and an warning event 
to help raise awareness of this possibility. If access from the CPUs different than the assigned Numa Node is undesirable then `single-numa-node` 
and the CPU manager should be configured in the Topology Manager policy setting which would force Kubelet to only select a Numa node if it will have enough memory 
and CPU's available.  In the future, in the case of workloads that span multiple Numa nodes, it may be desirable for Topology manager to have a new policy specific 
for Windows. This would require a separate KEP to add a new policy.

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

The testing plan is to enable basic tests in [Windows testing folder](https://github.com/kubernetes/kubernetes/tree/master/test/e2e/windows) in Alpha.  This will enable us to progress to a state we in Alpha that will allow our end users to test and give feedback in real world scenarios. 

We we also work to enable e2e_node test suite to run on Windows and enable the applicable [CPU](https://github.com/kubernetes/kubernetes/blob/master/test/e2e_node/container_manager_test.go)/[Memory](https://github.com/kubernetes/kubernetes/blob/master/test/e2e_node/memory_manager_test.go)/[Topology](https://github.com/kubernetes/kubernetes/blob/master/test/e2e_node/topology_manager_test.go) Manager tests for Beta.  The goal will be to enable as many of those tests as possible while recognizing some may not be applicable to Windows.  Where we find gaps we will fill them with Windows specific tests.

##### Prerequisite testing updates

##### Unit tests

- pkg/kubelet/cm/container_manager_windows.go
- pkg/kubelet/cm/internal_container_lifecycle_windows.go
- pkg/kubelet/winstats/cpu_topology_test.go

##### Integration tests

Integration tests do not run on Windows. Functionality will be covered by unit and e2e tests.

##### e2e tests

-  e2e_node will need to be enabled for Windows to add coverage.  We plan to enable just e2e tests that relate to memory/cpu/topology manager, not the full suite.

### Graduation Criteria

#### Alpha

- Feature implemented behind a feature flag
- Initial basic e2e tests in Windows e2e suite are added
- unit tests for Windows specific components are added

#### Beta

- Gather feedback from developers 
- e2e_node tests are in Testgrid and linked in KEP

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

### Version Skew Strategy

This feature is kubelet specific, so version skew strategy is N/A.

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

No, Additional settings are required to enable the features.  The default policies for CPU/Memory manager will be `None`, meaning that they will not interact with running of pods.  The Cluster administrator will need to set specific CPU/Memory/Topology manager policies 
to enable any features described here.

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

Yes, there is a number of Unit Tests designated for State file validation.

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

There are existing metrics provided by Managers that can be monitored:

```golang
// Metrics to track the CPU manager behavior
CPUManagerPinningRequestsTotalKey         = "cpu_manager_pinning_requests_total"
CPUManagerPinningErrorsTotalKey           = "cpu_manager_pinning_errors_total"
CPUManagerSharedPoolSizeMilliCoresKey     = "cpu_manager_shared_pool_size_millicores"
CPUManagerExclusiveCPUsAllocationCountKey = "cpu_manager_exclusive_cpu_allocation_count"

// Metrics to track the Memory manager behavior
MemoryManagerPinningRequestsTotalKey = "memory_manager_pinning_requests_total"
MemoryManagerPinningErrorsTotalKey   = "memory_manager_pinning_errors_total"

// Metrics to track the Topology manager behavior
TopologyManagerAdmissionRequestsTotalKey = "topology_manager_admission_requests_total"
TopologyManagerAdmissionErrorsTotalKey   = "topology_manager_admission_errors_total"
TopologyManagerAdmissionDurationKey      = "topology_manager_admission_duration_ms"
```

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

We will use the existing Metrics provided by CPU/Memory Manager.

https://github.com/kubernetes/enhancements/tree/master/keps/sig-node/3570-cpumanager#monitoring-requirements
https://github.com/kubernetes/enhancements/tree/master/keps/sig-node/1769-memory-manager#monitoring-requirements

###### How can an operator determine if the feature is in use by workloads?

<!--
Ideally, this should be a metric. Operations against the Kubernetes API (e.g.,
checking if there are objects with field X set) may be a last resort. Avoid
logs or events for this purpose.
-->

The memory/cpu manager will be under the pod resources API. And there are proposed metrics to improve this in [kubernetes/kubernetes#127155](https://github.com/kubernetes/kubernetes/pull/127155)

###### How can someone using this feature know that it is working for their instance?

- [X] Other (treat as last resort)
  - Details: check the kubelet metric `cpu_manager_pinning_requests_total`
  - check the kubelet metric `memory_manager_pinning_requests_total`

###### What are the reasonable SLOs (Service Level Objectives) for the enhancement?

n/a

###### What are the SLIs (Service Level Indicators) an operator can use to determine the health of the service?

These will be the same as cpu/memory/topology manager.

###### Are there any missing metrics that would be useful to have to improve observability of this feature?

Since the CPU/Memory/Topology manager are already implemented most of the metrics are implemented.  If we find missing
metrics on Windows we will address as we move to Beta/Stable.

### Dependencies


###### Does this feature depend on any specific services running in the cluster?

This will require changes to CRI and containerd Windows agents.

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