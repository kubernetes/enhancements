<!--
**Note:** When your KEP is complete, all of these comment blocks should be removed.

To get started with this template:

- [ ] **Pick a hosting SIG.**
  Make sure that the problem space is something the SIG is interested in taking
  up. KEPs should not be checked in without a sponsoring SIG.

- [ ] **Create an issue in kubernetes/enhancements**
  When filing an enhancement tracking issue, please make sure to complete all
  fields in that template. One of the fields asks for a link to the KEP. You
  can leave that blank until this KEP is filed, and then go back to the
  enhancement and add the link.

- [ ] **Make a copy of this template directory.**
  Copy this template into the owning SIG's directory and name it
  `NNNN-short-descriptive-title`, where `NNNN` is the issue number (with no
  leading-zero padding) assigned to your enhancement above.

- [ ] **Fill out as much of the kep.yaml file as you can.**
  At minimum, you should fill in the "Title", "Authors", "Owning-sig",
  "Status", and date-related fields.

- [ ] **Fill out this file as best you can.**
  At minimum, you should fill in the "Summary" and "Motivation" sections.
  These should be easy if you've preflighted the idea of the KEP with the
  appropriate SIG(s).

- [ ] **Create a PR for this KEP.**
  Assign it to people in the SIG who are sponsoring this process.

- [ ] **Merge early and iterate.**
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
# [KEP-3063](https://github.com/kubernetes/enhancements/issues/3063): Dynamic Resource Allocation with Control Plane Controller


