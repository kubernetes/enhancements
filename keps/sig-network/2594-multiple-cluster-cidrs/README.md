# KEP-2594: Enhanced NodeIPAM to support Discontiguous Cluster CIDR

<!-- toc -->
- [Release Signoff Checklist](#release-signoff-checklist)
- [Summary](#summary)
- [Motivation](#motivation)
  - [Goals](#goals)
  - [Non-Goals](#non-goals)
- [Proposal](#proposal)
  - [User Stories](#user-stories)
    - [Add more pod IPs to the cluster](#add-more-pod-ips-to-the-cluster)
    - [Add nodes with higher or lower capabilities](#add-nodes-with-higher-or-lower-capabilities)
    - [Provision discontiguous ranges](#provision-discontiguous-ranges)
  - [Notes/Constraints/Caveats](#notesconstraintscaveats)
  - [Risks and Mitigations](#risks-and-mitigations)
- [Design Details](#design-details)
  - [New Resource](#new-resource)
    - [Expected Behavior](#expected-behavior)
    - [Example: Delete In-use ClusterCIDRConfig](#example-delete-in-use-clustercidrconfig)
    - [Example: Allocations](#example-allocations)
  - [Controller](#controller)
    - [Data Structures](#data-structures)
    - [Startup](#startup)
    - [Reconciliation Loop](#reconciliation-loop)
    - [Event Watching Loops](#event-watching-loops)
      - [Node Added](#node-added)
      - [Node Updated](#node-updated)
      - [Node Deleted](#node-deleted)
      - [ClusterCIDRConfig Added](#clustercidrconfig-added)
      - [ClusterCIDRConfig Updated](#clustercidrconfig-updated)
      - [ClusterCIDRConfig Deleted](#clustercidrconfig-deleted)
  - [kube-controller-manager](#kube-controller-manager)
  - [Test Plan](#test-plan)
    - [Unit Tests and Benchmarks](#unit-tests-and-benchmarks)
    - [Integration Tests](#integration-tests)
    - [End-to-End Tests](#end-to-end-tests)
  - [Graduation Criteria](#graduation-criteria)
    - [Alpha -> Beta Graduation](#alpha---beta-graduation)
    - [Beta -> GA Graduation](#beta---ga-graduation)
    - [Make the Controller the new default](#make-the-controller-the-new-default)
    - [**TBD:** Mark the RangeAllocator as deprecated](#tbd-mark-the-rangeallocator-as-deprecated)
  - [Upgrade / Downgrade Strategy](#upgrade--downgrade-strategy)
    - [Upgrades](#upgrades)
    - [Downgrades](#downgrades)
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
  - [Share Resources with Service API](#share-resources-with-service-api)
    - [Pros](#pros)
    - [Cons](#cons)
  - [Nodes Register CIDR Request](#nodes-register-cidr-request)
    - [Pros](#pros-1)
    - [Cons](#cons-1)
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

Items marked with (R) are required *prior to targeting to a milestone /
release*.

-   [ ] (R) Enhancement issue in release milestone, which links to KEP dir in
    [kubernetes/enhancements](not the initial KEP PR)
-   [ ] (R) KEP approvers have approved the KEP status as `implementable`
-   [ ] (R) Design details are appropriately documented
-   [ ] (R) Test plan is in place, giving consideration to SIG Architecture and
    SIG Testing input (including test refactors)
-   [ ] (R) Graduation criteria is in place
-   [ ] (R) Production readiness review completed
-   [ ] (R) Production readiness review approved
-   [ ] "Implementation History" section is up-to-date for milestone
-   [ ] User-facing documentation has been created in [kubernetes/website], for
    publication to [kubernetes.io]
-   [ ] Supporting documentation—e.g., additional design documents, links to
    mailing list discussions/SIG meetings, relevant PRs/issues, release notes

[kubernetes.io]: https://kubernetes.io/
[kubernetes/enhancements]: https://git.k8s.io/enhancements
[kubernetes/kubernetes]: https://git.k8s.io/kubernetes
[kubernetes/website]: https://git.k8s.io/website

## Summary

Today, when Kubernetes' NodeIPAM controller allocates IP ranges for podCIDRs for
nodes, it uses a single range allocated to the cluster (cluster CIDR). Each node
gets a range of a fixed size from the overall cluster CIDR. The size is
specified during cluster startup time and cannot be modified later on.

Kubernetes' IPAM capabilities are an optional behavior that comes with
Kubernetes out of the box. It is not required for Kubernetes to function, and
users may use alternate mechanisms.

This proposal enhances how pod CIDRs are allocated for nodes by adding a new
CIDR allocator that can be controlled by a new resource `ClusterCIDRRange`. This
would enable users to dynamically allocate more IP ranges for pods. The new
functionality would remain optional, and be an enhancement for those using the
built-in IPAM functionality.

## Motivation

Today, IP ranges for podCIDRs for nodes are allocated from a single range
allocated to the cluster (cluster CIDR). Each node gets a range of a fixed size
from the overall cluster CIDR. The size is specified during cluster startup time
and cannot be modified later on. This has multiple disadvantages:

*   There is just one cluster CIDR from which all pod CIDRs are allocated. This
    means that users need to provision the entire IP range up front accounting
    for the largest cluster that may be created. This can waste IP addresses.
*   If a cluster grows beyond expectations, there isn't a simple way to add more
    IP addresses.
*   The cluster CIDR is one large range. It may be difficult to find a
    contiguous block of IP addresses that satisfy the needs of the cluster.
*   Each node gets a fixed size IP range within a cluster. This means that if
    nodes are of different sizes and capacity, users cannot allocate a bigger
    pod range to a given node with larger capacity and a smaller range to nodes
    with lesser capacity. This wastes a lot of IP addresses.

### Goals

*   Support multiple discontiguous IP CIDR blocks for Cluster CIDR
*   Support node affinity of CIDR blocks
*   Extensible to allow different block sizes allocated to nodes
*   Does not require master or controller restart to add/remove ranges for pods.

### Non-Goals

*   Not providing a generalized IPAM API to Kubernetes. We plan to enhance the
    RangeAllocator’s current behavior (give each Node a /XX from the Cluster
    CIDR as its `PodCIDR`)
*   No change to the default behavior of a Kubernetes cluster.
    *   This will be an optional API and can be disabled (as today’s NodeIPAM
        controllers may also be disabled)

## Proposal

This proposal enhances how pod CIDRs are allocated for nodes by adding a new
CIDR allocator that can be controlled by a new resource 'CIDRRange'. This
enables users to dynamically allocate more IP ranges for pods. In addition, it
gives users the capability to control what ranges are allocated to specific
nodes as well as the size of the pod CIDR allocated to these nodes.

### User Stories

#### Add more pod IPs to the cluster

A user created a cluster with an initial clusterCIDR value of 10.1.0.0/20. Each
node is assigned a /24 pod CIDR so the user could create a maximum of 16 nodes.
However, the cluster needs to be expanded but the user does not have enough IPs
for pods.

With this enhancement, the user can now allocate an additional CIDR for pods;
eg. 10.2.0.0/20 with the same configuration to allocate a /24 pod CIDR. This
way, the cluster can now grow by an additional 16 nodes.

#### Add nodes with higher or lower capabilities

A user created a cluster with an ample sized cluster CIDR. All the initial nodes
are of uniform capacity capable of running a maximum of 256 pods and they are
each assigned a /24 pod CIDR. The user is planning to add more nodes to the
system which are capable of running 500 pods. However, they cannot take
advantage of the additional capacity because all nodes are assigned a /24 pod
CIDR. With this enhancement the user configures a new allocation which uses the
original cluster CIDR but allocates a /23 instead of a /24 to each node. They
use the node selector to allocate these IPs only to the nodes with the higher
capacity.

#### Provision discontiguous ranges

A user wants to create a cluster with 32 nodes each with a capacity to run 256
pods. This means that each node needs a /24 pod CIDR range and they need a total
range of /19. However, there aren't enough contiguous IPs in the user's network.
They can find 4 free ranges of size /21 but no single contiguous /19 range.

Using this enhancement, the user creates 4 different CIDR configurations each
with a /21 range. The CIDR allocator allocates a /24 range from any of these /21
ranges to the nodes and the user can now create the cluster.

### Notes/Constraints/Caveats

This feature does not expand the ability of the NodeIPAM controller to change
the `Node.Spec.PodCIDRs` field. Once that field is set, either by the controller
or a third party, it will be treated as immutable. This is particularly relevant
in situtaitons where users start modifying or deleting the `ClusterCidrConfig`.
Under no circumstances will the controller attempt to revoke the allocated
CIDRs (more details on this are discussed below).

### Risks and Mitigations

-   Racing kube-controller-managers. If multiples of the controller are running
    (as in a HA control plane), how do they coordinate?
    -   The controllers will coordinate using the existing
        kube-controller-manager leader election.

## Design Details

### New Resource

```go
type ClusterCIDRConfig struct {
  metav1.TypeMeta
  metav1.ObjectMeta

  Spec ClusterCIDRConfigSpec
  Status ClusterCIDRConfigStatus
}

type ClusterCIDRConfigSpec {
    // An IP block in CIDR notation ("10.0.0.0/8", "fd12:3456:789a:1::/64")
    // +required
    CIDR string

    // This defines which nodes the config is applicable to. A nil selector can
    // be applied to any node.
    // +optional
    NodeSelector *v1.LabelSelector

    // Netmask size (e.g. 25 -> "/25") to allocate to a node.
    // Users would have to ensure that the kubelet doesn't try to schedule
    // more pods than are supported by the node's netmask (i.e. the kubelet's
    // --max-pods flag)
    // +required
    PerNodeMaskSize int
}

type ClusterCIDRConfigStatus {
  Conditions []metav1.Conditions
}

type ClusterCIDRConfigStatusType string
type ClusterCIDRConfigStatusReason string
var (
  // If the "active" condition is true, this ClusterCIDRConfig can be used by
  // the controller to allocate PodCIDRs for matching nodes.
  ClusterCIDRConfigActive ClusterCIDRConfigStatusType = "active"

  // If the "terminating" condition is true, this ClusterCIDRConfig was deleted
  // by a user and is being garbage collected. When all Nodes using PodCIDRs
  // from this range are deleted, the ClusterCIDRConfig will also be deleted.
  ClusterCIDRConfigTerminating ClusterCIDRConfigStatusType = "terminating"

  // If set as the reason on a false ClusterCIDRConfigActive condition, the
  // ClusterCIDRConfig no longer has any free IP blocks.
  ClusterCIDRConfigExhausted ClusterCIDRConfigStatusReason = "cidr-exhausted"

  // If set as the reason on a true ClusterCIDRConfigTerminating condition,
  // the ClusterCIDRConfig was used to allocate a Node's PodCIDR.
  ClusterCIDRConfigInUse ClusterCIDRConfigStatusReason "cidr-in-use"
)

```

#### Expected Behavior

-   Each node will be assigned up to one range from each `FamilyType`. In case
    of multiple matching ranges, attempt to break ties with the following rules:
    1.  Pick the `ClusterCIDRRange` whose `PerNodeMaskSize` is the fewest IPs.
        For example, `27` (32 IPs) picked before `25` (128 IPs).
    1.  Pick the `ClusterCIDRRange` whose `NodeSelector` matches the most
        labels on the `Node`. For example,
        `{'node.kubernetes.io/instance-type': 'medium', 'rack': 'rack1'}`
        before `{'node.kubernetes.io/instance-type': 'medium'}`.
    1.  Break ties arbitrarily.

-   An empty `NodeSelector` functions as a default that applies to all nodes.
    This should be the fall-back and not take precedence if any other range
    matches. If there are multiple default ranges, ties are broken using the
    scheme outlined above.

-   `CIDR` and `PerNodeMaskSize` are immutable after creation.

-   The controller will add a finalizer to the ClusterCIDRConfig object when it
    is created.
    -   On deletion of the object, make the object have the conditions:
        `[{"active": false}, {"terminating": true}]`.
    -   The controller checks to see if any Nodes are using `PodCIDRs` from this
        range -- if so it adds the `"cidr-in-use"` reason to the `"terminating"`
        condition with a message that lists the nodes using this range.

#### Example: Delete In-use ClusterCIDRConfig

If the user deletes a cluster in use by a node, the object will look like:

```json
"ClusterCIDRConfig": {
  "Metadata": {
    "finalizers": [
      "networking.kubernetes.io/cluster-cidr-config-finalizer",
    ]
  },
  "Status": {
    "Conditions": [
      {
        "Type": "active",
        "Status": "False"
      },
      {
        "Type": "terminating",
        "Status": "True",
        "Reason": "cidr-in-use",
        "Message": "ClusterCIDRRange in use by nodes: ['node1, 'node2']"
      }
    ]
  }
}
```

#### Example: Allocations

```go
[
  {
      // For existing clusters this is the same as ClusterCIDR
      CIDR:            "10.0.0.0/8",
      // Default for nodes not matching any other rule
      NodeSelector:    nil,
      // For existing API this is the same as NodeCIDRMaskSize
      PerNodeMaskSize: 24,
  },
  {
      CIDR:            "172.16.0.0/14",
      // Another range, also allocate-able to any node
      NodeSelector:    nil,
      PerNodeMaskSize: 24,
  },
  {
      CIDR:            "10.0.0.0/8",
      NodeSelector:    { key: "np" op: "IN" value:["np1"] },
      PerNodeMaskSize: 26,
  },
  {
      CIDR:            "192.168.0.0/16",
      NodeSelector:    { key: "np" op: "IN" value:["np2"] },
      PerNodeMaskSize: 26,
  },
  {
      CIDR:            "5.2.0.0/16",
      NodeSelector:    { "np": "np3" },
      PerNodeMaskSize: 20,
  },
  {
      CIDR:            "fd12:3456:789a:1::/64",
      NodeSelector:    { "np": "np3" },
      PerNodeMaskSize: 112,
  },
  ...
]
```

Given the above config, a valid potential configuration might be:

```
{"np": "np1"} --> "10.0.0.0/26"
{"np": "np2"} --> "192.16.0.0/26"
{"np": "np3"} --> "5.2.0.0/20", "fd12:3456:789a:1::/112"
{"np": "np4"} --> "172.16.0.0/24"
```

### Controller

Implement a new
[NodeIPAM controller](https://github.com/kubernetes/kubernetes/tree/master/pkg/controller/nodeipam)
The controller will set up watchers on the `ClusterCIDRConfig` objects and the
`Node` objects.

This controller relies on being a single writer (just as the current NodeIPAM
controller does as well). In the case of HA control planes with multiple
replicas, there will have to be some form of leader election to enforce only 1
active leader. This KEP proposes re-using the kube-controller-manager leader
election to pick a active controller.

#### Data Structures

We will use maps to store the allocated ranges and which node is using the
range. Because the number of nodes is expected to be on the order of thousands,
more sophisticated data structures are likely not required.

Prior investgations [here](https://github.com/kubernetes/kubernetes/pull/90184)
suggest that maps storing allocations will perform well under the number of
nodes we expect.

#### Startup

-   Fetch list of `ClusterCIDRConfig` and build internal data structure
-   If they are set, read the `--cluster-cidr` and `--node-cidr-mask-size` flags
    and attempt to create `ClusterCIDRConfig` with the name
    "created-from-flags". This will be used down the line for migrating users to
    the new allocator.
    -   In the dual-stack case, the flags `--node-cidr-mask-size-ipv4` and
        `--node-cidr-mask-size-ipv6` are used instead, they will also be used as
        necessary.
-   Fetch list of `Node`s. Check each node for `PodCIDRs`
    -   If `PodCIDR` is set, mark the allocation in the internal data structure
        and store this association with the node.
    -   If `PodCIDR` is set, but is not part of one of the tracked
        `ClusterCIDRConfig`, emit a K8s event but do nothing.
    -   If `PodCIDR` is not set, save Node for allocation in the next step.
        After processing all nodes, allocate ranges to any nodes without Pod
        CIDR(s) [Same logic as Node Added event]

#### Reconciliation Loop

This go-routine will watch for cleanup operations and failed allocations and
continue to try them in the background.

For example if a Node can't be allocated a PodCIDR, it will be periodically
retried until it can be allocated a range or it is deleted.

#### Event Watching Loops

##### Node Added

If the Node already has a `PodCIDR` allocated, mark the CIDRs as used.

Otherwise, go through the list of `ClusterCIDRConfig`s and find ranges matching
the node selector from each family. Attempt to allocate Pod CIDR(s) with the
given per-node size. If that `ClusterCIDRConfig` cannot fit a node, search for
another `ClusterCIDRConfig`.

If no `ClusterCIDRConfig` matches the node, or if all matching
`ClusterCIDRConfig`s are full, raise a K8s event and put the Node on the
reconciliation queue (infinite retries). Upon successfully allocating CIDR(s),
update the node object with the podCIDRs.

##### Node Updated

Check that its Pod CIDR(s) match internal allocation.

-   If node.spec.PodCIDRs is already filled up, honor that allocation and mark
    those ranges as allocated.
-   If the node.spec.PodCIDRs is filled with a CIDR not from any
    `ClusterCIDRConfig`, raise a K8sEvent.
-   If the ranges are already marked as allocated for some other node, raise
    another error event (there isn’t an obvious reconciliation step the
    controller can take unilaterally).

##### Node Deleted

Release said Node’s allocation from the internal data-structure.

If this Node is the last one using a particular `ClusterCIDRConfig` that has
been slated for deletion, trigger the deletion flow again (so that the finalizer
is removed and internal data structures are cleaned up).

##### ClusterCIDRConfig Added

Install a finalizer on the `ClusterCIDRConfig` called "networking.kubernetes.io/cluster-cidr-config-finalizer".

Update internal representation of CIDRs to include the new range. Every failed
Node Allocation is stored in a queue, that will be tried again with the new
range by the reconciliation loop.

##### ClusterCIDRConfig Updated

_`CIDR`, and `PerNodeMaskSize` are immutable_

Update any internal state as required.

If the `NodeSelector` changed to include a node that is currently awaiting a
PodCIDR allocation, it will be allocated by the reconiliation loop. 

If the `NodeSelector` changes to unselect a Node currently using that range,
there is no change to the Node's allocation.

##### ClusterCIDRConfig Deleted

1.  Update internal data structures to mark the range as terminating (so new
    nodes won't be added to it)
1.  Change the status of the ClusterCIDRConfig with the following conditions:
    `[{"active": false}, {"terminating": true}]`.
1.  Search the internal representation of the CIDR range to see if any Nodes are
    using the range. If they are, update the `"terminating"` condition with the
    `reason: "cidr-in-use"`.
1.  If there are no nodes using the range, remove the finalizer and cleanup all
    internal state.

### kube-controller-manager

The flag `--cidr-allocator-type` will be ammended to include a new type
"ClusterCIDRConfigAllocator".

The list of current valid types is
[here](https://github.com/kubernetes/kubernetes/blob/1ff18a9c43f59ffed3b2d266b31e0d696d04eaff/pkg/controller/nodeipam/ipam/cidr_allocator.go#L38).

### Test Plan

#### Unit Tests and Benchmarks

-   Ensure that the controller scales to ~5,000 nodes -- memory usage and reasonable
    allocation times

#### Integration Tests

-   Verify finalizers and statuses are persisted appropriately
-   Test watchers

#### End-to-End Tests

-   Run through some sample workflows. Just a few for example:
    -   Adding a node
    -   Adding a ClusterCIDRConfig
    -   Deleting a ClusterCIDRConfig that is in use
-   Run through the [user stories](#user-stories):
    -   Expand the ClusterCIDR (existing nodes without alloations are
        allocated and new nodes also get ranges.
    -   Use `NodeSelector` to allocate different sized CIDRs to different nodes.
    -   Create and use discontiguous ranges.

### Graduation Criteria

#### Alpha -> Beta Graduation

-   Gather feedback from users about any issues
-   Tests are in testgrid

#### Beta -> GA Graduation

-   Wait for 1 release to receive any additional feedback

#### Make the Controller the new default

After the GA graduation, change the default NodeIPAM allocator from
RangeAllocator to ClusterCIDRConfigAllocator. This will involve changing the
default value of the flag on the kube-controller-manager
(`--cidr-allocator-type`).

#### **TBD:** Mark the RangeAllocator as deprecated

In the same release that the ClusterCIDRConfigAllocator is made the default,
mark the RangeAllocator as deprecated.

After 2 releases, the code can be removed.

### Upgrade / Downgrade Strategy

#### Upgrades

There is no change to the defaults as part of the alpha, so existing clusters
will upgrade seemlessly.

If we want to use the new controller, users will have to change the
`--cidr-allocator-type` flag on the kube-controller-manager. The new controller
will respect the existing flags for `--cluster-cidr` and
`--node-cidr-mask-size`.

Users will also have to change the kube-proxy flags as outlined in [KEP
2450](https://github.com/kubernetes/enhancements/tree/master/keps/sig-network/2450-Remove-knowledge-of-pod-cluster-CIDR-from-iptables-rules).
The flag `--detect-local-mode` must be set to `NodeCIDR` to properly handle
nodes having discontiguous Pod CIDRs.

TODO: Should we change the default value of the flag in the kube-proxy?

#### Downgrades

Customers may "downgrade" by switching back the `--cidr-allocator-type` to
"RangeAllocator". The Node `PodCIDR` allocations will persist even after the
downgrade, and because the same flags are in use, the old allocator should work
for new nodes as well.

TODO: The `ClusterCIDRRange` objects may become un-deleteable as no controller
will remove the finalizer.

### Version Skew Strategy

<!--
If applicable, how will the component handle version skew with other
components? What are the guarantees? Make sure this is in the test plan.

Consider the following in developing a version skew strategy for this
enhancement:
- Does this enhancement involve coordinating behavior in the control plane and
  in the kubelet? How does an n-2 kubelet without this feature available behave
  when this feature is used?
- Will any other components on the node change? For example, changes to CSI,
  CRI or CNI may require updating that component before the kubelet.
-->

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
-->

-   [ ] Feature gate (also fill in values in `kep.yaml`)
    -   Feature gate name:
    -   Components depending on the feature gate:
-   [ ] Other
    -   Describe the mechanism:
    -   Will enabling / disabling the feature require downtime of the control
        plane?
    -   Will enabling / disabling the feature require downtime or reprovisioning
        of a node? (Do not assume `Dynamic Kubelet Config` feature is enabled).

###### Does enabling the feature change any default behavior?

<!--
Any change of default behavior may be surprising to users or break existing
automations, so be extremely careful here.
-->

###### Can the feature be disabled once it has been enabled (i.e. can we roll back the enablement)?

<!--
Describe the consequences on existing workloads (e.g., if this is a runtime
feature, can it break the existing applications?).

NOTE: Also set `disable-supported` to `true` or `false` in `kep.yaml`.
-->

###### What happens if we reenable the feature if it was previously rolled back?

###### Are there any tests for feature enablement/disablement?

<!--
The e2e framework does not currently support enabling or disabling feature
gates. However, unit tests in each component dealing with managing data, created
with and without the feature, are necessary. At the very least, think about
conversion tests if API types are being modified.
-->

### Rollout, Upgrade and Rollback Planning

<!--
This section must be completed when targeting beta to a release.
-->

###### How can a rollout fail? Can it impact already running workloads?

<!--
Try to be as paranoid as possible - e.g., what if some components will restart
mid-rollout?
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

<!--
Ideally, this should be a metric. Operations against the Kubernetes API (e.g.,
checking if there are objects with field X set) may be a last resort. Avoid
logs or events for this purpose.
-->

###### What are the SLIs (Service Level Indicators) an operator can use to determine the health of the service?

<!--
Pick one more of these and delete the rest.
-->

-   [ ] Metrics
    -   Metric name:
    -   [Optional] Aggregation method:
    -   Components exposing the metric:
-   [ ] Other (treat as last resort)
    -   Details:

###### What are the reasonable SLOs (Service Level Objectives) for the above SLIs?

<!--
At a high level, this usually will be in the form of "high percentile of SLI
per day <= X". It's impossible to provide comprehensive guidance, but at the very
high level (needs more precise definitions) those may be things like:
  - per-day percentage of API calls finishing with 5XX errors <= 1%
  - 99% percentile over day of absolute value from (job creation time minus expected
    job creation time) for cron job <= 10%
  - 99,9% of /health requests per day finish with 200 code
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

###### Will enabling / using this feature result in introducing new API types?

<!--
Describe them, providing:
  - API type
  - Supported number of objects per cluster
  - Supported number of objects per namespace (for namespace-scoped objects)
-->

###### Will enabling / using this feature result in any new calls to the cloud provider?

<!--
Describe them, providing:
  - Which API(s):
  - Estimated increase:
-->

###### Will enabling / using this feature result in increasing size or count of the existing API objects?

<!--
Describe them, providing:
  - API type(s):
  - Estimated increase in size: (e.g., new annotation of size 32B)
  - Estimated amount of new objects: (e.g., new Object X for every existing Pod)
-->

###### Will enabling / using this feature result in increasing time taken by any operations covered by existing SLIs/SLOs?

<!--
Look at the [existing SLIs/SLOs].

Think about adding additional work or introducing new steps in between
(e.g. need to do X to start a container), etc. Please describe the details.

[existing SLIs/SLOs]: https://git.k8s.io/community/sig-scalability/slos/slos.md#kubernetes-slisslos
-->

###### Will enabling / using this feature result in non-negligible increase of resource usage (CPU, RAM, disk, IO, ...) in any components?

<!--
Things to keep in mind include: additional in-memory state, additional
non-trivial computations, excessive access to disks (including increased log
volume), significant amount of data sent and/or received over network, etc.
This through this both in small and large cases, again with respect to the
[supported limits].

[supported limits]: https://git.k8s.io/community//sig-scalability/configs-and-limits/thresholds.md
-->

### Troubleshooting

<!--
This section must be completed when targeting beta to a release.

The Troubleshooting section currently serves the `Playbook` role. We may consider
splitting it into a dedicated `Playbook` document (potentially with some monitoring
details). For now, we leave it here.
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

### Share Resources with Service API

There have also been discussions about updating the service API to have multiple
ranges. One proposal is to share a common `CIDRRange` resource between both
APIs.

The potential for divergence between Service CIDRs and Pod CIDRs is quite high,
as discussed in the cons section below.

```
CIDRRange {
  Type      CIDRType
  CIDR string # Example "10.0.0.0/8" or "fd12:3456:789a:1::/64"
  Selector  v1.LabelSelector # Specifies which Services or Nodes can be
                             # assigned IPs from this block.
  BlockSize string # How large of an IP block to allocate. For services
                   # this would always be "/32". Example "/24"
}

var (
  ServiceCIDR CIDRType = "service"
  ClusterCIDR CIDRType = "cluster"
)
```

#### Pros

-   First-party resource to allow editing of ClusterCIDR or ServiceCIDR without
    cluster restart
-   Single IPAM resource for K8s. Potentially extensible for more use cases down
    the line.

#### Cons

-   Need a strategy for supporting divergence of Service and NodeIPAM APIs in
    the future.
    -   Already BlockSize feels odd, as Service will not make use of it.
-   Any differences in how Service treats an object vs how NodeIPAM treats an
    object are likely to cause confusion.
    -   Enforce API level requirements across multiple unrelated controllers

### Nodes Register CIDR Request

Nodes might register a request for CIDR (as a K8s resource). The NodeIPAM
controllers would watch this resource and attempt to fulfill these requests.

The major goals behind this design is to provide more flexibility in IPAM.
Additionally, it ensures that nodes ask for what they need and users don’t need
to ensure that the ClusterCIDRRange and the Node’s `--max-pods` value are in
alignment.

A major factor in not recommending this strategy is the increased complexity to
Kubernetes’ IPAM model. One of the stated non-goals was that this proposal
doesn’t seek to provide a general IPAM solution or to drastically change how
Kubernetes does IPAM.

Adn thend there was this whole thing where people said hey can we have linebreaks 

```
NodeCIDRRequest {
  NodeName  string # Name of node requesting the CIDR
  RangeSize string # Example "/24"
  CIDR      string # Populated by some IPAM controller. Example: "10.2.0.0/24"
}
```

#### Pros

-   Because the node is registering its request, it can ensure that it is asking
    for enough IPs to cover its `--max-pods` value.
-   Added flexibility to support different IPAM models:
    -   Example: Nodes can request additional Pod IPs on the fly. This can help
        address customer requests for centralized IP handling as opposed to
        assigning them as chunks.

#### Cons

-   Requires changes to the kubelet in addition to change to NodeIPAM controller
    -   Kubelet needs to register the requests
-   Potentially more confusing API.
-   _Minor: O(nodes) more objects in etcd. Could be thousands in large
    clusters._
