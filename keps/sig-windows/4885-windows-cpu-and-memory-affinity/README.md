<!--
**Note:** When your KEP is complete, all of these comment blocks should be removed.

To get started with this template:

- [x] **Pick a hosting SIG.**
  Make sure that the problem space is something the SIG is interested in taking
  up. KEPs should not be checked in without a sponsoring SIG.
- [x] **Create an issue in kubernetes/enhancements**
  When filing an enhancement tracking issue, please make sure to complete all
  fields in that template. One of the fields asks for a link to the KEP. You
  can leave that blank until this KEP is filed, and then go back to the
  enhancement and add the link.
- [x] **Make a copy of this template directory.**
  Copy this template into the owning SIG's directory and name it
  `NNNN-short-descriptive-title`, where `NNNN` is the issue number (with no
  leading-zero padding) assigned to your enhancement above.
- [x] **Fill out as much of the kep.yaml file as you can.**
  At minimum, you should fill in the "Title", "Authors", "Owning-sig",
  "Status", and date-related fields.
- [x] **Fill out this file as best you can.**
  At minimum, you should fill in the "Summary" and "Motivation" sections.
  These should be easy if you've preflighted the idea of the KEP with the
  appropriate SIG(s).
- [x] **Create a PR for this KEP.**
  Assign it to people in the SIG who are sponsoring this process.
- [x] **Merge early and iterate.**
  Avoid getting hung up on specific details and instead aim to get the goals of
  the KEP clarified and merged quickly. The best way to do this is to just
  start with the high-level sections and fill out details incrementally in
  subsequent PRs.

Just because a KEP is merged does not mean it is complete or approved. Any KEP
marked as `provisional` is a working document and subject to change. You can
denote sections that are under active debate as follows:

```
<<[UNRESOLVED optional short context or usernames ]>>
Stuff that is being argued.
<<[/UNRESOLVED]>>
```

When editing KEPS, aim for tightly-scoped, single-topic PRs to keep discussions
focused. If you disagree with what is already in a document, open a new PR
with suggested changes.

One KEP corresponds to one "feature" or "enhancement" for its whole lifecycle.
You do not need a new KEP to move from beta to GA, for example. If
new details emerge that belong in the KEP, edit the KEP. Once a feature has become
"implemented", major changes should get new KEPs.

The canonical place for the latest set of instructions (and the likely source
of this file) is [here](/keps/NNNN-kep-template/README.md).

**Note:** Any PRs to move a KEP to `implementable`, or significant changes once
it is marked `implementable`, must be approved by each of the KEP approvers.
If none of those approvers are still appropriate, then changes to that list
should be approved by the remaining approvers and/or the owning SIG (or
SIG Architecture for cross-cutting KEPs).
-->
# KEP-NNNN: Your short, descriptive title

<!--
This is the title of your KEP. Keep it short, simple, and descriptive. A good
title can help communicate what the KEP is and should be considered as part of
any review.
-->

<!--
A table of contents is helpful for quickly jumping to sections of a KEP and for
highlighting any additional information provided beyond the standard KEP
template.

Ensure the TOC is wrapped with
  <code>&lt;!-- toc --&rt;&lt;!-- /toc --&rt;</code>
tags, and then generate with `hack/update-toc.sh`.
-->

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

<!--
**Note:** This checklist is iterative and should be reviewed and updated every time this enhancement is being considered for a milestone.
-->

[kubernetes.io]: https://kubernetes.io/
[kubernetes/enhancements]: https://git.k8s.io/enhancements
[kubernetes/kubernetes]: https://git.k8s.io/kubernetes
[kubernetes/website]: https://git.k8s.io/website

## Summary

<!--
This section is incredibly important for producing high-quality, user-focused
documentation such as release notes or a development roadmap. It should be
possible to collect this information before implementation begins, in order to
avoid requiring implementors to split their attention between writing release
notes and implementing the feature itself. KEP editors and SIG Docs
should help to ensure that the tone and content of the `Summary` section is
useful for a wide audience.

A good summary is probably at least a paragraph in length.

Both in this section and below, follow the guidelines of the [documentation
style guide]. In particular, wrap lines to a reasonable length, to make it
easier for reviewers to cite specific portions, and to minimize diff churn on
updates.

