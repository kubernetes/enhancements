---
title: Support configuring an exact cpuset as "system-reserved" in Kubelet
authors:
  - "@Levovar"
owning-sig: sig-node
participating-sigs:
  - sig-node
reviewers:
  - "@ConnorDoyle"
  - "@derekwaynecarr"
  - "@jianzzha"
approvers:
  - TBD
editor: TBD
creation-date: 2019-10-14
last-updated: 2019-10-17
status: provisional
see-also:
  - https://github.com/kubernetes/community/pull/2435
  - https://github.com/kubernetes/kubernetes/pull/83592
replaces:
  - N/A
superseded-by:
  - N/A
---

# Support configuring an exact cpuset as "system-reserved" in Kubelet

## Table of Contents

<!-- toc -->
- [Release Signoff Checklist](#release-signoff-checklist)
- [Summary](#summary)
- [Motivation](#motivation)
  - [Goals](#goals)
  - [Non-Goals](#non-goals)
- [Proposal](#proposal)
  - [User Stories](#user-stories)
    - [User Story 1](#user-story-1)
    - [User Story 2](#user-story-2)
    - [User Story 3](#user-story-3)
  - [Implementation Details/Notes/Constraints](#implementation-detailsnotesconstraints)
    - [Configuration](#configuration)
    - [Inter-working with other features](#inter-working-with-other-features)
    - [Proper cpuset configuration for non-static policy use-cases](#proper-cpuset-configuration-for-non-static-policy-use-cases)
  - [Risks and Mitigations](#risks-and-mitigations)
- [Design Details](#design-details)
  - [Graduation Criteria](#graduation-criteria)
  - [Upgrade / Downgrade Strategy](#upgrade--downgrade-strategy)
<!-- /toc -->

## Release Signoff Checklist

- [ ] kubernetes/enhancements issue in release milestone, which links to KEP (this should be a link to the KEP location in kubernetes/enhancements, not the initial KEP PR)
- [ ] KEP approvers have set the KEP status to `implementable`
- [ ] Design details are appropriately documented
- [ ] Test plan is in place, giving consideration to SIG Architecture and SIG Testing input
- [ ] Graduation criteria is in place
- [ ] "Implementation History" section is up-to-date for milestone
- [ ] User-facing documentation has been created in [kubernetes/website], for publication to [kubernetes.io]
- [ ] Supporting documentation e.g., additional design documents, links to mailing list discussions/SIG meetings, relevant PRs/issues, release notes

## Summary
Kubelet supports configuring a subset of the Node resources it shall not actively manage via the "system-reserved" flag.
Currently only CPU, and memory are supported. 
The CPU Manager -more precisely its static policy- implemented within Kubelet interprets the absolute amount of "CPU shares" coming from this configuration as the "first X number of core IDs I shall not touch".
X denotes the smallest amount of vCPUs (threads, or physical cores) which are enough to satisfy the request.
I.e. for system-reserved=1200m CPU Manager excludes CPU cores 0, and 1 form the list of CPU cores usable by K8s workloads.
This KEP proposes to enhance the format, and the meaning of this flag so that it becomes possible to configure an exact list of CPU cores Kubelet shall entirely exclude from its supervision, under all circumstances.
The exclusion should be always applicable independently from other CPU management related configuration options such as enablement of CPU Manager, or the Node's CPU management policy (i.e. "none", or "static").

## Motivation
Kubelet always assumes that it is the primary software component managing the CPU cores of the host.
However, in certain infrastructures this might not always be the case.
While it is already possible to effectively take-away CPU cores from the Kubernetes managed workloads via the kube-reserved and system-reserved kubelet flags, the current implementation has two perceived shortcomings.
1: reservations are tied to the CPU Manager, and its static policy
2: this implicit way of declaring a Kubernetes managed CPU pool is not flexible enough to cover all use-cases

To accommodate the use-cases -mainly related to seamlessly inter-working with multiple resource managers on the same node-, the need arises to enhance existing CPU management implementation to be able to explicitly, and universally define a discontinuous pool of non-manageable CPUs.
Such feature could come in handy if one would like to:
- ensure proper resource accounting and separation between systemd processes and Kubernetes managed workloads
- ensure proper resource accounting and separation within a hybrid infrastructure (e.g. Openstack + Kubernetes resource managers running on the same node etc.)
- outsource the management of a subset of specialized, or optimized cores (e.g. real-time enabled CPUs, CPUs with different HT configuration etc.) to an external CPU manager
Note regarding third use-case: yes, it is not a nice use-case. But the unfortunate reality is that CPU Manager static policy is not used in a big number (maybe even majority) of the Kubernetes-based TelCo cloud implementations.
Until (if ever?) the underlying structure of CPU management can be aligned to serve all requirements, this non-intrusive change could serve as the stop-gap solution!

### Goals
The goal is to make any and all Kubernetes created cpuset cgroups explicitly adjustable to be able to ensure resources do not overlap between Kubernetes, and non-Kubernetes managed processes.

### Non-Goals
It is outside the scope of this KEP to restrict any other Kubernetes resource manager to a subset of a Node's resource group (like memory, devices, etc.).
It is also outside the scope of this KEP to enhance Kubelet's CPU manager itself with more fine-grained management policies.
The aim of this KEP is to continue to let Kubernetes manage some CPU cores however it sees fit, but at the same time also eliminate the possibility of accidentally scheduling workloads to restricted resources.
Lastly, while it would be an interesting research topic of how different CPU managers (one of them being Kubelet) could inter-work with each other in run-time to dynamically re-partition the CPU sets they manage, it is unfortunately also outside the scope of this simple KEP.
What this enhancement is trying to achieve first and foremost is guaranteed, statically configured isolation. Alignment of the isolated resources is left to the cloud infrastructure operators at this stage of the feature.

## Proposal

#### User Story 1
As an infrastructure operator, I would like to exclusively dedicate some discontinuously numbered CPU cores to Linux services not supervised by Kubernetes

Kubelet already having system-reserved flag enforces the idea that resource management community already recognized this basic use-case to be valid in today's changing world.
Not every low-level infrastructure process was able to, or wanted to transform its architecture to a containerized, micro-service based deployment model.
Kubernetes resource management currently advocates physically separating these different services to different Nodes, but basically every cloud infrastructure runs some processes directly on its hosts.
It is not necessarily the case that these services are always restricted to the first X, continuously numbered set of cores; especially on a multi-socket system.

This feature would give cloud administrators' a chance to be able to at least manually separate the CPU cores used by these processes; which in some cases might be mission critical (e.g. cores used by PTP synchronization, cloud high-availability services etc.)

#### User Story 2
As an infrastructure operator, I would like to run multiple cloud infrastructures in the same -edge- cloud

This user-story is actually very similar to the previous one, but concentrates on separating workloads. Imagine that an operator would like to run Openstack, VMware or any other popular cloud infrastructure next to Kubernetes, on the same set of Nodes.
Sometimes an operator simply does not have the possibility to separate her infrastructures on the host level, because simply there are not enough nodes available on the site. Typical use-case is an edge cloud, where usually multiple, high-available, NAS-including cloud infrastructures -or VIMs- need to be brought-up on only a handful of physical nodes (1-10).
Unless isolation can be guaranteed, the resource manager components of both infrastructures will inevitably contest for the same resources.
The different managers of more mature cloud infrastructures -for example Openstack- can be already configured to manage only a subset of a Nodes' resource.
If Kubernetes would also support the same feature, operators would be able to 1: isolate a common workload CPU pool from the operating system and host processes (see user story 1), and 2: manually divide this pool between the different infrastructures however they see fit.

#### User Story 3
As a TelCo edge cloud operator, I would like the CPUs of my latency sensitive applications to be managed by "out-of-tree" CPU managers

Basically the culmination of the previous two use-cases. The CPU manager can be called e.g. Nova if the latency sensitive application is running in VMs, in which case use-case 3 equals use-case 2.
But it could be also called CMK, or CPU-Pooler in case the latency sensitive application is running on top of Kubernetes.
Kubelet's current CPU Manager is not used in neither of the cases.
Note: the user story is taken -with some paraphrasing- from real life RFPs

### Implementation Details/Notes/Constraints

Kubernetes already contains code to remove a couple of CPU cores from the cpusets ("pools") it manages.
The three possible enhancements to be considered are the following:
- how to provide the possibility to optionally configure an exact cpuset as system-reserved
- how the exclusion should inter-work with other, similar features
- how to extend the scope of the exclusion to non-static policy use-cases too

#### Configuration
To avoid backward incompatible configuration API changes the KEP proposes to introduce a new configuration flag, called "reserved-cpus".
This setting should be a Node-level / Kubelet option, and should be configurable independently of all other flags.

This means inter-working with the existing CPU reservation mechanism for system processes has to be fully fleshed out, most importantly, with "system-reserved=cpu".
Proposal is to let both of the flags live at the same time. When both are set "reserved-cpus" simply takes priority.

#### Inter-working with other features
As Kubelet already has multiple other features related to lowering a Node's allocatable CPU capacity, it shall be defined how the additional Node capacity reduction due to the newly introduced flag should work exactly.
The idea is that whenever the Allocatable capacity of a Node is being calculated, we shall always first exclude the CPU cores listed in reserved-cpus from total capacity.
This will result in the total CPU pool usable by Kubernetes. All other allocatable-decreasing, or safety-margin leaving feature shall take this adjusted capacity as its baseline when doing further calculations.
An example of such an inter-working would be defining "kube-reserved", and "reserved-cpus" at the same time.
In this case for example with "reserved-cpus" set to "1,9", and "kube-reserved" is set to "cpu=500m" on a 16 core system with hyper-threading disabled the allocatable capacity should be:
16X1000m - len(1,9)X1000m - 500m = 14.500m.
At the same time no cpuset.cpus cgroup hierarchy parameter of any container created by Kubelet should include core IDs 1 and 9.

#### Proper cpuset configuration for non-static policy use-cases
This might be the biggest change, as the cpuset.cpus is currently only actively adjusted by the CPU Manager component.
Taking "reserved-cpus" into account whenever a container is created by Kubelet might require changes in the CRI, and dockershim implementations ("if RESERVED_CPUS then CONTAINER.CPUSET = TOTAL_CPUS-RESERVED_CPUS").

### Risks and Mitigations

As the outlined implementation concept is entirely backward compatible, no special risks are foreseen with the introduction of this functionality.

The feature itself could be seen as some kind of mitigation of a larger, more complex issue. If CPU manager would support the existence of sub-node level, explicitly configured and differently fine-tuned CPU pools; this feature might not even be required.
This idea was discussed a couple times in the past, but was put on hold due to the many risks it would have raised on the Kubernetes ecosystem.

Cloud infrastructure operators would be able to achieve their functional requirements even with the current CPU management architecture by making at least Kubelet's already existing pool(s) configurable.
Such a quick fix could ultimately give way to the introduction of configurable sub-node CPU pools -if community would decide so-, but that is a KEP for another day.

## Design Details
### Graduation Criteria
The feature can, and should be implemented in multiple iterations.

The first possible chunk could be introducing the new flag, and applying its effect to CPU Manager's static policy. This change would be inline with the current meaning of the "system-reserved" feature, therefore would be easy to implement.
So easy that it is basically already done:
https://github.com/kubernetes/kubernetes/pull/83592
This could constitute as the "alpha" release for the feature.

In the next step the reach of the flag could be extended to cover the cases when CPU Manager is turned off, or its management policy is set to "none".
In the scope of this phase CRI / shim implementations might require some minor tweaks.
This could be considered as the beta state of the feature.

In order to call the feature GA, alignment with all the existing allocatable capacity manipulating, and displaying features should be done irrespective of whether CPU manager is in use, and with what policy.
Including but not limited to kube-reserved, and eviction related functionalities etc.
The implementation should also ensure that always the proper amount of "allocatable" CPU resources are returned to the callers (CLI, REST, K8s scheduler etc.)

### Upgrade / Downgrade Strategy

Implementing the feature in a backward compatible manner as proposed we can ensure no special upgrade / downgrade strategy is required.
