---
title: Commit Class
authors:
  - "@kanatohodets"
owning-sig: sig-node
participating-sigs:
  - sig-scheduling
  - sig-scalability
reviewers: []
approvers:
  - TBD
editor: TBD
creation-date: 2019-05-30
last-updated: 2019-06-03
status: provisional
---

# Commit Class

## Table of Contents

- [Release Signoff Checklist](#release-signoff-checklist)
- [Summary](#summary)
- [Motivation](#motivation)
  - [Goals](#goals)
  - [Non-Goals](#non-goals)
- [Proposal](#proposal)
  - [API Design](#api-design)
  - [Implementation Details](#implementation-details)
  - [Risks and Mitigations](#risks-and-mitigations)
- [Design Details](#design-details)
  - [Graduation Criteria](#graduation-criteria)
  - [Upgrade / Downgrade Strategy](#upgrade--downgrade-strategy)
  - [Version Skew Strategy](#version-skew-strategy)
- [Implementation History](#implementation-history)
- [Drawbacks](#drawbacks)
- [Alternatives](#alternatives)
  - [Mutating Admission Webhook](#mutating-admission-webhook)
  - [Vertical Pod Autoscaler](#vertical-pod-autoscaler)
  - [Delegate to underlying infrastructure](#delegate-to-underyling-infrastructure)

## Release Signoff Checklist

- [ ] kubernetes/enhancements issue in release milestone, which links to KEP (this should be a link to the KEP location in kubernetes/enhancements, not the initial KEP PR)
- [ ] KEP approvers have set the KEP status to `implementable`
- [ ] Design details are appropriately documented
- [ ] Test plan is in place, giving consideration to SIG Architecture and SIG Testing input
- [ ] Graduation criteria is in place
- [ ] "Implementation History" section is up-to-date for milestone
- [ ] User-facing documentation has been created in [kubernetes/website], for publication to [kubernetes.io]
- [ ] Supporting documentation e.g., additional design documents, links to mailing list discussions/SIG meetings, relevant PRs/issues, release notes

**Note:** Any PRs to move a KEP to `implementable` or significant changes once it is marked `implementable` should be approved by each of the KEP approvers. If any of those
approvers is no longer appropriate than changes to that list should be approved by the remaining approvers and/or the owning SIG (or SIG-arch for cross cutting KEPs).

**Note:** This checklist is iterative and should be reviewed and updated every time this enhancement is being considered for a milestone.

[kubernetes.io]: https://kubernetes.io/
[kubernetes/enhancements]: https://github.com/kubernetes/enhancements/issues
[kubernetes/kubernetes]: https://github.com/kubernetes/kubernetes
[kubernetes/website]: https://github.com/kubernetes/website

## Summary

Over-committing physical host resources is a powerful tool for managing server
footprint, and therefore cost. The canonical example is CPU, where
virtualization platforms run 10+ virtual CPUs per physical CPU. Kubernetes
currently allows this by manipulating the ratio between resource `request` and
resource `limit` on a per-container basis. However, these values may be set by
a single application owner, who may not have a clear view on the utilization
level over the entire cluster.

This KEP proposes an API, `CommitClass`, to allow cluster operators to over- or
under-commit the physical resources on a group of nodes by modifying the
amount of resource advertised by kubelet to the scheduler.

## Motivation

Cost is a key quality metric for business value delivered by a Kubernetes
cluster. Cluster cost is typically proportional to the number of nodes required
to run the Pods present on the cluster, which is a function of the resources
requested by those Pods.

One traditional strategy for minimizing the number of nodes required to run the
desired workloads is over-commit of resources. The key hypothesis is that load
spikes between distinct workloads are typically uncoordinated, and so the same
resource may be allocated multiple times to different workloads. Higher
over-commit levels achieve better cost outcomes, but risk compromising service
quality if the uncoordinated-spike hypothesis does not hold.

Kubernetes currently provides for resource over-commit by allowing Pods to
declare themselves 'Burstable' or 'Best Effort' via container resource
request:limit ratios. This is a powerful approach, but achieving high
utilization at the cluster level requires coordination across many authors of
Pod specs, which is challenging in a 'Namespace as a Service' multi-tenant
Kubernetes environment, or surprising when implemented by fiat as a mutating
admission webhook or required usage of Vertical Pod Autoscaler.

The key belief behind this KEP is that Pod resource request:limit ratios are
the business of the owner of that Pod, and should be primarily concerned with
the cost of running that particular Pod. The cost of the cluster as a whole is
outside their scope; as such, Kubernetes should provide a "big hammer" for
cluster operators to use their wider perspective on cluster utilization to
reduce cost via platform-level over-commit policies.

### Goals

* Allow platform-level over- or under- subscription of resources on groups of
  nodes.

### Non-Goals

* Help cluster operators choose appropriate commit settings.
* Automatically adjust commit settings to maximize utilization.
* Replace VPA, HPA, or any other Pod-level right-sizing/auto-scaling API.
  These APIs are complimentary: they help Pod owners optimize their Pod
  footprint, while `CommitClass` helps cluster operators optimize the cluster
  footprint.

## Proposal

TODO

Approximately: introduce a `CommitClass` cluster-scoped object which defines
a node selector and a list of resource multipliers. All nodes which match the
node selector should apply the multipliers to the associated resource when
updating their Node API object.

#### Open question: conflict between multiple CommitClasses

How do you resolve conflict if multiple nodes are selected by the same
`CommitClass`? kubelet probably needs to record which `CommitClass` is in
effect on a Node object so that it is clear to operators what's up. One
alternative strategy would be to have a 'Binding' style object which enumerated
the Node names it applied to, but that interacts badly with Cluster autoscaler.

Nodes should default to a `CommitClass` where all multipliers are '1'.

#### Consideration: container CPU requests vs host physical CPUs
It is usually a bad idea to give a single VM more vCPUs than physical CPUs are
available on the host system. Should that be reflected here somehow for
containers?

#### Consideration: system reserved
System reserved resource slices should almost certainly not be subject to
`CommitClass` multipliers, so that node health is not at risk under high
multipliers ("just" workload health).

### API Design

TODO, perhaps something like:

```yaml
kind: CommitClass
apiVersion: node.k8s.io/v1alpha1
metadata:
    name: high-cpu-density
spec:
  nodes:
    nodeSelectorTerms:
    - matchExpressions:
      - key: beta.kubernetes.io/instance-type
      operator: In
      values:
      - compute-optimized
  resources:
  - name: cpu
    multiplier: 10
  - name: memory
    multiplier: 1.2
```

### Implementation Details

(very approximate, my initial guess)

With this proposal, the following changes are required:
 - Introduce a new cluster-scoped API object, `CommitClass`.
 - Add an informer for `CommitClass` objects to `kubelet`.
 - When updating the Node object with resources available for scheduling,
   `kubelet` should determine which `CommitClass` object is in effect, and use
   the multipliers of that `CommitClass` to change the resource amount
   advertised.
 - Determine a location in the Node API to record which `CommitClass` is in
   effect for that Node.
 - Change `kubelet` to record the `CommitClass` into that location.

In more detail, first cut implementation idea:

* Create a new kubelet manager, the `commitManager`. This manager should embed
  an informer/lister for `CommitClass` objects.
* Give the `commitManager` a func like `getNodeCommitSettings`. This func
  should take a 'node' object and return a map of resource name to multiplier,
  or nil.
* Add a new 'getter' to the `nodestatus.MachineInfo` function factory called
  `nodeCommitSettings`.  By default, this should be
  `klet.commitManager.getNodeCommitSettings`. At the end of the error
  checking `else` branch reading `machineInfoFunc()`, check the feature gate,
  and if positive, invoke `nodeCommitSettings` and store output into a map with
  function-level scope.
* If nil, do nothing. If err, log err and do nothing. If we get a result,
  iterate over the resulting map and multiply the value in
  `node.Status.Capacity[rname]` by the multiplier for that `rname`. This goes
  at the _end_ of the `else` clause so that it can touch all the different
  resource kinds.
* The system reserved resources should not be subject to CommitClass
  compression, where the real capacity that '2 CPUs' represents is shifted into
  '0.2 CPUs (or 2 vCPUs)' by `CommitClass` with 10x CPU commit. So
  `nodeAllocatableReservationFunc()` `allocatableReservation` should also be
  transformed by the `CommitClass` multipliers before writing to
  `node.Status.Allocatable`: that way, 2 real CPUs reserved will be 20 reserved
  vCPUs, which is a correct representation of the operator intent.

* Also learn whether the Admit function in PredicateAdmitHandler needs to
  apply the multipliers, or whether it uses the Node object already transformed.

### Risks and Mitigations

This proposal introduces new behavior to `kubelet` in a way which can
meaningfully impact its work. A feature gate may be appropriate.

## Design Details

### Graduation Criteria

TODO. Probably feature gate, alpha/beta/GA sequence, figure out graduation
criteria between levels.

### Upgrade / Downgrade Strategy

No action is required to keep existing behavior on upgrade: kubelets should
default (explicitly or implicitly) to a `CommitClass` where all multipliers are
'1'.

A cluster operator may start using this enhancement by upgrading to a version
of kubelet which supports it, and using the CommitClass API.

Downgrading kubelet will cause the CommitClass API to lose effect, which may
result in Pods being over-scheduled on a node.

### Version Skew Strategy

Kubelets from versions which do not support CommitClass will advertise the
exact amount of resources they have. As they are upgraded to a version with
CommitClass support, they will advertise a different amount of resources
depending on the CommitClass in effect. This may be surprising if a CommitClass
selector includes nodes which do not yet support it.

## Implementation History

- 2019-05-30: Draft KEP published.

## Drawbacks

The proposed CommitClass API has the potential to materially impact the cost
and service quality of a Kubernetes cluster. Using the CommitClass API to
over-commit may introduce workload performance degradation or instability.
Conversely, significantly under-committing cluster resources will result in
excessive cost. These issues may result in a higher support burden for the
Kubernetes community as cluster operators encounter these extremes.

This KEP introduces technical complexity to the `kubelet`. It also introduces
a layer of indirection for cluster operators debugging resource or performance
issues.

## Alternatives

### Mutating Admission Webhook

Systemic over- or under-commit may be implemented by a mutating admission
webhook which modifies a Pod's resource request by a particular amount. This is
[how it is implemented in
OpenShift](https://docs.openshift.com/container-platform/3.4/admin_guide/overcommit.html#configuring-masters-for-overcommitment).

This approach is very straightforward to implement, and requires no new API
from Kubernetes. However, it has a few drawbacks:

- In a multi-tenant environment, Pod owners may be surprised that their Pod
  specs have been mutated.
- The webhook is likely to change 'guaranteed' class Pods into 'burstable'
  ones, which impacts priority in case of resource contention. If it maintains
  'guaranteed' class by also reducing resource limits, it may starve the
  application.
- Changing over-commit level based on node characteristics requires careful
  business logic in the webhook (likely a Node informer).

### Vertical Pod Autoscaler

Pod resource requests may also be managed with use of the Vertical Pod
Autoscaler. With this approach, Pods which use minimal resources will find
their resource requests shrinking over time. If enough of a cluster's workloads
are managed by VPA, high cluster resource density may be achieved without
over-commit.

VPA is a powerful tool for right-sizing Pods, but is difficult to wield for the
purposes of cluster-wide footprint optimization:

- It is difficult to mandate VPA usage in a 'Namespace as a Service' style
  Kubernetes platform, where such a mandate crosses ownership boundaries.
- VPA complicates capacity planning for Pod owners, so not all Pod owners may
  want to use it.
- VPA may increase platform exposure to application memory or CPU leaks, and so
  may be an undesirable default.

### Delegate to Underlying Infrastructure

Virtual Machine infrastructures have provided distinct user-level resource
request/limit and platform-level over-commit settings for some time. The
Kubernetes project could take the position that Kubernetes clusters with an
interest in platform-level cost optimization should run on top of a VM
infrastructure, and thereby delegate the commit settings to the VM
orchestrator.
