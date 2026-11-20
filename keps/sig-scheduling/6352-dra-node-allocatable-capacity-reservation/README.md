# KEP-6352: DRA Node Allocatable Capacity Reservation

<!-- toc -->
- [Release Signoff Checklist](#release-signoff-checklist)
- [Summary](#summary)
- [Motivation](#motivation)
  - [Goals](#goals)
  - [Non-Goals](#non-goals)
- [Proposal](#proposal)
  - [User Stories](#user-stories)
  - [Risks and Mitigations](#risks-and-mitigations)
- [Design Details](#design-details)
  - [API](#api)
    - [API Validation](#api-validation)
  - [Kube-Scheduler Accounting](#kube-scheduler-accounting)
  - [Kubelet](#kubelet)
  - [Future Enhancements](#future-enhancements)
    - [Additional Aggregation Modes](#additional-aggregation-modes)
  - [Test Plan](#test-plan)
      - [Prerequisite testing updates](#prerequisite-testing-updates)
      - [Unit tests](#unit-tests)
      - [Integration tests](#integration-tests)
      - [e2e tests](#e2e-tests)
  - [Graduation Criteria](#graduation-criteria)
    - [Alpha](#alpha)
    - [Beta](#beta)
    - [GA](#ga)
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

- [ ] (R) Enhancement issue in release milestone, which links to KEP dir in [kubernetes/enhancements] (not the initial KEP PR)
- [ ] (R) KEP approvers have approved the KEP status as `implementable`
- [ ] (R) Design details are appropriately documented
- [ ] (R) Test plan is in place, giving consideration to SIG Architecture and SIG Testing input (including test refactors)
  - [ ] e2e Tests for all Beta API Operations (endpoints)
  - [ ] (R) Ensure GA e2e tests meet requirements for [Conformance Tests](https://github.com/kubernetes/community/blob/master/contributors/devel/sig-architecture/conformance-tests.md)
  - [ ] (R) Minimum Two Week Window for GA e2e tests to prove flake free
- [ ] (R) Graduation criteria is in place
  - [ ] (R) [all GA Endpoints](https://github.com/kubernetes/community/pull/1806) must be hit by [Conformance Tests](https://github.com/kubernetes/community/blob/master/contributors/devel/sig-architecture/conformance-tests.md) within one minor version of promotion to GA
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

This KEP adds a `reservation` field to a device's `nodeAllocatableResources` entry, alongside the `mapping`
and `overhead` fields introduced by [KEP-5517](https://github.com/kubernetes/enhancements/issues/5517). This
serves use cases where the DRA driver pod has to run a helper daemon process to manage an allocated device.
The scheduler subtracts that reserved amount from the node's allocatable when the device is first allocated,
and releases it when the last allocation goes away. The reserved capacity does not belong to the workload pod that 
references a claim for the device.

It serves a similar purpose as the kubelet's
[`--system-reserved`](https://kubernetes.io/docs/tasks/administer-cluster/reserve-compute-resources/) flag,
but is dynamic: the capacity is held only while the devices that need it are allocated, and the driver
declares the amount rather than the cluster administrator.

## Motivation

KEP-5517 makes a DRA allocation count against node allocatable for the pod that references the claim: the
footprint is written to the pod's status, enforced in the pod's cgroups, and counted in eviction, scoring, and
quota.

Driver-side helper processes do not fit that model. Their cost is not attributable to any single referencing
pod, because one process serves all of them, and it is incurred once per device rather than once per pod. A
driver today either runs the process with no requests, so the scheduler never sees it, or puts it in a pod the
scheduler places only after the workload is already bound, or duplicates it into every workload pod. The user
stories below show each case.

Two mechanisms already exist and neither covers this case. The `overhead.perPod` and `overhead.perContainer`
fields from KEP-5517 scale with the number of references, so they over-reserve for a process that is shared.
The kubelet's `--system-reserved` is static: it holds the capacity on every node whether or not any device
that needs it is allocated, and it requires the cluster administrator to know each driver's footprint in
advance.

### Goals

* To let a device declare the node allocatable capacity that the driver-side process serving it needs.
* To reserve that capacity once per active device, independent of the number of claims, allocations, pods, or
  containers using it.
* To hold the reservation only while the device has at least one active allocation, and release it when the
  last allocation goes away.
* To keep the reservation off the workload pod, leaving its cgroups, QoS class, and OOM score adjustment
  unchanged.

### Non-Goals

* Enforcing the reservation on the node. Kubelet does not read this field and does not update the driver pod's
  cgroups dynamically. Drivers remain responsible for keeping their helper processes within their declared
  limit.
* Replacing the kubelet's `--system-reserved` for capacity that is not tied to a device allocation.

## Proposal

A DRA driver declares, on each device it publishes, the node capacity that its helper process needs for that
device. The scheduler holds that capacity while the device is in use, which is while at least one claim
allocation for it exists. Because it is keyed on the device rather than on the pods, the amount held does not
change as pods that share the device come and go. The capacity comes out of the node's allocatable rather than
being charged to a pod, so no workload pod's requests, cgroups, status, or QoS class change.

### User Stories

**Story 1 (Shared filesystem mount):** Multiple pods on a node read the same storage bucket. One FUSE process
can serve all of them, and it needs roughly 1Gi of memory regardless of how many pods are reading. There are
two ways to run that process today, and each fails differently.

1. As a **sidecar** in every workload pod, like the
   [Cloud Storage FUSE CSI driver](https://github.com/GoogleCloudPlatform/gcs-fuse-csi-driver). The requests
   are part of the workload pod, so the accounting is correct, but the sharing is lost: the node runs one
   process and one cache per pod.
2. As a **separate mount pod** on the node, like the
   [JuiceFS CSI driver](https://github.com/juicedata/juicefs-csi-driver). One process serves every pod on the
   node, but the CSI controller creates that pod only after the first workload pod is already bound, so the
   scheduler chose the node without knowing the mount pod had to fit there too. On a node that is already
   full, the mount pod stays `Pending` and the workload it serves hangs in `ContainerCreating`, or it preempts
   the very workload it exists to serve
   ([juicefs-csi-driver#607](https://github.com/juicedata/juicefs-csi-driver/issues/607)).

With this KEP a driver that models each (volume, node) pair as a device declares the 1Gi on it. The scheduler
subtracts it when the first pod's claim is allocated, so the room is committed when the node is chosen. Pods
that join later add nothing, because the reservation is keyed on the device. Nodes where no pod reads the
bucket reserve nothing.

**Story 2 (Shared GPU with an MPS control daemon):** A cluster shares one physical GPU across several pods
using CUDA MPS. An `nvidia-cuda-mps-control` daemon has to run on the node while any MPS claim on that GPU is
active. The NVIDIA DRA driver currently runs a single-replica Deployment with `spec.nodeName` set directly to
the target node, and no `resources` block. Setting `nodeName` means the pod never goes through the scheduler,
and the empty `resources` block means there is nothing for the scheduler to account for even if it did. But
this does not always work, for example
[dra-driver-nvidia-gpu#1283](https://github.com/kubernetes-sigs/dra-driver-nvidia-gpu/issues/1283).

With this KEP the GPU device declares the daemon's footprint. The first pod to claim `gpu-0` causes the
scheduler to deduct that pod's own requests plus the daemon's. A second pod sharing `gpu-0` deducts only its
own requests, because the reservation is already held. When the last MPS claim on `gpu-0` goes away, the driver
tears the daemon down and the scheduler releases the reservation.

### Risks and Mitigations

* A driver declares more than its helper process actually uses, and the node strands that capacity. The amount
  is per device and held only while the device is allocated, so the exposure is bounded by the devices in use.
  Sizing guidance for driver authors is documented with the field.
* Nothing on the node enforces the reservation, so an unrelated pod can occupy the capacity at runtime.
  Closing that needs a contract between kubelet and DRA drivers for dynamic allocatable adjustment, or a
  dedicated cgroup slice for driver-managed processes, both out of scope for alpha. The scheduler no longer
  overcommits the node, which is the case this KEP targets.
* A driver leaks a helper process after the device is deallocated. The scheduler releases the reservation
  based on allocation state, so the leaked process becomes unaccounted capacity. Drivers need a startup
  reconciliation that terminates orphaned processes.
* An administrator who already budgets the driver's processes into the kubelet's `--system-reserved` must not
  let the driver also declare the field. The budget must live in exactly one place; declaring both withholds
  the same capacity twice while devices are allocated.

## Design Details

### API

```go
type NodeAllocatableResource struct {
	// Mapping ... (KEP-5517)
	Mapping *NodeAllocatableMapping `json:"mapping,omitempty" protobuf:"bytes,3,opt,name=mapping"`

	// Overhead ... (KEP-5517)
	Overhead *NodeAllocatableOverhead `json:"overhead,omitempty" protobuf:"bytes,4,opt,name=overhead"`

	// Reservation is node capacity held for the driver-side process that serves this device
	// while it is allocated, such as a GPU MPS control daemon or a shared filesystem mounter.
	// It is like the kubelet's system-reserved
	// (see https://kubernetes.io/docs/tasks/administer-cluster/reserve-compute-resources/),
	// except that it is only needed, and only held, while the device has an active allocation.
	// Unlike Overhead, it is attributed to the node and never to a workload pod: it is not
	// recorded in any pod's status and does not change any pod's cgroups, QoS class, or OOM
	// score adjustment.
	// The reservation is separate from the spec requests of the driver's infrastructure pod and
	// must not be repeated there, and capacity that a workload pod consumes through its own
	// requests must not be declared here. Capacity the administrator already withholds through
	// the kubelet's --system-reserved budget must not be declared here either. Any of these
	// would count the same capacity twice.
	// Reservation is always subtracted from the node's general allocatable capacity for the
	// resource, even when mapping is specified for the same resource.
	// +optional
	// +k8s:optional
	// +featureGate=DRANodeAllocatableReservation
	Reservation *NodeAllocatableReservation `json:"reservation,omitempty" protobuf:"bytes,5,opt,name=reservation"`
}

// NodeAllocatableReservation describes node capacity reserved for the driver-side process
// serving allocated devices. Its members differ in how the reservation aggregates across the
// active devices on a node. Exactly one member must be set.
type NodeAllocatableReservation struct {
	// PerDevice is reserved once for each active device that declares it, so the node total is
	// the sum over active devices. It is reserved exactly once per device: a device shared
	// through multiple allocations (allowMultipleAllocations) or consumed as capacity shares
	// still reserves this amount once, not once per allocation or per share.
	// Use this when the driver runs one process per device.
	// +required
	// +k8s:required
	PerDevice *resource.Quantity `json:"perDevice,omitempty" protobuf:"bytes,1,opt,name=perDevice"`
}
```

`NodeAllocatableReservation` struct has a single member in alpha. Modelling it as a struct so that more aggregation modes
can be introduced without an API break; see [Additional Aggregation Modes](#additional-aggregation-modes). 

#### API Validation

*   Exactly one member of `reservation` must be set (`perDevice` in alpha); an empty `reservation` is rejected.
*   The quantity must be non-negative.
*   KEP-5517's rule that at least one of `mapping` or `overhead` must be set on a `nodeAllocatableResources`
    entry is extended to: at least one of `mapping`, `overhead`, or `reservation` must be set.
*   `reservation` may only be declared on node-local devices: a slice with `spec.nodeName` set, or a device with
    its own node name under per-device node selection. It is rejected on devices published with `allNodes` or a
    node selector, because such a device has no single node to charge.
*   The enclosing `nodeAllocatableResources` map keys are restricted by KEP-5517 to `cpu`, `memory`,
    `hugepages-<size>`, and `ephemeral-storage`. This field inherits that restriction.

### Kube-Scheduler Accounting

The scheduler deducts `perDevice` from the node's allocatable exactly once per active device, irrespective of
the number of pods, containers, or claims using that device, and even when the device models consumable
capacity. The node total is the sum over active devices.

*   The value is never written to `pod.status.nodeAllocatableResourceClaimStatuses`, because it does not belong
    to any pod. The scheduler stores no reservation state: a device is reserved while at least one claim
    allocation for it exists, and the reservation ends when the last allocation goes away.
*   The two transitions are observed differently, on purpose. A device becoming active is visible to the very
    next scheduling cycle, even while the claim status write is still in flight, because promising the same
    capacity twice would overcommit the node. A device becoming inactive is observed through the API:
    deallocating or deleting a claim, or a `ResourceSlice` change that removes the declaration, are changes the
    scheduler watches, and the reservation stops being derived on the next look. Watch delay on the release side
    only holds capacity a little longer; it never promises it early. The same changes requeue the pods that the
    reservation had blocked; the queueing hints skip events that cannot release reserved capacity.
*   With consumable capacity ([KEP-5075](https://github.com/kubernetes/enhancements/issues/5075)), the
    reservation is taken once on first allocation, while each pod's fractional capacity draw-down is tracked
    separately in the claim's allocation result.
*   An allocation made for a pod that is still binding is visible to the next pod, so the device is counted as
    already active and the reservation is never taken twice. If the binding fails, the allocation is discarded
    and the reservation stops being derived.
*   The reservation comes off the node's general allocatable capacity even when `mapping` is set for the same
    resource, rather than out of a per-socket or per-pool capacity pool named by that mapping.
*   The accounting is recomputed from the allocation state in each scheduling cycle. The only thing cached
    across cycles is a small index of devices that declare the field, refreshed when a `ResourceSlice` changes.
*   Because the reservation is never recorded in any pod's status, it is never charged to any namespace's
    resource quota; it is node capacity, not pod usage.

### Kubelet

Kubelet does not read this field and takes no action on it. This is the structural difference from KEP-5517,
where every other node allocatable contribution reaches the node through pod status.

*   The value is never written to `pod.status.nodeAllocatableResourceClaimStatuses`, so it changes no
    pod-level or container-level cgroup value and no OOM score adjustment. Workload containers are not
    throttled, killed, or ranked based on capacity that belongs to a shared driver process.
*   Because the capacity is not part of any pod's cgroup, a pod exiting does not reclaim it. The reservation
    is held while any allocated claim references the device, and the capacity returns to the node once no
    such claim remains, that is, every one of them has been deallocated or deleted.

**Note:** The driver pod's limits must cover its base request from the pod spec plus the target
reservation total its devices can enable. Since limits are not reserved capacity, sizing them for the maximum 
does not affect scheduling, but prevents the daemon from being killed by its own container limit. 
Reserving capacity through the field does not increase the driver pod's requests; it only ensures the node is not overcommitted while the daemon runs. It also does not guarantee any additional protection against kubelet eviction during node pressure, 
so the driver pod should use appropriate priority for better protection against eviction.

### Future Enhancements

#### Additional Aggregation Modes

`perDevice` covers a driver that runs one process per device. A driver that cannot share one process across
allocations of the same device needs `perAllocation` which is reserved once per claim allocation. 
This is useful when allocations carry configuration that a single process cannot serve at the same time, such as different mount
options or credentials.

### Test Plan

[X] I/we understand the owners of the involved components may require updates to
existing tests to make this code solid enough prior to committing the changes necessary
to implement this enhancement.

##### Prerequisite testing updates

None. The accounting extends paths already covered by KEP-5517's test suite.

##### Unit tests

Unit tests will be added for all new and modified logic in `kube-apiserver` and `kube-scheduler`.

-   API validation and registry strategy:
    -   Ensuring the new `reservation` field is validated correctly, covering the allowed resource keys and
        non-negative quantities.
    -   Verifying the field is dropped on create and preserved on update while the feature gate is disabled.
-   Scheduler Plugin Logic (`DynamicResources`):
    -   Verify the reservation is deducted once for a device with multiple allocations, once for a device
        consumed as capacity shares, and summed across distinct active devices.
    -   Verify the reservation is released once no allocated claim references the device.
    -   Verify the value is never written to `pod.status.nodeAllocatableResourceClaimStatuses`.
-   Scheduler Framework:
    -   Verify a pod with no claims does not fit into capacity held by a reservation.
    -   Verify a held reservation is not treated as capacity that preempting a victim would free.

##### Integration tests

Integration tests will be added in `test/integration/dra`:

-   The first pod claiming a device with a declared reservation is placed with the reservation deducted, and a
    second pod sharing the device deducts only its own requests.
-   Two active devices each declaring a reservation reserve the sum of both.
-   Deallocating or deleting the last claim allocation for a device releases the reservation and requeues the
    pods it had blocked. Deleting the last consumer pod triggers this indirectly, once its claim is deallocated.
-   A pod that does not fit because of a reservation is unschedulable, with or without claims of its own, and
    becomes schedulable once the reservation is released.
-   With the feature gate disabled, the field has no effect on placement.

##### e2e tests

E2E tests will be added to `test/e2e/dra`:

-   Verify that the pod remains pending if the node cannot fit the pod's footprint including DRA node allocatable reservation.
-   Verify no workload pod's cgroups, status, or QoS class are affected.

### Graduation Criteria

#### Alpha
-   Feature implemented behind the `DRANodeAllocatableReservation` feature gate and disabled by default.
-   API changes for `NodeAllocatableReservation` introduced, with validation and feature gate handling.
-   Kube-Scheduler:
    *   The `DynamicResources` plugin deducts the reservation from node allocatable once per active device.
    *   The reservation is released once no allocated claim references the device.
-   All unit and integration tests outlined in the Test Plan are implemented and verified.

#### Beta
-   Gather feedback from alpha.
-   At least one DRA driver has integrated the field and validated it in a running cluster.

#### GA

-   The feature has been enabled by default for at least two releases with no critical bug reports.

### Upgrade / Downgrade Strategy

-   **Upgrade:** Enabling the feature gate on an existing cluster is safe. Devices that declare the field begin
    reserving capacity on the next scheduling cycle. Already running pods are unaffected. A node can be
    overcommitted relative to the new accounting until its pods turn over.

-   **Downgrade:** Disabling the feature gate requires a kube-scheduler restart. The scheduler stops deducting
    reservations, so its view of node capacity becomes optimistic and nodes running driver processes can be
    oversubscribed. This is the same state as before the feature existed. No pod is disrupted and no node-level
    state has to be unwound, because kubelet never acted on the field. Disabling the gate on kube-apiserver
    drops the field from newly created `ResourceSlice` objects and preserves it on updates to objects that
    already carry it.

### Version Skew Strategy

The field is read only by kube-scheduler and validated by kube-apiserver. Kubelet is not involved, so there is
no node skew consideration. An older scheduler will not understand the field. If `ResourceSlice` objects contain it, 
it is ignored, which is the pre-feature behavior for the pods that scheduler places.

## Production Readiness Review Questionnaire

### Feature Enablement and Rollback

<!--
This section must be completed when targeting alpha to a release.
-->

###### How can this feature be enabled / disabled in a live cluster?

- [X] Feature gate (also fill in values in `kep.yaml`)
  - Feature gate name: `DRANodeAllocatableReservation`
  - Components depending on the feature gate: kube-scheduler, kube-apiserver

This gate depends on `DRANodeAllocatableResources` from
[KEP-5517](https://github.com/kubernetes/enhancements/issues/5517). The field is a member of
`NodeAllocatableResource`, which that gate controls, so with it disabled the apiserver drops the enclosing
`nodeAllocatableResources` entry and the reservation never reaches the scheduler. Enabling
`DRANodeAllocatableReservation` on its own has no effect. 

###### Does enabling the feature change any default behavior?

No. Behavior changes only for devices whose `ResourceSlice` declares the field. A cluster whose drivers do not
set it sees no difference.

###### Can the feature be disabled once it has been enabled (i.e. can we roll back the enablement)?

Yes. Disabling the feature gate and restarting kube-scheduler stops the reservations from being deducted.
Running workloads are not disrupted, because no node-level state is associated with the reservation.

###### What happens if we reenable the feature if it was previously rolled back?

The scheduler resumes deducting reservations for active devices on the next scheduling cycle. Because the
accounting is derived from current allocation state rather than persisted anywhere, no reconciliation is
required.

###### Are there any tests for feature enablement/disablement?

Yes. Unit tests cover API field handling under the gate, including writing an object with the field set and
then disabling the gate. Integration tests verify placement with the gate enabled and disabled.

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
-->

###### How can an operator determine if the feature is in use by workloads?

- `Device` entries in a `ResourceSlice` with `nodeAllocatableResources[*].reservation` set.
- `ResourceClaim` objects allocating one of those devices. A device is reserving capacity exactly while at
  least one allocated claim references it.

###### How can someone using this feature know that it is working for their instance?

<!--
For instance, if this is a pod-related feature, it should be possible to determine if the feature is functioning properly
for each individual pod.
Pick one more of these and delete the rest.
Please describe all items visible to end users below with sufficient detail so that they can verify correct enablement
and operation of this feature.
Recall that end users cannot usually observe component logs or access metrics.
-->

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

No external or cluster-level services. Within the control plane it depends on
[KEP-5517](https://github.com/kubernetes/enhancements/issues/5517): the field lives inside
`NodeAllocatableResource` and has no effect unless `DRANodeAllocatableResources` is enabled. A DRA driver has
to publish the field for anything to be reserved, but no driver is required for the cluster to function.

### Scalability

<!--
For alpha, this section is encouraged: reviewers should consider these questions
and attempt to answer them.

For beta, this section is required: reviewers must answer these questions.
-->

###### Will enabling / using this feature result in any new API calls?

No.

###### Will enabling / using this feature result in introducing new API types?

No. The this KEP proposes extensions to an existing type, but not a new type itself.

###### Will enabling / using this feature result in any new calls to the cloud provider?

No.

###### Will enabling / using this feature result in increasing size or count of the existing API objects?

Yes. With the API changes proposed in this KEP, `ResourceSlice` objects would have additional fields.

###### Will enabling / using this feature result in increasing time taken by any operations covered by existing SLIs/SLOs?

Yes. The time to schedule a pod would increase if it reference claims with DRA node allocatable reservation. 

###### Will enabling / using this feature result in non-negligible increase of resource usage (CPU, RAM, disk, IO, ...) in any components?

No.

###### Can enabling / using this feature result in resource exhaustion of some node resources (PIDs, sockets, inodes, etc.)?

No. 

### Troubleshooting

<!--
This section must be completed when targeting beta to a release.
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

