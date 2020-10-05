# RuntimeClass Scheduling

## Table of Contents

<!-- toc -->
- [Summary](#summary)
- [Motivation](#motivation)
  - [Goals](#goals)
  - [Non-Goals](#non-goals)
- [Proposal](#proposal)
  - [User Stories](#user-stories)
    - [Windows](#windows)
    - [Sandboxed Nodes](#sandboxed-nodes)
- [Design Details](#design-details)
  - [API](#api)
  - [RuntimeClass Admission Controller](#runtimeclass-admission-controller)
  - [Labeling Nodes](#labeling-nodes)
  - [Graduation Criteria](#graduation-criteria)
- [Implementation History](#implementation-history)
- [Alternatives](#alternatives)
  - [Scheduler](#scheduler)
  - [RuntimeController Mix-in](#runtimecontroller-mix-in)
    - [RuntimeController](#runtimecontroller)
    - [Mix-in](#mix-in)
  - [NodeSelector](#nodeselector)
  - [Native RuntimeClass Reporting](#native-runtimeclass-reporting)
  - [Scheduling Policy](#scheduling-policy)
<!-- /toc -->

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
- Defining official or reserved label or taint schemas or for RuntimeClasses.

## Proposal

A new optional `Scheduling` structure will be added to the RuntimeClass API. The
scheduling struct includes both a `NodeSelector` and `Tolerations` that control
how a pod using that RuntimeClass is scheduled. The NodeSelector rules are
applied during scheduling, but the Tolerations are added to the PodSpec during
admission by the new RuntimeClass admission controller.

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

[Windows nodes]: ../../sig-windows/20190103-windows-node-support.md

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

The RuntimeClass definition is augmented with an optional `Scheduling` struct:

```go
type Scheduling struct {
    // nodeSelector lists labels that must be present on nodes that support this
    // RuntimeClass. Pods using this RuntimeClass can only be scheduled to a
    // node matched by this selector. The RuntimeClass nodeSelector is merged
    // with a pod's existing nodeSelector. Any conflicts will cause the pod to
    // be rejected in admission.
    // +optional
    NodeSelector map[string]string

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
a NodeSelectorTerm.

Since the RuntimeClass scheduling rules represent hard requirements (the node
supports the RuntimeClass or it doesn't), the scheduling API should not include
preferences, ruling out NodeAffinity. The NodeSelector type is much more
expressive than the `map[string]string` selector, but the top-level union logic
makes merging NodeSelectors messy (requires a cross-product). For simplicity,
we went with the simple requirements.

[NodeAffinity]: https://kubernetes.io/docs/concepts/configuration/assign-pod-node/#affinity-and-anti-affinity

**Tolerations**

While NodeSelectors and labels are used for steering pods _towards_ nodes,
[taints and tolerations][taint-and-toleration] are used for steering pods _away_
from nodes. If every pod had a RuntimeClass and every RuntimeClass had a strict
NodeSelector, then RuntimeClasses could use non-overlapping selectors in place
of taints & tolerations. However the same could be said of regular pod
selectors, yet taints & tolerations are still a useful addition. Examples of
[use cases](#user-stories) for including tolerations in RuntimeClass scheduling
inculde:

- Tainting Windows nodes with `windows.microsoft.com` to keep default linux pods
  away from the nodes. Windows RuntimeClasses would then need a corresponding
  toleration.
- Tainting "sandbox" nodes with `sandboxed.kubernetes.io` to keep services
  providing privileged node features away from sandboxed workloads. Sandboxed
  RuntimeClasses would need a toleration to enable them to be run on those
  nodes.

### RuntimeClass Admission Controller

The RuntimeClass admission controller is a new default-enabled in-tree admission
plugin. The role of the controller for scheduling is to merge the scheduling
rules from the RuntimeClass into the PodSpec. Eventually, the controller's
responsibilities may grow, such as to merge in [pod overhead][] or validate
feature compatibility.

Merging the RuntimeClass NodeSelector into the PodSpec NodeSelector is handled
by adding the key-value pairs from the RuntimeClass version to the PodSpec
version. If both have the same key with a different value, then the admission
controller will reject the pod.

Merging tolerations is straight forward, as we want to _union_ the RuntimeClass
tolerations with the existing tolerations, which matches the default toleration
composition logic. This means that RuntimeClass tolerations can simply be
appended to the existing tolerations, but an [existing
utilty][merge-tolerations] can reduce duplicates by merging equivalent
tolerations.

If the pod's referenced RuntimeClass does not exist, the admission controller
will reject the pod. This is necessary to ensure the pod is run with the
expected behavior.

[merge-tolerations]: https://github.com/kubernetes/kubernetes/blob/58021216b16ae6d5f24fb1c32ab541b2e79a365e/pkg/util/tolerations/tolerations.go#L62
[TaintBasedEvictions]: https://kubernetes.io/docs/concepts/configuration/taint-and-toleration/#taint-based-evictions

### Labeling Nodes

Node labeling & tainting is outside the scope of this proposal or feature. How
to label nodes is very environment dependent. Here are several examples:

- If runtimes are setup as part of node setup, then the node template should
  include the appropriate labels & taints.
- If runtimes are installed through a DaemonSet, then the scheduling should match
  that of the DaemonSet.
- If runtimes are manually installed, or installed through some external
  process, that same process should apply an appropriate label to the node.

If the RuntimeClass scheduling rules have security implications, special care
should be taken when choosing labels. In particular, labels with the
`[*.]node-restriction.kubernetes.io/` prefix cannot be added with the node's
identity, and labels with the `[*.]k8s.io/` or `[*.]kubernetes.io/` prefixes
cannot be modified by the node. For more details, see [Bounding Self-Labeling
Kubelets](../sig-auth/0000-20170814-bounding-self-labeling-kubelets.md)

### Graduation Criteria

This feature will be rolled into RuntimeClass beta in v1.15, thereby skipping
the alpha phase. This means the feature is expected to be beta quality at launch:

- Thorough testing, including unit, integration and e2e coverage.
- Thoroughly documented (as an extension to the [RuntimeClass documentation][]).

[RuntimeClass documentation]: https://kubernetes.io/docs/concepts/containers/runtime-class/

## Alternatives

### Scheduler

A new scheduler predicate could manage the RuntimeClass scheduling. It would
lookup the RuntimeClass associated with the pod being scheduled. If there is no
RuntimeClass, or the RuntimeClass does not include a scheduling struct, then the
predicate would permit the pod to be scheduled to the evaluated node. Otherwise,
it would check whether the NodeSelector matches the node.

Adding a dedicated RuntimeClass predicate rather than mixing the rules in to the
NodeAffinity evaluation means that in the event a pod is unschedulable there
would be a clear explanation of why. For example:

```
0/10 Nodes are available: 5 nodes do not have enough memory, 5 nodes don't match RuntimeClass
```

If the pod's referenced RuntimeClass does not exist at scheduling time, the
RuntimeClass predicate would fail. The scheduler would periodically retry, and
once the RuntimeClass is (re)created, the pod would be scheduled.

### RuntimeController Mix-in

Rather than resolving scheduling in the scheduler, the `NodeSelectorTerm`
rules and `Tolerations` are mixed in to the PodSpec. The mix-in happens in the
mutating admission phase, and is performed by a new `RuntimeController` built-in
admission plugin. The same admission controller is shared with the [Pod
Overhead][] proposal.

[Pod Overhead]: https://github.com/kubernetes/enhancements/pull/887

#### RuntimeController

RuntimeController is a new in-tree admission plugin that should eventually be
enabled on almost every Kubernetes clusters. The role of the controller for
scheduling is to merge the scheduling constraints from the RuntimeClass into the
PodSpec. Eventually, the controller's responsibilities may grow, such as to
merge in [pod overhead][] or validate feature compatibility.

Note that the RuntimeController is not needed if we implement [native scheduler
support](#runtimeclass-aware-scheduling).

#### Mix-in

The RuntimeClass scheduling rules are merged with the pod's NodeSelector &
Tolerations.

**NodeSelectorRequirements**

To avoid multiplicitive scaling of the NodeSelectorTerms, the
`RuntimeClass.Scheduling.NodeSelector *v1.NodeSelector` field is replaced with
`NodeSelectorTerm *v1.NodeSelectorTerm`.

The term's NodeSelectorRequirements are merged into the pod's node affinity
scheduling requirements:

```
pod.spec.affinity.nodeAffinity.requiredDuringSchedulingIgnoredDuringExecution.nodeSelectorTerms[*].matchExpressions
```

Since the `requiredDuringSchedulingIgnoredDuringExecution` NodeSelectorTerms are
unioned (OR'd), intersecting the RuntimeClass's NodeSelectorTerm means
the requirements need to be appended to _every_ NodeSelectorTerm.

**Tolerations**

Merging tolerations is much simpler as we want to _union_ the RuntimeClass
tolerations with the existing tolerations, which matches the default toleration
composition logic. This means that RuntimeClass tolerations can simply be
appended to the existing tolerations, but an [existing
utilty][merge-tolerations] can reduce duplicates by merging equivalent
tolerations.

[merge-tolerations]: https://github.com/kubernetes/kubernetes/blob/58021216b16ae6d5f24fb1c32ab541b2e79a365e/pkg/util/tolerations/tolerations.go#L62

### NodeSelector

Replacing the NodeSelector's `[]NodeSelectorRequirements` type with the
PodSpec's label `map[string]string` approach greatly simplifies the merging
logic, but sacrifices a lot of flexibliity. For exameple, the operator in
NodeSelectorRequriments enables selections like:

- Negative selection, such as "operating system is _not_ windows"
- Numerical comparison, such as "runc version is _at least_ X" (although it doesn't currently support semver)
- Set selection, such as "sandbox is _one of_ kata-cotainers or gvisor"

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
nodes that match a RuntimeClasses scheduling rules.

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