<!-- toc -->
- [Release Signoff Checklist](#release-signoff-checklist)
- [Summary](#summary)
- [Motivation](#motivation)
  - [Goals](#goals)
  - [Non-Goals](#non-goals)
- [Proposal](#proposal)
  - [User Stories](#user-stories)
    - [Network-attached accelerator](#network-attached-accelerator)
    - [Combined setup of different hardware functions](#combined-setup-of-different-hardware-functions)
  - [Notes/Constraints/Caveats](#notesconstraintscaveats)
- [Design Details](#design-details)
  - [ResourceClass extension](#resourceclass-extension)
  - [ResourceClaim extension](#resourceclaim-extension)
  - [ResourceClaimStatus extension](#resourceclaimstatus-extension)
  - [ResourceHandle extensions](#resourcehandle-extensions)
  - [PodSchedulingContext](#podschedulingcontext)
  - [Coordinating resource allocation through the scheduler](#coordinating-resource-allocation-through-the-scheduler)
  - [Resource allocation and usage flow](#resource-allocation-and-usage-flow)
  - [Scheduled pods with unallocated or unreserved claims](#scheduled-pods-with-unallocated-or-unreserved-claims)
  - [Cluster Autoscaler](#cluster-autoscaler)
  - [Implementing a plugin for node resources](#implementing-a-plugin-for-node-resources)
  - [Implementing optional resources](#implementing-optional-resources)
  - [Test Plan](#test-plan)
      - [Prerequisite testing updates](#prerequisite-testing-updates)
      - [Unit tests](#unit-tests)
      - [Integration tests](#integration-tests)
      - [e2e tests](#e2e-tests)
  - [Graduation Criteria](#graduation-criteria)
    - [Alpha -&gt; Beta Graduation](#alpha---beta-graduation)
    - [Beta -&gt; GA Graduation](#beta---ga-graduation)
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
- [x] (R) KEP approvers have approved the KEP status as `implementable`
- [x] (R) Design details are appropriately documented
- [x] (R) Test plan is in place, giving consideration to SIG Architecture and SIG Testing input (including test refactors)
- [x] (R) Graduation criteria is in place
- [x] (R) Production readiness review completed
- [x] (R) Production readiness review approved
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

Originally, this KEP introduced DRA in Kubernetes 1.26 and the ["structured
parameters" KEP](../4381-dra-structured-parameters/README.md) added an
extension. Now the roles are reversed: #4381 defines the base functionality
and this KEP is an optional extension.

With #4381, DRA drivers are limited by what the structured parameter model(s)
defined by Kubernetes support. New requirements for future hardware may depend
on changing Kubernetes first.

With this KEP, parameters and resource availability are completely opaque
to Kubernetes. During scheduling of a pod, the kube-scheduler and any DRA
driver controller(s) handling claims for the pod communicate back-and-forth through the
apiserver by updating a `PodSchedulingContext` object, ultimately leading to the
allocation of all pending claims and the pod being scheduled onto a node.

Beware that this approach poses a problem for the [Cluster
Autoscaler](https://github.com/kubernetes/autoscaler) (CA) or for any higher
level controller that needs to make decisions for a group of pods (e.g. a job
scheduler). It cannot simulate the effect of allocating or deallocating
claims over time. Only the third-party DRA drivers have the information
available to do this. Structured parameters from #4381 should be used
when cluster autoscaling is needed.

## Motivation

### Goals

<!--
List the specific goals of the KEP. What is it trying to achieve? How will we
know that this has succeeded?
-->

* More flexibility beyond what is currently supported by structured parameters:
  * Arbitrary parameters
  * Network-attached resources
  * Custom policies for matching of resource requests with available resources,
    like handling of optional resource requests or application-specific
    policies
* Prototyping future extensions with a control plane controller before
  proposing them as Kubernetes enhancements for DRA with structured parameters

### Non-Goals

* Supporting cluster autoscaling

## Proposal

A resource driver handles all operations that are specific to the allocation
and deallocation of a ResourceClaim. It does that in coordination with the
scheduler (for allocation) and kube-controller-manager (for deallocation).

![components](./components.png)


### User Stories

#### Network-attached accelerator

As a data center operator, I want to place accelerators in a separate enclosure
where they can get connected to different compute nodes through [Compute
Express Link™ (CXL™)](https://www.computeexpresslink.org/). This is more
cost-effective because not all workloads need these accelerators and more
flexible because worker nodes can be added or removed independently from
accelerators.

Jobs then compete for these accelerators by requesting them through a
ResourceClaim. The kube-scheduler ensures that all required resources for a pod
are reserved for it (CPU, RAM, and the ResourceClaim) before scheduling the pod
onto a worker node. The resource driver then dynamically attaches the
accelerator and detaches it again when the pod completes.

#### Combined setup of different hardware functions

As a 5G telco operator, I want to use the FPGA IP block, signal processor and
network interfaces provided by the [Intel FPGA
N3000](https://www.intel.com/content/www/us/en/products/details/fpga/platforms/pac/n3000.html)
card in a Kubernetes edge cluster. Intel provides a single resource driver that
has parameters for setting up all of these hardware functions together as
needed for a data flow pipeline.

### Notes/Constraints/Caveats

Scheduling is likely to be slower when many Pods request the new
resource types, both because scheduling such a Pod involves more
round-trips through the API server for ResourceClaimStatus updates and
because scheduling one Pod may affect other Pods in ways that cannot
be anticipated by the kube-scheduler. When many Pods compete for
limited resources, multiple attempts may be needed before a suitable
node is found.

The hardware that is expected to need this more flexible allocation
approach is likely to be used by pods that run for extended periods of time,
so this is not a major concern.

## Design Details

### ResourceClass extension

An optional field in ResourceClass enables using the DRA driver's control
plane controller:

```go
type ResourceClass struct {
    ...

    // ControllerName defines the name of the dynamic resource driver that is
    // used for allocation of a ResourceClaim that uses this class. If empty,
    // structured parameters are used for allocating claims using this class.
    //
    // Resource drivers have a unique name in forward domain order
    // (acme.example.com).
    //
    // This is an alpha field and requires enabling the DRAControlPlaneController
    // feature gate.
    //
    // +optional
    ControllerName string
}
```

### ResourceClaim extension

With structured parameters, allocation always happens only when a pod needs a
ResourceClaim ("delayed allocation"). With allocation through the driver, it
may also make sense to allocate a ResourceClaim as soon as it gets created
("immediate allocation").

Immediate allocation is useful when allocating a resource is expensive (for
example, programming an FPGA) and the resource therefore is meant to be used by
multiple different Pods, either in parallel or one after the other. Another use
case is managing resource allocation in a third-party component which fully
understands optimal placement of everything that needs to run on a certain
cluster.

The downside is that Pod resource requirements cannot be considered when choosing
where to allocate. If a resource was allocated so that it is only available on
one node and the Pod cannot run there because other resources like RAM or CPU
are exhausted on that node, then the Pod cannot run elsewhere. The same applies
to resources that are available on a certain subset of the nodes and those
nodes are busy.

Different lifecycles of a ResourceClaim can be combined with different allocation modes
arbitrarily. Some combinations are more useful than others:

```
+-----------+----------------------------------------------------------------------+
|           |                             allocation mode                          |
| lifecycle |             immediate              |   delayed                       |
+-----------+------------------------------------+---------------------------------+
| regular   | starts the potentially             | avoids wasting resources        |
| claim     | slow allocation as soon            | while they are not needed yet   |
|           | as possible                        |                                 |
+-----------+------------------------------------+---------------------------------+
| claim     | same benefit as above,             | resource allocated when needed, |
| template  | but ignores other pod constraints  | allocation coordinated by       |
|           | during allocation                  | scheduler                       |
+-----------+------------------------------------+---------------------------------+
```

```
type ResourceClaimSpec struct {
    ...

    // Allocation can start immediately or when a Pod wants to use the
    // resource. "WaitForFirstConsumer" is the default.
    // +optional
    //
    // This is an alpha field and requires enabling the DRAControlPlaneController
    // feature gate.
    AllocationMode AllocationMode
}

// AllocationMode describes whether a ResourceClaim gets allocated immediately
// when it gets created (AllocationModeImmediate) or whether allocation is
// delayed until it is needed for a Pod
// (AllocationModeWaitForFirstConsumer). Other modes might get added in the
// future.
type AllocationMode string

const (
    // When a ResourceClaim has AllocationModeWaitForFirstConsumer, allocation is
    // delayed until a Pod gets scheduled that needs the ResourceClaim. The
    // scheduler will consider all resource requirements of that Pod and
    // trigger allocation for a node that fits the Pod.
    //
    // The ResourceClaim gets deallocated as soon as it is not in use anymore.
    AllocationModeWaitForFirstConsumer AllocationMode = "WaitForFirstConsumer"

    // When a ResourceClaim has AllocationModeImmediate and the ResourceClass
    // uses a control plane controller, allocation starts
    // as soon as the ResourceClaim gets created. This is done without
    // considering the needs of Pods that will use the ResourceClaim
    // because those Pods are not known yet.
    //
    // When structured parameters are used, nothing special is done for
    // allocation and thus allocation happens when the scheduler handles
    // first Pod which needs the ResourceClaim, as with "WaitForFirstConsumer".
    //
    // In both cases, claims remain allocated even when not in use.
    AllocationModeImmediate AllocationMode = "Immediate"
)
```

### ResourceClaimStatus extension

```
type ResourceClaimStatus struct {
    ...
    // ControllerName is a copy of the driver name from the ResourceClass at
    // the time when allocation started. It is empty when the claim was
    // allocated through structured parameters,
    //
    // This is an alpha field and requires enabling the DRAControlPlaneController
    // feature gate.
    //
    // +optional
    ControllerName string

    // DeallocationRequested indicates that a ResourceClaim is to be
    // deallocated.
    //
    // The driver then must deallocate this claim and reset the field
    // together with clearing the Allocation field.
    //
    // While DeallocationRequested is set, no new consumers may be added to
    // ReservedFor.
    //
    // This is an alpha field and requires enabling the DRAControlPlaneController
    // feature gate.
    //
    // +optional
    DeallocationRequested bool
```

DeallocationRequested gets set by the scheduler when it detects
that pod scheduling cannot proceed because some
claim was allocated for a node for which some other pending claims
cannot be allocated because that node ran out of resources for those.

It also gets set by kube-controller-manager when it detects that
a claim is no longer in use.

### ResourceHandle extensions

Resource drivers can use each `ResourceHandle` to store data directly or
cross-reference some other place where information is stored.
This data is guaranteed to be available when a Pod is about
to run on a node, in contrast to the ResourceClass which
may have been deleted in the meantime. It's also protected from
modification by a user, in contrast to an annotation.

```
// ResourceHandle holds opaque resource data for processing by a specific kubelet plugin.
type ResourceHandle struct {
    ...

    // Data contains the opaque data associated with this ResourceHandle. It is
    // set by the controller component of the resource driver whose name
    // matches the DriverName set in the ResourceClaimStatus this
    // ResourceHandle is embedded in. It is set at allocation time and is
    // intended for processing by the kubelet plugin whose name matches
    // the DriverName set in this ResourceHandle.
    //
    // The maximum size of this field is 16KiB. This may get increased in the
    // future, but not reduced.
    //
    // This is an alpha field and requires enabling the DRAControlPlaneController feature gate.
    //
    // +optional
    Data string
}

// ResourceHandleDataMaxSize represents the maximum size of resourceHandle.data.
const ResourceHandleDataMaxSize = 16 * 1024
```


### PodSchedulingContext

PodSchedulingContexts get created by a scheduler when it processes a pod which
uses one or more unallocated ResourceClaims with delayed allocation and
allocation of those ResourceClaims is handled by control plane controllers.

```
// PodSchedulingContext holds information that is needed to schedule
// a Pod with ResourceClaims that use "WaitForFirstConsumer" allocation
// mode.
//
// This is an alpha type and requires enabling the DynamicResourceAllocation
// and DRAControlPlaneController feature gates.
type PodSchedulingContext struct {
    metav1.TypeMeta
    // Standard object metadata
    // +optional
    metav1.ObjectMeta

    // Spec describes where resources for the Pod are needed.
    Spec PodSchedulingContextSpec

    // Status describes where resources for the Pod can be allocated.
    Status PodSchedulingContextStatus
}
```

The name of a PodSchedulingContext must be the same as the corresponding Pod.
That Pod must be listed as an owner in OwnerReferences to ensure that the
PodSchedulingContext gets deleted when no longer needed. Normally the scheduler
will delete it.

Drivers must ignore PodSchedulingContexts where the owning
pod already got deleted because such objects are orphaned
and will be removed soon.

```
// PodSchedulingContextSpec describes where resources for the Pod are needed.
type PodSchedulingContextSpec struct {
    // SelectedNode is the node for which allocation of ResourceClaims that
    // are referenced by the Pod and that use "WaitForFirstConsumer"
    // allocation is to be attempted.
    SelectedNode string

    // PotentialNodes lists nodes where the Pod might be able to run.
    //
    // The size of this field is limited to 128. This is large enough for
    // many clusters. Larger clusters may need more attempts to find a node
    // that suits all pending resources. This may get increased in the
    // future, but not reduced.
    // +optional
    PotentialNodes []string
}
```

When allocation is delayed, the scheduler must set
the `SelectedNode` for which it wants the resource(s) to be allocated
before the driver(s) start with allocation.
The scheduler also needs to decide on which node a Pod should run and will
ask the driver(s) on which nodes the resource might be
made available. To trigger that check, the scheduler
provides the names of nodes which might be suitable
for the Pod and will update that list periodically until
all resources are allocated.

The driver must ensure that the allocated resource
is available on this node or update ResourceSchedulingStatus.UnsuitableNodes
to indicate where allocation might succeed.

When allocation succeeds, drivers should immediately add
the pod to the ResourceClaimStatus.ReservedFor field
together with setting ResourceClaimStatus.Allocated. This
optimization may save scheduling attempts and roundtrips
through the API server because the scheduler does not
need to reserve the claim for the pod itself.

The selected node may change over time, for example
when the initial choice turns out to be unsuitable
after all. Drivers must not reallocate for a different
node when they see such a change because it would
lead to race conditions. Instead, the scheduler
will trigger deallocation of specific claims as
needed through the ResourceClaimStatus.DeallocationRequested
field.

The ResourceClass.SuiteableNodes node selector can be
used to filter out nodes based on labels. This prevents
adding nodes here that the driver then would need to
reject through UnsuitableNodes.

```
// PodSchedulingContextStatus describes where resources for the Pod can be allocated.
type PodSchedulingContextStatus struct {
    // ResourceClaims describes resource availability for each
    // pod.spec.resourceClaim entry where the corresponding ResourceClaim
    // uses "WaitForFirstConsumer" allocation mode.
    // +optional
    ResourceClaims []ResourceClaimSchedulingStatus

    // If there ever is a need to support other kinds of resources
    // than ResourceClaim, then new fields could get added here
    // for those other resources.
}
```

Each resource driver is responsible for providing information about
those resources in the Pod that the driver manages. It can skip
adding this information once it has allocated the resource.

A driver must add entries here for all its pending claims, even if
the ResourceSchedulingStatus.UnsuitabeNodes field is empty,
because the scheduler may decide to wait with selecting
a node until it has information from all drivers.

```
// ResourceClaimSchedulingStatus contains information about one particular
// ResourceClaim with "WaitForFirstConsumer" allocation mode.
type ResourceClaimSchedulingStatus struct {
    // Name matches the pod.spec.resourceClaims[*].Name field.
    Name string

    // UnsuitableNodes lists nodes that the ResourceClaim cannot be
    // allocated for.
    //
    // The size of this field is limited to 128, the same as for
    // PodSchedulingContextSpec.PotentialNodes. This may get increased in the
    // future, but not reduced.
    // +optional
    UnsuitableNodes []string
}

// PodSchedulingContextNodeListMaxSize defines the maximum number of entries in
// the node lists that are stored in PodSchedulingContexts. This limit is part
// of the API.
const PodSchedulingContextNodeListMaxSize = 256
```

UnsuitableNodes lists nodes that the claim cannot be allocated for.
Nodes listed here will be ignored by the scheduler when selecting a
node for a Pod. All other nodes are potential candidates, either
because no information is available yet or because allocation might
succeed.

A change to the PodSchedulingContextSpec.PotentialNodes field and/or a failed
allocation attempt triggers an update of this field: the driver
then checks all nodes listed in PotentialNodes and UnsuitableNodes
and updates UnsuitableNodes.

It must include the prior UnsuitableNodes in this check because the
scheduler will not list those again in PotentialNodes but they might
still be unsuitable.

This can change, so the driver must also refresh this information
periodically and/or after changing resource allocation for some
other ResourceClaim until a node gets selected by the scheduler.


### Coordinating resource allocation through the scheduler

For immediate allocation, scheduling Pods is simple because the
resource is already allocated and determines the nodes on which the
Pod may run. The downside is that pod scheduling is less flexible.

For delayed allocation, a node is selected tentatively by the scheduler
in an iterative process where the scheduler suggests some potential nodes
that fit the other resource requirements of a Pod and resource drivers
respond with information about whether they can allocate claims for those
nodes. This exchange of information happens through the `PodSchedulingContext`
for a Pod. The scheduler has to involve the drivers because it
doesn't know what claim parameters mean and where suitable resources are
currently available.

Once the scheduler is confident that it has enough information to select
a node that will probably work for all claims, it asks the driver(s) to
allocate their resources for that node. If that
succeeds, the Pod can get scheduled. If it fails, the scheduler must
determine whether some other node fits the requirements and if so,
request allocation again. If no node fits because some resources were
already allocated for a node and are only usable there, then those
resources must be released and then get allocated elsewhere.

This is a summary of the necessary [kube-scheduler changes](#kube-scheduler) in
pseudo-code:

```
while <pod needs to be scheduled> {
  <choose a node, considering potential availability for those resources
   which are not allocated yet and the hard constraints for those which are>
  if <no node fits the pod> {
    if <at least one resource
            is allocated and unused or reserved for the current pod,
            uses delayed allocation, and
            was not available on a node> {
      <randomly pick one of those resources and
       tell resource driver to deallocate it by setting `claim.status.deallocationRequested` and
       removing the pod from `claim.status.reservedFor` (if present there)>
    }
  } else if <all resources allocated> {
    <schedule pod onto node>
  } else if <some unallocated resource uses delayed allocation> {
    <tell resource driver to allocate for the chosen node>
  }
}
```

Randomly picking a node without knowing anything about the resource driver may
or may not succeed. To narrow the choice of suitable nodes for all claims using
a certain resource class, a node selector can be specified in that class. That
selector is static and typically will use labels that determine which nodes may
have resources available.

To gather information about the current state of resource availability and to
trigger allocation of a claim, the scheduler creates one PodSchedulingContext
for each pod that uses claims. That object is owned by the pod and
will either get deleted by the scheduler when it is done with pod scheduling or
through the garbage collector. In the PodSchedulingContext, the scheduler posts
the list of all potential nodes that it was left with after considering all
other pod constraints and requirements. Resource drivers involved in the
scheduling of the pod respond by adding which of these nodes currently don't
have sufficient resources available. The next scheduling attempt is then more
likely to pick a node for which allocation succeeds.

This scheduling information is optional and does not have to be in sync with
the current ResourceClaim state, therefore it is okay to store it
separately.

Allowing the scheduler to trigger allocation in parallel to asking for more
information was chosen because for pods with a single resource claim, the cost
of guessing wrong is low: the driver just needs to inform the scheduler to try
again and provide the additional information.

Additional heuristics are possible without changing the proposed API. For
example, the scheduler might ask for information and wait a while before
making a choice. This may be more suitable for pods using many different
resource claims because for those, allocation may succeed for some claims and
fail for others, which then may need to go through the recovery flow with
deallocating one or more claims.

### Resource allocation and usage flow

The following steps shows how resource allocation works for a resource that
gets defined in a ResourceClaimTemplate and referenced by a Pod. Several of these steps may fail without changing
the system state. They then must be retried until they succeed or something
else changes in the system, like for example deleting objects.

* **user** creates Pod with reference to ResourceClaimTemplate
* **resource claim controller** checks ResourceClaimTemplate and ResourceClass,
  then creates ResourceClaim with Pod as owner
* if *immediate allocation*:
  * **resource driver** adds finalizer to claim to prevent deletion -> allocation in progress
  * **resource driver** finishes allocation, sets `claim.status.allocation` -> claim ready for use by any pod
* if *pod is pending*:
  * **scheduler** filters nodes based on built-in resources and the filter callback of plugins,
    which includes constraints imposed by already allocated resources
  * if *delayed allocation and resource not allocated yet*:
    * if *at least one node fits pod*:
      * **scheduler** creates or updates a `PodSchedulingContext` with `podSchedulingContext.spec.potentialNodes=<nodes that fit the pod>`
      * if *exactly one claim is pending (see below)* or *all drivers have provided information*:
        * **scheduler** picks one node, sets `podSchedulingContext.spec.selectedNode=<the chosen node>`
        * if *resource is available for this selected node*:
          * **resource driver** adds finalizer to claim to prevent deletion -> allocation in progress
          * **resource driver** finishes allocation, sets `claim.status.allocation` and the
            pod in `claim.status.reservedFor` -> claim ready for use and reserved for the pod
        * else *scheduler needs to know that it must avoid this and possibly other nodes*:
          * **resource driver** sets `podSchedulingContext.status.claims[name=name of claim in pod].unsuitableNodes`
    * else *pod cannot be scheduled*:
      * **scheduler** may trigger deallocation of some claim with delayed allocation by setting `claim.status.deallocationRequested` to true
      (see [pseudo-code above](#coordinating-resource-allocation-through-the-scheduler)) or wait
  * if *pod not listed in `claim.status.reservedFor` yet* (can occur for immediate allocation):
    * **scheduler** adds it to `claim.status.reservedFor`
  * if *resource allocated and reserved*:
    * **scheduler** sets node in Pod spec -> Pod ready to run
    * **scheduler** deletes `PodSchedulingContext` if one exists
* if *node is set for pod*:
  * if `resource not reserved for pod` (user might have set the node field):
    * **kubelet** refuses to start the pod -> permanent failure
  * else `pod may run`:
    * **kubelet** asks driver to prepare the resource
  * if `resource is prepared`:
    * **kubelet** creates container(s) which reference(s) the resource through CDI -> Pod is running
* if *pod has terminated* and *pod deleted*:
  * **kubelet** asks driver to unprepare the resource
  * **kubelet** allows pod deletion to complete by clearing the `GracePeriod`
* if *pod removed*:
  * **garbage collector** deletes ResourceClaim -> adds `claim.deletionTimestamp` because of finalizer
* if *ResourceClaim has `claim.deletionTimestamp` and `claim.status.reservedFor` is empty*:
  * **resource driver** deallocates resource
  * **resource driver** clears finalizer and `claim.status.allocation`
  * **API server** removes ResourceClaim

When exactly one claim is pending, it is safe to trigger the allocation: if the
node is suitable, the allocation will succeed and the pod can get scheduled
without further delays. If the node is not suitable, allocation fails and the
next attempt can do better because it has more information. The same should not
be done when there are multiple claims because allocation might succeed for
some, but not all of them, which would force the scheduler to recover by asking
for deallocation. It's better to wait for information in this case.

The flow is similar for a ResourceClaim that gets created as a stand-alone
object by the user. In that case, the Pod reference that ResourceClaim by
name. The ResourceClaim does not get deleted at the end and can be reused by
another Pod and/or used by multiple different Pods at the same time (if
supported by the driver). The resource remains allocated as long as the
ResourceClaim doesn't get deleted by the user.

If a Pod references multiple claims managed by the same driver, then the driver
can combine updating `podSchedulingContext.claims[*].unsuitableNodes` for all
of them, after considering all claims.

### Scheduled pods with unallocated or unreserved claims

As with structured parameters, there are several scenarios where a Pod might be
scheduled (= `pod.spec.nodeName` set) while the claims that it depends on are
not allocated or not reserved for it. The kubelet is refusing to run such pods.

In addition to the solutions described for structured parameters, using a control
plane controller provides one additional solution:
- When kube-controller-manager observes that allocation is missing, it creates
  a `PodSchedulingContext` with only the `spec.selectedNode` field set to the
  name of the node chosen for the pod. There is no need to list suitable nodes
  because that choice is permanent, so resource drivers don't need check for
  unsuitable nodes. All that they can do is to (re)try allocating the claim
  until that succeeds.
- If such a pod has allocated claims that are not reserved for it yet,
  then kube-controller-manager can (re)try to reserve the claim until
  that succeeds.

Once all of those steps are complete, kubelet will notice that the claims are
ready and run the pod. Until then it will keep checking periodically, just as
it does for other reasons that prevent a pod from running.

### Cluster Autoscaler

When [Cluster
Autoscaler](https://github.com/kubernetes/autoscaler/tree/master/cluster-autoscaler#cluster-autoscaler)
encounters a pod that uses a resource claim for node-local resources, it needs
to understand the parameters for the claim and available capacity in order
to simulate the effect of allocating claims as part of scheduling and of
creating or removing nodes.

This is not possible with opaque parameters as described in this KEP. If a DRA
driver developer wants to support Cluster Autoscaler, they have to use
structured parameters as defined in [KEP
#4381](https://github.com/kubernetes/enhancements/issues/4381).

Structured parameters are not necessary for network-attached resources because
adding or removing nodes doesn't change their availability and thus Cluster
Autoscaler does not need to understand their parameters.

### Implementing a plugin for node resources

The proposal depends on a central resource driver controller. Implementing that
part poses an additional challenge for drivers that manage resources
locally on a node because they need to establish a secure
communication path between nodes and the central controller.

How drivers implement that is up to the developer. This section
outlines a possible solution. If there is sufficient demand, common
code for this solution could be made available as a reusable Go
module.

- Each driver defines a CRD which describes how much resources are
  available per node and how much is currently allocated.
- RBAC rules ensure that only the driver can modify objects of that
  type. The objects can and should be namespaced, which makes it
  possible to add automatic cleanup via owner references (similar to
  CSIStorageCapacity).
- The kubelet driver publishes information about the local state via a
  CRD object named after the node. Driver developers can document
  those CRDs and then users can query the cluster state by listing
  those objects.
- The driver controller watches those objects and ResourceClaims. It
  can keep track of claims that are in the process of being allocated
  and consider that when determining where another claim might get
  allocated. For delayed allocation, the driver controller informs the
  scheduler by updating the ResourceClaimStatus.UnsuitableNodes field.
  Eventually, the scheduler sets the selected node field. For immediate allocation,
  the driver controller itself sets the selected node field.
- In both cases, the kubelet plugin waits for a ResourceClaim assigned to
  its own node and tries to allocate the resource. If that fails, it
  can unset the selected node field to trigger another allocation
  attempt elsewhere.

### Implementing optional resources

This can be handled entirely by a resource driver: its parameters can support a
range starting at zero or a boolean flag that indicates that something is not a
hard requirement. When asked to filter nodes for delayed allocation, the driver
reports nodes where the resource is available and only falls back to those
without it when resources are exhausted. When asked to allocate, it reserves
actual resources if possible, but also proceeds with marking the ResourceClaim
as allocated if that is not possible. Kubernetes then can schedule a pod using
the ResourceClaim. The pod needs to determine through information passed in by
the resource driver which resources are actually available to it.


### Test Plan

[X] I/we understand the owners of the involved components may require updates to
existing tests to make this code solid enough prior to committing the changes necessary
to implement this enhancement.

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

- `k8s.io/kubernetes/pkg/scheduler`: 2022-05-24 - 75.0%
- `k8s.io/kubernetes/pkg/scheduler/framework`: 2022-05-24 - 76.3%
- `k8s.io/kubernetes/pkg/controller`: 2022-05-24 - 69.4%
- `k8s.io/kubernetes/pkg/kubelet`: 2022-05-24 - 64.5%

##### Integration tests

<!--
This question should be filled when targeting a release.
For Alpha, describe what tests will be added to ensure proper quality of the enhancement.

For Beta and GA, add links to added tests together with links to k8s-triage for those tests:
https://storage.googleapis.com/k8s-triage/index.html
-->

The existing [integration tests for kube-scheduler which measure
performance](https://github.com/kubernetes/kubernetes/tree/master/test/integration/scheduler_perf#readme)
were extended to also [cover
DRA](https://github.com/kubernetes/kubernetes/blob/294bde0079a0d56099cf8b8cf558e3ae7230de12/test/integration/scheduler_perf/config/performance-config.yaml#L717-L779)
and to runs as [correctness
tests](https://github.com/kubernetes/kubernetes/commit/cecebe8ea2feee856bc7a62f4c16711ee8a5f5d9)
as part of the normal Kubernetes "integration" jobs. That also covers [the
dynamic resource
controller](https://github.com/kubernetes/kubernetes/blob/294bde0079a0d56099cf8b8cf558e3ae7230de12/test/integration/scheduler_perf/util.go#L135-L139).

kubelet were extended to cover scenarios involving dynamic resources.

For beta:

- kube-scheduler, kube-controller-manager: http://perf-dash.k8s.io/#/, [`k8s.io/kubernetes/test/integration/scheduler_perf.scheduler_perf`](https://testgrid.k8s.io/sig-release-master-blocking#integration-master)
- kubelet: ...


##### e2e tests

<!--
This question should be filled when targeting a release.
For Alpha, describe what tests will be added to ensure proper quality of the enhancement.

For Beta and GA, add links to added tests together with links to k8s-triage for those tests:
https://storage.googleapis.com/k8s-triage/index.html

We expect no non-infra related flakes in the last month as a GA graduation criteria.
-->

End-to-end testing depends on a working resource driver and a container runtime
with CDI support. A [test driver](https://github.com/kubernetes/kubernetes/tree/master/test/e2e/dra/test-driver)
was developed in parallel to developing the
code in Kubernetes.

That test driver simply takes parameters from ResourceClass
and ResourceClaim and turns them into environment variables that then get
checked inside containers. Tests for different behavior of an driver in various
scenarios can be simulated by running the control-plane part of it in the E2E
test itself. For interaction with kubelet, proxying of the gRPC interface can
be used, as in the
[csi-driver-host-path](https://github.com/kubernetes-csi/csi-driver-host-path/blob/16251932ab81ad94c9ec585867104400bf4f02e5/cmd/hostpathplugin/main.go#L61-L63):
then the kubelet plugin runs on the node(s), but the actual processing of gRPC
calls happens inside the E2E test.

All tests that don't involve actually running a Pod can become part of
conformance testing. Those tests that run Pods cannot be because CDI support in
runtimes is not required.

For beta:
- pre-merge with kind (optional, triggered for code which has an impact on DRA): https://testgrid.k8s.io/sig-node-dynamic-resource-allocation#pull-kind-dra
- periodic with kind: https://testgrid.k8s.io/sig-node-dynamic-resource-allocation#ci-kind-dra
- pre-merge with CRI-O: https://testgrid.k8s.io/sig-node-dynamic-resource-allocation#pull-node-dra
- periodic with CRI-O: https://testgrid.k8s.io/sig-node-dynamic-resource-allocation#ci-node-e2e-crio-dra

### Graduation Criteria

#### Alpha -> Beta Graduation

- In normal scenarios, scheduling pods with claims must not block scheduling of
  other pods by doing blocking API calls
- Implement integration with Cluster Autoscaler through structured parameters
- Gather feedback from developers and surveys
- Positive acknowledgment from 3 would-be implementors of a resource driver,
  from a diversity of companies or projects
- Tests are in Testgrid and linked in KEP
- At least one scalability test for a likely scenario (for example,
  several pods each using different claims that get created from
  templates)
- Documentation for users and resource driver developers published
- In addition to the basic features, we also handle:
  - reuse of network-attached resources after unexpected node shutdown

#### Beta -> GA Graduation

- 3 examples of real-world usage
- Agreement that quota management is sufficient
- Conformance and downgrade tests
- Scalability tests that mirror real-world usage as
  determined by user feedback
- Allowing time for feedback


### Upgrade / Downgrade Strategy

The usual Kubernetes upgrade and downgrade strategy applies for in-tree
components. Vendors must take care that upgrades and downgrades work with the
drivers that they provide to customers.

### Version Skew Strategy

There may be situations where dynamic resource allocation is enabled in some
parts of the cluster (apiserver, kube-scheduler), but not on some nodes. The
resource driver is responsible for setting ResourceClaim.AvailableOnNodes so
that those nodes are not included.

But if a Pod with ResourceClaims already got scheduled onto a node without the
feature enabled, kubelet will start it without those additional
resources. Applications must be prepared for this and refuse to run. This will
put the Pod into a failed state that administrators or users need to resolve by
deleting the Pod.

The same applies when the entire cluster gets downgraded to a version where
dynamic resource allocation is unsupported or the feature gets disabled via
feature gates: existing Pods with ResoureClaims will be scheduled as if those
resources were not requested.

## Production Readiness Review Questionnaire

### Feature Enablement and Rollback

###### How can this feature be enabled / disabled in a live cluster?

- [X] Feature gate (also fill in values in `kep.yaml`)
  - Feature gate name: DynamicResourceAllocation
  - Components depending on the feature gate:
    - kube-apiserver
    - kube-controller-manager
    - kube-scheduler
    - kubelet

###### Does enabling the feature change any default behavior?

No.

###### Can the feature be disabled once it has been enabled (i.e. can we roll back the enablement)?

Yes. Applications that were already deployed and are running will continue to
work, but they will stop working when containers get restarted because those
restarted containers won't have the additional resources.

###### What happens if we reenable the feature if it was previously rolled back?

Pods might have been scheduled without handling resources. Those Pods must be
deleted to ensure that the re-created Pods will get scheduled properly.

###### Are there any tests for feature enablement/disablement?

Tests for apiserver will cover disabling the feature. This primarily matters
for the extended PodSpec: the new fields must be preserved during updates even
when the feature is disabled.

### Rollout, Upgrade and Rollback Planning

###### How can a rollout fail? Can it impact already running workloads?

Workloads not using ResourceClaims should not be impacted because the new code
will not do anything besides checking the Pod for ResourceClaims.

When kube-controller-manager fails to create ResourceClaims from
ResourceClaimTemplates, those Pods will not get scheduled. Bugs in
kube-scheduler might lead to not scheduling Pods that could run or worse,
schedule Pods that should not run. Those then will get stuck on a node where
kubelet will refuse to start them. None of these scenarios affect already
running workloads.

Failures in kubelet might affect running workloads, but only if containers for
those workloads need to be restarted.

###### What specific metrics should inform a rollback?


One indicator are unexpected restarts of the cluster control plane
components. Another are an increase in the number of pods that fail to
start. In both cases further analysis of logs and pod events is needed to
determine whether errors are related to this feature.

###### Were upgrade and rollback tested? Was the upgrade->downgrade->upgrade path tested?

This will be done manually before transition to beta by bringing up a KinD
cluster with kubeadm and changing the feature gate for individual components.

###### Is the rollout accompanied by any deprecations and/or removals of features, APIs, fields of API types, flags, etc.?

No.

### Monitoring Requirements

###### How can an operator determine if the feature is in use by workloads?

There will be pods which have a non-empty PodSpec.ResourceClaims field and ResourceClaim objects.

###### What are the SLIs (Service Level Indicators) an operator can use to determine the health of the service?

For kube-controller-manager, metrics similar to the generic ephemeral volume
controller [were added](https://github.com/kubernetes/kubernetes/blob/163553bbe0a6746e7719380e187085cf5441dfde/pkg/controller/resourceclaim/metrics/metrics.go#L32-L47):

- [X] Metrics
  - Metric name: `resource_controller_create_total`
  - Metric name: `resource_controller_create_failures_total`
  - Metric name: `workqueue` with `name="resource_claim"`

For kube-scheduler and kubelet, existing metrics for handling Pods already
cover most aspects. For example, in the scheduler the
["unschedulable_pods"](https://github.com/kubernetes/kubernetes/blob/6f5fa2eb2f4dc731243b00f7e781e95589b5621f/pkg/scheduler/metrics/metrics.go#L200-L206)
metric will call out pods that are currently unschedulable because of the
`DynamicResources` plugin.

For the communication between scheduler and controller, the apiserver metrics
about API calls (e.g. `request_total`, `request_duration_seconds`) for the
`podschedulingcontexts` and `resourceclaims` resources provide insights into
the amount of requests and how long they are taking.

###### What are the reasonable SLOs (Service Level Objectives) for the above SLIs?

For Pods not using ResourceClaims, the same SLOs apply as before.

For kube-controller-manager, metrics for the new controller could be checked to
ensure that work items do not remain in the queue for too long, for some
definition of "too long".

Pod scheduling and startup are more important. However, expected performance
will depend on how resources are used (for example, how often new Pods are
created), therefore it is impossible to predict what reasonable SLOs might be.

The resource manager component will do its work similarly to the
existing volume manager, but the overhead and complexity should
be lower:

* Resource preparation should be fairly quick as in most cases it simply
  creates CDI file 1-3 Kb in size. Unpreparing resource usually means
  deleting CDI file, so it should be quick as well.

* The complexity is lower than in the volume manager
  because there is only one global operation needed (prepare vs.
  attach + publish for each pod).

* Reconstruction after a kubelet restart is simpler (call
  NodePrepareResource again vs. trying to determine whether
  volumes are mounted).

###### Are there any missing metrics that would be useful to have to improve observability of this feature?

No.

### Dependencies

The container runtime must support CDI.

###### Does this feature depend on any specific services running in the cluster?

A third-party resource driver is required for allocating resources.

### Scalability

###### Will enabling / using this feature result in any new API calls?

For Pods not using ResourceClaims, not much changes. kube-controller-manager,
kube-scheduler and kubelet will have additional watches for ResourceClaim and
ResourceClass, but if the feature isn't used, those watches
will not cause much overhead.

If the feature is used, ResourceClaim will be modified during Pod scheduling,
startup and teardown by kube-scheduler, the third-party resource driver and
kubelet. Once a ResourceClaim is allocated and the Pod runs, there will be no
periodic API calls. How much this impacts performance of the apiserver
therefore mostly depends on how often this feature is used for new
ResourceClaims and Pods. Because it is meant for long-running applications, the
impact should not be too high.

###### Will enabling / using this feature result in introducing new API types?

For ResourceClass, only a few (something like 10 to 20)
objects per cluster are expected. Admins need to create those.

The number of ResourceClaim objects depends on how much the feature is
used. They are namespaced and get created directly or indirectly by users. In
the most extreme case, there will be one or more ResourceClaim for each Pod.
But that seems unlikely for the intended use cases.

Kubernetes itself will not impose specific limitations for the number of these
objects.

###### Will enabling / using this feature result in any new calls to the cloud provider?

Only if the third-party resource driver uses features of the cloud provider.

###### Will enabling / using this feature result in increasing size or count of the existing API objects?

The PodSpec potentially changes and thus all objects where it is embedded as
template. Merely enabling the feature does not change the size, only using it
does.

In the simple case, a Pod references existing ResourceClaims by name, which
will add some short strings to the PodSpec and to the ContainerSpec. Embedding
a ResourceClaimTemplate will increase the size more, but that will depend on
the number of custom parameters supported by a resource driver and thus is hard to
predict.

The ResourceClaim objects will initially be fairly small. However, if delayed
allocation is used, then the list of node names or NodeSelector instances
inside it might become rather large and in the worst case will scale with the
number of nodes in the cluster.

###### Will enabling / using this feature result in increasing time taken by any operations covered by existing SLIs/SLOs?

Startup latency of schedulable stateless pods may be affected by enabling the
feature because some CPU cycles are needed for each Pod to determine whether it
uses ResourceClaims.

Actively using the feature will increase load on the apiserver, so latency of
API calls may get affected.

###### Will enabling / using this feature result in non-negligible increase of resource usage (CPU, RAM, disk, IO, ...) in any components?

Merely enabling the feature is not expected to increase resource usage much.

How much using it will increase resource usage depends on the usage patterns
and is hard to predict.

### Troubleshooting

###### How does this feature react if the API server and/or etcd is unavailable?

The Kubernetes control plane will be down, so no new Pods get
scheduled. kubelet may still be able to start or or restart containers if it
already received all the relevant updates (Pod, ResourceClaim, etc.).

###### What are other known failure modes?

- DRA driver does not or cannot allocate a resource claim.

  - Detection: The primary mechanism is through vendors-provided monitoring for
    their driver. That monitor needs to include health of the driver, availability
    of the underlying resource, etc. The common helper code for DRA drivers
    posts events for a ResourceClaim when an allocation attempt fails.

    When pods fail to get scheduled, kube-scheduler reports that through events
    and pod status. For DRA, that includes "waiting for resource driver to
    provide information" (node not selected yet) and "waiting for resource
    driver to allocate resource" (node has been selected). The
    ["unschedulable_pods"](https://github.com/kubernetes/kubernetes/blob/9fca4ec44afad4775c877971036b436eef1a1759/pkg/scheduler/metrics/metrics.go#L200-L206)
    metric will have pods counted under the "dynamicresources" plugin label.

    To troubleshoot, "kubectl describe" can be used on (in this order) Pod,
    ResourceClaim, PodSchedulingContext.

  - Mitigations: This depends on the vendor of the DRA driver.

  - Diagnostics: In kube-scheduler, -v=4 enables simple progress reporting
    in the "dynamicresources" plugin. -v=5 provides more information about
    each plugin method. The special status results mentioned above also get
    logged.

  - Testing: E2E testing covers various scenarios that involve waiting
    for a DRA driver. This also simulates partial allocation of node-local
    resources in one driver and then failing to allocate the remaining
    resources in another driver (the "need to deallocate" fallback).

- A Pod gets scheduled without allocating resources.

  - Detection: The Pod either fails to start (when kubelet has DRA
    enabled) or gets started without the resources (when kubelet doesn't
    have DRA enabled), which then will fail in an application specific
    way.

  - Mitigations: DRA must get enabled properly in kubelet and kube-controller-manager.
    Then kube-controller-manager will try to allocate and reserve resources for
    already scheduled pods. To prevent this from happening for new pods, DRA
    must get enabled in kube-scheduler.

  - Diagnostics: kubelet will log pods without allocated resources as errors
    and emit events for them.

  - Testing: An E2E test covers the expected behavior of kubelet and
    kube-controller-manager by creating a pod with `spec.nodeName` already set.

- A DRA driver kubelet plugin fails to prepare resources.

  - Detection: The Pod fails to start after being scheduled.

  - Mitigations: This depends on the specific DRA driver and has to be documented
    by vendors.

  - Diagnostics: kubelet will log pods with such errors and emit events for them.

  - Testing: An E2E test covers the expected retry mechanism in kubelet when
    `NodePrepareResources` fails intermittently.


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

Performance depends on a large extend on how individual DRA drivers are
implemented. Vendors will have to provide their own SLOs and troubleshooting
instructions.

## Implementation History

- Kubernetes 1.25: KEP accepted as "implementable".
- Kubernetes 1.26: Code merged as "alpha".
- Kubernetes 1.27: API breaks (batching of NodePrepareResource in kubelet API,
  AllocationResult in ResourceClaim status can provide results for multiple
  drivers).
- Kubernetes 1.28: API break (ResourceClaim names for claims created from
  a template are generated instead of deterministic), scheduler performance
  enhancements (no more backoff delays).
- Kubernetes 1.29, 1.30: most blocking API calls moved into Pod binding goroutine

## Drawbacks

<!--
Why should this KEP _not_ be implemented?
-->

The flow of information between the scheduler and DRA drivers through the
PodSchedulingContext is complex.
