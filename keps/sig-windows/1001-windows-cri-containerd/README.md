# Supporting CRI-ContainerD on Windows

## Table of Contents

<!-- TOC -->
- [Release Signoff Checklist](#release-signoff-checklist)
- [Summary](#summary)
- [Motivation](#motivation)
  - [Goals](#goals)
  - [Non-Goals](#non-goals)
- [Proposal](#proposal)
  - [User Stories](#user-stories)
    - [Improving Kubernetes integration for Windows Server containers](#improving-kubernetes-integration-for-windows-server-containers)
    - [Improved isolation and compatibility between Windows pods using Hyper-V](#improved-isolation-and-compatibility-between-windows-pods-using-hyper-v)
    - [Improve Control over Memory &amp; CPU Resources with Hyper-V](#improve-control-over-memory--cpu-resources-with-hyper-v)
    - [Improved Storage Control with Hyper-V](#improved-storage-control-with-hyper-v)
    - [Enable runtime resizing of container resources](#enable-runtime-resizing-of-container-resources)
  - [Implementation Details/Notes/Constraints](#implementation-detailsnotesconstraints)
    - [Proposal: Use Runtimeclass Scheduler to simplify deployments based on OS version requirements](#proposal-use-runtimeclass-scheduler-to-simplify-deployments-based-on-os-version-requirements)
    - [Proposal: Standardize hypervisor annotations](#proposal-standardize-hypervisor-annotations)
  - [Dependencies](#dependencies)
      - [Windows Server 2019](#windows-server-2019)
      - [CRI-ContainerD](#cri-containerd)
      - [CNI: Flannel](#cni-flannel)
      - [CNI: Kubenet](#cni-kubenet)
      - [CNI: GCE](#cni-gce)
      - [Storage: in-tree AzureFile, AzureDisk, Google PD](#storage-in-tree-azurefile-azuredisk-google-pd)
      - [Storage: FlexVolume for iSCSI &amp; SMB](#storage-flexvolume-for-iscsi--smb)
  - [Risks and Mitigations](#risks-and-mitigations)
    - [CRI-ContainerD availability](#cri-containerd-availability)
- [Design Details](#design-details)
  - [Test Plan](#test-plan)
  - [Graduation Criteria](#graduation-criteria)
    - [Alpha release](#alpha-release)
    - [Alpha -&gt; Beta Graduation](#alpha---beta-graduation)
    - [Beta -&gt; GA Graduation](#beta---ga-graduation)
  - [Upgrade / Downgrade Strategy](#upgrade--downgrade-strategy)
  - [Version Skew Strategy](#version-skew-strategy)
- [Production Readiness Review Questionnaire](#production-readiness-review-questionnaire)
  - [Feature enablement and rollback](#feature-enablement-and-rollback)
  - [Rollout, Upgrade and Rollback Planning](#rollout-upgrade-and-rollback-planning)
  - [Monitoring requirements](#monitoring-requirements)
  - [Dependencies](#dependencies-1)
  - [Scalability](#scalability)
  - [Troubleshooting](#troubleshooting)
- [Implementation History](#implementation-history)
- [Alternatives](#alternatives)
  - [CRI-O](#cri-o)
- [Infrastructure Needed](#infrastructure-needed)
<!-- /TOC -->

## Release Signoff Checklist

**ACTION REQUIRED:** In order to merge code into a release, there must be an issue in [kubernetes/enhancements] referencing this KEP and targeting a release milestone **before [Enhancement Freeze](https://github.com/kubernetes/sig-release/tree/master/releases)
of the targeted release**.

For enhancements that make changes to code or processes/procedures in core Kubernetes i.e., [kubernetes/kubernetes], we require the following Release Signoff checklist to be completed.

Check these off as they are completed for the Release Team to track. These checklist items _must_ be updated for the enhancement to be released.

Items marked with (R) are required *prior to targeting to a milestone / release*.

- [x] (R) Enhancement issue in release milestone, which links to KEP dir in [kubernetes/enhancements] (not the initial KEP PR)
- [x] (R) KEP approvers have set the KEP status to `implementable`
- [x] (R) Design details are appropriately documented
- [x] Test plan is in place, giving consideration to SIG Architecture and SIG Testing input
- [x] (R) Graduation criteria is in place
- [ ] (R) Production readiness review completed
- [ ] Production readiness review approved
- [x] "Implementation History" section is up-to-date for milestone
- [ ] User-facing documentation has been created in [kubernetes/website], for publication to [kubernetes.io]
- [ ] Supporting documentation e.g., additional design documents, links to mailing list discussions/SIG meetings, relevant PRs/issues, release notes

**Note:** Any PRs to move a KEP to `implementable` or significant changes once it is marked `implementable` should be approved by each of the KEP approvers. If any of those approvers is no longer appropriate than changes to that list should be approved by the remaining approvers and/or the owning SIG (or SIG-arch for cross cutting KEPs).

**Note:** This checklist is iterative and should be reviewed and updated every time this enhancement is being considered for a milestone.

[kubernetes.io]: https://kubernetes.io/
[kubernetes/enhancements]: https://github.com/kubernetes/enhancements/issues
[kubernetes/kubernetes]: https://github.com/kubernetes/kubernetes
[kubernetes/website]: https://github.com/kubernetes/website

## Summary

The ContainerD maintainers have been working on CRI support which is stable on Linux and Windows support has been added to ContainerD 1.13.
Supporting CRI-ContainerD on Windows means users will be able to take advantage of the latest container platform improvements that shipped in Windows Server 2019 / 1809 and beyond.

## Motivation

Windows Server 2019 includes an updated host container service (HCS v2) that offers more control over how containers are managed. This can remove some limitations and improve some Kubernetes API compatibility. However, the current Docker EE 18.09 release has not been updated to work with the Windows HCSv2, only ContainerD has been migrated. Moving to CRI-ContainerD allows the Windows OS team and Kubernetes developers to focus on an interface designed to work with Kubernetes to improve compatibility and accelerate development.

Additionally, users could choose to run with only CRI-ContainerD instead of Docker EE if they wanted to reduce the install footprint or produce their own self-supported CRI-ContainerD builds.

### Goals

- Improve the matrix of Kubernetes features that can be supported on Windows
- Provide a path forward to implement Kubernetes-specific features that are not available in the Docker API today
- Align with `dockershim` deprecation timelines once they are defined

### Non-Goals

- Running Linux containers on Windows nodes. This would be addressed as a separate KEP since the use cases are different.
- Deprecating `dockershim`. This is out of scope for this KEP. The effort to migrate that code out of tree is in [KEP PR 866](https://github.com/kubernetes/enhancements/pull/866) and deprecation discussions will happen later.

## Proposal

### User Stories

#### Improving Kubernetes integration for Windows Server containers

Moving to the new Windows HCSv2 platform and ContainerD would allow Kubernetes to add support for:

- Mounting single files, not just folders, into containers
- Termination messages (depends on single file mounts)
- /etc/hosts (c:\windows\system32\drivers\etc\hosts) file mapping

#### Improved isolation and compatibility between Windows pods using Hyper-V 

Hyper-V enables each pod to run within it’s own hypervisor partition, with a separate kernel. This means that we can build forward-compatibility for containers across Windows OS versions - for example a container built using Windows Server 1809, could be run on a node running Windows Server 1903. This pod would use the Windows Server 1809 kernel to preserve full compatibility, and other pods could run using either a shared kernel with the node, or their own isolated Windows Server 1903 kernels. Containers requiring 1809 and 1903 (or later) cannot be mixed in the same pod, they must be deployed in separate pods so the matching kernel may be used. Running Windows Server version 1903 containers on a Windows Server 2019/1809 host will not work.

In addition, some customers may desire hypervisor-based isolation as an additional line of defense against a container break-out attack.

Adding Hyper-V support would use [RuntimeClass](https://kubernetes.io/docs/concepts/containers/runtime-class/#runtime-class).
3 typical RuntimeClass names would be configured in CRI-ContainerD to support common deployments:

- runhcs-wcow-process [default] - process isolation is used, container & node OS version must match
- runhcs-wcow-hypervisor - Hyper-V isolation is used, Pod will be compatible with containers built with Windows Server 2019 / 1809. Physical memory overcommit is allowed with overages filled from pagefile.
- runhcs-wcow-hypervisor-1903 - Hyper-V isolation is used, Pod will be compatible with containers built with Windows Server 1903. Physical memory overcommit is allowed with overages filled from pagefile.

Using Hyper-V isolation does require some extra memory for the isolated kernel & system processes. This could be accounted for by implementing the [PodOverhead](https://kubernetes.io/docs/concepts/containers/runtime-class/#runtime-class) proposal for those runtime classes. We would include a recommended PodOverhead in the default CRDs, likely between 100-200M.

#### Improve Control over Memory & CPU Resources with Hyper-V

The Windows kernel itself cannot provide reserved memory for pods, containers or processes. They are always fulfilled using virtual allocations which could be paged out later. However, using a Hyper-V partition improves control over memory and CPU cores. Hyper-V can either allocate memory on-demand (while still enforcing a hard limit), or it can be reserved as a physical allocation up front. Physical allocations may be able to enable large page allocations within that range (to be confirmed) and improve cache coherency. CPU core counts may also be limited so a pod only has certain cores available, rather than shares spread across all cores, and applications can tune thread counts to the actually available cores.

Operators could deploy additional RuntimeClasses with more granular control for performance critical workloads:

- 2019-Hyper-V-Reserve: Hyper-V isolation is used, Pod will be compatible with containers built with Windows Server 2019 / 1809. Memory reserve == limit, and is guaranteed to not page out.
  - 2019-Hyper-V-Reserve-<N>Core: Same as above, except all but <N> CPU cores are masked out.
- 1903-Hyper-V-Reserve: Hyper-V isolation is used, Pod will be compatible with containers built with Windows Server 1903. Memory reserve == limit, and is guaranteed to not page out.
  - 1903-Hyper-V-Reserve-<N>Core: Same as above, except all but <N> CPU cores are masked out.

#### Improved Storage Control with Hyper-V

Hyper-V also brings the capability to attach storage to pods using block-based protocols (SCSI) instead of file-based protocols (host file mapping / NFS / SMB). These capabilities could be enabled in HCSv2 with CRI-ContainerD, so this could be an area of future work. Some examples could include:

Attaching a "physical disk" (such as a local SSD, iSCSI target, Azure Disk or Google Persistent Disk) directly to a pod. The kubelet would need to identify the disk beforehand, then attach it as the pod is created with CRI. It could then be formatted and used within the pod without being mounted or accessible on the host.

Creating [Persistent Local Volumes](https://kubernetes.io/docs/concepts/storage/volumes/#local) using a local virtual disk attached directly to a pod. This would create local, non-resilient storage that could be formatted from the pod without being mounted on the host. This could be used to build out more resource controls such as fixed disk sizes and QoS based on IOPs or throughput and take advantage of high speed local storage such as temporary SSDs offered by cloud providers.

#### Enable runtime resizing of container resources

With virtual-based allocations and Hyper-V, it should be possible to increase the limit for a running pod. This won’t give it a guaranteed allocation, but will allow it to grow without terminating and scheduling a new pod. This could be a path to vertical pod autoscaling. This still needs more investigation and is mentioned as a future possibility.

### Implementation Details/Notes/Constraints

The work needed will span multiple repos, SIG-Windows will be maintaining a [Windows CRI-Containerd Project Board] to track everything in one place.

#### Proposal: Use Runtimeclass Scheduler to simplify deployments based on OS version requirements

As of version 1.14, RuntimeClass is not considered by the Kubernetes scheduler. There’s no guarantee that a node can start a pod, and it could fail until it’s scheduled on an appropriate node. Additional node labels and nodeSelectors are required to avoid this problem. [RuntimeClass Scheduling](https://github.com/kubernetes/enhancements/blob/master/keps/sig-node/585-runtime-class/README.md) proposes being able to add nodeSelectors automatically when using a RuntimeClass, simplifying the deployment.

Windows forward compatibility will bring a new challenge as well because there are two ways a container could be run:

- Constrained to the OS version it was designed for, using process-based isolation
- Running on a newer OS version using Hyper-V.
This second case could be enabled with a RuntimeClass. If a separate RuntimeClass was used based on OS version, this means the scheduler could find a node with matching class.

#### Proposal: Standardize hypervisor annotations

There are large number of [Windows annotations](https://github.com/Microsoft/hcsshim/blob/master/internal/oci/uvm.go#L15) defined that can control how Hyper-V will configure its hypervisor partition for the pod. Today, these could be set in the runtimeclasses defined in the CRI-ContainerD configuration file on the node, but it would be easier to maintain them if key settings around resources (cpu+memory+storage) could be aligned across multiple hypervisors and exposed in CRI.

Doing this would make pod definitions more portable between different isolation types. It would also avoid the need for a "t-shirt size" list of RuntimeClass instances to choose from:

- 1809-Hyper-V-Reserve-2Core-PhysicalMemory
- 1903-Hyper-V-Reserve-1Core-VirtualMemory
- 1903-Hyper-V-Reserve-4Core-PhysicalMemory
- etc.

### Dependencies

##### Windows Server 2019

This work would be carried out and tested using the already-released Windows Server 2019. That will enable customers a migration path from Docker 18.09 to CRI-ContainerD if they want to get this new functionality. Windows Server 1903 and later will also be supported once they’re tested.

##### CRI-ContainerD

It was announced that the upcoming 1.3 release would include Windows support, but that release and timeline are still in planning as of early April 2019.

The code needed to run ContainerD is merged, and [experimental support in moby](https://github.com/moby/moby/pull/38541) has merged. CRI is in the process of being updated, and open issues are tracked on the [Windows CRI-Containerd Project Board]

The CRI plugin changes needed to enable Hyper-V isolation are still in a development branch [jterry75/cri](https://github.com/jterry75/cri/tree/windows_port/cmd/containerd) and don’t have an upstream PR open yet.

Code: mostly done
CI+CD: lacking

##### CNI: Flannel 
Flannel isn’t expected to require any changes since the Windows-specific metaplugins ship outside of the main repo. However, there is still not a stable release supporting Windows so it needs to be built from source. Additionally, the Windows-specific metaplugins to support ContainerD are being developed in a new repo [Microsoft/windows-container-networking](https://github.com/Microsoft/windows-container-networking). It’s still TBD whether this code will be merged into [containernetworking/plugins](https://github.com/containernetworking/plugins/), or maintained in a separate repo.

- Sdnbridge - this works with host-gw mode, replaces win-bridge
- Sdnoverlay - this works with vxlan overlay mode, replaces win-overlay

Code: in progress
CI+CD: lacking

##### CNI: Kubenet

The same sdnbridge plugin should work with kubenet as well. If someone would like to use kubenet instead of flannel, that should be feasible.

##### CNI: GCE

GCE uses the win-bridge meta-plugin today for managing Windows network interfaces. This would also need to migrate to sdnbridge.

##### Storage: in-tree AzureFile, AzureDisk, Google PD

These are expected to work and the same tests will be run for both dockershim and CRI-ContainerD.

##### Storage: FlexVolume for iSCSI & SMB

These out-of-tree plugins are expected to work, and are not tested in prow jobs today. If they graduate to stable we’ll add them to testgrid.

### Risks and Mitigations

#### CRI-ContainerD availability

As mentioned earlier, builds are not yet available. We will publish the setup steps required to build & test in the kubernetes-sigs/windows-testing repo during the course of alpha so testing can commence.

## Design Details

### Test Plan

The existing test cases running on Testgrid that cover Windows Server 2019 with Docker will be reused with CRI-ContainerD. Testgrid will include results for both ContainerD and dockershim.

- TestGrid: SIG-Windows: [flannel-l2bridge-windows-master](https://testgrid.k8s.io/sig-windows#flannel-l2bridge-windows-master) - this uses dockershim
- TestGrid: SIG-Windows: [containerd-l2bridge-windows-master](https://testgrid.k8s.io/sig-windows#containerd-l2bridge-windows-master) - this uses ContainerD

Test cases that depend on ContainerD and won't pass with Dockershim will be marked with `[feature:windows-containerd]` until `dockershim` is deprecated.

### Graduation Criteria

#### Alpha release

> Released with 1.18

- Windows Server 2019 containers can run with process level isolation using containerd
- TestGrid has results for Kubernetes master branch. CRI-ContainerD and CNI built from source and may include non-upstream PRs.

#### Alpha -> Beta Graduation

> Proposed for 1.19 or later

- Feature parity with dockershim, including:
  - Group Managed Service Account support
  - Named pipe & Unix domain socket mounts
- Support RuntimeClass to enable Hyper-V isolation
- Publicly available builds (beta or better) of CRI-ContainerD, at least one CNI
- TestGrid results for above builds with Kubernetes master branch

#### Beta -> GA Graduation

> Proposed for 1.20 or later

- Stable release of CRI-ContainerD on Windows, at least one CNI
- Master & release branches on TestGrid
- Perf analysis of pod-lifecycle operations performed and guidance around resource reservations and/or limits is updated for Windows node configuration and pod scheduling

### Upgrade / Downgrade Strategy

Because no Kubernetes API changes are expected, there is no planned upgrade/downgrade testing at the cluster level.

Node upgrade/downgrade is currently out of scope of the Kubernetes project, but we'll aim to include CRI-ContainerD in other efforts such as `kubeadm` bootstrapping for nodes.

As discussed in SIG-Node, there's also no testing on switching CRI on an existing node. These are expected to be installed and configured as a prerequisite before joining a node to the cluster.

### Version Skew Strategy

There's no version skew considerations needed for the same reasons described in upgrade/downgrade strategy.

## Production Readiness Review Questionnaire

### Feature enablement and rollback

_This section must be completed when targeting alpha to a release._

- **How can this feature be enabled / disabled in a live cluster?**
  - [ ] Feature gate (also fill in values in `kep.yaml`)
    - Feature gate name:
    - Components depending on the feature gate:
  - [x] Other
    - Describe the mechanism: Windows agent nodes are expected to have the CRI installed and configured before joining the node to a cluster.
    - Will enabling / disabling the feature require downtime of the control
      plane? No
    - Will enabling / disabling the feature require downtime or reprovisioning
      of a node? Yes

- **Does enabling the feature change any default behavior?**
  No

- **Can the feature be disabled once it has been enabled (i.e. can we rollback
  the enablement)?**
  No

- **What happens if we reenable the feature if it was previously rolled back?**
  This feature is not enabled/disabled like traditional features - nodes are configured with containerd prior to joining a cluster.
  A single Windows node will be configured with either Docker EE or containerd but different nodes running different CRIs can be joined to the same cluster with no negative impact.

- **Are there any tests for feature enablement/disablement?**
  No - As mentioned in [Upgrade / Downgrade Strategy](#upgrade--downgrade-strategy) there is no testing for switching CRI on an existing node.

### Rollout, Upgrade and Rollback Planning

_This section must be completed when targeting beta graduation to a release._

- **How can a rollout fail? Can it impact already running workloads?**
  Nodes with improperly configured containerd installations may result in the node never a schedule-able state, pod sandbox creation failures, or issues creating/starting containers.
  Ensuring proper configuration containerd installation/configuration is out-of-scope for this document.

- **What specific metrics should inform a rollback?**
  All existing node health metrics should be used to determine/monitor node health.

- **Were upgrade and rollback tested? Was upgrade->downgrade->upgrade path tested?**
  N/A

- **Is the rollout accompanied by any deprecations and/or removals of features,
  APIs, fields of API types, flags, etc.?**
  No

### Monitoring requirements

_This section must be completed when targeting beta graduation to a release._

- **How can an operator determine if the feature is in use by workloads?**
  The `status.nodeInfo.contaienrRuntimeVersion` property for a node indicates which CRI is being used for a node.

- **What are the SLIs (Service Level Indicators) an operator can use to
  determine the health of the service?**
  - [ ] Metrics
    - Metric name:
    - [Optional] Aggregation method:
    - Components exposing the metric:
  - [x] Other (treat as last resort)
    - Details: Checking the health of Windows node running containerd should be no different than checking the health of any other node in a cluster.

- **What are the reasonable SLOs (Service Level Objectives) for the above SLIs?**
  No

- **Are there any missing metrics that would be useful to have to improve
  observability if this feature?**
  No

### Dependencies

_This section must be completed when targeting beta graduation to a release._

- **Does this feature depend on any specific services running in the cluster?**
  Windows CRI-containerd does not add any additional dependencies/requirements for joining nodes to a cluster.

### Scalability

- **Will enabling / using this feature result in any new API calls?**
  No

- **Will enabling / using this feature result in introducing new API types?**
  No

- **Will enabling / using this feature result in any new calls to cloud
  provider?**
  No

- **Will enabling / using this feature result in increasing size or count
  of the existing API objects?**
  No

- **Will enabling / using this feature result in increasing time taken by any
  operations covered by [existing SLIs/SLOs][]?**
  No - But perf testing should be done to validate pod-lifecycle operations are not regressed compared to when equivalent nodes are configured with Docker EE.

- **Will enabling / using this feature result in non-negligible increase of
  resource usage (CPU, RAM, disk, IO, ...) in any components?**
  There are no expected increases in resource usage when using containerd - Additional perf testing will be done as prior of GA graduation.

### Troubleshooting

Troubleshooting section serves the `Playbook` role as of now. We may consider
splitting it into a dedicated `Playbook` document (potentially with some monitoring
details). For now we leave it here though.

_This section must be completed when targeting beta graduation to a release._

- **How does this feature react if the API server and/or etcd is unavailable?**

- **What are other known failure modes?**
  For each of them fill in the following information by copying the below template:
  - [Failure mode brief description]
    - Detection: How can it be detected via metrics? Stated another way:
      how can an operator troubleshoot without loogging into a master or worker node?
    - Mitigations: What can be done to stop the bleeding, especially for already
      running user workloads?
    - Diagnostics: What are the useful log messages and their required logging
      levels that could help debugging the issue?
      Not required until feature graduated to Beta.
    - Testing: Are there any tests for failure mode? If not describe why.

- **What steps should be taken if SLOs are not being met to determine the problem?**

[supported limits]: https://git.k8s.io/community//sig-scalability/configs-and-limits/thresholds.md
[existing SLIs/SLOs]: https://git.k8s.io/community/sig-scalability/slos/slos.md#kubernetes-slisslos

## Implementation History

- 2019-04-24 - KEP started, based on the [earlier doc shared SIG-Windows and SIG-Node](https://docs.google.com/document/d/1NigFz1nxI9XOi6sGblp_1m-rG9Ne6ELUrNO0V_TJqhI/edit)
- 2019-09-20 - Updated with new milestones
- 2020-01-21 - Updated with new milestones
- 2020-05-12 - Minor KEP updates, PRR questionnaire added

## Alternatives

### CRI-O

[CRI-O](https://cri-o.io/) is another runtime that aims to closely support all the fields available in the CRI spec. Currently there aren't any maintainers porting it to Windows so it's not a viable alternative.

## Infrastructure Needed

No new infrastructure is currently needed from the Kubernetes community. The existing test jobs using prow & testgrid will be copied and modified to test CRI-ContainerD in addition to dockershim.

[Windows CRI-Containerd Project Board]: https://github.com/orgs/kubernetes/projects/34