[documentation style guide]: https://github.com/kubernetes/community/blob/master/contributors/guide/style-guide.md
-->

## Motivation

<!--
This section is for explicitly listing the motivation, goals, and non-goals of
this KEP.  Describe why the change is important and the benefits to users. The
motivation section can optionally provide links to [experience reports] to
demonstrate the interest in a KEP within the wider Kubernetes community.

[experience reports]: https://github.com/golang/go/wiki/ExperienceReports
-->

Add support for CPU and memory affinity for Windows nodes by enabling the cpu, memory and topology managers for Windows, 
which are currently not enabled.

Enabling Low latency workloads co-hosted on the same nodes in Windows Server show noisy neighbor behaviors 
preventing them from achieving their expected performance goals. 
This feature is needed to add the necessary isolation to accomplish both high performance and co-hosting efficiency.  
The feature is enabled and available in Linux and Windows users are asking for the same features on Windows.

### Goals

<!--
List the specific goals of the KEP. What is it trying to achieve? How will we
know that this has succeeded?
-->
- Enable CPU manager for Windows allowing for CPU affinity for configured pods
- Enable Memory Manager for Windows allowing for memory affinity for configured pods
- Enable Topology Manager for Windows allowing for coordination of Memory and CPU affinity at the node level for scheduled pods

### Non-Goals

<!--
What is out of scope for this KEP? Listing non-goals helps to focus discussion
and make progress.
-->

- We do not wish to create new managers and instead re-use the existing logic provided 
- Modify or bypass any existing feature gated features.  Existing Policy features gates will still be used to progress specific policies related to the managers.

## Proposal

<!--
This is where we get down to the specifics of what the proposal actually is.
This should have enough detail that reviewers can understand exactly what
you're proposing, but should not include things like API designs or
implementation. What is the desired outcome and how do we measure success?.
The "Design Details" section below is for the real
nitty-gritty.
-->

