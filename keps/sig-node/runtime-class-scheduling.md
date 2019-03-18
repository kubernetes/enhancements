---
title: RuntimeClass Scheduling
authors:
  - "@tallclair"
owning-sig: sig-node
participating-sigs:
  - sig-scheduling
reviewers:
  - yastij
approvers:
  - bsalamat
  - dchen1107
  - derekwaynecarr
creation-date: 2019-03-14
status: provisional
see-also:
  - "/keps/sig-node/runtime-class.md"
replaces:
  - [RuntimeClass Scheduling Brainstorm](https://docs.google.com/document/d/1W51yBNTvp0taeEss56GTk8jczqFJ2d6jBeN6sCSlYZU/edit#)
---

# RuntimeClass Scheduling

## Table of Contents

  * [Summary](#summary)
  * [Motivation](#motivation)
    * [Goals](#goals)
    * [Non\-Goals](#non-goals)
  * [Proposal](#proposal)
    * [User Stories](#user-stories)
      * [Windows](#windows)
      * [Sandboxed Nodes](#sandboxed-nodes)
  * [Design Details](#design-details)
    * [API](#api)
    * [RuntimeController](#runtimecontroller)
      * [Mix\-in](#mix-in)
    * [Labeling Nodes](#labeling-nodes)
    * [Graduation Criteria](#graduation-criteria)
  * [Implementation History](#implementation-history)
  * [Alternatives](#alternatives)
    * [NodeSelector](#nodeselector)
    * [RuntimeClass\-aware scheduling](#runtimeclass-aware-scheduling)
    * [Native RuntimeClass Reporting](#native-runtimeclass-reporting)
    * [Scheduling Policy](#scheduling-policy)

## Summary

RuntimeClass scheduling enables native support for heterogeneous clusters where
every node does not necessarily support every RuntimeClass. This feature allows
pod authors to select a RuntimeClass without needing to worry about cluster
topology.

## Motivation

In the initial RuntimeClass implementation, we explicitly assumed that the
cluster nodes were homogenous with regards to RuntimeClasses. It was still
possible to run a heterogeneous cluster, but pod authors would need to set
appropriate [NodeSelector][] rules and [tolerations][taint-and-toleration] to
ensure the pods landed on supporting nodes.

As [use cases](#user-stories) have appeared and solidified, it has become clear
that heterogeneous clusters will not be uncommmon, and supporting a smoother
user experience will be valuable.

[NodeSelector]: https://kubernetes.io/docs/concepts/configuration/assign-pod-node/
[taint-and-toleration]: https://kubernetes.io/docs/concepts/configuration/taint-and-toleration/

### Goals

- Pods using a RuntimeClass that is not supported by all nodes in a cluster are
  automatically scheduled to nodes that support that RuntimeClass.
- RuntimeClass scheduling is compatible with other scheduling constraints. For
  example, a pod with a node selector for GPUs and a Linux runtime should be
  scheduled to a linux node with GPUs (an intersection).

### Non-Goals

- Replacing [SchedulingPolicy](#scheduling-policy) with RuntimeClasses.

The following are **currently** out of scope, but _may_ be revisited at a later
date.

- Automatic topology discovery or node labeling
- Automatically selecting a RuntimeClass for a pod based on node availability.
- Defining official or revserved label or taint schemas or for RuntimeClasses.

## Proposal

A new optional `Topology` structure will be added to the RuntimeClass API. The
topology includes both `NodeSelectorTerm` rules and `Tolerations` that are mixed
in to a pod using that RuntimeClass. The mix-in happens in the mutating
admission phase, and is performed by a new `RuntimeController` built-in
admission plugin. The same admission controller is shared with the [Pod
Overhead][] proposal.

[Pod Overhead]: https://github.com/kubernetes/enhancements/pull/887

### User Stories

#### Windows

The introduction of [Windows nodes][] presents an immediate use case for
heterogeneous clusters, where some nodes are running Windows, and some
linux. From the inherent differences in the operating systems, it is natural
that each will support a different set of runtimes. For example, Windows nodes
may support Hyper-V sandboxing, while linux nodes support Kata-containers. Even
native container support varies on each, with runc for Linux and runhcs for
Windows.

- As a **cluster administrator** I want to enable different runtimes on Windows
  and Linux nodes.
- As a **developer** I want to select a Windows runtime without worrying about
  scheduling constraints.
- As a **developer** I want to ensure my Linux workloads are not accidentally
  scheduled to windows nodes.

[Windows nodes]: https://github.com/kubernetes/enhancements/blob/master/keps/sig-windows/20190103-windows-node-support.md

#### Sandboxed Nodes

Some users wish to keep sandbox workloads and native workloads separate. For
example, a node running untrusted sandboxed workloads may have stricter
requirements about which trusted services are run on that node.

- As a **cluster administrator** I want to ensure that untrusted workloads are
  not colocated with sensitive data.
- As a **developer** I want run an untrusted service without worrying about
  where the service is running.
- As a **cluster administrator** I want to autoscale trusted and untrusted nodes
  independently.

## Design Details

### API

The RuntimeClass definition is augmented with an optional `Topology` struct:

```go
type Topology struct {
    // nodeSelectorTerm selects the set of nodes that support this RuntimeClass.
    // Pods using this RuntimeClass can only be scheduled to a node matched by this selector.
    // The nodeSelectorTerm is merged with a pod's other node affinity match
    // expressions by appending the additional requirements to each preexisting
    // NodeSelectorTerm.
    // +optional
    NodeSelectorTerm []corev1.NodeSelectorRequirement

    // tolerations adds tolerations to pods running with this RuntimeClass.
    // +optional
    Tolerations []corev1.Toleration
}
```

**NodeSelector vs. NodeAffinity vs. NodeSelectorRequirement**

The PodSpec's `NodeSelector` is a label `map[string]string` that must exactly
match a subset of node labels. [NodeAffinity][] is a much more complex and
expressive set of requirements and preferences. NodeSelectorRequirements are a
small subset of the NodeAffinity rules, that place intersecting requirements on
a NodeSelectorTerm. There is no way to logically intersect NodeAffinities, and
intersecting the NodeAffinity.NodeSelector multiplies the number of terms. Using
[native scheduler support](#runtimeclass-aware-scheduling) greatly simplifies
the intersection logic.

NodeSelectorRequirements fall in the sweet spot of being expressive enough to
represent a variety of topologies while still being logically intersectable with
other pod scheduling constraints. Including more advanced scheduling preferences
for a RuntimeClass is a use case that is better suited to
[SchedulingPolicy](#scheduling-policy).

[NodeAffinity]: https://kubernetes.io/docs/concepts/configuration/assign-pod-node/#affinity-and-anti-affinity

**Tolerations**

While NodeSelectors and labels are used for steering pods _towards_ nodes,
[taints and tolerations][taint-and-toleration] are used for steering pods _away_
from nodes. If every pod had a RuntimeClass and every RuntimeClass had a strict
NodeSelector, then RuntimeClasses could use non-overlapping selectors in place
of taints & tolerations. However the same could be said of regular pod
selectors, yet taints & tolerations are still a useful addition. Examples of
[use cases](#user-stories) for including tolerations in RuntimeClass topology
inculde:

- Tainting Windows nodes with `windows.microsoft.com` to keep default linux pods
  away from the nodes. Windows RuntimeClasses would then need a corresponding
  toleration.
- Tainting "sandbox" nodes with `sandboxed.kubernetes.io` to keep services
  providing privileged node features away from sandboxed workloads. Sandboxed
  RuntimeClasses would need a toleration to enable them to be run on those
  nodes.

### RuntimeController

RuntimeController is a new in-tree admission plugin that should eventually be
enabled on almost every Kubernetes clusters. The role of the controller for
scheduling is to merge the topology constraints from the RuntimeClass into the
PodSpec. Eventually, the controller's responsibilities may grow, such as to
merge in [pod overhead][] or validate feature compatibility.

Note that the RuntimeController is not needed if we implement [native scheduler
support](#runtimeclass-aware-scheduling).

#### Mix-in

The RuntimeClass topology is merged with the pod's NodeSelector & Tolerations.

**NodeSelectorRequirements**

NodeSelectorRequirements are merged into the pod's node affinity scheduling requirements:
```
pod.spec.affinity.nodeAffinity.requiredDuringSchedulingIgnoredDuringExecution[*].matchExpressions
```

Since the `requiredDuringSchedulingIgnoredDuringExecution` NodeSelectorTerms are
unioned (OR'd), intersecting the RuntimeClass's NodeSelectorRequirements means
the requirements need to be appended to _every_ NodeSelectorTerm.

**Tolerations**

Merging tolerations is much simpler as we want to _union_ the RuntimeClass
tolerations with the existing tolerations, which matches the default toleration
composition logic. This means that RuntimeClass tolerations can simply be
appended to the existing tolerations, but an [existing
utilty][merge-tolerations] can reduce duplicates by merging equivalent
tolerations.

[merge-tolerations]: https://github.com/kubernetes/kubernetes/blob/58021216b16ae6d5f24fb1c32ab541b2e79a365e/pkg/util/tolerations/tolerations.go#L62

### Labeling Nodes

Node labeling & tainting is outside the scope of this proposal or feature. How
to label nodes is very environment depnedent. Here are several examples:

- If runtimes are setup as part of node setup, then the node template should
  include the appropriate labels & taints.
- If runtimes are installed through a DaemonSet, then the topology should match
  that of the DaemonSet.
- If runtimes are manually installed, or installed through some external
  process, that same process should apply an appropriate label to the node.

### Graduation Criteria

This feature will be rolled into RuntimeClass beta in v1.15, thereby skipping
the alpha phase. This means the feature is expected to be beta quality at launch:

- Thorough testing, including unit, integration and e2e coverage.
- Thoroughly documented (as an extension to the [RuntimeClass documentation][]).

[RuntimeClass documentation]: https://kubernetes.io/docs/concepts/containers/runtime-class/

## Implementation History

- 2019-03-14: Initial KEP published.
- 2018-10-05: [RuntimeClass Scheduling
  Brainstorm](https://docs.google.com/document/d/1W51yBNTvp0taeEss56GTk8jczqFJ2d6jBeN6sCSlYZU/edit#)
  published.

## Alternatives

### NodeSelector

Replacing the NodeSelector's `[]NodeSelectorRequirements` type with the
PodSpec's label `map[string]string` approach greatly simplifies the merging
logic, but sacrifices a lot of flexibliity. For exameple, the operator in
NodeSelectorRequriments enables selections like:

- Negative selection, such as "operating system is _not_ windows"
- Numerical comparison, such as "runc version is _at least_ X" (although it doesn't currently support semver)
- Set selection, such as "sandbox is _one of_ kata-cotainers or gvisor"

### RuntimeClass-aware scheduling

Rather than the proposed mix-in approach, the scheduler could have built in
support for RuntimeClasses. When a pod is to be scheduled, the scheduler plugin
would fetch the pod's RuntimeClass and determine which nodes are supported.

**Advantages:**

- Native scheduler support is required if we want to intersect more advanced
  scheduling constraints, such as NodeAffinities.
- Native scheduler support is required if we ever want a feedback loop between
  the scheduling decision and the RuntimeClass selection. That is, if we want to
  select a RuntimeClass based on where there is availibility.
- Doesn't "pollute" the PodSpec with the RuntimeClass scheduling constraints.

**Disadvantages:**

- Adds complexity to the scheduler, which now needs to be able to lookup
  RuntimeClasses and take those decisions into account.
- Scheduling decision is more opaque, as there can be hidden constraints on the
  RuntimeClass preventing a pod from being schedulable.

Moving forward with the proposed mixin approach does not rule out native
scheduler support in the future. If we decided the advantages outweigh the
disadvantages in the future, we could seemlessly migrate to this approach
without any action being required on the user side.

### Native RuntimeClass Reporting

Rather than relying on labels to stear RuntimeClasses to supporting nodes, nodes
could directly list supported RuntimeClasses (or RuntimeHandlers) in their
status. Taking this approach would require native RuntimeClass-aware scheduling.

**Advantages:**

- RuntimeClass support is more explicit: it is easier to look at a node and see
  which runtimes it supports.

**Disadvantages:**

- Larger change and more complexity: this requires modifying the node API and
  introducing a new scheduling mechanism.
- Less flexible: the existing scheduling mechanisms have been carefully thought
  out and designed, and are extremely flexible to supporting a wide range of
  topologies. Simple 1:1 matching would lose a lot of this flexibility.

The visibility advantage could be achieved through different methods. For
example, a special request or tool could be implemented that would list all the
nodes that match a RuntimeClasess topology.

### Scheduling Policy

Rather than building scheduling support into RuntimeClass, we could build
RuntimeClass support into [SchedulingPolicy][]. For example, a scheduling
policy that places scheduling constraints on pods that use a particular
RuntimeClass.

**Advantages:**

- A more generic system, no special logic needed for RuntimeClasses.
- Scheduling constraints for correlated RuntimeClasses could be grouped together
  (e.g. linux scheduling constraints for all linux RuntimeClasses).

**Disadvantages:**

- Separating the scheduling policy into a separate object means a less direct
  user experience. The cluster administrator needs to setup 2 different
  resources for each RuntimeClass, and the developer needs to look at 2
  different resources to understand the full implications of choosing a
  particular RuntimeClass.

For the same reason that RuntimeClass scheduling is compatible with additional
pod scheduling constraints, it should also be compatible with additional
scheduling policies.

[SchedulingPolicy]: https://github.com/kubernetes/enhancements/pull/683
