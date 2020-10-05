# KEP-585: Runtime Class

## Table of Contents

<!-- toc -->
- [Summary](#summary)
- [Motivation](#motivation)
  - [Goals](#goals)
  - [Non-Goals](#non-goals)
  - [User Stories](#user-stories)
- [Proposal](#proposal)
  - [API](#api)
    - [Examples](#examples)
    - [Runtime Handler](#runtime-handler)
  - [Versioning, Updates, and Rollouts](#versioning-updates-and-rollouts)
  - [Implementation Details](#implementation-details)
    - [Monitoring](#monitoring)
  - [Risks and Mitigations](#risks-and-mitigations)
- [RuntimeClass Scheduling](#runtimeclass-scheduling)
  - [RuntimeClass Scheduling Motivation](#runtimeclass-scheduling-motivation)
    - [RuntimeClass Scheduling Goals](#runtimeclass-scheduling-goals)
    - [RuntimeClass Scheduling Non-Goals](#runtimeclass-scheduling-non-goals)
  - [RuntimeClass Scheduling Proposal](#runtimeclass-scheduling-proposal)
    - [RuntimeClass Scheduling User Stories](#runtimeclass-scheduling-user-stories)
      - [Windows](#windows)
      - [Sandboxed Nodes](#sandboxed-nodes)
  - [Design Details](#design-details)
    - [RuntimeClass Scheduling API](#runtimeclass-scheduling-api)
    - [RuntimeClass Admission Controller](#runtimeclass-admission-controller)
    - [Labeling Nodes](#labeling-nodes)
    - [RuntimeClass Scheduling Graduation Criteria](#runtimeclass-scheduling-graduation-criteria)
  - [RuntimeClass Scheduling Alternatives](#runtimeclass-scheduling-alternatives)
    - [Scheduler](#scheduler)
    - [RuntimeController Mix-in](#runtimecontroller-mix-in)
      - [RuntimeController](#runtimecontroller)
      - [Mix-in](#mix-in)
    - [NodeSelector](#nodeselector)
    - [Native RuntimeClass Reporting](#native-runtimeclass-reporting)
    - [Scheduling Policy](#scheduling-policy)
- [Graduation Criteria](#graduation-criteria)
- [Implementation History](#implementation-history)
- [Appendix](#appendix)
  - [Proposed Future Enhancements](#proposed-future-enhancements)
  - [Examples of runtime variation](#examples-of-runtime-variation)
<!-- /toc -->

## Summary

`RuntimeClass` is a new cluster-scoped resource that surfaces container runtime properties to the
control plane. RuntimeClasses are assigned to pods through a `runtimeClass` field on the
`PodSpec`. This provides a new mechanism for supporting multiple runtimes in a cluster and/or node.

## Motivation

There is growing interest in using different runtimes within a cluster. [Sandboxes][] are the
primary motivator for this right now, with both Kata containers and gVisor looking to integrate with
Kubernetes. Other runtime models such as Windows containers or even remote runtimes will also
require support in the future. RuntimeClass provides a way to select between different runtimes
configured in the cluster and surface their properties (both to the cluster & the user).

In addition to selecting the runtime to use, supporting multiple runtimes raises other problems to
the control plane level, including: accounting for runtime overhead, scheduling to nodes that
support the runtime, and surfacing which optional features are supported by different
runtimes. See [RuntimeClass Scheduling](#runtimeclass-scheduling) for information about scheduling.

[Sandboxes]: https://docs.google.com/document/d/1QQ5u1RBDLXWvC8K3pscTtTRThsOeBSts_imYEoRyw8A/edit

### Goals

- Provide a mechanism for surfacing container runtime properties to the control plane
- Support multiple runtimes per-cluster, and provide a mechanism for users to select the desired
  runtime

### Non-Goals

- RuntimeClass is NOT RuntimeComponentConfig.
- RuntimeClass is NOT a general policy mechanism.
- RuntimeClass is NOT "NodeClass". Although different nodes may run different runtimes, in general
  RuntimeClass should not be a cross product of runtime properties and node properties.

### User Stories

- As a cluster operator, I want to provide multiple runtime options to support a wide variety of
  workloads. Examples include native linux containers, "sandboxed" containers, and windows
  containers.
- As a cluster operator, I want to provide stable rolling upgrades of runtimes. For
  example, rolling out an update with backwards incompatible changes or previously unsupported
  features.
- As an application developer, I want to select the runtime that best fits my workload.
- As an application developer, I don't want to study the nitty-gritty details of different runtime
  implementations, but rather choose from pre-configured classes.
- As an application developer, I want my application to be portable across clusters that use similar
  but different variants of a "class" of runtimes.

## Proposal

The initial design includes:

- `RuntimeClass` API resource definition
- `RuntimeClass` pod field for specifying the RuntimeClass the pod should be run with
- Kubelet implementation for fetching & interpreting the RuntimeClass
- CRI API & implementation for passing along the [RuntimeHandler](#runtime-handler).

### API

`RuntimeClass` is a new cluster-scoped resource in the `node.k8s.io` API group.

> _The `node.k8s.io` API group would eventually hold the Node resource when `core` is retired.
> Alternatives considered: `runtime.k8s.io`, `cluster.k8s.io`_

_(This is a simplified declaration, syntactic details will be covered in the API PR review)_

```go
type RuntimeClass struct {
    metav1.TypeMeta
    // ObjectMeta minimally includes the RuntimeClass name, which is used to reference the class.
    // Namespace should be left blank.
    metav1.ObjectMeta

    Spec RuntimeClassSpec
}

type RuntimeClassSpec struct {
    // RuntimeHandler specifies the underlying runtime the CRI calls to handle pod and/or container
    // creation. The possible values are specific to a given configuration & CRI implementation.
    // The empty string is equivalent to the default behavior.
    // +optional
    RuntimeHandler string
}
```

The runtime is selected by the pod by specifying the RuntimeClass in the PodSpec. Once the pod is
scheduled, the RuntimeClass cannot be changed.

```go
type PodSpec struct {
    ...
    // RuntimeClassName refers to a RuntimeClass object with the same name,
    // which should be used to run this pod.
    // +optional
    RuntimeClassName string
    ...
}
```

An unspecified `nil` or empty `""` RuntimeClassName is equivalent to the backwards-compatible
default behavior as if the RuntimeClass feature is disabled.

#### Examples

Suppose we operate a cluster that lets users choose between native runc containers, and gvisor and
kata-container sandboxes. We might create the following runtime classes:

```yaml
kind: RuntimeClass
apiVersion: node.k8s.io/v1alpha1
metadata:
    name: native  # equivalent to 'legacy' for now
spec:
    runtimeHandler: runc
---
kind: RuntimeClass
apiVersion: node.k8s.io/v1alpha1
metadata:
    name: gvisor
spec:
    runtimeHandler: gvisor
----
kind: RuntimeClass
apiVersion: node.k8s.io/v1alpha1
metadata:
    name: kata-containers
spec:
    runtimeHandler: kata-containers
----
# provides the default sandbox runtime when users don't care about which they're getting.
kind: RuntimeClass
apiVersion: node.k8s.io/v1alpha1
metadata:
  name: sandboxed
spec:
  runtimeHandler: gvisor
```

Then when a user creates a workload, they can choose the desired runtime class to use (or not, if
they want the default).

```yaml
apiVersion: extensions/v1beta1
kind: Deployment
metadata:
  name: sandboxed-nginx
spec:
  replicas: 2
  selector:
    matchLabels:
      app: sandboxed-nginx
  template:
    metadata:
      labels:
        app: sandboxed-nginx
    spec:
      runtimeClassName: sandboxed   #   <----  Reference the desired RuntimeClass
      containers:
      - name: nginx
        image: nginx
        ports:
        - containerPort: 80
          protocol: TCP
```

#### Runtime Handler

The `RuntimeHandler` is passed to the CRI as part of the `RunPodSandboxRequest`:

```proto
message RunPodSandboxRequest {
    // Configuration for creating a PodSandbox.
    PodSandboxConfig config = 1;
    // Named runtime configuration to use for this PodSandbox.
    string RuntimeHandler = 2;
}
```

The RuntimeHandler is provided as a mechanism for CRI implementations to select between different
predetermined configurations. The initial use case is replacing the experimental pod annotations
currently used for selecting a sandboxed runtime by various CRI implementations:

| CRI Runtime | Pod Annotation                                              |
| ------------|-------------------------------------------------------------|
| CRIO        | io.kubernetes.cri-o.TrustedSandbox: "false"                 |
| containerd  | io.kubernetes.cri.untrusted-workload: "true"                |
| frakti      | runtime.frakti.alpha.kubernetes.io/OSContainer: "true"<br>runtime.frakti.alpha.kubernetes.io/Unikernel: "true" |
| windows     | experimental.windows.kubernetes.io/isolation-type: "hyperv" |

These implementations could stick with scheme ("trusted" and "untrusted"), but the preferred
approach is a non-binary one wherein arbitrary handlers can be configured with a name that can be
matched against the specified RuntimeHandler. For example, containerd might have a configuration
corresponding to a "kata-runtime" handler:

```
[plugins.cri.containerd.kata-runtime]
    runtime_type = "io.containerd.runtime.v1.linux"
    runtime_engine = "/opt/kata/bin/kata-runtime"
    runtime_root = ""
```

This non-binary approach is more flexible: it can still map to a binary RuntimeClass selection
(e.g. `sandboxed` or `untrusted` RuntimeClasses), but can also support multiple parallel sandbox
types (e.g. `kata-containers` or `gvisor` RuntimeClasses).

### Versioning, Updates, and Rollouts

Runtimes are expected to be managed by the cluster administrator (or provisioner). In most cases,
runtime upgrades (and downgrades) should be handled by the administrator, without requiring any
interaction from the user. In these cases, the runtimes can be treated the same way we treat other
node components such as the Kubelet, node OS, and CRI runtime. In other words, the upgrade process
means rolling out nodes with the updated runtime, and gradually draining and removing old nodes from
the pool. For more details, see [Maintenance on a
Node](https://kubernetes.io/docs/tasks/administer-cluster/cluster-management/#maintenance-on-a-node).

If the upgraded runtime includes new features that users wish to take advantage of immediately, then
node labels can be used to select nodes supporting the updated runtime. In the uncommon scenario
where substantial changes to the runtime are made and application changes may be required, we
recommend that the updated runtime be treated as a _new_ runtime, with a separate RuntimeClass
(e.g. `sandboxed-v2`). This approach has the advantage of native support for rolling updates through
the same mechanisms as any other application update, so the updated applications can be carefully
rolled out to the new runtime.

Runtime upgrades will benefit from better scheduling support, which is a feature we plan to add in a
future release.

### Implementation Details

The Kubelet uses an Informer to keep a local cache of all RuntimeClass objects. When a new pod is
added, the Kubelet resolves the Pod's RuntimeClass against the local RuntimeClass cache.  Once
resolved, the RuntimeHandler field is passed to the CRI as part of the
[`RunPodSandboxRequest`][runpodsandbox]. At that point, the interpretation of the RuntimeHandler is
left to the CRI implementation, but it should be cached if needed for subsequent calls.

If the RuntimeClass cannot be resolved (e.g. doesn't exist) at Pod creation, then the request will
be rejected in admission (controller to be detailed in a following update). If the RuntimeClass
cannot be resolved by the Kubelet when `RunPodSandbox` should be called, then the Kubelet will fail
the Pod. The admission check on a replica recreation will prevent the scheduler from thrashing. If
the `RuntimeHandler` is not recognized by the CRI implementation, then `RunPodSandbox` will return
an error.

[runpodsandbox]: https://github.com/kubernetes/kubernetes/blob/b05a61e299777c2030fbcf27a396aff21b35f01b/pkg/kubelet/apis/cri/runtime/v1alpha2/api.proto#L344

#### Monitoring

The first round of monitoring implementation for `RuntimeClass` covers the
following two areas and is finished (tracked in
[#73058](https://github.com/kubernetes/kubernetes/issues/73058)):

- `how robust is every runtime?` A new metric
  [RunPodSandboxErrors](https://github.com/kubernetes/kubernetes/blob/596a48dd64bcaa01c1d2515dc79a558a4466d463/pkg/kubelet/metrics/metrics.go#L351)
  is added to track the RunPodSandbox operation errors, broken down by
  RuntimeClass.
- `how expensive is every runtime in terms of latency?` A new metric
  [RunPodSandboxDuration](https://github.com/kubernetes/kubernetes/blob/596a48dd64bcaa01c1d2515dc79a558a4466d463/pkg/kubelet/metrics/metrics.go#L341)
  is added to track the duration of RunPodSandbox operations, broken down by
  RuntimeClass.

### Risks and Mitigations

**Scope creep.** RuntimeClass has a fairly broad charter, but it should not become a default
dumping ground for every new feature exposed by the node. For each feature, careful consideration
should be made about whether it belongs on the Pod, Node, RuntimeClass, or some other resource. The
[non-goals](#non-goals) should be kept in mind when considering RuntimeClass features.

**Becoming a general policy mechanism.** RuntimeClass should not be used a replacement for
PodSecurityPolicy. The use cases for defining multiple RuntimeClasses for the same underlying
runtime implementation should be extremely limited (generally only around updates & rollouts). To
enforce this, no authorization or restrictions are placed directly on RuntimeClass use; in order to
restrict a user to a specific RuntimeClass, you must use another policy mechanism such as
PodSecurityPolicy.

**Pushing complexity to the user.** RuntimeClass is a new resource in order to hide the complexity
of runtime configuration from most users (aside from the cluster admin or provisioner). However, we
are still side-stepping the issue of precisely defining specific types of runtimes like
"Sandboxed". However, it is still up for debate whether precisely defining such runtime categories
is even possible. RuntimeClass allows us to decouple this specification from the implementation, but
it is still something I hope we can address in a future iteration through the concept of pre-defined
or "conformant" RuntimeClasses.

**Non-portability.** We are already in a world of non-portability for many features (see [examples
of runtime variation](#examples-of-runtime-variation). Future improvements to RuntimeClass can help
address this issue by formally declaring supported features, or matching the runtime that supports a
given workload
mitaclly. Another issue is that pods need to refer to a RuntimeClass by name,
which may not be defined in every cluster. This is something that can be addressed through
pre-defined runtime classes (see previous risk), and/or by "fitting" pod requirements to compatible
RuntimeClasses.

## RuntimeClass Scheduling

RuntimeClass scheduling enables native support for heterogeneous clusters where
every node does not necessarily support every RuntimeClass. This feature allows
pod authors to select a RuntimeClass without needing to worry about cluster
topology.

### RuntimeClass Scheduling Motivation

In the initial RuntimeClass implementation, we explicitly assumed that the
cluster nodes were homogenous with regards to RuntimeClasses. It was still
possible to run a heterogeneous cluster, but pod authors would need to set
appropriate [NodeSelector][] rules and [tolerations][taint-and-toleration] to
ensure the pods landed on supporting nodes.

As [use cases](#runtimeclass-scheduling -user-stories) have appeared and solidified,
it has become clear that heterogeneous clusters will not be uncommmon, and supporting
a smoother user experience will be valuable.

[NodeSelector]: https://kubernetes.io/docs/concepts/configuration/assign-pod-node/
[taint-and-toleration]: https://kubernetes.io/docs/concepts/configuration/taint-and-toleration/

#### RuntimeClass Scheduling Goals

- Pods using a RuntimeClass that is not supported by all nodes in a cluster are
  automatically scheduled to nodes that support that RuntimeClass.
- RuntimeClass scheduling is compatible with other scheduling constraints. For
  example, a pod with a node selector for GPUs and a Linux runtime should be
  scheduled to a linux node with GPUs (an intersection).

#### RuntimeClass Scheduling Non-Goals

- Replacing [SchedulingPolicy](#scheduling-policy) with RuntimeClasses.

The following are **currently** out of scope, but _may_ be revisited at a later
date.

- Automatic topology discovery or node labeling
- Automatically selecting a RuntimeClass for a pod based on node availability.
- Defining official or reserved label or taint schemas or for RuntimeClasses.

### RuntimeClass Scheduling Proposal

A new optional `Scheduling` structure will be added to the RuntimeClass API. The
scheduling struct includes both a `NodeSelector` and `Tolerations` that control
how a pod using that RuntimeClass is scheduled. The NodeSelector rules are
applied during scheduling, but the Tolerations are added to the PodSpec during
admission by the new RuntimeClass admission controller.

#### RuntimeClass Scheduling User Stories

##### Windows

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

##### Sandboxed Nodes

Some users wish to keep sandbox workloads and native workloads separate. For
example, a node running untrusted sandboxed workloads may have stricter
requirements about which trusted services are run on that node.

- As a **cluster administrator** I want to ensure that untrusted workloads are
  not colocated with sensitive data.
- As a **developer** I want run an untrusted service without worrying about
  where the service is running.
- As a **cluster administrator** I want to autoscale trusted and untrusted nodes
  independently.

### Design Details

#### RuntimeClass Scheduling API

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

#### RuntimeClass Admission Controller

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

#### Labeling Nodes

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

#### RuntimeClass Scheduling Graduation Criteria

This feature will be rolled into RuntimeClass beta in v1.15, thereby skipping
the alpha phase. This means the feature is expected to be beta quality at launch:

- Thorough testing, including unit, integration and e2e coverage.
- Thoroughly documented (as an extension to the [RuntimeClass documentation][]).

[RuntimeClass documentation]: https://kubernetes.io/docs/concepts/containers/runtime-class/

### RuntimeClass Scheduling Alternatives

#### Scheduler

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

#### RuntimeController Mix-in

Rather than resolving scheduling in the scheduler, the `NodeSelectorTerm`
rules and `Tolerations` are mixed in to the PodSpec. The mix-in happens in the
mutating admission phase, and is performed by a new `RuntimeController` built-in
admission plugin. The same admission controller is shared with the [Pod
Overhead][] proposal.

[Pod Overhead]: https://github.com/kubernetes/enhancements/pull/887

##### RuntimeController

RuntimeController is a new in-tree admission plugin that should eventually be
enabled on almost every Kubernetes clusters. The role of the controller for
scheduling is to merge the scheduling constraints from the RuntimeClass into the
PodSpec. Eventually, the controller's responsibilities may grow, such as to
merge in [pod overhead][] or validate feature compatibility.

Note that the RuntimeController is not needed if we implement [native scheduler
support](#runtimeclass-aware-scheduling).

##### Mix-in

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

#### NodeSelector

Replacing the NodeSelector's `[]NodeSelectorRequirements` type with the
PodSpec's label `map[string]string` approach greatly simplifies the merging
logic, but sacrifices a lot of flexibliity. For exameple, the operator in
NodeSelectorRequriments enables selections like:

- Negative selection, such as "operating system is _not_ windows"
- Numerical comparison, such as "runc version is _at least_ X" (although it doesn't currently support semver)
- Set selection, such as "sandbox is _one of_ kata-cotainers or gvisor"

#### Native RuntimeClass Reporting

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

#### Scheduling Policy

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

## Graduation Criteria

Alpha:

- [x] Everything described in the current proposal:
  - [x] Introduce the RuntimeClass API resource
  - [x] Add a RuntimeClassName field to the PodSpec
  - [x] Add a RuntimeHandler field to the CRI `RunPodSandboxRequest`
  - [x] Lookup the RuntimeClass for pods & plumb through the RuntimeHandler in the Kubelet (feature
    gated)
- [x] RuntimeClass support in at least one CRI runtime & dockershim
  - [x] Runtime Handlers can be statically configured by the runtime, and referenced via RuntimeClass
  - [x] An error is reported when the handler or is unknown or unsupported
- [x] Testing
  - [x] Kubernetes E2E tests (only validating single runtime handler cases)

Beta:

- [x] Several major runtimes support RuntimeClass, and the current [untrusted annotations](#runtime-handler) are
  deprecated.
  - [x] [containerd](https://github.com/containerd/cri/pull/891)
  - [x] [CRI-O](https://github.com/kubernetes-sigs/cri-o/pull/1847)
  - [x] [dockershim](https://github.com/kubernetes/kubernetes/pull/67909)
- [x] Comprehensive test coverage
  - [x] RuntimeClasses are configured in the E2E environment with test coverage of a non-default
    RuntimeClass
- [x] Comprehensive coverage of RuntimeClass metrics. [#73058](http://issue.k8s.io/73058)
- [x] The update & upgrade story is revisited, and a longer-term approach is implemented as necessary.

[cri-validation]: https://github.com/kubernetes-sigs/cri-tools/blob/master/docs/validation.md

Stable:

- [x] Wide adoption of the feature
  - [x] Google relies on RuntimeClass in [gVisor](https://gvisor.dev/).
  - [x] RedHat uses RuntimeClass to install [kata](https://github.com/openshift/kata-operator) on OpenShift with CRI-O. Another use case is around using a custom runtime class for enabling user namespaces for certain workloads. We would like to rely on RuntimeClass to distinguish between Windows and Linux pods and have the security policies defaulted differently for Linux pods. We also want to use RuntimeClasses to differentiate between different flavors of Windows OSes as there is a tight coupling between a Windows Containers and the Windows host.
  - [x] Microsoft has plans to use RuntimeClass to control runtime to enable [Hyper-V isolated containers](https://docs.microsoft.com/en-us/virtualization/windowscontainers/manage-containers/hyperv-container) (which allow running containers targeting multiple Windows Server versions on the same agent node)
    - [Difficulties in mixed OS & arch clusters](https://docs.google.com/document/d/12uZt-KSG8v4CSyUDr0EC6btmzpVOZAWzqYDif3EoeBU/edit#heading=h.uno03u1f2t9i) (Discussions around usage in this document)
    - Example runtime class used in some Windows PROW jobs - [2004-hyperv-runtimeclass.yaml](https://github.com/kubernetes-sigs/windows-testing/blob/master/helpers/hyper-v-mutating-webhook/2004-hyperv-runtimeclass.yaml)
- [x] No release blocking feedback for API and functionality

## Implementation History

- 2020-10-17: RuntimeClass approved to be promoted as stable
- 2019-09-05: Implement RuntimeClass Scheduling as a beta stage feature. [Umbrella issue](https://github.com/kubernetes/kubernetes/issues/81016)
- 2019-03-25: RuntimeClass released as beta with Kubernetes v1.14
- 2019-03-14: Initial KEP for RuntimeClass Scheduling published.
- 2018-10-05: [RuntimeClass Scheduling Brainstorm](https://docs.google.com/document/d/1W51yBNTvp0taeEss56GTk8jczqFJ2d6jBeN6sCSlYZU/edit#) published
- 2018-09-27: RuntimeClass released as alpha with Kubernetes v1.12
- 2018-06-11: SIG-Node decision to move forward with proposal
- 2018-06-19: Initial KEP published.

## Appendix

### Proposed Future Enhancements

The following ideas may be explored in a future iteration:

- The following monitoring areas will be skipped for now, but may be considered for future:
  - how many runtimes does a cluster support?
  - how many scheduling failures were caused by unsupported runtimes or insufficient
  resources of a certain runtime?
  - how many runtimes node supports?
- Surfacing support for optional features by runtimes, and surfacing errors caused by
  incompatible features & runtimes earlier.
- Automatic runtime or feature discovery - initially RuntimeClasses are manually defined (by the
  cluster admin or provider), and are asserted to be an accurate representation of the runtime.
- Scheduling in heterogeneous clusters - it is possible to operate a heterogeneous cluster
  (different runtime configurations on different nodes) through scheduling primitives like
  `NodeAffinity` and `Taints+Tolerations`, but the user is responsible for setting these up and
  automatic runtime-aware scheduling is out-of-scope.
- Define standardized or conformant runtime classes - although I would like to declare some
  predefined RuntimeClasses with specific properties, doing so is out-of-scope for this initial KEP.
- [Pod Overhead][] - Although RuntimeClass is likely to be the configuration mechanism of choice,
  the details of how pod resource overhead will be implemented is out of scope for this KEP.
- Provide a mechanism to dynamically register or provision additional runtimes.
- Requiring specific RuntimeClasses according to policy. This should be addressed by other
  cluster-level policy mechanisms, such as PodSecurityPolicy.
- "Fitting" a RuntimeClass to pod requirements - In other words, specifying runtime properties and
  letting the system match an appropriate RuntimeClass, rather than explicitly assigning a
  RuntimeClass by name. This approach can increase portability, but can be added seamlessly in a
  future iteration.
- The cluster admin can choose which RuntimeClass is the default in a cluster.

[Pod Overhead]: https://docs.google.com/document/d/1EJKT4gyl58-kzt2bnwkv08MIUZ6lkDpXcxkHqCvvAp4/edit

### Examples of runtime variation

- Linux Security Module (LSM) choice - Kubernetes supports both AppArmor & SELinux options on pods,
  but those are mutually exclusive, and support of either is not required by the runtime. The
  default configuration is also not well defined.
- Seccomp-bpf - Kubernetes has alpha support for specifying a seccomp profile, but the default is
  defined by the runtime, and support is not guaranteed.
- Windows containers - isolation features are very OS-specific, and most of the current features are
  limited to linux. As we build out Windows container support, we'll need to add windows-specific
  features as well.
- Host namespaces (Network,PID,IPC) may not be supported by virtualization-based runtimes
  (e.g. Kata-containers & gVisor).
- Per-pod and Per-container resource overhead varies by runtime.
- Device support (e.g. GPUs) varies wildly by runtime & nodes.
- Supported volume types varies by node - it remains TBD whether this information belongs in
  RuntimeClass.
- The list of default capabilities is defined in Docker, but not Kubernetes. Future runtimes may
  have differing defaults, or support a subset of capabilities.
- `Privileged` mode is not well defined, and thus may have differing implementations.
- Support for resource over-commit and dynamic resource sizing (e.g. Burstable vs Guaranteed
  workloads)