The proposal requires very little changes to the code for the managers and instead extends the [Windows](https://learn.microsoft.com/en-us/windows/win32/procthread/processor-groups) concepts to a CAdvisor mapping to enable the [topology structure in kubelet](https://github.com/kubernetes/kubernetes/blob/cede96336a809a67546ca08df0748e4253ec270d/pkg/kubelet/cm/cpumanager/topology/topology.go#L34-L39).

There are no plans to change the core logic for selecting CPU's and NUMA nodes in the CPU/Memory/Tolopology managers from the existing KEPS ([memory-manager](keps/sig-node/1769-memory-manager)/[cpu-manager](keps/sig-node/3570-cpu-manager)/[topology-manager](keps/sig-node/693-topology-manager")).  The logic is currently in platform agnostic 
structures so the selection process is does not require changes for adoption on Windows.  The Windows specific considerations for each of the managers will be covered in separate sections in this document.  


### User Stories (Optional)

<!--
Detail the things that people will be able to do if this KEP is implemented.
Include as much detail as possible so that people can understand the "how" of
the system. The goal here is to make this feel real for users without getting
bogged down.
-->

The User stories on Windows are similar to Linux:

https://github.com/kubernetes/enhancements/tree/master/keps/sig-node/3570-cpumanager#user-stories-optional
https://github.com/kubernetes/enhancements/tree/master/keps/sig-node/1769-memory-manager#user-stories
https://github.com/kubernetes/enhancements/tree/master/keps/sig-node/693-topology-manager#user-stories-optional

### Notes/Constraints/Caveats (Optional)

<!--
What are the caveats to the proposal?
What are some important details that didn't come across above?
Go in to as much detail as necessary here.
This might be a good place to talk about core concepts and how they relate.
-->

### Risks and Mitigations

<!--
What are the risks of this proposal, and how do we mitigate? Think broadly.
For example, consider both security and how this will impact the larger
Kubernetes ecosystem.

How will security be reviewed, and by whom?

How will UX be reviewed, and by whom?

Consider including folks who also work outside the SIG or subproject.
-->

The technical risks are the same from existing keps: 
 - https://github.com/kubernetes/enhancements/tree/master/keps/sig-node/3570-cpumanager#risks-and-mitigations
 - https://github.com/kubernetes/enhancements/tree/master/keps/sig-node/1769-memory-manager#risks-and-mitigations
 - https://github.com/kubernetes/enhancements/tree/master/keps/sig-node/693-topology-manager#risks-and-mitigations

For sig-windows, we also see a risk to enabling a feature that has already Stable or fully featured on Linux.  To mitigate this risk we have opted to create a 
separate KEP with a feature flag so we can communicate our status effectively. 

Another risk is the testing implementation for these features is mostly in e2e_node which doesn't currently support Windows.  As a mitigation there was [some exploration ](https://github.com/jsturtevant/kubernetes/tree/e2e_node-windows) to see if these tests could be enabled on Windows so we can progress this feature with confidence in the testing suite.

## Design Details

<!--
This section should contain enough information that the specifics of your
change are understandable. This may include API specs (though not always
required) or even code snippets. If there's any ambiguity about HOW your
proposal will be implemented, this is the place to discuss them.
-->


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

In Cri we will add the following structure to the `WindowsContainerResources` in CRI:

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
		if GROUP_AFFINITY.Mask&(1<<i) != 0 {
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

The testing plan is to enable basic tests in [Windows testing folder](https://github.com/kubernetes/kubernetes/tree/master/test/e2e/windows) in Alpha.  This will enable us to progress to a state we in Alpha that will allow our end users to test and give feedback in real world scenarios. 

We we also work to enable e2e_node test suite to run on Windows and enable the applicable [CPU](https://github.com/kubernetes/kubernetes/blob/master/test/e2e_node/container_manager_test.go)/[Memory](https://github.com/kubernetes/kubernetes/blob/master/test/e2e_node/memory_manager_test.go)/[Topology](https://github.com/kubernetes/kubernetes/blob/master/test/e2e_node/topology_manager_test.go) Manager tests for Beta.  The goal will be to enable as many of those tests as possible while recognizing some may not be applicable to Windows.  Where we find gaps we will fill them with Windows specific tests.

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

- pkg/kubelet/cm/container_manager_windows.go
- pkg/kubelet/cm/internal_container_lifecycle_windows.go
- pkg/kubelet/winstats/cpu_topology_test.go

##### Integration tests

<!--
Integration tests are contained in k8s.io/kubernetes/test/integration.
Integration tests allow control of the configuration parameters used to start the binaries under test.
This is different from e2e tests which do not allow configuration of parameters.
Doing this allows testing non-default options and multiple different and potentially conflicting command line options.
-->

<!--
This question should be filled when targeting a release.
For Alpha, describe what tests will be added to ensure proper quality of the enhancement.

For Beta and GA, add links to added tests together with links to k8s-triage for those tests:
https://storage.googleapis.com/k8s-triage/index.html
-->

Integration tests do not run on Windows. Functionality will be covered by unit and e2e tests.

##### e2e tests

<!--
This question should be filled when targeting a release.
For Alpha, describe what tests will be added to ensure proper quality of the enhancement.

For Beta and GA, add links to added tests together with links to k8s-triage for those tests:
https://storage.googleapis.com/k8s-triage/index.html

We expect no non-infra related flakes in the last month as a GA graduation criteria.
-->

-  e2e_node will need to be enabled for windows to add coverage

### Graduation Criteria

<!--
**Note:** *Not required until targeted at a release.*

Define graduation milestones.

These may be defined in terms of API maturity, [feature gate] graduations, or as
something else. The KEP should keep this high-level with a focus on what
signals will be looked at to determine graduation.

Consider the following in developing the graduation criteria for this enhancement:
- [Maturity levels (`alpha`, `beta`, `stable`)][maturity-levels]
- [Feature gate][feature gate] lifecycle
- [Deprecation policy][deprecation-policy]

Clearly define what graduation means by either linking to the [API doc
definition](https://kubernetes.io/docs/concepts/overview/kubernetes-api/#api-versioning)
or by redefining what graduation means.

In general we try to use the same stages (alpha, beta, GA), regardless of how the
functionality is accessed.

[feature gate]: https://git.k8s.io/community/contributors/devel/sig-architecture/feature-gates.md
[maturity-levels]: https://git.k8s.io/community/contributors/devel/sig-architecture/api_changes.md#alpha-beta-and-stable-versions
[deprecation-policy]: https://kubernetes.io/docs/reference/using-api/deprecation-policy/

Below are some examples to consider, in addition to the aforementioned [maturity levels][maturity-levels].
-->
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

- Announce deprecation and support policy of the existing flag
- Two versions passed since introducing the functionality that deprecates the flag (to address version skew)
- Address feedback on usage/changed behavior, provided on GitHub issues
- Deprecate the flag


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

### Version Skew Strategy

<!--
If applicable, how will the component handle version skew with other
components? What are the guarantees? Make sure this is in the test plan.

Consider the following in developing a version skew strategy for this
enhancement:
- Does this enhancement involve coordinating behavior in the control plane and nodes?
- How does an n-3 kubelet or kube-proxy without this feature available behave when this feature is used?
- How does an n-1 kube-controller-manager or kube-scheduler without this feature available behave when this feature is used?
- Will any other components on the node change? For example, changes to CSI,
  CRI or CNI may require updating that component before the kubelet.
-->

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
  - Components depending on the feature gate:
- [ ] Other
  - Describe the mechanism:
  - Will enabling / disabling the feature require downtime of the control
    plane?
    No
  - Will enabling / disabling the feature require downtime or reprovisioning
    of a node?
    Yes it uses a feature gate. Memory manager also has a state file that requires cleanup.

###### Does enabling the feature change any default behavior?

<!--
Any change of default behavior may be surprising to users or break existing
automations, so be extremely careful here.
-->

No, Additional settings are required to enable the features.  The default policies for CPU/Memory manager will be `None`, meaning that they will not interact with running of pods.

###### Can the feature be disabled once it has been enabled (i.e. can we roll back the enablement)?

<!--
Describe the consequences on existing workloads (e.g., if this is a runtime
feature, can it break the existing applications?).

Feature gates are typically disabled by setting the flag to `false` and
restarting the component. No other changes should be necessary to disable the
feature.

NOTE: Also set `disable-supported` to `true` or `false` in `kep.yaml`.
-->

Yes.  Restarting of the pods will be required to remove the CPU/Memory affinity.

###### What happens if we reenable the feature if it was previously rolled back?

###### Are there any tests for feature enablement/disablement?

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

The memory/cpu manager will be under the pod resources API.

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

n/a

###### What are the SLIs (Service Level Indicators) an operator can use to determine the health of the service?

<!--
Pick one more of these and delete the rest.
-->

These will be the same as cpu/memory/topology manager.

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

No

###### Will enabling / using this feature result in introducing new API types?

<!--
Describe them, providing:
  - API type
  - Supported number of objects per cluster
  - Supported number of objects per namespace (for namespace-scoped objects)
-->

No

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

No

###### Will enabling / using this feature result in increasing time taken by any operations covered by existing SLIs/SLOs?

<!--
Look at the [existing SLIs/SLOs].

Think about adding additional work or introducing new steps in between
(e.g. need to do X to start a container), etc. Please describe the details.

[existing SLIs/SLOs]: https://git.k8s.io/community/sig-scalability/slos/slos.md#kubernetes-slisslos
-->

No

###### Will enabling / using this feature result in non-negligible increase of resource usage (CPU, RAM, disk, IO, ...) in any components?

<!--
Things to keep in mind include: additional in-memory state, additional
non-trivial computations, excessive access to disks (including increased log
volume), significant amount of data sent and/or received over network, etc.
This through this both in small and large cases, again with respect to the
[supported limits].

[supported limits]: https://git.k8s.io/community//sig-scalability/configs-and-limits/thresholds.md
-->

We will monitor for cpu consumption to query the CPU topology.  If required we may wish to implement a caching strategy.

###### Can enabling / using this feature result in resource exhaustion of some node resources (PIDs, sockets, inodes, etc.)?

<!--
Focus not just on happy cases, but primarily on more pathological cases
(e.g. probes taking a minute instead of milliseconds, failed pods consuming resources, etc.).
If any of the resources can be exhausted, how this is mitigated with the existing limits
(e.g. pods per node) or new limits added by this KEP?

Are there any tests that were run/should be run to understand performance characteristics better
and validate the declared limits?
-->

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
