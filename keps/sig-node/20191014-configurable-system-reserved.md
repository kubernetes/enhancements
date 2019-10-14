---
title: Support configuring an exact cpuset as "system-reserved" in Kubelet
authors:
  - "@Levovar"
owning-sig: sig-node
participating-sigs:
  - sig-node
reviewers:
  - "@ConnorDoyle"
  - "@alicedoe"
approvers:
  - TBD
editor: TBD
creation-date: 2019-10-14
last-updated: 2019-10-14
status: provisional
see-also:
  - https://github.com/kubernetes/community/pull/2435
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
  - [User Stories [optional]](#user-stories-optional)
    - [Story 1](#story-1)
    - [Story 2](#story-2)
  - [Implementation Details/Notes/Constraints [optional]](#implementation-detailsnotesconstraints-optional)
  - [Risks and Mitigations](#risks-and-mitigations)
- [Design Details](#design-details)
  - [Test Plan](#test-plan)
  - [Graduation Criteria](#graduation-criteria)
    - [Examples](#examples)
      - [Alpha -&gt; Beta Graduation](#alpha---beta-graduation)
      - [Beta -&gt; GA Graduation](#beta---ga-graduation)
      - [Removing a deprecated flag](#removing-a-deprecated-flag)
  - [Upgrade / Downgrade Strategy](#upgrade--downgrade-strategy)
  - [Version Skew Strategy](#version-skew-strategy)
- [Implementation History](#implementation-history)
- [Drawbacks [optional]](#drawbacks-optional)
- [Alternatives [optional]](#alternatives-optional)
- [Infrastructure Needed [optional]](#infrastructure-needed-optional)
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
Kubelet supports configuring a subset of the Node resources it shall not manage via the "system-reserved" flag.
Currently only CPU, and memory are supported. 
The CPU Manager implemented within Kubelet interprets the absolute amount of "CPU shares" coming from this configuration as the "first X number of core IDs I shall not touch".
X denotes the smallest amount of vCPUs (threads, or physical cores) which are enough to satisfy the request.
I.e. for system-reserved=1200m CPU Manager excludes the addition of CPU cores 0, and 1 to the list of CPU cores usable by Kubernetes managed workloads.
This KEP proposes to enhance the format, and the meaning of this flag so that it becomes possible to configure an exact list of CPU cores CPU Manager shall exclude from the CPU pool(s) it manages.
The exclusion should apply the same way, irrespective of the configured CPU management policy (i.e. "none, or "static").

## Motivation
Kubelet's in-built CPU Manager always assumes that it is the primary software component managing the CPU cores of the host.
However, in certain infrastructures this might not always be the case.
While it is already possible to effectively take-away CPU cores from the Kubernetes managed workloads via the kube-reserved and system-reserved kubelet flags, this implicit way of declaring a Kubernetes managed CPU pool is not flexible enough to cover all use-cases.

To accomodate the use-cases related to seamlessly inter-working with multiple resource managers on the same node, the need arises to enhance existing CPU manager with a method of explicitly defining a discontinuous pool of CPUs it can manage.
Such feature could come in handy if one would like to:
- ensure proper resource accounting and separation between systemd processes and Kubernetes managed workloads
- ensure proper resource accounting and separation within a hybrid infrastructure (e.g. Openstack + Kubernetes resource managers running on the same node etc.)
- outsource the management of a subset of specialized, or optimized cores (e.g. real-time enabled CPUs, CPUs with different HT configuration etc.) to an external CPU manager without any (other) change in Kubelet's CPU manager

### Goals
The goal is to make any and all Kubernetes supported CPU management policies restrictable to an exactly defined subset of a Node's capacity to be able to ensure resources do not overlap between Kubernetes, and non-Kubernetes managed processes.

### Non-Goals
It is outside the scope of this KEP to restrict any other Kubernetes resource manager to a subset of a Node's resource group (like memory, devices, etc.).
It is also outside the scope of this KEP to enhance Kubelet's CPU manager itself with more fine-grained management policies.
The aim of this KEP is to continue to let Kubernetes manage some CPU cores however it sees fit, but at the same time also eliminate the possibility of accidentally scheduling workloads to restricted resources.
Lastly, while it would be an interesting research topic of how different CPU managers (one of them being Kubelet) could inter-work with each other in run-time to dynamically re-partition the CPU sets they manage, it is unfortunately also outside the scope of this simple KEP.
What this enhancement is trying to achieve first and foremost is isolation. Alignment of the isolated resources is left to the cloud infrastructure operators at this stage of the feature.

## Proposal

#### User Story 1 - As an infrastructure operator, I would like to exclusively dedicate some discontinuously numbered CPU cores to Linux services not supervised by Kubernetes

Kubelet already having system-reserved flag enforces the idea that resource management community already recognized this basic use-case to be valid in today's changing world.

Not every low-level infrastructure process was able to, or wanted to transform its architecture to a containerized, micro-service based deployment model.

Kubernetes resource management currently advocates physically separating these different services to different Nodes, but basically every cloud infrastructure runs some processes directly on the host.
It is not necessarily the case that these services are always restricted to the first X, continously numbered set of cores; especially on a multi-socket system.

This feature would give cloud administrators' a chance to be able to at least manually separate the CPU cores used by these processes; which in some cases might be mission critical (e.g. PTP synchronization, cloud high-availabilty etc.)

#### User Story 2 - As an infrastructure operator, I would like to run multiple cloud infrastructures in the same -edge- cloud

This user-story is actually very similar to the previous one, but concentrates on separating workloads. Imagine that an operator would like to run Openstack, VMware or any other popular cloud infrastructure next to Kubernetes, on the same set of Nodes.

Sometimes an operator simply does not have the possibility to separate her infrastructures on the host level, because simply there are not enough nodes available on the site. Typical use-case is an edge cloud, where usually multiple, high-available, NAS-including cloud infrastructures -or VIMs- need to be brought-up on only a handful of physical nodes (1-10).

Unless isolation can be uaranteed, the resource manager components of both infrastructures will inevitably contest for the same resources.

The different managers of more mature cloud infrastructures -for example Openstack- can already be configured to manage only a subset of a nodes' resource.
If Kubernetes would also support the same feature, operators would be able to 1: isolate a common CPU pool from the operating system to workloads (see user story 1), and 2: manually divide this pool between the different infrastructures however they see fit.

### Implementation Details/Notes/Constraints

The pure implementation of the feature described in this document would be a fairly simple one. Kubernetes already contains code to remove a couple of CPU cores from the domain of its CPU management policies.
The only enhancement needed to be done is to:
- provide the possibility to otionally configure an exact cpuset as system-reserved, rather than an amont of shares
- remove the listed CPU cores from the list of the Node's allocatable CPU pool

The only tricky part is how to control the aforementioned functionality.
To avoid backward incompatible configuration changes it is porposed to introduce a new configuration flag, called system-reserved-cpuset; rather than changing the meaning of the existing.
This setting should be a Node-level setting, and work with any CPU management policy.

This means inter-working with the existing reservation mechanism has to be fully fleshed out.
Proposal is to subject the existing flag to the standard Kubernetes deprecation policy. During the deprecation process when both flags are defined system-reserved-cpuset simply takes preference.

### Risks and Mitigations

As the outlined implementation concept is entirely backward compatible, no special risks are foreseen with the introduction of this functionality.

The feature itself could be seen as some kind of mitigation of a larger, more complex issue. If CPU manager would support the existence of sub-node level, explicitly configured CPU pools; this feature might not even be required.
This idea was discussed multiple times, but was always put on hold by the community due to the many risks it would have raised on the Kubernetes ecosystem.

By making Kubelet's existing pool(s) configurable cloud infrastructure operators would still be able to achieve their functional requirements.
Such a quick fix could ultimately give way to the introduction of configurable sub-node CPU pools, but that is a KEP for another day.

##### Removing a deprecated flag

In the exact same release whenever the proposed flag is introduced, system-reserved would be marked as deprecated. 
Inline with the standard deprecation policy system-reserved flag would be removed two ajor Kubernetes releases later.

### Upgrade / Downgrade Strategy

Implementing the feature in a backward compatible manner, and together with the above described deprecation policy we can ensure no special upgrade / downgrade strategy is required.
