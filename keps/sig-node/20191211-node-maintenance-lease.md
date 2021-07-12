---
title: Node Maintenance Lease
authors:
  - "@michaelgugino"
owning-sig: sig-cluster-lifecycle
participating-sigs:
  - sig-node
reviewers:
  - "@neolit123"
approvers:
  - TBD
editor: TBD
creation-date: 2019-12-11
last-updated: 2020-06-25
status: provisional
see-also:
  - "https://github.com/kubernetes/enhancements/issues/1403"
---

# Node Maintenance Lease

## Table of Contents

<!-- toc -->
- [Release Signoff Checklist](#release-signoff-checklist)
- [Summary](#summary)
- [Motivation](#motivation)
  - [Goals](#goals)
  - [Non-Goals](#non-goals)
- [Proposal](#proposal)
  - [User Stories](#user-stories)
    - [Story 1](#story-1)
    - [Story 2](#story-2)
    - [Story 3](#story-3)
  - [Implementation Details/Notes/Constraints](#implementation-detailsnotesconstraints)
    - [Impact to the Kubelet](#impact-to-the-kubelet)
    - [Impact to Scheduleability](#impact-to-scheduleability)
    - [Acquiring and Releasing the Lease](#acquiring-and-releasing-the-lease)
      - [Acquisition](#acquisition)
      - [Release](#release)
    - [Expected Client Behavior](#expected-client-behavior)
  - [Risks and Mitigations](#risks-and-mitigations)
- [Alternatives](#alternatives)
  - [Annotations on Node or elsewhere](#annotations-on-node-or-elsewhere)
  - [Field in Node Spec Or Status](#field-in-node-spec-or-status)
  - [Use existing system namespace for lease object](#use-existing-system-namespace-for-lease-object)
  - [Use optional non-system namespace](#use-optional-non-system-namespace)
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

Node Maintenance Leases are useful to coordinate disruptive actions
controllers or administrators might wish to take against a node. This lease
is a voluntary mechanism and allows coordination among different components
without those components needing knowledge of each other.

## Motivation

Currently, there is no centralized way to inform a controller or user that
another controller or user is performing a disruptive operation against a node.
Examples of disruptive operations include:
1. Rebooting a node
1. Powering off a node for maintenance
1. Draining or Cordoning a node
1. Deleting/deprovisioning a node due to failed health checks

### Goals

1. Define a central point of coordination for various actors.
1. Define high-level steps for well-behaved clients to implement.

### Non-Goals

1. Define actors which might utilize the lease.
1. Define an enforcement mechanism for prevent components from "stealing" the
lease when it shouldn't.  This could be future work accomplished by an optional
webhook.

## Proposal

Utilize the existing `Lease` built-in API in the API group `coordination.k8s.io`
to coordinate maintenance or disruptions for nodes.

A new namespace `kube-node-maintenance` will be created to allow creating
lease objects that have the same name as a node.

The namespace `kube-node-maintenance` should be created automatically in the
same manner as the existing kube-node-lease namespace.  As of time of writing
this enhancement, these namespaces are created here:
https://github.com/kubernetes/kubernetes/blob/release-1.18/pkg/master/controller.go#L202

The Lease object will be created automatically by the NodeLifecycleController,
and the ownerRef will be the corresponding node so it is removed when the node
is deleted. If the Lease object is removed before the node is deleted, the
NodeLifecycleController will recreate it.

### User Stories

#### Story 1

My cluster's nodes receive live updates to their on-disk configurations by an
update agent.  Part of this upgrade process involves a reboot of the node.
From time to time, the node rejoining the cluster might be delayed a small
amount, especially on Bare Metal hosts.  I do not want automated actions to
deprovision my node.

An example of this story is any process that requires periodically rebooting
and node and using MachineHealthChecks. MachineHealthChecks can be configured
to automatically delete a machine that fails health checks.
MachineHealthChecks can easily be adapted to ignore any nodes that have a
currently acquired maintenance lease.


#### Story 2

I am investigating a problem with the node. I don't want the node to be disrupted
by automation while I'm performing my investigation or fixes, and I don't want
to stop the automation components from acting on other nodes.

An example of this story is clusters utilizing cluster-api and MachineHealthChecks.
MachineHealthChecks can be configured to automatically delete a machine that
fails health checks.  MachineHealthChecks can easily be adapted to ignore any
nodes that have a currently acquired maintenance lease.

#### Story 3

I have a configuration management system that requires the node to be online to perform configuration updates.

I have an automated-snapshot system that stops an instance and takes a snapshot for backup / data retention purposes.

I do not want snapshots to happen during an update.

### Implementation Details/Notes/Constraints

#### Impact to the Kubelet

None. The kubelet does not need to respond to this lease or interact with it in
any way.  This is for other components to coordination actions they intend to
take (or prevent) on the node that might be disruptive.

#### Impact to Scheduleability

None. The existence of acquiring of this lease does not imply any taints,
annotations, labels, cordoning or draining will take place. The Lease object
itself is not intended as a mechanism to start any automated process.

A user/controller acquires the Lease object, and then that user/controller
may add/remove taints, cordon or drain, if they choose to.  Other components
may wish to integrate with the new Lease object to avoid conflict. EG:
a component that ensures a taint is always present on a node might stop ensuring
that taint if there is a current valid lease holder, but that implementation
is out of scope here.

#### Acquiring and Releasing the Lease
The lease object shall conform to the [Lease v1 coordination.k8s.io](https://kubernetes.io/docs/reference/generated/kubernetes-api/v1.18/#lease-v1-coordination-k8s-io)
API.

##### Acquisition
To acquire the lease, a client must check for current lease expiry and ensure
the holderIdentity is not set to a value begging with "kubeadm".

Lease expiry is defined as comparing the current time to the lease's
renewTime + leaseDurationSeconds + 3 seconds (to allow for some clock drift).

A Lease is always considered "held" or unexpired if the holderIdentity is
set to a value begging with "kubeadm".  This is because it is burdensome for
a user to specify a time value without additional tooling.

If a client acquires a lease, it must set all fields of the lease object.
leaseDurationSeconds should be set to an appropriate time, as short as is
reasonable, not to exceed 1 hour, unless renewTime is periodically updated in
internvals less than 1 hour. Clients may hold a lease as long
as is necessary, ensuring that renewTime and leaseDurationSeconds are
periodically updated prior to lease expiration.

Clients should never set holderIdentity to a value that begins with
"kubeadm" as that value is reserved for administrator use.

Administrators may optionally set the lease holderIdentity to any value begging
with "kubeadm" and any leaseDurationSeconds to accommodate their needs without
burden of frequent updates.  leaseDurationSeconds are not required when setting
holderIdentity to any value beginning with "kubeadm".

Conformance to these acquisition specifications will allow for easier monitoring
and alerting of abandoned long-running leases or clients that do not otherwise
release their lease in a timely manner.

##### Release

Clients should first determine if they still hold the lease.

If the lease is still held, set leaseDurationSeconds to 0.

If an administrator set holderIdentity to a value beginning with "kubeadm",
that value should be removed or altered to not begin with "kubeadm".

#### Expected Client Behavior

1. Client has required RBAC permissions for new Namespace and Lease object
1. Client checks for the existence of a lease
1. If the lease exists, confirm the lease is not currently held by another user
1. Attempt to acquire the lease and set an appropriate LeaseDurationSeconds
1. If acquired, perform the necessary operations
1. Release the lease

### Risks and Mitigations

Clients might not behave appropriately.  We can mitigate this by providing
a library with an easy to use interface.

Cluster administrators might wish to acquire the lease without the use of
a controller as a measure to prevent others from acquiring it and performing
disruptive actions on a node.  Specifying a holderIdentity begging with
"kubeadm" allows them to do this easily with kubectl instead of having to rely
on an additional tool to set a complicated date/time format.

Leases may get abandoned from time to time due to controllers holding the lease
crashing, getting deleted, or otherwise failing to release in a timely fashion.
Specifying expected behavior of all lease-holders to use a lease duration of
less than 1 hour with more frequent renewals allows for definitive monitoring
and alerting boundaries of abandoned leases.

## Alternatives

### Annotations on Node or elsewhere

Use a designated annotation on the node object, or possibly
elsewhere.  Drawbacks to this include excess updates being applied to the node
object directly, which might have performance implications. Having such an
annotation natively seems like an anti-pattern and could easily be disrupted
by existing controllers/clients attempting to manage those annotations.

### Field in Node Spec Or Status

Extending the node API will be much more difficult.  This behavior is also
somewhat orthogonal to the node itself.

### Use existing system namespace for lease object

We could utilize an existing system namespace for the lease object. The primary
issue with this is name collision, especially in the kube-node-lease namespace.
The kube-node-lease namespace utilizes the same 1-to-1 mapping of Lease objects
to nodes and also shares node names.  Other existing namespaces might have
undesirable RBAC requirements.

### Use optional non-system namespace

Administrators that wish to enable this functionality could create a namespace
with a specific name.  Clients could use a Get-or-Create method to check the
lease.  This will make installing components that support this
functionality more difficult as RBAC will not be deterministic at install time
of those components.  This could also be handled as an add-on, but that still
presents the problem of ensure the add-on is enabled prior to trying to consume
this behavior.
